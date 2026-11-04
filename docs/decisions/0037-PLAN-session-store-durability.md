---
status: in-progress
date: 2026-11-04
associated-madr: "0037-MADR-hardening-integrity-and-performance.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement keeping the session store whole across a failed write and a killed writer

Associated MADR: [0037-MADR-hardening-integrity-and-performance.md](0037-MADR-hardening-integrity-and-performance.md)
(accepted 2026-10-30; amended 2026-11-04). This plan carries out D16 and the amendment's D36–D41, with the owner's
answers to Q8–Q10. It is the MADR's fourth plan (D34, D36) and needs none of the other three. When it is complete,
[gobble-cli#7](https://github.com/maccavelli/gobble-cli/issues/7)'s last open unit, U3, is done.

## Goal

When this plan is complete:

* an `Append` that fails, wholly or in part, leaves the session file and `Entries` as they were, on macOS, the Linux
  machine and the Windows laptop (D37);
* a failed first write leaves no file, and the next `Append` creates it (D37);
* `Open` cuts an unparseable torn last line of a current-version file and logs the bytes it cut; it keeps a complete
  entry that lacks only its newline, and never touches an older file (D37);
* a writer killed mid-append reopens with every entry it acknowledged, and the next `Append` gives a file that reads
  back with nothing skipped (D40);
* `parse` reads a file line by line with `bytes.Lines`, if the benchmark shows a gain (D41);
* the package comment and `docs/architecture.md` state the store's policy: no entry cap (D38), no sync (D39), and the
  one exception to "a file is never rewritten" (D37).

`acpserver/testdata/session.golden`, `session/jsonl/testdata/`, and every CLI golden are byte-identical throughout: a
session that never meets a failed write is recorded exactly as today.

## Scope

* **In:** the files each phase names, and nothing else. A file a phase needs and does not name is a deviation (C8).
* **Out:**
  * the MADR's plan 2 (D12–D15, D17–D22), including the wire's 10 MiB frame bound, and plan 3 (D23–D33), including
    D24's streaming `List`;
  * a cap on an entry's size (D38) and any `fsync` (D39), both decided against;
  * a torn header left by a writer killed during `create`, which stays `session.ErrCorrupt` (D37);
  * pushing and tagging, which need the owner's ask in the turn they happen (`AGENTS.md`);
  * commenting on or closing #7, which P4 drafts and the owner says when.

## Conventions every phase follows

These are [0037-PLAN-permission-and-file-tool-hardening.md](0037-PLAN-permission-and-file-tool-hardening.md)'s C1–C9,
restated so that this plan stands alone, with C10 of its own.

* **C1. Start state.**
  * The tree is clean apart from this plan's own uncommitted records, after P0.
  * `HEAD` is the previous phase's commit on `main`.
  * `go version` is `go1.27.3`.
  * `git rev-parse --path-format=absolute --git-path hooks` is the global hooks directory (checked at P0).
* **C2. Scratch work.**
  * Every probe and every deliberately broken copy lives in the session's scratch directory, never in the tree.
  * A broken copy is made as follows:
    1. List the tree's files with `git ls-files -co --exclude-standard`.
    2. Copy those files into `<scratch>/break-<phase>-<n>/`.
    3. Apply the named break there with an exact replacement whose old text is asserted present once: a Python script,
       stdlib only.
    4. Run the named command there.
* **C3. Long output.** Every gate writes its whole output to `<scratch>/<phase>-<gate>.log`, and its exit status is
  taken before any filter.
* **C4. Local gates (G-local),** in this order, each passing:
  1. `make pre-add-check FILES="<every changed .go file>"`;
  2. `make preflight`;
  3. `make race`.

  No phase touches `internal/cli/...`, so `make perf` and `make probe-conhost` are not run.
* **C5. Host gates (G-hosts).** These run on the Linux machine and the Windows laptop, on a copy of the tree, never
  their own checkouts:
  * `go test -count=1 ./...`;
  * `CGO_ENABLED=1 go test -race -count=1 ./...`;
  * `make preflight` up to `verify-build-metadata`, which needs a commit and is recorded as macOS-only;
  * `make check-records markdownlint`.

  Hostnames are never written. The Windows laptop matters most here: D37's cut goes through the path because of how
  Go opens an `O_APPEND` file there.
* **C6. Commit.**
  * At the end of each phase, after G-local and G-hosts pass: `git add` of exactly the phase's files, then
    `git commit --no-edit`, on `main`.
  * A commit to `main` is made only on the owner's word in the turn it happens. Approving this plan is not that word.
  * The phase's execution record, and this plan's `docs/README.md` row, go in the same commit.
* **C7. Records' paths.** No hostname, account name or real machine path goes into a committed file, a log excerpt
  included. Paths are quoted relative to the repository.
* **C8. Deviations.** Any of these stops the phase:
  * a failure the phase does not name as expected;
  * a file outside its list;
  * an API not as written here;
  * an `apidiff` finding. `session/jsonl` is a stable package, and this plan adds nothing exported.

  The owner is given:
  * the evidence: file, line, exact failure, and whether it is pre-existing;
  * resolutions, the recommended one first, each with its cost;
  * what doing nothing means.

  No workaround is offered. The chosen resolution is recorded as a dated deviation entry here, and in the MADR when a
  decision or a fact changes, before work continues.
* **C9. Seen failing.** Every new test or case is run against a broken copy (C2) that undoes the phase's change, and is
  seen to fail with the expected message. The execution record quotes that failure.
* **C10. A killed process's lock.** On Unix, `flock` is released when the holder dies. On Windows, `LockFileEx`'s lock
  is released by the system after the process ends, with no stated bound. A test that reopens a killed writer's
  session therefore retries `Open` on `session.ErrLocked` for at most 5 s after `Wait` returns. Any other error fails
  at once. This is the platform's documented release, not a loosened assertion. A retry that runs out fails the test.

## Order

P0, then P1 to P4 in order. P2's test uses P1's cut, and P3 changes the `parse` that P1 and P2 test.

## Implementation Steps

### P0 — the records

1. Check C1's hooks directory and record it.
2. The owner commits the MADR amendment, this plan, the note on
   [0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md)'s
   D22, and the `docs/README.md` rows, under the bootstrap exception: records only.
3. `make check-records markdownlint` passes before that commit.

### P1 — a failed write leaves the file and memory as they were (MADR D16, D37, D38, D39)

**Files:** `session/jsonl/store.go`, `session/jsonl/store_test.go`, `session/jsonl/doc.go`, `docs/architecture.md`,
this plan, `docs/README.md`.

1. **Facts re-read.** Before any edit, confirm each and record the result:
   * `store.go` is as the amendment quotes it: `Append` at `:303-325`, `create` at `:329-357`, `write` at `:367-379`,
     `readFile` at `:171-184`.
   * `TestMalformedAndTornLines` is at `store_test.go:148-183` and expects `skipped != 2` to fail.
   * **The Windows cut.** On the Windows laptop, a scratch program opens a file with `os.O_WRONLY|os.O_APPEND`, calls
     `f.Truncate` and then `os.Truncate(path, n)`. Expected: the first fails with an access error and the second
     succeeds. If the first succeeds, the amendment's reason is wrong. That stops the phase (C8), and the MADR's fact is
     corrected; the decision, to cut through the path, works on every host either way.
2. **The test seam.** `Store` gains two unexported fields, set by `New`:
   * `openFile func(name string, flag int, perm os.FileMode) (appendFile, error)`, defaulting to `os.OpenFile`;
   * `truncate func(name string, size int64) error`, defaulting to `os.Truncate`.

   `appendFile` is an unexported interface of `Write([]byte) (int, error)` and `Close() error`, and `fileLog.file`
   takes that type. `create` and `write` open through `openFile`. Nothing exported changes.
3. **The size.** `fileLog` gains `size int64`. `Open` sets it to the length of the bytes `readFile` read, after any cut
   in step 7. `readFile` returns that length beside `torn`. Each `write` that succeeds adds `len(b)`.
4. **`Append`.**
   * The entry is encoded first, and an encode error returns with nothing changed.
   * Before the file exists, the entry joins `l.entries` as today. That is the lazy creation of 0008-MADR D19 item 6.
     If it makes a conversation, `create` runs. When `create` fails, the entry is removed again.
   * Once the file exists, the line is written first, and the entry joins `l.entries` only when the write succeeds.
   * When `l.torn` is set, the newline put before the line is cleared only once the write succeeds.
5. **`write` on failure.** When `Write` returns an error, `write` calls `truncate(l.path, l.size)`. If that fails too,
   it sets `l.torn`. It returns the write's error, joined with the cut's when there is one.
6. **`create` on failure.** If the first write fails:
   * the new file is closed and removed;
   * the lock is released;
   * `l.file`, `l.release` and `l.written` are reset;
   * the id is returned to `s.creating`;
   * the error is returned.

   An encode failure, or a failure to open the file, is unchanged, except that the id stays pending.
7. **`Open`'s cut.** After `readFile`, when the file's version is current, it is torn, and its last line (the bytes
   after the last `\n`) does not decode as an entry with a type, `Open`:
   * cuts the file to the last newline through `truncate`;
   * logs at Warn `session: torn last line cut`, with `file` (the base name) and `bytes`;
   * clears `torn`.

   A cut that fails is returned from `Open` with the lock released. `Read` is unchanged.
8. **Docs.**
   * `doc.go`: the sentence "A file is never rewritten." becomes: under the writer lock, gobble only ever cuts off bytes
     it wrote itself, either a failed append's or an unparseable torn last line; entries have no size cap; and appends
     are not synced, so a crash keeps every acknowledged entry, while power loss can lose the last ones.
   * `docs/architecture.md`'s `session/jsonl` row gains the same three facts in one sentence.
9. **Tests** in `store_test.go`. Each uses a temp directory and the seam where it needs a failure:
   * `TestShortWriteIsCut`: the fake file writes half the line, then returns an error. Then:
     * `Append` returns the error;
     * `Entries` is unchanged;
     * the file's bytes equal those from before;
     * the next `Append` succeeds;
     * `Read` shows 0 skipped and exactly the entries that succeeded.
   * `TestFailedCutMarksTorn`: the write fails partway and `truncate` fails. Then:
     * `Append` returns both errors;
     * the next `Append` succeeds;
     * `Read` shows 1 skipped, and the next entry present and whole.
   * `TestFailedCreateRetries`: the first write fails partway. Then:
     * no session file exists (`Read` returns `session.ErrNotFound`);
     * `filelock.TryLock` on the lock path succeeds, proving the lock was released;
     * the next `Append` creates the file;
     * `Read` shows every entry but the failed one, and the new one.
   * `TestMalformedAndTornLines` changes by D37: `skipped != 2` becomes `skipped != 1`. It also asserts that:
     * the file has no byte of the torn line;
     * a captured log has `session: torn last line cut` with the torn line's length as `bytes`.
   * `TestOpenKeepsCompleteLineWithoutNewline`: a valid entry with no newline survives `Open`, and the next `Append`
     goes on a line of its own. `Read` shows both entries and 0 skipped.
   * `TestOpenLeavesOlderFile`: a v2 fixture is copied to a temp store and given a torn tail. Its bytes are unchanged
     after `Open` and `Close`.
   * `TestReadDoesNotCut`: `Read` of a torn file leaves its bytes unchanged.
10. **Seen failing (C9).** Each break is applied to a copy of the finished phase:
    * no cut in `write`: `TestShortWriteIsCut` reports a skipped line or a merged entry;
    * the entry added before the write: `TestShortWriteIsCut` reports `Entries` grown;
    * no torn mark when the cut fails: `TestFailedCutMarksTorn` reports the next entry missing;
    * `create` as it was: `TestFailedCreateRetries` reports a file present, or a held lock;
    * no cut in `Open`: `TestMalformedAndTornLines` reports `2 skipped`;
    * a cut without the decode check: `TestOpenKeepsCompleteLineWithoutNewline` reports an entry missing;
    * a cut without the version check: `TestOpenLeavesOlderFile` reports changed bytes.
11. **Unchanged.** `git status --short -- '*.golden' session/jsonl/testdata` is empty after the tests.
12. G-local and G-hosts (C4, C5). Commit (C6).

### P2 — a writer killed mid-append reopens with every acknowledged entry (MADR D40)

**Files:** `session/jsonl/crash_test.go` (new), this plan, `docs/README.md`.

1. **The helper.** `crash_test.go` holds the package's `TestMain`, which is a fresh one: `session/jsonl`'s tests have
   none today. When `GOBBLE_JSONL_WRITER` is set, the binary runs as a writer, as `internal/proctree/tree_test.go`'s
   helper does. Otherwise it runs `m.Run()`. The writer:
   * opens a store at `GOBBLE_JSONL_DIR`, creates session `s1`, and appends a user message, which creates the file;
   * in mode `loop`, appends entries of 256 KiB of text with ids in sequence, printing `ok <id>` after each `Append`
     returns nil;
   * in mode `torn`, appends three entries, printing each, then writes half of a fourth entry's line straight to the
     file, prints `torn`, and sleeps until it is killed.
2. **`TestKilledWriterReopens`,** in three rounds: `loop` killed after 3 acknowledgements, `loop` after 20, and
   `torn`. Each round:
   1. starts the writer, reads its output, and calls `Process.Kill` at the round's point, then `Wait`;
   2. opens the session, retrying only on `session.ErrLocked` (C10), with a logger captured into a buffer;
   3. asserts that every acknowledged id is in `Entries`, in order;
   4. asserts that at most one entry past the last acknowledged one is present, since it may have completed before the
      kill;
   5. asserts that the file ends in `\n`;
   6. when the log has `session: torn last line cut`, asserts that its `bytes` equals the size before `Open` minus the
      size after. In `torn` mode the line is required;
   7. appends one more entry, closes, and asserts that `Read` shows 0 skipped with that entry last.
3. **Seen failing (C9):**
   * no cut in `Open`: the `torn` round reports `1 skipped`. This happens on every run, since the torn tail is the
     writer's own;
   * `Open` cutting the last line even when it has a newline: the `loop` rounds report an acknowledged id missing.
   * The record notes whether any `loop` round tore a line on its own on each host. That is information, not an
     assertion.
4. **Run time.** Record the test's wall time on each host. Over 10 s on any host is a deviation (C8). The round sizes
   are then reconsidered with the owner; they are not cut silently.
5. G-local and G-hosts (C4, C5). Commit (C6).

### P3 — `parse` reads line by line (MADR D41)

**Files:** `session/jsonl/store.go`, `session/jsonl/bench_test.go` (new), this plan, `docs/README.md`.

1. **The benchmark first.** `bench_test.go` adds `BenchmarkParse`. It encodes a v3 header and 2,000 entries of about
   512 bytes of text, then calls `parse` on the bytes. Its result is checked once, outside the timer.
2. **Before.** On the phase's starting commit, run
   `go test -run '^$' -bench '^BenchmarkParse$' -benchmem -count=10 ./session/jsonl` on macOS. Record the median
   `ns/op`, `B/op` and `allocs/op`.
3. **The change.** `parse` iterates `bytes.Lines(b)`. It skips blank lines until the header, decodes the header, then
   decodes each further non-blank line as today. The `[][]byte` from `bytes.Split` is gone, and the error messages are
   unchanged.
4. **After.** Run the same command and record the same three numbers.
   * The change is kept only when `B/op` and `allocs/op` fall, and `ns/op` does not rise beyond the run's spread.
   * Otherwise step 3 is reverted, the benchmark stays, and the record says why. That is D41's "not made", not a
     deviation.
5. **Equivalence.**
   * `TestFixturesRoundTrip`, `TestFixturesMigrate`, `TestFixtureDamage` and P1's tests pass unchanged.
   * `go test -run '^$' -fuzz '^FuzzParse$' -fuzztime 60s ./session/jsonl` runs on a scratch copy (C2), since a failing
     input would be written under the package's `testdata/`. The record notes that it ran clean.
6. **Seen failing (C9).** The benchmark is not an assertion. The break for this phase is `parse` dropping the final line
   when it has no newline: `TestOpenKeepsCompleteLineWithoutNewline` and `FuzzParse`'s seeds report it.
7. G-local and G-hosts (C4, C5). Commit (C6).

### P4 — close-out

1. **Goldens.** `git diff --stat <P0>..HEAD -- '*.golden' session/jsonl/testdata` is empty.
2. **The disclosure guard,** run over the outgoing commits (`AGENTS.md`, "Identifiers"), is clean.
3. **#7.** Draft the closing comment in this record:
   * U3's parts, each with its commit and test;
   * D38's answer to the cap, and why;
   * D39's answer to `fsync`.

   Posting it and closing #7 happen on the owner's word, after the commits are pushed.
4. **Status.** This plan becomes `complete` when every Goal item is met. The MADR stays `accepted`.
5. Commit (C6).

## Verification

Each phase's own steps are its acceptance criteria. Across the plan:

* every test named in P1 and P2 exists, passes on all three hosts and under `-race`, and has its quoted failure on a
  broken copy (C9);
* #7's U3 acceptance, "crash-kill/reopen/replay test passes with zero silent truncation beyond the last complete line",
  is `TestKilledWriterReopens`: no acknowledged entry is lost, and any cut is logged with its size;
* the goldens and `session/jsonl/testdata/` are byte-identical to P0's commit;
* `apidiff` reports nothing for `session/jsonl`;
* P3's numbers before and after are in the record, with the decision they led to.

## Rollout and Rollback

* Each phase is one commit, and reverts on its own with `git revert`. Reverting P1 reverts P2 first, since P2's
  `torn` round needs `Open`'s cut.
* Nothing is pushed by this plan. The owner pushes, after the disclosure guard.
* **What users see change:**
  * P1: a session whose disk filled, or whose write failed, no longer loses the next entry or keeps an entry the file
    lacks. A torn last line from an earlier crash is cut when the session is resumed, with one warning in the log;
  * P2 and P3: nothing.
* A session file written by this plan's code is read by any earlier gobble and by Pi: the format is unchanged, and the
  only difference on disk is the absence of a torn line.

## Execution record

Approved by the owner on 2026-11-04: "proceed to p1".

### P0 record (2026-11-04): the records

* The hooks directory is the global one (C1).
* The owner committed the MADR amendment, this plan, 0039-MADR's note and the `docs/README.md` rows as `38da632`,
  records only. `make check-records markdownlint` passed before it.

### P1 record (2026-11-04): a failed write leaves the file and memory as they were

1. **Facts re-read.** `store.go` and `TestMalformedAndTornLines` were at the lines the amendment quotes.
   * **The Windows cut**, a scratch program. It opens a file `O_WRONLY|O_APPEND`, writes, then truncates through the
     handle and then through the path.
     * On the Windows laptop: `handle Truncate: … Access is denied.`, then `path Truncate: <nil>`. The open handle's
       next append landed at the new end (`"line one\nnext\n"`).
     * On macOS and the Linux machine, both truncates succeeded.

     The amendment's fact holds; no deviation.
2. **Built** in `session/jsonl/store.go`:
   * the seam: `Store.openFile` and `Store.truncate`, and the unexported `appendFile`;
   * `fileLog.size`;
   * `Append` writing before it records;
   * `write` cutting back by path, and marking torn when that fails;
   * `create` closing and removing a file whose first write failed, and releasing its lock;
   * `Open`'s `cutTornTail`.

   `readFile` now returns the bytes it read in place of a `torn` flag. As built:
   * **One rule for "an entry".** `decodeEntry` is the condition `parse` already used. It is shared so that `Open`'s cut
     and `parse`'s skip agree on what an entry is.
   * **The pending id.** Step 6's "the id is returned to `s.creating`" is built as: the id is removed only after the
     first write succeeds. It is never removed and put back, so the observable result is the one step 6 names.
   * **The tail.** `Open` does not cut a tail that is the file's first line, the header itself, when it has no newline.
3. **Docs.** `doc.go`'s package comment and `docs/architecture.md`'s `session/jsonl` row, as step 8 says.
4. **Tests**, in `store_test.go`:
   * `TestMalformedAndTornLines` now expects 1 skipped where it expected 2. It also asserts that the torn bytes are gone
     and that the log has `msg="session: torn last line cut"` with `bytes=<the torn line's length>`.
   * New: `TestShortWriteIsCut`, `TestFailedCutMarksTorn`, `TestFailedCreateRetries`,
     `TestOpenKeepsCompleteLineWithoutNewline`, `TestOpenLeavesOlderFile` and `TestReadDoesNotCut`. All pass.
5. **Seen failing (C9).** A scratch script applies each break to a copy, with its old text asserted present once. Each
   failed as expected:
   1. no cut in `write`: `store_test.go:265: file after a failed append:`;
   2. the entry added before the write: `store_test.go:262: Entries after a failed append: 2, want 1`;
   3. no torn mark when the cut fails: `store_test.go:301: read: 1 entries, 1 skipped, <nil>; want 2 and 1`;
   4. `create` as it was: `store_test.go:327: read after a failed create: session: not a valid session: …_s1.jsonl:
      empty file, want ErrNotFound`;
   5. no cut in `Open`: `store_test.go:171: file after Open keeps the torn line:`;
   6. a cut without the decode check: `store_test.go:378: read: 2 entries, 0 skipped, <nil>; want 3 and 0`;
   7. a cut without the version check: `store_test.go:404: Open changed an older file: 6550 bytes, want 6589`;
   8. **added**, since step 10 named no break for `TestReadDoesNotCut`, `Read` cutting:
      `store_test.go:416: Read changed the file: 8764 bytes, want 8803`.
6. **Unchanged.** `git status --short -- '*.golden' session/jsonl/testdata` was empty.
7. **G-local.**
   * `make pre-add-check` first failed on `gofmt`: a comment's alignment in `fileLog`.
   * `make preflight` then failed on gocritic's `typeDefFirst`, since `cutTornTail` sat above `fileLog`. It was moved
     below the type.
   * After both fixes, `make pre-add-check FILES=…` passed, `make preflight` printed "preflight passed", and `make race`
     passed.
   * The eight breaks were run again after the move, and all failed as before.
   * `apidiff` printed `no-op below v1.0.0`, and nothing exported was added.
8. **G-hosts.** The tree was copied to the Linux machine and the Windows laptop, 829 files each, both on `go1.27.3`.
   The Windows laptop ran with `TMP` and `TEMP` set to a scratch directory. On both hosts:
   * `go test -count=1 ./...` exited 0;
   * `CGO_ENABLED=1 go test -race -count=1 ./...` exited 0;
   * `make preflight` passed every gate up to `verify-build-metadata`. That gate failed as expected: `git rev-parse
     HEAD` exited 128 in the uncommitted copy. It is macOS-only (C5);
   * `make check-records markdownlint` exited 0.

The owner committed P1 as `7bd6800`.

### P2 record (2026-11-04): a writer killed mid-append reopens with every acknowledged entry

1. **The helper.** `session/jsonl/crash_test.go` holds the package's first `TestMain`, and `runWriter`. In mode `loop`,
   the writer appends 256 KiB entries and stops at 200 with exit 3 if nothing kills it. In mode `torn`, it appends
   three, then writes half of a fourth line past the store.
2. **`TestKilledWriterReopens`,** with rounds `loop-3`, `loop-20` and `torn-4`, as step 2 says. As built:
   * **Acknowledgements printed before the kill count too.** The writer keeps appending between the test's last read
     and the kill. So after `Process.Kill` the test reads the writer's output to its end, then calls `Wait`, since
     `os/exec` forbids `Wait` before the pipe's reads are done. Every `ok` line counts, which makes step 2.4's "at most
     one past the last acknowledged" exact.
   * **Checked on disk, not only in memory.** The first break run showed that step 2.3, checked against `Entries` alone,
     missed a cut that removed an acknowledged entry from the file but not from memory. That break still failed the
     test, but only at step 2.7's read-back (`read after the next append: 3 entries, 0 skipped, <nil>; want 4 and 0`).
     The test now also reads the file right after `Open`. It asserts 0 skipped and as many entries as the log holds, and
     looks for each acknowledged id there. This is stricter than step 2.3, in the phase's own file.
3. **Seen failing (C9),** on copies, each old text asserted present once:
   * **No cut in `Open`:** `torn-4` failed with
     * `file after Open: 4 entries, 1 skipped, <nil>; the log holds 4`,
     * `file does not end in a newline after Open`,
     * `torn writer: no cut logged`,
     * `read after the next append: 5 entries, 1 skipped, <nil>; want 5 and 0`.

     The two `loop` rounds passed, since neither tore a line.
   * **`Open` cutting the last line even when it has a newline:** this break had three edits: the newline check
     removed, the last line taken before a final newline, and the decode check off. `loop-3` failed with `file after
     Open: 2 entries, 0 skipped, <nil>; the log holds 3` and `acknowledged entry e0000002 missing at 2 in the file`.
     `loop-20` failed the same way at `e0000019`.
   * **Self-torn lines.** No `loop` round tore a line on its own on any host: each logged `torn tail cut: false`. Each
     kill landed between writes, so the torn path is covered by the `torn` round on every host, as D40 intends.
4. **Run time** of `TestKilledWriterReopens`: 0.08 s on macOS, 0.13 s on the Linux machine and 0.31 s on the Windows
   laptop. That is under step 4's 10 s on every host. Whether C10's retry ever ran is not logged.
5. **Unchanged.** `git status --short -- '*.golden' session/jsonl/testdata` was empty.
6. **G-local.** `make pre-add-check FILES="session/jsonl/crash_test.go"` passed, `make preflight` printed "preflight
   passed", and `make race` passed.
7. **G-hosts.** 830 files on each host, as in P1:
   * `go test -count=1 ./...` and `CGO_ENABLED=1 go test -race -count=1 ./...` exited 0;
   * `make preflight` passed up to `verify-build-metadata`, which exited on `git rev-parse HEAD`'s 128, as expected;
   * `make check-records markdownlint` exited 0.
