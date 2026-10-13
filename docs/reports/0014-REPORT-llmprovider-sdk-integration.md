---
status: observation
date: 2026-10-13
subject: integrating go-llmprovider-sdk as gobble's provider and model layer
examines: "gobble-cli at 051fead (main, clean); go-llmprovider-sdk at 4e81764 (main, v1.4.0-10, 0031-PLAN in-progress); gobble pin github.com/maccavelli/go-llmprovider-sdk v1.2.1"
---

# REPORT 0014 — Integrating go-llmprovider-sdk as gobble's model layer

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

This report records findings. It decides nothing. Each finding names the
gobble record it bears on; any choice among the options in F9 belongs in a
MADR in this repository.

The request that opened it: gobble will import the SDK for its model
layer, with tight API integration and native support, the SDK as the core
provider and model layer, and gobble kept lean. The question is the
impact on both trees as they stand, and whether that integration should
proceed now or wait.

Evidence is marked **(R)** read in source or records, or **(M)** measured
by a command against the two working trees on 2026-10-10. Live provider
calls were not re-run.

## Summary

gobble already talks to models only through go-llmprovider-sdk. The
import lives in `llm/provider`, pinned at **`v1.2.1`**. The SDK's latest
tag is **`v1.4.0`**. Ten commits past that tag, on SDK `main`, native
Responses streaming is implemented for `openai`, `grok`, and OpenCode's
`responses` route. That work is 0031-PLAN Phases 0–4; Phases 6–7 (the
conformance check and the `v1.5.0` tag) have not run. gobble's adapter
wraps every provider in `llmprovider.WithRetry`, which the SDK documents
as stripping native streaming. (R, M)

The destination the request names is already this repository's D10:
[0004-MADR-go-module-architecture.md](../decisions/0004-MADR-go-module-architecture.md)
makes `llm/provider` the only importer and `llm` the agent-facing facade.
What is not done is F4's catalog, auth, wizard, and live-id coverage, and
the adapter change 0004-MADR already wrote for the day a provider gains a
`Streamer`: retry only before the first event, so `WithRetry` cannot hide
the stream. (R)

**What the evidence supports.** Wait to deepen the integration until the
SDK tags the 0031 streaming release. In the same gobble change that pins
that tag, run F4 as native consumption of `catalog`, `auth`, and `wizard`,
and unwrap retry so `Streamer` is live. Keep the `llm` facade until the
SDK carries a per-tool `strict` flag (already a gobble blocker), image
items, and a tool-output error bit. Continue 0005-PLAN F3 (prompts,
skills, templates) in the meantime: it does not import the SDK. Do not
fill `llm/catalog` or `internal/auth` as gobble-owned duplicates of SDK
packages that already exist.

A pin-only bump to `v1.4.0` is API-compatible with today's adapter and
buys no native streaming. Pinning SDK `main` before `v1.5.0` would make
`adapter.Capabilities` report `NativeStreaming: true` while `Stream`
still waits on `Generate`. (R)

## Scope and method

- **gobble-cli** at `051fead` (2026-10-10 12:43 -0500, `main`, clean).
  Packages read: `llm`, `llm/provider`, `llm/llmtest`, `llm/catalog`,
  `agent`, `compaction`, `acpserver`, `internal/cli`, `internal/auth`,
  `internal/archtest`; records 0004, 0005, 0012, architecture.md,
  go.mod.
- **go-llmprovider-sdk** at `4e81764` (2026-10-10 16:11 -0500, `main`,
  `v1.4.0-10-g4e81764`). Packages read: `llmprovider` (contract, stream,
  retry, items, tools), `providers/{openai,grok,opencode}`, `catalog`,
  `auth`, `wizard`; records 0015, 0030-REPORT, 0031-MADR/PLAN,
  architecture.md, api-standards.md. Three files in that tree were
  staged and uncommitted (0031-PLAN Phase 5); they are named as such and
  are not treated as a release.
- **Counts.** Physical lines of non-test Go, `wc -l`. Git tags and
  `git log v1.2.1..HEAD`. `python3 scripts/check_records.py --next`
  printed `0014`.

## What exists

### gobble's model layer (R, M)

| Package | Lines (non-test) | Role today |
| :--- | ---: | :--- |
| `llm` | 284 | Stable facade: Pi-shaped `Message`/`Content`, sealed events, typed errors. Imports no SDK. |
| `llm/provider` | 373 | The only SDK importer. `New` / `Wrap` / `Ambient`. |
| `llm/llmtest` | 251 | Scripted `llm.Provider` for agent and compaction tests. |
| `llm/catalog` | 5 | Placeholder `doc.go`. |
| `internal/auth` | placeholder | Placeholder `doc.go`. |

`go.mod` requires `github.com/maccavelli/go-llmprovider-sdk v1.2.1`.
`internal/archtest` rule 4 fails any other package that imports it.
`agent`, `compaction`, and `acpserver` program against `llm.Provider` and
`llm.Message`. The CLI builds the process provider with
`provider.Ambient` (`internal/cli/agent.go`).

0005-PLAN's next executable phase is **F3** (context, prompts, skills,
templates). F1 (tools) and F2 (sessions) are complete. F4 (providers,
catalog, auth) is specified and not started as a phase; 0002-PLAN Phase 3
already landed the adapter and the ambient credential path.

### The SDK (R, M)

One module, Go 1.27.2, standard library plus `golang.org/x/term` in
`wizard` only. No binary. Apache-2.0. Tags through `v1.4.0`
(`e5772a5`, 2026-10-09). 47 commits after gobble's pin `v1.2.1`. (M)

Exported packages: `llmprovider`, `llmprovider/providers` and
`providers/<id>`, `llmprovider/auth`, `llmprovider/catalog`,
`llmprovider/llmtest`, `wizard`. Ten built-in ids. Five generation
wires. `api-check` against the latest `v1.*` tag.

Non-test Go: about 18,357 lines under `llmprovider/`, 2,335 under
`wizard/`. (M)

The public generation contract is `ID`, `Capabilities`,
`Generate(ctx, *Request) (*Response, error)`. `Stream(ctx, p, req)` is a
package function: a `Streamer` when present, otherwise `Generate`'s
result as events. (R)

### Version gap (M)

| Pin | What gobble would see |
| :--- | :--- |
| `v1.2.1` (gobble today) | All ten ids on the v1 contract. `NativeStreaming: Unsupported` on every built-in. `Tool` is `{Name, Description, Schema}`. Usage decoded. Auth in `llmprovider/auth`. |
| `v1.4.0` (latest tag) | Same public contract. Heuristic, retry, redaction, catalog-cache, and Go 1.27.2 work. Streaming still `Unsupported`. |
| `4e81764` (SDK `main`) | `openai` and `grok` implement `Streamer` and declare `Supported`; OpenCode declares `BestEffort` and streams on the `responses` route. `WithRetry` still reports `Unsupported` and buffers. No `v1.5.0` tag. 0031-PLAN Phases 6–7 not executed. |

---

## F1 — The adapter is already the integration (R)

`llm/provider.New` constructs with `providers.New`, turns model probes
and metadata fetch off, sets `WithClientInfo`, wraps `WithRetry` at Pi's
bounds (`MaxAttempts: 4`, `BaseDelay: 2s`, `MaxDelay: 60s`), and returns
an `llm.Provider`. `Stream` calls `llmprovider.Stream` and maps:

- `text_delta` → `llm.TextDelta`
- `reasoning_delta` → `llm.ThinkingDelta`
- `item` function call → `llm.ToolCallDone`
- `item` reasoning → `llm.ThinkingDelta` with `ProviderData`
- `done` → `llm.Usage` then `llm.Done`

Sentinels are mapped most-specific first because the SDK's overlap
(`ErrQuotaExhausted` beneath `ErrRateLimited`, `ErrContextOverflow`
beneath `ErrInvalidRequest`). An `*llmprovider.APIError` becomes
`*llm.APIError` with `Status` and `RetryAfter`. (R:
`llm/provider/provider.go`)

Ambient selection walks `providers.Default().Descriptors()` in registry
order and takes the first built-in whose standard variable is set, with
its first static model. That is the process provider today. (R:
`llm/provider/ambient.go`, `internal/cli/agent.go`)

**Bears on:** 0004-MADR D10, 0002-PLAN Phase 3, 0005-PLAN F4 step 1.

## F2 — Two type systems sit between the session and the wire (R)

gobble's `llm` is Pi-shaped. The SDK is item-shaped.

| gobble `llm` | SDK `llmprovider` |
| :--- | :--- |
| `Provider.Stream(ctx, *Request) iter.Seq2[Event, error]` | `Provider.Generate`; `Stream` is a package function; `Streamer` is optional |
| `Request.Messages []Message` with `Role` including `tool` | `Request.Instructions` + `Request.Input []Item` |
| `Content` blocks: text, image, thinking, tool_call, tool_result (`IsError`) | Sealed items: `MessageItem`, `FunctionCallItem`, `FunctionCallOutputItem`, `ReasoningItem` |
| `Capabilities.{ImageInput, NativeStreaming}` booleans | `Capabilities` with `Support` per Tools / ForcedToolChoice / Reasoning / Continuation / NativeStreaming |
| `Usage` with cache-read, cache-write, `Estimated` | `Usage` with input (includes cached), output (includes reasoning); no write count, no presence bit |
| Sealed events include `ToolCallStart` / `ToolCallDelta` | `Event` is a struct with `Type`; call arguments arrive as a finished `EventItem` |
| Thinking levels `off`/`minimal`/`low`/`medium`/`high`/`xhigh`/`max` | `Effort` `low`/`medium`/`high`/`xhigh` |

The conversion is lossy in both directions, in code that already runs:

1. **Tool-result errors.** `agent` sets `llm.Content.IsError` on a failed
   or cancelled tool (`agent/agent.go`). `toRequest` copies only
   `CallID` and `Output` onto `FunctionCallOutputItem`. The SDK type has
   no error bit, and the adapter does not mention `IsError`. (R)
2. **Images.** `llm.Content` accepts `ContentImage`. The adapter has no
   image case. `Capabilities.ImageInput` stays false, matching 0004-MADR
   and 0005-PLAN. (R)
3. **Tools round-trip through JSON.** The agent encodes specs as
   `jsontext.Value`; the adapter unmarshals `{name, description,
   inputSchema}` into `[]llmprovider.Tool`. (R)
4. **Usage estimation.** An all-zero SDK `Usage` is replaced by a
   four-bytes-per-token estimate with `Estimated: true`. Cache-write
   stays zero. The Messages wire already decodes
   `cache_creation_input_tokens` internally and folds it into
   `InputTokens` (`llmprovider/internal/wire/messages/messages.go`). (R)
5. **Session persistence is Pi v3.** `acpserver/history.go` and
   `compaction/serialize.go` convert `llm.Message` ↔ `session.Message`.
   That boundary exists whatever the provider SDK is. (R)

The original reason 0004-MADR gave for the facade — SDK S8 would remove
the old `Generate*` API, so embedders must not import the module — is
closed. The SDK is on a tagged v1 with `api-check`. The remaining reason
the facade is wider than the SDK is the table above, plus embedder
isolation from SDK minor bumps (archtest rule 1 and 4). (R)

**Bears on:** 0004-MADR package map and 2026-10-01 amendment (facade
wider on purpose), 0005-MADR stream-shape row.

## F3 — Native streaming is on SDK `main` and gobble's retry wrap hides it (R, M)

SDK 0031-MADR (accepted 2026-10-10) streams the Responses wire first.
0031-PLAN execution on `main`:

- Phases 0–4 committed: decoder, stream seam, `openai`/`grok`/`opencode`
  `Stream` methods, live checks (OpenAI 9 text deltas; Grok 9; OpenCode
  Go 2; OpenCode Zen skipped 403).
- Phase 5 (`WithRetry` doc and the buffered-stream guard) is staged,
  uncommitted.
- Phases 6–7 (conformance `Streamer` check, docs, **`v1.5.0` tag**) have
  no execution record.

On `4e81764`, `NativeStreaming` is `Supported` for `openai` and `grok`,
`BestEffort` for both OpenCode ids, `Unsupported` for the rest. (M)

gobble's `New` stores the **inner** provider's `Capabilities` and then
replaces the inner value with `WithRetry(p, …)`. `Stream` is called on
the wrapper. SDK `WithRetry` is not a `Streamer`; its `Capabilities`
force `NativeStreaming: Unsupported`; `llmprovider.Stream` over it runs
`Generate` and emits the whole reply. 0031-MADR D6 says a caller that
wants live deltas streams from `p` itself. (R: `llm/provider/provider.go`
`New`, SDK `retry.go`)

0004-MADR's 2026-10-01 amendment already records the gobble-side fix:
when any provider gains a `Streamer`, retry moves into the adapter and
retries only before the first event. That code is not written. At
`v1.2.1` every built-in is `Unsupported`, so the stored caps and the
wrapper agree. After a pin to SDK `main`, `adapter.Capabilities` would
report `NativeStreaming: true` (inner caps) while `Stream` still waited
on `Generate`. (R)

Function-call argument deltas are silent on the Responses stream
(0031-MADR Q4). gobble's `ToolCallStart` / `ToolCallDelta` events remain
unproduced. ACP `session/update` token chunks for text become possible
for the three streaming ids once the adapter unwraps retry; tool-call
streaming stays whole-item. (R)

**Bears on:** 0004-MADR streaming notes, 0005-MADR "Stream shape" row
("a later SDK `Streamer` is picked up with no gobble API change"),
0005-PLAN F4 accept (text case asserts one `TextDelta` while native
streaming is unsupported).

## F4 — Catalog, auth, and wizard are implemented in the SDK and empty in gobble (R)

SDK `llmprovider/catalog` lists, ranks, searches, and carries
models.dev-format metadata (cost, family, reasoning).
`llmprovider/auth` has `OAuthSession`, `FileTokenStore` with a
cross-process refresh lock, browser PKCE, device-code, Kilo device,
`CommandToken`, `VendorCLISession`. `wizard.ConfigureLLM` walks
provider → credential → model over a `Prompter`. (R: SDK
architecture.md)

gobble F4 as written in 0005-PLAN:

- `llm/catalog`: embedded snapshot and `models.json` overlay, Pi ids
  mapped to SDK ids.
- `internal/auth`: keyring plus 0600-file fallback, implementing
  `llmprovider.TokenStore`.
- `gobble auth login|logout|status|token` and ACP `authMethods`.
- `gobble models list|info`.

0005-MADR choice 11 (2026-10-07) already says the catalog's data stays
the SDK's; gobble overlays only what the SDK lacks. 0008-MADR maps
configure, login, and model list onto the SDK wizard, auth, and catalog,
imported through `llm/provider` (rule 4). (R)

Filling those gobble packages from Pi snapshots, or writing a second
OAuth loopback, would duplicate ~2,300 lines of wizard plus catalog and
auth that the SDK already tests, including live-tagged suites. That is
the opposite of a lean gobble. (R, M)

**Bears on:** 0005-PLAN F4 steps 5–8, 0008-MADR configure/login/catalog
row, 0004-MADR D10.

## F5 — Gaps that still force gobble-side width (R)

Recorded in gobble, still true of SDK `4e81764`:

| Gap | Where it is recorded | SDK today |
| :--- | :--- | :--- |
| Per-tool `strict` flag | 0012-MADR D3, D13; 0012-PLAN P7 | `Tool` is `{Name, Description, Schema}`. No `strict` field. |
| Native tool deferral (`defer_loading`, tool references) | 0012-MADR D13, 0011-REPORT T6 | No fields. gobble-side `tool_search` changes the `tools` list and breaks the prompt cache. |
| Cache control (`cache_control`) | 0012-MADR D13 | Not on `Request` or `Tool`. |
| Image input | 0004-MADR, 0005-PLAN F1 | No image `Item`. |
| Tool-output error bit | 0004-MADR 2026-10-01 content gaps | `FunctionCallOutputItem` has `CallID` and `Output` only. |
| Cache-write token count | 0004-MADR 2026-10-02 | Folded into `InputTokens` on the Messages wire; public `Usage` has no write field. |
| Generic Chat Completions constructor | 0005-MADR unnamed OpenAI-compatible row | Named ids only. `WithBaseURL` on `openai` speaks Responses. |
| Native streaming on Messages, Chat Completions, Gemini | 0031-MADR D8, out of scope | Responses only. Claude, Gemini, Hugging Face, Kilo, Together, Ollama stay `Unsupported`. |

*(2026-10-13, 0019-MADR F9: the strict flag and the tool-output error bit shipped at `v1.7.0`, native streaming at `v1.6.0`, and cache writes at `v1.10.0`. Still open: deferral, `cache_control`, image input and a generic Chat Completions constructor.)*

The first row is a **blocker** for a gobble decision already taken:
tools with optional arguments (read, bash, powershell) are to be sent
`strict: false`. Until the SDK field exists, 0012-PLAN P7 cannot run, and
OpenAI fills in optional arguments. (R: 0012-MADR D3)

**Bears on:** 0012-MADR D3/D13, 0012-PLAN P7, 0005-MADR 1.x unnamed
endpoints.

## F6 — Impact on gobble if integration deepens now (R)

Three depths, with cost as they stand.

### A. Pin bump only (`v1.2.1` → `v1.4.0`)

Compatible with the current adapter. No `Streamer` on built-ins, so
`WithRetry` and stored caps still agree. gobble would pick up retry,
error-classification, redaction, and catalog-cache fixes. Catalog, auth,
and wizard remain unused. Native streaming remains synthesised. This is
the 0004-PLAN "fleet library pins refreshed" shape of change. Low
product value on its own.

### B. Native consumption, facade kept (the 0004-MADR shape, executed)

Same pin as C below, plus F4:

- `llm/provider` constructs, streams, retries, and maps as today, with
  the 0004-MADR Streamer unwrap once a streaming tag exists.
- `llm/catalog` is an overlay and a Pi↔SDK id table over
  `catalog.List` / `catalog.Search` / metadata, not a second catalog.
- `internal/auth` implements `TokenStore` (and `RefreshLocker`) and
  hands sessions to `auth.LoginBrowserOAuth` / `StartDeviceOAuth`.
- Configure and `gobble models` go through `wizard` and `catalog` via
  `llm/provider` (rule 4).
  *(2026-10-13, 0020-MADR D2, D5: the ACP model option lists through `catalog` from 0020-MADR (0020-PLAN P2, `a45dad6`).)*
- `llm` stays the type `agent`, `compaction`, and embedders see.
- Tools become `[]llmprovider.Tool` at the adapter boundary without a
  JSON round-trip.

Touched gobble packages: `llm/provider`, `llm/catalog`, `internal/auth`,
`internal/cli` (auth and models verbs), `acpserver` (authMethods,
set_config_option). `agent` and `session` stay on `llm`. Archtest rule 4
unchanged. This is F4 as already written, plus the Streamer unwrap.

### C. Collapse `llm` onto SDK types

`agent.Config.Provider` becomes `llmprovider.Provider`; history becomes
`[]Item`; events become `llmprovider.Event`. `llm` and `llm/llmtest`
shrink or go. Archtest rules 1 and 4 change: `agent`, `compaction`, and
`acpserver` import the SDK.

Cost on this tree:

- `agent/agent.go` (513 lines) and its tests (572) are written in
  `llm.Message` and sealed events.
- `acpserver/history.go` (426) and `acpserver/agent.go` (884) persist Pi
  v3 from `llm.Message`.
- `compaction` summarises through `llm.Provider.Stream`.
- `llm/llmtest.Script` is the fake every agent and compaction test uses;
  SDK `llmprovider/llmtest.Fake` is `Generate`-shaped.

The Pi session boundary remains. The conversion moves from
`llm/provider` into `acpserver`/`compaction`, which is more packages
importing the SDK, not fewer lines overall. Embedders of `gobble` /
`agent` take `go-llmprovider-sdk` as a required module. A later SDK
`Item` (image, error bit) then changes gobble's public agent API unless
gobble still wraps it. That fights 0004-MADR's "embedders program
against `llm`" and the stable-tier promise on `agent` and `llm`.

**Bears on:** 0004-MADR import rules 1 and 4, 0005-PLAN F4, public SDK
tiers.

## F7 — Impact on the SDK (R)

gobble is named as a consumer in 0031-PLAN ("Consumers: prepare-commit-msg
and gobble-cli move under their own records"). No gobble change is in
scope there.

Work the SDK would take, for gobble's already-recorded requests, each as
a field on `Request` / `Tool` / `Item` (api-standards R5, R6, R9):

1. **`Tool.Strict *bool`** (or equivalent), sent on the Responses
   `strict` key. Unblocks 0012-PLAN P7. Additive; `api-check` would
   accept it on v1.
2. **`FunctionCallOutputItem` error bit**, mapped to Anthropic
   `is_error` and the equivalent on other wires.
3. **Image `Item`**, so ACP `promptCapabilities.image` can become true.
4. **`Usage` cache-write**, already decoded on Messages, not exported.
5. **Native deferral and `cache_control`**, 1.x in gobble, larger SDK
   work.
6. **Finish 0031 Phases 6–7 and tag `v1.5.0`**, so gobble can pin a
   streaming release rather than `main`.

None of those require gobble source in the SDK. The SDK's dependency
rule (stdlib + `x/term`) is compatible with gobble's "no vendor LLM
SDK" rule. gobble pulling `wizard` pulls `golang.org/x/term`, which
gobble already requires. (R)

**Bears on:** SDK 0015-MADR R5–R9, gobble 0012-MADR D13, gobble 0004-MADR
rule 4.

## F8 — Tightness that is available without waiting (R)

These keep gobble lean and do not depend on a new SDK tag:

- Do not implement a gobble-owned model catalog or OAuth. When F4
  starts, import the SDK packages through `llm/provider`.
- Do not add provider ids, wires, or a Chat Completions client in
  gobble. 0005-MADR already routes those through the SDK.
- Do not grow `llm.Capabilities` past what the adapter can send.
- Leave `llm/llmtest` as the agent-facing fake; it is 251 lines and
  keeps `agent` off the SDK.

Doing those now is a constraint on F4, not a code change.

## F9 — Now or wait (R, M)

### Option 1 — Deepen now, against `v1.2.1` or `v1.4.0`

Start F4, write catalog/auth/wizard adapters, keep synthesised streams.

- Good, because F4 is specified and the ten ids already construct.
- Good, because gobble would stop being a process that only has ambient
  API keys.
- Bad, because the adapter would be written around `Generate`-fallback
  streams and then rewritten for `Streamer` a tag later (0004-MADR
  already names that rewrite).
- Bad, because P7 (`strict`) still cannot land.
- Bad, because a gobble catalog written against `v1.2.1` static models
  would ignore `v1.4.0`'s metadata cache and ranking work.

### Option 2 — Wait for SDK `v1.5.0`, then one F4 change that is native

Continue F3. Do not fill `llm/catalog` or `internal/auth`. When 0031-PLAN
Phase 7 tags `v1.5.0`, pin it, unwrap retry, consume catalog/auth/wizard,
map live `text_delta` to ACP updates for the streaming ids.

- Good, because native streaming, retry policy, and F4 land once.
- Good, because gobble pins a release, matching 0004-MADR's "no gobble
  tag on a pseudo-version" rule (the spirit of it: consume tagged SDK
  API).
- Good, because F3 does not import the SDK, so gobble is not blocked.
- Neutral, because ChatGPT/Codex login and `gobble models` wait on F4.
- Neutral, because Claude and Gemini remain whole-reply streams until
  later SDK wires.
- Bad, because 0031 Phases 6–7 are not done; the wait is open-ended in
  this tree. The code for Phases 0–4 is on `main`.

### Option 3 — Collapse `llm` onto SDK types now

- Good, because one item model from adapter to wire.
- Bad, because Pi v3 and ACP still need a conversion, which then lives
  in `acpserver` and `compaction` and pulls the SDK into the core
  packages 0004-MADR isolates.
- Bad, because the SDK is still missing image, `IsError`, and `strict`,
  so the collapsed API would be narrower than gobble's session format
  and would grow again.
- Bad, because `agent` is a stable-tier package; changing its `Provider`
  to `llmprovider.Provider` is a public API change before `v1.0.0`.

### Option 4 — Pin SDK `main` now for streaming

- Bad, because `New`'s stored inner caps plus `WithRetry` would disagree
  (F3).
- Bad, because 0031 Phase 7 has not tagged, and gobble's own pin policy
  prefers a release tag.
- Bad, because Phase 5 is uncommitted in the SDK tree as observed.

### What the evidence supports

**Option 2.** Wait for a tagged streaming release. Treat F4 as the
integration change, native on catalog/auth/wizard and on `Streamer`.
Keep the `llm` facade (reject option 3) until the SDK grows the three
fields in F5 that gobble already persists or has decided to send
(`strict`, images, tool-output errors). Continue F3. Do not start a
gobble-owned catalog or OAuth. A `v1.4.0` pin-only bump can ride along
with any other `go.mod` change; it is not the integration.

The SDK-side enabler for gobble's already-accepted D3 is still
`Tool.Strict`. That can land in the streaming release or the one after;
P7 stays blocked until it does.

## Confirmation this report used

- `git -C go/gobble-cli rev-parse HEAD` → `051fead…`
- `git -C go/go-llmprovider-sdk describe --tags` → `v1.4.0-10-g4e81764`
- `git -C go/go-llmprovider-sdk tag --sort=-v:refname` → `v1.4.0` … `v1.0.0`
- `python3 go/gobble-cli/scripts/check_records.py --next` → `0014`
- `rg NativeStreaming: go/go-llmprovider-sdk/llmprovider/providers`
- `wc -l` on the packages in "What exists"

No `go test` of a changed adapter was run: this report changes no Go.

## Not verified

- Whether a gobble adapter that streams the inner provider and retries
  only before the first event preserves 0021-MADR D1's once-only
  after-reply retry. That is F4 work, against a tagged SDK.
- Live ACP `session/update` chunking through gobble's CLI on a native
  stream.
- Whether SDK `catalog` metadata supplies every field 0005-MADR's
  catalog row names (context window, image limits, thinking map). Choice
  11 says gobble overlays what it lacks; the overlay size was not
  measured against `v1.4.0` metadata.
- The three uncommitted SDK files (`0031-PLAN`, `retry.go`,
  `retry_0031_test.go`).

## Bears on

- [0004-MADR-go-module-architecture.md](../decisions/0004-MADR-go-module-architecture.md)
  D10, import rules 1 and 4, Streamer-unwrap note
- [0004-PLAN-go-module-architecture.md](../decisions/0004-PLAN-go-module-architecture.md)
  Phase 2 steps 10, 12, 13 (providers)
- [0005-MADR-v1-feature-scope.md](../decisions/0005-MADR-v1-feature-scope.md)
  models/providers/auth table, stream shape
- [0005-PLAN-v1-feature-scope.md](../decisions/0005-PLAN-v1-feature-scope.md)
  F3 (next), F4 (this report's subject)
- [0008-MADR-native-cli-mode.md](../decisions/0008-MADR-native-cli-mode.md)
  configure / login / catalog
- [0012-MADR-tool-schemas-descriptions-and-loading.md](../decisions/0012-MADR-tool-schemas-descriptions-and-loading.md)
  D3, D13
- [0012-PLAN-tool-schemas-descriptions-and-loading.md](../decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md)
  P7
- go-llmprovider-sdk `0031-MADR-native-streaming-on-the-responses-wire.md`
  and `0031-PLAN-native-streaming-on-the-responses-wire.md`
- go-llmprovider-sdk `0030-REPORT-v1-api-and-protocol-surfaces.md`
