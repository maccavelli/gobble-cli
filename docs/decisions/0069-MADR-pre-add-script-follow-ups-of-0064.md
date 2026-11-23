---
status: accepted
date: 2026-11-23
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Close the three loose ends of the pre-add script that 0064 left, and test the script

Associated PLAN: [0069-PLAN-pre-add-script-follow-ups-of-0064.md](0069-PLAN-pre-add-script-follow-ups-of-0064.md)

*(2026-11-23, first draft at `84dad4a`. Three rows were `open`, awaiting the owner's answers to Q1 to Q3.)*

*(2026-11-23, second draft, after an independent review: rows G15 to G19 added (Windows, bash 3.2, preflight's own
gofmt step, the scratch repository's `acpext` directory, what N counts); G8 and the cost lines of Q1 and Q2
corrected. Each finding was read at its line first and held.)*

*(2026-11-23, accepted: the owner answered Q1 to Q3 as recommended, relayed by the coordinator; see "Owner rulings".
No row is open.)*

## Context and Problem Statement

[0064-PLAN-retire-pre-add-golint-add-the-trio.md](0064-PLAN-retire-pre-add-golint-add-the-trio.md) ("P2 pass 2",
"Follow-ups, not built in P2") listed three defects of `scripts/go-precheck.sh` that no decided row of
[0064-MADR-retire-pre-add-golint-add-the-trio.md](0064-MADR-retire-pre-add-golint-add-the-trio.md) covered, and its
close-out left them open "unless the owner pulls them in". The owner's coordinator ledger lists the same three, and
adds "five code-quality items" to the same follow-up list. All three script defects are still true at `84dad4a`; each
was read at its line:

* **F1, gofmt's own exit status is ignored.** `scripts/go-precheck.sh:62` is
  `unformatted="$(gofmt -l "${files[@]}")"`. Only gofmt's standard output is read. A file that does not parse makes
  gofmt print its error to standard error and exit 2 with nothing on standard output, so `unformatted` is empty,
  `failed` stays 0, and with both skip variables set the script ends with `go-precheck: 1 file(s) clean`
  and exit 0. `make preflight` sets both skip variables on this step (`Makefile:196`) and catches the file later
  through `go vet` and lint, so the hole is real for a direct `make pre-add-check` with a skip variable and for any
  caller that runs the script alone.
* **F2, the closing line lists tools that did not run.** `:105` always prints
  `(gofmt, golangci-lint, govulncheck)`. With `GO_PRECHECK_SKIP_LINT=1` or `GO_PRECHECK_SKIP_VULN=1` the matching
  step printed `... skipped` earlier, but the last line still says it was clean. `make preflight` passes both, so its
  log ends the pre-add step with a line that claims two checks that did not run in that step.
* **F3, a path to a deleted Go file is dropped without a word.** `:39` keeps an argument only if
  `[ -f "$f" ]`, so a `.go` argument that does not exist is silently left out, and if it was the only one the script
  exits 0 at once (`:44`). `AGENTS.md` ("Pre-add checks") and `docs/guides/developing.md` say only that `FILES` that
  name no Go file exit 0. A `git add` of a removal is a legitimate way to reach this, so exit 0 may be right; the gap
  is that it is neither said nor shown. In the no-argument mode (`:47-48`) `git ls-files '*.go'` also lists a tracked
  file deleted from the working tree but not yet staged; today it reaches `gofmt -l` and the ignored status (F1)
  hides gofmt's "no such file" error, so fixing F1 alone would turn that case into exit 1 with no decision behind
  it.

The script has no test: 0064's P2 exercised every row on throwaway scratch copies by hand, and nothing in the
repository replays them. A gate that is changed without a test that fails on a broken copy is unused, so this record
also adds the test.

The "five code-quality items" are not script defects and are not described in any record at `84dad4a`: the ledger
names them in one line ("model search by id, an unlocked map, the refusals file across processes, uncancellable
lookups, function-swap test seams"), the review that found them is not among the repository's records, and none has
a line reference. Reading the tree turns up candidates (a shared map in `acpserver/choice.go:299` and
`acpserver/authsignin.go:139`; `internal/cli/refusal.go`, whose mutex guards one process only while the file is read
and rewritten by every process; `context.Background()` at `llm/provider/model.go:79`, `llm/provider/provider.go:221`
and `internal/cli/agentauth.go:53`; package variables replaced by tests, such as `internal/cli/agent.go:35`, `:55`
and `internal/cli/trust.go:220`; the search in `llm/provider/list.go:57` and `internal/cli/modelscmd.go:242`), but a
candidate is a guess about what a reviewer meant, not evidence. They are Go runtime changes with a different test
harness. Row G13 records the owner's ruling (Q3): they are not built here.

## Decision Drivers

* A check that passes while a tool it names failed or did not run is worse than no check (F1, F2).
* One reading of "exit 0 for a file that is gone" must hold in both modes of the script and be written where the
  other exit rules are written (F3).
* Every changed behaviour needs a test seen to fail on a broken copy of the script (the owner's standing rule for a
  gate), and the existing practice for a script's own test is a `--self-test` with plants (`scripts/check_libs.py`).
* The owner's rule is Python for scripts. `go-precheck.sh` is the one bash script here, and `make pre-add-check`,
  `make preflight` and the fleet's wrapper contract name it.
* No backward-compatibility engineering: the owner is the only user.
* CI runs `make preflight` on ubuntu, macos and windows (`.github/workflows/ci.yml:51`, `:96`), so the harness runs on
  all three, and the script must run under bash 3.2 (this Mac's `/bin/bash`, where an empty `"${a[@]}"` under `set -u`
  is an error before bash 4.4).
* Keep the change small: three defects, one test, three document lines.

## Considered Options

* **A. Fix the three defects in `scripts/go-precheck.sh` (bash, in place), and add a Python test harness for it.**
  Cost: about thirty script lines and one new file of about two hundred lines; the script stays bash, against the
  owner's Python rule, until a port.
* **B. Port the script to Python (`scripts/go_precheck.py`) in the same record, fixing the three defects as part of the
  port.** Cost: every row 0064 decided (S1 to S12) must be re-proved on the new file, `Makefile:175` and `:196` and
  both documents change, CI's Windows job runs `make preflight` through this script and has never run a Python
  version of it, and a behaviour difference hides inside a port. No CI file changes (CI runs `make preflight`), but
  the Windows job would run the Python version for the first time. Roughly three times option A's work.
* **C. Fix only F1 and F2, which are plain defects; leave F3 as documented-only.** Cost: leaves the
  no-argument deleted-file case exiting 1 under the F1 fix with no decision behind it (see F3), so it is not a stable
  stopping point.

## Decision Outcome

Chosen option: "A. Fix the three defects in bash, in place, and add a Python test harness", because it closes the
defects with the smallest change that can be proved, and keeps the port to Python a separate, test-first step: the
harness this record adds is written in Python against the script's observable contract (arguments, environment,
exit status, output), so it checks a Python port unchanged when the owner wants one. Q1 asks for the owner's ruling
on the language; the owner chose it (see "Owner rulings").

Decided behaviour (rows G1 to G12 below):

1. gofmt's exit status is read. A non-zero status from gofmt prints gofmt's own message under a `gofmt:` header
   and sets `failed`, whether or not it also listed files.
2. The closing line names only the steps that ran. `go-precheck: N file(s) clean (gofmt, golangci-lint,
   govulncheck).` when all ran; a skipped step is dropped from the parenthesis, so with both skip variables set the
   line reads `(gofmt)`. The existing `... skipped` lines on standard error are unchanged.
3. A deleted Go path is skipped with a one-line notice naming it on standard error, and the script goes on (exit 0
   when nothing else fails), the same in both modes of the script (owner, Q2).
4. A new `scripts/check_go_precheck.py` (`--self-test`, and `--script PATH` to run its tests against another copy of
   the script) drives the script in a scratch git repository (with an `acpext` directory, which the lint step enters)
   with a stub directory holding `go` and `golangci-lint` in front of the real `PATH` and the real `gofmt`, under the
   `PATH` `bash` and also under `/bin/bash` where that is a different program, and checks each decided row. It does
   not build a `PATH` of symlinks, which is fragile on Windows. `make check-go-precheck` runs it, and `make
   preflight` calls it.

### Consequences

* Good, because a malformed Go file can no longer give `1 file(s) clean` and exit 0.
* Good, because the last line of a log says what ran.
* Good, because the script's contract, kept until now in prose and one-off scratch runs, is replayed on every
  preflight, including on the Windows and macOS jobs.
* Bad, because the script stays bash for now; Q1 records that choice.
* Bad, because one more script and one more preflight step exist to maintain; the harness runs in seconds, offline,
  with stubs, and needs no network.
* Neutral, because `make pre-add-check`'s arguments, exit codes (`0`, `1`, `2`) and environment variables do not
  change.

### Confirmation

1. `scripts/check_go_precheck.py --self-test` exits 0 on the tree and each of its planted breaks (one per row, in the
   plan) exits non-zero naming the row.
2. `make preflight` passes and its pre-add step ends with `(gofmt)`.
3. `AGENTS.md` and `docs/guides/developing.md` state the closing-line rule and the deleted-path rule in the words of
   rows G5 to G8.

## Pros and Cons of the Options

### A. Fix in place, add a Python harness

* Good, because the smallest change, and every line changed has a row and a plant.
* Good, because the harness outlives a port.
* Bad, because the script is still bash.

### B. Port to Python now

* Good, because it meets the Python rule at once.
* Bad, because a port is a rewrite of a gate that CI runs on three operating systems; its twelve decided rows from
  0064 would be re-proved at the same time as three new ones, so a regression cannot be told from a defect.
* Bad, because the port's own record (shell on Windows runners, `bash` named in the Makefile) is a decision in its
  own right.

### C. Fix F1 and F2 only

* Good, because it is the smallest edit.
* Bad, because it leaves F3 documented-in-neither-place, and F1's fix then changes the no-argument deleted-file case
  to exit 1 by accident.

## State table

One row per input or state the script and the new harness can meet. `decided` rows give the outcome and what pins it
(plan step).

| # | Input or state | Status | Outcome | Pinned by |
| --- | --- | --- | --- | --- |
| G1 | A named Go file that does not parse, both skip variables set | decided (F1) | Exit 1; gofmt's message printed under a `gofmt:` header; no `clean` line. | plan P1-T1 |
| G2 | A named Go file that parses but is unformatted | decided, unchanged | Exit 1; the existing `gofmt: these files are not formatted` text and the file named. | plan P1-T2 |
| G3 | gofmt exits non-zero and also lists a file (one unparseable file and one unformatted file in one add) | decided (F1) | Exit 1; both the gofmt error and the unformatted list print. | plan P1-T3 |
| G4 | All three steps ran, files clean | decided (F2) | Exit 0; `go-precheck: N file(s) clean (gofmt, golangci-lint, govulncheck).` | plan P1-T4 |
| G5 | `GO_PRECHECK_SKIP_LINT=1` | decided (F2) | Exit 0 when clean; the closing line is `(gofmt, govulncheck)`; the `golangci-lint: skipped` line is unchanged. | plan P1-T5 |
| G6 | `GO_PRECHECK_SKIP_VULN=1` | decided (F2) | The closing line is `(gofmt, golangci-lint)`. | plan P1-T5 |
| G7 | Both skip variables set (what `make preflight` sets) | decided (F2) | The closing line is `(gofmt)`. | plan P1-T5 |
| G8 | A named `.go` path that does not exist, with or without other Go files, and a tracked `.go` file missing from the working tree in the no-argument mode; the script changes to the repository root first (`:32-33`), so a path written relative to a subdirectory also lands here | decided (F3, Q2) | The path is skipped with a one-line notice naming it on standard error; exit 0 if nothing else fails; the same in all these situations. Owner, relayed by the coordinator, 2026-11-23: "Skip with a notice, exit 0 (Recommended)". The outcome decided is written into `AGENTS.md` and `docs/guides/developing.md`. | plan P1-T6 (pins the answer) |
| G9 | Arguments that hold no Go file (docs only) | decided, unchanged (0064-MADR S1) | Exit 0 at once; no `go` or `golangci-lint` call (the stubs count zero). | plan P1-T7 |
| G10 | Non-Go arguments mixed with Go files | decided, unchanged | The non-Go arguments are ignored; gofmt sees only the Go files. | plan P1-T7 |
| G11 | `gofmt` not on `PATH` | decided, unchanged | `go-precheck: gofmt not found in PATH.`, exit 2, and nothing else on standard error. *(Amended 2026-11-23, PLAN deviations 1 and 2: the harness removes every directory on `PATH` that holds a `gofmt`; a second case, T8b, assembles a `PATH` where gofmt shares one directory with `mktemp` and the other tools, as inferred for the CI runner, so a guard that is gone cannot hide behind a later `mktemp` exit 2.)* | plan P1-T8, T8b |
| G12 | The harness run against a broken copy of the script | decided | Exit non-zero naming the failed row; this is the plant of each test. | plan, "Plants" |
| G15 | The harness on the Windows job (git-bash) | decided | Runs there as on the other two: stub directory in front of the real `PATH`, real `gofmt`, no symlinks. Its first Windows run is the preflight in CI after the owner's push; the plan records that it was not run on Windows by hand. | plan P1 (harness design), P5 |
| G16 | The script under bash 3.2 (`/bin/bash` on this Mac) | decided | Every test also runs under `/bin/bash` where it differs from `bash` on `PATH`; no new `"${a[@]}"` on a possibly empty array under `set -u`. A plant puts one back and T9 catches it. | plan P1-T9, PL7 |
| G17 | `make preflight`'s own gofmt step (`Makefile:179-182`, `unformatted="$$(gofmt -l .)"`) ignores gofmt's status the same way | out of scope | Preflight's next step, `pre-add-check` over every Go file (`Makefile:196`), fails on the same file once G1 is fixed (its exit 1 aborts the recipe), so preflight cannot pass; the early step would only change which line reports it, and fixing it duplicates the step. The plan's P5 runs preflight on a scratch copy with an unparseable file to show it fails. | plan P5 |
| G18 | What N in the closing line counts | decided | The Go files actually checked (`${#files[@]}`, existing files only); a skipped missing path is not counted and is named in its notice. | plan P1-T6 |
| G19 | The scratch repository for the tests | decided | Holds an `acpext` directory so the lint step's `cd acpext` succeeds with the stub; a base run that is red for a missing directory is a harness defect, not the expected red. | plan P1 |
| G13 | The five code-quality items of the ledger's follow-up list | decided (Q3) | Not built in 0069. Own record later, if the review's text is found. Owner, relayed by the coordinator, 2026-11-23: "Own record later, if text found (Recommended)". | none; no code or test of 0069 touches them |
| G14 | The script's language (bash or a Python port) | decided (Q1) | Fixed in bash in place; a port to Python is its own later record. Owner, relayed by the coordinator, 2026-11-23: "Fix in bash now, port later (Recommended)". | plan P2 |

0064-MADR's rows S1 to S12 stand; this record does not reopen them. S8 stays out of scope.

## Owner rulings

Relayed by the coordinator, 2026-11-23; each label is quoted exactly.

* **Q1, script language (G14):** "Fix in bash now, port later (Recommended)".
* **Q2, a path to a deleted Go file (G8):** "Skip with a notice, exit 0 (Recommended)". The cost stands: a mistyped
  path, or one written relative to a subdirectory (the script works from the repository root), passes with only the
  notice.
* **Q3, the five code-quality items (G13):** "Own record later, if text found (Recommended)".

No row is `open`.

## More Information

* Evidence at `84dad4a`: `scripts/go-precheck.sh:39`, `:44`, `:47-48`, `:62`, `:105`; `Makefile:175`, `:196`;
  `AGENTS.md` "Pre-add checks"; `docs/guides/developing.md` "The gates".
* The harness style follows `scripts/check_libs.py --self-test`; the plan's plants follow
  [0064-PLAN-retire-pre-add-golint-add-the-trio.md](0064-PLAN-retire-pre-add-golint-add-the-trio.md) P2-6.
* Sibling pre-add wrappers in other fleet repositories are not changed by this record.
