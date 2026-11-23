---
status: complete
date: 2026-11-22
associated-madr: "0067-MADR-windows-sharing-violation-when-reading-auth-json.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement Retrying a Config File's Read and auth.json's Replace on Windows

Associated MADR: [0067-MADR-windows-sharing-violation-when-reading-auth-json.md](0067-MADR-windows-sharing-violation-when-reading-auth-json.md)

*(2026-11-22, second draft, after independent review 1: tests and plants corrected (each plant can turn its named test red), a seam step so every red test compiles at the base, the writer side added, stage-only wording.)*

## Goal

After this PLAN, `fsx.ReadRegularFile` retries its open for a bounded window on a Windows sharing violation, and `auth.write` replaces
`auth.json` and `clients.json` through an exported retrying rename in `fsx`, so `go test -race -count=200 -run '^TestSignInTypeAheadNotSentAsPrompt$' ./internal/cli/`
has no failure on the Windows laptop (7 of 200 at `9a22518`) and a sign-in's save no longer fails under a reader.

The owner ruled on 2026-11-22 that this gates gobble v0.7.0: **this PLAN's build lands before the v0.7.0 tag** (the tag stays the owner's).

The MADR is `accepted` (2026-11-22): the owner answered Q1 "Retry reads and the save (Recommended)" and Q2 "2 seconds (Recommended)", deciding rows O6 and O7. This PLAN is written for those
answers (reads retried in `fsx.ReadRegularFile`, `auth.write` through `fsx.ReplaceFile`, a 2 s window; no FILE_SHARE_DELETE open, no per-process limit). It stays `proposed`:
nothing below Phase 0 starts before the owner approves it and says `proceed`.
Under the owner's current rule nothing is committed by the agent: each phase ends with its files **staged** and the gates recorded; the owner commits.

## Scope

### In scope

* `internal/fsx/regular.go` (the retry loop and its seams), new `internal/fsx/read_windows.go` and `read_other.go` (the sharing-violation predicate), `internal/fsx/write.go` (an exported retrying rename, `renameWindow` as a variable for tests), `internal/auth/store.go` (`auth.write` calls it).
* New tests in `internal/fsx` and `internal/auth`.
* This pair, its index rows, and the docs step (Phase 2).

### Out of scope

* `hostid.go:98` and `migrate.go:328` (they never replace an existing file; X5).
* The follow-up readers and writers of O1 (`choice.go:152`, `:245`, `:284`, `editor/history.go:31`, `:89`, the workspace path): a separate record.
* Any change to `TestSignInTypeAheadNotSentAsPrompt` or `storedKey` (D5); FILE_SHARE_DELETE opens (option C, not chosen); a per-process total wait (Q2 c, not chosen); a context for the wait (O2); hard-linked files (O3).

## Rules for Every Phase

* **Re-ground first.** Re-read each `path:line` the MADR cites at the phase's start commit and record each moved line.
* **Red, then green.** A test marked red runs at the phase's base and must fail on its assertion, never on a compile error: the seam step comes first so every test compiles.
* **Plants.** Each plant is one production change, made on a scratch copy, never in the tree; each must turn the test it names red within that test's own deadline (the `bounded` helper, `regular_test.go:28`); the record quotes each failing line.
* **Heavy runs** (`make preflight`, `make race`, `go test ./...`, any `-race`) go through `~/.agents/skills/available-dev-hosts/scripts/run-on-host` (with `--race` where `-race` is hidden), one at a time per host; a single package's `go test` without `-race` may stay on the session host. The Windows runs use `--need windows`.
* **No live run; no real gobble file is read.** Every test builds its file in `t.TempDir()`.
* **Stop conditions**, each a stop and a prompt: an existing test needs a change no row names; a row cannot be met as written; a third build pass of one phase.
* **Exit criterion of every phase:** no blocker, every row tested, plants caught.

## Implementation Steps

### Phase 0: re-ground (records only)

1. Re-read the MADR's citations at HEAD: `internal/auth/store.go:73`, `:358`, `:367`, `:412`, `:421`; `internal/auth/clients.go:38`, `:87`, `:115`, `:122`; `internal/auth/hostid.go:78`, `:98`, `:107`; `internal/auth/migrate.go:198`, `:323`, `:328`; `internal/fsx/regular.go:34-45`; `internal/fsx/write.go:131-133`, `:190`, `:196`; `internal/fsx/rename_windows.go:11`; `internal/cli/login_test.go:217`, `:856`; `internal/cli/credentials.go:64`, `internal/cli/login.go:149`, `internal/cli/auth.go:370`, `internal/cli/completion.go:144`, `internal/cli/main.go:389`; the 16 call sites of `ReadRegularFile` (`grep -rn 'ReadRegularFile(' --include=*.go .` must still count 16 outside tests).
2. The probes of the MADR (drafting) are recorded there; if HEAD's Go version differs from 1.27.4, run them again on a scratch module through `run-on-host --need windows` and record the difference. They are temp programs, never in the tree.
3. `docs/architecture.md`: grep for how it describes `fsx`'s reads and `auth.json`'s write; note whether Phase 2's docs step needs a line.

Exit: citations re-grounded; nothing but the records changed.

### Phase 1: seams, red tests, the changes, plants and gates

**Step 1a: the seams, behaviour unchanged (staged first).** In `internal/fsx/regular.go`: package variables `openFile` (default `os.OpenFile`), `retriable func(error) bool` (default `sharingViolation`), `readWindow` (default 2 s); `ReadRegularFile` calls `openFile` once. In `internal/fsx/write.go`: `renameWindow` becomes a variable; an exported `ReplaceFile(tmp, name string) error` that calls `rename(osFS{}, tmp, name)` (the name is fixed here); `auth.write` still calls `os.Rename`. `read_windows.go`: `sharingViolation(err) = errors.Is(err, windows.ERROR_SHARING_VIOLATION)`; `read_other.go`: `false`. `make preflight` unchanged.

**Step 1b: red tests (each compiles at 1a's tree; each must fail on its assertion):**

| Row | Test | File | Shape | Red at 1a |
| :--- | :--- | :--- | :--- | :--- |
| R1, R2, R4, R5, R6 | `TestReadRegularFileRetryLoop` (table; portable) | `internal/fsx/regular_retry_test.go` | `openFile`, `retriable` and `readWindow` replaced: (1) a retriable error twice then success returns the content, three calls; (2) a retriable error forever returns the **last** error after the window (called through `bounded`, the window set to 50 ms); (3) a retriable error then `fs.ErrNotExist` returns not-exist at once; (4) a retriable error then another error returns it at once; (5) a first non-retriable error: one call | the retry rows (1, 2 after-window count, 3, 4) fail: one call only |
| R3, R8 | `TestSharingViolationPredicate` (Windows: true for `ERROR_SHARING_VIOLATION`, false for `ERROR_ACCESS_DENIED`, `ERROR_FILE_NOT_FOUND` and a plain error) and `TestSharingViolationNever` (non-Windows: false for each) | `read_windows_test.go`, `read_other_test.go` | each file's own predicate, each platform | pass at 1a (the predicate exists): guards, caught by plant P4 |
| R1, R9, R10 | `TestReadRegularFileSurvivesAHeldHandle` (Windows) | `read_windows_test.go` | a holder of `CreateFile(DELETE, share read, write, delete)` on a temp file; `openFile` is wrapped so its **first refused attempt closes the holder** (no sleep, no goroutine race); the read must return the content | fails at 1a: the single attempt is refused |
| C1 to C3, C8 | `TestAuthReadersSurviveAHeldHandle` (Windows; table over `auth.Load`, `Clients.read` through `Clients.List`, `readHostID`) | `internal/auth/held_windows_test.go` | the same holder, released on the first refusal through an exported seam added in 1a: `fsx.SetOpenFile(f) (restore func())`, documented as a test seam | fails at 1a for each of the three |
| X1, X3 | `TestSetKeyReplaceSurvivesAReader` and `TestClientsSaveReplaceSurvivesAReader` (Windows) | `internal/auth/store_windows_test.go` | hold `auth.json` with `os.Open`, a goroutine closes it after 100 ms; `auth.SetKey` (and `Clients.Save`) must succeed and the file hold the new key | fails at 1a: `auth.json: rename ...: Access is denied.` at once (probe: the writer) |
| X2, X4 | `TestReplaceFileGivesUpAndNotTransient` (portable, through `bounded`) | `internal/fsx/write_test.go` | the filesystem seam of `rename` returns a transient error forever (`renameWindow` 50 ms): `ReplaceFile` returns the last error after the window; a non-transient error returns at once | passes at 1a (`ReplaceFile` is the existing `rename` loop): a guard, not a red test; its worth is shown by plant P5 |
| R9, R10 (statistical) | `-race -count=200` of `TestSignInTypeAheadNotSentAsPrompt` on the Windows laptop | `internal/cli` | the base's 7 of 200 | red at 1a: run once, record the lines |

**Step 1c: the changes (after the red run is recorded):**

1. `ReadRegularFile`: the loop. Retry `openFile` while `retriable(err)` and the deadline has not passed; pauses 1 ms growing to 100 ms; on any other error return it at once; at the deadline return the last error.
2. `auth.write`: `os.Rename(name, path)` becomes `fsx.ReplaceFile(name, path)`; the error wrapping and the temp file's removal stay as at `store.go:421-423`.
3. Nothing in `internal/cli` changes; `hostid.go` and `migrate.go` do not change.

**Plants** (scratch copy, each one production change; each must turn the named test red within its deadline):

| Plant | Production change | Turns red |
| :--- | :--- | :--- |
| P1 | the read loop removed (one open) | `TestReadRegularFileRetryLoop` (the retry rows), `TestReadRegularFileSurvivesAHeldHandle`, `TestAuthReadersSurviveAHeldHandle`, the `-count=200` run |
| P2 | `retriable` made `true` for every error | `TestReadRegularFileRetryLoop` rows (3), (4) and (5) (an error not retried would be retried until the window) |
| P3 | the read deadline removed | `TestReadRegularFileRetryLoop` row (2): fails inside `bounded`'s deadline, not at go test's 10 minutes |
| P4 | `sharingViolation` returns `true` on non-Windows, `false` on Windows | `TestSharingViolationNever` on a Linux host, `TestSharingViolationPredicate` on the laptop |
| P5 | `ReplaceFile`'s retry removed (`os.Rename` only) | `TestReplaceFileGivesUpAndNotTransient`, `TestSetKeyReplaceSurvivesAReader` |
| P6 | `auth.write` back to `os.Rename` | `TestSetKeyReplaceSurvivesAReader`, `TestClientsSaveReplaceSurvivesAReader` |
| P7 | `auth.Load` reads through `os.ReadFile` instead of `fsx.ReadRegularFile` | `TestAuthReadersSurviveAHeldHandle` (the `auth.Load` row) |

**Gates** (one at a time per host, through `run-on-host`):

* `go test -race -count=200 -run '^TestSignInTypeAheadNotSentAsPrompt$' ./internal/cli/` on the Windows laptop (`--need windows --race`): 0 failures, run twice at different times (the chance a 3.5 % flake passes 200 runs is under 0.1 %).
* `go test -race -count=20 ./internal/auth/ ./internal/fsx/` on the Windows laptop.
* `go test ./internal/fsx/ ./internal/auth/` on a Linux host and on macOS.
* `make preflight` on the three hosts, then `make check-records markdownlint`.

**Acceptance (Phase 1):** every decided row R1 to R11, C1 to C8, X1 to X6 has its test (or is named a row with no change: X5, X6, C3's existing tests) and its plant caught; the laptop gates pass; no existing test changed; files staged, not committed.

### Phase 2: close-out

1. Confirmation 1 to 4 checked and recorded; the MADR is `accepted` (2026-11-22); after the gates and the owner's approval this PLAN is `complete`.
2. The follow-up of O1 is written into the index as a named open item (an owner decision whether to open its record).
3. The v0.7.0 tag is told that this record's build is done (the owner's tag decision; this PLAN does not tag).
4. A docs step: a line in `docs/architecture.md` if Phase 0 step 3 found one needed; otherwise "none needed" recorded with the grep.

## Verification

The commands above, each with its exit status captured before any filter, output in the execution record: the red run (7 of 200 expected), the green runs (0 of 200, twice), the seven plants, the suites on three hosts, `make preflight` on three hosts, `scripts/check_records.py --check`.

## Rollout and Rollback

No rollout: a library change with no flag. Rollback is reverting the staged files or the owner's commit; the flake then returns at 3.5 % of `-race` runs on Windows as at `9a22518`.

## Execution Record

### Drafting (2026-11-22)

Records only; second draft after independent review 1. The reproduction (200 runs without `-race`: 0 failures; with `-race`: 7,
all `login_test.go:856: auth.json: open ...\auth.json: The process cannot access the file because it is being used by another process.`)
and the probes (i), (ii), (iii) and the writer's are in the MADR. The probes were temp programs in a scratch module; no file
of the repository was changed; no phase has started.

### Approval (2026-11-22)

The owner approved this PLAN, relayed by the coordinator: "Approve; build after the commit (Recommended)". The records
commit is `b6eb169`.

### Phase 0, pass 1 (2026-11-22, base `b6eb169`)

1. Citations: `git diff --stat 9a22518 b6eb169` changes only `docs/README.md` and this pair, so no cited source line can
   have moved. Each `path:line` of step 1 was printed at `b6eb169` and reads as the MADR says, including
   `store.go:28` (the 30 s lock deadline), `:79`, `:421-423` and the follow-up lines of row O1. `write.go:131-133` is the
   start of the hard-link branch; the branch itself is `:132-139`. `grep -rn 'ReadRegularFile(' --include=*.go .`,
   tests and the definition excluded, counts **16**, the MADR's list.
2. Go: `go.mod` says `go 1.27.4`, the probes' version; the probes were not run again.
3. `docs/architecture.md`: the `internal/fsx` row (line 150) says "atomic writes" and the `internal/auth` row (line 144)
   says `auth.json` is written owner-only and locked; neither describes a read or a rename retry, so the change makes no
   line false. Phase 2 step 4 records "none needed" unless the owner asks for a line.

Exit: met; only this file changed.

### Phase 1, pass 1 (2026-11-22): stopped before step 1a, no source changed

Stop condition "a row cannot be met as written" (rows X2, X4; the test `TestReplaceFileGivesUpAndNotTransient`):

* Step 1a fixes `ReplaceFile` as `rename(osFS{}, tmp, name)`, so no test can give `ReplaceFile` a filesystem that
  returns an injected error; a test that calls `rename` with a fake filesystem does not exercise `ReplaceFile`, and
  plant P5 (a change to `ReplaceFile`) cannot turn it red.
* `rename` retries only when `transient(err)` (`internal/fsx/write.go:201`), and off Windows `transient` is `false` for
  every error (`internal/fsx/rename_other.go:6`). A portable test whose fake rename returns "a transient error forever"
  sees one call and an immediate return on Linux and macOS, so its give-up half fails there on the finished tree.

Also found, for the same prompt (none blocks alone): `Clients.List` does not exist (`internal/auth/clients.go:36` has
`Clients.Load`, which calls `read`); the loop test replaces `retriable`, so plant P2 can only be the loop's use of the
predicate, not the variable's default; and P1's `-count=200` entry is the base's red run (the same reader). Waiting for
the owner's resolution.

### Deviation 1 (2026-11-22): the writer's give-up test is split

Found: Phase 1, pass 1 above. Decision: the owner ruled, relayed by the coordinator, "Split it; Windows give-up
(Recommended)". The row of step 1b for X2 and X4 is replaced:

* ~~`TestReplaceFileGivesUpAndNotTransient` (portable, through `bounded`), `internal/fsx/write_test.go`: the filesystem
  seam of `rename` returns a transient error forever~~
* **X2:** `TestReplaceFileGivesUp` (Windows), new `internal/fsx/write_windows_test.go`: the target is held with `os.Open`
  for the whole call, `renameWindow` is 50 ms, the call runs through `bounded`; `ReplaceFile` must return `Access is
  denied` after at least the window. A guard at 1a (`ReplaceFile` is `rename`'s loop); its worth is shown by P5.
* **X4:** `TestReplaceFileNotTransient` (portable), `internal/fsx/write_test.go`: a missing temp file returns
  `fs.ErrNotExist` well inside a 5 s `renameWindow`, through `bounded`.
* P5 is caught on the laptop by `TestReplaceFileGivesUp` and `TestSetKeyReplaceSurvivesAReader`.
* Files added to the phase: `internal/fsx/write_windows_test.go`.

### Notes (2026-11-22, decided by the coordinator from evidence and the owner's standing directions)

1. The auth held-handle table's `clients.json` row reads through `Clients.Load` (which calls `read`); `Clients.List`
   does not exist.
2. Plant P2 is made in the loop's use of the predicate (every error treated as retriable), because
   `TestReadRegularFileRetryLoop` replaces `retriable` itself.
3. Plant P1's `-count=200` entry cites the base red run (7 of 200 under `-race` at `9a22518`; 4 of 200 in this build's
   red run, below; the same reader) instead of running it again (the owner's direction against repeating runs that
   prove nothing new).
4. Phase 2 step 4: one sentence goes into `docs/architecture.md`, where `internal/fsx` and `internal/auth` are
   described, stating the read and replace retries on Windows (docs are kept complete).
5. Staging: nothing is staged by the building agent; the coordinator stages after its own gate and the owner commits.

### Phase 1, pass 2 (2026-11-22, base `b6eb169`, after deviation 1)

**Step 1a, the seams.** `internal/fsx/regular.go`: `openFunc`, the variables `openFile` (`os.OpenFile`), `retriable`
(`sharingViolation`) and `readWindow` (2 s), and the exported test seam `SetOpenFile(open) (restore func())`;
`ReadRegularFile` called `openFile` once. New `read_windows.go` (`errors.Is(err, windows.ERROR_SHARING_VIOLATION)`) and
`read_other.go` (`false`). `internal/fsx/write.go`: `renameWindow` a variable; `ReplaceFile(tmp, name)` is
`rename(osFS{}, tmp, name)`. `auth.write` unchanged. Checked by `go vet` of `internal/fsx` and `internal/auth` for
darwin, linux and windows (exit 0 each) and by the red run below; `make preflight` ran once, on the finished tree.

**Step 1b, red at 1a** (macOS for the portable tests, the Windows laptop through `run-on-host --need windows`; temp paths
shown as `<tmp>`):

| Row | Test | Red at 1a (failing line) |
| :--- | :--- | :--- |
| R1, R2, R4, R5, R6 | `TestReadRegularFileRetryLoop` (`internal/fsx/regular_retry_test.go`) | `regular_retry_test.go:95: open called 1 times; want 3` (released); `:87: gave up after 1 calls and 0s; want more than one call and at least the 50ms window` (held past); `:95: open called 1 times; want 2` (held then gone; held then another error); "another error first" passes, as planned |
| R3 | `TestSharingViolationPredicate` (`read_windows_test.go`) | passes (a guard; P4b) |
| R8 | `TestSharingViolationNever` (`read_other_test.go`) | passes (a guard; P4a) |
| R1, R9, R10 | `TestReadRegularFileSurvivesAHeldHandle` (`read_windows_test.go`) | `read_windows_test.go:85: ReadRegularFile under a held handle = "", open <tmp>\...\auth.json: The process cannot access the file because it is being used by another process.` |
| C1 to C3, C8 | `TestAuthReadersSurviveAHeldHandle` (`internal/auth/held_windows_test.go`; rows `auth.Load`, `Clients.Load`, `readHostID`) | `held_windows_test.go:105:` for each row: `auth.json: open ...`, `clients.json: open ...`, `host-id: open ...`, each the sharing-violation text |
| X1 | `TestSetKeyReplaceSurvivesAReader` (`internal/auth/store_windows_test.go`) | `store_windows_test.go:39: SetKey under a reader: auth.json: rename <tmp>\...\.auth.json.<n>.tmp <tmp>\...\auth.json: Access is denied.` |
| X3 | `TestClientsSaveReplaceSurvivesAReader` (same file) | `store_windows_test.go:62: Clients.Save under a reader: auth.json: rename ... clients.json: Access is denied.` |
| X2 | `TestReplaceFileGivesUp` (`internal/fsx/write_windows_test.go`) | passes (a guard; P5) |
| X4 | `TestReplaceFileNotTransient` (`internal/fsx/write_test.go`) | passes (a guard) |
| R9, R10 | `go test -race -count=200 -run '^TestSignInTypeAheadNotSentAsPrompt$' ./internal/cli/`, the laptop | **4 of 200** failed, each `login_test.go:856: auth.json: open <test temp dir>\auth.json: The process cannot access the file because it is being used by another process.` (the MADR's run at `9a22518`: 7 of 200) |

Every red test failed on its assertion; none failed to compile.

**Step 1c, the changes.** `ReadRegularFile` opens through `openRetrying`: `openFile` again while `retriable(err)` and
`readWindow` has not passed, pauses 1 ms doubling to 100 ms, any other error at once, the last error at the deadline.
`auth.write`: `os.Rename(name, path)` became `fsx.ReplaceFile(name, path)`; the wrapping and the temp file's removal are
unchanged. Nothing in `internal/cli`, `hostid.go` or `migrate.go` changed. One lint finding on the first pre-add check
(revive `var-declaration` on `retriable`'s explicit type) was fixed by dropping the type.

**Green.** Every test above passes on the laptop; the portable ones on macOS and Linux.

**Plants** (SDK `tools/plant.py --root <this repo>`, each on a scratch clone; the laptop's through `run-on-host --need
windows` from the clone; exit 0 for each spec, "all plant(s) caught"):

| Plant | Where run | Caught by (failing line) |
| :--- | :--- | :--- |
| P1 one open | macOS; the laptop | `regular_retry_test.go:95: open called 1 times; want 3`, `:87`; `read_windows_test.go:85: ... being used by another process.`; `held_windows_test.go:105` for each row. The `-count=200` entry cites the base red run above (note 3) |
| P2 every error retried | macOS | `regular_retry_test.go:95: open called 57 times; want 2` (held then gone), `56 times; want 2` (another error), `56 times; want 1` (first error) |
| P3 no read deadline | macOS | `regular_retry_test.go:73: still blocked after 10s` (`bounded`, held past the window) |
| P4a `sharingViolation` true off Windows | macOS; a Linux host | `read_other_test.go:20: sharingViolation(device or resource busy) = true; want false off Windows`, and each other case |
| P4b `sharingViolation` false on Windows | the laptop | `read_windows_test.go:29: sharingViolation(The process cannot access the file because it is being used by another process.) = false; want true`; also `read_windows_test.go:85`, `held_windows_test.go:105` |
| P5 `ReplaceFile` is `os.Rename` | the laptop | `write_windows_test.go:39: ReplaceFile gave up after 0s; want at least the 50ms window`; `store_windows_test.go:39`, `:62` |
| P6 `auth.write` back to `os.Rename` | the laptop | `store_windows_test.go:39: SetKey under a reader: ... Access is denied.`; `:62` |
| P7 `auth.Load` by `os.ReadFile` | the laptop | `held_windows_test.go:102: auth.Load did not open through fsx's seam, or was not refused with a sharing violation (<nil>)` |

**Gates** (each status captured before any filter):

| Command | Host | Exit |
| :--- | :--- | :--- |
| `make pre-add-check FILES=<the 12 Go files>` (`run-heavy`) | macOS | 2 (the revive finding), then 0 |
| `go test -race -count=200 -run '^TestSignInTypeAheadNotSentAsPrompt$' ./internal/cli/` (`--need windows --race`), 19:24 | the Windows laptop | 0 (0 of 200) |
| `go test -race -count=20 ./internal/auth/ ./internal/fsx/` | the Windows laptop | 0 |
| `make preflight` | the Windows laptop | 0 |
| the `-count=200` run again, 19:34 | the Windows laptop | 0 (0 of 200) |
| `go test -count=1 ./internal/fsx/ ./internal/auth/` | a Linux host; macOS | 0; 0 |
| `make preflight` | a Linux host (`run-on-host`); macOS (`run-heavy`) | 0; 0 |
| `make check-test-isolation` (the whole suite, both modules; not a PLAN gate, run because `ReadRegularFile` has 16 callers) | the Windows laptop (`pick-host`'s choice) | 0 ("check-test-isolation: ok") |
| `make check-records`, `make markdownlint`, SDK `tools/denyscan.py --root <this repo> --changed` | macOS | 0, 0, 0 (0 hits in 16 files) |

Exit: met for Phase 1: no blocker, every row tested (X5, X6 and C3 are rows with no change, as planned), plants caught;
no existing test, golden or fixture changed.

### Phase 2, pass 1 (2026-11-22): close-out

1. Confirmation: (1) the reader's and the writer's held-handle tests failed at the base and pass after (above); (2) 0 of
   200 twice on the laptop, against 4 of 200 at the base; (3) each plant turned its test red; (4) `make preflight`
   passed on the three hosts. The MADR stays `accepted`; this PLAN stays `in-progress` until the owner's commit.
2. Row O1's follow-up is written into `docs/README.md`'s 0067 PLAN row as a named open item.
3. The v0.7.0 tag: this record's build is done, unstaged, for the coordinator's gate and the owner's commit; this PLAN
   does not tag.
4. Docs: one sentence in `docs/architecture.md`'s `internal/fsx` row naming the retries of `ReadRegularFile` and
   `ReplaceFile` on Windows (note 4). The new exported names, `ReplaceFile` and `SetOpenFile`, carry doc comments.

### Phase 2, close-out (2026-11-22, after the commit)

The owner committed the build as `5962276` (`fix(fsx): retry Windows sharing violations on config reads and auth.json
replace`), on top of the records commit `b6eb169`. Phase 2 step 1 said this PLAN stays `in-progress` until that commit;
it is made, so this PLAN's `status` is `complete` (set by an assertion that the old line occurred once), and
`docs/README.md`'s 0067 PLAN row follows. The MADR stays `accepted`. Row O1's follow-up stays the named open item of the
index row: whether to open its record is the owner's. 0067 gated v0.7.0 (the owner's "Yes; fix 0067 first"); that gate
is met, and the v0.7.0 tag stays the owner's, under
[0055-PLAN](0055-PLAN-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md)'s note of this date.
