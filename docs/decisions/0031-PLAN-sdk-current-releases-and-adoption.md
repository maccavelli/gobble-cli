---
status: in-progress
date: 2026-10-22
associated-madr: "0031-MADR-sdk-current-releases-and-adoption.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement the current SDK releases, and hand back to them what gobble does that they now do

Associated MADR: [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md)

## Goal

gobble pins go-llmprovider-sdk `v1.21.0` (or the tag Q1 settles) and acp-go-sdk2 `v1.20.0`; replies stream live with
the SDK's retry; provider retries are shown; the ACP connection ends after its handlers; and gobble's copies of what the
SDKs do are gone (MADR D1–D8). What D9 leaves is named, not done.

## Scope

- **In:**
  - `go.mod`, `go.sum`;
  - `llm/llm.go`, `llm/doc.go`, `llm/provider` (`provider.go`, `login.go`), `llm/catalog` (`Price`);
  - `agent` (the turn's retries, a provider retry cause), `acpserver`, `acpclient`, `acpclient/acptest`;
  - `internal/cli`: the view (a cut-off reply, the provider's retry), `login.go`, `auth.go`, `modelscmd.go`, and the
    sign-in's prompter;
  - their tests;
  - records: this pair; dated notes in 0005-PLAN (F4-1b and the waits on the SDK), 0016-MADR (D4, D5), 0023-MADR and
    0023-PLAN (the pins), 0025-MADR and 0025-PLAN (deviation 6, P6); `docs/architecture.md`; `docs/README.md`.
- **Out:** everything in MADR D9; image input and the other things the SDKs lack; 0029-PLAN; the Windows job-object
  race in `internal/proctree`, for its own record.

## Order

0030-PLAN P5 first commits its deviation 6 and the captures it can make, and leaves its mid-reply capture to this
plan's P2 (a dated deviation in 0030-PLAN). Then P1 to P5 here, in order; then 0029-PLAN.

**Q3, the order.** As above (recommended); or this plan after 0029-PLAN, the mid-reply capture waiting until then.
The owner's answer, 2026-10-22: "0030 P5, then 0031, then 0029". The tag is `v1.21.0` (MADR Q1), and P4 is in
(MADR Q2). This plan stays `proposed` until the owner approves its execution.

## Implementation Steps

### P1 — the pins (D1)

1. `make check-libs` for go-llmprovider-sdk at the tag Q1 settles and for acp-go-sdk2 at `v1.20.0`: the scratch
   clone, `apidiff`, the libraries' tests and `govulncheck`. Each report's result goes in the execution record.
2. `go get github.com/maccavelli/go-llmprovider-sdk@<tag> github.com/maccavelli/acp-go-sdk2@v1.20.0`, `go mod tidy`.
   No code change is expected; one that is needed is a deviation.
3. Records: `docs/architecture.md`'s pin table; `docs/README.md`; a dated note in 0023-MADR's table and 0023-PLAN
   that the pins moved here.
4. Gates (below). Commit.

### P2 — live streaming, a cut-off reply, provider retries (D2–D4)

1. **Tests first,** each seen failing on the current code or a stub, on an `httptest` server streaming Server-Sent
   Events in the providers' own wire format (as `llm/provider/wire_test.go` does), never a real service:
   - the adapter reports `NativeStreaming` for a `Supported` provider and not for OpenCode;
   - a turn's first text delta reaches the agent's consumer before the server has sent its last event;
   - a failure before the first delta is retried, and the turn's text is shown once;
   - a failure after the first delta is not retried by the SDK or by the agent's offline and empty-reply retries:
     the server sees one request, the turn ends with the error, and the text is shown once;
   - a provider retry reaches the turn as a retry of the provider's cause, with its attempt, delay and error class;
   - the view shows a reply cut off by an error with the text already drawn and one line saying it was cut off.
2. **The code:** `WithStreamRetry` with `notify` in `llm/provider/provider.go`; `NativeStreaming` from the wrapper;
   a provider retry event in `llm` and a `RetryProvider` cause in `agent`, the notice carried from `notify` to the
   call through its context; the agent marks a call that has yielded text or thinking and does not resend it; the
   view's cut-off line; `llm/llm.go` and `llm/doc.go` say what is now true.
3. **0030-PLAN P5's capture of a resize mid-reply,** on macOS, the Linux machine and the Windows laptop, through its
   scratch tools, the stub streaming over two seconds: the narrowing lands between the reply's first and last bytes,
   and the check passes.
4. **Negatives,** one at a time on a scratch copy: `WithRetry` back (buffered); the resend after a delta allowed; the
   notice dropped; the cut-off line missing.
5. Records: 0005-PLAN's F4-1b and "issue 3" waits noted as done here; `docs/README.md`'s waits. Gates. Commit.

### P3 — the ACP connection's end, and its constants (D5, D6)

1. **Tests first:**
   - `acpclient.Conn.Close` does not return while a `SessionUpdate` handler still runs, and returns within its
     timeout when one never ends;
   - `acpserver.Serve` returns the input's error when the input fails, and nil at its end;
   - `acptest`'s cleanup fails a test whose handler outlives the connection.
2. **The code:** `Wait` and `Err` in `Serve`, `Conn.Close` and `acptest`, each bounded; the SDK's error codes and
   discriminators for the literals 0031-MADR names; `acpserver/binary_test.go` on `agentproc`.
3. **Negatives:** `Wait` dropped from `Close`; `Err` ignored in `Serve`; a literal code changed.
4. Records: 0016-MADR D4 and D5 noted as carried out here. Gates. Commit.

### P4 — the SDK's credential step (D7; only if Q2 keeps it here)

1. **Tests first,** every sign-in against an `httptest` server and a scripted `wizard.Prompter`; no test opens a
   browser, reaches a real sign-in, or writes outside a temporary `GOBBLE_HOME`:
   - `/login`, `gobble auth login` and the first-run offer each ask through gobble's prompter and save to
     `auth.json` in its shape, for a key, Grok's browser and device sign-ins, Kilo's device sign-in (saved as a key)
     and the ChatGPT plan sign-in;
   - no endpoint is asked; the methods gobble does not offer are not listed;
   - a refused key is asked again, and its notice carries the class and status, not the service's text;
   - gobble's model picker, not the wizard's, chooses the model.
2. **The code:** a `wizard.Prompter` over the line session; the three entry points on `ResolveCredential`; the parts
   of `internal/cli/login.go` and `auth.go` the wizard replaces, removed; OpenAI's sign-in no longer refused, with
   the terms the SDK's `auth` package names honoured.
3. **Negatives:** `KiloDeviceKey` off (a session saved); `NoEndpointPrompt` off (an endpoint asked); the store
   bypassed.
4. Records: 0025-MADR deviation 6 and 0025-PLAN P6 noted as superseded by this phase. Gates. Commit.

### P5 — the smaller copies, and the records (D8)

1. **Tests first:** `gobble models --search` finds what `catalog.Search` finds; `Price` equals `ModelCost.RatesFor`
   on tiered and flat models; a reply's usage counts `CacheWrite1hTokens`; `HasLogin` and `HasKeyLogin` follow the
   registry's `AuthMethods`.
2. **The code:** each replaced by the SDK's; `binary_test.go` already moved in P3.
3. **Records:** the stale waits on the SDK, each with a dated note (0005-PLAN's cache-write accounting, 0005-MADR's
   estimated usage, 0012-PLAN P7, 0004-MADR's `NativeStreaming` line); `docs/architecture.md`; `docs/README.md`;
   this plan complete.
4. Gates. Commit.

## Verification

- Every new test seen failing first, its failure quoted in the record; negatives on a scratch copy, one at a time.
- Each phase:
  - the pre-add check on its Go files;
  - on macOS, the Linux machine and the Windows laptop: the tests, the race detector, `make preflight` (on the
    copies, to `verify-build-metadata`), `make check-records markdownlint`, `make perf`, and `make probe-conhost` on
    the laptop.
- P1: `make check-libs`'s reports for both tags.
- P2: the mid-reply capture on all three hosts; the owner's live turn against a real provider.
- No test reaches a real service, a browser or a real sign-in.

## Rollout and Rollback

One commit per phase. Reverting P2 restores buffered replies and silent provider retries. Reverting P3 restores the
old connection end. Reverting P4 restores gobble's own sign-in, and `auth.json` stays readable by both, since its shape
does not change. Reverting P1 needs every later phase reverted first, since they use the new tags' names.

## Execution record

Approved by the owner's "Commit to main, then proceed", 2026-10-22, after the MADR's answers were recorded and
0030-PLAN P5 was committed (`3ec6761`).

### P1 execution record (2026-10-22)

1. **`make check-libs`** (`scripts/check_libs.py --lib … --tag …`), each on a scratch clone:
   - go-llmprovider-sdk, pinned `v1.17.0`, checking `v1.21.0`: get and tidy, version, modules, build, vet for linux,
     darwin and windows, test, api, govulncheck and advisories all `ok`; "0 failing checks".
   - acp-go-sdk2, pinned `v1.16.3`, checking `v1.20.0`: the same twelve checks `ok`; "0 failing checks".
2. **The pins:** `go get github.com/maccavelli/go-llmprovider-sdk@v1.21.0 github.com/maccavelli/acp-go-sdk2@v1.20.0`,
   `go mod tidy`. `go.mod`: the two lines (`v1.17.0` → `v1.21.0`, `v1.16.3` → `v1.20.0`); `go.sum`: those modules'
   four lines; no other module moved. No code changed: `go build ./...` and the tests passed as they were.
3. **Records:** `docs/architecture.md`'s pin table; `README.md`'s stack; dated notes in 0023-MADR (A1), 0023-PLAN
   (its "not done here") and `docs/README.md`'s 0005-PLAN entry, that the pins moved here and F4-1b is P2.
4. **Gates:**
   - The pre-add check on every tracked Go file: `577 file(s) clean (gofmt, golint, govulncheck)`.
   - macOS: `make preflight`, `preflight passed`; the tests and the race detector, 36 packages ok each.
   - The Linux machine and the Windows laptop, on a copy of the tree: the tests and the race detector, 36 packages
     ok on each; `make preflight` reached `apidiff` (golangci-lint `0 issues.`, govulncheck "No vulnerabilities
     found.") and stopped at `verify-build-metadata`, as the copies are not git checkouts.
   - `make check-records markdownlint`: exit 0 on each host.
   - The Windows laptop, `make probe-conhost`: `--- PASS: TestConhost`.
   - `make perf`, each within three times its baseline: macOS ReplyRender 32.7 µs, Reprint 9.67 ms,
     LineSessionStart 924 µs, KeyDraw 1.52 µs; the Linux machine 95.8 µs, 23.1 ms, 2.64 ms, 3.98 µs; the Windows
     laptop 29.8 µs, 7.81 ms, 4.99 ms, 1.09 µs.

**P1 complete, 2026-10-22.**

### P2 execution record (2026-10-22)

**A correction to step 1, 2026-10-22.** Step 1 says the tests stream "as `llm/provider/wire_test.go` does". It does
not: its canned wire answers JSON, which is all a buffered provider asks for. A probe on a scratch copy, with
`WithStreamRetry` in place of `WithRetry`, failed eleven `llm/provider` tests: ten wire tests with "stream ended
before" each wire's last event (`response.completed`, `message_stop`, `interaction.completed`, `[DONE]`), and
`TestNewReportsRetryCapabilities`, which asserts the buffering this phase removes. The agent's and the CLI's tests
passed unchanged. So the canned wire now answers each request in the form it asks for: JSON when it is not
streamed, and, when it is, the same reply as the wire's Server-Sent Events (`llm/provider/wire_sse_test.go`), the
empty-answer and refusal cases included. The converted tests pass on the buffered code before any change, and pass
streamed. There was no choice for the owner here: no route but the tests' conversion keeps the tree correct.

**Deviation 1, 2026-10-22, decided by the owner** ("Keep the re-run; guard failures only"). The agent's guard, as
MADR D3 first worded it ("a call that has yielded text or thinking is not sent again by the agent's offline or
empty-reply retries"), failed four cases of `TestEmptyReplyRunsAgain` (`agent/retry_test.go:201`, `:225`, `:238`,
`:254`): a reply that completes with thinking alone is empty by 0005-PLAN F5-1 decision 4, and is run again. The
guard now applies to a call that fails after showing text or thinking; a completed thinking-only reply is run again
as before. 0031-MADR D3 is amended; F5-1 and its tests are unchanged.

**Deviation 2, 2026-10-22, decided by the owner** ("Cause is the class"). 0005-PLAN F5-1 decision 5 planned the
SDK's retries to reach a client as `_gobble/retry` with `cause` `provider`; step 2 built them with the failure's
class as the cause, which the view's existing line shows ("retrying: rate-limited (attempt 1 of 3)"). The class it
is: 0031-MADR D4 is amended, 0005-PLAN's line is annotated, and the comments in `agent/event.go` and
`acpclient/update.go` that listed two causes now say the set is open.

1. **Tests first,** each seen failing on the buffered code, with `llm.ProviderRetry` added as a stub nothing
   produced:
   - `llm/provider/stream_test.go`, on canned wires for OpenAI, Claude, Gemini and Together, one per wire:
     - `TestStreamIsLive`: the wire holds the rest of the reply until the first text delta reaches the caller, or
       2 s pass. Failed on every wire: `stream: … invalid character 'e' looking for beginning of value (streamed
       false)`, the buffered provider asking for no stream.
     - `TestFailureAfterFirstDeltaNotRetried`: the stream cut after its first delta. Failed with `text ""`.
     - `TestProviderRetryReported`: a 503, then the stream. Failed as the others did.
     - `TestNewReportsRetryCapabilities`, turned round: `openai: native streaming false, want true`, and the same
       for Claude; OpenCode's best-effort stays false.
   - `agent/retry_test.go`:
     - `TestNoResendAfterShown/offline_after_text`: the half-shown text was sent again after an offline wait,
       `after 2 requests`;
     - `TestNoResendAfterShown/empty_after_thinking`: an empty-answer error after thinking, sent again;
     - `TestProviderRetryShown`: `retries []`.
   - `internal/cli/cutoff_test.go`, `TestCutOffReply`: `mid-reply` failed, `the cut-off line shown false, want
     true`; `before any reply` and `after a tool ran` passed, and stay as guards.
2. **The code:**
   - `llm/provider/provider.go`: `WithStreamRetry(p, policyOf(r), notice)`. `notice` hands each `RetryNotice` to a
     hook the call puts in its context; the hook yields `llm.ProviderRetry` (attempt, of how many, the wait, and
     `FailureKind`'s class) before the call's events, and a caller that stops there cancels the wait.
     `NativeStreaming` comes from the wrapped provider.
   - `agent/agent.go`: the reply marks `shown` once text or thinking is yielded. A call that fails after that ends
     the turn with its error and is not sent again by the offline, empty-reply or overflow resends (deviation 1). A
     `ProviderRetry` is a `Retry` start of its class (deviation 2), ended when the call's events begin or it ends.
   - `internal/cli/view.go` and `line.go`: the view marks a reply or thinking drawn last, cleared by a tool call
     or a plan; when the prompt fails, `cutOff` writes "(the reply was cut off)", dimmed, on its own line before the
     status line and the error, and keeps it for a reprint.
   - `llm/doc.go`, `llm/llm.go`: what is now true of the stream.
3. **The capture of a resize mid-reply** (0030-PLAN P5's, deviation 7 there), with 0030-PLAN P5's scratch tools,
   gobble built from this tree, the stub streaming over 2.2 s:

   | Host | Narrowing | Check | Narrowed at byte, within the reply's |
   |---|---|---|---|
   | macOS | re-wraps, cursor down | ok: 1 reply of 32 rows | 415, of 365–2024 |
   | macOS | re-wraps, cursor kept | ok | 421, of 397–2055 |
   | macOS | cuts | ok | 403, of 379–2037 |
   | The Linux machine | re-wraps, cursor down | ok | 800, of 776–2725 |
   | The Linux machine | re-wraps, cursor kept | ok | 829, of 805–2768 |
   | The Linux machine | cuts | ok | 844, of 794–2757 |
   | The Windows laptop | Windows' own | ok | 764, of 740–3393 |

   The mid-prompt captures were run again with streaming, and all seven passed. The timing probe of 0030-PLAN P5's
   deviation 7, which then saw the whole reply 12 ms after Enter, now saw its first text at 78 ms and its last at
   2.224 s, with the stub's stream running 0.004–2.215 s.
4. **Negatives,** one at a time on a scratch copy:
   - **The buffering `WithRetry` back:** `TestNewReportsRetryCapabilities` (`openai: native streaming false, want
     true`) and `TestStreamIsLive` (`streamed false`).
   - **The provider's notices dropped:** `TestProviderRetryReported`, `retries [], want one: attempt 1 of 3,
     unavailable, with its wait`.
   - **A resend allowed after text was shown:** `TestNoResendAfterShown`, `after 2 requests`.
   - **The provider's retry not shown by the agent:** `TestProviderRetryShown`, an end with no start.
   - **The cut-off line missing:** `TestCutOffReply`, `the cut-off line shown false, want true`.
   - **A tool not ending the reply:** `TestCutOffReply`, `the cut-off line shown true, want false`.

   Six mutations, six failures; the restored copy passed.
5. **Records:** 0031-MADR D3 and D4 amended (deviations 1 and 2); dated notes in 0005-PLAN (decision 3, F4-1b;
   F5-1 decision 5, issue 3), in `docs/README.md`'s 0005 entry, and in 0030-PLAN P5's deviation 7.

**Found by the gates and fixed in the phase:** `perfsprint`'s string concatenation in a loop, twice, and
`staticcheck`'s QF1001, all in the new tests.

**Gates:**

- The pre-add check on the seventeen changed `.go` files: `17 file(s) clean (gofmt, golint, govulncheck)`.
- macOS: `make preflight`, `preflight passed`; the tests and the race detector, 36 packages ok each.
- The Linux machine and the Windows laptop, on a copy of the tree: the tests and the race detector, 36 packages ok on
  each; `make preflight` reached `apidiff` (golangci-lint `0 issues.`, govulncheck "No vulnerabilities found.") and
  stopped at `verify-build-metadata`, as the copies are not git checkouts.
- `make check-records markdownlint`: exit 0 on each host.
- The Windows laptop, `make probe-conhost`: `--- PASS: TestConhost`.
- `make perf`, each within three times its baseline: macOS ReplyRender 32.9 µs, Reprint 9.39 ms, LineSessionStart
  827 µs, KeyDraw 1.54 µs; the Linux machine 123 µs, 28.4 ms, 3.24 ms, 5.09 µs; the Windows laptop 29.5 µs, 7.73 ms,
  5.13 ms, 1.10 µs.

**Not done, for the owner:** a live turn against a real provider, which the plan's verification asks the owner to
take; the owner's look at 0030-PLAN P5 is also still to come. P2 is committed with the rest of its steps done.
