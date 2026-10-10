package compaction

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/maccavelli/gobble-cli/session"
)

// The prompts are Pi's, word for word (compaction.ts and utils.ts at
// 312184edb).
const (
	systemPrompt = `You are a context summarization assistant. Your task is to read a conversation between a user and an AI assistant, then produce a structured summary following the exact format specified.

Do NOT continue the conversation. Do NOT respond to any questions in the conversation. ONLY output the structured summary.`

	summarizationPrompt = `The messages above are a conversation to summarize. Create a structured context checkpoint summary that another LLM will use to continue the work.

Use this EXACT format:

## Goal
[What is the user trying to accomplish? Can be multiple items if the session covers different tasks.]

## Constraints & Preferences
- [Any constraints, preferences, or requirements mentioned by user]
- [Or "(none)" if none were mentioned]

## Progress
### Done
- [x] [Completed tasks/changes]

### In Progress
- [ ] [Current work]

### Blocked
- [Issues preventing progress, if any]

## Key Decisions
- **[Decision]**: [Brief rationale]

## Next Steps
1. [Ordered list of what should happen next]

## Critical Context
- [Any data, examples, or references needed to continue]
- [Or "(none)" if not applicable]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

	updateInstructions = `Update the existing structured summary with new information. RULES:
- PRESERVE all existing information from the previous summary
- ADD new progress, decisions, and context from the new messages
- UPDATE the Progress section: move items from "In Progress" to "Done" when completed
- UPDATE "Next Steps" based on what was accomplished
- PRESERVE exact file paths, function names, and error messages
- If something is no longer relevant, you may remove it

Use this EXACT format:

## Goal
[Preserve existing goals, add new ones if the task expanded]

## Constraints & Preferences
- [Preserve existing, add new ones discovered]

## Progress
### Done
- [x] [Include previously done items AND newly completed items]

### In Progress
- [ ] [Current work - update based on progress]

### Blocked
- [Current blockers - remove if resolved]

## Key Decisions
- **[Decision]**: [Brief rationale] (preserve all previous, add new)

## Next Steps
1. [Update based on current state]

## Critical Context
- [Preserve important context, add new if needed]

Keep each section concise. Preserve exact file paths, function names, and error messages.`

	updatePrompt = "The messages above are NEW conversation messages to incorporate into the existing summary provided in <previous-summary> tags.\n\n" + updateInstructions

	turnPrefixPrompt = `The messages above are earlier context from an ongoing conversation. Later messages are stored separately and do not need to be reconstructed.

Create a concise checkpoint of the user's request and the progress shown above. This checkpoint will be placed before the later messages so the conversation can continue with the necessary context.

## Original Request
[What did the user ask for?]

## Progress So Far
- [Key decisions and work completed in these messages]

## Context Needed to Continue
- [Information from these messages needed to understand the later work]

Only summarize information explicitly present above. Do not infer or recreate later messages.`

	summaryPrefix = "The conversation history before this point was compacted into the following summary:\n\n<summary>\n"
	summarySuffix = "\n</summary>"

	branchSummaryPrefix = "The following is a summary of a branch that this conversation came back from:\n\n<summary>\n"
	branchSummarySuffix = "</summary>"
)

// toolResultMaxChars is where a tool result is cut when serialised.
const toolResultMaxChars = 2000

// SummaryText is a compaction's summary as the model reads it: Pi's
// compactionSummary message, a user message (messages.ts convertToLlm).
func SummaryText(summary string) string { return summaryPrefix + summary + summarySuffix }

// BranchSummaryText is a branch summary as the model reads it, Pi's
// branchSummary message as a user message.
func BranchSummaryText(summary string) string {
	return branchSummaryPrefix + summary + branchSummarySuffix
}

// Convert is one projected message as the model receives it, Pi's
// convertToLlm (messages.ts:148-196): a user, assistant or tool-result
// message is itself; a custom message is a user message of its content; a
// branch or compaction summary is a user message of its prefixed text; a
// bash execution is a user message of its text, or nil when it is excluded
// from the context. Any other role is nil.
func Convert(m *session.Message) *session.Message {
	text := func(s string) *session.Message {
		u := &session.Message{Role: session.RoleUser, Timestamp: m.Timestamp}
		_ = u.SetBlocks([]session.Block{{Type: session.BlockText, Text: s}}) //nolint:errcheck // a text block always encodes
		return u
	}
	switch m.Role {
	case session.RoleUser, session.RoleAssistant, session.RoleToolResult:
		return m
	case session.RoleCustom:
		u := &session.Message{Role: session.RoleUser, Content: m.Content, Timestamp: m.Timestamp}
		if blocks, err := m.Blocks(); err == nil {
			_ = u.SetBlocks(blocks) //nolint:errcheck // blocks just decoded encode
		}
		return u
	case session.RoleBranchSummary:
		return text(BranchSummaryText(m.Summary))
	case session.RoleCompactionSummary:
		return text(SummaryText(m.Summary))
	case session.RoleBashExecution:
		if m.ExcludeFromContext != nil && *m.ExcludeFromContext {
			return nil
		}
		return text(BashText(m))
	}
	return nil
}

// BashText is a bash execution as the model reads it (Pi's
// bashExecutionToText).
func BashText(m *session.Message) string {
	text := "Ran `" + m.Command + "`\n"
	if m.Output != nil && *m.Output != "" {
		text += "```\n" + *m.Output + "\n```"
	} else {
		text += "(no output)"
	}
	var code float64
	exited := len(m.ExitCode) > 0 && json.Unmarshal(m.ExitCode, &code) == nil
	switch {
	case m.Cancelled != nil && *m.Cancelled:
		text += "\n\n(command cancelled)"
	case exited && code != 0:
		text += "\n\nCommand exited with code " + strconv.FormatFloat(code, 'f', -1, 64)
	}
	if m.Truncated != nil && *m.Truncated && m.FullOutputPath != "" {
		text += "\n\n[Output truncated. Full output: " + m.FullOutputPath + "]"
	}
	return text
}

// textOf joins a message's text blocks with sep, as pi-ai's contentText.
func textOf(blocks []session.Block, sep string) string {
	var parts []string
	for _, b := range blocks {
		if b.Type == session.BlockText {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, sep)
}

// truncate keeps the first max characters, as Pi's truncateForSummary,
// counting UTF-16 code units.
func truncate(s string, maxChars int) string {
	n := utf16Len(s)
	if n <= maxChars {
		return s
	}
	units := 0
	cut := len(s)
	for i, r := range s {
		w := 1
		if r > 0xFFFF {
			w = 2
		}
		if units+w > maxChars {
			cut = i
			break
		}
		units += w
	}
	return fmt.Sprintf("%s\n\n[... %d more characters truncated]", s[:cut], n-maxChars)
}

// serialize is Pi's serializeConversation of convertToLlm's messages: the
// messages as tagged text, so the model summarises rather than continues
// them.
func serialize(msgs []*session.Message) string {
	var parts []string
	for _, m := range msgs {
		m = Convert(m)
		if m == nil {
			continue
		}
		blocks, err := m.Blocks()
		if err != nil {
			continue
		}
		switch m.Role {
		case session.RoleUser:
			if t := textOf(blocks, ""); t != "" {
				parts = append(parts, "[User]: "+t)
			}
		case session.RoleAssistant:
			var thinking, calls []string
			hasText := false
			for _, b := range blocks {
				switch b.Type {
				case session.BlockThinking:
					thinking = append(thinking, b.Thinking)
				case session.BlockToolCall:
					calls = append(calls, b.Name+"("+callArgs(b.Arguments)+")")
				case session.BlockText:
					hasText = true
				}
			}
			if len(thinking) > 0 {
				parts = append(parts, "[Assistant thinking]: "+strings.Join(thinking, "\n"))
			}
			if hasText {
				parts = append(parts, "[Assistant]: "+textOf(blocks, "\n"))
			}
			if len(calls) > 0 {
				parts = append(parts, "[Assistant tool calls]: "+strings.Join(calls, "; "))
			}
		case session.RoleToolResult:
			if t := textOf(blocks, ""); t != "" {
				parts = append(parts, "[Tool result]: "+truncate(t, toolResultMaxChars))
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

// callArgs is k=JSON(v) for each argument, in the order written, joined by
// ", ".
func callArgs(args jsontext.Value) string {
	ms, err := members(args)
	if err != nil {
		return ""
	}
	out := make([]string, len(ms))
	for i, m := range ms {
		out[i] = m.name + "=" + compactJSON(m.value)
	}
	return strings.Join(out, ", ")
}

type member struct {
	name  string
	value jsontext.Value
}

// members are an object's members in order.
func members(v jsontext.Value) ([]member, error) {
	d := jsontext.NewDecoder(strings.NewReader(string(v)))
	if tok, err := d.ReadToken(); err != nil || tok.Kind() != '{' {
		return nil, fmt.Errorf("not an object: %w", err)
	}
	var out []member
	for d.PeekKind() != '}' {
		tok, err := d.ReadToken()
		if err != nil {
			return nil, err
		}
		name := tok.String()
		val, err := d.ReadValue()
		if err != nil {
			return nil, err
		}
		out = append(out, member{name: name, value: val.Clone()})
	}
	return out, nil
}

// fileOps are the files the summarised calls read and changed.
type fileOps struct {
	read, written, edited map[string]bool
}

func newFileOps() fileOps {
	return fileOps{read: map[string]bool{}, written: map[string]bool{}, edited: map[string]bool{}}
}

// add records the read, write and edit calls of an assistant message
// (Pi's extractFileOpsFromMessage).
func (f fileOps) add(m *session.Message) {
	if m == nil || m.Role != session.RoleAssistant {
		return
	}
	blocks, err := m.Blocks()
	if err != nil {
		return
	}
	for _, b := range blocks {
		if b.Type != session.BlockToolCall {
			continue
		}
		var a struct {
			Path *string `json:"path"`
		}
		if json.Unmarshal(b.Arguments, &a) != nil || a.Path == nil || *a.Path == "" {
			continue
		}
		switch b.Name {
		case "read":
			f.read[*a.Path] = true
		case "write":
			f.written[*a.Path] = true
		case "edit":
			f.edited[*a.Path] = true
		}
	}
}

// Details are a compaction's file lists, Pi's CompactionDetails.
type Details struct {
	ReadFiles     []string `json:"readFiles"`
	ModifiedFiles []string `json:"modifiedFiles"`
}

// lists are the files only read, and the files changed, each sorted (Pi's
// computeFileLists).
func (f fileOps) lists() Details {
	modified := map[string]bool{}
	for p := range f.edited {
		modified[p] = true
	}
	for p := range f.written {
		modified[p] = true
	}
	d := Details{ReadFiles: []string{}, ModifiedFiles: []string{}}
	for p := range f.read {
		if !modified[p] {
			d.ReadFiles = append(d.ReadFiles, p)
		}
	}
	for p := range modified {
		d.ModifiedFiles = append(d.ModifiedFiles, p)
	}
	slices.Sort(d.ReadFiles)
	slices.Sort(d.ModifiedFiles)
	return d
}

// format is Pi's formatFileOperations: the lists as tagged blocks after a
// blank line, or nothing.
func (d Details) format() string {
	var sections []string
	if len(d.ReadFiles) > 0 {
		sections = append(sections, "<read-files>\n"+strings.Join(d.ReadFiles, "\n")+"\n</read-files>")
	}
	if len(d.ModifiedFiles) > 0 {
		sections = append(sections, "<modified-files>\n"+strings.Join(d.ModifiedFiles, "\n")+"\n</modified-files>")
	}
	if len(sections) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(sections, "\n\n")
}
