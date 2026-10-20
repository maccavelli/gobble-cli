---
status: accepted
date: 2026-10-20
decision-makers: repository owner
consulted: "GitHub's workflow syntax reference (path filters and the filter pattern cheat sheet), read 2026-10-20; the Actions timing and jobs APIs for runs 39461740658 and 39441632748; go-selfupdate-lib's docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md"
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Skip the full CI for a push or pull request that changes only the documentation, and check it with a one-job docs workflow

## Context and Problem Statement

Every push to `main` and every pull request runs all of `.github/workflows/ci.yml`, whatever it changes: the three-OS
`validate` matrix, `sbom`, and the release rehearsal with its five identity legs. A commit that changes only a record,
an index row or a guide gets the same run. Of the last 60 commits on `origin/main`, 25 changed only documentation
files (the docs set below), among them `9188f63`, whose push ran the full CI.

The owner asked, on 2026-10-20, to stop the full runs for Markdown-only pushes and pull requests, as go-selfupdate-lib
did the same day in its `docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md`. That record chose
`paths-ignore: ['**.md']` on its CI's `push` and `pull_request` triggers, and a one-job `docs.yml` running the Markdown
checks its CI runs; a docs-only push there ran "Docs" in 12 s instead of 18 jobs and 15–20 minutes, and its CI bills
no minutes, being public. This record decides the same question for gobble-cli, whose facts differ in one place: here
some tracked Markdown is product code.

Each fact below was observed on 2026-10-20, at `bdd8268` (the CI files are unchanged since `origin/main`, `d76e3d2`).

**What CI runs** (`.github/workflows/ci.yml`):

- Triggers: `push` to `main` and to tags `v*`, and every `pull_request` (`ci.yml:2-6`). No schedule, no
  `workflow_dispatch`. One concurrency group per ref, `ci-${{ github.ref }}`, cancelling an older run (`ci.yml:15-17`).
- `validate`, on `ubuntu-24.04`, `macos-15` and `windows-2025` (`ci.yml:26-102`): `go test -count=1 -artifacts`
  (`:57-61`), `make preflight` (`:70-72`), `make archtest` (`:73-75`); `make race` on Linux and macOS (`:78-81`); on
  Linux, `go mod tidy -diff` (`:82-84`) and a pinned shellcheck with actionlint (`:87-102`).
- `sbom` takes the dependency graph's SPDX SBOM, and on a tag fails unless the tag is `main`'s head (`ci.yml:107-141`,
  `:119-125`).
- `build` needs `validate` and `sbom` and calls go-selfupdate-lib's build workflow at `v1.17.0` (`ci.yml:143-148`); off
  a tag it is a rehearsal. 0004-PLAN replaced `make release-dry-run` with "the CI rehearsal of every push"
  ([0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md), note of 2026-10-07 under Phase 4).
  *(2026-10-20, corrected at P2 by the owner's choice: the quoted phrase is the note of 2026-10-07 under 0004-PLAN's
  `## Verification`; the note of that date under Phase 4 says "the build workflow's rehearsal on every push and pull
  request", and the amendment "Phase 4 steps 2–4 made executable" says "every push and pull request rehearses". All
  three say the rehearsal runs on every push, and D4's one note, after the Phase 4 note, covers them.)*
- `release` needs `build` and runs only when the build is not a rehearsal, that is on a `v*` tag (`ci.yml:150-163`).
- `.github/workflows/` holds `ci.yml` only. The "Dependency Graph" runs in the run list are GitHub's own automatic
  submission (event `dynamic`), not a workflow file of this repository.

**What checks the documentation.** `make preflight` runs, last, `check-records` and `markdownlint` (`Makefile:188-189`):

- `check-records` is `$(PYTHON) scripts/check_records.py --check-all` (`Makefile:220-221`): record naming, pairing and
  placement, and every relative link in the records, `docs/README.md`, `docs/architecture.md`, `docs/guides/**` and the
  root `README.md`. It walks the filesystem, not git, and needs only Python's standard library.
- `markdownlint` is `npx --yes markdownlint-cli2@0.23.2` (`Makefile:229-230`), with the fleet config, which excludes
  MADR and PLAN files by its own globs (`.markdownlint-cli2.jsonc`).
- CI runs both through `make preflight` on all three runners. No other step reads documentation. Skipping CI outright
  would skip these two checks.

**Some tracked Markdown is product code or test input.** Of 91 tracked `.md` files, 36 are read by Go:

- `tool/builtin/describe/*.md`, 17 files, are embedded (`tool/builtin/describe.go:16`, `//go:embed describe/*.md`)
  and parsed as `text/template` at construction (`describe.go:37-38`): the built-in tools' descriptions the model is
  sent. A broken template fails the package's tests and ships in the binary.
- `personality/builtin/**`, 8 `.md` files, are embedded (`personality/voice.go:16`, `//go:embed builtin`): gobble's
  personalities and messages. `prompt`'s golden system-prompt tests are built from them
  (`prompt/personality_test.go:43-65`).
- `prompt/testdata/**` (5 files) and `skill/testdata/**` (6 files) are fixtures that `prompt`'s and `skill`'s tests
  read (`prompt/sections_test.go:47`, `prompt/template_test.go:271`, `skill/skill_test.go:43`).

  go-selfupdate-lib's `**.md` filter would therefore skip the tests, the race run and the rehearsal for a change to
  what gobble ships. It is not usable here as written.

**The other 55 are read by nothing but people and the two checks:** `docs/**` (records, the index, architecture and
the guides), the root `README.md` and `AGENTS.md`, and the per-agent pointers `.claude/rules/*.md`,
`.grok/rules/*.md` and `.opencode/rules.md` (the docs set). The evidence:

- `//go:embed` cannot name a file above its package's directory, there is no `.go` file at the repository root, and
  `docs/` holds no Go package, so none of these can be embedded.
- No Go file names a path under `docs/` or reads the repository's own `README.md` or `AGENTS.md`. The tests that load
  context files (`AGENTS.md`, `CLAUDE.md`) write them into a temporary directory or copy `testdata/`; the one test that
  walks the repository, `TestNoXAIMethods`, reads `.go` files only (`acpserver/commands_test.go:263-287`).
- No script under `scripts/` reads a guide or the README; `check_records.py` reads the records and docs to check them.

**GitHub's path filters** (workflow syntax reference, read 2026-10-20):

- "When all the path names match patterns in `paths-ignore`, the workflow will not run." A push that changes one file
  outside the patterns runs the workflow.
- Path filters are not evaluated for pushes of tags: a `v*` tag keeps the full run, the release included.
- `*` "Matches zero or more characters, but does not match the `/` character"; `**` "Matches zero or more of any
  character". So `'*.md'` "Matches all `.md` files at the root of the repository", by the cheat sheet's `'*.js'` row,
  and `docs/**/*.md` is "A file with a `.md` suffix anywhere in the `docs` directory."
- "You cannot use both the `paths` and `paths-ignore` filters for the same event in a workflow."
- "If a workflow is skipped due to path filtering, branch filtering, or a commit message, then checks associated with
  that workflow will remain in a 'Pending' state." A pull request that requires such a check is then blocked.

**Repository settings** (read-only `gh api` GETs):

- The repository is public.
- `main` is not protected (`branches/main/protection` answers 404, "Branch not protected"), and
  `rules/branches/main` returns no rule.
- One ruleset exists, "release tags: owner only" (`25481802`): target `tag`, `refs/tags/v*`, enforcement `active`,
  rules `creation`, `update`, `deletion` and `non_fast_forward`, the repository admin role bypassing. It has no
  required status check. No check is required anywhere today.

**Minutes and time:**

- Run `39461740658`, the push of `d76e3d2` (code): 11 jobs, 10 run and `release` skipped; 2 macOS, 3 Windows and
  5 Ubuntu job runs, every one 0 billable ms by the timing API; 16 min 31 s from start to finish.
- Run `39441632748`, the push of `9188f63`, which changed `docs/README.md` and
  `docs/decisions/0004-PLAN-go-module-architecture.md` only: 11 jobs, 10 run, 0 billable ms; 13 min 30 s.
- This repository's CI therefore bills no minutes. The reasons left are turnaround (a docs-only push waits on a macOS
  and a Windows runner and the release rehearsal, about 13–17 minutes, for two checks that take seconds), less exposure
  to runner queues, and fewer runs. These are the reasons go-selfupdate-lib's owner went ahead for.

## Decision Drivers

* A docs-only change keeps its two checks, `check-records` and `markdownlint`, run as CI runs them.
* A change to anything Go reads, Markdown included, keeps the full run; so does every tag.
* A pattern that is too narrow costs only a full run; a pattern that is too wide skips the tests silently. The filter
  names what is safe to skip, rather than what is not.
* GitHub's own filters, no third-party action, and nothing that makes a skipped job read as a pass.
* Simple enough that a later change cannot quietly turn CI off for code.

## Considered Options

* **A.** `paths-ignore` on `ci.yml`'s `push` and `pull_request` triggers, naming the docs set, and nothing else.
* **B.** As A, plus a new `.github/workflows/docs.yml` whose `paths` name the same docs set (with the markdownlint
  config and itself), with one Linux job running `make check-records` and `make markdownlint`.
* **C.** One workflow: a first job decides from `git diff --name-only` whether anything outside the docs set changed,
  and every heavy job carries an `if:` on its output; a docs job always runs.
* **D.** `[skip ci]` in the message of a docs-only commit.

The docs set, in GitHub's filter syntax, for A and B:

```yaml
- '*.md'                  # the root README.md and AGENTS.md
- 'docs/**/*.md'          # records, the index, architecture, the guides
- '.claude/rules/*.md'
- '.grok/rules/*.md'
- '.opencode/rules.md'
```

go-selfupdate-lib's `'**.md'` is not among the options, for the reason in the Context: it would match the 36 files Go
reads.

## Decision Outcome

Chosen option: "B", because it keeps a docs-only change's two checks at the cost of one small workflow, uses GitHub's
own filters, leaves `ci.yml`'s jobs untouched, and never filters a tag. *Proposed: the owner decides at review.*
*(2026-10-20: accepted by the owner as recommended, option B, answering "gobble-cli 0026" to "Which repos' pairs do
you approve as recommended (MADR accepted, PLAN in-progress, then Phase 0/records and Phase 1 per its plan)?".)*

- **D1. The filter is the docs set, not `**.md`.** `ci.yml`'s `push` and `pull_request` triggers gain `paths-ignore`
  with the five patterns above. A Markdown file anywhere else (an embedded description, a personality, a test fixture)
  is outside them, so a change to it runs the full CI. A new kind of documentation outside the five patterns runs the
  full CI until a record adds it.
- **D2. `docs.yml`** runs on `push` to `main` and on `pull_request`, with `paths` naming the same five patterns plus
  `.markdownlint-cli2.jsonc` and `.github/workflows/docs.yml`. One `ubuntu-24.04` job checks out the tree and runs
  `make check-records` and `make markdownlint`, the targets `make preflight` runs, so the commands live in the
  Makefile once. It has `permissions: contents: read` and its own concurrency group, as `ci.yml` has.
- **D3. Tags and the release are unchanged.** `docs.yml` has no tag trigger; a `v*` tag runs `ci.yml` in full, and
  `release` still needs `build`, which needs `validate` and `sbom`. A docs-only pre-tag commit on `main` runs only
  `docs.yml`, and the tag's full run is the code proof; the code is the same as at the last commit whose full run
  passed, since a commit in the docs set changes nothing Go reads.
- **D4. The rehearsal of every push** (0004-PLAN, note of 2026-10-07) becomes the rehearsal of every push that
  changes something outside the docs set. 0004-PLAN gets a dated note saying so.
- **D5. No required check names a `ci.yml` job** while this split stands, since a docs-only pull request would leave
  it "Pending". None exists today. A record that adds a required check must say which workflow it belongs to.

### Consequences

* Good, because a docs-only push or pull request runs one Linux job of about a minute (go-selfupdate-lib's ran in
  12 s), with no macOS or Windows runner and no rehearsal to wait for.
* Good, because a change to anything gobble embeds or its tests read, and every tag, run as before.
* Good, because a docs-only push to `main` no longer starts a `ci.yml` run, so it no longer cancels a code push's run
  still in progress in the same concurrency group.
* Neutral, because a push that changes code and documentation runs both workflows, so the two checks run twice.
* Neutral, because `docs.yml` runs the two checks on Linux only, where `make preflight` runs them on three OSes; the
  checks read text, and `.gitattributes` keeps every text file LF on every checkout.
* Neutral, because `main`'s head can be a docs-only commit with no `ci.yml` run of its own; the CI result that covers
  its code is the last code commit's.
* Bad, because the five patterns are written twice, in `ci.yml`'s two triggers and in `docs.yml`, and must be kept in
  step; each file's comment names the other and this record. A mismatch toward too few patterns in `ci.yml` costs a
  full run; too many in `ci.yml`, or too few in `docs.yml`, leaves a change unchecked.
* Bad, because `docs.yml` is one more workflow for actionlint, and its checkout pin must follow `ci.yml`'s by hand:
  this repository has no pin checker.

### Confirmation

* actionlint passes on both workflows; `ci.yml`'s Linux leg runs it on every workflow in the directory.
* The two checks fail on a scratch clone with a planted broken link and a planted markdownlint error, and pass on the
  tree.
* On a scratch clone with every file of the docs set deleted, `go build ./...` and `go test -count=1 ./...` pass: no
  code reads the docs set. On a scratch clone with a broken template in one `tool/builtin/describe/*.md`, `go test` of
  `./tool/builtin/` fails: that file is rightly outside the filter.
* After the change lands, a docs-only push to `main` shows a "Docs" run and no "CI" run, and a push that changes a
  non-Markdown file shows "CI". The PLAN records both run IDs.
* The next release's tag push runs the full CI, recorded in that release's record.

## Pros and Cons of the Options

### A. `paths-ignore` only

* Good, because it is one list per trigger.
* Bad, because a docs-only commit then gets neither the records check nor markdownlint, and a broken link reaches
  `main` unchecked until the next code push.

### B. `paths-ignore` and a docs workflow

* Good, because the docs keep their two checks, run by the same Makefile targets, and `ci.yml` keeps its jobs.
* Good, because both filters are GitHub's own, documented behaviour, and a tag is never filtered.
* Bad, because two workflows hold the same pattern list, to be kept in step.

### C. One workflow, gated jobs

* Good, because one file holds it all, and the pattern list once.
* Bad, because `validate`, `sbom`, `build` and `release` each gain an `if:`, and `build` and `release` are calls to
  go-selfupdate-lib's reusable workflows whose `needs` chain a wrong condition could skip, with nothing to show it; a
  skipped job reads as neither pass nor fail.
* Bad, because the change-detection job runs on every push, and needs a full-history or two-commit checkout to diff.

### D. `[skip ci]`

* Bad, because the global `prepare-commit-msg` hook writes the commit messages, so a person would have to add it each
  time; it also skips the two checks.

## More Information

* The PLAN: [0026-PLAN-skip-full-ci-for-docs-only-pushes.md](0026-PLAN-skip-full-ci-for-docs-only-pushes.md).
* The model followed: go-selfupdate-lib's `docs/decisions/0028-MADR-skip-full-ci-for-docs-only-pushes.md` and
  `docs/decisions/0028-PLAN-skip-full-ci-for-docs-only-pushes.md`, accepted and complete on 2026-10-20. The facts
  taken from it are restated above: its `**.md` filter, its one-job docs workflow, its 12 s docs-only run, and its
  CI's 0 billable minutes.
* The CI being changed is described in `docs/architecture.md` ("Checks") and `docs/guides/developing.md` ("CI and the
  release rehearsal").
