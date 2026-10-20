package cli

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// modelPick is what --provider, --model and --thinking chose for the run
// (0005-PLAN F4-2c): the model's provider and id, and the thinking level;
// each empty when not given.
type modelPick struct {
	Provider, Model, Thinking string
}

// pickModel resolves --provider and --model by Pi's rules over the offline
// model set, with Pi's warning for a custom id. --thinking wins over a
// ":level" suffix. A resolution error is a usage error.
func (e *runEnv) pickModel(f *SharedFlags) (modelPick, error) {
	if f.Model == "" {
		return modelPick{Thinking: f.Thinking}, nil
	}
	res, err := catalog.ResolveModel(f.Provider, f.Model, f.Thinking, offlineModels(e.ctx, e.modelsJSON()), e.hasCredential)
	if err != nil {
		return modelPick{}, usageErrorf("%v", err)
	}
	if res.Warning != "" {
		e.out.Warnf("%s", res.Warning)
	}
	e.rememberModel(res.Model.Provider + "/" + res.Model.ID) // --model chooses too (0025-MADR D3)
	return modelPick{Provider: res.Model.Provider, Model: res.Model.ID, Thinking: cmp.Or(f.Thinking, res.Level)}, nil
}

// offlineModels is the offline model set (0005-PLAN F4-2c decision 2): each
// key provider's known models, then models.json's, each with what the
// catalog knows of it, and no request made; ctx bounds the lookups.
func offlineModels(ctx context.Context, mj *catalog.ModelsJSON) []llm.Model {
	var out []llm.Model
	for _, p := range provider.KeyProviders() {
		for _, id := range withModelsJSON(provider.KnownModels(p), mj, p) {
			out = append(out, provider.ModelInfoContext(ctx, p, "", id, mj))
		}
	}
	return out
}

// chosenProvider is the run's provider: --model's, else where the run
// starts, by the defaultProvider setting, the recent models or the registry's
// order (0025-MADR D3); empty with no credential and no setting.
func (e *runEnv) chosenProvider() string { return cmp.Or(e.pick.Provider, e.startProvider()) }

// defaultModel is the defaultModel setting as provider id's new-session
// model: used with defaultProvider when ids, the model choice, or the
// offline set holds it; otherwise "", with one warning. --model leaves it
// unread (0005-PLAN F4-2c decision 7).
func (e *runEnv) defaultModel(id string, ids []string) string {
	s := e.processSettings("").Settings
	if s.DefaultModel == "" || e.pick.Model != "" {
		return ""
	}
	switch {
	case s.DefaultProvider == "":
		e.defaultModelOnce.Do(func() { e.warnf("defaultModel %s is ignored: it needs defaultProvider", s.DefaultModel) })
	case slices.Contains(ids, s.DefaultModel) || slices.Contains(withModelsJSON(provider.KnownModels(id), e.modelsJSON(), id), s.DefaultModel):
		return s.DefaultModel
	default:
		e.defaultModelOnce.Do(func() {
			e.warnf("defaultModel %s is not a model of %s that gobble knows, so a new session starts with its first listed model", s.DefaultModel, id)
		})
	}
	return ""
}

// noCredentialMessage names the variable the run's provider needs: --model's,
// the defaultProvider setting's, or, with neither, the variables the
// ambient provider reads.
func (e *runEnv) noCredentialMessage() string {
	if v := provider.CredentialVar(e.pick.Provider); v != "" {
		return "no model credential: --model is " + e.pick.Provider + "/" + e.pick.Model + "; set " + v
	}
	if preferred := e.defaultProvider(); preferred != "" {
		if v := provider.CredentialVar(preferred); v != "" {
			return "no model credential: defaultProvider is " + preferred + "; set " + v
		}
	}
	return "no model credential: set one of " + strings.Join(provider.CredentialVars(), ", ")
}
