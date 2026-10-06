---
status: in-progress
date: 2026-10-05
associated-madr: "0002-MADR-cli-acp-headless-mcp-v1.md"
---
# Implement v1 as the native magic-cli-remote CLI: Kong over ACP, ACP stdio, MCP client

Associated MADR: [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md)

## Goal

A Go module in this repository that builds one binary. That binary (1)
serves ACP protocol version 1 on stdio as the agent process magic-cli-remote
already knows how to spawn, (2) exposes the same agent methods through
Kong, (3) connects to MCP servers over stdio and streamable HTTP so
their tools are callable during a prompt, and (4) executes agent-owned
slash commands (`/compact`, `/usage`, `/context`, `/model`, `/thinking`,
and the native set in the MADR) over `session/prompt`, with
`available_commands_update` listing only those handlers.

## Scope

In:

* `go.mod`, Kong, `github.com/coder/acp-go-sdk v0.13.5`. The config surface is undecided and will be either a native Kong facility or a surface we write.
* Agent loop sufficient to complete `session/prompt` with built-in tools
  `read`, `bash`, `edit`, `write` (grep/find/ls may follow in the same
  phase if cheap)
* ACP agent: `initialize`, `session/new`, `session/prompt`,
  `session/cancel`, `session/close`; then `session/list`, `session/resume`,
  `session/load`, `session/set_mode`, `session/set_config_option`
  (~~`session/set_model`~~ withdrawn 2026-09-29; see Amendments)
* Honest `available_commands_update` and executing slash handlers per the
  MADR native set
* Standard `usage_update` after prompts and after compact
* Session modes `default` and `plan`
* `_gobble/` extension methods listed in the MADR mapping table that the phase
  claims (`_gobble/compact`, `_gobble/usage`, `_gobble/steer`, `_gobble/follow_up`,
  `_gobble/clear_queue`, and the read/name helpers the phase freezes)
* MCP client: stdio + streamable HTTP; `gobble mcp add|remove|list`
* JSONL session persistence enough for `session/load` / `resume`
* CI: `go test` on darwin, linux, windows

Out (MADR): Charm TUI, TypeScript extensions, Pi JSONL RPC, codemode,
Chord/Pico, HTML export, npm package manager, SSE MCP, ACP-transport MCP
(`mcpCapabilities.acp`), mcremote/mcrelay WebSocket, `_x.ai/` methods.

These stay out of **this** plan. The widened v1 line ([0005-MADR](0005-MADR-v1-feature-scope.md)) delivers the
TUI, HTML export, data packages, the Pi RPC shim, and `exp/codemode` in
[0005-PLAN](0005-PLAN-v1-feature-scope.md), which runs after this plan.

Out of **this** repository: registering `provider.IDPi` in
magic-cli-remote. The companion checklist at the end of this plan is the
contract for that tree; it is not a phase here.

Chosen elsewhere, recorded here so a phase does not re-open them:

* D10, D14, D15, D16 — [0004-MADR](0004-MADR-go-module-architecture.md)
  (2026-09-30: 1.0 providers are the `llm/provider` adapter over
  `go-llmprovider-sdk`; MCP client is the official go-sdk).
* D11, D17 — [0003-MADR](0003-MADR-gobble-product-identity.md).

The version of `go-llmprovider-sdk` and of
`github.com/modelcontextprotocol/go-sdk` is recorded as a dated note in
the phase that first imports them. That is not a new architectural
choice.

## Implementation Steps

### Phase 0 — module and Kong skeleton

> *Superseded (2026-09-29) by [0004-PLAN](0004-PLAN-go-module-architecture.md), whose Phase 0 and Phase 1 have run.* That plan builds the module, package
> skeleton, Kong root, CI, and release pipeline to [0004-MADR](0004-MADR-go-module-architecture.md).
> Phase 1 of this plan starts from the tree 0004-PLAN leaves. Do not run the steps below as a second scaffold.

1. `go mod init` with the module path chosen for D17 (placeholder
   `github.com/maccavelli/gobble-cli` unless a MADR says otherwise).
2. `cmd/` main → `internal/cli` Kong root. Subcommands: `acp` (stdio
   agent), `prompt`, `mcp`, `auth` (auth may stub).
3. The file path is a constant until D11 is decided. The config surface is undecided and will be either a native Kong facility or a surface we write. Config implementation stays unspecified.
4. Makefile / CI: `go test ./...`, `gofmt`, vet on three OS.
5. Pin `github.com/coder/acp-go-sdk v0.13.5`. No Charm modules.

**Accept:** `go test ./...` passes; `go list -m github.com/coder/acp-go-sdk`
prints `v0.13.5`; `go list -deps` has no `charm.land` and no
`github.com/charmbracelet` module.

### Phase 1 — ACP stdio hello

1. Implement `acp.Agent` with `Initialize` advertising protocol `1`,
   `loadSession: false`, `mcpCapabilities` all false, empty `authMethods`.
   Capabilities flip to their real values in the phases that implement
   them (honest advertisement).
2. `NewSession` returns a session id; `Prompt` echoes a single
   `session/update` agent message chunk and a stop reason; `Cancel` and
   `CloseSession` succeed.
3. Wire `gobble acp` to
   `acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)`.
4. In-process test: `ClientSideConnection` over pipes, using the SDK,
   runs Initialize → NewSession → Prompt → CloseSession.

**Accept:** that test is red on a deliberately broken `Initialize`
(wrong protocol version) before it is green on the real agent. The
failure text is quoted in the phase handoff. `gobble acp` + SDK example
client (or the in-tree client test) completes a prompt.

### Phase 2 — Kong is an ACP client

1. Shared constructor: agent + pair of connections over `io.Pipe` (or
   equivalent), used by Kong and by tests.
2. `gobble prompt "…"` is `session/new` + `session/prompt` + print of
   agent text chunks to stdout, then `session/close`.
3. A test that the same `Prompt` params are sent whether the caller is
   the Kong command or a stdio client (compare JSON-RPC method name
   `session/prompt`).

**Accept:** `gobble prompt` does not import or call the agent loop type; it
calls ACP. A grep of `internal/cmd` for the agent-loop package fails
(only `internal/acp` / the connection package is allowed). *(2026-09-29:
per [0004-MADR](0004-MADR-go-module-architecture.md) the packages are
`internal/cli` and `acpclient`, and the check is archtest rule 6 rather
than a grep.)*

*(2026-10-05: steps 1–3 are expanded into executable steps in the Amendments entry "Phase 2 made executable" of this date. Step 2's `gobble prompt` is `gobble -p`, per the 0008 amendment.)*

### Phase 3 — real prompt loop and built-in tools

1. Agent loop: one provider through `llm/provider`, which is the
   `go-llmprovider-sdk` adapter (0004-MADR D10, 2026-09-30). First import
   of `github.com/maccavelli/go-llmprovider-sdk` lands here: pin a commit
   pseudo-version of that module's `origin/main` (probed
   `3d4aff5f2be9363877aae0987755e6781f1cae98` on 2026-09-30; re-resolve
   and record the chosen version here). Construct with
   `providers.New("openai", …)` or another id that already implements
   `llmprovider.Provider`. Call `llmprovider.Stream`, not the pre-S8
   `Generate*` API. Tests use `llm/llmtest` (a scripted fake that can
   emit per-token deltas); they do not need the SDK. Native
   `NativeStreaming` is `Unsupported` on every moved SDK provider, so a
   live provider in this phase will emit coarse chunks. The faux
   provider still streams deltas so ACP `session/update` tests stay
   strict.
2. Tools: `read`, `write`, `edit`, `bash` with the session `cwd`.
3. Stream ACP `session/update` for message chunks and tool calls.
4. `session/cancel` aborts the in-flight provider request and tool.
5. After each completed prompt, emit `usage_update` with at least `used`
   and `size` (size may be 0 until the model catalog supplies a window).
6. *(Added 2026-10-05, owner's decision; see Amendments.)* Draw ACP diff
   content in the CLI as a unified diff: a Myers line diff in
   `internal/cli/render`, tested on the edit tool's real output.

**Accept:** a faux-provider test: prompt that requires `read` of a temp
file produces a tool_call update and a final agent message containing
the file's contents. Cancel during a blocked bash is observed as a
cancelled stop reason. A completed prompt produces a `usage_update`
notification. A test that strips `usage_update` is shown red on a
client that requires it, then green on the real agent.

### Phase 4 — sessions

1. JSONL persistence. Compatibility with TypeScript Pi v3 is **not**
   required until D11/D5 say so; the format must be specified in a short
   in-tree comment or guide and round-trip in tests. *(2026-09-29:
   [0005-MADR](0005-MADR-v1-feature-scope.md) decides D5. Write the v3
   entry shapes from the start; full v3 parity is
   [0005-PLAN](0005-PLAN-v1-feature-scope.md) F2.)*
2. Enable `loadSession` / `session/list` / `session/resume` /
   `session/load` as the MADR table requires. Flip
   `AgentCapabilities.LoadSession` to true in `Initialize` in this phase.
3. `session/set_config_option` for model id and thinking level.
4. ~~`session/set_model` `{sessionId, modelId}` as mcremote `acpagent` already
   sends; honor `_meta.reasoningEffort` when present.~~ *Withdrawn
   (2026-09-29):* the SDK cannot route a non-schema method to the agent.
   Model and thinking are `session/set_config_option` categories `model`
   and `thought_level`, plus `/model` and `/thinking`.

**Accept:** NewSession → Prompt → CloseSession → LoadSession (or Resume)
replays or continues; list includes the id. A corrupt JSONL file fails
closed with an ACP error, not a panic. set_config_option (category
`model`) changes the model id observed on the next prompt. A
`session/set_model` request gets MethodNotFound, which is asserted, not
tolerated. Thinking mutability is
live (applies in this session).

### Phase 5 — MCP client

1. MCP client module, already chosen: `github.com/modelcontextprotocol/go-sdk`
   (0004-MADR D14). Record the resolved version here as a dated note.
2. Stdio and streamable HTTP. Advertise `mcpCapabilities.http = true`,
   `sse = false`, `acp = false`.
3. Union `session/new.mcpServers` with on-disk `mcp.json` (session name
   wins).
4. `gobble mcp add|remove|list`. `list` connects and prints tools; exit 1 on
   failure, matching TypeScript Pi.
5. Tool names `mcp__<server>__<tool>`.
6. OAuth: implement if the client library supports it in this phase;
   otherwise record the gap here before merging.

**Accept:** two fixtures — a stdio MCP server in-tree and an HTTP test
server — appear as tools during `session/prompt`. `gobble mcp list` agrees.
An SSE-only config is rejected. The Phase 1 initialize test still
passes (capabilities now include http).

### Phase 6 — native slash commands and session modes

This phase is the magic-cli-remote-native contract. An empty
`available_commands` list is **not** acceptable.

1. Implement `session/set_mode` with modes `default` and `plan`. Advertise
   them on `session/new`. Confirm switches with `current_mode_update`.
   Plan mode refuses edit/write/bash that mutate; `session/request_permission`
   is the approval path, not a grok `_x.ai/exit_plan_mode` notification.
2. Slash router inside `Prompt`: if the user text is `/name …`, dispatch
   rather than sending the slash string to the model. Unknown `/name` that
   was not advertised returns a short agent message that it is unknown,
   not a model turn.
3. Execute and advertise at least:
   * `/compact [instructions]` — summarise in place (TypeScript Pi
     compaction: replace history with a summary, keep going). Stream
     progress as `session/update`. Emit `usage_update` afterwards.
   * `/usage` — session tokens, cost, and any rate-limit/account figures
     the provider exposes; if account figures are unknown, say so and
     still print session usage.
   * `/context` and `/session` — context-window usage (same numbers as
     the last `usage_update`, plus session id / message count).
   * `/model [id]`, `/thinking [level]` — same effect as the ACP methods
     in Phase 4.
   * `/name [name]`, `/fork [turn-id]`, `/clone`, `/help`.
4. Do **not** advertise TUI chrome: `/settings`, `/hotkeys`, `/quit`,
   `/copy`.
5. Do **not** implement `_x.ai/` methods.
6. Freeze the advertised name list in this PLAN when the handlers land
   (dated note). Skills/templates may add names later; v1 may ship
   without those resources.

**Accept:** an in-process ACP client sends `/compact`, `/usage`, and
`/context` as `session/prompt` and receives `session/update` frames with
non-zero content. The same three tests are shown **failing first** on a
copy of the agent whose `Prompt` ignores leading slashes (echo or empty
turn — the grok silence shape); quote that failure in the handoff. After
compact, a subsequent prompt sees the shortened history. `set_mode` to
`plan` then `default` emits `current_mode_update` for each. A
deliberately advertised `/bogus` that has no handler fails a test that
asserts advertisement ⊆ handlers. `go list` / grep of the module for
`_x.ai/` is empty.

### Phase 7 — `_gobble/` extensions and remaining Kong commands

1. Implement `_gobble/compact` (optional instructions), `_gobble/usage`,
   `_gobble/steer`, `_gobble/follow_up`, `_gobble/clear_queue` against the agent
   queue. Compact via the extension is the same compaction as `/compact`.
2. Implement the read extensions used by the CLI (`_gobble/get_state` or
   whatever names the mapping table freezes in this phase — freeze them
   in this PLAN when the handlers land).
3. `_gobble/set_session_name` matches `/name`.
4. `gobble auth` as far as the chosen provider requires (may be env-only).

**Accept:** SDK `CallExtension` for `_gobble/steer` during a live faux prompt
queues a steer; `_gobble/compact` shortens history the same way `/compact`
does; unknown `_nope` returns method-not-found. Stdio clients that never
call extensions still complete Phase 3 and Phase 6 tests.

### Phase 8 — hardening

1. Windows stdio (ACP on pipes) and process-group kill for MCP stdio
   children, matching TypeScript Pi's "close stdin, SIGTERM, SIGKILL
   process group" intent on unix.
2. `go test` on the three OS in CI.
3. Update [architecture.md](../architecture.md) to describe the packages
   that actually exist (no rationale).

**Accept:** CI green on darwin, linux, windows. architecture.md names
every package under `internal/`, and archtest shows Charm imported only
by `internal/tui` ([0004-MADR](0004-MADR-go-module-architecture.md) rule 5).

## Verification

Repository-level, after the last in-scope phase:

```text
go test ./...
go vet ./...
gofmt -l .   # empty
make archtest   # replaces the former Charm grep; Charm only in internal/tui
```

Plus the per-phase accepts above. New gates must be shown failing on
broken input on a copy or scratch clone before they are trusted (global
agent rule). The Phase 6 slash tests in particular must have been seen
failing on an echo-only Prompt.

A stdio script that speaks only the JSON-RPC mcremote `acpagent` already
sends — `initialize`, `session/new` (with `mcpServers: []`),
`session/prompt` `/compact`, `session/set_mode` `plan`,
`session/set_config_option` (category `model`) — completes without
unknown-method errors. (`session/set_model` was dropped from this script
on 2026-09-29; see Amendments.)

## Companion work (magic-cli-remote, not this tree)

Do not mutate magic-cli-remote under this plan. When that repository
grows an `IDPi` provider, it needs its own MADR/PLAN pair. The contract
is the 2026-10-01 amendment of
[0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md).
Checklist:

1. `acpagent.Spec` with `ID` a new `provider.IDPi`, `DefaultBin` `gobble`,
   `DefaultArgs` `[]string{"acp"}`. `SessionMeta` nil.
   `SynthesizeAutoMode` false. `ExtensionNotifications` empty.
2. Parameterize Compact / Fork / Rename / Usage / Status / SetModel /
   SetThinking / Undo on the Spec (today they are grok `_x.ai/` and
   raw `session/set_model` on `*session`). Until those fields exist,
   the `IDPi` table uses `KindNative` for compact, usage, model,
   thinking, fork, and undo.
3. `command.Table` declaring every canonical Specs name. `KindNone`
   reasons in words a user reads. Default Specs `/model` is
   `KindDaemon` (relaunch); `IDPi` must not leave that default in
   place.
4. `ConfigureSession` applies the start-up model through
   `session/set_config_option` category `model`.
5. Map ACP `session_info_update` to `event.TypeSessionTitle` so `/name`
   is visible on the phone.
6. Probe, do not assume: live-tagged tests send `/compact`, `/usage`,
   `/context` and assert `session/update` frames. `/settings` is absent
   from `available_commands`. `session/set_model` is MethodNotFound.
   `_x.ai/` is absent from gobble.
7. Empty `CommandCaveat` unless a real quirk appears.
8. `KnownGoodVersion` once a gobble release exists; mismatch warns
   (MADR 0137).
9. Every connection uses `OverflowDropNewest` (MADR 0167), matching
   gobble's Phase 0 ACP `replace`.

Existing providers (grok, opencode, kilo, codex, fake) stay registered.

## Rollout and Rollback

No production users. Rollback is `git revert` of the phase commit.
Do not tag a v1 release until D17 (identity / User-Agent) is decided;
talking to providers with a placeholder name is a live-test-only act.

magic-cli-remote does not gain `IDPi` as a side effect of this plan.

## Execution record

~~None. This plan is proposed and has not been approved.~~ *(Superseded 2026-10-05: the owner approved Phase 1.)*

**Phase 1, 2026-10-05 — complete (staged; the owner commits).** It ran as expanded in the amendment of 2026-10-05, with capabilities advertised honestly per phase (owner's decision). Phases 2–8 have not run.

* **Files.**
  * `acpserver`: `doc.go`, `agent.go`, `serve.go`, `agent_test.go`, `binary_test.go` and `testdata/session.golden`.
  * `internal/cli`: `acp.go`, the `root.go` help line for `acp`, and two `main_test.go` cases. `gobble acp` with an empty stdin now exits 0, still writing nothing.
  * `gobble acp` now opens the log file (`<state>/logs/gobble.log`) at start. It hands the SDK connection the CLI's lazy logger, and the stub before it never logged. `TestGobbleACPBinary` points `GOBBLE_HOME` at a temporary directory for this reason.
* **Accept.**
  * The session test was red on a scratch copy whose `Initialize` answers protocol 2, failing with `agent_test.go:53: protocol version 2, want 1`. It is green on the real agent.
  * `TestGobbleACPBinary` drives the built `gobble acp` with the SDK's client through Initialize → NewSession → Prompt → CloseSession. Every stdout line is JSON-RPC, stderr is empty, and closing stdin ends the process with 0 within 2 s.
* **The handshake, as `testdata/session.golden` records it:**
  * `agentInfo {"name":"gobble","title":"gobble","version":…}`;
  * `sessionCapabilities {"close":{}}` and nothing else;
  * the echo's `session/update` arrives before the prompt's `end_turn` response.
* **Gates.**
  * `make preflight` printed `preflight passed` on Windows and in WSL.
  * `go test -race ./...` passed in WSL.
  * `golangci-lint` reported `0 issues.` for linux, darwin and windows.
  * The pre-add check was clean for all 8 files.
* **Negative tests**, each on a scratch copy, each failing as expected:
  * protocol 2, as quoted above;
  * `loadSession` advertised: `loadSession advertised before 0002-PLAN Phase 4 implements it`;
  * a start-up banner: `stdout line 1 is not a JSON-RPC message: "gobble acp ready"`;
  * `Serve` ignoring end of file: `gobble acp did not exit within 2 s of stdin closing`.
* **What the plan predicted wrongly.**
  1. **Step 3's `acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)` cannot appear in `internal/cli`:** rule 2 keeps the SDK in `acpserver`. `acpserver.Serve` owns the connection, and `cli` passes it the writers `Main` received, so C3 holds.
  2. **The 0008 amendment's D19 capability list conflicted with step 1.** The owner kept step 1's honest advertisement.
  3. **The SDK interface has methods this phase does not implement** (`authenticate`, `logout`, `session/list`, `session/resume`, `session/set_mode`, `session/set_config_option`). Each answers -32601, matching what is advertised.
  4. **The root `tools.go` still blank-imports the SDK,** though `acpserver` now imports it for real. Removing that pin, and archtest rule 2's exception for the module root, is left for a change that names both files.

**Phase 2, 2026-10-05 — complete (staged; the owner commits).** It ran as expanded in the amendment "Phase 2 made executable", with the owner's three decisions of that day:

* diff content waits for Phase 3;
* a failed turn still writes a result;
* the `SetLogger` race is fixed in the SDK fork first.

Phases 3–8 have not run.

* **Files.**
  * `acpclient`: `doc.go`, `conn.go`, `update.go` and `conn_test.go`.
  * `acpclient/acptest`: `script.go` and `script_test.go`.
  * `internal/cli`:
    * new: `agent.go`, `print.go`, `line.go` and `view.go`, with `print_test.go`, `line_test.go`, `view_test.go` and five goldens;
    * changed: `chat.go`, `input.go`, `flags.go`, `main.go` and `signal.go`, with their tests.
  * `internal/cli/render`: `plan.go` and `plan_test.go` with two goldens, and `Stats` in `status.go`.
  * From the deviation: `acpserver/serve.go`, `go.mod` and `go.sum` (`replace` to `v0.13.6-mcr.2`).
  * Records: `0004-MADR-go-module-architecture.md` (the fork's facts), `0005-MADR-v1-feature-scope.md` (the error field), and `0009-REPORT-magic-cli-remote-findings.md` M8.
* **Accept.**
  * `gobble -p` reaches the agent only through `acpclient`. Archtest rule 6 passes in `make preflight`, and `internal/cli` imports no SDK package (rule 2).
  * `TestPromptParamsMatchSDKClient` shows `acpclient` and the SDK's own client sending the same canonical params, `{"prompt":[{"text":"hi","type":"text"}],"sessionId":"s1"}`.
  * `TestPrintSendsACP` shows `gobble -p hi` sending exactly that, in the order `initialize`, `session/new`, `session/prompt`, `session/close`.
* **Gates.**
  * `make preflight` printed `preflight passed` on Windows and in WSL.
  * `go test ./...` on Windows: 17 packages `ok`, no `FAIL`.
  * WSL `go test -race ./...` exited 0 with 0 race reports. A further `-race -count=30` of the CLI print, line, exit-code and `acp` tests exited 0 with 0 reports.
  * `golangci-lint` printed `0 issues.` for linux, darwin and windows (with `CGO_ENABLED=0`).
  * The pre-add check was clean for all 25 Go files.
* **Negative tests,** each on a scratch copy, each failing as expected:
  * `Prompt` sending the text twice: `conn_test.go:149: acpclient sent […{"text":"hi"…},{"text":"hi"…}…], the SDK client sent […]` and `print_test.go:70: prompts […], want {"prompt":[{"text":"hi","type":"text"}],"sessionId":"s1"}`;
  * `session/new` sent at start: `line_test.go:120: before any prompt the agent received initialize,session/new; want only initialize`;
  * an interrupt hook that is never called: `line_test.go:157: condition not met within 5 s`, then `the line session did not end`;
  * the result object dropped on error: `print_test.go:141: last line "": jsontext: unexpected EOF`;
  * a `stream-json` header without `cwd`: `print_test.go:118: print-stream-json differs from testdata\print-stream-json.golden`;
  * the logger race, before the fix:
    * against `mcr.1`, WSL `-race` reported 2 races in `TestMainExitCodes`;
    * a clean clone of `13253a5` showed 1 race in 30 runs;
    * the fork's new `TestSetLoggerWhileReading` reported 14 races on the `mcr.1` code and none on `mcr.2`.
* **A real terminal.**
  * In WSL, a built `gobble` driven through a pty passed 10 of 10 sessions. Each showed the prompt, the typed line, the spinner, the echo agent's reply, a `0s` status line and a fresh prompt, then Ctrl+D exited 0.
  * The first, since-replaced harness reported a missing reply, although the history file showed the line was submitted. The harness's expected text left out the spinner frame. The rest of that failure was not isolated, and the rewritten harness has not reproduced it.
  * **Not done by the agent:** a hand-typed session on a Windows console. `make probe-conhost` drives the editor's test child, not `gobble`, so this check is left to the owner.
* **What the plan predicted wrongly.**
  1. **Kind names.** The update kinds are `KindAgentText`, `KindThought`, `KindToolCall`, `KindToolCallUpdate`, `KindPlan`, `KindUsage` and `KindOther`. Step 1's bare names collide with the `ToolCall` and `Usage` types.
  2. **`acpclient.Options.MaxQueued`** was added, so a test can see a drop with a queue of 1.
  3. **Print mode sends no `session/cancel` of its own.** The SDK's `ClientSideConnection.Prompt` sends one when its context is cancelled (`client_gen.go:291-297`).
  4. **Updates sent before the `session/new` answer were dropped,** because their session was not yet known. That is Phase 6's `available_commands_update` order. `NewSession` calls are now serialised, and such updates go to the session being created. `ScriptAgent.SessionUpdates` tests this.
  5. **The SDK cancels the prompt's context on `session/cancel`** (`agent_gen.go:291-300`). `ScriptAgent` therefore answers `cancelled` and not with an error, and Phase 3's agent must do the same.
  6. **A tool header is drawn again only when its rendered line changes.** `pending` and `in_progress` draw the same line, and printing it twice was noise.
  7. **The SDK's `SetLogger` raced with its reader.** The deviation of this date resolved it: the fork's `v0.13.6-mcr.2` adds `WithLogger`. Every gobble connection sets its logger at construction, and `SetLogger` is called nowhere.
  8. **The clock is a package variable, `clock`,** so the `stream-json` golden is fixed. Step 3 named only a `runEnv.now` field.
  9. **The line session's typed lines after the first are not composed:** `@path` is D4's rule for the first prompt only. Later lines go as plain text, as D4 states.

## Amendments

**2026-09-29 — native magic-cli-remote CLI.** The first draft of this
plan allowed Phase 6 to ship with an empty slash-command list, mapped
compact only as a `_gobble/` extension, and did not require `usage_update`,
`session/set_mode`, or `session/set_model`. The associated MADR now
requires honest executing slash commands and the ACP methods mcremote
already consumes. Phase 6 is that contract; a former empty-list
allowance is withdrawn. Phase 7 keeps `_gobble/` dual-path. Hardening moves
from Phase 7 to Phase 8. Companion `IDPi` work is documented and kept
out of this tree.

**2026-09-29 — `gobble` identity, SDK routing, wider v1 line.** The binary
is `gobble` and extension methods are `_gobble/…` ([0003-MADR](0003-MADR-gobble-product-identity.md)). Phase 0 is
superseded by [0004-PLAN](0004-PLAN-go-module-architecture.md). Phase 4 step 4 (`session/set_model`) is withdrawn:
`coder/acp-go-sdk` v0.13.5 answers MethodNotFound for non-schema methods
that do not start with `_`, so model and thinking go through
`session/set_config_option`. The Phase 8 and Verification Charm checks
become archtest rule 5. The Phase 2 grep becomes archtest rule 6 over
`internal/cli` and `acpclient`. Phase 4 writes Pi v3 entry shapes from
the start. Features the widened v1 line adds ([0005-MADR](0005-MADR-v1-feature-scope.md)) are
delivered by [0005-PLAN](0005-PLAN-v1-feature-scope.md) after this plan, not by new phases here.

**2026-09-30 — shared libraries as they exist.** D10 and D14 are no
longer open at phase time. Phase 3 imports `go-llmprovider-sdk` at a
commit pin and uses `providers.New` + `Stream`. Phase 5 pins the
official MCP go-sdk version. Tests still use `llm/llmtest` so ACP chunk
assertions do not depend on the SDK's `NativeStreaming: Unsupported`.

**2026-10-01 — native agent of magic-cli-remote.** Follows the fourth
amendment of the associated MADR. Companion checklist expands from a
Compact hook to Spec-parameterized ops, `ConfigureSession` for model,
`session_info_update` → title, and live probes that include
MethodNotFound for `session/set_model`. No new phases in this tree.

**2026-10-01 — cross-repository assessment (MADR fifth amendment).** No new phases. Changes to existing phases:

* **Phase 0.** Superseded by 0004-PLAN, as before. The ACP `replace` stays. The agent side keeps the SDK's default overflow policy. `acpclient` sets `OverflowDropNewest` with a drop handler (0004-MADR amendment of this date).
* **Phase 1.**
  * `Initialize` always sends `agentInfo {name:"gobble", title:"gobble", version}`.
  * `promptCapabilities.image` stays `false` until go-llmprovider-sdk carries image input (0004-MADR amendment of this date). `audio` is `false`.
* **Phase 3.**
  * The facade's `Usage` event is produced by the adapter. Until the SDK decodes usage (its 0015-PLAN S9), the adapter estimates tokens and marks the event `estimated`.
  * The Phase 3 `usage_update` accept still holds. `size` comes from gobble's catalog.
  * Step 1's pin follows 0004-MADR's amendment: develop against a recorded commit, and ship no tag on a pseudo-version.
* **Phase 4.**
  * Advertise `sessionCapabilities.list` and `close`.
  * Persist `model_change` and `thinking_level_change` entries, and restore model and thinking on `session/load`. Accept: a load in a fresh process continues with the saved model, with no `set_config_option` call.
* **Phase 6.**
  * `NewSession` and `LoadSession` send `available_commands_update`, `current_mode_update` and a first `usage_update` **before** returning. Accept: a frame-order test, shown failing on a copy that sends the update after the response.
  * The advertised set includes `/deep-research` once the skill exists (0005 X1).
  * Unknown `_` methods return -32601.
* **Verification.** The stdio script follows the corrected list in the MADR's fifth amendment (Confirmation changes).
* **Companion work.** The checklist above is replaced by magic-cli-remote `docs/decisions/0179-MADR-gobble-native-acp-provider.md` D1–D10. This plan still does not mutate that repository.

**2026-10-04 — configuration surface undecided.** The config surface is undecided and will be either a native Kong facility or a surface we write. Config implementation stays unspecified.

**2026-10-04 — native CLI mode and magic-cli-remote integration (0008-MADR).**

* **Phase 2, step 2.** `gobble prompt "…"` is replaced by `gobble -p "…"` on the default `chat` command (0008-MADR D1). This phase also wires:
  * the line session (D5);
  * the renderer of `internal/cli/render` (D7);
  * the print formats `text`, `json` and `stream-json` (D9);
  * Ctrl+C during a turn as `session/cancel`.
* **Phase 1.** `initialize` follows 0008-MADR D19 item 1:
  * `agentInfo.version` is parseable semver on every build;
  * capabilities are as listed there;
  * no network call before `initialize` returns (item 6).
* **Phase 3.** Agent output follows D19 items 2 and 3:
  * snapshot-shaped updates;
  * chunks coalesced at 50 ms or 4 KiB;
  * terminal tool statuses sent alone;
  * tool titles of at most 60 runes, with a self-contained summary of at most 400 bytes first.
* **Phase 4.** Sessions follow D19 items 5 and 8:
  * compact replay on `session/load`;
  * an exclusive per-session writer lock (`flock` / `LockFileEx`), with a second writer failing with exit 1 and the owning pid;
  * readers take no lock.
* **Phase 6.**
  * The line session's slash registry is the agent's `available_commands_update` united with `/help`, `/exit`, `/quit`, `/edit` and `/new`. An unknown name is a local error (D13).
  * Advertised names match `[A-Za-z0-9][A-Za-z0-9_-]*` (D19 item 7).
  * `model` and `thought_level` are ungrouped select config options with a category (D19 item 4).
  * The permission prompt is D16's one-line prompt.

**2026-10-05 — Phase 1 made executable; capabilities stay honest per phase (owner's decision).**
The 0008 amendment above says Phase 1's `initialize` follows 0008-MADR D19 item 1. That item lists the end state: `loadSession`, `sessionCapabilities.list`, MCP `http` and `embeddedContext`, all true. Step 1 of this phase says to advertise only what is implemented. The owner decided on 2026-10-05 that step 1 governs. Each phase advertises what it implements, and D19's list is reached as Phases 3–5 implement each item. The phase that turns a capability on carries a test of it. The 0008 amendment's other Phase 1 lines stand: the SemVer `agentInfo.version`, and no network before `initialize` returns.

* **Step 1, `acpserver/agent.go`.** `type Agent struct` implements `acp.Agent`. `New(opts Options) *Agent` takes `Options{Version string; NewID func() string}`; `NewID` defaults to the standard library's `uuid.NewV7().String()`.
  * `Initialize` returns:
    * protocol `acp.ProtocolVersionNumber` (1);
    * `agentInfo {name: "gobble", title: "gobble", version}`;
    * `authMethods: []`;
    * of the capabilities, only `sessionCapabilities.close`, because `CloseSession` works in this phase. `loadSession`, `list`, `resume`, MCP, image, audio and `embeddedContext` are false or absent.

    It makes no network call.
  * Methods this phase does not implement (`authenticate`, `logout`, `session/list`, `session/resume`, `session/set_mode`, `session/set_config_option`) answer -32601, matching what is advertised.
* **Step 2.**
  * `NewSession` records the session's `cwd` and returns a new id.
  * `Prompt` on a known session sends one `session/update` `agent_message_chunk` holding the prompt's text blocks joined by blank lines, then returns `end_turn`. An unknown session is invalid params (-32602).
  * `Cancel` succeeds. `CloseSession` forgets the session; an unknown one is invalid params.
* **Step 3, `acpserver/serve.go` and `internal/cli/acp.go`.**
  * `Serve(ctx, in, out, opts) error` runs the agent over `in` and `out` and sets the connection's logger.
  * It returns `nil` when `in` reaches end of file, and `context.Cause(ctx)` when the context is done.
  * `ACPCmd.Run` calls `Serve` with the process's stdin and stdout and the lazy logger. The logger has no stderr handler under `acp` (0004-PLAN Phase 2 step 4).
  * `gobble acp` therefore exits 0 at stdin EOF, and 130 or 143 on a signal. Archtest rule 2 allows `acpserver`'s SDK import; `internal/cli` imports `acpserver`, never the SDK.
* **Step 4, the tests.**
  * `acpserver/agent_test.go` runs Initialize → NewSession → Prompt → CloseSession through `acptest.Pair`, with fixed ids, against `testdata/session.golden`.
  * A capability test asserts the advertised set exactly, so a later phase's change is deliberate.
  * A version test checks that `agentInfo.version` parses as SemVer.
  * `acpserver/binary_test.go` builds `cmd/gobble` and drives `gobble acp` with the SDK's client over the child's stdin and stdout. It checks that every stdout line is a JSON-RPC message and stderr is empty, and that closing stdin ends the process with 0 within 2 s (0008-MADR D19 item 6).
  * `internal/cli`'s P4 tests change: `gobble acp` with an empty stdin now exits 0 and still writes nothing to stdout.
* **Accept**, as written: the session test is shown failing on a scratch copy whose `Initialize` returns protocol 2, and the failure is quoted in the handoff. Further negatives:
  * a capability advertised early fails the capability test;
  * a banner on stdout fails the binary test;
  * `Serve` that ignores EOF fails the 2 s bound.
* **Verification.**
  * `go test` for `acpserver` and `internal/cli`;
  * `golangci-lint` for linux, darwin and windows;
  * `make preflight` on Windows and in WSL.

  The agent stages; the owner commits.

**2026-10-05 — Phase 2 made executable (owner's decisions on diff content and the failed-turn result).**
Phase 2 wires 0008-MADR D1, D5, D7 and D9 to `acpclient` (0008-MADR D16), and delivers the flags 0008-PLAN P4 assigned to it: `--output-format`, `--show-thinking`, `--stats`, `--verbose` and `--cwd`. The agent is still Phase 1's echo agent, so the renderer for tool calls, thoughts, plans and usage is tested against a scripted agent. The owner decided two questions on 2026-10-05:

* **Diff content waits for Phase 3.** ACP diff content is `{path, oldText, newText}`, and a coloured unified diff needs a line-diff algorithm that gobble does not have. No agent sends a diff until Phase 3's `edit` tool. Phase 3 gains step 6 below. Until then, a diff block prints `diff <path> (+N -M lines)`, never nothing.
* **A failed turn still writes a result.** When `session/prompt` returns a JSON-RPC error, `json` and `stream-json` still write the result object, with `stopReason` `"error"` and an added `error` string, and the process exits 1. 0005-MADR's "Print mode" row gains a dated note of this field.

Choices made within the wording, recorded so a later phase does not re-open them:

* **`internal/cli` sees no SDK type.** Rule 2 keeps the SDK in `acpclient`, and a type alias would pass archtest while leaking the SDK's churn into `internal/cli`. So `acpclient` translates updates into its own types, and keeps each update's params object for `stream-json`.
* **The agent runs in process,** over two `io.Pipe`s, through a `ServeFunc`. `internal/cli` passes a closure over `acpserver.Serve`, and `acpclient` imports neither `acpserver` nor `agent`.
* **Ctrl+C during a turn is a signal.** The console leaves raw mode for the turn, because raw mode on Unix turns off output processing, so a bare `\n` would not return the carriage. The editor's raw mode is restored before the next prompt. The first interrupt during a turn goes to the turn and sends `session/cancel`; a second, before the turn ends, cancels the process with exit 130. TERM and HUP always end the process.
* **`session/new` is sent with the first prompt,** not at start-up. D5 forbids a network call before the first prompt, and Phase 5 connects MCP servers on `session/new` (D19 item 6). `initialize` runs at start: it is in process and makes no network call.
* **Exit codes for stop reasons (D10).** `end_turn`, `max_tokens`, `max_turn_requests` and `refusal` exit 0 and are reported in `stopReason`. `cancelled` that gobble did not request is the "provider-side abort" of D10, and exits 1. A JSON-RPC error exits 1. A signal exits 128 plus the signal.
* **An image the agent does not accept** ends the run with exit 1 and `Error: the agent does not accept images (<path>)` before `session/prompt` is sent (D19 item 1: never dropped silently). The echo agent advertises `image: false`.
* **`--cwd DIR`** is made absolute and must be a directory (otherwise exit 2). It is the session's `cwd`, and `@path` and `--file` resolve against it. gobble never calls `os.Chdir`.
* **`--show-thinking` and `--verbose` act on the line session's display.** Print-mode `text` and `json` carry the final text only, and `stream-json` carries every update anyway. `--stats` adds time to first token, and tokens per second when the turn's `usage.outputTokens` is known. It applies in both modes: in print mode, the status line goes to stderr only when `--stats` is given.
* **Print `text`** is the turn's agent text, sanitised (D8), with one trailing newline. `json` is `{"text","stopReason","usage","cost","sessionId"}`: `usage` is the `session/prompt` response's `usage` object or `null`, and `cost` is the last `usage_update` cost or `null`. `stream-json`'s header is `{"type":"session","id","cwd","timestamp"}`, with the timestamp in RFC 3339 UTC; then one line per `session/update` params object, as the SDK encodes it; then `{"type":"result",…}` of the `json` shape.

Steps:

1. **`acpclient`** (`doc.go` rewritten; new `conn.go`, `update.go`, `conn_test.go`):
   * `type ServeFunc func(ctx context.Context, in io.Reader, out io.Writer) error`.
   * `Start(ctx, serve ServeFunc, opts Options) (*Conn, error)`:
     * runs `serve` over two pipes;
     * makes the SDK client connection with `OverflowDropNewest` and a drop handler that calls `Options.OnDrop` (0004-MADR, amendment of 2026-10-01);
     * sends `initialize` with protocol 1, `clientInfo {name, version}` and no file-system or terminal capability;
     * fails on any protocol other than 1.

     `Options{Name, Version string; Logger *slog.Logger; OnDrop func(method string, total uint64)}`.
   * `(*Conn).Agent() AgentInfo` gives the name, title, version, and the `Image` and `LoadSession` capabilities. `(*Conn).Close() error` closes the client's pipe and waits for `serve` to return.
   * `(*Conn).NewSession(ctx, cwd string, on func(Update)) (*Session, error)` sends `mcpServers: []`. `(*Session).ID()`, `Prompt(ctx, Prompt) (Result, error)`, `Cancel(ctx)` and `Close(ctx)` follow.
     * `Prompt{Text string; Images []Image{MIME string; Data []byte}}` becomes a text block (when non-empty) and base64 image blocks.
     * Images on an agent without `image` fail with `ErrImagesUnsupported` before anything is sent.
     * `Result{StopReason string; Usage *Usage}` carries every field of the response's `usage`.
   * `Update{Kind; Text; Block; Tool ToolCall; Plan []PlanEntry; Context ContextUsage; Params []byte}`. Its kinds are `AgentText`, `Thought`, `ToolCall`, `ToolCallUpdate`, `Plan`, `Usage` and `Other`.
     * `Block` names a non-text content type.
     * `ToolCall{ID, Title, Status string; Content []ToolContent}` has empty strings for fields an update left unchanged.
     * `ToolContent{Text string; Diff *Diff{Path string; OldText *string; NewText string}; Terminal string}`.
     * `Params` is the `session/update` params object, encoded by `encoding/json` as the SDK encodes it.
   * Requests from the agent: `session/request_permission` is answered cancelled until Phase 6's prompt. File-system and terminal requests are method-not-found, matching the capabilities sent.
   * Tests, through `acptest.ScriptAgent` (step 2):
     * every update kind's translation, and that `Params` keeps the kind;
     * a dropped notification reaches `OnDrop`, at queue size 1;
     * `ErrImagesUnsupported`;
     * the step-3 test of the original phase: `Session.Prompt(Prompt{Text: "hi"})` and the SDK's own `ClientSideConnection.Prompt` with `TextBlock("hi")` send canonically equal params, `{"prompt":[{"text":"hi","type":"text"}],"sessionId":…}`.
2. **`acpclient/acptest/script.go`** (new, with `script_test.go`):
   * `ScriptAgent{Turns []Turn; Image bool}` implements `acp.Agent`. `Serve(ctx, in, out) error` runs it with the same EOF rule as `acpserver.Serve`.
   * `Turn{Updates []string; StopReason string; Usage string; ErrCode int; ErrMessage string; WaitCancel bool}`. Each update is the JSON of one ACP `SessionUpdate`, so a caller needs no SDK import.
   * It records `Methods()`, `Prompts()` (each params object, canonical), `Cwds()` and `Cancels()`.
   * A prompt past the last turn is a -32603 error naming the script.
3. **`internal/cli`:**
   * **`agent.go`** (new): `var agentServe = func(e *runEnv) acpclient.ServeFunc`, a closure over `acpserver.Serve` with the build's version and `e.log()`. Tests replace it. `(*runEnv).startAgent(ctx)` calls `acpclient.Start` with `Name: "gobble"`, the version, the logger, and an `OnDrop` that prints `Warning: dropped N ACP notifications (<method>)`.
   * **`chat.go`:** `Run` resolves `--cwd`, composes the input against it, then calls `runPrint` or `runLine`. The "not available yet" stub goes.
   * **`input.go`:** `composeInput` takes a base directory for `@path`.
   * **`flags.go`:** the five Phase 2 flags leave `laterFlags`. `plan0002P2` goes once unused.
   * **`print.go`** (new): the three formats and the exit rules above. Results are written through `Output.Result`, so a broken pipe exits 0 (D3). On a signal, it sends `session/cancel` under a 2 s context detached from the cancelled one, then returns.
   * **`line.go`** (new): the D5 loop over `editor.New`.
     * It owns stdin through `editor.Reader`, puts the console in `editor.Raw` mode, and keeps `<state>/history` (created with `appdirs.EnsurePrivateDir`; a failure is one `Warning:` and no history).
     * Prompts are `> ` and `… `, or `... ` with ASCII glyphs. The first prompt is the composed input, when there is one.
     * Each turn: leave raw mode; register the interrupt; spinner on stderr (`working`) until the first update; the view; the status line; back to raw mode.
     * A turn's error prints `Error: …` and returns to the prompt.
     * `io.EOF` and `editor.ErrExit` end the session with 0, after `session/close`, `Conn.Close`, `Editor.Restore` and the raw-mode restore.
     * Raw mode and the interrupt hook are fields, so tests drive the loop over a pipe.
   * **`view.go`** (new): the D7 display of updates, on stdout.
     * Agent text is sanitised, passed through `render.Stream`, then `render.Tables` and `render.Style` when colour is on; it is raw when colour is off.
     * Thoughts are hidden, or dimmed with `--show-thinking`.
     * Tool calls print `render.ToolHeader`, and again whenever the status changes; text content goes through `render.ToolOutput` under `--verbose`. Diff content prints `diff <path> (+N -M lines)`, with the path under `render.ShortPath`. Terminal content prints `terminal <id>`.
     * Plans print through `render.Plan`. A non-text block prints `[<type>]`.
     * The stream is flushed before every non-text event and at the turn's end.
     * The status line on stderr is `render.StatusLine` from the last `usage_update`, dimmed, plus `render.Stats` under `--stats`, plus the stop reason when it is not `end_turn`.
   * **`signal.go`:** `watchSignals` takes an `*interrupts`. `(*interrupts).during(cancel func()) (release func())` lets a turn take the first `os.Interrupt`; any other signal, or an interrupt with no turn or a second one, cancels as before. `main.go` passes it to `runEnv`, beside a `now func() time.Time` for the timestamp.
4. **`internal/cli/render`:**
   * **`plan.go`** (new): `func Plan(entries []PlanItem, s term.Stream, g term.Glyphs) string`, with `PlanItem{Content, Status string}`. Completed is `g.Check` in green, in progress is `g.Arrow` in bold, pending is `g.Bullet` dimmed. Content is sanitised.
   * **`status.go`:** `func Stats(ttft time.Duration, outputTokens int64, gen time.Duration, g term.Glyphs) string` gives `ttft 0.4s`, then `52 tok/s` when the tokens and the time are both known.
5. **Tests:**
   * `print_test.go`:
     * `text`, `json` and `stream-json` against goldens, with a fixed clock;
     * the method order `initialize`, `session/new`, `session/prompt`, `session/close`;
     * the prompt params equal to step 1's canonical JSON;
     * a JSON-RPC error giving exit 1 and the error result in both JSON formats;
     * an unrequested `cancelled` giving exit 1;
     * an image refused with exit 1;
     * `--cwd` as the session's cwd and as the `@path` base, and a bad `--cwd` giving exit 2;
     * a cancelled context sending `session/cancel`.
   * `line_test.go`, over a pipe with raw mode stubbed:
     * a prompt, the echo on stdout, and Ctrl+D exiting 0 after `session/close`;
     * `session/new` not sent before the first prompt;
     * an interrupt during a `WaitCancel` turn sending one `session/cancel` and returning to the prompt.
   * `view_test.go` goldens, colour on and off: Markdown with a table, thoughts hidden and shown, a tool call with 30 lines of output with and without `--verbose`, diff and terminal content, a plan, an image block, and the status line with `--stats`.
   * `signal_test.go`: one interrupt during a turn calls the hook and leaves the context alive; a second cancels with exit 130; TERM during a turn cancels at once.
   * `main_test.go`: the stub rows become echo-agent rows (`-p x` prints `x`, exit 0), and the later-flag table loses the five flags.
   * `plan_test.go` and the status tests cover step 4.
   * `TestPipedInputIsComposed` checks the composed prompt in the agent's received params instead of the stub's exit.

**Accept**, as the phase states it:
* `gobble -p` calls ACP and never `agent`, which is archtest rule 6.
* The same `session/prompt` params are sent from Kong and from a stdio SDK client (step 1's canonical-JSON test, and `print_test.go`'s).

Each new check is shown failing on a scratch copy:
* `Prompt` sending the text twice fails the params test;
* `session/new` sent at start fails the line test;
* an interrupt hook that is never called fails the cancel test;
* the result object dropped on error fails the error test;
* a `stream-json` header missing `cwd` fails its golden.

Each failure is quoted in the handoff.

**Verification.**
* `go test ./acpclient/... ./internal/cli/...`;
* `golangci-lint` for linux, darwin and windows;
* `make preflight` on Windows and in WSL, and `go test -race ./...` in WSL;
* a manual line session on a Windows console and in WSL: a prompt is echoed, and Ctrl+D exits 0. The echo agent answers at once, so a turn cannot be interrupted by hand; the cancel path is covered by `line_test.go` only.

The agent stages; the owner commits.

* **Phase 3, new step 6 (owner's decision of this date).** The edit tool's ACP diff content is drawn by a Myers line diff in `internal/cli/render` (`render.UnifiedDiff(path, old, new string) string`, fed to `render.Diff`), tested on the edit tool's real output. It replaces Phase 2's `diff <path> (+N -M lines)` line.
* **Deferred from Phase 2, named:**
  * slash and `@path` completion in the line editor wait for Phase 6, where D13's registry supplies the slash names;
  * the "no usable credential" warning of D1 waits for the first provider (Phase 3 and 0005-PLAN F4);
  * `/new`, `/help` and `/edit` wait for Phase 6 (D13). Until then they reach the agent as prompt text, which the echo agent repeats.

**Deviation, 2026-10-05 — the SDK's `SetLogger` races with its own reader (owner's decision: fix the fork first).**

* **Found:**
  * Phase 2's first WSL `go test -race ./...` stopped on a data race in two `TestMainExitCodes` rows.
  * The fork's `SetLogger` is a plain field write (`connection.go:236`). `loggerOrDefault` reads that field (`connection.go:239`) on the receive goroutine, reached from `shutdownReceive` at end of input (`connection.go:609`).
  * `NewConnection` starts that goroutine (`connection.go:212`) before the caller can call `SetLogger`, and the fork has no construction-time logger option.
* **It is pre-existing:**
  * A scratch clone of the committed Phase 1 tree (`13253a5`, clean) showed 1 race in 30 runs of `go test -race -count=30 -run 'TestMainExitCodes/acp|TestACPWritesNothingToStdout' ./internal/cli/`.
  * The site is `acpserver/serve.go:22–25`. Phase 1's single `-race` run did not hit it.
  * Phase 2's `acpclient.Start` and `acptest.ScriptAgent.Serve` copied the same pattern.
* **Decision (owner, 2026-10-05).**
  * Fix the root cause in the fork: a `WithLogger` connection option applied before the goroutines start, and an atomic `SetLogger`, released as `v0.13.6-mcr.2`.
  * Then gobble bumps its `replace` and sets the logger through the option at all three sites.
  * The gobble-local alternative was rejected because it would depend on the fork's internals: gating the SDK's first read until `SetLogger` returns.
  * Phase 2 waits until the tag exists.
* **Added to this phase's scope:**
  * `acpserver/serve.go`, Phase 1's file;
  * `go.mod` and `go.sum`, for the `replace`;
  * a dated amendment of the fork's facts in `0004-MADR-go-module-architecture.md`.
* **Verification added:**
  * The race regression is shown failing first: `go test -race -count=30` of the `acp` and print rows on a scratch copy that still calls `SetLogger`.
  * It then passes on the tree that uses the option.
* **Report:** `0009-REPORT-magic-cli-remote-findings.md` M8, because magic-cli-remote pins the same fork.
