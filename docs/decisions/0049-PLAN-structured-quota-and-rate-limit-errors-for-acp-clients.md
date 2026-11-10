---
status: in-progress
date: 2026-11-10
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
phase without a tag, when 1 to 9 are met and the record says why 10 was not. *Amended 2026-11-10:* and when the
amendment's Confirmation 11 and 12 are met (P7, P8).

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
  *Amended 2026-11-10 (P6 deviation 2):* and `acpclient`'s row, and `docs/guides/choosing-a-model.md`'s sentence on a
  turn's mark.
* *Added 2026-11-10, by the MADR's amendment, accepted the same day with Q6 (a) and Q7 (a); each phase runs on the owner's word:*
  * **P7, `GOBBLE_HOME` needs no platform directory** (MADR D11, Q7): `internal/appdirs/dirs.go` and its tests.
  * **P8, `gobble acp`'s model flags and misplaced chat flags** (MADR D12, Q6): `internal/cli/flags.go`,
    `internal/cli/acp.go`, `internal/cli/modelflags.go`, `internal/cli/main.go`, their tests, and the docs naming
    `gobble acp`'s flags.

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
  `127.0.0.1:9`. *Amended 2026-11-09 (P2 deviation; corrected at P3):* `llm` and `llm/provider` run without the HTTP
  proxies, with `GOPROXY=off`, because `llm/provider`'s `TestWireOffline` checks the dial `Probe` makes with no proxy
  named. Every other package keeps them. (P2's wording said the proxies stay only for packages under a `TestMain`
  network guard, naming `acpserver` and `internal/cli`; only `internal/cli`'s `TestMain` is that guard, from 0046, and
  `acpserver`'s serves the MCP fixture.)
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

### P1 record (2026-11-09)

C1: `HEAD` `a6636fa` (P0), `origin/main` `74d7f66`, the tree clean; acpext `v0.2.0` and go-llmprovider-sdk `v1.28.0`
required. No deviation.

1. **`acpext/types.go`:** `ErrorKind` with its fourteen constants and `LimitScope` with its eight, each doc saying the
   set is open; `ErrorData`'s fourteen members in sorted order, its doc naming what a failed turn carries;
   `RetryNotice`'s `retryAfterMs`, `serverDirected`, `code` and `httpStatus` after `error`, its doc telling `delayMs`
   from `retryAfterMs`.
2. **`acpext/doc.go`:** a paragraph after the refused-request one: a failed turn's -32000 or -32603, its members, and the
   wait over the cap.
3. **Tests:** `TestErrorDataSortedMembers` with all fourteen members; `TestKindsAndLimits` (the names, and every retry
   cause a kind of the same name); `ExampleErrorData_limit`; four frames in `frames`, `prompt-quota.error.json`,
   `prompt-rate-over-cap.error.json`, `prompt-after-retries.error.json` and `retry.start.provider.notice.json`, which
   `TestFramesRoundTrip` and `TestFramesCoverEveryMethod` take. `go -C acpext test` ok.
4. **Seen failing** (C2), `p1_0049_breaks.py`, a scratch copy of the module per case, offline (`GOPROXY=off`); the
   control passed, and `cases not as expected: 0`:
   * HEAD's `types.go` with P1's tests: does not compile, `./example_test.go:55:16: e.Kind undefined (type
     acpext.ErrorData has no field or method Kind)`, `undefined: acpext.KindQuota`, `undefined: acpext.LimitFreeTier`.
   * `retryAfterMs` misspelt `retryAfterMS` in `ErrorData`: `TestFramesRoundTrip`, `roundtrip_test.go:136:
     prompt-quota.error.json, encoding/json/v2: decode: … unknown object member name "retryAfterMs"`, and under
     `encoding/json`, which matches names without case, `:145: … encodes back as`; `TestErrorDataSortedMembers`,
     `:212: ErrorData encodes as {… "retryAfterMS":5601000 …}`.
   * `Retryable` a `bool` with `omitzero`, the tests changed only to compile: `TestFramesRoundTrip`, `:145:
     prompt-quota.error.json, encoding/json: encodes back as` (and v2), its `"retryable":false` dropped;
     `TestErrorDataSortedMembers`, the encoding without `retryable`.
   * `Limit` declared before `Kind`: `TestErrorDataSortedMembers`, `:212: ErrorData encodes as {…
     "limit":"free_tier","kind":"quota" …}`.
   * `LimitFreeTier` spelt `free-tier`: `TestKindsAndLimits`, `:235: limit 6 is "free-tier", want "free_tier"`;
     `TestErrorDataSortedMembers`; `ExampleErrorData_limit`.
5. **apidiff**, the root's `golang.org/x/exp/cmd/apidiff` built to the scratchpad, export data written in an extract of
   `acpext/v0.2.0` (`git archive`) and in the tree's `acpext/`: exit 0, "Compatible changes:" only: the nine
   `ErrorData` fields, the four `RetryNotice` fields, `ErrorKind`, `LimitScope` and their twenty-two constants, each
   "added". Nothing incompatible.
6. **Gates** (C6):
   * `make pre-add-check FILES="acpext/types.go acpext/doc.go acpext/roundtrip_test.go acpext/example_test.go"`:
     `4 file(s) clean (gofmt, golint, govulncheck)`.
   * `make check-acpext-deps`: `ok`. `make lint`: `0 issues`, acpext's included.
   * macOS: `make preflight` exit 0, `preflight passed`; `make race` 39 ok.
   * The Linux machine (amd64) and the Windows laptop (amd64, `TMP`/`TEMP` set), a scratch copy of 918 files each:
     acpext's tests ok; `go test -count=1 ./...` 38 ok; `-race` 38 ok.
   * `make check-records markdownlint` after this record: exit 0. The identifier scan of the diff (the local account
     name, the dev hosts' names, the org's mail domain, and real-machine home and temporary paths): nothing found.
7. **The owner's, not done here:** the commit and its push; then the annotated tag `acpext/v0.3.0` on that commit, in
   `acpext/v0.2.0`'s form, and its push. P3 starts when `go list -m github.com/maccavelli/gobble-cli/acpext@v0.3.0`
   succeeds through the proxy; P2 may run before, on the owner's word.

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

### P2 record (2026-11-09)

C1: `HEAD` `ece13c4` (P1), `origin/main` `ece13c4`, the tree clean; acpext `v0.2.0` and go-llmprovider-sdk `v1.28.0`
required. The tag `acpext/v0.3.0` not yet pushed; P2 started before it on the owner's word ("start P2 while the ci
runs"). The owner pushed it while P2 ran; it peels to `ece13c4`. P2 changes nothing in acpext and does not require it.

**Deviation (2026-11-09), C3 and the proxies.** Step 8's first run set `HTTPS_PROXY` and `HTTP_PROXY` to `127.0.0.1:9`,
as C3 said. Then `TestWireOffline` failed in every case, the unbroken control too: `offline_test.go:67: Probe dialed
["tcp 127.0.0.1:9"], want tcp gobble-offline.invalid:80 alone`. The test predates 0049. It checks the dial `Probe` makes
for a failed request, which goes to the proxy when the environment names one, and `llm/provider` has no `TestMain`
network guard. Its tests stay off the network by canned transports, a refused loopback port and an `.invalid` host.
`make test` and `make preflight` already run it with no proxy named. The owner chose to run `llm` and `llm/provider`
without the HTTP proxies, `GOPROXY=off` kept, and to narrow C3 (amended above) rather than add a guard to the package.
No file outside step 1 to 7's was added to the phase, and no MADR decision or fact changes.

1. **`llm`:** `APIError` gains `Provider`, `Code`, `Message`, `Retryable *bool`, `ResetAt`, `Limit` and `Attempts`
   after `Err`, each documented as zero when unknown and filled only by a provider adapter; `Error()` and `Unwrap()`
   are unchanged. New `llm/kind.go`: `Kind` in D3's order, and the eight `Limit*` constants. `ProviderRetry` gains
   `RetryAfter`, `ServerDirected`, `Code` and `Status`. `llm/doc.go`'s sentinel section gains a paragraph naming them.
2. **The mapping:** `mapErrorAt(id, err, now)`, with `mapError(err)` kept as `mapErrorAt("", err, time.Now())`. The
   stream path and the notice hook pass `a.id`. The SDK error's `Code`, `Message`, `Retryable()`, the limit and
   `ResetAt` (`now.Add(RetryAfter)` when positive) are copied beside `Status` and `RetryAfter`.
3. **`limitOf`,** beside `sentinels`, with D4's rows as four code sets (`freeTierCodes`, `monthlyCodes`,
   `creditsCodes`, `planCodes`), HTTP 402 as credits, `daily` for a Gemini quota on 429 (the service from the id, or
   the SDK's `Provider` before its `/`), `rate` for a rate limit that is not a quota. It never reads `RetryAfter`.
4. **Attempts:** `attempt` keeps the last notice's `Attempt` in `tried`; an error after one or more notices sets
   `Attempts` to `tried + 1`.
5. **The retry notice:** the hook yields `a.providerRetry(n)`, a new `adapter` method that fills the four fields from
   `n.ServerDirected` and the mapped `n.Err`. Placed after `type adapter`: preflight's `gocritic` refused it above the
   type (`typeDefFirst`), and it was moved before the gates below.
6. **`rejected`** returns `mapped` first when the SDK error is `ErrQuotaExhausted` (D8).
7. **Tests**, as built, where they differ from step 7's text:
   * `TestWireLimits` has twelve subtests, one per row. The credits row for OpenCode Zen is a 401 `CreditsError`. Both
     Gemini rows carry `Code` `RESOURCE_EXHAUSTED`. The SDK's code is the first of a detail's `reason`, a string `code`,
     `type` and `status` (`api_error.go:590-599` at `v1.28.0`). Gemini's `code` is the number 429 and these rows' details
     (`QuotaFailure`, `RetryInfo`) carry no `reason`, so `status` wins, the same for a per-day and a per-minute 429. So a client cannot read the period from Gemini's `code`; only `limit` tells them apart. MADR D4 is
     unaffected, since it reads Gemini's period from the SDK's kind, not the code.
   * `TestRetryNoticeCarriesServerWait`: `RetryAfter` 1 ms (step 7 said 2 ms), `MaxAttempts` 2, `MaxDelay` 1 s.
   * `TestAttemptsAfterRetries`: `MaxAttempts` 4, the SDK's name for step 7's "`MaxRetries` 3"; and a failure with no
     retry keeps `Attempts` 0.
   * The stored-login case is its own test, `TestQuotaIsNotARejectedLogin` in `limit_test.go`, not a case of
     `TestWireRejectedKey`; `provider_test.go` is unchanged.
   * `TestKind` and `TestKindAgreesWithFailureKind` as step 7; `TestWireLongWaitIsNotAWindow` gives `rate`, 5 h,
     retryable.
8. **Seen failing** (C2), `p2_0049_breaks.py`, a scratch copy of the tree per case, `go test ./llm/ ./llm/provider/`,
   `GOPROXY=off`, no HTTP proxy (the deviation above). The control passed; `cases not as expected: 0`:
   * the tree before P2 with only `TestWireRejectedKey`'s new case: `wire_test.go:437: 401 MonthlyLimitError with a
     source: opencode-zen rejected the API key from auth.json: llm: api status 401: llm: quota exhausted: llmprovider:
     quota exhausted: opencode-zen/messages HTTP 401 MonthlyLimitError: Monthly limit reached., want llm.ErrQuota that
     does not say the key was rejected`. MADR F9, observed.
   * `insufficient_quota`'s row removed: `TestWireLimits`, `limit_test.go:93: limit "" provider "openai" status 429
     code "insufficient_quota", want "credits" …`.
   * `RetryAfter > 4h` → `window` planted: `TestWireLongWaitIsNotAWindow`, `limit_test.go:126: err … (retry-after
     5h0m0s): rate limited …, want a retryable rate li[mit] …`.
   * `incomplete` before `empty_reply`: `TestKind`, `kind_test.go:38: an empty reply wrapping incomplete: Kind =
     "incomplete", want "empty_reply"`.
   * `ResetAt` left zero: `TestWireLimits`, four rows, e.g. `limit_test.go:106: ResetAt 0001-01-01 00:00:00 +0000 UTC,
     want the call's time plus 1h33m21s` and `:102: … want …, the instant the provider named`.
   * `Attempts` with no `+ 1`: `TestAttemptsAfterRetries`, `limit_test.go:171: err … openai failed after 4 attempts …`.
   * `RetryAfter` from `n.Wait`: `TestRetryNoticeCarriesServerWait`, `limit_test.go:160: retry {Attempt:1
     MaxAttempts:2 Wait:37.598071ms Kind:rate-limited RetryAfter:37.598071ms …}, want RetryAfter 1ms …`.
   * `rejected`'s guard removed: `TestQuotaIsNotARejectedLogin`, `limit_test.go:199: opencode-zen rejected the OAuth
     login in auth.json; sign in again …: … MonthlyLimitError: monthly, want llm.ErrQuota not led by the login's
     refusal`, and `TestWireRejectedKey` as in the first case.
9. **apidiff** of `llm` and `llm/provider`, `v0.5.0` (`git archive`) against the tree, with P1's build of
   `golang.org/x/exp/cmd/apidiff` (preflight's own apidiff is a no-op below `v1.0.0`): `-incompatible` empty for both.
   `llm`'s compatible changes are the seven `APIError` fields, the four `ProviderRetry` fields, `Kind` and the eight
   `Limit*` constants, each "added". `llm/provider`'s exported API is unchanged.
10. **Gates** (C6):
    * `make pre-add-check FILES="llm/errors.go llm/event.go llm/kind.go llm/kind_test.go llm/doc.go
      llm/provider/provider.go llm/provider/limit_test.go llm/provider/wire_test.go"`: `8 file(s) clean (gofmt,
      golint, govulncheck)`.
    * macOS: `make preflight`, first exit 2 on the `typeDefFirst` finding (step 5), then exit 0, `preflight passed`,
      `archtest` included and `internal/archtest` unchanged; `make race` exit 0, 39 ok.
    * The Linux machine and the Windows laptop (`TMP`/`TEMP` set), a scratch copy of 921 files each: acpext's tests ok;
      `go test -count=1 ./...` 38 ok; `-race` 38 ok; on each host, exit 0 and no `FAIL`.
    * `make check-records markdownlint` after this record: exit 0. The identifier scan of the diff and the three new
      files (the local account name, the dev hosts' names, the org's mail domain, and real-machine home and temporary
      paths): nothing found.

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
   * ~~`TestCompactQuotaData`: `/compact` and `_gobble/compact` failing on the same error carry the same data.~~
     *Amended 2026-11-09 (P3 deviation 1):* `TestCompactQuotaData`: `_gobble/compact` failing on the quota error answers
     -32603 with every member, the session's provider and `kind` `quota`; `/compact` failing on `llm.ErrAuth`
     (`compact.go:44`) answers -32000 with `provider` and `kind` `auth`; `/compact` failing on the quota stays the
     command's answer, with no JSON-RPC error, as before.
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

### P3 record (2026-11-09)

C1: `HEAD` `969b161` (P2), `origin/main` `ece13c4` (P2 not yet pushed), the tree clean; acpext `v0.2.0` and
go-llmprovider-sdk `v1.28.0` required. `go list -m github.com/maccavelli/gobble-cli/acpext@v0.3.0` through the proxy:
`github.com/maccavelli/gobble-cli/acpext v0.3.0`, exit 0. The gate is met.

**Deviation 1 (2026-11-09), `/compact` and the data.** Step 5's `TestCompactQuotaData` asserted that `/compact` and
`_gobble/compact` failing on the same error carry the same data. Read before any write: `_gobble/compact` answers every
failure through `promptError` (`acpserver/extensions.go:204`). `/compact` (`cmdCompact`) calls it only with no provider
for the session (`acpserver/compact.go:28`) or on `llm.ErrAuth` (`:44`). Any other failure, a quota included, is the
command's answer, a failed row or text, and the command returns `nil` (`:50-53`: "A command's failure is its answer,
not a JSON-RPC error … as Pi shows the failure and goes on"). The assertion was wrong when written. MADR D7 does not
decide that `/compact` should answer with an error, and its F1 is accurate. The owner chose to test what the code does
(step 5's bullet, amended above), not to change `/compact`'s answer. No file is added to the phase, and the MADR is
unchanged.

**Correction (2026-11-09), C3's amendment.** Writing the breaks, `acpserver/testmain_test.go` was read: its `TestMain`
serves the stdio MCP fixture and is no network guard. P2's amendment of C3 named it as one; only `internal/cli`'s is
(0046). C3's amendment is reworded to what was decided: `llm` and `llm/provider` without the HTTP proxies, every other
package with them. P3's breaks ran `acpserver` and `agent` with the proxies at `127.0.0.1:9`.

1. **The gate:** `go get github.com/maccavelli/gobble-cli/acpext@v0.3.0` (`upgraded … v0.2.0 => v0.3.0`) and `go mod
   tidy`: `go.mod` one line, `go.sum` its two. The module the proxy serves is the tree's `acpext/` (`diff -r` of the
   download against it: no difference).
2. **`agent`:** `Retry` gains `RetryAfter`, `ServerDirected`, `Code` and `Status` after `Err`, documented as a
   provider's retry's, on its start; `agent.go`'s `ProviderRetry` case copies them.
3. **`acpserver/prompt.go`:** the `agent.Retry` case writes `RetryAfterMs`, `ServerDirected`, `Code` and `HTTPStatus`.
   `promptError(provider, err)` is -32000 for `llm.ErrAuth` that is not `llm.ErrWorkspace`, -32603 otherwise, each with
   `errData(errorData(provider, err))`. `errorData` sets `error`, `provider` and `kind` always. For the workspace
   refusal it returns there, `error` the advice. Otherwise, from an `*llm.APIError`, it sets `httpStatus`, `code`,
   `providerMessage`, `retryAfterMs`, `resetAt` (UTC, to the second, RFC 3339), `retryable`, `attempts` when 2 or more,
   and `limit`.
4. **The five callers** pass `a.provOf(s)` (`agent.go:630`, `:751`, `extensions.go:204`) or `a.provOf(t.s)`
   (`compact.go:28`, `:44`). `acpserver/workspace_test.go`'s two calls of `promptError` take the provider `claude`:
   the arity change of step 3 reaches them, and the test is otherwise unchanged and passes.
5. **Tests**, `acpserver/errordata_test.go` (new) and additions to `retry_test.go` and `extensions_test.go`:
   * `TestErrorData`, nine subtests over `errorData` and `promptError`'s code: step 5's eight, and an `Attempts` of 1
     that sends no `attempts`. The quota's `ResetAt` is 08:05:00.4 at UTC-6, so the data's UTC and its second are both
     checked.
   * `TestPromptQuotaData`: a connected agent on `opencode-zen`, `openai` or `claude` whose `llmtest` script fails with
     each failure. The -32603's data, decoded refusing unknown members and re-encoded, equals the frame's canonical
     encoding: `prompt-quota.error.json`, `prompt-rate-over-cap.error.json` and `prompt-after-retries.error.json`.
   * `TestCompactQuotaData`, as amended (deviation 1). `_gobble/compact` gives the quota frame's data, its `error`
     `Summarization failed: ` before the turn's. `/compact` refused gives -32000 naming `opencode-zen` and `auth`.
     `/compact` on the quota ends its turn with no error, after the compaction's model call (three calls).
   * `TestRetryNoticeServerMembers`: a scripted `llm.ProviderRetry` gives a start equal to
     `retry.start.provider.notice.json` but for the session id, and an end.
   * `TestKindsAndLimitsMatchAcpext` as step 5.
6. **Seen failing** (C2), `p3_0049_breaks.py`, a scratch copy per case, `go test ./acpserver/ ./agent/`, the proxies
   at `127.0.0.1:9`, `GOPROXY=off`. The control passed; `cases not as expected: 0`. Each failing subtest is named:
   * the tree before P3 (`agent`, `acpserver`'s four files, `go.mod`, `go.sum` at `HEAD`) with P3's tests: does not
     compile, `errordata_test.go:158:4: unknown field Kind in struct literal of type acpext.ErrorData`, `undefined:
     acpext.KindQuota`;
   * `errorData` without the provider: `TestErrorData` (all nine), `TestPromptQuotaData` (all three),
     `TestCompactQuotaData/_gobble/compact` and `/compact refused`;
   * the failed turn's caller (`agent.go:751`) passing `""`: `TestPromptQuotaData`, its three frames, the data with no
     `provider`;
   * `ProviderMessage` set on the workspace refusal: `TestErrorData/workspace`, `got {… "provider":"claude",
     "providerMessage":"secret-ish"}`. The first run also expected `TestWorkspaceWords` to fail. That expectation was
     the harness's error: that test's error carries no `*llm.APIError`. The expectation was corrected and the whole
     set re-run;
   * `Retryable` dropped: `TestErrorData` (the four with a `retryable`), `TestPromptQuotaData` (all three),
     `TestCompactQuotaData/_gobble/compact`;
   * `ResetAt` in local time: `TestErrorData/quota`, `TestPromptQuotaData/prompt-quota.error.json`,
     `TestCompactQuotaData/_gobble/compact`, the `resetAt` at UTC-6;
   * `ServerDirected` unwritten: `TestRetryNoticeServerMembers`, `retry_test.go:213: retry start got {…
     "retryAfterMs":1000,"code":"rate_limit_exceeded",…}`, want `… "serverDirected":true …`;
   * `_gobble/compact`'s caller (`extensions.go:204`) passing `""`: `TestCompactQuotaData/_gobble/compact`;
   * beyond step 6's list, `agent`'s copy without `RetryAfter`: `TestRetryNoticeServerMembers`; and `attempts` sent for
     one attempt: `TestErrorData/one_attempt`.
7. **Goldens:** `git diff --quiet 74d7f66 -- acpserver/testdata acpclient/acptest/testdata internal/cli/testdata`
   exit 0, and nothing untracked there.
8. **apidiff** of `agent` and `acpserver`, `v0.5.0` against the tree, as P2's: `-incompatible` empty for both;
   `agent`'s compatible changes are `Retry`'s four fields, each "added"; `acpserver`'s exported API is unchanged.
9. **Gates** (C6):
   * `make pre-add-check FILES="agent/event.go agent/agent.go acpserver/prompt.go acpserver/agent.go
     acpserver/compact.go acpserver/extensions.go acpserver/errordata_test.go acpserver/retry_test.go
     acpserver/extensions_test.go acpserver/workspace_test.go"`: `10 file(s) clean (gofmt, golint, govulncheck)`.
   * macOS: `make preflight` exit 0, `preflight passed`, `archtest` ok with `internal/archtest` unchanged: `acpserver`
     imports neither `llm/provider` nor the SDK. `make race` exit 0, 39 ok.
   * The Linux machine and the Windows laptop (`TMP`/`TEMP` set), a scratch copy of 922 files each: acpext's tests ok;
     `go test -count=1 ./...` 38 ok; `-race` 38 ok; on each host, exit 0 and no `FAIL`.
   * `make check-records markdownlint` after this record: exit 0. The identifier scan of the diff and the new test file
     (the local account name, the dev hosts' names, the org's mail domain, and real-machine home and temporary paths):
     nothing found.
10. **Not done here:** `docs/architecture.md` still lists acpext `v0.2.0` among the direct requirements (`:188`). That
    row, with those for `llm`, `llm/provider` and `acpserver`, is P6's by this plan's scope, so it is left for P6.

### P4 — the owner's answers to Q2 and Q3

Each part runs only as answered; a part answered "not in this record" is recorded as not done, with the answer.

#### P4a — Q2 (a): the mark keeps its reset time

1. `mark` gains `resetAt time.Time`, from the failure's `*llm.APIError.ResetAt`.
2. ~~`StateOf`'s reason for `StateQuotaExhausted` and a rate-limited mark gains " — resets HH:MM" in the local clock,
   from the agent's `now`; nothing of the provider's text (0025-MADR D4).~~ *Amended 2026-11-09 (P4 deviation 1):* a
   quota mark's reason gains " — resets HH:MM" in the local clock, added in `markFailure` from the agent's `now`;
   nothing of the provider's text (0025-MADR D4). `StateOf` is unchanged, and a rate limit stays unmarked (MADR F11).
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

### P4 record (2026-11-09)

C1: `HEAD` `36626fd` (P3), `origin/main` `ece13c4` (P2 and P3 not yet pushed), the tree clean; acpext `v0.3.0` and
go-llmprovider-sdk `v1.28.0` required. An earlier "committed proceed" found P3 still staged with `HEAD` at `969b161`;
nothing was written until the owner's commit landed.

**Deviation 1 (2026-11-09), P4a and the rate-limited mark.** Step 2 put the reset time on "`StateQuotaExhausted` and
a rate-limited mark". Read before any write: `markFailure` (`acpserver/choice.go:269-284`) marks a provider only on
`llm.ErrAuth`, `llm.ErrQuota` or `llm.ErrNotPermitted`, so there is no rate-limited mark. MADR F11 says so, and Q2 (a)
describes the quota mark alone. Step 2 also put the text in `StateOf(id, err)`, which has no clock and names listing
failures too; `markFailure` makes the mark and has the agent's `now`. The owner chose option 1. A quota mark keeps
`ResetAt`, its reason gains " — resets HH:MM" in `markFailure`, and it lapses after that time; `StateOf` is unchanged,
and a rate limit stays unmarked. Step 2 is amended above. No file is added to the phase, and the MADR is unchanged.

**P4a** (Q2 (a)):

1. `mark` gains `resetAt`. `markFailure` sets it from the `*llm.APIError`'s `ResetAt` when the mark is
   `StateQuotaExhausted` and the provider gave one. The reason then gains "— resets HH:MM", the time on the agent's
   clock (`a.now().Location()`).
2. `choice()` drops every mark whose `resetAt` is set and not after the agent's `now`, before it applies the rest. A
   mark with no `resetAt` (credits, a refused key, not permitted) stays as before.
3. `TestQuotaMarkLapses` (`acpserver/choice_test.go`), `Options.Now` fixed at 07:00 UTC-6. The free-tier subtest uses
   P3's `quotaErr` (it lifts at 14:05:00.4 UTC). `/model` reads `**Fake** — quota exhausted (HTTP 429) — resets 08:05`,
   then `**Fake** — live` at 08:05:01. The credits subtest, a 402 with no reset time, reads `quota exhausted (HTTP
   402)` before and after, with nothing of the provider's text.

**P4b** (Q3 (a)):

1. New `acpclient/prompterror.go`: `PromptError{Code int; Data acpext.ErrorData}` over the `*acp.RequestError`
   (`Unwrap`). `Error()` is `Data.Error`, or the request error's text when that is empty. `Session.Prompt` turns a
   refused prompt into one when its data decodes as an `ErrorData` that is not zero. Unknown members are ignored, so a
   newer agent's data decodes; data that names no member, such as another agent's, leaves the error as before. Under
   auth_required it is still wrapped with `ErrAuthRequired`.
2. `acptest.Turn` gains `ErrData`, the error's data as JSON, sent as the `RequestError`'s data.
3. New `internal/cli/limit.go`: `limitWords`, the eight scopes in words, and `limitLine`, which builds "<words> on
   <provider>; resets at HH:MM, in <wait>". The time is on the session's clock. The reset part is left out with no
   time, and the wait once the time has passed. A limit not in the table is named as it is; "Limit" stands in when only
   a time is given. `line.go`'s failed-turn branch prints a `PromptError`'s text as the error, then the limit line, if
   any, as a note. Any other error prints as before.
4. `print.go`: `printResult` gains `ErrorData *acpext.ErrorData` (`errorData`, `omitzero`), after `error`. For a
   `PromptError`, `error` is its text and `errorData` its data. **Print mode's `error` text changes:** where it was
   `session/prompt: {"code":-32603,"message":…,"data":{…}}`, it is now the agent's message alone. The process's
   stderr line for a failed turn is now `Error: session/prompt: <message>`, not the JSON. An agent error with no
   gobble data is printed as before, and the auth path still exits 3 with gobble's sign-in message.
5. Tests: `TestPromptError` (`acpclient`) covers the quota frame's data, auth_required with data, no data, and another
   agent's data. `TestLimitLine` covers seven cases and words for all eight scopes. `TestLineSessionLimitError`: stderr
   has `Error: <the frame's error>` and `Free tier limit on opencode-zen; resets at 14:05, in 1h5m`, with no
   `session/prompt:` and no JSON, and the session goes on. `TestPrintErrorData`: the JSON result's `error` is the
   frame's message and its `errorData` the frame's data, decoded refusing unknown members. Today's
   `TestPrintErrorResult`, `TestPrintAuthRequired`, `TestLineSessionTurnError`, `TestLineSessionNoCredential` and
   `TestAuthRequired` pass unchanged.

**Seen failing** (C2), `p4_0049_breaks.py`, a scratch copy per case, `go test ./acpserver/ ./acpclient/...
./internal/cli/`, the proxies at `127.0.0.1:9`, `GOPROXY=off`. The control passed. Every line
`cleanup_test.go:82: the cleanup reported …` in the logs is the `t.Logf` of `TestCleanupWaitsForHandlers`, which
passes, the control's run included.

* The tree before P4a (`choice.go` at `HEAD`) with its test: `TestQuotaMarkLapses/free_tier`, `choice_test.go:329:
  /model before the reset, want "**Fake** — quota exhausted (HTTP 429) — resets 08:05"`.
* The tree before P4 with P4's tests: does not compile, `acpclient/prompterror_test.go:33:51: unknown field ErrData in
  struct literal of type …acptest".Turn`, `:56:37: undefined: acpclient.PromptError`.
* P4a, the lapse not applied (the `delete` planted out): `TestQuotaMarkLapses/free_tier`, `choice_test.go:333:
  /model after the reset, want "**Fake** — live"`.
* P4a, the time from `ResetAt`'s zero value, and the time in UTC rather than the agent's clock: each
  `TestQuotaMarkLapses/free_tier`, at `:329`.
* P4a, a mark with no reset time lapses (`resetAt.IsZero()` dropped from the test): `TestQuotaMarkLapses/credits`,
  `TestRefusedKeyMarksProvider` and `TestSignInLiftsMark` (`choice_test.go:269: /model after a refused key`). Every
  mark with no reset time went.
* P4b, the data not decoded: `TestPromptError` (`prompterror_test.go:59: quota: err "session/prompt:
  {\"code\":-32603,…`), `TestLineSessionLimitError`, `TestPrintErrorData`.
* P4b, the limit line missing: `TestLineSessionLimitError`. `errorData` absent: `TestPrintErrorData`.
* Beyond the plan's list: another agent's data taken as gobble's, `TestPromptError` (`:74: another agent's data`);
  whole hours as `2h0m`, `TestLimitLine/no_limit`; an unknown limit unnamed, `TestLimitLine/a_limit_not_in_the_table`.
* The first run had two cases not as expected, both the harness's errors. Planting out the whole lapse loop left `now`
  unused, so that copy did not compile; the break now replaces only the `delete`. The no-reset-time break expected
  `TestQuotaMarkLapses` alone, but it also lifts the refused-key marks, which two older tests hold. Both cases were
  corrected and re-run with the control: `cases not as expected: 0`.

**apidiff** of `acpclient`, `acpclient/acptest` and `acpserver`, `v0.5.0` against the tree, as P2's: `-incompatible`
empty for each. The compatible changes are `PromptError` and `Turn.ErrData`, each "added", and `acpserver`'s exported
API is unchanged.

**Gates** (C6):

* `make pre-add-check FILES="acpserver/choice.go acpserver/choice_test.go acpclient/prompterror.go
  acpclient/prompterror_test.go acpclient/conn.go acpclient/acptest/script.go internal/cli/limit.go
  internal/cli/limit_test.go internal/cli/line.go internal/cli/print.go"`: `10 file(s) clean (gofmt, golint,
  govulncheck)`.
* macOS: `make preflight` exit 0, `preflight passed`, `archtest` ok with `internal/archtest` unchanged; `make race`
  exit 0, 39 ok.
* The Linux machine and the Windows laptop (`TMP`/`TEMP` set), a scratch copy of 926 files each: acpext's tests ok;
  `go test -count=1 ./...` 38 ok; `-race` 38 ok; on each host, exit 0 and no `FAIL`.
* `make check-records markdownlint` after this record: exit 0. The identifier scan of the diff and the four new files
  (the local account name, the dev hosts' names, the org's mail domain, and real-machine home and temporary paths):
  nothing found.

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
3. **The records:** the MADR's Confirmation item by item in a "Completion" section with commits and evidence; ~~this
   plan `complete` with a P-record per phase~~ *Amended 2026-11-10 (P6 deviation 1):* this plan stays `in-progress`
   with P6's record, and the Completion section marks Confirmation 10 open while P5 waits on the SDK's tag; P5's record
   fills item 10 and sets the plan `complete`; `docs/README.md`'s rows; `docs/architecture.md`'s rows for `acpext`,
   `llm`, `llm/provider` and `acpserver`.
4. **Hand-off, the owner's to send:** a note for magic-cli-remote's 0243 PLAN naming acpext `v0.3.0` and the first gobble
   release that carries P3, and a status comment for #14. Neither is posted without the owner's word.
5. **Gates** (C6). Staged; the owner commits.

### P6 record (2026-11-10)

C1: `HEAD` `9eb22bc` (P4), `origin/main` `ece13c4` (P2 to P4 not yet pushed), the tree clean; acpext `v0.3.0` and
go-llmprovider-sdk `v1.28.0` required. P5 has not run: the owner has not named the SDK's tag.

**Deviation 1 (2026-11-10), the plan's status at P6.** Step 3 set this plan `complete` with a P-record per phase. The
plan put P5 before P6 on the assumption that the SDK's tag would come first. It has not, so P5 has no record and
Confirmation 10 is not met. The Goal allows completion without P5 only if the owner closes Q4's phase without a tag,
and the owner has not. The owner chose option 1: P6 closes with its record, the architecture and README rows, and the
MADR's Completion section for Confirmation 1 to 9, with item 10 marked open; this plan stays `in-progress`; P5's record
fills item 10 and sets it `complete`. Step 3 is amended above. No file is added to the phase. The MADR gains only the
Completion section step 3 named; no decision or fact changes.

**Deviation 2 (2026-11-10), two lines outside step 3's list.** Writing the rows, two lines were read that P4 left
behind. `docs/guides/choosing-a-model.md:161-162` said a turn's mark lasts "for the rest of the session", which P4a
made untrue for a quota with a reset time. `docs/architecture.md`'s `acpclient` row did not name `PromptError`.
The owner chose option (a): both are fixed in P6. The guide names the reset and the sign-in that lift a mark, and the
`acpclient` row names `PromptError`, the limit line and `errorData`. Scope is amended above. No decision or fact in the
MADR changes.

**The two findings moved into scope (2026-11-10).** The owner moved F15 and F16 (step 2's driver notes below) into this
record: "add these to the current scope, they need to be analyzed and remediated". They are analysed in the MADR's
amendment (F15, F16, D11, D12, Q6, Q7) and planned as P7 and P8, proposed. Neither changes P6.

1. **Goldens:** `git diff --quiet 74d7f66 -- acpserver/testdata acpclient/acptest/testdata internal/cli/testdata`
   exit 0, and nothing untracked there.
2. **Acceptance** (C5), `p6_0049_accept.py`, a scratchpad driver that is not committed:
   * **The binary** is built from the tree at `9eb22bc`. **The server** is a Python `ThreadingHTTPServer` on
     `127.0.0.1` that answers each case's scripted replies by path, the last one repeating, and records each
     request's path and time. A `GET`, the model listing, gets 404.
   * **Each case** has its own scratch home and working directory: `GOBBLE_HOME`, `HOME`, `USERPROFILE` and `XDG_*`
     inside it, and the environment cleared but for `PATH` and the system's variables. `HTTPS_PROXY` and `HTTP_PROXY`
     are at `127.0.0.1:9`. `models.json` points `opencode-zen`'s `baseUrl` at `<BASE>/zen/v1` (`api`
     `openai-completions`) and `openai`'s at `<BASE>/openai/v1`, each with the model `accept-model`, and the keys are
     dummy values.
   * **The ACP client** speaks to `gobble acp` over stdio: `initialize`, `session/new`, `session/set_config_option`
     setting the model (checked), then `session/prompt` with `hi`. It refuses every agent request with -32601 and
     records each frame.
   * Thirty-eight checks; the driver exits 1 on any failure.

   | Host | Checks | Failed | Exit |
   | :--- | :--- | :--- | :--- |
   | macOS (arm64) | 38 | 0 | 0 |
   | The Linux machine (amd64) | 38 | 0 | 0 |
   | The Windows laptop (amd64) | 38 | 0 | 0, on the second run (below) |

   The frames, from the macOS run, with the session id as `<SESSION>`:
   1. **Case 1,** a 429 `FreeUsageLimitError` with `Retry-After: 5601`: one request and no `_gobble/retry`. The
      answer is the frame below, with `resetAt` within a second of the request's time plus 5601 s:

      ```json
      {"jsonrpc":"2.0","id":4,"error":{"code":-32603,"message":"Internal error","data":{"code":"FreeUsageLimitError","error":"llm: api status 429: llm: quota exhausted: llmprovider: quota exhausted: opencode-zen/chat_completions HTTP 429 FreeUsageLimitError (retry-after 1h33m21s): Free usage limit reached. Free models work in OpenCode (https://opencode.ai); subscribe to OpenCode Go to keep going: https://opencode.ai/go","httpStatus":429,"kind":"quota","limit":"free_tier","provider":"opencode-zen","providerMessage":"Free usage limit reached. Free models work in OpenCode (https://opencode.ai); subscribe to OpenCode Go to keep going: https://opencode.ai/go","resetAt":"2026-11-10T06:45:41Z","retryAfterMs":5601000,"retryable":false}}}
      ```

   2. **Case 2,** a 401 `MonthlyLimitError`, the key stored by `gobble auth login opencode-zen --api-key -`: -32603,
      not auth_required, and no word of a rejected key:

      ```json
      {"jsonrpc":"2.0","id":4,"error":{"code":-32603,"message":"Internal error","data":{"code":"MonthlyLimitError","error":"llm: api status 401: llm: quota exhausted: llmprovider: quota exhausted: opencode-zen/chat_completions HTTP 401 MonthlyLimitError: Monthly limit reached.","httpStatus":401,"kind":"quota","limit":"monthly","provider":"opencode-zen","providerMessage":"Monthly limit reached.","retryable":false}}}
      ```

   3. **Case 3,** `openai`: a 429 with `Retry-After: 1` once, then a reply. Two requests:

      ```json
      {"jsonrpc":"2.0","method":"_gobble/retry","params":{"sessionId":"<SESSION>","phase":"start","cause":"rate-limited","attempt":1,"maxAttempts":4,"delayMs":1063,"retryAfterMs":1000,"serverDirected":true,"code":"rate_limit_exceeded","httpStatus":429}}
      {"jsonrpc":"2.0","method":"_gobble/retry","params":{"sessionId":"<SESSION>","phase":"end","cause":"rate-limited"}}
      {"jsonrpc":"2.0","id":4,"result":{"stopReason":"end_turn","usage":{"inputTokens":12,"outputTokens":3,"totalTokens":15}}}
      ```

   4. **Case 4,** `openai`: a 429 with `Retry-After: 120`, over the 60 s cap. One request, no `_gobble/retry`, and
      `resetAt` the request's time plus 120 s:

      ```json
      {"jsonrpc":"2.0","id":4,"error":{"code":-32603,"message":"Internal error","data":{"code":"rate_limit_exceeded","error":"llm: api status 429: llm: rate limited: llmprovider: rate limited: openai HTTP 429 rate_limit_exceeded (retry-after 2m0s): Rate limit reached for requests","httpStatus":429,"kind":"rate-limited","limit":"rate","provider":"openai","providerMessage":"Rate limit reached for requests","resetAt":"2026-11-10T05:14:22Z","retryAfterMs":120000,"retryable":true}}}
      ```

   5. **Case 5,** `gobble --model opencode-zen/accept-model -p hi --output-format json` against case 1's reply: exit 1.
      Its stdout follows; `errorData` is case 1's data, and `resetAt` is within a second of its own request plus
      5601 s:

      ```json
      {"text":"","stopReason":"error","usage":null,"cost":{"amount":0,"currency":"USD"},"sessionId":"<SESSION>","error":"llm: api status 429: llm: quota exhausted: llmprovider: quota exhausted: opencode-zen/chat_completions HTTP 429 FreeUsageLimitError (retry-after 1h33m21s): Free usage limit reached. Free models work in OpenCode (https://opencode.ai); subscribe to OpenCode Go to keep going: https://opencode.ai/go","errorData":{"code":"FreeUsageLimitError","error":"llm: api status 429: llm: quota exhausted: llmprovider: quota exhausted: opencode-zen/chat_completions HTTP 429 FreeUsageLimitError (retry-after 1h33m21s): Free usage limit reached. Free models work in OpenCode (https://opencode.ai); subscribe to OpenCode Go to keep going: https://opencode.ai/go","httpStatus":429,"kind":"quota","limit":"free_tier","provider":"opencode-zen","providerMessage":"Free usage limit reached. Free models work in OpenCode (https://opencode.ai); subscribe to OpenCode Go to keep going: https://opencode.ai/go","resetAt":"2026-11-10T06:45:43Z","retryAfterMs":5601000,"retryable":false}}
      ```

      Its stderr is `Error: session/prompt: llm: api status 429: …`, the agent's message after the prefix. Before it
      is a `WARN` line, `model listing failed; offering the curated static list`, from the server's 404 to the
      listing.

   **How the driver came to its form.** Each change below was made to the driver, not to gobble, and before any run
   was counted:
   * Step 2 named three ACP calls. The first run passed `--model` to `gobble acp`, which Kong refused as an unknown
     flag. The root's `--model` (`gobble --model opencode-zen/accept-model acp`) does not set an ACP session's model:
     the session started on `opencode-zen`'s default, `claude-haiku-5-5`, and its request went to `/messages`. The
     driver now sets the model as an ACP client does, with `session/set_config_option` (`configId` `model`), and
     checks it.
   * The Windows laptop's first run failed every case. gobble reads the `RoamingAppData` and `LocalAppData` known
     folders before `GOBBLE_HOME` (`internal/appdirs/base_windows.go`; `dirs.go:80-86`). Those folders follow
     `USERPROFILE`, and the scratch home had none, so each run failed with `appdirs: known folder RoamingAppData: The
     system cannot find the path specified.` The scratch `models.json` and `auth.json` were not read, and gobble
     dialled the real hosts (`opencode.ai/zen/v1/…`, `api.openai.com/v1/responses`). The proxy at `127.0.0.1:9` refused
     every dial (`proxyconnect tcp: dial tcp 127.0.0.1:9: … No connection could be made`), so no request left the
     laptop. The driver now gives the Windows scratch home `AppData\Roaming` and `AppData\Local`, and the second run
     passed 38 of 38.
   * Both are gobble's behaviour before 0049. Neither is changed in P6. The owner moved both into scope, as the MADR's
     F15 and F16, and as P7 and P8.
3. **Seen failing** (C2): the same driver against a binary built from `git archive 74d7f66`, the tree before 0049, in
   the scratchpad, on macOS. `checks: 38, failed: 20`. The 18 that passed check what 0049 did not change: the setup, the
   JSON-RPC codes, the requests and notices counted, the retry's `cause`, `error`'s text, and case 5's exit status.
   Every new member was missing:
   * case 1: `provider`, `kind`, `httpStatus`, `code`, `providerMessage`, `retryAfterMs`, `retryable`, `limit` and
     `resetAt` each failed, with the data `{"error":"llm: api status 429: …"}` alone;
   * case 2: `kind quota, limit monthly` and `error does not say the key was rejected`, with `error` `opencode-zen
     rejected the API key from auth.json: llm: api status 401: llm: quota exhausted: … MonthlyLimitError: Monthly limit
     reached.` (MADR F9);
   * case 3: the start's `retryAfterMs`, `serverDirected`, `code` and `httpStatus`;
   * case 4: `retryable true, retryAfterMs 120000, limit rate` and its `resetAt`;
   * case 5: `errorData` and its `resetAt` absent, and `error` `session/prompt: {"code":-32603,"message":"Internal
     error","data":{"error":"llm: api status 429: …"}}`.
4. **The records:**
   * The MADR's "Completion" section, with Confirmation 1 to 9 met and 10 open.
   * `docs/architecture.md`: acpext's paragraph, its requirement row at `v0.3.0`, and the rows for `llm`,
     `llm/provider`, `acpserver` and, by deviation 2, `acpclient`.
   * `docs/guides/choosing-a-model.md`'s mark sentence, by deviation 2.
   * `docs/README.md`'s two 0049 rows.
5. **Hand-off:** the note for magic-cli-remote's 0243 PLAN and the status comment for #14 are drafted in the hand-off
   to the owner and are not posted. No gobble release carries P3 yet: `v0.5.0` predates it. P2 to P4 are not yet
   pushed, so the comment cannot link those commits until they are.
6. **Gates** (C6). No Go file changed in P6, so there was no pre-add check. C6 runs the remote tests at P1 to P5, not
   at P6; P6's acceptance ran on both remote hosts.
   * macOS: `make preflight` exit 0, `preflight passed`, `check-records` and `markdownlint` included; `make race` exit
     0, 39 ok.
   * After deviation 2 and the amendment, which change docs alone, `make check-records markdownlint` exit 0.
   * The identifier scan of the diff found nothing. It looked for the local account name, the dev hosts' names, the
     org's mail domain, and real-machine home and temporary paths.

### P7 — `GOBBLE_HOME` needs no platform directory (MADR D11, Q7)

Q7 was answered (a), 2026-11-10: the steps run as written. Runs on the owner's word.

1. **C1** as every phase, with P6 committed.
2. **`internal/appdirs/dirs.go`:**
   * `System` reads the base through a package variable, `readBase = systemBase`, so that a test can make it fail.
   * ~~It calls `readBase` only when a role needs the platform default: `GOBBLE_HOME` is unset, and so is at least one
     of the four `GOBBLE_<ROLE>_DIR` variables. Otherwise it passes `Resolve` a zero `Base`.~~
   * ~~`Resolve`, `platform`, both `systemBase` files and the 0003-MADR table are unchanged. The doc comment on `System`
     says when the base is read.~~
   * *Amended 2026-11-10 (P7 deviation 1):* `Resolve` takes the base as a function, `base func() (Base, error)`, and
     calls it only when a role falls to the platform default: `GOBBLE_HOME` is unset, and so is at least one of the
     four `GOBBLE_<ROLE>_DIR` variables. Its error is then `Resolve`'s, unchanged. `System` passes `readBase`.
     `platform`, both `systemBase` files and the 0003-MADR table are unchanged. The doc comments on `Resolve` and
     `System` say when the base is read.
3. **Tests** (`internal/appdirs/dirs_test.go`):
   * `TestSystemGobbleHomeNeedsNoBase`: with `readBase` failing and `GOBBLE_HOME` set, `System` returns that
     directory for all four roles. With all four role variables set and no `GOBBLE_HOME`, it returns them.
   * `TestSystemBaseError`: with `readBase` failing and neither set, `System` returns the base's error unchanged.
   * `TestSystem` passes unchanged.
   * *Amended 2026-11-10 (P7 deviation 1):* `TestResolveReadsBaseOnlyWhenNeeded`: a counting base function is called
     zero times under `GOBBLE_HOME` and under all four role variables, and once with three of them set, or none.
     With three set, its error is `Resolve`'s. Today's `Resolve` cases pass their `Base` through a test helper and
     are otherwise unchanged.
4. **Seen failing** (C2), on a scratch copy:
   * today's `System` (the base read first) with the new tests: `TestSystemGobbleHomeNeedsNoBase` fails on both
     cases;
   * the condition planted as "`GOBBLE_HOME` unset" alone: the four-role case fails;
   * the base's error dropped: `TestSystemBaseError` fails;
   * *amended 2026-11-10 (P7 deviation 1):* the base read on every call: `TestResolveReadsBaseOnlyWhenNeeded` fails.
5. **Acceptance:** P6's driver without its Windows AppData folders, on the Windows laptop, with the binary built from
   the tree: 38 of 38. On macOS, `gobble config path` with `GOBBLE_HOME` set and `HOME` unset exits 0 and prints the
   directory; at `9eb22bc` it exited 1 (MADR F16).
6. **Gates** (C6), with the Linux machine and the Windows laptop running `go test ./...` and the race tests. Staged;
   the owner commits.

### P7 record (2026-11-10)

C1: `HEAD` `2db8e83` (the amendment's answers), `origin/main` `ece13c4` (P2 onwards not yet pushed), the tree clean;
acpext `v0.3.0` and go-llmprovider-sdk `v1.28.0` required.

**Deviation 1 (2026-11-10), `Resolve` reads the base for every role.** Step 2 was written as planned, and
`TestSystemGobbleHomeNeedsNoBase`'s second case failed: `dirs_test.go:194: System with the four role variables and no
base = {Config: Data: State: Cache:}, appdirs: home directory "" is not an absolute path`. With `GOBBLE_HOME` unset,
`Resolve` (`internal/appdirs/dirs.go:48-60`) builds every platform default before it applies the four role variables.
So all four set still need the base, and step 2's "`Resolve` … unchanged" cannot meet D11's "or all four role
variables". This predates 0049: at `9eb22bc`, `Resolve` fails the same way. The owner was offered three fixes:
`Resolve` skipping the defaults when all four are set; D11 narrowed to `GOBBLE_HOME`; or `Resolve` taking the base as a
function it calls only when a role needs it. The owner chose the third: "option 3 i want the most robust and
comprehensive fix". Steps 2 to 4 are amended above, and the MADR's D11 is amended: `Resolve`'s signature changes. No
file is added to the phase. `Resolve` is called only inside `internal/appdirs`.

1. **C1** as above.
2. **`internal/appdirs/dirs.go`:**
   * `Resolve(env, goos, base func() (Base, error))` resolves `GOBBLE_HOME` first. With it unset, it calls `base` only
     when a role has no variable of its own (`needsPlatform`), returns `base`'s error unchanged, and builds the
     platform defaults from what it gives. The role variables are then applied as before.
   * `System` is `Resolve(os.Getenv, runtime.GOOS, readBase)`, where `readBase = systemBase`.
   * `platform`, both `systemBase` files and the 0003-MADR table are unchanged.
   * One effect beyond the defect: with all four role variables set, a relative `XDG_*` value no longer produces a
     diagnostic. It named a default that no role used.
3. **Tests** (`internal/appdirs/dirs_test.go`):
   * `TestSystemGobbleHomeNeedsNoBase`, `TestSystemBaseError` and `TestResolveReadsBaseOnlyWhenNeeded` (four subtests),
     as amended. Each uses `failBase` or a counting base function that fails with `errBase`.
   * Today's `Resolve` cases pass their `Base` through `fixed`; otherwise they are unchanged. `TestSystem` is
     unchanged. All pass.
4. **Seen failing** (C2), `p7_0049_breaks.py`, a scratch copy per case, `go test ./internal/appdirs/`. The control
   passed; `cases not as expected: 0`:
   * the tree before P7 (`dirs.go` at `HEAD`) with P7's tests: does not compile, `dirs_test.go:74:55: cannot use
     fixed(base) (value of type func() (Base, error)) as Base value in argument to Resolve`, `undefined: readBase`;
   * `System` reading the base first, as at `HEAD`, through the new `Resolve`: `TestSystemGobbleHomeNeedsNoBase`,
     `dirs_test.go:181: System with GOBBLE_HOME and no base = {Config: Data: State: Cache:}, appdirs: known folder
     RoamingAppData: the base is unreadable`. Its four-role case is not reached, since the test stops at the first
     failure. The next break covers that case;
   * the condition planted as "`GOBBLE_HOME` unset" alone: `TestSystemGobbleHomeNeedsNoBase`, `:194: System with the
     four role variables and no base = {…}, appdirs: known folder RoamingAppData: the base is unreadable`; and
     `TestResolveReadsBaseOnlyWhenNeeded/all_four_role_variables`, `:240: base read 1 times, want 0`;
   * the base's error dropped: `TestSystemBaseError`, `:210: System with an unreadable base = appdirs: home directory
     "" is not an absolute path, want … the base is unreadable`; and `TestResolveReadsBaseOnlyWhenNeeded`'s `none` and
     `three role variables`, `:243: Resolve err = appdirs: RoamingAppData "" is not an absolute path`;
   * the base read on every call: all four `TestResolveReadsBaseOnlyWhenNeeded` subtests, `:240: base read 1 times,
     want 0` and `base read 2 times, want 1`.
5. **Acceptance**, with the binary built from the tree:
   * **The Windows laptop:** P6's driver with `--no-appdata`, a new option that gives the scratch home no AppData
     folders. `checks: 38, failed: 0`, `accept exit 0`. Afterwards no case's home has an `AppData` folder: gobble
     created none.
   * **macOS:** `gobble config path` in an empty environment with `GOBBLE_HOME` set exits 0 and prints that directory
     for all four roles; at `9eb22bc` it exited 1 (MADR F16). With only the four `GOBBLE_<ROLE>_DIR` variables set, it
     exits 0 and prints each. With neither set, it exits 1 with `appdirs: home directory: $HOME is not defined`, as
     before.
6. **Gates** (C6):
   * `make pre-add-check FILES="internal/appdirs/dirs.go internal/appdirs/dirs_test.go"`: `2 file(s) clean (gofmt,
     golint, govulncheck)`.
   * macOS: `make preflight` exit 0, `preflight passed`; `make race` exit 0, 39 ok.
   * The Windows laptop (`TMP`/`TEMP` set), a scratch copy of 926 files: acpext's tests ok; `go test -count=1 ./...`
     38 ok; `-race` 38 ok; exit 0 and no `FAIL`.
   * The Linux machine, a scratch copy of 926 files: acpext's tests ok; `go test -count=1 ./...` 38 ok; `-race` 38
     ok; exit 0 and no `FAIL`.
   * `docs/architecture.md`'s `internal/appdirs` row and the developing guide's `GOBBLE_HOME` line stay accurate, so
     neither changes.
   * `make check-records markdownlint` after this record: exit 0. The identifier scan of the diff (the local account
     name, the dev hosts' names, the org's mail domain, and real-machine home and temporary paths): nothing found.

### P8 — `gobble acp`'s model flags and misplaced chat flags (MADR D12, Q6)

Q6 was answered (a), 2026-11-10: the steps run as written. Runs on the owner's word, after P7.

1. **C1**, and a probe on a scratch copy before any write. How Kong records a chat flag given before another
   command: its `kong.Context.Path` element, and whether `kctx.Command()` and the flag's owner tell them apart. If
   Kong gives no way to tell a misplaced flag from chat's own, that is a deviation: stop and prompt.
2. **`gobble acp`'s flags:**
   * `--provider`, `--model` and `--thinking` move from `SharedFlags` into a new embedded `ModelFlags` struct
     (`internal/cli/flags.go`). `SharedFlags` embeds it, so chat's flags, help and completion are unchanged.
   * `ACPCmd` embeds `ModelFlags`. Its `Run` resolves them with `pickModel` before `agentOptions`, so `e.pick` reaches
     `acpserver.Override` as chat's does. The checks chat makes on these flags apply here too, such as "--provider
     requires --model".
   * `pickModel` takes `*ModelFlags`; its callers and tests pass chat's.
   * *2b, added 2026-11-10 (P8 deviation 1):* with `--login`, each of `--mode`, `--provider`, `--model` and
     `--thinking` is a usage error, exit 2, naming the flag: it serves sessions, and `--login` serves none.
     `TestACPLogin` gains the cases, and a break that drops the check is seen failing.
3. **Misplaced flags:** in `run` (`internal/cli/main.go`), after parsing, a flag of chat's that was given while the
   command is not chat is a usage error, exit 2: "`--model` is a flag of gobble's chat command; put it after
   `gobble acp`" for the three model flags under acp, and "`--print` is a flag of gobble's chat command, not of
   `gobble version`" otherwise.
4. **Tests** (`internal/cli`):
   * `TestACPModelFlags`: `gobble acp --model openai/gpt-zz:low --thinking high` gives agent options whose `Override`
     is `openai`, `gpt-zz:low`, `high`. An unknown model is a usage error, as chat's is.
   * `TestMisplacedChatFlags`: `gobble --model x version`, `gobble --print version` and
     `gobble --thinking high config path` each exit 2 with the message. `gobble --model openai/gpt-zz -p hi` and
     `gobble -p hi --model openai/gpt-zz` still run chat.
   * Today's `modelflags_test.go`, `acp_test.go` and the chat flag tests pass.
5. **Seen failing** (C2), on a scratch copy:
   * the tree before P8 with the new tests: `ACPCmd` has no `Model`, so it does not compile;
   * `Run` not calling `pickModel`: `TestACPModelFlags` fails;
   * the refusal removed: `TestMisplacedChatFlags` fails;
   * the refusal applied to chat too: its still-running cases fail.
6. **Acceptance:** P6's driver gains a sixth case, `gobble acp --model opencode-zen/accept-model` with no
   `session/set_config_option`. Its request goes to the scripted path, and its answer is case 1's. Run on macOS, the
   Linux machine and the Windows laptop.
7. **Docs:** `docs/architecture.md`'s `internal/cli` row; any guide that names `gobble acp`'s flags.
8. **Gates** (C6), with the remote tests. Staged; the owner commits.

### P8 record (2026-11-10)

C1: `HEAD` `ef6a870` (P7), `origin/main` `ece13c4` (P2 onwards not yet pushed), the tree clean; acpext `v0.3.0`,
go-llmprovider-sdk `v1.28.0` and kong `v1.16.1` required.

**Deviation 1 (2026-11-10), the serving flags beside `--login`.** With P8 staged, the hand-off named a silent drop the
phase left. `gobble acp --login <provider>` signs in and exits before `choose`, so `--model`, `--provider` and
`--thinking`, which P8 gave `gobble acp`, are accepted beside it and ignored. `--mode` beside `--login` already was,
before 0049. MADR D12 says no flag is silently dropped. The owner chose option (a): `gobble acp` refuses each of
`--mode`, `--provider`, `--model` and `--thinking` given with `--login`, a usage error, exit 2. This is step 2b, below
step 2's list. The MADR's D12 gains the bullet. No file is added to the phase: `internal/cli/acp.go` and
`internal/cli/acp_test.go` are P8's already.

1. **The probe** (step 1), `p8_probe_test.go` on a scratch copy, never in the tree. It parsed nine command lines and
   printed each `kong.Context.Path`:
   * A chat flag before another command is a `Path` element whose `*kong.Flag` is one of chat's own node's
     `Flags`, while `kctx.Selected()` is that other command. `--model x version` gave `app flag:model(chat)
     cmd:version`; `--model x acp` gave `app flag:model(chat) cmd:acp`.
   * Chat's own runs select chat, with the flag on either side: `--model x -p hi` and `-p hi --model x` both give
     `cmd:chat`.
   * A root flag is not chat's: `--log-level debug version` gave `flag:log-level`.
   * Kong's `trace` (`context.go:399-404` at `v1.16.1`) matches flags at the root against the default command's when
     it is `default:"withargs"`, which is why they parse.

   So Kong tells a misplaced flag from chat's own, and step 3 runs as written.
2. **`gobble acp`'s flags:**
   * `ModelFlags` (`--provider`, `--model`, `--thinking`) is a new struct in `internal/cli/flags.go`. `SharedFlags`
     embeds it where the three fields were, so chat's flags, help and completion are unchanged; the CLI's tests,
     completion's included, pass unchanged.
   * `ACPCmd` embeds it. A new `choose` sets the mode, refuses `--provider` without `--model`, and stores
     `pickModel`'s result in `e.pick`, which `agentOptions` passes on as `acpserver.Override`.
   * `pickModel` takes `*ModelFlags`. `chat.go` passes chat's, and six test calls construct `ModelFlags`.
   * A custom model id under `gobble acp` gets Pi's warning on stderr, as chat's does; stdout stays the protocol's.
3. **Misplaced flags:** `misplacedFlag` (`internal/cli/main.go`) runs right after parsing. A `Path` flag that is one of
   chat's own, while the selected command is not chat, is a usage error, exit 2:
   * when the selected command has a flag of that name: `--model is a flag of gobble's chat command; put it after
     `gobble acp``. `--mode` before `acp` is refused the same way, since acp has `--mode` too;
   * otherwise: `--print is a flag of gobble's chat command, not of `gobble version``.
4. **Tests:**
   * `TestACPModelFlags` (`acp_test.go`): `--model openai/gpt-zz:low --thinking high` gives the `Override` `openai`,
     `gpt-zz:low`, `high`. An unknown model and `--provider` alone are usage errors with chat's messages.
   * `TestMisplacedChatFlags` (`main_test.go`), nine command lines:
     * refused: `--model x version`, `--print version`, `--thinking high config path`, `--model … acp` and
       `--mode plan acp`;
     * still running: chat with `--model` on either side of `-p hi`, which exits 3 for want of a credential as
       before; `--log-level warn version`; and `acp --model openai/gpt-zz`, exit 0 with Pi's custom-id warning.
   * Every other test in `internal/cli/...` passes, `TestPickModel` with `ModelFlags`.
5. **Seen failing** (C2), `p8_0049_breaks.py`, a scratch copy per case, `go test ./internal/cli/`, the proxies at
   `127.0.0.1:9`, `GOPROXY=off`. The control passed; `cases not as expected: 0`:
   * the tree before P8 (`acp.go`, `main.go`, `flags.go`, `modelflags.go`, `chat.go` at `HEAD`) with P8's tests: does
     not compile, `acp_test.go:43:15: unknown field Model in struct literal of type ACPCmd`, `undefined: ModelFlags`;
   * `choose` not resolving the flags: `TestACPModelFlags`, `acp_test.go:48: override {Provider: Model: Thinking:
     Mode:}, want openai/gpt-zz:low at high`; and `TestMisplacedChatFlags`, no custom-id warning;
   * `choose` without `--provider`'s check: `TestACPModelFlags`, `:60: --provider alone: <nil>, want the usage error`;
   * the refusal removed: `TestMisplacedChatFlags`, `main_test.go:81: gobble ["--model" "x" "version"]: exit 0, stderr
     ""; want 2 …`, the same for all five refused lines;
   * the refusal applied to chat too: 57 tests fail, `TestMisplacedChatFlags` among them, e.g. `gobble ["-p" "hi"
     "--model" "openai/gpt-zz"]: exit 2, stderr "Error: --print is a flag of gobble's chat command; put it after
     `gobble chat`"`;
   * the put-after message never offered: `TestMisplacedChatFlags`, `--model is a flag of gobble's chat command, not
     of `gobble acp``.
6. **Acceptance:** P6's driver gained case 6. `gobble acp --model opencode-zen/accept-model` is followed by
   `session/new` and the prompt, with no `session/set_config_option`. Its checks: `session/new`'s model option reads
   `opencode-zen/accept-model`; the one request goes to `/zen/v1/chat/completions`; the answer's data is case 1's,
   `resetAt` aside.

   | Host | Checks | Failed | Exit |
   | :--- | :--- | :--- | :--- |
   | macOS (arm64) | 41 | 0 | 0 |
   | The Linux machine (amd64) | 41 | 0 | 0 |
   | The Windows laptop (amd64), with `--no-appdata` (P7) | 41 | 0 | 0 |

   Seen failing: the driver against a binary built from `git archive ef6a870`, the tree before P8. Cases 1 to 5 pass.
   Case 6's `gobble acp --model …` exits at once with `Error: unknown flag --model, did you mean "--mode"?`, and the
   driver stops at `initialize: gobble exited`, exit 1.
7. **Docs:**
   * `README.md`'s `gobble acp` item and `docs/architecture.md`'s ACP agent item name the three flags.
   * `docs/architecture.md`'s `internal/cli` row names the refusal and the shared flags.
   * `docs/guides/choosing-a-model.md`'s first rule says `gobble acp --model` does the same, after `acp`, and that
     `gobble --model … acp` is refused.
   * `docs/guides/credentials.md` names only `gobble acp --login` and stays as it is.
8. **Gates** (C6):
   * `make pre-add-check FILES="internal/cli/flags.go internal/cli/acp.go internal/cli/modelflags.go
     internal/cli/chat.go internal/cli/main.go internal/cli/acp_test.go internal/cli/main_test.go
     internal/cli/modelflags_test.go internal/cli/choice_test.go"`: `9 file(s) clean (gofmt, golint, govulncheck)`.
   * macOS: the first `make preflight` failed at `lint` with one finding in P8's own test, `acp_test.go:43:15: embedlit:
     embedded field type can be removed from struct literal (modernize)`. The literal now names `Model` and `Thinking`
     directly. After that, `make pre-add-check FILES="internal/cli/acp_test.go"` was clean, `make preflight` exit 0
     (`preflight passed`) and `make race` exit 0 (39 ok). The breaks above were re-run on that file, and the compile
     line quoted is from the re-run.
   * The remote runs below used the test before that literal change. The change is in how one test's value is
     written, not in what it checks, and no other file changed.
   * `make check-records markdownlint` after this record: exit 0. The identifier scan of the diff (the local account
     name, the dev hosts' names, the org's mail domain, and real-machine home and temporary paths): nothing found.
   * The Linux machine and the Windows laptop (`TMP`/`TEMP` set), a scratch copy of 926 files each: acpext's tests
     ok; `go test -count=1 ./...` 38 ok; `-race` 38 ok; on each host, exit 0 and no `FAIL`.

**Deviation 1's change (step 2b):**

1. `ACPCmd.Run` refuses `--login` beside any of `--mode`, `--provider`, `--model` and `--thinking`, before signing in,
   with `servingFlag`: `--model is for the sessions gobble acp serves; --login serves none`, exit 2.
2. `TestACPLogin` gains the four flags beside `acp --login grok`. Each exits 2 with the message, and the test's
   stubbed sign-in runs once, for the first, unrefused call.
3. **Seen failing**, two cases added to `p8_0049_breaks.py`; all eight breaks and the control re-run, `cases not as
   expected: 0`:
   * the check removed: `TestACPLogin`, `acp_test.go:38: exit 0, stdout "Logged in to grok\n", stderr ""; want 2, "",
     "Error: --mode is for the sessions gobble acp serves; --login serves none"`. The sign-in is the test's stub, so
     no real one ran;
   * `--mode` alone checked: `TestACPLogin`, the same line for `--provider`.
4. **Gates:**
   * The first `make preflight` failed at `lint`. `goconst` found `"provider"` and `"thinking"` repeated, in
     `servingFlag`'s list among others. The list now names each flag with its dashes (`"--mode"` and the rest), a
     literal used nowhere else in `internal/cli`, and the message is unchanged.
   * After that: `make pre-add-check FILES="internal/cli/acp.go internal/cli/acp_test.go"`, `2 file(s) clean`;
     `make preflight` exit 0, `preflight passed`; `make race` exit 0, 39 ok.
   * The Linux machine and the Windows laptop ran `go test ./...`, 38 ok, `-race`, 38 ok, and acpext's tests, each
     with exit 0 and no `FAIL`. They ran the tree before the `goconst` change, which spells four literals differently
     and changes no behaviour or message. The loopback acceptance does not reach `--login` and was not re-run.
   * `make check-records markdownlint` after this entry: exit 0. The identifier scan of the diff found nothing.

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
| 11 `GOBBLE_HOME` needs no platform directory (amendment) | P7 |
| 12 `gobble acp`'s model flags, misplaced chat flags refused (amendment) | P8 |

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
