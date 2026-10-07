package compaction

import (
	"encoding/json/jsontext"
	"slices"

	"github.com/maccavelli/gobble-cli/session"
)

// imageChars is what an image counts for, in characters (Pi's
// ESTIMATED_IMAGE_CHARS).
const imageChars = 4800

// roleCompaction is the role of a compaction's summary in a projection,
// Pi's compactionSummary.
const roleCompaction = "compactionSummary"

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
// characters, an image 4800, and a tool call its name and its JSON
// arguments. A role gobble does not read counts nothing.
func Estimate(m *session.Message) int {
	if m == nil {
		return 0
	}
	blocks, err := m.Blocks()
	if err != nil {
		return 0
	}
	chars := 0
	switch m.Role {
	case session.RoleUser, session.RoleToolResult:
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

// msg is one message of a projection: a session message, or a
// compaction's summary.
type msg struct {
	role    string
	m       *session.Message
	summary string
}

func (x msg) estimate() int {
	if x.role == roleCompaction {
		return tokens(utf16Len(x.summary))
	}
	return Estimate(x.m)
}

// cutPoint reports a message a cut may land on: never a tool result, which
// must stay with its call (Pi's isCutPointMessage).
func (x msg) cutPoint() bool {
	switch x.role {
	case session.RoleUser, session.RoleAssistant, roleCompaction:
		return true
	}
	return false
}

// turnStart reports a message that starts a turn (Pi's isTurnStartMessage).
func (x msg) turnStart() bool {
	return x.role == session.RoleUser || x.role == roleCompaction
}

// projected is one context entry and the messages it gives the model.
type projected struct {
	entry session.Entry
	msgs  []msg
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
		system := e.Type == session.TypeMessage && e.Message != nil && e.Message.Role == "system"
		if kept && !system {
			out = append(out, e)
		}
	}
	return append(out, path[at+1:]...)
}

// project gives each context entry its messages. Only the first entry's
// compaction contributes a summary: an older one inside the kept range is
// context-invisible, as in Pi's buildSessionProjection.
func project(ctx []session.Entry) []projected {
	out := make([]projected, len(ctx))
	for i, e := range ctx {
		out[i].entry = e
		switch {
		case e.Type == session.TypeCompaction && i == 0:
			out[i].msgs = []msg{{role: roleCompaction, summary: e.Summary}}
		case e.Type == session.TypeMessage && e.Message != nil:
			switch e.Message.Role {
			case session.RoleUser, session.RoleAssistant, session.RoleToolResult:
				out[i].msgs = []msg{{role: e.Message.Role, m: e.Message}}
			}
		}
	}
	return out
}

func entryTokens(p projected) int {
	n := 0
	for _, x := range p.msgs {
		n += x.estimate()
	}
	return n
}

// EstimateEntries is the estimated context of a path: the last valid
// assistant usage after the newest compaction, plus estimates of what
// follows it; without one, the sum of estimates (Pi's
// estimateProjectedContextTokens, with no system message, which gobble
// does not write into a session).
func EstimateEntries(path []session.Entry) int {
	proj := project(Context(path))
	var flat []msg
	var owner []string
	for _, p := range proj {
		for _, x := range p.msgs {
			flat = append(flat, x)
			owner = append(owner, p.entry.ID)
		}
	}
	latestCompaction := -1
	for i, e := range path {
		if e.Type == session.TypeCompaction {
			latestCompaction = i
		}
	}
	for i, x := range slices.Backward(flat) {
		used := assistantUsage(x)
		if used == 0 {
			continue
		}
		usageAt := -1
		for j, e := range path {
			if e.ID == owner[i] {
				usageAt = j
			}
		}
		if usageAt <= latestCompaction {
			break
		}
		for _, x := range flat[i+1:] {
			used += x.estimate()
		}
		return used
	}
	total := 0
	for _, x := range flat {
		total += x.estimate()
	}
	return total
}

// assistantUsage is an assistant message's context tokens, or 0 when it
// has none worth trusting: aborted, failed, or all zero (Pi's
// getAssistantUsage).
func assistantUsage(x msg) int {
	if x.role != session.RoleAssistant || x.m.Usage == nil {
		return 0
	}
	if x.m.StopReason == "aborted" || x.m.StopReason == "error" {
		return 0
	}
	u := x.m.Usage
	if u.TotalTokens > 0 {
		return int(u.TotalTokens)
	}
	return int(u.Input + u.Output + u.CacheRead + u.CacheWrite)
}
