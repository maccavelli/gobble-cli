---
status: proposed
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

Empty: nothing has run.
