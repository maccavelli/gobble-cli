---
status: proposed
date: 2026-10-20
associated-madr: "0024-MADR-cli-mode-polish.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement polishing CLI mode, beginning with `/help`

Associated MADR: [0024-MADR-cli-mode-polish.md](0024-MADR-cli-mode-polish.md)

*Proposed. It starts after the owner approves it, on `main` at or after `9188f63`. The MADR's Q1–Q3 are answered
(2026-10-20): entries indented under headings, D3's descriptions as proposed, `/help <command>` a later amendment.*

## Goal

`/help` in the line session prints an aligned, wrapped list with informative descriptions, and every local command
prints with the terminal out of raw mode (MADR D1–D4).

## Scope

- **In:** `internal/cli/line.go` (the loop), `internal/cli/slash.go` (the local commands, `/help`'s layout and
  descriptions), a new `internal/cli/helplist.go` for the layout if `slash.go` grows past it; `acpserver/commands.go`
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
