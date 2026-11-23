---
status: accepted
date: 2026-11-22
decision-makers: repository owner
consulted: "the owner's rulings of 2026-11-22, relayed by the coordinator: the Windows flake is investigated under its own record (0052-MADR, amendment of 2026-11-22, item 3), and it gates gobble v0.7.0 (\"Yes; fix 0067 first\"); independent review 1 of this record; the reproduction and probes below"
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Retry a Config File's Read, and auth.json's Replace, When Windows Refuses Them for a Moment

Associated PLAN: [0067-PLAN-windows-sharing-violation-when-reading-auth-json.md](0067-PLAN-windows-sharing-violation-when-reading-auth-json.md)

*(2026-11-22, second draft, after independent review 1: the writer side is decided (D4), the probes now discriminate and
were run (M1 established), every reader and writer has a row, the plants and tests are corrected.)*

*(2026-11-22, accepted: the owner answered Q1 and Q2 as recommended; see "Owner rulings". No row is open.)*

## Context and Problem Statement

`TestSignInTypeAheadNotSentAsPrompt` (`internal/cli/login_test.go`, 0063's test) failed twice on the Windows laptop,
each time passing on the next run: in 0052's P5 gate and in 0052's P6 pass 2, both with `auth.json: open ...: The
process cannot access the file because it is being used by another process.`
([0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md),
deviation 9; [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md),
amendment of 2026-11-22, item 3). P5's agent read it as the test's poll racing the key's save by rename; P6's as
`storedKey`'s read racing the writer. 0052 handed it to this record, and the owner ruled on 2026-11-22 that it gates the
v0.7.0 tag: **0067 is built before the v0.7.0 tag** (a prerequisite, not a follow-up; the tag stays the owner's).

### Reproduction (2026-11-22, HEAD `9a22518`, the Windows laptop, through `run-on-host --need windows`, no file of the repository changed)

| Command | Runs | Failures | Error line |
| :--- | :--- | :--- | :--- |
| `go test -count=200 -run '^TestSignInTypeAheadNotSentAsPrompt$' ./internal/cli/` | 200 | 0 | none |
| the same with `-race` | 200 | 7 (3.5 %) | `login_test.go:856: auth.json: open <test temp dir>\auth.json: The process cannot access the file because it is being used by another process.` (all 7 identical) |

`-race` slows both goroutines and widens the window; without it 20 runs (0052 P5) and 200 runs (here) passed. All seven
failures are an `open` error, the read side. A WSL distro is Linux, where a rename never conflicts, so it is no evidence.

### Probes (2026-11-22, temp programs in a scratch module, run on the Windows laptop, Go 1.27.4; none touched the repository)

| Probe | Result |
| :--- | :--- |
| (i) `os.Open` (Go opens with share read and write, never delete) against a holder: read access, share all | succeeds |
| (i) a holder with DELETE access, share read, write and delete; or share read and write; or DELETE plus read, share all | **refused** with the sharing-violation text, each |
| (i) a holder with read access, share read and write | succeeds |
| (ii) a rename loop replacing the file while 3000 reads open it: reads by `os.Open` | 81 of 3000 refused (2.7 %), the same text |
| (ii) the same loop, reads by raw `CreateFile` with read access and share read, write **and delete** | 0 of 3000 refused |
| (ii) renames in the same loops | 13 failed (of 67) against the `os.Open` reader, 6 failed (of 75) against the share-delete reader, each `Access is denied` |
| (iii) write by rename then read, in one goroutine, no other reader: 20 000 iterations | 0 refused |
| writer: a reader holding the target open, share read and write, or share read, write and delete; then `os.Rename` over it | refused both times with `Access is denied` |

What they show:

* **M1 is established.** The refusal is the replacing rename's own handle (DELETE access) against a reader that does not
  share delete, which is every `os.Open` on Windows (probes i, ii). A reader opened with share-delete is never refused (0 of 3000).
* **M2 (an outside scanner holding the file) is not supported**: no refusal in 20 000 quiet write-then-read rounds, with
  nothing of ours holding the file. It is not excluded for a machine with a live scanner; the retry covers it too.
* **The writer side is real and separate.** A replace fails with `Access is denied` while any reader holds the target
  open, whatever the reader's share mode (the writer probe; and 6 failures against the share-delete reader). So FILE_SHARE_DELETE on reads
  (option C) cannot fix the writer; only a retry can. It was seen before: 0005-PLAN's F1a deviation 1 (`TestWriteIsAtomic` failed with
  `renameat ... Access is denied` while a reader held the file), and `internal/fsx/write.go:192-195` says so.
* One result was not expected and is not explained: a holder with read access and share mode none did *not* refuse `os.Open` here. The decisions and tests below
  do not depend on it (the tests' holder is the DELETE-access one of probe i).

### What the code does (read at `9a22518`)

* The writer is `auth.write` (`internal/auth/store.go:412`): a temp file in the same directory, then `os.Rename(name, path)`
  (`:421`), with no retry. Go's `os.Rename` on Windows is `MoveFileEx` with replace-existing. `modify` (`:358`) holds
  the file lock around its `Load` (`:367`) and write. `clients.json` also goes through `auth.write` (`internal/auth/clients.go:115`).
* The reader is `auth.Load` (`:73`), which calls `fsx.ReadRegularFile` (`internal/fsx/regular.go:34`): a plain
  `os.OpenFile`, no retry. The test's `storedKey` (`login_test.go:217`) calls `auth.Load` and `t.Fatal`s on any
  error, inside `waitUntil`'s 5 ms poll.
* `fsx.ReadRegularFile` has **16 production call sites**: `internal/config/resolve.go:130`, `edit.go:75`, `update.go:46`,
  `rulesedit.go:89`; `internal/auth/store.go:74`, `clients.go:122`, `hostid.go:107`, `migrate.go:198`;
  `internal/cli/completion.go:144`, `config.go:366`, `modelsjson.go:96`, `migrate.go:34`; `mcpclient/config.go:84`,
  `mcpclient/file.go:159`; `permission/store.go:64`, `permission/trust.go:142`.
* Other readers of files replaced by rename that do not use `ReadRegularFile`: `internal/cli/choice.go:152` and `:245`
  (`os.ReadFile`; `recent-models.json` is written by `choice.go:284`, a plain `os.Rename`), `internal/cli/editor/history.go:31`
  (`os.Open`; written by `history.go:89`), and the workspace read path `internal/fsx/workspace.go:75`, `:97`
  (its test skips the refusal: `workspace_test.go:223-225`).
* The tree already retries this class of error for writes: `fsx.rename` (`internal/fsx/write.go:196`) retries for
  `renameWindow` (2 s, `:190`) when `transient` (`internal/fsx/rename_windows.go:11`: access denied or sharing violation). The auth store's renames do not use it.
* `auth.json` is replaced only by `auth.write`. `hostid.go:98` and `migrate.go:328` never replace an existing file:
  host-id's absence is checked under its lock (`hostid.go:78`), and the migration's `Lstat` refuses an existing backup (`migrate.go:323`).
* `internal/filelock` never renames or removes a lock file (`lock.go`'s `TryLock`, `lock_windows.go`, `lock_unix.go`).
* Gobble removes no config file: its only `os.Remove` outside its own temp files is `internal/cli/personalitycmd.go:332`, `:338`.
* A file with more than one hard link is written in place, not by rename (`internal/fsx/write.go:131-133`).

### What users see today when a read fails (and after the window it is the same text)

| Caller | Today |
| :--- | :--- |
| `internal/cli/credentials.go:64` | logs "auth.json ... the file is not used" and runs signed out |
| `internal/cli/login.go:149`, `internal/cli/auth.go:370` | the command fails with the error |
| the migration (`internal/cli/migrate.go:34`, `internal/auth/migrate.go:198`) | `auth.ErrUnreadable` is hidden (`internal/cli/main.go:389`) and tried again next run |
| `internal/cli/completion.go:144` | the error is swallowed; completion has no models |

## Decision Drivers

* The defect is in the product, not only the test: every unlocked reader of a config file on Windows can meet it, and a
  just-completed sign-in can fail to save when another process reads.
* Both directions are real and evidenced: the reader's `open` (7 of 200) and the writer's replace (seen in 0005 and by probe).
* No retry may hide a real failure: a missing file, a directory, a permission refusal, or a file that stays held must report as today, after a bound.
* The fix must be testable without timing luck: a held handle, and a seam that releases it on the first refusal.
* The tree's precedent is `fsx.rename`'s bounded retry; a second, different retry is a cost.
* A fix that only loosens the test, while the product keeps the exposure, is a workaround and is not offered as the fix.

## Considered Options

* **A. Retry both directions (recommended).** `fsx.ReadRegularFile` retries its open on a sharing violation; `auth.write` replaces through an exported retrying rename in `fsx`.
* **B. Retry reads only.**
* **C. Open reads with FILE_SHARE_DELETE (a raw `CreateFile`), plus the writer's retry.**
* **D. Change only the test.** `storedKey` tolerates the refusal.
* **E. Do nothing.**

(Locking reads of `auth.json` was considered and dropped: it covers only `auth.Load`, makes read-only commands create `auth.json.lock` and wait on it, and leaves the writer's refusal by a reader in another process.)

## Decision Outcome

Chosen option: "A. Retry both directions", because the probes show both refusals are transient contention that clears at once (M1),
the tree already retries the writer's kind with a bounded window, one change in `fsx.ReadRegularFile` covers sixteen call
sites including the test's, and one exported rename covers `auth.json` and `clients.json`. The owner chose it, with the 2 s window (Owner rulings).

### Owner rulings (2026-11-22, relayed by the coordinator)

| Question | Answer | Effect |
| :--- | :--- | :--- |
| Q1 Which fix? | "Retry reads and the save (Recommended)": option a, retry reads in `fsx.ReadRegularFile` on a sharing violation, and `auth.write`'s replace through the retrying `fsx.ReplaceFile` | row O6 decided; options B, C, D, E not taken; no FILE_SHARE_DELETE open |
| Q2 How long does a read or replace wait? | "2 seconds (Recommended)": option a, the 2 s window, as `renameWindow` | row O7 decided; no per-process total limit, no 500 ms |
| Earlier, the Windows flake | "Investigate under its own record (Recommended)"; then "Yes; fix 0067 first" (it gates v0.7.0) | D8 |

### The decisions

* **D1.** `fsx.ReadRegularFile` retries the `OpenFile` while the error is `ERROR_SHARING_VIOLATION` and the read window has not passed. Only that error:
  all 7 failures were it; `ERROR_ACCESS_DENIED` on a *read* comes only from a file pending deletion, and gobble removes no config file
  (`personalitycmd.go:332`, `:338` aside). Off Windows nothing is retried. The pause grows from 1 ms to 100 ms as `fsx.rename`'s does.
* **D2.** After the window, the **last** error is returned, so a file that stays held reports today's text after the window. If the error changes during the window to anything not retried
  (not-exist, not-regular, too-large, another error), that error is returned at once, so `auth.Load` still gives an empty `File` for a file that vanished.
* **D3.** Errors after the open (not regular, too large, a read error) are returned at once; they are outside the loop.
* **D4.** `auth.write` replaces through an exported `fsx` function (named in Phase 1) that is `fsx.rename`'s retrying loop, with `transient`'s predicate (access denied or sharing violation: the writer's refusal is `Access is denied`, the writer probe), window `renameWindow`. A failure after the window removes the temp file and returns as today.
  `hostid.go:98` and `migrate.go:328` need no change (they never replace an existing file).
* **D5.** `TestSignInTypeAheadNotSentAsPrompt` and `storedKey` are not changed.
* **D6.** Tests: a portable loop test through seams (the open, the predicate, the window), per-platform predicate tests, a Windows held-handle test of the reader whose seam releases the holder on the first refusal (no sleep), a Windows writer test (hold `auth.json` with `os.Open`, call `auth.SetKey`), and one held-handle table over `auth.Load`, `Clients.read` and `readHostID`. The PLAN names each.
  *(Note, 2026-11-22, build: the writer's give-up test is split, by the owner's ruling "Split it; Windows give-up
  (Recommended)": X2 is `TestReplaceFileGivesUp` (Windows, a real `os.Open` holder for the whole call), X4 is the
  portable `TestReplaceFileNotTransient`; a portable fake could not reach `ReplaceFile`, and off Windows `transient` is
  never true (0067-PLAN deviation 1). The table reads `Clients.read` through `Clients.Load`.)*
* **D7.** The readers that do not use `ReadRegularFile` (`choice.go:152`, `:245`, `editor/history.go:31`, the workspace path) and their writers (`choice.go:284`, `history.go:89`) are a **named follow-up**, not this record: none has been seen to fail, the workspace's own test already tolerates the refusal, and each is a different file with a different owner of the question.
* **D8.** The v0.7.0 tag waits for this record's build (the owner's ruling, above).

### Consequences

* Good, because `-race -count=200` on the laptop is expected to pass, a second gobble reading while one signs in no longer fails, and a sign-in no longer fails to save under a reader.
* Good, because one change covers sixteen readers and `clients.json` as well.
* Neutral, because a file held for the whole window now fails after the window, with the same text.
* Bad, because of the waits in Q2's cost list, which the owner decides.
* Bad, because the follow-up files keep the exposure until they get their own record.

### Confirmation

1. The reader's held-handle test and the writer's test are each seen failing at the base and passing after.
2. `go test -race -count=200 -run '^TestSignInTypeAheadNotSentAsPrompt$' ./internal/cli/` on the Windows laptop has 0 failures, twice (the base's is 7 of 200).
3. Each plant turns its test red (the PLAN names them).
4. `make preflight` passes on the three hosts, as every phase's gate does.

### State table

"Today" is `9a22518`. Each decided row's test and plant are in the PLAN. Rows marked `open` wait for the owner.

**Reader (D1 to D3)**

| Row | Input or state | Status | Outcome | Decided by |
| :--- | :--- | :--- | :--- | :--- |
| R1 | Windows; the open meets a replace (sharing violation); released inside the window | decided | the read succeeds | D1; the reader's held-handle test; the loop test |
| R2 | Windows; held past the window | decided | the last error, today's text, after the window | D2; the give-up test |
| R3 | Windows; the open fails with access denied | decided | returned at once, not retried (evidence in D1) | D1; the predicate test |
| R4 | the file does not exist at the first attempt | decided | returned at once; `auth.Load` gives an empty file | D3; the loop test |
| R5 | a sharing violation, then not-exist during the window | decided | not-exist returned at once; `auth.Load` gives an empty file | D2; the loop test (seam) |
| R6 | a sharing violation, then another error not retried | decided | that error returned at once | D2; the loop test (seam) |
| R7 | not regular, too large, or a read error after the open | decided | returned at once | D3; existing `regular_test.go` tests kept |
| R8 | POSIX; any open error | decided | returned at once, one attempt | D1; the per-platform predicate test |
| R9 | reader and writer in one process (the test's shape) | decided | R1 covers it | D1; Confirmation 2 |
| R10 | reader and writer in two processes | decided | R1 covers it; the held-handle test is process-independent | D1 |
| R11 | the real `auth.json` path vs the test's temp path | decided | one code path; no test reads a real file | D1 |

**Callers of the reader (what users see; the text after the window is today's)**

| Row | Input or state | Status | Outcome | Decided by |
| :--- | :--- | :--- | :--- | :--- |
| C1 | `credentials.go:64` | decided | covered; after the window, "the file is not used", signed out, as today | D1; the `auth.Load` table test |
| C2 | `login.go:149`, `auth.go:370` | decided | covered; after the window the command fails as today | D1; the `auth.Load` table test |
| C3 | the migration (`ErrUnreadable`, hidden, tried next run) | decided | covered; unchanged after the window | D1; the migration's existing tests |
| C4 | `completion.go:144` (error swallowed) | decided | covered; a held models.json stalls completion for up to the window, then none | D1; Q2's cost |
| C5 | startup reads of six to eight config files in a row | decided | covered; a whole-directory hold stalls up to k times the window | D1; Q2 (c) |
| C6 | readers inside the lock (`store.go:367`, `clients.go:87`, `hostid.go:78`, `migrate.go:171`) | decided | covered; refused only by an outside holder, and the lock is held up to the window longer; others wait up to 30 s (`store.go:28`) | D1; Q2 |
| C7 | readers outside the lock (`clients.go:38`, the test's poll, the CLI readers) | decided | covered; they meet gobble's own writer | D1 |
| C8 | `Clients.read` and `readHostID` go through the retry | decided | pinned by the held-handle table test | D6 |

**Writer (D4)**

| Row | Input or state | Status | Outcome | Decided by |
| :--- | :--- | :--- | :--- | :--- |
| X1 | `auth.write`'s replace is refused (`Access is denied`) by a reader; released inside the window | decided | the replace succeeds | D4; the writer test |
| X2 | held past the window | decided | the error as today (`auth.json: rename ...`), the temp file removed | D4; the writer give-up test |
| X3 | `clients.json` (goes through `auth.write`) | decided | covered by X1 | D4; the writer test through `Clients.Save` |
| X4 | a rename error that is not transient | decided | returned at once, the temp file removed | D4; the writer loop test |
| X5 | `hostid.go:98`, `migrate.go:328` | decided | unchanged: they never replace an existing file | D4; no test (a row, not a change) |
| X6 | the lock files | decided | nothing to retry: `filelock` never renames or removes one | `filelock/lock.go` |

**Other**

| Row | Input or state | Status | Outcome | Decided by |
| :--- | :--- | :--- | :--- | :--- |
| O1 | readers by `os.ReadFile`/`os.Open` (`choice.go:152`, `:245`, `editor/history.go:31`, `workspace.go:75`, `:97`) and their plain renames (`choice.go:284`, `history.go:89`) | decided | a named follow-up (D7), not this record | D7 |
| O2 | the caller cancels while a read or replace waits | decided | no context; bounded by the window | D1, D4 |
| O3 | a config file with more than one hard link (written in place) | out of scope | a reader can see a partial file; unrelated to this refusal; not `auth.write`'s files | `write.go:131-133` |
| O4 | a half-written `auth.json` or `clients.json` | decided | cannot happen: `auth.write` replaces by rename, a reader sees the old or the new file | `auth.write` |
| O5 | the test itself | decided | unchanged (D5) | D5 |
| O6 | which fix, and the reader's FILE_SHARE_DELETE open (option C) | decided | option A (2026-11-22): retry reads and the save; no FILE_SHARE_DELETE open | Q1 "Retry reads and the save (Recommended)"; D1, D4 |
| O7 | how long a read or replace waits | decided | 2 s, as `renameWindow` (2026-11-22); no per-process total limit | Q2 "2 seconds (Recommended)"; D1, D4 |

Counts: 11 + 8 + 6 + 7 = 32 rows; 31 decided, 1 out of scope, 0 open.

## Pros and Cons of the Options

### A. Retry both directions

* Good, because it covers both evidenced refusals with the tree's own pattern.
* Good, because a held-handle test with a seam makes it deterministic.
* Bad, because a file that never clears fails up to the window late, per read.
* Bad, because it treats the symptom; the mechanism (M1) is known, but the retry does not remove it.

### B. Retry reads only

* Good, because it is smaller and fixes the 7 of 200.
* Bad, because a just-completed sign-in can still fail to save when a reader in another process holds `auth.json`.

### C. Open reads with FILE_SHARE_DELETE, plus the writer's retry

* Good, because probe (ii) shows it removes the reader's refusal at its source (0 of 3000), with no wait on the read side.
* Bad, because it needs a raw `CreateFile` and an `os.NewFile` per platform in `fsx`, in a package with none for reads; the writer's retry is still needed (a replace fails whatever the reader's share mode).
* Bad, because it covers only gobble's own reads; an outside holder (M2) would still refuse them.

### D. Change only the test

* Good, because it is the smallest change.
* Bad, because the product keeps both exposures, and a test that tolerates what the product does not is a workaround.

### E. Do nothing

* Good, because it costs nothing now.
* Bad, because the flake gates v0.7.0 by the owner's ruling and can fail a later Windows gate.

## Owner Questions

Two, both answered 2026-11-22 as recommended (see "Owner rulings"; the options stay as asked). The earlier Q2 (which errors a read retries) and Q3 (the writers) were decided from evidence (D1, D4).

1. **Which fix?** (a) Retry reads in `fsx.ReadRegularFile` and retry `auth.write`'s rename (Recommended): both directions; two small changes; a file that never clears fails up to 2 s late. (b) Retry reads only: smaller; a sign-in's save can still fail under a reader in another process. (c) Open reads with FILE_SHARE_DELETE through a raw `CreateFile`, plus the writer's retry: removes the reader's refusal at its source (probe ii: 0 of 3000), with no read wait; new per-platform open code, and outside holders still refuse.
2. **How long does a read or replace wait?** (a) 2 s, as `renameWindow` (Recommended): one number; a whole-directory hold stalls startup up to k times 2 s (six to eight reads), tab completion up to 2 s, and a locked read holds the lock up to 2 s longer. (b) 500 ms: fails sooner; a slow scan may outlast it. (c) 2 s plus a per-process total limit: bounds the startup stall; one more mechanism to build and test.

## More Information

### Relation to other records

* [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md) and its PLAN: found the flake and handed it here; nothing of 0052 waits on this record, but the v0.7.0 tag it prepares waits (D8).
* [0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md](0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md): owns the test; its assertion is unchanged.
* [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md), deviation 1 of F1a: the earlier sighting of the writer's refusal.

### What could not be established

* Whether an outside scanner ever holds `auth.json` (M2): no evidence; the retry covers it either way.
* Why a share-none read holder did not refuse `os.Open` in probe (i); no decision depends on it.
* Whether any production run has met either failure: no report exists; the exposure is read from the code and the probes.
