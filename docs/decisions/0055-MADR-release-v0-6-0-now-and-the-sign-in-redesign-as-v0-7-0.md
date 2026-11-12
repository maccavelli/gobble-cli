---
status: accepted
date: 2026-11-12
decision-makers: repository owner
consulted: "this repository at cc1d55e (main, level with origin/main, 2026-11-12): git log and git diff from v0.5.0, the tag objects of v0.4.0, v0.5.0, acpext/v0.3.0 and acpext/v0.4.0, CI run 43131331640 on cc1d55e, v0.5.0's release run 42718701229 and its Release's assets, the module proxy's version list, .github/workflows/ci.yml and docs/guides/developing.md; the sibling libraries' tags from GitHub; magic-cli-remote at its origin/master d78d42ea: go.mod, internal/provider/gobble/gobble.go, its testdata/wire/0.4.0/meta.json, 0179-MADR D14 and A10, 0217-MADR F3k and its PLAN's P3j, 0246-MADR and its PLAN's gate G2, 0060-MADR; go-llmprovider-sdk's 0073 to 0076 records; the fleet record 0006-MADR-sequence-the-auth-contract-effort-across-four-repositories.md and its PLAN; the owner's decision of 2026-11-12, relayed by the cross-repository coordinator"
informed: "0051-PLAN and 0052-PLAN, whose release this renumbers; magic-cli-remote, whose gate 3 this release meets; go-llmprovider-sdk and the fleet record 0006, whose records name the release"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Release gobble v0.6.0 now from main, and release the ACP sign-in redesign as v0.7.0

## Context and Problem Statement

gobble's released `v0.5.0` binaries were built by CI with Go 1.27.3, so they carry GO-2026-6859 in `net/http`, which
they reach through the loopback listener of a browser sign-in
([0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md) F2, F3). `main` has
built with Go 1.27.4 since `88655c5`. On 2026-11-11 the owner chose not to cut a `v0.5.1` and to let the fix reach users
in `v0.6.0` (0054-MADR D6's note). Every record written so far names `v0.6.0` as the release of the ACP sign-in
redesign, [0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
and [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md).
That redesign waits on go-llmprovider-sdk tags that do not exist yet, so the fix would wait with it.

On 2026-11-12 the owner decided, in the cross-repository coordinator session: "Cutting v0.6.0 then v0.7.0 at the end."
gobble `v0.6.0` is released now, from `main` as it stands, and the redesign that the records call `v0.6.0` becomes
`v0.7.0`. This record writes that decision down, checks what `v0.6.0` carries and what it does not, and lists the records
that must change.

### What holds at `cc1d55e` (read 2026-11-12)

* **F1. The base.** `main` is `cc1d55e`, level with `origin/main`, and the tree is clean. CI run `43131331640` on
  `cc1d55e` concluded `success`. The newest release is `v0.5.0`: an annotated tag on `8371b96` whose message is
  `v0.5.0`, published 2026-11-09T23:28:37Z, immutable, GitHub's Latest. Its release run `42718701229` passed `sbom`,
  `validate` on three runners, `build / build`, five `build / identity` jobs and `release / publish`, and skipped
  `release / sign`. Its assets are `gobble-darwin-arm64`, `gobble-linux-amd64`, `gobble-linux-arm64`,
  `gobble-windows-amd64.exe`, `gobble-windows-arm64.exe`, `gobble.spdx.json` and `SHA256SUMS`. `v0.4.0` has the same tag
  form, with message `v0.4.0`.
* **F2. `v0.6.0` is unused.** The module proxy lists `v0.1.0` to `v0.5.0` and no `v0.6.0`
  (`proxy.golang.org/github.com/maccavelli/gobble-cli/@v/list`, read with `@v/list` alone, as
  [developing.md](../guides/developing.md), "When a release goes wrong", says to). `git ls-remote --tags origin` lists
  no `v0.6*` or `v0.7*` tag, and GitHub has no `v0.6.0` Release.
* **F3. The 32 commits of `git log --oneline v0.5.0..cc1d55e`, by what they bring:**

  | Group | Commits | In the `v0.6.0` binary |
  | :--- | :--- | :--- |
  | Records only (11) | `74d7f66`, `a6636fa`, `92c79c3`, `2db8e83`, `b735778`, `34365d6`, `86ad4a7`, `4fc6ef4`, `f402dda`, `834b6d1`, `011b63a` | no |
  | 0054: Go 1.27.4 and the pins (3) | `88655c5` (P1, Go 1.27.4 in both modules, for GO-2026-6859), `f441c73` (P2, acp-go-sdk2 `v1.36.0` and client system prompts in `append` mode, 0054-MADR D7), `b6adb5a` (P3, go-selfupdate-lib `v1.26.2` and CI's workflow pins) | yes |
  | 0049: a failed turn's limit as data (8) | `ece13c4` (acpext's `ErrorKind`, `LimitScope` and the new `ErrorData` and `RetryNotice` members, tagged `acpext/v0.3.0`), `969b161`, `36626fd` (the root on acpext `v0.3.0`), `9eb22bc`, `ef6a870` (P7, `GOBBLE_HOME` without a platform base), `cdfaead` (P8, `--provider`, `--model` and `--thinking` on `gobble acp`), `97f13a5` (P5b-1, the provider's reset instant and stated window in `llm`) | yes |
  | | `6cee0af` (acpext's `windowMinutes`, tagged in `acpext/v0.4.0`) | no: the root requires acpext `v0.3.0` |
  | 0050: the earlier pin refresh (2) | `87ab1da` (go-llmprovider-sdk `v1.29.0`, go-selfupdate-lib `v1.26.0`), `430bc32` (acp-go-sdk2 `v1.35.0`) | yes, as moved on by 0054 |
  | 0053: tests kept off host files (5) | `6bcb898`, `957a943`, `8a7262b`, `ac757a0`, `17d1605` | `957a943`'s walk ceiling is compiled in, and set only by tests' `TestMain`s ([0053-MADR-keep-tests-off-host-files-above-their-temp-root.md](0053-MADR-keep-tests-off-host-files-above-their-temp-root.md)); the rest is tests, a script and a CI job |
  | 0031-PLAN P4b-1 (1) | `d6c10a5` (acpext's openai entry `Login: true`, tagged `acpext/v0.4.0`) | no: the root requires acpext `v0.3.0` |
  | 0051-PLAN P1 (1) | `5007c69` (acpext's `AuthProvider` doc comment) | no: it reaches pkg.go.dev with `acpext/v0.5.0` (0051-PLAN P1 step 2) |
  | 0052-PLAN P1 (1) | `cc1d55e` (the four plan-session fields of `auth.json`, `MethodChatGPTPlan`, `AgentName`, and a dormant key dropped from an `oauth` entry) | yes, the fields inert (F5) |

* **F4. What reaches users in `v0.6.0`:**
  * **The fix for GO-2026-6859.** The release build takes Go from `go.mod` (0054-MADR F2), which says `go 1.27.4`.
  * **Client system prompts** (0054-MADR D7): `initialize` advertises `sessionCapabilities.systemPrompt: {}`, `append`
    mode alone; a client's prompt is applied as the `client_prompt` section and kept with a saved session. A session
    with no client prompt sends the same system prompt as before. acp-go-sdk2 `v1.36.0` refuses an `override` prompt
    and any malformed one with `-32602`.
  * **A failed turn's limit as data** ([0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md)):
    the `ErrorData` of a failed `session/prompt` or `_gobble/compact` gains `provider` and `kind` always, and `code`,
    `httpStatus`, `limit`, `providerMessage`, `resetAt`, `retryAfterMs`, `retryable` and `attempts` when known;
    `_gobble/retry` gains `retryAfterMs`, `serverDirected`, `code` and `httpStatus`. All are additions to acpext
    `v0.3.0`'s shapes. A 401 quota is no longer worded as a refused key, and the CLI prints the limit.
  * **`gobble acp --provider`, `--model` and `--thinking`** (0049-PLAN P8), and **`GOBBLE_HOME` resolving where the
    platform's base is unusable** (0049-PLAN P7).
  * **go-selfupdate-lib `v1.26.2`:** `gobble update`'s errors stop repeating `selfupdate:` (0054-MADR F7).
  * **A dormant key is dropped** when an `oauth` entry is written, such as a Grok sign-in over a stored Grok key
    (`internal/auth/store.go`'s `oauthFields`; 0052-MADR F25).
* **F5. What `v0.6.0` does not change.**
  * **`gobble auth login --json`, ACP `authenticate` and the advertised auth methods.** `git diff --quiet v0.5.0 cc1d55e
    -- internal/cli/auth.go internal/cli/auth_test.go acpserver/auth.go acpserver/auth_test.go internal/cli/agent.go
    internal/cli/acpext_test.go` exits 0. `authProviders()` (`internal/cli/agent.go:381-391`) gives openai by key alone,
    `gobble auth login openai --api-key -`, since `HasLogin` is Grok's alone (`llm/provider/login.go:67`), and
    `TestAcpextAuthProvidersAreTheRegistry` holds it equal to acpext `v0.3.0`'s list, whose `auth.go` is the same as
    `acpext/v0.2.0`'s, the list `v0.5.0` was built with.
  * **The plan sign-in stays unreachable.** `cc1d55e` maps `MethodChatGPTPlan` to the SDK's `AuthChatGPTPlan`, but
    `offers` (`llm/provider/signin.go:85-95`) answers false for it, so no surface lists or runs it (0052-PLAN's P1
    record).
  * **`_meta.gobble.*`.** No `_meta.gobble` member is added, removed or changed. `acpserver/testdata/session.golden`
    differs from `v0.5.0`'s by `"systemPrompt":{}` alone; its `_meta.gobble.extensions` list and `authMethods` are the
    same (0054-MADR Completion 5). 0051-PLAN's P2 onward, which adds `_meta.gobble.methods`, `method`, `fresh`,
    `remote` and `step`, has not started (`docs/README.md`'s 0051 PLAN row).
  * So the fleet's install hazard, "do not install gobble `v0.6.0` on a host that runs Magic until Magic's cut-over is
    installed there" (`0006-PLAN-sequence-the-auth-contract-effort-across-four-repositories.md`, stage 7 and H2), is
    about the redesign. It does not apply to this release. magic-cli-remote warns at `initialize` on a gobble version
    other than its `KnownGoodVersion` and never refuses one (`internal/provider/gobble/gobble.go` there).
* **F6. acpext needs no tag.** `git diff acpext/v0.4.0 cc1d55e -- acpext` is the `AuthProvider` doc comment
  (`5007c69`) and the `go 1.27.4` line (`88655c5`) alone. The root requires acpext `v0.3.0` from the proxy, never the
  tree's copy ([0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md) D5), so
  neither change is in the binary. 0054-MADR D2 decided that no acpext tag is made for the `go` line, and 0051-PLAN P1
  that the doc reaches pkg.go.dev with `acpext/v0.5.0`, its P3.
* **F7. The release workflow.** A `v*` tag runs `.github/workflows/ci.yml` in full; the docs-only filter never applies to
  a tag (`:3-10`). `sbom` fails unless the tag's commit is `main`'s head (`:165-170`), and `release` runs only when
  `build` is not a rehearsal (`:198`). So a release is cut from `main`'s head, and from nowhere else, unless that
  workflow changes.
* **F8. Gate 3 is met by this release.** magic-cli-remote's `0179-MADR-gobble-native-acp-provider.md` D14 gate 3 asks
  for "Both repositories on one acp-go-sdk2 tag"; its amendment A10 records the gate as knowingly unmet until gobble
  `v0.6.0`, which "must pin this repository's tag, which is `v1.36.0` by then". `go.mod` here pins acp-go-sdk2
  `v1.36.0` (since `f441c73`), and so does magic-cli-remote's `go.mod` at its origin/master `d78d42ea` (since its
  0217-PLAN P3j). A `v0.6.0` cut from `cc1d55e`'s tree therefore meets gate 3. magic-cli-remote's own
  `0060-MADR-local-unsigned-build-and-install.md` names a `v0.6.0`: that is magic-cli-remote's release of 2026-08-01,
  not gobble's, and is not touched.
* **F9. What magic-cli-remote has today.** `KnownGoodVersion = "0.4.0"` (`internal/provider/gobble/gobble.go`), with
  the wire fixture `internal/provider/gobble/testdata/wire/0.4.0/frames.jsonl` and its `meta.json`, captured
  2026-11-09 from gobble `0.4.0`; `fixture_test.go` reads the fixture named for `KnownGoodVersion` and checks that
  `meta.json` agrees. It requires acpext `v0.4.0`, one tag newer than the `v0.3.0` that `v0.6.0` is built with. A10
  keeps `KnownGoodVersion` and the `0.4.0` fixture until its
  `0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md` P4 replaces them with the G2
  release's.
* **F10. One sibling has a newer tag.** go-llmprovider-sdk's newest tag is `v1.30.0` (its 0076 minor); gobble pins
  `v1.29.0`. acp-go-sdk2 `v1.36.0`, go-selfupdate-lib `v1.26.2` and go-tui-lib `v2.12.0` are each library's newest. The
  SDK's next pin is 0051-PLAN P2's.

## Decision Drivers

* **Ship the security fix to users now.** Every installed `v0.5.0` stays exposed to GO-2026-6859 until a release that
  carries Go 1.27.4 exists, and the redesign's release waits on go-llmprovider-sdk's `authcontract/v0.1.0` and its
  `v1.31.0`, neither tagged.
* **Keep the redesign's release meaning intact.** The records give the redesign one release that changes
  `auth login --json`, ACP `authenticate` and `_meta.gobble` together, and that Magic's cut-over requires. That release
  keeps those properties under its new number.
* **One renumbering, not two.** The records are renumbered once, here, and the redesign's new number does not move
  again whatever else must ship before it.
* **Releases are cut the way this repository cuts them:** an annotated `vX.Y.Z` tag on `main`'s head, pushed on the
  owner's word, published by the `release` job (developing.md, "Tagging"; F7).
* **The version number says what changed.** `v0.6.0` adds a capability on the wire, error-data members and CLI flags
  (F4). Below `v1.0.0` a minor bump is the release that may add or change behaviour; a patch is a fix.

## Considered Options

* **A. `v0.6.0` now from `main`; the redesign becomes `v0.7.0`.**
* **B. A patch release `v0.5.1` now; the redesign stays `v0.6.0`.**
* **C. Wait, as before: the fix ships with the redesign in `v0.6.0`.**

## Decision Outcome

Chosen option: "A. `v0.6.0` now from `main`; the redesign becomes `v0.7.0`", because it is the owner's decision of
2026-11-12, it puts the Go 1.27.4 build in users' hands at once with the workflow as it is, it names the release by what
it carries, and it leaves the redesign one number of its own.

### D1. `v0.6.0` is cut now from `main` as it stands

* The tag's tree is `cc1d55e`'s, apart from records: `git diff --stat cc1d55e v0.6.0 -- . ':!docs'` is empty. The commits
  after `cc1d55e` are this record's P0 to P1 and touch only `docs/`.
* No code, test, module requirement, workflow or user-facing doc changes for the release. This repository keeps no
  changelog file and no version line in `README.md`; the Release's notes are the workflow's generated comparison, as
  `v0.4.0`'s and `v0.5.0`'s are.
* The pins are the ones `main` has (F10). go-llmprovider-sdk `v1.30.0` is not taken for this release.
* No commit lands on `main` between the tag's commit and the end of its release run, since `sbom` compares the tag with
  `main`'s head when it runs (F7).

### D2. The ACP sign-in redesign is released as `v0.7.0`

* Every record that names `v0.6.0` as the release of 0051 and 0052, the release that changes `auth login --json`,
  `authenticate` and `_meta.gobble` and that Magic's cut-over requires, means `v0.7.0` from this record on.
* Where a record names `v0.6.0` for something this release does carry (the Go fix, acp-go-sdk2 `v1.36.0`, client
  system prompts, the lockstep with magic-cli-remote), the statement stays true and gets a note that `v0.6.0` is this
  record's release.
* **A release before `v0.7.0` is a `v0.6.x` patch.** If another fix must reach users before the redesign, it is cut as
  `v0.6.1` and onward, by a record of its own, and `v0.7.0` keeps its meaning. This is what makes the renumbering
  happen once.
* `v0.7.0` must still meet magic-cli-remote's gate 3 when it is cut: 0246's G2 reads the G2 release's `go.mod` against
  magic-cli-remote's. That check moves from `v0.6.0` to `v0.7.0`; it is not dropped.

### D3. No acpext tag for `v0.6.0`

`v0.6.0` is built with acpext `v0.3.0`, as `main` is. The tree's acpext changes since `acpext/v0.4.0` reach no binary
(F6). `acpext/v0.5.0` stays 0051-PLAN P3's, and carries the doc comment and the `go 1.27.4` line as 0054-MADR D2 and
0051-PLAN P1 say.

### D4. The tag, and what happens when the run fails

* The owner pushes `main`, then creates the annotated tag `v0.6.0` on `main`'s head with the message `v0.6.0`, the form
  of `v0.4.0` and `v0.5.0` (F1), and pushes it. The `release` job publishes it.
* **A failed run is fixed forward.** If any job of the tag's run fails, the tag stays where it is and is not deleted. A
  failed job is read from its log, not re-run until it passes. The fix lands on `main` by an amendment of this record
  or a record of its own, and the release is `v0.6.1`. developing.md allows deleting a tag whose release was never
  published while the proxy does not list it; this record does not use that allowance, since any fetch of the version
  between the read of `@v/list` and the delete would fix it there, and one rule is easier to keep.
* A published Release is immutable (the repository's immutable-releases setting is on), so a defect in one is also fixed
  by `v0.6.1`.

### D5. The records

The gobble records that name `v0.6.0` get dated notes that quote what they amend (the PLAN's P0). The records in
magic-cli-remote, go-llmprovider-sdk and the fleet's docs directory that name gobble's `v0.6.0` are their own
repositories' to amend; the PLAN lists them for the cross-repository coordinator. magic-cli-remote's adoption of
`v0.6.0` (raising `KnownGoodVersion` and recapturing its wire fixture, F9) is its own follow-up, by its own records.

### Consequences

* Good, because users get a gobble built with Go 1.27.4 as soon as the release run passes, with `gobble update` from
  `v0.5.0` (`v0.5.0` carries the command, [0048-MADR-gobble-update-command.md](0048-MADR-gobble-update-command.md)).
* Good, because magic-cli-remote's gate 3 is met by a release for the first time since `912b9aa1` there (F8), and its
  A10 window closes.
* Good, because the work already on `main` (0049's error data, client system prompts, the `gobble acp` model flags)
  reaches users without waiting on the SDK.
* Good, because `v0.7.0` stays the one release that changes `auth login --json`, `authenticate` and `_meta.gobble`, and
  D2's patch rule keeps its number from moving again.
* Neutral, because 0054-MADR D6's "no `v0.5.1`" stays true: no patch of `v0.5.0` is cut.
* Neutral, because developing.md's post-release check from the previous release's binary runs for the first time:
  `v0.5.0` is the first release with `update`, so `v0.6.0` is the first release it can update to.
* Bad, because 102 lines across four places (32 in this repository, 39 in magic-cli-remote, 10 in go-llmprovider-sdk and 21
  in the fleet record) name gobble's `v0.6.0`, and each must be read for whether it meant the
  redesign or this release, and amended where it meant the redesign.
* Bad, because `v0.6.0`'s `initialize` answer adds `systemPrompt` to what magic-cli-remote's `0.4.0` fixture recorded,
  and its error data adds members; magic-cli-remote tests against neither until it recaptures (F9). A magic-cli-remote
  host that installs `v0.6.0` gets the version warning and the new members, which acpext `v0.4.0` decodes.
* Bad, because `v0.6.0` ships the four plan-session fields of `cc1d55e` in a release before the plan sign-in exists.
  They are inert (F5), but they are now released code that `v0.7.0` builds on, and a rollback from `v0.7.0` lands on
  `v0.6.0`, not `v0.5.0`.
* Bad, because the freeze of D1 holds `main` for the length of one release run.

### Confirmation

1. `git diff --stat cc1d55e v0.6.0 -- . ':!docs'` is empty, and `git show v0.6.0:go.mod` says `go 1.27.4` and pins
   acp-go-sdk2 `v1.36.0`, go-llmprovider-sdk `v1.29.0`, go-selfupdate-lib `v1.26.2`, go-tui-lib `v2.12.0` and acpext
   `v0.3.0`.
2. `v0.6.0` is an annotated tag on `main`'s head with the message `v0.6.0`, pushed by the owner; no acpext tag is made.
3. The tag's CI run concludes `success`: `validate` on `ubuntu-24.04`, `macos-15` and `windows-2025`, `test-isolation`,
   `sbom`, `build / build` and its five `build / identity` jobs, and `release / publish`; `release / sign` is skipped,
   as in run `42718701229`.
4. The Release `v0.6.0` is published, immutable and GitHub's Latest, with F1's seven asset names; `gh release verify
   v0.6.0` passes; each binary matches `SHA256SUMS` and its attestation verifies; the binary for macOS reports
   `v0.6.0 (release)` and Go `1.27.4`.
5. Gate 3: `git show v0.6.0:go.mod` and magic-cli-remote's `go.mod` at its origin/master both pin acp-go-sdk2 `v1.36.0`.
6. developing.md's "After each `v*` release" holds: no bypass in the tag's rule suites; `gobble update --check` from
   `v0.5.0`'s binary exits 10 naming `v0.6.0`, on macOS, the Linux machine and the Windows laptop; the Windows
   leftover check passes.
7. The notes in 0051, 0052 and 0054, MADR and PLAN, are made, each quoting what it amends; `docs/README.md` indexes this
   pair; `make check-records` passes.

## Pros and Cons of the Options

### A. `v0.6.0` now from `main`; the redesign becomes `v0.7.0`

* Good, because it ships the fix now with the release workflow unchanged: `main`'s head is what F7 allows.
* Good, because a minor number fits what `main` adds since `v0.5.0` (F4).
* Good, because it meets gate 3 with a release (F8).
* Neutral, because it reverses the timing of 0054-MADR D6's note, not its decision: there is still no `v0.5.1`.
* Bad, because every record that names `v0.6.0` must be read and many amended (D5).

### B. A patch release `v0.5.1` now; the redesign stays `v0.6.0`

* Good, because no record's version number changes.
* Bad, because a `v0.5.1` cut from `main`'s head carries features under a patch number: a new capability on the wire,
  new error-data members and new `gobble acp` flags (F4).
* Bad, because a `v0.5.1` that carries the Go fix alone must be cut from a branch of `v0.5.0`, and `sbom` refuses any
  tag that is not `main`'s head (F7). The workflow would need a change, by a record of its own, before the fix ships.
* Bad, because the owner ruled out a `v0.5.1` on 2026-11-11 (0054-MADR D6's note) and did not reopen it.

### C. Wait, as before: the fix ships with the redesign in `v0.6.0`

* Good, because no record changes, and one release carries everything.
* Bad, because installed `v0.5.0` binaries stay exposed for as long as the redesign takes, and that waits on two
  go-llmprovider-sdk tags and the owner's live probes.
* Bad, because gate 3 stays unmet by any release for the same time, the window A10 accepted as temporary.
* Bad, because what is on `main` now reaches users only with the largest change the CLI has had to its sign-in.

## More Information

### The records that name `v0.6.0`

Read 2026-11-12 with `grep -c 'v0\.6\.0'`. In this repository: 0051-MADR (1 line), 0051-PLAN (8), 0052-MADR (1),
0052-PLAN (5), 0054-MADR (12) and 0054-PLAN (5); `docs/README.md` names `v0.6.0` nowhere, and its 0054 MADR row's "no
`v0.5.1`" stays true. The PLAN's P0 gives each occurrence with what it becomes. The other repositories' records are
listed there too, for the coordinator.

### Not decided here

* **When magic-cli-remote adopts `v0.6.0`,** and whether it does before its cut-over or keeps `0.4.0` until 0246's P4
  as A10 says: magic-cli-remote's records.
* **go-llmprovider-sdk `v1.30.0`'s pin:** 0051-PLAN P2.
* **Anything else for `v0.7.0`:** 0051 and 0052, unchanged apart from the number.

### Related

* [0055-PLAN-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-PLAN-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md):
  the implementation.
* [0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md): D1, D6 and its
  note, D7.
* [0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
  and [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md):
  the redesign, now `v0.7.0`.
* [0048-MADR-gobble-update-command.md](0048-MADR-gobble-update-command.md): `v0.5.0`, the first release with
  `update`.
* [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md): D5, acpext's tags.
* [0043-MADR-protect-main-and-acpext-tags-without-bypass.md](0043-MADR-protect-main-and-acpext-tags-without-bypass.md):
  the rulesets the tag push passes.
* [developing.md](../guides/developing.md): "Releasing".
* magic-cli-remote `docs/decisions/0179-MADR-gobble-native-acp-provider.md`: D14 gate 3, A10.
* The fleet record `0006-MADR-sequence-the-auth-contract-effort-across-four-repositories.md`, outside any repository.
