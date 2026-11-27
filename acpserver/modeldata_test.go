package acpserver

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// dataDoc is a models.dev document stamped source and fetched at, told
// apart from another by its pointer.
func dataDoc(t *testing.T, source string, at time.Time) *provider.ModelData {
	t.Helper()
	d, err := provider.ParseModelData(strings.NewReader(`{"openai":{"models":{"gpt-x":{"name":"GPT X"}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	return d.Stamped(provider.ModelDataStamp{Source: source, Fetched: at})
}

// dataHarness is a limit harness whose current document is cur, whose
// ModelInfo gives each model the window and cap its document names in
// facts (none for a document it does not name), and which records every
// document ModelInfo was given.
type dataHarness struct {
	*limitHarness
	cur   atomic.Pointer[provider.ModelData]
	mu    sync.Mutex
	facts map[*provider.ModelData]int
	given []*provider.ModelData
}

func newDataHarness(t *testing.T, first *provider.ModelData, facts map[*provider.ModelData]int, turns ...llmtest.Turn) *dataHarness {
	t.Helper()
	h := &dataHarness{facts: facts}
	if first != nil {
		h.cur.Store(first)
	}
	h.limitHarness = newLimitHarnessWith(t, SessionSettings{}, map[string]llm.Model{}, func(o *Options) {
		o.ModelData = h.cur.Load
		o.ModelInfo = func(d *provider.ModelData, id, model string) llm.Model {
			h.mu.Lock()
			h.given = append(h.given, d)
			n := h.facts[d]
			h.mu.Unlock()
			if n == 0 {
				return llm.Model{Provider: id, ID: model}
			}
			return llm.Model{Provider: id, ID: model, MaxTokens: n, MaxTokensFrom: llm.CapCatalog, ContextWindow: 100 * n,
				Limits: llm.Limits{Output: fact(n, "models.dev"), StreamedOutput: fact(n, "models.dev"), Window: fact(100*n, "models.dev")}}
		}
		o.ModelDataLine = func(d *provider.ModelData) string {
			if d == nil {
				return "Model data: none"
			}
			return "Model data: " + d.Stamp().Source
		}
	}, turns...)
	return h
}

// window is the session's /context window figure.
func (h *dataHarness) window(id acp.SessionId) string {
	h.t.Helper()
	for line := range strings.SplitSeq(h.context(id), "\n") {
		if strings.HasPrefix(line, "Context: ") {
			return line
		}
	}
	h.t.Fatalf("no Context line in %q", h.context(id))
	return ""
}

// M21: a refresh that lands while sessions are open is current for the
// sessions that start after it; the open ones keep theirs (-race: the swap
// races the open session's reads) (0072-MADR D19; "Keep each session's
// facts from its start (Recommended)").
func TestModelDataSwapForNewSessions_0072(t *testing.T) {
	d1, d2 := dataDoc(t, "https://one.invalid/api.json", time.Now()), dataDoc(t, "https://two.invalid/api.json", time.Now())
	h := newDataHarness(t, d1, map[*provider.ModelData]int{d1: 1000, d2: 2000})
	open := h.session()
	var wg sync.WaitGroup
	wg.Go(func() { h.cur.Store(d2) })
	during := h.window(open)
	wg.Wait()
	later := h.session()
	if got := h.window(open); got != during || !strings.Contains(got, "of 100,000 tokens") {
		t.Errorf("the open session reads %q (and %q during the swap); want its own document's 100,000", got, during)
	}
	if got := h.window(later); !strings.Contains(got, "of 200,000 tokens") {
		t.Errorf("a session started after the refresh reads %q; want the new document's 200,000", got)
	}
}

// M37 (acpserver's half): /context and /session end with the line for the
// session's document, after 0071's output-limit line (0072-MADR D12).
func TestContextShowsModelData_0072(t *testing.T) {
	d1, d2 := dataDoc(t, "https://one.invalid/api.json", time.Now()), dataDoc(t, "https://two.invalid/api.json", time.Now())
	h := newDataHarness(t, d1, map[*provider.ModelData]int{d1: 1000})
	id := h.session()
	h.cur.Store(d2)
	for _, cmd := range []string{"/context", "/session"} {
		text, _ := slash(t, h.c, h.client, id, cmd)
		lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
		n := len(lines)
		if n < 2 || !strings.HasPrefix(lines[n-2], "Output limit: ") || lines[n-1] != "Model data: https://one.invalid/api.json" {
			t.Errorf("%s ends %q; want the output limit's line, then the session's document's", cmd, lines[max(0, n-2):])
		}
	}
}

// M50: a turn's cap is read from the session's document, and a refresh
// during the session does not change it (0072-MADR D16; 0071-MADR L1).
func TestTurnCapFromModelData_0072(t *testing.T) {
	d1, d2 := dataDoc(t, "https://one.invalid/api.json", time.Now()), dataDoc(t, "https://two.invalid/api.json", time.Now())
	h := newDataHarness(t, d1, map[*provider.ModelData]int{d1: 1000, d2: 2000}, llmtest.Text("one"), llmtest.Text("two"))
	id := h.session()
	h.mustPrompt(id, "before")
	h.cur.Store(d2)
	h.mustPrompt(id, "after the refresh")
	for i, r := range h.calls() {
		if r.MaxTokens != 1000 {
			t.Errorf("call %d asked for %d; want the session's document's 1000", i+1, r.MaxTokens)
		}
	}
}

// M60: a session takes the current document when it is created, loaded or
// forked, and keeps it; the document is in memory only, so a session loaded
// or resumed after a refresh takes the copy current then, and a model that
// copy lacks is unknown (the custom-id warning is --model's: internal/cli's
// TestSessionKeepsModelData_0072) (0072-MADR D19; case 9).
func TestSessionKeepsModelData_0072(t *testing.T) {
	d1, d2 := dataDoc(t, "https://one.invalid/api.json", time.Now()), dataDoc(t, "https://two.invalid/api.json", time.Now())
	d3 := dataDoc(t, "https://three.invalid/api.json", time.Now()) // names no model
	h := newDataHarness(t, d1, map[*provider.ModelData]int{d1: 1000, d2: 2000}, llmtest.Text("one"))
	id := h.session()
	h.mustPrompt(id, "a turn, so the session is kept")
	// A refresh lands before the fork: the fork takes the document current
	// at its fork, not its parent's (D19).
	h.cur.Store(d2)
	fork := newSession(t, h.c, t.TempDir(), map[string]any{"gobble": map[string]any{"forkFrom": string(id)}}).SessionId
	if got := h.window(fork); !strings.Contains(got, "of 200,000 tokens") {
		t.Errorf("a fork made after a refresh reads %q; want the document current at its fork, not its parent's", got)
	}
	h.cur.Store(d3)
	if got := h.window(id); !strings.Contains(got, "of 100,000 tokens") {
		t.Errorf("the created session reads %q after the refreshes; want its own document's", got)
	}
	if got := h.window(fork); !strings.Contains(got, "of 200,000 tokens") {
		t.Errorf("the fork reads %q after a later refresh; want its own document's", got)
	}
	for _, how := range []string{"load", "resume"} {
		if _, err := h.c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
			t.Fatal(err)
		}
		var err error
		if how == "load" {
			_, err = h.c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
		} else {
			_, err = h.c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: t.TempDir()})
		}
		if err != nil {
			t.Fatalf("%s: %v", how, err)
		}
		if got := h.window(id); !strings.Contains(got, "of an unknown window") {
			t.Errorf("after a %s the session reads %q; want the copy current at its %s, which lacks the model", how, got, how)
		}
		if d := h.live(id).data.Load(); d != d3 {
			t.Errorf("after a %s the session holds %p; want the current %p", how, d, d3)
		}
	}
}

// M61: a session that began with none takes the first document there is
// when it lands, and keeps it through a later refresh (0072-MADR D19; "Pick
// up the first download (Recommended)").
func TestSessionWithNoneGainsFirstCopy_0072(t *testing.T) {
	d1, d2 := dataDoc(t, "https://one.invalid/api.json", time.Now()), dataDoc(t, "https://two.invalid/api.json", time.Now())
	h := newDataHarness(t, nil, map[*provider.ModelData]int{d1: 1000, d2: 2000})
	id := h.session()
	if got := h.window(id); !strings.Contains(got, "of an unknown window") {
		t.Fatalf("with none: %q", got)
	}
	h.cur.Store(d1)
	if got := h.window(id); !strings.Contains(got, "of 100,000 tokens") {
		t.Errorf("after the first download: %q; want it taken", got)
	}
	h.cur.Store(d2)
	if got := h.window(id); !strings.Contains(got, "of 100,000 tokens") {
		t.Errorf("after a second refresh: %q; want the first kept", got)
	}
}
