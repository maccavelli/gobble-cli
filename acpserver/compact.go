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
	if _, err := a.modelProvider(); err != nil {
		return promptError(err)
	}
	t.say("Compacting the conversation…\n\n")
	res, used, err := a.compactSession(ctx, t.s, instructions)
	var reqErr *acp.RequestError
	switch {
	case ctx.Err() != nil:
		return context.Cause(ctx)
	case errors.Is(err, llm.ErrAuth):
		return promptError(err)
	case errors.As(err, &reqErr):
		return err
	case err != nil:
		return t.failed("Compaction failed: ", err)
	}
	t.say(fmt.Sprintf("Compacted from %d tokens.\n\n%s", res.TokensBefore, res.Summary))
	t.out.send(usedUpdate(used))
	return nil
}

// compactSession is the one compaction of /compact and _gobble/compact
// (0002-PLAN Phase 7). It writes the entry only when the summary succeeded,
// rebuilds the context from it, and returns the new estimate for the
// caller's usage_update. Pi's failures are returned as they are; a failed
// write is an *acp.RequestError.
func (a *Agent) compactSession(ctx context.Context, s *liveSession, instructions string) (compaction.Result, int, error) {
	model, err := a.modelProvider()
	if err != nil {
		return compaction.Result{}, 0, err
	}
	a.mu.Lock()
	s.compacting = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		s.compacting = false
		a.mu.Unlock()
	}()
	prep, err := compaction.Prepare(session.Path(s.log.Entries()), a.compaction)
	if err != nil {
		return compaction.Result{}, 0, err
	}
	chosen, think := a.choices(s)
	res, err := compaction.Compact(ctx, prep, compaction.Model{Provider: model, Model: chosen, Thinking: think}, instructions)
	if err != nil {
		return compaction.Result{}, 0, err
	}
	details, err := json.Marshal(res.Details)
	if err != nil {
		return compaction.Result{}, 0, acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	e := session.Entry{Type: session.TypeCompaction, Summary: res.Summary, FirstKeptEntryID: res.FirstKeptEntryID,
		TokensBefore: new(res.TokensBefore), Details: details, Usage: piUsage(res.Usage), FromHook: new(false)}
	if err := s.append(context.WithoutCancel(ctx), a.now(), e); err != nil {
		return compaction.Result{}, 0, acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	st := replayState(s.log.Entries())
	a.mu.Lock()
	s.history = st.history
	a.mu.Unlock()
	used := compaction.EstimateText(systemPrompt(s.cwd, a.tools)) + compaction.EstimateEntries(session.Path(s.log.Entries()))
	a.setUsed(s, used)
	return res, used, nil
}
