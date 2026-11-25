---
status: in-progress
date: 2026-11-24
associated-madr: "0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement Keeping a Cut Reply, Retrying an Accidental Cut, and the Per-Call Stream-Health Record

Associated MADR: [0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md](0070-MADR-keep-a-failed-partial-reply-and-log-stream-health-per-call.md)

*(2026-11-24, first draft, written with the MADR's fourth draft, when Q6 and Q7 were open.)*

*(2026-11-24, second draft: the MADR's fifth draft is `accepted` with no open row. Q1 was re-answered "Auto-retry, with
limits (Recommended)", superseding "Auto-continue, with limits (Recommended)"; Q7 "Retry notice live (Recommended)"; Q6
is moot. Phase 3 is now the auto-retry phase (design B: one case in `next()`, no SDK work); the continuation's
`Origin.Continued`, `gobble.continue` entry and `replay.go` change are gone from the scope.)*

*(2026-11-24, note from [0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md](0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md) D11: built after 0071's Phase 1, this PLAN's `TestModelCallRecord` expects `limit` to be the request's `MaxTokens`, not 0 (0071 L29).)*

## Goal

After this PLAN, a model call a failure ends is always written to the session in pi's shape, with pi's "Error: …" line
on reload; every attempt the turn replaces after it showed something is kept; a call an accidental failure cut is sent
again, the same request, at most 3 times in a row, under `retry.retryAfterCut`; and every model call writes one stream-health
record to the log, at `info` when it failed, was cancelled or was retried, else at `debug`. Every decided row of the
MADR has a test, red at the base where it changes behaviour, and a plant that turns it red.

Nothing starts before the owner approves this PLAN and says `proceed`. The agent stages nothing and commits nothing:
each phase ends with its files ready, its gates recorded, and the coordinator stages; the owner commits.

## Scope

### In scope

* `agent`: `agent.Message.Err`; the call a failed turn ends on; replaced attempts; the cancel during the empty-reply
  wait; the cut retry (`reply.started`, the case in `next()`, its count, the `cut` notice, the context value for the
  record).
* `llm`: the cut retry's context value; `llm.Retry`'s cut-retry switch.
* `llm/provider`: the adapter's record (`provider.go`), its `now` field and logger.
* `acpserver`: the writer (`"error"` entries), `stamped` (keeps `Failed`), `replayMessage` (the "Error: …" line), the
  append-failure path, the logger with the session.
* `internal/config`: `retry.retryAfterCut`, the schema golden; the setting reaching `llm.Retry`.
* Tests in each of those packages and in `internal/cli` (redaction, headless output).
* Docs and records (Phase 4).

### Out of scope

* Any SDK change. OTEL, `--stats`, the SDK's H4. A tool-call preview. Headless output when the turn ends on the error.
* A change to `acpext`: the `cut` cause is a value of an open set (`acpext/doc.go:112`).

## Rules for Every Phase

* **Re-ground first.** Re-read every `path:line` the MADR cites at the phase's start commit; record each moved line.
* **Red, then green.** A test of existing behaviour runs at the phase's base and fails on its assertion, never on a
  compile error. A test of a new name (`agent.Message.Err`, `reply.started`, the record's fields, the setting) takes
  its red from its plant. Guards pass at the base and are held by their plants.
* **Plants.** One production change each, made by hand on a scratch clone of the phase's tree, never in the tree; each
  must turn its own row's test red, and the record quotes the failing line. A plant its test does not catch is a
  deviation.
* **Seams.** Agent tests use `llmtest` scripts. Adapter tests use `provider.New` with `Options.HTTPClient` an in-memory
  `RoundTripper` (as `llm/provider/stream_test.go:34`) and a capturing `slog.Handler`; the provider, its `Set`
  (`NewSet`) and its `Cache` are built in the test, inside the `synctest` bubble where one is used. History tests are
  built as `failedTurnRequests` (`acpserver/resume_test.go:324`).
* **Hosts and heavy runs.** A single package's `go test` without `-race` runs on the session host.
  `make pre-add-check FILES="…"` runs through `~/.agents/skills/available-dev-hosts/scripts/run-heavy --`. Each phase's
  `make preflight` runs through `~/.agents/skills/available-dev-hosts/scripts/run-on-host` on each of the three hosts
  (this Mac, the Linux machine, the Windows laptop), one at a time per host, after `pick-host`; `make race` runs on the
  Linux machine with `run-on-host --race`, never on WSL. A run `run-on-host` reports as a fallback to this Mac does not
  count for another host. Long output goes to a scratch file and the exit code is captured before any filter.
* **No live call.** Every provider answer is a test double.
* **Stops**, each a stop and a prompt: a test outside the MADR's rows turns red; a row cannot be met as written; an
  exported API change no row names; a lint finding fixable only by a suppression; the same step failing twice; a
  third build pass of one phase.
* **Exit criterion of every phase:** no blocker, every row of the phase tested, plants caught, gates recorded, docs
  rows returned.

## Implementation Steps

### Phase 0: re-ground and discover (records and a scratch clone only)

1. Re-read the MADR's citations at HEAD (F1 to F21, every D and row); record each moved line.
2. On a scratch clone: add `agent.Message.Err` and emit it on the shown-guard path; run `go test ./agent ./acpserver
   ./internal/cli` there and list the existing tests that turn red (consumers that count events). The MADR expects
   none outside its rows; any other red is a stop.
3. Confirm on the clone: whether the architecture test lets an external test package of `llm/provider` import
   `agent` (T15; else the test sits in `internal/cli`); an input `toRequest` refuses (T22); that a `completed`
   Responses stream with no output maps to `llm.ErrEmptyReply` at `v1.32.0` (T15); that `stamped` replaces `Origin`
   whole (`acpserver/choice.go:250-251`).
4. Grep `docs/architecture.md`, `docs/guides/credentials.md`, `docs/guides/mcp-servers.md` and `README.md` for the
   failed turn, the retry settings and `gobble.log`; note the lines Phase 4 changes.

Exit: citations re-grounded; the red list recorded and equal to the MADR's expectation; nothing but the records changed.

### Phase 1: keep what a call produced (G rows, Q2 to Q4)

Red tests first (each compiles at the base where it uses only existing names; those that name `Err` take their red
from their plant):

| Rows | Test | File | Red at the base because |
| :--- | :--- | :--- | :--- |
| G1, G2, G3, G8, G9, G10, G16, G36, G37, G33 (agent half) | `TestFailedCallKept` and its subtests | `agent/keep_test.go` | no message precedes the error; nothing kept on the wait's cancel |
| G4 | `TestFailedCallKept/text_and_calls`, `TestLoadReplaysFailedEntry/calls` | `agent`, `acpserver/history_test.go` | nothing written |
| G7, G34, G35 | `TestReplacedAttemptsKept` | `agent/keep_test.go` | replaced attempts dropped |
| G7 (live) | `TestReplacedAttemptHistoryIsTheFiles` | `acpserver/resume_test.go` | nothing written, then (plant) `stamped` drops `Failed` |
| G33 (live) | `TestCancelInWaitHistoryIsTheFiles` | same | nothing written |
| G18 | `TestFailedEntryShape` | `acpserver/failedentry_test.go` | no entry |
| G19 | `TestShownFailureHistoryIsTheFiles` | `acpserver/resume_test.go` | a guard today (nothing is written); its plant writes `"stop"` |
| G20 | `TestFailedEntryAppendFails` | same, with `failAt` | the entry does not exist |
| G23, G24, G38, G40 | `TestLoadReplaysFailedEntry` | `acpserver/history_test.go` | no entry, no line |
| G26, G39 | `TestLastAssistantTextAfterFailure`, `TestLastAssistantTextAfterEmptyFailure` | `acpserver/extensions_test.go` | null (F3's probe) |
| G27, G28 | `TestSerializeKeepsFailedEntry`, `TestFailedEntryCounted` | `compaction`, `acpserver/commands_test.go` | guards; plants named in the MADR |
| G29 | `TestForkAfterFailedEntry` | `acpserver/extensions_test.go` | no entry to be the parent |
| G13, G15, G17 | `TestCancelWinsOverFailure`, `TestLengthCutIsNotFailed`, `TestRefusalIsNotFailed` | `agent` | guards |
| G5, G11, G12, G14, G21 | the existing tests the MADR names | | guards |

Build: `agent` (D2 to D4, D10, D11), `acpserver` (writer, `stamped`, `replayMessage`, the append-failure answer,
`extensions`). Each plant of the MADR's G rows on a scratch clone.

Gates: `go test ./agent ./acpserver ./compaction ./internal/cli` (session host); `make pre-add-check FILES="<the .go
files>"` through `run-heavy`; `make preflight` on the three hosts through `run-on-host`; `make race` on the Linux
machine.

### Phase 2: the stream-health record (T rows, Q5)

| Rows | Test | File | Red because |
| :--- | :--- | :--- | :--- |
| T1, T3, T13, T20, T22 | `TestModelCallRecord` and subtests | `llm/provider/record_test.go` | no record (its plant) |
| T2, T19 | `TestModelCallRecordOffAboveInfo`, `TestModelCallRecordNoAllocsAboveInfo` | same | new names; plants |
| T4 | `TestModelCallTiming` | same (`synctest`) | plant |
| T5 to T10 | `TestModelCallRetries`, `TestModelCallSignatureResend`, `TestModelCallDelivery`, `TestModelCallOutcome`, `TestModelCallCache` | same | plants |
| T11, T14 | `TestModelCallRecordHoldsNoContent` | same | plants |
| T12 | `TestModelCallRecordSurvivesRedaction` | `internal/cli/record_test.go` (a temp log file) | plant |
| T15 | `TestModelCallRecordPerRerun` | as Phase 0 step 3 decided | plant |
| T21 | `TestModelCallRecordBuffered` | `llm/provider` (in-package) | plant |
| T23 | `TestModelCallRecordLevel` | `llm/provider/record_test.go` | plant |

Build: the adapter's `logger`, `session`, `now`; the record built at the end of `Stream` (D15 to D21); the logger
`internal/cli/agent.go` already passes. `BenchmarkAdapterStream` run and reported at `info` and `warn`.

Gates: as Phase 1, with `go test ./llm/... ./internal/cli`.

### Phase 3: retry after an accidental cut (C rows, Q1 as re-answered, Q7)

| Rows | Test | File | Red at the base because |
| :--- | :--- | :--- | :--- |
| C1, C4, C5, C7, C8, C10, C11, C13 to C18 | `TestRetryAfterCut` and its subtests | `agent/cutretry_test.go` | a cut after an event ends the turn today (`agent/agent.go:514-517`); C3, C7, C17 are guards held by their plants |
| C2 | `TestRetryAfterCut/not_accidental` | same | a guard; plant |
| C6 | `TestRetryAfterCutRequest` | same | no second request |
| C9 | `TestRetryAfterCutHistoryIsTheFiles` | `acpserver/resume_test.go` | the turn fails; nothing written |
| C12 | `TestRetryAfterCutShows` | `acpserver` | no `cut` notice |
| C19, C20 | `TestRetryAfterCutSetting`; the schema golden of `internal/config/render_test.go` | `internal/config`, `agent` | no key |
| C21 | `TestTitleCutNotRetried` | `acpserver/title_test.go` | a guard; plant |
| C22 | `TestRetryAfterCutWritesOneEntryPerCall` | `acpserver` | plant |
| C23 | `TestModelCallRecord/cut_retry` | `llm/provider/record_test.go` | plant |
| C24 | `TestPrintAfterCutRetry` | `internal/cli/print_test.go` | a guard over `acptest.ScriptAgent`; plant |

Build (design B, all in gobble): `reply.started`, set on every provider event but a `ProviderRetry`; one case in
`next()` beside the shown guard, reusing the thinking-only rerun's path (`agent/agent.go:548-567`): write the cut
attempt per D4 when it showed something, emit `agent.Retry{Phase: start, Cause: "cut", Attempt, MaxAttempts: 3}`, wait
(the policy's doubling delay for `llm.ErrUnavailable`, `waitOnline` for `llm.ErrOffline`), `cutShort` on a cancel or
abort, then `continue`; the count of cut retries in a row, reset on `Done`; the context value read by the record
(`cut_retry`); `internal/config` (`retry.retryAfterCut`, default `true`, reaching `llm.Retry`; `Off` turns it off).

Gates: as Phase 1, over every package touched.

### Phase 4: records and docs

1. Dated notes: `0031-MADR-sdk-current-releases-and-adoption.md` D3 (D13 of the MADR);
   `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` D11 and R31, and its PLAN;
   `0066-MADR-set-each-session-model-up-from-discovered-limits.md` D8 and S21, and its PLAN.
2. Docs: `docs/architecture.md` rows for `agent`, `acpserver` and `llm/provider`; the setting (C20); the guides Phase 0
   named; `docs/README.md`'s index rows for the MADR and this PLAN.
3. Gates: `make check-records`, `markdownlint`, and the final `make preflight` on the three hosts; the record linter on
   the two records.

Return docs rows: `path:line — fixed` or `path:line — open: <reason>`, then `docs: searched <paths>`.

## Verification

* Every decided row of the MADR has its test, red at the base or held by its plant, and green after its phase.
* Every plant turns its own test red, its failing line quoted in the execution record.
* `make preflight` passes on the three hosts and `make race` on the Linux machine at the end of each phase.
* An independent review of the last build phase finds no blocker.

## Rollout and Rollback

* Rollout: the release after the last phase, a minor (`0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md`'s
  rule: a person sees new behaviour). The tag is the owner's.
* Rollback: `retry.retryAfterCut: false` turns the cut retry off at once; the record's level can be raised past
  `info` to stop it; the kept entries need no rollback, since pi and gobble both read `"error"` entries as failed.

## Execution record

~~Empty: nothing has run.~~

*(2026-11-24, approval.)* The owner approved this PLAN, relayed by the coordinator: "Approve; build now (Recommended)".
Its status moves from `proposed` to `in-progress`. The base is `50bf28a`, clean, with go-llmprovider-sdk still at
`v1.32.0`. Since the MADR was written at 97fa1f6, `0071-PLAN-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md`
Phases 1 and 1b landed (`ba8f623`, `61569d0`), touching `agent/agent.go`, `acpserver`, `llm` and the settings files.

### Phase 0, pass 1 (2026-11-24): stopped

**Step 1, citations re-grounded at `50bf28a`.** The cited code is unchanged; these lines moved:

* `agent/agent.go` (0071's `OutputLimit`, `limit`/`resendAt`/`stated`, `capResent`, `limitRefused`): `:414-417` →
  `:430-433`; `:439` → `:455`; `:328-331` → `:344-347`; `:332-340` → `:348-356`; `:342-347` → `:358-363`; `:365-367` →
  `:381-383`; `:509-510` → `:529-530`; `:511-513` → `:531-533`; `:514-517` (the shown guard) → `:534-537`; `:518-531`
  (overflow) → `:544-557`, now after 0071's cap resend at `:538-543`; `:533-547` (offline) → `:558-573`; `:548-567`
  (empty rerun) → `:574-593`; `:558-563` → `:584-589`; `:565-567` → `:591-593`; `:568-569` → `:594-595`; `:571-575` →
  `:597-601`; `:589-592` (`cutShort`) → `:637-640`; `:691-697` → `:750-756`; `:699-703` → `:758-762`; `:700` → `:759`;
  `:720-737` → `:779-796`; `:722` → `:781`; `:734` → `:793`; `:738-744` → `:797-803`; `:846` → `:905`.
* `acpserver/agent.go`: `:633-638` → `:636-641`; `:823-831` → `:828-836` (the error is now answered by
  `promptErrorNoted(…, limitLine(err, …), creditNote(err, …))`); `:845-848` → `:850-853`; `:1048-1050` → `:1074-1076`.
* `acpserver/prompt.go:272-278` → `:272-286`; `compaction/estimate.go:338-344` → `:345-354`;
  `internal/cli/render/status.go:40-49` → `:40-52`; `acpext/doc.go:112` → `:112-114`.
* Unmoved: `llm/provider/provider.go`, `acpserver/history.go`, `acpserver/choice.go` (`stamped` still replaces the whole
  `Origin`, `:250-251`), `acpserver/extensions.go`, `acpserver/commands.go:194-219`, `compaction/serialize.go`,
  `session/session.go`, `internal/config/keys.go:115-124`, `llm/llm.go:29-31`, `llm/kind.go`, `internal/logging`,
  `internal/cli/line.go:389-390`, `internal/cli/view.go:384-392`, `internal/cli/print.go`, the test seams.

**Step 2, scratch clone (`<scratch>/b0070/clone-p0`).** `agent.Message.Err` added and emitted on the shown-guard path;
`go test -count=1 ./agent ./acpserver ./internal/cli` exited 1. One red outside the MADR's rows:

```text
--- FAIL: TestMidStreamFailure (0.00s)
    agent_test.go:970: an agent.Message after a failed stream: {… Content:[{Type:text Text:par …}] … Err:llm: unavailable}
```

`TestMidStreamFailure` (`agent/agent_test.go:956-981`) scripts text then `llm.ErrUnavailable` and asserts no
`agent.Message` and one request, "a call that has shown text is not sent again". D2 (the message) and D12 (a cut of kind
`unavailable` is retried) each contradict it; no row names it. `acpserver` and `internal/cli` stayed green.

**Step 3.** The architecture test lets an external test package of `llm/provider` import `agent` (no rule of
`internal/archtest/check.go` covers that edge; rule 6, `:384-396`, refuses the `internal/cli` fallback). `toRequest`
refuses a request whose `Tools` is not JSON (`llm/provider/provider.go:654-657`). The SDK's empty answer maps to
`llm.ErrEmptyReply` (`provider.go:836`); the Responses wire's empty answer is tested at `v1.32.0`
(`llmprovider/internal/wire/responses/empty_0061_test.go`). Step 4 not run.

**Why it stopped.** The MADR does not decide how D12's cut retry composes with what 0071 put in `next()` and `call()`:

1. *The retry's output limit.* D12 (b) and C6 say the resent request is byte-identical. 0071 computes each call's cap
   again (`agent/agent.go:738-748`) by `llm.FitOutput` over `contextEstimate` (`acpserver/outputlimit.go:19-25`, `:50-61`),
   which reads the session file; `compaction.EstimateContext` counts an `"error"` entry's text (`compaction/estimate.go:333-341`;
   only its usage is ignored, `:352-353`). D4 writes the cut attempt before the retry, so a window-bound cap (basis
   `window`, `llm/limit.go:87-92`) comes out smaller and the request is not byte-identical. A cap held by a refusal
   (`t.stated`, L44) is kept by 0071 D8 (a) as amended ("every later call of that turn").
2. *0071's once-only resends across a cut retry.* `capResent`, `resent` and `empties` are local to `next()`
   (`agent/agent.go:522-524`), and D12 (b) resends with `continue` in the same loop, so a cut retry would inherit them.
   D12 (e) says a retry that "fails otherwise" ends the turn, written before 0071's cap resend existed; 0071 D8 (a)
   resends a cap refusal once per call, and L44 lists the offline, empty and overflow retries but not a cut retry.
3. *The kept entry's `errorMessage` after a cap refusal.* D2 writes "the error's text"; 0071 D8 (c) appends its line to
   the turn's error in `acpserver` (`limitLine`, `acpserver/outputlimit.go:77-83`) after the agent has emitted the
   message, so the file's text and the live error's would differ.

Nothing was built; the tree holds only this entry. The questions go to the owner through the coordinator.

### Deviation (2026-11-24): the owner's answers to the Phase 0 stop

Relayed by the coordinator, labels exact; recorded in the MADR as amendment B (Q8 to Q11) with rows G41, G42, C25, C26
and C27, before any code.

* Q1 of the stop (MADR Q8): "Exactly what the cut call sent (Recommended)". A cut retry resends the cut call's request
  unchanged, `MaxTokens` included, and does not ask `OutputLimit` again. Row C25.
* Q2 (MADR Q9): "Same call, they carry over (Recommended)". `capResent`, `resent` and `empties` carry over a cut retry.
  Rows C26 (refused on a retry before any limit resend: resent once) and C27 (after one: the turn ends, 0071 L39/L12).
* Q3 (MADR Q10): "The live text, line included (Recommended)". The D2 entry's `errorMessage` is the live error's text,
  0071's limit line included. Row G41.
* Q4 (MADR Q11): "Retarget to a non-retried failure (Recommended)". `TestMidStreamFailure` fails with an HTTP 400, asserts
  the message carries the error, and still guards "not sent again". Row G42.

Scope added: Phase 1 takes G41 (`acpserver`: the writer writes the live error's text; `acpserver/failedentry_test.go`)
and G42 (`agent/agent_test.go`, `TestMidStreamFailure`); Phase 3 takes C25 to C27 (`agent/cutretry_test.go`). Pass 1
continues after the stop.

### Phase 0 step 4, and Phases 1 and 3, pass 1 (2026-11-24): stopped again

**Phase 0 step 4.** The lines Phase 4 changes: `docs/architecture.md:111` (`agent`), `:130` (`acpserver`) and the
`llm/provider` row; `README.md:55` (the `retry.*` settings); `docs/guides/mcp-servers.md:81` (`logs/gobble.log`);
`docs/README.md`'s index rows for 0070 and its "I want to…" row at `:357`. `docs/guides/credentials.md` names no log line.

**Built so far, in the tree, not yet gated.** `agent` (`Message.Err`; `reply.started` and `kept`; D2's message before
the turn's error; D4's replaced attempts, in the turn's messages marked failed and in no request; G33's cancel; the cut
retry of D12 as amended by Q8 and Q9, `RetryCut`, the count reset on `Done`, the End at the retry's first event);
`llm` (`Retry.AfterCut`, on in `DefaultRetry`; `WithCutRetry`, `CutRetry`); `acpserver` (the writer's `"error"` entry
with the live text, D5's append failure, `stamped` keeping `Failed`, the "Error: …" line on load); `internal/config`
and `internal/cli` (`retry.retryAfterCut`, its template block, the schema golden regenerated). Phase 2 is not started.
The only red test was `TestRetryAfterCutSetting`, at the base, on its assertion:

```text
retrycut_test.go:14: retry.retryAfterCut: {Path: Kind:0 Default: …}, false; want a session bool, default true, with the record's doc
```

**Why it stopped.** `go test -count=1 ./...` through `run-heavy` on this Mac exited 1 with two reds, both in `agent`:
`TestMidStreamFailure` (G42, ruled) and one the rows do not name:

```text
--- FAIL: TestEmptyReplyRunsAgain/then_text (0.00s)
    retry_test.go:205: 3 requests, messages ["user:go" "assistant:ok"]
```

`TestEmptyReplyRunsAgain` (`agent/retry_test.go:188-280`) is cited by G5 as a guard for a call replaced before any
event. Its `then_text` case opens with `reasoningOnly()`, a thinking delta "hmm" that is shown, so D4 and G7 keep that
attempt in the turn's messages, marked failed, and `len(end.Messages)` is 3, not 2 (`:204`). Its other assertions (no
assistant message in the third request, usage 4 + 2) still hold.

A judgment also taken, for the owner to confirm: adding the key turned `internal/config`'s registry tests red, each
fixed as C19 and C20 make the key: the template block (`internal/config/settings.yaml`), the rendered document of
`TestRenderDocument` (`internal/config/render_test.go:45-51`), the schema golden (`go run ./internal/config/genschema`),
and `TestKeysMatchSettings`'s count and list (`internal/config/resolve_test.go:108-109`, 32 to 33, "and 0070-MADR D12's
retry.retryAfterCut"). No row names the last.

### Deviation (2026-11-24): the owner's answers to the second stop

Relayed by the coordinator, labels exact; recorded in the MADR as amendment C (Q12, Q13) before any further code.

* Q12: "Make it not-shown; G7 elsewhere (Recommended)". `TestEmptyReplyRunsAgain/then_text`'s first try shows nothing,
  so it guards G5; the shown attempt stays with `TestReplacedAttemptsKept`. Row G43; Phase 1 takes `agent/retry_test.go`.
* Q13: "Confirm, with a dated row (Recommended)". Row C28 names the three `internal/config` test changes the key needed;
  Phase 3 takes `internal/config/settings.yaml`, `render_test.go`, `resolve_test.go` and the schema golden.

Before going on, the coordinator asked for every remaining surprise in one stop: the full suite, `go vet` and the race
build on the current tree.

### Phases 1 to 4, pass 1 continued (2026-11-24)

**The surprise check.** `go test -count=1 ./...` through `run-heavy` on this Mac had shown only `TestMidStreamFailure`
and `TestEmptyReplyRunsAgain/then_text` red, both now ruled; `go vet ./...` exited 0, and every package compiled under
`-race` (`go test -race -run XXX_none ./...`, exit 0, run on this Mac without the queue: a compile, no test run). No
further stop.

**Built.** Phase 1: `agent` (`Message.Err`; D2's message before the turn's error, in `failOn`; D4's replaced attempts,
in `replaced`, kept in the turn's messages marked failed and left out of every request; G33's cancel, in `rerunEmpty`),
`acpserver` (the writer's `failedEntry`: `"error"`, the live text through `limitLine`, D5's held append error;
`stamped` keeps `Failed`; the "Error: …" line in `replayMessage`). Phase 2: `llm/provider/record.go` (`health`,
`begin`, `record`), the adapter's `logger`, `session`, `now`, and `streamFailed`, split out of `attempt`. Phase 3:
`agent` (`reply.started`, `cutRetry`, `retryCut`, `RetryCut`, the count reset on `Done`, the End at the retry's first
event, Q8's kept limit, Q9's carried counts), `llm` (`Retry.AfterCut`, on in `DefaultRetry`; `WithCutRetry`,
`CutRetry`), `internal/config` and `internal/cli` (`retry.retryAfterCut`, C28's three test changes). Lint fixes
without suppression: `run`, `next` and `call` take Run's ctx (contextcheck); `rerunEmpty` and `failOn` split out of
`next`, `streamFailed` out of `attempt`, the writer's append-failure path out of `Prompt` (gocognit, gocyclo); the
large tests' subtests became named functions; `jsonx` in place of `encoding/json` in tests.

**Rows to tests.** Red at `50bf28a` on their assertions (the test files copied onto a clean clone of the base):
`TestRetryAfterCutSetting` (internal/config), `TestFailedEntryShape`, `TestFailedEntryKeepsLimitLine`,
`TestFailedEntryAppendFails`, `TestLoadReplaysFailedEntry/{text,error_line,no_text,calls}`,
`TestLastAssistantTextAfterFailure`, `TestLastAssistantTextAfterEmptyFailure`, `TestForkAfterFailedEntry`,
`TestRetryAfterCutShows`, `TestRetryAfterCutWritesOneEntryPerCall`, `TestPrintAfterCutRetry`,
`TestModelCallRecordSurvivesRedaction`, `TestModelCallRecordPerRerun`. Guards green at the base and held by their
plants: `TestShownFailureHistoryIsTheFiles`, `TestReplacedAttemptHistoryIsTheFiles`, `TestCancelInWaitHistoryIsTheFiles`,
`TestRetryAfterCutHistoryIsTheFiles`, `TestLoadReplaysFailedEntry/aborted`, `TestFailedEntryCounted`,
`TestSerializeKeepsFailedEntry`, `TestTitleCutNotRetried`. The rest name new API (`Message.Err`, `Retry.AfterCut`,
`CutRetry`, the adapter's record) and take their red from their plants. Every test is green after the build.

**Plants.** 85, each one production change (C10 and C12-2 a change in two places of one function) on a scratch copy of
the tree, by `<scratch>/b0070/plant.py` over `plants.py`; each turned its own row's test red, its first failing line
quoted in `<scratch>/b0070/plants-final.log` and `plants-final2.log`; 0 uncaught. Four plants needed a second round:
G7-live and G33-live (the history tests now use a signed thinking block, which the wire would send were it not
failed), C22 (the test gained the abort of a cut retry's wait, the one path where a kept attempt meets D2) and C25 (the
plant made nil-safe).

**Deviations from the PLAN as written, recorded here.** (1) Test files: the Phase 1 acpserver rows sit in
`acpserver/failedentry_test.go`, the Phase 3 acpserver rows in `acpserver/cutretry_test.go`, C24 in
`internal/cli/cutretry_test.go`, G27 in `compaction/failedentry_test.go`, and T15 in `llm/provider/rerun_test.go`
(external package), not spread over the existing files the tables name. (2) Seams: C24 runs through the real agent
(`withModel`) rather than `acptest.ScriptAgent`, which sends no `_gobble/retry` notice; T19, T20 (`warnings`) and T21
use an in-package SDK streamer or the SDK's `llmtest.Fake` under the adapter, since no wire answer makes the SDK add a
warning and the allocation count needs no HTTP. (3) Plants adapted where the MADR's wording does not map onto the
built code: G29 (the failed entry not written), G34 (D2 skipped once an attempt was replaced), G35 (replaced attempts
dropped when compaction is on), C9 (the cut attempt emitted with no `Err`, as `end_turn`), C23 split in two (the
agent's context value and the record's read of it, with `TestRetryAfterCutRequest/cut_retry_context` added). (4) The
phases' gates ran once, over the whole build, not after each phase: nothing is staged between phases.

**Gates (2026-11-24).**

| Command | Host | Exit |
| :--- | :--- | :--- |
| `make pre-add-check FILES="<the 26 .go files>"` through `run-heavy` | this Mac | 0 (gofmt, golangci-lint, govulncheck clean; a first run, 2, gave 11 findings, each fixed without a suppression) |
| `go test -count=1 ./...` through `run-heavy` | this Mac | 0 |
| `make preflight` through `run-on-host --need darwin` | this Mac | 0 |
| `make preflight` through `run-on-host --need linux --race` | the Linux machine (not WSL) | 0 |
| `make preflight` through `run-on-host --need windows` | the Windows laptop | 0 |
| `make race` through `run-on-host --need linux --race` | the Linux machine (not WSL) | 0 (no data race; both modules) |
| `make check-test-isolation` through `run-heavy` | this Mac | 0 |
| `make check-records`; `make markdownlint` | this Mac | 0; 0 |
| `python3 -B <gitrepos>/go/go-llmprovider-sdk/tools/denyscan.py --changed` | this Mac | 0 (0 hits) |

No log reports a fallback. `BenchmarkAdapterStream` (100 deltas, this Mac): info 15,289 ns/op, warn 3,493 ns/op, no
logger 3,423 ns/op; 127 allocations each.

**Docs.** `docs/architecture.md` (the `agent`, `llm`, `llm/provider` and `acpserver` rows), `README.md` (the retry
line), `docs/README.md` (the 0070 index rows, a new "I want to…" row), `internal/config/settings.yaml` (the template's
retry block). `docs/guides/mcp-servers.md:81` stays true; `docs/guides/credentials.md` names no model call.

**Records.** Dated notes: `0031-MADR-sdk-current-releases-and-adoption.md` D3 (D13);
`0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` D11 and R31, and its PLAN;
`0066-MADR-set-each-session-model-up-from-discovered-limits.md` D8 and S21, and its PLAN.

**Not done.** The independent review of the last build phase (Verification); the coordinator's to order.
*(2026-11-24: done; see below.)*

*(2026-11-24, queue note.)* The `-race` compile of the surprise check (`go test -race -run XXX_none ./...`) ran on this
Mac outside `run-heavy`: a queue miss, since any `-race` run is heavy. It compiled and ran no test; every later heavy run
went through `run-heavy` or `run-on-host`.

### The review's notes, pass 1 continued (2026-11-24)

The independent review passed the build with notes, relayed by the coordinator; each closed here.

1. *D12 (d).* A call sent again after a cut that fails before any event now ends the cut's notice at once with the
   failure's text (`settleCut`), whatever path follows; before, an offline wait ended it as a success and a limit
   resend or a compaction left it open. `TestRetryAfterCutNoticeEnds/{offline,limit_refused,overflow}`; plant
   R1-notice-end caught (`cutretry_test.go:476: err <nil>, cut ends [""]; want one, with the offline failure`).
2. *Q10.* The saved text is `errorData(provider, limitLine(…)).Error`, the live error's, so a workspace refusal keeps
   `workspaceAdvice`. `TestFailedEntryKeepsWorkspaceAdvice`; plant R2-workspace caught (`failedentry_test.go:104:
   errorMessage "llm: api status 403: …", want the live error's "claude needs an Anthropic workspace …"`).
3. *The template.* The retry block says "up to maxDelayMs"; the schema golden, from the keys, is unchanged.
4. *Q8 with Q9.* Every resend of a cut call until it ends keeps its `MaxTokens` and asks no limit again (`turn.same`),
   the resend after an offline wait that follows a failed cut retry included; a limit refusal's stated value still
   takes its place, and a compacted request asks again (0071 D8 (a), L11). Dated annotations on Q8's and C25's rows.
   `TestRetryAfterCutKeepsLimitAfterOffline`; plant R4-kept-limit caught (`cutretry_test.go:519: caps [9999 9999
   9998] after 2 asks`).
5. *An offline cut with no prober, or `offlineWaitMs: 0`* (`agent/agent.go:734-741`): `retryCut` sleeps the doubled
   `BaseDelay` and resends. No row decides it: D12 (a) admits every `offline` cut, D12 (b) and C8 give only "`offline`:
   `waitOnline` within `OfflineWait`", and C19 covers only `retryAfterCut` and `Off`; while the offline path proper
   ends the turn at once in both cases (`TestOfflineWait` "no wait", "no probe"), and the setting says "0 does not
   wait". Stopped for the owner; no code changed.
6. *Wording.* `docs/README.md`'s row says "built; awaiting the coordinator's gate"; nothing is staged.

*(2026-11-24, deviation: the owner's answer to note 5, relayed by the coordinator, "Keep the short wait and resend".)*
Recorded as MADR amendment D, Q14, row C29, before any further change: an `offline` cut with no prober, or with
`offlineWaitMs: 0`, waits the doubled `BaseDelay` and is sent again within the cut cap, as built; no code change.
Scope added: `TestRetryAfterCutOfflineNoWait` (`agent/cutretry_test.go`) and its plant; the docs of
`retry.offlineWaitMs` (`internal/config/keys.go`, `settings.yaml`, the schema golden) and of the retry setting in
`README.md` and `docs/architecture.md`, which say a cut reply is still sent again after the retry delay.

**Gates of the notes (2026-11-24).** `make pre-add-check FILES="agent/agent.go agent/cutretry_test.go
acpserver/agent.go acpserver/failedentry_test.go"` through `run-heavy`: 0 (a first run, 2, found `next`'s cognitive
complexity at 51; `settleCut` was split out); all 88 plants through `run-heavy`: 88 caught; `go test -count=1 ./...`
through `run-heavy`: 0; `make check-records`, `make markdownlint`, `denyscan.py --changed`: see the hand-back.

