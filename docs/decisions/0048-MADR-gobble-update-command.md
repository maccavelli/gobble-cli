---
status: accepted
date: 2026-11-09
decision-makers: repository owner
consulted: "this repository at 2e83cce (main, v0.4.0, 2026-11-09); 0047-REPORT-selfupdate-adoption.md; go-selfupdate-lib v1.24.0 (74ab5dd) selfupdate, selfupdate/cli, selfupdate/releasespec, selfupdate/selfupdatetest and selfupdate/verify/ghattest; GitHub release v0.4.0 and its attestations"
informed: ""
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Ship `gobble update` on go-selfupdate-lib, superseding 0005-PLAN F10's self-update step

## Context and Problem Statement

gobble releases five raw binaries, `SHA256SUMS` and an SPDX SBOM to GitHub through go-selfupdate-lib's reusable
workflows, and stamps each release binary with the library's build identity. Nothing in gobble reads those releases:
`gobble update` is an unknown command (Kong usage, exit 2). A user on any release replaces the binary by hand.

[0047-REPORT-selfupdate-adoption.md](../reports/0047-REPORT-selfupdate-adoption.md) found the command feasible from
what is already in the tree, and found that the only record that describes it,
[0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md) F10 step 4, cannot be executed as written. It names
`go-core-lib` `v1.1.0`, a module renamed on 2026-10-02, and sits in a 1.0 bucket with print mode, the usage ledger,
`gobble doctor`, the Pi bridge and the v1.0.0 release gate. The owner asked for a new pair that supersedes that step.
This is that pair.

### What holds at `2e83cce` (re-read 2026-11-09)

Every 0047 finding cited below was re-read at this commit. The two that moved are marked.

* **The pin.** `go.mod:18` requires `github.com/maccavelli/go-selfupdate-lib v1.24.0`. The sibling clone's newest tag
  is `v1.24.0`. `make check-libs` on 2026-11-08 reported 0 failing checks for it.
* **The release (moved).** GitHub Latest is **v0.4.0**, published 2026-11-08T23:57:12Z, immutable. Its assets are
  `gobble-darwin-arm64`, `gobble-linux-amd64`, `gobble-linux-arm64`, `gobble-windows-amd64.exe`,
  `gobble-windows-arm64.exe`, `gobble.spdx.json` and `SHA256SUMS`, each with a GitHub `sha256:` digest.
* **Attestations (new).** `gh api repos/maccavelli/gobble-cli/attestations/sha256:<gobble-darwin-arm64's digest>`
  returns 2 attestations. The publish workflow attests what it publishes.
* **The release spec.** `internal/cli/selfupdate-release.json`: product `gobble`, package `./cmd/gobble`, identity
  args `version --identity`, the five platforms above, packaging `binary`, extra `gobble.spdx.json`, no prerelease
  channels. `TestReleaseSpec` (`internal/cli/release_test.go`) holds it.
* **Identity.** `internal/buildinfo` wraps the library's `buildinfo`. `gobble version --identity` prints
  `Info.Lib.String()`, `<tag> (release) <commit>` on a release build. `gobble version --json` prints `"version"`
  without the `v`. `Info.UserAgent()` is 0003-MADR's User-Agent.
* **Exit codes.** `ExitUpdateAvailable = 10` (`internal/cli/exit.go`), from 0008-MADR D10. `cli.Main` returns its
  code; `cmd/gobble/main.go` is the only `os.Exit`.
* **The root.** `internal/cli/root.go` has no `Update` field. `--version` is the root's flag (0008-MADR D21), so the
  exact-release flag is `--release`.
* **Boundary.** archtest rule 8 already lets `internal/cli` import `go-selfupdate-lib`. `golang.org/x/term`, which
  `selfupdate/cli` imports, is already a direct requirement; `golang.org/x/mod` is already indirect.
* **The agent.** `acpserver/rules.go` asks before `bash(gobble update*)` and `powershell(gobble update*)`. Child tools
  see `AI_AGENT=gobble`.
* **No goldens list the commands.** `internal/cli/testdata` and the completion tests name no command list that
  adding `update` changes.

### The library at `v1.24.0`

Read from the module cache, `go-selfupdate-lib@v1.24.0/selfupdate`.

* **`selfupdate/cli`** (`cli/doc.go`, `cli/flags.go`, `cli/run.go`) is the canonical command: `Flags` (`Check`,
  `Yes`, `Force`, `DryRun`, `JSON`, `Version`, `Channel`); `Flags.Request(product, buildinfo.Info)` refuses
  `--check` with `--yes`, `--force` or `--dry-run` before any updater exists; `Run(ctx, u, req, Options)` writes
  events and the one-line summary to stderr, or JSON Lines and one result object to stdout under `--json`, and
  never calls `os.Exit`; `Exit(stderr, res, err)` writes `update failed: …` and returns 0, 10 or 1.
  `Options.Signals` nil installs its own `os.Interrupt`/`SIGTERM` handler; an empty, non-nil slice installs none.
  `Options.Timeout` zero is `DefaultTimeout`, 15 minutes. `Command` parses with the standard `flag` package and
  names the exact-release flag `--version`, so gobble cannot use it (0008-MADR D10, D21).
* **`selfupdate.New(Config)`** fills no defaults. `Config` takes `Source`, `Versions`, `Assets`, `Verifiers`,
  `Installer`, `Reporter`, `Confirmer`, `Limits`, `ManifestVerifiers` and `Probes`. `NewGitHubUpdater` composes the
  same parts at safe defaults, but takes no `Verifiers` and no `Probes`.
* **`releasespec.Spec.AssetSelector()`** returns `NewExactAssetSelector(spec.Targets())` for `binary` packaging.
* **`NewImageVerifier(Platform{})`** requires the staged file to be an executable image for the running platform.
* **`NewVersionProber(args, want, timeout)`** runs the staged binary, then the installed one, and requires stdout to
  contain `want(tag)`; a nil `want` is the tag itself, `v0.4.0`.
* **`NewStandaloneInstaller(InstallOptions{})`** resolves the target from `os.Executable`. It refuses a symlink
  invocation, a target that is not a regular file, a setuid or setgid target, and a target outside the home
  directory or `TargetPolicy.AllowedRoots`. `CleanupPending(ctx)` takes and releases the target's lock, finishing a
  Windows leftover; `ErrConcurrentUpdate` from it is benign.
* **`GitHubOptions.Token`** empty falls back to `Credentials`, then `GH_TOKEN`, then `GITHUB_TOKEN`.
* **`verify/ghattest`** verifies a release's provenance attestation by running `gh attestation verify`. It is
  opt-in, needs `gh` installed **and logged in** even for a public repository, and fails the update on any of those
  missing. Its own doc says it suits a developer's machine more than an unattended one.
* **`selfupdatetest.NewGitHubServer(t, owner, repo, releases...)`** serves releases on loopback, with `APIBase`,
  `Client` and `Requests()`.

### What 0005 decided that this pair keeps

0005-MADR's "Self-update and distribution" bullets are accepted decisions. This pair keeps them unless a decision
below names one: the surface `update [--check] [--yes|-y] [--force] [--dry-run] [--json] [--release vX.Y.Z]`; flags
local to `update`, so the root's permission `--yes` never reaches `Request.Yes`; `--check` resolves no target and
takes no lock; update checks only on an explicit `gobble update`; no check from `gobble acp`, print mode or the line
session; `GH_TOKEN`/`GITHUB_TOKEN` used when set; the default permission rule; refusing to apply under
`AI_AGENT=gobble`; stable tags only; and, by the owner's answer to Q5, `NewVersionProber` on
`gobble version --json` with `NewImageVerifier`.

## Decision Drivers

* gobble does not reimplement self-update (0004-MADR); it composes the library's parts and adds only what is
  gobble's: the Kong grammar, the exit-code contract, the agent guard.
* One process exit (`cmd/gobble`), one signal owner (`cli.Main`), one stream rule (stdout for results only).
* Everything a test exercises runs on loopback. The 0046 guard fails any test that dials out.
* An update installs only what the release workflow built for this platform, and runs it before replacing anything.
* What the running user does not need, such as `gh` logged in or a channel with no releases, is not required of them.

## Considered Options

* **A. Wait for 0005-PLAN F10.** Keep the step where it is and build it with doctor, the usage ledger, the Pi bridge
  and the release gate, after F7 to F9.
* **B. `NewGitHubUpdater` behind a Kong command.** The library's one-call constructor, with `Platforms` from the
  embedded spec, run through `selfupdate/cli`'s `Flags.Request`, `Run` and `Exit`.
* **C. `selfupdate.New` from the embedded spec, with an image verifier and the version probe, behind a Kong
  command.** As B, but composed from parts so the staged binary is checked as an image and run before anything is
  replaced.
* **D. C, plus provenance through `verify/ghattest`, always on.**

## Decision Outcome

Chosen option: **"C"**, because it ships the command now from parts gobble already pins, keeps every library check
the release spec makes possible, and runs the new binary before it replaces the old one, which B cannot. D makes `gh`,
logged in, a requirement of every update. A leaves a feasible command behind four unrelated phases.

### D1. A Kong `update` command in `internal/cli`

`Root` gains `Update UpdateCmd`. Its flags are local: `--check`, `--yes`/`-y`, `--force`, `--dry-run`, `--json`,
`--release vX.Y.Z`. They fill `selfupdate/cli.Flags`; `--release` fills `Flags.Version`. `--channel` is not exposed
(Q1). Help ends with the library's exit-status paragraph, adapted to `--release`.

### D2. The updater is composed from the embedded release spec

`internal/cli` embeds `selfupdate-release.json`, the file the build workflow reads, so the updater and the release
cannot name different assets. The updater is `selfupdate.New` with:

* `Source`: `NewGitHubSource` for `maccavelli/gobble-cli`, `DefaultHTTPClient()`, `DefaultLimits()`, and
  `buildinfo.Identity().UserAgent()`;
* `Versions`: `NewStrictVersionPolicy()`;
* `Assets`: the spec's `AssetSelector()`;
* `Verifiers`: `NewImageVerifier(Platform{})`;
* `Probes`: `NewVersionProber` on `version --json`, wanting `"version":"<the tag without its v>"`, with a
  10-second timeout (Q5 (b)). `version --json` prints compact JSON: a binary stamped `v0.4.0`/`release` printed
  `{"name":"gobble","version":"0.4.0",…}` on 2026-11-09. The spec's `identity_args`, `version --identity`,
  stay the release workflow's check and are not the probe;
* `Installer`: `NewStandaloneInstaller(InstallOptions{})`;
* `Reporter` and `Confirmer`: a text reporter and a terminal confirmer on the command's own streams, which
  `cli.Run` replaces per run;
* `Limits`: `DefaultLimits()`.

`SHA256SUMS` is checked by the updater itself. Provenance is not checked (Q2).

### D3. The run goes through `selfupdate/cli`, and gobble keeps its exit and its signals

`Run` calls `Flags.Request("gobble", buildinfo.Identity().Lib)`. A contradiction it refuses is a usage error, exit 2.
Otherwise it calls `cli.Run` with gobble's stdout and stderr, stdin, `Interactive` from the terminal detection
`cli.Main` already did, `JSON` from `--json`, and `Signals` an empty, non-nil slice, since `cli.Main` owns the
signals and cancels the context. It returns `cli.Exit`'s code through `exitError` with an empty message, so `Main`
prints nothing more. A signal still exits 128 plus its number, as for every command.

### D4. Only an explicit run touches the install

* `CleanupPending` runs at the start of an `update` that may install, never for `--check`, which takes no lock
  (0005-MADR), and never from any other command. `ErrConcurrentUpdate` from it is ignored; any other error is a
  warning on stderr, and the run goes on.
* Nothing else in gobble checks for updates.

### D5. The agent guard

When `AI_AGENT` is `gobble`, `update` refuses anything but `--check`, before any request, with exit 1 and one line:
`gobble update: refusing to install from inside gobble; run it in your own terminal`. That is 0005-MADR's rule; it
counts `--dry-run` as installing because it downloads and runs the release (Q4).

### D6. Install locations

The library's target policy stands: a regular, non-symlinked file under the home directory. No `AllowedRoots` are
added. When the installer refuses the target, gobble adds one line after the library's error naming the resolved
path and saying to reinstall under the home directory, such as `~/.local/bin`, where `make install` puts it. The
0005-MADR bullet "a Homebrew-managed binary is detected, and `gobble update` prints `brew upgrade gobble`" moves to the
Homebrew tap's own work (0005-MADR "Homebrew tap", 1.x; 0005-PLAN X5): there is no tap, so there is nothing to detect
yet (Q3).

### D7. Tests stay on loopback

`internal/cli` tests run `gobble update` through `Main` against `selfupdatetest.NewGitHubServer`. A package variable
lets a test give the updater the server's `APIBase` and `Client`, an `ExecutablePath` under the test's home, and a
probe whose `Run` is a function. The test binary's own bytes are the fixture asset, so the image verifier checks a
real executable for the running platform. No test reaches GitHub.

### D8. Supersession

On acceptance, this MADR supersedes, and 0005 gains dated notes pointing here for:

* 0005-MADR's "Self-update" inventory row (the composition), and its "Self-update and distribution" bullets
  "Pin" and the Homebrew line;
* 0005-PLAN F10 step 4 and F10 Accept's `gobble update --check` bullet.

F10 steps 1 to 3, 5 and 6 stay in 0005. The bullets this pair keeps (listed in the Context) stay 0005's, and so
does "The TUI uses `Start` / `Stream`", which is F9's.

### Questions for the owner

Each has a recommendation first.

* **Q1. `--channel`.**
  * **(a) Not exposed.** *(recommended)* The spec has no channels and `TestReleaseSpec` asserts none; the flag could
    only fail. Add it with the first channel.
  * (b) Exposed, hidden from help.
  * (c) Exposed, as the library's surface has it.
* **Q2. Provenance.**
  * **(a) Not in this pair.** *(recommended)* `SHA256SUMS`, the GitHub digest, the image check and the probe stand.
    A later pair can add `ghattest` opt-in.
  * (b) Opt-in now, through a config key, when `gh` is on the path.
  * (c) Always on (option D).
* **Q3. Installs outside the home directory, and Homebrew.**
  * **(a) The library's policy, gobble's hint line, the Homebrew line moved to the tap's work.** *(recommended)*
  * (b) Add `/usr/local/bin` and `/opt/homebrew/bin` as `AllowedRoots`. Writing there usually needs `sudo`, and it
    widens what an update may replace.
  * (c) Detect a Homebrew symlink now and print `brew upgrade gobble`, for a formula that does not exist.
* **Q4. `--dry-run` under `AI_AGENT=gobble`.**
  * **(a) Refused, with every mode but `--check`.** *(recommended)* A dry run downloads and runs the release binary.
  * (b) Allowed: it replaces nothing.
* **Q5. The probe's command.**
  * (a) `version --identity`, the spec's `identity_args`. *(recommended)* The release workflow already checks that
    line on each platform's runner, and it contains the tag. This changes 0005-MADR's "`NewVersionProber` on
    `gobble version --json`", whose `"version"` lacks the `v`.
  * **(b) `version --json`, with `want` stripping the `v`.** *(chosen)*

#### Answers (2026-11-09)

The owner: "Q1. leave it out. Q2. not in this pair, Q3. brew out of scope. Q4. no dry run. Q5. version --json
proceed".

* **Q1:** (a), `--channel` is not exposed.
* **Q2:** (a), no provenance check in this pair.
* **Q3:** (a), the library's policy and gobble's hint line; the Homebrew line is out of this pair's scope.
* **Q4:** (a), under `AI_AGENT=gobble` a dry run is refused with every mode but `--check`. The `--dry-run` flag
  itself stays.
* **Q5:** (b), the probe runs `version --json`, as 0005-MADR said. D2 is written to it.

### Consequences

* Good, because users get `gobble update` from the next release, with the checks the library offers for raw
  binaries, and the agent cannot replace gobble under itself.
* Good, because the updater reads the same spec as the release workflow, so a platform change is one file.
* Good, because 0005-PLAN F10 loses a step it could not execute, and 0005's other phases keep their order.
* Neutral, because **v0.4.0 and every earlier release cannot update themselves.** The first release that can is the
  next tag; users reinstall once by hand. README says so.
* Neutral, because a dry run and an install download about 20 to 23 MB from GitHub, as the release assets measure.
* Bad, because a binary installed outside the home directory, such as `/usr/local/bin/gobble`, is refused until the
  user moves it, or until a later decision adds roots.
* Bad, because provenance is not verified: a release published with a stolen token outside the workflow would pass
  `SHA256SUMS`, which it publishes too. The tag rulesets of
  [0043-MADR-protect-main-and-acpext-tags-without-bypass.md](0043-MADR-protect-main-and-acpext-tags-without-bypass.md)
  are the control until Q2 (b) or (c) lands.

### Confirmation

1. `gobble update --help` lists the six flags and the exit-status paragraph, and no `--channel` or `--version`
   under `update`.
2. Against `selfupdatetest.GitHubServer`: `--check` with a newer release exits 10, prints
   `gobble: update available: <cur> -> <tgt>` on stderr and nothing on stdout; with none newer, exits 0; `--json`
   writes JSON Lines ending in one `"kind":"result"` object with `exit_code` equal to the process's; `--check --yes`
   exits 2 before any request; `--release` selects that tag.
3. An apply against the fixture replaces a file at the test's `ExecutablePath`, after the image verifier and the
   probe ran; a planted non-executable asset is refused with `ErrIntegrity` and replaces nothing; a planted
   `gobble-<os>-<arch>.tar.gz` is not selected.
4. Under `AI_AGENT=gobble`, `update`, `update --yes` and `update --dry-run` exit 1 with the guard's line and the
   server records no request; `update --check` runs.
5. A symlink invocation is refused with the library's error and gobble's hint, and nothing is replaced.
6. Each new test is seen failing against a planted break, on a scratch copy.
7. On a scratch home on macOS, the Linux machine and the Windows laptop, a binary built from the tree and stamped
   `v0.3.1`/`release` reports `v0.4.0` from real GitHub with `--check` (exit 10), verifies it with `--dry-run`, installs
   it with `--yes`, and the installed binary's `gobble version` prints `gobble 0.4.0`. On the Windows laptop the
   install also succeeds while another copy of the old binary is running. Cleaning up after that on the next run
   needs an installed release that has `update`, so it is checked after the first such release, by the release guide.
8. `make preflight` and `make race` pass; archtest is unchanged.

## Pros and Cons of the Options

### A. Wait for 0005-PLAN F10

* Good, because 0005's order stays as written.
* Bad, because F10 step 4 names a module that no longer exists and cannot be executed without this rewrite anyway
  (0015-REPORT F4, 0047-REPORT §6).
* Bad, because the command waits for hooks, MCP completion and the TUI, which share no files with it.

### B. `NewGitHubUpdater` behind a Kong command

* Good, because it is the shortest composition, and the library's documented common path.
* Bad, because it takes no verifiers and no probes: a staged binary is installed without being run first, and
  without an image check.
* Neutral, because `Platforms` must still come from the spec, so the spec is embedded either way.

### C. `selfupdate.New` from the embedded spec, with an image verifier and the version probe

* Good, because the staged binary is checked as an image and run before replacement, and again after.
* Good, because the probe reads `gobble version --json`, a machine-readable output 0005-MADR already named for it.
* Neutral, because it is about ten lines longer than B.
* Bad, because provenance is still not checked.

### D. C, plus `ghattest` always on

* Good, because a release made outside the workflow fails.
* Bad, because every update needs `gh` installed and logged in, even for this public repository, which most users of
  a raw binary will not have.

## More Information

### Not decided here

* `gobble doctor` and whether it checks for updates: 0005-PLAN F10 step 3.
* A Homebrew tap and its `brew upgrade` line: 0005-PLAN X5.
* Prerelease channels and tagging release candidates.
* Provenance verification (Q2 (a)).
* The next release tag. The owner asks for it.

### Related records

* [0047-REPORT-selfupdate-adoption.md](../reports/0047-REPORT-selfupdate-adoption.md): the feasibility findings.
* [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) and
  [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md) F10: superseded in part (D8).
* [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D10 (exit codes, no `cli.Command`) and D21
  (`--release`).
* [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): rule 8, and gobble not reimplementing
  self-update.
* [0046-MADR-own-background-listings-and-keep-tests-hermetic.md](0046-MADR-own-background-listings-and-keep-tests-hermetic.md)
  D3, D4: the tests' network guard.
* [0048-PLAN-gobble-update-command.md](0048-PLAN-gobble-update-command.md): the implementation.
