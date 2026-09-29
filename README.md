# pi-go

A Go rewrite of the [Pi](https://github.com/earendil-works/pi) agent harness: a terminal coding agent with tool calling, sessions, providers, and a Charm-based TUI.

This repository is a greenfield rewrite. It does not share history with the TypeScript monorepo. There is no Go module or binary yet.

**Documentation:** [docs/](docs/README.md)

## Status

The tree currently holds documentation only. Proposed v1 ([0002-MADR](docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md)) is the native [magic-cli-remote](https://github.com/maccavelli/magic-cli-remote) CLI: a Cobra process whose agent command API is [ACP](https://agentclientprotocol.com), a headless ACP stdio server that mcremote already knows how to spawn, and an MCP client. Agent slash commands (`/compact`, `/usage`, `/context`, …) execute over ACP. Charm TUI is out of that cut.

| I want to… | Start here |
| :--- | :--- |
| know whether a complete Go port is feasible, and what would have to change | [0001-REPORT](docs/reports/0001-REPORT-go-port-feasibility.md) |
| know what v1 is | [0002-MADR](docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md) |
| see the v1 implementation phases | [0002-PLAN](docs/decisions/0002-PLAN-cli-acp-headless-mcp-v1.md) |
| see what this repository contains today | [architecture.md](docs/architecture.md) |

Stack for v1: [Cobra](https://github.com/spf13/cobra), [Viper](https://github.com/spf13/viper), [ACP Go SDK](https://github.com/coder/acp-go-sdk) v0.13.5. Charm (Lip Gloss and related packages) is the TUI stack when a TUI exists.

## Source

The TypeScript product lives at [earendil-works/pi](https://github.com/earendil-works/pi). The report measured a clone of that tree at commit `312184edb` (2026-09-29).
