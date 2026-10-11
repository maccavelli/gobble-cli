---
status: proposed
date: 2026-10-10
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# PLAN 0014 — Move the ACP SDK from the fork to acp-go-sdk2 v0.14.0

Implements [0014-MADR-move-to-acp-go-sdk2.md](0014-MADR-move-to-acp-go-sdk2.md) decisions D1–D8, closing findings F1–F8.

## Goal

- `go list -m all` names `github.com/maccavelli/acp-go-sdk2 v0.14.0` and neither `github.com/coder/acp-go-sdk` nor `github.com/maccavelli/acp-go-sdk`.
- No `.go` file imports `github.com/coder/acp-go-sdk`.
- `acpserver/testdata/session.golden` is byte-identical to `HEAD`'s, and `TestPromptParamsMatchSDKClient` passes against sdk2's own client.
- `session/delete` and `elicitation/create` answer -32601; nothing new is advertised.
- Every gate of 0005-PLAN F1d-2's deviation 4 passes: lint for linux, darwin and windows; Windows `go test ./...`; `CGO_ENABLED=1 go test -race ./...` on Windows and on the second amd64 Linux machine; Windows and WSL `make preflight`.

## Scope

### In scope (the only files any phase may touch)

| | Files |
| :--- | :--- |
| P0 | none in the tree; one patch file in the session scratchpad |
| module | `go.mod`, `go.sum` |
| imports only (D2) | every `.go` file importing `github.com/coder/acp-go-sdk`: 42 at `HEAD`, under `acpserver/`, `acpclient/` and `internal/archtest/` |
| code (D3, D4) | `acpserver/agent.go`, `acpserver/clientterm.go`, `acpserver/prompt.go`; `acpclient/conn.go`, `acpclient/update.go`; `acpclient/acptest/client.go`, `acpclient/acptest/script.go`, `acpclient/acptest/pair_test.go` |
| test files (F4) | any `_test.go` under `acpserver/` or `acpclient/` whose only change is a D3 method on a test double or a D4 conversion, each named in the execution record |
| new tests | `acpserver/delete_test.go`; `acpclient/elicitation_test.go`; `acpserver/convert_test.go` and `acpclient/convert_test.go` for the conversion helpers |
| gates (D6) | `internal/archtest/check.go`, `internal/archtest/check_test.go` |
| records and docs (D6, D7) | this pair; `docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md` and `docs/decisions/0004-MADR-go-module-architecture.md` (one dated note each); `docs/decisions/0005-PLAN-v1-feature-scope.md` (F3-4 deviation 3, written before this pair); `docs/architecture.md`; `README.md`; `docs/README.md` |

### Out of scope

- **Using anything new in sdk2:** `session/delete`, elicitation, `UnstableMessageMcp`, the schema's other additions. Each is a feature, with its own record.
- **F3-4 itself.** It resumes after this plan, as its own sub-phase (D8).
- **magic-cli-remote's own SDK.** It stays on the fork; this repository does not change it.
- **Any change to gobble's exported types,** such as `acpclient.Usage` or `acptest.Terminals.ExitCode` (D4).

## Stability rule

Every phase ends with:

```bash
export GOTOOLCHAIN=go1.27.2
GO_PRECHECK_SKIP_VULN=1 make pre-add-check FILES="<the phase's .go files>"
go test ./...
make preflight
```

and the gates of 0005-PLAN F1d-2's deviation 4. The agent stages with `git add` and never commits; the owner commits with `git commit --no-edit`. No push and no tag without the owner's explicit request in that turn.

## Cross-cutting contracts

- **C1. The wire does not change** (D5). `session.golden` is never regenerated in this plan. A difference stops the phase and goes to the owner as a deviation.
- **C2. Only `acpserver` and `acpclient/...` import the SDK** (0004-MADR rule 2), before and after.
- **C3. Every new check is seen failing first,** on a scratch copy, never by dirtying the tree.
- **C4. No value is wrapped by a conversion.** A signed value converted to an unsigned field is range-checked: a negative count becomes 0, a value above the field's range its maximum, and a line number out of range is left out, as a line of 0 is now.
- **C5. Nothing is advertised that does not work** (0002-MADR's honest-advertisement rule): no `sessionCapabilities.delete`, no client `elicitation`.

C1 is the contract most at risk: a golden diff from a field order or an `omitempty` change is easy to "fix" by regenerating the file, and that would hide exactly the incompatibility this plan must catch.

## Dependency and delivery order

P0, then P1. P1 is one commit, because the tree does not build between its steps. F3-4 resumes after P1 is committed.

## Implementation Steps

### P0 — Set F3-4's work in progress aside (D8)

The tree holds F3-4's uncommitted edits to `acpclient/conn.go` and `acpclient/conn_test.go`: `Prompt.Context`, `unscanned`, and `TestPromptContext`. P1 changes `acpclient/conn.go` too, so they must not ride in P1's commit.

1. `git diff -- acpclient/conn.go acpclient/conn_test.go > <scratchpad>/f34-wip.patch`. Expect a non-empty file. Then `git apply --check -R <scratchpad>/f34-wip.patch` exits 0.
2. **Only with the owner's explicit approval in that turn,** since it discards uncommitted lines, though the patch keeps them: `git restore acpclient/conn.go acpclient/conn_test.go`. Expect `git status --short` to list only the record files.
3. F3-4 re-applies the patch with `git apply` when it resumes, after P1's commit. That is F3-4's step, not this plan's.

**Verification:** `git diff --stat -- acpclient/` is empty; the patch file exists and reverse-applies cleanly against the pre-restore tree.

### P1 — The move (D1–D7; closes F1–F8)

1. **The module (D1).**
   - Edit `go.mod`: delete the line `github.com/coder/acp-go-sdk v0.13.5` and the line `replace github.com/coder/acp-go-sdk => github.com/maccavelli/acp-go-sdk v0.13.6-mcr.2`.
   - Run `go get github.com/maccavelli/acp-go-sdk2@v0.14.0`.
   - Expect `go.sum` to gain `github.com/maccavelli/acp-go-sdk2 v0.14.0 h1:Db0ojAcFSQH5KeEdfxHwJKxJbis+xLgtHMZaqz655Eg=` and its `go.mod` line, `h1:KqDRvShTbZEOpKOQOyRUnbeFqlyH20LXi7k3UOsgv0I=`.
2. **The imports (D2).** A scratchpad script replaces `"github.com/coder/acp-go-sdk"` with `"github.com/maccavelli/acp-go-sdk2"` in every `.go` file. It asserts 42 files change and that no file still names the old path, `internal/archtest`'s strings excepted, which step 4 edits.
3. **The code (D3, D4).** The SDK's new types are converted where gobble meets them:
   - `acpserver/agent.go`: `DeleteSession` returns `acp.DeleteSessionResponse{}` and `acp.NewMethodNotFound(acp.AgentMethodSessionDelete)`. `Capabilities()` is unchanged.
   - `acpserver/clientterm.go`: `acp.TerminalId(id)` in each request; `string(resp.TerminalId)`; `OutputByteLimit` as a `uint64` inside its existing `> 0` branch; exit codes through `exitCode(*uint32) *int`.
   - `acpserver/prompt.go`:
     - `jsonRaw` returns `stdjson.RawMessage`, nil for empty input, so `omitempty` still drops it;
     - `acpUsage` and `usedUpdate` convert through `count(int64) uint64` (C4);
     - `acpLocations` sets a line through `line(int) *uint32`, nil when out of range (C4).
   - `acpserver/history.go` needs no edit beyond step 2: its `jsonRaw` call takes the new return type.
   - `acpclient/conn.go`:
     - `handler.CreateElicitation` returns `acp.NewMethodNotFound(acp.ClientMethodElicitationCreate)`;
     - `handler.CompleteElicitation` returns nil;
     - `Usage` is filled through `toInt(uint64) int`, which clamps (C4).
   - `acpclient/update.go`: `Terminal: string(c.Terminal.TerminalId)`.
   - `acpclient/acptest/client.go`:
     - `Client` gains the two elicitation methods, as the handler has them;
     - `Terminals.exit` returns `*uint32`, through a range check that answers an internal error for a negative `ExitCode`.
   - `acpclient/acptest/script.go` and `acpclient/acptest/pair_test.go`: `ScriptAgent.DeleteSession` and `stubAgent.DeleteSession` answer -32601.
   - **Then compile the tests,** `go test -count=1 -run '^$' ./...`. Each further error in a test file that is a D3 method on a test double or a D4 conversion is fixed in place and named in the execution record. Any error of another kind stops the phase as a deviation.
4. **The gates (D6).**
   - `internal/archtest/check.go:186` and `:208` name `github.com/maccavelli/acp-go-sdk2`.
   - `check_test.go`'s rule 2 cases import it.
   - 0004-MADR's rule 2 (`:430`) gets an inline dated note, with the old path struck through. 0002-MADR's pin (`:365`) gets a dated note pointing to 0014-MADR.
5. **The docs (D7).**
   - `docs/architecture.md:104`: the dependency row reads `github.com/maccavelli/acp-go-sdk2` `v0.14.0`.
   - `README.md:78`: the protocols line names sdk2 at `v0.14.0`.
   - `docs/README.md`: the pair's rows read their new status, and the "Run:" and "Next:" lines name P1 and F3-4. The rows themselves were added with this pair.
6. **The new tests:**
   - `acpserver/delete_test.go`, `TestDeleteSessionNotFound`: `session/delete` answers -32601, and `initialize` advertises no `sessionCapabilities.delete`.
   - `acpclient/elicitation_test.go`, `TestElicitationNotFound`: an agent's `elicitation/create` answers -32601; an `elicitation/complete` notification changes nothing; `initialize`'s client capabilities carry no `elicitation`.
   - `acpserver/convert_test.go` and `acpclient/convert_test.go`: `count`, `line`, `exitCode` and `toInt` at 0, at their limits, beyond them and negative.
7. **The wire (D5, C1).**
   - `go test ./acpserver -run '^TestSession$'`, then `git diff --exit-code acpserver/testdata/session.golden` exits 0.
   - `go test ./acpclient -run '^TestPromptParamsMatchSDKClient$'` passes.
   - Any golden difference stops the phase as a deviation.
8. **C3 negatives, on scratch copies, each mutation asserted to land once:**
   - `DeleteSession` returning success fails `TestDeleteSessionNotFound`;
   - `CreateElicitation` returning a response fails `TestElicitationNotFound`;
   - each conversion helper wrapping instead of clamping fails its test;
   - `jsonRaw` returning `[]byte("null")` for empty input fails `TestSession` on the golden;
   - `internal/cli` importing sdk2 fails `make archtest` rule 2;
   - `Capabilities()` advertising `Delete` fails `TestDeleteSessionNotFound`.
9. **Close:** the pre-add check on every changed `.go` file; the stability rule; `python3 scripts/check_records.py --check-all`; the identifier scan; stage.

**Verification:** the Confirmation block of 0014-MADR, every line as it states.

## Verification (whole plan)

### Acceptance criteria (mapped to MADR Confirmation)

| # | Criterion | MADR |
| :--- | :--- | :--- |
| A1 | `go list -m all` has sdk2 `v0.14.0` and no fork or coder module | D1 |
| A2 | No `.go` file imports `github.com/coder/acp-go-sdk` | D2 |
| A3 | `session/delete` and `elicitation/create` answer -32601; nothing new advertised | D3, C5 |
| A4 | Conversions clamp and never wrap; their tests cover the limits | D4, C4 |
| A5 | `session.golden` byte-identical; `TestPromptParamsMatchSDKClient` passes | D5, C1 |
| A6 | `make archtest` passes, and fails for an `internal/cli` import of sdk2 | D6, C2 |
| A7 | Docs and the two notes name sdk2 | D6, D7 |
| A8 | Every gate passes; every negative seen failing | Stability rule, C3 |

A5 is the one most likely to be dropped under pressure: regenerating the golden makes the suite green at once, and it would erase the only check that the wire is unchanged.

## Rollout and Rollback

- **Rollout:** P1's commit, by the owner.
- **Rollback:** revert that commit. The fork stays available in the module proxy at `v0.13.6-mcr.2`, and the revert restores its `replace`.

## Deferred (named, so they are not mistaken for oversights)

- **Later sdk2 tags.** sdk2 is under active development (F7); each move is an amendment of 0014-MADR, with the same wire check.
- **`session/delete`.** sdk2 makes it a stable `Agent` method; gobble's session deletion belongs to 0005-MADR's session features, not here.
- **Elicitation.** A client-side feature for the TUI and the CLI, with its own record.
- **magic-cli-remote's move off the fork.** That repository's decision.
