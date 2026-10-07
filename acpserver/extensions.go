package acpserver

import (
	"context"
	stdjson "encoding/json"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/command"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/session"
)

// extension is one _gobble/ method (0002-PLAN Phase 7, the frozen table):
// it decodes its params and gives its result.
type extension func(a *Agent, ctx context.Context, params jsontext.Value) (any, error)

// extensions are gobble's ACP extension methods. Every request carries
// sessionId; the other fields keep Pi's RPC names (rpc-types.ts).
var extensions = map[string]extension{
	"_gobble/steer":                   (*Agent).extSteer,
	"_gobble/follow_up":               (*Agent).extFollowUp,
	"_gobble/clear_queue":             (*Agent).extClearQueue,
	"_gobble/set_steering_mode":       (*Agent).extSetSteeringMode,
	"_gobble/set_follow_up_mode":      (*Agent).extSetFollowUpMode,
	"_gobble/compact":                 (*Agent).extCompact,
	"_gobble/usage":                   (*Agent).extUsage,
	"_gobble/set_session_name":        (*Agent).extSetSessionName,
	"_gobble/fork":                    (*Agent).extFork,
	"_gobble/clone":                   (*Agent).extClone,
	"_gobble/get_state":               (*Agent).extGetState,
	"_gobble/get_messages":            (*Agent).extGetMessages,
	"_gobble/get_entries":             (*Agent).extGetEntries,
	"_gobble/get_last_assistant_text": (*Agent).extGetLastAssistantText,
	"_gobble/get_session_stats":       (*Agent).extGetSessionStats,
	"_gobble/get_fork_messages":       (*Agent).extGetForkMessages,
}

// extensionNames are the methods initialize advertises, sorted.
func extensionNames() []string { return slices.Sorted(maps.Keys(extensions)) }

var _ acp.ExtensionMethodHandler = (*Agent)(nil)

// HandleExtensionMethod answers a _gobble/ method, and any other _ method
// with -32601. A result is encoded here with encoding/json/v2, as the
// session file is, and handed to the SDK as JSON it passes through.
func (a *Agent) HandleExtensionMethod(ctx context.Context, method string, params stdjson.RawMessage) (any, error) {
	ext, ok := extensions[method]
	if !ok {
		return nil, acp.NewMethodNotFound(method)
	}
	res, err := ext(a, ctx, params)
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(res)
	if err != nil {
		return nil, acp.NewInternalError(map[string]any{keyReason: err.Error()})
	}
	return stdjson.RawMessage(b), nil
}

// sessionParams is every request's sessionId.
type sessionParams struct {
	SessionID acp.SessionId `json:"sessionId"`
}

func (p sessionParams) session() acp.SessionId { return p.SessionID }

type messageParams struct {
	sessionParams `json:",inline"`
	Message       string `json:"message"`
}

type modeParams struct {
	sessionParams `json:",inline"`
	Mode          string `json:"mode"`
}

type compactParams struct {
	sessionParams      `json:",inline"`
	CustomInstructions string `json:"customInstructions"`
}

type nameParams struct {
	sessionParams `json:",inline"`
	Name          string `json:"name"`
}

type forkParams struct {
	sessionParams `json:",inline"`
	EntryID       string `json:"entryId"`
}

type entriesParams struct {
	sessionParams `json:",inline"`
	Since         *string `json:"since"`
}

// extSession decodes params and finds their live session: bad params and
// an unknown session are -32602, as for every session method.
func extSession[P interface{ session() acp.SessionId }](a *Agent, params jsontext.Value) (P, *liveSession, error) {
	var p P
	if len(params) == 0 {
		params = jsontext.Value("{}")
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return p, nil, acp.NewInvalidParams(map[string]any{keyReason: err.Error()})
	}
	a.mu.Lock()
	s, ok := a.sessions[p.session()]
	a.mu.Unlock()
	if !ok {
		return p, nil, invalidParams(p.session(), "unknown session")
	}
	return p, s, nil
}

// empty is the result of a method that answers nothing: {}.
type empty struct{}

// queuedResult is Pi's QueuedInputDisposition: gobble has no input
// handlers before 0005-PLAN F7, so a message is always queued.
type queuedResult struct {
	Disposition string `json:"disposition"`
}

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
	p, s, err := extSession[messageParams](a, params)
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
	add(s.queue, text)
	return queuedResult{Disposition: "queued"}, nil
}

type clearedResult struct {
	Steering []string `json:"steering"`
	FollowUp []string `json:"followUp"`
}

func (a *Agent) extClearQueue(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[sessionParams](a, params)
	if err != nil {
		return nil, err
	}
	steering, followUp := s.queue.clear()
	return clearedResult{Steering: steering, FollowUp: followUp}, nil
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
	p, s, err := extSession[modeParams](a, params)
	if err != nil {
		return nil, err
	}
	if !validQueueMode(p.Mode) {
		return nil, invalidParams(s.id, fmt.Sprintf("unknown queue mode %q; the modes are %s and %s", p.Mode, queueAll, queueOneAtATime))
	}
	set(s.queue, p.Mode)
	return empty{}, nil
}

// compactResult is Pi's CompactionResult.
type compactResult struct {
	Summary          string             `json:"summary"`
	FirstKeptEntryID string             `json:"firstKeptEntryId"`
	TokensBefore     int64              `json:"tokensBefore"`
	Details          compaction.Details `json:"details"`
}

// extCompact is /compact's compaction (compactSession), answered as Pi's
// RPC compact is: the result, or the failure as an error. It holds the
// session's turn, so session/cancel stops it, and sends usage_update but no
// message text, since it is no prompt turn.
func (a *Agent) extCompact(ctx context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[compactParams](a, params)
	if err != nil {
		return nil, err
	}
	turnCtx, release, err := a.claimTurn(ctx, s)
	if err != nil {
		return nil, err
	}
	defer release()
	res, used, err := a.compactSession(turnCtx, s, p.CustomInstructions)
	var reqErr *acp.RequestError
	switch {
	case turnCtx.Err() != nil:
		return nil, acp.NewInternalError(map[string]any{keyError: "Compaction cancelled"})
	case errors.Is(err, compaction.ErrAlreadyCompacted), errors.Is(err, compaction.ErrNothingToCompact):
		return nil, acp.NewInvalidRequest(map[string]any{keySessionID: s.id, keyReason: err.Error()})
	case errors.As(err, &reqErr):
		return nil, err
	case err != nil:
		return nil, promptError(err)
	}
	out := newUpdates(ctx, a.connection(), s.id)
	out.send(usedUpdate(used))
	if out.err != nil {
		return nil, out.err
	}
	return compactResult{Summary: res.Summary, FirstKeptEntryID: res.FirstKeptEntryID, TokensBefore: res.TokensBefore, Details: res.Details}, nil
}

func (a *Agent) connection() *acp.AgentSideConnection {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.conn
}

type usageResult struct {
	Text  string       `json:"text"`
	Stats sessionStats `json:"stats"`
}

// extUsage is /usage's text, with the figures it is made from.
func (a *Agent) extUsage(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[sessionParams](a, params)
	if err != nil {
		return nil, err
	}
	entries := s.log.Entries()
	return usageResult{Text: a.usageText(entries), Stats: a.statsOf(s, entries)}, nil
}

// extSetSessionName is /name's path (setName), for a name that is given.
func (a *Agent) extSetSessionName(ctx context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[nameParams](a, params)
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

type forkResult struct {
	SessionID session.ID `json:"sessionId"`
	Text      string     `json:"text,omitzero"`
}

// extFork is /fork <entry>'s path: a new session before that user message,
// written and closed for the caller to load. Its text is the message's,
// as Pi's RPC fork gives it for the editor.
func (a *Agent) extFork(ctx context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[forkParams](a, params)
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
	id, err := a.copySession(turnCtx, s, pathTo(entries, *target.ParentID))
	if err != nil {
		return nil, err
	}
	return forkResult{SessionID: id, Text: target.Message.Text()}, nil
}

// extClone is /clone's path: a new session holding the whole path.
func (a *Agent) extClone(ctx context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[sessionParams](a, params)
	if err != nil {
		return nil, err
	}
	turnCtx, release, err := a.claimTurn(ctx, s)
	if err != nil {
		return nil, err
	}
	defer release()
	entries := s.log.Entries()
	if len(entries) == 0 {
		return nil, invalidParams(s.id, "Cannot clone session: no current entry selected")
	}
	id, err := a.copySession(turnCtx, s, session.Path(entries))
	if err != nil {
		return nil, err
	}
	return forkResult{SessionID: id}, nil
}

// modelRef is the model get_state names, until 0005-PLAN F4's catalog gives
// Pi's Model.
type modelRef struct {
	Provider string `json:"provider,omitzero"`
	ID       string `json:"id"`
}

// stateResult is Pi's RpcSessionState, with gobble's mode and cwd.
type stateResult struct {
	Model                 *modelRef     `json:"model,omitzero"`
	ThinkingLevel         string        `json:"thinkingLevel"`
	IsStreaming           bool          `json:"isStreaming"`
	IsCompacting          bool          `json:"isCompacting"`
	SteeringMode          string        `json:"steeringMode"`
	FollowUpMode          string        `json:"followUpMode"`
	SessionFile           string        `json:"sessionFile,omitzero"`
	SessionID             acp.SessionId `json:"sessionId"`
	SessionName           string        `json:"sessionName,omitzero"`
	AutoCompactionEnabled bool          `json:"autoCompactionEnabled"`
	MessageCount          int           `json:"messageCount"`
	PendingMessageCount   int           `json:"pendingMessageCount"`
	Mode                  string        `json:"mode"`
	Cwd                   string        `json:"cwd"`
}

func (a *Agent) extGetState(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[sessionParams](a, params)
	if err != nil {
		return nil, err
	}
	st := replayState(s.log.Entries())
	model, think := a.choices(s)
	steering, followUp := s.queue.modes()
	a.mu.Lock()
	streaming, compacting := s.streaming, s.compacting
	a.mu.Unlock()
	res := stateResult{ThinkingLevel: think, IsStreaming: streaming, IsCompacting: compacting, SteeringMode: steering, FollowUpMode: followUp,
		SessionFile: a.fileOf(s), SessionID: s.id, SessionName: st.name, MessageCount: len(contextMessages(st)),
		PendingMessageCount: s.queue.pending(), Mode: a.modeOf(s), Cwd: s.cwd}
	if model != "" {
		res.Model = &modelRef{Provider: a.models().Provider, ID: model}
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

// compactionSummary is Pi's CompactionSummaryMessage, the message a
// compaction is in context.
type compactionSummary struct {
	Role         string `json:"role"`
	Summary      string `json:"summary"`
	TokensBefore int64  `json:"tokensBefore"`
	Timestamp    int64  `json:"timestamp"`
}

// contextMessages are the messages the model sees, in Pi's shapes: a
// compaction leads as its summary message (buildSessionContext).
func contextMessages(st state) []any {
	out := []any{}
	for i, e := range st.context {
		switch {
		case e.Type == session.TypeCompaction && i == 0:
			var before int64
			if e.TokensBefore != nil {
				before = *e.TokensBefore
			}
			out = append(out, compactionSummary{Role: "compactionSummary", Summary: e.Summary, TokensBefore: before, Timestamp: ms(e.Timestamp)})
		case e.Type == session.TypeMessage && e.Message != nil:
			out = append(out, e.Message)
		}
	}
	return out
}

type messagesResult struct {
	Messages []any `json:"messages"`
}

func (a *Agent) extGetMessages(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[sessionParams](a, params)
	if err != nil {
		return nil, err
	}
	return messagesResult{Messages: contextMessages(replayState(s.log.Entries()))}, nil
}

type entriesResult struct {
	Entries []session.Entry `json:"entries"`
	LeafID  *string         `json:"leafId"`
}

// extGetEntries is every entry in file order, or those after since; the
// leaf is the last entry (session.Path).
func (a *Agent) extGetEntries(_ context.Context, params jsontext.Value) (any, error) {
	p, s, err := extSession[entriesParams](a, params)
	if err != nil {
		return nil, err
	}
	entries := s.log.Entries()
	res := entriesResult{Entries: []session.Entry{}}
	if n := len(entries); n > 0 {
		res.LeafID = new(entries[n-1].ID)
	}
	if p.Since != nil {
		i := slices.IndexFunc(entries, func(e session.Entry) bool { return e.ID == *p.Since })
		if i < 0 {
			return nil, invalidParams(s.id, "Entry not found: "+*p.Since)
		}
		entries = entries[i+1:]
	}
	res.Entries = append(res.Entries, entries...)
	return res, nil
}

type textResult struct {
	Text *string `json:"text"`
}

// extGetLastAssistantText is Pi's getLastAssistantText: the last assistant
// message in context, skipping an aborted one with no content, its text
// blocks joined and trimmed; null when there is none.
func (a *Agent) extGetLastAssistantText(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[sessionParams](a, params)
	if err != nil {
		return nil, err
	}
	st := replayState(s.log.Entries())
	for _, e := range slices.Backward(st.context) {
		m := e.Message
		if e.Type != session.TypeMessage || m == nil || m.Role != session.RoleAssistant {
			continue
		}
		blocks, err := m.Blocks()
		if err != nil || m.StopReason == "aborted" && len(blocks) == 0 {
			continue
		}
		var b strings.Builder
		for _, bl := range blocks {
			if bl.Type == session.BlockText {
				b.WriteString(bl.Text)
			}
		}
		if text := strings.TrimSpace(b.String()); text != "" {
			return textResult{Text: &text}, nil
		}
		return textResult{}, nil
	}
	return textResult{}, nil
}

// sessionStats is Pi's SessionStats. cost is 0 and contextUsage absent
// until 0005-PLAN F4's catalog gives prices and a window.
type sessionStats struct {
	SessionFile       string        `json:"sessionFile,omitzero"`
	SessionID         acp.SessionId `json:"sessionId"`
	UserMessages      int           `json:"userMessages"`
	AssistantMessages int           `json:"assistantMessages"`
	ToolCalls         int           `json:"toolCalls"`
	ToolResults       int           `json:"toolResults"`
	TotalMessages     int           `json:"totalMessages"`
	Tokens            statsTokens   `json:"tokens"`
	Cost              float64       `json:"cost"`
}

type statsTokens struct {
	Input      int64 `json:"input"`
	Output     int64 `json:"output"`
	CacheRead  int64 `json:"cacheRead"`
	CacheWrite int64 `json:"cacheWrite"`
	Total      int64 `json:"total"`
}

// statsOf is /usage's totals in Pi's names.
func (a *Agent) statsOf(s *liveSession, entries []session.Entry) sessionStats {
	t := sessionTotals(entries)
	return sessionStats{SessionFile: a.fileOf(s), SessionID: s.id, UserMessages: t.user, AssistantMessages: t.assistant,
		ToolCalls: t.toolCalls, ToolResults: t.toolResults, TotalMessages: t.messages,
		Tokens: statsTokens{Input: t.input, Output: t.output, CacheRead: t.cacheRead, CacheWrite: t.cacheWrite,
			Total: t.input + t.output + t.cacheRead + t.cacheWrite}}
}

func (a *Agent) extGetSessionStats(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[sessionParams](a, params)
	if err != nil {
		return nil, err
	}
	return a.statsOf(s, s.log.Entries()), nil
}

type forkMessage struct {
	EntryID string `json:"entryId"`
	Text    string `json:"text"`
}

type forkMessagesResult struct {
	Messages []forkMessage `json:"messages"`
}

// extGetForkMessages are the user messages with text, as /fork lists them.
func (a *Agent) extGetForkMessages(_ context.Context, params jsontext.Value) (any, error) {
	_, s, err := extSession[sessionParams](a, params)
	if err != nil {
		return nil, err
	}
	res := forkMessagesResult{Messages: []forkMessage{}}
	for _, e := range s.log.Entries() {
		if e.Type == session.TypeMessage && e.Message != nil && e.Message.Role == session.RoleUser {
			if text := e.Message.Text(); text != "" {
				res.Messages = append(res.Messages, forkMessage{EntryID: e.ID, Text: text})
			}
		}
	}
	return res, nil
}
