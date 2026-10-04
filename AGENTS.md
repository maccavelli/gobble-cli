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
