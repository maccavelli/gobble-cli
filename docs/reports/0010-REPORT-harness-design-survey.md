# REPORT 0010 — Harness design survey: Pi, opencode and Kilo across the v1 line

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

This report decides nothing. It records how three coding-agent harnesses design the capabilities gobble's v1 line plans ([0005-MADR](../decisions/0005-MADR-v1-feature-scope.md)), so that gobble can take the best of each and be its own. The owner asked for it on 2026-10-07:

> I want to use the best of what is already designed, but make gobble uniquely gobble.

The choices are made in 0005-MADR's amendment of 2026-10-07, which cites this report.

## Sources and method

| Harness | Checkout | What it is |
| :--- | :--- | :--- |
| Pi | `312184edb` | The Go rewrite's origin. Its built-in tools were surveyed on 2026-10-07 for 0005-PLAN F1; the rest is in [0001-REPORT](0001-REPORT-go-port-feasibility.md) and 0005-MADR. |
| opencode | `ecc4916` | A TypeScript/Bun monorepo. `packages/opencode` is the shipped runtime ("v1"); `packages/core` is a durable rewrite in progress ("v2"), which says in its own header that it is unfinished (`packages/core/src/session/runner/llm.ts:43-91`). |
| Kilo | `9b17c7d522` | Kilo's CLI: a fork of opencode at `packages/opencode`, marked by about 490 `kilocode_change` comments, plus about 550 Kilo-only files under `src/kilocode/` and its own `packages/kilo-*`. |

goose's terminal-CLI mechanics are already assessed in [0006-MADR](../decisions/0006-MADR-goose-cli-port-candidates.md) and are not repeated here.

Five read-only surveys ran on 2026-10-07: three over opencode (tools; loop, sessions, prompts and permissions; providers, MCP, configuration and the user-facing surfaces), and two over Kilo (its changes to the fork; its own packages). Each claim below carries the file and line the survey read.

Path prefixes:

- **PI** = `pi/packages/coding-agent/src/`
- **OC** = `opencode/packages/opencode/src/`; **OCORE** = `opencode/packages/core/src/`
- **KI** = `kilocode/packages/opencode/src/` (the fork); **KP** = `kilocode/packages/`

## 1. Built-in tools (0005-PLAN F1; X3 for web and subagents)

### Read

- **Pi.** `{path, offset?, limit?}`, 1-based offset, head truncation at 2000 lines or 50 KB, with notices that name the next offset (PI `core/tools/read.ts:139-178`, `truncate.ts:11-12`). Images are found by magic bytes (`utils/mime.ts`). No line numbers in the output.
- **opencode.**
  - `{filePath, offset, limit=2000}`; a path may be a file or a directory (OC `tool/read.ts:28-36`).
  - Output wraps each line as `N: text` inside `<path>…</path><type>file</type><content>` (`:338-339`).
  - Lines are cut at 2000 characters, and the read streams and stops at 50 KB, never loading the whole file (`:14-16`, `:137-180`).
  - Footers name the next offset, or say `(End of file - total T lines)` (`:344-350`).
  - A missing file suggests up to three similar names in its directory, "Did you mean one of these?" (`:76-99`).
  - Binary files are refused: by an extension denylist, a NUL byte, or more than 30% non-printable bytes (`:182-227`).
  - Images and PDFs are sent as attachments (`:306-325`).
  - Reading a file pulls in the `AGENTS.md` files between it and the project root, once per message (`:300`, `:355-357`; `session/instruction.ts:179-221`).
  - `*.env` reads ask by default; `*.env.example` is allowed (`agent/agent.ts:130-135`).
- **Kilo.**
  - A missing path is permission-checked, with its real parent, before "not found" is said, so a model cannot probe which files exist (KI `tool/read.ts:87-101`).
  - After approval, a directory's real path is re-resolved, and a changed path is refused (a symlink-swap race) (`:235-263`).
  - docx, xlsx and ipynb are converted to text (`kilocode/tool/read-extract.ts`).
  - Truncated output stays valid Unicode (`:421-424`).

### Edit, write and patches

- **Pi.** `{path, edits:[{oldText, newText}]}`. Every edit is matched against the original file; uniqueness is counted in fuzzy space; overlaps are refused. The BOM and CRLF are kept. The fuzzy fallback is NFKC, trailing whitespace, quotes, dashes and special spaces (PI `core/tools/edit.ts`, `edit-diff.ts:34-55`, `:300-362`). There is a per-real-path mutation queue (`file-mutation-queue.ts`).
- **opencode.**
  - `{filePath, oldString, newString, replaceAll}`, one edit per call (OC `tool/edit.ts:47-56`).
  - Matching is a cascade of nine strategies, and the first one that finds a unique match wins (`:694-719`). The strategies are: exact; per-line trimmed; block anchors with Levenshtein scoring at 0.65 on the middle lines; whitespace-collapsed; indentation-flexible; escape-normalised (for over-escaped model output); whole-string trimmed; context anchors with half the middle lines equal; and every occurrence, for `replaceAll`. The code credits Cline and gemini-cli (`:1-4`).
  - A fuzzy match far larger than `oldString` is refused with "Re-read the file" (`:709-737`).
  - The user approves a unified diff before the write (`:145-153`).
  - A configured formatter then runs, the diff is recomputed after formatting (`:155-171`; `format/index.ts:73-114`, about 25 formatters in `format/formatter.ts`), and LSP errors in the file, at most 20, are appended to the result as "LSP errors detected in this file, please fix" (`:196-201`; `lsp/diagnostic.ts`).
  - `write` also reports LSP errors in up to five other files (`tool/write.ts:18`, `:79-90`).
  - Read-before-edit is only claimed in the tool descriptions. The code does not check it (`edit.txt:4`, `write.txt:5`).
  - v2 has a compare-and-swap instead: a write fails with "File changed after permission approval. Read it again before editing." when the file's bytes differ from what was read (OCORE `tool/edit.ts:113-116`, `:191-197`; `file-mutation.ts:60-63`, `:144-149`).
  - `apply_patch` takes a Codex-style envelope with add, delete, update and move sections and `@@` context lines (OC `tool/apply_patch.txt`; `patch/index.ts:70-241`). It verifies every hunk, asks one permission for all the files, and only then writes (`tool/apply_patch.ts:72-293`). It is offered instead of edit and write to GPT models other than gpt-4, and to open-weight models (`tool/registry.ts:297-300`).
- **Kilo.**
  - The file's encoding is detected (UTF-8 with or without a BOM, UTF-16/32, legacy encodings) and kept on write (`kilocode/encoding.ts`; KI `tool/edit.ts:142`, `:192`).
  - Diagnostics are limited to the edited files; upstream stored every project file's diagnostics, adding 100 KB or more per call (`tool/diagnostics.ts:1-20`).
  - An edit to Kilo's own configuration files returns parse, schema and unknown-key errors (`kilocode/config-validation.ts:24-60`).
  - Writing the agent manager's state file directly is refused (`kilocode/agent-manager/protection.ts`).

### Shell

- **Pi.** `{command, timeout?}` in seconds, with no default timeout. It finds bash through `shellPath`, Git Bash or `PATH`, and powershell is a separate tool on Windows. Output keeps the last 2000 lines or 50 KB, with the full output in a temporary file. On cancel or timeout the process group gets an immediate SIGKILL, or `taskkill /F /T` on Windows (PI `core/tools/bash.ts`, `utils/shell.ts:67-218`, `core/tools/output-accumulator.ts`).
- **opencode.**
  - One tool, `bash`, whatever the shell: bash, PowerShell 7, Windows PowerShell 5.1 or cmd. Its description is generated for the shell in use, including 5.1's lack of `&&` (OC `tool/shell/id.ts:14-17`; `tool/shell/prompt.ts:43-271`).
  - `{command, timeout (ms), workdir}`, where `workdir` is "use this instead of cd" (`prompt.ts:15-23`). The default timeout is 2 minutes; v1 has no maximum, and v2 caps it at 10 minutes (OC `tool/shell.ts:347`; OCORE `tool/bash.ts:19-32`).
  - On POSIX the command runs in its own process group. Abort and timeout kill it, then force-kill after 3 s (`shell.ts:303-309`, `:533-555`).
  - Output beyond the limits is streamed to a file, and the model sees the tail and "Full output saved to: <file>" (`:500-580`).
  - The command is parsed with tree-sitter (bash or PowerShell), and each sub-command is its own permission pattern (`:311-410`). "Always allow" is offered as a prefix sized by a command-arity table, such as `git checkout *` or `npm run dev *` (`permission/arity.ts`).
  - The path arguments of rm, cp, mv, cat and their PowerShell and cmd equivalents are resolved to detect writes outside the project (`shell.ts:28-66`, `:349-405`).
- **Kilo.**
  - The permission pattern masks operators only where the parse tree proves them inert: in quotes, heredocs, `/dev/null` and `2>&1`. So `grep "a|b"` no longer trips a `*|*` deny (`kilocode/tool/shell-pattern.ts`).
  - A command the parser cannot classify becomes its own raw pattern, failing closed (`kilocode/tool/shell-unparsed.ts`; KI `tool/shell.ts:413-419`).
  - Shells started for the model do not inherit the harness's own credentials or configuration variables (`kilocode/process/env.ts`).
  - After exit, the tool waits up to 3 s for the last of the output (`shell.ts:677-681`).
  - `background_process` starts a long-running command and monitors it within line and time caps (`kilocode/tool/background-process.txt`).

### Search and listing

- **Pi.** `grep` and `find` run the ripgrep and fd binaries, downloading them when they are missing. `ls` is plain code. Limits are 100 matches, 1000 files and 500 entries (PI `core/tools/grep.ts`, `find.ts`, `ls.ts`).
- **opencode.**
  - One shared ripgrep adapter for both tools (OCORE `ripgrep.ts`):
    - it uses `rg` from `PATH`, or downloads a pinned release (`ripgrep/binary.ts:94-115`);
    - it reads one row more than the limit, then kills `rg` early (`:126-131`);
    - an invalid regex is a typed error (`:135-137`).
  - Results are capped at 100 and grouped by file, with a hint to narrow the search (OC `tool/grep.ts:63-102`; `tool/glob.ts:49-62`).
  - There is no `ls` tool: listing a directory is a `read`.
  - Nothing sorts results by modification time.
  - `webfetch` returns Markdown, text or HTML; it times out after 30 s by default (120 s at most) and takes at most 5 MB. It negotiates the content type, and works around a Cloudflare challenge (`tool/webfetch.ts`).
- **Kilo.**
  - grep gains `literal`, `ignoreCase`, `context` and `limit`, and marks lines `[match]` or `[context]` (`kilocode/tool/grep-signal-controls.ts`).
  - `semantic_search` is offered only once the index is ready, and its hint is then added to the grep and glob descriptions (`kilocode/tool/registry.ts:46-47`, `:450-456`). See section 12 for the index.

### Todo, questions, plan exit, subagents, skills

- **Pi.** A todo tool exists only as an example extension (PI `../examples/extensions/todo.ts`). `tool_search` is a built-in extension: BM25 over deferred tools, which loads what it finds (PI `extensions/tool-search/tool.ts`).
- **opencode.**
  - **Todo.** `todowrite` replaces the whole list, `{content, status, priority}` (OC `tool/todo.ts`; `packages/schema/src/session-todo.ts:7-15`). Its rules: exactly one item in progress, and nothing marked complete before it is verified (`tool/todowrite.txt`). Subagents may not use it (`tool/task.ts:143-146`).
  - **Questions.** `question` asks one or more multiple-choice questions, `{question, header, options[{label, description}], multiple}`. A free-text answer is allowed, and the recommended option comes first (`tool/question.ts`).
  - **Plan exit.** `plan_exit` asks the user, then switches agent with an injected message (`tool/plan.ts:30-69`).
  - **Subagents.** `task` runs a subagent in a child session (`tool/task.ts:43-358`):
    - its description lists the available agents;
    - nesting is one level deep by default;
    - a child session can be resumed by `task_id`;
    - the child may not call `task` or the todo tool;
    - a background mode injects the result into the parent later.
  - **Skills.** `skill` loads a `SKILL.md` and lists up to ten of its files (`tool/skill.ts`).
- **Kilo.**
  - A subagent inherits its parent's edit, MCP and notebook denies as hard ceilings (`kilocode/tool/task.ts` `inherited()`).
  - A task can be resumed only by its own parent (KI `tool/task.ts:171`).
  - The child's cost is added to the parent on every exit path (`:493-521`).
  - The todo tool reports what each update changed: items changed, started and done (`kilocode/todo-view.ts`).

### Tool framework

- **opencode.**
  - **Descriptions** live in `.txt` files, some templated with the OS, the shell, the limits or the current year (OC `tool/*.txt`; `tool/shell/prompt.ts:28-34`).
  - **Bad arguments** come back to the model as "…Please rewrite the input so it satisfies the expected schema." (`tool/tool.ts:24-129`).
  - **Repairing calls.** A tool name in the wrong case is corrected. Any other unparseable call goes to a hidden `invalid` tool that returns a usable error (`session/llm.ts:296-317`; `tool/invalid.ts`).
  - **One truncation framework** covers every tool. The full output is kept under the data directory for 7 days, and the hint suggests a subagent when one is available (`tool/truncate.ts`).
  - **The result** is `{title, output, metadata, attachments}`, and `metadata` is for display only (`tool/tool.ts:48-53`).
  - **Doom loops.** Three identical calls in a row raise a `doom_loop` permission question (`session/processor.ts:29`, `:356-379`).
  - **v2** rejects a call to a tool that changed or disappeared since it was advertised (OCORE `tool/registry.ts:50-61`).
- **Kilo.** Bounded circuit breakers: three malformed calls in a row end the turn (`kilocode/session/processor.ts:380-428`). Tool schemas are sanitised of regex constructs that some providers reject (`kilocode/session/tool-schema.ts`).

## 2. Agent loop and retry (0005-PLAN F5)

- **Pi.** Steering and follow-up queues, which gobble already has (0002-PLAN Phase 7). Retries are exponential back-off (0005-MADR, Retry row).
- **opencode.**
  - **The loop.** Each iteration is one provider step, rebuilt from the database, so a message sent mid-turn is picked up at the next step (OC `session/prompt.ts:1052-1130`).
  - **Max steps** is set per agent; on the last step the model is told to finish (`:1178-1281`; OCORE `session/runner/max-steps.ts`).
  - **Retry** is opencode's own, not the SDK's (`session/retry.ts:26-155`):
    - at most 5 attempts;
    - back-off from 2 s, doubling, with 25% jitter, capped at 30 s without headers;
    - `retry-after-ms`, and `retry-after` in seconds or as an HTTP date, are honoured;
    - a context overflow is never retried.
  - **Cancel** waits 250 ms, then marks unfinished tools as aborted; the next request shows them as "[Tool execution was interrupted]" (`session/processor.ts:585-607`; `session/message-v2.ts:362-373`).
  - **v2** makes steering and queueing durable, first-class inputs (OCORE `session/sql.ts:140-166`; `session/runner/llm.ts:187-196`, `:392-415`).
- **Kilo.**
  - **Circuit breakers** for every "loop forever" case (KI `session/prompt.ts`; `kilocode/session/processor.ts`):
    - at most three compactions per turn;
    - a missing finish reason ends the step;
    - a `tool-calls` finish with no tool calls is a stop;
    - an empty or reasoning-only response is retried twice.
  - **Offline.** A network loss sets the session `offline`, probes the network, and retries without spending the retry budget (`session/network.ts`; `session/retry.ts:142-187`).
  - **Stalls.** A 10 s silence triggers a provider probe, and a first-byte deadline stops a 200 with no data from hanging the loop (`kilocode/provider/provider.ts:330-422`).
  - **Max steps** is sent as a user message, because some providers refuse an assistant prefill (KI `session/prompt.ts:1859`).
  - **Resuming.** An interrupted turn resumes with a `[TASK RESUMPTION]` message (`kilocode/session/continuation.ts`).

## 3. Sessions (0005-PLAN F2)

- **Pi.** JSONL v3 files with a tree, labels and context edits (0005-MADR, F2). gobble keeps this format for the Pi bridge (0003-MADR).
- **opencode.**
  - **Storage.** SQLite in WAL mode, with session, message, part and todo tables (OCORE `database/database.ts`; `session/sql.ts:22-117`).
  - **Event-sourced.** Every change is an event, and projectors write the rows (OC `session/session.ts:629-643`; OCORE `session/projector.ts`), so displays, sharing and plugins read one feed.
  - **Parts** include text, reasoning, file, tool, step-start and step-finish, snapshot, patch, subtask, compaction and retry (`packages/schema/src/v1/session.ts:66-317`).
  - **Forking** copies up to a message and remaps the IDs (OC `session/session.ts:691-732`).
  - **Child sessions** carry a `parent_id`.
  - **Titles** come from a hidden agent on a small model, after the first real message (`session/prompt.ts:193-253`).
- **Kilo.** Titles wait for up to four user messages and tool excerpts (`kilocode/session/title.ts`). Sessions are retained machine-wide for 30 days, when retention is turned on (`kilocode/session/retention.ts`). Large patch metadata is not stored (KI `session/message-v2.ts:116-203`).

## 4. Compaction (0005-PLAN F5)

- **Pi.** A cut point at 20000 recent tokens, a split turn, an iterative summary prompt and file lists (0002-PLAN Phase 6, already built).
- **opencode.**
  - **When.** Overflow is checked after each step and before each request. The usable context is the input limit minus `min(20k, max output)` (OC `session/overflow.ts:8-34`; `session/processor.ts:491-496`).
  - **What stays verbatim.** A tail of whole turns within a token budget: 2k to 15k, or 25% of the usable context (`session/compaction.ts:31-33`, `:115-269`).
  - **The summary** is rolling: the previous one is fed back, on a fixed template of Objective, Important Details, Work State, Next Move and Relevant Files, with rules for merging (OCORE `session/compaction.ts:16-55`, `:160-174`).
  - **Pruning** is a separate, opt-in pass: tool outputs older than the last two turns and beyond 40k tokens are cleared, when that frees more than 20k (OC `session/compaction.ts:273-317`).
  - **Cache stability (v2).** v2 freezes the system context per session and sends later changes as appended messages, so the prompt cache stays valid (OCORE `session/context-epoch.ts:40-77`).
- **Kilo.**
  - **Compaction starts before overflow,** at a set percentage of the window. The estimate starts from the provider's own token count, adds new content at a 1.3× factor, and prices media as placeholders (`kilocode/session/overflow.ts`).
  - **Oversized history** is summarised in chunks, three at a time (`kilocode/session/compaction-chunks.ts`).
  - **"Request too large"** is retried with tool outputs and media removed (`kilocode/session/compaction-payload-recovery.ts`).
  - **An empty summary** never replaces the session (KI `session/compaction.ts:548-579`).

## 5. Context, prompts, skills and commands (0005-PLAN F3)

- **Pi.** Discovers `AGENTS.md` and `CLAUDE.md`, plus `SYSTEM.md` and `APPEND_SYSTEM.md`. The system prompt is built in sections. It has Agent Skills, and prompt templates with shell-style arguments (0005-MADR, F3).
- **opencode.**
  - **Per-model prompts.** A base prompt is chosen for each model family (OC `session/system.ts:28-51`).
  - **Environment block.** The prompt carries the model, directory, root, git status, platform and date (`:69-104`).
  - **Instructions.**
    - Global: `~/.config/opencode/AGENTS.md`, or else `~/.claude/CLAUDE.md`.
    - Project: walking up from the directory, the first file name found among `AGENTS.md`, `CLAUDE.md` and `CONTEXT.md` wins, and all its copies up the tree are included.
    - Configuration can add globs and URLs (`session/instruction.ts:60-168`).
  - **Skills** come from `.claude/`, `.agents/`, configuration and remote indexes. They are listed in the prompt and loaded by a tool, and they are also slash commands (`skill/index.ts:173-346`; `command/index.ts:134-152`).
  - **Commands.** Markdown with frontmatter (`$1..$N`, `$ARGUMENTS`, `` !`cmd` ``); a command can run as a subtask (`config/command.ts`; `session/prompt.ts:1372-1451`).
  - **@-mentions.** `@file` reads the file inline, with `?start=&end=` ranges; `@agent` hands off to an agent (`:157-191`, `:808-990`).
- **Kilo.**
  - **Editor state per message.** The active and open files go into each user message as `<environment_details>`, not into the system prompt, so the cached prefix does not change (`kilocode/editor-context.ts`; `kilocode/session/prompt.ts:455-505`).
  - **Untrusted project instructions.** Project instruction files cannot use `{env:}`, and their `{file:}` is confined to the project (KI `session/instruction.ts:100-191`).
  - **Agent switches.** A switch adds a reminder saying whether the new agent can write (`kilocode/session/mode-reminders.ts`).
  - **Instruction changes** are appended as durable system messages, keeping the cached prefix unchanged (`CONTEXT.md:98-177`).

## 6. Agents, modes, permissions and trust (0005-PLAN F6)

- **Pi.** No permission prompts or plan mode are built in; they ship as example extensions (0005-MADR, Context). gobble has ACP allow-once and reject-once, and a plan mode (0002-PLAN).
- **opencode.**
  - **An agent** is a prompt, a permission ruleset, a model and a step limit, written in configuration or as Markdown (OC `agent/agent.ts:140-294`; `config/agent.ts`).
  - **Built-in agents:** `build` and `plan`, the subagents `general` and `explore` (read-only), and hidden ones for compaction, titles and summaries.
  - **Plan mode** is a set of permissions (edits denied except to plan files) plus a reminder each turn (`agent/agent.ts:156-181`; `session/reminders.ts:26-48`).
  - **Rules** are `{permission, pattern, allow|ask|deny}`. The last match wins, and no match means ask (`permission/index.ts:28-38`, `:200-202`). In `"git *"` the arguments are optional, and matching ignores case on Windows (OCORE `util/wildcard.ts`).
  - **Hidden tools.** A tool denied outright is removed from the model's tool list (`permission/index.ts:204-219`).
  - **Replies** are once, always or reject:
    - "Always" lasts for the running instance and also settles other pending questions it now covers.
    - A rejection cancels the rest, and can carry feedback to the model (`:109-167`).
    - v2 saves permissions per project (OCORE `permission/saved.ts`).
  - **Outside the worktree.** A path outside it asks `external_directory` for its directory (OC `tool/external-directory.ts:15-45`).
  - **Project trust: none.** Project configuration and plugins load unless an environment variable turns them off (`config/config.ts:420-470`).
- **Kilo.**
  - **Hard ceilings.** Read-only agents (ask, plan) cannot be widened by configuration, "always", a yolo mode or a subagent (`kilocode/agent/index.ts:211-270`). Read-only bash is an allowlist plus a block on operators and exec flags (`:87-152`).
  - **Protected files.** Edits to the harness's own configuration, `AGENTS.md` included, always ask, and "always" becomes "once" (`kilocode/permission/config-paths.ts`).
  - **Provenance.** Every approval records where it came from (agent, global, project, session, manual or default), so a display can explain it (`kilocode/permission/provenance.ts`).
  - **Headless runs** fail a subagent's question at once instead of hanging (`kilocode/permission/headless.ts`).
  - **Persisted "always".** "Always" rules are saved to global configuration, under a lock (KI `config/config.ts:1150`).
  - **The new agents** are `ask`, `debug` and `orchestrator`. `orchestrator` is deprecated, because every full agent can use `task` (KP `kilo-docs/pages/code-with-ai/agents/orchestrator-mode.md:9-16`).
  - **The plan follow-up** offers a new session, continuing here, or more refining; a new session gets a hand-over summary (`kilocode/plan-followup.ts:107-340`).

## 7. Hooks and plugins (0005-PLAN F7)

- **Pi.** Lifecycle hooks are part of its extension API (0005-MADR, F7). gobble plans exec hooks with JSON on stdin and stdout.
- **opencode.**
  - **Hooks** take an input and a mutable output, in order (`packages/plugin/src/index.ts:222-335`; OC `plugin/index.ts:284-297`). They include:
    - tool execution before and after;
    - the shell environment;
    - tool definitions;
    - chat parameters and headers;
    - transforms of the system prompt and the messages;
    - compaction.
  - **Plugins** are TypeScript, loaded from npm or `.opencode/plugin/`, and receive a client for the server's own API.
- **Kilo.** No change to the hook design beyond hiding tools per client.

## 8. MCP (0005-PLAN F8)

- **Pi.** An MCP client with OAuth and exposure modes, and `tool_search` (0005-MADR). gobble's `mcpclient` already covers stdio and streamable HTTP (0002-PLAN Phases 5 and 8).
- **opencode.**
  - **Servers** are `local` (stdio) or `remote`. A remote server is tried as streamable HTTP first, then SSE (OC `mcp/index.ts:269-370`).
  - **Status** is one of `connected`, `disabled`, `failed`, `needs_auth` or `needs_client_registration`, and tells the user what to run (`:83-106`).
  - **OAuth** uses dynamic client registration, with credentials bound to the server's URL (`mcp/oauth-provider.ts`; `mcp/auth.ts`).
  - **Tool names** are `server_tool`, sanitised.
  - **Tool lists.** A list that fails output-schema validation is fetched again without output schemas, and a repeated pagination cursor is detected (`mcp/catalog.ts:14-168`).
  - **Prompts and resources.** Prompts become slash commands. Resources are permissioned tools, `mcp:<server>:<uri>` (`session/tools.ts:27-31`).
  - **Instructions.** Server instructions go into the system prompt (`session/system.ts:123-135`).
- **Kilo.**
  - **Containers.** `--rm` is added to `docker run` and `podman run`.
  - **Stderr** is drained to the log (KI `mcp/index.ts:65-78`, `:428-447`).
  - **Credentials.** Servers do not inherit the harness's credentials.
  - **Default permission.** A default `<server>_*: ask` rule applies (`kilocode/agent/index.ts:359-366`).

## 9. Providers, models and auth (0005-PLAN F4)

- **Pi and gobble.** gobble's provider layer is go-llmprovider-sdk (0004-MADR).
- **opencode.**
  - **Model catalog.** A versioned catalog in the shape of models.dev, holding limits, tiered costs, reasoning options and status. It is cached for 5 minutes under a cross-process lock, embedded at build time, and refreshed hourly (OCORE `models-dev.ts:25-258`).
  - **Variants.** Reasoning levels are provider-neutral named "variants" (low, medium, high, xhigh, max, none) that map to each provider's options (OC `provider/transform.ts:586-592`, `:790+`, `:1717-1766`).
  - **Cost** accounting handles cache reads and writes, tiers and reasoning tokens (`session/session.ts:338-405`).
  - **A small model** runs titles and summaries (`provider/provider.ts:1961-2028`).
  - **Auth** is one owner-only file (`auth/index.ts`). Login methods come from plugins and share one flow across the CLI, the TUI and HTTP (`provider/auth.ts:11-221`).
  - **Enterprise bootstrap.** A `.well-known/opencode` document supplies credentials and configuration (`cli/cmd/providers.ts:325-351`).
- **Kilo.** Its gateway routes models on the server and reports which model actually served each request (KP `kilo-gateway/src/gateway-metadata.ts:7-99`). A `data_collection` setting is sent per request, and the catalog marks models that may train on prompts (`kilo-gateway/src/responses.ts:31-58`; `api/models.ts:35-60`).

## 10. Configuration

- **opencode.**
  - **Files.** JSONC with a JSON Schema.
  - **Layers**, lowest precedence first: remote, then global, then a file named by an environment variable, then project files walking up, then `.opencode/` directories holding Markdown agents and commands, then inline JSON, then managed directories and MDM (OC `config/config.ts:328-548`).
  - **Substitution.** `{env:VAR}` and `{file:path}` are substituted (`config/variable.ts`).
  - **Writes** keep comments (`config/config.ts:150-162`, `:656-680`).
- **Kilo.**
  - **Trust by origin.** Configuration is trusted by where it came from:
    - global and managed configuration is trusted;
    - project configuration may not use `{env:}`, and its `{file:}` stays inside the project.
    (KI `config/config.ts:336`, `:789-839`)
  - **Unknown keys** are warned about.
  - **Parse errors** are collected, not fatal.

## 11. CLI, headless, ACP, server and TUI (0005-PLAN F9, F10)

- **opencode.**
  - **One API for every surface.** The server is the agent. The TUI (in a worker thread), `run`, ACP, the web UI and the desktop app are all clients of one HTTP API generated from OpenAPI, and the local case is an in-process fetch (OC `cli/cmd/run.ts:948-961`; `cli/cmd/tui.ts:24-50`).
  - **Headless `run`** writes NDJSON events, and refuses permission questions unless `--auto` is given (`cli/cmd/run.ts:678-820`).
  - **ACP** is an adapter over that API (`acp/agent.ts:32-87`; `acp/service.ts:112-136`):
    - it advertises image and embedded-context prompts, and session fork;
    - reasoning variants are a `thought_level` configuration option;
    - an approved edit is written into the editor's buffer (`acp/permission.ts:40-115`).
  - **TUI.** SolidJS on opentui, with a command palette, dialogs, 33 themes, a diff view that switches between split and unified by width, and endpoints that let an IDE drive it (`packages/tui/src/`; OC `server/routes/instance/httpapi/groups/tui.ts`).
  - **Upgrades.** Patch releases are installed automatically; minor and major releases only notify (`cli/upgrade.ts:8-53`).
  - **Telemetry** is OTLP only, and opt-in.
- **Kilo.**
  - **Automation-friendly exits.** `run` exits 1 when the model returns nothing, and `--auto` also approves subagents' questions (KI `cli/cmd/run.ts:1110-1119`).
  - **No orphans.** A parent-process watchdog stops orphaned servers (`kilocode/parent-watchdog.ts`).
  - **TUI.** The TUI adds a vim mode and a cost alert.

## 12. Undo, memory, indexing, sandbox and telemetry (0005-PLAN X2, X5, X6)

- **Snapshots.**
  - **opencode** keeps a shadow git repository per worktree. It borrows the real repository's objects through `alternates`, and takes a tree before and after each step, with a patch per step (OC `snapshot/index.ts:71-390`; `session/processor.ts:99-102`, `:424-484`). Revert and unrevert follow (`session/revert.ts`).
  - **Kilo** adds:
    - a timeout on the first indexing of a large repository, with a choice to disable snapshots;
    - a lock shared across processes;
    - a check of every checkpoint before any file is restored;
    - restores that report failure honestly.
    (`kilocode/snapshot/*`; KI `snapshot/index.ts:532-563`)
- **Memory (Kilo).**
  - **Scope.** Memory is per project, and off by default.
  - **Capture.** At the end of a turn, a digest and a typed consolidation (facts, decisions, constraints, environment, corrections) are written. A taxonomy of skip reasons refuses transient, personal, secret and policy-shaped items (KP `kilo-memory/src/schema.ts`, `effect/capture.ts`, `prompts/typed-consolidation.txt:20-93`).
  - **Injection.** A small capped index is injected, marked "context, not instruction"; detail comes from a keyword recall tool (`recall/budget.ts`; `recall/recall.ts`).
- **Indexing (Kilo).**
  - **Chunks.** Tree-sitter chunks of 50 to 1000 characters, with content-addressed IDs.
  - **Storage.** LanceDB or Qdrant.
  - **Incremental.** Re-indexing by file hash.
  - **Worktrees.** A linked worktree overlays the main checkout's index, verified by file hash (KP `kilo-indexing/src/indexing/*`; `search-service.ts:32-100`).
- **Sandbox (Kilo).** It confines writes only:
  - Linux uses bubblewrap and macOS seatbelt; Windows is unavailable, and an unsupported platform fails closed.
  - Network is denied, or limited to an authenticated proxy that checks each host's TLS SNI and allows only public addresses.
  - `.git` and the harness's own configuration are read-only inside writable trees.
  - A project can only tighten the policy (KP `kilo-sandbox/src/*`; `kilo-docs/pages/getting-started/settings/sandboxing.md`).
- **Telemetry (Kilo).** PostHog, on by default, with an environment kill switch (KP `kilo-telemetry/src/*`). The report notes one anti-pattern: its opt-out key is named `openTelemetry`, like the unrelated OTLP export.

## 13. Testing and review practice (Kilo)

- `TESTING.md`:
  - Agents test the backend out of process, on an ephemeral port.
  - Its environment is hermetic: an in-memory database, no default plugins, telemetry off and inline configuration (`TESTING.md:7-200`).
- `REVIEW.md`: never repeat what CI says, look for bugs, races, leaks and trust boundaries, and prefer real implementations to mocks (`REVIEW.md:7-71`).

## Observations for the decision

These are the survey's patterns, not choices.

1. **Interfaces differ, and none is shared.**
   - Tool parameter names: Pi has `path` and `oldText`; opencode has `filePath` and `oldString`.
   - Edit shape: Pi has multi-edit; opencode has one edit per call, plus `apply_patch` for GPT models.
   - Bash timeout unit: Pi uses seconds; opencode uses milliseconds.

   A model sees whichever schema gobble sends, so the choice is gobble's. Only the session file, which the Pi bridge reads, is bound to Pi.
2. **Where opencode and Kilo add a safety layer, it is in the design, not the wording.** Examples:
   - compare-and-swap before a write;
   - fuzzy-match size guards;
   - permission checks before "not found";
   - shell patterns that fail closed;
   - hard ceilings for read-only agents;
   - protected configuration files;
   - circuit breakers on loops.
3. **Several strong ideas cost little in Go and fit gobble's existing pieces.**
   - Fit with what gobble has: diagnostics after an edit, through an LSP gobble does not yet have; a doom-loop permission; an `invalid`-tool repair path; and line-numbered reads.
   - Fit with the ACP agent: the `thought_level` option.
   - Fit with go-llmprovider-sdk's catalog work: a model catalog with variants.
4. **Some designs depend on infrastructure gobble has decided against or has not built:**
   - a server-first architecture: gobble is ACP-first, in process (0002-MADR);
   - SQLite sessions: gobble keeps JSONL for the bridge;
   - TypeScript plugins;
   - a hosted gateway.
5. **Prompt-cache stability is a first-class rule in both opencode v2 and Kilo:**
   - frozen context, with changes appended as messages;
   - editor state carried in user messages.

   gobble's system prompt assembly (F3) has not been built, so it can adopt this from the start.
