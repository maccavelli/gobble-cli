package acpext_test

import (
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/acpext"
)

// AuthProviders is the list gobble's internal/cli builds from the provider
// registry, as recorded at 0042-PLAN P1 step 1; gobble's own test keeps the
// two equal.
func TestAuthProviders(t *testing.T) {
	want := []acpext.AuthProvider{
		{ID: "gemini", Name: "Gemini (Google)", Command: "gobble auth login gemini --api-key -"},
		{ID: "openai", Name: "OpenAI", Command: "gobble auth login openai --api-key -"},
		{ID: "claude", Name: "Claude (Anthropic)", Command: "gobble auth login claude --api-key -"},
		{ID: "grok", Name: "Grok (xAI)", Command: "gobble auth login grok", Login: true},
		{ID: "opencode-zen", Name: "OpenCode Zen", Command: "gobble auth login opencode-zen --api-key -"},
		{ID: "opencode-go", Name: "OpenCode Go", Command: "gobble auth login opencode-go --api-key -"},
		{ID: "huggingface", Name: "Hugging Face", Command: "gobble auth login huggingface --api-key -"},
		{ID: "kilo", Name: "Kilo Gateway", Command: "gobble auth login kilo", Login: true},
		{ID: "together", Name: "Together AI", Command: "gobble auth login together --api-key -"},
	}
	got := acpext.AuthProviders()
	if !slices.Equal(got, want) {
		t.Fatalf("AuthProviders() = %+v,\nwant %+v", got, want)
	}
	got[0].ID = "changed"
	if acpext.AuthProviders()[0].ID == "changed" {
		t.Error("AuthProviders() returned a shared slice")
	}
}
