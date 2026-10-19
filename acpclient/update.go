package acpclient

import (
	"encoding/json"
	"fmt"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// Kind is the kind of a session update, as gobble's clients act on it.
type Kind int

// The update kinds. KindOther covers every update a client keeps only as
// state, such as config_option_update; its Params still carry it.
const (
	KindOther Kind = iota
	KindAgentText
	KindThought
	KindToolCall
	KindToolCallUpdate
	KindPlan
	KindUsage
	KindCommands
	KindMode
	KindSessionInfo
	// KindRetry is gobble's _gobble/retry notification, not a
	// session/update: Params holds the notification's params (0005-PLAN
	// F5-1 decision 5).
	KindRetry
	// KindCompaction is gobble's _gobble/compaction notification, not a
	// session/update: Params holds the notification's params (0005-PLAN
	// F5-2 decision 9).
	KindCompaction
	// KindPersonality is gobble's _gobble/personality notification, not a
	// session/update: Params holds the notification's params (0022-MADR
	// D6).
	KindPersonality
)

func (k Kind) String() string {
	switch k {
	case KindAgentText:
		return "agent_text"
	case KindThought:
		return "thought"
	case KindToolCall:
		return "tool_call"
	case KindToolCallUpdate:
		return "tool_call_update"
	case KindPlan:
		return "plan"
	case KindUsage:
		return "usage"
	case KindCommands:
		return "commands"
	case KindMode:
		return "mode"
	case KindSessionInfo:
		return "session_info"
	case KindRetry:
		return "retry"
	case KindCompaction:
		return "compaction"
	case KindPersonality:
		return "personality"
	}
	return "other"
}

// Update is one session/update, translated so that callers need no SDK
// type (0004-MADR rule 2).
type Update struct {
	Kind Kind
	// Text is the chunk's text, for KindAgentText and KindThought.
	Text string
	// Block names a chunk's content type when it is not text: "image",
	// "audio", "resource_link" or "resource".
	Block string
	// Tool is the call, for KindToolCall and KindToolCallUpdate.
	Tool ToolCall
	// Plan is the full plan, for KindPlan.
	Plan []PlanEntry
	// Context is the context window's use, for KindUsage.
	Context ContextUsage
	// Commands are the agent's slash commands, for KindCommands: the full
	// list, which replaces the one before.
	Commands []Command
	// ModeID is the session's mode, for KindMode.
	ModeID string
	// Title is the session's title, for KindSessionInfo.
	Title string
	// Retry is the turn's wait, for KindRetry.
	Retry Retry
	// Compaction is a compaction's start or end, for KindCompaction.
	Compaction Compaction
	// Personality is the session's personality, for KindPersonality.
	Personality Personality
	// Params is the session/update params object as the SDK encodes it,
	// for clients that pass the ACP stream through (0008-MADR D9).
	Params json.RawMessage
}

// Retry is a wait the agent's turn takes before calling the model again,
// as _gobble/retry reports it: Phase start or end, Cause offline or
// empty_reply. Attempt, MaxAttempts and Delay are an empty reply's; Error
// is why an End is not a success, or "".
type Retry struct {
	Phase, Cause         string
	Attempt, MaxAttempts int
	Delay                time.Duration
	Error                string
}

// methodRetry is gobble's retry notification (0005-PLAN F5-1).
const methodRetry = "_gobble/retry"

// retryOf decodes _gobble/retry's params, with the session they name.
func retryOf(params json.RawMessage) (acp.SessionId, Retry, error) {
	var n struct {
		SessionID   acp.SessionId `json:"sessionId"`
		Phase       string        `json:"phase"`
		Cause       string        `json:"cause"`
		Attempt     int           `json:"attempt"`
		MaxAttempts int           `json:"maxAttempts"`
		DelayMs     int64         `json:"delayMs"`
		Error       string        `json:"error"`
	}
	if err := json.Unmarshal(params, &n); err != nil {
		return "", Retry{}, fmt.Errorf("%s: %w", methodRetry, err)
	}
	return n.SessionID, Retry{Phase: n.Phase, Cause: n.Cause, Attempt: n.Attempt, MaxAttempts: n.MaxAttempts,
		Delay: time.Duration(n.DelayMs) * time.Millisecond, Error: n.Error}, nil
}

// Compaction is a compaction's start or end, as _gobble/compaction
// reports it: Phase start or end; Reason manual, threshold or overflow;
// TokensBefore a successful end's; Error why an end is not a success, or
// ""; Paused "turn" or "session" when the failure paused automatic
// compaction (0005-PLAN F5-2 decisions 9 and 11).
type Compaction struct {
	Phase, Reason string
	TokensBefore  int64
	Error, Paused string
}

// methodCompaction is gobble's compaction notification (0005-PLAN F5-2).
const methodCompaction = "_gobble/compaction"

// compactionOf decodes _gobble/compaction's params, with the session they
// name.
func compactionOf(params json.RawMessage) (acp.SessionId, Compaction, error) {
	var n struct {
		SessionID    acp.SessionId `json:"sessionId"`
		Phase        string        `json:"phase"`
		Reason       string        `json:"reason"`
		TokensBefore int64         `json:"tokensBefore"`
		Error        string        `json:"error"`
		Paused       string        `json:"paused"`
	}
	if err := json.Unmarshal(params, &n); err != nil {
		return "", Compaction{}, fmt.Errorf("%s: %w", methodCompaction, err)
	}
	return n.SessionID, Compaction{Phase: n.Phase, Reason: n.Reason, TokensBefore: n.TokensBefore, Error: n.Error, Paused: n.Paused}, nil
}

// Personality is a session's personality, as _gobble/personality reports
// it: its name and its level, 1 to 5 (0022-MADR D1, D6).
type Personality struct {
	Name  string
	Level int
}

// methodPersonality is gobble's personality notification (0022-PLAN P3).
const methodPersonality = "_gobble/personality"

// personalityOf decodes _gobble/personality's params, with the session
// they name.
func personalityOf(params json.RawMessage) (acp.SessionId, Personality, error) {
	var n struct {
		SessionID acp.SessionId `json:"sessionId"`
		Name      string        `json:"name"`
		Level     int           `json:"level"`
	}
	if err := json.Unmarshal(params, &n); err != nil {
		return "", Personality{}, fmt.Errorf("%s: %w", methodPersonality, err)
	}
	return n.SessionID, Personality{Name: n.Name, Level: n.Level}, nil
}

// ToolCall is a tool call or an update to one. In an update, an empty
// Title or Status, and nil Content, leave the field as it was.
type ToolCall struct {
	ID, Title, Status string
	Content           []ToolContent
}

// ToolContent is one block of a tool call's content: exactly one field is
// set.
type ToolContent struct {
	Text     string
	Diff     *Diff
	Terminal string
}

// Diff is a file change as ACP carries it. OldText is nil for a new file.
type Diff struct {
	Path    string
	OldText *string
	NewText string
}

// Command is one slash command the agent executes.
type Command struct {
	Name, Description, Hint string
}

// PlanEntry is one step of the agent's plan.
type PlanEntry struct {
	Content, Status, Priority string
}

// ContextUsage is a usage_update: the context window's use and the
// session's cost so far.
type ContextUsage struct {
	Used, Size int64
	Cost       *Cost
}

// Cost is an amount of money.
type Cost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// translate makes an Update from a notification.
func translate(n acp.SessionNotification) (Update, error) {
	params, err := json.Marshal(n)
	if err != nil {
		return Update{}, fmt.Errorf("encode session/update: %w", err)
	}
	u := Update{Params: params}
	s := n.Update
	switch {
	case s.AgentMessageChunk != nil:
		u.Kind = KindAgentText
		u.Text, u.Block = blockText(s.AgentMessageChunk.Content)
	case s.AgentThoughtChunk != nil:
		u.Kind = KindThought
		u.Text, u.Block = blockText(s.AgentThoughtChunk.Content)
	case s.ToolCall != nil:
		c := s.ToolCall
		u.Kind = KindToolCall
		u.Tool = ToolCall{ID: string(c.ToolCallId), Title: c.Title, Status: string(c.Status), Content: toolContent(c.Content)}
	case s.ToolCallUpdate != nil:
		c := s.ToolCallUpdate
		u.Kind = KindToolCallUpdate
		u.Tool = ToolCall{ID: string(c.ToolCallId), Content: toolContent(c.Content)}
		if c.Title != nil {
			u.Tool.Title = *c.Title
		}
		if c.Status != nil {
			u.Tool.Status = string(*c.Status)
		}
	case s.Plan != nil:
		u.Kind = KindPlan
		u.Plan = make([]PlanEntry, 0, len(s.Plan.Entries))
		for _, e := range s.Plan.Entries {
			u.Plan = append(u.Plan, PlanEntry{Content: e.Content, Status: string(e.Status), Priority: string(e.Priority)})
		}
	case s.UsageUpdate != nil:
		c := s.UsageUpdate
		u.Kind = KindUsage
		u.Context = ContextUsage{Used: toInt64(c.Used), Size: toInt64(c.Size)}
		if c.Cost != nil {
			u.Context.Cost = &Cost{Amount: c.Cost.Amount, Currency: c.Cost.Currency}
		}
	case s.AvailableCommandsUpdate != nil:
		u.Kind = KindCommands
		u.Commands = make([]Command, 0, len(s.AvailableCommandsUpdate.AvailableCommands))
		for _, c := range s.AvailableCommandsUpdate.AvailableCommands {
			cmd := Command{Name: c.Name, Description: c.Description}
			if c.Input != nil && c.Input.Unstructured != nil {
				cmd.Hint = c.Input.Unstructured.Hint
			}
			u.Commands = append(u.Commands, cmd)
		}
	case s.CurrentModeUpdate != nil:
		u.Kind = KindMode
		u.ModeID = string(s.CurrentModeUpdate.CurrentModeId)
	case s.SessionInfoUpdate != nil:
		u.Kind = KindSessionInfo
		if t, ok := s.SessionInfoUpdate.Title.Value(); ok {
			u.Title = t
		}
	}
	return u, nil
}

// blockText is a content block's text, or the name of its type when it is
// not text.
func blockText(b acp.ContentBlock) (text, block string) {
	switch {
	case b.Text != nil:
		return b.Text.Text, ""
	case b.Image != nil:
		return "", "image"
	case b.Audio != nil:
		return "", "audio"
	case b.ResourceLink != nil:
		return "", "resource_link"
	case b.Resource != nil:
		return "", "resource"
	}
	return "", "unknown"
}

func toolContent(cs []acp.ToolCallContent) []ToolContent {
	if cs == nil {
		return nil
	}
	out := make([]ToolContent, 0, len(cs))
	for _, c := range cs {
		switch {
		case c.Content != nil:
			text, block := blockText(c.Content.Content)
			if block != "" {
				text = "[" + block + "]"
			}
			out = append(out, ToolContent{Text: text})
		case c.Diff != nil:
			out = append(out, ToolContent{Diff: &Diff{Path: c.Diff.Path, OldText: c.Diff.OldText, NewText: c.Diff.NewText}})
		case c.Terminal != nil:
			out = append(out, ToolContent{Terminal: string(c.Terminal.TerminalId)})
		}
	}
	return out
}
