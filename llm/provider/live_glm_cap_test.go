//go:build live

package provider_test

import (
	"errors"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// Whether each service accepts the output limit gobble sends a GLM-5.3
// model, one short streamed prompt each, at the limit llm.FitOutput gives
// for the snapshot's cap and window (0071-MADR L34, Q1 "Yes, short live
// calls (Recommended)"). It is a measurement, not a red test: it logs only
// the limit sent, "ok" for a stream that succeeded, the finish reason, the
// token counts and a failure's kind and HTTP status, never a key, a reply or
// a message, and it
// fails on a refusal, which is a stop for the record.
func TestLive_GLMCapAccepted_0071(t *testing.T) {
	for _, c := range []struct {
		id, model, env string
	}{
		{"kilo", "z-ai/glm-5.3", "KILO_API_KEY"},
		{"kilo", "z-ai/glm-5.3-flash", "KILO_API_KEY"},
		{"opencode-go", "glm-5.3", "OPENCODE_API_KEY"},
		{"opencode-go", "glm-5.3-flash", "OPENCODE_API_KEY"},
		{"together", "zai-org/GLM-5.3", "TOGETHER_API_KEY"},
		{"together", "zai-org/GLM-5.3-Flash", "TOGETHER_API_KEY"},
	} {
		t.Run(c.id+"/"+c.model, func(t *testing.T) {
			key := llmtest.LiveCredential(t, c.env)
			info := provider.ModelInfo(c.id, "", "", nil, c.model, nil)
			msgs := []llm.Message{{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "Reply with the single word: ok."}}}}
			lim := llm.FitOutput(info.MaxTokens, info.MaxTokensFrom, info.ContextWindow, 16)
			p, err := provider.New(c.id, provider.Options{APIKey: key, Model: c.model, ClientName: "gobble-live-test"})
			if err != nil {
				t.Fatalf("provider: kind %s", llm.Kind(err))
			}
			var (
				usage  llm.Usage
				finish llm.StopReason
			)
			for ev, err := range p.Stream(t.Context(), &llm.Request{Model: c.model, Messages: msgs, MaxTokens: lim.Tokens}) {
				if err != nil {
					status := 0
					if ae, ok := errors.AsType[*llm.APIError](err); ok {
						status = ae.Status
					}
					t.Fatalf("sent %d (%s): status %d, kind %s", lim.Tokens, lim.Basis, status, llm.Kind(err))
				}
				switch ev := ev.(type) {
				case llm.Usage:
					usage = ev
				case llm.Done:
					finish = ev.StopReason
				}
			}
			// A successful stream carries no HTTP status on this path; only a
			// failure's llm.APIError does. "ok" is what is known.
			t.Logf("sent %d (%s): ok, finish %s, %d in, %d out, %d reasoning", lim.Tokens, lim.Basis, finish,
				usage.InputTokens, usage.OutputTokens, usage.ReasoningTokens)
		})
	}
}
