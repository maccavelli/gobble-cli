package provider

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"

	"github.com/maccavelli/gobble-cli/llm"
	gcatalog "github.com/maccavelli/gobble-cli/llm/catalog"
)

// The SDK's own parser accepts the embedded snapshot, and it lists models
// for each of the nine providers it has a section for (0005-PLAN F4-2a).
func TestSnapshotParses(t *testing.T) {
	meta, err := catalog.ParseMetadata(bytes.NewReader(gcatalog.Snapshot()))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"openai", "claude", "gemini", "grok", "opencode-zen", "opencode-go", "huggingface", "kilo", "together"} {
		if len(meta.Models(llmprovider.ProviderID(id))) == 0 {
			t.Errorf("%s: no models in the snapshot", id)
		}
	}
	if err := SnapshotErr(); err != nil {
		t.Errorf("ModelInfo's snapshot did not parse: %v", err)
	}
}

// cannedDoc is a models.dev document of two OpenAI models: one with every
// field gobble reads, and one with a name alone.
const cannedDoc = `{"openai":{"id":"openai","models":{
"full":{"name":"Full","limit":{"context":400000,"output":128000},"modalities":{"input":["text","image"],"output":["text"]},
 "reasoning":true,"reasoning_options":[{"type":"effort","values":["none","low","high"]}],
 "cost":{"input":1.25,"output":10,"cache_read":0.125,"cache_write":1.5,
  "tiers":[{"tier":{"type":"context","size":272000},"input":2.5,"output":15}]}},
"bare":{"name":"Bare"}}}}`

// models.json is a layer above the snapshot: an override's facts win field
// by field, a models[] model has Pi's defaults, and its level map reaches
// the model (0005-PLAN F4-2b).
func TestModelInfoModelsJSON(t *testing.T) {
	meta, err := catalog.ParseMetadata(strings.NewReader(cannedDoc))
	if err != nil {
		t.Fatal(err)
	}
	mj, _, err := gcatalog.ParseModelsJSON([]byte(`{"providers": {"openai": {
  "modelOverrides": {"full": {"contextWindow": 999, "cost": {"input": 9}}},
  "models": [{"id": "mine", "reasoning": true, "thinkingLevelMap": {"max": "max", "minimal": null}}]}}}`), SDKVersion(), (*Set)(nil).IDs())
	if err != nil {
		t.Fatal(err)
	}
	full := lookupModelWith(t.Context(), "openai", nil, "full", "", "", "", nil, meta, modelsJSONLayer{mj})
	// The output is held to the override's 999 window (0066-MADR D4 as
	// amended; 0071-MADR P38, "Expect 999, add a wider case (Recommended)").
	if full.Name != "Full" || full.ContextWindow != 999 || full.MaxTokens != 999 || full.Cost.Input != 9 || full.Cost.Output != 10 || !full.Priced {
		t.Errorf("full: %+v, want the override's window and input price over the snapshot's, the output held to the window", full)
	}
	// wide: a window above the output lets the snapshot's 128,000 through
	// (0071-MADR P38).
	wideMJ, _, err := gcatalog.ParseModelsJSON([]byte(`{"providers": {"openai": {"modelOverrides": {"full": {"contextWindow": 999999}}}}}`),
		SDKVersion(), (*Set)(nil).IDs())
	if err != nil {
		t.Fatal(err)
	}
	if wide := lookupModelWith(t.Context(), "openai", nil, "full", "", "", "", nil, meta, modelsJSONLayer{wideMJ}); wide.ContextWindow != 999999 ||
		wide.MaxTokens != 128000 {
		t.Errorf("wide: window %d, output %d; want the override's 999,999 and the snapshot's 128,000", wide.ContextWindow, wide.MaxTokens)
	}
	mine := ModelInfo("openai", "", "", nil, "mine", mj)
	if mine.Name != "mine" || mine.ContextWindow != 128000 || mine.MaxTokens != 16384 || !mine.Priced || mine.Cost.Input != 0 ||
		!mine.ReasoningKnown || !mine.Reasoning || !mine.LevelMap["max"] || mine.LevelMap["minimal"] {
		t.Errorf("mine: %+v, want Pi's defaults and the level map", mine)
	}
	if plain := ModelInfo("openai", "", "", nil, "mine", nil); plain.ReasoningKnown || plain.LevelMap != nil {
		t.Errorf("without models.json: %+v, want nothing known", plain)
	}
}

// A model's output cap carries its source (0071-MADR D1, L5, L6, L7): a
// model models.json defines without maxTokens takes Pi's 16,384, assumed; a
// maxTokens it states, an entry's or an override's, is the user's; the
// snapshot's cap is the catalog's; a model nothing knows has no cap and no
// source, for llm.FitOutput to assume.
func TestModelsJSONDefaultIsAssumed_0071(t *testing.T) {
	mj, _, err := gcatalog.ParseModelsJSON([]byte(`{"providers": {"openai": {
  "modelOverrides": {"over": {"maxTokens": 100000}, "gpt-4o": {"contextWindow": 200000}},
  "models": [{"id": "mine"}, {"id": "set", "maxTokens": 2048}]}}}`), SDKVersion(), (*Set)(nil).IDs())
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		model string
		mj    *gcatalog.ModelsJSON
		cap   int
		from  llm.CapSource
	}{
		{"mine", mj, 16384, llm.CapAssumed}, // Q2's 16,384, not the constant, so the constant is held too
		{"set", mj, 2048, llm.CapUser},
		{"over", mj, 100000, llm.CapUser},
		{"gpt-4o", mj, 16384, llm.CapCatalog}, // an override without maxTokens leaves the snapshot's
		{"gpt-4o", nil, 16384, llm.CapCatalog},
		{"no-such-model", nil, 0, ""},
	} {
		if m := ModelInfo("openai", "", "", nil, c.model, c.mj); m.MaxTokens != c.cap || m.MaxTokensFrom != c.from {
			t.Errorf("%s (models.json %v): cap %d from %q, want %d from %q", c.model, c.mj != nil, m.MaxTokens, m.MaxTokensFrom, c.cap, c.from)
		}
	}
	// The six GLM-5.3 models of 0071-MADR's table, from the snapshot alone.
	for _, c := range []struct {
		provider, model string
		window, cap     int
	}{
		{"kilo", "z-ai/glm-5.3", 1048576, 943718}, {"kilo", "z-ai/glm-5.3-flash", 1048576, 943718},
		{"opencode-go", "glm-5.3", 1000000, 131072}, {"opencode-go", "glm-5.3-flash", 1000000, 131072},
		{"together", "zai-org/GLM-5.3", 1048576, 262144}, {"together", "zai-org/GLM-5.3-Flash", 1048575, 400000},
	} {
		if m := ModelInfo(c.provider, "", "", nil, c.model, nil); m.ContextWindow != c.window || m.MaxTokens != c.cap || m.MaxTokensFrom != llm.CapCatalog {
			t.Errorf("%s/%s: window %d, cap %d from %q; want %d, %d from the catalog", c.provider, c.model, m.ContextWindow, m.MaxTokens,
				m.MaxTokensFrom, c.window, c.cap)
		}
	}
}

// The known models are the curated ones, then the document's that take
// tools, sorted, each once; one without tools is left out (0005-PLAN F4-2c).
func TestKnownModels(t *testing.T) {
	curated := Models("openai")
	if len(curated) == 0 {
		t.Fatal("openai has no curated models")
	}
	doc := `{"openai":{"id":"openai","models":{
"zeta-tool":{"name":"Z","tool_call":true},"alpha-tool":{"name":"A","tool_call":true},
"no-tools":{"name":"N","tool_call":false},"` + curated[0] + `":{"name":"C","tool_call":true}}}}`
	meta, err := catalog.ParseMetadata(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	want := append(slices.Clone(curated), "alpha-tool", "zeta-tool")
	if got := toolModels("openai", &meta); !slices.Equal(got, want) {
		t.Errorf("known models %q, want %q", got, want)
	}
	if got := toolModels("openai", nil); !slices.Equal(got, curated) {
		t.Errorf("with no document %q, want the curated %q", got, curated)
	}
}

// The wire table names every built-in the SDK registers, so each may name an api.
func TestModelsJSONIDs(t *testing.T) {
	var ids []string
	for _, d := range providers.Default().Descriptors() {
		ids = append(ids, string(d.ID))
	}
	slices.Sort(ids)
	if got := gcatalog.ProviderIDs(); !slices.Equal(got, ids) {
		t.Errorf("llm/catalog's ids %q, the SDK's %q", got, ids)
	}
}

// ModelInfo's fields are the document's where it gives them, unknown where
// it does not, and a model it lacks has its provider and id alone.
func TestModelInfo(t *testing.T) {
	meta, err := catalog.ParseMetadata(strings.NewReader(cannedDoc))
	if err != nil {
		t.Fatal(err)
	}
	m := lookupModelWith(t.Context(), "openai", nil, "full", "", "", "", nil, meta)
	if m.Name != "Full" || m.ContextWindow != 400000 || m.MaxTokens != 128000 || !slices.Equal(m.Input, []string{"text", "image"}) ||
		!m.ReasoningKnown || !m.Reasoning || !slices.Equal(m.Efforts, []string{"none", "low", "high"}) {
		t.Errorf("full: %+v", m)
	}
	want := llm.ModelCost{Input: 1.25, Output: 10, CacheRead: 0.125, CacheWrite: 1.5,
		Tiers: []llm.CostTier{{Above: 272000, Input: 2.5, Output: 15, CacheRead: 0.125, CacheWrite: 1.5}}}
	if !m.Priced || m.Cost.Input != want.Input || m.Cost.Output != want.Output || m.Cost.CacheRead != want.CacheRead ||
		m.Cost.CacheWrite != want.CacheWrite || !slices.Equal(m.Cost.Tiers, want.Tiers) {
		t.Errorf("full: cost %+v priced %v, want %+v", m.Cost, m.Priced, want)
	}
	if b := lookupModelWith(t.Context(), "openai", nil, "bare", "", "", "", nil, meta); b.Name != "Bare" || b.ContextWindow != 0 || b.Priced || b.ReasoningKnown {
		t.Errorf("bare: %+v, want its name alone", b)
	}
	if u := lookupModelWith(t.Context(), "openai", nil, "absent", "", "", "", nil, meta); u.Provider != "openai" || u.ID != "absent" || u.Name != "" || u.Priced {
		t.Errorf("absent: %+v, want its provider and id alone", u)
	}
}
