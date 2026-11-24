---
status: accepted
date: 2026-11-24
decision-makers: repository owner
consulted: "the owner's request of 2026-11-24 and the owner's answers to Q1 to Q5 and Q7 of 2026-11-24, with Q1 re-answered on new evidence, relayed by the coordinator (see Owner rulings); go-llmprovider-sdk's docs/reports/0091-REPORT-responses-api-across-providers.md, its addition of 2026-11-24 (ideas G1, G8, H1, H2, H4); the continuation research of 2026-11-24 (continuation-synthesis, continuation-opencode on opencode PR #60674, continuation-qwen-code; session scratch, not committed); two independent reviews of the earlier drafts, each finding re-read in the code before it was written in; this repository at 97fa1f6, read only, with one probe test run on a scratch clone; go-llmprovider-sdk at its tag v1.32.0 (gobble's pin, go.mod:16), read only; the read-only pi checkout at 1402973a9"
informed: go-llmprovider-sdk (no SDK change is needed; H4 is not required); magic-cli-remote, which shows gobble's ACP replay, retry notices and errors
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Keep the Part of a Reply a Failure Cut, Retry an Accidental Cut, and Log One Stream-Health Record per Model Call

Associated PLAN: `0070-PLAN-keep-a-failed-partial-reply-and-log-stream-health-per-call.md` (a draft beside this one, `proposed`).

*(2026-11-24, first draft. Second draft, after an independent review: F14 corrected (the log hides a key ending in
`token`, not `tokens`); F4 corrected and G33 decided; replaced attempts, a thinking-only partial, the SDK's unseen
resends, a buffered reply, a request refused before the SDK and two Responses cuts added; tests named with their seams;
pi's behaviour stated (F16); records cited by full filename. Third draft, after a review of the second: G33's
mechanism; pi's retry patterns; Q3's labels; Q5's effect on D15, T2 and T19; T4, T15 and T21 made able to fail.)*

*(2026-11-24, fourth draft, after the owner's answers to Q1 to Q5: Q1 "Auto-continue, with limits (Recommended)"; Q2
"pi's 'Error: …' line (Recommended)"; Q3 "Yes, keep each one (Recommended)"; Q4 "pi's way: always write it
(Recommended)"; Q5 "info if failed/retried, else debug (Recommended)". New questions Q6 and Q7.)*

*(2026-11-24, amendment: the owner answered Q7, "Retry notice live (Recommended)"; C12 decided.)*

*(2026-11-24, fifth draft: the owner re-answered Q1 on new evidence, "Auto-retry, with limits (Recommended)". The
earlier answer, "Auto-continue, with limits (Recommended)", is superseded by this re-answer. The evidence (the
continuation research of 2026-11-24): opencode removed its continuation in PR #60674, merged 2026-11-10, because "the
model often stitched a broken answer together", and now resends the same request, dropping a call still streaming at
the cut; qwen-code's hosted harness resends rather than continues, since "a restart and a perfect continuation are
indistinguishable to the consumer"; codex and pi also retry the whole request with the cut part left out. Design B of
that research replaces D12 (b) to (c), rows C6 to C11 and D7's replay exception: one case in `next()`, reusing the rerun
path; `reply.started` set on any event; the setting `retry.retryAfterCut`. Q7's notice cause becomes `cut`, and its
hidden "Continue…" entry is gone. C4's premise is corrected (F20). D13 now amends
`0031-MADR-sdk-current-releases-and-adoption.md` D3 to "a shown-then-cut call is replaced by a rerun". The
pi-compatibility "Bad" bullet is removed. Q6 is moot: a half-written call is thrown away with its attempt, and no tool
ran (`agent/agent.go:738-744`); C5 is decided so. No row is open, so the record is `accepted`.)*

## Context and Problem Statement

Two ideas from the upstream comparison (go-llmprovider-sdk `docs/reports/0091-REPORT-responses-api-across-providers.md`,
addition of 2026-11-24) are gobble's:

* **G1.** When a model call fails after it has streamed text or thinking to the user, gobble ends the turn with the
  error and writes nothing of what was shown. The session file, and every reload of it, loses the text the user saw.
* **G8.** gobble has no per-call view of stream health: how long the first event took, the longest silence, how many
  events, which retries and why, whether anything was delivered, and what the cache did.

This record decides both, and, by the owner's answer to Q1, retries a call an accidental failure cut.

### Facts read at 97fa1f6

* **F1. "Shown" is the agent's `reply.shown`.** It is set by a non-empty `llm.TextDelta` or `llm.ThinkingDelta`, each
  emitted to the turn's consumer as it arrives (`agent/agent.go:720-737`; the field, `:414-417`). A tool call is never
  shown while its call streams: the adapter yields no `llm.ToolCallStart` or `llm.ToolCallDelta`
  (`llm/provider/provider.go:424-446`), though `llm` declares them (`llm/event.go:51-65`), and the agent shows a call
  only at `ToolStart`, after the stream ends (`agent/agent.go:846`).
* **F2. A failure after something was shown ends the turn with the error, and no message.** `next` does not resend
  such a call (`agent/agent.go:514-517`, `0031-MADR-sdk-current-releases-and-adoption.md` D3 as amended) and yields the
  error (`:571-575`); `run` returns on `!ok` (`:328-331`), so no `agent.Message` and no `agent.End` follow. The writer
  writes an assistant entry only on `agent.Message` (`acpserver/agent.go:1048-1050`). On the error the prompt flushes
  the coalescer, sets the live history from the file (`replayFile`, `:633-638`,
  `0037-MADR-hardening-integrity-and-performance.md` D15) and answers the JSON-RPC error (`:823-831`;
  `acpserver/prompt.go:272-278`).
* **F3. Probe (scratch clone of 97fa1f6, not the tree).** A test that scripts `TextDelta{"half a reply"}` then an
  HTTP 400 failure, prompts once, and reads `_gobble/get_messages` and `_gobble/get_last_assistant_text` logged
  `PROBE roles=["user"] last_assistant_text=<nil>`: the shown text is not in the session.
* **F4. A cancel during a call keeps its partial today; a cancel during the empty-reply wait does not.** A call
  cancelled while it streams returns its reply with no error (`agent/agent.go:699-703`); `next` returns it
  (`:511-513`); `run` emits it as `agent.Message` and ends `StopCancelled` with cancelled results for its calls
  (`:332-340`). Its stop reason is empty, which `piStop` writes as pi's `"aborted"` (`acpserver/history.go:243-253`); the
  live copy is marked failed by `stamped` (`acpserver/choice.go:243-258`), which sets each assistant message's whole
  `Origin` (`:250-251`), and a reload marks it failed from `"aborted"` or `"error"` (`acpserver/history.go:357-358`), so
  it is never sent to a model (`llm/provider/replay.go:45-56`). A cancel during the wait before an empty-reply rerun
  (`agent/agent.go:558-563`) goes through `cutShort`, which ends the turn `StopCancelled` with no message
  (`:589-592`). The cancel path ends with `End` and never calls `replayFile` (`acpserver/agent.go:845-848`).
* **F5. The file already has the failed shape.** `session.Message` carries pi's `stopReason` and `errorMessage`
  (`session/session.go:156-157`); `piError` is `"error"` (`acpserver/history.go:238-241`), read as failed on load, and
  the context estimate ignores the usage of an `"aborted"` or `"error"` message (`compaction/estimate.go:338-344`).
* **F6. What a load shows.** `session/load` replays one `agent_message_chunk` per assistant message with its text, none
  when it has no text (`acpserver/history.go:502-505`), and each tool call in its final state, a call with no result as
  failed, "no result was recorded" (`:506-555`). Thinking is not replayed. An `agent_message_chunk` is the model's text
  to every ACP client.
* **F7. The CLI says the reply was cut off, live only.** On a failed prompt it draws "(the reply was cut off)" after
  the drawn reply (`internal/cli/line.go:389-390`; `internal/cli/view.go:384-392`). Headless JSON carries the shown text
  and the error; headless text prints only the error (`internal/cli/print.go:206-227`).
* **F8. The turn's own reruns and waits.** A completed thinking-only reply is run again twice (`agent/agent.go:439`,
  `:548-567`: a `Retry` start, a wait, `cutShort` on a cancel or abort, then `continue`; `0031-MADR-sdk-current-releases-and-adoption.md`
  D3 as amended, "Keep the re-run; guard failures only"); when they run out the turn fails with `errNoUsableReply`
  (`:568-569`); an aborted wait gives `errEmpty` (`:565-567`). An offline failure waits for the host by `waitOnline`
  and resends (`:533-547`). An overflow before anything is shown compacts once and resends (`:518-531`); the shown
  guard is tested first (`:514`). In each rerun or resend the replaced attempt's reply is dropped.
* **F9. Length cuts and refusals are not errors at the adapter.** A `FinishLength` reply is `Done{StopMaxTokens}`
  (`llm/provider/provider.go:500-510`), which the turn ends `max_tokens` (`agent/agent.go:365-367`); a refused answer or
  a content filter is `Done{StopRefusal}` (`provider.go:406-408`, `:506-507`). On `v1.32.0` a Responses cut is still
  `ErrIncomplete`; `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` D5 to D7 (accepted, not built) turn
  it into a `max_tokens` stop with a cause, and run a cut answer's complete calls.
* **F10. The SDK never retries after an event.** `WithStreamRetry` retries only before the first event; after it, "a
  failure ends the stream with that error, unchanged, and no EventDone" (go-llmprovider-sdk
  `llmprovider/retry.go:215-220` at `v1.32.0`). Each retry it waits for is reported through the notify gobble installs
  (`llm/provider/provider.go:260`, `:266-275`), including the once-only resend after a reply (`after`, the SDK's
  `llmprovider/retry.go:165-200`). Over a provider that is not a `Streamer`, `WithStreamRetry` is `WithRetryNotify`
  (`:228-233`), which buffers the whole reply.
* **F11. The SDK also resends inside one send, unseen.** One request may be sent up to `MaxSends`, 7, times for a
  refused reasoning setting or a renewed credential, before any event (go-llmprovider-sdk
  `llmprovider/internal/wire/retry.go:10-33`, `sse.go:138-151` at `v1.32.0`); no notice is given.
* **F12. Telemetry today.** `telemetry` is a placeholder (`telemetry/doc.go:1-5`); `--stats` adds a per-turn time to
  first token measured by the client (`internal/cli/render/status.go:40-49`); the SDK logs nothing per request.
* **F13. The adapter sees every call.** `adapter.Stream` and `attempt` (`llm/provider/provider.go:352-452`) see the
  request's conversion (`toRequest`, which can fail before the SDK, `:366-369`), each SDK event, each retry notice
  (`:386-392`), the mapped error and its kind (`:410-419`), the response's usage and warnings (`:438-445`), and a resend
  without thinking after a signature refusal (`:355-357`). Titles and automatic compactions stream through the same
  kind of adapter, marked background, and not through `agent.Run`. `Options` carries `Logger` and `SessionID`
  (`provider.go:49`, `:56-59`). An in-memory transport is the existing test seam (`llm/provider/stream_test.go:34`).
* **F14. The log redacts by key name.** Every handler hides the value of a key whose lower-case form is
  `authorization` or ends in `key`, `token`, `secret` or `password` (`internal/logging/redact.go:13`, `:25-36`). The file
  is JSON at the chosen level, `info` by default (`internal/logging/logging.go:59`; `internal/cli/root.go:21`); stderr
  shows warnings and above only (`logging.go:66-71`).
* **F15. Two records each name a per-call debug line, unbuilt.**
  `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` D11 (R31) and
  `0066-MADR-set-each-session-model-up-from-discovered-limits.md` D8 (S21); neither line exists.
* **F16. pi.** pi's agent loop ends the turn on an `"error"` or `"aborted"` message and keeps it whole, its tool calls
  included, even when empty (`<pi>/packages/agent/src/agent-loop.ts:244-256`); pi retries a failed call, whatever was
  shown, and keeps each failed attempt in the file, hidden from the model (`<pi>/packages/coding-agent/src/core/agent-session.ts:1241-1257`,
  `:3792-3822`); pi never reruns a thinking-only reply; pi draws an errored message's error as "Error: …"
  (`<pi>/packages/coding-agent/src/modes/interactive/components/assistant-message.ts:198`).
* **F17. Error kinds that mark an accidental cut.** A server error reported inside the stream (`overloaded_error`,
  `api_error`, `server_error`) is the SDK's `ErrProviderUnavailable` (go-llmprovider-sdk `llmprovider/api_error.go:532-533`
  at `v1.32.0`), as are a failed read of a reply and the idle timeout; gobble maps it to `llm.ErrUnavailable`, and
  `ErrOffline` to `llm.ErrOffline` (`llm/provider/provider.go:748-752`; kinds `unavailable` and `offline`,
  `llm/kind.go:29-42`). A rate limit is `rate_limited`, an overflow `context_overflow`; a refusal or filter is not an
  error (F9). `timeout` is `context.DeadlineExceeded`, a caller's deadline.
* **F18. The SDK emits no event for a tool call in progress.** Its stream has text deltas, reasoning deltas, items and
  done only (go-llmprovider-sdk `llmprovider/stream.go:15-22`); a function call arrives only whole, as an item, and
  gobble runs calls only after the stream ends (`agent/agent.go:738-744`, `:342-347`). A call half-written at a cut
  never reaches gobble, and no tool has run.
* **F19. The retry settings.** `retry.enabled`, `retry.maxRetries`, `retry.baseDelayMs`, `retry.maxDelayMs` and
  `retry.offlineWaitMs` are session settings (`internal/config/keys.go:115-124`); `llm.Retry.Off` "turns all of them
  off" (`llm/llm.go:29-31`), and `_gobble/set_auto_retry` sets it. The settings schema is pinned by a golden
  (`internal/config/testdata/settings.schema.json`).
* **F20. "Nothing shown" is not "no event".** An encrypted-only reasoning item (a `ThinkingDelta` with provider data
  and no text) and a released complete call are events that set no `shown` (`agent/agent.go:722`, `:734`, `:738-744`),
  and the SDK retries nothing after any event (F10). A cut after such events is neither retried by the SDK nor, today,
  by gobble: the turn ends.
* **F21. Upstream, on a cut after output** (the continuation research of 2026-11-24). opencode resent the cut reply with
  "Continue from where you left off…" until PR #60674 (merged 2026-11-10) replaced it with a resend of the same request,
  citing stitched answers, dropping a call still streaming at the cut and keeping nothing of the attempt when no local
  tool ran; qwen-code's hosted harness resends the original request; codex rebuilds the request from completed items
  only; pi reruns and hides the failed attempt (F16).

## Decision Drivers

* What the user saw is what the session keeps: a reload shows it (F3).
* A kept partial is never replayed to a model as the model's own turn (D7, no exception), as a cancelled one is not
  (`0025-MADR-provider-choice-and-auth.md` D2); a compaction summary reads it as quoted history, as it reads a cancelled
  one (D9). The live session's next request is the request a reload sends
  (`0037-MADR-hardening-integrity-and-performance.md` D15).
* An accidental cut is retried, the same request, within the owner's limits; nothing else adds a billed call.
* The session file stays pi's format (Q4).
* One record per model call, for every call kind, with no content and no secrets.
* The per-call debug lines of `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` and
  `0066-MADR-set-each-session-model-up-from-discovered-limits.md` are fields of that record.
* No SDK change.

## Considered Options

* **A. Keep failed calls pi's way from gobble's own signals, retry accidental cuts in the agent, and log one record per
  call in the provider adapter.**
* **B. Wait for the SDK to export its delivery state (H4) first.**
* **C. Measure in the agent instead of the adapter.**
* **D. Keep stream health in a file of its own, or in `--stats`.**
* **E. Change nothing.**

## Decision Outcome

Chosen option: "A", because the agent knows what it showed and whether any event arrived (F1, F20), the error's kind
says whether a cut was accidental (F17), and the adapter sees every call (F13), so nothing waits for the SDK. B tells
gobble nothing `shown`, `started` and the kind do not. C misses titles and compaction summaries and the adapter's
request and response values. D repeats the log's rotation, level and redaction, and `--stats` never sees a call. E
leaves F3.

### Owner rulings

*(2026-11-24, relayed by the coordinator; the labels are exact.)*

* **Q1, "Auto-retry, with limits (Recommended)"** (re-answered on the continuation research, F21). As offered: resend
  the same request, up to 3 in a row, for accidental cuts only (kind `unavailable` or `offline`, after any event of the
  call, not only after something shown), with a setting to turn it off and a `cut` retry notice live; the cut part
  stays visible and is saved as a failed attempt (D4, never sent to the model: D7 with no exception). Accepted costs:
  the reply's output is billed again per retry, and live it starts over after the notice. All in gobble, no SDK work.
  Decides G30, D12 and C1 to C24. ~~"Auto-continue, with limits (Recommended)"~~, the earlier answer (opencode's old
  continuation), is superseded by this re-answer.
* **Q2, "pi's 'Error: …' line (Recommended)".** Decides G24 and G38 (D8).
* **Q3, "Yes, keep each one (Recommended)".** Decides G7, G34 and G35 (D4), and so a retried cut attempt that showed
  something (D12).
* **Q4, "pi's way: always write it (Recommended)".** Decides G3, G4 and G37 (D2, D3).
* **Q5, "info if failed/retried, else debug (Recommended)"**, the option offered as "`info` for a call that failed, was
  cancelled or was retried; `debug` for the rest". Decides T23 (D15); D16, T2 and T19 change as that answer said.
* **Q7, "Retry notice live (Recommended)".** Decides C12 and D12 (d). Under Q1's re-answer the notice's cause is `cut`
  (as the re-answer offered), and the hidden "Continue…" entry it described no longer exists.
* **Q6** is moot under Q1's re-answer: a call half-written at a cut never reached gobble, is thrown away with its
  attempt, and no tool ran (F18). C5 is decided so.

### The decisions: G1, keeping what a call produced

* **D1. What "shown" means.** A call has shown something when the agent emitted a non-empty text or thinking delta of
  it, exactly `reply.shown` today (F1), whitespace included, and whatever a client's view mode hides
  (`0056-MADR-curate-the-terminal-session-view-with-view-modes.md`). A tool call is never shown while it streams.
* **D2. The call a failed turn ends on is always written (Q4).** When a turn ends with an error, the agent first emits
  the call it ends on as an `agent.Message` with `Err` set, empty or not: the shown guard (`agent/agent.go:514-517`),
  the reruns running out (`errNoUsableReply`, `:568-569`), an aborted wait (`errEmpty`, `:565-567`), a failure with
  nothing shown, and an accidental cut that is not retried (the cap reached, or the setting off; D12). The writer writes
  it with `stopReason` `"error"` and `errorMessage` the error's text (F5); the live history is set from the file
  (`replayFile`), as on every failure. A cancel during the empty-reply wait after a shown attempt is not an error: the
  attempt is emitted with no `Err` and its stop reason cleared, appended to the turn's messages, and ends
  `StopCancelled`, so it is written `"aborted"` and `stamped` marks it failed live, as a reload reads it (F4).
* **D3. What it holds (Q4).** Its thinking (closed blocks with their provider data, then any open text), its text, its
  complete tool calls (not run, no result), and the usage the call reported. A load shows each kept call as failed, "no
  result was recorded" (F6). This is pi's shape (F16).
* **D4. Replaced attempts that showed something are each kept (Q3).** An attempt the turn replaces after it showed
  something (a thinking-only reply run again, G7; a rerun that then fails before any event, G34; a resend after an
  overflow compaction, G35; a call retried after an accidental cut, D12) is written as its own `"error"` entry,
  `errorMessage` the reason it was replaced (the empty-reply error's text, or the failure's). The agent emits it with
  `Err` set, appends it to the turn's messages with `Origin.Failed` set, and goes on; `stamped` keeps `Failed` (today it
  replaces the whole `Origin`, `acpserver/choice.go:250-251`), so the live history equals a reload. Never sent to a
  model. A replaced attempt that showed nothing is not written.
* **D5. If writing fails.** The prompt answers the call's error, not the append's; the append's error is logged at
  warn; the live history is set from the file.
* **D6. Live, when the turn ends on the error.** The JSON-RPC error and the CLI's "(the reply was cut off)" line, as
  today (F2, F7). When a cut is retried, the client gets the `cut` notice (D12 (d)) and no error for it.
* **D7. Never replayed.** A failed entry is never sent to a model as the model's own turn
  (`0025-MADR-provider-choice-and-auth.md` D2). There is no exception. A resume holds every failed entry in the
  history, marked failed.
* **D8. What a load shows (Q2).** After the replayed text of an `"error"` entry, one `agent_message_chunk` on its own
  paragraph: "Error: " and the entry's `errorMessage`, as pi draws it. An entry with no text (thinking only, or
  nothing) shows the line alone; an `"aborted"` entry shows no line. Every ACP client shows the line as the model's
  text, which the answer accepted. A length cut's live notice
  (`0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` D8) is not stored and shows no line.
* **D9. Readers.** `_gobble/get_last_assistant_text` returns a failed entry's text, and null for one with no text,
  without looking further back, as pi's rule (`acpserver/extensions.go:478-503`); after a retried cut the last entry is
  the retry's reply. The compaction summary serialises a failed entry as it does an aborted message
  (`compaction/serialize.go:161-207`); its usage counts in the totals (`acpserver/commands.go:194-219`) and not in the
  context estimate (F5); `/fork` and the tree treat it as any entry.
* **D10. Cancel.** A cancel during a call is unchanged (F4); when a stream fails because the turn was cancelled, the
  cancel wins (`agent/agent.go:700-703`); a cancel during a cut retry's wait or call is D12 (e)'s.
* **D11. Length cuts and refusals.** A `max_tokens` or `refusal` stop is a completed reply (F9), never a failed entry,
  never retried; `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` R21 ("Stop with a notice") governs a
  length cut. Until its D5 to D7 are built, a Responses cut is `ErrIncomplete` (kind `incomplete`): kept by D2, not
  retried, its complete calls kept by D3 and not run.

### The decision: retrying an accidental cut (Q1)

* **D12. Retry after a cut.**
  * **(a) When.** The call failed with `errors.Is(err, llm.ErrUnavailable) || errors.Is(err, llm.ErrOffline)` (F17: a
    dropped connection, an idle timeout, a server error in the stream), after any event of the call: `reply` gains
    `started`, set on every event the provider yields but a `ProviderRetry` (text, thinking, provider data alone, a
    call, usage, a warning, done); the session's `retry.retryAfterCut` is on and retries are on (`Off` false); and
    fewer than 3 cut retries have been made in a row in this turn. Never: a refusal or content filter (not errors, F9),
    `context_overflow`, or any other kind. A cut with no event is not this path: the SDK has retried it before the
    first event (F10), then the error ends the turn, or `offline` takes today's wait (G5). A call half-written at the
    cut never reached gobble and is thrown away with the attempt; no tool ran (F18).
  * **(b) What happens.** One case in `next()` beside the shown guard (`agent/agent.go:514`), reusing the thinking-only
    rerun's path (`:548-567`): the cut attempt is written per D4 when it showed something (an attempt that showed
    nothing is dropped, its released calls with it); the agent emits `agent.Retry` with `Phase` start, `Cause` `cut`,
    `Attempt` the count and `MaxAttempts` 3; it waits, for `unavailable` the session policy's delay (`BaseDelay`
    doubled at each cut retry, capped at `MaxDelay`, `llm.Retry`), for `offline` by `waitOnline` within `OfflineWait`;
    a cancel or an abort ends the wait by `cutShort`; then `continue` resends the same request: the turn's messages are
    unchanged, so the request is byte-identical to the cut one.
  * **(c) How it is stored.** Only D4's entry. No `Origin` flag, no custom entry, no replay exception (D7).
  * **(d) How it shows (Q7, with the cause `cut`).** Live: the cut part as it streamed, then a `_gobble/retry` start
    notice (cause `cut`, attempt n of 3, its delay), ended at the retry's first event (`Err` empty) or its failure
    (`Err` the error's text), as a provider's retry is ended (`agent/agent.go:691-697`); then the reply starts over.
    `acpext.RetryCause` is an open set that clients accept unknown values of (`acpext/doc.go:112`), so `acpext` needs
    no constant and no release; gobble's CLI draws the notice as its other retry lines. On reload: the cut part's text,
    its "Error: …" line (D8), then the retry's reply.
  * **(e) What follows.** The turn's count of cut retries in a row rises by one at each and returns to 0 when a call
    ends `Done` (any stop); each turn starts at 0; it is separate from the empty-reply reruns' count. A retry cut
    accidentally again is retried again while the count is under 3; the cut that would be the fourth ends the turn with
    its error, written by D2. A retry that fails otherwise ends the turn with its error (D2). One that ends in a length
    cut ends the turn `max_tokens` with the notice of `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md`
    R21. One that returns an empty reply takes the empty-reply rerun path. One refused for overflow before any event is
    compacted once and resent (G11). A cancel during the wait ends the turn `StopCancelled`, the D4 entry already
    written; a cancel during the retried call is a cancel during a call (F4).
  * **(f) The setting.** `retry.retryAfterCut`, a boolean session setting, default `true`, in `internal/config/keys.go`
    beside the `retry.*` keys, its doc: "Whether a model call cut by a dropped connection, an idle timeout or a server
    error after its reply began is sent again from the start, at most 3 times in a row; the cut part stays in the
    session as a failed attempt." `retry.enabled: false` and `_gobble/set_auto_retry` off turn it off too, since `Off`
    turns every retry off (F19). Off, a cut ends the turn with its error (D2).
  * **(g) Which calls.** Agent turns only; titles and compaction summaries do not run through `agent.Run` (F13) and are
    never retried after a cut.
  * **(h) With Q3 and Q4.** One entry per call: a retried cut attempt is a D4 entry; the call the turn ends on is D2's;
    nothing is written twice.
  * **(i) Headless.** Text output and JSON hold what streamed: the cut part, then the retry's whole reply, so its start
    appears twice; when the retry completes, JSON carries no error.
* **D13. `0031-MADR-sdk-current-releases-and-adoption.md` D3.** A dated amendment: a shown-then-cut call is replaced by
  a rerun when its cut is accidental (D12), and its text is shown again as the reply starts over, which the owner
  accepted with Q1; any other failure after shown text still ends the turn with the error, and the call is never sent
  again in part. Its "Keep the re-run" amendment stands; replaced attempts are kept (D4).
* **D14. No SDK delivery state.** gobble tells an accidental cut from the rest by its error kind and `started`; the
  SDK's H4 (not built) is not needed.

### The decisions: G8, the stream-health record

* **D15. Where and at what level (Q5).** One `slog` record, message `model call`, per `adapter.Stream` call, through
  the adapter's logger, into the JSON log file, never on stderr. Its level is `Info` when the call's `outcome` is
  `error` or `cancelled`, or it was retried (`retries` above 0, `resent_without_thinking`, or `cut_retry` above 0),
  and `Debug` otherwise. Every call kind is covered. An adapter with no logger records nothing. OTEL is out of scope.
* **D16. Cost.** A call's health is known only at its end, so the adapter measures every call while `Info` is enabled,
  as it is by default: two monotonic clock reads per event and, at the end, one record, or none when the record's level
  is `Debug` and `Debug` is off. When the log is set above `info` (`warn`, `error`), it reads no clock, builds no record
  and allocates nothing more than with no logger. The clock is an unexported `now` field, `time.Now` unless a test sets
  it.
* **D17. The fields.** No key ends in `key`, `token`, `secret` or `password` (F14).

  | Field | What it is |
  | :--- | :--- |
  | `session`, `provider`, `model` | the provider's `SessionID`, its id, the request's model |
  | `background` | a title or automatic compaction (`llm.IsBackground`) |
  | `buffered` | the SDK provider is not a `Streamer`, so the reply arrived whole (F10) |
  | `cut_retry` | the call's place in a run of cut retries, 1 to 3, set by the agent through a context value the adapter reads; absent otherwise |
  | `limit` | the request's output limit, 0 for the SDK's default (`0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` D11) |
  | `cache` | the retention the request asked for |
  | `ttft_ms` | from the start of `Stream` to the first SDK event, retry waits and the SDK's unseen resends included; absent with no event |
  | `gap_max_ms` | the longest wait for an event after the consumer returned from the previous one; meaningless when `buffered` |
  | `handling_max_ms` | the longest time gobble's consumer held one event |
  | `duration_ms`, `events` | the call's whole time; the SDK events it yielded |
  | `retries`, `retry_kinds`, `retry_wait_ms`, `server_directed` | the notified retries before the first event: count, kinds in order, waits' sum, how many the service directed; the SDK's resends inside one send are not counted (F11) |
  | `resent_without_thinking` | sent again with no thinking after a signature refusal |
  | `warnings` | the response's warning kinds, in order |
  | `delivery` | `none`, `events` or `shown` (D1) |
  | `outcome` | `done`, `error`, `cancelled` or `stopped` |
  | `kind`, `status`, `code` | on `error`: the kind, HTTP status and provider's code; never the error's text |
  | `stop` | the stop reason of a `done` call |
  | `input`, `output`, `reasoning`, `cache_read`, `cache_write`, `estimated` | the call's usage (input includes cache reads and writes) |
  | `cache_miss` | `input` less `cache_read`, when caching was asked and usage reported |
  | `out_limit`, `out_limit_sent`, `out_limit_basis`, `out_limit_source` | `Response.OutputLimit`, added when `0066-MADR-set-each-session-model-up-from-discovered-limits.md` is built |

* **D18. Privacy.** The fields above and nothing else: no text, thinking, arguments, results, error text, URL, header
  or credential.
* **D19. Joining the two per-call lines.** The line of `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md`
  D11 is the `limit` field, and the line of `0066-MADR-set-each-session-model-up-from-discovered-limits.md` D8 the
  `out_limit*` fields; whichever PLAN builds first creates the record. The session-report line of
  `0066-MADR-set-each-session-model-up-from-discovered-limits.md` D8 stays that record's. Dated notes go into both
  records' rows (D11 and R31; D8 and S21) and their PLANs.
* **D20. Reruns.** Each offline or empty-reply rerun, each resend after an overflow compaction and each cut retry is its
  own record; a cut retry carries `cut_retry`.
* **D21. What it cannot see.** The SDK's resends inside one send (F11); a credential renewal. A request `toRequest`
  refuses gets a record with `outcome` `error`, `delivery` `none` and `duration_ms` only.
* **D22. Later.** H4 would split `delivery: none`; H1 and H2 of the SDK's comparison change no decision here.

### Consequences

* Good, because a reload shows what the user saw, with pi's "Error: …" line, and an accidental cut is finished without
  the user's help.
* Good, because a retry resends the same request: no stitched answer, no message the user did not type, and nothing
  of the cut attempt is ever sent to the model.
* Good, because each call's health is in the log when it went wrong, at the default level.
* Bad, because a retried cut bills the reply's output again, up to 3 times (the input is byte-identical, so a prompt
  cache should serve it), and live the reply starts over after the notice; the retry may answer differently from what
  was shown (accepted with Q1).
* Bad, because every call is timed at the default level, and every failed or retried call adds a log line.
* Bad, because `agent.Message` and the agent's state grow, and the session file gains `"error"` entries.

### Confirmation

* The PLAN's tests, one per decided row, red at the base where a row changes behaviour; each plant is one production
  change on a scratch clone, caught by its own row's test.
* `make preflight` passes on the three hosts, through `run-on-host`; `make race` on the Linux machine.
* An independent review of the PLAN's last build phase finds no blocker.

## State table

"Today" is 97fa1f6, with go-llmprovider-sdk at `v1.32.0`. Agent tests use `llmtest` scripts. "The adapter seam" is
`provider.New` with `Options.HTTPClient` an in-memory `RoundTripper` serving a scripted SSE body (as
`llm/provider/stream_test.go:34`) and `Options.Logger` over a capturing handler; the provider, its own `Set` (`NewSet`)
and its own `Cache` are built inside the test, inside the `synctest` bubble where one is used, never the process-wide
built-ins. "History test" means a test built as `failedTurnRequests` (`acpserver/resume_test.go:324`): the live
session's next request against the reloaded session's, equal in roles and wire bytes.

**G1: what is written**

| Row | Input or state | Status | Outcome | Decided by | Test / plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| G1 | A call fails after a non-empty text delta, not retried | decided | `agent.Message` with `Err` before the error; `"error"` entry with the text and any thinking | D1, D2, D3 | `TestFailedCallKept/text` (agent); plant: drop the emit |
| G2 | A call fails after thinking only | decided | as G1, the thinking | D2 | `TestFailedCallKept/thinking`; plant: `shown` set by text alone |
| G3 | The turn ends with an error on a call that showed nothing (no event, or complete calls only) | decided | an `"error"` entry all the same, empty or with its calls (Q4) | D2, D3 | `TestFailedCallKept/nothing_shown`, `/calls_only`; plant: emit only when shown |
| G4 | A call fails after text and complete calls | decided | the entry holds text, thinking and the calls (not run, no result); a load shows each call failed, "no result was recorded" (Q4) | D3 | `TestFailedCallKept/text_and_calls` and `TestLoadReplaysFailedEntry/calls`; plant: drop the calls |
| G5 | A call fails before any event and is retried, waited for or compacted | decided | as today: the replaced call, having shown nothing, is not written | D4 | the existing `TestOfflineWait`, `TestEmptyReplyRunsAgain` (guards) |
| G6 | A tool-call preview | out of scope | none exists (F1) | D1 | |
| G7 | A shown thinking-only reply run again, the rerun succeeds | decided | the replaced attempt is its own `"error"` entry, failed live and reloaded (Q3) | D4 | `TestReplacedAttemptsKept/rerun_succeeds` (agent); plant: do not emit replaced attempts. History test `TestReplacedAttemptHistoryIsTheFiles`; plant: `stamped` replaces `Origin` whole (the live copy loses `Failed` and is sent) |
| G8 | Reruns run out after a shown thinking-only reply | decided | the last attempt is the D2 entry (`errNoUsableReply`); the earlier ones D4's | D2, D4 | `TestFailedCallKept/reruns_exhausted`; plant: keep only on the shown-guard path |
| G9 | A rerun's wait aborted after a shown attempt | decided | the D2 entry with `errEmpty` | D2 | `TestFailedCallKept/wait_aborted`; plant: keep nothing for `errEmpty` |
| G10 | An overflow after shown text | decided | no compaction; the D2 entry with the overflow error; not retried | D2, D12 | `TestFailedCallKept/overflow_after_text`; plant: compact before the shown guard |
| G11 | An overflow before any event, nothing shown | decided | compact once, resend; nothing written for it | F8 | the existing overflow tests (guards) |
| G33 | A cancel during the empty-reply wait after a shown attempt | decided | the attempt written `"aborted"`, failed live by `stamped` | D2 | `TestFailedCallKept/cancel_in_wait`; plant: keep only on error paths. History test `TestCancelInWaitHistoryIsTheFiles`; plant: emit it with its `end_turn` stop (written `"stop"`, sent again) |
| G34 | A shown attempt replaced by a rerun that fails before any event | decided | the shown attempt is a D4 entry; the turn's last call the D2 entry (Q3) | D4 | `TestReplacedAttemptsKept/rerun_fails_before_event`; plant: keep replaced attempts only when the rerun succeeds |
| G35 | A shown attempt, then the rerun overflows before any event and is resent after compaction | decided | the shown attempt is a D4 entry (Q3) | D4 | `TestReplacedAttemptsKept/overflow_resend`; plant: drop it on the compaction path |

**G1: cancel, length, refusal**

| Row | Input or state | Status | Outcome | Decided by | Test / plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| G12 | A cancel during a call | decided | unchanged: `"aborted"`, failed live, calls answered cancelled | D10 | the existing cancel tests (guards) |
| G13 | A stream that fails because the turn was cancelled | decided | the cancel wins | D10 | `TestCancelWinsOverFailure` (a guard); plant: test the error before `ctx.Err()` at `agent.go:700` |
| G14 | The consumer stops | decided | nothing written | `agent.go:509-510` | the existing stop tests (guards) |
| G15 | A length cut | decided | completed, `"length"`, not retried; R21 of `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` | D11 | `TestLengthCutIsNotFailed` (a guard); plant: `Err` on any stop that is not end-turn |
| G16 | A Responses cut after text, before D5 of `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` | decided | a D2 entry; not retried (`incomplete`) | D11 | `TestFailedCallKept/responses_cut_text`; plant: exclude `ErrIncomplete` from D2 |
| G36 | A Responses cut after thinking only, before that record's D6 | decided | a D2 entry, thinking only; not retried | D11 | `TestFailedCallKept/responses_cut_thinking`; plant: for `ErrIncomplete`, write only an entry with text |
| G37 | A Responses cut with complete calls, before that record's D7 | decided | a D2 entry with its calls (Q4), not run; not retried | D3, D11 | `TestFailedCallKept/responses_cut_calls` (no tool runs); plant: run the calls before yielding the error |
| G17 | A refusal or content filter | decided | `Done{StopRefusal}`; never a failed entry, never retried | D11 | `TestRefusalIsNotFailed` (a guard); plant: yield the refusal as an error |

**G1: the file, ACP, reload**

| Row | Input or state | Status | Outcome | Decided by | Test / plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| G18 | The entry's shape | decided | assistant, pi's shape, `"error"`, `errorMessage`, priced usage | D2 | `TestFailedEntryShape` (acpserver, reads the file); plant: write `piStop(ev.StopReason)` |
| G19 | The next request after a failure that is not retried (an HTTP 400) | decided | live equals reloaded; neither carries the failed entry | D7 | History test `TestShownFailureHistoryIsTheFiles` asserting no assistant text "half"; plant: write `"stop"` for it |
| G20 | The entry's append fails | decided | the call's error answered; the append's at warn | D5 | `TestFailedEntryAppendFails` (with `failAt`, `resume_test.go:377`); plant: return `appendFailed` |
| G21 | Live ACP and CLI when the turn ends on the error | decided | the JSON-RPC error; "(the reply was cut off)" | D6 | the existing `TestCutOffReply` (`internal/cli/cutoff_test.go:14`) and error-data tests (guards) |
| G22 | Headless output when the turn ends on the error | out of scope | unchanged (F7) | D6 | |
| G23 | Load of a session with a failed entry with text | decided | its text replayed | D8 | `TestLoadReplaysFailedEntry/text`; plant: skip `"error"` in `replayMessage` |
| G24 | What follows it on load | decided | one chunk "Error: <errorMessage>" on its own paragraph (Q2) | D8 | `TestLoadReplaysFailedEntry/error_line`; plant: omit the line |
| G38 | Load of a failed entry with no text | decided | the "Error: …" line alone (Q2) | D8 | `TestLoadReplaysFailedEntry/no_text`; plant: emit the line only after text |
| G40 | Load of an `"aborted"` entry | decided | no line, as today | D8 | `TestLoadReplaysFailedEntry/aborted` (a guard); plant: emit the line for `"aborted"` too |
| G25 | Resume | decided | the history holds it failed, never sent | D7 | the reload half of G19 and C9 |
| G26 | `_gobble/get_last_assistant_text`, an entry with text | decided | its text | D9 | `TestLastAssistantTextAfterFailure` (the probe of F3, asserting); plant: skip `"error"` there |
| G39 | The same, an entry with no text | decided | null, not the reply before it | D9 | `TestLastAssistantTextAfterEmptyFailure`; plant: look past a message with no text |
| G27 | A compaction summary over it | decided | serialised as quoted history | D9 | `TestSerializeKeepsFailedEntry` (a guard); plant: filter `"error"` in `serializeOmitting` |
| G28 | Totals and cost | decided | its usage counts | D9 | `TestFailedEntryCounted` (a guard); plant: skip `"error"` in `sessionTotals` |
| G29 | `/fork` and the tree | decided | the entry is the leaf; the next prompt its child | D9 | `TestForkAfterFailedEntry` (`_gobble/get_entries`: the second user entry's parent); plant: write it without making it the leaf |

**G1: the SDK and the records**

| Row | Input or state | Status | Outcome | Decided by | Test / plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| G30 | What gobble does after an accidental cut | decided | retries the same request (Q1): rows C1 to C24 | D12 | the C rows |
| G31 | Which signals G1 and D12 use | decided | `shown`, `started` and the error's kind; no H4 | D14 | pinned by G1, G2, C2 and C4; no behaviour of its own |
| G32 | `0031-MADR-sdk-current-releases-and-adoption.md` D3 | decided | a dated amendment (D13) | D13 | `make check-records` |

**Retry after a cut (Q1)**

| Row | Input or state | Status | Outcome | Decided by | Test / plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| C1 | A call that showed text fails `unavailable` or `offline`, setting on, under the cap | decided | the cut attempt a D4 entry; a `cut` notice; the wait; the same request resent; no error to the client | D12 (a), (b) | `TestRetryAfterCut/unavailable`, `/offline` (agent); plant: never retry |
| C2 | A call fails with any other kind after an event, `context_overflow` included | decided | not retried; the turn ends with the error (D2) | D12 (a) | `TestRetryAfterCut/not_accidental` (each kind of `llm/kind.go:29-42` but `unavailable` and `offline`); plant: add `llm.ErrRateLimited` to the kinds retried |
| C3 | A refusal or content filter | decided | not retried (not an error) | D11 | `TestRetryAfterCut/refusal`; plant: treat `StopRefusal` after text as a cut |
| C4 | An accidental cut after events that showed nothing (an encrypted-only reasoning item, a released call) | decided | retried; the attempt, having shown nothing, is not written (D4). The earlier premise, "the SDK already retried it", was false (F20) | D12 (a), D4 | `TestRetryAfterCut/events_not_shown`; plant: require `shown` instead of `started` |
| C5 | A cut while a tool call was being written | decided | retried as any accidental cut; the half call never reached gobble and goes with the attempt; no tool ran (Q6 moot) | D12 (a); F18 | `TestRetryAfterCut/after_released_call` (a complete call, then the cut: no `ToolStart`, no tool run, the resent request equal to the first); plant: run the delivered calls before retrying |
| C6 | The resent request | decided | byte-identical to the cut call's; nothing of the attempt in it | D12 (b), D7 | `TestRetryAfterCutRequest` (`llmtest` records both requests); plant: append the cut attempt to the turn's messages unmarked before resending |
| C7 | An accidental cut with no event | decided | not this path: the SDK's own retry before the first event, then the turn ends, or `offline` waits as today (G5) | D12 (a) | `TestRetryAfterCut/no_event` (one request, the error); plant: drop the `started` condition |
| C8 | The wait before a retry | decided | `unavailable`: `BaseDelay` doubled at each cut retry, capped at `MaxDelay`; `offline`: `waitOnline` within `OfflineWait` | D12 (b) | `TestRetryAfterCut/backoff` (`synctest`; defaults give 2 s, 4 s, 8 s; an offline cut probes); plant: no wait |
| C9 | The history after a retried turn, live and reloaded | decided | equal; the cut attempt is a failed entry in both and in neither request | D12 (c), D7 | History test `TestRetryAfterCutHistoryIsTheFiles`; plant: emit the cut attempt with no `Err` (written `"stop"`, sent again) |
| C10 | A cancel during the wait before a retry | decided | `StopCancelled`; the D4 entry stays `"error"`; nothing more written | D12 (e) | `TestRetryAfterCut/cancel_in_wait`; plant: write the cut attempt only after the wait |
| C11 | A retry that returns an empty reply | decided | the empty-reply rerun path, with its own count | D12 (e) | `TestRetryAfterCut/then_empty`; plant: count empty reruns against the cut cap |
| C12 | How a retry shows, live and on load | decided | live: the cut part, a `_gobble/retry` notice of cause `cut`, attempt n of 3, ended at the retry's first event or failure, then the reply from its start; on load: the cut part, "Error: …", the retry's reply (Q7) | D12 (d) | `TestRetryAfterCutShows` (acpserver: live updates in order: chunk "half", a retry start of cause `cut`, attempt 1 of 3, its end, chunk "whole"; a load replays "half", the "Error: …" line, "whole"); plant 1: emit no `agent.Retry` before the retry; plant 2: end the notice before the wait |
| C13 | A retry cut accidentally again | decided | retried again; counted | D12 (e) | `TestRetryAfterCut/chain`; plant: reset the count at each retry |
| C14 | The cap | decided | at most 3 in a row per turn; the fourth accidental cut ends the turn with its error (D2); a `Done` resets; each turn starts at 0 | D12 (e) | `TestRetryAfterCut/cap` (four cuts; and a reset after a completed call); plant: cap at 4 |
| C15 | A retry fails otherwise | decided | the turn ends with that error; its entry kept (D2) | D12 (e) | `TestRetryAfterCut/then_fails`; plant: retry on any error after a cut retry |
| C16 | A retry ends in a length cut | decided | `max_tokens` with the length-cut notice | D12 (e) | `TestRetryAfterCut/then_length`; plant: retry after a length cut |
| C17 | A retry refused for overflow before any event | decided | compact once, resend | D12 (e) | `TestRetryAfterCut/then_overflow` (a guard); plant: end the turn instead |
| C18 | A cancel during the retried call | decided | `"aborted"`, as a cancel during a call; the earlier D4 entry stays | D10, D12 (e) | `TestRetryAfterCut/cancelled`; plant: drop the D4 entry on cancel |
| C19 | The setting | decided | `retry.retryAfterCut`, default `true`; off, or `retry.enabled: false`, or `_gobble/set_auto_retry` off: no retry after a cut | D12 (f) | `TestRetryAfterCutSetting` (internal/config and agent); plant: default `false`; second plant: `Off` not consulted |
| C20 | The setting's docs | decided | the key's `Doc`, the settings schema golden, `docs/architecture.md` | D12 (f) | the schema golden test of `internal/config/render_test.go`, regenerated; plant: a different default in `keys.go` |
| C21 | Titles and compaction summaries | decided | never retried after a cut | D12 (g) | `TestTitleCutNotRetried` (acpserver: a title cut after text makes one request); plant: retry the title once on a cut |
| C22 | Q3 and Q4 with a retry | decided | one entry per call: each retried cut attempt that showed something a D4 entry, the last call D2's | D12 (h) | `TestRetryAfterCutWritesOneEntryPerCall`; plant: write a retried attempt twice (as D4 and D2) |
| C23 | The record of a cut retry | decided | `cut_retry` 1 to 3; at `Info` (retried) | D15, D17 | `TestModelCallRecord/cut_retry`; plant: omit the context value |
| C24 | Headless output with a retry | decided | text and JSON hold the cut part and the retry's whole reply as streamed; no error when it completes | D12 (i) | `TestPrintAfterCutRetry` (internal/cli over `acptest.ScriptAgent`: chunks, a `_gobble/retry` start and end of cause `cut`, chunks, `end_turn`); plant: the printer drops text before the notice |

**G8: the record**

| Row | Input or state | Status | Outcome | Decided by | Test / plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| T1 | A call | decided | one `model call` record | D15 | `TestModelCallRecord` (the adapter seam); plant: no record |
| T2 | The log set above `info` | decided | no record, no clock read, no extra allocation | D16 | `TestModelCallRecordOffAboveInfo` (a counting `now`: zero calls at `warn`); plant: read `now` before `Enabled` |
| T3 | Which calls | decided | turns, titles, compaction summaries (`background`) | D15 | `TestModelCallRecord/background`; plant: `background` always false |
| T4 | Timing | decided | `ttft_ms`, `gap_max_ms`, `handling_max_ms`, `duration_ms`, `events` | D17 | `TestModelCallTiming` (in `synctest`, pull-based, no wait overlapping: the first event after 1 s; the second after a 3 s gap while the consumer holds nothing; the consumer holds the second 5 s; the third is written only when the consumer closes a channel at the hold's end; wants 1000, 3000, 5000); plant: gap from arrival to arrival (gives 5000) |
| T5 | Retries | decided | count, kinds, waits, server-directed | D17 | `TestModelCallRetries` (503, then 429 with Retry-After, then a stream); plant: count only the last notice |
| T6 | A signature resend | decided | `resent_without_thinking`, one record | D17 | `TestModelCallSignatureResend`; plant: one record per attempt |
| T7 | Delivery | decided | `none`, `events`, `shown` | D17 | `TestModelCallDelivery`; plant: `shown` on any event |
| T8 | Outcome | decided | `done` with `stop`; `error` with `kind`, `status`, `code`; `cancelled`; `stopped` | D17 | `TestModelCallOutcome`; plant: `error` for a cancel |
| T9 | Usage and cache | decided | usage; `cache_miss` = `input` − `cache_read` | D17 | `TestModelCallCache`; plant: subtract the writes too |
| T10 | No usage | decided | no usage fields | D17 | `TestModelCallOutcome/error`; plant: zeros written |
| T11 | Privacy | decided | the keys are exactly D17's (less `out_limit*` until that record is built) | D18 | `TestModelCallRecordHoldsNoContent` (an allowlist); plant: add the error's text |
| T12 | Redaction | decided | through the real `logging` file handler every field keeps its value | D17; F14 | `TestModelCallRecordSurvivesRedaction` (internal/cli, a temp log file); plant: rename `retry_kinds` to `retry_key` |
| T13 | The request limit of `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` D11 | decided | `limit` | D19 | `TestModelCallRecord` asserts `limit` 0; plant: omit it |
| T14 | The `OutputLimit` of `0066-MADR-set-each-session-model-up-from-discovered-limits.md` D8 | decided | absent at `v1.32.0`; added with that record | D19 | `TestModelCallRecordHoldsNoContent/no_out_limit`; plant: write `out_limit: 0` |
| T15 | The turn's reruns | decided | each its own record | D20 | `TestModelCallRecordPerRerun`: `agent.New` over a real adapter; the seam answers a `completed` Responses stream with no output, which the SDK fails as its empty-answer error (mapped to `llm.ErrEmptyReply`; Phase 0 confirms), then a text reply (not a `Done` with no text, which ends `done` and would let the plant through); two records, the first `error`, `empty_answer`; in an external test package of `llm/provider`, or of `internal/cli` if the architecture test refuses the import; plant: write the record only on `done` |
| T16 | `--stats` | out of scope | unchanged | D15 | |
| T17 | OTEL | out of scope | a later record | D15 | |
| T18 | H4's finer state | out of scope | a follow-up field | D22 | |
| T19 | Cost | decided | above `info`: allocations equal to no logger; at `info`: two clock reads per event, a benchmark reported | D16 | `TestModelCallRecordNoAllocsAboveInfo` (`testing.AllocsPerRun`, 100 events, against no logger, at `warn`); plant: build the attributes before `Enabled`; `BenchmarkAdapterStream` reported |
| T20 | The SDK's unseen resends | decided | not in `retries`; in `ttft_ms`; `warnings` carries the kinds | D17, D21 | `TestModelCallRecord/warnings`; plant: drop `warnings` |
| T21 | A non-`Streamer`, and a `Streamer` reporting `BestEffort` | decided | `buffered` true, then false | D17 | `TestModelCallRecordBuffered`, in-package (`Set.register` is unexported, `llm/provider/set.go:89`), on a `NewSet` built in the test; plant: `buffered` from `Capabilities().NativeStreaming` (the second case then comes out true, the SDK's `retry.go:142-145`) |
| T22 | `toRequest` fails | decided | `error`, `delivery` `none`, `duration_ms` only | D21 | `TestModelCallRecord/request_refused` (Phase 0 names the input); plant: return before the record |
| T23 | The record's level | decided | `Info` when `error`, `cancelled` or retried; `Debug` otherwise (Q5) | D15 | `TestModelCallRecordLevel` (four cases and a healthy one); plant: every record at `Debug` |

**Records and docs**

| Row | Input or state | Status | Outcome | Decided by | Test / plant |
| :--- | :--- | :--- | :--- | :--- | :--- |
| R1 | Dated notes | decided | `0031-MADR-sdk-current-releases-and-adoption.md` D3 (D13); `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md` D11, R31 and its PLAN; `0066-MADR-set-each-session-model-up-from-discovered-limits.md` D8, S21 and its PLAN | D13, D19 | `make check-records` |
| R2 | Docs | decided | `docs/architecture.md` (the `acpserver`, `agent` and `llm/provider` rows: failed entries, the cut retry, the record); the guides that name `gobble.log` (`docs/guides/credentials.md`, `docs/guides/mcp-servers.md`); the setting (C20); `docs/README.md`'s index | Confirmation | the PLAN's docs rows |

Counts: 14 + 8 + 15 + 3 + 24 + 23 + 2 = 89 rows; 84 decided, 5 out of scope (G6, G22, T16, T17, T18), 0 open.
*(Before the fifth draft: 83 decided, 1 open, C5. Before the Q7 amendment: 82 decided, 2 open.)* No row is open, so by
the state-table rule the record is `accepted` (2026-11-24).

## Owner questions

None open. *(2026-11-24: Q6 and Q7 were asked in the fourth draft. Q7 was answered "Retry notice live
(Recommended)"; Q6 became moot when Q1 was re-answered "Auto-retry, with limits (Recommended)". Q1's earlier answer and
Q6's options, a continuation-only question, are kept in the history of this file and summarised in "Owner rulings".)*

## Pros and Cons of the Options

### A. pi's way from gobble's signals; retry in the agent; one record in the adapter

* Good, because nothing waits for the SDK, the file stays pi's format, and the retry is the design codex, pi, qwen-code's
  hosted harness and opencode since #60674 use (F21).
* Bad, because the adapter cannot name the agent's own rerun reasons, nor the SDK's resends inside one send.

### B. Wait for the SDK's delivery state

* Bad, because it adds an SDK record and release, and tells gobble nothing `shown`, `started` and the kind do not.

### C. Measure in the agent

* Bad, because titles and compaction summaries are not agent calls, and the request's and response's values are the
  adapter's.

### D. A file of its own, or `--stats`

* Bad, because it repeats the log's rotation, level and redaction, and `--stats` never sees a call.

### E. Change nothing

* Bad, because what the user saw is lost on reload (F3), and every accidental cut needs the user.

## More Information

### Relation to other records

* `0031-MADR-sdk-current-releases-and-adoption.md` D3, as amended: amended by D13.
* `0025-MADR-provider-choice-and-auth.md` D2: failed entries are never sent (D7).
* `0037-MADR-hardening-integrity-and-performance.md` D15: live equals reloaded; the history tests pin it.
* `0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md`: its Q2 (R21, "Stop with a notice"), D5 to D8 and
  D11 bear on G15, G16, G36, G37, C16, D8 and T13.
* `0066-MADR-set-each-session-model-up-from-discovered-limits.md` D8, S21: T14.
* `0056-MADR-curate-the-terminal-session-view-with-view-modes.md`: view modes do not change "shown".
* `0057-MADR-cache-visibility-and-one-hour-pricing.md`: cache notices stay its own; T9 only logs.
* go-llmprovider-sdk `docs/reports/0091-REPORT-responses-api-across-providers.md`, addition of 2026-11-24: H1, H2, H4.

### Dependency on the SDK

None. The `out_limit*` fields wait for the SDK release carrying go-llmprovider-sdk
`docs/decisions/0089-MADR-model-output-limit-discovery.md`.

### Not verified

* Which existing tests the new events and entries turn red: the PLAN's Phase 0 lists them on a scratch clone.
* Whether the architecture test lets an external test package of `llm/provider` import `agent` (T15), the input
  `toRequest` refuses (T22), and the SDK's empty-answer mapping (T15): Phase 0.
* That a byte-identical resend hits the prompt cache on each service: judgment, not measured; the record's
  `cache_read` shows it per call.
