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
  "models": [{"id": "mine", "reasoning": true, "thinkingLevelMap": {"max": "max", "minimal": null}}]}}}`), SDKVersion())
	if err != nil {
		t.Fatal(err)
	}
	full := lookupModel(t.Context(), "openai", nil, "full", meta, modelsJSONLayer{mj})
	if full.Name != "Full" || full.ContextWindow != 999 || full.MaxTokens != 128000 || full.Cost.Input != 9 || full.Cost.Output != 10 || !full.Priced {
		t.Errorf("full: %+v, want the override's window and input price over the snapshot's", full)
	}
	mine := ModelInfo("openai", "", "mine", mj)
	if mine.Name != "mine" || mine.ContextWindow != 128000 || mine.MaxTokens != 16384 || !mine.Priced || mine.Cost.Input != 0 ||
		!mine.ReasoningKnown || !mine.Reasoning || !mine.LevelMap["max"] || mine.LevelMap["minimal"] {
		t.Errorf("mine: %+v, want Pi's defaults and the level map", mine)
	}
	if plain := ModelInfo("openai", "", "mine", nil); plain.ReasoningKnown || plain.LevelMap != nil {
		t.Errorf("without models.json: %+v, want nothing known", plain)
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

// The providers models.json may name are the SDK's built-in ids.
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
	m := lookupModel(t.Context(), "openai", nil, "full", meta)
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
	if b := lookupModel(t.Context(), "openai", nil, "bare", meta); b.Name != "Bare" || b.ContextWindow != 0 || b.Priced || b.ReasoningKnown {
		t.Errorf("bare: %+v, want its name alone", b)
	}
	if u := lookupModel(t.Context(), "openai", nil, "absent", meta); u.Provider != "openai" || u.ID != "absent" || u.Name != "" || u.Priced {
		t.Errorf("absent: %+v, want its provider and id alone", u)
	}
}
