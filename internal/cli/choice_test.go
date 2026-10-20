package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// envHere is a runEnv as modelEnv builds one, in the home already set.
func envHere(t *testing.T) *runEnv {
	t.Helper()
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out, stderr: &syncBuffer{}}}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	e.processSettings("")
	return e
}

// settled is the model choice once no provider is still listing.
func settled(t *testing.T, e *runEnv) acpserver.ModelChoice {
	t.Helper()
	var c acpserver.ModelChoice
	if !waitFor(func() bool {
		c = e.modelChoice()
		return !slices.ContainsFunc(c.Providers, func(p acpserver.ProviderChoice) bool { return p.State == acpserver.StateListing })
	}) {
		t.Fatalf("still listing: %+v", c.Providers)
	}
	return c
}

func byID(c acpserver.ModelChoice) map[string]acpserver.ProviderChoice {
	m := map[string]acpserver.ProviderChoice{}
	for _, p := range c.Providers {
		m[p.ID] = p
	}
	return m
}

// The model choice holds every key provider, in registry order, with its
// state: a live listing's models; a refused key's status and way out, never
// the provider's text; an offline one's static models; a public listing,
// not checked; a provider with no credential, its variable (0025-MADR D4,
// deviation 1).
func TestProviderChoicesStates(t *testing.T) {
	e, _ := modelEnv(t)
	for _, v := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "XAI_API_KEY", "OPENCODE_API_KEY"} {
		t.Setenv(v, "test-only")
	}
	withLister(t, func(_ context.Context, id, _ string, _ provider.Options) (provider.Listing, error) {
		switch id {
		case "openai":
			return provider.Listing{Live: true, Source: "live", Models: []string{"gpt-a", "gpt-b"}, FetchedAt: time.Now()}, nil
		case "claude":
			return provider.Listing{}, &llm.APIError{Status: 401, Err: fmt.Errorf("%w: Incorrect API key provided: sk-pl***nt", llm.ErrAuth)}
		case "grok":
			return provider.Listing{}, fmt.Errorf("list: %w", llm.ErrOffline)
		case "opencode-zen", "opencode-go":
			return provider.Listing{Live: true, Source: "live", Models: []string{"zen-1"}, FetchedAt: time.Now()}, nil
		}
		return provider.Listing{}, errors.New("unexpected listing of " + id)
	})
	c := settled(t, e)
	var ids []string
	for _, p := range c.Providers {
		ids = append(ids, p.ID)
	}
	if !slices.Equal(ids, provider.KeyProviders()) {
		t.Fatalf("providers %q, want %q", ids, provider.KeyProviders())
	}
	got := byID(c)
	check := func(id string, state acpserver.ProviderState, reason string, models []string) {
		t.Helper()
		p := got[id]
		if p.State != state || p.Reason != reason || !slices.Equal(p.Models, models) || p.Label != provider.Label(id) {
			t.Errorf("%s: %+v, want %s %q %q", id, p, state, reason, models)
		}
	}
	check("gemini", acpserver.StateNoCredential, "— set GEMINI_API_KEY, or gobble auth login gemini", nil)
	check("openai", acpserver.StateLive, "", []string{"gpt-a", "gpt-b"})
	check("claude", acpserver.StateKeyRejected, "(HTTP 401) — gobble auth login claude", nil)
	check("grok", acpserver.StateNotListed, "(offline)", provider.Models("grok"))
	check("opencode-zen", acpserver.StateNotChecked, "", []string{"zen-1"})
	check("huggingface", acpserver.StateNoCredential, "— set HF_TOKEN, or gobble auth login huggingface", nil)
	if c.Provider != "openai" || c.Default != "gpt-a" {
		t.Errorf("the run's provider %q default %q, want openai and gpt-a", c.Provider, c.Default)
	}
}

// The listings run at once, each in its own time: a slow one is shown as
// listing while the others' results are in (0025-MADR D4).
func TestProviderChoicesConcurrent(t *testing.T) {
	e, _ := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	t.Setenv("ANTHROPIC_API_KEY", "test-only")
	release := make(chan struct{})
	withLister(t, func(ctx context.Context, id, _ string, _ provider.Options) (provider.Listing, error) {
		if id == "claude" {
			select {
			case <-release:
			case <-ctx.Done():
			}
		}
		return provider.Listing{Live: true, Source: "live", Models: []string{id + "-1"}, FetchedAt: time.Now()}, nil
	})
	done := make(chan acpserver.ModelChoice, 1)
	go func() { done <- e.modelChoice() }()
	select {
	case c := <-done:
		got := byID(c)
		if got["openai"].State != acpserver.StateLive || got["claude"].State != acpserver.StateListing {
			t.Fatalf("openai %+v, claude %+v; want live and listing", got["openai"], got["claude"])
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the model choice waited for claude's listing")
	}
	close(release)
	settled(t, e) // claude's listing ends, and keeps its file, before the home is removed
}

// recent-models.json keeps ten recent models, newest first, each once,
// owner-only, and leaves the custom models' models.json beside it, under
// GOBBLE_HOME, as it was; an unreadable one is none, with a warning
// (0025-MADR D3, deviation 4).
func TestRecentModels(t *testing.T) {
	e, _ := modelEnv(t)
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	custom := []byte(`{"providers":{"local":{"baseUrl":"http://127.0.0.1:1/v1","models":[{"id":"mine"}]}}}`)
	if dirs.Config != dirs.State {
		t.Fatalf("under GOBBLE_HOME the config and state directories are one: %s, %s", dirs.Config, dirs.State)
	}
	writeFile(t, filepath.Join(dirs.Config, "models.json"), string(custom))
	for i := range 12 {
		e.rememberModel(fmt.Sprintf("openai/m%d", i))
	}
	e.rememberModel("openai/m5")
	got := e.recentModels()
	want := []string{"openai/m5", "openai/m11", "openai/m10", "openai/m9", "openai/m8", "openai/m7", "openai/m6", "openai/m4", "openai/m3", "openai/m2"}
	if !slices.Equal(got, want) {
		t.Fatalf("recent %q, want %q", got, want)
	}
	if b, err := os.ReadFile(filepath.Join(dirs.Config, "models.json")); err != nil || !bytes.Equal(b, custom) {
		t.Fatalf("the custom models.json is now %q (%v), want it as it was", b, err)
	}
	path := filepath.Join(dirs.State, "recent-models.json")
	if info, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("recent-models.json mode %v, want 0600", info.Mode().Perm())
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	next := envHere(t) // the file is read once a run
	stderr := next.out.err.(*bytes.Buffer)
	if got := next.recentModels(); got != nil || !strings.Contains(stderr.String(), "Warning: recent-models.json") {
		t.Fatalf("an unreadable recent-models.json gave %q, warnings %q", got, stderr.String())
	}
}

// A live listing is kept on disk; the next run shows it, stale, while its
// own listing runs, and hands it to the listing as its last good one
// (0025-MADR D4).
func TestLastGoodOnDisk(t *testing.T) {
	modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	t.Setenv("ANTHROPIC_API_KEY", "test-only")
	at := time.Now().Add(-2 * time.Hour).Round(time.Second)
	withLister(t, func(_ context.Context, id, _ string, _ provider.Options) (provider.Listing, error) {
		if id == "claude" {
			return provider.Listing{Live: true, Source: "live", Models: []string{"c1", "c2"}, Usable: []string{"c1", "c2", "c3"}, FetchedAt: at}, nil
		}
		return provider.Listing{Live: true, Source: "live", Models: []string{id + "-1"}, FetchedAt: time.Now()}, nil
	})
	settled(t, envHere(t)) // the first run lists claude live, and keeps it

	release := make(chan struct{})
	var mu sync.Mutex
	var handed provider.Options
	withLister(t, func(ctx context.Context, id, _ string, o provider.Options) (provider.Listing, error) {
		if id == "claude" {
			mu.Lock()
			handed = o
			mu.Unlock()
			select {
			case <-release:
			case <-ctx.Done():
			}
		}
		return provider.Listing{Live: true, Source: "live", Models: []string{id + "-1"}, FetchedAt: time.Now()}, nil
	})
	e := envHere(t)
	claude := byID(e.modelChoice())["claude"]
	if claude.State != acpserver.StateStale || !slices.Equal(claude.Models, []string{"c1", "c2"}) || claude.Age < time.Hour {
		t.Fatalf("claude while listing: %+v, want the kept listing, stale, two hours old", claude)
	}
	var got provider.Options
	waitFor(func() bool { // claude's listing runs on its own goroutine, and may not have begun
		mu.Lock()
		defer mu.Unlock()
		got = handed
		return got.LastGood != nil
	})
	if !slices.Equal(got.LastGood, []string{"c1", "c2", "c3"}) || !got.LastGoodAt.Equal(at) {
		t.Fatalf("the listing was handed %q at %s, want the kept usable ids at %s", got.LastGood, got.LastGoodAt, at)
	}
	close(release)
	settled(t, e) // claude's listing ends before the home is removed
}

// The agent remembers what /model and the option choose, and searches with
// the SDK's matcher; --model's choice is remembered too (0025-MADR D3, D5).
func TestChoiceWiring(t *testing.T) {
	e, _ := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	withLister(t, (&liveLister{ids: []string{"gpt-a"}}).list)
	o := e.agentOptions()
	if o.Remember == nil || o.SearchModels == nil {
		t.Fatal("the agent remembers nothing or searches without the SDK's matcher")
	}
	o.Remember("openai/gpt-a")
	if got := o.Models().Recent; !slices.Equal(got, []string{"openai/gpt-a"}) {
		t.Fatalf("recent %q", got)
	}
	static := provider.Models("openai")[0]
	if _, err := e.pickModel(&SharedFlags{Model: "openai/" + static}); err != nil {
		t.Fatal(err)
	}
	if got := e.recentModels(); len(got) == 0 || got[0] != "openai/"+static {
		t.Fatalf("--model was not remembered: %q", got)
	}
}
