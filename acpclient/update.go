package acpclient

import (
	"encoding/json"
	"fmt"

	acp "github.com/coder/acp-go-sdk"
)

// Kind is the kind of a session update, as gobble's clients act on it.
type Kind int

// The update kinds. KindOther covers every update a client keeps only as
// state, such as available_commands_update; its Params still carry it.
const (
	KindOther Kind = iota
	KindAgentText
	KindThought
	KindToolCall
	KindToolCallUpdate
	KindPlan
	KindUsage
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
	// Params is the session/update params object as the SDK encodes it,
	// for clients that pass the ACP stream through (0008-MADR D9).
	Params json.RawMessage
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
		u.Context = ContextUsage{Used: int64(c.Used), Size: int64(c.Size)}
		if c.Cost != nil {
			u.Context.Cost = &Cost{Amount: c.Cost.Amount, Currency: c.Cost.Currency}
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
			out = append(out, ToolContent{Terminal: c.Terminal.TerminalId})
		}
	}
	return out
}
