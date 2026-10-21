---
status: accepted
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
it does a turn. *Proposed: the owner decides at review.* *(2026-10-20: accepted by the owner, "accepted, approved to
proceed", after `6ab7928`.)* *(2026-10-20, the owner's answers: Q1 indented under
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
  | `/personality` | ~~`[name]`~~ `[boring\|dry\|snarky\|bombastic]` | ~~Show the personality, or switch to boring, dry, snarky or bombastic~~ List the personalities, or switch to one |
  | `/trust` | | Show whether this project is trusted, and choose for its next session |
  | `/permissions` | `[remove N \| allow\|ask\|deny RULE]` | List the rules in force, remove rule N, or add a rule (session, project or global) |

  The line session's five are in `internal/cli/slash.go`; the agent's in `acpserver/commands.go`, so every ACP client
  and the agent's own `/help` get them. *(2026-10-20, 0024-PLAN deviation 2, the owner's choice: `/personality`'s hint
  stays the one [0022-MADR-gobble-voice-and-personalities.md](0022-MADR-gobble-voice-and-personalities.md) decided,
  naming the built-ins, and its description no longer repeats them.)*
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

## Amendment (2026-10-20): a log record printed into the conversation, and the home directory it named

*Proposed: the owner decides at review.* *(2026-10-20: accepted by the owner, "committed, proceed", after
`381f381`, with Q4 answered A; executed the same day.)* The owner asked on 2026-10-20, after P1's run, to "add the 2 additional
found items to scope": the two findings of 0024-PLAN's P1 record. Each fact below was measured that day on the working
tree, P1's change applied and not yet committed.

### Context

**Finding 1: a log record in slog's text form, printed into the conversation.** On the Linux machine, the session
that `/help` opened printed, before the list:

```text
time=2026-10-20T15:31:11.659Z level=WARN msg="prompt: a file was not used" file=<home>/.agents/skills reason="not loaded: the project is not trusted (gobble trust, or /trust)"
```

- It is the agent's note on a skills directory an untrusted project holds back
  (`acpserver/sysprompt.go:208-225`, `note`), which 0005-PLAN F6-1 decision 8 asks for: "an untrusted project's are
  held back, each with a note".
- It reaches the terminal because every log record at warning level or above is written to stderr by
  `slog.NewTextHandler` (`internal/logging/logging.go:40-43`), and the line session passes its stderr in
  (`internal/cli/main.go:81`, `:222`). That is the owner's decision of 2026-10-05 in 0004-PLAN Phase 2 step 4
  (`0004-PLAN-go-module-architecture.md:808`): "a text handler at `max(Level, slog.LevelWarn)` … The terminal shows
  warnings and errors only".
- 0008-MADR D3 has every diagnostic on stderr "prefixed `Error:` or `Warning:`"; D18 keeps user output and logging
  apart. A log record on a terminal is the one diagnostic in neither form: a timestamp, `key=value` pairs and Go
  quoting, 178 columns here, never wrapped.
- The log file keeps every record as JSON (`logging.go:32-38`), whatever stderr shows.
- **Not measured:** a record logged while the editor reads, with the terminal raw, would reach it with bare line
  ends (MADR D1's bug, for a write the loop does not order). No such record was seen; the title and other background
  work log from the agent's goroutines.

**Finding 2: "its path was the account's real home": not a defect.** P1's record said the warned directory was the
account's own `.agents/skills` although the run set `HOME` to a scratch directory, and that "that directory's lookup
may not follow `HOME`". It does follow it:

- `ProjectSkillDirs` (`prompt/trust.go:12-40`) walks up from the working directory to the repository root, or to the
  filesystem root outside a repository, collecting each `.agents/skills`, and skips only the one in `$HOME` itself
  (`:30`); the agent passes `os.UserHomeDir()`, which reads `$HOME` (`acpserver/sysprompt.go:122-127`).
- The capture ran with `HOME` and the working directory both a scratch directory below the account's home, outside any
  repository, so the walk reached the account's home, whose `.agents/skills` was then a project's, held back as
  untrusted.
- A probe in a synthetic tree with no repository (`fakehome/.agents/skills/demo/SKILL.md`, the working directory
  `fakehome/proj`): with `HOME=fakehome`, no warning about `fakehome/.agents/skills`; with `HOME=fakehome/proj`, the
  warning. `prompt/trust_test.go:13-32` tests the exclusion inside a repository, where the walk stops at the root
  first; nothing tests it outside one.

### Decision

- **D6. On a terminal, a log record on stderr reads as gobble's own diagnostic** (owner question Q4).
  - When stderr is a terminal, the stderr handler writes a record at warning level or above as one diagnostic:
    `Warning:` or `Error:`, the message, then its attributes in parentheses as `key: value`, separated by `; `. No
    time, no level word, no quoting. A path under `$HOME` is shown with `~`. The diagnostic is wrapped at spaces to
    stderr's width, less one column, its continuation lines indented two columns.
  - The record above becomes: `Warning: prompt: a file was not used (file: ~/.agents/skills; reason: not loaded: the
    project is not trusted (gobble trust, or /trust))`.
  - Redaction applies as before (`ReplaceAttr`), and the level stays `max(Level, LevelWarn)`.
  - When stderr is not a terminal (a script, a pipe, CI), the handler stays slog's text, unchanged; under `gobble
    acp` there is still no stderr handler. The log file is unchanged.
  - It amends the owner's decision in 0004-PLAN Phase 2 step 4 for a terminal only, with a dated note there.
- **D7. Finding 2 is closed as a test, not a fix:** a test that, outside any repository, the walk skips `$HOME`'s own
  `.agents/skills` and keeps an ancestor's, seen failing on a scratch copy without the `$HOME` check. P1's record
  gets a dated correction of its finding.
- **Out of scope, named:** the bare line ends of a record logged while the editor reads (finding 1's last point),
  unmeasured; it waits for a later item, with a measurement first.

### Options for D6

* **A. A diagnostic on a terminal, slog text elsewhere** (recommended): a person reads `Warning:` lines; a script or
  CI log keeps parseable records; the file holds everything.
* **B. No log records on a terminal at all,** the file only: the conversation stays clean, but a warning a user should
  act on (an untrusted project's skills, a rotation failure) is then seen by no one.
* **C. Unchanged:** slog's text on every stderr, as today.

### Owner questions

* **Q4. Finding 1's form.** A, B or C above; A recommended. **Answer (2026-10-20): A, a `Warning:` line.**

## Amendment (2026-10-21): the workspace's name and branch above the prompt

*Proposed: the owner decides at review.* *(2026-10-21: accepted by the owner, "accepted, proceed".)* The owner asked on 2026-10-21, after `eb8f155`: "in cli mode, in the default
session prompt, show the project/workspace/repo name above the prompt", a CLI-mode polish item under D5. The owner's
answers to three questions of the same day are recorded as Q5–Q7 below. *(Revised twice the same day, before
review: the owner asked to "change it to sit below the prompt, not above it. it will keep the session chat cleaner that
way", and a draft for a line below the prompt that erased itself was written; the owner then returned to the first
design: "put it above the prompt as decided before" (Q8). This text is the first design's again.)* Each fact below was
measured that day at `eb8f155`.

### Context

- **The prompt is `x/term`'s.** The line session's editor (`internal/cli/editor/editor.go`) draws `> `, or `… ` on a
  continuation line, through `xterm.Terminal.SetPrompt` (`:75`) before each `ReadLine`. The terminal keeps the prompt
  on one line and measures it to place the cursor, so a prompt holding a line break would misplace it; a line above
  the prompt is written before it, not in it. The editor runs in raw mode, so that line ends in `\r\n`, as its
  notices do (`:127-132`).
- **The loop.** `internal/cli/line.go:184` calls `ed.ReadPrompt` once per submission, after each turn or local
  command, with the session's directory in `ls.cwd`.
- **Finding the repository.** `fsx.GitRepo` (`internal/fsx/git.go:15`) walks up from a directory for `.git`, a directory
  or a worktree's `gitdir:` file, as Pi's `findGitPaths` does, and gives the repository's directory and its common git
  directory; prompt's context files and the project settings use it. A worktree's `HEAD` is in its own git directory,
  which `GitRepo` reads but does not return.
- **What Pi shows.** Pi's footer (`packages/coding-agent/src/core/footer-data-provider.ts`, at Pi `1402973a9`) reads
  `HEAD`: `ref: refs/heads/<name>` gives the branch, anything else "detached"; only a reftable repository, whose
  `HEAD` reads `refs/heads/.invalid`, makes it run `git symbolic-ref`.

### Decision

- **D8. Above each new prompt, the workspace's name and branch** (Q5–Q7).
  - **The name** is the top-level directory's name of the git repository the session's directory is in, found as
    `fsx.GitRepo` finds it; outside a repository, the session's directory's own name.
  - **The branch,** in parentheses after it, is read from the repository's `HEAD`, a worktree's own: `ref:
    refs/heads/<name>` gives `<name>`, slashes kept; a commit id gives `detached`, as Pi's does. No `git` command is
    run, so a reftable repository's `.invalid` shows no branch. Outside a repository there is none:
    `gobble-cli (main)`, `gobble-cli (detached)`, `scratch`.
  - **When:** on a line of its own before each new prompt: at the first prompt and after each turn or local command;
    never before a continuation line. It is read again each time, so a branch the session switched shows at the next
    prompt.
  - **How:** the line as it is, no colour (Q7's preview); a control character in a name or a branch is dropped, as
    gobble's other terminal text is sanitized.
  - **Where:** the line session only. Print mode and `gobble acp` print no prompt.
  - **Built as:** `fsx.GitBranch(dir)`, `HEAD` read through `GitRepo`'s walk, its worktree case included; a `Header`
    on `editor.Options`, written before a fresh prompt; and the line session's header from `ls.cwd`.

### Consequences

* Good, because the user sees which repository and branch the session works in before each prompt, as Pi's footer
  shows them.
* Neutral, because each prompt reads `.git` and `HEAD`: a walk up the directories and one small file, no process.
* Bad, because a reftable repository shows its name with no branch, where Pi asks `git`.

### Options not chosen

* **Below the prompt, erasing itself** (the second draft, Q8): drawn on the row below the prompt while the user types,
  and erased before Enter, so the conversation's scrollback never holds it; also erased, until the next prompt, when
  the input wraps or the user recalls history or pastes. Bad, because it needs the editor's key reader to erase it
  before `x/term` handles each such key, with cursor save and restore new to the editor, for a line that vanishes
  while the user types long input.
* **The name with the path below the repository's top,** or **the directory's name alone** (Q5).
* **Once, at session start** (Q6).
* **The name alone, dimmed,** or **a labelled rule** (Q7).
* **Running `git` for a reftable `HEAD`,** as Pi does: one process per prompt in those repositories, for a branch the
  file does not name; not taken, the option chosen naming no `git` command.

### Owner questions

* **Q5. Which name.** **Answer (2026-10-21): the repository's name, else the directory's.**
* **Q6. When.** **Answer (2026-10-21): above every new prompt.**
* **Q7. How it looks.** **Answer (2026-10-21): the name with the git branch,** `gobble-cli (main)`.
* **Q8. Above or below.** The owner asked for below the prompt, then returned to above. **Answer (2026-10-21): above
  the prompt,** "as decided before".

## Amendment (2026-10-21): the prompt's width follows the terminal, and an entered prompt is word-wrapped

*Proposed: the owner decides at review.* *(2026-10-21: accepted by the owner, "Both, P4 then 0027".)* The owner reported on 2026-10-21, after `9dcd596`: "The prompt is being cut
and wrapped at around 80 chars, including in the middle of words. My terminal is much wider than 80 chars. The line
wrap needs to be dynamic, based on the available width of the user's terminal to a max of 120 chars, ensuring not to
cut words or punctuation such as a trailing comma when wrapping the prompt text." A CLI-mode polish item under D5.
Each fact was measured that day at `9dcd596`.

### Context

- **`x/term` assumes 80 columns.** `xterm.NewTerminal` starts with `termWidth: 80` (`golang.org/x/term` `v0.47.0`,
  `terminal.go:139`) until `SetSize` is called, and gobble never calls it: no `SetSize` in `internal/cli`. The terminal's
  real width is known (`internal/cli/term/caps.go:49`, `xterm.GetSize`) and used for gobble's own output, such as
  `/help` (`internal/cli/slash.go:154`), but not given to the line editor.
- **It breaks the line itself.** When the cursor reaches `termWidth`, `x/term` writes `\r\n` (`terminal.go:381-401`),
  so on any terminal the input breaks at column 80, wherever that falls in a word. P6's recordings show it: `> Fix
  the off-by-one … tell me whether the wh` / `ole test suite will pass …`. Its cursor arithmetic for editing a long
  line (`moveCursorToPos`, `terminal.go:286-316`) uses the same 80.
- **It wraps by character only.** `x/term` places each character at the next column and row; it has no notion of
  words, so a line editor built on it cannot keep a word whole while the user types without replacing its rendering.
- **gobble already wraps words.** `render.wrapIndented` (`internal/cli/render/commands.go:118`) fills lines to a width,
  breaking only at spaces, with wide characters measured (`displayWidth`); a word wider than a line is cut. `/help`
  uses it (D2).
- **Where an entered line sits.** After Enter, `x/term` has written the prompt and the line, then `\r\n`
  (`terminal.go:582-591`), at a width it knows, so the rows the line took are known when `ReadLine` returns.

### Decision

*(Revised the same day, before review, for Q9's answer: whole words while typing, which needs gobble's own line editor
in place of `x/term`'s rendering; that is
[0027-MADR-native-line-editor-word-wrap.md](0027-MADR-native-line-editor-word-wrap.md). The first draft also redrew an
entered prompt word-wrapped; 0027's editor makes that unneeded, so this amendment keeps the width alone, which ships
first.)*

- **D9. The prompt's width is the terminal's, to 120 columns** (Q9).
  - Before each prompt, the editor reads the terminal's width and gives `x/term` the smaller of it and 120
    (`SetSize`); 80 when the width is unknown. A resize between prompts takes effect at the next one. The input then
    breaks at the true width, not at column 80, and cursor movement in a long line uses the true width.
  - Words may still split at the right edge, by character, as `x/term` draws; keeping them whole is 0027's.

### Consequences

* Good, because the prompt uses the terminal's width, to 120 columns, at once, before 0027's editor.
* Good, because editing a long line moves the cursor correctly on a terminal wider or narrower than 80 columns.
* Neutral, because words still split at the right edge until 0027's editor replaces `x/term`'s rendering.

### Options not chosen

* **Redraw an entered prompt word-wrapped** (this amendment's first draft): words whole in the scrollback, split while
  typing; set aside by Q9's answer, which keeps them whole while typing too.
* **No cap:** the terminal's full width; the owner's 120 rules it out.

### Owner questions

* **Q9. Words while typing, or at Enter.** A: the width now, words kept whole once the prompt is entered. B: words kept
  whole while typing too, with gobble's own line editor. **Answer (2026-10-21): B,** "Whole words while typing";
  0027-MADR is that editor, and the width here ships first.
