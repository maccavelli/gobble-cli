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
// it cached. It waits for no listing: one not yet cached is fetched in the
// background for a later call (0005-PLAN F4-2a). With a login, the SDK first
// asks it for a token, which may refresh at the issuer before ModelInfo
// returns (go-llmprovider-sdk catalog.CachedModel; 0046-MADR).
// A model nothing knows has only its provider and id. mj, when not nil, is
// models.json: a layer above the snapshot, and its thinkingLevelMap
// (0005-PLAN F4-2b). With no apiKey, a provider with an OAuth login uses
// logins' stored login, when logins is not nil (0005-PLAN F4-3b). workspace
// is a Claude key's, for its listing (0035-MADR D5).
func ModelInfo(id, apiKey, workspace string, logins LoginStore, model string, mj *gcatalog.ModelsJSON) llm.Model {
	return ModelInfoWith(Options{APIKey: apiKey, Workspace: workspace, Logins: logins}, id, model, mj)
}

// ModelInfoWith is ModelInfo with o's key, workspace, logins and sessions,
// and o's transport and cache: a listing it starts in the background runs
// through o.Cache when set, so the run that owns the cache ends it, and a
// login's refresh is on the run's session (0046-MADR D7, D8). With a
// ClientName, the lookup names that application and ClientVersion to the SDK,
// which names a login's session with them before it reads its token, so a
// refresh the lookup starts, and the listing it starts in the background,
// name it (0051-MADR D9, N6). o's other fields are ignored.
func ModelInfoWith(o Options, id, model string, mj *gcatalog.ModelsJSON) llm.Model {
	ctx := context.Background()
	src, workspace, sdk := lookupSource(ctx, o, id)
	return modelInfoFrom(ctx, id, src, workspace, model, mj, o.ClientName, o.ClientVersion, sdk...)
}

// lookupSource is what a lookup of provider id under o reads with: the
// credential's token source, the stored login's when o has no key and one
// loads, else o's key's, else nil; the Claude workspace, a key's alone; and
// the SDK options of o's transport, base URL, cache and set.
func lookupSource(ctx context.Context, o Options, id string) (llmprovider.TokenSource, string, []llmprovider.Option) {
	sdk := Options{HTTPClient: o.HTTPClient, Headers: o.Headers, BaseURL: o.BaseURL, Cache: o.Cache, Set: o.Set}.sdkOptions()
	sdk = append(sdk, Options{BaseURL: o.BaseURL, Set: o.Set}.chatOptions(id)...)
	login, err := Options{Logins: o.Logins, Sessions: o.Sessions}.login(ctx, id)
	if err == nil && login != nil && o.APIKey == "" {
		return login, "", sdk
	}
	if o.APIKey != "" {
		return llmprovider.NewStaticToken(o.APIKey), o.Workspace, sdk
	}
	return nil, o.Workspace, sdk
}

// limitsSource is the token source a limits read of provider id uses: src;
// for a provider that takes no key, such as Ollama, with none, the empty
// static token its provider sends, so that discovery reads the context its
// server serves and a lookup finds it (0066-MADR S31).
func limitsSource(id string, src llmprovider.TokenSource) llmprovider.TokenSource {
	if src == nil && CredentialVar(id) == "" {
		return llmprovider.NewStaticToken("")
	}
	return src
}

// Discover reads every source of model's limits on provider id, as the SDK's
// catalog.LookupLimits does, so that a lookup after it, and the provider's
// own requests, read what the service states (0066-MADR D1, as amended
// 2026-11-25). It reads with what ModelInfoWith reads with: o's credential,
// transport, base URL, cache and set, the Claude workspace, gobble's name and
// version, the embedded snapshot as the "models.dev" layer and models.json
// above it. It passes no Kilo organization (0066-MADR S43). The SDK bounds
// it at 10 seconds; ctx ends it sooner. Its error is each source's failure,
// joined; what the other sources gave is kept either way, and a lookup reads
// it.
func Discover(ctx context.Context, o Options, id, model string, mj *gcatalog.ModelsJSON) error {
	src, workspace, sdk := lookupSource(ctx, o, id)
	opts := lookupOptions(workspace, o.ClientName, o.ClientVersion, sdk, lookupLayers(mj)...)
	_, err := catalog.LookupLimits(ctx, llmprovider.ProviderID(id), limitsSource(id, src), model, opts...)
	return err
}

// lookupLayers are a lookup's layers: the embedded snapshot, when it
// parsed, then models.json, when mj is not nil, which ranks above it.
func lookupLayers(mj *gcatalog.ModelsJSON) []catalog.Layer {
	var layers []catalog.Layer
	if meta, err := snapshot(); err == nil {
		layers = append(layers, *meta)
	}
	if mj != nil {
		layers = append(layers, modelsJSONLayer{mj})
	}
	return layers
}

// ModelInfoContext is ModelInfo for a caller with a context, which bounds
// the SDK's lookup (0005-PLAN F4-2c deviation 1), and no login.
func ModelInfoContext(ctx context.Context, id, apiKey, model string, mj *gcatalog.ModelsJSON) llm.Model {
	var src llmprovider.TokenSource
	if apiKey != "" {
		src = llmprovider.NewStaticToken(apiKey)
	}
	return modelInfoFrom(ctx, id, src, "", model, mj, "", "")
}

// modelInfoFrom is ModelInfoContext with src, the credential's token source, or
// nil for none, a Claude key's workspace, the application's name and version
// (clientName empty names none), and sdk, the SDK options a listing it starts
// takes (0046-MADR D7).
func modelInfoFrom(ctx context.Context, id string, src llmprovider.TokenSource, workspace, model string, mj *gcatalog.ModelsJSON, clientName, clientVersion string, sdk ...llmprovider.Option) llm.Model {
	m := lookupModelWith(ctx, id, src, model, workspace, clientName, clientVersion, sdk, lookupLayers(mj)...)
	if mj == nil {
		return capSourced(m, gcatalog.Facts{})
	}
	f, ok := mj.Facts(id, model)
	if ok && len(f.LevelMap) > 0 {
		m.LevelMap = map[string]bool{}
		for level, v := range f.LevelMap {
			m.LevelMap[level] = v != nil
		}
	}
	return capSourced(m, f)
}

// The sources of a limit, as the SDK names its layers (0071-MADR P17, P28):
// models.json's layer, and the layer of the caps services stated on earlier
// runs.
const (
	sourceModelsJSON    = "models.json"
	sourceServiceStated = "service-stated"
)

// capSourced is m with the source of its streamed output cap, the cap a turn
// asks for (0071-MADR D1, P17, P28): models.json's maxTokens, when it states
// one and its value is chosen or a service's stated cap held it, is the
// user's; Pi's default for a model models.json defines without one is
// assumed; any other known cap is the catalog's. An unknown cap is left
// unsourced, for llm.FitOutput to assume. A models.json maxTokens the SDK
// chose over a cap a service stated on an earlier run, ranked below it, is
// first held to that cap, as a cap a service stated in this process already
// holds it.
func capSourced(m llm.Model, f gcatalog.Facts) llm.Model {
	stated := f.MaxTokens != nil && f.MaxTokensStated
	if stated {
		m.Limits.Output = heldToServiceStated(m.Limits.Output)
		m.Limits.StreamedOutput = heldToServiceStated(m.Limits.StreamedOutput)
		m.MaxTokens = m.Limits.Output.Tokens
	}
	out := m.Limits.StreamedOutput
	switch {
	case !out.Known:
		m.MaxTokensFrom = ""
	case stated && (out.Source == sourceModelsJSON || statedAbove(out, sourceModelsJSON)):
		m.MaxTokensFrom = llm.CapUser
	case f.MaxTokens != nil && out.Source == sourceModelsJSON:
		m.MaxTokensFrom = llm.CapAssumed
	default:
		m.MaxTokensFrom = llm.CapCatalog
	}
	return m
}

// statedAbove reports whether source states more than f's chosen value.
func statedAbove(f llm.LimitFact, source string) bool {
	return slices.ContainsFunc(f.Others, func(o llm.LimitFact) bool { return o.Source == source && o.Tokens > f.Tokens })
}

// heldToServiceStated is f, chosen from models.json, held to the smallest cap
// the "service-stated" layer states below it, which then leads its evidence
// (0071-MADR P28).
func heldToServiceStated(f llm.LimitFact) llm.LimitFact {
	if !f.Known || f.Source != sourceModelsJSON {
		return f
	}
	best := -1
	for i, o := range f.Others {
		if o.Source == sourceServiceStated && o.Tokens > 0 && o.Tokens < f.Tokens && (best < 0 || o.Tokens < f.Others[best].Tokens) {
			best = i
		}
	}
	if best < 0 {
		return f
	}
	chosen := f.Others[best]
	others := append([]llm.LimitFact{{Tokens: f.Tokens, Known: true, Source: f.Source, Stale: f.Stale}},
		slices.Delete(slices.Clone(f.Others), best, best+1)...)
	return llm.LimitFact{Tokens: chosen.Tokens, Known: true, Conflicting: true, Source: chosen.Source, Stale: chosen.Stale, Others: others}
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

// lookupModelWith is ModelInfo over layers, the later ranking above the
// earlier, bounded by ctx; a listing it starts runs in workspace for Claude,
// with sdk, its SDK options: the transport and the run's cache (0046-MADR
// D7). When clientName is set, the call carries llmprovider.WithClientInfo
// with clientName and clientVersion, which the SDK's CachedModel hands to a
// session src as its identity before it reads the token, so a refresh the
// lookup starts names the application (0051-MADR D9, N6). gobble calls no
// UseIdentity itself.
func lookupModelWith(ctx context.Context, id string, src llmprovider.TokenSource, model, workspace, clientName, clientVersion string, sdk []llmprovider.Option, layers ...catalog.Layer) llm.Model {
	opts := lookupOptions(workspace, clientName, clientVersion, sdk, layers...)
	info, _ := catalog.CachedModel(ctx, llmprovider.ProviderID(id), src, model, opts...)
	// The limits read the cached listing by the same credential and client
	// info, so they find what the lookup's listing stored (0071-MADR P33).
	if ls := limitsSource(id, src); ls != nil {
		opts = append(opts, llmprovider.WithTokenSource(ls))
	}
	return withLimits(modelOf(id, model, info), catalog.CachedLimits(ctx, llmprovider.ProviderID(id), model, opts...))
}

// lookupOptions are a lookup's SDK options: sdk, then gobble's name and
// version when clientName is set, a Claude key's workspace, and layers, the
// later ranking above the earlier.
func lookupOptions(workspace, clientName, clientVersion string, sdk []llmprovider.Option, layers ...catalog.Layer) []llmprovider.Option {
	opts := slices.Clone(sdk)
	if clientName != "" {
		opts = append(opts, llmprovider.WithClientInfo(clientName, clientVersion))
	}
	if workspace != "" {
		opts = append(opts, llmprovider.For(llmprovider.ProviderClaude, catalog.WithClaudeWorkspace(workspace)))
	}
	for _, l := range layers {
		opts = append(opts, catalog.WithModelLayer(l))
	}
	return opts
}

// withLimits is m with the SDK's limits l, each with its source: MaxTokens is
// the output's chosen value, ContextWindow the window's, else the input
// limit's, 0 for unknown (0066-MADR D4, D9).
func withLimits(m llm.Model, l catalog.Limits) llm.Model {
	m.Limits = llm.Limits{Output: factOf(l.Output), StreamedOutput: factOf(l.StreamedOutput), Window: factOf(l.Window),
		Input: factOf(l.Input), Required: l.Required}
	m.MaxTokens, m.ContextWindow = m.Limits.Output.Tokens, 0
	switch {
	case m.Limits.Window.Known:
		m.ContextWindow = m.Limits.Window.Tokens
	case m.Limits.Input.Known:
		m.ContextWindow = m.Limits.Input.Tokens
	}
	return m
}

// factOf is the SDK's fact f as llm's: the chosen evidence's source and
// staleness, and every other statement in Others.
func factOf(f catalog.Fact) llm.LimitFact {
	out := llm.LimitFact{Tokens: f.Tokens, Known: f.Known, Conflicting: f.Conflicting}
	ev := f.Evidence
	if f.Known && len(ev) > 0 {
		out.Source, out.Stale, ev = ev[0].Source, ev[0].Stale, ev[1:]
	}
	for _, e := range ev {
		out.Others = append(out.Others, llm.LimitFact{Tokens: e.Tokens, Known: e.Tokens > 0, Source: e.Source, Stale: e.Stale})
	}
	return out
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
