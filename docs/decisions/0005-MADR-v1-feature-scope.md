---
status: proposed
date: 2026-10-04
decision-makers: repository owner
consulted: 0001-REPORT-go-port-feasibility.md, 0002-MADR-cli-acp-headless-mcp-v1.md, 0003-MADR-gobble-product-identity.md, 0004-MADR-go-module-architecture.md
informed: magic-cli-remote (companion command table), go-llmprovider-sdk, go-core-lib, go-tui-lib
---
# The v1 line carries every portable Pi capability, plus the ones Pi left to example extensions, tiered into a v1.0.0 gate, a v1.x train, and `exp/`

## Context and Problem Statement

[0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md)
cut v1 narrowly. Its scope was the Cobra CLI, the ACP stdio agent, the MCP
client, and native slash commands. It put the Charm TUI, the package
manager, HTML export, and "anything else" out of v1. The owner has since
asked to include as much as possible in v1, to be creative, and to mine
the original Pi sources for features.

A survey of Pi at `312184edb` (see More Information for paths) found
three kinds of capability:

1. **Built in and portable.** These are:
   * the eight tools, with exact limits;
   * the agent loop with steering and follow-up queues;
   * retry, auto-compaction, and branch summaries;
   * session JSONL v3 with a tree, labels, and context edits;
   * settings with patch-merge rules;
   * `AGENTS.md`/`CLAUDE.md` discovery, `SYSTEM.md`/`APPEND_SYSTEM.md`,
     and a sectioned system prompt;
   * Agent Skills, and prompt templates with shell-style argument
     substitution;
   * an MCP client with OAuth and exposure modes, plus `tool_search`
     (BM25);
   * project trust, cache warming, image auto-resize, HTML export, and
     `!cmd` shell input.
2. **Shipped only as example extensions.** Subagents
   (`examples/extensions/subagent/`), a todo tool (`todo.ts`), plan mode
   (`plan-mode/`), git checkpoints (`git-checkpoint.ts`), a permission
   gate, protected paths, and a sandbox. Pi has no built-in permission
   prompts, no background jobs, and no file-level undo. Its docs say so.
3. **Language-bound.** The jiti TypeScript extension host, the TypeScript
   SDK and browser `pi-ai`, and codemode's QuickJS VM. Also the
   experimental second product (Chord, `pi-durable`, `pi-protocol`,
   `pi-server`).

magic-cli-remote's canonical vocabulary (its MADR 0023) has 30 commands.
0002 had to mark 18 of them `KindNone` because the handler did not exist:
`/undo`, `/redo`, `/diff`, `/ps`, `/stop`, `/review`, `/permissions`, and
others. Several of those handlers are exactly the kind-2 features above.

[0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md)
gives each capability a package, and `exp/` a home for unstable ones. The
problem: decide what the v1 line contains, and how to keep a wide scope
shippable.

## Decision Drivers

* Owner: include as much as possible in v1, and take ideas from Pi.
* The magic-cli-remote-native contract from 0002 is unchanged: advertise
  only what executes, and emit the ACP events mcremote consumes. Every
  canonical command that gobble can execute natively is one fewer
  `KindNone` on the phone.
* Pi compatibility where Pi's formats are data: session v3, `mcp.json`,
  `models.json`, `SKILL.md`, prompt templates, and settings key names. A
  Pi user's assets keep working (see
  [0003-MADR-gobble-product-identity.md](0003-MADR-gobble-product-identity.md)).
* Open standards over private extension APIs: MCP for tools, Agent Skills
  for procedures, `AGENTS.md` for instructions, exec hooks with JSON over
  stdio for policy, and ACP for every client.
* A release that can actually be tagged: a wide scope needs a gate subset,
  or `v1.0.0` never ships.

## Considered Options

* Tiered v1 line: a `1.0` gate, a `1.x` train, `exp/`, and an explicit out
  list
* Keep 0002's narrow v1; everything else is v2
* Everything in scope must pass the `v1.0.0` gate (no tiers)
* Full Pi parity including TypeScript extensions through an embedded JS
  runtime

## Decision Outcome

Chosen option: "Tiered v1 line: a `1.0` gate, a `1.x` train, `exp/`, and
an explicit out list". It puts nearly every Pi capability, and the
example-only ones, into the v1 line, and keeps `v1.0.0` reachable.

Tier definitions:

| Tier | Meaning |
|---|---|
| **1.0** | Must work, with its tests, for the `v1.0.0` tag. |
| **1.x** | Decided here. Lands in a `v1.N` minor release without a new MADR, and is advertised only when it works. |
| **exp** | Lives under `exp/`, off by default, enabled by a setting. No compatibility promise. A later MADR promotes or drops it. |
| **out** | Not in the v1 line. A later MADR may revisit. |

### Capability inventory

#### Agent core (`agent`, `compaction`)

| Capability | Tier | Behaviour (Pi source) |
|---|---|---|
| Turn loop with parallel tool batches | 1.0 | Preflight sequentially, execute concurrently, emit results in source order. A tool marked `sequential` makes its batch sequential (`agent/src/types.ts`). |
| Steering and follow-up queues | 1.0 | Steer is injected after the current tool calls finish. Follow-up is injected only when idle. Modes are `one-at-a-time` and `all`. Over ACP these are `_gobble/steer`, `_gobble/follow_up`, and `_gobble/clear_queue`. In the TUI, Enter while busy steers and Alt+Enter queues a follow-up (`agent.ts:134-320`). |
| Retry | 1.0 | Defaults: `enabled`, 3 retries, 2 s base, 60 s cap. Honours `Retry-After`. Classification is by typed error, not regex. Context overflow is never retried; it compacts instead (`ai/src/utils/retry.ts`). |
| Auto-compaction | 1.0 | Trigger when `contextTokens > contextWindow − reserveTokens` (16384). Keep 20000 recent tokens. Never cut at a tool result. The summary is iterative, and a split turn is handled. The summary format has Goal, Constraints, Progress, Decisions, Next Steps, Critical Context, and read/modified file lists. On overflow: compact once, then retry (`docs/compaction.md`). |
| Manual compaction | 1.0 | `/compact [instructions]` and `_gobble/compact`. Emits `usage_update`. |
| Thinking levels | 1.0 | `off`, `minimal`, `low`, `medium`, `high`, `xhigh`, `max`, with a per-model map and budgets (1024, 2048, 8192, 16384). Exposed as ACP config option category `thought_level`. |
| Branch summaries | 1.x | Summarise an abandoned path when navigating the tree (`compaction/branch-summarization.ts`). |
| Context edits | 1.x | `context_edit` entries omit or replace an earlier entry's model context. `_gobble/context_edit`. |
| Prompt-cache warming | 1.x | Refresh at 90% of the TTL when the expected saving is at least $0.05. Recorded as `usage{kind:"cache_warm"}` (`core/cache-warmer.ts`). |
| Provider server tools | 1.x | Pass through provider-hosted tools (web search, code execution) where the adapter's SDK exposes them, enabled per model in settings. |

#### Tools (`tool/builtin`, `tool/toolsearch`)

| Tool | Tier | Behaviour |
|---|---|---|
| `read` | 1.0 | `{path, offset?, limit?}`. Head truncation at 2000 lines or 50 KB, with continuation hints. Images (png, jpeg, gif, webp, bmp) are returned as image content and auto-resized to 2000×2000 and 4.5 MB of base64, keeping the smaller of PNG and JPEG. EXIF orientation is applied. A text note replaces the image for text-only models. Pure Go (`image/*`, `golang.org/x/image`). This decides report D8. |
| `write` | 1.0 | `{path, content}`. Creates parent directories. Goes through the client's `fs/write_text_file` when advertised. |
| `edit` | 1.0 | `{path, edits:[{oldText,newText}]}` multi-edit. Each `oldText` must be unique in the original, and edits must not overlap. Preserves BOM and CRLF. Falls back to fuzzy matching (NFKC, trailing whitespace, smart quotes, dashes, special spaces). Returns ACP `diff` content (`tools/edit.ts`, `edit-diff.ts`). |
| `bash` | 1.0 | `{command, timeout?, background?}`. Tail truncation at 2000 lines or 50 KB, with the full output in `gobble-bash-<id>.log`. The child runs in its own process group, and the whole tree is killed on cancel. Honours `shellPath` and `shellCommandPrefix`. Children see the 0003 environment markers. Uses a client terminal when that is advertised and enabled. |
| `powershell` | 1.0 (windows) | Same contract as `bash`, with a UTF-8 prefix. |
| `grep` | 1.0 | `{pattern, path?, glob?, ignoreCase?, literal?, context?, limit?=100}`. Pure Go: `regexp`, a gitignore-aware walk, and `doublestar` globs. Lines are capped at 500 characters. No ripgrep download. This decides report D13. |
| `find` | 1.0 | `{pattern, path?, limit?=1000}`. Pure Go, includes hidden files, respects `.gitignore`, and skips `.git` and `node_modules`. |
| `ls` | 1.0 | `{path?, limit?=500}`. Sorted, with a `/` suffix on directories. |
| `tool_search` | 1.0 | BM25 over names, descriptions, and schema properties. Loaded tools are recorded in the transcript so they survive resume and fork (`extensions/tool-search/tool.ts`). |
| `todo` | 1.0 | `{todos:[{content,status,priority}]}`. Published as ACP `plan` updates and persisted as `custom{gobble.todo}`. `/todos`. Pi has this only as an example. |
| `task` (subagents) | 1.x | `{agent, prompt, description}`. Runs a child gobble session in-process through ACP, linked by `parentSession`. Progress is relayed as `tool_call_update`. Batches may run in parallel. Depth limit 1 by default. Agents are `agents/*.md` (frontmatter `name`, `description`, `model`, `thinking`, `tools`, `mode`; the body is the system prompt). Pi has this only as an example. |
| `web_fetch` | 1.x | `{url, format: markdown|text|html, maxBytes?}`. HTML is converted to Markdown. Open-world: asks by default. |
| Background jobs | 1.x | `bash{background:true}` returns a job id. `job_output{id, since?}` and `job_kill{id}`. Jobs die when the session closes. Enables `/ps` and `/stop`. Pi has no equivalent. |
| MCP resource tools | 1.0 | `list_mcp_resources`, `list_mcp_resource_templates`, `read_mcp_resource` (`docs/mcp.md`). |
| Exposure modes | 1.0 | `direct`, `deferred` (loaded by `tool_search`), `hidden`. The Pi values `codemode` and `codemode-deferred` are accepted only when `exp/codemode` is enabled; otherwise they are read as `deferred`, with a warning. |
| Per-file mutation queue | 1.0 | Writes to the same real path are serialised; different files still run in parallel (`file-mutation-queue.ts`). |
| Workspace confinement | 1.0 | File tools open paths through `os.Root` over `cwd` plus ACP `additionalDirectories`. A path outside those roots requires a permission decision. |

#### Sessions (`session`, `session/jsonl`)

| Capability | Tier | Behaviour |
|---|---|---|
| JSONL v3 read and write | 1.0 | Every Pi entry type is supported: `message`, `model_change`, `thinking_level_change`, `usage`, `compaction`, `context_edit`, `branch_summary`, `custom`, `custom_message`, `label`, `session_info`. v1 and v2 files migrate when loaded. gobble-only data is stored as `custom{gobble.*}`, so the files stay valid v3. Unknown fields round-trip. This decides report D5 for sessions. |
| ACP session methods | 1.0 | `session/list`, `session/load`, `session/resume`, `session/close`. `session_info_update` carries names and titles. |
| Names, fork, clone | 1.0 | `/name`, `/fork [entry]`, `/clone`, and the matching `_gobble/*` methods. |
| Tree navigation and labels | 1.x | `/tree` prints the tree. `/tree <id>` moves the leaf, with a branch summary. `/label`. The TUI has a tree navigator. |
| Automatic titles | 1.x | After the first turn, a cheap model call sets `session_info.name` when it is empty. |
| Pi import | 1.x | `gobble session list --pi` and `gobble session import`, through the 0003 bridge. |
| Export | 1.x | `gobble session export --format jsonl|md|html` and `/export`. The HTML is self-contained, rendered server-side with a Go Markdown renderer, and needs no JavaScript. |
| Share | 1.x | `/share` through `gh gist create` when `gh` is present. There is no hosted gateway. |
| Archive and delete | 1.x | `/archive` hides a session from lists (`custom{gobble.archived}`). `/delete` removes the session file after an ACP permission request. `_gobble/delete`. The unstable ACP `session/delete` is advertised only under `acp.unstable`. |

#### Context and prompts (`prompt`, `skill`, `command`)

| Capability | Tier | Behaviour |
|---|---|---|
| Context files | 1.0 | Per directory the first match wins: `AGENTS.override.md`, `AGENTS.md`, `CLAUDE.md`. Loaded from the config directory, then from `cwd` and each ancestor, with nested git worktrees de-duplicated. No trust is needed. `--no-context-files` disables it (`resource-loader.ts`). |
| `SYSTEM.md` and `APPEND_SYSTEM.md` | 1.0 | Replace or append to the default prompt. The trusted project copy wins over the user copy. |
| Sectioned system prompt | 1.0 | Named sections (`preamble`, `tools`, `rules`, `project_context`, `skills`, `cwd`, custom). Section changes are stored as system-message patches, so the transcript replays exactly (`core/system-prompt.ts`). |
| Agent Skills | 1.0 | `SKILL.md` frontmatter follows the agentskills.io spec. Discovery order: `.gobble/skills`, config `skills/`, `.agents/skills` from `cwd` up to the repo root, `~/.agents/skills`, then the Pi bridge. Only name, description, and location go into the prompt. `/skill:<name> [args]`. |
| Prompt templates | 1.0 | `prompts/*.md` with `description` and `argument-hint` frontmatter. Substitution supports `$1`, `$@`, `$ARGUMENTS`, `${1:-d}`, `${@:N}`, `${@:N:L}`, with shell-style quoting. Each template is a `/name` command. |
| `@` mentions | 1.0 | `@path` in a prompt becomes an ACP `resource_link` or an embedded resource. Prompt capability `embeddedContext = true`, `image = true`. |
| MCP prompts as commands | 1.x | MCP `prompts/list` entries appear as `/mcp:<server>:<prompt>`. Pi does not do this. |
| Built-in templates | 1.x | `/review` reviews the working-tree diff. `/init` drafts an `AGENTS.md`. The `deep-research` skill uses `task` and `web_fetch`. |

#### Models, providers, auth (`llm`, `llm/provider/*`, `llm/catalog`, `internal/auth`)

| Capability | Tier | Behaviour |
|---|---|---|
| Anthropic Messages | 1.0 | `go-llmprovider-sdk` id `claude` (`llmprovider/providers/claude`). API key. Thinking and caching as that package sends them. |
| OpenAI Responses | 1.0 | SDK id `openai` (`llmprovider/providers/openai`), including the ChatGPT/Codex backend the package already has. API key or that backend's OAuth. |
| Google Gemini | 1.0 | SDK id `gemini` (`llmprovider/providers/gemini`, Interactions wire). |
| Grok (xAI) | 1.0 | SDK id `grok` (`llmprovider/providers/grok`, Responses wire). |
| Other named SDK providers | 1.0 as each implements `llmprovider.Provider` | Ids already in the SDK: `opencode-zen`, `opencode-go`, `huggingface`, `kilo`, `together`, `ollama`. On 2026-09-30 those five still use the old `Generate*` API (0015-PLAN S7). gobble does not wrap that old API; each id joins 1.0 when `providers.New` returns it. |
| Stream shape | 1.0 | gobble calls `llmprovider.Stream`. Every moved provider sets `NativeStreaming` to `Unsupported`, so 1.0 ACP `session/update` text chunks are whole-message events synthesised from `Generate`, not token-by-token SSE. A later SDK `Streamer` is picked up with no gobble API change. |
| Unnamed OpenAI-compatible endpoints | 1.x | Groq, Cerebras, DeepSeek, OpenRouter, Fireworks, Mistral, LM Studio, llama.cpp server, vLLM. The SDK has no generic Chat Completions constructor (`WithBaseURL` on `openai` would send the Responses wire). These land when the SDK adds a Chat Completions provider that takes a base URL, or adds those ids. gobble does not speak those wires itself. |
| Bedrock, Vertex, Azure OpenAI | 1.x | Needs SDK support. gobble will not add `anthropic-sdk-go`, `openai-go/v3`, `google.golang.org/genai`, or `aws-sdk-go-v2` to speak them. |
| Subscription OAuth | 1.0 for ChatGPT/Codex through the SDK `openai` backend; 1.x for any other provider | Only for providers whose terms permit third-party clients. Each extra provider is checked and recorded in the PLAN before it lands. |
| go-llmprovider-sdk adapter | 1.0 | The only 1.0 provider implementation, in `llm/provider`. The "once it has a `go.mod` and public streaming" gate is withdrawn: the module exists, `Stream` exists, native per-token streaming does not. |
| Model catalog | 1.0 | Embedded snapshot in Pi's `Model` shape (context window, max tokens, cost tiers, input modalities, reasoning, thinking map, image input limits). `gobble models list|info`. The source is recorded in `NOTICE` if copied. |
| Catalog refresh | 1.x | `gobble models refresh` into the cache directory. |
| `models.json` | 1.0 | Pi-compatible custom providers and models, with `modelOverrides`. `$ENV`, `${ENV}`, and `!command` values are resolved at request time. |
| Credential store | 1.0 | Precedence: `--api-key`, then keyring (or the 0600 file), then `models.json`, then environment or ambient credentials. `gobble auth login|logout|status|token|import --from-pi`. ACP `authMethods` and `authenticate`/`logout`. |

#### MCP (`mcpclient`)

| Capability | Tier | Behaviour |
|---|---|---|
| stdio and Streamable HTTP | 1.0 | Through the official go-sdk. SSE is rejected. Advertises `mcpCapabilities.http = true`, `sse = false`, `acp = false`. |
| `mcp.json` | 1.0 | Pi-compatible schema (`command`, `args`, `env`, `cwd`, `url`, `headers`, `oauth`, `timeout`, `enabled`, `exposure`, `toolExposure` globs) with `${ENV}` and `!command` values. Unioned with `session/new.mcpServers`; the session's names win. |
| Tool naming and output | 1.0 | `mcp__<server>__<tool>`, at most 64 characters, with a hash suffix on overflow. Output over 20 KB is middle-truncated, with the full text in a file. |
| Lifecycle | 1.0 | Start-up waits up to 10 s. HTTP 408, 429, and 5xx are retried twice; tool calls are never retried. `list_changed` is handled, and the server reconnects on its next use. For a stdio server, stop means close stdin, then SIGTERM, then SIGKILL to the process group. Roots are `cwd` plus the additional directories. Progress notifications reset the timeout. |
| OAuth | 1.0 | Discovery, dynamic client registration, PKCE, a loopback callback guarded by `CrossOriginProtection`, and manual paste. Tokens are kept in the secret store. |
| CLI | 1.0 | `gobble mcp add|remove|list|get|login|logout`. `/mcp [status|reconnect]`. |
| Sampling and elicitation | 1.x | An MCP server's `sampling/createMessage` is served by the session's model after a permission request. Elicitation maps to ACP elicitation when the client advertises it (unstable), otherwise to `session/request_permission`. |
| `gobble mcp serve` | 1.x | gobble as an MCP **server**. It exposes `gobble_run` (prompt to final text in a new session) and `gobble_session_prompt`, so any MCP host can delegate to gobble. |

#### Clients and modes (`acpserver`, `acpclient`, `internal/cli`, `internal/tui`)

| Capability | Tier | Behaviour |
|---|---|---|
| `gobble acp` | 1.0 | The headless ACP agent, per 0002. |
| Print mode | 1.0 | `gobble -p "…"` or piped stdin. `--output-format text|json|stream-json`. `json` is one final object `{text, stopReason, usage, cost, sessionId}`. `stream-json` writes each ACP `session/update` params object as one JSONL line after a header line. The streaming vocabulary is ACP; there is no private event dialect. In non-interactive mode an `ask` decision resolves to deny unless `--yes` or mode `bypass` is set (Pi's "ask means skip"). |
| Charm TUI | 1.0 | `gobble` on a TTY. An ACP client over `acpclient`, on Charm v2 with fang-styled help. Includes: <br>• streamed Markdown transcript; <br>• collapsible tool cards with diffs; <br>• permission dialog; <br>• model, thinking, and mode pickers; <br>• session picker; <br>• multi-line editor with history; <br>• slash and `@path` completion; <br>• `!cmd` and `!!cmd`; <br>• Esc to interrupt; <br>• steer and follow-up keys; <br>• Ctrl+G for the external editor; <br>• a footer with model, mode, context %, and cost. <br>This decides report D6: Charm v2, extracting shared widgets into go-tui-lib as they settle. |
| TUI themes and keybindings | 1.x | Pi theme JSON and `keybindings.json` with Pi's action ids (`app.*`, `tui.*`) where the action exists. |
| Tree navigator, inline images | 1.x / exp | The tree navigator is 1.x. Kitty and iTerm2 inline images are exp. |
| Pi RPC shim | 1.x | `gobble rpc` speaks Pi's JSONL RPC (`docs/rpc-commands.md`) as an ACP client of an in-process agent. Existing Pi RPC clients work, and there is still one command API (0002's objection to dual protocols was two inventories; the shim is a translator with none of its own). Extension-UI records are out. |
| Shell completions and man pages | 1.0 | Through fang and Cobra. |

#### Safety and policy (`permission`, `checkpoint`)

| Capability | Tier | Behaviour |
|---|---|---|
| ACP modes | 1.0 | See the list after this table. |
| Rules | 1.0 | `permissions.allow`, `ask`, and `deny` patterns such as `bash(git status*)`, `edit(src/**)`, `mcp__github__*`, `web_fetch(domain:example.com)`. Deny beats ask, and ask beats allow. Defaults come from each tool's `readOnlyHint` and `destructiveHint`. |
| Permission requests | 1.0 | ACP `session/request_permission` with `allow_once`, `allow_always`, `reject_once`, `reject_always`. `allow_always` writes a rule. `/permissions` lists and edits rules. |
| Project trust | 1.0 | Pi's order: flag, then hook, then the saved decision in `trust.json`, then `defaultProjectTrust`. `gobble trust|untrust`, `/trust`. Protects `.gobble/*` and the project `.agents/skills`. |
| Protected paths | 1.0 | Built-in `deny` rules for `.git/**`, the gobble secret store, and `~/.ssh/**`. |
| Checkpoints | 1.x | See the paragraph after this table. `/undo`, `/redo`, and `/diff [turn]` then execute natively (Pi's `git-checkpoint.ts` idea, made exact). |
| OS sandbox for bash | exp | `exp/sandbox`: macOS Seatbelt profile, Linux Landlock. Opt-in. |

ACP modes:

* `default`: read-only tools allowed; edits and bash ask.
* `accept-edits`: edits allowed; bash asks.
* `plan`: read-only. Leaving plan mode goes through `request_permission`
  carrying the plan.
* `bypass`: no prompts, which is Pi's behaviour. It is advertised only
  when `permissions.allowBypass` is set.

A `current_mode_update` is emitted on every change.

Checkpoints:

* Before each mutating batch, the file contents that `edit` and `write`
  will change are journaled, which is exact.
* When the workspace is a git repository and `git` is on PATH, a
  shadow-index snapshot also covers changes made by bash. It uses its own
  `GIT_INDEX_FILE` and refs under the data directory, and it never touches
  the user's index, stash, or refs.

#### Extensibility (`hook`, packages, report D2/D7)

| Capability | Tier | Behaviour |
|---|---|---|
| Exec hooks | 1.0 | Configured under `hooks` in settings, and from project settings only when trusted. Each hook is `{event, match?, command[], timeout}`. The event is sent as JSON on stdin and a JSON outcome is read from stdout. Exit code 2 blocks, with stderr as the reason. Event names reuse Pi's: `session_start`, `session_shutdown`, `input`, `before_agent_start`, `tool_call`, `tool_result`, `session_before_compact`, `session_compact`, `turn_end`, `agent_settled`, `model_select`. |
| Go hooks | 1.0 | The same events as a `hook.Hook` passed to `gobble.New`, which gives embedders Pi's extension power in Go. `before_provider_request` is Go-only. |
| Packages | 1.x | `gobble pkg install|remove|list|update`. Sources are `git:<url>[@ref]`, a local path, or `https://…tar.gz#sha256=…`. A package bundles skills, prompts, agents, themes, hooks, an `mcp.json` fragment, and a `models.json` fragment, described by `gobble-package.json`. Pi packages with a `package.json` `pi` manifest load their skills, prompts, and themes; their TypeScript `extensions` are listed and skipped, with a warning. `packages.lock.json` pins the commit and sha256. Project packages need trust. This decides report D7. |
| OCI package source | exp | `oci://` through an ORAS-style client. |
| WASM tool plugins | exp | `exp/wasmtool`: WASI modules as tools under wazero, with no host filesystem unless granted. |
| Codemode | exp | `exp/codemode`: Pi's contract (`tools.*`, `text()`, `store/load`, `// @options:`) on `quickjs-wasi` under wazero, which is pure Go with CGO off. This decides report D3. |
| External ACP agents as subagents | exp | An `agents/*.md` entry with `acp: {command: […]}` makes `task` run another ACP agent binary as the child. gobble is then an ACP client too. |
| Virtual models and routing | exp | A setting-driven router maps a (model, thinking) pair to a physical model per request (`docs/virtual-models.md`). |

#### Observability and operations (`telemetry`, `internal/cli`)

| Capability | Tier | Behaviour |
|---|---|---|
| Structured logs | 1.0 | `slog` rolling files in the state directory, with redaction. `--log-level`. |
| Usage ledger | 1.0 | Every provider call is appended to `data/usage/*.jsonl` with provider, model, tokens, cost, and session. `/usage` shows the session and today. `gobble usage [--since] [--by model|provider|day]`. |
| OpenTelemetry | 1.x | GenAI semantic-convention spans (`invoke_agent`, `chat`, `execute_tool`) and token and duration metrics, exported over OTLP from the standard `OTEL_*` variables. Off by default. |
| Flight recorder | 1.x | `runtime/trace.FlightRecorder` ring. `gobble debug trace` and slow-turn automatic dumps. |
| `gobble doctor` | 1.0 | Checks directories, settings against the schema, provider credentials (a models list call), MCP start-up, `git` and shell presence, bridge sources, and terminal capabilities. `--json` output. |
| `/bug` | 1.x | A redacted diagnostic bundle (version, settings with secrets removed, the last N log lines, the session file on request). |
| No phone-home | 1.0 | gobble has no install telemetry, analytics, device id, or tracking id. Pi's `enableInstallTelemetry`, `enableAnalytics`, `trackingId`, and `deviceId` are dropped on `config import`. |

#### Distribution (report D9)

| Capability | Tier | Behaviour |
|---|---|---|
| GitHub releases | 1.0 | Six **raw binaries** named `gobble-<goos>-<goarch>[.exe]`, `SHA256SUMS`, SPDX SBOM extra asset, and provenance attestations, published by go-core-lib's reusable workflow (0004-PLAN Phase 4). Strict `vMAJOR.MINOR.PATCH` tags only. |
| `go install` | 1.0 | `go install github.com/maccavelli/gobble-cli/cmd/gobble@latest`. |
| Self-update | 1.0 | `gobble update [--check]` through `github.com/maccavelli/go-core-lib/selfupdate` `v1.1.0`. Product `gobble`. `Request.CheckOnly` for `--check`. `NewStandaloneInstaller`, `NewExactAssetSelector` for the six platforms, `NewGitHubSource` with the 0003-MADR User-Agent, `NewTextReporter` on stderr, `NewTerminalConfirmer`, `ExitCode` mapped to process status (0 current/declined/applied, 10 update available, 1 error). `--force` is `Request.Force`; `--yes` is `Request.Yes`. |
| Homebrew tap, container image | 1.x | A tap formula, and a `ko`-built distroless image for CI agents. |
| magic-cli-remote `KnownGoodVersion` | 1.0 | Published in the release notes for the companion Spec. |

### CLI surface

```
gobble [prompt…]                      TUI on a TTY; print mode with -p or piped stdin
gobble acp                            ACP agent on stdio
gobble rpc                            Pi JSONL RPC shim (1.x)
gobble session list|show|export|import|fork|rm|prune
gobble mcp add|remove|list|get|login|logout|serve
gobble auth login|logout|status|token|import
gobble models list|info|refresh
gobble config get|set|edit|path|validate|schema|import
gobble skills list|show|new
gobble prompts list
gobble agents list|show
gobble pkg install|remove|list|update          (1.x)
gobble tools list
gobble trust | untrust
gobble usage
gobble doctor
gobble update
gobble version
gobble debug trace                              (1.x)
gobble completion bash|zsh|fish|powershell
```

Global flags: `--model`, `--thinking`, `--mode`, `--cwd`, `--continue`,
`--resume <id>`, `--fork <id>`, `--tools`, `--no-context-files`,
`--no-skills`, `--no-pi-bridge`, `--approve`/`--no-approve` (project
trust), `--yes`, `--output-format`, `--api-key`, `--log-level`.

### Slash commands over ACP

Advertised in `available_commands_update` only once the handler lands.
The rule from 0002 still applies: advertisement is a subset of the
handlers, and a test enforces it.

| Command | Tier |
|---|---|
| `/help`, `/compact [instr]`, `/usage`, `/context`, `/session`, `/status`, `/model [id]`, `/thinking [level]`, `/mode [id]`, `/plan`, `/name [name]`, `/fork [entry]`, `/clone`, `/todos`, `/mcp [status\|reconnect]`, `/permissions`, `/trust`, `/reload`, `/skill:<name>`, template commands | 1.0 |
| `/tree [id]`, `/label`, `/undo`, `/redo`, `/diff [turn]`, `/ps`, `/stop [job]`, `/review`, `/init`, `/agents`, `/export [path]`, `/share`, `/archive`, `/delete`, `/mcp:<server>:<prompt>` | 1.x |
| TUI only, never advertised: `/settings`, `/hotkeys`, `/quit`, `/copy`, `/new`, `/resume`, `/login`, `/logout`, `/theme`, `/changelog`, `/editor` | 1.0 (TUI) |

### Companion `command.Table` target (magic-cli-remote `IDPi`)

This table supersedes the table in 0002-MADR's More Information as the
**target**. A row may be registered in magic-cli-remote only in the gobble
release whose `available_commands_update` contains the command. Until then
the 0002 row applies.

| Canonical | Kind | Mechanism | Tier |
|---|---|---|---|
| `help` | KindDaemon | daemon | — |
| `plan` | KindMode `plan` | `session/set_mode` | 1.0 |
| `mode` | KindMode | `session/set_mode` | 1.0 |
| `permissions` | KindNative | `/permissions` | 1.0 |
| `reviewer`, `approve` | KindNone | no separate reviewer or guardian | — |
| `model` | KindNative | `/model id`; becomes KindOp once the Spec routes `OpSetModel` through `session/set_config_option` | 1.0 |
| `thinking` | KindNative | `/thinking level`; becomes KindOp on the same condition | 1.0 |
| `context` | KindOp OpContext | last `usage_update` | 1.0 |
| `status` | KindNative | `/status` | 1.0 |
| `usage` | KindNative | `/usage` | 1.0 |
| `compact` | KindNative, then KindOp once the Compact hook exists | `/compact`, `_gobble/compact` | 1.0 |
| `clear`, `new`, `sessions` | KindDaemon | daemon | — |
| `goal`, `workflow`, `loop`, `fast`, `personality` | KindNone | no v1 equivalent | — |
| `deep-research` | KindNative | built-in skill through `/skill:deep-research` | 1.x |
| `review` | KindNative | `/review` | 1.x |
| `fork` | KindNative | `/fork` | 1.0 |
| `archive`, `delete` | KindNative | `/archive`, `/delete` | 1.x |
| `ps`, `stop` | KindNative | background jobs | 1.x |
| `diff`, `undo`, `redo` | KindNative | checkpoints | 1.x |

At full 1.x, 19 of the 30 canonical names execute natively or through a
mode or op. The other 11 are daemon-owned (4) or honestly `KindNone` (7).

### Out of the v1 line

TypeScript/jiti extensions and any JavaScript extension host; Pi's
TypeScript SDK and browser `pi-ai`; Chord, `pi-durable`, the Pico harness,
`pi-protocol` CBOR, `pi-server`, and `pi-client` (report D12); the
mcremote/mcrelay WebSocket; `_x.ai/*`; installing npm packages; the
Radius share gateway; install telemetry and analytics; the Codex
WebSocket transport; image-generation and classifier model types; Pi's
RPC extension-UI subprotocol; the special-purpose `/llama` command (the
compatible provider covers llama.cpp servers).

### Consequences

* Good, because nearly every Pi capability is in the v1 line. So are the
  ones Pi left to examples (subagents, todo, plan mode, checkpoints,
  permission gating), which become first-class and tested.
* Good, because magic-cli-remote gains 11 more canonical commands that
  execute natively (`/permissions`, `/status`, `/review`, `/archive`,
  `/delete`, `/ps`, `/stop`, `/diff`, `/undo`, `/redo`, `/deep-research`)
  compared with 0002's table, where 8 of the 30 names executed in the
  agent.
* Good, because every extensibility path is an open standard (MCP, Agent
  Skills, `AGENTS.md`, exec hooks over JSON on stdio, ACP) or a Go
  interface. Pi users' data assets carry over. Their TypeScript does not,
  and gobble says so.
* Good, because the tiers make `v1.0.0` a finite checklist, while 1.x
  items need no new decision to land.
* Neutral, because the TUI returns to v1 (0002 had excluded it). The
  import-boundary rules in 0004 keep Charm out of the agent core, and the
  TUI remains an ACP client.
* Neutral, because gobble adds permission prompts that Pi never had. Pi-like
  behaviour is one setting away (`bypass`). Remote users get approvals on
  the phone through ACP.
* Bad, because the scope is large. The 1.0 gate alone includes a TUI,
  four provider adapters, MCP with OAuth, and full session v3. Calendar
  time is long even with the tiers.
* Bad, because Pi parity details (the fuzzy edit, template quoting,
  compaction cut points) need golden tests against Pi's behaviour. Those
  tests have not been written, and some edge cases will differ until they
  are.

### Confirmation

* Each 1.0 row has at least one test named in
  [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md). The
  release-gate phase lists them all and records their results.
* The advertisement-is-a-subset-of-handlers test (0002 Phase 6) runs
  against the final 1.0 command list.
* A Pi-fixture suite runs gobble against Pi's documented behaviour:
  * session v3 samples from `docs/session-format.md` round-trip;
  * template substitution cases from `docs/prompt-templates.md`;
  * edit uniqueness, overlap, and fuzzy cases;
  * truncation boundaries at 2000 lines, 50 KB, and 500 characters.
* A Pi-user smoke test: with a fake `~/.pi/agent` holding one skill, one
  template, one MCP server, and one session, `gobble` lists the skill, runs
  the template, connects the server, and imports the session. The
  `~/.pi` tree's hash is unchanged.
* `go list ./exp/...` packages are unreachable from stable and beta
  packages (0004 archtest rule 7).

## Pros and Cons of the Options

### Tiered v1 line: a `1.0` gate, a `1.x` train, `exp/`, and an explicit out list

* Good, because it satisfies "as much as possible" without an unshippable
  gate.
* Good, because each tier has a concrete meaning a reviewer can check.
* Bad, because 1.x items can linger. Mitigation: each has a PLAN phase and
  a status.

### Keep 0002's narrow v1; everything else is v2

* Good, because it gives the fastest `v1.0.0`.
* Bad, because it contradicts the owner's instruction.
* Bad, because v2 of a Go module means a new import path. Pushing features
  past v1 costs embedders an import migration, whereas minor releases
  inside v1 cost nothing.

### Everything in scope must pass the `v1.0.0` gate (no tiers)

* Good, because it is simple to state.
* Bad, because one slow item (codemode, WASM plugins, OAuth for a
  particular provider) would block every other feature's release.

### Full Pi parity including TypeScript extensions through an embedded JS runtime

Use goja or QuickJS under wazero to host jiti-style extensions against a
re-created `ExtensionAPI`.

* Good, because Pi's extension gallery would run.
* Bad, because the extension API imports `pi-tui`, `pi-ai`, and
  `typebox`, and receives Node's full permissions. Re-creating that
  surface in a JS host inside Go is the "JS host written in Go" that
  0001-REPORT F1 warns against.
* Bad, because it would take the calendar the tiers above spend on
  features.

## More Information

* Decides report D1 (the v1 line is the shipping `gobble` a user runs, plus
  a Go SDK), D2, D3 (exp), D5, D6, D7, D8, D9 (distribution), D12 (out),
  and D13. D4, D10, D14, D15, and D16 are in 0004-MADR. D11, D17, and D18
  are in 0003-MADR.
* Amends [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md):
  its "What v1 is not" list and its companion table are superseded by this
  record. Its ACP-as-command-API decision stands.
* Implemented by [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md),
  after 0004-PLAN and 0002-PLAN.

### Amendment (2026-09-30): shared libraries as they exist

Still `proposed`. Follows 0004-MADR's amendment of this date.

* The go-llmprovider-sdk adapter moves from 1.x to **1.0** and is the
  only 1.0 provider implementation. Official vendor LLM SDKs are out of
  1.0. The unnamed OpenAI-compatible long tail, and Bedrock / Vertex /
  Azure, stay **1.x** because the SDK does not speak them today.
* Grok (xAI) is 1.0 through the SDK's `grok` id, not through a gobble
  compat preset.
* ChatGPT/Codex subscription auth is 1.0 through the SDK `openai`
  backend; other subscription OAuth stays 1.x.
* `gobble update` uses `go-core-lib/selfupdate` `v1.1.0`, not
  `mcplib/selfupdate`. Release assets are raw binaries, not archives.
* Pi sources (at `312184edb`, under `packages/`):
  * tools: `coding-agent/src/core/tools/*.ts`, `truncate.ts`,
    `file-mutation-queue.ts`, `utils/image-resize-core.ts`;
  * agent loop: `agent/src/agent.ts`, `agent/src/types.ts`;
  * compaction and retry: `coding-agent/docs/compaction.md`,
    `ai/src/utils/retry.ts`;
  * sessions: `coding-agent/docs/session-format.md`;
  * settings: `coding-agent/src/core/settings-manager.ts`;
  * context and skills: `resource-loader.ts`, `core/system-prompt.ts`,
    `docs/skills.md`, `docs/prompt-templates.md`;
  * MCP: `mcp/src/`, `coding-agent/docs/mcp.md`;
  * providers: `ai/src/types.ts`, `docs/models.md`;
  * extensions: `core/extensions/types.ts`, `docs/extensions.md`;
  * RPC: `docs/rpc.md`, `docs/rpc-commands.md`;
  * example extensions: `coding-agent/examples/extensions/`;
  * cache warming: `core/cache-warmer.ts`;
  * trust: `core/project-trust.ts`, `docs/security.md`.

### Amendment (2026-10-01): native agent of magic-cli-remote

Still `proposed`. Follows the fourth amendment of
[0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md).

The capability inventory and slash-command tiers in this record stand.
The companion `command.Table` above remains the **target**. How an
`IDPi` Spec reaches those rows is now specified in 0002: `KindNative`
until `acpagent.Spec` parameterizes Compact / Fork / Rename / Usage /
SetModel / SetThinking / Undo; then `KindOp` against `_gobble/…` and
`session/set_config_option`. Grok vendor methods stay grok's.

Pi HEAD is still `312184edb` and still has no ACP. Plan mode,
permissions, todo, subagents, and checkpoints remain first-class Go
work in this inventory (Pi ships them only as example extensions).
Retry classification stays typed errors. Background jobs (`/ps`,
`/stop`) remain a 1.x addition; Pi bash has no `background` field.

Subagent progress stays `tool_call_update` on the parent session:
acpagent drops child-session ids on the same ACP connection
(magic-cli-remote MADR 0051 D6).

### Amendment (2026-10-01, second): cross-repository assessment against code

Still `proposed`. Evidence was read in code:

* Pi at `312184edb` (paths under `packages/`);
* go-llmprovider-sdk at `940fee0`;
* go-core-lib at `v1.2.0`;
* magic-cli-remote at `772c041e`.

The tier model and the honest-advertisement rule are unchanged. Rows below either correct a fact or change a row's tier or behaviour. Each change is marked **(changed)**.

#### Pi facts corrected

| Row | Was | Pi's code says |
|---|---|---|
| Retry | "Honours `Retry-After`" as Pi behaviour | Pi's agent-level retry (`coding-agent/src/core/agent-session.ts:3664-3700`) is pure exponential back-off. Only `ai/src/utils/provider-retry.ts:23-63` reads `retry-after`, and it fails fast over 60 s. gobble honours `Retry-After` by design, not for parity. The schedule (3 retries, 2 s base, 60 s cap) is Pi's. |
| Auto-compaction | Summary sections "Goal, Constraints, Progress, Decisions, Next Steps, Critical Context" | Goal; Constraints & Preferences; Progress (Done / In Progress / Blocked); Key Decisions; Next Steps; Critical Context; then `<read-files>` and `<modified-files>` blocks. Branch summaries stop after Next Steps (`coding-agent/docs/compaction.md:236-273`). Tool results are cut to 2000 characters when serialised for the summary. |
| Context files | `AGENTS.override.md`, `AGENTS.md`, `CLAUDE.md` | Also `AGENTS.MD` and `CLAUDE.MD` (`core/resource-loader.ts:185`). |
| Sectioned system prompt | `preamble`, `tools`, `rules`, `project_context`, `skills`, `cwd`, custom | Also `docs` and `addendum`. `APPEND_SYSTEM.md` goes into `addendum`. Names match `^[a-z][a-z0-9_-]*$` (`core/system-prompt.ts:144-172`). |
| Prompt templates | the grammar list | Also `${@:-default}`. A template without `description` uses its first non-empty line. |
| Exposure modes | Pi values `codemode`, `codemode-deferred` | Pi's tool exposure set is `direct, model-only, codemode, deferred, hidden` (`core/extensions/types.ts:509`). Pi's **default MCP exposure is `codemode`** (`extensions/mcp/config.ts:109`), with a top-level `autoEnableCodemode`. **(changed)** gobble accepts `model-only` as `direct`. A bridged Pi server with no `exposure` key is `deferred` in gobble, and gobble then activates `tool_search` automatically (Pi warns instead). |
| `grep`, `find` | (Pi implementation not stated) | Pi spawns ripgrep (`rg --json --hidden`) and fd (`--glob --hidden --no-require-git`), auto-downloaded into `<agent>/bin` (`utils/tools-manager.ts:22-60`). gobble stays pure Go. Two differences are documented and tested: Pi's patterns are Rust-regex syntax and gobble's are RE2; and "skips `node_modules`" holds in Pi only through `.gitignore`. 0001-REPORT F12 ("application code plus `ignore`/`minimatch`") is wrong. |
| Default active tools | not stated | Pi activates `read, bash, edit, write`; grep, find and ls are off (`settings-manager.ts:213`; `cli/args.ts:455-457`), because they need the rg/fd download. **(changed)** gobble activates `read, bash, edit, write, grep, find, ls`, which are pure Go and need no download. `tool_search` is active when any tool is deferred. `defaultTools` patches apply as in Pi. |
| `task` frontmatter | `name, description, model, thinking, tools, mode` | Pi's example reads `name, description, tools, model`, has single / parallel / chain modes (at most 8 tasks, concurrency 4), and spawns `pi --mode json -p --no-session`. `thinking`, `mode` and the depth limit are gobble additions. |
| Provider server tools | presented beside Pi features | Pi's `Tool` type is function-only. This row is a gobble invention, and stays 1.x behind SDK support. |
| Context paragraph | "Its docs say so" (permission prompts, background jobs, undo) | Pi's docs state only the permission point (`coding-agent/README.md:42`, `docs/security.md:3`). The other two are true in code. |
| Model catalog | "Pi's generated catalog under MIT" (0005-PLAN F4.6) | `ai/src/providers/data/` is gitignored and absent at `312184edb`. It is generated from models.dev and OpenRouter. The snapshot's provenance is those sources, whose terms F4.6 must check. It is not Pi's MIT tree. |
| Subscription OAuth | (Pi flows not listed) | Pi has 9 flows (`ai/src/auth/oauth/load.ts:14-24`): Anthropic, OpenAI ChatGPT, OpenAI Codex, GitHub Copilot, OpenRouter, Kimi, Meta, xAI, Radius. Pi's Anthropic OAuth presents as Claude Code. That is the "terms" problem the row names. |
| Out list | "the Radius share gateway" | Radius is a model-routing **gateway provider** (the `pi-messages` API with OAuth). It also does share, `/bug` upload and an experimental session relay. All of it stays out. |
| TUI-only commands | includes `/theme`, `/editor` | Neither is a Pi command: Pi picks themes in `/settings` and opens the editor with Ctrl+G. They stay as gobble TUI commands. Pi's `/scoped-models` is added at **1.x (TUI)**. |
| Package count (0001-REPORT) | "fourteen packages plus a SQLite session backend" | 13 packages plus `session-backends/sqlite-node`. That backend serves harness format 4, which the shipping v3 `SessionManager` cannot use. |

Further 0001-REPORT errata are kept here rather than in the report, which records 2026-09-29's observation:

* F8's OAuth list: there is no Google OAuth.
* F8's wire-API list omits `azure-openai-responses`.
* F5's native clipboard: file paths are darwin-only; modifier state is darwin and win32 only.
* F7 "Node SEA": no build script was found; unverified.
* F12's built-in extensions are four: `llama.cpp`, `codemode`, `tool-search`, `mcp`.
* The `Settings` interface has no keybindings or MCP keys; those are separate files.

#### Sessions **(changed)**

* **Malformed lines are skipped with a diagnostic**, as in Pi (`session-manager.ts:355-370`). They do not fail the load.
  * A file whose header is unreadable fails closed with an ACP error.
  * gobble never rewrites a file on load.
  * 0005-PLAN F2's "corrupt line fails closed" accept is replaced.
* **The leaf is the last entry in file order**, as in Pi. "Leaf pointer" in F2 means the in-memory index. Moving the leaf becomes durable when the next entry is appended with that `parentId`.
* **Session files are created lazily**, at the first user or assistant message, as in Pi. `session/new` returns an id at once. `session/list` lists only sessions with a file, plus the live ones in this process.
* **Golden fixtures come from real Pi output.** `docs/session-format.md`'s samples are not valid JSON (`"usage":{...}`; ids such as `d4e5f6g7` are not hex).

#### Providers and models (from go-llmprovider-sdk `940fee0`) **(changed)**

* **The named SDK providers are all 1.0 now.** `opencode-zen`, `opencode-go`, `huggingface`, `kilo`, `together` and `ollama` implement `llmprovider.Provider`. "On 2026-09-30 those five still use the old `Generate*` API" was that day's fact. It was six ids in five packages.
* **Subscription OAuth.**
  * ChatGPT/Codex stays 1.0.
  * xAI Grok OAuth (browser and device) and Kilo device login exist in the SDK. They are **1.0 candidates**, each landing with a recorded terms check.
  * Anthropic Max, Gemini and GitHub Copilot subscription logins are **out of the v1 line**. The SDK declines them by policy (its 0016-MADR D10), and gobble adds no vendor auth code of its own.
* **Thinking levels.** The SDK accepts efforts `low…xhigh` only. gobble's `minimal` and `max` map as 0004-MADR's map states, and `/thinking` reports the degradation.
* **Images.**
  * `read` returns image content only when the model accepts images **and** the SDK can send them.
  * Today the SDK cannot, so `read` always takes the text-note path.
  * ACP `promptCapabilities.image` is `false` until SDK image input lands.
* **Usage and cost are estimated until the SDK decodes usage** (its S9). The usage ledger, `/usage`, `usage_update`, the footer and auto-compaction use the estimate, labelled as such.
* **Prompt-cache warming stays 1.x, and is blocked.**
  * The SDK sends no cache hints, and has one cached-token counter with no read/write split.
  * The row lands only once the SDK supports Claude `cache_control` and the split.
  * Pi's modes `off | streaming | idle` (default `streaming`, global-only setting) are the target behaviour.
* **`gobble doctor` and `gobble models list` never trigger billed model probes** (`WithModelProbes(false)`).

#### Self-update and distribution (from go-core-lib `v1.2.0`) **(changed)**

* **Surface.**
  * The command is `gobble update [--check] [--yes|-y] [--force] [--dry-run] [--json] [--version vX.Y.Z]`, which is the surface go-core-lib's 0004-MADR plans for `selfupdate/cli`.
  * These flags are local to `update`. The global `--yes` (permissions) never reaches `Request.Yes`.
  * `--json` writes `Result.Document()` to stdout; reporter text goes to stderr.
* **Paths through the library.**
  * `--check` uses `Checker.Check`, so it resolves no target and takes no lock.
  * Apply uses `RunWith`, with `NewVersionProber` on `gobble version --json` for the staged and post-install probes, and `NewImageVerifier`.
  * The TUI uses `Start` / `Stream`.
* **Pin.** The newest `v1.x` at F10, not "v1.1.0 current".
* **Install locations.**
  * selfupdate replaces only a regular, non-symlinked file under the home directory or an explicit allowed root, and it needs write access to that directory.
  * A Homebrew-managed binary (a symlink) is detected, and `gobble update` prints `brew upgrade gobble` instead.
  * On Windows, a known go-core-lib defect makes later updates fail with "Access is denied" while an old `gobble.exe` (for example a long-lived `gobble acp`) still runs (go-core-lib `0004-PLAN-h4-running-copy-end-to-end.md`). `gobble update` explains this, and `CleanupPending` at start-up treats it as benign.
* **No phone-home is kept.**
  * Update checks happen only on explicit `gobble update` or `gobble doctor`.
  * `gobble acp`, print mode and the TUI never check on their own.
  * `GH_TOKEN` / `GITHUB_TOKEN` are used when set, and a stale token fails with 401. Both behaviours are documented.
* **Agent self-modification.**
  * The default permission rule asks for `bash(gobble update*)`.
  * `gobble update` refuses to apply when `AI_AGENT=gobble` is set, that is, when it runs inside gobble's own tool. `--check` is still allowed.
* **Release assets.**
  * The SPDX SBOM is generated by gobble's workflow. It is not in `SHA256SUMS`.
  * Release candidates are not tagged until go-core-lib `v1.3.0` channels are adopted.

#### Driven by mcremote (magic-cli-remote `0179-MADR-gobble-native-acp-provider.md`) **(changed)**

| Capability | Tier | Behaviour |
|---|---|---|
| Commands before the first prompt | 1.0 | `session/new` and `session/load` send `available_commands_update`, `current_mode_update` and a first `usage_update` before they return (0002 fifth amendment). |
| Prompt while busy | 1.0 | A `session/prompt` that arrives while a turn runs is queued as a follow-up (Pi's default `one-at-a-time`). With `_meta.gobble.streamingBehavior = "steer"` it is a steer. Each request is answered when its own turn ends, after retries, overflow compaction and the follow-ups it queued, which is Pi's `agent_settled`, not `agent_end`. |
| Ask the user | 1.x | An `ask_user{question, options[]}` tool over `session/request_permission` with custom options, which is the only question path mcremote relays. Free-text answers need ACP elicitation (unstable): **exp**. Pi has this only as an example (`examples/extensions/question.ts`). |
| Plan visible on the phone | 1.0 | Leaving plan mode also publishes the plan as an ACP `plan` update. mcremote cuts permission-card text to 400 runes until 0179 D5. |
| Status notifications | 1.x | `_gobble/status{kind: retry\|compaction\|cache_warm, …}`, modelled on Pi's `auto_retry_*` and `compaction_*` events. mcremote shows them once its Spec subscribes. |
| Readiness probe | 1.0 | `gobble auth check [--json]`, exit 0 (ready) / 1 (no credential) / 2 (error), as Pi does. `gobble doctor --json` is already 1.0. |
| Quiet, offline start | 1.0 | `gobble acp` makes no network call before `session/new`. `--offline` / `GOBBLE_OFFLINE=1` also disables catalog refresh and `web_fetch`. |
| Client terminal default | 1.0 | `tools.bash.useClientTerminal` defaults to `false`, because the phone shows only `"terminal <id>"`. |
| `/deep-research` alias | 1.x | Advertised with the skill. mcremote's command-name rule rejects `/skill:…`. |
| `bypass` on the phone | 1.0 | `permissions.allowBypass` defaults to `false`. mcremote marks `bypass` dangerous only after 0179 D7. |

The companion table in this record stays the **target**. magic-cli-remote 0179 D9 is the form it takes there. The `KindNative` rows for `fork`, `undo`, `redo`, `diff`, `compact` and `thinking` are safe only once 0179 D3 (strict fallback) has landed. Until then the default-fallback hazard (0002 fifth amendment) applies. The count "19 of 30 at full 1.x" holds under 0179.

#### CLI flags **(changed)**

* **Flags at 1.0, matching Pi's meaning:**
  * `--provider`, `--system-prompt`, `--append-system-prompt`;
  * `--no-session`, `--session <path|id>`, `--session-dir`;
  * `--no-tools`, `--exclude-tools`, `--offline`;
  * `@file` arguments.
* **Flags at 1.x:**
  * `--models` (scoped models), `--skill`, `--prompt-template`, `--no-prompt-templates`;
  * `-n/--name`, `--session-id <id>` (a caller-chosen id, which suits a daemon), `--verbose`.
* `--resume` with no id opens the session picker in the TUI.
* `--list-models` is an alias of `gobble models list`.
* **`--mode` stays gobble's ACP mode id.** Pi's `--mode text|json|rpc` maps to `--output-format text`, `--output-format stream-json` and `gobble rpc`. Pi's `--mode json` event vocabulary (`docs/json.md`) is **out**, because gobble's stream is ACP's.
* **Print mode is selected when a prompt is given and stdin or stdout is not a terminal**, as in Pi (`docs/cli.md:30`). The exit status is 1 on error or abort, 129 on SIGHUP, and 143 on SIGTERM (`modes/print-mode.ts:60,147`).

#### Bridge

The Pi bridge's additions (relocated installs, packages, JSONC `models.json`, tolerant session reads) are in 0003-MADR's amendment of this date.

**Skill discovery order (changed)**, aligned with Pi's precedence, which is project before user:

1. `.gobble/skills`;
2. project `.agents/skills`, from `cwd` up to the repo root (trusted);
3. config `skills/`;
4. `~/.agents/skills`;
5. the Pi bridge.

Under first-wins, a name collision now resolves the way Pi resolves it (`core/package-manager.ts:2477-2560`).

### Amendment (2026-10-02): provider, update and TUI delivery against sibling source

Still `proposed`. This follows the 2026-10-02 amendment of
[0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md).
Its source pins and paths are evidence for today's inventory, not dependency
pins in gobble (which still has no `go.mod`). Earlier dated observations and
the tier choices remain historical where this amendment supersedes a fact.

* **F4, usage and errors.** `go-llmprovider-sdk` at `67fc56e` has completed
  the S8 API removals, S8c auth extraction and S9 usage decoding in source.
  `llm/provider` reads `Response.Usage` (including cache reads), treating
  all-zero usage as unavailable and estimating that turn. A nonzero SDK
  usage is reported as given, including any zero components. The usage
  ledger, `usage_update`, `/usage`, footer, cost and auto-compaction
  distinguish reported and estimated
  values. Error mapping uses `*APIError` and the sentinels; the special
  `*RateLimitError` and `*IncompleteError` compatibility cases are retired.
  `llmprovider/auth` supplies OAuth and token-store contracts. This does
  not unblock image input, cache-write accounting, prompt-cache warming or
  native token streaming; the SDK still lacks those capabilities and a tag.
* **F4, configuration.** The SDK now makes environment reads opt-in. gobble
  resolves credentials (including `ANTHROPIC_API_KEY`) and catalog settings
  itself, then passes options explicitly. It does not call
  `ModelProbesFromEnv` or `catalog.OptionsFromEnv` by default. Billed model
  probes stay disabled through `WithModelProbes(false)`.
* **F10, self-update.** `go-core-lib v1.3.0` adds prerelease channels, but
  gobble's 1.0 release and `gobble update` still select stable tags only. The
  current usable library surface is `selfupdate` plus its test helper;
  `selfupdate/cli`, `buildinfo` and `go-tui-lib/updatetea` are plans, not
  imports. The F10 implementation re-resolves the latest suitable `v1.x`
  tag. The `--check` path uses `Checker.Check`; apply uses `RunWith`, and a
  TUI may drive `Start` / `Stream`. Adding a `--channel` option or publishing
  gobble prereleases is deferred to a later explicit decision.
* **F9, TUI.** The existing `go-tui-lib v0.1.0` packages cover reusable
  workspace layout, pane hosting, glyphs, themes and golden rendering.
  F9 uses a corrected tagged version of those packages after its known
  `v0.1.0` defects are checked. Gobble owns its ACP transcript, tool cards,
  permission dialog, editor and session controls. Proposed sibling
  `stream`, `command`, `keymap`, `palette` and terminal-service plans do
  not satisfy any F9 acceptance criterion until code and a tested tag
  exist. F9 remains a 1.0 feature, but its go-tui-lib dependency and its
  local components are stated separately in its plan.

### Source correction (2026-10-02): go-core-lib after v1.3.0

The F10 inventory above describes `v1.3.0`. `v1.3.1` is now the latest
released tag observed (`351bd6a5ffbd4b352cacc5234f4a8c504c3c1b69`);
it still exports `selfupdate` and `selfupdate/selfupdatetest`. A `buildinfo`
package exists at later development commit `bf7221a5408984299a8511eb98f8f06eadd133bd`,
after that tag. F10 does not presume the development package can be imported.
It rechecks the released surface and the local `internal/buildinfo` boundary
when selecting a tag. The stable-only gobble release decision is unchanged.

The provider SDK's observed `efd9c61` commit adds catalog and setup-wizard
work but leaves the F4-facing `Provider`, `Response`, `Usage`, `APIError` and
auth contracts unchanged from the `67fc56e` inventory above. It still has
no tag, so the F4 release gate is unchanged.

### Source correction (2026-10-03): go-core-lib is now go-selfupdate-lib

* **The rename.** go-core-lib was renamed go-selfupdate-lib on 2026-10-02:
  both the repository and the module path (go-selfupdate-lib
  `docs/decisions/0009-MADR-rename-to-go-selfupdate-lib.md`).
  * `github.com/maccavelli/go-core-lib` ends at `v1.4.1` (`58411f1`), which
    `go` reports as deprecated. Its tags, `v1.1.0` included, still resolve
    when pinned. `go get github.com/maccavelli/go-core-lib@latest` now
    fails with a path mismatch.
  * `github.com/maccavelli/go-selfupdate-lib` starts at `v1.5.0`
    (`6deaa524cfb28aad90bea97a6d9162e5b4257204`), with `v1.4.1`'s API.
    The older tags are not versions of the new path.
  * The reusable release workflow is
    `maccavelli/go-selfupdate-lib/.github/workflows/publish-selfupdate-release.yml`.
* **The released surface has grown since `v1.3.1`.** `v1.4.0` (`4d7b053`)
  added `buildinfo` and `selfupdate/cli`. `v1.5.0` exports both, with
  `selfupdate` and `selfupdate/selfupdatetest`. `buildinfo` is therefore
  released, no longer development-only as the 2026-10-02 correction
  observed. Its `-X` symbols are
  `github.com/maccavelli/go-selfupdate-lib/buildinfo.version` and `….kind`.
* **What it means for F10.** `gobble update` binds
  `github.com/maccavelli/go-selfupdate-lib/selfupdate`, at `v1.5.0` or
  later, in place of `go-core-lib/selfupdate`. The update surface this
  record plans is the one go-core-lib's 0004-MADR planned for
  `selfupdate/cli`, and that package is now released. F10 checks the
  released `selfupdate/cli` and `buildinfo` against this record's surface
  and gobble's `internal/buildinfo` before it binds either. This correction
  does not choose.
* **What does not change.** The stable-only release decision, and the rest
  of F10.
* **The earlier text stays as written**, as dated history.

## Amendment — 2026-10-04: the CLI is Kong, not Cobra or fang

The owner directed that gobble's CLI is Kong, not Cobra or fang. This supersedes the Cobra/fang CLI choice in this record. The sentences above stay as written.

## Amendment — 2026-10-04: go-tui-lib, go-selfupdate-lib, and canonical code

Still `proposed`. The owner directed the rules below. The sentences above stay as written. The Kong amendment of this date is unchanged.

* **TUI.** The TUI uses `go-tui-lib` for all core TUI functionality. gobble does not reimplement core TUI. This supersedes the Charm TUI row's "extracting shared widgets into go-tui-lib as they settle" and the 2026-10-02 F9 note that gobble owns the transcript, tool cards, permission dialog, editor, and session controls as local core TUI. Those sentences stay as dated history. Where `go-tui-lib` does not yet export a core behaviour, gobble waits for the library rather than reimplementing it. `internal/tui` remains an ACP client.
* **Self-update.** Self-update uses `go-selfupdate-lib`. gobble does not reimplement self-update. The 2026-10-03 F10 binding of `github.com/maccavelli/go-selfupdate-lib/selfupdate` already chose the library; it did not say that gobble must not reimplement the updater.
* **Canonical code.** As much code as possible is canonical: use the sibling libraries and the standard library rather than local copies. This does not change the meaning of "canonical" in the companion command table.
* **Already recorded.** Code stays Go 1.27.1, idiomatic, and modular under 0004-MADR. This record does not change that. The CLI stays Kong.
