---
status: in-progress
date: 2026-11-14
associated-madr: "0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement treating the `internal/cli` race reports on the WSL2 distro as an environment fault

Associated MADR: [0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md](0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md)

## Goal

`make race` means what it says on every host that runs it: red for a race in gobble, green otherwise. The WSL2 distro
takes `-race` suites again only once the reproducer is green there (the MADR's Q1 and Q2 rulings of 2026-11-14). The
plan is complete when P0 to P3 are done and the MADR's Confirmation holds.

**Status.** `in-progress`: P0 and P1 are done by the commit that adds this pair (below). P2 waits on `wsl --update`
and its three runs; P3 waits on the dotfiles amendment.

## Scope

**Base.** `main` at `a3b5c26`, 0037-PLAN-resume-prefix-and-failed-turn-history.md's P1 commit. That commit also
edits `docs/README.md`, so this pair's rows are applied on top of it.

**In scope:**

* this pair and its two rows in `docs/README.md` (P0);
* one dated note in 0037-PLAN-resume-prefix-and-failed-turn-history.md, under its Deviation 2 (P1);
* `wsl --update` on the Windows laptop and the reproducer runs in the distro (P2).

**Out of scope:**

* any Go file, test, Makefile target or CI job: no option the MADR chose changes one;
* the routing change itself. It is an amendment to the dotfiles repository's
  `0015-MADR-load-aware-dev-host-test-dispatch.md`, pending there; this plan records its state and does not write it
  (P3);
* upstream reports and a gobble-free reproducer (the MADR's Q3 ruling, "Don't chase it").

## Implementation Steps

### P0 — the records

1. Apply this pair and its `docs/README.md` rows on the base above. Gates: `scripts/check_records.py --check-all`,
   `make check-records`, the disclosure redact dry run, and an identifier scan of the diff.

### P1 — the note in 0037's fifth plan

1. ~~In 0037-PLAN-resume-prefix-and-failed-turn-history.md, after the owner's ruling on Deviation 2, add a dated note
   that root-causes it here and corrects its "between goroutines".~~ *(2026-11-14: at `a3b5c26` the P1 agent's own
   dated note under Deviation 2 already gives the root cause, cites this MADR as pending, and records the race gates
   off the distro. Only what it lacked is added, as a second dated note after it: the correction that every report
   names one goroutine on both sides, not two; that the byte-for-byte binary check was made on the Linux machine, with
   Windows and macOS on builds of their own; and the link to this MADR, now `accepted` and no longer pending.)*
2. The note does not rewrite Deviation 2's text or the first note; it corrects them.
3. Gates: as P0.

### P2 — the WSL update and the re-admission check (Q1, Q2)

1. On the Windows laptop, record `wsl --version`, run `wsl --update`, and record `wsl --version` again, with the kernel
   version inside the distro (`uname -r`).
2. In the distro `Ubuntu-24.04`, at the then-current `main`, run the reproducer three times through the fleet's queue
   (`run-on-host --need linux --distro Ubuntu-24.04 --race`):
   `CGO_ENABLED=1 go test -race -count=40 -run '^TestPickModel$' ./internal/cli/`. Record each exit status and the number
   of `WARNING: DATA RACE` reports, split into one-goroutine and two-goroutine reports.
3. **Green:** three exit statuses of 0 and no report. The distro may take `-race` suites again; record that here and in
   P3's reference. **Not green:** any report or any non-zero exit. The distro stays off `-race`, and P2 is repeated after
   the next WSL kernel release. P2 is not worked round, and no step re-admits the distro on fewer than three green runs.

### P3 — the routing reference (Q1)

1. Record here the dotfiles commit that amends `0015-MADR-load-aware-dev-host-test-dispatch.md` to keep `-race` suites
   off the WSL distros, and the commit that lifts it after P2 is green, if it ever is.
2. Until the first of those lands, every brief that runs `make race` or `-race ./...` for this repository names a host
   other than the WSL distros.

## Verification

| Check | Command | Expect |
| :--- | :--- | :--- |
| re-admission (P2) | `go test -race -count=40 -run '^TestPickModel$' ./internal/cli/` in the distro, three runs | each exit 0, no report |
| reproducer before (MADR) | the same, at `cccef77` | red, one-goroutine reports only (3 of 3 runs) |
| the gate sees a real race | a planted two-goroutine `append` to `runEnv.overrides`, scratch copy only | red, two-goroutine reports |
| suites off the distro | `make race`; `go test -race -count=1 ./...` on the Linux machine and natively on the Windows laptop | green |
| records | `scripts/check_records.py --check-all`; `make check-records` | exit 0 |

## Rollout and Rollback

Nothing in gobble ships. The routing is lifted by the dotfiles amendment's own follow-up once P2 is green. Should the
distro misreport again after re-admission, it goes back off `-race` by the same route and P2 is repeated.

## Execution record

### Drafting (2026-11-14, in a scratch clone)

* The pair was drafted from the investigation recorded in the MADR, at `proposed`. The owner's rulings of 2026-11-14 on
  Q1 to Q3 were then recorded in the MADR, which was set `accepted`, and this plan was rewritten to the steps that remain.

### P0 and P1 (2026-11-14; done by the commit that adds this pair, on `a3b5c26`)

* P0: this pair and its two `docs/README.md` rows, after 0037's P1 rows.
* P1: the second dated note under Deviation 2 in 0037-PLAN-resume-prefix-and-failed-turn-history.md, as P1 step 1's
  annotation says.
* The same commit carries an unrelated ruling of the same day, on
  `0056-MADR-curate-the-terminal-session-view-with-view-modes.md` Q7, "Drop the third option", with its dated notes in
  that MADR and its PLAN's P3b step 0.
* Gates, in a scratch clone at `a3b5c26` with the patch applied: listed in the handoff that carries the patch.

### 0037 P1's race gates, off the distro (2026-11-14; from that plan's round 2, item 10)

0037-PLAN-resume-prefix-and-failed-turn-history.md P1 ran its race gates on hosts other than the WSL distros, as Q1
asks, after its last code change. Hosts are named by placeholder, as there:

* `make race` on the Linux machine (that plan's host L2): exit 0, 39 `ok`, 0 `FAIL`, 0 `DATA RACE`.
* `CGO_ENABLED=1 go test -race -count=1 ./...` on the Linux machine: exit 0, 38 `ok`, 0 `FAIL`, 0 `DATA RACE`,
  `internal/cli` included.
* `CGO_ENABLED=1 go test -race -count=1 ./...` on the Windows laptop, natively: exit 0, 38 `ok`, 0 `FAIL`.
* `make race` on the macOS laptop (host M): exit 0, 39 `ok`, 0 `FAIL`, 0 `DATA RACE`.
