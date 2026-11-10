---
status: complete
date: 2026-11-10
associated-madr: "0050-MADR-refresh-sibling-pins-to-current-releases.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement pinning each sibling library's current release

Associated MADR: [0050-MADR-refresh-sibling-pins-to-current-releases.md](0050-MADR-refresh-sibling-pins-to-current-releases.md)

## Goal

Every sibling pin in `go.mod` is the library's current release, with CI and the docs naming the same versions. The plan
is complete when the MADR's Confirmation 1 to 5 is met.

## Scope

**Base.** `main` at `cdfaead`, the tree clean; origin/main `ece13c4`, P2 onwards of 0049 not yet pushed.

**In scope:**

* **P1, the two tags that exist:**
  * `go.mod` and `go.sum`, for go-selfupdate-lib `v1.26.0` and, under Q1 (a), go-llmprovider-sdk `v1.29.0`;
  * `.github/workflows/ci.yml`'s two `uses:` pins;
  * `README.md`, `docs/architecture.md`, and the comment at `llm/provider/provider.go:736`;
  * under Q1 (a), the 0049 pair's P5 and D10 amendments;
  * under Q2 (b) or (c), `internal/cli/update.go`, `internal/cli/update_test.go`, and 0048-MADR A2-3's amendment.
* **P2, acp-go-sdk2 `v1.35.0`,** once tagged: `go.mod`, `go.sum`, `README.md` and `docs/architecture.md`.
* `docs/README.md`'s rows.

**Out of scope:** adopting go-llmprovider-sdk `v1.29.0`'s fields (0049-PLAN P5); go-selfupdate-lib's new sentinels
(MADR "More Information"); any change acp-go-sdk2 `v1.35.0` needs beyond its pin, which is amended in first; a push or
a tag.

## Conventions

* **C1.** Each phase first re-reads `git status`, `HEAD`, `origin/main` and `go.mod`'s pins.
* **C2.** Commands write to a scratch log, and their exit status is taken before any filter. Tests run with the proxies
  at `127.0.0.1:9`, except `llm` and `llm/provider` (0049-PLAN C3).
* **C3.** A module moving that the step does not name, a `check-libs` failure, or a test failing is a deviation. The
  phase stops and prompts.
* **C4. Gates:** `make pre-add-check` on any changed Go file; `make preflight`; `make race`; `go test ./...` and the
  race tests on the Linux machine and the Windows laptop; `make check-records markdownlint`; the identifier scan. The
  agent stages, and the owner commits with `git commit --no-edit`.

## Implementation Steps

### P0 — the records

1. The owner's answers to Q1 and Q2 go into the MADR, which becomes `accepted`, and this plan becomes `in-progress`.
   Under Q1 (b), P1 drops go-llmprovider-sdk; under Q2 (a), P1 drops the hint's files.
2. `docs/README.md`'s rows; `make check-records markdownlint`. Staged; the owner commits.

### P1 — go-selfupdate-lib `v1.26.0`, go-llmprovider-sdk `v1.29.0`

1. **C1.** Then `scripts/check_libs.py --lib go-selfupdate-lib --tag v1.26.0` and `--lib go-llmprovider-sdk --tag
   v1.29.0`, again at P1's base: `0 failing checks` each.
2. `go get github.com/maccavelli/go-selfupdate-lib@v1.26.0 github.com/maccavelli/go-llmprovider-sdk@v1.29.0`, then
   `go mod tidy`. `git diff go.mod` changes those two lines alone; `go.sum` changes only their hashes.
3. `.github/workflows/ci.yml:171` and `:183` move to `7c677842dface4b14b65b2e8b46aa0a7543bed13 # v1.26.0`. Before the
   edit, `gh api repos/maccavelli/go-selfupdate-lib/git/ref/tags/v1.26.0` is peeled again, to confirm the commit.
4. `README.md:155-163` and `docs/architecture.md:189-194` name the new tags; `llm/provider/provider.go:736`'s comment
   names `v1.29.0`.
5. Under Q1 (a), the 0049 records are amended:
   * 0049-PLAN P5: step 1 is "done by 0050-PLAN P1". Steps 2 to 4 name `v1.29.0`'s `APIError.ResetAt`,
     `APIError.Limit` (with its window, plan, per-day and quota parts) and `ClassifyStreamFailureBody`, and add
     `BudgetLimitError` to D4's table.
   * 0049-MADR D10: "a phase of the PLAN pins it" becomes "0050 pins `v1.29.0`; P5 maps its fields".
   * P5 itself does not run here.
6. Under Q2 (b) or (c), the hint changes as answered, `TestUpdateSymlink` is updated, and it is seen failing on a
   scratch copy against the old hint. 0048-MADR A2-3 gains a dated amendment.
7. **Gates** (C4). Staged; the owner commits.

### P2 — acp-go-sdk2 `v1.35.0`

1. Wait for the owner's tag. `gh api repos/maccavelli/acp-go-sdk2/tags` lists `v1.35.0`, and `go list -m
   github.com/maccavelli/acp-go-sdk2@v1.35.0` succeeds through the proxy.
2. Read its release notes or `CHANGELOG.md`. Then `scripts/check_libs.py --lib acp-go-sdk2 --tag v1.35.0`: `0 failing
   checks`. A failure, or a change gobble must make, is a deviation (C3), amended in before any code.
3. `go get github.com/maccavelli/acp-go-sdk2@v1.35.0`, `go mod tidy`: that line alone, and its hashes.
4. `README.md:155` and `docs/architecture.md:189` name `v1.35.0`.
5. **Gates** (C4). Staged; the owner commits. The MADR's Completion section and this plan's `complete` follow, with
   the README rows.

### P0 record (2026-11-10)

C1: `HEAD` `cdfaead`, `origin/main` `ece13c4`, the tree clean but for this pair and its `docs/README.md` rows;
pins as the MADR's F1. The owner's answers: Q1 (a), Q2 (a), and P1 at once. The MADR is `accepted` and this plan
`in-progress`. Under Q2 (a), P1 step 6 is not run, and `internal/cli/update.go` and `update_test.go` leave the
scope. P0 and P1 are staged together, as the owner chose.

### P1 record (2026-11-10)

C1: `HEAD` `cdfaead`, `origin/main` `ece13c4`; go-llmprovider-sdk `v1.28.0`, go-selfupdate-lib `v1.25.0`, acp-go-sdk2
`v1.34.2` and go-tui-lib `v2.12.0` required. No deviation.

1. **`check_libs.py` at P1's base:** `--lib go-selfupdate-lib --tag v1.26.0` and `--lib go-llmprovider-sdk --tag
   v1.29.0` each gave `0 failing checks; scratch removed`. Each covered get and tidy, version, modules, build, vet for
   linux, darwin and windows, test, staticcheck, lint, api, govulncheck and advisories.
2. **The pins:** `go get` printed `upgraded github.com/maccavelli/go-llmprovider-sdk v1.28.0 => v1.29.0` and `upgraded
   github.com/maccavelli/go-selfupdate-lib v1.25.0 => v1.26.0`. `go mod tidy` exit 0. `git diff go.mod` changes those
   two lines alone, and `go.sum` four lines, their two hashes each.
3. **CI:** `v1.26.0` peeled again, `tag -> commit 7c677842dface4b14b65b2e8b46aa0a7543bed13`. `ci.yml:171` and `:183`
   now name `@7c677842dface4b14b65b2e8b46aa0a7543bed13 # v1.26.0`; `875dec3` no longer appears in the file.
4. **Docs:**
   * `README.md`'s Stack names `v1.29.0` and `v1.26.0`; `docs/architecture.md`'s requirement rows name them.
   * `llm/provider/provider.go:736`'s comment names `v1.29.0`. Its claim holds there: `opencodeQuotaTypes`
     (`llmprovider/api_error.go:512`) still names every code in the four sets.
5. **0049 amended** (Q1 (a)):
   * 0049-PLAN P5 gains a dated amendment. Its step 1 is struck and marked done by this phase. The tag's fields and
     the bare pin's three changes are named. Steps 2 to 4 are to be amended to #15's "How to implement" before P5
     runs, and #15's two open choices go to the owner first.
   * 0049-MADR D10's "a phase of the PLAN pins it" is struck: 0050 pins it, P5 maps its fields.
   * The MADR's F6 here names the same three changes and cites #15, which the owner filed while this phase ran.
6. **Q2 (a):** not run; `internal/cli/update.go` is unchanged.
7. **Gates:**
   * `make pre-add-check FILES="llm/provider/provider.go"`: `1 file(s) clean (gofmt, golint, govulncheck)`.
   * macOS: `make preflight` exit 0, `preflight passed`; `make race` exit 0, 39 ok.
   * The Linux machine and the Windows laptop (`TMP`/`TEMP` set), a scratch copy each: acpext's tests ok; `go test
     -count=1 ./...` 38 ok; `-race` 38 ok; on each host, exit 0 and no `FAIL`.
   * `make check-records markdownlint` after this record: exit 0. The identifier scan of the diff (the local account
     name, the dev hosts' names, the org's mail domain, and real-machine home and temporary paths): nothing found.

### P2 record (2026-11-10)

C1: `HEAD` `87ab1da` (P0 and P1), `origin/main` `ece13c4`, the tree clean; go-llmprovider-sdk `v1.29.0`,
go-selfupdate-lib `v1.26.0`, go-tui-lib `v2.12.0` and acp-go-sdk2 `v1.34.2` required. No deviation.

1. **The tag:** acp-go-sdk2's tags listed `v1.35.0` (an annotated tag, `8e99166`), and
   `go list -m github.com/maccavelli/acp-go-sdk2@v1.35.0` through the proxy succeeded. The owner tagged it while P1
   ran.
2. **Its notes and `check_libs.py`:**
   * **The notes** (acp-go-sdk2's `docs/reports/0052-REPORT-release-notes-v1-35-0.md`): seven compatible additions and
     nothing incompatible, all `ToolCallStream`. The ACP schema stays at `1.25.0`.
   * **Two behaviour changes:** a prompt turn's unfinished tool-call streams are written before its answer, and the
     default message size is 32 MiB, up from 10 MiB.
   * **What gobble uses:** it starts no `ToolCallStream` and sets no message size anywhere; its `-32600` tests are
     gobble's own refusals. So neither change reaches it.
   * **`check_libs.py`:** `--lib acp-go-sdk2 --tag v1.35.0` gave `0 failing checks` before P1's commit, and again at
     `87ab1da` with `--from v1.34.2`.
3. **The pin:** `go get` printed `upgraded github.com/maccavelli/acp-go-sdk2 v1.34.2 => v1.35.0`, and `go mod tidy`
   exit 0. That `go.mod` line alone changed, and two `go.sum` lines. The pin was moved before step 2's second check,
   which read the committed base, not the tree.
4. **Docs:** `README.md:155` and `docs/architecture.md:189` name `v1.35.0`. `v1.34.2` remains only in records, which
   keep their history.
5. **Gates:**
   * No Go file changed, so there was no pre-add check.
   * macOS: `make preflight` exit 0, `preflight passed`; `make race` exit 0, 39 ok.
   * The Linux machine and the Windows laptop (`TMP`/`TEMP` set), a scratch copy each: acpext's tests ok; `go test
     -count=1 ./...` 38 ok; `-race` 38 ok; on each host, exit 0 and no `FAIL`.
   * `make check-records markdownlint` after this record: exit 0. The identifier scan of the diff found nothing.

Every sibling pin is now its library's current release. The MADR's Confirmation 1 to 5 is met (its "Completion"
section), and this plan is `complete`.

## Verification

| Confirmation | Where |
| :--- | :--- |
| 1 the pins, tidy clean | P1 step 2, P2 step 3 |
| 2 `check-libs` per library | P1 step 1, P2 step 2 |
| 3 CI and docs name one version | P1 steps 3 and 4, P2 step 4 |
| 4 the gates on three hosts | P1 step 7, P2 step 5 |
| 5 0049's P5 and D10 amended | P1 step 5 |

## Rollout and Rollback

* **Rollout.** Each phase's commit, pushed by the owner. The pins reach users with the next gobble release.
* **Rollback.** Reverting a phase's commit restores the earlier pins, the workflow commits and the docs. Each library
  is a minor release within `v1` with compatible changes (apidiff, F2), so nothing gobble wrote depends on the newer
  pins until 0049's P5 maps go-llmprovider-sdk's fields.
