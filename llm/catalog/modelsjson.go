package catalog

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/llm"
)

// ModelsJSON is a parsed models.json, in Pi's schema: per provider, a base
// URL, headers, models to add and overrides of models' facts (0005-PLAN
// F4-2b). Its providers are the ones gobble can use, by their SDK ids.
type ModelsJSON struct {
	providers map[string]mjProvider
}

// The fields of Pi's schema gobble reads; a field it does not use is kept
// raw, so that its presence can be reported.
type (
	mjFile struct {
		Providers *map[string]mjProvider `json:"providers"`
	}
	mjProvider struct {
		Name           *string            `json:"name"`
		BaseURL        string             `json:"baseUrl"`
		APIKey         *string            `json:"apiKey"`
		API            string             `json:"api"`
		Headers        map[string]string  `json:"headers"`
		Models         []mjModel          `json:"models"`
		ModelOverrides map[string]mjModel `json:"modelOverrides"`
		Compat         jsontext.Value     `json:"compat"`
		OAuth          jsontext.Value     `json:"oauth"`
		AuthHeader     *bool              `json:"authHeader"`
	}
	mjModel struct {
		ID               string             `json:"id"`
		Name             *string            `json:"name"`
		API              string             `json:"api"`
		BaseURL          string             `json:"baseUrl"`
		Reasoning        *bool              `json:"reasoning"`
		ThinkingLevelMap map[string]*string `json:"thinkingLevelMap"`
		Input            *[]string          `json:"input"`
		Cost             *mjCost            `json:"cost"`
		ContextWindow    *int               `json:"contextWindow"`
		MaxTokens        *int               `json:"maxTokens"`
		Headers          map[string]string  `json:"headers"`
		InputLimits      jsontext.Value     `json:"inputLimits"`
		PromptCache      jsontext.Value     `json:"promptCache"`
		SamplingParams   jsontext.Value     `json:"samplingParams"`
		Compat           jsontext.Value     `json:"compat"`
	}
	mjCost struct {
		Input      *float64  `json:"input"`
		Output     *float64  `json:"output"`
		CacheRead  *float64  `json:"cacheRead"`
		CacheWrite *float64  `json:"cacheWrite"`
		Tiers      *[]mjTier `json:"tiers"`
	}
	mjTier struct {
		InputTokensAbove int     `json:"inputTokensAbove"`
		Input            float64 `json:"input"`
		Output           float64 `json:"output"`
		CacheRead        float64 `json:"cacheRead"`
		CacheWrite       float64 `json:"cacheWrite"`
	}
)

// Pi's names for the wires the SDK speaks.
const (
	apiResponses   = "openai-responses"
	apiMessages    = "anthropic-messages"
	apiGoogle      = "google-generative-ai"
	apiCompletions = "openai-completions"
)

// providerAPIs are the SDK's ids, each with Pi's names for the wires it
// speaks: an api a models.json entry may name for it.
var providerAPIs = map[string][]string{
	"openai":       {apiResponses},
	"grok":         {apiResponses},
	"claude":       {apiMessages},
	"gemini":       {apiGoogle},
	"huggingface":  {apiCompletions},
	"kilo":         {apiCompletions},
	"together":     {apiCompletions},
	"ollama":       {apiCompletions},
	"opencode-zen": {apiMessages, apiGoogle, apiCompletions, apiResponses},
	"opencode-go":  {apiMessages, apiGoogle, apiCompletions, apiResponses},
}

// piAliases are Pi's provider ids that name an SDK id (0004-MADR's table).
var piAliases = map[string]string{"anthropic": "claude", "google": "gemini", "xai": "grok"}

// ProviderIDs are the SDK ids models.json may name, sorted.
func ProviderIDs() []string { return slices.Sorted(maps.Keys(providerAPIs)) }

// Pi's defaults for a model models.json defines (provider-composer.ts).
const (
	defaultWindow    = 128000
	defaultMaxTokens = 16384
)

// jsonComments and trailingCommas are Pi's stripJsonComments: each leaves
// string literals as they are.
var (
	jsonComments   = regexp.MustCompile(`"(?:\\.|[^"\\])*"|//[^\n]*`)
	trailingCommas = regexp.MustCompile(`"(?:\\.|[^"\\])*"|,(\s*[}\]])`)
)

// stripJSON removes a BOM, // comments and trailing commas, as Pi does.
func stripJSON(data []byte) []byte {
	data = bytes.TrimPrefix(data, []byte("\uFEFF"))
	data = jsonComments.ReplaceAllFunc(data, func(m []byte) []byte {
		if m[0] == '"' {
			return m
		}
		return nil
	})
	return trailingCommas.ReplaceAllFunc(data, func(m []byte) []byte {
		if m[0] == '"' {
			return m
		}
		return m[1:]
	})
}

// ParseModelsJSON reads a models.json. A file that is not JSON, has no
// providers object, or has a field of the wrong type is an error, and is
// ignored whole, as Pi ignores it. Otherwise each provider gobble cannot use
// is skipped, and each is named in a diagnostic, as is each field gobble
// does not use; sdk is the SDK's version, for the diagnostics.
func ParseModelsJSON(data []byte, sdk string) (*ModelsJSON, []string, error) {
	var f mjFile
	if err := json.Unmarshal(stripJSON(data), &f); err != nil {
		return nil, nil, fmt.Errorf("models.json: %w", err)
	}
	if f.Providers == nil {
		return nil, nil, errors.New("models.json: providers: required")
	}
	if err := checkDefinitions(*f.Providers); err != nil {
		return nil, nil, err
	}
	mj := &ModelsJSON{providers: map[string]mjProvider{}}
	var diags []string
	for _, key := range slices.Sorted(maps.Keys(*f.Providers)) {
		p := (*f.Providers)[key]
		id := key
		if alias, isAlias := piAliases[key]; isAlias {
			id = alias
			if _, both := (*f.Providers)[alias]; both {
				diags = append(diags, fmt.Sprintf("provider %q: %q is also given, and is used; %q is skipped", key, alias, key))
				continue
			}
		}
		if reason := unusable(id, p, sdk); reason != "" {
			diags = append(diags, fmt.Sprintf("provider %q: %s; skipped", key, reason))
			continue
		}
		if unused := unusedFields(p); len(unused) > 0 {
			diags = append(diags, fmt.Sprintf("provider %q: %s not used by gobble; ignored", key, strings.Join(unused, ", ")))
		}
		mj.providers[id] = p
	}
	return mj, diags, nil
}

// checkDefinitions is Pi's schema for what a definition requires: an id,
// and, with a cost, all four rates. A breach ignores the file.
func checkDefinitions(providers map[string]mjProvider) error {
	for _, key := range slices.Sorted(maps.Keys(providers)) {
		for i, m := range providers[key].Models {
			at := fmt.Sprintf("models.json: providers.%s.models.%d", key, i)
			if m.ID == "" {
				return fmt.Errorf("%s.id: required", at)
			}
			if c := m.Cost; c != nil && (c.Input == nil || c.Output == nil || c.CacheRead == nil || c.CacheWrite == nil) {
				return fmt.Errorf("%s.cost: input, output, cacheRead and cacheWrite are required", at)
			}
		}
	}
	return nil
}

// unusable is why gobble cannot use p as provider id, or "".
func unusable(id string, p mjProvider, sdk string) string {
	apis, known := providerAPIs[id]
	switch {
	case !known:
		return fmt.Sprintf("not a provider go-llmprovider-sdk %s builds (one of %s)", sdk, strings.Join(ProviderIDs(), ", "))
	case p.API != "" && !slices.Contains(apis, p.API):
		return fmt.Sprintf("api %q is not %s's wire (%s) in go-llmprovider-sdk %s", p.API, id, strings.Join(apis, ", "), sdk)
	case p.BaseURL == "" && len(p.Headers) == 0 && len(p.Compat) == 0 && len(p.ModelOverrides) == 0 && len(p.Models) == 0 &&
		p.APIKey == nil && len(p.OAuth) == 0 && p.AuthHeader == nil:
		return `must specify "baseUrl", "headers", "compat", "modelOverrides", or "models"`
	}
	for _, m := range p.Models {
		switch {
		case m.API != "" && !slices.Contains(apis, m.API):
			return fmt.Sprintf("model %q: api %q is not %s's wire (%s)", m.ID, m.API, id, strings.Join(apis, ", "))
		case m.ContextWindow != nil && *m.ContextWindow <= 0:
			return fmt.Sprintf("model %q: invalid contextWindow", m.ID)
		case m.MaxTokens != nil && *m.MaxTokens <= 0:
			return fmt.Sprintf("model %q: invalid maxTokens", m.ID)
		}
	}
	return ""
}

// unusedFields are the fields of p gobble does not use, named as Pi names
// them, sorted.
func unusedFields(p mjProvider) []string {
	set := map[string]bool{}
	mark := func(name string, present bool) {
		if present {
			set[name] = true
		}
	}
	mark("name", p.Name != nil)
	mark("compat", len(p.Compat) > 0)
	mark("oauth", len(p.OAuth) > 0)
	mark("authHeader", p.AuthHeader != nil)
	models := slices.Collect(maps.Values(p.ModelOverrides))
	for _, m := range p.Models {
		mark("models[].baseUrl", m.BaseURL != "")
		models = append(models, m)
	}
	for _, m := range models {
		mark("compat", len(m.Compat) > 0)
		mark("inputLimits", len(m.InputLimits) > 0)
		mark("promptCache", len(m.PromptCache) > 0)
		mark("samplingParams", len(m.SamplingParams) > 0)
	}
	return slices.Sorted(maps.Keys(set))
}

// BaseURL is provider id's base URL, or "".
func (mj *ModelsJSON) BaseURL(id string) string {
	if mj == nil {
		return ""
	}
	return mj.providers[id].BaseURL
}

// APIKey is provider id's apiKey as written, in Pi's value grammar, or ""
// (0005-PLAN F4-3a).
func (mj *ModelsJSON) APIKey(id string) string {
	if mj == nil || mj.providers[id].APIKey == nil {
		return ""
	}
	return *mj.providers[id].APIKey
}

// Models are the ids of the models provider id defines, in order, each once.
func (mj *ModelsJSON) Models(id string) []string {
	if mj == nil {
		return nil
	}
	var ids []string
	for _, m := range mj.providers[id].Models {
		if !slices.Contains(ids, m.ID) {
			ids = append(ids, m.ID)
		}
	}
	return ids
}

// definition is the last models[] entry of provider p for model, as Pi's
// upsert keeps the last.
func (p mjProvider) definition(model string) (mjModel, bool) {
	for _, m := range slices.Backward(p.Models) {
		if m.ID == model {
			return m, true
		}
	}
	return mjModel{}, false
}

// HasHeaders reports whether provider id has any header: its own, an
// override's or a definition's.
func (mj *ModelsJSON) HasHeaders(id string) bool {
	if mj == nil {
		return false
	}
	p := mj.providers[id]
	if len(p.Headers) > 0 {
		return true
	}
	for _, m := range p.ModelOverrides {
		if len(m.Headers) > 0 {
			return true
		}
	}
	return slices.ContainsFunc(p.Models, func(m mjModel) bool { return len(m.Headers) > 0 })
}

// Headers are model's raw header values on provider id, in Pi's order: the
// provider's, then the override's, then the definition's, a later one
// winning. They are resolved per request (ResolveValue).
func (mj *ModelsJSON) Headers(id, model string) map[string]string {
	if mj == nil {
		return nil
	}
	p := mj.providers[id]
	out := map[string]string{}
	maps.Copy(out, p.Headers)
	maps.Copy(out, p.ModelOverrides[model].Headers)
	if d, ok := p.definition(model); ok {
		maps.Copy(out, d.Headers)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// Facts are what models.json says of one model, each nil where it says
// nothing. Tiers replace a model's; LevelMap is its merged thinkingLevelMap,
// a null value turning a level off.
type Facts struct {
	Name                                                 *string
	Reasoning                                            *bool
	Input                                                *[]string
	ContextWindow, MaxTokens                             *int
	CostInput, CostOutput, CostCacheRead, CostCacheWrite *float64
	Tiers                                                *[]llm.CostTier
	LevelMap                                             map[string]*string
}

// Facts are model's facts on provider id: a models[] entry's, every field
// set with Pi's defaults, then its modelOverrides entry's over them; false
// when models.json says nothing of it.
func (mj *ModelsJSON) Facts(id, model string) (Facts, bool) {
	if mj == nil {
		return Facts{}, false
	}
	p, ok := mj.providers[id]
	if !ok {
		return Facts{}, false
	}
	var f Facts
	d, defined := p.definition(model)
	if defined {
		f = definedFacts(d)
	}
	o, overridden := p.ModelOverrides[model]
	if overridden {
		f.override(o)
	}
	return f, defined || overridden
}

// definedFacts are a models[] entry's facts, with Pi's defaults for what it
// leaves out.
func definedFacts(d mjModel) Facts {
	f := Facts{Name: new(d.ID), Reasoning: new(false), Input: new([]string{"text"}), ContextWindow: new(defaultWindow),
		MaxTokens: new(defaultMaxTokens), CostInput: new(0.0), CostOutput: new(0.0), CostCacheRead: new(0.0),
		CostCacheWrite: new(0.0), Tiers: new([]llm.CostTier{})}
	f.override(d)
	return f
}

// override sets each field m gives, a cost rate by rate, and merges its
// thinkingLevelMap over f's.
func (f *Facts) override(m mjModel) {
	set := func(dst **int, v *int) {
		if v != nil {
			*dst = new(*v)
		}
	}
	if m.Name != nil {
		f.Name = new(*m.Name)
	}
	if m.Reasoning != nil {
		f.Reasoning = new(*m.Reasoning)
	}
	if m.Input != nil {
		f.Input = new(slices.Clone(*m.Input))
	}
	set(&f.ContextWindow, m.ContextWindow)
	set(&f.MaxTokens, m.MaxTokens)
	if c := m.Cost; c != nil {
		for _, r := range []struct {
			dst **float64
			v   *float64
		}{
			{&f.CostInput, c.Input}, {&f.CostOutput, c.Output}, {&f.CostCacheRead, c.CacheRead}, {&f.CostCacheWrite, c.CacheWrite},
		} {
			if r.v != nil {
				*r.dst = new(*r.v)
			}
		}
		if c.Tiers != nil {
			tiers := make([]llm.CostTier, len(*c.Tiers))
			for i, t := range *c.Tiers {
				tiers[i] = llm.CostTier{Above: t.InputTokensAbove, Input: t.Input, Output: t.Output, CacheRead: t.CacheRead, CacheWrite: t.CacheWrite}
			}
			f.Tiers = &tiers
		}
	}
	if len(m.ThinkingLevelMap) > 0 {
		if f.LevelMap == nil {
			f.LevelMap = map[string]*string{}
		}
		maps.Copy(f.LevelMap, m.ThinkingLevelMap)
	}
}
