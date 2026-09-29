# pi-go documentation

`pi-go` is the Go rewrite of the Pi agent harness. This tree holds the reports that measured something, the decisions behind the rewrite, and the guides you follow to do a thing.

- **[architecture.md](architecture.md)** — how the pieces fit together, as they are now.
- **[guides/](guides/)** — task-shaped walkthroughs (none yet).
- **[decisions/](decisions/)** — numbered MADR/PLAN pairs.
- **[reports/](reports/)** — what was observed, deciding nothing.

## Records

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | REPORT | [Go port feasibility of the Pi agent harness](reports/0001-REPORT-go-port-feasibility.md) | observation |
| 0002 | MADR | [v1 native magic-cli-remote CLI: Cobra over ACP, ACP stdio, MCP client](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) | proposed |
| 0002 | PLAN | [Implement that v1 cut](decisions/0002-PLAN-cli-acp-headless-mcp-v1.md) | proposed |

## I want to…

| I want to… | Start here |
| :--- | :--- |
| know whether a complete Go port is feasible, and what would have to change | [0001-REPORT](reports/0001-REPORT-go-port-feasibility.md) |
| know what v1 is: native magic-cli-remote CLI, ACP, headless stdio, MCP | [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) |
| see how v1 maps onto mcremote slash commands (`/compact`, `/usage`, …) | [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) |
| see the v1 implementation phases | [0002-PLAN](decisions/0002-PLAN-cli-acp-headless-mcp-v1.md) |
| understand what this repository contains today | [architecture.md](architecture.md) |
