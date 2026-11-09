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
  Windows leftover; `ErrConcurrentUpdate` from it is benign. *(2026-11-09: the symlink refusal sees only a path
  `os.Executable` returns as the symlink, which on Linux it never does; see "Amendment (2026-11-09)".)*
  *(Closed by go-selfupdate-lib `v1.25.0`, Amendment A2: refused on Linux too.)*
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

*(2026-11-09 amendment.)* On Linux, "non-symlinked" holds for the file replaced, not for the path invoked: a
symlink invocation updates the file it points at when that file passes the policy, and gets no hint. On macOS it
is refused, with the hint. See "Amendment (2026-11-09)". *(Closed by go-selfupdate-lib `v1.25.0`, Amendment A2: refused on Linux too.)* A symlink refusal gets A2-3's hint.

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

### Amendment (2026-11-09): a symlink invocation on Linux

Found at 0048-PLAN P3 step 2, the live run. A binary built from the tree, stamped `v0.3.1`/`release`, at
`<scratch home>/.local/bin/gobble`, invoked through `<scratch home>/link/gobble`, a symlink to it:

* **macOS (darwin/arm64):** exit 1, `update failed: selfupdate: gobble: selfupdate: refusing to update through a
  symlink`, then gobble's hint; the file unchanged. As D6 and Confirmation 5 say.
* **The Linux machine (linux/amd64):** exit 0, `gobble: updated v0.3.1 -> v0.4.0`, and the file the symlink points at
  replaced with the release asset.

**Why.** The installer takes the raw path from `os.Executable` (`selfupdate/replace.go`, `osExecutable`) and refuses
it when `os.Lstat` reports a symlink (`selfupdate/target.go`, `resolveTarget`). On Linux, `os.Executable` reads
`/proc/self/exe`, which the kernel has already resolved, so the raw path is never the symlink and that check cannot
fire; Go's documentation says the result may be the symlink or its target, by operating system. The file replaced still
passes every other check: a regular file, not setuid or setgid, under the home directory. P1's `TestUpdateSymlink`
passes on Linux because it sets `TargetPolicy.ExecutablePath` to the symlink, which skips `os.Executable`.

**The owner's decision** (2026-11-09: "option 1, draft the issue, proceed"): record it as the library's behaviour on
Linux, and have the library fixed, rather than detect the invoked path in gobble, which would reimplement self-update
behaviour ([0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md)). So:

* D6 (above) gains a dated note; Confirmation 5 (below) is narrowed, struck through where the text changes; the facts on
  `NewStandaloneInstaller` carry a note.
* README's "Updating" section says that on Linux a symlink invocation updates the file it points at.
* An issue for go-selfupdate-lib was drafted and handed to the owner, not posted. A later pair takes up the version that
  fixes it. *(2026-11-09: filed on the owner's word as [maccavelli/go-selfupdate-lib#9](https://github.com/maccavelli/go-selfupdate-lib/issues/9).)*
* A new Consequence names the risk: a symlink into a tree another tool manages under the home directory, such as
  Homebrew on Linux in `~/.linuxbrew`.

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
* Bad, because on Linux a symlink invocation updates the file it points at (Amendment (2026-11-09)): a symlink
  into a tree another tool manages under the home directory, such as Homebrew on Linux in `~/.linuxbrew`, has that
  tool's copy replaced, where macOS refuses. Until go-selfupdate-lib sees the invoked path on Linux.
  *(Closed by go-selfupdate-lib `v1.25.0`, Amendment A2: refused on Linux too.)*

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
5. ~~A symlink invocation is refused with the library's error and gobble's hint, and nothing is replaced.~~
   *(2026-11-09 amendment)* Where `os.Executable` returns the symlink (macOS), a symlink invocation is refused
   with the library's error and gobble's hint, and nothing is replaced; a test that gives the installer the
   symlink's path shows the same on Linux. On Linux, invoked through a symlink, the file it points at is checked
   and replaced. *(Closed by go-selfupdate-lib `v1.25.0`, Amendment A2: refused on Linux too.)* At `v1.25.0` a symlink invocation is refused on every platform, with the
   library's error and A2-3's hint, and nothing is replaced (Confirmation 10 and 11).
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
* Seeing a symlink invocation on Linux: go-selfupdate-lib's, [maccavelli/go-selfupdate-lib#9](https://github.com/maccavelli/go-selfupdate-lib/issues/9) (Amendment (2026-11-09)). *(Closed by go-selfupdate-lib `v1.25.0`, Amendment A2: refused on Linux too.)*

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

## Completion (2026-11-09): 0048-PLAN carried out

[0048-PLAN-gobble-update-command.md](0048-PLAN-gobble-update-command.md) is complete. Its P3 commit is the one that
adds this section.

| Item | Status | Commit | Evidence |
| :--- | :--- | :--- | :--- |
| 1 Help: six flags, the exit status, no `--channel` | **done** | `e22954f` (P1) | `TestUpdateHelp` |
| 2 Check, JSON, contradiction, `--release` | **done** | `e22954f` (P1) | `TestUpdateCheck`, `TestUpdateJSON`, `TestUpdateContradiction`, `TestUpdateRelease` |
| 3 Apply, a bad image, an archive | **done** | `e22954f` (P1) | `TestUpdateApply`, `TestUpdateRefusesBadImage`, `TestUpdateIgnoresArchive` |
| 4 The agent guard | **done** | `e22954f` (P1), P3 | `TestUpdateAgentGuard`; live, exit 1 with the guard's line on all three hosts |
| 5 A symlink invocation | **done as amended** | `e22954f` (P1), P3 | `TestUpdateSymlink`; live, refused on macOS; on the Linux machine the target was replaced (Amendment (2026-11-09)) |
| 6 Seen failing | **done** | `e22954f` (P1) | the PLAN's P1 record: thirteen planted breaks and a control, each as expected; the `Signals` break not run, with the reason |
| 7 Live from real GitHub | **done** | P3 | the PLAN's P3 record: `--check` exit 10, `--dry-run` unchanged, `--yes` installed v0.4.0's digest, `gobble version` `gobble 0.4.0`, on all three hosts; on the Windows laptop, installed beside a running `gobble acp` |
| 8 Gates | **done** | P1, P3 | the PLAN's P1 and P3 records |

**Found at acceptance,** besides the amendment:

* The Windows leftover was cleaned live after all. The first `--yes` left `.gobble.exe.selfupdate-bak-<digits>`; the
  next update that could install, from a rebuilt binary that has `update`, removed it before downloading, as D4 has
  `CleanupPending` do. Confirmation 7's last sentence assumed only an installed release could show that.
* The library leaves `.gobble.selfupdate.lock` (`.gobble.exe.selfupdate.lock` on Windows) beside the binary, and on
  Windows a `.gobble.exe.selfupdate.cleanup` file with the pending backup.

**Still the owner's:** ~~filing the drafted go-selfupdate-lib issue;~~ ~~the first release tag that carries
`update`.~~ *(2026-11-09: `v0.5.0`, tagged by the owner on `8371b96` and published 2026-11-09T23:28:37Z, immutable, GitHub's Latest; its release run `42718701229` passed.)*
The issue was filed on 2026-11-09, on the owner's word, as [maccavelli/go-selfupdate-lib#9](https://github.com/maccavelli/go-selfupdate-lib/issues/9).

## Amendment A2 (2026-11-09, accepted): pin go-selfupdate-lib `v1.25.0`

### What changed

go-selfupdate-lib `v1.25.0` (`875dec3c9d475ca47c4b012ca43742de832ef626`, tagged 2026-11-09) fixes [maccavelli/go-selfupdate-lib#9](https://github.com/maccavelli/go-selfupdate-lib/issues/9), which
this pair's P3 found ("Amendment (2026-11-09)"). Read from the sibling clone at that tag:

* **The fix.** `resolveTarget` (`selfupdate/target.go`) also checks the path the program was started by, `os.Args[0]`,
  made absolute when it has a separator and looked up on `PATH` when it has none, when no `TargetPolicy.ExecutablePath`
  is set. It refuses when that path is a symlink to the running binary. A symlinked directory on the way is still
  followed. On macOS and Windows the result is as at `v1.24.0`.
* **The API.** One exported addition, `ErrSymlinkInvocation`, which the refusal wraps; `make apicheck` there reports
  `v1.25.0` compatible with `v1.24.0`. The `failed` event's class for it is `symlink-invocation`, not `error`.
* **The message.** `selfupdate: refusing to update through a symlink: <link> links to <target>; run <target>, or set
  TargetPolicy.ExecutablePath to it`, the old text its prefix.
* **The workflows.** `git diff --stat v1.24.0 v1.25.0 -- .github` is empty: the build and publish workflows
  `.github/workflows/ci.yml` calls are unchanged.
* **The issue** is still open on GitHub at the time of writing; the fix's commit names it. Closing it is the
  library's. *(2026-11-09 correction: the owner closed it as completed at 20:40:18Z, before A2's records were
  committed, with a comment naming `v1.25.0`, the fix commit `b8a5bd0` and the tag's commit `875dec3`. The record
  read the issue once, earlier, and did not read it again.)*

`scripts/check_libs.py --lib go-selfupdate-lib` (2026-11-09), on a scratch clone: pinned `v1.24.0`, checking
`v1.25.0`; get and tidy, version, modules, build, vet for linux, darwin and windows, test, staticcheck, lint, api,
govulncheck and advisories all `ok`; `0 failing checks`. No other module moved.

### What it breaks in gobble

`targetHint` (`internal/cli/update.go`) names `TargetPolicy.ExecutablePath`, or else `os.Executable`'s path, as the
file that "is not one", and says to install gobble under the home directory. At `v1.25.0` on Linux, a symlink
invocation is refused, but `os.Executable` there is the link's **target**: the hint would name a file that is a
regular file under the home directory, say it is not one, and tell the user to install gobble where it already is. On
macOS it names the link, as now. And for any symlink refusal the library's own line now names the target and says to
run it, so D6's "install gobble there" advice is wrong for that case on every platform.

### Decisions

* **A2-1. The pin.** `go.mod` requires go-selfupdate-lib `v1.25.0`; `go.sum` follows; nothing else moves. README's
  Stack line and `docs/architecture.md`'s requirements table name it.
* **A2-2. The workflows** (Q6). Both `uses:` lines in `.github/workflows/ci.yml` move to
  `875dec3c9d475ca47c4b012ca43742de832ef626 # v1.25.0`, so the workflows and the module name one version, as
  [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md)
  moved them at `v1.24.0` with no workflow change either.
* **A2-3. The hint** (Q7). When the target's error wraps `selfupdate.ErrSymlinkInvocation`, gobble's line after the
  library's is `hint: gobble does not update through a symlink; run the file the link points to, named above, as
  <that file> update.` It names no path of its own. Every other refusal keeps D6's hint. gobble offers no way to set
  `TargetPolicy.ExecutablePath`.
* **A2-4. The records.** D6's dated note, Confirmation 5's amendment and the Linux Consequence of
  "Amendment (2026-11-09)" each gain a note that `v1.25.0` closes them; README's Linux sentence goes; Confirmation 5
  holds on every platform again.

### Questions for the owner

* **Q6. The CI workflow pins.**
  * **(a) Move both to `875dec3 # v1.25.0`.** *(recommended, chosen)* One version named everywhere; no workflow
    changed.
  * (b) Leave them at `74ab5dd # v1.24.0`, since the workflows are identical; the comment then names an older version
    than `go.mod`.
* **Q7. The symlink hint.**
  * **(a) A symlink-specific hint, chosen by `errors.Is(err, selfupdate.ErrSymlinkInvocation)`, naming no path.**
    *(recommended, chosen)* Correct on every platform; the library's line names both paths.
  * (b) No gobble hint after a symlink refusal: the library's line already says to run the target, but also says to
    set `TargetPolicy.ExecutablePath`, which a gobble user cannot.
  * (c) Keep D6's hint for every refusal. On Linux it names the wrong file (above).

#### Answers (2026-11-09)

The owner: "Q6 a, Q7 a, proceed". Q6 (a): both workflow pins move to `v1.25.0`'s commit. Q7 (a): the symlink hint, chosen by `ErrSymlinkInvocation`.

### Consequences

* Good, because a symlink invocation is refused on Linux too, so a link into a tree another tool manages under the
  home directory, such as Homebrew on Linux in `~/.linuxbrew`, is never overwritten; the first amendment's Bad
  Consequence is closed.
* Good, because the hint is chosen by the library's sentinel, not by its text.
* Neutral, because a Linux user who ran gobble through a link, and whose update worked at `v1.24.0`, is now refused and
  told to run the target. No gobble release has shipped `update` yet, so no user has relied on it.

### Confirmation

9. `go list -m github.com/maccavelli/go-selfupdate-lib` gives `v1.25.0`; `git diff` of `go.mod` changes that one line.
10. `TestUpdateSymlink` sees the library's `links to` line and A2-3's hint, and not D6's; a new test of a target
    outside the home directory sees D6's hint naming it. Each is seen failing against a planted break.
11. Live, on a scratch home, as P3: on the Linux machine and macOS a symlink invocation exits 1 with A2-3's hint and
    replaces nothing; the direct steps (check, dry run, install, agent guard, and on the Windows laptop the install
    beside `gobble acp`) pass as in P3 on all three hosts.
12. `make preflight`, `make race`, `scripts/check_libs.py --lib go-selfupdate-lib`, and the tests and race tests on the
    Linux machine and the Windows laptop pass.

### Completion of A2 (2026-11-09)

Carried out by 0048-PLAN's "Amendment of 2026-11-09: P4"; its P4 record has the evidence.

| Item | Status | Evidence |
| :--- | :--- | :--- |
| 9 The pin | **done** | `go list -m` gives `v1.25.0`; `go.mod` changed one line; no other module moved |
| 10 The tests | **done** | `TestUpdateSymlink` and `TestUpdateOutsideHome`; four planted breaks and a control, each as expected |
| 11 Live | **done** | 11 of 11 on macOS, the Linux machine and the Windows laptop; the Linux symlink invocation refused with A2-3's hint |
| 12 Gates | **done** | preflight, race, `check_libs.py`, and the tests and race tests on the two dev hosts |

**Still the owner's:** ~~closing go-selfupdate-lib#9 (the library's);~~ ~~the first release tag that carries
`update`.~~ *(2026-11-09 correction: nothing is left.* [maccavelli/go-selfupdate-lib#9](https://github.com/maccavelli/go-selfupdate-lib/issues/9) *was already closed as completed at
20:40:18Z, before this section was written. The first release that carries `update` is `v0.5.0`, tagged by the owner on `8371b96` and published 2026-11-09T23:28:37Z, immutable, GitHub's Latest; its release run `42718701229` passed.)*
