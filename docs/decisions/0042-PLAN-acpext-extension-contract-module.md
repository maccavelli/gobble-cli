---
status: complete
date: 2026-11-06
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

*(Re-read 2026-11-04 at `9e667c4`, before approval:*

* *None of the source files in scope has changed since `67f72fb`. `git diff --stat 67f72fb..HEAD` over them is
  empty, and the `extensions` table still has twenty requests.*
* *go-llmprovider-sdk `v1.26.0`, which 0035-PLAN P30 would pin, has the same registry as `v1.25.0`: `providers.go`
  is unchanged, and so are the provider ids. So the nine providers of MADR D3 stand whichever lands first.*
* *0035-PLAN P29 changes `acpserver/agent_test.go` and `acpserver/auth_test.go`. If it lands first, P2 starts from
  its commit, as C1 has it.*
* *Six existing test files name the identifiers P2 moves. They are added to the scope below, and P2 step 8 holds
  them to renames.)*

*(Re-read 2026-11-06 at `08439b4`, before approval, with the MADR's amendment of 2026-11-06 (A1 to A4):*

* *The root pins acp-go-sdk2 `v1.34.0` and go-llmprovider-sdk `v1.26.1` (0035-PLAN P29, P30). acpext requires
  neither (A1). `authProviders()`, run on a scratch copy, still gives the MADR's nine providers.*
* *The source files in scope are unchanged since `67f72fb` but for `acpserver/agent_test.go` and
  `acpserver/auth_test.go`, which P29 changed; P2 starts from `HEAD`.)*

**In scope:**

* `acpext/` (new): `go.mod`, ~~`go.sum`,~~ `LICENSE`, `doc.go`, `methods.go`, `types.go`, `auth.go`, their tests,
  `example_test.go` and `testdata/` *(amended 2026-11-06: no `go.sum`, A1)*;
* `Makefile`, `.github/workflows/ci.yml` and `.gitignore`;
* the root `go.mod` and `go.sum`, for the one requirement;
* `acpserver/`: `extensions.go`, `autocompact.go`, `compact.go`, `prompt.go`, `personality.go`, `auth.go` and
  `agent.go`, for their types and constants only;
* `acpclient/update.go`, `sessions.go` and `conn.go`, likewise;
* `internal/cli/agent.go`'s `authProviders`, and a new root test beside it;
* *(2026-11-04)* the identifier renames only, in `acpserver/agent_test.go`, `acpserver/autocompact_test.go`,
  `acpserver/personality_switch_test.go`, `acpserver/prompt_test.go`, `acpserver/retry_test.go` and
  `acpclient/sessions_test.go`;
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

Files: `acpext/` (all of it), `Makefile`, `.github/workflows/ci.yml`, `.gitignore`, this plan, `docs/README.md`;
*(deviation 2)* `.golangci.yml` and the MADR.

1. **The facts,** re-read; a difference is a deviation:
   * the twenty-entry `extensions` table, and each param and result struct it decodes or returns, with its fields,
     JSON names and options (`acpserver/extensions.go`, `personality.go`);
   * the three notification constants and params (`prompt.go`, `autocompact.go`, `personality.go`), and every value
     of their phases, reasons, pauses and causes;
   * the queue modes and dispositions (`acpserver/queue.go`, `extensions.go`);
   * `capabilitiesMeta`, `AuthProvider`, `loginSuffix`, and the error-data keys and their three uses;
   * what `internal/cli/agent.go`'s `authProviders` returns today, run once and recorded in the record;
   * *(2026-11-06)* every key of every error's data, with its uses (MADR A3: `sessionId` among them), and every
     value a retry's cause takes (`agent/event.go`, `llm/provider.FailureKind`).
2. **`acpext/go.mod`:** `module github.com/maccavelli/gobble-cli/acpext` and `go 1.27.3`, ~~and `require
   github.com/maccavelli/acp-go-sdk2 v1.31.2`~~ and no requirement *(amended 2026-11-06, A1)*. Then `go -C acpext mod
   tidy`, which writes no `go.sum`, and `acpext/LICENSE` copied from the root.
3. **The package** (D2, D3):
   * `doc.go`: the package doc. It says what acpext holds, that gobble serves it, that the `jsontext.Value` fields are
     gobble's session-file shapes and not frozen, and that a change is a new minor tag (D5);
   * `methods.go`: the twenty-three `Method…` constants, `Requests()` and `Notifications()`;
   * `types.go`: `MetaKey`, `CapabilitiesMeta`, each params and result type, the typed string constants, and
     `ErrorData`;
   * `auth.go`: `AuthProvider`, `AuthProviders()` and `LoginSuffix`.

   The record gets a table of each acpext type against the source line it copies. Every exported name has a doc
   comment; `golint` passes.

   *(Amended 2026-11-06, the MADR's A1 to A3):*
   * *every `SessionID` field is `string`; nothing imports acp-go-sdk2;*
   * *every type is flat, its shared shapes named as A2 names them;*
   * *`CloneResult` is `{SessionID}`, and the nested result types A3 lists are exported;*
   * *`ErrorData` is `{Command, Error, Provider, Reason, SessionID}`, declared in that order, each `omitzero`;*
   * *the typed constants are `Phase`, `CompactionReason`, `CompactionPause`, `RetryCause` (A3's twelve values,
     documented as open), `QueueMode` and `Disposition`, and each field that holds one has its type;*
   * *`doc.go` has a table of each method against its params and result types, and says how a client reads
     `CapabilitiesMeta` with `acp.DecodeMeta`;*
   * *`example_test.go` holds `Example` functions that decode a `_gobble/retry` frame and build a
     `SetSessionNameParams`, with `encoding/json`, for pkg.go.dev.*
4. **The tests,** in `acpext`:
   * **Round trip.** `testdata/` holds one frame per method, copied from what gobble sends today. The source is
     `acpserver`'s and `acpclient`'s test expectations and `acpserver/testdata/session.golden`. A frame no test
     records yet is captured once from `gobble acp` at the base, with its identifiers replaced.
     *(Deviation 1, 2026-11-06: captured from `acpserver`'s own test harness, not the
     binary.)*

     Each frame decodes into its acpext type and encodes back to the same JSON, compared as values. The test runs
     with `encoding/json` and with `encoding/json/v2`. *(Amended 2026-11-06, MADR Confirmation 8: each decode
     rejects unknown members, `DisallowUnknownFields` and `RejectUnknownMembers(true)`, as magic-cli-remote's D11
     item 9 does; the comparison is of the two values after `jsontext.Value.Canonicalize`.)*
   * **Lists.** `Requests()` and `Notifications()` are sorted, distinct, and each name starts `_gobble/`.
     `AuthProviders()` matches the list recorded in step 1.

   **Seen failing (C9),** each on its own scratch copy:
   * a JSON name changed (`name` to `title` in `SetSessionNameParams`);
   * an `omitzero` dropped from `RetryNotice.Attempt`;
   * a request missing from `Requests()`;
   * a provider dropped from `AuthProviders()`;
   * *(2026-11-06)* a member added to the recorded retry frame, which the strict decode refuses;
   * *(2026-11-06)* `SessionID` dropped from `ErrorData`, against a recorded `-32602` refusal's data.
5. **`check-acpext-deps`** (D6), a Makefile target in `preflight`. It runs `go -C acpext list -deps -f '{{if not
   .Standard}}{{.ImportPath}}{{end}}' ./...` and fails on any path outside ~~acp-go-sdk2 and~~ acpext *(amended
   2026-11-06, A1)*. It is seen failing on a scratch copy importing `github.com/maccavelli/gobble-cli/jsonx`, which
   also fails acpext's build, ~~and~~ `golang.org/x/sys/unix`, which does not, and *(2026-11-06)*
   `github.com/maccavelli/acp-go-sdk2`.
6. **Per-module gates** (D6). The Makefile names `MODULES := . acpext`. `preflight`'s tidy check, vet, staticcheck,
   golangci-lint, govulncheck and fix-check, and the `test` and `race` targets, each run once per module, in its
   directory.

   *(Amended 2026-11-06, MADR A3.)* The tools are the root's, run against `acpext/`: `go tool govulncheck -C acpext
   ./...`; `bin/tools/staticcheck`, per OS, in `acpext/`; `go tool golint` over acpext's files; golangci-lint in
   `acpext/` with the root's configuration; and `go -C acpext` for `vet`, `test`, `fix -diff` and `mod tidy -diff`.
   `acpext/go.mod` gains no `tool` directive.

   They are seen failing on scratch copies:
   * an untidy `acpext/go.mod`, with an unused requirement;
   * a vet error in an acpext file;
   * a failing acpext test.

   Each fails `make preflight` or `make test`, and passes with the root alone.
7. **CI.** The `validate` job's `go mod download`, `go test` and `go mod tidy -diff` steps run per module. acpext has
   no `go.sum`, so the setup's cache key is unchanged *(2026-11-06, A1)*.
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
Scope, the six test files of its 2026-11-04 item among them; AGENTS.md; `docs/architecture.md`; this plan;
`docs/README.md`.

1. **The facts,** re-read: the files and lines of P1 step 1 are unchanged since P1, and acpext's tag is P1's commit.
2. **`go get github.com/maccavelli/gobble-cli/acpext@v0.1.0`.** The root `go.mod` gains one requirement and no
   `replace`. `go mod tidy` is clean.
3. **`acpserver`:**
   * its private param, result and notice structs, method constants, `capabilitiesMeta` and `loginSuffix` are deleted,
     and their uses name acpext's;
   * `AuthProvider` becomes acpext's, with no alias left in `acpserver`;
   * the error data is built from `acpext.ErrorData`, still through `jsonx` (0039-MADR D14);
   * *(2026-11-06, MADR A1 and A2)* `extSession` decodes the params into their acpext type, then into
     `acpext.SessionParams` for the session, converting it to `acp.SessionId`; the first decode's error is the one
     answered. Each notification and result sets `SessionID: string(id)`.

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
   *(Amended 2026-11-06, MADR A4: the path is matched as a module of its own before `relOf` maps a root package,
   so acpext is never read as a root package; a scratch break of that order is seen failing too.)*
8. **The wire is unchanged** (Confirmation 4). `acpserver/testdata/session.golden` and every test expectation in
   `acpserver` and `acpclient` are unchanged: ~~`git diff --stat` lists no testdata and no `_test.go` change but step
   6's new file and step 7's test~~ `git diff --stat` lists no testdata change. The `_test.go` changes are step 6's
   new file, step 7's test, and the six files' renames. *(Amended 2026-11-04, before approval: those six files name
   the moved identifiers: `AuthProvider` at `acpserver/agent_test.go:55-56`; `setNameParams` at
   `acpclient/sessions_test.go:35`; `methodCompaction`, `methodPersonality`, `methodRetry`, `retryNotice`,
   `compactionNotice`, `personalityNotice` and the phase, reason and pause constants in
   `acpserver/autocompact_test.go:113`, `personality_switch_test.go:68-248`, `prompt_test.go:466-469` and
   `retry_test.go:80-84`.)*
   * **A rename check,** a scratch Python script, holds them to renames. It maps each acpext name in a changed test
     line back to the identifier it replaced, and fails unless every changed line then equals its base line. It is
     seen failing on a scratch copy where one renamed line also changes an expected value (C9). *(Amended
     2026-11-06, A1: the one other change it accepts is a session id's `string(…)` conversion in a `SessionID`
     field, as at `personality_switch_test.go:143`.)*
   * `go test ./...` and `go test -race ./...` pass.
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
   *(2026-11-06, A1: it also names no acp-go-sdk2, unless the scratch module requires it itself.)*
3. **`go -C acpext list -m all`** is recorded (Confirmation 1).
4. **The MADR's Confirmation 1 to 7,** each with its evidence.
5. **A note for magic-cli-remote,** written into the record for the owner to pass on. It names the tag and the
   types its 0179 D13 uses. *(2026-11-06)* It also says that `ErrorData` has `sessionId`, that acpext requires no
   acp-go-sdk2, so its D12 gate 6 holds by construction, and that its D12 gate 3's pin is `v1.34.0` in both.
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
| 8 strict round trip, seen failing *(2026-11-06)* | P1 | step 4 |

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

*Approved 2026-11-06 by the owner's "Commit to main and do p0", after the records commit `0c07a72`, the pair as
amended on 2026-11-04 and 2026-11-06. The MADR is `accepted`; this plan is `in-progress`.*

### P0 record (2026-11-06): the records

The base is `0c07a72`, the amendment's commit. No deviation.

1. **0004-MADR** has a dated italic note after its Decision Outcome's opening paragraph, before `### Module`. It
   says that 0042-MADR adds the nested module `acpext/` for the extension contract alone, importing only the
   standard library (A1), that the root is still one module, and that its Confirmation's `go list -m` is
   unchanged. The text above it is unchanged: the diff adds lines only.
2. **`docs/README.md`**'s two 0042 rows give the pair's status: the MADR accepted, this plan in progress, P0
   complete.
3. `make check-records markdownlint` passed, 0 issues.
4. The identifier scan over the staged records found nothing.
5. Committed on `main` on the owner's word in the same turn.

### Deviation 1 (2026-11-06): P1 step 4's frames, captured through the harness

- **Found** at P1 step 4, before any acpext file was written (`main` at `8632eeb`). Step 4 says a frame no test
  records yet "is captured once from `gobble acp` at the base". `acpserver/testdata/session.golden` records only
  `initialize`'s `_meta.gobble` and one `_gobble/personality`, and no test keeps a result's raw bytes: the extension
  tests decode each into a map (`acpserver/extensions_test.go:92-104`). So nearly every frame needs capturing. The
  `gobble acp` binary cannot give `_gobble/compact`'s or `_gobble/usage`'s without a live model and the owner's key,
  and its ids, timestamps and token counts differ on every run.
- **Decision,** the owner's ("Harness capture"): a scratch `_test.go`, on a scratch copy and never committed, drives
  `acpserver`'s own `Agent` over `acptest`, with `llmtest`'s scripted model, as the extension tests do. It records:
  - each result's raw bytes, as `ClientSideConnection.CallExtension` returns them;
  - each notification's raw params (`acptest.Notice.Params`);
  - each refusal's `data`.

  It is the same server code and the same encoder as the binary's, with no key, and it is repeatable. The params
  frames, which gobble receives and does not send, are written from `acpserver`'s own JSON tags. Ids and paths of
  the scratch run are replaced with fixed values, as step 4 already asks.
- **Files added to scope:** none. The capture test is scratch only. The MADR does not change.

### Deviation 2 (2026-11-06): the root's JSON rule and acpext's tests

- **Found** at P1 step 6, running golangci-lint in `acpext/` with the root's `.golangci.yml`, as MADR A3 has it:

  ```text
  acpext/example_test.go:4:2: import 'encoding/json' is not allowed from list 'json': use jsonx, on encoding/json/v2 (0039-MADR D24) (depguard)
  acpext/roundtrip_test.go:5:2: import 'encoding/json' is not allowed from list 'json': use jsonx, on encoding/json/v2 (0039-MADR D24) (depguard)
  ```

  New with acpext; no root file imports `encoding/json`. acpext's tests need it: Confirmation 8's round trip is
  under both encoders, and the examples show the encoder an importer (magic-cli-remote, the SDK) uses. acpext cannot
  import `jsonx` (MADR A1). `forbidigo`'s `encoding/json/v2` rule is already excluded for `_test.go` files.
- **Decision,** the owner's ("Exclude acpext tests"): one exclusion in the root `.golangci.yml`, `depguard` on
  `^acpext/.*_test\.go$` for the text `0039-MADR D24`, with a comment citing this deviation. acpext's own code stays
  under the rule. MADR A3 has a note.
- **Files added to scope:** `.golangci.yml`, and the MADR for A3's note.

### P1 record (2026-11-06): the module

The base is `8632eeb`, P0's commit. Deviations 1 and 2, above.

1. **The facts** (step 1), re-read at the base, each as the MADR's D2, D3 and A3 have it:

   | acpext | source |
   | :--- | :--- |
   | `SessionParams`, `MessageParams`, `ModeParams`, `CompactParams`, `EnabledParams` | `acpserver/extensions.go:80`, `:86`, `:91`, `:96`, `:101` |
   | `SetSessionNameParams`, `ForkParams`, `GetEntriesParams` | `:106` (`nameParams`), `:111`, `:116` (`entriesParams`) |
   | `SetPersonalityParams`, `PersonalityResult` | `acpserver/personality.go:280`, `:287` |
   | `QueuedResult`, `ClearedResult`, `CompactResult`, `UsageResult` | `acpserver/extensions.go:145`, `:183`, `:220`, `:269` |
   | `ForkResult`, `CloneResult` | `:384` (`forkResult`; clone sets no `text`, `:433`) |
   | `ModelRef`, `StateResult`, `MessagesResult`, `EntriesResult`, `TextResult` | `:438`, `:444`, `:503`, `:515`, `:543` |
   | `SessionStats`, `ContextUsage`, `TokenStats` | `:580`, `:595`, `:601` (`statsTokens`) |
   | `ForkMessage`, `ForkMessagesResult` | `:632`, `:637` |
   | `RetryNotice`; `Phase`; `RetryCause` | `acpserver/prompt.go:81`; `acpserver/autocompact.go:24-25`; `agent/event.go:87-93` and `llm/provider/list.go:106-128` |
   | `CompactionNotice`, `CompactionReason`, `CompactionPause` | `acpserver/autocompact.go:32`, `:26-28`, `:44-51` |
   | `PersonalityNotice` | `acpserver/personality.go:25` |
   | `QueueMode`, `Disposition` | `acpserver/queue.go:13-14`; `acpserver/extensions.go:180` |
   | `CapabilitiesMeta`, `MetaKey` | `acpserver/agent.go:394`, `:36` |
   | `AuthProvider`, `LoginSuffix` | `acpserver/auth.go:16`, `:32` |
   | `ErrorData` | `acpserver/agent.go:44-47`, `:417-418`, `:786`; `auth.go:80`; `prompt.go:280-288` |

   `authProviders()`, run on a scratch copy at go-llmprovider-sdk `v1.26.1`, gave the nine of MADR D3, in order, with
   `Login` for grok and kilo; `AuthProviders()` and its test hold that list.
2. **`acpext/go.mod`** (step 2): `module github.com/maccavelli/gobble-cli/acpext`, `go 1.27.3`, no requirement.
   `go -C acpext mod tidy -diff` is empty and writes no `go.sum`. `acpext/LICENSE` is the root's (`cmp` equal).
3. **The package** (step 3): `doc.go` (the method table, how a client calls and reads `CapabilitiesMeta` with
   `acp.DecodeMeta`, the raw fields, versions), `methods.go` (the 23 constants, `Requests()`, `Notifications()`),
   `types.go` and `auth.go`, as A1 to A3 have them. golint is clean on each file.
4. **The tests** (step 4): `roundtrip_test.go`, `methods_test.go`, `auth_test.go` and `example_test.go`, 22 tests and
   examples, passing. `testdata/` holds 34 frames: 20 results (the six `{}` results have no type and are not kept), 6
   notifications, 4 errors' data, `initialize`'s `_meta.gobble` from `session.golden`, and 9 params frames written from
   `acpserver`'s tags (deviation 1). The capture ran every `acpserver` test with a recorder on a scratch copy, 660
   lines; the frames keep gobble's own number spelling, and their scratch paths are `/work/…`.
5. **Seen failing** (C9), each on its own scratch copy, by a script; every expectation held:
   - `name` to `title` in `SetSessionNameParams`: `set_session_name.params.json, encoding/json: decode: json: unknown
     field "name"`, the v2 decode likewise, and `ExampleSetSessionNameParams` failed;
   - `omitzero` dropped from `RetryNotice.Attempt`: `retry.end.notice.json, encoding/json: encodes back as …`, under
     both encoders;
   - `MethodUsage` dropped from `Requests()`: `TestRequests` and `TestCapabilitiesMetaIsRequests` failed;
   - a provider dropped from `AuthProviders()`: `TestAuthProviders` failed;
   - a member `jitterMs` added to `retry.start.notice.json`: `json: unknown field "jitterMs"`, and v2's `unknown object
     member name "jitterMs"`;
   - `SessionID` dropped from `ErrorData`: `invalid-params.error.json` and `mid-turn.error.json` refused `sessionId`
     under both encoders;
   - an unbroken copy passed.
6. **`check-acpext-deps`** (step 5), in `preflight`. Seen failing: an import of `jsonx` (`check-acpext-deps: go list
   failed: acpext does not build from the standard library alone`); `golang.org/x/sys/unix`, required and building
   (`acpext imports outside the standard library: golang.org/x/sys/unix`); and `github.com/maccavelli/acp-go-sdk2`
   (the same, naming it).
   - *Found by C9:* the first run passed the `jsonx` copy. macOS's GNU Make 3.81 ignores `.SHELLFLAGS`, so a recipe
     does not stop at a failing command: `go list` failed and the recipe went on. The gate now checks `go list`'s
     status, and each per-module loop keeps every module's failure in `rc`, which a vet error planted in `jsonx`, the
     root's, before acpext's passing turn, now shows (`make vet` exit 2).
7. **Per-module gates** (step 6). `MODULES := . acpext`; `test`, `race`, `vet`, `lint`, `staticcheck`, `vulncheck`,
   `tidy` and `fix-check` loop over it, each in the module's directory with the root's tools; `preflight`'s tidy check
   runs `go -C acpext mod tidy -diff`. Seen failing, each passing with the root alone: an unused requirement in
   `acpext/go.mod` (the tidy diff); `x = x` in an acpext file (`make vet`: `self-assignment of x`); a failing acpext
   test (`make test`). The root's `go list ./...` names no acpext package.
   - Deviation 2's exclusion is seen holding acpext's own code to the rule: `encoding/json` imported in
     `acpext/types.go` is refused (`depguard`).
8. **CI** (step 7): `go -C acpext mod download`, `go -C acpext test -count=1 ./...` and `go -C acpext mod tidy -diff`
   join their steps. `actionlint` passed.
9. **`.gitignore`** (step 8) has `go.work` and `go.work.sum`.
10. **Nothing in the root changed** (step 9): the root's `go.mod` and `go.sum` are unchanged; `go list -m` prints one
    module.
11. **G-local:** `make pre-add-check` over acpext's 8 files: clean; `make preflight`: passed, with `go vet (acpext)`,
    `golangci-lint (acpext)`, `govulncheck (acpext)` and `check-acpext-deps: ok`; `make race`: 39 ok, the root's 38 and
    acpext.
12. **G-hosts,** on a copy of the tree, 878 files: on each host `make test` 39 ok and `make race` 39 ok, `make
    check-records markdownlint` passed, and preflight stopped at `verify-build-metadata`, as on every copy, past
    `check-acpext-deps: ok`.
    - On the Windows laptop the first run stopped in `staticcheck`'s darwin pass, in root packages: `could not import
      github.com/maccavelli/gobble-cli/internal/cli/complete (can't find export data (bufio: buffer full))`, and the same
      for `editor`, `render` and `term`. It failed the old way too, the binary run from the root. With an empty build
      cache of its own it passed, and the run was repeated that way and passed.
    - The cause was the host's build cache: 21 of its 45,132 output files were zero-filled from the start and did not
      hash to their names, the four test variants' export data among them. On the owner's word they were removed,
      each re-checked first, and the darwin pass then passed on the host's own cache. Nothing of P1's caused it.
13. ~~**Not done:** the commit, then the disclosure guard, the push and the tag (step 11), each on the owner's
    word.~~ *(Done 2026-11-06, below.)*
14. **Committed** as `4d1eb01` on the owner's "Commit to main". **The disclosure guard** over the outgoing commits,
    `0c07a72`, `8632eeb` and `4d1eb01`, exited 0 with no finding.
15. **Pushed and tagged** on the owner's "push and tag" (C12): `main` `08439b4..4d1eb01`; the annotated tag
    `acpext/v0.1.0` (`7ef324d`) on `4d1eb01`, in the form of the repository's `v*` tags, pushed as a new tag.
16. **The module proxy** served it at 2026-11-06T16:00:25Z, from an empty module cache:
    `go list -m -json github.com/maccavelli/gobble-cli/acpext@v0.1.0` gave `v0.1.0`, `Time` 2026-11-06T15:54:37Z,
    `Origin` `4d1eb013…` at `refs/tags/acpext/v0.1.0`. `go mod download`, through the checksum database, gave
    `h1:vdk8XDwqrEMor1YAz77soPCER8gQ+ik/QjsOe2hexBg=` (`go.mod` `h1:go2H/pww4XSwD3VOrUI7St71lhNSFuNuSmT2FgZAjFc=`).
    Its zip holds acpext's 44 files and nothing of the root's, `LICENSE` among them (MADR D1).
17. **P1 is complete.** P2's gate is met.

### P2 record (2026-11-06): gobble serves and reads from acpext

The base is `4d1eb01`, P1's commit, with P1's tag record uncommitted beside it. No deviation.

1. **The facts** (step 1): `git diff --stat 8632eeb..HEAD` over `acpserver`, `acpclient`, `internal`, `agent` and `llm`
   is empty, so P1 step 1's lines stand; the tag `acpext/v0.1.0` is P1's commit `4d1eb01`.
2. **The requirement** (step 2): `go get github.com/maccavelli/gobble-cli/acpext@v0.1.0` ("added … v0.1.0"); its
   `go.sum` lines are the proxy's (`h1:vdk8XDwq…`, `go.mod` `h1:go2H/pww…`). After the imports, `go mod tidy` made it a
   direct requirement. No `replace`.
3. **`acpserver`** (step 3):
   - deleted: the param, result and notice structs, `methodRetry`, `methodCompaction`, `methodPersonality`, the phase
     and reason constants, `compactPause`, `pauseTurn`, `pauseSession`, `capabilitiesMeta`, `AuthProvider`,
     `loginSuffix`, `keyError` and `keySessionID`. Their uses name acpext's. No alias is left;
   - the `extensions` table is keyed by `acpext.Method…`; `Capabilities` advertises `acpext.CapabilitiesMeta` under
     `gobble`, now `acpext.MetaKey`;
   - `extSession` decodes the params into their acpext type, then into `acpext.SessionParams` for the session, as MADR
     A2 has it; each notification and result sets `SessionID` with `string(id)`;
   - every error's data, 32 maps, is built from `acpext.ErrorData` by `errData`, which encodes it with `jsonx`; a struct
     of strings cannot fail there, and its members are sorted as `jsonx.Strings` wrote them (0039-MADR D14);
   - `CompactResult.Details`, and each of `MessagesResult.Messages` and `EntriesResult.Entries`, are encoded by
     `jsonx`, one value each, as the whole result once was;
   - kept: `pauseNone`, the zero `acpext.CompactionPause`; `queueOneAtATime` and `queueAll`, now defined from acpext's
     values, since step 3 does not list them and `queue_test.go` and `settings_test.go` name them; and `keyReason`,
     which `extensions_test.go`'s `errText` reads.
4. **`acpclient`** (step 4): `update.go` decodes into `acpext.RetryNotice`, `CompactionNotice` and
   `PersonalityNotice`, `conn.go` registers them by acpext's names, and `sessions.go` sends
   `acpext.SetSessionNameParams` to `acpext.MethodSetSessionName`. `go doc -all ./acpclient`, 390 lines, is the same
   as the base's (`diff` exit 0).
5. **`internal/cli`** (step 5): `authProviders` returns `[]acpext.AuthProvider`.
6. **The root test** (step 6), `internal/cli/acpext_test.go`, with no new exported API:
   - `TestAcpextRequestsAreServed`: `acpserver.Capabilities()`'s `_meta.gobble`, decoded, is `acpext.Requests()`;
   - `TestAcpextNotificationsAreSent`: read from `acpserver`'s source with `go/ast`, each `notify` names an acpext
     notification constant, each acpext notification is sent, and `NotifyExtension` is called in `notify` alone;
   - `TestAcpextAuthProvidersAreTheRegistry`: `acpext.AuthProviders()` equals `authProviders()`.

   Seen failing (C9), each on a scratch copy: a twenty-first `"_gobble/twenty_first"` entry in the table only
   (`acpserver serves […]`); acpext's last provider dropped, through a local `go.work` on the copy (C13)
   (`acpext.AuthProviders() = […], the registry gives […]`); and `out.notify("_gobble/other", nil)` added
   (`../../acpserver/commands.go:742:2: a notification not named by an acpext notification constant`).
7. **archtest** (step 7): rule 12, only `acpserver`, `acpclient` and `internal/cli` import acpext, and `relOf` names
   acpext's paths as another module's before it maps a root package (MADR A4). Four table cases: the module, `agent`
   importing acpext, the three allowed, and `jsonx` importing acpext, which rule 10 does not count. `doc.go` says
   twelve rules. Seen failing: an import of acpext in `agent/doc.go` (`rule 12: …/agent imports …/acpext`); and the
   `relOf` line removed (`rule 10: …/jsonx imports …/acpext`, and the real-tree case of rule 12 too, since acpext's
   own packages were then read as the root's).
8. **The wire is unchanged** (step 8): `git diff --stat` lists no testdata or golden. The `_test.go` changes are step
   6's new file, step 7's cases, and the six files' renames. The rename check (`renamecheck.py`, scratch) passed: 24
   changed lines, each a rename, among them the four `string(id)` conversions of `personality_switch_test.go`. Seen
   failing on a copy where a renamed line in `prompt_test.go` also changed `Level: 3` to `Level: 4` (`not a rename`,
   quoting both lines). `go test` passed over `acpserver`, `acpclient` and `internal/cli`.
9. **No `"_gobble/` literal** (step 9) is left in a non-test Go file outside acpext (`git grep` exit 1).
10. **Documents** (step 10): AGENTS.md's module sentence names acpext, its tag form and the no-`replace`, no-`go.work`
    rule. `docs/architecture.md` describes the nested module, adds rule 12, the lint exclusion for acpext's tests,
    acpext's row in Dependencies, the per-module gates and `check-acpext-deps` in Checks, and `acpext/` in the Tree.
11. **G-local:** `make pre-add-check` over the 23 changed Go files: clean; `make preflight`: passed, golangci-lint 0
    issues in each module, `check-acpext-deps: ok`; `make race`: 39 ok.
12. **G-hosts,** on a copy of the tree, 879 files, on the Linux machine and the Windows laptop: on each, `make test` 39
    ok and `make race` 39 ok, `make check-records markdownlint` passed, and preflight stopped at
    `verify-build-metadata`, as on every copy, past `check-acpext-deps: ok`. The Windows laptop ran on its own build
    cache, repaired in P1's record.
13. ~~**Not done:** the commit, and its push, P3's gate (C12), each on the owner's word.~~ *(Done: P3's record.)*

### P3 record (2026-11-06): acceptance and close-out

The base is `14201ed`. No deviation.

1. **P2 committed and pushed** (P3's gate, C12), on the owner's "commit and push to main": the disclosure guard over
   `14201ed` exited 0; `main` `4d1eb01..14201ed`. CI on `14201ed` passed every job: `validate` on Linux, macOS and
   Windows, `sbom`, `build` and its five `identity` jobs; `release` skipped, as on any push but a `v*` tag.
   - CI on P1's `4d1eb01` was marked failed by one job alone, `build / identity (gobble, darwin/arm64)`, cancelled
     after 15 minutes with no step run, so with no runner; every test, preflight and build step had passed. The same
     job passed on `14201ed`.
2. **`go install`** (step 1): `go install github.com/maccavelli/gobble-cli/cmd/gobble@14201ed` with empty `GOPATH`,
   `GOMODCACHE` and `GOCACHE` in a scratch directory, `GOFLAGS=-mod=mod`, `GOWORK=off`, through the public proxy:
   exit 0. `go version -m` on the binary names `mod github.com/maccavelli/gobble-cli
   v0.1.2-0.20261106165406-14201ed5073d` and `dep github.com/maccavelli/gobble-cli/acpext v0.1.0
   h1:vdk8XDwqrEMor1YAz77soPCER8gQ+ik/QjsOe2hexBg=`, with no `=>` line.
3. **The consumer** (step 2): a scratch module outside the tree, `example.com/consumer`, its own empty module cache.
   `go get github.com/maccavelli/gobble-cli/acpext@v0.1.0` added it from the proxy, with the same two `go.sum` lines.
   It decodes `testdata/retry.start.notice.json` into `RetryNotice` with `encoding/json` and unknown fields refused,
   and prints `_gobble/retry {SessionID:00dc6acf-… Phase:start Cause:empty_reply Attempt:1 MaxAttempts:2 DelayMs:2000
   Error:} ok=true`. `go list -m all` is `example.com/consumer` and `github.com/maccavelli/gobble-cli/acpext v0.1.0`:
   no go-llmprovider-sdk, and, by A1, no acp-go-sdk2.
4. **`go -C acpext list -m all`** (step 3): `github.com/maccavelli/gobble-cli/acpext`, alone.
5. **The MADR's Confirmation** (step 4):

   | # | Confirmation | Evidence |
   | :--- | :--- | :--- |
   | 1 | acpext's graph is acpext alone (A1) | step 4 above |
   | 2 | `check-acpext-deps` passes, seen failing | P1 record, item 6: `jsonx`, `golang.org/x/sys/unix`, acp-go-sdk2 |
   | 3 | the root requires acpext at a tag, no `replace`; `go install` works and `go version -m` names it | `go.mod:20`; no `replace` or `go.work`; step 2 above |
   | 4 | `session.golden` and the extension tests pass with their frames unchanged | P2 record, item 8 |
   | 5 | the root test passes, its three assertions seen failing | P2 record, item 6 |
   | 6 | archtest's rule passes, seen failing | P2 record, item 7 |
   | 7 | a consumer from the proxy decodes `_gobble/retry`, with no go-llmprovider-sdk | step 3 above |
   | 8 | the strict round trip, each planted failure seen | P1 record, item 5 |
6. **The note for magic-cli-remote** (step 5), on the owner's word, is its issue
   [maccavelli/magic-cli-remote#36](https://github.com/maccavelli/magic-cli-remote/issues/36). It cites 0179 at
   `0f595dc9`: acpext requires no module (`:234`, `:524`, `:590`, and gate 6 at `:578-580`); `ErrorData` has
   `sessionId` (D3 `:390-391`, against D8's mid-turn refusal at `:465`); gate 3's `v1.31.2` (`:575`) is `v1.34.0` in
   both; the `string` session ids and flat types; every D13 name as written, with D9's constants and the open
   `RetryCause`; `acp.DecodeMeta` for `_meta.gobble`; the nine providers; acpext's own strict round trip; and that
   gates 2 and 4 wait on a gobble release at or after `14201ed`, since `v0.1.1` predates acpext.
7. **Status** (step 6): every Confirmation is met. This plan is `complete`; the MADR stays `accepted`.
