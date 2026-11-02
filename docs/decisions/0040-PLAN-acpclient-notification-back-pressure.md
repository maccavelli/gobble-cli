---
status: in-progress
date: 2026-11-01
associated-madr: "0040-MADR-acpclient-notification-back-pressure.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement pausing the agent instead of dropping its notifications when gobble's client falls behind

Associated MADR: [0040-MADR-acpclient-notification-back-pressure.md](0040-MADR-acpclient-notification-back-pressure.md)

## Goal

`acpclient` names acp-go-sdk2's `OverflowBlock`, so a full notification queue pauses the agent instead of dropping its
updates; the drop API goes, by Q1; a test shows every update arrives, and the records say so. The plan is complete when
the MADR's Confirmation 1 to 5 is met.

## Scope

**Base.** `main` at `9250596`, acp-go-sdk2 `v1.31.1`, Go 1.27.3.

**In scope:** `acpclient/conn.go`, `acpclient/conn_test.go`, `internal/cli/agent.go`, and the records named in P0 and
P1.

**Out of scope:** the agent side's policy, already `OverflowBlock` (0035-MADR D1's amendment of 2026-11-01); the
queue's size and byte budget, which stay the SDK's defaults; any gobble message for the SDK's pause log.

## Conventions every phase follows

0039-PLAN's C1 to C11 apply, as written there: the start state, line numbers re-read, scratch work, long output to a
log, G-local (`make pre-add-check`, `make preflight`, `make race`, and `make perf` for a phase touching
`internal/cli`), G-hosts (the Linux machine and the Windows laptop, `make perf` and `make probe-conhost` for
`internal/cli`), deviations stopped and recorded, every new test seen failing on a broken copy, `git commit --no-edit`
by the owner, and no identifier of a real machine in a committed file.

## Implementation Steps

### P0 — the records

Files: this pair; `docs/README.md`.

1. `make check-records markdownlint` passes.
2. The identifier scan over the staged records finds nothing.
3. Staged; the owner commits.

### P1 — `OverflowBlock` (MADR D1 to D6; Q1)

Files: `acpclient/conn.go`, `acpclient/conn_test.go`, `internal/cli/agent.go`,
`docs/decisions/0004-MADR-go-module-architecture.md`, `docs/decisions/0008-MADR-native-cli-mode.md`, this plan,
`docs/README.md`. *(Deviation 1: `0008-MADR` is not changed; the MADR is amended.)*

1. **The facts,** re-read at the phase's start; a difference is a deviation: `acpclient.Start` has one caller
   (`internal/cli/agent.go:259`); `OnDrop`, `Dropped` and `MaxQueued` are set nowhere else but `conn_test.go`; no
   notification handler or callback waits for the agent (the MADR's reading).
2. **The policy** (D1): `connOpts` names `acp.WithNotificationOverflowPolicy(acp.OverflowBlock)`, with a comment citing
   the MADR.
3. **The drop API** (D2, by Q1):
   - (a): `Options.OnDrop` and its `WithNotificationDropHandler`, `Conn.Dropped()`, and the CLI's `OnDrop` warning
     (`internal/cli/agent.go:263-265`) go; `startAgentWith`'s doc no longer speaks of drops;
   - (b): they stay, their docs saying they apply to a dropping policy, which no caller can choose today.
4. **`MaxQueued`** (D3): its doc says it is the count at which the reader pauses.
5. **The test** (D5): `TestDropReported` gives way to `TestBlockedUpdatesArriveInOrder`. Four numbered updates, `"u0"` to
   `"u3"`, with `MaxQueued: 1` and a callback that blocks until released: the prompt does not return while the
   callback is blocked; released, the callback has seen `u0` to `u3`, each once, in order; a second prompt succeeds.
   Seen failing (C9) on a copy that names `OverflowDropNewest`.
6. **`go doc ./acpclient`** is diffed against the base: the removed names (Q1 (a)) and the changed docs, nothing else.
7. **Assert by search:** no production file names `OverflowDropNewest`; under Q1 (a), no `OnDrop`, `Dropped(` or "dropped %d
   ACP notifications" remains.
8. **The records** (D6): 0004-MADR's amendment of 2026-10-01 and 0008-MADR's "Delivery" fact each gain a dated note
   naming this record. *(Deviation 1: 0008-MADR's fact is Magic's; only 0004-MADR gets the note.)*
9. **Goldens:** `print-stream-json.golden`, `print-stream-json-escapes.golden`, `print-json.golden` and the view goldens
   are byte-identical.
10. G-local with `make perf`; G-hosts with `make perf` and `make probe-conhost`. **Commit.**

### P2 — close-out

Files: this plan, the MADR if a fact changed, `docs/README.md`.

1. A comment for gobble-cli#8, written into the execution record for the owner to post: what changed, with P1's commit.
   Posting waits for the owner's word.
2. The disclosure guard over the outgoing commits, its output recorded. Nothing is pushed.
3. This plan's status becomes `complete`; the MADR stays `accepted`.

## Verification

| MADR Confirmation | Phase | Check |
| :--- | :--- | :--- |
| 1 `OverflowBlock` named | P1 | step 7's search; step 2's code |
| 2 the test, seen failing | P1 | `TestBlockedUpdatesArriveInOrder` |
| 3 the drop API gone (Q1 (a)) | P1 | step 7's search; step 6's `go doc` diff |
| 4 nothing else changes | P1 | the full suite; step 9's goldens |
| 5 the gates | P1 | G-local, G-hosts |

## Rollout and Rollback

* **One commit for the code,** P1; reverting it restores dropping and the drop API.
* **User-visible:** a slow terminal or `stream-json` reader slows the agent instead of losing updates; the warning
  "dropped N ACP notifications" is gone, and a pause past 5 seconds is logged instead.
* **No configuration, data or wire change.**

## Execution record

Each phase appends its record here.

**Approved 2026-11-01,** the owner's "proceed", with Q1 answered (a).

### P0 record (2026-11-01): the records

This pair and its `docs/README.md` rows, committed records-only under the bootstrap exception. `make check-records
markdownlint` passed; the identifier scan over the staged records found nothing.

### Deviation 1 (2026-11-01): P1 step 8, 0008-MADR's "Delivery" fact is magic-cli-remote's

- **Found** at P1 step 8. 0008-MADR's "Delivery" fact (`0008-MADR-native-cli-mode.md:116`) reads "The SDK fork's inbound
  queue is 1,024 notifications with `OverflowDropNewest`, so any notification can be dropped while the daemon is
  stalled", citing magic-cli-remote's `acpagent/session.go:582-603` and its 0167-MADR. It is Magic's daemon, not gobble's
  client; the MADR's D6 misread it. 0004-MADR's amendment, "`acpclient` sets `OverflowDropNewest` …", is gobble's.
- **The owner's answer:** "Skip 0008; correct D6 (Recommended)". The option not taken was a note on 0008 contrasting gobble's client with
  Magic's daemon.
- **Decision.** 0008-MADR is not changed; 0040-MADR D6 is amended; P1 step 8 and its file list are annotated.
- **Files added:** none; `0008-MADR` leaves P1's list.

### Deviation 2 (2026-11-01): P1, Node crashed in `npx` on the Windows laptop

- **Found** in P1's G-hosts: on the Windows laptop, `make check-records markdownlint` exited 2. `check_records.py
  --check-all` passed; `npx --yes markdownlint-cli2@0.23.2` crashed before linting: `npx: line 65: … Segmentation fault
  "$NODE_EXE" "$NPX_CLI_JS" "$@"`, `make: *** [makefile:236: markdownlint] Error 139`. The Linux machine and the Mac
  passed the same gate at once.
- **Rerun** on the same copy: `records=0`, "Linting: 56 files", "Summary: 0 issues in 0 files".
- **The owner's answer:** "Record, keep the rerun (Recommended)". The option not taken was investigating the laptop's
  Node install before P1 commits.
- **Decision.** An environment fault, not the tree's; the rerun stands as P1's Windows records gate. Nothing changed.
- **Files added:** none.

### P1 record (2026-11-01): `OverflowBlock`

The base is P0's `8250cc4`. Deviations 1 and 2.

1. **The facts** (step 1) held: `acpclient.Start` has one caller (`internal/cli/agent.go:259`); `OnDrop`, `Dropped` and
   `MaxQueued` were used in `acpclient/conn.go`, `conn_test.go` and `internal/cli/agent.go` alone.
2. **The policy** (step 2): `connOpts` names `acp.WithNotificationOverflowPolicy(acp.OverflowBlock)`, its comment citing
   D1.
3. **The drop API** (step 3, Q1 (a)): `Options.OnDrop`, its `WithNotificationDropHandler`, `Conn.Dropped()` and the CLI's
   `OnDrop` warning are gone; `startAgentWith`'s doc no longer speaks of drops. `gofmt` realigned its options literal.
4. **`MaxQueued`** (step 4): "how many inbound notifications wait before the connection stops reading the agent until a
   handler returns: the agent is paused, and nothing is dropped".
5. **The test** (step 5): `TestBlockedUpdatesArriveInOrder` replaces `TestDropReported`: four updates `u0` to `u3`,
   `MaxQueued: 1`, a callback blocked until released; the prompt has not returned 300 ms in; released, the callback saw
   `u0` to `u3` in order; a second prompt succeeds. **Seen failing** on a copy naming `OverflowDropNewest`:
   `conn_test.go:305: updates ["u0" "u1"], want ["u0" "u1" "u2" "u3"], each once, in order`.
6. **`go doc ./acpclient`** (step 6), against `8250cc4`: `Conn.Dropped` and `Options.OnDrop` removed, `MaxQueued`'s doc
   changed, nothing else.
7. **Assert by search** (step 7): no non-test file names `OverflowDropNewest`, and no file `OnDrop`, `Dropped(` or
   "dropped %d ACP" (`grep` exits 1 each).
8. **The records** (step 8, deviation 1): 0004-MADR's amendment of 2026-10-01 has a dated note naming this record.
   0008-MADR is unchanged; 0040-MADR D6 is amended.
9. **Goldens** (step 9): none changed.
10. **G-local:** `make pre-add-check` on the three Go files: clean; `make preflight`: passed; `go test ./...` and `make
    race`: 38 ok; `make perf` passed at load 113: ReplyRender 48.41 µs, Reprint 6.65 ms, LineSessionStart 1.84 ms,
    KeyDraw 1.947 µs.
11. **G-hosts,** 809 files shipped to each: `go test` and race passed on both, 38 ok; `make check-records markdownlint`
    passed on the Linux machine, and on the Windows laptop on its rerun (deviation 2); preflight stops at
    `verify-build-metadata`, as on every copy; `make perf` passed on both (the Linux machine ReplyRender 170.79 µs,
    Reprint 18.59 ms, LineSessionStart 4.23 ms, KeyDraw 6.108 µs; the Windows laptop 28.39 µs, 3.82 ms, 5.18 ms,
    1.245 µs); the conhost probe on the Windows laptop: passed.
