package provider

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// The nine providers that take a key, each with its variable and its
// default model at go-llmprovider-sdk v1.10.2 to v1.12.0, read from the SDK's
// descriptors (0019-MADR's table) and its provider.go:62-74.
var keyDefaults = []struct{ id, envVar, model string }{
	{"gemini", "GEMINI_API_KEY", "gemini-3.5-flash-lite"},
	{"openai", "OPENAI_API_KEY", "gpt-6-luna"},
	{"claude", "ANTHROPIC_API_KEY", "claude-haiku-5-5"},
	{"grok", "XAI_API_KEY", "grok-4.6"},
	{"opencode-zen", "OPENCODE_API_KEY", "claude-haiku-5-5"},
	{"opencode-go", "OPENCODE_API_KEY", "claude-haiku-5-5"},
	{"huggingface", "HF_TOKEN", "deepseek-ai/DeepSeek-V4-Flash-0731"},
	{"kilo", "KILO_API_KEY", "deepseek/deepseek-v4.1-flash"},
	{"together", "TOGETHER_API_KEY", "zai-org/GLM-5.3-Flash"},
}

// Each provider, named by defaultProvider with only its variable set, is
// chosen with the SDK's first curated model, so a bump that moves a default
// fails here, naming the provider (0019-MADR D2, D15). With no preference
// the registry order decides, so OPENCODE_API_KEY alone is opencode-zen
// (F14).
func TestAmbientDefaultModels(t *testing.T) {
	ids := make([]string, len(keyDefaults))
	for i, w := range keyDefaults {
		ids[i] = w.id
	}
	if got := KeyProviders(); !slices.Equal(got, ids) {
		t.Fatalf("KeyProviders() = %q, want %q", got, ids)
	}
	for _, w := range keyDefaults {
		s, err := Choose(envOf(map[string]string{w.envVar: "test-only"}), w.id)
		if want := (Selection{ID: w.id, Model: w.model, EnvVar: w.envVar}); err != nil || s != want {
			t.Errorf("%s: Choose = %+v, %v; want %+v", w.id, s, err, want)
		}
	}
	s, err := Choose(envOf(map[string]string{"OPENCODE_API_KEY": "test-only"}), "")
	if err != nil || s.ID != "opencode-zen" {
		t.Fatalf("no preference, OPENCODE_API_KEY alone: %+v, %v; want opencode-zen", s, err)
	}
}

// A preference whose variable is unset is the missing-credential error
// naming that variable, never another provider; a value that is not a
// provider taking a key is an error naming it (0019-MADR D15).
func TestChoosePreferred(t *testing.T) {
	s, err := Choose(envOf(map[string]string{"OPENCODE_API_KEY": "test-only"}), "openai")
	if s != (Selection{}) || !IsNoCredential(err) || !errors.Is(err, llm.ErrAuth) ||
		!strings.Contains(err.Error(), "defaultProvider is openai; set OPENAI_API_KEY") {
		t.Fatalf("openai without its key: %+v, %v", s, err)
	}
	all := map[string]string{}
	for _, v := range CredentialVars() {
		all[v] = "test-only"
	}
	for _, id := range []string{"ollama", "nope"} {
		s, err := Choose(envOf(all), id)
		if s != (Selection{}) || err == nil || IsNoCredential(err) || !strings.Contains(err.Error(), `"`+id+`"`) {
			t.Errorf("%s: %+v, %v; want an error naming it", id, s, err)
		}
	}
	if v := CredentialVar("opencode-go"); v != "OPENCODE_API_KEY" {
		t.Errorf("CredentialVar(opencode-go) = %q", v)
	}
	if v := CredentialVar("ollama"); v != "" {
		t.Errorf("CredentialVar(ollama) = %q, want none", v)
	}
}
