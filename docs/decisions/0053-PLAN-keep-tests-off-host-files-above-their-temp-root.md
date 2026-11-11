---
status: in-progress
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
2. **`scripts/check_test_isolation.py`,** stdlib only. *(Amended by deviation 1: two sibling temp roots, `GOTMPDIR`
   at the second.)*
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
2. **`internal/fsx/ceiling.go`:** *(Amended by deviation 1: the ceiling is a set, `SetWalkCeiling(dirs ...string)`,
   and `WalkCeiling() []string`.)*
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
     (`:29`) ends before reading. *(Replaced by deviation 2: the two loops become one walk with one check.)*
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
2. **Each `TestMain` calls `fsx.SetWalkCeiling(os.TempDir())` before `m.Run()`.** *(Amended by deviation 1:
   `fsx.SetWalkCeiling(os.Getenv("GOTMPDIR"), os.TempDir())`.)* On an error it prints
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
   made absolute, and its message names `TestMain`. *(Amended by deviation 1: it fails unless the ceiling holds
   `os.TempDir()`, and `GOTMPDIR` too when that is set.)*
4. **Breaks** (C4): each package's `TestMain` call removed, one at a time, must fail that package's
   `TestWalksBounded`. That is six breaks. *(Added by deviation 1: in one package, `TestMain` passing `os.TempDir()`
   alone, run with `GOTMPDIR` set to another directory, must fail `TestWalksBounded`.)*
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

### P0 record (2026-11-11)

* **The answers:** Q1 to Q4, each (a). They are recorded in the MADR's "Answers (2026-11-11)", and the MADR is
  `accepted`.
* **The records:** this pair and its `docs/README.md` rows passed `make check-records markdownlint`. The owner
  committed them as `4fc6ef4`.
* **Execution approved:** the owner's "proceed" on 2026-11-11, so this plan is `in-progress`.
* **Not changed:** the steps' alternatives under (b) and (c) are not taken. They stay in the text as the options that
  were offered.

### Deviation 1 (2026-11-11, in P1): the test binary has two temp roots

* **Found.** P1 step 4's `make check-test-isolation` ran on the base code on the three hosts.
  * On macOS and the Windows laptop it failed the same 30 tests, and found the same two stray files.
  * On the Linux machine it reported 0 failing tests and 0 files.
* **Cause.** That host exports `GOTMPDIR`, and Go's `t.TempDir()` is made under `GOTMPDIR` when that is set (Go 1.27
  `src/testing/testing.go:1575-1615`). So the tests' directories were not under the script's planted root.
  * A scratch probe there (`TestZZProbe`, on the remote copy only, since removed) logged `TMPDIR` and `os.TempDir()`
    at the planted root, and `t.TempDir()` under the host's `GOTMPDIR`.
  * Nothing in the tree's code was wrong. The MADR's F2, and with it D2's choice of `os.TempDir()`, assumed that
    `t.TempDir()` is always under `os.TempDir()`.
* **Consequence had it gone unnoticed.** D2's ceiling would match no `t.TempDir()` on such a host, and its
  `TestWalksBounded` would still pass.
* **Decision.** The owner chose "a ceiling at both roots". It is recorded as the MADR's amendment A1, which amends F2,
  D1, D2, D4 and Confirmation 3 and 5.
* **Steps amended:** P1 step 2, P2 step 2, P3 steps 2 to 4. Each is annotated where it stands.
* **Files added to the phase:** none.

### Deviation 2 (2026-11-11, in P2, before any code): one walk for the project's skills

* **Found.** P2 step 6's breaks were traced before step 3 was written, and one of them cannot fail as planned.
  * `prompt.ProjectSkillDirs` (`prompt/trust.go:12-40`, at `6bcb898`) walks twice. The first loop (`:18-28`) only
    finds `stop`, the nearest `.git`. The second (`:29-38`) collects `.agents/skills` from `cwd` up to `stop`, or to
    the root.
  * With a ceiling, the second loop's check alone ends the walk at the ceiling, whatever the first loop found.
  * So removing the first loop's check changes no result. It only stats a `.git` above the ceiling. That break
    could not fail, and the MADR D1's "the ceiling is never read" would go unchecked there.
* **Decision.** The owner chose "one loop, one check", recorded in the MADR as amendment A2.
  * `ProjectSkillDirs` becomes one walk from `cwd`. It stops at the ceiling before reading it. Otherwise it appends
    the directory's `.agents/skills`, except home's, and stops after a directory holding `.git`, or at the root.
  * That is the same result as the two loops: the second loop also took the `.git` directory's own `.agents/skills`
    before it stopped.
  * The existing `TestTrustFiles` and `TestProjectSkillDirsOutsideRepository` hold it.
* **Steps amended:** P2 step 3's `ProjectSkillDirs` item, annotated where it stands.
  * Step 4's `TestProjectSkillDirsStopAtCeiling` also puts a `.agents/skills` at the ceiling, which must not be
    listed, and a `.git` above it.
  * Step 6's breaks for `ProjectSkillDirs` are one: the walk's check removed.
* **Files added to the phase:** none.

### P2 record (2026-11-11)

Scripts and logs are in the session scratchpad (`p2_0053_breaks.py`, `p2_0053_*.log`, `p2_*.log`).

1. **C1.**
   * `HEAD` was `6bcb898`, P1, committed by the owner.
   * `origin/main` was `86ad4a7`.
   * The tree held only the owner's untracked 0051 and 0052 records.
   * Deviation 2 was found and answered before any code.
2. **`internal/fsx/ceiling.go`,** as A1 amends it.
   * **The value:** an unexported `atomic.Pointer` to a slice of `{path, os.FileInfo}`.
   * **`SetWalkCeiling(dirs ...string) error`:**
     * an empty value is skipped, and none clears the ceiling;
     * each directory is made absolute and stat'ed, and a missing path or a non-directory is an error;
     * every directory is checked before any is stored, so a refusal leaves the ceiling as it was.
   * **`WalkCeiling() []string`** returns the set paths, or nil.
   * **`AtWalkCeiling(dir) bool`** is one atomic load when none is set, and otherwise one `os.Stat` and `os.SameFile`
     against each.
3. **The walks.**
   * `fsx.gitDirs` (`internal/fsx/git.go`) and `fsx.repoTop` (`internal/fsx/ignore.go`) check the ceiling first in
     each step, and return "not found".
   * `prompt.ContextFiles` (`prompt/context.go`) breaks before `fromDir`.
   * `prompt.ProjectSkillDirs` (`prompt/trust.go`) is one walk, per deviation 2 and A2. The loop condition is
     `!fsx.AtWalkCeiling(d)`. Each step takes the directory's `.agents/skills`, except home's, then ends after a
     directory holding `.git`, or at the root.
   * Each doc comment names 0053-MADR D1.
   * The existing `TestTrustFiles`, `TestProjectSkillDirsOutsideRepository` and the rest of `prompt`'s tests pass on
     the one walk.
4. **Tests.** Each sets the ceiling through a `setCeiling` helper that restores the previous one with `t.Cleanup`.
   The helper is in `internal/fsx/ceiling_test.go` and `prompt/ceiling_test.go`. None of the tests is parallel.
   * **`TestWalkCeiling`:**
     * two directories and an empty value are set;
     * `<a>/x/..` and a trailing separator match;
     * a child, the parent and a missing path do not;
     * a file and a missing path are refused, and the ceiling is unchanged after each;
     * clearing it gives nil and no match.
   * **`TestGitRepoStopsAtCeiling`:**
     * a repository above the ceiling is not found from below, by `GitRepo` and `GitBranch`;
     * one at the ceiling is not found from it;
     * with the ceiling at the root's parent, the repository is found.
   * **`TestRepoTopStopsAtCeiling`:** a `.git` file at the ceiling is not read. With the ceiling above it, it is the
     top.
   * **`TestContextFilesStopAtCeiling`:** files above, at and below the ceiling. Only the cwd's file loads with the
     ceiling. With the ceiling above them, all three load, root-most first.
   * **`TestProjectSkillDirsStopAtCeiling`:** `.agents/skills` above, at and below the ceiling, and a `.git` at the
     top. Only the cwd's parent's skills are listed with the ceiling. With the ceiling above them, all three are
     listed, ending at the `.git` directory's own.
   * All five ran and passed (`go test -run Ceiling -v`).
5. **`.golangci.yml`:** `forbidigo` gained `^fsx\.SetWalkCeiling$` with pkg
   `^github\.com/maccavelli/gobble-cli/internal/fsx$`. The message is "only a test binary sets the walk ceiling
   (0053-MADR D1)". `_test.go` files are exempt through the existing exclusion.
6. **Breaks** (C4, `p2_0053_breaks.py`, each on its own scratch copy). All 8 cases were as expected:

   | Break | Failed | First failure |
   | :--- | :--- | :--- |
   | control | none | |
   | `gitDirs`' check removed | `TestGitRepoStopsAtCeiling` | `ceiling_test.go:78: above the ceiling: found "<tmp>…"` |
   | `repoTop`'s check removed | `TestRepoTopStopsAtCeiling` | `ceiling_test.go:106: with the ceiling: repoTop = "<tmp>…", want none` |
   | `ContextFiles`' check removed | `TestContextFilesStopAtCeiling` | `ceiling_test.go:39: with the ceiling: [<tmp>… <tmp>… <tmp>…], want the cwd's file only` |
   | `ProjectSkillDirs`' check removed | `TestProjectSkillDirsStopAtCeiling` | `ceiling_test.go:69: with the ceiling: [three paths], want [one]` |
   | `os.SameFile` replaced by the path string | `TestWalkCeiling` | `ceiling_test.go:41: AtWalkCeiling("<tmp>…") = false, want true` |
   | a file accepted as a ceiling | `TestWalkCeiling` | `ceiling_test.go:51: SetWalkCeiling("<tmp>…") accepted` |
   | a non-test caller in `internal/config` | `golangci-lint` exit 1 | ``zz_ceiling_break.go:6:33: use of `fsx.SetWalkCeiling` forbidden because "only a test binary sets the walk ceiling (0053-MADR D1)" (forbidigo)`` |

   * The `ProjectSkillDirs` break was first written as `; true; {`. That left `trust.go`'s `fsx` import unused, so
     it failed to compile rather than fail the test. That was a fault in the break, not in the code. It was rewritten
     as `; !fsx.AtWalkCeiling(d) || true; {` and run again, and the case was as expected.
7. **`make check-test-isolation`:** exit 1, with the same 30 tests, a sorted list identical to P1's, and the same two
   files. No `TestMain` sets the ceiling yet, as predicted.
8. **Gates** (C5):
   * `make pre-add-check` on the seven changed Go files: "7 file(s) clean", with `govulncheck` skipped there and run
     by preflight.
   * `make preflight`: "preflight passed".
   * `make race`: exit 0.
   * The Linux machine: `go test ./...` 38 `ok`, `acpext` `ok`, `go test -race ./...` exit 0.
   * The Windows laptop, with `TMP` and `TEMP` at `C:\gtmp` as before, since its home folder's `AGENTS.md` still
     leaks until P3: the same results.
   * `make check-records markdownlint` and the identifier scan are below, before staging.
9. **Not changed: two pre-existing workarounds in `prompt/trust_test.go`.** `TestProjectSkillDirsOutsideRepository`
   (`:40-67`) has two:
   * it skips when the temp directory is inside a repository;
   * it filters out an ancestor's skills ("a real ancestor of root may hold its own").

   Both work around the host, which is the exposure this record closes. Once P3 sets the ceiling, neither should be
   needed. They are outside P2's scope and were left, to be raised with the owner.

### P1 record (2026-11-11)

Scripts and logs are in the session scratchpad (`p1_0053_*.log`, `p1_0053_isolation_*.txt`, `p1_0053_autocompact.py`,
`p1_*_fails*.txt`).

1. **C1.**
   * `HEAD` was `4fc6ef4`, P0's records, committed by the owner.
   * `origin/main` was `86ad4a7`.
   * The tree held only the owner's untracked 0051 and 0052 records.
   * The Go code was `86ad4a7`'s throughout this phase.
2. **`scripts/check_test_isolation.py`,** as step 2 says, amended by deviation 1.
   * **The temp roots:** two siblings below the plants, `t/…` for `TMPDIR`, `TMP` and `TEMP`, and `g/…` for `GOTMPDIR`.
     Each is 131 characters on all three hosts.
   * **The run:** `go test -json`, so the failing top-level tests are read per package from the events, and a
     package that fails with no failing test is reported too.
   * **The logs** go to a sibling directory of the root, so they are not counted as strays. The root and the logs are
     removed unless `--keep` is given.
   * Two defects in the script were found and fixed before its first full report:
     * **Its logs counted as strays.** The first run wrote its `go test -json` logs under its own root and reported
       them as files written above the temp root. The logs moved beside the root.
     * **The planted `AGENTS.md` was too small.** At one line, `acpserver`'s `TestAutoCompactSystemPrompt`, one of
       the issue's five, did not fail. A probe (`p1_0053_autocompact.py`, `-run ^TestAutoCompactSystemPrompt$`) gave:
       a 2-byte file passes; a 4,001-byte file fails under a 136-character `TMPDIR` and passes under a
       120-character one; a 40,010-byte file fails under either. The test sees an ancestor file only when it moves
       the system prompt's token count past its scripted compaction. So the plant is about 40 KB (39,201 bytes).
3. **`Makefile`:** `check-test-isolation` runs `$(PYTHON) scripts/check_test_isolation.py`, beside `check-libs`. It
   is in `.PHONY` and not in `preflight`.
4. **Seen failing at the base,** on the three hosts. The macOS report:

   ```text
   check-test-isolation: planted .agents/skills/anc/SKILL.md, .git/HEAD, .gitignore, AGENTS.md above two temp roots: TMPDIR, TMP and TEMP of 131 characters, GOTMPDIR of 131
   == root module: go test exit 1; 30 failing in 6 package(s)
   == acpext module: go test exit 0; 0 failing in 0 package(s)
   == files written above the temp roots: 2
      .gobble/settings.yaml
      .gobble/settings.yaml.lock
   check-test-isolation: FAILED
   ```

   * **The 30 tests** are exactly the MADR's F3 27, under More Information, and F5's 3:
     * `acpserver` 4;
     * `internal/cli` 15;
     * `internal/config` 2;
     * `internal/fsx` 3;
     * `prompt` 4;
     * `tool/builtin` 2.
   * **The other hosts:** the Linux machine and the Windows laptop reported the same 30 tests, as sorted lists
     identical to macOS's, and the same two files (`.gobble\settings.yaml` and its `.lock` on Windows).
   * **Before deviation 1:** the single-root version had reported the same on macOS and Windows, and 0 and 0 on the
     Linux machine.
5. **Gates** (C5). No Go file changed, so `make pre-add-check` has nothing to check. The hosts' `go test ./...` gate
   was not run again: the Go code is `86ad4a7`'s, which CI ran on three OSes. Each host instead ran the new check,
   whose `go test ./...` failed only as step 4 predicts. `make preflight` and `make check-records markdownlint` are
   below.

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
