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
