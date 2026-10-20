package cli

import (
	"errors"
	"log/slog"
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

// errNoListingKey is a listing not made, because the provider's key did not
// resolve (0005-PLAN F4-3a deviation 3).
var errNoListingKey = errors.New("the provider's key did not resolve, so its models were not listed")

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

	// kept is the listing an earlier run kept on disk, shown while this
	// one runs (0025-MADR D4).
	kept kept
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

// peek is the kept listing without waiting: ok is false while the first
// listing runs. It starts the listing, and a retry as get does.
func (l *modelListing) peek() (provider.Listing, bool) {
	l.start()
	select {
	case <-l.ready:
		return l.get(), true
	default:
		return provider.Listing{}, false
	}
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
	key := e.knownKey(sel.ID) // in Pi's order (0005-PLAN F4-3a)
	login := key == "" && e.knownLogin(sel.ID)
	o := e.providerOptions(sel.ID, provider.Options{ClientName: appName, ClientVersion: identity().Version})
	if login {
		o.Logins = e.logins() // a login lists as a key does (0005-PLAN F4-3b decision 9)
	}
	// The listing an earlier run kept is shown while this one runs, and
	// served after a transient failure (0025-MADR D4).
	k := readKept(sel.ID, time.Now())
	o.LastGood, o.LastGoodAt = k.usable, k.at
	l := newModelListing(func() provider.Listing {
		if key == "" && !login {
			// A key that did not resolve, already warned of, lists nothing: no
			// request goes without one (0005-PLAN F4-3a deviation 3).
			return provider.Listing{Models: provider.Models(sel.ID), Err: errNoListingKey}
		}
		r, err := listModels(e.ctx, sel.ID, key, o)
		if err != nil {
			r = provider.Listing{Models: provider.Models(sel.ID), Err: err}
		}
		if !r.Live {
			e.log().WarnContext(e.ctx, "model listing failed; offering the curated static list",
				slog.String("provider", sel.ID), slog.String("kind", provider.FailureKind(r.Err)))
		}
		if r.Live && r.Source == "live" {
			writeKept(sel.ID, r)
		}
		return r
	}, time.Now)
	l.kept = k
	if e.listings == nil {
		e.listings = map[string]*modelListing{}
	}
	e.listings[sel.ID] = l
	return l
}

// startModelListing starts every credentialed provider's listing, the run's
// first, so the first session rarely waits for it (0020-MADR D3) and the
// others' states are ready soon after (0025-MADR D4).
func (e *runEnv) startModelListing() {
	if sel, err := provider.ChooseWith(e.hasCredential, e.chosenProvider()); err == nil {
		e.modelListing(sel).start()
	}
	for _, id := range provider.KeyProviders() {
		if e.hasCredential(id) {
			e.modelListing(provider.Selection{ID: id}).start()
		}
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
