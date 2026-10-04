---
status: completed
date: 2026-10-04
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# PLAN 0007 — Bring gobble-cli up to the fleet repository scaffold

Implements [0007-MADR-repository-scaffolding-to-fleet-standard.md](0007-MADR-repository-scaffolding-to-fleet-standard.md) decisions D1–D8, closing findings F1–F10 and F12. D9 (what is not added) is honoured by C2. F11 and F13 are honoured by C2 and by the Stability rule.

## Goal

Every item below is an observable state of the branch when this plan is done:

1. `make preflight` exits 0 in Git Bash on Windows. It also exits 0 in WSL Ubuntu 24.04 on a copy of the same working tree. Today it cannot run at all on Linux (F4).
2. `make preflight` runs two new gates: `check-records` (`python3 scripts/check_records.py --check-all`) and `markdownlint` (`npx --yes markdownlint-cli2@0.23.2`). Both exit 0 on the tree. Both have been seen to fail on a planted defect.
3. `python3 scripts/check_records.py --next` prints `0008`.
4. These files exist, and each is `cmp`-identical to its go-selfupdate-lib source at `a65e12b8afa0`:
   - `.claude/.gitignore`
   - `.claude/rules/madr-and-plan-skill.md`
   - `.grok/rules/madr-plan-before-mutating-work.md`
   - `.opencode/rules.md`
   - `opencode.json`
   - `.markdownlint-cli2.jsonc`
5. `AGENTS.md` exists with the text in P4.
6. Every tracked file reports `attr/text=auto eol=lf` in `git ls-files --eol`.
7. `make lint` reports `unused`, `whitespace`, `gofmt` and `goimports` on their plants, and 0 issues on the tree.
8. 0004-PLAN carries a dated amendment for D8, and its struck lines keep their original text.
9. `docs/README.md` indexes 0007-MADR and 0007-PLAN with their final status, and has no broken link.

## Scope

### In scope (the only files any phase may touch)

New files:

- `AGENTS.md`
- `.claude/.gitignore`
- `.claude/rules/madr-and-plan-skill.md`
- `.grok/rules/madr-plan-before-mutating-work.md`
- `.opencode/rules.md`
- `opencode.json`
- `.markdownlint-cli2.jsonc`
- `.gitattributes`
- `scripts/check_records.py`

Changed files:

- `.gitignore`
- `Makefile`
- `.golangci.yml`
- `README.md`: markdownlint fixes only.
- `docs/README.md`
- `docs/architecture.md`: markdownlint fixes, plus the Tree block.
- `docs/reports/0001-REPORT-go-port-feasibility.md`: formatting only.
- `docs/decisions/0004-PLAN-go-module-architecture.md`: the D8 amendment only.
- `docs/decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md`: status and the Observed section only.
- `docs/decisions/0007-PLAN-repository-scaffolding-to-fleet-standard.md`: this file, for status and the Execution record.

### Out of scope

- **`.github/workflows/ci.yml` and the release pipeline.** These belong to 0004-PLAN Phase 4. This plan amends that phase in P6 and creates no workflow.
- **Any Go source file.** The planted Go files in P5 live only in scratch copies.
- **The rest of `docs/architecture.md`.** Its "What is not here" section is stale (it says no package map exists). 0004-PLAN Phase 5 owns the full rewrite, and this plan touches only the lines named in P3.
- **Every sibling repository.** They are read only.
- **`go.mod` and `go.sum`.** No requirement changes. The fleet tools are invoked as they are today.

## Stability rule

**Branch first.** The organisation's Git gate (ADR 011, skill `enforcing-git-branch-gates`) forbids commits on `main`. P0 creates ~~`feature/<JIRA-KEY>-0007-scaffolding`~~ `feature/0007-scaffolding` before any file is written. ~~The owner supplies the Jira key, and none is invented.~~ The project is exempt from the Jira key (owner, 2026-10-04). If the owner declines a `feature/` branch, execution stops before P1.

**Every phase ends with these commands, in this order, and their exit statuses are captured before any filter.**

1. Gates in Git Bash on Windows:

   ```text
   LOG="$SCRATCH/0007-pN-win.log"; make preflight > "$LOG" 2>&1; GATE=$?
   ```

   Expected: `GATE=0` and the last line `preflight passed`. Before P2, preflight runs on Windows only.
2. From P2 on, the same gates in WSL, on a copy of the working tree:

   ```text
   wsl -d Ubuntu-24.04 --exec bash -c 'rm -rf ~/scratch/gobble-0007 && cp -a /mnt/c/<path-to>/gobble-cli ~/scratch/gobble-0007 && cd ~/scratch/gobble-0007 && make preflight' > "$SCRATCH/0007-pN-wsl.log" 2>&1; WGATE=$?
   ```

   Expected: `WGATE=0`. Run it with `MSYS_NO_PATHCONV=1` from Git Bash, because MSYS otherwise rewrites the `/mnt/c` path (MADR measurement 3).
3. `git diff --check` is clean.
4. `git status --porcelain` lists only files in **In scope**.

**Commits.**
- ~~One commit per phase, made only on the owner's explicit "commit" for that phase. An approval that says "commit each phase" counts.~~ *(Deprecated 2026-10-04 by the owner's P0 answer.)*
- Staged Go files: none, since no phase changes Go source.
- ~~The commit-message mechanism follows the owner's answer to the precondition in P0 (see that phase).~~ *(Deprecated 2026-10-04 by the owner's P0 answer.)*
- **2026-10-04, owner's answer:** "you will add and stage, i will commit and push." The agent runs `git add` at the end of each phase and never runs `git commit`. The owner decides how the staged phases are committed, and the commit message comes from the owner's own workflow. A phase's "commit" in this plan therefore means "staged and handed back".

**Pushes and tags.** `git push`, tags and merge requests are not permitted unless the owner asks in the same turn.

**On a failure.** If a gate fails for a reason this plan does not fix, that is a deviation. Stop and prompt with evidence and resolutions, per the owner's deviation rule. Do not widen scope, and do not weaken a gate. The most likely case is a Linux-only failure in an existing preflight step that F4 has kept from ever running.

## Cross-cutting contracts

- **C1. Copied files stay copied.** Each file named in Goal item 4 is `cmp`-identical to its go-selfupdate-lib source at `a65e12b8afa0`. Its last change there is `e94750a`. `scripts/check_records.py` differs from magic-cli-remote `9778cbc17b39` (last changed there at `2f6cb5c5`) only by the deletions listed in P3.
- **C2. Nothing outside D1–D8 is added.** D9's list stays out: Dependabot, `.claude/settings.json`, `.kilo/`, depguard, `.editorconfig`, `SECURITY.md`, `CODEOWNERS`, `CONTRIBUTING.md`, `CHANGELOG.md`, and a local git identity.
- **C3. No gate is weakened to pass.**
  - Markdown issues are fixed in the Markdown. No rule is disabled or excluded.
  - No `//nolint`, linter exclusion or `-text` attribute is added to make a phase green.
  - The records check is not taught to ignore a link.

  **This is the contract most at risk.** The 17 MD004 issues sit in a historical REPORT, and a one-line exclusion of `docs/reports/**` in the markdownlint config would clear them faster than editing the REPORT. That exclusion would break C1, since the config would no longer be the fleet's, and C3.
- **C4. Formatting edits change no words.** P3's script compares each edited Markdown file's normalised text before and after, and the edit is accepted only when they are equal.
- **C5. Every new gate is seen to fail before it is trusted.** The plant always goes in a scratch copy, never in the working tree.
- **C6. No identifiers.** No added or changed file carries a hostname, account name, internal domain or real-machine path. Commands in this plan write `<path-to>` and `<user>`.

## Dependency and delivery order

P0 → P1 → P2 → P3 → P4 → P5 → P6. ~~Each phase is one commit.~~ Each phase ends staged; the owner commits.

- **P1** (line endings and ignores) comes first: P3 brings a Python script, and its `__pycache__/` must already be ignored.
- **P2** (portable Makefile) comes before every phase that verifies in WSL.
- **P3** (docs gates) comes before **P4** (agent files), so that `AGENTS.md` is linted by preflight from the moment it exists.
- **P5** (linters) is independent of P3 and P4. It follows them so that each phase changes one gate.
- **P6** writes the records last, so that they describe what ran.

Relation to 0004-PLAN:
- P1 lands before 0004-PLAN Phase 3 adds a golden file.
- P2 lands before 0004-PLAN Phase 4.
- If 0004-PLAN Phase 2 runs on `main` while this branch is open, rebasing is the owner's decision. This plan does not rebase on its own.

## Implementation Steps

Throughout, `$SCRATCH` is the session scratchpad. `<sibling>` is a sibling checkout next to gobble-cli. Verification scripts are written to `$SCRATCH` with the Write tool and run by path. None of them is committed.

### P0 — Branch and preconditions (no decision; closes F13)

1. Run `git branch --show-current`. Expected: `main`.
2. Ask the owner for:
   - **(a)** the Jira key for the branch name. Do not invent one.
   - **(b)** how commit messages are produced on this branch. The organisation's gate requires a subject of the form `<JIRA-KEY>: <summary> [agent]`. The owner's global rule says messages come only from the `prepare-commit-msg` hook, through `git commit --no-edit`. The two cannot both hold: a hook-written subject has no key, and supplying one means `-m`. Organisation instructions take precedence, so the default is the organisation form. The owner confirms the default or names another.
3. ~~Once (a) is answered, run `git switch -c feature/<JIRA-KEY>-0007-scaffolding`. Expected: `Switched to a new branch`. Then run `git branch --show-current`. Expected: that name.~~ *(Deprecated 2026-10-04.)*

   **2026-10-04, the owner's answers:**
   - (a) "this project is exempt from" the Jira requirement.
   - (b) "you will add and stage, i will commit and push."

   So the branch is `feature/0007-scaffolding`, with no key. Run `git switch -c feature/0007-scaffolding`. Expected: `Switched to a new branch 'feature/0007-scaffolding'`.

   The agent never commits (Stability rule, Commits). `AGENTS.md`'s Commits section keeps the fleet text from P4.
4. Record (a) and (b) in this PLAN's Execution record (P6).

**Verification:** `git branch --show-current` prints the `feature/` name. `git status --porcelain` shows only the untracked 0007 records and the `docs/README.md` index edit from writing them.

### P1 — Line endings and ignores (D3, D4; closes F3, F8)

1. **Create `.gitattributes`** with exactly this text. It is go-selfupdate-lib's file without its fixture exemption, with the provenance line pointed at this record:

   ```text
   # Keep every text file LF in the working tree on ALL platforms.
   #
   # Without this, a Windows checkout with core.autocrlf=true (the GitHub runner
   # default) rewrites LF to CRLF on the way out of git. CI runs on Windows, and
   # that silently breaks:
   #
   #   * byte-exact fixtures, such as golden transcripts under testdata/;
   #   * anything a POSIX shell executes, where a CR becomes part of the last
   #     argument on every line.
   #
   # eol=lf is deliberate rather than `text=auto` alone: auto would still hand
   # Windows CRLF. Nothing in this repository needs CRLF.
   # (docs/decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md D3)
   * text=auto eol=lf

   # Binary files must never be touched by EOL conversion.
   *.png   binary
   *.jpg   binary
   *.jpeg  binary
   *.gif   binary
   *.ico   binary
   *.webp  binary
   *.pdf   binary
   *.zip   binary
   *.gz    binary
   *.ttf   binary
   *.otf   binary
   *.woff  binary
   *.woff2 binary
   ```

2. **Append to `.gitignore`** after its last line (`.vscode/`), keeping every existing line:

   ```text

   # Python (scripts/check_records.py; 0007-MADR D4)
   __pycache__/
   *.py[cod]

   # Scratch and tool output (the fleet .gitignore)
   coverage_*.out
   *.log
   *.tmp
   tmp/
   temp/
   .cache/
   .worktrees/
   ```

3. **Renormalise in a scratch clone, never in the working repository's index.**
   - Run `git clone -q <path-to>/gobble-cli "$SCRATCH/renorm"`.
   - Copy the new `.gitattributes` into the clone.
   - In the clone, run `git add --renormalize .`, then `git status --porcelain`.
   - Expected: the only line is `A  .gitattributes`, because every tracked file is already LF (MADR measurement 6).

**Verification:**
- `git ls-files --eol | grep -v 'attr/text=auto eol=lf'` prints nothing. The exit status of `grep -v` is 1 here and is the expected one.
- `git check-ignore -q scripts/__pycache__/x.pyc` exits 0.
- `git ls-files -ci --exclude-standard` prints nothing, so no tracked file is newly ignored.
- **Negative test, in scratch clones only.**
  - `git -c core.autocrlf=true clone -q <path-to>/gobble-cli "$SCRATCH/crlf-without"` (no `.gitattributes`): a scratch script that reads `README.md` as bytes must report CRLF present. This proves the check can see the failure.
  - In `$SCRATCH/renorm` from step 3, commit `.gitattributes`. This is a scratch commit that is never pushed.
  - `git -c core.autocrlf=true clone -q "$SCRATCH/renorm" "$SCRATCH/crlf-with"`: the same script must report no CRLF.
  - Quote both outputs.
- Then run the Stability rule, Windows only.

### P2 — Portable Makefile (D5; closes F4)

1. **In `Makefile`, replace these five lines** (lines 6–10):

   ```make
   SHELL := C:/PROGRA~1/Git/usr/bin/bash.exe
   .SHELLFLAGS := -eu -o pipefail -c
   # Git bash does not put its coreutils on PATH when make starts it from
   # PowerShell. Recipes need grep, diff, cp, and mv.
   export PATH := C:/PROGRA~1/Git/usr/bin:$(PATH)
   ```

   with:

   ```make
   # Recipes need bash (for pipefail) and grep, diff, cp and mv. On Windows
   # that is Git's bash, and Git's usr/bin goes on PATH because Git bash does
   # not put its coreutils there when make starts it from PowerShell. Set
   # GIT_BASH if Git is installed elsewhere. Every other host uses /bin/bash
   # (0007-MADR D5).
   ifeq ($(OS),Windows_NT)
     GIT_BASH ?= C:/PROGRA~1/Git/usr/bin/bash.exe
     SHELL := $(GIT_BASH)
     export PATH := $(dir $(GIT_BASH)):$(PATH)
   else
     SHELL := /bin/bash
   endif
   .SHELLFLAGS := -eu -o pipefail -c
   ```

   Do the replacement with a scratch Python script. It asserts that the old block occurs exactly once before it writes, then asserts that `PROGRA~1` now appears only on the `GIT_BASH ?=` line.
2. **Replace the `lint` recipe:**

```make
lint:
	golangci-lint run ./...
```

   with:

```make
# golangci-lint v2.14.0 is the fleet pin (0007-MADR D5).
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found. Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0" >&2; \
		exit 2; \
	}
	golangci-lint run ./...
```

   In `preflight`, replace `@echo "==> golangci-lint"; golangci-lint run ./...` with `@echo "==> golangci-lint"; $(MAKE) --no-print-directory lint`. Recipe lines are tab-indented.

**Verification:**
- `make clean` exits 0 under each of Git Bash, `pwsh -NoProfile -Command "make clean"`, and the WSL copy.
- `make -p -n clean | grep '^SHELL :='` shows `C:/PROGRA~1/Git/usr/bin/bash.exe` on Windows and `/bin/bash` in WSL.
- **Negative test:** a scratch copy of the *pre-P2* Makefile run in WSL still fails `make clean` with `Error 127`. That shows the WSL check can tell the difference.
- In Git Bash, `env PATH=/usr/bin "$(command -v make)" lint` exits 2 and prints the install hint, because golangci-lint is not under `/usr/bin`. This command changes no file.
- Then run the Stability rule on both hosts. This is the first WSL `make preflight` ever run. Any Linux-only failure in an existing step is a deviation (Stability rule).

### P3 — Docs gates: markdownlint and records (D2, D6; closes F2, F5, F6)

1. **Copy the markdownlint config:** `cp <sibling>/go-selfupdate-lib/.markdownlint-cli2.jsonc .markdownlint-cli2.jsonc`, then `cmp` the two files. Expected: exit 0.
2. **Create `scripts/check_records.py`.** Copy it from magic-cli-remote at `9778cbc17b39`, then make exactly these deletions and one replacement, using a scratch Python script that asserts each anchor occurs exactly once:
   - Replace the docstring (lines 2–19) with:

     ```text
     """Record and docs-link tooling for the fixed docs tree.

     Taken from magic-cli-remote's scripts/check_records.py (its PLAN 0175 /
     0180), without the apps/mobile tree and without --write-index: this
     repository's docs/README.md index is written by hand
     (docs/decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md D6).

     Modes:
       --next         print the next unused NNNN across the whole repository
       --check        validate numbered records: filename shape, number pairing,
                      placement, and relative markdown links
       --check-all    --check plus link checks for unnumbered docs
                      (docs/guides/**, docs/README.md, docs/architecture.md,
                      and the root README.md)

     Errors print to stdout, warnings to stderr; exit status is 1 iff errors were
     found. Warnings: a number claimed by more than one MADR, a PLAN without a
     same-number MADR, a GATES without a same-number PLAN, and records sitting
     outside their kind directory.
     """
     ```

   - Delete the constants `KIND_ORDER`, `TOC_BEGIN`, `TOC_END` and `INDEX_PATH`.
   - Delete the method `Record.title`.
   - In `unnumbered_docs`, delete the two `apps/mobile` `elif` branches.
   - Delete the function `write_index`.
   - In `main`, delete:
     - the `--write-index` argument;
     - `or args.write_index` from the no-mode condition;
     - the `if args.write_index:` block.

   Verification for this step:
   - `python3 -m py_compile scripts/check_records.py` exits 0. The resulting `__pycache__` is ignored (P1).
   - `git diff --no-index <sibling>/magic-cli-remote/scripts/check_records.py scripts/check_records.py` contains only those changes. Read the diff in full, not through `head`.
3. **Add the Makefile targets.** Add `check-records markdownlint` to the `.PHONY` list. After the `fix-check` recipe, insert:

```make
# Records and docs links (0007-MADR D6).
check-records:
	python3 scripts/check_records.py --check-all

# The fleet markdownlint config; records are excluded by its own globs
# (0007-MADR D2).
markdownlint:
	npx --yes markdownlint-cli2@0.23.2
```

   In `preflight`, before `@echo "preflight passed"`, insert:

```make
	@echo "==> check-records"; $(MAKE) --no-print-directory check-records
	@echo "==> markdownlint"; $(MAKE) --no-print-directory markdownlint
```

4. **Fix F6 in `docs/README.md`.** Replace the line `- **[guides/](guides/)** — task-shaped walkthroughs (none yet).` with `- **guides/** — task-shaped walkthroughs. None exists yet; the first is \`developing.md\`, from [0004-PLAN](decisions/0004-PLAN-go-module-architecture.md) Phase 5.`
5. **Fix `docs/architecture.md`'s Tree block.**
   - Give its fence the language `text`.
   - Replace `  guides/                 unnumbered how-to documents (empty)` with `  guides/                 unnumbered how-to documents (not created yet)`.
   - Insert these lines after `NOTICE`, in this order, with the same column alignment:
     - `AGENTS.md                 agent instructions (P4 of 0007-PLAN adds it)`
     - `.markdownlint-cli2.jsonc  fleet Markdown style`
     - `.gitattributes            LF on every platform`
     - `scripts/check_records.py  record numbering and link check`

   P4 then rewrites the `AGENTS.md` line to `AGENTS.md                 agent instructions`.
6. **Fix the 29 markdownlint issues** at the lines MADR measurement 2 lists. Re-read the live line numbers from `make markdownlint` output first, because step 4 and the index row moved `docs/README.md`.
   - **MD013:** rewrap the reported paragraph or list item at no more than 100 columns, on word boundaries. A list item's continuation lines are indented two spaces. No word is added, removed or reordered.
   - **MD004:** on the 17 reported lines of the 0001 REPORT, replace the leading `* ` with `- ` and keep the indentation.
   - **MD040:** give the two bare fences the language `text`. One is `docs/architecture.md`'s Tree block (step 5). The other is the 0001 REPORT's `pi [options]` block.

**Verification:**
- **C4 check.** A scratch script compares, for each of the four edited Markdown files, `HEAD`'s text against the working tree's after normalising:
  - whitespace collapsed;
  - a leading `- ` or `* ` list marker reduced to one token;
  - the words ``` ```text ``` reduced to ``` ``` ```;
  - in `docs/architecture.md` and `docs/README.md`, the lines from steps 4 and 5 removed from both sides.

  Expected: equal for every file. On a scratch copy where one word in the REPORT is changed, the comparison must report a difference.
- `make check-records` exits 0. `python3 scripts/check_records.py --next` prints `0008`.
- `make markdownlint` exits 0 with `0 error(s)`.
- **Negative tests on a scratch copy of the tree:**
  - a planted `[x](missing.md)` in `README.md` fails `make check-records` with `broken relative link: README.md:<n>: missing.md`;
  - a planted `* item` list in `README.md` fails `make markdownlint` with `MD004/ul-style`;
  - the same 224-character word paragraph as P4's plant, appended to a scratch copy of the 0007-MADR, does **not** fail, because the config excludes records. As a control, the same paragraph in `README.md` **does** fail with `MD013`.

  Quote all three outputs.
- Then run the Stability rule on both hosts.

### P4 — Agent instruction set (D1; closes F1, F12)

1. **Copy the pointer files** from go-selfupdate-lib at `a65e12b8afa0`, byte-for-byte, creating directories as needed:
   - `.claude/.gitignore`
   - `.claude/rules/madr-and-plan-skill.md`
   - `.grok/rules/madr-plan-before-mutating-work.md`
   - `.opencode/rules.md`
   - `opencode.json`

   Then `cmp` each one against its source. Expected: exit 0 for all five.
2. **Create `AGENTS.md`** with exactly this text:

````markdown
# AGENTS.md

Instructions for AI coding agents working in this repository. All agents read
this file. A repository-local `CLAUDE.md` / `.claude/rules/` / `.grok/rules/` /
`.opencode/rules.md` wins only where it is more specific than this file.

`gobble-cli` is the repository for **gobble**, a Go rewrite of the Pi coding-agent
harness: one module, `github.com/maccavelli/gobble-cli`, holding the `gobble`
binary (`cmd/gobble`) and a contract-first Go SDK. Requires Go 1.27.1. gobble has
two terminal modes: a native CLI mode, the default, and an enhanced TUI mode. The
command library is Kong. Core TUI is go-tui-lib and self-update is
go-selfupdate-lib; gobble reimplements neither, and where a behaviour is not in
those libraries yet, gobble waits rather than copying it
(`docs/decisions/0004-MADR-go-module-architecture.md`). What exists today is in
`docs/architecture.md`; the build order is in `docs/README.md`.

## Dependencies

No module may be required without a MADR in this repository that names it.
`go.mod` and `go.sum` change with the code that needs them: a requirement is
added in the commit that adds its first import, and removed in the commit that
removes its last. `go mod tidy` is clean at every commit; `make preflight`
checks it.

Package tiers and import boundaries are the package map in
`docs/decisions/0004-MADR-go-module-architecture.md`. `internal/archtest`
enforces them (`make archtest`, part of `make preflight`). A change that adds a
forbidden import edge is fixed in the code, or by amending that record and its
rule together, never by editing the test alone.

Shipped binaries are pure Go: `make check-cgo-off` refuses `CGO_ENABLED` other
than `0` and any `import "C"`. `make race` turns cgo on for the race detector
only.

## MADR and PLAN before mutating work

**Whenever the user asks for an MADR and a plan, load the
`madr-and-plan-writing` skill first** and follow it for authoring, naming and
review. This applies both to writing a fresh pair and to amending an existing
one.

The name is exact — it is the `name:` field of the skill. A mistyped call
returns `Unknown skill`, and an agent that proceeds without the skill writes
something shaped like a MADR while missing the required headings and the
directory rule below. Verify against the filesystem rather than memory:

```bash
ls -d ~/.claude/skills/*madr* && grep '^name:' ~/.claude/skills/*madr*/SKILL.md
```

**Read-only investigation is allowed with no pair.** Reading, searching,
`git log` / `git show` / `git diff`, and existing tests or diagnostics that
do not write the tree do not need a MADR.

**Mutating work is not.** Before the first write, name the
`docs/decisions/NNNN-MADR-*` / `docs/decisions/NNNN-PLAN-*` pair being
executed, or stop and write one.

Mutating means: creating, editing, or deleting files; staging or committing
(except the bootstrap exception below); dependency or lockfile changes;
CI / config / hook changes; builds or installers that write the tree,
`$HOME`, or a live service; generating committed artifacts.

Order:

1. Investigate (read-only).
2. Write or amend the MADR (`status: proposed` unless the owner already
   decided). Present it. Do not implement.
3. Write or amend the PLAN. Present it.
4. Mutate **only after** the owner explicitly approves execution
   (`proceed`, `execute the plan`, `do phase N`). Stay inside that PLAN.
5. Anything discovered mid-execution that is out of scope waits: amend the
   pair, re-approve, then continue. Completing a phase is not permission to
   invent the next unwritten one.

Follow-up vs greenfield:

- **Same topic** (debug, leftover phase, bug found in that plan's live
  run): amend that number. Add a PLAN phase or an amendment in the MADR. Do
  not silently rewrite historical rationale.
- **Greenfield**: next unused `NNNN`, new MADR, new PLAN, same slug. No
  mutation until that PLAN is approved.

Bootstrap exception: authoring `docs/decisions/NNNN-MADR-*`,
`docs/decisions/NNNN-PLAN-*`, this file, and the per-agent pointers
(`.claude/rules/`, `.grok/rules/`, `.opencode/rules.md`) does not require a
*prior* pair. Putting source, tests, CI, or product config in that same commit
is a violation.

`git push` and tags still need an explicit ask in the same turn.

## Records

```text
docs/decisions/NNNN-MADR-short-slug.md
docs/decisions/NNNN-PLAN-short-slug.md
docs/reports/NNNN-REPORT-short-slug.md
docs/reports/NNNN-GATES-short-slug.md
```

- `NNNN` is a zero-padded 4-digit number, one sequence across
  `docs/decisions/` and `docs/reports/`. A MADR and its PLAN share the same
  number and the same slug.
- **Next number** comes from `python3 scripts/check_records.py --next`.
  Never reuse a number, never renumber an existing record, never leave a gap
  deliberately.
- `make check-records` validates naming, pairing, placement, and every
  relative link in the records and the unnumbered docs.
- Cite records by full filename, never by number alone. Cite another
  repository's record by repository and filename; a relative link cannot reach
  it.
- `docs/README.md` indexes every record by hand. Update it in the same change.

## Pre-add checks

Before staging Go files, run:

```bash
make pre-add-check                 # every tracked Go file
make pre-add-check FILES="a.go b.go"
```

It runs `scripts/go-precheck.sh`: plain `gofmt` (not `gofumpt`) on the files,
`golint` per file, and `govulncheck ./...` (`GO_PRECHECK_SKIP_VULN=1` skips it
offline). `golint` and `govulncheck` run from the `tool` directives in `go.mod`
(`go tool golint`, `go tool govulncheck`), not from a global install. A file
that fails is not committed.

Before every commit, run `make preflight`. It runs every gate: gofmt drift,
`go mod tidy`, the pre-add check, `go vet`, `staticcheck` for linux, darwin and
windows, `golangci-lint` (v2.14.0), `govulncheck`, `go fix -diff`, `archtest`,
`apidiff`, `verify-build-metadata`, `check-records`, and `markdownlint`.

Run `make` from Git Bash, or from PowerShell. The Makefile uses Git's bash when
`OS` is `Windows_NT` (set `GIT_BASH` if Git is installed elsewhere) and
`/bin/bash` on every other host. In PowerShell, a bare `bash` is WSL, not Git
Bash.

## Identifiers

Nothing committed carries a hostname, account name, org-internal path, or a
real-machine absolute path. Use placeholders (`<user>`, `/home/<user>/...`).

- When recording an identifier scan in a record or a commit, describe what was
  scanned for ("the local account name", "the hostname domain"); never quote
  it.
- Pushes to GitHub pass through a global pre-push disclosure guard. Before
  asking the owner to push, run the guard itself over the outgoing commits:

  ```bash
  echo "refs/heads/main $(git rev-parse HEAD) refs/heads/main $(git rev-parse origin/main)" |
    python3 ~/.global-git-hooks/github-disclosure.py pre-push origin "$(git remote get-url origin)"
  ```

- Never bypass it with `--no-verify`.

## Commits

`git commit --no-edit`. Never pass `-m` / `--message` / `-F`. The global
`prepare-commit-msg` hook writes the message from the staged diff.
````

   If P0(b) chose the organisation subject form, replace the Commits section's body with that rule as the owner states it, and record the change in the Execution record. This is the only permitted variation from the text above.
3. **Update the `AGENTS.md` line in `docs/architecture.md`'s Tree block** to `AGENTS.md                 agent instructions`.

**Verification:**
- `make markdownlint` exits 0 with `AGENTS.md` in its file count.
- Each `ls -d …` command quoted in `AGENTS.md` and the pointer files resolves on this host. The `.grok` one uses `~/.grok/skills`.
- `python3 ~/.global-git-hooks/github-disclosure.py --help`, or equivalent, shows that the guard script exists at the quoted path.
- `grep -n 'user.name' AGENTS.md` prints nothing (F12).
- **Negative test:** on a scratch copy, append to `AGENTS.md` a paragraph of 45 repetitions of `word` separated by single spaces (224 characters, with spaces past column 200). `make markdownlint` must fail with `MD013/line-length … [Expected: 200; Actual: 224]`. That proves `AGENTS.md` is inside the lint globs.
  - A 201-character run with **no** spaces is not a valid plant. MD013's non-strict default ignores a line with no whitespace past the limit. Such a plant passed with `0 issues` when this plan was written, on 2026-10-04.
- Then run the Stability rule on both hosts.

### P5 — Linter set (D7; closes F7)

1. **In `.golangci.yml`**, replace `    - unparam\n  settings:\n` with `    - unparam\n    - unused\n    - whitespace\n  settings:\n`. Assert exactly one occurrence first.
2. **Append to `.golangci.yml`**, at the end of the file:

   ```yaml
   formatters:
     enable:
       - gofmt
       - goimports
     exclusions:
       generated: lax
       paths:
         - third_party$
         - builtin$
         - examples$
   ```

3. **Diff against mcplib.** Run `git diff --no-index <sibling>/mcplib/.golangci.yml .golangci.yml`. Expected: only mcplib's application-filename exclusion rule and its `godot` text rule remain as differences. Read the diff in full.

**Verification:**
- For each of `GOOS=linux`, `darwin` and `windows` with `CGO_ENABLED=0`, `golangci-lint run -c .golangci.yml ./...` reports `0 issues.` on the tree.
- **Negative tests.** Each plant goes in its own scratch copy, each is confirmed `gofmt -l`-clean first except where gofmt is the target, and each must fail with the linter named. The plant bodies are those of MADR measurement 5:
  - `hook/zz_planted.go` with a blank line before a function's closing brace → `unnecessary trailing newline (whitespace)`;
  - a blank line after `if x > 0 {` → `unnecessary leading newline (whitespace)`;
  - an import block mixing `"fmt"`, `"github.com/coder/acp-go-sdk"` and `"os"` → `File is not properly formatted (goimports)`;
  - an unexported function nothing calls → `is unused (unused)`;
  - imports `"os"` before `"fmt"` → `(gofmt)`.

  Quote each failure.
- Then run the Stability rule on both hosts.

### P6 — Records (D8; closes F9 and F10's record side)

1. **In `docs/decisions/0004-PLAN-go-module-architecture.md`, strike in place, without deleting:**
   - Line 38: `` `.golangci.yml`, `.editorconfig`, `LICENSE`, `NOTICE` `` becomes `` `.golangci.yml`, ~~`.editorconfig`~~ *(struck 2026-10-04: no fleet repository has one; 0007-MADR D8)*, `LICENSE`, `NOTICE` ``.
   - At the end of Phase 0 step 7's paragraph, append: ` *(2026-10-04: mcplib's set is these plus \`unused\` and \`whitespace\`, with the \`gofmt\` and \`goimports\` formatters; 0007-PLAN P5 adds them.)*`
   - At the end of Phase 4 step 1's list, append the item: `* *(2026-10-04, 0007-MADR D8: see the amendment of that date for the CI hygiene this job also carries.)*`
2. **Append this amendment** after 0004-PLAN's last line:

   ```markdown

   **2026-10-04 — repository scaffold (0007-MADR D8).** [0007-MADR-repository-scaffolding-to-fleet-standard.md](0007-MADR-repository-scaffolding-to-fleet-standard.md) brought the repository up to the fleet scaffold. This PLAN changes as follows:

   * **Scope.** `.editorconfig` is struck. No fleet repository has one; gofmt, `.gitattributes` and markdownlint cover its job.
   * **Phase 0 step 7.** The linter set is mcplib's full set: the 18 listed plus `unused` and `whitespace`, with the `gofmt` and `goimports` formatters. 0007-PLAN P5 added them.
   * **Phase 4, CI hygiene,** in addition to the steps above:
     * the workflow declares `permissions: contents: read` at the top level; the publish job keeps its own wider block;
     * `concurrency: { group: ci-${{ github.ref }}, cancel-in-progress: true }`;
     * `actions/checkout` runs with `persist-credentials: false`;
     * on Linux, golangci-lint is installed at `v2.14.0`, and `go mod tidy -diff` runs;
     * on Linux, shellcheck `v0.11.0` is downloaded with its SHA-256 checked, and runs over `scripts/*.sh`;
     * on Linux, actionlint `v1.7.12` runs over `.github/workflows/`.

     `make preflight` already runs markdownlint and the records check, so Phase 4 adds no separate step for them.
   * **Phase 4, Makefile.** `make preflight` runs on Linux and macOS since 0007-PLAN P2, which this phase's CI depends on.
   ```

3. **`docs/README.md`:**
   - Set the 0007 MADR row's status to the owner's decision. `accepted` is expected once this plan is approved.
   - Set the 0007 PLAN row to `completed`.
   - Set 0004-PLAN's status cell to add "amended 2026-10-04 (0007)".
   - Add these "I want to…" rows:
     - `| know the rules agents follow in this repository | [AGENTS.md](../AGENTS.md) |`
     - `| know why the repository has the files and gates it has (AGENTS.md, lint configs, records check) | [0007-MADR](decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md) |`
4. **This PLAN:**
   - Set `status: completed`.
   - Append `## Execution record (YYYY-MM-DD)`. It records: the P0 answers; each phase's commit; each gate's exit status on both hosts; every negative test's quoted failure; and what the plan predicted wrongly.
5. **0007-MADR:**
   - Set `status` to the owner's decision.
   - Append `## Observed — execution results (YYYY-MM-DD)`, covering whether F4's fix held under a full Linux preflight and anything execution contradicted.

**Verification:**
- `make check-records` exits 0. `make markdownlint` exits 0.
- `grep -n '~~' docs/decisions/0004-PLAN-go-module-architecture.md` shows the new strike, and the struck text is still present inside it.
- **Identifier scan.** A scratch script reads the local account name, the WSL account name and the organisation's mail and git-host domains from the environment at run time, so none of them is written into the script. It scans every file `git diff --name-only main...HEAD` lists for those names and for the macOS, Linux and Windows user-profile path prefixes. The prefixes are written in the script, not in this plan, so the plan cannot match itself. Expected: no hit, apart from `/home/<user>/` placeholders.

  Then show it can fail: copy `AGENTS.md` to a scratch path, append the local account name, and confirm the scan reports it.
- Then run the Stability rule on both hosts.

## Verification (whole plan)

From a fresh copy of the branch in WSL, and in Git Bash on Windows:

```text
make preflight                                   # exit 0; ends "preflight passed"
python3 scripts/check_records.py --next          # 0008
git ls-files --eol | grep -v 'attr/text=auto eol=lf'   # no output
for f in .claude/.gitignore .claude/rules/madr-and-plan-skill.md \
         .grok/rules/madr-plan-before-mutating-work.md .opencode/rules.md \
         opencode.json .markdownlint-cli2.jsonc; do
  cmp "$f" "<sibling>/go-selfupdate-lib/$f" || echo "DIFF $f"
done                                             # no DIFF lines
git diff --cached --stat                         # every In-scope file staged; the owner commits
```

Every check added by this plan has a recorded failure on a planted input:

- `.gitattributes`: the CRLF clone pair (P1);
- the WSL Makefile check (P2);
- the lint install hint (P2);
- `check-records`, `markdownlint` and the C4 comparison (P3);
- `AGENTS.md` in the lint globs (P4);
- the four linters and the formatters (P5);
- the identifier scan (P6).

### Acceptance criteria (mapped to MADR Confirmation)

| # | Criterion | MADR |
|---|---|---|
| A1 | `check_records.py --next` prints `0008`, and `make check-records` exits 0 | Confirmation lines 1–2; D6 |
| A2 | `make markdownlint` exits 0 under the byte-identical fleet config | Confirmation line 3; D2 |
| A3 | `make preflight` exits 0 in Git Bash on Windows | Confirmation line 4; D5 |
| A4 | `make preflight` exits 0 in WSL Ubuntu 24.04 | Confirmation line 5; D5, F4 |
| A5 | Every tracked file is `attr/text=auto eol=lf` | Confirmation line 6; D3 |
| A6 | The six copied files are `cmp`-identical to go-selfupdate-lib `a65e12b8afa0` | Confirmation lines 7–8; D1, D2 |
| A7 | `scripts/__pycache__/x.pyc` is ignored | Confirmation line 9; D4 |
| A8 | Planted bad link, `*` list item, and each of `unused`, `gofmt`, `whitespace`, `goimports` fail as named | Confirmation, negative checks; D2, D6, D7 |
| A9 | The pre-P2 Makefile fails in WSL with Error 127, and the post-P2 one passes | Confirmation, negative checks; D5 |
| A10 | The identifier scan is clean and is shown failing on a plant | Confirmation, last line; C6 |
| A11 | 0004-PLAN carries the D8 amendment and keeps its struck text | D8 |
| A12 | The four reformatted Markdown files are word-identical to `HEAD` | D2; C4 |

**The criterion most likely to be dropped is A4.** WSL preflight is slower and runs a full module build, staticcheck for three GOOS values, and govulncheck on a second host. It is tempting to accept `make clean` passing as proof. It is not: F4 is the reason no Linux preflight has ever run, so A4 is the first time the existing gates meet Linux, and the place a hidden defect would show up.

## Rollout and Rollback

- There are no users and no release. ~~Each phase is one commit on `feature/<JIRA-KEY>-0007-scaffolding`.~~ Each phase is staged on `feature/0007-scaffolding`, and the owner commits and pushes (2026-10-04).
- Merging to `main` is the owner's act, through a merge or pull request they choose. This plan opens none unless asked.
- **Rollback.** `git revert` the phase commit. The phases are independent, except that:
  - P3's `check-records` target needs P1's `.gitignore` entry to keep `__pycache__/` out of the tree;
  - P4's lint coverage needs P3.

  Reverting P2 restores the Windows-only Makefile, and with it F4.

## Deferred (named, so they are not mistaken for oversights)

- **`ci.yml` and the release workflow.** 0004-PLAN Phase 4 owns them, with this plan's amendment. Writing CI before the Phase 2 binary exists would leave `verify-build-metadata` and `release-dry-run` with nothing to check.
- **A test for `scripts/go-precheck.sh`**, like go-selfupdate-lib's `go-precheck_test.sh`. It is worth having, but it tests the pre-add contract, which 0004-PLAN Phase 0 owns. It belongs in an amendment there.
- **Running `golangci-lint` per GOOS in `make lint`**, as the libraries do. gobble's `staticcheck` already runs per GOOS. Making `lint` do the same is a toolchain change for 0004-PLAN, not a scaffold copy.
- **The `docs/architecture.md` rewrite.** Its "What is not here" section is stale. 0004-PLAN Phase 5 rewrites the file.
- **`SECURITY.md`, `CODEOWNERS`, `CONTRIBUTING.md` and `CHANGELOG.md`.** These need a fleet-wide record, likely in magic-cli-remote, and gobble follows it (0007-MADR D9).
- **Carrying fixes from magic-cli-remote's `check_records.py`.** The copy is a fork, and fixes travel by hand (0007-MADR Consequences).

## Execution record (2026-10-04)

**Status.** P0–P6 ran on 2026-10-04 on `feature/0007-scaffolding`. Every changed file is staged. Nothing is committed: per the owner's P0 answer, the owner commits and pushes. No phase deviated from its decisions. The prediction errors below are about the verification method, not the change.

**P0 answers.**
- (a) The project is exempt from the Jira key, so the branch is `feature/0007-scaffolding`.
- (b) The agent adds and stages; the owner commits and pushes.

The Stability rule's commit lines and P0 step 3 carry dated deprecation marks, and the original text is kept.

**Gates.** After every phase, `make preflight` exited 0 in Git Bash on Windows. From P2 on, it also exited 0 in WSL Ubuntu 24.04 on a copy of the working tree. From P3 it ran 13 steps: gofmt, `go mod tidy`, pre-add-check, `go vet`, staticcheck (×3 GOOS), golangci-lint, govulncheck, fix-check, archtest, apidiff, verify-build-metadata, check-records, markdownlint. `git diff --check` was clean after every phase. A4, the first Linux preflight in the repository's history, passed on its first run (P2): nothing Linux-only was hiding behind F4.

**Negative tests, each seen to fail.** All ran on scratch copies.
- **P1.** With the host's global attributes isolated (see the first prediction error), a clone without `.gitattributes` checked `README.md` out with CRLF, and all 72 files lacked `attr/text=auto eol=lf`. With `.gitattributes`: no CRLF, and 0 files lacking it.
- **P2, shell.** The pre-P2 Makefile in WSL: `make: C:/PROGRA~1/Git/usr/bin/bash.exe: No such file or directory`, `Error 127`, exit 2. The post-P2 Makefile: exit 0 in Git Bash, PowerShell 7 and WSL. `make -p` showed `SHELL := C:/PROGRA~1/Git/usr/bin/bash.exe` on Windows and `SHELL := /bin/bash` in WSL.
- **P2, lint hint.** With golangci-lint's directory removed from `PATH`, `make lint` printed `golangci-lint not found. Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0` and exited 2.
- **P3, records.** A planted `[x](missing.md)` in `README.md` gave `broken relative link: README.md:84: missing.md`, exit 2.
- **P3, markdownlint.**
  - A planted `* item` gave `MD004/ul-style`, exit 2.
  - A 224-character word paragraph gave `MD013/line-length [Expected: 200; Actual: 224]` in `README.md`. In the 0007-MADR it passed, because records are excluded.
- **P3, C4.** One word changed in the REPORT ("harness" to "harnesses") made the word-identity comparison report `DIFFERS` at token 417, exit 1. On the real edit, all four files were word-identical to `HEAD`.
- **P4.** The same 224-character paragraph appended to `AGENTS.md` gave `MD013`, exit 2, so `AGENTS.md` is inside the lint globs.
- **P5.** Each plant failed with its linter named:
  - `unnecessary trailing newline (whitespace)`;
  - `unnecessary leading newline (whitespace)`;
  - `File is not properly formatted (goimports)`;
  - `func plantedUnused is unused (unused)`;
  - `File is not properly formatted (gofmt)`.

  The tree itself: `0 issues.` on linux, darwin and windows.
- **P6.** The identifier scan is recorded in the 0007-MADR's Observed section.

**What the plan predicted wrongly.**
1. **The host's global attributes hid the P1 checks.** P1's checks assumed that the only attributes in play were the repository's. This host's user-global `core.attributesFile` already says `* text=auto eol=lf`. With it in place, `git ls-files --eol` reported every file as `eol=lf` without the repository's file, and a `core.autocrlf=true` clone produced no CRLF. Neither check could fail. Both were re-run with `core.attributesFile` pointed at an empty file.
2. **The CRLF test method changed.** The plan's CRLF pair needed a scratch commit of `.gitattributes`. `git cat-file --filters --path=README.md HEAD:README.md` gives the same evidence with no commit and no checkout, so it was used instead.
3. **P1's renormalise expectation was wrong in form.** P1 expected `A  .gitattributes` from the renormalise clone. It printed `?? .gitattributes`, because `git add --renormalize` touches tracked files only. The evidence that mattered, no tracked file changed, holds.
4. **The Stability rule's WSL log was overwritten.** As written, it redirected `wsl.exe`'s output to a Windows file. `wsl.exe` relays stderr through a second handle, and Go's `downloading` line overwrote the first log lines, `==> gofmt` and `==> go mod tidy`. The exit status was unaffected. The gate was re-run with the log written inside Linux and then printed, and every later phase used that form.
5. **P2's lint-hint command could not start make.** `env PATH=/usr/bin make lint` exited 127, because `make` on this host is a native Windows executable and needs the Windows system directories (`api-ms-win-crt-utility-l1-1-0.dll`). The test used the full `PATH` minus golangci-lint's directory instead.
6. **The markdownlint count at step 6 was 28, not 29.** P3 step 5 had already fixed the architecture Tree fence, one of the 29.
7. **The per-phase whitespace check missed staged files.** The Stability rule's `git diff --check` compares the working tree with the index only. This PLAN was staged at P0, so no per-phase check covered it. The whole-plan run of `git diff --check --cached` reported `space before tab in indent` on ten lines. They were the Makefile recipe lines inside this PLAN's list-indented ` ```make ` blocks, which were three spaces followed by the tab that `make` requires.

   Fix: the four affected blocks were moved to column 0. Their tabs are unchanged, and their recipe lines were checked against the real `Makefile`. No check was loosened. The whole-plan check is `git diff --check --cached main`.
