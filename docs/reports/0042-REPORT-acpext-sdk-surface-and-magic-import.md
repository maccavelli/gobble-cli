---
status: observation
date: 2026-11-04
subject: "acp-go-sdk2 extension surface versus the proposed acpext nested module, and the exact contract Magic needs to import for a native gobble provider"
examines: "gobble-cli origin 627035c (acpserver/extensions.go, auth.go, acpclient); acp-go-sdk2 v1.33.0 (extensions.go, helpers_gen.go, client_gen.go, gates); magic-cli-remote origin e2d6493b (acpagent Host, Spec, 0208 P6–P8) and 0179-MADR including its D13 amendment; gobble-cli 0042-MADR/PLAN proposed; 0038-REPORT; 0016-MADR A2; 0039-MADR jsonx"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# REPORT 0042 — acp-go-sdk2 surface, acpext contents, and Magic import

This report records findings. It decides nothing. The owner asked on 2026-11-04 for an assessment of acp-go-sdk2's API and features, of the proposed gobble-cli `acpext` module with **only** acp-go-sdk2 as a non-stdlib dependency, and of exactly what that module must hold so magic-cli-remote can implement gobble as a native ACP provider (ACP `authenticate` and phone 0074 included).

Every claim names its source: a file at the baseline above, a numbered record, or the schema the SDK generated from (ACP **1.25.0**).

## 1. Answer

**Build acpext as 0042 option A.** Omit it and Magic hand-writes `_gobble/*` JSON (0179-REPORT R3, the `instructions` / `customInstructions` class of bug). Put a package inside gobble-cli's root module and Magic's `go.mod` gains go-llmprovider-sdk (Magic 0185). Put `_gobble/*` in acp-go-sdk2 and the generic SDK carries a vendor.

**acpext is types, names, and two static lists. It is not a second SDK.** The ACP SDK already owns connections, JSON-RPC, initialize gates, `CallExtension` / `NotifyExtension`, `ExtensionMux`, `AuthMethod` constructors, `Authenticate`, closed enums, and OverflowBlock. gobble already uses those. Magic's `acpagent.Host.CallExtension` is the same `CallExtension` with a timeout and `ErrNotImplemented` mapping (`host.go:57-60`, `:211`). Duplicating any of that in acpext would fight both trees.

**Put the full frozen `_gobble/*` contract in the first tag, not only the four methods 0179 wires.** Twenty request names, three notification names, every params/result struct, `AuthProviders()`, `ErrorData`, `CapabilitiesMeta`. Magic's first Spec still only *calls* rename and clone and *handles* retry and compaction (0179 D8, D9). The rest compile for later `Spec.Ops` / `CallExtension` without a second acpext tag. That is the streamline: one contract module, thin Magic wiring.

**Do not put behaviour in acpext.** No `HandleExtensionMethod`, no TokenStore, no 0074, no `Host`, no session-file types. gobble keeps dispatch (`0042-MADR` D4 last bullet; `acpserver/extensions.go:63-77`). Magic keeps `internal/provider/gobble/auth.go` writing gobble's `auth.json`.

**Do not wait on acp-go-sdk2#14.** Constructors and advertised-auth helpers would be niceties for Magic's handshake. They are not required to publish acpext. Pin policy stays: acpext's `require` of acp-go-sdk2 **≤** Magic's pin (0042 D5, 0179 D12 gate 6). Today that is **v1.31.2** on both consumers; newest SDK tag is **v1.33.0**.

## 2. What acp-go-sdk2 already is (v1.33.0)

Schema **1.25.0**. Module path `github.com/maccavelli/acp-go-sdk2`. Floor Go 1.26.10. gobble and Magic are Go 1.27.3, where `encoding/json.RawMessage` is an alias of `jsontext.Value` (0038-REPORT).

### 2.1 Core protocol (do not reimplement)

Typed client and agent connections generated from the schema: `initialize`, `authenticate`, `session/new|load|list|close|prompt|cancel|set_config_option|set_mode`, MCP, permissions, elicitation, slash commands (`availableCommands`), usage, compaction_update, tool_call / tool_call_update, session_info_update.

Gates (v1.32.0 / 0049 S3, S5, S8, S9): capability checks on send; unadvertised `authenticate` `methodId` → `ErrUnadvertisedAuthMethod`; empty `PromptResponse.stopReason` → `-32603`; empty tool-call `content`/`locations` sent as `[]` and clear.

`agentproc` (Command, InProcess, Transcript, LineTap, OverflowBlock). Magic and gobble both use it. **Out of acpext.**

`remote` and `clienthost`. Neither consumer imports `remote`. Magic implements fs itself; gobble's Magic Spec omits fs/terminal (0179 D2). **Out of acpext.**

### 2.2 Extension API (leverage this)

ACP extensions are JSON-RPC methods whose names start with `_` ([extensibility](https://agentclientprotocol.com/protocol/extensibility#extension-methods)). The SDK:

| API | Role |
| --- | --- |
| `ExtensionMethodHandler` | `HandleExtensionMethod(ctx, method, json.RawMessage) (any, error)`; unrecognized → `NewMethodNotFound` (`extensions.go:35-37`) |
| `ExtensionMux` + `Handle[P,R]` / `Notify[P]` | Decode params with `encoding/json`, absent as `{}`; panic on non-`_` names and duplicate register (`extensions.go:39-103`) |
| `ClientSideConnection.CallExtension` | Request `_…`; result `json.RawMessage`; does **not** check initialize (`extensions.go:256-272`) |
| `AgentSideConnection.NotifyExtension` / client `NotifyExtension` | Notification `_…` |
| `SendRequest[T]` | Escape hatch for non-`_` vendor methods; Magic Host also has `RawRequest` for those |

Magic's Host is the Spec-facing wrap of `CallExtension` (`host.go:57-60`). 0179 D8 is `h.CallExtension(ctx, acpext.MethodClone, params, &out)` once types exist.

gobble's **client** already moved notification decode onto `ExtensionMux` (0039-PLAN P4). gobble's **agent** still uses a private `map[string]extension` and `jsonx.Marshal` of the result (`extensions.go:28-77`). 0042 D4 keeps that dispatch: moving it onto `Handle[P,R]` would change how bad params are refused (jsonx vs SDK `decodeParams`). **acpext must not force that move.**

### 2.3 Auth API (leverage this; catalog lives in acpext)

| API | Role |
| --- | --- |
| `NewAuthMethodAgent(id, name)` | Agent method; gobble sets `Description` after (`auth.go:49-51`) |
| `NewAuthMethodTerminal(id, name)` | Terminal method; gobble sets `Args` `--login <id>` (`auth.go:61-63`) |
| `OfferedAuthMethods(caps, methods)` | Drops terminal methods unless the client advertised `auth.terminal` (`helpers.go:740`) |
| `Authenticate` | Client send; v1.32.0+ refuses unadvertised and terminal ids before the write |
| `NewAuthRequired` | Agent `-32000` when no credential (`auth.go:80`) |

Magic 0179 D3: `SafeAuthMethodIDs` = agent ids from `acpext.AuthProviders()`; omit `auth.terminal` so gobble will not advertise `*-login`. Phone 0074 writes gobble's store; then `authenticate` is a presence check (0016-MADR A2). **Writing the store is Magic `auth.go`, not acpext.**

### 2.4 What the SDK must not grow

Vendor `_gobble/*` types. 0179-REPORT R3 already said they do not belong in the generic SDK. Issue 11 (constructors, `Kind()`, `ClientRefusals`) shipped. Issue 14 (Prompt constructors, advertised-id helpers) is optional follow-up.

## 3. What gobble serves today (the wire acpext must freeze)

`acpserver/extensions.go:28-49` — twenty requests, all `_gobble/…`. Frozen set: 0002-PLAN Phase 7, plus 0005 F5-1/F5-2 and 0022 personality.

### 3.1 Requests

| Method | Params (JSON names) | Result | Turn |
| --- | --- | --- | --- |
| `_gobble/steer` | `sessionId`, `message` | `{disposition}` | no |
| `_gobble/follow_up` | same | `{disposition}` | no |
| `_gobble/clear_queue` | `sessionId` | `{steering, followUp}` | no |
| `_gobble/set_steering_mode` | `sessionId`, `mode` | `{}` | no |
| `_gobble/set_follow_up_mode` | same | `{}` | no |
| `_gobble/compact` | `sessionId`, `customInstructions` | `{summary, firstKeptEntryId, tokensBefore, details}` | **claims** |
| `_gobble/usage` | `sessionId` | `{text, stats}` | no |
| `_gobble/set_session_name` | `sessionId`, `name` | `{}` | **claims** |
| `_gobble/fork` | `sessionId`, `entryId` | session id + `_meta` | claims |
| `_gobble/clone` | `sessionId` | `{sessionId}` | **claims** |
| `_gobble/get_state` | `sessionId` | state object | no |
| `_gobble/get_messages` | `sessionId` | `{messages}` | no |
| `_gobble/get_entries` | `sessionId`, `since?` | `{entries}` | no |
| `_gobble/get_last_assistant_text` | `sessionId` | text | no |
| `_gobble/get_session_stats` | `sessionId` | stats | no |
| `_gobble/get_fork_messages` | `sessionId` | messages | no |
| `_gobble/set_auto_retry` | `sessionId`, `enabled` | `{}` | no |
| `_gobble/abort_retry` | `sessionId` | `{}` | no |
| `_gobble/set_auto_compaction` | `sessionId`, `enabled` | `{}` | no |
| `_gobble/set_personality` | `sessionId` + personality params | `{}` | no |

`details` on compact, and `messages` / `entries` arrays, are gobble session-file shapes (`compaction.Details`). 0042 D2: those fields become `json.RawMessage` / `jsontext.Value` in acpext and are **not frozen**. `ForkResult.SessionID` / `CloneResult.SessionID` are `acp.SessionId`.

### 3.2 Notifications (agent → client)

| Method | Params |
| --- | --- |
| `_gobble/retry` | `sessionId`, `phase`, `cause`, `attempt?`, `maxAttempts?`, `delayMs?`, `error?` |
| `_gobble/compaction` | `sessionId`, `phase`, `reason`, `tokensBefore?`, `error?`, `paused?` |
| `_gobble/personality` | `sessionId`, `name`, `level` |

Phases `start`/`end`; compaction reasons `manual`/`threshold`/`overflow`; pauses `turn`/`session`; queue modes `one-at-a-time`/`all`; disposition `queued`. Constants in acpext (0042 D2).

### 3.3 Auth and `_meta` (not `_gobble/*` methods)

- `AuthProvider{ID, Name, Command string; Login bool}` (`auth.go:16-19`). `ID` is the **agent** `methodId`.
- Runtime list from `llm/provider` KeyProviders (`internal/cli/agent.go:370`). Nine keyed: gemini, openai, claude, grok, opencode-zen, opencode-go, huggingface, kilo, together. Ollama has no key. Only grok and kilo have `Login` (0016 A2, 0031 D7: no OpenAI Codex-client OAuth in gobble).
- Terminal ids: `ID + "-login"` only if client sent `auth.terminal`.
- `authenticate` is presence check; no secret on the wire (`auth.go:68-83`).
- Error `data`: `provider`, `reason`, `command`, `error` (`ErrorData`). Codes are SDK's (`NewAuthRequired`, `NewInvalidParams`).
- `initialize` `agentCapabilities._meta.gobble`: `CapabilitiesMeta{Extensions []string}` (the twenty names). Other `_meta.gobble` members (permissions marks, `switchTo`, mentions, feedback) stay in acpserver/acpclient (0042 D2). Magic does not need them for 0179.

### 3.4 Already standard ACP (not acpext)

`session/set_config_option` for grouped `model` / `thought_level` (0208 P7). Slash commands (`/compact` as typed command). `session/load|list|close`. `authenticate`. Image/audio omitted by gobble caps. **Magic must not invent `_gobble/set_model`.**

## 4. What Magic will call (0179, current contract)

0208 P6: empty `Spec.Ops` is mute; grok ops live in `internal/provider/grok`. P7: grouped options + `DangerousModeIDs`. P8: one `Descriptor` per agent; gobble is **not** registered (`registered.go` is fake, grok, opencode, codex, kilo).

0179 chosen option **C** (working-tree amendment): phone 0074 + handshake `authenticate`; D2 omits fs/terminal.

| 0179 | Magic uses | acpext name |
| --- | --- | --- |
| D3 catalog / SafeAuthMethodIDs | `AuthProviders()`, `ErrorData` | `auth.go` |
| D8 Rename | `_gobble/set_session_name` | `MethodSetSessionName`, `SetSessionNameParams` |
| D8 Fork | `_gobble/clone` (whole session; not `_gobble/fork`) | `MethodClone`, `SessionParams`, `CloneResult` |
| D9 | `_gobble/retry`, `_gobble/compaction` | `MethodRetry`, `RetryNotice`, `MethodCompaction`, `CompactionNotice` + phase/reason constants |
| D11 | `_meta.gobble` extensions list | `MetaKey`, `CapabilitiesMeta` |
| D7 | `bypass` mode | **SDK session modes**, not acpext |
| Confirmation | `DangerousModeIDs: []string{"bypass"}` | Spec field |

0179 explicitly **does not** wire steer, follow_up, compact-as-extension, usage, fork-by-entry, get_*, personality, queue modes. Those remain typed in acpext so a later Magic PLAN can `CallExtension` without waiting on gobble.

Magic **must not** import acpext from `acpagent` (0208 D2, 0179 D13). Only `internal/provider/gobble`.

## 5. Exact acpext inventory (first tag `acpext/v0.1.0`)

Match 0042-PLAN P1 files. One package, exported names with docs, `golint` clean.

### 5.1 `go.mod`

```text
module github.com/maccavelli/gobble-cli/acpext
go 1.27.3
require github.com/maccavelli/acp-go-sdk2 v1.31.2
```

`check-acpext-deps`: `go -C acpext list -deps` names only the standard library, acpext, and acp-go-sdk2 (and the SDK's own requirements). Seen failing on `jsonx` and on `golang.org/x/sys/unix`.

**Do not** require go-llmprovider-sdk, go-tui-lib, gobble-cli, jsonx, Kong.

`acp.SessionId` is the only SDK type the params need. Prefer `acp.SessionId` over `string` for `sessionId` so Magic and gobble share the identifier type.

JSON tags only. Round-trip tests run **both** `encoding/json` and `encoding/json/v2` (0042-PLAN P1 step 4). gobble continues to encode through `jsonx` (0039). Magic Host uses the SDK's encoder. Tags must survive both.

Opaque fields: `json.RawMessage` (alias of `jsontext.Value` at 1.27). Document that they are session-file shapes, not the contract.

### 5.2 `methods.go`

- 20 `Method…` constants, values exactly the map keys in `extensions.go:28-49`.
- 3 notification constants (`_gobble/retry`, `_gobble/compaction`, `_gobble/personality`).
- `Requests() []string` sorted, distinct, each `_gobble/`.
- `Notifications() []string` sorted.

No `"_gobble/` literals outside acpext after P2 (0042-PLAN P2 step 9), except tests.

### 5.3 `types.go`

Every params/result struct from §3, named as 0042 D2 (`SetSessionNameParams`, `CloneResult`, `SessionParams`, `RetryNotice`, `CompactionNotice`, …). Shared `SessionParams` for session-only methods.

Typed string constants: phases, compaction reasons/pauses, retry causes, queue modes, `DispositionQueued`.

`MetaKey` (the `_meta` member name `gobble`). `CapabilitiesMeta`. `ErrorData` with `omitzero` on each field.

**No** permission-mark structs, `switchTo`, mentions, feedback. Those are gobble-cli ↔ gobble-cli.

### 5.4 `auth.go`

`AuthProvider`, `LoginSuffix = "-login"`, `AuthProviders() []AuthProvider` **static**, registry order, nine entries, `Login` true only for grok and kilo.

Root test (P2): `AuthProviders()` equals `internal/cli` `authProviders()` derived from `llm/provider`. That test lives in the **root** module (it may import llm). acpext itself never imports llm. Adding an SDK provider is: update the static list, tag acpext, `go get`, then the equality test passes (0042 consequence).

Do **not** put `Has`, `AuthOptions`, or `NewAuthMethod*` wrappers that set Description/Args. gobble `auth.go` already calls `acp.NewAuthMethodAgent` / `Terminal`. Magic builds 0074 catalog rows from `AuthProvider` (id, name, `Login` → offer `oauth_device` for grok/kilo, `api_key` for all nine, never `oauth_browser` as completable).

### 5.5 Optional `call.go` (same tag, still SDK-only)

Thin functions on `*acp.ClientSideConnection`:

```go
func Clone(ctx context.Context, c *acp.ClientSideConnection, p SessionParams) (CloneResult, error)
```

Implemented as `c.CallExtension(ctx, MethodClone, p)` plus `json.Unmarshal` into the result. **gobble `acpclient` should use these in P2.** Magic **cannot**: Specs see `Host`, not `ClientSideConnection`. Magic stays `h.CallExtension(ctx, acpext.MethodClone, p, &out)`.

Do not take a `func(ctx, method, params, out any) error` to abstract Host. That is a second, worse API.

Do not add `Register(mux *acp.ExtensionMux, …)` that binds gobble behaviour. That belongs in acpserver if a later record moves dispatch.

### 5.6 Tests in acpext (no gobble import)

- One recorded frame per method from `session.golden` / existing tests; decode + encode value-equal under v1 and v2 JSON.
- `Requests` / `Notifications` / `AuthProviders` lists.
- Fail-first: rename JSON `name`→`title`; drop `omitzero` on `RetryNotice.Attempt`; drop a request; drop a provider.

## 6. What Magic `internal/provider/gobble` builds on top (not acpext)

This is 0179 P1, after `go get acpext@v0.1.0`:

1. Spec: `DefaultBin gobble`, args `["acp"]`, no fs/terminal/`auth.terminal`, `KnownGoodVersion` = gobble **product** tag that `go version -m`s this acpext.
2. `SafeAuthMethodIDs` from `AuthProviders()` ids; `HasHeadlessCredential`; `AuthCatalogList` / `SetCredential` / `ClearCredential` / `StartDeviceAuth` (grok, kilo device only) writing gobble's `auth.json`.
3. `Ops.Rename` / `Ops.Fork` via Host + acpext types.
4. `ExtensionNotifications` for retry → `Host.Notice`, compaction → `Host.UpdateCompaction`.
5. `DangerousModeIDs: []string{"bypass"}` at 0179 P3 (field exists: 0208 P7 on master).
6. Fixtures recaptured from that gobble tag (D11). acpext says what gobble *means*; the capture says what the release *sent*.

Phone 0074 screens already exist. No Dart branch on `"gobble"`. No `go-llmprovider-sdk` in Magic's graph (`go list -m all` after the get).

## 7. Streamline vs richness

| Put in acpext | Leave out |
| --- | --- |
| All 23 method names and types | Behaviour, dispatch, jsonx |
| Static `AuthProviders` + `ErrorData` | TokenStore, 0074, `Has` |
| Opaque `RawMessage` for session-file blobs | Frozen session-file structs |
| Optional SDK `CallExtension` wrappers | `Host` wrappers, `ExtensionMux` registration of gobble |
| Round-trip frames | Live `gobble acp` inside acpext tests |

Richness for Magic is **availability**: compact, usage, steer, fork-by-entry, personality are in the module on day one. 0179 stays thin. A later Magic PLAN adds `Ops.Compact` → `MethodCompact` without gobble shipping a new acpext field.

Omitting types "until Magic needs them" forces an acpext tag per Magic feature and re-opens D5 ordering. That is the opposite of streamlined.

## 8. Pin and tag order (unchanged, still the gate)

1. 0042 P0 records.
2. P1 module + tests + `check-acpext-deps`. Owner tags **`acpext/v0.1.0`**. Proxy serves it.
3. P2: root `go get …/acpext@v0.1.0`, no `replace`; gobble serves from it; goldens **byte-identical**.
4. gobble **product** tag (after v0.1.1) whose binary names that acpext.
5. Magic 0179 P1 `go get` the same acpext tag; recapture from that gobble tag; both trees on the **same** acp-go-sdk2 tag.

Tagging gobble **before** step 2 cannot satisfy 0179 D12 gate 6.

acpext may stay on acp-go-sdk2 v1.31.2 if Magic moves to v1.33.0 (require ≤ Magic). gobble **root** should match Magic (0184 D12). v1.32.0+ empty `PromptResponse` and unadvertised authenticate are gobble/Magic test issues, not acpext's.

## 9. Risks this report does not close

- Static `AuthProviders()` vs live registry: equality test is the backstop; a forgotten acpext tag is a failed gobble `preflight`, not a silent Magic catalog hole.
- `ExtensionMux` vs jsonx dispatch: leaving dispatch in acpserver is a deliberate wire-stability choice (0042 D4). Revisit only with a PLAN that recaptures goldens.
- Magic Host timeout around `CallExtension`: acpext wrappers on `ClientSideConnection` will not share that timeout. Specs must keep using Host.
- 0179 D13 lives in Magic's **working tree** as of this report's Magic origin `e2d6493b` plus uncommitted 0179 amendment. If that amendment is discarded, Magic origin is still option A (no authenticate). acpext's `AuthProviders()` remains the right catalog either way.

## 10. What to do with 0042

This report supports **keeping 0042 option A** and **not shrinking P1 to 0179's four methods**. Optional amendment when 0042 executes: add `call.go` wrappers to P1's file list; keep dispatch out of acpext (already D4). P0 still waits on "do P0".
