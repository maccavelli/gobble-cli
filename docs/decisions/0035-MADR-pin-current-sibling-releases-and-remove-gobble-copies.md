---
status: accepted
date: 2026-10-29
decision-makers: repository owner
consulted: none
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Pin every sibling library's current tag, adopt what it now provides, and remove gobble's copies

## Context and Problem Statement

The owner asked on 2026-10-29: "re-assess the sources, dependencies, magic-cli-remote maturity, and pins. draft an
updated, detailed, comprehensive madr for review that includes what all we can pin, use, adopt, and clean out of the
codebase."

gobble depends on three of the owner's libraries and will depend on a fourth:

- acp-go-sdk2, go-llmprovider-sdk and go-selfupdate-lib are required in `go.mod`;
- go-tui-lib is named by [0032-MADR-go-tui-lib-integration-boundary.md](0032-MADR-go-tui-lib-integration-boundary.md)
  but not yet required.

A fifth repository, magic-cli-remote, is no dependency. It is the ACP client that is to run gobble as an agent, and its
records set a rule on gobble's acp-go-sdk2 pin.

Each library has tagged several releases since gobble last moved. Three open records each hold part of the answer, and
each is now out of date:

- [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md) (accepted) pins
  go-llmprovider-sdk `v1.22.2` and go-selfupdate-lib `v1.22.0`;
- [0032-MADR-go-tui-lib-integration-boundary.md](0032-MADR-go-tui-lib-integration-boundary.md) (accepted) was written
  against go-tui-lib `v0.12.0`, when nothing was adoptable;
- [0034-MADR-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md](0034-MADR-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md)
  (proposed, never executed) targets acp-go-sdk2 `v1.21.0`.

This record decides, in one place, which tag of each library gobble pins, what gobble then adopts from it, which of
gobble's own code goes, and what still waits. If it is accepted, it supersedes 0034 and amends 0031 D1 and 0032. It does
not implement anything. Its plan is [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md).

### How this was measured, 2026-10-29

Everything below was read on 2026-10-29 by read-only means:

- **gobble:** `main` at `ccd8aa8`, 9 commits ahead of `origin/main`, working tree clean, Go 1.27.3. `make preflight`
  and `go test -race ./...` pass at `ccd8aa8` in a scratch clone.
- **The five repositories:** scratch clones, fetched that day, read at their newest tags and their default branches,
  plus the GitHub API for releases, CI runs and issues.
- **The probes:**
  - `make check-libs`, run once per library;
  - `go doc` of every name named here, at its tag;
  - `go list -deps` of the go-tui-lib packages on linux, darwin and windows;
  - a scratch clone of gobble moved to all three newest SDK tags at once, then built, tested, staticchecked and linted.
    It was run twice: as the tags leave it, and with the one-line fix of F1.

Nothing was written to gobble's tree. The scratch clones were not committed anywhere.

### The pins

| Library | gobble pins | Newest tag (date) | Tags between | Tag's CI | GitHub release |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `github.com/maccavelli/acp-go-sdk2` | `v1.20.0` | `v1.26.0` (2026-10-29) | `v1.21.0` … `v1.25.0` | success | yes |
| `github.com/maccavelli/go-llmprovider-sdk` | `v1.22.2` | `v1.24.1` (2026-10-27) | `v1.23.0`, `v1.24.0` | success | yes |
| `github.com/maccavelli/go-selfupdate-lib` | `v1.22.0` | `v1.22.3` (2026-10-28) | `v1.22.1`, `v1.22.2` | success | no (a `CHANGELOG.md` entry) |
| `github.com/maccavelli/go-tui-lib/v2` | not required | `v2.6.1` (2026-10-29) | — | success | no |
| magic-cli-remote | not a dependency | `v0.23.0` (2026-10-24) | — | success | yes |

Since each library's default branch passed its newest tag, it has had only records, docs, CI or tests, apart from
go-selfupdate-lib. Its `main` has `fix(service)` and `feat(servicetest)` commits that are not tagged, in a package
gobble does not import. The ACP schema is `1.25.0` in every acp-go-sdk2 tag from `v1.20.0` to `v1.26.0`.

### What moving to every newest tag does to gobble today

The combined probe, with no source change:

| Gate | Result |
| :--- | :--- |
| `go get` of the three tags, `go mod tidy`, `go build ./...` | ok |
| `make archtest` | ok |
| `go test ./...` | **fails:** `llm/provider` `TestWireOffline` |
| `make staticcheck` (linux, darwin, windows) | **fails:** SA1019, 16 findings on 12 lines |
| `make lint` (golangci-lint) | **fails:** the same 12 lines, through its staticcheck |

`make check-libs` reported acp-go-sdk2 `v1.26.0` and go-selfupdate-lib `v1.22.3` as passing, and go-llmprovider-sdk
`v1.24.1` as failing its tests. It runs neither staticcheck nor golangci-lint, so it did not see F2.

- **F1. go-llmprovider-sdk `v1.23.0` hides gobble's offline wait.**
  - Since `v1.23.0` a request that cannot be sent for want of a network is `llmprovider.ErrOffline`, which also
    matches `ErrProviderUnavailable` (its CHANGELOG, "Changed").
  - gobble's `mapError` (`llm/provider/provider.go:616-640`) takes the first matching row of `sentinels`
    (`:509-518`), so the error becomes `llm.ErrUnavailable`. It tags `llm.ErrOffline` only when no row matched
    (`:632`, `case !kind && offline(err)`), so it never does.
  - The agent's wait for the network (`agent/agent.go:503`, `errors.Is(err, llm.ErrOffline)`) then never starts. A
    dropped connection spends the ordinary retries and fails the turn.
  - `TestWireOffline` catches it, for a refused dial and for an unresolvable host:
    `err llm: unavailable: llmprovider: offline: Post "http://127.0.0.1:…/v1/responses": dial tcp …: connection refused, want llm.ErrOffline`.
  - The probe added one row, `{llmprovider.ErrOffline, llm.ErrOffline}`, above the unavailable row. With it
    `go test ./...` passes, with every package ok.
- **F2. acp-go-sdk2 `v1.23.0` deprecates the unstable schema's names, and gobble uses some of them.**
  - From `v1.23.0` every declaration from the unstable ACP schema is marked `Deprecated: … stays under v1 for
    compatibility only`, by acp-go-sdk2's `0039-MADR-five-adopter-additions-on-schema-1-25-0.md`. Counted in its
    `*.go` files: none at `v1.22.0`, 274 at `v1.23.0`.
  - The SDK's new package `unstable` takes only new definitions. A field already merged onto a stable type has no
    counterpart there, and `PromptResponse.Usage` is one.
  - gobble's uses, the same on every GOOS:

    | Site | Name | Why gobble has it |
    | :--- | :--- | :--- |
    | `acpserver/prompt.go:295-296` | `acp.Usage` | builds the turn's token usage |
    | `acpserver/agent.go:746` | `PromptResponse.Usage` | sends it on the prompt's answer |
    | `acpclient/conn.go:339` | `PromptResponse.Usage` | reads it on the client |
    | `acpclient/acptest/script.go:189-190` | `PromptResponse.Usage`, `acp.Usage` | the scripted agent sends it |
    | `acpclient/acptest/script_test.go:49` | `PromptResponse.Usage` | tests that |
    | `acpserver/prompt_test.go:140-141` | `PromptResponse.Usage` | tests that |
    | `acpserver/agent_test.go:144`, `:148` | `SessionCapabilities.Fork`, `McpCapabilities.Acp` | asserts gobble does **not** advertise them |
    | `acpserver/mcp_test.go:200` | `McpServer.Acp`, `McpServerAcpInline` | asserts an MCP-over-ACP server is refused |

  - magic-cli-remote meets the same deprecation: 56 findings. It reads per-turn usage from prompt answers since
    `5acbd2d3`, so gobble's `PromptResponse.Usage` is data its client uses. It chose a narrow suppression with a gate
    (its 0207-MADR amendment H, D45, and `scripts/check_staticcheck_ignored.py`).
- **F3. Neither failure is a reason to stay.**
  - F1 is one table row.
  - F2 is a lint policy for names the SDK keeps under v1, with no exported API removed. `apidiff` in `check-libs`
    reports every move compatible.
- **F4. `check-libs` cannot see F2, and does not cover go-tui-lib.**
  - `scripts/check_libs.py` names three libraries (`:66-68`).
  - Per library it runs tidy, build, vet on three GOOS, test, apidiff and govulncheck. It runs no staticcheck or
    golangci-lint.

### acp-go-sdk2, `v1.20.0` to `v1.26.0`: what replaces gobble's code

| gobble today | The SDK's | Since |
| :--- | :--- | :--- |
| Five hand-built `SessionUpdate` literals (`acpserver/modes.go:85`, `commands.go:531`, `:597`, `:728`, `prompt.go:319`) | `UpdateCurrentMode`, `UpdateConfigOptions`, `UpdateSessionTitle`, `UpdateAvailableCommands`, `UpdateUsage` | `v1.21.0` |
| Nine client stubs answering `NewMethodNotFound` (`acpclient/conn.go:552-600`) | embedded `ClientRefusals` | `v1.21.0` |
| `translate`'s switch, with no name for an unknown variant (`acpclient/update.go:239`) | `SessionUpdate.Kind()` | `v1.21.0` |
| Two agent stubs, `DeleteSession` and `Logout` (`acpserver/agent.go:1176-1180`, `:1244-1246`) | embedded `AgentRefusals`. gobble's own `CloseSession`, `ListSessions` and `ResumeSession` take precedence over its methods. | `v1.22.0` |
| The reject-option loop (`acpclient/conn.go:504-510`) | `PickPermissionOption(opts, RejectOnce, RejectAlways)` | `v1.22.0` |
| `resolveBareCommands` (`acpserver/rewrite.go`, 36 lines) | `ResolveStdioCommands(exec.LookPath)`: the same rule (absolute is kept, a resolved command is replaced, anything else is left for the SDK's `-32602`). It also reads `session/fork`. | `v1.22.0` |
| `ended` treating any `ErrConnectionClosed` as clean (`acpserver/serve.go:62`), the close a notification overflow makes included. That is a defect: an overflow is reported as a clean end. | `EndedCleanly`, false for the overflow's close | `v1.25.0` |
| `handlersDone` (a goroutine and a timer), three copies: `acpserver/serve.go:71`, `acpclient/conn.go:207`, `acpclient/acptest/pair.go:65`. It is also called from `acptest/script.go:84-87`. | `Connection.WaitContext(ctx)` | `v1.25.0` |
| `acpserver.Serve`'s body: `NewAgentSideConnection`, the `Done`/`ctx` select, `endWait`, `ended` (`serve.go:20-66`) | `ServeAgent(ctx, agent, in, out, opts...)`: a 5 s handler wait, `EndedCleanly`, and on ctx's end a graceful `Shutdown` that answers a prompt in flight `cancelled`, which gobble does not do today. The agent's `Close` stays the caller's. | `v1.26.0` |
| `acptest.ScriptAgent`'s serve loop (`acpclient/acptest/script.go:80-92`) | `ServeAgent` | `v1.26.0` |
| `acpclient.Start` running gobble's agent over two `io.Pipe`s on a goroutine, with `Close`'s own timeout (`acpclient/conn.go:116-221`, `Start` and `Close`) | `agentproc.InProcess{Serve: …}.Start(client)`: the same `*Process` a child gets, three-step `Close`, and `ErrAbandoned` for a `Serve` that will not stop | `v1.26.0` |

Also available, and **not** a replacement for anything gobble has:

- `clienthost` (Terminals, Files, Refusals). gobble's client advertises no fs or terminal capability (0014-MADR D3).
- `ToolCallTimeline` and `CompactionTimeline`. gobble's client keeps no tool-call entity state: it renders each update
  as it comes.
- `PresentNotice` and `PresentCompaction`. Sending notices and stable compaction is 0016-MADR D8–D10, a product choice.
- `ExtensionMux`. gobble's `_gobble/*` handler is one switch.
- Elicitation checks. gobble offers no elicitation.
- `NewTranscript`, `Command.StderrTail`, and acptest's `ReadRecords`, `Golden`, `ReplayAgent` and `ConnectReplay`.
  gobble runs no child ACP agent outside `acpserver/binary_test.go`, which already uses `agentproc`.
- acptest's `ConnectRecorded` and `ValidateGolden`. gobble's own `acpclient/acptest` (`Connect`, `Recorder`,
  `ScriptAgent`) has 60 importing test files, and is what the SDK's `acptest` was drawn from.

### go-llmprovider-sdk, `v1.22.2` to `v1.24.1`: what replaces gobble's code

These are the slices of [gobble-cli#4](https://github.com/maccavelli/gobble-cli/issues/4) (go-llmprovider-sdk issue 9,
its `0061-MADR-consumer-gaps-from-gobble-issue-9.md`) that shipped.

| gobble today | The SDK's | Since |
| :--- | :--- | :--- |
| `offline()` and `unreachable()`, a `net` error walk with Winsock codes (`llm/provider/provider.go:544-574`) | `errors.Is(err, llmprovider.ErrOffline)`. gobble's `Probe` and the stored `offlineURL` stay, since the SDK's error still wraps the `*url.Error`. | `v1.23.0` |
| `sdkEmptyAnswer`, a match on the SDK's message suffix, in `emptyAnswer` and `refusedAnswer` (`:520-541`) | `errors.Is(err, llmprovider.ErrEmptyAnswer)`. `APIError.Reason` still separates an empty answer from a `content_filter` refusal. | `v1.23.0` |
| `"max"` sent as `EffortXHigh` (`reasoningFor`, `:496-499`) | `llmprovider.EffortMax`, stepped down by the SDK to the highest effort a model takes, with `WarningEffortDegraded` | `v1.23.0` |
| No workspace: `signin.go:173` keeps no `Result.Workspace`, and no provider is built with one | `wizard.Result.Workspace`, `claude.WithWorkspace`, `catalog.WithClaudeWorkspace`, `ErrWorkspaceRequired`/`ErrWorkspaceNotFound` | `v1.24.0` |

Behaviour that changes under gobble without its code changing:

- An offline error is now retried by the SDK's own `WithStreamRetry` before gobble's wait sees it (F1).
- A refused reasoning effort or thinking shape is learned and the request is sent again: `wire.Send`'s one resend
  becomes at most 7 sends.
- Effort follows the model's facts, not name rules.
- Ollama's `xhigh` is sent as `xhigh`.

Also available. Each changes what gobble offers, so none is a copy to drop:

- `ImageItem` and `Capabilities.Images`: image input, for 0005-MADR, 0008-MADR D19 and 0018-MADR.
- `chatcompat`, with `Discover`: custom OpenAI-compatible servers. `llm/catalog/modelsjson.go` refuses a provider the
  SDK does not build.
- `WithReasoningRefusals` and `ProbeReasoning`: keeping learned refusals across runs.
- `Catalog.Search`: a facts-aware search. It changes what `gobble models --search` and the ACP picker return.

### go-selfupdate-lib, `v1.22.0` to `v1.22.3`

- No exported identifier changes (`apidiff` in `check-libs`).
- gobble's non-test code imports only `buildinfo` and `selfupdate/releasespec`. Neither package changed between the
  tags, and neither did the two reusable release workflows `ci.yml:166` and `:178` pin.
- The non-test changes are in `selfupdate` and `selfupdate/service`, which gobble uses only in `internal/cli/release_test.go`
  (`selfupdate.Platform`).
- So the move is `go.mod`, `go.sum`, and the two workflow pins, which follow the library
  ([0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md) A2), from
  `84805563ed252df384f59ced45a3daf529702ab4 # v1.22.0` to `0620e96ba7eb47d45cc94b474dc2db87d029ce40 # v1.22.3`.
- The library's `0038-MADR` puts a pinned update plan (`Plan`, `Apply`) in a future `v1.23.0`. `gobble update`
  (0005-PLAN F10 step 4) is not built.

### go-tui-lib, `v0.12.0` to `v2.6.1`

The module is now `github.com/maccavelli/go-tui-lib/v2`. It moved from `v0.12.0` (2026-10-11) to `v1.13.0` and
`v2.0.0`, both on 2026-10-24, and has tagged seven more releases since. Every gobble request to it is delivered: no issue is open, and
issue 10, the line-mode transcript, closed on 2026-10-29 with `stream` and `transcript`. Its records name gobble as the
consumer.

The closure of each package gobble's CLI would import, from `go list -deps` on linux, darwin and windows at `v2.6.1`:

| Package | External modules in its closure | Charm apart from `x/ansi` |
| :--- | :--- | :--- |
| `safetext`, `glyph`, `termkey` | none | none |
| `ask`, `pick`, `lineedit`, `inline`, `stream`, `transcript`, `termcap`, `keymap`, `fuzzy` | `x/ansi` and its four text modules | none |
| `tuitest` | also `colorprofile`, `x/term`, `xo/terminfo` | **yes**; rule 5 admits it in tests only |

`x/ansi` `v0.11.8`, `clipperhouse/displaywidth` `v0.11.0`, `clipperhouse/uax29/v2` `v2.7.0`,
`lucasb-eyer/go-colorful` `v1.4.1` and `mattn/go-runewidth` `v0.0.27` are the modules and versions 0032-MADR
amendment A1.2 already names. So the first import needs no new dependency decision.

| gobble today | go-tui-lib | Since |
| :--- | :--- | :--- |
| The one-line permission prompt: `permissionLine`, `chooseKey`, `readWhy`, `maxWhy` (`internal/cli/permission.go`, 231 lines) | `ask`: rows of keyed answers, wrapped, one of which reads a reason. This is 0029-MADR's layout, which 0032 D4 made go-tui-lib's first component. | `v2.3.0` |
| `internal/cli/term/sanitize.go` (154 lines) | `safetext` (`Text`, `Line`, `Strict`; a streaming `Sanitizer`) | `v2.0.0` |
| `internal/cli/term/caps.go` (85 lines) | `termcap` | `v2.0.0` |
| `glyphsFor` (`internal/cli/view.go:98`) | `glyph` | `v2.0.0` |
| The numbered-menu `questions.choose` (`internal/cli/prompt.go:108`), and the model and session choices | `pick` (fzf's reverse layout, inline) | `v2.4.0` |
| `internal/cli/editor` (10 files, 1,764 lines) | `lineedit.Composer`: history, completion, bracketed paste, external editor, keymap | `v2.5.0` |
| `internal/cli/render` `mdstream.go`, `mdstyle.go`, `table.go`, `wrap.go` (1,229 lines), and `internal/cli/transcript.go` (125) | `stream.Printer` and `transcript.Transcript`, with gobble's 24 and 30 cases and its `style.golden` and `tables.golden` as the library's tests | `v2.6.0` |

These line counts are what each package replaces. Each adoption step measures what actually goes, since `render` and
`term` hold code no library has, such as `udiff.go`, `spinner.go`, `resize*.go` and `console_windows.go`.

How the library's output differs from gobble's, as issue 10's closing comment lists it:

- tables are laid out with colour off too;
- a link's text is underlined and its URL dimmed;
- a multi-line setext heading draws only its last line as the heading;
- `SGR 0` is written before a reprint;
- the notice is the catalogue's text, unstyled by default;
- rows are counted as terminal rows, not lines.

Options restore gobble's output where one exists: `stream.WithoutTableLayout()` and
`transcript.WithNoticeStyle(stream.Dim)`.

Performance, against 0030-MADR D5: the library's reprint benchmark is gobble's own. At `v2.6.1` a reprint of 4 MiB
takes 5.2 ms and 43,541 allocations, against gobble's 9.5 ms and 49,696 at `ccd8aa8`, both measured on macOS. A reply
render takes about 24 µs against gobble's 35 µs. `TestReprintBudget` and `TestReplyBudget` in the library fail above
gobble's numbers. Timings were not measured on Linux or Windows.

`internal/archtest` is not ready for `/v2`. `tuiLib` is `"github.com/maccavelli/go-tui-lib"` (`check.go:40`). The
closure rule matches `/v2` paths, since `pathUnder` is a prefix match. `isTUITest` checks `tuiLib+"/tuitest"`, which
`/v2/tuitest` does not match, so the test-only exception would refuse the real path. A stray v1 import would also pass.

### magic-cli-remote: maturity, and what it asks of gobble

- **Its state:**
  - It is pre-1.0: `v0.23.0`. It has 119 `v` tags since `v0.2.0` (2026-07-19), 6 of them in October.
  - It holds about 94,000 lines of non-test Go, 644 test files and 364 records.
  - CI passes on its `master`.
  - `master` is 49 commits past `v0.23.0`, untagged. They include `0edeffba` (2026-10-29), which pins acp-go-sdk2
    `v1.26.0` and go-selfupdate-lib `v1.22.3` with no source change: its 0207-MADR amendment I, D48 and D49.
- **What it is to gobble:**
  - It is an application with no importable package, so gobble can adopt nothing from it.
  - It does not run gobble yet. Its gobble provider, `0179-MADR-gobble-native-acp-provider.md`, is `proposed`, and
    `0208-MADR-make-the-provider-layer-agent-neutral.md` is in progress. No gobble `Spec` is in its code.
- **Its rule for gobble:**
  - Its `0184-MADR-adopt-acp-go-sdk2-and-offload-protocol-handling.md` D12 says: "Magic and gobble pin the same
    `acp-go-sdk2` tag and bump it together, each bump checked by both repositories' live ACP suites."
  - Its 0207 amendment H (question 15, owner's answer "Magic first") puts Magic first, with gobble following in its
    own repository. That is the source of [gobble-cli#6](https://github.com/maccavelli/gobble-cli/issues/6)'s "Magic
    moves first".
  - No gobble record adopts D12; gobble is listed there as informed.
  - Magic has now moved, to `v1.26.0` on `master`. So gobble pinning `v1.26.0` satisfies D12 rather than waiting on it.
- **Its open issues:** [magic-cli-remote#28](https://github.com/maccavelli/magic-cli-remote/issues/28) and
  [#29](https://github.com/maccavelli/magic-cli-remote/issues/29) are its own adoptions. Neither waits on gobble.

### What the libraries still do not have

- **A per-request header option.** go-llmprovider-sdk declined it in its 0061-MADR D6, so `headerTransport` stays.
- **A price for one-hour cache writes.** `catalog.ModelRates` has one `CacheWrite` rate, so
  `Usage.CacheWrite1hTokens` would change no figure gobble shows (0031-MADR D8, amended).
- **Tool-call argument deltas,** behind go-llmprovider-sdk's 0031-MADR Q4.
- **go-selfupdate-lib's `Plan`/`Apply`,** in its untagged `v1.23.0`.

These are not library gaps; they wait on gobble's own records:

- **0016-MADR D8–D10:** what gobble sends as notices and compaction.
- **Native tool deferral:** 0012-MADR and 0021-MADR.
- **The ChatGPT plan sign-in:** 0031-PLAN P4b. It needs no new tag: `LoginChatGPTPlan` is in every pin since `v1.15.0`.
- **The Windows job-object race in `internal/proctree`:** noted in 0031-MADR "More Information". `agentproc` is for
  ACP agents and does not apply to MCP servers or the shell tool.

### Records and docs that are out of date

| Where | What it says | Fact on 2026-10-29 |
| :--- | :--- | :--- |
| 0034 MADR and PLAN | pin `v1.21.0`; constructors, `ClientRefusals`, `Kind()` | `v1.26.0` is current; #6 retargeted to `v1.25.0`; this record supersedes it |
| 0031-MADR D1, 2026-10-24 note | go-tui-lib's newest tag is `v0.12.0`; none of its issues 2–12 is tagged | `v2.6.1`; every request delivered |
| 0032 MADR and PLAN | the v1 module path; "nothing adoptable" | `/v2`; the packages above are adoptable |
| 0029-PLAN P2 | stopped until go-tui-lib's inline question is tagged | `ask` is in `v2.3.0` |
| `docs/README.md` "I want to…" | "pin acp-go-sdk2 `v1.21.0`" | as for 0034 |
| `docs/README.md` "Build order" | its run list stops at 0022 | 0023 to 0034 are missing |
| `internal/cli/auth.go:192`, quoted in `docs/guides/credentials.md:143-147` | "openai's login waits for go-llmprovider-sdk's Sign in with ChatGPT" | the SDK has had it since `v1.15.0`; the wait is gobble's 0031-PLAN P4b |
| `llm/llm.go:45` | "ImageInput stays false until the SDK accepts image input" | the SDK does since `v1.23.0`; the wait is gobble's own |
| `llm/doc.go:47`, `docs/guides/choosing-a-model.md:42` | `max` is sent as `xhigh` | true until D3 lands |
| `README.md` "Stack", `docs/architecture.md` pin table | the current pins | accurate today, and they move with each pin |

## Decision Drivers

* **0004-MADR:** gobble waits for a library rather than copying it, and drops its copy once the library has the thing.
* **The owner's direction** (0031-MADR): adopt as much of the SDKs as possible, to keep gobble lean.
* **No regression:**
  - F1's offline wait is kept;
  - `acpserver/testdata/session.golden` stays byte-identical (0016-MADR D5);
  - the CLI's output stays the same unless a decision says otherwise;
  - every gate passes on macOS, the Linux host and the Windows host.
* **The default binary stays free of any TUI framework** (0008-MADR F5; 0032-MADR D1 and A1).
* **Each step small enough to land, verify and revert on its own,** with `go mod tidy` clean at every commit.
* **gobble's gates are gobble's.** A rule set in another repository binds gobble only through a gobble record.

## Considered Options

* A. Pin each library's newest tag, changing only what the gates force (F1, F2), and adopt nothing else.
* B. Pin every newest tag in one ordered sequence, adopt each replacement listed above as its own phase, and leave
  product-surface features to their own records.
* C. Leave 0031, 0032 and 0034 as they are, amending each separately as its library moves.
* D. Wait: for magic-cli-remote to tag its `v1.26.0` move, and for go-tui-lib's `/v2` to settle.

## Decision Outcome

Chosen option: **"B. Pin every newest tag in one ordered sequence, and adopt each replacement as its own phase"**,
because every library now ships what gobble waited for. The two failures the move meets are a table row and a lint
policy. And one record keeps the order, and the cross-library facts (check-libs, archtest, Magic's rule), in one place.

### The pins

- **D1. Pin exactly:**
  - acp-go-sdk2 `v1.26.0`;
  - go-llmprovider-sdk `v1.24.1`;
  - go-selfupdate-lib `v1.22.3`, with both `ci.yml` workflow pins at `0620e96ba7eb47d45cc94b474dc2db87d029ce40`;
  - `github.com/maccavelli/go-tui-lib/v2` `v2.6.1`, in the commit of its first import (`AGENTS.md`, "Dependencies").

  Each is checked first by `make check-libs` as extended by D2. Each pin is its own commit. A tag newer than these, at
  the time a pin step runs, is a deviation for the owner, not a silent substitute.

  *Amended 2026-10-29, by the PLAN's deviation 1, from the owner's request "i want to update all pins to current
  versions" and answer "The newest (Recommended)". `git ls-remote --tags` that day listed two newer tags:*
  - ***go-selfupdate-lib `v1.23.0`** (`67520be8af1ad1b6228c29e4b1aa3e61e1ca7ae4`, 2026-10-29), in place of `v1.22.3`:*
    - *`python3 scripts/check_libs.py --lib go-selfupdate-lib --tag v1.23.0` passed all 12 checks, `api`
      included;*
    - *between `v1.22.3` and `v1.23.0`, `buildinfo`, `selfupdate/releasespec`, the module's `go.mod` and the two
      reusable release workflows are unchanged. `selfupdate/selfupdatetest`, which gobble's tests import, gains
      `Publish` and a stricter `VerifyReleaseSource`;*
    - *the rest, `Plan` and `Apply`, the handoff's plan, `servicetest` and the service fixes, is in packages gobble
      does not import;*
    - *the two `ci.yml` pins become `67520be8af1ad1b6228c29e4b1aa3e61e1ca7ae4 # v1.23.0`.*
  - ***go-tui-lib/v2 `v2.6.2`** (2026-10-29), in place of `v2.6.1`. It fixes a parser panic in `termcap` and
    `inline`, by bounding a reply's parameter separators (`696f4af`). `inline` is under `ask` and `lineedit`.
    `go.mod` is unchanged, so A1.2's module list still holds.*

  *acp-go-sdk2 `v1.26.0` and go-llmprovider-sdk `v1.24.1` are still the newest.*
- **D2. `check-libs` sees what preflight sees.**
  - `scripts/check_libs.py` runs `make staticcheck` and `make lint` on its scratch module for each library, so that F2
    would have been reported.
  - It gains go-tui-lib `/v2`, with `./internal/cli/...` and `./internal/archtest/...` as its packages.
  - Its self-test gains a case that fails on a deprecated use. That case is seen failing on a scratch copy before it is
    trusted.

  *Amended 2026-10-29, while writing the PLAN:*
  - *The self-test is offline and runs no staticcheck. So the failure on a deprecated use is seen by `check-libs`' own
    live run instead: acp-go-sdk2 `v1.26.0` against gobble before D6 lands, which F2 shows fails staticcheck and lint.*
  - *The self-test gains the offline cases D2 adds: a `/v2` module path, the newest tag of that major only, a library
    not yet required, and the modules a first import may add.*

### go-llmprovider-sdk

- **D3. The pin carries F1's fix in the same commit:** `{llmprovider.ErrOffline, llm.ErrOffline}` in `sentinels`, above
  the unavailable row. `TestWireOffline` is the check, and it already fails without the row.
- **D4. Then gobble's copies go.**
  - `offline()` and `unreachable()` give way to `errors.Is(err, llmprovider.ErrOffline)`, for the row and for recording
    `offlineURL`. `Probe` stays.
  - `sdkEmptyAnswer` gives way to `ErrEmptyAnswer`, with `Reason` still choosing between `emptyAnswer` and
    `refusedAnswer`.
  - `"max"` maps to `EffortMax`.
  - `llm/doc.go`, `docs/guides/choosing-a-model.md` and `llm/llm.go:45`'s comment are corrected in the same commits.
- **D5. The Anthropic workspace is kept.**
  - The stored credential gains the wizard's `Result.Workspace`.
  - `llm/provider` builds the Claude provider, and its listings, with `claude.WithWorkspace` and
    `catalog.WithClaudeWorkspace`.
  - `ErrWorkspaceRequired` and `ErrWorkspaceNotFound` are worded for the user.

  Today a key not scoped to a workspace fails, before and after the pin alike. This makes it work. It changes
  `auth.json`'s shape, for Q5.

  *Facts found 2026-10-29, while writing the PLAN (read-only, gobble at `ccd8aa8`):*
  - *Only `/login` and `gobble setup` reach the wizard (`internal/cli/login.go:60`, `:201`, `:283`).*
  - *Three paths write `{type, key}` with no wizard, so no workspace can come from them:*
    - *`gobble auth login claude --api-key -` (`internal/cli/auth.go:123`), the Claude path the credentials guide
      documents;*
    - *`gobble acp --login` (`internal/cli/acp.go:56-79`), which ACP's terminal sign-in runs;*
    - *the Pi import (`internal/cli/piimport.go:197`).*
  - *`auth.Set` replaces the whole entry with `{type, key}` (`internal/auth/store.go:144-161`), so a stored
    workspace would be lost on the next key write. `SetOAuth`'s merge is the pattern that keeps fields.*
  - *`claude.WithWorkspace` and `catalog.WithClaudeWorkspace` are scoped to the Claude provider. In a shared option
    list they go inside `llmprovider.For(llmprovider.ProviderClaude, …)`.*
  - *The wizard reads `ANTHROPIC_WORKSPACE_ID` as its prompt's default only under `Options.AllowEnv`, which also
    offers the environment's key as a credential. gobble sets neither `AllowEnv` nor `LookupEnv` today.*

  *So Q5's answer covers the wizard's paths only. Whether the key-only paths take a workspace, and whether
  `AllowEnv` is set, are [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md)'s questions PQ1 and PQ2. Its workspace phase waits on them; no other phase
  does.*

  *(Answered 2026-10-29: PQ1 (a), a `--workspace` flag on `gobble auth login` and a prompt in `gobble acp --login
  claude`; PQ2 (a), `ANTHROPIC_WORKSPACE_ID` is not read. Q5's "the wizard's default answer" is corrected by PQ2.)*

### acp-go-sdk2

- **D6. The unstable names are kept, under a narrow, checked suppression** (Q2).
  - Each of F2's 12 lines carries `//lint:ignore SA1019` with a reason naming this record.
  - A gate run by `make staticcheck` refuses:
    - any ignore directive outside SA1019;
    - one without that reason;
    - any SA1019 finding on a name outside the acp-go-sdk2 package.

    It is seen failing first on a scratch copy.
  - The gate is the shape of magic-cli-remote's `scripts/check_staticcheck_ignored.py`, written for gobble's tree.
  - It lands in the pin's commit, since the pin fails `make staticcheck` without it.

  *Amended 2026-10-29, by the PLAN's deviation 3, from the owner's answer "House idiom + general gate
  (Recommended)".*

  *P7 found that the gate as written refused five older suppressions:*
  - *SA1019 on MCP roots, at `mcpclient/conn.go:136`;*
  - *ST1005 on Pi's error texts, at `mcpclient/value.go:76` and `:79`, and at `compaction/compaction.go:51` and
    `:53`.*

  *Each follows a recorded house idiom, 0002-PLAN Phase 5 deviation 3: a `//lint:ignore` line for standalone
  staticcheck, paired with `//nolint:staticcheck // as the lint:ignore above` on the line itself for golangci-lint.
  So D6 is carried out in that idiom:*
  - *each of F2's 12 lines has both;*
  - *`.golangci.yml` gains no exclusion, so no future deprecation is silenced centrally;*
  - *the gate, `scripts/check_staticcheck_ignored.py`, holds every suppression in the module to the idiom:*
    - *each `//lint:ignore` cites a record;*
    - *each is paired with a `//nolint:staticcheck`, and each `//nolint:staticcheck` with a `//lint:ignore`;*
    - *each ignored finding sits under a directive of its own check.*

  *MCP roots stays as 0002-PLAN decided. The MCP go-sdk deprecates roots by SEP-2577, "functional during the
  deprecation window (at least twelve months)", with migration through tool parameters, resource URIs or
  configuration. Moving off it is a product choice for a record of its own.*
- **D7. The five `SessionUpdate` literals become the SDK's constructors** (0034-MADR D2). `session.golden` stays
  byte-identical, and a difference stops the phase (0034-MADR D3).
- **D8. The refusals.** `acpclient`'s `handler` embeds `ClientRefusals`, and the nine stubs go. `acpserver.Agent`
  embeds `AgentRefusals`, and `DeleteSession` and `Logout` go. Tests that assert `-32601` keep asserting the wire code.
- **D9. `SessionUpdate.Kind()`** names a variant `translate` has no `Kind*` constant for (0034-MADR D5).
  `KindCompaction` stays `_gobble/compaction`.
- **D10. `PickPermissionOption`** replaces the reject-option loop.
- **D11. The connection's end.**
  - `acpserver.Serve` becomes `acp.ServeAgent`, with `WithInboundRewrite(acp.ResolveStdioCommands(exec.LookPath))`,
    the prompt overlap policy and the logger, followed by `agent.Close()`.
  - `endWait`, `ended`, `handlersDone` and `rewrite.go` go.
  - This fixes the overflow-as-clean defect.
  - On ctx's end, a prompt in flight is answered `cancelled`, where today the connection closes under it. That is a
    wire-visible change at shutdown only, and it has its own test.
  - `acpclient.Conn.Close` and `acptest`'s cleanups use `WaitContext`. `ScriptAgent` serves through `ServeAgent`.
- **D12. `acpclient.Start` runs the agent through `agentproc.InProcess`** (Q4).
  - Its pipes, goroutine and `closeTimeout` wait go.
  - `Close` gains the runner's three steps and `ErrAbandoned`.
  - This is the last acp-go-sdk2 phase, since it changes how the CLI's agent stops.

  *(2026-10-30, by the PLAN's deviation 9: held. At `v1.26.0`, `agentproc.Process.Close` waits without bound for a
  client handler that never returns, where gobble's `Close` is bounded. D12 waits for
  [acp-go-sdk2#12](https://github.com/maccavelli/acp-go-sdk2/issues/12), and `acpclient.Start` stays gobble's until a tag
  carries it.)*

### go-tui-lib

- **D13. `archtest` names `/v2`.**
  - `tuiLib` becomes `github.com/maccavelli/go-tui-lib/v2`.
  - A new case refuses the v1 path.
  - The `tuitest` exception matches `/v2/tuitest`.

  Each case is seen failing first on a scratch copy. This lands before, or in, the first import.

  *Amended 2026-10-29, by the PLAN's deviation 5, from the owner's answer "Teach archtest test closures":*
  - *`archtest` loads the graph with `go list -deps -test`, so that a go-tui-lib package only a test imports has its
    closure checked. P14's tests import `inline/inlinetest`, whose closure was unknown before.*
  - *A test build's package variants and test mains are skipped. `TestLoadGraphTestClosures` checks both halves.*
- **D14. Adoption order**, each its own phase with the pin in the first. 0030-MADR D5's benchmarks are no worse
  after any step. Steps 1 to 4 keep the CLI's goldens. Step 5 takes the library's look and re-records them (Q3):
  1. `ask`, which is 0029-PLAN P2 resumed (0032-MADR D4). It replaces the one-line prompt and `readWhy`.
  2. `safetext`, `termcap` and `glyph`, in place of `term/sanitize.go`, `term/caps.go` and `glyphsFor`.
  3. `pick`, for the numbered menus and the model and session choices.
  4. `lineedit`, in place of `internal/cli/editor`.
  5. `stream` and `transcript`, in place of `render`'s Markdown, tables and wrapping and `internal/cli/transcript.go`.

     By Q3's answer, gobble takes the library's defaults, to canonicalise its output:
     - tables are laid out with colour off too;
     - links are styled;
     - a multi-line setext heading draws only its last line as the heading;
     - `SGR 0` is written before a reprint;
     - rows are counted as terminal rows.

     No option is set to keep gobble's old look. Two guards come from gobble's own rules, not from that look:
     - **The notice's text is gobble's.** "Earlier output is not shown again" comes through `transcript.WithCatalog`
       from gobble's voice catalogue, since every text gobble shows can be overridden
       ([0022-MADR-gobble-voice-and-personalities.md](0022-MADR-gobble-voice-and-personalities.md)). Its style is
       the library's default.
     - **Piped output is unchanged.** Before the swap, the phase establishes whether output that is not a terminal
       (print mode, a pipe) goes through the renderer today, and in what form. If it is the reply's raw source today,
       it stays raw: the library's table layout and styling apply to a terminal only. So bytes a script reads do not
       change.

     The goldens are re-recorded. The phase's evidence is the golden diff, each change traced to one of the
     differences above, and reviewed. A change that matches none of them stops the phase.

  `tuitest` is not adopted here. Its closure would add `colorprofile`, `x/term` and `xo/terminfo` to `go.mod`, which
  0032-MADR A1.2 does not name.

  *Facts found 2026-10-29, while writing the PLAN.* They come from reading gobble at `ccd8aa8` and go-tui-lib at
  `v2.6.1` with `go doc`, and they correct this record's go-tui-lib table where they disagree with it:

  - ***`termcap` replaces nothing.***
    - `internal/cli/term/caps.go` detects TTY, colour, width and Unicode per stream (`Detect`, `:32`).
    - `termcap` has no TTY, width or locale detection: its colour profile is set by the host, and `Probe` needs a
      raw-mode reply loop.
    - The decisions that would match are in `launch`, which links Bubble Tea and is refused by D1. So `caps.go` stays.
  - ***`pick` has nothing to replace.***
    - The CLI has no model or session picker: `/model` is the agent's numbered table, answered with `/model N`
      (`acpserver/choice.go:301`).
    - `questions.choose` (`internal/cli/prompt.go:108`) is a cooked, numbered line question, run with no raw mode, and
      possibly on a stdin that is not a terminal (`internal/cli/auth.go:200`).
    - `pick` needs raw mode, filters by fuzzy match, and has no title row, no default and no single-option rule.
  - ***`safetext` cleans differently from `term.Sanitize`.***
    - `Text()` also removes U+200B, U+FEFF, U+2060–2064, U+00AD and tag characters, and turns U+2028 and U+2029 into
      `\n`.
    - Invalid UTF-8 gives one U+FFFD per maximal subpart, where gobble gives one per byte.
    - CAN and SUB abort a sequence.
    - An unterminated control string is abandoned after 64 KiB, and the text after it is shown, where gobble drops
      everything to the end.
    - `term.Sanitize` also cleans print mode's output (`internal/cli/print.go:201`). Replacing it there changes bytes
      a script reads, which Q3's guard keeps as they are.
    - `term.EscapeBidi` has no caller outside its test: it is dead code.
  - ***`ask` differs from 0029-MADR D1 in four places:***
    - D1 has a reject row ending "(hides the tool)", and `ask` has no suffix field;
    - D1 leaves a word wider than its row whole, while `ask` breaks it at the width;
    - D1 echoes after `> `, while `glyph`'s Unicode prompt is `❯`;
    - the reason prompt and echo texts are `ask`'s catalogue's, and gobble's voice catalogue would supply them
      through `WithCatalog`.

    `ask`'s reason limit, 2048 bytes, equals gobble's `maxWhy`.
  - ***`lineedit` changes the editor's behaviour:***
    - a trailing `\`, or alt or shift with enter, puts a newline inside one buffer, where gobble ends the line and
      joins the lines on submit;
    - a paste is inserted or submitted, with no "N pasted lines" notice and no joining;
    - alt+b/f/d move by alphanumeric words, where gobble's word moves go by spaces.

    The history file, the external editor's process, the Ctrl+C exit window, `/exit`, `Raw` and the 120-cell cap stay
    the host's, which is gobble's.
  - ***Two pieces of `render` have no library equivalent.***
    - Thinking text is wrapped by plain `render.Wrap` (`internal/cli/view.go:519`). `stream` has no plain, streaming
      wrapper, so `wrap.go` stays for it.
    - `tool.go`, `commands.go` and `message.go` use `table.go`'s width helpers, which `x/ansi`'s `StringWidth` and
      `Truncate` can replace (0032-MADR A1.4).
  - ***Print mode never goes through `render`.***
    - Print mode writes `term.Sanitize`d source (`print.go:201-214`).
    - A piped line view goes through `render.Stream` with no style, no table layout and no wrap, since colour is off
      (`view.go:274`, `line.go:254-259`).
    - `CLICOLOR_FORCE=1` with a pipe styles the output and lays its tables out, but does not wrap it.
    - No transcript is kept off a terminal (`chat.go:121`).
  - ***`transcript.Transcript` is used on one goroutine.*** gobble's is shared by three, under a mutex
    (`internal/cli/transcript.go:15`), so gobble keeps a locked wrapper.

  *So, of D14's steps:*
  - *step 1, `ask`, waits on the PLAN's PQ4;* *(Carried out 2026-10-29 by P14. By the PLAN's deviation 4, from the
    owner's answer "Catalogue the prompt words (Recommended)", the prompt's words are 24 `prompt.*` voice keys, with
    `ask`'s own three formats left at its defaults.)*
  - *step 2 becomes `glyph`, with `safetext` by PQ5, and `termcap` dropped;*
  - *step 3, `pick`, is PQ6;* *(2026-10-30, P16: neither `pick` nor `termcap` is adopted. `caps.go` stays gobble's, since
    `termcap` has no TTY, width or locale detection.)*
  - *step 4, `lineedit`, is PQ7;*
  - *step 5 stands, with `wrap.go` kept for thinking text and the width helpers moved to `x/ansi`.*

  *(Answered 2026-10-29: PQ4 (a) `ask`'s behaviour, 0029-MADR D1 amended; PQ5 (a) `safetext` on terminal paths, print
  mode and the piped view keeping `term.Sanitize`; PQ6 (a) neither `pick` nor `termcap`; PQ7 (a) `lineedit`'s
  behaviour, gobble keeping its host parts.)*

  *The Consequences' "up to about 3,700 lines" is high: `caps.go`, `wrap.go`, `external.go` and the history and
  Ctrl+C code stay. The PLAN's phases record what each one removes.*

### The records, the docs, and Magic

- **D15. The records.** On acceptance:
  - 0034's MADR and PLAN become `superseded by 0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md`;
  - 0031-MADR D1 and D9 get dated notes pointing here;
  - 0032's MADR and PLAN get dated amendments: the `/v2` path, the closure table above, and P3 onward carried out by
    this record's plan;
  - 0029-PLAN P2 gets a dated note that it resumes here;
  - 0016-MADR D13 gets a note that it is carried out by D7.

  Historical rationale is annotated, not rewritten.

  *(2026-10-29, at acceptance: carried out by [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md) P0, a commit of records only.)*
- **D16. The docs, one phase ahead of the pins** (it needs no tag):
  - `auth.go:192` and `credentials.md` name 0031-PLAN P4b rather than the SDK;
  - `docs/README.md`'s build order gains 0023 to 0035, and its "I want to…" row points here;
  - `README.md` and `docs/architecture.md` move with each pin, in that pin's commit.
- **D17. Magic's D12 does not gate gobble** (Q1).
  - gobble moves when its own gates pass.
  - gobble's issue and record name the tag it pins, so Magic can check its D12 against it.
  - At `v1.26.0` the two repositories are on the same tag.
  - [gobble-cli#6](https://github.com/maccavelli/gobble-cli/issues/6) item 2, and [#5](https://github.com/maccavelli/gobble-cli/issues/5),
    are answered by this record's plan.
- **D18. Not this record**, each for its own record when wanted:
  - image input;
  - `chatcompat` and `Discover`;
  - keeping learned reasoning refusals across runs;
  - `Catalog.Search`;
  - the SDK's `acptest` in place of gobble's;
  - `clienthost`, the timelines, `PresentNotice` and `PresentCompaction` (0016-MADR D8–D10);
  - `ExtensionMux`;
  - native tool deferral;
  - 0031-PLAN P4b;
  - the `proctree` race;
  - go-selfupdate-lib `v1.23.0`.

  Each gives gobble something new, rather than removing a copy. The waits listed above stand.

  *(2026-10-30, by the PLAN's P16 and PQ6 (a): go-tui-lib's `pick` joins this list. The CLI has no picker for it to
  replace, so a picker is new UI, for a record of its own.)*

  *(2026-10-29: [0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md](0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md), accepted the same day in another session, is the record for go-llmprovider-sdk's
  remainder:*
  - *one shared registry;*
  - *the SDK's warnings;*
  - *learned refusals kept across runs;*
  - *Chat Completions servers from models.json;*
  - *the facts search;*
  - *images.*

  *Its D1 starts after this record's go-llmprovider-sdk phases, and its D9 leaves the pins, the offline row, the
  copies and the workspace here. So of this list, image input, `chatcompat`, the learned refusals and
  `Catalog.Search` are 0036's, and `Discover` in `gobble setup` is left by its D5 to a later record.)*

### Questions for the owner

- **Q1. Magic's rule (D17).** Should gobble not gate on Magic (recommended)? Magic has already moved to `v1.26.0` on
  `master`, and its own amendment H puts gobble after it in gobble's repository. Or should gobble adopt D12 as its
  own rule, so that every acp-go-sdk2 bump waits on Magic's tag and both live ACP suites?
- **Q2. The unstable names (D6).** The recommended answer is a narrow suppression with a gate, keeping
  `PromptResponse.Usage`, which Magic's client reads, and the tests that assert what gobble does not advertise. The
  alternative is to stop sending per-turn usage on the prompt answer, and rewrite those tests without the deprecated
  names. That drops data a client reads, and the tests' wire checks become JSON literals.
- **Q3. go-tui-lib's output (D14 step 5).** Should gobble keep today's output through the library's options, with
  goldens unchanged except where no option exists (recommended)? Or take the library's defaults, re-recording the
  goldens, so that tables are laid out with colour off and links are styled?
- **Q4. `agentproc.InProcess` (D12).** In this record as the last acp-go-sdk2 phase (recommended), or left for a later
  record?
- **Q5. Where the workspace is kept (D5).** In the stored credential beside the key (recommended)? Or only from
  `ANTHROPIC_WORKSPACE_ID`, with `auth.json` unchanged?
- **Q6. Order.** The recommended order:
  1. D16, the docs;
  2. D2, check-libs;
  3. go-selfupdate-lib;
  4. go-llmprovider-sdk: D3, then D4, then D5;
  5. acp-go-sdk2: D1 with D6, then D7 to D11, then D12;
  6. go-tui-lib: D13, then D14.

  Alternatively go-tui-lib's `ask` first, as #6 ranks it: it waits on no SDK.

**The owner's answers, 2026-10-29** (three options offered per question, a recommendation first):

- **Q1.** "No gate (Recommended)". D17 stands. gobble does not adopt Magic's D12 as a gate.
- **Q2.** "Narrow checked ignore (Recommended)". D6 stands.
- **Q3.**
  - First answer: "i think we should adopt the library for as much as we can to canonicalize the code. option 1 vs
    option 3, pros, cons. explain."
  - Asked again with the comparison, the answer was "Defaults + gobble guards (Recommended)": the library's
    defaults, with the notice's text from gobble's catalogue and piped output kept raw where it is raw today.
  - D14 step 5 is written to that answer, superseding the draft's "keep today's output". The rejected options were
    pure library defaults (the library's English notice text, against 0022-MADR) and today's output kept through
    `WithoutTableLayout` and `WithNoticeStyle(Dim)`.
- **Q4.** "Yes, last acp phase (Recommended)". D12 stands, as the last acp-go-sdk2 phase.
- **Q5.** "Stored with credential (Recommended)". D5 stands:
  - `auth.json`'s Claude credential gains the workspace;
  - `ANTHROPIC_WORKSPACE_ID` stays only the wizard's default answer;
  - a stored file without the field still reads.
- **Q6.** "Risk-first sequence (Recommended)". The order is the one listed above.

**Accepted by the owner, 2026-10-29:** "mark the madr accepted". The plan is [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md).

### Consequences

* Good, because every pin is current, and #4, #5 and #6's library items are answered in one place.
* Good, because up to about 3,700 lines of gobble's own code are candidates to go:
  - the editor, the renderer's Markdown and wrapping, the transcript, the sanitiser, the caps, the prompt;
  - in `acpserver`/`acpclient`: the serve loop, three `handlersDone`, eleven stubs, the rewrite, five literals;
  - in `llm/provider`: the offline walk and the suffix match.

  Each phase confirms its own count.
* Good, because the overflow-as-clean defect is fixed, and an interrupted `gobble acp` answers its prompt `cancelled`.
* Good, because F1 is fixed as the pin lands, and `check-libs` would catch the next F2.
* Good, because the permission prompt, pickers and editor become the components both of gobble's modes will share.
* Bad, because the suppression keeps names the SDK has deprecated. A future `/v2` of acp-go-sdk2 may remove them, and
  then gobble's per-turn usage needs a stable field or goes.
* Bad, because go-tui-lib is young: three majors in October, and seven tags since `v2.0.0`. A `/v3` would mean an
  import-path move. The pin buffers it, and nothing moves without a gobble record.
* Bad, because it is many phases, each with three-host gates, before gobble's feature work resumes.
* Bad, because the CLI's terminal output visibly changes at D14 step 5 (Q3). Its evidence is a reviewed golden diff,
  not an unchanged golden.
* Neutral, because the default binary gains `x/ansi` and its four text modules, as 0032-MADR A1 already accepted.
* Neutral, because the SDK's learned reasoning resends, at most 7 sends where there was one resend, may bill a refused
  request more than once before it is learned.

### Confirmation

[0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md) proves each decision as follows:

- **The pins:** `go list -m` reports each tag of D1, and the `ci.yml` pins match go-selfupdate-lib's.
- **check-libs:** it fails on a scratch copy with a deprecated use planted (D2).
- **Offline:** `TestWireOffline` passes with D3's row and fails without it, on a scratch copy.
- **The suppression:** `make staticcheck` reports exactly F2's findings as ignored, and the gate fails on a planted
  directive outside them (D6).
- **The wire:** `session.golden` is unchanged (D7). No non-test `handlersDone`, `ended`, `resolveBareCommands`,
  `NewMethodNotFound` stubs or `SessionUpdate{…Update:` literals remain in `acpserver` or `acpclient` (D7 to D11). A
  test shows a prompt answered `cancelled` when `ServeAgent`'s ctx ends (D11).
- **The boundary:** `archtest` refuses a v1 go-tui-lib import and accepts `/v2/tuitest` in a test, each seen failing
  first (D13).
- **The CLI's output:** the goldens are unchanged, or re-recorded as named, and 0030-MADR D5's benchmarks are no worse
  (D14).
- **Every phase:** `make preflight` and `go test -race ./...` pass on macOS, the Linux host and the Windows host.

## Pros and Cons of the Options

### A. Pin only, fix what breaks

* Good, because it is three small commits: a sentinel row, a suppression with its gate, and two workflow SHAs.
* Bad, because every copy the libraries now replace stays, against 0004-MADR. That includes the overflow-as-clean
  defect, which only `EndedCleanly` (or a copy of it) fixes.
* Bad, because go-tui-lib stays unrequired, and 0029-PLAN P2 stays stopped though its component is tagged.

### B. One ordered sequence of pins and adoptions

* Good, because it does what 0004-MADR and the owner's direction ask, phase by phase, each reversible.
* Good, because the cross-library facts (F1, F2, check-libs, archtest's path, Magic's rule) are decided once.
* Bad, because one record covers four libraries. Its plan must keep each phase independent, so that a stall in one
  library does not hold the others.

### C. Amend 0031, 0032 and 0034 separately

* Good, because each library's history stays in its own record.
* Bad, because 0034 is three acp-go-sdk2 tags behind, and amending it to `v1.26.0` turns most of it into new text.
* Bad, because F2, D2 and D17 belong to no single one of them, and would be split or repeated.

### D. Wait for Magic's tag and for go-tui-lib to settle

* Good, because Magic's D12 would hold by the letter: a tag, not a `master` commit.
* Bad, because no gobble record makes Magic a gate, and Magic itself puts gobble after it in gobble's own repository.
* Bad, because "settled" has no measure. go-tui-lib's issue tracker is empty, and every package gobble would use is
  tagged with its tests built from gobble's own cases.

## More Information

- **gobble's issues:**
  [gobble-cli#3](https://github.com/maccavelli/gobble-cli/issues/3) (P4b only),
  [#4](https://github.com/maccavelli/gobble-cli/issues/4),
  [#5](https://github.com/maccavelli/gobble-cli/issues/5) and
  [#6](https://github.com/maccavelli/gobble-cli/issues/6).
- **gobble's records:**
  - [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): wait, then drop the copy; rule 5.
  - [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md): D5,
    D8–D10 and D13.
  - [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md):
    D14, `check-libs`.
  - [0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md): A2, the workflow pins follow the library.
  - [0029-MADR-permission-prompt-layout.md](0029-MADR-permission-prompt-layout.md) and
    [0029-PLAN-permission-prompt-layout.md](0029-PLAN-permission-prompt-layout.md): P2.
  - [0030-MADR-resize-and-performance.md](0030-MADR-resize-and-performance.md): D5, the performance budget.
  - [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md): D1, D8 and D9.
  - [0032-MADR-go-tui-lib-integration-boundary.md](0032-MADR-go-tui-lib-integration-boundary.md): D1, D4, D7, A1.
  - [0034-MADR-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md](0034-MADR-pin-acp-go-sdk2-v1-21-0-and-adopt-session-update-helpers.md):
    superseded on acceptance.
- **The libraries' records, in their repositories:**
  - acp-go-sdk2: `0036`, `0038`, `0039`, `0041` and `0042-MADR`, and `docs/reports/0042-REPORT-release-notes-v1-26-0.md`;
  - go-llmprovider-sdk: `0061`, `0062` and `0064-MADR`, and its `CHANGELOG.md`;
  - go-selfupdate-lib: `0038-MADR` and its `CHANGELOG.md`;
  - go-tui-lib: `0020` and `0023-MADR`, and issue 10's closing comment;
  - magic-cli-remote: `0179`, `0184` (D12), `0207` (amendments H and I) and `0208-MADR`.
- **Found while reading, outside this record:** go-selfupdate-lib's `main` has untagged `service` fixes that its
  `v1.23.0` will carry. gobble imports no `service` package.
