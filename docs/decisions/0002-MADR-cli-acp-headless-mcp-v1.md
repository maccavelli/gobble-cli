---
status: proposed
date: 2026-10-04
decision-makers: repository owner
consulted: 0001-REPORT-go-port-feasibility.md, magic-cli-remote (ACP stdio adapter, command tables, protocol-v1/v2), TypeScript Pi at 312184edb
informed: go-llmprovider-sdk, go-core-lib, acp-go-sdk, magic-cli-remote
---
# v1 is the native magic-cli-remote CLI: Kong over ACP, ACP stdio, MCP client

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
magic-cli-remote**. Other CLI providers in that daemon continue. gobble is
fully optimized for, and fully supported by, that stack — leveraging the
existing **mcremote** / **mcrelay** protocols and the canonical slash-command
standard (magic-cli-remote MADR 0023), and exposing agent CLI commands such
as `/compact` and `/usage` **natively over ACP** so they actually execute.
gobble may be rewritten; TypeScript Pi's JSONL RPC, Charm TUI, and jiti
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
  providers in that daemon keep working; gobble is the fully supported,
  fully optimized agent.
* One agent command API, used by Kong, by editors, and by mcremote, rather
  than a Kong vocabulary plus a private RPC dialect plus a TUI-only slash
  catalog.
* mcremote/mcrelay already define the phone/desktop hop (WebSocket
  protocol-v1 / v2). gobble belongs on the **ACP stdio** hop those daemons
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
* The CLI library is Kong. The config surface is undecided and will be either a native Kong facility or a surface we write. gobble has two modes:
  a native terminal CLI mode, which is the default, and an enhanced terminal
  TUI mode. Core TUI is go-tui-lib. gobble does not reimplement core TUI.
  Where a core behaviour is not in the library yet, gobble waits rather than
  copying it. Self-update is go-selfupdate-lib. gobble does not reimplement
  self-update.

## Considered Options

* Native magic-cli-remote CLI: Kong over one `acp.Agent`, ACP stdio
  headless, MCP client, honest slash commands that execute (chosen)
* Cobra v1.10.2, or fang (`charmbracelet/fang` v1.0.0) on Cobra, as the CLI library (rejected)
* Cobra CLI plus a faithful port of TypeScript Pi JSONL RPC, ACP added later (rejected)
* Dual headless protocols in v1 (Pi JSONL RPC and ACP) (rejected)
* ACP-only stdio agent with no Cobra command surface beyond `acp` (rejected)
* gobble speaks mcremote WebSocket (protocol-v1/v2) in addition to ACP (rejected)
* Grok-shaped adapter: advertise a TUI catalog and rely on daemon
  workarounds (`KindNone`, native-name aliases, vendor `_x.ai/*`) (rejected)

## Decision Outcome

Chosen option: "Native magic-cli-remote CLI: Kong over one `acp.Agent`,
ACP stdio headless, MCP client, honest slash commands that execute",
because the owner asked for those three surfaces, for ACP to expose the
agent command API, and for gobble to be the native agent of
magic-cli-remote.

One `acp.Agent` implementation is the command API. The Kong CLI is an ACP
**client** of that implementation. Headless RPC is ACP JSON-RPC 2.0 on
stdio — the same process role mcremote already spawns. MCP is an agent
capability (stdio and streamable HTTP), configured from `mcp.json` and from
`session/new.mcpServers`. Agent-owned slash commands (`/compact`, `/usage`,
`/context`, `/model`, `/thinking`, and the rest of the native set below)
are handled in `session/prompt` and, where mcremote has an optional
session interface, also as `_gobble/` extension methods so either a
`KindNative` or a `KindOp` table mapping works.

gobble does **not** implement mcremote or mcrelay WebSocket. Those stay in
magic-cli-remote. Other providers there (grok, opencode, kilo, codex, fake)
continue. A future `provider.IDPi` Spec in that repository is companion
work specified here, not executed in this tree.

TypeScript Pi's JSONL `--mode rpc` is **out of v1**. A later MADR may add a
translation shim. Charm TUI, jiti extensions, codemode, Chord/Pico, and a
browser/TypeScript SDK remain out of v1. *(Scope superseded 2026-09-29 by
[0005-MADR](0005-MADR-v1-feature-scope.md): the TUI and the JSONL RPC shim
return to the v1 line as ACP clients, and codemode returns under `exp/`.
The ACP-as-command-API decision above is unchanged.)*

### Consequences

* Good, because editors that already speak ACP (Zed and anything using
  `acp-go-sdk`) can spawn this binary without a private protocol.
* Good, because the CLI, an IDE, and mcremote execute the same methods; a
  Kong command cannot drift from what the phone can call.
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
  `_gobble/`-prefixed extension methods. Editors that ignore unknown extensions
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
  unchanged would call methods gobble does not implement. Companion work
  must parameterize those hooks (see More Information).

### Confirmation

* A stdio round-trip against `github.com/coder/acp-go-sdk` at the pinned
  version: `Initialize`, `NewSession`, `Prompt`, `Cancel`, and
  `CloseSession` succeed with the SDK's example client or an equivalent
  in-tree test.
* Every Kong command that drives the agent is implemented as a call to
  that same `acp.Agent` (in-process connection), asserted by a test that
  the CLI and a stdio client produce the same ACP method names for the
  same user action.
* `session/new` with a stdio MCP server and with a streamable-HTTP MCP
  server advertises tools to the model; `pi mcp list` reports the same
  servers. SSE is not implemented (TypeScript Pi rejects `sse`; ACP's SSE
  capability stays `false`).
* ~~No package under this module imports a Charm / Lip Gloss / Bubble Tea
  module in v1.~~ *Superseded (2026-09-29, second amendment):* only
  `internal/tui` imports Charm, and it is an ACP client ([0004-MADR](0004-MADR-go-module-architecture.md) import rules
  5 and 6).
* A mapping table in this record's More Information is kept honest: each
  TypeScript Pi RPC command is either an ACP baseline method, a `_gobble/`
  extension, a Kong config command, or explicitly out of v1.
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
  thinking (option categories `model` and `thought_level`).
  ~~`session/set_model` with `{sessionId, modelId}` (the request
  mcremote `acpagent.SetModel` already sends) also changes the model.~~
  *Withdrawn (2026-09-29, second amendment):* the SDK cannot route that
  method to the agent.
* No `_x.ai/` method is implemented or advertised.

## Pros and Cons of the Options

### Native magic-cli-remote CLI: Kong over one `acp.Agent`, ACP stdio headless, MCP client, honest slash commands that execute (chosen)

The CLI process always hosts (or is) an ACP agent. Headless mode is
`AgentSideConnection` on stdin/stdout — the process mcremote spawns.
Interactive-ish CLI commands (`prompt`, `session new`, `session resume`)
are `ClientSideConnection` calls, in-process over pipes or to a spawned
stdio child. Slash commands the agent owns execute in `Prompt` and, where
needed for `KindOp`, as `_gobble/` extensions. The PLAN picks in-process vs
child transport; this record requires that the bytes are ACP.

* Good, because it matches the owner's ACP, CLI, MCP, and
  magic-cli-remote-native requirements with one command dialect.
* Good, because mcremote already knows how to be an ACP client of this
  shape; the remaining work there is a Spec and an honest command table.
* Bad, because TypeScript Pi RPC clients are not served in v1.

### The command library ocp-login already ships (rejected)

Named in Considered Options. It supplies styled help, errors, version
output, man pages, and completions. gobble does not use it.

* Good, because that pair is already in production in ocp-login.
* Bad, because the owner chose Kong. It is not gobble's command tree,
  help path, or completion path.

### Faithful TypeScript Pi JSONL RPC, with ACP added later (rejected)

Reproduce `--mode rpc` record-for-record, then wrap it.

* Good, because existing Pi RPC clients keep working.
* Bad, because the owner asked for ACP as the command API, and this option
  makes ACP a façade.
* Bad, because JSONL `type: prompt` and JSON-RPC `session/prompt` would
  both exist and diverge.

### Dual headless protocols in v1 (Pi JSONL RPC and ACP) (rejected)

Ship both from day one.

* Good, because no client is left behind.
* Bad, because v1 then maintains two envelopes, two command inventories,
  and two test matrices for the same agent.
* Neutral, because a later shim can translate JSONL to ACP once the ACP
  API is stable.

### ACP-only stdio agent with no command surface beyond `acp` (rejected)

A single `pi` that only speaks ACP on stdio. Users use an editor,
mcremote, or `acp-go-sdk`'s example client.

* Good, because the smallest binary that is an ACP agent.
* Bad, because the owner asked for a CLI, including `pi mcp …`, `pi auth …`,
  and print-style prompting, which TypeScript Pi offers without an editor.

### gobble speaks mcremote WebSocket (protocol-v1/v2) in addition to ACP (rejected)

Put the daemon protocol inside the agent binary.

* Good, because a phone could connect without a separate mcremote.
* Bad, because that duplicates mcremote/mcrelay (auth, pairing, resume,
  history ring, canonical command resolution) and splits the fleet's
  WebSocket into two implementations.
* Bad, because the existing architecture is already
  phone → mcrelay → mcremote → ACP stdio → agent; adding a second
  WebSocket server on the agent does not use those protocols, it forks
  them.

### Grok-shaped adapter: advertise a TUI catalog and rely on daemon workarounds (rejected)

Ship `available_commands` as a superset, mark the ones that do not work
`KindNone` in the daemon table, and copy `_x.ai/*` extensions.

* Good, because it matches how grok is integrated today.
* Bad, because the owner asked for a rewrite that is fully supported, and
  MADR 0023 exists precisely because that shape silently drops `/compact`
  and `/usage`.
* Bad, because `_x.ai/*` is grok vendor; implementing it in gobble would
  couple the native CLI to another product's extensions.

## More Information

### Amendment (2026-09-29)

This record is still `proposed`. The first draft chose ACP as the sole
agent command API, mapped TypeScript Pi RPC onto baseline ACP plus `_gobble/`
extensions, and allowed v1 to ship with an empty slash-command list. The
owner then required native magic-cli-remote support: honest slash commands
that execute over ACP (`/compact`, `/usage`, and the rest of the native
set), integration with mcremote/mcrelay as they exist, and other providers
left in place. The Decision Outcome, Confirmation, mapping tables, and
companion Spec below replace that empty-list allowance. The ACP-as-command-API
choice, the JSONL-RPC-out-of-v1 choice, and the Kong-is-an-ACP-client
choice are unchanged.

### Amendment (2026-09-29, second): `gobble` identity, SDK routing correction, wider v1 line

Still `proposed`. Four changes; the ACP-as-command-API decision, the
Kong-is-an-ACP-client rule, and the honest-advertisement rule are
unchanged.

1. **Identity.** The owner named the runtime binary `gobble`. [0003-MADR](0003-MADR-gobble-product-identity.md) fixes
   the rest (directories, `GOBBLE_` environment, User-Agent, ACP
   `agentInfo`). In this record the product and binary now read `gobble`
   where they read `gobble-cli`, and extension methods read `_gobble/…` where
   they read `_pi/…`. The repository and module path keep `gobble-cli`.
2. **Correction: `session/set_model` cannot be served.** This record said
   v1 implements `session/set_model` because mcremote's
   `acpagent.SetModel` sends it. `github.com/coder/acp-go-sdk` v0.13.5 has
   no such method in its schema. `AgentSideConnection` sends only
   `_`-prefixed methods to `ExtensionMethodHandler` (`extensions.go`), and
   `agent_gen.go` answers every other unknown method with
   `NewMethodNotFound`. gobble therefore handles model and thinking changes
   only through `session/set_config_option` (categories `model` and
   `thought_level`) and the `/model` and `/thinking` slash commands. The
   companion Spec must route `OpSetModel` and `OpSetThinkingLevel` through
   `SetConfigOption`, which mcremote already has, or map them `KindNative`.
   Options not taken: forking the SDK to add a route, or wrapping the
   stdio reader to intercept the method before the SDK sees it. Both would
   put a non-standard method into the agent that the standard makes
   unnecessary.
3. **Module shape and toolchain** are decided in [0004-MADR](0004-MADR-go-module-architecture.md). [0004-PLAN](0004-PLAN-go-module-architecture.md) replaces
   Phase 0 of the associated PLAN.
4. **v1 line.** [0005-MADR](0005-MADR-v1-feature-scope.md) widens v1 at the owner's request ("include as much as
   possible"), with tiers (1.0 gate, 1.x train, `exp/`, out). It supersedes
   "What v1 is not" and the companion command table below as the target.
   [0005-PLAN](0005-PLAN-v1-feature-scope.md) follows the associated PLAN.

### What v1 is

| Surface | Role |
|---|---|
| Kong | Process entry, flags, `mcp` / `auth` / `config` subcommands, print-style `prompt`. The config surface is undecided and will be either a native Kong facility or a surface we write. |
| `acp.Agent` | The agent command API, including slash commands |
| ACP stdio | Headless RPC; the process magic-cli-remote spawns |
| MCP client | stdio and streamable HTTP; tools become ordinary agent tools |

### What v1 is not

> *Superseded (2026-09-29, second amendment) by [0005-MADR](0005-MADR-v1-feature-scope.md):* the Charm TUI, HTML
> export, data-package install, and the Pi JSONL RPC shim return to the v1
> line; codemode returns under `exp/`. The rest of this list stays out.

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
                                                   gobble acp
```

* mcremote interprets **canonical** slash commands (magic-cli-remote
  `internal/command`, MADR 0023) and either runs them itself
  (`KindDaemon`), calls `session/set_mode` (`KindMode`), calls an optional
  session interface (`KindOp`), or forwards `/name` as `session/prompt`
  (`KindNative`).
* Non-canonical names the agent advertised are forwarded as `session/prompt`.
* protocol-v2 is a WebSocket delta (caps, resume, liveness). It does not
  change the ACP hop. gobble is unaffected by v2 negotiation.
* gobble never opens a magic-cli-remote WebSocket.

### SDK and protocol pin

* Module: `github.com/coder/acp-go-sdk` at **v0.13.5**. magic-cli-remote
  requires the same version, then `replace`s it with the owner's fork
  `v0.13.6-mcr.1` (its MADR 0167). Whether gobble carries that replace is
  decided in [0004-PLAN](0004-PLAN-go-module-architecture.md) Phase 0.
* `ProtocolVersionNumber = 1`.
* Agent baseline on that SDK: `initialize`, `authenticate`, `logout`,
  `session/new`, `session/prompt`, `session/cancel`, `session/list`,
  `session/resume`, `session/close`, `session/set_mode`,
  `session/set_config_option`.
* Optional: `session/load` (`AgentLoader`) when session JSONL exists —
  advertised only once it works.
* ~~Additional method mcremote already sends on ACP sessions:
  `session/set_model` with `{sessionId, modelId}`. v1 implements it as a
  model switch. Unknown `_meta` keys are ignored; `_meta.reasoningEffort`
  is accepted as a thinking-level change so a companion Spec can reuse
  today's `SetThinkingLevel` **or** route thinking through
  `session/set_config_option`.~~ *Corrected (2026-09-29, second
  amendment):* `session/set_model` is not in the ACP schema this SDK
  generates. The SDK's agent side answers MethodNotFound for any
  non-schema method that does not start with `_`. Model and thinking are
  `session/set_config_option` categories `model` and `thought_level`.
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
   `session/update` into daemon events, and maps permissions. gobble is
   that binary. There is no `provider.IDPi` today (`IDFake`, `IDGrok`,
   `IDOpencode`, `IDCodex`, `IDKilo` only).
2. **Advertisement is not capability.** MADR 0023 (accepted 2026-07-25):
   grok advertises `/compact` and `/context` over ACP; sending them
   returned zero `session/update` frames. A provider `command.Table` beats
   `available_commands`. gobble advertises only what `Prompt` executes.
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
   not forward it. gobble still accepts stdio and HTTP on `session/new`
   (Zed, Kong, `mcp.json`) and advertises `http = true`, `sse = false`,
   `acp = false`.
6. **Grok vendor extensions stay grok's.** `ExtensionNotifications` on the
   grok Spec subscribe to `_x.ai/models/update`, MCP status, session
   notifications, and related methods. Several others are deliberately
   declined (announcements, settings, queue). gobble uses `_gobble/` for
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
   gobble's job is to make the agent-side mappings true, and to emit the
   ACP events (`available_commands_update`, `usage_update`,
   `current_mode_update`, `config_option_update`) the daemon already
   consumes.

### Native slash-command contract

Rules:

1. Advertise a command iff `session/prompt` with that `/name` executes it
   (non-empty `session/update` stream or a structured extension result
   that the prompt path also surfaces as agent text).
2. Dual-path every command that MADR 0023 prefers as `KindOp`: implement
   the slash form **and** a `_gobble/` method the companion Spec can call, so
   either table mapping works.
3. Daemon-owned commands (`/help`, `/clear`, `/new`, `/sessions`) work
   even if the agent is silent. gobble still handles native equivalents for
   Kong and for ACP clients that are not mcremote (`/new` → `session/new`
   semantics in-process, `/help` lists advertised commands).
4. Codex-only or provider-specific commands that gobble does not implement
   are omitted from `available_commands` and, in the companion table,
   `KindNone` with a user-readable reason. They are never advertised as
   working.

### Intended companion `command.Table` (magic-cli-remote `IDPi`)

> *Superseded as the target (2026-09-29, second amendment) by the table in
> [0005-MADR](0005-MADR-v1-feature-scope.md).* This table stays the floor for a gobble release that predates the
> 0005 handlers. The `model` and `thinking` rows cannot use
> `session/set_model` (see the second amendment). Until the Spec routes
> those ops through `session/set_config_option`, they are `KindNative`
> (`/model`, `/thinking`).

This table is the contract for a future Spec in magic-cli-remote. It is
not implemented in this repository. Cite magic-cli-remote MADR 0023 and
`internal/command` when landing it. Every canonical name gets an entry
(conformance test in that repo).

| Canonical | Kind | Mechanism | Notes |
|---|---|---|---|
| `help` | KindDaemon | daemon | gobble also answers `/help` over ACP for non-mcremote clients |
| `plan` | KindMode, ModeID `plan` | `session/set_mode` | v1 implements modes `default` and `plan` |
| `mode` | KindMode | `session/set_mode` | |
| `permissions` | KindNone | — | reason: this agent uses `/mode` for agent modes, not permission presets |
| `reviewer` | KindNone | — | no separate approval reviewer |
| `approve` | KindNone | — | no Guardian denial |
| `model` | KindOp, OpSetModel | `session/set_model` and `session/set_config_option` | mid-session, conversation kept |
| `thinking` | KindOp, OpSetThinkingLevel | `session/set_config_option`; `_meta.reasoningEffort` accepted | mutability **live** |
| `context` | KindOp, OpContext | daemon from last `usage_update` | also slash `/context` and Pi `/session` (non-canonical alias advertised) |
| `status` | KindNone | — | no host runtime status in v1 |
| `usage` | KindNative, Native `usage` | `session/prompt` `/usage`; `_gobble/usage` for a later RuntimeSession | account/rate-limit if known, else session tokens/cost |
| `compact` | KindNative, Native `compact` until Compact is Spec-hooked; then KindOp OpCompact | `/compact [instructions]` and `_gobble/compact` | **do not** call `x.ai/compact_conversation` |
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
| `delete` | KindNone | — | no native permanent-delete over ACP; Kong may delete files |
| `ps` | KindNone | — | no terminal registry |
| `stop` | KindNone | — | no stoppable terminals |
| `diff` | KindNone | — | no diff op in v1; user can ask the agent |
| `undo` | KindNone | — | no undo op in v1 |
| `redo` | KindNone | — | no redo op in v1 |

Companion Spec sketch (fields that exist on `acpagent.Spec` today):

* `ID`: new `provider.IDPi` (exact string chosen in that repo).
* `DefaultBin`: `gobble` ([0003-MADR](0003-MADR-gobble-product-identity.md)).
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
| `set_model`, `cycle_model`, `get_available_models` | ACP `session/set_config_option` (category `model`); `session/set_model` withdrawn by the second amendment |
| `set_thinking_level`, `cycle_thinking_level`, `get_available_thinking_levels` | ACP `session/set_config_option` (thinking); `_meta.reasoningEffort` accepted |
| `get_state`, `get_messages`, `get_entries`, `get_tree`, `get_last_assistant_text`, `get_session_stats` | `_gobble/…` extensions (read models); stats also via `/session` and `/usage` slash |
| `steer`, `follow_up`, `clear_queue` | `_gobble/steer`, `_gobble/follow_up`, `_gobble/clear_queue` |
| `compact`, `set_auto_compaction`, `set_auto_retry`, `abort_retry` | slash `/compact [instructions]`; `_gobble/compact`; `_gobble/set_auto_compaction` (retry policy may wait) |
| `bash`, `abort_bash` | `_gobble/bash` (agent-local exec; not ACP `terminal/*`, which is a **client** capability) |
| `fork`, `clone`, `get_fork_messages` | slash `/fork` `/clone`; out of ACP unstable fork |
| `set_session_name` | slash `/name`; `_gobble/set_session_name` |
| `export_html` | out of this cut; `gobble session export --format html` and `/export` in [0005-MADR](0005-MADR-v1-feature-scope.md) (1.x) |
| `set_steering_mode`, `set_follow_up_mode` | `_gobble/…` |

Kong `gobble mcp add|remove|list|login|logout` and `gobble auth …` are
**configuration** commands. They edit files / credentials. They are not ACP
methods. A running session sees MCP changes on the next `session/new` or
via a `_gobble/mcp_reload` extension if v1 implements live reload — live reload
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

Which Go MCP client library (report D14) is chosen in
[0004-MADR](0004-MADR-go-module-architecture.md): the official
`github.com/modelcontextprotocol/go-sdk`. The PLAN records the version
it pins in Phase 5.

### CLI as ACP client

The binary has two process roles, selected by Kong, and two terminal modes.

1. **Agent** — stdio ACP server (headless). Editors and mcremote spawn
   `gobble acp` (binary named by [0003-MADR](0003-MADR-gobble-product-identity.md)).
2. **Client** — the native terminal CLI mode, which is the default. A Kong
   command runs an in-process agent through
   `ClientSideConnection` / `AgentSideConnection` on an in-memory pipe, or
   prints a one-shot `session/prompt`.
3. **Enhanced terminal TUI** — selected explicitly. A TTY with no prompt
   does not start it. It is an ACP client. Core TUI is go-tui-lib. gobble
   does not reimplement core TUI. Where a core behaviour is not in the
   library yet, gobble waits rather than copying it.

A Kong command must not call the agent loop except through ACP methods
(baseline or `_gobble/` extension). That is the
"extensive use" rule.

### Launch shape mcremote will use

Analogous to grok's `grok agent --no-leader … stdio`:

* DefaultBin `gobble`
* DefaultArgs `acp`
* `KnownGoodVersion` set when a release exists; mismatch warns, never
  refuses (magic-cli-remote MADR 0137).
* Capability is proven by live probes, not by this MADR's table alone.

### Still open (not this MADR)

> *Updated (2026-09-29, second amendment):* D11, D17 and D18 are decided
> by [0003-MADR](0003-MADR-gobble-product-identity.md); D4, D10, D14, D15 and D16 by [0004-MADR](0004-MADR-go-module-architecture.md); D1–D3, D5–D9, D12 and D13 by
> [0005-MADR](0005-MADR-v1-feature-scope.md). The paragraph below is the state before those records.

From the report, still open after this cut: D8 (images), D9
(distribution), D10 (providers / `go-llmprovider-sdk`), D11 (home
directory vs TypeScript Pi), D13 (grep/find), D14 (MCP client module),
D16 (settings merge), D17 (binary name, User-Agent), D18 (derived vs
clean-room). D3 (codemode), D6 (Charm), D7 (package manager), D12
(experimental stack) are out of v1 by this decision.

Companion `IDPi` registration, Compact Spec hook, and live-tagged tests
live in magic-cli-remote and need a MADR/PLAN pair **there** before that
tree is mutated.

### Amendment (2026-09-30, third): shared libraries as they exist

Still `proposed`. D10 is decided in [0004-MADR](0004-MADR-go-module-architecture.md)
as of that record's 2026-09-30 amendment: 1.0 providers are the
`llm/provider` adapter over `go-llmprovider-sdk`; official vendor LLM
SDKs are not 1.0 dependencies. D14 remains the official MCP go-sdk.
The ACP-as-command-API decision, the Kong-is-an-ACP-client rule, and
the honest-advertisement rule are unchanged. [0002-PLAN](0002-PLAN-cli-acp-headless-mcp-v1.md)
Phase 3 and Phase 5 record the module versions they pin.

### Amendment (2026-10-01, fourth): gobble is the native agentic CLI of magic-cli-remote

Still `proposed`. The owner directed a live assessment of magic-cli-remote
(transports, protocols, API and command surfaces, provider wrappers) and
of TypeScript Pi, so gobble can be built from the ground up as the
**native agentic CLI of the magic-cli-remote platform**. Other providers
there (grok, opencode, kilo, codex, fake) stay registered. The
ACP-as-command-API decision, the Kong-is-an-ACP-client rule, and the
honest-advertisement rule are unchanged. The Grok-shaped adapter option
stays rejected, now with file-level evidence.

Evidence is from the magic-cli-remote and Pi working trees on 2026-10-01.
Pi HEAD is still `312184edb68c38248e1acfc3eec68500ba49d9cb` (`v0.99.0`,
zero commits since 0001-REPORT). Pi has no ACP implementation.

#### Transport layering

```
phone / desktop  --WSS JSON, protocol-v1/v2-->  mcrelay (optional)  -->  mcremote daemon
                                                                          |
                                                                          | ACP JSON-RPC 2.0 stdio
                                                                          v
                                                                       gobble acp
```

* The phone hop is `GET /v1/ws` (`docs/protocol-v1.md`). Envelope
  `{v, type, id, payload}`, 1 MiB frames, pair-URI TLS pin. Message
  types live in `internal/protocol/messages.go`: session create / prompt /
  cancel / set_mode / set_config_option / fork / rename / diagnostics,
  `commands.list`, `permission.respond`, provider auth, workspace, and
  Codex-only extras.
* Protocol-v2 (`docs/protocol-v2.md`, MADR 0068) is a **delta on that
  hop**: capability negotiation, liveness, resume, history ring. It does
  not change the ACP hop. gobble never opens a magic-cli-remote WebSocket.
* Agent hops already in the daemon:

  | Provider | Package | Agent transport |
  |---|---|---|
  | grok | `internal/provider/grok` over `acpagent` | ACP stdio subprocess (`grok … agent --no-leader stdio`) |
  | opencode | `internal/provider/opencode` over `httpagent` | shared `opencode serve` HTTP + SSE |
  | kilo | `internal/provider/kilo` over `httpagent` | shared `kilo serve` HTTP + SSE |
  | codex | `internal/provider/codex` | Codex app-server JSON-RPC over WebSocket |
  | fake | `internal/provider/fake` | in-process, for tests |

  There is no `provider.IDPi`. The native slot for gobble is **ACP stdio
  through `acpagent`**, the same machinery grok uses, with a Spec of
  gobble's own. HTTP/SSE and Codex app-server are other products' wires.

#### Command surface (magic-cli-remote MADR 0023, live)

`internal/command.Specs` is the canonical vocabulary (30 names). A
provider `Table` beats `available_commands`. Kinds:

* `KindDaemon` — `/help`, `/clear`, `/new`, `/sessions` (and default
  `/model` in Specs; grok overrides `/model` to `KindOp`).
* `KindMode` — `/plan`, `/mode` via `session/set_mode`.
* `KindOp` — requires an optional interface on the live session
  (`CompactSession`, `ModelSession`, `ThinkingSession`, `ForkSession`,
  `DiffSession`, `RevertSession`, `RuntimeSession`, …). `/context` is
  `OpContext` and is true only after a `usage_update` has been stored.
* `KindNative` — forward `/name` as `session/prompt`. The forwarded
  name may differ (grok maps `/context` → `/session-info`).
* `KindNone` — unavailable, with a user-readable `Note`.

`commands.list` / `remote_commands` advertise the **resolved** list,
including unavailable rows with reasons. Clients invoke with
`session.prompt` text `/name args…`.

#### Provider wrappers evaluated

**Grok (`acpagent` + grok Spec)** is the only shipping ACP stdio
adapter, and it is grok-shaped in the session methods:

| Session op | Implementation today | gobble consequence |
|---|---|---|
| `Compact` | `x.ai/compact_conversation` (`acpagent/sessioncaps.go`) | An `IDPi` Spec that reuses this method calls a grok vendor RPC. Grok's own `commandTable` still maps `compact` to `KindNone`. |
| `SetModel` | raw `session/set_model` (`acpagent/session.go`) | The pinned SDK answers MethodNotFound for this method (second amendment). |
| `SetThinkingLevel` | `session/set_model` `_meta.reasoningEffort` then `session/resume` snapshot (`acpagent/thinking.go`) | Same MethodNotFound. |
| `Fork` | `_x.ai/session/fork` (`acpagent/fork.go`) | Grok vendor. |
| `Rename` | `x.ai/session/rename` | Grok vendor. |
| `RuntimeUsage` / `RuntimeStatus` | `_x.ai/session/usage` and `_x.ai/billing` (`acpagent/runtime.go`) | Grok vendor. |
| `UndoLast` | `_x.ai/rewind/points` then `_x.ai/rewind/execute` (`acpagent/rewind.go`) | Grok vendor. gobble 1.x checkpoints stay `/undo` native until a Spec hook exists. |
| Plan approval | client extension `_x.ai/exit_plan_mode` (`acpagent/extensions.go`) | Grok vendor; translated to a permission card. |
| Questions | `_x.ai/ask_user_question` | Grok vendor. |

`acpagent.Spec` already parameterizes launch, `Commands`,
`ExtensionNotifications`, `StaticModes`, `SynthesizeAutoMode`,
`SessionMeta`, and `ConfigureSession`. `ConfigureSession` runs after
`session/new` and is the documented place to apply a model through
`session/set_config_option`. Grok leaves it nil and uses `SessionMeta`
instead. Compact / Fork / Rename / Usage / SetModel / SetThinking are
**hardcoded on `*session`**, not Spec fields.

Grok `ExtensionNotifications` subscribe to `_x.ai/models/update`,
MCP status, session notifications, MCP membership, MCP init progress.
Four others are declined (`announcements`, `settings`, `sessions/changed`,
`queue/changed`). gobble's Spec sets this map empty.

**OpenCode and Kilo** are the model of a table that tells the truth:
`compact`, `model`, `fork`, `diff`, `undo`, `redo` are `KindOp` because
the HTTP dialect implements those interfaces over engine routes. They
do not forward slash text and hope. ACP has no HTTP summarize route, so
gobble's equivalent of those routes is **honest slash execution plus
`_gobble/` extensions**, then Spec hooks so `KindOp` can resolve.

**Codex** is a third wire (app-server JSON-RPC, collaboration modes,
Guardian, Fast, personality). Those ops stay Codex's. gobble does not
grow them to look like Codex.

**Fake** implements the session contract for tests (turn-busy, control
events, a startup `usage_update` so `/context` resolves). gobble's
`acptest` pair is the ACP analogue.

**MCP on the daemon hop.** `acpagent.buildMcpServers` forwards only
configured servers whose transport the agent advertised
(`mcpCapabilities.http` / `sse`). Stdio MCP is the ACP default untagged
form; the daemon config type has no stdio case, so those entries are
dropped with "unknown transport". gobble still accepts stdio and HTTP on
`session/new` from Zed, Kong, and its own `mcp.json`, and advertises
`http = true`, `sse = false`, `acp = false` so mcremote can forward HTTP
MCP from daemon config.

**ACP client capabilities mcremote already hosts** and gobble must use
when advertised: `fs/read_text_file` and `fs/write_text_file` (file
tools then see editor buffers; every callback is a tool event), and
`terminal/*` (`acpagent/terminal.go`) when `tools.bash.useClientTerminal`
is on. Permission requests are standard `session/request_permission`.
Plan-mode exit uses that path (0005), with no `_x.ai/exit_plan_mode`.

Further adapter facts that shape the native contract:

* Specs default `/model` is `KindDaemon` (relaunch, context lost).
  `ModelSession` is what keeps the conversation. gobble's start-up table
  uses `KindNative` `/model` (config option) until the Spec routes
  `OpSetModel` through `SetConfigOption`.
* `session/new` from acpagent does not send `AdditionalDirectories`.
  Workspace extra roots are gobble settings / ACP client fs, not a daemon
  field today.
* `session/update` `sessionInfoUpdate` is unhandled in acpagent, so a
  gobble `session_info_update` does not become phone `session_title` until
  that mapping lands in the companion pair. `/name` still executes for
  Kong and other ACP clients.
* Child-session ids on the same ACP connection are dropped (MADR 0051
  D6). 0005 subagents relay progress as `tool_call_update` on the parent
  session, which is the path the adapter already forwards.
* Every ACP connection is built with
  `OverflowDropNewest` (MADR 0167). The Phase 0 `replace` to
  `acp-go-sdk v0.13.6-mcr.1` is required on gobble's agent side as well.

**SDK overflow.** magic-cli-remote MADR 0167: upstream `acp-go-sdk`
closes the connection when its 1024-notification queue fills. The fleet
`replace` to `github.com/maccavelli/acp-go-sdk v0.13.6-mcr.1` is the
Phase 0 pin so gobble's agent side has the same overflow policy its main
client relies on.

#### Native gobble contract (ACP hop)

This is the floor a first-class `IDPi` Spec can drive with almost no
workarounds. Capability inventory and slash tiers stay in
[0005-MADR](0005-MADR-v1-feature-scope.md).

1. **Process.** `DefaultBin` `gobble`, `DefaultArgs` `[]string{"acp"}`.
   `KnownGoodVersion` from the gobble release notes; mismatch warns
   (magic-cli-remote MADR 0137).
2. **Initialize.** Protocol `1`. Honest `AgentCapabilities`:
   `loadSession` once JSONL load works; `mcpCapabilities.http = true`,
   `sse = false`, `acp = false`; prompt `image` and `embeddedContext`
   true. Headless-safe `authMethods` only.
3. **Baseline methods** the SDK already routes: `initialize`,
   `authenticate`, `logout`, `session/new`, `session/prompt`,
   `session/cancel`, `session/list`, `session/resume`, `session/close`,
   `session/set_mode`, `session/set_config_option`, `session/load`.
4. **Events mcremote already maps** (`acpagent/session.go` →
   `internal/event`): agent message chunks, tool_call / tool_call_update
   (with ACP `ToolKind`, `locations`, `diff` for edits), plan updates
   (todo widget), `available_commands_update`, `usage_update`
   (`used`, `size`, optional `cost` — this is what makes `/context`
   `KindOp`), `current_mode_update`, `session_info_update`,
   `config_option_update`, `session/request_permission`.
5. **Modes** advertised on `session/new`: at least `default` and `plan`.
   0005 adds `accept-edits` and `bypass` (bypass only when
   `permissions.allowBypass`). `SynthesizeAutoMode` stays false if gobble
   advertises its own modes. Confirm every switch with
   `current_mode_update`.
6. **Model and thinking** are `session/set_config_option` categories
   `model` and `thought_level`, plus `/model` and `/thinking`.
   `ConfigureSession` on the companion Spec applies the start-up model
   the same way.
7. **Slash honesty.** `available_commands_update` ⊆ handlers.
   `/compact`, `/usage`, `/context` (and `/session` as a non-canonical
   alias), `/model`, `/thinking`, `/name`, `/fork`, `/clone`, `/help`
   execute over `session/prompt` with a non-empty `session/update`
   stream. TUI chrome (`/settings`, `/hotkeys`, `/quit`, `/copy`) is
   never advertised.
8. **Dual-path `_gobble/`** for every command MADR 0023 prefers as
   `KindOp`: `_gobble/compact`, `_gobble/usage`, `_gobble/steer`,
   `_gobble/follow_up`, `_gobble/clear_queue`, `_gobble/set_session_name`,
   and the read helpers the mapping table freezes. Editors that ignore
   extensions still run the slash forms.
9. **Empty `ExtensionNotifications`.** No `_x.ai/` methods, no grok
   `_meta` on `session/new`.
10. **Empty `CommandCaveat`** unless a real session-wide quirk appears.

#### Companion Spec (magic-cli-remote, still not this tree)

`IDPi` needs a MADR/PLAN pair **there**. The robust implementation is
to **parameterize `acpagent.Spec`** so Compact / Fork / Rename / Usage /
SetModel / SetThinking are Spec-chosen methods, then point those fields
at `_gobble/…` and `session/set_config_option`. Until that lands, the
`IDPi` table uses `KindNative` for those rows so the daemon forwards
slash text that gobble actually executes.

| Canonical | Until Spec hooks | After Spec hooks |
|---|---|---|
| compact | KindNative `/compact` | KindOp `OpCompact` → `_gobble/compact` |
| usage | KindNative `/usage` | KindOp `OpUsage` → `_gobble/usage` (or `RuntimeSession`) |
| context | KindOp `OpContext` once `usage_update` has been seen | same |
| model | KindNative `/model` | KindOp `OpSetModel` → `SetConfigOption` category `model` |
| thinking | KindNative `/thinking` | KindOp `OpSetThinkingLevel` → category `thought_level` |
| fork | KindNative `/fork` | KindOp `OpFork` → `_gobble/fork` (JSONL branch) |
| plan, mode | KindMode | same (`session/set_mode`) |
| permissions, status | KindNative once handlers exist (0005) | same |
| help, clear, new, sessions | KindDaemon | same |
| reviewer, approve, goal, workflow, loop, fast, personality | KindNone with a readable reason | same unless 0005 later grows them |

Live-tagged tests in that repo (MADR 0023 checklist): send `/compact`,
`/usage`, `/context` and assert `session/update` frames; assert
`/settings` is absent; assert `session/set_model` is MethodNotFound;
assert `_x.ai/` is absent from the module.

#### Pi at `312184edb`: keep, port, redesign, drop

Pi has **no ACP**. gobble supplies that skin. Behaviour worth keeping is
ported into Go packages; the wire is ACP.

**Keep / port (data and documented behaviour)**

* Session JSONL v3 tree, v1/v2 migrate on load, unknown fields
  round-trip (`packages/coding-agent/docs/session-format.md`).
* Prompt templates (`docs/prompt-templates.md`) and Agent Skills
  (`docs/skills.md`, first-wins, collision warning). Discovery order
  kept; paths are gobble's, with a read-only Pi bridge (0003).
* Context files `AGENTS.override.md` / `AGENTS.md` / `CLAUDE.md`;
  `SYSTEM.md` / `APPEND_SYSTEM.md`; sectioned system-prompt patches.
* `mcp.json` schema, SSE rejected, tool names `mcp__<server>__<tool>`,
  20 KB middle truncation.
* Settings JSON key names and Pi merge rules (`settings-manager.ts`).
* Tool contracts and limits: read 2000 lines / 50 KB; image resize
  2000×2000 and 4.5 MB base64; edit uniqueness, overlap, BOM/CRLF,
  fuzzy fallback; bash tail truncate and process-group kill; grep line
  cap 500; find/ls limits; per-realpath mutation queue.
* Agent loop: parallel tool batches (preflight sequential, results in
  source order), sequential tools, steer after the current batch,
  follow-up when idle (`packages/agent/src/agent-loop.ts`).
* Compaction cut-point rules and summary format (`docs/compaction.md`).
* Retry numbers (3, 2 s, 60 s cap, honour `Retry-After`). Classifier
  is typed errors, matching go-llmprovider-sdk.

**Redesign (same idea, ACP/Go shape)**

* Plan mode, permission gate, todo, subagents, checkpoints: Pi ships
  these only as example extensions. gobble makes them first-class (0005)
  over ACP modes, `session/request_permission`, `plan` updates, and
  `_gobble/` / slash commands.
* TUI: Charm v2 as an `acpclient`, not a port of `pi-tui`.
* Image pipeline: Go `image` / `x/image` (report D8).
* grep/find: pure Go (report D13). No rg/fd download.
* Background jobs (`/ps`, `/stop`): new in 0005 1.x; Pi bash has no
  `background` field.

**Drop (fights mcremote, or is language-bound)**

* jiti TypeScript extension host; TypeScript SDK; browser `pi-ai`.
* Chord, Pico/`pi-durable`, `pi-protocol` CBOR, `pi-server`/`pi-client`.
* Pi JSONL `--mode rpc` as a 1.0 command API (1.x shim over ACP is 0005).
* Sharing `~/.pi` writes and `PI_*`.
* Advertising TUI chrome over ACP.
* Implementing `_x.ai/` so the grok Spec would drive gobble unchanged.
* Speaking protocol-v1/v2 from the agent binary.
* Install telemetry, analytics, `deviceId`.
* Photon WASM; npm install of TypeScript extensions.

RPC `steer` / `follow_up` / `clear_queue` / `compact` become `_gobble/`
plus slash. RPC `set_model` / thinking become `session/set_config_option`.
`get_commands` honesty (TUI commands omitted because they would not
execute) is the same rule as ACP `available_commands_update`.

#### Confirmation added by this amendment

* A stdio client that speaks only the JSON-RPC mcremote `acpagent`
  already sends — `initialize`, `session/new` (`mcpServers: []`),
  `session/prompt` `/compact`, `session/set_mode` `plan`,
  `session/set_config_option` category `model` — completes without
  unknown-method errors.
* The grok-silence shape (echo or empty turn on a leading slash) is
  shown failing the Phase 6 tests before those tests are green.
* `grep` of the module for `_x.ai/` is empty.
* File tools through an ACP client that advertises `fs.writeTextFile`
  produce a `fs/write_text_file` frame.
* Companion `IDPi` work remains a magic-cli-remote MADR/PLAN pair. This
  record is the contract that pair must match.

### Amendment (2026-10-01, fifth): cross-repository assessment against code

Still `proposed`. The owner directed a cross-reference of every source:

* magic-cli-remote at `64282e29` / `772c041e`;
* the acp-go-sdk fork at tag `v0.13.6-mcr.1`;
* go-llmprovider-sdk at `940fee0`;
* go-core-lib at `v1.2.0` / `11c1c93`;
* Pi at `312184edb`.

All evidence was read in code, not in earlier records. These three are unchanged:

* ACP as the command API;
* the Kong-is-an-ACP-client rule;
* the honest-advertisement rule.

The magic-cli-remote half of the contract is now a record in that repository: magic-cli-remote `docs/decisions/0179-MADR-gobble-native-acp-provider.md` (`proposed`). It **supersedes** the "Companion Spec" sections of this record as the contract there. The companion table below is replaced by that record's D9, and the companion checklist by its D1–D10. That record names the provider `provider.IDGobble`, with wire id `"gobble"`. Read `IDPi` in this record as that name.

#### Corrections to facts this record asserted

1. **`usage_update` is not standard ACP.** Schema 0.13.5 marks `UsageUpdate` **UNSTABLE** (acp-go-sdk `types_gen.go:9201-9206`). gobble still emits it: mcremote stores it, and `OpContext` depends on it. The Confirmation's "standard ACP `usage_update`" reads "the schema's unstable `usage_update`". Clients that do not know it ignore it.
2. **Fourth amendment, "Events mcremote already maps", item 4, is partly false.** Today `acpagent` does the following:
   * drops `session_info_update` (`session.go:1784`, the default branch);
   * drops `usage_update.cost` (`:1757-1764`);
   * ignores tool-call `locations`;
   * reduces a `diff` to the text `"diff <path>"` (`:2556-2586`);
   * forwards config options without their category (`:1940-1991`).

   The "Further adapter facts" paragraph already said this for `session_info_update`; item 4 contradicted it. gobble emits all of them anyway. magic-cli-remote 0179 D5 maps them.
3. **ACP `fs/*` on mcremote is disk I/O, not editor buffers.** mcremote answers `fs/read_text_file` and `fs/write_text_file` with `os.ReadFile` / `os.WriteFile` (mode 0644, optional `fs_roots`, an audit event per call; `session.go:2309-2382`). The "editor buffers" benefit holds for Zed-class clients only. Routing through the client is still right: every write becomes visible on the phone as a tool event.
4. **`terminal/*` output is invisible on the phone.** The phone sees only `"terminal <id>"`. gobble's default for `tools.bash.useClientTerminal` is therefore **off**. Under mcremote the in-process `bash` tool streams its output as `tool_call_update` content, which the phone renders.
5. **Codex's daemon transport is stdio by default** (`app-server --listen stdio://`, magic-cli-remote `internal/config/config.go:838`). In the fourth amendment's provider table, read "Codex app-server JSON-RPC (stdio by default; WebSocket, Unix socket or managed proxy optional)".
6. **`DefaultArgs` is a function**, `func(cfg Config) []string` (`acpagent.go:51`). The sketch reads `func(acpagent.Config) []string { return []string{"acp"} }`.
7. **`ConfigureSession` runs after `session/new` only, not after `session/load`** (`acpagent.go:57-60,878-892`). Resumed and forked sessions get their model only once 0179 D6 lands. Until then gobble **persists** the session's model and thinking level in the JSONL tree (`model_change`, `thinking_level_change`) and restores them on `session/load`. That is Pi's behaviour, and it makes load correct without the daemon's help.
8. **The model picker reads grok `_meta` or `Spec.ListModels`, never `configOptions`** (`acpagent.go:310-401`). gobble's models appear on the phone only through 0179 D6. gobble publishes them as the `options` of its `model` config option, which is exactly what D6 reads.
9. **The fork `replace` is not "required" on the agent side.** The fork's overflow policy governs a connection's *inbound* notification queue, and it applies only when passed as an option. On the agent side, the inbound notifications are `session/cancel`. Dropping one would lose a cancel (fork `connection.go:136-151`). The decision, recorded in 0004-MADR's amendment of this date:
   * gobble carries the `replace`, because its API adds the options;
   * the agent-side `AgentSideConnection` keeps the default `OverflowCloseConnection`;
   * gobble's in-process clients (`acpclient`, used by the CLI, the TUI and `task`) set `OverflowDropNewest` with a drop handler that surfaces a notice, as mcremote does.
10. **The magic-cli-remote record numbers are cited by full filename from now on.** magic-cli-remote `0175-MADR-conform-docs-tree-to-adopted-record-layout.md` moved its records into `docs/decisions/`. Citations: MADR 0023 is `0023-MADR-canonical-slash-commands.md`, 0137 is `0137-MADR-prompt-to-first-token-latency-regression.md`, 0167 is `0167-MADR-the-acp-sdk-is-dormant-and-its-bounded-queue-is-an-availability-defect.md`, 0068 is `0068-MADR-protocol-v2-reconnect-resilient-transport.md`, and 0051 is `0051-MADR-auto-approve-chat-noise.md`. magic-cli-remote 0169 is `status: proposed` there, so it is not "fleet policy".

#### New finding that changes the companion plan: default fallback

magic-cli-remote `command.Resolve` (`internal/command/command.go:235-263`) falls through to the vocabulary default whenever a declared mapping is unavailable. A `KindNative` row is unavailable until the agent advertises that name. `KindOp` availability is a static Go type assertion on `*acpagent.session` (`internal/session/commands.go:62-88`), and that session implements `CompactSession`, `ThinkingSession`, `UndoSession`, `ForkSession`, `RenameSession`, `ModelSession` and `RuntimeSession` with grok vendor bodies.

So the plan in this record ("`KindNative` until Spec hooks exist") is **unsafe on its own**. In the window before gobble's first `available_commands_update`, `/compact` resolves to `_x.ai/compact_conversation`, `/thinking` to raw `session/set_model`, and `/undo` to `_x.ai/rewind/*`. The phone's Fork and Rename menus call `_x.ai/session/fork` and `x.ai/session/rename` directly.

The fix is in magic-cli-remote 0179:

* D1 makes session ops Spec fields;
* D2 reports capability from the Spec;
* D3 adds strict fallback;
* D4 makes the phone menus follow the resolution.

gobble's part:

* `NewSession` and `LoadSession` send `available_commands_update` **before** they return their response. They then send `current_mode_update` and a first `usage_update` (`used` = the size of the system prompt plus history, `size` = the model window), so `/context` resolves at once. The SDK lets the agent send `session/update` during the request. acpagent keeps a frame that arrives before `session/new` returns, because its child-session filter compares against an empty live id at that point (magic-cli-remote `internal/provider/acpagent/session.go:1622-1627`).
* An unknown `_`-prefixed method gets JSON-RPC -32601. mcremote maps that to "not implemented" rather than a raw error.

#### Further contract facts (verified in magic-cli-remote)

* **mcremote sends no `clientInfo`**, so gobble cannot detect mcremote. gobble's behaviour under mcremote is selected by capabilities alone, never by client name.
* **`!cmd` never reaches gobble from the phone**: the daemon intercepts a leading `!` unless the session implements `ExecutionSession` (`manager.go:915-964`). gobble still handles `!cmd` for Kong, the TUI and editors.
* **`/skill:<name>` and `/mcp:<server>:<prompt>` reach gobble as plain prompt text.** Their names fail mcremote's `isCommandName` (`commands.go:345-358`), so they never appear in `remote_commands`. They execute, because gobble's router runs on all prompt text. For the canonical `deep-research`, gobble also advertises a `/deep-research` alias.
* **mcremote sends only text, image and audio blocks**, never `resource_link`. gobble parses `@path` in prompt text as well as accepting resource blocks. gobble sets `promptCapabilities.audio` to `false`.
* **A prompt sent while a turn is running is queued by the daemon** (a FIFO of 4). It is not sent as a steer. gobble defines `session/prompt`-while-busy in 0005's amendment of this date.
* **The `bypass` mode gets no confirmation gate on the phone** until 0179 D7 (`DangerousModeIDs`).
* **One gobble process runs per session**, plus an optional warm spare, with a 30 s start budget. `gobble acp` start-up does nothing slow before answering `initialize`:
  * no update check;
  * no catalog refresh;
  * no MCP connect (MCP servers connect on `session/new`, inside the 10 s wait).
* **The daemon's environment is inherited**, with no per-provider env. gobble resolves credentials from its own store (`internal/auth`), so a service-launched `gobble acp` works without API keys in the service environment.
* **`agentInfo.version` is read for `KnownGoodVersion`.** gobble always sends it.
* **mcremote calls these methods when they are advertised:** `session/list` (`sessionCapabilities.list`), `session/load` (`loadSession`) and unstable `session/delete`. gobble advertises `sessionCapabilities.list` and `close` from Phase 4. It does not advertise unstable `delete` (0005 keeps that under `acp.unstable`).

#### Confirmation changes

* The script in "Confirmation added by this amendment" (fourth) covers the methods acpagent actually sends: `initialize`, `authenticate` (when configured), `session/new`, `session/load`, `session/prompt` (including `/compact`), `session/cancel`, `session/set_mode`, `session/set_config_option` (with `configId`), `session/list` and `session/close`. It then asserts two failures:
  * `session/set_model` gets MethodNotFound;
  * an unknown `_x.ai/compact_conversation` gets -32601.
* "`session/set_config_option` category `model`" reads: the request carries the `configId` of the option whose declared category is `model`. Category is a property of the option declaration, not of the request.
* A test asserts that `available_commands_update` precedes the `session/new` response on the wire. It is shown failing on a copy that sends it after the response.

### Amendment (2026-10-04): configuration surface undecided

The CLI library stays Kong. The config surface is undecided and will be either a native Kong facility or a surface we write.
