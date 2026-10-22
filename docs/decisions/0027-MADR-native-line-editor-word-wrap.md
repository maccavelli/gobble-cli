---
status: accepted
date: 2026-10-21
decision-makers: repository owner
consulted: golang.org/x/term v0.47.0 (terminal.go); a survey of Go line editors (chzyer/readline, ergochat/readline, reeflective/readline, nyaosorg/go-readline-ny, godevsig/glib lined)
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# gobble's own line editor for the native CLI mode, wrapping whole words while the user types

## Context and Problem Statement

The owner reported on 2026-10-21, after `9dcd596`: "The prompt is being cut and wrapped at around 80 chars, including
in the middle of words. My terminal is much wider than 80 chars. The line wrap needs to be dynamic, based on the
available width of the user's terminal to a max of 120 chars, ensuring not to cut words or punctuation such as a
trailing comma when wrapping the prompt text." Asked whether words should be kept whole once a prompt is entered or
while it is typed, the owner answered "Whole words while typing"
([0024-MADR-cli-mode-polish.md](0024-MADR-cli-mode-polish.md), Q9). The width alone ships first, as 0024's D9. Each fact
below was measured on 2026-10-21 at `9dcd596`.

**The line editor today** is `golang.org/x/term`'s `Terminal` behind a thin adapter
([0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D6, from its finding F7: "build on `x/term` with a thin
adapter"). `internal/cli/editor` is 591 lines around it: `editor.go` (the read loop, prompts, the header), `keys.go`
(the key reader: two-stage Ctrl+C, paste markers), `paste.go` (coalescing a paste's lines), `history.go`, `reader.go`
(the reader that owns stdin), `external.go` (`/edit`), and Console Host's virtual-terminal input on Windows.

**What `x/term` cannot do** (`golang.org/x/term` `v0.47.0`, `terminal.go`):

- **It wraps by character.** It places each character at the next column, and when the column reaches its width writes
  `\r\n` itself (`advanceCursor`, `:381-401`). It has no notion of a word, so a word at the edge is split.
- **Its layout is private.** The line, the cursor and the rows drawn are unexported fields; redrawing them in another
  shape means drawing them itself, which is the editor's rendering, not an adapter.
- **What it handles,** which gobble keeps whichever editor draws the line: Ctrl+A/E, Home/End, Ctrl+B/F and the arrows,
  Alt+←/→ by word, Backspace, Delete, Ctrl+D, Ctrl+K, Ctrl+U, Ctrl+W, Ctrl+T, Ctrl+L, ↑/↓ and Ctrl+P/N for history,
  bracketed paste, and a callback for each printable key and Tab (`bytesToKey`, `:181-270`; `handleKey`, `:511-662`).

**No library found does it.** A survey on 2026-10-21 of Go line editors (chzyer/readline, ergochat/readline,
reeflective/readline, nyaosorg/go-readline-ny, godevsig/glib's `lined`) found none that documents keeping words whole
while the user types: chzyer/readline lists "single-line input strings that are long enough to wrap" among its TODOs;
reeflective/readline documents multi-line buffers, not word wrap. The survey read their documentation, not their code.
go-tui-lib hosts Bubble Tea's `textarea`, but only `internal/tui` may import Charm or go-tui-lib
([0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md), rule 5), and the native mode is a line
client, not a full-screen program.

**What gobble already has:** word wrap to a width (`render.wrapIndented`, `internal/cli/render/commands.go:118`,
breaking at spaces); display widths of wide characters (`golang.org/x/text/width`, required, used by
`internal/cli/render/table.go`); the terminal's width (`internal/cli/term/caps.go:49`); raw mode and the reader that
owns stdin (`internal/cli/editor`).

## Decision Drivers

* The owner's requirement: the prompt wraps at the terminal's width, to 120 columns, never inside a word or before its
  punctuation, while it is typed.
* Nothing the editor does today is lost: every key above, history, paste, completion, the two-stage Ctrl+C, the
  header, `/edit`, Windows Console Host.
* No new module (AGENTS.md: a module needs a MADR naming it); rule 5 keeps Charm out of `internal/cli`.
* Testable without a terminal: the layout a pure function, the drawing checked against a screen model.

## Considered Options

* **A. gobble's own line editor:** a buffer, a word-wrapping layout and a renderer in `internal/cli/editor`, in place of
  `x/term`'s `Terminal`; `x/term` kept for raw mode and the terminal's size.
* **B. A third-party line editor,** adopted with a MADR naming its module.
* **C. Keep `x/term`'s `Terminal`,** with the width (0024 D9) and words split at the edge while typing.

## Decision Outcome

Chosen option: "A. gobble's own line editor", because it is the only option that meets the owner's answer (Q9): no
library found keeps words whole while typing, and `x/term`'s layout cannot be reshaped from outside. *Proposed: the
owner decides at review.* *(2026-10-21: accepted by the owner, "Both, P4 then 0027", with Q1 and Q2 answered A.)*

- **D1. The buffer.** The editor holds the line being typed as text and a cursor position, in runes, and the prompt in
  front of it. It replaces `x/term`'s `Terminal` for reading and drawing the line; `x/term` stays for raw mode
  (`MakeRaw`), the terminal's size (`GetSize`) and `IsTerminal`. It amends 0008-MADR D6 with a dated note.
- **D2. The layout,** a pure function of the prompt, the text and the width, so it is tested without a terminal:
  - the width is the terminal's, read when the line is drawn, to at most 120 columns, 80 when unknown;
  - rows break only at spaces: a word keeps its punctuation (`comma,`, `end.`, `"quoted"`); the space at a break is
    not drawn, and the text is unchanged;
  - a word wider than a whole row is cut at the row's end, the only cut inside a word;
  - continuation rows are indented under the text, by the prompt's width;
  - characters are measured by their display width (`golang.org/x/text/width`), so a wide character takes two columns;
  - the cursor's row and column are found from the same layout, including the cursor at the end of a row.
- **D3. The drawing.** After each key, the editor draws the line again: up to the line's first row (the rows it drew
  last time), `\r`, erase to the end of the screen, the rows, then the cursor moved to its place, by relative moves.
  The terminal's width is read at each drawing, so a resize takes effect at the next key on every system, without a
  signal. A line taller than the screen is drawn from the row that holds the cursor. After Enter, the line stays as
  drawn, word-wrapped, and the cursor goes below it.
  *Amended 2026-10-21 (0027-PLAN P4, deviation 4; the owner's answer "Draw without blanking"):* the owner saw the
  line flicker in Windows Terminal. Windows Terminal's console host passes the drawing through as written, and
  erasing to the end of the screen before writing the rows let the terminal paint the line blank between the two.
  The cursor also jumped to column 0 and back. The drawing no longer erases first. It rewrites only the rows that
  changed, each from its first changed column, then erases what is left of a row that got shorter and any rows the
  line no longer has. The cursor is hidden while it moves. Each drawing that writes is wrapped in synchronized output
  (DEC mode 2026), which a terminal that lacks it ignores. A first drawing still starts with `\r` and an erase to
  the end of the screen, since nothing of the line is on screen yet.
  *Amended 2026-10-21 (0027-PLAN P5; approved with the owner's answer "Fix, P4, P5, one look"):* Ctrl+D on an empty line and a second Ctrl+C leave the line the
  same way, the cursor at the start of the row below it. The line session leaves raw mode and bracketed paste before
  it writes the goodbye, so the goodbye has a row of its own and ends with a real line end. Found in P4's captures,
  on the new editor and on `x/term`'s alike: the goodbye was written on the last prompt's row, after its text, and
  ended by a line feed in raw mode.
- **D4. The keys:** each one `x/term` handles today, with its meaning, plus:
  - ↑/↓ move between the line's rows when it has more than one, and recall history from its first and last row
    (Q1);
  - Ctrl+A/E and Home/End go to the start and end of the line, not of the row (Q1's companion).
  - *Amended 2026-10-21 (0027-PLAN P3, deviation 1; the owner's answer "Also add Ctrl+←/→"):* `ESC b` / `ESC f`
    and Ctrl+←/→ (`ESC[1;5D` / `ESC[1;5C`) also move by word, as Alt+←/→ do. Measured in P3: macOS Terminal sends
    `ESC b` / `ESC f` for Option+←/→, which `x/term` does not read, and Ctrl+←/→ send `ESC[1;5D` / `ESC[1;5C` in every
    terminal that receives them. Every other key `x/term` reads arrives as it reads it, in every terminal measured;
    Terminal's Home and End send nothing, so Ctrl+A/E are its way to the line's ends.
- **D5. What stays as it is:** the key reader (two-stage Ctrl+C, the paste markers, the reader that owns stdin),
  a paste coalesced into one submission with its notice (Q2), backslash continuation, the completion callback, the
  header (0024 D8), history from its file, `/edit`, the `… ` continuation prompt, and Console Host's input.
  *Amended 2026-10-21 (0027-PLAN P5; approved with the owner's answer "Fix, P4, P5, one look"):* the reader that owns stdin (`editor.Reader`) reads only when it is
  asked to: one read of stdin for each `ReadContext` that finds nothing buffered, none ahead. Today it reads ahead
  from `Start`. On Windows, a console read takes the mode in force when it starts and keeps it after the mode changes.
  So a read started before raw mode, or during a turn or a question, stays a line read. It returns only at the
  next Enter, with every key before it, arrows and Ctrl+C included, in one chunk. P4's headless captures under
  Windows Terminal's console host saw this: the first read returned 379 bytes at the first Enter in 1 of 3 runs of
  the new editor and 2 of 6 of `x/term`'s. The line session starts the reader before it enters raw mode
  (`internal/cli/line.go`), because the first-run questions read cooked lines through it first. Reading only when
  asked starts each read in the mode of whoever asked: raw for the editor, cooked for a question. The reader still
  owns stdin for the process's life and never closes it (0008-MADR F20). On macOS and Linux a mode change applies to
  a read already waiting, so nothing changes there but the read-ahead.
- **D6. Tests:** the layout by table; the editor driven by key bytes against a small screen model in the test package
  (cursor moves, `\r`, line feed, erase to end of screen and of line), checked by the screen it ends with; and in a
  pseudo-terminal on macOS and Linux. Windows by the owner's eye in Windows Terminal and Console Host.
  *Amended 2026-10-21 (0027-PLAN P4, deviation 3; the owner's answer "Share the screen model"):* the screen
  model is a test package of its own, `internal/cli/term/termtest`, so the line session's tests in `internal/cli`
  check the screen too; it also takes home, erasing the whole screen, and private modes (bracketed paste).
- **D7. No new module;** `golang.org/x/term` and `golang.org/x/text` are already required.

### Consequences

* Good, because the prompt wraps at the terminal's width, to 120 columns, with whole words, while it is typed and in the
  scrollback.
* Good, because the layout is gobble's, so later wishes (a wider cap, a multi-line buffer) are changes to one function.
* Bad, because gobble now owns the editing and drawing `x/term` gave it: every key, the cursor arithmetic and the
  terminal quirks, on three systems; P3's parity tests carry that.
* Bad, because a redraw after each key writes more to the terminal than `x/term`'s incremental drawing; on a slow
  remote link a long line may flicker. Measured in the PLAN. *(Deviation 4: with only changed text written, typing the
  owner's 337-character report writes about 1,230 bytes, against `x/term`'s 343.)*
* Neutral, because 0008-MADR D6's "thin adapter" becomes an editor of gobble's own, a reversal recorded there.

### Confirmation

0027-PLAN's tests, each seen failing first; the parity table of every key above; the pseudo-terminal captures; the
owner's look on Windows; the gates on macOS, the Linux machine and the Windows laptop.

## Pros and Cons of the Options

### A. gobble's own line editor

* Good, because it is the only option found that keeps words whole while typing.
* Good, because it adds no module and keeps rule 5.
* Bad, because it is the most code, and replaces a tested library's editing.

### B. A third-party line editor

* Good, because editing, history and completion would come ready made.
* Bad, because none found documents word wrap while typing, so it would still need A's layout and drawing inside it.
* Bad, because it adds a module, and 0008 F7 found the candidates stale, large, or frameworks with their own rendering.

### C. Keep `x/term`'s `Terminal`

* Good, because it is no change beyond 0024 D9.
* Bad, because words split at the edge while typing, which the owner's answer rules out.

## Owner questions

* **Q1. ↑ and ↓ in a line of several rows.** A: move between the line's rows, and recall history from its first and
  last row (recommended: as zsh and fish do, so a long line can be edited by row). B: always recall history, as today.
  **Answer (2026-10-21): A, rows, then history.**
* **Q2. A multi-line paste.** A: as today, one submission with its "N pasted lines" notice, shown when sent
  (recommended: no change to what is sent). B: the pasted lines placed in the buffer, editable before sending, which
  makes the buffer multi-line, a larger change. **Answer (2026-10-21): A, as today.**

## More Information

- [0024-MADR-cli-mode-polish.md](0024-MADR-cli-mode-polish.md) D9 (the width, first) and Q9 (the owner's answer).
- [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D6 and F7: the editor this replaces.
- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md) rule 5.
- The plan: [0027-PLAN-native-line-editor-word-wrap.md](0027-PLAN-native-line-editor-word-wrap.md).
