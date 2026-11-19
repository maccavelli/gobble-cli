---
status: complete
date: 2026-11-19
associated-madr: "0062-MADR-pin-go-selfupdate-lib-v1-26-3-and-go-tui-lib-v2-12-1.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement pinning go-selfupdate-lib v1.26.3 and go-tui-lib v2.12.1

Associated MADR: [0062-MADR-pin-go-selfupdate-lib-v1-26-3-and-go-tui-lib-v2-12-1.md](0062-MADR-pin-go-selfupdate-lib-v1-26-3-and-go-tui-lib-v2-12-1.md)

*Approved in advance by the owner on 2026-11-18 ("One record pair per repository, approved now through CI; magic-cli-remote after its lint phases (Recommended)").*

## Goal

`go.mod` requires go-selfupdate-lib `v1.26.3` and go-tui-lib/v2 `v2.12.1`; `ci.yml`, `README.md` and
`docs/architecture.md` name the same tags; no other requirement moves and no code changes.

## Scope

* In: `go.mod`, `go.sum`; `.github/workflows/ci.yml` (`:191`, `:203`); `README.md`; `docs/architecture.md`; one dated
  pointer line on `0054-MADR-go-1-27-4-and-sibling-pin-refresh.md`; `docs/README.md` rows for 0062.
* Out: any Go source, `acpext`, a tag, a release, an install, and magic-cli-remote.

## Implementation Steps

1. On a `git archive` copy of the base, `go get` both tags and `go mod tidy`; list every requirement that moves, and
   whether `acpext` is affected. Stop if anything but the two modules moves, a major version changes or an advisory
   appears.
2. Read both libraries' history between the tags for what changes for this consumer.
3. Peel each library's tag to its commit; apply the `go get` and `go mod tidy` to the tree; edit `ci.yml:191` and
   `:203`, `README.md` and `docs/architecture.md`, each edit asserted to occur once.
4. Gates: gofmt, `go vet`, `go mod tidy -diff` in both modules, `make lint`, `make vulncheck`, `check_libs.py` for each
   library, `go test` for both modules, `make check-records`, `make markdownlint`, the disclosure check, then
   `make preflight` after the last record edit.
5. Stage the records and the docs index only; the coordinator commits and pushes.

## Verification

The MADR's Confirmation list. Released changelog entries and earlier records keep their old versions; they are history.

## Rollout and Rollback

Commit and push to `main` by the coordinator on a green gate; CI is the check. No release is cut. Rollback: revert the
commit.

## Execution record

* 2026-11-18, base: branch `p3-acpext-v0.5.0`, `HEAD` and local `main` both `a3a2d7c`, clean tree. The number came from
  `python3 scripts/check_records.py --next` ("0062").
* 2026-11-18, step 1, scratch copy of `a3a2d7c`: with `GOPROXY=off`, `go get` failed, the module cache lacking go-tui-lib
  `v2.12.1` ("module lookup disabled by GOPROXY=off"). With the public proxy for that one command, it downloaded
  go-tui-lib/v2 `v2.12.1` (its `.mod` and `.zip`) and nothing else; go-selfupdate-lib `v1.26.3` was already in the cache.
  `go mod tidy` with `GOPROXY=off`: exit 0. Changes: `go.mod` two lines, `go.sum` four lines (the two modules' `h1:` and
  `go.mod` sums); no other requirement, direct or indirect, moved. `go build ./...` exit 0. `go -C acpext mod tidy
  -diff`: exit 0; `acpext/go.mod` has no requirement and there is no `acpext/go.sum`.
* 2026-11-18, step 2: the MADR's Facts. Neither release raises a requirement above this module's `go 1.27.4`.
* 2026-11-18, step 3: the tags peeled with `git ls-remote --tags` (go-selfupdate-lib `v1.26.3` to
  `1dffdc9dee24a7534513dc51c9039408be37c09b`, go-tui-lib `v2.12.1` to `6ac83559e3f1d366a903a23b69b22eaa5146a2af`).
  `go get` of both tags with `GOPROXY=off` (the cache now held them): exit 0; `go mod tidy`: exit 0; `go.mod` and
  `go.sum` are byte-identical to the scratch copy's. Changed lines: `go.mod:19` and `:20`; `go.sum:53` to `:56`;
  `.github/workflows/ci.yml:191` and `:203` (`@1dffdc9… # v1.26.3`); `README.md:178` and `:179`;
  `docs/architecture.md:201` and `:202`. Every other statement of the two older versions is in a committed record
  (history) and stays. A dated pointer line was appended to 0054-MADR.
* 2026-11-18, gates, `GOPROXY=off` where the tool allows it, exit status taken before any filter:

  | Gate | Exit | Result |
  | --- | --- | --- |
  | `gofmt -l .` | 0 | no file listed |
  | `go vet ./...` in `.` and in `acpext` | 0, 0 | clean |
  | `go build ./...` | 0 | clean |
  | `go mod tidy -diff` in `.` and `acpext` | 0, 0 | nothing to change |
  | `make lint` (`golangci-lint` v2.15.0, the PATH binary's `version` line) | 0 | "0 issues." for `.` and `acpext` |
  | `make vulncheck` | 0 | "No vulnerabilities found." for both modules |
  | `actionlint .github/workflows/ci.yml` | 0 | clean |
  | `check_libs.py --lib go-selfupdate-lib --tag v1.26.3`, through `run-heavy` (18:41:55 to 18:43:48) | 0 | "0 failing checks", its advisories check `ok` |
  | `check_libs.py --lib go-tui-lib --tag v2.12.1`, through `run-heavy` (18:43:48 to 18:45:46) | 0 | "0 failing checks", `archtest` and advisories `ok` |
  | `go test -count=1 ./...` in `.` and `acpext`, through `run-heavy` (18:46:06 to 18:47:30) | 0 | every package `ok` |
* 2026-11-18, `make preflight` through `run-heavy` (18:47:53 to 18:56:05, 491 s): exit 0, "preflight passed" (gofmt,
  go mod tidy, pre-add-check, go vet, staticcheck, golangci-lint, govulncheck, check-acpext-deps, fix-check, archtest,
  apidiff, verify-build-metadata, check-records, check-icon, markdownlint). No Go file changed, so no further pre-add.
  `-race` is not run by CI's `ci.yml`, so it was not run.
* 2026-11-18, after this entry `make check-records`, `make markdownlint` and the disclosure scan of the changed files
  were run again. The plan stays `in-progress` until the commit, the push and CI's result are recorded.
* Deviations: none. Not done: no commit, push, tag or release, as scoped; no live test and no `-race` run.
* 2026-11-19, close-out. The coordinator's gate on the tree: `make lint` 0; `make check-records` 0;
  `make markdownlint` 0; `go mod tidy -diff` 0 in the root and in `acpext`; `make vulncheck` 0; `make preflight` through
  `run-heavy` 0; `go test -race -count=1 -timeout 30m ./...` through `run-heavy` 0, 38 packages ok, the tree unchanged
  by the gate (the `-race` run is the one the 2026-11-18 entry lists as not done). The records are `2d17e31` and the
  change `080205f` on `p3-acpext-v0.5.0`; `main` was fast-forwarded `a3a2d7c..080205f`, no force, the pre-push guard run.
  CI on `080205f` on `main`, attempt 1, success: `validate` on ubuntu-24.04, macos-15 and windows-2025 (17 steps each),
  `test-isolation`, `sbom`, `build / build` (18 steps), `build / identity` for windows/arm64, linux/amd64, linux/arm64,
  darwin/arm64 and windows/amd64; `release` skipped (no tag); Docs and Dependency Graph success. The two reusable-workflow
  pins at the `v1.26.3` commit therefore ran in the `build` job, which the 2026-11-18 entries could not show.
* 2026-11-19, acceptance, each criterion against that evidence:
  * Goal and Confirmation 1, `go.mod`, `go.sum`, `ci.yml`, `README.md` and `docs/architecture.md` name the two tags and
    nothing else changed: met; `080205f` changes those five files alone, 12 lines each way, and no Go file.
  * Confirmation 2, `make preflight` passes and `golangci-lint` v2.15.0 reports "0 issues." on both modules: met;
    preflight and `make lint` 0 in this record's run and in the coordinator's.
  * Confirmation 3, `check_libs.py` reports "0 failing checks" for each library, advisories included: met by this
    record's own runs (2026-11-18, both exit 0); the coordinator did not repeat them.
  * Confirmation 4, CI green after the push: met; the run above.
  * Out of scope and unchanged: `acpext`, tags, a release, an install, magic-cli-remote.
* 2026-11-19, status: `in-progress` becomes `complete`, the value of 0053-PLAN, 0054-PLAN, 0055-PLAN and 0059-PLAN.
  `scripts/check_records.py` does not check a status value; `complete` is the skill's word and the repository's
  majority, `completed` being used by eight older plans.
