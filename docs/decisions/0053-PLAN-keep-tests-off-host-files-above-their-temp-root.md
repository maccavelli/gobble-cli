---
status: proposed
date: 2026-11-11
associated-madr: "0053-MADR-keep-tests-off-host-files-above-their-temp-root.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement stopping every upward walk at the test binary's temp root, and checking the trust menu against its own layout

Associated MADR: [0053-MADR-keep-tests-off-host-files-above-their-temp-root.md](0053-MADR-keep-tests-off-host-files-above-their-temp-root.md)

## Goal

No file above a test binary's temp root changes a test's result, and no test writes above it, on any host. The trust
menu's tests hold for a temp root of any length. A committed check plants every known file above a long temp root and
fails on the first test that reads one. The plan is complete when the MADR's Confirmation 1 to 6 is met.

## Scope

**Base.** `main` at `86ad4a7`, equal to `origin/main`. The tree is clean except for the untracked 0051 and 0052
records, which are the owner's and are not touched, and this pair.

**In scope**, as the MADR's D1 to D4 under the owner's answers to Q1 to Q4:

* **P1:** `scripts/check_test_isolation.py`, and the `check-test-isolation` target in `Makefile`.
* **P2, the hook and the walks:**
  * `internal/fsx`: a new `ceiling.go` and `ceiling_test.go`; `git.go`, `ignore.go`, and their tests;
  * `prompt/context.go`, `prompt/trust.go`, and their tests;
  * `.golangci.yml`'s `forbidigo` rule.
* **P3, the `TestMain`s:**
  * `acpserver/testmain_test.go` and `internal/cli/testmain_test.go`;
  * a new `testmain_test.go` in `prompt`, `internal/config`, `internal/fsx` and `tool/builtin`;
  * `TestWalksBounded` in each of the six.
* **P4:** `internal/cli/trust_test.go`.
* **P5, CI, docs and records:**
  * `.github/workflows/ci.yml`, under Q4 (a);
  * `docs/architecture.md`;
  * this pair's Completion;
  * `docs/README.md`'s rows.

**Out of scope:**

* any change to what users' walks find;
* `permission.TrustStore.Lookup`, and tests whose working directory is the source tree (MADR, "Not in scope");
* the untracked 0051 and 0052 records;
* posting on gobble-cli#17 or closing it, which is drafted for the owner;
* a push or a tag.

## Conventions

* **C1.** Each phase first re-reads `git status`, `HEAD` and `origin/main`. Anything not named in Scope is left alone.
* **C2.** Commands write to a scratch log, and their exit status is taken before any filter.
* **C3. Deviations.** Any of these stops the phase and prompts, with evidence and resolutions, before anything else is
  changed:
  * a failing test that the step does not predict;
  * a lint finding;
  * a break that does not fail;
  * a count from the isolation check other than the one the step states.
* **C4. Breaks.** Every break runs on its own scratch copy of the tree, from a Python script in the session scratchpad.
  The tree is never dirtied to see a test fail. Each break states which tests must fail, and the record quotes the
  failure.
* **C5. Gates:**
  * `make pre-add-check FILES="…"` on every changed Go file;
  * `make preflight` and `make race`;
  * `go test ./...` and the race tests on the Linux machine and the Windows laptop;
  * `make check-records markdownlint`;
  * the identifier scan of the staged diff.

  The agent stages, and the owner commits with `git commit --no-edit`.

## Implementation Steps

### P0 — the records

1. The owner's answers to Q1 to Q4 go into the MADR, which becomes `accepted`, and this plan becomes `in-progress`.
   Steps that depend on an answer say so, and the ones an answer drops are struck through, not deleted.
2. `docs/README.md`'s two rows; `make check-records markdownlint`. Staged; the owner commits.

### P1 — the isolation check (D4), seen failing at the base

1. **C1.**
2. **`scripts/check_test_isolation.py`,** stdlib only.
   * **Setup:** it makes its root with `tempfile.mkdtemp(prefix="gobble-isolation-")` and plants D4's four files there.
   * **The temp root:** it makes a temp root below them. The depth is chosen so the root's absolute path is at least
     130 characters, and at most 160 so a Windows path stays short of 260.
   * **The run:** it sets `TMPDIR`, `TMP` and `TEMP` to that temp root, then runs `go test -count=1 ./...` at the
     repository root and in `acpext/`, logging both under its root.
   * **The report:** the failing top-level tests by package, and every file under its root that is neither planted
     nor under the temp root.
   * **The result:** exit 1 if either list is non-empty or a `go test` run failed, 0 otherwise. It removes its root
     unless `--keep` is given.
   * **No ignore change:** `.gitignore:27` already ignores `__pycache__/`.
3. **`Makefile`:**
   * `check-test-isolation: ; python3 scripts/check_test_isolation.py`, added to `.PHONY`;
   * not added to `preflight`, since it is a second full test run.
4. **Seen failing.** Run `make check-test-isolation` on the tree at the base, with exit status 1. The report names at
   least:
   * MADR F3's 27 tests;
   * F5's 3;
   * F4's `.gobble/settings.yaml` and `.gobble/settings.yaml.lock`.

   The record quotes the report. A count other than that is a deviation (C3).
5. **Gates** (C5; no Go file changes). Staged; the owner commits. The commit's check fails until P4, by design. It is
   in no gate yet.

### P2 — the ceiling and the walks (D1, under Q1)

1. **C1.**
2. **`internal/fsx/ceiling.go`:**
   * the ceiling is an unexported `atomic.Pointer` to its cleaned absolute path and `os.FileInfo`;
   * `SetWalkCeiling(dir string) error`: `""` clears it; otherwise the directory is made absolute and stat'ed, and a
     missing path or a non-directory is an error;
   * `WalkCeiling() string` returns the set path, or `""`;
   * `AtWalkCeiling(dir string) bool`: false when none is set or `dir` cannot be stat'ed, otherwise `os.SameFile`.

   The doc comment names 0053-MADR D1 and says that only a test binary sets the ceiling.
3. **The walks check it before reading a directory:**
   * `fsx.gitDirs` (`internal/fsx/git.go:22`) returns "no repository";
   * `fsx.repoTop` (`internal/fsx/ignore.go:132`) returns `""`;
   * `prompt.ContextFiles` (`prompt/context.go:77`) ends its walk;
   * in `prompt.ProjectSkillDirs`, the first loop (`prompt/trust.go:18`) ends with no stop found, and the second
     (`:29`) ends before reading.
   * Under Q1 (b), only the two `prompt` walks check it, and the `fsx` walks are unchanged. Under Q1 (c), `repoTop`
     is unchanged.
4. **Tests.** Each test sets the ceiling through a helper that restores the previous value with `t.Cleanup`. These
   tests do not call `t.Parallel()`.
   * **`internal/fsx` `TestWalkCeiling`:**
     * a missing path and a regular file are errors;
     * an unnormalised spelling of the ceiling matches it (`<dir>/x/..`, and a trailing separator);
     * `""` clears it, and `WalkCeiling` reports each state.
   * **`internal/fsx` `TestGitRepoStopsAtCeiling` and `TestRepoTopStopsAtCeiling`.** The fixture is
     `<tmp>/.git/HEAD`, with the ceiling at `<tmp>/c` and the walk from `<tmp>/c/p`. With the ceiling set, no
     repository is found. With it cleared, `<tmp>` is found.
   * **`prompt` `TestContextFilesStopAtCeiling`.** `<tmp>/AGENTS.md` lies above a ceiling at `<tmp>/c`. Set, it is
     excluded; cleared, it is the first file.
   * **`prompt` `TestProjectSkillDirsStopAtCeiling`.** The fixture is `<tmp>/.agents/skills` and `<tmp>/.git` above
     the ceiling. Set, neither is used: no skill directory is listed, and the walk does not stop at `<tmp>`. Cleared,
     `<tmp>/.agents/skills` is listed.
5. **`.golangci.yml`:** a `forbidigo` entry with pattern `^fsx\.SetWalkCeiling$` and pkg
   `^github\.com/maccavelli/gobble-cli/internal/fsx$`. Its message is "only a test binary sets the walk ceiling
   (0053-MADR D1)". The existing `_test.go` exclusion (`.golangci.yml:147-149`) exempts the tests.
6. **Breaks** (C4):
   * each walk's check removed, one at a time, must fail its test above;
   * `os.SameFile` replaced by a string comparison must fail `TestWalkCeiling`'s unnormalised case;
   * `SetWalkCeiling` accepting a file must fail `TestWalkCeiling`;
   * a non-test file in `internal/config` calling `fsx.SetWalkCeiling` must give `golangci-lint run ./internal/config/`
     a `forbidigo` finding with the message above.
7. **`make check-test-isolation`:** the same report as P1, since no `TestMain` sets the ceiling yet. A different count
   is a deviation.
8. **Gates** (C5). Staged; the owner commits.

### P3 — the `TestMain`s (D2, under Q2)

1. **C1.** List the `func TestMain(` definitions in the six packages. The expected ones are `acpserver` and
   `internal/cli` only. Anything else is a deviation before any edit.
2. **Each `TestMain` calls `fsx.SetWalkCeiling(os.TempDir())` before `m.Run()`.** On an error it prints
   `walk ceiling: <err>` to stderr and exits 2.
   * `internal/cli` does it in the child that runs the tests, after `checkNetGuard`. Its doc comment gains the
     sentence and 0053-MADR D2.
   * `acpserver` does it after `mcptest.ServeStdioIfAsked()`.
   * `prompt`, `internal/config`, `internal/fsx` and `tool/builtin` each gain a `testmain_test.go` that does only
     this.
   * Under Q2 (b), every package with tests gets one, through a shared helper in a test-only package. That is first
     amended into this step with its file list. Under Q2 (c), this step and step 3 are struck, and P2's helper is
     called by each exposed test instead.
3. **`TestWalksBounded` in each of the six.** It fails unless `fsx.WalkCeiling()` is `filepath.Clean(os.TempDir())`
   made absolute, and its message names `TestMain`.
4. **Breaks** (C4): each package's `TestMain` call removed, one at a time, must fail that package's
   `TestWalksBounded`. That is six breaks.
5. **`make check-test-isolation`:** MADR F3's 27 tests no longer fail, and no file is left above the temp root. F5's
   three trust tests still fail. Any other result is a deviation.
6. **Gates** (C5), with `make archtest` confirming no import rule is crossed by the tests' new `internal/fsx` imports.
   Staged; the owner commits.

### P4 — the trust tests against `ask.Layout` (D3, under Q3)

1. **C1.** Read `askOn` (`internal/cli/permission.go:185`) for the width method and the glyph set the asker draws
   with. Compare the frame left in the scrollback after an answer with `ask.Layout`'s rows. If what stays is not
   `Layout`'s rows followed by the answer's prompt row, that is a deviation (C3), before any edit.
2. **Two helpers in `internal/cli/trust_test.go`:**
   * `trustQuestion(proj string, parentOffered bool) ask.Question`, built from literals and the two paths;
   * `drawnRows(screen string, q ask.Question, width int) int`. It counts how many times `ask.Layout(q, width, …)`'s
     rows, all but the last, appear on the screen as one contiguous run of rows, each trimmed on the right, in order.
3. **The three tests change.**
   * `TestTrustQuestion` (`:268-278`), `TestTrustQuestionNoParent` (`:313-316`) and `TestLineViewTrust` (`:342-348`)
     replace their row-substring checks and their title count with `drawnRows(…) == 1`, at `promptWidth`'s 120. That
     keeps "drawn once" exact when the title spans rows.
   * The answer's row, the `untrustedLine` count and the saved-trust checks stay as they are.
   * Under Q3 (b), the helper is a wrap-aware matcher instead, and the MADR's D3 is amended first. Under Q3 (c), this
     phase is struck, and the host gates' short-temp-root requirement goes into `docs/architecture.md` in P5.
4. **Breaks** (C4). Each must fail its test:
   * one expected label changed;
   * the expected layout taken at 80 columns;
   * the cancel answer dropped from the expected question.
5. **Under a long temp root.** On scratch, run the three tests with a 200-character `TMPDIR` and with the default.
   Both pass.
6. **`make check-test-isolation`:** exit 0, no failing test and no file above the temp root.
7. **Gates** (C5). Staged; the owner commits.

### P5 — CI, docs, hosts and the records

1. **C1.**
2. **Under Q4 (a):** `.github/workflows/ci.yml` gains a `test-isolation` job on `ubuntu-24.04`, with the same checkout
   and Go setup pins as the existing Linux job, running `make check-test-isolation`.
   * The job is new, and its seen-failing evidence is P1 step 4: the same target, at the base, exit 1.
   * It is not pushed to see it fail, since a push needs the owner's word.
   * Under Q4 (b) or (c), the file is unchanged.
3. **`docs/architecture.md`:**
   * `internal/fsx`'s row (`:146`) adds "the walk ceiling a test binary sets (0053-MADR D1)";
   * the gates paragraph names `make check-test-isolation`, where it runs, and that the host gates no longer need a
     clone outside the home folder.
4. **Hosts.** On the Linux machine and the Windows laptop, each from a fresh clone at this phase's tree: `go test
   ./...`, the race tests and `make check-test-isolation`.
   * On the Windows laptop, the clone and its temp root are under the home folder, the case 0049-PLAN P5b-1 had to
     avoid. The home folder's own `AGENTS.md` is left in place.
   * All pass. A failure is a deviation.
5. **The records:**
   * the MADR's Completion section, against Confirmation 1 to 6;
   * this plan becomes `complete`, with each phase's record;
   * `docs/README.md`'s rows;
   * a comment for gobble-cli#17, drafted in the record for the owner to post.
6. **Gates** (C5). Staged; the owner commits. The push is the owner's.

## Verification

* **Confirmation 1:** P2 steps 4 and 6.
* **Confirmation 2:** P2 steps 5 and 6.
* **Confirmation 3:** P3 steps 3 and 4.
* **Confirmation 4:** P4 steps 3 to 5.
* **Confirmation 5:** P1 step 4, failing at the base, and P4 step 6 and P5 step 4, passing on three hosts.
* **Confirmation 6:** P5 step 4, and each phase's gates.

Each phase's record quotes the isolation check's report and each break's failure. Running `make
check-test-isolation` from a clean checkout of the last commit reproduces the result.

## Rollout and Rollback

* **What changes:** tests, one internal hook and its five checks, a lint rule, a script and a CI job. For users, only
  the walks' added check of an unset ceiling, which returns at once.
* **Rollback:** revert the phase's commit.
* **Order:** P1's commit holds a check that fails until P4's commit. It is wired to no gate until P5, so reverting P2
  to P4 leaves the tree as it was at the base, with the check still reporting the exposure.
* **The CI job** is added last, so CI never runs it while it fails.
