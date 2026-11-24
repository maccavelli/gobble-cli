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
| L15 | `TestCreditRefusalLine_0071` (`quota_402`, `rate_429`, `kilo_503`, `uncapped_call`) | `acpserver/outputlimit_test.go` | no line | the line omitted; then, apart, on an uncapped call |
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
