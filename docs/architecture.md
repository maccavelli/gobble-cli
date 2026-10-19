# Architecture

How `gobble-cli` is put together, as it is now. This file carries no history and no rationale — when
you want to know *why* something is this way, the decision records under [decisions/](decisions/)
hold the argument, and [README.md](README.md) points at the ones people ask for most.

## What it is

One Go module, `github.com/maccavelli/gobble-cli`, at Go 1.27.2. It builds one binary, `gobble`, a
coding agent, and it is also an embeddable Go SDK. Shipped binaries are pure Go (`CGO_ENABLED=0`).

The binary has these surfaces:

- **The native CLI mode, the default.** `gobble` with no command is `chat`: a line session in a
  terminal, or one prompt with `-p` or piped input. It is an ACP client of gobble's own agent,
  in process.
- **The ACP agent.** `gobble acp` serves the agent on stdio, for editors and magic-cli-remote.
  `gobble acp --login <provider>` signs in, or stores a key, in a terminal an ACP client opens for
  its terminal auth method.
- **Commands.** `version`, `completion`, `config path|get|set|unset|edit|init|update|validate|schema`,
  `mcp add|remove|list`, `models list|info`, `auth login|logout|status|check|import`,
  `personality list|show|check|path|reset`, `trust [DIR] [--list]` and `untrust [DIR] [--forget]`.
- **Settings and trust.** Settings are a commented `settings.yaml` in the config directory, created
  from an embedded template on the first run. A trusted project's `.gobble/settings.yaml` and
  `--set path=value` layer over it, and flags over all of them. A project is trusted by
  `--approve`, by having no file that needs trust, by a decision saved in `trust.json`, by
  `defaultProjectTrust: always`, or by the answer to the line session's question; `--no-approve`
  and `defaultProjectTrust: never` refuse it, and with nothing decided it is not trusted. A trusted
  project's own `SYSTEM.md`, `APPEND_SYSTEM.md`, skills, prompt templates and personality files
  load with its settings.
- **Permissions.** Every tool call is decided by `permissions.rules`, an ordered list of `allow`,
  `ask` and `deny` rules: the global file's, then a trusted project's, then `--set`'s. A rule
  names a tool, or a tool and a pattern over each path the call names, relative to the project,
  or over each simple command of a bash call (PowerShell's whole command). The last that matches decides; with none, a read-only
  built-in tool runs and every other call, an MCP tool's among them, asks. A path outside the
  workspace asks unless an `external_directory` rule allows it, and a tool denied whole is hidden
  from the model. Whatever the rules say, a write under `.git` or `~/.ssh`, or to gobble's
  credential, trust and permission files, is refused; a write to gobble's or the project's
  configuration, an `AGENTS.md` or the project's skills, and every delete, ask. Print mode
  refuses every question; `--yes` allows once what a rule or the default asks, never those. Each
  decision is logged with its rule and source. A question offers allow and reject once, for
  this session, for this project (kept in the config directory's `permissions.json`, by the
  project root's real path, trusted or not) or everywhere (appended to the global
  `settings.yaml`); only "once" when a limit asked, for a repeated call, or when no rule can
  match the call alone (a glob in a path or a command's prefix). An "always" saves the call's
  paths exactly, and a shell command's arity prefix (`git status *`), which the options name; it
  applies from the next call. A plain `rm`, `rmdir`, `mv`, `cp` or `mkdir` runs as the native
  tool, through its rules and limits, and `gobble update` asks by a default rule. A reject can carry a reason the model reads
  (`[f]` in the line view). `/permissions` lists the rules in force with their sources, removes
  a session, project-store or global-file rule, and adds one to any of the three.
- **Modes.** A session runs in `default`, `accept-edits` (file edits allowed where no rule of
  the user's decides; deletes, commands and outside paths ask), `plan` (read-only, whatever the
  rules allow, until a plan is approved) or `bypass` (every question answered but gobble's own
  limits'; deny rules still refuse), offered only with the global `permissions.allowBypass`. `/mode`,
  `/plan`, ACP's `session/set_mode` and `--mode` change it; it is not saved with the session.
- **Grants.** `request_permissions`, deferred until a declined file call or `tool_search` loads it,
  asks the person for read or write access to paths, for the turn or the session, with a reason;
  a grant answers the questions about calls under its paths, deny rules and gobble's limits still
  holding, and `/permissions` lists and removes grants after the rules.
- **The enhanced TUI mode.** Not built yet: `internal/tui` is a placeholder, and go-tui-lib is not
  required yet.

The command library is Kong, in `internal/cli` and `internal/cli/complete` only. Sessions are stored
as JSONL in Pi's v3 shapes, under gobble's data directory.

## Packages

`go list ./...` prints each of these. The module root holds no package; `go.mod` carries the `tool`
directives.

The tier is each package's `Stability:` line in its `doc.go`. **stable** packages are covered by
`apidiff` from `v1.0.0`. **beta** packages are public and may change in a minor release.
**internal** and **experimental** packages carry no promise. `internal/frontmatter` and
`internal/config/genschema` have no `doc.go`; both are internal by their path.

| Package | Tier | What it is |
| :--- | :--- | :--- |
| `cmd/gobble` | internal | The `gobble` command: `main` prepares the console, then calls `internal/cli`. |
| `gobble` | stable | The embedding facade. |
| `agent` | stable | The turn loop: model calls, tool calls one at a time, approval, and the steering and follow-up queues. Tools are offered by tier: direct ones are sent, deferred ones wait behind `tool_search` until a call loads them with their family, and hidden ones are neither sent nor found. With the network down it probes the provider's host every 3 s, up to the session's wait, and resends; an empty reply runs again after 2 s and 4 s; each wait is a `Retry` event. Before each model call, and once on a context overflow, it asks its compaction hook. Three identical calls in a row ask whether the turn may go on; three consecutive calls whose arguments are refused end it; a call whose name differs from a tool's only in case runs that tool; a refusal ends the turn. |
| `llm` | stable | The model facade the agent sees: requests with an output cap, streamed events, usage, and the errors the loop acts on (`ErrContextOverflow`, `ErrOffline`, `ErrEmptyReply`); the session's retry settings (`Retry`, `DefaultRetry`) and the optional `Prober`. |
| `llm/provider` | beta | `llm` over go-llmprovider-sdk: each provider built by id, wrapped in the SDK's retry with the session's settings, or none when retry is off; a request the network could not carry is `llm.ErrOffline`, probed at the host it went to, and the SDK's empty answer is `llm.ErrEmptyReply`. The provider is chosen by `defaultProvider`, or the first, in the SDK's registry order, with a credential from any source. The Grok and Kilo logins, a Pi login's import, and a `LoginStore` adapted to the SDK's token store. |
| `llm/catalog` | beta | The embedded models.dev snapshot, the level and price rules (0005-PLAN F4-2a), `models.json` (F4-2b), and Pi's model resolution (F4-2c). |
| `llm/llmtest` | stable | A scripted `llm.Provider` and request assertions, for tests. |
| `tool` | stable | The tool contract, with the workspace roots each call gets. A tool publishes a schema with its bounds and defaults, and validates the Go type's own. A tool's spec carries its exposure and family, and a result can name tools to load. A call's environment carries the client's files and terminals when it offers them, and a tool names its locations before it runs and in its result. A result can say its arguments were refused, which the agent counts; `Redescribed` gives a tool a user's description. |
| `tool/builtin` | beta | The built-in tools: `read` (numbered lines, directory listings), `write`, `edit` (multi-edit with layered, guarded matching and a compare-and-swap), `grep` (RE2, `path:N: text`, sorted), `find` (globs), `tree` (a breadth-first entry budget), `move`, `delete`, `copy` and `mkdir` (confined, both paths locked in a fixed order, a cross-device move copied then deleted), `todo` (the task's checklist, replaced whole by each call and returned as the result's plan), `tool_search` (ranks the deferred tools and loads the best, offered only while one is deferred), the three MCP resource tools (`list_mcp_resources`, `list_mcp_resource_templates` and `read_mcp_resource`, over a source `mcpclient` implements), and `bash`, which with `UseClientTerminal` runs in the client's terminal when it offers one, and, on Windows, `powershell` (process trees, a default timeout, the whole output of long commands kept 7 days). Descriptions are Markdown templates in describe/, filled from the options; a user's template replaces a built-in's when it passes the same checks. A bash call's permission subjects are its simple commands, parsed with `mvdan.cc/sh`, each with opencode's arity prefix for an "always", Kilo's masking, wrappers and `bash -c` unwrapped, and a raw subject for what does not parse; its outside paths include `cd`'s and file operands'; a plain `rm`, `rmdir`, `mv`, `cp` or `mkdir` is handed to the native tool (0012-MADR D10). |
| `tool/toolsearch` | beta | An in-house BM25 over tool specs, with the name, the first sentence, the rest of the description and the schema's properties weighted 3, 2, 1 and 1. |
| `permission` | beta | The policy contract that approves tool calls, which also asks whether a run of identical calls may go on. Project trust: `DecideTrust`'s order (the run's flag; nothing that needs trust; a hook's answer, reserved for hooks; the nearest saved decision; `defaultProjectTrust`; the question), and `trust.json`, keyed by real path, locked for every change and written owner-only, which never holds a home directory or a filesystem root. The rules (0005-PLAN F6-2a): `ParseSpec` reads `<tool>` or `<tool>(<pattern>)`, and `Engine` decides a call by the last rule that matches over its layers, the defaults before them, the strictest of the call's subjects, `external_directory` for each path outside the roots, then gobble's limits, which no rule lifts; `Hidden` names a tool denied whole. `Remember` turns an "always" answer into rules that match the call alone, and `RuleStore` keeps `permissions.json`, locked and owner-only (0005-PLAN F6-2b). |
| `session` | stable | The session log contract, in Pi's JSONL v3 shapes: every Pi entry type, with `custom` entries for extension state such as `gobble.todo`. v1 and v2 entries are migrated in memory, and a session's name is Pi's. `Tree` keeps the entries with a leaf that moves as Pi's does, resolves labels, and writes a path as Pi's `createBranchedSession` does. |
| `session/jsonl` | stable | Sessions as JSONL files, with a writer lock. A v1 or v2 file is read and migrated, never written: an append to it is refused. |
| `compaction` | beta | Compaction over gobble's session entries: Pi's cut point and split turn; when to compact before a model call (over the window less `reserveTokens`, or at `thresholdPercent` of it, or at `thresholdTokens`); a summary request capped at 80 percent of `reserveTokens`, on opencode's template with a rolling update from the previous summary, and refused when it is empty. A Pi-format summary still counts as one. The user may replace each of the summary's four prompts. It owns the model's context as Pi projects it, with every entry type and context edit applied. |
| `command` | beta | The slash-command vocabulary shared by the agent and its clients. |
| `prompt` | beta | The system prompt: Pi's sections, from the context files (`AGENTS.md` and its kin, from the config directory and every directory above the cwd), `SYSTEM.md` and `APPEND_SYSTEM.md`, the personality's brief and its rules, and the user's tools and rules sections; the patches between two sets of sections; the nested context files a read brings in; and prompt templates, loaded and expanded as Pi loads and expands them (`$1`, `$@`, `${1:-default}`, `${@:N:L}`, shell-like quoting). A trusted project's own `SYSTEM.md` and `APPEND_SYSTEM.md` win over the user's; `TrustFiles` lists a project's files that wait for trust. |
| `skill` | stable | Agent Skills, found as Pi finds them, the first of two with one name kept: a trusted project's `.gobble/skills` and each `.agents/skills` from the cwd up to the repository root, then the config directory's `skills/`, then `~/.agents/skills`. The system prompt lists them, and `/skill:<name>` sends one's body. |
| `personality` | beta | gobble's voice: a Markdown catalog of every message a person reads, one file per personality (`boring`, `dry`, `snarky`, `bombastic`), each variant keeping boring's placeholders; the briefs the system prompt carries; the user's own personalities and changes under `personalities/` in the config directory; and a trusted project's messages over them for one session. |
| `hook` | stable | The lifecycle hook contract. |
| `checkpoint` | beta | File snapshots for undo (placeholder). |
| `telemetry` | beta | OpenTelemetry wiring (placeholder). |
| `acpserver` | beta | gobble's ACP agent: sessions, prompts, slash commands, modes, compaction, MCP servers and the `_gobble/` extension methods. `initialize` advertises an authentication method per provider, and terminal sign-ins when the client offers them. A session's settings, trust and project files are fixed when it starts. A session's mode is `default`, `accept-edits`, `plan` or `bypass`, which only `permissions.allowBypass` offers, and starts as `Override.Mode` gives it. `request_permissions` is the session's own deferred tool, unless the tool flags leave it out (`Options.NoRequestPermissions`); its grants answer questions under their paths, and a declined call a grant could cover loads it. A tool result's plan is sent as a `plan` update, recorded as a `gobble.todo` entry, replayed by `session/load` and shown by `/todos`. Each tool load is recorded as a `gobble.tools` entry, and a session's next prompt starts with every tool its path has loaded; the MCP resource tools load at the first prompt that has them. A client that offers reading and writing files has read, write and edit do their text I/O through `fs/*`, after the path is confined here; one that offers terminals gets bash's commands through `terminal/*`. `tool_call` and `tool_call_update` carry each call's locations. A session's history, its replay and `_gobble/get_messages` follow Pi's context of every entry type; a prompt, `/name` or `/compact` on an older file is refused, and `/clone` continues it in a v3 file. A live session's entries and leaf are a `session.Tree`, every path is its, and `/fork` and `/clone` copy labels as Pi does. A session's system prompt is a baseline, kept in a `gobble.prompt` entry and fixed; a change of its sections is a patch, sent as a message at its place, and `/compact` writes a new baseline. A read below the cwd adds each nested `AGENTS.md` once per session, recorded in a `gobble.context` entry. Prompt templates are found when a session starts, a trusted project's `.gobble/prompts` before the config directory's `prompts/`, and advertised with their argument hints; `/<name> [args]` sends one's expansion, a built-in winning over a template of its name, and a queued message is expanded as it is queued. Retry waits are `_gobble/retry` notices, and `_gobble/set_auto_retry` and `_gobble/abort_retry` steer them; a session compacts on its own at the threshold and on an overflow, with `_gobble/compaction` notices and `_gobble/set_auto_compaction`. A session speaks its personality, which `/personality`, `_gobble/set_personality` and a `gobble.personality` entry carry, and a level-5 personality asks first. With `smallModel` set, a session is named from its first prompt in the background. `/trust` shows the project's decision and saves an answer, and an untrusted project is told once. Each tool call is decided by the session's permission rules and gobble's limits, and each decision is logged with its rule and source; a question one of the limits asks carries `_meta.gobble.limit`, a repeated call's `_meta.gobble.repeated`, and a tool the rules deny whole is left out of the request, `tool_search` and the system prompt. A question's eight options save rules to the session, the project store or the global file, it carries its deciding rule in `_meta.gobble.rule`, a reject's `_meta.gobble.feedback` reaches the model, and `/permissions` lists, removes and adds rules. The commands are `/help`, `/compact`, `/usage`, `/context`, `/session`, `/model`, `/thinking`, `/mode`, `/plan`, `/name`, `/fork`, `/clone`, `/todos`, `/personality` and `/trust`. A prompt's files come ahead of its text, as Pi's CLI writes them: each embedded resource and `resource_link`, then each file an `@path` word names, read inside the workspace's roots; a text block marked `_meta.gobble.mentions: false` is not scanned, and a queued message is given its files too. `initialize` advertises `embeddedContext`, and no images. |
| `acpclient` | beta | gobble's own ACP client, used by `internal/cli`. A prompt's `Context` goes ahead of its text as text blocks marked `_meta.gobble.mentions: false`; the CLI puts its stdin and attached files there. The `_gobble/retry`, `_gobble/compaction` and `_gobble/personality` notices reach it as updates. A permission request's `_meta.gobble` limit, repeated and once-only marks reach `Options.Permission`, and an answer's `Feedback` goes back as `_meta.gobble.feedback`. |
| `acpclient/acptest` | stable | An agent and a client over in-memory pipes, with a frame recorder and golden transcripts, for tests. Its client can serve files and scripted terminals, and records extension notices. |
| `mcpclient` | beta | MCP servers over stdio and streamable HTTP, through the go-sdk. Each stdio server runs as an `internal/proctree` tree and is stopped as Pi stops one. A server that advertises resources is listed, read and searched through the resource tools. |
| `mcpclient/mcptest` | beta | An MCP server fixture for tests, over stdio and HTTP. |
| `internal/cli` | internal | The command edge: the Kong grammar, `chat` and its flags, the line session, print mode, the exit codes, signals and output. Project trust for the run: `--approve` and `--no-approve`, the line session's question before the session starts, `gobble trust` and `gobble untrust`, and a trusted project's personality and prompt files for each session. Each session's permission rules, the line view's numbered options and `[f] deny and say why`, `permissions.json` and the global file's edits for the agent, and `--yes`, with which print mode allows once a question a rule or the default asks. The personality's welcome, goodbye and ambient lines, the small model's provider, and `auth import --from-pi`. It holds `selfupdate-release.json`, the release spec. |
| `internal/cli/complete` | internal | Native shell completion for bash, zsh, fish and PowerShell. |
| `internal/cli/editor` | internal | The line editor: `golang.org/x/term` behind gobble's adapter, with history, paste and an external editor. |
| `internal/cli/render` | internal | Terminal output of the native CLI mode: streamed Markdown, tool output, a status line and a spinner whose label the personality may change. |
| `internal/cli/term` | internal | Terminal capabilities, styles, escape sanitising and Windows console setup. |
| `internal/tui` | internal | The enhanced TUI mode (placeholder). |
| `internal/appdirs` | internal | The config, data, state and cache directories, created owner-only. |
| `internal/buildinfo` | internal | The build's version, commit, toolchain, platform and User-Agent. |
| `internal/logging` | internal | The one slog logger: a rolling owner-only file, and warnings on stderr, both redacted. |
| `internal/archtest` | internal | The import rules below, as a test. |
| `internal/auth` | internal | `<config>/auth.json` in Pi's shape, owner-only and locked for every change, and Pi's credential order (0005-PLAN F4-3a); its OAuth logins as `provider.LoginStore`, each provider's refreshes under their own lock (0005-PLAN F4-3b). |
| `internal/config` | internal | Settings: a commented `settings.yaml` from an embedded template, layered global, project (trusted) and `--set`, edited in place; the JSON Schema. `permissions.rules` is the one list, whose layers are joined, not replaced, and which `AppendRule` and `RemoveRule` edit in place. |
| `internal/config/genschema` | internal | The command that writes the settings' JSON Schema, `internal/config/testdata/settings.schema.json`. |
| `internal/filelock` | internal | An exclusive lock file between processes, for session files, `mcp.json`, settings, `auth.json` and `trust.json`. |
| `internal/proctree` | internal | A command as a tree that stops whole: its own process group on Unix, a kill-on-close Job object on Windows. |
| `internal/frontmatter` | internal | The YAML frontmatter of skills and templates, as Pi reads it, through `go.yaml.in/yaml/v3`. |
| `internal/fsx` | internal | Workspace confinement for file tools: Pi's path resolution, `os.Root` over the working and additional directories, atomic writes, and a per-path lock. A glob engine written to git's rules, the repository's ignore rules, and a confined walker that applies them. Renames, removals, copies and directories through the roots. The repository finder. |
| `exp/a2a` | experimental | Agent2Agent endpoint (experimental placeholder). |
| `exp/acpws` | experimental | ACP over WebSocket (experimental placeholder). |
| `exp/codemode` | experimental | Code-mode sandbox (experimental placeholder). |
| `exp/sandbox` | experimental | OS sandbox for shell tools (experimental placeholder). |
| `exp/wasmtool` | experimental | WASM tool host (experimental placeholder). |

## Import rules

`internal/archtest` (`make archtest`) checks these on `go list -deps -json ./...`:

1. The core packages (`agent`, `llm`, `tool`, `session`, `skill`, `hook`, `compaction`, `prompt`,
   `command`, `permission`, `checkpoint`, `personality`) import no ACP, MCP or provider SDK, no
   vendor LLM SDK, no go-core-lib or go-selfupdate-lib, and no Charm; the OpenTelemetry SDK only
   in `telemetry`. Kong is kept out of them by rule 9.
2. Only `acpserver` and `acpclient/...` import the ACP SDK.
3. Only `mcpclient` imports the MCP go-sdk.
4. Only `llm/provider` imports go-llmprovider-sdk; no vendor LLM SDK is imported.
5. Only `internal/tui` may import Charm and go-tui-lib.
6. `internal/cli` and `internal/tui` never import `agent`; they go through `acpclient`.
7. Only `cmd/gobble` imports `exp/...`.
8. Only `internal/cli` imports go-selfupdate-lib, and `internal/buildinfo` its `buildinfo`.
9. Only `internal/cli/...` imports Kong.

## Dependencies

The direct requirements in `go.mod`:

| Module | Version |
| :--- | :--- |
| `github.com/alecthomas/kong` | `v1.16.1` |
| `github.com/maccavelli/acp-go-sdk2` | `v1.16.0` |
| `github.com/modelcontextprotocol/go-sdk` | `v1.8.0` |
| `github.com/maccavelli/go-llmprovider-sdk` | `v1.15.0` |
| `github.com/maccavelli/go-selfupdate-lib` | `v1.15.2` |
| `github.com/google/jsonschema-go` | `v0.4.3` |
| `go.yaml.in/yaml/v3` | `v3.0.5` |
| `golang.org/x/sys`, `golang.org/x/term`, `golang.org/x/text` | `v0.49.0`, `v0.47.0`, `v0.43.0` |

`apidiff`, `golint`, `govulncheck` and `staticcheck` run from `tool` directives.

## Checks

- `make pre-add-check`: gofmt and golint on the named Go files, or on every tracked one with no
  `FILES`, then govulncheck over the module; `GO_PRECHECK_SKIP_VULN=1` skips govulncheck.
- `make preflight`: first `check-cgo-off`, then every gate — gofmt, `go mod tidy`, the pre-add
  check, `go vet`, staticcheck for linux, darwin and windows, golangci-lint, govulncheck,
  `go fix -diff`, archtest, apidiff, build metadata, the records check and markdownlint.
- `make race`: the tests under the race detector, with cgo on.
- `make apidiff`: a no-op below `v1.0.0`. From a `v1.0.0` tag it fails until the comparison is
  written.
- Outside `preflight`: `make check-libs` compares the sibling libraries with their latest tags, over
  the network; `make probe-conhost` and `make completion-shells` run checks that need a console or
  the real shells.
- `.github/workflows/ci.yml`, on pushes to `main`, on `v*` tags and on pull requests: on Linux,
  macOS and Windows, `go test`, `make preflight` and `make archtest`; `make race` on Linux and
  macOS; on Linux, `go mod tidy -diff`, shellcheck and actionlint.
- The release, in the same workflow:
  - `sbom` takes the dependency graph's SPDX SBOM as `gobble.spdx.json`;
  - `build` is go-selfupdate-lib's build workflow over `internal/cli/selfupdate-release.json`. It
    builds the five binaries (darwin/arm64, linux/amd64, linux/arm64, windows/amd64,
    windows/arm64), checks them, writes `SHA256SUMS`, and runs `gobble version --identity` on each
    platform. On a tag the spec admits it builds the release; on any other run it is a rehearsal;
  - `release`, on a `v*` tag only, is the library's publish workflow, which releases and attests.

## Tree

```text
README.md                 repository entry; links here
LICENSE                   Apache License 2.0
NOTICE                    Pi MIT credit; models.dev and opencode MIT credits; sibling Apache-2.0 libraries
AGENTS.md                 agent instructions
.claude/ .grok/ .opencode/  per-agent pointers to AGENTS.md
opencode.json             opencode's project settings
Makefile                  the gates above
go.mod, go.sum            the module and its sums
.github/workflows/ci.yml  CI
.golangci.yml             the lint set
.markdownlint-cli2.jsonc  fleet Markdown style
.gitattributes            LF on every platform
.gitignore                ignored paths
scripts/                  the pre-add check, the records check, the build-metadata check, the sibling-library
                          check, the models.dev snapshot generator
cmd/ … exp/               the packages above
docs/
  README.md               ToC and the "I want to…" matrix
  architecture.md         this file
  guides/                 how-to guides: choosing-a-model.md, credentials.md, developing.md, models-json.md
  decisions/              MADR/PLAN pairs
  reports/                numbered observations
```
