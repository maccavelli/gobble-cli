package acpserver

import (
	"context"
	"errors"
	"strconv"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/personality"
	"github.com/maccavelli/gobble-cli/session"
)

// cmdCompact is /compact [instructions]: Pi's manual compaction (0002-PLAN
// Phase 6, owner's decision of 2026-10-06). It streams its progress,
// writes Pi's compaction entry only when the summary succeeded, rebuilds
// the context from it, and sends the new estimate as usage_update. To a
// client that shows compactions as rows of their own, it says nothing of
// a compaction that ran: the row carries the summary or the error, and a
// compaction written before a cancel still sends its estimate (0016-MADR
// P3-Q4).
func (a *Agent) cmdCompact(ctx context.Context, t *cmdTurn, instructions string) error {
	if _, err := a.sessionProvider(t.s); err != nil {
		return promptError(err)
	}
	report := a.newCompactionReport(t.out, t.s, acpext.CompactionManual)
	if !report.rows() {
		t.say(t.text(personality.CompactStart, nil) + "\n\n")
	}
	res, used, err := a.compactSession(ctx, t.s, instructions, report)
	shown := report.ended(res, err, ctx.Err() != nil, pauseNone)
	var reqErr *acp.RequestError
	switch {
	case shown && err == nil:
		t.out.send(a.contextUpdate(t.s, used))
		return context.Cause(ctx)
	case ctx.Err() != nil:
		return context.Cause(ctx)
	case errors.Is(err, llm.ErrAuth):
		return promptError(err)
	case errors.As(err, &reqErr):
		return err
	case shown: // the failed row is the command's answer
		return nil
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

// compactSession is /compact's and _gobble/compact's compaction (0002-PLAN
// Phase 7), with the session's model: prepareCompaction, then
// runCompaction, which reports its start to report. Pi's failures are
// returned as they are; a failed write is an *acp.RequestError.
func (a *Agent) compactSession(ctx context.Context, s *liveSession, instructions string, report compactionReport) (compaction.Result, int, error) {
	prep, model, err := a.prepareCompaction(s)
	if err != nil {
		return compaction.Result{}, 0, err
	}
	chosen, think := a.choices(s)
	m := compaction.Model{Provider: model, Model: chosen, Thinking: think, MaxTokens: a.modelOf(a.provOf(s), chosen).MaxTokens}
	return a.runCompaction(ctx, s, prep, m, instructions, report)
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
// trigger (0005-PLAN F5-2 decision 11). The entry's id is reserved before
// the summary is asked for, and report told it, so the client names the
// compaction by the id the entry is written with and replayed by
// (0016-MADR P3-Q1); the caller reports the end.
func (a *Agent) runCompaction(ctx context.Context, s *liveSession, prep *compaction.Preparation, m compaction.Model, instructions string,
	report compactionReport) (compaction.Result, int, error) {
	a.mu.Lock()
	s.compacting = true
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		s.compacting = false
		a.mu.Unlock()
	}()
	id := s.tree.Reserve()
	defer s.tree.Release(id) // nothing once the entry has spent it
	report.started(id)
	res, err := compaction.Compact(ctx, prep, m, compaction.Request{Instructions: instructions, PlanMode: a.modeOf(s) == modePlan,
		Prompts: a.compactionPromptsOf(s)})
	if err != nil {
		return compaction.Result{}, 0, err
	}
	details, err := jsonx.Marshal(res.Details)
	if err != nil {
		return compaction.Result{}, 0, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	e := session.Entry{Type: session.TypeCompaction, ID: id, Summary: res.Summary, FirstKeptEntryID: res.FirstKeptEntryID,
		TokensBefore: new(res.TokensBefore), Details: details, Usage: pricedUsage(res.Usage, a.modelOf(a.provOf(s), m.Model)), FromHook: new(false)}
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
