---
status: in-progress
date: 2026-10-20
associated-madr: "0024-MADR-cli-mode-polish.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement polishing CLI mode, beginning with `/help`

Associated MADR: [0024-MADR-cli-mode-polish.md](0024-MADR-cli-mode-polish.md)

*Proposed. Approved on the owner's word "accepted, approved to proceed" after `6ab7928`, and P1 run the same day,
with deviations 1 and 2 below. It starts after the owner approves it, on `main` at or after `9188f63`. The MADR's Q1–Q3 are answered
(2026-10-20): entries indented under headings, D3's descriptions as proposed, `/help <command>` a later amendment.*

## Goal

`/help` in the line session prints an aligned, wrapped list with informative descriptions, and every local command
prints with the terminal out of raw mode (MADR D1–D4).

## Scope

- **In:** `internal/cli/line.go` (the loop), `internal/cli/slash.go` (the local commands, `/help`'s layout and
  descriptions), ~~a new `internal/cli/helplist.go` for the layout if `slash.go` grows past it~~ *(deviation 1: a new
  `internal/cli/render/commands.go` and its test)*; `acpserver/commands.go`
  (the agent's descriptions and hints); their tests; `acpserver/testdata/session.golden`; `docs/README.md`.
- **Out:** `/help <command>` (MADR Q3, unless the owner takes it); the agent's own `/help` layout, which ACP clients
  render; any other CLI-mode polish, which amends the MADR first (D5).

## Implementation Steps

### P1 — `/help` and the local commands

1. **The raw-mode test first** (D4): a line-session test whose `raw` function records whether the terminal is raw at
   each write, driving `/help`, `/new` and `/nosuch`; it fails today, each written while raw.
2. **D1:** the loop leaves raw mode after `ReadPrompt` returns a submission and re-enters it before the next read;
   `ls.slash` and the turn both run out of raw mode; `/edit` calls `compose` directly. `/edit`'s existing test passes
   unchanged.
3. **The layout tests** (D4): a fixed command set rendered at 100, 60 and 32 columns, checked line by line for the
   width, the description column, the too-wide entry and the wrap's indent; they fail against today's
   `helpText`.
4. **D2:** `helpText` takes the width, read with `xterm.GetSize` on standard output when it is a terminal, else
   `caps.Out.Width`; it lays the two sections out as D2 says, with `displayWidth` for column counts and the name bold
   when colour is on.
5. **D3:** the descriptions and hints of D3's table (or the owner's), in `slash.go` and `acpserver/commands.go`;
   `session.golden` regenerated with `ACPTEST_UPDATE=1` and its diff read line by line: only descriptions and hints
   change. `ambient_test.go`'s help check follows the new heading.
6. **Records:** this plan's execution record; `docs/README.md`'s rows.

## Verification

- Each new test seen failing on a scratch copy of the code before its change, and the failure quoted in the record.
- A pseudo-terminal capture of `gobble`, built to a scratch directory and run with a scratch home and no credential,
  at 100 and 60 columns: between `/help` and the next prompt no bare `\n`, and, replayed with xterm's rules, every line
  starting at column 0 and none wider than the terminal. Run on macOS and on the Linux machine.
- **Windows, measured before and after:** the console mode `MakeRaw` sets for output on the Windows laptop, and
  `/help` there before and after the change, by whatever terminal the laptop allows a capture from (a ConPTY session
  over ssh, or the owner's look at a console). What was and was not seen is recorded; a Windows difference that D1
  does not fix is a deviation, stopped and asked.
- The pre-add check on the changed `.go` files; lint for linux, darwin and windows; `go test -count=1 ./...`,
  `CGO_ENABLED=1 go test -race -count=1 ./...` and `make preflight` on macOS, the Linux machine and the Windows laptop,
  the remote two on a copy of the tree; `check-records` and `markdownlint`.

## Rollout and Rollback

One commit, the owner's. Reverting it restores the old list, descriptions and loop; nothing is stored or migrated.

## Execution record

**Deviation 1, 2026-10-20, decided by the owner** ("In internal/cli/render"): the layout counts terminal columns, a
wide East Asian character taking two, and the package's count is `displayWidth`, unexported in `internal/cli/render`
(`render/table.go:16`), which `internal/cli/helplist.go` cannot call.

- **Chosen:** the layout in a new `internal/cli/render/commands.go`, beside `displayWidth`, with its test
  `internal/cli/render/commands_test.go`; `slash.go` calls it. No code is duplicated and nothing new is exported but
  the layout itself.
- **Not chosen:** exporting `displayWidth` from `render/table.go` and keeping the layout in `internal/cli`.
- **Files added to P1's scope:** `internal/cli/render/commands.go`, `internal/cli/render/commands_test.go`.

**Deviation 2, 2026-10-20, decided by the owner** ("Keep 0022's hint"): MADR D3's table set `/personality`'s hint to
`[name]`, but accepted 0022-MADR decided "Its input hint names the built-ins: `[boring|dry|snarky|bombastic]`"
(`0022-MADR-gobble-voice-and-personalities.md:229`), and `acpserver/personality_switch_test.go:159` pins it; every ACP
client that shows a hint, the phone app among them, reads it.

- **Chosen:** the hint stays `[boring|dry|snarky|bombastic]`; D3's row is annotated, its description "List the
  personalities, or switch to one", since the hint names them and `/personality` with no name lists each with its level
  and summary (0022-MADR). 0022-MADR and the test are unchanged.
- **Not chosen:** D3's `[name]`, with a dated note in 0022-MADR and the test changed to pin it.
- No file is added to the step's scope.

**P1, run 2026-10-20,** on `main` at `6ab7928`, with deviations 1 and 2 above.

1. **The raw-mode test:** `TestLocalCommandsPrintOutOfRaw` (`internal/cli/slash_test.go`) wraps the harness's stdout
   and stderr so each write records whether the terminal was raw, then sends `/help`, `/nosuch` and `/new`. It looks
   for each command's own output (`Commands`, `unknown command /nosuch`, `new session`), not the editor's echo of what
   was typed, which is rightly written in raw mode: a first draft that looked for `/help` matched the echo. On the
   unchanged code it failed for all three:

   ```text
   slash_test.go:249: "Commands" was written with the terminal in raw mode
   slash_test.go:249: "unknown command /nosuch" was written with the terminal in raw mode
   slash_test.go:249: "new session" was written with the terminal in raw mode
   ```

2. **D1:** the loop (`internal/cli/line.go`) runs `ls.slash` and the turn inside one `outOfRaw` per submission;
   `slash` loses its `outOfRaw` parameter, and `/edit` calls `compose` directly, already out of raw mode. The test
   passes; `TestSlashEdit`, `TestLineSessionTurn` (raw mode entered twice, restored twice) and the other slash and
   line-session tests pass unchanged.
3. **The layout tests** (`internal/cli/render/commands_test.go`, deviation 1): `TestCommandListAligned` (60 columns)
   and `TestCommandListNarrow` (32), each against text worked out by hand from D2 before the layout was written;
   `TestCommandListFits` (no line wider than 100, 80, 60, 45, 40, 32 or 20 columns, none ending in a space);
   `TestCommandListColor` (only the name bold, at the plain list's columns); `TestCommandListSanitizes`. On a scratch
   copy of the tree whose `CommandList` printed today's format, all five failed: the two layouts unequal; lines of 103
   and 132 columns at every width; "no bold name"; and `an escape reached the list: "Agent commands:\n  /tpl — Run
   it\x1b[2J now\n"`.
4. **D2:** `render.CommandList` (`internal/cli/render/commands.go`) and `helpText` (`internal/cli/slash.go`), which
   reads standard output's width with `GetSize` when it is a terminal, else `caps.Out.Width`. Two details beyond D2's
   text, both within it:
   - the name column is also at most half the room, `(width-4)/2`, so a description keeps at least half the line;
   - `/help` lays out for one column less than the terminal: a capture at 100 columns had a line of exactly 100, which
     a terminal that wraps as soon as its last column is written would follow with a blank row.
5. **D3:** the descriptions and hints, in `internal/cli/slash.go` and `acpserver/commands.go`, with deviation 2's
   `/personality` row. `session.golden` regenerated with `ACPTEST_UPDATE=1`: it failed before, and a script compared
   it with `HEAD` frame by frame, line 4 alone changed, and in it only descriptions and hints, `/clone`'s unchanged,
   names and order the same. `TestSlashHelpUnion`, `TestLineAmbient` and `TestCommandsAnswer` follow the new text.
6. **Records:** this record; the statuses; `docs/README.md`'s rows.

**The pseudo-terminal captures** (`gobble` built to a scratch directory, run with a scratch home and no credential,
`/help` then `/exit`; a check replays the bytes with xterm's rules):

- **Before, macOS, 100 columns:** `bare LF 22`, the list one row of 1106 columns: `NOT ok`.
- **After, macOS:** 100 columns, `28 lines, widest 99 of 100 columns, bare LF 0`, `ok`; 60 columns, `43 lines, widest
  59 of 60 columns, bare LF 0`, `ok`.
- **After, the Linux machine:** the same figures at both widths, `ok`, checked from the list's heading: the session the
  list opened printed a log line first (below).

At 100 columns the list reads:

```text
Commands
  /help              Show this list of commands
  /exit              Leave gobble (Ctrl+D also leaves)
  /quit              Leave gobble; the same as /exit
  /edit [text]       Write the prompt in your editor, starting from text; it is sent when the
                     editor closes
  /new               Start a new session in this directory, closing the current one

Agent commands
  /compact [focus]   Summarize the conversation to free context; focus says what the summary
                     should keep
  /usage             Show this session's token counts and cost
  ...
  /personality [boring|dry|snarky|bombastic]
                     List the personalities, or switch to one
  ...
```

**Windows, measured and not seen.**

- `golang.org/x/term` `v0.47.0`'s Windows `makeRaw` clears only input flags (echo, processed and line input) and sets
  virtual-terminal input, on the handle it is given (`term_windows.go:25-36`); gobble gives it standard input
  (`internal/cli/editor/editor.go:147-153`). The console's output mode is not touched, so raw mode cannot drop a line
  end there, and D1 changes nothing on Windows but the order of restore and print.
- A capture was tried: today's and the fixed binary, cross-compiled and copied to the Windows laptop, run through
  `ssh -tt` in a ConPTY console. The console opened and closed with nothing written, and so did a plain `echo`: the
  laptop's ssh runs no command under a pseudo-terminal. No Windows capture was made, and the owner's look at a console
  was not asked for. The two binaries stay in the laptop's scratch directory for that look.

**Gates:**

- the pre-add check on the eight changed `.go` files: `go-precheck: 8 file(s) clean (gofmt, golint, govulncheck)`;
- macOS: `go test -count=1 ./...` and `CGO_ENABLED=1 go test -race -count=1 ./...`, 35 packages ok each; `make
  preflight`, `preflight passed` (golangci-lint `0 issues.`, staticcheck for three systems, "No vulnerabilities
  found.");
- the Linux machine and the Windows laptop, over ssh with `-o BatchMode=yes -o ConnectTimeout=5`, on a copy of the
  working tree, Windows with `TMP` and `TEMP` outside the user's home: the tests and the race detector, 35 packages ok
  on each; `make preflight`, every step through `apidiff` passed (golangci-lint `0 issues.`), stopping at
  `verify-build-metadata` with `git rev-parse HEAD` exiting 128, since a copy has no commit; `make check-records
  markdownlint`, "0 issues in 0 files".

**Found, not done here (for the next CLI-polish items, MADR D5):**

- **A log record printed into the conversation.** On the Linux machine, opening the session printed
  `time=… level=WARN msg="prompt: a file was not used" file=<home>/.agents/skills reason="not loaded: the project is
  not trusted (gobble trust, or /trust)"`, a 178-column log line, where a user expects a short note.
- **Its path was the account's real home,** though the run set `HOME` to a scratch directory: that directory's lookup
  may not follow `HOME`. Not investigated.

**Not done, named:** `/help <command>` (MADR Q3, a later amendment); the agent's own `/help` layout, which ACP
clients render.

## Amendment (2026-10-20): P2, the log record on a terminal, and the home walk's test

*Proposed,* for the MADR's amendment of 2026-10-20 (D6, D7). It starts after the owner approves it, and after P1's
commit. Q4 is answered (2026-10-20): option A, a `Warning:` line on a terminal.

**Correction of P1's finding (2026-10-20).** "Its path was the account's real home, though the run set `HOME` to a
scratch directory: that directory's lookup may not follow `HOME`" is wrong: the lookup follows `HOME`, and the capture
put `HOME` below the account's home, outside a repository. See the MADR's amendment, finding 2.

**Scope.** In: `internal/logging/logging.go` and a new `internal/logging/terminal.go` (the terminal form), their
tests; `internal/cli/main.go` (stderr's terminal and width passed in); `prompt/trust_test.go`; 0004-PLAN's dated note;
`docs/README.md`. Out: a record logged while the editor reads (the MADR's out-of-scope item); the wording of any one log
message.

**Steps.**

1. **The handler's tests first,** in `internal/logging`, each seen failing on today's text handler:
   - a warning with a path under `$HOME` and a reason: one `Warning:` line, attributes `key: value` in parentheses,
     the path with `~`, no `time=` or `level=`;
   - an error: `Error:`;
   - a record wider than the width: wrapped at spaces, no line wider than the width less one, continuation lines
     indented two;
   - a sensitive attribute: `[REDACTED]`, as `TestRedaction` has it for the text handler;
   - an info record: nothing on stderr; the file still gets it;
   - stderr not a terminal: slog's text, as `TestFileGetsLevelStderrGetsWarnings` has it, unchanged.
2. **D6:** `logging.Options` gains the terminal's width (0 for "not a terminal"); `Open` uses the terminal form when it
   is set. `internal/cli/main.go` passes `caps.Err.Width` when `caps.Err.TTY`, and `$HOME` for the `~`.
3. **D7:** `TestProjectSkillDirsOutsideRepository` in `prompt/trust_test.go`: a tree with no `.git`, `home` and an
   ancestor each holding `.agents/skills`, the working directory below `home`; the result holds the ancestor's and
   not `home`'s. Seen failing on a scratch copy whose `ProjectSkillDirs` lacks the `$HOME` check.
4. **The capture again:** on the Linux machine, P1's capture with `HOME` below a directory holding `.agents/skills`, so
   the record is logged: it reads as D6's `Warning:` line, wrapped within the width; `help_check` passes from the
   first line after `/help`, with no `--from-heading`.
5. **Records:** this amendment's execution record; 0004-PLAN Phase 2 step 4's dated note; `docs/README.md`'s rows.

**Verification.** Each new test seen failing first, its failure quoted; the capture of step 4; the pre-add check on
the changed `.go` files; `go test -count=1 ./...`, `CGO_ENABLED=1 go test -race -count=1 ./...` and `make preflight`
on macOS, the Linux machine and the Windows laptop, the remote two on a copy of the tree; `check-records` and
`markdownlint`.

**Rollout and rollback.** One commit, the owner's. Reverting it restores slog's text on a terminal; the log file and
`gobble acp` are untouched either way.
