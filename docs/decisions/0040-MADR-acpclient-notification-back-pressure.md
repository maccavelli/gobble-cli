---
status: accepted
date: 2026-11-01
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Pause the agent instead of dropping its notifications when gobble's client falls behind

## Context and Problem Statement

`acpclient`, gobble's in-process ACP client, names acp-go-sdk2's `OverflowDropNewest`
(`acpclient/conn.go:133`). When its inbound notification queue is full, 1,024 by default, each notification past it is
dropped, counted by `Conn.Dropped()`, and handed to `Options.OnDrop`, which the CLI turns into a warning, "dropped %d ACP
notifications (%s)" (`internal/cli/agent.go:263-265`). [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md)
chose that in its amendment of 2026-10-01, because the SDK's default then closed the connection.

The cost is lost output. In print mode with `--output-format stream-json`, `printSink.on` writes each update to standard
output; a slow reader of that output fills the queue, and the updates past it are missing from the stream, with only a
warning on standard error. The line view loses reply text the same way, on a terminal slower than the agent.

acp-go-sdk2 `v1.29.0` added a third policy, `acp.OverflowBlock`, and made it the default
([acp-go-sdk2 0044-MADR](https://github.com/maccavelli/acp-go-sdk2/blob/v1.29.0/docs/decisions/0044-MADR-notification-queue-back-pressure.md)).
gobble pins `v1.31.1` ([0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md)
P24). [gobble-cli#8](https://github.com/maccavelli/gobble-cli/issues/8) asks to adopt it. The owner, 2026-11-01:
"switch the client to overflowblock."

### What `OverflowBlock` does, at `v1.31.1`

From the SDK's release notes for `v1.29.0` and `v1.30.0`, and its README's "Closing and limits":

* **The pause.** When the queue holds its limit, the connection stops reading the agent until a notification handler
  returns. The agent's writes wait, so the in-process agent is slowed at its `session/update` sends. Nothing is dropped,
  and the connection stays open.
* **What waits behind a pause.** Nothing else is read either: a `session/request_permission` from the agent waits behind
  the queue. A request the client sent returns no later than before, since it returns only once the notifications
  received before its answer are handled.
* **The one rule.** A notification handler must not wait for a request or a `$/cancel_request` the agent sends later, nor
  for code that waits for one.
* **The reading-on limit.** Where a notification handler waits for the answer to a request it sent, or `Initialize` waits
  behind held notifications, the connection reads on to twice the limit, then closes with
  `acp.ErrNotificationQueueOverflow`.
* **The byte budget** (`v1.30.0`): the connection also counts the bytes it holds, 64 MiB by default, and pauses at
  whichever limit comes first.
* **The log.** A pause of 5 seconds is logged at `Warn` through the connection's logger: "notification queue full;
  reading paused".

### gobble's handlers, read for the rule (at `9250596`)

* `handler.SessionUpdate` (`acpclient/conn.go:408-430`) and `handler.deliver` (`:477-488`), the extension notices' path,
  take `h.mu` to find the session's callback, release it, translate, and call the callback. `h.mu` is never held across
  a call to the agent; `newMu`, held across `NewSession` (`acpclient/sessions.go:88-96`), is taken by no handler.
* The callbacks are `printSink.on` (`internal/cli/print.go`) and `lineSession.onUpdate` (`internal/cli/line.go:406`),
  which calls `view.on`. Each takes its own lock and writes its output. None sends a request to the agent or waits for
  one.
* The permission prompt runs on the request's own goroutine (`internal/cli/permission.go`), and holds no lock a callback
  takes while it waits for the user.
* `acpclient.Start` has one caller, `internal/cli/agent.go:259`.

So the rule holds: a pause means the agent waits for the terminal, or for standard output's reader, which is the
back-pressure a slow consumer should apply.

### The agent side

`acpserver.Serve` names no policy, so since the `v1.31.1` pin it has `OverflowBlock`, as
[0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md)'s
D1 amendment of 2026-11-01 records. A client sends the agent only `session/cancel` among notifications, and
`Agent.Cancel` waits for nothing the client sends later.

## Decision Drivers

* No reply text or update is lost: `stream-json` is read by scripts.
* The connection stays open: a full queue is not a failure.
* The SDK's default, named, so a test can pin it.
* No API that can no longer fire.

## Considered Options

* **A. `OverflowBlock`, named; the drop API removed.**
* **B. `OverflowBlock`, named; `OnDrop` and `Dropped()` kept for a caller that opts back into dropping.**
* **C. Keep `OverflowDropNewest`.**

## Decision Outcome

Chosen option: **"A. `OverflowBlock`, named; the drop API removed"**, because it ends the loss the owner asked to end,
and leaves no field that never fires. Q1 below is answered (a).

* **D1. `acpclient.Start` names `acp.WithNotificationOverflowPolicy(acp.OverflowBlock)`,** although it is the SDK's
  default, so that a test can pin it and a later SDK default cannot change it silently.
* **D2. The drop API goes** (Q1): `Options.OnDrop`, `Conn.Dropped()`, and the CLI's "dropped %d ACP notifications"
  warning (`internal/cli/agent.go:263-265`). Under `OverflowBlock` the drop handler is never called, and the count stays
  0. `acpclient` is `beta`, and no caller outside `internal/cli` sets them.
* **D3. `Options.MaxQueued` stays,** its meaning now the count at which the reader pauses; its doc says so.
* **D4. The pause's log.** The SDK's `Warn` after 5 seconds goes through `Options.Logger`, the run's log, as every
  connection diagnostic does. No gobble message is added.
* **D5. The test.** `TestDropReported` (`acpclient/conn_test.go:263`) asserts dropping, so it is replaced, not fixed: with
  `MaxQueued: 1` and a callback blocked, the agent's updates wait; released, every update arrives, in order, and a
  second prompt on the same connection succeeds. It is seen failing with `OverflowDropNewest` put back.
* **D6. The records.** 0004-MADR's amendment of 2026-10-01 gains a dated note that this record supersedes its
  `OverflowDropNewest` for `acpclient`; 0008-MADR's "Delivery" fact (`:116`) gains one that gobble's client no longer
  drops.

### Consequences

* Good, because no notification is lost: a slow terminal or reader slows the agent, and `stream-json` holds every
  update.
* Good, because the connection never closes or drops on a full queue, and a request gobble sent is not delayed by it.
* Good, because two exported names and a warning that can no longer fire are gone.
* Neutral, because a request the agent sends, such as a permission question, waits while the reader is paused; gobble's
  handlers return on their own, so the pause ends.
* Bad, because a reader that stops reading standard output stops the agent too, where before it lost updates. That is
  the intended trade; the pause is logged after 5 seconds.
* Bad, because removing `OnDrop` and `Dropped()` changes `acpclient`'s exported API. `make apidiff` is a no-op below
  `v1.0.0`; the change is recorded here and checked by `go doc`.

### Confirmation

1. `acpclient/conn.go` names `acp.OverflowBlock`, and no production file names `OverflowDropNewest`.
2. The test of D5 passes, and fails on a copy that names `OverflowDropNewest`.
3. `OnDrop`, `Dropped` and the drop warning are gone (Q1 (a)); `go doc ./acpclient` shows the change and nothing else.
4. Every other test passes unchanged; `print-stream-json.golden`, `print-json.golden` and the view goldens are
   byte-identical.
5. `make preflight`, `make race` and `make perf` pass on the three hosts, and `make probe-conhost` on the Windows laptop.

## Pros and Cons of the Options

### A. `OverflowBlock`, named; the drop API removed

* Good, because nothing is lost and nothing dead stays.
* Bad, because the exported API changes.

### B. `OverflowBlock`, named; the drop API kept

* Good, because the exported API keeps its names.
* Bad, because `OnDrop` never fires and `Dropped()` is always 0, since `acpclient` offers no way to choose dropping;
  keeping them means adding a policy option no caller wants.

### C. Keep `OverflowDropNewest`

* Good, because nothing changes.
* Bad, because updates keep being lost when a reader is slow, the loss #8 reports.

## More Information

### Questions for the owner

* **Q1. The drop API.**
  * **(a) Recommended.** Remove `Options.OnDrop`, `Conn.Dropped()` and the CLI's drop warning.
  * **(b)** Keep them, documented as applying to a dropping policy no caller can choose today.

  *Answered 2026-11-01: "Remove them (Recommended)", (a). With the owner's "switch the client to overflowblock", the
  record is accepted.*

### Relationship to other records and issues

* [gobble-cli#8](https://github.com/maccavelli/gobble-cli/issues/8): this record carries it out.
* [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md), amendment of 2026-10-01: superseded for
  `acpclient`'s overflow policy (D6).
* [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md): its "Delivery" fact gains a note (D6).
* [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md) and its PLAN record the earlier choice,
  after Magic's 0167-MADR; they are history and are left as they are.
* [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md):
  its D1 amendment of 2026-11-01 left this to #8.
