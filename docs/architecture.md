# Architecture

How `pi-go` is put together, as it is now. This file carries no history and no rationale — when you want to know *why* something is this way, the decision records under [decisions/](decisions/) hold the argument, and [README.md](README.md) points at the ones people ask for most.

## What it is

A Git repository for a Go rewrite of the Pi coding-agent CLI and the libraries that CLI needs. The stated binary name is `pigo`. The stated command and configuration stack is Cobra and Viper. The stated terminal-UI stack is Charm (Lip Gloss and related packages).

There is no Go module, no `cmd/` tree, and no binary. The working tree is documentation. The package map in [0004-MADR](decisions/0004-MADR-go-module-architecture.md) is proposed, not built. This file describes it only once the packages exist.

The planned 1.0 provider library is `github.com/maccavelli/go-llmprovider-sdk`. The planned self-update library and release-publish workflow are `github.com/maccavelli/go-core-lib` `v1.1.0`. Neither is imported in this tree yet.

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

No package map, no CLI surface, no session format, and no provider list belong in this file until they exist in this tree. Those surfaces of the TypeScript product are inventoried in the report.
