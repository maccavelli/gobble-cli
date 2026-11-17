---
status: complete
date: 2026-11-16
associated-madr: "0059-MADR-stop-excluding-confusing-naming-in-test-files.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement stopping excluding revive's `confusing-naming` in test files

Associated MADR: [0059-MADR-stop-excluding-confusing-naming-in-test-files.md](0059-MADR-stop-excluding-confusing-naming-in-test-files.md)

## Goal

`make lint` gives the same verdict on every run for a pair of names that differ only by case, wherever the pair lives:
the MADR's D1 in `.golangci.yml`, D2's list equal to `revive.rules` minus `confusing-naming`, the docs that describe the
lint gate saying so, and the two gaps the root-cause work left closed by runs on gobble's own tree. The plan is complete
when the MADR's Confirmation 1 to 6 holds and the owner has ruled on Q1.

This record is `proposed`. Nothing below starts before the owner accepts the MADR and approves this plan, whole or by
phase.

Dated 2026-11-16: the owner accepted the MADR and approved this plan on 2026-11-15 with the label "Approve; run it at phase boundaries", relayed by the coordinator.

## Scope

**Base.** The branch `p3-acpext-v0.5.0`, whose tip was `c4c5ec0` when this pair was drafted. Another agent builds 0051
and 0052 phases on that branch, so every phase here starts at a phase boundary the coordinator names, and runs its
experiments in scratch clones, never in the working tree.

**In scope:**

* this pair and its two `docs/README.md` rows (P0);
* the evidence runs on scratch clones (P1);
* `.golangci.yml`'s exclusion rules and their comments (P2);
* the docs that describe the lint gate (P2): `docs/guides/developing.md`, `docs/architecture.md`, `docs/README.md`'s
  "I want to…" table, and dated notes in `0007-MADR-repository-scaffolding-to-fleet-standard.md` D7 and
  `0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` Deviation 4;
* the record of the brief rules' retirement (P3).

**Out of scope:**

* any Go file, test, `Makefile` target, CI workflow, or script (a drift check is Q2 (b), and would be a phase of its
  own);
* the golangci-lint pin (MADR D5);
* the coordination ledger in the docs repository, which the coordinator owns; P3 records here what the coordinator
  changes there;
* `main`: this plan's commits reach it with the branch's fast-forward, which 0051-PLAN C8's note of 2026-11-15 governs.

## Conventions

* **K1. Start state.** Each phase records `git status`, `HEAD`, the branch tip and `origin/p3-acpext-v0.5.0`, and the
  golangci-lint version (`golangci-lint version`, expect `2.14.0`). A difference from this plan is a deviation.
* **K2. Scratch only for plants.** Every plant goes in a scratch clone of the branch tip (`git clone` of the local
  checkout into `<scratch>`), never in the working tree. Each planted Go file is `gofmt -l` clean before it is linted, so
  a formatting finding cannot hide the one under test.
* **K3. Cold runs.** Each lint run gets a new, empty `GOLANGCI_LINT_CACHE` under `<scratch>`, removed after the run,
  except in P1 step 6, which tests a warm cache on purpose. A configuration variant is copied over the clone's
  `.golangci.yml` before its runs, because `make lint` reads `$(CURDIR)/.golangci.yml`.
* **K4. Evidence.** Each run writes its full output and its exit status to `<scratch>` files, the status taken before
  any filter. Counts are made by a script that reads every run file in full, and the record quotes the counts and one
  finding line of each kind, temporary paths as `<scratch>`.
* **K5. Heavy runs.** Loops of `make lint` and every `make preflight` run one at a time on their host:
  on this Mac through the fleet's `run-heavy -- <command>`, elsewhere through `run-on-host`. A `-race` run never goes to a WSL distro (0058-MADR-cli-race-reports-on-the-wsl2-distro-are-not-gobble-races.md
  Q1), and no race result from one is recorded.
* **K6. Deviations stop the phase,** recorded here with a date, and in the MADR when a decision or a fact changes.
* **K7. Identifiers.** No hostname, account name or real-machine path in a committed file.

## Implementation Steps

### P0 — the records

Model class and effort: lightweight, low.

Files: `docs/decisions/0059-MADR-stop-excluding-confusing-naming-in-test-files.md`, this file, `docs/README.md`.

1. At a phase boundary on the branch (K1), place this pair in `docs/decisions/` and its two rows in `docs/README.md`,
   after the 0058 rows (the rows as drafted beside this pair). Confirm `python3 scripts/check_records.py --next` still
   prints `0059` before placing; if not, renumber the pair and its rows to the printed number before anything else, and
   record it.
2. Gates: `python3 scripts/check_records.py --check-all`, `make check-records markdownlint`, and an identifier scan of
   the diff. The commit holds only the pair and `docs/README.md` (AGENTS.md's bootstrap exception). It is docs-only, so
   CI's `ci.yml` skips it and `docs.yml` checks it (0026-MADR-skip-full-ci-for-docs-only-pushes.md).
3. On the owner's acceptance: the MADR `accepted`, dated, with the answers to its Q1 and Q2 quoted as the owner gave
   them; this plan `in-progress`, with the answer to Q3; the two README rows' status columns to match.

**Acceptance.** The gates exit 0; the pair is on the branch as `proposed`, then `accepted` with the answers.

### P1 — the configuration, proven on scratch clones

Model class and effort: mainstream, medium.

Files: none in the working tree. In scratch clones only: `.golangci.yml` variants, the plants
`llm/provider/zz_other_test.go`, `zz_pair_a_test.go`, `zz_pair_b_test.go`, and the `LoginOf` rename in
`llm/provider/export_test.go` and `llm/provider/login_fields_test.go`. This plan's execution record takes the results
(committed with P2).

Estimated cost on this Mac: 79 `make lint` runs (steps 3 to 7) at about 37 s each (the MADR's batch 12), about 50
minutes in `run-heavy`'s queue.

1. **Clone and variants.** Clone the branch tip into `<scratch>/p1` (K2). A Python script reads the clone's
   `.golangci.yml`, asserts that the first exclusion rule (`.golangci.yml:142-149` at `c4c5ec0`) occurs exactly once
   and that `linters.settings.revive.rules` holds 23 names including `confusing-naming`, and writes three variants:
   * `today.yml`, the file unchanged;
   * `new.yml`, D1 applied: `revive` removed from the first rule, and the second rule added after it with the comments
     of P2 step 1;
   * `none.yml`, `revive` removed from the first rule and no second rule (revive checks test files in full).
2. **D2's list.** The same script asserts that the alternation in `new.yml`'s `text` equals, as an ordered list,
   `revive.rules` without `confusing-naming`, and that `confusing-naming` is not in it. Seen to fail first: run the
   assertion on a copy of `new.yml` with `confusing-naming` added to the list, and on one with `errorf` removed; each
   must fail. Then `golangci-lint config verify --config` on each variant: exit 0.
3. **Another revive rule in a test file (closes the gap of the MADR's batch 11).** Add
   `llm/provider/zz_other_test.go`:

   ```go
   package provider

   import "errors"

   // PlantBad is planted by the measurement.
   func PlantBad() error { return errors.New("Planted Error.") }
   ```

   Five cold `make lint` runs with each of `none.yml`, `today.yml` and `new.yml`, interleaved.
   * `none.yml`: 5 of 5 non-zero, each with the line
     `llm/provider/zz_other_test.go:6:…: error-strings: error strings should not be capitalized or end with punctuation or a newline (revive)`.
     This is the check seen to fire. `none.yml` lets all 22 rules check every existing test file too, so other revive
     findings in test files may appear beside it (the MADR's option B, not measured until now): count them by rule and
     record them, without acting on them. If the plant's line does not appear, stop: a deviation, and a plant of another
     rule is chosen with the owner, not assumed.
   * `today.yml` and `new.yml`: 5 of 5 exit 0, "0 issues." for both modules.
   Remove the plant.
4. **The collision, with the gate's command (closes the gap of batches 9 to 11).** In the clone, rename the test helper
   `LoginFromSDKSession` back to `LoginOf` in `llm/provider/export_test.go` (its comment and its declaration, two
   occurrences at `c4c5ec0`) and in `llm/provider/login_fields_test.go` (one call at `c4c5ec0`), asserting each count
   before replacing. This recreates `d82abd0`'s pair with production `loginOf` (`llm/provider/login.go:136` at
   `c4c5ec0`). Twenty cold `make lint` runs with `today.yml` and twenty with `new.yml`, interleaved.
   * `new.yml`: 20 of 20 non-zero, each with exactly one `confusing-naming` line, naming `loginOf` at
     `llm/provider/login.go` or `LoginOf` at `llm/provider/export_test.go`; acpext "0 issues." each time. Record the
     split by file.
   * `today.yml`: record the number of red runs. No threshold; the MADR measured 4 to 14 of 20 with the package and root
     commands. Should all 20 be red, record it: the gap this closes is `new.yml`'s, not today's.
5. **A pair wholly inside test files (D3's new behaviour, with the gate's command).** Remove step 4's plant. Add
   `llm/provider/zz_pair_a_test.go` declaring `func PlantPair() {}` and `llm/provider/zz_pair_b_test.go` declaring
   `func Plantpair() {}`, each `package provider` with a doc comment. Five cold `make lint` runs with `new.yml`: 5 of 5
   non-zero, one `confusing-naming` line each, at either file. Remove the plant.
6. **A warm cache cannot keep a green (the MADR's assumption).** Step 4's plant again, `new.yml`: three caches, three
   runs per cache sharing it. Expect 9 of 9 non-zero. A green run is a deviation: the assumption is false, and the MADR
   is amended before P2.
7. **Today's tree, clean.** An unplanted clone of the tip with `new.yml`: ten cold `make lint` runs, 10 of 10 exit 0,
   "0 issues." for both modules. With them, account for the four case-only pairs the MADR lists: re-read their
   declarations at the tip (`git grep -n -w` for each name), record each pair's kinds (type and function, or two
   constants) and files, and confirm none is a pair of functions or methods. A new function or method pair at the tip
   would show as a finding in these runs, and is a deviation, fixed by a rename in its own phase.

**Acceptance.** Steps 2 to 7 give the counts above, quoted in the execution record with one finding line of each kind.
Every run file is kept under `<scratch>` until P3.

### P2 — the configuration and the docs on the branch

Model class and effort: mainstream, medium.

Files: `.golangci.yml`, `docs/guides/developing.md`, `docs/architecture.md`, `docs/README.md`,
`docs/decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md`,
`docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md`, this file (P1's and P2's record). Any
other document found false or incomplete joins by a dated deviation (0051-PLAN C9).

1. **The configuration.** At a phase boundary the coordinator names (K1), apply D1 to `.golangci.yml` by a Python script
   that asserts the old rule occurs exactly once. `cmp` the result against P1's `new.yml`: identical. The comments:

   ```yaml
         # Test files are not checked by these linters.
         - linters:
             - errcheck
             - gosec
             - unparam
             - gocritic
             - goconst
           path: _test\.go
         # Nor by revive, but for confusing-naming. revive compares a package's
         # names in whatever order it reads the files, so a pair split between a
         # test file and another file is reported on either one; dropping it in
         # test files made the verdict depend on that order (0059-MADR). The text
         # is golangci-lint's "<rule>: <message>". It names every rule under
         # revive.rules except confusing-naming, in that order: add or remove a
         # rule in both places together, and never add confusing-naming here
         # (0059-MADR D2). A change of the golangci-lint pin repeats 0059-PLAN
         # P1's plants (0059-MADR D5).
         - linters:
             - revive
           path: _test\.go
           text: '^(atomic|…|var-declaration): '
   ```

   (The `text` line is D1's in full.)
2. **Docs (every document the change makes false or incomplete, in the same commit, as 0051-PLAN C9 asks).**
   * `docs/guides/developing.md`, "The gates": after the list of preflight's gates, a paragraph saying what
     golangci-lint does not check in test files (errcheck, gosec, unparam, gocritic, goconst, and revive's rules but
     `confusing-naming`), that `confusing-naming` is checked everywhere because its finding lands on either file of a
     pair, that one cold `make lint` decides it, that the finding may name either file from run to run, and that it is
     fixed by a rename, never by `//nolint` (MADR D3).
   * `docs/architecture.md`, the paragraph on what golangci-lint keeps out of test files (`:181-184` at `c4c5ec0`): one
     sentence on the test-file exclusions and `confusing-naming`, citing the MADR.
   * `docs/README.md`, "I want to…": a row, "know which lint checks skip test files, and why `confusing-naming` does not"
     → the MADR and `guides/developing.md`.
   * `0007-MADR-repository-scaffolding-to-fleet-standard.md` D7: a dated note that revive's entry in the test-file
     exclusion is superseded by 0059-MADR D1, the rest of D7 standing; the original text untouched.
   * `0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` Deviation 4: a dated note that the root cause is
     completed in 0059-MADR (the finding is dropped when it lands on the test file, and a warm cache keeps either
     outcome), and that D1 makes the verdict order-independent; the original text untouched.
   * `AGENTS.md` and the `Makefile`'s `lint` comment: read; neither describes the exclusion, so neither is expected to
     change. Record that.
   Each added sentence is checked against `.golangci.yml` and P1's runs before it is kept.
3. **Gates (0051-PLAN C6's set),** every one after the last file change:
   * no Go file changes, so `make pre-add-check` has no `FILES`; `make preflight` runs it over every Go file anyway;
   * `make preflight` on this Mac through `run-heavy`: exit 0, "preflight passed", golangci-lint "0 issues." for both
     modules;
   * `make preflight` on the Linux machine through `run-on-host`: exit 0;
   * `make race` through `run-on-host --race` on a host that is not a WSL distro, and `make check-test-isolation`
     through `run-on-host`: exit 0. No Go file changed, so neither can be changed by this phase; they keep C6 whole;
   * `python3 scripts/check_records.py --check-all`, `make check-records markdownlint`: exit 0;
   * an identifier scan of the diff: nothing.
4. **Commit and push.** Before the first commit, `git rev-parse --path-format=absolute --git-path hooks` is the global
   hooks directory or a wrapper that chains to it. The commit is made with `git commit --no-edit` by whoever 0051-PLAN
   C8's note of 2026-11-15 names for the branch (the coordinator commits each approved phase there). The disclosure
   guard runs before the push. The push to `p3-acpext-v0.5.0` runs CI on draft pull request 18; `.golangci.yml` is
   outside `ci.yml`'s `paths-ignore`, so the full workflow runs.

**Acceptance.** Every gate exits 0; the commit is on the branch; CI on pull request 18 for that commit is green on
`validate (ubuntu-24.04)`, `validate (macos-15)` and `validate (windows-2025)`, each `make preflight` with golangci-lint
"0 issues." for both modules. A red CI run is a deviation, read before anything is rerun.

### P3 — close-out and the brief rules

Model class and effort: lightweight, low.

Files: this file, the MADR, `docs/README.md`.

1. Record P2's CI run: its id, head SHA and each job's conclusion, read with `gh run view`.
2. Record the owner's Q1 ruling and what follows: under Q1 (a), from this date briefs for this repository stop carrying
   "six cold lint runs" and "no capitalization-only names". The coordinator changes the ledger (the docs repository's
   `0006-PLAN-sequence-the-auth-contract-effort-across-four-repositories.md`, rows `S-gobble` and `S-gobble-lint`);
   this plan records the date and the ledger commit. Under (b) or (c), record what stays and until when.
   *Dated note, 2026-11-16:* the ledger is a plain file outside version control, so no ledger commit exists to cite. Per the owner's ruling, relayed by the coordinator ("Record the date and say the ledger is unversioned"), this plan records the date and states that the ledger is unversioned. See the execution record, "P3", step 2, and its dated deviation.
3. Delete the P1 run files under `<scratch>`, once their counts are in the execution record.
4. This plan `complete`, the MADR's Confirmation marked with where each item was met, both README rows updated. Gates
   as P0 step 2.
5. After the branch's fast-forward to `main` (0051-PLAN C8's note: after 0051-PLAN P6 and 0052-PLAN P4), a dated note
   here with the CI run of `main`'s push. It is not a completion criterion: the plan completes at step 4.

**Acceptance.** The records gates exit 0; the plan is `complete` with each Confirmation item's evidence named.

## Verification

| Check | Command | Expect | MADR Confirmation |
| :--- | :--- | :--- | :--- |
| another revive rule fires, then is dropped (P1 step 3) | 5 cold `make lint` per variant, `error-strings` plant | `none` 5 of 5 red; `today`, `new` 5 of 5 green | 1 |
| collision with the gate's command (P1 step 4) | 20 cold `make lint` per variant, the `LoginOf` plant | `new` 20 of 20 red; `today` recorded | 2 |
| warm cache (P1 step 6) | 3 caches × 3 runs, `new`, the `LoginOf` plant | 9 of 9 red | 3 |
| today's tree (P1 step 7) | 10 cold `make lint`, `new`, unplanted | 10 of 10 green, both modules | 4 |
| D2's list (P1 step 2) | the variant script's assertion, and its two broken copies | passes; each broken copy fails | 5 |
| test-only pair (P1 step 5) | 5 cold `make lint`, `new` | 5 of 5 red | (D3) |
| the branch (P2 step 3) | `make preflight` on this Mac and the Linux machine; `make race`; `make check-test-isolation` | exit 0 | 4 |
| CI (P2 step 4) | draft pull request 18 | `validate` green on three runners | 6 |
| records (P0, P3) | `scripts/check_records.py --check-all`; `make check-records markdownlint` | exit 0 | — |

## Rollout and Rollback

**Rollout.** P0's records commit and P2's commit land on `p3-acpext-v0.5.0`, each at a phase boundary between 0051 and
0052 phases (Q3). CI runs on draft pull request 18 with every push to the branch. Both reach `main` with the branch's
fast-forward, after 0051-PLAN P6 and 0052-PLAN P4 land and every gate is green there (0051-PLAN C8's note of
2026-11-15). No tag and no release follows from this plan.

**Rollback.** `git revert` of P2's commit, as a new commit on the branch (or on `main` after the fast-forward), restores
the path-only exclusion; no history is rewritten. The two brief rules come back with it, the MADR gets a dated
amendment, and this plan a deviation entry. Reasons to roll back: a CI runner or host where D1's configuration does
not load; a flood of test-file findings after a golangci-lint upgrade that the plants did not foresee. A new
`confusing-naming` finding in a test file is not a reason: it is D3 working, fixed by a rename.

## Open questions for the owner

* **Q1. When the two brief rules retire.** In the MADR, with options (a), (b), (c); recommended (a), at P3.

  Answer, 2026-11-15 (the owner's ruling, relayed by the coordinator): option (a), label "Both, once the fix is proven"; see the MADR's Q1.
* **Q2. A permanent check on D1's list.** In the MADR, with options (a) and (b); recommended (a), no new check.

  Answer, 2026-11-15 (the owner's ruling, relayed by the coordinator): option (a), label "Comment and one-off check only"; see the MADR's Q2.
* **Q3. When P2 lands on the branch.**
  * (a) At the first phase boundary after approval, between 0051 and 0052 phases. *Recommended:* the remaining phases
    on the branch then drop the six-run loop (under Q1 (a)) and get an order-independent lint on pull request 18.
    Cost: one more commit interleaved with the branch's phases, and P1's 50 minutes of `make lint` in this Mac's queue.
  * (b) After the fast-forward, on `main`. Cost: the six-run loop stays on every remaining branch phase, and the
    order-dependent verdict stays on pull request 18 until then.

  Answer, 2026-11-15 (the owner's ruling, relayed by the coordinator): option (a), label "First phase boundary after approval".

## Execution record

### Drafting (2026-11-15)

* The pair was drafted read-only against the branch at `c4c5ec0`, while another agent built in the checkout (the owner's
  "Draft it on the branch now"), from the root-cause outputs of 2026-11-15 and from source read in the module cache.
  The checks made while drafting ran outside the working tree, and are in the MADR's More Information.
* The number came from `python3 -B scripts/check_records.py --next` at `c4c5ec0` (`0059`), matched by a scan of every
  tracked `NNNN-{MADR,PLAN,REPORT,GATES}-*.md` (0001 to 0058 used) and by a search of every local and remote branch for a
  `0059-` file (none). Another record can take `0059` before this pair is placed; P0 step 1 checks.

### P0 — the records (2026-11-16)

* Provenance: placed by a lightweight model on the coordinator's brief, from the drafts outside the working tree.
* Start state: `git status` clean; `HEAD` and the branch tip `17f2d6c` on `p3-acpext-v0.5.0`.
* `python3 scripts/check_records.py --next` printed `0059`, so the pair was placed without renumbering.
* Files placed, unchanged from the drafts: `docs/decisions/0059-MADR-stop-excluding-confusing-naming-in-test-files.md`
  and `docs/decisions/0059-PLAN-stop-excluding-confusing-naming-in-test-files.md`.
* Rows added to `docs/README.md` in the `## Records` table after the 0058 PLAN row: the 0059 MADR row and the 0059 PLAN
  row. The third row (the `## I want to…` row) is not added; it belongs to P2.
* Owner's approval, given 2026-11-15, relayed by the coordinator: "Approve; run it at phase boundaries". Recorded as
  accepted and approved in the MADR's Decision Outcome and in this plan's status note.
* Owner's answers, relayed by the coordinator: Q1 "Both, once the fix is proven" (option (a)); Q2 "Comment and one-off
  check only" (option (a)); Q3 "First phase boundary after approval" (option (a)). Recorded under each question in the
  MADR and this plan.
* Frontmatter: MADR `status: accepted`, PLAN `status: in-progress`, both dated 2026-11-16. README status cells: MADR
  `accepted 2026-11-16`, PLAN `in progress 2026-11-16` (the form the table uses for a plan under way, as in the 0051, 0052 and 0058 rows).
* Gates, each run after the last edit to the files it checks:
  * `python3 scripts/check_records.py --check-all`: exit 0, no output.
  * `make check-records markdownlint`: exit 0 (markdownlint-cli2 v0.23.2: 0 issues in 58 files).
  * Identifier scan of `git diff --cached` for the home-path and session-path patterns, the account name and the host
    name of the working machine: no match (grep exit 1).
* Staged, not committed; the coordinator commits on the branch.

### P1 — the configuration, proven on scratch clones (2026-11-16)

* Provenance: run by a mainstream model at medium effort, on the coordinator's brief, in scratch clones; no file in the working tree was changed. The entry is placed here by P2 from the P1 agent's draft, each figure checked against its saved counts files (see the correction below).
* Start state (K1): the clone is `git clone` of the local repository, checked out at `4410333` (the branch tip when P1 ran); `git status` in the clone clean before the variants; `golangci-lint version` 2.14.0 (go1.27.4). The working tree was not read except through `git show` / `git clone`. Total: 79 `make lint` runs (steps 3 to 7), each chunk through `run-heavy` on this Mac. The plan estimated 37 s a run and 50 minutes. Measured: a mean of 26 s per cold run (70 runs), and the 9 warm-cache runs of step 6 took 25 s for each cache's first run and 2 s for the next two; 29 minutes inside `run-heavy` (09:31 to 10:00), 32 minutes from the first ticket, which waited 175 s in the queue. Every run file (`.out`, `.rc`, `.sec`) is kept under `<scratch>/runs/s3` to `s7`; every cold run had a new empty `GOLANGCI_LINT_CACHE` under `<scratch>/caches`, removed after the run (step 6 excepted). Each planted Go file was `gofmt -l` clean before linting; each variant was copied over the clone's `.golangci.yml` before every run; the exit status was written before any filter; counts come from `<scratch>/bin/counts.py`, which reads every run file in full.
* Step 1 (variants): the script asserted that the first exclusion rule (`.golangci.yml:142-149` at `4410333`) occurs once, that `revive.rules` holds 23 names including `confusing-naming`, that the generated `text` equals the MADR's D1 line, and that the plan's P2 step 1 block, with `…` replaced by the list, holds the old rule minus `revive` exactly once. It wrote `today.yml` (unchanged), `new.yml` (D1 with P2's comments) and `none.yml` (`revive` removed from the first rule, no second rule).
* Step 2 (D2's list): the assertion passed on `new.yml` (22 names, in `revive.rules` order, `confusing-naming` absent). Seen to fail on both broken copies: with `confusing-naming` added, `AssertionError: confusing-naming is in the list`; with `errorf` removed, `AssertionError: list != revive.rules minus confusing-naming: got [...]`. `golangci-lint config verify --config` exit 0 for each of the three variants.
* Step 3 (`error-strings` plant, 15 runs, interleaved none/today/new): `none.yml` 5 of 5 red (make exit 2), each with `llm/provider/zz_other_test.go:6:43: error-strings: error strings should not be capitalized or end with punctuation or a newline (revive)`. `today.yml` 5 of 5 and `new.yml` 5 of 5 green, "0 issues." for both modules each time. As the plan expects.
  * Other revive findings in existing test files under `none.yml` (the MADR's option B, measured now for the first time): 6 per run, the same 6 in all 5 runs, all `redefines-builtin-id`, 0 of any other rule. In 6 files: `acpserver/answers_test.go:149`, `internal/archtest/check_test.go:15`, `internal/fsx/ops_test.go:199`, `permission/store_test.go:131`, `tool/builtin/fileops_test.go:221` (each a local named `real`, shadowing the built-in function), and `internal/cli/ambient_test.go:180` (a name `any`, shadowing the built-in type). Quoted: `acpserver/answers_test.go:149:5: redefines-builtin-id: redefinition of the built-in function real (revive)`. Recorded, not acted on.
* Step 4 (the `LoginOf` plant, 40 runs, interleaved today/new): the rename asserted 2 occurrences in `llm/provider/export_test.go` and 1 in `llm/provider/login_fields_test.go`. `new.yml`: 20 of 20 red, exactly one `confusing-naming` line each; split by file `llm/provider/login.go` 11, `llm/provider/export_test.go` 9; acpext "0 issues." each time. `today.yml`: 8 of 20 red (all with the line at `llm/provider/login.go`), 12 green with "0 issues." for both modules. Quoted: `llm/provider/login.go:136:6: confusing-naming: Method 'loginOf' differs only by capitalization to function 'LoginOf' in llm/provider/export_test.go (revive)`. As expected (no threshold for `today.yml`; the MADR measured 4 to 14 of 20).
* Step 5 (pair wholly in test files, 5 runs, `new.yml`): 5 of 5 red, one `confusing-naming` line each; at `llm/provider/zz_pair_a_test.go` 4, `llm/provider/zz_pair_b_test.go` 1. Quoted: `llm/provider/zz_pair_a_test.go:4:6: confusing-naming: Method 'PlantPair' differs only by capitalization to function 'Plantpair' in llm/provider/zz_pair_b_test.go (revive)`.
* Step 6 (warm cache, `new.yml`, the `LoginOf` plant, 3 caches x 3 runs): 9 of 9 red, one `confusing-naming` line each. Each cache kept its first run's file: caches 1 and 3 `llm/provider/login.go:136` (3 each), cache 2 `llm/provider/export_test.go:22` (3). The assumption holds: the verdict is red whichever file the cache keeps.
* Step 7 (today's tree, unplanted clone, `new.yml`, 10 cold runs): 10 of 10 exit 0, "0 issues." for both modules each time; no finding of any kind. The four case-only pairs, re-read with `git grep -n -w` at `4410333`: `Selection` (type) `llm/provider/ambient.go:20` and `selection` (function) `llm/provider/ambient.go:69`; `RequestPermissionsTool` and `requestPermissionsTool` (two constants) `acpserver/grants.go:24,26`; `PromptError` (type) `acpclient/prompterror.go:17` and `promptError` (function) `acpclient/prompterror.go:43`; `ToolContent` (type) `acpclient/update.go:225` and `toolContent` (function) `acpclient/update.go:355`. No method is declared under any of the eight names, no declaration is in a test file, and none is a pair of functions or methods. A further `promptError` function exists in `acpserver/prompt.go:272`, in another package, with no `PromptError` there.
* Deviations: none. No run was repeated or discarded; every run is in the counts.
* Not verified: CI's cache state and runners (no run outside this Mac); `-race`, `make preflight` and the other hosts (P2 step 3); the `text` regex under another golangci-lint version (D5); only the `error-strings` rule was planted of the 22 excluded rules, so the exclusion of the other 21 in test files rests on the list assertion, not on a plant each; the variant order within a round was fixed (none, today, new), not randomised; `make lint` exit status is make's 2 for a red run, golangci-lint's own status was not recorded separately.
* Correction made when placing: the draft said "mean 26 s each, 32 minutes of run time". The counts files give 26 s for the 70 cold runs, 23 s over all 79 (the 9 warm runs are short), and 29 minutes of run time in `run-heavy`, 32 from the first ticket with its 175 s wait. Every other figure in the draft matched the counts files and the run files read. No file in the working tree was changed by P1.

### P2 — the configuration and the docs on the branch (2026-11-16)

* Provenance: run by a mainstream model at medium effort, as the phase prices it, on the coordinator's brief.
* Approval and phase boundary: the owner approved the plan on 2026-11-15 with "Approve; run it at phase boundaries", and answered Q1 "Both, once the fix is proven", Q2 "Comment and one-off check only" and Q3 "First phase boundary after approval" (the owner's ruling, relayed by the coordinator). The coordinator named this phase boundary (K1): after 0051 P6's commit `b78503d`.
* Start state (K1): branch `p3-acpext-v0.5.0`, `HEAD` `b78503d`, level with `origin/p3-acpext-v0.5.0`, `git status` clean; `golangci-lint` 2.14.0 (go1.27.4). `git diff 4410333 HEAD -- .golangci.yml` is empty, so P1's `new.yml` still applies to this file, and `.golangci.yml:142-149` is still the first exclusion rule.
* Step 1, the configuration. `apply_d1.py` (Python 3, stdlib) asserted that the old rule occurs exactly once and that `revive.rules` holds 23 names including `confusing-naming`, built `text` from the 22 others, and wrote the file: `.golangci.yml:142` the comment "Test files are not checked by these linters.", `:143-149` the first rule without `revive`, `:150-158` the revive comment, `:159-162` the second rule with the `text` line. `cmp` of the result against P1's `new.yml`: identical (exit 0). `golangci-lint config verify --config .golangci.yml`: exit 0.
  * The tree moved since P1 (`b78503d` added about 5900 lines of Go and docs in `acpserver`, `internal/cli`, `llm/provider`). One cold `make lint` on a scratch clone of `b78503d` with the new `.golangci.yml`, a new empty `GOLANGCI_LINT_CACHE`, through `run-heavy`, 18:13:55 to 18:14:26, exit 0: "golangci-lint (.)" "0 issues." and "golangci-lint (acpext)" "0 issues."; no `confusing-naming` finding, no finding of any kind.
  * D2's one-off check (`check_list.py`): the `text` line names every rule under `revive.rules` except `confusing-naming`, in order (23 rules, 22 in the list, `confusing-naming` absent): exit 0 on the result. Seen to fail on two scratch copies: with `errorf` removed from the `text` line, exit 1, `AssertionError: list != revive.rules minus confusing-naming: got [...]`; with a rule `exported` added under `revive.rules`, exit 1, the same assertion. It also fails on the file before the change (exit 1, `AssertionError: revive text rule count 0`), where there is no second rule.
* Step 2, the docs. Each sentence was checked against `.golangci.yml` as changed and against P1's counts files.
  * `docs/guides/developing.md:131-139`, a paragraph after "The gates" list: the five linters named in the plan plus `forbidigo` (also excluded for `_test.go`, `.golangci.yml:163-165`); revive's other rules excluded by the finding's text (`:159-162`); the one name table per package and the either-file landing (MADR, P1 step 4: 11 and 9 of 20); one cold `make lint` decides it and a warm cache cannot keep a green (P1 step 6: 9 of 9 red); the rename, never `//nolint` (MADR D3); the list is kept with `revive.rules` (`.golangci.yml:150-158`).
  * `docs/architecture.md:185-188`, after the paragraph on what `golangci-lint` keeps out of test files: one sentence on the other test-file exclusions and `confusing-naming`, citing 0059-MADR D1; checked against `.golangci.yml:143-162`.
  * `docs/README.md:307`, "I want to…": "know which lint checks skip test files, and why `confusing-naming` does not", to the MADR and `guides/developing.md#the-gates`. The 0059 PLAN row (`:135`) now says P1 is done and P2 is staged.
  * `docs/decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md:197`, a dated note after D7's text: revive's entry in the test-file exclusion is superseded by 0059-MADR D1, the rest of D7 standing. Checked against `.golangci.yml:143-162`. The original text is untouched.
  * `docs/decisions/0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md:1016-1021`, a dated bullet at the end of Deviation 4: the root cause is completed in 0059-MADR; the exclusion dropped the finding when it landed on the test file; a warm cache keeps either outcome (MADR batch 7); D1 makes the verdict order-independent (P1 steps 4 and 6). The original text is untouched.
  * `AGENTS.md` (read): its gate list names `golangci-lint (v2.14.0)` and does not describe the exclusion; not changed. The `Makefile`'s `lint` comment (read, `:132`): "golangci-lint v2.14.0 is the fleet pin (0007-MADR D5)"; not changed.
  * Searched for other documents the change makes false: `grep -rn -i -E 'six cold|six runs|capitalization-only|revive' --include=*.md`: every other hit is a historical record of an earlier finding or run, which stands as written. The brief rules are in no record of this repository (MADR, "What it costs today"), and their retirement is P3's.
* Gates (step 3, 0051-PLAN C6's set). Every file change except this table came before the first gate; the table and the records gates' rows came after, and the records gates were run again after the last edit (the second row from the end). Each exit status was taken before any filter. No Go file changed, so `make pre-add-check` has no `FILES`; `make preflight` runs it over every Go file.

  | Gate | Where | Exit | Start to end (2026-11-16) |
  | :--- | :--- | :--- | :--- |
  | `make preflight` | the Linux machine, `run-on-host --need linux --race` | 0; "preflight passed"; golangci-lint "0 issues." for `.` and for `acpext` | 18:16:23 to 18:27:58 |
  | `make preflight` | this Mac, `run-heavy` (queued 18:16, started 18:17:25) | 0; "preflight passed"; golangci-lint "0 issues." for `.` and for `acpext` | 18:17:25 to 18:24:57 |
  | `make race` | the Windows laptop (native `windows/amd64`; pick-host excluded its WSL distro for a race run), `run-on-host --race` | 0; 39 packages `ok`, no `FAIL`, no `DATA RACE` | 18:28:16 to 18:33:09 |
  | `make check-test-isolation` | the same laptop, `run-on-host` | 0; root and acpext "0 failing", "files written above the temp roots: 0" | 18:33:28 to 18:35:10 |
  | `python3 scripts/check_records.py --check-all` | this Mac | 0, no output | 18:35:32 to 18:35:33 |
  | `make check-records markdownlint` | this Mac | 0; markdownlint-cli2 v0.23.2, "Summary: 0 issues in 0 files" (the MADR and PLAN files are ignored by its globs) | 18:35:33 to 18:35:35 |
  | `python3 scripts/check_records.py --check-all`, `make check-records markdownlint`, run again after the rows above were written | this Mac | 0 and 0 | after this file's last edit |
  | identifier scan of `git diff --cached` (the account name, the home and session path prefixes, the hostnames of the fleet's machines, queue ticket ids) | this Mac | no match (grep exit 1) | after staging |

  The cold lint of step 1 (above, exit 0, 18:13:55 to 18:14:26) ran on the scratch clone before the gates. The two preflights ran at the same time on different hosts; each host ran one heavy run at a time.
* Last-modified times, against the first gate's start (18:16:23): `.golangci.yml` 18:14:31, `docs/architecture.md` 18:15:00, `docs/guides/developing.md` 18:15:00, the 0007 MADR and the 0052 PLAN 18:15:14, `docs/README.md` 18:15:28. This file's execution record was written at 18:15 to 18:16 and its gate table and the rows after the records gates came after the gates (as the gates paragraph says); the preflights, `make race` and `make check-test-isolation` ran on the other files as they stand.
* Not done: the commit, the push and CI on pull request 18 (step 4, the coordinator's); P3 (the brief rules stay until P3); no Go file, `Makefile`, script or workflow changed.
* Staged, not committed; the coordinator commits on the branch and watches CI on pull request 18.

### P3 — close-out and the brief rules (2026-11-16)

* Provenance: run by a lightweight model at low effort, on the coordinator's brief, steps 1 to 4 only. No Go file, config, workflow or script changed; no heavy run; no commit, push or tag.
* Start state (K1): branch `p3-acpext-v0.5.0`, `HEAD` `aff572a`, level with `origin/p3-acpext-v0.5.0`, `git status` clean. This phase ran no lint, so `golangci-lint` was not re-read.
* Step 1, CI on `aff572a` (read with `gh`, read-only). `gh pr checks 18`: every check `pass`, except `release`, which is `skipping`. The `CI` run is `43990126887` and the `Docs` run is `43990126524`; both `completed` with conclusion `success` and head SHA `aff572a99c54857bb8314ff499f67dd9b65f4244`. URLs: `https://github.com/<owner>/gobble-cli/actions/runs/<run id>`.
  * `CI` run `43990126887` jobs, conclusion `success` unless stated: `validate (ubuntu-24.04)` (job `133728286286`), `validate (macos-15)` (`133728285826`), `validate (windows-2025)` (`133728286270`), `sbom` (`133728286313`), `test-isolation` (`133728286490`), `build / build` (`133734781960`), `build / identity` for darwin/arm64, linux/amd64, linux/arm64, windows/amd64 and windows/arm64 (`133736116419`, `133736116475`, `133736116477`, `133736116593`, `133736116556`), and `release` `skipped` (`133736212504`).
  * `Docs` run `43990126524`: job `docs` (`133728284216`) `success`.
  * The `golangci-lint` lines of the three `validate` logs (`gh run view <run id> --job <job id> --log`, saved under `<scratch>`), timestamp prefixes removed. Each log has the same lint sequence: "golangci-lint (.)", "0 issues.", "golangci-lint (acpext)", "0 issues.", and then, after the markdownlint summary "Summary: 0 issues in 0 files", the line "preflight passed":
    * `validate (ubuntu-24.04)`: "golangci-lint (.)", "0 issues.", "golangci-lint (acpext)", "0 issues.", "preflight passed".
    * `validate (macos-15)`: the same sequence.
    * `validate (windows-2025)`: the same sequence.
  * Every job in the acceptance is green, so no stop.
* Step 2, the owner's Q1 ruling and what follows. Q1 is answered option (a), in the owner's words, label "Both, once the fix is proven" (the owner's ruling, relayed by the coordinator; the MADR's Q1 and this plan's Q1). Under (a), from 2026-11-16, with P2 committed as `aff572a` and its CI green (step 1), briefs for this repository stop carrying "six cold lint runs" and "no capitalization-only names".
  * The coordinator changed the ledger's rows `S-gobble` and `S-gobble-lint` on 2026-11-16, as the coordinator reports. This phase did not read the ledger.
  * Dated deviation, 2026-11-16: step 2 says this plan records "the ledger commit". The ledger is a plain file outside version control, so no commit exists to cite. Per the owner's ruling, relayed by the coordinator, "Record the date and say the ledger is unversioned", this plan records the date the rules retire (2026-11-16) and states that the ledger is unversioned. Nothing else changes now. Options not chosen: put the ledger under git first; drop the citation from the plan. Open item for the owner: putting the ledger under git. The committed step text is left as written; the dated note at step 2 points here.
* Step 3, P1's run files. The counts are in P1's record above (steps 3 to 7; the counts files `counts-s3.txt` to `counts-s7.txt` are kept under `<scratch>`). Deleted with `rm -rf` on exactly these paths, after `ls -d` and `du -sh`:
  * `<scratch>/gobble-0059-p1/runs`, 948K;
  * `<scratch>/gobble-0059-p1/caches`, 0 B (empty);
  * `<scratch>/gobble-0059-p1/p1`, the scratch clone, 131M.
  After the delete, `ls -d` on the three paths printed "No such file or directory" for each. Kept under `<scratch>/gobble-0059-p1`: the counts files, `bin`, `variants`, `ref`, the logs, `golangci-version.txt` and `P1-record.md`. Nothing in the repository or in the `~/.cache` session folder was touched.
* Step 4, completion. This plan's `status` is `complete` (2026-11-16). The MADR's Confirmation items 1 to 6 are marked with their evidence (MADR, Confirmation). Both `docs/README.md` rows are updated. The MADR's status stays `accepted`, as the step does not change it.
  * Step 5 (a dated note after the branch's fast-forward to `main`, with `main`'s CI run) is NOT done. It remains for after the fast-forward and is not a completion criterion.
  * The gates of P0 step 2 were run after the last edit to these three files. Their exit statuses are in the phase's handoff, not written here, because a record edit after the gates would leave them stale.
* Not verified in this phase: no lint or test ran here. Confirmation 4's ten cold runs were measured at `4410333` (P1 step 7), with the same configuration as at `aff572a`. At `aff572a` the evidence is CI's three `validate` jobs and P2 step 1's cold run at `b78503d`.
