package cli

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// startChoice is where a run starts when --model names no provider, chosen
// once a process (0025-MADR D3).
type startChoice struct {
	provider, model string
	note            string
}

// startProvider is the provider a run starts on when --model names none:
// the defaultProvider setting's; else the newest recent model's whose
// provider has a credential and is not refused; else the first provider
// with a key, in the registry's order. A setting or a recent model that
// cannot be used falls to the next, with a warning. With no credential at
// all it is the setting's, so the error names its variable (0019-MADR D15;
// 0025-MADR D3).
func (e *runEnv) startProvider() string { return e.start().provider }

// startModel is the recent model a run starts on, when its provider was
// chosen from the recent models; "" otherwise.
func (e *runEnv) startModel() string { return e.start().model }

// startNote is the line session's note when the run started on the first
// provider with a key while several had one, so the user knows how to
// choose; "" otherwise.
func (e *runEnv) startNote() string { return e.start().note }

func (e *runEnv) start() startChoice {
	e.startMu.Lock()
	defer e.startMu.Unlock()
	if !e.started {
		e.startAt, e.started = e.chooseStart(), true
	}
	return e.startAt
}

// restart has the next start() choose again, after the credentials
// changed (0025-MADR D6).
func (e *runEnv) restart() {
	e.startMu.Lock()
	defer e.startMu.Unlock()
	e.started = false
}

func (e *runEnv) chooseStart() startChoice {
	if e.pick.Provider != "" { // --model chose; nothing is waited for or noted
		return startChoice{provider: e.pick.Provider}
	}
	keyed := slices.DeleteFunc(provider.KeyProviders(), func(id string) bool { return !e.hasCredential(id) })
	setting := e.defaultProvider()
	if len(keyed) == 0 {
		return startChoice{provider: setting}
	}
	if setting != "" {
		why := e.unusable(setting)
		if why == "" {
			return startChoice{provider: setting}
		}
		e.warnf("defaultProvider %s %s; the run starts on another provider", setting, why)
	}
	for _, ref := range e.recentModels() {
		id, model, ok := strings.Cut(ref, "/")
		if !ok || !slices.Contains(keyed, id) {
			continue
		}
		if why := e.unusable(id); why != "" {
			e.warnf("%s is not used: %s", ref, why)
			continue
		}
		return startChoice{provider: id, model: model}
	}
	first := keyed[0]
	c := startChoice{provider: first}
	if len(keyed) > 1 {
		c.note = fmt.Sprintf("Starting on %s (%s), the first provider with a key; /model switches, and the defaultProvider setting or --model chooses where a run starts.",
			provider.Label(first), first)
	}
	return c
}

// unusable is why provider id cannot start a run: no credential, or a
// listing that refused its key or quota; "" when it can. It waits for the
// provider's listing, which the agent started (0025-MADR D3, D4).
func (e *runEnv) unusable(id string) string {
	if !e.hasCredential(id) {
		return "has no credential"
	}
	state, reason, _ := listingState(id, e.modelListing(provider.Selection{ID: id}).get(), time.Now())
	if !state.Usable() {
		return strings.TrimSpace(string(state) + " " + reason)
	}
	return ""
}
