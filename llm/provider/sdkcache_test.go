package provider

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
)

// hangingListing answers every request by waiting for its context to end,
// and counts the requests that started and the ones that returned. No test
// here reaches the network (0020-PLAN C1).
type hangingListing struct {
	started, returned atomic.Int64
	first             chan struct{}
}

func newHangingListing() *hangingListing { return &hangingListing{first: make(chan struct{})} }

func (h *hangingListing) RoundTrip(r *http.Request) (*http.Response, error) {
	if h.started.Add(1) == 1 {
		close(h.first)
	}
	defer h.returned.Add(1)
	<-r.Context().Done()
	return nil, r.Context().Err()
}

// The run's cache reaches the SDK through Options (0046-MADR D7). A listing
// given a cache lists into it, and once the cache is closed a listing given
// it is refused, while one without it still lists. A model lookup given an
// open cache starts its background fetch under that cache, so Close cancels
// the fetch and returns only after it has ended.
func TestOptionsCacheReachesTheSDK(t *testing.T) {
	run := metadataRuns.Add(1)
	metadata := fmt.Sprintf("metadata.test/%s-%d-%d.json", t.Name(), os.Getpid(), run)
	key := fmt.Sprintf("sk-test-%d-%d", os.Getpid(), run)
	_, o := cannedClient(map[string]route{openaiModels: {status: http.StatusOK, body: openaiListing},
		metadata: {status: http.StatusOK, body: "{}"}})
	o.ModelMetadataURL = "https://" + metadata
	c := &Cache{}
	o.Cache = c
	if l, err := List(t.Context(), "openai", key, o); err != nil || !l.Live {
		t.Fatalf("a listing given the cache: %+v (%v), want live", l, err)
	}
	if err := c.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if l, err := List(t.Context(), "openai", key, o); !errors.Is(err, llmprovider.ErrInvalidRequest) {
		t.Fatalf("a listing given the closed cache: %+v (%v), want the SDK's refusal, ErrInvalidRequest", l, err)
	}
	o.Cache = nil
	if l, err := List(t.Context(), "openai", key, o); err != nil || !l.Live {
		t.Fatalf("a listing given no cache: %+v (%v), want live", l, err)
	}

	h := newHangingListing()
	open := &Cache{}
	facts := Options{APIKey: fmt.Sprintf("sk-facts-%d-%d", os.Getpid(), run), HTTPClient: &http.Client{Transport: h}, Cache: open}
	_ = ModelInfoWith(facts, "openai", "gpt-5-nano", nil)
	select {
	case <-h.first:
	case <-time.After(5 * time.Second):
		t.Fatal("the lookup started no background fetch")
	}
	if err := open.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	if s, r := h.started.Load(), h.returned.Load(); r != s {
		t.Fatalf("after Close, %d of %d fetches had returned, want every one: the fetch is not the cache's", r, s)
	}
}
