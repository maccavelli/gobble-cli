---
status: superseded by 0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md
date: 2026-10-29
associated-madr: "0034-MADR-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement pinning acp-go-sdk2 v1.21.0 and adopting its SessionUpdate helpers

*Superseded 2026-10-29, never executed, by [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md). Its D2–D5 are that record's D7–D9, at acp-go-sdk2 `v1.26.0`;
the work is [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md)'s P7–P10.*

Associated MADR: [0034-MADR-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md](0034-MADR-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md)

## Goal

gobble pins acp-go-sdk2 `v1.21.0`; the five `acpserver` `SessionUpdate` literals are the SDK helpers; `acpclient.handler`
embeds `ClientRefusals`; `SessionUpdate.Kind()` names variants that are not a `Kind*` case. `session.golden` is
byte-identical. 0016 D8–D10 stay unbuilt.

Closes [gobble-cli#5](https://github.com/maccavelli/gobble-cli/issues/5).

## Scope

- **In:**
  - `go.mod`, `go.sum`;
  - `acpserver/commands.go`, `acpserver/modes.go`, `acpserver/prompt.go`;
  - `acpclient/conn.go`, `acpclient/update.go`, and their tests;
  - `acpserver/testdata/session.golden` (must not change);
  - records: this pair; dated notes in 0016-MADR (D5, D13), 0031-MADR (D1, D9), 0023-MADR and 0023-PLAN (the pin
    moved); `docs/architecture.md`; `README.md`; `docs/README.md`.
- **Out:** 0016 D8–D10; `_gobble/*`; `session/delete`; elicitation UI; advertising fs/terminal; `acptest` agent
  stubs; unstable constructors; 0031-PLAN P4b; 0033; go-llmprovider-sdk and go-selfupdate-lib pins.

## Order

This pair does not wait on 0031-PLAN P4b, 0029-PLAN P2, 0032-PLAN or 0033-PLAN (MADR F9, Q3). P1 to P4 in order.
The plan stays `proposed` until the owner answers Q1–Q3 and approves execution.

This PLAN assumes the MADR's recommended answers: Q1 pin now; Q2 `Kind()` in this pair; Q3 do not wait on P4b.
An answer otherwise amends this PLAN before the step it changes: Q1 changes P1, Q2 drops or keeps P4, Q3 only
the start date.

## Implementation Steps

### P1 — the pin (D1)

1. `make check-libs` for acp-go-sdk2 at `v1.21.0` (`scripts/check_libs.py --lib acp-go-sdk2 --tag v1.21.0`), on a
   scratch clone of this repository outside the working tree. Record get/tidy, version, other modules, build, vet
   for linux/darwin/windows, tests of `./acpclient/...` and `./acpserver/...`, api against `v1.20.0`, govulncheck
   and advisories. Expect api "0 incompatible, 11 compatible" and no other module moving.
2. `go get github.com/maccavelli/acp-go-sdk2@v1.21.0`, `go mod tidy`. No code change is expected; one that is
   needed is a deviation.
3. Records: `docs/architecture.md` pin table; `README.md` stack; dated notes in 0023-MADR, 0023-PLAN and 0031-MADR
   D1 that the acp pin moved here; `docs/README.md`.
4. Gates (Verification). Commit.

### P2 — constructors (D2, D3)

1. Replace, and only these:

   | Site | Becomes |
   | :--- | :--- |
   | `acpserver/commands.go` `setConfigUpdate` | `acp.UpdateConfigOptions(opts)` |
   | `acpserver/commands.go` `setNameLocked` | `acp.UpdateSessionTitle(acp.PatchValue(name))` |
   | `acpserver/commands.go` `startFrames` | `acp.UpdateAvailableCommands(availableCommands(templates))` |
   | `acpserver/modes.go` `modeUpdate` | `acp.UpdateCurrentMode(acp.SessionModeId(current))` |
   | `acpserver/prompt.go` `contextUpdate` | `acp.UpdateUsage(used, size)`, then `Cost` on `UsageUpdate` when priced |

2. `git diff --exit-code acpserver/testdata/session.golden`. A diff stops the phase (D3).
3. A grep (or equivalent test) of non-test `acpserver` `*.go` finds no
   `SessionUpdate{AvailableCommandsUpdate:`, `CurrentModeUpdate:`, `ConfigOptionUpdate:`, `SessionInfoUpdate:`,
   `UsageUpdate:`.
4. Dated note in 0016-MADR D5 / D13: the five send sites use the helpers at `v1.21.0`.
5. Planted breaks, each in its own scratch copy, never in the tree:
   - leave `setConfigUpdate` as a composite literal: the grep/test fails;
   - alter `session.golden` by one byte: `TestSession` (or the golden helper) fails.
6. Gates. Commit.

### P3 — `ClientRefusals` (D4)

1. Embed `acp.ClientRefusals` on `acpclient.handler`. Delete the nine stub methods. Keep `SessionUpdate` and
   `RequestPermission`. Keep `var _ acp.Client = (*handler)(nil)`.
2. Existing tests that assert `-32601` for fs/terminal/elicitation still pass. `CompleteElicitation` still returns
   nil.
3. Planted breaks, scratch copies:
   - add `SessionUpdate` to `ClientRefusals` usage by forgetting gobble's method: `var _ acp.Client` fails to
     compile;
   - restore a `ReadTextFile` that returns success: the method-not-found test fails.
4. Dated note in 0031-MADR D9: the nine stubs are gone.
5. Gates. Commit.

If Q2 is no, stop after P3 except the records close in P4's records step without `Kind()`.

### P4 — `Kind()` (D5)

1. Where `translate` falls through as `KindOther`, tests may assert `n.Update.Kind()` equals the wire
   discriminator (`config_option_update` in `acpclient/conn_test.go`). Production `Kind` constants do not grow.
2. Do not add `KindNotice` or a stable `KindCompaction`.
3. Planted break: a `config_option_update` fixture whose `Kind()` assertion expects `usage_update` fails.
4. Dated notes: 0016-MADR D13 and 0031-MADR D9 this pair complete; issue 5 closed when the owner asks in that
   turn. `docs/README.md`. This PLAN `complete`.
5. Gates. Commit.

## Verification

- Every new check seen failing first on a planted break in a scratch copy; the failure quoted in the execution
  record.
- Each phase:
  - `make pre-add-check` on its Go files;
  - on macOS, the Linux machine and the Windows laptop: the tests, the race detector, `make preflight` (on the
    copies, to `verify-build-metadata`), `make check-records markdownlint`.
- P1: `make check-libs` report for `v1.21.0`.
- P2: `session.golden` identical; no remaining composite literals in non-test `acpserver`.
- P3: `handler` embeds `ClientRefusals`; nine methods gone; `-32601` tests pass.
- P4: `Kind()` used for the `KindOther` discriminator; no new `Kind*` constants.
- Identifier scan of staged files: no hostname, account, or real-machine absolute path.
- No test reaches a real service.

## Rollout and Rollback

One commit per phase. Reverting P4 restores pointer-only `translate` tests. Reverting P3 restores the nine stubs.
Reverting P2 restores the literals. Reverting P1 needs P2–P4 reverted first, since they name `v1.21.0` symbols.

## Execution record

Empty until the owner answers Q1–Q3 and approves execution.
