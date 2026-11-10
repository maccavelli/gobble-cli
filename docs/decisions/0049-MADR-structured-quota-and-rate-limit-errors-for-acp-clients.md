---
status: accepted
date: 2026-11-09
decision-makers: repository owner
consulted: "this repository at 74d7f66 (main, v0.5.0 + records, 2026-11-09); acpext v0.2.0; go-llmprovider-sdk v1.28.0 (llmprovider/api_error.go, retry.go, api_error_test.go); acp-go-sdk2 v1.34.2 (errors.go, SessionUpdateNotice); maccavelli/gobble-cli#14; maccavelli/go-llmprovider-sdk#13; magic-cli-remote 0243-MADR-show-exactly-which-usage-limit-an-agent-hit-and-when-it-lifts.md at 119a0338"
informed: "magic-cli-remote (its 0243-MADR D7), go-llmprovider-sdk (#13)"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Send ACP clients a failed turn's limit as data: which kind, which limit, and when it lifts

## Context and Problem Statement

When a model call fails on a quota, a usage limit or a rate limit, gobble tells its ACP client in one flat string. The
go-llmprovider-sdk error that caused it is still in the error chain inside gobble, with the provider, the HTTP status,
the provider's own code, its redacted message and the wait it asked for. None of that reaches the client as data.
magic-cli-remote is changing its phone card to say exactly which limit was hit and when it lifts
(`0243-MADR-show-exactly-which-usage-limit-an-agent-hit-and-when-it-lifts.md`, its D3, D5 and D7). It can only be as
precise as what gobble sends. The request is [maccavelli/gobble-cli#14](https://github.com/maccavelli/gobble-cli/issues/14).

This record decides what gobble sends, how the data crosses gobble's own package boundaries to get there, and what
waits for go-llmprovider-sdk ([maccavelli/go-llmprovider-sdk#13](https://github.com/maccavelli/go-llmprovider-sdk/issues/13),
which the owner is working on as this is written).

### What a client gets today

gobble 0.4.0, for an OpenCode Zen free-tier limit, as magic-cli-remote captured it (#14; 0243-MADR F1):

```json
{"code":-32603,"message":"Internal error","data":{"error":"llm: api status 429: llm: quota exhausted: llmprovider: quota exhausted: opencode-zen/chat_completions HTTP 429 FreeUsageLimitError (retry-after 1h33m21s): Free usage limit reached. Free models work in OpenCode (https://opencode.ai); subscribe to OpenCode Go to keep going: https://opencode.ai/go"}}
```

### What holds at `74d7f66` (read 2026-11-09)

Each item was read in the tree or the module cache, not taken from the issue.

* **F1. One string.** `promptError` (`acpserver/prompt.go:267-278`) answers a failed turn with `-32000`
  (auth_required) for `llm.ErrAuth`, a fixed advice text for `llm.ErrWorkspace` (0035-MADR D5), and `-32603` otherwise,
  each with `acpext.ErrorData{Error: err.Error()}`. It is called from five places: `acpserver/agent.go:630` (no
  provider for the session) and `:751` (the turn failed), `acpserver/compact.go:28` and `:44` (`/compact`), and
  `acpserver/extensions.go:204` (`_gobble/compact`).
* **F2. `acpext.ErrorData`** (`acpext/types.go:327-339`, acpext `v0.2.0`) has `command`, `error`, `provider`, `reason`
  and `sessionId`, each `omitzero`, declared in their JSON names' sorted order, which `TestErrorDataSortedMembers`
  (`acpext/roundtrip_test.go:187-197`) holds (0042-MADR A3). `provider` is filled only by `authenticate`'s refusal
  (`acpserver/auth.go:67`), never on a failed turn.
* **F3. The SDK's error.** `*llmprovider.APIError` (go-llmprovider-sdk `v1.28.0`, `llmprovider/api_error.go:99-124`)
  has `Provider` (`"gateway/route"` for a gateway, such as `opencode-zen/responses`), `Status` (0 for a failure inside
  a stream), `Kind` (a sentinel), `Code` (the service's error type or code), `Message` (redacted, at most 512 bytes),
  `RetryAfter` and `Reason`, and a `Retryable()` method (`:176-188`: false for exhausted quota, true for a rate limit
  or an unavailable service the service did not mark final). `RetryAfter` comes from `retry-after-ms` or `Retry-After`,
  else from the body: ChatGPT's `resets_at` turned relative with `time.Until` (`:256-261`), or Google's `RetryInfo`.
* **F4. The package boundary.** `internal/archtest` rule 4: only `llm/provider` imports go-llmprovider-sdk
  (`docs/architecture.md:162`; `internal/archtest/check.go:308`). `acpserver` imports `llm` and not `llm/provider`
  (`go list -f '{{.Imports}}' ./acpserver`). So `acpserver` cannot read `*llmprovider.APIError`, as #14's requirement 1
  says it should; whatever it sends must cross in gobble's own `llm` types. `llm` is a **stable** package
  (`docs/architecture.md:104`): it may only grow.
* **F5. What crosses today.** `llm/provider.mapError` (`llm/provider/provider.go:674-692`) wraps the SDK error with the
  first matching `llm` sentinel (`sentinels`, `:598-613`: workspace, quota before rate-limited, context overflow,
  incomplete, auth, not-permitted, offline before unavailable, unsupported), and returns an `*llm.APIError`
  (`llm/errors.go:42-48`) with only `Status`, `RetryAfter` and the wrapped error. The provider's code, its message,
  retryability and the provider's name are reachable only as text.
* **F6. The retry notice.** `llm/provider.New` wraps every provider in `llmprovider.WithStreamRetry(p, policyOf(r),
  notice)` (`:232`). Each SDK `RetryNotice` (`Provider`, `Attempt`, `MaxAttempts`, `Wait` with jitter,
  `ServerDirected`, `Err`; `llmprovider/retry.go:36-51`) becomes `llm.ProviderRetry{Attempt, MaxAttempts, Wait, Kind}`
  (`llm/provider/provider.go:323-327`; `llm/event.go:8-18`), then `agent.Retry` (`agent/agent.go:693-700`;
  `agent/event.go:68-75`), then `_gobble/retry`'s `acpext.RetryNotice{sessionId, phase, cause, attempt, maxAttempts,
  delayMs, error}` (`acpserver/prompt.go:105-108`; `acpext/types.go:46-57`). The server's own requested wait, whether
  the server directed it, its code and its status are dropped on the way.
* **F7. A wait over the cap sends nothing.** The SDK does not wait longer than `RetryPolicy.MaxDelay` and calls no
  notify for such a wait (`retry.go:27-30`, `:105-107`); gobble's default cap is 60 s (`llm.DefaultRetry`,
  `llm/llm.go:42`). Exhausted quota is never retried (F3). In both cases the client gets the -32603 string alone, and
  a turn that failed after retries says so only as text: `llmprovider: <id> failed after N attempts: …`
  (`retry.go:177`).
* **F8. The SDK's limit vocabulary.** `classifyAPIError` (`api_error.go:410-471`) makes these exhausted quota: OpenCode
  `FreeUsageLimitError`, `GoUsageLimitError`, `BlackUsageLimitError`, `CreditsError`, `MonthlyLimitError`,
  `UserLimitError` (`:383`); Kilo `PROMOTION_MODEL_LIMIT_REACHED`, or a body containing `FreeUsageLimitError`; OpenAI
  `usage_limit_reached`, `insufficient_quota`, `subscription_sharing_usage_limit_exceeded` (`:387`); any HTTP 402; and
  Gemini's 429 when its `QuotaFailure` names a per-day quota (`:445`, `:596`). Every other 429 is a rate limit. The
  SDK's fixtures (`api_error_test.go:30-43`) show the code each carries; Kilo's free-usage-by-text case carries no code
  (`Code ""`).
* **F9. A 401 quota is worded as a refused key.** OpenCode sends `MonthlyLimitError`, `CreditsError` and
  `UserLimitError` with HTTP 401 (`api_error_test.go:34`). `APIError.Unwrap` (`api_error.go:162-170`) then returns both
  `ErrQuotaExhausted` and the status sentinel `ErrAuthFailure`. gobble's mapping tests quota first, so the turn is
  `llm.ErrQuota`, not `llm.ErrAuth`, and the client is not sent to sign in. But `adapter.rejected`
  (`llm/provider/provider.go:275-285`) tests `llmprovider.ErrAuthFailure` on the SDK error itself, so with a key from a
  named source the error reads "opencode-zen rejected the API key from auth.json: …", and with a stored login it is
  led by `LoginRejected`. Found by reading the code; no test covers a 401 quota (`TestWireRejectedKey`,
  `wire_test.go:404-428`, sends `invalid_api_key` only).
* **F10. gobble's own client prints the whole error.** `acpclient` returns the SDK's `*acp.RequestError`
  (`acpclient/conn.go:331-336`), whose `Error()` is the error object as compact JSON (acp-go-sdk2 `errors.go:45-57`).
  The line session prints it as is (`internal/cli/line.go:391`), and print mode puts it in its result's `error`
  (`internal/cli/print.go:209`). Every member added to `ErrorData` lengthens that line.
* **F11. The provider-state mark.** A failed turn's `ErrAuth`, `ErrQuota` or `ErrNotPermitted` marks its provider
  (`acpserver/choice.go:254-284`), shown by `/providers` and `/model` as "quota exhausted (HTTP 429)". A mark's reason is
  gobble's words and the status, never the provider's text (0025-MADR D4, as amended 2026-10-20), and it has no reset
  time, so it stays until a sign-in or a later listing.
* **F12. Notices.** acpserver can queue an ACP `notice` session update for a client that offered `session.notices`
  (`acpserver/notice.go:22-51`, 0016-MADR D8). acp-go-sdk2 `v1.34.2`'s `SessionUpdateNotice` has `_meta`, `description`,
  `severity` and `title` (`types_gen.go:15869-15883`).
* **F13. The contract's rules.** acpext is its own module, tagged `acpext/vX.Y.Z`; a contract change is a new minor
  tag, committed, pushed and tagged before the root requires it (0042-MADR D5). A client must take a member or a value
  it does not know (`acpext/doc.go`, "Versions"). Only `acpserver`, `acpclient/...` and `internal/cli/...` import acpext
  (archtest rule 12).
* **F14. What the SDK does not have yet** (#13, read at `v1.28.0`): an absolute reset instant (`resets_at` and an
  HTTP-date become a relative `RetryAfter`); ChatGPT's `plan_type`; Gemini's per-day flag or `quotaId` as a field;
  Kilo's `metadata`; a `RetryAfter` on a stream failure. The SDK's `main` was still at its `v1.28.0` closeout
  (`d62142c`) on GitHub when this was read; no names for those fields exist yet.

### What magic-cli-remote will read

Its 0243-MADR D3 describes a limit by `Scope` (an open set: `rate`, `window`, `daily`, `weekly`, `monthly`, `credits`,
`free_tier`, `plan`), `WindowMinutes`, `Provider` (the upstream, route suffix dropped), `Code` and `ProviderMessage`, and
D7 reads gobble's structured data when a gobble release sends it, and gobble's text until then. Its owner chose to show
the provider's message on the phone (0243-MADR Q2).

## Decision Drivers

* **Accuracy.** Say which limit and when it lifts, as far as a source states it. Never infer a window from how long the
  wait is: a 4-hour retry-after does not make a 5-hour window.
* **Additive on the wire.** `error` keeps its text, so magic-cli-remote's text reading and every other client keep
  working; every new member is optional (0042-MADR D5; `acpext/doc.go`).
* **The boundaries hold.** archtest rule 4 (only `llm/provider` imports the SDK) and rule 12 (who imports acpext) are
  not loosened; `llm` grows only by additions.
* **One classification.** The provider's code is read once, in `llm/provider`, beside the SDK.
* **No new leak.** Provider text reaches a client under the same rule as today's `error`; gobble's own state reasons
  keep 0025-MADR D4.
* **Ship without waiting.** What `v1.28.0` already carries goes out now; what needs the SDK lands when its tag does.
* **gobble's own client does not get worse** (F10).

## Considered Options

* **A. Text only.** Fix the 401 wording (F9) and leave the wire as it is; clients keep parsing `error`.
* **B. Structured `ErrorData`, carried through gobble's `llm` facade, in a new acpext minor tag; `_gobble/retry`
  widened the same way; the SDK's #13 fields adopted when tagged.**
* **C. As #14 words requirement 1: `promptError` reads `*llmprovider.APIError` from the chain.**
* **D. Wait for go-llmprovider-sdk #13, then do B in one step.**
* **E. A structured `session.notices` notice instead of new error members.**

## Decision Outcome

Chosen option: "B", because it is the only option that sends the limit as data now without loosening a package
boundary (C breaks archtest rule 4, F4), keeps today's `error` for every reader (A sends nothing new), does not hold the
parts `v1.28.0` already carries hostage to another repository's release (D), and answers on the channel every client
already reads, the prompt's error, rather than one only clients that offered `session.notices` get (E).

### D1. The wire: `ErrorData` gains nine members, in acpext `v0.3.0`

acpext's `ErrorData` becomes, in its JSON names' sorted order (0042-MADR A3), each `omitzero`:

| Member | Go | Meaning |
| :--- | :--- | :--- |
| `attempts` | `Attempts int` | Tries made, when the failure came after retries (2 or more); absent otherwise |
| `code` | `Code string` | The provider's own code, such as `FreeUsageLimitError` |
| `command` | `Command string` | unchanged |
| `error` | `Error string` | unchanged: the whole error text, as today |
| `httpStatus` | `HTTPStatus int` | The provider's HTTP status; absent for a failure inside a stream |
| `kind` | `Kind ErrorKind` | gobble's class of the failure (D3); set on every failed turn |
| `limit` | `Limit LimitScope` | Which limit, when a source states it (D4); absent otherwise |
| `provider` | `Provider string` | gobble's provider id of the session, such as `opencode-zen`; now set on a failed turn too |
| `providerMessage` | `ProviderMessage string` | The provider's own redacted message alone, without gobble's or the SDK's prefixes |
| `reason` | `Reason string` | unchanged |
| `resetAt` | `ResetAt string` | When the limit lifts, RFC 3339 in UTC to the second (D5) |
| `retryAfterMs` | `RetryAfterMs int64` | The wait the provider asked for, in milliseconds, not jittered |
| `retryable` | `Retryable *bool` | Whether the same request can succeed later, when gobble knows; absent when it does not |
| `sessionId` | `SessionID string` | unchanged |

* `Retryable` is a pointer so that a known `false` (exhausted quota) is sent and an unknown one is not.
* `ResetAt` is a string, so the format is gobble's choice, not an encoder's, and acpext stays free of `time`.
* `ErrorKind` and `LimitScope` are new string types in acpext with a constant for each value, and the doc says, as for
  `RetryCause`, that both are open sets a client must take unknown values of.
* `LimitScope`'s constants are magic-cli-remote's names (0243-MADR D3): `LimitRate` `rate`, `LimitWindow` `window`,
  `LimitDaily` `daily`, `LimitWeekly` `weekly`, `LimitMonthly` `monthly`, `LimitCredits` `credits`, `LimitFreeTier`
  `free_tier`, `LimitPlan` `plan`. gobble sends `window` and `weekly` only when a source states the window (D10);
  nothing does at `v1.28.0`.
* A window's length (0243-MADR's `WindowMinutes`) gets no member now: no source states one. It is a later minor tag,
  with its first source.
* The doc's paragraph on refused requests names the new members and which errors carry them.

### D2. `_gobble/retry` gains four members, in the same tag

`RetryNotice` gains, after its present members and each `omitzero`, on a start that a provider's retry began:
`retryAfterMs` (`RetryAfterMs int64`, the server's value, where `delayMs` stays the jittered wait gobble takes),
`serverDirected` (`ServerDirected bool`), `code` (`Code string`) and `httpStatus` (`HTTPStatus int`). Its members are
not in sorted order today and are not reordered: appending keeps every current byte.

A wait the SDK refuses over the cap (F7) gets no notice of its own. The turn's -32603 answers at once, and its data now
says what happened: `kind` `rate-limited`, `retryable` true, `retryAfterMs` above the session's cap, `resetAt`, and no
`attempts` beyond 1. acpext's doc says so.

### D3. The failure's kind, named once, in `llm`

`llm` gains `Kind(err error) string`, which names an error by gobble's sentinels, most specific first: `workspace`,
`quota`, `rate-limited`, `context-overflow`, `empty_reply`, `incomplete`, `auth`, `not-permitted`, `offline`,
`unavailable`, `unsupported`, then `timeout` (`context.DeadlineExceeded`), `canceled` (`context.Canceled`), else
`other`. `empty_reply` comes before `incomplete` because an empty answer wraps both (`mapError`, `:685-687`).

* Where a class is also a `_gobble/retry` cause, the name is the same: `quota`, `rate-limited`, `auth`,
  `not-permitted`, `unavailable`, `workspace`, `timeout`, `canceled`, `other`, `offline`, `empty_reply`
  (`acpext/types.go:31-44`).
* `llm/provider.FailureKind` (`list.go:106-133`) is left as it is: it names log lines and retry causes, and changing
  what it returns for an offline or empty failure would change those. A test holds the two equal on every class both
  name.
* acpext's `ErrorKind` constants are these names. A test in `acpserver`, which imports both, holds `llm.Kind`'s outputs
  inside acpext's constants.

### D4. The limit, from the provider's code, in `llm/provider`

`llm/provider` sets the limit when it maps an SDK error, from the SDK's kind, the provider's code, the service and the
status. It is read from what the provider stated, never from the length of a wait:

| Source | `limit` |
| :--- | :--- |
| `FreeUsageLimitError` (OpenCode), `PROMOTION_MODEL_LIMIT_REACHED` (Kilo) | `free_tier` |
| `MonthlyLimitError` (OpenCode) | `monthly` |
| `CreditsError` (OpenCode), `insufficient_quota` (OpenAI), `insufficient_balance` (Kilo), any HTTP 402 | `credits` |
| `GoUsageLimitError`, `BlackUsageLimitError` (OpenCode), `usage_limit_reached`, `subscription_sharing_usage_limit_exceeded` (OpenAI) | `plan` |
| Gemini, exhausted quota on HTTP 429 | `daily` |
| any other rate limit (`ErrRateLimited`) | `rate` |
| anything else, `UserLimitError` and Kilo's free usage named only in its message included | empty |

* **Gemini's `daily`** rests on the SDK's classifier at `v1.28.0`: a Gemini 429 is exhausted quota only when its
  `QuotaFailure` names a per-day quota (F8). It is replaced by the SDK's own flag when a tag exposes one (D10).
* **`UserLimitError`** is a quota (F8) whose period no source states, so it stays empty rather than guessed.
* **Kilo's free usage by message** reaches gobble with no code (F8), so it stays empty until the SDK carries one.
* `llm` gains the eight scopes as `llm.Limit*` constants, so `llm/provider` names them without acpext (archtest rule
  12). A test in `acpserver` holds them equal to acpext's.

### D5. When it lifts, and the wait asked for

`llm/provider` fills, on the `*llm.APIError` it already returns (F5):

* `RetryAfter`, as now.
* `ResetAt time.Time`: the provider's absolute instant once the SDK gives one (D10); until then, the time the error was
  classified plus `RetryAfter`, when `RetryAfter` is positive. The SDK itself turns `resets_at` into `RetryAfter` with
  `time.Until` at classification, so the instant gobble rebuilds is the provider's within the call's own latency. Zero
  when there is no wait, as for credits.
* acpserver sends `resetAt` from it as RFC 3339 in UTC, truncated to the second, and `retryAfterMs` from `RetryAfter`.

### D6. The facade carries the rest

`llm.APIError` (stable, F4) gains fields, each zero when unknown, filled only by `llm/provider`:

* `Provider string`: gobble's provider id (`a.id`), not the SDK's `gateway/route` string, which stays in the error's
  text. This is the id a gobble user selects, and the upstream magic-cli-remote shows (0243-MADR D3: route suffix
  dropped).
* `Code string` and `Message string`: the SDK's `Code` and its redacted `Message`.
* `Retryable *bool`: the SDK's `Retryable()`, when the SDK error is an `*APIError`.
* `ResetAt time.Time` (D5), `Limit string` (D4), `Attempts int`: the tries made, when the failure came after the SDK's
  retries (counted from the retry notices the call saw: the last notice's `Attempt` plus one).

`llm.ProviderRetry` gains `RetryAfter time.Duration`, `ServerDirected bool`, `Code string` and `Status int`, from the
notice and its error; `agent.Retry` gains the same, and acpserver writes them into D2's members. Every change to `llm`,
`agent` and acpext is an addition: no field, type or function is removed or changes type.

### D7. The prompt's error is built from it

`promptError` takes the session's provider id and the error, and builds `ErrorData` with `error` as today, `provider`
and `kind` always, and, from an `*llm.APIError` in the chain, `httpStatus`, `code`, `providerMessage`, `retryAfterMs`,
`resetAt`, `retryable`, `attempts` and `limit`. The codes do not change: `-32000` for `llm.ErrAuth`, `-32603` otherwise.

* All five callers (F1) pass the provider: the session's, from `a.provOf(s)`.
* The workspace refusal keeps its fixed advice and sends no `providerMessage`, as 0035-MADR D5 keeps the service's words
  out of it; it sends `provider` and `kind` `workspace`.
* **Provider text.** `providerMessage` is the same SDK-redacted text that `error` already carries in full, under the
  same rule, so it adds no exposure. 0025-MADR D4 still governs gobble's own provider-state reasons, which stay gobble's
  words (D9).

### D8. A quota is never worded as a refused key

`adapter.rejected` leaves an error that is also `llmprovider.ErrQuotaExhausted` unchanged, for a key from a source and
for a stored login alike (F9). A test with OpenCode's 401 `MonthlyLimitError` is written first and seen failing on the
code before the fix.

### D9. Questions for the owner, and their answers

* **Q1. A `session.notices` notice for a limit.**
  * **(a) Not in this record.** *(recommended)* The prompt's error already carries every member; a notice would repeat
    it to a subset of clients, and magic-cli-remote reads the error (0243-MADR D7).
  * (b) Also queue a `warning` notice, its title "Quota exhausted on <provider>" or "Rate-limited by <provider>", its
    description the provider's message and when it lifts, and the same `ErrorData` under `_meta.gobble`, for a client
    that offered `session.notices`.
  * (c) As (b), and make it the only carrier of the new members (option E).
* **Q2. The reset time on gobble's provider-state mark (F11).**
  * **(a) Keep the mark's reset time, show it, and let the mark lapse when it passes.** *(recommended)* `/providers` and
    `/model` read "quota exhausted (HTTP 429) — resets 14:05"; after that time the mark is dropped, so the provider is
    offered again without a sign-in. The time is gobble's datum, not provider text, so 0025-MADR D4 holds.
  * (b) Show the time, keep the mark until a sign-in or a listing, as today.
  * (c) Not in this record.
* **Q3. gobble's own client (F10).**
  * **(a) Decode the data in `acpclient`, and print the text.** *(recommended)* `acpclient` returns a turn's refused
    prompt as an error that carries the decoded `acpext.ErrorData` and whose text is its `error`; the line session
    prints that and, when `limit` or `resetAt` is set, one more line ("free tier limit on opencode-zen; resets at 14:05,
    in 1h33m"); print mode's result keeps `error` as that text and gains `errorData`, the object. The CLI's line is no
    longer a JSON blob, and gobble's own client proves the members.
  * (b) Leave the CLI as it is: its line grows with every member.
  * (c) As (a) for the line session only; print mode unchanged.
* **Q4. The SDK's #13 fields.**
  * **(a) A phase of this record's PLAN, run when the owner names the go-llmprovider-sdk tag that carries them.**
    *(recommended)* Its steps are amended to the tag's names before it runs.
  * (b) A pair of its own, later.
* **Q5. `resetAt` before the SDK gives an instant.**
  * **(a) Rebuild it from `RetryAfter` at classification (D5).** *(recommended)* It is what #14 asks, and within the
    call's latency of the provider's instant.
  * (b) Send only `retryAfterMs` until the SDK gives an instant.

#### Answers (2026-11-09)

The owner: "Q1: A, Q2: A, Q3: A, Q4: A, Q5: A". Each is the recommended option.

* **Q1:** (a), no `session.notices` notice; the prompt's error carries the members.
* **Q2:** (a), the mark keeps its reset time, shows it, and lapses when it passes.
* **Q3:** (a), `acpclient` decodes the data; the line session prints the text and a limit line; print mode gains
  `errorData`, and its `error` becomes the text.
* **Q4:** (a), the SDK's #13 fields are a phase of this record's PLAN, run when the owner names the tag.
* **Q5:** (a), `resetAt` is rebuilt from `RetryAfter` at classification until the SDK gives an instant.

### D10. What the SDK's #13 changes, when tagged (Q4)

When a go-llmprovider-sdk tag carries #13's fields, a phase of the PLAN pins it (`make check-libs` first, as 0035-MADR
D2 requires) and maps, in `llm/provider` only:

* the absolute reset instant into `llm.APIError.ResetAt`, in place of D5's rebuilt one;
* the per-day flag (or `quotaId`) into `limit` `daily`, in place of D4's Gemini inference;
* a stated window, if any provider states one parseably, into `window` (and a length member, in a further acpext
  minor tag) or `weekly`; never from the wait's length;
* a stream failure's `RetryAfter` and reset into the same fields, which today are zero for a stream (F14);
* ChatGPT's `plan_type` and Kilo's `metadata`: only if a later record decides a member for them; this record does not.

### Consequences

* Good, because magic-cli-remote, and any ACP client, can say which limit was hit, from which provider, with its code,
  the provider's words, and when it lifts, from data rather than by parsing a Go error chain.
* Good, because a wait gobble refused over its cap is visible as data, where today it is visible only as text.
* Good, because a 401 quota stops reading as a refused key (D8).
* Good, because the boundaries hold: acpserver reads gobble's `llm` types, and only `llm/provider` reads the SDK.
* Neutral, because `error` keeps its long text; a client that wants the provider's words alone reads `providerMessage`.
* Neutral, because `limit` is empty for `UserLimitError`, for Kilo's free usage named only in its message, and for every
  window and weekly limit, until a source states them.
* Neutral, because `resetAt` is gobble's reconstruction until the SDK gives an instant (Q5), off by at most the call's
  latency.
* Bad, because acpext takes a minor tag, `v0.3.0`, that the owner tags and pushes before the root can use it (0042-MADR
  D5), and magic-cli-remote must move to it to decode the new members.
* Bad, because `llm.APIError` grows from three fields to nine; it is the stable facade's error type, and every field
  must stay.

### Confirmation

1. acpext `v0.3.0`: `ErrorData`'s members encode in sorted order with every member set; each new frame in
   `acpext/testdata` round-trips under `encoding/json` and `encoding/json/v2` with unknown members refused; apidiff
   reports only compatible additions from `acpext/v0.2.0`.
2. A canned OpenCode Zen HTTP 429 `FreeUsageLimitError` with `Retry-After: 5601`, through `gobble acp`'s prompt, answers
   -32603 with `error` unchanged from today's text, `provider` `opencode-zen`, `kind` `quota`, `httpStatus` 429, `code`
   `FreeUsageLimitError`, `providerMessage` the provider's message alone, `retryAfterMs` 5601000, a `resetAt` within a
   second of the call plus 5601 s, `retryable` false and `limit` `free_tier`.
3. Each row of D4's table, through `llm/provider` on a canned wire, gives its `limit`; an unlisted code gives none; a
   long wait alone never gives `window`.
4. A 429 rate limit retried once and then served gives a `_gobble/retry` start with `retryAfterMs`, `serverDirected`,
   `code` and `httpStatus`; a 429 asking for more than the cap gives no notice and a -32603 with `retryable` true and
   `retryAfterMs` above the cap; a turn that fails after retries carries `attempts`.
5. OpenCode's 401 `MonthlyLimitError` with a key from `auth.json` is `llm.ErrQuota` and its text does not say the key
   was rejected; the test failed on the code before D8.
6. `internal/archtest` passes unchanged; acpserver still does not import `llm/provider` or the SDK.
7. Each new test is seen failing against a planted break, on a scratch copy.
8. The answers to Q1 to Q3 are built as answered, each with its tests.
9. `make preflight` and `make race` pass, and the tests and race tests on the Linux machine and the Windows laptop.
10. Q4: when the SDK's tag is named, its phase's own confirmation, written into the PLAN before it runs.

### Completion (2026-11-10)

Items 1 to 9 are met; item 10 is open. The evidence is in the PLAN's P-records
([0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md)).
The commits are P0 `a6636fa`, P1 `ece13c4` (tagged `acpext/v0.3.0`), P2 `969b161`, P3 `36626fd` and P4 `9eb22bc`; P6
is the commit that adds this section.

1. **Met, P1.** `TestErrorDataSortedMembers` encodes all fourteen members in sorted order. The four new frames
   round-trip under both encoders, with unknown members refused (`TestFramesRoundTrip`). apidiff from `acpext/v0.2.0`
   reports compatible additions only.
2. **Met, P3 and P6.** `TestPromptQuotaData`'s -32603 data is byte-equal to `prompt-quota.error.json`'s canonical
   encoding. P6's loopback run of `gobble acp` on macOS, the Linux machine and the Windows laptop answered OpenCode Zen's
   429 `FreeUsageLimitError` with every member as this item states, `resetAt` within a second of the request plus
   5601 s.
3. **Met, P2.** `TestWireLimits`'s twelve subtests cover every row of D4, with `UserLimitError` and Kilo's
   message-only free usage giving none; `TestWireLongWaitIsNotAWindow` gives `rate` for a 5-hour wait. Gemini's `code` is
   `RESOURCE_EXHAUSTED` for a per-day and a per-minute 429 alike, so only `limit` tells them apart. D4 reads the period
   from the SDK's kind, not the code, and is unaffected.
4. **Met, P2, P3 and P6.**
   * `TestRetryNoticeCarriesServerWait` and `TestRetryNoticeServerMembers` cover the retry start's four members.
   * `TestAttemptsAfterRetries` and the `prompt-after-retries.error.json` frame cover `attempts`.
   * The `prompt-rate-over-cap.error.json` frame covers a wait over the cap.
   * In P6's run, case 3's start carried `retryAfterMs` 1000, `serverDirected`, `code` and `httpStatus` 429. Case 4's
     120 s wait sent no notice and answered -32603 with `retryable` true and `retryAfterMs` 120000.
5. **Met, P2 and P6.** On the tree before P2, `TestWireRejectedKey`'s 401 `MonthlyLimitError` case failed with
   "opencode-zen rejected the API key from auth.json: …", which is F9 observed. After D8 it is `llm.ErrQuota` and
   names no rejected key; `TestQuotaIsNotARejectedLogin` covers a stored login. P6's case 2 ran through `gobble acp`
   with the key in `auth.json`: `kind` `quota`, `limit` `monthly`, and no rejected key. gobble at `74d7f66` said the key
   was rejected.
6. **Met, P2 to P4.** `make preflight`'s `archtest` passed with `internal/archtest` unchanged; acpserver imports neither
   `llm/provider` nor the SDK.
7. **Met, every phase.** On scratch copies: five breaks at P1, eight at P2, ten at P3 and twelve at P4, each failing as
   its record quotes. P6's acceptance driver failed 20 of its 38 checks against gobble at `74d7f66`.
8. **Met, P4.** Q1 (a): nothing was built, as answered. Q2 (a): the quota mark keeps its reset time, shows it and
   lapses after it. By P4's deviation 1 that is the quota mark alone, since a rate limit makes no mark (F11). Q3 (a):
   `acpclient.PromptError`, the line session's limit line, and print mode's `errorData`.
9. **Met, P1 to P4 and P6.** `make preflight` and `make race` passed at each phase. The Linux machine and the Windows
   laptop ran acpext's tests, `go test ./...` and the race tests at P1 to P4, and P6's acceptance.
10. **Open.** P5 waits for the owner to name the go-llmprovider-sdk tag that carries #13's fields (Q4 (a)). The PLAN
    stays `in-progress` until P5's record meets this item (the PLAN's P6 deviation 1).

Items 11 and 12, which the amendment below adds, are open.

### Amendment (2026-11-10, proposed): two defects found by P6's acceptance

P6's loopback acceptance found two defects that predate this record and are outside its decision. On 2026-11-10 the
owner moved both into this record's scope, to be analysed and fixed: "add these to the current scope, they need to be
analyzed and remediated". Both were read at `9eb22bc` and probed with the binary built from it, each in a scratch home.

* **F15. A chat flag before another command is accepted and dropped.**
  * `--provider`, `--model`, `--thinking` and every other flag in `SharedFlags` (`internal/cli/flags.go:14`) belong
    to `chat`, the default command (`ChatCmd`, `internal/cli/chat.go:13-16`). `gobble acp` has `--login`, `--device`
    and `--mode` (`internal/cli/acp.go:20-24`), so `gobble acp --model …` is refused as an unknown flag.
  * Placed before a command, Kong accepts chat's flags and the command ignores them. In probes, each exits 0 and does
    only the command's work:
    * `gobble --model opencode-zen/x version`;
    * `gobble --print version`;
    * `gobble --thinking high config path`;
    * `gobble --model opencode-zen/x acp --help`, which shows acp's help.
  * After the command, the flag is refused: `gobble version --model x` exits 2 with `unknown flag --model`.
  * Only `ChatCmd.Run` calls `pickModel`. That fills `e.pick`, which `agentOptions` passes on as
    `acpserver.Override` (`internal/cli/agent.go:109`). Under `gobble acp`, `e.pick` is zero, so
    `gobble --model X acp` serves sessions that start on the settings' model with no word of the flag. P6 saw this:
    the session started on `opencode-zen`'s default model.
  * An ACP client sets the model with `session/set_config_option` (`configId` `model`). A root `--set` reaches
    `gobble acp`: `gobble --set defaultModel=x acp --help` exits 0. So the settings' `defaultProvider` and
    `defaultModel` are a working route today.
* **F16. `GOBBLE_HOME` does not spare the platform's directories.**
  * `appdirs.System` (`internal/appdirs/dirs.go:80-87`) calls `systemBase()` before `Resolve`, and returns its error.
    * On Windows, `systemBase` (`base_windows.go:16-30`) reads the RoamingAppData and LocalAppData known folders with
      `KF_FLAG_DEFAULT`, which fails when a folder is missing, then `os.UserHomeDir`.
    * Elsewhere it is `os.UserHomeDir` (`base_unix.go:10-16`), which fails with `HOME` unset.
  * `Resolve` itself needs no base under `GOBBLE_HOME`: `TestResolveGobbleHomeNeedsNoBase` says "GOBBLE_HOME needs no
    platform base, so it works where HOME is unusable". 0003-MADR keeps `GOBBLE_HOME` "for portable installs and
    tests". `System` undoes both promises.
  * On the Windows laptop, a profile without those folders failed with `appdirs: known folder RoamingAppData: The
    system cannot find the path specified.` On macOS, `GOBBLE_HOME` set and `HOME` unset make `gobble config path`
    exit 1 with `appdirs: home directory: $HOME is not defined`.
  * Every caller of `systemDirs` treats the failure as "no directory".
    * Under `gobble acp` that meant no `models.json`, so the providers' default hosts were dialled with the
      environment's keys (the PLAN's P6 record). `gobble auth login` failed with `auth.json: the config directory
      cannot be resolved`.
    * This "no directory" fallback is deliberate (0005-PLAN F3-1; the log file, magic-cli-remote 0157-MADR D8). It is
      not changed here.

#### D11. `GOBBLE_HOME` and the role variables need no platform directory (F16)

`System` reads the platform's base only when a role falls to the platform default. That is, when `GOBBLE_HOME` is
unset and at least one of `GOBBLE_CONFIG_DIR`, `GOBBLE_DATA_DIR`, `GOBBLE_STATE_DIR` and `GOBBLE_CACHE_DIR` is unset.
This holds on every OS. `Resolve`'s signature and its table (0003-MADR) are unchanged.

* With `GOBBLE_HOME` set, a Windows profile without its AppData folders and a Unix process without `HOME` both
  resolve.
* With neither set, the base is read as today, and its error is unchanged.
* gobble does not create the known folders (`KF_FLAG_CREATE`). They are the OS profile's to make, and the defect is
  the order of reads, not their absence (Q7).

#### D12. gobble acp chooses its sessions' model as chat does, and a misplaced chat flag is refused (F15)

As Q6 answers. Under Q6 (a):

* `gobble acp` gains `--provider`, `--model` and `--thinking`, as chat's: same names, help, completion and resolution
  (`pickModel`, Pi's rules over the offline model set).
* Like chat's, they win over every session's own, a loaded one included, through the existing `acpserver.Override`.
  A client can still change the model afterwards with `session/set_config_option`.
* A flag of chat's given with any other command is a usage error (exit 2), naming the flag and its command, not a
  value silently dropped.

#### Questions for the owner on the amendment (Q6, Q7)

* **Q6. `gobble acp` and the model flags (F15).**
  * **(a) Both: `gobble acp --provider/--model/--thinking` as chat's, winning over every session's own, and chat's flags
    refused before any other command.** *(recommended)* An ACP client's launcher, such as magic-cli-remote's, can pin
    the model on the command line, and no flag is ever silently dropped. Cost: `internal/cli` only. The flags'
    struct is shared with chat, and a test per command family.
  * (b) As (a), but the acp flags set only a new session's model, and a loaded session keeps its own. This needs a new
    `acpserver` option beside `Override`, since `Override` applies to every session.
  * (c) Refuse the misplaced flags only. `gobble acp`'s start model stays the settings' (`--set defaultModel=…`).
* **Q7. The platform base under `GOBBLE_HOME` (F16).**
  * **(a) D11: read the base only when a role needs it.** *(recommended)*
  * (b) As (a), and on Windows ask for the known folders with `KF_FLAG_CREATE`, so that a profile without them gets
    them. Not recommended: it writes the OS profile, which no role asked for.

#### Confirmation of the amendment

11. `appdirs.System` with `GOBBLE_HOME` set returns that directory for all four roles when the platform base
    fails. A test injects a failing base and sees it pass, and sees it fail against today's order. The Windows laptop
    runs P6's acceptance without the driver's AppData folders and passes. With neither variable set, the base's error
    is returned as today.
12. As Q6 answers. Under (a), `gobble acp --model P/M` starts a session on `P/M`, and `--thinking` sets its level.
    `gobble --model x version`, `gobble --print version` and `gobble --thinking high config path` each exit 2 naming
    the flag. Chat's own parsing (`gobble --model P/M -p hi`, `gobble -p hi --model P/M`) is unchanged. P6's acceptance
    sets the model by the flag in one case and passes on three hosts.

## Pros and Cons of the Options

### A. Text only

* Good, because no wire change and no tag.
* Bad, because a client still parses `llm: api status 429: llm: quota exhausted: llmprovider: …` for the code, the wait
  and the provider's words, and the limit's scope is not in the text at all for most codes.
* Bad, because a wait refused over the cap stays invisible as data.

### B. Structured `ErrorData` through the `llm` facade

* Good, because every member is additive and optional, and `error` is unchanged.
* Good, because acpserver stays on gobble's own types; the SDK is read in one place.
* Good, because what `v1.28.0` carries ships now, and D10 has a defined place for what #13 adds.
* Neutral, because `llm.APIError`, `llm.ProviderRetry` and `agent.Retry` grow.
* Bad, because two module releases are needed, in order: acpext, then gobble.

### C. Read `*llmprovider.APIError` in acpserver

* Good, because it is the shortest path, as #14 sketches it.
* Bad, because it breaks archtest rule 4, which the architecture record holds so that the SDK can be replaced behind
  `llm` (0004-MADR); a change to the rule needs its own record.

### D. Wait for #13, then B

* Good, because one release would carry the absolute instant and the per-day flag.
* Bad, because nothing reaches magic-cli-remote until another repository ships, though most of the gain needs no SDK
  change; the owner asked for both sides to move now.

### E. A structured notice instead

* Good, because a notice is the protocol's place for a message to the user, with a title.
* Bad, because only a client that offered `session.notices` gets it, and the error, which every client gets, would stay
  a string; magic-cli-remote reads the error (0243-MADR D7).

## More Information

### Not decided here

* A member for a window's length, and the `window` and `weekly` scopes in practice: with their first source (D10).
* ChatGPT's `plan_type` and Kilo's `metadata` on the wire (D10, last item).
* A remaining-quota reading from a successful reply's headers (#13 requirement 5).
* `acpclient.Retry`'s new members in gobble's CLI retry line: Q3 covers the failed turn, not the retry line.
* magic-cli-remote's own changes: its 0243 pair.

### Related records and issues

* [maccavelli/gobble-cli#14](https://github.com/maccavelli/gobble-cli/issues/14): the request this record answers.
* [maccavelli/go-llmprovider-sdk#13](https://github.com/maccavelli/go-llmprovider-sdk/issues/13): the SDK's part (D10).
* magic-cli-remote `docs/decisions/0243-MADR-show-exactly-which-usage-limit-an-agent-hit-and-when-it-lifts.md`: the
  consumer, its D3 (the scopes), D4 (the provider's words), D7 (reading gobble's data).
* [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md): D5 (tags and their
  order), A3 (sorted `ErrorData`).
* [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md): D4, provider text out of gobble's
  state reasons.
* [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md):
  D2 (`make check-libs` before a pin), D5 (the workspace refusal's words).
* [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md): D4, the provider
  retry as gobble's retry notice.
* [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md): D8, notices
  only to a client that offered them.
* [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): the package tiers and import rules.
* [0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md):
  the implementation.
