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

### P3 execution record (2026-10-21)

**Run** on `main` at `d6e5bb7`.

1. **The measurement,** with no keyboard: scratch probes outside the tree, each reading in `x/term`'s raw mode as the
   editor does (on Windows, with virtual-terminal input on), 31 keys each.
   - **Windows** (the Windows laptop, Windows 11 build 26200): each key written into the probe's own console as a
     key-down and a key-up event (`WriteConsoleInputW`), as Windows Terminal hands keys to its console host, and read
     back. Hosts: the ssh session's console; Console Host (`conhost.exe`, headless); Windows Terminal 1.24's console
     host (`OpenConsole.exe`, headless); and, inside each of the last two, the probe started from Windows PowerShell
     5.1 and from PowerShell 7.6. All seven alike.
   - **Linux** (the Linux machine, Ubuntu 26.04): xterm 407 and xfce4-terminal 1.1.5 (VTE 0.84, the library of
     GNOME's terminals), on a virtual display, each key pressed with `xdotool` and followed by `#` as a separator.
     xfce4-terminal ran with its menu bar hidden: shown, its Alt+F opens the File menu.
   - **macOS** (this repository's development machine): WezTerm and Terminal, each key pressed by System Events after
     checking the probe's window was in front; kitty 0.49.2 and iTerm2 3.6.11, each key posted to that application's
     process only. Mission Control's "Move a Space" shortcuts are on there, so Ctrl+←/→ were not pressed in WezTerm and
     Terminal: typed, they never reach a terminal. Posted to kitty's and iTerm2's process, they bypassed it.

   | Keys | What arrives | `x/term` |
   | --- | --- | --- |
   | ←, →, ↑, ↓, Delete, Backspace, Alt+←/→, Ctrl+A/E/B/F/K/U/W/T/L/P/N/D/C, Tab, Enter | as `x/term` reads them, in every terminal | read |
   | Home / End | `ESC[H` / `ESC[F`; in Terminal, nothing (it scrolls its window) | read |
   | Option+←/→ in Terminal | `ESC b` / `ESC f`, where every other terminal sends `ESC[1;3D` / `ESC[1;3C` | **not read** |
   | Ctrl+←/→ | `ESC[1;5D` / `ESC[1;5C` | not read |
   | Alt+B / Alt+F | `ESC b` / `ESC f`; xterm `â` / `æ` (its eight-bit input); with Option typing symbols (Terminal, kitty, iTerm2 as installed), `∫` / `ƒ` | not read; xterm's and the Mac's characters are typed |
   | Shift+Tab | `ESC[Z` | not read |
   | Shift+Enter | `\r`, as Enter | as Enter |

**Deviation 1, 2026-10-21, decided by the owner** ("Also add Ctrl+←/→"): in macOS Terminal, Option+←/→ send `ESC b` /
`ESC f`, not the `ESC[1;3D` / `ESC[1;3C` `x/term` reads for Alt+←/→, so they do nothing in gobble today; and Terminal's
Home and End send nothing at all.

- **Chosen:** `ESC b` / `ESC f` and Ctrl+←/→ (`ESC[1;5D` / `ESC[1;5C`) move by word, as Alt+←/→ do. MADR D4 is
  amended. Step 2's table gains them, tested against Alt+←/→'s result, since `x/term` has no result of its own for them.
- **Not chosen:** `ESC b` / `ESC f` alone; no new keys, Terminal's Option+←/→ recorded as doing nothing.
- **Home and End in Terminal:** recorded, not fixed: gobble cannot act on a key it never receives; Ctrl+A/E reach the
  line's ends there.
- **Shift+Enter** sends what Enter sends in every terminal measured, so it cannot mean anything of its own.
- **Files added to P3's scope:** none.

2. **Tests first, the parity table** (`internal/cli/editor/buffer_test.go`). Its reference is `x/term`'s `Terminal`
   itself, run on the same bytes over an in-memory connection, then asked for its line and cursor by a key that reaches
   its completion callback: what `x/term` gives today is measured, not written down. `TestKeysMatchXterm`: 55 cases
   from a typed state (the moves, Backspace and Ctrl+H, Home, End, Ctrl+A/B/E/F, Alt+←/→, Delete, Ctrl+D, K, U, W, T,
   L, Tab, unknown sequences, wide and accented text, Enter, CR LF, LF, Ctrl+D and Ctrl+C ending the input, pastes,
   control keys inside a paste, the 4096-rune limit, history), each on one row and, but for the nine that use ↑ or ↓,
   on rows of 12 columns. `TestKeysSplitAcrossReads` (one byte at a time); `TestWordKeys` (`ESC b`, `ESC f` and
   Ctrl+←/→ against Alt+←/→'s result, deviation 1); `TestRowKeys` (19 cases: ↑/↓ and Ctrl+P/N by row and then history,
   the column kept or the nearest before it, wide characters, a full row, a recalled entry edited by row);
   `TestPlacesMatchLayout` (the cursor's place at every rune index is the layout's, on 13 texts); `TestClearScreenKey`;
   `TestCompletionCallback`. Against a stub, all seven failed: 54 of 54 parity cases on one row and 45 of 45 on rows
   of 12. One case was added after the stub's run and before the negatives, a paste of three lines, the only shape
   whose middle line both starts and ends inside the paste.

   Two things the reference showed about `x/term`, recorded, not copied: after Ctrl+D or Ctrl+C ends the input, it
   keeps the read that held the key and ends every later `ReadLine` the same way, so its line there cannot be probed
   (those cases compare the lines and the end of input); and inside a paste it would put an unknown escape sequence in
   the line as the rune `U+D800`, where the buffer drops it.
3. **The key handling** (`internal/cli/editor/buffer.go`): `decodeKey` (the control bytes and escape sequences
   `x/term` reads, plus deviation 1's four; a key's start waits for its end; inside a paste only the end marker is a
   key); `lineBuffer`, the line and cursor in runes, whose `feed` applies keys until a line ends, the input ends or
   Ctrl+L asks for a clear screen, with `x/term`'s paste marking and CR LF; ↑/↓ by row through `placesOf`, the cursor's
   place at each rune index in one pass over the layout's rows, then history; the completion callback as `x/term` calls
   it. The key reader is unchanged. Not used by the editor yet: nothing a user sees changes.
4. **Negatives,** on a scratch copy, one mutation at a time: `ESC b` not read; Ctrl+← not read (each `pos:14`, want
   `pos:9`); ↑ always recalling (`pos 25`, want `15`); ↓ always recalling; the row's first place, not the nearest; no
   next-row place past a full row (`place (0, 13), layout (1, 2)`); a word back stopping at the spaces; a line begun in a
   paste not marked pasted (`pasted:[true false false]`, want `[true true false]`); CR LF as two Enters; no line limit;
   Ctrl+D on an empty line going on; a key's start not waited for (`héllo[D[1;3D…` typed as text); Ctrl+T at the end
   swapping past it (a panic, index out of range); ↓ not bringing back the user's own line; the callback given a rune
   offset (`2`, want `3`); Ctrl+L not reported. Sixteen mutations, sixteen failures; the restored copy passed.

**Found by the gates and fixed in the phase:** `gofmt` drift in `buffer_test.go` (the alignment of `fixedHistory`'s
three one-line methods), in the pre-add check and `make preflight` on macOS; the tests and the race detector had passed
on all three hosts. `gofmt -w` on the file, and every gate run again.

**Gates,** after the fix: the pre-add check on the two new `.go` files, `2 file(s) clean (gofmt, golint, govulncheck)`;
macOS `make preflight`, `preflight passed`, and the tests and the race detector, 35 packages ok each; the Linux machine
and the Windows laptop, Go 1.27.3, on a copy of the working tree, all 756 files, Windows with `TMP` and `TEMP` outside
the user's home: the tests and the race detector, 35 packages ok on each, `make preflight` through `apidiff`
(golangci-lint `0 issues.`, govulncheck "No vulnerabilities found."), stopping at `verify-build-metadata` on a copy;
`make check-records markdownlint`, "0 issues in 0 files".
