# Architecture

How `gobble-cli` is put together, as it is now. This file carries no history and no rationale — when
you want to know *why* something is this way, the decision records under [decisions/](decisions/)
hold the argument, and [README.md](README.md) points at the ones people ask for most.

## What it is

One Go module, `github.com/maccavelli/gobble-cli`, at Go 1.27.1. It builds one binary, `gobble`, a
Go rewrite of the Pi coding agent, and it is also an embeddable Go SDK. Shipped binaries are pure Go
(`CGO_ENABLED=0`).

The binary has these surfaces:

- **The native CLI mode, the default.** `gobble` with no command is `chat`: a line session in a
  terminal, or one prompt with `-p` or piped input. It is an ACP client of gobble's own agent,
  in process.
- **The ACP agent.** `gobble acp` serves the agent on stdio, for editors and magic-cli-remote.
- **Configuration commands.** `version`, `completion`, `config path`, and `mcp add|remove|list`.
- **The enhanced TUI mode.** Not built yet: `internal/tui` is a placeholder, and go-tui-lib is not
  required yet.

The command library is Kong, in `internal/cli` only. Sessions are stored as JSONL in Pi's v3 shapes,
under gobble's data directory.

## Packages

`go list ./...` prints each of these. The module root holds no package; `go.mod` carries the `tool`
directives.

The tier is each package's `Stability:` line in its `doc.go`. **stable** packages are covered by
`apidiff` from `v1.0.0`. **beta** packages are public and may change in a minor release.
**internal** and **experimental** packages carry no promise.

| Package | Tier | What it is |
| :--- | :--- | :--- |
| `cmd/gobble` | internal | The `gobble` command: `main` calls `internal/cli`. |
| `gobble` | stable | The embedding facade. |
| `agent` | stable | The turn loop: model calls, tool calls, approval, and the steering and follow-up queues. |
| `llm` | stable | The model facade the agent sees. |
| `llm/provider` | beta | `llm` over go-llmprovider-sdk, with the ambient credential selection. |
| `llm/catalog` | beta | The model catalog (placeholder). |
| `llm/llmtest` | stable | A scripted `llm.Provider` and request assertions, for tests. |
| `tool` | stable | The tool contract, with the workspace roots each call gets. |
| `tool/builtin` | beta | The built-in tools: `read` (numbered lines, directory listings), `write`, `edit` (multi-edit with layered, guarded matching and a compare-and-swap) and `bash`. |
| `tool/toolsearch` | beta | Deferred tool exposure (placeholder). |
| `permission` | beta | The policy contract that approves tool calls. |
| `session` | stable | The session log contract, in Pi's JSONL v3 shapes. |
| `session/jsonl` | stable | Sessions as JSONL files, with a writer lock. |
| `compaction` | beta | Pi's manual compaction: the cut point, the summary and the entry. |
| `command` | beta | The slash-command vocabulary shared by the agent and its clients. |
| `prompt` | beta | System prompt assembly (placeholder). |
| `skill` | stable | Agent Skills parsing and discovery. |
| `hook` | stable | The lifecycle hook contract. |
| `checkpoint` | beta | File snapshots for undo (placeholder). |
| `telemetry` | beta | OpenTelemetry wiring (placeholder). |
| `acpserver` | beta | gobble's ACP agent: sessions, prompts, slash commands, modes, compaction, MCP servers and the `_gobble/` extension methods. |
| `acpclient` | beta | gobble's own ACP client, used by `internal/cli`. |
| `acpclient/acptest` | stable | An agent and a client over in-memory pipes, with a frame recorder and golden transcripts, for tests. |
| `mcpclient` | beta | MCP servers over stdio and streamable HTTP, through the go-sdk. Each stdio server runs in its own process group, or a Job object on Windows, and is stopped as Pi stops one. |
| `mcpclient/mcptest` | beta | An MCP server fixture for tests, over stdio and HTTP. |
| `internal/cli` | internal | The command edge: the Kong grammar, `chat` and its flags, the line session, print mode, the exit codes, signals and output. It holds `selfupdate-release.json`, the release spec. |
| `internal/cli/complete` | internal | Native shell completion for bash, zsh, fish and PowerShell. |
| `internal/cli/editor` | internal | The line editor: `golang.org/x/term` behind gobble's adapter, with history, paste and an external editor. |
| `internal/cli/render` | internal | Terminal output of the native CLI mode: streamed Markdown, tool output, a status line and a spinner. |
| `internal/cli/term` | internal | Terminal capabilities, styles, escape sanitising and Windows console setup. |
| `internal/tui` | internal | The enhanced TUI mode (placeholder). |
| `internal/appdirs` | internal | The config, data, state and cache directories, created owner-only. |
| `internal/buildinfo` | internal | The build's version, commit, toolchain, platform and User-Agent. |
| `internal/logging` | internal | The one slog logger: a rolling owner-only file, and warnings on stderr, both redacted. |
| `internal/archtest` | internal | The import rules below, as a test. |
| `internal/auth` | internal | The credential store (placeholder). |
| `internal/config` | internal | Layered settings (placeholder). |
| `internal/fsx` | internal | Workspace confinement for file tools: Pi's path resolution, `os.Root` over the working and additional directories, atomic writes, and a per-path lock. |
| `exp/a2a` | experimental | Agent2Agent endpoint (experimental placeholder). |
| `exp/acpws` | experimental | ACP over WebSocket (experimental placeholder). |
| `exp/codemode` | experimental | Code-mode sandbox (experimental placeholder). |
| `exp/sandbox` | experimental | OS sandbox for shell tools (experimental placeholder). |
| `exp/wasmtool` | experimental | WASM tool host (experimental placeholder). |

## Import rules

`internal/archtest` (`make archtest`) checks these on `go list -deps -json ./...`:

1. The core packages (`agent`, `llm`, `tool`, `session`, `skill`, `hook`, `compaction`, `prompt`,
   `command`, `permission`, `checkpoint`) import no ACP, MCP or provider SDK, no Kong and no Charm.
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
| `github.com/coder/acp-go-sdk` | `v0.13.5`, replaced by `github.com/maccavelli/acp-go-sdk` `v0.13.6-mcr.2` |
| `github.com/modelcontextprotocol/go-sdk` | `v1.8.0` |
| `github.com/maccavelli/go-llmprovider-sdk` | `v1.2.1` |
| `github.com/maccavelli/go-selfupdate-lib` | `v1.9.0` |
| `github.com/google/jsonschema-go` | `v0.4.3` |
| `golang.org/x/sys`, `golang.org/x/term`, `golang.org/x/text` | `v0.48.0`, `v0.46.0`, `v0.42.0` |

`golint`, `govulncheck` and `staticcheck` run from `tool` directives.

## Checks

- `make pre-add-check`: gofmt, golint and govulncheck on the named Go files.
- `make preflight`: every gate — gofmt, `go mod tidy`, the pre-add check, `go vet`, staticcheck and
  golangci-lint, govulncheck, `go fix -diff`, archtest, apidiff, build metadata, the records check
  and markdownlint.
- `make race`: the tests under the race detector, with cgo on.
- `make apidiff`: a no-op below `v1.0.0`. From a `v1.0.0` tag it fails until the comparison is
  written.
- `.github/workflows/ci.yml`: on Linux, macOS and Windows, `go test`, `make preflight` and
  `make archtest`; `make race` on Linux and macOS; on Linux, `go mod tidy -diff`, shellcheck and
  actionlint.
- The release, in the same workflow:
  - `sbom` takes the dependency graph's SPDX SBOM as `gobble.spdx.json`;
  - `build` is go-selfupdate-lib's build workflow over `internal/cli/selfupdate-release.json`. It
    builds the five binaries (darwin/arm64, linux/amd64, linux/arm64, windows/amd64,
    windows/arm64), checks them, writes `SHA256SUMS`, and runs `gobble version --identity` on each
    platform. On every push and pull request this is a rehearsal;
  - `release`, on a `v*` tag only, is the library's publish workflow, which releases and attests.

## Tree

```text
README.md                 repository entry; links here
LICENSE                   Apache License 2.0
NOTICE                    Pi MIT credit; sibling Apache-2.0 libraries
AGENTS.md                 agent instructions
Makefile                  the gates above
.github/workflows/ci.yml  CI
.golangci.yml             the lint set
.markdownlint-cli2.jsonc  fleet Markdown style
.gitattributes            LF on every platform
scripts/                  the pre-add check, the records check, the build-metadata check
cmd/ … exp/               the packages above
docs/
  README.md               ToC and the "I want to…" matrix
  architecture.md         this file
  guides/                 how-to guides: developing.md
  decisions/              MADR/PLAN pairs
  reports/                numbered observations
```
