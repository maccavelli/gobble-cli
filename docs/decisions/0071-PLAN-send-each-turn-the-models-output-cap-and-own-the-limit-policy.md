---
status: in-progress
date: 2026-11-24
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
| L15 (pass 2, before acpext's tag) | `TestCreditRefusalLine_0071` (`quota_402`, `rate_429`, `kilo_503`, `other_503`: the error text byte-identical, no line); `TestPromptQuotaData` unchanged | `acpserver/outputlimit_test.go`; `acpserver/errordata_test.go` | a guard | the line appended to the error text, as pass 1 built it |
| L15, L43 (Phase 1b, after `acpext/v0.6.0`) | `TestCreditRefusalLine_0071` gains `outputLimitNote` on each capped failure and none on `uncapped_call` and `other_503`; the terminal client's line after the error | `acpserver`; `internal/cli/limit_test.go` | new | the member not set; then, apart, on an uncapped call; the CLI's line omitted |
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
phases that build discovery, the observation store and the notices (V6).

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
