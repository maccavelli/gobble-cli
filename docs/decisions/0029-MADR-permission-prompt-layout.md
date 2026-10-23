---
status: accepted
date: 2026-10-22
decision-makers: repository owner
consulted: the owner's report of a permission prompt on 2026-10-22; `acpserver/question.go`, `acpserver/prompt.go`, `internal/cli/permission.go`
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Lay the native CLI mode's permission prompt out as a grouped menu, with the full command and its patterns once

## Context and Problem Statement

On 2026-10-22 the owner called the permission prompts "hideous", with this example, one line on the terminal:

```text
run echo '=== staged ==='; git diff --cached --stat; echo '…?  [1] Allow once  [2] Allow `echo *`, `git diff *`, `git log *` for this session  [3] Allow `echo *`, `git diff *`, `git log *` for this project  [4] Allow `echo *`, `git diff *`, `git log *` everywhere  [5] Reject  [6] Reject `echo *`, `git diff *`, `git log *` for this session  [7] Reject `echo *`, `git diff *`, `git log *` for this project  [8] Reject `echo *`, `git diff *`, `git log *` everywhere  [f] deny and say why  [c] cancel
```

Three things make it so:

- **The command is cut.** gobble's agent cuts every permission request's title to 60 runes, on one line
  (`maxTitle`, `cutTitle`, `acpserver/prompt.go:25`, `:240`; `ruleTitle`, `:254`). The full input goes out as the
  tool call's `rawInput`, but `acpclient` does not carry it (`acpclient/update.go:193`), so the line session cannot
  show what it is asked to run.
- **The patterns repeat.** A tool call's question has eight options. The agent names each remembered one with the
  patterns it saves, so they appear six times (`question.options`, `acpserver/question.go:51-80`).
- **It is one line.** The line session writes the title, then `[n] name` for each option, two spaces apart, then
  `[f]` and `[c]`, all on one line of stderr (`permissionLine`, `internal/cli/permission.go:61`). This is
  0008-MADR D16's one-line question. Since 0005-PLAN F6-1 decision 6 it is numbered by option name where two options
  share a kind, which a tool call's eight always do. The terminal wraps the line at its edge, mid-word.

Other questions use the same line: trust (`acpserver/trust.go`), a grant (`acpserver/grants.go`), leaving plan mode
(`acpserver/modes.go`), the insults personality (`acpserver/personality.go`), and a repeated call's question. Another
ACP agent's questions would too. Their options are two or three, with short names.

The option names stay as they are for other ACP clients, which show each option as a button with its name.

## Decision Drivers

* The owner's report: the prompt shows the whole command, says the patterns once, and is laid out to be read.
* The keys stay as they are: `1`–`8`, `f` and `c` for a tool call; the same keys as today for every other question.
* Other ACP clients see what they see today: a short title and named options.
* Every model-made string is sanitised before it reaches the terminal (0008-MADR D8).
* Testable without a terminal: the prompt is a pure function of the request and the width, checked on `termtest`'s
  screen.

## Considered Options

* **A. A grouped menu.** The whole command, wrapped; the patterns once; Allow and Reject each on a row of scopes; then
  `[f]` and `[c]`. Chosen by the owner on 2026-10-22, from three drafts.
* **B. A vertical list:** the whole command and the patterns once, then one option per row.
* **C. Two-step keys:** `y`, `n`, `a…`, `N…`; a remembered answer asks a second question for the scope.
* **D. Today's line, shortened in the client only:** the client takes the patterns out of the option names and
  wraps the line, with no change to the agent.

## Decision Outcome

Chosen option: "A. A grouped menu", because the owner chose it. It is also the shortest layout that shows the whole
command and every choice, and it keeps today's keys.

- **D1. A tool call's question** (the eight options of `question.options`, known by their option ids) is drawn as:

  ```text
  Run echo '=== staged ==='; git diff --cached --stat; echo '=== log
    ==='; git log --oneline -5?
  Patterns: echo *, git diff *, git log *
    Allow   [1] once  [2] session  [3] project  [4] everywhere
    Reject  [5] once  [6] session  [7] project  [8] everywhere
    [f] reject and say why   [c] cancel
  >
  ```

  - **The first rows:** the call's whole description, its first letter capitalised, with `?`. It is wrapped at
    spaces, its continuation rows indented two columns, and a word wider than the row is left whole (0028-MADR D3).
  - **`Patterns:`** is shown only when the question offers remembered answers.
  - **When the remembered rejections hide the tool,** the Reject row ends `(hides the tool)`.
  - **The notes above the question** stay as they are: the limit, the repeated call, why only "once" is offered.
  - **The answer** is echoed after `> `, in words: `allow for this session`, `reject everywhere`.
- **D2. Every other question** is drawn as its title, wrapped as in D1, then its options, laid out in rows that never
  split an option. The keys and labels are today's. Trust's three options are numbered by name; the others use
  0008-MADR D16's `y`/`a`/`n`/`N`, and `[c] cancel` ends the list.
- **D3. What the agent sends.** A tool call's request carries, in `_meta.gobble`, the call's whole description
  (`title`, one line, cut at 2000 runes), the patterns it saves (`saves`) and whether a remembered rejection hides the
  tool (`hides`). The request's `title` stays cut to 60 runes and the option names stay as they are, for other ACP
  clients.

  *Amended 2026-10-22 (0029-PLAN P1, deviation 1; the owner's answer "Tools stop cutting"):* the agent did not
  have the whole description to send. Each built-in tool's describer cut it to 60 runes itself
  (`tool/builtin/builtin.go`, `title`, `maxTitle = 60`), before `acpserver` cut it again. The describers now keep a
  description on one line and no longer cut it; `acpserver`'s `cutTitle` still cuts every title an ACP client
  sees, a tool call's and a permission request's, to 60 runes, so other clients see no change.
- **D4. What the client reads.** `acpclient.PermissionRequest` gains `FullTitle`, `Saves` and `Hides`, read from
  `_meta.gobble`. The fields are added, so `acpclient`'s beta API stays compatible. With no `title` in the meta, the
  client uses the request's title.
- **D5. The width:** stderr's, read when the question is asked, at most 120 columns, as for gobble's own messages
  (0028-MADR D6).
- **D6. Sanitised:** the description, the patterns, the title and every option name pass through `term.Sanitize`, as
  today.

### Consequences

* Good, because the whole command is shown before it runs, and the patterns once.
* Good, because the keys and the answers sent to the agent are unchanged.
* Good, because other ACP clients see what they see today.
* Bad, because the question is taller: five or six rows for a command, against one row the terminal wrapped.
* Bad, because `_meta.gobble` grows three keys that the CLI depends on. Another client gains nothing from them unless
  it reads them.
* Neutral, because 0008-MADR D16's "one line" is replaced by D1 and D2, recorded there as a dated note.

### Confirmation

0029-PLAN's tests, each seen failing first. The prompt is checked on `termtest`'s screen with the owner's example at
60, 100 and 177 columns. The agent's meta and the client's fields are checked too. Then the existing prompt tests,
pseudo-terminal captures, the owner's look, and the gates on all three hosts.

## Pros and Cons of the Options

### A. A grouped menu

* Good, because it shows every choice in four rows below the command, with the keys as today.
* Bad, because it needs the agent to send the patterns and the whole description apart from the option names.

### B. A vertical list

* Good, because each option reads as a sentence.
* Bad, because it is ten rows of options for a command, two more than today's options, above the cursor.

### C. Two-step keys

* Good, because it is the shortest.
* Bad, because a remembered answer takes two keys and the keys change from today's numbers.

### D. Today's line, shortened in the client only

* Good, because it changes only the client.
* Bad, because the command stays cut at 60 runes. The client cannot get it back.
* Bad, because taking the patterns out of the option names depends on the agent's wording.

## More Information

- [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D8 and D16: the question on one line, sanitised.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) and 0005-PLAN F6-1 decision 6 and F6-2b decision
  F: the numbered options and `[f]`.
- [0028-MADR-reply-word-wrap.md](0028-MADR-reply-word-wrap.md) D3 and D6: words kept whole; gobble's messages wrapped.
- The plan: [0029-PLAN-permission-prompt-layout.md](0029-PLAN-permission-prompt-layout.md).
