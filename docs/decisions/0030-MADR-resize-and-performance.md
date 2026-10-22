---
status: accepted
date: 2026-10-22
decision-makers: repository owner
consulted: measurements of the line session in a pseudo-terminal on 2026-10-22 (below); 0027-MADR and 0028-MADR
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Redraw the native CLI mode at the new width when the terminal is resized, and hold its performance to a measured budget

## Context and Problem Statement

Trying 0028-PLAN P4's builds on 2026-10-22, the owner reported:

> resizing by narrowing the window can cut words the wrap doesn't dynamically adjust. resizing causes a pretty long
> delay before the terminal is re-drawn, we need to look at how we can improve terminal performance, making it more
> "snappy" and "fluid". we need to ensure line wrap works during terminal resizing. we need to optimize the cli mode
> overall to keep it lean and performant, but also rich and full-featured.

**Measured** on 2026-10-22, macOS, a pseudo-terminal 120 columns wide, gobble built from `ec998bc`, its model a scratch
stub on 127.0.0.1, three runs:

| What | Result |
|---|---|
| Start to the first prompt | 50 to 102 ms |
| A key to gobble's first output byte, 83 keys | median 0.1 to 0.2 ms, worst 1.2 to 1.5 ms |
| Enter to the stubbed reply's last row | 16 to 31 ms |
| The terminal narrowed to 60 columns, at the prompt | **no output within 2 s** |

Typing and drawing are fast. The delay the owner sees is gobble not reacting to a resize at all. Nothing in
`internal/cli` handles `SIGWINCH`, or a Windows console's size change. The editor redraws only after a key, and every
row already written (0027's prompt rows, 0028's wrapped replies and messages) stays as it was written. The terminal
then re-wraps those rows at its new edge, mid-word if it must. 0028-MADR accepted this as a consequence ("the breaks
are fixed when written"); the owner's report reverses it.

A line-mode client cannot change rows already in the terminal's scrollback. It can only clear the screen and its
scrollback, and write everything again. A full-screen program redraws its own screen; that is the TUI mode
(0002-MADR), not this one.

## Decision Drivers

* The owner's requirements: words are never cut, before or after a resize; a resize is redrawn at once; the CLI mode
  stays lean and fast while it gains features.
* Measured, not guessed: a baseline, and a budget that a test enforces.
* The prompt, replies, messages, tool output, the permission prompt (0029) and slash-command output all follow one
  rule.
* macOS, Linux and Windows (Console Host and Windows Terminal) alike.

## Considered Options

The owner chose among these on 2026-10-22:

* **A. Redraw the live area at once, and reprint the conversation once resizing settles.** Chosen.
* **B. Redraw the live area only;** rows already written stay as they are.
* **C. The live area, and the conversation reprinted on request** (a key or command), never on its own.

## Decision Outcome

Chosen option: "A", because it is the only one under which no word is ever cut. The owner accepted its cost:
clearing the terminal's scrollback.

- **D1. Noticing a resize.** On macOS and Linux, `SIGWINCH`. On Windows, the console's size, read every 100 ms while
  the line session runs, unless P1 finds a console event that a reader in virtual-terminal input mode receives.
  A resize wakes the editor's read and the view; it is not a key.
- **D2. The live area, at once:** the prompt being typed, and the rows of a reply still arriving, drawn again at the
  new width within 50 ms of the resize. Before this is built, P1 measures how each terminal reflows the cursor's
  rows when narrowed. If any terminal leaves the cursor somewhere the editor cannot find, the live area there waits
  for D3, and that is recorded as a deviation.
- **D3. The conversation, on settle.** About 150 ms after the last resize, gobble writes, at the new width:
  - clear the screen and the scrollback (`CSI 2 J`, `CSI 3 J`, home);
  - everything the line session has shown;
  - the live area.

  It writes this as one synchronized update (DEC mode 2026, as 0027's drawing does).
- **D4. The transcript.** The line session keeps what it has shown, in order, as pieces it can draw again at a width:
  - the header and each sent prompt (0027's layout);
  - each reply and its thinking, kept as the Markdown released (0028's styling and wrapping);
  - each message (`render.Message`);
  - output whose layout depends on the width, such as `/help`'s list, kept as a function of the width;
  - everything else, such as tool output, diffs and the status line, kept as written.

  It is bounded: at most 2 MB or 2000 pieces. Past that, the oldest go, and the reprint begins with one line saying
  so.
- **D5. A performance budget,** with each item a test or benchmark in `internal/cli`, measured in P1 and enforced
  from P2:
  - a key's drawing writes at most the changed text plus 40 bytes;
  - rendering a 1000-row transcript at a new width allocates at most a recorded number of times;
  - benchmarks for start-up to the first prompt, a key, a reply's rendering, and a reprint of 1000 rows, with
    recorded baselines;
  - a budget test that fails when a benchmark is more than 3× its baseline. That test is tagged `perf` and run by
    `make perf`, not by `make preflight`, since timing varies by host.
- **D6. Print mode and off a terminal:** unchanged. No transcript is kept, and nothing is redrawn.
- **D7. Supersedes** 0028-MADR's consequence "the breaks are fixed when written" for the line session on a
  terminal, and 0027-PLAN P4's "found, not fixed" note on reflow when narrowing. Both are recorded there as dated
  notes.

### Consequences

* Good, because a resize is redrawn at once and no word is cut, before or after it.
* Good, because performance has a baseline and a test, so a regression is noticed.
* Bad, because a resize clears the terminal's scrollback, including what came before gobble started.
* Bad, because the transcript costs memory, up to its bound, and a reprint of a long session takes time. D5
  measures both.
* Bad, because Windows may need a 100 ms poll of the console's size.

### Confirmation

0030-PLAN's tests, each seen failing first, and its measurements before and after. Then captures on all three hosts,
with a resize mid-prompt and mid-reply; the owner's look in WezTerm and Windows Terminal; and the gates.

## Pros and Cons of the Options

### A. Live area at once, conversation on settle

* Good, because nothing on screen is ever cut.
* Bad, because the scrollback is cleared, and the transcript and reprint cost memory and time.

### B. Live area only

* Good, because it is the least work, and the scrollback is untouched.
* Bad, because rows already written are cut by the terminal when the window narrows, which the owner rules out.

### C. Live area, conversation on request

* Good, because the scrollback is cleared only when asked.
* Bad, because cut words remain until the person asks.

## More Information

- [0027-MADR-native-line-editor-word-wrap.md](0027-MADR-native-line-editor-word-wrap.md) D3: the prompt's drawing.
- [0028-MADR-reply-word-wrap.md](0028-MADR-reply-word-wrap.md): replies and messages wrapped, breaks fixed when
  written.
- [0029-MADR-permission-prompt-layout.md](0029-MADR-permission-prompt-layout.md): comes after this, by the owner's
  order.
- The plan: [0030-PLAN-resize-and-performance.md](0030-PLAN-resize-and-performance.md).
