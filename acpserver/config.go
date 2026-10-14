package acpserver

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/session"
)

// Config option ids, which are also their categories (0008-MADR D19 item 4).
const (
	configModel    = "model"
	configThinking = "thought_level"
)

// thinkingOff is the level of a session that asks for no reasoning.
const thinkingOff = "off"

// cacheShort is a session's cacheRetention when its settings name none
// (0021-MADR D3).
const cacheShort = "short"

// cacheOf is a cacheRetention setting's hint: none asks for nothing; short
// and long are llm's values, as any other is, for the SDK to refuse
// (0021-MADR D1, D3).
func cacheOf(setting string) llm.CacheRetention {
	if setting == "none" {
		return llm.CacheNone
	}
	return llm.CacheRetention(setting)
}

// modelOf is what the catalog knows of the session provider's model.
func (a *Agent) modelOf(model string) llm.Model { return a.info(a.models().Provider, model) }

// levelsOf are the thinking levels model offers, by Pi's rule
// (catalog.Levels); a model the catalog does not know offers off to xhigh.
func (a *Agent) levelsOf(model string) []string { return catalog.Levels(a.modelOf(model)) }

// SessionSettings are the settings a session takes when it starts
// (0018-MADR D16, D17). A zero field takes the agent's default.
type SessionSettings struct {
	// SteeringMode and FollowUpMode are the queue modes: one-at-a-time or
	// all.
	SteeringMode, FollowUpMode string
	// Thinking is a new session's level; a loaded or resumed session keeps
	// its history's.
	Thinking string
	// CacheRetention is the prompt cache the session's turns ask for:
	// none, short or long (0021-MADR D3).
	CacheRetention string
	// Compaction is /compact's settings.
	Compaction compaction.Settings
	// MCPStartupWait bounds the first prompt's wait for MCP servers.
	MCPStartupWait time.Duration
	// MCPStopGrace and MCPStopTimeout are a stdio server's stop timings;
	// zero leaves mcpclient's own.
	MCPStopGrace, MCPStopTimeout time.Duration
}

// sessionSettings are the settings for a session in cwd: Options.Settings',
// each zero field filled from today's defaults, Options.Compaction and
// MCPOptions.StartupWait among them, so a partial value never mixes
// defaults. A nil Options.Settings is today's defaults.
func (a *Agent) sessionSettings(cwd string) SessionSettings {
	var s SessionSettings
	if a.settings != nil {
		s = a.settings(cwd)
	}
	s.SteeringMode = cmp.Or(s.SteeringMode, queueOneAtATime)
	s.FollowUpMode = cmp.Or(s.FollowUpMode, queueOneAtATime)
	s.Thinking = cmp.Or(s.Thinking, thinkingOff)
	s.CacheRetention = cmp.Or(s.CacheRetention, cacheShort)
	s.Compaction.ReserveTokens = cmp.Or(s.Compaction.ReserveTokens, a.compaction.ReserveTokens)
	s.Compaction.KeepRecentTokens = cmp.Or(s.Compaction.KeepRecentTokens, a.compaction.KeepRecentTokens)
	s.MCPStartupWait = cmp.Or(s.MCPStartupWait, a.mcp.StartupWait, DefaultStartupWait)
	return s
}

// sessionQueue is a session's queues in its settings' modes (0018-PLAN P5
// deviation 18).
func sessionQueue(ss SessionSettings) *queue {
	q := newQueue()
	q.setSteeringMode(ss.SteeringMode)
	q.setFollowUpMode(ss.FollowUpMode)
	return q
}

// ModelChoice is the models a session may choose from: the provider's id,
// its curated models, and the default. A zero ModelChoice offers no model
// option.
type ModelChoice struct {
	Provider string
	Models   []string
	Default  string
}

// configOptions are a session's config options: model, when there are
// models to choose, and thought_level, the model's levels with the
// session's clamped to them. Both are ungrouped selects with a category
// (D19 item 4).
func (a *Agent) configOptions(model, thinking string) []acp.SessionConfigOption {
	choice := a.models()
	var out []acp.SessionConfigOption
	if len(choice.Models) > 0 {
		out = append(out, selectOption(configModel, "Model", acp.SessionConfigOptionCategoryModel, model, choice.Models))
	}
	levels := a.levelsOf(model)
	return append(out, selectOption(configThinking, "Thinking", acp.SessionConfigOptionCategoryThoughtLevel,
		catalog.Clamp(levels, thinking), levels))
}

func selectOption(id, name string, category acp.SessionConfigOptionCategory, current string, values []string) acp.SessionConfigOption {
	opts := make(acp.SessionConfigSelectOptionsUngrouped, len(values))
	for i, v := range values {
		opts[i] = acp.SessionConfigSelectOption{Name: v, Value: acp.SessionConfigValueId(v)}
	}
	return acp.SessionConfigOption{Select: &acp.SessionConfigOptionSelect{
		Id: acp.SessionConfigId(id), Name: name, Category: &category, Type: "select",
		CurrentValue: acp.SessionConfigValueId(current),
		Options:      acp.SessionConfigSelectOptions{Ungrouped: &opts},
	}}
}

// setConfig sets the model or the thinking level from the session's next
// prompt and records the change (model_change, thinking_level_change)
// before anything reports it. A new model clamps the session's level to its
// own, recorded only when that changes it, as Pi does (0005-PLAN F4-2a). It
// is the one path of set_config_option, /model and /thinking.
func (a *Agent) setConfig(ctx context.Context, s *liveSession, id, value string) ([]acp.SessionConfigOption, error) {
	choice := a.models()
	model, think := a.choices(s)
	var entries []session.Entry
	switch id {
	case configModel:
		if !slices.Contains(choice.Models, value) {
			return nil, invalidParams(s.id, fmt.Sprintf("unknown model %q", value))
		}
		clamped := catalog.Clamp(a.levelsOf(value), think)
		a.mu.Lock()
		s.model, s.think = value, clamped
		a.mu.Unlock()
		entries = append(entries, session.Entry{Type: session.TypeModelChange, Provider: choice.Provider, ModelID: value})
		if clamped != think {
			entries = append(entries, session.Entry{Type: session.TypeThinkingLevelChange, ThinkingLevel: clamped})
		}
	case configThinking:
		if !slices.Contains(a.levelsOf(model), value) {
			return nil, invalidParams(s.id, fmt.Sprintf("unknown thinking level %q", value))
		}
		a.mu.Lock()
		s.think = value
		a.mu.Unlock()
		entries = append(entries, session.Entry{Type: session.TypeThinkingLevelChange, ThinkingLevel: value})
	default:
		return nil, invalidParams(s.id, fmt.Sprintf("unknown config option %q", id))
	}
	for _, e := range entries {
		if err := s.append(ctx, a.now(), e); err != nil {
			return nil, appendFailed(err)
		}
	}
	model, think = a.choices(s)
	return a.configOptions(model, think), nil
}

// Override is a run's own model and thinking level, --model's and
// --thinking's, which win over every session the agent opens (0005-PLAN
// F4-2c). Provider is the model's provider; an empty field is not given.
type Override struct {
	Provider, Model, Thinking string
}

// applyOverride sets a session its history restored to the agent's
// Override, and records each change as /model and /thinking record one. A
// model that differs is set; then the level is Thinking clamped to the
// model's, or, with no Thinking, the session's own clamped when the model
// changed, as Pi clamps it (agent-session.ts:2527-2549). A zero Override
// does nothing.
func (a *Agent) applyOverride(ctx context.Context, s *liveSession) error {
	o := a.override
	if o == (Override{}) {
		return nil
	}
	model, think := a.choices(s)
	var entries []session.Entry
	changed := o.Model != "" && o.Model != model
	if changed {
		model = o.Model
		entries = append(entries, session.Entry{Type: session.TypeModelChange, Provider: o.Provider, ModelID: model})
	}
	level := think
	switch {
	case o.Thinking != "":
		level = catalog.Clamp(a.levelsOf(model), o.Thinking)
	case changed:
		level = catalog.Clamp(a.levelsOf(model), think)
	}
	if level != think {
		entries = append(entries, session.Entry{Type: session.TypeThinkingLevelChange, ThinkingLevel: level})
	}
	a.mu.Lock()
	s.model, s.think = model, level
	a.mu.Unlock()
	for _, e := range entries {
		if err := s.append(ctx, a.now(), e); err != nil {
			return appendFailed(err)
		}
	}
	return nil
}

// choices is the session's model and thinking level. setConfig changes them
// from any request's goroutine, so both are read under a.mu (0002-PLAN
// Phase 7, deviation 1).
func (a *Agent) choices(s *liveSession) (model, think string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return s.model, s.think
}

// newSessionMeta is session/new's _meta.gobble: a chosen session id, a
// session to fork, and a name (0002-PLAN Phase 4, owner's decision of
// 2026-10-06).
type newSessionMeta struct {
	SessionID, ForkFrom, Name string
}

func readNewSessionMeta(meta map[string]any) newSessionMeta {
	g, _ := meta[gobble].(map[string]any) //nolint:errcheck // absent or not an object is none
	str := func(k string) string {
		s, _ := g[k].(string) //nolint:errcheck // absent or not a string is none
		return s
	}
	return newSessionMeta{SessionID: str("sessionId"), ForkFrom: str("forkFrom"), Name: str("name")}
}

// parseTimestamp reads an entry timestamp, or gives the zero time.
func parseTimestamp(ts string) time.Time {
	t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		return time.Time{}
	}
	return t
}
