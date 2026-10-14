//go:build live_openai || live_claude || live_gemini || live_grok

package provider_test

import (
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// liveTurn sends one short prompt, with gobble's built-in tools and their
// strict flags, to provider id's model with the key in the environment
// variable env, skipping when it is unset, and checks for a reply, text or a
// tool call, and its usage. Errors are reported with the key redacted
// (0005-PLAN F4-1a; 0012-PLAN P7).
func liveTurn(t *testing.T, id, model, env string) {
	t.Helper()
	key := llmtest.LiveCredential(t, env)
	p, err := provider.New(id, provider.Options{APIKey: key, Model: model, ClientName: "gobble-live-test"})
	if err != nil {
		t.Fatal(liveRedact(err.Error(), key))
	}
	req := &llm.Request{
		Model:    model,
		Messages: []llm.Message{{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "Reply with the single word: ok. Do not call a tool."}}}},
		Tools:    builtinSpecs(t),
	}
	var text strings.Builder
	var call string
	var usage llm.Usage
	for ev, err := range p.Stream(t.Context(), req) {
		if err != nil {
			t.Fatal(liveRedact(err.Error(), key))
		}
		switch ev := ev.(type) {
		case llm.TextDelta:
			text.WriteString(ev.Text)
		case llm.ToolCallDone:
			call = ev.Name
		case llm.Usage:
			usage = ev
		}
	}
	if (strings.TrimSpace(text.String()) == "" && call == "") || usage.InputTokens == 0 || usage.OutputTokens == 0 {
		t.Fatalf("reply %q, call %q, with usage %+v, want text or a call, and non-zero usage", text.String(), call, usage)
	}
	t.Logf("%s %s: %q, call %q, %d in, %d out, estimated %v", id, model, text.String(), call, usage.InputTokens, usage.OutputTokens, usage.Estimated)
}

// liveRedact removes key from s.
func liveRedact(s, key string) string {
	if key == "" {
		return s
	}
	return strings.ReplaceAll(s, key, "<redacted>")
}
