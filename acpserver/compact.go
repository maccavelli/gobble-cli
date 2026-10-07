package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/session"
)

// cmdCompact is /compact [instructions]: Pi's manual compaction (0002-PLAN
// Phase 6, owner's decision of 2026-10-06). It streams its progress,
// writes Pi's compaction entry only when the summary succeeded, rebuilds
// the context from it, and sends the new estimate as usage_update.
func (a *Agent) cmdCompact(ctx context.Context, t *cmdTurn, instructions string) error {
	model, err := a.modelProvider()
	if err != nil {
		return promptError(err)
	}
	s := t.s
	t.say("Compacting the conversation…\n\n")
	prep, err := compaction.Prepare(session.Path(s.log.Entries()), a.compaction)
	if err != nil {
		return t.failed("Compaction failed: ", err)
	}
	res, err := compaction.Compact(ctx, prep, compaction.Model{Provider: model, Model: s.model, Thinking: s.think}, instructions)
	switch {
	case ctx.Err() != nil:
		return context.Cause(ctx)
	case errors.Is(err, llm.ErrAuth):
		return promptError(err)
	case err != nil:
		return t.failed("Compaction failed: ", err)
	}
	details, err := json.Marshal(res.Details)
	if err != nil {
		return acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	e := session.Entry{Type: session.TypeCompaction, Summary: res.Summary, FirstKeptEntryID: res.FirstKeptEntryID,
		TokensBefore: new(res.TokensBefore), Details: details, Usage: piUsage(res.Usage), FromHook: new(false)}
	if err := s.append(context.WithoutCancel(ctx), a.now(), e); err != nil {
		return acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	st := replayState(s.log.Entries())
	a.mu.Lock()
	s.history = st.history
	a.mu.Unlock()
	t.say(fmt.Sprintf("Compacted from %d tokens.\n\n%s", res.TokensBefore, res.Summary))
	used := compaction.EstimateText(systemPrompt(s.cwd, a.tools)) + compaction.EstimateEntries(session.Path(s.log.Entries()))
	a.setUsed(s, used)
	t.out.send(usedUpdate(used))
	return nil
}
