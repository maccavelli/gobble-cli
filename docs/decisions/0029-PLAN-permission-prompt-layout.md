---
status: in-progress
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

## Execution record

Approved by the owner's "start 0029", 2026-10-22, after 0031-PLAN P5 (`ebe2be5`), as the owner's order of that
day had it ("0030 P5, then 0031, then 0029").

### P1 execution record (2026-10-22)

**Deviation 1, 2026-10-22, decided by the owner** ("Tools stop cutting"). Before P1's first test, a read of how a
permission request's title is made found that `permission.Rule.Title` is the tool's own description
(`agent/agent.go:789`, `:844`, `:954`), and that each of the fourteen built-in describers cuts it to 60 runes
(`tool/builtin/builtin.go`, `title`, `maxTitle = 60`) before `acpserver` cuts it again (`cutTitle`,
`acpserver/prompt.go:108`, `:256`). Step 2's meta "from `permission.Rule`'s title" would carry the cut command.
The describers now keep a description on one line and no longer cut it; `acpserver` still cuts every title an ACP
client sees. 0029-MADR D3 is amended. Files added to P1's scope: `tool/builtin/builtin.go` and
`tool/builtin/builtin_test.go`, whose `TestBounds` and `TestDescribeTitles` pinned the cut.

1. **Tests first,** each seen failing on the code before it, or on a stub (`acpclient.PermissionRequest`'s three
   fields, added and read by nothing):
   - `acpserver/answers_test.go`, `TestQuestionFullTitle`: the owner's example as a bash call carries
     `title` "run echo '=== staged ==='; …; git log --oneline -5", `saves` `echo *`, `git diff *`, `git log *`, and no
     `hides`; its request's title stays 60 runes, cut, and its options' names stay; a tool with no subject carries
     `hides`; a delete, which offers no "always", carries its title alone; a 2,500-rune command's title is cut at 2,000
     runes; a grant's question carries none of the three. It failed on the old code with each tool call's `_meta`
     holding its rule alone (`a command's _meta map[gobble:map[rule:map[from:default rule:ask: bash]]]`) and `a long
     command's title has 0 runes, want 2000, cut`; the grant, title and option checks passed, and stay as guards.
   - `acpclient/sessions_test.go`, `TestPermissionFullTitle`: the three keys give `FullTitle`, `Saves` and `Hides`;
     with no meta, a mistyped one or another agent's, the request's title and no marks. It failed with `with the
     meta: "", [], hides false`.
   - `tool/builtin/builtin_test.go`, turned round by deviation 1: `TestBounds`, a 100-rune description whole and on one
     line, failed `title has 60 runes, cut true; want all 100`; `TestDescribeTitles`, a long command's description
     whole, failed with the description cut at 60.
   - `TestQuestionOptions`'s check of a write's meta took the new key: the rule and `title` "write a.txt", and nothing
     else (it had checked the rule alone).
2. **The code:** `tool/builtin`'s `title` keeps a description on one line, whole, and `maxTitle` goes;
   `acpserver/question.go`'s `meta` adds `title` (cut at `maxFullTitle`, 2,000 runes), `saves` when there are patterns,
   and `hides` beside an "always", from `ask`, which sets the question's title from the rule's; `acpclient` reads the
   three keys, a mistyped one being none, and `FullTitle` falls back to the request's title.
3. **Negatives,** one at a time on a scratch copy:
   - **the full title not sent:** `TestQuestionOptions` and `TestQuestionFullTitle` fail;
   - **the tools cutting again:** `TestBounds` and `TestDescribeTitles` fail;
   - **the patterns not sent**, and **hides not sent:** `TestQuestionFullTitle` fails;
   - **the client ignoring the title:** `TestPermissionFullTitle` fails, the request's cut title in its place;
   - **no fallback to the request's title:** `TestPermissionFullTitle` fails for no meta, a mistyped one and another
     agent's.

   Six mutations, six failures; the restored copy passed. Two guards were found equivalent and are recorded: `saves`
   had been sent "beside an always", but only an "always" ever has patterns (`acpserver/modes.go`), so that check was
   dropped from the code; and `hides` is sent only beside an "always", as `question.options` shows it, though no
   reachable question has a subjectless call that offers "once" alone, so no test can tell it from sending `hides`
   always. It is kept, for the meta to say what the options do.

**Found, not fixed (outside this plan):** a bash call with an empty command is offered "always", and its "always"
saves an empty pattern (`saves` `[""]`), which the options' names show as `` ` ` ``.

**Gates:**

- The pre-add check on the six changed `.go` files: `6 file(s) clean (gofmt, golint, govulncheck)`.
- macOS: `make preflight`, `preflight passed` (`apidiff` is a no-op below `v1.0.0`); the tests and the race detector,
  36 packages ok each.
- The Linux machine and the Windows laptop, on a copy of the tree: the tests and the race detector, 36 packages ok on
  each; `make preflight` reached `apidiff` (golangci-lint `0 issues.`, govulncheck "No vulnerabilities found.") and
  stopped at `verify-build-metadata`, as the copies are not git checkouts.
- `make check-records markdownlint`: exit 0 on each host.
- The Windows laptop, `make probe-conhost`: `--- PASS: TestConhost`.
- `make perf`, each within three times its baseline: macOS ReplyRender 34.2 µs, Reprint 10.0 ms, LineSessionStart
  971 µs, KeyDraw 1.58 µs; the Linux machine 136 µs, 34.3 ms, 4.94 ms, 5.31 µs; the Windows laptop 28.5 µs, 7.42 ms,
  4.98 ms, 1.08 µs.

**P1 complete, 2026-10-22.**

**P2 stopped by the owner, 2026-10-22** ("Stop p2"), before any of it was written, and an evaluation of integrating
go-tui-lib asked for instead ("Evaluate integrating the go-tui-lib sdk now so we can build the more complex cli and
tui elements in a canonical manner"). P2 waits for that evaluation's outcome.
