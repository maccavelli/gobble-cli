---
status: complete
date: 2026-11-04
associated-madr: "0012-MADR-tool-schemas-descriptions-and-loading.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement native tool deferral through go-llmprovider-sdk

Associated MADR: [0012-MADR-tool-schemas-descriptions-and-loading.md](0012-MADR-tool-schemas-descriptions-and-loading.md)
(accepted 2026-10-08; amended 2026-11-04, proposed). This plan carries out the amendment's D16–D21, with the owner's
answers to Q5 and Q6 (2026-11-04: (a) to each). It is the MADR's second plan.
[0012-PLAN-tool-schemas-descriptions-and-loading.md](0012-PLAN-tool-schemas-descriptions-and-loading.md) is the first,
and is complete. This plan closes item 8 of [gobble-cli#4](https://github.com/maccavelli/gobble-cli/issues/4).

## Goal

When this plan is complete:

- a session's `llm.Request.Tools` is in one order on every prompt between compactions. That order is the direct tools,
  `tool_search`, then the loaded tools in load order, and `tool_search` stays while any tool is deferred (D19);
- every request names its deferred tools in `llm.Request.Deferred`, and its search tool in `llm.Request.Search`. A tool
  result that loaded tools names them in `llm.Content.Loads` (D16);
- `llm/provider` sends them to go-llmprovider-sdk as `Tool.Deferred`, `Tool.Search` and one `ToolLoadItem` per load,
  after the loading result. With nothing deferred, it sends today's request (D16);
- on the wires F21 names native (Claude 4.5 and later, GPT-5.4 and later with a key, OpenCode's routes for them) and on
  Grok, the tools a request sends are the same before and after a load;
- a loaded tool whose loading result the request does not hold is sent as an ordinary tool. That covers start-of-session
  families, loads before a compaction, and sessions recorded before this plan (D17);
- a load is recorded with its call, and a resumed or forked session replays it (D18);
- `WarningDeferralEmulated` is logged at Debug, once per session and model, if 0036-PLAN P2 has landed by P4 (D21).

Every CLI golden, `acpserver/testdata/session.golden` and `session/jsonl/testdata/` is byte-identical throughout. A
session file gains a `callId` only in a `gobble.tools` entry.

## Scope

- **In:** the files each phase names, and nothing else. A file a phase needs and does not name is a deviation (C8).
- **Out:**
  - MCP servers' default `deferred` exposure, which is 0005-PLAN F8 (MADR D7). This plan makes F8's loads native when it
    lands, and does not land it;
  - the ChatGPT sign-in session, whose wire F21 lists as native, and which is 0031-PLAN P4b;
  - a newer go-llmprovider-sdk pin (D20), and `0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md`'s
    P29–P31;
  - reading the SDK's warnings in general, which is 0036-PLAN P2;
  - pushing, tagging, and commenting on or closing #4. Each waits for the owner's word, in the turn it happens.

## Conventions every phase follows

These are [0037-PLAN-session-store-durability.md](0037-PLAN-session-store-durability.md)'s C1–C9, restated so that this
plan stands alone.

- **C1. Start state.**
  - The tree is clean apart from this plan's own records, and any other session's uncommitted records, which no phase
    stages.
  - `HEAD` is the previous phase's commit on `main`.
  - `go version` is `go1.27.3`.
  - `git rev-parse --path-format=absolute --git-path hooks` is the global hooks directory (checked at P0).
- **C2. Scratch work.**
  - Every probe and every deliberately broken copy lives in the session's scratch directory, never in the tree.
  - A broken copy is made as follows:
    1. List the tree's files with `git ls-files -co --exclude-standard`.
    2. Copy those files into `<scratch>/break-<phase>-<n>/`.
    3. Apply the named break there with an exact replacement whose old text is asserted present once: a Python script,
       stdlib only.
    4. Run the named command there.
- **C3. Long output.** Every gate writes its whole output to `<scratch>/<phase>-<gate>.log`, and its exit status is
  taken before any filter.
- **C4. Local gates (G-local),** in this order, each passing:
  1. `make pre-add-check FILES="<every changed .go file>"`;
  2. `make preflight`;
  3. `make race`.

  No phase touches `internal/cli/...`, so `make perf` and `make probe-conhost` are not run.
- **C5. Host gates (G-hosts).** These run on the Linux machine and the Windows laptop, on a copy of the tree, never
  their own checkouts:
  - `go test -count=1 ./...`;
  - `CGO_ENABLED=1 go test -race -count=1 ./...`;
  - `make preflight` up to `verify-build-metadata`, which needs a commit and is recorded as macOS-only;
  - `make check-records markdownlint`.

  Hostnames are never written.
- **C6. Commit.**
  - At the end of each phase, after G-local and G-hosts pass: `git add` of exactly the phase's files, then
    `git commit --no-edit`, on `main`.
  - A commit to `main` is made only on the owner's word, in the turn it happens. Approving this plan is not that word.
  - The phase's execution record, and this plan's `docs/README.md` row, go in the same commit.
- **C7. Records' paths.** No hostname, account name or real machine path goes into a committed file, a log excerpt
  included. Paths are quoted relative to the repository.
- **C8. Deviations.** Any of these stops the phase:
  - a failure the phase does not name as expected;
  - a test assertion outside the ones a phase names as changed;
  - a file outside its list;
  - an API not as written here;
  - an `apidiff` finding other than the compatible additions each phase names. *(2026-11-04: `apidiff` is a no-op
    while the module is below v1.0.0 (`apidiff: no-op below v1.0.0`), so each phase's additions are checked by reading
    its diff of the exported API, and the record says so.)*

  The owner is given:
  - the evidence: file, line, exact failure, and whether it is pre-existing;
  - resolutions, the recommended one first, each with its cost;
  - what doing nothing means.

  No workaround is offered. The chosen resolution is recorded as a dated deviation entry here, and in the MADR when a
  decision or a fact changes, before work continues.
- **C9. Seen failing.** Every new test or case is run against a broken copy (C2) that undoes the phase's change, and is
  seen to fail with the expected message. The execution record quotes that failure.

## Order

P0, then P1 to P4 in order.

- P1 stands alone. It fixes F23 on every wire, and is worth having even if the owner stops after it.
- P2 adds the fields and the adapter. Nothing fills the fields yet, so every request is still today's.
- P3 has the agent fill them. It is the phase that changes what the native wires receive.
- P4 closes out.

This plan does not depend on 0035-PLAN's P29–P31 or on 0036-PLAN. If either lands between phases, the next phase starts
from that commit, as C1 has it.

## Implementation Steps

### P0 — the records

1. The owner has committed the MADR's amendment, this plan, the 0036-PLAN P2 note and the `docs/README.md` rows.
2. Q5 and Q6 are answered: (a) to each, on 2026-11-04, so D16 and D21 stand as written. Q5's answer adds Grok to
   P4's probe.
3. C1's hooks check. Nothing else is written.

### P1 — the list stays put across prompts (MADR D19)

Files: `agent/agent.go`, `agent/agent_test.go`, `acpserver/toolload_test.go`. *(2026-11-04, deviation 1: and
`acpserver/agent.go`.)*

1. **Tests first,** in `agent/agent_test.go`:
   - `TestToolsStableAcrossPrompts`:
     - a first turn loads `b` with `tool_search`, which brings its family, `c`;
     - a second agent is given `Loaded: b, c`;
     - its first request's tools equal the first turn's last request's: `a, tool_search, b, c`.
   - `TestSearchStaysWhenAllLoaded`: an agent given every deferred tool as `Loaded` (`b, c, d`) still sends
     `tool_search`, in the order `a, tool_search, b, c, d`.
2. **Assertions this decision changes,** and no others (C8):
   - `TestLoadedFromConfig` (`agent/agent_test.go:708`) becomes `a, tool_search, d, b`;
   - `acpserver/toolload_test.go:108` and `:125` become `read, tool_search, widget_tool`.
3. **The code,** in `agent.New` (`agent/agent.go:108-148`):
   - `tool_search` is offered after the direct tools and before `Config.Loaded`;
   - it is offered while any tool of `Config.Tools` is deferred and not hidden, loaded or not, rather than only while an
     unloaded one remains (`:141`);
   - *(2026-11-04, deviation 1)* `acpserver` passes `Search` only while a deferred tool outside the session-start
     families is among the session's tools (`acpserver/agent.go`, where it builds `agent.Config`). This is MADR D19 as
     narrowed;
   - the doc comments of `Config.Loaded`, `Agent` and `Offered` say the new order.
4. **Planted breaks** (C9):
   - the old order, with `Loaded` before `tool_search`: `TestToolsStableAcrossPrompts` fails;
   - the old rule at `:141`: `TestSearchStaysWhenAllLoaded` fails;
   - *(2026-11-04, deviation 1)* `acpserver` passing `Search` whatever the tools: `TestToolFlagsSelectResourceTools`
     fails at `internal/cli/mcp_test.go:273`.
5. **Gates and commit.** G-local and G-hosts (C4, C5). `apidiff` reports nothing. Commit (C6).

### P2 — the fields, and the adapter (MADR D16)

Files: `llm/llm.go`, `llm/doc.go`, `llm/provider/provider.go`, `llm/provider/deferral_test.go` (new).

1. **The API, all additions:**
   - `llm.Request.Deferred jsontext.Value` (`json:"deferred,omitzero"`): the specs of the deferred tools this request
     may load, in the same encoding as `Tools`;
   - `llm.Request.Search string` (`json:"search,omitzero"`): the name, in `Tools`, of the search tool;
   - `llm.Content.Loads []string` (`json:"loads,omitzero"`), on a `ContentToolResult`.

   Each doc comment says a provider that reads neither request field sends `Tools` as before.
2. **Tests first,** in `llm/provider/deferral_test.go`:
   - `TestToRequestDeferral`, on `toRequest`:
     - a request with `Deferred` empty gives the same `llmprovider.Request` as today, compared field by field;
     - with `Deferred`, the SDK's `Tools` are the entries of `Tools` not named in it, in order, then each deferred spec,
       with `Deferred` set, and the `Search` tool has `Search` set;
     - after a result whose `Loads` names a deferred tool, one `ToolLoadItem` follows it, with its call id;
     - a `Loads` name that `Deferred` does not hold gives no item. The case is a tool sent as an ordinary one (D17).
   - `TestWireDeferral`, over `wireIDs` (`llm/provider/wire_test.go:41`), through `cannedWire`. Each id gets two
     requests, before and after a `tool_search` result loads `request_permissions`. The expectation for each id is F21's
     form for that id and model:
     - ~~`claude`, `opencode-zen` and `opencode-go`~~ `claude` and `opencode-zen` *(2026-11-04, deviation 2)*, on
       `claude-haiku-5-5`, the Messages native form: the tools array is byte-identical in both bodies, and holds
       `defer_loading`. The second body holds a `tool_reference` to `request_permissions`;
     - `openai`, on `gpt-6-luna`, the Responses native form: the tools array is byte-identical in both bodies, and the
       second body has a `tool_search_output` naming `request_permissions`. *(2026-11-04, deviation 2: so does an
       added case, `opencode-go` on `gpt-6-luna`, OpenCode Go's Responses route.)*;
     - `grok`: the tools array is byte-identical in both bodies, holds `use_tool`, and does not hold
       `request_permissions`. A reply calling `use_tool` with `tool_name` `request_permissions` comes back as an
       `llm.ToolCallDone` named `request_permissions`;
     - every other id, the appended form: `request_permissions` is absent from the first body's tools and last in the
       second's. *(2026-11-04, deviation 2: `opencode-go` on `claude-haiku-5-5` is one, since OpenCode Go sends
       `claude-` over Chat Completions.)*

     An id whose body shows another form is a deviation (C8). The SDK chooses the form, so a mismatch means F21 is
     wrong.
3. **The code:** `toRequest` (`llm/provider/provider.go:426`) as D16 has it. `llm/doc.go`'s mapping gains a paragraph
   on deferral.
4. **Planted breaks** (C9):
   - `Deferred` not set on the SDK's tools: `TestToRequestDeferral` and the native cases of `TestWireDeferral` fail;
   - no `ToolLoadItem`: `TestToRequestDeferral` fails, and so do the `tool_reference` and `tool_search_output` cases;
   - an item for a name `Deferred` does not hold: the SDK refuses with `ErrInvalidRequest`, and `TestToRequestDeferral`'s
     D17 case fails.
5. **Gates and commit.** G-local and G-hosts. `apidiff` reports the three fields as compatible additions to `llm`, and
   nothing else. Commit (C6).

### P3 — the agent fills them, and the session keeps the call (MADR D16–D18)

Files: `agent/agent.go`, `agent/event.go`, `agent/agent_test.go`, `acpserver/agent.go`, `acpserver/history.go`,
`acpserver/toolload_test.go`. *(2026-11-04, deviation 3: and `llm/provider/provider.go` and
`llm/provider/deferral_test.go`.)*

1. **Tests first:**
   - `agent`, `TestLoadRecordedOnResult`. After a load, the request carries:
     - `Deferred`, holding `b`, `c` and `d`;
     - `Search`, set to `tool_search`;
     - `Loads`, naming `b` and `c`, on the `tool_search` result.

     The `Loaded` event carries that result's call id. `Tools` is as P1 has it.
   - ~~`agent`, `TestLoadWithoutResultIsOrdinary`~~ and ~~`TestCompactionMakesLoadsOrdinary`~~ *(2026-11-04, deviation
     3: replaced by the adapter's tests below, since the adapter decides D17)*.
   - `agent`, `TestGrantLoadRecordedOnResult`. A declined call a grant could cover has `Loads` naming the
     permissions family on its result, and its `Loaded` event carries the call's id. *(Added by deviation 3. The
     declined call's load was in step 2 with no test of its own.)*
   - `llm/provider`, in `TestToRequestDeferral` *(deviation 3)*:
     - `loaded without its result`: `request_permissions` is in `Tools` and `Deferred`, and no result carries its
       load. The SDK's tools are `read, tool_search+search, request_permissions, lsp+deferred`, with no
       `ToolLoadItem`. This case covers a compaction, a session-start family and an older session, which all reach
       the adapter alike;
     - `TestDeferralAfterCancelledTurn`, through `claude`'s wire: the `tool_search` call's assistant message is
       marked failed, as a cancelled turn's last is. The body sends `request_permissions` without `defer_loading`,
       and holds no `tool_reference`.
   - `acpserver`, `TestResumeReplaysLoads`, beside `acpserver/toolload_test.go`'s existing case:
     - the `gobble.tools` entry carries `callId`;
     - after a resume, and in a fork, the first request's history has `Loads` on that call's result, and `Deferred`
       holds `widget_tool`;
     - a session whose entry has no `callId`, written by hand as an older gobble writes it, ~~sends `widget_tool` in
       `Tools` and not in `Deferred`~~ has no `Loads` on any result, so the adapter sends `widget_tool` as an
       ordinary tool *(2026-11-04, deviation 3)*.
2. **The code:**
   - `agent.Loaded` gains `CallID` (`agent/event.go:55`).
   - The turn sets `Loads` on the result that loaded (`agent/agent.go:755-765`), for both `tool_search`'s load and the
     declined call's.
   - ~~Each request computes `Deferred` from the session's deferred tools, leaving out those D17 makes ordinary, and sets
     `Search`. This is done again after a compaction.~~ *(2026-11-04, deviation 3)* Each request names every deferred
     tool of the session in `Deferred`, in `Config.Tools` order, and sets `Search` to `tool_search` while it is
     offered.
   - *(Deviation 3)* `toRequest` keeps a tool in both `Tools` and `Deferred` as an ordinary tool, in its place,
     unless a result in the replayed input carries its load (MADR D16, D17 as refined).
   - `acpserver`:
     - `toolsEntry` writes `callId` (`acpserver/agent.go:857`);
     - `replayPath`, or the step that turns its messages into `llm` content (`acpserver/history.go:198-258`), sets
       `Loads` on the result with that id;
     - `loadedTools` is unchanged, so `Config.Loaded` still holds every loaded name.
3. **Planted breaks** (C9):
   - `Loads` not set: `TestLoadRecordedOnResult` fails;
   - ~~every loaded tool kept in `Deferred`: `TestLoadWithoutResultIsOrdinary` and `TestCompactionMakesLoadsOrdinary`
     fail;~~ *(deviation 3)* the adapter sending every `Deferred` tool deferred, whether its load survived or not:
     `loaded without its result` and `TestDeferralAfterCancelledTurn` fail;
   - *(deviation 3)* the declined call's `Loads` not set: `TestGrantLoadRecordedOnResult` fails;
   - `callId` not written: `TestResumeReplaysLoads` fails.
4. **Gates and commit.** G-local and G-hosts. `apidiff` reports `agent.Loaded.CallID` as a compatible addition, and
   nothing else. Commit (C6).

### P4 — the warning, the docs, and close-out (MADR D21)

Files: `docs/architecture.md`, this plan, the MADR's Observed section, `docs/README.md`. If 0036-PLAN P2 has landed,
also the file where it logs warnings, with its test.

1. **D21.**
   - If 0036-PLAN P2 has landed, `WarningDeferralEmulated` is logged at Debug, once per session and model, and a test
     beside P2's `TestWarningLoggedOnce` shows it is not logged at Warn. It is seen failing on a copy that logs it at
     Warn (C9).
   - If P2 has not landed, this step records that, and 0036-PLAN P2's note carries D21.
2. **Docs:**
   - `docs/architecture.md`'s `agent` and `llm` rows describe deferral as built;
   - the MADR gains "Observed — execution results (2026-MM-DD)" for D16–D21;
   - 0005-PLAN F8 gains a note that MCP deferral is native from its first load.
3. **The owner-run probe (optional, the owner's keys, run outside the tree).**
   - **Claude,** on `claude-haiku-5-5`:
     - two requests, one before a load and one after, with `Cache` set, then a third with no deferred tools;
     - the probe reads `CacheReadTokens` on the second request, and `InputTokens` with and without the deferred
       definitions.
   - **Grok,** on the current Grok model (Q5 (a)):
     - a turn whose `tool_search` result loads `request_permissions`;
     - the probe checks that the model's next call through `use_tool` reaches gobble as a `request_permissions`
       call whose arguments pass gobble's validation;
     - it reads `CacheReadTokens` on the requests after the load.
     - A wrong call or no cache read is a deviation (C8). It is answered by the small opt-out amendment Q5's answer
       names, or by an SDK amendment.
   - The MADR's "[unverified]" on input tokens is replaced by what it shows, or stays if the owner does not run it.
4. **Goldens.** `git diff --stat <P0>..HEAD -- '*.golden' acpserver/testdata session/jsonl/testdata` is empty.
5. **#4.** Draft the comment on item 8 here. Posting it waits for the owner's word. #4 stays open while items 5, 6 and 7
   do.
6. **Status.** This plan becomes `complete` when every Goal item is met, and the MADR's amendment becomes `accepted`.
   Commit (C6).

## Verification

Each phase's own steps are its acceptance criteria. Across the plan:

- every test P1–P4 names exists, passes on all three hosts and under `-race`, and has its quoted failure on a broken
  copy (C9);
- `TestWireDeferral` shows an unchanged tools array across a load on every native id and on Grok, and the appended form
  on every other id;
- the goldens and `session/jsonl/testdata/` are byte-identical to P0's commit;
- the exported API gains only the four compatible additions, by `apidiff` once the module is v1.0.0 and by reading
  the diff until then (C8's note): `llm.Request.Deferred`, `llm.Request.Search`,
  `llm.Content.Loads` and `agent.Loaded.CallID`.

## Rollout and Rollback

- Each phase is one commit, and reverts on its own with `git revert`, latest first: P3 needs P2's fields.
- Reverting P3 leaves sessions with `callId` in their `gobble.tools` entries. An earlier gobble ignores the unknown field
  and reads the names as it does today, so no session is lost.
- Nothing is pushed by this plan. The owner pushes, after the disclosure guard.
- **What users see change:**
  - P1: nothing visible. After a load, the next prompt keeps the cache it would have lost.
  - P3: on Claude 4.5 and later, GPT-5.4 and later with a key, and Grok, loading `request_permissions` no longer breaks
    the prompt cache, except where F21 says a Responses load is appended.

## Execution record

Approved by the owner on 2026-11-04: "committed, proceed to p1 0012". The records are `1657e76`. Q5 and Q6 were
answered the same day: (a) to each, Q5 with a Grok check in P4's probe.

### P0 record (2026-11-04): the records

- `HEAD` was `1657e76` on `main`, with the tree clean.
- `git rev-parse --path-format=absolute --git-path hooks` gave the global hooks directory, and `go version` gave
  `go1.27.3`.

### Deviation 1 (2026-11-04): P1, `tool_search` offered for tools that load at session start

- **Found.**
  - Steps 1–4 were done, and the pre-add check and `make preflight` passed. `make race` then failed one test outside
    the phase's files: `--- FAIL: TestToolFlagsSelectResourceTools`, with
    `mcp_test.go:273: -t: ["read" "tool_search" "list_mcp_resources"]`.
  - Its session-start cases were run on a scratch copy, with each `Fatalf` made an `Errorf`. Both failed: `:273`, as
    above, and `:282` (`--exclude-tools request_permissions`), whose tools ended in
    `"tool_search" "list_mcp_resources" "list_mcp_resource_templates" "read_mcp_resource"`.
  - This was not pre-existing: it was P1's rule.
- **Cause.**
  - MADR D19 offered `tool_search` while any tool was deferred, loaded or not.
  - The MCP resource tools are deferred but load before the first request (`acpserver/agent.go:885`), so they never
    sit behind a search.
  - That broke `-t`'s promise to send the tools named, and 0005-PLAN F6-5 deviation 1's rule that a session without
    `request_permissions` has no `tool_search`.
- **Decision,** the owner's choice of three put ("Narrow in acpserver"):
  - `tool_search` stays while a deferred tool outside the session-start families is among the session's tools,
    loaded or not. `acpserver` passes no `Search` otherwise, and the agent keeps P1's rule.
  - The two choices not taken:
    - an `agent.Config.StartLoaded` field, the same behaviour with more API;
    - keeping only the order fix, which leaves a cache break after `request_permissions` loads.
- **Records:** MADR D19 is narrowed in place, with the original struck through.
- **Scope:** `acpserver/agent.go` joins P1. `internal/cli/mcp_test.go` is unchanged, and becomes the narrowed rule's
  check: step 4 gains its break.

### P1 record (2026-11-04): the list stays put across prompts

- **Files:** `agent/agent.go`, `agent/agent_test.go`, `acpserver/toolload_test.go`, and, by deviation 1,
  `acpserver/agent.go`.
- **The code:**
  - `agent.New` offers `tool_search` after the direct tools and before `Config.Loaded`, while any tool of
    `Config.Tools` is deferred, loaded or not.
  - The doc comments of `Config.Search`, `Config.Loaded`, `Agent` and `Offered` say so.
  - `acpserver`'s new `searchFor` passes `builtin.ToolSearch()` only while a deferred tool outside the session-start
    families is among the session's tools (deviation 1).
- **Tests first,** run on the code as it stood. Each failed as planned:
  - `TestLoadedFromConfig`: `agent_test.go:709: request tools = ["a" "d" "b" "tool_search"], want ["a" "tool_search"
    "d" "b"]`;
  - `TestToolsStableAcrossPrompts`: `agent_test.go:735: the next prompt's tools`, with `b` sent before
    `tool_search`;
  - `TestSearchStaysWhenAllLoaded`: `agent_test.go:748: request tools = ["a" "b" "c" "d"], want ["a" "tool_search"
    "b" "c" "d"]`;
  - `TestToolLoadRecordedAndReplayed`: `toolload_test.go:108` and `:125`, `request tools = ["read" "widget_tool"
    "tool_search"], want ["read" "tool_search" "widget_tool"]`.
- **Planted breaks** (C2, each on its own scratch copy):
  1. **The old order,** with `Loaded` before `tool_search`.
     - `TestToolsStableAcrossPrompts` failed at `agent_test.go:735`. The next prompt's tools began
       `[{"name":"a",...},{"name":"b",...`, and the last request's began `[{"name":"a",...},{"name":"tool_search",...`.
     - `TestLoadedFromConfig` and `TestSearchStaysWhenAllLoaded` failed with it.
  2. **The old rule,** with `tool_search` only while an unloaded tool remains.
     - `TestSearchStaysWhenAllLoaded` failed: `request tools = ["a" "b" "c" "d"], want ["a" "tool_search" "b" "c" "d"]`.
  3. **`acpserver` passing `tool_search` whatever the tools** (deviation 1).
     - `TestToolFlagsSelectResourceTools` failed: `mcp_test.go:273: -t: ["read" "tool_search" "list_mcp_resources"]`.
- **G-local:**
  - `make pre-add-check` on the four files: `go-precheck: 4 file(s) clean (gofmt, golint, govulncheck).`;
  - `make preflight`: `preflight passed`, with `apidiff: no-op below v1.0.0`;
  - `make race`: exit 0, 38 packages `ok`.
- **G-hosts:**
  - Both hosts got the tree as 832 files, with no `._*` files, on `go1.27.3`.
  - On the Linux machine and on the Windows laptop:
    - `go test -count=1 ./...` exited 0, with 38 packages `ok`;
    - `CGO_ENABLED=1 go test -race -count=1 ./...` exited 0;
    - `make preflight` passed every gate from `gofmt` to `apidiff`, then stopped at `verify-build-metadata`, which
      needs a commit (macOS-only, C5);
    - `make check-records markdownlint` exited 0.
- **Goldens:** none changed.
- **Not done in P1:** nothing D16–D18 or D21 needs. `llm.Request` is unchanged, so every provider receives the tools in
  D19's order and nothing else.

### Deviation 2 (2026-11-04): P2, OpenCode Go has no Claude Messages route

- **Found.**
  - The adapter and the tests of steps 1–3 were written, and `TestToRequestDeferral`, `TestWireDeferralGrokCall` and
    nine of `TestWireDeferral`'s ten ids passed. All of them had failed on the code before step 3.
  - `TestWireDeferral/opencode-go` failed: `messages: the tools changed at the load`. Its bodies held Chat Completions
    tools (`{"function":{...},"type":"function"}`), with `request_permissions` appended after the load.
  - This was not pre-existing: the test is new.
- **Cause.**
  - The SDK has no Claude Messages route on OpenCode Go. Its `heuristicRoute` sends `claude-` to Messages on Zen only,
    and everything else on Go to Chat Completions (`providers/opencode/route.go:116-143`).
  - Step 2's expectation was the plan's error. F21's "OpenCode's messages route" stays true, of Zen.
  - OpenCode Go's route table sends `gpt-6-luna` to `@ai-sdk/openai`, the Responses route, which is native from
    GPT-5.4 on.
- **Decision,** the owner's choice of two put ("Correct + add Go GPT case"):
  - `opencode-go` on `claude-haiku-5-5` expects the appended form;
  - a case is added, `opencode-go` on `gpt-6-luna`, which expects the Responses native form;
  - F21 gains a note saying which gateway routes which family.
  - The choice not taken was to correct the expectation alone, which leaves OpenCode Go's native path untested in
    gobble.
- **Also, before the test was trusted:** its two SDK checks used `llmprovider.Capabilities{}`, whose zero `Tools` is
  `Unsupported`, and refused with `llmprovider: unsupported: the request needs tools`. They now pass
  `Tools: llmprovider.Supported`. That was a defect in the new test, not in an assertion, and nothing was loosened.
- **Scope:** no file is added.

### P2 record (2026-11-04): the fields, and the adapter

- **Files:** `llm/llm.go`, `llm/doc.go`, `llm/provider/provider.go`, and `llm/provider/deferral_test.go` (new), as
  listed. No file was added.
- **The API, read from the diff of `llm/llm.go`.** Three fields are added and nothing else changes:
  - `Request.Deferred jsontext.Value` (`json:"deferred,omitzero"`);
  - `Request.Search string` (`json:"search,omitzero"`);
  - `Content.Loads []string` (`json:"loads,omitzero"`).

  `apidiff` printed `apidiff: no-op below v1.0.0` (C8's note).
- **The code:**
  - `toRequest` decodes `Deferred` before the messages, and skips a `Tools` entry that `Deferred` names.
  - It sends each deferred spec after the other tools with `Deferred: true`, and marks the `Search` tool, only when
    something is deferred.
  - After each `FunctionCallOutputItem`, it adds one `ToolLoadItem` for each name in `Loads` that `Deferred` holds.
  - A small `toolSpecs` decoder replaces the inline struct.
  - `llm/doc.go`'s SDK mapping gains a paragraph.
- **Tests first,** on the fields with the adapter unchanged:
  - `TestToRequestDeferral/loaded`: `tools ["read" "tool_search" "request_permissions"], want ["read"
    "tool_search+search" "request_permissions+deferred" "lsp+deferred"]`;
  - `TestToRequestDeferral/a_load_the_request_does_not_defer`: `loads [], want only request_permissions`;
  - `TestWireDeferral`: `the tools changed at the load` for `openai`, `claude`, `grok`, `opencode-zen` and
    `opencode-go`. That last expectation was wrong (deviation 2);
  - `TestWireDeferralGrokCall`: `calls ["use_tool {...}"], want request_permissions with its input`.
  - `TestToRequestDeferral/nothing_deferred` and the appended ids passed before and after the change, as they should,
    since they assert today's behaviour.
- **After the change, and deviation 2:** all pass.
  - `TestToRequestDeferral` passes its three subtests.
  - `TestWireDeferral` passes 11 cases: the ten ids, and `opencode-go/gpt-6-luna`.
  - `TestWireDeferralGrokCall` passes.
- **Planted breaks** (C2):
  1. **`Deferred: false` on the SDK's deferred tools.**
     - `TestToRequestDeferral/loaded`: `tools ["read" "tool_search+search" "request_permissions" "lsp"]`.
     - Every `TestWireDeferral` case and `TestWireDeferralGrokCall` failed with `llmprovider: invalid request: input 3
       loads "request_permissions", which is not a deferred tool of the request`. That includes the added
       `opencode-go/gpt-6-luna`.
  2. **No `ToolLoadItem`.**
     - `TestToRequestDeferral/loaded`: `input [...] has no item after s1's result`.
     - `TestToRequestDeferral/a_load_the_request_does_not_defer`: `loads [], want only request_permissions`.
     - `TestWireDeferral`: the messages cases (`claude`, `opencode-zen`) failed on the missing `tool_reference`, and
       the responses cases (`openai`, `opencode-go/gpt-6-luna`) on the missing `tool_search_output`.
     - The appended cases failed too, since with no load the SDK appends nothing.
  3. **An item for a name `Deferred` does not hold.**
     - `TestToRequestDeferral/a_load_the_request_does_not_defer`: `loads ["request_permissions" "read"], want only
       request_permissions`.
     - `TestToRequestDeferral/nothing_deferred`: `with Search and Loads but nothing deferred:`. It failed before the
       SDK's check was reached, so the step 4 wording ("the SDK refuses") describes where it would have failed next.
- **G-local:**
  - `make pre-add-check` on the four files: `go-precheck: 4 file(s) clean (gofmt, golint, govulncheck).`;
  - `make preflight`: `preflight passed`;
  - `make race`: exit 0, 38 packages `ok`.
- **G-hosts:**
  - Both hosts got 833 files, with no `._*` files. The Windows laptop is on `go1.27.3`.
  - On the Linux machine and on the Windows laptop:
    - `go test -count=1 ./...` exited 0, with 38 packages `ok`;
    - `CGO_ENABLED=1 go test -race -count=1 ./...` exited 0;
    - `make preflight` passed `gofmt` to `apidiff`, then stopped at `verify-build-metadata` (C5);
    - `make check-records markdownlint` exited 0.
- **Not done in P2:** nothing fills the new fields yet. Every request gobble sends is still the one P1 sends, which
  P3 changes.

### Deviation 3 (2026-11-04): P3, the agent cannot see what `replay` drops

- **Found** while reading for step 2, before any of P3's code was written.
  - `llm/provider`'s `replay` drops a failed assistant message and its calls' results (`llm/provider/replay.go:49-74`).
  - A cancelled turn's last assistant message is marked failed (`acpserver/choice.go:244-248`).
  - Suppose a turn's last step ran `tool_search`, its result loaded a tool, and the turn was then cancelled. The load is
    recorded, and the agent's history holds the result with `Loads`. Under step 2 as written, the agent would send the
    tool deferred, and the SDK would receive no load. On the native wires and Grok the model could not see the tool,
    while gobble held it as loaded.
  - This was not pre-existing: it was P3's design.
- **Decision,** the owner's choice of two put ("Decide in the adapter"):
  - `toRequest`, which works on the history after `replay`, sends a tool that is in both `Tools` and `Deferred` as
    deferred only when a surviving result carries its load. Otherwise the tool stays an ordinary tool, in its place.
  - The agent names every deferred tool in `Deferred`, with no scan of its history. A compaction, a session-start
    family, an older session and a cancelled turn all reach the same rule.
  - The choice not taken was for the agent to copy `replay`'s rule about failed messages, which would duplicate it in
    `agent`, and break again without notice when `replay` changes.
- **Records:** MADR D16's adapter bullet and D17 are refined in place, with the originals struck through.
- **Scope:**
  - `llm/provider/provider.go` and `llm/provider/deferral_test.go` join P3.
  - The agent's two D17 tests become adapter tests, and `TestGrantLoadRecordedOnResult` is added.

### P3 record (2026-11-04): the agent fills them, and the session keeps the call

- **Files:**
  - as listed: `agent/agent.go`, `agent/event.go`, `agent/agent_test.go`, `acpserver/agent.go`, `acpserver/history.go`
    and `acpserver/toolload_test.go`;
  - by deviation 3: `llm/provider/provider.go` and `llm/provider/deferral_test.go`. No other file was added.
- **The API, read from the diff.** One exported field is added: `agent.Loaded.CallID string`. Everything else P3 adds
  is unexported (C8's note).
- **The code:**
  - **`agent.New`** encodes every deferred tool's spec, loaded or not, in `Config.Tools` order, once.
    - Each request carries that as `Deferred`.
    - It sets `Search` to `tool_search` while `tool_search` is offered and something is deferred.
  - **`runTools`** names a load on the result that made it (`Loads`), for `tool_search`'s load and for a declined
    call's, and emits `Loaded{Names, CallID}`.
  - **`toRequest`** sends a tool that is in both `Tools` and `Deferred` as deferred only when a `ToolLoadItem` it sends
    loads it. Otherwise the tool stays ordinary, in its place.
    - `Search` is cleared when nothing goes deferred, so that request is today's.
  - **`acpserver`:**
    - `toolsData` gains `callId` (`omitzero`). `toolsEntry` takes the call id: the agent's for a `Loaded` event, and
      `""` for a session-start load.
    - `replayPath` collects each entry's call id and names into `state.loads`, and `addMessage` sets `Loads` on that
      call's result. `loadOf` skips an entry with no call id.
- **Tests first,** run on the code as it stood, with only `agent.Loaded.CallID` added so the tests compiled. Each
  failed as planned:
  - `TestLoadRecordedOnResult`: `agent_test.go:785: request 0: Deferred [], Search ""; want b, c, d and tool_search`;
  - `TestGrantLoadRecordedOnResult`: `agent_test.go:832: the declined call's result {...}`, with no `Loads`;
  - `TestToRequestDeferral/loaded_without_its_result`: `tools [... "request_permissions+deferred" ...], want [...
    "request_permissions" ...]`;
  - `TestDeferralAfterCancelledTurn`: `deferral_test.go:251`, with `request_permissions` sent deferred;
  - `TestResumeReplaysLoads`: `toolload_test.go:102: gobble.tools call ids [""], want s1`;
  - `TestReplayLoadsByCall/with_its_call`: `toolload_test.go:154: c1's result loads [] (found true), want
    ["widget_tool"]`.

  `TestReplayLoadsByCall/from_before_D18` passed before and after, as it asserts today's behaviour.
- **After the change:** `agent`, `acpserver`, `llm/...` and `internal/cli` pass. The tree needed one `gofmt -w`, of
  `acpserver/history.go`'s struct alignment.
- **Planted breaks** (C2, each on its own copy):
  1. **`Loads` not set on the result.**
     - `TestLoadRecordedOnResult`: `the search's result loads [], want b and c`.
     - `TestGrantLoadRecordedOnResult` failed with it.
     - `TestResumeReplaysLoads` failed too: `request 2: s1's result loads [] (found true), want widget_tool`.
  2. **The adapter sending every `Deferred` tool deferred.**
     - `TestToRequestDeferral/loaded_without_its_result` failed.
     - `TestDeferralAfterCancelledTurn` failed at `deferral_test.go:251`.
  3. **The declined call's load emitted, but not named on its result.**
     - `TestGrantLoadRecordedOnResult` failed.
  4. **`callId` not written.**
     - `TestResumeReplaysLoads`: `gobble.tools call ids [""], want s1`.
  5. **`addMessage` not setting `Loads`.** This break was added to the plan's four, to hold the rebuild step.
     - `TestResumeReplaysLoads` failed with `request 3: s1's result loads [] (found true), want widget_tool`: only the
       fork, which is rebuilt from the file. The live session's next prompt keeps `Loads` in memory.
     - `TestReplayLoadsByCall/with_its_call` failed with it.
- **G-local:**
  - `make pre-add-check` on the eight Go files: `go-precheck: 8 file(s) clean (gofmt, golint, govulncheck).`;
  - `make preflight`: `preflight passed`;
  - `make race`: exit 0, 38 packages `ok`.
- **G-hosts:**
  - Both hosts got 833 files, with no `._*` files.
  - On the Linux machine and on the Windows laptop:
    - `go test -count=1 ./...` exited 0, with 38 packages `ok`;
    - `CGO_ENABLED=1 go test -race -count=1 ./...` exited 0;
    - `make preflight` passed `gofmt` to `apidiff`, then stopped at `verify-build-metadata` (C5);
    - `make check-records markdownlint` exited 0.
- **Goldens:** none changed. No golden holds a `gobble.tools` entry.
- **What this changes for a user:** loading `request_permissions` no longer changes the tools list on Claude 4.5 and
  later (`claude`, OpenCode Zen), GPT-5.4 and later (`openai`, either gateway's responses route), or Grok.
  - The exception is F21's: a Responses load from a declined call is appended.
  - Every other wire sends the tools in P1's order, as before.
  - The cache saving itself is unmeasured here. P4's owner-run probe reads it.

### P4 record (2026-11-04): the warning, the docs, and close-out

- **D21.** 0036-PLAN P2 has not landed: `git grep -n Warning -- llm/provider agent` finds no SDK warning read. Under
  step 1's second branch, nothing is coded here. 0036-PLAN P2's dated note of 2026-11-04 carries the Debug rule into
  that phase.
- **Docs:**
  - `docs/architecture.md`:
    - the `agent` row gains the list's order, `tool_search`'s rule, and `Deferred`/`Search`/`Loads`;
    - the `llm` row names the fields;
    - the `llm/provider` row says how each wire sends the tools, and when a loaded tool is sent as an ordinary one.
  - The MADR gains "Observed — execution results (2026-11-04)". Its amendment's heading says `accepted`, with a note
    that its earlier "proposed" marks predate the outcome.
  - 0005-PLAN F8 gains its dated note.
- **The owner-run probe.**
  - **Where it is.** Written as `llm/provider/deferral_probe_test.go`, build tag `deferral_probe`. It was placed in a
    scratch copy of the tree, never in the tree.
  - **What it checks.**
    - `TestProbeClaudeDeferral` sends three requests, with a system prompt long enough to cache: before a load, after
      it, and with no deferred tools. It logs the usage of each, and the input tokens the deferred definitions cost. It
      fails when the request after the load reads no cache.
    - `TestProbeGrokDeferral` sends a history whose `tool_search` result loaded `request_permissions`. It checks the
      model's next call comes back as `request_permissions` with that tool's own arguments, and that the repeated
      request reads the cache.
  - **What was run.**
    - `go vet -tags deferral_probe ./llm/provider/` exited 0.
    - Run with `ANTHROPIC_API_KEY` and `XAI_API_KEY` unset, both tests `SKIP`, through `llmtest.LiveCredential`.
  - **Not exercised:** it has not run against a service, and so has not been seen to fail (C9). That needs the owner's
    keys. Until it runs, the MADR's "[unverified]" on input tokens stands.
  - *(2026-11-04, after the commit.)* The owner ran it twice.
    - **The first run failed,** both times on the probe:
      - `TestProbeClaudeDeferral`: `llm: incomplete response: llmprovider: incomplete response: length`, from its
        256-token cap;
      - `TestProbeGrokDeferral`: `unknown object member name "network"`, from an untagged struct under JSON v2's
        case-sensitive matching. Grok's call itself was right.
    - With both fixed in the scratch copy, the second run passed. The MADR's Observed section has the numbers, and the
      one result the run does not explain.
    - *(Later the same day.)* A third run gave each prompt a fresh nonce, and repeated the request after the load.
      It passed. Claude's request after the load read the whole prompt the one before it wrote (8,494 tokens), and
      wrote only the 258 that came after, which bears out D16. The MADR's "The probe's third run" has the numbers.
- **Goldens:** `git diff --stat 1657e76..HEAD -- '*.golden' acpserver/testdata session/jsonl/testdata` is empty, and so
  is the working tree's.
- **#4,** the comment drafted for item 8. Posting it waits for the owner's word.

  > Item 8, native tool deferral, is done, under `0012-MADR-tool-schemas-descriptions-and-loading.md`'s amendment of
  > 2026-11-04 (D16–D21), carried out by `0012-PLAN-native-tool-deferral.md`.
  >
  > - **The list stays put** (2ff819e). A loaded tool keeps its place after `tool_search` at the next prompt, and
  >   `tool_search` stays once everything has loaded. Before, each mid-session load broke the prompt cache a second
  >   time on every provider.
  > - **The SDK's native forms** (4ae9290, 3a50558).
  >   - Every request names its deferred tools and search, and a result names what it loaded.
  >   - go-llmprovider-sdk then sends a load in each wire's form, and the tools list does not change: `defer_loading`
  >     and `tool_reference` on Claude 4.5 and later (and OpenCode Zen), the client `tool_search` on GPT-5.4 and later
  >     (and both OpenCode gateways' Responses route), and `use_tool` on Grok.
  >   - Every other wire appends, as before.
  > - **A load the history no longer carries is sent as an ordinary tool.** That covers a compaction, a cancelled
  >   turn, a session-start load, and a session from before.
  > - **Loads are saved with their call,** so a resumed or forked session loads natively too.
  > - **Today this covers `request_permissions`,** the one tool loaded mid-session. MCP servers' tools join when
  >   0005-PLAN F8 defers them.
  > - **The SDK's `deferral_emulated` warning** will be logged at Debug once 0036-PLAN P2 reads warnings.
  >
  > #4 stays open for items 5 (waiting on the SDK), 6 and 7 (0036).
- **Status:** every Goal item is met. The last is conditional on 0036-PLAN P2, and is carried by its note. This plan is
  `complete`, and the MADR's amendment is `accepted`.
