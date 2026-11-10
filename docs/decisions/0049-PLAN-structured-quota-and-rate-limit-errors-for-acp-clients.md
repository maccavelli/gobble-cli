---
status: in-progress
date: 2026-11-09
associated-madr: "0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement sending ACP clients a failed turn's limit as data

Associated MADR: [0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md)

## Goal

* acpext `v0.3.0` carries the MADR's D1 and D2 members; `gobble acp` fills them from gobble's `llm` facade (D3 to D7).
* A 401 quota is no longer worded as a refused key (D8).
* gobble's own provider marks and client follow the owner's answers to Q2 and Q3.
* When go-llmprovider-sdk tags #13's fields, they replace gobble's reconstructions (D10, Q4).

The plan is complete when the MADR's Confirmation 1 to 10 is met, Q4's phase included, or, if the owner closes Q4's
phase without a tag, when 1 to 9 are met and the record says why 10 was not.

## Scope

**Base.** `main` at `74d7f66`, equal to `origin/main`, clean, when this plan was written; acpext `v0.2.0`;
go-llmprovider-sdk `v1.28.0`; acp-go-sdk2 `v1.34.2`.

**In scope,** by phase:

* **P0, the records:** the owner's answers to Q1 to Q5 in the MADR, the MADR `accepted`, this plan `in-progress`, and
  `docs/README.md`'s rows.
* **P1, acpext `v0.3.0`** (the acpext module only):
  * `acpext/types.go`: `ErrorKind`, `LimitScope`, `ErrorData`'s members, `RetryNotice`'s members;
  * `acpext/doc.go`: the error and retry paragraphs;
  * `acpext/roundtrip_test.go`, `acpext/example_test.go`, new frames in `acpext/testdata/`.
* **P2, `llm` and `llm/provider`** (the root; no acpext import):
  * `llm/errors.go` (`APIError`'s fields), a new `llm/kind.go` (`Kind`, the `Limit*` constants), `llm/event.go`
    (`ProviderRetry`'s fields), `llm/doc.go`;
  * `llm/provider/provider.go` (`mapError`'s filling, the limit table, the attempts count, `rejected`), and new tests in
    `llm/provider/` and `llm/`.
* **P3, `agent` and `acpserver`:** `go.mod`/`go.sum` (acpext `v0.3.0`); `agent/event.go` and `agent/agent.go`
  (`Retry`'s fields); `acpserver/prompt.go`, `acpserver/agent.go`, `acpserver/compact.go`, `acpserver/extensions.go`;
  new tests in `acpserver/`.
* **P4, the owner's answers to Q2 and Q3,** each built only as answered:
  * Q2 (a): `acpserver/choice.go`'s mark;
  * Q3 (a) or (c): `acpclient/conn.go` (and a new `acpclient/error.go`), `acpclient/acptest/script.go`,
    `internal/cli/line.go`, `internal/cli/print.go`.
* **P5, go-llmprovider-sdk's #13 fields** (Q4 (a)): `go.mod`/`go.sum`, `llm/provider/`, and the MADR's D10, when the
  owner names the tag.
* **P6, acceptance and the records;** `docs/architecture.md`'s rows for `acpext`, `llm`, `llm/provider` and `acpserver`.

**Out of scope:** Q1's notice, unless the owner answers (b) or (c), which then adds a step to P3 by amendment; any
member the MADR's "Not decided here" lists; magic-cli-remote's code; go-llmprovider-sdk's code; `llm/provider.FailureKind`'s
outputs (MADR D3); a gobble release tag; a push; posting on #14 or #13 without the owner's word.

## Conventions every phase follows

* **C1. Start state.** Each phase first re-reads `git status`, `HEAD`, `origin/main`, and the go-llmprovider-sdk and
  acpext versions `go.mod` requires. A difference from this plan is a deviation.
* **C2. Seen failing.** Every new test and assertion is seen failing on a scratch copy of the tree, never in the working
  tree: against the tree before the phase where that compiles, and against each planted break the phase lists. The
  record quotes each failure, with temporary paths as `<TMP>`.
* **C3. Evidence.** Commands write to a scratch log, with their exit status taken before any filter. Scratch runs set
  `HOME`, `USERPROFILE` and `XDG_*` into the scratch directory, explicit `GOCACHE`/`GOMODCACHE`, and the proxies at
  `127.0.0.1:9`.
* **C4. Deviations stop the phase,** are recorded here with a date, and in the MADR when a decision or a fact changes,
  before work goes on.
* **C5. No real effects.** Every test and probe stays on loopback: canned wires (`llm/provider/wire_test.go`'s
  `cannedWire`), go-llmprovider-sdk's `sdktest` fakes, `llmtest` scripts, and `acptest` connections. P6's acceptance
  runs `gobble` against a loopback server through `models.json`'s `baseUrl`, with a dummy key. Nothing in this plan
  calls a real provider.
* **C6. Gates and commits.**
  * Every changed Go file passes `make pre-add-check FILES=…`.
  * The tree passes `make preflight` and `make race`; the identifier scan of the diff finds nothing.
  * The Linux machine and the Windows laptop run `go test ./...`, the race tests and acpext's own tests on a scratch
    copy, at P1, P2, P3, P4 and P5.
  * The agent stages; the owner commits with `git commit --no-edit`. The disclosure guard runs before any push.
* **C7. Identifiers.** No hostname, account name or real-machine path in a committed file. Dev hosts are named by
  platform.
* **C8. Order of the modules** (0042-MADR D5). P1's acpext change is committed, pushed and tagged `acpext/v0.3.0` by the
  owner, and served by the module proxy, before P3 requires it. P2 needs no acpext change and may run while P1's tag
  waits.

## Implementation Steps

### P0 — the records

1. Record the owner's answers to Q1 to Q5 in the MADR under "Answers (date)", set it `accepted`, and set this plan
   `in-progress`. An answer that changes a step is written into the step, struck through where it replaces text: Q1 (b)
   or (c) adds P3 step 4b; Q2 (b) or (c) and Q3 (b) change or drop P4's parts; Q4 (b) drops P5; Q5 (b) changes P2
   step 3's `ResetAt` and P3's `resetAt`.
2. `docs/README.md`: the 0049 rows' states. Its "I want to…" row for a failed turn's error, added with this proposal,
   stays as it is.
3. `make check-records markdownlint` passes. Staged; the owner commits.

### P0 record (2026-11-09)

C1: `HEAD` and `origin/main` at `74d7f66`; the only changes were this pair and its `docs/README.md` rows, uncommitted;
go-llmprovider-sdk `v1.28.0`, acpext `v0.2.0`. No deviation.

1. The owner's answers, 2026-11-09: "Q1: A, Q2: A, Q3: A, Q4: A, Q5: A", in the MADR's "Answers (2026-11-09)"; the
   MADR `accepted`, this plan `in-progress`. Every answer is the option the steps were written for, so no step changes:
   P3 step 4b is not run (Q1 (a)); P4a runs (Q2 (a)); P4b runs with its print-mode step 4 (Q3 (a)); P5 stays, waiting
   for the owner to name the SDK's tag (Q4 (a)); P2 step 2 and P3 step 3 rebuild `ResetAt` (Q5 (a)).
2. `docs/README.md`: the 0049 rows' states; the "I want to…" row added with the proposal stays.
3. Gates: `make check-records markdownlint`, below. No Go file changed.

P1 runs on the owner's word.

### P1 — acpext `v0.3.0` (MADR D1, D2)

1. **`acpext/types.go`:**
   * `type ErrorKind string`, with constants `KindWorkspace` `workspace`, `KindQuota` `quota`, `KindRateLimited`
     `rate-limited`, `KindContextOverflow` `context-overflow`, `KindEmptyReply` `empty_reply`, `KindIncomplete`
     `incomplete`, `KindAuth` `auth`, `KindNotPermitted` `not-permitted`, `KindOffline` `offline`, `KindUnavailable`
     `unavailable`, `KindUnsupported` `unsupported`, `KindTimeout` `timeout`, `KindCanceled` `canceled`, `KindOther`
     `other`. Its doc: an open set, named as `_gobble/retry`'s causes where both name a class.
   * `type LimitScope string`, with `LimitRate`, `LimitWindow`, `LimitDaily`, `LimitWeekly`, `LimitMonthly`,
     `LimitCredits`, `LimitFreeTier` (`free_tier`), `LimitPlan`. Its doc: an open set; `window` and `weekly` are sent
     only when a source states the window.
   * `ErrorData`: `Attempts int`, `Code string`, `Command`, `Error`, `HTTPStatus int` (`httpStatus`), `Kind ErrorKind`,
     `Limit LimitScope`, `Provider`, `ProviderMessage string`, `Reason`, `ResetAt string`, `RetryAfterMs int64`,
     `Retryable *bool`, `SessionID`, declared in that order, every tag `omitzero`. Its doc names which errors carry which
     members, that `resetAt` is RFC 3339 in UTC to the second, and that `retryable` is absent when unknown.
   * `RetryNotice`: after `Error`, `RetryAfterMs int64` (`retryAfterMs`), `ServerDirected bool` (`serverDirected`),
     `Code string` (`code`), `HTTPStatus int` (`httpStatus`), each `omitzero`; its doc says `delayMs` is the jittered
     wait taken and `retryAfterMs` the server's.
2. **`acpext/doc.go`:** the "refused request" paragraph names the new members, the -32603 a failed turn answers with,
   and the over-cap case (MADR D2's second paragraph).
3. **Tests:**
   * `TestErrorDataSortedMembers` sets all fourteen members and wants
     `{"attempts":3,"code":"c","command":"c","error":"e","httpStatus":429,"kind":"quota","limit":"free_tier","provider":"p","providerMessage":"m","reason":"r","resetAt":"2026-11-09T14:05:00Z","retryAfterMs":5601000,"retryable":false,"sessionId":"s"}`.
   * New frames, written from the JSON tags as P3 will send them, each an entry in `frames` with `of[acpext.ErrorData]`
     or `of[acpext.RetryNotice]`:
     * `prompt-quota.error.json`: MADR Confirmation 2's data, with `resetAt` `2026-11-09T14:05:00Z`;
     * `prompt-rate-over-cap.error.json`: `kind` `rate-limited`, `limit` `rate`, `httpStatus` 429, `retryAfterMs`
       120000, `retryable` true, `resetAt`, `provider`, `error`;
     * `prompt-after-retries.error.json`: `kind` `unavailable`, `httpStatus` 503, `attempts` 4, `retryable` true;
     * `retry.start.provider.notice.json`: a provider retry's start with `retryAfterMs`, `serverDirected` true, `code`
       and `httpStatus`.
   * `TestFramesRoundTrip` and `TestFramesCoverEveryMethod` take them as they take today's frames.
   * `ExampleErrorData_limit`: decodes `prompt-quota.error.json`'s object and prints `limit`, `resetAt` and
     `providerMessage`.
   * `TestKindsAndLimits`: `ErrorKind`'s and `LimitScope`'s constants are the MADR's names, in a fixed list, so that P3's
     cross-module test has something to compare.
4. **Seen failing** (C2), each on a scratch copy of the module:
   * the module before P1: `undefined: acpext.ErrorKind` (and the new fields), at compile;
   * `retryAfterMs` misspelt `retryAfterMS` in a tag: `TestFramesRoundTrip`'s unknown-member refusal on the new frames;
   * `Retryable` as `bool` with `omitzero`: `prompt-quota.error.json`'s `"retryable":false` does not re-encode, so the
     round trip fails;
   * `Limit` declared before `Kind`: `TestErrorDataSortedMembers` fails;
   * `LimitFreeTier` as `free-tier`: `TestKindsAndLimits` fails.
5. **apidiff,** as 0016-PLAN P3a ran it, with the module's tool on export data of `acpext/v0.2.0` and of the change:
   only compatible additions (`ErrorKind`, `LimitScope`, their constants, and the new fields), nothing incompatible.
6. **Gates** (C6): `make pre-add-check FILES="acpext/types.go acpext/doc.go acpext/roundtrip_test.go
   acpext/example_test.go"`; `make check-acpext-deps` (the standard library only); `make lint`; `make preflight`;
   `make race`; acpext's tests on the Linux machine and the Windows laptop.
7. **Staged; the owner commits and pushes, then tags** `acpext/v0.3.0` on that commit, annotated, in `acpext/v0.2.0`'s
   form, and pushes the tag under 0043-MADR's ruleset. The record notes when
   `go list -m github.com/maccavelli/gobble-cli/acpext@v0.3.0` succeeds through the proxy. **Gate for P3.**

### P2 — `llm` and `llm/provider` (MADR D3 to D6, D8)

1. **`llm`:**
   * `llm/errors.go`: `APIError` gains `Provider string`, `Code string`, `Message string`, `Retryable *bool`,
     `ResetAt time.Time`, `Limit string` and `Attempts int`, after its present fields, each documented as zero when
     unknown and filled only by a provider adapter. `Error()` and `Unwrap()` are unchanged, so every error's text is.
   * New `llm/kind.go`: `Kind(err error) string`, with the MADR's D3 order, and the constants `LimitRate`,
     `LimitWindow`, `LimitDaily`, `LimitWeekly`, `LimitMonthly`, `LimitCredits`, `LimitFreeTier`, `LimitPlan` with
     acpext's values.
   * `llm/event.go`: `ProviderRetry` gains `RetryAfter time.Duration`, `ServerDirected bool`, `Code string`, `Status int`.
   * `llm/doc.go`: the error section names `Kind` and the new fields.
2. **`llm/provider/provider.go`, the mapping:**
   * `mapError` becomes `mapErrorAt(id llmprovider.ProviderID, err error, now time.Time) error`; `mapError(err)` stays
     for callers with no provider (`providers.New`'s failure, `:221`), as `mapErrorAt("", err, time.Now())`. The stream
     path (`:347`) and the notice hook (`:323-327`) call `mapErrorAt(a.id, …, time.Now())`.
   * On an SDK `*APIError`, the returned `*llm.APIError` gets `Provider` (gobble's id), `Code`, `Message`,
     `Retryable` (`ae.Retryable()`), `Limit` (step 3) and `ResetAt` (`now.Add(ae.RetryAfter)` when positive, else zero),
     beside `Status` and `RetryAfter` as today.
3. **The limit table,** `limitOf(id llmprovider.ProviderID, ae *llmprovider.APIError) string`, beside `sentinels`, with
   the MADR's D4 rows: by code for the seven codes and the four plan codes; `credits` for HTTP 402; `daily` for Gemini's
   exhausted quota on 429; `rate` for `ErrRateLimited` that is not exhausted quota; empty otherwise. It never reads
   `RetryAfter`.
4. **Attempts.** `attempt` keeps the last retry notice's `Attempt`; when the call ends in an error after one or more
   notices, the returned `*llm.APIError` (if any) gets `Attempts` = that `Attempt` + 1.
5. **The retry notice.** The hook (`:323-327`) fills `ProviderRetry`'s new fields from `n.ServerDirected` and, from the
   mapped `n.Err`, `RetryAfter`, `Code` and `Status`.
6. **`rejected`** (`:275-285`) returns `mapped` unchanged when `errors.Is(err, llmprovider.ErrQuotaExhausted)`, before
   either of its branches (MADR D8).
7. **Tests** (`llm/kind_test.go`, `llm/provider/limit_test.go`, and additions to `wire_test.go` and `provider_test.go`):
   * `TestKind`: every `llm` sentinel and both context errors give their name; an empty reply wrapping
     `ErrIncomplete` gives `empty_reply`; quota beside rate-limited gives `quota`; `nil` and a plain error give `other`.
   * `TestKindAgreesWithFailureKind`: for each class `FailureKind` names, `llm.Kind` gives the same string.
   * `TestWireLimits`: through `New` and a `cannedWire`, each row: OpenCode Zen 429 `FreeUsageLimitError` with
     `Retry-After: 5601` (free_tier, `Code`, `Message` the provider's sentences alone, `Retryable` false, `Provider`
     `opencode-zen`, `ResetAt` within a second of now + 5601 s); OpenCode Zen 401 `MonthlyLimitError` (monthly);
     `CreditsError` (credits); OpenCode Go `GoUsageLimitError` (plan); OpenAI 429 `insufficient_quota` (credits); OpenAI
     429 `usage_limit_reached` with a `resets_at` (plan, `ResetAt` that instant to the second); Kilo 429
     `PROMOTION_MODEL_LIMIT_REACHED` (free_tier); Kilo 402 `insufficient_balance` (credits); Gemini 429 with a per-day
     `QuotaFailure` (daily); Gemini 429 without one (rate); OpenCode `UserLimitError` (empty); Kilo's
     `FreeUsageLimitError` in the message alone (empty).
   * `TestWireLongWaitIsNotAWindow`: a 429 rate limit with `Retry-After: 18000` gives `rate`, never `window`.
   * `TestRetryNoticeCarriesServerWait`: an `sdktest` fake failing once with a 429 `RetryAfter` 2 ms under a 1 s cap,
     then answering: one `ProviderRetry` with `RetryAfter` 2 ms, `ServerDirected` true, `Code` and `Status` 429.
   * `TestAttemptsAfterRetries`: a fake failing four times with a retryable 503 under `MaxRetries` 3: the error's
     `Attempts` is 4.
   * `TestWireRejectedKey` gains: OpenCode Zen 401 `MonthlyLimitError` with `CredentialSource` `auth.json` is
     `llm.ErrQuota` and its text does not contain "rejected the API key"; the same through a stored login does not
     contain `LoginRejected`'s sentence.
8. **Seen failing** (C2):
   * the tree before P2, with only `TestWireRejectedKey`'s new cases (they compile there): the 401 quota case fails
     with "opencode-zen rejected the API key from auth.json: …" (MADR F9, now observed);
   * `insufficient_quota`'s row removed: `TestWireLimits/openai-insufficient_quota` fails;
   * a planted inference, `RetryAfter > 4h` → `window`: `TestWireLongWaitIsNotAWindow` fails;
   * `incomplete` before `empty_reply` in `Kind`: `TestKind` fails;
   * `ResetAt` left zero: the free-tier and `usage_limit_reached` rows fail;
   * `Attempts` = the last `Attempt` (no + 1): `TestAttemptsAfterRetries` fails;
   * `ProviderRetry.RetryAfter` from `n.Wait`: `TestRetryNoticeCarriesServerWait` fails (the jittered wait is not 2 ms);
   * `rejected`'s quota guard removed: `TestWireRejectedKey` fails.
9. **Gates** (C6), with `internal/archtest` unchanged and passing. Staged; the owner commits.

### P3 — `agent` and `acpserver` (MADR D2, D7)

1. **C1, and the gate:** acpext `v0.3.0` served. `go get github.com/maccavelli/gobble-cli/acpext@v0.3.0`, `go mod
   tidy`: `go.mod` changes that one line, `go.sum` its two. Anything else moving is a deviation.
2. **`agent`:** `Retry` (`agent/event.go:68-75`) gains `RetryAfter time.Duration`, `ServerDirected bool`, `Code string`,
   `Status int`; `agent/agent.go:693-700` copies them from `llm.ProviderRetry`.
3. **`acpserver/prompt.go`:**
   * The `agent.Retry` case (`:105-108`) writes `RetryAfterMs`, `ServerDirected`, `Code` and `HTTPStatus`.
   * `promptError(err)` becomes `promptError(provider string, err error)`, building `ErrorData` as the MADR's D7 says,
     through a new `errorData(provider string, err error) acpext.ErrorData`: `Kind` from `llm.Kind`; from
     `errors.AsType[*llm.APIError]`, `HTTPStatus`, `Code`, `ProviderMessage` (`Message`), `RetryAfterMs`, `ResetAt`
     (`UTC().Truncate(time.Second).Format(time.RFC3339)` when not zero), `Retryable`, `Attempts` (when 2 or more) and
     `Limit`. The workspace case keeps `workspaceAdvice` and leaves `ProviderMessage` empty.
4. **The five callers** pass the session's provider, `a.provOf(s)`: `acpserver/agent.go:630`, `:751`,
   `acpserver/compact.go:28`, `:44`, `acpserver/extensions.go:204`.
   * *4b, only if Q1 is (b) or (c):* the step the amendment writes.
5. **Tests** (`acpserver/errordata_test.go`, additions to `retry_test.go` and `extensions_test.go`):
   * `TestErrorData`: a table over `errorData`: the full quota `*llm.APIError` → every member; a rate limit over the cap
     (`retryable` true, `retryAfterMs` 120000, no `attempts`); a 503 after retries (`attempts` 4); `llm.ErrOffline`
     (`kind` `offline`, `provider`, nothing else); an empty reply (`empty_reply`); the workspace refusal (fixed advice,
     no `providerMessage`, `kind` `workspace`); `llm.ErrAuth` (`provider`, `kind` `auth`); a plain error (`other`).
   * `TestPromptQuotaData`: a connected agent whose `llmtest` provider fails with the quota `*llm.APIError` of
     Confirmation 2 (its `ResetAt` fixed at 2026-11-09T14:05:00Z): the prompt's -32603 data is byte-equal to
     `acpext/testdata/prompt-quota.error.json`'s canonical encoding; so are the over-cap and after-retries frames.
   * `TestCompactQuotaData`: `/compact` and `_gobble/compact` failing on the same error carry the same data.
   * `TestRetryNoticeServerMembers`: a scripted `llm.ProviderRetry` with the new fields gives a `_gobble/retry` start
     equal to `acpext/testdata/retry.start.provider.notice.json` but for the session id.
   * `TestKindsAndLimitsMatchAcpext`: `llm`'s `Limit*` constants equal acpext's `LimitScope` constants, and
     `llm.Kind` of each `llm` sentinel and both context errors is one of acpext's `ErrorKind` constants.
   * Today's tests keep passing unchanged: `retry_test.go`'s `errText` reads `error`, which is unchanged.
6. **Seen failing** (C2):
   * the tree before P3, with only the new tests: `undefined: errorData` and the `promptError` arity, at compile;
   * the provider not passed (`""`): `TestErrorData` and `TestPromptQuotaData` fail;
   * `ProviderMessage` set on the workspace refusal: `TestErrorData/workspace` fails;
   * `Retryable` dropped: the quota case fails (`retryable` absent, want false);
   * `ResetAt` in local time: the frame's bytes differ;
   * one notice member unwritten (`ServerDirected`): `TestRetryNoticeServerMembers` fails;
   * one caller left on the old data (`extensions.go:204` with no provider): `TestCompactQuotaData` fails.
7. **Goldens:** `git diff --quiet 74d7f66 -- acpserver/testdata acpclient/acptest/testdata internal/cli/testdata`
   exit 0: no recorded session has a failed turn, so none changes. A change is a deviation.
8. **Gates** (C6), archtest unchanged: `acpserver` imports neither `llm/provider` nor the SDK. Staged; the owner
   commits.

### P4 — the owner's answers to Q2 and Q3

Each part runs only as answered; a part answered "not in this record" is recorded as not done, with the answer.

#### P4a — Q2 (a): the mark keeps its reset time

1. `mark` gains `resetAt time.Time`, from the failure's `*llm.APIError.ResetAt`.
2. `StateOf`'s reason for `StateQuotaExhausted` and a rate-limited mark gains " — resets HH:MM" in the local clock,
   from the agent's `now`; nothing of the provider's text (0025-MADR D4).
3. A mark whose `resetAt` has passed, by the agent's `now`, is dropped when the provider list is built, so the provider
   is offered again.
4. Tests, with `Options.Now` fixed: the reason before the time; the mark gone after it; a quota with no `resetAt` (credits)
   kept as today. Seen failing: the lapse not applied; the time from `ResetAt`'s zero value.

#### P4b — Q3 (a), or (c): gobble's own client

1. `acpclient`: a new `PromptError` type, returned for a refused `session/prompt` whose data decodes as
   `acpext.ErrorData`: `Code int`, `Data acpext.ErrorData`, and the `*acp.RequestError` it wraps (`Unwrap`). Its
   `Error()` is `Data.Error`, or the request error's text when `Error` is empty. `ErrAuthRequired` still wraps it, so
   `errors.Is(perr, acpclient.ErrAuthRequired)` holds as today.
2. `acpclient/acptest`: `Turn` gains `ErrData string`, the error's data as JSON, so CLI tests can script it.
3. `internal/cli/line.go:385-391`: prints `PromptError`'s text; when `Data.Limit` or `Data.ResetAt` is set, one more
   line: "<limit words> on <provider>; resets at HH:MM, in <duration>", or without the reset part. The limit words come
   from a fixed table in `internal/cli` for the eight scopes, with the scope itself for one it does not know.
4. Q3 (a) only: `internal/cli/print.go`: `printResult` gains `ErrorData *acpext.ErrorData` (`errorData`, `omitzero`),
   set from a `PromptError`; `error` becomes `PromptError`'s text. The record notes that print mode's `error` text
   changes from today's `session/prompt: {…}`.
5. Tests: a line session scripted with a quota error's data prints the provider's text and the limit line; print mode's
   JSON has `errorData` equal to the scripted object; an error with no data prints as today; the auth path is unchanged.
   Seen failing: the data not decoded (`Error()` is the JSON); the limit line missing; `errorData` absent.

### P5 — go-llmprovider-sdk's #13 fields (MADR D10; Q4 (a))

Starts when the owner names the go-llmprovider-sdk tag that carries #13's fields. Before step 1, this phase and the
MADR's D10 are amended, with a date, to the tag's names and to what it actually carries; until then the steps below name
the fields by their purpose.

1. `scripts/check_libs.py --lib go-llmprovider-sdk --tag <tag>`: 0 failing checks (0035-MADR D2). `go get`, `go mod
   tidy`: only go-llmprovider-sdk moves.
2. `llm/provider`: `ResetAt` from the SDK's absolute instant when it is set, else D5's rebuild; `daily` from the SDK's
   per-day flag, replacing the Gemini inference; a stream failure's wait and instant into the same fields.
3. A stated window, if the tag gives one: `window` or `weekly`, and, if a length is wanted on the wire, a further acpext
   minor tag by amendment of the MADR (its "Not decided here").
4. Tests: the SDK's instant wins over the rebuild (a `resets_at` 3 hours ahead with a 1-hour `Retry-After`); a Gemini
   per-day 429 is `daily` by the flag; a stream failure carries `retryAfterMs` and `resetAt`. Seen failing: each with
   the mapping removed.
5. Gates (C6). Staged; the owner commits.

### P6 — acceptance

1. **Goldens:** P3 step 7's check, again at P6's base.
2. **A scratch run on loopback** (C5), on macOS here, the Linux machine and the Windows laptop: a scratch home whose
   `models.json` points `opencode-zen`'s and `openai`'s `baseUrl` at a local server, `OPENCODE_API_KEY` and
   `OPENAI_API_KEY` set to dummy values, the binary built from the tree. A Python driver speaks ACP to `gobble acp` over
   stdio (initialize, session/new, session/prompt) and records each frame:
   1. a 429 `FreeUsageLimitError` with `Retry-After: 5601`: the -32603's data as Confirmation 2 says, `resetAt` within
      a second of the request plus 5601 s;
   2. a 401 `MonthlyLimitError`, with the key in `auth.json`: -32603, `kind` `quota`, `limit` `monthly`, and `error`
      not saying the key was rejected;
   3. a 429 rate limit with `Retry-After: 1` once, then a reply: one `_gobble/retry` start with `retryAfterMs` 1000,
      `serverDirected` true, `code` and `httpStatus` 429, then the reply;
   4. a 429 with `Retry-After: 120` (over the 60 s cap): no `_gobble/retry`, and a -32603 with `retryable` true,
      `retryAfterMs` 120000 and `resetAt`;
   5. Q3 (a) only: `gobble -p hi --output-format json` against case 1: `errorData` as case 1's data, `error` its text.
   The record quotes each frame, with the session id and temporary paths replaced.
3. **The records:** the MADR's Confirmation item by item in a "Completion" section with commits and evidence; this plan
   `complete` with a P-record per phase; `docs/README.md`'s rows; `docs/architecture.md`'s rows for `acpext`, `llm`,
   `llm/provider` and `acpserver`.
4. **Hand-off, the owner's to send:** a note for magic-cli-remote's 0243 PLAN naming acpext `v0.3.0` and the first gobble
   release that carries P3, and a status comment for #14. Neither is posted without the owner's word.
5. **Gates** (C6). Staged; the owner commits.

## Verification

The MADR's Confirmation, as the phases map it:

| Confirmation | Where |
| :--- | :--- |
| 1 acpext's encoding, frames and apidiff | P1 steps 3 to 5 |
| 2 the free-tier error's data | P3 step 5 (`TestPromptQuotaData`), P6 step 2.1 |
| 3 the limit table, and no window from a wait | P2 step 7 (`TestWireLimits`, `TestWireLongWaitIsNotAWindow`) |
| 4 the retry notice, the over-cap error, attempts | P2 step 7, P3 step 5, P6 steps 2.3 and 2.4 |
| 5 the 401 quota's wording | P2 steps 7 and 8, P6 step 2.2 |
| 6 archtest unchanged | P2 step 9, P3 step 8 |
| 7 seen failing | each phase's "Seen failing" step |
| 8 Q1 to Q3 as answered | P3 step 4b, P4 |
| 9 the gates on three hosts | each phase's gates (C6) |
| 10 the SDK's fields | P5 |

## Rollout and Rollback

* **Rollout.** acpext `v0.3.0` first (P1, the owner's tag), then a gobble release carrying P3 and P4, which the owner
  tags. magic-cli-remote reads the members once it requires acpext `v0.3.0` and the user runs that gobble release;
  until then it keeps reading `error`, which is unchanged. P5 lands with a later gobble release after the SDK's tag.
* **Rollback.**
  * acpext `v0.3.0` stays published, since a tag is never moved or deleted once served (0042-MADR D5; 0043-MADR); a
    client that does not want the members ignores them.
  * Reverting P3's commit returns the wire to `error` alone, with acpext `v0.3.0` still required, which is harmless;
    reverting P2's commit as well removes the facade's fields and the 401 fix. P4's parts revert on their own.
  * A gobble release that sent a wrong member is fixed by the next release. Until then a client that finds a member
    absent or a value it does not know falls back to `error`; magic-cli-remote's 0243-MADR D3 falls back to its own
    code table and the text.
