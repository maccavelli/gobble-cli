---
status: complete
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

  *(2026-11-14, amendment, the owner's ruling "Amend the promise", relayed by the coordinator: the session files are
  byte-identical to what gobble wrote before, except a tool call with empty argument text, whose entry is written
  `"arguments":{}` with no `gobble.rawArguments` member. The cause is the owner's ruling "Yes, in this plan": empty
  argument text is treated as `{}` on arrival, so the entry builder sees an object and has nothing raw to keep. The
  entry reads back as `{}`, as before. Verified in P3: the `512db99` code, given the file the finished phase wrote for such
  a call, gives the arguments back as `{}`. Not verified by running Pi: the member is gobble's own, which Pi ignores
  (`gobbleMember`'s comment in `acpserver/history.go`), so its absence is not a change Pi can see.)*


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

   *(2026-11-14, amendment, the owner's ruling "Yes, in this plan", from P1's execution: "otherwise the text unchanged"
   holds for everything except the empty text, which is `{}`. At `cccef77`, `acpserver/history.go:96-106` stores any
   text that does not decode as a JSON object as `Arguments` `{}` with `gobble.rawArguments` set to the text, and
   `:316-321` returns `rawArguments` only when it is not empty, else `string(b.Arguments)`: so `""` comes back as
   `{}`. The function returns `{}` for `""`; whitespace-only text and every other non-object text are unchanged, as
   they come back. `TestToolArgsAsStored` and `TestArgsTakeTheStoresForm` each gain an empty case, and a sixth planted
   break removes the empty-text rule.)*
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

*(2026-11-14, carried out in P1's working tree, before P1's commit, at the coordinator's word: the owner's ruling on
step 8's assessment, verbatim, "One wire per shape, 5 wires". `TestResumeKeepsPrefix` covers one wire per body shape,
five in all; the harness stays a recorded one-off, its tables in P1's record. See "P1a record".)*

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

### P2a — a test for each of the three pre-turn replays (added 2026-11-14)

*(2026-11-14, amendment, the owner's ruling, verbatim: "Those 3 untested replays need tests". It supersedes the
coordinator's earlier instruction, "Keep the three untested replays as built", which was the coordinator's call and not
the owner's. The step comes before P3's close-out, and P3's `complete` waits for it.)*

**Files:** `acpserver/resume_test.go`, this plan, `docs/README.md`, and the 0037 MADR's dated note on the sentence it
annotates. No production file: the step is test-only, and a test that exposes a defect in `Prompt` or `replayFile` stops
it (C6).

1. **Facts re-read (C1).** The three pre-turn calls of `a.replayFile(s)` in `Prompt` (`acpserver/agent.go`), in the order
   `Prompt` reaches them: after the `writePrompt` failure (the baseline or patch entry of `gobble.prompt`), after the
   user entry's failed append, and after the `gobble.tools` entry's failed append. Their lines are recorded in the P2a
   record, read from the file.
2. **One test for each,** in the style of `TestFailedAppendHistoryIsTheFiles`: make that one append fail, then assert that
   the live session's history equals what a fresh load of the file gives, not that a function ran.
3. **Each seen failing (C7),** on a scratch copy of the finished step: that site's `a.replayFile(s)` call removed, that
   site's test fails and the other two still pass. A test that passes with its replay removed is a finding, not
   something to force: the replay at that site is dead code, or the test cannot tell it from none, and the owner decides.
4. **C5**, then the MADR's sentence "The three pre-turn replays have no test of their own" gets a dated note that they now
   do. Commit (C8).

*(2026-11-14, amendment, the owner's ruling on P2a's finding, relayed by the coordinator. The finding stands as it was
found: with each call removed on a scratch copy of the tree at `124213c`, the test's cases for the system-prompt-write
exit (`:698`) and the user-entry exit (`:709`) passed, and the case for the `gobble.tools` exit (`:729`) failed with
`live: []`, `file: ["user{text|two|||}"]`. The three options put to the owner were "Keep as a safety net" (no code change,
the two replays recorded as unprovable), "Keep and prove by double plant", and "Remove the two". The owner chose
**"Remove the two"**: delete the two replay calls that cannot matter, keep the tools-entry replay with its proven test,
and keep the cases for the other two exits as checks that memory equals the file there, each seen to fail on a plant that
lets memory run ahead. Why the two cannot matter: at most one append precedes each of those exits, and memory is
updated only after a write succeeds, so at those exits the file holds nothing that the history lacks.)*

*(2026-11-14, amendment to P2a's scope, by the same ruling: the files are `acpserver/agent.go`, only the two
`a.replayFile(s)` calls at the system-prompt-write exit and the user-entry exit of `Prompt`, each replaced by a comment
saying why no replay is needed there, and nothing else in production code; and `acpserver/resume_test.go`. This widens P2a
from test-only to a two-line production change. P2a's step 3 now reads: each of the four cases of the two exits is seen
failing on a plant that lets memory and the file disagree at that exit; the tools-entry case is seen failing with its
replay removed; a case that no plausible plant can make fail is a finding that stops the step.)*

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

  *(2026-11-14, amendment, as Goal 5's: the same exception. A tool call with empty argument text is written
  `"arguments":{}` without the `gobble.rawArguments` member that `512db99` wrote beside it, and reads back the same.
  Earlier gobble reads it (verified in P3); Pi is not run, and ignores the member by `gobbleMember`'s comment.)*

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

<!-- p1-record-begin -->
### P1 record (2026-11-14): a resumed session sends what the live session sent

Run by a mainstream-tier agent at medium effort, from base `cccef77` (the plan names `512db99`; C1's re-read follows).

**Facts re-read at `cccef77` (C1).** All as step 1 states, with these lines moved from `512db99`'s numbers: the files
block is prepended at `acpserver/agent.go:674`, the comment citing 0005-PLAN F3-4 is `:671-673`, `userMessage` is
`:1038-1046`, `userEntryMessage` is `acpserver/history.go:61`, the object-or-raw argument split is `:96-106`,
`agent/agent.go:724` receives `llm.ToolCallDone` and `:416` copies `c.Args`, and `acpserver/mentions_test.go:126` expects
two user blocks. `agent` already imports `jsonx`.

**The change.**

* `jsonx.StoredArguments(text string) string` (step 2): when `text` decodes as a JSON object (the predicate
  `assistantEntryMessage` uses), its encoding by `jsonx.Marshal`; the empty text is `{}` (the owner's amendment, below); any other text unchanged.
  `apidiff`: one addition.
* `agent/agent.go`, the `llm.ToolCallDone` case: `ev.Args = jsonx.StoredArguments(ev.Args)`, beside `t.repair` (step 3).
* `acpserver/history.go`: `userText` is the one join (`"\n\n"`), used by `userEntryMessage` and by the new `asEntry`,
  which makes a message of two or more blocks one text block; a message of one block or none is returned as it is.
  `acpserver/agent.go`'s `Prompt` calls `asEntry` after the files are prepended; the comment at `:671-673` is reworded
  (step 4).
* Tests: `TestResumeKeepsPrefix` (`acpserver/resume_test.go`, new; four cases on five wires after P1a), `TestToolArgsAsStored`
  (`acpserver/history_test.go`; 12 cases), `TestArgsTakeTheStoresForm` (`agent/agent_test.go`; 3 cases), and `TestPromptCarriesFiles`' one changed assertion (the model's
  first request holds one text, the files, `"\n\n"`, then `explain @a.txt`).

**Deviation 1 (2026-11-14, resolved inside the phase; for the owner's notice).** Step 6's fourth planted break,
"step 2's function compacting a non-object: `TestToolArgsAsStored` fails on `[1]`", cannot fail on `[1]`: it is already
compact, so a function that compacts any valid JSON returns `[1]` unchanged. Run against the plan's nine cases, that
break passed (`ok`, exit 0). The resolution added one case to `TestToolArgsAsStored`, `spaced array` (`[ 1 ]`, expected
unchanged), which the break fails on (quoted below). No assertion was loosened, and the plan's other cases are as written.
The plan's wording of that break should read "fails on `[ 1 ]`" when the owner next amends it.

**Amendment 1: the empty argument text (2026-11-14, the owner's ruling "Yes, in this plan").** My hand-off of P1's first
build found that a tool call whose argument text is empty is not given back as it was stored. At `cccef77`,
`acpserver/history.go:96-106` (`assistantEntryMessage`) stores any text that does not decode as a JSON object as
`Arguments` `{}` with `gobble.rawArguments` set to the text (`b.Arguments = jsontext.Value("{}")`, then
`extra(map[string]string{"rawArguments": c.ToolArgs})`), and `:316-321` (`assistantContent`) returns `rawArguments` only
`if ... raw.RawArguments != ""`, otherwise `args := string(b.Arguments)`: so `""` is given back as `{}`, while a live
request carried `""`. Whitespace-only text is not empty and comes back unchanged. The PLAN's step 2 and the MADR's D42
carry dated notes; `jsonx.StoredArguments` returns `{}` for `""` and is otherwise as before; `TestToolArgsAsStored` has
`empty` (`""` to `{}`) and `blank` (`" "` unchanged); `TestArgsTakeTheStoresForm` has `empty`; `TestResumeKeepsPrefix` has
`empty arguments`, which breaks on the two wires that carry arguments as a string. A tool whose call has no arguments
now receives `{}` where it received nothing; no existing test depended on that (the suites pass). This changes the
"unchanged" half of step 2's rule for one input and no other.

#### Step 1a and step 7a: the one-time measurement, every wire

Harness: a throwaway test in package `acpserver` of a scratch copy, never in the tree. Its scripted sessions are
`TestResumeKeepsPrefix`'s three cases, each recorded once and rendered through every wire below: `provider.New` with a
dummy key, `Retry: Off`, a recording transport that answers 400, `llm.CacheShort` on every request, so no live provider
was called and no credential was used. The comparison removes every member whose name holds `cache` (any case, any
depth) and compares the rest in order: all members except the message sequence ("tools/system held"), then each earlier
request's message items against the post-reload request's, as a prefix. The "before" runs are at `cccef77`; the "after"
runs are on the finished phase. Every id `provider.Providers()` returns was built (10 of 10); none is listed as
unbuildable. Each gateway route was measured separately, with a model the route table sends there: the route is read
from the path the body went to. Three facts about the choice of models: `ollama`'s catalog lists none, so `llama3` is an
arbitrary name; `opencode-go`'s route table has no Google route, so none was measured for it; Kilo's two rows differ
only in the `anthropic/` prefix, which adds a `cache_control` member. "Cause": (a) is the prompt's blocks, (b) the
arguments' spacing; "none" after the change. The table's cells read before, then after.

| wire (model) | route path | tools/system held | mention | two text blocks | spaced arguments | cause | cache markers in the body | session or cache headers |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| claude messages (claude-haiku-5-5) | /v1/messages | yes | breaks at request 0, item 0 -> holds | breaks at request 0, item 0 -> holds | holds -> holds | (a) blocks -> none | cache_control | none |
| openai responses (gpt-6-luna) | /v1/responses | yes | breaks at request 0, item 0 -> holds | breaks at request 0, item 0 -> holds | breaks at request 1, item 1 -> holds | both -> none | prompt_cache_key | none |
| gemini interactions (gemini-3.8-flash) | /v1beta/interactions | yes | breaks at request 0, item 0 -> holds | breaks at request 0, item 0 -> holds | holds -> holds | (a) blocks -> none | none | none |
| grok (grok-4.6) | /v1/responses | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | prompt_cache_key | none |
| opencode-zen messages route (claude-haiku-5-5) | /zen/v1/messages | yes | breaks at request 0, item 0 -> holds | breaks at request 0, item 0 -> holds | holds -> holds | (a) blocks -> none | cache_control | X-Opencode-Session |
| opencode-zen responses route (gpt-6-luna) | /zen/v1/responses | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | prompt_cache_key | X-Opencode-Session |
| opencode-zen chat route (deepseek-v4-flash) | /zen/v1/chat/completions | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | none | X-Opencode-Session |
| opencode-zen google route (gemini-3.5-flash) | /zen/v1/models/gemini-3.5-flash:streamGenerateContent | yes | breaks at request 0, item 0 -> holds | breaks at request 0, item 0 -> holds | holds -> holds | (a) blocks -> none | none | X-Opencode-Session |
| opencode-go messages route (minimax-m3) | /zen/go/v1/messages | yes | breaks at request 0, item 0 -> holds | breaks at request 0, item 0 -> holds | holds -> holds | (a) blocks -> none | cache_control | X-Opencode-Session |
| opencode-go responses route (gpt-6-luna) | /zen/go/v1/responses | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | prompt_cache_key | X-Opencode-Session |
| opencode-go chat route (deepseek-v4.1-flash) | /zen/go/v1/chat/completions | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | none | X-Opencode-Session |
| huggingface (deepseek-ai/DeepSeek-V4-Flash-0731) | /v1/chat/completions | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | none | none |
| kilo anthropic/ models (anthropic/claude-haiku-5-5) | /api/gateway/chat/completions | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | cache_control | none |
| kilo other models (deepseek/deepseek-v4.1-flash) | /api/gateway/chat/completions | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | none | none |
| together (zai-org/GLM-5.3-Flash) | /v1/chat/completions | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | prompt_cache_key | none |
| ollama (llama3) | /v1/chat/completions | yes | breaks at request 0, item 1 -> holds | breaks at request 0, item 1 -> holds | breaks at request 1, item 2 -> holds | both -> none | none | none |

Reading the table, from the recorded bodies:

* Before the change, every wire broke the prefix at the first request of `mention` and `two text blocks`: 16 of 16. The
  first diverging item is the first user message (item 0, or item 1 on the wires that put the system prompt in the
  sequence: `grok`, the OpenCode responses routes and every chat-completions wire). Its first form is the files block alone, or `first block`; its
  post-reload form is the files block, a blank line and the text, or `first block` + a blank line + `second block`.
* `spaced arguments` broke on the 11 wires that carry arguments as a string (the Responses wires: `openai`, `grok`, the
  OpenCode Zen and Go responses routes; every chat-completions wire), at the assistant's tool call. It held on the 5 that
  carry them as an object (`claude`, the two OpenCode messages routes, `gemini`, the OpenCode Zen Google route): they
  re-encode the object and drop the spacing themselves.
* Tools and system held on every wire, before and after.
* Cache markers in the post-reload body under `cacheRetention: short`: `cache_control` on `claude`, both OpenCode
  messages routes and Kilo's `anthropic/` models; `prompt_cache_key` on `openai`, `grok`, both OpenCode responses routes
  and `together`; none on `gemini`, the OpenCode chat and Google routes, `huggingface`, Kilo's other models and `ollama`.
  The OpenCode gateways also send an `X-Opencode-Session` header on every route.
* After the change: 16 of 16 wires hold the prefix in all three cases, and tools and system still hold. None is left
  breaking, so step 7a raises no deviation. The harness took about one second for 16 wires and three cases (mostly the
  scripted sessions, not the renderers).

#### Tests, seen failing at `cccef77` (C7), and passing

Run on a scratch export of `cccef77` with the new test files copied in.

* `TestResumeKeepsPrefix` with its five wires (P1a), at the base, `go test -count=1 -run TestResumeKeepsPrefix
  ./acpserver/` exit 1:
  * `mention` and `two text blocks` fail on all five wires, requests 0, 1 and 2: `Messages`, `Responses`, `Gemini
    interactions` and `OpenCode Google route` at item 0, `chat-completions` at item 1 (the system prompt is item 0), e.g.
    `Messages: request 0's item 0 of 2 is not the post-reload request's`;
  * `spaced arguments` fails on `Responses` at item 1 (`request 1's item 1 of 3`, `request 2's item 1 of 5`) and on
    `chat-completions` at item 2 (`request 1's item 2 of 4`, `request 2's item 2 of 6`); the other three hold, as the
    measurement shows;
  * `empty arguments` fails on the same two wires at the same items: at `cccef77` a live `""` is not the reloaded `{}`.
* `TestArgsTakeTheStoresForm`, at the base, exit 1: `the tool ran with "{\"path\": \"x\"}", want "{\"path\":\"x\"}"` and
  `the next request carries the call's arguments ["{\"path\": \"x\"}"], want ["{\"path\":\"x\"}"]`.
* `TestToolArgsAsStored`, at the base: the package does not build, `acpserver/history_test.go:109:17: undefined:
  jsonx.StoredArguments`; its failure is the planted breaks below.
* On the finished phase: `go test -count=1 ./acpserver/ ./agent/ ./jsonx/` passes (exit 0), `TestPromptCarriesFiles`
  included. No existing test other than the one named changed.

#### Planted breaks (C2, C7), each on a scratch copy of the finished phase

A build failure is not counted as a catch; each of these built.

1. Step 4's call (`prompt = asEntry(prompt)`) removed, `TestResumeKeepsPrefix`, exit 1: `mention` and `two text blocks`
   fail on all five wires at item 0 (item 1 on `chat-completions`), requests 0, 1 and 2: `Messages: request 0's item 0
   of 2 is not the post-reload request's`, and likewise `Responses`, `Gemini interactions`, `OpenCode Google route`
   and `chat-completions` (`request 0's item 1 of 3`).
2. Step 3's call removed (`ev.Args = jsonx.StoredArguments(ev.Args)`): `TestResumeKeepsPrefix/spaced_arguments` exit 1,
   `Responses: request 1's item 1 of 3 is not the post-reload request's` (and request 2, item 1 of 5) and
   `chat-completions: request 1's item 2 of 4` (and request 2, item 2 of 6); `empty_arguments` the same two wires;
   the other three wires hold; `TestArgsTakeTheStoresForm/spaced_object` exit 1, `the tool ran with
   "{\"path\": \"x\"}", want "{\"path\":\"x\"}"`, and `/empty` exit 1 (`the tool ran with "", want "{}"`).
3. `StoredArguments` built on `encoding/json`'s `Compact`, `TestToolArgsAsStored` exit 1:
   `TestToolArgsAsStored/escaped_slash` fails, `StoredArguments("{\"path\":\"a\\/b.go\"}") = "{\"path\":\"a\\/b.go\"}", want
   "{\"path\":\"a/b.go\"}"`.
4. `StoredArguments` compacting a non-object (any valid JSON, not only an object): with the plan's nine cases it passed
   (Deviation 1). With `spaced array` added it exits 1: `TestToolArgsAsStored/spaced_array` fails,
   `StoredArguments("[ 1 ]") = "[1]", want "[ 1 ]"`.
5. The empty-text rule removed (`if text == "" { return "{}" }`), exit 1 each: `TestToolArgsAsStored/empty` fails,
   `StoredArguments("") = "", want "{}"`; `TestArgsTakeTheStoresForm/empty` fails, `the tool ran with "", want "{}"` and
   `the next request carries the call's arguments [""], want ["{}"]`; `TestResumeKeepsPrefix/empty_arguments` fails on
   `Responses` (`request 1's item 1 of 3`, `request 2's item 1 of 5`) and `chat-completions` (`request 1's item 2 of 4`,
   `request 2's item 2 of 6`). The other plants were run again after P1a and the amendment, with the results above.

#### Step 7: unchanged

* `git diff --exit-code cccef77 -- '*.golden' session/jsonl/testdata acpserver/testdata` exit 0.
* The session files of the `mention` and `spaced arguments` cases (two prompts each, a read call between), written by the
  tree at `cccef77` and by the phase (before P1a and Amendment 1, which change neither case's file), each 8 lines, compared with ids and timestamps masked and the
  temporary directory written `<tmp>`: `mention`: equal, `spaced arguments`: equal; exit 0.

#### Step 8: the assessment for the owner

*Facts from the two tables.* The 16 wires and routes fall into five body shapes: the Messages shape (`claude`, the two
OpenCode messages routes), the Responses shape (`openai`, `grok`, the two OpenCode responses routes), the
chat-completions shape (`huggingface`, `together`, `ollama`, Kilo twice, the two OpenCode chat routes), `gemini`'s
interactions shape and the OpenCode Zen Google route's `generateContent` shape. Both causes sat in gobble's own layers
(`acpserver` and `agent`), upstream of any wire: every wire showed the same first divergence for cause (a), and cause
(b) showed only where the wire sends arguments as a string. No wire's own rendering introduced a divergence of its own,
before or after. Two shapes, `claude` and `openai`, give both the string and the object form of arguments and both
cache-marker styles (`cache_control`, `prompt_cache_key`). They leave uncovered: the chat-completions renderers (7 of
the 16), whose system prompt is an item of the sequence, `gemini`'s and the Google route's renderers, and any later SDK
change that makes one of those renderers re-encode a message or an argument text differently on its own. The harness's
cost: about one second for the 16 wires and three cases, one model id per wire to keep current against the catalog and
the OpenCode route tables, and a name-based marker stripper (`cache`, any case) that can hide a member added to a body
for a non-cache reason.

The two questions: which wires `TestResumeKeepsPrefix` renders permanently, and whether the harness is kept as a test.
Options, without a recommendation, as the coordinator's brief asks:

1. **Keep `claude` and `openai`; the harness stays a recorded one-off.** Cost: nothing added; test time stays near
   0.03 s for the permanent test; no new fixtures. Leaves uncovered: the five-shape gap above. A regression that lives
   in a chat-completions or Gemini renderer is found only by a user's cache missing.
2. **Render one wire per body shape, five in all, and keep the harness as a one-off.** Add `together` or `huggingface`
   (chat-completions), `gemini` and the OpenCode Zen Google route to the permanent test. Cost: three more model ids and
   wire names in `resume_test.go`, about 0.05 s more, and a stripper that handles `prompt_cache_key` and the Google
   shape's `parts` and `contents` sequences. Leaves uncovered: a divergence in one gateway's route table or header
   handling, and a second renderer of an already-covered shape.
3. **Keep the harness as a permanent table-driven test over all 16 wires and routes.** Cost: about 1 s added to the
   `acpserver` package on every run, on three hosts and under `-race`; 16 model ids tied to the catalog and the OpenCode
   route snapshot (a catalog or route change then fails a test that is not about the change); the name-based stripper
   lives in the tree. Leaves uncovered: only what the scripted sessions do not exercise (thinking blocks, images,
   compaction).

**The ruling (2026-11-14), verbatim: "One wire per shape, 5 wires".** Option 2, with the harness a recorded one-off: this
section's tables are its record, and no harness is in the tree. P1a is carried out below.

#### P1a record (2026-11-14): one wire per shape

`TestResumeKeepsPrefix` renders each case through `resumeWires`, five `wireCase`s of provider, model id and the end of
the path the request must go to (a model id that a route table later sends elsewhere fails loudly on the path, not
silently by covering one shape twice). Model ids, each the one least tied to a catalog or route table:

| Shape | Provider and model id | Path asserted | Why this id |
| :--- | :--- | :--- | :--- |
| Messages | `claude`, `claude-haiku-5-5` | `/v1/messages` | the provider's own wire for any id |
| Responses | `openai`, `gpt-6-luna` | `/v1/responses` | the provider's own wire for any id |
| chat-completions | `ollama`, `llama3` | `/v1/chat/completions` | a local engine with no catalog, route table or prefix rule: the id is free text (together, huggingface and Kilo each tie the id to a catalog or a prefix) |
| Gemini interactions | `gemini`, `gemini-3.8-flash` | `/v1beta/interactions` | the provider's own wire for a chat model id |
| OpenCode Google route | `opencode-zen`, `gemini-3.5-flash` | `:streamGenerateContent` | the only provider with this route, and its embedded route table gives it to a Gemini id; the path check holds that table to this shape |

`wireShape` now compares everything in the body but the message sequence (`messages`, `input` or `contents`), cache
markers removed, as the measurement did, in place of only `tools`, `system` and `instructions`. Each wire is seen
failing at `cccef77` and on plants 1, 2 and 5 as quoted above, at the items the measurement table names. The files this
adds to P1's list: none (`acpserver/resume_test.go` is P1's).

#### Gates (C5)

Hosts are named by placeholder: host L is a Linux target, host L2 a second Linux target, host W the Windows laptop run
natively. Every run was `run-on-host`, one at a time, queued. The gates ran twice: first on the build before P1a and
Amendment 1 (round 1), then, after the last code change, again (round 2). Round 1 is kept because it is where
Deviation 2 was found.

**Round 1 (before P1a and Amendment 1).**

1. `make pre-add-check` on the eight changed `.go` files: exit 0.
2. `make preflight` on host L: first run exit 2, `acpserver/resume_test.go:144:1: cognitive complexity 53 of func
   TestResumeKeepsPrefix is high (> 50) (gocognit)`, in this phase's own new test. Resolution: the test's session
   script and its prefix check became two helpers, `resumedRequests` and `checkPrefix`, with no behaviour change and no
   `nolint`. Second run exit 0, 438 s.
3. `make race` on host L: exit 2, a data race outside this phase (Deviation 2).
4. `go test -count=1 ./...`: host L exit 0 (38 packages `ok`, 32 s), host W exit 0 (38 `ok`, 56 s).
5. `-race ./...` on hosts L and W: not run (Deviation 2).

**Round 2 (after P1a and Amendment 1, the last code change).**

6. `make pre-add-check` on the eight changed `.go` files: exit 0 (`go-precheck: 8 file(s) clean (gofmt, golint,
   govulncheck)`); `golangci-lint run ./acpserver/... ./agent/... ./jsonx/...` on the session host: 0 issues.
7. `go test -count=1 ./acpserver ./agent ./jsonx` on the session host: exit 0.
8. `CGO_ENABLED=1 go test -race -count=1 ./acpserver ./agent ./jsonx` on host L2: exit 0 (`ok acpserver 54.9s`, `ok
   agent 2.8s`, `ok jsonx 1.0s`). Only these three packages, per the owner's ruling on Deviation 2.
9. `make preflight` on another host: exit 0 on host L2, after `pick-host`, 746 s with its queue wait; `golangci-lint` 0 issues, records and markdownlint clean. Run after the last code change, and before this record gained these lines.
10. The race gates, on hosts whose race detector is sound (Deviation 2's resolution), all after the last code change
    and with no code changed since. Host L2 is the Linux machine itself, which `run-on-host --need linux` cannot be told
    to use: `pick-host` chooses between it and the laptop's WSL2 distro by score, and the two were within 0.003 of each
    other. A wrapper started `run-on-host`, read the `target=` line, and killed and retried any run that landed on the
    distro (four landings killed within seconds, before any command ran); only a run whose target was the Linux machine
    was kept, so no `-race` result here is from a distro.
    * `make race` on host L2: exit 0 (39 `ok`, 0 `FAIL`, 0 `DATA RACE`), 3 min 7 s.
    * `CGO_ENABLED=1 go test -race -count=1 ./...` on host L2: exit 0 (38 `ok`, 0 `FAIL`, 0 `DATA RACE`), 4 min 55 s;
      `internal/cli` included.
    * `CGO_ENABLED=1 go test -race -count=1 ./...` on host W, natively through `sh -c` (on the laptop `bash` is WSL's):
      exit 0 (38 `ok`, 0 `FAIL`; `internal/cli` 68.6 s, `acpserver` 50.5 s), 1 min 42 s.
    * `make race` on host M, this Mac, through `run-heavy` (queued behind another agent's SDK gate for about ten
      minutes): exit 0 (39 `ok`, 0 `FAIL`, 0 `DATA RACE`), 11 min 28 s including the wait.

**Deviation 2 (2026-11-14; resolved 2026-11-14, no code change).** `make race` on host L (a WSL2 distro of the Windows
laptop) failed in `internal/cli`, a package this phase does not touch, with data races between goroutines reading
config and files while the test body sets up state (`internal/fsx.ReadRegularFile` and `internal/config.Resolve` against
the test, and `yaml` parsing, from `internal/cli.(*runEnv).defaultTrust`): `--- FAIL` for `TestToolFlagsSelectMCPTools`,
`TestRunEndsListings`, `TestCompactionPromptReachesModel`, `TestPrintSendsACP`, `TestSessionValues` and `TestUpdateJSON`,
8 `WARNING: DATA RACE` blocks in the log; `acpserver`, `agent` and `jsonx` passed under `-race`. It was pre-existing: the
same `-race` run of `./internal/cli/` on host L at base `cccef77`, with none of this phase's files, failed too, with a
different set of failing tests (`TestAuthLogin`, `TestChoiceWiring`, `TestLineCompactThroughTheAgent`,
`TestMainExitCodes`, `TestPrintModeTrust`, `TestProjectSettingsNeedTrust`, `TestToolFlagsSelectResourceTools` and
`TestUpdateIgnoresArchive`) and 9 `DATA RACE` blocks, so it was nondeterministic and independent of this change.

*(2026-11-14, the owner's ruling, "Root-cause it first", and its result. A separate agent root-caused the reports on
scratch clones: 161 reports over 24 runs each name a single goroutine on both sides, which is not a data race, and the
same test binary, byte for byte, is clean on the Linux machine and on Windows; it is red only inside the laptop's WSL2
distro. The root cause is an environment fault of that distro's race detector, not a gobble defect, so nothing lands in
`internal/cli` and there is no fix to wait for. The full evidence is going into a separate record,
`0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md`, which is proposed and not yet committed;
this entry cites it as pending, and the race gates stand on this plan's own runs. The race gates were run on the Linux
machine, Windows and macOS instead of the distro (round 2, item 10): all three exit 0, `internal/cli` included.
`-race` on the three touched packages had already passed on the Linux machine (item 8). The WSL2 distro is not used
for any `-race` gate of this plan.)*

*(2026-11-14, later: a correction, and the record now written. The reports this deviation describes as "data races
between goroutines reading config and files while the test body sets up state" each name one goroutine on both sides,
the access and the previous access alike: no report names a second goroutine, so none is a data race, and none is
test-only sharing. The byte-for-byte check of one test binary was made on the Linux machine; Windows and macOS ran
builds of their own, also clean. The evidence, the verdict and the owner's rulings of 2026-11-14 are in
[0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md](0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md),
now `accepted`, and its plan; it is no longer pending.)*

**Rulings and escalations in P1, in order.** Deviation 1 (the fourth plant's wording; resolved in the phase). The
gocognit failure (fixed in the test). Q12's ruling on the assessment, "One wire per shape, 5 wires" (P1a, carried out in
P1's tree). Amendment 1, "Yes, in this plan" (the empty argument text). Deviation 2 and its ruling, "Root-cause it
first" (resolved: an environment fault of the WSL2 distro's race detector; the race gates passed on the Linux machine,
Windows and macOS).

*Provenance: mainstream model, medium effort. Outcome: P1's code, P1a and Amendment 1 built, seen failing at `cccef77`
and on five planted breaks, passing; every round 2 gate green, the race gates on the Linux machine, Windows and macOS
included; staged, not committed. Escalations: Deviation 1 (resolved in the phase), Deviation 2 (resolved, the
owner's, ruled "Root-cause it first"). Rulings: Q12's "One wire per shape, 5 wires", "Yes, in this plan" for empty
arguments.*

<!-- p1-record-end -->

<!-- p2-record-begin -->
### P2 record (2026-11-14): a failed turn leaves the live session the file's

Run by a mainstream-tier agent at medium effort, from base `feafd65` (P1 is `a3b5c26`; the records commit is
`feafd65`).

**Facts re-read at `feafd65` (C1),** the exits of `Prompt` in `acpserver/agent.go` at that base, and whether the file can
hold entries that live history lacks there:

| Line | Exit | The file can be ahead of live history | Handled |
| :--- | :--- | :--- | :--- |
| `:632` | `settleVoice` cancelled, before any entry | no: nothing is written yet | left as it is |
| `:647` | `sessionProvider` failed, before any entry | no | left as it is |
| `:686` | `writePrompt` failed | yes: the baseline or patch entry may be written | replays |
| `:696` | the user entry failed to append | the prompt is absent from both; a patch entry is in both | replays (no observable change) |
| `:715` | the `gobble.tools` entry failed | yes: the file holds the prompt, live history does not | replays |
| `:757-770` | a model error, compacted or not | yes: the prompt, and every message and result written before the error | replays; the non-compacted branch was `append(s.history, prompt)` at `:766` |
| `:775` | an append failed inside the turn | yes: the entries written before the failed one | replays |
| `:791-792` | `out.err` after `End` | no: `s.history` already holds the turn's messages | left as it is |
| `:802` | the trailing `cancelled(s)` | only if `Run` ended with neither `End` nor an error, which it does only after the consumer stops iterating, and `Prompt` never stops | left as it is |

**The change.** `(*Agent).replayFile(s)` sets `s.history` from `s.state().history` under `a.mu`, as the compacted branch
did. The five exits marked "replays" call it, the model-error exit once for both branches (the non-compacted branch is
now the same replay, step 2). It runs only on a failure.

*(2026-11-14, annotation, by the owner's ruling "Remove the two" under P2a: three exits call it, not five: the model-error
exit, the in-turn append failure and the `gobble.tools` exit. The system-prompt-write and user-entry exits do not, because the
history already equals the file's there.)*

**Tests** (`acpserver/resume_test.go`), each run once in one process and once with a close and a load into a second
`Agent` over the same store before the second prompt, comparing the second prompt's request by roles and block types
and by its message bytes on the `claude` and `openai` wires with the cache markers removed. They compare the messages
only: the two runs are two sessions, whose system prompts name their own directories and ids.

| Test | Scripted | Seen failing at `feafd65` |
| :--- | :--- | :--- |
| `TestFailedTurnHistoryIsTheFiles` | a `read` call, then a 400 on the next model call | `live: [system:text user:text user:text]`, `reloaded: [system:text user:text assistant:tool_call tool:tool_result user:text]` |
| `TestFailedAppendHistoryIsTheFiles` | a store wrapper that fails the append of the first tool result (`failAt`, one shot): the first prompt answers the append failure | `the live request [system:text user:text] lacks the assistant's tool call that the file holds`, `live: [system:text user:text]`, `reloaded: [system:text user:text assistant:tool_call user:text]` |

Both fail at `feafd65` (exit 1) and pass on the fix. `go test` on `acpserver`, `agent` and `jsonx`: exit 0.

**Planted breaks (C2, C7),** each on a scratch copy of the finished phase, each built and failed (exit 1):

1. The model-error exit restored to `s.history = append(s.history, prompt)`: `TestFailedTurnHistoryIsTheFiles`
   fails, `live: [system:text user:text user:text]`, `reloaded: [system:text user:text assistant:tool_call
   tool:tool_result user:text]`; the other test passes.
2. The in-turn append-failure exit left without the replay: `TestFailedAppendHistoryIsTheFiles` fails, `the live request
   [system:text user:text] lacks the assistant's tool call that the file holds`; the other test passes.

**Not exercised by a test, and why.** The replays at `:686`, `:696` and `:715`. At `:686` and `:696` the file and live
history agree after a failed append, so no request differs; at `:715` the file holds the prompt that live history
lacks, but a `gobble.tools` entry is written only for a deferred tool of a session-start family, which the test
sessions do not have (a default session's file holds only `gobble.prompt` custom entries). They are the same one-line
call as the exits that are tested, and cost nothing when the file and memory agree. No plant is claimed for them.

*(2026-11-14, annotation, by the owner's ruling "Remove the two" under P2a: the replays at `:686` and `:696` of the base,
the system-prompt-write and user-entry exits, were removed, because the file cannot be ahead there: at most one append
precedes each exit and memory is updated only after a write succeeds. `Prompt` replays at the in-turn append failure, the
model error and the `gobble.tools` exit, not at "the three append failures before the turn". The third, `:715`, is
kept and tested (`TestFailedPreTurnAppendHistoryIsTheFiles/tools_entry`, seen failing with its replay removed). The table
above and the paragraph above stay as they were written.)*

**Cost (step 5).** `s.state()` for the tests' sessions of 2 and 3 messages took 10.9 to 19.0 microseconds on the session
host, measured by a timer in a scratch copy and not kept. No budget is set.

**Goldens.** `git diff --exit-code HEAD -- '*.golden' session/jsonl/testdata acpserver/testdata` exit 0.

#### Gates (C5)

Hosts by placeholder: L2 is the Linux machine itself, W the Windows laptop run natively, M this Mac. Every gate ran
after the last code change (the one-line `unconvert` fix in `resume_test.go`), and no code changed after it. Times are UTC,
queue waits included.

1. `make pre-add-check` on `acpserver/agent.go` and `acpserver/resume_test.go`: exit 0 (`go-precheck: 2 file(s) clean
   (gofmt, golint, govulncheck)`); `golangci-lint run ./acpserver/... ./agent/... ./jsonx/...` on the session host:
   0 issues, after the fix to the one finding it first made (`unconvert`, `string(m.Content[0].Type)` in `roleSeq`).
2. `go test -count=1 ./acpserver ./agent ./jsonx` on the session host: exit 0.
3. `make preflight` on host L2 after `pick-host`: exit 0, 18:36:11 to 18:48:41 (12 min 30 s); `golangci-lint` 0 issues,
   records and markdownlint clean. It ran before this record gained these gate lines.
4. The race gates, on hosts whose race detector is sound only (0058-MADR; Deviation 2 of P1). Host L2 was pinned by
   starting `run-on-host --need linux --race`, reading the `target=` line, and killing a run that landed on the laptop's
   WSL2 distro (none had to be killed this time):
   * `make race` on host L2: exit 0 (39 `ok`, 0 `FAIL`, 0 `DATA RACE`), 18:48:41 to 18:51:54;
   * `CGO_ENABLED=1 go test -race -count=1 ./...` on host L2: exit 0 (38 `ok`, 0 `FAIL`, 0 `DATA RACE`), 18:51:54 to
     18:54:56;
   * the same natively on host W through `sh -c`: exit 0 (38 `ok`, 0 `FAIL`), 18:54:56 to 18:56:33;
   * `make race` on host M through `run-heavy`: exit 0 (39 `ok`, 0 `FAIL`, 0 `DATA RACE`), 18:56:33 to 18:57:18, no
     queue wait. `make race` does not pass `-count=1`, so 36 of the 39 packages are Go test-cache hits from P1's run on
     the same host with the same inputs; `acpserver` (29.8 s) and `internal/cli` (36.5 s), whose inputs changed, ran.
     The uncached results on hosts L2 and W cover every package under `-race` after this change.

*Provenance: mainstream model, medium effort. Outcome: P2's code and its two tests built, seen failing at `feafd65` and on
two planted breaks, passing; every gate green; staged, not committed. Escalations: none in P2. Rulings carried: P1's
Deviation 2 (the race detector of the WSL2 distro is not used for `-race`).*

<!-- p2-record-end -->

<!-- p2a-record-begin -->
### P2a record (2026-11-14): the pre-turn replays

Run by a mainstream-tier agent at medium effort, from `124213c`.

**Facts re-read (C1).** `Prompt`'s three pre-turn calls of `a.replayFile(s)` at `124213c`, read from `acpserver/agent.go`:
`:698` after the `writePrompt` failure (the `gobble.prompt` baseline or patch entry), `:709` after the user entry's failed
append, `:729` after the `gobble.tools` entry's failed append.

**The finding.** One table test, `TestFailedPreTurnAppendHistoryIsTheFiles`, five cases, was written first: each fails the
named append once, in a prompt that fails before its turn, then compares the live session's history (role, and each
block's type, text, call id, tool and arguments) with the history of a second `Agent` that fresh-loads the file, an absent
file giving none. With each call removed in turn on a scratch copy of `124213c`: the cases for `:698` (`system prompt
baseline`, `system prompt patch`) and for `:709` (`user entry`, `user entry after a patch`) all passed, and `tools entry`
(`:729`) failed with `live: []`, `file: ["user{text|two|||}"]`. The file cannot be ahead at the first two exits: at most one
append precedes each, it failed whole, and memory takes the patch's update only after a write succeeds.

**The ruling, and the options.** The three options put to the owner were "Keep as a safety net", "Keep and prove by double
plant" and "Remove the two". The owner chose **"Remove the two"**, relayed by the coordinator (see P2a's amendments above).

**What changed.** `acpserver/agent.go`: the two `a.replayFile(s)` calls at the system-prompt-write exit and the user-entry
exit are removed, each replaced by a comment saying why no replay is needed there; the `gobble.tools` exit keeps its call;
no other production change; `replayFile`'s comment is still true. `acpserver/resume_test.go`: the test above, with
`preTurnAgent`, `historyOf` and `preTurnFailure` and the existing `failAt` store wrapper. No existing test's result changed:
`go test ./acpserver ./agent ./jsonx` passes.

**Each case seen failing (C7),** on a scratch copy of the finished step, one plant at a time, each built and failed
(exit 1); the cases not named passed:

1. *Memory runs ahead at the system-prompt-write exit:* `s.history = append(s.history, prompt)` before that exit's return.
   `system prompt baseline` fails, `live: ["user{text|two|||}"]`, `file: []`; `system prompt patch` fails, live
   `["user{text|one|||}" "assistant{text|a|||}" "user{text|two|||}"]`, file `["user{text|one|||}" "assistant{text|a|||}"]`.
2. *The patch's update joins the history before its write is known to have succeeded:* the `s.history = append(s.history,
   *update)` block moved above the `w.err` check. `system prompt patch` fails (live holds the `Updated system prompt
   section "project_context"` message, the file does not); `user entry after a patch` also fails, because the update is then
   in the history twice; `system prompt baseline` passes (no update).
3. *Memory runs ahead at the user-entry exit:* the same `append(s.history, prompt)` before that exit's return.
   `user entry` fails, `live: ["user{text|two|||}"]`, `file: []`; `user entry after a patch` fails, with the live
   history holding the prompt that the file lacks.
4. *The `gobble.tools` exit's replay removed:* `tools entry` fails, `live: []`, `file: ["user{text|two|||}"]`; the other
   four pass.

Every one of the five cases fails on at least one plant: `system prompt baseline` on 1, `system prompt patch` on 1 and 2,
`user entry` on 3, `user entry after a patch` on 2 and 3, `tools entry` on 4.

**Not verified.** Nothing is claimed for a failure inside `writePrompt` that leaves a partial write (it adds one entry,
so it cannot).

#### Gates (C5)

Hosts by placeholder: L is a WSL2 distro of the Windows laptop, which a plain suite may use (the owner's note, P1's
Deviation 2); L2 the Linux machine itself; W the Windows laptop run natively; M this Mac. Every gate ran after the last
code change (the two removed calls and the plants' test file), and no code changed after it. Times are UTC, queue waits
included.

1. `make pre-add-check` on `acpserver/agent.go` and `acpserver/resume_test.go`: exit 0 (`go-precheck: 2 file(s) clean
   (gofmt, golint, govulncheck)`); `golangci-lint run ./acpserver/... ./agent/... ./jsonx/...`: 0 issues. A first `gofmt -l`
   named `acpserver/resume_test.go` before the new test's formatting was applied; it is clean.
2. `go test -count=1 ./acpserver ./agent ./jsonx` on host M: exit 0. No existing test's result changed.
3. `make preflight` on host L, a plain gate (`pick-host` chose the distro for it): exit 0, 19:43:27 to 19:51:08; the lint
   reports 0 issues.
4. The race gates on hosts whose race detector is sound only (0058-MADR). Host L2 was pinned by starting `run-on-host
   --need linux --race`, reading the `target=` line, and killing a run that landed on the distro (none had to be killed):
   * `make race` on host L2: exit 0 (39 `ok`, 0 `FAIL`, 0 `DATA RACE`), 19:51:08 to 19:54:19;
   * `CGO_ENABLED=1 go test -race -count=1 ./...` on host L2: exit 0 (38 `ok`, 0 `FAIL`, 0 `DATA RACE`), 19:54:19 to
     19:57:25;
   * the same natively on host W through `sh -c`: exit 0 (38 `ok`, 0 `FAIL`), 19:57:25 to 19:59:07;
   * `make race` on host M through `run-heavy`: exit 0 (39 `ok`, 0 `FAIL`, 0 `DATA RACE`), 19:59:07 to 19:59:53, no queue
     wait. `make race` does not pass `-count=1`, so 36 of the 39 packages are Go test-cache hits (same inputs, same
     host); `acpserver` (30.9 s) and the packages whose inputs changed ran. The uncached runs on hosts L2 and W cover
     every package under `-race` after this change.

**Documentation search (the owner's standing requirement, 2026-11-14).** Searched `README.md`, `docs/README.md`,
`docs/architecture.md`, `docs/guides/*.md`, `AGENTS.md`, every `*.md` outside the 0037 records for "replayFile", "failed
append", "failed turn", "live history", "s.history" and related words, and the doc comments of `acpserver/` and
`agent/`. There is no repository `CHANGELOG.md` (the only ones are inside `.opencode/node_modules`). The doc comments of
the files in this change are corrected here (`Prompt`'s exit comments, `replayFile`'s comment, which is still true). The
outside findings are listed in the hand-off, not edited: `acpserver/doc.go:6-7` says the history is "held in memory until
0002-PLAN Phase 4 persists it", stale since the history is the file's; nothing else states what `Prompt` does on a failed
append or a failed turn.

*Provenance: mainstream model, medium effort. Outcome: P2a done: two replay calls removed by the owner's ruling "Remove
the two", the tools-entry replay kept and tested, five cases each seen failing on a plant; every gate green; staged, not
committed. Escalations: the P2a finding (replays that cannot matter), ruled by the owner, relayed by the coordinator.
Rulings carried: "Those 3 untested replays need tests", "Remove the two".*

#### Addition (2026-11-14): the coordinator's review, and a second gate run

The P2a set above was committed as `45a67a8` before these fixes; they are a further change on top of it.

**What the review found.** (1) The doc comment above `TestFailedPreTurnAppendHistoryIsTheFiles` ended "...and each exit
replays the file", false as built: only the `gobble.tools` exit replays. (2) The `docs/README.md` row for this plan,
marked `complete`, still said "P1 executed ..., awaiting its commit and the owner's ruling on step 8's assessment", "P2
executed ..., awaiting its commit" and "P2 every error exit replays the file". Reading the five staged files for the same
assumption found two more statements with no dated note beside them: the 0037 MADR's D43, "every error exit sets
`s.history` from `s.state()`", and the P2 record's "The change", "The five exits marked 'replays' call it".

**What changed.** The test's comment now says what the test checks and why it holds at each exit: at the first two exits
the history already equals the file's, so those cases check that equality and fail when the history is let run ahead; at the
`gobble.tools` exit the replay is what makes them equal, so that case fails without it. The README row says P1 was committed
as `a3b5c26`, the step-8 assessment was ruled "One wire per shape, 5 wires" and carried out as P1a, P2 was committed as
`124213c` with replays at the model-error, in-turn append and `gobble.tools` exits only, the tests were seen failing at
`cccef77` and `feafd65`, and P3 closed. D43 and "The change" carry a dated annotation. The comments of `preTurnFailure`,
`preTurnAgent` and `historyOf` make no such assumption and are unchanged.

**The pause.** The work stopped for a quota pause on 2026-11-14 while these fixes were being made, and resumed on the
owner's word. On resuming, the tree was read: HEAD `45a67a8`, nothing staged, four files with unstaged changes
(`acpserver/resume_test.go`, `docs/README.md`, the 0037 MADR, this plan), every hunk whole and none half-applied.

**The gate run that counts.** Every file change was made by 20:03:11 UTC; this run started after it and its full output and
exit statuses are held. Hosts by placeholder as above; UTC, queue waits included:

1. `make pre-add-check` on `acpserver/resume_test.go`: exit 0 (`go-precheck: 1 file(s) clean (gofmt, golint,
   govulncheck)`); `golangci-lint` on `acpserver`: 0 issues; `go test -count=1 ./acpserver`: exit 0.
2. `make preflight` on host L (a plain gate): exit 0, 20:03:49 to 20:11:27.
3. `make race` on host L2 (the Linux machine itself, `target` checked): exit 0 (39 `ok`, 0 `FAIL`, 0 `DATA RACE`), 20:11:27 to
   20:14:34.
4. `CGO_ENABLED=1 go test -race -count=1 ./...` on host L2: exit 0 (38 `ok`), 20:14:34 to 20:17:44.
5. The same natively on host W through `sh -c`: exit 0 (38 `ok`), 20:17:44 to 20:19:20.
6. `make race` on host M through `run-heavy`: exit 0 (39 `ok`, 0 `FAIL`) in the run above at 20:19:20 to 20:20:56, which may
   have run after the pause began and so is not relied on; it was run again after the resume, exit 0 (39 `ok`, 0 `FAIL`,
   0 `DATA RACE`; `acpserver` ran uncached, 29.5 s, the rest from the test cache), 22:42:36 to 22:43:23, no queue wait.
   This second run is the one that counts for host M.
7. Record checks after the last record change: exit 0 each (see the hand-off).

*Provenance: mainstream model, medium effort. Outcome: the review's two findings and two further stale statements fixed or
annotated; every gate green after the last file change; staged, not committed. Escalations: none. Rulings carried: "Remove
the two", and the owner's "resume".*

<!-- p2a-record-end -->

<!-- p3-record-begin -->
### P3 record (2026-11-14): close-out

Run by a mainstream-tier agent at medium effort, from `124213c` (P1 is `a3b5c26`, P2 `124213c`; P0 and the plan's records
are `cccef77`). This is a records phase: the only runs are the checks it lists and the Verification section's test
commands.

**The plan's status was set `complete` here, put back to `in-progress` the same day, and set `complete` again after P2a,** because the owner's ruling "Those
3 untested replays need tests" adds P2a before this close-out (see P2a). Goal item 5 was not met for one case when this
record was first written; with the owner's "Amend the promise" amendment it is. Every other Goal item and every Verification item is met. The
evidence, item by item:

| Item | Met | Evidence |
| :--- | :--- | :--- |
| Goal 1: every request carries its predecessors as a byte prefix, live and reloaded, cache markers removed | yes | `TestResumeKeepsPrefix`, four cases on five wires (P1, P1a); seen failing at `cccef77` and on plants 1, 2 and 5; passing at `124213c` on hosts M, L2 and W (P3's runs below) |
| Goal 2: the one-time measurement is recorded and the owner has ruled on its assessment | yes | P1's record holds the 16-wire before and after tables; the ruling "One wire per shape, 5 wires" is carried out as P1a |
| Goal 3: after a failed model call or append, the next request is a reload's | yes | `TestFailedTurnHistoryIsTheFiles` and `TestFailedAppendHistoryIsTheFiles` (P2), each seen failing at `feafd65` and on a planted break |
| Goal 4: a prompt is one text block as its entry joins it, and arguments are in the store's form on arrival | yes | `TestPromptCarriesFiles`, `TestArgsTakeTheStoresForm`, `TestToolArgsAsStored`; `a3b5c26` |
| Goal 5: the session files this plan's code writes are byte-for-byte what `512db99` would write for the same turns | yes, as amended for one case | see "Goal 5" and Deviation 3 below |
| Verification 1: the `acpserver` test command passes on all three hosts, and under `-race` | yes | plain: 6 `PASS` (the five tests and `TestArgsTakeTheStoresForm`) on host M, host L2 (the Linux machine) and host W, 18:59 to 19:01 UTC at `124213c`; `-race`: the whole tree on hosts L2, W and M after P2's last code change (P2's record) |
| Verification 2: `TestArgsTakeTheStoresForm` passes | yes | the same runs |
| Verification 3: P1's record holds both tables and the owner's ruling, carried out by P1a | yes | P1's record, "Step 1a and step 7a", "P1a record" |
| Verification 4: every named test has its quoted failure at its base and on each planted break | yes | P1's record (bases `cccef77`, five plants) and P2's record (base `feafd65`, two plants) |
| Verification 5: goldens, `session/jsonl/testdata` and `acpserver/testdata` are identical to P0's commit | yes | `git diff --stat cccef77..HEAD -- '*.golden' session/jsonl/testdata acpserver/testdata` is empty |
| Verification 6: `apidiff` reports one addition, in `jsonx`, and nothing else | yes, in its meaning | `make apidiff` is a no-op below `v1.0.0` (no stable tag); the exported identifiers added to non-test `.go` files from `cccef77` to `124213c` are `jsonx.StoredArguments` alone |
| P3 step 1: goldens | yes | as Verification 5 |
| P3 step 2: the disclosure guard over the plan's commits | yes | the guard run over `cccef77..124213c`, exit 0; this commit is covered when it is pushed |
| P2a: a test for each pre-turn replay, by the owner's ruling "Those 3 untested replays need tests" | yes | P2a record: the tools-entry replay tested and seen failing without it; the two that cannot matter removed by "Remove the two"; five cases each seen failing on a plant |
| P3 step 3: `complete` when every Goal item is met | yes | Goal 5 as amended; P2a done |

**Goal 5, the item that was open.** The session files of `TestResumeKeepsPrefix`'s `mention` and `spaced arguments` cases are
byte-for-byte equal (ids and timestamps masked) between an export of `512db99` and one of `124213c`; no source of
`acpserver`, `agent`, `jsonx` or `session` changed between `512db99` and `cccef77` (`git log` over those paths is empty),
so P1's comparison against `cccef77` was already this one. A third case, a call whose argument text is empty, differs on
the assistant entry's one block:

* at `512db99`: `"name":"read","arguments":{},"gobble":{"rawArguments":""}`
* at `124213c`: `"name":"read","arguments":{}`

The cause is the owner's ruling "Yes, in this plan": the empty text is now `{}` on arrival, so the entry builder sees an
object and writes no `gobble.rawArguments` member. The file reads back as `{}` exactly as before, which is the point of
the change; only the redundant member is gone. The Rollout section's "unchanged in format and in bytes" has the same exception. Nothing else differs.

Options put to the owner, recommended first:

1. **Amend Goal 5 and the Rollout sentence** to say "except a call with empty argument text, whose entry has no
   `gobble.rawArguments` member". Cost: a dated PLAN amendment, no code; this plan then becomes `complete` in P3's
   commit. Uncovered: nothing, since the reader's result is the same.
2. **Keep the bytes exactly:** carry the model's original argument text beside the stored form so the entry builder
   can write the old pair. Cost: a new field on `llm.ToolCallDone` or `agent.Message`, tests, and a re-run of P1's gates,
   for a member that carries no information. It would widen Q13 (a), which the owner ruled as "in the agent, on arrival".
3. **Withdraw the empty-text rule** (the "Yes, in this plan" ruling). Cost: the one-time prefix miss comes back for a
   call with empty arguments on the Responses and chat-completions wires, which `TestResumeKeepsPrefix/empty_arguments`
   measures, and the two empty-argument tests go.

**Deviation 3 (2026-11-14, resolved; recorded by the mainstream model at medium effort).**

* *Found:* the session file of a scripted session with one `read` call whose argument text is empty, written by an export
  of `512db99` and by an export of `124213c` (a throwaway test in each, masked ids and timestamps, a stdlib comparison),
  differs on the assistant entry: `"arguments":{},"gobble":{"rawArguments":""}` against `"arguments":{}`. `mention` and
  `spaced arguments` are equal. No golden or committed test shows it, because no committed test writes a call with empty
  arguments to a file and compares bytes; `TestResumeKeepsPrefix/empty_arguments` and `TestToolArgsAsStored/empty`
  show the in-memory and read-back form.
* *Options put to the owner:* (1) amend the promise; (2) keep the bytes exactly, by carrying the model's raw text beside
  the stored form; (3) withdraw the empty-arguments rule.
* *The owner's choice, relayed by the coordinator:* "Amend the promise", option 1. Goal 5 and the Rollout line carry a
  dated amendment; no code changed. (`complete` followed P2a.)
* *Verified here:* an export of `512db99`, given the file the finished phase wrote for that call (it has no
  `rawArguments` member), gives the arguments back as `{}` through `assistantContent`. *Not verified:* Pi, which was not
  run; the basis is `gobbleMember`'s comment that the member is gobble's own, which Pi ignores.

**Changes in P3.** The MADR's D15 and D42 carry dated "as built" notes with the commits `124213c` and `a3b5c26`. The README
rows are updated, with the amendment and Deviation 3 above, and, after P2a, the status set to `complete`. `0058-PLAN-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md` cites nothing P3 changes, so it
has no note. No code and no test changed.

**Checks.** `scripts/check_records.py --check-all`, `make check-records`, markdownlint, the redact dry run and an
identifier grep over the added lines: all exit 0 or clean, run after this record was written.

*Provenance: mainstream model, medium effort. Outcome: P3's checks run; every Goal and Verification item met, Goal 5 as
amended by the owner's ruling "Amend the promise"; the plan is `complete` after P2a; staged, not committed. Escalations: Goal 5's
empty-arguments case, ruled by the owner (relayed by the coordinator). Rulings carried: "Amend the promise".*
<!-- p3-record-end -->
