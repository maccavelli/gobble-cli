# REPORT 0011 — Native tools across six harnesses: which operations gobble implements itself

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

This report decides nothing. It is the evidence for the 0005-MADR amendment of 2026-10-08, "native tools", which adds the tools below to gobble's line. The owner asked on 2026-10-07 and 2026-10-08:

> What about native tools like read readFile, edit editFile, writeFile, findFile, listFiles, listDirs, renameFile, renameDir, deleteFile, deleteDir, etc where the native tools already have the correct shell commands and syntax for their tasks on any OS? Should we consider building native tooling or just use shell commands? Or both?

Then:

> Copy and mkdir. Find me 5 more commands we should include as native tools. Evaluate the kilo, opencode, codex, goose, pi, and grok sources in gitrepos for ideas.

And:

> Add all proposed tools. Write it up. Be detailed and prove all assertions

## Sources

| Harness | Checkout | Tools surveyed |
| :--- | :--- | :--- |
| Pi | `312184edb` | built-in tools (0010-REPORT) |
| opencode | `ecc4916` | `packages/opencode/src/tool/` (0010-REPORT) |
| Kilo | `9b17c7d522` | its fork's tools (0010-REPORT) |
| codex | `0e1520605` | `codex-rs/core/src/tools/` |
| goose | `01eb01a` | `crates/goose/src/agents/platform_extensions/`, `crates/goose-mcp/` |
| grok-build | `2bdd1d6` | `crates/codegen/xai-grok-tools/src/implementations/` |
| gobble | `f6253fc` | `tool/`, the 0005 records |

Pi, opencode and Kilo were surveyed for [0010-REPORT](0010-REPORT-harness-design-survey.md). codex, goose and grok-build were surveyed on 2026-10-08 by three read-only agents, one for each. Their reports gave file and line citations.

## How the claims were proved

Every claim this report rests on is a row in the evidence table at the end. A script in the session scratchpad (`q178_claims.py` with `q180_claims2.py`) opened each cited file and checked the claim:

- **A positive claim** passes when a pattern stating it matches in that file. The table gives the line the pattern matched and the text on it.
- **An absence claim** ("no harness has a tool for X") passes when the pattern matches nothing: no line of a tool registry, or no entry of a tool directory.

The run proves 88 claims, and all of them pass. The checks were themselves seen to fail before they were trusted:

- **The positive checks.** The first run failed 6 claims. The agents' citations were wrong in six places:
  - grok-build's `kill_task`, `get_task_output`, PDF and PPTX readers are under `grok_build/` or the shared `read_file/`, not the paths cited;
  - 0005-MADR writes its `ls` row as a table row, not in bold;
  - **Kilo's `read-extract.ts` does not convert `.ipynb`.** Its notebook tools do.

  The last of these was also wrong in [0010-REPORT](0010-REPORT-harness-design-survey.md) §1, which is corrected alongside this report. Three more patterns first matched only an import line or an unrelated line. They were tightened to the line that shows the claim: opencode's directory read, grok-build's handling of `&`, and Pi's tool list.
- **The absence checks.** Each was run once with a pattern for a tool that does exist in that registry or directory: opencode's `lsp`, Pi's `ls`, goose's `tree`, codex's `view_image`, grok-build's `list_dir` and opencode's `read`. All six were reported. So an absence check that passes means the name really is not there, as far as its pattern reaches. That reach is the limit of the method: an absence claim covers the names in its pattern, in the registry it names.

## The rule: native or shell

From the six sources, and from gobble's own design so far, a tool is native when at least one of these holds. Otherwise the operation stays a shell command.

1. **The shell spelling differs by OS.** Removing, copying, listing and starting background work all take different commands and quoting in bash, PowerShell and cmd. grok-build carries code just to give `&` a meaning per shell (K8). Its explore and plan presets keep a read-only agent read-only by leaving the shell out altogether.
2. **Permissions need the exact paths.** A native tool names its paths, so gobble's confinement applies to them directly (0005-MADR, Confinement; F1a's `tool.Confined`). A shell command has to be parsed to find them, which is F6's `mvdan.cc/sh` work, and PowerShell cannot be parsed in Go at all.
3. **A client or undo needs a structured result.** ACP has `delete` and `move` tool kinds (acp-go-sdk `types_gen.go:6617-6618`), and gobble's `tool.Kind` has both (B1, B2). X2's file journal can record a native operation exactly; for a shell `rm` it has only snapshots.
4. **There is no shell equivalent.** Examples are semantic code navigation, and structured cells of a notebook.

What every harness leaves to the shell, gobble does too (A1–A6): file status, diffs, git, checksums and archives.

## Findings, per tool

### `move`, `delete`, `copy`, `mkdir` — file operations

- **No harness has them as tools** (A1–A6). Pi, opencode, codex, goose, Kilo and grok-build all leave renaming, deleting, copying and creating directories to the shell. codex comes closest: its `apply_patch` format can add, delete and move files (0010-REPORT §1).
- **gobble already has the tool kinds for them.** `tool.Kind` has `delete` and `move` (B1, B2), and ACP renders those kinds specially.
- **gobble's reasons are rules 1–3 above.** These operations differ most between shells, they are the ones a permission check most needs to see exactly, and plan mode and undo most need to recognise them.

### `tree` — the layout of a project

- **goose's developer extension has a `tree` tool** (G1). It walks with the `ignore` crate, which honours `.gitignore` (G2), and takes a depth (G3).
- **grok-build's `list_dir`** is gitignore-aware (K1). It lists every first-level entry before it goes deeper, under an output budget, so one large subtree cannot crowd out the rest (K2).
- **opencode has no `ls` or `tree` tool** (A7). Its `read` lists a directory's entries (O1).
- **Pi has a separate `ls`** (P1), and 0005-MADR plans one for gobble (B3).

### `lsp` — code navigation by symbol

- **grok-build's `lsp` tool** has go-to-definition, find-references and workspace-symbol, among others (K3–K5). It also pushes diagnostics after edits (K15).
- **opencode's `lsp` tool** has the same operations (O2), behind an experimental flag (O4).
- gobble's 0005-PLAN already has X7, "feedback after edits", for diagnostics (B5). The navigation tool shares its LSP client.

### Process sessions — `job_output`, `job_input`, `job_kill`, and `monitor`

- **codex's `exec_command`** takes a yield window (X3), at most 30 s (X10). A command still running when the window ends returns a session id (X4). `write_stdin` then sends that session input (X2).
- **codex caps sessions and plain-texts them.** It runs at most 64 sessions (X9), and forces `PAGER=cat` in them (X11).
- **grok-build's background tasks.** It has `get_task_output` (K7) and `kill_task` (K6), and it reminds the model when a background task completes, so the model need not poll (K14).
- **grok-build's `monitor`** streams a command's output lines as events (K11), under a rate limiter (K12).
- **Kilo's `background_process`** runs and manages long-running processes (L1).
- gobble's 0005-PLAN X2 plans `bash{background:true}`, `job_output` and `job_kill` (B4). It has no input to a running process, and no monitor.

### `notebook` — Jupyter notebooks by cell

- **Kilo has notebook read, edit and execute tools** (L3, L7, L8). They act on `.ipynb` files (L6).
- **A text edit of a notebook is fragile.** A `.ipynb` file is JSON with embedded outputs, so changing it by text replacement can easily leave it invalid. That is rule 4.

### Documents in `read` — PDF, DOCX, XLSX, PPTX

- **goose** extracts text from PDF (G4) and DOCX (G5). Its XLSX tool caps a range at 100,000 cells (G6).
- **grok-build's `read_file`** reads PDF (K9) and PPTX (K10).
- **Kilo's `read`** converts DOCX (L4) and XLSX (L5).
- **grok-build and Kilo fold documents into `read`** rather than adding tools.

### `request_permissions` — a scoped grant on request

- **codex's `request_permissions`** asks the client for read, write or network access for the turn or the session (X5). It is specified beside `exec_command` (X6), so a command that needs more access can ask instead of failing.

### Deferred exposure — keeping the tool list small

- **codex** has a deferred exposure (X7), and a `tool_search` that finds deferred tools (X8), ranked with BM25 (X12).
- **grok-build's `search_tool`** also ranks with BM25 (K16).
- **gobble already has the pieces.** `tool.Exposure` has `deferred` (B6), and 0005-PLAN F1d builds `tool_search`.
- **The cost.** Every tool's description is sent with every request, so a tool that is rarely needed can be deferred and found when wanted.

## Also verified

- **goose's `read_image`** caps an image at 20 MiB (G7) and takes a crop rectangle (G8). gobble's image reading waits for the provider SDK to carry images (0005-PLAN, F1 made executable, decision 3), so this is noted, not proposed.
- **grok-build's `web_fetch`** refuses private and loopback addresses (K13). 0005-PLAN X3 plans `web_fetch`. This is the design to follow when it is built.

## Schemas, descriptions and loading (2026-10-08)

The owner proposed, on 2026-10-08:

> For those native tools let's determine their schemas. I propose using json-rpc 2.0, and for tool descriptions use hybrid markdown. Lazy load all but tool_search and write a harness instruction to always consult tool_search before performing file or directory operations. Evaluate this, offer other ideas, recommendations, pros and cons of the approach. If we want or need bm25 search we can use bleve for that.

This section evaluates the proposal. It decides nothing. Claims marked with an id are in the evidence table. Claims about providers are quoted from their documentation, fetched on 2026-10-08, and listed under "Provider documentation" below: the script cannot check a web page.

### 1. JSON-RPC 2.0 for the schemas

**What the code shows.** Two separate layers are involved.

- **A model never sees JSON-RPC.** It sees a tool's name, description and JSON Schema in its provider's own field:
  - gobble derives the schema from each tool's Go input type (S1, S2) and hands it to go-llmprovider-sdk (S3), whose `Tool` is just a name, a description and a schema (S4);
  - the SDK then writes it as Anthropic's `input_schema` (S5), OpenAI Responses' `parameters` (S6), or Chat Completions' `parameters` (S7).
- **JSON-RPC 2.0 is the envelope between processes.**
  - ACP frames are JSON-RPC 2.0 (S8).
  - MCP is JSON-RPC 2.0 too, and its tools carry a JSON Schema `inputSchema` (S9).

**Pros of the proposal.**

- One wire format for every boundary gobble has, editor and MCP alike.
- A JSON-RPC method per tool would make the native tools callable from outside gobble.

**Cons.**

- JSON-RPC 2.0 has no schema language: a method's `params` are whatever the two sides agree. So it cannot be the tool schemas.
- A request carries an `id` and a method name that the provider wire has no place for.

**Recommendation.**

- **The schema is JSON Schema (2020-12),** derived from Go input types as now, with one convention for every tool:
  - `camelCase` parameter names, as `oldText` is now;
  - every field described;
  - optional fields `omitzero`;
  - `additionalProperties: false`;
  - enums for fixed choices;
  - numeric bounds stated.
- **JSON-RPC 2.0 stays the transport,** as it already is through ACP and MCP. Its natural home for the native tools is X4's `gobble mcp serve` (0005-MADR, `gobble mcp serve` row). That server could expose the native tools as MCP tools, so any MCP host could use gobble's confined, permission-aware file operations.
- **A per-provider schema adapter is a later need:** opencode rewrites tool schemas per model (S10). OpenAI's strict mode, for example, wants every property required.

### 2. Hybrid Markdown descriptions

**What the code shows.** All four harnesses with written descriptions use Markdown.

- opencode keeps each description in a Markdown file with headings (D3).
- codex keeps some as Markdown templates (D4).
- Pi's `tool_search` description is Markdown with a heading (D2). Pi also puts usage guidelines in the system prompt, apart from the description (D1).

**Pros.**

- Headings and lists give a model a predictable shape: what the tool does, when to use it, its rules, an example.
- Tool search covers names, descriptions and argument descriptions (Anthropic's documentation), so the descriptive words also make a tool findable.

**Cons.**

- Every description is sent with every request. Markdown structure costs tokens.
- A long body crowds out the first line, which is what a catalog or a search result shows.

**Recommendation: one shape for every tool's description.**

1. A first sentence that stands alone, the summary a search result and a catalog show.
2. Short Markdown sections: **Use when**, **Rules**, and one **Example** where the arguments are not obvious.
3. Each description is an embedded `.md` file, templated with the OS, the shell and the limits.
4. A budget per tool, checked by a test.
5. Guidance that spans tools, such as "prefer read to cat" or "a delete always asks", goes in one system-prompt section, as Pi does (D1), not repeated in each description.

### 3. Lazy-loading everything but `tool_search`

**What the six harnesses do.** None defers its core tools.

- codex defers only third-party tools, and keeps its own direct (T1). It gives deferral guidance only to models that support search (T2).
- Pi's `tool_search` is itself off until a tool is deferred (T3), and it searches only deferred tools (T4).
- grok-build's search finds MCP tools (T5).

**What the providers say.**

- **Keep the frequent tools loaded.**
  - Anthropic: "Keep your 3–5 most frequently used tools non-deferred so Claude can call them without searching first."
  - OpenAI: keep "a small set of functions, or functions needed for most tasks" loaded, and defer "a large catalog where each task needs only a few functions".
- **Deferral is for many tools.** Anthropic: standard tool calling "is a better fit when you have fewer than 10 tools", and tool selection accuracy "degrades once you exceed 30–50 available tools".
- **Changing tools breaks the cache.**
  - Both providers' caches treat tools as the first part of the prompt prefix.
  - Anthropic: "Modifying tool definitions (names, descriptions, parameters) invalidates the entire cache."
  - OpenAI lists changes to "tool names, descriptions, schemas, ordering" as changes to the prefix.
  - A cache hit costs a tenth of the input price on Anthropic, and less on some models.
- **Native deferral keeps the cache.**
  - Anthropic's server-side tool search keeps deferred tools out of the prefix and adds a found tool inline, so "prompt caching is preserved". It refuses a request in which every tool is deferred.
  - OpenAI's tool search is "designed to preserve the model's cache", but "changing the loaded tool set will break the model's cache from that point forward", and it needs `gpt-5.4` or later.

**What gobble's SDK offers.** go-llmprovider-sdk `v1.2.1` has no deferred loading, no tool references and no cache control (T6). So a gobble-side `tool_search` can only load a tool by changing the `tools` list, which changes the prefix and loses the cache.

**Pros of the proposal.**

- The smallest first prompt.
- Selection stays accurate however many MCP tools a session has.
- One discovery path for everything.

**Cons.**

- **Every task starts with a search.** A task that touches a file begins with a `tool_search` call: one more round trip, and more tokens, before any work.
- **The cache breaks per family.** Each newly loaded set breaks the prompt cache, on every provider gobble uses, until the SDK can defer tools natively. That is the cost the cache was meant to save.
- **An instruction is not enforcement.** A model may skip the search, and with bash deferred too it can do nothing until it searches.
- **The model routes around deferred file tools.** If file operations are deferred but bash is loaded, the model will use `rm` and `mv`, which defeats the reason the native tools exist (the rule's points 2 and 3). codex closes the same gap for patches by catching `apply_patch` run through its shell tool and running the native tool instead (S11).
- **It runs against both providers' guidance** and every surveyed harness.

**Recommendation: two tiers, and enforcement in the harness, not the prompt.**

1. **Direct, always loaded:**
   - read, write, edit (or `apply_patch` for GPT-family models);
   - grep, find, `tree`;
   - move, delete, copy, mkdir;
   - bash, and powershell on Windows;
   - `tool_search`.

   That is 13 on Windows, near Anthropic's "fewer than 10" and far below "30–50". The file operations are direct because a deferred one loses to bash.
2. **Deferred, found by `tool_search`:** `lsp`, `notebook`, `monitor`, `request_permissions`, the job tools, and every MCP tool.
   - **They load by family.** One search for "jobs" loads `job_output`, `job_input`, `job_kill` and `monitor`. That means fewer round trips, and fewer cache breaks.
   - **The harness loads a family when an event needs it.** For example, when bash returns a job id, the job tools load at once.
3. **The prompt names the categories, not a procedure.** One system-prompt section says which kinds of tools `tool_search` can find, as Anthropic advises ("Add a system prompt section describing available tool categories").
4. **Shell file operations are caught in the harness.** F6 parses bash commands with `mvdan.cc/sh` (0005-MADR amendment of 2026-10-07, choice 5). When a command is a plain `rm`, `mv`, `cp` or `mkdir`, the harness:
   - runs the native tool instead, as codex does for patches (S11);
   - or refuses with "use `delete`".

   Either keeps the native tools' confinement and undo whatever the model chooses.
5. **The SDK gains native deferral and cache control.** Then gobble uses each provider's own deferral (`defer_loading` with tool references on Anthropic; `tool_search` with `defer_loading` on OpenAI) and keeps its cache. This is a request to go-llmprovider-sdk, recorded here and not made in that repository.

### 4. BM25 through bleve

**What the code and a measurement show.**

- **bleve can score with BM25.** It does so only when an index's mapping asks (M1); its default is TF-IDF (M2).
- **Its cgo is optional.** faiss vector search needs the `vectors` build tag (M3).
- **cobra stays out.** cobra is used by bleve's own CLI (M4) and was not required by a library build: it is absent from the scratch module's `go.mod`.
- **Pi's `tool_search` ranks with an in-house BM25** (M5), about 110 non-blank lines with its tokenizer and stop words (`tool-search/tool.ts:39-156`).

**The measurement.** A scratch module indexed six tool descriptions in memory with BM25, and searched them once (`bleveprobe`, 2026-10-08, built `CGO_ENABLED=0 -trimpath -ldflags "-s -w"` on Windows):

| Measure | Value |
| :--- | :--- |
| bleve's latest version, from the module proxy | `v2.6.1` (2026-08-24) |
| bleve's own `require` lines | 37 |
| the probe's `go.mod` requirements after `go mod tidy` | 27, including `go-faiss`, `bbolt` and protobuf as indirect |
| the probe's `go.sum` lines | 70 |
| non-standard packages in the probe's build | 124 |
| the probe's binary | 15,223,808 bytes |
| a hello-world binary, same flags | 1,650,688 bytes |
| gobble's binary today, same flags | 14,364,160 bytes, from 24 requirements |

The query "rename a directory" ranked `move` first (1.162), well ahead of `mkdir` (0.124), so the scoring works.

**Pros of bleve.**

- Mature, pure Go without the `vectors` tag.
- BM25 is built in.
- Persistent on-disk indexes, analyzers and stemming come with it.

**Cons for `tool_search`.**

- **It adds about 13.6 MB.** The probe's binary is about 13.6 MB larger than hello-world, which would nearly double gobble's binary.
- **It more than doubles gobble's requirements,** and every new module enlarges what `govulncheck` and the release must cover.
- **The corpus is tiny.** It is tens to hundreds of tool descriptions, held in memory, which Pi ranks in about 110 lines (M5).

**Recommendation.**

- An in-house BM25 for `tool_search` (0005-PLAN F1d), with tokenizing, stop words and per-field weights, and no dependency.
- Reconsider bleve when a feature needs a persistent full-text index over a large corpus, and name it in that feature's MADR:
  - searching past sessions (X1);
  - project memory recall (X1);
  - documents (X8).

### 5. Draft schemas for the native tools

Parameters as JSON Schema properties. "req" marks a required property; everything else is optional, with its default. Paths resolve against the working directory and are confined (F1a).

| Tool | Parameters |
| :--- | :--- |
| `read` | `path` req; `offset` integer ≥ 1; `limit` integer 1–2000 |
| `write` | `path` req; `content` req |
| `edit` | `path` req; `edits` req: array, at least 1, of `{oldText` req, `newText` req`}` |
| `apply_patch` | `patch` req: the envelope (0010-REPORT §1) |
| `grep` | `pattern` req; `path`; `glob`; `ignoreCase` boolean; `literal` boolean; `context` integer 0–10; `limit` integer 1–1000, default 100 |
| `find` | `pattern` req (a glob); `path`; `limit` integer 1–5000, default 1000 |
| `tree` | `path`; `depth` integer 1–10, default 2; `limit` integer 1–2000 entries, default 500 |
| `move` | `from` req; `to` req; `overwrite` boolean, default false |
| `delete` | `path` req; `recursive` boolean, default false |
| `copy` | `from` req; `to` req; `overwrite` boolean, default false |
| `mkdir` | `path` req |
| `bash`, `powershell` | `command` req; `timeout` integer 1–600 seconds, default 120; `workdir` |
| `job_output` | `id` req; `wait` integer 0–300 seconds, default 0 |
| `job_input` | `id` req; `text` req; `close` boolean (close the job's input after) |
| `job_kill` | `id` req |
| `monitor` | `command` req; `timeout` integer seconds; `workdir` |
| `lsp` | `operation` req: one of `definition`, `references`, `hover`, `implementation`, `documentSymbols`, `workspaceSymbols`; `path`; `line` and `column` integers ≥ 1; `query` |
| `notebook` | `path` req; `operation` req: one of `read`, `edit`, `insert`, `delete`; `cell` integer ≥ 0; `source`; `cellType` one of `code`, `markdown` |
| `request_permissions` | `permissions` req: `{read` array of paths, `write` array of paths, `network` boolean`}`; `reason` req; `scope` one of `turn`, `session`, default `turn` |
| `tool_search` | `query` req; `limit` integer 1–20, default 5 |

**Open questions for the decision.** These are left to the owner:

- whether results also get a structured form for clients, beside the text for the model. MCP has `outputSchema`; opencode's rewrite separates a structured result from the model's text (0010-REPORT §1, Tool framework);
- whether schemas follow one provider's strict rules everywhere, or are adapted per provider (S10).

### Provider documentation (quoted 2026-10-08)

- Anthropic, prompt caching (`platform.claude.com/docs/en/docs/build-with-claude/prompt-caching`):
  - "Cache prefixes are created in the following order: `tools`, `system`, then `messages`."
  - "Modifying tool definitions (names, descriptions, parameters) invalidates the entire cache."
  - "Cache read tokens are 0.1 times the base input tokens price (see the table footnote for per-model exceptions)."
- Anthropic, tool search tool (`platform.claude.com/docs/en/agents-and-tools/tool-use/tool-search-tool`):
  - "Keep your 3–5 most frequently used tools non-deferred so Claude can call them without searching first."
  - "Claude's ability to pick the right tool degrades once you exceed 30–50 available tools."
  - Standard tool calling "is a better fit when you have fewer than 10 tools".
  - "The prefix is untouched, so prompt caching is preserved."
  - "At least one tool must have defer_loading=false. All tools cannot be deferred."
  - "Add a system prompt section describing available tool categories."
- OpenAI, prompt caching (`developers.openai.com/api/docs/guides/prompt-caching`):
  - "Prompt caching is enabled by default for supported OpenAI models."
  - "Cache reuse requires the entire rendered prefix to match." It lists changes to "tool names, descriptions, schemas, ordering" as changes to the prefix.
- OpenAI, tool search (`developers.openai.com/api/docs/guides/tools-tool-search`):
  - "only `gpt-5.4` and later models support `tool_search`."
  - "tool search is designed to preserve the model's cache."
  - "changing the loaded tool set will break the model's cache from that point forward."
  - Keep loaded "a small set of functions, or functions needed for most tasks."

## Evidence

| Result | Id | Claim | Where | Matched |
| :--- | :--- | :--- | :--- | :--- |
| pass | G1 | goose's developer extension registers a tree tool | goose `crates/goose/src/agents/platform_extensions/developer/mod.rs:161` | `"tree".to_string(),` |
| pass | G2 | goose's tree walks with the ignore crate (gitignore-aware) | goose `crates/goose/src/agents/platform_extensions/developer/tree.rs:5` | `use ignore::WalkBuilder;` |
| pass | G3 | goose's tree takes a depth | goose `crates/goose/src/agents/platform_extensions/developer/tree.rs:13` | `#[serde(default = "default_depth")]` |
| pass | K1 | grok-build's list_dir is gitignore-aware | grok-build `crates/codegen/xai-grok-tools/src/implementations/grok_build/list_dir/mod.rs:280` | `/// Shared 'WalkBuilder' for seed and deep walk (same 'RespectGitignore' flags).` |
| pass | K2 | grok-build's list_dir has an output budget | grok-build `crates/codegen/xai-grok-tools/src/implementations/grok_build/list_dir/mod.rs:7` | `//! Seeds depth-1 children (capped at 'MAX_SEED_ITEMS') before the budgeted deep walk` |
| pass | O1 | opencode's read lists a directory's entries | opencode `packages/opencode/src/tool/read.ts:102` | `const items = yield* fs.readDirectoryEntries(filepath)` |
| pass | P1 | Pi has a separate ls tool | pi `packages/coding-agent/src/core/tools/index.ts:95` | `export type ToolName = "read" \| "bash" \| "powershell" \| "edit" \| "write" \| "grep" \| "find"` |
| pass | K3 | grok-build's lsp tool has goToDefinition | grok-build `crates/codegen/xai-grok-tools/src/implementations/lsp/types.rs:109` | `GoToDefinition,` |
| pass | K4 | grok-build's lsp tool has findReferences | grok-build `crates/codegen/xai-grok-tools/src/implementations/lsp/types.rs:110` | `FindReferences,` |
| pass | K5 | grok-build's lsp tool has workspaceSymbol | grok-build `crates/codegen/xai-grok-tools/src/implementations/lsp/types.rs:114` | `WorkspaceSymbol,` |
| pass | O2 | opencode's lsp tool has goToDefinition | opencode `packages/opencode/src/tool/lsp.ts:12` | `"goToDefinition",` |
| pass | O3 | opencode's registry gates the lsp tool | opencode `packages/opencode/src/tool/registry.ts:29` | `import { LspTool } from "./lsp"` |
| pass | X1 | codex defines exec_command | codex `codex-rs/core/src/tools/handlers/shell_spec.rs:15` | `pub fn create_exec_command_tool(options: CommandToolOptions) -> ToolSpec {` |
| pass | X2 | codex defines write_stdin | codex `codex-rs/core/src/tools/handlers/shell_spec.rs:117` | `pub fn create_write_stdin_tool() -> ToolSpec {` |
| pass | X3 | exec_command takes a yield window | codex `codex-rs/core/src/tools/handlers/shell_spec.rs:30` | `let yield_time_ms_description = if cfg!(windows) {` |
| pass | X4 | a still-running command returns a session id | codex `codex-rs/core/src/tools/handlers/shell_spec.rs:120` | `"session_id".to_string(),` |
| pass | K6 | grok-build has kill_task | grok-build `crates/codegen/xai-grok-tools/src/implementations/grok_build/kill_task/mod.rs:1` | `//! 'kill_task' tool — new architecture ('Tool' trait).` |
| pass | K7 | grok-build has get_task_output | grok-build `crates/codegen/xai-grok-tools/src/implementations/grok_build/task_output/mod.rs:1` | `//! 'get_task_output' tool — output/status for one or many background tasks.` |
| pass | K8 | grok-build gives & a meaning per shell | grok-build `crates/codegen/xai-grok-config/src/shell.rs:167` | `pub fn ampersand_semantics(&self) -> AmpersandSemantics {` |
| pass | L1 | Kilo has background_process | kilocode `packages/opencode/src/kilocode/tool/background-process.txt:1` | `Run and manage long-running background processes.` |
| pass | L2 | Kilo has notebook tools | kilocode `packages/opencode/src/kilocode/tool/notebook.ts:48` | `return Readable.from([cells.length ? cells.join("\n\n") : "(Notebook contains no markdown` |
| pass | L3 | Kilo registers notebook tools | kilocode `packages/opencode/src/kilocode/tool/registry.ts:13` | `import { NotebookEditTool, NotebookExecuteTool, NotebookReadTool } from "./notebook-host"` |
| pass | G4 | goose's pdf_tool extracts text | goose `crates/goose-mcp/src/computercontroller/pdf_tool.rs:20` | `"extract_text" => {` |
| pass | G5 | goose's docx_tool extracts text | goose `crates/goose-mcp/src/computercontroller/docx_tool.rs:226` | `fn extract_text_from_docx(docx: &Docx) -> String {` |
| pass | G6 | goose's xlsx_tool caps a range at 100k cells | goose `crates/goose-mcp/src/computercontroller/xlsx_tool.rs:8` | `const MAX_RANGE_CELLS: u64 = 100_000;` |
| pass | K9 | grok-build's read_file reads PDF | grok-build `crates/codegen/xai-grok-tools/src/implementations/read_file/pdf.rs:1` | `//! PDF text extraction and page rendering shared by read tools.` |
| pass | K10 | grok-build's read_file reads PPTX | grok-build `crates/codegen/xai-grok-tools/src/implementations/read_file/pptx.rs:1` | `//! PPTX text extraction shared by read tools.` |
| pass | L4 | Kilo's read converts docx | kilocode `packages/opencode/src/kilocode/tool/read-extract.ts:2` | `import * as Docx from "./read-docx"` |
| pass | L5 | Kilo's read converts xlsx | kilocode `packages/opencode/src/kilocode/tool/read-extract.ts:4` | `import * as Xlsx from "./xlsx"` |
| pass | L6 | Kilo's notebook tools read .ipynb (read-extract does not) | kilocode `packages/opencode/src/kilocode/tool/notebook.ts:30` | `return path.extname(filepath).toLowerCase() === ".ipynb"` |
| pass | K11 | grok-build has monitor | grok-build `crates/codegen/xai-grok-tools/src/implementations/grok_build/monitor/types.rs:22` | `/// Default monitor timeout (non-persistent). 10 hours to avoid short` |
| pass | K12 | monitor is rate-limited | grok-build `crates/codegen/xai-grok-tools/src/implementations/grok_build/monitor/rate_limiter.rs:3` | `use super::types::{AUTO_KILL_THRESHOLD_MS, RATE_LIMIT_REFILL_MS};` |
| pass | X5 | codex has request_permissions | codex `codex-rs/core/src/tools/handlers/request_permissions.rs:1` | `use codex_protocol::permissions::FileSystemSandboxPolicyContext;` |
| pass | X6 | request_permissions is specified beside exec | codex `codex-rs/core/src/tools/handlers/shell_spec.rs:161` | `pub fn create_request_permissions_tool(description: String) -> ToolSpec {` |
| pass | X7 | codex has a deferred tool exposure | codex `codex-rs/tools/src/tool_executor.rs:35` | `ToolExposureSurface::Deferred => Self::DEFERRED,` |
| pass | X8 | codex's tool_search loads deferred tools | codex `codex-rs/core/src/tools/handlers/tool_search_spec.rs:16` | `pub(crate) fn create_tool_search_tool(` |
| pass | B1 | gobble's tool.Kind has delete | gobble-cli `tool/tool.go:43` | `KindDelete  Kind = "delete"` |
| pass | B2 | gobble's tool.Kind has move | gobble-cli `tool/tool.go:44` | `KindMove    Kind = "move"` |
| pass | B3 | 0005-MADR plans an ls tool | gobble-cli `docs/decisions/0005-MADR-v1-feature-scope.md:125` | `\| 'ls' \| 1.0 \| ~~'{path?, limit?=500}'. Sorted, with a '/' suffix on directories.~~ *(2026` |
| pass | B4 | 0005-PLAN X2 plans job_output | gobble-cli `docs/decisions/0005-PLAN-v1-feature-scope.md:387` | `*(2026-10-08, 0005-MADR amendment "native tools": step 3 is replaced by process sessions:` |
| pass | B5 | 0005-PLAN has X7, feedback after edits | gobble-cli `docs/decisions/0005-PLAN-v1-feature-scope.md:487` | `### X7 — feedback after edits 1.x` |
| pass | B6 | gobble's tool.Exposure has deferred | gobble-cli `tool/tool.go:58` | `// ExposureDeferred keeps the tool behind search.` |
| pass | K13 | grok-build's web_fetch blocks private and loopback addresses | grok-build `crates/codegen/xai-grok-tools/src/implementations/grok_build/web_fetch/ssrf.rs:4` | `//! - Non-public addresses (loopback, RFC 1918, link-local, CGNAT, TEST-NET,` |
| pass | G7 | goose's read_image caps an image at 20 MiB | goose `crates/goose/src/agents/platform_extensions/developer/image.rs:14` | `const MAX_IMAGE_BYTES: u64 = 20 * 1024 * 1024;` |
| pass | G8 | goose's read_image takes a crop rectangle | goose `crates/goose/src/agents/platform_extensions/developer/image.rs:29` | `pub crop: Option<CropParams>,` |
| pass | X9 | codex runs at most 64 exec sessions | codex `codex-rs/core/src/unified_exec/mod.rs:82` | `pub(crate) const MAX_UNIFIED_EXEC_PROCESSES: usize = 64;` |
| pass | X10 | an exec_command yield is at most 30 s | codex `codex-rs/core/src/unified_exec/mod.rs:77` | `pub(crate) const MAX_YIELD_TIME_MS: u64 = 30_000;` |
| pass | X11 | codex's sessions force PAGER=cat | codex `codex-rs/core/src/unified_exec/process_manager.rs:100` | `("PAGER", "cat"),` |
| pass | X12 | codex's tool_search ranks with BM25 | codex `codex-rs/core/src/tools/handlers/tool_search.rs:14` | `use bm25::Embedder;` |
| pass | K14 | grok-build reminds the model when a background task completes | grok-build `crates/codegen/xai-grok-tools/src/reminders/task_completion.rs:1` | `//! Background task and subagent completion reminder.` |
| pass | K15 | grok-build pushes LSP diagnostics after edits | grok-build `crates/codegen/xai-grok-tools/src/reminders/lsp_diagnostics.rs:1` | `//! Cross-cutting reminder: notifies LSP of file changes and drains diagnostics.` |
| pass | K16 | grok-build's search_tool ranks with BM25 | grok-build `crates/codegen/xai-grok-tools/src/implementations/search_tool/mod.rs:1` | `//! 'search_tool' — discover MCP tools via BM25 keyword search.` |
| pass | O4 | opencode's lsp tool is behind a flag | opencode `packages/opencode/src/tool/registry.ts:247` | `...(flags.experimentalLspTool ? [tool.lsp] : []),` |
| pass | L7 | Kilo can run notebook cells | kilocode `packages/opencode/src/kilocode/tool/notebook-host.ts:189` | `export const NotebookExecuteTool = Tool.define<` |
| pass | L8 | Kilo edits notebook cells | kilocode `packages/opencode/src/kilocode/tool/notebook-host.ts:133` | `export const NotebookEditTool = Tool.define<` |
| pass | A1 | opencode registers no native file-operation, diff, git, checksum or archive tool | opencode `packages/opencode/src/tool/registry.ts` | matching lines: none |
| pass | A2 | Pi's built-in tool names include none of them | pi `packages/coding-agent/src/core/tools/index.ts` | matching lines: none |
| pass | A3 | goose's developer extension registers none of them | goose `crates/goose/src/agents/platform_extensions/developer/mod.rs` | matching lines: none |
| pass | A4 | Kilo's tool registry registers none of them | kilocode `packages/opencode/src/kilocode/tool/registry.ts` | matching lines: none |
| pass | A5 | codex has no handler for any of them | codex `codex-rs/core/src/tools/handlers` | 57 entries, matching: none |
| pass | A6 | grok-build has no tool directory for any of them | grok-build `crates/codegen/xai-grok-tools/src/implementations/grok_build` | 32 entries, matching: none |
| pass | A7 | opencode has no ls or tree tool; read lists directories | opencode `packages/opencode/src/tool` | 41 entries, matching: none |
| pass | S1 | a gobble tool's schema is a JSON value, inputSchema | gobble-cli `tool/tool.go:22` | `InputSchema jsontext.Value 'json:"inputSchema,omitzero"'` |
| pass | S2 | the schema is derived from the input's Go type | gobble-cli `tool/tool.go:291` | `schema, err := jsonschema.For[In](nil)` |
| pass | S3 | gobble hands the provider SDK a name, a description and the JSON Schema | gobble-cli `llm/provider/provider.go:234` | `out.Tools = append(out.Tools, llmprovider.Tool{Name: s.Name, Description: s.Description, S` |
| pass | S4 | the SDK's Tool is a name, a description and a schema | go-llmprovider-sdk@v1.2.1 `llmprovider/provider.go:17` | `type Tool struct {` |
| pass | S5 | the SDK sends the schema as Anthropic's input_schema | go-llmprovider-sdk@v1.2.1 `llmprovider/internal/wire/tools.go:55` | `list[i] = map[string]any{KeyName: tool.Name, keyDescription: tool.Description, "input_sche` |
| pass | S6 | and as OpenAI Responses' parameters | go-llmprovider-sdk@v1.2.1 `llmprovider/internal/wire/tools.go:30` | `keyParameters:  ToolSchema(tool.Schema),` |
| pass | S7 | and as Chat Completions' parameters | go-llmprovider-sdk@v1.2.1 `llmprovider/internal/wire/chatcompletions/chatcompletions.go:26` | `keyParameters      = "parameters"` |
| pass | S8 | ACP frames are JSON-RPC 2.0 | acp-go-sdk@v0.13.6-mcr.2 `connection.go:708` | `res := anyMessage{JSONRPC: "2.0"}` |
| pass | S9 | an MCP tool's schema is JSON Schema, inputSchema | mcp go-sdk@v1.8.0 `mcp/protocol.go:1921` | `InputSchema any 'json:"inputSchema"'` |
| pass | S10 | opencode rewrites tool schemas per model | opencode `packages/opencode/src/provider/transform.ts:1575` | `export function schema(model: Provider.Model, schema: JSONSchema7): JSONSchema7 {` |
| pass | S11 | codex intercepts apply_patch run through its shell tool | codex `codex-rs/core/src/tools/handlers/unified_exec/exec_command.rs:372` | `let intercepted_patch = intercept_apply_patch(` |
| pass | D1 | Pi puts a tool's usage guidelines in the system prompt, apart from its description | pi `packages/coding-agent/src/core/tools/read.ts:78` | `promptGuidelines: [...readToolSystemPromptContribution.guidelines],` |
| pass | D2 | Pi's tool_search description is Markdown with a heading | pi `packages/coding-agent/src/extensions/tool-search/tool.ts:229` | `return '# Tool discovery\n\nSearches over deferred tool metadata with BM25 and exposes mat` |
| pass | D3 | opencode's descriptions are Markdown files | opencode `packages/opencode/src/tool/todowrite.txt:3` | `## When to use` |
| pass | D4 | codex keeps descriptions as Markdown templates | codex `codex-rs/core/templates/search_tool/tool_description.md:1` | `# Apps (Connectors) tool discovery` |
| pass | T1 | codex defers third-party tools, and keeps its own direct | codex `codex-rs/core/src/tools/spec_plan.rs:588` | `/// Run after namespace overrides so eligible third-party tools always remain deferred.` |
| pass | T2 | codex gives deferral guidance only to models that support search | codex `codex-rs/core/src/tools/spec_plan.rs:879` | `let deferred_tools_guidance_enabled = model_info.supports_search_tool;` |
| pass | T3 | Pi's tool_search is itself off until a tool is deferred | pi `packages/coding-agent/src/extensions/tool-search/index.ts:14` | `pi.registerTool({ ...createToolSearchToolDefinition({ tools: pi }), defaultActive: false }` |
| pass | T4 | Pi's tool_search searches only deferred tools | pi `packages/coding-agent/src/extensions/tool-search/tool.ts:192` | `return exposure === "codemode" \|\| exposure === "deferred";` |
| pass | T5 | grok-build's search finds MCP tools | grok-build `crates/codegen/xai-grok-tools/src/implementations/search_tool/mod.rs:1` | `//! 'search_tool' — discover MCP tools via BM25 keyword search.` |
| pass | T6 | the SDK has no deferred loading, tool references or cache control | go-llmprovider-sdk@v1.2.1 `llmprovider` | `90 files, matching: 0` |
| pass | M1 | bleve scores with BM25 when the mapping asks | bleve@v2.6.1 `index_alias_impl.go:629` | `rv = m.ScoringModel == index.BM25Scoring` |
| pass | M2 | bleve's default scoring is TF-IDF | bleve_index_api@v1.4.1 `indexing_options.go:37` | `const DefaultScoringModel = TFIDFScoring` |
| pass | M3 | bleve's faiss vector search needs the vectors build tag | bleve@v2.6.1 `search_knn.go:15` | `//go:build vectors` |
| pass | M4 | cobra is used by bleve's own CLI, not its library | bleve@v2.6.1 `cmd/bleve/cmd/bulk.go:24` | `"github.com/spf13/cobra"` |
| pass | M5 | Pi's tool_search is an in-house BM25 ranker | pi `packages/coding-agent/src/extensions/tool-search/tool.ts:2` | `* Tool discovery: a BM25 ranker over tool metadata, shared by 'searchTools()' in codemode` |
