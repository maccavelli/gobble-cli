package acpserver

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"unicode/utf8"

	acp "github.com/coder/acp-go-sdk"

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

// systemPrompt is the constant system prompt until 0005-PLAN F3 builds it
// from sections and context files.
func systemPrompt(cwd string, tools []tool.Tool) string {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Spec().Name
	}
	var b strings.Builder
	b.WriteString("You are gobble, a coding agent. You help the user with software engineering tasks in their project.\n")
	fmt.Fprintf(&b, "The working directory is %s, on %s. Relative paths resolve against it.\n", cwd, runtime.GOOS)
	if len(names) > 0 {
		fmt.Fprintf(&b, "Use the tools (%s) to read and change files and run commands; do not guess at file contents.\n", strings.Join(names, ", "))
	}
	b.WriteString("A call to a tool that changes something may be declined by the user; then do not retry it, and say what you would have done.\n")
	b.WriteString("Answer concisely, in Markdown.")
	return b.String()
}

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
}

func newUpdates(ctx context.Context, conn *acp.AgentSideConnection, id acp.SessionId) *updates {
	u := &updates{conn: conn, ctx: context.WithoutCancel(ctx), session: id}
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
		u.send(acp.StartToolCall(acp.ToolCallId(ev.Call.ID), cutTitle(ev.Title),
			acp.WithStartKind(toolKind(ev.Spec.Kind)),
			acp.WithStartStatus(acp.ToolCallStatusPending),
			acp.WithStartRawInput(rawInput(ev.Call))))
	case agent.ToolRun:
		u.flush()
		u.send(acp.UpdateToolCall(acp.ToolCallId(ev.Call.ID), acp.WithUpdateStatus(acp.ToolCallStatusInProgress)))
	case agent.ToolEnd:
		u.flush()
		status := acp.ToolCallStatusCompleted
		if ev.Result.IsError {
			status = acp.ToolCallStatusFailed
		}
		// The terminal status goes alone in its update, with the content
		// (0008-MADR D19 item 2).
		u.send(acp.UpdateToolCall(acp.ToolCallId(ev.Call.ID),
			acp.WithUpdateStatus(status), acp.WithUpdateContent(resultContent(ev.Result))))
	case agent.Usage:
		*last = ev.Usage
	case agent.End:
		u.flush()
	}
}

func toolKind(k tool.Kind) acp.ToolKind {
	if k == "" {
		return acp.ToolKind(tool.KindOther)
	}
	return acp.ToolKind(k)
}

func rawInput(c tool.Call) any {
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

// cutTitle is a title on one line, at most maxTitle runes.
func cutTitle(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if utf8.RuneCountInString(s) <= maxTitle {
		return s
	}
	return string([]rune(s)[:maxTitle-1]) + "…"
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
		ToolCall:  acp.ToolCallUpdate{ToolCallId: acp.ToolCallId(r.CallID), Title: new(cutTitle(r.Title)), RawInput: jsonRaw(r.Input)},
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

// jsonRaw passes JSON through the SDK's encoding/json as it is: a
// jsontext.Value is a json.RawMessage, so it encodes verbatim. Empty is
// none.
func jsonRaw(v jsontext.Value) any {
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
		return acp.NewAuthRequired(map[string]any{"error": err.Error()})
	}
	return acp.NewInternalError(map[string]any{"error": err.Error()})
}

// acpUsage is the turn's usage for the prompt response.
func acpUsage(u llm.Usage) *acp.Usage {
	out := &acp.Usage{InputTokens: int(u.InputTokens), OutputTokens: int(u.OutputTokens), TotalTokens: int(u.InputTokens + u.OutputTokens)}
	if u.ReasoningTokens > 0 {
		out.ThoughtTokens = new(int(u.ReasoningTokens))
	}
	if u.CacheReadTokens > 0 {
		out.CachedReadTokens = new(int(u.CacheReadTokens))
	}
	return out
}

// usageUpdate is the context window's use after a turn: the last model
// call's input and output. The size is 0 until the catalog gives one
// (0005-PLAN F4).
func usageUpdate(last llm.Usage) acp.SessionUpdate {
	return usedUpdate(int(last.InputTokens + last.OutputTokens))
}

// usedUpdate is a usage_update of used tokens.
func usedUpdate(used int) acp.SessionUpdate {
	return acp.SessionUpdate{UsageUpdate: &acp.SessionUsageUpdate{SessionUpdate: "usage_update", Used: used}}
}
