---
status: proposed
date: 2026-10-20
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Polish CLI mode, beginning with `/help`: cooked output for local commands, an aligned list, and informative descriptions

## Context and Problem Statement

The owner reported on 2026-10-20: "when you issue /help the output is splattered all across the terminal as bizarre
spaced wrapped lines. the help output should be a list, aligned to the left margin of the terminal, with informative
help messages/descriptions for each. we need to greatly improve and enhance CLI mode. this is the first thing I want
polished and fixed." This record decides `/help` and the bug under it, and is the record later CLI-mode polish amends.
Each fact below was measured that day at `9188f63`.

**The bug: local commands print while the terminal is raw.**

- The line session puts the terminal in raw mode for the editor (`internal/cli/line.go:126`) and takes it out only
  for a turn or the external editor, through `outOfRaw` (`line.go:144-160`). A submission that is a local command is
  handled by `ls.slash` inside raw mode (`line.go:180`); `/help` writes its text with `e.out.Result`
  (`internal/cli/slash.go:79-83`), a plain write (`internal/cli/output.go:27-35`).
- Raw mode turns off the terminal's output processing, so `\n` is no longer written as `\r\n`: the cursor moves down
  one row and keeps its column. Lines printed in cooked mode reach the terminal as `\r\n`; `/help`'s reach it as bare
  `\n`.
- Captured from `gobble` in a pseudo-terminal of 100 columns, with a scratch home and no credential (the bytes after
  `/help`, abridged):

  ```text
  > /help\r\nCommands:\n  /help \xe2\x80\x94 List the commands\n  /exit \xe2\x80\x94 Leave the session\n ...
  ```

  Replayed with xterm's rules (a bare LF keeps the column; text past the last column wraps), the lines start at
  columns 0, 9, 36, 63, 90, 38, 66, 81, 72, 17, 77, 29, …, and four of the first twelve take two rows: the
  "splattered" output the owner saw.
- The same path prints `/new`'s "new session" line (`slash.go:84-88`), the hidden `/gobble`'s text (`:82-83`), the
  "unknown command" error (`:113`) and a session that cannot open (`:80-81`, `:108-110`). The agent's commands
  (`/usage`, `/model`, …) are not affected: they are sent as turns, which leave raw mode first (`line.go:160`, `:186`).

**The list itself** (`slash.go:117-147`): a heading `Commands:`, the five local commands, then `Agent commands:` and
the agent's, each as `  /name hint — description` on one line, with no alignment, no wrapping, and no use of the
terminal's width, which `internal/cli/term` already measures (`term/caps.go:44-56`). The longest entry, `/permissions`,
is one line of 111 characters, its indent included.

**Windows is unmeasured.** `golang.org/x/term`'s `MakeRaw` changes the console's input mode there; whether `/help`
prints the same way on a Windows console is not known, and the PLAN measures it.

**The descriptions** are the agent's (`acpserver/commands.go:28-44`), sent to every ACP client with
`available_commands_update` and printed by the agent's own `/help` (`commands.go:157-169`), and the line session's five
(`slash.go:19-25`). Several say less than the command does, and one is stale:

- `/mode`'s hint is `[default|plan]`; the modes are `default`, `accept-edits`, `plan` and, when allowed, `bypass`
  (0005-PLAN F6-4; `cmdMode`, `commands.go:398-423`).
- `/context` and `/session` run the same handler (`commands.go:112-113`, `cmdContext`), with two different
  descriptions.
- `/fork` with no argument lists the user's messages to fork before (`commands.go:466-480`), which its description
  does not say; `/model` and `/thinking` with no argument also list the choices.
- `/permissions`' hint, `[remove N | allow|ask|deny RULE [session|project|global]]`, is longer than the description.

**What pins them:** `acpserver/testdata/session.golden` holds the advertised descriptions; `internal/cli/ambient_test.go:201-211`
reads `/help`'s output after `Commands:`. No test runs `/help` on a terminal in raw mode.

## Decision Drivers

* The owner's ask: a list, aligned at the left, with informative descriptions; CLI mode polished, `/help` first.
* One rule for the terminal, so the next local command cannot reintroduce the bug.
* Fit any width: no line longer than the terminal, measured when `/help` runs, since a terminal is resized.
* The descriptions stay the agent's, so every ACP client gets the same text (0008-MADR D13's one registry).

## Considered Options

* **A. The terminal is raw only while the editor reads;** every local command runs out of raw mode, as a turn does.
* **B. Translate `\n` to `\r\n`** in what the line session writes while raw.
* **C. Write `\r\n` in `/help`'s text** only.

## Decision Outcome

Chosen option: "A. The terminal is raw only while the editor reads", because it fixes every local command at one
place, keeps output byte-identical between cooked and raw paths, and lets Ctrl+C reach a local command as a signal, as
it does a turn. *Proposed: the owner decides at review.* *(2026-10-20, the owner's answers: Q1 indented under
headings; Q2 D3's table as proposed; Q3 a later amendment.)*

- **D1. Out of raw mode for every local command.** After the editor returns a submission, the line session leaves raw
  mode before it handles it, local command or prompt, and re-enters raw mode before the next read. `/edit`'s external
  editor, already out of raw mode, no longer leaves it a second time.
- **D2. `/help` is an aligned, wrapped list.**
  - Two sections, each a heading at column 0: `Commands`, the line session's own, then `Agent commands`, the agent's
    and its templates. Each entry on its own line, indented two columns (owner question Q1).
  - Two columns: the command with its hint, then the description. The first column is as wide as the widest entry,
    at most 32 columns; an entry wider than that puts its description on the next line, at the description column.
  - A description longer than the room left is wrapped at spaces, its continuation lines at the description column.
    No line is wider than the terminal.
  - The width is the terminal's when `/help` runs (`golang.org/x/term`'s `GetSize` on standard output), else the
    width found at start (`term.Caps`), else 80. Below 40 columns, every description goes on its own line, indented
    four columns.
  - With colour on (`term.Stream.Color`), the command name is bold; nothing else is coloured. With it off, the output
    is plain text. The separator `—` is gone, so ASCII terminals need no fallback.
  - A template's description, which the user writes, is sanitized (`term.Sanitize`) and wrapped like the others.
- **D3. Informative descriptions,** each saying what the command does with no argument and with one; hints a short
  synopsis. Proposed text (owner question Q2):

  | Command | Hint | Description |
  | :--- | :--- | :--- |
  | `/help` | | Show this list of commands |
  | `/exit` | | Leave gobble (Ctrl+D also leaves) |
  | `/quit` | | Leave gobble; the same as /exit |
  | `/edit` | `[text]` | Write the prompt in your editor, starting from text; it is sent when the editor closes |
  | `/new` | | Start a new session in this directory, closing the current one |
  | `/compact` | `[focus]` | Summarize the conversation to free context; focus says what the summary should keep |
  | `/usage` | | Show this session's token counts and cost |
  | `/context` | | Show the session's file, mode, model, message counts and context-window use |
  | `/session` | | The same as /context |
  | `/model` | `[model]` | Show the current model and the ones available, or switch to model |
  | `/thinking` | `[level]` | Show the thinking level and the model's levels, or set level |
  | `/mode` | `[mode]` | Show the session mode and the modes available, or switch to mode |
  | `/plan` | | Enter plan mode: read-only until you approve a plan |
  | `/name` | `[name]` | Show the session's name, or rename it |
  | `/fork` | `[entry]` | List your earlier messages, or start a new session from before entry |
  | `/clone` | | Start a new session from a copy of this one |
  | `/todos` | | Show the agent's todo list for this session |
  | `/personality` | `[name]` | Show the personality, or switch to boring, dry, snarky or bombastic |
  | `/trust` | | Show whether this project is trusted, and choose for its next session |
  | `/permissions` | `[remove N \| allow\|ask\|deny RULE]` | List the rules in force, remove rule N, or add a rule (session, project or global) |

  The line session's five are in `internal/cli/slash.go`; the agent's in `acpserver/commands.go`, so every ACP client
  and the agent's own `/help` get them.
- **D4. Tests,** each seen failing on the current code first:
  - the line session handles `/help`, `/new` and an unknown command with the terminal out of raw mode, through a raw
    function that records its state;
  - the list at widths 100, 60 and 32: every line at most the width, the descriptions of one section starting at one
    column, a too-wide entry's description on its own line, the wrap's continuation indented to the description
    column;
  - the advertised descriptions, through `acpserver/testdata/session.golden`;
  - a pseudo-terminal run on macOS and Linux: after `/help`, no bare `\n`, and every line starting at column 0.
- **D5. Later CLI-mode polish amends this record,** one item at a time, as the owner names them.

### Consequences

* Good, because `/help` reads as a list at any width, and every local command prints with its line ends.
* Good, because the descriptions say what each command does and takes, in every ACP client too.
* Neutral, because Ctrl+C during a local command is now a signal, as during a turn; the local commands are quick.
* Bad, because the ACP golden and the help test change with the text, and an ACP client that cached the old
  descriptions shows them until it reads the next `available_commands_update`.

### Confirmation

The PLAN's tests (D4), each seen failing first; the pseudo-terminal capture and its replay showing every line at
column 0; the gates on macOS, the Linux machine and the Windows laptop.

## Pros and Cons of the Options

### A. Raw only while the editor reads

* Good, because one place fixes every local command, today's and later ones.
* Good, because a local command then behaves as a turn does: line ends, Ctrl+C.
* Bad, because it touches the line session's loop, which `/edit` also uses; the PLAN keeps `/edit`'s test.

### B. Translate line ends while raw

* Good, because the loop is unchanged.
* Bad, because every writer must go through the translator, and a writer that does not brings the bug back.
* Bad, because Ctrl+C during a local command stays a key, not a signal.

### C. `\r\n` in `/help` only

* Good, because it is the smallest change.
* Bad, because `/new`, `/gobble` and the error lines keep the bug, and so does every later local command.

## Owner questions

* **Q1. The entries' indent.** Offered: two columns under a heading at column 0 (recommended: the headings then read
  as sections); or every line, headings and entries, at column 0. **Answer: indented under headings.**
* **Q2. The descriptions.** D3's table as proposed (recommended), or as the owner edits it. **Answer: as
  proposed.**
* **Q3. `/help <command>`,** a longer help for one command with its arguments and an example. Offered: a later
  amendment of this record (recommended, so `/help` is fixed first); or in this change. **Answer: a later
  amendment.**

## More Information

- [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D13: the line session's one registry of slash commands.
- [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md) F6-4: the modes.
- The plan: [0024-PLAN-cli-mode-polish.md](0024-PLAN-cli-mode-polish.md).
