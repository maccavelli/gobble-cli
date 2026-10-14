package provider

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

	"github.com/maccavelli/gobble-cli/llm"
	gcatalog "github.com/maccavelli/gobble-cli/llm/catalog"
)

// snapshot is the embedded models.dev document, parsed once. A document the
// SDK refuses leaves every model unknown, and SnapshotErr reports it.
var snapshot = sync.OnceValues(func() (*catalog.Metadata, error) {
	meta, err := catalog.ParseMetadata(bytes.NewReader(gcatalog.Snapshot()))
	if err != nil {
		return nil, fmt.Errorf("provider: the embedded model catalog cannot be read, so models are unknown: %w", err)
	}
	return &meta, nil
})

// SnapshotErr is why the embedded catalog could not be read, or nil. The
// caller logs it: models are then unknown, never refused (0005-PLAN F4-2a
// deviation 1).
func SnapshotErr() error {
	_, err := snapshot()
	return err
}

// ModelInfo is what the catalog knows of model on provider id, from the
// embedded snapshot and, with apiKey, the provider's listing as the SDK has
// it cached. It never waits and makes no request of its own; a listing not
// yet cached is fetched in the background for a later call (0005-PLAN F4-2a).
// A model nothing knows has only its provider and id.
func ModelInfo(id, apiKey, model string) llm.Model {
	meta, err := snapshot()
	if err != nil {
		return llm.Model{Provider: id, ID: model}
	}
	var src llmprovider.TokenSource
	if apiKey != "" {
		src = llmprovider.NewStaticToken(apiKey)
	}
	return lookupModel(*meta, id, src, model)
}

// lookupModel is ModelInfo over the document meta.
func lookupModel(meta catalog.Metadata, id string, src llmprovider.TokenSource, model string) llm.Model {
	info, _ := catalog.CachedModel(context.Background(), llmprovider.ProviderID(id), src, model,
		llmprovider.WithoutModelMetadata(), catalog.WithModelLayer(meta))
	return modelOf(id, model, info)
}

// modelOf is info as llm's Model: each field only where a layer set it.
func modelOf(id, model string, info catalog.ModelInfo) llm.Model {
	m := llm.Model{Provider: id, ID: model}
	if info.Has(catalog.FieldName) {
		m.Name = info.Name
	}
	if info.Has(catalog.FieldContextWindow) {
		m.ContextWindow = info.ContextWindow
	}
	if info.Has(catalog.FieldMaxOutputTokens) {
		m.MaxTokens = info.MaxOutputTokens
	}
	if info.Has(catalog.FieldInput) {
		for _, in := range info.Input {
			m.Input = append(m.Input, string(in))
		}
	}
	if info.Has(catalog.FieldReasoning) {
		m.Reasoning, m.ReasoningKnown = info.Reasoning.Supported, true
	}
	if info.Has(catalog.FieldReasoningEfforts) {
		for _, e := range info.Reasoning.Efforts {
			m.Efforts = append(m.Efforts, string(e))
		}
	}
	if info.Has(catalog.FieldCostInput) && info.Has(catalog.FieldCostOutput) {
		c := info.Cost
		m.Priced = true
		m.Cost = llm.ModelCost{Input: c.Input, Output: c.Output, CacheRead: c.CacheRead, CacheWrite: c.CacheWrite}
		for _, t := range c.Tiers {
			m.Cost.Tiers = append(m.Cost.Tiers, llm.CostTier{Above: t.Above, Input: t.Input, Output: t.Output,
				CacheRead: t.CacheRead, CacheWrite: t.CacheWrite})
		}
	}
	return m
}
