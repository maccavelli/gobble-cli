package provider

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

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

// ModelInfo's fields are the document's where it gives them, unknown where
// it does not, and a model it lacks has its provider and id alone.
func TestModelInfo(t *testing.T) {
	meta, err := catalog.ParseMetadata(strings.NewReader(cannedDoc))
	if err != nil {
		t.Fatal(err)
	}
	m := lookupModel(meta, "openai", nil, "full")
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
	if b := lookupModel(meta, "openai", nil, "bare"); b.Name != "Bare" || b.ContextWindow != 0 || b.Priced || b.ReasoningKnown {
		t.Errorf("bare: %+v, want its name alone", b)
	}
	if u := lookupModel(meta, "openai", nil, "absent"); u.Provider != "openai" || u.ID != "absent" || u.Name != "" || u.Priced {
		t.Errorf("absent: %+v, want its provider and id alone", u)
	}
}
