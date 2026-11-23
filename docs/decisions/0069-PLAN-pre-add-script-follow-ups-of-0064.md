---
status: proposed
date: 2026-11-23
associated-madr: "0069-MADR-pre-add-script-follow-ups-of-0064.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement closing the pre-add script's three loose ends and testing the script

Associated MADR: [0069-MADR-pre-add-script-follow-ups-of-0064.md](0069-MADR-pre-add-script-follow-ups-of-0064.md)

*(2026-11-23: aligned with the owner's rulings in the MADR's "Owner rulings": Q1 "Fix in bash now, port later
(Recommended)", Q2 "Skip with a notice, exit 0 (Recommended)", Q3 "Own record later, if text found
(Recommended)", so the five code-quality items are outside this plan. The plan stays `proposed` until the owner
approves it.)*

## Goal

`scripts/go-precheck.sh` fails on a Go file gofmt cannot parse, names in its last line only the steps that ran, and
skips a deleted Go path with a notice on standard error and exit 0 (row G8); `scripts/check_go_precheck.py` replays rows G1 to G11 offline,
and `make preflight` runs it.

## Scope

* In: `scripts/go-precheck.sh`; new `scripts/check_go_precheck.py`; `Makefile` (a `check-go-precheck` target, listed in
  `.PHONY`, called from `preflight`); `AGENTS.md` ("Pre-add checks"); `docs/guides/developing.md` ("The gates");
  `docs/README.md` rows (returned to the coordinator, not edited by the drafter).
* Out: 0064-MADR's rows S1 to S12; the five code-quality items; a Python port; other repositories' wrappers; CI files.

## Implementation Steps

Every check and every plant runs on a `git archive` copy of the tip under `<scratch>`, never on the tree. Staged
only; the owner commits. Heavy runs go through `~/.agents/skills/available-dev-hosts/scripts/run-heavy` (or
`run-on-host`); never `-race` on WSL.

**Exit criterion: no blocker, every row tested, plants caught.**

1. **P0, re-ground.** At the phase's start commit read every line this plan cites: `scripts/go-precheck.sh:39`, `:44`,
   `:47-48`, `:62`, `:105`; `Makefile:175`, `:196`; `AGENTS.md` "Pre-add checks"; `docs/guides/developing.md` "The
   gates". Record whether each holds; correct by annotation where one moved. Confirm F1 live: an unparseable file and
   both skip variables give exit 0 and `1 file(s) clean`.
2. **P1, tests first (red).** Write `scripts/check_go_precheck.py` (Python 3 standard library). `--script PATH` selects the
   script under test (default `scripts/go-precheck.sh`); `--self-test` runs every test, then every plant (below) and
   fails unless each plant makes its named test fail. A base run red for a missing `acpext` or a
   harness error is not the expected red (G19). Each test builds a scratch `git init` repository holding the
   needed `.go` files and a `acpext` directory (the lint step enters it) and a stub directory in front of the real `PATH` (no symlinked `PATH`:
   fragile on Windows) holding a stub `go` (a `tool govulncheck -version` and a run both exit 0, each call appended to
   a counter file) and a stub `golangci-lint` (counted; exits 0); gofmt is the real one. Every test runs under `bash`
   from `PATH` and also under `/bin/bash` where that is another program (bash 3.2 on this Mac). Tests, each mapped to rows:
   * T1 (G1): `bad.go` containing `package`; skip both; expect exit 1, `gofmt:` header and gofmt's message, no `clean`.
   * T2 (G2): a parseable unformatted file; expect exit 1, `are not formatted`, the file named.
   * T3 (G3): both files in one call; expect exit 1, both outputs.
   * T4 (G4): a clean file, no skip variables; expect exit 0 and `(gofmt, golangci-lint, govulncheck)`.
   * T5 (G5, G6, G7): the three skip combinations; expect exit 0 and the three closing lines.
   * T6 (G8): a named missing `.go` path alone, with a clean file, and a tracked-then-removed file in the
     no-argument mode; expect exit 0 and a one-line notice naming the path on standard error, the same text in all three.
   * T7 (G9, G10): `README.md` only: exit 0, stub counters zero; `README.md` plus a clean `.go`: gofmt saw only the
     `.go`.
   * T8 (G11): the directory holding the real `gofmt` removed from `PATH`: exit 2 and the message.
   * T9 (G16): the whole suite under `/bin/bash` 3.2; a run that cannot find it says so and is not a pass.
   T6 also pins G18: N counts existing files only.
   Run T1, T5 (lint and vuln combinations) and T6 against the unchanged script. **Expected red:** T1 (exit 0), T5
   (closing line), T6 (no notice). T2, T3 partly, T4, T7, T8 are green already. A traceback or a harness error is not the
   expected red; fix the harness first. Record each test's output, the red ones included.
3. **P1 plants** (one production change each, on a scratch clone of the fixed script; the self-test applies them with
   `assert` before replace and reports which test turned red):
   * PL1: the `gofmt` status check removed: T1 fails (T3 still exits 1 from the unformatted branch, so only T1 is sure).
   * PL2: the closing line restored to the fixed three names: T5 fails.
   * PL3: the missing-file notice removed: T6 fails.
   * PL4: the no-Go-file early exit removed: T7 fails only because T7 runs without the skip variables, so lint and
     `go` are reached and the stub counters are non-zero.
   * PL5: the `need gofmt` guard removed: T8 fails.
   * PL6: `failed=1` removed from the unformatted branch: T2 fails.
   * PL7: an expansion of a possibly empty array added under `set -u`: T9 fails under bash 3.2 only.
4. **P2, the script.** At `:62` capture gofmt's standard output and standard error separately (a scratch file removed on
   exit) and its status; a non-zero status prints the message under `gofmt:` and sets `failed`; the unformatted list
   prints as before. At `:39` and `:47-48` print the notice (row G8) for a missing `.go` path (same code for both modes). Build
   the closing line from the steps that ran. Update the script's header comment (`Exit codes`, the deleted-path rule).
   `bash -n` under the host's bash and, if available, bash 3.2.
5. **P3, wiring.** Makefile: `check-go-precheck` runs `$(PYTHON) scripts/check_go_precheck.py --self-test`, is added to
   `.PHONY` and to `preflight` beside `check-records`.
6. **P4, docs.** `AGENTS.md` and `docs/guides/developing.md`: the closing line names the steps that ran, gofmt's own
   failure fails the add, and the deleted-path rule in the words of row G8; the sample line in the guide is checked
   against a real run. `docs/README.md` index rows (see the hand-back). `make check-records`, `make markdownlint`.
7. **P5, gates.** `bash -n` exit 0; `scripts/check_go_precheck.py --self-test` exit 0 with every plant caught; `make
   lint` (both modules, "0 issues."); `make preflight` through `run-heavy`, its pre-add step ending with
   `(gofmt)` and its `check-go-precheck` step passing; a `git archive` copy with an unparseable Go file fails `make preflight` (G17: the early gofmt step passes, the pre-add
   step fails); the tree's `git status` shows only this plan's files. The harness was not run on Windows by hand; its first
   Windows run is CI's preflight (G15).

## Verification

The red run of P1 and its three expected failures; the green run after P2 with the seven plants each failing the named
test; the real script on a scratch copy with an unparseable file (exit 1) and with `make preflight`.

## Rollout and Rollback

Staged only; the owner commits and pushes after the pre-add check on the changed files. Rollback: revert the commit; the
script's exit codes (`0`, `1`, `2`) and variables do not change.

## Execution record

*(Empty. Pass 1 of each phase is recorded here with its verification output, each deviation with a date, and what was
not done and why.)*
