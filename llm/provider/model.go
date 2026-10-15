package provider

import (
	"bytes"
	"context"
	"fmt"
	"runtime/debug"
	"slices"
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

// sdkPath is go-llmprovider-sdk's module path.
const sdkPath = "github.com/maccavelli/go-llmprovider-sdk"

// SDKVersion is the go-llmprovider-sdk version this binary was built with,
// as its build information records it, or "(unknown)".
func SDKVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		for _, d := range bi.Deps {
			if d.Path == sdkPath {
				if d.Replace != nil {
					return d.Replace.Version
				}
				return d.Version
			}
		}
	}
	return "(unknown)"
}

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
// A model nothing knows has only its provider and id. mj, when not nil, is
// models.json: a layer above the snapshot, and its thinkingLevelMap
// (0005-PLAN F4-2b). With no apiKey, a provider with an OAuth login uses
// logins' stored login, when logins is not nil (0005-PLAN F4-3b).
func ModelInfo(id, apiKey string, logins LoginStore, model string, mj *gcatalog.ModelsJSON) llm.Model {
	ctx := context.Background()
	login, err := Options{Logins: logins}.login(ctx, id)
	if err == nil && login != nil && apiKey == "" {
		return modelInfoFrom(ctx, id, login, model, mj)
	}
	return ModelInfoContext(ctx, id, apiKey, model, mj)
}

// ModelInfoContext is ModelInfo for a caller with a context, which bounds
// the SDK's lookup (0005-PLAN F4-2c deviation 1), and no login.
func ModelInfoContext(ctx context.Context, id, apiKey, model string, mj *gcatalog.ModelsJSON) llm.Model {
	var src llmprovider.TokenSource
	if apiKey != "" {
		src = llmprovider.NewStaticToken(apiKey)
	}
	return modelInfoFrom(ctx, id, src, model, mj)
}

// modelInfoFrom is ModelInfoContext with src, the credential's token source, or
// nil for none.
func modelInfoFrom(ctx context.Context, id string, src llmprovider.TokenSource, model string, mj *gcatalog.ModelsJSON) llm.Model {
	var layers []catalog.Layer
	if meta, err := snapshot(); err == nil {
		layers = append(layers, *meta)
	}
	if mj == nil {
		return lookupModel(ctx, id, src, model, layers...)
	}
	m := lookupModel(ctx, id, src, model, append(layers, modelsJSONLayer{mj})...)
	if f, ok := mj.Facts(id, model); ok && len(f.LevelMap) > 0 {
		m.LevelMap = map[string]bool{}
		for level, v := range f.LevelMap {
			m.LevelMap[level] = v != nil
		}
	}
	return m
}

// KnownModels are the models gobble knows of provider id with no request:
// its curated models, then the snapshot's models for it that take tools, by
// id, each once (0005-PLAN F4-2c), as Pi's generator keeps only those.
func KnownModels(id string) []string {
	meta, err := snapshot()
	if err != nil { // the curated models alone; SnapshotErr reports it
		return Models(id)
	}
	return toolModels(id, meta)
}

// toolModels is KnownModels over meta; nil holds no models.
func toolModels(id string, meta *catalog.Metadata) []string {
	ids := Models(id)
	if meta == nil {
		return ids
	}
	for _, m := range meta.Models(llmprovider.ProviderID(id)) {
		if m.Has(catalog.FieldTools) && m.Tools && !slices.Contains(ids, m.ID) {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

// lookupModel is ModelInfo over layers, the later ranking above the earlier,
// bounded by ctx.
func lookupModel(ctx context.Context, id string, src llmprovider.TokenSource, model string, layers ...catalog.Layer) llm.Model {
	opts := []llmprovider.Option{llmprovider.WithoutModelMetadata()}
	for _, l := range layers {
		opts = append(opts, catalog.WithModelLayer(l))
	}
	info, _ := catalog.CachedModel(ctx, llmprovider.ProviderID(id), src, model, opts...)
	return modelOf(id, model, info)
}

// modelsJSONLayer is models.json as an SDK layer: each fact it gives is
// marked known, so it outranks the snapshot's field by field.
type modelsJSONLayer struct{ mj *gcatalog.ModelsJSON }

func (modelsJSONLayer) Name() string { return "models.json" }

func (l modelsJSONLayer) Model(p llmprovider.ProviderID, id string) (catalog.ModelInfo, bool) {
	f, ok := l.mj.Facts(string(p), id)
	if !ok {
		return catalog.ModelInfo{}, false
	}
	var info catalog.ModelInfo
	var known []catalog.Field
	if f.Name != nil {
		info.Name, known = *f.Name, append(known, catalog.FieldName)
	}
	if f.Reasoning != nil {
		info.Reasoning.Supported, known = *f.Reasoning, append(known, catalog.FieldReasoning)
	}
	if f.Input != nil {
		for _, in := range *f.Input {
			info.Input = append(info.Input, catalog.Modality(in))
		}
		known = append(known, catalog.FieldInput)
	}
	if f.ContextWindow != nil {
		info.ContextWindow, known = *f.ContextWindow, append(known, catalog.FieldContextWindow)
	}
	if f.MaxTokens != nil {
		info.MaxOutputTokens, known = *f.MaxTokens, append(known, catalog.FieldMaxOutputTokens)
	}
	for _, r := range []struct {
		v     *float64
		dst   *float64
		field catalog.Field
	}{
		{f.CostInput, &info.Cost.Input, catalog.FieldCostInput}, {f.CostOutput, &info.Cost.Output, catalog.FieldCostOutput},
		{f.CostCacheRead, &info.Cost.CacheRead, catalog.FieldCostCacheRead},
		{f.CostCacheWrite, &info.Cost.CacheWrite, catalog.FieldCostCacheWrite},
	} {
		if r.v != nil {
			*r.dst, known = *r.v, append(known, r.field)
		}
	}
	if f.Tiers != nil {
		for _, t := range *f.Tiers {
			info.Cost.Tiers = append(info.Cost.Tiers, catalog.ModelCostTier{Above: t.Above,
				Input: t.Input, Output: t.Output, CacheRead: t.CacheRead, CacheWrite: t.CacheWrite})
		}
		known = append(known, catalog.FieldCostTiers)
	}
	return info.Known(known...), true
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
