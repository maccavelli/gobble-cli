---
status: proposed
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
