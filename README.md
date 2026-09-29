# pi-go

A Go rewrite of the [Pi](https://github.com/earendil-works/pi) agent harness: a terminal coding agent with tool calling, sessions, providers, and a Charm-based TUI.

This repository is a greenfield rewrite. It does not share history with the TypeScript monorepo. The first record is a feasibility report; there is no Go module or binary yet.

**Documentation:** [docs/](docs/README.md)

## Status

The tree currently holds documentation only.

| I want to… | Start here |
| :--- | :--- |
| know whether a complete Go port is feasible, and what would have to change | [0001-REPORT](docs/reports/0001-REPORT-go-port-feasibility.md) |
| see what this repository contains today | [architecture.md](docs/architecture.md) |

Stated stack for the eventual binary: [Cobra](https://github.com/spf13/cobra) and [Viper](https://github.com/spf13/viper) for the CLI and configuration; [Charm](https://github.com/charmbracelet) (Lip Gloss and the rest of that stack) for terminal UI.

## Source

The TypeScript product lives at [earendil-works/pi](https://github.com/earendil-works/pi). The report measured a clone of that tree at commit `312184edb` (2026-09-29).
