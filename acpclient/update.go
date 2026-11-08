package acpclient

import (
	"encoding/json/jsontext"
	"fmt"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/gobble-cli/acpext"

	"github.com/maccavelli/gobble-cli/jsonx"
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
	// F5-2 decision 9). With Options.Compaction it is also ACP's
	// compaction_update and compaction_summary_chunk, a session/update
	// (gobble-cli 0016-MADR D10).
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
	// Variant is the wire's sessionUpdate value, the SDK's Kind(), for every
	// update, a KindOther's included: "config_option_update", or a variant
	// newer than this client, by its own name (0035-MADR D9).
	Variant string
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
	// for clients that pass the ACP stream through (0008-MADR D9). It is
	// set only when Options.Params is (0039-MADR D16). For KindRetry,
	// KindPersonality and a KindCompaction from _gobble/compaction it is the
	// notification's params, as received.
	Params jsontext.Value
}

// Retry is a wait the agent's turn takes before calling the model again,
// as _gobble/retry reports it: Phase start or end, Cause offline,
// empty_reply, or a provider retry's failure class, such as rate-limited
// (0031-MADR D4, as amended). Attempt, MaxAttempts and Delay are an empty
// reply's or a provider retry's; Error
// is why an End is not a success, or "".
type Retry struct {
	Phase, Cause         string
	Attempt, MaxAttempts int
	Delay                time.Duration
	Error                string
}

// retryOf decodes _gobble/retry's params, with the session they name.
func retryOf(params jsontext.Value) (acp.SessionId, Retry, error) {
	var n acpext.RetryNotice
	if err := jsonx.Unmarshal(params, &n); err != nil {
		return "", Retry{}, fmt.Errorf("%s: %w", acpext.MethodRetry, err)
	}
	return acp.SessionId(n.SessionID), Retry{Phase: string(n.Phase), Cause: string(n.Cause), Attempt: n.Attempt, MaxAttempts: n.MaxAttempts,
		Delay: time.Duration(n.DelayMs) * time.Millisecond, Error: n.Error}, nil
}

// Compaction is a compaction's start or end, as _gobble/compaction
// reports it: Phase start or end; Reason manual, threshold or overflow;
// TokensBefore a successful end's; Error why an end is not a success, or
// ""; Paused "turn" or "session" when the failure paused automatic
// compaction (0005-PLAN F5-2 decisions 9 and 11).
//
// From ACP's compaction_update and compaction_summary_chunk, offered by
// Options.Compaction, it is the compaction so far, as the SDK's
// CompactionTimeline folds it: ID and Status are the protocol's, an
// unknown status as it arrived; Summary is the summary's text so far;
// Reason, TokensBefore and Paused come from _meta.gobble. An in_progress
// update is a start and a completed, failed or cancelled one an end, a
// cancelled one with the Error "cancelled", as _gobble/compaction says; a
// chunk, and a status the protocol does not define, are neither. ID and
// Status are "" for _gobble/compaction (gobble-cli 0016-MADR D9, D10).
type Compaction struct {
	Phase, Reason string
	TokensBefore  int64
	Error, Paused string
	ID, Status    string
	Summary       string
}

// compactionFrom is the compaction e as Compaction describes it; chunk
// says a summary chunk changed it.
func compactionFrom(e acp.CompactionEntity, chunk bool) Compaction {
	c := Compaction{ID: string(e.Id), Status: string(e.Status), Error: e.Error}
	for _, b := range e.Summary {
		text, block := blockText(b)
		if block != "" {
			text = "[" + block + "]"
		}
		c.Summary += text
	}
	if meta, ok, err := acp.DecodeMeta[acpext.CompactionMeta](e.Meta, acpext.MetaKey); ok && err == nil {
		c.Reason, c.TokensBefore, c.Paused = string(meta.Reason), meta.TokensBefore, string(meta.Paused)
	}
	switch {
	case chunk:
	case e.Status == acp.CompactionStatusInProgress:
		c.Phase = string(acpext.PhaseStart)
	case e.Status == acp.CompactionStatusCancelled:
		c.Phase, c.Error = string(acpext.PhaseEnd), "cancelled"
	case e.Status == acp.CompactionStatusCompleted, e.Status == acp.CompactionStatusFailed:
		c.Phase = string(acpext.PhaseEnd)
	}
	return c
}

// compactionOf decodes _gobble/compaction's params, with the session they
// name.
func compactionOf(params jsontext.Value) (acp.SessionId, Compaction, error) {
	var n acpext.CompactionNotice
	if err := jsonx.Unmarshal(params, &n); err != nil {
		return "", Compaction{}, fmt.Errorf("%s: %w", acpext.MethodCompaction, err)
	}
	return acp.SessionId(n.SessionID), Compaction{Phase: string(n.Phase), Reason: string(n.Reason), TokensBefore: n.TokensBefore, Error: n.Error,
		Paused: string(n.Paused)}, nil
}

// Personality is a session's personality, as _gobble/personality reports
// it: its name and its level, 1 to 5 (0022-MADR D1, D6).
type Personality struct {
	Name  string
	Level int
}

// personalityOf decodes _gobble/personality's params, with the session
// they name.
func personalityOf(params jsontext.Value) (acp.SessionId, Personality, error) {
	var n acpext.PersonalityNotice
	if err := jsonx.Unmarshal(params, &n); err != nil {
		return "", Personality{}, fmt.Errorf("%s: %w", acpext.MethodPersonality, err)
	}
	return acp.SessionId(n.SessionID), Personality{Name: n.Name, Level: n.Level}, nil
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

// translate makes an Update from a notification. With params, it encodes
// the notification as Update.Params, with encoding/json/v2 and v1's
// escaping of <, >, & and U+2028 and U+2029, so its bytes are those the SDK
// writes (0039-MADR D16, Q4). With tl, the session's compactions, a
// compaction_update or compaction_summary_chunk is folded into it and is
// KindCompaction; without it, both are KindOther.
func translate(n acp.SessionNotification, params bool, tl *acp.CompactionTimeline) (Update, error) {
	s := n.Update
	u := Update{Variant: s.Kind()}
	if params {
		b, err := jsonx.Marshal(n, jsontext.EscapeForHTML(true), jsontext.EscapeForJS(true))
		if err != nil {
			return Update{}, fmt.Errorf("encode session/update: %w", err)
		}
		u.Params = b
	}
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
	case tl != nil && (s.CompactionUpdate != nil || s.CompactionSummaryChunk != nil):
		e, _ := tl.Apply(s)
		u.Kind = KindCompaction
		u.Compaction = compactionFrom(e, s.CompactionSummaryChunk != nil)
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
