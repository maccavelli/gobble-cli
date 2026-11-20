---
status: accepted
date: 2026-11-20
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Retire per-file golint from the pre-add; restore the trio in revive

## Context and Problem Statement

The fleet decision — dotfiles
`docs/decisions/0028-MADR-retire-per-file-golint-from-the-fleet-pre-add-rule.md`
(2026-11-20, cited by name; a relative link cannot reach across repositories)
— retired the per-file `golint` pre-add rule and standardized the pre-add lint
on golangci-lint-on-staged. Its 2026-11-20 amendment then corrected a fact:
the revive carrier rules (`exported`, `package-comments`, `var-naming`) are
explicitly enabled only in magic-cli-remote, go-selfupdate-lib,
go-llmprovider-sdk and go-tui-lib, and NOT in this repository, whose
`revive.rules` is an explicit list of twenty-one rules.

*(2026-11-20, amendment A1: "golangci-lint-on-staged" is the fleet record's name for its rule. This repository's
decision is whole-tree lint over both modules, as Option A, the Decision Outcome and `scripts/go-precheck.sh` say; no
step here lints only the staged files. See "Amendments".)*

gobble-cli is the one repository still executing the retired rule:
`scripts/go-precheck.sh` step 2 runs `go tool golint` per file (from the
module `tool` directive), named in the script banner, in this repository's
`AGENTS.md` pre-add paragraph and in `docs/guides/developing.md`; `make
preflight` feeds it every tracked file. The pre-add lint and `make lint` are
two linters over two scopes — the double coverage the fleet decision found.

## Decision Drivers

* The fleet rule now reads: pre-add lint = golangci-lint over the packages of
  the staged files (whole-tree, as the sibling wrappers run it); the golint
  fallback is gone.
  *(2026-11-20, amendment A1: "staged files" here describes the fleet rule's words, not this record's choice, which
  lints both modules whole whatever was staged; the parenthesis is the decision.)*
* Dropping the per-file step must not drop exported-comment coverage: this
  repository's `revive.rules` list omits golint's three carrier rules, so the
  same record adds them — the exact replacement magic-cli-remote's
  `0131-PLAN-replace-golint-with-golangci-lint.md` made (it enabled exactly
  that trio behind its planted-finding fence).
* `0059-MADR` D5 requires the wrapper plants re-run when the pin changes;
  they re-ran at the current pin `v2.15.0` on 2026-11-18
  ([0061-PLAN-move-golangci-lint-to-v2-15-0.md](0061-PLAN-move-golangci-lint-to-v2-15-0.md),
  steps 3-6 at full count) and the pin moved the same day
  ([0061-MADR-move-golangci-lint-to-v2-15-0.md](0061-MADR-move-golangci-lint-to-v2-15-0.md),
  superseding 0007-MADR and 0023-MADR D3 for the version only). This record
  changes revive's ruleset, not the pin; its own plant covers the trio, and
  the exclusion format does not change, so 0059's text-keyed machinery is
  untouched.
* The pre-add lint must not drift from `make lint`: same command, same
  config, same modules, host GOOS.
* A `git archive` copy of the tip with the trio added lints "0 issues." on
  both modules — the tree is already clean under the new rules.

## Considered Options

* **A. Retire the per-file golint step; the pre-add lint runs the `make lint`
  command over both modules; the trio joins `revive.rules`; the `tool`
  directive goes.** Good, because the pre-add and the tree lint become one
  linter over one scope with no drift; exported-comment coverage is not
  merely preserved but widened (whole-tree at every pre-add instead of
  staged files only); the archived `golang.org/x/lint/golint` leaves the
  `tool` directives. Bad, because a one-file pre-add call now lints
  whole-tree, slower than per-file golint — the trade every sibling
  repository already took.
* **B. Keep golint beside golangci-lint.** Good, because nothing changes.
  Bad, because it is the double coverage the fleet decision found; keeps an
  archived tool pinned; and contradicts the fleet rule this repository's own
  agents follow.
* **C. Retire golint without the trio.** Good, because the config diff is
  smaller. Bad, because exported-comment coverage silently ends: golint was
  this repository's only exported-comment checker, and magic-cli-remote
  faced the identical choice and added the trio.

## Decision Outcome

Chosen option: "A", per the fleet decision and this record; the steps, fences
and gates are in
[0064-PLAN-retire-pre-add-golint-add-the-trio.md](0064-PLAN-retire-pre-add-golint-add-the-trio.md).

* Good, because the pre-add lint and the tree lint are one linter over one
  scope with no drift.
* Good, because exported-comment coverage survives (the trio) and widens:
  whole-tree at every pre-add instead of staged files only.
* Good, because the archived `golang.org/x/lint/golint` leaves the `tool`
  directives; the homedir's golint install becomes a dotfiles matter, as
  0028-MADR already records.
* Bad, because a one-file pre-add call now lints whole-tree — slower than
  per-file golint; the trade every sibling repository already took.
* Neutral, because 0059-MADR D5 concerns pin changes and was already met at
  `v2.15.0` (plants steps 3-6 at full count, 2026-11-18); this record changes
  revive's ruleset, not the pin, and its own plant covers the trio.

## Consequences

* `AGENTS.md`, `docs/guides/developing.md`, the script banner and `go.mod`
  all say one thing: pre-add = plain `gofmt` on the files, golangci-lint over
  both modules whole (the `make lint` command), `govulncheck ./...`.
* The trio can flag packages golint had been covering only at preflight; the
  tree lints clean with them before this lands ("0 issues." both modules on
  the archive copy, 2026-11-20), so nothing needs fixing or suppressing.
* Findings arrive as golangci-lint lines — the format 0059's text-keyed revive
  exclusion is keyed on; the trio are new rules, not re-keyed exclusions, so
  no exclusion text changes.
* CI gates the change after the owner's push; no release.

## Confirmation

* The planted missing-comment file fails the new pre-add naming `file:line`,
  and the tree returns to "0 issues." both modules once fixed — dated in the
  plan's execution record.
* `make preflight` passes with the retirement in place.
* `grep -rn golint` over this repository's non-record files returns nothing
  outside the historical records.
  *(2026-11-20, amendment A1: not met. The plan's execution record does not show this grep run, and at `09a6adf` it
  finds `docs/architecture.md:207` (`golint` in the `tool` directive list) and `:212` (`make pre-add-check`: "gofmt
  and golint"). The two comments in `scripts/go-precheck.sh` (`:10`, `:67`) name the retired step as history.)*

## Amendments

### Amendment A1 (2026-11-20): an independent review of the pre-add lint step

*Status of this amendment: accepted (2026-11-20). Option A stands as accepted. The questions R2 to R5 below were
answered by the owner on 2026-11-20, relayed by the coordinator; each answer is quoted as a label in the state table,
and no row remains `open`. The plan's phase P2 builds them.*

A read-only review at `b6d0fc5` checked this record against the tree, and each finding was confirmed again at
`09a6adf` (the review's base plus `7ee9a94`, `9332cef` and `09a6adf`; of the files below only `docs/architecture.md`
and `docs/README.md` changed since, and the line numbers given are those at `09a6adf`):

* **R1.** `docs/architecture.md:207` and `:212` still name `golint` (Confirmation item 3 above).
* **R2.** `scripts/go-precheck.sh:77` prints `tail -40` of both modules' lint output joined, `.` first and `acpext`
  second, so past 40 lines the root module's findings are the ones dropped, with no notice. The exit status is still
  1 (`:72`).
* **R3.** `scripts/go-precheck.sh:72-78`: golangci-lint prints `0 issues.` on a clean run, so `out` is never empty and
  a clean run writes `golangci-lint:` and one `0 issues.` line per module to stderr.
* **R4.** `scripts/go-precheck.sh:37-39`: arguments that hold no Go file now print "no Go files to format." and go on to
  lint both modules whole and run `govulncheck ./...`, which needs the network. Before this record the script exited 0
  at once (`81670f3^:scripts/go-precheck.sh`, "nothing for this script to say"). This record did not state that
  consequence.
* **R5.** `make preflight` runs golangci-lint over both modules twice: inside `pre-add-check` (`Makefile:196`) and
  again as `lint` (`:199`). (`govulncheck` already ran twice for the root module before this record, `:196` and
  `:200`.)
* **R6.** The index row in `docs/README.md` gave this record a title that is not its H1 and said "staged"; the
  wording at "Decision Drivers" is annotated above. The row is corrected in the same change as this amendment.

#### State table: `scripts/go-precheck.sh`'s lint step

One row per input or state the step can meet. `decided` rows give the outcome and what pins it; the plan's phase P2
maps a step to every row that is not `out of scope`. "Not yet run" means the step has not been executed.

| # | Input or state | Status | Outcome | Pinned by |
| --- | --- | --- | --- | --- |
| S1 | Arguments with no Go file (a docs-only add) | decided (R4) | Exit 0 at once, as before 0064, before lint and `govulncheck`. Owner, relayed by the coordinator, 2026-11-20: "Exit 0 at once, as before 0064 (Recommended)". | 0064-PLAN P2-4 (run 2026-11-20, P2 pass 1). |
| S2 | No arguments (`make pre-add-check` with no `FILES`) | decided | gofmt over every tracked Go file, lint of both modules, `govulncheck`; exit 0 on a clean tree. | 0064-PLAN P2-6 (run 2026-11-20, P2 pass 1). |
| S3 | Go files, zero findings | decided (exit and output; R3) | Exit 0 from the lint step. A module's lint output is printed only when that module's lint fails, so a clean run prints neither `golangci-lint` nor `0 issues.`. Owner, relayed by the coordinator, 2026-11-20: "Nothing; print a module only when it fails (Recommended)". | Exit: 0064-PLAN's gates (2026-11-20). Output: 0064-PLAN P2-3 (run 2026-11-20, P2 pass 1). |
| S4 | Findings, 40 lines or fewer | decided | Exit 1; every finding under `golangci-lint:`, each named `file:line`. *(A1 annotation, 2026-11-20, P2 pass 2: since P2-2 the header is `golangci-lint (<module>):`, per R2 and row S6; "under `golangci-lint:`" describes the script before P2.)* | 0064-PLAN's scratch plant (2026-11-20), repeated as 0064-PLAN P2-6. |
| S5 | Findings over 40 lines | decided (R2) | No `tail`; each module's findings print in full, exit 1. Owner, relayed by the coordinator, 2026-11-20: "Print each module whole (Recommended)". | 0064-PLAN P2-2 (run 2026-11-20, P2 pass 1). |
| S6 | Findings in both modules | decided (R2) | Each module's findings print in full under `golangci-lint (<module>):`, as `make lint` labels them; no module's lines are dropped. Owner, relayed by the coordinator, 2026-11-20: "Print each module whole (Recommended)". | 0064-PLAN P2-2 (run 2026-11-20, P2 pass 1). |
| S7 | golangci-lint not on `PATH` | decided | `go-precheck: golangci-lint not found in PATH.`, exit 2, before `govulncheck` (`:69`). | 0064-PLAN P2-6 (run 2026-11-20, P2 pass 1). |
| S8 | golangci-lint installed at a version other than `v2.15.0` | out of scope | The pin is 0061-MADR's. Neither `make lint` nor this step checks the version, and this record's rule is "the same command". | — |
| S9 | golangci-lint fails with no findings (a configuration or type-check error) | decided | Exit 1; its message under `golangci-lint:` (`\|\| failed=1`, `:72`). *(A1 annotation, 2026-11-20, P2 pass 2: since P2-2 the header is `golangci-lint (<module>):`, per R2 and row S6; "under `golangci-lint:`" describes the script before P2.)* | 0064-PLAN P2-6 (run 2026-11-20, P2 pass 1). |
| S10 | `govulncheck` with no network, Go files in the add | decided | `govulncheck`'s failure, exit 1; `GO_PRECHECK_SKIP_VULN=1` skips it. Unchanged by this record (`AGENTS.md`, "Pre-add checks"). | 0064-PLAN P2-6 (run 2026-11-20, P2 pass 1). |
| S11 | `govulncheck` with no network, no Go file in the add | decided (follows S1; R4) | Exit 0: the script exits before `govulncheck` runs, so the network is not needed. Owner, relayed by the coordinator, 2026-11-20: "Exit 0 at once, as before 0064 (Recommended)". | 0064-PLAN P2-4 (run 2026-11-20, P2 pass 1). |
| S12 | The script run from `make preflight` | decided (R5) | Preflight's `pre-add-check` invocation sets a skip variable for the script's lint and vulnerability steps, since preflight's own `lint` and `vulncheck` steps follow; lint runs once per module. Owner, relayed by the coordinator, 2026-11-20: "Preflight tells the script to skip lint and vuln (Recommended)". | 0064-PLAN P2-5 (run 2026-11-20, P2 pass 1). |

All twelve rows are `decided` or `out of scope` (S8). Amendment A1 is accepted on 2026-11-20.
