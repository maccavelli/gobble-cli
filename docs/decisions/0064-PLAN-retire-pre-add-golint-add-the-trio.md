---
status: implemented
date: 2026-11-20
associated-madr: "0064-MADR-retire-pre-add-golint-add-the-trio.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Retire per-file golint from the pre-add; restore the trio in revive

Associated MADR: [0064-MADR-retire-pre-add-golint-add-the-trio.md](0064-MADR-retire-pre-add-golint-add-the-trio.md)

## Goal

`scripts/go-precheck.sh` no longer runs `go tool golint`; its lint step is the
exact `make lint` command over both modules; `revive.rules` carries
`exported`, `package-comments` and `var-naming`; `go.mod` drops
`golang.org/x/lint/golint`; every doc that names the old step says the new
one.

## Scope

* In: `scripts/go-precheck.sh` (step 2 and banner), `go.mod` tool directive
  (and both modules' go sums via tidy), `.golangci.yml` (`revive.rules`
  additions only), `AGENTS.md` pre-add paragraph,
  `docs/guides/developing.md` (tool list, sample output), the 0064 record
  pair, `docs/README.md` rows.
* Out: any other linter, the pin (stays `v2.15.0`, 0061), code changes beyond
  what the trio's findings force (bounded, in the same change), CI, releases.

## Implementation Steps

1. Scratch first (`git archive` copies of the tip): with the trio added and
   the per-file step replaced, `make lint` counts what the trio flags; a
   planted missing-comment file and a planted `error-strings` break fire as
   named `file:line` findings through the new wrapper step; reverting the
   plants returns "0 issues." for `.` and `acpext`.
2. Apply on the tree in one pass: the script change (golint step out,
   per-module golangci-lint in — the `make lint` command and modules, config
   by absolute path, host GOOS), the banner string; the trio into
   `revive.rules`; the `tool` directive dropped and both modules tidied; the
   docs (`AGENTS.md`, `developing.md` tool list and sample output) moved to
   the same wording; fix whatever the trio flags (bounded).
3. Gates: `gofmt -l .` clean; `go vet ./...` in `.` and in `acpext` (plus
   `GOOS=windows`); `make lint` "0 issues." both modules; `go mod tidy -diff`
   in both; `make check-records`, `make markdownlint`; `make preflight`
   through `run-heavy`.

## Verification

`make lint` prints "0 issues." for `.` and `acpext` with the trio in place;
the planted missing-comment file fails the new pre-add naming `file:line`,
and passes once fixed; `make preflight` passes.

## Rollout and Rollback

Commit and push by the coordinator once the owner asks; CI is the check.
Rollback: revert the commit.

## Execution record

* 2026-11-20, scratch fences (git archive copy of `365159e`): with the trio added and the per-file step replaced,
  `golangci-lint run -c .golangci.yml ./...` and the same in `acpext` print "0 issues." each; a planted
  missing-comment file (`internal/archtest/plant_export.go`, an exported function with no doc comment) fails the new
  wrapper naming `plant_export.go:3:1: exported: exported function PlantedExport should have comment or be unexported
  (revive)`; a planted `error-strings` break (`plant_err.go`, capitalized message) fails as `ST1005`; both through the
  wrapper's golangci-lint step, named `file:line`; reverting the plants returns "0 issues." for `.` and for `acpext`.
* 2026-11-20, tree application: `scripts/go-precheck.sh` step 2 replaced (per-file `go tool golint` out; per-module
  `golangci-lint run --config "$REPO_ROOT/.golangci.yml" ./...` in — the `make lint` command and modules, host GOOS;
  wrapper banner updated); the trio into `.golangci.yml`'s `revive.rules` (nothing flagged: the archive copy was
  already clean under the trio); `golang.org/x/lint/golint` out of the `tool` directives, both modules tidied
  (`go.sum` drops 13 lines); `AGENTS.md` pre-add paragraph and `docs/guides/developing.md` (tool list and sample
  output) moved to the same wording.
* 2026-11-20, gates: `gofmt -l .` clean; `go vet ./...` in `.`, in `acpext`, and with `GOOS=windows`, all exit 0;
  `go mod tidy -diff` in `.` and in `acpext`, no dependency moved; `make lint` "0 issues." for `.` and for `acpext`
  with the trio; `make markdownlint` 0 issues (59 files); `make preflight` through `run-heavy` passed — with
  `PYTHON=/opt/homebrew/bin/python3.12`: the first run died at `verify-build-metadata` on this host's python3.9
  (`datetime` has no `UTC`), which is an environment drift, not a gate failure; the house `PYTHON ?= python3`
  override covers it.
