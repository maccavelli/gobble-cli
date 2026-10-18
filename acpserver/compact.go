package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"strconv"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/personality"
	"github.com/maccavelli/gobble-cli/session"
)

// cmdCompact is /compact [instructions]: Pi's manual compaction (0002-PLAN
// Phase 6, owner's decision of 2026-10-06). It streams its progress,
// writes Pi's compaction entry only when the summary succeeded, rebuilds
// the context from it, and sends the new estimate as usage_update.
func (a *Agent) cmdCompact(ctx context.Context, t *cmdTurn, instructions string) error {
	if _, err := a.sessionProvider(t.s); err != nil {
		return promptError(err)
	}
	t.say(t.text(personality.CompactStart, nil) + "\n\n")
	t.out.notify(methodCompaction, compactionNotice{SessionID: t.s.id, Phase: phaseStart, Reason: reasonManual})
	res, used, err := a.compactSession(ctx, t.s, instructions)
	t.out.notify(methodCompaction, manualEnd(ctx, t.s.id, res, err))
	var reqErr *acp.RequestError
	switch {
	case ctx.Err() != nil:
		return context.Cause(ctx)
	case errors.Is(err, llm.ErrAuth):
		return promptError(err)
	case errors.As(err, &reqErr):
		return err
	case err != nil:
		// A command's failure is its answer, not a JSON-RPC error: the turn
		// ends normally, as Pi shows the failure and goes on.
		t.say(t.text(personality.CompactFailed, personality.Args{argError: err.Error()}))
		return nil //nolint:nilerr // the failure is the command's answer (above)
	}
	t.say(t.text(personality.CompactDone, personality.Args{argTokens: strconv.FormatInt(res.TokensBefore, 10)}) + "\n\n" + res.Summary)
	t.out.send(a.contextUpdate(t.s, used))
	return nil
}

// manualEnd is a manual compaction's _gobble/compaction end (0005-PLAN F5-2
// decision 9): its tokens before, or its error.
func manualEnd(ctx context.Context, id acp.SessionId, res compaction.Result, err error) compactionNotice {
	n := compactionNotice{SessionID: id, Phase: phaseEnd, Reason: reasonManual}
	switch {
	case ctx.Err() != nil:
		n.Error = "cancelled"
	case err != nil:
		n.Error = err.Error()
	default:
		n.TokensBefore = res.TokensBefore
	}
	return n
}

// compactSession is /compact's and _gobble/compact's compaction (0002-PLAN
// Phase 7), with the session's model: prepareCompaction, then
// runCompaction. Pi's failures are returned as they are; a failed write
// is an *acp.RequestError.
func (a *Agent) compactSession(ctx context.Context, s *liveSession, instructions string) (compaction.Result, int, error) {
	prep, model, err := a.prepareCompaction(s)
	if err != nil {
		return compaction.Result{}, 0, err
	}
	chosen, think := a.choices(s)
	m := compaction.Model{Provider: model, Model: chosen, Thinking: think, MaxTokens: a.modelOf(chosen).MaxTokens}
	return a.runCompaction(ctx, s, prep, m, instructions)
}

// prepareCompaction finds what a compaction of s summarises, once its file
// can be written and its provider is built.
func (a *Agent) prepareCompaction(s *liveSession) (*compaction.Preparation, llm.Provider, error) {
	if w, ok := s.log.(interface{ Writable() error }); ok {
		if err := w.Writable(); err != nil { // refused before the summary is asked for
			return nil, nil, appendFailed(err)
		}
	}
	model, err := a.sessionProvider(s)
	if err != nil {
		return nil, nil, err
	}
	prep, err := compaction.Prepare(s.tree.Path(), s.compaction)
	if err != nil {
		return nil, nil, err
	}
	return prep, model, nil
}

// runCompaction asks m for prep's summary and records it: Pi's compaction
// entry only when the summary succeeded, after a new system prompt
// baseline; the context rebuilt from it; and the new estimate, returned
// for the caller's usage_update. A success ends any pause of the automatic
// trigger (0005-PLAN F5-2 decision 11).
func (a *Agent) runCompaction(ctx context.Context, s *liveSession, prep *compaction.Preparation, m compaction.Model, instructions string) (compaction.Result, int, error) {
	a.mu.Lock()
	s.compacting = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		s.compacting = false
		a.mu.Unlock()
	}()
	res, err := compaction.Compact(ctx, prep, m, instructions)
	if err != nil {
		return compaction.Result{}, 0, err
	}
	details, err := json.Marshal(res.Details)
	if err != nil {
		return compaction.Result{}, 0, acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	e := session.Entry{Type: session.TypeCompaction, Summary: res.Summary, FirstKeptEntryID: res.FirstKeptEntryID,
		TokensBefore: new(res.TokensBefore), Details: details, Usage: pricedUsage(res.Usage, a.modelOf(m.Model)), FromHook: new(false)}
	// The new baseline goes just before the compaction, which stays the
	// path's last entry, as Prepare's "Already compacted" reads it (0005-PLAN
	// F3-1 deviation 3).
	write := func(e session.Entry) error { return s.append(context.WithoutCancel(ctx), a.now(), e) }
	if err := a.rebase(s, write); err != nil {
		return compaction.Result{}, 0, appendFailed(err)
	}
	if err := write(e); err != nil {
		return compaction.Result{}, 0, appendFailed(err)
	}
	st := s.state()
	a.mu.Lock()
	s.history = st.history
	s.pause, s.pauseErr = pauseNone, ""
	a.mu.Unlock()
	used := compaction.EstimateText(a.systemText(s)) + compaction.EstimateEntries(s.tree.Path())
	a.setUsed(s, used)
	return res, used, nil
}
