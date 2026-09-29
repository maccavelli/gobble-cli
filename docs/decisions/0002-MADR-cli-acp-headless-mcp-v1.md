---
status: proposed
date: 2026-09-29
decision-makers: repository owner
consulted: 0001-REPORT-go-port-feasibility.md
informed: go-llmprovider-sdk, acp-go-sdk, magic-cli-remote
---
# v1 is the native magic-cli-remote CLI: Cobra over ACP, ACP stdio, MCP client

## Context and Problem Statement

[0001-REPORT-go-port-feasibility.md](../reports/0001-REPORT-go-port-feasibility.md)
established that a source-faithful port of the whole TypeScript monorepo is
not achievable, and that a product rewrite of the shipping `pi` CLI is.
Eighteen decisions were left open. The owner named the v1 product cut: the
**CLI**, **headless RPC**, and **MCP** capabilities of TypeScript Pi, with
**extensive use of the Agent Client Protocol (ACP)** to expose the agent
command API.

A first draft of this record chose ACP as that command API. The owner then
tightened the target: v1 is designed to be the **native CLI supported by
magic-cli-remote**. Other CLI providers in that daemon continue. pi-go is
fully optimized for, and fully supported by, that stack — leveraging the
existing **mcremote** / **mcrelay** protocols and the canonical slash-command
standard (magic-cli-remote MADR 0023), and exposing agent CLI commands such
as `/compact` and `/usage` **natively over ACP** so they actually execute.
pi-go may be rewritten; TypeScript Pi's JSONL RPC, Charm TUI, and jiti
extensions are not constraints on the shape.

TypeScript Pi's headless control surface is a custom JSONL command protocol
(`pi --mode rpc`, 33 commands in `rpc-commands.md`). ACP is JSON-RPC 2.0
over stdio. The fleet already consumes ACP as a **client**: magic-cli-remote
pins `github.com/coder/acp-go-sdk v0.13.5`, protocol version `1`. Its
daemon (mcremote) speaks a WebSocket protocol to phones and desktops
(protocol-v1, v2 delta in MADR 0068) and spawns agent CLIs as ACP stdio
subprocesses through `internal/provider/acpagent`. TypeScript Pi has no ACP
implementation. The current ACP provider in that daemon is grok; its
`available_commands` catalog is its TUI's, and several advertised commands
return nothing over the protocol (MADR 0023: advertisement is not
capability).

The problem is how v1 is shaped so that (1) the CLI, the headless protocol,
and MCP are one command API, and (2) that API is the agent hop
magic-cli-remote already knows how to drive, with every agent-owned command
honestly advertised and actually executed.

## Decision Drivers

* Owner-stated v1 surfaces: CLI, headless RPC, MCP. Interactive TUI,
  TypeScript extensions, Chord/Pico, and the TypeScript SDK are outside that
  cut.
* Owner-stated integration target: native CLI of magic-cli-remote; other
  providers in that daemon keep working; pi-go is the fully supported,
  fully optimized agent.
* One agent command API, used by Cobra, by editors, and by mcremote, rather
  than a Cobra vocabulary plus a private RPC dialect plus a TUI-only slash
  catalog.
* mcremote/mcrelay already define the phone/desktop hop (WebSocket
  protocol-v1 / v2). pi-go belongs on the **ACP stdio** hop those daemons
  already spawn. It does not reimplement the WebSocket.
* magic-cli-remote MADR 0023: a provider table beats advertisement;
  capabilities are proven by execution and by optional session interfaces,
  not by catalog entries. grok's inert `/compact` is the failure this
  rewrite is free to refuse.
* ACP is the editor-facing standard; the in-house Go SDK and a production
  ACP client already exist in this fleet.
* TypeScript Pi's MCP role is a **client** of stdio and streamable-HTTP
  servers (`mcp.json`, `pi mcp add|list|login|logout`). ACP already carries
  MCP server configs on `session/new` and advertises `mcpCapabilities`.
* Cobra and Viper remain the CLI and configuration stack. Charm/Lip Gloss
  apply when a TUI exists; v1 has none.

## Considered Options

* Native magic-cli-remote CLI: Cobra over one `acp.Agent`, ACP stdio
  headless, MCP client, honest slash commands that execute
* Cobra CLI plus a faithful port of TypeScript Pi JSONL RPC, ACP added later
* Dual headless protocols in v1 (Pi JSONL RPC and ACP)
* ACP-only stdio agent with no Cobra command surface beyond `acp`
* pi-go speaks mcremote WebSocket (protocol-v1/v2) in addition to ACP
* Grok-shaped adapter: advertise a TUI catalog and rely on daemon
  workarounds (`KindNone`, native-name aliases, vendor `_x.ai/*`)

## Decision Outcome

Chosen option: "Native magic-cli-remote CLI: Cobra over one `acp.Agent`,
ACP stdio headless, MCP client, honest slash commands that execute",
because the owner asked for those three surfaces, for ACP to expose the
agent command API, and for pi-go to be the native agent of
magic-cli-remote.

One `acp.Agent` implementation is the command API. The Cobra CLI is an ACP
**client** of that implementation. Headless RPC is ACP JSON-RPC 2.0 on
stdio — the same process role mcremote already spawns. MCP is an agent
capability (stdio and streamable HTTP), configured from `mcp.json` and from
`session/new.mcpServers`. Agent-owned slash commands (`/compact`, `/usage`,
`/context`, `/model`, `/thinking`, and the rest of the native set below)
are handled in `session/prompt` and, where mcremote has an optional
session interface, also as `_pi/` extension methods so either a
`KindNative` or a `KindOp` table mapping works.

pi-go does **not** implement mcremote or mcrelay WebSocket. Those stay in
magic-cli-remote. Other providers there (grok, opencode, kilo, codex, fake)
continue. A future `provider.IDPi` Spec in that repository is companion
work specified here, not executed in this tree.

TypeScript Pi's JSONL `--mode rpc` is **out of v1**. A later MADR may add a
translation shim. Charm TUI, jiti extensions, codemode, Chord/Pico, and a
browser/TypeScript SDK remain out of v1.

### Consequences

* Good, because editors that already speak ACP (Zed and anything using
  `acp-go-sdk`) can spawn this binary without a private protocol.
* Good, because the CLI, an IDE, and mcremote execute the same methods; a
  Cobra command cannot drift from what the phone can call.
* Good, because ACP `session/new` already takes `mcpServers`, so MCP is not
  a side channel bolted onto a custom RPC.
* Good, because v1 drops the 19k-line TUI, the jiti extension host, and the
  experimental Chord stack that the report identified as the largest
  non-portables.
* Good, because slash commands are designed against MADR 0023's rule
  (advertisement is capability): `/compact` over ACP summarises; it does
  not return silence.
* Neutral, because `coder/acp-go-sdk v0.13.5` implements protocol version
  **1** with method names `authenticate` / `logout` / `session/set_mode`.
  The public ACP v2 docs use `auth/login` and `auth/logout`. v1 pins the
  SDK the fleet already runs, not the website's v2 names.
* Neutral, because Pi-only operations that ACP does not baseline (`steer`,
  `follow_up`, `clear_queue`, typed compact with instructions) become
  `_pi/`-prefixed extension methods. Editors that ignore unknown extensions
  still run the baseline and the slash forms.
* Neutral, because registering `IDPi` in magic-cli-remote is a change in
  **that** repository. This record specifies the Spec and command table
  that work must match; it does not land it.
* Bad, because existing TypeScript Pi RPC clients (`RpcClient`, IDE
  integrations written to `rpc-commands.md`) will not speak to this binary
  until a shim exists.
* Bad, because sharing `~/.pi` and `mcp.json` with TypeScript Pi (report
  D11) is still open: this MADR does not choose the config directory or
  product name (report D17).
* Bad, because today's shared `acpagent` session implements grok vendor
  paths (`session/set_model` `_meta.reasoningEffort`,
  `x.ai/compact_conversation`). An `IDPi` Spec that reuses those paths
  unchanged would call methods pi-go does not implement. Companion work
  must parameterize those hooks (see More Information).

### Confirmation

* A stdio round-trip against `github.com/coder/acp-go-sdk` at the pinned
  version: `Initialize`, `NewSession`, `Prompt`, `Cancel`, and
  `CloseSession` succeed with the SDK's example client or an equivalent
  in-tree test.
* Every Cobra command that drives the agent is implemented as a call to
  that same `acp.Agent` (in-process connection), asserted by a test that
  the CLI and a stdio client produce the same ACP method names for the
  same user action.
* `session/new` with a stdio MCP server and with a streamable-HTTP MCP
  server advertises tools to the model; `pi mcp list` reports the same
  servers. SSE is not implemented (TypeScript Pi rejects `sse`; ACP's SSE
  capability stays `false`).
* No package under this module imports a Charm / Lip Gloss / Bubble Tea
  module in v1.
* A mapping table in this record's More Information is kept honest: each
  TypeScript Pi RPC command is either an ACP baseline method, a `_pi/`
  extension, a Cobra config command, or explicitly out of v1.
* `available_commands_update` lists only commands `session/prompt` actually
  executes. A test sends `/compact`, `/usage`, and `/context` as
  `session/prompt` and observes `session/update` frames (not a zero-token
  empty turn). The same test is shown failing first on an echo-only
  `Prompt` (the grok silence shape).
* After a prompt, the agent emits a standard ACP `usage_update`
  (`used`, `size`, optional `cost`) so a daemon `OpContext` mapping can
  resolve from tracked usage.
* `session/set_mode` honors at least `default` and `plan`, each confirmed
  by `current_mode_update`. `session/set_config_option` changes model and
  thinking. `session/set_model` with `{sessionId, modelId}` (the request
  mcremote `acpagent.SetModel` already sends) also changes the model.
* No `_x.ai/` method is implemented or advertised.

## Pros and Cons of the Options

### Native magic-cli-remote CLI: Cobra over one `acp.Agent`, ACP stdio headless, MCP client, honest slash commands that execute

The CLI process always hosts (or is) an ACP agent. Headless mode is
`AgentSideConnection` on stdin/stdout — the process mcremote spawns.
Interactive-ish CLI commands (`prompt`, `session new`, `session resume`)
are `ClientSideConnection` calls, in-process over pipes or to a spawned
stdio child. Slash commands the agent owns execute in `Prompt` and, where
needed for `KindOp`, as `_pi/` extensions. The PLAN picks in-process vs
child transport; this record requires that the bytes are ACP.

* Good, because it matches the owner's ACP, CLI, MCP, and
  magic-cli-remote-native requirements with one command dialect.
* Good, because mcremote already knows how to be an ACP client of this
  shape; the remaining work there is a Spec and an honest command table.
* Bad, because TypeScript Pi RPC clients are not served in v1.

### Cobra CLI plus a faithful port of TypeScript Pi JSONL RPC, ACP added later

Reproduce `--mode rpc` record-for-record, then wrap it.

* Good, because existing Pi RPC clients keep working.
* Bad, because the owner asked for ACP as the command API, and this option
  makes ACP a façade.
* Bad, because JSONL `type: prompt` and JSON-RPC `session/prompt` would
  both exist and diverge.

### Dual headless protocols in v1 (Pi JSONL RPC and ACP)

Ship both from day one.

* Good, because no client is left behind.
* Bad, because v1 then maintains two envelopes, two command inventories,
  and two test matrices for the same agent.
* Neutral, because a later shim can translate JSONL to ACP once the ACP
  API is stable.

### ACP-only stdio agent with no Cobra command surface beyond `acp`

A single `pi` that only speaks ACP on stdio. Users use an editor,
mcremote, or `acp-go-sdk`'s example client.

* Good, because the smallest binary that is an ACP agent.
* Bad, because the owner asked for a CLI, including `pi mcp …`, `pi auth …`,
  and print-style prompting, which TypeScript Pi offers without an editor.

### pi-go speaks mcremote WebSocket (protocol-v1/v2) in addition to ACP

Put the daemon protocol inside the agent binary.

* Good, because a phone could connect without a separate mcremote.
* Bad, because that duplicates mcremote/mcrelay (auth, pairing, resume,
  history ring, canonical command resolution) and splits the fleet's
  WebSocket into two implementations.
* Bad, because the existing architecture is already
  phone → mcrelay → mcremote → ACP stdio → agent; adding a second
  WebSocket server on the agent does not use those protocols, it forks
  them.

### Grok-shaped adapter: advertise a TUI catalog and rely on daemon workarounds

Ship `available_commands` as a superset, mark the ones that do not work
`KindNone` in the daemon table, and copy `_x.ai/*` extensions.

* Good, because it matches how grok is integrated today.
* Bad, because the owner asked for a rewrite that is fully supported, and
  MADR 0023 exists precisely because that shape silently drops `/compact`
  and `/usage`.
* Bad, because `_x.ai/*` is grok vendor; implementing it in pi-go would
  couple the native CLI to another product's extensions.

## More Information

### Amendment (2026-09-29)

This record is still `proposed`. The first draft chose ACP as the sole
agent command API, mapped TypeScript Pi RPC onto baseline ACP plus `_pi/`
extensions, and allowed v1 to ship with an empty slash-command list. The
owner then required native magic-cli-remote support: honest slash commands
that execute over ACP (`/compact`, `/usage`, and the rest of the native
set), integration with mcremote/mcrelay as they exist, and other providers
left in place. The Decision Outcome, Confirmation, mapping tables, and
companion Spec below replace that empty-list allowance. The ACP-as-command-API
choice, the JSONL-RPC-out-of-v1 choice, and the Cobra-is-an-ACP-client
choice are unchanged.

### What v1 is

| Surface | Role |
|---|---|
| Cobra + Viper | Process entry, flags, `mcp` / `auth` / `config` subcommands, print-style `prompt` |
| `acp.Agent` | The agent command API, including slash commands |
| ACP stdio | Headless RPC; the process magic-cli-remote spawns |
| MCP client | stdio and streamable HTTP; tools become ordinary agent tools |

### What v1 is not

Charm TUI, TypeScript/jiti extensions, npm package install of extensions,
codemode, Chord, Pico/`pi-durable`, `pi-protocol` CBOR, HTML export,
browser `pi-ai`, TypeScript SDK, Pi JSONL `--mode rpc`, mcremote/mcrelay
WebSocket, `_x.ai/` vendor extensions.

### Placement in the magic-cli-remote stack

```
phone / desktop  --WS protocol-v1/v2-->  mcrelay  -->  mcremote daemon
                                                      |
                                                      | ACP JSON-RPC 2.0 stdio
                                                      v
                                                   pi-go acp
```

* mcremote interprets **canonical** slash commands (magic-cli-remote
  `internal/command`, MADR 0023) and either runs them itself
  (`KindDaemon`), calls `session/set_mode` (`KindMode`), calls an optional
  session interface (`KindOp`), or forwards `/name` as `session/prompt`
  (`KindNative`).
* Non-canonical names the agent advertised are forwarded as `session/prompt`.
* protocol-v2 is a WebSocket delta (caps, resume, liveness). It does not
  change the ACP hop. pi-go is unaffected by v2 negotiation.
* pi-go never opens a magic-cli-remote WebSocket.

### SDK and protocol pin

* Module: `github.com/coder/acp-go-sdk` at **v0.13.5** (same pin as
  magic-cli-remote).
* `ProtocolVersionNumber = 1`.
* Agent baseline on that SDK: `initialize`, `authenticate`, `logout`,
  `session/new`, `session/prompt`, `session/cancel`, `session/list`,
  `session/resume`, `session/close`, `session/set_mode`,
  `session/set_config_option`.
* Optional: `session/load` (`AgentLoader`) when session JSONL exists —
  advertised only once it works.
* Additional method mcremote already sends on ACP sessions:
  `session/set_model` with `{sessionId, modelId}`. v1 implements it as a
  model switch. Unknown `_meta` keys are ignored; `_meta.reasoningEffort`
  is accepted as a thinking-level change so a companion Spec can reuse
  today's `SetThinkingLevel` **or** route thinking through
  `session/set_config_option`.
* Unstable methods (`session/fork`, NES, document/*) are not a v1
  requirement. `/fork` is a slash command (copy JSONL / branch the
  session) rather than ACP `UnstableForkSession`.

A later SDK bump (v2 method names) is a new MADR.

### Compatibility assessment (magic-cli-remote × TypeScript Pi × this cut)

Evidence from magic-cli-remote (docs and code, 2026-09-29) and TypeScript
Pi at the commit named in 0001-REPORT (`312184edb`).

1. **Layering already matches.** mcremote's ACP adapter
   (`internal/provider/acpagent`) launches a binary, handshakes
   `initialize`, opens `session/new` with `mcpServers`, streams
   `session/update` into daemon events, and maps permissions. pi-go is
   that binary. There is no `provider.IDPi` today (`IDFake`, `IDGrok`,
   `IDOpencode`, `IDCodex`, `IDKilo` only).
2. **Advertisement is not capability.** MADR 0023 (accepted 2026-07-25):
   grok advertises `/compact` and `/context` over ACP; sending them
   returned zero `session/update` frames. A provider `command.Table` beats
   `available_commands`. pi-go advertises only what `Prompt` executes.
3. **Canonical vocabulary is daemon-owned.** `internal/command.Specs`
   lists help, plan, mode, permissions, reviewer, approve, model,
   thinking, context, status, usage, compact, clear/reset, new, sessions,
   goal, deep-research, workflow, loop, fast, personality, review, fork,
   archive, delete, ps, stop, diff, undo, redo. Defaults prefer daemon or
   `KindOp` over forwarding, because advertised commands often do not
   execute.
4. **`KindOp` is proven by Go interfaces on the live session**, not by
   claims. `OpCompact` requires `provider.CompactSession`. On `acpagent`,
   `(*session).Compact` currently calls grok's `x.ai/compact_conversation`
   for **every** ACP Spec. `OpSetModel` / `OpSetThinkingLevel` call
   `session/set_model` (grok `_meta.reasoningEffort` plus a
   `session/resume` snapshot read-back for thinking). `OpContext` is true
   only after a `usage_update` has been stored (`e.lastUsage != nil`).
   grok's table still marks `/compact` and `/usage` `KindNone`. An `IDPi`
   Spec that reuses those methods unchanged would invoke grok vendor RPCs.
5. **MCP on the daemon hop is HTTP/SSE only.** `acpagent.buildMcpServers`
   forwards configured servers whose transport the agent advertised
   (`mcpCapabilities.http` / `sse`) and drops the rest. Stdio MCP is the
   ACP default untagged form; the daemon config type in that function does
   not forward it. pi-go still accepts stdio and HTTP on `session/new`
   (Zed, Cobra, `mcp.json`) and advertises `http = true`, `sse = false`,
   `acp = false`.
6. **Grok vendor extensions stay grok's.** `ExtensionNotifications` on the
   grok Spec subscribe to `_x.ai/models/update`, MCP status, session
   notifications, and related methods. Several others are deliberately
   declined (announcements, settings, queue). pi-go uses `_pi/` for
   Pi-only ops and standard ACP otherwise. Plan approval uses
   `session/request_permission`.
7. **TypeScript Pi slash commands that are TUI chrome must not be
   advertised.** Built-ins at `packages/coding-agent/docs/slash-commands.md`:
   models/settings (`/settings`, `/model`, `/thinking`, `/scoped-models`,
   `/login`, `/logout`, `/llama`); sessions (`/new`, `/resume`, `/name`,
   `/session`, `/tree`, `/fork`, `/clone`, `/compact [instructions]`,
   `/import`); export (`/copy`, `/export`, `/share`, `/bug`); runtime
   (`/trust`, `/reload`, `/hotkeys`, `/changelog`, `/quit`). `/session`
   reports file, id, message count, tokens, and cost — the canonical
   `/context` idea. `/compact` already takes optional instructions (RPC
   `compact.customInstructions`). `/settings`, `/hotkeys`, `/quit`,
   `/copy` are TUI; advertising them over ACP would recreate grok's
   catalog lie.
8. **protocol-v1 `commands.list` / `remote_commands`.** The daemon
   advertises the resolved canonical list, including unavailable commands
   with reasons. Clients invoke with `session.prompt` text `/name args…`.
   pi-go's job is to make the agent-side mappings true, and to emit the
   ACP events (`available_commands_update`, `usage_update`,
   `current_mode_update`, `config_option_update`) the daemon already
   consumes.

### Native slash-command contract

Rules:

1. Advertise a command iff `session/prompt` with that `/name` executes it
   (non-empty `session/update` stream or a structured extension result
   that the prompt path also surfaces as agent text).
2. Dual-path every command that MADR 0023 prefers as `KindOp`: implement
   the slash form **and** a `_pi/` method the companion Spec can call, so
   either table mapping works.
3. Daemon-owned commands (`/help`, `/clear`, `/new`, `/sessions`) work
   even if the agent is silent. pi-go still handles native equivalents for
   Cobra and for ACP clients that are not mcremote (`/new` → `session/new`
   semantics in-process, `/help` lists advertised commands).
4. Codex-only or provider-specific commands that pi-go does not implement
   are omitted from `available_commands` and, in the companion table,
   `KindNone` with a user-readable reason. They are never advertised as
   working.

### Intended companion `command.Table` (magic-cli-remote `IDPi`)

This table is the contract for a future Spec in magic-cli-remote. It is
not implemented in this repository. Cite magic-cli-remote MADR 0023 and
`internal/command` when landing it. Every canonical name gets an entry
(conformance test in that repo).

| Canonical | Kind | Mechanism | Notes |
|---|---|---|---|
| `help` | KindDaemon | daemon | pi-go also answers `/help` over ACP for non-mcremote clients |
| `plan` | KindMode, ModeID `plan` | `session/set_mode` | v1 implements modes `default` and `plan` |
| `mode` | KindMode | `session/set_mode` | |
| `permissions` | KindNone | — | reason: this agent uses `/mode` for agent modes, not permission presets |
| `reviewer` | KindNone | — | no separate approval reviewer |
| `approve` | KindNone | — | no Guardian denial |
| `model` | KindOp, OpSetModel | `session/set_model` and `session/set_config_option` | mid-session, conversation kept |
| `thinking` | KindOp, OpSetThinkingLevel | `session/set_config_option`; `_meta.reasoningEffort` accepted | mutability **live** |
| `context` | KindOp, OpContext | daemon from last `usage_update` | also slash `/context` and Pi `/session` (non-canonical alias advertised) |
| `status` | KindNone | — | no host runtime status in v1 |
| `usage` | KindNative, Native `usage` | `session/prompt` `/usage`; `_pi/usage` for a later RuntimeSession | account/rate-limit if known, else session tokens/cost |
| `compact` | KindNative, Native `compact` until Compact is Spec-hooked; then KindOp OpCompact | `/compact [instructions]` and `_pi/compact` | **do not** call `x.ai/compact_conversation` |
| `clear` / `reset` | KindDaemon | daemon | |
| `new` | KindDaemon | daemon | |
| `sessions` | KindDaemon | daemon | |
| `goal` | KindNone | — | no autonomous goal in v1 |
| `deep-research` | KindNone | — | not a v1 agent command |
| `workflow` | KindNone | — | not a v1 agent command |
| `loop` | KindNone | — | not a v1 agent command |
| `fast` | KindNone | — | no Fast service tier |
| `personality` | KindNone | — | no personality setting |
| `review` | KindNone | — | no inline review command |
| `fork` | KindNative, Native `fork` | `/fork [turn-id]` copies/branches JSONL | ACP unstable fork out of v1 |
| `archive` | KindNone | — | no native archive |
| `delete` | KindNone | — | no native permanent-delete over ACP; Cobra may delete files |
| `ps` | KindNone | — | no terminal registry |
| `stop` | KindNone | — | no stoppable terminals |
| `diff` | KindNone | — | no diff op in v1; user can ask the agent |
| `undo` | KindNone | — | no undo op in v1 |
| `redo` | KindNone | — | no redo op in v1 |

Companion Spec sketch (fields that exist on `acpagent.Spec` today):

* `ID`: new `provider.IDPi` (exact string chosen in that repo).
* `DefaultBin`: `pi-go` (until D17 says otherwise).
* `DefaultArgs`: `[]string{"acp"}`.
* `Commands`: the table above, every Specs name declared.
* `CommandCaveat`: empty unless a session-wide quirk appears.
* `StaticModes`: `{default, plan}` only if `session/new` returns none;
  agent-supplied list wins.
* `SynthesizeAutoMode`: false unless v1 later lacks a native auto and the
  daemon should inject one.
* `ExtensionNotifications`: empty. No `_x.ai/` handlers.
* `SessionMeta`: nil. No grok `_meta` on session/new.
* Compact: **new Spec hook** required. Today's `(*session).Compact` is
  grok-only. Until that hook exists, the table must stay `KindNative` for
  `compact` so the daemon forwards `/compact` as a prompt.
* Thinking: prefer `SetConfigOption`; do not require grok's
  `session/resume` snapshot read-back.

Live-tagged tests in that repo (MADR 0023 checklist): send `/compact`,
`/usage`, `/context` and assert output comes back; assert `/settings` is
not advertised.

### TypeScript Pi RPC → v1 mapping

Source of the left column: TypeScript Pi `rpc-commands.md` at the commit
named in the report (`312184edb`).

| Pi RPC command | v1 |
|---|---|
| `prompt` | ACP `session/prompt` |
| `abort` | ACP `session/cancel` |
| `new_session` | ACP `session/new` |
| `switch_session` | ACP `session/load` or `session/resume` |
| `get_commands` | ACP `session/update` / `available_commands_update` (honest list) |
| `set_model`, `cycle_model`, `get_available_models` | ACP `session/set_config_option` and `session/set_model` |
| `set_thinking_level`, `cycle_thinking_level`, `get_available_thinking_levels` | ACP `session/set_config_option` (thinking); `_meta.reasoningEffort` accepted |
| `get_state`, `get_messages`, `get_entries`, `get_tree`, `get_last_assistant_text`, `get_session_stats` | `_pi/…` extensions (read models); stats also via `/session` and `/usage` slash |
| `steer`, `follow_up`, `clear_queue` | `_pi/steer`, `_pi/follow_up`, `_pi/clear_queue` |
| `compact`, `set_auto_compaction`, `set_auto_retry`, `abort_retry` | slash `/compact [instructions]`; `_pi/compact`; `_pi/set_auto_compaction` (retry policy may wait) |
| `bash`, `abort_bash` | `_pi/bash` (agent-local exec; not ACP `terminal/*`, which is a **client** capability) |
| `fork`, `clone`, `get_fork_messages` | slash `/fork` `/clone`; out of ACP unstable fork |
| `set_session_name` | slash `/name`; `_pi/set_session_name` |
| `export_html` | out of v1 |
| `set_steering_mode`, `set_follow_up_mode` | `_pi/…` |

Cobra `pi mcp add|remove|list|login|logout` and `pi auth …` are
**configuration** commands. They edit files / credentials. They are not ACP
methods. A running session sees MCP changes on the next `session/new` or
via a `_pi/mcp_reload` extension if v1 implements live reload — live reload
is not required.

### MCP

TypeScript Pi: `~/.pi/agent/mcp.json` and trusted `.pi/mcp.json`; stdio
(`command`/`args`) and streamable HTTP (`url`); SSE rejected; OAuth login
for HTTP; tool names `mcp__<server>__<tool>`; exposure `direct` / codemode.

ACP: `NewSessionRequest.McpServers` is required (may be an empty array).
`McpCapabilities` flags `http`, `sse`, and unstable `acp`. Stdio MCP
servers are the default untagged form.

v1:

* Advertise `mcpCapabilities.http = true`, `sse = false`, `acp = false`.
* Accept stdio and HTTP servers from `session/new` **and** from the
  on-disk `mcp.json` (union; session-supplied names override file names).
* `pi mcp …` mutates the file, matching TypeScript Pi's CLI.
* OAuth for HTTP MCP is in v1 if the chosen Go MCP client supports it;
  otherwise it is a dated PLAN deviation, not a silent drop.

Which Go MCP client library (report D14) is **not** chosen here. The PLAN
picks one in its MCP phase and records the module path.

### CLI as ACP client

The binary has two process roles, selected by Cobra:

1. **Agent** — stdio ACP server (headless). Editors and mcremote spawn
   `pi-go acp` (exact binary name still D17).
2. **Client** — Cobra command runs an in-process agent through
   `ClientSideConnection` / `AgentSideConnection` on an in-memory pipe, or
   prints a one-shot `session/prompt`. No Charm views.

A Cobra command must not call the agent loop except through ACP methods
(baseline, `session/set_model`, or `_pi/` extension). That is the
"extensive use" rule.

### Launch shape mcremote will use

Analogous to grok's `grok agent --no-leader … stdio`:

* DefaultBin `pi-go`
* DefaultArgs `acp`
* `KnownGoodVersion` set when a release exists; mismatch warns, never
  refuses (magic-cli-remote MADR 0137).
* Capability is proven by live probes, not by this MADR's table alone.

### Still open (not this MADR)

From the report, still open after this cut: D8 (images), D9
(distribution), D10 (providers / `go-llmprovider-sdk`), D11 (home
directory vs TypeScript Pi), D13 (grep/find), D14 (MCP client module),
D16 (settings merge), D17 (binary name, User-Agent), D18 (derived vs
clean-room). D3 (codemode), D6 (Charm), D7 (package manager), D12
(experimental stack) are out of v1 by this decision.

Companion `IDPi` registration, Compact Spec hook, and live-tagged tests
live in magic-cli-remote and need a MADR/PLAN pair **there** before that
tree is mutated.
