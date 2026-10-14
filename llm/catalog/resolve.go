package catalog

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/llm"
)

// ProviderID is gobble's id for name, an SDK id or Pi's name for one
// (0004-MADR), without case.
func ProviderID(name string) (string, bool) {
	n := strings.ToLower(strings.TrimSpace(name))
	if id, ok := piAliases[n]; ok {
		return id, true
	}
	if _, ok := providerAPIs[n]; ok {
		return n, true
	}
	return "", false
}

// Resolution is what --provider and --model chose (0005-PLAN F4-2c).
type Resolution struct {
	// Model is the chosen model; a custom one has only Provider and ID.
	Model llm.Model
	// Level is a ":level" suffix's thinking level, or "".
	Level string
	// Custom reports a model id the models do not hold, used as given.
	Custom bool
	// Warning is Pi's warning for a custom id, or "".
	Warning string
}

// ResolutionError is a choice ResolveModel cannot make, in Pi's words.
type ResolutionError struct{ text string }

func (e *ResolutionError) Error() string { return e.text }

func resolutionError(format string, args ...any) error {
	return &ResolutionError{text: fmt.Sprintf(format, args...)}
}

// ResolveModel resolves --provider, --model and --thinking over models by
// Pi's resolveCliModel (core/model-resolver.ts:406-606), in its order and
// words. authed reports whether a provider has a credential. An empty model
// is no choice.
func ResolveModel(provider, model, thinking string, models []llm.Model, authed func(string) bool) (Resolution, error) {
	if model == "" {
		return Resolution{}, nil
	}
	if len(models) == 0 {
		return Resolution{}, resolutionError("No models available. Check your installation or add models to models.json.")
	}
	canon := providerNames(models)
	prov := ""
	if provider != "" {
		p, ok := canon(provider)
		if !ok {
			return Resolution{}, resolutionError("Unknown provider %q. Use --list-models to see available providers/models.", provider)
		}
		prov = p
	}
	pattern, inferred := model, false
	if prov == "" {
		if before, after, ok := strings.Cut(model, "/"); ok {
			if p, ok := canon(before); ok {
				prov, pattern, inferred = p, after, true
			}
		}
	}
	if prov == "" {
		if r, done, err := exactAcross(model, models, authed); done {
			return r, err
		}
	}
	if provider != "" {
		pattern = stripProvider(model, prov, provider)
	}
	candidates := models
	if prov != "" {
		candidates = slices.DeleteFunc(slices.Clone(models), func(m llm.Model) bool { return m.Provider != prov })
	}
	if m, level, ok := parsePattern(pattern, candidates); ok {
		if inferred && !authed(m.Provider) {
			if raw := credentialedRaw(model, m, models, authed); raw != nil {
				return Resolution{Model: *raw}, nil
			}
		}
		return Resolution{Model: m, Level: level}, nil
	}
	if inferred {
		if m, ok := firstExact(model, models); ok {
			return Resolution{Model: m}, nil
		}
		if m, level, ok := parsePattern(model, models); ok {
			return Resolution{Model: m, Level: level}, nil
		}
	}
	if prov != "" {
		return custom(prov, pattern, thinking), nil
	}
	return Resolution{}, resolutionError("Model %q not found. Use --list-models to see available models.", model)
}

// providerNames is the canonical provider of a name, without case: a
// provider of models, or Pi's name for one.
func providerNames(models []llm.Model) func(string) (string, bool) {
	byLower := map[string]string{}
	for _, m := range models {
		byLower[strings.ToLower(m.Provider)] = m.Provider
	}
	return func(name string) (string, bool) {
		if p, ok := byLower[strings.ToLower(name)]; ok {
			return p, true
		}
		if id, ok := ProviderID(name); ok {
			p, ok := byLower[id]
			return p, ok
		}
		return "", false
	}
}

// exactAcross is Pi's exact match across every provider when none is named:
// one match wins, then the sole credentialed one, else Pi's ambiguity error.
// done is false when nothing matched exactly.
func exactAcross(model string, models []llm.Model, authed func(string) bool) (r Resolution, done bool, err error) {
	exact := matching(models, func(m llm.Model) bool { return isExactRef(m, model) })
	switch len(exact) {
	case 0:
		return Resolution{}, false, nil
	case 1:
		return Resolution{Model: exact[0]}, true, nil
	}
	withKey := matching(exact, func(m llm.Model) bool { return authed(m.Provider) })
	if len(withKey) == 1 {
		return Resolution{Model: withKey[0]}, true, nil
	}
	refs := make([]string, len(exact))
	for i, m := range exact {
		refs[i] = m.Provider + "/" + m.ID
	}
	slices.SortFunc(refs, CompareIDs)
	hint := "More than one matching provider is authenticated."
	if len(withKey) == 0 {
		hint = "No matching provider is authenticated."
	}
	return Resolution{}, true, resolutionError("Model %q is ambiguous across providers: %s. %s Use --provider or provider/model.",
		model, strings.Join(refs, ", "), hint)
}

// stripProvider is model without a "<provider>/" prefix, tolerated when
// --provider names it, as the canonical id or as given.
func stripProvider(model, prov, given string) string {
	for _, p := range []string{prov, given} {
		if len(model) > len(p) && strings.EqualFold(model[:len(p)+1], p+"/") {
			return model[len(p)+1:]
		}
	}
	return model
}

// credentialedRaw is the sole credentialed model, other than m, whose id is
// the whole input: a provider taken from the prefix gives way to it when its
// own has no credential.
func credentialedRaw(model string, m llm.Model, models []llm.Model, authed func(string) bool) *llm.Model {
	raw := matching(models, func(o llm.Model) bool {
		return strings.EqualFold(o.ID, model) && !sameModel(o, m) && authed(o.Provider)
	})
	if len(raw) != 1 {
		return nil
	}
	return &raw[0]
}

// firstExact is the first model whose id or provider/id is model, without
// case.
func firstExact(model string, models []llm.Model) (llm.Model, bool) {
	i := slices.IndexFunc(models, func(m llm.Model) bool { return isExactRef(m, model) })
	if i < 0 {
		return llm.Model{}, false
	}
	return models[i], true
}

// custom is Pi's custom model: the id, with a ":level" split off unless
// --thinking is given, and Pi's warning. gobble copies no other model's
// facts onto it (0005-PLAN F4-2c decision 3).
func custom(prov, pattern, thinking string) Resolution {
	id, level := pattern, ""
	if thinking == "" {
		if i := strings.LastIndex(pattern, ":"); i >= 0 && isLevel(pattern[i+1:]) {
			id, level = pattern[:i], pattern[i+1:]
		}
	}
	return Resolution{Model: llm.Model{Provider: prov, ID: id}, Level: level, Custom: true,
		Warning: fmt.Sprintf("Model %q not found for provider %q. Using custom model id.", id, prov)}
}

// parsePattern is Pi's parseModelPattern in its strict, --model form: the
// whole pattern, else a level split at the last ':' and the rest tried
// again; any other suffix is no match.
func parsePattern(pattern string, models []llm.Model) (llm.Model, string, bool) {
	if m, ok := tryMatch(pattern, models); ok {
		return m, "", true
	}
	i := strings.LastIndex(pattern, ":")
	if i < 0 || !isLevel(pattern[i+1:]) {
		return llm.Model{}, "", false
	}
	m, _, ok := parsePattern(pattern[:i], models)
	if !ok {
		return llm.Model{}, "", false
	}
	return m, pattern[i+1:], true
}

// tryMatch is Pi's tryMatchModel: an exact reference, else the models whose
// id or name holds the pattern, an alias before a dated id, the highest id.
func tryMatch(pattern string, models []llm.Model) (llm.Model, bool) {
	if m, ok := exactRef(pattern, models); ok {
		return m, true
	}
	p := strings.ToLower(pattern)
	matches := matching(models, func(m llm.Model) bool {
		return strings.Contains(strings.ToLower(m.ID), p) || strings.Contains(strings.ToLower(m.Name), p)
	})
	if len(matches) == 0 {
		return llm.Model{}, false
	}
	if aliases := matching(matches, func(m llm.Model) bool { return isAlias(m.ID) }); len(aliases) > 0 {
		matches = aliases
	}
	return slices.MaxFunc(matches, func(a, b llm.Model) int { return CompareIDs(a.ID, b.ID) }), true
}

// exactRef is Pi's findExactModelReferenceMatch: a unique provider/id, else
// a unique provider and id split at the first '/', else a unique bare id,
// each without case.
func exactRef(ref string, models []llm.Model) (llm.Model, bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return llm.Model{}, false
	}
	canonical := matching(models, func(m llm.Model) bool { return strings.EqualFold(m.Provider+"/"+m.ID, ref) })
	switch len(canonical) {
	case 1:
		return canonical[0], true
	case 0:
	default:
		return llm.Model{}, false
	}
	if p, id, ok := strings.Cut(ref, "/"); ok {
		p, id = strings.TrimSpace(p), strings.TrimSpace(id)
		if p != "" && id != "" {
			split := matching(models, func(m llm.Model) bool { return strings.EqualFold(m.Provider, p) && strings.EqualFold(m.ID, id) })
			switch len(split) {
			case 1:
				return split[0], true
			case 0:
			default:
				return llm.Model{}, false
			}
		}
	}
	ids := matching(models, func(m llm.Model) bool { return strings.EqualFold(m.ID, ref) })
	if len(ids) != 1 {
		return llm.Model{}, false
	}
	return ids[0], true
}

// datedID is an id ending in a date, -YYYYMMDD.
var datedID = regexp.MustCompile(`-\d{8}$`)

// isAlias is Pi's isAlias: an id ending in -latest, or in no date.
func isAlias(id string) bool { return strings.HasSuffix(id, "-latest") || !datedID.MatchString(id) }

// isLevel reports whether s is one of Pi's thinking levels, off to max.
func isLevel(s string) bool { return slices.Contains(levelOrder, s) }

// isExactRef reports whether model is m's id or provider/id, without case.
func isExactRef(m llm.Model, model string) bool {
	return strings.EqualFold(m.ID, model) || strings.EqualFold(m.Provider+"/"+m.ID, model)
}

func sameModel(a, b llm.Model) bool { return a.Provider == b.Provider && a.ID == b.ID }

// matching is the models keep accepts, in order.
func matching(models []llm.Model, keep func(llm.Model) bool) []llm.Model {
	var out []llm.Model
	for _, m := range models {
		if keep(m) {
			out = append(out, m)
		}
	}
	return out
}

// CompareIDs orders ids as Pi's localeCompare orders ids of letters,
// digits, '-', '.' and '/': without case first, then by byte.
func CompareIDs(a, b string) int {
	if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}
