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
