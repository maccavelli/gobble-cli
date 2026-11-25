---
status: in-progress
date: 2026-11-25
associated-madr: "0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement Sending Each Turn the Model's Known Output Cap, Under a Ceiling and Fitted to the Window, and Owning the Per-Turn Limit on the SDK's Facts

Associated MADR: [0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md)

*(2026-11-24, first draft, written with the MADR's first draft, while Q1 and Q2 were open.)*

*(2026-11-24, second draft, with the MADR's second draft: the owner's answers Q1 "Yes, short live calls (Recommended)", Q2
"16,384, fitted, shown 'assumed' (Recommended)" and Q3 "64,000 unless you set maxTokens (Recommended)"; the review's
findings: Phase 0's red run covers `./internal/cli/render` and `./llm/...`; `TestStats` and `llm/provider/model_test.go:62`
are named; L20's, L23's and V2's plants made catchable; the one estimate, the floor, the cap-caused resend, the ladder and
the credit line added (L35 to L41). Q4 is open: Phase 2's V2 tests are written to its answer.)*

*(2026-11-24, amendment: the owner answered Q4, "Always, gobble's policy (Recommended)"; the MADR is `accepted` with no row
open. V2's test and plant are those of option (1); the rows for options (2) and (3) are struck. This PLAN stays `proposed`
until the owner approves it.)*

*(2026-11-25, deviation, the owner's "gobble: v1.33.0 pin first, then 0072 (Recommended)": Phase 2 is re-planned as the pin
of go-llmprovider-sdk `v1.33.0` and the convergence, in four parts, 2.0 to 2c, before any of 0072; it no longer waits for
0065-PLAN's Phase 2 or for 0066-PLAN's phases, which it builds. See "Deviation (2026-11-25): pin first" and "Phase 2 as
re-planned" below the original Phase 2. The MADR's rows P1 to P23 are its amendment of this date; P6 and P21 wait on Q5 and
Q6, so the parts that build them wait too.)* *(2026-11-25, second: Q5 and Q6 answered; nothing waits. See "Deviation
(2026-11-25, second)".)*

## Goal

After Phase 1, on go-llmprovider-sdk `v1.32.0`, every agent turn and every compaction summary asks for the model's stated
output cap from gobble's catalog, at most 64,000 unless the user set `maxTokens`, held to the room the window leaves by one
estimate the automatic trigger shares; a refusal the cap caused is resent once at the room or cap the service states; and the
user sees the value and why in `/context`, `/session` and `--stats`. After Phase 2, on `v1.33.0`, the same rule reads the
SDK's discovered facts and a stated cap is kept. Every decided row has a test, red at the base where it changes behaviour,
and a plant that turns it red.

Nothing starts before the MADR is `accepted`, this PLAN is approved, and the owner says `proceed`. The agent stages nothing
and commits nothing: each phase ends with its files ready and its gates recorded; the coordinator stages; the owner commits.

## Scope

### In scope

* `llm`: `llm/limit.go` (D13's limit names) and `llm/limitrefusal.go` (`ReadLimitRefusal`, the seven forms of F17).
* `agent`: `Config.OutputLimit`; `Config.Compact`'s tools parameter; D8 (a)'s resend in `next()`, before the overflow case.
* `compaction`: `EstimateContext`; `Model.Window` and `MaxTokensFrom`; the summary's cap per ladder step; D8 (a) in the ladder.
* `acpserver`: one estimator for the trigger and the fit; the per-prompt functions; the last value per session; D8 (c) and
  (d)'s lines; `/context`, `/session`; `_meta.gobble.outputLimit`.
* `acpclient`: `Result.OutputLimit`. `internal/cli`: the `--stats` suffix, live and headless; `render.Stats`.
* `llm/catalog`, `llm/provider`: `Facts.MaxTokensStated`, `Model.MaxTokensFrom`, `AssumedMaxTokens`.
* Records and docs. Phase 2: the bump's use of `CachedLimits`, the SDK's refused-cap path, `Response.OutputLimit`,
  `WithLimitObservations`.

### Out of scope

* The title's cap (D7) until V8. Probing (V10). Any SDK change. The SDK bump itself (0065-PLAN's Phase 2, as amended).
  *(2026-11-25: the bump is now in scope, as Phase 2a, by "gobble: v1.33.0 pin first, then 0072 (Recommended)"; the title's
  cap is V8, decided in 0065's amendment of that date and built in Phase 2c.)*

## Rules for Every Phase

* **Re-ground first.** Re-read every `path:line` the MADR cites at the phase's start commit; record each moved line.
* **Red, then green.** A test of existing behaviour fails at the phase's base on its assertion, never on a compile error. A
  test of a new name takes its red from its plant. Guards pass at the base and are held by their plants.
* **Plants.** One production change each, on a scratch clone, never in the tree, each turning its own row's test red; the
  execution record quotes the failing line. gobble has no `tools/plant.py`; plants are made as
  `0070-PLAN-keep-a-failed-partial-reply-and-log-stream-health-per-call.md` makes them, one recorded edit each.
* **Seams.** `llmtest` scripts and their recorded requests; the compaction harness and its fake catalog
  (`acpserver/autocompact_test.go:44-45`); `acptest.ScriptAgent`; refusals scripted as `llm.APIError` values carrying the
  forms' exact text from go-llmprovider-sdk's records (`window_0088_record_test.go:148-162`, `limit_refusal.go:102-115`).
* **Hosts and heavy runs.** A single package's `go test` without `-race` on the session host. `make pre-add-check FILES="…"`
  through `~/.agents/skills/available-dev-hosts/scripts/run-heavy --`. `make preflight` through
  `~/.agents/skills/available-dev-hosts/scripts/run-on-host` on this Mac, the Linux machine and the Windows laptop, one at a
  time per host, after `pick-host`; `make race` on the Linux machine with `run-on-host --race`, never WSL. A fallback to this
  Mac does not count for another host. Long output to a scratch file, the exit code captured before any filter.
* **Live calls**: Q1's six only (L34), with `KILO_API_KEY`, `OPENCODE_API_KEY` and `TOGETHER_API_KEY` from the environment by
  name, never read, printed or compared; only the status, the finish reason, the token counts and the error's kind kept.
* **Stops**, each a stop and a prompt: a test outside the MADR's rows turns red; a row cannot be met as written; an exported
  name not in D13; a lint finding fixable only by a suppression; the same step failing twice; a third build pass of a phase.
* **Exit criterion of every phase:** no blocker, every row of the phase tested, plants caught, gates recorded, docs rows
  returned.

## Implementation Steps

### Phase 0: re-ground and list the reds (records and a scratch clone only)

1. Re-read the MADR's citations at HEAD (F1 to F18, D1 to D13, every row); record each moved line.
2. On a scratch clone, build a stub of D2, D3 and D6 (the fit in `acpserver`, the shared estimator in the trigger, the
   summary's cap by `FitOutput`) and run `go test ./llm/... ./agent ./compaction ./acpserver ./acpclient/...
   ./internal/cli/...` (which includes `./internal/cli/render`). Expected reds, exactly: `compaction`'s `TestSummaryCaps`
   (L17). Expected compile-only changes, no assertion touched: `agent/compact_test.go`'s fake `Compact` (the tools parameter,
   D4) and `internal/cli/render`'s `TestStats` call (L27). Expected green: every `acpserver/autocompact_test.go` trigger test
   (each count is anchored and clear of its threshold by more than its system text), `TestTitleFirstPrompt`, and
   `llm/provider/model_test.go:62` (16,384 unchanged). Any other red is a stop.
3. Grep `docs/architecture.md`, `docs/guides/`, `README.md` for the output limit, `/context`, `--stats`, `maxTokens` and
   `reserveTokens`; note the lines Phase 1 changes.

Exit: citations re-grounded; the red list equal to the expectation; nothing but scratch changed.

### Phase 1: the interim, on `v1.32.0` (L rows)

"New" means the test names a name this phase adds, so its red is its plant's.

| Rows | Test | File | Red at the base because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| L1, L33 | `TestTurnSendsCap_0071` (subtests: a 32,000 catalog cap → 32,000 `stated`; the six GLM-5.3 models → 64,000 `ceiling`) | `acpserver/outputlimit_test.go` | the request carries 0 | `Config.OutputLimit` left nil; then, apart, the ceiling dropped |
| L2, L3, L4, L35 | `TestFitOutput` (subtests `window`, `floor`, `no_window`, `full`) | `llm/limit_test.go` | new | the margin dropped; the floor 1; an unknown window read as no room; 1 sent when full |
| L2 (turn) | `TestTurnCapHeldToWindow_0071` (a 1,000,000-token anchored session on a 1,048,576 window) | `acpserver/outputlimit_test.go` | 0 at the base | the turn's estimate read without the trailing messages |
| L5, L6, L18 | `TestTurnSendsCap_0071/assumed`; `TestModelsJSONDefaultIsAssumed_0071`; `TestSummaryCaps/unknown` (16,384) | `acpserver`; `llm/provider/model_test.go`; `compaction/auto_test.go` | 0 at the base; new | `AssumedMaxTokens` 0; then `MaxTokensFrom` never `assumed` |
| L5 (guard) | `TestModelsJSON` at `llm/provider/model_test.go:62` (still 16,384) | same | a guard | `defaultMaxTokens` changed |
| L7 | `TestTurnSendsCap_0071/user_above_ceiling` (`modelOverrides` `maxTokens` 100,000 → 100,000 `user`) | `acpserver` | 0 at the base | the ceiling applied to `user` |
| L8 | `TestTurnCapSameAtEveryLevel_0071` | `acpserver` | 0 at the base | `OutputLimit` skipped when `Thinking` is set |
| L9, L41 | `TestOneEstimate_0071` (subtests `anchored_no_system_twice`, `unanchored_counts_tools`: the trigger and the fit read the same value) | `acpserver/outputlimit_test.go` | the trigger adds the system text and no tools | the trigger keeps its own estimate |
| L10 | `TestCapPerCall_0071` (a tool round: two requests, each with its value) | `agent/outputlimit_test.go` | new | computed once per turn |
| L11 | `TestOverflowResendRecomputesCap_0071` | `agent/outputlimit_test.go` | new | the resend reuses the first value |
| L12, L13, L14 | `TestLimitRefusalLine_0071` (`names_field`, `other_invalid`, `fills_context`, `refused_after_resend`) | `acpserver/outputlimit_test.go` | no line | the line omitted; then, apart, on every invalid request |
| ~~L15~~ | ~~`TestCreditRefusalLine_0071` (`quota_402`, `rate_429`, `kilo_503`, `uncapped_call`)~~ *(2026-11-24: replaced by the two rows below, "Move the line into a new field")* | `acpserver/outputlimit_test.go` | no line | the line omitted; then, apart, on an uncapped call |
| L15 (pass 2, before acpext's tag) | `TestCreditRefusalLine_0071` (`quota_402`, `rate_429`, `kilo_503`, `other_503`: the error text byte-identical, no line); `TestPromptQuotaData` unchanged *(2026-11-24, Phase 1b: "unchanged" holds for pass 2, where no line is set; from Phase 1b its expected data is each frame plus the note, "Expect frame plus note (Recommended)", see the deviation below)* | `acpserver/outputlimit_test.go`; `acpserver/errordata_test.go` | a guard | the line appended to the error text, as pass 1 built it |
| L15, L43 (Phase 1b, after `acpext/v0.6.0`) | `TestCreditRefusalLine_0071` gains `outputLimitNote` on each capped failure and none on `uncapped_call` and `other_503`; the terminal client's line after the error | `acpserver`; `internal/cli/limit_test.go` | new | the member not set; then, apart, on an uncapped call; the CLI's line omitted |
| L15 (guard, Phase 1b, 2026-11-24) | `TestPromptQuotaData`: each frame decoded, its expected `OutputLimitNote` set on a quota or rate frame whose call is capped, re-encoded, compared byte for byte | `acpserver/errordata_test.go` | red at Phase 1b's code, on its assertion, before the edit | the note dropped from a capped quota frame; P2 to P4 |
| L45 (Phase 1b, 2026-11-24) | `TestPrintErrorDataNote_0071`: print mode's JSON `errorData` carries the note of a capped credit refusal and the text output prints no note | `internal/cli/print_test.go` | a guard over unchanged code | `finishPrint` writing no `errorData`; then, apart, the note printed in the text output *(file: `internal/cli/limit_test.go`, beside `TestPrintErrorData`; subtests `json` and `text`)* |
| L42 | `TestRoundTrip`'s new frame `prompt-output-limit-note.error.json`; the sorted-members test; `ExampleErrorData_outputLimitNote` | `acpext/roundtrip_test.go`, `acpext/types_test.go`, `acpext/example_test.go` | new | the tag misspelled; then `omitzero` dropped, which changes every existing frame |
| L44 *(2026-11-24)* | `TestStatedCapHeldForTurn_0071` (`overflow_resend`: a refusal states 16,000, an overflow compacts, the resend asks 16,000, not the cap computed again; `next_call`: the call after a tool round asks 16,000; `computed_below`: a computed cap below M is sent as computed) | `agent/outputlimit_test.go` | the later call asks the full cap again | the hold dropped |
| L17 | `TestSummaryCaps`, its cases rewritten: no window, `C` 100,000 → 64,000 and 64,000 `ceiling`; `C` 6,000 → 6,000 and 6,000; a 20,000 window → `R` each | `compaction/auto_test.go:107-142` | the shares (8,000 and 5,000; 6,000 and 5,000) | the 80 % share kept |
| L17 (ladder) | `TestSummaryCapPerLadderStep_0071` (an overflow drops tool results; the next request's cap is recomputed over its smaller prompt) | `compaction` | new | the first step's cap reused |
| L19 | `TestAutoCompactPreviousModel` (unchanged: 4,096 and 2,048) | `acpserver/autocompact_test.go:552-580` | a guard | the session's model's cap for the previous model |
| L20 | `TestTitleCapUnchanged_0071` (a title model with a stated cap of 8,192 → 128) | `acpserver/title_test.go` | a guard | the title sent `FitOutput(C, W, E)` in place of `min(128, C)` (8,192) |
| L21, L22 | `TestCapAfterModelSwitch_0071` (`same_provider`, `other_provider`) | `acpserver` | 0 at the base | the value cached on the session at its first prompt |
| L23 | `TestCapConcurrentSessions_0071` (two sessions at once on models capped 64,000 and 4,096; each request and each `/context` shows its own), under `-race` in the Linux `make race` | `acpserver` | 0 at the base | the last value kept on the `Agent`, not the session |
| L25, L26 | `TestContextShowsOutputLimit_0071` (`before_call`, one subtest per basis) | `acpserver/commands_test.go` | no line | the line omitted; then `window` worded as `stated` |
| L27 | `TestPromptResponseOutputLimit_0071`; `TestResultOutputLimit_0071`; `TestStats` with new cases ("out-limit 64.0k"; the three existing cases' text unchanged with no limit) | `acpserver`; `acpclient`; `internal/cli/render/plan_test.go:20-36` | new; `TestStats` a guard for its old cases | the meta not set; the member ignored; the suffix omitted |
| L28 | `TestPrintStatsShowsOutputLimit_0071` | `internal/cli/print_test.go` | new | headless passes no limit |
| L29 | 0070's `TestModelCallRecord`, `limit` = the request's `MaxTokens` | `llm/provider/record_test.go` | only if 0070 is built first: it asserts 0 | the record writes 0 |
| L30 | `TestNoCallNoOutputLimit_0071` | `acpserver` | new | the meta written with 0 |
| L31, L32 | `make check-records`; `scripts/check_records.py --check-all`; `make markdownlint` | records; `docs/guides/models-json.md` | notes and docs absent | a dated note's link mistyped on a scratch copy |
| L36 | `TestTurnSendsCap_0071/wrong_low_cap` (a 4,096 listed cap → 4,096) | `acpserver` | 0 at the base | the ceiling raised to the window |
| L37 | `TestCapWhenPrepareRefuses_0071` (over the threshold, `Prepare` refuses: the call goes ahead with `full`) | `acpserver` | 0 at the base | `full` sent as 1 |
| L38 | `TestCapSmallReserve_0071` (`reserveTokens` 2,000 at the threshold → 1,024) | `acpserver` | 0 at the base | the floor dropped |
| L39 | `TestReadLimitRefusal` (each of the seven forms, a non-form, a room at or below 0); `TestCapRefusalResentOnce_0071` (Anthropic's `a + b > c` and OpenRouter's window form: one resend at the room before any compaction; a cap form: one resend at `M`) | `llm/limitrefusal_test.go`; `agent/outputlimit_test.go` | new | each form's pattern broken in turn; then the resend skipped, so the OpenRouter form compacts |
| L40 | `TestSummaryCapRefusalResent_0071` (a summary refused in the OpenRouter window form is resent at the room, no tool result dropped) | `compaction` | new | the ladder reads it as an overflow |
| L34 | `TestLive_GLMCapAccepted_0071`, `//go:build live`, six subtests, each one short streamed prompt at the cap gobble sends | `llm/provider/live_glm_cap_test.go` | not a red test: a measurement (status, finish reason, token counts, error kind) | none |

Build, in order:

1. `llm`: `limit.go` (`FitOutput`, the constants, `CapSource`, `LimitBasis`), `limitrefusal.go`; `llm/catalog`'s
   `defaultMaxTokens` from `llm.AssumedMaxTokens`; `Facts.MaxTokensStated`; `llm/provider`'s `MaxTokensFrom` in
   `modelInfoFrom` (`llm/provider/model.go:107-123`).
2. `compaction`: `EstimateContext` (`estimate.go:293-333`); `Model.Window`, `MaxTokensFrom`; `summariseOnce`'s cap per
   request; the ladder's D8 (a) (`compaction.go:379-413`); `replyCap` gone.
3. `agent`: `Config.OutputLimit` in `turn.call`; `Config.Compact`'s tools parameter; D8 (a)'s once-per-call resend in
   `next()` before the overflow case (`agent/agent.go:514-531`), its value used by the next `call()`.
4. `acpserver`: one estimator (`autocompact.go:103-107` changes to it); the per-prompt functions; the last value on
   `liveSession`; `Model.Window` at `compact.go:70`, `autocompact.go:176-180`; D8 (c) and (d)'s lines where the failure is
   answered (`acpserver/agent.go:829-831`); `/context` (`commands.go:330-335`); `_meta.gobble.outputLimit`
   (`agent.go:857-861`), merged with `cleared` on a cancel.
5. `acpclient` (`conn.go:264-272`, `:340`); `internal/cli` (`view.go:473-495`, `print.go:232-237`, `render/status.go:40-49`).
6. Records: D12's dated notes in 0065 (MADR, PLAN), 0066 (MADR, PLAN), 0004 (`:745`), 0070 (MADR, PLAN, wherever it then
   is); `docs/README.md`'s rows for this MADR and PLAN.
7. Docs: `docs/guides/models-json.md` (`maxTokens` sent on every turn and summary, fitted; the 64,000 ceiling and how to lift
   it; the assumed 16,384); `docs/architecture.md`'s `llm`, `agent`, `compaction`, `acpserver`, `acpclient` rows; the
   `reserveTokens` doc (it now also bounds the room a call has); any line Phase 0 step 3 found.
8. L34: run the live test once per model on the session host; record the six results. A refusal or a 402 or 429 at the cap
   is a stop with the evidence.

Gates: `go test ./llm/... ./agent ./compaction ./acpserver ./acpclient/... ./internal/cli/...` on the session host;
`make pre-add-check FILES="<the .go files>"` through `run-heavy`; `make preflight` on the three hosts through `run-on-host`;
`make race` on the Linux machine through `run-on-host --race`; the record linter on the MADR and the PLAN.

Return docs rows: `path:line — fixed` or `path:line — open: <reason>`, then `docs: searched <paths>`.

### Phase 2: the convergence, on `v1.33.0` (V rows), after the SDK's tag

Starts when go-llmprovider-sdk `v1.33.0` is tagged and gobble pins it (0065-PLAN's Phase 2, as amended), after the 0066-PLAN
phases that build discovery, the observation store and the notices (V6). *(2026-11-25: superseded by "Phase 2 as re-planned"
below: the pin is Phase 2a, 0066's phases are 2b and 2c; the table below stands as each V row's test, re-pointed there.)*

1. Records first: the adoption amendments of 0065 and 0066, dated, deciding V7 to V9 with their own rows and tests.
2. Re-ground F13's SDK lines at the tag, and the caller's refused-cap path of the SDK's 0089 amendment; a change is a stop.

| Rows | Test | File | Red at Phase 1's end because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| V1 | `TestTurnCapFromLimits_0071` (a kept cap makes `StreamedOutput` differ from `Output`; a models.json layer gives `user`) | `acpserver` | Phase 1 reads `a.modelOf` | `Output` read for `StreamedOutput`; then the source read as `catalog` |
| V2 (Q4 "Always, gobble's policy (Recommended)") | `TestExplicitLimit_0071` (subtests `turn` and `summary`, each over Together's and OpenCode Go's chat routes and the Messages wire: `OutputLimit.Sent` true, `Basis` `request`, `Tokens` D2's value) | `llm/provider` (in-memory transport) | the field does not exist at `v1.32.0` | `MaxOutputTokens` 0 on the summary; then, apart, 0 on the turn on an optional wire |
| ~~V2 under Q4 (2) or (3)~~ | ~~struck 2026-11-24: Q4 answered (1)~~ | | | |
| V3 | `TestFitOutput`, `TestTurnCapHeldToWindow_0071` with V1's facts | `llm`; `acpserver` | guards | the window read from `Output` |
| V4 | `TestRefusedCapLearned_0071` (the SDK's caller path; and the `CachedLimits` fallback: two requests, the second at the cap; saved; `model.limit.learned`) | `acpserver`, `internal/cli` | no resend at a learned cap | no resend; then nothing saved |
| V5 | `TestRefusalNoStatedCap_0071` | `acpserver` | a guard over L12 and L39 | a resend at the same value |
| V6 | `TestContextShowsSentLimit_0071` (`request`, `chatgpt_none`) | `acpserver` | Phase 1 shows the asked value | the asked value shown |

Build: the per-prompt functions read `catalog.CachedLimits`; D8 (a)'s cap forms take the SDK's path where it reports them;
`/context`, `/session`, `--stats` read the last `Response.OutputLimit`. Gates: as Phase 1.

### Deviation (2026-11-25): pin first

The owner's ruling, relayed by the coordinator, quoted exactly: **"gobble: v1.33.0 pin first, then 0072 (Recommended)"**.
gobble pins go-llmprovider-sdk `v1.33.0` (tagged at `138d52f`) first, with this Phase 2 and the adoption amendments of 0065
and 0066, then builds 0072 on it. The MADR carries the amendment (rows P1 to P23, Q5 and Q6); 0065, 0066 and 0072 carry theirs
of the same date.

* **What moves into this Phase 2:** the pin itself (0065-PLAN's Phase 2, which is not built as written there); 0066-PLAN's
  Phases 1 and 2, as 0066's amendment re-scopes them; 0065's V8 (the title) and the two 0065-PLAN Phase 2 tests that survive
  (R20's and R24's); from 0072, the removal of `WithModelMetadataURL`, `Options.ModelMetadataURL` and `WithoutModelMetadata`
  (its Phase 4 build's first half) and three of its D21 names, `ModelData`, `ParseModelData`, `Options.ModelData` (P3).
* **What is dropped:** 0072's `v1.32.0` listing URL (its M22) is never built; 0065-PLAN's Phase 2 raise cases and its
  `TestAgentTurn_RaisedLimitRefusedResent_0065` (R26) are superseded by 0071 (0065's amendment).
* **Order:** 2.0, 2a, 2b, 2c, each its own pass record and gate; then 0072's Phases 0, 1, 2, 2b (was 4) and 3 on `v1.33.0`
  (0072-PLAN's deviation of this date). Parts that build P6 or P21 wait for Q5 or Q6; the rest do not.

### Phase 2 as re-planned (2026-11-25): pin first

Rules: this PLAN's "Rules for Every Phase", unchanged. gobble still has no `tools/plant.py`; each plant is one recorded
production edit on a scratch clone, as Phase 1's were. The base of 2.0 and 2a is 073fd1f; each later part's base is the
commit that ends the part before it.

#### Phase 2.0: re-ground and list the reds (records and a scratch clone only)

1. Re-read every citation of the MADR (F1 to F26, D1 to D13, every row) at 073fd1f, and the SDK's at `v1.33.0`; record each
   moved line. A changed SDK fact (F13, F19 to F25) is a stop.
2. On a scratch clone of 073fd1f: `go.mod` at `v1.33.0`, P2's deletions and P3's seam as stubs, nothing else. Run `go vet
   ./...` and `go test -count=1 ./...` on every module through `run-heavy`. Expected reds, exactly: P5's
   `TestListFetchesMetadata` (deleted in 2a) and `TestListFullCatalog`'s request count; P8's tests, each "refused before
   sending" on a Messages route (at least F21's five); the P9 and P10 tests do not yet exist. Any other red is a stop, with the
   failing line quoted.
3. Grep `README.md`, `docs/architecture.md`, `docs/guides/models-json.md`, `docs/guides/choosing-a-model.md` and the doc
   comments of `llm/provider` for `ModelMetadataURL`, "metadata fetch", "8192", `opencode.ai/api.json` and `v1.32.0`; note the
   lines 2a to 2c change.

Exit: citations re-grounded; the red list equal to the expectation; nothing but scratch changed.

#### Phase 2a: the pin (P1 to P16)

| Rows | Test | File | Red at 073fd1f because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| P1, P2 | the build: `go vet ./...` on every module at the tag | all | the removed names do not compile at the tag | `go.mod` back at `v1.32.0`: "undefined: catalog.WithMetadata"; then, apart, `provider.go:215`'s option restored: "undefined: llmprovider.WithoutModelMetadata" |
| P3 | `TestListHandsModelData_0071` (`handed`: `Options.ModelData` holding `factDoc` reaches `List`'s ranking and facts; `chat`: the same through `listChat`; `none`: nil hands none; no metadata request in any case) | `llm/provider/list_test.go` | new names | `listWith` drops `o.ModelData`; then, apart, `listChat` drops it |
| P4 | `TestListFullCatalog`, `TestListSourceAndUsable`, `TestListServesLastGood`, `TestListingSearchByFacts`, the `sdkcache_test.go` and `chatserver_test.go` tests, `TestModelsSearchListsFactsFirst`: each hands the bytes it served | `llm/provider`, `internal/cli` | do not compile at the tag (`ModelMetadataURL`) | `listWith` drops `o.ModelData`: `TestListingSearchByFacts` and `TestModelsSearchListsFactsFirst` lose the document's facts *(2026-11-25, second: `chatserver_test.go:122` deletes the field; it served nothing)* |
| P5 | `TestListFullCatalog`: one request, the listing's, with the key; `TestListFetchesMetadata` deleted, its name in the execution record | `llm/provider/list_test.go` | two requests at 073fd1f (the fetch) | `listWith` lists with no credential: "the listing with the key" fails |
| P6 (Q5 (1)) | `TestListingRanksByBuiltInCopy_0071` (production's listing `Options` carry the parsed built-in copy: a listing served in memory ranks as the copy ranks it; the lookups keep the layer; `New` and the wizard get none) | `internal/cli` | new | the copy left off the listing's `Options`; then, apart, handed to `New` |
| P6 (Q5 (2) or (3)) | `TestListingRanksWithoutDocument_0071` (2), or `TestEveryCallHandsBuiltInCopy_0071` (3); only the chosen option's test is built | `internal/cli`, `llm/provider` | new | a copy handed (2); `New` given none (3) |
| P7 | `TestEveryCallNamesLimit_0071` (`turn`, `summary`, `title`: a recording provider; each request's `MaxTokens` at least 1, the title's with its model's cap unknown included) | `acpserver/outputlimit_test.go` | a guard: green at 073fd1f | the title's `maxTokens` 0 when the cap is unknown |
| P8 | each test 2.0 listed, its request given `MaxTokens: llm.AssumedMaxTokens` | as listed | "refused before sending" at the tag | `toRequest` drops `MaxOutputTokens`: each refused again |
| P9 | `TestAgentTurn_SalvageCompleteCalls_0065` (a Chat Completions stream through gobble's adapter: a whole call, then a cut call, `finish_reason` `length`: one `ToolCallDone`, a `tool_call_cut` warning, `Done{StopToolUse}`; the turn runs the call and makes the next model call; the warning logged once at warn; no result for the cut call) | `llm/provider/cut_test.go`, `agent/outputlimit_test.go` | at `v1.32.0` the SDK returns `ErrIncomplete`: the raw error | `stopOf` tests `FinishLength` before calls: `max_tokens`, no tool run |
| P10 | `TestResponsesCutTextStops_0071` (a Responses stream: text, then `response.incomplete`, `max_output_tokens`: the text and `Done{StopMaxTokens}`) | `llm/provider/cut_test.go` | at `v1.32.0`, `llm.ErrIncomplete` | `stopOf`'s `FinishLength` case removed: `end_turn` |
| P11 | `TestReasoningOnlyCutUnchanged_0071` (reasoning deltas, then `length` with no text: `llm.ErrIncomplete`) | `llm/provider/cut_test.go` | a guard | the adapter maps that error to `Done{StopMaxTokens}` |
| P12 | `TestAgentTurn_KiloWindowFormCompacts_0065`, reworded (`room`: Kilo's message stating a room of at least 1: one resend at it, no summary; `over`: the input alone past the window: a summary, then the turn resent); and its kind, `llm.ErrContextOverflow`, through the adapter | `agent/outputlimit_test.go`, `llm/provider/cut_test.go` | a guard for `room`; `over`'s kind is `ErrInvalidRequest` at `v1.32.0` | `limitRefused` reads the overflow before the window form: `room` compacts |
| P14 | `TestDoneCarriesOutputLimit_0071` (a Claude stream asked 32,000: `{32000, true, "request", ""}`; thinking `high` asked 2,000 on Messages: the raised value, F9) | `llm/provider` | new names | `Done.OutputLimit` left zero; then the request's value copied for the wire's |
| P15 | `TestAPIErrorCarriesObserved_0071` (Anthropic's `max_tokens: 64000 > 32000` refusing a caller's 64,000: `Observed` {model, "", "", 32000, a time}; through `WithStreamRetry`; OpenRouter's window form: zero) | `llm/provider` | new names | `Observed` not copied; then copied with a zero `Seen` |
| P16 | `TestModelCallRecordOutLimit_0071` (a completed call's four fields; a call failed before its done event: none); 0070's `no_out_limit` subtest replaced by it, its name in the execution record | `llm/provider/record_test.go` | new | `out_limit` 0 written on a failed call; then `out_limit_basis` omitted |

Build, in order: `go get` the tag and `go mod tidy` (`go.mod`, `go.sum` alone); P2's six sites; P3's seam and its two readers;
P4's and P5's test edits; Q5's option (P6); P8's test edits; P14 to P16 in `llm/provider` and `llm`. Records: the execution
record. Docs: every line 2.0 step 3 found that names the fetch or the field.

#### Phase 2b: discovery and the facts a call reads (P17, V3; 0066 S1 to S9, S22 to S28, S30, S31, S35, S37, S39 to S43)

| Rows | Test | File | Red at 2a's end because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| 0066's rows | 0066-PLAN's Phase 1 tests as its amendment of this date re-points them | as there | as there | as there |
| P17 (V1) | `TestTurnCapFromLimits_0071` (`streamed`: a kept stream cap makes `StreamedOutput` differ from `Output`, and the turn asks `StreamedOutput`; `user`: models.json `maxTokens` 20,000 over a listing of 128,000; `assumed_default`: a model models.json defines without `maxTokens`; `unknown`; `input_only`: Claude's input limit as `W`) | `acpserver`, `llm/provider` | 2a reads `a.modelOf`'s merged `MaxTokens` | `Output` read for `StreamedOutput`; then the source read as `catalog` for Pi's default |
| V3 | `TestFitOutput`, `TestTurnCapHeldToWindow_0071` over P17's facts | `llm`, `acpserver` | guards | the window read from `Output` |

Build: 0066-PLAN Phase 1's code as amended (`provider.Discover` over `catalog.LookupLimits`, the session's wait, `llm.Model.Limits`
with `StreamedOutput`, compaction's window); `modelInfoFrom` fills `Limits` from `catalog.CachedLimits`; `turnLimits.limit`
and `compactionModel` read it.

#### Phase 2c: the convergence and what is shown, kept and told (P18 to P22, V2, V4 to V8; 0066 S10, S12 to S21)

| Rows | Test | File | Red at 2b's end because | Plant |
| :--- | :--- | :--- | :--- | :--- |
| P18 (V2) | `TestExplicitLimit_0071`, as Phase 2's V2 row | `llm/provider` | a guard at the tag (2a made it true) | as Phase 2's V2 row |
| P19 (V4) | `TestRefusedCapLearned_0071` (`observed`: a refusal whose `Observed` is 16,000 under 64,000 sent: two requests, the second at 16,000, basis `refusal`; saved with model, base URL, scope, tokens and time; `model.limit.learned` once; the next prompt's provider and `CachedLimits` given it back) | `agent`, `acpserver`, `internal/cli` | no resend from `Observed`, nothing saved | no resend from `Observed`; then, apart, nothing saved |
| P20 (V4) | `TestRefusalNoStatedCap_0071` (`window`: OpenRouter's form: resent at the room, nothing saved; `unvalidated`: a cap form with no `Observed`: resent, nothing saved) | `agent`, `internal/cli` | a guard over L39 | a saved entry from a window form |
| P21 (Q6 (1)) | `TestSavedCapHoldsLimit_0071` (a cap saved 1 hour ago for the model, base URL and scope: the first call asks it, basis `remembered`; 25 hours ago, or another scope: the computed cap) | `acpserver`, `llm/provider` | new | the remembered evidence ignored; then the scope not compared |
| P21 (Q6 (2) or (3)) | `TestSavedCapFillsGapOnly_0071` (2), or `TestNoObservationFile_0071` (3); only the chosen option's test is built | as above | new | a saved cap applied to a stated model (2); a file written (3) |
| P22 (V5, V6) | `TestContextShowsSentLimit_0071` (`request`; `raised`; `chatgpt_none`); `TestPromptResponseOutputLimit_0071` and `TestStats` gain the `none` basis | `acpserver`, `internal/cli/render` | 2b shows the asked value | the asked value shown; then the raise not named |
| V7 (0066 S10) | `TestNotice_LimitUnknown_0066`, re-worded by 0066's amendment | `acpserver`, `internal/cli` | as there | as there |
| V8 (0065's amendment) | `TestTitleCapFromLimits_0071` (`min(128, C)` with P17's `C`; unknown: 128); `TestTitleCapUnchanged_0071` unchanged | `acpserver/title_test.go` | 2b reads the merged `MaxTokens` | the title reads `Output`, not `StreamedOutput` |
| 0066's rows | 0066-PLAN's Phase 2 tests as its amendment re-points them | as there | as there | as there |

Build: `agent.Config.SaveObservation`; `internal/cli`'s `limit-observations.json` (0066 D7) and `provider.Options.Observations`;
the `Observed` read before L39's in `limitRefused`; Q6's option; `/context`, `/session`, `_meta.gobble.outputLimit` and
`--stats` from `Done.OutputLimit`; 0066's notices and displays; the title.

Gates of every part: as Phase 1's. Docs of 2c: `docs/guides/models-json.md` (the saved caps; what the line says),
`docs/architecture.md`'s `llm`, `llm/provider`, `agent`, `acpserver` and `internal/cli` rows, `README.md`'s state files.

Exit of each part: no blocker, every row of the part tested, plants caught, gates recorded, docs rows returned.

### Deviation (2026-11-25, second): the owner's answers and the review

The owner's answers, relayed by the coordinator and quoted in the MADR's second amendment of this date: Q5 "Built-in copy to
listings only (Recommended)"; Q6 "Hand them in as a caller layer (Recommended)"; case 10 "The service's cap wins, shown as such
(Recommended)"; case 4 "End the turn, keep the text, notify (Recommended)"; case 5 "Discard a cut title (Recommended)"; case 7
"gobble's basis, wire value (Recommended)"; case 9 "Show what was asked, marked (Recommended)". No row is open.

* **Struck in Phase 2a's table:** the `P6 (Q5 (2) or (3))` row; P6 builds Q5 (1)'s `TestListingRanksByBuiltInCopy_0071`
  alone. In Phase 2c's table: the `P21 (Q6 (1))` and `P21 (Q6 (2) or (3))` rows, replaced by P21's and P24's row below.
* **Re-pointed in Phase 2a:**
  * P4: `chatserver_test.go:122` deletes the field (it named a closed address and served nothing); the other sites hand the
    bytes they served.
  * P8: the fix is one line in `wireStream` (`llm/provider/wire_test.go:137`): `req.MaxTokens = llm.AssumedMaxTokens` when
    the request names none, which covers `TestWireDeferral`, `TestDeferralAfterCancelledTurn`, `TestWireDeferralGrokCall`,
    `TestWireLimits`, `TestWireLongWaitIsNotAWindow`, `TestWireOffline`, `TestWireEmptyAnswer`, `TestWireRefusal`,
    `TestWireReplay`, `TestWireCases`, `TestWireRateLimitResets`, `TestWireOverflowSplit`, `TestWireToolResultError` and
    `TestWireStrict`; the five direct `New` sites take the same value. Phase 2.0's expected reds are these tests' Claude and
    OpenCode messages-route cases and the five direct sites; any other red is a stop.
  * P7: one plant per case: `turn`, `acpserver`'s `agent.Config.OutputLimit` left nil; `summary`, the ladder's `FitOutput`
    result replaced by 0; `title`, as written.
  * P9: an agent-side plant beside the adapter's: the turn treats a `tool_call_cut` warning as its end, so the whole call does
    not run.
* **Re-pointed in Phase 2c:** P19's saved caps reach later runs by P24's layer, not `WithLimitObservations`. V8's fixture is a
  title model whose `StreamedOutput` (100, from a kept stream cap) is below 128 and differs from its `Output` (8,192), so the
  plant "the title reads `Output`" (128 sent) is caught.

| Part | Rows | Test | File | Red at the part's base because | Plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 2a | P29 | `TestCutCallAfterText_0071` (`chat`, `messages`: text, then a cut call, `length`: the text, a `tool_call_cut` warning, `Done{StopMaxTokens}`; the turn ends with `turn.stop.max_tokens_tool`'s line over ACP, in the CLI and in headless JSON); `TestOnlyCutCall_0071` (only a cut call: `llm.ErrIncomplete`) | `llm/provider/cut_test.go`, `agent`, `acpserver`, `internal/cli` | at `v1.32.0` the SDK returns `ErrIncomplete`; the key is new | the notice omitted; then `turn.stop.max_tokens`'s line shown; then the text dropped |
| 2a | P30 | `TestTitleCutDiscarded_0071` (a title reply ending `max_tokens`: usage written, no name set) | `acpserver/title_test.go` | the cut title is used today | the stop reason ignored |
| 2a | P35 | `TestPromptResponseOutputLimit_0071/none` (`_meta.gobble.outputLimit` `{0, "none"}`; `acpclient.Result.OutputLimit` holds it) | `acpserver`, `acpclient` | `Tokens > 0` drops it at `acpserver/agent.go:914` and `acpclient/conn.go:353` | each check kept, apart |
| 2b | P33 | `TestTurnCapFromLimits_0071/listing_only` (a model only the cached listing states, behind a key: its cap is read) | `acpserver`, `llm/provider` | new case | `CachedLimits` without `WithTokenSource` |
| 2b | P28 | `TestTurnCapFromLimits_0071/user_over_observed`, `/user_over_saved` (models.json 100,000; the service stated 16,000 in this process, or in a saved entry: 16,000 sent, basis `service`, the note once) | `acpserver` | new cases | the models.json value sent; then the note on every call |
| 2c | P21, P24 | `TestSavedCapLayer_0071` (`lookup`, `limits`, `discover`, `new`: each receives the "service-stated" layer below models.json; an entry 25 hours old, of another base URL or of another scope is not stated) | `llm/provider`, `internal/cli` | new | the layer left off `New`; then off `CachedLimits`; then added after models.json |
| 2c | P25 | `TestSavedCapDroppedWhenExceeded_0071` (a completed call reporting 20,000 output tokens over a saved 16,000: the entry gone; at 15,000: kept) | `internal/cli` | new | no drop |
| 2c | P26 | `TestSavedCapPerProvider_0071` (an opencode-zen cap is not handed to claude) | `internal/cli` | new | the provider id not compared |
| 2c | P27 | `TestAgent_UnreadableObservations_0066/bad_entry` (entries with no provider, no model, 0 tokens or a zero `Seen`: dropped, one warning; the good ones handed) | `internal/cli` | new case | entries not checked: the zero-`Seen` entry handed |
| 2c | P31 | `TestContextShowsSentLimit_0071/sdk_basis_raw` (an SDK basis `fact`: shown `sdk:fact`, logged once at warn) | `acpserver` | new case | shown in gobble's words |
| 2c | P32 | `TestContextShowsSentLimit_0071/failed_asked`, `/loaded_session`, `/forked_session` | `acpserver` | new cases | a failed call shows no line; then a loaded session shows the stored call's |
| 2c | P34 | `TestSummaryCapRefusalResent_0071/observed`; `TestTitleObservedResent_0071`; `TestAPIErrorCarriesObserved_0071/kilo_range` (Kilo's range form through the adapter: `Observed` M); `TestLimitRefusalLine_0071/refused_again_observed` | `compaction`, `acpserver`, `llm/provider` | new cases | the ladder ignores `Observed`; then the title does not resend; then D8 (c)'s check ignores `Observed` |

## Verification

* Every decided row has its test, red at the base or held by its plant, and green after its phase.
* Every plant turns its own test red, its failing line quoted in the execution record.
* `make preflight` passes on the three hosts and `make race` on the Linux machine at the end of each phase.
* An independent review of each build phase finds no blocker.

## Rollout and Rollback

* Rollout: Phase 1 in the next gobble release, a minor by `0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md`'s
  rule; Phase 2 in the release that adopts `v1.33.0`. Tags are the owner's.
* Rollback: a model's cap is lowered or raised with models.json `maxTokens` at once; reverting Phase 1 restores 0 (the SDK's
  8,192), the reported failure.

## Execution record

*(2026-11-24, execution entry.)* The owner approved this PLAN and Phase 1's start, 2026-11-24: "Approve; start Phase 1 now (Recommended)". The 0070 and 0071 pairs are placed in the records; 0070-PLAN stays `proposed` (not yet approved). Phase 1 is built next, each step its own commit, staged by the coordinator after a clean-clone gate and a code review. Nothing has run yet.

### Phase 0, pass 1 (2026-11-24, at `bf816c1`)

* **Step 1, re-ground.** Every gobble `path:line` the MADR and this PLAN cite reads as cited at `bf816c1` (F1 to F18, D3, D4, D12's `0004-MADR-go-module-architecture.md:745`, `internal/config/keys.go:107`, `docs/guides/models-json.md:70-75`, the PLAN's test files and lines). One moved path: the go-llmprovider-sdk paths written `internal/wire/…` (F13, F17, the Rules' "Seams") are under the module's `llmprovider/` directory, `llmprovider/internal/wire/limit_refusal.go` and `llmprovider/window_0088_record_test.go` at `33e6f09`; their lines are as cited.
* **Step 2, the red list.** A stub of D2, D3 and D6 on a scratch clone (`llm.FitOutput`, the shared estimator in the trigger and the turn, each summary request's cap by `FitOutput`), with D4's `Compact` parameter and `render.Stats`' limit argument. `go test -count=1 ./llm/... ./agent ./compaction ./acpserver ./acpclient/... ./internal/cli/...` through `run-heavy` on this Mac: every package `ok` but `compaction`, whose one red is `TestSummaryCaps` (L17), on its assertion: `auto_test.go:139: the reserve's shares: caps [64000 64000], want 8000 then 5000`, `the model's limit: caps [6000 6000], want 6000 then 5000`, `unknown limit: caps [16384 16384], want 0 then 0`. Compile-only changes, no assertion touched: `agent/compact_test.go`'s fake `Compact` (D4) and `TestStats`' call (L27), as expected; and one not listed, below. Green as expected: every `acpserver/autocompact_test.go` test, `TestTitleFirstPrompt`, `llm/provider/model_test.go:62`.
* **Deviation (2026-11-24), compile-only.** `compaction/summary_test.go`'s `TestSummaryLadder` calls `summariseFitting` three times (`:184`, `:218`, `:225`) with the cap argument D6 removes (`replyCap` and the shares go; each step computes its own cap, L17's ladder plant). Resolution taken: the three call sites drop the argument; no assertion changes. It is the same kind as the two compile-only changes this phase lists.
* **Step 3, docs.** `docs/guides/models-json.md:4`, `:72`, `:74`; `docs/architecture.md:110` (`llm`), `:120` (`compaction`, "a summary request capped at 80 percent of `reserveTokens`"), `:128` (`acpserver`); `internal/config/keys.go:108`'s `reserveTokens` doc ("A summary's reply is capped at 80 percent of it"); `docs/README.md:159-160`.

Exit: met; nothing but scratch changed.

### Phase 1, pass 1 (2026-11-24, from `bf816c1`): stopped

Built in the tree, unstaged: steps 1 to 5 (`llm/limit.go`, `llm/limitrefusal.go`, `Model.MaxTokensFrom`, `Facts.MaxTokensStated`, `defaultMaxTokens` from `llm.AssumedMaxTokens`; `compaction.EstimateContext`, `Model.Window` and `MaxTokensFrom`, each request's cap and D8 (a) in the ladder; `agent.Config.OutputLimit`, `Compact`'s tools, D8 (a)'s resend; `acpserver/outputlimit.go`, the shared estimator, D8 (c) and (d)'s lines, `/context`, `_meta.gobble.outputLimit`; `acpclient.Result.OutputLimit`, `--stats`), the `reserveTokens` doc and the schema it generates, and the tests of the Phase 1 table, each green. `acptest.Turn` gains `Meta`, a test fixture's scripted `_meta`, for L27 and L28 (a seam, not product API).

**Stop.** The module's `go test -count=1 ./...` through `run-heavy` on this Mac turns three existing `acpserver` tests red, none named by a row; each red is a decided row's output, and none is pre-existing (green at `bf816c1`, Phase 0's run):

* `TestPromptQuotaData` (`errordata_test.go:216-220`), three of its four frames: the prompt error's `error` now ends with D8 (d)'s line, "…; gobble asked for at most 16,384 output tokens (assumed); some services hold credit or rate against that number; set maxTokens for m1 under modelOverrides in models.json to ask for less", while the test compares byte for byte with `acpext/testdata/prompt-quota.error.json`, `prompt-rate-over-cap.error.json` and `prompt-weekly.error.json`, the acpext contract's frames (L15).
* `TestSession` (`agent_test.go:128`): `testdata/session.golden`'s record 11, the prompt response, now carries `"_meta":{"gobble":{"outputLimit":{"basis":"assumed","tokens":16384}}}` (L27).
* `TestCancelClearsQueue` (`extensions_test.go:330`): the cancelled response's `_meta` is now `{"gobble":{"cleared":{…},"outputLimit":{"basis":"assumed","tokens":16384}}}`, step 4's merge (L27).

Not yet done in this pass: the red-at-base runs, the plants, the records and docs steps 6 and 7, the live calls (step 8) and the gates. The resolution goes to the owner.

### Deviation (2026-11-24): the owner's ruling on the stop, "Move the line into a new field"

Relayed by the coordinator, quoted exactly: **"Move the line into a new field".** D8 (d)'s line goes in a new
`acpext.ErrorData` member, not the error text, which stays byte-identical, so `TestPromptQuotaData` and acpext's example
frames stay as they are. The MADR carries the amendment (D8 (d), D13, L15, new L42 and L43, "Amendment of 2026-11-24").

* **Scope added to Phase 1:** the acpext module, `acpext/types.go` (`OutputLimitNote`, `outputLimitNote`, `omitzero`, in its
  sorted place), `acpext/doc.go`, `acpext/testdata/prompt-output-limit-note.error.json` (new), `acpext/roundtrip_test.go`,
  `acpext/example_test.go` (L42).
* **Split, as 0049-PLAN's P5a and P5b were:** gobble pins `acpext v0.5.0` with no `replace`, so it cannot set the member
  until acpext is tagged. Phase 1 now ends with D8 (d) adding nothing (L15's pass-2 row: the error text byte-identical).
  **Phase 1b**, after the owner's word: (1) the owner pushes `acpext/v0.6.0` (an annotated tag on the commit that carries
  the member; the agent pushes no tag); (2) gobble's `go.mod` moves to `acpext v0.6.0`; (3) `acpserver` sets
  `outputLimitNote` on a capped call's quota, rate, credit or Kilo 503 failure, and `internal/cli` shows it on the line
  after the error; (4) `TestCreditRefusalLine_0071` gains the member's assertions and `uncapped_call`, and
  `internal/cli/limit_test.go` the line's; each with its plant (L15 and L43 rows of the table); (5) the gates of Phase 1.
* **Expected changes named here** (each the decided output of L27 and step 4, nothing more): `TestSession`'s
  `acpserver/testdata/session.golden` record 11 (line 12) carries `"_meta":{"gobble":{"outputLimit":{"basis":"assumed","tokens":16384}}}`;
  `TestCancelClearsQueue`'s expected `_meta` is `{"gobble":{"cleared":{"followUp":["f"],"steering":["s"]},"outputLimit":{"basis":"assumed","tokens":16384}}}`.
  Each test's other assertions are unchanged.

### Phase 1, pass 2 (2026-11-24)

The D8 (d) line is out of `acpserver/outputlimit.go` (`creditOrRate` gone); D8 (c)'s stays. `go test -count=1 ./acpserver/` is
`ok`, `TestPromptQuotaData`, `TestSession` and `TestCancelClearsQueue` among it; `go test ./...` in `acpext` is `ok`.

* **Reds at the base.** The tests of existing behaviour, copied onto a scratch clone of `bf816c1` with the regions marked
  `// 0071 new names` left out (`<scratch>/b0071/base_copy.py`), fail there on their assertions, never on a build:
  `TestTurnSendsCap_0071` (each case `outputlimit_test.go:223: the turn asked for 0, want 64000`, and the `_meta` line),
  `TestTurnCapHeldToWindow_0071` (`the turn asked for 0 … want the room 44379`), `TestTurnCapSameAtEveryLevel_0071`,
  `TestOneEstimate_0071` (`1 summaries: the trigger counted the system prompt twice`; `0 summaries, want 1: the trigger did not
  count the tools`), `TestLimitRefusalLine_0071` (three of four; `other_invalid` a guard), `TestCapAfterModelSwitch_0071`,
  `TestCapConcurrentSessions_0071`, `TestContextShowsOutputLimit_0071`, `TestPromptResponseOutputLimit_0071`,
  `TestNoCallNoOutputLimit_0071`, `TestCapWhenPrepareRefuses_0071`, `TestCapSmallReserve_0071`, `TestTitleCapUnchanged_0071`
  (its turn's line; the title's 128 a guard) and `TestSummaryCaps` (`caps [8000 5000], want 64000 then 64000`). Guards green at
  the base: `TestCreditRefusalLine_0071` (pass 2's row), `TestAutoCompactPreviousModel`, `TestModelInfoModelsJSON`. Every other
  test names a name this phase adds; its red is its plant's.
* **Plants.** 47, each one production change on a scratch copy (`<scratch>/b0071/plant.py`, `plants_0071.py`), each caught by
  its own row's test; the failing lines are in `<scratch>/b0071/logs/plants*.log`. The harness was seen to report a plant not
  caught (P08b and P35 at first: see below). Rows L1 to L41's plants as the table names them, and L42's two, except:
  * **L36's plant** as written ("the ceiling raised to the window") cannot turn its test red: the case's 4,096 is below both
    the ceiling and the 1,000,000 window. Substituted: a listed cap below the ceiling raised to it (P32), caught,
    `outputlimit_test.go:223: the turn asked for 64000, want 32000`.
  * **L6's `AssumedMaxTokens` 0** was not caught at first by `TestModelsJSONDefaultIsAssumed_0071`, which compared with the
    constant itself; its expected value is now the literal 16,384, and the plant is caught (`cap 0 from "assumed", want 16384`).
  * **L39's Anthropic cap form**, first planted as "output token", still matched by prefix; replanted as "allowd", caught.
  * **L31, L32:** a dated note's link mistyped on a scratch copy fails `scripts/check_records.py --check-all`: "broken
    relative link: …0065-MADR-…md:22: 0071-MADR-send-each-turn-the-model-output-…".
* **Lint, same pass.** `make pre-add-check` found `gocognit` 51 on `(*turn).next` and an `errorlint` comparison in a test; fixed
  without a suppression (`limitRefused` returns the overflow, so `next`'s case reads as before), and the agent plants re-run,
  caught.
* **L34, the six live calls** (`go test -tags live -run TestLive_GLMCapAccepted_0071 ./llm/provider/`, this Mac, keys from the
  environment by name):

  | Provider | Model | Sent | Status | Finish | Tokens in / out / reasoning |
  | :--- | :--- | ---: | ---: | :--- | :--- |
  | kilo | `z-ai/glm-5.3` | 64,000 (ceiling) | 200 | end_turn | 20 / 39 / 42 |
  | kilo | `z-ai/glm-5.3-flash` | 64,000 (ceiling) | 200 | end_turn | 20 / 96 / 104 |
  | opencode-go | `glm-5.3` | 64,000 (ceiling) | 200 | end_turn | 20 / 63 / 60 |
  | opencode-go | `glm-5.3-flash` | 64,000 (ceiling) | 200 | end_turn | 20 / 3 / 0 |
  | together | `zai-org/GLM-5.3` | 64,000 (ceiling) | 200 | end_turn | 20 / 40 / 36 |
  | together | `zai-org/GLM-5.3-Flash` | 64,000 (ceiling) | 200 | end_turn | 20 / 49 / 46 |

  No refusal, 402 or 429 at the cap. *(2026-11-24, after the review: the Status column is inferred from each call's
  success, not read; the test then logged "status 200" without a status to read, and now logs "ok". The calls were not
  re-run.)*
* **Gates.** `go test -count=1 ./...` through `run-heavy`, this Mac: exit 0. `make pre-add-check FILES=…` (the 36 changed `.go`
  files), `run-heavy`, this Mac: exit 0. `make preflight`, `run-heavy`, this Mac: "preflight passed", exit 0. `make preflight
  race` through `run-on-host --need linux --race`, target the Linux machine (linux/amd64, not WSL), "preflight passed", every
  module `ok` under `-race`, remote status 0. `make preflight` through `run-on-host --need windows`, target the Windows laptop
  (windows/amd64), "preflight passed", remote status 0. No log has a fallback line. `make check-records`, `make markdownlint`:
  exit 0. `denyscan.py --changed`: 0 hits in 51 files. Observed, not built: the Windows preflight's gofmt step counted 305
  files where this Mac's and the Linux machine's counted 752; a follow-up for the host tooling, outside this record's rows.
* **Docs and records** (steps 6 and 7): `docs/guides/models-json.md` (`maxTokens`, "The output limit"); `docs/architecture.md`
  (acpext's member, `llm`, `agent`, `llm/catalog`, `compaction`, `acpserver`, `acpclient`); `internal/config/keys.go`'s
  `reserveTokens` doc and its schema; `docs/README.md`'s 0071 rows; D12's dated notes in 0065 (MADR, PLAN), 0066 (MADR,
  PLAN), 0070 (MADR, PLAN) and 0004 (`:746`). L29 is 0070's: 0070 is built second.
* **Other notes.** `TestContextShowsOutputLimit_0071` is in `acpserver/outputlimit_test.go`, beside its harness, not
  `commands_test.go`. `acptest.Turn` gains `Meta`, a test fixture's scripted `_meta`, for L27 and L28.

Exit of Phase 1 (pass 2): every row of the phase tested except L15's Phase 1b row and L43, which wait on `acpext/v0.6.0`;
plants caught; gates recorded. An independent review is the coordinator's. Phase 1b waits on the owner's tag.

### Deviation (2026-11-24): the review's notes, pass 2 continuing

The independent review passed Phase 1 with notes, relayed by the coordinator; they are closed in pass 2 (no third pass).

* **The owner's ruling, quoted exactly: "Keep the stated value (Recommended)".** The case: after an L39 resend at a stated
  value M, a later call of the same turn, an offline wait's, an empty reply's or an overflow compaction's, asked the full cap
  again (`agent/agent.go:739`, `t.limit, t.resendAt = req.MaxTokens, 0`), and a second refusal ended at `case at > 0:`
  (`:538`) a turn that would have succeeded. The MADR's D8 (a) is amended and row L44 added: every later call of the turn
  asks for at most M, its cap computed again (L11) and then held to M. Red first: `TestStatedCapHeldForTurn_0071` fails on
  the tree before the fix; then the hold in `turn.call`; then its plant.
* **Docs.** `docs/architecture.md`'s `llm` row and `docs/guides/models-json.md` said the limit is never below 1,024; the code
  sends `min(C', 1,024)` with less room (`llm/limit.go:94`), so a cap below 1,024 is sent as it is, and a refusal's resend may
  be any stated value of at least 1 (`agent/agent.go:621`). Both reworded.
* **L34's status column.** `llm/provider/live_glm_cap_test.go` logged "status 200" without reading a status: the SDK
  `v1.32.0` path gobble streams through gives a successful stream no HTTP status: only a failure carries one, an
  `llm.APIError`'s `Status` (`llm/errors.go:53-66`) or a retried failure's `llm.ProviderRetry.Status` (`llm/event.go:17-26`). The test now logs "ok"; the table's 200s are inferred from each call's success, see the note
  under it. The calls are not re-run.
* **Follow-ups, recorded, not built** (outside the decided rows):
  * Kilo's window form, "Your request contains approximately T tokens": gobble reads T as the request's total, input plus
    the limit sent (`llm/limitrefusal.go:59`), as go-llmprovider-sdk does (its 0088-MADR ST92, decided under its OQ14). The
    only source is Kilo's documentation (its C97, `kilocode:packages/kilo-docs/pages/gateway/api-reference.md:392`); no
    recorded Kilo reply shows whether T includes the limit. A recorded reply would settle it.
  * `llm.ReadLimitRefusal` reads the seven forms from any error's text. It does not take the SDK's V1 (a text matching a cap
    form and a window form rejected), V2 (a cap form's N equal to the limit sent), the per-route form sets, or the
    invalid-request kind check (go-llmprovider-sdk `llmprovider/internal/wire/limit_refusal.go` at `33e6f09`). The MADR's
    D8 (a) decides "reads the forms of F17 from the error's message" and nothing of those checks: a follow-up for a later
    amendment, or for Phase 2, where the SDK's own reader is used.
  * D8 (a) resends at any stated room or cap of at least 1 below the limit sent, as L39 decides: no change; noted only.
* **Result, pass 2 continued.** Red before the fix, on its assertion: `outputlimit_test.go:182: caps [60002 16000 60002],
  want [60002 16000 16000]` (`overflow_resend` and `next_call`; `computed_below` a guard). The fix: `turn.stated`, set at an
  L39 resend, and `turn.call` holds a larger computed limit to it, asked of `Config.OutputLimit` as the refusal's. Green
  after. Plant P46 (the hold's condition never true) caught with the same line; P12, P14, P15 and P42 re-run and caught.
  `make pre-add-check` on the three changed `.go` files through `run-heavy`: exit 0; `go vet -tags live ./llm/provider/`:
  exit 0; `make check-records`, `make markdownlint`: exit 0; `denyscan.py --changed`: 0 hits; `go test -count=1 ./...`
  through `run-heavy`, this Mac: exit 0.
* **Coverage closed (2026-11-24, the delta check):** L44's test gains `offline_wait` and `empty_reply`, the two other retries
  the ruling's case names, each in a `synctest` bubble; green. P46 fails both, with `overflow_resend` and `next_call`:
  `outputlimit_test.go:194: caps [60002 16000 60002], want [60002 16000 16000]` in each of the four.

### Phase 1b, pass 1 (2026-11-24, from `ba8f623`): the owner's word

The owner's word the 1b split waited for, relayed by the coordinator, quoted exactly: **"sdk committed and pushed, acpext
tag pushed"**. The tag `acpext/v0.6.0` is on `ba8f623`, the commit that carries `ErrorData.OutputLimitNote` (L42); the
agent pushed no tag. The tree was clean at `ba8f623` when Phase 1b started. Built next, in the PLAN's order: (2) `go.mod`
to `acpext v0.6.0`, no `replace`; (3) `acpserver` sets `outputLimitNote` and `internal/cli` shows it on the line after the
error; (4) the tests of L15's 1b row and L43, red first, each with a plant; (5) docs; (6) the gates.

* **Step 1.** `go get github.com/maccavelli/gobble-cli/acpext@v0.6.0` and `go mod tidy`: exit 0, the module proxy resolved the
  tag (`go: upgraded … acpext v0.5.0 => v0.6.0`); `go.mod` and `go.sum` change in those lines alone; no `replace`.
* **Steps 2 and 3, built, unstaged.** `acpserver/outputlimit.go`: `creditNote` (D8 (d)'s line for a capped call's 402 or
  `ErrQuota`, 429 or `ErrRateLimited`, or a 503 whose `APIError.Provider` is `kilo`) and `creditOrRate`;
  `acpserver/prompt.go`: `promptErrorNoted`; `acpserver/agent.go`: the failed turn's `promptErrorNoted` with the note;
  `internal/cli/line.go`: the data's `outputLimitNote`, after the error and the limit line. The error text is untouched.
* **Step 4, reds at the base** (a scratch clone of `ba8f623` with `go.mod` and `go.sum` at `acpext v0.6.0` and the two test
  files, the regions marked `// 0071 new names` left out; `<scratch>/b0071b/redbase.py`): `TestCreditRefusalLine_0071`
  `quota_402`, `rate_429`, `kilo_503`: `outputlimit_test.go:402: outputLimitNote "", want "gobble asked for at most 32,000
  output tokens (stated); some services hold credit or rate against that number; …"`, and `:405: the member is present:
  false, want true`; `other_503` a guard. `TestLineSessionOutputLimitNote_0071/with_note`: `limit_test.go:128: stderr "…
  Credits used up on kilo\n", want it to hold "… Credits used up on kilo\ngobble asked for at most 64,000 output tokens
  (ceiling); …"`; `without_note` a guard. `uncapped_call` names `creditNote`: its red is P2's.
* **Step 4, plants** (`<scratch>/b0071b/plants.py`, one production edit each on a scratch clone, `go test -count=1` on the
  row's package): P1 the member never set (`promptError` for `promptErrorNoted`), `outputlimit_test.go:413: outputLimitNote
  "", want "gobble asked for at most 32,000 …"`; P2 the note on an uncapped call, `outputlimit_test.go:427: a turn whose calls
  carried no limit: note "gobble asked for at most 0 output tokens (); …", want none`; P3 the note on any 503 (the Kilo
  test dropped), `outputlimit_test.go:413: outputLimitNote "gobble asked for at most 32,000 output tokens (stated); …", want
  ""` (`other_503`); P4 the wording changed, `outputlimit_test.go:413: … to ask for fewer", want "…"`; P5 (L43) the CLI omits
  the note, `limit_test.go:128: stderr "… Credits used up on kilo\n", want it to hold "… \ngobble asked for at most 64,000
  …"`. Each is caught by its own test; P2 to P4 also turn `TestPromptQuotaData` red, which is red already, below.
* **Stop (2026-11-24): an existing test needing a change no row names.** With the member set, `go test -count=1 ./acpserver/`
  turns `TestPromptQuotaData` red on three of its four frames (`errordata_test.go:220`): the test runs a prompt on a model the
  catalog does not know, so the failing call is capped (16,384, assumed) and the data now carries `outputLimitNote`, while the
  test compares the data byte for byte with `acpext/testdata/prompt-quota.error.json`, `prompt-rate-over-cap.error.json` and
  `prompt-weekly.error.json`, which have none. L15's 1b row says the test stays unchanged; it cannot, while L15 and L43 decide
  that a capped call's quota or rate failure carries the line. The frames are not at fault (the member is optional and absent
  from them). Resolutions went to the owner. The docs steps were done; the gates were not run.

### Deviation (2026-11-24): the owner's rulings on the stop, Phase 1b pass 1 continuing

Relayed by the coordinator, quoted exactly: **"Expect frame plus note (Recommended)"** and **"Test the JSON note, text as
is (Recommended)"**. The MADR carries both (L15 annotated, the "unchanged" text kept; new row L45; Counts 49 decided, 55 rows).

* **Scope added to Phase 1b:** `acpserver/errordata_test.go`, `TestPromptQuotaData` alone; `internal/cli/limit_test.go`,
  `TestPrintErrorDataNote_0071` (L45). The acpext frames and `TestCreditRefusalLine_0071` are untouched.
* **`TestPromptQuotaData`.** Its expected data is each frame decoded strictly, `OutputLimitNote` set (the note of a capped,
  assumed 16,384 call on `m1`) on the quota, rate-over-cap and weekly frames and none on `prompt-after-retries` (claude's 503),
  re-encoded, compared byte for byte with the prompt error's data. It also fails if a frame itself carries a note. Red before
  the edit, on its assertion (`errordata_test.go:220: prompt error -32603 {…"outputLimitNote":"gobble asked for at most 16,384
  output tokens (assumed); …"…}, want -32603 {…}`, three frames), green after.
* **L45.** `TestPrintErrorDataNote_0071` (`json`: the result's `errorData.outputLimitNote` equals the frame's; `text`: no
  "gobble asked for" in stdout or stderr). It tests code that is already built (`finishPrint` writes `errorData` from the
  prompt error), so it is green at the base of Phase 1b by construction; its red is its plants'.
* **Plants** (`<scratch>/b0071b/plants2.py`, one production edit each on a scratch clone, the package run whole):
  P1 member never set: `TestCreditRefusalLine_0071` and `TestPromptQuotaData` (`errordata_test.go:234`); P2 note on an uncapped
  call: `TestCreditRefusalLine_0071` alone (`outputlimit_test.go:427`); P3 note on any 503: both (`errordata_test.go:234` on
  `prompt-after-retries`); P4 wording: both; P5 CLI omits the note: `TestLineSessionOutputLimitNote_0071`
  (`limit_test.go:128`); **P6 the note dropped from a capped quota frame that states a reset time (the quota, rate and weekly
  frames): `TestPromptQuotaData` alone** (`errordata_test.go:234`); P7 quota failures lose the note: both; P8 print mode writes
  no `errorData`: `TestPrintErrorDataNote_0071` and the existing `TestPrintErrorData` (`limit_test.go:155: errorData null`);
  P9 print mode's text output prints the note: `TestPrintErrorDataNote_0071/text` alone (`limit_test.go:200: text output
  carries the note: stdout "", stderr "gobble asked for at most 64,000 …`).
* **Lint, same pass.** `make pre-add-check` first found `unused` on `frameData` (`errordata_test.go`), whose one caller the
  rewritten `TestPromptQuotaData` no longer has; the helper is removed (no suppression, no other caller), and the check re-run.
* **Gates, Phase 1b.** `make pre-add-check FILES=…` (the seven changed `.go` files), `run-heavy`, this Mac: exit 0 (after the
  `frameData` fix). `go test -count=1 ./...`, `run-heavy`, this Mac: exit 0. `make preflight`, `run-heavy`, this Mac:
  "preflight passed", exit 0. `make preflight race`, `run-on-host --need linux --race`, target the Linux machine
  (linux/amd64, not WSL; the distro excluded for a race run): "preflight passed", `go test -race` on the root module and
  acpext, remote status 0. `make preflight`, `run-on-host --need windows`, target the Windows laptop (windows/amd64):
  "preflight passed", remote status 0. No log has a fallback line. `make check-records`, `make markdownlint`, `denyscan.py
  --changed`: ~~see the hand-back~~ *(2026-11-24, closed after the review)*: `make check-records` exit 0, no finding;
  `make markdownlint` exit 0, no finding; `denyscan.py --changed` exit 0, "0 hit(s) in 14 file(s)".
* **Not built, noted:** print mode's text output prints no limit line and so no note, by L45's ruling.
* **Notes (2026-11-24, after the review).** (1) L15's "Kilo's 503 for a 402" is built as *any* 503 from Kilo on a capped
  call: the SDK's error cannot tell an upstream's 402 from another 503 (the MADR's F14 and 0065 R34: through Kilo an
  upstream's 402 arrives as a 503), so `creditOrRate` tests the status and `APIError.Provider` alone. `other_503` covers a
  non-Kilo 503. (2) L45's test, `TestPrintErrorDataNote_0071`, lives in `internal/cli/limit_test.go`, beside
  `TestPrintErrorData`, as the table row's annotation says.

Exit of Phase 1b: every row of the phase tested (L15's 1b row, L43, L45), plants caught, gates recorded. The independent
review is the coordinator's.

### Phase 2 execution entry (2026-11-25)

The owner approved the build, relayed by the coordinator, quoted exactly: **"Approve; build 0071 Phase 2 now (Recommended)"**.
This pass builds Phases 2.0 and 2a as "Phase 2 as re-planned" and its second deviation record them, from the MADR's rows of its
two 2026-11-25 amendments assigned to 2a (P1 to P12, P14 to P16, P29, P30, P35) and the 0065 rows placed there (R20 as P12,
R24 as P9, D7's `turn.stop.max_tokens_tool` with P29). Phases 2b and 2c are not started. The tree was clean at `2cbec91`,
which is 073fd1f with the records of this date only (no `.go`, `go.mod` or `go.sum` change), so it stands as the base the
re-planned Phase 2 names. The agent stages nothing and commits nothing.

### Phase 2.0, pass 1 (2026-11-25, at `2cbec91` and go-llmprovider-sdk `v1.33.0`)

* **Step 1, re-ground.** The SDK read at its tag with `git show v1.33.0:<path>` (the checkout carries another stream's
  uncommitted work; its working copy was not read). F13's lines hold as the amendment cites them: `limit_rule.go:58-83`
  (`ChooseLimit`: a request's value sent as given, basis `request`; the refusal before sending at `:78-81`), `:207-240`
  (`Check`: a kept cap at `:215-225`, a caller's value not resent at `:226-227`), `:258-290` (`Finish`, `markObserved`);
  `api_error.go:189-199` (`Observed`); `catalog/limits.go:33-44`, `:105-118`, `:202-216`, `:335-345`, `:450-467`,
  `:497-507`; `item.go:145-147`; `sent_limit.go:9-22`; `limit_observation.go:31-49`; `catalog/model_metadata.go:242-260`;
  `internal/limitmemo/limitmemo.go:133-136`; `providers/kilo/kilo.go:531-534`. F19's removed names and F22's cut answers are
  as the CHANGELOG's `[v1.33.0]` states them. gobble at `2cbec91`: every site F19 to F26 and P2 to P35 cite reads as cited
  (`go.mod:16`; `llm/provider/provider.go:53-55`, `:215`, `:241-243`, `:411`, `:434-438`, `:549-559`, `:647-648`, `:873`;
  `model.go:82-90`, `:107-111`, `:175-186`; `signin.go:344`; `image_test.go:209`; `list.go:129-131`, `:159-161`;
  `list_test.go:103`, `:117-122`, `:143-164`, `:204`, `:227`, `:277`; `sdkcache_test.go:45`; `chatserver_test.go:122`;
  `internal/cli/modelscmd_test.go:193`; `internal/cli/models.go:323`; `agent/agent.go:778-791`, `:906-933`;
  `acpserver/agent.go:786-811`, `:914`; `acpserver/outputlimit.go:29-33`, `:50-61`, `:123`; `acpserver/title.go:187-191`;
  `compaction/compaction.go:412`, `:415-422`, `:531`, `:556-558`; `llm/event.go:137-140`; `llm/errors.go:53-66`;
  `llm/provider/record.go:128`; `record_test.go:381`, `:484`, `:526`; `replay_test.go:184`, `:206`; `workspace_test.go:18`;
  `wire_test.go:131-156`, `:137`, `:158-160`; `agent/warning.go:40-56`; `acpclient/conn.go:353`). No line moved; no SDK
  fact changed. F1 to F18 and D1 to D13 are the record's facts at 97fa1f6, re-grounded by Phase 0 and built by Phase 1;
  2a cites them only through the amendment's lines above.
* **Step 2, the red list.** A scratch clone of `2cbec91` (`<scratch>/b0071p2/base`): `go get …@v1.33.0` and `go mod tidy`
  (exit 0, `go.mod` and `go.sum` on the SDK's lines alone), P2's six deletions, P3's seam as a stub (`<scratch>/b0071p2/stub20.py`;
  `Options.ModelMetadataURL` kept as an ignored field so the tests that set it compile). Without the stub, `go vet ./...`
  fails at the six sites, `undefined: llmprovider.WithModelMetadataURL` (`list.go:130`, `:160`) and `undefined:
  llmprovider.WithoutModelMetadata` (`model.go:176`, `provider.go:215`, `signin.go:344`, `image_test.go:209`). With it, `go vet
  ./...` exit 0 on both modules; `go test -count=1 ./...` through `run-heavy`, this Mac: 28 tests red, in `llm/provider` and
  `internal/cli` only (`<scratch>/b0071p2/logs/2.0-redlist.txt`):
  * P4: `TestListingSearchByFacts` (`list_test.go:287: vision [], want the model whose input holds image`),
    `TestModelsSearchListsFactsFirst` (`modelscmd_test.go:216: models list vision: exit 0, ids ["aaa-vision"]`).
  * P5: `TestListFullCatalog` (`list_test.go:118: requests ["GET api.openai.com/v1/models"], want the listing with the key
    and the metadata document`), `TestListFetchesMetadata` (`list_test.go:162`).
  * P8: 24 tests, each refused before sending on a Claude or OpenCode messages route (`llmprovider: invalid request: claude:
    no source states an output limit for model "claude-haiku-5-5", and the wire requires one: …`), or counting 0 requests
    for that refusal (`cache_test.go:112`, `:157`): `TestWireDeferral`, `TestDeferralAfterCancelledTurn`, `TestWireLimits`,
    `TestWireEmptyAnswer`, `TestWireRefusal`, `TestWireReplay`, `TestWireCases`, `TestWireRateLimitResets`,
    `TestWireToolResultError`, `TestWireStrict`, `TestModelCallSignatureResend`, `TestWireSignatureRetry`, `TestWireWorkspace`,
    `TestWireStatedWindow`, `TestWireBudgetLimit`, `TestCacheWire`, `TestSessionIDPerProvider`, `TestImageWires`,
    `TestTextOnlyModelImage`, `TestStreamIsLive`, `TestFailureAfterFirstDeltaNotRetried`, `TestProviderRetryReported`,
    `TestWireBaseURL`, `TestWireRejectedKey`.
* **Deviation (2026-11-25), the red list's length.** The second deviation's list (the `wireStream` callers and five direct
  sites) is short of the run by eleven tests, and one of its sites is not red (`record_test.go:484`, whose `callReq()` names
  4,096). Every extra red is the refusal before sending that P8's row decides ("a test … that Phase 2.0's red run finds
  refused by `limit_rule.go:78-81`"); none is a red no row names, so this is no stop. The MADR's dated note of this date
  records the measured list. No decision changes.
* **Step 3, docs.** `README.md:178` and `docs/architecture.md:202` name `v1.32.0`; `llm/provider/provider.go:53-55`
  (`ModelMetadataURL`), `:211` ("the model metadata fetch are off") and `llm/provider/list.go:98` ("model metadata on") name
  the fetch. The `v1.32.0` citations at `list.go:57`, `signin.go:173`, `signinrun.go:219`, `:234` cite that tag's code as
  history and stay. `docs/guides/models-json.md` and `docs/guides/choosing-a-model.md` name neither.

Exit: met; nothing but scratch changed (and this record).

### Phase 2a, pass 1 (2026-11-25, from `2cbec91`)

Built in a scratch clone (`<scratch>/b0071p2/work`, each step a recorded script beside it), then copied into the tree over
files checked equal to `2cbec91`'s (`to_tree.py`); `go get github.com/maccavelli/go-llmprovider-sdk@v1.33.0` and `go mod
tidy` were then run in the tree: exit 0, `go.mod:16` at `v1.33.0`, `go.sum` on the SDK's two lines alone, no `replace`,
acpext's pin unchanged.

* **P1, P2.** `go.mod`; the six sites: `New`'s and the wizard's `WithoutModelMetadata` deleted, the lookups' `opts` a clone
  of the SDK options, `image_test.go`'s option deleted, `list.go`'s two `WithModelMetadataURL` sites hand
  `Options.ModelData`; `Options.ModelMetadataURL` removed.
* **P3.** `llm/provider/modeldata.go`: `ModelData` (the parsed `catalog.Metadata`), `ParseModelData` (over
  `catalog.ParseMetadata`), `Options.ModelData`; only `List` and `listChat` read it.
* **P4, P5.** The URL sites hand the bytes they served (`modelData(t, …)`; `factMetaDoc` in `internal/cli`);
  `chatserver_test.go:122`'s field deleted; `TestListFetchesMetadata` deleted; `TestListFullCatalog` asserts one request,
  the listing's, with the key.
* **P6.** `internal/cli/models.go`: `builtInModelData`, the built-in copy parsed once, set on the listing's `Options` in
  `modelListing` alone.
* **P8.** `wire_test.go`'s `assumedLimit` (16,384 when a request names none), called by `wireStream` beside `req.Model =
  model` and at each other refused test's request; no other assertion changed.
* **P9 to P12, P29 (adapter).** No production change: at the tag the adapter already yields a cut answer's whole calls and
  its `tool_call_cut` warning, and `stopOf` gives `tool_use` or `max_tokens`; Kilo's window form maps to
  `llm.ErrContextOverflow`.
* **P14 to P16.** `llm/sentlimit.go` (`SentLimit`, `LimitObservation`); `Done.OutputLimit`; `APIError.Observed`, set in
  `mapErrorAt`; `WarningToolCallCut`, `WarningLimitLearned`; the record's four `out_limit*` fields on a call whose done event
  arrived. 0070's `TestModelCallRecordHoldsNoContent/no_out_limit` is replaced by `TestModelCallRecordOutLimit_0071`, and its
  key list takes the four fields.
* **P29.** `personality.TurnStopMaxTokensTool` (`turn.stop.max_tokens_tool`): boring's line as the MADR writes it, and one
  line each in dry, snarky and bombastic, in their voices, each with `{model}` and the same advice; `TestBoringIsTodaysText`
  pins boring's text, as each key since 0022 is pinned. `acpserver/cutcall.go`: the turn's provider is watched (a Prober
  stays one) for a call that ends `max_tokens` after a `tool_call_cut` warning; the turn then sends the line as an agent
  message after its text, live only, so the terminal and headless print and JSON carry it as agent text (0065 D8's
  channel).
* **P30.** `acpserver/title.go`: a title call that ends `max_tokens` writes its usage and returns an error, logged, and
  names nothing.
* **P35.** `llm.BasisNone`; `turnMeta` and `acpclient` keep a limit whose basis is set, so `{0, "none"}` is carried and a
  turn with no call still carries none (L30).
* **Exported names**, all in P23: `provider.ModelData`, `provider.ParseModelData`, `provider.Options.ModelData`,
  `llm.SentLimit`, `llm.Done.OutputLimit`, `llm.LimitObservation`, `llm.APIError.Observed`, `llm.WarningToolCallCut`,
  `llm.WarningLimitLearned`, `llm.BasisNone`, `personality.TurnStopMaxTokensTool`; removed: `provider.Options.ModelMetadataURL`
  (P2).

Rows to tests (files): P1, P2 the build; P3 `TestListHandsModelData_0071` (`handed`, `none`, `chat`; `list_test.go`); P4 the
URL sites' tests; P5 `TestListFullCatalog`; P6 `TestListingRanksByBuiltInCopy_0071` (`internal/cli/limit_test.go`); P7
`TestEveryCallNamesLimit_0071` (`turn`, `summary`, `title`; `acpserver/outputlimit_test.go`); P8 the 24 tests; P9
`TestAgentTurn_SalvageCompleteCalls_0065` (`llm/provider/cut_test.go`, `agent/outputlimit_test.go`); P10
`TestResponsesCutTextStops_0071`; P11 `TestReasoningOnlyCutUnchanged_0071`; P12 `TestAgentTurn_KiloWindowFormCompacts_0065`
(the adapter's kind in `cut_test.go`; `room` and `over` in `agent`); P14 `TestDoneCarriesOutputLimit_0071` (`asked`,
`raised_for_thinking`; `llm/provider/sentlimit_test.go`); P15 `TestAPIErrorCarriesObserved_0071` (`anthropic_cap`,
`openrouter_window`); P16 `TestModelCallRecordOutLimit_0071` (`completed`, `failed`); P29 `TestCutCallAfterText_0071`
(`chat`, `messages` in `cut_test.go`; `agent`; `cut_call`, `cut_text` in `acpserver`; `cli`, `headless_json` in
`internal/cli/limit_test.go`) and `TestOnlyCutCall_0071`; P30 `TestTitleCutDiscarded_0071`; P35
`TestPromptResponseOutputLimit_0071/none` (`acpserver`, `acpclient`). A test name shared by two packages is one row's test
in each layer, as the PLAN's table names two files for it.

* **Reds at the base.** The tests of behaviour that exists at `2cbec91`, copied onto a clean clone of it (go-llmprovider-sdk
  `v1.32.0`) with the regions that name 2a's names left out and `llm.WarningToolCallCut` written as its string
  (`<scratch>/b0071p2/base_copy.py`), run there; each red is on its assertion (line numbers are the base copy's):
  * P9: `cut_test.go:160: calls [], stop "", err … llmprovider: incomplete response: length; want the whole call alone, then
    tool_use`.
  * P10: `cut_test.go:169: text "half an answ", stop "", err … incomplete response: max_output_tokens; want the text, then
    max_tokens`.
  * P12 (the adapter): `cut_test.go:190: err llm: api status 400: llmprovider: invalid request: kilo HTTP 400 … (other), want
    llm.ErrContextOverflow`.
  * P29: `cut_test.go:207` (`chat`, `messages`: `… incomplete response: length; want the text, no call, max_tokens`);
    `acpserver` `outputlimit_test.go:728: agent messages ["Let me read it."], want ["Let me read it." "\n\nThe reply hit the
    output limit while writing a tool call, …"]`; `internal/cli` `limit_test.go:241` (`cli`: the stdout lacks the line) and
    `:252` (`headless_json`: `Text:Let me read it. StopReason:max_tokens …, want max_tokens and the text, then the line`).
  * P30: `title_test.go:386: the session took the cut title; logs ""`, `:389: logs "", want the cut named`.
  * P35: `acpserver` `outputlimit_test.go:583: _meta {} (<nil>), want {"gobble":{"outputLimit":{"tokens":0,"basis":"none"}}}`;
    `acpclient` `outputlimit_test.go:51: OutputLimit <nil>, want {Tokens:0 Basis:none}`.
  * Guards green at the base: P11 `TestReasoningOnlyCutUnchanged_0071`, `TestOnlyCutCall_0071`; P7
    `TestEveryCallNamesLimit_0071` (all three); P12 `agent` `room` and `over`; the `agent` parts of P9 and P29; `acpserver`
    `TestCutCallAfterText_0071/cut_text`.
  * First run, two reds were a wait's timeout (`title_test.go:380`, `limit_test.go:228: condition not met within 5 s`), not
    an assertion: each test now waits for its work to end either way, then asserts; the reds above are the rerun's.
  * New names, red from their plants: P1 to P6, P14 to P16, and P29's key in `TestBoringIsTodaysText`.
* **Plants** (`<scratch>/b0071p2/plants.py`: one production edit at a time on a scratch copy of the built tree, undone
  before the next; the row's package(s) run with every 2a test and every P8 test selected; logs in
  `<scratch>/b0071p2/logs/plants/`). The harness's control, a comment changed, was reported not caught, as it must be. Every
  plant caught:
  * P1a, the pin reverted (`go.mod` and `go.sum` at `v1.32.0`): the build, `llm/provider/modeldata.go:37:38: undefined:
    catalog.WithMetadata`. P1b, `New`'s option restored: `provider.go:218:15: undefined: llmprovider.WithoutModelMetadata`.
  * P3a (P3, P4), `listWith` drops `o.ModelData`: `TestListHandsModelData_0071` (`list_test.go:149: vision [], want
    ["gpt-6-luna"]`), `TestListingSearchByFacts` (`:319`), `TestModelsSearchListsFactsFirst` (`modelscmd_test.go:225`), and
    `TestListingRanksByBuiltInCopy_0071`, which lists through `listWith`. P3b, `listChat` drops it:
    `TestListHandsModelData_0071` alone (`list_test.go:168: vision [], want the document's image model`).
  * P5, `listWith` lists with no credential: `TestListFullCatalog` (`list_test.go:115: requests ["GET
    api.openai.com/v1/models"], want one: the listing with the key`); also `TestListServesLastGood` and
    `TestListingSearchByFacts`, whose listings then fail.
  * P6a, the copy left off the listing's `Options`: `limit_test.go:297: the listing's Options carry no document`. P6b, the
    copy in `providerOptions` (what `New` and the wizard are given): `limit_test.go:320: the options New and the sign-in
    wizard are given carry a document`.
  * P7a, `OutputLimit` left nil: `outputlimit_test.go:667: the turn asked for 0, want a limit` (and L27's
    `TestPromptResponseOutputLimit_0071`). P7b, the ladder's `FitOutput` times 0: `:682: summary 1 asked for 0, want a
    limit`. P7c, the title's 0 when its cap is unknown: `:697: the title asked for 0 with its model's cap unknown, want a
    limit`.
  * P8, `toRequest` drops `MaxOutputTokens`: each of the 24 refused again (e.g. `wire_test.go:192: … no source states an
    output limit for model "claude-haiku-5-5" …`; `cache_test.go:112: 0 requests, want 1`), and the 2a tests that stream a
    Claude route (P14, P15, P16, P29's `messages`).
  * P9a, `stopOf` tests `FinishLength` before calls: `cut_test.go:160: calls ["read {\"path\":\"x\"}"], stop "max_tokens",
    err <nil>; want the whole call alone, then tool_use`. P9b, the turn ends at a `tool_call_cut` warning: `agent`
    `outputlimit_test.go:235: 1 calls, stop "max_tokens"; want the whole call run and a second call`.
  * P10, `stopOf`'s `FinishLength` case removed: `cut_test.go:174: … stop "end_turn" …; want the text, then max_tokens` (and
    P29's `cut_test.go:212`).
  * P11, the adapter maps the error to `Done{StopMaxTokens}`: `cut_test.go:183: err <nil>, dones [{max_tokens {0 false  }}];
    want llm.ErrIncomplete, not an empty reply, and no Done` (and `TestOnlyCutCall_0071`, `TestWireEmptyAnswer`).
  * P12, `limitRefused` reads the overflow first: `agent` `outputlimit_test.go:274: caps [60002 60002], want [60002 31072]`.
  * P14a, `Done.OutputLimit` left zero: `sentlimit_test.go:77: Done.OutputLimit {Tokens:0 Sent:false Basis: Source:}, want
    {Tokens:32000 Sent:true Basis:request Source:}`. P14b, the request's value for the wire's: `:99: Done.OutputLimit
    {Tokens:2000 …} with 8192 on the wire; want the raised wire value`.
  * P15a, `Observed` not copied: `sentlimit_test.go:132: Observed {Model: … Tokens:0 Seen:0001-01-01 …}`. P15b, a zero
    `Seen`: `:132: Observed {Model:claude-haiku-5-5 … Tokens:32000 Seen:0001-01-01 00:00:00 +0000 UTC}`.
  * P16a, the fields on a failed call: `record_test.go:549: out_limit present on a call that failed before its done event`.
    P16b, `out_limit_basis` omitted: `record_test.go:541: fields map[…], want out_limit 4096, sent, basis request, no
    source`.
  * P29a, the line omitted: `acpserver` `outputlimit_test.go:745: agent messages ["Let me read it."], want […]`, and
    `internal/cli` `cli` and `headless_json` (`limit_test.go:250`, `:261`). P29b, `turn.stop.max_tokens`'s line shown:
    `outputlimit_test.go:745: agent messages ["Let me read it." "\n\nmax_tokens"], want […]`, and both `internal/cli` cases.
    P29c, the text dropped (the agent's reply text reset at the warning): `agent` `outputlimit_test.go:304: the turn's
    messages ["user:go"] lack the reply's text`.
  * P30, the stop reason ignored: `title_test.go:386: the session took the cut title; logs ""`.
  * P35a, `turnMeta`'s `Tokens > 0` kept: `outputlimit_test.go:595: _meta {} (<nil>), want …`. P35b, `acpclient`'s kept:
    `outputlimit_test.go:51: OutputLimit <nil>, want {Tokens:0 Basis:none}`.
  * Plants caught beyond their own row's test are named above; each is caught by its own.
* **Lint, same pass.** `make pre-add-check` first found `acpserver/title_test.go:381` not gofmt'd (a long `waitFor` line,
  wrapped by gofmt); `gofmt -w`, no suppression, re-run: exit 0.
* **Docs.** `README.md:178`, `docs/architecture.md:202` at `v1.33.0`; `docs/architecture.md`'s `agent` (`:111`), `llm`
  (`:112`), `llm/provider` (`:113`), `acpserver` (`:130`), `acpclient` (`:131`) and `internal/cli` (`:135`) rows; the doc
  comments of `Options.ModelData`, `New` and `List`.
* **Gates** (exit codes taken before any filter; logs in `<scratch>/b0071p2/logs/gate-*.log`; no log has a fallback line):

  | Command | Where | Exit |
  | :--- | :--- | ---: |
  | `make pre-add-check FILES=…` (the 35 changed `.go` files), `run-heavy` | this Mac | 2, then 0 (the gofmt finding above) |
  | `go test -count=1 ./...`, `run-heavy` | this Mac | 0 |
  | `make preflight`, `run-heavy` | this Mac | 0, "preflight passed" |
  | `make preflight race`, `run-on-host --need linux --race` | the Linux machine (linux/amd64, not WSL) | 0, "preflight passed", both modules `ok` under `-race` |
  | `make preflight`, `run-on-host --need windows` | the Windows laptop (windows/amd64) | 0, "preflight passed" |
  | `make check-test-isolation` | this Mac | 0 |
  | `make check-records`, `make markdownlint` | this Mac | 0, 0 issues |
  | `denyscan.py --root <gobble> --changed` | this Mac | 0, "0 hit(s) in 45 file(s)" |
  | `github-disclosure.py redact` (dry run) | this Mac | 0, "would edit 0 file(s)" |
  | `go vet -tags live ./llm/provider/ ./internal/cli/ ./acpserver/` | this Mac | 0 |

  `make check-test-isolation` runs both modules' tests, a heavy run, and it ran outside `run-heavy`, beside the queued
  pre-add check: a breach of the queue rule, noted; its result stands (exit 0).
* **Not done in 2a:** no live call (2a names none); Phases 2b and 2c are not started. An independent review is the
  coordinator's.

Exit of Phase 2a (pass 1): every row of 2a tested, red at the base or held by its plant; plants caught; gates recorded;
docs rows returned.

### Deviation (2026-11-25): the review's notes, Phase 2a pass 1 continuing

The independent review passed Phase 2a with notes, relayed by the coordinator; they are closed in pass 1 (no second pass).
The owner's answers, quoted exactly: **"Approve all four (Recommended)"** (P29's four lines are the decided text) and
**"Show the notice anyway (Recommended)"** (new row P36). The MADR carries both ("Amendment of 2026-11-25, third"), with the
measured note on Kilo's and Ollama's listings.

| Part | Rows | Test | File | Red at the base because | Plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 2a | P29 (its lines) | `TestToolCutLines_0071` (each built-in personality's `turn.stop.max_tokens_tool` is its one decided variant) | `personality/builtins_test.go` | new key | one personality's line changed (snarky's) |
| 2a | P36 | `TestCutCallNoticeBeforeQueued_0071` (text, then a cut call, a follow-up queued: the notice once, after the text and before the queued message's echo; the turn goes on to the next call and ends `end_turn`) | `acpserver/outputlimit_test.go` | no notice at the base | the notice sent only when the turn ends `max_tokens` |
| 2a | P7 (review) | `TestEveryCallNamesLimitColdClaude_0071` (built as a test of its own beside `TestEveryCallNamesLimit_0071`, which runs on scripts): a turn through `provider.New("claude", …)` on a canned wire, with no source stating the model's limit, is sent, not refused before sending | `acpserver/outputlimit_test.go` | a guard | the turn's limit removed (`OutputLimit` nil): the SDK refuses the call before sending |

The review's other notes: `acpserver/outputlimit_test.go`'s check of `llm.BasisNone` against its own value, which cannot
fail, compares `turnMeta`'s output with the constant instead; `internal/cli/limit_test.go`'s check that the lookup of
`o3-mini` has a window, which passes at the base, is dropped (P6's checks are the listing's document and `providerOptions`').
Files added to the phase's scope: `personality/builtins_test.go`.

**Result, pass 1 continued.**

* **Built.** `acpserver/cutcall.go`'s `take` reads and forgets the cut, so the notice is sent once; `acpserver/agent.go`
  sends it (`sayCut`) before the first `agent.Queued` echo after a cut call (P36), as well as at a `max_tokens` end (P29).
  The tests above, and the two review fixes.
* **Red at the base** (the base copy, `<scratch>/b0071p2/redbase`): P36 `outputlimit_test.go:760: updates ["agent Let me
  read it." "user and then this" "agent on it"], want ["agent Let me read it." "agent \n\nThe reply hit the output limit …"
  "user and then this" …]`. `TestEveryCallNamesLimitColdClaude_0071` is a guard (at `v1.32.0` the SDK sends 8,192), held by
  its plant; `TestToolCutLines_0071` names the new key, held by its plant.
* **Plants** (`plants.py`, every plant of 2a run again on a scratch copy of the tree, the control again not caught; logs in
  `<scratch>/b0071p2/logs/plants2/`): all caught, the earlier ones as before. New: P29d, snarky's line changed,
  `builtins_test.go:238: snarky: ["Ran out of tokens halfway through a tool call, so that call is history. …"], want "Ran out
  of tokens halfway through a tool call, so that call is gone. …"`; P36, the queued notice not sent, `outputlimit_test.go:851:
  updates ["agent Let me read it." "user and then this" "agent on it"], want […]`; P7d, the turn's limit removed,
  `outputlimit_test.go:816: the turn failed: {"code":-32603,…"llmprovider: invalid request: claude: no source states an
  output limit …`; P35c, `BasisNone`'s value changed, `outputlimit_test.go:605: llm.BasisNone's _meta
  {"gobble":{"outputLimit":{"tokens":0,"basis":"no"}}} …, want …"none"…`. P29b now also turns
  `TestCutCallNoticeBeforeQueued_0071` red.
* **Kilo and Ollama.** Measured, not built: the built-in copy changes neither listing's order (the MADR's third amendment,
  "Measured"; the probe was a scratch test over the SDK's Kilo fixture, removed with its copy).
* **Lint, same pass.** `make pre-add-check` found `acpserver/outputlimit_test.go:15` not goimports-sorted (the new imports
  placed after the module's); sorted by hand, re-run: exit 0.
* **Docs.** `docs/architecture.md:130`, the `acpserver` row: the notice before the first queued message (P36).
* **Gates, pass 1 continued** (this Mac, every heavy run through `run-heavy`; logs `gate-*2.log`, `gate-preadd3.log`,
  `gate-preadd4.log`):

  | Command | Where | Exit |
  | :--- | :--- | ---: |
  | `make pre-add-check FILES=…` (the 36 changed `.go` files), `run-heavy` | this Mac | 2 (goimports), then 0 |
  | `go test -count=1 ./...`, `run-heavy` | this Mac | 0 |
  | `make check-test-isolation`, `run-heavy` | this Mac | 0 |
  | `make check-records`, `make markdownlint` | this Mac | 0, 0 issues |
  | `denyscan.py --changed` | this Mac | 0, 0 hits |
  | `github-disclosure.py redact` (dry run) | this Mac | 0, 0 files |

  `make preflight` and `make race` were not run again: not asked for this continuation.
