---
status: accepted
date: 2026-11-11
decision-makers: repository owner
consulted: "this repository at 17d1605 (main, level with origin/main, 2026-11-11); CI runs 43042877946 and 43080286906 and the failed-step log of the second; acp-go-sdk2 at v1.35.0 and v1.36.0, read from a clone with git show, and its docs/reports/0054-REPORT-release-notes-v1-36-0.md at the tag; go-selfupdate-lib at v1.26.0, v1.26.1 and v1.26.2, its CHANGELOG.md, tags and workflows; magic-cli-remote's go.mod at its origin/master eefb7c25; a scratch build and an elicitation probe reported by the cross-repository session of 2026-11-11, not re-run here; later on 2026-11-11, the owner's rulings on Q1, a patch release and D2, relayed by the cross-repository coordinator, and for D7 acp-go-sdk2's docs/decisions/0054-MADR-adopt-acp-schema-1-31-0-and-client-system-prompts.md and its PLAN, its README's System prompts section, and types_gen.go at v1.36.0, with this repository's prompt/sections.go, acpserver/sysprompt.go and acpserver/agent.go at 17d1605"
informed: "0051-PLAN and 0052-PLAN, which start from the base this record leaves; magic-cli-remote, which moves acp-go-sdk2 in step with gobble and imports acpext"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Move Go to 1.27.4 for GO-2026-6859, then pin acp-go-sdk2 v1.36.0 and go-selfupdate-lib v1.26.2

## Context and Problem Statement

`main`'s CI has failed since 2026-11-11: govulncheck reports GO-2026-6859 in Go 1.27.3's `net/http`, and CI installs the
Go that `go.mod` names. `build` and `release` need `validate`, so no release can be cut, and the sign-in work of
[0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
and [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
would start from a red base. Two sibling libraries have also released since
[0050-MADR-refresh-sibling-pins-to-current-releases.md](0050-MADR-refresh-sibling-pins-to-current-releases.md) pinned
them. On 2026-11-11 the owner approved this record: move Go to 1.27.4, and refresh acp-go-sdk2 and go-selfupdate-lib,
before 0051-PLAN's P2. The same day the owner approved moving gobble-cli and magic-cli-remote together from acp-go-sdk2
`v1.35.0` to `v1.36.0` before gobble's `v0.6.0`.

### What holds at `17d1605` (read 2026-11-11)

* **F1. CI is red for two pushes, and green for the one before.**
  * Run `43042877946`, the push of `86ad4a7`: `validate` failed on `ubuntu-24.04`, `macos-15` and `windows-2025`, each
    at the step `make preflight`. `sbom` passed; `build` and `release` were skipped.
  * Run `43080286906`, the push of `17d1605`: `validate` failed on `ubuntu-24.04` and `macos-15` at `make preflight`,
    and on `windows-2025` at `go test`, the step before it (F8), so that runner never reached preflight.
    `test-isolation` and `sbom` passed; `build` and `release` were skipped.
  * The Linux and macOS logs of that run, at preflight's pre-add check: "Vulnerability #1: GO-2026-6859 … HTTP/1 server
    crash due to MaxBytesReader header race in net/http … Found in: net/http@go1.27.3 … Fixed in: net/http@go1.27.4",
    45 example traces, "Your code is affected by 1 vulnerability from the Go standard library.", then
    `make: *** [Makefile:181: preflight] Error 1` on Linux.
  * Run `42898157216`, the push of `d6c10a5` on 2026-11-10, passed. `go.mod`, `go.sum` and `acpext/go.mod` are the same
    at `d6c10a5` and `17d1605` (`git diff --stat` shows none of them), so what changed is the vulnerability report,
    not gobble's requirements.
* **F2. CI and the release build take their Go from `go.mod`.**
  * `validate` and `test-isolation` set `go-version-file: go.mod` (`.github/workflows/ci.yml:62`, `:144`).
  * go-selfupdate-lib's build workflow, which builds the release, sets `go-version-file` to the caller's `go.mod`
    with `GOTOOLCHAIN: local` (`build-selfupdate-release.yml:107`, `:114` at `v1.26.2`).
  * `go.mod:3` is `go 1.27.3`. So every CI job and every release binary uses Go 1.27.3 until that line moves.
* **F3. The vulnerable code is in gobble's graph.** gobble starts one HTTP/1 server: the loopback listener of a
  browser sign-in, in go-llmprovider-sdk. Two of the 45 traces name it: `llm/provider/signin.go:176`,
  "`provider.SignIn` calls `wizard.ResolveCredential`, which eventually calls `http.ServeMux.HandleFunc`", and the same
  call reaching `http.Server.Shutdown`. The listener binds a loopback address for the length of a sign-in.
* **F4. The development hosts already run Go 1.27.4.** `go version` on the macOS host gives `go1.27.4` (2026-11-11).
  0049-PLAN's P5b-1 record of 2026-11-11 gives `go1.27.4` on the Linux machine and the Windows laptop
  (`0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md`, "P5b-1 record", step 5). A local gate
  therefore analyses Go 1.27.4's standard library while CI analyses 1.27.3's. That reading is not measured here; the
  PLAN's P1 measures it both ways on a scratch copy.
* **F5. Where the Go version is written** (`git grep` over the tracked files outside the records):

  | Place | Text | Moves |
  | :--- | :--- | :--- |
  | `go.mod:3` | `go 1.27.3` | yes |
  | `acpext/go.mod:3` | `go 1.27.3` | yes (D2) |
  | `scripts/check_libs.py:124` | `GOTOOLCHAIN="go1.27.3"` | yes |
  | `AGENTS.md:16` | "Requires Go 1.27.3." | yes |
  | `docs/architecture.md:9` | "at Go 1.27.3" | yes |
  | `docs/guides/developing.md:9` | "**Go 1.27.3.** … `go version go1.27.3 …`" | yes |
  | `docs/guides/developing.md:36` | a `gobble version` transcript of 2026-10-07, `go: 1.27.2` | no: it shows that build |
  | `internal/cli/update_test.go:82` | a fixture's `Go: "1.27.3"` | no: a test value, not a requirement |
  | `internal/buildinfo/buildinfo_test.go`, `internal/cli/main_test.go:211`, `internal/cli/testdata/version.*.golden` | fixtures' `1.27.1` | no |
  | `session/doc.go:8`, `docs/README.md:25` | "Go 1.27.1": a comment on what that release changed, and 0004-MADR's title | no |

  `.github/workflows/ci.yml` names no Go version of its own (F2).
* **F6. acp-go-sdk2 `v1.36.0`** (tag `0e68b2b`, commit `3be7af6`, 2026-11-11; its
  `docs/reports/0054-REPORT-release-notes-v1-36-0.md` at the tag).
  * **Go.** `go 1.26.11`, up from `1.26.10`, and `toolchain go1.27.4`, for the same GO-2026-6859. Its floor is below
    gobble's `go` line, so pinning it moves no `go` line here. Its requirements are unchanged.
  * **API.** Its `make api` against `v1.35.0`: "0 incompatible, 57 compatible", 11 in the root package, all for the
    client system prompt, and 46 in package `unstable`, which gobble does not import.
  * **Schema.** The stable ACP schema moves from `1.25.0` to `1.31.0` (`schema/version`).
  * **Files gobble's records cite.** `connection.go` and `middleware.go` do not differ between the tags.
    `agent_gen.go` differs in two doc-comment links. `types_gen.go` and `schema/schema.json` move; 0051-MADR's note
    of 2026-11-11 under U8 gives each cited line at `v1.36.0`.
  * **One change on the wire.** A `session/new`, `session/load` or `session/resume` that carries `systemPrompt` is
    answered `-32602` by an agent that does not advertise `sessionCapabilities.systemPrompt`, and the agent's handler
    is not called. `v1.35.0`'s request types have no such field and decode with `encoding/json`, which drops a member
    it does not know, so the same request was served there with the prompt ignored.
  * **gobble today** advertises `additionalDirectories`, `close`, `list` and `resume`, and no `systemPrompt`
    (`acpserver/agent.go:395-400`), and no gobble code reads a client's system prompt. magic-cli-remote sends none:
    no Go file there names `systemPrompt` (its tree, 2026-11-11).
  * **Reported, not re-run here.** The cross-repository session of 2026-11-11 built `17d1605` in a scratch copy with a
    `replace` to the `v1.36.0` tree and ran the tests of `acpserver`, `acpclient` and `acpclient/acptest`, which passed;
    and ran a probe of a request-scoped elicitation sent from inside `authenticate`, which passed on both tags. The
    PLAN's own `check_libs.py` run is this record's evidence.
* **F7. go-selfupdate-lib `v1.26.2`** (tag `5df0abb`, commit `e62d77cddaf264cc6ec7bbc4f39b90322542f752`, as GitHub
  peels it; its `CHANGELOG.md`).
  * **`v1.26.1`, changed (2026-11-10):** an error names the package once. A run's error was
    `selfupdate: <product>: selfupdate: <reason>`, and a wrapped sentinel repeated the prefix again; now the wrapped
    error's own leading `selfupdate:` is dropped. A sentinel printed alone keeps its prefix, and `errors.Is` and
    `errors.As` match as before.
  * **`v1.26.2`, fixed (2026-11-10):** a managed install's failure names its phrase once, where it read
    "managed install failed: managed install failed".
  * **No exported name changes.** The one new package is `internal/errprefix`. `go.mod` is the same at the three tags
    (`go 1.26.10`, `toolchain go1.27.3`), so pinning it moves no `go` line and no other requirement.
  * **No workflow changes.** No file under `.github/workflows` differs between `v1.26.0` and `v1.26.2`.
  * **What gobble reads.** `errors.Is` with `ErrSymlinkInvocation` (`internal/cli/update.go`), which holds. Its tests
    check three pieces of text: "refusing to update through a symlink", "links to" and "outside allowed roots"
    (`internal/cli/update_test.go:347-348`, `:372`). Each is still in the library's source at `v1.26.2`. What a user
    sees change: `gobble update`'s error lines stop repeating `selfupdate:`.
  * **CI pins the library's two reusable workflows** at `v1.26.0`'s commit
    (`.github/workflows/ci.yml:191`, `:203`, `7c677842… # v1.26.0`). 0048-MADR Q6 (a) moves them with the module.
* **F8. One Windows test failed in run `43080286906`, apart from all of the above.** It is recorded under "Observed,
  not decided" in More Information.
* **F9. Docs naming the two pins:** `README.md:155` and `:162`; `docs/architecture.md:189` and `:193`.
* **F10. The other siblings are current.** go-llmprovider-sdk `v1.29.0` and go-tui-lib `v2.12.0` are each library's
  newest tag. go-llmprovider-sdk's next minor is 0051-PLAN P2's to pin. acpext stays at `v0.3.0`, on purpose
  (0052-MADR, QA (a)).
* **F11. magic-cli-remote** requires `go 1.27.4`, go-selfupdate-lib `v1.26.2`, acp-go-sdk2 `v1.35.0` and acpext
  `v0.3.0` (its `go.mod` at `eefb7c25`). Its `docs/decisions/0179-MADR-gobble-native-acp-provider.md` gate 3 asks for
  "Both repositories on one acp-go-sdk2 tag".

## Decision Drivers

* **A green `main` before more work lands on it.** 0051-PLAN and 0052-PLAN run `make preflight` at every phase and need
  CI's verdict at their close.
* **`go.mod` is where gobble's Go is stated** (0004-MADR's amendments of 2026-10-08 and 2026-10-21): CI and the release
  build read it, and a toolchain older than a fix must not build gobble unwarned.
* **Current releases, each checked by `make check-libs` before it is pinned** (0035-MADR D2), and **one version named
  everywhere**: the module, the CI workflows and the docs (0048-MADR Q6 (a)).
* **Lockstep with magic-cli-remote on acp-go-sdk2** (its 0179-MADR gate 3; the owner's ruling of 2026-11-11 that
  `v0.6.0` is the gobble release that must pin the same tag as magic-cli-remote).
* **Small steps that revert alone:** the toolchain, and each pin, in a commit of its own.
* **No decision hidden in a pin:** the one behaviour `v1.36.0` changes is decided in the open, with a test.

## Considered Options

* **A. One record, three steps:** Go 1.27.4 first, then acp-go-sdk2 `v1.36.0`, then go-selfupdate-lib `v1.26.2`, each
  its own commit, all before 0051-PLAN P2.
* **B. Go 1.27.4 only.** The pins wait for 0051-PLAN P2, or for a later refresh.
* **C. Keep `go 1.27.3`, and give CI Go 1.27.4** by a `toolchain go1.27.4` line or a pinned `go-version`.
* **D. Fold all of it into 0051-PLAN P2.**

## Decision Outcome

Chosen option: "A. One record, three steps", because it is what the owner approved on 2026-11-11, it turns CI green
with the smallest first commit, and it leaves 0051-PLAN P2 one thing to move, go-llmprovider-sdk. The Go move must come
first for a mechanical reason as well: `check_libs.py` runs govulncheck on a clone of the committed base under the
toolchain it names (`scripts/check_libs.py:124`, `:327`), so at `17d1605` it would fail every library on GO-2026-6859
whatever the tag.

*(2026-11-11, later: by the owner's answer to Q1 (D5's note), the second step, acp-go-sdk2 `v1.36.0`, also carries
D7's feature: gobble advertises client system prompts and honours them. That makes the PLAN's P2 feature work, not
only a pin move, by the owner's choice. The three steps and their order stand.)*

### D1. Go 1.27.4 in the root module

* `go.mod`'s `go` line becomes `1.27.4`. No `toolchain` line is added: one equal to the `go` line says nothing more
  (0004-MADR, amendment of 2026-10-21).
* CI and the release build follow `go.mod` (F2); no workflow file changes for this.
* `scripts/check_libs.py:124`'s `GOTOOLCHAIN` becomes `go1.27.4`. `AGENTS.md:16`, `docs/architecture.md:9` and
  `docs/guides/developing.md:9` name Go 1.27.4. F5's "no" rows stay as they are.
* Dated notes go into the records that state the toolchain: 0004-MADR's Toolchain line, and 0023-MADR's Go row.

### D2. acpext's `go` line moves with the root's, and no acpext tag is made for it

* **It moves.** `acpext/go.mod`'s `go` line becomes `1.27.4`. 0042-MADR D1 set it "as the root module, for fleet
  lockstep", and nothing has decided otherwise. After D1, CI runs acpext's tests and govulncheck under Go 1.27.4 only,
  so a `go 1.27.3` line there would state a floor that nothing tests. A dated note goes into 0042-MADR D1.
* **Not for the vulnerability's sake.** acpext imports one package, `encoding/json/jsontext`, and no `net/http`;
  GO-2026-6859 does not reach it. The reason is the lockstep rule, and a floor that CI checks.
* **No tag now.** The root requires acpext at a tag from the module proxy, never the tree's copy (0042-MADR D5), so the
  line changes no build of the root. It is no change to the contract or the wire, which is what 0042-MADR D5 tags.
  The tree's `acpext/` is `acpext/v0.4.0`'s content (`git diff acpext/v0.4.0 HEAD -- acpext` is empty), so a tag made
  now would be a `v0.4.1` holding that content and this line, which no consumer needs: the root pins `v0.3.0` on
  purpose, and magic-cli-remote's next acpext pins are `v0.4.0` and then `v0.5.0`.
* **The next tag carries it.** `acpext/v0.5.0` (0051-PLAN P3) is cut from a tree whose `go` line is `1.27.4`. A module
  that requires it needs Go 1.27.4 or newer; magic-cli-remote, its one consumer, is already there (F11). 0051-PLAN P3
  names the new floor in that tag's notes.
* **The root's acpext pin does not move.** It stays `v0.3.0`.

*(2026-11-11, later: confirmed. The owner accepted D2 as written: `acpext/go.mod` moves to `1.27.4` with the root, no
acpext tag is made for it, and `acpext/v0.5.0` carries it.)*

### D3. The pins

* acp-go-sdk2 `v1.35.0` → `v1.36.0`; go-selfupdate-lib `v1.26.0` → `v1.26.2`. go-llmprovider-sdk, go-tui-lib and
  acpext stay (F10).
* Each move follows `scripts/check_libs.py --lib <name> --tag <tag>` giving `0 failing checks` at the base D1 leaves,
  and is `go get <module>@<tag>` and `go mod tidy`, with that `go.mod` line and its two `go.sum` lines changing and
  nothing else. Any other line that moves is a deviation.
* **Lockstep.** gobble's `main` and magic-cli-remote both pin acp-go-sdk2 `v1.36.0` before gobble's `v0.6.0` is tagged;
  magic-cli-remote moves by its own record. If either repository's pin moves again before `v0.6.0`, it moves in both,
  or the release waits (0051-PLAN's rollout, note of 2026-11-11).

### D4. CI and docs name the same versions

* `.github/workflows/ci.yml`'s two `uses:` lines (`:191`, `:203`) move to
  `e62d77cddaf264cc6ec7bbc4f39b90322542f752 # v1.26.2`, the tag peeled again before the edit.
* `README.md:155` and `docs/architecture.md:189` name acp-go-sdk2 `v1.36.0`; `README.md:162` and
  `docs/architecture.md:193` name go-selfupdate-lib `v1.26.2`.

### D5. The client system prompt: the pin brings the gate; what gobble advertises is Q1

*(2026-11-11, later: Q1 is answered. The owner ruled that gobble advertises client system prompts and honours them.
That is option (b) in substance, with one difference the owner set: it is decided in this record, as D7, and not in a
record of its own as (b) proposed. Option (a), marked recommended, was not chosen. Q1 and its options are kept below
as they were put. The paragraph that follows stands as the account of what the pin alone does; the PLAN's P2 is
rewritten around D7.)*

With `v1.36.0`, a client that sends `systemPrompt` on a session request gets `-32602` from gobble, where the prompt was
silently ignored (F6). Neither answer applies the client's prompt. The question is which of them gobble gives, and
whether gobble takes on applying the prompt at all. The PLAN's P2 does not edit `go.mod` before the owner answers it.

* **Q1. What gobble does about `sessionCapabilities.systemPrompt`.**
  * **(a) Do not advertise it; rely on the SDK's refusal.** *(recommended)* A client is told `-32602`, naming the
    capability, in place of a prompt that silently did nothing. No client in this fleet sends the field (F6). A test
    holds it: gobble's `initialize` advertises no `systemPrompt`, and a `session/new`, `session/load` and
    `session/resume` that carry one are each refused `-32602` with no session created, loaded or resumed. Cost: a
    client that sent the field to gobble and did not mind it being ignored now fails until it stops sending it.
  * (b) Advertise `{}` (append mode) and append the client's text to gobble's system prompt. This is a feature, not a
    pin: it touches `prompt`, the personalities and prompt caching, and needs a record of its own before any code. Cost:
    the pin waits for that record, or lands first under (a) and (b) follows.
  * (c) Advertise it and ignore the prompt, to keep `v1.35.0`'s behaviour. Not recommended: gobble would claim a
    capability it does not have.

### D6. No release is tagged by this record

This repository's convention does not require one. The Go move of 2026-10-21 was one commit and no tag (0004-PLAN's
entry of that date), and 0050-PLAN put "a push or a tag" out of scope: "The pins reach users with the next gobble
release." That release is `v0.6.0`, the owner's to tag after 0051-PLAN and 0052-PLAN. No acpext tag is made either
(D2). Whether an earlier patch release should carry the Go fix alone is in "Not decided here".

*(2026-11-11, later: decided by the owner. **No `v0.5.1` is cut;** the fix reaches users in `v0.6.0`. The consequence,
stated plainly: gobble's released `v0.5.0` binaries were built by CI with Go 1.27.3 (F2), so they carry GO-2026-6859
in `net/http`, which they reach through the OAuth loopback listener of a browser sign-in (F3). An installed `v0.5.0`
binary stays exposed until its user moves to `v0.6.0`, whenever that is tagged, and this record does nothing to
shorten that. What F3 says of the exposure is unchanged: the listener binds a loopback address for the length of a
sign-in.)*

### D7. gobble advertises client system prompts and honours them (the owner's answer to Q1)

*(Added 2026-11-11, later, by the owner's ruling.)* This is feature work inside a pin record, by the owner's choice.
What acp-go-sdk2 `v1.36.0` defines is set out first, since gobble relies on it; what gobble decides follows; the one
choice the schema leaves open, `override`, is Q2, which awaits the owner.

* **What the schema and the SDK define** (acp-go-sdk2 `docs/decisions/0054-MADR-adopt-acp-schema-1-31-0-and-client-system-prompts.md`
  F3 to F5, D5 and D6; its `README.md`, "System prompts"; `SystemPrompt`, `SystemPromptMode` and the three
  `SessionSystemPrompt…Capabilities` types in `types_gen.go` at `v1.36.0`, read with `git show v1.36.0:types_gen.go`):
  * `systemPrompt` on `session/new`, `session/load` and `session/resume` is `{mode, content}`, both required;
    `content` is an array of `ContentBlock`, and `mode` is `append`, `override`, or another string.
  * `sessionCapabilities.systemPrompt: {}` advertises `append`; `systemPrompt.mode.override: {}` adds `override`.
  * `append` adds the content to the agent's own instructions, and the agent "SHOULD give the content system-level
    precedence, for example in the model's system prompt"; where it is placed and how it is formatted is the
    agent's. `override` replaces the agent's "built-in default instructions", changes no tool, MCP server, permission
    or configuration, and may keep "context the Agent supplies from the workspace or environment".
  * On load and resume an omitted or `null` prompt lets the agent "keep a system prompt it stored with the session or
    use its default instructions"; a client should send again the prompt it created the session with, and an agent
    given a different one should apply it and may defer that until it next rebuilds its prompt.
  * An agent that cannot apply a prompt must return an error and must not create, load or resume the session;
    `-32602` is the recommended error.
  * The SDK refuses on receipt, with `-32602` and before gobble's handler is called: a prompt when the capability is
    not advertised; `override` without `mode.override`; any mode the schema does not list, `_`-prefixed, empty and
    `null` included; a content block that is not `text`; and empty `content`. gobble's handler therefore sees only
    `append`, or `override` once advertised, with one or more text blocks.
* **Advertised.** `Capabilities()` (`acpserver/agent.go:384-405`) sets `SessionCapabilities.SystemPrompt` to `{}`,
  `append` mode. `mode.override` is Q2's, below; until the owner answers it, it is not advertised, and the SDK refuses
  an `override` prompt.
* **How an `append` prompt joins gobble's own** (`prompt.Build`, `prompt/sections.go:136-163`; `Agent.sections`,
  `acpserver/sysprompt.go:54-109`). The text of the blocks, joined by a blank line, becomes one new section,
  `client_prompt`, tagged as every section but the preamble is. It goes after `addendum` (the user's
  `APPEND_SYSTEM.md` or `--append-system-prompt`) and before `project_context`, so the sections are: preamble,
  personality, personality_rules, tools, rules, addendum, client_prompt, project_context, skills, cwd. Nothing of
  gobble's is removed or reworded: the preamble or a `SYSTEM.md` or `--system-prompt` in its place, the personality
  and its house rules, the tools and rules sections, the addendum, the context files (`AGENTS.md` and its kin) and the
  skills are what they are without a client prompt. The section is in the system message, so it has the system-level
  precedence the schema asks for. A block's `annotations` and `_meta` are not read.
* **What persists with a saved session.** The prompt is a custom entry in the session's file, `gobble.system_prompt`
  (`session.CustomSystemPrompt`, beside `CustomPersonality` in `session/session.go:41-55`), its data `{"mode":
  "append", "text": …}`, the joined text. `session/new` writes it with the session, as it writes a name given in
  `_meta.gobble` (`acpserver/agent.go:433-513`), and `restore` reads the path's last one, as it reads the personality
  (`acpserver/agent.go:525-551`; `replayPath`, `acpserver/history.go:200`). A fork through `_meta.gobble.forkFrom`
  copies it with the other entries; a `systemPrompt` on that `session/new` then replaces it. The system prompt is
  frozen per session as a baseline in `gobble.prompt` entries, later changes being patches
  (`acpserver/sysprompt.go:256-296`), so the client's section is in the baseline the first turn writes, and the prompt
  cache's prefix holds from turn to turn.
* **`session/load` and `session/resume`** (`acpserver/agent.go:1099-1169`):
  * with no `systemPrompt`, the stored one is kept and nothing is written. The schema allows keeping it or using the
    default; gobble keeps it, since the session's baseline already holds it and dropping it would send the model a
    patch that removes it at the next turn;
  * with the same mode and text, nothing is written;
  * with a different one, a new `gobble.system_prompt` entry is written and the live session takes it. The next turn's
    sections then differ from the baseline's, and gobble's patch path applies the change: a `gobble.prompt` patch and
    its update message, in that turn. The system message holds the new text from the next new baseline, a
    compaction's (`rebase`, `acpserver/sysprompt.go:290-296`). That is the schema's "may defer that until it next
    rebuilds its prompt", with the change reaching the model from the next turn;
  * a client cannot remove a stored prompt on load or resume: omitted and `null` mean "keep", and empty content is
    refused by the SDK. A new session is the way.
* **What gobble cannot use, and refuses itself**, with `-32602` and `ErrorData{SessionID, Reason}` (`invalidParams`,
  `acpserver/agent.go:426-428`), creating, loading or resuming nothing, as the schema requires: a prompt whose joined
  text is blank, white space only; and a text holding `</client_prompt>`, which would close its own section's tag, the
  rule `prompt.CheckSection` applies to a user's section text (0022-MADR D15). The check runs before the store is
  touched, so a refused `session/new` writes no file. Nothing else: gobble sets no length limit of its own, and a prompt
  too large for the model's window fails the first turn as any over-window context does. A non-text block, an
  unlisted mode and empty content never reach gobble (the SDK, above).
* **gobble's own surfaces are unchanged.** `--system-prompt` and `--append-system-prompt` stay options of the
  in-process agent (`internal/cli/flags.go:55-56`, `acpserver.PromptOptions`); gobble's own ACP client sends no
  `systemPrompt`; `gobble acp` gains no flag. A session with no client prompt sends the same system prompt, byte for
  byte, as before.
* **The `initialize` answer changes on the wire:** `agentCapabilities.sessionCapabilities` gains `systemPrompt: {}`.
  `acpserver/testdata/session.golden` changes by that member alone; `acpclient/acptest/testdata/initialize.golden` is a
  stub agent's and does not change. magic-cli-remote's gobble wire fixtures are recaptured against `v0.6.0` in any
  case (magic-cli-remote `docs/decisions/0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`,
  Gate G2 and P7's L4), so this adds no recapture there. magic-cli-remote sends no `systemPrompt` (F6).

#### Q2. `override`: awaiting the owner

*(2026-11-11, later: **answered by the owner: (a).** gobble does not advertise `systemPrompt.mode.override`; it offers
`append` only, and the SDK answers an `override` prompt `-32602`. Adding `override` later is a record of its own. The
question and its options are kept below as they were put.)*

The schema leaves to the agent whether it offers `override`, and what its "built-in default instructions" are. gobble's
nearest analogue is a `SYSTEM.md` or `--system-prompt`, `prompt.Options.Custom` (`prompt/sections.go:91-93`): it
replaces the preamble and leaves out the tools and rules sections, and keeps the personality, its house rules, the
addendum, the context files, the skills and `cwd` (`:136-163`). acp-go-sdk2's 0054-MADR, "For gobble-cli", leaves
"whether personality may stay" to this record. This record does not decide it.

* **(a) Do not advertise `override`; `append` only.** *(recommended)* The SDK answers an `override` prompt `-32602`.
  * Good, because the safety posture does not change: the rules section ("a call to a tool that changes something
    may be declined by the user", "Commands run on …") and the personality's house rules always reach the model.
  * Good, because no client in this fleet asks for it, and codex-acp offers `append` alone (acp-go-sdk2 0054-MADR F7).
  * Good, because `mode.override` can be added later as an addition, breaking no client.
  * Bad, because a client that wants to replace gobble's instructions cannot, and is refused.
* **(b) Advertise it, as gobble's `Custom`.** The client's text takes the preamble's place, and a `SYSTEM.md`'s or
  `--system-prompt`'s with it; the tools and rules sections are left out; the personality, house rules, addendum,
  context files, skills and `cwd` stay.
  * Good, because it reuses a tested path whose effect users of `SYSTEM.md` already know.
  * Good, because the schema lets workspace and environment context stay.
  * Bad, because the personality and its house rules are gobble's own instructions, not workspace context, so
    "replace the built-in default instructions" is only partly done.
  * Bad, because the rules section's lines leave with it: the change of posture the schema's RFD made a double
    opt-in (acp-go-sdk2 0054-MADR F5).
* **(c) Advertise it, replacing every instruction of gobble's own.** The client's text replaces the preamble, the
  personality and its house rules, and the tools and rules sections; the addendum (the user's host file), context
  files, skills and `cwd` stay as workspace and environment context.
  * Good, because it is the closest reading of the schema's words.
  * Bad, because it is a prompt shape gobble has nowhere else, and `/personality` then changes nothing the session's
    model is told.
  * Bad, because of (b)'s safety cost, and more.

Until the owner answers, the PLAN's P2 builds (a), which is what the SDK enforces when `mode.override` is not
advertised. (b) or (c) is an amendment of D7 and of P2 before any of its code.

### Consequences

* Good, because `main`'s CI passes govulncheck again, and a tag can build a release.
* Good, because no toolchain older than the fix builds gobble or acpext unwarned: Go 1.27.3 refuses a `go 1.27.4`
  module.
* Good, because 0051-PLAN's U8 is re-read at the tag gobble will ship, before any sign-in code is written.
* Good, because CI's workflows, the module and the docs name one go-selfupdate-lib version again.
* Neutral, because the development hosts already run Go 1.27.4; the change makes `go.mod` say so.
* Neutral, because `gobble update`'s error lines lose a repeated `selfupdate:`; no test of gobble's reads the old text.
* Bad, because a module that imports gobble's packages, or acpext from `v0.5.0`, needs Go 1.27.4.
* Bad, because between gobble's pin and magic-cli-remote's, the two `main` branches name different acp-go-sdk2 tags.
  Gate 3 reads releases, and no gobble release pins `v1.35.0` or `v1.36.0` yet (`v0.5.0` pins `v1.34.2`), so the gap
  is on `main` only.
* Bad, under Q1 (a), because a client that sends `systemPrompt` to gobble is refused where it was served.
  *(2026-11-11, later: (a) was not chosen. Under D7 such a client is served, its prompt applied, unless it sends
  `override` (Q2) or content D7 refuses.)*
* *(2026-11-11, later.)* Good, because a client's system prompt reaches the model with system-level precedence, and
  stays with a saved session across `session/load` and `session/resume`.
* *(2026-11-11, later.)* Neutral, because a session with no client prompt sends the same system prompt as before.
* *(2026-11-11, later.)* Bad, because the PLAN's P2 is feature work, not only a pin: its commit moves the pin and
  changes the agent's advertisement and prompt, and reverts as one.
* *(2026-11-11, later.)* Bad, because the `initialize` answer changes on the wire, and `session.golden` with it.
* *(2026-11-11, later.)* Bad, because installed `v0.5.0` binaries stay exposed to GO-2026-6859 until their users move
  to `v0.6.0`: the owner chose no `v0.5.1` (D6's note).

### Confirmation

1. `go.mod` and `acpext/go.mod` say `go 1.27.4`, with no `toolchain` line; `go mod tidy` is clean in both modules; a
   Go 1.27.3 toolchain refuses each module, and the base's tree under Go 1.27.3 reports GO-2026-6859, each seen on a
   scratch copy.
2. `scripts/check_libs.py` names `go1.27.4`, and its `--self-test` passes. `AGENTS.md`, `docs/architecture.md` and
   `docs/guides/developing.md` name Go 1.27.4.
3. `go.mod` requires acp-go-sdk2 `v1.36.0` and go-selfupdate-lib `v1.26.2`; go-llmprovider-sdk `v1.29.0`, go-tui-lib
   `v2.12.0` and acpext `v0.3.0` are unchanged; no other module moved.
4. `check_libs.py` for each moved library, at its tag: `0 failing checks`.
5. Q1 is answered and recorded. Under (a), its test passes and is seen failing on each planted break the PLAN names.
   *(2026-11-11, later: Q1 is answered (D5's note). In its place: D7's tests pass, each seen failing on the PLAN's
   planted breaks; and Q2 is answered and recorded, or P2 is built under its (a) and says so.)*
6. `ci.yml`'s two workflow pins name `v1.26.2`'s commit; `README.md` and `docs/architecture.md` name the two tags.
7. `make preflight`, `make race` and `make check-test-isolation` pass on macOS; `go test ./...`, the race tests and
   acpext's tests pass on the Linux machine and the Windows laptop.
8. After the owner's push, `main`'s CI run passes `validate` on `ubuntu-24.04`, `macos-15` and `windows-2025`, with
   `test-isolation`, `sbom` and `build`; `release` is skipped, as on any untagged push.
9. The notes in 0004-MADR, 0023-MADR and 0042-MADR are made, each quoting what it amends.

### Completion (2026-11-11)

Each item below is checked against
[0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md](0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md)'s phase records. All
nine are met. The commits are P0 and P1 `88655c5`, P2 `f441c73` and P3 `b6adb5a`, pushed by the owner together; P4 is
the commit that adds this section.

1. **Met** (`88655c5`; P1 record items 2, 3, 6 and 7). Both `go.mod` files say `go 1.27.4`, one line each, and neither
   has a `toolchain` line. `go mod tidy` exit 0 in both modules, with `go.sum` unchanged, and `make preflight`'s tidy
   check passed. On scratch copies, Go 1.27.3 refused each module ("go.mod requires go >= 1.27.4", exit 1), and the
   base's tree under Go 1.27.3 reported GO-2026-6859 (govulncheck exit 3).
2. **Met** (`88655c5`; P1 record items 4 and 7). `scripts/check_libs.py:124` sets `GOTOOLCHAIN="go1.27.4"`, and its
   `--self-test` exit 0. `AGENTS.md:16`, `docs/architecture.md:9` and `docs/guides/developing.md:9` name Go 1.27.4.
   The PLAN's Verification row 2 `git grep` also lists `docs/README.md:25` and `:26`, the 0004 rows' status cells that
   name the earlier move as history; this item does not ask about them, and they were left (P1 record item 4).
3. **Met** (`f441c73`, P2 record item 4; `b6adb5a`, P3 record item 3). `go.mod` requires acp-go-sdk2 `v1.36.0` and
   go-selfupdate-lib `v1.26.2`; each `go get` moved its own requirement and its two `go.sum` lines alone.
   go-llmprovider-sdk `v1.29.0`, go-tui-lib `v2.12.0` and acpext `v0.3.0` are unchanged: `git diff 17d1605 b6adb5a --
   go.mod` shows the `go` line and the two requirements, and nothing else.
4. **Met** (P2 record item 3, at `88655c5`; P3 record item 2, at `f441c73`). `check_libs.py`: `0 failing checks` for
   acp-go-sdk2 `v1.36.0` and for go-selfupdate-lib `v1.26.2`, its `api` check `ok` in each.
5. **Met,** in its amended form (`f441c73`; P2 record items 2, 10 and 11).
   * Q1 is answered (D5's note) and carried out as D7. Q2 is answered (a) and recorded (Q2's note), so P2 built
     `append` only.
   * D7's nine tests pass, with `TestBuildClientPrompt` and `TestCheckClient` for `prompt`. Each failed on its planted
     break on a scratch copy, sixteen plants in all, none a build failure.
   * `acpserver/testdata/session.golden` changed by `"systemPrompt":{}` alone; no other golden changed.
6. **Met** (`b6adb5a`, P3 record items 4 and 5; `f441c73`, P2 record item 9). `ci.yml:191` and `:203` name
   `e62d77cddaf264cc6ec7bbc4f39b90322542f752 # v1.26.2`, `7c677842` is gone from the file, and actionlint exit 0.
   `README.md:156` and `docs/architecture.md:189` name acp-go-sdk2 `v1.36.0`; `README.md:163` and
   `docs/architecture.md:193` name go-selfupdate-lib `v1.26.2`. P2's added README line moved the cited lines by one.
7. **Met,** at P1, P2 and P3 (P1 record item 7, P2 record item 12, P3 record item 7).
   * macOS: `make preflight`, `GOFLAGS=-count=1 make race` (39 `ok`, none cached) and `make check-test-isolation`,
     each exit 0.
   * The Linux machine and the Windows laptop: `go test -count=1 ./...` and `CGO_ENABLED=1 go test -count=1 -race
     ./...` (38 `ok` each) and acpext's tests, each exit 0.
   * At P3, `make race`'s first run exited 2 at `TestWireOffline`, in a shell that still exported the proxy variables.
     The owner accepted the second run, in a shell with none, exit 0, as P3's race gate. Both runs are in the P3
     record.
8. **Met** (P4 record items 1 and 2). CI run `43119674374`, the push of `b6adb5a`, concluded `success`.
   `validate (ubuntu-24.04)`, `validate (macos-15)`, `validate (windows-2025)`, `test-isolation`, `sbom` and
   `build / build`, with its five `build / identity` jobs, are `success`; `release` is `skipped`. The owner pushed P1
   to P3 together, so one run reads all three (PLAN P1 step 8). Runs `43042877946` and `43080286906` had failed.
9. **Met** (`88655c5`; P1 record item 5). 0004-MADR's Toolchain line, 0023-MADR's Go row and 0042-MADR D1 each carry
   a 2026-11-11 note that quotes what it amends and links this record.

**The owner's decisions during execution:**

* **Q2 answered (a):** `override` is not advertised, and the SDK refuses it (Q2's note; PLAN P2 record item 2).
* **P3's race gate:** the second run accepted. The first is a misconfigured harness, not a test failure, so the
  PLAN's C5 stop and a record of its own did not apply (PLAN P3 record item 7's note).

## Pros and Cons of the Options

### A. One record, three steps

* Good, because the first commit alone turns CI green, and each later one reverts alone.
* Good, because 0051-PLAN P2 then moves one module, and its "a requirement that moves besides these two is a
  deviation" keeps its meaning.
* Good, because gobble and magic-cli-remote reach acp-go-sdk2 `v1.36.0` well before `v0.6.0`.
* Neutral, because it adds three commits and a CI run before the sign-in work starts.

### B. Go 1.27.4 only

* Good, because it is the smallest change that turns CI green.
* Bad, because the acp-go-sdk2 move would land inside 0051-PLAN, whose U8 evidence is line-cited at `v1.35.0`, and the
  system-prompt gate would arrive unnoticed among sign-in changes.
* Bad, because CI's workflow pins and the module would keep naming an older go-selfupdate-lib than magic-cli-remote's.

### C. Keep `go 1.27.3`, give CI Go 1.27.4

* Good, because a module that imports gobble's packages keeps the lower floor.
* Bad, because anyone building gobble with Go 1.27.3 builds the vulnerable `net/http` unwarned.
* Bad, because the release build uses `GOTOOLCHAIN: local` with the Go `go.mod` names (F2), so a `toolchain` line would
  have to be honoured by a workflow gobble does not own.
* Bad, because 0004-MADR's amendment of 2026-10-21 refused the same option for the same reasons.

### D. Fold it into 0051-PLAN P2

* Good, because there is one fewer record.
* Bad, because `main` stays red until go-llmprovider-sdk's next minor is tagged, which waits on that repository's own
  records and an owner-run probe.
* Bad, because a toolchain move, two pins and a new SDK minor in one phase cannot be reverted apart.

## More Information

### Observed, not decided: `TestRuleStoreConcurrent` on the Windows runner

* **What was seen.** Run `43080286906`, `validate (windows-2025)`, step `go test`:

  ```text
  --- FAIL: TestRuleStoreConcurrent (2.04s)
      store_test.go:186: permissions.json: locked by pid 5016
      (six such lines)
      store_test.go:192: 2 rules, <nil>; want 8
  FAIL    github.com/maccavelli/gobble-cli/permission    2.720s
  ```

  Every other package of that step passed.
* **What the code says.** The test starts eight goroutines, each adding one rule under the file's lock
  (`permission/store_test.go:179-194`). `RuleStore.Add` takes the lock with `filelock.EditPolicy`, whose deadline is
  2 s (`permission/store.go:168`; `internal/filelock/lock.go:53`). The test took 2.04 s, and six of the eight writers
  gave up with "locked by pid", the lock's refusal at its deadline. All eight are goroutines of the one test process.
* **What it is not.** No commit between `86ad4a7` and `17d1605` touches `permission/` or `internal/filelock/`
  (`git log 86ad4a7..17d1605 -- permission internal/filelock` is empty), so 0053's work did not change this code. The
  run for `86ad4a7` passed `go test` on Windows and failed later, at preflight. Of the 29 earlier runs `gh run list`
  returned for the workflow, back to 2026-10-14, none failed at `go test` on Windows: 27 concluded `success`,
  `42248276478` failed by a cancelled `identity` job alone, and `43042877946` is the run for `86ad4a7`. One failure in
  one run is all the evidence there is.
* **What is not known.** Whether it is the runner's speed, a Windows cost of the lock's retry, or a fault in the lock.
  Nothing here was run to tell.
* **Not decided.** This record changes neither the test, the lock nor its deadline. If the failure recurs on any host
  or runner while the PLAN runs, the PLAN stops and the failure gets a record of its own (PLAN, C5). It is not skipped,
  and no run is repeated until it passes.

### Not decided here

* **An earlier patch release for the Go fix.** `v0.5.0`'s binaries were built by CI with Go 1.27.3 (F2). Whether a
  `v0.5.1` should carry Go 1.27.4 ahead of `v0.6.0` is the owner's to ask; nothing here plans it.
  *(2026-11-11, later: decided by the owner, no `v0.5.1`; D6's note gives the decision and its consequence.)*
* **Applying a client's system prompt** (Q1 (b)): a record of its own.
  *(2026-11-11, later: decided by the owner, in this record: D7. Whether `override` is offered stays open, as Q2.)*
* **go-selfupdate-lib's sentinels** `ErrSpecialModeBits` and `ErrNoAllowedRoot` for more specific `gobble update`
  hints: still an addition to 0048, as 0050-MADR left it.

### Not verified, with the step that settles it

* That a local govulncheck under Go 1.27.4 is clean and under Go 1.27.3 reports GO-2026-6859 (F4): PLAN P1 step 2.
* That `go get` of each tag moves its own line alone, and that gobble's tests pass against each: PLAN P2 step 3 and P3
  step 1 (`check_libs.py`), then the gates.
* That the module proxy serves both tags: PLAN P2 step 1 and P3 step 1. Both tags exist on GitHub (2026-11-11).

### Revisions of 2026-11-11

The owner's rulings of 2026-11-11, given after this record was accepted and relayed by the cross-repository
coordinator. Each is a dated note beside the text it changes, and the record stays `accepted`; Q2 was open when this
paragraph was written. *(Later the same day: the owner answered Q2 (a), `append` only; see Q2's note and the
Completion section.)*

* **Q1 answered:** gobble advertises client system prompts and honours them (D5's note). **D7 added,** with **Q2**,
  `override`, then awaiting the owner, since answered (a). Decision Outcome, Consequences, Confirmation 5 and "Not decided here" carry notes.
* **No `v0.5.1`** (D6's note, "Not decided here", Consequences): the fix reaches users in `v0.6.0`, and installed
  `v0.5.0` binaries stay exposed until then.
* **D2 confirmed** as written.
* **`consulted`:** what D7 was read from.

### Related

* [0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md](0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md): the implementation.
* [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): the amendments of 2026-10-08 and
  2026-10-21, the two earlier Go moves and their reasoning.
* [0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md): the Go row.
* [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md):
  D2, `check-libs` before a pin.
* [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md): D1, acpext's `go`
  line; D5, its tags.
* [0048-MADR-gobble-update-command.md](0048-MADR-gobble-update-command.md): Q6 (a), the workflow pins.
* [0050-MADR-refresh-sibling-pins-to-current-releases.md](0050-MADR-refresh-sibling-pins-to-current-releases.md): the
  pins this record moves on from.
* [0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md):
  U8, re-read at `v1.36.0`.
* magic-cli-remote `docs/decisions/0179-MADR-gobble-native-acp-provider.md`: gate 3.
* acp-go-sdk2 `docs/reports/0054-REPORT-release-notes-v1-36-0.md`; go-selfupdate-lib `CHANGELOG.md`.
