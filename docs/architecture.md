# Architecture

How `gobble-cli` is put together, as it is now. This file carries no history and no rationale — when you want to know *why* something is this way, the decision records under [decisions/](decisions/) hold the argument, and [README.md](README.md) points at the ones people ask for most.

## What it is

A Git repository for a planned Go rewrite of the Pi coding-agent CLI. The
planned binary name is `gobble`. The proposed command and configuration stack
is Cobra and Viper; the proposed terminal UI uses Charm v2.

There is no Go module, no `cmd/` tree, and no binary. The working tree is documentation. The package map in [0004-MADR](decisions/0004-MADR-go-module-architecture.md) is proposed, not built. This file describes it only once the packages exist.

## Sibling sources

The three local sibling repositories are active source trees. The versions
and commits below are observed source evidence, not requirements in this
repository. No `go.mod` or gobble package imports them yet.

| Sibling repository | Current observed surface | Planned gobble boundary |
| :--- | :--- | :--- |
| [go-llmprovider-sdk](https://github.com/maccavelli/go-llmprovider-sdk) | No tag; observed commit `efd9c61` adds catalog and setup-wizard work after `67fc56e` without changing the adapter-facing contract. `llmprovider` has ten built-in providers, `Response.Usage`, `*APIError` and opt-in environment helpers; `llmprovider/auth` holds OAuth and token-store code. Built-in native streaming is unsupported. | A planned `llm/provider` adapter under gobble's own `llm` facade. F4 in [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md) sets the usage, error and credential rules. |
| [go-selfupdate-lib](https://github.com/maccavelli/go-selfupdate-lib), formerly go-core-lib | Latest tag `v1.5.0` (`6deaa52`), the first under `github.com/maccavelli/go-selfupdate-lib`, exports `selfupdate`, `selfupdate/cli`, `selfupdate/selfupdatetest` and `buildinfo`. `buildinfo` and `selfupdate/cli` arrived in `v1.4.0`, and opt-in prerelease channels in `v1.3.0`. The old path `github.com/maccavelli/go-core-lib` ends at `v1.4.1`, deprecated. | The planned `gobble update` and raw-binary release workflow use a rechecked stable `v1.x` tag of the new path, `v1.5.0` or later. Gobble's scaffold has its own planned `internal/buildinfo`. F10 in [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md) compares it with the released `buildinfo` and selects the tag. |
| [go-tui-lib](https://github.com/maccavelli/go-tui-lib) | Tag `v0.1.0` (`5c57806`) exports `layout`, `workspace`, `glyph`, `theme` and `tuitest` on Charm v2. Observed later commit `7907590` changes documentation only. The recorded workspace defects await a corrected tag; transcript, picker, permission dialog and editor packages are absent. | Planned `internal/tui` uses tested reusable workspace pieces. Gobble-specific ACP panes stay in gobble. F9 in [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md) sets the integration check. |

## Tree

```
README.md                 repository entry; links here
LICENSE                   Apache License 2.0
NOTICE                    Pi MIT credit; sibling Apache-2.0 libraries
docs/
  README.md               ToC and the "I want to…" matrix
  architecture.md         this file
  decisions/              MADR/PLAN pairs
  reports/                numbered observations
  guides/                 unnumbered how-to documents (empty)
```

The first record is [0001-REPORT-go-port-feasibility.md](reports/0001-REPORT-go-port-feasibility.md). It describes the TypeScript product that would be rewritten; it is not a description of code in this repository.

## What is not here

No implemented package map, CLI surface, session format or provider list
exists in this tree. The proposed package map and import rules are in
[0004-MADR](decisions/0004-MADR-go-module-architecture.md); the TypeScript
product's surfaces are inventoried in the [report](reports/0001-REPORT-go-port-feasibility.md).
