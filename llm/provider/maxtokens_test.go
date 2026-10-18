package provider

import (
	"net/http"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// capOf is the value of the first field whose name holds "max" and
// "token", at any depth of body, or nil when there is none.
func capOf(v any) any {
	switch v := v.(type) {
	case map[string]any:
		for k, x := range v {
			if l := strings.ToLower(k); strings.Contains(l, "max") && strings.Contains(l, "token") {
				return x
			}
			if got := capOf(x); got != nil {
				return got
			}
		}
	case []any:
		for _, x := range v {
			if got := capOf(x); got != nil {
				return got
			}
		}
	}
	return nil
}

// A request's MaxTokens reaches each wire's body as its output cap
// (0005-PLAN F5-2 decision 7).
func TestMaxTokensWire(t *testing.T) {
	for _, c := range []struct{ id, model string }{
		{"claude", "claude-haiku-5-5"},
		{"openai", "gpt-6-luna"},
		{"gemini", "gemini-3.5-flash-lite"},
		{"grok", "grok-4.6"},
		{"kilo", "anthropic/claude-haiku-5.5"},
	} {
		t.Run(c.id, func(t *testing.T) {
			const n = 1234
			w := &wire400{}
			p, err := New(c.id, Options{APIKey: "k", HTTPClient: &http.Client{Transport: w}})
			if err != nil {
				t.Fatal(err)
			}
			req := &llm.Request{Model: c.model, MaxTokens: n, Messages: []llm.Message{text(llm.RoleUser, "hi")}}
			for range p.Stream(t.Context(), req) { //nolint:revive // the 400 is the test's; only the body matters
			}
			bodies, _ := w.requests()
			if len(bodies) != 1 {
				t.Fatalf("%d requests, want 1", len(bodies))
			}
			if got := capOf(bodies[0]); got != float64(n) {
				t.Errorf("the body's cap %v, want %d", got, n)
			}
		})
	}
}
