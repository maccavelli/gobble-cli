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
// one request, the listing's, with the key, and no generation (0020-MADR
// D2, F7). The models past the six are the SDK's order: from
// go-llmprovider-sdk v1.14.0, newest first by release date, then by id (its
// 0041-MADR D3, llmprovider/catalog/menu.go), so the two undated ones here
// sort by id. From v1.33.0 the SDK fetches no model metadata document: the
// listing ranks by the one Options.ModelData hands in, here the bytes the
// fetch was served before (0071-MADR P4, P5). The key is each run's own,
// since from v1.14.0 the SDK caches each listing's key for the process.
func TestListFullCatalog(t *testing.T) {
	run := metadataRuns.Add(1)
	key := fmt.Sprintf("sk-test-%d-%d", os.Getpid(), run)
	c, o := cannedClient(map[string]route{openaiModels: {status: http.StatusOK, body: openaiListing}})
	o.ModelData = modelData(t, "{}")
	l, err := List(t.Context(), "openai", key, o)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"gpt-6-luna", "gpt-5.6-luna", "gpt-4.1-mini", "gpt-5.6-terra", "gpt-6.1-sol", "gpt-6-astra", "gpt-5-mini", "gpt-5-nano"}
	if !l.Live || l.Err != nil || !slices.Equal(l.Models, want) {
		t.Fatalf("listing %+v, want live %q", l, want)
	}
	seen, auth := c.requests()
	byRequest := map[string]string{}
	for i, r := range seen {
		byRequest[r] = auth[i]
	}
	if len(seen) != 1 || byRequest["GET "+openaiModels] != "Bearer "+key {
		t.Fatalf("requests %q, want one: the listing with the key", seen)
	}
}

// A listing reads the document Options.ModelData hands in, and fetches none:
// handed, List's search reads its facts; the same through a Chat
// Completions server's listing; nil hands none, so the facts are not known.
// No case requests a document (0071-MADR P3).
func TestListHandsModelData_0071(t *testing.T) {
	fetched := func(t *testing.T, seen []string) {
		t.Helper()
		for _, s := range seen {
			if f := strings.Fields(s); len(f) < 2 || !strings.HasSuffix(f[1], "/models") {
				t.Errorf("request %q, want only the listing's", s)
			}
		}
	}
	for _, c := range []struct {
		name   string
		data   *ModelData
		vision []string
	}{
		{"handed", modelData(t, factDoc), []string{"gpt-6-luna"}},
		{"none", nil, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			run := metadataRuns.Add(1)
			w, o := cannedClient(map[string]route{openaiModels: {status: http.StatusOK, body: factModels}})
			o.ModelData = c.data
			l, err := List(t.Context(), "openai", fmt.Sprintf("sk-data-%d-%d", os.Getpid(), run), o)
			if err != nil || !l.Live {
				t.Fatalf("listing %+v (%v), want live", l, err)
			}
			if got := SearchListing(l, "vision"); !slices.Equal(got, c.vision) {
				t.Errorf("vision %q, want %q", got, c.vision)
			}
			seen, _ := w.requests()
			fetched(t, seen)
		})
	}
	t.Run("chat", func(t *testing.T) {
		// A base URL and key of each run's own, so no listing the SDK cached
		// for the process answers.
		run := metadataRuns.Add(1)
		base := fmt.Sprintf("http://127.0.0.1:1/data-%d-%d/v1", os.Getpid(), run)
		wire := &chatWire{list: factModels}
		set, _ := chatSet(t, `{"providers":{"local":{"baseUrl":"`+base+`"}}}`)
		l, err := List(t.Context(), "local", fmt.Sprintf("k-data-%d-%d", os.Getpid(), run), Options{Set: set, BaseURL: base,
			HTTPClient: &http.Client{Transport: wire}, ModelData: modelData(t, factDoc)})
		if err != nil || !l.Live {
			t.Fatalf("listing %+v (%v), want live", l, err)
		}
		if got := SearchListing(l, "vision"); !slices.Equal(got, []string{"gpt-6-luna"}) {
			t.Errorf("vision %q, want the document's image model", got)
		}
		wire.mu.Lock()
		seen := slices.Clone(wire.seen)
		wire.mu.Unlock()
		fetched(t, seen)
	})
}

// modelData is doc parsed as a test's models.dev document (0071-MADR P4).
func modelData(t testing.TB, doc string) *ModelData {
	t.Helper()
	d, err := ParseModelData(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	return d
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

// metadataRuns numbers the listing tests' keys, each run's own, which the
// clock cannot: on Windows it does not move between runs (0020-PLAN P1
// deviation 2). It numbered the metadata URLs too until go-llmprovider-sdk
// v1.33.0 stopped fetching one (0071-MADR P5).
var metadataRuns atomic.Int64

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
	if l.Live || !slices.Equal(l.Models, Models("openai")) || !errors.Is(l.Err, llm.ErrAuth) || ErrorKind(l.Err) != "auth" {
		t.Fatalf("refused: %+v (kind %s), want the static models and auth", l, ErrorKind(l.Err))
	}

	_, o = cannedClient(map[string]route{openaiModels: {hang: true}})
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	start := time.Now()
	l, err = List(ctx, "openai", "k", o)
	if err != nil {
		t.Fatal(err)
	}
	if took := time.Since(start); l.Live || !slices.Equal(l.Models, Models("openai")) || ErrorKind(l.Err) != "timeout" || took > 2*time.Second {
		t.Fatalf("hung: %+v (kind %s) after %s, want the static models and timeout within 2s", l, ErrorKind(l.Err), took)
	}

	if _, err := List(t.Context(), "nope", "k", Options{}); err == nil {
		t.Fatal("an unknown provider listed")
	}
}

// A live listing says where it came from and when, and carries every usable
// id, which a later run hands back as its last good listing (0025-MADR D4).
func TestListSourceAndUsable(t *testing.T) {
	run := metadataRuns.Add(1)
	_, o := cannedClient(map[string]route{openaiModels: {status: http.StatusOK, body: openaiListing}})
	o.ModelData = modelData(t, "{}")
	before := time.Now()
	l, err := List(t.Context(), "openai", fmt.Sprintf("sk-src-%d-%d", os.Getpid(), run), o)
	if err != nil {
		t.Fatal(err)
	}
	if !l.Live || l.Source != "live" || l.FetchedAt.Before(before.Add(-time.Second)) {
		t.Fatalf("listing %+v, want live, source live, fetched now", l)
	}
	for _, m := range l.Models {
		if !slices.Contains(l.Usable, m) {
			t.Fatalf("usable %q lacks listed %q", l.Usable, m)
		}
	}
}

// After a transient failure, a last good listing from an earlier run within
// a day is served as a stale listing, with its time (0025-MADR D4).
func TestListServesLastGood(t *testing.T) {
	run := metadataRuns.Add(1)
	_, o := cannedClient(map[string]route{openaiModels: {status: http.StatusServiceUnavailable, body: `{"error":{"message":"busy"}}`}})
	o.ModelData = modelData(t, "{}")
	o.LastGood, o.LastGoodAt = []string{"gpt-6-luna", "gpt-5-mini"}, time.Now().Add(-time.Hour)
	l, err := List(t.Context(), "openai", fmt.Sprintf("sk-stale-%d-%d", os.Getpid(), run), o)
	if err != nil {
		t.Fatal(err)
	}
	if !l.Live || l.Source != "stale" || !slices.Contains(l.Models, "gpt-6-luna") || !l.FetchedAt.Equal(o.LastGoodAt) {
		t.Fatalf("listing %+v, want the last good one, stale, fetched at %s", l, o.LastGoodAt)
	}
}

// The listings that need no key, and so cannot reject one, are OpenCode Zen
// and Go's, Hugging Face's and Kilo's (go-llmprovider-sdk v1.17.0,
// llmprovider/catalog/discovery.go: 1012, 1045, 1251).
func TestPublicListing(t *testing.T) {
	public := []string{"opencode-zen", "opencode-go", "huggingface", "kilo"}
	for _, id := range append(KeyProviders(), "ollama") {
		if got := PublicListing(id); got != slices.Contains(public, id) {
			t.Errorf("PublicListing(%q) = %v", id, got)
		}
	}
}

// Search finds a model by part of its id, best first.
func TestSearch(t *testing.T) {
	got := Search("openai", []string{"gpt-6-luna", "gpt-5-mini", "gpt-5-nano"}, "mini")
	if len(got) == 0 || got[0] != "gpt-5-mini" || slices.Contains(got, "gpt-6-luna") {
		t.Fatalf("Search mini = %q, want gpt-5-mini first and no gpt-6-luna", got)
	}
}

// factWire is two chat models. Neither id contains "vision" or "200k".
// The document gives one an image input and a window under 200k, and the
// other text only and a window of 400k (0036-MADR D6).
const factModels = `{"object":"list","data":[{"id":"gpt-6-luna","object":"model"},{"id":"gpt-6-astra","object":"model"}]}`

const factDoc = `{"openai":{"id":"openai","models":{
"gpt-6-luna":{"name":"Luna","limit":{"context":128000},"modalities":{"input":["text","image"],"output":["text"]},"tool_call":true},
"gpt-6-astra":{"name":"Astra","limit":{"context":400000},"modalities":{"input":["text"],"output":["text"]},"tool_call":true}}}}`

// A live listing's search reads the document's facts. vision finds the model
// whose input holds image, and 200k the one whose window is at least
// 200,000. The id search over the same ids finds neither (0036-MADR D6).
func TestListingSearchByFacts(t *testing.T) {
	run := metadataRuns.Add(1)
	_, o := cannedClient(map[string]route{openaiModels: {status: http.StatusOK, body: factModels}})
	o.ModelData = modelData(t, factDoc)
	l, err := List(t.Context(), "openai", fmt.Sprintf("sk-facts-%d-%d", os.Getpid(), run), o)
	if err != nil {
		t.Fatal(err)
	}
	orders := !slices.Equal(l.Models, []string{"gpt-6-luna", "gpt-6-astra"}) && !slices.Equal(l.Models, []string{"gpt-6-astra", "gpt-6-luna"})
	if !l.Live || orders {
		t.Fatalf("listing %+v, want both chat models live", l)
	}
	if got := SearchListing(l, "vision"); !slices.Equal(got, []string{"gpt-6-luna"}) {
		t.Fatalf("vision %q, want the model whose input holds image", got)
	}
	if got := SearchListing(l, "200k"); !slices.Equal(got, []string{"gpt-6-astra"}) {
		t.Fatalf("200k %q, want the model whose window is at least 200000", got)
	}
	if got := Search("openai", l.Models, "vision"); len(got) != 0 {
		t.Fatalf("id search vision %q, want none", got)
	}
	if got := Search("openai", l.Models, "200k"); len(got) != 0 {
		t.Fatalf("id search 200k %q, want none", got)
	}
}
