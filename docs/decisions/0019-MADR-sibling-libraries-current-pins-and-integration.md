---
status: accepted
date: 2026-10-13
decision-makers: repository owner
consulted: none
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Pin acp-go-sdk2 v0.17.0, go-llmprovider-sdk v1.10.2 and go-selfupdate-lib v1.15.2, and integrate each where the library now does the work

## Context and Problem Statement

gobble-cli imports three of the owner's libraries:

- **acp-go-sdk2:** the ACP server and client (`acpserver`, `acpclient`, `acpclient/acptest`);
- **go-llmprovider-sdk:** every model call (`llm/provider`);
- **go-selfupdate-lib:** build identity (`internal/buildinfo`), the release specification test (`internal/cli/release_test.go`), and the reusable release workflows (`.github/workflows/ci.yml`).

On 2026-10-12 and 13 the owner tagged all three again. gobble pins `acp-go-sdk2 v0.14.0`, `go-llmprovider-sdk v1.2.1` and `go-selfupdate-lib v1.9.0` (`go.mod:15-17`). The latest tags are `v0.17.0`, `v1.10.2` and `v1.15.2`. [0015-REPORT-sibling-library-pins-and-order.md](../reports/0015-REPORT-sibling-library-pins-and-order.md), written the day before, recommended taking the tags of its day; none of its pins has landed, and every one of its tags is now behind. [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md) proposes `v0.15.0` exactly.

The owner asked for the current versions to be pinned, their API surfaces and compatibility assessed, and the most idiomatic, standards-conforming and robust way to integrate them evaluated, with the findings in this record. It decides:

- the pin of each library, and how each lands;
- what in gobble changes because each library now enforces, carries or reports something it did not;
- which library features gobble takes now, and which stay with the records that own them;
- how the compatibility checks that found these facts keep running.

It decides nothing about the libraries themselves, which are the owner's separate repositories. Where gobble needs something a library lacks, this record names it as a request, as [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md) has gobble wait rather than copy.

### What was measured, not assumed

Measured read-only on 2026-10-12, against gobble at `108d9f0` (`main`, clean before and after; `git status --porcelain` empty). Every library was read from the module cache at the named tag, through `go mod download`. Every build and test ran on a scratch clone of `108d9f0`, never in the working tree, with `GOTOOLCHAIN=go1.27.2`, `CGO_ENABLED=0` and every credential-shaped variable removed from the environment. The host is Windows; linux and darwin were reached by `GOOS` for `vet`, and a POSIX path rule was simulated where noted. Logs and scripts are in the session scratchpad, under `r-acp/`, `r-llm/`, `r-su/`, `v-combined/` and `v-llm-defaults/`.

**The tags.** Peeled with `git ls-remote`, summed by `go mod download -json`:

| Module | Tag | Commit | `h1:` (module) | `h1:` (`go.mod`) |
| :--- | :--- | :--- | :--- | :--- |
| acp-go-sdk2 | `v0.17.0` | `e21928c4e2a7fa8e18163b97ffec459948f9e7c8` | `d+ZASkLAYayy4xPtvaLEQShRy6BYoX21RbZsFSjNus8=` | `KqDRvShTbZEOpKOQOyRUnbeFqlyH20LXi7k3UOsgv0I=` |
| go-llmprovider-sdk | `v1.10.2` | `e46528198cee98e7a532a0824662e3e08ae12937` | `llPpBo+EKnQE4ksae+ZBK9xFOQ6z5NpK18H4cosmyJY=` | `en9k+ZhlMT41q/+FtXQrLE/3qDaBrWFr8TeLyZw2+Jg=` |
| go-selfupdate-lib | `v1.15.2` | `436a7f5976a6815491c7384527fef90869574a1b` | `FuQ++o9ojyQUDWR7g7WvLKYBzyNplztF3KrziyynlFE=` | `yZNc7rTDTKgGS2DaqNKRS9ph09bsIsP6ymJBCiV1Kfo=` |

- **acp-go-sdk2** has four tags, `v0.14.0` to `v0.17.0`. `v0.15.0` to `v0.17.0` were published within about 12 hours (2026-10-12 14:00 UTC to 2026-10-13 01:43 UTC), 31, 19 and 14 commits apart. Each ships its release notes in the module (`docs/reports/0013-…-v0-15-0.md`, `0014-…-v0-16-0.md`, `0017-…-v0-17-0.md`).
- **go-llmprovider-sdk** has 18 tags; 13 between `v1.2.1` and `v1.10.2`, cut in 6.5 days, 5 of them on 2026-10-12. It publishes no GitHub releases; its notes are its README's "Status" section.
- **go-selfupdate-lib** has 25 tags; 12 between `v1.9.0` and `v1.15.2`, cut in 7 days. Its notes are its migration guide (`docs/guides/migrating-from-mcplib-selfupdate.md` §10–§15).

**The API, by apidiff.** gobble's own `golang.org/x/exp/cmd/apidiff`, built from the repository's tool pin, ran in module mode (`-m -w` to export each tag, then `-m old new`). apidiff exits 0 whether or not it reports a break, so its text is the verdict; each run was proved by running it in reverse and seeing the breaks it then reports.

| Library | Range | Incompatible | Compatible | Reverse run |
| :--- | :--- | ---: | ---: | :--- |
| go-llmprovider-sdk | `v1.2.1`→`v1.10.2`, every step | 0 | 79 | 80 incompatible |
| go-selfupdate-lib | `v1.9.0`→`v1.15.2`, every step, three systems | 0 | 30 | 30 incompatible |
| acp-go-sdk2 | `v0.14.0`→`v0.15.0` | 14 | 140 | many removals |
| acp-go-sdk2 | `v0.15.0`→`v0.16.0` | 2 | 77 | |
| acp-go-sdk2 | `v0.16.0`→`v0.17.0` | 1 | 10 | |

- **acp-go-sdk2's incompatible changes** are all in package `acp`:
  - `v0.15.0`: `ErrorCode` became an `int32`, its nine variant types became constants, and `ErrorCodeOther` is gone (11 of the 14); `ExtRequest`, `ExtResponse` and `ExtNotification` became `jsontext.Value`; `ElicitationFormMode` is no longer comparable.
  - `v0.16.0`: `NewSessionConfigOptionSelect` and `NewSessionConfigOptionBoolean` gained `id` and `name` parameters.
  - `v0.17.0`: `ClientSideConnection` is no longer comparable. Its release notes say "the exported API only grows"; apidiff disagrees on this one point.
- **gobble uses none of them.** It builds config options as literals (`acpserver/config.go:99`), and holds the connection by pointer. The `Agent`, `AgentLoader`, `AgentExperimental`, `Client` and `ClientExperimental` interfaces are identical at all four tags.
- **go-llmprovider-sdk and go-selfupdate-lib kept their v1 promise:** no incompatible change at any step.

**gobble on each tag.** On scratch clones: `go get`, `go mod tidy`, `go build ./...`, `go vet` (three systems), and `go test -count=1` over the packages that import each library.

- **go-llmprovider-sdk `v1.10.2`:** every step exits 0, as does `go test ./...` (31 packages) and `govulncheck` ("No vulnerabilities found"). A negative control at `v1.0.0` fails to build (`llm/provider/provider.go:138: it.Signature undefined`).
- **go-selfupdate-lib `v1.15.2`:** every step exits 0, as do `scripts/verify_build_metadata.py` (7 of 7) and `govulncheck`. The shipped binary links only the library's `buildinfo` (`go list -deps ./cmd/gobble`). Negatives: a wrong `-X` symbol fails `verify-build-metadata`; a planted sixth platform fails `TestReleaseSpec` at `internal/cli/release_test.go:39`; at `v1.5.0`, `go build` passes but `go vet` fails, since `releasespec` is imported only by a test.
- **acp-go-sdk2:** `go get`, `tidy`, `build` and `vet` exit 0 at every tag. The tests do not:

| Tag | Failing tests on Windows | With a POSIX path rule simulated |
| :--- | ---: | ---: |
| `v0.14.0` | 0 | — |
| `v0.15.0` | 1 | — |
| `v0.16.0` | 25 | 2 |
| `v0.17.0` | 27 | 5 |

- **All three together** (`v-combined`): `go get` of the three tags, `tidy`, `build`, and `vet` for linux, darwin and windows exit 0. `go.mod` changes in three lines and `go.sum` in six; no other module moves. `go test ./...` fails exactly the 27 tests acp-go-sdk2 `v0.17.0` fails alone, and nothing else, so the three bumps do not interact.
- **Natively on Linux and macOS** (`v-native/`, `q450_native_posix.py`): gobble's HEAD and the combined bumped tree were each sent as a tarball to a fresh directory on the second Linux machine (linux/amd64) and on the macOS laptop (darwin/arm64), tested there with Go 1.27.2 and `go test -count=1 ./...`, and removed.
  - HEAD passes on both: 31 packages ok.
  - The bumped tree fails on both with exactly the 5 tests the simulated POSIX rule left: `TestAdditionalDirectoriesSent`, `TestDeleteSessionNotFound`, `TestElicitationNotFound`, `TestStartRefusesOtherProtocols` and `TestUnofferedTransportsRefused`. None of the 22 path failures appears, so the simulation was faithful.
- **No new transitive module at any tag of any library.** acp-go-sdk2 has no requirements at any tag (its `go.mod` sum is the same at all four). go-llmprovider-sdk requires `x/term v0.43.0` and `x/sys v0.47.0`, and go-selfupdate-lib also `x/mod v0.40.0`; gobble already selects newer versions of each.

**The advisory.** GitHub security advisory GHSA-579q-9j9v-7w8w (medium, published 2026-10-12 20:40 UTC) affects acp-go-sdk2 `= 0.15.0` exactly and is patched in `0.16.0`: `v0.15.0` accepted an empty `cwd` and path (0016-MADR's F3). `govulncheck` reports "No vulnerabilities found" on a clone pinned to `v0.15.0`: the advisory is not in the Go vulnerability database, so gobble's `govulncheck` gate cannot catch that pin.

**acp-go-sdk2's behaviour, by a probe run at each tag** (`r-acp/probe/probe_test.go`; its results differ by tag, so it detects each change):

| Probe | `v0.14.0` | `v0.15.0` | `v0.16.0` | `v0.17.0` |
| :--- | :--- | :--- | :--- | :--- |
| `session/new` with `cwd:""` | -32602 "cwd is required" | accepted | -32602 "cwd: \"\" is not an absolute path" | same |
| `session/new` with a relative `cwd` | accepted | accepted | -32602 | same |
| `"/posix/abs"` received on Windows | accepted | accepted | -32602 | same |
| a `session/list` answer with one relative `cwd` | 2 sessions | 2 sessions | -32603, the whole answer | same |
| a stdio MCP `command:"npx"` | accepted | accepted | -32602 "command: \"npx\" is not an absolute path" | same |
| a typed request before `initialize` | sent | sent | sent | `ErrNotInitialized` |
| a `notice` update without `session.notices` | sent | sent | sent | `*CapabilityError` |
| an image prompt block, agent not advertising images | sent | sent | sent | `*CapabilityError` |

- **The path rule.** From `v0.16.0`, 17 path properties must be absolute, checked when sent with a portable rule and when received with the host's `filepath.IsAbs` (SDK `connection.go:81`, `:1528` at `v0.17.0`). So a POSIX path such as `/work` is refused when received on Windows.
- **The remedy the SDK names for bare commands** is `WithInboundRewrite`, which sees each decoded message before it is validated, resolving the command first, "for example with `exec.LookPath`" (SDK release notes for `v0.16.0`, `:53-54`).
- **`v0.17.0`'s gates.** Methods and values are gated on what each side advertised: delete, logout, the file system, terminals, `additionalDirectories`, MCP transports, image, audio and resource blocks, notice and compaction updates, boolean options and terminal auth. A gated send is a `*CapabilityError`; a gated method received is -32601, a gated value -32602, and a handler result carrying one -32603. The protocol version is negotiated, with `ProtocolVersionError`. A cancelled turn is answered `stopReason: cancelled`, and pending permission requests are answered cancelled. A malformed JSON-RPC line is answered -32700 or -32600 instead of being dropped.
- **Not gated:** `Connection()`, `SendRequest`, `SendNotification` and `CallExtension`. gobble's `_gobble/*` calls go through `CallExtension` (`acpclient/sessions.go:102`).
- **Schema and protocol are unchanged:** ACP `1.25.0` and protocol version 1 at every tag; `diff -r schema/` between `v0.14.0` and `v0.17.0` is empty.

**go-llmprovider-sdk's behaviour.** Measured by running `providers.Default().Descriptors()` at each tag, as `llm/provider/ambient.go:27-37` reads it (`StaticModels[0]`, the default model):

| Provider | `v1.2.1` | `v1.7.1` | `v1.10.2` |
| :--- | :--- | :--- | :--- |
| gemini | `gemini-3.7-flash` | `gemini-3.5-flash-lite` | `gemini-3.5-flash-lite` |
| openai | `gpt-4.1-mini` | `gpt-6-luna` | `gpt-6-luna` |
| claude | `claude-haiku-4-5` | `claude-haiku-5-5` | `claude-haiku-5-5` |
| grok | `grok-4.6` | `grok-4.6` | `grok-4.6` |
| opencode-zen | `deepseek-v4.1-flash` | same | `claude-haiku-5-5` |
| opencode-go | `mimo-v2.6-flash` | same | `claude-haiku-5-5` |
| huggingface | `deepseek-ai/DeepSeek-V4-Flash-0731` | same | same |
| kilo | `deepseek/deepseek-v4.1-flash` | same | same |
| together | `deepseek-ai/DeepSeek-V4.1-Flash` | same | `zai-org/GLM-5.3-Flash` |
| ollama | none | none | none |

- **The registry order** is unchanged since `v1.2.1`: gemini first, so `Choose` still prefers it when its key is set.
- **Removed ids** include OpenAI's `gpt-4.1-nano` and `o4-mini`, which the SDK records as shut down on 2026-10-23 (its 0033-MADR), and `claude-sonnet-4-6`. `gpt-4.1-mini`, gobble's one hard-coded id (the strict live test), is still listed.
- **Streaming capability.** Since `v1.6.0` the inner providers report `NativeStreaming: Supported` for 8 ids. `WithRetry` still buffers the stream and reports `Unsupported` (SDK `retry.go:60-66`, `:95-98`, unchanged since `v1.7.0`). gobble's `New` copies the inner provider's capabilities (`llm/provider/provider.go:65`, `:82`). So on `v1.10.2` gobble would report live streaming while text arrives whole. Nothing outside tests reads the flag today.
- **Strict schemas.** `Tool.Strict` (from `v1.7.0`) is sent on the Responses and Chat-gateway wires, and from `v1.8.0` also on Claude's Messages wire (SDK `claude.go:20-21`); not on Gemini or Ollama. Only a `nil` `Strict` sends the bytes of before.
- **Usage.** From `v1.10.0` the SDK reports cache-write tokens, and from `v1.10.1` Claude's reasoning tokens. gobble's `llm.Usage` has `CacheWriteTokens`, but the adapter never sets it (`llm/provider/provider.go:167-170`), and `llm/doc.go:50` still says cache writes are unknown.
- **What "input" means, on each side.**
  - The SDK's `InputTokens` is every input token, its cache reads and writes included, so `InputTokens - CachedTokens - CacheWriteTokens` is the uncached input (`contract.go:141-163` at `v1.10.2`). At gobble's current pin, `v1.2.1`, it already included the cache reads (`contract.go:145`).
  - Pi's `input` excludes both. Read from Pi at `312184e` (the owner's fork of Pi, `main`): on Chat Completions `input = max(0, prompt_tokens - cacheRead - cacheWrite)` and `totalTokens = input + output + cacheRead + cacheWrite` (`packages/ai/src/api/openai-completions.ts:1522-1546`); on Responses, "OpenAI includes cached and cache-write tokens in input_tokens, so subtract both" (`openai-responses-shared.ts:562-572`); on Anthropic's Messages, `input` is `input_tokens`, which that service reports without its cache tokens (`anthropic-messages.ts:618-621`).
  - gobble's session history writes the SDK's `InputTokens` as Pi's `input`, and `InputTokens + OutputTokens` as its `totalTokens` (`acpserver/history.go:115-117`). So on the current pin, any turn with cache reads is stored with an `input` larger than Pi's for the same turn; `totalTokens` is already Pi's.

**go-selfupdate-lib's `gobble update` surface at `v1.15.2`,** read from source (the subject of 0005-PLAN F10, not built yet):

- **Discovery:** GitHub REST, HTTPS only, with redirects kept on HTTPS and credentials not carried across origins; token order explicit, provider, `GH_TOKEN`, `GITHUB_TOKEN`; a 404 is `ErrNoRelease` (from `v1.11.0`; `github.go:327-330`). gobble has no release yet.
- **Assets:** `NewExactAssetSelector`, `<product>-<goos>-<goarch>[.exe]`; a missing asset wraps `ErrUnsupportedPlatform` (from `v1.10.1`; `assets.go:107-111`).
- **Integrity:** HTTPS metadata, the asset's GitHub digest, and the `SHA256SUMS` entry. No signature; an opt-in provenance check, `verify/ghattest`, needs a logged-in `gh` (from `v1.12.0`).
- **Replacing a running executable on Windows:** a hard-link backup, then `MoveFileEx(REPLACE_EXISTING|WRITE_THROUGH)`, retried every 10 ms on a sharing violation or access denied until a lock timeout (default 5 s). A backup the old image still holds goes on a cleanup receipt, which the next start's `CleanupPending` removes (`replace_windows.go:34-171`; `standalone.go:66-75`). No helper process, except for updates started inside a service. Unix renames.
- **Guards and recovery:** symlinked targets and targets outside the home directory or `AllowedRoots` are refused (`target.go:17-45`, `:102-119`); a `PostInstall` probe rolls back on failure; an update journal, and from `v1.12.0` an update refuses to start while an unresolved journal exists; `KeptBackups`.
- **Check only:** `Checker.Check`, or `Request.CheckOnly` with `ErrUpdateAvailable`; `ExitCode` maps to 0, 10 and 1, which is gobble's `ExitUpdateAvailable` (`internal/cli/exit.go:16`).
- **Not in the library:** an offline switch, and a proxy option (it uses the caller's `http.Client`).
- **The release workflows' interface** (inputs, outputs, secrets) is identical `v1.9.0`→`v1.15.2`. gobble calls both at `39b12945fad311174252038745f5a88f71c4c66c # v1.9.0` (`.github/workflows/ci.yml:145`, `:157`).

### Findings

- **F1. Every library's latest tag is usable by gobble, and no tag adds a module.** The three bumps together change three `go.mod` lines and six `go.sum` lines, build and vet on three systems, and break only acp-go-sdk2's tests. go-llmprovider-sdk and go-selfupdate-lib need no Go change.
- **F2. acp-go-sdk2 `v0.15.0` is the one version gobble must not pin,** and gobble's security gate cannot see why: its advisory is not in the Go vulnerability database. 0016-MADR's D1 pins exactly that version.
- **F3. acp-go-sdk2 `v0.16.0` checks received paths with the host's rule, and gobble's tests and golden use POSIX paths.** With a POSIX rule simulated, 23 of `v0.16.0`'s 25 Windows failures pass, and 22 of `v0.17.0`'s 27: each comes from a fixture with `cwd` `/work`, `/w` or `/work/project`, and `acpserver/testdata/session.golden` records `"cwd":"/work"`. gobble's CI runs `windows-2025`, so the bump fails CI until the fixtures are portable.
- **F4. Two production behaviours change with `v0.16.0`.**
  - One stored session whose header `cwd` is not absolute makes `session/list` answer -32603 for every session: gobble's `ListSessions` maps each header into a `SessionInfo` (`acpserver/agent.go:815-837`), and neither `session` nor `session/jsonl` enforces absoluteness. Whether such files exist on users' machines is **[unverified]**.
  - A client's stdio MCP server with a bare command, such as `npx`, is refused -32602 before gobble's MCP code runs. gobble's own CLI sends no MCP servers (`acpclient/sessions.go:83`); which editors send bare commands is **[unverified]**.
- **F5. `v0.17.0` enforces rules gobble also checks, so gobble's checks become unreachable from a conforming peer:** the protocol version check (`acpclient/conn.go:125-127`), the SSE and ACP transport refusals (`acpserver/mcp.go:111-114`), and the client's -32601 stubs. The empty-`cwd` fallback on load and resume (`acpserver/agent.go:736-738`) is reachable only on `v0.15.0`.
- **F6. `v0.17.0`'s gates make a capability mistake fail a turn.** `updates.send` keeps the first `SessionUpdate` error (`acpserver/prompt.go:57-62`), so an ungated notice or compaction update, 0016's D8 and D9, ends the turn. The same gates make F4-3's terminal auth fail `initialize` unless it is advertised. gobble's agent passes its own gates today: it advertises exactly what it implements (`acpserver/agent.go:209-226`).
- **F7. The 5 `v0.17.0` failures that are not about paths are tests asserting gobble's own refusal where the SDK now refuses first:** `TestElicitationNotFound` (an unscoped elicitation, then an unadvertised one), `TestUnofferedTransportsRefused` (a relative stdio command, then unadvertised transports), `TestDeleteSessionNotFound`, `TestStartRefusesOtherProtocols`, and `TestAdditionalDirectoriesSent`, whose `acptest.ScriptAgent` advertises only close and images (`acpclient/acptest/script.go:118-129`). `TestLineSessionFirstPrompt` also panics on a failure, indexing `p[0]` after a `waitUntil` that only reports (`internal/cli/line_test.go:163-168`): a latent test defect.
- **F8. go-llmprovider-sdk `v1.10.2` changes what gobble offers and reports, with no test failing.** Six of the nine providers with a default change it (the table); two listed ids stop answering on 2026-10-23 on gobble's current pin; gobble's streaming flag would turn true while streaming stays buffered; strict mode would reach Claude if 0012-PLAN P7 sets `true` as written.
- **F9. go-llmprovider-sdk now has what several planned entries were to build or wait for:** the strict flag and the tool-result error bit (`v1.7.0`), native streaming on every wire (`v1.6.0`), forced-choice warnings (`v1.8.0`), `catalog.ModelInfo` with limits, costs and a caller layer (`v1.9.0`), cache-write and Claude reasoning tokens (`v1.10.0`, `v1.10.1`), and `TokenStore`, `RefreshLocker`, browser and device logins and `FedRAMP` for F4-3. It still lacks a streaming `WithRetry` (F4-1b's blocker), images, `cache_control`, deferral, and a generic Chat Completions provider. gobble's request for a streaming `WithRetry` has no SDK record, and the SDK repository has no issues.
- **F10. go-selfupdate-lib `v1.15.2` changes nothing at run time,** since the shipped binary links only its `buildinfo`. Its pin is in three places that must move together: `go.mod:17` and the two workflow `uses:` lines. `docs/architecture.md:111` and `README.md:87-88` also name `v1.9.0`.
- **F11. Several records state facts the new tags contradict:**
  - 0015-REPORT: `catalog.ModelInfo` shipped in `v1.9.0`, not `v1.8.0` (F2, F6); three of its four default-model changes at `v1.7.1` are wrong (`:163-166`: it has `gpt-5.6-luna`, `gemini-3.6-flash` and Grok `grok-3-mini-fast`→`grok-4.7`, where the SDK has `gpt-6-luna`, `gemini-3.5-flash-lite` and `grok-4.6` unchanged); its tags and "acp-go-sdk2 has one tag" are behind.
  - 0016-MADR: D1 pins the advisory's version; D2's `cwd` check is the SDK's from `v0.16.0`, and its fallback unreachable; its statement that responses are not validated is wrong from `v0.16.0`; D3's -32601 assertion cannot hold on `v0.17.0`; D7's accessors are `Negotiated()`'s job.
  - 0014-MADR: F7 ("its only tag"); its Confirmation's -32601 lines for the delete and elicitation tests.
  - 0005-MADR: `:966-967`, the SDK has no model limits or cost (superseded at `v1.9.0`); `:598`, a Windows "Access is denied" defect the library fixed in `v1.5.1`; `:268`, "six raw binaries" where there are five; `:587` and `:608`, and 0005-PLAN F10, still naming `go-core-lib v1.1.0`.
  - 0005-PLAN F4-1a pins `v1.7.0`, now 7 tags old; 0014-REPORT's gap table lists strict, the error bit, cache writes and native streaming as missing.
  - gobble's comments: `llm/doc.go:50` (cache writes unknown) and `llm/llm.go:21` (streaming).
- **F12. The checks that found these facts are cheap and have caught defects each time,** but live only in session scratchpads. apidiff's exit status is not a verdict; `go build` alone misses a test-only import; `govulncheck` misses a GitHub-only advisory; and the libraries tag several times a day.
- **F13. gobble's sessions already store a larger `input` than Pi's,** whenever a provider reports cache reads: the SDK's `InputTokens` includes them, Pi's `input` does not, and `acpserver/history.go:116` copies one into the other. The defect is in the current pin, not introduced by the bump; `v1.10.2`'s cache-write count widens it once D2 maps that count.
- **F14. gobble's ambient choice can never reach opencode-go** *(2026-10-13, found in 0019-PLAN P3; deviation 2)*. opencode-zen and opencode-go share `OPENCODE_API_KEY` (SDK `llmprovider/provider.go:69-70` at `v1.10.2`, `:63-64` at `v1.2.1`), and opencode-zen comes first in the registry (`providers/providers.go:34-35`), so `Choose` with that variable set always takes opencode-zen (`llm/provider/ambient.go:27-37`). Nothing else in gobble chooses a provider: `Ambient` and `Choose` are the only callers (`internal/cli/agent.go:30`, `:82`; `internal/cli/line.go:72`), and `--provider` is refused until 0005-PLAN F4 (`internal/cli/flags.go:33`, `:69`). So one of the ten providers is unreachable on the current pin and on `v1.10.2`, and D2's default-model test cannot reach it through `Choose` as written.
- **F15. acp-go-sdk2 `v0.17.0` races its own constructor, and can crash the agent** *(2026-10-13, found by 0019-PLAN P4's race gates; deviation 11)*. `NewAgentSideConnection` assigns `asc.conn = NewConnection(...)` only after `NewConnection` has started the read goroutine (SDK `agent_gen.go:28-34`), and the handler reads `a.conn` for `initialize` (`agent_gen.go:214`, `a.conn.checkInbound`). An `initialize` that arrives at once is handled before the assignment: 8 data-race reports, all on that pair, across `acpclient` and `internal/cli`, with `-race` on Windows and on the second Linux machine, where one run panicked with `checkInbound(0x0, ...)`, a nil-pointer dereference (`connection.go:1521`) that ends the process. `v0.14.0`'s handler reads no `a.conn`; the read came with `checkInbound` in `v0.16.0`. `NewClientSideConnection` has the same shape (`client_gen.go:24-27`), unreached, since no request reaches a client before its `initialize`. `v0.17.0` is the latest tag. *(Fixed in `v0.17.1`, tagged 2026-10-13 at `c77f25a45e33`: both side constructors build the connection, assign it, then start reading (SDK `buildConnection`, `start`), with `TestConstructorAssignsBeforeReading` under `-race`; apidiff `v0.17.0`→`v0.17.1` reports 0 incompatible changes.)*

## Decision Drivers

- **Do not pin a version with a known defect,** whatever the gate says (F2).
- **Pin what is released,** by its tag, never a pseudo-version, so each pin has notes and a sum (0004-MADR).
- **Let the library own what it now enforces.** A rule checked twice drifts; a rule gobble checks after the SDK has refused is dead code (F5).
- **CI on every system stays green at every commit,** including `windows-2025` (F3).
- **No behaviour changes silently.** Where a bump changes what users see (default models, streaming, strict mode), the change is stated and tested (F8).
- **One record decides each thing.** A decision this record makes is not left also open in another (F11).
- **Keep each change reviewable:** one library per commit, so a regression bisects to one library.

## Considered Options

- **A. Take each library's latest tag, one commit each, with acp-go-sdk2's tests made host-portable and its two production effects handled, and integrate each library's new work where a planned entry already wants it** (chosen).
- **B. Take go-llmprovider-sdk and go-selfupdate-lib now, and hold acp-go-sdk2 at `v0.14.0`** until its API settles (it broke something in each of three minors in 12 hours).
- **C. Take acp-go-sdk2 `v0.16.0`,** the smallest step past the advisory, and leave `v0.17.0`'s gates and negotiation for later.
- **D. Keep the current pins.**

## Decision Outcome

Chosen option: **A**. It is the only option that leaves gobble on releases with no known defect, on the protocol enforcement the SDK now ships, and with no rule checked in two places.

### The decisions

- **D1. Require `github.com/maccavelli/go-selfupdate-lib v1.15.2`,** with the sums above, in its own commit, with no Go change:
  - `go.mod:17`, and both reusable-workflow `uses:` lines at `436a7f5976a6815491c7384527fef90869574a1b # v1.15.2` (`.github/workflows/ci.yml:145`, `:157`), in the same commit, since the workflows and the module are one library;
  - `docs/architecture.md:111` and `README.md:87-88` name `v1.15.2`.
- **D2. Require `github.com/maccavelli/go-llmprovider-sdk v1.10.2`,** with the sums above, in its own commit. In that commit: *(2026-10-13: go-llmprovider-sdk `v1.12.0` is pinned by 0020-MADR D1 (0020-PLAN P1, `1625013`, deviation 3).)*
  - **the streaming flag is honest:** `llm/provider`'s `New` reports the capabilities of the provider it calls, the `WithRetry` wrapper, not the inner provider's, so `NativeStreaming` is `Unsupported` until F4-1b (F8);
  - **cache writes are reported:** the adapter maps `Usage.CacheWriteTokens` into `llm.Usage.CacheWriteTokens`, and `llm/doc.go:50` and `llm/llm.go:21` say what the SDK now reports;
  - **Pi's `input` is Pi's (F13):** `llm.Usage` keeps the SDK's meaning, every input token, and the session history converts at Pi's boundary, writing `input` as `InputTokens - CacheReadTokens - CacheWriteTokens`, floored at 0, and `totalTokens` as `InputTokens + OutputTokens`, as Pi writes them. A test records a turn with cache reads and writes and checks the stored `input`, `cacheRead`, `cacheWrite` and `totalTokens` against Pi's arithmetic;
  - **the default models are the SDK's:** gobble keeps taking `StaticModels[0]` (0002-PLAN Phase 3), and a test pins the ambient choice's model for each provider to the SDK's list, so a later bump that changes a default fails a test that names it; *(2026-10-13, P3 deviation 2: the ambient choice cannot reach opencode-go (F14), so the test chooses each provider through D15's `defaultProvider`, and checks the ambient choice's registry order separately;)*
  - **0005-PLAN F4-1a is annotated:** it is re-based on `v1.10.2`, which already carries its error flag, and its other steps keep their text.
  - The commit lands before 2026-10-23, when two ids gobble lists today stop answering.
- **D3. Require `github.com/maccavelli/acp-go-sdk2 v0.17.0`,** with the sums above, in its own commit, never `v0.15.0`. *(2026-10-13, 0019-PLAN P4 deviation 11: `v0.17.1`, once the owner tags it with F15's constructor race fixed; `v0.17.0` crashes the agent under a fast `initialize`. Its sums are recorded in the PLAN when it is pinned.)* The same commit carries D4 to D9; D3 never lands without them. Before that commit is staged, magic-cli-remote's ACP session runs against the bumped agent, since `v0.17.0`'s gates refuse what a peer did not advertise and no editor has run against it.
- **D4. ACP fixtures are absolute on the host they run on, and goldens are host-independent.**
  - Every test path that reaches the SDK (`cwd`, additional directories, file paths, stdio commands) is absolute on the running host: derived from `t.TempDir()`, or for a fixed fixture path, from a helper in `acpclient/acptest` that maps a POSIX fixture path to an absolute path on the host.
  - A golden records a placeholder, not the host's path: `acptest`'s recorder replaces the test's root directory with the token `/work` before it compares or writes, so `acpserver/testdata/session.golden` stays byte-identical, and the same golden holds on every system. *(2026-10-13, 0019-PLAN P4 deviation 4: the rewrite alone is not enough, since the golden's first `usage_update` estimates the system prompt, whose cwd section is `C:/work` on Windows; `TestSession` names `/work` in its prompt through a test-only `PromptOptions` field, as it already names `GOOS` linux.)*
  - No test is skipped on Windows and no assertion loosened to get there.
- **D5. A stored session with a non-absolute `cwd` does not break `session/list`.** `ListSessions` leaves out, and logs once per file, a stored session whose header `cwd` is not absolute, so the other sessions still list. Loading or resuming one is refused with -32602 and a reason naming the file's `cwd`.
  *(2026-10-12, at planning: the rule is the SDK's own, `(*acp.SessionInfo).Validate`, which checks with the host's `filepath.IsAbs` (SDK `types_gen.go:11640-11641`), so a session gobble lists is one it can open on this host; the SDK's outbound check of the whole answer uses a looser, portable rule, so nothing it would refuse is listed. The refusal on load and resume needs no code: under `v0.17.0` both requests always carry the client's absolute `cwd`, which gobble prefers to the header's (`acpserver/agent.go:736`), and D7 removes the header fallback, so a stored header's `cwd` never reaches a session.)*
- **D6. A client's bare stdio MCP command is resolved, not refused.** gobble's agent connection takes `WithInboundRewrite`, which resolves a stdio MCP server's `command` with `exec.LookPath` when it is not absolute, before the SDK validates it, as the SDK's release notes name. A command that does not resolve is left as it is, and the SDK refuses it with its own -32602. This keeps what gobble accepted before `v0.16.0`; `mcp.json`'s commands are unaffected, since they never pass through ACP.
- **D7. gobble stops checking what the SDK enforces at `v0.17.0`.**
  - The protocol version check (`acpclient/conn.go:125-127`) is removed; a `*acp.ProtocolVersionError` is reported in gobble's existing words.
  - The SSE and ACP transport refusals (`acpserver/mcp.go:111-114`) are removed, since the SDK refuses an unadvertised transport first.
  - The empty-`cwd` fallback on load and resume (`acpserver/agent.go:736-738`) is removed, since an empty `cwd` cannot reach it.
  - Each test that asserted gobble's refusal asserts the SDK's (F7): the error type (`*acp.CapabilityError`, `*acp.ValidationError`, `*acp.ProtocolVersionError`) or the code the peer receives, never weaker.
- **D8. Tests follow the SDK's gates, without weakening.**
  - `acptest.ScriptAgent` gains its advertised capabilities as a field, defaulting to today's, so `TestAdditionalDirectoriesSent` runs against an agent that advertises additional directories, and a new case shows the SDK refusing them toward one that does not.
  - `TestElicitationNotFound` and `TestDeleteSessionNotFound` assert the gate: the typed call fails with `*acp.CapabilityError`, and the agent's handler is never reached.
  - `TestLineSessionFirstPrompt` stops on a failed wait instead of indexing an empty slice.
- **D9. `acpclient` reports the SDK's reason.** Where a request is refused with -32602, the error gobble prints carries the SDK's `data.error` text, and a send the SDK refuses carries its `*acp.ValidationError` field, so a user sees which value was refused instead of "Invalid params" (F3's logs print only that).
- **D10. 0016-MADR's decisions are re-based, not duplicated.** On acceptance of this record, 0016-MADR gets a dated note:
  - its D1 is superseded by D3 here, and its D2 by the SDK's own check (F5), with D7 here removing the fallback;
  - its D3 is superseded by D8 here;
  - its D7's accessors are written over `Negotiated()`;
  - its D4 to D6 and D8 to D13 stay its own, `proposed`, and are re-checked against `v0.17.0` before its PLAN is written, since F6 makes D8 and D9's gating a condition of the turn not failing. *(P4 deviation 11: against `v0.17.1`.)*
- **D11. go-llmprovider-sdk's new features are taken by the entries that own them, re-based on `v1.10.2`, not by new work here:**
  - **0012-PLAN P7 (the strict flag)** is amended before it runs: `Strict` stays `nil`, the service default, for every provider except those whose live test has shown it accepts gobble's schemas with `true`, since from `v1.8.0` `true` also reaches Claude (F8); *(2026-10-13, the owner's choice at 0012-PLAN P7: P7 never sends `true`. It sends `false` for a tool with an argument a call may leave out, and no flag otherwise, so the condition on `true` does not arise; 0012-MADR D3, amended that day.)*
  - **0005-PLAN F4-2 (the model catalog)** is re-made on `v1.9.0`'s `catalog.ModelInfo`, `LookupModel` and `WithModelLayer`, with gobble's embedded snapshot passed as a layer, as the SDK's 0033-MADR designs it, in place of a gobble-side resolver;
  - **0005-PLAN F4-3 (credentials)** uses the SDK's `TokenStore`, `RefreshLocker` and logins, and advertises terminal auth before it offers one (F6);
  - **F4-1b (live streaming)** still waits for a streaming `WithRetry`. The owner files that request against go-llmprovider-sdk as a record there, since none exists (F9).
- **D12. `gobble update` stays 0005-PLAN F10's, against the newest go-selfupdate-lib at that phase.** This record's feature map is F10's input. F10's entry is corrected before it runs: no `go-core-lib`, the Windows "Access is denied" defect fixed in the library's `v1.5.1`, and five raw binaries.
- **D13. The records are corrected by dated notes, never rewritten** (F11): 0015-REPORT's three default models and its `v1.8.0` metadata; 0014-MADR's F7 and Confirmation lines; 0005-MADR `:268`, `:587`, `:598`, `:608` and `:966-967`; 0014-REPORT's gap table; and 0005-PLAN F4-1a and F10 as D2 and D12 say.
- **D14. The compatibility checks become a repository script, run before each pin and on demand, not in preflight.** `scripts/check_libs.py` (and `make check-libs`), for each of the three libraries:
  - takes the latest semver tag, never a branch;
  - on a scratch clone outside the working tree: `go get`, `go mod tidy`, `go build`, `go vet` for linux, darwin and windows, and `go test -count=1 ./...`;
  - reads apidiff's text for incompatible changes from the pinned tag, with the reverse run as its negative control;
  - runs `govulncheck`, and reads the library's GitHub security advisories, which `govulncheck` does not see (F2);
  - reports the `go.mod` and `go.sum` diff, and fails on a new module.

  It needs the network, so it stays out of `make preflight`, which must run offline.
- **D15. A `defaultProvider` setting names the provider** *(2026-10-13, 0019-PLAN P3 deviation 2, the owner's decision: "make opencode-go selectable", by the setting rather than the flags)*. 0018-MADR D19 reserved the key for 0005-PLAN F4; it comes forward alone, in D2's commit, added as D19 says a key is added: its Go type, schema, drift test and template block.
  - **The key:** `defaultProvider`, an enum of `""` and the nine providers that take a key, in the SDK's registry order. `""`, the default, keeps today's ambient choice. It is a process key (0018-MADR D16), read once at start, from any file; under `gobble acp`, as for every process key, from the global file and `--set` only.
  - **The choice:** `llm/provider`'s `Choose(env, preferred)` takes the setting. With `preferred` set, it is that provider with its first curated model when its credential variable is set; otherwise `ErrNoCredential` naming that variable, never another provider, since a silent fall-back would send the work to a provider the user did not choose. The no-credential message gobble prints names the same variable.
  - **The enum is the SDK's:** a test fails when the SDK's key-taking ids, in order, differ from the enum, so a bump that adds or reorders a provider fails a test that names it.
  - **What stays F4's:** Ollama is not in the enum, since it takes no key and has no curated model, so choosing it needs `defaultModel`; `defaultModel` and the model scope stay 0005-PLAN F4-2's, with `--provider` and `--model`, which win over the key for their run when F4-2 lands.

### Consequences

- **Good:**
  - gobble runs on releases with no known defect, and on the SDK's enforcement of protocol paths, capabilities and versions.
  - Rules are checked once, by the library that owns them; gobble loses dead code instead of keeping it in step.
  - Fixtures and goldens hold on every system, so Windows CI tests the same protocol as POSIX.
  - Users see current models, an honest streaming flag, and which value a refusal was about.
  - The checks that found this record's facts run on demand, from the repository.
  - *(2026-10-13, D15)* Every provider that takes a key can be chosen, opencode-go included, in every mode.
- **Bad:**
  - acp-go-sdk2's bump is a real change: about 27 tests, the golden recorder, and four production edits, in one commit.
  - acp-go-sdk2 is below v1 and broke something in each of its last three minors; the next minor will likely need the same work again, which D14 makes cheap to see but not free.
  - Six providers' default models change for users who set nothing.
  - D6 resolves a client's command against gobble's `PATH`, as gobble effectively did before `v0.16.0`; a client that relied on a bare name being refused would see it run.
- **Neutral:**
  - go-selfupdate-lib's bump changes nothing at run time.
  - No module is added.

### Confirmation

```text
go list -m github.com/maccavelli/acp-go-sdk2 github.com/maccavelli/go-llmprovider-sdk github.com/maccavelli/go-selfupdate-lib
                                                   -> v0.17.0, v1.10.2, v1.15.2   (P4 deviation 11: v0.17.1)
grep -c '436a7f5976a6815491c7384527fef90869574a1b # v1.15.2' .github/workflows/ci.yml -> 2
git diff <base> -- go.mod | grep -c '^[-+]\s'      -> 6: three requirement lines, no other module
go test ./...                                      -> ok on Windows, and natively on the second Linux machine and the macOS laptop; -race on Windows and Linux
go test ./acpserver -run Usage                     -> a turn with cache reads and writes stores Pi's input and totalTokens
magic-cli-remote's ACP session against D3's agent  -> a session opens, prompts and closes, with no capability or validation error
git diff --exit-code acpserver/testdata/session.golden -> no diff
go test ./llm/provider -run Default                -> each provider's ambient model is the SDK's StaticModels[0]
go test ./acpserver -run List                      -> a stored session with a relative cwd is left out, and the rest list
go test ./acpserver -run Bare                      -> a resolvable bare stdio command connects; an unresolvable one is -32602
python scripts/check_libs.py                       -> every library: latest tag, 0 incompatible, no new module, no advisory
go test ./llm/provider -run 'Default|Choose'       -> (2026-10-13, D15) each of the nine is chosen by defaultProvider with its model;
                                                      a set preference without its variable is ErrNoCredential naming it
go test ./internal/cli -run DefaultProvider        -> (2026-10-13, D15) the enum is the SDK's key-taking ids; the setting reaches
                                                      the agent's provider and model list; -p names the preferred variable
```

Every new test is seen failing on a scratch copy before it is trusted, as the repository's rule requires.

## Pros and Cons of the Options

### A — Latest tags, one commit each, acp-go-sdk2's tests made portable and its effects handled (chosen)

- Good: no pin with a known defect (F2); every library on what it now enforces and reports (F5, F9).
- Good: each commit is one library, so a regression bisects to it; the combined run shows the bumps do not interact.
- Good: dead checks go, and Windows CI tests what POSIX tests.
- Bad: acp-go-sdk2's commit is large, and its next minor will likely ask for the same.
- Bad: users' default models change.

### B — go-llmprovider-sdk and go-selfupdate-lib now, acp-go-sdk2 held at `v0.14.0`

- Good: the strongest argument for it: acp-go-sdk2 is below v1 and broke something in each of three minors within 12 hours, so waiting for it to settle could save doing this work twice.
- Good: the two v1 bumps are free and would land today.
- Bad: gobble keeps its own copies of rules the SDK now enforces, and keeps diverging from the SDK it is the reference client of.
- Bad: 0016-MADR stays pinned to the advisory's version, and its planned surfaces would be built against a version they must then be rebuilt on.
- Bad: the waiting has no end condition; the SDK has no v1 date.

### C — acp-go-sdk2 `v0.16.0`

- Good: the smallest step past the advisory: 25 failures instead of 27, and no gates yet.
- Bad: `v0.17.0`'s gates are what 0016's D8 and D9 must be built against (F6); building on `v0.16.0` and moving again repeats the work.
- Bad: two tags behind the day it lands.

### D — Keep the current pins

- Good: no work now.
- Bad: two ids gobble lists stop answering on 2026-10-23.
- Bad: every planned F4 entry builds against a version the SDK has moved 13 tags past.
- Bad: the gap to acp-go-sdk2 grows, and 0016-MADR stays proposed on a version with an advisory.

## More Information

### Evidence index

| Claim | Source |
| :--- | :--- |
| gobble's pins | `go.mod:15-17` at `108d9f0` |
| The tags, commits and sums | `git ls-remote` of each repository; `go mod download -json` (`q448-dl.json`) |
| acp-go-sdk2's tag times and notes | `r-acp/releases.json`, `dl-v*.json`; SDK `docs/reports/0013-`, `0014-`, `0017-REPORT-release-notes-*.md` |
| go-llmprovider-sdk's and go-selfupdate-lib's tags | `gh api …/tags` (`q445_lib_versions.py`); `r-llm/versions.txt`, `r-su` proxy times |
| apidiff counts and the reverse runs | `r-llm/api/summary.txt`; `r-su/apidiff-out*/`; `r-acp/diff-*.module.txt` |
| apidiff exits 0 on a break | the reverse runs' exit status in each research log |
| go-llmprovider-sdk `v1.10.2` passes, `v1.0.0` fails | `r-llm/compat-v1102.out`, `compat-neg.out` |
| go-selfupdate-lib `v1.15.2` passes, and its negatives fail | `r-su/compat-out-v1152/`, `neg-out/` |
| acp-go-sdk2's failing tests per tag | `r-acp/logs-v0.1[4-7].0/test.log`, `plogs2-v0.1[67].0/test.log`, counted by `q446_count_fails.py` |
| The three bumps together | `v-combined/logs/` (`q449_combined.py`): build, vet ×3 exit 0; 27 failures, the same set |
| No new module | the `go.mod`/`go.sum` diffs in each research log and in `v-combined/logs/diff-gomod.log` |
| GHSA-579q-9j9v-7w8w | `r-acp/advisories.json`: medium, `= 0.15.0`, patched `0.16.0`, published 2026-10-12T20:40:47Z |
| `govulncheck` misses it | `r-acp/logs-v0.15.0/vuln.log` |
| The host path rule | SDK `connection.go:81`, `:1528` at `v0.17.0` |
| The bare-command remedy | SDK `docs/reports/0014-REPORT-release-notes-v0-16-0.md:53-54` |
| acp-go-sdk2's probe results | `r-acp/probe/probe_test.go`, `probe-v*.log` |
| `v0.17.0`'s gates and negotiation | SDK `connection.go:1549-1562`; `agent_gen.go:567-587`; the `v0.17.0` notes |
| gobble's advertisement | `acpserver/agent.go:209-226` |
| `ScriptAgent`'s advertisement | `acpclient/acptest/script.go:118-129` |
| The latched update error | `acpserver/prompt.go:57-62` |
| The unreachable checks | `acpclient/conn.go:125-127`; `acpserver/mcp.go:111-114`; `acpserver/agent.go:736-738` |
| `ListSessions` | `acpserver/agent.go:815-837` |
| The latent panic | `internal/cli/line_test.go:163-168` |
| The default models | `v-llm-defaults/` (`q447_llm_defaults.py`), running `providers.Default().Descriptors()` |
| gobble's default-model choice | `llm/provider/ambient.go:27-37` |
| The streaming flag | `llm/provider/provider.go:65`, `:82`; SDK `retry.go:60-66`, `:95-98` |
| Strict on Messages from `v1.8.0` | SDK `claude.go:20-21` at `v1.10.2`, `:20` at `v1.7.0` |
| Cache writes unmapped | `llm/provider/provider.go:167-170`; `llm/doc.go:50`; `llm/llm.go` `Usage` |
| `InputTokens` includes cache tokens | SDK `contract.go:141-163` at `v1.10.2`, `:145` at `v1.2.1` |
| Pi's `input` excludes them | Pi at `312184e`: `packages/ai/src/api/openai-completions.ts:1522-1546`, `openai-responses-shared.ts:562-572`, `anthropic-messages.ts:618-621`; `types.ts:427-441` |
| gobble's history copies one into the other | `acpserver/history.go:115-117` |
| The native Linux and macOS runs | `v-native/*.log` (`q450_native_posix.py`): HEAD 31 ok on both; bumped, the same 5 failures on both; remote directories removed |
| The `gobble update` surface | `r-su` findings: `github.go`, `assets.go`, `replace_windows.go`, `standalone.go`, `target.go`, `errors.go` at `v1.15.2` |
| The workflow pins | `.github/workflows/ci.yml:145`, `:157` |
| 0015-REPORT's default models | `docs/reports/0015-REPORT-sibling-library-pins-and-order.md:163-166` |
| 0016-MADR's decisions | `docs/decisions/0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md:191-233` |
| The shared OpenCode variable (F14) | SDK `llmprovider/provider.go:69-70` at `v1.10.2`, `:63-64` at `v1.2.1`; registry order `providers/providers.go:34-35` |
| gobble's only provider choosers (F14) | `internal/cli/agent.go:30`, `:82`; `internal/cli/line.go:72`; `--provider` refused at `internal/cli/flags.go:33`, `:69` |
| `defaultProvider` reserved for F4 (D15) | `docs/decisions/0018-MADR-configuration-surface.md:566` (D19) |
| What a new key touches (D15) | a scratch clone of `e8181b5` with the key added and the schema regenerated (`q461_defaultprovider_probe.py`): `TestRenderDocument` and `TestKeysMatchSettings` fail, every other `internal/config`, `internal/cli` and `acpserver` test passes |

### Related records

- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): gobble waits for its libraries rather than copying them; rule 2 keeps ACP in `acpserver` and `acpclient`.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) and [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F4 (models and credentials) and F10 (`gobble update`), re-based by D11 and D12.
- [0012-MADR-tool-schemas-descriptions-and-loading.md](0012-MADR-tool-schemas-descriptions-and-loading.md) and its PLAN: P7, the strict flag, amended by D11.
- [0014-MADR-move-to-acp-go-sdk2.md](0014-MADR-move-to-acp-go-sdk2.md): the move to the SDK; its pin is superseded by D3.
- [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md): re-based by D10.
- [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md): D19 reserved `defaultProvider` for F4; D15 brings it forward (2026-10-13).
- [0014-REPORT-llmprovider-sdk-integration.md](../reports/0014-REPORT-llmprovider-sdk-integration.md) and [0015-REPORT-sibling-library-pins-and-order.md](../reports/0015-REPORT-sibling-library-pins-and-order.md): the earlier surveys, corrected by D13.

### Open questions for the plan

None. The three this record first left open were answered by the owner on 2026-10-12:

- **Pi's `input`:** included. Pi's source settles it (F13), and D2 converts at Pi's boundary.
- **Real editors on `v0.17.0`:** agreed. magic-cli-remote's ACP session runs against D3's commit before it is staged (D3, Confirmation).
- **Linux and macOS natively:** run. The bumped tree was tested natively on the second Linux machine and on the macOS laptop, and fails exactly where the POSIX simulation said ("What was measured"); the plan's gates run there too.
