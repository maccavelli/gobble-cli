---
status: proposed
date: 2026-11-13
associated-madr: "0056-MADR-curate-the-terminal-session-view-with-view-modes.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement curating the terminal session view with `/view raw`, `verbose`, `default` and `terse`

Associated MADR: [0056-MADR-curate-the-terminal-session-view-with-view-modes.md](0056-MADR-curate-the-terminal-session-view-with-view-modes.md)

## Goal

The line session draws a turn in one of four modes, `default` unless chosen otherwise: like actions grouped on one
line, each exit status said once, thinking folded, with `/view`, `--view` and the `view` setting to choose (MADR D2 to
D8); `--verbose` and `--show-thinking` retired (D7); and the ACP stream byte-equal across every mode (D1). Groups grow
live in two stages (the owner's ruling on Q1, "Staged hybrid"): in the spinner's label (P3), then in a status row drawn
through go-tui-lib's `inline.Region`, to which every other writer yields (P3b; MADR D10). The plan is complete when the
MADR's Confirmation 1 to 11 is met.

This record is `proposed`. The MADR is `accepted` (2026-11-13): the owner ruled on Q1 to Q6 that day, each as the
steps below are written (Q1 the staged hybrid; Q2 to Q6 the draft's (a)). **Q7, type-ahead echo, is open**, and P3b
decides it (step 0). Nothing below starts before the owner approves this PLAN's execution, whole or by phase, and not
before `v0.7.0` (Prerequisite).

*(2026-11-13, amended before execution by the owner's ruling "Shared layer + go-tui-lib survey"; MADR "Amendment
2026-11-13: a renderer-neutral curation layer", D11 to D16.)* The curation is built first as a layer of its own,
`internal/curate`, and tested alone, with a golden per mode and planted breaks, in a new phase, **P1b**. The line view
then becomes a thin consumer of it: P2 takes the mode type from it, and P3 is re-cut to draw its items. **The phase
order is now P0, P1, P1b, P2, P3, P3b, P4.** The plan is complete when the MADR's Confirmation 1 to 14 is met;
Confirmation 15 is `0005-PLAN-v1-feature-scope.md` F9's. This PLAN stays `proposed`: code after `v0.7.0` still needs the
owner's approval. Existing phase text is annotated, not rewritten.

## Prerequisite

**This plan lands after gobble `v0.7.0`.** `v0.7.0` releases the sign-in redesign of
[0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
and [0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
([0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md) D2).
`0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`'s P4 step 4b edits `internal/cli/view.go` (`retryLine`) and `acpclient/update.go`, two of this plan's
files; its sign-in surfaces (`/login`, `/logout`, the wizard) share the local command registry `/view` joins. P0
checks the tag before anything else.

## Scope

**Base.** `main` at the commit tagged `v0.7.0`, or later, level with `origin/main`, with a clean tree. The audit's
lines were read at `f078889`; P0 re-reads each one this plan names at the base and records where it moved.

**In scope:**

* `acpclient/update.go`, its tests: `ToolCall.Kind` (MADR D3).
* `internal/cli/view.go` and a new `internal/cli/viewmode.go`: the mode, the per-event rules of D2, groups (D3), the
  failure tail (D4), thinking (D5), `raw` (D8).
* `internal/cli/render/`: new pure layout for a group line, a diff stat, the `terse` action line, the `plan` line;
  `tool.go`'s `ToolOutput` hint.
* `internal/cli/line.go` (`displayFor`, the connection's `Params`, the spinner's restart), `slash.go` (`/view`),
  `flags.go` (`--view`; `--verbose` and `--show-thinking` removed), `print.go` (`--stats` alone),
  `completion.go` and `internal/cli/complete/` where they list flags or local commands.
* `internal/config/settings.go` and its known-settings list and tests: the `view` key (D6).
* `personality/keys.go`, `personality/builtin/*/messages.md`, `personality/boring_test.go`,
  `personality/builtins_test.go`: the group phrases and the hidden-lines text (D3, D7).
* Tests and goldens under `internal/cli/testdata/` and `internal/cli/render/testdata/`.
* `docs/architecture.md`, `README.md`, `docs/README.md`; dated notes in `0008-MADR-native-cli-mode.md` D7, D12, D13 and `0005-MADR-v1-feature-scope.md` "CLI
  flags" (MADR D9).
* P3b only: a new `internal/cli/statusrow.go`; `internal/cli/output.go` (`diag`, `logRecord`);
  `internal/cli/permission.go` (`askPermission`'s yield); `internal/cli/render/spinner.go` (retired, or kept for its
  frames alone); their tests; and dated amendments of `0030-MADR-resize-and-performance.md` D2, `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b and
  `0008-MADR-native-cli-mode.md` D5's spinner line (MADR D9), written in P3b and in no earlier phase. If Q7 is answered (b) or (c),
  `internal/cli/line.go`'s turn-time terminal mode and a dated amendment of `0024-MADR-cli-mode-polish.md` D1 join P3b by amendment of
  this PLAN.
* *(2026-11-13, the amendment; P1b unless named otherwise:)*
  * a new package `internal/curate`, with its tests and `testdata/` (MADR D11 to D14). P1 creates it, with its doc and
    the fixture alone.
  * `acpclient/update.go` and `acpclient/conn.go`: `Update.Method` (D12), in P1, beside `ToolCall.Kind`.
  * `internal/cli/render/udiff.go`: its line diff moves to `internal/curate`, and the hunk layout stays.
  * `internal/archtest/check.go` and `check_test.go`: rule 13 (D11).
  * a dated note in `0004-MADR-go-module-architecture.md`: rule 13 in its import rules, and `internal/curate` in its
    package map.
  * ~~a new `internal/cli/viewmode.go`~~: the mode is `curate.Mode` (P2 step 1, annotated).

**Out of scope:** everything under `acpserver/`, `agent/`, `tool/`, `acpext/`, `llm/`; print mode's formats; the TUI;
the permission prompt's layout; the sign-in surfaces; magic-cli-remote (MADR, "Not in scope"). A step that seems to
need one of these stops as a deviation. *(2026-11-13: "the TUI" means its renderer, `internal/tui`, and its drawing,
which are `0005-PLAN-v1-feature-scope.md` F9's; the curation it will consume is in scope, as `internal/curate`. Q8 to
Q10 of the MADR are not this plan's to decide.)* *(2026-11-13, later: the owner ruled on Q8 to Q10, all for the TUI
and go-tui-lib; no step of this plan changes. The line view draws each item at its default fold, and `/view`
applies from the next turn, as P2 already has it.)*

## Conventions

* **C1. Each phase re-reads its files at the base it starts from**, and records a moved line in the execution record.
* **C2. Every new or changed test is seen failing on a planted break before it is trusted.** The break is made in a
  scratch copy of the tree (a `git worktree` or a copy under the session's scratch directory), never in the working
  tree; the record names the break and quotes the failure's first line.
* **C3. Goldens** use the existing `golden` helper (`internal/cli/main_test.go`) and its `-update` flag. A golden is
  regenerated only by a step that says so, and its diff is read whole and summarised in the record.
* **C4. Deviations** stop the phase: evidence, resolutions with costs, the consequence of doing nothing; the owner
  chooses; a dated entry here, and in the MADR when a decision or fact changes.
* **C5. Heavy runs** (`make preflight`, `go test ./...`, `make race`, `make check-test-isolation`, `make perf`) run one
  at a time per host: on the session host through `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>`,
  elsewhere through `~/.agents/skills/available-dev-hosts/scripts/run-on-host --need <goos> -- <command>`. A
  single-package run while iterating (`go test ./internal/cli/`, `./internal/cli/render/`, `./acpclient/`, one at a
  time) may stay on the session host; `./internal/cli/...` is several packages and goes through `run-heavy`.
  *(2026-11-13: before any heavy run, `~/.agents/skills/available-dev-hosts/scripts/pick-host` names the host, and the
  run goes there through `run-on-host`, or through `run-heavy` when it names the session host. When the Linux machine
  is busy, a WSL host serves `--need linux`. `go test ./internal/curate/` is a single package and may stay on the
  session host.)*
* **C6. Phase gates:** `make pre-add-check FILES="…"` on every changed Go file; `make preflight` through C5; for P2 and
  later, `make check-test-isolation` through C5. `git diff --exit-code <base> -- acpserver agent tool acpext llm` is
  empty after every phase.
* **C7. One commit per phase**, `git commit --no-edit` after its gates, on the owner's word.

## Implementation Steps

### P0 — Preconditions and answers

1. `git tag -l v0.7.0` prints the tag, and `git merge-base --is-ancestor v0.7.0 HEAD` succeeds. `0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` and
   `0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` are `complete`. If not, stop: this plan waits.
2. ~~Record the owner's answers to Q1 to Q6 in the MADR, set it `accepted`, and amend this PLAN where an answer differs
   from the recommendation.~~ *(2026-11-13: done before execution; the MADR records the rulings and is `accepted`, and
   this PLAN carries them.)* Confirm the MADR still reads `accepted` with Q7 open, and that no later ruling has changed
   a step; a change is a deviation (C4).
3. Re-read every `file:line` the MADR's audit cites, at the base (C1); record each move in this PLAN's execution record.
   `retryLine` is expected to have moved by `0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` P4 step 4b.
4. Apply the MADR's D9 notes to `0008-MADR-native-cli-mode.md` D7, D12, D13 and `0005-MADR-v1-feature-scope.md` "CLI flags", dated, beside the text each changes.
   The amendments of `0030-MADR-resize-and-performance.md` D2, `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b and `0008-MADR-native-cli-mode.md` D5 wait for P3b.
   *(2026-11-13: the note in `0008-MADR-native-cli-mode.md` D7 names `internal/curate` as where the rendering rules
   are decided (MADR D11 to D14). The note in `0004-MADR-go-module-architecture.md` waits for P1b, which writes it with
   archtest rule 13.)*

Acceptance: the tag is an ancestor of the base; the rulings stand; the notes are in place. A records-only
commit (the bootstrap exception).

### P1 — acpclient carries the tool's kind, and a fixture of real agent updates

No visible change.

1. `acpclient/update.go`: `ToolCall` gains `Kind string`, set from `tool_call`'s `kind` and from a `tool_call_update`'s
   when present; an update without one leaves it as it was, as `Title` and `Status` do. Its doc names the ACP kinds.
2. `internal/cli/testdata/turn-realistic.updates.golden`: the `session/update` params, one JSON line each, of a scripted
   turn run through the real acpserver: thoughts; three shell calls, one exiting 3 with two lines of output, one
   printing 40 lines (`printf` with forty arguments); a `read`; an `edit`; a `todo` with its plan; a `grep`; the reply; `usage_update`. The commands are
   `echo`, `printf` and `exit`, so the bytes are the same wherever bash runs; the working directory is replaced by
   `<cwd>`; session ids come from a counter.
3. The test that writes and checks that fixture (below) lives in `internal/cli`, using `acpserver.New` with
   `llmtest.NewScript` and `acpclient/acptest`, and skips where `tool/builtin/shell_test.go` skips ("no usable bash").
4. *(2026-11-13, the amendment:)* `acpclient`: `Update.Method` (MADR D12), set from the notification's method for
   `KindRetry`, `KindPersonality` and a `KindCompaction` from `_gobble/compaction`, and empty for a `session/update`.
5. *(2026-11-13, the amendment:)* Steps 2 and 3 are moved: the fixture is
   `internal/curate/testdata/turn-realistic.updates.golden`, and its test lives in `internal/curate`, which this phase
   creates with its `doc.go` alone ("Stability: internal"; the layer's code is P1b's). The layer's goldens (P1b) and
   the line view's (P2, P3) read that one fixture, `internal/cli`'s tests through a path relative to their package.

| Test | What it checks | Planted break (C2) |
| :--- | :--- | :--- |
| `TestToolCallKind` (`acpclient`) | `tool_call` with `kind: "execute"` gives `Tool.Kind` `execute`; an update with `kind: "read"` changes it; an update without `kind` keeps it; an unknown kind passes through as sent | `translate` not copying `Kind` for `tool_call`; the update case overwriting `Kind` with `""` |
| `TestRealisticTurnFixtureIsCurrent` (`internal/cli`; 2026-11-13: `internal/curate`, step 5) | the scripted turn through acpserver gives exactly the fixture's lines | one fixture line's summary text altered in the copy; `maxSummary` set to 40 in the copy's `acpserver/prompt.go` |
| `TestUpdateMethod` (`acpclient`, 2026-11-13) | each of `_gobble/retry`, `_gobble/compaction` and `_gobble/personality` gives its method in `Update.Method`; a `session/update` leaves it empty | `Method` not set for `_gobble/personality`; `Method` set to the variant for a `session/update` |

Acceptance: both tests pass and each break failed; `go test ./acpclient/` and `go test ./internal/cli/` pass on the
session host, one package at a time (C5); C6. *(2026-11-13: the three tests, and `go test ./internal/curate/` as well.)*

### P1b — The curation layer, built and tested alone (added 2026-11-13; MADR D11 to D14)

No visible change: nothing outside `internal/curate`, `render/udiff.go`'s line diff and `internal/archtest` calls the
layer yet. Every rule of the MADR's D14 is proven here, at the layer, before any renderer consumes it.

1. `internal/curate`: `Mode`, `ParseMode` and `Modes`; `Curator`, made per turn with a mode, with `Feed(acpclient.Update)`,
   `Ask(callID)` and `End(acpclient.Result)` each returning `[]Change`, and `Status()` returning a `StatusRow`;
   `Change` (`Op` `Open`, `Grow` or `Close`, an `ID` and an `Item`); the items `ToolCard`, `ActivityRow`, `Diff`,
   `Thought`, `Plan`, `Reply`, `Note`, `TurnEnd` and `Raw`; `Fold` (`Hidden`, `Summary`, `Tail`, `Full`) and `Sep`
   (D12, D13). The package doc names MADR D11 to D16.
2. The rules of D14, per mode: what each input makes, folds, groups and closes; grouping by ACP kind (D3); the failure
   tail of five lines, without the shell tool's status line, with the count of lines above (D4); thoughts `Hidden` in
   `default` and `terse` with `StatusRow` thinking (D5); `Raw` items, with `Update.Method` for a notification (D8).
3. The line diff: `render/udiff.go`'s `splitLines`, `myers` and `backtrack` move to `internal/curate`, which counts
   `+a −d` from them; `render.UnifiedDiff` keeps its hunk layout and calls the layer's line diff. The `udiff-*.golden`
   files do not change.
4. `internal/archtest`: rule 13, "`internal/curate` imports, in non-test code, only the standard library, `acpclient`
   and `jsonx`", with its cases in `check_test.go`. In the same commit, the dated note in
   `0004-MADR-go-module-architecture.md`: rule 13 under "Import-boundary rules", and `internal/curate` in the package
   map's note of internal packages (MADR D11).
5. Goldens (C3): `internal/curate/testdata/curate-<mode>.golden` for the four modes, from the P1 fixture. Each is the
   change stream in a plain text form, one line per `Change` (its op, its ID, its item's type and fields, its fold),
   and the `StatusRow` after each input. The form is the layer's, readable without a terminal, and is also what P3's
   replay test reads back.

| Test | What it checks | Planted break (C2) |
| :--- | :--- | :--- |
| `TestModeGoldens` (`curate`) | the fixture in each mode gives exactly its golden | `default`: rows keyed by title instead of ACP kind; `verbose`: an item opened for a `pending` call; `terse`: one row per kind; `raw`: `KindOther` skipped |
| `TestModeTable` (`curate`) | one row per `acpclient.Kind` and mode, D14's, each giving the item, fold and closing it names; a kind without a row fails | the thought row in `default` set to `Full`; a new `Kind` constant added in the copy's `acpclient` with no row |
| `TestExitStatusOnce` (`curate`; was P3's `TestDefaultExitStatusOnce`) | in `default`, the failed call's `ToolCard` is `Tail`, with `exit status 3` once in its summary and its two output lines, without `[exit status 3]`; each successful call's card is `Hidden` | the tail keeping the `[exit status N]` line; a success's card `Tail` |
| `TestShellStatusLineRecognised` (`curate`; was P3's) | the layer's recogniser accepts what `tool/builtin`'s shell result writes for an exit of 3 and for a stop at a timeout, run through the real tool where bash runs | the pattern changed to `[exit code`; the stop form dropped |
| `TestClosingRules` (`curate`; the layer's half of P3's `TestGroupClosesBefore`) | in `default`, an open row closes before a `Reply`, a `Plan`, an `Ask`, a call of another kind, a `Note` and `End`, and not on a thought; in `terse`, a call of another kind joins the row | `Ask` not closing the row; a thought closing it |
| `TestStatusRow` (`curate`; the layer's half of P3's `TestSpinnerLabelGrows` and `TestThinkingFolded`) | after a group's first and second calls finish while a third runs, `Status()` holds the two and the running call, then grows; while thoughts stream it says thinking; after the row closes it holds no row | the snapshot not updated when a call finishes; thinking not set |
| `TestClosedStaysClosed` (`curate`) | each ID is closed once and has no change after; at `End`, every finished call of the fixture is in exactly one closed `ActivityRow`, in every mode but `raw` | a `Grow` sent after `Close`; the last row dropped at `End` |
| `TestDiffStat` (`curate`) and the existing `udiff` goldens (`render`) | `+a −d` for a new file, one hunk and two hunks; `render.UnifiedDiff` unchanged | the added count off by one; a hunk's context lost in the move |
| `TestRule13` (`archtest`) | a fixture where `internal/curate` imports `internal/cli/render`, `github.com/charmbracelet/x/ansi` or `personality` fails rule 13; one that imports `acpclient` and `jsonx` passes; a test import of `acpserver` passes | the rule's check skipping non-test imports; the allow-list holding `personality` |

Acceptance: the tests pass and each break failed; `go test ./internal/curate/`, `./internal/cli/render/` and
`./internal/archtest/` pass on the session host, one package at a time (C5); the four goldens read as the MADR's D14
table says, checked by eye against it and noted in the record; C6. Its own commit, after P1's.

### P2 — The modes, chosen; `raw`; the flags retired; the stream proven unchanged

The view's rendering in `verbose` is today's with `--verbose --show-thinking`; in `default` and `terse` it is still
today's default. Only the plumbing and `raw` change here.

1. ~~`internal/cli/viewmode.go`: `type viewMode` with `raw`, `verbose`, `default`, `terse`; parsing and the list for
   `/view`, `--view` and the setting.~~ *(2026-11-13, the amendment: the mode is `curate.Mode` from P1b; `/view`,
   `--view` and the setting parse with `curate.ParseMode` and list `curate.Modes`. No `viewmode.go` is written.)*
2. `flags.go`: `--view` (`enum:",raw,verbose,default,terse"`, `default:""`, as `--thinking` does, so an absent flag
   leaves the setting to decide; with completion); `--verbose` and
   `--show-thinking` removed (Q3 (a)). `display` holds the mode; `stats` stays.
3. `internal/config/settings.go`: `View string` (`yaml:"view"`), validated against the four, known to `gobble config`
   and `--set`; an unknown value is a diagnostic and `default` is used.
4. `line.go`: the mode by D6's precedence; the connection starts with `Params` on (D8). `slash.go`: `/view` in
   `localCommands`, with its description and hint; the handler of D6.
5. `view.go`: in `raw`, each update's `Params` written as one line on stdout, and each `_gobble/*` notification as
   `{"method":…,"params":…}`; the status line as in every mode. *(2026-11-13, the amendment: in `raw` the view feeds a
   `curate.Curator` and writes each `Raw` item as its line, the method from `Update.Method`; `line.go` hands
   `_gobble/personality` on to the view after setting the voice. The other modes keep today's path until P3.)*
6. The four built-in catalogs' `lines.hidden` text names `/view verbose`.

| Test | What it checks | Planted break (C2) |
| :--- | :--- | :--- |
| `TestViewModeLeavesACPUnchanged` | one scripted turn through the line session (`agentServe` replaced, both directions teed) in each of the four modes: the frames are byte-equal across modes | the copy's line session appending the mode's name to the prompt text; the copy turning `Compaction` off in `raw` |
| `TestViewFlag` | `--view terse` starts in `terse`; `--view bogus` exits 2 naming the four; `--verbose` and `--show-thinking` exit 2 as unknown flags | the `enum` tag dropped; `--verbose` left in `SharedFlags` |
| `TestViewSetting` | `view: terse` starts in `terse`; `--view verbose` wins over it; `view: bogus` gives a diagnostic and `default` | the setting read after the flag; the bad value kept |
| `TestSlashView` | `/view` prints the mode and the four; `/view verbose` changes the next turn's mode and not the running one's; `/view bogus` is an error and the mode stays; `/help` lists `/view`; completion offers the four | the handler not storing the mode; `view` missing from `localCommands` |
| `TestViewRawGolden` (`view-raw.golden`) | the fixture through `raw` equals the fixture's own lines, and equals `-p --output-format stream-json`'s update lines for the same turn | `raw` writing `u.Text`; `raw` skipping `KindOther` |
| `TestViewVerboseIsTodaysVerbose` | the fixture through `verbose` equals today's `--verbose --show-thinking` rendering, captured as `view-verbose-p2.golden` before step 2 | `verbose` wired to the default display |
| `TestHiddenLinesHintNamesView` (`personality`) | every built-in catalog's `lines.hidden` names `/view verbose` and none names `--verbose` | one catalog left unchanged |

Acceptance: the tests pass and each break failed; existing tests that named `--verbose` or `--show-thinking` are
changed to `--view verbose`, each listed in the record; C6, including the empty diff under `acpserver/` and the rest.

### P3 — The rendering per mode (stage 1 of Q1's ruling)

*(2026-11-13, re-cut by the amendment, MADR D13, D14 and D16: the line renderer becomes a thin consumer of the
layer. The rules of steps 2 to 4 are the layer's since P1b. Here `view.on` feeds the turn's `curate.Curator` and
draws the `Change`s it returns: an `ActivityRow` on `Close` through `render.ToolGroup` or `render.ActionLine`; a
`ToolCard` by its `Fold`; a `Plan` by its `Form`; a `Reply` through the Printer as today; `Sep` as one blank line.
`view.go` keeps no `switch` on `acpclient.Kind`, and never reads the mode to decide what to show. Each step below
is read that way.)*

1. `render`: `ToolGroup(kind, calls, width, glyphs, voice)` for D3's line, cut to one row *(2026-11-13: it takes a
   `curate.ActivityRow`, as `ActionLine` does, and `DiffStat` lays out the layer's counts)*; `DiffStat(diff)` (`+a −d`);
   `ActionLine` for `terse`; `PlanLine` for `terse`'s plan.
2. `view.go`, `default`: an open group per D3, closed before reply text, a plan, a permission prompt, another kind,
   a retry or compaction note, and the turn's end; a failed call's detail per D4 (its summary, then the output's last
   five lines without the shell tool's status line); no output for a success; diffs as `DiffStat` on the group line
   (Q5 (a)); thoughts not printed (Q6 (a)); one blank line before reply text that follows an action line.
   *(2026-11-13: decided by the layer's items, steps 2 to 4 alike; the view draws them.)*
3. `view.go`, `verbose`: one line per call at its terminal status, no pending header; the output block, unbounded,
   when the call has one, else its summary, then its diffs, unbounded (MADR D4: the summary is the first text block,
   by `0008-MADR-native-cli-mode.md` D19 item 3); thoughts dimmed under a `thinking` label, a blank line after.
4. `view.go`, `terse`: one action line per run of actions between replies; one line per failure; `PlanLine`.
5. `line.go` and `view.go`: the spinner restarted while a call runs in every mode but `raw`, labelled with the running
   call's title, or `thinking` while thoughts stream; stopped before every stdout write, as today. *(2026-11-13, Q1's
   ruling, stage 1:)* in `default` the label shows the open group growing, `ran go build ./... ✓ +1 +2 · go test
   ./...` (the finished calls as `ToolGroup` lays them out, then the running call's title), and in `terse` the action
   line so far; the view hands the spinner a new label as each call starts or finishes; `render/spinner.go` cuts the
   label to the terminal's width less the frame, so it never wraps. *(2026-11-13: the label is drawn from
   `Curator.Status()`, the layer's `StatusRow`; the cut to the width is the renderer's.)*
6. `personality`: keys for each kind's phrase (`ran {n} commands`, `read {n} files`, `searched {n} times`, `edited {n}
   files`, `fetched {n}`), `thinking`, and the `n failed` suffix, in all four built-in catalogs.
7. Goldens regenerated (C3): `view-<mode>-plain.golden` and `view-<mode>-color.golden` for the four modes from the
   fixture at width 80, and `view-default-narrow.golden` at width 40; `view-verbose-p2.golden` removed, since
   `verbose` changes by step 3.

| Test | What it checks | Planted break (C2) |
| :--- | :--- | :--- |
| `TestViewModeGoldens` | the fixture in each mode, plain and colour, matches its golden | `default`: groups keyed by title instead of kind; `verbose`: the pending header printed again; `terse`: one action line per kind; `raw`: a trailing space on each line |
| `TestToolGroupLine` (`render`) | a group of one reads as its call; of four, `ran 4 commands ✓ a · b · +2`; at every width from 20 to 200 the line is one row, never wider than the width; a failure count shows `✗ n failed` | the width check off by one; the `+n` count wrong when titles are cut |
| ~~`TestDefaultExitStatusOnce`~~ *(2026-11-13: moved to P1b as `TestExitStatusOnce`; here the default golden shows it)* | in `default`, the fixture's failed call shows `exit status 3` exactly once, followed by its two output lines; the successful calls print no output line | the tail keeping the `[exit status N]` line; success output printed |
| ~~`TestShellStatusLineRecognised`~~ *(2026-11-13: moved to P1b, with the recogniser)* | the view's recogniser accepts what `tool/builtin`'s shell result writes for an exit of 3 and for a stop at a timeout, run through the real tool where bash runs | the recogniser's pattern changed to `[exit code`; the stop form dropped |
| `TestGroupClosesBefore` *(2026-11-13: the closing rules are P1b's `TestClosingRules`; this keeps the screen order, the group line written before the permission question and before the status line)* | in `default`, an open group's line is written before reply text, a plan, a permission prompt (its question drawn after the group line), another kind, a retry note, the turn's status line; a thought does not close it | the permission path not closing the group; a thought closing it |
| `TestThinkingFolded` *(2026-11-13: the fold is P1b's; this keeps the drawing of the label and of `verbose`'s thoughts)* | `default` and `terse` print no thought text; the spinner's label says `thinking` while thoughts stream; `verbose` prints them under the label, separated from the reply | the thought printed in `default`; the label missing in `verbose` |
| `TestSpinnerRestarts` | while a call runs after earlier output, the spinner draws on stderr with the call's title, and stops before the next stdout write | the restart removed; the stop before a write removed |
| `TestSpinnerLabelGrows` *(2026-11-13: the snapshot is P1b's `TestStatusRow`; this keeps the label's text and its cut to the width)* | in `default`, after the first and second calls of a group finish while a third runs, the label reads `ran <a> ✓ +1 · <c>`, then grows; at every width from 20 to 200 the frame and label are one row, never wider than the width | the label not updated when a call finishes; the cut ignoring the frame's width |
| `TestReprintKeepsCuratedTurn` | a resize during a `default` turn reprints the group lines and the reply as written | a group line written with `emit` (not kept by the transcript) |
| `TestLineViewConsumesTheStream` (2026-11-13; MADR Confirmation 13) | for each mode, P1b's golden change stream, read back and drawn by the line renderer alone, gives the same stdout and stderr bytes as the fixture's updates through `view.on` | the copy's view reading `u.Tool.Content` in `on` instead of the item; the copy's view drawing a `Hidden` card |
| `TestViewHasNoKindSwitch` (2026-11-13) | `internal/cli/view.go`, parsed, has no `switch` or comparison on `acpclient.Kind`, and does not read the mode outside the `Curator`'s construction | a `case acpclient.KindToolCall:` restored in the copy |
| the existing view tests (`view_test.go`, `view_wrap_test.go`, `reprint_test.go`, `compactionview_test.go`, `cutoff_test.go`, `retry_test.go`, `limit_test.go`) | pass, run in the mode each was written for | — (existing; each changed test listed in the record with its diff) |

Acceptance: the tests pass and each break failed; the four modes' goldens read as the MADR's D2 table says, checked
by eye against it and noted in the record; `make perf` within its budget (C5); C6. *(2026-11-13: and
`TestLineViewConsumesTheStream` passes in all four modes.)*

### P3b — The status row (stage 2 of Q1's ruling; MADR D10)

Stage 1's output is unchanged off a terminal and when a group closes; only the bottom row's drawing changes.

*(2026-11-13, the amendment, MADR D16: this phase stays the line renderer's alone. The row draws the layer's
`StatusRow`, as stage 1's label did; the yields, the collapse, the fallback and Q7 are not the layer's, and the TUI
inherits none of them. Nothing in `internal/curate` changes in this phase.)*

0. **Q7, type-ahead echo.** Before step 1, write `TestStatusRowUnderEcho` (below) against a first, throwaway
   status row in a scratch copy, show the owner its screens for the MADR's three options, and record the ruling as a
   dated note under the MADR's Q7. Under (a) the steps stand; under (b) or (c) this PLAN is amended first (Scope
   names the files), and `0024-MADR-cli-mode-polish.md` D1 is amended by a dated note in this phase.
1. `internal/cli/statusrow.go`: a `statusRow` owning one `inline.Region` on stderr, made with
   `inline.WithoutCursorQuery()`, `inline.WithHeight(1)` and the editor's reflow choice (`editor.go:153-156` at
   `f078889`); `Show(text)` (frame and text, cut to the width), `Hide()`, `Print(text)`, `Resize(width)`; a frame
   ticker that takes the view's lock, replacing the spinner's goroutine. Nothing is drawn when stderr is not a
   terminal.
2. `line.go` and `view.go`: the turn's status row replaces `render.StartSpinner` (`line.go:341-346` at `f078889`);
   `beforeWrite` (`view.go:436-442`) becomes the row's `Hide`, and the view shows the row again after each write
   while a turn runs; the row shows what stage 1's label showed.
3. `output.go`: while a row is shown, `diag` and `logRecord` write through its `Print`; otherwise as today.
4. `permission.go`: `askPermission`'s `v.quiet()` becomes a yield: the open group closes (its line written), the row
   is hidden; the prompt draws as today; the next update shows the row again.
5. `view.go`: `tryReprint` writes the reprint as the row's `Print(out)` while it is shown, else `out` alone as today;
   `resized` passes the width to the row.
6. `render/spinner.go`: removed, or kept for its frames alone, whichever leaves no second writer of the bottom row.
7. Dated amendments, written now (MADR D9): `0030-MADR-resize-and-performance.md` D2, the live area also holds the open group's row;
   `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D14 step 5b, mid-turn with the row shown the reprint is `region.Print(out)`; `0008-MADR-native-cli-mode.md` D5,
   the status row replaces the spinner.
8. By the owner's eye, recorded with what was seen: a grouped turn with a long command, a permission question mid
   group, a resize mid group and a log warning mid group, in Windows Terminal, in Console Host and under a terminal
   multiplexer (tmux), and in one macOS and one Linux terminal. No automated test covers the first three.

Tests run on go-tui-lib's `inlinetest.Screen` with `Cooked` set, the terminal's state during a turn, unless named
otherwise; each frame is the bytes up to a `?2026l`.

| Test | What it checks | Planted break (C2) |
| :--- | :--- | :--- |
| `TestStatusRowOneRow` | at widths 20 to 200, with titles of every length and wide characters, the row is one screen row and the cursor stays on it | the row's text not cut to the width; `WithHeight(1)` left out |
| `TestStatusRowYields` | a stdout write, a `Notef`, a log record, a permission question and a mid-turn reprint each leave the screen with their text whole and the row drawn once, below it, in the last frame; no frame shows the row over other text | `diag` writing past the row; the reprint written as `out` alone with the row shown |
| `TestStatusRowCollapses` | after a group closes, the screen and the scrollback hold stage 1's group line and none of the row's states; the transcript holds the same line | the row committed instead of hidden; the group line written to stderr |
| `TestStatusRowOffTerminal` | with stderr not a terminal, stdout and stderr equal stage 1's for the fixture, byte for byte | the row drawn without the terminal check |
| `TestStatusRowResize` | a resize mid group (60 to 40 columns, each reflow kind of `inlinetest.Screen.Resize`) leaves one row, redrawn at the new width, and no cursor report is asked for | `WithoutCursorQuery` left out (a `CSI 6 n` appears); the width not passed on |
| `TestStatusRowUnderEcho` | an echoed key and an echoed Enter written between frames, as the terminal's echo writes them: what the screen shows after the next two frames, under the answer to Q7 | the Q7 answer's mitigation removed |
| `TestViewModeGoldens`, `TestViewModeLeavesACPUnchanged` | unchanged from P3 and P2: stdout and the ACP frames are what they were | — (existing) |
| `make race` | the race detector over `internal/cli`, with the notification goroutine, the permission callback, the resize router, the settle timer and the row's ticker sharing `v.mu`, through `run-heavy` on the session host and `run-on-host --need linux`, one at a time per host (C5) | a copy whose ticker draws without taking `v.mu` fails under `-race` |

Acceptance: Q7 ruled and recorded; the tests pass and each break failed; `make race` passes on both hosts; step 8's
observations recorded; the three amendments in place; C6. Its own commit, after P3's.

### P4 — Documents, the three hosts, close-out

1. `docs/architecture.md` (the CLI's view), `README.md` (`--view`, `/view`, `view:`), `docs/README.md` (the pair's
   status). *(2026-11-13: `docs/architecture.md` also describes `internal/curate`, its input and items, and that the
   TUI mode will draw from it.)*
2. The gates (C6) on macOS through `run-heavy`, and on Linux and Windows through
   `run-on-host --need linux` and `run-on-host --need windows`, one at a time per host.
3. A live check on the session host, with the owner's own credentials and the owner present: one turn in each mode,
   the output read by the owner. No provider is called by any automated test.
4. The MADR's Completion section; this PLAN `complete`.

Acceptance: Confirmation 1 to 11 met and recorded; the commit is the owner's. *(2026-11-13: Confirmation 1 to 14.)*

## Verification

* MADR Confirmation 1: `TestViewModeLeavesACPUnchanged`; `git diff --exit-code <base> -- acpserver agent tool acpext llm`
  empty; `acpserver/testdata/session.golden` unchanged.
* Confirmation 2: `TestRealisticTurnFixtureIsCurrent` and `TestViewModeGoldens`, with their breaks.
* Confirmation 3: `TestDefaultExitStatusOnce`, `TestShellStatusLineRecognised`.
* Confirmation 4: `TestViewFlag`, `TestViewSetting`, `TestSlashView`.
* Confirmation 5: `TestViewFlag`, `TestHiddenLinesHintNamesView`.
* Confirmation 6: `TestReprintKeepsCuratedTurn`.
* Confirmation 7: P4 step 2.
* Confirmation 8: `TestSpinnerLabelGrows`.
* Confirmation 9: `TestStatusRowOneRow`, `TestStatusRowYields`, `TestStatusRowCollapses`, `TestStatusRowOffTerminal`,
  `TestStatusRowResize`, with their breaks.
* Confirmation 10: P3b's `make race` on two hosts.
* Confirmation 11: P3b step 0 (Q7 ruled) and step 8 (the owner's observations).
* *(2026-11-13, the amendment:)* Confirmation 3 is now held first at the layer, by P1b's `TestExitStatusOnce` and
  `TestShellStatusLineRecognised`, and on screen by P3's `default` golden.
* Confirmation 12: P1b's `TestModeGoldens`, `TestModeTable`, `TestClosingRules`, `TestStatusRow` and
  `TestClosedStaysClosed`, with their breaks.
* Confirmation 13: P3's `TestLineViewConsumesTheStream` and `TestViewHasNoKindSwitch`.
* Confirmation 14: P1b's `TestRule13`, and `make archtest` within `make preflight` (C5, C6).
* Confirmation 15: not this plan's; `0005-PLAN-v1-feature-scope.md` F9 carries it.

## Rollout and Rollback

* **Rollout.** One commit per phase on `main`; no release is cut by this plan. The next `v0.7.x` or `v0.8.0` carries
  it, by a record of its own.
* **Rollback.** Revert the phase commits in reverse order. *(2026-11-13: P1b's commit is reverted after P2's and
  before P1's; with it go archtest rule 13 and its note in `0004-MADR-go-module-architecture.md`.)* A `view:` key left in `settings.yaml` is ignored by an
  earlier build, which ignores unknown keys (`internal/config/merge.go:53`). Command lines that use `--view` fail on an
  earlier build with exit 2; those that used `--verbose` work again.
* **No ACP effect either way.** The agent's stream is the same before, during and after, which P2's test holds.

## Execution record

Not started.

* **2026-11-13, amended before execution.** The owner's ruling, "Shared layer + go-tui-lib survey". The MADR gains D11
  to D16 and Q8 to Q10; this PLAN gains P1b and the dated notes in Scope, C5, P0, P1, P2, P3, P3b, P4, Verification
  and Rollback. No phase has run, and no code changed.
* **2026-11-13, the owner's rulings on the MADR's Q8 to Q10** (Q8 "Yes, up to full detail", Q9 "No, from the next
  turn", Q10 (a), "Draft a go-tui-lib record now"), recorded in the MADR. No step of this plan changes.
