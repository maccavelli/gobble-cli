---
status: implemented
date: 2026-11-20
associated-madr: "0061-MADR-move-golangci-lint-to-v2-15-0.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement moving golangci-lint to v2.15.0

Associated MADR: [0061-MADR-move-golangci-lint-to-v2-15-0.md](0061-MADR-move-golangci-lint-to-v2-15-0.md)

*Approved in advance by the owner on 2026-11-18 ("One short record pair per repository, approved now through CI").*

## Goal

golangci-lint `v2.15.0` is the pin everywhere it is written, and the tree lints clean with it.

## Scope

* In: `Makefile` (comment and install hint), `.github/workflows/ci.yml`, `AGENTS.md`, `docs/guides/developing.md`; one
  dated pointer line on `0007-MADR-repository-scaffolding-to-fleet-standard.md`; `docs/README.md` rows for 0061.
* Out: any code, dependency, configuration change or release.

## Implementation Steps

1. Run `v2.15.0` on a `git archive` copy of the tip, both modules: the red (expected "0 issues.").
2. Repeat 0059-PLAN P1's plants (steps 3, 4 and 5) with `v2.15.0` on scratch copies.
3. Move the pin in every place named in Scope.
4. Gates: gofmt, `go vet`, `make lint`, `go mod tidy -diff`, tests, `make check-records`, `make markdownlint`,
   `actionlint`, `make preflight`.

## Verification

`make lint` prints "0 issues." for `.` and `acpext`; the plants behave as 0059-PLAN P1 recorded; `make preflight`
passes.

## Rollout and Rollback

Commit and push by the coordinator; CI is the check. Rollback: revert the commit.

## Execution record

* 2026-11-18, step 1: `v2.15.0` (go1.27.4) on a `git archive` copy of `243ffc6`: "0 issues." for the root module and for
  `acpext`, exit 0 each. The pin was found in `Makefile` (comment, install hint), `.github/workflows/ci.yml`,
  `AGENTS.md` and `docs/guides/developing.md`; the other mentions are in committed records and stay.
* 2026-11-18, step 2, 0059-PLAN P1's plants at `v2.15.0` on scratch copies, with fewer runs than P1 (the queue was busy):
  * step 3, an `error-strings` plant in a test file: with the second revive exclusion removed, exit 1 and
    `zz_other_test.go:6:43: error-strings: ... (revive)`; with the tip's configuration, 3 of 3 runs exit 0.
  * step 4, the `LoginOf` collision (a test helper renamed back to `LoginOf`, as 0059-PLAN P1): 10 of 10 cold runs exit 1,
    one `confusing-naming` line each (8 at `llm/provider/login.go:136`, 2 at `llm/provider/export_test.go:22`).
  * step 5, a case-only pair wholly inside test files: 5 of 5 cold runs exit 1, one `confusing-naming` line each.
  * Not repeated in this first pass: step 4's 20 runs (10 were made) and step 6's warm cache, because the shared
    queue was busy. Both were then run in full, below.
* 2026-11-18, step 2 completed, 11:01 to 11:12 local, on a fresh scratch copy of `243ffc6` with the `LoginOf` plant,
  `v2.15.0`, the tip's `.golangci.yml`:
  * step 4: 20 cold runs (a new cache each): 20 of 20 exit 1, one `confusing-naming` line each (14 at
    `llm/provider/export_test.go`, 6 at `llm/provider/login.go`); `acpext` "0 issues." each time.
  * step 6: three caches, three runs per cache sharing it: 9 of 9 exit 1, one `confusing-naming` line each. Each cache
    kept its first run's file (cache 1 `export_test.go`, cache 2 `login.go`, cache 3 `export_test.go`).
  * 0059-MADR D5 reads: "a change of the pin repeats the plan's plants before it lands." The plants are 0059-PLAN P1's
    steps 3, 4 and 5; step 6 reuses step 4's plant. Steps 3 to 6 have all been run at `v2.15.0`, step 4 at its full 20
    runs, so D5 is met in full. Steps 1, 2 and 7 (variants, D2's list, today's tree) are not plants: step 7's
    equivalent is the clean `make lint` below.
* 2026-11-18, gates, on the change as it stands at this entry. Start and end times were kept only for the two heavy
  runs; the others ran in seconds and are given by exit status.

  | Gate | Exit | Result |
  | --- | --- | --- |
  | `gofmt -l .` | 0 | no file listed |
  | `go vet ./...` in `.`, in `acpext`, and with `GOOS=windows` | 0, 0, 0 | clean |
  | `make lint` (`golangci-lint` v2.15.0) | 0 | "0 issues." for `.` and `acpext` |
  | `go mod tidy -diff` in `.` and `acpext` | 0, 0 | no dependency moved |
  | `actionlint .github/workflows/ci.yml` | 0 | clean |
  | `make check-records`, `make markdownlint` | 0, 0 | 0 issues |
  | `make preflight` through `run-heavy` (ended 09:39:19, 456 s) | 0 | passed |
  | `go test -count=1 ./...` in `.` and in `acpext` through `run-heavy` (ended 09:40:44, 84 s) | 0 | passed |

  `make preflight` ran: gofmt, go mod tidy, pre-add-check, go vet, staticcheck, golangci-lint, govulncheck,
  check-acpext-deps, fix-check, archtest, apidiff, verify-build-metadata, check-records, check-icon, markdownlint. No
  Go file changed, so there was no further pre-add. `-race` is not run by CI's `ci.yml`, so it was not run.
* 0023-MADR D3 also decided the pin: it received a dated pointer line to 0061-MADR, as 0007-MADR did.
  0059-MADR D5's sentence "The golangci-lint pin stays `v2.14.0`" is now out of date; it has no pointer line, since
  none was asked for.
* 2026-11-20, close-out: the pin move landed as push `a3a2d7c` ("chore(lint): bump golangci-lint pin from v2.14.0 to
  v2.15.0"), an ancestor of `origin/main`. CI ran on that exact commit and passed: run `44358064517` (the CI workflow,
  18 m 51 s) and `44358063294` (Docs, 14 s), both success, 2026-11-18T17:23. Every later push carries the move and
  stayed green (the `chore(deps)` bump at 2026-11-19T14:35, run `44545387817`, 20 m 39 s; the 0062 close-out and 0063
  docs pushes of 2026-11-19T15:19 and 2026-11-20T00:40). All four Scope files read `v2.15.0` at the tip (`Makefile`
  comment and install hint, `.github/workflows/ci.yml`, `AGENTS.md`, `docs/guides/developing.md`), and 0007-MADR and
  0023-MADR carry their dated pointer lines. Goal met; nothing in Scope remains. `docs/README.md`'s 0061 PLAN row moves
  to implemented in the same working tree; that update rides the repository's next docs change, which is in flight.
