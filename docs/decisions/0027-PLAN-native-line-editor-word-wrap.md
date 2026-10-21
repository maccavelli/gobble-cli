---
status: in-progress
date: 2026-10-21
associated-madr: "0027-MADR-native-line-editor-word-wrap.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement gobble's own line editor, wrapping whole words while the user types

Associated MADR: [0027-MADR-native-line-editor-word-wrap.md](0027-MADR-native-line-editor-word-wrap.md)

*Proposed. Approved 2026-10-21 ("Both, P4 then 0027"), Q1 and Q2 answered A. It starts after the owner approves it and answers Q1 and Q2, on `main` after 0024-PLAN P4 (the width), so
the width the editor reads is already the terminal's. Each phase is one commit; each starts by measuring again what it
relies on, and a fact that differs is a deviation, stopped and asked.*

## Goal

In the line session, the prompt wraps at the terminal's width, to 120 columns, never inside a word or before its
punctuation, while it is typed and in the scrollback, with every editing key, history, paste and completion as they are
today (MADR D1–D7).

## Scope

- **In:** `internal/cli/editor` (the buffer, the layout, the renderer, the keys, and the read loop moved onto them);
  its tests and a screen model for them; `internal/cli/line.go` where it builds the editor; 0008-MADR D6's dated note;
  `docs/architecture.md`; `docs/README.md`.
- **Out:** a multi-line buffer (Q2 B), unless the owner takes it; the agent's replies; the TUI mode.

## Implementation Steps

### P1 — the layout (D2)

1. **Tests first:** a table of prompt, text, width and cursor, with the rows and the cursor's row and column wanted:
   breaks only at spaces; `comma,` and `end.` kept whole; a word wider than a row cut; continuation rows indented by
   the prompt's width; wide characters as two columns; the width capped at 120 and 80 when unknown; the cursor at a
   row's end, at the text's end, inside a cut word; an empty text.
2. **`layout(prompt, text, cursor, width)`** in `internal/cli/editor`: rows and the cursor's place. Unused by the
   editor yet: no visible change.

### P2 — the screen model and the renderer (D3)

1. **A screen model** in the test package: a grid that takes `\r`, line feed, cursor up/down/forward/back, erase to the
   end of the screen and of the line, and printable runes, and answers "what is on screen, where is the cursor". Its
   own tests first.
2. **Tests first:** drawing a layout, then another (longer, shorter, fewer rows), leaves exactly the new rows and the
   cursor in its place; a line taller than the screen keeps the cursor's row on screen.
3. **The renderer:** from the rows drawn last time to the new ones, by relative moves.

### P3 — the keys (D4, D5)

1. **Measure first:** the bytes each key sends on macOS Terminal, a Linux terminal and Windows Terminal and Console Host
   (through the key reader), as 0008-MADR measurement 10 did for paste. A key that differs from `x/term`'s table is a
   deviation.
2. **Tests first, a parity table:** for each key `x/term` handles, the buffer and cursor after it, as `x/term` gives
   them today on a one-row line, and on a wrapped line; ↑/↓ by row and history (Q1); Ctrl+A/E to the line's ends.
3. **The key handling** on the buffer, using the key reader as it is.

### P4 — the editor on them (D1, D5)

1. **Tests first:** the editor's existing tests pass unchanged on the new editor (paste coalescing, Ctrl+C twice,
   Ctrl+D, `/exit`, history, Tab, the header); new: a long prompt typed at widths 60, 100 and 200 ends on a screen with
   no word split and rows of at most 120 columns; a resize between keys redraws at the new width.
2. **The read loop** on the buffer, the layout and the renderer; `x/term`'s `Terminal` no longer used; `x/term` kept for
   raw mode and the size.
3. **In a pseudo-terminal,** on macOS and the Linux machine, at 60, 100 and 200 columns: typing, editing in the
   middle, history recall, a paste, Tab, Ctrl+C; the final screens rendered from the bytes. Windows: the owner's look
   in Windows Terminal and Console Host. A screen that differs from the tests is a deviation.
4. **Records:** 0008-MADR D6's dated note; `docs/architecture.md`'s line-session lines; `docs/README.md`.

## Verification

- Every new test seen failing first, on the unchanged code or on a scratch copy, its failure quoted in the record.
- Each phase: the pre-add check on its changed `.go` files; `go test -count=1 ./...`, `CGO_ENABLED=1 go test -race
  -count=1 ./...` and `make preflight` on macOS, the Linux machine and the Windows laptop, the remote two on a copy of
  the tree; `check-records` and `markdownlint`.
- P4's captures, and the owner's look on Windows, before the phase is complete.

## Rollout and Rollback

One commit per phase. P1 to P3 change nothing a user sees. P4 switches the editor; reverting it returns `x/term`'s
`Terminal` with 0024's width. Nothing is stored.

## Execution record

### P1 execution record (2026-10-21)

**Run** on `main` at `4e21173`, on the owner's word "Both, P4 then 0027", with no deviation.

1. **Tests first,** seen failing against stubs of `lineWidth` (0) and `layout` (no rows): `TestLineWidth` and all
   fourteen cases of `TestLayout` (`internal/cli/editor/layout_test.go`). Three expected widths in the first draft of
   the table were wrong for what their cases meant (a last row exactly full, a first row that fit more): corrected
   before any code, at 13, 11 and 19 columns.
2. **The layout** (`internal/cli/editor/layout.go`): `lineWidth` (the terminal's width, to 120, 80 when unknown);
   `layout(prompt, text, cursor, columns)`, rows breaking at the last space a row's room reaches, the space taken by the
   break, a word wider than a row cut, continuation rows indented by the prompt's width, wide characters as two columns
   (`golang.org/x/text/width`), the cursor placed by the same rows and moved to the next row's start when it would
   stand past the last column. The editor's `size` (0024 P4) uses `lineWidth`, its own rule removed; `TestWidth`
   holds it. Not used by the editor yet: nothing a user sees changes.
3. **Negatives,** on a scratch copy, one mutation at a time, the tree untouched: rows breaking by character
   (`["> the quick brow" "  n fox jumps"]`); the break keeping its space; wide characters as one column; no cap at 120
   (`lineWidth(200) = 200`); no 80 when unknown; the cursor past the last column; continuation rows not indented; the
   editor's width ignoring the cap (`the first row held 152 columns, want 120`). Eight mutations, eight failures (one
   redone so it built); the restored copy passed.

**Found by the gates and fixed in the phase:** gocritic's `importShadow`, `layout`'s parameter `width` shadowing the
`width` package: it is `columns`. Seen on macOS, the Linux machine and the Windows laptop, whose first runs were on the
tree before the rename; each ran again after it.

**Gates:** the pre-add check on the three changed `.go` files, `3 file(s) clean (gofmt, golint, govulncheck)`, after the
rename; macOS `make preflight`, `preflight passed`, and the tests and the race detector, 35 packages ok each; the Linux
machine and the Windows laptop, Go 1.27.3, on a copy of the working tree, Windows with `TMP` and `TEMP` outside the
user's home: the tests and the race detector, 35 packages ok on each, then, on the renamed tree, `make preflight`
through `apidiff` (golangci-lint `0 issues.`, govulncheck "No vulnerabilities found."), stopping at
`verify-build-metadata` on a copy; `make check-records markdownlint`, "0 issues in 0 files".

### P2 execution record (2026-10-21)

**Run** on `main` at `b619163`, with no deviation.

1. **The screen model** (`internal/cli/editor/screen_test.go`, test code only): a grid with a cursor that takes
   printable runes (a wide one in two cells), `\r`, line feed (scrolling at the bottom), cursor up, down, forward and
   back, and erase to the end of the screen and of the line; a rune in the last column leaves a wrap pending, as
   terminals do. `TestScreenModel`'s twelve cases seen failing against a stub (all twelve), then passing. One expected
   value in the first draft was wrong by hand-tracing (`b    Y` for `b   Y`), corrected before the model was written.
2. **The renderer** (`internal/cli/editor/draw.go`): `draw` moves up by the cursor's row in the last drawing, `\r`,
   erases to the end of the screen, writes the rows joined by `\r\n`, then moves to the cursor's row, `\r`, and
   forward to its column; a line taller than the screen is drawn from a window of rows that holds the cursor. `finish`
   leaves the line drawn and moves below it. `TestDrawRedraws`, `TestDrawFullRows`, `TestDrawTallLine` and
   `TestDrawFinish` seen failing against a stub, then passing. `TestDrawTallLine` was strengthened before the
   negatives, with the cursor at a tall line's start: without it, a renderer with no window passed, the last rows
   scrolling into the same place.
3. **Negatives,** on a scratch copy, one mutation at a time: the redraw from the cursor's row, not the line's top; no
   erase; the column moved to without a row start (`cursor (2, 8)` for `(2, 4)`); no window on a tall line
   (`["  r4" … "  r7"]` for `["> r0" … "  r3"]`); `finish` leaving the cursor on the line; and in the screen model, no
   pending wrap, no scrolling, an erase that leaves the line. Eight mutations, eight failures; the restored copy
   passed. Not used by the editor yet: nothing a user sees changes.

**Gates:** the pre-add check on the three new `.go` files, `3 file(s) clean (gofmt, golint, govulncheck)`; macOS `make
preflight`, `preflight passed`, and the tests and the race detector, 35 packages ok each; the Linux machine and the
Windows laptop, Go 1.27.3, on a copy of the working tree, all 754 files, Windows with `TMP` and `TEMP` outside the user's
home: the tests and the race detector, 35 packages ok on each, `make preflight` through `apidiff` (golangci-lint `0
issues.`, govulncheck "No vulnerabilities found."), stopping at `verify-build-metadata` on a copy; `make check-records
markdownlint`, "0 issues in 0 files".
