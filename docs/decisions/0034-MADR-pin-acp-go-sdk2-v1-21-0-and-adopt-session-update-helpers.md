---
status: superseded by 0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md
date: 2026-10-29
decision-makers: repository owner
consulted: none
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Pin acp-go-sdk2 v1.21.0, replace SessionUpdate literals with its helpers, and embed ClientRefusals

*Superseded 2026-10-29, never executed, by [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md). Its D2–D5 are that record's D7–D9, at acp-go-sdk2 `v1.26.0`;
the work is [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md)'s P7–P10.*

## Context and Problem Statement

gobble pins `github.com/maccavelli/acp-go-sdk2 v1.20.0` (`go.mod`, `docs/architecture.md`). On 2026-10-25 that
module tagged `v1.21.0` (`7f3b210`). The tag implements
[acp-go-sdk2#11](https://github.com/maccavelli/acp-go-sdk2/issues/11): constructors for every stable
`SessionUpdate` variant that `helpers.go` did not already build, `SessionUpdate.Kind()`, and `ClientRefusals`.
The schema is still `1.25.0`; `make api` against `v1.20.0` is "0 incompatible, 11 compatible".

[gobble-cli#5](https://github.com/maccavelli/gobble-cli/issues/5) is gobble's consumption of that surface.
[0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md) waits for the library and drops the
copy when the library has the thing. [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md)
D13 asked for the constructors; [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md)
D9 left them unfiled, then a note of 2026-10-25 named this pair as 0034.

This record decides the pin and which copies gobble drops. It does not send notices or stable compaction
updates (0016 D8–D10), and it does not implement constructors in this repository.

### What was measured, 2026-10-25

- **The sibling tag.** Local clone of acp-go-sdk2: `git describe --tags --abbrev=0` is `v1.21.0`, 0 commits
  ahead of the tag, schema `version` still `1.25.0`. CI on the tag completed successfully. The GitHub *release*
  for `v1.21.0` is unpublished; the annotated tag exists. The module version is the tag.
- **gobble's pin.** `go list -m github.com/maccavelli/acp-go-sdk2` is `v1.20.0`. go-llmprovider-sdk `v1.22.2`
  and go-selfupdate-lib `v1.22.0` are unchanged by this record.
- **The five send-site literals** (non-test `acpserver`):

  | Site | Literal |
  | :--- | :--- |
  | `acpserver/commands.go` `setConfigUpdate` | `SessionUpdate{ConfigOptionUpdate: &SessionConfigOptionUpdate{ConfigOptions: opts}}` |
  | `acpserver/commands.go` `setNameLocked` | `SessionUpdate{SessionInfoUpdate: &SessionSessionInfoUpdate{Title: PatchValue(name)}}` |
  | `acpserver/commands.go` `startFrames` | `SessionUpdate{AvailableCommandsUpdate: &SessionAvailableCommandsUpdate{AvailableCommands: …}}` |
  | `acpserver/modes.go` `modeUpdate` | `SessionUpdate{CurrentModeUpdate: &SessionCurrentModeUpdate{CurrentModeId: …}}` |
  | `acpserver/prompt.go` `contextUpdate` | `SessionUpdate{UsageUpdate: u}` after setting `Used`, `Size`, optional `Cost` |

  Message, thought, plan and tool-call already use `UpdateAgentMessageText`, `UpdatePlan`, `StartToolCall`
  (`acpserver/history.go:328-335`). Those stay.
- **The nine client stubs.** `acpclient/conn.go` `handler` advertises no fs, terminal or elicitation. Each of
  `ReadTextFile`, `WriteTextFile`, `CreateTerminal`, `KillTerminal`, `TerminalOutput`, `ReleaseTerminal`,
  `WaitForTerminalExit`, `CreateElicitation` returns `NewMethodNotFound(ClientMethod…)`. `CompleteElicitation`
  returns nil. `var _ acp.Client = (*handler)(nil)`. `SessionUpdate` and `RequestPermission` stay gobble's.
- **`translate`.** `acpclient/update.go` switches on pointer fields for agent text, thought, tool call, plan,
  usage, commands, mode and session info. `config_option_update` is `KindOther` (`acpclient/conn_test.go`).
  Notice, compaction and unknown have no `Kind*` constant. `KindCompaction` is `_gobble/compaction`, not the
  stable session update.
- **The SDK surface at `v1.21.0`**, from acp-go-sdk2's
  `0036-MADR-session-update-constructors-kind-and-client-refusals.md` and `helpers.go` / `refusals.go`:
  `UpdateAvailableCommands`, `UpdateCurrentMode`, `UpdateConfigOptions`, `UpdateSessionTitle(Patch[string])`,
  `UpdateUsage(used, size uint64)`, `UpdateNotice`, `UpdateCompaction`, `UpdateCompactionSummary` /
  `UpdateCompactionSummaryText`, `SessionUpdate.Kind()`, `ClientRefusals`. New constructors set the variant's
  `SessionUpdate` discriminator field; the six older helpers still leave it empty. `MarshalJSON` fills it either
  way, so the wire is the same; `reflect.DeepEqual` is not. `ClientRefusals` implements only the nine gated
  methods. Magic must not embed it whole (real fs/terminal); gobble's CLI client should.

### Findings

- **F1.** `v1.21.0` is additive on the v1 module path. Schema 1.25.0 is unchanged. A pin-only bump is expected
  to compile and pass tests unchanged, as 0031-PLAN P1 did for `v1.20.0`.
- **F2.** The five literals are the remainder of 0016 D13's send path. Replacing them is a local-wrapper drop,
  not a change to what gobble sends.
- **F3.** `UpdateUsage` takes `used` and `size`. `Cost` is still assigned on the returned variant, as today.
- **F4.** `UpdateSessionTitle` takes `Patch[string]`. gobble already uses `PatchValue(name)`.
- **F5.** Embedding `ClientRefusals` on `handler` and deleting the nine methods keeps `SessionUpdate` and
  `RequestPermission` compile-required. Tests that assert `-32601` keep asserting the wire code.
- **F6.** `Kind()` names a declined or `KindOther` discriminator without a second unmarshal. Adding
  `KindNotice` or a stable `KindCompaction` is 0016 D10, not this record.
- **F7.** `acpserver/testdata/session.golden` must stay byte-identical (0016 D5). A golden diff is a wire
  change and stops the phase.
- **F8.** The SDK's GitHub release for `v1.21.0` is unpublished. The tag is the module version. Waiting on
  release notes does not change what `go get` fetches.
- **F9.** 0031-PLAN P4b (ChatGPT plan), 0029-PLAN P2, 0032-PLAN adoptions and 0033 (docs CI) do not import
  these names. This pair does not wait on them.

## Decision Drivers

* 0004-MADR: wait for the library; drop the copy when it has the thing.
* 0016-MADR D5 / D13: constructors, not local wrappers; `session.golden` byte-identical.
* 0031-MADR D9: the helpers were "not now" until an issue existed; it does, and the tag shipped.
* gobble-cli issue 5: pin the tag, replace the five literals, embed `ClientRefusals` if it shipped, use `Kind()`
  for unnamed variants.
* No protocol change: schema 1.25.0, additive API.
* Each phase small enough to land, verify and revert on its own.

## Considered Options

* A. Pin `v1.21.0` only, as 0031-PLAN P1 did for `v1.20.0`.
* B. Pin `v1.21.0`, and adopt the constructors, `ClientRefusals` and `Kind()`, phase by phase, leaving 0016 D8–D10
  and the rest of D13 (session/delete, elicitation UI, images) with their owners.
* C. Wait until acp-go-sdk2 publishes a GitHub release for `v1.21.0`.
* D. Also implement 0016 D8–D10 (when gobble sends notices and stable compaction updates) in this pair.

## Decision Outcome

Chosen option: **"B. Pin `v1.21.0`, and adopt the constructors, `ClientRefusals` and `Kind()`, phase by phase"**,
because the tag is the module version, the copies are the ones issue 5 named, and D8–D10 is still a product
choice about what gobble sends.

- **D1. Pin `github.com/maccavelli/acp-go-sdk2 v1.21.0`, exactly**, after `make check-libs` for that tag on a
  scratch clone (0019-MADR D14). `go get` and `go mod tidy`. No other module moves. go-llmprovider-sdk and
  go-selfupdate-lib stay at 0031 D1's pins. The unpublished GitHub release is not a gate (F8).
- **D2. Replace the five `acpserver` literals with the SDK helpers:** `UpdateConfigOptions`, `UpdateSessionTitle`,
  `UpdateAvailableCommands`, `UpdateCurrentMode`, `UpdateUsage`. Optional `Cost` is set on the returned
  `UsageUpdate` pointer (F3). `PatchValue` is unchanged (F4).
- **D3. `acpserver/testdata/session.golden` stays byte-identical.** A diff stops the constructors phase (F7).
- **D4. Embed `acp.ClientRefusals` on `acpclient.handler` and delete the nine stub methods.** `SessionUpdate` and
  `RequestPermission` stay gobble's. Elicitation, fs and terminal still answer `-32601` (or nil for
  `CompleteElicitation`). Working implementations are not added. `acptest` agent stubs are out of scope.
- **D5. Use `SessionUpdate.Kind()` where `translate` or a test names a variant that is not a `Kind*` case.** Do
  not add `KindNotice` or a stable `KindCompaction`. `KindCompaction` remains `_gobble/compaction`.
- **D6. Not this record:** 0016 D8–D10; `_gobble/*`; `session/delete`; elicitation UI; advertising fs/terminal;
  unstable `plan_update` / `subagent_*` constructors; grok `session/set_model`; 0031 P4b; 0033.

**Q1, the unpublished GitHub release.** Pin `v1.21.0` now (recommended), or wait for `gh release view v1.21.0` to
succeed.

**Q2, `Kind()`.** In this pair (recommended), or constructors and `ClientRefusals` only.

**Q3, order relative to 0031-PLAN P4b.** This pair does not wait (recommended): it does not touch the wizard.
Alternatively it runs after P4b.

### Consequences

* Good, because the five literals and nine stubs become SDK names, and 0016 D13's request is met at a pinned tag.
* Good, because `session.golden` staying identical proves the wire did not move.
* Good, because `ClientRefusals` keeps `SessionUpdate` and `RequestPermission` compile-required.
* Neutral, because new constructors set the discriminator field and old helpers do not; tests that
  `reflect.DeepEqual` a constructed value to a decoded one must compare encodings.
* Neutral, because Magic's twin ([magic-cli-remote#28](https://github.com/maccavelli/magic-cli-remote/issues/28))
  is independent.
* Bad, because a later schema variant named `Kind` would force an SDK rename (acp-go-sdk2 0036-MADR R10). No
  variant in schema 1.25.0 has that name.
* Bad, because D8–D10 still have no PLAN.

### Confirmation

[0034-PLAN-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md](0034-PLAN-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md):
`go list -m` reports `v1.21.0`; no `SessionUpdate{AvailableCommandsUpdate:` (and the other four) in non-test
`acpserver`; `handler` embeds `ClientRefusals`; `session.golden` unchanged; `Kind()` used for the
`config_option_update` `KindOther` path; each new check first seen to fail on a planted break in a scratch copy.

## Pros and Cons of the Options

### A. Pin only

* Good, because it is the smallest `go.mod` change.
* Bad, because gobble keeps writing the copies 0004-MADR says to drop.

### B. Pin and adopt the three items

* Good, because it is issue 5's acceptance list, and 0016 D13's request.
* Good, because each item reverts on its own.
* Bad, because it is four commits and three-host gates, for no user-visible ACP change.

### C. Wait for the GitHub release

* Good, because the release notes would be the public text of `v1.21.0`.
* Bad, because `go get @v1.21.0` already fetches the tag; the wait is publication hygiene in the SDK repo
  (its 0036-PLAN step 6), not a gobble pin gate.

### D. Also 0016 D8–D10

* Good, because constructors then have a caller for `UpdateNotice` and `UpdateCompaction`.
* Bad, because when to send those updates is a product choice 0016 still owns, and this pair would smuggle it.

## More Information

- [gobble-cli#5](https://github.com/maccavelli/gobble-cli/issues/5)
- [acp-go-sdk2#11](https://github.com/maccavelli/acp-go-sdk2/issues/11)
- acp-go-sdk2 `0036-MADR-session-update-constructors-kind-and-client-refusals.md` (that repository, tag `v1.21.0`)
- [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md) D5, D13
- [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md) D1, D9
- Implementation: [0034-PLAN-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md](0034-PLAN-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md)
