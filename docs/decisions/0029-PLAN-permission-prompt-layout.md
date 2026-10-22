---
status: proposed
date: 2026-10-22
associated-madr: "0029-MADR-permission-prompt-layout.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement the grouped permission prompt

Associated MADR: [0029-MADR-permission-prompt-layout.md](0029-MADR-permission-prompt-layout.md)

## Goal

A tool call's permission question shows the whole command, wrapped, and the patterns once, with Allow and Reject on a
row each and today's keys. Every other question is wrapped, its options never split. Other ACP clients see no change
(MADR D1–D6).

## Scope

- **In:**
  - `acpserver/question.go`, `acpserver/prompt.go`: the meta's `title`, `saves` and `hides`;
  - `acpclient`: the request's `FullTitle`, `Saves` and `Hides`;
  - `internal/cli/permission.go`: the prompt;
  - their tests;
  - the prompt tests that read today's line: `internal/cli/permission_test.go`, `internal/cli/trust_test.go`,
    `internal/cli/personality_run_test.go` (12 assertions on `[c] cancel`). Their new form is what D1 and D2 decide,
    so it is not a deviation;
  - 0008-MADR D16's dated note; `docs/architecture.md`; `docs/README.md`.
- **Out:**
  - the option names and the 60-rune title other clients see;
  - the TUI mode;
  - the keys.

Order with 0028-PLAN P3 and P4 is the owner's to choose.

## Implementation Steps

### P1 — what the agent sends and the client reads (D3, D4)

1. **Tests first:**
   - `acpserver`: a tool call's question carries `_meta.gobble.title` (the whole description, cut at 2000 runes),
     `saves` and `hides`, and its title and option names are unchanged. Other questions carry none of the three.
   - `acpclient`: a request with the three keys gives `FullTitle`, `Saves` and `Hides`; without them, empty.
2. **The meta** in `question.meta`, from `permission.Rule`'s title and `q.saves` and `q.hides`; and **the fields** in
   `acpclient`'s request reading. Nothing the line session shows changes yet.

### P2 — the prompt (D1, D2, D5, D6)

1. **Tests first,** on `termtest`'s screen, through `permissionPrompt(request, width)`:
   - the owner's example at 60, 100 and 177 columns (capped at 120): the whole command, no word cut, `Patterns:` once,
     the Allow and Reject rows, `[f]` and `[c]`;
   - a question with `hides`;
   - the notes above it;
   - trust's question and a grant's;
   - another agent's question with long option names: no option split across rows;
   - a model-made description holding escape sequences, sanitised.

   The answer echoed in words: `allow for this session`, `reject everywhere`.
2. **`permissionPrompt`** in place of `permissionLine`, using `render`'s wrapping. `askPermission` reads stderr's
   width, at most 120. The existing prompt tests take the new form.

### P3 — on real terminals, and the records

1. **Captures:** a scripted model asks for a bash call in a pseudo-terminal at 60, 100 and 200 columns, on macOS and
   the Linux machine, and the Windows laptop headless. Each screen is drawn on `termtest` and checked, and the
   question is cancelled, so nothing runs. If no question can be reached from the binary without a test hook, that
   is a deviation, raised first.
2. **The owner's look** in WezTerm and Windows Terminal.
3. **Records:** 0008-MADR D16's dated note; `docs/architecture.md`; `docs/README.md`.

## Verification

- Every new test seen failing first, its failure quoted in the record.
- Negatives on a scratch copy, one mutation at a time:
  - the full title not sent;
  - the patterns repeated in the rows;
  - a word cut;
  - an option split across rows;
  - the width not capped;
  - a key changed;
  - the description not sanitised.
- Each phase:
  - the pre-add check;
  - the tests and the race detector, and `make preflight` (with `apidiff` for `acpclient`), on macOS, the Linux
    machine and the Windows laptop;
  - `check-records` and `markdownlint`.

## Rollout and Rollback

One commit per phase. P1 changes nothing a user sees, and other clients see no change at all. P2 changes the prompt;
reverting it restores the line. Nothing is stored.
