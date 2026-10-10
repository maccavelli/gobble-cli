package acpserver

import (
	"context"
	"fmt"
	"slices"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/session"
)

// Config option ids, which are also their categories (0008-MADR D19 item 4).
const (
	configModel    = "model"
	configThinking = "thought_level"
)

// thinkingOff is the level of a session that asks for no reasoning.
const thinkingOff = "off"

// thinkingLevels are the levels thought_level offers, gobble's --thinking
// levels with off first.
var thinkingLevels = []string{thinkingOff, "minimal", "low", "medium", "high", "xhigh"}

// ModelChoice is the models a session may choose from: the provider's id,
// its curated models, and the default. A zero ModelChoice offers no model
// option.
type ModelChoice struct {
	Provider string
	Models   []string
	Default  string
}

// configOptions are a session's config options: model, when there are
// models to choose, and thought_level. Both are ungrouped selects with a
// category (D19 item 4).
func configOptions(choice ModelChoice, model, thinking string) []acp.SessionConfigOption {
	var out []acp.SessionConfigOption
	if len(choice.Models) > 0 {
		out = append(out, selectOption(configModel, "Model", acp.SessionConfigOptionCategoryModel, model, choice.Models))
	}
	return append(out, selectOption(configThinking, "Thinking", acp.SessionConfigOptionCategoryThoughtLevel, thinking, thinkingLevels))
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

func validThinking(level string) bool { return slices.Contains(thinkingLevels, level) }

// setConfig sets the model or the thinking level from the session's next
// prompt and records the change (model_change, thinking_level_change)
// before anything reports it. It is the one path of set_config_option,
// /model and /thinking.
func (a *Agent) setConfig(ctx context.Context, s *liveSession, id, value string) ([]acp.SessionConfigOption, error) {
	choice := a.models()
	var e session.Entry
	switch id {
	case configModel:
		if !slices.Contains(choice.Models, value) {
			return nil, invalidParams(s.id, fmt.Sprintf("unknown model %q", value))
		}
		a.mu.Lock()
		s.model = value
		a.mu.Unlock()
		e = session.Entry{Type: session.TypeModelChange, Provider: choice.Provider, ModelID: value}
	case configThinking:
		if !validThinking(value) {
			return nil, invalidParams(s.id, fmt.Sprintf("unknown thinking level %q", value))
		}
		a.mu.Lock()
		s.think = value
		a.mu.Unlock()
		e = session.Entry{Type: session.TypeThinkingLevelChange, ThinkingLevel: value}
	default:
		return nil, invalidParams(s.id, fmt.Sprintf("unknown config option %q", id))
	}
	if err := s.append(ctx, a.now(), e); err != nil {
		return nil, appendFailed(err)
	}
	model, think := a.choices(s)
	return configOptions(choice, model, think), nil
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
