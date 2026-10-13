package provider

import (
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"

	"github.com/maccavelli/gobble-cli/llm"
)

// wire400 records each request's decoded body and session headers, and
// answers 400, which the SDK does not retry. No test here reaches a
// service (0021-PLAN C1).
type wire400 struct {
	mu      sync.Mutex
	bodies  []map[string]any
	headers []http.Header
}

func (w *wire400) RoundTrip(r *http.Request) (*http.Response, error) {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err := json.Unmarshal(b, &body); err != nil {
		return nil, err
	}
	w.mu.Lock()
	w.bodies = append(w.bodies, body)
	w.headers = append(w.headers, r.Header.Clone())
	w.mu.Unlock()
	return &http.Response{StatusCode: http.StatusBadRequest, Header: http.Header{"Content-Type": {"application/json"}},
		Body: io.NopCloser(strings.NewReader(`{"error":{"message":"test"}}`)), Request: r}, nil
}

func (w *wire400) requests() ([]map[string]any, []http.Header) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.bodies, w.headers
}

// send streams one request with a system and a user message through a
// provider built with o, and returns the stream's error.
func send(t *testing.T, id, model string, cache llm.CacheRetention, o Options) error {
	t.Helper()
	p, err := New(id, o)
	if err != nil {
		t.Fatal(err)
	}
	req := &llm.Request{Model: model, Cache: cache, Messages: []llm.Message{text(llm.RoleSystem, "be brief"), text(llm.RoleUser, "hi")}}
	for _, err := range p.Stream(t.Context(), req) {
		if err != nil {
			return err
		}
	}
	return nil
}

// marker is a cache_control's TTL, or "" for none.
func marker(v any) string {
	m, ok := v.(map[string]any)
	if !ok || m["type"] != "ephemeral" {
		return ""
	}
	ttl, _ := m["ttl"].(string) //nolint:errcheck // a missing TTL is none
	return ttl
}

// systemMarker is the TTL on the last system block, or "" when the system
// prompt is a string or carries none.
func systemMarker(body map[string]any) string {
	blocks, ok := body["system"].([]any)
	if !ok || len(blocks) == 0 {
		return ""
	}
	last, _ := blocks[len(blocks)-1].(map[string]any) //nolint:errcheck // not an object is none
	return marker(last["cache_control"])
}

// Each wire carries the cache fields the retention asks for, the routing key
// is the session id, and "" sends none of them (0021-MADR D1, D4; 0021-PLAN
// C2).
func TestCacheWire(t *testing.T) {
	for _, c := range []struct {
		id, model     string
		cache         llm.CacheRetention
		top, system   string // cache_control TTLs, "" for none
		key           bool   // prompt_cache_key is the session id
		header, value string // a session header, and its value
	}{
		{id: "claude", model: "claude-haiku-5-5", cache: llm.CacheNone},
		{id: "claude", model: "claude-haiku-5-5", cache: llm.CacheShort, top: "5m", system: "5m"},
		{id: "claude", model: "claude-haiku-5-5", cache: llm.CacheLong, top: "1h", system: "1h"},
		{id: "openai", model: "gpt-6-luna", cache: llm.CacheNone},
		{id: "openai", model: "gpt-6-luna", cache: llm.CacheShort, key: true},
		{id: "kilo", model: "anthropic/claude-haiku-5.5", cache: llm.CacheShort, top: "5m", header: "X-KiloCode-TaskId", value: "sess-1"},
		{id: "opencode-zen", model: "claude-haiku-5-5", cache: llm.CacheLong, top: "1h", system: "1h", header: "x-opencode-session", value: "sess-1"},
		{id: "gemini", model: "gemini-3.5-flash-lite", cache: llm.CacheShort},
	} {
		t.Run(c.id+"/"+string(c.cache), func(t *testing.T) {
			w := &wire400{}
			_ = send(t, c.id, c.model, c.cache, Options{APIKey: "k", SessionID: "sess-1", HTTPClient: &http.Client{Transport: w}}) //nolint:errcheck // the 400 is the test's
			bodies, headers := w.requests()
			if len(bodies) != 1 {
				t.Fatalf("%d requests, want 1", len(bodies))
			}
			body := bodies[0]
			if got := marker(body["cache_control"]); got != c.top {
				t.Errorf("cache_control %q, want %q", got, c.top)
			}
			if got := systemMarker(body); got != c.system {
				t.Errorf("system block's cache_control %q, want %q", got, c.system)
			}
			if key, ok := body["prompt_cache_key"]; ok != c.key || (c.key && key != "sess-1") {
				t.Errorf("prompt_cache_key %v (sent %v), want sent %v as sess-1", key, ok, c.key)
			}
			if c.header != "" && headers[0].Get(c.header) != c.value {
				t.Errorf("%s %q, want %q", c.header, headers[0].Get(c.header), c.value)
			}
		})
	}
}

// An unknown retention is refused before any request: the SDK's check,
// which gobble does not repeat (0021-MADR D1).
func TestCacheUnknownRefused(t *testing.T) {
	w := &wire400{}
	err := send(t, "claude", "claude-haiku-5-5", "forever", Options{APIKey: "k", HTTPClient: &http.Client{Transport: w}})
	if !errors.Is(err, llmprovider.ErrInvalidRequest) || !strings.Contains(err.Error(), "unknown cache retention") {
		t.Fatalf("error %v, want the SDK's unknown cache retention", err)
	}
	if bodies, _ := w.requests(); len(bodies) != 0 {
		t.Fatalf("%d requests sent, want none", len(bodies))
	}
}

// Each provider sends its own session id; with none, the SDK's random one,
// which is neither (0021-MADR D4).
func TestSessionIDPerProvider(t *testing.T) {
	sent := func(session string) string {
		t.Helper()
		w := &wire400{}
		_ = send(t, "opencode-zen", "claude-haiku-5-5", llm.CacheNone, Options{APIKey: "k", SessionID: session, HTTPClient: &http.Client{Transport: w}}) //nolint:errcheck // the 400 is the test's
		_, headers := w.requests()
		if len(headers) != 1 {
			t.Fatalf("%d requests, want 1", len(headers))
		}
		return headers[0].Get("x-opencode-session")
	}
	a, b, random := sent("a"), sent("b"), sent("")
	if a != "a" || b != "b" || random == "" || random == "a" || random == "b" {
		t.Fatalf("session headers %q, %q and %q; want a, b and a random id", a, b, random)
	}
}
