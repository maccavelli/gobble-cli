---
status: proposed
date: 2026-10-03
decision-makers: repository owner
consulted: 0001-REPORT-go-port-feasibility.md, 0002-MADR-cli-acp-headless-mcp-v1.md, 0003-MADR-pigo-product-identity.md
informed: go-llmprovider-sdk, go-core-lib, mcplib, go-tui-lib, magic-cli-remote
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
* **Providers (probed 2026-09-30).** `github.com/maccavelli/go-llmprovider-sdk`
  is a Go 1.27.1 module (`go.mod` present). It depends on the standard
  library and `golang.org/x/term` only (the latter in `wizard`). It has
  **no git tag** and **no LICENSE file**. 0015-MADR (canonical v1 API) is
  `accepted`; 0015-PLAN is `in-progress` in S7. On `origin/main` at
  `3d4aff5f2be9363877aae0987755e6781f1cae98` the v1 contract is in the
  tree and four providers implement it (`openai`, `claude`, `gemini`,
  `grok` under `llmprovider/providers/<id>`). `huggingface`, `kilo`,
  `together`, `ollama`, and the OpenCode gateways still live as the old
  `Generate*` API in `llmprovider/` until S7 finishes. The v1 surface
  pigo consumes:
  * `llmprovider.Provider` (`ID`, `Capabilities`, `Generate`);
  * `llmprovider.Stream(ctx, Provider, *Request) iter.Seq2[Event, error]`,
    which calls a `Streamer` when present and otherwise emits
    `Generate`'s result as `text_delta` / `reasoning_delta` / `item` /
    `done` events;
  * `NativeStreaming` is `Unsupported` on every moved provider
    (0015-PLAN lists native streaming as out of scope);
  * `APIError` with `Kind`, `Status`, `RetryAfter`, and sentinels
    `ErrRateLimited`, `ErrQuotaExhausted`, `ErrAuthFailure`,
    `ErrContextOverflow`, `ErrUnsupported`, `ErrInvalidRequest`,
    `ErrIncomplete`, `ErrNotPermitted`, `ErrProviderUnavailable`;
  * `WithRetry` / `RetryPolicy` (does not retry context overflow or
    exhausted quota);
  * `llmprovider/providers.New(id, opts...)` and `Default()`;
  * `llmprovider/llmtest` (`Run(t, Harness)`, `Fake`);
  * `OAuthSession` and `TokenStore` for subscription auth.
  There is no generic OpenAI-compatible constructor. Chat Completions
  exists only as named providers (`huggingface`, `kilo`, `together`,
  `ollama`). Official vendor SDKs (`anthropic-sdk-go`, `openai-go/v3`,
  `google.golang.org/genai`) are **not** dependencies of that module
  and are **not** 1.0 dependencies of pigo.
* **CLI and TUI.** Cobra v1.10.2, Viper v1.21.0, `charmbracelet/fang`
  v1.0.0, and Charm v2 (`charm.land/bubbletea/v2` v2.0.9, `bubbles/v2`
  v2.2.1, `lipgloss/v2` v2.0.6) are used in production by `ocp-login`.
* **Fleet build conventions** (magic-cli-remote, mcplib, ocp-login):
  * Build: `CGO_ENABLED=0`, `-trimpath`, `-tags netgo,osusergo`, and
    `-ldflags -X main.version/commit/date`.
  * Code quality: `scripts/go-precheck.sh` (gofmt, per-file golint,
    govulncheck), golangci-lint v2 with the shared linter set, and a pinned
    staticcheck run for three GOOS values (magic-cli-remote MADR 0170).
  * CI: `setup-go` with `go-version-file: go.mod`. Dependabot is off by
    policy, and no repository uses goreleaser. pigo publishes through
    go-core-lib's reusable workflow (below), not by copying
    `softprops/action-gh-release` into this tree.
  * Logging: `log/slog` everywhere.
  * Self-update and GitHub-release publication (probed 2026-09-30):
    `github.com/maccavelli/go-core-lib` is tagged `v1.0.0`, `v1.0.1`,
    `v1.1.0`. The current release is **`v1.1.0`** at commit
    `96b30961180671ab3697585951219001ecbb1c90` (Apache-2.0). The live
    package is `selfupdate` (plus `selfupdatetest`). It selects exact
    raw-binary assets named `ExactAssetName(product, platform)` =
    `<product>-<goos>-<goarch>[.exe]`, verifies `SHA256SUMS`, and
    replaces the running binary. It does not import a CLI framework.
    Programs publish through
    `.github/workflows/publish-selfupdate-release.yml` pinned to that
    tag commit. That workflow accepts only a strict
    `vMAJOR.MINOR.PATCH` tag (release candidates do not pass). It does
    not yet ship `appdirs` or `buildinfo`; those stay in pigo.

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
* Every third-party SDK (ACP, MCP, go-llmprovider-sdk, go-core-lib,
  Charm, Cobra, OpenTelemetry) is imported by exactly one package or
  package family, so a replacement or bump is local.
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
llm/                      stable  Message/Content/Usage/Cost/Model types; Provider contract; typed errors.
                                  Agent-facing, not a re-export of go-llmprovider-sdk (that module's
                                  S8 will remove its old API; embedders must not import it).
llm/catalog/              beta    Model catalog: embedded snapshot, models.json overlay, refresh
llm/provider/             beta    The only 1.0 provider implementation: adapter →
                                  github.com/maccavelli/go-llmprovider-sdk/llmprovider
                                  (providers.New, Stream, WithRetry, APIError, TokenStore).
                                  One package: the per-vendor directories existed to isolate
                                  official vendor SDKs, which 1.0 does not import.
llm/llmtest/              stable  Scripted fake provider and stream assertions for embedders' tests.
                                  Event kinds match llm's sealed set; F4 maps llmprovider.Event onto it.
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
internal/auth/                    Keyring + file credential store; implements llmprovider.TokenStore
                                  for provider OAuth. MCP OAuth uses the MCP go-sdk and
                                  golang.org/x/oauth2.
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
* Retry and classification use typed errors, aligned with
  go-llmprovider-sdk so `errors.Is` works after the adapter unwraps:
  * `*llm.APIError{Status, RetryAfter}` wrapping the SDK `APIError`;
  * sentinels `llm.ErrContextOverflow`, `llm.ErrRateLimited`,
    `llm.ErrQuota`, `llm.ErrAuth` matching
    `llmprovider.ErrContextOverflow`, `ErrRateLimited`,
    `ErrQuotaExhausted`, `ErrAuthFailure`;
  * inspected with `errors.Is` and `errors.AsType[*llm.APIError]`;
  * never regex over message text, which is Pi's approach
    (`ai/src/utils/retry.ts`).
* The 1.0 adapter maps `llmprovider.Event` onto the sealed set:
  `text_delta` → `TextDelta`, `reasoning_delta` → `ThinkingDelta`,
  `item` with a function call → `ToolCallDone` (the SDK yields complete
  items, not deltas, while `NativeStreaming` is `Unsupported`),
  `done` plus `Response.Usage` → `Usage` then `Done`.
* Provider construction and retry go through the SDK:
  `providers.New` and `llmprovider.WithRetry`. Context overflow is not
  retried there; compaction-on-overflow is pigo's (`agent` / `compaction`).

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
| Auth to providers | The SDK's OAuth sessions (PKCE, device grant, loopback, refresh, revocation) | `go-llmprovider-sdk` `OAuthSession` / `TokenStore`, persisted by `internal/auth` |
| Auth to MCP | OAuth 2.1: PKCE (RFC 7636), device grant (RFC 8628), loopback redirect (RFC 8252), metadata (RFC 8414, RFC 9728), dynamic registration (RFC 7591) | the MCP go-sdk's auth support where it has it; `golang.org/x/oauth2` otherwise |
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
   * import no ACP SDK, MCP SDK, `go-llmprovider-sdk`, `go-core-lib`,
     vendor LLM SDK, Cobra, Viper, fang, Charm, or OpenTelemetry SDK;
   * the OTel **API** is allowed only in `telemetry`.
2. Only `acpserver` and `acpclient/...` import `github.com/coder/acp-go-sdk`.
3. Only `mcpclient` imports `github.com/modelcontextprotocol/go-sdk`.
4. Only `llm/provider` imports `github.com/maccavelli/go-llmprovider-sdk`.
   No package imports `github.com/anthropics/anthropic-sdk-go`,
   `github.com/openai/openai-go`, or `google.golang.org/genai`.
5. Only `internal/tui` imports `charm.land/...`. Only `internal/cli`
   imports Cobra, Viper, or fang.
6. `internal/cli` and `internal/tui` never import `agent`. They drive the
   agent through `acpclient` (0002's rule).
7. No stable or beta package imports `exp/...`. Only `cmd/pigo` wires
   `exp` features, behind settings.
8. Only `internal/cli` imports `github.com/maccavelli/go-core-lib`.

The test runs `go list -deps -json ./...` and fails with the offending
edge. It must be shown failing on a scratch copy with a planted edge
before it is trusted. The Confirmation's planted edges remain
`internal/cli → agent` and `agent → charm.land/…`. Rules 4 and 8 are
shown failing when those imports first exist (0002-PLAN Phase 3,
0005-PLAN F10).

### Consequences

* Good, because embedders get a real Go SDK: `pigo.New` plus
  `tool.New[In, Out]` and a custom `llm.Provider` is a complete agent,
  with no CLI or TUI in their dependency graph.
* Good, because every external protocol and SDK sits behind one package.
  Bumping ACP to schema v2 method names, or bumping
  `go-llmprovider-sdk`, each touch one directory (`acpserver`/`acpclient`
  and `llm/provider` respectively).
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
* Good, because 1.0's provider dependency is `go-llmprovider-sdk`
  (stdlib + `x/term`). It does not pull `anthropic-sdk-go`,
  `openai-go/v3`, `google.golang.org/genai`, or `aws-sdk-go-v2`.
* Neutral, because pigo then ships the SDK's provider set and its
  current stream shape: `Stream` synthesises events from `Generate`
  until a provider sets `NativeStreaming`. Token-by-token ACP
  `session/update` chunks wait on that SDK capability.
* Neutral, because `go-llmprovider-sdk` has no release tag and no
  LICENSE file as of 2026-09-30. pigo pins a commit pseudo-version
  and records the gap; a later tag replaces the pin.
* Bad, because the public surface is large for a v0 codebase. Mistakes in
  stable contracts become permanent at `v1.0.0`. Mitigation: everything
  starts as beta, and the release gate promotes packages to stable
  individually.
* Bad, because `go 1.27.1` in `go.mod` forces embedders onto 1.27.1 or
  newer. Accepted for fleet lockstep.

### Confirmation

* `go list -m` shows one module. `go list ./...` matches the package map
  (extras are allowed only under `internal/` and `exp/`).
  `go list -m` does not name `anthropic-sdk-go`, `openai-go`, or
  `google.golang.org/genai`. After F4 it names
  `github.com/maccavelli/go-llmprovider-sdk`. After F10 it names
  `github.com/maccavelli/go-core-lib`.
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
  here; the 1.0 implementation is an adapter over `go-llmprovider-sdk`;
  official vendor LLM SDKs are not 1.0 dependencies), D14 (official MCP
  go-sdk), D15, D16, and the toolchain half of D9 (release assets and
  the publish workflow follow `go-core-lib` `v1.1.0`). 0005 decides
  `pigo update` and the rest of D9.
* Scaffolded by [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md).
  That plan replaces Phase 0 of
  [0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md).
* Versions quoted were re-probed on 2026-09-30. Each PLAN phase
  re-resolves and records the version it pins.
* Cross-repository sources, cited by repository and number:
  * magic-cli-remote MADR 0023 (canonical slash commands), 0157 (rolling
    logs), 0167 (ACP SDK queue fork), 0169 (toolchain policy), 0170
    (preflight, pinned staticcheck);
  * go-llmprovider-sdk 0015-MADR (accepted v1 API) and 0015-PLAN
    (in-progress at S7 on 2026-09-30);
  * go-core-lib `v1.1.0` (`selfupdate`, reusable publish workflow);
  * fleet `docs/0005-MADR-github-origin-consolidation.md`.

### Amendment (2026-09-30): consume the shared libraries as they exist

Still `proposed`. The owner directed that `go-llmprovider-sdk` and
`go-core-lib` are in v1 and can be used, and that the records follow
the consumption model below. The module shape, the `llm` facade, the
import-boundary idea, and ACP-as-command-API are unchanged.

1. **D10.** 1.0 talks to models only through
   `github.com/maccavelli/go-llmprovider-sdk`. `llm` stays the small
   stable embedder surface (Pi-shaped messages, sealed events, typed
   errors) so embedders do not import that module and so S8's old-API
   removal does not break them. `llm/provider` is the single adapter
   package. The `llm/provider/{anthropic,openai,google,compat}`
   directories, and the 1.0 `require` of `anthropic-sdk-go` /
   `openai-go/v3` / `genai`, are withdrawn. Unnamed OpenAI-compatible
   presets (Groq, Cerebras, DeepSeek, OpenRouter, Fireworks, Mistral,
   LM Studio, llama.cpp, vLLM) wait on an SDK Chat Completions
   provider that takes a base URL, or on those ids being added there;
   they are 1.x in 0005. Named SDK ids that have not yet implemented
   `llmprovider.Provider` (S7 remaining: OpenCode, HuggingFace, Kilo,
   Together, Ollama) join 1.0 as that work lands in the SDK; pigo does
   not wrap the old `Generate*` API.
2. **Streaming.** Call `llmprovider.Stream` so a later native
   `Streamer` is picked up without an API change. Do not claim
   token-by-token ACP chunks in 1.0: every moved provider sets
   `NativeStreaming` to `Unsupported`.
3. **D9 (toolchain half).** Release assets are the raw binaries
   `selfupdate.ExactAssetName` selects, plus `SHA256SUMS`. Publication
   is the go-core-lib reusable workflow pinned to `v1.1.0`
   (`96b30961180671ab3697585951219001ecbb1c90`). `pigo update` is 0005.
   `internal/appdirs` and `internal/buildinfo` stay in this module.
4. **Pins.** `go-core-lib v1.1.0`. `go-llmprovider-sdk` has no tag:
   the first import (0002-PLAN Phase 3) records a commit
   pseudo-version of `origin/main` and is re-resolved at 0005-PLAN F4.
   No relative `replace`. The ACP `replace` to
   `github.com/maccavelli/acp-go-sdk v0.13.6-mcr.1` is the Phase 0
   choice (this record already recommended it).
5. **NOTICE.** `go-core-lib` is Apache-2.0; the pigo `NOTICE` names it.
   `go-llmprovider-sdk` has no LICENSE in tree; do not invent one.

### Amendment (2026-10-01): Apache-2.0

Still `proposed`. The owner directed Apache-2.0 for this repository,
`go-llmprovider-sdk`, and `go-core-lib`. 0003-MADR records the pigo
licence. `LICENSE` and `NOTICE` land ahead of 0004-PLAN Phase 0.
`NOTICE` names Pi (MIT), `go-core-lib` (Apache-2.0), and
`go-llmprovider-sdk` (Apache-2.0, that repository's
`0018-MADR-apache-2-license.md`). The 2026-09-30 "do not invent" line
above is closed. The 2026-09-30 probe that the SDK had no LICENSE
file remains that day's fact.

### Amendment (2026-10-01, second): the shared libraries measured in code

Still `proposed`. Three sibling trees were re-read and built in scratch copies:

* go-llmprovider-sdk at `940fee0`, pushed 2026-10-01 15:09 -0500 ("extract shared wire format packages");
* go-core-lib at `v1.2.0` and at its working tree;
* the acp-go-sdk fork at `v0.13.6-mcr.1`.

`go build`, `go vet` and `go test ./...` are green at `940fee0`. These are unchanged:

* the module shape;
* the package map;
* the boundary rules (rule 8 is refined below);
* the decision that 1.0 talks to models only through go-llmprovider-sdk.

#### go-llmprovider-sdk: facts as of `940fee0`

* **All 10 provider ids implement `llmprovider.Provider`.**
  * The ids are `openai`, `claude`, `gemini`, `grok`, `opencode-zen`, `opencode-go`, `kilo`, `huggingface`, `together` and `ollama`, in their own packages. 0015-PLAN S7 is finished.
  * The Context's "four providers … the rest still old `Generate*`" was true at `3d4aff5` only.
  * S7b (wire and transport extraction) is in flight: commit 1 is `940fee0`, and commit 2 is uncommitted and did not build at 15:11.
* **No tag.** `LICENSE` (Apache-2.0) landed in `8150c35`.
  * The SDK's own 0015-PLAN says: "Intermediate phases may break the API. No consumer imports this module before v1.0.0-rc.1."
  * A `v0.0.0-…` pseudo-version also makes the ChatGPT backend send `client_version` 0.0.0, which hides some models (`discovery.go:21-41`).
* **No wire decodes usage.** No non-test code sets `Response.Usage` or `Response.Model`, so `Usage` is always zero. That is S9 (not started).
  * `Usage` has a single `CachedTokens` counter, with no read/write split.
  * `FinishReason` is set only by Chat Completions.
* **Content gaps.**
  * Items are text, function call, function output and reasoning only (`item.go:15-55`).
  * There is no image input and no `is_error` on tool output.
  * There is no opaque reasoning payload: Claude's thinking `signature` and `redacted_thinking` are dropped (`messages.go:88-142`), and Responses drops `ReasoningItem` on input.
  * Gemini's thought signature does round-trip (`FunctionCallItem.Signature`).
* **No prompt caching.** The only cache field is ChatGPT's `prompt_cache_key`. Claude's system prompt is a single string.
* **Errors.**
  * A plain 429 returns `*RateLimitError`, not `*APIError` (`api_error.go:201-203,226`).
  * A truncated response is `*IncompleteError`.
  * Both fold into `APIError` at S8.
  * `ErrQuotaExhausted` also matches `ErrRateLimited`, and `ErrContextOverflow` also matches `ErrInvalidRequest`.
* **Retry.**
  * `WithRetry` defaults to 3 attempts, a 1 s base and a 30 s cap, with up to +25% jitter. A `Retry-After` above the cap returns at once.
  * **The wrapper hides any `Streamer`** (`retry.go:43`).
* **Defaults pigo must override.**
  * Billed model probes are **on** (up to 6 generations, `options.go:181-190`).
  * `MaxTokens` is 8192.
  * The default client has a 300 s `ResponseHeaderTimeout`.
  * Non-streamed bodies are capped at 1 MiB.
  * The SDK reads `CLAUDE_API_KEY`, not `ANTHROPIC_API_KEY` (`provider.go:180`). Its own 0016-MADR D12 chose the latter.
* **Reasoning effort** accepts only `low|medium|high|xhigh` (`contract.go:52-71`). Explicit budgets are honoured only by Claude before 4.7 and by OpenCode's Google route.
* **Auth.**
  * Implemented:
    * `OAuthSession`, `TokenStore` and `RefreshLocker` (a cross-process refresh lock);
    * `FileTokenStore`;
    * `LoginBrowserOAuth` (PKCE loopback) and `StartDeviceOAuth` (a `DeviceLogin` with `Wait`/`Cancel`);
    * `RevokeOAuthSession`, `CommandToken`, `VendorCLISession`.
  * Issuers are OpenAI/ChatGPT and xAI Grok, plus Kilo device login.
  * Claude, Gemini and Together refuse OAuth sessions. SDK 0016-MADR D10 declines Anthropic and Gemini subscription OAuth, and treats Copilot as impersonation-only.
  * **S8c moves these into `llmprovider/auth`.**
* **Model metadata.** It exposes `ReasoningEfforts` only; context window, cost and modalities are not public. pigo's own catalog remains mandatory.
* **Ids.**
  * SDK ids differ from Pi's: `claude`↔`anthropic`, `gemini`↔`google`, `grok`↔`xai`. Codex is a credential mode of `openai`.
  * `WithSessionID` is fixed per provider instance.

#### Decisions changed by these facts

1. **Pin policy.**
   * pigo may *develop* against a recorded SDK commit.
   * It **tags no release** on a pseudo-version of go-llmprovider-sdk. The first pigo tag requires an SDK `v1.0.0-rc.N` or later.
   * The owner requests that tag from the SDK once S8c (final import paths) has landed, and preferably S9 (usage).
   * Until then, 0002-PLAN Phase 3 and 0005-PLAN F4 record each commit they build against, with the API breaks still to come.
2. **The `llm` facade is wider than the SDK, on purpose.** The sealed set becomes:
   * `TextDelta`, `ThinkingDelta`;
   * `ToolCallStart`, `ToolCallDelta`, `ToolCallDone`;
   * `Usage`;
   * `Done{StopReason}`.

   Changes to the types:
   * `llm.Usage` carries input, output, reasoning, cache-read and cache-write tokens, plus `Estimated bool`.
   * `llm.Content` carries images.
   * Thinking blocks and tool calls carry an opaque `ProviderData`. It is persisted as Pi's v3 fields `thinkingSignature`, `thoughtSignature`, `textSignature` and `redacted`, which those field names already reserve.

   The adapter fills what the SDK provides. Until the SDK carries a feature, the facade's `Capabilities` says so, and pigo advertises nothing it cannot send. In particular, ACP `promptCapabilities.image` stays **false** until SDK image input exists.
3. **Usage is estimated until S9.**
   * `llm/provider` estimates input and output tokens (a byte-based estimator per model family, recorded in the catalog) and sets `Estimated`.
   * The usage ledger, `usage_update`, auto-compaction and cost all consume the estimate.
   * `/usage` labels it "estimated".
   * When the SDK reports real usage, the adapter prefers it, with no API change.
4. **Error mapping order.**
   * Map `ErrQuotaExhausted` before `ErrRateLimited`, and `ErrContextOverflow` before `ErrInvalidRequest`.
   * Handle `*RateLimitError` and `*IncompleteError` until S8.
   * `llm` adds `ErrIncomplete`, `ErrNotPermitted`, `ErrUnavailable` and `ErrUnsupported`.
5. **Retry.**
   * While no provider streams natively, `llm/provider` wraps `WithRetry` with an explicit `RetryPolicy{MaxAttempts: 4, BaseDelay: 2 s, MaxDelay: 60 s}`, which is Pi's 3 retries and 60 s cap.
   * Tests assert bounds, not exact steps, because of the jitter.
   * When any provider gains a `Streamer`, retry moves into the adapter. It retries only before the first event, so `WithRetry` cannot hide the stream.
6. **Adapter configuration.**
   * Always `WithModelProbes(false)`.
   * `MaxOutputTokens` per model from `llm/catalog`.
   * One provider instance per ACP session, so `WithSessionID` (ChatGPT `prompt_cache_key`, OpenCode/Kilo session headers) is per session.
   * pigo supplies its own `*http.Client`: `ProxyFromEnvironment`, and no response-header timeout shorter than the model's generation limit. A non-streamed long generation must not time out and be re-billed by retry.
   * Credentials are passed explicitly (`WithAPIKey` / `WithTokenSource`). pigo never depends on the SDK reading environment variables, which S10 removes anyway. pigo reads Pi's names, such as `ANTHROPIC_API_KEY`.
7. **Thinking-level map** (`llm/catalog`):

   | pigo level | Effort sent | Note |
   |---|---|---|
   | `off` | no `Reasoning` | |
   | `minimal` | `low` | recorded as a degradation |
   | `low`, `medium`, `high`, `xhigh` | the same value | |
   | `max` | `xhigh` | recorded as a degradation |

   Budgets are sent only where the SDK honours them.
8. **Provider-id map.**
   * `llm/catalog` owns a Pi↔SDK id table: `anthropic`→`claude`, `google`→`gemini`, `xai`→`grok`, and `openai-codex`/`openai-chatgpt`→`openai` with the subscription credential.
   * `models.json`, `config import --from-pi` and the bridge use it.
   * An unmapped Pi provider fails closed with a clear error.
9. **Token store.**
   * `internal/auth` implements `TokenStore` **and** `RefreshLocker`, with a file lock in the state directory. The TUI, `pigo acp` children and the CLI share credentials, and a refresh token spent twice revokes the whole token family.
   * The import path follows S8c (`llmprovider/auth`). Archtest rule 4 is unaffected.
10. **Tool schemas.** `llm/provider` sanitizes tool JSON Schema for the Gemini wire, as Pi does. This is an inference from Pi's behaviour, not a probed Gemini rejection. The PLAN records a live probe before relying on it.

#### acp-go-sdk fork: overflow policy (corrects the 2026-09-30 recommendation)

* The fork's facts:
  * It adds `ConnectionOption`, `WithMaxQueuedNotifications`, `WithNotificationOverflowPolicy`, `WithNotificationDropHandler` and `DroppedNotifications`, in 5 files (+570/−8).
  * Its generated code and `extensions.go` are byte-identical to v0.13.5.
  * The policy governs **inbound** notifications only, and defaults to `OverflowCloseConnection`.
* pigo carries the `replace`, because pigo's in-process clients need the option API.
* `acpserver` uses the default policy. Its inbound notifications are `session/cancel`, and dropping one would lose a cancel.
* `acpclient` sets `OverflowDropNewest` with a drop handler that surfaces a notice. That client serves the CLI, the TUI, `task` subagents and `acptest`.
* The exit plan stays magic-cli-remote `0167-MADR-the-acp-sdk-is-dormant-and-its-bounded-queue-is-an-availability-defect.md`.

#### go-core-lib: facts as of `v1.2.0`

* **Tags.** `v1.2.0` = `cfc95c883220b013c21705e1ebe3a268bdd63b56`, tagged 2026-10-01 10:24 -0500. It is additive:
  * `RunWith` and `Start`/`Stream` (a non-blocking interaction stream suited to the TUI);
  * `WithCredentials`.

  The publish workflow is byte-identical to `v1.1.0`. `v1.1.0` remains valid.
* **Prerelease channels are planned for `v1.3.0`.** The accepted MADR is go-core-lib `0005-MADR-opt-in-prerelease-channels.md`; Step 2 is uncommitted.
* **Only `selfupdate` and `selfupdate/selfupdatetest` exist.** go-core-lib's 0004-MADR plans `buildinfo`, `selfupdate/cli` and `selfupdate.UserAgent`.
* **The pin rule.**
  * At 0005-PLAN F10, re-resolve to the newest `v1.x` tag.
  * The `go.mod` require and the workflow `uses:` line carry the **same** tag's peeled SHA.
  * "Current release `v1.1.0`" in the Context is that day's fact.
* **`internal/buildinfo` follows go-core-lib's planned rule.** A build is `ReleaseBuild` only when it is stamped `buildKind=release` **and** its version is a strict `vX.Y.Z`. The `debug.ReadBuildInfo` fallback is display-only. A `go install …@vX` build is therefore a local build: `pigo update` needs `--force` to replace it, and `--check` reports it as replaceable.
* **Archtest rule 8, refined.** Only `internal/cli` imports `github.com/maccavelli/go-core-lib` in non-test code. `_test.go` files in any package may import `selfupdate/selfupdatetest`.
* **Supply chain.**
  * The reusable workflow attests `staging/*` and waits for an immutable release.
  * It does **not** generate or attest an SBOM.
  * The client verifies `SHA256SUMS` and GitHub digests, not signatures or attestations.
  * The standards table row "SPDX SBOM" is pigo's own job: 0004-PLAN Phase 4 names and pins the generator. The SBOM is an extra asset and is **not** listed in `SHA256SUMS`, because the verifier requires that file to list exactly the binaries.

#### Corrections to wording

* **go-llmprovider-sdk is the fleet's in-house SDK, not an official vendor SDK.** The README's "official SDK at each protocol boundary" applies to ACP and MCP.
* **magic-cli-remote `0169-MADR-standardize-toolchains-on-current-supported-advisory-free-releases.md` is `proposed`.** "Fleet MADR 0169 policy" reads "magic-cli-remote 0169 (proposed)".
* **`models.json` is JSONC in Pi.** `internal/config` strips comments before `encoding/json/v2` decodes it. `settings.json` and `mcp.json` stay strict JSON.

### Amendment (2026-10-02): current sibling source and a shared TUI workspace

Still `proposed`. The observations below are from the local sibling source
trees, not a claim that pigo imports them yet. `go-llmprovider-sdk` is at
`67fc56e3df785e4113f664f39f398031c41b1275` (its PLAN has an uncommitted
edit), `go-core-lib` has tag `v1.3.0` at
`f97c6817ec9c902d1af690198ed0e2e2abb7750c`, and `go-tui-lib` has tag
`v0.1.0` at `5c578065764f5fe569474819286f0d67e99ffdcd`. The latter's
current HEAD changes documentation only; its released Go source is unchanged.

#### Provider SDK

* All ten built-in ids are registered by `llmprovider/providers.Default` and
  constructed by `providers.New` (`providers/providers.go`). The old
  `Generate*` API has been removed. `Provider.ID` returns `ProviderID`, a
  named string type (`contract.go`), so the adapter converts ids explicitly.
* S8c moved OAuth sessions, stores and login helpers to `llmprovider/auth`.
  S9 now decodes `Response.Usage` in Responses, Messages, generateContent and
  Chat Completions (`internal/wire/*`). The earlier assertion that usage is
  always zero is obsolete. The public `Usage` has input, output, reasoning
  and cached token counts, but no cache-write count or measured/estimated
  flag; a zero count can still mean the service reported none. Because the
  SDK does not expose presence bits, pigo treats an all-zero token usage as
  unavailable and estimates the turn, marking its own `llm.Usage.Estimated`.
  Any nonzero response usage is taken as reported; a zero component within
  it is left at zero rather than invented. Cache-write remains unknown.
* S8 unified structured failures as `*APIError` (`api_error.go`). The
  adapter no longer needs branches for `*RateLimitError` or
  `*IncompleteError`; it still tests sentinel precedence because quota
  matches rate limit and overflow matches invalid request.
* S10 made ambient configuration opt-in. Provider constructors do not read
  API-key environment variables. `ProviderEnvVars` lists their names, and
  `ModelProbesFromEnv`, `catalog.OptionsFromEnv` and auth's
  `GrokFlowFromEnv` read environment only when the caller selects them.
  pigo keeps explicit credential and catalog configuration and does not
  pass those opt-in helpers implicitly.
* `LICENSE` is Apache-2.0. There is still no git tag. The existing rule
  remains: development may pin a recorded commit, while a pigo release
  waits for an SDK `v1.0.0-rc.N` or later. `NativeStreaming` is still
  `Unsupported` on all built-ins, and `Item` still has no image input,
  tool-output error bit or opaque thinking payload. The adapter must keep
  advertising those limits until the SDK code changes.

#### Self-update library

* `v1.3.0` adds opt-in prerelease channels: `Request.Channel`,
  `CheckRequest.Channel`, `NewSemverPolicy` and a channel input in the
  reusable publish workflow. The stable path remains the default; a
  prerelease is not selected without a configured channel
  (`selfupdate/types.go`, `checker.go`, `version.go`, and
  `.github/workflows/publish-selfupdate-release.yml`).
* `selfupdate` and `selfupdate/selfupdatetest` remain the only Go packages.
  `buildinfo`, `selfupdate/cli` and `updatetea` are not present. pigo retains
  `internal/buildinfo` and binds the update flags and TUI interactions
  itself. The F10 pin rule selects the newest suitable `v1.x` tag when
  implementation starts and pins the workflow to that tag's peeled commit.
  The pigo stable-only release decision remains in force; adopting
  prerelease channels would need a later decision and plan amendment.

#### TUI library

* `go-tui-lib v0.1.0` provides `layout`, `workspace`, `glyph`, `theme` and
  `tuitest`. `layout.Solve` is a pure solver; `workspace.New` hosts caller
  panes in Bubble Tea. It has no transcript renderer, picker, permission
  dialog, editor or update component. Its module uses Charm v2 and Go
  1.27.1; it does not yet require `go-core-lib` (`go.mod`).
* The sibling's `0002-PLAN-harden-workspace-v0-1-1.md` and
  `0004-PLAN-integrate-charm-v2-and-go-1-27.md` are proposed. Their fixes
  and later `stream`, `command`, `keymap`, `palette`, `termcap` and `termsvc`
  packages are not current API. The `v0.1.0` workspace has recorded
  defects, so an F9 integration must be tested against a corrected tagged
  version before depending on it.
* **Decision:** `internal/tui` consumes `go-tui-lib` for released reusable
  workspace, layout, glyph and theme behaviour after that validation. It
  keeps pigo-specific ACP panes and flows locally. Only `internal/tui`
  imports `go-tui-lib`, as it alone imports Charm; the core and public Go
  SDK remain TUI-free. `tuitest` may be imported by `_test.go` files for
  rendering checks. This extends import-boundary rule 5 without changing
  the package map or making the TUI library a prerequisite for the module
  scaffold.

### Source correction (2026-10-02): go-core-lib release and development HEAD

The inventory above was pinned to `go-core-lib v1.3.0`. Its newer `v1.3.1`
tag peels to `351bd6a5ffbd4b352cacc5234f4a8c504c3c1b69` and still has
only `selfupdate` and `selfupdate/selfupdatetest` as Go packages. At observed
development commit `bf7221a5408984299a8511eb98f8f06eadd133bd`, a
`buildinfo` package exists. It exposes `Identity` and `LDFlags`; it is
not in `v1.3.1` and is not a pigo dependency. `selfupdate/cli` is still a
plan. The local `internal/buildinfo` choice remains for the scaffold; its
relationship to a released go-core-lib `buildinfo` must be checked when the
first relevant dependency is selected. No pigo module or import exists yet.

The provider SDK first advanced to `d4ca92575855c1ba9196422eeeb34df2ad5c9f6a`
with a plan-only change. Observed commit
`efd9c615c85f7a85d27e3c636b2fbb641adeab5e` then added catalog listing
and setup-wizard handling for providers outside the built-in catalog, plus
API enforcement tooling. Its `Provider`, `Response`, `Usage`, `APIError` and
auth contracts assessed at `67fc56e3df785e4113f664f39f398031c41b1275`
are unchanged. It still has no tag; these commits are not pigo dependencies.

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
* **What it means for this record.** Where this MADR names
  `github.com/maccavelli/go-core-lib` as a planned dependency, the module
  pigo will require is `github.com/maccavelli/go-selfupdate-lib`, at
  `v1.5.0` or later. That covers import-boundary rule 8, the pins, the
  release workflow and `NOTICE`. The old path gets no further releases.
  pigo requires neither module yet.
* **Still open, as before.** F10 checks, when it selects a tag, whether
  pigo keeps its own `internal/buildinfo` or uses the released `buildinfo`
  (0005-PLAN F10). This correction does not decide it.
* **The earlier text stays as written**, as dated history.
