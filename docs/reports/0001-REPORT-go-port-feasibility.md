---
date: 2026-09-29
subject: technical feasibility of porting the Pi TypeScript monorepo completely to Go
examines: "github.com/earendil-works/pi at 312184edb (clone origin github.com/maccavelli/pi, branch main, clean tree); sibling Go libraries go-llmprovider-sdk, go-tui-lib, mcplib, ocp-login"
---
# Go Port Feasibility of the Pi Agent Harness

This report records findings. It decides nothing. Each finding names the
decision it bears on; those decisions belong in a MADR in this repository
(`docs/decisions/`), and the work in its PLAN.

Stated constraints from the request that opened this repository: the rewrite
uses **Cobra** and **Viper** for the command line and configuration, and
**Charm** (Lip Gloss and the rest of that stack) for terminal UI.

## Summary

A **source-faithful 1:1 port of every package**, including the TypeScript SDK,
browser `pi-ai`, jiti-loaded TypeScript extensions, and the npm package
ecosystem, is **not achievable**. Those surfaces are language-shaped. Go cannot
host them without embedding a JavaScript runtime, at which point the result is
no longer a Go program with a Go extension API.

A **product rewrite in Go** of the thing a user runs as `pi` — interactive TUI,
print / JSON / RPC modes, agent loop, built-in tools, JSONL sessions, skills,
prompt templates, JSON themes, settings, MCP client, provider/auth layer, and
a Charm TUI — is **technically feasible**. It is a multi-year rewrite of roughly
200k lines of TypeScript source (plus 189k lines of tests), not a translation.
The TUI, provider catalog, and extension system have to be redesigned rather
than transcribed.

The evidence:

- The monorepo is fourteen packages plus a SQLite session backend. The
  coding-agent CLI is 86k lines of source by itself. (M)
- Extensions are TypeScript modules loaded in-process with `jiti`. Skills,
  prompt templates, and themes are Markdown and JSON. (R, F1, F2)
- Codemode executes model-written JavaScript in QuickJS compiled to WebAssembly.
  (R, F3)
- `@earendil-works/pi-tui` is a custom differential renderer with an
  emacs-style editor, OKLCH colors, Kitty/iTerm2 images, and N-API clipboard
  helpers. Charm is a different programming model. (R, F5)
- `pi-ai` ships 42 provider factories, a generated model catalog, OAuth, image
  generation, classifiers, and a browser-clean core. The in-house
  `go-llmprovider-sdk` is being extracted from `mcplib` with 9 provider ids
  and no Charm/Cobra surface. (M, R, F8, F13)
- Session files are versioned JSONL with a documented v3 tree. A Go binary can
  read and write that format. (R, F9)
- Chord, `pi-durable`, `pi-protocol` / `pi-client` / `pi-server`, and the
  Pico harness are a second, experimental product inside the same monorepo.
  (R, F10)

## Scope and method

The TypeScript tree was read at `312184edb` (clean `main`, 2026-09-29
19:04:43 +0200, message "Add [Unreleased] section for next cycle"). Origin
is `github.com/maccavelli/pi`; upstream is `github.com/earendil-works/pi`.
Package versions in that tree are `0.99.0`. Sibling Go repositories were
read at their working trees under the same parent directory.

Each finding carries an evidence level:

- **M:** measured by a command against `312184edb` (line counts, `git`,
  directory inventory).
- **R:** read in source or package documentation, not executed.

No TypeScript test, build, or live provider request was run. No Go scratch
module was compiled. Charm, Cobra, and Viper behaviour is taken from the
in-house `ocp-login` tree (Cobra 1.10.2, Viper 1.21.0, Charm v2
`charm.land/{bubbletea,bubbles,lipgloss}/v2`) and from the libraries'
public docs.

## What exists

### Size

Line counts are physical lines of `.ts` / `.js` / `.mjs` under each package
`src/` and `test/`, plus native C/Obj-C in `packages/tui/native`. Examples
are listed separately.

| Package | Source | Tests | Notes |
|---|---|---|---|
| `coding-agent` | 304 files, 86,536 lines | 321 files, 67,979 lines | CLI, TUI mode, tools, extensions, RPC, package manager |
| `ai` | 193 files, 26,042 lines | 166 files, 40,974 lines | 42 provider factories; generated catalog |
| `agent` | 117 files, 33,588 lines | 85 files, 29,399 lines | Agent loop plus `harness/` (Pico / durable runtime) |
| `tui` | 45 files, 19,158 lines | 49 files, 19,588 lines | plus 6 native files, 1,016 lines |
| `durable` | 42 files, 12,845 lines | 54 files, 13,900 lines | Pico storage (memory, JSONL, SQLite) |
| `chord` | 29 files, 8,808 lines | 30 files, 9,726 lines | Facets, services, replicated state, deltas |
| `mcp` | 18 files, 3,054 lines | 8 files, 1,320 lines | MCP client; no official SDK dependency |
| `sqlite-node` | 15 files, 1,973 lines | 6 files, 1,909 lines | `node:sqlite` session backend |
| `server` | 16 files, 1,966 lines | 7 files, 1,066 lines | Experimental routed server |
| `codemode` | 10 files, 1,579 lines | 3 files, 906 lines | QuickJS-on-WASM sandbox |
| `evals` | 5 files, 1,446 lines | 6 files, 815 lines | Docker eval harness |
| `client` | 8 files, 1,135 lines | 5 files, 808 lines | Experimental protocol client |
| `telemetry` | 6 files, 935 lines | 2 files, 243 lines | Vendor-neutral contracts |
| `protocol` | 8 files, 869 lines | 3 files, 567 lines | CBOR framed envelopes, protocol v8 |
| **Total** | **816 files, 199,934 lines** | **745 files, 189,200 lines** | |
| `coding-agent/examples` | 106 files, 16,150 lines | — | SDK, extensions, plugins |

The git tree lists 1,737 `*.ts` files and 1,997 paths under `packages/`. (M)

`coding-agent/src` breakdown: `core` 37,933; `modes` 22,169 (interactive TUI
is most of this); `experimental` 10,701; `extensions` 6,544; `utils` 3,658;
`cli` 1,939. (M)

### Package roles

| Package | Role | Public consumers |
|---|---|---|
| `pi-coding-agent` | The `pi` binary, TypeScript SDK, RPC, package manager | npm `@earendil-works/pi-coding-agent`; `pi` on PATH |
| `pi-agent-core` | Stateful agent, tool execution, event stream | coding-agent; published SDK |
| `pi-ai` | Unified provider API, catalogs, auth, streaming | agent, coding-agent; **browser** |
| `pi-tui` | Differential terminal UI | coding-agent interactive mode |
| `pi-mcp` | MCP client (stdio, Streamable HTTP, OAuth) | coding-agent MCP extension |
| `pi-codemode` | QuickJS WASM sandbox for model-written JS | coding-agent `codemode` tool |
| `chord` | Facet/service/replicated-state runtime | experimental coding-agent, durable |
| `pi-durable` | Durable conversation/task/document storage | Pico / harness |
| `pi-protocol` / `pi-client` / `pi-server` | Experimental CBOR protocol and local server | experimental CLI |
| `pi-telemetry` | Telemetry contracts | agent, coding-agent |
| `pi-session-backend-sqlite-node` | SQLite session backend | optional |

Root README also points at a separate `pi-chat` repository (Slack/chat),
outside this tree.

### CLI surface the Go binary would have to match

From `packages/coding-agent/docs/cli.md` and `src/cli/args.ts` (R):

```
pi [options] [--] [@files...] [messages...]
pi install | remove | uninstall | update | list
pi config
pi auth <check|print-api-key|print-bearer-token>
pi mcp <list|login|logout>
```

Modes: interactive TUI, `--print`, `--mode json`, `--mode rpc`, `--export`
(HTML). Flags cover provider, model, thinking, session continue/resume/fork,
tools, extensions, skills, themes, project trust, and **unknown flags**
reserved for extensions.

Cobra maps onto this surface. Unknown extension flags are the part that
needs a Cobra design (F14).

### Configuration the Viper layer would have to match

Settings are JSON, not YAML. Locations (R, `docs/configuration.md`,
`settings-manager.ts`):

- user: `~/.pi/agent/settings.json`
- project: `.pi/settings.json` (after project trust)
- env: `PI_*` overrides, plus `PI_AGENT_DIR`, `PI_SESSION_DIR`

The `Settings` interface is a large nested JSON object: model defaults,
compaction, retry, tools, packages, extensions, skills, themes, terminal,
images, keybindings, MCP servers, cache warming. Viper can read JSON and
bind env. Compatibility with existing `~/.pi` files is a product decision
(F15).

## Findings

### F1 — Extensions are TypeScript loaded in-process (R)

`packages/coding-agent/src/core/extensions/loader.ts` loads extension
modules with `jiti`. A factory receives `ExtensionAPI` and may register
tools, commands, shortcuts, CLI flags, providers, MCP servers, virtual
models, renderers, and event handlers. Extensions import
`@earendil-works/pi-coding-agent`, `pi-agent-core`, `pi-tui`, `pi-ai`, and
`typebox`. They run with the process's OS permissions.

The published package gallery and `pi install npm:…` / `git:…` distribute
those TypeScript packages. `docs/packages.md` states Pi supplies those
workspace packages to extension code at runtime.

A Go binary cannot load a `.ts` factory and honour that import graph.
Options that preserve *some* extension story:

* embed a JS runtime (goja, or wazero + the existing QuickJS WASM) and
  keep the TypeScript API — the process is then a JS host written in Go;
* replace the API with Go plugins (`plugin` build mode, cgo, poor Windows
  story) or a WASM/RPC extension host;
* drop third-party TypeScript extensions and keep only skills / prompts /
  themes, which are data.

**Bears on:** D1 (what "complete" means), D2 (extension host).

### F2 — Skills, prompts, and themes are data (R)

Skills are Markdown with YAML frontmatter (`skills.ts`). Prompt templates
are Markdown. Themes are JSON (`modes/interactive/theme/*.json` plus a
JSON schema). Discovery walks conventional directories and `package.json`
`pi` manifests.

These port without a language change. A Go `pi install` that only installs
data packages is feasible; one that installs TypeScript extensions is F1.

**Bears on:** D2, D7 (package manager scope).

### F3 — Codemode is a JavaScript VM, not a Go interpreter (R)

`@earendil-works/pi-codemode` compiles `quickjs-wasi/quickjs.wasm` and runs
model-written JavaScript. The model is documented as writing `tools.read(…)`,
`text()`, `store()` / `load()`, top-level `await`, and `// @options:` lines.
Nested tool calls stay out of the LLM context.

Go has no obligation to execute that JavaScript itself. A port that keeps
codemode must embed a JS engine (the same WASM module under wazero, or
another QuickJS binding). Interpreting the scripts as Go is a different
product: models generate JS today.

**Bears on:** D3 (codemode).

### F4 — The TypeScript SDK and browser `pi-ai` have no Go equivalent (R)

`docs/sdk.md`: `@earendil-works/pi-coding-agent` embeds Pi in a Node.js or
Bun process. That is a TypeScript import. A Go module can offer a Go SDK;
it cannot be a drop-in for `import { createAgentSession } from
"@earendil-works/pi-coding-agent"`.

`pi-ai` documents a side-effect-free browser core. OAuth and Bedrock
streaming are Node-only and lazy-loaded so browser bundles stay clean.
A Go rewrite does not replace a browser TypeScript library. Compiling Go
to WASM would still not preserve that import API.

RPC mode (`docs/rpc.md`) is the language-independent control surface:
JSONL on stdin/stdout. A Go `pi --mode rpc` can serve existing RPC
clients if the record contract is kept (F11).

**Bears on:** D1, D4 (SDK story), D5 (RPC compatibility).

### F5 — The TUI is a custom renderer; Charm is a rewrite, not a translation (R, M)

`pi-tui` (19k source lines, 20k test lines) implements:

* main-screen and alternate-screen renderers with differential updates and
  CSI 2026 synchronized output;
* a component tree (Editor, Markdown, Image, SelectList, SettingsList,
  MouseRegion, stacks, ScrollView, Loader, …);
* an emacs-style Editor: kill ring, undo, grapheme/word segmentation, paste
  markers, autocomplete;
* OKLCH / OKHSL color math;
* Kitty and iTerm2 inline images;
* N-API native modules for clipboard (text, image, file paths) and
  modifier-key state on Darwin, Win32, and Linux X11 (`native/`, 1,016
  lines of C/Obj-C).

Charm (as used in-house by `ocp-login`: `charm.land/bubbletea/v2`,
`bubbles/v2`, `lipgloss/v2`) is The Elm Architecture: `Model`, `Update`,
`View`. Lip Gloss styles strings. Bubbles supplies inputs, lists, and a
textarea that is not this Editor. Glamour renders Markdown; it does not
render LaTeX (`tui/src/latex.ts`) or mermaid (`coding-agent` mermaid
component). Clipboard in the Go fleet already appears as
`github.com/atotto/clipboard` (indirect via Charm). Native image clipboard
and file-path clipboard still need platform code.

Interactive mode in coding-agent is another 22k lines (`modes/`) of
session selectors, model/theme/login dialogs, diff rendering, footer,
trust UI, and transcript components on top of `pi-tui`.

A Charm TUI that *feels* like Pi is feasible. A line-by-line port of
`pi-tui` onto Lip Gloss is not: the runtime model is different. `go-tui-lib`
exists as an empty extraction target for Charm patterns and currently has
no packages.

**Bears on:** D6 (Charm v1 vs v2; how much Editor/Markdown/image fidelity).

### F6 — Image pipelines depend on Photon WASM (R)

`coding-agent/src/utils/photon.ts` loads `@silvia-odwyer/photon-node`
(`photon_rs_bg.wasm`) for resize, EXIF orientation, clipboard image
decode, and format conversion. The Bun compile copies that WASM next to
the binary.

Go replacements exist (`image`, `x/image`, `disintegration/imaging`,
`bimg`). Behaviour and quality need a comparison suite; none was run.

**Bears on:** D8 (image pipeline).

### F7 — Distribution gets simpler in Go; the npm package manager does not (R)

Today's install paths:

* `npm i -g --ignore-scripts @earendil-works/pi-coding-agent` (Node ≥ 22.19);
* `curl -fsSL https://pi.dev/install.sh | sh` (managed releases);
* `bun build --compile` standalone binary, with native `.node` prebuilds
  and Photon WASM;
* Node SEA;
* `pi update --self`.

A Go binary is one artifact per OS/arch. That removes Node, Bun, N-API
prebuilds, and shrinkwrap/install-lock machinery. Self-update from GitHub
releases is a solved pattern in this fleet (`prepare-commit-msg`,
`ocp-login`).

`pi install` / `remove` / `list` / `update --extensions` still talks to
npm and git to fetch **TypeScript** packages (F1). A Go `pi install` that
keeps that protocol either shells out to `npm` (settings already allow
`npmCommand`) or reimplements registry fetch, and still cannot execute
the installed JS without F1's host.

**Bears on:** D7, D9 (distribution).

### F8 — The provider layer is the second-largest library, and the Go SDK does not cover it (M, R)

`packages/ai/src/providers/all.ts` registers 42 factories:

amazon-bedrock, ant-ling, anthropic, azure-openai-responses, baseten,
cerebras, cloudflare-ai-gateway, cloudflare-workers-ai, deepseek,
fireworks, github-copilot, google, google-vertex, groq, huggingface,
kimi-coding, meta, minimax, minimax-cn, mistral, moonshotai,
moonshotai-cn, nvidia, openai, openai-codex, opencode, opencode-go,
openrouter, qwen-token-plan, qwen-token-plan-cn,
qwen-token-plan-individual, radius, together, typesafe,
vercel-ai-gateway, xai, xiaomi, xiaomi-token-plan-ams,
xiaomi-token-plan-cn, xiaomi-token-plan-sgp, zai, zai-coding-cn.

Wire APIs in `packages/ai/src/api/` include Anthropic Messages, OpenAI
Completions and Responses, Codex Responses (WebSocket), Google
Generative AI, Vertex, Bedrock Converse Stream, Mistral Conversations,
and a `pi-messages` gateway. Auth includes API keys, OAuth (OpenAI
ChatGPT, GitHub Copilot, Google, Anthropic CLI-style), and credential
stores. Image generation and classifiers are first-class catalogs.

`go-llmprovider-sdk` is being extracted from `mcplib` (`0001-REPORT` and
`0002-MADR` in that repository). At the time of that report it covered 9
ids (openai, gemini, claude, grok, opencode-zen, opencode-go,
huggingface, kilo, ollama), raw `net/http`, OAuth loopback/device-code,
and a configuration wizard. It has no image-generation or classifier
API. It is the natural Go home for providers; it is not a port of
`pi-ai`.

**Bears on:** D10 (provider ownership: this repo vs `go-llmprovider-sdk`).

### F9 — Session JSONL v3 is a documented, language-independent contract (R)

`docs/session-format.md`: JSONL, version 3, tree via `id` / `parentId`,
header line `type: session`, message / compaction / branch-summary /
custom entries. Version 1 and 2 migrate on load. Paths look like
`~/.pi/agent/sessions/--<cwd>--/<timestamp>_<id>.jsonl`.

A Go session store can read and write this format. Compaction and branch
summaries are agent behaviour on top of the tree. Appendix B of
`packages/agent/docs/harness.md` discusses coding-agent v3 compatibility
for the newer harness; that harness is not what the shipping CLI uses
today (F10).

Sharing `~/.pi` with TypeScript Pi is a compatibility decision, not a
feasibility block.

**Bears on:** D11 (session compatibility).

### F10 — Chord, durable, protocol, and Pico are a second product (R, M)

`chord` is documented as standalone: facets, services, replicated state,
delta tracking, Go-like context. 8.8k source lines.

`pi-durable` is the Pico storage layer (memory, JSONL, SQLite) with
conformance tests. 12.8k source lines.

`pi-protocol` v8 is CBOR length-prefixed envelopes; `pi-client` /
`pi-server` are experimental. Coding-agent `src/experimental/` is 10.7k
lines (mini, micro, coordinator, session workers, plugin services).

`packages/agent/src/harness/` contains the AgentHarness specification
implementation (Pico3, drive/recovery, JSONL session repo). The shipping
CLI path is `Agent` + `SessionManager` in coding-agent `core/`, not this
harness.

Porting "the monorepo" includes this stack. Porting "the `pi` a user
runs" can defer it. Chord's draft proxies and structural sharing are
JavaScript-object semantics; a Go port of Chord is a new design around
structs and codecs.

**Bears on:** D1, D12 (experimental stack in or out of v1).

### F11 — RPC, JSON, and print modes are the portable product APIs (R)

Print, JSONL events, and RPC are documented independently of TypeScript.
RPC is the integration path the docs recommend for "other languages,
isolated processes, IDEs". Keeping those record shapes lets existing
clients talk to a Go binary.

HTML export (`core/export-html/`) embeds `highlight.min.js` and
`marked.min.js`. A Go exporter can emit HTML without those scripts, or
keep them as assets. Fidelity is a product choice.

**Bears on:** D5.

### F12 — Built-in tools are ordinary OS work (R)

Built-in tools: `read`, `bash`, `powershell`, `edit`, `write`, `grep`,
`find`, `ls`, plus built-in extensions `codemode` and `tool_search`.
Path canonicalization, ignore files, diff/edit, and subprocess execution
are all standard-library-shaped in Go (`os/exec`, `path/filepath`,
`modernc.org/sqlite` or `database/sql` if a SQLite backend is wanted).

Grep currently uses application code plus `ignore` / `minimatch`. A Go
port can keep application grep or shell out to `rg`; that is a behaviour
decision. Project trust, sandboxing, and "runs with the user's OS
permissions" are policy, not a language issue. Containerization
(`docs/containerization.md`) is an extension (Gondolin) plus Docker /
OpenShell patterns; those remain valid around a Go binary.

**Bears on:** D13 (grep/find implementation).

### F13 — In-house Go libraries cover part of the stack, none of Pi (R)

| Repository | State | Overlap |
|---|---|---|
| `go-llmprovider-sdk` | docs + accepted migration MADR; extraction in progress | providers, OAuth, catalogs — subset of `pi-ai` |
| `go-tui-lib` | README only, no packages | intended Charm extraction target |
| `mcplib` | mature MCP **server** library | Pi's `pi-mcp` is a **client** |
| `ocp-login` | production Cobra + Viper + Charm v2 TUI | CLI/TUI patterns, not agent semantics |
| `magic-cli-remote` | Cobra + Viper, ACP, MCP; no Charm in `go.mod` | remote-agent CLI experience |
| `go-core-lib` | README stub | none yet |

Nothing in those repositories implements an agent loop, session tree,
coding tools, or Pi's interactive transcript. They reduce the cost of
providers, CLI, and TUI scaffolding. They do not make the port small.

**Bears on:** D6, D10, D14 (MCP client: write one, or adopt a Go MCP
client SDK).

### F14 — Cobra fits the subcommands; extension flags are the sharp edge (R)

Hand-rolled `parseArgs` in `cli/args.ts` collects unknown flags into
`unknownFlags` for extensions. Cobra can `FSet.ParseErrorsWhitelist.UnknownFlags`
or use a passthrough, but the help text today includes extension-registered
options. That interaction has to be designed; it is not a blocker.

`pi-ai` also ships a small `cli.ts` (OAuth helper). Coding-agent is the
user-facing CLI.

**Bears on:** D15 (Cobra command layout).

### F15 — Viper can read the JSON settings; schema and merge rules are the work (R)

Viper supports JSON, env prefix, and nested keys. Pi's merge rules are
specific: project overrides user; resource lists concatenate; `defaultTools`
has `+name` / `-name` patch semantics; some keys are global-only
(`defaultProjectTrust`, `deviceId`). Viper's default merge will not
reproduce that without an explicit layer.

Keybindings, themes, and MCP server blocks are structured JSON. Keeping
the on-disk schema lets TypeScript Pi and Go Pi share a home directory
(D11). Changing it to YAML because Viper prefers YAML splits the user
base.

**Bears on:** D11, D16 (settings schema).

### F16 — License permits a rewrite; identity does not come along (R)

The TypeScript tree is MIT, Copyright (c) 2025 Mario Zechner. A rewrite
that copies substantial source must retain that notice. A clean-room
implementation from the public docs and observed behaviour is a different
legal posture. This report does not choose one.

The product names `pi`, `pi.dev`, the logo, and npm scope
`@earendil-works` are the upstream project's. This repository is a
separate rewrite and needs its own binary name, config directory, and
User-Agent before it talks to providers (see also `go-llmprovider-sdk`
identity findings).

**Bears on:** D17 (name, config dir, User-Agent), D18 (derived vs
clean-room).

## Charm / Cobra / Viper mapping

| Pi surface | Stated Go stack | Fit |
|---|---|---|
| `cli/args.ts`, package/auth/mcp subcommands | Cobra | Good. Subcommands already exist. |
| `settings.json`, `PI_*` env | Viper | Good for load/bind; custom merge for patch lists. |
| Component tree, differential render, alt screen | Bubble Tea + Lip Gloss | Rewrite. Elm architecture replaces the component tree. |
| Editor (kill ring, undo, autocomplete) | Bubbles textarea + custom | Large custom widget. |
| Markdown + LaTeX | Glamour + extra | Markdown yes; LaTeX/mermaid extra. |
| Inline images | Charm image / custom Kitty | Feasible; needs a terminal-capability matrix. |
| Native clipboard / modifiers | `atotto/clipboard` + `x/sys` | Text yes; image/files/modifiers need platform code. |
| Themes as JSON | Lip Gloss styles from JSON | Feasible; OKLCH math is extra (`lucasb-eyer/go-colorful` is already in `ocp-login`). |

`ocp-login` already runs this CLI/TUI combination on darwin, linux, and
windows. That is existence proof for the *stack*, not for an agent
transcript UI.

## Per-package feasibility

| Package | Go product rewrite | 1:1 API port | Notes |
|---|---|---|---|
| coding-agent CLI (print/json/rpc/TUI) | Feasible | N/A (new binary) | Core of this repository |
| coding-agent TypeScript SDK | No | No | Language surface (F4) |
| coding-agent extensions (TS/jiti) | Only with a JS host | No | F1 |
| agent-core loop | Feasible | Approximate | Event/tool types become Go structs |
| ai providers | Feasible, large | No (and browser no) | Prefer `go-llmprovider-sdk` (F8) |
| tui | Feasible as Charm rewrite | No | F5 |
| mcp client | Feasible | Approximate | F13, D14 |
| codemode | Feasible if a JS VM is embedded | Approximate | F3 |
| skills / prompts / themes | Feasible | Yes (data) | F2 |
| session JSONL v3 | Feasible | Yes if chosen | F9 |
| chord | Feasible as new design | No (JS proxies) | F10 |
| durable / Pico harness | Feasible, large | Approximate | F10; not on the shipping CLI path |
| protocol/client/server | Feasible | Approximate | Experimental; CBOR in Go is easy |
| telemetry | Feasible | Approximate | Small |
| evals | Feasible later | N/A | Docker harness |
| sqlite-node | Replace with a Go SQLite driver | No | Node builtin |

## Decisions a MADR here needs to make

Each is open. The evidence column says which finding bears on it.

| # | Decision | Options seen | Evidence |
|---|---|---|---|
| D1 | What "complete" means for v1 | shipping `pi` CLI only; CLI plus Go SDK; whole monorepo including Chord/Pico/protocol | F1, F4, F10 |
| D2 | Extension host | none in v1 (skills/themes/prompts only); embed JS; WASM/RPC plugins; Go `plugin` | F1, F2 |
| D3 | Codemode | embed QuickJS WASM; drop; replace with a non-JS sandbox | F3 |
| D4 | SDK | Go module for embedders; RPC-only; none | F4, F11 |
| D5 | Wire compatibility | keep RPC/JSON records and session v3; new contracts | F9, F11 |
| D6 | Charm generation | Charm v2 as in `ocp-login`; Charm v1 (`github.com/charmbracelet/…`); extract shared widgets into `go-tui-lib` as they appear | F5, F13 |
| D7 | Package manager | data-only; shell out to npm and refuse JS; full JS host | F1, F2, F7 |
| D8 | Image pipeline | Go `image` stack; embed Photon WASM | F6 |
| D9 | Distribution | GitHub-release binaries + self-update; also npm wrapper; keep curl installer | F7 |
| D10 | Providers | depend on `go-llmprovider-sdk` and extend it; vendor a Pi-shaped layer here | F8, F13 |
| D11 | Home directory | share `~/.pi` and JSON settings with TypeScript Pi; use a new dir | F9, F15, F16 |
| D12 | Experimental stack | out of v1; in a later tree; in this module from the start | F10 |
| D13 | grep/find | pure Go; require `rg`/`fd` | F12 |
| D14 | MCP client | write against the MCP spec; adopt a Go client SDK | F13 |
| D15 | Cobra layout | one root command with modes as flags (Pi today); subcommands per mode | F14 |
| D16 | Settings merge | reimplement Pi's JSON patch rules; simplify | F15 |
| D17 | Product identity | binary name, config dir, User-Agent, module path | F16 |
| D18 | Provenance | clean-room from docs; derived rewrite retaining MIT notice | F16 |

## Shape of the work

For orientation only; a PLAN owns the steps after a MADR chooses D1–D18.

A CLI-first rewrite, given D1 = shipping `pi`, looks like:

1. Seed this module: `go.mod`, Cobra root, Viper JSON settings, CI on
   darwin/linux/windows, identity (D17).
2. Agent loop + built-in tools + session JSONL (D5, D11, D13).
3. Provider/auth via `go-llmprovider-sdk` or an in-tree layer (D10),
   starting with a small provider set and expanding.
4. Print, JSON, and RPC modes (D5) — these prove the loop without a TUI.
5. Charm interactive TUI (D6): transcript, editor, selectors, themes.
6. MCP client (D14), skills/prompts/themes (D2 data path).
7. Package manager and extension host as D2/D7 allow.
8. Codemode if D3 keeps it.
9. Self-update and installers (D9).
10. Chord/Pico/protocol only if D12 says so.

Order 2–4 is the smallest program that is recognisably Pi. The TUI is
the largest user-visible piece after that. Extensions and the
experimental stack dominate calendar time if they are in scope.

## Not verified

- No TypeScript test or `npm run build` was run at `312184edb`.
- No Go scratch module was written or compiled for this report.
- No live provider request was made. Provider-count and API-family claims
  are from source inventory, not a runtime catalog dump.
- Charm v2 capability vs Pi's Editor, CSI 2026, Kitty images, and
  modifier-key queries was not prototyped.
- wazero + `quickjs.wasm` was not loaded.
- Session v3 samples were not round-tripped through a Go decoder.
- Behavioural parity of a Go grep/edit/read against the TypeScript tools
  was not measured.
- Whether `go-llmprovider-sdk`'s OpenAI/Anthropic/xAI wire details match
  `pi-ai` was not compared byte-for-byte.
- Legal review of derived-vs-clean-room (D18) was not done.

## Later observation (2026-09-30)

This report is an observation from 2026-09-29. It is not rewritten. The
library rows in F13 and the "each is open" table above are that day's
facts. On 2026-09-30 the sibling libraries were re-probed, and the
decisions that this report left open were recorded in MADRs:

* **`go-llmprovider-sdk`** is the module
  `github.com/maccavelli/go-llmprovider-sdk` (Go 1.27.1, stdlib +
  `x/term`). It has a v1 contract (`Provider`, `Stream`, `APIError`,
  `providers.New`, `llmtest`) on `origin/main` at
  `3d4aff5f2be9363877aae0987755e6781f1cae98`. 0015-PLAN is in S7: four
  providers implement the contract (`openai`, `claude`, `gemini`,
  `grok`); the rest still use the old `Generate*` API. There is no git
  tag and no LICENSE file. Native streaming is `Unsupported` on every
  moved provider; `Stream` synthesises events from `Generate`.
* **`go-core-lib`** is tagged `v1.0.0` / `v1.0.1` / `v1.1.0`
  (`v1.1.0` = `96b30961180671ab3697585951219001ecbb1c90`, Apache-2.0).
  The live package is `selfupdate`. It is not a README stub.
* **D10, D9 (toolchain), D14** are decided in
  [0004-MADR](../decisions/0004-MADR-go-module-architecture.md).
  **`gobble update`** is decided in
  [0005-MADR](../decisions/0005-MADR-v1-feature-scope.md).

The feasibility conclusions (source-faithful port is not achievable; a
product rewrite is) are unchanged.

## Later observation (2026-10-01)

The owner directed Apache-2.0 for this repository, `go-llmprovider-sdk`,
and `go-core-lib`. `go-core-lib` already was. gobble's `LICENSE` is the
fleet copy; `NOTICE` credits Pi (MIT) and both sibling libraries.
`go-llmprovider-sdk` records the same choice in
`0018-MADR-apache-2-license.md`. The 2026-09-30 "no LICENSE file" line
for that SDK is that day's fact.
