---
status: accepted
date: 2026-11-16
decision-makers: repository owner
consulted: "the root-cause measurements of 2026-11-15 on this Mac, in scratch clones at d82abd0 and 9e21c8f, golangci-lint v2.14.0 with revive v1.17.0, recorded under More Information; CI run 43735397427 on d82abd0; 0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md Deviation 4; the source of revive v1.17.0 and of golangci-lint v2.14.0's revive wrapper in the module cache; checks made while drafting, outside the working tree (More Information); the owner's rulings of 2026-11-15, relayed by the cross-repository coordinator"
informed: "0007-MADR-repository-scaffolding-to-fleet-standard.md, whose D7 kept the test-file exclusion this record changes in part; 0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md, whose Deviation 4 found the failure; the cross-repository coordinator, whose briefs for this repository carry the 'six cold lint runs' and 'no capitalization-only names' rules"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Stop excluding revive's `confusing-naming` in test files, so the lint verdict does not depend on file order

## Context and Problem Statement

On 2026-11-15 commit `d82abd0` passed `make preflight` on the Linux machine and `make lint` on this Mac, and CI run
43735397427 on the same commit failed in `make preflight`'s lint on `validate (ubuntu-24.04)` and `validate (macos-15)`, with
`validate (windows-2025)` green:

```text
llm/provider/login.go:142:6: confusing-naming: Method 'loginOf' differs only by capitalization to function 'LoginOf' in llm/provider/export_test.go (revive)
```

`9e21c8f` renamed the test helpers (`LoginOf` to `LoginFromSDKSession`, `SessionOf` to `SDKSessionOf`), and
0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md Deviation 4 recorded the finding as intermittent. The
owner's ruling of 2026-11-15 was "Root-cause it, then decide". This record gives the root cause, the measurements, and
the decision that followed.

### Why the verdict depends on file order

Three pieces of code combine. Line numbers are revive v1.17.0's and golangci-lint v2.14.0's, read in the module cache
(`<modcache>`).

1. **revive keeps one name table per package, and the second name reports.** `rule/confusing_naming.go:45` holds a
   package-level table, `var allPkgs = packages{pkgs: make([]pkgMethods, 1)}`, keyed by the `*lint.Package`; within it,
   names are keyed by receiver type (`_` for plain functions) and by `strings.ToUpper(name)` (`:82`). `checkMethodName`
   (`:75-118`) reports a failure when the upper-cased name is already in the table (`:87-111`), and otherwise records it
   (`:117`). So of two names that differ only by case, the one checked first is recorded and the one checked second is
   reported, at its own file and line.
2. **revive lints a package's files concurrently, in map order.** `lint/package.go:56-61`, `Files()` returns
   `map[string]*File`; `:212-219` starts one goroutine per file:

   ```go
   for _, file := range p.Files() {
       eg.Go(func() error {
           return file.lint(rules, config, failures)
       })
   }
   ```

   Go randomises map iteration, and the goroutines race for the table's lock, so which file is checked first changes
   from run to run.
3. **gobble drops every revive finding in a test file.** `.golangci.yml:142-149`:

   ```yaml
         - linters:
             - errcheck
             - gosec
             - unparam
             - revive
             - gocritic
             - goconst
           path: _test\.go
   ```

When a pair is split between a production file and a `_test.go` file of the same package, the finding lands on the
production file in some runs (reported, red) and on the test file in others (dropped, green). golangci-lint v2.14.0's
wrapper (`pkg/golinters/revive/revive.go:109-113`) passes each analysis pass's files to revive as one package, and
writes each finding's text as `fmt.Sprintf("%s: %s", failure.RuleName, failure.Failure)` (`:137`), which is what the
chosen option keys on.

The revive version is established two ways: `go version -m` on the installed golangci-lint binary reports `mod
github.com/golangci/golangci-lint/v2 v2.14.0` and `dep github.com/mgechev/revive v1.17.0`, and golangci-lint v2.14.0's
own `go.mod` in the module cache requires `github.com/mgechev/revive v1.17.0` (line 102). The Makefile (`:132-138`)
and CI (`.github/workflows/ci.yml:73-77`) both install `golangci-lint@v2.14.0`, which resolves the same requirement.

### Measured (2026-11-15, this Mac, cold cache unless stated)

Each run had a new `GOLANGCI_LINT_CACHE`. "Package" is `golangci-lint run --config .golangci.yml ./llm/provider/...`;
"root module" is `golangci-lint run --config .golangci.yml ./...` in the root; `make lint` runs `./...` in both modules,
`.` and `acpext`, as the gate does. "Today's" is the configuration above; options C, D and E are described under
Considered Options.

| Batch | Tree | Config | Command | Runs | Red | Where the finding landed |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | `d82abd0` | today's | package | 20 | 4 | `login.go:142` each time |
| 2 | `d82abd0` | today's | package, `-j 1` | 20 | 9 | `login.go:142` |
| 3 | `d82abd0` | today's | package, `GOMAXPROCS=1` | 20 | 12 | `login.go:142` |
| 4 | `d82abd0` | today's | root module | 20 | 11 | `login.go:142` |
| 5 | `9e21c8f` | today's | root module | 20 | 0 | none |
| 6 | `9e21c8f` | today's | `./...` in `acpext` | 20 | 0 | none |
| 7 | `d82abd0` | today's | package, 8 caches, 5 runs per cache (warm after the first) | 40 | 30 | 6 caches red on all 5 runs, 2 caches green on all 5 |
| 8 | `d82abd0`'s `llm/provider` | revive's rule alone, called directly | 500 iterations | 500 | — | `login.go` 301, `export_test.go` 199 |
| 9 | scratch clone (note 1), `d82abd0`'s two test files restored | today's / C / E | package | 20 each | 14 / 20 / 20 | C: `login.go` 13, `export_test.go` 7; E: 19 and 1 |
| 10 | the same, plus two `_test.go` files declaring `PlantPair` and `Plantpair` | today's / C / E | package | 10 each | 0 / 10 / 10 | split between the two test files |
| 11 | the same, plus a `_test.go` file returning `errors.New("Planted Error.")` | today's / C / E | package | 5 each | 0 / 0 / 0 | none; no run with the exclusion removed, so not seen to fire |
| 12 | scratch clone (note 1), unplanted | today's / C / E, interleaved | `make lint` | 10 each | 0 / 0 / 0 | "0 issues." for both modules; wall seconds mean 40.6 / 37.0 / 37.4 (whole seconds; today's first run 50) |
| 13 | — | D | run | 1 | refused | `can't load config: error in exclude rule #1: invalid text regex: error parsing regexp: invalid or unsupported Perl syntax: (?!` |

Note 1: the clone used for batches 9 to 12 no longer exists and its commit is not in the outputs. Its scripts restore
the two test files to `9e21c8f`'s versions after batch 9, so it is taken to be at `9e21c8f`. That is an inference.

What the batches establish:

* The finding lands on either file, and today's configuration turns that into a red or a green run: red in 4, 9, 11,
  12 and 14 of 20 across five batches. Six cold runs are not a guarantee of seeing it.
* Neither `-j 1` (golangci-lint's own concurrency) nor `GOMAXPROCS=1` fixes the order (batches 2 and 3).
* A warm cache keeps whichever outcome its first run had (batch 7). A clean local run on a warm cache says nothing
  about the next cold run, and CI starts without a golangci-lint cache: `ci.yml` has no step that restores one (read,
  not measured).

### What the rule counts, and the four existing pairs

revive's rule looks only at function and method declarations (`Visit`, `:191-209`: `*ast.FuncDecl`) and at the fields
of one struct type (`checkStructFields`, `:167-189`). Types, constants and variables are not in its table. An
independent review found four case-only pairs in the tree: `Selection` (a type) and `selection` (a function) in
`llm/provider/ambient.go:20,69`; `RequestPermissionsTool` and `requestPermissionsTool` (two constants) in
`acpserver/grants.go:24,26`; `PromptError` (a type) and `promptError` (a function) in `acpclient/prompterror.go:17,43`;
`ToolContent` (a type) and `toolContent` (a function) in `acpclient/update.go:225,355`. None is a pair of functions or
methods, all eight names are in non-test files, and so none is a finding under today's configuration or the chosen
one. The drafting scan under More Information agrees: 0 findings at `c4c5ec0` and at `9e21c8f`.

### Where the exclusion came from

No record decided it on its merits. `.golangci.yml` was written by
`0004-PLAN-go-module-architecture.md` Phase 0 step 7 ("Write `.golangci.yml` (v2) with mcplib's linter set"), in
`925ef0a`, and the test-file rule is mcplib's, byte for byte. `0007-MADR-repository-scaffolding-to-fleet-standard.md`
D7 brought the linter set up to mcplib's and said "Leave everything else unchanged", which kept the rule.

### What it costs today

Since that failure, as the coordinator reports, briefs for this repository carry two rules: six cold lint runs per
phase, and no names that differ only by capitalization. The coordination ledger's brief for 0051 P4 names the first
("six cold lint runs"), and 0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md's later gate tables
show the loops (for example "six runs, each with a new `GOLANGCI_LINT_CACHE`, under `run-heavy`"). Neither rule is in a
record of this repository.

## Decision Drivers

* A gate's verdict must not depend on scheduling. A lint that is green on one run and red on the next teaches every
  reader to rerun it until it passes.
* What passes locally on a cold cache must pass in CI, and the reverse.
* The owner's rulings of 2026-11-15: root cause first; then "Stop excluding that rule in test files"; then, for the
  mechanism, "Text-keyed exclusion".
* No workaround: no warm cache, no pinned order, no `//nolint`, no loop that reruns until green.
* No gate gets weaker: everything reported today is still reported.
* The cost of keeping the configuration right when revive's rule list changes.

## Considered Options

* **A.** Keep the exclusion as it is, and rely on naming discipline and repeated cold runs.
* **B.** Stop excluding revive in test files altogether.
* **C.** Split the exclusion, and key revive's part on the finding's text, naming every enabled revive rule except
  `confusing-naming` (the text-keyed exclusion).
* **D.** Key revive's part on a negative lookahead, `text: '^(?!confusing-naming)'`.
* **E.** Drop revive from the path exclusion, and give each of the 22 other revive rules revive's own `exclude: [TEST]`.
* **F.** Pin the outcome instead: a warm cache, or a fixed file order.

## Decision Outcome

Chosen option: "C", because it makes `confusing-naming` report in every run whichever file it lands on (20 of 20 cold
runs against 14 of 20 in batch 9), it keeps the other 22 revive rules out of test files as today, it adds no time
(batch 12), it reports strictly more than today and nothing less, and it is the mechanism the owner chose.

The owner's rulings of 2026-11-15, relayed by the coordinator, in order, each a label the owner chose from options the
coordinator put:

* "Root-cause it, then decide" (before this record).
* "Stop excluding that rule in test files". This record reads it as ruling out A and F, which keep the exclusion, and
  B, which stops excluding every revive rule rather than that one.
* "Text-keyed exclusion": the mechanism, option C here.
* "Draft it on the branch now": where this record is drafted, the branch `p3-acpext-v0.5.0`.

Which options the coordinator put beside each label is not recorded here; the labels are quoted as relayed.

On 2026-11-15 the owner approved this pair with the label "Approve; run it at phase boundaries", relayed by the coordinator; the pair was drafted outside the working tree on 2026-11-15 and placed on the branch `p3-acpext-v0.5.0` on 2026-11-16, after 0051-PLAN's P5 was committed there as `17f2d6c`.

The decisions:

* **D1. The exclusion is split.** `.golangci.yml`'s first exclusion rule loses `revive`, and a second rule covers revive
  in test files by text:

  ```yaml
        - linters:
            - errcheck
            - gosec
            - unparam
            - gocritic
            - goconst
          path: _test\.go
        - linters:
            - revive
          path: _test\.go
          text: '^(atomic|blank-imports|constant-logical-expr|context-as-argument|context-keys-type|dot-imports|empty-block|error-naming|error-return|error-strings|errorf|if-return|increment-decrement|indent-error-flow|range|receiver-naming|redefines-builtin-id|superfluous-else|time-naming|unexported-return|unreachable-code|var-declaration): '
  ```

  Each rule carries a comment that cites this record. The plan gives the comment text.
* **D2. The list is revive's enabled rules minus `confusing-naming`.** It names, in `revive.rules`' order, every rule
  under `linters.settings.revive.rules` except `confusing-naming`: 22 today (`.golangci.yml:116-138` lists 23). A rule
  added to or removed from `revive.rules` is added to or removed from the list in the same change. `confusing-naming`
  never joins it.
* **D3. Test files now get `confusing-naming` findings, and they are fixed by renaming.** Two kinds are new: two
  functions or methods of one package, both in test files, whose names differ only by case; and two fields of a struct
  declared in a test file whose names differ only by case. A finding is fixed by a rename, never by `//nolint`: the
  finding moves between the two files from run to run, so a `//nolint` on one of them would itself be order-dependent.
* **D4. The two brief rules lose their reason.** Once the plan's checks hold, the "six cold lint runs" rule has nothing
  to catch, and the linter enforces the part of "no capitalization-only names" it can see. Retiring them is the owner's
  (Q1).
* **D5. Nothing else changes.** The golangci-lint pin stays `v2.14.0` (0023-MADR-pin-current-releases.md). D1 depends on
  the wrapper's text format (`revive.go:137`); a change of the pin repeats the plan's plants before it lands.

### Consequences

* Good, because the verdict on a `confusing-naming` pair no longer depends on file order, cache, host or CI runner: the
  pair is reported on every run (batch 9, C: 20 of 20).
* Good, because nothing reported today is dropped: non-test files are unchanged, and in test files the exclusion drops a
  subset of what it drops today.
* Good, because the six-run lint loop per phase can go (Q1), and a single clean `make lint` again means what it says
  for this rule.
* Neutral, because the file that carries the finding still varies (batch 9, C: `login.go` 13 times, `export_test.go`
  7); only the verdict is fixed. Instructions that name "the line" must say "either name of the pair".
* Bad, because pairs wholly inside test files, and case-only fields in a test-file struct, become findings (batch 10, C:
  10 of 10). Today's tree has none (batch 12, and the drafting scan), so the cost is to future test code.
* Bad, because the list in D1 must track `revive.rules` by hand. Drift is loud or harmless in every direction but one:
  a rule added to `revive.rules` and not to the list applies to test files too, and turns the lint red if a test file
  breaks it; a rule removed and left in the list is dead text; a golangci-lint upgrade that changes the text format
  stops the list matching, so all 22 rules report in test files. The one silent and harmful drift is someone adding
  `confusing-naming` to the list, which restores today's fault. The comment of D1 warns against it; whether a check
  should also catch it is Q2.
* Bad, because D1 ties gobble's configuration to golangci-lint's `"<rule>: <message>"` text, an implementation detail of
  its wrapper, not a documented contract (D5).

### Confirmation

The plan confirms the decision; each check is seen to fail first where it can fail.

1. A planted `error-strings` violation in a test file fails `make lint` with revive removed from the exclusion, and
   passes with today's configuration and with D1's (closes the gap of batch 11).
   **Met** (P1 step 3, record of 2026-11-16): `none.yml` 5 of 5 red, each with the `error-strings` line at `llm/provider/zz_other_test.go:6:43`; `today.yml` and `new.yml` 5 of 5 green, "0 issues." for both modules.
2. The `d82abd0` collision, planted on a scratch clone, fails 20 of 20 cold `make lint` runs with D1's configuration,
   the command the gate runs (batch 9 used the package command).
   **Met** (P1 step 4, record of 2026-11-16): `new.yml` 20 of 20 red, one `confusing-naming` line each, split 11 at `llm/provider/login.go` and 9 at `llm/provider/export_test.go`; acpext "0 issues." each time.
3. With the collision planted and D1's configuration, a warm cache gives a red run every time.
   **Met** (P1 step 6, record of 2026-11-16): 9 of 9 red on three caches, one `confusing-naming` line each.
4. The unplanted tip of the branch passes 10 of 10 cold `make lint` runs with D1's configuration, both modules
   "0 issues.".
   **Met** (P1 step 7 at `4410333`, 10 of 10 cold runs exit 0 with `new.yml`, "0 issues." for both modules; P2 step 1, one cold run at `b78503d` with D1, exit 0; CI on `aff572a`, the three `validate` jobs with "0 issues." for both modules, record of 2026-11-16).
5. The list in D1 equals `revive.rules` minus `confusing-naming`, checked by script.
   **Met** (P1 step 2, and P2 step 1's one-off check of the committed file, record of 2026-11-16): the assertion passes on `new.yml` and fails on both broken copies; the file as committed passes, exit 0, and fails on its two broken copies, exit 1.
6. CI on draft pull request 18 is green on the commit that carries D1, on all three `validate` runners.
   **Met** (P3 step 1, record of 2026-11-16): the commit carrying D1 is `aff572a`; `CI` run `43990126887` on it has `validate (ubuntu-24.04)`, `validate (macos-15)` and `validate (windows-2025)` each `success`, and `gh pr checks 18` shows every check `pass` except `release`, `skipping`.

## Pros and Cons of the Options

### A. Keep the exclusion, and rely on naming discipline and repeated cold runs

* Good, because nothing changes in the repository.
* Bad, because the verdict stays order-dependent: today's configuration was red in 4 to 14 of 20 cold runs on the same
  pair, so six runs can all be green; `d82abd0` passed the local gates and failed CI.
* Bad, because every phase pays for a six-run loop (and a warm cache makes one local run worthless, batch 7).
* Bad, because "no capitalization-only names" is a rule for people, and the linter cannot see most of it: the four pairs
  of a type or constant already in the tree break it and are not findings.

### B. Stop excluding revive in test files altogether

* Good, because the configuration is simplest: no list to keep in step with `revive.rules`.
* Good, because `confusing-naming` is reported on every run, as in C.
* Bad, because 22 rules start checking test files at once. How many findings that gives today is not measured; the
  plan's P1 step 3 counts them as a by-product.
* Bad, because it goes past the owner's ruling, which named one rule.

### C. Text-keyed exclusion (chosen)

* Good, because measured: 20 of 20 on the planted collision, 10 of 10 on a test-only pair, today's tree clean on both
  modules, no time added (batches 9, 10, 12).
* Good, because the change is one exclusion rule in one file, and every finding of today is still reported.
* Neutral, because the other 22 rules' exclusion in test files was not seen to fire on gobble's tree (batch 11); a
  scratch-module check while drafting saw it fire (More Information), and the plan repeats it on the tree.
* Bad, because the list must track `revive.rules`, and D1 depends on golangci-lint's text format (Consequences).

### D. Negative-lookahead exclusion

* Good, because it would name only the one rule, with nothing to keep in step.
* Bad, because golangci-lint compiles `text` with Go's `regexp` (RE2), which has no lookahead: the run refuses the
  configuration (batch 13). While drafting, `golangci-lint config verify` exited 0 on the same file, so the refusal
  shows only when the lint runs.

### E. revive's per-rule `exclude: [TEST]`

* Good, because measured equal to C: 20 of 20, 10 of 10, clean, no time added (batches 9, 10, 12).
* Good, because the exclusion sits on each rule's own line, so a reader of `revive.rules` sees it.
* Neutral, because it has the same drift as C: a rule added without `exclude` applies to test files.
* Bad, because 22 lines change instead of one rule, and it relies on revive's own file filter (`lint/filefilter.go:67`,
  `TEST`) rather than golangci-lint's exclusions, where the repository's other exclusions live.
* Bad, because the owner chose C.

### F. A warm cache, or a fixed file order

* Good, because no configuration changes.
* Bad, because it is a workaround: it hides the order dependence instead of removing it. A warm cache keeps the first
  run's outcome, red or green (batch 7: 6 caches red, 2 green); CI starts cold.
* Bad, because no knob fixes the order: `-j 1` and `GOMAXPROCS=1` still gave 9 and 12 red of 20 (batches 2, 3), and
  revive's file order comes from Go's randomised map iteration.

## More Information

### Checks made while drafting (2026-11-15, outside the working tree)

The working tree was busy with another phase, so nothing was run in it. These ran on this Mac in a scratch directory:

* **`config verify` and D.** Three configurations generated from `c4c5ec0`'s `.golangci.yml` (today's, C, and D):
  `golangci-lint config verify --config <file>` exited 0 on each. `golangci-lint run` with D exited 3 with batch 13's
  message.
* **The other rules' exclusion, seen to fire on a toy module.** A two-file scratch module: `a.go` with `func helper()`,
  `a_test.go` with `func Helper()` and `errors.New("Planted Error.")` (revive's `error-strings` reports a message ending
  in `.` at confidence 0.8, `rule/error_strings.go:180-186`, which golangci-lint keeps: it drops only confidence below
  the configured value, `revive.go:119`, which defaults to revive's 0.8, `revive.go:279` and revive's
  `config/config.go:331`). Eight cold runs per configuration:

  | Config | Red | `error-strings` at `a_test.go` | `confusing-naming` |
  | :--- | :--- | :--- | :--- |
  | revive removed from the path exclusion, no text rule | 8 | 8 | 8 (6 at `a.go`, 2 at `a_test.go`) |
  | today's | 7 | 0 | 7 (all at `a.go`) |
  | C | 8 | 0 | 8 (6 at `a.go`, 2 at `a_test.go`) |

  This shows the mechanism works, not that it works on gobble's tree; the plan's P1 repeats it there.
* **A static scan of the case-only pairs.** A program built against revive v1.17.0 ran the `confusing-naming` rule over
  `git archive` exports of `c4c5ec0` and `9e21c8f`, files grouped as `go/build` selects them for `darwin`, `linux` and
  `windows` (each package; each package with its in-package tests; each external test package; 93 groups): 0 findings
  for each commit and each OS. The same program, as its own check, found the `d82abd0` pair (1 finding), and on a planted
  copy a test-only function pair and a case-only field pair in a test-file struct. This is the rule's verdict, not
  golangci-lint's; the plan's runs are the evidence.

### Facts, assumptions, and what was not measured

* Facts: the code lines quoted above; batches 1 to 13; the drafting checks.
* Assumption: batches 9 to 12 ran on `9e21c8f` (note 1). The branch tip at drafting is `c4c5ec0`, 76 files later; the
  plan's P1 measures the tip.
* Assumption: golangci-lint applies exclusion rules after its cache, so a warm cache cannot keep an excluded outcome
  under D1. Not measured; the plan's P1 step 6 measures it.
* Not measured: option B's findings on today's tree; CI's cache state (read from the workflow only); options C and E
  with `make lint` on a planted case (batches 9 to 11 used the package command, while today's configuration was measured
  with the root module's command in batch 4).

### Open questions for the owner

* **Q1. When do the two brief rules retire (D4)?**
  * (a) Both, when the plan's P1 checks hold and CI on draft pull request 18 is green on the commit that carries D1
    (the plan's P3). *Recommended:* from then on a single cold `make lint` decides a `confusing-naming` pair, and the
    linter enforces the case-only rule for functions, methods and struct fields.
  * (b) Retire "six cold lint runs" then, and keep "no capitalization-only names" as a naming rule for people. Cost:
    the four pairs of a type or constant already in the tree break it, so it needs an exception or four renames, and
    nothing checks it.
  * (c) Keep both until D1 reaches `main` with the fast-forward after 0051-PLAN P6 and 0052-PLAN P4. Cost: the
    six-run loop stays on every phase until then, with nothing left for it to catch.

  Answer, 2026-11-15 (the owner's ruling, relayed by the coordinator): option (a), label "Both, once the fix is proven".
* **Q2. Should a check catch drift in D1's list (Consequences)?**
  * (a) No new check: D1's comment, and the plan's one-off check that the list equals `revive.rules` minus
    `confusing-naming`. *Recommended:* every drift but one is loud or harmless, and the one (adding
    `confusing-naming` to the list) is a deliberate edit against the comment.
  * (b) A permanent check run by `make preflight` (for example a few lines in an existing `scripts/` check, or a test in
    `internal/archtest`) that the list equals `revive.rules` minus `confusing-naming`. Cost: a new gate, seen to fail
    on a planted list before it is trusted, and a plan phase of its own; it would catch the one silent drift.

  Answer, 2026-11-15 (the owner's ruling, relayed by the coordinator): option (a), label "Comment and one-off check only".

### Relationship to earlier records

* `0007-MADR-repository-scaffolding-to-fleet-standard.md` D7 kept mcplib's exclusion unchanged; this record supersedes
  that part of D7, revive's entry in the test-file exclusion, and leaves the rest of D7 standing. The plan adds a dated
  note there.
* `0007-PLAN-repository-scaffolding-to-fleet-standard.md` C3 says "No `//nolint`, linter exclusion or `-text` attribute
  is added to make a phase green." D1 changes a linter exclusion by decision, not to make a phase green, and it narrows
  what is excluded.
* `0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md` Deviation 4 found the failure and fixed it by a
  rename; the plan adds a dated note there that points here.

### Related

* [0059-PLAN-stop-excluding-confusing-naming-in-test-files.md](0059-PLAN-stop-excluding-confusing-naming-in-test-files.md)
* [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md), Phase 0 step 7, which wrote `.golangci.yml`.
* [0007-MADR-repository-scaffolding-to-fleet-standard.md](0007-MADR-repository-scaffolding-to-fleet-standard.md), D7.
* [0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md), Deviation 4.
* [0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md), C6 and C8, the branch and its gates.
* [0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md), the golangci-lint pin.
