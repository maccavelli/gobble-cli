package cli

import (
	"log/slog"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// listModels lists a provider's models. TestMain replaces it with a lister
// that fails offline, so no test lists over the network (0020-MADR D7).
var listModels = provider.List

// listRetry is the least time between a failed listing and its retry
// (0020-MADR D3).
const listRetry = time.Minute

// modelListing is the process's listing of one provider's models
// (0020-MADR D3): started once, waited for by the first reader, kept when
// live, and retried in the background at most once per listRetry when it
// failed.
type modelListing struct {
	fetch func() provider.Listing
	now   func() time.Time
	once  sync.Once
	ready chan struct{} // closed when the first listing has ended

	mu       sync.Mutex
	result   provider.Listing
	at       time.Time // when result was listed
	retrying bool
}

func newModelListing(fetch func() provider.Listing, now func() time.Time) *modelListing {
	return &modelListing{fetch: fetch, now: now, ready: make(chan struct{})}
}

// start begins the first listing, once.
func (l *modelListing) start() {
	l.once.Do(func() {
		go func() {
			r := l.fetch()
			l.mu.Lock()
			l.result, l.at = r, l.now()
			l.mu.Unlock()
			close(l.ready)
		}()
	})
}

// get waits for the first listing, starts a retry when the kept one failed
// listRetry or more ago, and returns the kept one.
func (l *modelListing) get() provider.Listing {
	l.start()
	<-l.ready
	l.mu.Lock()
	defer l.mu.Unlock()
	if !l.result.Live && !l.retrying && l.now().Sub(l.at) >= listRetry {
		l.retrying = true
		go l.retry()
	}
	return l.result
}

// retry lists again and keeps the answer, live or not, with its time.
func (l *modelListing) retry() {
	r := l.fetch()
	l.mu.Lock()
	defer l.mu.Unlock()
	l.result, l.at, l.retrying = r, l.now(), false
}

// modelListing is the listing of sel's models for this process, made with
// sel's credential. A failed listing is the static models, and a warning
// names the provider and the failure's kind, never its text (0020-MADR D6).
func (e *runEnv) modelListing(sel provider.Selection) *modelListing {
	e.listingsMu.Lock()
	defer e.listingsMu.Unlock()
	if l, ok := e.listings[sel.ID]; ok {
		return l
	}
	key := os.Getenv(sel.EnvVar)
	o := e.providerOptions(sel.ID, provider.Options{ClientName: appName, ClientVersion: identity().Version})
	l := newModelListing(func() provider.Listing {
		r, err := listModels(e.ctx, sel.ID, key, o)
		if err != nil {
			r = provider.Listing{Models: provider.Models(sel.ID), Err: err}
		}
		if !r.Live {
			e.log().WarnContext(e.ctx, "model listing failed; offering the curated static list",
				slog.String("provider", sel.ID), slog.String("kind", provider.FailureKind(r.Err)))
		}
		return r
	}, time.Now)
	if e.listings == nil {
		e.listings = map[string]*modelListing{}
	}
	e.listings[sel.ID] = l
	return l
}

// startModelListing starts the run's provider's listing when a credential
// is set, so the first session rarely waits for it (0020-MADR D3).
func (e *runEnv) startModelListing() {
	if sel, err := provider.Choose(os.Getenv, e.chosenProvider()); err == nil {
		e.modelListing(sel).start()
	}
}

// models is the listed models and the default: the listing's first when it
// is live, else the static default (0020-MADR D4). List never gives a live
// listing with no model; the length check keeps a replaced lister from
// panicking here.
func models(sel provider.Selection, l provider.Listing) ([]string, string) {
	if l.Live && len(l.Models) > 0 {
		return slices.Clone(l.Models), l.Models[0]
	}
	return slices.Clone(l.Models), sel.Model
}
