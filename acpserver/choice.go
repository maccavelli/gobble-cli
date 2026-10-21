package acpserver

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/personality"
)

// ProviderState is a provider's state in the model choice: whether its
// models may be chosen, and why not (0025-MADR D4).
type ProviderState string

// The states, in gobble's words. A reason beside one is gobble's words for
// the failure's kind and its HTTP status, never the provider's text, which
// can carry part of the key (0025-PLAN deviation 1; 0020-MADR F5).
const (
	StateLive           ProviderState = "live"
	StateStale          ProviderState = "stale"
	StateNotChecked     ProviderState = "not checked"
	StateNotListed      ProviderState = "not listed"
	StateListing        ProviderState = "listing"
	StateKeyRejected    ProviderState = "key rejected"
	StateQuotaExhausted ProviderState = "quota exhausted"
	StateNotPermitted   ProviderState = "not permitted"
	StateNoCredential   ProviderState = "no credential" //nolint:gosec // G101: a state's name, not a credential
)

// Usable reports whether a provider in state s offers its models: a live,
// stale, unchecked or still-listing one, and one not listed, which offers
// its static models.
func (s ProviderState) Usable() bool {
	switch s {
	case StateLive, StateStale, StateNotChecked, StateNotListed, StateListing:
		return true
	}
	return false
}

// ProviderChoice is one provider in the model choice (0025-MADR D4).
type ProviderChoice struct {
	ID, Label string
	State     ProviderState
	// Reason is the state's detail and the way out, such as "(HTTP 401) —
	// gobble auth login claude"; "" for none.
	Reason string
	// Models are its models, offered when State is usable.
	Models []string
	// Age is a stale listing's age.
	Age time.Duration
	// Source is where its credential comes from, in Pi's words, such as
	// "env: OPENAI_API_KEY"; Methods how to sign in, such as "API key;
	// browser or device sign-in" (0025-MADR D5).
	Source, Methods string
	// Known are the models gobble knows of it without a listing, shown by
	// /models when it offers none.
	Known []string
}

// StateOf is provider id's state after err, a listing's or a turn's
// failure, and its reason: gobble's words for the kind, the HTTP status when
// err carries one, and for a refused key the way to sign in again. A nil err
// is live (0025-MADR D4, deviation 1).
func StateOf(id string, err error) (ProviderState, string) {
	status := ""
	if ae, ok := errors.AsType[*llm.APIError](err); ok && ae.Status > 0 {
		status = fmt.Sprintf("(HTTP %d)", ae.Status)
	}
	switch {
	case err == nil:
		return StateLive, ""
	case errors.Is(err, llm.ErrQuota):
		return StateQuotaExhausted, status
	case errors.Is(err, llm.ErrAuth):
		return StateKeyRejected, joinReason(status, "— gobble auth login "+id)
	case errors.Is(err, llm.ErrNotPermitted):
		return StateNotPermitted, status
	case errors.Is(err, llm.ErrOffline):
		return StateNotListed, "(offline)"
	case errors.Is(err, context.DeadlineExceeded):
		return StateNotListed, "(timed out)"
	case errors.Is(err, llm.ErrUnavailable), errors.Is(err, llm.ErrRateLimited):
		return StateNotListed, cmpOr(status, "(unavailable)")
	}
	return StateNotListed, cmpOr(status, "(listing failed)")
}

func joinReason(parts ...string) string {
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " ")
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// refOf is model named with its provider, provider/model, or model alone
// when there is no provider.
func refOf(provider, model string) string {
	if provider == "" {
		return model
	}
	return provider + "/" + model
}

// providerList is c's providers: its Providers, or Provider's models alone,
// live, for a choice built before 0025.
func (c ModelChoice) providerList() []ProviderChoice {
	if len(c.Providers) > 0 {
		return c.Providers
	}
	if len(c.Models) == 0 {
		return nil
	}
	return []ProviderChoice{{ID: c.Provider, Label: cmpOr(c.Provider, "Models"), State: StateLive, Models: c.Models}}
}

// splitRef is the provider and model ref names among ps: the provider whose
// id leads it with a slash, the longest such, or the empty provider when ps
// holds it; ok is false when none does.
func splitRef(ref string, ps []ProviderChoice) (provider, model string, ok bool) {
	best := -1
	for i, p := range ps {
		if p.ID != "" && strings.HasPrefix(ref, p.ID+"/") && (best < 0 || len(p.ID) > len(ps[best].ID)) {
			best = i
		}
	}
	if best >= 0 {
		return ps[best].ID, strings.TrimPrefix(ref, ps[best].ID+"/"), true
	}
	for _, p := range ps {
		if p.ID == "" {
			return "", ref, true
		}
	}
	return "", "", false
}

// pick is one numbered entry of /model's list.
type pick struct {
	section, provider, model string
}

// recentShown is how many recent models /model lists (0025-MADR D5).
const recentShown = 5

// sectionRecent is the recent models' section of /model's list and group of
// the model option.
const sectionRecent = "Recent"

// picks are /model's entries, in their numbered order: the recent models a
// usable provider still offers, up to five, then each usable provider's
// models (0025-MADR D5, deviation 2).
func (c ModelChoice) picks() []pick {
	ps := c.providerList()
	var out []pick
	for _, r := range c.Recent {
		if len(out) == recentShown {
			break
		}
		p, m, ok := splitRef(r, ps)
		if !ok {
			continue
		}
		if i := slices.IndexFunc(ps, func(x ProviderChoice) bool { return x.ID == p }); i >= 0 && ps[i].State.Usable() && slices.Contains(ps[i].Models, m) {
			out = append(out, pick{sectionRecent, p, m})
		}
	}
	for _, p := range ps {
		if p.State.Usable() {
			for _, m := range p.Models {
				out = append(out, pick{p.Label, p.ID, m})
			}
		}
	}
	return out
}

// choice is the model choice with the providers this process saw refused
// marked (0025-MADR D4's lazy marking).
func (a *Agent) choice() ModelChoice {
	c := a.models()
	a.marksMu.Lock()
	defer a.marksMu.Unlock()
	if len(a.marks) == 0 {
		return c
	}
	ps := slices.Clone(c.providerList())
	for i, p := range ps {
		if m, ok := a.marks[p.ID]; ok && m.gen == a.generation(p.ID) {
			ps[i].State, ps[i].Reason = m.state, m.reason
		}
	}
	c.Providers = ps
	return c
}

// modelsOf are provider's offered models, when it is usable (0025-MADR D1).
func (a *Agent) modelsOf(provider string) []string {
	for _, p := range a.choice().providerList() {
		if p.ID == provider && p.State.Usable() {
			return p.Models
		}
	}
	return nil
}

// canonicalProvider is gobble's id for a provider a log names, Pi's
// "anthropic" as "claude" among them, so a Pi session's turns are the same
// model's (0025-MADR D2); a name gobble does not know is kept.
func canonicalProvider(name string) string {
	if id, ok := catalog.ProviderID(name); ok {
		return id
	}
	return name
}

// stamped is a turn's messages with origin on each assistant one; a
// cancelled turn's last assistant message is marked failed, so it is not
// replayed (0025-MADR D2).
func stamped(msgs []llm.Message, origin llm.Origin, cancelled bool) []llm.Message {
	out := slices.Clone(msgs)
	last := -1
	for i := range out {
		if out[i].Role == llm.RoleAssistant {
			out[i].Origin, last = origin, i
		}
	}
	if cancelled && last >= 0 {
		out[last].Origin.Failed = true
	}
	return out
}

// mark is a provider's state after a refused turn, under its credential's
// generation then.
type mark struct {
	state  ProviderState
	reason string
	gen    uint64
}

// generation is provider id's credential generation, or 0 when the
// embedder gives none.
func (a *Agent) generation(id string) uint64 {
	if a.credGen == nil {
		return 0
	}
	return a.credGen(id)
}

// markFailure marks provider when a turn's err says its key was refused,
// its quota is exhausted or it is not permitted; any other failure marks
// nothing (0025-MADR D4).
func (a *Agent) markFailure(provider string, err error) {
	if !errors.Is(err, llm.ErrAuth) && !errors.Is(err, llm.ErrQuota) && !errors.Is(err, llm.ErrNotPermitted) {
		return
	}
	state, reason := StateOf(provider, err)
	gen := a.generation(provider)
	a.marksMu.Lock()
	defer a.marksMu.Unlock()
	if a.marks == nil {
		a.marks = map[string]mark{}
	}
	a.marks[provider] = mark{state, reason, gen}
}

// heading is a provider's line above its models: its label, its state, and
// the state's reason or a stale listing's age.
func heading(p ProviderChoice) string {
	h := "**" + p.Label + "** — " + string(p.State)
	if p.State == StateStale && p.Age > 0 {
		h += " (" + p.Age.Round(time.Minute).String() + " old)"
	}
	if p.Reason != "" {
		h += " " + p.Reason
	}
	return h
}

// pickerText is /model's list as markdown: a section per recent models and
// usable provider, each a table of its entries with their numbers, then the
// providers that offer none, each with its state (0025-MADR D5, deviation
// 2). keep reports which entries to list; numbers stay those of picks.
func (a *Agent) pickerText(c ModelChoice, picks []pick, keep func(int) bool) string {
	var b strings.Builder
	section := ""
	ps := c.providerList()
	for i, p := range picks {
		if !keep(i) {
			continue
		}
		if p.section != section {
			section = p.section
			title := "**" + sectionRecent + "**"
			if section != sectionRecent {
				if j := slices.IndexFunc(ps, func(x ProviderChoice) bool { return x.ID == p.provider }); j >= 0 {
					title = heading(ps[j])
				}
			}
			b.WriteString("\n" + title + "\n\n| # | Model | Context | In $/M | Out $/M | Thinking |\n| --: | :-- | --: | --: | --: | :-- |\n")
		}
		info := a.info(p.provider, p.model)
		fmt.Fprintf(&b, "| %d | %s | %s | %s | %s | %s |\n", i+1, refOf(p.provider, p.model),
			tokensShort(info.ContextWindow), price(info, info.Cost.Input), price(info, info.Cost.Output), levelRange(catalog.Levels(info)))
	}
	if all := keep(-1); all {
		for _, p := range ps {
			if !p.State.Usable() {
				b.WriteString("\n" + heading(p) + "\n")
			}
		}
	}
	b.WriteString("\n/model N or /model provider/model switches, and with --default new runs start on it; /model words lists the matches.\n")
	return b.String()
}

// tokensShort is a window as 200K or 1M, or "" when unknown.
func tokensShort(n int) string {
	switch {
	case n <= 0:
		return ""
	case n >= 1_000_000 && n%1_000_000 == 0:
		return fmt.Sprintf("%dM", n/1_000_000)
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%dK", n/1000)
	default:
		return strconv.Itoa(n)
	}
}

// price is a rate per million tokens, or "" when the model is not priced.
func price(info llm.Model, rate float64) string {
	if !info.Priced {
		return ""
	}
	return fmt.Sprintf("%.2f", rate)
}

// levelRange is a model's thinking levels as their first and last, or ""
// for a model that only offers off.
func levelRange(levels []string) string {
	switch {
	case len(levels) == 0 || len(levels) == 1 && levels[0] == thinkingOff:
		return ""
	case len(levels) == 1:
		return levels[0]
	}
	return levels[0] + "–" + levels[len(levels)-1]
}

// matches reports which of picks match words: each word in the entry's ref
// or the model's name, or, with SearchModels, the SDK's matcher over the
// provider's models (0025-MADR D5).
func (a *Agent) matches(picks []pick, words string) func(int) bool {
	q := strings.ToLower(strings.TrimSpace(words))
	found := map[string]bool{}
	if a.search != nil {
		byProvider := map[string][]string{}
		for _, p := range picks {
			if !slices.Contains(byProvider[p.provider], p.model) {
				byProvider[p.provider] = append(byProvider[p.provider], p.model)
			}
		}
		for prov, models := range byProvider {
			for _, m := range a.search(prov, models, q) {
				found[refOf(prov, m)] = true
			}
		}
	}
	return func(i int) bool {
		if i < 0 {
			return false
		}
		ref := strings.ToLower(refOf(picks[i].provider, picks[i].model))
		if found[refOf(picks[i].provider, picks[i].model)] {
			return true
		}
		name := strings.ToLower(a.info(picks[i].provider, picks[i].model).Name)
		for w := range strings.FieldsSeq(q) {
			if !strings.Contains(ref, w) && !strings.Contains(name, w) {
				return false
			}
		}
		return q != ""
	}
}

// stateText is a provider's state as /providers shows it: the state, a
// stale listing's age, and its reason.
func stateText(p ProviderChoice) string {
	s := string(p.State)
	if p.State == StateStale && p.Age > 0 {
		s += " (" + p.Age.Round(time.Minute).String() + " old)"
	}
	return joinReason(s, p.Reason)
}

// cmdProviders is /providers: the session's providers as a table, each with
// its state, its credential's source in Pi's words and how to sign in
// (0025-MADR D5).
func (a *Agent) cmdProviders(_ context.Context, t *cmdTurn, _ string) error {
	ps := a.choice().providerList()
	if len(ps) == 0 {
		t.say(t.text(personality.ModelNone, nil))
		return nil
	}
	var b strings.Builder
	b.WriteString("| Provider | State | Credential | Sign-in |\n| :-- | :-- | :-- | :-- |\n")
	for _, p := range ps {
		fmt.Fprintf(&b, "| %s (%s) | %s | %s | %s |\n", p.Label, p.ID, stateText(p), p.Source, p.Methods)
	}
	b.WriteString("\n/models <provider> lists one provider's models.\n")
	t.say(b.String())
	return nil
}

// cmdModels is /models [provider] [words]: one provider's models, or every
// provider's, those matching words, numbered as /model numbers them, so
// /model N picks one; a provider that offers none shows its state and the
// models gobble knows of it (0025-MADR D5).
func (a *Agent) cmdModels(_ context.Context, t *cmdTurn, arg string) error {
	c := a.choice()
	ps := c.providerList()
	words := strings.TrimSpace(arg)
	prov := ""
	if first, rest, _ := strings.Cut(words, " "); first != "" && slices.ContainsFunc(ps, func(p ProviderChoice) bool { return p.ID == first }) {
		prov, words = first, strings.TrimSpace(rest)
	}
	if prov != "" {
		p := ps[slices.IndexFunc(ps, func(x ProviderChoice) bool { return x.ID == prov })]
		if !p.State.Usable() {
			refs := make([]string, len(p.Known))
			for i, m := range p.Known {
				refs[i] = refOf(p.ID, m)
			}
			text := heading(p)
			if len(refs) > 0 {
				text += "\n\nKnown models, offered once it is usable: " + strings.Join(refs, ", ") + "."
			}
			t.say(text)
			return nil
		}
	}
	picks := c.picks()
	match := func(int) bool { return true }
	if words != "" {
		match = a.matches(picks, words)
	}
	keep := func(i int) bool {
		if i < 0 {
			return prov == "" && words == ""
		}
		return picks[i].section != sectionRecent && (prov == "" || picks[i].provider == prov) && match(i)
	}
	for i := range picks {
		if keep(i) {
			t.say(a.pickerText(c, picks, keep))
			return nil
		}
	}
	t.say(fmt.Sprintf("No model matches %q.", cmpOr(words, prov)))
	return nil
}
