# gobble-cli

**gobble** is a planned Go rewrite of the [Pi](https://github.com/earendil-works/pi) agent harness. The planned product is a coding agent with tool calling, sessions, providers, MCP, and a Charm TUI, with the [Agent Client Protocol](https://agentclientprotocol.com) as its command API. The planned runtime binary is `gobble`; the repository is `gobble-cli`, and the planned module path is `github.com/maccavelli/gobble-cli`.

This repository is a greenfield rewrite. It does not share history with the TypeScript monorepo. There is no Go module or binary yet.

**Documentation:** [docs/](docs/README.md)

## Status

The tree currently holds documentation only. The decisions below are
`proposed`; the documentation-only Phase D of 0004-PLAN is complete.

- **[0002-MADR](docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md)** makes one `acp.Agent` the command API.
  - Cobra, the TUI, editors and magic-cli-remote's `mcremote` daemon are all ACP clients of it.
  - `gobble acp` is the headless stdio agent.
- **[0003-MADR](docs/decisions/0003-MADR-gobble-product-identity.md)** fixes the identity.
  - Names: `gobble`, `GOBBLE_*`, `_gobble/` extensions.
  - Directories: XDG, plus a read-only bridge that imports a Pi user's `~/.pi` assets.
- **[0004-MADR](docs/decisions/0004-MADR-go-module-architecture.md)** shapes the module: one Go 1.27.1 module of contract-first packages.
  - The ACP and MCP boundaries use their Go SDKs. The provider boundary uses
    the fleet's [go-llmprovider-sdk](https://github.com/maccavelli/go-llmprovider-sdk)
    through gobble's planned `llm` facade.
  - Import boundaries will be enforced by a test.
  - A planned `exp/` tree holds unstable work.
- **[0005-MADR](docs/decisions/0005-MADR-v1-feature-scope.md)** puts nearly every Pi capability in the v1 line, tiered into a v1.0.0 gate, a v1.x train, and `exp/`.
  - Pi features that exist only as example extensions are built in: subagents, todo/plan, checkpoints with undo, permission gating, and background jobs.

| I want to… | Start here |
| :--- | :--- |
| know whether a complete Go port is feasible, and what would have to change | [0001-REPORT](docs/reports/0001-REPORT-go-port-feasibility.md) |
| know how gobble talks to editors and magic-cli-remote | [0002-MADR](docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md) |
| know what the binary, directories and wire names are | [0003-MADR](docs/decisions/0003-MADR-gobble-product-identity.md) |
| see the package layout, SDK surface and Go 1.27 idioms | [0004-MADR](docs/decisions/0004-MADR-go-module-architecture.md) |
| see everything that is in v1, and at which tier | [0005-MADR](docs/decisions/0005-MADR-v1-feature-scope.md) |
| see the build order | [docs/README.md](docs/README.md#build-order) |
| see what this repository contains today | [architecture.md](docs/architecture.md) |
| see the current sibling-library inventory and integration boundary | [architecture.md](docs/architecture.md#sibling-sources), [0004-MADR](docs/decisions/0004-MADR-go-module-architecture.md), [0005-PLAN](docs/decisions/0005-PLAN-v1-feature-scope.md) |

Stack for v1:

- **CLI and config:** [Cobra](https://github.com/spf13/cobra) with [fang](https://github.com/charmbracelet/fang), and [Viper](https://github.com/spf13/viper).
- **Protocols:** [ACP Go SDK](https://github.com/coder/acp-go-sdk) v0.13.5 (with the fleet `replace` to `github.com/maccavelli/acp-go-sdk v0.13.6-mcr.1`) and the [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk).
- **LLM providers:** [go-llmprovider-sdk](https://github.com/maccavelli/go-llmprovider-sdk) behind gobble's planned `llm` facade. It has no tag yet; observed commit `efd9c61` includes catalog and setup-wizard work, while its adapter-facing contract and usage decoding match the assessed `67fc56e` source. Native streaming is unsupported by its built-in providers. Official Anthropic / OpenAI / Google vendor SDKs are not planned 1.0 dependencies.
- **Self-update:** [go-selfupdate-lib](https://github.com/maccavelli/go-selfupdate-lib) `selfupdate`, formerly go-core-lib, renamed at `v1.5.0` (`6deaa52`). `v1.5.0` exports `selfupdate`, `selfupdate/cli`, `selfupdate/selfupdatetest` and `buildinfo`; the old path `github.com/maccavelli/go-core-lib` ends at `v1.4.1`, deprecated. Opt-in prerelease channels arrived in `v1.3.0`; gobble's planned releases and update selection stay stable-only. F10 decides whether gobble uses the released `buildinfo` or its own planned `internal/buildinfo`. The release workflow publishes the raw binaries that `selfupdate` selects.
- **TUI:** [go-tui-lib](https://github.com/maccavelli/go-tui-lib) `v0.1.0` (`5c57806`) supplies reusable layout, workspace, glyph, theme and test packages on Charm v2. Its workspace has recorded defects; the planned TUI integration waits for a corrected tested tag. Gobble-specific ACP panes remain local to gobble.

These libraries are sibling source trees under active development. **This
repository does not import any of them yet.** The dated source evidence and
integration steps are in [0004-MADR](docs/decisions/0004-MADR-go-module-architecture.md)
and [0005-PLAN](docs/decisions/0005-PLAN-v1-feature-scope.md).

## Source

The TypeScript product lives at [earendil-works/pi](https://github.com/earendil-works/pi). The report and the v1 scope were measured against a clone of that tree at commit `312184edb` (2026-09-29).

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
Third-party notices are in [NOTICE](NOTICE).
