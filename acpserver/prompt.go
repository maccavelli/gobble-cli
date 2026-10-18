package acpserver

import (
	"context"
	stdjson "encoding/json"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
)

const (
	// maxTitle and maxSummary are the phone-legible bounds of a tool call's
	// title, in runes, and its first content block, in bytes (0008-MADR D19
	// item 3).
	maxTitle   = 60
	maxSummary = 400

	allowOnce  acp.PermissionOptionId = "allow_once"
	rejectOnce acp.PermissionOptionId = "reject_once"
)

// updates sends one turn's session updates. Text and thoughts are
// coalesced; anything else flushes both first, so the order on the wire is
// the order of events.
type updates struct {
	conn    *acp.AgentSideConnection
	ctx     context.Context
	session acp.SessionId
	text    *coalescer
	thought *coalescer
	err     error
	// tools and env name a call's locations at tool_call (tool.Locator).
	tools map[string]tool.Tool
	env   tool.Env
	// running is the call running now, and terms the client terminal each
	// call created, which its updates carry (0005-PLAN F1d-3).
	running acp.ToolCallId
	terms   map[acp.ToolCallId]string
}

func newUpdates(ctx context.Context, conn *acp.AgentSideConnection, id acp.SessionId) *updates {
	u := &updates{conn: conn, ctx: context.WithoutCancel(ctx), session: id, terms: map[acp.ToolCallId]string{}}
	u.text = newCoalescer(func(s string) { u.send(acp.UpdateAgentMessageText(s)) })
	u.thought = newCoalescer(func(s string) { u.send(acp.UpdateAgentThoughtText(s)) })
	return u
}

func (u *updates) send(su acp.SessionUpdate) {
	if u.err != nil || u.conn == nil {
		return
	}
	u.err = u.conn.SessionUpdate(u.ctx, acp.SessionNotification{SessionId: u.session, Update: su})
}

// notify sends an extension notification about the session, as send sends
// an update: once one fails, nothing more is sent.
func (u *updates) notify(method string, params any) {
	if u.err != nil || u.conn == nil {
		return
	}
	u.err = u.conn.NotifyExtension(u.ctx, method, params)
}

// retryNotice is _gobble/retry's params (0005-PLAN F5-1 decision 5).
type retryNotice struct {
	SessionID   acp.SessionId `json:"sessionId"`
	Phase       string        `json:"phase"`
	Cause       string        `json:"cause"`
	Attempt     int           `json:"attempt,omitzero"`
	MaxAttempts int           `json:"maxAttempts,omitzero"`
	DelayMs     int64         `json:"delayMs,omitzero"`
	Error       string        `json:"error,omitzero"`
}

func (u *updates) flush() {
	u.thought.flush()
	u.text.flush()
}

// event translates one agent event.
func (u *updates) event(ev agent.Event, last *llm.Usage) {
	switch ev := ev.(type) {
	case agent.TextDelta:
		u.thought.flush()
		u.text.add(ev.Text)
	case agent.ThinkingDelta:
		u.text.flush()
		u.thought.add(ev.Text)
	case agent.ToolStart:
		u.flush()
		opts := []acp.ToolCallStartOpt{acp.WithStartKind(toolKind(ev.Spec.Kind)),
			acp.WithStartStatus(acp.ToolCallStatusPending), acp.WithStartRawInput(rawInput(ev.Call))}
		if l, ok := u.tools[ev.Call.Name].(tool.Locator); ok {
			if locs := acpLocations(l.Locations(ev.Call, u.env)); len(locs) > 0 {
				opts = append(opts, acp.WithStartLocations(locs))
			}
		}
		u.send(acp.StartToolCall(acp.ToolCallId(ev.Call.ID), cutTitle(ev.Title), opts...))
	case agent.Retry:
		u.flush()
		u.notify(methodRetry, retryNotice{SessionID: u.session, Phase: string(ev.Phase), Cause: string(ev.Cause),
			Attempt: ev.Attempt, MaxAttempts: ev.MaxAttempts, DelayMs: ev.Delay.Milliseconds(), Error: ev.Err})
	case agent.ToolRun:
		u.flush()
		u.running = acp.ToolCallId(ev.Call.ID)
		u.send(acp.UpdateToolCall(u.running, acp.WithUpdateStatus(acp.ToolCallStatusInProgress)))
	case agent.ToolEnd:
		u.flush()
		status := acp.ToolCallStatusCompleted
		if ev.Result.IsError {
			status = acp.ToolCallStatusFailed
		}
		// The terminal status goes alone in its update, with the content
		// (0008-MADR D19 item 2).
		id := acp.ToolCallId(ev.Call.ID)
		content := resultContent(ev.Result)
		if term := u.terms[id]; term != "" {
			content = append([]acp.ToolCallContent{acp.ToolTerminalRef(term)}, content...)
		}
		opts := []acp.ToolCallUpdateOpt{acp.WithUpdateStatus(status), acp.WithUpdateContent(content)}
		if locs := acpLocations(ev.Result.Locations); len(locs) > 0 {
			opts = append(opts, acp.WithUpdateLocations(locs))
		}
		u.send(acp.UpdateToolCall(id, opts...))
		u.running = ""
		if ev.Result.Plan != nil {
			u.send(planUpdate(ev.Result.Plan))
		}
	case agent.Usage:
		*last = ev.Usage
	case agent.Queued:
		u.flush()
		u.send(acp.UpdateUserMessageText(promptTextOf(ev.Message)))
	case agent.End:
		u.flush()
	}
}

// locate gives the updates the turn's tools and environment, so each
// call's tool_call names its locations.
func (u *updates) locate(tools []tool.Tool, env tool.Env) {
	u.tools = make(map[string]tool.Tool, len(tools))
	for _, t := range tools {
		u.tools[t.Spec().Name] = t
	}
	u.env = env
}

// terminal attaches a client terminal to the running call, so the client
// shows it live; the call's last update carries it too.
func (u *updates) terminal(id string) {
	if u.running == "" {
		return
	}
	u.terms[u.running] = id
	u.send(acp.UpdateToolCall(u.running, acp.WithUpdateContent([]acp.ToolCallContent{acp.ToolTerminalRef(id)})))
}

// acpLocations are locations as ACP's, a line of 0 left out.
func acpLocations(locs []tool.Location) []acp.ToolCallLocation {
	out := make([]acp.ToolCallLocation, 0, len(locs))
	for _, l := range locs {
		loc := acp.ToolCallLocation{Path: l.Path}
		loc.Line = line(l.Line)
		out = append(out, loc)
	}
	return out
}

// promptTextOf is a user message's text blocks, joined as its entry joins
// them (userEntryMessage).
func promptTextOf(m llm.Message) string {
	var texts []string
	for _, c := range m.Content {
		if c.Type == llm.ContentText {
			texts = append(texts, c.Text)
		}
	}
	return strings.Join(texts, "\n\n")
}

func toolKind(k tool.Kind) acp.ToolKind {
	if k == "" {
		return acp.ToolKind(tool.KindOther)
	}
	return acp.ToolKind(k)
}

func rawInput(c tool.Call) stdjson.RawMessage {
	if len(c.Args) == 0 {
		return nil
	}
	return jsonRaw(c.Args)
}

// resultContent is a finished call's content: a self-contained summary of at
// most maxSummary bytes first, then the diffs, then the output.
func resultContent(r tool.Result) []acp.ToolCallContent {
	text := r.Text()
	sum := r.Summary
	if sum == "" {
		sum, _, _ = strings.Cut(strings.TrimSpace(text), "\n")
	}
	out := []acp.ToolCallContent{acp.ToolContent(acp.TextBlock(cutBytes(sum, maxSummary)))}
	for _, d := range r.Diffs {
		if d.OldText != nil {
			out = append(out, acp.ToolDiffContent(d.Path, d.NewText, *d.OldText))
		} else {
			out = append(out, acp.ToolDiffContent(d.Path, d.NewText))
		}
	}
	if len(r.Diffs) == 0 && text != "" && text != sum {
		out = append(out, acp.ToolContent(acp.TextBlock(text)))
	}
	return out
}

// planUpdate is a tool's plan as ACP's plan update, one entry per item in
// order. An empty plan is an update with no entries, which clears the
// client's plan.
func planUpdate(plan []tool.PlanItem) acp.SessionUpdate {
	entries := make([]acp.PlanEntry, len(plan))
	for i, it := range plan {
		entries[i] = acp.PlanEntry{Content: it.Content, Status: acp.PlanEntryStatus(it.Status), Priority: acp.PlanEntryPriority(it.Priority)}
	}
	return acp.UpdatePlan(entries...)
}

// cutTitle is a title on one line, at most maxTitle runes.
func cutTitle(s string) string { return cutTo(s, maxTitle) }

// cutTo is s on one line, cut to n runes.
func cutTo(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n-1]) + "…"
}

// ruleTitle is a permission request's title: the call's, and for a
// repeated call's question what it asks about, the call's cut to leave it
// room (0005-PLAN F5-3 decision 1).
func ruleTitle(r permission.Rule) string {
	if r.Repeated == 0 {
		return cutTitle(r.Title)
	}
	suffix := fmt.Sprintf(" (the same call %d times in a row)", r.Repeated)
	return cutTo(r.Title, maxTitle-utf8.RuneCountInString(suffix)) + suffix
}

// cutBytes is s cut to n bytes on a rune boundary.
func cutBytes(s string, n int) string {
	if len(s) <= n {
		return s
	}
	cut := n - len("…")
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}

// acpPolicy approves calls by asking the client (session/request_permission),
// with one allow and one reject option. A cancelled request is a refusal.
type acpPolicy struct {
	conn    *acp.AgentSideConnection
	session acp.SessionId
}

func (p *acpPolicy) Decide(ctx context.Context, r permission.Rule) (permission.Decision, error) {
	if p.conn == nil {
		return permission.Decision{}, errors.New("no client connection")
	}
	resp, err := p.conn.RequestPermission(ctx, acp.RequestPermissionRequest{
		SessionId: p.session,
		ToolCall:  acp.ToolCallUpdate{ToolCallId: acp.ToolCallId(r.CallID), Title: new(ruleTitle(r)), RawInput: jsonRaw(r.Input)},
		Options: []acp.PermissionOption{
			{OptionId: allowOnce, Name: "Allow", Kind: acp.PermissionOptionKindAllowOnce},
			{OptionId: rejectOnce, Name: "Reject", Kind: acp.PermissionOptionKindRejectOnce},
		},
	})
	if err != nil {
		return permission.Decision{}, err
	}
	if s := resp.Outcome.Selected; s != nil && s.OptionId == allowOnce {
		return permission.Decision{Allow: true}, nil
	}
	return permission.Decision{Allow: false}, nil
}

// jsonRaw passes JSON through the SDK's encoding/json as it is, as a
// json.RawMessage, which encodes verbatim. Empty is none, which the
// field's omitempty leaves out (0014-MADR D4).
func jsonRaw(v jsontext.Value) stdjson.RawMessage {
	if len(v) == 0 {
		return nil
	}
	return v
}

// promptError is the JSON-RPC error of a failed turn: auth_required for a
// missing or refused credential, otherwise an internal error carrying the
// message.
func promptError(err error) error {
	if errors.Is(err, llm.ErrAuth) {
		return acp.NewAuthRequired(map[string]any{keyError: err.Error()})
	}
	return acp.NewInternalError(map[string]any{keyError: err.Error()})
}

// acpUsage is the turn's usage for the prompt response.
func acpUsage(u llm.Usage) *acp.Usage {
	out := &acp.Usage{InputTokens: count(u.InputTokens), OutputTokens: count(u.OutputTokens), TotalTokens: count(u.InputTokens + u.OutputTokens)}
	if u.ReasoningTokens > 0 {
		out.ThoughtTokens = new(count(u.ReasoningTokens))
	}
	if u.CacheReadTokens > 0 {
		out.CachedReadTokens = new(count(u.CacheReadTokens))
	}
	if u.CacheWriteTokens > 0 {
		out.CachedWriteTokens = new(count(u.CacheWriteTokens))
	}
	return out
}

// contextUpdate is a usage_update of used tokens: the session model's
// window as its size, 0 when unknown, and the session's cost in USD when the
// model is priced (0005-PLAN F4-2a).
func (a *Agent) contextUpdate(s *liveSession, used int) acp.SessionUpdate {
	model, _ := a.choices(s)
	info := a.modelOf(model)
	u := &acp.SessionUsageUpdate{SessionUpdate: "usage_update", Used: count(int64(used)), Size: count(int64(info.ContextWindow))}
	if info.Priced {
		u.Cost = &acp.Cost{Amount: sessionTotals(s.log.Entries()).cost, Currency: "USD"}
	}
	return acp.SessionUpdate{UsageUpdate: u}
}

// count is a token count as ACP's: a negative count, which no provider
// should report, is 0 rather than wrapped (0014-PLAN C4).
func count(n int64) uint64 {
	if n < 0 {
		return 0
	}
	return uint64(n)
}

// line is a location's line as ACP's: none for 0, a negative line or one
// past ACP's range, rather than a wrapped one (0014-PLAN C4).
func line(n int) *uint32 {
	if n <= 0 || n > math.MaxUint32 {
		return nil
	}
	return new(uint32(n))
}
