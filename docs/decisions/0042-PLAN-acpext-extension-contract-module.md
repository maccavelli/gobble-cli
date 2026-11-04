---
status: proposed
date: 2026-11-03
associated-madr: "0042-MADR-acpext-extension-contract-module.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement publishing gobble's ACP extension contract as the nested module `acpext`, which gobble serves from and magic-cli-remote imports

Associated MADR: [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md)

## Goal

The module `github.com/maccavelli/gobble-cli/acpext` exists, is tagged `acpext/v0.1.0`, and holds gobble's
`_gobble/*` names and shapes, the `_meta.gobble` advertisement, and the auth methods and error data (MADR D1 to D3).

gobble's `acpserver` and `acpclient` then use it, with no change on the wire (D4). The plan is complete when the
MADR's Confirmation 1 to 7 is met.

## Scope

**Base.** `main` at `67f72fb`, acp-go-sdk2 `v1.31.2`, Go 1.27.3.

**In scope:**

* `acpext/` (new): `go.mod`, `go.sum`, `LICENSE`, `doc.go`, `methods.go`, `types.go`, `auth.go`, their tests and
  `testdata/`;
* `Makefile`, `.github/workflows/ci.yml` and `.gitignore`;
* the root `go.mod` and `go.sum`, for the one requirement;
* `acpserver/`: `extensions.go`, `autocompact.go`, `compact.go`, `prompt.go`, `personality.go`, `auth.go` and
  `agent.go`, for their types and constants only;
* `acpclient/update.go`, `sessions.go` and `conn.go`, likewise;
* `internal/cli/agent.go`'s `authProviders`, and a new root test beside it;
* `internal/archtest/check.go` and its test;
* AGENTS.md, `docs/architecture.md`, a dated note on `docs/decisions/0004-MADR-go-module-architecture.md`, this pair
  and `docs/README.md`.

**Out of scope** (the MADR's "Not decided here"):

* moving `acpserver`'s dispatch onto `acp.Handle`;
* any new method, field or value;
* the other `_meta.gobble` members;
* a gobble release. It is the owner's, after P2, and magic-cli-remote's 0179 waits for it.

## Conventions every phase follows

0039-PLAN's C1 to C11 apply, as written there:

* the start state, and line numbers re-read;
* scratch work, and long output to a log;
* G-local: `make pre-add-check`, `make preflight` and `make race`;
* G-hosts: the Linux machine and the Windows laptop. P1 and P2 change the Makefile and the module graph, so both
  phases run them;
* deviations stopped and recorded;
* every new test and gate seen failing on a broken copy;
* `git commit --no-edit` by the owner;
* no identifier of a real machine in a committed file.

Two more for this plan:

* **C12. Pushes and tags.** P1's push and its tag `acpext/v0.1.0` are the owner's, each on an explicit word, after
  the disclosure guard has run over the outgoing commits. P2 does not start until the tag is on the remote and the
  module proxy serves it.
* **C13. No `replace` and no `go.work`.** Neither is ever committed (MADR D5). A local `go.work`, ignored by
  `.gitignore`, may be used on a scratch copy only.

## Implementation Steps

### P0 — the records

Files: this pair; `docs/decisions/0004-MADR-go-module-architecture.md`; `docs/README.md`.

1. **0004-MADR** gains a dated italic note after its Decision Outcome. It says that 0042-MADR adds the nested module
   `acpext/` for the extension contract alone, that the root is still one module, and that its Confirmation's `go list
   -m` is unchanged. Its text above the note is unchanged.
2. `docs/README.md`'s two 0042 rows give the pair's status.
3. `make check-records markdownlint` passes.
4. The identifier scan over the staged records finds nothing.
5. Staged; the owner commits.

### P1 — the module (MADR D1 to D3, D5, D6)

**Gate:** P0 committed.

Files: `acpext/` (all of it), `Makefile`, `.github/workflows/ci.yml`, `.gitignore`, this plan, `docs/README.md`.

1. **The facts,** re-read; a difference is a deviation:
   * the twenty-entry `extensions` table, and each param and result struct it decodes or returns, with its fields,
     JSON names and options (`acpserver/extensions.go`, `personality.go`);
   * the three notification constants and params (`prompt.go`, `autocompact.go`, `personality.go`), and every value
     of their phases, reasons, pauses and causes;
   * the queue modes and dispositions (`acpserver/queue.go`, `extensions.go`);
   * `capabilitiesMeta`, `AuthProvider`, `loginSuffix`, and the error-data keys and their three uses;
   * what `internal/cli/agent.go`'s `authProviders` returns today, run once and recorded in the record.
2. **`acpext/go.mod`:** `module github.com/maccavelli/gobble-cli/acpext`, `go 1.27.3`, and `require
   github.com/maccavelli/acp-go-sdk2 v1.31.2`. Then `go -C acpext mod tidy`, and `acpext/LICENSE` copied from the
   root.
3. **The package** (D2, D3):
   * `doc.go`: the package doc. It says what acpext holds, that gobble serves it, that the `jsontext.Value` fields are
     gobble's session-file shapes and not frozen, and that a change is a new minor tag (D5);
   * `methods.go`: the twenty-three `Method…` constants, `Requests()` and `Notifications()`;
   * `types.go`: `MetaKey`, `CapabilitiesMeta`, each params and result type, the typed string constants, and
     `ErrorData`;
   * `auth.go`: `AuthProvider`, `AuthProviders()` and `LoginSuffix`.

   The record gets a table of each acpext type against the source line it copies. Every exported name has a doc
   comment; `golint` passes.
4. **The tests,** in `acpext`:
   * **Round trip.** `testdata/` holds one frame per method, copied from what gobble sends today. The source is
     `acpserver`'s and `acpclient`'s test expectations and `acpserver/testdata/session.golden`. A frame no test
     records yet is captured once from `gobble acp` at the base, with its identifiers replaced.

     Each frame decodes into its acpext type and encodes back to the same JSON, compared as values. The test runs
     with `encoding/json` and with `encoding/json/v2`.
   * **Lists.** `Requests()` and `Notifications()` are sorted, distinct, and each name starts `_gobble/`.
     `AuthProviders()` matches the list recorded in step 1.

   **Seen failing (C9),** each on its own scratch copy:
   * a JSON name changed (`name` to `title` in `SetSessionNameParams`);
   * an `omitzero` dropped from `RetryNotice.Attempt`;
   * a request missing from `Requests()`;
   * a provider dropped from `AuthProviders()`.
5. **`check-acpext-deps`** (D6), a Makefile target in `preflight`. It runs `go -C acpext list -deps -f '{{if not
   .Standard}}{{.ImportPath}}{{end}}' ./...` and fails on any path outside acp-go-sdk2 and acpext. It is seen failing
   on a scratch copy importing `github.com/maccavelli/gobble-cli/jsonx`, which also fails acpext's build, and
   `golang.org/x/sys/unix`, which does not.
6. **Per-module gates** (D6). The Makefile names `MODULES := . acpext`. `preflight`'s tidy check, vet, staticcheck,
   golangci-lint, govulncheck and fix-check, and the `test` and `race` targets, each run once per module, in its
   directory.

   They are seen failing on scratch copies:
   * an untidy `acpext/go.mod`, with an unused requirement;
   * a vet error in an acpext file;
   * a failing acpext test.

   Each fails `make preflight` or `make test`, and passes with the root alone.
7. **CI.** The `validate` job's `go mod download`, `go test` and `go mod tidy -diff` steps run per module.
   `make preflight`, `make archtest` and `make race` already do. `actionlint` passes.
8. **`.gitignore`** gains `go.work` and `go.work.sum`.
9. **Nothing in the root changes:** `git status` lists only this phase's files, and `go list -m` in the root still
   prints one module.
10. G-local, then G-hosts. **Commit.**
11. **The disclosure guard** over the outgoing commits, with its output recorded. Then, on the owner's word only (C12),
    the push and the tag `acpext/v0.1.0` on P1's commit. The record notes when
    `go list -m github.com/maccavelli/gobble-cli/acpext@v0.1.0` succeeded through the module proxy.

### P2 — gobble serves and reads from acpext (MADR D4, D6)

**Gate:** `acpext/v0.1.0` served by the module proxy (P1 step 11).

Files: the root `go.mod` and `go.sum`; the `acpserver`, `acpclient`, `internal/cli` and `internal/archtest` files in
Scope; AGENTS.md; `docs/architecture.md`; this plan; `docs/README.md`.

1. **The facts,** re-read: the files and lines of P1 step 1 are unchanged since P1, and acpext's tag is P1's commit.
2. **`go get github.com/maccavelli/gobble-cli/acpext@v0.1.0`.** The root `go.mod` gains one requirement and no
   `replace`. `go mod tidy` is clean.
3. **`acpserver`:**
   * its private param, result and notice structs, method constants, `capabilitiesMeta` and `loginSuffix` are deleted,
     and their uses name acpext's;
   * `AuthProvider` becomes acpext's, with no alias left in `acpserver`;
   * the error data is built from `acpext.ErrorData`, still through `jsonx` (0039-MADR D14).

   Encoding and decoding stay `jsonx`'s.
4. **`acpclient`:** the three constants and the params decoding in `update.go`, and `setNameParams` in `sessions.go`,
   name acpext's. Its exported `Update`, `Retry`, `Compaction` and `Personality` types are unchanged; `go doc
   ./acpclient` is diffed against the base and shows no change.
5. **`internal/cli`:** `authProviders` returns `[]acpext.AuthProvider`.
6. **The root test** (D4), `internal/cli/acpext_test.go`. It asserts:
   * the `extensions` table's keys equal `acpext.Requests()`, through an exported-for-test accessor or `Capabilities()`
     `_meta`, whichever P2 step 1 finds with no new exported API;
   * every method `acpserver` notifies is in `acpext.Notifications()`;
   * `acpext.AuthProviders()` equals `authProviders()`.

   **Seen failing (C9),** each on a scratch copy:
   * a twenty-first `_gobble/` entry in the table only;
   * `acpext.AuthProviders()` with its last provider dropped, by a local `go.work` on the scratch copy (C13);
   * a notify of `"_gobble/other"`.
7. **archtest** (D4). The new rule is that only `acpserver`, `acpclient` and `internal/cli` import
   `github.com/maccavelli/gobble-cli/acpext`. It is seen failing on a scratch copy importing it from `agent/`.
8. **The wire is unchanged** (Confirmation 4). `acpserver/testdata/session.golden` and every test expectation in
   `acpserver` and `acpclient` are unchanged: `git diff --stat` lists no testdata and no `_test.go` change but step 6's
   new file and step 7's test. `go test ./...` and `go test -race ./...` pass.
9. **Assert by search:** outside acpext and the tests, no non-test Go file holds a `"_gobble/` string literal.
10. **Documents.** AGENTS.md's module sentence names acpext, its tag form and the no-`replace` rule.
    `docs/architecture.md` adds acpext to the module and package map.
11. G-local, then G-hosts. **Commit.**

### P3 — acceptance and close-out (MADR Confirmation)

**Gate:** P2 committed and pushed, on the owner's word (C12).

Files: this plan, the MADR, `docs/README.md`.

1. **`go install github.com/maccavelli/gobble-cli/cmd/gobble@<P2's commit>`** with an empty `GOMODCACHE` and
   `GOFLAGS=-mod=mod`, in a scratch directory. It succeeds, and `go version -m` on the binary names
   `github.com/maccavelli/gobble-cli/acpext v0.1.0` (Confirmation 3).
2. **A scratch consumer module** outside the tree requires `acpext@v0.1.0` from the proxy. It decodes a recorded
   `_gobble/retry` frame into `RetryNotice`, and `go list -m all` names no go-llmprovider-sdk (Confirmation 7).
3. **`go -C acpext list -m all`** is recorded (Confirmation 1).
4. **The MADR's Confirmation 1 to 7,** each with its evidence.
5. **A note for magic-cli-remote,** written into the record for the owner to pass on. It names the tag and the
   types its 0179 D13 uses.
6. **Status.** This plan becomes `complete`; the MADR becomes `accepted` if it was not already.

## Verification

| MADR Confirmation | Phase | Check |
| :--- | :--- | :--- |
| 1 acpext's graph is the SDK's | P3 | step 3 |
| 2 `check-acpext-deps`, seen failing | P1 | step 5 |
| 3 required by tag; `go install` works | P2, P3 | P2 step 2; P3 step 1 |
| 4 the wire is unchanged | P2 | step 8 |
| 5 the root test, seen failing | P2 | step 6 |
| 6 archtest's rule, seen failing | P2 | step 7 |
| 7 a consumer gets no go-llmprovider-sdk | P3 | step 2 |

## Rollout and Rollback

* **P1 adds a module nothing uses yet.** Reverting P1 before its tag is pushed removes it. After the push, the tag
  stays, since a published version cannot be withdrawn from the proxy. A broken `v0.1.0` is fixed by `v0.1.1`, with a
  `retract` in acpext's `go.mod` if it must not be used.
* **P2 is one commit.** Reverting it puts back `acpserver`'s and `acpclient`'s own types, and drops the requirement.
  The wire is the same on both sides of the revert.
* **User-visible:** none. No command, output, configuration or wire byte changes.
* **For importers:** magic-cli-remote can require `acpext@v0.1.0` from P1 step 11. Its types then match the wire of
  every gobble built from P2 on.

## Execution record

Each phase appends its record here.
