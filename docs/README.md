# gobble-cli documentation

`gobble-cli` is the repository for **gobble**, a coding agent written in Go that takes the best of several agent harnesses and makes it its own. This tree holds three kinds of document:

- reports that measured something;
- the decisions behind gobble's design;
- guides you follow to do a thing.

- **[architecture.md](architecture.md)** — how the pieces fit together, as they are now.
- **guides/** — task-shaped walkthroughs: [developing.md](guides/developing.md), on building, testing, the gates and adding a package.
- **[decisions/](decisions/)** — numbered MADR/PLAN pairs.
- **[reports/](reports/)** — what was observed, deciding nothing.

## Records

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | REPORT | [Go port feasibility of the Pi agent harness](reports/0001-REPORT-go-port-feasibility.md) | observation |
| 0002 | MADR | [v1 native magic-cli-remote CLI: Kong over ACP, ACP stdio, MCP client](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) | proposed (amended 2026-10-04 (0008); Phase 7's extension set 2026-10-07) |
| 0002 | PLAN | [Implement the ACP core](decisions/0002-PLAN-cli-acp-headless-mcp-v1.md) | in progress (Phases 1–3 complete 2026-10-05, Phases 4–6 complete 2026-10-06, Phases 7–8 complete 2026-10-07 (CI green, run 37650274632); amended 2026-10-04 (0008), 2026-10-05, 2026-10-06, 2026-10-07 (Phases 7 and 8 made executable)) |
| 0003 | MADR | [The product is `gobble`: binary, module path, directories, wire names, read-only Pi bridge](decisions/0003-MADR-gobble-product-identity.md) | proposed (amended 2026-10-01) |
| 0004 | MADR | [One Go 1.27.1 module of contract-first packages, open standards at every boundary](decisions/0004-MADR-go-module-architecture.md) | proposed (decision text rewritten 2026-10-04; amended 2026-10-05; go-tui-lib note 2026-10-06; release pipeline amended 2026-10-07) |
| 0004 | PLAN | [Scaffold: module, contracts, import boundaries, toolchain, release](decisions/0004-PLAN-go-module-architecture.md) | in progress (Phase D, Phase 0, and Phase 1 complete; Phase 2 steps 2–4 complete 2026-10-05, and steps 1, 5 (in part), 6–9 and 11 through 0008-PLAN P2–P4; Phase 3 complete 2026-10-05; Phase 2 steps 10, 12, 13 not started; Phase 5 made executable and complete 2026-10-07; Phase 4 step 1 through 0002-PLAN Phase 8 2026-10-07; Phase 4 steps 2–4 made executable 2026-10-07 and complete 2026-10-07 (CI rehearsal green, run 37677754325); amended 2026-10-04 (0007); amended 2026-10-04 (0008); Phase 2 steps 2–4 amended 2026-10-05; fleet library pins refreshed 2026-10-06) |
| 0005 | MADR | [The v1 line: every portable Pi capability, tiered 1.0 / 1.x / exp](decisions/0005-MADR-v1-feature-scope.md) | accepted 2026-10-07 (decision text rewritten 2026-10-04; platform note 2026-10-07; reference-harness amendment 2026-10-07; native tools amendment 2026-10-08; 0012 amendment 2026-10-08; grep row's globs 2026-10-08) |
| 0005 | PLAN | [Implement the v1.0.0 gate, the v1.x train, and `exp/`](decisions/0005-PLAN-v1-feature-scope.md) | in progress (approved 2026-10-07; decision text rewritten 2026-10-04; F4, F5, F8, F9 and F10 notes 2026-10-07; F10 step 6 `apidiff` note 2026-10-07; F1 made executable 2026-10-07; restructured into five sub-phases and F1a reworked 2026-10-07; F1a complete 2026-10-07; F1b complete 2026-10-07; native tools placed 2026-10-08; 0012 notes 2026-10-08; F1c made executable 2026-10-08) |
| 0006 | MADR | [Adopt goose's terminal-CLI mechanics, not its product command tree](decisions/0006-MADR-goose-cli-port-candidates.md) | proposed (amended 2026-10-04 (0008)) |
| 0007 | MADR | [Bring the repository up to the fleet scaffold: agent rules, lint and line-ending config, records tooling, portable Makefile](decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md) | accepted (amended 2026-10-04 (0008)) |
| 0007 | PLAN | [Bring gobble-cli up to the fleet repository scaffold](decisions/0007-PLAN-repository-scaffolding-to-fleet-standard.md) | completed (staged; the owner commits) |
| 0008 | MADR | [The native CLI mode: a lightweight line client on Kong, x/term and the standard library, natively integrated with magic-cli-remote](decisions/0008-MADR-native-cli-mode.md) | accepted (`tools.go` note 2026-10-07) |
| 0008 | PLAN | [Implement the native CLI mode](decisions/0008-PLAN-native-cli-mode.md) | completed (2026-10-05; P8 emphasis follow-up 2026-10-05) |
| 0009 | REPORT | [magic-cli-remote findings from gobble work](reports/0009-REPORT-magic-cli-remote-findings.md) | observation (running; updated 2026-10-07) |
| 0010 | REPORT | [Harness design survey: Pi, opencode and Kilo across the v1 line](reports/0010-REPORT-harness-design-survey.md) | observation (2026-10-07) |
| 0011 | REPORT | [Native tools across six harnesses: which operations gobble implements itself](reports/0011-REPORT-native-tools-survey.md) | observation (2026-10-08; 88 verified claims; schemas, descriptions, loading and BM25 evaluated 2026-10-08) |
| 0012 | MADR | [Tool schemas, descriptions and loading: JSON Schema from Go types, sectioned Markdown descriptions, two exposure tiers, and an in-house BM25 `tool_search`](decisions/0012-MADR-tool-schemas-descriptions-and-loading.md) | accepted (2026-10-08; 38 verified claims) |
| 0012 | PLAN | [Tool schemas and descriptions: the contract, the five built-in tools, and the strict probe](decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md) | completed (P1–P6 and P8, 2026-10-08; P7 waits on go-llmprovider-sdk's per-tool strict flag) |

0003-MADR has no PLAN of its own. 0004-PLAN Phase 2 and 0005-PLAN F10 implement it.

## Build order

The plans run in sequence.

- **0004-PLAN.**
  - Run: Phase D, Phase 0 (`925ef0abf83e475c33b3ffae14685617b035639e`) and Phase 1 (`661b14e768407264a38687e2db99201cae2a04a1`).
  - Run: Phase 2's steps 2–4, and its steps 1, 5 (in part), 6–9 and 11 through [0008-PLAN](decisions/0008-PLAN-native-cli-mode.md), which is completed.
  - Run: Phase 3, on 2026-10-05.
  - Run: Phase 4, step 1 through 0002-PLAN Phase 8 and steps 2–4 on 2026-10-07.
  - Run: Phase 5 on 2026-10-07.
  - Not run: Phase 2's steps 10, 12 and 13, which wait on sessions and providers.
- **0002-PLAN.** Run: Phases 1–8, completed on 2026-10-07.
- **0005-PLAN.**
  - Run: F1a, the file tools, and F1b, bash and powershell, on 2026-10-07.
  - Next: F1c, as two sub-phases, F1c-1 and F1c-2, once approved.
- **0012-PLAN.** Run: P1–P6 and P8 on 2026-10-08. P7 waits on go-llmprovider-sdk's per-tool strict flag.
- `git ls-remote origin refs/heads/main` on 2026-10-08 returned `740f7b71450238e747dba99b4fa1eccd488ad79e`.

The plans:

1. **[0004-PLAN](decisions/0004-PLAN-go-module-architecture.md)** — source documentation in Phase D, then the scaffold in Phases 0–5. It replaces 0002-PLAN Phase 0.
2. **[0002-PLAN](decisions/0002-PLAN-cli-acp-headless-mcp-v1.md) Phases 1–8** — the ACP agent, Kong as an ACP client, the first tools, sessions, MCP, and native slash commands.
3. **[0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md)** — F1–F10 form the v1.0.0 gate. X1–X8 are the v1.x train and `exp/`.
4. **[0012-PLAN](decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md)** — tool schemas and descriptions for the built-in tools, and the strict probe.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| know where the project started: whether a complete Go port of Pi was feasible | [0001-REPORT](reports/0001-REPORT-go-port-feasibility.md) |
| know why ACP is the command API and how gobble sits under mcremote | [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) |
| see how gobble maps onto mcremote slash commands (`/compact`, `/usage`, …) | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (target table), [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) (floor) |
| know why gobble cannot answer `session/set_model` | [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md), second amendment |
| know where gobble keeps config, sessions and credentials | [0003-MADR](decisions/0003-MADR-gobble-product-identity.md) |
| know the licence | [0003-MADR](decisions/0003-MADR-gobble-product-identity.md) (Apache-2.0), [LICENSE](../LICENSE), [NOTICE](../NOTICE) |
| use my existing Pi skills, MCP servers and sessions with gobble | [0003-MADR](decisions/0003-MADR-gobble-product-identity.md) (read-only Pi bridge) |
| see the package map, public SDK tiers and import rules | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) |
| know which Go 1.25–1.27 features gobble uses, and for what | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (idiom list) |
| know which standards each boundary speaks (ACP, MCP, Agent Skills, OTel, OAuth, XDG) | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) |
| check whether a Pi feature made it into v1, and at which tier | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) |
| extend gobble (hooks, MCP servers, skills, packages, Go SDK) | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md), Extensibility |
| see what has to pass before `v1.0.0` is tagged | [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md), F10 |
| start building | [0004-PLAN](decisions/0004-PLAN-go-module-architecture.md) |
| know how gobble talks to models | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (D10: `llm` facade over go-llmprovider-sdk) |
| know how `gobble update` and GitHub releases work | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (self-update), [0004-PLAN](decisions/0004-PLAN-go-module-architecture.md) Phase 4 (assets) |
| see which sibling libraries exist today and what gobble will use | [architecture.md](architecture.md#sibling-sources), [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (2026-10-04: core TUI is go-tui-lib; self-update is go-selfupdate-lib), [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md) (F9, F10) |
| use the default terminal CLI, or the enhanced TUI | [architecture.md](architecture.md), [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (two terminal modes) |
| understand what this repository contains today | [architecture.md](architecture.md) |
| build gobble, run the gates, or add a package | [guides/developing.md](guides/developing.md) |
| see which operations gobble runs as native tools rather than shell commands, and why | [0011-REPORT](reports/0011-REPORT-native-tools-survey.md), [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (amendment of 2026-10-08) |
| know how gobble's tools are described to a model, which are always loaded, and how `tool_search` finds the rest | [0012-MADR](decisions/0012-MADR-tool-schemas-descriptions-and-loading.md), [0011-REPORT](reports/0011-REPORT-native-tools-survey.md) (evaluation) |
| see how other harnesses design what gobble builds, and which design gobble chose | [0010-REPORT](reports/0010-REPORT-harness-design-survey.md), [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (amendment of 2026-10-07) |
| see each package's tier and role | [architecture.md](architecture.md#packages) |
| review what gobble work found about magic-cli-remote, and the changes it needs there | [0009-REPORT](reports/0009-REPORT-magic-cli-remote-findings.md) |
| know the rules agents follow in this repository | [AGENTS.md](../AGENTS.md) |
| know why the repository has the files and gates it has (AGENTS.md, lint configs, records check) | [0007-MADR](decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md) |
