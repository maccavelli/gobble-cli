---
status: complete
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

*(2026-11-20, Deviation 1: the last clause is not met at `09a6adf`: `docs/architecture.md:207` and `:212` still name
`golint`. The status went from `implemented` to `in-progress`, the plan vocabulary's word for a plan with work left.)*

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
  *(2026-11-20, Deviation 1: `docs/architecture.md` was not moved, and the MADR's Confirmation grep for `golint` is
  not recorded as run.)*
* 2026-11-20, gates: `gofmt -l .` clean; `go vet ./...` in `.`, in `acpext`, and with `GOOS=windows`, all exit 0;
  `go mod tidy -diff` in `.` and in `acpext`, no dependency moved; `make lint` "0 issues." for `.` and for `acpext`
  with the trio; `make markdownlint` 0 issues (59 files); `make preflight` through `run-heavy` passed — with
  `PYTHON=/opt/homebrew/bin/python3.12`: the first run died at `verify-build-metadata` on this host's python3.9
  (`datetime` has no `UTC`), which is an environment drift, not a gate failure; the house `PYTHON ?= python3`
  override covers it.

### Deviation 1 (2026-11-20): an independent review finds the goal not met and four undecided states

* **Found** by a read-only review at `b6d0fc5`, each point confirmed again at `09a6adf` (line numbers are those at
  `09a6adf`); the MADR's amendment A1 records them as R1 to R6:
  * F1: `docs/architecture.md:207` (`golint` in the `tool` directive list) and `:212` ("gofmt and golint") still name
    the old step. The Goal's last clause and the MADR's Confirmation grep are not met, and this record did not show
    the grep run.
  * F2: `scripts/go-precheck.sh:77`, `tail -40` over both modules' output joined, drops the root module's lines first,
    with no notice. The exit status stays 1.
  * F3: `scripts/go-precheck.sh:72-78`, a clean run writes `golangci-lint:` and `0 issues.` per module to stderr.
  * F4: `scripts/go-precheck.sh:37-39`, an add with no Go file now lints whole-tree and runs `govulncheck` (network);
    before, it exited 0 at once. The MADR did not state it.
  * F5: `Makefile:196` and `:199`, `make preflight` lints both modules twice.
  * F6: `docs/README.md`'s MADR row had a title other than the MADR's H1 and said "staged"; MADR Decision Drivers
    called the rule "staged files (whole-tree…)".
* **Decision so far.** F6 is fixed in this records change (the README row, and dated annotations in the MADR). The
  status is `in-progress` while F1 stands. F2 to F5 were states the MADR did not decide: rows S1, S3, S5, S6, S11 and
  S12 of the MADR's state table. The owner decided them on 2026-11-20 (see "Execution record, P2 approval" below).
  Nothing in the tree outside the records changes until the owner's go-ahead for the phase below.
* **Files added to scope by P2 below:** `docs/architecture.md` (F1), `scripts/go-precheck.sh` (F2 to F4),
  `Makefile` (F5, if the answer moves a step). Each is named again in the step that touches it.

## P2 — the review's follow-up (approved 2026-11-20; build after the owner commits the records)

*The Implementation Steps 1 to 3 above, executed on 2026-11-20, are this plan's first phase, P1. Every check below
runs on a `git archive` copy of the tip under `<scratch>`, never on the tree; every plant is made on such
a copy, and the copy is discarded after. Steps that name rows state the outcome the owner decided on 2026-11-20 (the
MADR's state table, amendment A1); none waits on an answer.*

**Exit criterion: no blocker, every row tested, plants caught.**

1. **P2-0, re-ground.** Read at the phase's start commit every line this phase cites: `scripts/go-precheck.sh:37-39`,
   `:69`, `:72-78`; `Makefile:196`, `:199`, `:200`; `docs/architecture.md:207`, `:212`. Record whether each holds,
   and correct by annotation where one moved.
2. **P2-1, the docs (R1).** `docs/architecture.md:207`: drop `golint` from the `tool` directive list. `:212`: `make
   pre-add-check` is gofmt on the named Go files, golangci-lint over both modules whole, and `govulncheck`.
   *Check:* `grep -rn golint` over the non-record files (excluding `docs/decisions/` and `docs/reports/`) lists only
   the hits named in the step's record (the two history comments in `scripts/go-precheck.sh` and `docs/README.md`'s
   0064 rows); any other hit fails. *Plant:* on a scratch copy, `golint` written back into
   `docs/guides/developing.md`: the check lists it and fails.
3. **P2-2, the output cap (S5, S6; R2, decided).** No `tail`; each
   module's output printed whole under `golangci-lint (<module>):`, as `make lint` labels it. *Check:* a scratch copy
   with 30 planted missing-comment findings in `.` and 30 in `acpext`; stderr names all 60 `file:line`s, each under
   its module's label; exit 1. *Plant:* `tail -40` restored: the root module's findings are missing and the check
   fails.
4. **P2-3, the clean run's output (S3; R3, decided).** A module's lint output is printed only
   when that module's run fails. *Check:* a clean scratch copy; stderr holds neither `golangci-lint` nor `0 issues.`;
   exit 0; stdout's last line is `go-precheck: … file(s) clean`. *Plant:* the unconditional print restored: the check
   fails on the header.
5. **P2-4, an add with no Go file (S1, S11; R4, decided).** Arguments that hold no Go file
   exit 0 at once, as before 0064. *Check:* `scripts/go-precheck.sh README.md` on a scratch copy, with a stub
   `golangci-lint` first on `PATH` that writes a marker file, and `HTTPS_PROXY` set to a closed local port: exit 0,
   no marker, no `govulncheck` line. *Plant:* `:39`'s current behaviour (go on to lint): the marker exists and the
   check fails.
6. **P2-5, `make preflight` (S12; R5, decided).** Preflight's `pre-add-check` line sets an
   environment variable that skips the script's lint step (and `GO_PRECHECK_SKIP_VULN=1`), because `lint` and
   `vulncheck` follow it. *Check:* on a scratch copy, the `pre-add-check` command exactly as `make -n preflight` prints
   it, run with a counting stub `golangci-lint` first on `PATH`: the stub counts 0 calls from the script;
   `make lint` alone counts 2 (one per module). *Plant:* the variable dropped from that line: the script's count is 2
   and the check fails.
7. **P2-6, the decided rows (S2, S4, S7, S9, S10).** On scratch copies:
   * S2: no arguments on a clean copy, exit 0; with one unformatted Go file added, exit 1 and the file named.
     *Plant:* the `git ls-files '*.go'` loop (`:41-43`) removed: the unformatted file passes and the check fails.
   * S4: one planted missing-comment file: exit 1, its `file:line` named. *Plant:* `|| failed=1` removed from `:72`:
     exit 0 and the check fails.
   * S7: `PATH` without golangci-lint: `go-precheck: golangci-lint not found in PATH.`, exit 2, no `govulncheck` line.
     *Plant:* the `need golangci-lint || exit 2` line removed: the exit is not 2.
   * S9: `--config` pointed at a file that does not parse: exit 1, the message shown. *Plant:* `|| failed=1` removed:
     exit 0.
   * S10: one Go file, `HTTPS_PROXY` at a closed local port: exit 1 from `govulncheck`; with
     `GO_PRECHECK_SKIP_VULN=1`, `govulncheck: skipped` and exit 0. *Plant:* the `case $?` that sets `failed` removed:
     exit 0 offline, and the check fails.
8. **P2-7, records.** The MADR's state table is already `decided` on every row and amendment A1 `accepted` (records
   change of 2026-11-20); this step checks that each row's pin still names the step that ran it and marks it run. This plan's execution record numbers each pass of the
   phase and quotes each check's and plant's output. `docs/README.md`'s 0064 rows. If P2-2 to P2-5 change what the
   script prints or skips, `AGENTS.md`'s "Pre-add checks" and `docs/guides/developing.md` say the same.
9. **P2-8, gates.** `bash -n scripts/go-precheck.sh`; `make lint`; `make check-records`; `make markdownlint`;
   `make preflight` through `run-heavy` or `run-on-host`.

### Execution record, P2 approval (2026-11-20)

* The owner, relayed by the coordinator, ruled on 2026-11-20 on R2 to R5 and on this phase. The labels, quoted only as
  labels: R2 (S5, S6) "Print each module whole (Recommended)"; R3 (S3's output) "Nothing; print a module only when it
  fails (Recommended)"; R4 (S1, S11) "Exit 0 at once, as before 0064 (Recommended)"; R5 (S12) "Preflight tells the
  script to skip lint and vuln (Recommended)".
* Phase P2 approved: "Approve P2; build it after I commit the records (Recommended)". Nothing in P2 is built until the
  owner has committed these records and says to proceed. ~~P2 has had no pass yet; each pass will be numbered here.~~ *(2026-11-20: pass 1 is recorded below.)*
* Row-to-step map checked 2026-11-20: S1, S11 to P2-4; S3 to P2-3 (output) and the P1 gates (exit); S5, S6 to P2-2;
  S12 to P2-5; S2, S4, S7, S9, S10 to P2-6. Each step names its plant. S8 is `out of scope`. No decided row lacks a
  step. R1 (the docs) is P2-1.

### Execution record, P2 pass 1 (2026-11-20)

* **P2-0, re-ground at `f7677e2`.** Every cited line holds: `scripts/go-precheck.sh:37-39` (the "no Go files to
  format." line, then lint), `:69` (`need golangci-lint || exit 2`), `:72-78` (`lint_out`, `tail -40`),
  `Makefile:196`, `:199`, `:200`, `docs/architecture.md:207`, `:212`. None moved.
* **Files changed (unstaged):** `scripts/go-precheck.sh`, `Makefile`, `docs/architecture.md`, `AGENTS.md`,
  `docs/guides/developing.md`, this plan, the MADR's pin cells, `docs/README.md`'s two 0064 rows.
* **What the script now does.** Arguments with no Go file `exit 0` at once (S1, S11). The lint step prints a module's
  output whole, under `golangci-lint (<module>):`, only when that module's run fails (S3, S5, S6; no `tail`). A new
  environment variable, `GO_PRECHECK_SKIP_LINT=1`, skips the lint step and prints `golangci-lint: skipped
  (GO_PRECHECK_SKIP_LINT=1)`; the Makefile's preflight line sets it with `GO_PRECHECK_SKIP_VULN=1` (S12).
* **Method.** Every check and plant ran on a copy of `git archive HEAD` under `<scratch>` with only these files
  copied over (the scratch copy given its own throwaway `git init`); `BASH_ENV` was unset for the runs because the
  host's value re-orders `PATH` and defeats a stub. Full output is in `<scratch>` files named for the row.
* **P2-1 (R1).** `docs/architecture.md:207` no longer lists `golint`; `:212` now says gofmt on the named files,
  golangci-lint over both modules whole, then govulncheck, with the two skip variables and the no-Go-file exit.
  `grep -rn golint --exclude-dir=.git --exclude-dir=decisions --exclude-dir=reports .` lists exactly four hits:
  `docs/README.md:145` and `:146` (the 0064 rows) and `scripts/go-precheck.sh:10` and `:72` (the history comments).
  Plant: `golint` appended to `docs/guides/developing.md` on a scratch copy: the same grep also lists
  `docs/guides/developing.md:315:golint`, which fails the check.
* **P2-2 (S5, S6).** 30 planted missing-comment findings in `.` and 30 in `acpext`: exit 1; stderr has
  `golangci-lint (.):` and `golangci-lint (acpext):`, 30 `plant_R.go:<line>:<col>` and 30 `plant_A.go:<line>:<col>`
  lines. Plant (the old joined `lint_out` and `tail -40` restored): 0 `plant_R.go` and 12 `plant_A.go` lines, so the
  root module's findings are the ones dropped; the check, which requires 30 and 30, fails.
* **P2-3 (S3, output).** Clean copy, one Go file, `GO_PRECHECK_SKIP_VULN=1`: exit 0; stderr is only
  `govulncheck: skipped (GO_PRECHECK_SKIP_VULN=1)`, no `golangci-lint` and no `0 issues.`; stdout's last line is
  `go-precheck: 1 file(s) clean (gofmt, golangci-lint, govulncheck).` Plant (unconditional print restored): stderr
  gains `golangci-lint (.):` / `0 issues.` and the same for `acpext`; the check fails.
* **P2-4 (S1, S11).** `scripts/go-precheck.sh README.md`, a stub `golangci-lint` first on `PATH` writing a marker,
  `HTTPS_PROXY` at a closed local port: exit 0, no marker, no `govulncheck` line. Plant (`:39`'s old line, go on to
  lint): exit 1, marker written, `govulncheck: fetching vulnerabilities: ... connection refused`.
* **P2-5 (S12).** The command `make -n preflight` prints for the pre-add step
  (`GO_PRECHECK_SKIP_LINT=1 GO_PRECHECK_SKIP_VULN=1 bash scripts/go-precheck.sh <every Go file>`), with a counting stub
  first on `PATH`: exit 0, the stub counted 0 calls; `make lint` alone counted 2. Plant (the variable dropped from the
  Makefile line): the script's count is 2.
* **P2-6 (S2, S4, S7, S9, S10).**
  * S2: no arguments, clean copy, real network: exit 0, `go-precheck: 700 file(s) clean (gofmt, golangci-lint,
    govulncheck).` With one unformatted Go file added and lint skipped (so only the gofmt step can speak): exit 1,
    `gofmt: these files are not formatted` and the file named. Plant (the `git ls-files '*.go'` loop emptied): exit 0.
    *(Without the lint skip the plant still exits 1, because golangci-lint's own gofmt formatter flags the file; the
    skip isolates the step the plant removes.)*
  * S4: one file with two planted exported functions: exit 1, `plant_R.go:3:1: exported: exported function
    PlantedRN1 should have comment or be unexported (revive)` under `golangci-lint (.):`. Plant (`failed=1` removed
    from the failure block): exit 0.
  * S7: `PATH` of symlinks to git, gofmt, go, sed, bash and a few coreutils, no golangci-lint:
    `go-precheck: golangci-lint not found in PATH.`, exit 2, no `govulncheck` line. Plant (the `need` line removed):
    exit 1, `golangci-lint: command not found` per module.
  * S9: `.golangci.yml` replaced by an unparseable file in the copy: exit 1, `Error: can't load config: ... yaml:
    line 1: did not find expected ',' or ']'` under `golangci-lint (.):` and `(acpext):`. Plant (`failed=1`
    removed): exit 0 with the same message printed.
  * S10: one Go file, `HTTPS_PROXY` closed: exit 1, `govulncheck: fetching vulnerabilities: ... connection
    refused`; with `GO_PRECHECK_SKIP_VULN=1`: `govulncheck: skipped`, exit 0. Plant (the `case` arm that sets `failed`
    emptied): exit 0 offline, the message still shown.
* **P2-7 (records).** The ten "not yet run" pins in the MADR's state table now read "run 2026-11-20, P2 pass 1"; each
  names the step that ran it (rows and steps as mapped in the approval record above, unchanged). The
  `docs/README.md` 0064 rows are updated. `AGENTS.md` ("Pre-add checks") and `docs/guides/developing.md` say the same
  as the script (the no-Go-file exit, per-module output on failure, `GO_PRECHECK_SKIP_LINT`).
* **Not done:** S8 is out of scope by the MADR. The script's closing line still reads "(gofmt, golangci-lint,
  govulncheck)" when a skip variable was set; the decided rows do not cover its wording, so it is left as P1 wrote it.
* **P2-8, gates (pass 1).** `bash -n scripts/go-precheck.sh` exit 0. In a `git clone` of `f7677e2` with the files above
  copied over (so the other builder's test files were not in it): `make lint` exit 0, "0 issues." for `.` and
  `acpext`; `make preflight` through `run-heavy` with `PYTHON=/opt/homebrew/bin/python3.12` (the host's default
  `python3` drift recorded in P1) exit 0, "preflight passed", its pre-add step printing `golangci-lint: skipped` and
  `govulncheck: skipped`, and `lint` and `vulncheck` running once each after it. In the tree: `make check-records`
  exit 0; `make markdownlint` "0 issues in 0 files" over 59 files.
* **Docs step.** `grep -rn golint --exclude-dir=.git --exclude-dir=decisions --exclude-dir=reports .` after the
  change lists only `docs/README.md:145`, `docs/README.md:146`, `scripts/go-precheck.sh:10` and `scripts/go-precheck.sh:72`
  (the two index rows and the two history comments), which is the P2-1 allowance; the MADR's Confirmation item 3 is
  met. *(Line `:72` is `:67` of the earlier amendment's count; the comment moved when the header grew.)*

### P2 pass 2 (2026-11-20): the independent review of pass 1, and two record corrections

* **Review verdict on pass 1: FAIL, on two record statements; no code finding.** The reviewer exercised every decided row S1 to S7 and S9 to S12 on its own scratch copy of `f7677e2` plus the eight P2 files, under bash 3.2, without the builder's harness. Every row's observed exit status and output matched its decided outcome, and one break of the reviewer's own per row was detected. No behaviour change in `scripts/go-precheck.sh` or the `Makefile` lacks a row. No shell defect new in P2. `make -n preflight` shows both skip variables on the pre-add line, followed by preflight's own `lint` and `vulncheck`, so nothing is skipped overall; CI runs `make preflight`; no other caller of the script exists.
* **B1 (corrected in this pass).** 0064-MADR rows S4 and S9 said findings and errors print "under `golangci-lint:`". Since P2-2 the header is `golangci-lint (<module>):`, as R2 and row S6 decide. Both Outcome cells carry a dated A1 annotation; the pass-1 record built the new header without saying those two rows' wording was superseded.
* **B2 (corrected in this pass).** `docs/README.md`'s 0064 PLAN row said every row S1 to S12 was tested. S8 is out of scope and was not tested. The row now says every decided row.
* **Gates seen by the reviewer:** `bash -n scripts/go-precheck.sh` exit 0 under bash 5 and 3.2; `scripts/check_records.py --check-all` exit 0; `make lint` on its scratch copy exit 0, "0 issues." for both modules. `make preflight` was not re-run by the reviewer; pass 1's run stands. The script was not run under Git-bash on Windows.
* **Host note.** On a machine whose shell start-up file rebuilds `PATH`, a caller cannot hide or replace `golangci-lint` through `PATH`; the scratch runs unset that start-up variable. This comes from the host's setup, not from the script, and the script behaves the same for a user there.
* **Follow-ups, not built in P2 (no decided row covers them; they wait for the owner):** (1) `scripts/go-precheck.sh:62` ignores gofmt's own exit status, as it did before 0064: with both skip variables set, a file that does not parse gives exit 0 and "1 file(s) clean"; preflight still catches it later through vet and lint. (2) The closing line (about `:105`) still lists "(gofmt, golangci-lint, govulncheck)" when steps were skipped. (3) `AGENTS.md` says `FILES` that name no Go file exit 0; a path to a deleted `.go` file also exits 0, which it does not say.
* **Pass count:** this is the second build pass of P2. It changes records only. The plan stays `in-progress` until the owner's commit is pushed and CI is green.

### Close-out (2026-11-20)

* P2 landed in `93ecb5e`, the owner's commit and push of 2026-11-20, together with the 0036 test-only step.
* Before the commit the coordinator ran its own gate on the staged tree: 20 checks, all passing (the exact file set, test files only among Go changes, no committed record line lost, no identifier, `gofmt`, `go vet`, `make lint` on both modules, `go test` of `llm/provider`, `internal/cli` and `acpserver`, the records check, markdown lint). The coordinator also ran `make preflight` on a scratch clone of the staged tree through the heavy-run queue: exit 0, "preflight passed".
* CI on `93ecb5e`: workflow CI, run `44830571357`, concluded success: `validate` on ubuntu-24.04, macos-15 and windows-2025, `test-isolation`, `sbom`, `build` and its five identity jobs all succeeded; `release` was skipped (no tag). Workflow Docs, run `44830570773`, succeeded.
* Not run anywhere by hand: the pre-add script under Git-bash on Windows. CI's windows-2025 `validate` job runs `make preflight`.
* The phase exit criterion "no blocker, every row tested, plants caught" is met: pass 1 built it, the independent review confirmed every decided row and failed two record statements (B1, B2), pass 2 corrected them; two passes in all.
* The three follow-ups listed in "P2 pass 2" remain open and are not part of this plan unless the owner pulls them in.
* The plan is complete.
