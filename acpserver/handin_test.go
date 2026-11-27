package acpserver

import (
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"weak"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The tests of 0072-PLAN's Phase 2b in acpserver: each provider built for a
// session gets the session's document (0072-MADR D17, D22, D23, M62, M71,
// M72, M86; "Builders take the document (Recommended)").

// buildRec records each provider build: who built it, for which session,
// and the source of the document it was given ("none" for nil).
type buildRec struct {
	mu  sync.Mutex
	got []string
}

func (r *buildRec) add(kind, sid string, d *provider.ModelData) {
	src := "none"
	if d != nil {
		src = d.Stamp().Source
	}
	r.mu.Lock()
	r.got = append(r.got, kind+" "+sid+" "+src)
	r.mu.Unlock()
}

// of are the records of session id, without it.
func (r *buildRec) of(id acp.SessionId) []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []string
	for _, g := range r.got {
		if f := strings.SplitN(g, " ", 3); f[1] == string(id) {
			out = append(out, f[0]+" "+f[2])
		}
	}
	return out
}

// newBuildHarness is a limit harness whose current document is cur's and
// whose three builders record what they are given, then build as the
// harness does; the title's is a small scripted model.
func newBuildHarness(t *testing.T, cur *atomic.Pointer[provider.ModelData], rec *buildRec, turns ...llmtest.Turn) *limitHarness {
	t.Helper()
	small := llmtest.NewScript(llmtest.Text("T1"), llmtest.Text("T2"), llmtest.Text("T3"), llmtest.Text("T4"))
	return newLimitHarnessWith(t, SessionSettings{}, map[string]llm.Model{}, func(o *Options) {
		turn, other := o.Provider, o.ProviderFor
		o.ModelData = cur.Load
		o.Provider = func(d *provider.ModelData, sid string, r llm.Retry) (llm.Provider, error) {
			rec.add("session", sid, d)
			return turn(d, sid, r)
		}
		o.ProviderFor = func(d *provider.ModelData, id, sid string, r llm.Retry) (llm.Provider, error) {
			rec.add("switched", sid, d)
			return other(d, id, sid, r)
		}
		o.Title = TitleOptions{Provider: func(d *provider.ModelData, sid string, _ llm.Retry) (llm.Provider, error) {
			rec.add("title", sid, d)
			return small, nil
		}, ProviderID: "fake", Model: "m1"}
	}, turns...)
}

// M62 (acpserver's half), M86: the turn's, compaction's, a switched
// provider's and the title's builds each get the session's document, not
// the current one; a session started after a refresh gets the new one.
func TestProviderGetsSessionModelData_0072(t *testing.T) {
	d1, d2 := dataDoc(t, "https://one.invalid/api.json", time.Now()), dataDoc(t, "https://two.invalid/api.json", time.Now())
	var cur atomic.Pointer[provider.ModelData]
	cur.Store(d1)
	rec := &buildRec{}
	h := newBuildHarness(t, &cur, rec, llmtest.Text("one"), llmtest.Text("two"))
	turn, compact, switched := h.session(), h.session(), h.session()
	cur.Store(d2) // a refresh lands after the three began
	h.mustPrompt(turn, "the turn")
	h.mustPrompt(compact, "/compact") // compaction builds the session's provider first
	if _, err := setOption(t, h.c, switched, configModel, "other/x9"); err != nil {
		t.Fatal(err)
	}
	later := h.session()
	h.mustPrompt(later, "after the refresh")
	waitFor(t, func() bool { return len(rec.of(turn)) >= 2 && len(rec.of(later)) >= 2 }) // the titles, in the background
	one, two := "https://one.invalid/api.json", "https://two.invalid/api.json"
	for _, c := range []struct {
		name string
		id   acp.SessionId
		want []string
	}{
		{"turn and title", turn, []string{"session " + one, "title " + one}},
		{"compaction", compact, []string{"session " + one}},
		{"switched", switched, []string{"switched " + one}},
		{"new session", later, []string{"session " + two, "title " + two}},
	} {
		got := rec.of(c.id)
		slices.Sort(got)
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: builds %q; want %q", c.name, got, c.want)
		}
	}
}

// M71: a session that began with none, a prompt blocked mid-call when the
// first download lands: that call completes on the provider it took; the
// next prompt's provider is built with the document; a session that holds
// a document keeps its provider through a refresh (0072-MADR D22).
func TestSessionProviderRebuiltOnFirstCopy_0072(t *testing.T) {
	d1, d2 := dataDoc(t, "https://one.invalid/api.json", time.Now()), dataDoc(t, "https://two.invalid/api.json", time.Now())
	var cur atomic.Pointer[provider.ModelData]
	rec := &buildRec{}
	release := make(chan struct{})
	var first atomic.Bool
	first.Store(true)
	h := newLimitHarnessWith(t, SessionSettings{}, map[string]llm.Model{}, func(o *Options) {
		o.ModelData = cur.Load
		o.Provider = func(d *provider.ModelData, sid string, _ llm.Retry) (llm.Provider, error) {
			if sid == "" {
				return llmtest.NewScript(), nil // initialize's image check
			}
			rec.add("session", sid, d)
			if first.CompareAndSwap(true, false) {
				return gated{Provider: llmtest.NewScript(llmtest.Text("blocked")), open: release}, nil
			}
			return llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"), llmtest.Text("c")), nil
		}
	})
	none := h.session()
	done := make(chan error, 1)
	go func() {
		_, err := prompt(t, h.c, none, "a prompt under way")
		done <- err
	}()
	waitFor(t, func() bool { return len(rec.of(none)) == 1 })
	cur.Store(d1) // the first download lands while the call waits
	close(release)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("the call under way: %v; want it to complete on the provider it took", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the call under way did not complete")
	}
	h.mustPrompt(none, "the next prompt")
	cur.Store(d2)
	h.mustPrompt(none, "after a second refresh")
	if got, want := rec.of(none), []string{"session none", "session https://one.invalid/api.json"}; !slices.Equal(got, want) {
		t.Errorf("the none-session's builds %q; want %q: rebuilt once, with its first document", got, want)
	}
	held := h.session() // takes d2
	h.mustPrompt(held, "one")
	cur.Store(d1)
	h.mustPrompt(held, "two")
	if got := rec.of(held); len(got) != 1 {
		t.Errorf("a session holding a document was built %d times (%q); want once, kept through a refresh", len(got), got)
	}
}

// M72: across a refresh, providers are built once per session that began
// with none, and for no other; a document no session holds any longer, nor
// the agent, is released (0072-MADR D23).
func TestModelDataRefreshCost_0072(t *testing.T) {
	var cur atomic.Pointer[provider.ModelData]
	var builds atomic.Int64
	h := newLimitHarnessWith(t, SessionSettings{}, map[string]llm.Model{}, func(o *Options) {
		o.ModelData = cur.Load
		o.Provider = func(_ *provider.ModelData, sid string, _ llm.Retry) (llm.Provider, error) {
			if sid != "" {
				builds.Add(1)
			}
			return llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b"), llmtest.Text("c")), nil
		}
	})
	n1, n2 := h.session(), h.session() // both begin with none
	h.mustPrompt(n1, "x")
	h.mustPrompt(n2, "x")
	d1 := dataDoc(t, "https://one.invalid/api.json", time.Now())
	released := weak.Make(d1)
	cur.Store(d1) // the first download
	before := builds.Load()
	h.mustPrompt(n1, "y")
	h.mustPrompt(n2, "y")
	if got := builds.Load() - before; got != 2 {
		t.Errorf("the first download rebuilt %d providers; want one per session that began with none, 2", got)
	}
	held := h.session()
	h.mustPrompt(held, "x")
	cur.Store(dataDoc(t, "https://two.invalid/api.json", time.Now())) // a refresh
	before = builds.Load()
	for _, id := range []acp.SessionId{n1, n2, held} {
		h.mustPrompt(id, "z")
	}
	if got := builds.Load() - before; got != 0 {
		t.Errorf("a refresh rebuilt %d providers; want none: every open session holds a document", got)
	}
	d1 = nil //nolint:ineffassign,wastedassign // the test drops its own reference, so only the sessions hold it
	_ = d1
	for _, id := range []acp.SessionId{n1, n2, held} {
		if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 20 && released.Value() != nil; i++ {
		runtime.GC()
		time.Sleep(10 * time.Millisecond)
	}
	if released.Value() != nil {
		t.Error("the first document is still held after the last session holding it ended and a collection")
	}
}
