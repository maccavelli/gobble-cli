package provider

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

	"github.com/maccavelli/gobble-cli/llm"
)

// The tests of 0071-PLAN's Phase 2c in llm/provider: the caps services
// stated on earlier runs as a caller's layer (0071-MADR P21, P24, the owner's
// "Hand them in as a caller layer (Recommended)"), every call's explicit
// limit (P18, V2) and Kilo's range form (P34).

// savedModel is a Claude model no snapshot states, so only what a test hands
// states its limits.
const savedModel = "claude-saved-1"

// savedCap is a cap saved for model ago.
func savedCap(model string, tokens int, ago time.Duration, base, scope string) llm.LimitObservation {
	return llm.LimitObservation{Model: model, BaseURL: base, Scope: scope, Tokens: tokens, Seen: time.Now().Add(-ago)}
}

// The saved caps reach every lookup, the limits a call reads, discovery and
// New as the "service-stated" layer, ranked below models.json and above the
// listing; a cap 25 hours old, of another base URL or of another scope is
// not stated (0071-MADR P21, P24; 0066-MADR S14, S28, S44).
func TestSavedCapLayer_0071(t *testing.T) {
	fresh := []llm.LimitObservation{savedCap(strings.ToUpper(savedModel), 16000, time.Hour, "", "")}
	t.Run("lookup", func(t *testing.T) {
		m := ModelInfoWith(Options{Observations: fresh}, "claude", savedModel, nil)
		if m.MaxTokens != 16000 || m.Limits.Output.Source != sourceServiceStated {
			t.Errorf("MaxTokens %d, output %+v; want the saved 16,000, service-stated", m.MaxTokens, m.Limits.Output)
		}
	})
	t.Run("limits", func(t *testing.T) {
		w := &limitWire{listing: claudeListingOf(savedModel, 128000, 200000)}
		o := claudeOpts(t, w, "sk-saved-limits")
		o.Observations = fresh
		if err := Discover(t.Context(), o, "claude", savedModel, nil); err != nil {
			t.Fatal(err)
		}
		m := ModelInfoWith(o, "claude", savedModel, nil)
		if s := m.Limits.StreamedOutput; s.Tokens != 16000 || s.Source != sourceServiceStated || otherFrom(s, "listing:claude").Tokens != 128000 {
			t.Errorf("streamed %+v; want the saved 16,000 over the listing's 128,000", s)
		}
		mj := mjOf(t, `{"providers":{"claude":{"modelOverrides":{"`+savedModel+`":{"maxTokens":10000}}}}}`)
		m = ModelInfoWith(o, "claude", savedModel, mj)
		if s := m.Limits.StreamedOutput; s.Tokens != 10000 || s.Source != "models.json" || otherFrom(s, sourceServiceStated).Tokens != 16000 {
			t.Errorf("streamed %+v; want models.json's 10,000 over the saved 16,000, which ranks below it", s)
		}
	})
	t.Run("discover", func(t *testing.T) {
		var got catalog.Limits
		old := lookupLimits
		lookupLimits = func(ctx context.Context, p llmprovider.ProviderID, _ llmprovider.TokenSource, model string,
			opts ...llmprovider.Option) (catalog.Limits, error) {
			got = catalog.CachedLimits(ctx, p, model, opts...)
			return got, nil
		}
		t.Cleanup(func() { lookupLimits = old })
		if err := Discover(t.Context(), Options{Observations: fresh}, "claude", savedModel, nil); err != nil {
			t.Fatal(err)
		}
		if f := got.Output; f.Tokens != 16000 || len(f.Evidence) == 0 || f.Evidence[0].Source != sourceServiceStated {
			t.Errorf("discovery's output %+v; want the saved 16,000, service-stated", f)
		}
	})
	t.Run("new", func(t *testing.T) {
		c := &cannedWire{}
		p, err := New("claude", Options{APIKey: "k", Model: savedModel, HTTPClient: &http.Client{Transport: c}, Retry: &llm.Retry{Off: true},
			Observations: fresh})
		if err != nil {
			t.Fatal(err)
		}
		req := wireRequest()
		req.Model = savedModel // no limit named: the SDK chooses it from its sources
		if _, err := drain(t, p, req); err != nil {
			t.Fatalf("the call failed: %v; want it sent at the saved cap", err)
		}
		if got := sentMaxTokens(t, c); got != 16000 {
			t.Errorf("max_tokens %d, want the saved 16,000", got)
		}
	})
	t.Run("not_stated", func(t *testing.T) {
		for name, o := range map[string]Options{
			"25 hours old":    {Observations: []llm.LimitObservation{savedCap(savedModel, 16000, 25*time.Hour, "", "")}},
			"another base":    {Observations: []llm.LimitObservation{savedCap(savedModel, 16000, time.Hour, "https://other.example/v1", "")}},
			"another scope":   {Workspace: "wrkspc_a", Observations: []llm.LimitObservation{savedCap(savedModel, 16000, time.Hour, "", "wrkspc_b")}},
			"another model":   {Observations: []llm.LimitObservation{savedCap("claude-saved-2", 16000, time.Hour, "", "")}},
			"its own base":    {BaseURL: "https://gw.example/v1/", Observations: []llm.LimitObservation{savedCap(savedModel, 16000, time.Hour, "https://gw.example/v1", "")}},
			"its own scope":   {Workspace: "wrkspc_a", Observations: []llm.LimitObservation{savedCap(savedModel, 16000, time.Hour, "", "wrkspc_a")}},
			"an hour old too": {Observations: fresh},
		} {
			m := ModelInfoWith(o, "claude", savedModel, nil)
			stated := m.Limits.Output.Source == sourceServiceStated && m.MaxTokens == 16000
			if want := strings.HasPrefix(name, "its own") || name == "an hour old too"; stated != want {
				t.Errorf("%s: output %+v, stated %v; want %v", name, m.Limits.Output, stated, want)
			}
		}
	})
}

// A saved cap seen more than five minutes after now is not stated; one
// within the skew is (0066-MADR S14; 0071-PLAN's review deviation of
// 2026-11-25, R2).
func TestSavedCapSeenAhead_0071(t *testing.T) {
	for name, c := range map[string]struct {
		ahead  time.Duration
		stated bool
	}{"an hour ahead": {time.Hour, false}, "a minute ahead": {time.Minute, true}} {
		m := ModelInfoWith(Options{Observations: []llm.LimitObservation{savedCap(savedModel, 16000, -c.ahead, "", "")}}, "claude", savedModel, nil)
		if stated := m.Limits.Output.Source == sourceServiceStated && m.MaxTokens == 16000; stated != c.stated {
			t.Errorf("%s: output %+v, stated %v; want %v", name, m.Limits.Output, stated, c.stated)
		}
	}
}

// Every turn and every summary names its limit on every wire, optional ones
// included: the request's value is sent as given, basis request (0071-MADR
// P18, V2, the owner's "Always, gobble's policy (Recommended)").
func TestExplicitLimit_0071(t *testing.T) {
	sent := llm.FitOutput(131072, llm.CapCatalog, 1000000, 2000).Tokens // D2's value: 64,000 under the ceiling
	tools := jsontext.Value(`[{"name":"read","description":"read a file","inputSchema":{"type":"object"}}]`)
	routes := []struct{ name, id, model string }{
		{"together", "together", "zai-org/GLM-5.3"},
		{"opencode_go_chat", "opencode-go", "glm-5.3"},
		{"messages", "claude", "claude-haiku-5-5"},
	}
	for _, call := range []string{"turn", "summary"} {
		t.Run(call, func(t *testing.T) {
			for _, r := range routes {
				c := &cannedWire{}
				p, err := New(r.id, Options{APIKey: "k", Model: r.model, HTTPClient: &http.Client{Transport: c}, Retry: &llm.Retry{Off: true}})
				if err != nil {
					t.Fatal(err)
				}
				req := wireRequest()
				req.Model, req.MaxTokens = r.model, sent
				if call == "turn" {
					req.Tools = tools
				}
				evs, err := drain(t, p, req)
				if err != nil {
					t.Fatalf("%s: %v", r.name, err)
				}
				if d, want := doneOf(t, evs), (llm.SentLimit{Tokens: sent, Sent: true, Basis: "request"}); d.OutputLimit != want {
					t.Errorf("%s: Done.OutputLimit %+v, want %+v", r.name, d.OutputLimit, want)
				}
				if got := wireLimit(t, c); got != sent {
					t.Errorf("%s: the wire's limit %d, want %d", r.name, got, sent)
				}
			}
		})
	}
}

// wireLimit is the output limit the one request c saw carried, in whichever
// field its wire names it.
func wireLimit(t *testing.T, c *cannedWire) int {
	t.Helper()
	_, bodies := c.requests()
	if len(bodies) != 1 {
		t.Fatalf("%d requests, want 1", len(bodies))
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(bodies[0]), &body); err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"max_tokens", "max_completion_tokens", "max_output_tokens"} {
		if v, ok := body[f].(float64); ok {
			return int(v)
		}
	}
	return 0
}
