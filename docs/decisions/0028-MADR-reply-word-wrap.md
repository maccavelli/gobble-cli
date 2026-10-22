---
status: accepted
date: 2026-10-22
decision-makers: repository owner
consulted: the owner's WezTerm session of 0027-PLAN P5's test build (read with `wezterm cli get-text`); goose's streaming Markdown (0008-MADR F9)
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Word-wrap the agent's replies and gobble's own messages in the native CLI mode

## Context and Problem Statement

0027-MADR made the prompt wrap whole words at the terminal's width, to 120 columns. Trying 0027-PLAN P5's build in
WezTerm on macOS, the owner reported: "it still cut the text off mid-word". The pane, 177 columns wide, read with
`wezterm cli get-text` on 2026-10-21, showed the prompts whole. The cuts were in text printed after a turn:

- the model's reply: `I'll be precise, c` / `ourteous, and to the point.`;
- an error from the turn: `… llmprovider: kilo failed after 4 attempts: llmprovider: ` /
  `provider unavailable: …`.

0027 left replies out of scope. Nothing in gobble wraps them:

- **Replies.** `view.on` passes each text update through `render.Stream` (`internal/cli/render/mdstream.go`). That
  stream releases Markdown once later chunks cannot change its meaning, sometimes in the middle of a line. With
  colour on, `render.Tables` and `render.Style` colour the released text in place, keeping every marker. The view then
  writes it to stdout (`internal/cli/view.go:110`, `:241-251`). The terminal wraps each line at its last column,
  wherever that falls.
- **gobble's own messages.** `Output.diag` writes `Notef`, `Warnf` and errors to stderr as one line each
  (`internal/cli/output.go:59`), and the terminal wraps them the same way.

This is what 0008-MADR D7 chose. Its finding F9 reads: "colouring Markdown source in place, from a stable-prefix
buffer, gives clean streaming with no layout engine and no reflow. Copying the output still yields valid Markdown."
Wrapping replies reverses the "no reflow" half of D7. The other half still holds: Markdown allows a paragraph to break
across lines, so prose broken at spaces is the same Markdown.

**What gobble already has:**

- word wrap to a width: `render.wrapIndented` (`internal/cli/render/commands.go:118`) and the editor's layout
  (`internal/cli/editor/layout.go`);
- display widths of wide characters: `golang.org/x/text/width`;
- the terminal's width, read live: `lineSession.width` (`internal/cli/slash.go:152`);
- the fence, heading and marker rules `render.Style` already applies (`internal/cli/render/mdstyle.go`).

Rule 5 of 0004-MADR keeps Charm and go-tui-lib out of `internal/cli`, as 0027 found.

**Print mode** (`gobble -p`, `internal/cli/print.go`) has its own path and writes the reply raw, as Pi's print mode
does (0008-MADR F9). It is a machine-facing contract, so it is out of scope unless the owner says otherwise (Q2).

## Decision Drivers

* The owner's requirement, carried over from the prompt: text gobble shows on a terminal is never cut inside a word.
* Streaming stays streaming: text shows as it arrives, not paragraph by paragraph.
* What 0008 D7 protects, minus the reflow: copied prose is still valid Markdown with the same meaning. Code blocks,
  tables, headings and tool output are copied unchanged.
* Off a terminal nothing changes: pipes, files and print mode get the bytes they get today.
* No new module; rule 5 keeps Charm out of `internal/cli`.
* Testable without a terminal: the wrapper a pure, streaming function, the view checked on `termtest`'s screen.

## Considered Options

* **A. Wrap prose as it streams.** A streaming wrapper in `internal/cli/render` sits between `render.Stream` and
  `render.Style`. It breaks prose at spaces, at the terminal's width read when each break is decided, and holds back
  only the word still arriving. Code fences, tables, headings and indented code pass through unwrapped. On stderr,
  gobble's messages are wrapped the same way, with continuation rows indented under the message.
* **B. Wrap each paragraph once it is complete:** buffer prose until a blank line or a block, then wrap it whole.
* **C. Leave wrapping to the terminal,** as 0008 D7 has it.
* **D. A third-party wrapper or Markdown renderer**, adopted with a MADR naming its module.

## Decision Outcome

Chosen option: "A. Wrap prose as it streams", proposed, because it is the only option that keeps words whole and the
reply streaming without adding a module. It keeps copied prose valid Markdown, and the code, tables and tool output
that must stay exact are left alone.

- **D1. Where.** Only where the line session writes to a terminal: the reply text and thinking text of `view`, when
  stdout is a terminal, and `Output.diag`'s messages, when stderr is a terminal. Pipes, files and print mode are
  unchanged (Q2).
- **D2. The width:** the terminal's, read when each break is decided, at most 120 columns, the prompt's rule
  (0027-MADR D2; Q1).
- **D3. What wraps.** Prose wraps: paragraphs, list items, block quotes and thinking text. These do not: lines inside
  a code fence, indented code (four spaces or a tab), table rows, headings, tool output, diffs, plans and the status
  line. A row ends at a space. A word wider than the row goes on a row of its own, unbroken, so a long URL or path is
  never cut; the terminal wraps it as it does today.
- **D4. Continuation rows.**
  - A list item's continuation rows are indented to its text, under the marker.
  - A block quote's continuation rows repeat its `>` markers.
  - A paragraph's continuation rows start at column 0, as the paragraph does.
  - A row never begins with a word that would open a block in Markdown (`#`, `-`, `*`, `+`, `>`, `|`, a fence, or a
    number followed by `.` or `)`). The break moves one word earlier, so the copied text means what it meant.
- **D5. Streaming.** The wrapper keeps the current row's column and kind across releases. It holds back only the last
  word when no space or line end has followed it yet, and releases it when one comes. `view.flush`, which runs before
  every non-text event and at the turn's end, flushes it too. Feeding a reply in any chunks gives the same output as
  feeding it whole.
  *Amended 2026-10-22 (0028-PLAN P1, deviation 1; the owner's answer "Hold one more word"):* D4's rule cannot be kept
  if only the word still arriving is held. When a word that would open a block must start a row, the word before it
  has already been written. So the wrapper also holds the last finished word on the current row until the next word
  shows whether the break must move. Streamed prose shows up to one word later than it arrives. The flushes are
  unchanged.
- **D6. gobble's messages** (`Notef`, `Warnf`, errors) wrap at stderr's width, to 120 columns, at spaces. Continuation
  rows are indented two columns, and a word wider than the row is left whole, as in D3.
- **D7. Order.** The wrapper runs on the text `render.Stream` releases, before `render.Tables` and `render.Style`.
  So it counts no escape sequences, and `Style` sees the line starts the wrapper made, none of which opens a block
  (D4).
  *Amended 2026-10-22 (0028-PLAN P2, deviation 2; the owner's answer "Wrap after Style"):* `Style` colours an
  emphasis or code span only when it sees the whole span within one call and one line. `render.Stream` releases
  whole spans, but the wrapper's held words (D5, as amended) can split one, as can a row break. For example,
  `**very bold** and` would be released as `**very`, with `bold** and` held. So the view styles each release first,
  as it did before 0028, and the wrapper runs on the styled text:
  - escape sequences count as zero width and travel with the word they are in;
  - a row break drops only spaces, never an escape;
  - a line's kind and a word's opening a block are read from the text with escapes removed.
- **D8. No new module:** `golang.org/x/text/width` is already required.

### Consequences

* Good, because replies and messages keep their words whole at the terminal's width, as the prompt does.
* Good, because copied prose is still valid Markdown with the same meaning, and code, tables and tool output are
  copied unchanged.
* Bad, because the breaks are fixed when written. Widening the window does not rejoin them, and narrowing it makes the
  terminal wrap the wrapped rows again, raggedly. Terminals reflow only their own soft wraps.
* Bad, because a copied paragraph has line breaks where it was wrapped. Markdown renders it the same, but plain-text
  destinations show the breaks.
* Bad, because the last word of a release can wait for the next chunk: a few characters, briefly.
* Neutral, because 0008-MADR D7's "no reflow" is reversed for prose on a terminal, recorded there as a dated note.

### Confirmation

0028-PLAN's tests, each seen failing first: the wrapper by table, and by every split of a reply into chunks; the view
on `termtest`'s screen; the messages on stderr. Then pseudo-terminal captures on macOS and the Linux machine, the
Windows laptop headless, and the owner's look in WezTerm and Windows Terminal. The gates run on all three hosts.

## Pros and Cons of the Options

### A. Wrap prose as it streams

* Good, because the reply streams as it does today and its words stay whole.
* Good, because it reuses what gobble has (the widths, the fence and marker rules) and adds no module.
* Bad, because the wrapper must carry state across releases (the row's column, the fence, the list's indent, the word
  still arriving), the most code of the options.

### B. Wrap each paragraph once it is complete

* Good, because each paragraph is wrapped whole, with no state between chunks.
* Bad, because a reply shows paragraph by paragraph, and a long paragraph shows late, which goes against the
  streaming 0008 D7 chose.

### C. Leave wrapping to the terminal

* Good, because it is no change. The terminal reflows its soft wraps on resize, and copied text has no breaks.
* Bad, because words are cut at the edge, which the owner's report rules out.

### D. A third-party wrapper or Markdown renderer

* Good, because ANSI-aware word wrap exists ready-made.
* Bad, because it adds a module. A full Markdown renderer would also replace 0008 D7's source colouring with its own
  rendering, and the Charm ones are excluded from `internal/cli` by rule 5.
* Bad, because none found handles a stream that is released mid-line with block-aware continuation rows. This was
  judged from documentation, not measured.

## Owner questions

* **Q1. The width.** A: the terminal's width, at most 120 columns, the prompt's rule (recommended: one rule for
  everything gobble draws, and long rows are harder to read). B: the terminal's full width, however wide.
  **Answer (2026-10-21): A, at most 120 columns.**
* **Q2. Print mode on a terminal** (`gobble -p` with stdout a terminal). A: unchanged, raw (recommended: print mode is
  a machine-facing contract, and the output is often piped). B: wrapped as in the line session when stdout is a
  terminal. **Answer (2026-10-21): A, print mode unchanged.**

## More Information

- [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) F9 and D7: Markdown coloured in place, no reflow.
- [0027-MADR-native-line-editor-word-wrap.md](0027-MADR-native-line-editor-word-wrap.md) D2: the prompt's width.
- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md) rule 5.
- The plan: [0028-PLAN-reply-word-wrap.md](0028-PLAN-reply-word-wrap.md).
