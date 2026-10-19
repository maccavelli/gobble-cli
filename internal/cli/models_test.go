package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/personality"
)

// withLister replaces listModels for one test.
func withLister(t *testing.T, fn func(context.Context, string, string, provider.Options) (provider.Listing, error)) {
	t.Helper()
	old := listModels
	listModels = fn
	t.Cleanup(func() { listModels = old })
}

// liveLister answers every listing live with ids, and counts the calls and
// what each named.
type liveLister struct {
	ids   []string
	calls atomic.Int64
	mu    sync.Mutex
	got   []string // "id key" per call
}

func (l *liveLister) list(_ context.Context, id, key string, _ provider.Options) (provider.Listing, error) {
	l.calls.Add(1)
	l.mu.Lock()
	l.got = append(l.got, id+" "+key)
	l.mu.Unlock()
	return provider.Listing{Live: true, Models: slices.Clone(l.ids)}, nil
}

// modelEnv is a runEnv as `gobble acp` builds one, in a fresh home, with
// warnings written to the returned buffer.
func modelEnv(t *testing.T) (*runEnv, *syncBuffer) {
	t.Helper()
	t.Setenv("GOBBLE_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	warnings := &syncBuffer{}
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out, stderr: warnings}}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	e.processSettings("")
	return e, warnings
}

// The tests' lister fails offline, so no test lists over the network
// (0020-MADR D7).
func TestListerIsOffline(t *testing.T) {
	if _, err := listModels(t.Context(), "openai", "k", provider.Options{}); !errors.Is(err, errOfflineList) {
		t.Fatalf("listModels gave %v, want the offline lister's error", err)
	}
}

// The model choice is the listing's, its first the default, listed with the
// chosen provider's credential (0020-MADR D3, D4).
func TestModelChoiceFromListing(t *testing.T) {
	e, _ := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	l := &liveLister{ids: []string{"gpt-6-astra", "gpt-6-luna", "gpt-5-mini"}}
	withLister(t, l.list)
	c := e.modelChoice()
	if c.Provider != "openai" || c.Default != "gpt-6-astra" || !slices.Equal(c.Models, l.ids) {
		t.Fatalf("model choice %+v, want openai's listing with gpt-6-astra the default", c)
	}
	if !slices.Equal(l.got, []string{"openai test-only"}) {
		t.Fatalf("listed %q, want openai with its key once", l.got)
	}
}

// One listing serves the process: the agent's start and every model choice,
// from any goroutine (0020-MADR D3).
func TestModelChoiceListsOnce(t *testing.T) {
	e, _ := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	l := &liveLister{ids: []string{"gpt-6-luna"}}
	withLister(t, l.list)
	e.agentOptions()
	var wg sync.WaitGroup
	for range 5 {
		wg.Go(func() { e.modelChoice() })
	}
	wg.Wait()
	if n := l.calls.Load(); n != 1 {
		t.Fatalf("%d listings, want 1", n)
	}
}

// A failed listing offers the static models and default, and its warning
// names the provider and the failure's kind, never its text, in the warning
// stream or the log file (0020-MADR D4, D6).
func TestModelChoiceFallsBack(t *testing.T) {
	e, warnings := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	withLister(t, func(context.Context, string, string, provider.Options) (provider.Listing, error) {
		return provider.Listing{}, fmt.Errorf("%w: Incorrect API key provided: sk-pl***nt", llm.ErrAuth)
	})
	static := provider.Models("openai")
	if c := e.modelChoice(); c.Provider != "openai" || c.Default != static[0] || !slices.Equal(c.Models, static) {
		t.Fatalf("model choice %+v, want openai's static models", c)
	}
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	logged, err := os.ReadFile(filepath.Join(dirs.State, "logs", "gobble.log"))
	if err != nil {
		t.Fatal(err)
	}
	w := warnings.String()
	for _, want := range []string{"model listing failed; offering the curated static list", "provider=openai", "kind=auth"} {
		if !strings.Contains(w, want) {
			t.Errorf("warnings %q lack %q", w, want)
		}
	}
	if strings.Contains(w, "sk-pl") || strings.Contains(string(logged), "sk-pl") {
		t.Fatalf("the failure's text reached the warnings or the log:\n%s\n%s", w, logged)
	}
}

// A failed listing is retried in the background, once a minute at most, and
// a live one is kept (0020-MADR D3).
func TestModelListingRetries(t *testing.T) {
	var now atomic.Int64 // seconds since the start
	start := time.Now()
	fake := func() time.Time { return start.Add(time.Duration(now.Load()) * time.Second) }
	var fetches atomic.Int64
	release := make(chan struct{})
	l := newModelListing(func() provider.Listing {
		if fetches.Add(1) == 1 {
			return provider.Listing{Models: []string{"static"}}
		}
		<-release
		return provider.Listing{Live: true, Models: []string{"listed"}}
	}, fake)

	// A retry runs in its own goroutine, so "none started" is only known
	// after giving one the time to start.
	noRetry := func(when string) {
		t.Helper()
		if waitForWithin(200*time.Millisecond, func() bool { return fetches.Load() > 1 }) {
			t.Fatalf("%d fetches %s, want 1", fetches.Load(), when)
		}
	}
	if r := l.get(); r.Live {
		t.Fatalf("first get %+v, want the failed listing", r)
	}
	noRetry("at the start")
	now.Store(59)
	l.get()
	noRetry("at 59 s")
	now.Store(60)
	if r := l.get(); r.Live {
		t.Fatalf("get at 60 s %+v, want the failed listing while the retry runs", r)
	}
	if l.get(); !waitFor(func() bool { return fetches.Load() == 2 }) {
		t.Fatalf("%d fetches after 60 s, want one retry", fetches.Load())
	}
	l.get() // a second get while the retry runs starts none
	close(release)
	if !waitFor(func() bool { return l.get().Live }) {
		t.Fatal("the retry's live listing was not kept")
	}
	now.Store(3600)
	if r := l.get(); !r.Live || fetches.Load() != 2 {
		t.Fatalf("an hour on: %+v after %d fetches, want the kept live listing after 2", r, fetches.Load())
	}
}

// waitFor polls cond for up to 2 s.
func waitFor(cond func() bool) bool { return waitForWithin(2*time.Second, cond) }

// waitForWithin polls cond for up to d.
func waitForWithin(d time.Duration, cond func() bool) bool {
	for deadline := time.Now().Add(d); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if cond() {
			return true
		}
	}
	return cond()
}

// With no credential there is no model choice and no listing (0020-MADR D3).
func TestNoCredentialNoListing(t *testing.T) {
	e, _ := modelEnv(t)
	l := &liveLister{ids: []string{"x"}}
	withLister(t, l.list)
	e.agentOptions()
	if c := e.modelChoice(); c.Provider != "" || c.Models != nil || c.Default != "" {
		t.Fatalf("model choice %+v, want none", c)
	}
	if n := l.calls.Load(); n != 0 {
		t.Fatalf("%d listings without a credential, want 0", n)
	}
}

// The ACP model option offers the listed models, past the static six;
// /model takes any of them, and the next request names it (0020-MADR D5;
// issue 1). It runs the agent gobble's modes run, through acpclient, since
// internal/cli may not import the SDK (archtest rule 2; 0020-PLAN P2
// deviation 4); set_config_option reaches the same setConfig, which
// acpserver's TestConfigOptionsPersist covers.
func TestModelOptionOffersListedModels(t *testing.T) {
	e, _ := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	listed := []string{"gpt-6-luna", "gpt-5.6-luna", "gpt-4.1-mini", "gpt-5.6-terra", "gpt-6.1-sol", "gpt-6-astra", "gpt-5-nano", "gpt-5-mini"}
	withLister(t, (&liveLister{ids: listed}).list)
	script := llmtest.NewScript(llmtest.Text("ok"))
	withModel(t, script)
	conn, err := e.startAgent(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Error(err)
		}
	})
	var mu sync.Mutex
	var updates []acpclient.Update
	s, err := conn.NewSession(t.Context(), t.TempDir(), func(u acpclient.Update) {
		mu.Lock()
		updates = append(updates, u)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "/model gpt-5-nano"}); err != nil {
		t.Fatal(err)
	}
	said, offered, current := replies(&mu, updates)
	// gobble's sessions speak dry (0022-PLAN P2): the reply is one of its
	// variants, whole.
	dry, err := personality.New(personality.Options{Name: personality.Dry})
	if err != nil {
		t.Fatal(err)
	}
	if want := dry.Variants(personality.ModelSet, personality.Args{"model": "gpt-5-nano"}); !slices.Contains(want, said) {
		t.Fatalf("/model gpt-5-nano answered %q, want one of %q", said, want)
	}
	if !slices.Equal(offered, listed) || current != "gpt-5-nano" {
		t.Fatalf("model option offers %q with %q current, want %q with gpt-5-nano", offered, current, listed)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "x"}); err != nil {
		t.Fatal(err)
	}
	if reqs := script.Requests(); len(reqs) != 1 || reqs[0].Model != "gpt-5-nano" {
		t.Fatalf("%d requests, want one naming gpt-5-nano", len(reqs))
	}
}

// configUpdate is the part of a config_option_update's params the test
// reads.
type configUpdate struct {
	Update struct {
		SessionUpdate string `json:"sessionUpdate"`
		ConfigOptions []struct {
			ID           string `json:"id"`
			CurrentValue string `json:"currentValue"`
			Options      []struct {
				Value string `json:"value"`
			} `json:"options"`
		} `json:"configOptions"`
	} `json:"update"`
}

// replies is the agent's text, and the model option of the last
// config_option_update: its values and its current value.
func replies(mu *sync.Mutex, updates []acpclient.Update) (said string, offered []string, current string) {
	mu.Lock()
	defer mu.Unlock()
	var text strings.Builder
	for _, u := range updates {
		if u.Kind == acpclient.KindAgentText {
			text.WriteString(u.Text)
		}
		var c configUpdate
		if u.Kind != acpclient.KindOther || json.Unmarshal(u.Params, &c) != nil || c.Update.SessionUpdate != "config_option_update" {
			continue
		}
		for _, o := range c.Update.ConfigOptions {
			if o.ID == "model" {
				offered, current = nil, o.CurrentValue
				for _, v := range o.Options {
					offered = append(offered, v.Value)
				}
			}
		}
	}
	return text.String(), offered, current
}
