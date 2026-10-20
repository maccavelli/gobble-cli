---
status: in-progress
date: 2026-10-20
associated-madr: "0026-MADR-skip-full-ci-for-docs-only-pushes.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement skipping the full CI for docs-only pushes, with a docs workflow

Associated MADR: [0026-MADR-skip-full-ci-for-docs-only-pushes.md](0026-MADR-skip-full-ci-for-docs-only-pushes.md)
(option B, D1–D5).

*Proposed. Nothing in it runs until the owner approves it and chooses the MADR's option. It starts on `main` at or
after `bdd8268`.* *(2026-10-20: approved by the owner; see "Approval (2026-10-20)" in the execution record.)*

## Goal

A push or pull request whose every changed file is in the docs set (the MADR's five patterns) runs one Linux job,
`make check-records` and `make markdownlint`, instead of `ci.yml`'s eleven. Any other change, a Markdown file Go reads
included, and every tag, runs `ci.yml` as before.

## Scope

### In scope

| Phase | What | Commit |
| :--- | :--- | :--- |
| P0 | this pair and its `docs/README.md` rows | records only |
| P1 | `ci.yml`'s two filters, the new `docs.yml`, and the seen-failing runs | CI |
| P2 | the documents: `docs/architecture.md`, `docs/guides/developing.md`, a dated note in 0004-PLAN | docs |
| P3 | the live checks, and closing | records |

### Out of scope

- Any change to `ci.yml`'s jobs or steps, or to the Makefile's targets.
- go-selfupdate-lib's build and publish workflows, which `ci.yml` calls.
- Required status checks: none exist (MADR, "Repository settings"; D5).
- `AGENTS.md`: it does not describe CI, so it is unchanged.
- No release follows: the change is CI and documents only.

## Rules

- AGENTS.md's gate and records rules; actionlint on each changed workflow.
- A new check is seen failing before it is relied on, on a scratch clone outside the repository: this repository has
  no `scripts/plant-copy.sh`. The clone is `git clone` of the repository into the session's scratch directory, so it
  holds the committed tree only and the working tree is never touched. Its paths are written `<scratch>/…` in this
  record.
- Each phase is one commit on the owner's word; each push on the owner's word.

## Implementation Steps

### P0 — records

This pair, `status: proposed`, and `docs/README.md`'s two rows. Checks: `make check-records`, `make markdownlint`.

### P1 — the filters and the docs workflow

1. **`ci.yml`:**
   - `on.push` gains, beside `branches` and `tags`:

     ```yaml
     paths-ignore:
       - '*.md'
       - 'docs/**/*.md'
       - '.claude/rules/*.md'
       - '.grok/rules/*.md'
       - '.opencode/rules.md'
     ```

   - `on.pull_request:` (today a bare key, `ci.yml:6`) gains the same list.
   - A comment above `on:` cites this pair, says a tag push is never filtered, says a Markdown file outside these
     patterns (an embedded description, a personality, a fixture) runs the full CI on purpose, and names `docs.yml` as
     the file whose `paths` must list the same five patterns.
2. **`.github/workflows/docs.yml`,** new:

   ```yaml
   name: Docs
   on:
     push:
       branches: [main]
       paths: &docs
         - '*.md'
         - 'docs/**/*.md'
         - '.claude/rules/*.md'
         - '.grok/rules/*.md'
         - '.opencode/rules.md'
         - '.markdownlint-cli2.jsonc'
         - '.github/workflows/docs.yml'
     pull_request:
       paths: *docs
   permissions:
     contents: read
   concurrency:
     group: docs-${{ github.ref }}
     cancel-in-progress: true
   jobs:
     docs:
       runs-on: ubuntu-24.04
       timeout-minutes: 10
       steps:
         - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
           with:
             persist-credentials: false
         - name: make check-records
           run: make check-records
         - name: make markdownlint
           run: make markdownlint
   ```

   - The two steps are the targets `make preflight` runs for these checks (`Makefile:188-189`), unchanged: they run
     `python3 scripts/check_records.py --check-all` (`Makefile:220-221`, `PYTHON` defaulting to `python3`,
     `Makefile:80`) and `npx --yes markdownlint-cli2@0.23.2` (`Makefile:229-230`). Like `ci.yml`, the job uses the
     runner image's Python and Node, with no setup action, and needs no Go.
   - The checkout is pinned to `ci.yml`'s SHA (`ci.yml:36`).
   - If actionlint refuses the YAML anchor, the `pull_request` list is written out in full; that is a formatting
     choice, not a deviation.
   - A header comment says what the workflow is for, cites this pair, and names `ci.yml`'s `paths-ignore` as the list
     to keep in step.
3. **Seen failing,** on scratch clones of the P1 commit's tree (`<scratch>/clone-*`), each with its output kept in
   the scratch directory:
   - the tree: `make check-records` and `make markdownlint` exit 0;
   - a broken relative link planted in `docs/guides/developing.md`: `make check-records` exits 1 with
     `broken relative link: docs/guides/developing.md:<line>: …`, and markdownlint still exits 0;
   - `##CI` planted as a heading in `README.md`: `make markdownlint` exits 1 with an `MD018/no-missing-space-atx`
     error, and `check-records` still exits 0;
   - **the docs set is unread:** every tracked file matching the five patterns deleted (55 files at `bdd8268`):
     `go build ./...` and `go test -count=1 ./...` exit 0;
   - **the rest is read:** `{{` planted unclosed in `tool/builtin/describe/read.md`: `go test -count=1
     ./tool/builtin/` exits 1, so that file must stay outside the filter.

   Each failure is quoted in the execution record.
4. **The pattern list,** checked by a scratch script (stdlib Python, GitHub's `*` and `**` rules): over `git ls-files
   '*.md'`, the five patterns match the 55 docs-set files and none of the 36 under `personality/builtin/`,
   `tool/builtin/describe/`, `prompt/testdata/` and `skill/testdata/`; and the two workflows' lists are the same five,
   read from the YAML.
5. **Checks:** actionlint on both workflows (`go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`, as
   `ci.yml:102` runs it); `make check-records`; `make markdownlint`. The filters themselves are proven only on GitHub,
   in P3.

### P2 — the documents

1. **`docs/architecture.md`:**
   - "Checks", the `ci.yml` bullet (`architecture.md:174-176`): a push or pull request whose changes are all in the
     docs set (named) skips `ci.yml`, and `.github/workflows/docs.yml` runs `make check-records` and `make markdownlint`
     on one Linux runner instead; a Markdown file Go reads runs the full CI; a tag is never filtered.
   - "The release" sub-bullets: the rehearsal runs on every push outside the docs set.
   - "Tree": a `.github/workflows/docs.yml` line under `ci.yml`'s.
2. **`docs/guides/developing.md`,** "CI and the release rehearsal" (`developing.md:163-172`): the same split, in one
   sentence after the opening line, and `build`'s bullet no longer saying the rehearsal runs on every push.
3. **[0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md):** a dated note after the note of
   2026-10-07 under Phase 4 that names "the CI rehearsal of every push": a push whose changes are all in the docs set
   no longer rehearses (0026-MADR D4). The note of 2026-10-07 is not rewritten.
4. **`docs/README.md`:** this pair's rows, and the 0004-PLAN row's status gains the note.
5. **Checks:** `make check-records`, `make markdownlint`, and an identifier scan of the changed files for the local
   account name and the hostname domain. This commit changes Markdown only, all in the docs set, so its push is P3's
   docs-only test.

### P3 — live checks and closing

1. After P1 and P2 are pushed, on the owner's word:
   - P1's push changes workflow files, so it starts both "CI" and "Docs"; both run IDs and results are recorded.
   - P2's push changes only the docs set: GitHub's run list for that commit (`actions/runs?head_sha=…`) holds one run,
     "Docs", and no "CI" run.
2. Optional, on the owner's word: a pull request that changes only `docs/README.md` shows "Docs" and no "CI".
3. The next release's tag shows a full "CI" run; that release's record notes it, and this plan names it as left to
   that release.
4. `## Verification, at closing`, `status: complete`, and the index row.

## Verification

- **V1 (filters):** `ci.yml` has the five `paths-ignore` patterns on `push` and `pull_request`; `docs.yml` exists
  with the same five in `paths`, plus its two own files, and the two Make targets; actionlint passes on both.
- **V2 (seen failing):** the broken link fails only `check-records`, the MD018 error fails only `markdownlint`, and
  the template plant fails `go test ./tool/builtin/`; the tree passes all of them.
- **V3 (the docs set is unread):** with the 55 docs-set files deleted, `go build ./...` and `go test -count=1 ./...`
  pass.
- **V4 (patterns):** the classification of step 4 of P1: 55 matched, 36 not, the two lists equal.
- **V5 (live):** a docs-only push ran only "Docs"; a push with a non-Markdown change ran "CI" (P3).
- **V6 (tags):** the next tag runs the full CI, left to that release.
- **V7 (docs):** `architecture.md`, `developing.md` and 0004-PLAN say so; `check-records` and `markdownlint` clean;
  the identifier scan finds nothing.

## Rollout and Rollback

- **Rollout:** P1 and P2 on `main`, pushed on the owner's word.
- **Rollback:** `git revert` of P1 restores the full CI for every push and removes `docs.yml`; P2's sentences are then
  reverted in a docs commit. Nothing outside the repository depends on the split, and no required check names either
  workflow.

## Execution record

~~*None yet: the pair is proposed.*~~

### Approval (2026-10-20)

The owner approved this pair as recommended, answering "gobble-cli 0026" to "Which repos' pairs do you approve as
recommended (MADR accepted, PLAN in-progress, then Phase 0/records and Phase 1 per its plan)?". The MADR became
`accepted`, with option B chosen, and this plan `in-progress`. The approval covers P0, P1 and P2 as written; P3 needs
the pushes, each on the owner's word.

### P0 (2026-10-20)

This pair, and `docs/README.md`'s two rows for it: the MADR `accepted` with the owner's words in its Decision Outcome,
this plan `in-progress` with the approval above, and the rows' statuses to match (the step's "`status: proposed`" was
written before the approval, which supersedes it). Run in the working tree, on
`main` at `bdd8268`; both checks read the documentation only.

- `make check-records`: exit 0, no finding.
- `make markdownlint`: exit 0, `Linting: 55 files` / `Summary: 0 issues in 0 files`.

No deviation.

### P1 (2026-10-20)

On `main` at `bdd8268`, with P0's records in the working tree.

1. **`ci.yml`:** `on.push` gains `paths-ignore` with the five patterns, beside `branches` and `tags`, and
   `on.pull_request` (a bare key until now) the same list. A comment above `on:` cites this pair, says a tag push is
   never filtered, says a Markdown file outside the patterns (an embedded tool description, a personality, a test
   fixture) runs the full CI on purpose, and names `docs.yml` as the file whose `paths` must list the same five. No job
   or step changed.
2. **`.github/workflows/docs.yml`,** new, as step 2 has it: `push` to `main` with `paths: &docs` (the five patterns,
   `.markdownlint-cli2.jsonc` and `.github/workflows/docs.yml`), `pull_request` with `paths: *docs`; top-level
   `permissions: contents: read`; concurrency group `docs-${{ github.ref }}` with `cancel-in-progress`; one
   `ubuntu-24.04` job, `timeout-minutes: 10`, the checkout pinned to `ci.yml`'s SHA with `persist-credentials: false`,
   then `make check-records` and `make markdownlint`. actionlint accepts the anchor, so the list is written once. A
   header comment says what the workflow is for, cites this pair, and names `ci.yml`'s `paths-ignore` as the list to
   keep in step.
3. **Seen failing.** Each clone is `git clone` of the repository at `bdd8268` into `<scratch>/clone-*`, with this
   change's five files (`ci.yml`, `docs.yml`, the pair, `docs/README.md`) copied in, and one plant; all output was
   kept in the scratch directory. `make` reports a failed recipe as exit 2 with `Error 1`, the recipe's own exit
   status; the step's "exits 1" means the recipe's.

   | Clone | `make check-records` | `make markdownlint` |
   | :--- | :--- | :--- |
   | the tree (`clone-tree`) | 0 | 0, `Linting: 55 files` / `Summary: 0 issues in 0 files` |
   | `See [nothing](no-such-file.md).` appended to `docs/guides/developing.md` | **2**, `broken relative link: docs/guides/developing.md:183: no-such-file.md` / `1 broken relative link(s)` / `make: *** [check-records] Error 1` | 0 |
   | `##CI` appended to `README.md` | 0 | **2**, `README.md:137:1 error MD018/no-missing-space-atx No space after hash on atx style heading [Context: "##CI"]` / `make: *** [markdownlint] Error 1` |

   Each plant fails only its own check.

   - **The docs set is unread.** In `clone-nodocs`, every file matching the five patterns was deleted: the 55 tracked
     at `bdd8268` and this change's two records, 57 in all, among them `README.md`, `AGENTS.md`, the three agent
     pointers and every file under `docs/`. `go build ./...` exited 0. The first `go test -count=1 ./...` exited 1, one
     test failing:

     ```text
     --- FAIL: TestLastGoodOnDisk (0.01s)
         choice_test.go:211: the listing was handed [] at 0001-01-01 00:00:00 +0000 UTC, want the kept usable ids at 2026-10-20 13:04:14 -0500 CDT
     FAIL	github.com/maccavelli/gobble-cli/internal/cli	11.880s
     ```

     It is pre-existing and unrelated to the deletion: the test reads what its stub lister was handed without waiting
     for the background listing to call it (`internal/cli/choice_test.go:187-212` at `bdd8268`), and in `clone-tree`,
     with every document present, `go test -count=100 -cpu=1,2 -run 'TestLastGoodOnDisk$' ./internal/cli/` failed 101
     of 200 runs with the same line. It is not this plan's to fix. Run again, `go test -count=1 ./...` exited 0 in
     `clone-nodocs`, and in `clone-tree` at the same time; that pass is V3's evidence.
   - **The rest is read.** In `clone-template`, `{{` appended unclosed to `tool/builtin/describe/read.md`:
     `go test -count=1 ./tool/builtin/` exited 1, and 0 in `clone-tree`:

     ```text
     --- FAIL: TestToolsAndAnnotations (0.00s)
     panic: builtin: read description: template: read.md:23: unclosed action started at read.md:22 [recovered, repanicked]
     FAIL	github.com/maccavelli/gobble-cli/tool/builtin	0.679s
     ```

4. **The pattern list,** by a scratch stdlib-Python script over `git ls-files '*.md'` in `clone-tree`, reading both
   workflows' lists from their YAML text (the alias resolved to its anchor):

   ```text
   tracked .md: 91; docs set: 55; outside: 36
   untracked .md in the clone (this change's records): ['docs/decisions/0026-MADR-skip-full-ci-for-docs-only-pushes.md', 'docs/decisions/0026-PLAN-skip-full-ci-for-docs-only-pushes.md'], matched: [True, True]
   outside the docs set but not under the four Go-read trees: []
     personality/builtin/: 8 outside, 0 matched
     tool/builtin/describe/: 17 outside, 0 matched
     prompt/testdata/: 5 outside, 0 matched
     skill/testdata/: 6 outside, 0 matched
   non-.md tracked files matched: []
   lists equal: True
   ```

   The script's first version translated `**` as "any characters" alone, so `docs/**/*.md` needed a directory under
   `docs/` and missed `docs/README.md` and `docs/architecture.md` (53 matched, 38 outside, exit 1). GitHub's cheat sheet
   row for `docs/**/*.md` lists `docs/README.md` among its matches (read 2026-10-20), so `**/` matches zero or more
   directories; the script was corrected to that, and the clone setup with it. The workflows did not change. With one
   pattern (`.opencode/rules.md`) removed from a copy of `docs.yml`, the script exits 1, `lists equal: False`.
5. **Checks:**
   - actionlint, `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12` in `clone-tree`, as `ci.yml:123` runs it
     (`ci.yml:102` before this change), with shellcheck 0.11.0 on `PATH`: exit 0, `Linting 2 files`, `Found total 0
     errors` for `.github/workflows/docs.yml` and for `.github/workflows/ci.yml`. On a copy with `timeout-minutes`
     misspelt in `docs.yml` it exits 1: `.github/workflows/docs.yml:39:5: unexpected key "timeout-minute" for "job"
     section`.
   - `make check-records` and `make markdownlint`: the tree row above.
   - `make preflight` in `clone-tree`, run there because the working tree holds other uncommitted work: exit 0, every
     gate from `==> gofmt` to `==> markdownlint` and `preflight passed`; govulncheck `No vulnerabilities found.`; the
     clone's `git status` afterwards shows only this change's five files.
   - An identifier scan of the changed files for the local account name, the host name and local absolute paths:
     none.

   The change has no Go file, so `make pre-add-check` has nothing of its own to check. The filters themselves are
   proven only on GitHub, in P3.

No deviation from the step as written.

### P2 — stopped before its first edit (2026-10-20), waiting on the owner

P2 step 3 places the 0004-PLAN note "after the note of 2026-10-07 under Phase 4 that names 'the CI rehearsal of every
push'", and the MADR's Context cites that phrase to the same place ("note of 2026-10-07 under Phase 4"). No note is
both. In [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md) at `bdd8268`:

- the note of 2026-10-07 under Phase 4 (line 395) ends "the build workflow's rehearsal on every push and pull request
  replaces them";
- the phrase "the CI rehearsal of every push" is in the note of 2026-10-07 under `## Verification` (line 425):
  "Its two lines are replaced by the CI rehearsal of every push, recorded under Phase 4 steps 2–4";
- the amendment "Phase 4 steps 2–4 made executable" of 2026-10-07 says "Until then every push and pull request
  rehearses" (line 1007).

The substance of the MADR's fact holds (0004-PLAN's notes of 2026-10-07 say the rehearsal runs on every push); its
location does not, and the step cannot be placed without choosing between the notes. By the approval's terms, a
contradiction of a fact the MADR asserts stops the work for the owner, so nothing of P2 is written: not
`docs/architecture.md`, `docs/guides/developing.md`, 0004-PLAN, nor the 0004-PLAN row in `docs/README.md`. The
resolutions offered, each recorded here and in a dated note on the MADR's Context before P2 resumes:

1. *(recommended)* One dated note after the Phase 4 note (line 395), saying a push whose changes are all in the docs
   set no longer rehearses (0026-MADR D4), and that this covers the Verification note of 2026-10-07 and the
   amendment's line too; the MADR's citation annotated with the phrase's real place. Cost: one sentence in the MADR, one
   in this plan.
2. A dated note after each of the two notes of 2026-10-07 (lines 395 and 425), with the same MADR annotation. Cost: two
   notes saying one thing.
3. One note after the Verification note (line 425), where the quoted phrase is, with the same MADR annotation. Leaves
   the Phase 4 note, which says the same, without one.

Doing nothing leaves P2 undone: once P1 is pushed, `docs/architecture.md`, `docs/guides/developing.md` and 0004-PLAN
say the rehearsal runs on every push, which is no longer true, and P3 has no docs-only commit to push.
