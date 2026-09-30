# pi-go documentation

`pi-go` is the repository for **pigo**, the Go rewrite of the Pi agent harness. This tree holds three kinds of document:

- reports that measured something;
- the decisions behind the rewrite;
- guides you follow to do a thing.

- **[architecture.md](architecture.md)** — how the pieces fit together, as they are now.
- **[guides/](guides/)** — task-shaped walkthroughs (none yet).
- **[decisions/](decisions/)** — numbered MADR/PLAN pairs.
- **[reports/](reports/)** — what was observed, deciding nothing.

## Records

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | REPORT | [Go port feasibility of the Pi agent harness](reports/0001-REPORT-go-port-feasibility.md) | observation |
| 0002 | MADR | [v1 native magic-cli-remote CLI: Cobra over ACP, ACP stdio, MCP client](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) | proposed (amended twice) |
| 0002 | PLAN | [Implement the ACP core](decisions/0002-PLAN-cli-acp-headless-mcp-v1.md) | proposed |
| 0003 | MADR | [The product is `pigo`: binary, module path, directories, wire names, read-only Pi bridge](decisions/0003-MADR-pigo-product-identity.md) | proposed |
| 0004 | MADR | [One Go 1.27.1 module of contract-first packages, open standards at every boundary](decisions/0004-MADR-go-module-architecture.md) | proposed |
| 0004 | PLAN | [Scaffold: module, contracts, import boundaries, toolchain, release](decisions/0004-PLAN-go-module-architecture.md) | proposed |
| 0005 | MADR | [The v1 line: every portable Pi capability, tiered 1.0 / 1.x / exp](decisions/0005-MADR-v1-feature-scope.md) | proposed |
| 0005 | PLAN | [Implement the v1.0.0 gate, the v1.x train, and `exp/`](decisions/0005-PLAN-v1-feature-scope.md) | proposed |

0003-MADR has no PLAN of its own. 0004-PLAN Phase 2 and 0005-PLAN F10 implement it.

## Build order

The three plans run in sequence:

1. **[0004-PLAN](decisions/0004-PLAN-go-module-architecture.md)** — the scaffold. It replaces 0002-PLAN Phase 0.
2. **[0002-PLAN](decisions/0002-PLAN-cli-acp-headless-mcp-v1.md) Phases 1–8** — the ACP agent, Cobra as an ACP client, the first tools, sessions, MCP, and native slash commands.
3. **[0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md)** — F1–F10 form the v1.0.0 gate. X1–X6 are the v1.x train and `exp/`.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| know whether a complete Go port is feasible, and what would have to change | [0001-REPORT](reports/0001-REPORT-go-port-feasibility.md) |
| know why ACP is the command API and how pigo sits under mcremote | [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) |
| see how pigo maps onto mcremote slash commands (`/compact`, `/usage`, …) | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (target table), [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) (floor) |
| know why pigo cannot answer `session/set_model` | [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md), second amendment |
| know where pigo keeps config, sessions and credentials | [0003-MADR](decisions/0003-MADR-pigo-product-identity.md) |
| use my existing Pi skills, MCP servers and sessions with pigo | [0003-MADR](decisions/0003-MADR-pigo-product-identity.md) (read-only Pi bridge) |
| see the package map, public SDK tiers and import rules | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) |
| know which Go 1.25–1.27 features pigo uses, and for what | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (idiom list) |
| know which standards each boundary speaks (ACP, MCP, Agent Skills, OTel, OAuth, XDG) | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) |
| check whether a Pi feature made it into v1, and at which tier | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) |
| extend pigo (hooks, MCP servers, skills, packages, Go SDK) | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md), Extensibility |
| see what has to pass before `v1.0.0` is tagged | [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md), F10 |
| start building | [0004-PLAN](decisions/0004-PLAN-go-module-architecture.md) |
| understand what this repository contains today | [architecture.md](architecture.md) |
