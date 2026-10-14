package catalog

import (
	"errors"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// resolveModels are the models TestResolveModel resolves over.
var resolveModels = []llm.Model{
	{Provider: "openai", ID: "gpt-6-luna", Name: "GPT-6 Luna"},
	{Provider: "openai", ID: "gpt-6-astra", Name: "GPT-6 Astra"},
	{Provider: "openai", ID: "gpt-6-nova"},
	{Provider: "openai", ID: "shared-1"},
	{Provider: "claude", ID: "claude-sonnet-9", Name: "Claude Sonnet 9"},
	{Provider: "claude", ID: "claude-sonnet-9-20261001"},
	{Provider: "claude", ID: "claude-opus-9-20260901"},
	{Provider: "claude", ID: "claude-opus-9-20261001"},
	{Provider: "claude", ID: "shared-1"},
	{Provider: "kilo", ID: "openai/gpt-6-nova"},
	{Provider: "kilo", ID: "openai/o-special"},
	{Provider: "kilo", ID: "shared-1"},
}

// --provider and --model resolve by each of Pi's resolveCliModel steps, in
// its words (0005-PLAN F4-2c).
func TestResolveModel(t *testing.T) {
	cases := []struct {
		name, provider, model, thinking string
		authed                          []string
		want                            string // provider/id
		level, warning, err             string
		custom                          bool
	}{
		{name: "exact provider/id without case", model: "OpenAI/GPT-6-Luna", want: "openai/gpt-6-luna"},
		{name: "a bare id one provider holds", model: "gpt-6-astra", want: "openai/gpt-6-astra"},
		{name: "a shared id, one credentialed", model: "shared-1", authed: []string{"claude"}, want: "claude/shared-1"},
		{name: "a shared id, none credentialed", model: "shared-1",
			err: `Model "shared-1" is ambiguous across providers: claude/shared-1, kilo/shared-1, openai/shared-1. No matching provider is authenticated. Use --provider or provider/model.`},
		{name: "a shared id, two credentialed", model: "shared-1", authed: []string{"claude", "kilo"},
			err: `Model "shared-1" is ambiguous across providers: claude/shared-1, kilo/shared-1, openai/shared-1. More than one matching provider is authenticated. Use --provider or provider/model.`},
		{name: "an unknown provider", provider: "nope", model: "x", err: `Unknown provider "nope". Use --list-models to see available providers/models.`},
		{name: "--provider with its prefix on the model", provider: "openai", model: "openai/gpt-6-luna", want: "openai/gpt-6-luna"},
		{name: "--provider by Pi's name", provider: "Anthropic", model: "anthropic/claude-sonnet-9", want: "claude/claude-sonnet-9"},
		{name: "a substring of an id", model: "astra", want: "openai/gpt-6-astra"},
		{name: "a substring of a name", model: "Sonnet 9", want: "claude/claude-sonnet-9"},
		{name: "an alias before a dated id", model: "sonnet-9", want: "claude/claude-sonnet-9"},
		{name: "the latest dated id", model: "opus-9", want: "claude/claude-opus-9-20261001"},
		{name: "a level suffix", model: "luna:high", want: "openai/gpt-6-luna", level: "high"},
		{name: "an unknown suffix", model: "luna:turbo", err: `Model "luna:turbo" not found. Use --list-models to see available models.`},
		{name: "Pi's name as the prefix", model: "anthropic/claude-sonnet-9", want: "claude/claude-sonnet-9"},
		{name: "a taken provider gives way to a credentialed raw id", model: "openai/gpt-6-nova", authed: []string{"kilo"},
			want: "kilo/openai/gpt-6-nova"},
		{name: "a taken provider with its own credential keeps it", model: "openai/gpt-6-nova", authed: []string{"openai", "kilo"},
			want: "openai/gpt-6-nova"},
		{name: "the whole input across providers", model: "openai/o-special", want: "kilo/openai/o-special"},
		{name: "a custom id", model: "openai/gpt-7", want: "openai/gpt-7", custom: true,
			warning: `Model "gpt-7" not found for provider "openai". Using custom model id.`},
		{name: "a custom id's level split off", model: "openai/gpt-7:high", want: "openai/gpt-7", level: "high", custom: true,
			warning: `Model "gpt-7" not found for provider "openai". Using custom model id.`},
		{name: "a custom id keeps its suffix with --thinking", model: "openai/gpt-7:high", thinking: "low", want: "openai/gpt-7:high", custom: true,
			warning: `Model "gpt-7:high" not found for provider "openai". Using custom model id.`},
		{name: "not found", model: "zzz", err: `Model "zzz" not found. Use --list-models to see available models.`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			authed := func(p string) bool { return slices.Contains(c.authed, p) }
			r, err := ResolveModel(c.provider, c.model, c.thinking, resolveModels, authed)
			if c.err != "" {
				var re *ResolutionError
				if !errors.As(err, &re) || err.Error() != c.err {
					t.Fatalf("error %v, want %q", err, c.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := r.Model.Provider + "/" + r.Model.ID; got != c.want || r.Level != c.level || r.Custom != c.custom || r.Warning != c.warning {
				t.Errorf("got %s level %q custom %v warning %q; want %s level %q custom %v warning %q",
					got, r.Level, r.Custom, r.Warning, c.want, c.level, c.custom, c.warning)
			}
		})
	}
}

// No --model is no choice, and no models at all is Pi's error.
func TestResolveModelEmpty(t *testing.T) {
	none := func(string) bool { return false }
	if r, err := ResolveModel("openai", "", "", resolveModels, none); err != nil || r.Model.Provider != "" || r.Model.ID != "" || r.Level != "" || r.Custom {
		t.Errorf("no --model: %+v, %v; want no choice", r, err)
	}
	want := "No models available. Check your installation or add models to models.json."
	if _, err := ResolveModel("", "x", "", nil, none); err == nil || err.Error() != want {
		t.Errorf("no models: %v, want %q", err, want)
	}
}

// A provider's name is its SDK id or Pi's name for it, without case.
func TestProviderID(t *testing.T) {
	for name, want := range map[string]string{"Anthropic": "claude", "google": "gemini", "XAI": "grok", "openai": "openai", " kilo ": "kilo"} {
		if got, ok := ProviderID(name); !ok || got != want {
			t.Errorf("ProviderID(%q) = %q, %v; want %q", name, got, ok, want)
		}
	}
	if got, ok := ProviderID("nope"); ok {
		t.Errorf("ProviderID(nope) = %q, want none", got)
	}
}
