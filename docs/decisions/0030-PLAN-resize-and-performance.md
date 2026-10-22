---
status: in-progress
date: 2026-10-22
associated-madr: "0030-MADR-resize-and-performance.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement redrawing on resize, and the performance budget

Associated MADR: [0030-MADR-resize-and-performance.md](0030-MADR-resize-and-performance.md)

## Goal

A resize redraws the line session at the new width: the live area at once, and the whole conversation once
resizing settles. No word is cut. Performance has a recorded baseline and a budget that tests enforce (MADR D1–D7).

## Scope

- **In:**
  - `internal/cli/term`: resize notice, `SIGWINCH` and the Windows size;
  - `internal/cli/editor`: woken by a resize; the live redraw;
  - `internal/cli` (`line.go`, `view.go`, `output.go`): the transcript and the reprint;
  - `internal/cli/render`: drawing pieces at a width;
  - the benchmarks and budget tests, and a `make perf` target;
  - their tests;
  - the dated notes in 0028-MADR and 0027-PLAN; `docs/architecture.md`; `docs/README.md`.
- **Out:**
  - print mode;
  - the TUI mode;
  - 0029's permission prompt, which follows and draws through the same transcript.

## Implementation Steps

### P1 — measure first (D1, D2, D5)

1. **Reflow on narrowing.** In each terminal (WezTerm, kitty, iTerm2, Terminal.app, xterm, VTE, Console Host and
   Windows Terminal's console host), draw a three-row prompt, narrow the window, and read where the cursor and the
   rows went. GUI terminals on macOS are driven only while the screen is unlocked and the terminal is in front; the
   rest are driven headless.
2. **Windows' resize.** Check whether a reader in virtual-terminal input mode sees any event when the console is
   resized, under Console Host and Windows Terminal's host. If none, D1's 100 ms poll stands.
3. **The baseline:** start-up, a key, a reply's rendering, and the costs a reprint of 10,000 rows from a 4 MB
   transcript would have, as benchmarks in a scratch copy, on all three hosts. The time each terminal takes to
   show a reprint of 1,000 and 10,000 rows, and whether `CSI 3 J` clears its scrollback.

The findings are recorded before P2. A terminal where the cursor cannot be found decides D2 there, as a deviation.

### P2 — the resize notice and the live redraw (D1, D2)

1. **Tests first:**
   - a resize notice wakes the editor's read without a key, and the prompt is drawn at the new width on
     `termtest`'s screen;
   - a reply still arriving is drawn at the new width;
   - the Windows poll notices a changed size, using a stub size.
2. **The notice** in `internal/cli/term`, **the editor's wake** in `editor.Reader`, and **the redraw.**

### P3 — the transcript and the reprint (D3, D4, D6)

1. **Tests first,** on `termtest`'s screen:
   - a session of a prompt, a reply with a list, code and a table, a message, a tool's output and `/help`, narrowed
     from 120 to 60 columns: after the settle, the screen and scrollback hold everything, every prose row whole at
     60, and the code and table as written;
   - several resizes within 150 ms reprint once;
   - the bound: past 4 MB, the oldest pieces go and the reprint says so;
   - a reprint writes at most the last 10,000 rows at the new width, after one line saying earlier output was not
     reprinted;
   - print mode and off a terminal: nothing kept.
2. **The transcript** at the line session's three writers (`Output`, the view, the editor), **the reprint** as one
   synchronized update, and **the bound.**

### P4 — the budget (D5)

1. **Benchmarks** in `internal/cli`, with the baselines from P1 recorded in this plan.
2. **Budget tests:**
   - the bytes a key's drawing writes;
   - the allocations of a 1000-row reprint;
   - each seen failing on a scratch copy made slower or larger.
3. **`make perf`:** the benchmarks against 3× their baselines, run on all three hosts.
4. *(Deviation 2.)* **The optimisation:** `render.Wrap` and `render.Stream` build words and lines in reused buffers,
   not by `+=` per character. Targets, against P1's baseline: at least 3× fewer allocations, and the last-10,000-row
   reprint at least 2× faster on each host. Every render test and every split test passes unchanged, and the budget
   tests hold the targets.

### P5 — on real terminals, and the records

1. **Captures** on macOS and the Linux machine (pty), and the Windows laptop (headless): a prompt typed, a stubbed
   reply, then narrowed from 120 to 60 columns mid-prompt and mid-reply. The final screen and scrollback are checked
   as in 0028-PLAN P4.
2. **The owner's look** in WezTerm and Windows Terminal.
3. **Records:** 0028-MADR's and 0027-PLAN's dated notes; `docs/architecture.md`; `docs/README.md`.

## Verification

- Every new test seen failing first, its failure quoted in the record.
- Negatives on a scratch copy, one mutation at a time:
  - the resize ignored;
  - no settle, a reprint per notice;
  - the transcript missing a kind of piece;
  - the scrollback not cleared;
  - the reprint not synchronized;
  - the bound not kept;
  - a transcript kept in print mode.
- Each phase:
  - the pre-add check;
  - the tests and the race detector, and `make preflight`, on macOS, the Linux machine and the Windows laptop;
  - `check-records` and `markdownlint`;
  - from P4, `make perf` on all three.

## Rollout and Rollback

One commit per phase. P1 changes nothing. P2 adds the live redraw; P3 the reprint, which clears the scrollback on a
resize; reverting P3 alone keeps the live redraw. P4 adds tests only. Nothing is stored.

## Execution record

### P1 execution record (2026-10-22)

Approved by the owner's "Proceed", 2026-10-22, with D3 and D4 as revised that day (4 MB, bytes only; the last 10,000
rows reprinted).

All measurements used scratch programs outside the tree.

- `reflowprobe` runs inside a terminal. It draws a header and a three-row prompt 80 columns wide, as gobble's editor
  does, and reads the cursor's place (DSR). After the resize it reads it again and redraws at the new width by one
  strategy:
  - **reflow**: up by the rows the old ones make when re-wrapped;
  - **truncate**: up by the rows as drawn;
  - **none**: no redraw.
- Each terminal's text was read back and checked for exactly the new rows below the header, nothing left over:
  - WezTerm and kitty through their own command-line tools;
  - iTerm2 and Terminal.app by AppleScript, each window named by the id its creation returned;
  - xterm through its printer command;
  - xfce4-terminal through Select All and the selection, on a private Xvfb display;
  - Windows through a scratch pseudo-console harness (`CreatePseudoConsole`, `ResizePseudoConsole`, as Windows
    Terminal hosts a console), its output drawn with `pyte`.

**A mistake, put right:** the first iTerm2 driver addressed "the current window", which was not always the probe's.
It read, and then closed, "the current window" three times. iTerm2 had not been running and was launched by the
driver at 10:13:37, so every window it had was the driver's own: the three probes' and the default window iTerm2 opens
at launch. None of the owner's windows was touched. The driver was rewritten to name only the window its creation
returned, and to quit an app it launched.

**Step 1, narrowing from 80 columns** (to 41–43):

| Terminal | Cursor's row after | reflow | truncate | none |
|---|---|---|---|---|
| WezTerm (a pane narrowed by a split) | down, 4 → 7 | clean | rows left | words cut (`lazy d` / `og`) |
| kitty | down, 4 → 7 | clean | rows left | (re-wrapped) |
| Terminal.app | down, 4 → 7 | clean | rows left | (re-wrapped) |
| Windows pseudo-console | down, 4 → 7 | clean | rows left | words cut |
| iTerm2 | **kept, 4** | the prompt's first row left, above | rows left | (re-wrapped) |
| xfce4-terminal (VTE) | **kept, 4** | the prompt's first row left, above | rows left | (re-wrapped) |
| xterm | kept, 4, column cut to 43 | header gone | **clean** | rows cut at 43 |

Three behaviours:

- **Re-wraps, cursor moves down:** WezTerm, kitty, Terminal.app, Windows. The reflow method is clean.
- **Re-wraps, cursor keeps its row:** iTerm2 and VTE. They push the re-wrapped rows above the cursor into the
  scrollback, so the prompt's top is out of reach of any cursor move, and the redraw leaves its old first row there.
- **Cuts rows off, no re-wrap:** xterm. The truncate method is clean.

The cursor's place after a resize (DSR) tells the three apart: a row moved down by the re-wrapped count; a row kept
with a re-wrapped column; or a row kept with the column cut to the width. In none does the live redraw stay clean
everywhere without knowing which.

**Step 2, Windows' resize:** after `ResizePseudoConsole`, the console's input queue held one record, a
`WINDOW_BUFFER_SIZE_EVENT`, and the probe's next read in virtual-terminal input mode returned the DSR reply alone. A
ReadFile reader does not receive the event. So D1's 100 ms poll stands as written. The probe's poll saw the change
(`80x24 -> 43x24`). Waiting on the input handle and peeking would see the event without polling, at the cost of a
second Windows read path.

**Step 3, the baseline:**

- **The terminals,** with the scrollback filled, then 1,000 and 10,000 rows written as one synchronized update. The
  time is to the reply of a DSR sent after the write, and covers parsing, not painting:

  | Terminal | 1,000 rows | 10,000 rows | `CSI 3 J` | Old lines kept without it (the control) |
  |---|---|---|---|---|
  | WezTerm | 7–12 ms | 32–40 ms | cleared | 11,300 |
  | kitty | 4–5 ms | 10 ms | cleared | 2,030, its scrollback's limit |
  | iTerm2 | 56–169 ms | 41–108 ms | cleared | 10,022, its scrollback's limit |
  | Terminal.app | 2 ms | 25–26 ms | cleared | 11,300 |
  | xfce4-terminal | 17 ms | 22–23 ms | cleared | 998, its scrollback's limit |
  | xterm | 7 ms | 69–80 ms | **not verified**: its print gave the screen only (22 lines in the control) | — |
  | Windows pseudo-console | 3 ms | 30 ms | passed on to the hosting terminal (`CSI 2 J`, `CSI 3 J` in its output) | — |

  kitty keeps 2,000 lines and xfce4-terminal 1,000 by default, so most of a 10,000-row reprint scrolls out of
  them.
- **gobble,** benchmarks in a scratch copy, five runs on each host. A reply (1.6 KB) is rendered as the view does:
  streamed in chunks of 64 bytes, styled, wrapped. The transcript is that reply repeated to 4 MB, rendered at 60
  columns.

  | Benchmark | macOS | Linux machine | Windows laptop | Allocations |
  |---|---|---|---|---|
  | one reply | 92–94 µs | 312–373 µs | 80–250 µs | 2,673 (52 KB) |
  | the whole 4 MB | 0.42 s | 1.25–1.50 s | 1.12–1.15 s | 12.2 million (238 MB) |
  | the last 10,000 rows | 36 ms | 113–126 ms | 95–98 ms | 1.03 million (20 MB) |

  Rendering allocates about 1.6 times per byte, since the wrapper and the Markdown stream build their words and
  lines a character at a time.
- **The line session,** in a pty, three runs each:
  - start to the first prompt: macOS 55–108 ms, the Linux machine 71–95 ms;
  - a key to its echo: median 0.1 ms on macOS and 0.3–0.4 ms on the Linux machine, at worst 3.5 ms;
  - Enter to the stubbed reply's last row: macOS 16–31 ms, the Linux machine 40–68 ms;
  - a resize: no output within 2 s on either.

  Windows was not timed, since the headless host is driven over ssh.

**Deviation 1, 2026-10-22, decided by the owner** ("Pick per resize"). Step 1 found that no single live redraw is
clean everywhere: iTerm2 and VTE push the prompt's top out of reach, and xterm cuts rows off where the others re-wrap.
After each resize the editor reads the cursor's place (DSR) and picks the matching method. In iTerm2 and VTE it draws
from the screen's first row and leaves the rest to D3. 0030-MADR D2 is amended. Added to P2: reading the DSR reply in
the key reader, and tests of the three methods.

**Deviation 2, 2026-10-22, decided by the owner** ("Yes, add it"). Step 3 found rendering at about 1.6 allocations per
byte. P4 gains step 4, reworking `render.Wrap` and `render.Stream` against targets of 3x fewer allocations and 2x
faster reprints. 0030-MADR D5 is amended.

**P1 complete, 2026-10-22.** It changed nothing in the tree; its programs stayed in scratch.

### P2 execution record (2026-10-22)

1. **Tests first,** each seen failing on stubs:
   - The screen model (`internal/cli/term/termtest`). `TestScreenAnswersCursorQuery` failed with `reply "", want
     "\x1b[2;5R"`. `TestScreenResize`, with P1's three behaviours (`ReflowDown`, `ReflowKeepRow`, `Cut`), failed with
     the rows unchanged. Its first draft used a 6-row screen, where the re-wrapped rows scrolled the header away in
     both re-wrapping kinds; it was redrawn on 10 rows before the code was written.
   - The watcher (`internal/cli/term`). `TestPollSizeNotices` and `TestWatchResizeSignal`, which sends SIGWINCH to the
     test process, failed with `no channel`. `TestWatchResizeNotATerminal` passed, as it asks for none.
   - The reader. `TestReaderWakes` failed with a deadlock on a `ReadUntil` that ignored the resize.
   - The editor (`internal/cli/editor/resize_test.go`). `TestResizeRedrawsAtOnce`, in the three kinds, and
     `TestResizeWithoutAReply` failed with the owner's cut words on screen (`"words fi" "ll words …"`).
     `TestCursorReportIsNoText` passed: an unknown sequence was already dropped.
   - The line session. `TestLineSessionRedrawsOnResize` failed with `condition not met within 5 s`: no cursor query.
   - The view. `TestReplyAfterResizeWrapsAtNewWidth` passed: 0028 already reads the width at each release. It stays
     as a guard, and a mutation shows it can fail.
2. **The code:**
   - `term.WatchResize`: SIGWINCH on macOS and Linux; on Windows the console's size read every 100 ms; notices
     coalesced; nil for a file that is not a terminal.
   - `editor.Reader.ReadUntil`: a read that a resize, or the end of a wait, stops with no bytes, its asked read kept.
   - In the editor, `Options.Resize`. On a resize it writes `CSI 6 n` and waits up to 150 ms (`cursorWait`) for the
     report, holding typed keys meanwhile. It then redraws from `renderer.lift` rows up, as a first drawing.
     `takeCursorReport` takes the report out of the input.
   - `lift` counts the rows the old ones re-wrap to, or the rows as drawn where the terminal cuts. The cursor
     report's column decides which when the cursor's column was past the new width, `XTERM_VERSION` otherwise.
   - The line session passes `term.WatchResize` on stdout to the editor.
3. **A real terminal,** a look beyond the plan: gobble built from the working tree in a new WezTerm window, against the
   scratch stub. A long prompt was sent as text, and the pane narrowed by a split from 80 to 43 columns, with no key
   after. The prompt was clean: 5 rows at 43, every word whole. The same check on a build of `5a2d106` (before P2)
   showed `"> … long enough to fill t"`, `"hree rows …"`.

**Negatives,** on a scratch copy, one mutation at a time:

- SIGWINCH not noticed;
- a change of size not noticed;
- notices not coalesced;
- a resize not waking the read;
- a woken read asked again (`2 reads of stdin, want 1`);
- no cursor query;
- the report not taken out;
- a cut never told by the column;
- the rows as drawn, never re-wrapped;
- the first drawing not moved up;
- keys fed while the report is awaited;
- the line session giving the editor no resize;
- the reply's width read once (`row 2 is 97 columns, more than 40`);
- the model's kept row lost;
- the model answering no query.

Two passed at first:

- **Capping the lift at the cursor's row** was an equivalent mutation: moving up stops at the screen's first row in
  every terminal and in the model, so the cap changed nothing. It was taken out, and with it `resized`'s row.
- **Keys fed while the report is awaited:** no test typed during the wait. `TestKeysTypedWhileTheReportIsAwaited`
  was added, its keys sent between the query and the reply.

Fifteen mutations, fifteen failures; the restored copy passed.

**Found by the gates and fixed in the phase:**

- revive's `confusing-naming` on macOS: `watchResize` differed from `WatchResize` only by case, and was renamed
  `resizeNotices`.
- `errcheck` on the Windows laptop alone: the size read's error was discarded in `resize_windows.go`, a file the other
  hosts do not build. It is now handled: a failed read counts as no size.

**Gates, after both fixes:**

- The pre-add check on the sixteen changed `.go` files: `16 file(s) clean (gofmt, golint, govulncheck)`.
- macOS: `make preflight`, `preflight passed`; the tests and the race detector, 36 packages ok each.
- The Linux machine and the Windows laptop, on a copy of the tree: the tests and the race detector, 36 packages ok on
  each. `make preflight` reached `apidiff` (golangci-lint `0 issues.` on both, govulncheck "No vulnerabilities
  found.") and stopped at `verify-build-metadata`, as the copies are not git checkouts.
- `make check-records markdownlint`: "0 issues in 0 files".
- The Windows laptop, `make probe-conhost`: `--- PASS: TestConhost`.

**P2 complete, 2026-10-22.**

### P3 execution record (2026-10-22)

1. **Tests first,** each seen failing on stubs:
   - The transcript (`internal/cli/transcript_test.go`): `TestTranscriptRendersAtAWidth` (fixed text as written, a
     message wrapped, a function of the width at 20 and 40), `TestTranscriptRenderKeepsTheNewestRows` and
     `TestTranscriptBound`. Each failed with an empty render.
   - The editor (`internal/cli/editor/resize_test.go`):
     - `TestReprintOnSettle` failed with `"old output" "> hello world"` still on screen;
     - `TestResizesReprintOnce` failed with `0 reprints, want 1`;
     - `TestShownLines` failed with `shown at 12: []`.
   - The view: `TestViewReprintsAtTheNextLineEnd` failed with `not reprinted at the line's end`. `TestWrapIdle` failed
     with `a new wrapper is not idle`.
   - The line session (`internal/cli/reprint_test.go`): `TestLineSessionReprintsOnResize` covers a reply with prose,
     a list and code, the startup warning, a sent prompt, the status line and `/help`, narrowed from 100 to 60
     columns. Its first run failed because of the test: it waited for `to the point.` as one piece of text, which the
     reply's wrapping at 100 columns broke across rows. It was corrected to look for the words; the mutations below
     show it failing for its own reasons. `TestNoTranscriptOffATerminal` passed, as it asks that nothing changes off
     a terminal.
2. **The code:**
   - `transcript` (`internal/cli/transcript.go`): pieces drawn at a terminal's width:
     - fixed text;
     - messages (`render.Message`, to 120 columns);
     - functions of the width (`/help`, the editor's lines);
     - growing sources (a reply's Markdown, thinking).

     It is bounded at 4 MB by bytes alone. `render` walks back from the newest piece to 10,000 rows and draws no
     piece it will not show. `reprintText` puts a dimmed line first when anything was left out.
   - `Output.rec`: `Result` keeps fixed text and `diag` keeps a message's source; `emit` writes without keeping.
   - The view keeps a reply's and thinking's sources and shows their rows unkept; everything else it writes is kept
     as written, the status line too. `resized` starts a 150 ms settle. The reprint is made when the view is at a
     line's start with both wrappers idle (`render.Wrap.Idle`); a reprint still due when the turn ends goes to the
     editor (`afterTurn`).
   - The editor: `Options.Shown` for the header, each line sent (its layout at any width) and each notice. After
     the live redraw, a 150 ms settle wait, then `Options.Reprint` drawn behind `CSI 2 J`, `CSI 3 J` and home. The
     renderer's `prefix` puts all of it, and the line, in one synchronized update.
   - The line session makes the transcript only when it has a resize notice, that is, on a terminal. It routes the
     terminal's resizes to the view while a turn runs, else to the editor (`route`). `/help` is kept as
     `helpText(width)`, and a permission question with its answer as shown.
3. **A real terminal,** a look beyond the plan: gobble in a new WezTerm window against the scratch stub's mixed reply,
   the pane narrowed by a split after the reply. The whole pane, scrollback included, was read once settled. At 43
   columns: prose whole, no row over the width, the list continued, the reply shown once. The same check on a build of
   `d772e34` (before P3) showed `"…specific task—wh"`, `"ether it’s writing code…"`.

**Negatives,** on a scratch copy, one mutation at a time:

- a reprint at each resize, no settle (`3 reprints, want 1`);
- the scrollback not cleared;
- the reprint not synchronized;
- the editor's lines not shown;
- the header not shown;
- messages not kept;
- the reply kept as written, not as its source (its words cut at 60);
- the reply's rows kept as well as its source (`reprinted 2 times`);
- the status line not kept;
- `/help` kept as written (`a /help row of 60 columns, more than 59`);
- the bound not kept;
- the row limit not kept (`100 pieces drawn for 10 rows`);
- a transcript kept off a terminal;
- a reprint in the middle of a line;
- a late reprint not handed to the editor;
- the router sending nothing to the editor;
- a wrapper idle in a partial escape.

Four passed at first:

- **The reply's rows kept as well as its source:** the tests looked for the words but not how often they appeared.
  Both reprint tests now count the reply's first line.
- **The row limit:** the output was the same, but every older piece was drawn, which is the cost the limit exists to
  save. `TestTranscriptRenderDrawsOnlyWhatItShows` counts the pieces drawn.
- **The wrappers reset after a reprint:** an equivalent mutation. A reprint is made only when both wrappers are
  idle, and a Stream releases a code fence only once it is closed, so a fresh wrapper and an idle one are the same.
  The reset was taken out. A test written for a reprint inside an open fence was dropped, since the Stream never
  lets the wrapper see one.
- **`Idle` ignoring held words:** equivalent, since a line's words are empty at its start. `Idle` was cut to what
  matters, and a partial escape case was added to its test.

Seventeen mutations, seventeen failures; the restored copy passed.

**Found by the gates and fixed in the phase:** gofmt drift in `view.go`, and `modernize`'s `slices.Backward` and
`strings.Cut`. Each was fixed before the gates ran on all three hosts.

**Gates:**

- The pre-add check on the fourteen changed `.go` files: `14 file(s) clean (gofmt, golint, govulncheck)`.
- macOS: `make preflight`, `preflight passed`; the tests and the race detector, 36 packages ok each.
- The Linux machine and the Windows laptop, on a copy of the tree: the tests and the race detector, 36 packages ok on
  each. `make preflight` reached `apidiff` (golangci-lint `0 issues.` on both, govulncheck "No vulnerabilities
  found.") and stopped at `verify-build-metadata`, as the copies are not git checkouts.
- `make check-records markdownlint`: exit 0 on each host.
- The Windows laptop, `make probe-conhost`: `--- PASS: TestConhost`.

**P3 complete, 2026-10-22.**

### P4 execution record (2026-10-22)

**Deviation 3, 2026-10-22, decided by the owner** ("option 1"). The gates found `make preflight` failing at
`markdownlint` on macOS and the Windows laptop: `internal/cli/render/testdata/reply.md:18:5 error MD034/no-bare-urls
Bare URL used`. The file is new in P4: the benchmarks' mixed reply, whose long bare URL is deliberate, a word wider
than the row, as a model sends it. It is input for the renderer, not documentation, so it is renamed
`internal/cli/render/testdata/reply.txt`, and its four readers in the tests follow. Its content is unchanged, and
markdownlint keeps covering every document. No decision changes, so 0030-MADR is not amended.

**Deviation 4, 2026-10-22, decided by the owner** ("Post-rework, record it"). Step 1 says the baselines are P1's.
P1's figures came from a benchmark in a scratch copy, not from the benchmarks P4 adds: on the Windows laptop P1
measured the last 10,000 rows at 95–98 ms, and P4's `BenchmarkReprint` on HEAD's unchanged code at 48.2 ms. Three
times P1's figures would also pass a reworked reprint slowed several times over (108 ms against 9.4 ms on macOS). The
baselines are therefore P4's own benchmarks after step 4's rework, the median of five runs on each host, and the
reprint's allocation budget is the count after the rework (5,051) and a quarter more, 6,300. The targets against P1
are checked once, below. 0030-MADR D5 is amended.

**Deviation 5, 2026-10-22, decided by the owner** ("Record it, leave Stream"). Step 4 names `render.Wrap` and
`render.Stream`, after P1's finding that both build their words and lines a character at a time. That is true of
the wrapper only. `render.Stream` appends once per chunk (`internal/cli/render/mdstream.go`, `m.buf += chunk`) and
returns slices of its buffer, so it is left as it is; the targets are met by the wrapper's rework alone. 0030-MADR
D5's amendment is annotated.

1. **Benchmarks,** each on the reply of 0028-PLAN P4's captures (`internal/cli/render/testdata/reply.txt`, 1.6 KB:
   prose, a list, code, a table and a long URL):
   - `render`: `BenchmarkReplyRender`, the reply streamed in chunks of 64 bytes, styled and wrapped as the view does;
     `BenchmarkReplyRedraw`, the same reply drawn again whole at a new width;
   - `cli`: `BenchmarkReprint`, a 4 MB transcript of replies drawn back to its last 10,000 rows at 60 columns, as a
     reprint does; `BenchmarkLineSessionStart`, the line session in process from start to its first prompt;
   - `editor`: `BenchmarkKeyDraw`, a key typed at the end of a long prompt, drawn.

   The line session's test harness (`newLineHarness`, `withAgent`) takes `testing.TB`, so the start-up benchmark
   shares it. No test changed otherwise.
2. **Budget tests,** run by `go test` on every host, since they count rather than time:
   - `TestKeyDrawBudget` (`editor`): a key at a long line's end, one in its middle, and a cursor move each write at
     most the bytes after what each row's old and new text share, plus 40. It passed on HEAD's code, as 0027's
     renderer already rewrites only a row's changed end; the first mutation below shows it failing.
   - `TestReprintAllocationBudget` (`cli`): 1,000 rows of replies drawn again at 60 columns allocate at most 6,300
     times. On HEAD's code it failed at 99,821 allocations; after step 4, 5,051.
3. **`make perf`:** `TestPerfBudget`, tagged `perf`, in `cli` (ReplyRender, Reprint, LineSessionStart) and in
   `editor` (KeyDraw), each against three times its baseline for the host's GOOS. A host with no baseline is logged,
   not failed. The baselines (deviation 4):

   | Benchmark | macOS | Linux machine | Windows laptop |
   |---|---|---|---|
   | ReplyRender | 30 µs | 110 µs | 54 µs |
   | Reprint | 9.4 ms | 29 ms | 12 ms |
   | LineSessionStart | 830 µs | 2.9 ms | 6.9 ms |
   | KeyDraw | 1.5 µs | 5 µs | 1.4 µs |

4. **The optimisation:**
   - `render.Wrap` keeps its words, gap, held word and line in byte buffers that are reused, and swaps them where it
     copied strings: a held word becomes the current one by exchanging the two buffers. Runes are decoded in place
     (`utf8.DecodeRuneInString`); a column advances by `advance(col, r)`.
   - `runeWidth`, shared with the table, answers printable ASCII (most of what is drawn) without the width tables.
   - `render.Stream` is not changed (deviation 5). With the wrapper reworked, a reply's allocations fell from 2,673
     to 242 without it.

   Every render test and every split test passes unchanged.

**Measurements,** five runs on each host of P4's benchmarks, on HEAD's code (with P4's test files, which run on it
unchanged) and after step 4. Medians:

| Benchmark | macOS | Linux machine | Windows laptop | Allocations | Bytes |
|---|---|---|---|---|---|
| ReplyRender | 94.1 → 29.5 µs (3.2×) | 464 → 109 µs (4.2×) | 134 → 53.8 µs (2.5×) | 2,673 → 242 | 52 → 14.7 KB |
| ReplyRedraw | 87.9 → 23.7 µs (3.7×) | 437 → 73.7 µs (5.9×) | 229 → 63.4 µs (3.6×) | 2,559 → 129 | 51 → 13.3 KB |
| Reprint | 38.0 → 9.4 ms (4.0×) | 118 → 29.4 ms (4.0×) | 48.2 → 11.6 ms (4.1×) | 985,246 → 49,696 | 21.7 → 7.3 MB |
| LineSessionStart | 960 → 834 µs | 3.6 → 2.9 ms | 6.5 → 6.9 ms | 4,604 (Windows 4,832), unchanged | unchanged |
| KeyDraw | 1.5 → 1.5 µs | 4.9 → 5.0 µs | 1.4 → 1.4 µs | 8, unchanged | 808, unchanged |

After the rework the runs spread least on macOS (within 5 per cent), more on the Linux machine (Reprint 25.2–31.0 ms)
and most on the Windows laptop (ReplyRender 24.8–79.4 µs, Reprint 7.3–25.4 ms).

**The targets, against P1's baseline:**

- Allocations at least 3× fewer: a reply 2,673 → 242 (11×); the last 10,000 rows 1.03 million → 49,696 (20×).
- The last 10,000 rows at least 2× faster: macOS 36 → 9.4 ms (3.8×), the Linux machine 113–126 → 29.4 ms (3.8–4.3×),
  the Windows laptop 95–98 → 11.6 ms (8.2–8.4×). Against HEAD's code in P4's own benchmark, 4.0–4.1× on each host.

Both met. Start-up and a key's drawing were not targets and did not change.

**Negatives,** one mutation at a time on a scratch copy; each must fail the named test (the timing ones with
`-tags perf`):

- **A changed row written whole** (`TestKeyDrawBudget`): `104 bytes for 1 runes changed, more than 41`.
- **The word built by joining strings, rune by rune** (`TestReprintAllocationBudget`): `27125 allocations for 1,000
  rows, more than 6300`.
- **A held word copied, not swapped** (`TestReprintAllocationBudget`): `12305 allocations for 1,000 rows, more than
  6300`.
- **The reprint slowed by 30 ms** (`TestPerfBudget`): `Reprint: 44.455666ms per op, more than 3 times its baseline
  9.4ms`.
- **A key's drawing slowed by 20 µs** (`TestPerfBudget`): `KeyDraw: 29.865µs per op, more than 3 times its baseline
  1.5µs`.
- **The ASCII fast path's width wrong,** **a gap carried onto a new row,** and **widths counted in bytes**
  (`TestWrap`): each failed with wrapped rows differing from the want.

Eight mutations, eight failures; the restored copy passed.

**Found by the gates and fixed in the phase:** `markdownlint` on the benchmarks' reply (deviation 3).

**Gates:**

- The pre-add check on the eleven changed `.go` files: `11 file(s) clean (gofmt, golint, govulncheck)`.
- macOS: `make preflight`, `preflight passed`; the tests and the race detector, 36 packages ok each.
- The Linux machine and the Windows laptop, on a copy of the tree: the tests and the race detector, 36 packages ok on
  each. `make preflight` reached `apidiff` (golangci-lint `0 issues.` on both, govulncheck "No vulnerabilities
  found.") and stopped at `verify-build-metadata`, as the copies are not git checkouts.
- `make check-records markdownlint`: exit 0 on each host, and again on macOS once this record was written.
- The Windows laptop, `make probe-conhost`: `--- PASS: TestConhost`.
- `make perf`, each within three times its baseline:

  | Benchmark | macOS | Linux machine | Windows laptop |
  |---|---|---|---|
  | ReplyRender | 32.6 µs (30) | 136 µs (110) | 28.9 µs (54) |
  | Reprint | 9.57 ms (9.4) | 29.9 ms (29) | 7.65 ms (12) |
  | LineSessionStart | 827 µs (830) | 3.73 ms (2.9) | 5.13 ms (6.9) |
  | KeyDraw | 1.50 µs (1.5) | 5.28 µs (5) | 1.08 µs (1.4) |

**P4 complete, 2026-10-22.**
