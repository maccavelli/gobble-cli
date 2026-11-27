package provider

import (
	"context"
	"runtime/debug"
	"slices"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

	"github.com/maccavelli/gobble-cli/llm"
	gcatalog "github.com/maccavelli/gobble-cli/llm/catalog"
)

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

// ModelInfo is what the catalog knows of model on provider id, from the
// current document (CurrentModelData: the copy gobble keeps, else none;
// 0072-MADR D7) and, with apiKey, the provider's listing as the SDK has
// it cached. It waits for no listing: one not yet cached is fetched in the
// background for a later call (0005-PLAN F4-2a). With a login, the SDK first
// asks it for a token, which may refresh at the issuer before ModelInfo
// returns (go-llmprovider-sdk catalog.CachedModel; 0046-MADR).
// A model nothing knows has only its provider and id. mj, when not nil, is
// models.json: a layer above the document, and its thinkingLevelMap
// (0005-PLAN F4-2b). With no apiKey, a provider with an OAuth login uses
// logins' stored login, when logins is not nil (0005-PLAN F4-3b). workspace
// is a Claude key's, for its listing (0035-MADR D5).
func ModelInfo(id, apiKey, workspace string, logins LoginStore, model string, mj *gcatalog.ModelsJSON) llm.Model {
	return ModelInfoWith(Options{APIKey: apiKey, Workspace: workspace, Logins: logins}, id, model, mj)
}

// ModelInfoWith is ModelInfo with o's key, workspace, logins and sessions,
// o's document (o.ModelData: a session's; nil reads the current one,
// 0072-MADR D19, M80), and o's transport and cache: a listing it starts in the background runs
// through o.Cache when set, so the run that owns the cache ends it, and a
// login's refresh is on the run's session (0046-MADR D7, D8). With a
// ClientName, the lookup names that application and ClientVersion to the SDK,
// which names a login's session with them before it reads its token, so a
// refresh the lookup starts, and the listing it starts in the background,
// name it (0051-MADR D9, N6). o's Observations, the caps services stated on
// earlier runs, are a layer below models.json's (0071-MADR P24). o's other
// fields are ignored.
func ModelInfoWith(o Options, id, model string, mj *gcatalog.ModelsJSON) llm.Model {
	ctx := context.Background()
	src, workspace, sdk := lookupSource(ctx, o, id)
	return modelInfoFrom(ctx, id, src, workspace, model, mj, o.ClientName, o.ClientVersion, o.ModelData, savedLayer(o, id), sdk...)
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
// version, o's document (o.ModelData, else the current one) handed in by
// catalog.WithMetadata (0072-MADR M55), o's saved caps as the
// "service-stated" layer, and models.json above it (0071-MADR P24). It passes no Kilo organization (0066-MADR S43). The SDK
// bounds it at 10 seconds; ctx ends it sooner. Its error is each source's
// failure, joined; what the other sources gave is kept either way, and a
// lookup reads it.
func Discover(ctx context.Context, o Options, id, model string, mj *gcatalog.ModelsJSON) error {
	src, workspace, sdk := lookupSource(ctx, o, id)
	sdk = append(sdk, documentOf(o.ModelData).options()...)
	opts := lookupOptions(workspace, o.ClientName, o.ClientVersion, sdk, lookupLayers(mj, savedLayer(o, id))...)
	_, err := lookupLimits(ctx, llmprovider.ProviderID(id), limitsSource(id, src), model, opts...)
	return err
}

// CachedLimits are model's limits on provider id as they are cached now,
// with what a lookup reads with (o's document, handed in, the saved caps' and
// models.json's layers, the run's cache and set, a Claude key's workspace),
// as the SDK's
// catalog.CachedLimits reads them: it starts no request. A key's token keys
// the cached listing; a stored login's is not asked, since it may refresh,
// so a provider signed in with one reads the layers alone (0066-MADR D6,
// S19; 0071-MADR P46).
func CachedLimits(o Options, id, model string, mj *gcatalog.ModelsJSON) llm.Limits {
	sdk := Options{HTTPClient: o.HTTPClient, Headers: o.Headers, BaseURL: o.BaseURL, Cache: o.Cache, Set: o.Set}.sdkOptions()
	sdk = append(sdk, Options{BaseURL: o.BaseURL, Set: o.Set}.chatOptions(id)...)
	var src llmprovider.TokenSource
	if o.APIKey != "" {
		src = llmprovider.NewStaticToken(o.APIKey)
	}
	sdk = append(sdk, documentOf(o.ModelData).options()...)
	opts := lookupOptions(o.Workspace, o.ClientName, o.ClientVersion, sdk, lookupLayers(mj, savedLayer(o, id))...)
	if ls := limitsSource(id, src); ls != nil {
		opts = append(opts, llmprovider.WithTokenSource(ls))
	}
	return withLimits(llm.Model{}, catalog.CachedLimits(context.Background(), llmprovider.ProviderID(id), model, opts...)).Limits
}

// lookupLimits is the SDK's catalog.LookupLimits; a test replaces it to read
// the options discovery hands it.
var lookupLimits = catalog.LookupLimits

// documentOf is the document a lookup reads: data, a session's, else the
// current one (CurrentModelData), never two (0072-MADR M19, D19, M80). It is
// handed to the SDK by catalog.WithMetadata, the hand-in, not as a layer
// (0072-MADR D17, M55; 0093-MADR D1 at go-llmprovider-sdk v1.33.0).
func documentOf(data *ModelData) *ModelData {
	if data == nil {
		return CurrentModelData()
	}
	return data
}

// lookupLayers are a lookup's caller layers: saved, the caps services stated
// on earlier runs, when not nil; then models.json, when mj is not nil, above
// it (0071-MADR P21, P24). The models.dev document is not a layer: documentOf
// hands it in (0072-MADR M55, M75).
func lookupLayers(mj *gcatalog.ModelsJSON, saved catalog.Layer) []catalog.Layer {
	var layers []catalog.Layer
	if saved != nil {
		layers = append(layers, saved)
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
	return modelInfoFrom(ctx, id, src, "", model, mj, "", "", nil, nil)
}

// modelInfoFrom is ModelInfoContext with src, the credential's token source, or
// nil for none, a Claude key's workspace, the application's name and version
// (clientName empty names none), data, the document, handed in (nil: the
// current one), saved, the saved caps' layer or nil, and sdk, the SDK
// options a listing it starts takes (0046-MADR D7; 0071-MADR P24; 0072-MADR
// M55, M80).
func modelInfoFrom(ctx context.Context, id string, src llmprovider.TokenSource, workspace, model string, mj *gcatalog.ModelsJSON, clientName, clientVersion string, data *ModelData, saved catalog.Layer, sdk ...llmprovider.Option) llm.Model {
	sdk = append(slices.Clone(sdk), documentOf(data).options()...)
	m := lookupModelWith(ctx, id, src, model, workspace, clientName, clientVersion, sdk, lookupLayers(mj, saved)...)
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
// its curated models, then the current document's models for it that take
// tools, by id, each once (0005-PLAN F4-2c; 0072-MADR D7), as Pi's generator
// keeps only those.
func KnownModels(id string) []string {
	d := CurrentModelData()
	if d == nil { // the curated models alone: no model data (0072-MADR D13)
		return Models(id)
	}
	return toolModels(id, &d.doc.meta)
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
// marked known, so it outranks the document's field by field.
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
