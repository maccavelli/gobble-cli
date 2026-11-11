package acpserver

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/session"
)

// pauseNone is no pause of the automatic trigger: acpext.CompactionPause's
// zero value, beside its PauseTurn and PauseSession (0005-PLAN F5-2
// decision 11).
const pauseNone acpext.CompactionPause = ""

// maxTurnCompactions bounds a turn's compactions, at the threshold and on
// overflow together (Kilo's; 0005-PLAN F5-2 decision 4).
const maxTurnCompactions = 3

var errCompactionsExhausted = fmt.Errorf("compaction did not bring the context under the model's window after %d attempts", maxTurnCompactions)

// overflowFailed ends a turn whose overflow could not be compacted
// (0005-PLAN F5-2 decision 5).
func overflowFailed(err error) error {
	return fmt.Errorf("the context is over the model's window, and compacting it failed: %w", err)
}

// pauseOf is the pause a failed automatic compaction takes: the session's
// when asking again with the same history and model cannot do better, the
// turn's otherwise (0005-PLAN F5-2 decision 11).
func pauseOf(err error) acpext.CompactionPause {
	if errors.Is(err, llm.ErrContextOverflow) || errors.Is(err, compaction.ErrSummaryRefused) {
		return acpext.PauseSession
	}
	return acpext.PauseTurn
}

// autoCompaction is one turn's agent.Config.Compact (0005-PLAN F5-2
// decisions 1, 2, 4, 5, 11 and 12).
type autoCompaction struct {
	a          *Agent
	s          *liveSession
	provider   llm.Provider
	providerID string
	out        *updates
	// done counts the turn's compactions; compacted says there was one.
	done      int
	compacted bool
}

// compact compacts the session before a model call, when the estimate is
// over a threshold, or after the call overflowed. A threshold compaction
// that Prepare refuses, or that fails, lets the call go ahead; an overflow
// that cannot be compacted ends the turn.
func (c *autoCompaction) compact(ctx context.Context, overflow bool) (*agent.Compacted, error) {
	a, s := c.a, c.s
	a.mu.Lock()
	off, pause, why, settings := s.autoCompactOff, s.pause, s.pauseErr, s.compaction
	a.mu.Unlock()
	if off {
		return nil, nil
	}
	if a.planGated(s, c.providerID) {
		// No automatic compaction under a plan session without consent; an
		// overflow ends the turn and says how to go on (0031-MADR D7,
		// amended 2026-11-10).
		if overflow {
			return nil, errPlanOverflow
		}
		return nil, nil
	}
	reason := acpext.CompactionThreshold
	if overflow {
		reason = acpext.CompactionOverflow
		if pause == acpext.PauseSession {
			return nil, fmt.Errorf("the context is over the model's window, and automatic compaction is paused after it failed: %s; run /compact, or change the model", why)
		}
	} else {
		if pause != pauseNone {
			return nil, nil
		}
		chosen, _ := a.choices(s)
		used := compaction.EstimateText(a.systemText(s)) + compaction.EstimateEntries(s.tree.Path())
		if !compaction.Over(used, a.modelOf(c.providerID, chosen).ContextWindow, settings) {
			return nil, nil
		}
	}
	if c.done == maxTurnCompactions {
		return nil, errCompactionsExhausted
	}
	prep, _, err := a.prepareCompaction(s)
	switch {
	case err != nil && overflow:
		return nil, overflowFailed(err)
	case err != nil:
		return nil, nil //nolint:nilerr // at the threshold, a session Prepare refuses is left as it is, and the call goes ahead (decision 5)
	}
	c.out.flush()
	report := a.newCompactionReport(c.out, s, reason)
	res, used, err := a.runCompaction(ctx, s, prep, a.summaryModel(s, c.provider, c.providerID), "", report)
	if err != nil {
		if ctx.Err() != nil {
			report.ended(res, err, true, pauseNone)
			return nil, context.Cause(ctx)
		}
		p := pauseOf(err)
		a.mu.Lock()
		s.pause, s.pauseErr = p, err.Error()
		a.mu.Unlock()
		report.ended(res, err, false, p)
		if overflow {
			return nil, overflowFailed(err)
		}
		return nil, nil
	}
	c.done++
	c.compacted = true
	report.ended(res, nil, false, pauseNone)
	c.out.send(a.contextUpdate(s, used))
	a.mu.Lock()
	history := slices.Clone(s.history)
	a.mu.Unlock()
	return &agent.Compacted{History: history, System: a.systemText(s)}, nil
}

// summaryModel is the model an automatic compaction asks for its summary
// (0005-PLAN F5-2 decision 12): the model that wrote the context's last
// assistant message, when it is another of the session's provider's that
// the provider still offers, since it can read the context the session's
// model may not; otherwise the session's model.
func (a *Agent) summaryModel(s *liveSession, p llm.Provider, providerID string) compaction.Model {
	chosen, think := a.choices(s)
	m := compaction.Model{Provider: p, Model: chosen, Thinking: think, MaxTokens: a.modelOf(providerID, chosen).MaxTokens}
	prev, provider := lastAssistantModel(s.tree.Path())
	if prev != "" && prev != chosen && provider == providerID && slices.Contains(a.modelsOf(providerID), prev) {
		m.Model, m.Thinking, m.MaxTokens = prev, catalog.Clamp(a.levelsOf(providerID, prev), think), a.modelOf(providerID, prev).MaxTokens
	}
	return m
}

// lastAssistantModel is the model and provider of path's last assistant
// message, or "" when it has none.
func lastAssistantModel(path []session.Entry) (model, provider string) {
	for _, e := range slices.Backward(path) {
		if e.Type == session.TypeMessage && e.Message != nil && e.Message.Role == session.RoleAssistant {
			return e.Message.Model, e.Message.Provider
		}
	}
	return "", ""
}
