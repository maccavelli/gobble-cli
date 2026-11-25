package provider

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

	"github.com/maccavelli/gobble-cli/llm"
)

// budgetLayer states that every model reasons with a budget of 1,024 to
// 32,000 tokens.
type budgetLayer struct{}

func (budgetLayer) Name() string { return "test" }

func (budgetLayer) Model(_ llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	return catalog.ModelInfo{ID: id, Reasoning: catalog.ModelReasoning{Supported: true, BudgetMin: 1024, BudgetMax: 32000}}.
		Known(catalog.FieldReasoning, catalog.FieldReasoningBudget), true
}

// doneOf is the one llm.Done of evs, failing the test unless there is
// exactly one.
func doneOf(t *testing.T, evs []llm.Event) llm.Done {
	t.Helper()
	var dones []llm.Done
	for _, ev := range evs {
		if d, ok := ev.(llm.Done); ok {
			dones = append(dones, d)
		}
	}
	if len(dones) != 1 {
		t.Fatalf("%d Done events, want 1", len(dones))
	}
	return dones[0]
}

// sentMaxTokens is the max_tokens of the one request c saw.
func sentMaxTokens(t *testing.T, c *cannedWire) int {
	t.Helper()
	_, bodies := c.requests()
	if len(bodies) != 1 {
		t.Fatalf("%d requests, want 1", len(bodies))
	}
	var body struct {
		MaxTokens int `json:"max_tokens"`
	}
	if err := json.Unmarshal([]byte(bodies[0]), &body, json.RejectUnknownMembers(false)); err != nil {
		t.Fatal(err)
	}
	return body.MaxTokens
}

// The limit a call carried reaches llm.Done from the SDK's report: gobble's
// value as given, basis request; and on Messages, where a thinking budget
// raises a small limit, the value on the wire (0071-MADR P14; F9).
func TestDoneCarriesOutputLimit_0071(t *testing.T) {
	t.Run("asked", func(t *testing.T) {
		p, err := New("claude", Options{APIKey: "k", Model: "claude-haiku-5-5", HTTPClient: &http.Client{Transport: &cannedWire{}},
			Retry: &llm.Retry{Off: true}})
		if err != nil {
			t.Fatal(err)
		}
		req := wireRequest()
		req.Model, req.MaxTokens = "claude-haiku-5-5", 32000
		evs, err := drain(t, p, req)
		if err != nil {
			t.Fatal(err)
		}
		d := doneOf(t, evs)
		if want := (llm.SentLimit{Tokens: 32000, Sent: true, Basis: "request"}); d.OutputLimit != want {
			t.Errorf("Done.OutputLimit %+v, want %+v", d.OutputLimit, want)
		}
	})
	t.Run("raised_for_thinking", func(t *testing.T) {
		// A model whose facts state a thinking budget takes an enabled
		// budget, which raises a limit not above it (the SDK's
		// messages.AddThinkingFor).
		c := &cannedWire{}
		raw, err := NewSet().registry().New(llmprovider.ProviderClaude, llmprovider.WithAPIKey("k"), llmprovider.WithModel("budgeted"),
			llmprovider.WithModelProbes(false), llmprovider.WithHTTPClient(&http.Client{Transport: c}), catalog.WithModelLayer(budgetLayer{}))
		if err != nil {
			t.Fatal(err)
		}
		req := wireRequest()
		req.Model, req.Thinking, req.MaxTokens = "budgeted", "high", 2000
		evs, err := drain(t, Wrap(raw), req)
		if err != nil {
			t.Fatal(err)
		}
		d := doneOf(t, evs)
		wire := sentMaxTokens(t, c)
		if wire <= 2000 || d.OutputLimit.Tokens != wire || !d.OutputLimit.Sent || d.OutputLimit.Basis != "request" {
			t.Errorf("Done.OutputLimit %+v with %d on the wire; want the raised wire value, sent, basis request", d.OutputLimit, wire)
		}
	})
}

// A refusal of the call's limit that states the model's cap reaches llm's
// error as Observed, through the SDK's stream retry; a window form states no
// cap and leaves it zero (0071-MADR P15).
func TestAPIErrorCarriesObserved_0071(t *testing.T) {
	refused := func(t *testing.T, id, model, body string, limit int) *llm.APIError {
		t.Helper()
		// Retry nil: wrapped in the SDK's stream retry, which carries the
		// observation to the final error.
		p, err := New(id, Options{APIKey: "k", Model: model,
			HTTPClient: &http.Client{Transport: &cannedWire{status: http.StatusBadRequest, body: body}}})
		if err != nil {
			t.Fatal(err)
		}
		req := wireRequest()
		req.Model, req.MaxTokens = model, limit
		_, err = drain(t, p, req)
		ae, ok := errors.AsType[*llm.APIError](err)
		if !ok {
			t.Fatalf("err %v, want an *llm.APIError", err)
		}
		return ae
	}
	t.Run("anthropic_cap", func(t *testing.T) {
		before := time.Now()
		ae := refused(t, "claude", "claude-haiku-5-5", `{"type":"error","error":{"type":"invalid_request_error",`+
			`"message":"max_tokens: 64000 > 32000, which is the maximum allowed number of output tokens for claude-haiku-5-5"}}`, 64000)
		ob := ae.Observed
		if ob.Model != "claude-haiku-5-5" || ob.BaseURL != "" || ob.Scope != "" || ob.Tokens != 32000 || ob.Seen.Before(before.Add(-time.Second)) {
			t.Errorf("Observed %+v, want claude-haiku-5-5, no base URL or scope, 32000, seen now", ob)
		}
	})
	t.Run("openrouter_window", func(t *testing.T) {
		ae := refused(t, "kilo", "deepseek/deepseek-v4.1-flash", `{"error":{"message":"This endpoint's maximum context length is 131072 tokens. `+
			`However, you requested about 160002 tokens (100000 of text input, 60002 in the output).","type":"invalid_request_error","code":400}}`, 60002)
		if ae.Observed != (llm.LimitObservation{}) {
			t.Errorf("Observed %+v, want none for a window form", ae.Observed)
		}
	})
}
