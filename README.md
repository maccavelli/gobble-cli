# gobble-cli

**gobble** is a coding agent written in Go: a terminal CLI, an [Agent Client Protocol](https://agentclientprotocol.com)
agent for editors, and an embeddable Go SDK, in one pure-Go binary. The repository is `gobble-cli`; the
binary is `gobble`; the module is `github.com/maccavelli/gobble-cli`.

gobble is its own design. It takes the best of what other agent harnesses have already worked out —
[Pi](https://github.com/earendil-works/pi), [opencode](https://github.com/anomalyco/opencode),
[Kilo](https://github.com/Kilo-Org/kilocode), [goose](https://github.com/aaif-goose/goose),
[codex](https://github.com/openai/codex) and [grok-build](https://github.com/xai-org/grok-build) —
and makes each piece gobble's own, in its own words and to its own rules. Nothing is copied for its
own sake. Compatibility is kept only where it serves gobble's users: a Pi user's sessions, settings,
skills and MCP servers carry over through a read-only bridge.

**Documentation:** [docs/](docs/README.md)

## What it draws from

Each choice is argued, with source evidence, in the records:

| From | What gobble takes | Where it is decided |
| :--- | :--- | :--- |
| Pi | The baseline capability list, the tool shapes, session JSONL compatibility, and the read-only bridge for `~/.pi` assets | [0003-MADR](docs/decisions/0003-MADR-gobble-product-identity.md), [0005-MADR](docs/decisions/0005-MADR-v1-feature-scope.md) |
| opencode | Layered edit matching, numbered reads, last-match-wins permission rules, and the compaction template | [0010-REPORT](docs/reports/0010-REPORT-harness-design-survey.md); 0005-MADR, amendment of 2026-10-07 |
| Kilo | Hard permission ceilings, protected paths, loop circuit breakers, and notebook and document handling | the same |
| goose | Terminal-CLI mechanics, `tree`, and document extraction | [0006-MADR](docs/decisions/0006-MADR-goose-cli-port-candidates.md), [0011-REPORT](docs/reports/0011-REPORT-native-tools-survey.md) |
| codex | `apply_patch`, process sessions, `request_permissions`, and deferred tools behind a BM25 `tool_search` | 0011-REPORT; 0005-MADR, amendment of 2026-10-08 |
| grok-build | Gitignore-aware `tree` with an output budget, the `lsp` tool, `monitor`, and completion reminders | the same |

What is gobble's own: one ACP agent as the command API for every client, native file operations that
the permission rules and undo can see, a two-tier tool exposure that keeps the prompt cache stable, and
error messages written in one style throughout
([0012-MADR](docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md), accepted).

## Status

Early, and built in the order [docs/README.md](docs/README.md#build-order) gives. What runs today:

- **`gobble`**, the native CLI mode: a line session in a terminal, or one prompt with `-p` or piped input.
- **`gobble acp`**: the ACP agent on stdio, for editors and magic-cli-remote. It has sessions, slash
  commands, modes, manual compaction, MCP servers and gobble's `_gobble/` extension methods.
- **Tools:** `read`, `write` and `edit`, confined to the workspace roots; `grep`, `find` and `tree`,
  which follow the repository's `.gitignore` rules; `move`, `delete`, `copy` and `mkdir`, where every
  delete asks first; `todo`, a checklist the client shows as the plan; `bash`, and `powershell` on
  Windows, run as process trees that stop whole.
- **Sessions** as JSONL files in Pi's v3 shapes.
- **`gobble mcp add|remove|list`**, `gobble config path`, `gobble version` and `gobble completion`.

Not built yet: `tool_search`, the MCP resource tools and the client's file and terminal methods
(F1d), the permission engine (F6), the enhanced TUI mode, and `gobble update`. The full v1 line, tiered into
a v1.0.0 gate, a v1.x train and `exp/`, is in [0005-MADR](docs/decisions/0005-MADR-v1-feature-scope.md).

| I want to… | Start here |
| :--- | :--- |
| see what this repository contains today | [architecture.md](docs/architecture.md) |
| see everything in the v1 line, and at which tier | [0005-MADR](docs/decisions/0005-MADR-v1-feature-scope.md) |
| see how other harnesses design what gobble builds, and which design gobble chose | [0010-REPORT](docs/reports/0010-REPORT-harness-design-survey.md), [0011-REPORT](docs/reports/0011-REPORT-native-tools-survey.md) |
| know how gobble talks to editors and magic-cli-remote | [0002-MADR](docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md) |
| know what the binary, directories and wire names are | [0003-MADR](docs/decisions/0003-MADR-gobble-product-identity.md) |
| see the package layout, SDK surface and Go idioms | [0004-MADR](docs/decisions/0004-MADR-go-module-architecture.md) |
| see the build order | [docs/README.md](docs/README.md#build-order) |
| build gobble, run the gates, or add a package | [developing.md](docs/guides/developing.md) |
| know where the project started: the Go port feasibility study of Pi | [0001-REPORT](docs/reports/0001-REPORT-go-port-feasibility.md) |

## Stack

- **Modes:** a native terminal CLI mode, the default, and an enhanced terminal TUI mode, not built yet.
- **CLI:** [Kong](https://github.com/alecthomas/kong) `v1.16.1`.
- **Protocols:** the [ACP Go SDK](https://github.com/coder/acp-go-sdk) `v0.13.5`, replaced by the
  fleet's fork at `v0.13.6-mcr.2`, and the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) `v1.8.0`.
- **Models:** [go-llmprovider-sdk](https://github.com/maccavelli/go-llmprovider-sdk) `v1.2.1`, behind
  gobble's own `llm` facade. Only `llm/provider` imports it.
- **Tool schemas:** JSON Schema 2020-12 from Go types, with [jsonschema-go](https://github.com/google/jsonschema-go) `v0.4.3`.
- **Build identity and self-update:** [go-selfupdate-lib](https://github.com/maccavelli/go-selfupdate-lib)
  `v1.9.0`. gobble does not reimplement self-update.
- **TUI:** [go-tui-lib](https://github.com/maccavelli/go-tui-lib), not required yet. gobble does not
  reimplement core TUI; where a behaviour is not in the library yet, gobble waits rather than copying it.

Shipped binaries are pure Go (`CGO_ENABLED=0`).

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
Third-party notices are in [NOTICE](NOTICE).
