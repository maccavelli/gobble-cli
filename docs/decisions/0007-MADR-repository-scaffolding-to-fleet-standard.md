---
status: accepted
date: 2026-10-04
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Bring gobble-cli up to the fleet repository scaffold: agent rules, lint and line-ending config, records tooling, and a portable Makefile

## Context and Problem Statement

The owner asked on 2026-10-04 for gobble-cli's missing "standard repo scaffolding" to be put in place: "assess the sibling repos, determine what we need to duplicate in this repo to keep things consistent, and standards/best practices adherent".

gobble-cli has a Go module, a Makefile, a lint config, a pre-add script, and a docs tree (0004-PLAN Phases D, 0 and 1). It has none of the files that the other fleet repositories use to tell agents and tools how to work in them. Its Makefile also has a defect that would stop the CI that 0004-PLAN Phase 4 plans to add.

This record covers the scaffold around the code. It does not cover the code, and it does not cover the CI and release pipeline, which [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md) Phase 4 owns. It changes Phase 4 only by amendment (D8).

**The comparison set.** The owner's own Go repositories on GitHub are the reference:

- **go-selfupdate-lib**, the newest scaffold. Its `docs/decisions/0001-MADR-scaffold-shared-go-library.md` records the scaffold as "go-llmprovider-sdk, adapted". Its scaffold files are byte-identical to go-llmprovider-sdk's (table below).
- **go-llmprovider-sdk**, the source of that scaffold.
- **magic-cli-remote**, the CLI application. gobble's Makefile already names it as the source of its target names (`Makefile:1-4`).
- **mcplib**, the stated source of gobble's linter set (0004-PLAN Phase 0 step 7).

The following were looked at and excluded:

- Upstream clones that are not the owner's work: codex, goose, kilocode, opencode, grok-build, and the acp-go-sdk fork.
- Repositories that are not Go: a Flutter app, a React starter, and a dotfiles repository.
- go-tui-lib, which holds one file.
- Three repositories on the organisation's internal GitLab. They use a different host, a different CI system (`.gitlab-ci.yml`), and record layouts that predate the fleet tree.

### What was measured, not assumed

All checks were read-only against the live repositories, or were run on scratch copies outside every repository, on 2026-10-04.

1. **File presence and content hash** (SHA-256, first 12 hex characters) across the comparison set:

   | File | go-llmprovider-sdk | go-selfupdate-lib | magic-cli-remote | mcplib | gobble-cli |
   |---|---|---|---|---|---|
   | `AGENTS.md` | yes | yes | yes | yes | **no** |
   | `.claude/.gitignore` | `ced771be89a8` | `ced771be89a8` | `ced771be89a8` | no | **no** |
   | `.claude/rules/madr-and-plan-skill.md` | `12c6efea28ca` | `12c6efea28ca` | `2c3c7aa889a2` | no | **no** |
   | `.grok/rules/madr-plan-before-mutating-work.md` | yes | yes | yes | no | **no** |
   | `.opencode/rules.md` | yes | yes | yes | no | **no** |
   | `opencode.json` | `40a650f6e916` | `40a650f6e916` | `5e153f288ee2` | no | **no** |
   | `.markdownlint-cli2.jsonc` | `3c7bf774fdad` | `3c7bf774fdad` | `3c7bf774fdad` | `3c7bf774fdad` | **no** |
   | `.gitattributes` | no | `81595007a946` | `2ed6def6f9c5` | no | **no** |
   | `.github/workflows/ci.yml` | yes | yes | yes | yes | **no** (0004-PLAN Phase 4) |
   | `.github/dependabot.yml` | no | no | yes | no | no |
   | `.claude/settings.json`, `.kilo/` | no | no | yes | no | no |
   | `scripts/check_records.py` | no | no | yes | no | **no** |
   | `docs/guides/` | yes | yes | yes | no | **no** |
   | `LICENSE` | `c71d239df917` | `c71d239df917` | `c71d239df917` | no | `c71d239df917` |
   | `.editorconfig`, `SECURITY.md`, `CODEOWNERS`, `CONTRIBUTING.md`, `CHANGELOG.md` | no | no | no | no | no |

2. **Markdown lint.** `markdownlint-cli2@0.23.2` ran on a scratch copy of gobble's tracked Markdown with the fleet `.markdownlint-cli2.jsonc`. It linted 4 files and found 29 issues; exit 1. The issues by rule:
   - 10 × MD013 (line length): `README.md` 5, `docs/architecture.md` 4, `docs/README.md` 1.
   - 17 × MD004 (`*` list marker where the fleet pins `-`): `docs/reports/0001-REPORT-go-port-feasibility.md`.
   - 2 × MD040 (fence without a language): `docs/architecture.md:26` and `docs/reports/0001-REPORT-go-port-feasibility.md:129`.

   With no config file, the same copy gave 874 issues in 11 files, because the default globs include the MADR and PLAN records that the fleet config excludes.
3. **Makefile under Linux.** A scratch copy of gobble's `Makefile` was run under WSL Ubuntu 24.04 with GNU Make 4.3, using `wsl.exe --exec make clean`. `--exec` keeps the distribution's login shell from expanding `$?` early. An earlier run without it printed a false `exit=0`, and that run was discarded.
   - As committed: exit 2, with `make: C:/PROGRA~1/Git/usr/bin/bash.exe: No such file or directory` and `make: *** [Makefile:126: clean] Error 127`.
   - With the `SHELL` line replaced by `SHELL := bash` and the `export PATH` line removed: exit 0.
4. **Records tooling.** magic-cli-remote's `scripts/check_records.py` ran on a scratch copy of gobble's tracked tree:
   - `--next` printed `0007`.
   - `--check` exited 0.
   - `--check-all` exited 1 with `broken relative link: docs/README.md:10: guides/`.
   - On a second copy with a planted `[architecture.md](missing.md)`, `--check-all` reported both broken links and exited 1.
5. **Linter set.** gobble's `.golangci.yml` enables 18 linters. mcplib's enables 20. The difference is `unused` and `whitespace`; mcplib also has a `formatters:` block with `gofmt` and `goimports`. golangci-lint 2.14.0 (go1.27.1) ran on a scratch copy of gobble with those three additions, for `GOOS` linux, darwin and windows with `CGO_ENABLED=0`:
   - Current tree: `0 issues.`, exit 0, on all three.
   - With a planted file: exit 1 on all three. It reported `File is not properly formatted (gofmt)` and `func unusedHelper is unused (unused)`.
   - The planted file also had a blank line at the start of a function body. `whitespace` did **not** report it. This record does not claim that `whitespace` has been seen to fail.
   - *Added 2026-10-04, while writing the PLAN.* Separate plants, each confirmed `gofmt`-clean first so a formatting finding could not hide the others, were each linted on their own copy:
     - A blank line before a function's closing brace gave `unnecessary trailing newline (whitespace)`.
     - A blank line after `if x > 0 {` gave `unnecessary leading newline (whitespace)`.
     - An import block that mixed `"fmt"`, `"github.com/coder/acp-go-sdk"` and `"os"` in one group gave `File is not properly formatted (goimports)`.
     - A blank line straight after a function's opening brace was still not reported. `whitespace`'s default settings do not check that case.
8. **Portable `SHELL`, three shells.** *Added 2026-10-04, while writing the PLAN.* A scratch Makefile held the D5 conditional: `ifeq ($(OS),Windows_NT)`, then `GIT_BASH ?=`, `SHELL := $(GIT_BASH)` and `export PATH := $(dir $(GIT_BASH)):$(PATH)`, `else` `SHELL := /bin/bash`, with the existing `.SHELLFLAGS`. Every run exited 0 with `pipefail on`:
   - Git Bash and PowerShell 7 resolved Git's bash and Git's `grep`.
   - WSL Ubuntu 24.04 resolved `/bin/bash` and `/usr/bin/grep`.

   The WSL distribution has go 1.27.1, golangci-lint 2.14.0, Node, Python 3 and GNU Make, so `make preflight` can be run there.
6. **Line endings.** `git ls-files --eol` reports all 72 tracked files as `i/lf w/lf`. On this host, `core.autocrlf` is `false` both locally and globally.
7. **Local git identity.** `git config --local user.name` is unset in gobble-cli, go-selfupdate-lib, go-llmprovider-sdk and magic-cli-remote. The hooks path resolves to the global hooks directory, so the commit-message generator applies.

### Findings

**F1. gobble has no agent instruction files.** Every agent that works in the repository falls back to global rules. Those rules do not know gobble's gates (`make preflight`, `archtest`, `check-cgo-off`), its dependency rules, or its numbering. Both libraries carry the same set, byte-identical: `AGENTS.md`, `.claude/rules/madr-and-plan-skill.md`, `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md`, `opencode.json` and `.claude/.gitignore`. magic-cli-remote carries the same set plus extras.

**F2. gobble has no markdownlint config, and its non-record Markdown fails the fleet config with 29 issues in 4 files.** The fleet file is byte-identical in all four compared repositories. Without it, an editor or CI run applies markdownlint's defaults, which lint the records as well (874 issues). 0004-PLAN's execution record has already had to work around this by counting the default-config issues before and after each edit.

**F3. gobble has no `.gitattributes`.** Nothing is broken today: every file is LF and this host does not convert line endings. Two upcoming phases change that. 0004-PLAN Phase 3 adds golden transcripts, and Phase 4 adds a Windows CI leg. magic-cli-remote had every golden test fail on `windows-latest` for this reason (its MADR 0116 P11, quoted in its `.gitattributes`). Both siblings now force `eol=lf`. The claim that GitHub's Windows runners default to `core.autocrlf=true` is taken from those siblings' comments and is **[unverified]** here.

**F4. The Makefile cannot run on Linux or macOS.** `Makefile:6` sets `SHELL := C:/PROGRA~1/Git/usr/bin/bash.exe`, and `Makefile:10` prepends the same Windows directory to `PATH`. Both came in with `925ef0a` (0004-PLAN Phase 0). Every recipe fails with Error 127 off Windows (measurement 3). 0004-PLAN Phase 4 runs `make preflight` on ubuntu and macos runners, so its CI would be red on its first run for a reason unrelated to the code. No compared sibling Makefile sets `SHELL`. magic-cli-remote instead documents running make from Git Bash, or from PowerShell with Git's `usr/bin` on `PATH`. **This is a pre-existing defect**, confirmed on an unmodified copy of the committed file.

**F5. gobble has no records tooling.** Numbering and link checks have been done by hand: each Phase D run in 0004-PLAN's execution record wrote a throwaway link resolver. magic-cli-remote's `check_records.py` already does the job: it validates naming, pairing, placement and relative links, and prints the next number. It worked on gobble's tree without changes (measurement 4), and it was seen to fail on a planted bad link.

**F6. `docs/README.md:10` links `guides/`, which does not exist.** The broken link is live today. The documentation tree rule is that a directory is created by its first real document, not left as an empty placeholder. 0004-PLAN Phase 5 writes the first guide (`docs/guides/developing.md`).

**F7. gobble's linter set is two linters and the formatters short of the set it claims to copy.** 0004-PLAN Phase 0 step 7 says "mcplib's linter set" and then lists 18 linters. mcplib, go-selfupdate-lib and go-llmprovider-sdk also enable `unused` and `whitespace`, plus the `gofmt` and `goimports` formatters. Adding them costs nothing on today's tree (0 issues on three GOOS values). `unused` and the `gofmt` formatter were seen to catch a planted defect; `whitespace` was not. *(2026-10-04: `whitespace` and `goimports` have since been seen to fail on their own plants; see measurement 5.)*

**F8. `.gitignore` lacks the Python artefact entries.** F5 brings a Python script into the repository, and the owner's global rule requires `__pycache__/` and `*.py[cod]` to be ignored before that happens. The sibling `.gitignore` files also ignore `*.log`, `*.tmp`, `tmp/`, `temp/`, `.cache/`, `.worktrees/` and `coverage_*.out`, and gobble's does not.

**F9. 0004-PLAN's Scope lists `.editorconfig` (`0004-PLAN-go-module-architecture.md:38`), but Phase 0 is marked complete without one.** No compared repository has an `.editorconfig`. In the fleet, Go formatting is `gofmt`'s job, line endings are `.gitattributes`'s job, and Markdown style is markdownlint's job. The scope line is stale.

**F10. go-selfupdate-lib's CI has hygiene steps that 0004-PLAN Phase 4 does not list.** They are:
- a top-level `permissions: contents: read`;
- `concurrency` with `cancel-in-progress`;
- `persist-credentials: false` on checkout;
- `go mod tidy -diff`;
- golangci-lint installed at a pinned `v2.14.0`;
- shellcheck `v0.11.0`, checksum-verified;
- `markdownlint-cli2@0.23.2`;
- actionlint `v1.7.12`.

gobble has a shell script (`scripts/go-precheck.sh`), Markdown, and will have workflows, so each of these applies.

**F11. Some sibling scaffolding should not be copied.** Each item has a reason:
- Dependabot is off by fleet policy (0004-PLAN Phase 4 step 4, and go-selfupdate-lib 0001-MADR §8). Only magic-cli-remote has it.
- `.claude/settings.json` and `.kilo/` exist only in magic-cli-remote, and both are specific to that application.
- `depguard` import rules: gobble enforces its import rules with `internal/archtest` (0004-MADR).
- The libraries' replacement of `golint` with `revive`: gobble follows magic-cli-remote's `golint` contract (0004-PLAN Phase 0 step 6).
- Release scripts: 0004-PLAN Phase 4 fetches them at a pinned SHA.

**F12. One sentence in the sibling `AGENTS.md` is untrue where it is used.** go-selfupdate-lib's `AGENTS.md` says the repository "sets a local `user.name` / `user.email`". None of the four checked repositories does (measurement 7). gobble's copy must not carry that sentence.

**F13. The working branch is `main`.** The organisation's Git rule (ADR 011) forbids committing on `main`. Executing this record therefore starts with a branch. The record's own commit does too.

## Decision Drivers

- **Consistency.** An agent or maintainer who moves between fleet repositories should find the same files, the same words and the same commands.
- **Gates that tell the truth.** Every check added is seen to fail before it is relied on. No check is taught to pass on nothing.
- **No second owner for one thing.** CI and release belong to 0004-PLAN Phase 4. Import rules belong to `internal/archtest`. This record must not create a second home for either.
- **Fix defects rather than route around them.** F4 and F6 are live defects.
- **Copy, then change only where gobble differs.** Every difference from the source file is stated with its reason, so a later diff against the sibling explains itself.
- **Best practice where the fleet is silent, but no unilateral divergence.** A file that no fleet repository has, such as `SECURITY.md`, is a fleet decision rather than a gobble one.

## Considered Options

- **A. Adopt the fleet library scaffold, adapted to a CLI, in one new record. CI stays in 0004-PLAN Phase 4, which this record amends.**
- **B. Fold the scaffold into 0004-PLAN** as another phase and amendment, with no new record.
- **C. Copy magic-cli-remote's scaffold whole**, including Dependabot, `.claude/settings.json`, `.kilo/`, `GNUmakefile` and the flake ledger.
- **D. Minimal: add `AGENTS.md` and its pointers only.**
- **E. As A, and also write `ci.yml` now**, taking CI out of 0004-PLAN Phase 4.

## Decision Outcome

Chosen option: **A**. It closes every finding and reuses the fleet's files nearly verbatim. It keeps CI and release in the one record that already owns them. The defects (F4, F6) are fixed rather than worked around.

### The decisions

**D1. Add the agent instruction set.**
- Take these files byte-for-byte from go-selfupdate-lib: `.claude/.gitignore`, `.claude/rules/madr-and-plan-skill.md`, `.grok/rules/madr-plan-before-mutating-work.md`, `.opencode/rules.md` and `opencode.json`.
- Write `AGENTS.md` from go-selfupdate-lib's sections. Keep its wording for "MADR and PLAN before mutating work", "Records", "Identifiers" and "Commits". Write the rest for gobble:
  - **Scope.** gobble is a CLI and SDK module. Core TUI is go-tui-lib. Self-update is go-selfupdate-lib. Neither is reimplemented here (0004-PLAN Goal).
  - **Dependencies.** No module is required without a record naming it. The import rules are enforced by `internal/archtest` and `make archtest`, not by depguard.
  - **Pre-add checks.** `make pre-add-check` runs `scripts/go-precheck.sh`: gofmt, per-file golint, govulncheck. `make preflight` runs every gate.
  - **Windows.** Run make from Git Bash, or from PowerShell with Git's `usr/bin` on `PATH`. In PowerShell, `bash` is WSL.
  - **Records.** The next number comes from `scripts/check_records.py --next` (D6).
- Drop the sentence on local `user.name` (F12).

**D2. Add the fleet `.markdownlint-cli2.jsonc` byte-for-byte, and fix the 29 issues it reports.**
- Wrap lines over 200 characters.
- Change `*` list markers to `-` in the 0001 REPORT.
- Add a language to the two bare fences.

The REPORT edit is formatting only. A normalised-text comparison must show that no word changed. The MADR and PLAN records stay excluded by the config's own globs.

**D3. Add `.gitattributes`** in go-selfupdate-lib's form: `* text=auto eol=lf` plus the binary patterns, with no fixture exemption, because gobble has no fixtures yet. It lands before 0004-PLAN Phase 3 adds any golden file. The record that adds a fixture testing line endings adds that fixture's `-text` line.

**D4. Merge `.gitignore`.** Keep every existing entry. Add `__pycache__/`, `*.py[cod]`, `coverage_*.out`, `*.log`, `*.tmp`, `tmp/`, `temp/`, `.cache/` and `.worktrees/` from the sibling file.

**D5. Make the Makefile portable.**
- Set `SHELL` and the `PATH` prefix only when `$(OS)` is `Windows_NT`.
- On Windows, take the shell from an overridable `GIT_BASH ?= C:/PROGRA~1/Git/usr/bin/bash.exe`, so that a Git installed elsewhere needs no edit.
- Everywhere else, use `SHELL := /bin/bash`. The recipes use `-o pipefail`, which needs bash, so `sh` will not do.
- Keep `.SHELLFLAGS` as it is.
- Add install hints that pin golangci-lint `v2.14.0`, the fleet's current pin, to the `lint` target.

**D6. Adopt `scripts/check_records.py` from magic-cli-remote.**
- Strip its `apps/mobile/` handling, which gobble has no use for.
- Keep `--next`, `--check` and `--check-all`.
- Do not use `--write-index`. gobble's index has a hand-kept Status column and a "Build order" section that the generator would not keep.
- Add `make check-records` (`--check-all`) and `make markdownlint` (`npx --yes markdownlint-cli2@0.23.2`), and add both to `make preflight`.
- Fix F6 by turning `docs/README.md`'s `guides/` bullet into plain text that names 0004-PLAN Phase 5 as the first guide. Phase 5 restores the link when `developing.md` exists.

**D7. Bring `.golangci.yml` to the set it claims.**
- Enable `unused` and `whitespace`.
- Add the `formatters:` block (`gofmt`, `goimports`) from mcplib.
- Leave everything else unchanged.

The PLAN proves `whitespace` and `goimports` on planted defects that each one actually reports, because measurement 5 did not see either fail.

**D8. Amend 0004-PLAN by dated entry, and leave the original text in place.** The amendment does three things:
1. Strike `.editorconfig` from Scope (F9).
2. Note under Phase 0 step 7 that the set is now mcplib's 20 linters plus formatters (D7).
3. Add F10's CI hygiene to Phase 4:
   - top-level `permissions: contents: read`;
   - `concurrency` with `cancel-in-progress`;
   - `persist-credentials: false`;
   - golangci-lint installed at `v2.14.0`;
   - `go mod tidy -diff`;
   - shellcheck `v0.11.0`, checksum-verified, over `scripts/*.sh`;
   - actionlint `v1.7.12`.

   `make preflight` already covers markdownlint and the records check (D6), so Phase 4 gains no separate step for those.

**D9. Do not add the following.** Each item gets its reason:
- Dependabot: fleet policy.
- `.claude/settings.json` and `.kilo/`: application-specific, and only in magic-cli-remote.
- depguard: `archtest` owns import rules.
- `.editorconfig`: no fleet repository has one, and gofmt, `.gitattributes` and markdownlint already cover its job.
- `SECURITY.md`, `CODEOWNERS`, `CONTRIBUTING.md`, `CHANGELOG.md`: no fleet repository has them. They are a fleet-wide decision, named under Open questions.
- A local git identity (F12).

### Consequences

- Good, because an agent opening gobble now gets its gates, dependency rules and record numbering from the repository itself, in the same words as the sibling libraries.
- Good, because 0004-PLAN Phase 4's CI can go green on Linux and macOS. Without D5 it could not.
- Good, because Windows CI and golden transcripts land on an LF-pinned tree, which avoids magic-cli-remote's 0116 failure before it happens.
- Good, because link and numbering checks become one `make` target instead of a throwaway script per phase. The live broken link is fixed.
- Neutral, because `make preflight` now needs Node (`npx`) and Python 3 on the developer's host. Both are on this host. A host without them fails preflight with a clear "not found" message rather than skipping. *(The owner, 2026-10-04: "all my hosts have the required tooling for this.")*
- Neutral, because `scripts/check_records.py` is a fork of magic-cli-remote's file. A fix there must be carried here by hand, as the other copied files already are.
- Bad, because the 0001 REPORT gets a formatting-only edit. D2's normalised-text comparison is the guard against a content change.
- Bad, because `whitespace` and `goimports` are not yet proven to fail. Until the PLAN proves them, they are config, not gates. *(2026-10-04: both have now failed on scratch plants (measurement 5). The PLAN repeats those plants against the committed config.)*

### Confirmation

Run from a scratch clone of the finished branch. Expected results are in comments.

```text
python scripts/check_records.py --next          # 0008 (this record is 0007)
make check-records                              # exit 0
make markdownlint                               # exit 0, "0 error(s)"
make preflight                                  # exit 0 on Windows (Git Bash)
wsl --exec make preflight                       # exit 0 on Ubuntu 24.04 (D5)
git ls-files --eol | grep -v 'i/lf.*attr/text=auto eol=lf'   # only binary-pattern files, or nothing
cmp .markdownlint-cli2.jsonc <go-selfupdate-lib>/.markdownlint-cli2.jsonc    # identical
cmp opencode.json <go-selfupdate-lib>/opencode.json                          # identical (and the four pointer files)
git check-ignore -q scripts/__pycache__/x.pyc   # exit 0
```

Negative checks, each run on a scratch copy and each seen to fail:
- a planted bad relative link fails `make check-records`;
- a planted `*` list item in `README.md` fails `make markdownlint`;
- a planted unused function, unformatted import block, leading blank line in a function body, and ungrouped import each fail `make lint` with `unused`, `gofmt`, `whitespace` and `goimports` named;
- the committed Makefile with D5 applied runs `make clean` under WSL with exit 0, and the pre-D5 copy still fails with Error 127.

An identifier scan over every added or changed file finds no hostname, account name or real-machine path.

## Pros and Cons of the Options

### A — Fleet library scaffold, adapted to a CLI, one record; CI stays in 0004 Phase 4 (chosen)

- Good, because the files are the siblings' own, so a `cmp` against the source is the review.
- Good, because CI keeps one owner. This record only amends it.
- Good, because both live defects (F4, F6) are fixed.
- Neutral, because it adds a seventh record for work that is mostly copying.
- Bad, because 0004-PLAN gains another amendment, in a record that already carries several dated amendments.

### B — Fold into 0004-PLAN

- Good, because 0004-PLAN already owns the toolchain, and no new number is needed. This is the strongest argument for B: Makefile and `.golangci.yml` changes are arguably Phase 0 leftovers, and the skill's same-topic rule says to amend the same number.
- Bad, because agent rules, markdownlint and records tooling are not module architecture. 0004 would decide repository conventions inside a record about packages.
- Bad, because 0004-PLAN is in progress with its own approval sequence. A new phase would be interleaved with Phases 2–5.

### C — Copy magic-cli-remote whole

- Good, because it is the closest application, and gobble already borrows its target names.
- Bad, because Dependabot contradicts the fleet policy that 0004-PLAN Phase 4 step 4 already records.
- Bad, because `.claude/settings.json` allowlists `mcremote` binaries, and the flake ledger and `GNUmakefile` exist for that repository's Flutter and Windows-service legs. None of them fits gobble.

### D — `AGENTS.md` and pointers only

- Good, because it is the smallest change and is docs-only.
- Bad, because F4 stays. 0004-PLAN Phase 4 CI would fail on Linux and macOS at its first recipe.
- Bad, because F2, F3, F5, F6 and F7 stay open, and the agent file would point at checks that do not exist.

### E — Also write `ci.yml` now

- Good, because the scaffold would be complete in one pass.
- Bad, because CI and release would have two owners. Phase 4's build and publish jobs, the reusable-workflow pin, and its immutable-release precondition all live in 0004-PLAN.
- Bad, because CI without the Phase 2 binary has nothing to stamp, so `verify-build-metadata` and `release-dry-run` cannot be wired honestly yet.

## More Information

### Evidence index

| Claim | Source |
|---|---|
| Scaffold file presence and hashes | Measurement 1: SHA-256 of each file in each working tree on 2026-10-04 |
| go-selfupdate-lib's scaffold is go-llmprovider-sdk's, adapted | go-selfupdate-lib `docs/decisions/0001-MADR-scaffold-shared-go-library.md` §4–§5 |
| 29 markdownlint issues in 4 files under the fleet config; 874 with none | Measurement 2: `markdownlint-cli2@0.23.2` on a scratch copy |
| magic-cli-remote's golden tests failed on Windows from CRLF | magic-cli-remote `.gitattributes` header, citing its MADR 0116 P11 |
| GitHub Windows runners default to `core.autocrlf=true` | go-selfupdate-lib and magic-cli-remote `.gitattributes` comments — **[unverified]** here |
| All 72 tracked files LF; `core.autocrlf=false` on this host | Measurement 6: `git ls-files --eol`, `git config --get core.autocrlf` |
| `Makefile:6` and `Makefile:10` hard-code a Windows shell, since `925ef0a` | `git log -S'SHELL := C:/PROGRA~1' -- Makefile` |
| That Makefile fails on Linux with Error 127; a portable `SHELL` passes | Measurement 3: WSL Ubuntu 24.04, GNU Make 4.3, scratch copy |
| No compared sibling Makefile sets `SHELL` | `grep -n '^SHELL'` over the four sibling Makefiles: no match |
| 0004-PLAN Phase 4 runs `make preflight` on ubuntu and macos | `0004-PLAN-go-module-architecture.md` Phase 4 step 1 |
| `check_records.py` gives `0007`, finds the `guides/` link, and fails on a planted link | Measurement 4 |
| `docs/README.md:10` links the missing `guides/` | `docs/README.md:10`; `ls docs` |
| 0004-PLAN step 7 lists 18 linters as "mcplib's linter set"; mcplib enables 20 plus formatters | `0004-PLAN-go-module-architecture.md:135-138`; unified diff of mcplib's and gobble's `.golangci.yml` |
| Added linters clean today; `unused` and `gofmt` caught a plant; `whitespace` did not | Measurement 5 |
| `whitespace` and `goimports` each fail on a specific gofmt-clean plant | Measurement 5, 2026-10-04 addition |
| The D5 conditional `SHELL` works in Git Bash, PowerShell 7 and WSL Ubuntu | Measurement 8: a scratch Makefile run in all three |
| `.editorconfig` is in 0004-PLAN Scope; no compared repository has one | `0004-PLAN-go-module-architecture.md:38`; measurement 1 |
| Sibling CI hygiene steps and pins | go-selfupdate-lib `.github/workflows/ci.yml` |
| Dependabot is off by fleet policy | `0004-PLAN-go-module-architecture.md:379`; go-selfupdate-lib 0001-MADR §8 |
| gobble's import rules are archtest's job | 0004-MADR; `internal/archtest/` |
| No local `user.name` in the four repositories | Measurement 7: `git config --local --get user.name` |
| Current branch is `main`; org rule forbids committing there | `git status`; organisation Git rule ADR 011 |

### Related records

- [0007-PLAN-repository-scaffolding-to-fleet-standard.md](0007-PLAN-repository-scaffolding-to-fleet-standard.md): the execution plan for D1–D8.
- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md) and [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md): the toolchain, CI and release owner. D8 amends the PLAN.
- go-selfupdate-lib `docs/decisions/0001-MADR-scaffold-shared-go-library.md`: the scaffold this record copies.
- go-llmprovider-sdk `docs/decisions/0002-MADR-migrate-llmprovider-from-mcplib.md`, amendments one to five: where that scaffold came from.
- magic-cli-remote MADR 0116 (CRLF goldens), 0145 (Windows make), and PLANs 0175/0180 (`check_records.py`, per that script's docstring).

### Open questions for the plan

These are execution questions. Every decision above is settled.

- **Phase order.** D3 (`.gitattributes`) must precede any golden file, so it lands no later than 0004-PLAN Phase 3. D5 must precede 0004-PLAN Phase 4. The PLAN should place both early.
- **The branch name** for execution, under the org Git rule, for example `chore/0007-repository-scaffolding`. *(Answered 2026-10-04: `feature/0007-scaffolding`; the project is exempt from the Jira key, and the owner commits.)*
- **The exact planted inputs** that make `whitespace` and `goimports` report, since measurement 5's input did not trigger `whitespace`. *(Answered 2026-10-04 by measurement 5's addition. [0007-PLAN](0007-PLAN-repository-scaffolding-to-fleet-standard.md) P5 uses those inputs.)*
- **Not decided here: whether the fleet adopts `SECURITY.md`, `CODEOWNERS`, `CONTRIBUTING.md` or `CHANGELOG.md`.** That belongs to a fleet-wide record, likely in magic-cli-remote. gobble follows when that record exists.

## Observed — execution results (2026-10-04)

[0007-PLAN](0007-PLAN-repository-scaffolding-to-fleet-standard.md) P0–P6 ran, and its Execution record has the evidence. Against this record's claims:

- **F4 and D5 held.** The first `make preflight` ever run on Linux passed, all 13 steps, on its first attempt.
- **F2 and D2 held.** markdownlint reports 0 issues in 6 files. The REPORT edit was formatting only, and a word-identity comparison that was shown to catch a one-word change confirmed it.
- **F5, F6 and D6 held.** `check_records.py --next` prints `0008`, `--check-all` is clean, and a planted link fails it.
- **F7 and D7 held.** `whitespace`, `goimports`, `unused` and `gofmt` each failed on a plant against the committed config.
- **Measurement 6 needs a qualification.** On this host the user-global `core.attributesFile` already forces `eol=lf`. That is why this host's checkout was LF regardless of the repository. The repository's `.gitattributes` matters for hosts and runners without that file, and that was proven with the global file isolated.
- **Identifier scan.** The 0007 change set was scanned for the local account names, the organisation's mail and git-host domains, and user-profile path prefixes. It found none.
- **Commit ownership.** The owner, not the agent, commits and pushes the staged change (0007-PLAN P0).
