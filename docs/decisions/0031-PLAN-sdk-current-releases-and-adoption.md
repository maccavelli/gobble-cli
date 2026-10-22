---
status: proposed
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
