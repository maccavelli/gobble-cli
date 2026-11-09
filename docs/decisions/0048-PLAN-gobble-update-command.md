---
status: in-progress
date: 2026-11-09
associated-madr: "0048-MADR-gobble-update-command.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement shipping `gobble update` on go-selfupdate-lib, superseding 0005-PLAN F10's self-update step

Associated MADR: [0048-MADR-gobble-update-command.md](0048-MADR-gobble-update-command.md)

## Goal

* `gobble update` checks, verifies and installs gobble's GitHub releases through go-selfupdate-lib `v1.24.0`,
  composed as the MADR's D1 to D7 say.
* 0005's self-update step points here (D8).

The plan is complete when the MADR's Confirmation 1 to 8 is met. Tagging the first release that carries the command is
the owner's, and not in this plan.

## Scope

**Base.** `main` at `2e83cce` (v0.4.0), equal to `origin/main`, clean, when this plan was written. If P0 starts from
another commit, C1 records it.

**In scope,** by phase:

* P0, the records: this pair, accepted with the owner's answers to Q1 to Q5; `docs/README.md`'s 0048 rows; the
  dated supersession notes in `docs/decisions/0005-MADR-v1-feature-scope.md` and
  `docs/decisions/0005-PLAN-v1-feature-scope.md`.
* P1, the command:
  * a new `internal/cli/update.go`: `UpdateCmd`, the embedded spec, the composition, the guard, the cleanup, the
    run and the hint;
  * `internal/cli/root.go`: the `Update` field;
  * a new `internal/cli/update_test.go`;
  * `internal/cli/release_test.go`: its comment names this pair, not F10.
* P2, the docs: `README.md` (the command, and that v0.4.0 and earlier cannot update themselves), `docs/architecture.md`
  (`internal/cli`'s row), `docs/guides/developing.md` (a post-release `gobble update --check` from the previous
  release).
* P3, acceptance and the records.

**Out of scope:** the MADR's "Not decided here"; any package but `internal/cli`; `go.mod` and `go.sum`, which
should not change (P1 checks it, and a change is a deviation); a release tag; a push.

## Conventions every phase follows

* **C1. Start state.** Each phase first re-reads `git status`, `HEAD` and `origin/main`. A difference from this plan is
  a deviation.
* **C2. Seen failing.** Every new test is seen failing on a scratch copy of the tree, never in the working tree:
  against the tree before the phase, and against a planted break of the new code. The record quotes each failure.
* **C3. Evidence.** Commands write to a scratch log, with their exit status taken before any filter. Scratch runs set
  `HOME`, `USERPROFILE` and `XDG_*` into the scratch directory, and explicit `GOCACHE`/`GOMODCACHE`.
* **C4. Deviations stop the phase,** and are recorded here, and in the MADR when a decision or fact changes, before
  work goes on.
* **C5. Real effects.** Tests and P1's probes stay on loopback; the 0046 guard holds them. P3 step 2 is the one step
  that reaches GitHub on purpose: read-only API and asset downloads from `maccavelli/gobble-cli`'s public releases,
  into a scratch home, replacing only a binary built for the step. Nothing else in the plan reaches the network.
* **C6. Gates and commits.**
  * The Go files pass `make pre-add-check FILES=…`.
  * The tree passes `make preflight` and `make race`, and the identifier scan finds nothing.
  * The Linux machine and the Windows laptop run `go test ./...` and the race tests on a scratch copy at P1 and P3.
  * The agent stages. The owner commits with `git commit --no-edit`, or tells the agent to. The disclosure guard runs
    before any push.
* **C7. Identifiers.** No hostname, account name or real-machine path in a committed file. Temporary paths in quoted
  output become `<TMP>`; the dev hosts are named by platform.

## Implementation Steps

### P0 — the records

1. Record the owner's answers to Q1 to Q5 in the MADR, set it `accepted`, and set this plan `in-progress`. A changed
   answer that changes a step is written into the step here, struck through where it replaces text.
2. Add the 0048 MADR and PLAN rows to `docs/README.md`, and a row to its "I want to…" matrix beside 0047's.
3. In 0005-MADR, after the Self-update inventory row and after each bullet D8 names, a dated note:
   *(2026-11-09, [0048-MADR](0048-MADR-gobble-update-command.md) D8: superseded; …)* naming what replaced it. In
   0005-PLAN, the same note after F10 step 4 and after F10 Accept's `gobble update --check` bullet.
4. `make check-records markdownlint` passes. Staged; the owner commits.

### P0 record (2026-11-09)

C1: `HEAD` and `origin/main` at `2e83cce`; the only changes were this pair, untracked. No deviation.

1. The owner's answers, 2026-11-09: Q1 (a), Q2 (a), Q3 (a), Q4 (a), Q5 (b), quoted in the MADR's "Answers
   (2026-11-09)". Q5 (b) changes D2's probe to `version --json`, wanting `"version":"<the tag without its v>"`; a
   binary stamped `v0.4.0`/`release` from `2e83cce` printed
   `{"name":"gobble","version":"0.4.0","commit":"2e83cce…","date":"2026-11-08T23:15:40Z","go":"1.27.3","os":"darwin","arch":"arm64"}`.
   With Q5 (b), 0005-MADR's probe bullet and its TUI bullet stay 0005's, so D8 names fewer bullets than proposed.
   P1 step 3 gains `TestUpdateProbeWant`, and P1 step 5 its break.
2. `docs/README.md`: the 0048 MADR and PLAN rows, and 0048 first in the "know how `gobble update` … work" row.
3. Dated notes: 0005-MADR's Self-update inventory row, its "Pin" bullet and its Homebrew bullet; 0005-PLAN F10 step 4
   and its Accept bullet.
4. Gates: `make check-records markdownlint` exit 0, `0 issues in 0 files` (markdownlint skips MADR and PLAN files).
   The identifier scan of the diff (the local account name, the dev hosts' names, the org's mail domain, and
   real-machine home and temporary paths) found nothing. No Go file changed.

### P1 — the command (D1 to D7)

1. **`internal/cli/update.go`.**
   * `//go:embed selfupdate-release.json`, parsed once with `releasespec.Parse`. A parse error is a failure of the
     command, not a panic at start-up, so no other command depends on it.
   * `UpdateCmd` with the D1 flags, Kong `help:` text for each, and the exit-status paragraph in the command's help.
   * `updaterDeps`, a package variable of the parts a test replaces: `APIBaseURL *url.URL`, `Client *http.Client`,
     `TargetPolicy selfupdate.TargetPolicy`, and `ProbeRun` (the `VersionProbeOptions.Run`). Production leaves them
     zero.
   * `newUpdater(spec, deps, stdin, stderr) (*selfupdate.Updater, *selfupdate.StandaloneInstaller, error)`, D2's
     composition. It returns the installer so `Run` can call `CleanupPending` on it. The probe is
     `NewVersionProberWith` on `["version", "--json"]`, `Want` a function `probeWant(tag)` returning
     `"version":"` + the tag without its `v` + `"`, `Timeout` 10 s, `Run` from `updaterDeps`.
   * `(*UpdateCmd).Run(e *runEnv) error`:
     1. Build `cli.Flags` from the flags; `Flags.Request("gobble", identity().Lib)`, through the `identity` variable
        `internal/cli/version.go` already has for tests; a refusal is `usageErrorf`.
     2. When `os.Getenv("AI_AGENT") == "gobble"` and the request is not `CheckOnly`: `failf` with D5's line.
     3. Build the updater. When the request is not `CheckOnly`, `CleanupPending(e.ctx)`: ignore
        `ErrConcurrentUpdate`, warn on any other error.
     4. `cli.Run(e.ctx, u, req, cli.Options{Stdout, Stderr, Stdin: e.stdin, Interactive, JSON, Signals:
        []os.Signal{}})`, with `Interactive` from `e.out.caps`.
     5. `code := cli.Exit(stderr, res, err)`. When the error is the installer's target refusal (D6), one hint line
        after it. Return nil for 0, else `&exitError{code: code}` with no message.
2. **`internal/cli/root.go`:** `Update UpdateCmd \`cmd:"" help:"Check for, and install, a newer gobble release."\``,
   after `Version`.
3. **`internal/cli/update_test.go`**, through `Main` as the other command tests run, each with its own home (`HOME`
   and `USERPROFILE`), and a `selfupdatetest.NewGitHubServer` whose assets are the test binary's own bytes, named
   for the running platform, with a `SHA256SUMS`:
   * `TestUpdateHelp`: Confirmation 1.
   * `TestUpdateCheck`: newer release → 10, the summary on stderr, stdout empty; same release → 0, `up to date`.
   * `TestUpdateJSON`: JSON Lines, last line `"kind":"result"`, `exit_code` equals the returned code.
   * `TestUpdateContradiction`: `--check --yes`, `--check --force`, `--check --dry-run` → 2, server saw no request.
   * `TestUpdateRelease`: `--release v0.2.0` with v0.2.0 and v0.3.0 published selects v0.2.0 (`--force` where the
     library needs it for a downgrade).
   * `TestUpdateApply`: `--yes` against a stamped-release identity replaces the file at `TargetPolicy.ExecutablePath`
     (a copy of the test binary under the home); the probe's `Run` saw the staged and the installed path; the file's
     bytes are the asset's.
   * `TestUpdateRefusesBadImage`: an asset of random bytes with a matching `SHA256SUMS` → 1, `ErrIntegrity`'s text,
     target unchanged.
   * `TestUpdateIgnoresArchive`: only `gobble-<os>-<arch>.tar.gz` published → 1, no asset selected.
   * `TestUpdateAgentGuard`: `AI_AGENT=gobble` with no flag, `--yes`, `--dry-run` → 1 and D5's line, no request;
     `--check` → 10.
   * `TestUpdateSymlink` (not Windows): the executable path a symlink → 1, the library's error and the hint, target
     unchanged.
   * `TestUpdateProbeWant`: `probeWant("v0.4.0")` is `"version":"0.4.0"`, and `gobble version --json` run
     through `Main` with the identity stamped `v0.4.0`/`release` contains it. It holds the probe and the version
     command to one contract.
   * The test's identity is set through the existing `identity` variable (`internal/cli/version.go`).
4. **Checks.** `go mod tidy` leaves `go.mod` and `go.sum` unchanged (exit status of `git diff --quiet`).
   `make archtest` passes unchanged.
5. **Seen failing (C2),** each on its own scratch copy:
   * the tests against the tree before P1 (no `update` command): each fails on Kong's unknown command;
   * the guard removed: `TestUpdateAgentGuard` fails;
   * `Verifiers` emptied: `TestUpdateRefusesBadImage` fails;
   * `Signals` left nil: a test that cancels the context mid-run shows the library's handler, not `Main`'s exit, or,
     if that is not observable in-process, the record says so and why;
   * `--release` not passed through: `TestUpdateRelease` fails;
   * `probeWant` keeping the `v`: `TestUpdateProbeWant` fails.
6. **Gates (C6),** including the Linux machine and the Windows laptop. Staged; the owner commits.

### P2 — the docs

1. `README.md`: `gobble update` in the commands list, out of the not-yet-built list at line 110; a short "Updating"
   section: the command, its flags, that it installs only under the home directory, that a release up to and including
   v0.4.0 cannot update itself and is replaced by hand once, and `GH_TOKEN`/`GITHUB_TOKEN`.
2. `docs/architecture.md`: `internal/cli`'s row names `update` and what it composes.
3. `docs/guides/developing.md`: after a release, run `gobble update --check` from the previous release's binary and
   expect exit 10 naming the new tag; then, on Windows, `gobble update --yes` while another copy runs, and a second
   `gobble update --check` from the installed binary that leaves no `.gobble.exe.selfupdate-*` file. Both need the
   previous release to carry the command, so the first release with it is excepted.
4. `make check-records markdownlint` passes. Staged; the owner commits.

### P3 — acceptance

1. **Goldens:** `git diff --quiet 2e83cce HEAD -- internal/cli/testdata acpserver/testdata acpclient/acptest/testdata`
   exit 0.
2. **Live run (C5),** on macOS here, the Linux machine and the Windows laptop, each in a scratch home:
   1. Build `./cmd/gobble` from the tree with `VERSION=v0.3.1 BUILD_KIND=release` into
      `<scratch home>/.local/bin/gobble[.exe]`.
   2. `gobble version --identity` prints `v0.3.1 (release) <commit>`.
   3. `gobble update --check`: exit 10, `gobble: update available: v0.3.1 -> v0.4.0` (or the then-Latest tag; the
      record names it).
   4. `gobble update --dry-run --yes`: exit 0, `dry run: v0.4.0 verified; nothing installed`; the binary is unchanged
      (its SHA-256 before and after).
   5. `gobble update --yes`: exit 0, `updated v0.3.1 -> v0.4.0`; `gobble version` prints `gobble 0.4.0`; the file's
      SHA-256 equals the release's digest for that platform.
   6. `AI_AGENT=gobble gobble update --yes` (with the old binary rebuilt): exit 1, D5's line.
   7. macOS and Linux: a symlink to the binary, invoked: exit 1, refused, the hint.
   8. Windows laptop only: rebuild the v0.3.1 binary, start `gobble acp` from it and keep it running, run
      `gobble update --yes` from the same file: exit 0, and the installed file is v0.4.0's; stop `gobble acp`; the
      record lists any `.gobble.exe.selfupdate-*` file left beside it.
   The v0.4.0 binary has no `update`, so each run ends there. Cleaning a Windows leftover on the next run needs an
   installed binary that has `update`, so it cannot be run live before the first release that carries the command.
   The library's `cleanup_windows_test.go` covers it, and `docs/guides/developing.md`'s post-release check (P2 step 3)
   runs it from that release on. Versions are quoted as the library prints them.
3. **Records:** the MADR's Confirmation, item by item, in a "Completion" section with commits and evidence; this plan
   `complete` with a P-record per phase; `docs/README.md`'s rows.
4. **Gates (C6).** Staged; the owner commits.

## Verification

The MADR's Confirmation 1 to 8, as P1 step 3 and P3 step 2 map them: 1 to 5 by P1's tests, 6 by P1 step 5, 7 by P3
step 2, 8 by the gates.

## Rollout and Rollback

* **Rollout.** The command ships with the next tag, which the owner makes. After it, `docs/guides/developing.md`'s new
  check runs from the previous release, from the release after that one on.
* **Rollback.** Revert P1's commit: `update` becomes an unknown command again, and no other command is affected. A
  release that carries a broken `update` is fixed by the next release, installed by hand where the broken one cannot
  install it.
