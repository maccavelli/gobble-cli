package cli

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// stalledFetch answers a request by waiting: for its context, or, with
// release set, for release alone. It counts the requests that started and
// the ones that returned. Its client is the only one the request meets, so
// nothing reaches the network.
type stalledFetch struct {
	release           chan struct{}
	started, returned atomic.Int64
	first             chan struct{}
}

func newStalledFetch(release chan struct{}) *stalledFetch {
	return &stalledFetch{release: release, first: make(chan struct{})}
}

func (s *stalledFetch) RoundTrip(r *http.Request) (*http.Response, error) {
	if s.started.Add(1) == 1 {
		close(s.first)
	}
	defer s.returned.Add(1)
	if s.release != nil {
		<-s.release // the context is ignored
		return nil, errors.New("released")
	}
	<-r.Context().Done()
	return nil, r.Context().Err()
}

// startFact starts a model lookup's background fetch through e's cache, as
// the agent's ModelInfo hook does with a key, and waits for it to begin.
func startFact(t *testing.T, e *runEnv, s *stalledFetch) {
	t.Helper()
	o := provider.Options{APIKey: "test-only", HTTPClient: &http.Client{Transport: s}, Cache: &e.cache}
	_ = provider.ModelInfoWith(o, "openai", "gpt-5-nano", nil)
	select {
	case <-s.first:
	case <-time.After(5 * time.Second):
		t.Fatal("the lookup started no background fetch")
	}
}

// Every provider.Options the run builds carries the run's cache: the run's
// provider, with and without a chosen provider; a switched session's; the
// listing's; the ModelInfo hook's; and providerOptions, which the small
// model and the sign-in build on (0046-MADR D7).
func TestRunOptionsCarryTheCache(t *testing.T) {
	e, _ := modelEnv(t)
	want := &e.cache
	var built []*provider.Cache
	old := ambientProvider
	ambientProvider = func(_ func(string) (string, bool), _ string, o provider.Options) (llm.Provider, provider.Selection, error) {
		built = append(built, o.Cache)
		return nil, provider.Selection{}, errors.New("recorded")
	}
	t.Cleanup(func() { ambientProvider = old })
	_, _ = modelProvider(e)("s1", llm.DefaultRetry) // no credential: no provider is chosen
	t.Setenv("OPENAI_API_KEY", "test-only")
	e.refreshCredentials("openai")
	_, _ = modelProvider(e)("s2", llm.DefaultRetry)
	_, _ = providerFor(e)("openai", "s3", llm.DefaultRetry)
	for i, c := range built {
		if c != want {
			t.Errorf("provider %d built with cache %p, want the run's %p", i, c, want)
		}
	}
	if len(built) != 3 {
		t.Errorf("%d providers built, want 3", len(built))
	}
	if c := e.providerOptions("openai", provider.Options{}).Cache; c != want {
		t.Errorf("providerOptions' cache %p, want the run's %p", c, want)
	}
	listed := make(chan *provider.Cache, 1)
	withLister(t, func(_ context.Context, _, _ string, o provider.Options) (provider.Listing, error) {
		listed <- o.Cache
		return provider.Listing{}, errors.New("recorded")
	})
	l := e.modelListing(provider.Selection{ID: "openai"})
	l.start()
	if c := <-listed; c != want {
		t.Errorf("the listing's cache %p, want the run's %p", c, want)
	}
	oldFacts := modelFacts
	var facts *provider.Cache
	modelFacts = func(o provider.Options, id, model string, mj *catalog.ModelsJSON) llm.Model {
		facts = o.Cache
		return llm.Model{Provider: id, ID: model}
	}
	t.Cleanup(func() { modelFacts = oldFacts })
	_ = e.agentOptions().ModelInfo("openai", "gpt-5-nano")
	if facts != want {
		t.Errorf("the ModelInfo hook's cache %p, want the run's %p", facts, want)
	}
}

// endListings closes the run's cache: a model lookup's background fetch,
// waiting on its context, is cancelled and has returned when endListings
// does (0046-MADR D7).
func TestEndListingsClosesTheCache(t *testing.T) {
	e, _ := modelEnv(t)
	s := newStalledFetch(nil)
	startFact(t, e, s)
	e.endListings(t.Context())
	if st, r := s.started.Load(), s.returned.Load(); r != st {
		t.Fatalf("after endListings, %d of %d fetches had returned, want every one", r, st)
	}
}

// A fetch that ignores its cancellation holds endListings for the bound at
// most, and the Warn record says the cache's work was not seen to end
// (0046-MADR D7, Q4).
func TestEndListingsCacheBound(t *testing.T) {
	e, warnings := modelEnv(t)
	old := endListingsWait
	endListingsWait = 50 * time.Millisecond
	t.Cleanup(func() { endListingsWait = old })
	release := make(chan struct{})
	s := newStalledFetch(release)
	startFact(t, e, s)
	begin := time.Now()
	e.endListings(t.Context())
	took := time.Since(begin)
	close(release)
	if !waitFor(func() bool { return s.returned.Load() == s.started.Load() }) {
		t.Fatal("the fetch did not return once released")
	}
	if took < endListingsWait || took > 2*time.Second {
		t.Errorf("endListings took %s, want the bound, %s", took, endListingsWait)
	}
	if w := warnings.String(); !strings.Contains(w, "model listings still running at the end of the run") || !strings.Contains(w, "cache") {
		t.Errorf("warnings %q, want the record naming the cache", w)
	}
}
