---
status: proposed
date: 2026-09-29
decision-makers: repository owner
consulted: 0001-REPORT-go-port-feasibility.md, 0002-MADR-cli-acp-headless-mcp-v1.md, 0003-MADR-pigo-product-identity.md
informed: go-llmprovider-sdk, mcplib, go-tui-lib, magic-cli-remote
---
# pigo is one Go 1.27.1 module of contract-first packages, with an open standard at every boundary and a thin Cobra edge

## Context and Problem Statement

[0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md)
chose ACP as the agent command API. Its PLAN began with "`go mod init` and
a Cobra skeleton" and left the shape of the module open. Several report
decisions are still open and all bear on that shape: D4 (Go SDK), D10
(providers), D14 (MCP client), D15 (Cobra layout), D16 (settings merge),
and D9 (distribution).

The owner asked for pigo to be scaffolded in the most idiomatic Go 1.27.1
way that is also modular, open to extension, adherent to the relevant
SDKs, standard at the protocol level, and future-proof. The owner also
asked for v1 to include as much as possible
([0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md)). A wide
v1 only stays maintainable if the module separates contracts from
implementations and keeps every third-party SDK behind one package.

What the fleet and toolchain provide on 2026-09-29 (read in source or
probed):

* **Toolchain.** `go1.27.1`. `$GOROOT/api/go1.27.txt` lists
  `encoding/json/v2` and `encoding/json/jsontext` as public API, and the
  `jsonv2` experiment is on by default in `internal/buildcfg`. The same
  file lists a new standard-library `uuid` package (`New`, `NewV7`),
  `testing/synctest.Sleep`, `net/http/httptest.NewTestServer`, and
  `strings.CutLast`. Go 1.26 added `errors.AsType`, `slog.NewMultiHandler`,
  `testing.T.ArtifactDir`, and the spec change that lets `new` take an
  expression (`new(123)`). Go 1.25 added `sync.WaitGroup.Go`,
  `testing/synctest.Test`, `runtime/trace.FlightRecorder`,
  `http.CrossOriginProtection`, and the wider `os.Root` API. `go fix` runs
  the modernizer suite through `cmd/fix`.
* **ACP.** `github.com/coder/acp-go-sdk` v0.13.5 has no dependencies, uses
  `log/slog`, and targets ACP schema 0.13.5 (`ProtocolVersionNumber = 1`).
  magic-cli-remote pins it through a `replace` to the owner's fork
  `v0.13.6-mcr.1`, which adds a notification-overflow policy
  (magic-cli-remote MADR 0167). Stable in that schema: `session/list`,
  `session/resume`, `session/close`, `logout`. Unstable: fork, delete, NES,
  documents, providers, elicitation, `mcp/*`. `session/set_model` is **not
  in the schema**. The SDK's agent-side dispatcher sends only methods that
  begin with `_` to `ExtensionMethodHandler`, and answers every other
  unknown method with MethodNotFound (`agent_gen.go`, `extensions.go` at
  v0.13.5).
* **MCP.** The official `github.com/modelcontextprotocol/go-sdk` is the
  fleet's MCP library: `mcplib` v1.6.0 builds on go-sdk v1.6.1, and the
  module cache holds v1.8.0. `mark3labs/mcp-go` appears nowhere in the
  fleet.
* **Providers.** `go-llmprovider-sdk` has no `go.mod` and still imports
  `mcplib`. Its current API aggregates responses and has no public
  streaming. Its proposed MADR 0015 plans streaming `Event`s. The official
  vendor SDKs are in the module cache: `anthropic-sdk-go` v1.57.0,
  `openai-go/v3` v3.42.0, `google.golang.org/genai` v1.63.0.
* **CLI and TUI.** Cobra v1.10.2, Viper v1.21.0, `charmbracelet/fang`
  v1.0.0, and Charm v2 (`charm.land/bubbletea/v2` v2.0.9, `bubbles/v2`
  v2.2.1, `lipgloss/v2` v2.0.6) are used in production by `ocp-login`.
* **Fleet build conventions** (magic-cli-remote, mcplib, ocp-login):
  * Build: `CGO_ENABLED=0`, `-trimpath`, `-tags netgo,osusergo`, and
    `-ldflags -X main.version/commit/date`.
  * Code quality: `scripts/go-precheck.sh` (gofmt, per-file golint,
    govulncheck), golangci-lint v2 with the shared linter set, and a pinned
    staticcheck run for three GOOS values (magic-cli-remote MADR 0170).
  * CI: `setup-go` with `go-version-file: go.mod`. Releases from `v*`
    tags use `softprops/action-gh-release` with `SHA256SUMS`. Dependabot
    is off by policy, and no repository uses goreleaser.
  * Logging and self-update: `log/slog` everywhere, and `mcplib/selfupdate`
    for GitHub-release self-update.

The problem: choose the module shape, the package boundaries, the
contracts, the standards each boundary speaks, and the toolchain rules.
The shape has to carry a wide v1 without the CLI, the TUI, or any vendor
SDK leaking into the agent core.

## Decision Drivers

* Idiomatic Go:
  * package names equal their directory names;
  * small interfaces defined where they are consumed;
  * `context.Context` first;
  * no global mutable registries;
  * errors that can be inspected by type;
  * `internal/` for anything not promised.
* The protocol hierarchy is **ACP up** (clients: editors, mcremote, the
  pigo CLI, the pigo TUI) and **MCP out** (tool servers). Each has an
  official Go SDK that pigo uses, rather than re-implementing the wire.
* A public Go SDK (report D4) that embedders can import without pulling in
  Cobra, Viper, or Charm.
* Every third-party SDK (ACP, MCP, the vendor LLM SDKs, Charm, Cobra,
  OpenTelemetry) is imported by exactly one package or package family, so
  a replacement or bump is local.
* Future-proofing:
  * a JSON forward-compatibility story (unknown fields survive a
    round-trip);
  * API compatibility enforced by a tool, not by memory;
  * an `exp/` tree where new protocols can land without a stability
    promise.
* Fleet conventions (toolchain lockstep, precheck, release flow, no
  goreleaser, no dependabot) unless there is a stated reason to differ.
* Testability: a scripted fake model, in-memory ACP pairs, and
  deterministic time. No live network in the default `go test`.

## Considered Options

* One module; contract-first public packages; adapters behind them;
  `internal/` edge; `exp/` for unstable work
* Binary-only: everything under `internal/`; extract an SDK later
* Multi-module repository (`go.work`): separate modules for the SDK,
  providers, TUI, and CLI
* `pkg/` + `internal/` + `cmd/` "standard project layout"
* Build on an open-source Go agent framework (Google ADK for Go, CloudWeGo
  Eino, Genkit Go, LangChainGo)

## Decision Outcome

Chosen option: "One module; contract-first public packages; adapters
behind them; `internal/` edge; `exp/` for unstable work". It yields a
real Go SDK (D4) while keeping one version, one `go.mod`, and one CI
graph. It also puts each external SDK and each open protocol behind a
single package that the import-boundary test can enforce.

### Module

```
module github.com/maccavelli/pi-go

go 1.27.1

tool (
	golang.org/x/exp/cmd/apidiff
	golang.org/x/lint/golint
	golang.org/x/vuln/cmd/govulncheck
	honnef.co/go/tools/cmd/staticcheck
)
```

* The `go` directive is `1.27.1` in lockstep with the toolchain, as
  magic-cli-remote does (fleet MADR 0169 policy: newest supported,
  advisory-free). CI reads it through `go-version-file`.
* Developer tools are `tool` directives, so `go tool govulncheck ./...`
  needs no global install. golangci-lint v2 is installed by its own
  pinned installer, not as a `tool` dependency, as that project advises.
* ACP pin: `github.com/coder/acp-go-sdk v0.13.5`. The PLAN decides at
  Phase 0 whether to carry magic-cli-remote's `replace` to the fork
  `v0.13.6-mcr.1`. A `replace` only affects pigo's own build, not
  embedders. Recommendation: carry it, so the agent side has the same
  overflow policy that its main client relies on, and use the exit plan
  from magic-cli-remote MADR 0167.

### Package map

Stability tiers: **stable** packages are covered by `apidiff` from
`v1.0.0`. **beta** packages are public and may change in a minor release
with a changelog entry. **internal** and **exp** carry no promise.

```
cmd/pigo/                 main: signal.NotifyContext → cli.Main; nothing else
pigo/                     stable  Embedding facade: New(ctx, ...Option) → *Runtime;
                                  Runtime.ServeACP(ctx, r, w), Runtime.Connect(ctx) (in-process ACP client)
agent/                    stable  Turn loop, tool batches, steering/follow-up queues, events as iter.Seq
llm/                      stable  Message/Content/Usage/Cost/Model types; Provider contract; typed errors
llm/catalog/              beta    Model catalog: embedded snapshot, models.json overlay, refresh
llm/provider/anthropic/   beta    adapter → github.com/anthropics/anthropic-sdk-go (also Bedrock, Vertex)
llm/provider/openai/      beta    adapter → github.com/openai/openai-go/v3 (Responses + Chat Completions; Azure)
llm/provider/google/      beta    adapter → google.golang.org/genai (Gemini API + Vertex)
llm/provider/compat/      beta    OpenAI-compatible endpoints (Groq, Cerebras, DeepSeek, xAI, OpenRouter, Ollama, llama.cpp, vLLM, LM Studio, …)
llm/llmtest/              stable  Scripted fake provider and stream assertions for embedders' tests
tool/                     stable  Tool contract, Spec (JSON Schema 2020-12), Annotations, Kind, Exposure, Registry
tool/builtin/             beta    read write edit bash powershell grep find ls tool_search todo task web_fetch jobs
tool/toolsearch/          beta    BM25 index over tool specs (deferred exposure)
session/                  stable  Entry tree, projection, Store contract (Pi JSONL v3 superset)
session/jsonl/            stable  File store; reads v1/v2/v3; writes v3; preserves unknown fields
compaction/               beta    Cut-point selection, summariser, branch summaries
prompt/                   beta    System-prompt sections, context files (AGENTS.md…), templates, skills catalog
skill/                    stable  Agent Skills (SKILL.md) parsing and discovery
command/                  beta    Slash-command registry and router (native set; templates; skills; MCP prompts)
permission/               beta    Policy engine: rules, modes, decisions, trust
hook/                     stable  Lifecycle hook contract; exec-hook runner (JSON on stdin/stdout)
checkpoint/               beta    Shadow-git snapshots for /undo /redo /diff
mcpclient/                beta    MCP client manager → github.com/modelcontextprotocol/go-sdk
acpserver/                beta    acp.Agent implementation over agent/ → github.com/coder/acp-go-sdk
acpclient/                beta    Client-side helpers (in-process pipe, update fan-out) used by CLI/TUI/tests
acpclient/acptest/        stable  In-memory agent↔client pair and frame recorder for embedders' tests
telemetry/                beta    OpenTelemetry wiring (GenAI semantic conventions); no-op by default
internal/cli/                     Cobra/Viper/fang command tree; an ACP client, never imports agent/
internal/tui/                     Charm v2 program; an ACP client over acpclient/
internal/config/                  Typed settings, layered load, Pi merge semantics, JSON Schema emit
internal/appdirs/                 XDG/Known-Folder resolution (0003)
internal/auth/                    Keyring + file credential store; OAuth 2.1 flows (golang.org/x/oauth2)
internal/buildinfo/               ldflags + debug.ReadBuildInfo fallback
internal/logging/                 slog setup, rolling file sink, redaction
internal/fsx/                     os.Root workspace, atomic write, per-realpath mutation queue
internal/archtest/                Import-boundary test (below)
exp/codemode/                     wazero + QuickJS WASM sandbox (0001-REPORT F3)
exp/wasmtool/                     WASM tool plugins (wazero, WASI)
exp/acpws/                        ACP over WebSocket / Unix socket (remote-agent transport)
exp/a2a/                          Agent2Agent agent card and task endpoint
exp/sandbox/                      OS sandbox profiles for bash (Seatbelt, Landlock)
```

Names with a `client` or `server` suffix (`mcpclient`, `acpserver`)
avoid shadowing the SDK package names `mcp` and `acp`, which appear
alongside them in the same file.

### Contracts (shape, not final signatures)

```go
// package llm
type Provider interface {
	ID() string
	Stream(ctx context.Context, req *Request) iter.Seq2[Event, error]
}
type Event interface{ event() } // sealed: TextDelta, ThinkingDelta, ToolCallDelta, ToolCallDone, Usage, Done

// package tool
type Tool interface {
	Spec() Spec // name, description, input JSON Schema, annotations, ACP kind, exposure
	Run(ctx context.Context, call Call, env Env) (Result, error)
}
// Typed constructor: the input schema is derived from In with github.com/google/jsonschema-go
// (the same derivation the MCP go-sdk uses).
func New[In, Out any](name, description string, fn func(context.Context, In, Env) (Out, error), opts ...Option) Tool

// package session
type Store interface {
	Create(ctx context.Context, h Header) (Log, error)
	Open(ctx context.Context, id ID) (Log, error)
	List(ctx context.Context, f Filter) iter.Seq2[Summary, error]
}

// package hook
type Hook interface {
	Handle(ctx context.Context, ev Event) (Outcome, error) // Outcome: continue | block{reason} | modify{…}
}

// package pigo
func New(ctx context.Context, opts ...Option) (*Runtime, error)
// Options: WithProviders, WithTools, WithHooks, WithSessionStore, WithPolicy, WithLogger, WithTracerProvider, WithDirs
```

* Registries are values passed in through options. There is no `init()`
  self-registration and no package-level mutable map. Default sets are
  plain functions: `builtin.Tools()`, `provider.Defaults(cfg)`.
* Streams are `iter.Seq2[T, error]`. Callers `break` to stop, and the
  producer's `context.Context` carries cancellation.
* Retry and classification use typed errors:
  * `*llm.APIError{Status, RetryAfter}`, `llm.ErrContextOverflow`,
    `llm.ErrRateLimited`, `llm.ErrQuota`, `llm.ErrAuth`;
  * inspected with `errors.Is` and `errors.AsType[*llm.APIError]`;
  * never regex over message text, which is Pi's approach
    (`ai/src/utils/retry.ts`).

### Standards at every boundary

| Boundary | Standard | Implementation |
|---|---|---|
| Client ↔ agent (editors, mcremote, CLI, TUI) | ACP v1, JSON-RPC 2.0 | `acpserver` / `acpclient` on `coder/acp-go-sdk` |
| Agent ↔ tool servers | MCP (the go-sdk's latest spec revision; Pi accepts `2024-11-05` to `2025-11-25`) | `mcpclient` on `modelcontextprotocol/go-sdk` |
| Tool parameter schemas | JSON Schema 2020-12 | `google/jsonschema-go`, shared with the MCP SDK |
| Tool behaviour hints | MCP `ToolAnnotations` (`readOnlyHint`, `destructiveHint`, `idempotentHint`, `openWorldHint`) | `tool.Annotations`, which also drives the permission defaults |
| Skills | Agent Skills (`SKILL.md`, agentskills.io) | `skill` |
| Project instructions | `AGENTS.md` (also `AGENTS.override.md`; reads `CLAUDE.md`, as Pi does) | `prompt` |
| Directories | XDG Base Directory; Windows Known Folders | `internal/appdirs` |
| Auth to providers and MCP | OAuth 2.1: PKCE (RFC 7636), device grant (RFC 8628), loopback redirect (RFC 8252), metadata (RFC 8414, RFC 9728), dynamic registration (RFC 7591) | `golang.org/x/oauth2`; the MCP SDK's auth support where it has it |
| Observability | OpenTelemetry traces and metrics with GenAI semantic conventions; OTLP set by `OTEL_*` environment variables | `telemetry`; off unless `OTEL_EXPORTER_OTLP_ENDPOINT` or a setting enables it |
| Proxies, colour | `HTTP(S)_PROXY`, `NO_PROXY`, `NO_COLOR` | `http.ProxyFromEnvironment`; Lip Gloss colour profile |
| Versioning, supply chain | SemVer 2.0; GitHub artifact attestations (SLSA provenance); SPDX SBOM | release workflow |

ACP-native behaviour, beyond the method list in 0002:

* **File tools.** When the client advertises `fs.readTextFile` or
  `fs.writeTextFile`, the file tools read and write through the client.
  An editor's unsaved buffer is then the file the model sees. Without that
  capability they fall back to `os.Root`.
* **Bash.** When the client advertises `terminal` and the setting
  `tools.bash.useClientTerminal` is on, `bash` runs in a client terminal
  (`terminal/create`) that the editor displays.
* **Tool calls.** They carry ACP `ToolKind`, `locations` for
  follow-along, and `diff` content for edits.
* **Plans.** The `todo` tool publishes ACP `plan` updates.
* **Sessions.** Session titles go out as `session_info_update`; usage and
  cost as `usage_update`.
* **Model and thinking.** These are `session/set_config_option` options
  with categories `model` and `thought_level`. The SDK cannot route
  `session/set_model` (see Context). 0002's amendment records the
  consequence for the companion table.

### Settings (report D16)

* The file format stays JSON, with the same key names as Pi where the
  meaning is the same, so `config import --from-pi` is a mapping rather
  than a translation. Each file may name a `$schema`, and
  `pigo config schema` prints the JSON Schema generated from the Go
  types. That gives editors completion and validation.
* `internal/config` owns layering and Pi's merge rules (0001-REPORT F15):
  * nested objects merge deeply; arrays and scalars replace;
  * `defaultTools` and resource arrays apply `+name`, `-name`, and
    `!glob` patches;
  * global-only keys are ignored in project files;
  * project files load only for a trusted project.
* Viper binds flags and `PIGO_*` environment variables on top of the
  merged result. Viper does not do the file merge itself.
* Wire and file types use `encoding/json/v2`:
  * `omitzero` on optional fields;
  * a `jsontext.Value` field tagged `json:",unknown"` on every session
    entry and settings object.

  Keys pigo does not know, whether from a newer pigo or from Pi, then
  survive a read-modify-write round-trip.

### Cobra layout (report D15)

* Root `pigo`: with a TTY and no prompt argument it starts the TUI.
  `pigo -p "…"` (or piped stdin) runs print mode.
* Verbs are subcommands (`acp`, `session`, `mcp`, `auth`, `models`,
  `config`, `pkg`, …). The full list is in
  [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md).
* `fang` supplies styled help, errors, `--version`, man pages, and
  completions. `fang` is Charm-styled, so only `internal/cli` imports it.
* Hook-registered and package-registered flags are declared from their
  manifests before `Execute`, so they appear in `--help`. This answers
  0001-REPORT F14 without swallowing unknown flags.

### Idiom list (Go 1.25–1.27 features used on purpose)

| Feature | Use |
|---|---|
| `encoding/json/v2`, `jsontext` (1.27) | wire and file types; `unknown` round-trip; `omitzero`; streaming JSONL decode with `jsontext.Decoder` |
| `uuid.NewV7` (1.27, stdlib) | session ids that sort by time, so file names sort chronologically. Entry ids stay 8-hex for v3 compatibility. |
| `testing/synctest` (`Test` 1.25; `Sleep` 1.27) | deterministic tests of retry back-off, timeouts, cache-warming timers, MCP start-up waits |
| `httptest.NewTestServer` (1.27) | provider and MCP HTTP fakes bound to the test's lifetime |
| `testing.T.ArtifactDir` (1.26) | golden ACP transcripts written on failure for CI upload |
| `errors.AsType` (1.26) | typed provider-error inspection |
| `slog.NewMultiHandler` (1.26) | one logger fanning out to the rolling file, stderr (never in ACP mode), and an OTel bridge |
| `new(expr)` (1.26) | optional pointer fields in ACP and MCP structs without helper functions |
| `os.Root` (1.24; wider API 1.25) | file tools confined to the workspace roots; blocks symlink and `..` escapes |
| `sync.WaitGroup.Go` (1.25) | tool-batch fan-out where `errgroup` semantics are not wanted |
| `runtime/trace.FlightRecorder` (1.25) | always-on ring buffer, dumped by `pigo debug trace` or on a slow-turn threshold |
| `http.CrossOriginProtection` (1.25) | guards every loopback HTTP listener (OAuth callback, `exp/acpws`) |
| `iter`, range-over-func (1.23) | provider streams, session entry walks, `List` results |
| `//go:embed` | built-in prompts, catalog snapshot, JSON Schemas, HTML export template |
| `tool` directives (1.24) | pinned developer tools |
| `go fix` modernizers | CI gate: `go fix -diff ./...` must print nothing |
| PGO (`cmd/pigo/default.pgo`) | added once a representative benchmark profile exists (v1.x) |

### Import-boundary rules (enforced by `internal/archtest`)

1. `agent`, `llm`, `tool`, `session`, `skill`, `hook`, `compaction`,
   `prompt`, `command`, `permission`, `checkpoint`:
   * import no ACP SDK, MCP SDK, vendor LLM SDK, Cobra, Viper, fang,
     Charm, or OpenTelemetry SDK;
   * the OTel **API** is allowed only in `telemetry`.
2. Only `acpserver` and `acpclient/...` import `github.com/coder/acp-go-sdk`.
3. Only `mcpclient` imports `github.com/modelcontextprotocol/go-sdk`.
4. Only `llm/provider/<vendor>` imports that vendor's SDK.
5. Only `internal/tui` imports `charm.land/...`. Only `internal/cli`
   imports Cobra, Viper, or fang.
6. `internal/cli` and `internal/tui` never import `agent`. They drive the
   agent through `acpclient` (0002's rule).
7. No stable or beta package imports `exp/...`. Only `cmd/pigo` wires
   `exp` features, behind settings.

The test runs `go list -deps -json ./...` and fails with the offending
edge. It must be shown failing on a scratch copy with a planted edge
before it is trusted.

### Consequences

* Good, because embedders get a real Go SDK: `pigo.New` plus
  `tool.New[In, Out]` and a custom `llm.Provider` is a complete agent,
  with no CLI or TUI in their dependency graph.
* Good, because every external protocol and SDK sits behind one package.
  Bumping ACP to schema v2 method names, swapping an LLM SDK, or adopting
  `go-llmprovider-sdk` as an adapter each touch one directory.
* Good, because JSON `unknown` round-tripping makes the session and
  settings files forward-compatible with newer pigo and with Pi.
* Good, because the idioms above remove whole test classes of flakiness
  (`synctest`) and whole bug classes (`os.Root`, typed retry
  classification).
* Good, because `exp/` gives codemode, WASM plugins, ACP-over-WebSocket,
  and A2A a place to exist without committing the stable API.
* Neutral, because a single module means one version for everything. A
  breaking change in a stable package needs `v2` of the whole module. The
  tiers and `apidiff` keep that rare.
* Neutral, because the official vendor SDKs add dependency weight
  (anthropic-sdk-go's Bedrock path pulls `aws-sdk-go-v2`; genai pulls
  Google auth). In exchange the wire formats are maintained by the vendors.
* Bad, because the public surface is large for a v0 codebase. Mistakes in
  stable contracts become permanent at `v1.0.0`. Mitigation: everything
  starts as beta, and the release gate promotes packages to stable
  individually.
* Bad, because `go 1.27.1` in `go.mod` forces embedders onto 1.27.1 or
  newer. Accepted for fleet lockstep.

### Confirmation

* `go list -m` shows one module. `go list ./...` matches the package map
  (extras are allowed only under `internal/` and `exp/`).
* `internal/archtest` passes, and it has been shown failing on a scratch
  clone with a planted `internal/cli → agent` edge and a planted
  `agent → charm.land/…` edge.
* `go vet`, `staticcheck` (for three GOOS values), golangci-lint v2,
  `go tool govulncheck ./...`, and `go fix -diff ./...` are all clean.
* `apidiff` runs in CI against the previous tag for packages marked
  stable (from `v1.0.0`).
* Every public package has a `doc.go` stating its tier, and at least one
  runnable `Example` exists for `pigo`, `tool`, and `llm/llmtest`.
* A round-trip test reads a session file with an unknown entry field and
  an unknown entry type, rewrites it, and asserts that both survive
  byte-equivalent after JSON normalisation.

## Pros and Cons of the Options

### One module; contract-first public packages; adapters behind them; `internal/` edge; `exp/` for unstable work

* Good, because it meets every driver with one `go.mod` and one release.
* Good, because contract packages have no heavy dependencies, and
  embedders pay only for the adapters they import (Go prunes unused
  packages, and module graph pruning limits `go.sum` churn).
* Bad, because keeping it disciplined depends on `archtest` and `apidiff`,
  not on convention.

### Binary-only: everything under `internal/`; extract an SDK later

* Good, because it makes no API promise and allows the fastest refactors.
* Bad, because report D4 and the owner's "SDK adherent" and "future
  proofing" ask for an embeddable surface. Extracting one later means an
  import-path change for every early embedder.
* Bad, because nothing forces the contract-versus-adapter separation until
  the extraction, which is where it is most expensive.

### Multi-module repository (`go.work`): separate modules for the SDK, providers, TUI, and CLI

* Good, because each module is versioned independently, and TUI or
  provider dependencies never touch the SDK's `go.sum`.
* Bad, because cross-module changes need coordinated tags, and the fleet
  has no multi-module release tooling (`scripts/bump-libs.sh` works
  repository by repository).
* Bad, because Go's module graph pruning already gives most of the
  dependency isolation at far lower cost.

### `pkg/` + `internal/` + `cmd/` "standard project layout"

* Good, because the pattern is familiar from many repositories.
* Bad, because `pkg/` adds a path element that carries no meaning. The Go
  project's module-layout guidance ("Organizing a Go module") does not use
  it, and neither do the fleet libraries.

### Build on an open-source Go agent framework (Google ADK for Go, CloudWeGo Eino, Genkit Go, LangChainGo)

None of these is in the local module cache. They were not probed. The
assessment rests on their public documentation.

* Good, because they come with ready-made agent loops, tool abstractions,
  and provider plugins.
* Bad, because each owns the event model and the session model. pigo's
  contract is ACP updates and Pi session v3, so the framework's types would
  be translated at every turn.
* Bad, because their extension points (graphs, flows, callbacks) are not
  ACP, MCP, or Agent Skills. Standard-first is the stated driver.
* Neutral, because `llm.Provider` and `tool.Tool` are small enough that an
  adapter to or from such a framework can live in `exp/` later.

## More Information

* Decides report D4 (Go SDK: yes, tiered), D10 (the contract is owned
  here; adapters wrap the official vendor SDKs; `go-llmprovider-sdk`
  becomes an adapter once it publishes a module with streaming), D14
  (official MCP go-sdk), D15, D16, and the toolchain half of D9. 0005
  decides the rest of D9 and the scope-level report decisions.
* Scaffolded by [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md).
  That plan replaces Phase 0 of
  [0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md).
* Versions quoted are those in the local module cache on 2026-09-29. Each
  PLAN phase re-resolves and records the version it pins.
* Cross-repository sources, cited by repository and number:
  * magic-cli-remote MADR 0023 (canonical slash commands), 0157 (rolling
    logs), 0167 (ACP SDK queue fork), 0169 (toolchain policy), 0170
    (preflight, pinned staticcheck);
  * go-llmprovider-sdk MADR 0015 (proposed v1 API);
  * mcplib `selfupdate`;
  * fleet `docs/0005-MADR-github-origin-consolidation.md`.
