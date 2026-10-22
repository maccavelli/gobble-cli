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
