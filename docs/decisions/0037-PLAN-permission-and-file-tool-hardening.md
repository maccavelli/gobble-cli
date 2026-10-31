---
status: in-progress
date: 2026-10-30
associated-madr: "0037-MADR-hardening-integrity-and-performance.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement hardening the permission rules and the file tools

Associated MADR: [0037-MADR-hardening-integrity-and-performance.md](0037-MADR-hardening-integrity-and-performance.md)
(accepted 2026-10-30). This plan carries out its category 1, D1–D11, with the owner's answers to Q1 and Q6. It is the
first of the MADR's three plans (D34). The other two are written after this one is approved.

## Goal

When this plan is complete:

* an intercepted `mv` or `cp` into an existing directory does what the shell does, and nothing removes a directory to
  make room for a file (D1);
* no letter case, symlink, Windows spelling, or bash command reaches a protected path without the protected-path limit
  (D2, D3);
* a deny rule for a command or a path cannot be got round by quoting, case, a tab, an absolute program path or a symlink
  (D4, D5);
* a write through a symlink writes its target, and a file with several hard links keeps them (D6, Q1 (a));
* no file tool and no configuration reader hangs on a FIFO, reads past its need, or holds a file it should not
  (D7, D10);
* an edit refuses an ambiguous overlapping match, and keeps the line endings it did not touch (D8);
* command output on disk is capped (D9);
* glob matching is linear, and a recursive copy into itself, by any spelling, is refused and can be cancelled (D11).

Every gate passes on macOS, the Linux machine and the Windows laptop at each phase. `acpserver/testdata/session.golden`
and the CLI's goldens are byte-identical throughout: nothing here changes what a session records or what the terminal
shows for a call that was allowed before.

## Scope

* **In:** the files each phase names, and nothing else. A file a phase needs and does not name is a deviation (C8).
* **Out:**
  * the MADR's categories 2 and 3 (D12–D33), which are its other two plans;
  * 0035's held phases, P12 and P17–P19;
  * the upstream issues go-llmprovider-sdk#10 and acp-go-sdk2#13;
  * pushing and tagging, which need the owner's ask in the turn they happen (`AGENTS.md`).

## Conventions every phase follows

These are 0035-PLAN's conventions, restated so that this plan stands alone.

* **C1. Start state.**
  * The tree is clean, after P0.
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
  3. `make race`;
  4. for a phase that touches `internal/cli/...`, also `make perf`.
* **C5. Host gates (G-hosts).** These run on the Linux machine and the Windows laptop, on a copy of the tree, never
  their own checkouts:
  * `go test -count=1 ./...`;
  * `CGO_ENABLED=1 go test -race -count=1 ./...`;
  * `make preflight` up to `verify-build-metadata`, which needs a commit and is recorded as macOS-only;
  * `make check-records markdownlint`.

  A phase that touches `internal/cli/...` also runs `make perf` on both, and `make probe-conhost` on the Windows laptop.
  Hostnames are never written.
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
  * an `apidiff` finding.

  The owner is given:
  * the evidence: file, line, exact failure, and whether it is pre-existing;
  * resolutions, the recommended one first, each with its cost;
  * what doing nothing means.

  No workaround is offered. The chosen resolution is recorded as a dated deviation entry here, and in the MADR when a
  decision or a fact changes, before work continues.
* **C9. Seen failing.** Every new test or case is run against a broken copy (C2) that undoes the phase's change, and is
  seen to fail with the expected message. The execution record quotes that failure.
* **C10. Platform tests.**
  * A test that needs a FIFO or a symlink skips, with its reason, where the platform cannot make one: FIFOs off Unix,
    symlinks on Windows without the privilege.
  * A test of a case-insensitive volume runs where the temp directory is one. On macOS's default volume it runs; it is
    skipped elsewhere, with the reason.

## Implementation Steps

### P0 — the start: the records, and P17 to its branch (MADR Q6 (a))

Files:

* `docs/decisions/0037-MADR-hardening-integrity-and-performance.md`;
* `docs/decisions/0037-PLAN-permission-and-file-tool-hardening.md`;
* `docs/README.md`;
* `AGENTS.md`.

Each commit below is made on the owner's word in its turn.

1. **The records on `main`.** Commit the four files by pathspec:

   ```text
   git commit --no-edit -- AGENTS.md docs/README.md docs/decisions/0037-MADR-hardening-integrity-and-performance.md docs/decisions/0037-PLAN-permission-and-file-tool-hardening.md
   ```

   0035's P17 change set stays out of the commit, its ten deletions still staged.
2. **P17 to `feature/p17-lineedit`.**
   * Before the commit:
     * `make preflight`, with P17 in the tree, passes;
     * `git status --short` lists exactly 0035's P17 record: its 18 modified files, its ten staged deletions, and
       `internal/cli/editor/parity_test.go`.
   * Then:
     * `git switch -c feature/p17-lineedit`;
     * `git add -A -- internal/cli`;
     * `git commit --no-edit`.
3. **Back to `main`.**
   * `git switch main`.
   * `git status --short` is empty, and `HEAD` is step 1's commit.
   * `go test -count=1 ./internal/cli/...` passes on `main`'s editor.
4. **The hooks.** `git rev-parse --path-format=absolute --git-path hooks` is the global hooks directory.

*Acceptance:* `main`'s tree is clean, and `feature/p17-lineedit` holds P17 whole, one commit after step 1's.

### P1 — `mv` and `cp` into a directory (MADR D1; F1)

Files:

* `tool/tool.go`: `Env.Rerouted`;
* `agent/agent.go`: sets it;
* `tool/builtin/fileops.go`;
* `tool/builtin/fileops_test.go`;
* `agent/agent_test.go`;
* `docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md`, for a dated note.

1. **The tool resolves a directory destination, for a call the shell made.**
   * `tool.Reroute` gets only the call (`tool/tool.go:286-288`), so `interceptShell` cannot test the disk. The agent
     already knows a rerouted call's first tool (`agent/agent.go:779-782`).
   * `tool.Env` gains `Rerouted string`, the name of the tool a call was rerouted from, `json:"-"`. `agent` sets it
     before the permission check, so the call's `Subject` and its `Run` see the same `Env`.
   * A new helper in `fileops.go`, `destination(ws, env, from, to)`, returns `to/<base(from)>` when `env.Rerouted` is
     `bash` and `to` is an existing directory, and `to` otherwise. It `Stat`s `to`, following a symlink as the shell
     does.
   * The tools' permission subject, now `pathSubject("from", "to")` (`fileops.go:53`, `:89`), becomes a
     move-and-copy subject that resolves `to` through it. Their outside check, `twoPaths` (`:113-116`), `runMove`
     and `runCopy` use it too. The permission rules therefore see the path that is written.
   * A call the model makes to `move` or `copy` keeps today's meaning: `to` is the new path.
2. **`makeRoom` never removes a directory for a non-directory source.**
   * When the destination exists, is a directory, and the source is not a directory, `makeRoom` refuses with
     "`<to>` is a directory; a file cannot replace it". It does so whether or not `overwrite` is set.
   * A directory replacing a directory is unchanged.
   * `makeRoom` gains the source's `fs.FileInfo` as a parameter.
3. **Tests.**
   * `TestInterceptIntoDirectory`: `mv -f f.txt d`, `mv f.txt d`, `cp -f f.txt d` and `cp -r src d`, with `d` an
     existing directory holding `precious.txt`:
     * the file or directory ends inside `d`;
     * `d/precious.txt` is unchanged;
     * the result names `d/f.txt`.
   * `TestMakeRoomKeepsDirectory`: the native `move` and `copy`, called by the model with `overwrite: true`, a file
     source and a directory destination, are refused with the message, and the directory is intact.
   * `TestReroutedEnv` (`agent`): a rerouted call's `Subject` and `Run` both see `Env.Rerouted` as `bash`. A direct
     call sees it empty.
   * The existing interception cases pass unchanged.
4. **Seen failing (C9):**
   * on a copy without step 1, `TestInterceptIntoDirectory` fails, with `d/precious.txt` missing;
   * on a copy without step 2, `TestMakeRoomKeepsDirectory` fails.
5. **0012-MADR D10** gets a dated note: an intercepted `mv` or `cp` resolves an existing directory destination as the
   shell does (0037-MADR D1).
6. **`apidiff`:** `tool.Env` gains one field. A finding is a deviation (C8).
7. G-local and G-hosts. **Commit.**

### P2 — paths compared as the filesystem resolves them (MADR D2, D5; F2, F4)

Files:

* `internal/fsx/workspace.go`: export `RealPath`, add `SamePath` and `Under`;
* `internal/fsx/workspace_test.go`;
* `permission/rules.go`;
* `permission/store.go`;
* `permission/trust.go`;
* `permission/rules_test.go`;
* `permission/store_test.go`.

1. **`fsx` gains the comparison.**
   * `realPath` becomes `RealPath`, unchanged.
   * `fsx.Fold(p)` is `p` compared as the platform's default volume compares names. On darwin and windows that is
     Unicode case folding after NFC normalisation, because APFS ignores both. Elsewhere it is unchanged. On windows, each
     element's trailing dots and spaces are stripped.
   * `fsx.SamePath(a, b)` and `fsx.Under(p, dir)` compare cleaned paths through `Fold`.
   * `relativeFold` keeps serving Windows confinement, unchanged.
2. **The protected and config paths.** `protected` and `configPath` (`permission/rules.go:292-322`) test each subject
   twice:
   * as given, and through `fsx.RealPath`;
   * by element through `Fold`: `.git` in any element, `AGENTS.md` as the base name, and `under` through `fsx.Under`.

   Either reading's match is the verdict.
3. **Path rules.**
   * `patternMatches` (`permission/rules.go:218-230`) matches a path subject's relative form for both its lexical and
     its real path.
   * On darwin and windows it matches with the pattern and the subject both folded. A path deny then holds for every
     spelling.
   * `Evaluate` takes the stricter verdict of the readings, as it does across subjects today.
4. **The project store's lookup** (`permission/store.go:93`, `:154`) compares keys through `fsx.SamePath`.
5. **Trust stays exact.** `samePath` in `trust.go` is kept as it is, and its comment says why: folding a grant would
   widen it (MADR D5). `trust.go:264` is unchanged.
6. **Tests.**
   * `TestProtectedFoldsCase`, under `allow: write`:
     * `.GIT/config`, `.Git/config` and `docs/../.GIT/config` are denied (`limitProtected`);
     * `Agents.md` and `.Gobble/settings.yaml` ask (`limitConfig`).
   * `TestProtectedRealPath`: with `docs/x -> ../.git`, `docs/x/config` is denied.
   * `TestPathRulesFoldCase`:
     * with `deny: read(.env)`, `read /r/.ENV` is denied on darwin and windows;
     * with `deny: edit(Secrets/**)`, `edit /r/secrets/k` is denied on darwin and windows, and allowed on linux, where it
       is another file.
   * `TestPathRulesRealPath`: `deny: read(secret/**)`, and a symlink `link -> secret`: `read link/k` is denied.
   * `TestStoreLookupFoldsCase`: a "reject always" saved under `/w/Proj` loads for `/w/proj` on darwin and windows.
   * `TestTrustStaysExact`: a trust grant for `/w/Proj` is not found for `/w/proj`.
   * On Windows, `TestFoldWindows`: `.git.` and `.git ` fold to `.git`.
7. **Seen failing (C9).** On a copy whose `protected` keeps the exact `.git` compare, `TestProtectedFoldsCase` fails.
   On a copy without the real-path reading, `TestProtectedRealPath` and `TestPathRulesRealPath` fail.
8. G-local and G-hosts. **Commit.**

### P3 — bash's literal write targets meet the protected-path limit (MADR D3; F2)

Files:

* `tool/tool.go`: `Subject.Writes`;
* `tool/builtin/shellparse.go`;
* `tool/builtin/shell.go`;
* `tool/builtin/shellparse_test.go`;
* `permission/permission.go`: `Rule.Writes`, `json:"writes,omitzero"`;
* `permission/rules.go`;
* `permission/rules_test.go`;
* `agent/agent.go`, `approve`, to carry `subject.Writes` into the rule;
* `docs/architecture.md`.

1. **The parser names write targets.** `parseShell` collects `sc.writes`, beside `sc.paths`:
   * every literal output redirection's file: `>`, `>>`, `>|`, `&>`, `&>>` and `<>`, which are `RdrOut`, `AppOut`,
     `RdrClob`, `RdrAll`, `AppAll` and `RdrInOut`, sinks excluded as `redirectPath` excludes them;
   * every literal file operand of the writing programs among `fileProgram`: `tee`, `truncate`, `rm`, `rmdir`, `mv`,
     `cp`, `mkdir`, `touch`, `ln`, `chmod` and `chown`;
   * the file operands of `sed`, `perl` and `ruby` when an in-place flag is among the options: `-i`, `-i<suffix>` or
     `--in-place`.

   A target computed at run time is not literal, and is not collected.
2. **The tool resolves them.** `bashSubject` returns `Writes`, each target resolved against the call's workdir as
   `bashOutside` resolves its paths: `fsx.Resolve`, with `~` at the home directory.
3. **The rule carries them.** `approve` (`agent/agent.go:842-851`) copies `subject.Writes` into `rule.Writes`.
4. **The limit.** `limits` (`permission/rules.go:268-288`) denies any rule whose `Writes` holds a protected path, with
   `limitProtected`, whatever its tool and its verdict. It asks for a config path, with `limitConfig`, unless the
   verdict is deny. Both use P2's readings.
5. **`docs/architecture.md`** says in the permission section:
   * bash's protected-path check covers the literal targets above, and no others;
   * pipelines into `xargs`, `find -delete`, and paths built at run time are beyond it.
6. **Tests.**
   * `TestShellWrites`: each redirection operator and each writing program, quoted and unquoted operands, `sed -i` and
     `sed` without it, a sink, `$VAR` and a command substitution, each giving its expected `Writes`.
   * `TestBashWritesProtected`: under `allow: bash(echo *)`, `echo x > .git/config`, `echo x >> .GIT/config`,
     `tee .git/hooks/pre-commit` and `sed -i s/a/b/ .git/config` are denied (`limitProtected`). `echo x > out.txt` is
     allowed.
   * `TestBashWritesConfig`: `echo x > AGENTS.md`, under `allow: bash(echo *)`, asks.
7. **Seen failing (C9).** On a copy whose `limits` ignores `Writes`, `TestBashWritesProtected` fails.
8. **`apidiff`:** `tool.Subject` and `permission.Rule` gain one field each, as `tool.Env` did at P1. A finding is a
   deviation (C8).
9. G-local and G-hosts. **Commit.**

### P4 — commands matched in their raw and their normalised form (MADR D4; F3)

Files:

* `tool/builtin/shellparse.go`;
* `tool/builtin/shellparse_test.go`;
* `permission/rules.go`;
* `permission/rules_test.go`;
* `docs/architecture.md`.

1. **A normalised subject.** For each simple command, `call` (`tool/builtin/shellparse.go:99`) appends a second subject
   after the raw one, with an empty save (`""`), when it differs from the raw one. The normalised subject is:
   * the command's words as `wordValue` reads them, unquoted and with escapes removed, joined by single spaces;
   * the program word reduced to its base name when it is an absolute or relative path;
   * on darwin and windows, the program word lower-cased.

   The raw subject, and its save, are unchanged, so "always" answers save what they save today.
2. **Matching.** `wildcard` (`permission/rules.go:113-137`) folds case on darwin as it does on windows. Every subject's
   verdict is already combined by strictness (`Evaluate`, `:178-183`), so a deny on either form denies.
3. **Tests.** `TestCommandBypasses`, under `allow: bash` then `deny: bash(rm *)`:
   * `\rm -rf build`, `"rm" -rf build`, `r''m -rf build`, `rm<TAB>-rf build` and `/bin/rm -rf build` are denied;
   * on darwin and windows, `RM -rf build` is denied;
   * the denied rows of MADR F3's table stay denied;
   * `echo build | xargs rm -rf` and `find build -delete` are not denied, and the test says so with the docs' reason.
4. **`docs/architecture.md`:** deny rules for commands see the program and its literal words. `xargs`, `find -delete`
   and commands assembled at run time are beyond them.
5. **Seen failing (C9).** On a copy without the normalised subject, `TestCommandBypasses` fails on `\rm`.
6. G-local and G-hosts. **Commit.**

### P5 — writes through symlinks and hard links (MADR D6, Q1 (a); F5)

Files:

* `internal/fsx/write.go`;
* `internal/fsx/write_test.go`;
* `internal/fsx/write_unix.go` and `internal/fsx/write_windows.go`, if the directory sync and the link count need
  per-platform code.

1. **A symlink is followed inside the roots.**
   * `Workspace.WriteFile` `Lstat`s the path. When it is a symlink, the target is resolved with `fsx.RealPath`, and must
     be inside a root (`ErrOutside` otherwise, as `os.Root` reports it).
   * The atomic write then happens beside the target, and the link is untouched.
   * `WriteFileMode`, for files outside a workspace, follows the link the same way, with no root.
2. **A file with more than one hard link is written in place.** When the target's link count is above one, the write
   opens it with `O_WRONLY|O_TRUNC`, writes, syncs and closes. The file's mode and every link are kept.
   * The count is `Stat_t.Nlink` on Unix, and `GetFileInformationByHandle`'s `NumberOfLinks` on Windows.
3. **The directory is synced.** On Unix, after the rename, the parent directory is opened and synced. A sync error is
   returned, as a write's is.
4. **Tests.**
   * `TestWriteThroughSymlink`: with `CLAUDE.md -> AGENTS.md`, `WriteFile(CLAUDE.md)` leaves `CLAUDE.md` a symlink and
     `AGENTS.md` with the new bytes.
   * `TestWriteSymlinkOutsideRoot`: a link to a path outside the roots is refused with `ErrOutside`.
   * `TestWriteHardLinkInPlace`: with `a` and `b` linked, writing `a` leaves `b` with the new bytes, and the link count
     at 2.
   * `TestWriteNewFileAtomic`: a single-link file is still replaced by rename, its inode changing. This is tested on Unix.
5. **Seen failing (C9).** On a copy without step 1, `TestWriteThroughSymlink` fails with `CLAUDE.md` a regular file. On
   a copy without step 2, `TestWriteHardLinkInPlace` fails.
6. G-local and G-hosts. **Commit.**

### P6 — the file tools read only regular files, and only what they need (MADR D7; F6)

Files:

* `internal/fsx/workspace.go`: `OpenRegular`, `ReadRegular`;
* `internal/fsx/walk.go`: `Walker.ReadRegular`;
* `internal/fsx/ops.go`: `CopyFile`, streaming;
* `internal/fsx/write.go`: `writeAtomicFrom`, an `io.Reader` form of `writeAtomic`;
* `internal/fsx/nonblock_unix.go` and `internal/fsx/nonblock_other.go`: the open flag;
* `internal/fsx/regular.go`: `ReadRegularFile`, for files outside a workspace (P7);
* `internal/fsx/*_test.go`;
* `tool/builtin/read.go`;
* `tool/builtin/builtin.go`, `textIO.read`;
* `tool/builtin/grep.go`, `candidate`;
* `tool/builtin/read_test.go`;
* `tool/builtin/grep_test.go`.

1. **Opening.** On Unix, `fsx` opens a file for reading with `O_RDONLY|O_NONBLOCK`, so a FIFO's open returns at once,
   then `Stat`s the open file. On other platforms the flag is 0.
   * `OpenRegular` returns `ErrNotRegular` for anything that is not a regular file, and closes it.
   * `ReadRegular(abs, max)` reads at most `max` bytes, and returns `ErrTooLarge` past it.
   * The two sentinels wrap `fs.ErrInvalid`.
2. **`read`.**
   * Local reads go through `OpenRegular` and stream: lines are read with a `bufio.Reader` up to `offset + limit`, and
     the remaining newlines are counted in 64 KiB chunks for the total.
   * A read through the client's `ReadTextFile` first checks `OpenRegular` locally, then reads as today. The client
     returns the whole text.
   * An image read keeps its whole-file read, through `ReadRegular` with a 20 MiB limit.
   * `numbered`'s output is unchanged for every existing test.
3. **`grep`.** `candidate` (`tool/builtin/grep.go:321-333`) reads through `Walker.ReadRegular(rel, maxGrepFileBytes)`.
   A non-regular file is skipped silently, as an unreadable one is today.
4. **`copy`.** `CopyFile` opens the source with `OpenRegular`, and streams it into `writeAtomicFrom`. The atomic rename,
   the sync and the mode are as `writeAtomic`'s.
5. **Tests.**
   * `TestReadRefusesFIFO`, Unix: `read` of a FIFO returns an error naming it "not a regular file" within 100 ms. The
     test has a 2 s deadline.
   * `TestGrepSkipsFIFO`, Unix: `grep` over a directory holding a FIFO and a file returns the file's match within 1 s.
   * `TestReadStreamsLimit`: an 8 MiB file read with `limit: 50` allocates under 2 MiB, measured with `runtime.MemStats`
     `TotalAlloc`. Its total-lines line is right.
   * `TestCopyFileStreams`: a 16 MiB copy allocates under 2 MiB, and its bytes and mode match.
   * `TestCopyRefusesFIFO`, Unix.
6. **Seen failing (C9).** On a copy without step 1's flag and check, `TestReadRefusesFIFO` fails on its deadline. On a
   copy with today's whole-file read, `TestReadStreamsLimit` fails on its allocation bound.
7. G-local and G-hosts. **Commit.**

### P7 — configuration files read as regular files, within 1 MiB (MADR D10; F9)

Files:

* `internal/config/resolve.go:131`;
* `internal/config/update.go:46`;
* `internal/config/edit.go:75`;
* `internal/config/rulesedit.go:90`;
* `mcpclient/config.go:83`;
* `mcpclient/file.go:163`;
* `permission/store.go:64`;
* `permission/trust.go:142`;
* `internal/cli/modelsjson.go:52`;
* `internal/auth/store.go:65`;
* the tests beside each.

1. **One reader.** Each `os.ReadFile` above becomes `fsx.ReadRegularFile(path, fsx.ConfigLimit)`, where `ConfigLimit` is
   1 MiB (P6's `regular.go`).
   * A missing file still reads as `fs.ErrNotExist`, so each caller's "missing is empty" path is unchanged.
   * `ErrNotRegular` and `ErrTooLarge` become the caller's existing error for an unreadable file: a `Diagnostic` in
     `internal/config`, and the error each other caller already returns. Each names the file and the reason.
2. **The project layer before trust** (`internal/config/resolve.go:101-103`) gets the same reader, so a FIFO or a link to
   `/dev/zero` cannot block or exhaust `Resolve`.
3. **Tests.**
   * `TestResolveRefusesFIFO`, Unix: `Resolve` with `.gobble/settings.yaml` a FIFO returns within 1 s, with a diagnostic
     naming "not a regular file".
   * `TestResolveRefusesLarge`: a 2 MiB `settings.yaml` gives a diagnostic naming the 1 MiB limit, and allocates under
     4 MiB.
   * `TestResolveDevZero`, Unix: `.gobble/settings.yaml -> /dev/zero` gives "not a regular file".
   * One case per other reader: a FIFO is refused, and a missing file is as before.
4. **Seen failing (C9).** On a copy whose `readLayer` keeps `os.ReadFile`, `TestResolveRefusesFIFO` fails on its
   deadline.
5. This phase touches `internal/cli`, so G-local adds `make perf`, and G-hosts add `make perf` and `make probe-conhost`.
   **Commit.**

### P8 — edits: overlapping matches, and the line endings left alone (MADR D8; F7)

Files:

* `tool/builtin/editmatch.go`;
* `tool/builtin/edit.go`;
* `tool/builtin/editmatch_test.go`;
* `tool/builtin/edit_test.go`.

1. **Uniqueness.**
   * In `locate` (`editmatch.go:186-197`), a count of one, in either layer, is accepted only if no second occurrence
     starts inside the first: `strings.Index(base[at+1:], old)` finds none before `at+len(old)`.
   * When one does, the error is `duplicate(i, 2)`.
   * A count above one keeps today's number in the message.
2. **Line endings.**
   * A file with no `\r` is edited as it is, with no conversion.
   * Otherwise, the view converts only `\r\n` to `\n`, so a lone `\r` stays a character, and each line of the view is a
     line of the file.
   * `applyEdits` returns its matched spans with its result. `edit.go` rebuilds the file line by line:
     * a line no span touches is copied from the original, its own ending kept;
     * a line a span touches takes the replacement, its endings in the file's dominant ending (`lineEnding`).
   * `toLF` and `restoreEnding` are removed if nothing else uses them.
3. **Tests.**
   * `TestEditOverlappingDuplicate`: in `x\n}\n}\n}\n`, the old text `}\n}\n` is refused as a duplicate (2), and the
     file is unchanged.
   * `TestEditKeepsUntouchedEndings`: in `a\r\nb\nc\n`, editing `b` to `B` gives `a\r\nB\r\nc\n`. The edited line takes
     the dominant ending, CRLF, from the first newline, and the other lines keep theirs.
   * `TestEditLoneCR`: `b\rstill b\n`, with `b` edited elsewhere in the file, keeps its `\r`.
   * `TestEditLFOnly`: a file with no `\r` is byte-identical outside the edit.
   * The existing edit tests pass unchanged. A test whose expectation depended on whole-file conversion is a deviation
     (C8), and its old and new expectations are recorded.
4. **Seen failing (C9).** On a copy with `strings.Count` alone, `TestEditOverlappingDuplicate` fails. On a copy with
   `toLF` and `restoreEnding`, `TestEditKeepsUntouchedEndings` fails.
5. G-local and G-hosts. **Commit.**

### P9 — the spill file's cap (MADR D9; F8)

Files:

* `tool/builtin/output.go`;
* `tool/builtin/output_test.go`.

1. **The cap.**
   * `spillCap` is a package variable, `256 << 20`.
   * Past it, `spill.Write` stops writing to the file, and sets `cut`. It still counts bytes and lines, and still keeps
     the tail in memory.
2. **The result.** Where the result names the spill file, it adds "the file holds the first 256 MiB" when `cut` is set.
3. **Tests.** `TestSpillCap`, with `spillCap` set to 1 MiB for the test: 4 MiB of output leaves the file at 1 MiB, the
   counts at 4 MiB, the tail intact, and the result's note present.
4. **Seen failing (C9).** On a copy without the cap, `TestSpillCap` fails on the file's size.
5. G-local and G-hosts. **Commit.**

### P10 — linear globs, and recursive operations that compare real paths and stop on cancel (MADR D11; F10, F11)

Files:

* `internal/fsx/glob.go`;
* `internal/fsx/glob_test.go`;
* `internal/fsx/ops.go`: `IsRoot`, `Inside`;
* `internal/fsx/ops_test.go`;
* `tool/builtin/fileops.go`: `copyEntry`, `countEntries` and their callers, with a context;
* `tool/builtin/fileops_test.go`.

1. **`**` collapses.** `compile` (`glob.go:67-94`) collapses consecutive `**` segments into one.
2. **Memoised matching.**
   * `matchSegs` becomes an iterative or memoised match over (pattern index, name index), with a table of
     `len(pat)+1` × `len(name)+1` entries allocated once per match.
   * The results equal today's on every existing glob and ignore test.
3. **Real paths.** `IsRoot` and `Inside` compare `RealPath` of both sides, through P2's `Fold`.
4. **Cancel.**
   * `runMove`, `runDelete` and `runCopy` take their context, now discarded as `_`.
   * `copyEntry` and `countEntries` check it before each entry, and return `context.Cause`.
   * `RemoveAll` runs uninterrupted once started (MADR D11, as noted).
5. **Tests.**
   * `TestGlobLinear`: `**/**/**/**/**/**/**/**/**/z` against 31 segments, and `**/a/` six times then `z` against 41,
     each matches as before, within 50 ms.
   * `TestGlobMatchesUnchanged`: a table of today's glob cases, unchanged.
   * `TestCopyIntoItselfRealPath`: copying `ws/a` to `RealPath(ws)/a/b` is refused, "cannot be copied into itself".
   * `TestCopyHonoursCancel`: a copy of a 2,000-entry tree with a cancelled context returns `context.Canceled` and
     copies fewer entries.
6. **Seen failing (C9).** On a copy with today's `matchSegs`, `TestGlobLinear` fails on its bound. On a copy with
   lexical `Inside`, `TestCopyIntoItselfRealPath` fails with its 5 s test deadline.
7. G-local and G-hosts. **Commit.**

### P11 — close-out

Files:

* this plan;
* the MADR, if a fact changed;
* `docs/README.md`;
* `docs/architecture.md`, if P3's and P4's notes need one more line.

1. Every acceptance criterion in "Goal" is checked against the phases' records. A criterion not met is named, with its
   reason.
2. This plan's status becomes `complete`. The MADR stays `accepted`, with plans 2 and 3 to follow.
3. The disclosure guard runs over the outgoing commits, and its output is recorded (`AGENTS.md`, "Identifiers").
   Nothing is pushed.
4. `make check-records markdownlint`. **Commit.**

## Verification

Each phase's own steps are its acceptance criteria. Across the plan:

* every test named in P1–P10 exists, passes on all three hosts, and has its quoted failure on a broken copy (C9);
* `acpserver/testdata/session.golden`, and the CLI's goldens, are byte-identical to P0's commit:
  `git diff --stat <P0>..HEAD -- '*.golden'` is empty;
* `make preflight`, `make race` on all three hosts, and `make perf` and `make probe-conhost` where `internal/cli`
  changed (P7);
* `apidiff`'s report shows only the added fields: `tool.Env.Rerouted` (P1), `tool.Subject.Writes` and
  `permission.Rule.Writes` (P3);
* MADR F1–F11's probes, rebuilt as the tests above, now pass.

## Rollout and Rollback

* Each phase is one commit, and reverts on its own with `git revert`, except where a later phase builds on it:
  * P3, P4 and P10 use P2's `Fold`, `SamePath` and `RealPath`;
  * P7 uses P6's `ReadRegularFile`.

  Reverting P2 reverts P3, P4 and P10 first. Reverting P6 reverts P7 first.
* Nothing is pushed by this plan. The owner pushes, after the disclosure guard.
* **What users see change:**
  * P1: `mv` and `cp` into a directory work as in a shell, and the native tools refuse to replace a directory with a
    file;
  * P2, P3 and P4: a call that used to get past a deny rule or the protected paths is denied, or asked about;
  * P6 and P7: a FIFO or a device is refused by name, and a configuration file over 1 MiB is refused;
  * P8: an ambiguous overlapping edit is refused, and an edit no longer rewrites other lines' endings;
  * P9: the spill file stops at 256 MiB.
* **P17** stays on `feature/p17-lineedit` until go-tui-lib#13 ships. It is then rebased onto `main` under 0035-PLAN.

## Execution record

Approved by the owner on 2026-10-30: "proceed".

The owner's answers, from the MADR, 2026-10-30:

* Q1 (a), "Write in place", in P5;
* Q6 (a), "P17 to a branch", in P0.

### P0 record (2026-10-30): the records, and P17 to its branch

* **Start.** `main` at `61f7115`. The hooks are the global directory, and Go is `go1.27.3`. The tree held 0035's P17
  whole (18 modified files, 10 staged deletions, `internal/cli/editor/parity_test.go`) and the four record files.
  `make preflight` passed with P17 in the tree.
* **The commits were run by the owner.** The permission check refused `git commit` from the agent.
  1. The two new records were untracked, so a first `git commit -- <paths>` failed with "pathspec … did not match any
     file(s) known to git". They were added with `git add`, and the pathspec commit made `2a8d70c` on `main`, the four
     files alone.
  2. `git switch -c feature/p17-lineedit`, then `git add -A -- internal/cli` (18 modified, 10 deleted, 1 added), then
     `git commit --no-edit` made `8630b77`.
  3. `git switch main`.
* **Acceptance:**
  * `main` is at `2a8d70c`, with an empty `git status --short`;
  * `git log main..feature/p17-lineedit` is `8630b77` alone;
  * `go test -count=1 ./internal/cli/...` passes on `main`'s editor.

### P1 record (2026-10-30): `mv` and `cp` into a directory

The base is P0's `2a8d70c`.

1. **`tool.Env.Rerouted`** (`tool/tool.go`), `json:"-"`, is documented in `Env`'s comment. `agent`'s `runTool` sets it
   from `rerouted` before `Outside`, `Subject` and `Run` (`agent/agent.go`).
2. **`destination`, `fromTo` and `shownTo`** (`tool/builtin/fileops.go`).
   * `destination(env, from, to)` gives `to/<base(from)>` when `env.Rerouted` is `bash` and `to` is an existing
     directory (`Workspace.Stat`, following a link), and `to` otherwise.
   * `move` and `copy` take `fromTo` as their subject in place of `pathSubject("from", "to")`. `twoPaths`, `runMove` and
     `runCopy` use `destination` too.
   * The result names the real destination, for example "moved f.txt to d/f.txt".
   * Their `Locations` stay `argLocations("from", "to")`. `acpserver` computes them from the session's own environment
     (`acpserver/prompt.go:104`), with no `Rerouted`, as a display hint at the call's start. The result line names the
     path written.
3. **`makeRoom`** takes the source's `fs.FileInfo`. It refuses "`<to>` is a directory; a file cannot replace it" for a
   source that is not a directory and a destination that is one, whether or not `overwrite` is set. A directory still
   replaces a directory under `overwrite`.
4. **Tests.**
   * `TestInterceptIntoDirectory`: `mv -f f.txt d`, `mv f.txt d`, `cp -f f.txt d` and `cp -r src d`.
   * `TestMakeRoomKeepsDirectory`: move and copy, `overwrite` false and true.
   * `TestReroutedEnv` (`agent`), end to end through bash:
     * the rule's subjects are `f.txt` and `d/f.txt`;
     * the file lands in `d`;
     * the model's own `move` to `d` keeps `d` as its subject, and does not run.
   * The existing `TestMove`, `TestCopy` and `TestShellReroute` pass unchanged.
5. **Seen failing (C9),** each on its own scratch copy:
   * `destination` always `to`: `TestInterceptIntoDirectory/mv_-f_f.txt_d` fails, `result (isError true): move: d is a
     directory; a file cannot replace it`, want `moved f.txt to d/f.txt`;
   * `makeRoom`'s guard removed: `TestMakeRoomKeepsDirectory` fails, `move: d already exists; set overwrite to replace
     it`, want an error containing `move: d is a directory; a file cannot replace it`;
   * `agent` not setting `Rerouted`: `TestReroutedEnv` fails, the rule's subjects naming `d`, not `d/f.txt`;
   * both steps undone, F1 as found: `TestInterceptIntoDirectory/mv_-f_f.txt_d` fails, `result (isError false): moved
     f.txt to d`. That is the result that removed `d`.
6. **0012-MADR D10** has a dated note.
7. **`apidiff`.** `make preflight`'s step reports "apidiff: no-op below v1.0.0". The module is below v1.0.0, so P1's and
   P3's `apidiff` steps cannot find anything until it is not. This is recorded, not a deviation: no finding was made.
8. **G-local:**
   * `make pre-add-check` on the 5 Go files: clean;
   * `make preflight`: passed;
   * `make race`: 36 ok.
9. **G-hosts:** on the Linux machine and the Windows laptop, `go test` passed, race passed with 36 ok, and `make check-records markdownlint` passed. `make preflight` stopped at `verify-build-metadata`, which needs a commit, as on every copy..
