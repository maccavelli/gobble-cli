package compaction

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"slices"
	"time"

	"github.com/maccavelli/gobble-cli/session"
)

// imageChars is what an image counts for, in characters (Pi's
// ESTIMATED_IMAGE_CHARS).
const imageChars = 4800

// utf16Len is a string's length as JavaScript counts it, in UTF-16 code
// units, so estimates match Pi's.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r > 0xFFFF {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// tokens is Pi's chars/4 estimate, rounded up.
func tokens(chars int) int { return (chars + 3) / 4 }

// EstimateText is a text's tokens by the same heuristic.
func EstimateText(s string) int { return tokens(utf16Len(s)) }

// Estimate is a message's tokens by Pi's chars/4 heuristic
// (compaction.ts estimateTokens): text and thinking count their
// characters, an image 4800, a tool call its name and its JSON arguments,
// a bash execution its command and output, and a branch or compaction
// summary its summary. A role gobble does not read counts nothing.
func Estimate(m *session.Message) int {
	if m == nil {
		return 0
	}
	switch m.Role {
	case session.RoleBashExecution:
		chars := utf16Len(m.Command)
		if m.Output != nil {
			chars += utf16Len(*m.Output)
		}
		return tokens(chars)
	case session.RoleBranchSummary, session.RoleCompactionSummary:
		return tokens(utf16Len(m.Summary))
	}
	blocks, err := m.Blocks()
	if err != nil {
		return 0
	}
	chars := 0
	switch m.Role {
	case session.RoleUser, session.RoleToolResult, session.RoleCustom:
		for _, b := range blocks {
			switch b.Type {
			case session.BlockText:
				chars += utf16Len(b.Text)
			case session.BlockImage:
				chars += imageChars
			}
		}
	case session.RoleAssistant:
		for _, b := range blocks {
			switch b.Type {
			case session.BlockText:
				chars += utf16Len(b.Text)
			case session.BlockThinking:
				chars += utf16Len(b.Thinking)
			case session.BlockToolCall:
				chars += utf16Len(b.Name) + utf16Len(compactJSON(b.Arguments))
			}
		}
	default:
		return 0
	}
	return tokens(chars)
}

// compactJSON is a value as JSON.stringify writes it: without whitespace.
func compactJSON(v jsontext.Value) string {
	if len(v) == 0 {
		return "{}"
	}
	c := v.Clone()
	if err := c.Compact(); err != nil {
		return string(v)
	}
	return string(c)
}

// cutPoint reports a message a cut may land on: never a tool result, which
// must stay with its call (Pi's isCutPointMessage).
func cutPoint(m *session.Message) bool {
	return m.Role == session.RoleAssistant || turnStart(m)
}

// turnStart reports a message that starts a turn (Pi's isTurnStartMessage).
func turnStart(m *session.Message) bool {
	switch m.Role {
	case session.RoleUser, session.RoleBashExecution, session.RoleCustom, session.RoleBranchSummary, session.RoleCompactionSummary:
		return true
	}
	return false
}

// Projected is one context entry and the messages it gives the model, in
// Pi's AgentMessage shapes (session-manager.ts ProjectedSessionEntry).
type Projected struct {
	Entry    session.Entry
	Messages []*session.Message
}

// Context is a path's model context as Pi builds it (session-manager.ts
// buildContextEntries): with no compaction, the path; otherwise the newest
// compaction, then the path's entries from its first kept entry up to it,
// then every entry after it.
func Context(path []session.Entry) []session.Entry {
	at := -1
	for i, e := range path {
		if e.Type == session.TypeCompaction {
			at = i
		}
	}
	if at < 0 {
		return path
	}
	c := path[at]
	out := []session.Entry{c}
	kept := false
	for _, e := range path[:at] {
		if e.ID == c.FirstKeptEntryID {
			kept = true
		}
		if kept && !isSystem(e) {
			out = append(out, e)
		}
	}
	return append(out, path[at+1:]...)
}

func isSystem(e session.Entry) bool {
	return e.Type == session.TypeMessage && e.Message != nil && e.Message.Role == session.RoleSystem
}

// Project gives each context entry its messages, as Pi's
// buildSessionProjection does: the last context_edit of an entry among
// ctx replaces its content, or, with a null replacement, leaves it out;
// and only the first entry's compaction gives its summary, an older one in
// the kept range giving nothing.
//
// A Pi system message, a system entry or a compaction's systemMessage,
// carries Pi's prompt and tool state: it is kept in the file and not
// projected, as gobble's model sees gobble's own prompt and tools
// (0005-PLAN F2).
func Project(ctx []session.Entry) []Projected {
	edits := map[string]session.Entry{}
	for _, e := range ctx {
		if e.Type == session.TypeContextEdit {
			edits[e.TargetID] = e
		}
	}
	out := make([]Projected, len(ctx))
	for i, e := range ctx {
		out[i].Entry = e
		if e.Type == session.TypeCompaction && i > 0 {
			continue
		}
		out[i].Messages = edit(entryMessages(e), edits[e.ID])
	}
	return out
}

// entryMessages is what one entry gives the model before edits (Pi's
// sessionEntryToContextMessages), without system messages.
func entryMessages(e session.Entry) []*session.Message {
	switch e.Type {
	case session.TypeMessage:
		if e.Message == nil || e.Message.Role == session.RoleSystem {
			return nil
		}
		m := e.Message
		switch m.Role {
		case session.RoleUser, session.RoleAssistant, session.RoleToolResult:
			if missing(m.Content) {
				c := *m
				c.Content = jsontext.Value("[]")
				m = &c
			}
		}
		return []*session.Message{m}
	case session.TypeCustomMessage:
		content := e.Content
		if missing(content) {
			content = jsontext.Value("[]")
		}
		return []*session.Message{{Role: session.RoleCustom, CustomType: e.CustomType, Content: content, Display: e.Display,
			Details: e.Details, Timestamp: millis(e.Timestamp)}}
	case session.TypeBranchSummary:
		if e.Summary == "" {
			return nil
		}
		return []*session.Message{{Role: session.RoleBranchSummary, Summary: e.Summary, FromID: e.FromID, Timestamp: millis(e.Timestamp)}}
	case session.TypeCompaction:
		return []*session.Message{{Role: session.RoleCompactionSummary, Summary: e.Summary, TokensBefore: e.TokensBefore, Timestamp: millis(e.Timestamp)}}
	}
	return nil
}

// missing reports a JSON member that is absent or null, as Pi's
// "== null" tests one.
func missing(v jsontext.Value) bool { return len(v) == 0 || string(v) == "null" }

// edit applies a context_edit to an entry's messages (Pi's
// projectContextEntry). A replacement's string content is one text block
// for an assistant or a tool result.
func edit(msgs []*session.Message, e session.Entry) []*session.Message {
	if e.Type != session.TypeContextEdit {
		return msgs
	}
	if missing(e.Replacement) {
		return nil
	}
	var r struct {
		Content jsontext.Value `json:"content"`
	}
	if json.Unmarshal(e.Replacement, &r) != nil {
		return msgs
	}
	out := make([]*session.Message, len(msgs))
	for i, m := range msgs {
		out[i] = m
		switch m.Role {
		case session.RoleUser, session.RoleAssistant, session.RoleToolResult, session.RoleCustom:
		default:
			continue
		}
		c := *m
		c.Content = r.Content
		if m.Role == session.RoleAssistant || m.Role == session.RoleToolResult {
			var s string
			if json.Unmarshal(r.Content, &s) == nil {
				_ = c.SetBlocks([]session.Block{{Type: session.BlockText, Text: s}}) //nolint:errcheck // a text block always encodes
			}
		}
		out[i] = &c
	}
	return out
}

// millis is an entry timestamp in milliseconds, as new Date(ts).getTime()
// reads it, or 0.
func millis(ts string) int64 {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return 0
	}
	return t.UnixMilli()
}

func entryTokens(p Projected) int {
	n := 0
	for _, m := range p.Messages {
		n += Estimate(m)
	}
	return n
}

// EstimateEntries is the estimated context of a path: the last valid
// assistant usage, plus estimates of what follows it, when no context_edit
// or compaction comes after that usage; otherwise the sum of estimates
// (Pi's estimateProjectedContextTokens, with no system message: gobble
// writes none into a session, and projects none).
func EstimateEntries(path []session.Entry) int {
	proj := Project(Context(path))
	var flat []*session.Message
	var owner []string
	for _, p := range proj {
		for _, m := range p.Messages {
			flat = append(flat, m)
			owner = append(owner, p.Entry.ID)
		}
	}
	invalidating := -1
	for i, e := range path {
		if e.Type == session.TypeCompaction || e.Type == session.TypeContextEdit {
			invalidating = i
		}
	}
	for i, m := range slices.Backward(flat) {
		used := assistantUsage(m)
		if used == 0 {
			continue
		}
		usageAt := -1
		for j, e := range path {
			if e.ID == owner[i] {
				usageAt = j
			}
		}
		if usageAt <= invalidating {
			break
		}
		for _, m := range flat[i+1:] {
			used += Estimate(m)
		}
		return used
	}
	total := 0
	for _, m := range flat {
		total += Estimate(m)
	}
	return total
}

// assistantUsage is an assistant message's context tokens, or 0 when it
// has none worth trusting: aborted, failed, or all zero (Pi's
// getAssistantUsage).
func assistantUsage(m *session.Message) int {
	if m.Role != session.RoleAssistant || m.Usage == nil {
		return 0
	}
	if m.StopReason == "aborted" || m.StopReason == "error" {
		return 0
	}
	u := m.Usage
	if u.TotalTokens > 0 {
		return int(u.TotalTokens)
	}
	return int(u.Input + u.Output + u.CacheRead + u.CacheWrite)
}
