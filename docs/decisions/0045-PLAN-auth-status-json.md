---
status: in-progress
date: 2026-11-07
associated-madr: "0045-MADR-auth-status-json.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement giving `gobble auth status` a `--json` form that reports every key provider's credential presence in one call

Associated MADR: [0045-MADR-auth-status-json.md](0045-MADR-auth-status-json.md)

## Goal

`gobble auth status --json` prints MADR D2's object for every key provider, from the same rows as the text form (D3).
It runs nothing and writes nothing to stderr (D4), and reports an unreadable file as D5's error. The shape is
documented and pinned (D6). The plan is complete when the MADR's Confirmation 1 to 8 is met.

## Scope

**Base.** `main` at `3dbf756`, equal to `origin/main`, clean.

**In scope:**

* `internal/cli/auth.go`: `AuthStatusCmd` gains `JSON`. The row type and the one function that computes rows.
  The text rendering from those rows, and the JSON writer and its error object.
* `internal/cli/credentials.go`: `newResolver` keeps auth.json's load error, beside the resolver it builds, for D5.
* `internal/cli/modelsjson.go`: the models.json cache keeps a read or parse error beside its value, for D5.
* *Deviation 1 (2026-11-07):* `internal/cli/main.go`, where `runEnv` holds the field step 3 names.
* `internal/cli/auth_test.go`: the new tests.
* `internal/cli/testdata/auth-status.json.golden`: new.
* `docs/guides/credentials.md`: the `--json` form.
* `docs/README.md`: the two 0045 rows.
* this pair.

**Out of scope:** `acpext/` (D7); `auth check` (MADR "Not decided here"); `go.mod`; the workflows; a release tag;
magic-cli-remote's side; posting on gobble-cli#13, which waits for the owner's word.

## Conventions every phase follows

* **C1. Start state.** Each phase first re-reads `git status`, `HEAD` and `origin/main`. A difference from this plan is
  a deviation.
* **C2. Seen failing.** Every new test is seen failing on a scratch copy of the tree, never in the working tree: first
  against today's code, then against a planted break of the new code.
* **C3. Evidence.** Commands write to a scratch log, with their exit status taken before any filter. The phase record
  quotes the lines it relies on.
* **C4. Deviations stop the phase,** and are recorded here, and in the MADR when a decision or fact changes, before work
  goes on.
* **C5. Records and commits.** `make check-records markdownlint` passes, and the identifier scan finds nothing, before
  staging. The agent stages. The owner commits, or tells the agent to, with `git commit --no-edit`. The disclosure
  guard runs before any push.
* **C6. Identifiers.** No hostname, account name or real-machine path in a committed file. The dev hosts are named by
  platform only.
* **C7. No real effects.** The package's `TestMain` already unsets every credential variable and lists offline. The
  new tests use `newCfgEnv`'s scratch home. No test reaches a service, and no test runs a key's command except to prove
  that it is not run.

## Implementation Steps

### P0 — the records

Files: this pair; `docs/README.md`.

1. `docs/README.md` gains the two 0045 rows, `proposed`.
2. `make check-records markdownlint` passes.
3. Staged for the owner's review. The owner answers the MADR's Q1 and Q2, and commits.

### P1 — the command, its tests and its guide

**Gate:** P0 committed, Q1 and Q2 answered, and the owner's approval of this plan.

Files: those in Scope.

1. **The tests** (`internal/cli/auth_test.go`), written first.
   * `TestAuthStatusJSON`: a `newCfgEnv` home holding one provider of each row kind in D2.
     * `auth.json` holds an `api_key` with a key (openai), an `api_key` with an empty key (opencode-zen), an OAuth
       login (grok), an OAuth entry for a provider with no login (claude), and an entry of an unknown type (together).
     * `models.json` holds a `!command` key for huggingface. The command would create a file in the scratch home.
     * `GEMINI_API_KEY` is set.
     * Each value holds the word `secret`.
     * The test runs `auth status --json`. It wants exit 0, stderr empty, stdout equal to
       `testdata/auth-status.json.golden`, and no `secret` in stdout or stderr.
     * The command's file must not exist afterwards. `auth.json` must be byte-identical to what was written.
   * `TestAuthStatusJSONAgrees`: for the same home, each provider's `source` in the JSON is the source its text row
     names (`auth.json`, `models.json`, the variable for `env`, `none`).
   * `TestAuthStatusJSONError`, one case at a time:
     * an `auth.json` that is not JSON, holding `secret`;
     * an `auth.json` that is a directory;
     * a `models.json` that is a directory.

     Each wants exit 2, stderr empty, stdout D5's object naming the file, and no `secret` anywhere. A `models.json`
     with an unknown field still answers, with exit 0.
   * The golden file is written once with the package's `-update` flag (`internal/cli/main_test.go:20`). It is then
     read in full and checked against D2, field by field, before it is committed.
2. **Seen failing** (C2), on scratch copies:
   * today's code: every new test fails (`unknown flag --json`);
   * a planted stderr write in the JSON path: `TestAuthStatusJSON` fails on stderr;
   * the resolver's `Credential` in place of `Has` for one row: the `!command` runs, and the file check fails;
   * D5's error check removed: the error cases exit 0 with rows of `none`;
   * `expires` written in local time: the golden comparison fails.
3. **The code.**
   * `credentials.go`: `newResolver` keeps auth.json's load error on the `runEnv` (for example `credsErr`), set
     whenever `e.creds` is built or rebuilt. The warning in text mode is unchanged.
   * `modelsjson.go`: the cache holds `(value, error)` for each path. `readModelsJSON` still warns and returns nil, and
     a new accessor returns the error.
   * `auth.go`:
     * `statusRow{Provider, Source, Variable, AuthType, Usable, Expires, Reason}` with D2's JSON names and
       `omitempty`;
     * `e.statusRow(id)`, which walks D2's order: the auth.json entry, then `Has`;
     * the text cell, rendered from a row into today's exact strings;
     * `AuthStatusCmd.JSON`, help "Print JSON: every key provider's credential source, never a credential.". With it,
       `e.out.err = io.Discard` and `e.logs.stderr = nil`, as `AuthLoginCmd.Run` does. D5's checks come first, then
       `jsonx.Marshal(..., jsontext.WithIndent("  "))` and a newline.
4. **The guide.** `docs/guides/credentials.md`, after the `auth status` example:
   * an example of `--json`;
   * a table of the fields and values from D2;
   * D5's error object and exit 2;
   * D6's stability rule, in one sentence.
5. **Gates.**
   * `make pre-add-check FILES="..."` on every changed Go file.
   * `make preflight` and `make race`.
   * On the Linux and Windows dev hosts, the tree shipped to a scratch directory: `go test ./internal/cli/` and the race
     run. The Windows run covers the directory-as-file cases and the `!command` under Git Bash.
   * A binary built from the tree, run against a scratch `GOBBLE_HOME`: `gobble auth status --json` prints the shape,
     and `gobble auth status` prints the table as before.
6. **Records.** This plan's execution record and the `docs/README.md` rows. Staged; the owner commits.

### P2 — acceptance

**Gate:** P1 committed and pushed on the owner's word.

1. CI passes on its three runners.
2. The plan's status becomes `complete`, and the MADR's `accepted`, with the `docs/README.md` rows to match.
3. On the owner's word, a comment on gobble-cli#13 names the commit and says the command reaches Magic at the next
   `v*` release. The issue closes when a release carries it, which the owner tags.

## Verification

| Confirmation | Where |
| :--- | :--- |
| 1. The golden JSON, exit 0, stderr empty | P1 step 1, `TestAuthStatusJSON` |
| 2. No value; no command run | P1 step 1, `TestAuthStatusJSON` |
| 3. `auth.json` unchanged | P1 step 1, `TestAuthStatusJSON` |
| 4. D5's error object and exit 2 | P1 step 1, `TestAuthStatusJSONError` |
| 5. Text unchanged; the two agree | `TestAuthStatus`; `TestAuthStatusJSONAgrees` |
| 6. Seen failing | P1 step 2 |
| 7. preflight and CI | P1 step 5, P2 step 1 |
| 8. The guide | P1 step 4 |

## Rollout and Rollback

* **Rollout.** The command lands on `main` in P1, and reaches users and Magic at the next `v*` release, which this plan
  does not make.
* **Rollback.** Before a release: revert P1's commit. After a release, the flag stays, since removing it breaks Magic
  (D6). A fault in it is fixed forward.

## Execution record

### Deviation 1 (2026-11-07): `internal/cli/main.go` joins the scope

Step 3 keeps auth.json's load error "on the `runEnv`". `runEnv` is declared in `internal/cli/main.go`, which Scope did
not list. The field, `authErr`, is three lines there, beside `creds`, and guarded by `credsBuild` as `creds` is. No
decision or fact in the MADR changes, so the MADR is not amended. Recorded here rather than stopped for, since the
step as written cannot be done without it; the owner may still refuse it at review.

### P1 record (2026-11-07)

On the owner's "committed, proceed" after P0 (`53d42fd`), with Q1 and Q2 answered. Start state (C1): `main` at
`53d42fd`, `origin/main` at `3dbf756`, the tree clean.

**Step 1, the tests** (`internal/cli/auth_test.go`), written first.

* `statusFixture`: auth.json with openai's `api_key`, opencode-zen's empty key, grok's OAuth login, claude's OAuth
  entry and together's `wellknown` entry; models.json with huggingface's `!touch '<home>/command-ran' && echo secret`;
  `GEMINI_API_KEY` set; every value holding `secret`.
* `TestAuthStatusJSON`: exit 0, stderr empty, no `secret`, the command's file absent, auth.json byte-identical, and
  stdout equal to `testdata/auth-status.json.golden`. `newCfgEnv` changes directory, so the test changes back to the
  package before `golden`.
* `TestAuthStatusJSONAgrees`: the JSON's source agrees with the text's cell for every row, and the ids are
  `KeyProviders()`.
* `TestAuthStatusJSONError`: auth.json not JSON, auth.json a directory, models.json not JSON, models.json a directory.
  Each gives exit 2, stderr empty, `error.file` naming the file, its message naming it, and no `secret`. A models.json
  with a diagnostic (`anthropic` and `claude` both given) answers with exit 0 and stderr empty.
* The golden file, written with `-update` and read in full: nine rows, in `KeyProviders()` order. gemini `env` with
  `GEMINI_API_KEY`; openai `api_key`, usable; claude `oauth`, not usable, `oauth_unused`, `2026-06-09T10:13:20Z`;
  grok `oauth`, usable, the same expiry; opencode-zen `api_key`, not usable, `empty_key`; huggingface `models.json`,
  usable; together `auth.json`, not usable, `unknown_type`, no `authType`; opencode-go and kilo `none`. The first
  write had `expires` before `reason`. The struct was reordered so the output follows MADR D2's example, and the file
  was written again.

**Step 2, seen failing,** by `scratchpad/status_seenfail.py` (not committed), each on a copy of the tree. It was run
on the final code, after the lint fixes below:

| Mode | Failure |
| :--- | :--- |
| today's code (the changed `.go` files at `HEAD`) | every new test: `exit 2, stderr "Error: unknown flag --json…"`, and `TestAuthStatusJSONAgrees` `status --json: jsontext: unexpected EOF` |
| stderr left on, and a warning on the path | `TestAuthStatusJSON`: `exit 0, stderr "Warning: planted\n"` |
| `Credential` in place of `Has` | `TestAuthStatusJSON`: `huggingface's !command ran: <nil>`; `TestAuthStatus`: no `grok          models.json` |
| D5's check removed | each error case: `exit 0`, rows of `none` |
| `expires` in local time (`TZ=Asia/Kolkata`) | `auth-status.json differs from testdata/auth-status.json.golden` |

**Step 3, the code.**

* `internal/cli/auth.go`:
  * `AuthStatusCmd.JSON`;
  * `runJSON`: stderr and the log's stderr lines off; then D5's checks; then the report, indented by two spaces;
    exit 2 with the error;
  * `statusError`, the type and the `runEnv` method;
  * `authStatusRow` (`statusRow` was taken by a test's variable in `reprint_test.go`) and `e.statusRow(id)`, which
    walks D2's order through `Has`;
  * `credentialSource` is now the row's text, so `providers`' wording (`providerscmd.go:99`) is unchanged.
* `internal/cli/credentials.go`: `newResolver` sets `e.authErr`. `authJSONError()` returns it once `credentials()`
  has built the resolver.
* `internal/cli/modelsjson.go`:
  * the cache holds `modelsJSONRead{mj, err}`;
  * `loadModelsJSON` returns the error it warns;
  * `readModelsJSON` keeps its signature for `regular_test.go`;
  * `modelsJSONError()` is new.
* `internal/cli/main.go`: `authErr` (deviation 1).
* The first `make preflight` stopped at golangci-lint with three findings. errcheck flagged `readModelsJSON`
  discarding `loadModelsJSON`'s error; it now checks it. goconst flagged `"auth.json"`, a string with three
  occurrences, twice; `auth.go` now uses `statusAuthJSON`, `statusModelsJSON` and `statusNone` for its literals.

**Step 4, the guide.** `docs/guides/credentials.md`, after the `auth status` example:

* `--json`'s example;
* the fields' table;
* D5's error, with `<config>` for the directory, as `models-json.md` writes it;
* D6 in one sentence, linking the MADR.

**Step 5, gates.**

* `make pre-add-check FILES=` the five Go files: "5 file(s) clean (gofmt, golint, govulncheck)".
* `make preflight`: "preflight passed"; golangci-lint 0 issues on both modules; markdownlint 0 issues.
* `make race`: 39 ok, no failure.
* The Linux dev host (Go 1.27.3, 888 files shipped, no `._*`): `go test ./internal/cli/` ok, and `-race` ok.
* The Windows dev host (Go 1.27.3, `TMP` and `TEMP` set to a scratch directory, 888 files, no `._*`): the same, both ok.
* A binary built from the tree, against scratch `GOBBLE_HOME`s:
  * `auth status` prints the table as before;
  * `--json` prints the rows, exit 0, stderr empty;
  * a truncated auth.json gives `{"error": {"file": "auth.json", "message": "<home>/auth.json: invalid auth.json:
    jsontext: unexpected EOF within \"/openai\" after offset 49"}}`, exit 2;
  * a models.json directory gives `open <home>/models.json: not a regular file`, exit 2;
  * a truncated models.json gives `models.json: jsontext: unexpected EOF within \"/providers/huggingface\" after
    offset 49`, exit 2;
  * no output holds a value;
  * the text form with the truncated auth.json keeps today's warning on stderr, then rows without the file.
* The identifier scan of every changed file found nothing.

**Step 6** is this record and the `docs/README.md` rows. Staged; the owner commits. P2 waits for the push.
