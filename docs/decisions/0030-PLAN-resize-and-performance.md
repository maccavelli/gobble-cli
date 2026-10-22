---
status: proposed
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
3. **The baseline:** start-up, a key, a reply's rendering, and the costs a reprint of 1000 rows would have, as
   benchmarks in a scratch copy, on all three hosts.

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
   - the bound: past 2000 pieces, the oldest go and the reprint says so;
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
