---
status: proposed
date: 2026-09-30
associated-madr: "0002-MADR-cli-acp-headless-mcp-v1.md"
---
# Implement v1 as the native magic-cli-remote CLI: Cobra over ACP, ACP stdio, MCP client

Associated MADR: [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md)

## Goal

A Go module in this repository that builds one binary. That binary (1)
serves ACP protocol version 1 on stdio as the agent process magic-cli-remote
already knows how to spawn, (2) exposes the same agent methods through
Cobra, (3) connects to MCP servers over stdio and streamable HTTP so
their tools are callable during a prompt, and (4) executes agent-owned
slash commands (`/compact`, `/usage`, `/context`, `/model`, `/thinking`,
and the native set in the MADR) over `session/prompt`, with
`available_commands_update` listing only those handlers.

## Scope

In:

* `go.mod`, Cobra, Viper, `github.com/coder/acp-go-sdk v0.13.5`
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
* `_pigo/` extension methods listed in the MADR mapping table that the phase
  claims (`_pigo/compact`, `_pigo/usage`, `_pigo/steer`, `_pigo/follow_up`,
  `_pigo/clear_queue`, and the read/name helpers the phase freezes)
* MCP client: stdio + streamable HTTP; `pigo mcp add|remove|list`
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
* D11, D17 — [0003-MADR](0003-MADR-pigo-product-identity.md).

The version of `go-llmprovider-sdk` and of
`github.com/modelcontextprotocol/go-sdk` is recorded as a dated note in
the phase that first imports them. That is not a new architectural
choice.

## Implementation Steps

### Phase 0 — module and Cobra skeleton

> *Superseded (2026-09-29) by [0004-PLAN](0004-PLAN-go-module-architecture.md).* That plan builds the module, package
> skeleton, Cobra root, CI, and release pipeline to [0004-MADR](0004-MADR-go-module-architecture.md). The steps below
> are kept for history; Phase 1 starts from the tree 0004-PLAN leaves.

1. `go mod init` with the module path chosen for D17 (placeholder
   `github.com/maccavelli/pi-go` unless a MADR says otherwise).
2. `cmd/` main → `internal/cmd` Cobra root. Subcommands: `acp` (stdio
   agent), `prompt`, `mcp`, `auth` (auth may stub).
3. Viper reads JSON; file path is a constant until D11 is decided.
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
3. Wire `pigo acp` to
   `acp.NewAgentSideConnection(agent, os.Stdout, os.Stdin)`.
4. In-process test: `ClientSideConnection` over pipes, using the SDK,
   runs Initialize → NewSession → Prompt → CloseSession.

**Accept:** that test is red on a deliberately broken `Initialize`
(wrong protocol version) before it is green on the real agent. The
failure text is quoted in the phase handoff. `pigo acp` + SDK example
client (or the in-tree client test) completes a prompt.

### Phase 2 — Cobra is an ACP client

1. Shared constructor: agent + pair of connections over `io.Pipe` (or
   equivalent), used by Cobra and by tests.
2. `pigo prompt "…"` is `session/new` + `session/prompt` + print of
   agent text chunks to stdout, then `session/close`.
3. A test that the same `Prompt` params are sent whether the caller is
   the Cobra command or a stdio client (compare JSON-RPC method name
   `session/prompt`).

**Accept:** `pigo prompt` does not import or call the agent loop type; it
calls ACP. A grep of `internal/cmd` for the agent-loop package fails
(only `internal/acp` / the connection package is allowed). *(2026-09-29:
per [0004-MADR](0004-MADR-go-module-architecture.md) the packages are
`internal/cli` and `acpclient`, and the check is archtest rule 6 rather
than a grep.)*

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
4. `pigo mcp add|remove|list`. `list` connects and prints tools; exit 1 on
   failure, matching TypeScript Pi.
5. Tool names `mcp__<server>__<tool>`.
6. OAuth: implement if the client library supports it in this phase;
   otherwise record the gap here before merging.

**Accept:** two fixtures — a stdio MCP server in-tree and an HTTP test
server — appear as tools during `session/prompt`. `pigo mcp list` agrees.
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

### Phase 7 — `_pigo/` extensions and remaining Cobra

1. Implement `_pigo/compact` (optional instructions), `_pigo/usage`,
   `_pigo/steer`, `_pigo/follow_up`, `_pigo/clear_queue` against the agent
   queue. Compact via the extension is the same compaction as `/compact`.
2. Implement the read extensions used by the CLI (`_pigo/get_state` or
   whatever names the mapping table freezes in this phase — freeze them
   in this PLAN when the handlers land).
3. `_pigo/set_session_name` matches `/name`.
4. `pigo auth` as far as the chosen provider requires (may be env-only).

**Accept:** SDK `CallExtension` for `_pigo/steer` during a live faux prompt
queues a steer; `_pigo/compact` shortens history the same way `/compact`
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
to match is in
[0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md)
(intended `command.Table`, Spec sketch, Compact hook). Checklist copied
from magic-cli-remote MADR 0023:

1. `acpagent.Spec` with `DefaultBin` `pigo`, `DefaultArgs` `acp`.
2. `command.Table` declaring every canonical Specs name. `KindNone`
   reasons in words a user reads. `compact` stays `KindNative` until
   `(*session).Compact` is a Spec hook (today it always calls
   `x.ai/compact_conversation`).
3. Probe, do not assume: live-tagged tests send `/compact`, `/usage`,
   `/context` and assert output. `/settings` is absent from
   `available_commands`.
4. No grok `ExtensionNotifications`. Empty `CommandCaveat` unless a
   real quirk appears.
5. `KnownGoodVersion` once a pigo release exists; mismatch warns.
6. `OpSetModel` / `OpSetThinkingLevel` go through `SetConfigOption`,
   or the rows stay `KindNative` (`/model`, `/thinking`). Never
   `session/set_model` (MADR second amendment).

Existing providers (grok, opencode, kilo, codex, fake) stay registered.

## Rollout and Rollback

No production users. Rollback is `git revert` of the phase commit.
Do not tag a v1 release until D17 (identity / User-Agent) is decided;
talking to providers with a placeholder name is a live-test-only act.

magic-cli-remote does not gain `IDPi` as a side effect of this plan.

## Execution record

None. This plan is proposed and has not been approved.

## Amendments

**2026-09-29 — native magic-cli-remote CLI.** The first draft of this
plan allowed Phase 6 to ship with an empty slash-command list, mapped
compact only as a `_pigo/` extension, and did not require `usage_update`,
`session/set_mode`, or `session/set_model`. The associated MADR now
requires honest executing slash commands and the ACP methods mcremote
already consumes. Phase 6 is that contract; a former empty-list
allowance is withdrawn. Phase 7 keeps `_pigo/` dual-path. Hardening moves
from Phase 7 to Phase 8. Companion `IDPi` work is documented and kept
out of this tree.

**2026-09-29 — `pigo` identity, SDK routing, wider v1 line.** The binary
is `pigo` and extension methods are `_pigo/…` ([0003-MADR](0003-MADR-pigo-product-identity.md)). Phase 0 is
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
