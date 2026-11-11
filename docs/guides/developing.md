# Developing gobble

How to build gobble, run its tests and gates, add a package, and write a live test. What the
packages are and how they may import each other is in [architecture.md](../architecture.md). Why
they are that way is in the [decision records](../README.md).

## What you need

- **Go 1.27.4.** `go.mod` names it, and `go version` should print `go version go1.27.4 …`.
- **GNU make and bash.** On Windows, run `make` from Git Bash or from PowerShell. The Makefile uses
  Git's bash either way. In PowerShell a bare `bash` is WSL, not Git Bash.
- **Python 3**, for the records and build-metadata checks. The Makefile runs `python3`; set
  `PYTHON=python` where the interpreter has that name.
- **golangci-lint v2.14.0** on `PATH`:
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0`.
- **Node**, for `npx markdownlint-cli2`.

`golint`, `govulncheck`, `staticcheck` and `apidiff` come from the `tool` directives in `go.mod`, so
they need no install.

## Build and run

```sh
go build ./...          # every package
make bin/gobble.exe     # the stamped binary; bin/gobble on Linux and macOS
```

`make bin/gobble` stamps the build with `git describe` and the kind `local`, with cgo off. Shipped
binaries are pure Go, and `make check-cgo-off` refuses a build with cgo on.

```console
$ ./bin/gobble.exe version
gobble 0.0.0-dev+2513638e2cea.dirty
commit: 2513638e2ceaedf58285bc692dc9e69038ebe310
date:   2026-10-07T19:52:04Z
go:     1.27.2 windows/amd64
```

A local build reports its version as `0.0.0-dev`, plus the commit. `version --json` prints the same
fields as one JSON object. `version --identity` prints the line the release workflow checks.

`gobble` with no command is the line session, `gobble -p "…"` answers one prompt, and `gobble acp`
serves the agent on stdio. A prompt needs a model credential: from `--api-key -`, `auth.json`,
`models.json`'s `apiKey` or the provider's variable ([credentials.md](credentials.md)).

## Test

```sh
make test               # go test ./...
make race               # the race detector, with cgo on
```

The race detector needs cgo and a C toolchain. On Windows, run `make race` in WSL.

### Golden transcripts

`acpclient/acptest` records ACP frames, and tests compare them with files under `testdata/`. When a
transcript changes on purpose, rewrite the goldens and review the diff:

```sh
ACPTEST_UPDATE=1 go test ./...
git diff -- '*.golden'
```

A mismatch without the variable fails the test and leaves the actual transcript as an artifact;
run `go test -artifacts -outputdir <dir>` to keep it.

### What the tests may touch

`internal/cli`'s tests reach no network, and keep to their own temporary directories
([0046-MADR](../decisions/0046-MADR-own-background-listings-and-keep-tests-hermetic.md)).

- Its `TestMain` replaces everything that would reach a provider: the model lister, the model facts' listing,
  the browser opener and the sign-ins.
- The tests run behind a guard. `TestMain` starts a local proxy, then runs the test binary again with the proxy
  variables naming it, because the MCP SDK reads them before any `TestMain` can set them. A request for a host
  off the machine gets a 403, and the run fails after the tests, naming each host:

  ```text
  tests tried to reach the network (0046-MADR D4):
    api.openai.com:443 x30
  ```

  `localhost` and loopback addresses are never proxied, so an `httptest` server works as usual.
- Each test sets its own `GOBBLE_HOME`, and writes nothing outside it.

### Suites outside the default run

| Target | What it runs | Where |
| :--- | :--- | :--- |
| `make completion-shells` | Completion scripts in real shells. A missing shell fails the suite. | Windows: Git Bash, `pwsh`, `powershell.exe`. Linux: bash, fish. macOS: zsh, bash, `/bin/bash`. |
| `make probe-conhost` | The line editor in a real Console Host window. It borrows the clipboard, and saves and restores it. | Windows only |

Neither is part of `make preflight` or CI.

## The gates

Before you stage a Go file, run the pre-add check on it:

```console
$ make pre-add-check FILES="internal/archtest/check.go internal/archtest/check_test.go"
…
go-precheck: 2 file(s) clean (gofmt, golint, govulncheck).
```

With no `FILES` it checks every tracked Go file. `GO_PRECHECK_SKIP_VULN=1` skips govulncheck when
you are offline.

Before you commit, run every gate:

```sh
make preflight
```

It ends with `preflight passed`. In order, it runs:

1. cgo off, gofmt drift, and `go mod tidy` drift;
2. the pre-add check on every Go file;
3. `go vet`, then `staticcheck` for linux, darwin and windows;
4. `golangci-lint`, `govulncheck`, and `go fix -diff`;
5. `archtest`, the import rules;
6. `apidiff`, a no-op below `v1.0.0`;
7. `verify-build-metadata`, which builds `cmd/gobble` stamped and unstamped;
8. `check-records`, the records' names, pairs and links;
9. `check-icon`, which checks that `assets/gobble.svg` is what its generator writes;
10. `markdownlint`.

Any one of them fails the run. Each is also a target of its own (`make archtest`, `make lint`, …),
for a quicker loop.

## Add a package

1. **Choose its tier.**
   - A public package starts as `beta`. The `v1.0.0` release gate promotes packages to `stable`
     one by one.
   - A package only gobble uses goes under `internal/`.
   - Unstable work goes under `exp/`. Only `cmd/gobble` may import it.
2. **Write its `doc.go`.** The package comment's first sentence says what it is. The next line
   states the tier:

   ```go
   // Package widget is an example package from the developing guide.
   // Stability: beta
   package widget
   ```

3. **Check the import rules.** `make archtest` enforces them; architecture.md lists them. The
   protocol SDKs, the provider SDK, go-selfupdate-lib, Kong, Charm and go-tui-lib each have one
   permitted home, so a new package that imports one fails:

   ```text
   --- FAIL: TestImportRules/rule_9_module (0.00s)
       check_test.go:139: unexpected edge:
           rule 9: github.com/maccavelli/gobble-cli/widget imports github.com/alecthomas/kong
   ```

   A new permitted edge means amending
   [0004-MADR](../decisions/0004-MADR-go-module-architecture.md) and its rule in
   `internal/archtest/check.go` together, never editing the test alone.
4. **Add its row** to the package table in architecture.md, with the same tier as its `doc.go`.
5. **A new module dependency needs a decision record.** No module is required unless a MADR in
   this repository names it. `go.mod` and `go.sum` change in the commit that adds the first import.

## Live tests

A test that calls a real model provider is a live test. `llm/provider` has one for each of
`openai`, `claude`, `gemini` and `grok`, and default CI runs none. The convention, from `llm/llmtest`:

- the file carries the build tag `live_<provider>`, gobble's provider id, such as `live_claude`;
- the test reads its credential with `llmtest.LiveCredential`, which skips the test when the
  provider's usual variable is unset:

  ```go
  //go:build live_claude

  key := llmtest.LiveCredential(t, "ANTHROPIC_API_KEY")
  ```

- you run it with `go test -tags live_claude ./llm/provider/`.

Never commit a credential or put one on a command line. Export it in your shell.

## CI and the release rehearsal

`.github/workflows/ci.yml` runs on every push to `main`, on every pull request, and on `v*` tags.
A push or pull request whose changes are all in the docs set (`*.md` at the root, `docs/**/*.md`,
`.claude/rules/*.md`, `.grok/rules/*.md` and `.opencode/rules.md`) skips it, and
`.github/workflows/docs.yml` runs `make check-records` and `make markdownlint` on one Linux runner
instead; a Markdown file Go reads, such as a built-in tool's description, runs the full CI, and a `v*`
tag is never filtered. Its jobs:

- `validate` runs `go test`, `make preflight` and `make archtest` on Linux, macOS and Windows, and
  `make race` on Linux and macOS;
- `sbom` saves the dependency graph's SPDX SBOM;
- `build` builds the five release binaries and runs `gobble version --identity` on each platform.
  Without a tag this is a rehearsal, stamped `rehearsal-<commit>`, so a docs-only push or pull
  request has none;
- `release` publishes a GitHub release, only on a `v*` tag at `main`'s head.

## Releasing

### What GitHub protects

No rule has a bypass actor, so a rule binds everyone, the owner and an agent under the owner's login
included. [0043-MADR](../decisions/0043-MADR-protect-main-and-acpext-tags-without-bypass.md) has the
reasons.

| Ref | Protected by | Refused |
| :--- | :--- | :--- |
| `main` | the ruleset `gobble-cli-main` | a force-push, and deleting it |
| `acpext/vX.Y.Z` tags | the ruleset `gobble-cli-acpext-tags` | moving the tag, to any commit, and deleting it |
| a `v*` tag with a published release | immutable releases, a repository setting | moving the tag, and deleting it |
| a `v*` tag with no published release | nothing | — |

A fast-forward push to `main` passes. Creating a tag fires no rule: neither ruleset has a `creation`
rule, and there is no `v*` ruleset.

### Tagging

- **A gobble release** is an annotated `vX.Y.Z` tag on `main`'s head, pushed on the owner's word. The
  `release` job publishes it.
- **An acpext version** is an annotated `acpext/vX.Y.Z` tag, pushed before the root requires it, in
  the order [0042-MADR](../decisions/0042-MADR-acpext-extension-contract-module.md) D5 gives.

### When a release goes wrong

- **A `v*` tag whose release was never published** may be deleted and pushed again on the fixed
  commit, but only if the module proxy does not list that version:

  ```bash
  curl -fsS https://proxy.golang.org/github.com/maccavelli/gobble-cli/@v/list
  ```

  Ask for `@v/list` alone. Asking for the version's `.info`, `.mod` or `.zip` can make the proxy
  fetch it, and from then on that version is fixed.
- **If the proxy lists it,** the tag stays, and the fix is the next patch version. `v0.1.0` is
  listed, so it stays for good.
- **A tag whose release was published** stays, since the release is immutable. The fix is the next
  patch version.
- **A wrong acpext tag** stays, because the ruleset refuses moving or deleting it. The fix is the next
  patch version, with a `retract` in `acpext/go.mod` if the bad version must not be used.

### After each `v*` release

- The tag push prints no "Bypassed rule violations" line.
- No rule suite for the tag records a bypass:

  ```bash
  gh api 'repos/maccavelli/gobble-cli/rulesets/rule-suites?ref=refs/tags/vX.Y.Z&time_period=day' \
    --jq '[.[] | select(.result == "bypass")] | length'    # 0
  ```

- The release is immutable: `gh release view vX.Y.Z --json isImmutable` gives `true`.
- `gobble update` finds it. From the previous release's binary, installed under your home directory,
  `gobble update --check` exits 10 and names the new tag. On Windows, start `gobble acp` from that binary and
  leave it running, run `gobble update --yes` from the same file, stop `gobble acp`, then run a plain
  `gobble update` from the installed binary: it reports the release up to date and exits 0, and no
  `.gobble.exe.selfupdate-*` file is left beside it. A plain run, not `--check`, because only a run that may
  install finishes a leftover. Both need the previous release to have `update`, so they start with the release after the
  first one that carries it ([0048-MADR](../decisions/0048-MADR-gobble-update-command.md)).

### When the tag rule comes back

go-selfupdate-lib's release guide asks to restrict who can create `v*` tags. gobble-cli doesn't,
because one account can write and it is the owner's, and no workflow pushes a tag that starts a
release. If a second account ever gets write access, or a workflow gets a token that can push tags and
start runs, that advice applies again, and restricting `v*` creation needs a new record.

## Change the icon

gobble's icon, a cartoon turkey, is `assets/gobble.svg`. `scripts/gen_icon.py` generates it, and it
is never edited by hand: `make check-icon`, part of preflight, fails when the two differ.

1. Edit the constants at the top of `scripts/gen_icon.py`.
2. Write the SVG and its previews. This needs `rsvg-convert` and ImageMagick's `magick`:

   ```sh
   python3 scripts/gen_icon.py --preview /tmp/gobble-icon
   ```

3. Look at `gobble-24-x.png` and `gobble-48-x.png`, the small sizes enlarged. The icon is one
   colour, so its detail (the eye, the wing, the stripe in the tail) must stay visible as gaps.
4. Commit the script and `assets/gobble.svg` together.

Consumers fetch the file at a `v*` tag, so a change reaches them at the next release. Moving or
renaming it breaks them, and needs a record that says so
([0044-MADR](../decisions/0044-MADR-publish-gobbles-icon-and-its-generator.md), D4).

## Decision records

Mutating work follows a MADR and its PLAN in `docs/decisions/`. `AGENTS.md` has the workflow.

- `python3 scripts/check_records.py --next` prints the next free number.
- A follow-up to existing work amends that record with a dated entry; it does not rewrite the
  record's history.
- [docs/README.md](../README.md) indexes every record by hand. Update it in the same change.
