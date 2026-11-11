---
status: proposed
date: 2026-11-11
associated-madr: "0054-MADR-go-1-27-4-and-sibling-pin-refresh.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement moving Go to 1.27.4 and pinning acp-go-sdk2 v1.36.0 and go-selfupdate-lib v1.26.2

Associated MADR: [0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md)

## Goal

`main`'s CI is green on Linux, macOS and Windows again; both modules state Go 1.27.4 (MADR D1, D2); `go.mod` pins
acp-go-sdk2 `v1.36.0` and go-selfupdate-lib `v1.26.2` (D3), with CI and the docs naming the same versions (D4); and the
owner's answer to Q1 is recorded and held by a test (D5). The plan is complete when the MADR's Confirmation 1 to 9 is
met.
*(2026-11-11, later: the owner answered Q1, to advertise client system prompts and honour them, and the MADR's D7
says how. P2 is rewritten around it, so this plan also builds that feature: by the owner's choice it is feature work,
not only a pin move. `override` is the MADR's Q2, awaiting the owner; P2 builds its (a) until then.)*

The MADR is `accepted` (the owner, 2026-11-11). This PLAN is `proposed`: nothing below starts before the owner approves
it, whole or by phase. It lands before
[0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)'s
P2.

## Scope

**Base.** `main` at `17d1605`, level with `origin/main`. The tree holds six untracked records and nothing else: this
pair, and the 0051 and 0052 pairs, all uncommitted. `go.mod` requires Go 1.27.3, acp-go-sdk2 `v1.35.0`,
go-llmprovider-sdk `v1.29.0`, go-selfupdate-lib `v1.26.0`, go-tui-lib `v2.12.0` and acpext `v0.3.0`.

**In scope:**

* **P0:** this pair; `docs/README.md`'s two rows.
* **P1, Go 1.27.4:** `go.mod`; `acpext/go.mod`; `scripts/check_libs.py` (`:124`); `AGENTS.md` (`:16`);
  `docs/architecture.md` (`:9`); `docs/guides/developing.md` (`:9`); dated notes in
  `0004-MADR-go-module-architecture.md` (the Toolchain line), `0023-MADR-pin-current-releases.md` (the Go row) and
  `0042-MADR-acpext-extension-contract-module.md` (D1's `go` line).
* **P2, acp-go-sdk2 `v1.36.0`:** `go.mod`, `go.sum`; `README.md` (`:155`); `docs/architecture.md` (`:189`); under Q1
  (a), one new test in `acpserver`; the MADR's D5, for the owner's answer.
  *(2026-11-11, later, in place of "under Q1 (a), one new test": MADR D7's files. `session/session.go`
  (`CustomSystemPrompt`); `prompt/sections.go` and `prompt/sections_test.go` (`Options.Client`, the `client_prompt`
  section and its check); `acpserver/agent.go` (`Capabilities()`, `NewSession`, `open`, `LoadSession`,
  `ResumeSession`, `liveSession`, `restore`); `acpserver/history.go` (`replayPath`'s state); `acpserver/sysprompt.go`
  (`sections`); `acpserver/systemprompt.go` and `acpserver/systemprompt_test.go` (new); `acpserver/agent_test.go`
  (`TestCapabilitiesAreHonest`); `acpserver/testdata/session.golden`; `README.md`'s system-prompt bullet (`:59-60`)
  and `docs/architecture.md`'s `prompt` and `acpserver` rows (`:118`, `:124`).)*
* **P3, go-selfupdate-lib `v1.26.2`:** `go.mod`, `go.sum`; `.github/workflows/ci.yml` (`:191`, `:203`); `README.md`
  (`:162`); `docs/architecture.md` (`:193`).
* **P4:** the MADR's Completion section; this PLAN's status; `docs/README.md`'s status cells.

**Out of scope:**

* go-llmprovider-sdk's next minor and `authcontract` (0051-PLAN P2); the root's acpext pin, which stays `v0.3.0`; an
  acpext tag (MADR D2).
* Any code change either tag needs beyond its pin, which is amended in first. Applying a client's system prompt (Q1
  (b)) is one.
  *(2026-11-11, later: no longer out of scope. The owner's answer to Q1 puts it in P2 (MADR D7), and this is that
  amendment. `override` (MADR Q2 (b) or (c)) stays out until the owner answers it.)*
* `TestRuleStoreConcurrent`, `permission/` and `internal/filelock/` (MADR, "Observed, not decided"; C5).
* magic-cli-remote's own move to acp-go-sdk2 `v1.36.0`; acp-go-sdk2; go-selfupdate-lib.
* A push, a gobble release tag, or a patch release for the Go fix (MADR D6).
  *(2026-11-11, later: the owner ruled that no `v0.5.1` is cut; MADR D6's note.)*

## Conventions

* **C1. Start state.** Each phase first re-reads `git status --short`, `HEAD`, `origin/main` and `go.mod`'s `go` line
  and pins. A difference from this plan is a deviation.
* **C2. Evidence.** Each command writes to a scratch log, and its exit status is taken before any filter. Tests run
  with the proxies at `127.0.0.1:9`, except `llm` and `llm/provider` (0049-PLAN C3). Temporary paths are quoted as
  `<TMP>`.
* **C3. Seen failing.** Every new test and every claim of the form "the old toolchain is refused" is seen failing on a
  scratch copy of the tree, never in the working tree, and the record quotes the failure.
* **C4. Deviations stop the phase.** A module or a `go` line moving that the step does not name, a `check_libs.py`
  failure, a failing test, or a golden transcript that changes, stops the phase. The owner is given the evidence and the
  ways to resolve it, and the choice is recorded here, dated, before work continues.
* **C5. `TestRuleStoreConcurrent`, if it recurs.** If that test, or any other test this plan does not touch, fails on
  a host or on a CI runner while this plan runs:
  * the phase stops, and the failing output is quoted in the execution record as it was printed;
  * the test is not skipped, not marked to be skipped, and its lock's deadline is not changed here;
  * the run is not repeated until it passes. A later green run does not erase the failed one, and is not this plan's
    acceptance by itself;
  * a record of its own is opened for it, numbered by `python3 scripts/check_records.py --next`, starting from the
    MADR's "Observed, not decided";
  * this plan's CI criterion (P4) then waits for that record's outcome, or for the owner's explicit word to accept a
    named run.
* **C6. Gates and commits.**
  * `make pre-add-check FILES="…"` on each changed Go file; `make preflight`; `make race`; `make
    check-test-isolation`; `python3 scripts/check_libs.py --self-test`.
  * On the Linux machine and the Windows laptop, on a scratch copy of the tree: `go test -count=1 ./...`, the race
    tests, and `go -C acpext test -count=1 ./...`. The route is the one 0050-PLAN's C4 and 0053-PLAN used. On Windows,
    `make` runs from Git Bash or PowerShell (`docs/guides/developing.md`, "What you need"), and `TMP` and `TEMP` point
    at a scratch directory outside the user's home (0049-PLAN, "P5b-1 record", step 5).
  * `make check-records markdownlint`; the identifier scan of the diff.
  * The agent stages. The owner commits with `git commit --no-edit`. The disclosure guard runs before a push, and the
    push is the owner's, on an explicit word.
* **C7. One phase, one commit.** P1, P2 and P3 are a commit each, so each reverts alone. P0 may be staged with P1, as
  the owner chooses.

## Implementation Steps

### P0 — the records

1. This PLAN becomes `in-progress` on the owner's approval. The MADR is already `accepted`.
2. `docs/README.md`: the 0054 MADR and PLAN rows. They are not written with the pair, which was authored before
   approval: the index is a tracked file outside the bootstrap exception (`AGENTS.md`, "MADR and PLAN before mutating
   work"), as 0051-PLAN's P0 step 3 treats its own rows.
3. `make check-records markdownlint`: exit 0.

Acceptance: both rows link the two files; `check-records` passes.

### P1 — Go 1.27.4 in both modules (MADR D1, D2)

1. **C1.** Then measure: `go.dev/dl/?mode=json`'s newest stable Go, and `go version` on the three hosts. A patch newer
   than `go1.27.4`, or a host not on `go1.27.4`, is a deviation.
2. **Seen at the base** (C3), on a scratch copy made with `git archive HEAD`:
   * `GOTOOLCHAIN=go1.27.3 go tool govulncheck ./...`: exit 3, "Vulnerability #1: GO-2026-6859", as CI's run
     `43080286906` printed it;
   * `GOTOOLCHAIN=go1.27.4 go tool govulncheck ./...`: exit 0, "No vulnerabilities found."
3. **The two `go` lines:**

   ```sh
   go mod edit -go=1.27.4
   go mod tidy
   go -C acpext mod edit -go=1.27.4
   go -C acpext mod tidy
   ```

   `git diff go.mod acpext/go.mod` is one line in each file. `go.sum` does not change, and acpext has none. No
   `toolchain` line appears in either file.
4. **The other places** (MADR F5's "yes" rows): `scripts/check_libs.py:124` sets `GOTOOLCHAIN="go1.27.4"`;
   `AGENTS.md:16`, `docs/architecture.md:9` and `docs/guides/developing.md:9` name Go 1.27.4. F5's "no" rows are not
   edited, and the record names them.
5. **The records,** each a dated note that quotes what it amends:
   * `0004-MADR-go-module-architecture.md`, the Toolchain line: a note dated 2026-11-11 naming `go1.27.4` and
     `0054-MADR-go-1-27-4-and-sibling-pin-refresh.md`, in the form of its notes of 2026-10-08 and 2026-10-21;
   * `0023-MADR-pin-current-releases.md`, the Go row;
   * `0042-MADR-acpext-extension-contract-module.md` D1, "`go 1.27.3`, as the root module, for fleet lockstep": now
     `go 1.27.4`, by MADR D2, with no tag of its own.
6. **The old toolchain is refused** (C3), on a scratch copy of the changed tree:
   * `GOTOOLCHAIN=go1.27.3 go build ./...`: exit 1, "go.mod requires go >= 1.27.4";
   * `GOTOOLCHAIN=go1.27.3 go -C acpext build ./...`: the same, for acpext's `go.mod`.
7. **Gates** (C6). No Go file changes, so the pre-add check has nothing to check. `make preflight` reports govulncheck
   clean for both modules (`MODULES := . acpext`, `Makefile:8`).
8. Staged; the owner commits. **After the owner's push:** `main`'s CI run passes `validate` on the three runners, where
   runs `43042877946` and `43080286906` failed. P2 starts from that commit. If the owner chooses to push P1 to P3
   together, P4 reads one run for all three.

Acceptance: steps 2 and 6 fail and pass as written; `git diff --stat` names only the files of this phase; the gates
pass on three hosts.

### P2 — acp-go-sdk2 `v1.36.0`, and client system prompts (MADR D3, D5, D7)

*(2026-11-11, later: rewritten around MADR D7, the owner's answer to Q1, that gobble advertises client system prompts
and honours them. This phase is feature work, not only a pin move, by the owner's choice. The phase as first written
is kept, word for word, under "P2 as first written" at its end; its heading was "P2 — acp-go-sdk2 `v1.36.0` (MADR D3,
D5)".)*

1. **C1.** Then the tag: `gh api repos/maccavelli/acp-go-sdk2/tags` lists `v1.36.0`, and `go list -m
   github.com/maccavelli/acp-go-sdk2@v1.36.0`, run from a scratch directory, succeeds through the proxy.
2. **Q2 is read first** (MADR D7). *(2026-11-11, later: the owner answered (a), so this phase builds `append` only
   and the stop below for (b) or (c) does not arise.)* With no answer from the owner, this phase builds (a): `append` only. Under (b) or
   (c) the phase stops here until MADR D7 and this phase are amended for it, the row `TestSystemPromptOverrideRefused`
   of step 10 and step 8's capability with them.
3. `python3 scripts/check_libs.py --lib acp-go-sdk2 --tag v1.36.0`, at P1's commit: `0 failing checks`. It covers get
   and tidy with no other module moving, build, vet for three OSes, the importing packages' tests, staticcheck, lint,
   apidiff from `v1.35.0` (compatible) and govulncheck.
4. **The pin:**

   ```sh
   go get github.com/maccavelli/acp-go-sdk2@v1.36.0
   go mod tidy
   ```

   `git diff go.mod` is that requirement's line alone; `go.sum` changes by its two lines. The `go` line stays
   `1.27.4`: the library's floor is `1.26.11`. With the pin alone the tree builds and its tests pass, and gobble
   advertises no system prompt, so the SDK refuses every one (MADR F6); steps 5 to 9 then build D7. Pin and feature are
   this phase's one commit (C7).
5. **`session/session.go`:** `CustomSystemPrompt = "gobble.system_prompt"`, its doc naming its data,
   `{"mode": "append", "text": …}`.
6. **`prompt/sections.go`:** `Options.Client string`; `Build` adds a `client_prompt` section, tagged, after `addendum`
   and before `project_context`, when `Client` is set, and nothing when it is not; a check of a client's text, blank or
   holding `</client_prompt>`, in the manner of `CheckSection` (MADR D7, "What gobble cannot use").
7. **`acpserver/systemprompt.go` (new):** the joining of a `*acp.SystemPrompt`'s text blocks with a blank line, step
   6's check, and a refusal through `invalidParams` (`acpserver/agent.go:426`); the `gobble.system_prompt` entry's
   writer and reader, as `acpserver/personality.go:26-42` writes and reads `gobble.personality`.
8. **`acpserver/agent.go` and its neighbours:**
   * `Capabilities()` (`:384-405`): `SystemPrompt: &acp.SessionSystemPromptCapabilities{}`, with no `Mode`.
   * `liveSession` holds the client prompt; `replayPath` (`acpserver/history.go:200`) reads the path's last
     `gobble.system_prompt` into its state, and `restore` (`:525-551`) sets it.
   * `NewSession` (`:433-513`) checks `p.SystemPrompt` before `a.store.Create`, so a refusal writes no file, and
     appends the entry once the log exists, as it appends a name.
   * `open`, `LoadSession` and `ResumeSession` (`:1055-1169`) check the prompt before `a.store.Open`; after `restore`,
     a prompt that differs from the stored one is appended as a new entry and taken by the live session; an absent or
     equal one writes nothing.
   * `sections` (`acpserver/sysprompt.go:54-109`) passes the session's prompt as `prompt.Options.Client`.
9. **Docs:** `README.md:155` and `docs/architecture.md:189` name `v1.36.0`; `README.md:59-60`'s system-prompt bullet
   and `docs/architecture.md`'s `prompt` and `acpserver` rows (`:118`, `:124`) name the client's `append` prompt, its
   `client_prompt` section, and that it is kept with the session.
10. **Tests,** each seen failing (C3) on its planted break, and, where it compiles, on the tree after step 4 alone. A
    request the SDK's typed client would refuse to send (an unadvertised mode, a non-text block, empty content) is
    sent as a raw JSON-RPC frame. The model is `llmtest`'s scripted one, and the system message is read with
    `llmtest.AssertSystem` (`llm/llmtest/assert.go:14`).

    | Test | Asserts | Planted break |
    | :--- | :--- | :--- |
    | `TestCapabilitiesAreHonest` (`acpserver/agent_test.go:137`, extended) | `sessionCapabilities.systemPrompt` is advertised, and `systemPrompt.mode.override` is not (Q2 (a)) | `Capabilities()` without `SystemPrompt`; `Mode.Override` set |
    | `TestSystemPromptReachesTheModel` | a `session/new` with an `append` prompt of two text blocks: the first turn's request has a system message holding `client_prompt` with both texts, joined by a blank line, after `addendum` and before `project_context`, and every section the session has without it | the client's text dropped before `Build`; the section placed before `personality` |
    | `TestSystemPromptKeepsGobblesOwn` | with a `SYSTEM.md`, an `APPEND_SYSTEM.md`, a context file and a personality: the client's `append` keeps all four, and the tools and rules sections stay as they are without it | a client prompt read as `Custom`, so the preamble, tools and rules give way |
    | `TestSystemPromptOverrideRefused` | under Q2 (a): an `override` prompt on `session/new`, `session/load` and `session/resume` is `-32602` naming `sessionCapabilities.systemPrompt.mode.override`, and nothing is created, loaded or resumed (the store and `session/list` as before) | `Mode.Override` advertised, so the requests are served |
    | `TestSystemPromptRefusedContent` | `append` with blank text, and with text holding `</client_prompt>`: `-32602` with `ErrorData.Reason`, no session file written, nothing loaded; the SDK's own refusals, empty `content`, an `image` block and mode `_review`, are `-32602` with gobble's handler not called (its call count is zero) | the blank check removed, so a session is created; the check run after `a.store.Create`, so a file is left |
    | `TestSystemPromptPersists` | a session created with a prompt and one turn, closed, then `session/load` with no `systemPrompt`: the next turn's system message still holds it, no new `gobble.system_prompt` entry is written, and no `gobble.prompt` patch is; `session/resume` the same | the entry not written at `session/new`, so the reloaded session's next turn sends a patch that removes `client_prompt` |
    | `TestSystemPromptBeforeFirstTurn` | a prompt given at `session/new`, the session closed before any turn, then loaded with none: its first turn's baseline holds it | the prompt kept in memory only |
    | `TestSystemPromptChangedOnLoad` | `session/load` with a different text: one new entry; the next turn carries a `gobble.prompt` patch for `client_prompt` and its update message; after `/compact`, the new baseline's system message holds the new text | a different prompt on load ignored |
    | `TestNoSystemPromptUnchanged` | a session with no `systemPrompt` sends the system message it sent before this phase, byte for byte (`prompt/testdata/system.golden` unchanged) | `Build` adding an empty `client_prompt` section |

11. **The goldens.** `acpserver/testdata/session.golden` changes by `"systemPrompt":{}` in the `initialize` answer's
    `sessionCapabilities`, and by nothing else; its diff is quoted in the record. `acpclient/acptest/testdata/initialize.golden`
    is a stub agent's answer and does not change. Any other golden that changes is a deviation (C4). magic-cli-remote's
    gobble wire fixtures hold this `initialize` answer; they are recaptured at `v0.6.0` in any case (magic-cli-remote
    `docs/decisions/0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`, Gate G2 and P7's
    L4), and nothing here waits for them.
12. **Gates** (C6), with `make pre-add-check` on every changed Go file. Staged; the owner commits.

Acceptance: `check_libs.py` passes; the pin moved its own line alone; the MADR records Q1's answer (D5's note) and D7;
step 10's tests pass and each plant failed; `session.golden` changed by the one member; the gates pass on three hosts.
magic-cli-remote's move to `v1.36.0` is not this phase's: MADR D3's lockstep is read at `v0.6.0`.

#### P2 as first written (superseded 2026-11-11, later; kept as it was written)

1. **C1.** Then the tag: `gh api repos/maccavelli/acp-go-sdk2/tags` lists `v1.36.0`, and `go list -m
   github.com/maccavelli/acp-go-sdk2@v1.36.0`, run from a scratch directory, succeeds through the proxy.
2. **The owner answers Q1** (MADR D5) before `go.mod` is edited. The answer is written into the MADR's D5, dated.
   * **(a), do not advertise:** steps 3 to 8 run as written.
   * **(b), advertise and apply the prompt:** this phase stops here. Applying a client's prompt gets a record of its
     own, or an amendment of this pair, before any code. The owner says whether the pin lands under (a) meanwhile.
   * **(c), advertise and ignore:** the same stop, since the MADR recommends against it; step 5's "advertise" row is
     then the test.
3. `python3 scripts/check_libs.py --lib acp-go-sdk2 --tag v1.36.0`, at P1's commit: `0 failing checks`. It covers get
   and tidy with no other module moving, build, vet for three OSes, the importing packages' tests, staticcheck, lint,
   apidiff from `v1.35.0` (compatible) and govulncheck.
4. **The pin:**

   ```sh
   go get github.com/maccavelli/acp-go-sdk2@v1.36.0
   go mod tidy
   ```

   `git diff go.mod` is that requirement's line alone; `go.sum` changes by its two lines. The `go` line stays
   `1.27.4`: the library's floor is `1.26.11`.
5. **Q1's test,** in `acpserver`, seen failing on each planted break (C3). It names no identifier that `v1.35.0` lacks:
   it reads the capability from the `initialize` answer's JSON, and sends each session request as a raw JSON-RPC frame,
   since acp-go-sdk2's typed client refuses to send `systemPrompt` to an agent that did not advertise it.

   | Answer | Test | Asserts | Planted break |
   | :--- | :--- | :--- | :--- |
   | (a) rely on the refusal | `TestSystemPromptIsRefused` | `initialize`'s `agentCapabilities.sessionCapabilities` has no `systemPrompt`; a `session/new`, a `session/load` and a `session/resume` that each carry `systemPrompt` are answered `-32602`; no session is created, loaded or resumed (the store and `session/list` are as before); the same three requests without the field are served | `Capabilities()` advertising `systemPrompt: {}`, so the three are served; acp-go-sdk2 pinned back to `v1.35.0` in the scratch copy, where the field is dropped and a session is created |
   | (b) or (c) advertise | `TestSystemPromptIsAdvertised` | `initialize` advertises `systemPrompt: {}`; a `session/new` with an `append` text prompt is served, and under (b) its text reaches the model's system prompt (`llmtest.AssertSystem`); an `override` prompt is refused `-32602`, since `mode.override` is not advertised | the capability removed; under (b), the client's text dropped before the model call |

6. **Nothing else on the wire moves.** `acpserver`'s and `acpclient/acptest`'s golden transcripts are unchanged. A
   golden that changes is a deviation (C4).
7. `README.md:155` and `docs/architecture.md:189` name `v1.36.0`. `v1.35.0` remains only in records, which keep their
   history; 0051-MADR's note of 2026-11-11 under U8 already gives its cited lines at `v1.36.0`.
8. **Gates** (C6), with `make pre-add-check` on the new test file. Staged; the owner commits.

Acceptance: `check_libs.py` passes; the pin moved its own line alone; Q1's answer is in the MADR; its test passes and
each plant failed; the gates pass on three hosts. magic-cli-remote's move to `v1.36.0` is not this phase's: MADR D3's
lockstep is read at `v0.6.0`.

### P3 — go-selfupdate-lib `v1.26.2` (MADR D3, D4)

1. **C1.** Then `go list -m github.com/maccavelli/go-selfupdate-lib@v1.26.2` from a scratch directory, and `python3
   scripts/check_libs.py --lib go-selfupdate-lib --tag v1.26.2`, at P2's commit: `0 failing checks`, apidiff from
   `v1.26.0` compatible.
2. **The pin:**

   ```sh
   go get github.com/maccavelli/go-selfupdate-lib@v1.26.2
   go mod tidy
   ```

   That `go.mod` line alone, and its two `go.sum` lines.
3. **CI's workflow pins.** `gh api repos/maccavelli/go-selfupdate-lib/git/ref/tags/v1.26.2` is peeled to its commit
   again; it must be `e62d77cddaf264cc6ec7bbc4f39b90322542f752`, and any other commit is a deviation.
   `.github/workflows/ci.yml:191` and `:203` then name `@e62d77cddaf264cc6ec7bbc4f39b90322542f752 # v1.26.2`, and
   `7c677842` no longer appears in the file. `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12`, as CI runs
   it: exit 0.
4. `README.md:162` and `docs/architecture.md:193` name `v1.26.2`.
5. **The messages.** `go test -count=1 -run 'TestUpdate' ./internal/cli/` passes with no test edited. A test whose
   expected text must change for `v1.26.1`'s or `v1.26.2`'s wording is a deviation (C4): the MADR's F7 found none.
6. **Gates** (C6). No Go file changes. Staged; the owner commits.

Acceptance: `check_libs.py` passes; the pin moved its own line alone; `ci.yml`, `README.md` and `docs/architecture.md`
name `v1.26.2`; the gates pass on three hosts.

### P4 — CI on three runners, and close-out (MADR Confirmation)

1. **After the owner's push of P1 to P3:** the CI run for `main`'s head is read with `gh run view <id> --json jobs`.
   `validate (ubuntu-24.04)`, `validate (macos-15)`, `validate (windows-2025)`, `test-isolation`, `sbom` and `build`
   conclude `success`; `release` is `skipped`. The run's id and each job's conclusion go into the record.
2. A job that fails is read from its log, not re-run to green. If it is the Windows lock test, C5 applies; anything
   else is a deviation (C4).
3. Each Confirmation item is recorded with its evidence in the MADR's Completion section. This PLAN becomes `complete`;
   `docs/README.md`'s two status cells follow.

Acceptance: Confirmation 1 to 9 met, or each unmet item named with its reason.

## Verification

Each command's exit status is captured before any filter.

| Confirmation | Command or reading | Where |
| :--- | :--- | :--- |
| 1 the `go` lines; old toolchain refused; the base reports GO-2026-6859 | `git diff go.mod acpext/go.mod`; `go mod tidy -diff`; `go -C acpext mod tidy -diff`; the scratch runs under `GOTOOLCHAIN=go1.27.3` | P1 steps 2, 3, 6 |
| 2 `check_libs.py` and the docs name 1.27.4 | `python3 scripts/check_libs.py --self-test`; `git grep -n '1\.27\.3' -- . ':!docs/decisions' ':!docs/reports'` shows only F5's "no" row | P1 step 4 |
| 3 the pins, nothing else moved | `git diff go.mod go.sum` per phase | P2 step 4, P3 step 2 |
| 4 `check-libs` per library | `python3 scripts/check_libs.py --lib <name> --tag <tag>` | P2 step 3, P3 step 1 |
| 5 Q1 answered, and its test | `go test -count=1 -run 'TestSystemPrompt' ./acpserver/`, and each plant | P2 steps 2, 5 |
| 5, *(2026-11-11, later)* D7's tests | `go test -count=1 -run 'TestSystemPrompt\|TestNoSystemPromptUnchanged\|TestCapabilitiesAreHonest' ./acpserver/`; `go test -count=1 ./prompt/`; each plant; the diff of `acpserver/testdata/session.golden` | P2 steps 10, 11 |
| 6 CI and docs name one version | `git grep -n -E 'v1\.26\.[02]\|v1\.3[56]\.0\|7c677842\|e62d77c' -- README.md docs/architecture.md .github` | P2 step 7, P3 steps 3, 4 |
| 7 the gates on three hosts | `make preflight`; `make race`; `make check-test-isolation`; the hosts' `go test` runs | each phase's gates |
| 8 CI green on Linux, macOS and Windows | `gh run view <id> --json jobs` | P4 step 1 |
| 9 the notes in 0004, 0023 and 0042 | `make check-records markdownlint` | P1 step 5 |

## Rollout and Rollback

* **Rollout.** Three commits on `main`, each pushed by the owner. P1 alone turns CI green. The pins and the Go fix
  reach users with the next gobble release, `v0.6.0`; no tag is made here (MADR D6). 0051-PLAN's P2 and 0052-PLAN's
  phases then start from `main` as P3 leaves it.
* **Rollback of P3:** revert its commit. `go.mod`, the two workflow pins and the docs return to `v1.26.0` together.
* **Rollback of P2:** revert its commit, test included. gobble is back on `v1.35.0`. If magic-cli-remote has moved to
  `v1.36.0` by then, the two differ until one of them moves again, which matters at `v0.6.0` and not before (MADR D3).
  *(2026-11-11, later: the revert also takes the capability away. A session saved meanwhile with a client prompt
  keeps its `gobble.system_prompt` entry, which the reverted build does not read; its baseline still holds the
  `client_prompt` section, so that build's next turn sends a patch that removes it. Nothing else of the file is lost.)*
* **Rollback of P1:** reverting it returns `go.mod` to `go 1.27.3`, and CI's govulncheck fails again. It is reverted
  only to be replaced, for instance by a newer patch release of Go.
* **Nothing published is undone.** No tag is made, and acpext's `go` line reaches a consumer only with
  `acpext/v0.5.0`.

## Execution record

Not started. The MADR was accepted by the owner on 2026-11-11; this PLAN awaits the owner's approval. Writing the pair
changed no file but the pair's two: `docs/README.md` has no 0054 rows yet (P0 step 2).

### Corrections of 2026-11-11 (records only; no phase has started)

The owner's rulings of 2026-11-11, later that day, relayed by the cross-repository coordinator. Each is a dated note
beside the text it changes.

* **Goal and Scope:** Q1 answered; P2's files for MADR D7; applying a client's prompt is in scope, `override` is not
  until the owner answers Q2; no `v0.5.1`.
* **P2 rewritten** around MADR D7, the phase as first written kept below it: the pin, then the `client_prompt`
  section, the `gobble.system_prompt` entry, load and resume, gobble's own refusals, nine tests, and the one golden
  that changes. It is feature work, by the owner's choice.
* **Verification** row 5's companion, and **the rollback of P2.**
