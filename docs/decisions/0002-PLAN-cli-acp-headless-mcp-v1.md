---
status: in-progress
date: 2026-10-06
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

*(2026-10-05: steps 1–6 are expanded into executable steps in the Amendments entry "Phase 3 made executable" of this date. The owner's decisions there replace step 1's commit pin with a tag, and add ambient credential selection and ACP approval of mutating tools.)*

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

*(2026-10-06: expanded into two executable parts in the Amendments entry "Phase 4 made executable" of this date. "A corrupt JSONL file fails closed" is read as 0005-MADR "Sessions (changed)" amends it: an unreadable header fails closed, and a malformed line is skipped with a diagnostic.)*

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

*(2026-10-06: made executable in the Amendments entry "Phase 5 made executable" of this date. Step 1's version is `github.com/modelcontextprotocol/go-sdk v1.8.0`, the newest release on the module proxy that day. Step 6: the go-sdk supports OAuth, but the owner decided on 2026-10-06 that MCP OAuth lands with 0005-PLAN F8, which owns it with the token store. This is the gap step 6 asks to record: in Phase 5, an HTTP server that answers 401 shows as needing sign-in.)*

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

*(2026-10-06: made executable, in two parts, in the Amendments entry "Phase 6 made executable" of this date. Step 6's frozen list is there: `help`, `compact`, `usage`, `context`, `session`, `model`, `thinking`, `mode`, `plan`, `name`, `fork` and `clone`.)*

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

**Phase 3, 2026-10-05 — complete (staged; the owner commits).** It ran as expanded in the amendment "Phase 3 made executable", with the owner's three decisions: ambient credentials, ACP approval, and a tagged SDK. Phases 4–8 have not run.

* **Pins.**
  * `go list -m` prints `github.com/maccavelli/go-llmprovider-sdk v1.1.1`, the owner's tag of `e9083c3`, and `github.com/google/jsonschema-go v0.4.3`.
  * No pseudo-version is required.
* **Files.**
  * New:
    * `agent`: `agent.go`, `event.go` and `agent_test.go`;
    * `llm/provider`: `provider.go`, `ambient.go` and `provider_test.go`;
    * `tool/builtin`: `builtin.go`, `files.go`, `bash.go` and `builtin_test.go`;
    * `tool/tool_test.go`;
    * `acpserver`: `prompt.go`, `coalesce.go`, `prompt_test.go` and `coalesce_test.go`;
    * `acpclient/handler_test.go`;
    * `internal/cli`: `render/udiff.go`, `render/udiff_test.go` with four goldens, and `testmain_test.go`.
  * Changed:
    * `llm/llm.go`, `tool/tool.go` and `permission/permission.go`;
    * the package docs of `llm`, `llm/provider`, `agent`, `tool`, `tool/builtin` and `acpserver`;
    * `acpserver/agent.go`, `acpclient/conn.go`, `acpclient/acptest/client.go` and `pair.go`;
    * `internal/cli/agent.go`, `acp.go`, `chat.go`, `flags.go`, `line.go`, `main.go`, `print.go` and `view.go`;
    * their tests and goldens, and `go.mod` and `go.sum`.
* **Accept.** Each item is through `acpserver` and `acptest.Connect`.
  * `TestReadReachesTheReply`: a `read` of a temp file gives `tool_call` (`read notes.md`, pending, kind read), `in_progress`, then `completed` with the summary `read notes.md (1 lines)` first. The test provider's reply holds `the answer is 42`.
  * `TestCancelBlockedBash`: `sleep 30`, cancelled, ends `cancelled` within 5 s.
  * `TestUsageUpdateAfterEveryPrompt`: every prompt ends with `usage_update`, `used` > 0 and `size` 0.
  * `TestWriteAsksPermission`: a refused write leaves no file and fails the call; an allowed one writes it, with the diff after the summary.
  * `TestCoalesceByTime` and `TestCoalesceBySize` (`testing/synctest`): 200 deltas inside 10 ms give one chunk, and 10 KiB gives three chunks of at most 4 KiB.
  * `TestPhoneLegibleBounds`: titles of at most 60 runes, first blocks of at most 400 bytes.
* **Gates.**
  * `make preflight` printed `preflight passed` on Windows and in WSL.
  * Windows `go test ./...`: 19 packages `ok`, no `FAIL`. WSL `go test -race ./...` exited 0 with 0 race reports.
  * `golangci-lint` printed `0 issues.` for linux, darwin and windows.
  * The pre-add check was clean for all 48 Go files.
* **A real binary.** In WSL, a built `gobble` with every credential variable removed exits 3 from `-p hi`, with `Error: no model credential: set one of GEMINI_API_KEY, OPENAI_API_KEY, …`.
  * A pty line session prints the warning at start and the same error on a prompt, returns to the prompt, and exits 0 on Ctrl+D.
  * `TestGobbleACPBinary` shows `gobble acp` answering a prompt with -32000 and no network call.
* **Negative tests,** each on a scratch copy, each failing as expected:
  * no `usage_update`: `prompt_test.go:156: prompt 1: last update {… AgentMessageChunk:0x… …} is not usage_update`;
  * no permission request before `write`: `prompt_test.go:178: permission requests []`;
  * the coalescer forwarding every delta: `coalesce_test.go:39: sent 200 chunks before the wait`;
  * titles not cut, in both `acpserver` and `tool/builtin`: `prompt_test.go:253: title is 317 runes`;
  * `Cancel` not cancelling: `prompt_test.go:234: the turn did not end within 5 s of session/cancel`;
  * `UnifiedDiff` with no context: `udiff_test.go:34: udiff-two-hunks differs from testdata\udiff-two-hunks.golden`.
* **What the plan predicted wrongly.**
  1. **The tools are in three files.** `read`, `write` and `edit` are in `files.go`, and the shared bounds and title helpers in `builtin.go`. Step 5 named a file per tool.
  2. **`tool` needed more than `Describer`.**
     * `WithDescribe` lets a tool built with `New` title its calls.
     * `TextResult`, `ErrorResult` and `Result.Text` give a result the model reads as text.
     * An `Out` of type `Result` is passed through, so a tool sets its `Summary` and `Diffs`.
  3. **The agent has a `ToolRun` event,** sent when an approved call starts, so the client sees `pending`, then `in_progress`. `Run` takes the `tool.Env`.
  4. **The SDK cancels the prompt's context itself on `session/cancel`.** With gobble's own `Cancel` removed, `TestCancelBlockedBash` still passed. So `Cancel` is defensive, and the negative removes both paths.
  5. **Phase 2's tests used -32000 for a generic failure.** It is ACP's `auth_required`, so three tests moved to -32603, and new tests cover -32000 as exit 3.
  6. **`UnifiedDiff` names an absolute or `~` path as it is.** Step 11's `a/` and `b/` prefixes made `a/~/src/a.go`; they stay for relative paths.
  7. **Ambient selection.**
     * `provider.Ambient` returns the `Selection` with the provider.
     * `Choose` makes the choice without building it, for the line session's warning.
     * `internal/cli` clears the credential variables once, in a `TestMain`, not test by test.
  8. **Smaller additions.**
     * `acptest.Client` gains `Allow` and `Permissions()`.
     * `acptest.Connect` sets a discarding logger, so its connections no longer print `connection closed` through the SDK's default logger in every test. That noise predates this phase.
  9. **On Windows, the `bash` tool prefers Git for Windows' `bash.exe`, found beside `git`.** `System32\bash.exe` is the WSL launcher, and would run the command in another system.
  10. **A line-session turn refused with `auth_required` prints the variables,** as print mode does, not the raw JSON-RPC error.
* **Found, not this phase's:** `render.Style` does not style emphasis.
  * 0008-MADR D7 requires it, but 0008-PLAN's step for `Style` (`0008-PLAN-native-cli-mode.md:590`) left it out. The file, committed in `4e25c8a`, is untouched here.
  * The owner decided on 2026-10-05 to fix it as a dated 0008-PLAN follow-up, after this commit, as its own change.

**Phase 4 Part A, 2026-10-06 — complete (staged; the owner commits).** It ran as expanded in the amendment "Phase 4 made executable", with the owner's three decisions of that date. Part B (the CLI flags and `--session` completion) has not run.

* **Files.**
  * `session`: `session.go`, `doc.go`, `memory.go` and `session_test.go`.
  * `session/jsonl`: `store.go`, `lock.go`, `lock_unix.go`, `lock_windows.go`, `lock_other.go`, `doc.go` and `store_test.go`.
  * `agent`: `agent.go` and `event.go`.
  * `acpserver`: `agent.go`, `history.go`, `config.go`, `serve.go`, `agent_test.go`, `session_test.go` and `testdata/session.golden`.
  * `llm/provider/ambient.go`; `internal/cli/agent.go` and `main.go`.
  * `go.mod`: `golang.org/x/sys` becomes a direct requirement, which 0008-MADR D19 item 8 names.
* **Accept**, each through `acptest.Connect` over a real `jsonl` store in a temporary directory:
  * `TestLoadReplaysInANewAgent`: the replay is `user:read the notes`, `tool:read notes.md:completed`, `agent:it says forty-two`, then `usage`. The next request ends user, assistant, tool, assistant, user.
  * `TestResumeWithoutReplay`: no updates, and the history is carried.
  * `TestListSessions`: the `cwd` filter, the first prompt as title, a `_meta` name as title (`named\nsession` stored as `named session`), and a live session without a file.
  * `TestCorruptSessions`: a malformed line is skipped. A bad header gives -32602, `not a valid session`.
  * `TestConfigOptionsPersist`: the next request has model `m2` and thinking `high`, and a new agent restores both with no `set_config_option` call.
  * `TestSetModelIsMethodNotFound`: a raw `session/set_model` gets -32601.
  * `TestSecondWriterIsRefused`: -32600, `session <id> is open in another gobble process (pid N)`.
  * `TestResultWrittenBeforeItsUpdate`: the result line is on disk when the terminal update is written.
  * `TestNoFileWithoutConversation`: no file after `session/new`, nor after a prompt refused for want of a credential.
  * `TestNewSessionMeta`: a chosen id, a name, and a fork whose `parentSession` is the source's absolute path. A duplicate id, an invalid id and an unknown source each give -32602.
  * The store's own tests cover Pi's layout and file name, the lock with its pid, torn and malformed lines, fail-closed headers and versions, flat stores, and an unwritten session leaving nothing.
* **Gates.**
  * `make preflight` printed `preflight passed` on Windows and in WSL. WSL `go test -race ./...` exited 0 with 0 race reports, so the `flock` path is exercised there.
  * Windows `go test ./...` showed no `FAIL`.
  * `golangci-lint` printed `0 issues.` for linux, darwin and windows.
  * The pre-add check was clean for all 22 Go files.
* **Negative tests,** each on a scratch copy, each failing as expected:
  * replay without the tool call: `session_test.go:106: replay user:read the notes,agent:it says forty-two,usage`;
  * the writer lock not taken: `session_test.go:313: second writer: <nil>; want invalid request with "session … is open in another gobble process (pid …)"`;
  * the tool result written after its update: `terminal update written true, result on disk then false`;
  * the file created before any message: `store_test.go:64: files before any message: [--srv-work-proj--/2024-12-03T14-00-00-000Z_….jsonl …]`;
  * the model change not applied: `session_test.go:238: request model "" thinking "high", want m2 and high`.
* **What the plan predicted wrongly.**
  1. **The first ordering test could not fail.** It checked the file from the client's update handler, but notifications are asynchronous, so the write usually landed first. With the update sent before the write, the test still passed. It now checks the file in a writer between the agent and its pipe, when the agent writes the frame, and fails as quoted above.
  2. **The "file created at `session/new`" negative targets the store's lazy guard** (`TestLazyCreationAndLayout`): an agent-level `session/new` appends nothing, so it cannot show the mutation.
  3. **Open logs must be closed when a connection ends.** Locks and files would otherwise stay held, and on Windows an open file cannot be removed. `Agent.Close` closes them, and `Serve` calls it.
  4. **Files beyond Part A's list.**
     * `llm/provider` gains `Models(id)`, for the model option's list.
     * `internal/cli/main.go` gains a `store` field on `runEnv`.
     * The jsonl store gains `PathOf`, so a fork records the source's absolute path as Pi does.
     * The platform-independent lock code is in `lock.go`, and `lock_other.go` is a no-op for the GOOS values gobble does not ship.
  5. **`session.Log.Entries` returns a slice, not Phase 1's iterator,** because a writer always holds its entries in memory. `Summarize`, `SortSummaries` and `HasConversation` are exported, so both stores list alike.
  6. **Every entry is written with `parentId`, `null` at a root,** as Pi writes it, including an entry read without one.

**Phase 4 Part B, 2026-10-06 — complete (staged; the owner commits). Phase 4 is complete.** It ran as the amendment wrote it, with the three deviations decided before it began (the message count through `_meta`, `-n` for new sessions only, and `--tools` completion). Phases 5–8 have not run.

* **Files.**
  * `acpclient`: `conn.go`, and new `sessions.go` and `sessions_test.go`.
  * From deviation 1: `acpserver/agent.go`, `config.go` and `session_test.go`.
  * `internal/cli`: `chat.go`, `completion.go`, `flags.go`, `line.go` and `print.go`, and new `sessions.go` and `sessions_test.go`.
* **Accept**, through `Main` against gobble's real agent with a scripted model, each over its own `GOBBLE_HOME`:
  * `TestContinueResumesTheNewest`: `-c` sends `second,B1,third`. `TestContinueWithNoneStartsNew`: no session means a new one, as Pi's `continueRecent` does.
  * `TestSessionValues`: an id prefix (`beta`), a title prefix in any case (`AVOC`) and a path each resume.
    * `alpha` gives `matches 2 sessions`, and `zzz` gives `no session matches`.
    * A file outside the store gives `session stranger is not in the session store`. Each of these exits 2.
  * `TestSessionIDAndName`: the file `…_my.id.jsonl` holds the name. A taken id exits 2. `-n` with `-c` exits 2 with the Phase 7 message.
  * `TestFork`: the request carries `original,one,forked`, and `parentSession` ends `_src.jsonl`. The source is unchanged.
  * `TestNoSessionAndSessionDir`: `--no-session` writes nothing, and `--session-dir` writes one file in the directory given.
  * `TestLockedSessionExits1`: exit 1, `Error: session held is open in another gobble process (pid N)`.
  * `TestLineSessionResumedLine`: `resumed earlier question (2 messages)`, then the history reaches the model.
  * `TestSessionAndToolCompletion`: `newer`, then `older` with `Old one · <date> · <cwd>`, then `:36`, with nothing on stderr and the store unchanged. `--tools` lists `read`, `write`, `edit` and `bash`.
  * `acpclient`'s unit tests cover `_meta` building and the mapping of the agent's refusals.
* **Gates.**
  * `make preflight` printed `preflight passed` on Windows and in WSL. WSL `go test -race ./...` exited 0 with 0 race reports.
  * Windows `go test ./...` showed no `FAIL`.
  * `golangci-lint` printed `0 issues.` for linux, darwin and windows.
  * The pre-add check was clean for all 13 Go files.
* **Negative tests,** each on a scratch copy, each failing as expected:
  * `-c` picking the oldest: `sessions_test.go:88: the continued request carries "first,A1,third", want the newest session's history`;
  * a prefix matching two sessions accepted: `sessions_test.go:150: --session alpha: exit 1, … want 2 and "--session alpha matches 2 sessions: "`;
  * `--no-session` writing a file: `sessions_test.go:208: --no-session wrote [….jsonl]`.
* **What the plan predicted wrongly.**
  1. **The `--session` directive is `:36`, not the `:32` of 0008-MADR's Confirmation.** The engine adds no-file completion to every value flag, as 0008-PLAN's record found for enum sources.
  2. **Completion offers ids.** A name shows in the description, and a name prefix is resolved by `--session` itself, not offered as a candidate.
  3. **The `--no-session` test first looked one directory too shallow.** The negative passed that assertion, and only a later one caught the file; it now looks in `sessions/` and fails at its own line.
  4. **The ambiguous-prefix negative exits 1, not 0.** The mutated code accepts the prefix, then the scripted model has no reply left. The test asserts exit 2, so it fails either way.
  5. **New files.** The session code is in new `acpclient/sessions.go` and `internal/cli/sessions.go`, not in `conn.go` and `chat.go`.
  6. **How the line session opens a session.** It resumes at start, which reads the file and makes no network call. A new session is still created with the first prompt. The resumed line is written to stdout, dimmed.
  7. **A refused `session/new` (a taken or invalid id, an unknown fork source) exits 2,** because it comes from a flag's value.
  8. **One `gobble` constant in `acpserver`** names the agent and the `_meta` key, because lint counted three uses of the string.

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

**2026-10-05 — Phase 3 made executable (owner's decisions: ambient credentials, ACP approval, a tagged SDK).**
Phase 3 replaces the echo agent with the turn loop. Read-only investigation on this date found that the contracts the loop needs are Phase 1 stubs:
* `llm.Content` has no field for a tool call's id, name or arguments, nor for a tool result;
* `tool.New` neither derives a schema nor runs its function;
* `tool.Env` is empty;
* `agent` and `tool/builtin` declare nothing.

The owner decided three questions on 2026-10-05:

* **Provider selection: ambient environment, in a fixed order.**
  * gobble reads the SDK's standard credential variables itself. They come from `llmprovider/providers.Default().Descriptors()`: `GEMINI_API_KEY`, `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, `XAI_API_KEY`, `OPENCODE_API_KEY`, `HF_TOKEN`, `KILO_API_KEY` and `TOGETHER_API_KEY` on `v1.1.0`.
  * The first descriptor, in registry order, that needs a key and has its variable set is used, with its first curated model (`StaticModels[0]`). `ollama` needs no key and is never chosen this way.
  * There is no new `GOBBLE_` variable and no config file. 0005-PLAN F4 later puts `--provider`, `--model`, `--api-key` and the store in front of it.
  * With no credential, `initialize` and `session/new` still succeed. A prompt fails with ACP `auth_required` (-32000), and print mode exits 3 (0008-MADR D10).
* **Mutating tools ask through ACP, and the CLI declines.**
  * Before a tool whose spec is not read-only (`write`, `edit`, `bash`), the agent sends `session/request_permission` with `allow_once` and `reject_once` options.
  * An ACP client answers it. gobble's own CLI answers `reject_once` until Phase 6's prompt, so through the CLI the model can only read. It is told the user declined.
* **The SDK is pinned at a tag the owner cuts from `origin/main`** (`e9083c3` on this date, 10 commits after `v1.1.0`). Phase 3's code waits for that tag. The tag replaces the commit pin of step 1, so no pseudo-version is ever required.

Choices made within the wording, recorded so a later phase does not re-open them:

* **The facade grows, without breaking anything.**
  * `llm.Content` gains `ToolCallID`, `ToolName`, `ToolArgs` (JSON text) and `IsError`.
  * The content types are `text`, `image`, `thinking`, `tool_call` (an assistant message) and `tool_result` (a `tool` message). Phase 1's unused `"tool"` is dropped from the comment.
  * `tool.Env` gains `Cwd`.
  * `tool.Result` gains `Title`, `Summary` and `Diffs []tool.Diff{Path; OldText *string; NewText}`. A tool may implement `tool.Describer` (`Describe(call Call, env Env) string`) to title a call before it runs.
  * `permission.Rule` gains `CallID`, `Title` and `Input`.
  * Each change only adds; apidiff stays clean.
* **Package roles follow 0004-MADR.**
  * `agent` runs the loop over `llm.Provider`, `[]tool.Tool` and a `permission.Policy`, and yields sealed events. It imports no ACP, SDK or Kong package (rule 1).
  * `acpserver` turns the events into ACP updates, and implements the `Policy` with `session/request_permission`.
  * `llm/provider` is the only importer of the SDK (rule 4).
  * `internal/cli` passes `acpserver` a provider factory, and never imports `agent` (rule 6).
* **Construction is lazy.** `acpserver.Options.Provider` is a `func() (llm.Provider, error)`, called on a session's first prompt. So `initialize` stays free of credentials and network (D19 item 6), and so does start-up (D5).
* **Adapter options:**
  * `WithAPIKey`, `WithModel`, `WithClientInfo("gobble", <SemVer>)`, `WithModelProbes(false)` (billed probes stay off, 0005-MADR) and `WithoutModelMetadata()`, so construction never reaches the network.
  * The result is wrapped in `WithRetry(RetryPolicy{MaxAttempts: 4, BaseDelay: 2 s, MaxDelay: 60 s})` (0004-MADR).
* **Tools are Phase 3's minimum.** 0005-PLAN F1 brings parity: offsets, fuzzy edit, process groups, `os.Root`, `grep`/`find`/`ls`.
  * `read {path}` returns at most 2,000 lines or 50 KB, with a note when cut.
  * `write {path, content}` creates parent directories.
  * `edit {path, oldText, newText}` needs exactly one match.
  * `bash {command, timeout?}` runs `bash -c` from `PATH` in the session `cwd`. Its combined output keeps the last 2,000 lines or 50 KB, with the exit status, and is killed on cancel through `exec.CommandContext` with `WaitDelay` 2 s.
  * Paths are relative to the session `cwd`. Absolute paths are taken as given until F1's confinement.
* **The system prompt is a short constant in `acpserver`:** gobble's identity, the `cwd`, the OS, and the tool list. 0005-PLAN F3 replaces it.
* **Loop limits.**
  * Tool calls run one after another; batching is a later phase.
  * At most 100 model calls per prompt, after which the turn ends with `max_turn_requests`.
  * A provider error ends the prompt with a JSON-RPC error: `auth_required` for `llm.ErrAuth`, or internal error with the message otherwise.
* **Usage.**
  * After each completed prompt the agent sends `usage_update` with `used` = the last model call's input plus output tokens, and `size` 0, because the catalog is F4. It sends no cost.
  * The prompt response's `usage` sums the turn's model calls.
  * An all-zero SDK report is replaced by an estimate (bytes ÷ 4) and marked `Estimated` (`llm` doc).
* **D19 items 2 and 3.**
  * Message and thought text are coalesced to at most one chunk per 50 ms or 4 KiB.
  * A tool call starts as `pending` with a title of at most 60 runes (`read internal/cli/term.go`, `run go test ./...`), then goes to `in_progress`.
  * It ends with its terminal status in an update of its own, whose first content block is a summary of at most 400 bytes. Diffs and output follow.
* **The CLI.**
  * `-t/--tools`, `--exclude-tools` and `--no-tools` leave `laterFlags`. They filter the in-process agent's tool set. An unknown tool name, or `--no-tools` with either of the others, is a usage error.
  * The line session prints `Warning: no model credential: set one of <variables>` at start when none is set. D1's `gobble configure` does not exist until 0005-PLAN F4, so the warning names the variables instead.
  * An `auth_required` turn exits 3 in print mode.
* **Tests never reach a real provider.** Every test that starts gobble's real agent clears the credential variables (`provider.CredentialVars()`). The `live_*` tests are F4's.

Steps:

1. **`go.mod`:** `go get github.com/maccavelli/go-llmprovider-sdk@<owner's tag>` and `github.com/google/jsonschema-go@v0.4.3` (0004-MADR names both), then `go mod tidy`. Record the tag here.
2. **`llm`** (`llm.go` and `doc.go`): the `Content` fields and types above.
3. **`tool`** (`tool.go`, a new `tool_test.go`):
   * `New` derives `InputSchema` with `jsonschema.For[In]`.
   * `Run` decodes and validates `call.Args` against the resolved schema, calls `fn`, and encodes `Out`.
   * An input that does not decode or validate is `Result{IsError: true}` with the reason. A context error is returned as an error.
   * A string `Out` is the text the model sees.
   * Plus `Env.Cwd`, the `Result` fields, `Diff` and `Describer`.
4. **`permission`** (`permission.go`): the `Rule` fields.
5. **`tool/builtin`** (new `builtin.go`, `read.go`, `write.go`, `edit.go`, `bash.go` and `builtin_test.go`):
   * `Tools() []tool.Tool` returns the four tools above, each with its kind, its annotations (`read` read-only; `write`, `edit` and `bash` destructive), and a `Describe` title.
6. **`agent`** (new `agent.go`, `event.go` and `agent_test.go`):
   * `New(Config{Provider, Tools, System, Model, Policy, MaxCalls})`, and `(*Agent).Run(ctx, history []llm.Message, prompt llm.Message) iter.Seq2[Event, error]`.
   * The sealed events are `TextDelta`, `ThinkingDelta`, `ToolStart{Call, Spec, Title}`, `ToolEnd{Call, Result}`, `UsageEvent` and `End{StopReason, Messages}`. `End.Messages` are the messages to append to the history.
   * A cancelled context ends the turn with `End{StopReason: "cancelled"}`, and no error.
7. **`llm/provider`** (new `provider.go`, `stream.go`, `ambient.go` and `provider_test.go`):
   * `New(id string, o Options) (llm.Provider, error)`, and `Wrap(p llmprovider.Provider) llm.Provider` for tests. The request, event and error mapping follow `llm`'s doc.
   * `Ambient(env func(string) string, o Options) (llm.Provider, error)`, with `ErrNoCredential` wrapping `llm.ErrAuth` and naming the variables. `CredentialVars() []string`.
   * The tests run on the SDK's `llmtest.Fake`: text, a tool call, usage on `done`, an all-zero usage estimated, a 429 keeping `RetryAfter`, overflow not retried, and ambient order and absence.
8. **`acpserver`** (`agent.go`, `serve.go`, new `prompt.go` and `coalesce.go`, and tests):
   * `Options` gains `Provider func() (llm.Provider, error)` and `Tools []tool.Tool` (nil means `builtin.Tools()`).
   * Each session keeps its history in memory; Phase 4 persists it.
   * `Prompt` runs the agent and translates its events as above. `Cancel` cancels the session's running turn. The permission `Policy` uses `session/request_permission`.
   * The capabilities do not change.
9. **`acpclient`** (`conn.go`):
   * `session/request_permission` is answered with the first `reject_once` option, else `reject_always`, else cancelled.
   * `ErrAuthRequired` is returned when a prompt fails with -32000.
10. **`internal/cli`:**
    * `agent.go` and `acp.go` pass `Provider: provider.Ambient(os.Getenv, …)`, and the tool filter.
    * `flags.go` handles the three tool flags.
    * `print.go` maps `ErrAuthRequired` to exit 3.
    * `line.go` prints the credential warning.
    * `view.go` draws a diff through step 11's `render.UnifiedDiff`, and drops Phase 2's `diff <path> (+N -M lines)` line.
11. **`internal/cli/render`** (new `udiff.go` and `udiff_test.go`; this is Phase 3 step 6):
    * `UnifiedDiff(path string, old *string, new string) string` is a Myers line diff with 3 lines of context, `--- a/<path>` / `+++ b/<path>` headers (`/dev/null` for a new file), and `@@` hunks.
    * It is tested on fixtures and on the real output of step 5's `edit`.
12. **Tests that drove the echo agent move to `llmtest` or `acptest`.**
    * `acpserver`'s `TestSession` golden is rebuilt on a scripted provider.
    * `TestGobbleACPBinary` runs with the credential variables cleared. It asserts that the prompt fails with -32000, stdout carries only JSON-RPC, and the process exits 0 within 2 s of end of input.
    * `internal/cli` tests clear the variables too.

**Accept**, as the phase states it, through `acpserver` with `acptest.Connect` and a test provider:
* A prompt that needs `read` of a temp file gives a `tool_call`, then a terminal `tool_call_update`. The test provider answers with the tool result it was sent, so the final agent message holds the file's contents.
* A `bash` of `sleep 30`, cancelled, ends with `cancelled` within 5 s.
* A completed prompt sends `usage_update`. That test is shown red on a scratch copy that does not send it.
* A `write` asks permission. The CLI's rejecting client leaves the file unwritten, and an allowing test client writes it.
* 200 deltas inside 10 ms become one chunk, and a 10 KiB delta becomes three. This uses `testing/synctest`.
* Every title is at most 60 runes, and the first content block of every terminal update is at most 400 bytes.

Each new check is shown failing first on a scratch copy:
* no `usage_update`;
* no permission request before `write`;
* the coalescer forwarding every delta;
* a title not cut to 60 runes;
* `Cancel` not cancelling the turn;
* `UnifiedDiff` without context lines.

**Verification.**
* `go test ./...`;
* `golangci-lint` for linux, darwin and windows;
* `make preflight` on Windows and in WSL, and `go test -race ./...` in WSL;
* a pty-driven `gobble -p` and line session in WSL, with the credential variables cleared: exit 3 and the warning.
* `go list -m github.com/maccavelli/go-llmprovider-sdk` prints the recorded tag.

The agent stages; the owner commits.

**Deferred from Phase 3, named:**
* the catalog's context `size` and cost (0005-PLAN F4);
* image input, which waits on the SDK's image support (0005-MADR);
* `os.Root` confinement and full tool parity (F1);
* the CLI's permission prompt (Phase 6);
* parallel tool batches, steering and follow-up queues (Phase 7 and the `agent` road map).

**2026-10-06 — Phase 4 made executable, in two parts (owner's decisions: `_meta` for the new-session flags, resume without replay in the CLI, two staged parts).**

The facts come from Pi's own source, read in full at `maccavelli/pi` `312184edb`, the commit report 0001 examined (owner's choice of source, 2026-10-06):
* `packages/coding-agent/docs/session-format.md`;
* `packages/coding-agent/docs/message-types.md`;
* `packages/coding-agent/src/core/session-manager.ts`.

They are:
* A file is a header line `{"type":"session","version":3,"id","timestamp","cwd","parentSession"?}`, then one entry per line: `{type, id, parentId, timestamp, …}`, where `id` is 8 hex characters, collision-checked (`:277-284`), and `parentId` is `null` at a root.
* Session ids are UUIDv7 and must match `^[A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?$` (`:264-274`).
* Files are named `<timestamp with : and . made ->_<id>.jsonl` (`:1079-1080`), in `--<cwd without its leading separator, with / \ : made ->--/` (`:589-594`). A custom session directory is flat, and is filtered by the header's `cwd` (`:1906-1914`).
* A file is created only once a user or assistant message exists. Until then the entries stay in memory (`:1160-1189`). After that, each entry is one appended line.
* Messages are Pi's `AgentMessage`:
  * `user {content: string | blocks, timestamp}`;
  * `assistant {content: [text | thinking | toolCall], api, provider, model, usage, stopReason, timestamp}`;
  * `toolResult {toolCallId, toolName, content, isError, timestamp}`.
* The leaf is the last entry in file order (`:1103-1121`). Context is the path from the leaf to the root through `parentId`. Model and thinking come from `model_change` and `thinking_level_change` on the path, and from assistant messages' `provider` and `model`; thinking defaults to `off` (`:418-433`).
* Pi skips malformed lines and rejects only a file whose first line is not a session header (`:616-670`).

**Superseded by 0005-MADR "Sessions (changed)"**, recorded here so the accept below is read correctly:
* Phase 4's "a corrupt JSONL file fails closed" now means a file whose header is unreadable.
* A malformed line is skipped with a diagnostic, as in Pi, and gobble never rewrites a file on load.

The owner decided three questions on 2026-10-06:

* **`--session-id`, `--fork` and `-n/--name` ride on `session/new`'s `_meta`,** as `{"gobble": {"sessionId", "forkFrom", "name"}}`.
  * `_meta` is ACP's own extensibility point: no new method, and any client may use it.
  * An invalid or duplicate id, or an unknown fork source, is invalid params.
* **The CLI resumes without replay.** `-c` and `--session` use `session/resume`. The line session prints one line, `resumed <name or first prompt> (N messages)`. `session/load`'s compact replay (D19 item 5) is built and tested for ACP clients.
* **Two staged parts,** each with its own gates, negative tests and execution entry, each committed by the owner.

Choices made within the wording:
* **Version.** Phase 4 reads and writes v3 only. A header with another version fails closed with `unsupported session version N`; v1 and v2 migration is 0005-PLAN F2.
* **Unknown entries.** Entry types other than `message`, `model_change`, `thinking_level_change` and `session_info` are kept verbatim. They take no part in context until F2 (`compaction`, `context_edit`, `branch_summary`, `custom_message`) and Phase 6 (compaction).
* **Location.** Sessions live in the data directory's `sessions/` (0003-MADR), laid out as Pi lays them out, so F2's import copies files as they are.
* **What is written.**
  * gobble writes no system-message entry: Pi loads sessions without one (session-format.md), and F3 owns the prompt.
  * Assistant messages carry `api: "go-llmprovider-sdk"`, the provider id and the model. Usage is in Pi's shape with zero cost until the catalog; an estimate adds `"estimated": true`, an extra member Pi ignores.
  * Thinking blocks carry Pi's `thinkingSignature` (or the encrypted payload with `redacted: true`), and keep gobble's full provider data under an extra `gobble` member. Tool calls carry `arguments` as an object, and `thoughtSignature`.
* **Durability (D19 item 6).**
  * The user's message is written before the turn starts.
  * Each assistant message is written when its model call ends, before its tool calls are reported.
  * Each tool result is written before its terminal `tool_call_update` is sent.
  * Each write is one unbuffered `write` of one line.
* **The writer lock (D19 item 8).**
  * The writer holds an exclusive lock on a sidecar `<file>.lock` that holds its pid: `flock` on Unix, and `LockFileEx` on a range past the data on Windows, so the pid stays readable. Both go through `golang.org/x/sys`, which 0008-MADR D19 names.
  * Readers (`session/list`, a fork's source) take no lock.
  * A second writer gets `ErrLocked{ID, PID}`, which ACP returns as invalid request, `session <id> is open in another gobble process (pid N)`. Part B makes it exit 1.
* **Config options (D19 item 4).** `model` and `thought_level` are ungrouped `select` options with `category` set.
  * `model` lists the ambient provider's curated models and appears only when a provider is chosen.
  * `thought_level` lists `off, minimal, low, medium, high, xhigh`.
  * A change writes `model_change` or `thinking_level_change`, and applies from the next prompt (the model through `llm.Request.Model`, thinking through `llm.Request.Thinking`).
  * On load, the saved model is restored when its provider is the current one; otherwise the current provider's default is used and nothing is written.
* **`session/list`** returns sessions that have a file, plus the live ones in this process, newest first. A title is the session name, else the first user prompt cut to 60 runes. There is no paging.

**Part A** (files: `session/session.go`, `doc.go`, `memory.go`, `path.go` and tests; `session/jsonl/store.go`, `log.go`, `lock_unix.go`, `lock_windows.go`, `doc.go` and tests; `agent/agent.go` and `event.go`; `acpserver/agent.go`, `history.go`, `config.go` and tests; `internal/cli/agent.go`; `go.mod` and `go.sum`):

1. **`session`:**
   * `Header`, `Entry`, `Message`, `Block`, `Usage` and `Cost` in Pi's v3 shapes, each with a `json:",embed"` `Unknown` member, so unknown members round-trip.
   * `NewEntryID`, `ValidID`, and `Path(entries) []Entry` (leaf to root, in order).
   * The `Store` and `Log` contracts, with `Log.Close` releasing the writer, and `Store.Read` for a reader.
   * `NewMemoryStore()` for `--no-session` and tests.
2. **`session/jsonl`:** `New(dir string, o Options{Flat bool; Now func() time.Time; Logger *slog.Logger}) *Store`. It provides:
   * lazy creation, opening with the lock, and listing with the `cwd` filter;
   * a skip-and-warn reader;
   * the fail-closed header and version check.
3. **`agent`:**
   * `Config.Thinking`, sent as `llm.Request.Thinking`.
   * A `Message{Message, Usage, StopReason}` event after each model call, before its tools run.
   * Tool results carry `ToolName`.
4. **`acpserver`:**
   * `Options` gains `Store session.Store` (nil is a memory store), `Models func() ModelChoice` and `Now func() time.Time`.
   * Capabilities: `loadSession`, `sessionCapabilities.list`, `resume` and `close`.
   * `NewSession` (with `_meta`), `Prompt` (writing as above), and `LoadSession` (compact replay, then a `usage_update` snapshot).
   * `ResumeSession`, `ListSessions`, `CloseSession` (releasing the lock), and `SetSessionConfigOption`.
5. **`internal/cli`:** the agent's store is `jsonl.New(<data>/sessions)` for `gobble acp` and the in-process agent, with the ambient provider's `ModelChoice`.

**Part A accept,** through `acptest.Connect`, and over `gobble acp` where named:
* `NewSession` → `Prompt` (with a tool call) → `CloseSession` → `LoadSession` in a **new** `Agent` over the same directory:
  * the replay is one `user_message_chunk`, one `agent_message_chunk` per assistant message, one `tool_call` in its final state, then `usage_update`;
  * a next prompt sends the whole history to the model.
* `ResumeSession` continues with no replay. `ListSessions` includes the id, with its title, and filters by `cwd`.
* A file whose header is unreadable fails closed with an ACP error. A malformed middle line is skipped, and the load succeeds.
* `set_config_option` with `model` changes the model the next request names. A load in a new `Agent` continues with the saved model, with no `set_config_option` call. `thought_level` applies to the next prompt of the same session.
* A raw `session/set_model` request gets -32601, asserted.
* A second `Agent` loading a session the first holds gets the lock error, with the first's pid.
* The terminal `tool_call_update` is sent only after its `toolResult` line is on disk: the test client reads the file when the update arrives.
* No file exists after `session/new` alone, nor after a prompt refused for want of a credential.

Each is shown failing first on a scratch copy, the failure quoted:
* replay missing the tool call;
* the lock not taken;
* the tool result written after its update;
* the file created at `session/new`;
* the model change not applied.

**Part B** (files: `acpclient/conn.go` and tests; `internal/cli/chat.go`, `flags.go`, `line.go`, `print.go`, `completion.go` and tests):

1. **`acpclient`:**
   * `ResumeSession`, `ListSessions` and `NewSession` options (`ID`, `ForkFrom`, `Name`) through `_meta`;
   * `ErrSessionLocked`, carrying the agent's message.
2. **`internal/cli`:** the seven flags leave `laterFlags`.
   * `-c` resumes the newest session for the `cwd`.
   * `--session` takes a path (its header's id, which must be in the store), an id, or a unique prefix of an id or name. It is ambiguous or not found → exit 2.
   * `--session-id`, `--fork` and `-n` go through `_meta`.
   * `--no-session` uses a memory store; `--session-dir` uses a flat `jsonl.New(dir, Flat)`.
   * A locked session exits 1 with `Error: session <id> is open in another gobble process (pid N)`.
   * The line session prints the `resumed …` line.
3. **Completion:** `--session` and `--fork` complete from the store, newest first, keep-order, described as `name · date · cwd`. A test over `GOBBLE_COMPLETE` checks that nothing is created.

**Part B accept:**
* each flag through `Main` against a real `acpserver` with a scripted provider;
* the D12 conflicts, unchanged;
* exit 1 on a lock;
* the 0008-MADR Confirmation line for `GOBBLE_COMPLETE=bash … --session`.

Shown failing first:
* `-c` picking the oldest;
* a prefix matching two sessions accepted;
* `--no-session` writing a file.

**Verification, each part.**
* `go test ./...`;
* `golangci-lint` for linux, darwin and windows;
* `make preflight` on Windows and in WSL, and WSL `go test -race ./...`.

The agent stages; the owner commits.

**Deferred from Phase 4, named:**
* v1 and v2 migration, compaction, context edits, branches, labels, fork and clone over the tree, and Pi import (0005-PLAN F2, and Phase 6 for compaction);
* replay in the CLI (owner's decision);
* paging in `session/list` (until a list is long enough to need it);
* `session_info_update` for names (Phase 6, with `/name`).

**Deviations before Part B, 2026-10-06 (owner's decisions):**

1. **The message count of `resumed … (N messages)` comes from the agent.**
   * ACP's `SessionInfo` has no count.
   * `session/resume` and `session/load` therefore answer with `_meta {"gobble": {"title", "messages"}}`, the channel `session/new` already uses. `messages` counts the path's user and assistant messages.
   * This adds `acpserver/agent.go`, and a test in `acpserver/session_test.go`, to Part B's files.
2. **`-n/--name` names a new session only.**
   * With `-c` or `--session` it exits 2 with `--name with --continue or --session is not yet available (0002-PLAN Phase 7)`. Renaming an existing session waits for Phase 7's `_gobble/set_session_name`.
   * This is D12's rule that a flag is never silently ignored.
3. **`--tools` and `--exclude-tools` completion is wired to `builtin.Names()`.**
   * It still listed nothing: its comment named 0002-PLAN Phase 3, and Phase 3 did not wire it. That is a gap from Phase 3; `internal/cli/completion.go` is unchanged since 0008-PLAN P5.
   * It is fixed in Part B's `completion.go`, with a test.

**2026-10-06 — Phase 5 made executable (owner's decisions: MCP OAuth in 0005-PLAN F8, `${NAME}` values now, background connections with a first-prompt wait, direct exposure, two items pulled from F8, Pi's tool-flag rules).**

The facts come from Pi's source at `maccavelli/pi` `312184edb`, the commit Phase 4 read.

Read in full:
* `packages/coding-agent/docs/mcp.md`;
* `packages/coding-agent/src/extensions/mcp/`: `config.ts`, `cli.ts`, `tools.ts`, `runtime.ts` and `index.ts`;
* `packages/coding-agent/src/core/`: `mcp-servers.ts` and `resolve-config-value.ts`;
* `packages/mcp/src/`: `client.ts`, `transports/stdio.ts` and `protocol/content.ts`.

Read in part: `packages/coding-agent/docs/cli.md`, its Tools and MCP commands sections.

Not read: `packages/mcp/src/transports/streamable-http.ts` (gobble uses the go-sdk's transport) and the OAuth files (F8).

The facts:
* **The schema** is checked one entry at a time (`mcp-servers.ts:152-196`).
  * A name matches `^[A-Za-z0-9_-]+$`, and an entry is an object.
  * `exposure` is one of `codemode`, `codemode-deferred`, `deferred`, `direct` or `hidden`. `toolExposure` maps names to those, `enabled` is a boolean, and `timeout` is a positive number of seconds.
  * `type: "sse"` is rejected: `server "<name>": legacy SSE transport is not supported; use the streamable HTTP URL`.
  * `url` (http or https) makes an HTTP server when `type` is absent, `http` or `streamable-http`. It takes a `headers` string map and a checked `oauth` object.
  * Otherwise `command` makes a stdio server when `type` is absent or `stdio`. It takes an `args` string array, an `env` string map and a `cwd` string.
  * Otherwise: `server "<name>" needs either "command" (stdio) or "url" (streamable HTTP)`.
* **Errors are per entry.** An invalid entry is skipped and reported as `<path>: <message>`. A file that is not an object with an `mcpServers` object is one error. A missing file is no servers (`config.ts:68-92`).
* **Values** (`resolve-config-value.ts`):
  * A value that starts with `!` is a command.
  * Otherwise `$NAME` and `${NAME}` interpolate, and `$$` and `$!` escape. `${` with no `}`, or with a name that is not `[A-Za-z_][A-Za-z0-9_]*`, stays literal.
  * An unset or empty variable fails with `Failed to resolve MCP server "<name>" header "<key>" from environment variable: <VAR>` (`env "<key>"` for env values; `:229-251`).
  * Values are resolved when the transport is built (`runtime.ts:96-120`), so a failure fails that server, not the file.
* **Stdio** (`runtime.ts:87-120`, `stdio.ts:94`): `~` and `~/` expand in `command`, `args` and `cwd`; a relative `cwd` resolves against the session's directory; the environment is the process's own plus `env`.
* **Tool names** (`tools.ts:80-89`): `mcp__<server>__<tool>`, with every character outside `[A-Za-z0-9_-]` made `_`. A name over 64 characters, or one another tool already has, is cut to 55 characters and given `_` and the first 8 hex digits of sha256(`<server>\0<tool>`).
* **Tool definitions** (`tools.ts:232-305`, `runtime.ts:47,216`):
  * The description is the trimmed description, else the title, else `MCP tool <tool> from server <server>`.
  * The input schema gets `"type": "object"` and `"properties": {}` when either is missing.
  * The four boolean annotations are copied.
  * A call times out after `timeout` seconds, 60 by default.
* **Results** (`content.ts:78-117`, `tools.ts:206-226`):
  * text passes through, and an image goes to the model as an image;
  * audio becomes `[audio <mime> omitted]`;
  * a resource link becomes `[Resource <uri> "<title or name>" (<mime>, <size>): <description>]`;
  * an embedded text resource becomes its text; another embedded resource becomes `[binary resource <uri> (<mime>) omitted]`;
  * a result with no content but `structuredContent` becomes that JSON, indented two spaces;
  * an `isError` result with no text gets `MCP tool <server>/<tool> returned an error`.
* **Start-up** (`index.ts:791-848`):
  * Connections start when the session starts.
  * The first prompt waits until they finish or 10 s pass, once per session, counted from that prompt.
  * A server that connects later adds its tools from then on.
  * Problems are reported once, after start-up.
* **`pi mcp`** (`cli.ts`):
  * `add` writes `Added|Replaced <scope> MCP server "<name>" in <path>.`, then `Check it with: pi mcp list`. It rewrites the file keeping the rest of it, in the file's own indentation (`config.ts:168-182`).
  * `remove` writes `Removed <scope> MCP server "<name>" from <path>.`, or exits 1 with `No <scope> MCP server named "<name>" in <path>.`.
  * `list` connects every enabled server. For each it prints `<name>: connected, N tools (<exposure>, <scope>)`, the transport, `tools: a, b`, and the error with the server's stderr tail (2000 characters). Then come `config error: …` lines.
  * `list` exits 1 when there is a config error, or an enabled server is not connected. `--json` prints `{servers, errors}`.
* **Tool flags** (`cli.md:119-126`): `--tools` is an allowlist over all tools, MCP tools included; `--exclude-tools` removes names; `--no-tools` disables all.

Measured on the go-sdk:
* **Version.** `v1.8.0` is the newest release on the module proxy on 2026-10-06.
* **What it adds to `go.mod`,** measured on a scratch copy by importing it and running `go mod tidy`:
  * direct: `github.com/modelcontextprotocol/go-sdk v1.8.0`, which 0004-MADR D14 names;
  * indirect, coming with it: `github.com/segmentio/asm v1.1.3`, `github.com/segmentio/encoding v0.5.4`, `github.com/yosida95/uritemplate/v3 v3.0.2`, `golang.org/x/oauth2 v0.35.0` and `golang.org/x/time v0.15.0`.
* **401.** `StreamableClientTransport` takes an `OAuthHandler` (`mcp/streamable.go:1966-1997`). Without one, a 401 is an error whose text ends `Unauthorized` (`:2607-2609`). There is no typed error, so gobble's transport records the status itself.
* **Stopping.** `CommandTransport.Close` closes stdin, waits `TerminateDuration` (5 s by default), then terminates the process (`mcp/cmd.go:15-30`).

The owner decided six questions on 2026-10-06:

1. **MCP OAuth lands with 0005-PLAN F8,** which owns it together with the keyring token store and `gobble mcp login|logout`. go-llmprovider-sdk's `llmprovider/auth` is provider-only (`oauth_loopback.go:184-204` accepts OpenAI and Grok only), and 0004-MADR already routes MCP OAuth to the go-sdk.
   * In Phase 5, an HTTP server with no `Authorization` header that answers 401 shows as `needs sign-in`, with `MCP server "<name>" requires sign-in, which arrives with 0005-PLAN F8 (gobble mcp login)`.
   * With an `Authorization` header, a 401 is a failure, `Unauthorized`.
2. **`$NAME`, `${NAME}`, `$$` and `$!` work in Phase 5.** A `!command` value fails its server: `MCP server "<name>" header "<key>": command values (!…) arrive with 0005-PLAN F8`.
3. **Connections run in the background.** They start on `session/new`, `session/load` and `session/resume`, which return at once.
   * The session's first prompt waits up to 10 s for any still in progress, counted from that prompt as Pi counts it.
   * A server that connects later is in the tools of the next prompt.
   * Problems are logged once, at Warn, after that wait: on stderr in the CLI modes, and in the log file under `gobble acp`.
4. **Every MCP tool is declared directly.** `exposure`, `toolExposure` and `oauth` are checked as Pi checks them, and kept when `gobble mcp add` or `remove` rewrites the file, but not applied. A server that sets `exposure` or `toolExposure` is logged once: `MCP server "<name>": exposure and toolExposure apply from 0005-PLAN F8; its tools are declared directly`. `enabled: false` is honoured.
5. **Two items are pulled forward from F8:** Pi's tool-name sanitizing with the hash suffix, and the per-call `timeout`. Progress notifications resetting the timeout stay in F8.
6. **The tool flags follow Pi.**
   * `--no-tools` turns MCP off.
   * `-t` keeps only the names listed, MCP names included.
   * `--exclude-tools` removes the names listed.
   * A name that starts `mcp__` is accepted unchecked, since MCP names exist only after connecting. Any other unknown name is still a usage error.
   * When the flags leave no MCP name selectable, nothing connects.

Choices made within the wording:
* **Where the file is.** `<config>/mcp.json` (0003-MADR table). There is no project file until trust (0005-PLAN), and no Pi files until the bridge.
* **The union.**
  * The file's servers come first, in file order.
  * A session server of the same name replaces it in place, and other session servers follow, in request order.
  * Values from an ACP client are used as given, not interpolated: the client has already resolved them.
* **Refused ACP entries.**
  * An SSE entry is refused as invalid params, `MCP server "<name>": the SSE transport is not offered (mcpCapabilities.sse is false)`. An ACP-transport entry is refused the same way, naming `mcpCapabilities.acp`.
  * An ACP server name that fails Pi's name rule is invalid params, with Pi's message.
* **Capabilities.** `mcpCapabilities` is `{http: true, sse: false, acp: false}`.
* **Identity.**
  * MCP `clientInfo` is `{name: "gobble", version: <semver>}` (0003-MADR).
  * HTTP requests carry `buildinfo`'s User-Agent and the configured headers, through a `RoundTripper` that also records a 401.
  * The session's directory is offered as the one root, as Pi offers it.
* **Bounds.**
  * `initialize` and `tools/list` share the server's `timeout`, as Pi's request timeout covers `initialize` (`client.ts:418`).
  * The last 64 KiB of a stdio server's stderr is kept, and an error shows its last 2000 characters.
* **Stopping.** A stdio server stops through the go-sdk's `CommandTransport.Close`. The process-group stop is Phase 8 step 1 and F8 step 3.
* **How a tool shows.** MCP tools are ACP kind `other`, titled `<server>/<tool>`. A call asks for approval unless its tool has `readOnlyHint`; the agent's rule is unchanged.
* **Images.** gobble's tool results are text (`tool.Result`), so an image becomes `[image <mime> omitted]`. It reaches the model once 0005-PLAN F1 step 2 carries images.
* **Exit codes** are 0008-MADR D10's: 2 for usage, 1 for failure.
* **`gobble mcp add` flags.**
  * The flags are `--url`, `--env KEY=VALUE` (repeatable), `--cwd`, `--header KEY=VALUE` (repeatable), `--bearer-token-env-var NAME` (writes `Authorization: Bearer ${NAME}`) and `--exposure`. The command and its arguments follow `--`.
  * There is no `--local`, which needs trust. There are no `--oauth-*` flags: they arrive with F8's OAuth, and a literal secret on the command line is not allowed here.
* **`gobble mcp list`** prints Pi's text without the scope, with the exposure in force: `<name>: connected, 2 tools (direct)`.

**Files:**
* `go.mod`, `go.sum`;
* `mcpclient/`:
  * `doc.go`;
  * new `config.go`, `file.go`, `value.go`, `name.go`, `tool.go`, `conn.go` and `manager.go`;
  * their tests;
* new `mcpclient/mcptest/doc.go` and `mcptest.go`;
* `acpserver/`:
  * `agent.go`, `prompt.go`, `serve.go` and `agent_test.go`;
  * new `mcp.go`, `mcp_test.go` and `testmain_test.go`;
* `internal/cli/`:
  * `root.go`, `agent.go`, `chat.go`, `flags.go`, `completion.go`, `completion_test.go` and `testmain_test.go`;
  * new `mcp.go` and `mcp_test.go`;
* `docs/decisions/0002-PLAN-cli-acp-headless-mcp-v1.md`;
* `docs/decisions/0005-PLAN-v1-feature-scope.md`: an F8 note on what Phase 5 delivered;
* `docs/README.md`.

**Steps:**

1. **`mcpclient/config.go`:**
   * `Server{Name, Type, Command string; Args []string; Env map[string]string; Cwd, URL string; Headers map[string]string; Timeout time.Duration; Enabled bool; Exposure string; ToolExposure map[string]string; OAuth bool}`.
   * `Validate(name string, raw jsontext.Value) (Server, error)` gives Pi's checks and messages, word for word. `Server.Transport()` is the URL, or the command and its arguments.
   * `Config{Path string; Servers []Server; Errors []string}`, and `Load(path string) Config`. A missing file is empty, and a malformed one is a single error.
2. **`mcpclient/file.go`:**
   * `Add(path, name string, entry jsontext.Value) (replaced bool, err error)` and `Remove(path, name string) (bool, error)`.
   * They read the file as ordered members, keeping every other member and the order of servers. An added server goes last, and a replaced one stays in place.
   * They write in the file's first indentation, else two spaces, with a final newline. The write goes to a temporary file in the same directory, renamed over the old one, with mode 0600 in a directory made 0700.
3. **`mcpclient/value.go`:** `Resolve(value, what string, getenv func(string) string) (string, error)` gives Pi's template rules and messages. A value starting `!` gives decision 2's error.
4. **`mcpclient/name.go`:** `ToolName(server, tool string, taken func(string) bool) string`, Pi's algorithm.
5. **`mcpclient/tool.go`:** an MCP tool as a `tool.Tool`, a `tool.Describer` that titles calls `<server>/<tool>`.
   * Its `Spec` is the name, Pi's description fallback, the input schema with `type` and `properties` filled in, `KindOther`, and the annotations.
   * `Run` calls `tools/call` within the server's timeout. A timeout gives the error result `MCP tool <server>/<tool> timed out after N s`.
   * The result is converted as above, with `IsError` kept.
6. **`mcpclient/conn.go`:**
   * A stdio server is an `exec.Cmd` with `~` expanded, the `cwd` resolved, and the environment the process's plus the resolved `env`. Its stderr goes to a 64 KiB tail.
   * An HTTP server uses `StreamableClientTransport` with an `http.Client` whose `RoundTripper` adds the resolved headers and the User-Agent, and records a 401.
   * A connection runs `initialize` and lists the tools, following every page. Its state is `connecting`, `connected`, `needs sign-in`, `failed` or `closed`, with an error text.
7. **`mcpclient/manager.go`:**
   * `Start(servers []Server, o Options) *Manager` connects each enabled server in its own goroutine. `Options` holds `{Cwd, ClientName, ClientVersion, UserAgent; Getenv func(string) string; Home string}`.
   * `Wait(ctx) error` waits until every connection has finished.
   * `Tools(allow func(string) bool) []tool.Tool` gives the connected servers' tools, in server order, named across the whole set.
   * `Statuses() []Status`, where `Status` is `{Name, Transport, State, Error; Tools []string; Exposure string}`. Also `Problems() []string` and `Close() error`.
   * `mcpclient/doc.go` describes the package and drops "Phase 1 declares the package only".
8. **`mcpclient/mcptest`:** fixtures for tests, built on the go-sdk server.
   * `NewServer()` offers `echo {text}` (read-only), `add {a, b}`, `sleep {ms}`, and the tool `dotted.name`.
   * `ServeStdioIfAsked()` serves the fixture on stdin and stdout when `GOBBLE_MCPTEST=stdio`, then exits. `GOBBLE_MCPTEST_DELAY` delays the start, and `GOBBLE_MCPTEST_FAIL` writes a line to stderr and exits 3.
   * `StdioServer(t) (command string, env map[string]string)` is this test binary with the variable set.
   * `HTTPServer(t, bearer string) string` serves the fixture over streamable HTTP. Given a bearer, it answers 401 to any request without it.
9. **`acpserver`:**
   * `Options` gains `Logger *slog.Logger`, and `MCP MCPOptions{Config string; Allow func(string) bool; UserAgent string; StartupWait time.Duration; Getenv func(string) string}`. `Serve` passes its logger when the agent has none.
   * `Capabilities` sets `McpCapabilities{Http: true}`.
   * New `mcp.go` converts ACP servers, builds the union and starts a session's `Manager`. `NewSession`, `LoadSession` and `ResumeSession` start it; `CloseSession` and `Close` close it.
   * `Prompt` waits once (decision 3), logs the problems once, and runs the agent with the built-ins plus `Manager.Tools(Allow)`. The system prompt lists them all.
10. **`internal/cli`:**
    * `agentOptions` sets `MCP.Config` to `<config>/mcp.json`, `UserAgent`, and `Allow` from the tool flags (decision 6).
    * `toolSet` accepts `mcp__` names, and turns MCP off for `--no-tools`, or for `-t` without an `mcp__` name.
    * The `-t` and `--no-tools` help keeps its wording, which is now true of MCP tools too.
    * New `mcp.go` adds `MCPCmd{Add, Remove, List}` to `Root`:
      * `list --json` prints `{servers: [Status…], errors}`;
      * `add` validates its entry with `Validate` before writing.
    * `completion.go` registers `mcpserver`: the server names of `<config>/mcp.json`, read only. `mcp remove`'s argument uses it, and the coverage test covers the new flags.

**Accept,** each through the real `acpserver`, or through `Main`, with `llmtest.Script` as the model and each over its own `GOBBLE_HOME`:
* **Both fixtures appear as tools during `session/prompt`.** A stdio fixture in `mcp.json` and an HTTP fixture in `session/new`'s `mcpServers` give `mcp__fx__echo`, `mcp__fx__add`, `mcp__web__echo` and so on in the model's request (`llmtest.AssertTools`). A call of `mcp__fx__echo {"text": "hi"}` returns `hi` to the next request, and a call of `mcp__web__add` returns `5`.
* **The session's name wins.** A file entry `web` pointing at a closed port, and a session `web` pointing at the fixture, give `web`'s tools.
* **`gobble mcp list` agrees.**
  * Over both fixtures it exits 0, with `fx: connected, 4 tools (direct)` and a `tools:` line naming the four, in the order the server lists them.
  * With a server started with `GOBBLE_MCPTEST_FAIL` it exits 1, and prints that server's stderr line.
  * `--json` decodes.
* **An SSE-only config is rejected.** In `mcp.json`, `list` exits 1 with `config error: <path>: server "old": legacy SSE transport is not supported; use the streamable HTTP URL`. On `session/new`, it is invalid params.
* **The Phase 1 initialize test still passes,** now asserting `http` true and `sse` and `acp` false.
* **`${NAME}`.**
  * The HTTP fixture with a bearer connects when `Authorization: Bearer ${FX_TOKEN}` has the variable set.
  * Unset, it fails with `Failed to resolve MCP server "web" header "Authorization" from environment variable: FX_TOKEN`.
  * With no header, it shows `needs sign-in` with decision 1's text, and `list` exits 1.
* **Names.** `dotted.name` is `mcp__fx__dotted_name`. A 70-character tool name gives a 64-character name, whose suffix matches a vector computed with Python's `hashlib`.
* **Timeout.** `sleep {ms: 2000}` with `"timeout": 0.5` gives the timeout error result, within 2 s.
* **Start-up.**
  * With `StartupWait` of 200 ms and `GOBBLE_MCPTEST_DELAY=1500ms`, the first prompt runs without `fx`'s tools, and one Warn names `fx`.
  * A prompt after `fx` connects has them.
* **Tool flags.**
  * `-t read,mcp__fx__echo`: the request has exactly those two.
  * `--exclude-tools mcp__fx__add`: the request lacks it.
  * `--no-tools`: no request has an `mcp__` tool, and the fixture process is never started (its start writes a marker file).
* **`add` and `remove`.**
  * Both keep another top-level member, the servers' order and a four-space indentation.
  * `add` prints `Replaced` the second time.
  * `remove` of an unknown name exits 1.
  * `mcp remove <TAB>` lists the file's servers and creates nothing.

Each is shown failing first on a scratch copy, the failure quoted:
* `http` false;
* the file's entry winning the union;
* SSE accepted;
* `list` exiting 0 on a failed server;
* names not sanitized;
* `${NAME}` not expanded;
* the timeout not applied;
* `--no-tools` leaving MCP on.

**Verification:**
* `go test ./...`;
* `golangci-lint` for linux, darwin and windows;
* `make preflight` on Windows and in WSL, and WSL `go test -race ./...`;
* the pre-add check on every changed Go file;
* the identifier and hidden-character scans.

The agent stages; the owner commits.

**Deferred from Phase 5, named:**
* to 0005-PLAN F8:
  * OAuth, and `gobble mcp get|login|logout`;
  * `!command` values;
  * applying `exposure` and `toolExposure`;
  * the 20 KB truncation with the full-output file;
  * `list_changed`, reconnects and retries, and the progress-reset timeout;
  * `mcp.log`, and `/mcp`;
* to Phase 8 and F8: the process-group stop;
* to 0005-PLAN (trust and the bridge): the project `mcp.json`, `--local`, and Pi's own files;
* to 0005-PLAN F1 step 2: images in MCP results;
* to 0005-PLAN F1 step 8: the MCP resource tools.

**Deviations during Phase 5, 2026-10-06 (owner's decisions):**

1. **`acpserver/testdata/session.golden` joins Phase 5's files.**
   * It is Phase 1's golden transcript for `TestSession`, and records `initialize`'s capabilities. It was last changed in 97dc23e.
   * Turning on `http` changes its line 2 only: `"mcpCapabilities":{}` becomes `"mcpCapabilities":{"http":true}`. TestSession's failure against the unmodified golden showed exactly that line.
   * It is regenerated with `ACPTEST_UPDATE=1`, and a diff confirms line 2 is the only change.
2. **`internal/cli/main.go` joins Phase 5's files.**
   * `runEnv`, which Kong binds into every command, is defined there. The MCP tool choice from `-t`, `--exclude-tools` and `--no-tools` reaches `agentOptions` through it, as `tools` already does. The change is two fields, `mcpOff` and `mcpAllow`, beside `tools`.
   * The edit was made before the deviation was raised; the owner kept it rather than routing the choice through `agentOptions`' parameters, which would have touched `internal/cli/acp.go` instead.
3. **Two staticcheck findings are kept, with staticcheck's own `//lint:ignore`.** `make preflight`'s standalone staticcheck failed on Windows and in WSL, while `golangci-lint` passed, because `//nolint` silences only golangci.
   * **SA1019** at `mcpclient/conn.go`: `mcp.Client.AddRoots` and `mcp.Root` are deprecated from MCP protocol 2026-07-28, and still served for at least twelve months. The amendment offers the session's directory as the one root, as Pi does, and servers on earlier protocols, such as the filesystem server, read roots.
   * **ST1005** at `mcpclient/value.go`: Pi's messages begin `Failed to resolve`, with a capital, and the amendment quotes them word for word.
   * Each line gets `//lint:ignore <check> <reason>`, the repository's first, and keeps its golangci `//nolint:staticcheck`: golangci-lint reads only `//nolint`, and staticcheck only `//lint:ignore`, as running both showed.

**Phase 5, 2026-10-06 — complete (staged; the owner commits).** It ran as the amendment wrote it, with the three deviations above. Phases 6–8 have not run.

* **Files.**
  * `go.mod` and `go.sum`: `github.com/modelcontextprotocol/go-sdk v1.8.0`, with the five indirect modules the amendment measured, and no others.
  * `mcpclient`: `doc.go`, and new `config.go`, `file.go`, `value.go`, `name.go`, `tool.go`, `conn.go` and `manager.go`, with `config_test.go`, `file_test.go`, `value_test.go` and `manager_test.go`.
  * New `mcpclient/mcptest/doc.go` and `mcptest.go`.
  * `acpserver`: `agent.go`, `serve.go`, `agent_test.go` and `testdata/session.golden` (deviation 1), and new `mcp.go`, `mcp_test.go` and `testmain_test.go`.
  * `internal/cli`: `root.go`, `agent.go`, `chat.go`, `completion.go`, `main.go` (deviation 2) and `testmain_test.go`, and new `mcp.go` and `mcp_test.go`.
  * Listed but unchanged: `acpserver/prompt.go` (its system prompt already lists every tool it is given), `internal/cli/flags.go` (the help is unchanged, as step 10 says) and `internal/cli/completion_test.go` (its coverage walk covers the new flags as written).
* **Accept,** each through the real `acpserver` or through `Main`:
  * `TestMCPToolsInPrompt`: a stdio fixture from `mcp.json` and an HTTP fixture from `session/new` both give their tools to the model's request. `mcp__fx__echo` returns `hi` and `mcp__web__add` returns `5` to the next request, and the system prompt lists them.
  * `TestSessionServerWins`: a session's `web` replaces the file's, which points at a closed port.
  * `TestMCPListAgrees`: `gobble mcp list` exits 0 with `fx: connected, 4 tools (direct)` and `web: connected, 4 tools (direct)`. With a server that fails at start, it exits 1 and prints that server's stderr line. `--json` decodes.
  * `TestMCPListRejectsSSE`: `config error: <path>: server "old": legacy SSE transport is not supported; use the streamable HTTP URL`, exit 1. `TestUnofferedTransportsRefused`: SSE, ACP-transport and badly named servers on `session/new` are invalid params.
  * `TestCapabilitiesAreHonest` asserts `http` true and `sse` and `acp` false, and `TestSession`'s golden changed in line 2 only.
  * `TestBearerAndSignIn`: `Bearer ${FX_TOKEN}` connects with the variable set. Unset, it fails with `Failed to resolve MCP server "web" header "Authorization" from environment variable: FX_TOKEN`. With no header, a 401 is `needs-auth`, with decision 1's text, and `mcp list` exits 1 (`TestMCPListEmptyAndSignIn`).
  * `TestToolName` matches four vectors computed with Python's `hashlib` the way Node computes them, including a 70-character name, a taken name, and a character outside the Basic Multilingual Plane.
  * `TestCallTimeout`: `sleep {ms: 5000}` with a 0.5 s timeout gives `MCP tool fx/sleep timed out after 0.5 s`, well within 3 s.
  * `TestStartupWait`: with a 200 ms wait and a 1500 ms server start, the first prompt runs without `fx`'s tools and logs `still connecting` with `server=fx`; the prompt after the server connects has them. `TestResumeConnects`: `session/resume` connects the servers it carries.
  * `TestToolFlagsSelectMCPTools`: `-t read,mcp__fx__echo` gives exactly those two; `--exclude-tools mcp__fx__add` removes one; `--no-tools` and `-t read` start no server, by the fixture's marker file.
  * `TestMCPAddRemove`: `add` with and without `--`, `Replaced` on the second, the HTTP entry with `Bearer ${DOCS_TOKEN}` and `exposure`, and `remove` keep another top-level member and the file's four-space indentation, matched byte for byte. Six misuses exit 2; removing an unknown name exits 1. `TestMCPServerCompletion`: `mcp remove <TAB>` offers `fx` and `web` and creates nothing.
  * `mcpclient`'s own tests cover 21 cases of Pi's validation messages, `Load`'s per-entry errors, `Resolve`'s template rules, the result conversion, the User-Agent and headers on HTTP requests, the stderr tail, and a disabled server never started.
* **Gates.**
  * `make preflight` printed `preflight passed` on Windows and in WSL. WSL `go test -race ./...` exited 0.
  * Windows `go test ./...` showed no `FAIL`.
  * `golangci-lint` printed `0 issues.` for linux, darwin and windows.
  * The pre-add check was clean for all 28 Go files.
* **Negative tests,** each on a scratch copy, each failing at its own line:
  * `http` not advertised: `agent_test.go:109: MCP over streamable HTTP is implemented (Phase 5) and must be advertised`;
  * the file's entry winning the union: `mcp_test.go:179: request tools ["read" "write" "edit" "bash"], want the session's web`;
  * SSE accepted: `mcp_test.go:90: exit 1, stdout "old: failed (direct)\n  https://example.com/sse\n  calling \"initialize\": sending \"initialize\": Method Not Allowed\n", want "config error: <tmp>/mcp.json: server \"old\": legacy SSE transport …"`;
  * `list` exiting 0 on a failed server: `mcp_test.go:74: with a failed server: exit 0`;
  * names not sanitized: `value_test.go:61: ToolName("fx", "dotted.name", taken=false) = "mcp__fx__dotted.name" (20), want "mcp__fx__dotted_name"`;
  * `${NAME}` not expanded: `value_test.go:28: Resolve("Bearer ${TOKEN}") = "Bearer ${TOKEN}", <nil>; want "Bearer t0k"`, and the bearer server then failed with `Unauthorized`;
  * the timeout not applied: `manager_test.go:130: sleep = {Output:"slept" IsError:false …}`;
  * `--no-tools` leaving MCP on: `mcp_test.go:241: [--no-tools]: ["mcp__fx__add" "mcp__fx__dotted_name" "mcp__fx__echo" "mcp__fx__sleep"]`.
* **What the plan predicted wrongly.**
  1. **Three files outside the list** (deviations 1 and 2), and three listed files needing no change (above).
  2. **`//nolint` is not enough for `make preflight`** (deviation 3): its standalone staticcheck reads only `//lint:ignore`, and golangci-lint reads only `//nolint`.
  3. **`Start` takes a context.** contextcheck required one; the connections keep its values but not its cancellation, so a session's servers outlive the request that started them.
  4. **Closing servers never fails a request.** A stdio server's exit status on close is logged at Debug, not returned from `session/close` or `Close`.
  5. **The fixture's tools come back sorted by name** (`add`, `dotted.name`, `echo`, `sleep`): the go-sdk server lists them so, and gobble keeps the server's order.
  6. **The "SSE accepted" negative needed two edits.** With the SSE check alone removed, `type: "sse"` still matches neither branch and fails with "needs either …", so the mutation also routed it to HTTP.
  7. **Kong's `passthrough` on the command argument** takes the command with or without `--`, as Pi's parser does, so `gobble mcp add fs npx -y srv` works too.

**2026-10-06 — Phase 6 made executable, in two parts (owner's decisions: two staged parts, Pi's manual compaction now, fork and clone switch the CLI through `_meta`, an `exit_plan_mode` tool now).**

Phase 6 carries the work five records assign to it:
* its own steps 1–6;
* the 2026-10-01 amendment: `available_commands_update`, `current_mode_update` and a first `usage_update` before `session/new` and `session/load` return; unknown `_` methods answer -32601;
* the 0008 amendment: D13's registry union, D19 item 7's command names, and D16's permission prompt;
* Phase 2's deferral of slash and `@path` completion and of `/new`, `/help` and `/edit`;
* Phase 4's deferral of `session_info_update` for names.

The facts come from Pi at `maccavelli/pi` `312184edb`.

Read in full:
* `packages/coding-agent/docs/compaction.md` and `docs/slash-commands.md`;
* `packages/coding-agent/src/core/compaction/compaction.ts` and `compaction/utils.ts`;
* `packages/coding-agent/src/core/messages.ts` and `core/slash-commands.ts`;
* `packages/coding-agent/src/core/session-manager.ts` (read in full for Phase 4).

Read in part:
* `core/agent-session.ts`: `compact()` (`:2679-2824`), `setSessionName` (`:3844-3849`), `getUserMessagesForForking`, `getSessionStats` and `getContextUsage` (`:4063-4180`);
* `modes/interactive/interactive-mode.ts`: the command dispatch (`:3100-3225`), and `handleThinkingCommand`, `handleCloneCommand`, `handleNameCommand`, `handleSessionCommand` and `handleCompactCommand`;
* `packages/ai/src/utils/text.ts`: `contentText`.

The facts:
* **Manual `/compact` is automatic compaction's preparation** (`agent-session.ts:2679-2703`).
  * `prepareCompaction(path, settings)` returns nothing when the path ends in a compaction, which fails `Already compacted`, or when there is nothing to summarise, which fails `Nothing to compact (session too small)`.
  * The defaults are `reserveTokens` 16384 and `keepRecentTokens` 20000 (`compaction.ts:126-130`).
* **Context with a compaction** (`session-manager.ts:461-512`). The newest compaction on the path becomes one user message, `The conversation history before this point was compacted into the following summary:\n\n<summary>\n` + summary + `\n</summary>` (`messages.ts:11-17,176-183`). It is followed by the path's entries from `firstKeptEntryId` up to the compaction, then every entry after it.
* **Token estimate** (`compaction.ts:276-349`): characters / 4, rounded up. JavaScript characters are UTF-16 code units. An image counts 4800 characters, and a tool call counts its name plus its JSON arguments.
  * `tokensBefore` is the last valid assistant usage after the latest compaction, `totalTokens || input + output + cacheRead + cacheWrite`, plus estimates of the messages after it. With no such usage, it is the sum of estimates (`:196-262`).
* **Cut point** (`compaction.ts:802-870`).
  * Walk back from the newest entry, adding estimates, until the total reaches `keepRecentTokens`. Then cut at the first valid point at or after that entry, else the last valid point. A valid point is an entry with a user, assistant, bash, custom, branch-summary or compaction-summary message, never a tool result. Without reaching the budget, the cut is the first valid point.
  * Then step back over context-invisible entries.
  * A cut at a non-turn-start entry is a split turn, whose turn start is the nearest turn-start entry before it.
* **Summary** (`compaction.ts:507-579,645-766,942-1119`; `utils.ts`).
  * The conversation is serialised as `[User]:`, `[Assistant thinking]:`, `[Assistant]:`, `[Assistant tool calls]: name(k=json, …); …` and `[Tool result]:` lines, joined by blank lines. Tool results are cut at 2000 characters with `\n\n[... N more characters truncated]`.
  * It is wrapped as `<conversation>…</conversation>`, then `<previous-summary>…</previous-summary>` when one exists, then the first-summary or update prompt, with `\n\nAdditional focus: <instructions>`.
  * The system prompt is `SUMMARIZATION_SYSTEM_PROMPT`.
  * A split turn adds a turn-prefix summary, joined as `\n\n---\n\n**Turn Context (split turn):**\n\n`.
  * `<read-files>` and `<modified-files>` lists are appended. They come from `read`, `write` and `edit` calls, merged with the previous compaction's `details`.
  * Failures are `Summarization failed: <message>`, `… generation hit the token cap and the summary is incomplete`, and `Summarization attempted to call a tool`.
* **The entry** (`session-manager.ts:91-104,1260-1284`): `{type:"compaction", id, parentId, timestamp, summary, firstKeptEntryId, tokensBefore, details:{readFiles, modifiedFiles}, usage, fromHook}`.
* **Commands** (`interactive-mode.ts`):
  * A command is the whole editor text `/name` or `/name <args>`.
  * `/name` with no argument shows `Session name: <name>`, or warns `Usage: /name <name>`. With one, it appends `session_info` and shows `Session name set: <name>`.
  * `/session` shows a Session Info block: Name, File, ID; then Messages (Total, User, Assistant, Tools `N calls, M results`); then Tokens (Input, Output, Total); then Cost.
  * `/fork` forks before a chosen user message; the candidates are every user message entry with text.
  * `/clone` duplicates at the leaf, or shows `Nothing to clone yet`.
  * `/thinking <level>` rejects an unknown level with `Unknown thinking level "<x>". Available levels: …`.
* **`contentText`** joins text blocks with `"\n"`, or the separator given. The serialiser passes `""` for user and tool-result content.

The owner decided four questions on 2026-10-06:

1. **Two parts,** each with its own gates, negative tests and execution entry, each committed by the owner.
   * Part A is the agent.
   * Part B is the CLI.
2. **`/compact` is Pi's manual compaction, ported now.**
   * It includes the cut point at 20000 recent tokens, the split turn, the iterative update prompt, the file lists, and Pi's prompts word for word.
   * 0005-PLAN F5 keeps the automatic trigger, overflow compact-and-retry, and the agent-level retry.
3. **`/fork` and `/clone` write a new stored session, report its id, and set `_meta.gobble.switchTo` on the prompt response.**
   * gobble's line session then resumes the new session, as Pi's terminal switches to it.
   * Other ACP clients ignore `_meta` and keep the current session.
4. **Plan mode comes with an `exit_plan_mode` tool,** which carries the plan through `session/request_permission` and publishes it as an ACP `plan` update.
   * 0005-PLAN F6 keeps `accept-edits`, `bypass` and the rule engine.

Choices made within the wording:
* **What counts as a command.** A command is the prompt's text, trimmed, beginning `/`, where everything up to the first whitespace matches D19 item 7's `[A-Za-z0-9][A-Za-z0-9_-]*`.
  * `/usr/local/bin is wrong` and `/skill:review` are therefore prompt text for the model.
  * A command that is not advertised gets `Unknown command /<name>. /help lists the commands.` It is not a model turn.
  * A command writes no user message, and makes no model call except `/compact`'s summary.
* **The advertised set,** frozen here (step 6): `help`, `compact`, `usage`, `context`, `session`, `model`, `thinking`, `mode`, `plan`, `name`, `fork` and `clone`.
  * Step 3 asks for "at least" its list. `mode` and `plan` are added because modes land in this phase, and 0005-MADR lists both as 1.0.
  * `/session` is `/context`'s non-canonical alias (0002-MADR).
  * TUI chrome is never advertised.
* **Frame order.** `session/new`, `session/load` and `session/resume` send `available_commands_update`, `current_mode_update` and a `usage_update` before they respond. `session/load` sends them after its replay. Resume is included because it starts a live session too.
* **Usage.** `used` is the estimate of the system prompt plus the context. `size` stays 0 until the catalog (0005-PLAN F4).
  * `/usage` sums the session's assistant and compaction usage, as Pi's `getSessionStats` does. It says the figures are estimated, and that account and rate-limit figures are not reported by the provider.
  * Cost shows `$0.000` until the catalog.
* **`/context`** is Pi's Session Info, as plain text, with `Mode` and `Model` lines, and a `Context:` line of the same `used` as the last `usage_update`.
* **`/model`, `/thinking`, `/mode` and `/name`.** Each shares its path with the ACP method that does the same: `set_config_option`, `set_mode`, and the session-info entry.
  * Each sends the update that reports the change (`config_option_update`, `current_mode_update`, `session_info_update`) after the entry is written (D19 item 6).
  * With no argument, each shows the current value and the choices.
* **`/fork` and `/clone`.**
  * With no argument, `/fork` lists the user messages as `<entry id>  <first line, at most 60 runes>`, under `Send /fork <entry> to start a new session before one of these messages:`.
  * `/fork <entry>` copies the path up to that message's parent. When the message is the first one, it answers `Nothing comes before that message; start a new session instead.`: files are created lazily, so an empty session cannot be switched to.
  * `/clone` copies the whole path.
  * The copy is written through `store.Create`, with `parentSession` set, then closed so its lock is free. The answer is `Forked to session <id>.` or `Cloned to session <id>.`.
* **Compaction in gobble.**
  * The summary request carries no output cap, because `llm.Request` has no `maxTokens` until 0005-PLAN F4. Pi caps it at 80 % of `reserveTokens`.
  * `tokensBefore` counts no system message, because gobble writes none into the session.
  * Pi's advance over `context_edit` omissions is not ported: gobble has no `context_edit` entries before 0005-MADR's 1.x context edits.
  * Entry types gobble does not act on (`custom_message`, `branch_summary`) are context-invisible, as Phase 4 decided.
* **The progress of `/compact`.**
  * It streams `Compacting the conversation…`, then `Compacted from N tokens.` and the summary, then a `usage_update` with the new estimate.
  * A failure is `Compaction failed: <message>`, or Pi's own `Already compacted` or `Nothing to compact (session too small)`.
  * With no credential, the prompt fails with `auth_required`, as any prompt does.
* **Replay of a compacted session.** `session/load` replays the compaction as one `agent_message_chunk`, `Compacted from N tokens:\n\n<summary>`, then the kept and later messages. The `resumed … (N messages)` count still counts the whole path.
* **Modes.** `default` ("Ask before changing files or running commands.") and `plan` ("Read-only: explore and plan, then leave plan mode with an approved plan.") are advertised in every session response.
  * The mode is not persisted: a loaded session starts in `default`. Pi has no mode entry.
  * An unknown mode id is invalid params.
* **Plan mode.**
  * A non-read-only tool call is refused without asking, with `Plan mode is read-only, so <tool> did not run. Call exit_plan_mode with the plan to ask the user to leave plan mode.`
  * `permission.Decision` gains `Refusal`, the text the model reads when a policy refuses without asking. `agent` uses it as the whole result.
  * `exit_plan_mode {plan}` is offered only in plan mode, and is marked read-only because it asks for itself. It does three things:
    * publishes the plan as a `plan` update: one pending, medium-priority entry per non-empty line, with list markers removed;
    * sends `session/request_permission` for its own call, titled `Leave plan mode and carry out the plan`, with the plan cut to 400 bytes as the first content block, and options `Start implementing` (`allow_once`) and `Keep planning` (`reject_once`);
    * on allow, sets the mode to `default` and sends `current_mode_update`.
* **Unknown `_` methods.** The SDK already answers -32601. A test pins it.

**Part A** (files):
* `command/doc.go`, and new `command.go` and `command_test.go`;
* `session/session.go` and `session_test.go`;
* `compaction/doc.go`, and new `compaction.go`, `estimate.go`, `serialize.go` and `compaction_test.go`;
* `permission/permission.go`;
* `agent/agent.go` and `agent_test.go`;
* `acpserver/`:
  * `agent.go`, `history.go`, `config.go`, `prompt.go`, `agent_test.go`, `session_test.go` and `testdata/session.golden`;
  * new `commands.go`, `modes.go`, `compact.go`, `commands_test.go`, `modes_test.go` and `compact_test.go`.

1. **`command`:**
   * `type Command struct{Name, Description, Hint string}`;
   * `Parse(text string) (name, args string, ok bool)`, by the rule above;
   * `ValidName(string) bool`.
2. **`session`:**
   * `TypeCompaction = "compaction"`.
   * `Entry` gains `Summary`, `FirstKeptEntryID` (`firstKeptEntryId`), `TokensBefore *int64`, `Details jsontext.Value`, `Usage *Usage` and `FromHook *bool`, in Pi's names.
   * A test round-trips a Pi-shaped compaction entry.
3. **`compaction`:**
   * `Settings{ReserveTokens, KeepRecentTokens int}` and `DefaultSettings`.
   * `Context(path []session.Entry) []session.Entry`.
   * `Estimate(m *session.Message) int`, and `EstimateEntries`.
   * `Prepare(path, Settings) (*Preparation, error)`, with `ErrAlreadyCompacted` and `ErrNothingToCompact` carrying Pi's texts.
   * `Compact(ctx, *Preparation, Model{Provider llm.Provider; Model, Thinking string}, instructions string) (Result, error)`. `Result` is `{Summary, FirstKeptEntryID string; TokensBefore int64; Usage llm.Usage; Details}`.
   * `SummaryText(summary string) string`, the context message.
   * Pi's prompts and serialisation, word for word.
4. **`permission` and `agent`:** `Decision.Refusal`, used verbatim by `agent.approve`.
5. **`acpserver/history.go`:**
   * `replayState` builds the history from `compaction.Context`, with the compaction as a user message of `compaction.SummaryText`.
   * `state` keeps the path's messages for the count, and the context for replay.
   * `replayUpdates` renders a compaction as above.
6. **`acpserver/modes.go`:**
   * the two modes, and `liveSession.mode`;
   * `SetSessionMode`;
   * the plan-mode policy around `acpPolicy`;
   * the `exit_plan_mode` tool, built per prompt.
7. **`acpserver/commands.go`:**
   * the frozen registry, `availableCommands()`, and the dispatch in `Prompt` before anything is written;
   * the handlers above.
   * The session-start frames come from one `startFrames` function, called by `NewSession`, `LoadSession` and `ResumeSession` before they respond. Each response carries `Modes`.
8. **`acpserver/compact.go`:** the `/compact` handler. It sets `s.cancel` so `session/cancel` stops the summary, and writes the entry only on success.
9. **`acpserver/config.go`:** `setConfig(ctx, s, id, value)` is shared by `SetSessionConfigOption`, `/model` and `/thinking`, and sends `config_option_update`.

**Part A accept,** through `acptest.Connect` and a scripted model:
* **The phase's own accept.**
  * `/compact`, `/usage` and `/context` sent as `session/prompt` give non-empty `agent_message_chunk`s, and make no model request except the summary.
  * After `/compact`, the next prompt's request starts with the summary message, then only the kept messages (with `KeepRecentTokens` set small in the test).
  * `set_mode` to `plan` then `default` gives one `current_mode_update` each.
* **Every advertised command executes.** A table built from `availableCommands()` itself sends each one and asserts a handled answer, never `Unknown command` and never a model turn.
* **Plan mode.** The model's `write` is refused with the plan-mode text, and no file appears. `exit_plan_mode` publishes the `plan` update and asks; on allow, the mode is `default` and a later `write` asks as usual.
* **Frame order.** `available_commands_update`, `current_mode_update` and `usage_update` precede the `session/new` response on the wire (the recorder's transcript), and likewise for `session/load`.
* **The stdio script** of the fifth amendment's Confirmation completes without an unknown-method error: `initialize`, `session/new`, `session/prompt` `/compact`, `session/set_mode` `plan`, and `session/set_config_option` with the model's `configId`. Raw `session/set_model` and `_x.ai/compact_conversation` get -32601.
* **No `_x.ai/` in the module.** A walk of the module's `.go` files finds no `_x.ai/` string. The test builds its needle from two parts, so it does not find itself.
* **Compaction against Pi.**
  * A property test over seeded random sessions never puts `firstKeptEntryId` on a tool result.
  * A constructed session gives the expected cut, and a split turn.
  * The serialisation matches a fixture written from Pi's code.
  * A Pi-shaped compaction entry loads, and becomes the summary message in context.
* **Sessions.**
  * `/fork <entry>` and `/clone` write a new session with the expected entries and `parentSession`, and the response carries `_meta.gobble.switchTo`.
  * `/name x` writes `session_info` and sends `session_info_update` with title `x`.
  * The golden transcript changes by the new frames and `modes` only.

Each is shown failing first on a scratch copy, the failure quoted:
* `Prompt` ignoring a leading slash, the grok silence shape;
* an advertised `/bogus` with no handler;
* the session-start frames sent after the response;
* a cut allowed on a tool result;
* plan mode letting `write` run;
* the compaction left out of the next prompt's context.

**Part B** (files):
* `acpclient/`: `conn.go`, `update.go`, `sessions.go`, and their tests;
* `internal/cli/`: `line.go`, `sessions.go`, `chat.go` and `line_test.go`, and new `slash.go`, `permission.go`, `slash_test.go` and `permission_test.go`.

1. **`acpclient`:**
   * `Update` gains the kinds `Commands` (with `Commands []Command{Name, Description, Hint}`), `Mode` (`ModeID`) and `SessionInfo` (`Title`).
   * `Session.Commands()` returns the latest list.
   * `Result.SwitchTo` comes from the response's `_meta.gobble.switchTo`.
   * `Options.Permission func(ctx, PermissionRequest) PermissionAnswer` answers `session/request_permission`. Without it, the client declines as it does today.
2. **`internal/cli/slash.go`:** the D13 registry, the agent's commands united with the local `/help`, `/exit`, `/quit`, `/edit` and `/new`.
   * `/help` prints both lists.
   * `/new` closes the session and opens a new one in process.
   * `/edit [text]` composes the prompt through `editor.Compose`.
   * A valid name outside the registry prints `Error: unknown command /<name>; /help lists the commands` on stderr, and is never sent.
   * Before the first session exists, a slash input opens the session first. `session/new` makes no model call, so D5 holds.
3. **`internal/cli/line.go`:**
   * completion through `editor.Options.Complete`: slash names from the registry (only local ones before a session exists, D19 item 9), and `@path` from `os.ReadDir`;
   * after a result with `SwitchTo`, the session closes and resumes the new one, printing the `resumed …` line.
4. **`internal/cli/permission.go`:** D16's prompt.
   * One line on stderr, built from the options offered: `[y] allow once`, `[a] always`, `[n] deny`, `[N] never`, and `[c] cancel` always, each shown only when offered.
   * The title is sanitised (D8).
   * One key is read from the line session's reader. Ctrl+C means cancel.
   * Print mode and a non-terminal stdin keep declining, until F6's `--yes`.

**Part B accept,** through `Main` and the line session's scripted input:
* `/help` lists both sets.
* `/bogus` errors locally, and the agent receives no prompt.
* `/new` starts a new session.
* `/edit` returns the fake editor's text as the prompt.
* `Tab` after `/co` completes from the registry.
* `y` lets a `write` run, `n` refuses it, and Ctrl+C cancels.
* `/clone` switches the line session to the clone, and the next prompt reaches it.

Shown failing first:
* an unknown slash sent to the agent;
* `SwitchTo` ignored;
* `n` treated as allow.

**Verification, each part.**
* `go test ./...`;
* `golangci-lint` for linux, darwin and windows;
* `make preflight` on Windows and in WSL, and WSL `go test -race ./...`;
* the pre-add check on every changed Go file;
* the identifier and hidden-character scans.

The agent stages; the owner commits.

**Deferred from Phase 6, named:**
* to 0005-PLAN F5: automatic compaction (the threshold and its triggers), overflow compact-and-retry, and agent-level retry;
* to F6: `accept-edits` and `bypass`, the rule engine, `allow_always` rules, `/permissions`, `/trust` and `--yes`;
* to F4: the context window `size`, prices and the summary's output cap;
* to Phase 7: `_gobble/compact`, `_gobble/usage`, `_gobble/set_session_name` and the read helpers (the dual path);
* to 0005-PLAN X1: `/deep-research`;
* to 1.x: `/status`, `/todos`, `/tree`, branch summaries and context edits.

**Deviations during Phase 6, 2026-10-06 (owner's decisions):**

1. **`acpserver/prompt_test.go` joins Part A's files.**
   * `TestOnePromptAtATime` failed on this tree with `second prompt err = <nil>, want invalid request`.
   * It waited for any update to know the first prompt was running. `session/new`'s three start frames now satisfy that wait at once, so the second prompt raced ahead of the first.
   * Its wait now waits for the first prompt's tool call. The assertion, -32600 for a second prompt, is unchanged.

**Phase 6 Part A, 2026-10-06 — complete (staged; the owner commits).** It ran as the amendment wrote it, with deviation 1 above. Part B and Phases 7–8 have not run.

* **Files.**
  * `command`: `doc.go`, and new `command.go` and `command_test.go`.
  * `session`: `session.go` and `session_test.go`.
  * `compaction`: `doc.go`, and new `compaction.go`, `estimate.go`, `serialize.go` and `compaction_test.go`.
  * `permission/permission.go`, and `agent/agent.go` and `agent_test.go`.
  * `acpserver`: `agent.go`, `history.go`, `config.go`, `prompt.go`, `agent_test.go`, `session_test.go`, `prompt_test.go` (deviation 1) and `testdata/session.golden`, and new `commands.go`, `modes.go`, `compact.go`, `commands_test.go`, `modes_test.go` and `compact_test.go`.
* **Accept,** through `acptest.Connect` and a scripted model:
  * `TestEveryAdvertisedCommandExecutes`: every name of the session's `available_commands_update`, read from the wire, sent as a prompt, is answered, never `Unknown command`, with no model request.
  * `TestCommandsAnswer`: `/usage` (`Session usage`), `/context` (`Session Info`), `/session` (`Context: `), `/compact` on a small session (`Compaction failed: Nothing to compact (session too small)`), `/help` and `/bogus`. One model request in all. `/usr/local/bin is missing` reaches the model.
  * `TestCompactShortensContext`: `/compact keep the API` streams `Compacting the conversation…`, then `Compacted from N tokens.` with the summary, then a `usage_update`. The summary request carries `Additional focus: keep the API`. The next request holds the summary message, then only the kept messages.
  * `TestCompactionLoads`: a compaction in Pi's shape loads, replays as `Compacted from 42 tokens:`, and the next request starts from its summary and the kept entry.
  * `TestSetModeUpdates`: `default` and `plan` are advertised. `set_mode` to `plan` then `default` gives `current_mode_update`s `default,plan,default`, the first from `session/new`. An unknown mode is -32602.
  * `TestPlanModeRefusesWrite`: `/plan`, then the model's `write` is refused with the plan-mode text, nothing is asked, and no file appears. `exit_plan_mode` is offered.
  * `TestExitPlanMode`: the plan becomes three `plan` entries with their markers removed. The permission request is for the tool's own call, carrying the plan. Allow returns to `default`, and the next `write` asks and runs.
  * `TestStartFramesPrecedeTheResponse`: the three frames precede the `session/new` and `session/load` responses on the recorder's wire.
  * `TestMcremoteScript`: the acpagent script over `Serve` has no error, and `session/set_model` and the x.ai method answer -32601. `TestNoXAIMethods` finds no x.ai string in the module's `.go` files.
  * `TestNameCommand`, `TestModelAndThinkingCommands` and `TestForkAndClone`:
    * `/name` writes the entry and sends `session_info_update`.
    * `/thinking` and `/model` change the next request and send `config_option_update`.
    * `/fork` lists the user messages, and `/fork <entry>` writes a session holding only `first`. `/clone` writes `first,second`. Each response's `_meta.gobble.switchTo` names the new session, and the fork resumes. Forking the first message is refused.
  * `compaction`'s own tests:
    * a cut at a turn, and a split turn;
    * `Already compacted` and `Nothing to compact`;
    * 500 seeded random sessions with no cut on a tool result;
    * the context, and the serialisation, against a fixture written by hand from Pi's code;
    * the request and the file lists, the iterative update, the split-turn merge, and the three summary failures.
  * The golden transcript changed by the three frames before the `session/new` response, and `modes` in it, only.
* **Gates.**
  * `make preflight` printed `preflight passed` on Windows and in WSL. WSL `go test -race ./...` exited 0.
  * Windows `go test ./...` showed no `FAIL`.
  * `golangci-lint` printed `0 issues.` for linux, darwin and windows.
  * The pre-add check was clean for all 26 Go files.
* **Negative tests,** each on a scratch copy, each failing at its own line:
  * `Prompt` ignoring a leading slash: `commands_test.go:67: /help: {"code":-32603,"message":"Internal error","data":{"error":"llmtest: script exhausted"}}`;
  * an advertised `/bogus` with no handler: `commands_test.go:69: /bogus: stop end_turn, text "Unknown command /bogus. /help lists the commands."`;
  * the frames after the response: `commands_test.go:123: "method":"session/new" answered after only map[]`;
  * a cut on a tool result: `compaction_test.go:127: session 0: cut at tool result e0000024`;
  * plan mode letting `write` run: `modes_test.go:66: plan mode let write run`;
  * the compaction left out of the next context: `compact_test.go:93: context after compact: 5 messages, …`.
* **What the plan predicted wrongly.**
  1. **One test file outside the list** (deviation 1).
  2. **A failed `/compact` always reads `Compaction failed: <message>`,** with Pi's own messages after the colon. That is how Pi's `compaction_end` reports them. The amendment had them bare.
  3. **`exit_plan_mode` is its own `tool.Tool`.** `tool.New`'s function never sees the call, and the permission request must carry the call's id. Its spec still comes from `tool.New`.
  4. **Phase 5's staticcheck resolution applies again.** `compaction`'s `ErrAlreadyCompacted` and `ErrNothingToCompact` carry Pi's capitalised texts, so each has `//lint:ignore ST1005` beside its `//nolint`, as deviation 3 of Phase 5 decided for its own two lines.
  5. **A command's failure is its answer.** `cmdTurn.failed` says it and ends the turn normally, which is also how `nilerr` reads it.
  6. **Model and provider come from the whole path, history from the context.** Before Phase 6, `addMessage` set both as it went. With a compaction in the path, the summarised messages still name the model.
  7. **`session/load`'s usage frame is now the estimate** (system prompt plus context), as every session start sends it. Before, it was the last assistant's tokens.
  8. **Three of my own test mistakes,** each caught before the gates:
     * an iterative-update fixture whose cut fell on an assistant message, which Pi's rule splits;
     * a raw-frame matcher that skipped the -32601 answer, because its error data names the method;
     * `TestNoXAIMethods` finding the x.ai literal in its neighbour, now built from parts.

**Deviation before Part B's code, 2026-10-06 (owner's decision):**

2. **`internal/cli/agent.go` joins Part B's files.**
   * D16's prompt must reach `acpclient.Options.Permission`, which `startAgent` builds (`agent.go:98-111`). Print mode calls `startAgent` too (`print.go:131`), and keeps declining.
   * `startAgent(ctx)` becomes `startAgentWith(ctx, permission)` and a one-line `startAgent` that passes nil, so `print.go` is unchanged. The line session calls `startAgentWith` with its prompt.

**Phase 6 Part B, 2026-10-06 — complete (staged; the owner commits). Phase 6 is complete.** It ran as the amendment wrote it, with deviation 2 above. Phases 7–8 have not run.

* **Files.**
  * `acpclient`: `conn.go`, `update.go` and `sessions.go`, with `conn_test.go` and `sessions_test.go`.
  * `internal/cli`: `agent.go` (deviation 2), `line.go` and `line_test.go`, and new `slash.go`, `permission.go`, `slash_test.go` and `permission_test.go`.
  * Listed but unchanged: `internal/cli/sessions.go` and `chat.go`. The session code they hold already did what the line session needed (`openSession`, `resumedLine`).
* **Accept,** through the line session's scripted input, over a scripted agent or over gobble's own agent on a scripted model:
  * `TestSlashHelpUnion`: `/help` lists the local commands (`/new`, `/edit [text]`) and the agent's (`/compact [instructions]`), `/help` once, with no model call.
  * `TestSlashUnknownStaysLocal`: `/bogus now` prints `Error: unknown command /bogus; /help lists the commands`, and the agent receives no prompt.
  * `TestSlashNew`: `/new` closes the session, and the next prompt opens another: `initialize,session/new,session/prompt,session/close,session/new,session/prompt,session/close`.
  * `TestSlashEdit`: `/edit draft it` starts the editor on `draft it`, and its text is the prompt.
  * `TestComplete`: `/ne` completes to `/new `, `/e` stays at the common prefix, `@notes` to `@notes.md `, `@no` to `@not`, and `@s` to `@sub/`.
  * `TestCloneSwitches`: `/cl` and Tab complete to the agent's `/clone`. The line session prints `resumed first (2 messages)` and the next prompt goes to the clone, which alone holds it, with its history.
  * `TestPermissionPrompt`: the line reads `write a.txt?  [y] allow once  [n] deny  [c] cancel`. A key not shown (`z`) is ignored. Then `y` lets the write run, `n` refuses it, and Ctrl+C cancels it, each echoing its answer. `TestPermissionLine` checks the sanitised title and the keys of the options offered.
  * `acpclient`: `TestTranslate` covers the three new kinds, and `Session.Commands`. `TestRequestPermission` covers the mapping, an answer naming no option cancelling, and the old declining without `Permission`.
* **Gates.**
  * `make preflight` printed `preflight passed` on Windows and in WSL. WSL `go test -race ./...` exited 0.
  * Windows `go test ./...` showed no `FAIL`.
  * `golangci-lint` printed `0 issues.` for linux, darwin and windows.
  * The pre-add check was clean for all 12 Go files.
* **Negative tests,** each on a scratch copy, each failing at its own line:
  * an unknown slash sent to the agent: `slash_test.go:47: condition not met within 5 s`, then `slash_test.go:53: stderr "… Error: session/prompt: …"`;
  * `SwitchTo` ignored: `slash_test.go:146: condition not met within 5 s`;
  * `n` treated as allow: `permission_test.go:56: key "n": file written true`.
* **What the plan predicted wrongly.**
  1. **One file outside the list** (deviation 2), and two listed files needing no change.
  2. **`/help` opens the session, as any agent command does.** Its list then includes the agent's commands, and `session/new` makes no model call.
  3. **The permission prompt answers with a key, not a line.** It reads from the editor's own reader in raw mode, which no one else reads during a turn. The answer is echoed after the line.
  4. **`/new` resets the session flags.** `--session-id` and `-n` named the first session only.
