---
status: in-progress
date: 2026-11-23
associated-madr: "0069-MADR-pre-add-script-follow-ups-of-0064.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement closing the pre-add script's three loose ends and testing the script

Associated MADR: [0069-MADR-pre-add-script-follow-ups-of-0064.md](0069-MADR-pre-add-script-follow-ups-of-0064.md)

*(2026-11-23: aligned with the owner's rulings in the MADR's "Owner rulings": Q1 "Fix in bash now, port later
(Recommended)", Q2 "Skip with a notice, exit 0 (Recommended)", Q3 "Own record later, if text found
(Recommended)", so the five code-quality items are outside this plan. The plan stays `proposed` until the owner
approves it.)*

*(2026-11-23, approved: the owner, relayed by the coordinator: "Approve both; 0069 then 0068 (Recommended)". 0069 is
built first; 0068 waits for it. Status `in-progress`; execution starts at `3a33d36`.)*

## Goal

`scripts/go-precheck.sh` fails on a Go file gofmt cannot parse, names in its last line only the steps that ran, and
skips a deleted Go path with a notice on standard error and exit 0 (row G8); `scripts/check_go_precheck.py` replays rows G1 to G11 offline,
and `make preflight` runs it.

## Scope

* In: `scripts/go-precheck.sh`; new `scripts/check_go_precheck.py`; `Makefile` (a `check-go-precheck` target, listed in
  `.PHONY`, called from `preflight`); `AGENTS.md` ("Pre-add checks"); `docs/guides/developing.md` ("The gates");
  `docs/README.md` rows (returned to the coordinator, not edited by the drafter).
* Out: 0064-MADR's rows S1 to S12; the five code-quality items; a Python port; other repositories' wrappers; CI files.

## Implementation Steps

Every check and every plant runs on a `git archive` copy of the tip under `<scratch>`, never on the tree. Staged
only; the owner commits. Heavy runs go through `~/.agents/skills/available-dev-hosts/scripts/run-heavy` (or
`run-on-host`); never `-race` on WSL.

**Exit criterion: no blocker, every row tested, plants caught.**

1. **P0, re-ground.** At the phase's start commit read every line this plan cites: `scripts/go-precheck.sh:39`, `:44`,
   `:47-48`, `:62`, `:105`; `Makefile:175`, `:196`; `AGENTS.md` "Pre-add checks"; `docs/guides/developing.md` "The
   gates". Record whether each holds; correct by annotation where one moved. Confirm F1 live: an unparseable file and
   both skip variables give exit 0 and `1 file(s) clean`.
2. **P1, tests first (red).** Write `scripts/check_go_precheck.py` (Python 3 standard library). `--script PATH` selects the
   script under test (default `scripts/go-precheck.sh`); `--self-test` runs every test, then every plant (below) and
   fails unless each plant makes its named test fail. A base run red for a missing `acpext` or a
   harness error is not the expected red (G19). Each test builds a scratch `git init` repository holding the
   needed `.go` files and a `acpext` directory (the lint step enters it) and a stub directory in front of the real `PATH` (no symlinked `PATH`:
   fragile on Windows) holding a stub `go` (a `tool govulncheck -version` and a run both exit 0, each call appended to
   a counter file) and a stub `golangci-lint` (counted; exits 0); gofmt is the real one. Every test runs under `bash`
   from `PATH` and also under `/bin/bash` where that is another program (bash 3.2 on this Mac). Tests, each mapped to rows:
   * T1 (G1): `bad.go` containing `package`; skip both; expect exit 1, `gofmt:` header and gofmt's message, no `clean`.
   * T2 (G2): a parseable unformatted file; expect exit 1, `are not formatted`, the file named.
   * T3 (G3): both files in one call; expect exit 1, both outputs.
   * T4 (G4): a clean file, no skip variables; expect exit 0 and `(gofmt, golangci-lint, govulncheck)`.
   * T5 (G5, G6, G7): the three skip combinations; expect exit 0 and the three closing lines.
   * T6 (G8): a named missing `.go` path alone, with a clean file, and a tracked-then-removed file in the
     no-argument mode; expect exit 0 and a one-line notice naming the path on standard error, the same text in all three.
   * T7 (G9, G10): `README.md` only: exit 0, stub counters zero; `README.md` plus a clean `.go`: gofmt saw only the
     `.go`.
   * T8 (G11): the directory holding the real `gofmt` removed from `PATH`: exit 2 and the message.
   * T9 (G16): the whole suite under `/bin/bash` 3.2; a run that cannot find it says so and is not a pass.
   T6 also pins G18: N counts existing files only.
   Run T1, T5 (lint and vuln combinations) and T6 against the unchanged script. **Expected red:** T1 (exit 0), T5
   (closing line), T6 (no notice). T2, T3 partly, T4, T7, T8 are green already. A traceback or a harness error is not the
   expected red; fix the harness first. Record each test's output, the red ones included.
3. **P1 plants** (one production change each, on a scratch clone of the fixed script; the self-test applies them with
   `assert` before replace and reports which test turned red):
   * PL1: the `gofmt` status check removed: T1 fails (T3 still exits 1 from the unformatted branch, so only T1 is sure).
   * PL2: the closing line restored to the fixed three names: T5 fails.
   * PL3: the missing-file notice removed: T6 fails.
   * PL4: the no-Go-file early exit removed: T7 fails only because T7 runs without the skip variables, so lint and
     `go` are reached and the stub counters are non-zero.
   * PL5: the `need gofmt` guard removed: T8 fails.
   * PL6: `failed=1` removed from the unformatted branch: T2 fails.
   * PL7: an expansion of a possibly empty array added under `set -u`: T9 fails under bash 3.2 only.
4. **P2, the script.** At `:62` capture gofmt's standard output and standard error separately (a scratch file removed on
   exit) and its status; a non-zero status prints the message under `gofmt:` and sets `failed`; the unformatted list
   prints as before. At `:39` and `:47-48` print the notice (row G8) for a missing `.go` path (same code for both modes). Build
   the closing line from the steps that ran. Update the script's header comment (`Exit codes`, the deleted-path rule).
   `bash -n` under the host's bash and, if available, bash 3.2.
5. **P3, wiring.** Makefile: `check-go-precheck` runs `$(PYTHON) scripts/check_go_precheck.py --self-test`, is added to
   `.PHONY` and to `preflight` beside `check-records`.
6. **P4, docs.** `AGENTS.md` and `docs/guides/developing.md`: the closing line names the steps that ran, gofmt's own
   failure fails the add, and the deleted-path rule in the words of row G8; the sample line in the guide is checked
   against a real run. `docs/README.md` index rows (see the hand-back). `make check-records`, `make markdownlint`.
7. **P5, gates.** `bash -n` exit 0; `scripts/check_go_precheck.py --self-test` exit 0 with every plant caught; `make
   lint` (both modules, "0 issues."); `make preflight` through `run-heavy`, its pre-add step ending with
   `(gofmt)` and its `check-go-precheck` step passing; a `git archive` copy with an unparseable Go file fails `make preflight` (G17: the early gofmt step passes, the pre-add
   step fails); the tree's `git status` shows only this plan's files. The harness was not run on Windows by hand; its first
   Windows run is CI's preflight (G15).

## Verification

The red run of P1 and its three expected failures; the green run after P2 with the seven plants each failing the named
test; the real script on a scratch copy with an unparseable file (exit 1) and with `make preflight`.

## Rollout and Rollback

Staged only; the owner commits and pushes after the pre-add check on the changed files. Rollback: revert the commit; the
script's exit codes (`0`, `1`, `2`) and variables do not change.

## Execution record

### 2026-11-23, execution at `3a33d36`, pass 1 of P0 to P5

* **P0, re-ground.** Every cited line holds: `scripts/go-precheck.sh:39`, `:44`, `:47-48`, `:62`, `:105`;
  `Makefile:175` (`pre-add-check`), `:196` (the preflight step); `AGENTS.md` "Pre-add checks"; `docs/guides/developing.md`
  "The gates". F1 live on a scratch copy: `bad.go` containing `package`, both skip variables set: gofmt's message on
  standard error, then `go-precheck: 1 file(s) clean (gofmt, golangci-lint, govulncheck).`, exit 0. A named missing
  `.go` path: exit 0, no word.
* **P1, tests first.** `scripts/check_go_precheck.py` written (T1 to T9, plants PL1 to PL7). The first run against the
  unchanged script was not a valid red: it ran the real `go` and `golangci-lint`, because `BASH_ENV` (set on this host)
  is sourced by a non-interactive bash and rewrites `PATH` in front of the stub directory. That was a harness defect
  (G19's kind), not the expected red; the harness now removes `BASH_ENV` and `ENV` from the child environment and the
  step was re-run. Valid red against the unchanged script: T1 fails (exit 0 and `1 file(s) clean`), T3 fails (no gofmt
  error line; the unformatted list still prints), T5 fails (closing line names `golangci-lint` with the lint step
  skipped), T6 fails (no notice). T2, T4, T7, T8 and T9 (under `/bin/bash` 3.2) green, as the plan expected. The fix of
  the harness is the only rework inside P1; it is not a second build pass.
* **P2, the script.** `go-precheck.sh`: `add_go` prints `go-precheck: skipping <path>: no such file (deleted?)` for a
  missing `.go` path in both modes; gofmt's standard error goes to a `mktemp` file (removed on exit) and its status is
  read; a non-zero status prints `gofmt: failed with exit status N:` and the message and sets `failed`; the closing line
  is `go-precheck: N file(s) clean (<steps that ran>).` built from a string (no empty-array expansion). Header comment
  updated. `bash -n` exit 0 under `bash` 5 and `/bin/bash` 3.2.
* **P3, wiring.** `Makefile`: `check-go-precheck` (`$(PYTHON) scripts/check_go_precheck.py --self-test`), in `.PHONY`
  and in `preflight` after `check-records`.
* **P4, docs.** `AGENTS.md` ("Pre-add checks", the preflight list) and `docs/guides/developing.md` ("The gates", the
  preflight list, now eleven items) state the closing-line rule, gofmt's own failure and the deleted-path rule. The
  sample line in the guide was checked against a real run: `make pre-add-check FILES="internal/archtest/check.go
  internal/archtest/check_test.go"` printed `go-precheck: 2 file(s) clean (gofmt, golangci-lint, govulncheck).`.
  `docs/README.md` row 0069 PLAN set to `in-progress`.
* **P5, gates.** Green: `--self-test` exit 0 with PL1 to PL7 caught by T1, T5, T6, T7, T8, T2, T9 (each plant on a scratch
  copy of the script, caught only by its named test failing). `make check-records` exit 0. `make markdownlint` 0
  issues. Identifier scan (`denyscan.py --changed`): 0 hits in 7 files. `make preflight` through `run-on-host`:
  macOS, Linux (landed in WSL `Ubuntu-24.04` of the Windows laptop, chosen by `pick-host`, not the Linux machine) and
  Windows (git-bash, the harness's first Windows run: T1 to T8 green under Git's bash; no bash older than 4.4 there,
  so T9 skipped and PL7 reported not catchable, not counted as caught). Each pre-add step ended `(gofmt)`. G17: a
  `git archive` copy with an unparseable `broken_g17.go` made `make preflight` exit 2 at the pre-add step with
  `gofmt: failed with exit status 2:`.
* **Readings, not deviations.** (1) T9 and PL7 need a bash older than 4.4; on a host without one the harness prints
  `skip T9` or `n/a` and does not count it as a pass or a catch (CI's Linux and Windows jobs). The plan's "a run that
  cannot find it says so and is not a pass" is met without failing preflight there. PL7 was caught on the macOS host,
  where `/bin/bash` is 3.2. (2) The Windows host prints `CYG_SYS_BASHRC: unbound variable` from `/etc/bash.bashrc` on
  every bash start under the queue; it is the host's startup file, present on steps this record does not touch.
* **Not done.** No run on the Linux machine natively (pick-host chose WSL); nothing staged or committed (the brief
  forbids it; the owner stages and commits).

### 2026-11-23, deviation: T8 fails on CI hosts with a second gofmt

CI run 45259145746 on `f6c6b47` failed on ubuntu-24.04 and windows-2025 with `FAIL T8 ... T8: exit 2 expected`
(rc=0, stdout `go-precheck: 1 file(s) clean (gofmt).`); macOS passed. Cause: `scripts/check_go_precheck.py` T8 (G11,
gofmt missing from PATH) hid gofmt by removing only the directory of the first `gofmt` found, and those runner images
carry a second gofmt on PATH, so the script under test still found one. The dev hosts have a single Go install each,
which is why the case passed there. Owner ruling (2026-11-23): "Hide every gofmt on PATH (Recommended)". Fix: the
harness now removes every PATH directory that holds a gofmt, found with the lookup the script uses (`command -v`),
mirrored as `shutil.which("gofmt", path=<entry>)` per entry, which also tries the Windows `.exe` names. When git lives
in a removed directory, a `git` wrapper in the stubs directory reaches it, so the case no longer depends on git and
gofmt sharing a directory. `--self-test` gained a case that plants a second gofmt directory (a scratch copy of the real
gofmt) and shows the old single-directory logic still sees gofmt while the new logic does not. `--self-test` passed on
macOS, Windows (git-bash) and Linux (the second-gofmt case ran on each), and `make check-go-precheck` passed on macOS. This
plan's acceptance is met only once CI passes on the fix commit.

### 2026-11-23, deviation 2: PL5 not caught on Ubuntu CI

CI run 45263844075 on `4e07a74`: windows-2025 and macos-15 pass; ubuntu-24.04 fails in `make check-go-precheck`
(Makefile:263) with `ok T1..T8 [/usr/bin/bash]`, `ok T8 second gofmt: old logic fooled, new logic hides every gofmt`,
`plant PL5: NOT caught: T8 stayed green`, `self-test: FAILED`. PL5 (`scripts/check_go_precheck.py`) turns
`need gofmt || exit 2` into `need gofmt || true`.

Cause, confirmed by a read-only probe (a PL5 copy of the real script run through the harness's own `run_case`, T8's
case; the CI log itself was not read, so the runner's layout is inferred from the reproduction):

* On the macOS dev host (gofmt in its own directory, `mktemp` in `/usr/bin`) the PL5 copy exits 1: stderr is
  `go-precheck: gofmt not found in PATH.`, `gofmt: failed with exit status 127:`, `line 89: gofmt: command not found`,
  then the two skip lines. T8's `rc == 2` check is red there, so PL5 is caught on this host.
* With one directory holding gofmt, `mktemp`, `sed`, `rm` and `git` and nothing else on `PATH` (a layout where gofmt sits
  in a system directory, as the Ubuntu runner image is inferred to have), hiding every gofmt directory also hides
  `mktemp`. The PL5 copy then exits 2: stderr is `go-precheck: gofmt not found in PATH.` and `pl5.sh: line 87: mktemp:
  command not found`. The exit 2 comes from `gofmt_err="$(mktemp)" || exit 2` (`scripts/go-precheck.sh:87`), with no gofmt
  run at all. T8 checks only the exit code and the guard's line, so it stays green.
* The brief's first reading (the gofmt run fails and also exits 2) was wrong: that path exits 1. An assertion that
  forbids only a `gofmt: failed` line would not catch the CI case.

Owner ruling (2026-11-23, relayed by the coordinator): "Exact stderr + CI-layout case (Recommended)": T8 requires stderr
to be exactly the guard's line; a second T8 case rebuilds CI's layout (gofmt sharing a folder with mktemp and the other
tools), so PL5 is shown caught on this Mac, the Linux machine and the Windows laptop before the push.

Files in scope: `scripts/check_go_precheck.py` and the 0069 records. The script is unchanged. Changes: T8 asserts
stderr is exactly `go-precheck: gofmt not found in PATH.`; new test T8b runs the same assertions on a `PATH` whose only
directory holds gofmt together with `mktemp`, `sed`, `rm` and `git`; new plant PL5b (the PL5 text) must turn T8b red.
Row G11 in the MADR is amended to match. 0069 is not complete: it closes only when CI is green on the fix commit.
