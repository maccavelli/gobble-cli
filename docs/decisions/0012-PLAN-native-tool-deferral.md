---
status: proposed
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
  - an `apidiff` finding other than the compatible additions each phase names.

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

Files: `agent/agent.go`, `agent/agent_test.go`, `acpserver/toolload_test.go`.

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
   - the doc comments of `Config.Loaded`, `Agent` and `Offered` say the new order.
4. **Planted breaks** (C9):
   - the old order, with `Loaded` before `tool_search`: `TestToolsStableAcrossPrompts` fails;
   - the old rule at `:141`: `TestSearchStaysWhenAllLoaded` fails.
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
     - `claude`, `opencode-zen` and `opencode-go`, on `claude-haiku-5-5`, the Messages native form: the tools array is
       byte-identical in both bodies, and holds `defer_loading`. The second body holds a `tool_reference` to
       `request_permissions`;
     - `openai`, on `gpt-6-luna`, the Responses native form: the tools array is byte-identical in both bodies, and the
       second body has a `tool_search_output` naming `request_permissions`;
     - `grok`: the tools array is byte-identical in both bodies, holds `use_tool`, and does not hold
       `request_permissions`. A reply calling `use_tool` with `tool_name` `request_permissions` comes back as an
       `llm.ToolCallDone` named `request_permissions`;
     - every other id, the appended form: `request_permissions` is absent from the first body's tools and last in the
       second's.

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
`acpserver/toolload_test.go`.

1. **Tests first:**
   - `agent`, `TestLoadRecordedOnResult`. After a load, the request carries:
     - `Deferred`, holding `b`, `c` and `d`;
     - `Search`, set to `tool_search`;
     - `Loads`, naming `b` and `c`, on the `tool_search` result.

     The `Loaded` event carries that result's call id. `Tools` is as P1 has it.
   - `agent`, `TestLoadWithoutResultIsOrdinary`. With `Loaded: d` and a history that holds no result loading `d`, `d`
     is in `Tools` and not in `Deferred`. Given the history of `TestLoadRecordedOnResult`, the next prompt's request
     keeps `b` and `c` in `Deferred`.
   - `agent`, `TestCompactionMakesLoadsOrdinary`. A compaction during the turn drops the loading result. The next
     request has `b` and `c` in `Tools` and not in `Deferred`, and no `Loads` survives in the compacted history.
   - `acpserver`, `TestResumeReplaysLoads`, beside `acpserver/toolload_test.go`'s existing case:
     - the `gobble.tools` entry carries `callId`;
     - after a resume, and in a fork, the first request's history has `Loads` on that call's result, and `Deferred`
       holds `widget_tool`;
     - a session whose entry has no `callId`, written by hand as an older gobble writes it, sends `widget_tool` in
       `Tools` and not in `Deferred`.
2. **The code:**
   - `agent.Loaded` gains `CallID` (`agent/event.go:55`).
   - The turn sets `Loads` on the result that loaded (`agent/agent.go:755-765`), for both `tool_search`'s load and the
     declined call's.
   - Each request computes `Deferred` from the session's deferred tools, leaving out those D17 makes ordinary, and sets
     `Search`. This is done again after a compaction.
   - `acpserver`:
     - `toolsEntry` writes `callId` (`acpserver/agent.go:857`);
     - `replayPath`, or the step that turns its messages into `llm` content (`acpserver/history.go:198-258`), sets
       `Loads` on the result with that id;
     - `loadedTools` is unchanged, so `Config.Loaded` still holds every loaded name.
3. **Planted breaks** (C9):
   - `Loads` not set: `TestLoadRecordedOnResult` fails;
   - every loaded tool kept in `Deferred`: `TestLoadWithoutResultIsOrdinary` and `TestCompactionMakesLoadsOrdinary`
     fail;
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
- `apidiff` reports only the four compatible additions: `llm.Request.Deferred`, `llm.Request.Search`,
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

Not started. The plan is `proposed`, waiting for the owner's approval. Q5 and Q6 were answered on 2026-11-04: (a) to each, Q5 with a Grok check in P4's probe.
