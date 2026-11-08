---
status: observation
date: 2026-11-08
subject: "technical feasibility of wiring gobble update from go-selfupdate-lib v1.24.0, and why a new MADR/PLAN pair should supersede 0005-PLAN F10's self-update step"
examines: "gobble-cli HEAD feee80887519e965aaeaaebb3d6360062fc21fa8 (v0.3.1-14-gfeee808-dirty, main ahead origin/main by 5); go.mod pin go-selfupdate-lib v1.24.0; go-selfupdate-lib tag v1.24.0 peeled 74ab5dd376a4c864debb32b2d8abfc7856059ba3; GitHub release v0.3.1 published 2026-11-07T19:04:23Z; 0005-MADR Self-update and Distribution rows; 0005-PLAN F10; 0008-MADR D10 D14 D21; 0004-MADR rule 8; 0015-REPORT F4/F216; magic-cli-remote internal/updateclient as a contrast only"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# REPORT 0047 — Adopting go-selfupdate-lib for `gobble update`

This report records findings. It decides nothing. The owner asked on 2026-11-08 for the technical feasibility of adding self-update support to gobble-cli from the current codebase, written as an adoption report for later work, and stated that a **new MADR/PLAN pair will supersede the existing F10 self-update record**.

Every claim names its source: a file at the baseline above, a numbered record, a command whose output is quoted, or the GitHub release API. Tags: **(R)** read from source or a record; **(M)** measured by running something.

## 1. Answer

**Wiring `gobble update` is feasible now.** The library, the five-platform GitHub assets, the reusable release workflows, the build-identity stamps, the Kong `--version` collision, and the exit-code 10 slot are already in the tree. What is missing is a Kong `update` command in `internal/cli` that composes APIs gobble already pins.

**Do not implement it as 0005-PLAN F10 step 4 as written.** That step still names `github.com/maccavelli/go-core-lib/selfupdate` `v1.1.0` at commit `96b30961180671ab3697585951219001ecbb1c90` (`0005-PLAN-v1-feature-scope.md:352-353`). The module was renamed on 2026-10-02; gobble pins **v1.24.0**; GitHub Latest is **v0.3.1** with the five raw binaries the updater would fetch. F10 is also a mixed 1.0 bucket (print mode, usage, doctor, update, Pi bridge, release-gate table). Print mode `-p` already exists (`internal/cli/flags.go:16-17`).

**The next work is a new pair (next unused number after this report: 0048), which supersedes 0005-PLAN F10 step 4 and 0005-MADR's Self-update rows.** The owner stated that intent in the request that produced this report. The rest of F10 (doctor, remaining Pi bridge, the v1.0.0 gate table) stays in 0005 until that pair's MADR says otherwise.

**0015-REPORT's "nothing to update to yet" is stale.** That report froze gobble at `6a8b9d8` on 2026-10-11, with no git tag (`0015-REPORT-sibling-library-pins-and-order.md:218`). GitHub release **v0.3.1** exists, published 2026-11-07T19:04:23Z, with `gobble-darwin-arm64`, `gobble-linux-amd64`, `gobble-linux-arm64`, `gobble-windows-amd64.exe`, `gobble-windows-arm64.exe`, `SHA256SUMS`, and `gobble.spdx.json` (M, `gh release view v0.3.1`).

**magic-cli-remote's `internal/updateclient` is the wrong template.** It composes a managed installer, HandOff, codesign, and two products. gobble is one standalone binary. The library's `ExampleNew_standalone` / `NewGitHubUpdater` is the matching path (`go-selfupdate-lib` `selfupdate/example_test.go:25-67`, `selfupdate/githubupdater.go:44-50` at v1.24.0).

## 2. Scope and method

- **This tree:** `git rev-parse HEAD` = `feee80887519e965aaeaaebb3d6360062fc21fa8`; `git describe --tags --long --dirty` = `v0.3.1-14-gfeee808-dirty`; `main` ahead of `origin/main` (`4970cca`) by 5. Working-tree dirt on 2026-11-08 is acpext compaction-meta and 0016-PLAN; none of the self-update files cited below were dirty. (M)
- **Pin:** `go list -m github.com/maccavelli/go-selfupdate-lib` → `github.com/maccavelli/go-selfupdate-lib v1.24.0` (exit 0). `go.mod:18` requires the same tag. (M, R)
- **Library tag:** in the sibling clone, `git describe --tags --abbrev=0` = `v1.24.0`; `git rev-parse v1.24.0^{}` = `74ab5dd376a4c864debb32b2d8abfc7856059ba3`; log subject `feat(redact): hide Env values in formatted option types`, 2026-11-02. (M)
- **Release:** GitHub API on `maccavelli/gobble-cli` release `v0.3.1` (M).
- **Records:** 0004-MADR rule 8; 0005-MADR Distribution and Self-update; 0005-PLAN F10; 0008-MADR D10, D14, D21 and 0008-PLAN amendment of 2026-11-06; 0015-REPORT; 0019-MADR D12; 0035-PLAN P28. (R)
- **Search:** `rg` over `*.go` for `type UpdateCmd`, `CleanupPending`, `NewGitHubSource`, `NewGitHubUpdater`, `NewStandaloneInstaller`, `selfupdate/cli`, `DoctorCmd`. (M)
- **Not done:** no product command was written; no tag; no live `gobble update` against GitHub.

## 3. What is already shipped

### 3.1 Library pin, current with the sibling

| Check | Result |
| :--- | :--- |
| gobble `go.mod` | `github.com/maccavelli/go-selfupdate-lib v1.24.0` (`go.mod:18`) |
| `go list -m` | same, exit 0 |
| library newest `v*` | `v1.24.0` |
| CI reusable workflows | `build-selfupdate-release.yml@74ab5dd… # v1.24.0` and `publish-selfupdate-release.yml@74ab5dd… # v1.24.0` (`.github/workflows/ci.yml:171`, `:183`) |
| SHA of that pin | `74ab5dd376a4c864debb32b2d8abfc7856059ba3` equals `v1.24.0^{}` |

0035-PLAN P28 recorded the pin on 2026-11-03: `go get …@v1.24.0`, no source change, `go test -count=1 ./internal/buildinfo/... ./internal/cli/` passed (`0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md`, P28 record).

### 3.2 Release pipeline and assets

`internal/cli/selfupdate-release.json` (R):

```json
{
  "schema": 1,
  "products": [
    {"name": "gobble", "package": "./cmd/gobble", "identity_args": ["version", "--identity"]}
  ],
  "platforms": [
    {"os": "darwin", "arch": "arm64"},
    {"os": "linux", "arch": "amd64"},
    {"os": "linux", "arch": "arm64"},
    {"os": "windows", "arch": "amd64"},
    {"os": "windows", "arch": "arm64"}
  ],
  "packaging": "binary",
  "extras": [
    {"name": "gobble.spdx.json"}
  ]
}
```

`TestReleaseSpec` (`internal/cli/release_test.go:17-49`) parses that file with `releasespec.Parse`, asserts product `gobble`, package `./cmd/gobble`, identity args `version --identity`, the five platforms above, `PackagingBinary`, extra `gobble.spdx.json`, and **no prerelease channels** ("gobble update selects stable tags only (0005-MADR)"). The comment at `:15-16` says F10 embeds the same file for the updater.

CI (`.github/workflows/ci.yml:169-189`) (R):

- `build` job `uses:` the library workflow at `74ab5dd` with `spec-path: internal/cli/selfupdate-release.json`.
- `release` job runs when `needs.build.outputs.rehearsal == 'false'` (a `v*` tag), with `contents: write`, `id-token: write`, `attestations: write`.

GitHub release **v0.3.1** (M, 2026-11-08): not draft, not prerelease, published 2026-11-07T19:04:23Z.

| Asset | Size (bytes) |
| :--- | ---: |
| `gobble-darwin-arm64` | 20 799 058 |
| `gobble-linux-amd64` | 22 110 368 |
| `gobble-linux-arm64` | 20 250 784 |
| `gobble-windows-amd64.exe` | 22 771 712 |
| `gobble-windows-arm64.exe` | 20 537 344 |
| `gobble.spdx.json` | 34 087 |
| `SHA256SUMS` | 438 |

Those five binary names are what `NewExactAssetSelector` / `releasespec.Spec.AssetSelector` will look for. 0005-MADR's Distribution row still says "Six **raw binaries**" and then notes five since 2026-10-07 (`0005-MADR-v1-feature-scope.md:274`). The live spec and the live release are five.

### 3.3 Build identity

Makefile (`Makefile:88-94`, `:109`) stamps `github.com/maccavelli/go-selfupdate-lib/buildinfo.version` and `.kind`. Local builds use `git describe --tags --always --dirty` and `BUILD_KIND=local`. A release build sets `VERSION` to the tag and `BUILD_KIND=release`.

`internal/buildinfo` wraps `lib.Identity()` (`internal/buildinfo/buildinfo.go:9`, `:41-46`). A version that is not a clean SemVer tag becomes `0.0.0-dev[+<revision>[.dirty]]` (0008-MADR D19; `semver` at `:71-83`). `Info.UserAgent` is `gobble/<version> (<GOOS>; <GOARCH>) go/<go version>` (`:85-89`), which is 0003-MADR's User-Agent and the string `NewGitHubSource` should send.

`gobble version` and root `--version` are live (`internal/cli/version.go`; `Root.ShowVersion` in `internal/cli/root.go:22`). `--identity` prints `i.Lib.String()` for the release workflow (`version.go:18-21`, `:60-61`). 0008-MADR D21 (2026-11-06) already chose that `--version` is the root flag, so `update`'s exact-release flag is **`--release`**, filling `selfupdate/cli.Flags.Version` (`0008-MADR-native-cli-mode.md:738-762`). 0008-PLAN's amendment of that date records: "The `update` command does not exist yet; its `--release` flag is recorded in 0005-MADR and 0005-PLAN F10, and built by F10" (`0008-PLAN-native-cli-mode.md:1080-1081`).

### 3.4 CLI contracts already coded

| Contract | Where it lives |
| :--- | :--- |
| Exit 10 = update available | `ExitUpdateAvailable = 10` (`internal/cli/exit.go:11-16`); 0008-MADR D10 table (`0008-MADR-native-cli-mode.md:489-496`) |
| `Main` never calls `os.Exit` | `cli.Main` returns the code (`internal/cli/main.go:31-34`); `cmd/gobble/main.go:11-20` is the only `os.Exit` |
| Import boundary | archtest rule 8: only `internal/cli` may import `go-selfupdate-lib` other than `buildinfo`; `internal/buildinfo` may import only `…/buildinfo`; tests may import `selfupdatetest` (`internal/archtest/check.go:203-215`, `:403-423`) |
| Default permission | `ask: bash(gobble update*)` and `ask: powershell(gobble update*)` in `acpserver/rules.go:82-87` (0005-PLAN F6-3) |
| Agent marker | `markers` always starts with `AI_AGENT=gobble` (`acpserver/agent.go:1244-1248`); 0005-MADR Self-update: apply refuses when that is set, `--check` still allowed (`0005-MADR-v1-feature-scope.md:617`) |
| No phone-home on ACP / print / TUI | 0005-MADR Self-update (`:610-612`); there is no update check in `ACPCmd`, print mode, or the line session, because there is no updater yet |
| `--offline` is not this work | `laterFlags["offline"]` = `0005-PLAN X3` (`internal/cli/flags.go:65`, `:77-78`) |

### 3.5 What currently imports the library

`rg --glob '*.go' github.com/maccavelli/go-selfupdate-lib` (M):

| File | Import |
| :--- | :--- |
| `internal/buildinfo/buildinfo.go` | `…/buildinfo` (production) |
| `internal/cli/main_test.go` | `…/buildinfo` (test) |
| `internal/cli/release_test.go` | `…/selfupdate` and `…/selfupdate/releasespec` (**test only**) |
| `internal/archtest/check.go`, `check_test.go` | path strings for rule 8 |

There is **no** production import of `selfupdate`, `selfupdate/cli`, or `selfupdate/selfupdatetest`. The comment at `internal/buildinfo/buildinfo.go:37` names `selfupdate/cli` as the future consumer of `Info.Lib`.

## 4. What is not wired

`rg --glob '*.go'` for `type UpdateCmd`, `CleanupPending`, `NewGitHubUpdater`, `NewGitHubSource`, `NewStandaloneInstaller` returned no matches. `selfupdate/cli` appears only in that buildinfo comment. `DoctorCmd` has no matches. (M)

`internal/cli/root.go:6-23` (R): Kong `Root` lists Chat, Version, Completion, Config, ACP, MCP, Models, Providers, Auth, Setup, Personality, Trust, Untrust, plus `--log-level`, `--set`, `--version`. There is no `Update` field.

So `gobble update` is today an unknown command (Kong usage, exit 2). 0008-PLAN's D21 amendment already said so.

F10 step 3 `gobble doctor [--json]` is also absent. 0005-MADR says doctor may check for updates (`:611`). Doctor is not required to ship `update`; it is a separate F10 item.

## 5. Library surface at v1.24.0 (what to call)

Read in go-selfupdate-lib at `v1.24.0` (`74ab5dd`).

### 5.1 Standalone composition (the F10 shape)

`selfupdate/example_test.go` `ExampleNew_standalone` (`:25-67`) (R):

- `NewGitHubSource(GitHubOptions{Repository, Client: DefaultHTTPClient(), UserAgent, Limits: DefaultLimits()})`
- `NewExactAssetSelector([]Platform{…})` (or `releasespec.Spec.AssetSelector` when a spec is embedded)
- `NewStandaloneInstaller(InstallOptions{})`
- `New(Config{Source, Versions: NewStrictVersionPolicy(), Assets, Installer, Reporter: NewTextReporter(os.Stderr), Confirmer: NewTerminalConfirmer(os.Stdin, os.Stderr), Limits})`

That is the same list 0005-PLAN F10 step 4 already names (`:354-361`), with the module path corrected from `go-core-lib`.

### 5.2 Convenience constructor

`NewGitHubUpdater` (`selfupdate/githubupdater.go:44-50`) composes those parts at safe defaults for the **running platform only** unless `Platforms` is set. For gobble, the spec's five platforms should be passed (or `New` used with `Spec.AssetSelector()`), so a `--dry-run` for another GOOS/GOARCH still selects the right asset. `Client` nil means `DefaultHTTPClient()`: transport timeouts, **no** `Client.Timeout`, so a large download is bounded by context (`githubupdater.go:32-35`, `:108`).

### 5.3 CLI helper, which gobble must not call as `Command`

`selfupdate/cli` (`cli/doc.go:1-12`, `cli/command.go:13-27`, `cli/flags.go:25-41`, `:116-139`, `cli/run.go:138`, `:360`) (R):

- Surface: `<prog> update [--check] [--yes|-y] [--force] [--dry-run] [--json] [--version vX.Y.Z] [--channel NAME]`
- `Flags.Request` maps those onto `selfupdate.Request` and refuses `--check` with `--yes` / `--force` / `--dry-run`
- `Run` executes; `Exit` maps `selfupdate.ExitCode` (0 / 10 / 1)
- `Command` parses with the standard `flag` package and **returns an exit status for the caller to `os.Exit`**

0008-MADR D10 (`:500`): "go-selfupdate-lib's `cli.Command` is not called. gobble calls `Flags.Request` and `Run` and maps errors to this table." D14 and D21: gobble declares flags in Kong and fills `Flags.Version` from **`--release`**. It does not use `Flags.Bind`, `Parse`, or `HelpText`, which name `--version`.

`cmd/gobble/main.go:11-12` is the only `os.Exit`. Calling `cli.Command` from a Kong `Run` would either double-exit or fight `Main`'s return-code contract.

### 5.4 Test fixture

`selfupdatetest.GitHubServer` is exported (`selfupdate/selfupdatetest`, used from `selfupdate/e2e_github_test.go:120-165`). 0005-PLAN F10 Accept (`:382-386`) already names it: `--check` against a fixture `gobble-linux-amd64` exits 10; a planted `gobble-linux-amd64.tar.gz` is not selected. 0046's `internal/cli` network guard will fail the package if a test dials a real host (`0046-MADR` D3, D4). Tests inject `HTTPClient` / `APIBaseURL` pointing at that fixture.

### 5.5 Installer policy

`NewStandaloneInstaller` (`selfupdate/standalone.go:29-31`) plus `TargetPolicy` refuses a symlink invocation (`selfupdate/target_test.go:36`, "accepted symlink invocation"). Homebrew-managed binaries are typically a symlink into the Cellar; the installer refuses that path. 0005-MADR (`:607`) still wants the user-facing line `brew upgrade gobble`. The Homebrew **tap** itself is 0005-MADR 1.x (`:277`) / 0005-PLAN X5, not this command.

`CleanupPending` (`standalone.go:70-81`) finishes a Windows leftover rename. No gobble `.go` file calls it (M). Call it from `update`'s `Run`, not from `gobble acp` or completion (0008-MADR D17: completion runs before any network or file but the ones it completes).

Windows "Access is denied" on a running copy was a go-core-lib defect; 0005-MADR (`:608-609`) and 0019-MADR D12 record it fixed in go-selfupdate-lib `v1.5.1`. The pin is `v1.24.0`.

## 6. Why F10 cannot be the implementation record

### 6.1 The step is stale

0005-PLAN F10 step 4 (`:350-368`) still opens with `github.com/maccavelli/go-core-lib/selfupdate` `v1.1.0` (`96b30961…`). A 2026-10-13 note says the module is go-selfupdate-lib from `v1.5.0` and that F10 binds the newest tag, then names **`v1.15.2`**. Accept still says `go list -m github.com/maccavelli/go-core-lib` prints `v1.1.0` (`:386-387`), with a note to print go-selfupdate-lib instead.

0015-REPORT F4 (`:216-223`) already said F10 step 4 "must be re-made executable" before it is built. That was 2026-10-11. It was not re-made. The pin has since moved through 0019, 0023, 0031, 0035 P3/P22/P28 to **v1.24.0**.

### 6.2 F10 is a mixed 1.0 bucket

F10 steps (`:347-376`):

1. Print mode `-p`, `--output-format` — **live** (`SharedFlags.Print`, `OutputFormat`; 0008-PLAN completed).
2. Usage ledger / `gobble usage` — not this report.
3. `gobble doctor` — absent.
4. `gobble update` — absent; this report's subject.
5. Pi bridge remainder — `auth import --from-pi` exists (`internal/cli/piimport.go`); `session list --pi` / `session import` / `--no-pi-bridge` not searched here in full.
6. Release-gate table and `apidiff` against the last pre-release — 0004-PLAN Phase 5; `make apidiff` is a no-op below v1.0.0.

Executing "F10" as one phase would mix a small Kong wrapper with doctor, Pi, and the v1.0.0 gate. 0005-PLAN's Order (`:43`) also places F10 after F7 hooks, F8 MCP completion, and F9 TUI. Those phases do not share files with the updater. Waiting on them is a process constraint of 0005, not a technical one.

### 6.3 Owner intent for the next pair

The owner, in the request that produced this report: write the findings as an adoption report, and **write a new MADR/PLAN pair that supersedes the existing Fn record**.

That pair (number **0048** if taken next; `scripts/check_records.py --next` printed `0047` before this file existed) should:

- **Supersede** 0005-PLAN F10 step 4 and 0005-MADR's Self-update bullets (`:592-621`) and the Self-update inventory row (`:276`).
- **Leave** F10 steps 1–3, 5, 6 in 0005 unless that MADR takes them.
- **Pin** go-selfupdate-lib at the newest `v1.x` at execution time (today **v1.24.0**), exact tag, not `@latest`.
- **Bind** as §5: Kong `UpdateCmd`, `--release` → `Flags.Version`, `Flags.Request` + `cli.Run` + `ExitCode` mapped onto `ExitOK` / `ExitUpdateAvailable` / `ExitFailure` / `ExitUsage`.
- **Compose** from the embedded spec (`internal/cli/selfupdate-release.json`), `NewGitHubSource` with `buildinfo.Identity().UserAgent()`, `DefaultHTTPClient()`, `NewStrictVersionPolicy`, the spec's selector, `NewStandaloneInstaller`, text reporter on stderr, terminal confirmer on stdin/stderr. `NewGitHubUpdater` is acceptable if `Platforms` comes from the spec.
- **Refuse apply** when `os.Getenv("AI_AGENT") == "gobble"`; `--check` still runs.
- **Print** `brew upgrade gobble` when the installer refuses a symlink in a Homebrew prefix.
- **Test** on `selfupdatetest.GitHubServer` under 0046's guard; no live GitHub in `go test`.
- **Not** call `selfupdate/cli.Command`.
- **Not** add a background check to `gobble acp`, `-p`, or the line session.
- **Not** copy magic-cli-remote's managed / HandOff / codesign path.
- **Not** tag v1.0.0 (0005-PLAN F10 Accept already forbids that).

0005-PLAN should then gain a dated note pointing at the new pair, the way other superseded steps do. That note is the new pair's to write, not this report's.

## 7. Contrast: magic-cli-remote

`magic-cli-remote/internal/updateclient/client.go:1-11` (R): the package "adapts go-selfupdate-lib's selfupdate package to mcremote and mcrelay" and is "the only place this repository knows about GitHub releases". `Options.Managed`, codesign, `libservice`, HandOff appear in that file (`:54-70`, imports `:25-31`). `cmd/mcremote/main.go` handles `updateclient.ExitError` and a detached update run (`:18-38`).

gobble has no service unit, no second binary, and no codesign identity in the release spec. Importing that design would pull `selfupdate/service` and `selfupdate/codesign` into `internal/cli` without a product that needs them. archtest rule 8 would allow the import (it is still go-selfupdate-lib); the product would not.

## 8. Risks the next pair has to name

1. **darwin/amd64.** Not a target (`selfupdate-release.json` platforms; 0004-MADR amendment 2026-10-07). An Intel Mac has no asset. The updater reports no matching asset. That is already the release contract.
2. **`--channel`.** Spec has `PrereleaseChannels` empty (`TestReleaseSpec` `:47-49`). Exposing the library flag is optional; it will find no channel.
3. **Probes.** 0005-MADR (`:602`) wants `NewVersionProber` on `gobble version --json`. The release spec's identity command is `version --identity`. The pair should pick one and keep it equal to `identity_args`.
4. **TUI `Start` / `Stream`.** 0005-MADR (`:603`) names them for a TUI. F9 has not run. The CLI command does not need them.
5. **Working tree.** On 2026-11-08 HEAD was dirty with acpext compaction-meta (0016-PLAN). The new pair starts from a stated `HEAD` in its PLAN C1, not from this report's dirty snapshot.
6. **F7–F9 order.** Starting the new pair before F7 is a 0005-PLAN deviation if 0005's Order is left unchanged. The superseding pair is how that deviation is recorded: 0005 no longer owns the step.
7. **GH_TOKEN.** 0005-MADR (`:613`): `GH_TOKEN` / `GITHUB_TOKEN` are used when set; a stale token fails 401. Both are the library's. Document them; do not add a gobble-specific token flag.

## 9. Claims index

Each row was checked at the baseline in §2. (R) = read, (M) = measured.

| # | Claim | Evidence |
| :--- | :--- | :--- |
| 1 | gobble pins go-selfupdate-lib v1.24.0 | `go.mod:18`; `go list -m` exit 0 (M) |
| 2 | v1.24.0 is the library's newest v* tag | sibling `git describe --tags --abbrev=0` (M) |
| 3 | CI workflows are pinned to that tag's commit | `ci.yml:171`, `:183`; `git rev-parse v1.24.0^{}` = `74ab5dd…` (M) |
| 4 | Release spec lists five raw-binary platforms, product gobble | `internal/cli/selfupdate-release.json`; `TestReleaseSpec` |
| 5 | GitHub v0.3.1 publishes those five binaries plus SHA256SUMS and SPDX | `gh release view v0.3.1` (M) |
| 6 | `Root` has no update command | `internal/cli/root.go:6-23` |
| 7 | No `UpdateCmd`, `DoctorCmd`, `CleanupPending`, `NewGitHubSource` in gobble `.go` files | `rg` empty (M) |
| 8 | Production selfupdate import is test-only except buildinfo | `rg` import list in §3.5 (M) |
| 9 | Exit 10 is already named | `internal/cli/exit.go:16`; 0008-MADR D10 |
| 10 | `--version` is a root flag; update's exact tag is `--release` | 0008-MADR D21; `versionFlag` in `version.go:43-54`; 0008-PLAN `:1080-1081` |
| 11 | `cli.Command` is forbidden; Kong + `Flags.Request` + `Run` | 0008-MADR D10 `:500`, D21 `:759-762`; `cmd/gobble/main.go` sole `os.Exit` |
| 12 | Default rules ask for `gobble update*` | `acpserver/rules.go:86-87` |
| 13 | Child tools see `AI_AGENT=gobble` | `acpserver/agent.go:1248` |
| 14 | F10 step 4 still names go-core-lib v1.1.0 | `0005-PLAN-v1-feature-scope.md:352-353`, `:386-387` |
| 15 | 0015 said there was no release to update to | `0015-REPORT:218`; superseded by claim 5 |
| 16 | Library standalone path and `NewGitHubUpdater` exist at v1.24.0 | `example_test.go:25-67`; `githubupdater.go:44-50` |
| 17 | `selfupdatetest.GitHubServer` is the Accept fixture | 0005-PLAN `:382`; library `e2e_github_test.go` |
| 18 | Magic's updater is managed/service | `magic-cli-remote/internal/updateclient/client.go:1-70` |
| 19 | Print mode `-p` already exists (F10 step 1) | `internal/cli/flags.go:16-17` |
| 20 | archtest rule 8 allows `internal/cli` to import `selfupdate` | `internal/archtest/check.go:403-414`; `check_test.go:172-174` |
| 21 | `--offline` is X3, not update | `internal/cli/flags.go:77-78` |
| 22 | Next unused number before this report was 0047 | `python3 scripts/check_records.py --next` (M) |

## 10. What this report does not do

- It does not accept a design. The new pair does that.
- It does not amend 0005. The new pair's PLAN writes the dated supersession note.
- It does not implement `gobble update`.
- It does not tag, push, or pin a newer go-selfupdate-lib than v1.24.0.
- It does not close 0005-PLAN. F7, F8, F9, and the rest of F10 remain.
