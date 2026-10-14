package catalog

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// parse is ParseModelsJSON of doc, failing t on a whole-file error.
func parse(t *testing.T, doc string) (*ModelsJSON, []string) {
	t.Helper()
	mj, diags, err := ParseModelsJSON([]byte(doc), "v1.12.0")
	if err != nil {
		t.Fatal(err)
	}
	return mj, diags
}

// is reports whether p is set to v.
func is[T comparable](p *T, v T) bool { return p != nil && *p == v }

// hasDiag reports whether one of diags holds every part.
func hasDiag(diags []string, parts ...string) bool {
	return slices.ContainsFunc(diags, func(d string) bool {
		for _, p := range parts {
			if !strings.Contains(d, p) {
				return false
			}
		}
		return true
	})
}

// models.json is read as Pi reads it: comments, trailing commas and a BOM
// are accepted; a file that is not JSON, has no providers or has a field of
// the wrong type is an error; a provider gobble cannot use is skipped, and a
// field it does not use reported (0005-PLAN F4-2b).
func TestParseModelsJSON(t *testing.T) {
	mj, diags := parse(t, "\uFEFF"+`{
  // a comment
  "providers": {
    "anthropic": {"baseUrl": "https://proxy.example/v1", "headers": {"x-a": "1",},},
    "mystery": {"baseUrl": "https://x.example"},
    "openai": {"api": "openai-completions", "baseUrl": "https://y.example"},
    "grok": {},
    "kilo": {"models": [{"id": "m", "contextWindow": 0}]},
    "together": {"baseUrl": "https://t.example", "compat": {}, "models": [{"id": "m", "promptCache": {}}]},
    "gemini": {"apiKey": "$GEMINI_KEY", "baseUrl": "https://g.example"},
  },
}`)
	if got := mj.BaseURL("claude"); got != "https://proxy.example/v1" {
		t.Errorf("anthropic is claude: base %q", got)
	}
	for _, c := range []struct {
		name  string
		parts []string
	}{
		{"unknown key", []string{`"mystery"`, "go-llmprovider-sdk v1.12.0", "skipped"}},
		{"wrong api", []string{`"openai"`, `api "openai-completions"`, "skipped"}},
		{"empty entry", []string{`"grok"`, "must specify", "skipped"}},
		{"window of 0", []string{`"kilo"`, "invalid contextWindow", "skipped"}},
		{"unused fields", []string{`"together"`, "compat, promptCache not used by gobble"}},
	} {
		if !hasDiag(diags, c.parts...) {
			t.Errorf("%s: no diagnostic with %q in %q", c.name, c.parts, diags)
		}
	}
	// apiKey is used, as written, so it is not reported (0005-PLAN F4-3a).
	if hasDiag(diags, `"gemini"`) || mj.APIKey("gemini") != "$GEMINI_KEY" {
		t.Errorf("apiKey: diagnostics %q, APIKey %q; want none and the value as written", diags, mj.APIKey("gemini"))
	}
	for _, skipped := range []string{"openai", "grok", "kilo"} {
		if mj.BaseURL(skipped) != "" || mj.Models(skipped) != nil {
			t.Errorf("%s was not skipped", skipped)
		}
	}
	if mj.BaseURL("together") != "https://t.example" || mj.BaseURL("gemini") != "https://g.example" {
		t.Error("a provider with only a reported field, or apiKey, was skipped")
	}
	for name, doc := range map[string]string{
		"not JSON":     `{"providers": `,
		"no providers": `{}`,
		"wrong type":   `{"providers": {"openai": {"baseUrl": 7}}}`,
		"model id":     `{"providers": {"openai": {"models": [{"name": "x"}]}}}`,
		"cost rates":   `{"providers": {"openai": {"models": [{"id": "x", "cost": {"input": 1}}]}}}`,
	} {
		if _, _, err := ParseModelsJSON([]byte(doc), "v1.12.0"); err == nil {
			t.Errorf("%s: no whole-file error", name)
		}
	}
	if _, diags := parse(t, `{"providers": {"xai": {"baseUrl": "a"}, "grok": {"baseUrl": "b"}}}`); !hasDiag(diags, `"xai"`, `"grok" is also given`) {
		t.Errorf("an alias beside its id: %q", diags)
	}
}

// Facts follow Pi's merge: a models[] entry takes Pi's defaults for what it
// leaves out; an override sets only what it gives, a cost rate by rate, and
// replaces tiers; an override applies over a models[] entry; level maps
// merge.
func TestFacts(t *testing.T) {
	mj, _ := parse(t, `{"providers": {"openai": {
  "models": [{"id": "custom", "reasoning": true, "thinkingLevelMap": {"xhigh": "xhigh", "minimal": null}}],
  "modelOverrides": {
    "custom": {"maxTokens": 4096, "thinkingLevelMap": {"max": "max"}},
    "gpt-x": {"contextWindow": 200000, "cost": {"cacheRead": 0.5, "tiers": [{"inputTokensAbove": 1000, "input": 1, "output": 2, "cacheRead": 3, "cacheWrite": 4}]}}
  }}}}`)
	c, ok := mj.Facts("openai", "custom")
	if !ok || !is(c.Name, "custom") || !is(c.Reasoning, true) || c.Input == nil || !slices.Equal(*c.Input, []string{"text"}) ||
		!is(c.ContextWindow, 128000) || !is(c.MaxTokens, 4096) || !is(c.CostInput, 0.0) || !is(c.CostOutput, 0.0) ||
		c.Tiers == nil || len(*c.Tiers) != 0 {
		t.Errorf("custom: %+v", c)
	}
	if got := slices.Sorted(maps.Keys(c.LevelMap)); !slices.Equal(got, []string{"max", "minimal", "xhigh"}) || c.LevelMap["minimal"] != nil {
		t.Errorf("custom: level map %v, want max, minimal (off) and xhigh", c.LevelMap)
	}
	o, ok := mj.Facts("openai", "gpt-x")
	if !ok || o.Name != nil || !is(o.ContextWindow, 200000) || o.CostInput != nil || !is(o.CostCacheRead, 0.5) || o.Tiers == nil ||
		!slices.Equal(*o.Tiers, []llm.CostTier{{Above: 1000, Input: 1, Output: 2, CacheRead: 3, CacheWrite: 4}}) {
		t.Errorf("gpt-x: %+v", o)
	}
	if _, ok := mj.Facts("openai", "absent"); ok {
		t.Error("a model models.json does not name has facts")
	}
	if !slices.Equal(mj.Models("openai"), []string{"custom"}) {
		t.Errorf("Models = %q", mj.Models("openai"))
	}
}

// A model's headers are the provider's, then its override's, then its
// definition's, a later one winning.
func TestModelsJSONHeaders(t *testing.T) {
	mj, _ := parse(t, `{"providers": {"openai": {"headers": {"a": "p", "b": "p", "c": "p"},
  "modelOverrides": {"m": {"headers": {"b": "o", "c": "o"}}},
  "models": [{"id": "m", "headers": {"c": "d"}}]}}}`)
	if got := mj.Headers("openai", "m"); !maps.Equal(got, map[string]string{"a": "p", "b": "o", "c": "d"}) {
		t.Errorf("m: %v", got)
	}
	if got := mj.Headers("openai", ""); !maps.Equal(got, map[string]string{"a": "p", "b": "p", "c": "p"}) {
		t.Errorf("no model: %v", got)
	}
	if !mj.HasHeaders("openai") || mj.HasHeaders("claude") {
		t.Error("HasHeaders")
	}
}
