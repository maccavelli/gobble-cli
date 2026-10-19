package compaction

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/session"
)

// The summary's prompts are opencode's (packages/core/src/session/
// compaction.ts at 501b15af26, SUMMARY_TEMPLATE, SUMMARY_RULES and
// buildPrompt), fitted to gobble's request, which gives the conversation
// in <conversation> and a previous summary in <previous-summary>; the User
// Messages section and the quoted instruction under Next Move are
// grok-build's ideas, in gobble's words (0005-PLAN F5-4 decision 1). The
// text keeps opencode's MIT notice: LICENSE.opencode, and NOTICE.
const (
	firstSentence = "You MUST summarize the conversation above into a structured summary that will be given to another agent to resume the work."

	sharedLead = "Summarize only what the user and the assistant said and did. Leave out instructions and setup the assistant was given rather than told by the user: repository conventions, instruction files such as AGENTS.md, and environment details like the session ID. The next agent receives current versions of all of these separately."

	summaryTemplate = `You MUST use this format for your response (you may omit sections that aren't applicable). Do not include the <template> tags in your response.
<template>
## Objective
- [one or two brief sentences describing what the user is trying to accomplish]

## Requirements
- [constraints, preferences, requirements, and scope boundaries stated by the user, or "(none)"]

## User Messages
- [every message the user sent, in order, those the previous summary lists first; each word for word, or its first 1,000 characters and "[… n more characters]" when it is longer]

## Decisions
- [decisions already made and why, or "(none)"]

## Work State
Break the objective into smaller goals and report which are completed, which are being worked on, and which are blocked.
### Completed
- [goals that have been completed; otherwise "(none)"]

### Active
- [goals currently being worked on; otherwise "(none)"]

### Blocked
- [anything blocking progress, and why; otherwise "(none)"]

## Next Move
Latest instruction: "[the user's latest instruction that the first action below carries out, quoted word for word, or (none)]"
1. [ordered list of next actions, or "(none)"]

## Relevant Files
List the files and directories, other than the current working directory, that another agent would need to open to continue this work. Include at most 15, most important first. Do not list every file that was read or changed. Include paths outside the current working directory when relevant. If none, write "(none)".
- ` + "`[file or directory path]`" + `: [brief reason it matters]

## Important Context
- [facts the next agent cannot continue without and cannot easily find on its own; or "(none)"]
</template>`

	summaryRules = `Rules:
- Keep each section concise. Use terse, single-line bullets, not prose paragraphs or nested lists.
- Prefer short references over detailed restatement. It is fine to leave out information the next agent can recover from the code or the files listed above.
- Preserve exact file paths, symbols, commands, error strings, URLs, and identifiers.
- Carry forward only user questions or requests that remain unanswered or require further action. Do not repeat ones that newer history has answered or resolved. Preserve exact wording when carrying one forward.
- Preserve consequential workflow state, including whether changes are uncommitted, committed, pushed, under review, or merged.
- Do not mention the summary process or that context was compacted.`

	noTools    = "Do not continue the task or call tools."
	onlyOutput = "Return only the structured summary in the requested format. Do not include a preamble, explanation, or other commentary."

	updateSentence = "Update the existing checkpoint in the <previous-summary> block above into one consolidated summary."

	legacySentence = "The existing checkpoint was written with an earlier format that recorded far more detail than this one asks for. Rewrite it at the level of detail described below rather than carrying its detail forward. Keep its requirements, decisions, and open questions; they came from earlier conversation with the user."

	precedence = "Newer history always takes precedence over the existing checkpoint. Preserve previous information unless newer history clearly contradicts, supersedes, resolves, or makes it stale. If something is no longer relevant to continuing the work, you may remove it."

	reconcile = "Incorporate newer requirements, decisions, progress, and context. Reconcile Work State and Next Move: move completed work out of Active, remove resolved blockers and answered questions, and preserve unresolved or pending work."

	noWrappers = "Return only the updated Markdown sections. Do not reproduce the `<previous-summary>` or `<summary>` wrapper tags from the previous checkpoint."
)

// shared are the instructions both summaries end with.
var shared = []string{sharedLead, summaryTemplate, summaryRules, noTools, onlyOutput}

// summaryPrompt is the first summary's prompt.
var summaryPrompt = strings.Join(append([]string{firstSentence}, shared...), "\n\n")

// updatePrompt is an update's prompt; legacy, for a previous summary in
// Pi's format, asks for it to be rewritten in the new headings, as
// opencode rewrites its own older checkpoints.
func updatePrompt(legacy bool) string {
	parts := []string{updateSentence}
	if legacy {
		parts = append(parts, legacySentence)
	}
	parts = append(parts, precedence, reconcile, noWrappers)
	return strings.Join(append(parts, shared...), "\n\n")
}

// piFormat reports a summary in Pi's format, which a compaction before
// 0005-PLAN F5-4 wrote: a Goal heading and no Objective.
func piFormat(summary string) bool {
	return strings.Contains(summary, "## Goal") && !strings.Contains(summary, "## Objective")
}

// The prompt ids, as <config>/personalities/prompts/<id>.md names them
// (0005-PLAN F5-4 decision 8; 0022-MADR D15).
const (
	PromptSystem     = "compaction.system"
	PromptSummary    = "compaction.summary"
	PromptUpdate     = "compaction.update"
	PromptTurnPrefix = "compaction.turn_prefix"
)

// PromptIDs are every compaction prompt a user's file may replace.
func PromptIDs() []string {
	return []string{PromptSystem, PromptSummary, PromptUpdate, PromptTurnPrefix}
}

// Prompts are the user's texts for compaction's prompts, each in place of
// the whole of gobble's; an empty one is gobble's.
type Prompts struct {
	System, Summary, Update, TurnPrefix string
}

// Set puts text, checked by CheckPrompt, at id.
func (ps *Prompts) Set(id, text string) error {
	if err := CheckPrompt(id, text); err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	switch id {
	case PromptSystem:
		ps.System = text
	case PromptSummary:
		ps.Summary = text
	case PromptUpdate:
		ps.Update = text
	case PromptTurnPrefix:
		ps.TurnPrefix = text
	}
	return nil
}

// CheckPrompt checks the user's text for a compaction prompt: it has
// words, closes none of the blocks the code puts around the conversation,
// and, for an update, names the <previous-summary> block the code fills
// (0005-PLAN F5-4 decision 8).
func CheckPrompt(id, text string) error {
	switch {
	case !slices.Contains(PromptIDs(), id):
		return fmt.Errorf("no compaction prompt is called %s", id)
	case strings.TrimSpace(text) == "":
		return errors.New("it is empty")
	case strings.Contains(text, "</conversation>"), strings.Contains(text, "</previous-summary>"):
		return errors.New("it closes a block the code puts around the conversation, </conversation> or </previous-summary>")
	case id == PromptUpdate && !strings.Contains(text, "<previous-summary>"):
		return errors.New("an update's prompt names the <previous-summary> block, which holds the summary to update")
	}
	return nil
}

// Request is what a compaction is asked to do beyond its preparation: the
// user's focus, the session's mode for the reminder, and the user's
// prompts.
type Request struct {
	Instructions string
	PlanMode     bool
	Prompts      Prompts
}

func (r Request) system() string { return or(r.Prompts.System, systemPrompt) }

func (r Request) turnPrefix() string { return or(r.Prompts.TurnPrefix, turnPrefixPrompt) }

// base is the history request's prompt: the user's, or gobble's first or
// update prompt.
func (r Request) base(previous string) string {
	if previous == "" {
		return or(r.Prompts.Summary, summaryPrompt)
	}
	return or(r.Prompts.Update, updatePrompt(piFormat(previous)))
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// controlTags are the tags of gobble's blocks a summary's text could
// close or open (0005-PLAN F5-4 decision 6).
var controlTags = []string{"summary", "analysis", "conversation", "previous-summary", "turn-request", "todo-list", "mode",
	"read-files", "modified-files"}

// neutralize puts a zero-width space after the "<" of each of gobble's
// tags in text, so text written into a summary cannot close its block
// (grok-build's neutralize_compaction_control_tokens, gobble's tags).
func neutralize(text string) string {
	for _, t := range controlTags {
		text = strings.ReplaceAll(text, "</"+t+">", "<\u200b/"+t+">")
		text = strings.ReplaceAll(text, "<"+t+">", "<\u200b"+t+">")
	}
	return text
}

// cleanSummary is a model's summary without a leading <analysis> block, a
// scratchpad, and without an outer <summary> block around it (grok-build's
// format_compact_summary). An unclosed leading <analysis> leaves nothing.
func cleanSummary(text string) string {
	t := strings.TrimSpace(text)
	for strings.HasPrefix(t, "<analysis>") {
		end := strings.Index(t, "</analysis>")
		if end < 0 {
			return ""
		}
		t = strings.TrimSpace(t[end+len("</analysis>"):])
	}
	if inner, ok := strings.CutPrefix(t, "<summary>"); ok {
		if inner, ok := strings.CutSuffix(inner, "</summary>"); ok {
			t = strings.TrimSpace(inner)
		}
	}
	return t
}

// todoItem is a gobble.todo entry's item, as tool.PlanItem writes it.
type todoItem struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

// todoList is the path's last todo list as /todos prints it: [x]
// completed, [>] in progress, [ ] pending, one item per line; "" when
// there is none, or it is empty.
func todoList(path []session.Entry) string {
	var last []todoItem
	for _, e := range path {
		if e.Type != session.TypeCustom || e.CustomType != session.CustomTodo {
			continue
		}
		var d struct {
			Todos []todoItem `json:"todos"`
		}
		if json.Unmarshal(e.Data, &d) == nil {
			last = d.Todos
		}
	}
	lines := make([]string, len(last))
	for i, it := range last {
		mark := "[ ]"
		switch it.Status {
		case "completed":
			mark = "[x]"
		case "in_progress":
			mark = "[>]"
		}
		lines[i] = mark + " " + it.Content
	}
	return strings.Join(lines, "\n")
}

// reminder is the live state a compaction's text ends with (0005-PLAN F5-4
// decision 5): the todo list, and plan mode.
func reminder(todos string, plan bool) string {
	var b strings.Builder
	if todos != "" {
		b.WriteString("\n\n<todo-list>\n" + neutralize(todos) + "\n</todo-list>")
	}
	if plan {
		b.WriteString("\n\n<mode>plan</mode>")
	}
	return b.String()
}
