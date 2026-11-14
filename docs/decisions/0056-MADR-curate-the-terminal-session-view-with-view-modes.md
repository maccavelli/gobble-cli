---
status: accepted
date: 2026-11-13
decision-makers: repository owner
consulted: "the owner's words of 2026-11-12 and 2026-11-13, quoted below; this repository at f078889 (main, level with origin/main, 2026-11-13), read for the audit below; a sample turn captured on 2026-11-13 from a scratch clone at f078889 with a throwaway test (never committed) that drove acpserver, through acpclient, into the line session's view, with llmtest's scripted model and no provider; go-tui-lib v2.12.0's inline, inline/inlinetest, transcript, frame and stream packages, read from the module cache; a read-only feasibility study of the live group row, 2026-11-13, under More Information; the owner's rulings of 2026-11-13 on Q1 to Q6 and on the type-ahead question, relayed by the coordinator; the owner's ruling of 2026-11-13, \"Shared layer + go-tui-lib survey\", relayed by the coordinator, with this repository at 8fc4233 (main, clean) read for the amendment of that date; a read-only survey of go-tui-lib at its tag v2.12.0 against F9, 2026-11-13, relayed by the coordinator, under More Information"
informed: "`0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` and `0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`, whose v0.7.0 lands first and whose P4 edits internal/cli/view.go; magic-cli-remote, whose ACP stream this record leaves unchanged"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Curate the terminal client's session view: group actions, show each exit status once, fold thinking, and offer `/view raw`, `verbose`, `default` and `terse`, with the ACP stream left raw

## Context and Problem Statement

gobble's native CLI mode is an ACP client of gobble's own agent, in process
([0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md), "CLI as ACP client";
[0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D5). It draws every ACP update it receives, one after
another, close to how the update arrives. The phone (magic-cli-remote) gets the same stream from `gobble acp` and
formats it itself.

The owner, 2026-11-12: "Gobble running as a cli in a terminal is its own client. We will need to visit this in the
future because right now the output in the cli session chat is beyond chaotic and disorganized."

The owner, 2026-11-13: "A few things I noticed about the gobble session chat output noise, shell/commands output exit
codes. Entire commands printed to session chat. Each action as its own message instead of printing something like ran
a command, then +2 +3 +4 commands on a single line. Printing thinking/thought on a line. Every action taken is
printed, in full, on its own line, nothing is cultured or curated. For the phone app, it formats client session output
itself so it should get the full raw output over acp but for the gobble cli client it needs to provide formatting and
easy to read and follow session chat language and status. This could be selectable as a mode for view, as /view raw,
verbose, default, and terse. Ideas to consider."

The owner chose on 2026-11-13 "Audit + draft record now", with the code to land after gobble `v0.7.0`, because
[0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
and [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
change the sign-in surfaces, and `0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`'s P4 edits the same view code
([0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md) D2).

The question this record answers: how should the terminal client present a turn, per event, in each of four view
modes the owner named, so that it reads as a curated session, while every ACP client, the phone first, keeps
receiving the raw and complete stream?

### The audit: what a turn prints at `f078889` (read 2026-11-13; facts, no conclusions)

**The path.** The line session starts gobble's agent in process (`internal/cli/agent.go:249-260`, `agentServe`) over
acpclient (`internal/cli/agent.go:267-280`), with `Params` off and `Compaction` on (`internal/cli/line.go:127`). Each
update reaches `lineSession.onUpdate` (`internal/cli/line.go:416-438`); `_gobble/personality` changes the voice there
and stops (`:421-431`); every other update goes to the running turn's `view.on` (`internal/cli/view.go:116-191`),
one `switch` on `acpclient.Kind` (`acpclient/update.go:19-44`). Between turns there is no view
(`internal/cli/line.go:348-355`), so an update then is drawn by no one. The turn's spinner runs on stderr
(`internal/cli/line.go:341-346`) and stops at the view's first write, once; it never restarts within the turn
(`internal/cli/view.go:436-442`). A reply is written to stdout; status, retry and compaction lines to stderr. No live
region is drawn while a turn runs (`internal/cli/view.go:554-559`; `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b).

**What the agent sends.** acpserver translates each agent event in order, flushing coalesced text and thought before
any other event, "so the order on the wire is the order of events" (`acpserver/prompt.go:32-35`, `:88-139`). A tool
call is three updates: `tool_call` with status `pending`, the tool's kind and its raw input (`:96-105`);
`tool_call_update` `in_progress` (`:111-114`); and `tool_call_update` `completed` or `failed` with its content
(`:115-136`). A title is at most 60 runes (`maxTitle`, `:26`; `cutTitle`, `:231`); a finished call's content is a
summary of at most 400 bytes first, then its diffs, or else its whole output when that differs from the summary
(`resultContent`, `:197-215`; `maxSummary`, `:27`). These are the phone-legible rules of
[0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D19 item 3. A shell call's title is `run <the whole
command>` (`tool/builtin/shell.go:316-322`), its output ends with a line `[exit status N]` (`:194`), and its summary
is `exit status N: <the output's first line>` (`:198`).

**What acpclient keeps.** `acpclient.ToolCall` carries `ID`, `Title`, `Status` and `Content` alone
(`acpclient/update.go:216-221`): the tool's kind, its raw input and its locations, which the agent sends
(`acpserver/prompt.go:98-104`), are dropped in translation (`acpclient/update.go:284-297`). A content block that is
not text becomes `[image]`, `[audio]` and the like (`:355-375`). The CLI client offers the compaction capability
alone (`acpclient/conn.go:160`): no terminal, no notices, no elicitation.

| Update | Where the view draws it | What it prints today | The noise it causes |
| :--- | :--- | :--- | :--- |
| `agent_message_chunk`, text | `view.go:122-128`, `reply` `:193-203`, `text` `:316-334` | the reply, through go-tui-lib's `stream.Printer`: Markdown styled when colour is on, wrapped to at most 120 columns | none of its own; the reply is the signal, but it starts on the line after the last tool output with no gap |
| `agent_message_chunk`, not text | `view.go:124-126` | `[image]` (or `[audio]`, `[resource_link]`, `[resource]`) on its own line | small |
| `agent_thought_chunk` | `view.go:129-142` | nothing by default; with `--show-thinking`, the thought as it streams, dimmed when colour is on, wrapped, with no label; a blank line is not put between it and the reply that follows (`reply`, `:196-201`, adds only a line end) | with colour off a thought cannot be told from the reply; each thought burst between calls is its own unlabelled run of lines |
| `tool_call` (`pending`) | `view.go:143-148`, `header` `:88-95`, `render.ToolHeader` `render/tool.go:32-42` | `▸ <title> …`: the title cleaned, on one line, truncated to the terminal width; for a shell call the command itself, up to 60 runes | one line per call before it runs |
| `tool_call_update` (`in_progress`) | `view.go:149-163` | nothing: its header equals the pending one, and `header` prints only a changed header (`:83-95`) | none |
| `tool_call_update` (`completed`, `failed`) | `view.go:149-163`, `content` `:280-304` | a second header line, `▸ <title> ✓` or `✗`, then **every** content block: the summary, then the output (more than 20 lines shows the first 10 and the last 10 and `… N lines hidden (--verbose shows all)`, `render/tool.go:23-28`, `:59-78`), or each diff as a coloured unified diff, or `terminal <id>` | each call takes at least two header lines; a shell call's summary repeats its output's first line, and its exit status is shown twice (`exit status 0: …` and `[exit status 0]`); a read shows the file's text, a grep its matches; nothing groups consecutive calls |
| `plan` | `view.go:164-172`, `render.Plan` `render/plan.go:26-43` | the whole checklist, every time it changes | the todo tool's summary line above it says the same thing in words |
| `usage_update` | `view.go:173-175` | nothing until the turn ends; then `finish` (`:444-468`) writes the status line on stderr: elapsed, a 20-cell bar, percent, tokens, cost (`render/status.go:25-38`), and a stop reason other than `end_turn` (`view.go:491-493`) | none |
| `_gobble/retry` | `view.go:176-179`, `retryLine` `:242-254` | a note on stderr at a wait's start | none; `0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` P4 step 4b changes its comparisons at `:247` and `:249` |
| compaction (`_gobble/compaction`, `compaction_update`, `compaction_summary_chunk`) | `view.go:180-188` | a manual compaction's start, completion with summary, or failure, as reply text (`manualCompaction`, `:212-225`); an automatic one's start and failure as a note on stderr (`compactionLine`, `:261-278`) | none |
| `_gobble/personality` | `line.go:421-431` | nothing; the voice of later lines changes | none |
| `available_commands_update`, `current_mode_update`, `session_info_update` (the title) | `view.go:121-190` has no case for them | nothing; acpclient keeps the commands for `/help` and completion | none |
| `config_option_update`, `user_message_chunk`, any newer variant (`KindOther`) | `view.go:189` | nothing | none |
| `notice` | not reached: the CLI does not offer `session.notices`, so the agent keeps none for it (`acpserver/notice.go:22-38`) | nothing | none |
| a permission request | `lineSession.askPermission`, `internal/cli/permission.go:247`; the question's title `:87` | go-tui-lib's `ask` prompt, titled with the call's **full** title (`FullTitle`, not the 60-rune one) | for a shell call the whole command, however long |

**The display flags** (`internal/cli/flags.go:27-29`; [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md)
D12) reach the view as `display{showThinking, verbose, stats}` (`internal/cli/line.go:91-94`, `view.go:21-30`):

* `--show-thinking` draws thoughts (`view.go:131`); print mode never draws them (`internal/cli/print.go:88-89`).
* `--verbose` lifts the 20-line bound on tool output and diffs (`view.go:295`, `:301`); every built-in personality's
  hidden-lines text names it (`personality/builtin/boring/messages.md:58` and the three others).
* `--stats` adds time to first token and tokens per second to the status line (`view.go:484-490`,
  `render/status.go:43-49`), in the line session and in print mode (`internal/cli/print.go:232-237`).
* `--log-level` (`internal/cli/root.go:21`) is the level of `<state>/logs/gobble.log` (`internal/cli/main.go:86`,
  `:314-327`). The terminal shows warnings and errors only, at `max(level, warn)`
  (`internal/logging/logging.go:12`, `:53-59`), so `debug` and `info` never reach the session; `error` hides the
  warnings there too. It is independent of the three above.

**Slash commands.** One registry (`0008-MADR-native-cli-mode.md` D13): the line session's local commands, `help`, `exit`, `quit`, `edit`,
`new`, `login`, `logout` (`internal/cli/slash.go:22-30`), united with the agent's advertised ones
(`registry`, `:42-56`; the agent's names at `acpserver/commands.go:31-48`). `slash` handles a local one and refuses an
unknown one locally (`:77-126`). No command is named `view`, in either list. `/thinking` is the agent's, and sets the
model's thinking level, not its display.

**The ACP stream is not the view.** `gobble acp` serves `acpserver.Serve` straight onto stdout
(`internal/cli/acp.go:53`); no view code runs. In the line session the view consumes acpclient's translated updates
after they arrive; nothing the view does is sent back, and acpserver imports nothing from `internal/cli`.
`-p --output-format stream-json` writes each update's params as the SDK encodes them (`internal/cli/print.go:114-122`),
also without the view. A change confined to `internal/cli` and acpclient's translation cannot change a byte the
agent writes.

### A sample turn, as the default view prints it today

Captured 2026-11-13 from the scratch clone: seven tool calls (three shell calls, one failing, one with 40 lines of
output; a read; an edit; a todo list; a grep), with a thought before three of them, then the reply. Width 100, no
colour; the working directory shown as `<cwd>`, and the home directory set to it so paths show as `~/`. The middle
of the 40-line output is elided here as `[…]`.

```text
▸ run ls && printf 'one\ntwo\n' > out.txt && cat out.txt …
▸ run ls && printf 'one\ntwo\n' > out.txt && cat out.txt ✓
exit status 0: notes.md
notes.md
one
two
[exit status 0]
▸ run seq 1 40 …
▸ run seq 1 40 ✓
exit status 0: 1
1
2
[…]
10
… 21 lines hidden (--verbose shows all)
32
[…]
40
[exit status 0]
▸ run test -f missing.txt …
▸ run test -f missing.txt ✗
exit status 1:
[exit status 1]
▸ read notes.md …
▸ read notes.md ✓
read notes.md (3 lines)
1: alpha
2: beta
3: gamma
(end of file, 3 lines)
▸ edit notes.md …
▸ edit notes.md ✓
edited notes.md: 1 replacement
--- ~/notes.md
+++ ~/notes.md
@@ -1,3 +1,3 @@
 alpha
-beta
+BETA
 gamma
▸ todo …
▸ todo ✓
todo list updated: 1 completed, 1 in progress, 1 pending
  ✓ look around
  ▸ fix notes
  • run tests
▸ grep alpha in . …
▸ grep alpha in . ✓
grep .: 1 match in 1 file
notes.md:1: alpha
I listed the files, ran **seq**, and fixed `notes.md`.

Done.
--- stderr
1s
```

Seven actions took 62 lines before the reply's first; 14 of them are headers, two per call. With
`--show-thinking --verbose --stats` the same turn took 86, the three thoughts printed as plain lines (for example
`That failed check needs a look.` directly above `▸ run test -f missing.txt …`) and all 41 lines of `seq`'s output
shown. The updates received, in order: `available_commands_update`, `current_mode_update`, `usage_update` and
`_gobble/personality` at `session/new`; then per call `tool_call`, `tool_call_update`, `tool_call_update`, with
`agent_thought_chunk` before the first, third and last, and `plan` after the todo call; then `agent_message_chunk`
and `usage_update`.

## Decision Drivers

* **The ACP stream stays raw and complete.** The phone formats what it receives; it must keep receiving every update,
  byte for byte, whatever the terminal shows (the owner's words; `0008-MADR-native-cli-mode.md` D19).
* **A turn reads as a session, not a log.** One line per group of like actions, the reply set apart, failures shown
  with their cause, each exit status said once.
* **Nothing is lost, only folded.** Every mode below `raw` can be raised to `verbose` with one command, and the hidden
  detail is still in the session file and on the wire.
* **One way to choose.** The owner named four modes and `/view`; the existing flags must not become a second,
  overlapping control.
* **go-tui-lib owns terminal mechanics.** gobble does not reimplement a behaviour go-tui-lib has, and waits where it
  has none ([0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md); AGENTS.md).
* **The resize reprint keeps working.** Whatever the view writes is kept by the transcript and drawn again at a
  resize ([0030-MADR-resize-and-performance.md](0030-MADR-resize-and-performance.md) D3, D4; `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14).
* **The owner is the only user.** No compatibility shims for retired flags unless asked.

## Considered Options

* **A. Curate in the client: four view modes, groups written once when they close, each exit status once, thinking
  folded; the ACP stream untouched.**
* **B. As A, but the open group is a live row redrawn in place while it grows, through go-tui-lib's `inline.Region`.**
* **C. Curate on the server: shorter titles, summaries without output, fewer updates.**
* **D. Keep today's view; fix the duplicates only (one header per call, the summary or the output, not both).**

## Decision Outcome

*(2026-11-13, the owner's ruling on Q1, "Staged hybrid". The draft's choice of A is kept below, struck through, with
its reasoning; stage 1 is A as written.)*

*(2026-11-13, the owner's ruling "Shared layer + go-tui-lib survey": the curation that D2 to D5 describe is a
renderer-neutral layer, `internal/curate`, which the line view and the future TUI mode both draw from, and `/view`
applies to both. See "Amendment 2026-11-13: a renderer-neutral curation layer", D11 to D16, after D10. The chosen
option and D1 to D10 stand, each with a dated note where the amendment changes where it lives.)*

Chosen option: "B. As A, but the open group is a live row redrawn in place while it grows, through go-tui-lib's
`inline.Region`", **staged**:

* **Stage 1** (`0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P3) is A as written: one stdout line per group, written when the group closes, while the
  spinner's label shows the open group growing (`ran go build ./... ✓ +1 +2 · go test ./...`), cut to the width.
* **Stage 2** (`0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P3b) replaces the spinner with a status row drawn through `inline.Region`, to which every
  other writer yields (D10). When the group closes, the row collapses to stage 1's one line; when stderr is not a
  terminal, behaviour stays plain A.

The owner chose it after the feasibility study under More Information found B feasible with conditions. Staging puts
the growing row on screen at stage 1's risk, and gives the new drawing, its record amendments and its open question
(Q7) a phase of their own. The ACP stream is untouched in both stages (D1).

~~Chosen option: "A. Curate in the client: four view modes, groups written once when they close, each exit status
once, thinking folded; the ACP stream untouched", because it meets every driver: the phone's stream does not change
by a byte, the default reads one line per group of actions as the owner described, and it needs no new terminal
mechanics, so the mid-turn reprint of `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b stays as it is. B gives a livelier row at the cost of a
region drawn mid-turn, which `0030-MADR-resize-and-performance.md` D3 and that D14 step 5b avoided on purpose; it is Q1's option (b).~~ C breaks the first driver. D
leaves "every action, in full, on its own line".

This record is `accepted` (2026-11-13, the owner). The owner ruled on Q1 to Q6 that day; each ruling is a dated note
under its question below, and D1 to D10 are the decisions. **One question stays open:** Q7, how the live row meets
type-ahead echo, owned by `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P3b and decided there with the screen tests in hand (the owner's ruling of
2026-11-13, "Decide in stage 2"); `0024-MADR-cli-mode-polish.md` D1 stands until then. *(2026-11-13, the amendment:
Q8 to Q10, below Q7, are open too; they concern the TUI mode and go-tui-lib, and none blocks
`0056-PLAN-curate-the-terminal-session-view-with-view-modes.md`.)* *(2026-11-13, later: the owner ruled on Q8 to
Q10, each a dated note under its question; Q7 alone stays open.)* Nothing is built before the owner approves the
execution of
[0056-PLAN-curate-the-terminal-session-view-with-view-modes.md](0056-PLAN-curate-the-terminal-session-view-with-view-modes.md),
and that plan starts only after `v0.7.0` is tagged.

### D1. The ACP stream is the agent's, and no view mode changes it

* Nothing under `acpserver/`, `agent/`, `tool/`, `acpext/` or `llm/` changes for this record. The 60-rune title, the
  400-byte summary first, and the three updates per call stay as `0008-MADR-native-cli-mode.md` D19 has them; `acpserver/testdata/session.golden`
  does not change.
* The view reads only what acpclient gives it. acpclient's translation gains the tool call's kind (D3), read from
  updates the agent already sends; acpclient's handshake does not change. *(2026-11-13, the amendment, D12: the
  curation layer reads what acpclient gives it, and the view reads the layer; acpclient also gains `Update.Method`,
  the method it already receives for a `_gobble/*` notification.)*
* A test runs one scripted turn through the line session in each of the four modes and finds the bytes each side
  wrote, client to agent and agent to client, equal across the modes.

### D2. Four modes, chosen per event

`default` is the mode a session starts in. The modes differ only in what the terminal shows:

*(2026-11-13, the amendment, D14: what this table decides per event, whether an item is made, how much of it is
shown, what it groups with and what closes it, is the curation layer's contract, restated there in the layer's terms;
the cells' typography below, a JSON line, dimmed, styled and wrapped, on stderr, a blank line, `▸`, is how the line
renderer draws those items. The TUI draws the same items its own way.)*

| Event | `raw` | `verbose` | `default` | `terse` |
| :--- | :--- | :--- | :--- | :--- |
| reply text | the update's params, one JSON line (D8) | the reply, styled and wrapped as today | as `verbose`, with one blank line before it when an action line precedes it | as `default` |
| non-text block | JSON line | `[image]` | `[image]` | `[image]` |
| thought | JSON line | dimmed, under a `thinking` label, wrapped, a blank line after | not printed; the spinner's label says `thinking` (D5) | not printed; spinner as `default` |
| `tool_call` `pending`, `in_progress` | JSON line | nothing printed; the spinner's label names the call | nothing printed; spinner as `verbose` | nothing printed; spinner as `verbose` |
| call `completed` | JSON line | one line `▸ <title> ✓`, then its content, unbounded: the output block when there is one, else the summary, then any diffs (D4) | joins the open group (D3); no output | counted into the action line (D3) |
| call `failed` | JSON line | one line `▸ <title> ✗`, then its content as for `completed` | joins the group, which says `✗ n failed`; after the group line, the call's title, its summary (for a shell call, its exit status, said once) and the last five lines of its output (D4) | one line per failure: `✗ <title>: <summary>` |
| diff | JSON line | the unified diff, coloured, unbounded | the edit group's line names each file with `+a −d` (Q5) | counted |
| terminal reference | JSON line | `terminal <id>` | not printed | not printed |
| `plan` | JSON line | the checklist, each time | the checklist when it changed, after the open group closes | one line: `plan: <done>/<all>, <the step in progress>` |
| `usage_update`, turn end | JSON line; the status line on stderr | the status line on stderr | as `verbose` | as `verbose` |
| `_gobble/retry`, compaction | JSON line (its method and params) | as today | as today | as today |
| `_gobble/personality`, `available_commands_update`, `current_mode_update`, `session_info_update`, `config_option_update`, `user_message_chunk` | JSON line | not printed | not printed | not printed |
| permission request | the prompt, as today | as today | the open group closes first; then the prompt as today | as `default` |

The status line and `--stats` (D7) are the same in every mode. A mode changes nothing already printed: a reprint at a
resize draws what was written, in the mode it was written in.

### D3. Like actions group by the tool's ACP kind, written once when the group closes (Q1)

* acpclient's `ToolCall` gains `Kind`, the ACP `ToolKind` from `tool_call` (and from a `tool_call_update` that sets it).
  A group is a run of finished calls of one kind: `execute`, `read`, `search`, `edit` (with `delete` and `move`),
  `fetch`; any other kind, `other` among them, groups by title.
* `default` closes the open group before anything else it prints: reply text, a plan, a permission prompt, a call of
  another kind, a retry or compaction note, the turn's end. A thought, which `default` does not print, does not close
  it. The group's line is written once, then: `▸ ran 3 commands ✓ go build ./... · go vet ./... · +1`, `▸ read 2
  files ✓ notes.md · go.mod`, `▸ edited 2 files ✓ a.go +2 −1 · b.go +5`. A group of one reads as the call does:
  `▸ ran go build ./... ✓`. The line is one terminal row: the titles are cut to fit and the rest counted as `+n`.
* While a group is open, `default` restarts the turn's spinner on stderr with the running call's title, so a long
  command still shows what is running. The spinner is gobble's (`internal/cli/render/spinner.go`) and already does
  this before the first output; the view now restarts it after a write instead of stopping it for the rest of the turn.
  *(2026-11-13, Q1's ruling, stage 1:)* the label shows the open group growing, `ran go build ./... ✓ +1 +2 · go
  test ./...`: the finished calls as the group line would show them, then the running call's title. The view gives
  the label to the spinner as each call finishes or starts, and the label is cut to the terminal's width, so it never
  wraps (today's spinner does not cut it, `spinner.go:47-61`). Stage 2 draws the same text in D10's status row.
* `terse` folds every group between two replies into one action line: `▸ ran 3 commands, read 2 files, edited 1 file
  ✓`, closed by the same events.
* The words are the personality's: new keys for each kind's phrase, in all four built-in catalogs
  ([0022-MADR-gobble-voice-and-personalities.md](0022-MADR-gobble-voice-and-personalities.md)).
  *(2026-11-13, the amendment, D13: the grouping and its closing rules are the curation layer's; each renderer phrases
  the layer's items with these same keys, so the line view and the TUI use one set of words.)*

### D4. A command and its exit status: the title once, the status once, the output only when it failed

* A shell call's title stays the agent's (`run <command>`, at most 60 runes). The group line cuts each to fit; the full
  command is in `verbose`, `raw`, the permission prompt and the session file.
* A call's status is its glyph: `✓` for `completed`, `✗` for `failed`. A shell call is `failed` exactly when its exit
  status is not 0 or it was stopped at its timeout (`tool/builtin/shell.go:188-196`), so `default` and `terse` say
  nothing more of a success.
* A failure is shown with its summary, which for a shell call is `exit status N: <first line>`, and the last five
  lines of its output. The output's final line, when it is the shell tool's own status line (`[exit status N]` or
  `[stopped after …]`, `tool/builtin/shell.go:189`, `:194`), is left out of that tail, so the status is said once. When
  lines were cut above the tail, the personality's hidden-lines text says how many, naming `/view verbose`. The view
  recognises that line by the shell tool's format, which lives in this repository; a test holds the two together.
  *(2026-11-13, the amendment, D11: the recogniser and the failure tail are the curation layer's, not the view's; the
  test moves with them.)*
* `verbose` shows a call's output block, unbounded, and leaves out its summary when it has one: the agent writes the
  summary from the output (`acpserver/prompt.go:197-215`; for a shell call, its exit status and first line), so the
  output says it all. A call with no output block (an edit's diffs, a result that is its summary alone) shows its
  summary, then its diffs. Each exit status is then said once in `verbose` too, by the output's own last line.

### D5. Thinking is folded in `default` and `terse` (Q6)

The model's thoughts are not printed in `default` or `terse`; while thoughts stream, the spinner's label says
`thinking` in the personality's words. `verbose` prints them dimmed under a label, so a thought can be told from the
reply with colour off. `raw` prints their updates. The session file keeps them in every mode.

### D6. The mode is chosen by `/view`, `--view` and the `view` setting (Q4)

* **`/view [raw|verbose|default|terse]`** joins the line session's local commands (`0008-MADR-native-cli-mode.md` D13;
  `internal/cli/slash.go:22-30`). With no argument it prints the mode and the four; with one it sets the mode for the
  rest of the process, from the next turn, `/new` sessions included. It saves nothing. An unknown mode is an error
  printed locally, and the mode stays.
* **`--view <mode>`** on the default command sets the starting mode for the run. It is an `enum` flag with completion
  (`0008-MADR-native-cli-mode.md` D12, D17).
* **`view: <mode>`** in `settings.yaml` ([0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md)) is
  the default for every run; an unknown value is a configuration diagnostic and `default` is used.
* Precedence: `/view`, then `--view`, then the setting, then `default`.
* *(2026-11-13, the amendment, D15: the three controls, with this precedence, set the mode of either renderer, the
  line session's or the TUI's.)*

### D7. `--verbose` and `--show-thinking` retire into `--view`; `--stats` and `--log-level` stay (Q3)

* `--verbose` and `--show-thinking` are removed. Their behaviour is `--view verbose`, which shows both full output and
  thinking. The hidden-lines text in the four built-in catalogs names `/view verbose` in place of `--verbose`. No
  alias is kept: the owner is the only user. An old command line that names either flag exits 2 with the parser's
  unknown-flag message (`0008-MADR-native-cli-mode.md` D10's usage code); no hidden flag is kept to word it otherwise.
* `--stats` stays, independent of the mode, in the line session and in print mode.
* `--log-level` stays as it is: it sets the log file's level and is not a view setting.

### D8. `raw` shows the ACP updates themselves (Q2)

`raw` writes each update's params as one JSON line, as the SDK encodes them, which is what `-p --output-format
stream-json` writes for the same turn: the stream the phone receives, without its JSON-RPC framing. gobble's
`_gobble/*` notifications are written as their method and params. The line session starts its connection with
acpclient's `Params` on (`internal/cli/line.go:127`), which encodes each update once more on the client side
([0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md) D16);
the agent's bytes do not change. The status line still goes to stderr. Permission prompts are drawn as in every
mode.

*(2026-11-13, the amendment, D12 and D13: the layer makes one `Raw` item per update, which the line renderer writes as
that JSON line; a notification's method comes from acpclient's new `Update.Method`, since neither the layer nor
`internal/tui` may import acpext (rule 12).)*

### D9. Where it lives, and the records it amends on acceptance

* ~~The modes, the grouping and the closing rules live in `internal/cli` (the view);~~ the pure layout of a group line, a
  diff stat and the action line in `internal/cli/render`. acpclient gains `ToolCall.Kind`, an addition to a beta
  package's type. *(2026-11-13, the amendment, D11 to D13: the modes, the grouping, the closing rules, the failure
  tail and the line diff's counts live in the curation layer, `internal/curate`; the view in `internal/cli` feeds it
  and draws its items; `render` keeps the layout, `render.UnifiedDiff`'s hunks drawn over the layer's line diff, so
  there is one diff implementation. acpclient also gains `Update.Method`. Amended on execution, besides the records
  below: [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md), its package map and its import
  rules, which gain rule 13 (D11), written with the archtest change in
  `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P1b.)*
* Amended on acceptance, each with a dated note beside the text it changes: `0008-MADR-native-cli-mode.md` D7 (rendering, by D2 to D5),
  D12 (the flag set, by D6 and D7) and D13 (the local commands, by D6); `0005-MADR-v1-feature-scope.md` "CLI flags" (`--verbose` at 1.x, by
  D7). `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P0 writes these.
  ~~`0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b is unchanged under Q1 (a), and amended only if the owner answers Q1 (b).~~
  *(2026-11-13, Q1's ruling:)* amended **when `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P3b runs**, not before, each as a dated amendment written in
  that phase: `0030-MADR-resize-and-performance.md` D2 (the live area also holds the open group's row) and
  `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b (mid-turn, with the status row drawn, the reprint is written as
  `region.Print(out)`, not as `out` alone); and `0008-MADR-native-cli-mode.md` D5's spinner line (the status row replaces it). Stage 1
  amends none of them.

### D10. Stage 2: the status row (Q1's ruling; `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P3b)

* **One owner of the bottom row.** A status row on stderr, drawn through go-tui-lib's `inline.Region` (v2.12.0 or
  the pin of the day), replaces the spinner (`internal/cli/line.go:341-346`; `render/spinner.go`). It is made with
  `inline.WithoutCursorQuery()`, as the permission prompt's is (`internal/cli/permission.go:187`), since during a
  turn nobody reads the terminal's input and a cursor report would land in the next prompt; with `WithHeight(1)`, one
  row; and with the editor's reflow choice (`internal/cli/editor/editor.go:153-156`). Each row is cut to the width,
  which the Region also enforces (`inline/region.go:29-40`). It shows the frame, the open group as stage 1's label
  does, `thinking` while thoughts stream, or the personality's working line.
* **Every other writer yields to it.**
  * The view's stdout writes (`view.put`, `internal/cli/view.go:415-427`) clear the row first, and the view draws it
    again after, as `beforeWrite` stops the spinner today (`:436-442`).
  * `Output.diag` and `Output.logRecord` (`internal/cli/output.go:80-103`) write through the row's `Print` while it
    is drawn: the text above, the row again below (`inline/region.go:120-142`). Today they write past a running
    spinner (`view.go:176-179` with `output.go:80-89`).
  * The permission prompt: `askPermission`'s `v.quiet()` (`internal/cli/permission.go:252`) becomes a yield that
    closes the open group (its line written, as in stage 1) and clears the row; the prompt draws its own region as
    today (`:257-265`); the next update draws the row again.
  * The mid-turn reprint: `tryReprint` (`view.go:560-581`) writes the transcript's reprint as `region.Print(out)`
    while the row is drawn, one synchronized update, as go-tui-lib's transcript asks of a host with a region
    (`transcript/doc.go`) and as `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b's accepted note does at the prompt.
  * A resize (`lineSession.route`, `line.go:463-478`) gives the Region the new width; it redraws by the reflow kind.
* **Collapse and fallback.** When the group closes, the row is cleared and stage 1's line is written on stdout, which
  the transcript keeps; the scrollback holds that line and never the row's intermediate states. When stderr is not a
  terminal, no row is drawn, as the spinner draws nothing then (`spinner.go:39-41`), and the output is plain A.
* **Concurrency.** The Region is driven under the view's lock (`v.mu`) by: the notification goroutine (`view.on`),
  the permission callback, the resize router, the settle timer (`view.go:547-551`), and the row's frame ticker, which
  replaces the spinner's goroutine. `make race` covers them.
* **Open: Q7**, type-ahead echo, below; decided in `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P3b.

### Amendment 2026-11-13: a renderer-neutral curation layer

*Accepted 2026-11-13. The owner's ruling, verbatim, relayed by the coordinator: "Shared layer + go-tui-lib survey".
Read at `8fc4233`. D1 to D10 stand, except where a dated note beside them points here.*

**The finding behind it.** D2 to D5, as first written, build the curation (grouping like actions, saying each exit
status once, folding thinking, and the four modes) inside the line view's one `switch` on `acpclient.Kind`
(`internal/cli/view.go:116-191`, unchanged at `8fc4233`), and D9 placed the modes, the grouping and the closing rules
in `internal/cli`. The enhanced TUI mode, the full-screen mode of
[0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) F9 ("Enhanced terminal TUI") and
[0032-MADR-go-tui-lib-integration-boundary.md](0032-MADR-go-tui-lib-integration-boundary.md) D6, needs the same
decisions over the same ACP updates; `internal/tui` holds only its package doc. Built in the line view, the curation
would be built a second time for the TUI, and the two copies could drift. That goes against 0032-MADR's driver "both
modes, the line-mode CLI and the TUI, served by one implementation of each element".

**The ruling.** The curation becomes a layer of its own, which both renderers draw from, and `/view` applies to both
modes. The survey half of the ruling is under More Information, "go-tui-lib `v2.12.0` against F9".

#### D11. The layer is `internal/curate`, and it imports only the standard library, `acpclient` and `jsonx`

* **Package:** `github.com/maccavelli/gobble-cli/internal/curate`, internal by its path. It holds the modes, D14's
  per-event contract, the grouping and its closing rules (D3), the failure tail and the shell status-line recogniser
  (D4), thinking's fold (D5), `raw`'s items (D8), and the line diff with its `+a −d` counts.
* **Allowed in non-test code:** the standard library, `acpclient` and `jsonx`. Nothing else.
* **Refused, each for a reason the records already give:**
  * go-tui-lib, and every Charm module, `x/ansi` included. Rule 5, as amended by 0032-MADR D1 and A1, admits them
    only in `internal/cli` and `internal/tui`. A third package gets none, so the TUI's graph cannot reach the native
    CLI through the layer, and the layer measures no width.
  * Kong (rule 9); acpext (rule 12, hence D12's `Update.Method`); acp-go-sdk2 (rule 2); `agent` (rule 6's reason:
    the renderers read ACP, never the agent).
  * `internal/cli/...` and `internal/tui`: each renderer imports the layer, never the reverse, and `internal/cli`
    still never reaches `internal/tui` (rule 5).
  * `personality`, `render` and `term`: words, layout and terminal facts are the renderers' (D13).
* **Its tests** may import `acpserver`, `llm/llmtest`, `acpclient/acptest` and `tool/builtin`, for the fixture of real
  agent updates and the status-line test (`0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P1, P1b).
* **It does no I/O, reads no clock and starts no goroutine.** One goroutine drives it, under its renderer's lock, as
  go-tui-lib's `inline.Region` is driven (`inline/region.go:29-40` at `v2.12.0`).
* **Enforced** by `internal/archtest` rule 13: "`internal/curate` imports, in non-test code, only the standard library,
  `acpclient` and `jsonx`". A dated note in [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md)
  adds the rule to its import rules and the package to its package map. Both are written in the same phase as the test
  (`0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P1b), since `AGENTS.md` has a rule and its record
  change together.

#### D12. Input: every ACP update, two client events, and a mode

* **Updates.** Every `acpclient.Update`, in arrival order, of every kind, through `Feed`. That includes
  `_gobble/personality`, which the line session hands on after setting the voice.
* **Two acpclient fields.** `ToolCall.Kind` (D3), and `Update.Method`: the notification's method for `KindRetry`,
  `KindPersonality` and a `KindCompaction` from `_gobble/compaction`, and empty for a `session/update`. With it, `raw`
  writes `{"method":…,"params":…}` without acpext.
* **Two events no update carries.**
  * `Ask(callID)`: a permission request for that call, raised by the renderer before it draws its prompt or dialog.
  * `End(acpclient.Result)`: the turn's end.
* **The mode.** `curate.Mode` is one of `raw`, `verbose`, `default` or `terse`. `curate.ParseMode` and `curate.Modes`
  serve `/view`, `--view` and `view:`. A renderer makes a `curate.Curator` per turn with the mode in force then, so a
  mode changes from the next turn (D6).

#### D13. Output: changes to typed items, one type per TUI component

Each input returns a list of `Change` values, `{Op, ID, Item}`, where `Op` is `Open`, `Grow` or `Close`. An item is
closed once and never changes after. `Curator.Status()` returns a snapshot of the status row.

The item types match, one to one, the components the TUI will need. The survey under More Information finds each of
those components absent from go-tui-lib `v2.12.0`, apart from the transcript. Typed this way, the TUI renderer is a
thin consumer once go-tui-lib gains them, and the line renderer draws the same types its own way:

| Item | What it carries | The TUI component it feeds | What the line renderer draws |
| :--- | :--- | :--- | :--- |
| `ToolCard` | one call: its ID, ACP kind, title, status, summary, output, `Diff`s and terminal IDs, and its `Fold` | a collapsible tool card | D2's `verbose` call block, or D4's failure detail |
| `ActivityRow` | a run of finished calls: the count and failed count per kind, each call's title and diff stat, and the calls' `ToolCard`s | a grouped activity row | D3's group line, or `terse`'s action line |
| `Diff` | a path, the old and new text, and `+a −d`; inside a `ToolCard` | a diff view | a unified diff in `verbose`; the stat on the group line in `default` |
| `StatusRow` (from `Status()`) | thinking or not, the open `ActivityRow` so far, and the running call | a status row | the spinner's label (stage 1); D10's row (stage 2) |
| `Thought` | the thought text, growing | a thought display | dimmed under a label, in `verbose` |
| `Plan` | the entries, whether they changed, and a `Form`: a checklist or a summary | a plan checklist | D2's checklist, or `terse`'s plan line |
| `Reply` | the reply text, growing, or a non-text block's name | the transcript | go-tui-lib's `stream.Printer`, as today |
| `Note` | an `acpclient.Retry` or `acpclient.Compaction` | a notice in the transcript | today's retry and compaction lines |
| `TurnEnd` | the last usage and the `acpclient.Result` | the footer | the status line on stderr |
| `Raw` | an update's `Params`, or a notification's method and params | a raw line | one JSON line (D8) |

* **`Fold`** is the mode's choice for an item's body. It is one of:
  * `Hidden`;
  * `Summary`;
  * `Tail`: D4's summary, then the last five lines without the shell tool's status line, then the count of lines
    above;
  * `Full`.
* **Every item carries its full detail, whatever its fold.** Q8 asks what the TUI may do with it. *(2026-11-13,
  Q8's ruling: the mode sets each item's default fold, and in the TUI any item opens up to the full detail it
  carries.)*
* **`Sep`** asks for a gap before an item, such as D2 `default`'s blank line before a reply that follows an action
  line.
* **Neither renderer branches on the mode.** The items say what to show, fold, group and close; a renderer decides
  only how to draw them.
* **The layer carries no words, glyphs, colour or width.** Each renderer phrases the items with the personality's keys
  (D3: one set of keys for both), sanitises what did not come from gobble
  ([0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D8), and lays out.
* **State-only updates make no item outside `raw`.** These are the commands, the mode, the session's info and its
  config options. A renderer that shows that state, such as the TUI's footer or the line session's `/help`, reads it
  from acpclient as today.

#### D14. The per-event table is the layer's contract, not the line renderer's

D2's table, restated in the layer's terms. A layer-level golden per mode holds it, together with a table test with one
row per `acpclient.Kind` (`0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P1b).

| Input | `raw` | `verbose` | `default` | `terse` |
| :--- | :--- | :--- | :--- | :--- |
| reply text, non-text block | `Raw` | `Reply` | `Reply`, `Sep` after an action line | as `default` |
| thought | `Raw` | `Thought`, `Full` | `Thought`, `Hidden`; `StatusRow` thinking | as `default` |
| `tool_call` `pending`, `in_progress` | `Raw` | no item; `StatusRow` names the call | as `verbose` | as `verbose` |
| call `completed` | `Raw` | an `ActivityRow` of one call, its `ToolCard` `Full` (the output block, or else the summary, then diffs) | joins the open `ActivityRow` of its ACP kind; its `ToolCard` `Hidden`; its diff stats on the row | joins the run's one `ActivityRow`, whatever its kind; `Hidden` |
| call `failed` | `Raw` | as `completed` | joins the row, which counts it failed; when the row closes, its `ToolCard` follows, `Tail` | joins the row; its `ToolCard` follows, `Summary` |
| terminal reference | `Raw` | in the `ToolCard`, shown | in the `ToolCard`, `Hidden` | as `default` |
| `plan` | `Raw` | `Plan`, checklist, each time | `Plan`, checklist, when it changed, after the open row closes | `Plan`, summary |
| `usage_update`; `End` | `Raw`; `TurnEnd` | `TurnEnd` at `End` | as `verbose` | as `verbose` |
| `_gobble/retry`, compaction | `Raw` | `Note` | `Note` | `Note` |
| `_gobble/personality`, commands, mode, session info, config option, user message, a newer variant | `Raw` | no item | no item | no item |
| `Ask` | no item | no item | closes the open row | as `default` |

* **Closing.** In `default`, the open `ActivityRow` closes on a `Reply`, a `Plan`, an `Ask`, a finished call of
  another kind, a `Note` or `End`, and never on a thought. In `terse`, the same, except that a call of another kind
  joins the row (D3).
* **Permission prompts.** The prompt or dialog itself is never an item. It is each renderer's own:
  go-tui-lib's `ask` in the line session, a dialog in the TUI.

#### D15. `/view`, `--view` and `view:` apply to both modes

* The three controls, with D6's precedence, set the mode of whichever renderer runs: the line session or the TUI.
* `/view` is a local command in both. `--view` is on the default command, and holds when the TUI is the mode chosen.
  `view:` in `settings.yaml` is one default for both.
* In both, a mode applies from the next turn. Q9 asks whether the TUI may re-curate earlier turns. *(2026-11-13,
  Q9's ruling: it may not; `/view` applies from the next turn in both modes, with no replay.)*
* D7's retirement of `--verbose` and `--show-thinking` holds for both.

#### D16. What the TUI inherits, and what stays the line renderer's

* **The TUI inherits, through the layer:**
  * D1: no mode changes the ACP stream;
  * D2's table, as D14;
  * D3's grouping by ACP kind and its closing rules;
  * D4's exit status said once, and its failure tail;
  * D5's fold of thinking;
  * D6's controls, as D15;
  * D8's `Raw` items;
  * the `StatusRow` snapshot, and the personality's keys.
* **These stay the line renderer's:**
  * stage 1's spinner, and its label cut to the terminal's width (D3);
  * stage 2's status row drawn through `inline.Region`, every writer's yield to it, and its collapse and fallback
    (D10);
  * Q7, type-ahead echo during a turn in cooked mode;
  * the transcript's reprint at a resize
    ([0030-MADR-resize-and-performance.md](0030-MADR-resize-and-performance.md) D3;
    `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b);
  * stdout and stderr ([0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D3);
  * the group line's one-row layout with `+n`, the blank lines, glyphs, colour, and wrapping
    ([0028-MADR-reply-word-wrap.md](0028-MADR-reply-word-wrap.md)).
* **Why the status row and Q7 stay behind.** A full-screen TUI owns its input and redraws whole frames, so neither
  D10's yields nor Q7's echo arise there. The TUI draws `StatusRow` as one of its own components.
* **The TUI's drawing is 0005-PLAN F9's.** `0005-PLAN-v1-feature-scope.md` F9, when it is made executable, takes the
  layer as its input and carries Confirmation 15. This record builds no TUI code. 0005-MADR F9 and 0032-MADR are
  unchanged: the TUI is still an ACP client over acpclient, with the layer in between, and the layer draws nothing, so
  it is not core TUI.

### Questions for the owner, and the rulings

Each has three options, as the draft put them, the recommended first. The owner ruled on Q1 to Q6 on 2026-11-13;
each ruling follows its options, and the chosen option is the decision. Q7 was added that day from the feasibility
study and is open.

**Q1. How does a group of like actions appear?**

* **(a) Written once, when the group closes; the spinner names the running call meanwhile (D3).** Cost: the group's
  line appears only when the next kind of event arrives, so a run of ten quick reads shows as one spinner, then one
  line; no new terminal mechanics, and the mid-turn reprint is untouched.
* (b) A live row, redrawn in place as the group grows (`▸ ran go build ./... ✓ +1 +2 +3`), through go-tui-lib's
  `inline.Region`, committed when the group closes. Cost: a region drawn while a turn runs, which `0030-MADR-resize-and-performance.md` D3 and
  `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b chose not to do; the permission prompt's own region and the resize reprint must hand over to
  it and back; amending both records and a larger PLAN phase.
* (c) No groups: one line per call, written at its terminal status, nothing else. Cost: still one line per action, the
  owner's main complaint; the smallest change.

*Ruling, 2026-11-13: "Staged hybrid". **Decision: (b), as the staged hybrid**: stage 1 is (a) with the spinner's
label showing the group growing, cut to the width (D3); stage 2 is the live status row of D10, which collapses to
(a)'s line when the group closes, and is plain (a) when stderr is not a terminal. The amendments of
`0030-MADR-resize-and-performance.md` D2 and `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14
step 5b are written as dated amendments when `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P3b runs (D9).*

**Q2. What does `raw` show?**

* **(a) The ACP updates as JSON lines, the phone's view (D8).** Cost: one more encode per update on the client side
  for every mode, since `Params` is set when the connection starts; unreadable as a chat, by design.
* (b) Every update decoded and printed in full, unstyled, ungrouped and unbounded. Cost: barely different from
  `verbose`, so two modes for one need.
* (c) No `raw`; three modes. Cost: departs from the owner's list; `-p --output-format stream-json` remains for the
  stream.

*Ruling, 2026-11-13: "The ACP updates as JSON lines". **Decision: (a)** (D8).*

**Q3. What happens to `--verbose` and `--show-thinking`?**

* **(a) Removed; `--view verbose` replaces both; `--stats` kept (D7).** Cost: an old command line fails with exit 2
  until it is changed; the four catalogs' hidden-lines text changes.
* (b) Kept as hidden aliases of `--view verbose`. Cost: two names for one thing, against "one way to choose"; a
  compatibility shim the owner has said not to build unasked.
* (c) Kept as modifiers on any mode (thinking or full output on top of `default`). Cost: sixteen combinations to
  test and explain; the modes stop being the one control.

*Ruling, 2026-11-13: "Remove them". **Decision: (a)**: `--verbose` and `--show-thinking` go; an old command line
exits 2 with the parser's unknown-flag message (D7).*

**Q4. Does a mode persist?**

* **(a) `/view` lasts for the process; `view` in `settings.yaml` is the lasting default; `--view` is per run (D6).**
  Cost: a mode wanted for good is set in the file, not by the command.
* (b) `/view` also writes the setting. Cost: a command that changes a file as a side effect; a project's
  `.gobble/settings.yaml` or the global one has to be chosen.
* (c) Per session, saved in the session file and restored on resume. Cost: a terminal preference stored by the agent, in
  the file every client resumes from; an ACP or `_meta` change to carry it, against D1.

*Ruling, 2026-11-13: "/view for this run; setting is the default". **Decision: (a)** (D6).*

**Q5. How does `default` show an edit?**

* **(a) The group line names each file with its `+a −d`; the diff body is in `verbose` (D2).** Cost: reviewing an edit
  takes `/view verbose` or the session file.
* (b) The diff body, bounded at 20 lines (10 and 10) as today, below the group line. Cost: an edit-heavy turn stays
  long.
* (c) The full diff. Cost: the longest turns of all.

*Ruling, 2026-11-13: "Diffstat only". **Decision: (a)** (D2).*

**Q6. How does `default` show thinking?**

* **(a) Not printed; the spinner's label says `thinking` (D5).** Cost: a thought is visible only in `verbose` or `raw`.
* (b) One folded line per thought burst, `· thought for 4 s`. Cost: a line per burst, the noise the owner named.
* (c) Printed dimmed under a label, as `verbose` does. Cost: `default` and `verbose` differ less.

*Ruling, 2026-11-13: "Hidden; spinner says thinking". **Decision: (a)**: not printed in `default` or `terse`, the
spinner's label (stage 2: the status row) says `thinking`; labelled and dimmed in `verbose`; as JSON lines in `raw`
(D5).*

**Q7. How does the live row meet type-ahead echo? (Open; owned by `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P3b.)**

During a turn the terminal is out of raw mode with echo on (`internal/cli/line.go:206-223`; `0024-MADR-cli-mode-polish.md` D1), so on Unix
a key typed during a turn is echoed at the cursor. The Region redraws a changed row by relative column moves from
where it believes the cursor stands (`inline/region.go:35-40`, `:412-430`), so an echoed key can misplace the next
drawing; a typed Enter moves the cursor down a row and leaves a stale row above. Today's spinner rewrites from `\r`
every frame, so it heals within a frame. Windows echoes only during a pending console read, so the risk is expected
to be Unix's alone; that is unverified.

* (a) Accept it as a known artifact, mitigated: one row, cut to the width, and each frame drawn from the row's start
  (a cleared and redrawn row, not a relative rewrite), so the row heals as the spinner does. Cost: a stale row is
  still left behind after a typed Enter; no terminal-mode change.
* (b) Echo off during a turn (`ECHO` cleared, `ICANON`, `ISIG` and output processing kept), restored before the
  prompt. Cost: amends `0024-MADR-cli-mode-polish.md` D1; type-ahead is no longer shown while it is typed; a mode change on every turn,
  on three systems.
* (c) Raw mode during the turn, with the keys read and kept as type-ahead for the next prompt. Cost: the largest
  change: amends `0024-MADR-cli-mode-polish.md` D1, takes over Ctrl+C handling during a turn, and changes the line editor's input path.

*Ruling, 2026-11-13: "Decide in stage 2". **Open.** `0056-PLAN-curate-the-terminal-session-view-with-view-modes.md` P3b decides it with screen tests in hand
(`inlinetest.Screen` with `Cooked` set, an echoed key written between frames), and records the owner's answer here as
a dated note. `0024-MADR-cli-mode-polish.md` D1 stands until then.*

*(2026-11-13, the amendment: Q8 to Q10 are added from it and from the go-tui-lib survey, each open. None blocks
`0056-PLAN-curate-the-terminal-session-view-with-view-modes.md`; each is decided before
`0005-PLAN-v1-feature-scope.md` F9 is made executable, and recorded here as a dated note.)* *(2026-11-13, later:
the owner ruled on all three that day, relayed by the coordinator; each ruling follows its options, which stand as
written.)*

**Q8. May the TUI open a folded item beyond what the mode shows? ~~(Open.)~~ (Ruled 2026-11-13.)**

* **(a) Yes.** Each item starts at the mode's fold, and a tool card or an activity row can be opened to the full detail
  every item carries (D13). Cost: the TUI must draw and remember a fold state per item. A turn's items hold whole
  outputs in memory, which they hold anyway for `verbose`. This meets 0005-MADR F9's "collapsible tool cards".
* (b) No. The mode is a ceiling in the TUI as in the line view, and more detail takes `/view verbose`. Cost: F9's
  collapsible cards open to nothing more than the mode shows; the TUI loses what it is best placed to give.
* (c) Only failures open beyond the mode. Cost: a rule a person has to learn; a successful call's output stays out of
  reach.

*Ruling, 2026-11-13: "Yes, up to full detail". **Decision: (a)**: the mode sets each item's default fold (D13); in
the TUI any item can be opened up to the full detail it carries, so the TUI keeps a fold state per item. The line
view draws each item at its default fold.*

**Q9. Does `/view` in the TUI re-curate the turns already shown? ~~(Open.)~~ (Ruled 2026-11-13.)**

* **(a) No: from the next turn, as in the line view (D6, D15).** Cost: the TUI does less than a full-screen redraw
  allows. One rule holds for both modes, and nothing is kept beyond each turn's items.
* (b) Yes. The TUI keeps each turn's updates and replays them through a new `Curator` at `/view`, since the layer is
  pure. Cost: the session's updates are kept in memory; the two modes then follow different rules for `/view`; and a
  test must prove that a replay equals a live run.
* (c) Only the running turn is re-curated. Cost: a partial rule, with the replay machinery of (b) all the same.

*Ruling, 2026-11-13: "No, from the next turn". **Decision: (a)**: `/view` re-curates from the next turn in both
modes (D6, D15); no turn is replayed, and no updates are kept for it.*

**Q10. What happens to the components the TUI needs that go-tui-lib `v2.12.0` lacks? ~~(Open.)~~ (Ruled 2026-11-13.)**

The six components are the tool card, the activity row, the diff view, the status row, the thought display and the plan
checklist (More Information). This record decides nothing about go-tui-lib itself; the question is the owner's.

* **(a) Request them from go-tui-lib, as `0032-MADR-go-tui-lib-integration-boundary.md` D3 and D5 do for its
  elements, each shaped by D13's item types, while F9 waits for their tags (0005-MADR F9).** Cost: F9's activity
  display waits on another repository's releases. The line view is unaffected, since it draws them itself.
* (b) F9 starts on what exists (the workspace, `streamtea.DocPane`, `asktea.Dialog`, `palette`, `lineedit`) and shows
  tool activity as plain transcript text until the components are tagged. Cost: an interim TUI that is plainer than
  the line view. Whether that interim drawing counts as "copying" under 0005-MADR F9 needs the owner's reading.
* (c) Defer F9 as a whole until the components exist. Cost: no TUI at all in the meantime; the layer waits for its
  second consumer.

*Ruling, 2026-11-13, given earlier that day: "Draft a go-tui-lib record now". **Decision: (a)**: the tool card, the
activity row, the diff view, the status row, the thought display and the plan checklist are requested from
go-tui-lib, each shaped like D13's item types, and F9 waits for their tags (0005-MADR F9). The request is a go-tui-lib agent-widgets record, to be drafted,
in go-tui-lib; it has no number yet. This record still decides nothing about go-tui-lib's design.*

### Consequences

* Good, because the phone's stream does not change: D1's test holds the bytes equal across modes.
* Good, because the sample turn's 62 lines of actions become about ten in `default` (five group lines, the failure's
  title and status, the plan's three steps) and three in `terse` (the action line, the failure, the plan).
* Good, because each exit status is said once, and a failure is shown with its cause where a success says nothing.
* Good, because one control, `/view`, with a flag and a setting behind it, replaces two flags and gains two modes.
* Neutral, because `raw` is a debugging view, not a chat; it is the phone's stream for whoever needs to see it.
* Bad, because the view must recognise the shell tool's own status line by its text (D4), a coupling inside this
  repository that a test holds.
* ~~Bad, because under Q1 (a) a group's line appears only when the group closes; the spinner names the running call in
  the meantime.~~ *(2026-11-13, Q1's ruling:)* Neutral, because a group's line is written when it closes, while the
  spinner's label (stage 1) and the status row (stage 2) show it growing.
* Bad, because `--verbose` and `--show-thinking` stop working (Q3's ruling).
* Bad, because acpclient grows a field, and every view mode needs golden output of its own to stay honest.
* Good, because stage 2 makes one owner of the terminal's bottom row, where today the spinner and the stderr
  diagnostics can write over each other (`internal/cli/output.go:80-103`, `render/spinner.go:47-61`).
* Bad, because stage 2 amends `0030-MADR-resize-and-performance.md` D2 and `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b, adds a goroutine that
  takes the view's lock, and routes `Output`'s stderr writes through the row.
* Bad, because the row is not proven on Windows Console Host or under a terminal multiplexer by any test; both are
  checked by the owner's eye in P3b, and Q7 is open.

*(2026-11-13, the amendment:)*

* Good, because the curation is built once. The TUI inherits D1 to D8 through the layer and rebuilds none of it, and
  the two modes cannot drift apart.
* Good, because the layer is tested alone, with no terminal: a golden per mode over its change stream, and a table
  test over D14.
* Good, because the item types match the TUI's components one to one, so the TUI renderer is a thin consumer once
  go-tui-lib gains them.
* Good, because the line view becomes a thin consumer too. Its `switch` on `acpclient.Kind` goes; what it draws is a
  function of the items.
* Neutral, because every item carries its full detail whatever its fold, which costs memory for one turn's items.
  `verbose` needs that detail anyway.
* Bad, because the layer adds a package, archtest rule 13, an acpclient field (`Update.Method`), and an indirection in
  the line view: updates, then changes, then drawing.
* Bad, because the item types are fixed before the TUI exists. F9 may find them wrong and amend this record.
* Bad, because `render`'s line diff moves into the layer, a move inside this repository that the `udiff-*` goldens
  hold.
* ~~Bad, because Q8 to Q10 stay open until F9 is made executable.~~ *(2026-11-13, the rulings on Q8 to Q10:)* Bad,
  because the TUI keeps a fold state per item (Q8), and F9's activity display waits on go-tui-lib's tags of the
  six components (Q10), requested by a go-tui-lib agent-widgets record, to be drafted.

### Confirmation

1. Under each of the four modes, the same scripted turn through the line session writes byte-equal frames in both
   directions; `acpserver/testdata/session.golden` is unchanged; no file under `acpserver/`, `agent/`, `tool/`,
   `acpext/` or `llm/` is changed by the PLAN's commits.
2. A golden per mode, in colour and without, renders one fixture of real agent updates (captured from acpserver by a
   test that also proves the fixture current), and each golden was seen to fail on a planted break.
3. In `default`, a failed shell call's exit status appears once; a successful one prints no output.
4. `/view`, `--view` and `view:` set the mode with the precedence of D6; an unknown mode is refused each way.
5. `--verbose` and `--show-thinking` are refused with exit 2 (Q3's ruling); no built-in catalog names `--verbose`.
6. The resize reprint draws a curated turn as it was written.
7. The gates of the PLAN pass on macOS, Linux and Windows.
8. Stage 1: the spinner's label shows the open group growing and is never wider than the terminal.
9. Stage 2, on `inlinetest.Screen` with `Cooked` set: the status row is one row at every width; each writer of D10
   yields to it; the bytes split at each `?2026l` show no frame with the row drawn over other text; after a group
   closes, the screen and the scrollback hold stage 1's line and none of the row's states; with stderr not a
   terminal, the output equals stage 1's. Each test was seen failing on a planted break in a scratch copy.
10. Stage 2: `make race` passes over the goroutines that share `v.mu`, run through `run-heavy` or `run-on-host`.
11. Stage 2: the owner has looked at a grouped turn in Windows Terminal, in Console Host and under a terminal
    multiplexer, and Q7 is answered and recorded.

*(2026-11-13, the amendment, D11 to D16:)*

12. The layer alone: for each mode, the fixture of real agent updates through `internal/curate` gives a change stream
    equal to its golden, written in a plain text form. A table test has one row per `acpclient.Kind` and mode, D14's,
    and fails when a kind is added without a row. Each was seen to fail on a planted break. Confirmation 3's exit
    status is checked here, at the layer.
13. The line renderer is a thin consumer. The layer's recorded change stream, drawn by the line renderer alone, gives
    the same stdout and stderr bytes as the fixture's updates through the line session's view. `internal/cli/view.go`
    holds no `switch` on `acpclient.Kind`.
14. `internal/archtest` rule 13 passes. It was seen failing in a scratch copy on a planted import, in `internal/curate`,
    of `internal/cli/render`, of `x/ansi`, and of `personality`.
15. *(Carried by `0005-PLAN-v1-feature-scope.md` F9, not by this record's PLAN.)* The TUI renderer passes 13's replay
    test against the same goldens, and one turn fed to both renderers in each mode gives them equal change streams.

## Pros and Cons of the Options

### A. Curate in the client, groups written when they close

* Good, because no byte of the ACP stream changes, so the phone and every other client are unaffected.
* Good, because it uses the view's existing write path and spinner, and leaves `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b alone.
* Good, because it works the same on and off a terminal: a group line is ordinary output.
* Neutral, because acpclient gains the tool's kind, which the agent already sends.
* Bad, because a group's line waits for the group to close.
* Bad, because it couples the failure tail to the shell tool's status-line format.

### B. A live group row through go-tui-lib's `inline.Region`

* Good, because the row grows as the owner pictured it, `+2 +3 +4`, while the commands run.
* Good, because go-tui-lib already draws such rows in place (`inline` package), so gobble writes no cursor code.
* Bad, because a region drawn mid-turn reverses `0030-MADR-resize-and-performance.md` D3 and `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b, which write the mid-turn
  reprint without one.
* Bad, because the permission prompt draws its own region (`internal/cli/permission.go:187`), so the two must hand
  over; and off a terminal it degrades to A anyway.

*(2026-11-13, from the feasibility study:)* `0030-MADR-resize-and-performance.md` D3 does not forbid a mid-turn region; its reprint ends with "the live
area", which its D2 defines without a group row. The amendments are to its D2 and to `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b's
mid-turn `out` alone (D9). Staged as the owner ruled, B's first stage is A.

### C. Curate on the server

* Good, because every client would receive less to format.
* Bad, because it changes the ACP stream the phone formats itself, against the owner's words and `0008-MADR-native-cli-mode.md` D19.
* Bad, because the phone's full output would be gone, with no client able to ask for it back.

### D. Fix the duplicates only

* Good, because it is the smallest change: one header per call, the summary or the output.
* Bad, because every action keeps its own line and its output, the owner's main complaint.
* Bad, because it offers no `/view`.

## More Information

### Feasibility of the live row (read-only study, 2026-11-13)

Verdict at the time: **feasible with conditions**, the conditions being D10's.

* **The primitive.** go-tui-lib `v2.12.0`, pinned at `go.mod:19`, has `inline.Region`: rows below the scrollback,
  redrawn in place by relative moves, with `Draw`, `Print`, `Commit`, `Clear`, `Resize`, `HandleInput`,
  `HandleTimeout` and `Restore` (`inline/region.go:71-276`), and the options `WithSynchronized`, `WithReflow`,
  `WithoutCursorQuery`, `WithHeight` and `WithReportWait` (`inline/options.go:26-58`). It does no I/O, reads no clock
  and starts no goroutine; one goroutine drives it (`region.go:29-40`). A row wider than the width is cut to it. Each
  drawing is one synchronized update (mode 2026) with the cursor hidden. `Commit` leaves rows whole in the scrollback
  (`:144-161`); `Print` writes text above the region and draws it again below (`:120-142`). go-tui-lib has no spinner
  component; `frame` paces drawing.
* **gobble's two uses today.** The line editor, `inline.NewRegion(inline.WithReflow(reflow))`
  (`internal/cli/editor/editor.go:162`, the reflow kind chosen at `:153-156`), and the permission prompt,
  `ask.WithRegion(inline.NewRegion(inline.WithoutCursorQuery()))` (`internal/cli/permission.go:187`). The editor's
  draws only at the prompt; the permission prompt's draws mid-turn, in raw mode for its own read (`:257`), and leaves
  only its rows once answered. No region stays drawn while a turn runs.
* **Off a terminal, colour off, resize.** The Region does not check for a terminal; the host decides, as the spinner
  does (`render/spinner.go:39-41`). Rows are the host's styled or plain text; `Commit` resets a styled row. A resize
  sends `CSI 6 n` unless `WithoutCursorQuery`, which redraws at once by the reflow kind (`region.go:206-224`). During
  a turn the terminal is cooked and nobody reads its input, so `WithoutCursorQuery` is required.
* **The reprint.** go-tui-lib's `transcript/doc.go`: "A host with an inline.Region writes the reprint with the region's
  Print". `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b's accepted note writes it "as `region.Print(out)` at the prompt, and as
  `out` alone mid-turn" (`:1019-1021`), which `tryReprint` does (`internal/cli/view.go:572-576`); a mid-turn row
  changes the second half. `0030-MADR-resize-and-performance.md` D3 already ends its reprint with "the live area"; its D2 defines that area as "the prompt being
  typed, and the rows of a reply still arriving".
* **Tests.** go-tui-lib's `inlinetest.Screen` (`inline/inlinetest/screen.go:21-62`) models the cursor moves, `\r`,
  line feed, erases, scrollback, mode 2026 and DECTCEM; with `Cooked` set, a line feed also returns to the row's
  start, as output processing does outside raw mode, which is the terminal's state during a turn. gobble uses it in
  `internal/cli/permission_test.go:116`, `internal/cli/editor/editor_screen_test.go` and
  `internal/cli/editor/resize_test.go`; gobble's own `internal/cli/term/termtest.Screen` is a second model. `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md`
  D14 step 5b's confirmation already splits a drawing at each `?2026l` and checks every frame. No pseudo-terminal
  harness was found.
* **Windows.** Virtual terminal processing is enabled on stdout and stderr (`internal/cli/term/console_windows.go:34`);
  the editor's and the permission prompt's regions already draw in Windows Terminal and Console Host, the editor with a
  live Console Host suite (`make probe-conhost`, `Makefile:281-282`) and the rest by the owner's eye
  (`0027-MADR-native-line-editor-word-wrap.md` D6). Resize there is polled every 100 ms (`0030-MADR-resize-and-performance.md` D1). **Gap:** no
  test draws a region in Console Host while a turn runs; it is checked by the owner's eye in P3b.
* **Terminal multiplexers.** No record or test covers them; the only mention is go-tui-lib's tmux passthrough API
  (`0032-MADR-go-tui-lib-integration-boundary.md:154-155`). `\r`, `CSI K` and relative moves are universal; mode 2026
  under tmux and screen is unverified, and is ignored where absent (`inline/options.go:28-31`). **Gap:** checked by
  the owner's eye in P3b.
* **Found in passing.** `Output.diag` and `Output.logRecord` write to stderr without stopping the spinner
  (`internal/cli/output.go:80-103`), and a retry note during a turn does not stop it (`view.go:176-179`), so the two
  can share a row today. Read in the code; not seen on screen. D10's single owner removes it.
* **Boundary.** `0032-MADR-go-tui-lib-integration-boundary.md` D3 has complex elements built in go-tui-lib, and
  gobble "builds no copy". D10 drives the library's primitive directly, as the line editor does, and draws no cursor
  code of its own.

### go-tui-lib `v2.12.0` against F9 (survey, 2026-11-13)

A read-only survey of go-tui-lib at its tag `v2.12.0`, its latest, the half of the owner's ruling of 2026-11-13 that
the amendment does not build, relayed by the coordinator. Paths are go-tui-lib's at that tag; every package named
carries "Stability: stable".

* **What F9 can build on today.**
  * The full-screen shell: `workspace.Workspace` (`workspace/workspace.go:145`) and `launch.Run`
    (`launch/run.go:49`).
  * A transcript: `streamtea.DocPane` (`stream/streamtea/pane.go:26`) over `stream.Doc` (`stream/document.go:47`).
    `transcript.Transcript` is line-mode only.
  * A permission dialog: `asktea.Dialog` (`ask/asktea/dialog.go:30`).
  * Pickers: `palette.Picker` (`palette/picker.go:33`).
  * The editor: `lineedittea.ComposerPane` over `lineedit.Composer` (`lineedit/composer.go:44`).
  * Completion: `lineedit.Completer`, `palette.Files` and `command.Registry`.
  * Streamed Markdown: `stream.Printer` (`stream/printer.go:24`) and `stream.Doc`. There is no code highlighter, and
    `stream/glamourmd` is planned.
  * Theming and colour off: `theme.Theme` (`theme/theme.go:253`), `glyph.ASCII()`, and `NO_COLOR`
    (`launch/colour.go:14-22`).
  * Tests: `tuitest.Golden` (`tuitest/tuitest.go:105`), `inlinetest.Screen` and `launchtest.Terminal`.
* **What is absent.**
  * Collapsible tool cards. Only pieces exist: `stream.Bounded` (`stream/bounded.go:35`) and `stream.Interpret`
    (`stream/interpret.go:63`). "ACP-shaped agent widgets" is a candidate in go-tui-lib's own reports, with no record.
  * Grouped activity rows. Only glyphs exist: `glyph.Set`'s `Done`, `Failed`, `Running` and `Pending`
    (`glyph/glyph.go:61-66`).
  * A diff view. A `diffview` is only named, in a go-tui-lib report.
  * A spinner or status-bar component. There are `frame`'s ticks, `layout.WithFooter` (`layout/presets.go:87`), and
    the line-mode `inline.Region`; a spinner is listed as planned.
  * A plan checklist component.
  * A full-screen thought display.
  * A virtualised full-screen transcript.
* **What 0005-MADR says to do about it.** [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) F9: "Where a
  core behaviour is not in the selected tag yet, F9 waits rather than copying it."
* **The consequence for the layer.** D13's output types map one to one onto those six absent components, the tool
  card, the activity row, the diff, the status row, the thought and the plan, and onto the transcript, which exists.
  When go-tui-lib gains a component, the TUI renderer adapts the matching item to it and adds nothing else. Neither the
  layer nor this record decides anything about go-tui-lib: Q10 asks the owner. *(2026-11-13, Q10's ruling: the six
  components are requested from go-tui-lib, shaped like D13's types, by a go-tui-lib agent-widgets record, to be drafted; F9 waits for them.)*

### Not in scope

* The ACP stream, the agent's titles and summaries, and `acpserver`. The shell summary's trailing `:` when a failed
  command printed nothing (`exit status 1:`, `tool/builtin/shell.go:198`) is the agent's text; changing it changes the
  stream, and is a record of its own if wanted.
* Print mode's formats ([0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D9); `--stats` keeps its
  meaning there.
* The enhanced TUI mode (`internal/tui`, a package doc alone at `f078889`). *(2026-11-13, the amendment: its drawing
  is still out of scope, and is 0005-PLAN F9's; its curation is now in scope, as the layer it shares with the line
  view, D11 to D16.)*
* The permission prompt's layout ([0029-MADR-permission-prompt-layout.md](0029-MADR-permission-prompt-layout.md)); it
  keeps the full title.
* The sign-in surfaces. `/login`, `/logout`, `gobble setup`, the first-run offer and the wizard's steps are `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`
  D3's terminal surfaces and `0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`'s; they draw outside the view, and `/view` does not change them. `/view` only
  joins the same local registry as `/login` and `/logout`.
* Re-rendering earlier turns in a new mode, at `/view` or at a reprint.
* Showing the session's title, mode changes or queued messages in the chat; drawn by no mode today, they stay so.
* Syntax highlighting (`0008-MADR-native-cli-mode.md` D7).
* magic-cli-remote.

### How this meets the sign-in redesign released as `v0.7.0`

* `0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` P4 step 4b changes `retryLine`'s comparisons (`internal/cli/view.go:247`, `:249`) to acpext's constants
  and respells the retry causes (`0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D6), and `acpclient/update.go`'s doc of `Retry`. This record's PLAN edits
  the same two files, so it starts from `main` after `v0.7.0` and re-reads every line cited here.
* `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D4's ACP sign-in by elicitation is answered by ACP clients that offer elicitation; the CLI's acpclient
  offers none at `f078889` (`acpclient/conn.go:160`), and nothing here changes its capabilities.
* The `retry` and limit lines (`0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md` Q3 (a); `internal/cli/line.go:399-401`) are shown in every mode, in the
  vocabulary of `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` D6 once it lands.

### Related records

* [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md): the CLI as an ACP client.
* [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md): D5 the turn, D7 rendering, D12 flags, D13 slash
  commands, D19 the phone-legible stream.
* [0022-MADR-gobble-voice-and-personalities.md](0022-MADR-gobble-voice-and-personalities.md): the view's words.
* *(2026-11-13, the amendment:)* [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): the
  package map and import rules that rule 13 joins; [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md)
  F9 and [0032-MADR-go-tui-lib-integration-boundary.md](0032-MADR-go-tui-lib-integration-boundary.md) D1, A1, D3, D5
  and D6: the TUI mode, the Charm boundary, and the components asked of go-tui-lib.
* [0028-MADR-reply-word-wrap.md](0028-MADR-reply-word-wrap.md), [0030-MADR-resize-and-performance.md](0030-MADR-resize-and-performance.md),
  [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md)
  D14: wrapping, the transcript and the reprint.
* [0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md)
  D16: `Update.Params`.
* [0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md):
  the limit line.
* [0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md),
  [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md),
  [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md):
  the release this follows.
