package provider

import (
	"encoding/json/v2"
	"net/http"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// TestRefusalsPassedBack: on a canned wire, New with a saved refusal for
// openai's model m at high sends m a lower effort, never high, on its first
// request (0036-MADR D4).
func TestRefusalsPassedBack(t *testing.T) {
	c := &cannedWire{}
	p, err := New("openai", Options{
		APIKey:     "k",
		Model:      "m",
		HTTPClient: &http.Client{Transport: c},
		Retry:      &llm.Retry{Off: true},
		Refusals:   []llm.ReasoningRefusal{{Model: "m", Effort: "high"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := drain(t, p, &llm.Request{
		Model:    "m",
		Thinking: "high",
		Messages: []llm.Message{text(llm.RoleUser, "hi")},
	}); err != nil {
		t.Fatal(err)
	}
	_, bodies := c.requests()
	if len(bodies) != 1 {
		t.Fatalf("%d requests, want the first only: %q", len(bodies), bodies)
	}
	effort := bodyEffort(t, bodies[0])
	if effort == "" || effort == "high" || !lowerThanHigh(effort) {
		t.Fatalf("first request effort %q, want a lower effort, never high; body %s", effort, bodies[0])
	}
}

// bodyEffort is the effort a canned request body asks for.
func bodyEffort(t *testing.T, body string) string {
	t.Helper()
	var v any
	if err := json.Unmarshal([]byte(body), &v); err != nil {
		t.Fatal(err)
	}
	return walkEffort(v)
}

func walkEffort(v any) string {
	switch v := v.(type) {
	case map[string]any:
		for _, key := range []string{"effort", "reasoning_effort", "thinking_level", "thinkingLevel"} {
			if s, ok := v[key].(string); ok {
				return s
			}
		}
		for _, inner := range v {
			if s := walkEffort(inner); s != "" {
				return s
			}
		}
	case []any:
		for _, inner := range v {
			if s := walkEffort(inner); s != "" {
				return s
			}
		}
	}
	return ""
}

// lowerThanHigh reports that effort sits below high on the SDK's ladder.
func lowerThanHigh(effort string) bool {
	switch effort {
	case "none", "minimal", "low", "medium":
		return true
	default:
		return false
	}
}
