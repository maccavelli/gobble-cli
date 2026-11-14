---
status: accepted
date: 2026-11-14
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Harden the permission rules and file tools, keep state whole and secrets out of logs, and speed up the paths that grow

## Context and Problem Statement

0035's P12 waits for acp-go-sdk2#12, and P17, P18 and P19 wait for go-tui-lib#13
([0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md),
deviations 9 and 11). In that gap the owner asked, on 2026-10-30: "is there anything we can take a look at to see if we
can find opportunities to improve code efficiency, better optimize, enhance hardening and performance, or leverage
heuristics and algorithms to make the codebase more robust and performant?" Once the survey below had answered, the
owner asked for "a single, detailed, accurate, expanded, and complete madr covering all 3 catagories", with a plan per
category to follow from it.

The survey found three kinds of problem in the code that stays gobble's:

1. **The permission rules and the file tools can be got round, or do damage.**
   * A `mv -f` into a directory deletes the directory.
   * The `.git` protection and deny rules can be got round by letter case, a symlink or a quoted word.
   * An edit or a write through a symlink replaces the link.
   * Some inputs hang the tools, or take exponential time.
2. **Some state drifts from the disk, some inputs are unbounded, and secrets reach the log.**
   * After a failed turn the live history is not the file's.
   * A short write merges two session entries.
   * One large `@path` or MCP result can end a session.
   * Bash commands carrying credentials are logged at the default level.
3. **Some paths cost more than they need to, and grow with the tree or the session.**
   * Ignore matching dominates a walk.
   * Listing sessions decodes every file.
   * Permission patterns are compiled on every match.
   * The loop guard and the compaction estimate use heuristics that miss known cases.

This record decides what gobble does about each finding. It decides nothing about the code that 0035 replaces with
go-tui-lib (`internal/cli/editor`, `internal/cli/render`, `internal/cli/transcript.go`). Nor does it reimplement what a
sibling library provides: AGENTS.md's rule is that gobble waits for the library.

### How the code was surveyed, 2026-10-30

* **Who.** Four read-only surveys ran in parallel, one per area:
  * the tools and the filesystem layer: `tool/builtin`, `internal/fsx`, `tool`, `tool/toolsearch`, `internal/proctree`,
    `internal/filelock`;
  * the agent loop: `acpserver`, `agent`, `compaction`, `prompt`, `personality`, `skill`, `hook`, `command`;
  * persistence and configuration: `session`, `session/jsonl`, `internal/config`, `internal/auth`, `internal/appdirs`,
    `internal/logging`, `internal/frontmatter`, `internal/buildinfo`, `gobble`;
  * providers, MCP and permissions: `llm`, `llm/provider`, `llm/catalog`, `mcpclient`, `permission`, `acpclient`.
* **How.** Each survey read the code, and probed and benchmarked on a scratch copy of the tree outside the repository.
  Nothing in the repository was changed.
  * The base is `61f7115`, with the working tree holding 0035's uncommitted P17, which no finding touches.
  * Figures are from darwin/arm64, Apple M1 Pro, Go 1.27.3.
  * The probes and benchmarks were throwaway. Each plan rebuilds the one it needs as a committed test, seen failing first.
* **Checked.** The top claims of each survey were read again at the cited lines before this record was written. One
  claim did not hold and is not used: the tools survey listed `tool/builtin/editstrategy.go:125-131` as a separate
  counting fault. It advances past each match, so it is the same non-overlapping count as F7, not a second fault.
* **Evidence key**, used on every fact below:
  * **M**, measured, with the figure;
  * **T**, shown by a probe test on the scratch copy;
  * **R**, verified by reading the cited lines;
  * **I**, inferred, not shown.

### Category 1 — the permission rules and the file tools

* **F1. `mv -f file dir` and `cp -f file dir` delete the directory (T, R).**
  * `tool/builtin/intercept.go:100-115` sends `mv [-f] a b` and `cp [-r|-R|-f] a b` to the native `move` and `copy`
    tools with `to: b`. `-f` sets `overwrite`.
  * `makeRoom` (`tool/builtin/fileops.go:226-240`) calls `ws.RemoveAll(abs)` on an existing destination when `overwrite`
    is set.
  * A shell puts the file *inside* an existing directory `b`. Intercepted, `mv -f f.txt d` answered "moved f.txt to
    d": `d/precious.txt` was gone and `d` was a regular file. `cp -f f.txt d` did the same.
  * Without `-f`, `mv f d` and `cp -r src d` into an existing directory are refused with "already exists". That is
    wrong, but nothing is lost.
  * [0012-MADR-tool-schemas-descriptions-and-loading.md](0012-MADR-tool-schemas-descriptions-and-loading.md) D10 holds
    that only exact simple commands are caught and that the flags map exactly. They do not map exactly when the last
    operand is an existing directory.
* **F2. The protected paths can be got round by letter case, an in-repo symlink, or bash (M, R).**
  * `permission/rules.go:292-310`:
    * `protected` compares the `.git` element exactly;
    * `configPath` compares `filepath.Base(p) == "AGENTS.md"` exactly;
    * `samePath` (`permission/trust.go:276-284`) ignores case only on Windows.
  * The limit covers only the native writers (`permission/rules.go:161-162`: `write`, `edit`, `move`, `delete`,
    `copy`, `mkdir`).
  * With only `allow: write`, on macOS's default case-insensitive volume:
    * `.git/config` was denied;
    * `.GIT/config` and `.Git/config` were allowed, and the write landed in the real `.git/config`;
    * `Agents.md` and `.Gobble/settings.yaml` were allowed with no question, where `AGENTS.md` asks;
    * a symlink `docs/x -> ../.git` made `docs/x/config` allowed, and the write landed in `.git/config`;
    * under `allow: bash(echo *)` and `bash(sed *)`, `echo '[core]' > .git/config` and `sed -i … .git/config` were
      allowed.
  * A `.git/config` with `core.fsmonitor` or `core.hooksPath` runs a command on the user's next `git`. In accept-edits
    mode the edit tools are allowed by default (`acpserver/rules.go:166-168`). A malicious repository can commit the
    symlink.
  * On Windows, a trailing dot or space in an element (`.git./config`) is not compared away (I).
* **F3. A `deny: bash(rm *)` rule can be got round by quoting, case, a tab or a path (M, R).**
  * A command's subject is its raw source text (`tool/builtin/shellparse.go:99`, `subjectText`). `wildcard`
    (`permission/rules.go:110-137`) ignores case only on Windows.
  * Rules `allow: bash` then `deny: bash(rm *)`:

    | Result | Commands |
    | --- | --- |
    | allowed | `RM -rf build` (on macOS `type RM` finds `/bin/RM`), `\rm -rf build`, `"rm" -rf build`, `r''m -rf build`, `rm<TAB>-rf build`, `/bin/rm -rf build`, `echo build \| xargs rm -rf`, `find build -delete` |
    | denied | `rm -rf build`, `a && rm …`, `echo $(rm …)`, `command`/`env`/`sudo rm …`, `FOO=1 rm …` |

  * Shell splitting is sound: `&&`, `;`, `|`, `$(…)`, the wrappers and `bash -c` all become separate subjects, and the
    strictest verdict wins.
* **F4. Path rules ignore case and symlinks (M for case, R for symlinks).**
  * `patternMatches` (`permission/rules.go:218-230`) matches with `fsx.CompileGlob` case-sensitively.
    `relative` (`:234-241`) is lexical.
  * With `deny: read(.env)` and `deny: edit(Secrets/**)` written in lower case, `read /r/.ENV` and `edit /r/Secrets/k`
    were allowed while the lower-case forms were denied.
  * The project store's lookup uses the same case-sensitive `samePath` on macOS. A working directory spelled in another
    case does not load the project's saved "reject always" rules.
* **F5. A write through a symlink replaces the link (T, R).**
  * `writeAtomic` (`internal/fsx/write.go:63-101`) renames its temp file onto the given path.
  * With `CLAUDE.md -> AGENTS.md`, an edit of `CLAUDE.md` left a regular `-rw-r--r--` file and `AGENTS.md` unchanged.
    A hard link breaks the same way (T).
  * The parent directory is not synced after the rename (R).
* **F6. `read`, `grep` and `copy` hang on a FIFO, and `read` loads the whole file (M, R).**
  * The readers do no file-type check:
    * `read` (`tool/builtin/read.go:41-72`) reads the whole file;
    * `grep`'s `candidate` (`tool/builtin/grep.go:321-333`) reads any file under 8 MiB, and a FIFO's size is 0;
    * `CopyFile` (`internal/fsx/ops.go:102-115`) reads the whole file into memory.
  * With a 500 ms context, `read` of a FIFO, and `grep` over a directory holding one, were still blocked after 2 s. The
    goroutine leaks.
  * A 64 MB file read with `limit: 50` allocated 151 MB in 35 ms.
  * `@path` mentions already require a regular file
    ([0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md) F3-4; `acpserver/mentions.go:100`, `mentionPath`).
* **F7. An edit's uniqueness check misses overlapping matches, and an edit rewrites line endings it did not touch (T, R).**
  * `locate` (`tool/builtin/editmatch.go:186-197`) counts with `strings.Count`, which counts non-overlapping matches.
    In `x\n}\n}\n}\n`, the old text `}\n}\n` occurs twice, but the edit was accepted as unique and changed the first.
  * `lineEnding`, `toLF` and `restoreEnding` (`editmatch.go:35-56`) give every line the first newline's ending:
    * in `a\r\nb\nc\n`, editing `b` to `B` gave `a\r\nB\r\nc\r\n`;
    * a lone `\r` inside a line became a line break.
  * Pi has the same behaviour; gobble inherited it.
* **F8. Spilled command output has no size cap (M, R).** `tool/builtin/output.go:57-61` writes every byte after the
  spill to the file.
  * `yes … | head -c 300000000` spilled at 467 MB/s.
  * At that rate the default 2-minute timeout writes about 56 GB to the state directory, and the 10-minute maximum about
    280 GB.
* **F9. A settings file is read whole, before trust, whatever kind of file it is (M, R).**
  * `readLayer` (`internal/config/resolve.go:130-131`) uses `os.ReadFile`, for the global and the project layer. The
    untrusted project layer is read to list its disabled keys (`:101-103`).
  * A FIFO as `.gobble/settings.yaml` blocked `Resolve` past 2 s.
  * A 32 MiB file took 517 ms and allocated 247 MiB.
  * A committed symlink to `/dev/zero` would grow until memory ran out (I). Git cannot commit a FIFO, but it can commit
    that symlink.
* **F10. `**` glob matching is exponential (M, R).** `matchSegs` (`internal/fsx/glob.go:103-129`) recurses at every
  split for every `**`.
  * One match of `**/**/**/**/**/**/**/**/**/z` against 31 segments took 6.67 s. With seven `**` it took 154 ms.
  * `**/a/` six times, then `z`, against 41 segments took 216 ms.
  * The pattern can come from a model's `find` or `grep` glob, or from a cloned repository's `.gitignore`. The walk
    checks its context only between entries, so a cancel cannot stop one match.
* **F11. A copy or move into itself, through another spelling of the path, runs away (T, R).**
  * `IsRoot` and `Inside` (`internal/fsx/ops.go:159-168`, `:194-196`) compare paths lexically.
  * `copyEntry` and `countEntries` (`tool/builtin/fileops.go:252-299`) take no context.
  * Copying `ws/a` to `<real path of ws>/a/b` (on macOS `/var` is `/private/var`) built a tree 461 levels deep in 230 s,
    until it was killed.

### Category 2 — state integrity, bounds and secrets

* **F12. One large `@path`, or one large pasted message, can end a session (R, traced).**
  * `mentionFiles` (`acpserver/mentions.go:81-96`) reads the file whole and puts it in the user entry.
  * A compaction summary carries user text uncut (`compaction/serialize.go:170-172`).
  * The overflow compaction keeps the newest turn's start. The summary request then overflows, and only tool results can
    be omitted, so it ends in `errTooLarge` (`compaction/compaction.go:413`). `pauseOf` (`acpserver/autocompact.go:65-72`)
    then pauses automatic compaction for the session.
  * From then on every prompt overflows, and every load reads the huge line again.
  * 0005-PLAN F3-4 sets no size limit.
* **F13. MCP results, descriptions and schemas are unbounded (R).**
  * `mcpTool.Run` returns the whole text (`mcpclient/tool.go:118-122`), up to the MCP go-sdk's 16 MiB frame.
  * [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) decides "Output over 20 KB is middle-truncated, with
    the full text in a file", and 0005-PLAN F8 step 2 carries it. It is not built.
  * A result that large is sent again on every later model call. When it is the newest entry, the overflow compaction
    keeps it, because a tool result is not a cut point.
  * Descriptions and schemas (`mcpclient/tool.go:47-81`) are sent on every call and are the server's untrusted text.
  * Resource reads, by contrast, are capped at 50 KB (`tool/builtin/resources.go:215`).
* **F14. Closing an MCP server waits as long as a grandchild holds its stderr (M, R).**
  * `cmd.Stderr` is not a file (`mcpclient/conn.go:198-201`), so `cmd.Wait` waits for the copy, and `cmd.WaitDelay` is
    not set. `stopper.Close` waits on `<-s.done` after the kill (`mcpclient/proc.go:84-95`).
  * With a grandchild that called `setsid` and held stderr for 8 s, `Close` took 7.75 s. With `WaitDelay` at 1 s it took
    1.00 s.
  * A long-lived daemon makes the wait unbounded, which hangs `Manager.Close`, `gobble mcp list`, session close and exit.
* **F15. After a failed turn, the live history is not the file's (M, R).**
  * While a turn runs, `w.event` writes each assistant message and tool result to the file. On an error with no
    compaction, only `prompt` is added to `s.history` (`acpserver/agent.go:712-727`). The append-failure exit
    (`:730-733`) does not update it either.
  * A model call failing with 400 after a `read` left the next request as `system, user:"first", user:"second"`, the
    read and its result missing.
  * A reload would show them, so the live session and the reloaded one disagree. The model can redo an edit it already
    made.
* **F16. A short write merges the next session entry into the broken line (M, R).**
  * `fileLog.Append` (`session/jsonl/store.go:303-325`) adds the entry to `l.entries` before writing it. `write`
    (`:367-379`) returns the error but never sets `l.torn`.
  * A half-written line followed by one more `Append` reloaded as `entries=1 skipped=1`: the next entry was written
    onto the partial line and lost with it.
  * Memory then holds an entry the file does not. `create` can leave a partial header the same way.
* **F17. Secrets reach the log at the default level (M, R).**
  * `logDecision` (`acpserver/rules.go:218-230`) logs `slog.Any("subjects", r.Subjects)` at Info, and `--log-level`
    defaults to `info` (`internal/cli/root.go:20`).
  * `redact` (`internal/logging/redact.go:18-35`) matches attribute names only: `authorization` exactly, or a name
    ending in `key`, `token`, `secret` or `password`.
  * Logged subjects keep the secret in two shapes:
    * a heredoc writing `OPENAI_API_KEY=sk-live-abc123` was logged as the subject
      `cat > .env <<EOF_OPENAI_API_KEY=sk-live-abc123_EOF`;
    * `curl -H "Authorization: Bearer ghp_SECRET" …` was kept verbatim.
  * A heredoc body of any size goes in whole. The log keeps 5 backups for 28 days (`internal/logging/rolling.go:24-25`).
    It is owner-only, but it is what people attach to bug reports.
* **F18. `auth.json` is replaced with no Windows retry and no directory sync (R).**
  * `internal/auth/store.go:390-403` uses a bare `os.Rename`. `fsx`'s writer retries a rename that Windows refuses for
    up to 2 s (`internal/fsx/write.go:103-122`).
  * [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md) dropped the directory sync for settings
    because "Settings hold no secrets (D15)". The credential file is the case that reasoning keeps it for.
  * On Windows, a scanner or another gobble process holding the file makes the rename fail. A refreshed OAuth login is
    then discarded (I for how often). If the provider rotates refresh tokens, the next run must sign in again.
  * `internal/cli/choice.go:245` has the same bare rename.
* **F19. Log rotation fails on Windows while a second process holds the log; the log then grows without bound (M with
  the failure injected, R).**
  * Every process shares `<state>/logs/gobble.log`. Go on Windows opens files without `FILE_SHARE_DELETE`, so
    `rotate`'s rename (`internal/logging/rolling.go:121-140`) fails while another process has the log open.
  * `open` then sets the size counter over the limit, so every later write retries the whole rotation.
  * Per record that is 2.4 µs for a plain write and 108–266 µs with the failing rotation. The file reached 1.79 MB
    against a 1 KB limit.
  * On Unix each process keeps its own counter, so the bound is approximate (I).
* **F20. The config directory is created group-readable (M, R).**
  * `lockSettings` (`internal/config/edit.go:89-92`) makes the directory `0750`, and first-run `config.Init` reaches it
    first. After it, under umask 022, the directory was `rwxr-x---`.
  * The records say otherwise:
    * 0005-PLAN-v1-feature-scope.md's line for `internal/auth` says `auth.json` is "created 0600 in a 0700 directory";
    * [docs/architecture.md](../architecture.md) says `internal/appdirs` creates its directories owner-only.
  * The file's own mode stays 0600, so only the names in the directory are exposed.
  * `appdirs.FileIsOwnerOnly` exists, but nothing calls it.
* **F21. `gobble mcp list` writes a server's text to the terminal uncleaned (R).** `listText`
  (`internal/cli/mcp.go:279-296`) writes `s.Tools` and `s.Error`, which holds up to 2000 runes of the server's stderr,
  with no `term.Clean`. A server can write OSC 52, OSC 8 or cursor moves. JSON mode is safe.
* **F22. Two small robustness gaps (R).**
  * `failureKinds` (`llm/provider/list.go:107-120`) has no `llm.ErrOffline`, so the SDK's offline retries show the
    cause "other".
  * `acpclient`'s handler never deletes a session's `commands` entry. An update for an unknown session during a failed
    `session/new` stays mapped to that call's callback (`acpclient/conn.go:386-402`). That is a few KB per `/new`.

### Category 3 — performance and heuristics

* **F23. Ignore matching makes a walk cost entries × depth × rules × depth (M, R).**
  * `Walker.ReadDir` calls `Ignored` for each entry (`internal/fsx/walk.go:94`). `Ignored` matches every ancestor again
    (`internal/fsx/ignore.go:31-41`), though the walk has already accepted them.
  * `match` splits the path once per rule (`glob.go:100`). A `**/x` rule tries every depth.
  * The test tree had 10,920 files, 6 levels deep, and a 132-rule `.gitignore` (GitHub's Node and Go templates).

    | Path | Time | Memory |
    | --- | --- | --- |
    | `Walk`, today | 7.87 s | 1.12 GB in 14.7 M allocations |
    | A `grep` that matched nothing, today | 8.78 s | |
    | A prototype: no ancestor re-check, one split per set, literal segments as strings, a `**/x` tail at its one position | 1.29 s | 72 MB in 1.87 M allocations |

    `internal/fsx`'s tests passed against the prototype.
* **F24. Listing sessions reads and decodes every file before filtering by cwd (M, R).**
  * `Store.List` (`session/jsonl/store.go:233-258`) decodes each file whole, then compares `h.Cwd`.
  * It is called by shell completion on every TAB (`internal/cli/completion.go:95`) and by ACP `session/list`
    (`acpserver/agent.go:1086`).
  * Measured on 500 sessions, 67 MB, 20 cwds:

    | Path | Time | Memory |
    | --- | --- | --- |
    | `List`, today | 503 ms | 309 MB |
    | Listing one cwd, today | 481 ms | |
    | Reading only that cwd's 25 files | 10 ms | |
    | A trimmed decoder of only what `Summarize` reads | 195 ms | 37 MB |

* **F25. Permission patterns are compiled on every match (M, R).**
  * `wildcard` builds and compiles a regexp per call (`permission/rules.go:113-137`), and `patternMatches` compiles a
    glob per call (`:224`).
  * They are reached once per tool by `Hidden`, in `visible` (`acpserver/agent.go:669`) and in the system prompt's
    sections.

    | Benchmark | Today | With a cache |
    | --- | --- | --- |
    | One tool-name wildcard | 1752 ns, 44 allocations | 40 ns, 0 allocations |
    | One command wildcard | 3539 ns | 149 ns |
    | `Hidden` × 100 tools, 200 rules | 34 ms, 59 MB | about 0.8 ms (estimated) |
    | `Evaluate`, 3 commands, 200 rules | 3.06 ms | not measured |

  * Every "always" answer adds an exact rule, so a project store grows (I).
* **F26. Edit's forgiving layer runs four regexps, again for each edit (M, R).**
  * `fuzzy` (`tool/builtin/editmatch.go:68-79`) runs four `ReplaceAllString` passes. It is called on the whole content
    inside the per-edit check, and again (`:147-152`).
  * For a 1 MB file, `fuzzy` takes 41 ms, of which 40 ms is the regexps and 0.3 ms is NFKC.
  * One `strings.Map` pass takes 1.5 ms, its output equal on a string covering the quote, dash and space ranges.
  * A five-edit call through the layer takes 214 ms and 48 MB.
* **F27. `grep` tests the pattern on every line of every file (M, R).** `grepper.file` (`tool/builtin/grep.go:163-172`)
  splits each file and calls `MatchString` per line. Over the repository's 585 Go files, a search took 6.18 ms and 5 MB,
  against 0.40 ms and 15 KB with a whole-buffer test first.
* **F28. Work before each model call grows with the whole session (M, R).**
  * Each trigger copies the whole path (`Tree.Path`), decodes each baseline's JSON and projects the whole context.
  * `s.state()` replays the whole session:
    * on every `_gobble/get_state` (`acpserver/extensions.go:464-469`), to fill `MessageCount`;
    * in `titleWanted` (`acpserver/title.go:50-58`), before its cheaper `HasConversation` test.
  * A synthetic session, with a 19 KB baseline and a 4 KB result per step:

    | Work | 300 steps | 3,000 steps |
    | --- | --- | --- |
    | per-call trigger | 1.6 ms, 2.7 MB | 8.2 ms, 33 MB |
    | `Path()` | 0.44 ms, 1.1 MB | 3.4 ms, 14.6 MB |
    | `s.state()` | 10 ms, 6 MB | 81 ms, 67 MB |

  * Skills are loaded twice per prompt.
* **F29. `os.Root` resolves a path one directory at a time, and the recursive operations open a root per entry (M, R).**
  * A file 9 levels deep took 243 µs through the workspace root, and 31 µs through a root opened at its directory.
  * `Workspace.Stat` took 250 µs, against 2 µs for `os.Stat`.
  * With F23 fixed, a walk is 90% system calls, 76% of them `openat`.
* **F30. The loop guard catches only a call repeated back to back (M, R).**
  * `repeated` (`agent/agent.go:921-938`) compares a call with the one before it. `repeatLimit` is 3 (`:894`). The
    bad-arguments breaker counts only `BadArguments` results.
  * 30 replies alternating `ping{n:0}` and `ping{n:1}` ran all 30 calls with no question.
  * A read, a failing edit, a read, a failing edit, …, and an edit retried with a slightly changed old text, both run to
    `DefaultMaxCalls`, 100 (`agent/agent.go:23`). Each call sends the whole context again.
  * 0005-PLAN F5-3 decision 1 took the back-to-back rule from opencode.
* **F31. The compaction estimate counts the system prompt twice when anchored, and leaves out tool specs when not
  (R, sizes M).**
  * `used := compaction.EstimateText(a.systemText(s)) + compaction.EstimateEntries(s.tree.Path())` appears three times:
    `acpserver/autocompact.go:111`, `acpserver/compact.go:136` and `acpserver/commands.go:725`.
  * `EstimateEntries` anchors on the last assistant usage (`compaction/estimate.go:309-320`), whose input tokens
    already include the system prompt and the tool specs.
  * In this repository:
    * the system prompt is 11.3 KB, about 2,829 estimated tokens, counted twice whenever the estimate is anchored;
    * the 12 direct tool specs are 15.4 KB, about 3,841 tokens, missing whenever it is not anchored.
  * On a 32,768-token window with the default 16,384 reserve, the double count fires compaction about 17% early. On
    200 k it is about 1.5%.
  * chars ÷ 4 also undercounts code, JSON and CJK text (I).
* **F32. Summary requests send thinking and tool-call arguments uncut (M on a small sample, R).**
  * `compaction/serialize.go:174-195` joins every thinking block and every call's whole arguments. Only tool results
    are cut, to `toolResultMaxChars`, 2,000 (`:44`). The omission ladder drops only tool results.
  * Over the owner's 27 local session files (counts printed, nothing else read out):
    * thinking was 24.5% of 346,990 serialised characters;
    * tool arguments were 14.6%;
    * cut tool results were 41%.
  * The sessions are short and mostly tests, so the shares are indicative only.

### Looked at and found sound

* `internal/filelock`: `flock`, and `LockFileEx` past the data, honour the context.
* `internal/proctree`: the Unix process group and the Windows kill-on-close job are sound.
* `fsx.Lock` reference counting is correct.
* Windows reserved names are refused through `filepath.IsLocal`.
* Shell splitting for permissions is sound (F3).
* Prompt-cache stability is sound:
  * the tool order is deterministic;
  * loaded tools are appended last;
  * the system prompt is a fixed baseline, with patch messages for changes;
  * the prompt has no time-varying text.

  *(2026-11-14, by the amendment of that date: within one process. Across a resume it is not: F33.)*
* Usage is not counted twice. Orphaned tool calls get a result.
* The retry goroutines and timers end.
* The ACP connection's shutdown and cancellation are sound (0035 P11).
* YAML parsing is sound:
  * `frontmatter.Parse` stops a billion-laughs document in 0.6 ms with yaml.v3's "excessive aliasing";
  * config's leaf walk does not follow aliases into a blow-up.
* The clocks are sound: deadlines are monotonic, and stamps are UTC.
* `llm/catalog` parses once, and its lookups are trivial.
* No credential is logged by the provider layer: `Login.LogValue` redacts it, and the Gemini key goes in a header.
* `go test -race` over `agent`, `compaction`, `acpserver`, `prompt`, `skill`, `personality`, `command` and `hook` was
  clean.

### What belongs elsewhere

* **Upstream, in go-llmprovider-sdk.** `withHeaders` (`llm/provider/provider.go:150-160`) builds its client on
  `http.DefaultTransport` when `models.json` sets headers. That drops the SDK client's 300 s response-header timeout, so
  a gateway that accepts and never answers hangs the turn (R, the survey's reading of the SDK). The fix is an SDK option
  to add request headers to its own client: [go-llmprovider-sdk#10](https://github.com/maccavelli/go-llmprovider-sdk/issues/10).

  *(2026-11-03: go-llmprovider-sdk `v1.25.0` answered it with `llmprovider.NewHTTPClient`, and
  [0041-MADR-models-json-headers-on-sdk-http-client.md](0041-MADR-models-json-headers-on-sdk-http-client.md) adopts it:
  `withHeaders` wraps the SDK's own client when given none.)*
* **Upstream, in acp-go-sdk2.** The client drops the newest notification when its queue is full
  ([0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md), amendment of 2026-10-01). Back-pressure
  is the SDK's to offer: [acp-go-sdk2#13](https://github.com/maccavelli/acp-go-sdk2/issues/13).
* **Decided already, and not reopened here:**
  * parallel tool batches, a 1.0 decision in 0005-MADR built by 0005-PLAN;
  * MCP's reconnect, `list_changed` and start-up wait (0005-PLAN F8 steps 3 to 5);
  * MCP stdio servers inheriting the environment (0002-PLAN);
  * `!command` header values not cached (0005-PLAN F4-2b).
* **Noted and not taken:**
  * session appends are one write per line with no fsync, which survives a process crash and loses at most the tail on
    power loss *(2026-11-04: now a decision, D39, Q10 (a))*;
  * `proctree`'s `Exited` sends only SIGTERM to an orphan;
  * `toolsearch` recomputes document frequency per search, which is trivial at real tool counts;
  * `go test ./internal/auth` takes about 27 s because a test waits out the lock policy.

## Decision Drivers

* **Do no harm first.** A tool that deletes data, or a rule that can be got round into running code, outranks any cost
  in time.
* **Fail stricter.** Where a check cannot tell two readings apart, it takes the stricter verdict. That way a fix can
  only make a deny stronger, never weaker.
* **Bound what is untrusted.** Every byte from a file, a model, a server or a project goes through a size and type check
  before it is held in memory or sent on.
* **One source of truth.** The live session and its file never disagree, and memory never holds what the disk does not.
* **Measure, and keep it measured.** A performance change ships with the benchmark that shows it, held to a budget as
  [0030-MADR-resize-and-performance.md](0030-MADR-resize-and-performance.md) D5 holds the CLI's.
* **Wait for the library.** Where a sibling library owns the behaviour, gobble asks it, and does not copy it.
* **Keep the records true.** A decision here that changes an earlier one names it, and the earlier record gets a dated
  note when the plan carries it out.

## Considered Options

* **A. One decision over the three categories, carried out by three plans, safety first.**
* **B. Fix only what can do harm now** (F1–F3, F5, F6, F12, F15–F17), and leave the rest as a REPORT.
* **C. A MADR per category,** each with its own plan.
* **D. Record the findings as a REPORT, and decide nothing until the held 0035 phases are done.**

## Decision Outcome

Chosen option: **"A. One decision over the three categories, carried out by three plans, safety first"**:

* the findings share their evidence, and several share code (F10 under F23; F2 and F4 under F25);
* the owner asked for one record;
* three plans keep each category's work, review and rollback apart.

The decisions follow the categories. Each names the findings it answers. Numbers chosen here are starting values, and a
plan may tune one only through a deviation.

### Category 1 — the permission rules and the file tools

* **D1. An intercepted `mv` or `cp` does what the shell does with a directory (F1).**
  * When the last operand is an existing directory, the interception sends `to: <dir>/<base(from)>`. That is the
    shell's rule (not `-T`).
  * Independently, `makeRoom` never removes a directory to make room for a source that is not a directory. A directory
    may still replace a directory, as the native tool's `overwrite` does today.
  * 0012-MADR D10 gets a dated note.
* **D2. The protected paths and the config paths are checked as the filesystem resolves them (F2).**
  * Path elements are compared without regard to case on darwin and windows. A case-sensitive macOS volume is then
    protected more than it needs to be, never less.
  * Each check runs on the lexical path and on the real path, with the deepest existing ancestor resolved as `fsx`'s
    `realPath` does (exported for this). The stricter verdict is kept.
  * On Windows, trailing dots and spaces are stripped from each element before comparing.
* **D3. The protected-path limit also covers a bash call's literal write targets (F2).**
  * The targets are the output redirections, and the destination operands of the writing commands that `parseShell`
    already classifies. Plan 1 lists the commands from `tool/builtin/shellparse.go`.
  * A target computed at run time is not seen.
  * `docs/architecture.md` says that this protection is best-effort for bash, and why.
* **D4. A command is matched in its raw and its normalised form, and the stricter verdict wins (F3).**
  * The normalised form is the unquoted words joined by single spaces, the program's absolute path reduced to its base
    name, and the program word case-folded on darwin and windows.
  * The saved "always" patterns already use unquoted words, and are unchanged.
  * The docs say that `xargs` and `find -delete` are beyond pattern rules.
* **D5. Path rules match the lexical and the resolved path, without regard to case on darwin and windows, and the
  stricter verdict wins (F4).**
  * The project store's lookup folds case the same way, which can only add rules.
  * Trust grants stay exact: folding a grant would widen it on a case-sensitive volume, so a differently spelled cwd is
    asked again.

  *(2026-10-30, by 0037-PLAN-permission-and-file-tool-hardening.md's deviation 1: "the stricter verdict wins" is
  amended. A reading that no rule matches takes only the default. That default does not outrank a reading a rule
  matches. Per subject, the verdict is the strictest among the readings a rule matched, or the default when none
  matched. A deny for either spelling still denies, and an allow for either spelling still allows. As first written,
  with `/tmp` and `/var` symlinks on macOS, every absolute allow rule for such a path asked instead.)*

  *(2026-10-30, by the plan's deviation 2: "stay exact" means that trust grants compare as they did before 0037, never
  through `fsx.Fold`. On windows they have always ignored case (`permission/trust.go:282-283`), as NTFS does, and they
  still do.)*
* **D6. A write through a symlink writes the link's target, and keeps the link (F5).**
  * The target is resolved inside the workspace roots, and `os.Root` refuses an escape. The write is atomic beside the
    target.
  * A file with more than one hard link is written in place, which keeps the link: truncate, write, sync (Q1 (a)).
  * On Unix the parent directory is synced after the rename.
* **D7. The file tools read only regular files, and read only what they need (F6).**
  * `read`, `grep` and `copy` check the type of the opened file, not a separate `stat`. They refuse, or for `grep` skip,
    anything that is not a regular file.
  * `read` streams up to `offset + limit` and counts the remaining lines in fixed chunks.
  * `copy` streams through the atomic writer.

  *(2026-11-02, by 0037-PLAN-permission-and-file-tool-hardening.md's deviation 6: `grep` naming one file checks it the
  same way, and a file that is not regular is the call's error there, not a skip. `edit` and `write` share `read`'s
  local open, so they refuse such a file too, with no new size limit.)*
* **D8. Edits count overlapping matches, and keep the line endings they did not touch (F7).**
  * Every layer's count advances one byte after each match and stops at two.
  * A file with no `\r` is not converted at all.
  * A file with `\r` keeps each untouched line's own ending. Only the replaced span takes the file's dominant ending,
    as `replaceKeepingLines` already does for fuzzy matches. A lone `\r` is not a line break for this conversion.

  *(2026-11-02, by 0037-PLAN-permission-and-file-tool-hardening.md's deviation 8: layer 3's escape-normalised strategy
  counted only non-overlapping matches too, and now advances one byte after each.)*
* **D9. The spill file is capped at 256 MiB (F8).** Lines and bytes are still counted past the cap, and the result says
  the file was cut there.
* **D10. Every settings and configuration file is read as a regular file, through a 1 MiB limit (F9).**
  * That covers `settings.yaml`, global and project. Plan 1 audits `mcp.json`, `models.json`, `permissions.json` and
    the trust file, and applies the same rule to each.
  * A refused file is a diagnostic, as a malformed one is today.

  *(2026-11-02, by 0037-PLAN-permission-and-file-tool-hardening.md's deviation 7: `gobble config validate` and shell
  completion read the same files through the same reader, so neither blocks on a file gobble refuses.)*
* **D11. Glob matching is linear, and recursive file operations stop on cancel and compare real paths (F10, F11).**
  * At compile time, consecutive `**` collapse into one, as git's wildmatch does.
  * Matching is memoised on the pattern index and the name index, which makes it O(P·N). The prototype took
    0.5–54 µs for F10's inputs.
  * `IsRoot` and `Inside` compare real paths.
  * `copyEntry`, `countEntries` and the recursive delete check the context per entry.

  *(2026-10-30, found while writing plan 1: the delete's removal is `os.Root.RemoveAll`, which takes no context and
  cannot be stopped part-way. The recursive delete checks the context in its count, `countEntries`, before it removes
  anything. The removal, once started, runs to its end.)*

  *(2026-11-03, by 0037-PLAN-permission-and-file-tool-hardening.md's deviation 12: the path an operation acts on is
  real up to its own name, which is not followed, since move, delete and copy act on a link itself; a root is real
  whole, and the lexical check stays.)*

### Category 2 — state integrity, bounds and secrets

* **D12. A mention is bounded, and so is user text in a summary request (F12).**
  * An `@path` file is checked with `stat` before it is read, and read through a limit. Over the `read` tool's limits
    (2,000 lines or 50 KiB, `tool/builtin/builtin.go:22-23`), it is inserted cut to them, with a line naming its size and
    pointing the model at `read` for the rest.
  * In a summary request, user text over 16,000 characters is cut to its head and tail with a marker.
  * The omission ladder can drop user text that remains too large, before it gives up with `errTooLarge`.
* **D13. MCP results are cut at 20 KB, and every tool result has a backstop (F13).**
  * This carries out 0005-PLAN F8 step 2's truncation now: a 20 KB middle cut, with the full text written to the tool
    output directory as `bash`'s spill is.
  * `agent` caps any one tool result at 256 KiB, as a last resort for third-party tools.
  * An MCP description is cut at 2,048 characters. A tool whose schema is over 64 KiB is not offered, and the reason
    is shown in `gobble mcp list`.
  * 0005-PLAN F8 gets a dated note that 0037 carried out the truncation.
* **D14. An MCP server's process wait is bounded (F14).** `cmd.WaitDelay` is set to the server's `StopTimeout`.
* **D15. Every way a turn can end leaves the live history equal to the file's (F15).** Each error exit sets
  `s.history` from `s.state()`, as the compacted branch already does. The replay costs about 10 ms at 300 steps, and
  runs only on failure.

  *(2026-11-14: carried out by `0037-PLAN-resume-prefix-and-failed-turn-history.md`, with D42, not by plan 2 (D44).)*
* **D16. A session entry is in memory only once it is wholly on disk (F16).**
  * `Append` and `create` note the file's size before writing. On an error they truncate back to it, or, when the
    truncate fails, mark the log torn.
  * The entry joins `l.entries` only after the write succeeds.
  * `Open`, which holds the writer's lock, cuts an unparseable last line that has no newline, a torn write. It logs the
    bytes cut once.

  *(2026-11-04: carried out by `0037-PLAN-session-store-durability.md`, not plan 2 (D36). How it is built, including
  the failed `create` and the Windows truncate, is D37.)*
* **D17. The log redacts secrets by value, and caps long strings (F17).**
  * `redact` keeps its key rule, and adds `cookie`, `set-cookie`, `proxy-authorization`, `credential` and
    `credentials`.
  * It replaces, inside any string value:
    * `Bearer` and `Basic` credentials;
    * the common token prefixes (`sk-`, `ghp_`, `gho_`, `github_pat_`, `glpat-`, `xox?-`, `AKIA`);
    * `NAME=value` where NAME ends in `KEY`, `TOKEN`, `SECRET` or `PASSWORD`;
    * the password of a URL's user information.
  * Any string attribute over 512 bytes is cut, with a marker.
  * The permission decision log stays at Info, by 0005-PLAN F6-2a's decision 12 and Q2 (a).
* **D18. `auth.json` is replaced through `fsx`'s retrying rename, and its directory is synced (F18).**
  * The owner-only creation and the Windows DACL are kept. The retrying rename is exported from `fsx` for this.
  * The parent directory is synced on Unix.
  * `internal/cli/choice.go:245` uses the same rename.
* **D19. A failed log rotation is not retried on every write, and on Windows the log is opened shareable for delete
  (F19).**
  * After a failure, rotation is tried again only once the file passes the next multiple of the limit.
  * On Windows the file is opened with `FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE` through
    `windows.CreateFile` and `os.NewFile`.
  * On Unix the per-process bound stays approximate. That is recorded, not fixed.
* **D20. The global config directory is created 0700, and `auth.json` is checked for its mode (F20).**
  * The directory is not created through `EnsurePrivateDir`, which refuses a symlinked leaf. A user who links the
    directory to dotfiles keeps working.
  * `auth.Load` warns once when `auth.json` is readable by group or others, through `appdirs.FileIsOwnerOnly`.
* **D21. `gobble mcp list` cleans a server's text for the terminal (F21).** It uses `term.Clean` with the terminal flag,
  as 0035 P15 does for model text.
* **D22. The two small gaps are closed (F22).**
  * `failureKinds` gains `llm.ErrOffline` as "offline", ahead of "unavailable".
  * `acpclient`'s handler deletes a session's `commands` entry on remove, and drops an unknown session's mapping when
    its `session/new` fails.

### Category 3 — performance and heuristics

* **D23. A walk matches each entry once, against rules split once (F23).** This builds on D11.
  * `Walker.ReadDir` uses `match` with no ancestor re-check.
  * The path is split once per ignore set, and literal segments are compared as strings. A `**/x` tail is matched at
    its one position.
  * `Walker.Ignored`, which the skill and prompt scans use, keeps the ancestor check.
* **D24. Listing sessions reads only what it lists (F24).**
  * With a cwd filter in the non-flat layout, only that cwd's directory is read. The header's cwd is still compared,
    because `EncodeCwd` maps `/a-b` and `/a/b` to the same name.
  * Summaries come from a streaming decoder that reads only what `Summarize` uses. A version-1 or version-2 file falls
    back to the full parse.
  * A summary cache keyed by path, size and mtime is not adopted now. It is the next step if lists stay slow.
* **D25. Permission patterns are compiled once (F25).**
  * A package-level cache keyed by kind, pattern and case rule holds the compiled matcher. A pattern with no wildcard is
    compared as a string.
  * `Spec` gains no field, because it is compared by value in `Remember` and `RuleStore.Add`.
  * This lands after D4 and D5, which change what is matched.
* **D26. Edit's forgiving layer runs once per call, in one pass (F26).**
  * `fuzzy(content)` is computed once per call, when first needed.
  * The four regexps become one rune map, with a test pinning its output against the regexps' over the four
    character ranges.
* **D27. `grep` tests a whole file before its lines (F27).**
  * A file is split into lines only when a `(?m)` whole-buffer test matches.
  * The test is used only when the pattern has no end-of-text anchor. That is decided from `regexp/syntax`, because
    lines are matched with their `\r` trimmed. Otherwise every line is tested, as today.
* **D28. Per-call work is proportional to what changed, not to the session (F28).**
  * `Tree` keeps the active path: appended on `Add`, rebuilt on `Branch` and `ResetLeaf`, and returned as a clipped
    shared slice.
  * The system text is cached by its baseline entry's id.
  * The anchored estimate walks back from the leaf only to the last usage.
  * `_gobble/get_state` reads a running message count.
  * `titleWanted` tests `HasConversation` before it replays.
  * Skills are loaded once per prompt.
* **D29. Workspace operations reuse an open root as they descend (F29).**
  * The walker keeps a root per directory.
  * Recursive copy, count and delete hold one root for the operation.
* **D30. The loop guard counts repeats per turn, and failures per tool (F30).**
  * Each canonical (name, arguments) key is counted for the whole turn. Its third occurrence asks the existing question,
    adjacent or not.
  * Five consecutive error results from the same tool, whatever their arguments, ask the same question.
  * This amends 0005-PLAN F5-3 decision 1. The thresholds are Q3 (a)'s.
* **D31. The compaction estimate counts each part once (F31).**
  * One function replaces the three copies. When anchored, it is the usage plus the estimate of the entries after it.
    When not anchored, it is the system text plus the encoded tool specs plus the entries, each at chars ÷ 4.
  * Calibration waits, by Q4 (a). "Revisit", under More Information, says when.
* **D32. Summary requests cut thinking and tool arguments as they cut tool results (F32).**
  * Each argument value over `toolResultMaxChars` is cut, except values that are paths.
  * Each message's thinking is cut to the same bound.
  * The omission ladder drops thinking, then arguments, before it gives up.
* **D33. Each performance decision ships with its benchmark, under a budget.**
  * Plan 3 adds benchmarks for D23–D28 behind the `perf` build tag, with baselines measured on macOS, the Linux machine
    and the Windows laptop.
  * Each is held to three times its baseline, as 0030-MADR D5 holds the CLI's.

### The plans

* **D34. Three plans, in this order:**
  1. **[0037-PLAN-permission-and-file-tool-hardening.md](0037-PLAN-permission-and-file-tool-hardening.md)**: D1–D11.
  2. **`0037-PLAN-state-integrity-bounds-and-secrets.md`**: D12–D22.
  3. **`0037-PLAN-performance-and-heuristics.md`**: D23–D33.

  Plan 3 needs plan 1, because D23 builds on D11 and D25 on D4 and D5. Plan 2 needs neither. Each plan is written,
  reviewed and approved on its own, after this record is accepted, and no code changes before its approval. The plan
  slugs are Q5 (a)'s. Each name becomes a link when its plan is written.

  Before plan 1's first phase, 0035's uncommitted P17 is committed to `feature/p17-lineedit`, leaving `main`'s tree
  clean (Q6 (a)).

  *(2026-11-04, by the amendment's D36 and Q8 (a): a fourth plan,
  [0037-PLAN-session-store-durability.md](0037-PLAN-session-store-durability.md), carries D16 and D37–D41 and needs
  none of the others. Plan 2 keeps D12–D15 and D17–D22.)*

  *(2026-11-14, by the amendment's D44: a fifth plan,
  [0037-PLAN-resume-prefix-and-failed-turn-history.md](0037-PLAN-resume-prefix-and-failed-turn-history.md), carries
  D15 and D42 and needs none of the others. Plan 2 keeps D12–D14 and D17–D22.)*
* **D35. The upstream items are the owner's issues, filed 2026-10-30:** [go-llmprovider-sdk#10](https://github.com/maccavelli/go-llmprovider-sdk/issues/10) for the header option, and [acp-go-sdk2#13](https://github.com/maccavelli/acp-go-sdk2/issues/13) for
  back-pressure (Q7). gobble adopts each when a tag carries it, through a record of its own.

  *(2026-11-03: both are adopted. acp-go-sdk2#13 by
  [0040-MADR-acpclient-notification-back-pressure.md](0040-MADR-acpclient-notification-back-pressure.md), and
  go-llmprovider-sdk#10 by [0041-MADR-models-json-headers-on-sdk-http-client.md](0041-MADR-models-json-headers-on-sdk-http-client.md).)*

### Questions for the owner

The owner answered each on 2026-10-30. The answer is noted under its question, and the decisions above are written to
it.

* **Q1. A file with more than one hard link (D6).**
  * **(a) Recommended.** Write it in place: truncate, write, sync. This keeps the link, as editors that copy back do. It
    gives up the atomic rename for that file only.
  * **(b)** Keep the atomic rename, and say in the result that the link was broken.
  * **(c)** Refuse the write, and tell the model to ask.

  *Answered: (a), "Write in place".*
* **Q2. The level of the permission decision log (D17).**
  * **(a) Recommended.** Keep Info, with D17's redaction and cap.
  * **(b)** Move the subjects to Debug, and keep the verdict at Info.
  * **(c)** Both: redact, and move the subjects to Debug.

  *Answered: (a), "Info, redacted".*
* **Q3. The loop guard's thresholds (D30).**
  * **(a) Recommended.** The third occurrence of a key in a turn, and five consecutive errors from one tool.
  * **(b)** Detect only cycles of period 2 or 3, over the last six calls.
  * **(c)** Keep the back-to-back rule, and add only the error count.

  *Answered: (a), "Per-turn count + errors".*
* **Q4. Calibrating the estimate (D31).**
  * **(a) Recommended.** Fix the double count and the missing specs now. Calibrate later, if overflows are still seen.
  * **(b)** Also calibrate characters per token for each session from successive usage reports, smoothed.
  * **(c)** Fix only the double count.

  *Answered: (a), "Fix now, calibrate later".*
* **Q5. The plans' names.** AGENTS.md says "A MADR and its PLAN share the same number and the same slug".
  `madr-and-plan-writing` and `scripts/check_records.py` allow several plans under one MADR, each with a slug of its
  own.
  * **(a) Recommended.** Three plans with their own slugs, as D34 names them, and AGENTS.md's sentence amended to allow
    several plans. That edit falls under the bootstrap exception.
  * **(b)** One plan, `0037-PLAN-hardening-integrity-and-performance.md`, in three parts.
  * **(c)** Three MADRs, option C.

  *Answered: (a), "Three slugs + amend". AGENTS.md's "Records" section now says so.*
* **Q6. 0035's uncommitted P17 while this work lands.** The phases of these plans commit to `main`, and P17's change set
  is in the same working tree. Tests and gates run on the working tree, so with P17 present they would not test what is
  committed.
  * **(a) Recommended.** Commit P17's change set to its own branch, `feature/p17-lineedit`, so that `main`'s tree is
    what the gates test. When go-tui-lib#13 ships, P17 resumes there.
  * **(b)** Commit each phase by pathspec, and run each gate on a clean export of what is committed.
  * **(c)** Stash P17 while these plans run, and restore it afterwards.

  *Answered: (a), "P17 to a branch". The commit is made on the owner's word in its own turn, before plan 1's first
  phase.*
* **Q7. The upstream issues (D35).**
  * **(a) Recommended.** Draft both now, and file each on the owner's word.
  * **(b)** File neither until the plans are done.
  * **(c)** File only go-llmprovider-sdk's, which affects a user today.

  *Answered: the owner had already filed both on 2026-10-30, as [go-llmprovider-sdk#10](https://github.com/maccavelli/go-llmprovider-sdk/issues/10) and [acp-go-sdk2#13](https://github.com/maccavelli/acp-go-sdk2/issues/13), and both are being worked on.*

### Amendment (2026-11-04): the session store, and gobble-cli#7's U3

On 2026-11-04 the owner asked to assess [gobble-cli#7](https://github.com/maccavelli/gobble-cli/issues/7): "ensure it is
actually still open … if it is open we need to work on that to get it completed and optimized." Four of #7's five units
are settled by [0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md).
The fifth, U3 (the `session/jsonl` store's hardening), is not, and this record never received it:

* **The handoff did not land.** 0039-MADR D22 (2026-10-31) passed three items to this record: "an `fsync` policy and an
  entry size cap for the session store", and "0038's `bytes.Lines` for reading a session file", for the performance
  category. This record, written a day earlier, has no decision for any of them. Only D16 touches the store, and its
  plan (D34's plan 2) is not written.
* **The code, re-read on 2026-11-04 at `67f72fb`:**
  * `Append` adds the entry to `l.entries` before it writes (`session/jsonl/store.go:309`), and `write` returns a failed
    write's error without marking the log torn (`:375-377`). F16 stands as surveyed.
  * `create` deletes the pending id and marks the log written before its write (`:352-356`). A failed first write leaves
    a partial file and a log that believes it is on disk.
  * Nothing calls `Sync`. `parse` builds a slice of every line with `bytes.Split` before decoding one (`:192`).
  * No test kills a writer. `TestMalformedAndTornLines` (`session/jsonl/store_test.go:148-183`) plants a torn line by
    hand and expects it skipped, not cut.
* **#7's size cap rests on a misreading.** #7 asks for a write-side cap "mirroring the 1 MiB read cap mcremote applies"
  on engine frames. magic-cli-remote's 1 MiB is its WebSocket frame to the phone
  (magic-cli-remote `internal/provider/output/output.go:15`, `internal/config/config.go:106-109`). magic-cli-remote does
  not read gobble's session files, and gobble's own reader reads a file whole, with no line limit (`store.go:172`). One
  legitimate entry can be large: `read` loads an image of up to 20 MiB (`tool/builtin/read.go:35`), base64 in the
  entry.
* **An `O_APPEND` file cannot be cut through its own handle on Windows.** Go 1.27.3 opens it without `FILE_WRITE_DATA`
  (`syscall/syscall_windows.go:385-393`), which setting a file's end needs. D16's truncate must open the path. The
  plan's first step confirms this on the Windows laptop before any code depends on it.

The owner answered three questions on 2026-11-04:

* **Q8. Where U3's work goes.**
  * **(a) Recommended.** A plan of its own, `0037-PLAN-session-store-durability.md`, carrying D16 and D36–D41. Plan 2
    keeps D12–D15 and D17–D22.
  * **(b)** The whole of plan 2, D12–D22, with the store first.

  *Answered: (a), "Own small plan".*
* **Q9. A cap on a session entry's size.**
  * **(a) Recommended.** No cap, decided and recorded. An entry is bounded where it is made.
  * **(b)** A ceiling well above any source, such as 64 MiB.
  * **(c)** 1 MiB, as #7 asks.

  *Answered: (a), "No cap, recorded".*
* **Q10. An `fsync` policy for appends.**
  * **(a) Recommended.** None. A process crash loses no entry `Append` returned for, and power loss can lose the tail,
    which D16's cut leaves clean.
  * **(b)** One sync at the end of each turn.
  * **(c)** A sync for every append.

  *Answered: (a), "None, keep 0037's call".*

The decisions, written to those answers:

* **D36. The session store has a plan of its own, which closes #7's U3 (Q8 (a)).**
  * D34 gains a fourth plan, `0037-PLAN-session-store-durability.md`, carrying D16 and D37–D41. It needs none of the
    other three.
  * D34's plan 2 keeps D12–D15 and D17–D22.
  * This record is the receiving end of 0039-MADR D22's store items. The wire's 10 MiB frame bound, also sent here by
    D22, stays with plan 2's resource bounds and is not in this plan.
* **D37. D16, as it is built.**
  * The log keeps its file's size in memory. `Open` sets it from the bytes it read, and each write that succeeds
    advances it. No append adds a `stat`.
  * A write that fails, wholly or in part, is cut back to that size through the path (`os.Truncate`), never through the
    append handle. When the cut fails too, the log is marked torn, so the next line starts on a line of its own.
  * The entry joins `l.entries` only after its write succeeds. A failed `Append` leaves `Entries` as it was.
  * For `create`, the size before is "no file". A failed first write closes and removes the new file, releases the
    lock, and returns the id to pending. The log stays unwritten, holding the entries before the failed one, so the
    next `Append` tries `create` again.
  * `Open` cuts the last line only when all three hold: the file is of the current version, since an older file is
    never written; the line has no newline; and it does not decode as an entry. The cut is logged once at Warn, with the
    file's base name and the bytes cut. A complete entry that only lacks its newline is kept, and the next line starts
    on its own, as today.
  * `Read` holds no lock and never cuts.
  * A writer killed during `create`'s write can leave a torn header. `Open` and `Read` report that file as
    `session.ErrCorrupt`, as today. It is not cut, since an empty file is no less corrupt.
  * The package comment's "A file is never rewritten" gains its one exception: under the writer lock, gobble cuts the
    bytes of its own failed append, and an unparseable torn last line.
* **D38. A session entry has no size cap (Q9 (a)).**
  * An entry is bounded where it is made: D12 for mentions, D13's 256 KiB backstop for a tool result, `read`'s 20 MiB
    for an image.
  * A store cap would refuse a turn's record after the turn had run, and leave the live session ahead of its file,
    which D15 exists to prevent.
  * magic-cli-remote's 1 MiB is a WebSocket frame, not a reader of these files. #7's request is answered by this
    decision, not met.
* **D39. Appends are not synced (Q10 (a)).**
  * This turns the survey's "noted and not taken" into the decision #7 asked for.
  * A process crash or kill loses no entry `Append` returned for: the operating system holds the bytes. Power loss or a
    kernel crash can lose the last entries, and can leave a torn last line. D37's cut leaves that file clean on the next
    `Open`.
  * The package comment and `docs/architecture.md`'s `session/jsonl` row say so.
* **D40. A writer killed mid-append reopens with every acknowledged entry (#7's U3 acceptance).**
  * A test re-runs its own binary as a writer that appends large entries and reports each `Append` that returned. The
    test kills it, by `Process.Kill` on every host, then opens the file and checks:
    * every entry the writer reported is present, in order;
    * at most the one entry being written when it died is missing;
    * a torn tail was cut and logged, with the bytes the file lost;
    * the next `Append` writes a file that reads back with nothing skipped.
  * One round kills the writer during its loop, wherever the kill lands. Another has the writer leave half a line on
    disk itself before it is killed, so the torn path runs on every host, every time.
  * It runs in `go test ./...` on all three hosts, and under `-race`.
* **D41. A session file is read line by line, without first slicing every line (0038's item).**
  * `parse` iterates with `bytes.Lines` (Go 1.24). It keeps the per-line skip and warning that Pi's format needs, which
    a streaming `jsontext.Decoder` would not.
  * A benchmark measures `parse` before and after, with `-benchmem`. The gain claimed is only what it shows. If it
    shows none, the change is not made, and the plan records why.
  * `FuzzParse` and the fixture tests are unchanged and pass.
  * D24's streaming summaries for `List` stay plan 3's.

### Amendment (2026-11-14): a resumed session's request prefix, and the failed turn's history

*Accepted 2026-11-14 for what the owner ruled: F33 and F15 are fixed, and the fix is built as soon as its plan is
approved, ahead of gobble `v0.7.0`. How F33 is fixed is open: Q11 to Q13 below. The plan,
[0037-PLAN-resume-prefix-and-failed-turn-history.md](0037-PLAN-resume-prefix-and-failed-turn-history.md), is
`proposed`.*

*(2026-11-14, later: the owner ruled on Q11 to Q13, each a dated note under its question: Q11 (a) and Q13 (a); Q12 a
one-time measurement across every wire, then an assessment. The owner also ruled "Approve and build" on the plan,
which is now `in-progress`.)*

**Where this comes from.** On 2026-11-14 a read-only analysis of gobble's prompt caching ran probes on a scratch clone
of `main` at `512db99`, outside the repository. The owner then ruled, that day: the records to draft now are "Resume
prefix + failed-turn history" and "Cache visibility + 1-hour pricing", and the timing is "Resume/failed-turn fix now;
rest after v0.7.0". The second record is
[0057-MADR-cache-visibility-and-one-hour-pricing.md](0057-MADR-cache-visibility-and-one-hour-pricing.md). This
amendment is the first. It belongs here because it is this record's driver "One source of truth" ("The live session
and its file never disagree, and memory never holds what the disk does not") failing in a second place, and because
F15, which it also carries out, is this record's finding with no plan.

**What was run.** The probes were throwaway tests in package `acpserver` of the scratch clone. Each drives a session
through acpclient against llmtest's scripted model, closes it, loads it in a second `Agent` over the same store, and
prompts again. Each request is then rendered through the real `claude` and `openai` wires of `llm/provider` with a
transport that records the body and answers 400, so nothing reaches a service. Two prototype tests, which the plan
turns into its regression tests, were then run at `512db99` and failed as quoted below. The evidence key is the one
above.

* **F33. A resumed session does not send the messages the live session sent, so the provider's cached prefix is lost
  on resume (T, R).**
  * **Cause (a): a prompt's blocks.** In memory, a prompt with an `@path` mention, an embedded resource or a resource
    link is a files block followed by its text blocks: `prompt.Content = append([]llm.Content{{Type:
    llm.ContentText, Text: files}}, prompt.Content...)` (`acpserver/agent.go:675`). `userMessage` makes one block per
    ACP text block (`acpserver/agent.go:1038-1046`), so a prompt of two text blocks is two blocks without any mention.
    On disk the user entry joins them into one string: `return &session.Message{Role: session.RoleUser, Content:
    jsonx.Quote(strings.Join(texts, "\n\n")), Timestamp: ms(ts)}` (`acpserver/history.go:68`). A reload makes one block
    of it (`acpserver/history.go:253-260`).
  * **Cause (b): a tool call's arguments.** The assistant entry keeps the model's arguments as a JSON value:
    `b.Arguments = jsontext.Value(c.ToolArgs)` (`acpserver/history.go:98`). Written and read back through the store,
    the value is compacted: a probe on the scratch clone gave `{"path": "a.go"}` → `{"path":"a.go"}`,
    `{\n  "path": "a.go"\n}` → `{"path":"a.go"}` and `{"path":"a\/b.go"}` → `{"path":"a/b.go"}`, and kept key order,
    `1.0` and non-ASCII text as given. Live history holds the model's own bytes. Anthropic's Messages wire sends
    arguments as a decoded object, so it is unaffected; OpenAI's Responses wire sends them as a string, so it differs
    (T). Other string-argument wires are inferred to differ the same way (I).
  * **Measured.** One session of two prompts: the first names `@notes.md` and is answered after a `read` call whose
    arguments hold a space, the second is answered at once; then closed, loaded, and prompted once more. The last request before the close against the first after the load, with the
    cache markers removed, item by item:

    | Wire | Tools | System | Messages |
    | :--- | :--- | :--- | :--- |
    | `claude` | same (17,090 bytes) | same (876 bytes) | diverge at item 0 of 6 |
    | `openai` | same (16,036 bytes) | same (851 bytes) | diverge at item 0 of 6 |

    Item 0 before the close is the files block alone. After the load it is the files block, a blank line, and `look at
    @notes.md please`. Aligned past that item, the `openai` wire's `function_call` differs again, `"{\"path\":
    \"notes.md\"}"` against `"{\"path\":\"notes.md\"}"`, and the `claude` wire's items are equal.
  * **The prototype test at `512db99`**, three cases, each prompting twice, reloading and prompting once more, and
    asserting that every earlier request's message items are a byte prefix of the post-reload request's:
    * `mention`: fails on both wires, `request 0's item 0 of 2 is not the post-reload request's`, and the same for
      requests 1 and 2;
    * `two text blocks`: fails on both wires, before `{"content":"first block","role":"user"}`, after
      `{"content":"first block\n\nsecond block","role":"user"}`;
    * `spaced arguments`: fails on `openai` only, `request 1's item 1 of 3`, before
      `{"arguments":"{\"path\": \"notes.md\"}",…}`, after `{"arguments":"{\"path\":\"notes.md\"}",…}`; `claude` passes.
  * **What it costs.** The tools and the system prompt still match, so the provider can read those back. From the
    first message on, the prefix is new: the whole history is sent uncached and, on Anthropic's routes, written again
    at the write premium ([0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md),
    Consequences: 1.25 times the input with `short`, 2 times with `long`). It happens once per resume, for any session
    whose history holds a prompt with a file, a resource, a link or two text blocks, or, on a string-argument wire, a
    tool call whose arguments are not compact. How often models send arguments that are not compact is not measured
    (I).
  * **What the records say.** The comment at `acpserver/agent.go:671-673` says "the user entry holds them, so a resumed
    session gives the model what it had", after 0005-PLAN F3-4's "The user entry holds the expanded message, so a
    resumed session gives the model what it had"
    ([0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md)). That holds for the text, not for the bytes.
    "Prompt-cache stability is sound", under "Looked at and found sound" above, and 0021-MADR F5 ("Its bodies are
    byte-identical across calls and provider instances") were measured within one process, never across a resume.
* **F15, re-read at `512db99` (M, R).** It stands as surveyed, with the lines moved.
  * On a model error with no compaction, only the prompt joins live history: `s.history = append(s.history, prompt)
    // the file keeps the prompt too` (`acpserver/agent.go:764`). The compacted branch already replays the file
    (`:755-761`). The append-failure exits (`:684`, `:694`, `:713`, `:773`) leave `s.history` as it was.
  * A probe: a `read` call, then a scripted 400 on the next model call, then a second prompt. Its requests by role:
    1. `[system:text user:text]`
    2. `[system:text user:text assistant:tool_call tool:tool_result]`
    3. after the failure, in the same process: `[system:text user:text user:text]`
    4. after a reload: `[system:text user:text assistant:tool_call tool:tool_result user:text assistant:text user:text]`
  * The prototype test, the next request in the same process against the next request after a reload, fails at
    `512db99`: `live: [system:text user:text user:text]`, `reloaded: [system:text user:text assistant:tool_call
    tool:tool_result user:text]`.
  * F15 also feeds F33: after a failed turn, the live session's next request is not the file's, so the first request
    after a reload breaks the prefix again, even with F33 fixed.

The owner ruled, 2026-11-14, that both are fixed now. The decisions, written to that ruling:

* **D42. Every request a session sends carries, as a byte prefix of its messages, the messages of every earlier
  request of that session, live or reloaded (F33).**
  * "Earlier" is since the last intended change of prefix: the three of 0021-MADR F6 (a deferred tool's load, an MCP
    server's late tools, plan mode) and a compaction, which rewrites the history by design.
  * The comparison is on the wire, with the cache markers removed, because the markers move by design.
  * How it is met is Q11, with Q13 under Q11 (a). Which wires the test covers is Q12.

  *(2026-11-14, the rulings: memory takes the file's form (Q11 (a)), and a tool call's arguments take the store's form
  in `agent`, on arrival (Q13 (a)). For the wires, the plan's P1 measures every wire once before the permanent test's
  coverage is settled (Q12's ruling).)*
* **D43. D15 is built now, with D42, in one plan.** The owner's timing: the plan is executed as soon as it is approved,
  ahead of `v0.7.0`, between phases of
  [0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
  and [0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md),
  never inside one. D15 is unchanged: every error exit sets `s.history` from `s.state()`.
* **D44. A fifth plan, `0037-PLAN-resume-prefix-and-failed-turn-history.md`, carries D15 and D42.**
  * D34's plan 2, still unwritten, keeps D12–D14 and D17–D22. This follows D36's precedent.
  * The plan's regression tests are the probes made permanent: `TestResumeKeepsPrefix` from the prefix probe, and
    `TestFailedTurnHistoryIsTheFiles` from the failed-turn probe. Each was seen failing at `512db99` as quoted above.
  * The notes in other records wait for Q11's answer, since their wording depends on it, and the plan's P0 writes
    them: 0005-PLAN F3-4's "so a resumed session gives the model what it had", and 0021-MADR F5.

    *(2026-11-14: Q11 is ruled, and both notes are written.)*

### Questions for the owner (2026-11-14)

*(2026-11-14, later: the owner ruled on all three. The options stand as written, and each ruling follows its options.
Until then this heading read "open".)*

* **Q11. How live history and the file come to agree (D42).**
  * **(a) Recommended. Memory takes the file's form.** The prompt is joined into one text block, with the same
    `"\n\n"` join as its user entry, before the turn's first request. A tool call's arguments are put in the store's
    form when they arrive: an object compacted as the store writes it; anything else left raw, since the store already
    keeps that raw under `gobble.rawArguments` (`acpserver/history.go:99-106`) and gives it back.
    * Every resumed session already sends this form today, so the model sees nothing it has not seen.
    * Cost: the live request changes shape (one block where there were several), so `TestPromptCarriesFiles`'s
      expectation (`acpserver/mentions_test.go:126`) changes with it. The store's compaction rule gains a second user,
      tested equal to the round trip.
    * Old sessions: their files already hold this form, so a resume loads what it loads today. The first request after
      resuming a session written before the fix misses the cache once, as today. A session written after the fix and
      resumed within the cache's lifetime reads its prefix.
  * **(b) The file keeps memory's form.** A user entry of several blocks is stored as an array of text blocks, which
    the session format already reads (`session/session.go:231-244`) and the reload turns into blocks
    (`acpserver/history.go:253-260`). The model's raw arguments
    are stored beside `arguments`, in `gobble.rawArguments`, whenever the store's form differs.
    * Cost: user entries change shape on disk. Every reformatted argument is stored twice. Two writers must stay in
      step with two readers.
    * Old sessions: their files hold the joined form, which is all a resume can load. The first request after the
      resume misses once. Their older turns stay joined, and only turns written after the fix round-trip exactly.
  * **(c) Live history is always the file's.** After every turn, `s.history` is replayed from `s.state()`.
    * Cost: alone it is worse than today. Within a turn the agent sends the in-memory forms, and the next turn would
      send the file's, so the prefix would break inside a live session at every such turn. It therefore needs (a)'s
      shaping as well. It also adds the replay to every turn: F28 measured `s.state()` at 10 ms and 6 MB for 300 steps,
      and 81 ms and 67 MB for 3,000.
    * Good: it would cover F15 by construction.
    * Old sessions: as (a).

  *Ruling, 2026-11-14: "Memory takes the file's form". **Decision: (a).***
* **Q12. Which wires `TestResumeKeepsPrefix` renders.**
  * **(a) Recommended. `claude` and `openai`.** They are the two shapes of arguments on the wire, an object and a
    string, and both were seen failing at `512db99`. Cost: the other wires (Together, xAI and the gateways' routes)
    are covered by reading, not by the test.
  * **(b) Every wire that caches:** `claude`, `openai`, `grok`, `together`, `kilo` and OpenCode Zen's two routes. Cost:
    a longer test, and one more fixture to keep as each wire changes. The cause is in gobble's history, not in a wire.
  * **(c) The `llm.Request` alone, compared message by message before any wire.** Cost: it tests gobble's structures,
    not the bytes a provider caches. It flags argument spacing that the `claude` wire decodes away, and misses any
    difference a wire's own encoding brings.

  *Ruling, 2026-11-14, none of the three options, verbatim: "test resume-prefix across all providers. once. measure.
  assess whether we should test every caching wire, or only the 2 going forward based on results and facts."*
  **Decision:**
  * The plan's P1 runs a one-time measurement of the resume-prefix check on every provider wire that gobble can
    render offline. As in the probe, a recording transport answers 400, so no live provider is called.
  * It records, per wire, whether the prefix held before the fix and after it, and which cause, (a) or (b), the wire
    exposes.
  * It ends with an assessment put to the owner: whether the permanent test covers every caching wire, or the two
    shapes `claude` and `openai`. Whether the measurement harness is kept as a test, or recorded as a one-off, is part
    of the same question.
  * Until the owner rules on that assessment, the permanent `TestResumeKeepsPrefix` covers `claude` and `openai`.
* **Q13. Under Q11 (a), where a tool call's arguments take the store's form.**
  * **(a) Recommended. In `agent`, when a call arrives,** beside the name's repair (`agent/agent.go:724-726`). Every
    later request of the turn, and the history kept at its end, carry the same bytes. Cost: `agent` applies the store's
    rule, through one function in `jsonx` that both share. A test holds it equal to a write and read through the store.
  * **(b) In `llm/provider`'s adapter, as it makes `llm.ToolCallDone`.** Cost: the rule lives beside the SDK, far from
    the store. llmtest's scripted turns bypass the adapter, so the `acpserver` tests would not exercise it.
  * **(c) In `acpserver`, when the turn's messages are kept at its end.** Cost: the turn's own later requests still send
    the raw form, and the next turn the stored one, so the prefix breaks inside a live session. `TestResumeKeepsPrefix`
    would catch it.

  *Ruling, 2026-11-14: "In the agent, on arrival". **Decision: (a).***

### Consequences

* Good, because the one finding that destroys data (F1) and the one that turns a write into code execution (F2) are
  the first work done.
* Good, because a deny rule can only get stricter: every normalisation adds a reading and keeps the stricter verdict.
* Good, because no input from a file, a server or a project can hang a tool, fill the disk or end a session.
* Good, because the live session, its file and its memory agree after any failure.
* Good, because a log attached to a bug report no longer carries the credentials that commands in it contained.
* Good, because the measured gains are large where they matter: a walk 6× faster with 15× less memory, a cwd's session
  list about 47× faster, a pattern match about 40× faster. Benchmarks hold them.
* Good, because the loop guard and the compaction estimate stop wasting calls and tokens in the cases measured.
* Neutral, because five earlier records get dated notes, and no earlier decision is reversed: 0012-MADR D10, 0005-PLAN
  F5-3 and F8, and the statements in 0005-PLAN and `docs/architecture.md` that F20 contradicts.
* Bad, because folding case on darwin can ask about a path on a case-sensitive volume that does not need it.
* Bad, because bash's protection stays best-effort. A path computed at run time is not seen, and the docs say so.
* Bad, because three plans of about thirty decisions are a long piece of work. It runs while 0035 waits.
* Bad, because the cut values (256 MiB, 20 KB, 16,000 characters, 2,048 characters, 1 MiB, 512 bytes) are judgement,
  not measurement, and some may need tuning.
* Good, because a failed append leaves the session file and the session in memory as they were, on every host, and a
  killed writer's session reopens with every entry it acknowledged (D37, D40). *(Added 2026-11-04.)*
* Neutral, because `TestMalformedAndTornLines` changes what it expects: the torn line it plants is cut by `Open`, so
  one line is skipped where two were (D37). *(Added 2026-11-04.)*
* Bad, because power loss can still lose the last entries of a session (D39), and #7's entry cap is declined rather
  than built (D38). *(Added 2026-11-04.)*
* Bad, because the store gains a test seam, two unexported function fields for opening and cutting a file, so that a
  short write can be made on every host (D37). *(Added 2026-11-04.)*
* Good, because a resumed session reads back the prefix it wrote, and a failed turn leaves the live session sending
  what its file holds (D42, D15). *(Added 2026-11-14.)*
* Neutral, because under Q11 (a) a live prompt with a file is sent as one text block, as every resumed session already
  sends it. *(Added 2026-11-14.)*
* Bad, because a session written before the fix still misses the cache once on its first resume. *(Added 2026-11-14.)*

### Confirmation

* Each decision gets at least one test that is seen failing on a scratch copy before the change, with the failure
  quoted in the plan's record. The survey's probes are the starting point:
  * `mv -f` into a directory (D1);
  * `.GIT/config`, and a symlink to `.git` (D2);
  * `echo > .git/config` (D3);
  * the bypass table (D4);
  * `.ENV` (D5);
  * an edit through `CLAUDE.md -> AGENTS.md` (D6);
  * a FIFO under `read` and `grep` (D7);
  * the overlapping `}\n}\n`, and mixed line endings (D8);
  * the spill cap (D9);
  * a FIFO and an over-limit settings file (D10);
  * F10's patterns under a time bound, and a copy into itself by real path (D11);
  * a large mention (D12);
  * a 600 KB MCP result (D13);
  * a grandchild holding stderr (D14);
  * a failed second call (D15);
  * a short write (D16);
  * a heredoc secret and a `Bearer` header (D17);
  * an alternating loop (D30);
  * the anchored and unanchored estimates (D31).
  * *(Added 2026-11-04:)* a short write, a failed cut, a failed first write, and a torn last line in a current and in
    an older file (D37); a writer killed mid-append, and one killed with half a line on disk (D40); `parse` before and
    after, by benchmark (D41).
  * *(Added 2026-11-14:)* a prompt with a mention, a prompt of two text blocks, and spaced arguments, each across a
    close and a load, on the wires Q12 names (D42); a failed model call after a tool call, and a failed append, each
    against a reload (D15).
* Each performance decision's benchmark shows its gain against the survey's figure, and is held to its budget by
  `make perf` on all three hosts (D33). D41's benchmark is measured and recorded, not held to a budget.
* The gates at each phase are those of every plan here:
  * `make pre-add-check`, `make preflight` and `make race` locally;
  * the same and `make perf` on the Linux machine and the Windows laptop;
  * `make probe-conhost` where `internal/cli` changes.
* `make check-records` and `markdownlint` pass with the plans and the dated notes in place.

## Pros and Cons of the Options

### A. One decision over the three categories, carried out by three plans, safety first

* Good, because the shared evidence is written once, and the dependencies between categories are explicit (D34).
* Good, because each category is still approved, executed and rolled back on its own.
* Good, because it is what the owner asked for.
* Bad, because a single record is long, and a reader of one plan has to find its part.

### B. Fix only what can do harm now

* Good, because the urgent findings land sooner.
* Bad, because the measured performance gains and the heuristic fixes wait, with no date.
* Bad, because several harmful findings are close to performance ones in the code (F10 under F23; F2 and F4 under F25).
  Splitting them means touching the same code twice.

### C. A MADR per category

* Good, because each record is shorter.
* Bad, because the survey's method and its shared facts would be repeated or cross-referenced three times.
* Bad, because the owner asked for one record.

### D. Record the findings, and decide nothing until 0035 is done

* Good, because nothing competes with 0035 for attention.
* Bad, because F1, F2 and F17 stay in the code while 0035 waits on two upstream issues, for an unknown time.
* Bad, because the findings go stale as the code moves.

## More Information

* **Survey provenance.** The four reports were model output from read-only agents, and their claims were checked
  against the cited lines before this record was written (Context, "How the code was surveyed").
  * The figures are from one machine, and each plan measures again at its start.
  * F32's sample used the owner's local session files. The survey's brief had not said it could, and the owner was told.
    Only counts were printed.
* **Related records:**
  * [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md):
    held phases, and the excluded code;
  * [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) and
    [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F3-4, F5-3, F6-2a's decision 12, F8;
  * [0012-MADR-tool-schemas-descriptions-and-loading.md](0012-MADR-tool-schemas-descriptions-and-loading.md): D10;
  * [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md): D8 and D15; *(2026-11-02, by
    0037-PLAN-permission-and-file-tool-hardening.md's deviation 9: and D18, whose edit lock's waits P8a
    jitters.)*
  * [0030-MADR-resize-and-performance.md](0030-MADR-resize-and-performance.md): D5's budgets;
  * [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): the client's queue.
  * *(Added 2026-11-04:)* [0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md):
    D22, whose store items the amendment receives;
    [0038-REPORT-json-v2-standardization-and-sdk-surface.md](../reports/0038-REPORT-json-v2-standardization-and-sdk-surface.md):
    section 8's U3 and its `bytes.Lines` item; [gobble-cli#7](https://github.com/maccavelli/gobble-cli/issues/7): U3.
  * *(Added 2026-11-14:)* [0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md): F5 and
    F6, the prefix and its intended changes; [0057-MADR-cache-visibility-and-one-hour-pricing.md](0057-MADR-cache-visibility-and-one-hour-pricing.md):
    the same analysis's second record.
* **Revisit:**
  * the cut values in D9, D10, D12, D13 and D17, once the plans' tests and real use show where they bind;
  * D24's summary cache, if a list is still slow at a user's real session count;
  * Q4's calibration, if overflows still happen after D31.
