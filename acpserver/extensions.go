package acpserver

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/command"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/session"
)

// extension is one _gobble/ method (0002-PLAN Phase 7, the frozen table):
// it decodes its params and gives its result.
type extension func(a *Agent, ctx context.Context, params jsontext.Value) (any, error)

// extensions are gobble's ACP extension methods, keyed by acpext's names
// (0042-MADR D4). Every request carries sessionId; the other fields keep
// Pi's RPC names (rpc-types.ts).
var extensions = map[string]extension{
	acpext.MethodSteer:                (*Agent).extSteer,
	acpext.MethodFollowUp:             (*Agent).extFollowUp,
	acpext.MethodClearQueue:           (*Agent).extClearQueue,
	acpext.MethodSetSteeringMode:      (*Agent).extSetSteeringMode,
	acpext.MethodSetFollowUpMode:      (*Agent).extSetFollowUpMode,
	acpext.MethodCompact:              (*Agent).extCompact,
	acpext.MethodUsage:                (*Agent).extUsage,
	acpext.MethodSetSessionName:       (*Agent).extSetSessionName,
	acpext.MethodFork:                 (*Agent).extFork,
	acpext.MethodClone:                (*Agent).extClone,
	acpext.MethodGetState:             (*Agent).extGetState,
	acpext.MethodGetMessages:          (*Agent).extGetMessages,
	acpext.MethodGetEntries:           (*Agent).extGetEntries,
	acpext.MethodGetLastAssistantText: (*Agent).extGetLastAssistantText,
	acpext.MethodGetSessionStats:      (*Agent).extGetSessionStats,
	acpext.MethodGetForkMessages:      (*Agent).extGetForkMessages,
	acpext.MethodSetAutoRetry:         (*Agent).extSetAutoRetry,
	acpext.MethodAbortRetry:           (*Agent).extAbortRetry,
	acpext.MethodSetAutoCompaction:    (*Agent).extSetAutoCompaction,
	acpext.MethodSetPersonality:       (*Agent).extSetPersonality,
}

// extensionNames are the methods initialize advertises, sorted.
func extensionNames() []string { return slices.Sorted(maps.Keys(extensions)) }

var _ acp.ExtensionMethodHandler = (*Agent)(nil)

// HandleExtensionMethod answers a _gobble/ method, and any other _ method
// with -32601. A result is encoded here with encoding/json/v2, as the
// session file is, and handed to the SDK as JSON it passes through.
func (a *Agent) HandleExtensionMethod(ctx context.Context, method string, params jsontext.Value) (any, error) {
	ext, ok := extensions[method]
	if !ok {
		return nil, acp.NewMethodNotFound(method)
	}
	res, err := ext(a, ctx, params)
	if err != nil {
		return nil, err
	}
	b, err := jsonx.Marshal(res)
	if err != nil {
		return nil, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	return jsontext.Value(b), nil
}

// extSession decodes params into their acpext type, and finds the live
// session they name: bad params and an unknown session are -32602, as for
// every session method. The session is read from a second decode into
// acpext.SessionParams, since acpext's types carry no method of acpserver's
// (0042-MADR A2); the first decode's error is the one answered.
func extSession[P any](a *Agent, params jsontext.Value) (P, *liveSession, error) {
	var p P
	if len(params) == 0 {
		params = jsontext.Value("{}")
	}
	if err := jsonx.Unmarshal(params, &p); err != nil {
		return p, nil, acp.NewInvalidParams(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	var sp acpext.SessionParams
	if err := jsonx.Unmarshal(params, &sp); err != nil {
		return p, nil, acp.NewInvalidParams(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	id := acp.SessionId(sp.SessionID)
	a.mu.Lock()
	s, ok := a.sessions[id]
	a.mu.Unlock()
	if !ok {
		return p, nil, invalidParams(id, "unknown session")
	}
	return p, s, nil
}

// empty is the result of a method that answers nothing: {}.
type empty struct{}

func (a *Agent) extSteer(_ context.Context, params jsontext.Value) (any, error) {
	return a.enqueue(params, (*queue).steer)
}

func (a *Agent) extFollowUp(_ context.Context, params jsontext.Value) (any, error) {
	return a.enqueue(params, (*queue).followUpWith)
}

// enqueue queues a message's text, trimmed. gobble's own commands cannot be
// queued, as Pi refuses its agent-side ones (agent-session.ts:2183-2193):
// they are prompts.
func (a *Agent) enqueue(params jsontext.Value, add func(*queue, string)) (any, error) {
	p, s, err := extSession[acpext.MessageParams](a, params)
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(p.Message)
	if text == "" {
		return nil, invalidParams(s.id, "Message cannot be empty")
	}
	if name, _, ok := command.Parse(text); ok && slices.ContainsFunc(commands, func(c command.Command) bool { return c.Name == name }) {
		return nil, invalidParams(s.id, fmt.Sprintf("Command \"/%s\" cannot be queued. Send it as a prompt when the turn ends.", name))
	}
	// Queued as Pi queues it, expanded: a skill, then a template
	// (agent-session.ts:2106-2107; 0005-PLAN F3-3 deviation 2), with
	// the files its @path words name first, as a prompt has them
	// (F3-4 deviation 2).
	files := mentionFiles(text, s.cwd, fsx.Workspace{Roots: s.env().Roots}, map[string]bool{})
	text, _ = a.expand(s, text)
	text = files + text
	add(s.queue, text)
	// Pi's QueuedInputDisposition: gobble has no input handlers before
	// 0005-PLAN F7, so a message is always queued.
	return acpext.QueuedResult{Disposition: acpext.DispositionQueued}, nil
}

func (a *Agent) extClearQueue(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[acpext.SessionParams](a, params)
	if err != nil {
		return nil, err
	}
	steering, followUp := s.queue.clear()
	return acpext.ClearedResult{Steering: steering, FollowUp: followUp}, nil
}

func (a *Agent) extSetSteeringMode(_ context.Context, params jsontext.Value) (any, error) {
	return a.setQueueMode(params, (*queue).setSteeringMode)
}

func (a *Agent) extSetFollowUpMode(_ context.Context, params jsontext.Value) (any, error) {
	return a.setQueueMode(params, (*queue).setFollowUpMode)
}

// setQueueMode sets a queue's mode for this session; the modes are not
// persisted before the configuration-surface record.
func (a *Agent) setQueueMode(params jsontext.Value, set func(*queue, string)) (any, error) {
	p, s, err := extSession[acpext.ModeParams](a, params)
	if err != nil {
		return nil, err
	}
	if !validQueueMode(string(p.Mode)) {
		return nil, invalidParams(s.id, fmt.Sprintf("unknown queue mode %q; the modes are %s and %s", p.Mode, queueAll, queueOneAtATime))
	}
	set(s.queue, string(p.Mode))
	return empty{}, nil
}

// extCompact is /compact's compaction (compactSession), answered as Pi's
// RPC compact is: the result, or the failure as an error. It holds the
// session's turn, so session/cancel stops it, and sends usage_update but no
// message text, since it is no prompt turn.
func (a *Agent) extCompact(ctx context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[acpext.CompactParams](a, params)
	if err != nil {
		return nil, err
	}
	turnCtx, release, err := a.claimTurn(ctx, s)
	if err != nil {
		return nil, err
	}
	defer release()
	out := newUpdates(ctx, a.connection(), s.id)
	report := a.newCompactionReport(out, s, acpext.CompactionManual)
	res, used, err := a.compactSession(turnCtx, s, p.CustomInstructions, report)
	shown := report.ended(res, err, turnCtx.Err() != nil, pauseNone)
	var reqErr *acp.RequestError
	switch {
	case shown && err == nil:
		// Written: its result, a cancel after the write included, as the
		// completed row says (0016-MADR P3-Q4).
	case turnCtx.Err() != nil:
		return nil, acp.NewInternalError(errData(acpext.ErrorData{Error: "Compaction cancelled"}))
	case errors.Is(err, compaction.ErrAlreadyCompacted), errors.Is(err, compaction.ErrNothingToCompact):
		return nil, acp.NewInvalidRequest(errData(acpext.ErrorData{SessionID: string(s.id), Reason: err.Error()}))
	case errors.As(err, &reqErr):
		return nil, err
	case err != nil:
		return nil, promptError(err)
	}
	out.send(a.contextUpdate(s, used))
	if out.err != nil {
		return nil, out.err
	}
	// Pi's CompactionResult; its details are gobble's session-file shape.
	details, err := jsonx.Value(res.Details)
	if err != nil {
		return nil, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
	}
	return acpext.CompactResult{Summary: res.Summary, FirstKeptEntryID: res.FirstKeptEntryID, TokensBefore: res.TokensBefore, Details: details}, nil
}

func (a *Agent) connection() *acp.AgentSideConnection {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.conn
}

// extUsage is /usage's text, with the figures it is made from.
func (a *Agent) extUsage(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[acpext.SessionParams](a, params)
	if err != nil {
		return nil, err
	}
	entries := s.log.Entries()
	return acpext.UsageResult{Text: a.usageText(s, nil, entries), Stats: a.statsOf(s, entries)}, nil // boring: a client's text (0022-MADR D2)
}

// extSetSessionName is /name's path (setName), for a name that is given.
func (a *Agent) extSetSessionName(ctx context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[acpext.SetSessionNameParams](a, params)
	if err != nil {
		return nil, err
	}
	if oneLine(p.Name) == "" {
		return nil, invalidParams(s.id, "Session name cannot be empty")
	}
	turnCtx, release, err := a.claimTurn(ctx, s)
	if err != nil {
		return nil, err
	}
	defer release()
	if _, err := a.setName(turnCtx, s, newUpdates(ctx, a.connection(), s.id), p.Name); err != nil {
		return nil, err
	}
	return empty{}, nil
}

// extSetAutoRetry is Pi's set_auto_retry: it writes retry.enabled to the
// global settings, as Pi writes its own, and sets the session's. Turned
// off, it also ends any wait of the running turn, and every later one, at
// once; turned on, it takes effect from the session's next turn, and the
// SDK's own retries from the next session's provider (0005-PLAN F5-1
// decision 6).
func (a *Agent) extSetAutoRetry(_ context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[acpext.EnabledParams](a, params)
	if err != nil {
		return nil, err
	}
	if p.Enabled == nil {
		return nil, invalidParams(s.id, "enabled must be true or false")
	}
	if a.save != nil {
		if err := a.save("retry.enabled", strconv.FormatBool(*p.Enabled)); err != nil {
			return nil, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s.retry.Off = !*p.Enabled
	if s.retry.Off {
		a.abortRetry(s)
	}
	return empty{}, nil
}

// extSetAutoCompaction is Pi's set_auto_compaction: it writes
// compaction.enabled to the global settings, as Pi writes its own, and sets
// the session's at once, the running turn's next check included. Turned
// on, it also ends a pause of the trigger (0005-PLAN F5-2 decisions 10 and
// 11).
func (a *Agent) extSetAutoCompaction(_ context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[acpext.EnabledParams](a, params)
	if err != nil {
		return nil, err
	}
	if p.Enabled == nil {
		return nil, invalidParams(s.id, "enabled must be true or false")
	}
	if a.save != nil {
		if err := a.save("compaction.enabled", strconv.FormatBool(*p.Enabled)); err != nil {
			return nil, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s.autoCompactOff = !*p.Enabled
	if *p.Enabled {
		s.pause, s.pauseErr = pauseNone, ""
	}
	return empty{}, nil
}

// extAbortRetry is Pi's abort_retry: it ends the running turn's wait, if
// one is under way; the turn then ends with the error that caused it
// (0005-PLAN F5-1 decision 6).
func (a *Agent) extAbortRetry(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[acpext.SessionParams](a, params)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if s.waiting {
		a.abortRetry(s)
	}
	return empty{}, nil
}

// abortRetry closes the running turn's abort channel, once. The caller
// holds a.mu.
func (a *Agent) abortRetry(s *liveSession) {
	if s.retryAbort != nil {
		close(s.retryAbort)
		s.retryAbort = nil
	}
}

// extFork is /fork <entry>'s path: a new session before that user message,
// written and closed for the caller to load. Its text is the message's,
// as Pi's RPC fork gives it for the editor.
func (a *Agent) extFork(ctx context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[acpext.ForkParams](a, params)
	if err != nil {
		return nil, err
	}
	turnCtx, release, err := a.claimTurn(ctx, s)
	if err != nil {
		return nil, err
	}
	defer release()
	entries := s.log.Entries()
	target, refusal := forkTarget(entries, p.EntryID)
	if refusal != "" {
		return nil, invalidParams(s.id, refusal)
	}
	id, err := a.copySession(turnCtx, s, s.tree.Branched(*target.ParentID))
	if err != nil {
		return nil, err
	}
	return acpext.ForkResult{SessionID: string(id), Text: target.Message.Text()}, nil
}

// extClone is /clone's path: a new session holding the whole path.
func (a *Agent) extClone(ctx context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[acpext.SessionParams](a, params)
	if err != nil {
		return nil, err
	}
	turnCtx, release, err := a.claimTurn(ctx, s)
	if err != nil {
		return nil, err
	}
	defer release()
	leaf := s.tree.Leaf()
	if leaf == "" {
		return nil, invalidParams(s.id, "Cannot clone session: no current entry selected")
	}
	id, err := a.copySession(turnCtx, s, s.tree.Branched(leaf))
	if err != nil {
		return nil, err
	}
	return acpext.CloneResult{SessionID: string(id)}, nil
}

// extGetState is Pi's RpcSessionState, with gobble's mode and cwd; the
// model is named until 0005-PLAN F4's catalog gives Pi's Model.
func (a *Agent) extGetState(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[acpext.SessionParams](a, params)
	if err != nil {
		return nil, err
	}
	st := s.state()
	model, think := a.choices(s)
	steering, followUp := s.queue.modes()
	a.mu.Lock()
	streaming, compacting, auto := s.streaming, s.compacting, !s.autoCompactOff
	a.mu.Unlock()
	res := acpext.StateResult{ThinkingLevel: think, IsStreaming: streaming, IsCompacting: compacting,
		SteeringMode: acpext.QueueMode(steering), FollowUpMode: acpext.QueueMode(followUp),
		SessionFile: a.fileOf(s), SessionID: string(s.id), SessionName: st.name, MessageCount: len(contextMessages(st)),
		PendingMessageCount: s.queue.pending(), Mode: a.modeOf(s), Cwd: s.cwd, AutoCompactionEnabled: auto,
		Personality: a.voice(s).Name()}
	if model != "" {
		res.Model = &acpext.ModelRef{Provider: a.models().Provider, ID: model}
	}
	return res, nil
}

// fileOf is the session's file, or "" when the store has none.
func (a *Agent) fileOf(s *liveSession) string {
	if pf, ok := a.store.(interface{ PathOf(session.ID) string }); ok {
		return pf.PathOf(session.ID(s.id))
	}
	return ""
}

// contextMessages are the messages the model sees, in Pi's shapes, as
// buildSessionContext gives them: a compaction leads as its summary
// message, and every entry type and context edit applies (0005-PLAN F2).
func contextMessages(st state) []*session.Message {
	out := []*session.Message{}
	for _, p := range st.context {
		out = append(out, p.Messages...)
	}
	return out
}

func (a *Agent) extGetMessages(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[acpext.SessionParams](a, params)
	if err != nil {
		return nil, err
	}
	msgs, err := rawEach(contextMessages(s.state()))
	if err != nil {
		return nil, err
	}
	return acpext.MessagesResult{Messages: msgs}, nil
}

// extGetEntries is every entry in file order, or those after since, and
// the tree's leaf (Pi's getLeafId), null when there is none.
func (a *Agent) extGetEntries(_ context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[acpext.GetEntriesParams](a, params)
	if err != nil {
		return nil, err
	}
	entries := s.log.Entries()
	var res acpext.EntriesResult
	if leaf := s.tree.Leaf(); leaf != "" {
		res.LeafID = new(leaf)
	}
	if p.Since != nil {
		i := slices.IndexFunc(entries, func(e session.Entry) bool { return e.ID == *p.Since })
		if i < 0 {
			return nil, invalidParams(s.id, "Entry not found: "+*p.Since)
		}
		entries = entries[i+1:]
	}
	if res.Entries, err = rawEach(entries); err != nil {
		return nil, err
	}
	return res, nil
}

// rawEach is each of vs encoded as acpext's raw session-file shapes are,
// by jsonx, as the whole result once was; [] when there are none.
func rawEach[T any](vs []T) ([]jsontext.Value, error) {
	out := make([]jsontext.Value, 0, len(vs))
	for _, v := range vs {
		b, err := jsonx.Marshal(v)
		if err != nil {
			return nil, acp.NewInternalError(errData(acpext.ErrorData{Reason: err.Error()}))
		}
		out = append(out, b)
	}
	return out, nil
}

// extGetLastAssistantText is Pi's getLastAssistantText: the last assistant
// message in context, skipping an aborted one with no content, its text
// blocks joined and trimmed; null when there is none.
func (a *Agent) extGetLastAssistantText(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[acpext.SessionParams](a, params)
	if err != nil {
		return nil, err
	}
	for _, m := range slices.Backward(contextMessages(s.state())) {
		if m.Role != session.RoleAssistant {
			continue
		}
		blocks, err := m.Blocks()
		if err != nil || m.StopReason == piAborted && len(blocks) == 0 {
			continue
		}
		var b strings.Builder
		for _, bl := range blocks {
			if bl.Type == session.BlockText {
				b.WriteString(bl.Text)
			}
		}
		if text := strings.TrimSpace(b.String()); text != "" {
			return acpext.TextResult{Text: &text}, nil
		}
		return acpext.TextResult{}, nil
	}
	return acpext.TextResult{}, nil
}

// statsOf is /usage's totals in Pi's names, as Pi's SessionStats: cost is
// the sum of each usage's cost, and contextUsage is present when the
// model's window is known (0005-PLAN F4-2a).
func (a *Agent) statsOf(s *liveSession, entries []session.Entry) acpext.SessionStats {
	t := sessionTotals(entries)
	st := acpext.SessionStats{SessionFile: a.fileOf(s), SessionID: string(s.id), UserMessages: t.user, AssistantMessages: t.assistant,
		ToolCalls: t.toolCalls, ToolResults: t.toolResults, TotalMessages: t.messages,
		Tokens: acpext.TokenStats{Input: t.input, Output: t.output, CacheRead: t.cacheRead, CacheWrite: t.cacheWrite,
			Total: t.input + t.output + t.cacheRead + t.cacheWrite}, Cost: t.cost}
	model, _ := a.choices(s)
	if window := a.modelOf(a.provOf(s), model).ContextWindow; window > 0 {
		used := a.usedOf(s)
		st.ContextUsage = &acpext.ContextUsage{Tokens: used, ContextWindow: window, Percent: percentOf(used, window)}
	}
	return st
}

func (a *Agent) extGetSessionStats(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[acpext.SessionParams](a, params)
	if err != nil {
		return nil, err
	}
	return a.statsOf(s, s.log.Entries()), nil
}

// extGetForkMessages are the user messages with text, as /fork lists them.
func (a *Agent) extGetForkMessages(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[acpext.SessionParams](a, params)
	if err != nil {
		return nil, err
	}
	res := acpext.ForkMessagesResult{Messages: []acpext.ForkMessage{}}
	for _, e := range s.log.Entries() {
		if e.Type == session.TypeMessage && e.Message != nil && e.Message.Role == session.RoleUser {
			if text := e.Message.Text(); text != "" {
				res.Messages = append(res.Messages, acpext.ForkMessage{EntryID: e.ID, Text: text})
			}
		}
	}
	return res, nil
}
