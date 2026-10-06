# REPORT 0009 — magic-cli-remote findings from gobble work

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

This is a running report. It decides nothing. It collects what work in this repository has found about magic-cli-remote, so the owner can review it there later. The owner keeps changes to magic-cli-remote out of this project's scope (2026-10-04), so nothing here has been changed in that repository.

**Rule for maintaining it (owner, 2026-10-05):**

- Every magic-cli-remote discovery made while working on gobble is added here, dated, in the same change that discovers it.
- Each finding names the magic-cli-remote file, line and commit, and the gobble record that found it.
- A finding is never deleted. When it is resolved upstream, mark it resolved, with the date and the commit.

## Baseline

magic-cli-remote `main` was at `9778cbc1` (2026-10-04) for every finding below, unless a finding says otherwise. `git log 9778cbc1..HEAD` was empty on 2026-10-05.

## Index

| ID | Date | Kind | Summary | Status |
|---|---|---|---|---|
| M1 | 2026-10-04 | contract | The provider record is named `pigo`, not `gobble` | open |
| M2 | 2026-10-04 | contract | 0179 D2 and D3 (operation reporting, strict fallback) are not implemented | open |
| M3 | 2026-10-04 | companion | Companion changes gobble needs, listed in 0008-MADR D20 | open |
| M4 | 2026-10-04 | dependency | Delivery behaviour gobble's ACP stream is built around | reference |
| M5 | 2026-10-05 | design defect | 0157-MADR D6's collision rule can delete the newest log backup | open |
| M6 | 2026-10-05 | test quality | Two `appdirs` tests skip depending on host state | open |
| M7 | 2026-10-05 | contract | The version pin ignores prerelease, so any prerelease or pseudo-version build matches a release pin | open |
| M8 | 2026-10-05 | dependency defect | The pinned ACP SDK fork's `SetLogger` races with the connection's reader goroutine | open |

## Findings

### M1 — the provider record is named `pigo` (2026-10-04)

- **Found by:** [0008-MADR](../decisions/0008-MADR-native-cli-mode.md) F22 and measurement 13.
- **Fact:**
  - The record is `docs/decisions/0179-MADR-pigo-native-acp-provider.md` (`status: proposed`), created as `pigo` in `72588004`.
  - It names `provider.IDPigo`, wire id `"pigo"`, `DefaultBin: "pigo"` and the `_pigo/compact`, `_pigo/usage`, `_pigo/set_session_name` and `_pigo/fork` methods (`0179…:250-271`).
  - "gobble" occurs nowhere in magic-cli-remote's records or Go code.
- **Effect on gobble:** [0002-MADR](../decisions/0002-MADR-cli-acp-headless-mcp-v1.md)'s fifth amendment cited a nonexistent `0179-MADR-gobble-native-acp-provider.md`. It was corrected in gobble by a dated amendment (0008-PLAN P0).
- **Wanted upstream:** rename the provider to gobble, with `_gobble/*` methods (M3).

### M2 — 0179 D2 and D3 are not implemented (2026-10-04)

- **Found by:** 0008-MADR measurement 13.
- **Fact:**
  - `command.Table` is a plain map (`internal/command/command.go:149`).
  - Neither `strict` nor `OpReporter` occurs in `internal/`.
- **Effect on gobble:** until these land, a `KindNative` row falls back to grok vendor calls (0002-MADR, fifth amendment). gobble must not be registered as a native provider before then.

### M3 — companion changes gobble needs (2026-10-04)

- **Found by:** [0008-MADR](../decisions/0008-MADR-native-cli-mode.md) D20, which records these items and does not make them. Each needs a record in magic-cli-remote first.
- **Items:**
  1. Rename 0179's provider to gobble: `IDGobble`, wire id `"gobble"`, `DefaultBin: "gobble"`, and `_gobble/*` methods in place of `_pigo/*` (M1).
  2. Implement 0179 D2 and D3 before gobble is registered (M2).
  3. Implement 0179 D5: session title, `usage_update.cost`, tool `locations` and diff text, and config categories and groups.
  4. Implement 0179 D6 and D7: `ConfigureSession` after `session/load`, the model catalog from `configOptions`, and `DangerousModeIDs`.
  5. Implement the deferred items in 0179's "More Information": the `!` gate, a steer path, stdio MCP, `clientInfo` in `initialize`, per-provider environment, and question delivery.
  6. Size-check outbound broadcast and replay (0181), and dedupe replayed events by message identity rather than exact text.

### M4 — delivery behaviour gobble depends on (2026-10-04)

- **Found by:** 0008-MADR measurement 13. gobble's D19 rules are built on these facts, so a change to any of them upstream must be checked against D19.

| Behaviour | Where |
|---|---|
| The SDK fork's inbound queue holds 1,024 notifications with `OverflowDropNewest`, and there is no re-delivery | `acpagent/session.go:582-603`; 0167-MADR D17, D18 |
| `usage_update` and `available_commands` are dropped when the events channel is full; chunks are retried every 50 ms | `session.go:1263-1305`, `:1427-1437` |
| Chunks are coalesced in 80 ms windows with an 8 KiB cap; non-terminal `tool_call_update`s per tool are merged | `internal/chunkbuf/chunkbuf.go:28-33`, `:84-110` |
| Tool summaries are cut to 400 bytes (`tool_call`) and 600 (updates); a diff becomes `"diff <path>"` | `session.go:1668-1705`, `:2556-2609` |
| Only ungrouped `select` options and booleans are forwarded, without the category | `session.go:1953-1958` |
| Replay is deduped on (type, text, tool id, agent session id); the 256-slot channel drops the oldest before the pump attaches | `internal/session/manager.go:270-272`, `:369-401`; `session.go:1446-1461` |
| The phone's context chip turns at 75% and 90% | `apps/mobile/lib/features/chat/chat_screen.dart:3326-3380` |
| One process per session plus a warm spare; `initialize` and `session/new` 30 s, `session/load` 120 s; close bounded at 2 s, then the process group is killed | `acpagent.go:157-162`, `:791-794`, `:810-817` |
| `agentInfo.version` is compared on major.minor.patch; an unparseable version never matches | `internal/provider/version.go:8-40`; `acpagent/version.go:18-56` |

### M5 — 0157-MADR D6's collision rule can delete the newest backup (2026-10-05)

- **Found by:** 0004-PLAN Phase 2 step 4 (execution record of 2026-10-05). gobble's rolling log sink is written from 0157's design.
- **Upstream text:** D6 names backups `<base>-<UTC timestamp>.log` and "appends `-1`, `-2`, … before the extension until the name is free", then prunes by count after each rotation (`docs/decisions/0157-MADR-daemon-owned-rolling-logs.md:349-358`).
- **Defect:**
  - Suppose several rotations fall in one millisecond and pruning runs between them.
  - Pruning removes the oldest backup, which is the unsuffixed name.
  - The next rotation finds that name free and reuses it.
  - The newest backup now carries the lowest suffix, sorts as the oldest, and is deleted by the next prune.
- **Evidence:** gobble's `TestCollisionGetsASuffix` performs eight rotations in one millisecond with five backups kept. With "first free" naming the run kept backups `-1` to `-5`, so the three newest were lost.
- **Fix used in gobble:** number after the highest suffix already used for that timestamp (`internal/logging/rolling.go`, `backupName`).
- **Status upstream:** 0157 is `proposed` and its rotator is not implemented at `9778cbc1`: `internal/logging` holds only `slog.go`. The defect is in the design, and can be fixed before the code is written.

### M6 — two `appdirs` tests skip depending on host state (2026-10-05)

- **Found by:** 0004-PLAN Phase 2 step 3, while copying `internal/appdirs` into gobble.
- **Fact:**
  - `internal/appdirs/converge_test.go:44` skips when "the host created this file owner-only already".
  - `internal/appdirs/security_windows_test.go:209` skips when the host cannot resolve the `LA` alias.
- **Effect:** on such hosts the property each test pins goes unchecked, and the suite still reports success. gobble keeps only the deterministic cases. Its copy of the `LA` test fails instead of skipping, and it passed on this Windows host.
- **Wanted upstream (suggestion):** build the non-private fixture explicitly, rather than relying on the host's temp ACL, and fail when the alias cannot be resolved.

### M7 — the version pin ignores prerelease (2026-10-05)

- **Found by:** 0008-PLAN P4. gobble's first built binary reported a Go VCS pseudo-version.
- **Fact:** `agentInfo.version` is compared on major.minor.patch, "ignoring a leading `v` and any pre-release or build suffix" (`internal/provider/version.go:8-40`; `acpagent/version.go:18-56`; M4).
- **Effect:** any build whose version has a prerelease part matches the release with the same core.
  - `1.3.0-rc.1` matches a `1.3.0` pin.
  - After a `v1.2.3` tag, a Go pseudo-version for a dev build is `1.2.4-0.<time>-<hash>`, and it matches a `1.2.4` pin it never was.
- **What gobble does:** every build that is not a clean tag reports `0.0.0-dev+<revision>[.dirty]`, which matches no release pin (0004-PLAN Phase 2 step 2, deviation of 2026-10-05). A deliberate prerelease tag (`v1.3.0-rc.1`) is still reported as itself, and so still matches `1.3.0` upstream.
- **Wanted upstream (suggestion):** compare the full SemVer, or treat a prerelease as below its release when checking a pin.

### M8 — the pinned ACP SDK fork's `SetLogger` races with its reader (2026-10-05)

- **Found by:** 0002-PLAN Phase 2, WSL `go test -race ./...` (deviation of 2026-10-05).
- **Why it is here:**
  - magic-cli-remote pins this fork: `go.mod:64-68`, `replace github.com/coder/acp-go-sdk => github.com/maccavelli/acp-go-sdk v0.13.6-mcr.1`, citing MADR 0167 D16–D20.
  - gobble carries the same `replace`.
- **Fact, in the fork at `v0.13.6-mcr.1`:**
  - `SetLogger` is a plain field write (`connection.go:236`).
  - `loggerOrDefault` reads that field (`connection.go:239`) on the receive goroutine, for example from `shutdownReceive` at end of input (`connection.go:609`).
  - `NewConnection` starts that goroutine (`connection.go:212`) before a caller can call `SetLogger`. There is no option to set the logger at construction.
- **Effect:**
  - Any caller that sets a logger races with the reader. The race detector reports it when the input ends early: gobble saw 1 race in 30 runs on its Phase 1 tree.
  - magic-cli-remote at `9778cbc1` calls `SetLogger` nowhere (0 matches), so it uses `slog.Default()` and is not exposed today. A future call would be.
- **What gobble does:** the owner decided on 2026-10-05 to fix the fork first. gobble's 0002-PLAN Phase 2 waits for the fixed tag, then sets its logger through the option.
  - *2026-10-05:* the fork's `v0.13.6-mcr.2` (`565efea`) carries the fix: `WithLogger` and an atomic `SetLogger`. gobble pins it.
  - magic-cli-remote still pins `mcr.1`, so this stays open until its `replace` moves.
- **Wanted in the fork (suggestion):**
  - a `WithLogger(*slog.Logger)` `ConnectionOption` applied before the goroutines start;
  - an atomic `SetLogger`, so existing callers are safe too.

  Whether upstream `coder/acp-go-sdk` has the same defect after v0.13.5 is **[unverified]**.
