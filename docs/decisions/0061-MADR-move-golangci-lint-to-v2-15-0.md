---
status: accepted
date: 2026-11-18
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Move golangci-lint from v2.14.0 to v2.15.0, the fleet standard

## Context and Problem Statement

The owner ruled on 2026-11-18, in the owner's own words: "standardize on golangci-lint v2.15.0". Asked how the moves
proceed in the repositories still pinned lower, the owner chose "One short record pair per repository, approved now
through CI (Recommended)": a record, the pin moved everywhere it appears, any findings fixed in the same change, then
commit and push, with no release.

gobble-cli pins `v2.14.0`. [0007-MADR-repository-scaffolding-to-fleet-standard.md](0007-MADR-repository-scaffolding-to-fleet-standard.md)
made it the fleet's pin (the `Makefile` comment cites it), and
[0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md) D3 offered `v2.15.0` and the owner kept `v2.14.0`
with the fleet. [0059-MADR-stop-excluding-confusing-naming-in-test-files.md](0059-MADR-stop-excluding-confusing-naming-in-test-files.md)
D5 requires that a change of the pin repeats the plants of its plan's P1 before it lands, because its text-keyed revive
exclusion depends on the wrapper's message format.

Re-verified on 2026-11-18 at `243ffc6`: the pin is written in `Makefile` (comment and install hint), `.github/workflows/ci.yml`,
`AGENTS.md` and `docs/guides/developing.md`. `golangci-lint` v2.15.0 (go1.27.4) reports "0 issues." on the root module
and on `acpext` on a `git archive` copy of the tip, so only the pin moves.

## Considered Options

* A. Move the pin to `v2.15.0` now, fixing what it flags (nothing flags today).
* B. Move the pin and suppress findings with `//nolint` or exclusions.
* C. Stay at `v2.14.0`.

## Decision Outcome

Chosen option: "A", because the owner ruled the fleet standard, v2.15.0 is clean on this tree, and nothing needs
suppressing. This supersedes the pin of 0007-MADR and 0023-MADR D3 for the version only; their other decisions stand.
No behaviour, dependency or release changes.

### Consequences

* Good, because every fleet repository lints with one version.
* Bad, because 0059-MADR D1 depends on the wrapper's text format; the plants of 0059-PLAN P1 are repeated at `v2.15.0`
  before the change lands (D5 there).

### Confirmation

* `golangci-lint` v2.15.0 reports "0 issues." on both modules through `make lint`.
* 0059-PLAN P1's plants behave as recorded: the `confusing-naming` collision is reported, a pair wholly inside test
  files is reported, another revive rule in a test file is dropped.
* `make preflight` passes, and CI is green after the push.

## More Information

* [0061-PLAN-move-golangci-lint-to-v2-15-0.md](0061-PLAN-move-golangci-lint-to-v2-15-0.md) implements this record.
