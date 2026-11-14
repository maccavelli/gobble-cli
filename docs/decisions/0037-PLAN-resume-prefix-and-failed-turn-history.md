---
status: in-progress
date: 2026-11-14
associated-madr: "0037-MADR-hardening-integrity-and-performance.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement keeping a resumed session's request prefix, and a failed turn's history, equal to the file's

Associated MADR: [0037-MADR-hardening-integrity-and-performance.md](0037-MADR-hardening-integrity-and-performance.md)
(accepted 2026-10-30; amended 2026-11-04 and 2026-11-14). This plan carries out D15 and the 2026-11-14 amendment's D42,
by D43 and D44. It is the MADR's fifth plan and needs none of the other four. ~~Q11 to Q13 are open. The steps below
are written to the recommended answers, Q11 (a), Q12 (a) and Q13 (a). Any other answer rewrites P1 before approval.~~

*(2026-11-14, the owner's rulings: Q11 "Memory takes the file's form", (a); Q13 "In the agent, on arrival", (a); Q12
none of the options: "test resume-prefix across all providers. once. measure. assess whether we should test every
caching wire, or only the 2 going forward based on results and facts." P1 is restructured to Q12's ruling: step 1a
and step 6a measure every wire once, before and after the change, and step 8 puts the assessment to the owner. Until
that is ruled, `TestResumeKeepsPrefix` covers `claude` and `openai`. Execution: "Approve and build". This plan is
`in-progress`; see the execution record.)*

## Goal

When this plan is complete:

* every request a session sends, live or after a close and a load, carries the messages of every earlier request of
  that session as a byte prefix, on the `claude` and `openai` wires, with the cache markers removed (D42);
* *(2026-11-14, Q12's ruling)* a one-time measurement of that check on every provider wire gobble can render offline
  is recorded, before and after the change, and the owner has ruled on the assessment that follows it: the permanent
  test's wires, and the harness's fate;
* after a failed model call or a failed append, the live session's next request is the request a reload of its file
  would send (D15);
* a live prompt that carries files, a resource, a link or several text blocks is one text block, joined as its user
  entry joins it, and a tool call's arguments are in the store's form from the moment they arrive (Q11 (a), Q13 (a));
* the session files this plan's code writes are byte-for-byte what `512db99` would write for the same turns.

## Scope

* **In:** the files each phase names, and nothing else. A file a phase needs and does not name is a deviation (C6).
* **Out:**
  * D34's plan 2 (D12–D14, D17–D22) and plan 3 (D23–D33);
  * [0057-MADR-cache-visibility-and-one-hour-pricing.md](0057-MADR-cache-visibility-and-one-hour-pricing.md), the
    same analysis's second record, whose code waits for `v0.7.0`;
  * any change to the session file's format, or to a stored session;
  * the three intended prefix changes of
    [0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md) F6, and compaction;
  * pushing and tagging, which need the owner's ask in the turn they happen (`AGENTS.md`).

## Conventions every phase follows

* **C1. Start state.**
  * The tree is clean apart from this plan's own uncommitted records, after P0.
  * `HEAD` is the previous phase's commit on `main`.
  * `go version` is `go1.27.4`.
  * `git rev-parse --path-format=absolute --git-path hooks` is the global hooks directory (checked at P0).
  * Each phase re-reads its files at the base it starts from, and records any line that moved since `512db99`.
* **C2. Scratch work.** Every probe and every planted break is made in a scratch copy of the tree under the session's
  scratch directory (`<scratch>/break-<phase>-<n>/`, from `git ls-files -co --exclude-standard`), never in the working
  tree. A break is an exact replacement whose old text is asserted present once, by a stdlib-only Python script. The
  record names the break and quotes the failure's first lines.
* **C3. Long output.** Every gate writes its whole output to `<scratch>/<phase>-<gate>.log`, and its exit status is
  taken before any filter.
* **C4. Heavy runs.** `make preflight`, `make race` and `go test ./...` run one at a time per host. Before each,
  `~/.agents/skills/available-dev-hosts/scripts/pick-host` names the host; the run goes there through
  `~/.agents/skills/available-dev-hosts/scripts/run-on-host --need <goos> -- <command>`, or through
  `~/.agents/skills/available-dev-hosts/scripts/run-heavy -- <command>` when it names the session host. A WSL host
  serves `--need linux` when the Linux machine is busy. A single-package `go test ./acpserver/`, `./agent/` or
  `./jsonx/` while iterating may stay on the session host.
* **C5. Phase gates,** in this order, each passing:
  1. `make pre-add-check FILES="<every changed .go file>"`;
  2. `make preflight` (C4);
  3. `make race` (C4);
  4. `go test -count=1 ./...` and `CGO_ENABLED=1 go test -race -count=1 ./...` with `--need linux` and
     `--need windows` (C4).

  No phase touches `internal/cli/...`, so `make perf` and `make probe-conhost` are not run.
* **C6. Deviations.** Any of these stops the phase: a failure the phase does not name; a file outside its list; an API
  not as written here; an `apidiff` finding; a golden that changes. The owner is given the evidence (file, line, exact
  failure, whether pre-existing), the resolutions with the recommended first and each one's cost, and what doing
  nothing means. No workaround is offered. The chosen resolution is a dated deviation entry here, and in the MADR when
  a decision or a fact changes, before work continues.
* **C7. Seen failing.** Every new test or case fails at its phase's base on a scratch copy (C2) before the change, and
  fails again on each planted break after it. The record quotes each failure.
* **C8. Commit.** One commit per phase, `git add` of exactly its files, then `git commit --no-edit`, on `main`, after
  C5 and on the owner's word in the turn it happens. The phase's execution record and this plan's `docs/README.md` row
  go in the same commit.
* **C9. Records' paths.** No hostname, account name or real machine path goes into a committed file, a quoted failure
  included: a temporary directory is written `<tmp>`.

## Order, and the other plans in flight

P0, then P1, P2 and P3 in order. P2's test compares against a reload, which needs P1's equality to hold.
*(2026-11-14: P1a, added by Q12's ruling, runs once the owner has ruled on P1's assessment, before P3. It is empty
when that ruling keeps P1's coverage and records the harness as a one-off.)*

**Merge order.** The owner's timing is "now", ahead of `v0.7.0` (MADR D43). Two plans are mid-way on `main`:

| Plan | Phases still to run at `512db99` | Files they name that this plan also changes |
| :--- | :--- | :--- |
| [0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md) | P2 to P8 | none. Its `acpserver` files are `prompt.go`, `auth.go`, `authsignin.go` (new), `auth_test.go`, `errordata_test.go`, `retry_test.go`, `prompt_test.go` and `testdata/authenticate-elicitation.golden` (new); its `agent` file is `agent/event.go`. |
| [0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md) | P2-2 to P6 | `acpserver/agent.go`, in P3 (`Options.PlanConsent`, `planGated`), which is staged with 0051-PLAN P4 as one commit. |

This plan changes `agent/agent.go`, `jsonx/jsonx.go`, `acpserver/agent.go`, `acpserver/history.go` and tests. So:

* **Recommended, and ruled with "Approve and build" (2026-11-14):** P1 and P2 land on `main` before 0051-PLAN's P2
  starts, and outside 0056-PLAN's phases. The 0051-PLAN P4 and 0052-PLAN P3 commit is
  then written on top of them. The one shared file, `acpserver/agent.go`, is edited in different functions: there
  `Options` and the title and compaction gates, here `Prompt`'s prompt shaping and its error exits.
* **If a 0051 or 0052 phase is already running** when this plan is approved, this plan's phases land between that
  plan's phase commits, never inside one. When they land after the 0051-PLAN P4 and 0052-PLAN P3 commit, C1's re-read
  of `acpserver/agent.go` records the moved lines.
* **0056.** [0056-PLAN-curate-the-terminal-session-view-with-view-modes.md](0056-PLAN-curate-the-terminal-session-view-with-view-modes.md)
  starts after `v0.7.0`, and its C6 requires `git diff --exit-code <base> -- acpserver agent tool acpext llm` to be
  empty after each of its phases. This plan therefore lands before 0056-PLAN's P0, or after its P4, never between.

## Implementation Steps

### P0 — the records, and the answers

1. Check C1's hooks directory and record it.
2. The owner answers Q11, Q12 and Q13. An answer other than the recommended one is recorded here as a dated entry, and
   P1 is rewritten to it before P1 is approved.
3. Write the dated notes that wait for Q11 (MADR D44), each beside the text it qualifies:
   * [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md) F3-4, "The user entry holds the expanded message,
     so a resumed session gives the model what it had": under Q11 (a), the live prompt is now one block, joined as the
     entry joins it, so the resumed session gives the model the same bytes (0037-MADR F33, D42);
   * [0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md) F5, "Its bodies are
     byte-identical across calls and provider instances": within a process; across a resume, not until this plan
     (0037-MADR F33).
4. The owner commits the MADR amendment, this plan, the two notes and the `docs/README.md` rows, under the bootstrap
   exception: records only. `make check-records markdownlint` passes before that commit.

*(2026-11-14: steps 1 to 3 are done; see "P0 record". Step 4, the commit, is the owner's.)*

### P1 — a resumed session sends what the live session sent (MADR D42; Q11 (a), Q12's ruling, Q13 (a))

**Files:** `jsonx/jsonx.go`, `jsonx/jsonx_test.go`, `agent/agent.go`, `agent/agent_test.go`, `acpserver/agent.go`,
`acpserver/history.go`, `acpserver/history_test.go`, `acpserver/mentions_test.go`, `acpserver/resume_test.go` (new),
this plan, `docs/README.md`.

1. **Facts re-read (C1).** Confirm and record:
   * `acpserver/agent.go:675` prepends the files block; `:1038-1046` (`userMessage`) makes one block per text block;
     `:671-673` is the comment that cites 0005-PLAN F3-4;
   * `acpserver/history.go:61-69` (`userEntryMessage`) joins with `"\n\n"`; `:96-106` keeps an object's arguments as a
     `jsontext.Value` and anything else raw under `gobble.rawArguments`;
   * `agent/agent.go:724-726` receives `llm.ToolCallDone`, and `:416` copies `c.Args` into the assistant message;
   * `acpserver/mentions_test.go:126` expects two user blocks;
   * `agent` already imports `jsonx`, and `internal/archtest`'s rule 10 lets `jsonx` import nothing of this module.

1a. **The one-time measurement, before (Q12's ruling, 2026-11-14).** Run this on a scratch copy at the phase's base
    (C2). The harness is a throwaway test in package `acpserver` of that copy, not in the tree.
    * **The wires.** Every id `provider.Providers()` returns (`llm/provider/ambient.go:91-97`), each with a model the
      catalog lists for it. A gateway with more than one route (OpenCode Zen's messages and responses routes; Kilo's
      `anthropic/` models and its others) is measured once per route, with a model that takes that route.
    * **No live provider is called.** Each provider is built with `provider.New`, a dummy key, `Retry: Off` and the
      recording transport that answers 400. A wire that cannot be built that way, such as one needing a sign-in rather
      than a key, is listed with the reason, not measured.
    * **The run.** The scripted sessions are `TestResumeKeepsPrefix`'s three cases, each rendered through every wire.
    * **What is recorded.** A table in the execution record, one row per wire and route, with these columns:
      * whether the tools and system prompt held;
      * whether the messages' prefix held for each case, `mention`, `two text blocks` and `spaced arguments`;
      * the first diverging item;
      * the cause the wire exposes: (a) blocks, (b) arguments, both, or none;
      * whether the wire sends any cache marker or routing key under `cacheRetention: short`, from the recorded body.
2. **The store's argument form, once.** `jsonx` gains one exported function that gives a tool call's arguments in the
   form the session store gives them back: when the text decodes as a JSON object, its encoding as a `jsontext.Value`
   through `jsonx.Marshal`; otherwise the text unchanged. Its name and doc are recorded here. `apidiff` reports an
   addition only.
3. **Arguments at receipt (Q13 (a)).** In `agent/agent.go`, the `llm.ToolCallDone` case sets `ev.Args` through step
   2's function, beside `t.repair`. Tool execution, the loop guard's key and every later request of the turn then see
   the same bytes the file will hold.
4. **The prompt as its entry (Q11 (a)).** In `acpserver/agent.go`'s `Prompt`, after the files are prepended and before
   the entry is written, the prompt's text blocks become one text block, joined with `"\n\n"`. `userEntryMessage` and
   this step share one join, so they cannot drift. The comment at `:671-673` is reworded to say so.
5. **Tests.**

   | Test | What it holds | Seen failing at the base (C7) |
   | :--- | :--- | :--- |
   | `TestResumeKeepsPrefix` (`acpserver/resume_test.go`) | three cases, `mention`, `two text blocks` and `spaced arguments`: a scripted `read` call, two replies, a close, a load into a second `Agent` over the same store, one more prompt; each request rendered through the `claude` and `openai` wires of `llm/provider` with a recording transport that answers 400 and `Retry: Off`; with `cache_control` removed, every earlier request's tools and system equal the post-reload request's, and its message items are a byte prefix of the post-reload request's | at `512db99`, as the MADR quotes: `mention` and `two text blocks` on both wires at item 0; `spaced arguments` on `openai` at item 1 |
   | `TestToolArgsAsStored` (`acpserver/history_test.go`) | for `{"path":"a.go"}`, `{"path": "a.go"}`, a multi-line object, `{"path":"café.go"}`, `{"path":"a\/b.go"}`, `{"b":1,"a":2}`, `{"n":1.0}`, `[1]` and `{"path":` (invalid): step 2's form equals what `assistantEntryMessage`, a write and a read through `jsonl`, and `assistantContent` give back | step 2's function absent at the base: the test does not compile there; recorded as such, and its planted break below is its failure |
   | `TestArgsTakeTheStoresForm` (`agent/agent_test.go`) | a scripted call with `{"path": "x"}`: the tool receives, and the turn's next request carries, `{"path":"x"}`; an invalid argument text reaches both unchanged | at the base the next request carries `{"path": "x"}` |
   | `TestPromptCarriesFiles` (`acpserver/mentions_test.go`, changed) | `userTextsOf` the first request is one text, the files then `"\n\n"` then `explain @a.txt`; the entry's text is unchanged | it is the existing test's expectation changed, named here as the phase's one changed assertion, with its reason (Q11 (a)) |

   The `spaced arguments` case's prompt is a single text block with no mention, so it isolates cause (b). The
   `two text blocks` case has no mention, so it isolates the join from `@path` handling.
6. **Planted breaks (C7),** each on a scratch copy of the finished phase:
   * step 4's join removed: `TestResumeKeepsPrefix/mention` and `/two_text_blocks` fail on both wires at item 0;
   * step 3's call removed: `TestResumeKeepsPrefix/spaced_arguments` fails on `openai` at item 1, and
     `TestArgsTakeTheStoresForm` fails;
   * step 2's function built on `encoding/json`'s `Compact`, which keeps `\/`: `TestToolArgsAsStored` fails on
     `{"path":"a\/b.go"}`;
   * step 2's function compacting a non-object: `TestToolArgsAsStored` fails on `[1]`.
7. **Unchanged.** `git diff --exit-code <base> -- '*.golden' session/jsonl/testdata acpserver/testdata` is empty. On
   scratch copies at the base and at the finished phase, the session files of `TestResumeKeepsPrefix`'s `mention` and
   `spaced arguments` cases are kept, their timestamps and ids masked by a stdlib Python script, and compared: they
   are equal. The record quotes the comparison.

7a. **The one-time measurement, after (Q12's ruling).** Step 1a's harness runs again, on a scratch copy of the
    finished phase, over the same wires and cases. Its table gains an "after" column per case. A wire whose prefix
    still breaks after the change is a deviation (C6): the change has missed a cause.
8. **The assessment, put to the owner (Q12's ruling).** Written in the execution record from 1a's and 7a's tables
   and the facts behind them, it asks two things:
   * whether the permanent `TestResumeKeepsPrefix` renders every caching wire, or only the two shapes `claude` and
     `openai`;
   * whether the measurement harness is kept as a test, or stays a recorded one-off.

   It gives three options, the recommended one first, each with its cost: test time, fixtures to maintain, and what
   the two shapes leave uncovered, as the tables show it. Until the owner rules, `TestResumeKeepsPrefix` covers
   `claude` and `openai`, as step 5 has it. P1's commit does not wait for the ruling.
9. C5. Commit (C8).

### P1a — the permanent test's wires, per the owner's ruling on P1's assessment (added 2026-11-14, Q12's ruling)

**Files:** `acpserver/resume_test.go`, with a test file for the harness if it is kept; this plan; `docs/README.md`.
The ruling may name more files. Any it does not name is a deviation (C6).

1. Write the ruling here as a dated entry, verbatim.
2. If the ruling widens `TestResumeKeepsPrefix` or keeps the harness as a test: make that change. Each newly covered
   wire is seen failing on a planted break (C7), step 4's join removed, at the item its row in step 1a's table names.
3. If the ruling keeps `claude` and `openai` and the harness as a one-off: nothing changes in the tree, and the record
   says so.
4. C5 when the tree changed. Commit (C8).

### P2 — a failed turn leaves the live session the file's (MADR D15)

**Files:** `acpserver/agent.go`, `acpserver/resume_test.go`, this plan, `docs/README.md`.

1. **Facts re-read (C1).** `Prompt`'s exits at `512db99`: `:684`, `:694`, `:713` (append failures before the turn);
   `:755-766` (a model error, compacted or not); `:771-774` (an append failure inside the turn); the cancel paths
   `:632` and `:800`; and `:789-790` (`out.err` after `End`, where `s.history` is already the turn's). Record each
   exit's line at the phase's base, and whether its file can hold entries live history lacks.
2. **The change.** Every exit at which the file may hold entries `s.history` lacks sets `s.history` from
   `s.state().history` under `a.mu`, as the compacted branch does at `:755-761`. The non-compacted branch at
   `:762-765` becomes the same replay. An exit that writes nothing (the cancel at `:632`, before any entry) is left as
   it is, and the record says why.
3. **Tests** in `acpserver/resume_test.go`:

   | Test | What it holds | Seen failing at the base (C7) |
   | :--- | :--- | :--- |
   | `TestFailedTurnHistoryIsTheFiles` | a `read` call, then a scripted 400 on the next model call, then a second prompt: run once in one process and once with a close and a load before the second prompt; the second prompt's request is the same in both, by role sequence and by its `claude` and `openai` wire bytes with the cache markers removed | at `512db99`: `live: [system:text user:text user:text]`, `reloaded: [system:text user:text assistant:tool_call tool:tool_result user:text]` |
   | `TestFailedAppendHistoryIsTheFiles` | a store wrapping `jsonl` that fails the append of the turn's first tool result: the prompt answers `appendFailed`'s error; the next prompt's request is the request a reload of the file sends | at the base the live request lacks the assistant's tool call that the file holds |

4. **Planted breaks (C7):** the non-compacted branch restored to `append(s.history, prompt)`:
   `TestFailedTurnHistoryIsTheFiles` fails with `live: [system:text user:text user:text]`; the append-failure exit left
   without the replay: `TestFailedAppendHistoryIsTheFiles` fails.
5. **Cost.** The replay runs only on failure (D15). The record notes `s.state()`'s time for the test's session; no
   budget is set.
6. C5. Commit (C8).

### P3 — close-out

1. **Goldens.** `git diff --stat <P0>..HEAD -- '*.golden' session/jsonl/testdata acpserver/testdata` is empty.
2. **The disclosure guard,** run over the outgoing commits (`AGENTS.md`, "Identifiers"), is clean.
3. **Status.** This plan becomes `complete` when every Goal item is met. The MADR stays `accepted`, with D15 and D42
   carried out.
4. Commit (C8).

## Verification

Each phase's own steps are its acceptance criteria. Across the plan:

* `go test -count=1 -run 'TestResumeKeepsPrefix|TestToolArgsAsStored|TestPromptCarriesFiles|TestFailedTurnHistoryIsTheFiles|TestFailedAppendHistoryIsTheFiles' ./acpserver/`
  passes on all three hosts, and under `-race`;
* `go test -count=1 -run 'TestArgsTakeTheStoresForm' ./agent/` passes;
* *(2026-11-14, Q12's ruling)* P1's execution record holds step 1a's and step 7a's tables for every wire and route,
  and the owner's ruling on step 8's assessment, carried out by P1a;
* every test named above has its quoted failure at its base and on each planted break (C7);
* the goldens, `session/jsonl/testdata/` and `acpserver/testdata/` are byte-identical to P0's commit;
* `apidiff` reports one addition, in `jsonx`, and nothing else.

## Rollout and Rollback

* Each phase is one commit and reverts on its own with `git revert`. Reverting P1 after P2 is safe: P2's tests compare
  a live session with a reload, and fail only if P1's join or form is reverted while a mention or spaced arguments are
  in the session, which P2's tests do not use.
* Nothing is pushed by this plan. The owner pushes, after the disclosure guard.
* **What users see change:**
  * P1: a resumed session reads back the cached prefix its live session wrote, from the first request after the
    resume, for any session written after the change. A session written before it misses once, as today. The model
    receives a prompt with files as one block, as every resumed session already sends it.
  * P2: after a failed model call or a failed append, the model's next request holds the tool calls that ran and their
    results, as the file does.
* The session file is unchanged in format and in bytes. Any earlier gobble, and Pi, read what this plan's code writes.

## Execution record

~~Not started. The plan is `proposed`; Q11 to Q13 wait for the owner.~~

**Approved by the owner on 2026-11-14: "Approve and build".** The rulings of that day:

* Q11: "Memory takes the file's form", (a).
* Q12: none of the options, verbatim: "test resume-prefix across all providers. once. measure. assess whether we
  should test every caching wire, or only the 2 going forward based on results and facts." P1 gains steps 1a, 7a and
  8, and P1a is added.
* Q13: "In the agent, on arrival", (a).

The plan is `in-progress`. It lands before 0051-PLAN's P2, and outside 0056-PLAN's phases.

### P0 record (2026-11-14): the records, and the answers

1. **Hooks.** `git rev-parse --path-format=absolute --git-path hooks` names the global hooks directory (C1). It is
   described here, not quoted (C9).
2. **The answers** are as listed above. Q12's answer, which was not one of the options, is written into P1 and P1a
   before any code.
3. **The notes are written,** each dated 2026-11-14 beside the text it qualifies:
   * [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md), F3-4's "The user entry holds the expanded
     message" bullet;
   * [0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md), F5.
4. **Not done:** the commit. It is the owner's, under the bootstrap exception, with these records only.
