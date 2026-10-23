---
status: accepted
date: 2026-10-22
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Pin go-llmprovider-sdk v1.21.0 and acp-go-sdk2 v1.20.0, and hand back to them what gobble does that they now do

## Context and Problem Statement

The owner asked on 2026-10-22 to "pin the current SDK version, 1.20.0 and do an analysis of how this changes adoption
and implementation. i want to adopt as much from the SDK as possible to keep the codebase lean and optimized", and then
to "also assess the current version of acp-go-sdk2". Every fact below was read that day: from `go.mod`, from the two
SDKs' tags (cloned read-only into a scratch directory, `go doc -all` of each public package compared tag to tag), from
their release notes and decision records, and from gobble's code at `main` `ec5240f`.

**What gobble pins, and what is current:**

| Module | Pinned (`go.mod`) | Current | Releases between |
| :--- | :--- | :--- | :--- |
| `github.com/maccavelli/go-llmprovider-sdk` | `v1.17.0` | `v1.21.0` (2026-10-21) | `v1.18.0`, `v1.19.0`, `v1.20.0`, `v1.20.1` |
| `github.com/maccavelli/acp-go-sdk2` | `v1.16.3` | `v1.20.0` (2026-10-22) | `v1.17.0`, `v1.17.1`, `v1.18.0`, `v1.19.0` |

The owner named "1.20.0". go-llmprovider-sdk has a `v1.20.0` but it is not current; acp-go-sdk2's current release is
`v1.20.0`. Q1 below settles which go-llmprovider-sdk tag is pinned.

### go-llmprovider-sdk, v1.17.0 to v1.21.0

No exported name was removed and no signature changed. The additions are `llmprovider.WithStreamRetry` (`v1.19.0`) and,
in `wizard`, `ResolveCredential`, `Options.KiloDeviceKey`, `Options.SkipMethods` and `Options.NoEndpointPrompt`
(`v1.21.0`). Behaviour a consumer can see:

- `v1.18.0`: a typed-nil token source is refused with `ErrInvalidRequest` by every `New`, `catalog.List`, `ListRaw`
  and `LookupModel`; it panicked before. `List` with no token source matches `ErrInvalidRequest`.
- `v1.19.0`: `WithStreamRetry(p, policy, notify)`, "live text and a retry together". Its record, the SDK's
  `0051-MADR-opt-in-streaming-retry.md`, was written for gobble and names `llm/provider/provider.go` and `llm/llm.go`
  as the two lines to change.
- `v1.20.0`: a failure reported inside a 200 stream is retried at most once per call (it was up to `MaxAttempts`,
  each perhaps billed); a stream index below 0 or from 4096 ends the stream with `ErrIncomplete` (a negative one
  panicked); fragmented reasoning and Gemini text no longer cost time and memory growing with N² (540 MiB for 16,000
  four-byte fragments); concurrent `catalog.List` calls and model probes share one fetch.
- `v1.20.1`: requires Go 1.27.3 (GO-2026-6809 in `crypto/x509`); the SDK's `0055-MADR` says to take `v1.20.1` or
  later rather than `v1.20.0`. gobble's `go.mod` is already `go 1.27.3`.
- `v1.21.0`: the wizard's notices that report a service's error show its class and HTTP status, never its message.

The module's own requirements are identical from `v1.17.0` to `v1.21.0`.

### acp-go-sdk2, v1.16.3 to v1.20.0

The ACP schema is `1.25.0` in every release, so the protocol gobble speaks does not change. Nothing exported was removed
or changed in signature. Added: `Err()` and `Wait()` on `Connection`, `AgentSideConnection` and
`ClientSideConnection`, and `ErrNotificationQueueOverflow` (`v1.18.0`); the `agentproc` package, which starts an
agent as a child process tree and connects to it (`v1.19.0`, extended in `v1.20.0`). Behaviour a consumer can see:
the constructors panic on a nil argument, and peer faults log at Warn rather than Error (`v1.17.0`); a notification's
error after the peer's end of output also matches `io.EOF` (`v1.18.0`); the client records the `initialize` answer
on its reader and holds messages that come before it, and the read buffer is 128 KiB (`v1.20.0`). Its `go.mod` asks for
`toolchain go1.27.3` and `golang.org/x/sys v0.49.0`, both of which gobble already has.

### What gobble does that the SDKs now do, or did all along

- **Live streaming is held back by gobble's own wrapping.** `llm/provider/provider.go:202` wraps every provider in
  `llmprovider.WithRetry`, which buffers; `provider.go:249` therefore reports `NativeStreaming` false, and
  `llm/llm.go:46` says so. Every SDK provider streams natively (the two OpenCode providers best-effort). 0005-PLAN's
  F4-1b, live streaming, waits for exactly the release that `v1.19.0` is (`docs/README.md:108`, `:150-155`;
  `0005-PLAN-v1-feature-scope.md` decision 3). No gobble record mentions `v1.19.0` or later.
- **Provider retries are silent.** `WithRetryNotify` has existed since `v1.15.0` and is unused; 0005-PLAN waits "once
  the SDK offers a callback (issue 3)".
- **The ChatGPT sign-in.** `internal/cli/auth.go:217` refuses OpenAI's login as waiting for the SDK, but
  `auth.LoginChatGPTPlan` is in the pinned `v1.17.0` (`llmprovider/auth/oauth_plan.go:94`). 0025-PLAN P6 owns it and
  has not run.
- **The wizard.** gobble runs its own sign-in (`internal/cli/login.go`, `internal/cli/auth.go`) over the SDK's Grok
  and Kilo flows. 0025-PLAN's deviation 6 dropped the SDK's wizard for three reasons: it forced its own model menu, its
  Kilo device sign-in saved a session rather than a key, and it asked for an endpoint. `v1.21.0`'s `ResolveCredential`,
  `KiloDeviceKey` and `NoEndpointPrompt` answer all three; the SDK's `0056-MADR` was written from gobble's issue 5 and
  names gobble's `/login` as their caller.
- **The ACP connection's end.** `acpserver/serve.go:39-40` returns `agent.Close()` as soon as `Done()` closes, while
  handlers may still run, and never reports a read error. `acpclient/conn.go:173-191` waits on a channel of its own,
  and on a timeout returns without closing the connection. `Wait()` and `Err()` do both properly.
- **ACP constants written by hand.** `acpclient/conn.go:32` defines `authRequiredCode = -32000`, and
  `acpclient/sessions.go:52,54` compare with `-32600` and `-32602`; the SDK has `ErrorCodeAuthenticationRequired`,
  `ErrorCodeInvalidRequest` and `ErrorCodeInvalidParams`. Discriminator strings are written by hand at
  `acpserver/commands.go:531,597,728` and `acpserver/modes.go:85`. 0016-MADR's D4 and D5, still `proposed`, say to use
  the SDK's.
- **Smaller copies:** `gobble models --search` uses its own fuzzy match (`internal/cli/modelscmd.go:197-248`) while
  the ACP picker uses `catalog.Search`; `llm/catalog.Price` repeats `catalog.ModelCost.RatesFor`'s tier rule;
  `Usage.CacheWrite1hTokens` is not mapped; `HasLogin` and `HasKeyLogin` (`llm/provider/login.go:71,75`) hard-code Grok
  and Kilo where `Descriptor.AuthMethods` says it; `acpserver/binary_test.go:92-140` drives a child agent by hand
  where `agentproc` does it.

### What the SDKs still lack

Image input (no image `Item` through `v1.21.0`), tool-call argument deltas, an effort above `xhigh`, a generic
OpenAI-compatible provider, a per-request header option (gobble's `headerTransport` stays), an exported sentinel for the
SDK's empty answer (gobble matches its message), and offline detection (gobble's `ErrOffline` and probe stay). In
acp-go-sdk2: helpers for the update variants 0016-MADR D13 asked for, and a default client for fs, terminal and
elicitation (gobble's stubs stay). gobble keeps these until the SDKs have them.

## Decision Drivers

* The owner's direction: adopt as much of the SDKs as possible, to keep gobble lean.
* 0004-MADR-go-module-architecture.md: gobble waits for a library rather than copying it, and drops its copy when the
  library has the thing.
* Live text is the largest user-visible gain, and its absence blocks 0030-PLAN P5's capture of a resize mid-reply.
* No regression: a reply shown once, every gate on macOS, the Linux machine and the Windows laptop.
* Each adoption small enough to land, verify and revert on its own.

## Considered Options

* A. Pin the current releases only, as 0023-PLAN's amendments did.
* B. Pin the current releases, and adopt what they now do, phase by phase, leaving named items for later.
* C. Pin, and adopt live streaming alone.

## Decision Outcome

Chosen option: "B. Pin the current releases, and adopt what they now do, phase by phase", because it is the owner's
direction, removes the copies the SDKs make unnecessary, and turns on live text, while each phase lands and reverts on
its own.

- **D1. The pins.** go-llmprovider-sdk `v1.21.0` (Q1) and acp-go-sdk2 `v1.20.0`, each checked first by
  `make check-libs` (0019-PLAN P1) and landed with `go get` and `go mod tidy`. The behaviour changes above are taken as
  they are: gobble passes no typed-nil token source and no nil to an ACP constructor, and does not use the wizard
  until D7.
- **D2. Live streaming (0005-PLAN F4-1b).** `llm/provider` wraps each provider with `WithStreamRetry(p, policy,
  notify)` in place of `WithRetry`. `NativeStreaming` is reported from the wrapped provider, true when it is
  `Supported`; the OpenCode providers' `BestEffort` stays reported as false. Retry off still means no wrapper.
- **D3. A failure after the first delta ends the turn.** `WithStreamRetry` does not retry once text has passed. gobble
  does not either: a call that has yielded text or thinking is not sent again by the agent's offline or empty-reply
  retries, so no text is shown twice. The turn ends with the error, and the text already shown stays, followed by a
  line saying the reply was cut off. A failure before the first delta is retried as today.

  *Amended 2026-10-22 (0031-PLAN P2, deviation 1; the owner's answer "Keep the re-run; guard failures only"):* the
  guard is for a call that fails after showing text or thinking (offline, an empty-answer error, an overflow). A call
  that completes with thinking alone is still run again, its thinking shown again, as 0005-PLAN F5-1 decision 4 has
  it and as it was when replies were buffered.
- **D4. Provider retries are shown.** The `notify` of D2 reaches the turn as a retry of its own cause, the provider's,
  with its attempt, delay and error class, shown as the offline and empty-reply waits are. This answers 0005-PLAN's
  "once the SDK offers a callback (issue 3)".

  *Amended 2026-10-22 (0031-PLAN P2, deviation 2; the owner's answer "Cause is the class"):* the retry's cause
  is the failure's class (`unavailable`, `rate-limited`, `timeout`, `quota`, …), so the line reads "retrying:
  rate-limited (attempt 1 of 3)". 0005-PLAN's planned cause `provider` is superseded, and `_gobble/retry`'s causes
  are an open set.
- **D5. The ACP connection's end.** `acpserver.Serve` waits for the connection's handlers (`Wait`), bounded by a
  timeout since gobble's notification handlers do not read their context, before closing the agent, and returns
  `Err()` when the input failed other than by its end. `acpclient.Conn.Close` closes the connection after `Wait`, on a
  timeout as well. `acptest`'s cleanups wait the same way, so a leaked handler fails a test.
- **D6. ACP's constants.** 0016-MADR D4 and D5 are carried out: the SDK's error codes and discriminators replace the
  literals named above.
- **D7. The SDK's credential step.** `/login`, `gobble auth login` and the first-run sign-in use
  `wizard.ResolveCredential`, with `KiloDeviceKey` (Kilo's device token stored as a key), `NoEndpointPrompt`, and
  `SkipMethods` for the methods gobble does not offer, over a `wizard.Prompter` that draws in gobble's line session.
  gobble keeps its model picker and its `auth.json` store. The ChatGPT plan sign-in comes with it, which replaces
  0025-PLAN P6. Whether D7 is in this record is Q2.

  *Amended 2026-10-22 (0031-PLAN P4, deviation 3; the owner's answer "B: hold P4 for 3 SDK gaps"):* D7 waits for
  three go-llmprovider-sdk changes: a structured device-code hook carrying the verification URL, the code and its expiry; an option that starts no
  paste prompt, for a sign-in with no terminal; and no wait for Enter once a browser sign-in has finished. At `v1.21.0` a device code reaches the
  caller only inside a formatted notice (`wizard/auth.go:656`), with no expiry, so `gobble auth login --json`'s
  `device_code` event (0005-PLAN F4-3c decision 6) has no source; a browser sign-in always starts a paste prompt
  and, once the browser has finished, waits for Enter (`wizard/auth.go:715-754`). With them, every sign-in path
  goes through the wizard, and none is kept twice. The wizard's caller lives in `llm/provider`, behind a prompter
  interface of gobble's, since only that package imports go-llmprovider-sdk
  ([0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md) rule 4). The ChatGPT plan sign-in
  comes with it, with what it needs of gobble: the plan session's fields in `auth.json` and the store, and the
  consent to background use that 0025-PLAN P6 step 2 names. The three changes are go-llmprovider-sdk issues 6, 7 and
  8.
- **D8. The smaller copies go.** `gobble models --search` uses `catalog.Search`; `llm/catalog.Price` uses
  `ModelCost.RatesFor`; `CacheWrite1hTokens` is counted; `HasLogin` and `HasKeyLogin` read `Descriptor.AuthMethods`;
  `acpserver/binary_test.go` uses `agentproc`.

  *Amended 2026-10-22 (0031-PLAN P5, deviation 4; the owner's answer "Search only; the rest noted"):* of D8, the
  search and the binary test are carried out. `Price` stays gobble's: it lives in `llm/catalog`, called from
  `acpserver`, neither of which may import go-llmprovider-sdk (0004-MADR rule 4), and calling `RatesFor` through
  `llm/provider` takes more conversion code than the four-line tier loop it would replace. `CacheWrite1hTokens`
  waits for the SDK to give a rate to price it by: `ModelRates` has one cache-write rate, so counting the tokens
  would change no figure gobble shows. `HasLogin` and `HasKeyLogin` go with D7, where `SkipMethods` replaces them:
  the registry's `AuthMethods` lists methods gobble does not offer, so until then gobble's own list stays.
- **D9. Not now, each with its reason:**
  - native tool deferral (`Tool.Deferred`, `ToolLoadItem`): it changes the prompt cache's design, which
    0012-MADR and 0021-MADR own;
  - 0016-MADR D8–D10, ACP's stable notice and compaction variants for gobble's `_gobble/compaction`: a change to
    what gobble sends, for its own decision;
  - the listing's once-per-process guard and minute retry (`internal/cli/models.go`): `v1.20.0` makes concurrent
    listings safe, but what the guard also decides needs measuring first;
  - the update-variant helpers 0016-MADR D13 asks acp-go-sdk2 for: no issue was ever filed, and filing one is the
    owner's to do;
  - Ollama as a keyless choice, and what the SDKs lack (above).

**Q1, the go-llmprovider-sdk tag.** `v1.21.0` (recommended): a superset of `v1.20.x` with no breaking change, the Go
1.27.3 floor, and D7's entry points. `v1.20.1`: the same without D7. `v1.20.0`, as named: builds safely here only
because gobble's toolchain is already 1.27.3, and the SDK advises against it.

**Q2, the wizard.** In this record as D7 (recommended), or left to 0025-PLAN P6 and a later record.

**The owner's answers, 2026-10-22:** Q1 "v1.21.0"; Q2 "Yes, phase 4"; and 0031-PLAN's Q3, the order,
"0030 P5, then 0031, then 0029".

### Consequences

* Good, because replies arrive as they are written, and 0030-PLAN P5 can capture a resize mid-reply.
* Good, because gobble stops buffering, retrying and signing in where the SDKs do it, and loses the code that did.
* Good, because the ACP connection ends only after its handlers, and reports a broken input.
* Bad, because a stream that fails after its first delta is no longer retried; the user sees a cut-off reply and
  sends the prompt again. Before, a buffered retry hid the failure at the cost of waiting for the whole reply.
* Bad, because the wizard's prompts and wording become the SDK's, within what its `Prompter` lets gobble draw.
* Neutral, because peer faults on the ACP connection log at Warn, so on a terminal they read "Warning:", not "Error:".
* Neutral, because each ACP connection takes 64 KiB more for its read buffer.

### Confirmation

0031-PLAN's tests, each seen failing first; `make check-libs` for both SDKs at their new tags; on macOS, the Linux
machine and the Windows laptop, the tests, the race detector, `make preflight`, `make perf` and `make probe-conhost`;
the 0030-PLAN P5 captures, now with a resize mid-reply; and a live turn against a real provider by the owner, which
shows text arriving as it is written.

## Pros and Cons of the Options

### A. Pin the current releases only

* Good, because it is one small commit, as 0023-PLAN's amendment of 2026-10-19 was.
* Bad, because every copy listed above stays, and live streaming still waits, though its release has shipped.

### B. Pin, and adopt phase by phase

* Good, because it does what the owner asked, and each phase stands alone.
* Bad, because it is five phases of work before 0029-PLAN resumes.

### C. Pin, and adopt live streaming alone

* Good, because it takes the largest gain soonest.
* Bad, because the other copies stay, against the owner's direction.

## More Information

- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): wait for a library, never copy it.
- [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F4-1b, live streaming, which D2 and D3 carry out.
- [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md): D4 and
  D5, which D6 carries out; D8–D10 and D13, which D9 leaves.
- [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md):
  `make check-libs`.
- [0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md): the pins this record moves on.
- [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md): deviation 6, which dropped the wizard,
  and P6, the ChatGPT sign-in, which D7 replaces.
- [0030-MADR-resize-and-performance.md](0030-MADR-resize-and-performance.md): P5's capture of a resize mid-reply, which
  waits for D2.
- The SDKs' own records: go-llmprovider-sdk `0050`, `0051`, `0053`, `0055` and `0056-MADR`; acp-go-sdk2 `0024` and
  `0026`–`0030-MADR`, in their repositories.
- Found while reading, not decided here: `internal/proctree/tree_windows.go` assigns the Windows job object after
  the process starts, the race acp-go-sdk2 `v1.20.0` fixed in its own `agentproc` (a child leaked in 20 of 20 of its
  trials). It is a gobble defect for a record of its own.
