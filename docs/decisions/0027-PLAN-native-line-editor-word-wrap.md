---
status: complete
date: 2026-10-22
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
  `docs/architecture.md`; `docs/README.md`. P5 (proposed 2026-10-21): `internal/cli/editor/reader.go` and the
  goodbye in `internal/cli/line.go`.
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
   Ctrl+D, `/exit`, history, Tab, the header) *(deviation 2: `TestHeader` and `TestWidth` checked on the screen)*; new: a long prompt typed at widths 60, 100 and 200 ends on a screen with
   no word split and rows of at most 120 columns; a resize between keys redraws at the new width.
2. **The read loop** on the buffer, the layout and the renderer; `x/term`'s `Terminal` no longer used; `x/term` kept for
   raw mode and the size.
3. **In a pseudo-terminal,** on macOS and the Linux machine, at 60, 100 and 200 columns: typing, editing in the
   middle, history recall, a paste, Tab, Ctrl+C; the final screens rendered from the bytes. Windows: the owner's look
   in Windows Terminal and Console Host. A screen that differs from the tests is a deviation. *(Deviation 4: the
   look found flicker; the drawing no longer erases first, and the look moves to P5.)*
4. **Records:** 0008-MADR D6's dated note; `docs/architecture.md`'s line-session lines; `docs/README.md`.

### P5 — the first line on Windows, and the goodbye (D3, D5 as amended; approved 2026-10-21 by the owner's answer "Fix, P4, P5, one look")

Added after P4's captures found two defects that predate the new editor (P4's "Found, not fixed"). P5 starts after
P4's commit. Files: `internal/cli/editor/reader.go`, `editor.go` and their tests; `internal/cli/line.go` and its
tests; `doc.go` where it describes the reader.

1. **Measure first, on the Windows laptop:** P4's headless driver under Console Host and Windows Terminal's console
   host, ten runs each, keys sent at once after start. Record each run's first read (a scratch build that logs reads,
   as in P4). This is the baseline the fix is measured against, on the P4 build.
2. **Tests first,** each seen failing on the code as it is:
   - `editor.Reader`: `Start` reads nothing until asked. Each `ReadContext` that finds nothing buffered starts one
     read of stdin, and none after it. The test uses a stub stdin that counts reads and blocks until released.
   - The line session: no read of stdin is waiting while the session is out of raw mode for a turn. The test uses a
     stub stdin and the stub raw mode, a prompt sent, and the scripted agent's turn held until it checks. Today the
     reader starts its next read as soon as it delivers the Enter.
   - The editor, on the screen: Ctrl+D on an empty line, and a second Ctrl+C, leave the cursor at column 0 of the
     row below the line, its rows unchanged.
   - The line session, on the screen of everything it wrote: after a second Ctrl+C, the goodbye is on a row of its
     own below the last prompt. Raw mode and bracketed paste are already off when it is written.
3. **The fix:** `Reader` starts a read only for a `ReadContext` that asks; its goroutine waits for a request or the
   context's end. `readLine` finishes the drawing before it returns the end of input. The line session restores the
   terminal and turns bracketed paste off before the goodbye; the deferred restores then find nothing to undo.
4. **Measure again,** as in step 1, on the fixed build: no first read holding more than one line's keys in twenty
   runs. Then P4's captures again on macOS, the Linux machine and both Windows hosts: 24 of 24 each, and the last
   screen showing the goodbye on its own row. Then the owner's look in Windows Terminal and Console Host, moved here
   from P4 by deviation 4: no flicker, the first typed keys shown, the goodbye on its own row.
5. **Negatives,** on a scratch copy, one mutation at a time: the reader reading ahead again; one ask starting two
   reads; `readLine` not finishing on the end of input; the goodbye written before the terminal is restored.
6. **Records:** this plan's execution record; `docs/README.md`.

## Verification

- Every new test seen failing first, on the unchanged code or on a scratch copy, its failure quoted in the record.
- Each phase: the pre-add check on its changed `.go` files; `go test -count=1 ./...`, `CGO_ENABLED=1 go test -race
  -count=1 ./...` and `make preflight` on macOS, the Linux machine and the Windows laptop, the remote two on a copy of
  the tree; `check-records` and `markdownlint`.
- P4's captures, and the owner's look on Windows, before the phase is complete.
- P5's measurements on the Windows laptop, before and after the fix, and the captures again.

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

### P4 execution record (2026-10-21)

**Run** on `main` at `1cc1616`.

**Deviation 2, 2026-10-21, decided by the owner** ("Check the screen"): on the new read loop, two of the editor's
existing tests failed, each reading `x/term`'s output bytes rather than the screen. `TestHeader`
(`internal/cli/editor/editor_test.go:160`) counts `proj (main)\r\n> `, and the editor draws its first prompt as
`\r\x1b[J> ` ("the header before 0 prompts, read 3 times, want 3 and 3"); `TestWidth` (`:213`) counts the bytes from
`> ` to the first `\r\n`, which now hold the redraws' escape sequences ("the first row held 111 columns, want 100";
131 for 120, 71 for 60, 91 for 80). Not pre-existing: both passed on `x/term`'s `Terminal`. The behaviour each states
holds: the header is written once above each new prompt, and rows break at the width.

- **Chosen:** both tests keep their claims, checked on the screen model as the new tests are: the header's row once
  above each new prompt, never above a continuation prompt, escapes and line breaks dropped; the first row of a
  150-character word holding 100, 120, 60, 80 and 80 columns. Each rewritten check seen failing on a broken copy.
- **Not chosen:** drawing `x/term`-like bytes so the tests pass as written (no erase before a first prompt, a key typed
  at the line's end appended instead of a redraw): a second drawing path, amending MADR D3.
- **Files added to P4's scope:** none; `internal/cli/editor/editor_test.go` was in it.

**Deviation 3, 2026-10-21, decided by the owner** ("Share the screen model"): three of the line session's tests in
`internal/cli` read `x/term`'s output bytes too. `TestPromptHeader` (`internal/cli/line_test.go:274`) waits for
`proj (main)\r\n> ` twice and times out ("condition not met within 5 s"); `TestLineAmbient`
(`internal/cli/ambient_test.go:214`) ends `/help`'s output at the next `\n> `, finds none, and reads on into the
commands typed after it ("/help shows a hidden command"); `TestPromptWidth` passes only because its 150 characters
arrive in one read. Not pre-existing; the behaviour each states holds. The screen model was in the editor's test
package, out of their reach.

- **Chosen:** the screen model moves to a test package of its own, `internal/cli/term/termtest`, with its tests; the
  editor's tests and these three check the screen with it. 0027-MADR D6 and 0004-MADR's package map are amended.
- **Not chosen:** the three tests finding the prompt by the new editor's drawing (`\x1b[J> `), still reading bytes.
- **Files added to P4's scope:** `internal/cli/term/termtest/screen.go` and `screen_test.go` (the model and its
  tests, moved from `internal/cli/editor/screen_test.go`, which goes); `internal/cli/line_test.go`;
  `internal/cli/ambient_test.go`; `docs/decisions/0004-MADR-go-module-architecture.md`.

1. **Tests first.** The screen model's three new cases (a private mode shows nothing, erasing the whole screen, home)
   seen failing on the model as it was (`"a2004hb200"`), then passing. `Options.Height` added, unused by the old editor,
   so the new tests ran against `x/term`'s `Terminal`: `TestTypingKeepsWordsWhole`, `TestResizeBetweenKeys`,
   `TestEditingMidLineOnScreen` and `TestTallLineShowsTheCursorRow` (`internal/cli/editor/editor_screen_test.go`)
   failed, the owner's report on the model's screen (`"> The prompt is being cut … 80 chars, in" "cluding in the middle
   …"`, `"…My termi" "nal is much wider…"`, `"…a traili" "ng comma…"`); `TestClearScreenOnScreen` passed, Ctrl+L
   being the same in both, and is shown able to fail by step 6. Their first draft ended the line's rows at the first
   empty row; a run's last read draws the next prompt below a sent line, so they end at it, and check the cursor there.
2. **The read loop** (`internal/cli/editor/editor.go`): `readLine` feeds each read's bytes to the buffer and draws the
   line again after them, at the width and height read then (`Options.Width`, `Options.Height`; `line.go` gives the
   terminal's height); Enter draws it with the cursor at its end and moves below it; Ctrl+L clears the screen and draws
   it at the top; bracketed paste is turned on by the first `ReadPrompt` (at `New` before) and off by `Restore`, each
   write's error returned (`Restore` now returns one; the line session warns of it, the Console Host suite joins it). `x/term` is left for raw mode
   (`editor.Raw`) and the line session's size. The package's comments no longer describe `x/term`'s `Terminal`.
3. **Deviations 2 and 3** carried out: `TestHeader` and `TestWidth` check the screen; the model moved to
   `internal/cli/term/termtest` with its tests (`TestScreen`, two cases added: a sequence with parameters shows nothing;
   cooked, a line feed is a new line); `TestPromptHeader`, `TestLineAmbient` and `TestPromptWidth` check the screen
   of everything the session wrote (`screenOf`, cooked). Every editor test and every line-session test passes.
4. **Pseudo-terminal captures** (scratch drivers outside the tree): `gobble` built from the working tree, in a pty of
   60, 100 and 200 columns by 40 rows, on macOS and on the Linux machine, with a scratch home holding two history
   entries (`/help` and the owner's report), a dummy key and a dead proxy; Enter only on `/help`, a local command, so
   no prompt reached a model. Stages: the report typed in chunks of 16 bytes; edited in the middle (Ctrl+A, Alt+→
   three times, `really `, Ctrl+E, Tab); cleared by Ctrl+C; recalled by ↑, then ↑ again; `/help`; a two-line paste;
   Ctrl+C clearing it; Ctrl+C twice to leave. Each stage's bytes drawn on `termtest`'s screen. A scratch check over the
   typed, edited and recalled stages: every row at most the width to 120 columns, continuation rows indented, the rows
   joined by spaces equal to the text. 24 of 24 on macOS and the Linux machine (widest rows 60, 100 and 119 of 60,
   100 and 120). The check seen failing on `x/term`'s capture at 100 columns (one unindented row; a crash on ↑, which
   recalled history there). The second ↑ moves up a row, keeping the column (52), in every capture. Leaving: gobble
   exited on the second Ctrl+C each time.

   **On Windows, beyond the plan:** the same stages, headless on the Windows laptop, under Console Host
   (`conhost.exe --headless`) and Windows Terminal 1.24's console host (`OpenConsole.exe --headless`), keys sent as the
   host's input, the host's own rendering drawn by a terminal emulator library (`pyte`) in a scratch environment:
   24 of 24 alike. This does not replace the owner's look (step 3), which is still to come.

   **Bytes per key,** which MADR's Consequences said would be measured here: the report's 337 characters, typed in
   chunks of 16 bytes, wrote 4425 to 4548 bytes on the new editor and 343 on `x/term`'s, each read drawing the whole line again.
5. **Records:** 0008-MADR D6's dated note; `docs/architecture.md`'s `internal/cli/editor` row and a
   `internal/cli/term/termtest` row; `docs/README.md`'s 0004, 0008 and 0027 rows.
6. **Negatives,** on a scratch copy, one mutation at a time: the width read only at a line's first drawing (`row 0 is
   74 columns, more than 60`); no cap at 120 (`199 columns`); no height given (the cursor's row off the screen); Enter
   not moving below the line; Ctrl+L not clearing (`"old output" "more"` left); the header with no line end; no
   header (`condition not met within 5 s`); no drawing after a read; bracketed paste not turned on; the line session
   giving no width (a first row of 80 columns, not 100); `/help` showing `/humor` (`/help shows a hidden command`); and in the model,
   a line feed not cooked, parameters stopping at `;` (`"a31mb"`), a private mode not known (`"a2004hb200"`), no
   erase of the whole screen. Fifteen mutations, fifteen failures; the restored copy passed. Two first choices changed
   nothing on the screen and were replaced: a header ended by `\n` alone (the next drawing starts with `\r`) and a
   private mode drawn as an ordinary sequence (`h` and `l` move nothing); one did not build and was redone; one went
   to `acpserver`'s command list, which the line session's scripted agent does not use, and was moved to the line
   session's own.

**Found by the gates and fixed in the phase:** golangci-lint's `errcheck` on the two discarded writes of the paste
mode (`editor.go:91`, `:98`), on all three hosts, and gocritic's `importShadow`, `termtest.New`'s parameter `width`
shadowing the `width` package, on macOS: the writes' errors are returned (step 2), the parameters are `columns, rows`.
The negatives ran again on the fixed code, the paste mutation moved to `ReadPrompt`: fifteen failures; every gate ran
again.

**Found, not fixed (outside this plan):**

- **Windows, the first line of a session.** The line session starts the reader that owns stdin before it puts the
  console in raw mode (`internal/cli/line.go`: `r.Start`, then `ls.raw()`). A read Console Host starts in line mode
  stays a line read: it returns at the first Enter, with every key before it in one piece, and the console, not
  gobble, holds the line meanwhile. Seen in the headless captures under Windows Terminal's console host, by a scratch
  build that logged each read: the first read returned 379 bytes at the Enter of `/help` in 1 of 3 runs of the new
  editor, and in 2 of 6 runs of `x/term`'s (built from `1cc1616`), so it predates P4. Raised with the owner.
- **The goodbye** after a second Ctrl+C is written on the row of the prompt it ended (`… Goodbye. …`), on the new
  editor and on `x/term`'s alike.
- **Narrowing a terminal that reflows its rows** (most do) may leave the last drawing's rows misplaced for one
  redraw: the editor moves up by the rows it drew, which the terminal has since reflowed. The screen model does not
  reflow; not measured. *(2026-10-22: measured and fixed by
  [0030-PLAN-resize-and-performance.md](0030-PLAN-resize-and-performance.md): P1 measured three ways terminals
  narrow, P2 redraws the prompt by the one the cursor's place shows, and P3 reprints the conversation.)*

**Gates,** after the fixes: the pre-add check on the fourteen changed `.go` files, `14 file(s) clean (gofmt, golint,
govulncheck)`; macOS `make preflight`, `preflight passed`, and the tests and the race detector, 36 packages ok each; the
Linux machine and the Windows laptop, Go 1.27.3, on a copy of the working tree, Windows with `TMP` and `TEMP` outside the
user's home: the tests and the race detector, 36 packages ok on each, `make preflight` through `apidiff` (golangci-lint
`0 issues.`, govulncheck "No vulnerabilities found."), stopping at `verify-build-metadata` on a copy; `make
check-records markdownlint`, "0 issues in 0 files"; and on the Windows laptop `make probe-conhost`, the live Console
Host suite, `--- PASS: TestConhost`.

**The owner's look, 2026-10-21:** in Windows Terminal with PowerShell 7, "janky … not seeing the first typed chars",
then "Flicker." The first part is the pre-existing first read above. Reproduced headless with an interactive pwsh 7.6.6
in Windows Terminal's console host, gobble started from it and keys typed one at a time. In 4 of 11 runs of P4's
build, the first read returned everything typed before the first Enter in one chunk (`"hello
there\x7f\x7f\x7f\x1b[D\x1b[DX\x03\x1b[A\x03/help\r"`). A scratch build reading only when asked (P5) showed 0 in 10.
The flicker is P4's own: the console host passes gobble's bytes through, and each key's drawing goes to column 0,
erases to the end of the screen and writes every row again (`\r\x1b[J> hello\r\x1b[7C`). A terminal that paints
between the erase and the rewrite shows the line blank, and the cursor jumps to column 0 and back.

**Deviation 4, 2026-10-21, decided by the owner** ("Draw without blanking"; order "Fix, P4, P5, one look"). The
renderer no longer erases before it draws. It rewrites only rows that changed, each from its first changed column,
then erases what is left of a row that got shorter and rows the line no longer has. The cursor is hidden while it
moves, and each drawing that writes is wrapped in synchronized output (DEC mode 2026), which terminals that lack it
ignore. 0027-MADR D3 is amended. Added to the phase: `termtest.Screen` tracks the cursor's visibility; a renderer
test feeds each drawing to the screen one sequence at a time. At every step, each row must keep what its old and new
text share, and the cursor must be hidden away from its old and new places. The captures, the bytes per key and the
gates run again. The owner's look moves to P5: P4 is committed without it, and P5's look on Windows covers both
phases.

**Deviation 4 carried out:**

1. **Tests first,** each seen failing on the renderer as it was. `TestScreenHidesTheCursor` (`after "\x1b[?25l",
   hidden false`), with `termtest.Screen.Hidden` added but not yet set. `TestDrawNeverBlanks` takes the line through
   twelve drawings: typing, deleting, a cursor move, a rewrap to two rows and back. It failed with `after "\r", the
   cursor shows at [1 0], neither [1 2] nor [1 3]`. Its first draft compared rows with trailing spaces the screen
   trims, and was corrected. `TestDrawWritesOnlyWhatChanged` failed with `a key typed at the end: wrote 8 runes
   ("\r\x1b[J> hellox\r\x1b[8C"), want 1`.
2. **The renderer** (`internal/cli/editor/draw.go`) keeps the rows it drew and the cursor's row and column. A row
   that changed is written from its first changed column, a row that got shorter has its rest erased, and rows the
   line no longer has are erased. Unchanged rows are skipped. A drawing that writes anything is
   `CSI ?2026h CSI ?25l … CSI ?25h CSI ?2026l`. A first drawing, or one after Ctrl+L (`forget`), starts with `\r` and
   an erase to the end of the screen. The screen model tracks DECTCEM. Every editor and line-session test passes
   unchanged.
3. **Captures again,** as in step 4. macOS and the Linux machine: 24 of 24, the second ↑ keeping column 52. The
   Windows laptop, Console Host and Windows Terminal's console host: 24 of 24, ↑ keeping columns 57, 52 and 105.
   Five of eleven Windows runs lost the first line to the first read above (Console Host at 100 columns twice;
   Windows Terminal's host at 60, 100 and 200 once each). Each was run again until clean, and the lost runs were
   kept. **Bytes per key:** the report's 337 characters, typed in chunks of 16, now write 1227 to 1249 bytes, down
   from 4425 to 4548; `x/term`'s editor wrote 343.
4. **Negatives,** on a scratch copy: P4's fifteen mutations, the first rewritten for the renderer's new state, plus
   eight more. A drawing that erases first: `after "\x1b[J", row 0 shows "", losing ">"`. The cursor not hidden:
   `the cursor shows at [1 0]`. No synchronized update: `is not one synchronized update`. A changed row written
   whole: `wrote 8 runes`. A shorter row's rest not erased: `"> the quick"` left. Rows no longer in the line not
   erased: `"  brown fox" "  jumps"` left. A cursor-only move not drawn: the cursor at (1, 13), want (1, 5). The
   model not tracking the cursor: `hidden false`. Twenty-three mutations, twenty-three failures; the restored copy
   passed.
5. **Gates again:**
   - The pre-add check on the fifteen changed `.go` files: `15 file(s) clean (gofmt, golint, govulncheck)`.
   - macOS: `make preflight`, `preflight passed`; the tests and the race detector, 36 packages ok each.
   - The Linux machine and the Windows laptop, on a copy of the tree: the tests and the race detector, 36 packages ok
     on each. `make preflight` reached `apidiff` (golangci-lint `0 issues.`, govulncheck "No vulnerabilities
     found.") and stopped at `verify-build-metadata`, as the copies are not git checkouts.
   - `make check-records markdownlint`: "0 issues in 0 files".
   - The Windows laptop, `make probe-conhost`: `--- PASS: TestConhost`.

**P4 complete, 2026-10-21,** without the owner's look, which deviation 4 moved to P5.

**Found, not fixed (outside this plan), during deviation 4:** after the second Ctrl+C, gobble printed its goodbye but
sometimes had not exited 3 s later. This happened on the Linux machine in 2 of 6 runs of this build and 1 of 5 of
`x/term`'s (built from `1cc1616`), so it predates P4.

### P5 execution record (2026-10-21)

Approved by the owner's answer "Fix, P4, P5, one look", after P4's commit `31799e6`.

1. **Measured first** on the Windows laptop: P4's headless driver at 100 columns, keys sent once the prompt shows.
   The build was P4's commit with a scratch key log. The first read held the keys typed before the first Enter, 379
   bytes, in 6 of 10 runs under Console Host and 3 of 10 under Windows Terminal's console host; the other runs' first
   read was one typed chunk of 16 bytes.
2. **Tests first,** each seen failing on the code as it was:
   - `TestReaderReadsOnlyWhenAsked` (`internal/cli/editor/reader_test.go`, in a `synctest` bubble, stdin a stub
     that counts reads): `1 reads started before any was asked for`.
   - `TestLineSessionReadsOnlyWhenAsked` (`internal/cli/line_test.go`). The harness's stdin counts reads started out
     of raw mode. With no credential, the sign-in question is offered and declined, then a prompt, its turn, Ctrl+D:
     `3 reads of stdin started out of raw mode; want 1, the sign-in question's`, in three runs of three.
   - `TestEndOfInputLeavesTheLine` (`internal/cli/editor/editor_screen_test.go`), Ctrl+D and a second Ctrl+C:
     `screen [">" ...], cursor (0, 2)`. Its first draft pressed Ctrl+C on a typed line, which clears the line
     rather than arming the exit, so that case was ended by the input running out. It was corrected to two Ctrl+C
     on an empty line, and step 5 shows it failing for its own reason.
   - `TestGoodbyeOnItsOwnRow` (`internal/cli/line_test.go`), stdout and stderr in one log that notes raw mode at
     each write: `the goodbye was written in raw mode`, `the goodbye was written with bracketed paste on`, and the
     screen's row `"> Very good. I shall be here, tidying the context."`.
3. **The fix:**
   - `Reader` (`internal/cli/editor/reader.go`) waits for an ask, or its context's end, before each read of stdin.
     `ReadContext` asks once when nothing is buffered, and a read asked for and not served is served to the next
     call.
   - `readLine` finishes the drawing before it returns the end of input.
   - The line session (`internal/cli/line.go`) turns bracketed paste off and leaves raw mode before the goodbye, and
     skips its deferred restores.
   - `keys_test.go`'s session cancels its reader's context at the end, since the reader now waits for an ask
     rather than in a read that closing the pipe ends.
   - The package comment describes the reader.
4. **Measured again,** as in step 1, on the working tree: 0 of 10 under each host, every first read 16 bytes.
   P4's captures again: macOS and the Linux machine 24 of 24; the Windows laptop 24 of 24 under both hosts, no run
   losing the first line. A scratch check of each capture's last screen (`check_goodbye.py`): the second Ctrl+C
   leaves the bare prompt, and the goodbye is on the row below it, in all twelve.
5. **Negatives,** on a scratch copy, one mutation at a time:
   - The reader reading ahead: `1 reads started before any was asked for`; the line session's test hangs
     (`condition not met within 5 s`).
   - One ask starting two reads: `2 reads started after the one asked for was served, want 1`.
   - `readLine` not finishing on the end of input: both cases, `cursor (0, 2)` and `cursor (1, 2)`.
   - Bracketed paste still on at the goodbye: `the goodbye was written with bracketed paste on`.
   - Raw mode still on at the goodbye: `the goodbye was written in raw mode`.

   Five mutations, five failures; the restored copy passed.

**Found, not fixed (outside this plan):** the slow exit above is also seen on macOS. In five runs each at 100 columns,
gobble had not exited 3 s after its goodbye in 4 of 5 runs of P4's commit and 1 of 5 of P5's. P5 neither causes nor
fixes it.

**Found by the gates and fixed in the phase:** golangci-lint's `gocognit` on all three hosts, `cognitive complexity 56
of func (*lineSession).run is high (> 50)`, from the goodbye's restores. They moved to `lineSession.goodbye`; the two
goodbye mutations were pointed at it, and all five failed again, the restored copy passing.

**Gates, after the fix:**

- The pre-add check on the eight changed `.go` files: `8 file(s) clean (gofmt, golint, govulncheck)`.
- macOS: `make preflight`, `preflight passed`; the tests and the race detector, 36 packages ok each.
- The Linux machine and the Windows laptop, on a copy of the tree: the tests and the race detector, 36 packages ok on
  each. `make preflight` reached `apidiff` (golangci-lint `0 issues.`, govulncheck "No vulnerabilities found.") and
  stopped at `verify-build-metadata`, as the copies are not git checkouts.
- `make check-records markdownlint`: "0 issues in 0 files".
- The Windows laptop, `make probe-conhost`: `--- PASS: TestConhost`.

**The owner's look, 2026-10-22,** in Windows Terminal and Console Host (step 4, moved here from P4 by deviation 4):
"seems fine". The owner committed P5 as `6361bd9`, with 0028's proposed records.

**P5 complete, 2026-10-22; this plan complete.**
