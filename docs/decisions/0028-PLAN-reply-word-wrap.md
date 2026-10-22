---
status: in-progress
date: 2026-10-22
associated-madr: "0028-MADR-reply-word-wrap.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement word wrap for the agent's replies and gobble's own messages

Associated MADR: [0028-MADR-reply-word-wrap.md](0028-MADR-reply-word-wrap.md)

## Goal

On a terminal, the line session's replies, thinking text and messages wrap at spaces at the terminal's width, to 120
columns (Q1), while they stream, and are never cut inside a word. Code, tables, headings and tool output are left
exactly as they are. Off a terminal, and in print mode (Q2), nothing changes (MADR D1–D8).

## Scope

- **In:**
  - `internal/cli/render`: a new streaming wrapper, its tests, and the shared helpers the wrapper and `Style` need
    (fence and marker detection);
  - `internal/cli/view.go`: the wrapper between `render.Stream` and `render.Tables`/`render.Style`, and for thinking
    text;
  - `internal/cli/output.go`: `Output.diag`'s messages wrapped on a terminal;
  - their tests;
  - 0008-MADR D7's dated note; `docs/architecture.md`; `docs/README.md`.
- **Out:**
  - print mode, unless the owner answers Q2 B;
  - the TUI mode;
  - tool output, diffs, plans and the status line;
  - reflowing text already written when the window is resized.

Starts after 0027-PLAN P5 is committed.

## Implementation Steps

### P1 — the wrapper (D3, D4, D5, D7)

1. **Tests first,** a table of input, width and wanted output:
   - prose broken at spaces, no row wider than the width;
   - `comma,` and `end.` kept whole;
   - a word wider than the row alone on its row, unbroken;
   - wide characters counted as two columns;
   - list items, unordered and ordered, continued under their text;
   - block quotes, nested, continued with their markers;
   - no row starting with `#`, `-`, `*`, `+`, `>`, `|`, a fence, or a number followed by `.` or `)`;
   - fenced code (backticks and tildes, longer runs, an unclosed fence), indented code, tables and headings passed
     through unchanged;
   - blank lines and hard line ends kept.

   Then a streaming property: for every reply in the table, every split into two chunks, and random splits into many,
   the chunks fed in turn and then flushed give the same output as the whole reply fed at once.
2. **`render.Wrap`:**
   - a type with `Push(text, width) string` and `Flush() string`, taking what `render.Stream` releases;
   - it keeps the row's column, the fence state, the list or quote prefix, and the word still arriving;
   - the width is passed to each `Push`, so the view can read it live (D2).

   The fence and marker detection `Style` uses moves to shared helpers, so the two cannot disagree. Unused by the view
   yet: no visible change.

### P2 — the view (D1, D2)

1. **Tests first,** on `termtest`'s screen, through `view.on` with a stub output:
   - the owner's reply sentence streamed in small chunks at 60, 100 and 177 columns (capped at 120): no word cut, no
     row over the width;
   - a reply with a code block, a table and a list: the code and the table byte for byte, the list continued under
     its text;
   - a tool call between two paragraphs: the held word flushed before the tool's header;
   - thinking text wrapped;
   - stdout not a terminal: the bytes unchanged from today.
2. **The view** wraps released text with the width read at each `Push`, when `caps.Out.TTY`, before `Tables` and
   `Style`. `flush` flushes the wrapper. Thinking text goes through its own wrapper.

### P3 — gobble's messages (D6)

1. **Tests first:** a long `Warnf` and an error at 60 and 100 columns on a terminal: broken at spaces, continuation
   rows indented two columns, a long word whole. Off a terminal: one line, as today.
2. **`Output.diag`** wraps when `caps.Err.TTY`, at stderr's width, read live, to 120 columns.

### P4 — on real terminals, and the records

1. **Captures:**
   - scratch drivers outside the tree; gobble built from the working tree; a scratch home;
   - a scripted agent reply, so no prompt reaches a model: prose, a list, a quote, a code block, a table and a long
     URL;
   - a pseudo-terminal at 60, 100 and 200 columns, on macOS and the Linux machine;
   - the Windows laptop headless, under Console Host and Windows Terminal's console host;
   - each screen drawn on `termtest` and checked: no word cut, no row over the width, code and table unchanged.

   If no scripted reply can be reached from the binary without test hooks, that is a deviation, raised before
   anything is added.
2. **The owner's look** in WezTerm on macOS and in Windows Terminal.
3. **Records:** 0008-MADR D7's dated note; `docs/architecture.md`'s `internal/cli/render` row; `docs/README.md`.

## Verification

- Every new test seen failing first, on the unchanged code or on a scratch copy, its failure quoted in the record.
- Negatives on a scratch copy, one mutation at a time:
  - a row allowed past the width;
  - a break inside a word;
  - a block-opening word at a row's start;
  - a fence wrapped;
  - the held word not flushed;
  - the width not capped;
  - wrapping off a terminal.
- Each phase:
  - the pre-add check on its changed `.go` files;
  - `go test -count=1 ./...`, `CGO_ENABLED=1 go test -race -count=1 ./...` and `make preflight` on macOS, the Linux
    machine and the Windows laptop, the remote two on a copy of the tree;
  - `check-records` and `markdownlint`.

## Rollout and Rollback

One commit per phase. P1 changes nothing a user sees. P2 and P3 turn wrapping on for replies and messages on a
terminal; reverting either restores today's output. Nothing is stored, and no flag or setting is added.

## Execution record

### P1 execution record (2026-10-22)

Approved by the owner's "proceed", 2026-10-22, after 0027-PLAN P5's commit `6361bd9`.

**Deviation 1, 2026-10-22, decided by the owner** ("Hold one more word"). Before any code was written, MADR D4 (no row
starts with a word that would open a block, the break moving one word earlier) proved impossible under D5 (only the
word still arriving held back): by the time a block-opening word arrives, the word before it has been written. The
wrapper also holds the last finished word on the current row until the next word is known. 0028-MADR D5 is amended.
Steps 1 and 2 are unchanged.

1. **Tests first** (`internal/cli/render/wrap_test.go`), against a stub `Wrap` that passed text through:
   - `TestWrap`: 29 cases at widths 18 and 20, one of them with no width; 30 after the negatives below added one.
   - `TestWrapAnySplit`: every split of each case in two, and 50 random splits into chunks of 1 to 6 runes.
   - `TestWrapStreams`: what each push releases.

   On the stub, the 19 cases that need a break failed (`got "The quick brown fox jumps over the lazy dog.\n"`), as
   did the any-split and streaming tests. The ten pass-through cases passed, since they ask for the text unchanged.
   One expectation of `TestWrapStreams` was corrected before the code was written: by deviation 1, `"- one two"`
   releases `"- "` and holds `one`.
2. **`render.Wrap`** (`internal/cli/render/wrap.go`):
   - `Push(text, width)` and `Flush()`.
   - A line's kind is decided from its start, once enough of it has arrived, by `lineKind` and a helper for each kind
     of start. A fence is decided only at its line's end.
   - Prose words are placed by `endWord`, holding the last finished word (deviation 1).
   - Fence, heading and marker detection call the package's own `fenceRun`, `isFenceClose`, `isHeading` and
     `markerLen`, which `Style` uses. They were already package-level, so the step's "moves to shared helpers" needed
     no move.
   - D4's guard also covers thematic breaks and setext underlines: a word made only of `-`, `*`, `_`, `=` or `+`.
     They open a block as the markers listed in D4 do.
   - A fence inside a quote passes its own line only, so a quoted fence cannot leave the rest of a reply unwrapped.
   - Unused by the view: no visible change.

   The first run printed nothing: resetting the line's state by assigning the whole struct also emptied the output
   builder. It was fixed to reset the line's fields only.

**Found by the gates and fixed in the phase:** `gocognit` on `lineKind` (51 > 50), split into `headingKind`,
`fenceKind`, `bulletKind` and `orderedKind`, with the same behaviour.

**Negatives,** on a scratch copy, one mutation at a time:

- a row allowed 5 columns past the width;
- a break inside a word;
- a block-opening word allowed at a row's start;
- a fence wrapped as prose;
- the held word not flushed;
- a quote's marker not repeated;
- a list item's rows not indented;
- widths counted in bytes;
- a heading wrapped;
- a table row wrapped;
- a fence decided before its line ends.

The last passed at first: no case had a backtick after the opening run. The case "backticks closed on their line are
inline code, not a fence" was added and fails under it. Eleven mutations, eleven failures; the restored copy passed.

**Gates:**

- The pre-add check on the two new `.go` files: `2 file(s) clean (gofmt, golint, govulncheck)`.
- macOS: `make preflight`, `preflight passed`; the tests and the race detector, 36 packages ok each.
- The Linux machine and the Windows laptop, on a copy of the tree: the tests and the race detector, 36 packages ok on
  each. `make preflight` reached `apidiff` (golangci-lint `0 issues.`, govulncheck "No vulnerabilities found.") and
  stopped at `verify-build-metadata`, as the copies are not git checkouts.
- `make check-records markdownlint`: "0 issues in 0 files".
- The Windows laptop, `make probe-conhost`: `--- PASS: TestConhost`.

**P1 complete, 2026-10-22.**
