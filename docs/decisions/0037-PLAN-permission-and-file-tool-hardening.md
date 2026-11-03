---
status: complete
date: 2026-11-03
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
   *(Deviation 3: the normalised subject's save is the raw subject's save, not `""`, so that `Remember` makes the same
   rule twice and keeps it once; it is added only where that save is not empty.)*
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
   * *(Deviation 5: `RealPath` cannot resolve a dangling link, so such a link is read with `os.Readlink`, one hop at a
     time, up to 40 hops.)*
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
   * *(Deviation 6: `grep` naming one file reads it through `ReadRegular` too, and a file that is not regular is an
     error there, not a skip. `textIO.read`'s local path, which `edit` and `write` share, opens through
     `OpenRegular`.)*
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
* by deviation 7: `internal/cli/config.go:365` and `internal/cli/completion.go:143`;
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
* `tool/builtin/edit_test.go`;
* by deviation 8: `tool/builtin/editstrategy.go` and `tool/builtin/fuzz_test.go`.

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

### P8a — the edit lock's waits, jittered (deviation 9; 0018-MADR D18)

Added by deviation 9. P8 is held, uncommitted, until this phase passes; the two land in one commit.

Files:

* `internal/filelock/lock.go`;
* `internal/filelock/lock_test.go`.

1. **Jitter.** In `Lock`, each wait before the last is drawn uniformly from half to all of the policy's scheduled gap
   (`d/2 + rand.N(d/2+1)`, `math/rand/v2`). The wait cut to end at the deadline is not jittered, so the last try is
   still at the deadline and the `*HeldError` comes no sooner (0018-PLAN P7, deviation 24). `EditPolicy` is unchanged:
   20 ms, doubling to 100 ms, for 2 s.
2. **Tests.**
   * ~~`TestLockWaitersSpread`: 16 goroutines each take the same lock with `EditPolicy`, hold it 1 ms and release it;
     all are done within 1 s. In lock-step the waits alone come to about 20 + 40 + 80 + 12 × 100 ms, 1.34 s.~~
     *(2026-11-03, deviation 10: replaced by `TestLockJitter`, a test of the draw itself; lock-step measured 0.88–1.37 s,
     so the bound passed the broken code in 2 runs of 3.)*
   * `TestLockDeadline`, `TestLockOutlastsPiBudget` and the other existing lock tests pass unchanged.
   * `TestTrustConcurrentSaves` is unchanged. On the Windows laptop it runs 20 times under race, and the durations are
     recorded.
3. **Seen failing (C9).** On a copy without the jitter, ~~`TestLockWaitersSpread` fails on its 1 s bound~~
   *(deviation 10: `TestLockJitter` fails)*.
4. G-local and G-hosts over P8 and P8a together; on the Windows laptop the full race suite runs three times. **Commit**
   P8 and P8a together, with both records.

### P9 — the spill file's cap (MADR D9; F8)

Files:

* `tool/builtin/output.go`;
* `tool/builtin/output_test.go`;
* by deviation 11: `tool/builtin/shell.go` and `docs/architecture.md`.

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
* `tool/builtin/fileops_test.go`;
* by deviation 12: the MADR, for D11's note.

1. **`**` collapses.** `compile` (`glob.go:67-94`) collapses consecutive `**` segments into one.
2. **Memoised matching.**
   * `matchSegs` becomes an iterative or memoised match over (pattern index, name index), with a table of
     `len(pat)+1` × `len(name)+1` entries allocated once per match.
   * The results equal today's on every existing glob and ignore test.
3. **Real paths.** ~~`IsRoot` and `Inside` compare `RealPath` of both sides, through P2's `Fold`.~~
   *(2026-11-03, deviation 12: the path an operation acts on is compared as `RealPath` of its directory and its
   own name, the last link not followed; a root is `RealPath` whole; today's lexical check stays too.)*
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
   * `TestRealPathKeepsLinks` (deviation 12): a symlink to a directory moves into that directory, and a symlink to
     the workspace root is deleted, as today.
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
  * P10: a copy or a move into itself by another spelling of the path is refused, and a copy or a recursive
    delete stops on cancel *(added at P11: the list stopped at P9)*.
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

### Deviation 1 (2026-10-30): P2's readings, and the default

* **Found.** P2 matched each path subject as given and as resolved, and kept the stricter verdict, as MADR D5 and P2
  step 3 say. Two existing tests then failed. Both pass at `e13d154`.
  * `permission/rules_test.go:135`, `TestEvaluateSubjects`: `allow: write(/tmp/**)` for `/tmp/scratch/out.txt` gave
    `{Effect:ask By:ask: write}`.
  * `permission/store_test.go:74`, `TestRemember`, "an outside path, allowed": a saved allow gave `ask`.

  On macOS `/tmp` resolves to `/private/tmp`, and `/var` to `/private/var`. The resolved reading matched no rule, so it
  took the default, ask, and ask is stricter than the allow the given reading matched. Every absolute allow rule for a
  path with a symlink in it would have asked.
* **The owner's answer, 2026-10-30:** "Defaults don't outrank (Recommended)". The options not taken were:
  * consulting the resolved reading for deny and ask rules only;
  * keeping D5 literal, and changing the two tests.
* **Decision.**
  * `Evaluate` takes, per subject, `matchReadings`: the strictest verdict among the readings a rule matched, or the
    default's when no rule matched any.
  * Across subjects the strictest wins, as before.
  * MADR D5 has a dated amendment.
  * Both tests pass unchanged.
* **Files added to scope:** none.

### Deviation 2 (2026-10-30): P2's trust test on windows

* **Found.** On the Windows laptop, `go test` and `make race` failed one test, `TestTrustStaysExact`:
  `store_test.go:242: Lookup(…\proj) found the grant for …\Proj: ok true, err <nil>`. macOS and the Linux machine
  passed.
  * This is not a regression. Trust's `samePath` (`permission/trust.go:282-283`) has compared with `strings.EqualFold`
    on windows since before 0037, as NTFS ignores case. P2 did not change it.
  * MADR D5's "trust grants stay exact" meant unchanged. Step 6, and the test as written, asserted "not found" on every
    platform.
* **The owner's answer, 2026-10-30:** "Test follows trust's rule (Recommended)". The options not taken were:
  * making trust compare case-sensitively on windows;
  * folding trust everywhere, which reverses D5 on a case-sensitive macOS volume.
* **Decision.**
  * `TestTrustStaysExact` asserts trust as it compares: not found on darwin, found on windows.
  * No code changes.
  * MADR D5 has a dated note.
* **Files added to scope:** none.

### P2 record (2026-10-30): paths compared as the filesystem resolves them

The base is P1's `e13d154`.

1. **`fsx`** (`internal/fsx/workspace.go`).
   * `realPath` is exported as `RealPath`, unchanged, at its five calls.
   * `Fold` is new. On darwin and windows it applies NFC, then `cases.Fold()`; on windows each element also drops its
     trailing dots and spaces. Elsewhere it returns the path unchanged.
     `golang.org/x/text` was already required, so `go.mod` is unchanged.
   * `SamePath` and `UnderDir` compare through `Fold`. *The plan named the second `Under`. revive's confusing-naming
     refused it beside `ignore.go`'s unexported `under`, a file outside P2, so the new function took the name
     `UnderDir`.*
2. **`permission/rules.go`.**
   * `readings(p)`: `p` cleaned, and `fsx.RealPath(p)` when that differs.
   * `protected` and `configPath` test every reading. They test `.git` among `folded(q)`'s elements, `AGENTS.md` by
     `Fold` of the base name, and the anchors through `under` and `samePlace`, which take each anchor as given and as
     resolved.
   * `patternMatches` compiles and matches path patterns through `folded`.
   * `relative` tries the root as given and as resolved.
   * `Evaluate` takes `matchReadings` per subject, by deviation 1.
3. **`permission/store.go`:** the project store's two lookups compare through `fsx.SamePath`.
4. **`permission/trust.go`:** `samePath` is unchanged. Its comment now says why a trust grant is not folded.
5. **Tests.**
   * `TestProtectedFoldsCase`: denies and asks on darwin and windows, and allows on linux, where the spellings are
     other files. *The plan's step 6 listed the denials without the platform; the MADR's D2 limits folding to darwin and
     windows, and the test follows the MADR.*
   * `TestProtectedRealPath` and `TestPathRulesRealPath`, the latter with deviation 1's allow through a link.
   * `TestPathRulesFoldCase`.
   * `TestStoreLookupFoldsCase` and `TestTrustStaysExact`, in `store_test.go`, which is in P2's list. Both run where the
     temp volume ignores case, as on macOS and windows, and skip elsewhere, with the reason (C10).
     `TestTrustStaysExact` asserts trust as it compares, by deviation 2: not found on darwin, found on windows.
   * `TestFold` and `TestFoldWindows`, the latter on Windows.
   * The existing tests pass unchanged, after deviation 1.
6. **Seen failing (C9),** each on its own scratch copy, absolute paths left out here:
   * `.git` compared exactly: `TestProtectedFoldsCase` fails, `.GIT/config: {Effect:allow By:allow: write Limit:}, want
     deny`, and the same for `.Git/config` and `docs/../.GIT/config`;
   * no real-path reading:
     * `TestProtectedRealPath` fails, `docs/x/config through a link to .git: {Effect:allow …}, want the protected limit`;
     * `TestPathRulesRealPath` fails, `link/k, a link to secret: {Effect:allow …}, want the deny`;
   * a default outranking a matched reading, as first written: `TestEvaluateSubjects` (`an absolute pattern:
     {Effect:ask …}`), `TestRemember` and `TestPathRulesRealPath` (`an allow for the real path, through the link:
     {Effect:ask …}, want the allow`) fail;
   * the store compared exactly: `TestStoreLookupFoldsCase` fails, `Rules(…/proj) = [], <nil>; want the reject saved
     for …/Proj`;
   * trust folded: `TestTrustStaysExact` fails, `Lookup(…/proj) found the grant for …/Proj: ok true`. The first try of
     this break failed to build, because `trust.go` already imports `fsx`. It was rerun without the duplicate import.
   * `Fold` a no-op: `TestFold` fails, `Fold(".GIT") == Fold(".git") is false, want true`, and the same for the
     `SamePath` and `UnderDir` lines.
7. **G-local:**
   * `make pre-add-check` on the 7 Go files: clean;
   * `make preflight`: passed, after the rename in step 1;
   * `make race`: 36 ok.
   * After deviation 2's test change: `make pre-add-check` on `permission/store_test.go` was clean, `make preflight`
     passed, and `go test -race ./permission/` was ok.
8. **G-hosts:** first run: the Linux machine passed; on the Windows laptop `TestTrustStaysExact` failed in `go test` and race, which is deviation 2. After it, on both hosts, `go test` passed, race passed with 36 ok, and `make check-records markdownlint` passed. `make preflight` stopped at `verify-build-metadata`, as on every copy..

### Paused after P2 (2026-10-31)

The owner asked for acp-go-sdk2's new release to be pinned, and go-tui-lib `v2.7.0` arrived the same day. Both resume
held work in [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md),
by its deviation 12, which the owner ordered first ("0035 first (Recommended)"). This plan resumes at P3 once that
plan's P19 is done. 0035's P17 returns to `main` as this plan's Rollout and Rollback says: `feature/p17-lineedit` is
rebased onto `main`, and the owner fast-forwards `main` to it.


### Resumed at P3 (2026-11-01)

0035-PLAN's P19 is done, so this plan resumes at P3, as the pause above says, on the owner's "proceed". `main` is at
`5e41482`. 0035's P17 reached `main` through that plan's own phases, as its execution record shows.

### P3 record (2026-11-01): bash's literal write targets meet the protected-path limit

The base is `5e41482`. No deviation.

1. **The parser** (step 1): `shellCommand` gains `writes`. `add` collects a redirection's literal file when `writesTo`
   says it writes: `RdrOut`, `AppOut`, `RdrClob`, `RdrAll`, `AppAll` and `RdrInOut`, a sink excluded by
   `redirectPath`. `call` collects `writeOperands`: every literal operand of `tee`, `truncate`, `rm`, `rmdir`, `mv`,
   `cp`, `mkdir`, `touch`, `ln`, `chmod` and `chown` (`writeProgram`), by `fileOperands`; and `inPlaceFiles` for `sed`,
   `perl` and `ruby`, when `-i`, `-i<suffix>`, `--in-place` or `--in-place=…` is among the options, the script left out:
   the word after `-e`, `--expression` (and `-f`, `--file` for `sed`), or after a `perl`/`ruby` bundle ending in `e` or
   `E`, else the first operand.
   * A writing program's every operand counts, such as `chmod`'s mode; none names a protected path.
   * `perl -pi -e …`, its in-place flag bundled, is not recognised: the step names `-i`, `-i<suffix>` and `--in-place`.
     `docs/architecture.md` says so.
   * golangci-lint's goconst found `"sed"` four times in the new code; it is the constant `sed`.
2. **The tool** (step 2): `bashSubject` returns `Writes`, each resolved by `resolve` against the workdir, as
   `bashOutside` resolves its paths, each once.
3. **The rule** (step 3): `approve` sets `rule.Writes = subject.Writes`.
4. **The limit** (step 4): `limits` checks `r.Writes` first, for any tool: a protected path denies (`limitProtected`), and
   a config path asks (`limitConfig`) unless the verdict is deny. Both use P2's `protected` and `configPath`.
5. **`docs/architecture.md`** (step 5): the permission section names the redirections, programs and in-place flags the
   check covers, and what it does not: a pipeline into `xargs`, `find -delete`, a bundled flag such as `perl -pi`, and
   a path built at run time.
6. **Tests** (step 6):
   * `TestShellWrites` (`tool/builtin/shellparse_test.go`): each redirection operator, each writing program, quoted
     operands, `sed -i`, `sed -i.bak -e`, `--in-place`, `sed -i -f`, `sed` without it, `perl -i -pe`, `ruby -i -pe`,
     `perl -pe` without it, a sink, a read, `$F`, a command substitution and `bash -c`. A second part checks step 2's
     resolution through `bashSubject`: against the cwd, once; against the workdir; `~` at the home directory;
   * `TestBashWritesProtected` and `TestBashWritesConfig` (`permission/rules_test.go`), each rule carrying its `Writes` as
     bash's Subject gives them, since `permission`'s tests do not import `tool/builtin`. `.GIT/config` is denied where
     the volume folds case, `folds()`, and allowed by the rule on linux, as P2's `TestProtectedFoldsCase` has it.
7. **Seen failing** (step 7, C9), each on a scratch copy:

   | Break | The failure |
   | :--- | :--- |
   | `limits` ignoring `Writes` (the step's) | `TestBashWritesProtected`: `"echo x > .git/config": {Effect:allow By:allow: bash(echo *) Limit:}, want deny, limit "gobble never changes files under .git, …"`, and the same for each protected case; `TestBashWritesConfig`: `echo x > AGENTS.md: {Effect:allow …}, want the limit's question` |
   | the parser collecting no writes | `TestShellWrites`: `"echo x > out": [], want ["out"]`, and each case after |
8. **`apidiff`** (step 8): `make preflight`'s step reports "apidiff: no-op below v1.0.0", as at P1. `go doc -all` of `tool`
   and `permission`, against `5e41482`, shows `Subject.Writes` and `Rule.Writes` added, and nothing else.
9. **G-local:** `make pre-add-check` on the 8 changed Go files: clean; `make preflight`: passed, its first run stopped at
   the goconst finding above; `go test ./...`: 38 ok; `make race`: 38 ok. P3 touches nothing under `internal/cli`, so no
   `make perf`.
10. **G-hosts,** 809 files shipped to each: the first run, before the goconst fix, passed `go test` and race and stopped
    preflight at the same finding on both; on the fixed tree, `go test` and race passed on both, 38 ok,
    `make check-records markdownlint` passed on both, and preflight stops at `verify-build-metadata`, as on every copy.

### Deviation 3 (2026-11-01): P4 step 1, the normalised subject's save

- **Found** at P4's start. Step 1 appends the normalised subject "with an empty save (`""`)", and says "always" answers
  save what they save today. Those contradict:
  - `Engine.Remember` (`permission/rules.go:431-460`) makes a rule from every subject, and an empty save means the
    subject's exact text;
  - the question's patterns are `Remember`'s (`acpserver/modes.go:163-173`).

  So `/bin/rm -rf build`, answered "always allow", would also save `bash(rm -rf build)`, and the question would show it.
  A normalised form holding `*`, such as `"ls" *.go` read as `ls *.go`, would make `Remember` refuse, leaving only
  "once" where "always" is offered today.
- **The owner's answer:** "Normalised form reuses raw save (Recommended)". The option not taken was a match-only field,
  `Forms`, on `tool.Subject` and `permission.Rule`.
- **Corrected before any code, the same day.** The chosen option's "or the raw text when that save is empty" would make
  `Remember` refuse where it saves today: a raw save is empty only when a word of the arity prefix is an expansion, such
  as `git ${SUB} -f`, or for the `PATH=…` subject, and `Remember` checks a save against `*?[{` but an exact subject
  against `*?` alone. The owner's answer: "Skip when raw save empty (Recommended)". The option not taken kept the raw
  text as the save.
- **Decision.** The normalised subject is added only when it differs from the raw subject and the raw save is not
  empty, and its save is the raw subject's. `Remember` then makes the same rule for both and keeps it once, so saved
  rules, the question's patterns and whether "always" is offered are as today. A test pins `Remember`'s rules for each
  bypass. A command whose arity prefix holds an expansion gets no normalised form; its deny pattern cannot name that
  word literally anyway. Step 1 is annotated.
- **Files added:** none.

### Deviation 4 (2026-11-02): P4, three parser tests see the normalised subject

- **Found** running P4's tests: `TestParseShellSubjects` (`tool/builtin/shellparse_test.go:42`),
  `TestParseShellMasking` (`:84`) and `TestParseShellWrappers` (`:109`, `:120`) list each command's exact subjects, and
  now see a second, normalised one: `git status` after `"git" 'status'`, `echo test` after
  `echo test > output.txt`, and 17 subjects at depth 8 where there were 9. A raw subject keeps its
  masked redirections, its quoting and a `bash -c` script as written, so its normalised form differs. It is MADR D4's
  design; the step did not name these tests.
- **The owner's answer:** "Update the expected lists (Recommended)". The option not taken added a normalised subject only
  for a difference a deny pattern could be dodged by, which would still miss `echo hi > out.txt` under
  `deny: bash(echo hi)`.
- **Decision.** Each failing case's expected subjects, and saves, gain the normalised entry as the parser gives it,
  written in full; the masking cases assert the raw subject and its normalised form as a pair; the depth tests count
  both forms. Nothing is loosened. Each change is named in the P4 record.
- **Files added:** none; `shellparse_test.go` is P4's.

### P4 record (2026-11-02): commands matched in their raw and their normalised form

The base is P3's `062909d`. Deviations 3 and 4.

1. **The normalised subject** (step 1, deviation 3): `call` adds, after a command's raw subject, `normalised`: its words
   as `wordValue` reads them, a non-literal word as written, joined by single spaces; the program by `path.Base` when it
   holds a `/`, and lower-cased on darwin and windows. It is added when it differs from the raw subject and the raw save
   is not empty, with the raw subject's save, so `Remember` keeps one rule for both.
2. **Matching** (step 2): `wildcard` ignores case on darwin as on windows. `Evaluate` already takes the strictest
   subject's verdict, so a deny on either form denies.
3. **Tests** (step 3, deviation 4):
   * `TestCommandBypasses` (`tool/builtin/shellparse_test.go`, which imports `permission`; `permission` imports no
     `tool` package): under `allow: bash` then `deny: bash(rm *)`, `\rm -rf build`, `"rm" -rf build`,
     `r''m -rf build`, `rm<TAB>-rf build` and `/bin/rm -rf build` are denied, with F3's denied rows; `RM -rf build` on
     darwin and windows; `echo build | xargs rm -rf` and `find build -delete` are allowed, the test saying why. A last
     part pins `Remember`: an "always" saves `rm *`, `/bin/rm *`, `echo *` and `git status *`, once each, and nothing for
     a normalised form;
   * `TestWildcard` (`permission/rules_test.go`) gains a case: `RM -RF build` against `rm *` matches on darwin and
     windows, and not on linux. Without it no test held step 2, since the normalised subject already lower-cases the
     program;
   * by deviation 4, `TestParseShellSubjects` ("quoted words", "a redirection kept"), `TestParseShellMasking` (each of
     its eight cases now a raw and normalised pair) and `TestParseShellWrappers` (`bash -c 'rm x && ls'`,
     `sh -ec "git push"`, and depth 8's count, 9 to 17) list the normalised subjects as the parser gives them.
4. **`docs/architecture.md`** (step 4): a bash command is matched as written and as the shell runs it; a deny rule sees
   its program and its literal words; `xargs`, `find -delete` and a command assembled at run time are beyond it.
5. **Seen failing** (step 5, C9), each on a scratch copy:

   | Break | The failure |
   | :--- | :--- |
   | no normalised subject (the step's) | `TestCommandBypasses`: `"\\rm -rf build": {Effect:allow By:allow: bash Limit:}, want denied`, and the same for `"rm"`, `r''m`, the tab and `/bin/rm` |
   | the normalised subject's save `""`, as step 1 first read | `TestCommandBypasses`: `"\\rm -rf build": always saves ["rm *" "rm -rf build"] (true), want ["rm *"]`, and the same for each row |
   | `wildcard` folding case on windows alone | `TestWildcard`: `wildcard(RM -RF build, rm *) = false, want true on darwin` |
6. **G-local:** `make pre-add-check` on the 4 changed Go files: clean; `make preflight`: passed; `go test ./...`: 38 ok;
   `make race`: 38 ok. P4 touches nothing under `internal/cli`, so no `make perf`.
7. **G-hosts,** 809 files shipped to each: `go test` and race passed on both, 38 ok; `make check-records markdownlint`
   passed on both; preflight stops at `verify-build-metadata`, as on every copy.

### Deviation 5 (2026-11-02): P5, a dangling symlink is replaced by a regular file

- **Found** after P5's gates had passed, by a probe on a scratch copy. Step 1 resolves a symlink with `fsx.RealPath`,
  which cannot resolve a link whose target does not exist yet: it returns the link's own path, and `writeAtomic`
  renames over it. With `CLAUDE.md -> AGENTS.md` and no `AGENTS.md`, the probe printed
  `inside: err=<nil> link mode=-rw-r--r-- target exists=false`: the link was replaced and the target not made. A dangling
  link to a path outside the roots was refused by `os.Root` (`statat escape.md: path escapes from parent`), not with
  `ErrOutside`, and kept. The behaviour is as before P5, which replaced every link, but contradicts MADR D6 ("writes the
  link's target, and keeps the link") for this case.
- **The owner's answer:** "Follow it with Readlink (Recommended)". The options not taken were to refuse a write through
  a dangling link, or to keep the behaviour and amend D6 to say so.
- **Decision.** `follow` resolves a link that `filepath.EvalSymlinks` (what `RealPath` tries first) can resolve as
  step 1 says. A dangling one is read with `os.Readlink`, a relative target joined to the link's directory, and the
  result followed again, up to 40 hops, Linux's own limit; past that the write fails, and the link is kept. The write
  then creates the target, and the root check applies to it, so an outside target gets `ErrOutside`. A test,
  `TestWriteThroughDanglingSymlink`, covers a target inside the roots, one outside, and a loop, and each case is seen
  failing. D6 is unchanged: this makes the code meet it. Step 1 is annotated.
- **Files added:** none; `write.go` and `write_test.go` are P5's.

### P5 record (2026-11-02): writes through symlinks and hard links

The base is P4's `c4eb473`. Deviation 5.

1. **A symlink is followed** (step 1, deviation 5): `follow` (`internal/fsx/write.go`) `Lstat`s the path; a symlink is
   resolved with `filepath.EvalSymlinks`, what `RealPath` tries first, and a dangling one is read with `os.Readlink`, a
   relative target joined to the link's directory, up to `maxHops` (40) links. `Workspace.WriteFile` then opens the
   target through its root as before, so a target outside every root gets `ErrOutside`; `WriteFileMode` follows the
   same way, with no root. The link itself is not written.
2. **A file with more than one hard link is written in place** (step 2): `writeAtomic` asks `links` for the existing
   file's count, and above one calls `writeInPlace`: `O_WRONLY|O_TRUNC`, write, sync, close, through `fill`, which the
   atomic path now shares. `links` is `Stat_t.Nlink` in `write_unix.go` (`//go:build !windows`, as
   `internal/appdirs/base_unix.go`), and `GetFileInformationByHandle`'s `NumberOfLinks` in `write_windows.go`, which
   opens the file to read it.
3. **The directory is synced** (step 3): after the rename, `syncDir` opens the parent through the same `fileSystem` and
   syncs it, and its error is returned. On Windows it does nothing.
4. **Tests** (step 4, `internal/fsx/write_test.go`, a new file):
   * `TestWriteThroughSymlink`: `CLAUDE.md -> AGENTS.md` stays a link and `AGENTS.md` holds the new bytes; a second
     part does the same through `WriteFileMode`, which step 1 names and no listed test held;
   * `TestWriteSymlinkOutsideRoot`: refused with `ErrOutside`; the link and its target unchanged;
   * `TestWriteThroughDanglingSymlink` (deviation 5): a dangling link's target inside the roots is made and the link
     kept; one outside is refused with `ErrOutside`, the target not made; `a.md -> b.md -> a.md` fails and both stay
     links;
   * `TestWriteHardLinkInPlace`: `b` holds the new bytes, `a` and `b` are one file, and `links` counts 2;
   * `TestWriteNewFileAtomic`: a single-link file gets a new inode. It skips on Windows, as the step says.

   The symlink tests skip, with the reason, where a symlink cannot be made (C10); on the Windows laptop they ran.
5. **Seen failing** (step 5, C9), each on a scratch copy:

   | Break | The failure |
   | :--- | :--- |
   | step 1 undone: a link written as itself (the step's) | `TestWriteThroughSymlink`: `CLAUDE.md is no longer a symlink (mode -rw-------) after the write, want the link kept`; `TestWriteThroughDanglingSymlink` the same |
   | `WriteFileMode` alone not following | `TestWriteThroughSymlink`: `linked.json is no longer a symlink (mode -rw-------) after the write, want the link kept` |
   | a followed link written past the root check | `TestWriteSymlinkOutsideRoot`: `write through the link = <nil>, want ErrOutside`; `TestWriteThroughDanglingSymlink`: `write through the dangling link out = <nil>, want ErrOutside` |
   | deviation 5 undone: `RealPath` alone, as step 1 first read | `TestWriteThroughDanglingSymlink`: `CLAUDE.md is no longer a symlink (mode -rw-r--r--) after the write, want the link kept` |
   | the hop bound removed | `TestWriteThroughDanglingSymlink`: `panic: test timed out after 20s` |
   | step 2 undone: no write in place (the step's) | `TestWriteHardLinkInPlace`: `b = "old\n", want the new bytes` |
   | every existing file written in place | `TestWriteNewFileAtomic`: `in.txt kept its inode, want a new file renamed into place` |

   A first loop break kept the bound and replaced only its error with the link's path. It passed: `os.Root` then
   refuses the loop itself when it stats the link. The bound is what that break must undo, so it was changed to remove
   it.
6. **Not done:** step 3 has no test. A directory sync changes nothing a process can see; only a crash after the rename
   would show it.
7. **G-local:** `make pre-add-check` on the 4 Go files: clean; `make preflight`: passed; `go test ./...`: 38 ok;
   `make race`: 38 ok. P5 touches nothing under `internal/cli`, so no `make perf`.
8. **G-hosts,** 812 files shipped to each: `go test` and race passed on both, 38 ok, and the five tests above passed
   on the Linux machine; on the Windows laptop four passed and `TestWriteNewFileAtomic` skipped.
   `make check-records markdownlint` passed on both; preflight stops at `verify-build-metadata`, as on every copy.
   On the Linux machine the first run's log ended after the race count, likely cut short by the remote processes of
   a run stopped earlier for the `links` change; preflight and records were rerun on that copy: preflight stopped at
   `verify-build-metadata`, and records passed.

### Deviation 6 (2026-11-02): P6, grep's single file and textIO.read's local path

- **Found** reading the code before P6's first write. Step 3 changes `candidate`, `grep`'s directory walk, only; `grep`
  naming one file reads it in `runGrep` (`tool/builtin/grep.go:88-92`, `s.ws.ReadFile(s.abs)`), where a FIFO's size is
  0, and blocks. The file list names `textIO.read` (`tool/builtin/builtin.go:159`), but step 2 describes only its client
  path; its local path is `ws.ReadFile`, which `edit` (`tool/builtin/edit.go:46`, `:64`) and `write` (`readOld`,
  `tool/builtin/write.go:51`) also read through, and which blocks on a FIFO the same way. Both hangs are older than P6.
- **The owner's answer:** "Cover both in P6 (Recommended)". The options not taken were to do the steps as written and
  record both as not done, or to fix `grep` alone.
- **Decision.**
  * `grep` naming one file reads it through `Workspace.ReadRegular(abs, maxGrepFileBytes)`. `ErrTooLarge` counts it as
    too big, as today; `ErrNotRegular` is the call's error, `grep <name>: not a regular file`, since a skip would read
    as "no matches".
  * `textIO.read`'s local path opens through `OpenRegular` and reads the whole file, with no new size limit, so `edit`
    and `write` naming a FIFO fail at once with "not a regular file".
  * Two more Unix tests, each seen failing: `TestGrepFileRefusesFIFO` and `TestEditRefusesFIFO`, each within 1 s.
  * MADR D7 gets a dated note.
- **Files added:** none; `grep.go`, `builtin.go` and `grep_test.go` are P6's, and `TestEditRefusesFIFO` goes in
  `read_test.go`, beside `TestReadRefusesFIFO`.

### P6 record (2026-11-02): the file tools read only regular files, and only what they need

The base is P5's `7114b5e`. Deviation 6.

1. **Opening** (step 1):
   * `internal/fsx/regular.go`: `ErrNotRegular` ("not a regular file") and `ErrTooLarge` ("file too large") are a
     `fileError` whose `Unwrap` is `fs.ErrInvalid`. `regular` checks the opened file's own `Stat`, and closes it when it
     is not a regular file. `readAtMost` refuses a file whose size is over the limit before reading it, then reads at
     most limit + 1 bytes, so a file that grows is caught; the error names the limit (`file too large: over 1 KiB`).
     `ReadRegularFile`, for P7, opens a path outside any workspace the same way.
   * `nonblock_unix.go` (`//go:build unix`) is `syscall.O_NONBLOCK`; `nonblock_other.go` is 0.
   * `Workspace.OpenRegular` and `ReadRegular` (`workspace.go`) open through the root as `ReadFile` does, with
     `O_RDONLY|nonblock`; both go through `openRegularInfo`, which also returns the info.
2. **`read`** (step 2): `openText` (`tool/builtin/read.go`) opens an image whole through `ReadRegular` up to
   `maxImageBytes` (20 MiB); a file the client offers through `textIO.read`, which checks it with `OpenRegular` first;
   and any other file through `OpenRegular`, streamed by a 64 KiB `bufio.Reader`. The image and binary sniffs read the
   reader's first 4096 bytes, the bytes they read before. `numbered` now takes the reader: it holds only the lines it
   shows, at most `lineKeep` (8,004) bytes of each, and `countRest` counts the rest in 64 KiB chunks. Every existing
   `TestReadText` output is unchanged.
3. **`grep`** (step 3, deviation 6): `Walker` keeps an `open` function through its root, and `Walker.ReadRegular`
   reads through it. `candidate` reads through `Walker.ReadRegular(rel, maxGrepFileBytes)`, so its `fs.DirEntry`
   parameter went; `ErrTooLarge` counts the file as too big, as the size check did, and anything else unreadable is
   skipped. Grep naming one file reads it through `Workspace.ReadRegular`, and `ErrNotRegular` is the call's error,
   `grep pipe: not a regular file`; `search.size`, which only that path read, went with the old check.
4. **`copy`** (step 4): `CopyFile` opens the source with `openRegularInfo` and streams it into `writeFrom`, which is
   `WriteFile` taking an `io.Reader`; `writeAtomicFrom` is `writeAtomic`'s reader form, and `fill` copies with
   `io.Copy`. `WriteFile` and `writeAtomic` pass a `bytes.Reader`. The rename, the sync, the mode, and P5's links are
   unchanged.
5. **`textIO.read`** (deviation 6): its local path opens through `OpenRegular` and reads with `io.ReadAll`, with no
   new limit, so `edit` and `write` naming a FIFO fail at once.
6. **Tests** (step 5, deviation 6):
   * `tool/builtin/read_test.go`: `TestReadRefusesFIFO` (`read pipe: not a regular file`, under 100 ms, a 2 s
     deadline); `TestEditRefusesFIFO`, which covers `write` as well (`edit pipe: not a regular file` and
     `write pipe: not a regular file`, a 1 s deadline); `TestReadStreamsLimit` (an 8 MiB file read with `limit: 50`
     allocates under 2 MiB; the footer gives 131,072 lines); and two `TestReadText` cases for the held prefix: a
     20,000-byte line, and 3,000 four-byte characters, each cut at 2,000.
   * `tool/builtin/grep_test.go`, a new file: `TestGrepSkipsFIFO` and `TestGrepFileRefusesFIFO`, each with a 1 s
     deadline.
   * `internal/fsx/ops_test.go`: `TestCopyFileStreams` (16 MiB, under 2 MiB allocated, the bytes and the 0640 mode
     match) and `TestCopyRefusesFIFO` (`ErrNotRegular` within 2 s, and nothing made).
   * `internal/fsx/regular_test.go`, a new file: `TestReadRegular` and `TestReadRegularRefusesFIFO`, which hold the
     three readers directly, `ReadRegularFile` having no caller until P7: a regular file read, a directory and a FIFO
     `ErrNotRegular`, a 1,025-byte file over a 1 KiB limit `ErrTooLarge`, each also `fs.ErrInvalid`.
   * The FIFO tests make the FIFO with `mkfifo(1)`, so the test files build on Windows without per-platform files, and
     skip there with the reason (C10). On the Windows laptop the six skipped; everything else ran.
   * Measured on a scratch copy on macOS, three runs each: the limited read allocates about 404 KB, the copy about
     60 KB.
7. **Seen failing** (step 6, C9), each on a scratch copy:

   | Break | The failure |
   | :--- | :--- |
   | step 1 undone: no flag, no check (the step's) | `TestReadRefusesFIFO`: `read {"path":"pipe"} was still running after 2s`; every other FIFO test on its deadline the same way |
   | the flag kept, the check gone | `TestReadRefusesFIFO`: `read of a FIFO = "(empty file)" (isError false), <nil>`; `TestGrepFileRefusesFIFO`: `"grep pipe: no matches for hit"`; `TestCopyRefusesFIFO`, `TestReadRegularRefusesFIFO` |
   | step 2 undone: today's whole-file read (the step's) | `TestReadStreamsLimit`: `read of 8 MiB with limit 50 allocated 8746232 bytes, want under 2 MiB` |
   | a shown line held to 2,000 bytes | `TestReadText`: the long-line cases, without the cut |
   | step 3 undone: `candidate` with `ReadFile` | `TestGrepSkipsFIFO`: `grep {"pattern":"hit"} was still running after 1s` |
   | deviation 6 undone for one file | `TestGrepFileRefusesFIFO`: `grep {"pattern":"hit","path":"pipe"} was still running after 1s` |
   | deviation 6 undone for `textIO.read` | `TestEditRefusesFIFO`: `edit {"path":"pipe",…} was still running after 1s` |
   | step 4 undone: copy whole | `TestCopyFileStreams`: `a 16 MiB copy allocated 16820408 bytes, want under 2 MiB`; `TestCopyRefusesFIFO`: `still blocked after 2s` |
   | the limit not enforced | `TestReadRegular`: `Workspace.ReadRegular(big.txt) = <nil>, want ErrTooLarge naming the 1 KiB limit`, and the same for the other two |

   With the flag kept and the check gone, `TestGrepSkipsFIFO` passes: the FIFO opens at once and reads empty, so it has
   no match. The flag is what it holds.
8. **G-local:** the first `make preflight` stopped at `golangci-lint` on the new code: gosec G602 on `buf[k-1]`
   (`countRest` now indexes the chunk read), ineffassign on a test's `content = ""` (the content is no longer held),
   noctx on `exec.Command` (now `exec.CommandContext(t.Context(), …)`), revive's redefines-builtin-id on five `max`
   parameters (now `limit`), and confusing-naming on `openRegular` beside `OpenRegular` (now `openRegularInfo`). Each
   was fixed in the code, and the breaks were rerun on the result. Then `make pre-add-check` on the 14 Go files: clean;
   `make preflight`: passed; `go test ./...`: 38 ok; `make race`: 38 ok. P6 touches nothing under `internal/cli`, so no
   `make perf`.
9. **G-hosts,** 817 files shipped to each: `go test` and race passed on both, 38 ok; the new tests passed on the Linux
   machine, and on the Windows laptop as item 6 says. `make check-records markdownlint` passed on both; preflight stops
   at `verify-build-metadata`, as on every copy. A host run stopped for the lint fixes left its preflight running on
   the Linux machine; that process tree, in the scratch copy's directory, was stopped before the rerun, and nothing else
   there was touched.

### Deviation 7 (2026-11-02): P7, `config validate` and completion read the same files

- **Found** reading the code before P7's first write. Two readers of D10's files are not in P7's list.
  `jsonDiagnostics` (`internal/cli/config.go:365`), which `gobble config validate` runs, reads `mcp.json`, `auth.json`,
  `models.json`, `permissions.json` and the trust file with `os.ReadFile`; shell completion's `completionModelsJSON`
  (`internal/cli/completion.go:143`) reads `models.json` the same way. After P7 as written, gobble would refuse a FIFO
  there at once while `config validate` and tab completion block on it, and read a large one whole. Both are older
  than P7.
  Two of P7's line references had moved, the functions unchanged: `mcpclient/config.go:84` and `mcpclient/file.go:159`.
  `personality.md`, `messages.md`, skills, prompt templates, `recent-models.json` and the catalog cache are prompt
  content or state, not D10's configuration files, and P7 leaves them as they are.
- **The owner's answer:** "Cover both in P7 (Recommended)". The option not taken was P7 as written, with a later phase
  for these two.
- **Decision.**
  * `jsonDiagnostics` reads each file through `fsx.ReadRegularFile(path, fsx.ConfigLimit)`; a refused file is its
    existing "could not read" error diagnostic, naming the file and the reason.
  * `completionModelsJSON` reads through the same call; a refused file is no models, as an unreadable one is today.
  * Two more Unix tests, each seen failing: `TestConfigValidateRefusesFIFO` and
    `TestCompletionModelsJSONRefusesFIFO`, each within 1 s.
  * MADR D10 gets a dated note.
- **Files added:** `internal/cli/config.go`, `internal/cli/completion.go`, and the tests beside them.

### P7 record (2026-11-02): configuration files read as regular files, within 1 MiB

The base is P6's `603bad6`. Deviation 7.

1. **One reader** (step 1, deviation 7): `fsx.ConfigLimit` is 1 MiB (`internal/fsx/regular.go`). Each reader below now
   calls `fsx.ReadRegularFile(path, fsx.ConfigLimit)` where it called `os.ReadFile`, and keeps its own handling of what
   comes back:
   * `readLayer` (`internal/config/resolve.go`), `Update` (`update.go`), `edit` (`edit.go`) and `editRules`
     (`rulesedit.go`);
   * `Load` (`mcpclient/config.go`) and `readForEdit` (`mcpclient/file.go`);
   * `RuleStore.Load` (`permission/store.go`) and `TrustStore.Load` (`permission/trust.go`);
   * `readModelsJSON` (`internal/cli/modelsjson.go`) and `Load` (`internal/auth/store.go`);
   * by deviation 7, `jsonDiagnostics` (`internal/cli/config.go`) and `completionModelsJSON`
     (`internal/cli/completion.go`).

   A missing file is still `fs.ErrNotExist`, so every "missing is empty" path is unchanged. A refused file takes each
   caller's existing route for an unreadable one: an error `Diagnostic` from `readLayer` (`open <file>: not a regular
   file`) and from `jsonDiagnostics` (`could not read: open <file>: not a regular file`), the error `Update`, `Set`, `AppendRule`, `Add`, `Remove`, both
   stores and `auth.Load` already return, `Load`'s one entry in `Config.Errors`, `readModelsJSON`'s warning, and
   completion's nil. `os` left the imports of `resolve.go`, `rulesedit.go` and `mcpclient/config.go`; `internal/auth`
   imports `internal/fsx` for the first time, which no archtest rule forbids, and `fsx` imports no package of this
   module.
2. **The project layer before trust** (step 2): `Resolve` reads the project layer through `readLayer`, so item 1 covers
   it; `TestResolveRefusesFIFO` checks it on an untrusted project.
3. **Tests** (step 3, deviation 7), each package's in a new `regular_test.go`, with its own `fifo` (`mkfifo(1)`,
   skipping on Windows, C10) and a deadline helper:
   * `internal/config`: `TestResolveRefusesFIFO` (an untrusted project's FIFO `settings.yaml`, within 1 s, an error
     diagnostic naming "not a regular file"); `TestResolveRefusesLarge` (a 2 MiB global `settings.yaml`: a diagnostic
     naming `file too large: over 1 MiB`, under 4 MiB allocated; it runs on every platform); `TestResolveDevZero`
     (`.gobble/settings.yaml -> /dev/zero`, "not a regular file", skipped on Windows); `TestEditorsRefuseFIFO`, with
     `Update`, `Set` and `AppendRule` each refusing a FIFO within 1 s as `fsx.ErrNotRegular`, and each giving
     `fs.ErrNotExist` for a missing file, as before.
   * `mcpclient`: `TestConfigRefusesFIFO`. `Load` of a missing file is empty, and `Add` to one creates it; `Load` of a
     FIFO is one error naming "not a regular file", and `Add` to it is `fsx.ErrNotRegular`, each within 1 s.
   * `permission`: `TestStoresRefuseFIFO`, for `permissions.json` and the trust file: missing is empty, and a FIFO is
     `fsx.ErrNotRegular` with the file's name first, within 1 s.
   * `internal/auth`: `TestLoadRefusesFIFO`, the same for `auth.json`.
   * `internal/cli`: `TestModelsJSONRefusesFIFO` (no models, warned "not a regular file"; missing is none, unwarned);
     `TestConfigValidateRefusesFIFO` (`gobble config validate` with a FIFO `auth.json` exits 1 within 1 s, printing
     `<file>: could not read: open <file>: not a regular file`; with no files it prints nothing, which
     `TestConfigValidateExits` already holds); `TestCompletionModelsJSONRefusesFIFO` (nil within 1 s, as for a missing
     file).
   * Measured on a scratch copy on macOS, three runs: `TestResolveRefusesLarge`'s `Resolve` allocates about 10 KB; the
     size is refused from the opened file's own `Stat`, before a byte is read.
4. **Seen failing** (step 4, C9), each on a scratch copy; a reader "put back" calls `os.ReadFile` again:

   | Break | The failure |
   | :--- | :--- |
   | `readLayer` put back (the step's) | `TestResolveRefusesFIFO`: `still blocked after 1s`; `TestResolveRefusesLarge`: `Resolve of a 2 MiB settings.yaml allocated 14812248 bytes, want under 4 MiB`, and `diagnostics [], want an error about <tmp>/config/settings.yaml naming "file too large: over 1 MiB"` |
   | the flag kept, the type check gone | `TestResolveRefusesFIFO`: only the untrusted-project warning, no error; `TestResolveDevZero`: `read <tmp>/settings.yaml: file too large: over 1 MiB`, not "not a regular file" |
   | `ConfigLimit` at 1 GiB | `TestResolveRefusesLarge`: `diagnostics [], want an error … naming "file too large: over 1 MiB"` |
   | `Update`, `edit` and `editRules` put back | `TestEditorsRefuseFIFO/Update`, `/Set` and `/AppendRule`: each `still blocked after 1s` |
   | `mcpclient.Load` put back | `TestConfigRefusesFIFO`: `still blocked after 1s` at `Load` |
   | `readForEdit` put back | `TestConfigRefusesFIFO`: `still blocked after 1s` at `Add` |
   | both stores put back | `TestStoresRefuseFIFO/permissions.json` and `/trust.json`: each `still blocked after 1s` |
   | `auth.Load` put back | `TestLoadRefusesFIFO`: `still blocked after 1s` |
   | `readModelsJSON` put back | `TestModelsJSONRefusesFIFO`: `still blocked after 1s` |
   | `jsonDiagnostics` put back (deviation 7) | `TestConfigValidateRefusesFIFO`: `still blocked after 1s` |
   | `completionModelsJSON` put back (deviation 7) | `TestCompletionModelsJSONRefusesFIFO`: `still blocked after 1s` |

   `TestResolveDevZero` is not run on the step's copy: with `os.ReadFile` it reads `/dev/zero` until memory runs out,
   F9's own finding. The no-check copy holds it, failing on the limit instead. The allocation bound was first checked
   after the diagnostic, which stopped the test before it; it now comes first, so the step's copy shows both failures.
5. **G-local:** `make pre-add-check` on the 18 Go files: clean; `make preflight`: passed; `make race`: 38 ok;
   `make perf`: passed (`TestPerfBudget`). The first `go test ./...` failed one test, `TestLastGoodOnDisk`, in its
   teardown: `TempDir RemoveAll cleanup: … /catalog: directory not empty`. That is the late writer under a test's
   `GOBBLE_HOME` that 0039-PLAN's deviations 14 and 18 and 0035-PLAN's deviation 14 recorded, which waits for a record
   of its own; P7 changes nothing it runs. The second run: 38 ok.
6. **G-hosts,** 822 files shipped to each: `go test` and race passed on both, 38 ok; `make check-records markdownlint`
   passed on both; preflight stops at `verify-build-metadata`, as on every copy. `make perf` passed on both
   (`TestPerfBudget` in `internal/cli` and in `internal/cli/editor`), and `make probe-conhost` passed on the Windows
   laptop (`TestConhost`). Run alone with `-v`, the thirteen new tests passed on the Linux machine. On the Windows laptop
   `TestResolveRefusesLarge` passed; the FIFO cases skipped ("Windows has no FIFOs") and `TestResolveDevZero` skipped
   ("Windows has no /dev/zero"), each test's missing-file checks running before its skip.

### Deviation 8 (2026-11-02): P8, layer 3's escaped match, and the fuzz test

- **Found** reading the code before P8's first write.
  * Layer 3's `escapeNormalized` (`tool/builtin/editstrategy.go:124-132`) steps past each whole match
    (`off += i + len(plain)`), so it too counts only non-overlapping matches. On a scratch copy, the file
    `x\n}\n}\n}\n` with the oldText `}\n}\n`, written with real newlines or as the escaped literal `}\\n}\\n`, was
    accepted as unique and gave `x\nY\n}\n` both times. D8 says every layer; step 1 covers `locate`'s first two, and
    `editstrategy.go` is not in P8's list. `lineTrimmed` and `indentFlexible` advance one line at a time, so they already
    count overlapping blocks. The fault is older than P8.
  * `FuzzEditMatch` (`tool/builtin/fuzz_test.go:27-35`) calls `applyEdits` and `toLF`, both of which step 2 changes;
    the file is not in P8's list.
- **The owner's answers:** "Fix it in P8 (Recommended)" and "Update fuzz_test.go (Recommended)". The options not taken
  were to leave layer 3 and record it as not done, and to keep `toLF` for the fuzz test alone.
- **Decision.**
  * `escapeNormalized` advances one byte after each match, so an overlapping second match is a second candidate, and
    `locate`'s existing duplicate error refuses it (`occurs 2 times`). `TestEditOverlappingDuplicate` gains the escaped
    case, seen failing.
  * `FuzzEditMatch` converts its content as the view does, only `\r\n`, and takes `applyEdits`' new result; its two
    properties stay. `toLF` goes.
  * MADR D8 gets a dated note.
- **Files added:** `tool/builtin/editstrategy.go` and `tool/builtin/fuzz_test.go`.

### Deviation 9 (2026-11-02): P8's Windows race run, sixteen racing trust saves outlast the edit lock

- **Found** in P8's G-hosts. The Windows laptop's first full race run failed `TestTrustConcurrentSaves`
  (`permission/trust_test.go:204`, 2.03 s): three of its sixteen concurrent `TrustStore.Save` calls returned
  `trust.json: locked by pid 69216`, and thirteen entries were saved. P8 changes only `tool/builtin`.
- **The owner's answer:** "Stop; investigate first". The option not taken was to record it as a load-dependent failure
  and finish P8.
- **Investigated,** read-only, on scratch copies:
  * Alone under race, 20 runs each on the Windows laptop: 20 of 20 passed at P8's tree (28.8 s) and at `603bad6`,
    before P7 changed `TrustStore.Load` (28.6 s). A second full race run of P8's tree passed, 38 ok.
  * Every run takes about 1.4 s, plain or under race, on the Windows laptop at `c4eb473` (before P5), `603bad6` and P8's
    tree, and on macOS (1.39–1.50 s). No phase of this plan changed it.
  * One uncontended save takes about 5.5 ms on the Windows laptop: the lock 0.2–0.4 ms, `Load` 0.1 ms, the atomic write
    about 1 ms. Sixteen need about 90 ms of work, so the 1.4 s is waiting.
  * **The cause.** `filelock.Lock` waits on a fixed schedule, 20, 40, 80, then 100 ms, with no jitter. Sixteen waiters
    that start together wake together: one takes the lock each round, and the rest sleep the next 100 ms, so sixteen
    saves take about 1.4 s of `EditPolicy`'s 2 s. A loaded host stretches the rounds past the deadline.
  * With each wait drawn from half to all of its gap, in a scratch copy of `lock.go`: 0.19–0.29 s plain and 0.20–0.33 s
    under race on the Windows laptop, ten runs each, and 0.41–0.61 s on macOS, five runs.
- **Decision.** A new phase, P8a, jitters `Lock`'s waits; `EditPolicy`'s numbers and the deadline's last try stay.
  P8 waits, uncommitted, for P8a, and they land in one commit. 0018-MADR D18 and this plan's MADR get dated notes.
- **Files added:** `internal/filelock/lock.go`, `internal/filelock/lock_test.go`, and
  `docs/decisions/0018-MADR-configuration-surface.md`.

### Deviation 10 (2026-11-03): P8a, the lock test is a test of the draw

- **Found** in P8a's seen-failing run. On a copy without the jitter, `TestLockWaitersSpread` passed 2 runs of 3 (0.88 s
  each) and failed one (`16 waiters took 1.370763625s, want under 1s`). Ten runs each on macOS: with a 1 ms hold,
  jittered 0.11–0.21 s, lock-step 0.97–1.37 s; with a 5 ms hold, 0.21–0.35 s and 1.08–1.39 s. The goroutines start far
  enough apart that more than one can win a round, so lock-step is not a fixed floor, and a bound low enough to catch
  it, about 0.6 s, would leave the jittered case about twice its time, a load-dependent test of a load-dependent fault.
- **The owner's answer:** "Deterministic test of the draw (Recommended)". The option not taken was the timing test with
  a 5 ms hold and a 0.6 s bound.
- **Decision.** The draw moves into `spread(d, left)` in `lock.go`. `TestLockJitter` draws 1000 waits with `d < left`:
  each lies in `[d/2, d]`, and they vary; with `d` at or past `left` the wait is exactly `left`, so the deadline's last
  try holds. `TestLockWaitersSpread` is not added. The measurements stay in the record, and `TestTrustConcurrentSaves`
  stays the end-to-end check.
- **Files added:** none.

### P8 record (2026-11-03): edits count overlapping matches, and leave the line endings they did not touch

The base is P7's `e042fef`. Deviation 8.

1. **Uniqueness** (step 1, deviation 8): `occurrences(base, s)` (`tool/builtin/editmatch.go`) is `strings.Count` and
   the first index, with a lone count raised to 2 when `s` occurs again from one byte after the first match. `locate`
   uses it at the exact and the normalised layer, so an overlapping second match is `duplicate(i, 2)`; a count above
   one keeps its number. Layer 3's `escapeNormalized` (`editstrategy.go`) looks again one byte after each match, so an
   overlapping second match is a second candidate, and `locate`'s existing duplicate error refuses it.
2. **Line endings** (step 2):
   * `lfView` converts only `\r\n` to `\n`; a lone `\r` stays a character, so each line of the view is a line of the
     file. The edits' oldText and newText get the same conversion, so a lone `\r` in oldText matches one in the file.
     `toLF` and `restoreEnding` are gone; nothing else used them once `FuzzEditMatch` moved to `lfView`.
   * `applyEdits` returns, with its result, the lines it rewrote: `[]lineEdit`, each a run of the view's lines and its
     replacement text, which is its matched spans widened to whole lines. `lineEdits` builds them, as
     `replaceKeepingLines` did, now at every layer, not the normalised layer alone; `spliceLines` puts them in place.
   * `edit.go`: a file with no `\r` is written as `applyEdits` returns it, with no conversion. Otherwise the file is
     rebuilt from its own lines: an untouched line is copied with its own ending, and the lines an edit wrote take the
     file's dominant ending, `lineEnding`'s first newline.
3. **Tests** (step 3, deviation 8):
   * `tool/builtin/editmatch_test.go`, a new file: `TestEditOverlappingDuplicate`, with `x\n}\n}\n}\n` and `}\n}\n`
     refused as `occurs 2 times`, the file unchanged, at the exact layer, the normalised layer (`‘}\n‘}\n` against
     `'}\n'}\n'}\n`) and the escape-normalised one (the literal `}\\n}\\n`). `TestEditBehaviour`'s `duplicate` keeps
     `occurs 3 times`.
   * `tool/builtin/edit_test.go`: `TestEditKeepsUntouchedEndings` (`a\r\nb\nc\n`, `b` to `B`, gives `a\r\nB\r\nc\n`;
     the same through the normalised layer, `b  `; and two written lines, `a\r\nb\nc\nd\r\n` to
     `a\r\nB\r\nC\r\nd\r\n`); `TestEditLoneCR` (`b\rstill b\n` keeps its `\r` with the edit elsewhere, and oldText
     `b\rstill` matches it); `TestEditLFOnly` (`a\nb  \n\tc\nd` and a normalised edit, each byte-identical outside the
     edit, a last line with no newline included).
   * The existing edit tests pass unchanged, the five CRLF and BOM cases of `TestEditBehaviour` included; no
     expectation depended on whole-file conversion.
   * `tool/builtin/fuzz_test.go`: `FuzzEditMatch` takes `lfView` and the new result, and one more seed, below.
4. **The fuzzer's find.** Run for 30 s on a scratch copy, `FuzzEditMatch` panicked within a second on the content `\r`,
   oldText `" "` and newText `"0"`: both normalise to `""`, `Count("", "")` is 1, and `occurrences` sliced
   `base[1:]` of an empty base. At `e042fef` the same input is not a panic, since `toLF` made the content `\n`; the
   fault was P8's own. `occurrences` looks for a second match only while the first starts inside
   `base`, and the input is now a seed. A second run, 120 s and about 2.5 million inputs, found nothing.
5. **Seen failing** (step 4, C9, deviation 8), each on a scratch copy:

   | Break | The failure |
   | :--- | :--- |
   | `strings.Count` alone (the step's) | `TestEditOverlappingDuplicate/exact` and `/normalised`: `result = "edited f.txt: 1 replacement" (isError false), want the error "edit f.txt: oldText occurs 2 times; …"` |
   | `escapeNormalized` stepping past each whole match (deviation 8) | `TestEditOverlappingDuplicate/escape-normalised`: the same |
   | `toLF` and `restoreEnding` back (the step's) | `TestEditKeepsUntouchedEndings/exact` and `/normalised`: `file = "a\r\nB\r\nc\r\n", want "a\r\nB\r\nc\n"`; `TestEditLoneCR/edited_elsewhere`: `file = "TARGET\nb\nstill b\n", want "TARGET\nb\rstill b\n"`, and `/matched` |
   | the edits' oldText converting a lone `\r` | `TestEditLoneCR/matched`: `oldText was not found` |
   | the written lines left with the view's `\n` | `TestEditKeepsUntouchedEndings`, all three: `/several_lines` gives `"a\r\nB\nC\nd\r\n"` |
   | an LF-only file rebuilt with a CRLF ending | `TestEditLFOnly/exact`: `file = "a\nb  \n\tC\r\nd", want "a\nb  \n\tC\nd"`, and `/normalised` |
   | no guard at the end of `base` | `FuzzEditMatch/seed#7`: `panic: runtime error: slice bounds out of range [1:0]` |

   `TestEditKeepsUntouchedEndings/several_lines` passes on the step's copy, since the old conversion also made that
   file CRLF throughout; the written-lines break holds it. `TestEditLFOnly` passes on the step's copy too: the old code
   already left a file with no `\r` whole. It guards the no-`\r` path, and fails when that path is skipped and the file
   is rebuilt with a CRLF ending.
6. **G-local:** `make pre-add-check` on the 6 Go files: clean; `make preflight`: passed; `go test ./...`: 38 ok;
   `make race`: 38 ok. P8 touches nothing under `internal/cli`, so no `make perf`.
7. **G-hosts,** first run, 823 files shipped to each: on the Linux machine `go test` and race passed, 38 ok, and
   `make check-records markdownlint` passed; on the Windows laptop `go test` and records passed, and race failed
   `TestTrustConcurrentSaves` (`trust.json: locked by pid 69216`), which is deviation 9. P8 was held for P8a, and the
   final gates ran over both: P8a's record, items 4 and 5.

### P8a record (2026-11-03): the edit lock's waits, jittered

The base is P7's `e042fef`, with P8 held in the tree. Deviations 9 and 10.

1. **Jitter** (step 1, deviation 10): `spread(d, left)` (`internal/filelock/lock.go`) is the wait before `Lock`'s next
   try: `left` when the gap `d` reaches the deadline, so the last try is still at the deadline; otherwise
   `d/2 + rand.N(d/2+1)`, from `math/rand/v2`. golangci-lint's gosec G404 (a weak random number generator) is marked
   `//nolint:gosec // G404: a retry's spread, not a secret`. `EditPolicy` is unchanged.
2. **Tests** (step 2, deviation 10): `TestLockJitter` (`internal/filelock/lock_test.go`) draws 1000 waits with a 100 ms
   gap and 1 s left: each within `[50ms, 100ms]`, at least 100 distinct; with the gap at or past the time left, the wait
   is exactly the time left. `TestLockWaitersSpread`, as step 2 first wrote it, was not kept: lock-step passed it in 2
   runs of 3. `TestLockDeadline`, `TestLockOutlastsPiBudget`, `TestLockRetries` and `TestLockContext` pass unchanged.
   Measured on macOS, sixteen contended lockers, ten runs each: jittered 0.11–0.21 s with a 1 ms hold and 0.21–0.35 s
   with 5 ms; lock-step 0.97–1.37 s and 1.08–1.39 s. `TestTrustConcurrentSaves` on macOS: 0.35–0.53 s, from 1.39–1.50 s.
3. **Seen failing** (step 3, C9), each on a scratch copy:

   | Break | The failure |
   | :--- | :--- |
   | no jitter, every wait its gap | `TestLockJitter`: `1000 draws gave 1 distinct waits, want them spread` |
   | the last wait jittered too | `TestLockJitter`: `spread(1s, 1s) = 702.317733ms, want exactly 1s` and `spread(2s, 1s) = 1.372208618s, want exactly 1s` |
   | no jitter, against step 2's first test | `TestLockWaitersSpread`: passed at 0.88 s twice, failed once with `16 waiters took 1.370763625s, want under 1s`: deviation 10 |

4. **G-local,** over P8 and P8a: `make pre-add-check` on the 8 Go files: clean; the first `make preflight` stopped at
   golangci-lint's G404, item 1; then `make preflight`: passed; `go test ./...`: 38 ok; `make race`: 38 ok.
5. **G-hosts,** over P8 and P8a, 823 files shipped to each: `go test` and race passed on both, 38 ok;
   `make check-records markdownlint` passed on both; preflight stops at `verify-build-metadata`, as on every copy. The
   Windows laptop then ran the full race suite three more times, 38 ok each, and `TestTrustConcurrentSaves` 20 times
   under race, all passing in 0.19–0.40 s, where it had taken 1.30–1.42 s. P8's and P8a's new tests, run alone with
   `-v`, passed on both hosts.
   * The first attempt at these gates shipped nothing: the scratch host-gate script was missing, and the Windows runs
     it went on to tested the copy left there by P8's run, which is why their trust timings were still 1.30–1.42 s. The
     script was written again, and the gates above are its second run, the shipped copy checked to hold `spread`.

### Deviation 11 (2026-11-03): P9, the notice is written in shell.go, and architecture.md's line

- **Found** reading the code before P9's first write.
  * The notice that names the spill file is built by `outputResult` (`tool/builtin/shell.go:170-177`, "the whole output
    is in <path>"), not in `output.go`; step 2 cannot add its note without `shell.go`, which is not in P9's list.
  * `docs/architecture.md:101` says bash keeps "the whole output of long commands", which is no longer true past the cap.
  * A gap in the list, not a failure; nothing is broken before P9.
- **The owner's answer:** "Add both files to P9 (Recommended)". The option not taken was `shell.go` in P9 and the
  architecture line at P11.
- **Decision.** A cut spill's notice reads "the output is in <path>; the file holds the first <cap>", the size written
  from `spillCap`, so the test's 1 MiB cap reads "1 MiB". A spill under the cap keeps today's "the whole output is in
  <path>", so `TestBashLongOutput` is unchanged. `docs/architecture.md:101` says the output is kept "up to 256 MiB".
- **Files added:** `tool/builtin/shell.go` and `docs/architecture.md`.

### P9 record (2026-11-03): the spill file's cap

The base is P8 and P8a's `f7e1f1f`. Deviation 11.

1. **The cap** (step 1): `spillCap` (`tool/builtin/output.go`) is a package variable, `256 << 20`. `spill.save` writes to
   the file at most `spillCap` bytes in all, `saved` counting them; the write that reaches past it is written up to the
   cap and sets `cut`, and nothing more is written. `open`'s first write of what was in memory goes through `save` too.
   `Write` still counts every byte and line and keeps the tail in memory; only the file stops.
2. **The result** (step 2, deviation 11): `outputResult` (`tool/builtin/shell.go`), for a cut file with no write error,
   says "the output is in <path>; the file holds the first <n> MiB", `n` being `spillCap>>20`. An uncut spill keeps
   "the whole output is in <path>", so `TestBashLongOutput` is unchanged. `docs/architecture.md`'s `tool/builtin` row
   says the output of long commands is kept "up to 256 MiB".
3. **Tests** (step 3): `TestSpillCap` (`tool/builtin/output_test.go`, a new file) sets `spillCap` to 1 MiB and writes
   4 MiB of 64-byte lines in 3000-byte chunks, so one write straddles the cap. The file is exactly the output's first
   1 MiB; the byte and line counts are the whole output's; the tail is `tailKeep` bytes and the output's last; the
   result names the file and "the file holds the first 1 MiB", and ends with the last line. No test in the package runs
   in parallel, so setting the variable is safe.
4. **Seen failing** (step 4, C9), each on a scratch copy:

   | Break | The failure |
   | :--- | :--- |
   | no cap, every byte written (the step's) | `file holds 4194304 bytes, want the output's first 1048576` |
   | the cut file named as the whole output | `result does not say "the output is in <tmp>/gobble-bash-<n>.log; the file holds the first 1 MiB)"` |
   | the tail kept no more once the file is cut | `tail is 102400 bytes, want the output's last 102400` |

5. **G-local:** `make pre-add-check` on the 3 Go files: clean; `make preflight`: passed; `go test ./...`: 38 ok;
   `make race`: 38 ok. P9 touches nothing under `internal/cli`, so no `make perf`.
6. **G-hosts,** 824 files shipped to each: `go test` and race passed on both, 38 ok; `make check-records markdownlint`
   passed on both; preflight stops at `verify-build-metadata`, as on every copy. `TestSpillCap` and
   `TestBashLongOutput`, run alone with `-v`, passed on both hosts.

### Deviation 12 (2026-11-03): P10, a real path that follows the last link refuses calls that work today

- **Found** in a probe on a scratch copy before P10's first write. With `Inside` as
  `UnderDir(RealPath(to), RealPath(from))` and `IsRoot` as `UnderDir(RealPath(r), RealPath(abs))`
  (`internal/fsx/ops.go:159-168`, `:194`), two calls that succeed at `ae80138` were refused:
  * a symlink `l` to the directory `d`, moved to `d/l2`: `move l to d/l2: a directory cannot move into itself`;
  * a symlink to the workspace root, deleted: `delete rootlink: a workspace root cannot be deleted`.

  Move, delete and copy act on the link itself, through `Lstat`, and `RealPath` follows the last link as well. Step 3,
  as written, would add these refusals; nothing is broken before P10.
- **The owner's answer:** "Resolve the parent (Recommended)". The option not taken was `RealPath` of the whole path,
  with the two refusals recorded as a change users see.
- **Decision.**
  * The path an operation acts on is compared as `RealPath` of its directory, joined with its own name: the last element
    is not followed. A workspace root is `RealPath` whole. The comparison is `UnderDir`, through P2's `Fold`.
  * Today's lexical check stays, so a call refused today is still refused.
  * F11's copy of `ws/a` to `<real path of ws>/a/b` is refused by the real comparison.
  * `TestRealPathKeepsLinks` checks the two probe calls succeed, on a host that can make symlinks (C10). MADR D11 gets a
    dated note.
- **Files added:** `docs/decisions/0037-MADR-hardening-integrity-and-performance.md`, for D11's note.

### P10 record (2026-11-03): linear globs, and recursive operations that compare real paths and stop on cancel

The base is P9's `ae80138`. Deviation 12.

1. **`**` collapses** (step 1): `compile` (`internal/fsx/glob.go`) passes its segments through `collapse`, which makes
   each run of `**` one, the leading `**` of a pattern without a `/` included.
2. **The table match** (step 2): `matchSegs` fills one table of `(len(pat)+1) × (len(name)+1)` booleans per call, from
   the end: whether `pat[i:]` matches `name[j:]`. A `**` is `next[j] || row[j+1]`, a trailing one needs one segment
   at least, and a plain segment calls `path.Match` only when the rest of the pattern matches the rest of the name.
   Every existing glob, ignore and walk test passes unchanged.
3. **Real paths** (step 3, deviation 12): `Inside` (`internal/fsx/ops.go`) is `within(from, to)`, as before, or
   `UnderDir(realEntry(to), realEntry(from))`. `IsRoot` is `within(abs, r)`, as before, or
   `UnderDir(RealPath(r), realEntry(abs))`. `realEntry(p)` is `RealPath` of `p`'s directory joined with `p`'s own name,
   so a link the operation acts on is not followed. `UnderDir` compares through P2's `Fold`.
4. **Cancel** (step 4): `runMove`, `runDelete` and `runCopy` (`tool/builtin/fileops.go`) take their context.
   `copyEntry` checks it at the start of each call, so before each entry it copies, and `countEntries` before each
   entry it counts; each returns `context.Cause`. `moveCopy`, the cross-device move's copy, takes the context with it,
   and `TestMove`'s replacement takes the new signature. `RemoveAll` runs uninterrupted once started (MADR D11's note of 2026-10-30).
   `tool.Run` returns `context.Cause` whenever the context is done when the call returns, so a cancelled call is the
   cause, not a result.
5. **Tests** (step 5, deviation 12):
   * `internal/fsx/glob_test.go`: `TestGlobLinear`, F10's two patterns against 31 and 41 segments, each matching and
     not matching as before within 50 ms, and the run of nine `**` compiled to two segments. `TestGlobMatchesUnchanged`,
     a table of twenty patterns with runs of `**`, a trailing `**` and `**` between others; then every pattern of up to
     five segments from `**`, `a`, `*` and `b` against every path of up to six segments from `a` and `b`, 1,365 by 127,
     compared with `recursiveSegs`, the match it replaced, kept in the test.
   * `internal/fsx/ops_test.go`: `TestIsRootRealPath`, with the root a symlink: the link, its real path and the
     directory above are roots, a path under the real one is not; a path under the real `a` is `Inside` the link's `a`,
     and a sibling is not. P10's steps name no `IsRoot` test; `ops_test.go` is in its list.
   * `tool/builtin/fileops_test.go`: `TestCopyIntoItselfRealPath`, with the workspace a symlink and a 5 s deadline:
     copying `a` to `<real path of ws>/a/b` is refused, "a directory cannot be copied into itself", and nothing is made;
     `TestRealPathKeepsLinks`, deviation 12's two calls; `TestCopyHonoursCancel`, a tree of 40 directories of 49 files,
     2,000 entries, copied with a cancelled context, `context.Canceled` and fewer than 2,000 entries copied, then
     deleted recursively with it, `context.Canceled` and all 2,000 left.
   * Each symlink test skips, with its reason, where a symlink cannot be made (C10).
6. **Seen failing** (step 6, C9, deviation 12), each on a scratch copy:

   | Break | The failure |
   | :--- | :--- |
   | `glob.go` as at `ae80138`, with `collapse` the identity (the baseline) | `TestGlobLinear`: `the run of nine compiled to ["**" "**" "**" "**" "**" "**" "**" "**" "**" "z"], want [** z]`; `"**/**/**/**/**/**/**/**/**/z" against 31 segments took 3.461854916s, want within 50ms`; `"**/a/**/a/**/a/**/a/**/a/**/a/z" against 41 segments took 114.413791ms`. `TestGlob` and `TestGlobMatchesUnchanged` pass there |
   | today's recursive `matchSegs`, with `collapse` (the step's) | `TestGlobLinear`: `"**/a/**/a/**/a/**/a/**/a/**/a/z" against 41 segments took 113.176791ms, want within 50ms` |
   | no `collapse` | `TestGlobLinear`: `the run of nine compiled to ["**" "**" "**" "**" "**" "**" "**" "**" "**" "z"], want [** z]` |
   | a trailing `**` matching no segment | `TestGlobMatchesUnchanged`: `"a/**/**" matching "a" (dir true) = true, want false`, `"a/**/b/**" matching "a/b"`, and `["**"] against [] = true, want false` |
   | lexical `Inside` (the step's) | `TestCopyIntoItselfRealPath` (5.08 s): `copy gave context deadline exceeded, want its refusal` |
   | lexical `IsRoot` | `TestIsRootRealPath`: `IsRoot("<tmp>/real") = false, want true`, and for the directory above |
   | `RealPath` of the whole path, step 3 as written | `TestRealPathKeepsLinks`: `move l to d/l2: a directory cannot move into itself` |
   | the same in `IsRoot` alone | `TestRealPathKeepsLinks`: `delete rootlink: a workspace root cannot be deleted` |
   | `copyEntry` without its check | `TestCopyHonoursCancel` (27.12 s): `the cancelled copy made 2000 entries, want fewer than 2000` |
   | `countEntries` without its check | `TestCopyHonoursCancel`: `the cancelled delete left 0 entries, want 2000` |

7. **G-local:** `make pre-add-check` on the 6 Go files: clean; the first `make preflight` stopped at golangci-lint's
   `intrange` in `TestGlobMatchesUnchanged` (`for n := 0; n <= 6; n++`), made `for n := range 7`; then `make preflight`:
   passed; `go test ./...`: 38 ok; `make race`: 38 ok, both run before the loop's change, and `internal/fsx` and
   `tool/builtin` rerun under race after it: ok. P10 touches nothing under `internal/cli`, so no `make perf`.
8. **G-hosts,** 824 files shipped to each, after the loop's change: `go test` and race passed on both, 38 ok;
   `make check-records markdownlint` passed on both; preflight stops at `verify-build-metadata`, as on every copy.
   P10's six new tests, run alone with `-v`, passed on both hosts, none skipped: the Windows laptop makes symlinks.

### P11 record (2026-11-03): close-out

The base is P10's `5791512`.

1. **Goal** (step 1), each criterion against the records:

   | Criterion | Met by |
   | :--- | :--- |
   | an intercepted `mv` or `cp` into a directory does what the shell does; no directory is removed for a file (D1) | P1: `TestInterceptIntoDirectory`, `TestMakeRoomKeepsDirectory` |
   | no letter case, symlink, Windows spelling or bash command reaches a protected path without the limit (D2, D3) | P2: `TestProtectedFoldsCase`, `TestProtectedRealPath`, `TestFoldWindows`; P3: `TestBashWritesProtected`, `TestBashWritesConfig` |
   | a deny rule is not got round by quoting, case, a tab, an absolute program path or a symlink (D4, D5) | P4: `TestCommandBypasses`; P2: `TestPathRulesFoldCase`, `TestPathRulesRealPath` |
   | a write through a symlink writes its target, and hard links are kept (D6, Q1 (a)) | P5: `TestWriteThroughSymlink`, `TestWriteHardLinkInPlace`; a dangling symlink is replaced by a regular file, deviation 5 |
   | no file tool or configuration reader hangs on a FIFO, reads past its need, or holds a file it should not (D7, D10) | P6: the `…RefusesFIFO` tests, `TestReadStreamsLimit`, `TestCopyFileStreams`; P7: twelve readers through `ReadRegularFile` within 1 MiB, deviation 7 |
   | an edit refuses an ambiguous overlapping match and keeps the line endings it did not touch (D8) | P8: `TestEditOverlappingDuplicate`, `TestEditKeepsUntouchedEndings`, `TestEditLoneCR`, `TestEditLFOnly`, deviation 8 |
   | command output on disk is capped (D9) | P9: `TestSpillCap`, deviation 11 |
   | glob matching is linear; a recursive copy into itself, by any spelling, is refused and can be cancelled (D11) | P10: `TestGlobLinear`, `TestCopyIntoItselfRealPath`, `TestCopyHonoursCancel`, deviation 12 |
   | every gate passes on the three hosts at each phase | each record's G-local and G-hosts. The failures found on the way were deviation 2 (P2, Windows) and deviation 9 (P8, Windows), each fixed before its phase's commit, and the known `TestLastGoodOnDisk` teardown flake at P7, passing on its rerun |
   | the goldens are byte-identical throughout | **not met as written; met in substance.** `git diff --stat 2a8d70c..HEAD -- '*.golden'` is not empty: `internal/cli/render/testdata/style.golden` and `tables.golden` by `cce5ab0` (0035-PLAN) and `internal/cli/testdata/print-stream-json-escapes.golden` by `cfa514f` (0039-PLAN), both landed while this plan was paused after P2. `git show --stat` of each of this plan's ten commits, `e13d154`, `df92312`, `062909d`, `c4eb473`, `7114b5e`, `603bad6`, `e042fef`, `f7e1f1f`, `ae80138` and `5791512`, touches no golden |

   **Verification,** across the plan:
   * Every test the steps name exists in the tree but `TestLockWaitersSpread`, which deviation 10 replaced with
     `TestLockJitter`; every test the records name exists but the same one. Each record from P1 to P10 quotes its
     seen-failing runs.
   * `apidiff` is a no-op below v1.0.0, as P1 and P3 recorded. In its place, the exported declarations each of the ten
     commits adds or removes outside `internal/`, from `git show`: `tool.Env.Rerouted` (`e13d154`), and
     `permission.Rule.Writes` and `tool.Subject.Writes` (`062909d`); the others none. That is the list the Verification
     section gives.
   * `make perf` and `make probe-conhost` ran where `internal/cli` changed, P7's `e042fef` alone among the ten: `make perf`
     on all three hosts, `make probe-conhost` on the Windows laptop, all passing.
2. **Status** (step 2): this plan is `complete`. The MADR stays `accepted`, with its plans 2 and 3 to follow (D34).
   "Rollout and Rollback"'s list of what users see gains P10's line. `docs/architecture.md` is unchanged: P3, P4 and P9
   wrote their own lines, and none of its statements on settings, locks, globs, the file tools or the spill file is
   contradicted by P5–P10.
3. **The disclosure guard** (step 3), run as `AGENTS.md` gives it over the outgoing commits, `origin/main` `5e41482` to
   `5791512` (`062909d` to `5791512`, eight commits): exit 0, no output. This phase's commit is outgoing too; the
   pre-push hook runs the same guard over it. Nothing was pushed.
4. **Records** (step 4): `make check-records markdownlint` passed.
