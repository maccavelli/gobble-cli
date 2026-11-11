---
status: accepted
date: 2026-11-11
decision-makers: repository owner
consulted: "gobble-cli#17; this repository at 86ad4a7 (main, 2026-11-11); go-tui-lib v2.12.0's ask package (doc.go, layout.go); 0049-PLAN's P5b-1 record, step 5; four planted-ancestor runs of the whole suite on macOS, 2026-11-11, recorded under More Information"
informed: "0049-PLAN (its host gates' scratch-copy workaround); 0046-MADR (the network guard this mirrors for the filesystem)"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Stop every upward walk at the test binary's temp root, and check the trust menu against its own layout, so no host file changes a test

## Context and Problem Statement

[gobble-cli#17](https://github.com/maccavelli/gobble-cli/issues/17) reports five tests in `prompt` and `acpserver`
that read a real `AGENTS.md` sitting above their temporary directory. On a host whose temp root is under a home
directory holding one, they fail at plain `HEAD`. The issue was found while gating 0049-PLAN P5b-1 on the Windows
laptop. The gate had to run its scratch copy on the system drive to get a clean result, and every later host gate
inherits that workaround.

The owner asked for the fix at full scope on 2026-11-11. That covers the context-file walk, the project-skill walk
next to it, and a second fragility the reproduction found in the trust-menu tests. Checking for more of the same kind
then found a third and wider exposure: an ancestor `.git`. That one makes tests **write** outside their temp root.
Q1 asks whether it is in scope.

### What holds at `86ad4a7` (read 2026-11-11)

* **F1. Five upward walks read the filesystem, and none stops short of the root.**

  | Walk | Where | Reads in each directory | Stops at |
  | :--- | :--- | :--- | :--- |
  | `prompt.ContextFiles` | `prompt/context.go:77-88` | the five context-file names (`:28`) | the filesystem root |
  | `prompt.ProjectSkillDirs`, first loop | `prompt/trust.go:18-28` | `.git` | the first `.git`, or the root |
  | `prompt.ProjectSkillDirs`, second loop | `prompt/trust.go:29-38` | `.agents/skills`, skipping home's | the first loop's `.git`, or the root |
  | `fsx.GitRepo` / `gitDirs` | `internal/fsx/git.go:15-52` | `.git`, a directory or a worktree's file | the first `.git`, or the root |
  | `fsx.repoTop` | `internal/fsx/ignore.go:132-144` | `.git` | the first `.git`, or the root |

  * `prompt.shadowedFile` (`prompt/context.go:105`), `config.ProjectFile` (`internal/config/resolve.go:59`) and the
    line view's header (`internal/cli/line.go:301-304`) reach `fsx.GitRepo`.
  * The confined walker reaches `repoTop` (`internal/fsx/walk.go:53`) for the repository's ignore rules.
  * `permission.TrustStore.Lookup` (`permission/trust.go:158`) also walks up, but it compares path strings with the
    store's keys and reads no file on the way. It is not exposed.
  * Walking to the root is the intended behaviour for users (Pi's `loadProjectContextFiles`, `findGitRoot`), and
    nothing here changes it.
* **F2. Every test puts its directories in `t.TempDir()`,** under `os.TempDir()` (`TMPDIR`, or `TMP`/`TEMP` on
  Windows). *(Amended by A1: under `GOTMPDIR` when that is set.)* Nothing bounds a walk at that root. So whatever sits in an ancestor of the temp root is read as if it
  were the test's own.
  * `acpserver`'s `TestMain` (`acpserver/testmain_test.go:11`) serves the MCP fixture.
  * `internal/cli`'s `TestMain` (`internal/cli/testmain_test.go`) guards the network and the sign-ins (0046-MADR D4;
    0025-PLAN deviation 9).
  * Neither one covers the filesystem. `prompt`, `internal/config`, `internal/fsx` and `tool/builtin` have no
    `TestMain`.
* **F3. What a planted ancestor breaks.** Each run put files in the parent of `TMPDIR` and ran
  `go test -count=1 ./...` (More Information has the method).
  * **The control,** with nothing planted, fails three tests (F5).
  * **Beyond the control:**

    | Planted above the temp root | Tests failing | Packages |
    | :--- | ---: | :--- |
    | `AGENTS.md` | 5 | `prompt` 4, `acpserver` 1: the issue's list, complete |
    | `.agents/skills/anc/SKILL.md` | 5 | `acpserver` 2, `internal/cli` 3 |
    | `.git/HEAD` | 16 | `internal/cli` 12, `internal/config` 2, `internal/fsx` 2 |
    | `.gitignore` containing `*`, with no `.git` | 0 | none |
    | `.git/HEAD` and `.gitignore` containing `*` | 20 | the 16, and `acpserver` 1, `internal/fsx` 1, `tool/builtin` 2 |

  * **The union:** 27 tests in six packages. With the control's three, 30 tests depend on the host.
* **F4. With a `.git` above the temp root, tests write outside it.** Both `.git` runs left `.gobble/settings.yaml` and
  `.gobble/settings.yaml.lock` in the planted ancestor.
  * The cause: `config.ProjectFile` takes the ancestor's repository root as the project root.
  * For example, `TestConfigSetProject` printed `created <ancestor>/.gobble/settings.yaml`, not the path under its
    own temp directory.
  * The exposed host is a developer whose home directory is a git repository (a dotfiles checkout) and whose temp
    root is under it, as Windows's `%LOCALAPPDATA%\Temp` is. On that host the suite writes into the home folder's
    `.gobble`.
* **F5. Three trust-menu tests assume the project's path fits on one 120-column row.**
  * The tests: `TestTrustQuestion` (`internal/cli/trust_test.go:238`), `TestTrustQuestionNoParent` (`:299`) and
    `TestLineViewTrust` (`:325`).
  * **Why they depend on the path:** they draw the trust question at `promptWidth` (`internal/cli/permission.go:171`).
    That is `editor.MaxWidth`, 120 (`internal/cli/editor/editor.go:82`), with no terminal, and the cap with one
    (`internal/cli/output.go:108-113`). They then search the screen's rows (`shownOn`) for whole strings that contain
    the path, such as `[2] Trust the parent folder (<parent>)`.
  * **Why the layout is right:** go-tui-lib's `ask.Layout` breaks a word wider than its row at the width "so no row is
    wider and every word is shown" (go-tui-lib v2.12.0 `ask/doc.go:5-11`). So a parent path over 88 characters puts
    the label on two rows. That is correct, and the test's substring check fails.
  * **When it trips:** every default temp root seen so far is short enough. A long one trips it, such as the scratch
    roots of a host gate.
  * **The control run shows it:** all three fail under a 130-character `TMPDIR` with nothing planted.
* **F6. `ask.Layout` is exported and pure** (go-tui-lib v2.12.0 `ask/layout.go:24`). Given a question, a width, a
  width method and a glyph set, it returns the rows. go-tui-lib's own tests pin its wrapping rules. `internal/cli`
  already imports `ask` (`internal/cli/permission.go:187`).
* **F7. A rule that only tests may call something has a precedent.**
  * `golangci-lint`'s `forbidigo` keeps the `encoding/json/v2` calls out of non-test code
    (`.golangci.yml:63-74`).
  * forbidigo does not run on `_test.go` files (`.golangci.yml:147-149`).
  * `internal/cli`'s `TestNoBrowserOrSignIn` fails if `TestMain`'s replacements are not in place. That is the
    precedent for a test that a `TestMain` did its part.

## Decision Drivers

* No file outside a test's temp root changes a test's result, on any host, and no test writes outside it.
* The walks' behaviour for users is unchanged, and no production path can turn the bound on.
* The bound covers the tests written later, not just the 30 found today.
* An assertion keeps its exactness: no test is loosened, skipped or given a wider screen than gobble draws.
* No exported API grows for a test's sake. The root module's public packages, `prompt` among them, keep their
  signatures.
* A regression is caught by a check that has been seen to fail, not by the next host that happens to have the file.

## Considered Options

* **A.** One walk ceiling in `internal/fsx`. Every upward filesystem walk honours it, and each test binary's `TestMain`
  sets it to the temp root. The trust tests compare the screen with `ask.Layout`, and an isolation check plants every
  known file above a long temp root.
* **B.** The issue's (a) as written: an unexported stop in `prompt`, set per test by a helper, with `acpserver` reaching
  it through an exported `prompt` option.
* **C.** The issue's (b): each test binary points `TMPDIR`/`TMP`/`TEMP` at a fresh root, hoping nothing is above it.
* **D.** Filter in the tests: `acpserver` and `internal/cli` wrap `prompt.ContextFiles` and `ProjectSkillDirs` in
  package-level seams that drop results outside the temp root, and the walks stay unbounded.
* **E.** Change the walks for everyone: stop at the home directory.

## Decision Outcome

Chosen option: "A", accepted 2026-11-11 with Q1 to Q4 each (a), because it is the only option that bounds every exposed walk (F1, F3), including the one
that writes (F4), in one place, for current and future tests, without a public API change or a production behaviour
change. It also replaces the trust tests' host-dependent substring checks with an exact comparison against the layout
the library defines (F5, F6).

### D1. `internal/fsx` holds one walk ceiling; every upward walk stops before reading it

*Amended by A1: the ceiling is a set of directories.*

* **The hook:** `internal/fsx` gains an unexported ceiling, set by `SetWalkCeiling(dir string) error`, cleared by
  `""`, and read through `AtWalkCeiling(dir string) bool` and `WalkCeiling() string`.
* **How a directory is matched:** `SetWalkCeiling` stats `dir` once. `AtWalkCeiling` is true when a ceiling is set and
  `os.SameFile` matches the directory's stat against it. So a symlinked spelling (`/var` against `/private/var`), a
  case-folded one, or a Windows short name all match. When no ceiling is set, it costs nothing.
* **Where it is checked:** each walk in F1 checks it before reading a directory, and stops there as it would at the
  root:
  * `prompt.ContextFiles`;
  * both loops of `prompt.ProjectSkillDirs`;
  * `fsx.gitDirs`;
  * `fsx.repoTop`.
* **The ceiling is never read.** The ceiling directory and every directory above it go unread.
* **No production path sets it.** A `forbidigo` rule forbids `fsx.SetWalkCeiling` outside `_test.go` files (F7), so
  the rule is a lint failure in `make preflight`. The setter is exported only from an `internal` package, so the
  module's public API is unchanged.
* **Under Q1 (b)** the hook is the same, and only the two `prompt` walks check it.

### D2. Each exposed package's `TestMain` sets the ceiling to the temp root

*Amended by A1: to `GOTMPDIR` when it is set, and to `os.TempDir()`.*

* **The value:** the ceiling is `os.TempDir()`, set before `m.Run()`. That is the narrowest directory that holds every
  `t.TempDir()`, including the worktree test's sibling (`prompt/context_test.go:65`).
* **A failed set** exits 2 with the reason, as `internal/cli`'s `TestMain` does for an unset variable.
* **The packages** are the six that F3 found:
  * `acpserver` and `internal/cli`, whose `TestMain`s gain the line (`internal/cli`'s in the child that runs the
    tests);
  * `prompt`, `internal/config`, `internal/fsx` and `tool/builtin`, each gaining a `testmain_test.go`.
* **A check that `TestMain` did it:** each of the six gains `TestWalksBounded`, which fails unless `fsx.WalkCeiling()`
  is the temp root, after `TestNoBrowserOrSignIn`'s pattern.
* **Under Q2 (b)** every package with tests gets the `TestMain` instead. Under Q2 (c) no `TestMain` is used, and each
  exposed test calls a helper.

### D3. The trust-menu tests compare the screen with the question's own layout

* **What replaces the row checks:** in the three tests of F5, the substring checks on the screen give way to two
  checks.
* **First check, the question's content:** the expected question is written in the test as literal strings and the
  project's paths: title, notes, labels and keys. It is not built through `permissionQuestion`, so the test is not
  circular.
* **Second check, the drawing:** `ask.Layout(expected, width, …)` gives the rows a user sees at the width the question
  was drawn at. The screen must show exactly those rows, in order, below what came before.
* **The width** is `promptWidth`'s 120, unchanged. No test widens the screen.
* **What it catches:** a path long enough to wrap is then drawn and checked as it wraps. A dropped label, a wrong path
  or a different layout still fails.
* **The `untrustedLine` count and the saved-trust checks** are kept as they are.
* **Not changed:** `shownWrapped` (`internal/cli/view_wrap_test.go:78`) serves the line view's replies, which wrap at
  spaces, and stays.

### D4. An isolation check plants every known file above a long temp root

*Amended by A1: two sibling temp roots, one for `TMPDIR`, `TMP` and `TEMP`, one for `GOTMPDIR`.*

* **The script:** `scripts/check_test_isolation.py` creates its own root under the system's temp directory and plants
  in it:
  * `AGENTS.md`;
  * `.agents/skills/anc/SKILL.md`;
  * `.git/HEAD`;
  * `.gitignore` containing `*`.
* **The temp root it gives the tests** is a directory below that one, deep enough that its path is 130 characters or
  more. That is longer than F5's 88-character limit, short of Windows's 260. It sets `TMPDIR`, `TMP` and `TEMP` to it.
* **What it runs:** `go test -count=1 ./...`, and `acpext`'s tests from `acpext/`.
* **It fails** on any test failure, or on any file under its root that it did not plant and that is not under the
  temp root (F4).
* **It runs as** `make check-test-isolation`. Where it runs is Q4.
* **It is its own seen-failing proof:** at `86ad4a7` it must report F3's 27 and F5's 3, and F4's two files.

### D5. Questions for the owner

Each question has three options, the recommended one first.

* **Q1. Which walks are bounded?**
  * **(a) All five in F1, recommended.** F3 and F4 show the `.git` walks are the widest exposure (20 tests) and the
    only one that writes outside the temp root.
  * **(b) The two `prompt` walks only,** as asked on 2026-11-11 before the `.git` finding. The 16 to 20 `.git` tests
    and the writes are left for another record.
  * **(c) The two `prompt` walks and `fsx.gitDirs`, but not `repoTop`.** This stops the writes, but leaves the
    ignore-rule tests (`tool/builtin`, `internal/fsx` `TestWalkWithoutRepository`, `acpserver` `TestLocations`)
    exposed to an ancestor `.gitignore`.
* **Q2. Where is the ceiling set?**
  * **(a) In the `TestMain` of the six packages F3 found,** with `TestWalksBounded` in each, recommended. D4's check
    finds any other package that turns out to be exposed.
  * **(b) In a `TestMain` for every package with tests,** about forty, through a shared helper. This is the widest
    cover, but it adds a `TestMain` to packages that walk nothing.
  * **(c) Per test, through a helper,** as the issue proposes. Each new test that walks must remember it.
* **Q3. How do the trust tests check the menu?**
  * **(a) Against `ask.Layout`, D3, recommended.** The check is exact, and it holds for a path of any length.
  * **(b) Against a wrap-aware matcher,** a `shownWrapped` that also joins rows broken mid-word. It is simpler, but it
    accepts any break position, so it is a weaker check.
  * **(c) Unchanged,** with the host gates required to use a short temp root. This records the fragility rather than
    removing it.
* **Q4. Where does D4's check run?**
  * **(a) In each host gate, and as one CI job on Linux, recommended.** The job adds one more full test run, about the
    length of the Linux test job.
  * **(b) In each host gate only,** through `make check-test-isolation`, not in CI.
  * **(c) Nowhere.** D1's per-walk tests and D2's `TestWalksBounded` are the only guards.

#### Answers (2026-11-11)

The owner answered all four questions, each with (a):

* **Q1 (a).** All five walks in F1 are bounded, `fsx.gitDirs` and `fsx.repoTop` included.
* **Q2 (a).** The ceiling is set in the `TestMain` of the six packages F3 found, with `TestWalksBounded` in each.
* **Q3 (a).** The trust tests compare the screen with `ask.Layout`'s rows (D3).
* **Q4 (a).** `make check-test-isolation` runs in each host gate and as one CI job on Linux, added last.

The decision is accepted as option A, with D1 to D4 as written. The alternatives under (b) and (c) in D1 to D3 do not
apply.

### Amendment A1 (2026-11-11): the test binary has two temp roots

This amendment comes from
[0053-PLAN-keep-tests-off-host-files-above-their-temp-root.md](0053-PLAN-keep-tests-off-host-files-above-their-temp-root.md)
deviation 1. The owner chose "a ceiling at both roots".

* **The finding.** P1's isolation check failed the same 30 tests, and found the same two stray files, on macOS and the
  Windows laptop. On the Linux machine it reported none.
  * That host exports `GOTMPDIR`.
  * Go's `t.TempDir()` is `os.MkdirTemp(os.Getenv("GOTMPDIR"), …)` (Go 1.27 `src/testing/testing.go:1575-1615`). So
    it is under `GOTMPDIR` when that is set, and under `os.TempDir()` only when it is not.
  * A probe on a scratch copy there, with `TMPDIR` at a planted root, gave `t.TempDir()` under the host's `GOTMPDIR`.
* **F2 as amended.** A test's `t.TempDir()` is under `GOTMPDIR` when it is set, and otherwise under `os.TempDir()`.
  A directory made by `os.MkdirTemp("", …)` or under `os.TempDir()` is under `os.TempDir()` either way. Today those
  are the two binary tests' build directories (`acpserver/binary_test.go:25`, `internal/cli/complete/binary_test.go:17`).
  F2's sentence "under `os.TempDir()` (`TMPDIR`, or `TMP`/`TEMP` on Windows)" holds only where `GOTMPDIR` is unset.
* **D1 as amended.**
  * `SetWalkCeiling(dirs ...string) error` takes each non-empty directory, stat'ed once. With none given, the ceiling
    is cleared.
  * `AtWalkCeiling(dir)` is true when `os.SameFile` matches any of them.
  * `WalkCeiling() []string` returns the set paths.
  * Everything else in D1 holds.
* **D2 as amended.**
  * Each `TestMain` calls `fsx.SetWalkCeiling(os.Getenv("GOTMPDIR"), os.TempDir())`, and `SetWalkCeiling` skips an
    empty value.
  * `TestWalksBounded` fails unless the ceiling holds `os.TempDir()`, and `GOTMPDIR` too when it is set.
  * A ceiling at `os.TempDir()` alone would match nothing that `t.TempDir()` makes on a host like the Linux machine,
    and D2's check as first written would still have passed.
* **D4 as amended.**
  * The isolation check makes two sibling temp roots below its planted files, each of 130 to 160 characters. It sets
    `TMPDIR`, `TMP` and `TEMP` to the first and `GOTMPDIR` to the second, overriding the host's own `GOTMPDIR`.
  * Files under either root are the tests' own. Every other file not planted is a stray.
  * So the check sees every host the same way, and it covers both roots of the ceiling.
* **Confirmation as amended.**
  * Item 3 adds a break in which `TestMain` passes `os.TempDir()` alone, run with `GOTMPDIR` set. It must fail
    `TestWalksBounded`.
  * Item 5's 30 tests and two files must be reported on the Linux machine too.
* **Out of scope, observed.** A gobble binary that a test starts as a child process runs production code, so it has
  no ceiling. None of the binary tests failed under the plants. D4's check is what would show one that becomes
  exposed.

### Consequences

* Good, because no planted file above the temp root changes a result, and no test writes above it (F3, F4). The host
  gates no longer need a scratch copy outside the home folder.
* Good, because the bound is one hook, and a test package gets it with one `TestMain` line, so later tests are
  covered without each remembering it.
* Good, because users' walks are unchanged, and lint stops a production call to the setter.
* Good, because the trust tests check exactly what is drawn, for any path length, and gain an exact check of the
  question's content.
* Neutral, because each walk makes one more `os.Stat` per directory, only while a ceiling is set: in tests, never for
  users.
* Bad, because four packages gain a `TestMain`. A package added later that walks needs one too, and only D4's check
  says so.
* Bad, because the trust tests now depend on `ask.Layout`'s exported signature. A go-tui-lib change to it is a
  compile error in those tests, not a silent pass.

### Confirmation

1. **Each walk's own test.** Each walk in F1 that D1 bounds gets a test that plants its file above a ceiling, both
   inside `t.TempDir()`:
   * with the ceiling set, a walk from below excludes the file;
   * with it cleared, the same walk finds it;
   * each is seen failing on a scratch copy with that walk's ceiling check removed.
2. **The lint rule.** `forbidigo` fails on a scratch non-test file calling `fsx.SetWalkCeiling`, and passes on the
   tree.
3. **`TestWalksBounded`** fails in each of the six packages on a scratch copy whose `TestMain` omits the call.
4. **The trust tests** pass with the default temp root and under D4's long one. They are seen failing on a scratch
   copy with one expected label changed, and with the expected layout taken at 80 columns.
5. **D4's check** reports at least F3's 27 tests, F5's 3 and F4's two files when run against `86ad4a7`. After the fix
   it reports none, on macOS, the Linux machine and the Windows laptop.
6. **The gates pass:** `make preflight`, `make race`, and `go test ./...` on the three hosts. On the Windows laptop
   that includes a clone under the home folder, the case 0049-PLAN P5b-1 had to avoid.

## Pros and Cons of the Options

### A. One ceiling in `internal/fsx`, set by each test binary, an exact trust layout, an isolation check

* Good, because it covers all five walks, including the one that writes (F4).
* Good, because the `TestMain` placement covers later tests, as 0046-MADR D4's network guard does for the network.
* Good, because the public API is unchanged: the setter is in an `internal` package, and lint keeps it out of
  production.
* Neutral, because it adds about six `TestMain` lines, four new `testmain_test.go` files and a script.
* Bad, because a newly exposed package is found only by D4's check, not by the compiler.

### B. The issue's (a): a stop in `prompt`, per test, `acpserver` through an exported option

* Good, because it is the smallest change for the five tests the issue lists.
* Bad, because `acpserver`'s test binary cannot set another package's unexported variable. The issue's fallback is
  an option on `prompt.Options`, but that struct belongs to `Build`, not to `ContextFiles(configDir, cwd)`. Either
  way it means a public signature change for a test.
* Bad, because it leaves the skills walk and the `.git` walks unbounded: 22 of F3's 27 tests, and F4's writes.
* Bad, because per-test helpers miss the next test written.

### C. The issue's (b): a fresh temp root per binary

* Good, because it needs no code change in the walks.
* Bad, because it only moves the problem. Any root under the profile still has the profile's files above it, and on
  Windows every default temp root is under it.
* Bad, because the fresh root itself has to sit somewhere: under the system's temp directory, whose ancestors C cannot
  choose and does not check.

### D. Filter the results in the tests

* Good, because `prompt`'s API and code are untouched.
* Bad, because it filters only what reaches the seams: `fsx.GitRepo` inside `config.ProjectFile` still finds the
  ancestor's repository, and still writes there (F4).
* Bad, because a filtered list can hide a wrong walk, for example one that reads the ancestor and then drops it.

### E. Stop the walks at the home directory for everyone

* Good, because it also bounds the tests on every host whose temp root is under the home folder.
* Bad, because it changes what users get. gobble's walk goes to the root by design (`prompt/context.go:59-63`, after
  Pi's `loadProjectContextFiles`), so a context file above the home folder, such as one in an organisation's shared
  folder, would no longer load.
* Bad, because it does nothing for a temp root outside the home folder that has an ancestor `.git`.

## More Information

### The method (2026-11-11, macOS, the tree at `86ad4a7` plus the untracked 0051 and 0052 records)

* **Where it ran:** each run used a fresh directory in the session's scratch area. The planted files went in that
  directory, and `TMPDIR` was a child of it (about 130 characters).
* **The command:** `GOPROXY=off GOFLAGS=-mod=readonly go test -count=1 ./...`.
* **The runs:**
  * nothing planted (the control);
  * `AGENTS.md`;
  * `.agents/skills/anc/SKILL.md`;
  * `.git/HEAD` with `.gitignore` containing `*`;
  * then, on six packages, `.git/HEAD` alone and `.gitignore` alone.
* **What was compared:** the failing top-level tests, per package, less the control's three. Afterwards, the files
  left in each planted directory were listed.
* **The scripts** (`i17_repro.py`, `i17_plants.py`, `i17_gitsplit.py`, `i17_map.py`) and their logs are in the session
  scratchpad, not committed. D4 makes the method a committed check.
* **The tree** was not written.

### Failing tests by planted file, beyond the control

* **`AGENTS.md`:**
  * `prompt`: `TestConfigFirstAndUnreadable`, `TestGoldenSystemPrompt`, `TestGoldenSystemPromptDry`,
    `TestWorktreeShadowing`;
  * `acpserver`: `TestAutoCompactSystemPrompt`, failing with `llmtest: script exhausted` on its second prompt, as the
    issue says.
* **`.agents/skills`:**
  * `acpserver`: `TestNoticesWhenOffered`, `TestPromptOptionsAndTrust`;
  * `internal/cli`: `TestFirstRunSetup`, `TestPrintModeTrust`, `TestSessionSettingsReachAgent`.
  * Each one sees an extra project skill directory, so an extra notice or trust note. For example, `notices_test.go:124`
    gave `5 notices, want 4`, and `sysprompt_test.go:271` said the trust note was logged 2 times.
* **`.git/HEAD`:**
  * `internal/cli`: `TestConfigEdit`, `TestConfigGetOrigin`, `TestConfigSetGlobalOnlyProjectRefused`,
    `TestConfigSetProject`, `TestConfigUpdateDryRun`, `TestConfigValidateExits`, `TestConfigValidateJSONFiles`,
    `TestFirstRunSetup`, `TestPrintModeTrust`, `TestPromptHeader`, `TestSessionSettingsReachAgent`,
    `TestTrustCommands`;
  * `internal/config`: `TestOriginsAndLines`, `TestRulesJoinAcrossLayers`;
  * `internal/fsx`: `TestGitBranch`, `TestGitRepo`.
  * Two examples: `resolve_test.go:351` gave `ProjectFile outside a repository = "<ancestor>/.gobble/settings.yaml"`,
    and `git_test.go:57` gave `a plain directory found "<ancestor>"`.
* **`.git/HEAD` with `.gitignore` containing `*`:** the sixteen above, and:
  * `acpserver`: `TestLocations`;
  * `internal/fsx`: `TestWalkWithoutRepository`;
  * `tool/builtin`: `TestBoundsHold`, `TestGrepInvalidUTF8File`.
* **`.gitignore` alone:** none. `repoTop` finds no repository, so no ignore file above the temp root is read.
* **The control's three:** `internal/cli`'s `TestTrustQuestion`, `TestTrustQuestionNoParent` and `TestLineViewTrust`,
  failing on rows such as `[2] Trust the parent folder (<parent>)` broken across two rows (F5).

### Not in scope

* Tests whose working directory is the source tree. Two `internal/cli` tests use `os.Getwd()`, and the golden prompt
  tests copy their fixture out precisely to avoid this. They walk the clone itself, which is the same on every host,
  and stop at the clone's own `.git`.
* `permission.TrustStore.Lookup`, which reads no file as it walks (F1).
* Any change to what users' walks find.

### Related records

* [0046-MADR-own-background-listings-and-keep-tests-hermetic.md](0046-MADR-own-background-listings-and-keep-tests-hermetic.md)
  D4 keeps the test binaries off the network from `TestMain`. This record does the same for the filesystem above the
  temp root.
* [0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md)
  records the finding, in P5b-1's record, step 5.
* The plan is
  [0053-PLAN-keep-tests-off-host-files-above-their-temp-root.md](0053-PLAN-keep-tests-off-host-files-above-their-temp-root.md).
