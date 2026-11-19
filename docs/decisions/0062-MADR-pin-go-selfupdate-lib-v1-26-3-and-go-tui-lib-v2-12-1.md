---
status: accepted
date: 2026-11-18
decision-makers: repository owner
consulted: "this repository at a3a2d7c (main, 2026-11-18); go-selfupdate-lib at v1.26.2 and v1.26.3 and go-tui-lib at v2.12.0 and v2.12.1, read from local clones and, for the tags, from git ls-remote"
informed: "magic-cli-remote, which requires the same two libraries and moves after its lint phases"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Pin go-selfupdate-lib v1.26.3 and go-tui-lib v2.12.1

## Context and Problem Statement

[0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md) pinned
go-selfupdate-lib `v1.26.2`; go-tui-lib `v2.12.0` was pinned earlier and kept by that record's F10. Both libraries
have since tagged a patch release, and `go.mod:19` and `go.mod:20` name the older ones. `AGENTS.md` ("Dependencies")
requires a MADR in this repository for every module a change moves.

Asked how the consumer bumps should proceed (gobble-cli and magic-cli-remote require the two older tags), the owner
chose, as the option's label: "One record pair per repository, approved now through CI; magic-cli-remote after its
lint phases (Recommended)". For this repository that means: now, with a record pair; commit and push to the default
branch on a green gate; nothing installed on the owner's machine and no release cut.

### Facts, read on 2026-11-18

* **Tags.** `git ls-remote --tags` lists go-selfupdate-lib `v1.26.3` (annotated tag `10316bc`, commit
  `1dffdc9dee24a7534513dc51c9039408be37c09b`, dated 2026-11-17) as its newest tag, over `v1.26.2` (commit
  `e62d77cddaf264cc6ec7bbc4f39b90322542f752`); and go-tui-lib `v2.12.1` (annotated tag `05583ac`, commit
  `6ac83559e3f1d366a903a23b69b22eaa5146a2af`, 2026-11-17) as its newest `v2` tag, over `v2.12.0`.
* **Both are Go-pin releases.** go-selfupdate-lib `v1.26.2..v1.26.3` moves its `go.mod` from `go 1.26.10` and
  `toolchain go1.27.3` to `go 1.26.11` and `toolchain go1.27.4` (GO-2026-6859, in `net/http`, reached through
  `GitHubSource`'s client; its CHANGELOG, "Security"), its own CI and docs, and `internal/testproc` and test files. No
  non-test file in an importable package differs. go-tui-lib `v2.12.0..v2.12.1` moves `go 1.27.2` to `go 1.27.4` in its
  `go.mod` and in `go.work`, test templates and docs, and changes no source file. Neither adds, removes or alters an API
  or a behaviour a program can see.
* **The Go requirement does not rise for this consumer.** This module is `go 1.27.4`, so go-selfupdate-lib's `go
  1.26.11` and go-tui-lib's `go 1.27.4` are both met. Neither library's `go.mod` changes a requirement: each
  `go.mod` diff is the `go` (and, for go-selfupdate-lib, `toolchain`) line alone.
* **The CI workflow pins move.** `.github/workflows/ci.yml:191` and `:203` call go-selfupdate-lib's build and publish
  workflows at `e62d77c… # v1.26.2`. No file under `.github/workflows` of the library other than its own `ci.yml`
  differs between the tags, but both reusable workflows take their Go version from the library's `go.mod`
  (`go-version-file`, `GOTOOLCHAIN: local`), which now reads `toolchain go1.27.4`: a caller at `v1.26.3` builds its
  release tools with Go 1.27.4, as its CHANGELOG says. The pins are kept at the tag in use (0048-MADR Q6 (a);
  0054-MADR D4).
* **A scratch copy shows the whole change.** On a `git archive` copy of `a3a2d7c`, `go get` of the two tags and
  `go mod tidy` change the two `go.mod` requirement lines and four `go.sum` lines (a `h1:` and a `go.mod` sum for each
  module), and no other requirement, direct or indirect. `acpext` has no `go.sum`, imports the standard library alone,
  and `go -C acpext mod tidy -diff` is clean: it is not affected.

## Considered Options

* A. Pin both tags now, in one commit, with their CI pins and docs; no code change.
* B. Wait for magic-cli-remote and move the three repositories together.
* C. Stay at `v1.26.2` and `v2.12.0`.

## Decision Outcome

Chosen option: "A", because the owner chose it, the two releases change nothing this consumer can see, and the scratch
copy moves nothing else. The pins follow the libraries' latest tags, as 0054-MADR D3 and 0050-MADR did.

### Consequences

* Good, because gobble requires each library's newest tag, and a build of gobble now reaches the libraries compiled
  against Go 1.27.4 where it matters.
* Neutral, because no behaviour changes, no API is added or removed, and `acpext` is untouched; no tag or release is
  cut and nothing is installed.
* Bad, because `ci.yml`'s two workflow pins are a third place that names go-selfupdate-lib's version and each must be
  peeled to its commit before it is edited; a wrong SHA fails CI, not the local gates.

### Confirmation

* `go.mod` requires go-selfupdate-lib `v1.26.3` and go-tui-lib/v2 `v2.12.1`, the `go.sum` lines are the four the scratch
  copy produced, and `git diff` of the two files shows no other line.
* `make preflight` passes, and `golangci-lint` v2.15.0 reports "0 issues." on both modules.
* `python3 scripts/check_libs.py --lib go-selfupdate-lib --tag v1.26.3` and `--lib go-tui-lib --tag v2.12.1` report
  "0 failing checks", their advisory check included.
* CI is green after the push.

## More Information

* [0062-PLAN-pin-go-selfupdate-lib-v1-26-3-and-go-tui-lib-v2-12-1.md](0062-PLAN-pin-go-selfupdate-lib-v1-26-3-and-go-tui-lib-v2-12-1.md)
  implements this record.
* [0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md): F7 and D3, the pin
  of go-selfupdate-lib `v1.26.2` that this record moves on from; it receives a dated pointer line.
* [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md): the root requires
  `acpext` at a tag; no `replace` and no committed `go.work` is added here.
* go-selfupdate-lib `CHANGELOG.md` ("v1.26.3") and its 0045-MADR; go-tui-lib `docs/guides/migrating.md` ("v2.12.1").
