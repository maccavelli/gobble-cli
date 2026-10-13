package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
)

// route is one canned answer: a status and a JSON body, or a request that
// hangs until its context ends.
type route struct {
	status int
	body   string
	hang   bool
}

// canned answers its routes, keyed "host/path", and every other request
// 404; it records each request as "METHOD host/path" with its
// Authorization header. No test here reaches the network (0020-PLAN C1).
type canned struct {
	routes map[string]route
	mu     sync.Mutex
	seen   []string
	auth   []string
}

func (c *canned) RoundTrip(r *http.Request) (*http.Response, error) {
	key := r.URL.Host + r.URL.Path
	c.mu.Lock()
	c.seen = append(c.seen, r.Method+" "+key)
	c.auth = append(c.auth, r.Header.Get("Authorization"))
	c.mu.Unlock()
	rt, ok := c.routes[key]
	if !ok {
		rt = route{status: http.StatusNotFound, body: "{}"}
	}
	if rt.hang {
		<-r.Context().Done()
		return nil, r.Context().Err()
	}
	return &http.Response{StatusCode: rt.status, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(rt.body)), Request: r}, nil
}

func (c *canned) requests() ([]string, []string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.seen), slices.Clone(c.auth)
}

func cannedClient(routes map[string]route) (*canned, Options) {
	c := &canned{routes: routes}
	return c, Options{HTTPClient: &http.Client{Transport: c}}
}

const openaiModels = "api.openai.com/v1/models"

// openaiListing is the static six, two more chat models, and an embedding
// and a speech model the SDK's curation leaves out.
const openaiListing = `{"object":"list","data":[` +
	`{"id":"gpt-6-luna","object":"model"},{"id":"gpt-5.6-luna","object":"model"},{"id":"gpt-4.1-mini","object":"model"},` +
	`{"id":"gpt-5.6-terra","object":"model"},{"id":"gpt-6.1-sol","object":"model"},{"id":"gpt-6-astra","object":"model"},` +
	`{"id":"gpt-5-mini","object":"model"},{"id":"gpt-5-nano","object":"model"},` +
	`{"id":"text-embedding-4-large","object":"model"},{"id":"tts-2","object":"model"}]}`

// kiloListing is a kilo-auto router, which only the capable profile
// admits, and the static default, in the shape of Kilo's public listing.
const kiloListing = `{"data":[` +
	`{"id":"kilo-auto/efficient","name":"Auto Efficient","created":0,"architecture":{"input_modalities":["text"],"output_modalities":["text"]},` +
	`"top_provider":{"context_length":1000000,"max_completion_tokens":65536},"pricing":{"prompt":"-1","completion":"-1"},"context_length":1000000,` +
	`"supported_parameters":["max_tokens","tools","reasoning"],"preferredIndex":0},` +
	`{"id":"deepseek/deepseek-v4.1-flash","name":"DeepSeek: DeepSeek V4.1 Flash","created":1789021285,"architecture":{"input_modalities":["text"],"output_modalities":["text"]},` +
	`"top_provider":{"context_length":1048576,"max_completion_tokens":943718},"pricing":{"prompt":"0.0000003","completion":"0.0000012"},"context_length":1048576,` +
	`"supported_parameters":["max_tokens","tools","reasoning"],"preferredIndex":6}]}`

// A live listing gives the full curated catalog, past the static six, from
// one request with the key, and no generation or metadata request
// (0020-MADR D2, F7).
func TestListFullCatalog(t *testing.T) {
	c, o := cannedClient(map[string]route{openaiModels: {status: http.StatusOK, body: openaiListing}})
	l, err := List(t.Context(), "openai", "sk-test", o)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"gpt-6-luna", "gpt-5.6-luna", "gpt-4.1-mini", "gpt-5.6-terra", "gpt-6.1-sol", "gpt-6-astra", "gpt-5-nano", "gpt-5-mini"}
	if !l.Live || l.Err != nil || !slices.Equal(l.Models, want) {
		t.Fatalf("listing %+v, want live %q", l, want)
	}
	seen, auth := c.requests()
	if !slices.Equal(seen, []string{"GET " + openaiModels}) || !slices.Equal(auth, []string{"Bearer sk-test"}) {
		t.Fatalf("requests %q with %d authorization headers, want one listing with the key", seen, len(auth))
	}
}

// List ranks with the capable profile, which admits the kilo-auto router
// the utility profile leaves out, and puts it first (0020-MADR D2, F4).
func TestListCapableProfile(t *testing.T) {
	_, o := cannedClient(map[string]route{"api.kilo.ai/api/gateway/models": {status: http.StatusOK, body: kiloListing}})
	l, err := List(t.Context(), "kilo", "k", o)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"kilo-auto/efficient", "deepseek/deepseek-v4.1-flash"}; !l.Live || !slices.Equal(l.Models, want) {
		t.Fatalf("listing %+v, want live %q", l, want)
	}
}

// metadataRuns numbers TestListFetchesMetadata's metadata URLs, which the
// clock cannot: on Windows it does not move between runs (0020-PLAN P1
// deviation 2).
var metadataRuns atomic.Int64

// List fetches the model metadata the open catalogs rank with. The SDK
// fetches it alongside the listing and waits for it only when the listing
// answers, so the listing here answers (0020-PLAN P1 deviation 1); a URL of
// each run's own keeps the SDK's in-process cache, and its remembered
// failures, from answering (0020-MADR D2, F6).
func TestListFetchesMetadata(t *testing.T) {
	path := fmt.Sprintf("/%s-%d-%d.json", t.Name(), os.Getpid(), metadataRuns.Add(1))
	zen := `{"object":"list","data":[{"id":"claude-haiku-5-5","object":"model"},{"id":"deepseek-v4.1-flash","object":"model"}]}`
	c, o := cannedClient(map[string]route{"opencode.ai/zen/v1/models": {status: http.StatusOK, body: zen}})
	o.ModelMetadataURL = "https://metadata.test" + path
	l, err := List(t.Context(), "opencode-zen", "z", o)
	if err != nil {
		t.Fatal(err)
	}
	seen, _ := c.requests()
	if !l.Live || !slices.Contains(seen, "GET opencode.ai/zen/v1/models") || !slices.Contains(seen, "GET metadata.test"+path) {
		t.Fatalf("listing %+v with requests %q, want a live listing and the metadata document", l, seen)
	}
}

// A refused key or a listing past its deadline gives the static models
// with the reason's kind; an unknown provider is an error (0020-MADR D2,
// F5).
func TestListFallsBack(t *testing.T) {
	refused := `{"error":{"message":"Incorrect API key provided: sk-pl***nt","code":"invalid_api_key"}}`
	_, o := cannedClient(map[string]route{openaiModels: {status: http.StatusUnauthorized, body: refused}})
	l, err := List(t.Context(), "openai", "sk-planted", o)
	if err != nil {
		t.Fatal(err)
	}
	if l.Live || !slices.Equal(l.Models, Models("openai")) || !errors.Is(l.Err, llm.ErrAuth) || FailureKind(l.Err) != "auth" {
		t.Fatalf("refused: %+v (kind %s), want the static models and auth", l, FailureKind(l.Err))
	}

	_, o = cannedClient(map[string]route{openaiModels: {hang: true}})
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	l, err = List(ctx, "openai", "k", o)
	if err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); l.Live || !slices.Equal(l.Models, Models("openai")) || FailureKind(l.Err) != "timeout" || took > 2*time.Second {
		t.Fatalf("hung: %+v (kind %s) after %s, want the static models and timeout within 2s", l, FailureKind(l.Err), took)
	}

	if _, err := List(t.Context(), "nope", "k", Options{}); err == nil {
		t.Fatal("an unknown provider listed")
	}
}

// FailureKind names each class, wrapped or not, and never an error's
// text (0020-MADR D6).
func TestFailureKind(t *testing.T) {
	for _, k := range failureKinds {
		if got := FailureKind(fmt.Errorf("x: %w", k.err)); got != k.kind {
			t.Errorf("FailureKind(%v) = %q, want %q", k.err, got, k.kind)
		}
	}
	if got := FailureKind(errors.New("sk-pl***nt")); got != "other" {
		t.Errorf("FailureKind of an unknown error = %q, want other", got)
	}
	for _, k := range failureKinds {
		if strings.Contains(k.kind, "sk-") || strings.Contains(k.kind, " ") {
			t.Errorf("kind %q is not a bare class name", k.kind)
		}
	}
}
