---
status: accepted
date: 2026-10-29
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Adopt what go-llmprovider-sdk v1.23.0 and v1.24.0 added: one registry, warnings, learned refusals, Chat Completions servers, a facts search and images

## Context and Problem Statement

The owner asked on 2026-10-29 for an assessment of how gobble adopts go-llmprovider-sdk "in the most adaptable,
compatible, idiomatic, and standards adherent manner possible", and for the work still needed "for a full adoption and
for the removal of duplicate code". The owner then asked for this record: "draft 0035 … be thorough. analyze
codebases. ground in facts. focus on adherence to standards and best practices, being extensible, flexible, and
efficient."

Number 0035 was taken while this record was being researched. Another session wrote
`0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` at 20:14 on 2026-10-29. That record is
`proposed`, has no PLAN yet, and is not yet committed, so this record names it as code text, not as a link. *(2026-11-06:
it is committed and `accepted`, [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md), and its PLAN is too; More Information links them.)* It covers
the four sibling libraries. For go-llmprovider-sdk it decides:

- the pin, `v1.24.1` (its D1);
- the offline row that the pin needs (D3);
- removing gobble's copies of the empty-answer and offline rules, and sending `max` as `EffortMax` (D4);
- keeping the Anthropic workspace (D5).

It leaves the rest of the SDK's new surface out: "image input; `chatcompat` and `Discover`; keeping learned reasoning
refusals across runs; `Catalog.Search`" (its D18, "each for its own record when wanted"). The owner chose, on
2026-10-29, that this record takes exactly that remainder ("0036, its D18 scope"). Two things the assessment found,
which 0035 does not cover, are added: gobble never reads the SDK's warnings, and it builds the SDK's registry again on
every lookup.

This record decides how gobble takes that surface, keeping the boundary that only `llm/provider` imports the SDK
([0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md) rule 4,
`internal/archtest/check.go:265`). It does not implement anything.

### How this was measured, 2026-10-29

- **gobble:** `main` at `ccd8aa8`, Go 1.27.3, `go.mod:16` requiring go-llmprovider-sdk `v1.22.2`. The tree's only
  changes are the other session's 0035 MADR and its `docs/README.md` row, neither of which is code.
- **The SDK:** the local clone at tag `v1.24.1` (`04dc55a`, 2026-10-27; `v1.23.0` is `5555147`, 2026-10-26). Every
  name below was read with `go doc` or `git grep` on the tag, and each location was printed by a script, not copied
  by hand. The SDK's `main` has changed no Go file since the tag.
- **The probes**, all in scratch directories outside gobble's tree:
  - `scripts/check_libs.py --lib go-llmprovider-sdk --tag v1.24.1`. It passed 11 of its 12 checks: get and tidy,
    version, modules, build, vet for linux, darwin and windows, api, govulncheck and advisories. It failed one: test,
    with `llm/provider` `TestWireOffline` (`err llm: unavailable: llmprovider: offline: Post
    "http://127.0.0.1:…/v1/responses": dial tcp …: connect: connection refused, want llm.ErrOffline`). `apidiff`
    from `v1.22.2` lists 25 compatible changes and no incompatible one.
  - A scratch clone of gobble moved to `v1.24.1` (`go get`, `go mod tidy`: `go.mod` and `go.sum` alone change):
    `go build ./...`, `go vet ./...` and `go vet -tags live_gateways ./...` pass. `go test ./...` passes in 35
    packages and fails `TestWireOffline` alone. With 0035's D3 row added, the tests of `llm/...`, `agent` and
    `acpserver` pass.
  - A benchmark of the registry lookup gobble does (`providers.Default().Descriptors()`, then a scan for the id),
    against one registry built once, on macOS, three runs each. Each lookup builds the 10 built-ins anew: 8.5–8.7 µs,
    14,272 B and 85 allocations. On a registry built once the same lookup takes 88–97 ns, 96 B and 1 allocation.
  - A request with a tool's output followed by an image (`FunctionCallOutputItem`, then `ImageItem`), sent to a local
    test server through four providers, each request body recorded:

    | Wire (provider) | Where the image went |
    | :--- | :--- |
    | Messages (`claude`) | an `image` block in the same user message, after the `tool_result` block |
    | Responses (`openai`) | a user message holding an `input_image`, after the `function_call_output` item |
    | Interactions (`gemini`) | a `user_input` holding an `image`, after the `function_result` item |
    | Chat Completions (`chatcompat`) | a user message holding an `image_url` part, after the `tool` message |

    How each service treats an image placed there was not measured. No request left the machine.

### What the SDK added that gobble does not use

The search found no use, in gobble, of any name below (`git grep` over `*.go` for `Warnings`, `ReasoningRefusal`,
`ImageItem`, `chatcompat`, `Catalog.Search` and the others: two hits, both in `internal/logging` tests, neither the
SDK's).

| The SDK's (`v1.24.1`) | Where | What it gives |
| :--- | :--- | :--- |
| `Response.Warnings`, with `WarningEffortDegraded`, `WarningReasoningLearned`, `WarningImageDropped` | `llmprovider/warning.go:28`, `:34`, `:38` | a request answered, but not as asked |
| `ReasoningRefusal`, `WithReasoningRefusals` | `llmprovider/refusal.go:47` | refusals a caller keeps and passes back, so that "a later process never sends it" |
| `chatcompat.New`, `WithID`, `Descriptor` | `providers/chatcompat/chatcompat.go:170`, `:136`; `descriptor.go:9` | any Chat Completions server under an id the caller picks, registered in the caller's `Registry` |
| `chatcompat.Discover` | `providers/chatcompat/discover.go:63` | the local servers running now |
| `Catalog.Search`, `Match.Why` | `catalog/search.go:205`; `catalog/model_matcher.go:44` | a listing's search by facts: `vision`, `tools`, `128k`, a family, a year |
| `ImageItem`, `Capabilities.Images` | `llmprovider/item.go:79`; `capabilities.go:48` | an image in a user turn, by bytes or https URL |

### What gobble does instead today

- **Warnings are dropped.** The adapter's `EventDone` case (`llm/provider/provider.go:337`) reads `Usage` and
  `FinishReason` from the response and nothing else. A lowered effort, a learned refusal or a dropped image is never
  shown or logged.
- **The registry is rebuilt per call.** `llm/provider/ambient.go` calls `providers.Default().Descriptors()` at
  `:49`, `:61`, `:81`, `:93`, `:103`, `:152` and `:163`, and `signin.go:174` passes `providers.Default()`. The SDK's
  `DefaultWith` builds a new `Registry` each time (`providers/providers.go:65`). `Label`, `CredentialVar` and the
  others run once per provider in loops, such as the picker's choices (`internal/cli/choice.go:36`).
- **models.json's provider table is closed.** `llm/catalog/modelsjson.go:85` lists the SDK's ten ids by hand, with
  Pi's wire names. Any other key is skipped as "not a provider go-llmprovider-sdk %s builds" (`:196`).
  `llm/catalog` may not import the SDK, so a new SDK built-in needs a gobble edit before models.json can name it. A
  provider's `name` is reported as unused (`:225`).
- **The search is by id and label only.** `provider.Search` calls `catalog.Search` over a list of ids
  (`llm/provider/list.go:54`). The ACP picker (`internal/cli/agent.go:86`) and `gobble models --search`
  (`internal/cli/modelscmd.go:209`) both use it. `List` keeps `Recommended`, `Usable` and the source, and drops the
  `Catalog` with its facts (`list.go:100`).
- **Images are not carried:**
  - `toRequest` has no case for `llm.ContentImage` (`provider.go:415-463`; the last case is `ContentToolResult` at
    `:443`).
  - `Capabilities` reports only native streaming (`:261`), and `llm.Capabilities.ImageInput` "stays false until the
    SDK accepts image input" (`llm/llm.go:45`).
  - The ACP agent advertises no image (`acpserver/agent.go:370`). Its `userMessage` keeps text blocks only (`:931`),
    and `promptFiles` leaves out image and audio blocks (`acpserver/mentions.go:127`).
  - `read` answers an image with `[The image was not sent: gobble cannot send images to models yet.]`
    (`tool/builtin/read.go:31`). It sniffs `.bmp` too (`:103`), which the SDK does not take.
  - MCP image content becomes `[image … omitted]` (`mcpclient/tool.go:159`).
  - A tool result is text: the agent builds `llm.Content{Type: ContentToolResult, … Text: res.Text()}`
    (`agent/agent.go:754`), and `tool.Result` has no image field (`tool/tool.go:173`).
  - The session format already has image blocks (`session/session.go:198`), and compaction already estimates them
    (`compaction/estimate.go:67`). The history conversion carries text alone (`acpserver/history.go:87`, `:275`).
  - The CLI client refuses a prompt with images when the agent does not advertise them (`acpclient/conn.go:25`), as
    [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D19 item 1 says.

### What the records already decided

- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md)'s `read` row (`:118`) decides image content for 1.0:
  "auto-resized to 2000×2000 and 4.5 MB of base64, keeping the smaller of PNG and JPEG. EXIF orientation is applied.
  A text note replaces the image for text-only models. Pure Go (`image/*`, `golang.org/x/image`)." So
  `golang.org/x/image` is already named by a record, as `AGENTS.md` ("Dependencies") requires.
- 0008-MADR D19 (`:690`): "`promptCapabilities.image` follows 0005-MADR: false until the SDK's image input lands, then
  true when the configured default model accepts images. `initialize` precedes any session, so this cannot be per
  model. An image sent to a model that cannot take one ends the turn with a readable error and is never dropped
  silently." D4 (`:404`): "An image `@path` becomes an ACP image content block."
- [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md) parks `images.autoResize` (`:162`): "no
  consumer; waits on `llm` carrying images".

### gobble-cli#4, reconciled

[gobble-cli#4](https://github.com/maccavelli/gobble-cli/issues/4) listed nine items before `v1.23.0` shipped. What
the SDK's
`0061-MADR-consumer-gaps-from-gobble-issue-9.md` (in that repository) then decided, and who carries each item:

| #4 item | The SDK, at `v1.24.1` | Carried by |
| :--- | :--- | :--- |
| 1. empty answer | `ErrEmptyAnswer` | 0035's D4 |
| 2. offline | `ErrOffline`, gobble's rule moved into the SDK (`internal/transport/offline.go:18`) | 0035's D3 and D4 |
| 3. `max` | `EffortMax` | 0035's D4 |
| 4. headers | declined: "No new API" (its D6, `:337`); a caller's `RoundTripper` through `WithHTTPClient` | nobody: `headerTransport` stays |
| 5. one-hour cache-write price | deferred until models.dev publishes one (its D7, `:345`) | nobody yet |
| 6. unnamed OpenAI-compatible endpoints | `chatcompat` | this record, D5 |
| 7. image input | `ImageItem` | this record, D7 and D8 |
| 8. native tool deferral | shipped earlier (its 0040-MADR) | 0012-MADR and 0021-MADR, not this record |
| 9. argument deltas | declined: "a tool call arrives whole" (its D8, `:350`) | nobody |

The workspace (0035's D5), the warnings (D3), the learned refusals (D4), the facts search (D6) and the registry (D2)
are not in #4.

## Decision Drivers

* **0004-MADR:** wait for the library, then drop the copy. Rule 4: only `llm/provider` imports the SDK, so every SDK
  type stays behind gobble's own types.
* **0005-MADR and 0008-MADR D19:** image input as those records already specify it, never dropped silently.
* **Extensible:** a provider the SDK adds, or a server the user names, should need no gobble edit.
* **Efficient:** no work repeated per call that one build serves, and no request a kept answer saves.
* **Compatible:** gobble's `llm` and `tool` packages are `Stability: stable`. Changes to them are additions, as
  `apidiff` in `make preflight` requires.
* **No regression:** `acpserver/testdata/session.golden` byte-identical unless a phase names the change; every gate on
  macOS, the Linux machine and the Windows laptop.
* **Each phase lands, verifies and reverts on its own.**

## Considered Options

* A. Leave it to later records, one per feature, as 0035's D18 says.
* B. One record for the remainder, phase by phase, on one shared registry, after 0035's go-llmprovider-sdk phases.
* C. The same, with images split out to a record of their own.

## Decision Outcome

Chosen option: **"B. One record for the remainder, phase by phase, on one shared registry"**, because the registry
is the seam both Chat Completions servers and the search need. The warnings carry what images and efforts report. And
the image path is already specified by 0005-MADR and 0008-MADR, so a separate record would only repeat them. Q3 lets
the owner split images out anyway.

### The order

- **D1. After 0035's go-llmprovider-sdk phases.**
  - This record starts once gobble requires go-llmprovider-sdk `v1.24.1` with 0035's D3 row and D4 changes in.
  - If 0035 is rejected, or changes those decisions, this record is amended before its first phase, to carry the pin
    and the D3 row itself. The `TestWireOffline` failure above shows the pin cannot land without that row.
  - The workspace (0035's D5) is not a precondition: no phase here reads it.
  - *2026-11-06, read again at the 0036-PLAN's P0 of that date: the precondition is met. 0035-PLAN's P4 (`bd79d2a`, the
    pin and D3's offline row) and P5 (`43649b8`, D4's copies gone) landed on 2026-10-30. The pin has moved on since,
    by 0035's later phases, to `v1.26.1`, and to `v1.27.0` by its P32 if approved. So "`v1.24.1`" above is the floor,
    not an exact tag: this record needs `v1.24.1` or later with D3's row and D4 in. Every SDK name this record uses is
    still exported, unchanged, at `v1.26.1` and at `v1.27.0`, and gobble still uses none of them. The lines this record
    cites in gobble were measured at `ccd8aa8`; several have moved, and each phase reads its own again.*
  - [0031-PLAN-sdk-current-releases-and-adoption.md](0031-PLAN-sdk-current-releases-and-adoption.md) P4b, the ChatGPT
    plan sign-in, needs none of this record. Its `SignIn` change and D2's change of `SignIn`'s registry touch the
    same function, so whichever lands second rebases on the first.

### One registry, and an open provider table

- **D2. `llm/provider` builds the SDK's registry once, and models.json's table follows it.**
  - The built-ins' registry is built once per process (`sync.OnceValue(providers.Default)`). Every package function
    that calls `providers.Default()` today reads it. This is the benchmark's 8.6 µs and 85 allocations per lookup
    against 90 ns and 1.
  - A run's providers are a value of gobble's own: `provider.Set`, a name for the PLAN to confirm. It holds the
    built-ins and the Chat Completions servers models.json names (D5), and wraps an SDK `Registry` that no other
    package sees. `internal/cli` builds it once per run from models.json, and `New`, `List`, `ModelInfo`, `Methods`
    and `SignIn` take it through their options. A nil set is the built-ins alone, so a caller that passes none keeps
    today's behaviour.
  - `llm/catalog`'s parser takes the set of SDK ids from its caller, rather than from `providerAPIs`' keys. The table
    keeps only what the SDK does not say: Pi's wire names for each built-in. A built-in the table does not name takes
    models.json with no `api`.

### What a response says

- **D3. The SDK's warnings reach gobble.**
  - `llm` gains a sealed event, `llm.Warning{Kind, Model, Detail}`. The adapter yields one for each of the done
    event's `Response.Warnings`, before `Usage` and `Done`. `Kind` is the SDK's string. `llm` does not import the
    SDK, so it names the kinds gobble acts on as its own constants, and passes any other through unchanged.
  - The agent logs each warning at Warn, once per kind and model in a session, so a degraded effort on every turn is
    said once. The terminal already shows Warn records. Sending them to an ACP client is
    [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md)
    D8's, not this record's.
  - Adding an event type is an addition to `llm`'s sealed set. gobble's own switches gain a case or ignore it.
- **D4. Learned refusals are kept across runs** (Q1).
  - A `WarningReasoningLearned`'s `Refused` is saved in the state directory, beside `recent-models.json`, one list
    per provider, bounded and deduplicated.
  - `New` passes the provider's saved list back with `WithReasoningRefusals`. A later process then never sends that
    model the refused effort or shape, which saves the refused request and its resend on that process's first turn.
  - A file gobble cannot read is ignored with a warning: the SDK relearns, as it does today.
  - `ProbeReasoning` is not called. Each probe is a billed generation.

### Chat Completions servers

- **D5. A models.json provider gobble does not build is a Chat Completions server, when it says so.**
  - A key that is neither a built-in nor a Pi alias is kept when it has `baseUrl`, and an `api` that is
    `openai-completions` or absent. Any other key is still skipped, with a reason that says gobble has no generic wire
    for its `api`.
  - It is built with `chatcompat.New`, `WithID` (its key) and `WithBaseURL`, and registered in the run's set (D2).
    Its label is models.json's `name`, now used, or its key.
  - **Its key** is models.json's `apiKey` in Pi's grammar, or `auth.json`'s entry under its id, or none. The SDK sends
    no `Authorization` header without a key.
  - **Its models** are its listing, through `catalog.List` with `WithID`, and the models models.json defines. Their
    facts come from models.json's layer, then the server's.
  - **It is chosen only when named:** by `--model id/model`, by `defaultProvider`, or in the picker. It is never
    ambient (Q2). A local server needs no key, so "the first provider with a credential" would always pick one that
    happens to be configured.
  - `Discover` is not used here. Offering the servers running now in `gobble setup` is a product change for its own
    record.
  - *(Amended 2026-11-20, when P4 landed.)* A kept server is `ChatServer` (`llm/catalog/modelsjson.go:27`).
    `chatServer` (`:278`) is true when the key is not a caller id and not a wire-table built-in, `baseUrl` is set,
    and `api` is empty or `openai-completions`. `unknownProvider` (`:265`) skips any other unknown api with
    `gobble has no generic wire for api %q`, and skips a key with no base URL with the same not-a-provider reason
    as before. `ChatServers` (`:200`) returns them sorted by id. The label is `name` when it is set, else the key.
    `AddChatServers` (`llm/provider/set.go:99`) registers each on the run's set with `chatcompat.Descriptor` and a
    factory of `chatcompat.New`, `WithID` and `WithBaseURL`. `listChat` (`llm/provider/list.go:120`) lists with
    `catalog.List`, `chatOptions` (`set.go:138`: `WithID`, and `WithBaseURL` when the caller did not set one) and
    does not send `WithProfile`, `WithLimit` or `WithLastGood`. `ModelInfoWith` (`llm/provider/model.go:78`) adds
    the same options (`:81`). `ChooseWith` (`llm/provider/ambient.go:39`) returns a server on the set when it is the
    preferred id, with or without a credential (`:48`); an empty preference still scans only key-taking built-ins
    (`:53`). `KeyProviders` and `Providers` stay the built-ins. The CLI registers the servers once per run
    (`bindChat`, `internal/cli/modelsjson.go:117`, `chatOnce` at `internal/cli/main.go:249`) after `chooseWith`
    reads models.json (`:49`). The picker adds one choice per server (`internal/cli/choice.go:77`). `knownProvider`
    (`internal/cli/modelscmd.go:32`) accepts one, and `printModels` lists it when `--provider` names it (`:81`).
    `offlineModels` includes each server's models.json models (`internal/cli/modelflags.go:49`). A credential is
    the same path as any id: `auth.json`, then models.json's `apiKey`, then the variable, which is empty here.
    `New` adds `WithAPIKey` only when a key is set (`llm/provider/provider.go:219`). `Discover` is not called.
    `providerOptions`, cited at `internal/cli/modelsjson.go:158` when P3 landed, is now `:172`.

### The search

- **D6. Where gobble holds a live listing, it searches the listing's facts.**
  - `Listing` keeps its `Catalog`, unexported. `llm/provider` adds a search over a `Listing` that returns ids, best
    first, by `Catalog.Search`.
  - The picker and `gobble models --search` take the listing's matches first, then `catalog.Search`'s over the ids
    the listing does not hold: the static, kept and models.json models. `provider.Search` stays for a plain list.
  - `Match.Why` is not shown. Showing it changes the picker's and the table's layout, for a later record.
  - Search results change: `claude vision 200k` finds the Claude models with image input and a window of at least
    200k, where today it finds none.
  - *(Amended 2026-11-20, when P5 landed.)* The search is `SearchListing` (`llm/provider/list.go:71`),
    the name the plan wrote. `Listing` keeps the SDK catalog unexported (`catalog`, `:42`); `held`
    (`:43`) says it was stored. `List` stores both on a live built-in (`:139`) and `listChat` on a
    live Chat Completions server (`:163`). A listing that failed stores neither. `HasCatalog`
    (`:85`) tells a catalog-bearing listing from one a test marks live with no catalog, which stays
    the id search. `provider.Search` is unchanged. `Match.Why` is not shown. The two callers are
    the picker's `SearchModels` (`internal/cli/agent.go:99`), which reads the run's cached listing
    (`cachedListing`, `internal/cli/models.go:283`), and `searchModels` (`internal/cli/modelscmd.go:214`).
    Both use `listedSearch` (`:241`). A non-empty `gobble models list` is not sorted again
    (`printModels`, `:137`): the listing's matches come first, then the id matches among the models
    it does not hold. A list with no search words stays sorted by provider, then id. The numbered
    picker keeps its order; the callback's returned ids are the search's order.

### Images

- **D7. The adapter carries images.**
  - `llm.ContentImage` in a user message becomes an `ImageItem` (`RoleUser`, `MediaType`, `Data`) in place.
  - In a tool message it becomes an `ImageItem` right after its call's `FunctionCallOutputItem`. That is the shape the
    probe saw land beside the tool's output on all four wires.
  - `llm.Capabilities.ImageInput` is true when the provider's `Capabilities.Images` is not `Unsupported`.
  - gobble sends bytes, not URLs, except for an ACP image that carries only a URI. Where a route cannot fetch an image
    by URL (Ollama, OpenCode's google route), the SDK drops it with a `WarningImageDropped` (D3). A form a service
    refuses, such as a GIF or WebP on Grok, is refused by the SDK; D8's preparation sends PNG or JPEG.
  - A model whose facts state its input without `image` is refused by the SDK before the network, with
    `ErrUnsupported`. That ends the turn with a readable error (0008-MADR D19).
- **D8. Images come in where the records say** (Q3):
  - **The ACP prompt.** An image block becomes `ContentImage`. `promptCapabilities.image` is true when the configured
    default model accepts images (its facts' input holds `image`, or are unknown on a provider that takes images), as
    0008-MADR D19 says. The history conversion carries `session.BlockImage` both ways.
  - **`read`.** When the session's model takes images, `read` returns image content: `tool.Result` gains its images
    and `tool.Env` says whether the model takes them, both additions. Otherwise it returns today's note, reworded to
    name the model. The image is prepared as 0005-MADR's row says: EXIF orientation applied, at most 2000×2000 and
    4.5 MB of base64, the smaller of PNG and JPEG. It always ends as PNG or JPEG, so a BMP, which the SDK refuses, and
    a GIF or WebP, which Grok refuses, are sent too. An animated GIF sends its first frame. This adds
    `golang.org/x/image`, in the commit of its first import.
  - **MCP.** An image result becomes the same image content, prepared the same way, or the same note.
  - **`@path`.** The CLI client sends an image `@path` as an ACP image block (0008-MADR D4). The agent's own `@path`
    expansion does the same for a model that takes images.
  - **`images.autoResize`** is unparked as a key, default true. Off sends an image unchanged within 4.5 MB, and a
    larger one as the note.

### Not this record

- **D9.**
  - 0035's decisions: the pins, the offline row, the copies, the workspace, and every library but this one.
  - Native tool deferral: 0012-MADR and 0021-MADR.
  - The one-hour cache-write price: the SDK deferred it.
  - Argument deltas: the SDK declined them.
  - Headers: the SDK declined a hook, so `headerTransport` stays as the SDK recommends.
  - `Discover` in `gobble setup`; `ProbeReasoning`; `Match.Why`; image output; 0031-PLAN P4b.
    *(2026-11-20: `Discover` here is `chatcompat.Discover`. `wizard.Options.Discover`, which gobble already sets, is a
    live listing check and is not that function. Keeping a `forced_tool_choice_learned` warning across runs with
    `WithModelForcedChoice` is the same shape as D4 and is not this record. The one-hour cache-write price is still
    absent from the SDK through `v1.31.0`; showing cache figures is
    [0057-PLAN-cache-visibility-and-one-hour-pricing.md](0057-PLAN-cache-visibility-and-one-hour-pricing.md), which is
    proposed and decides nothing here. 0031-PLAN P4b is now
    [0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md).)*
  - Two things gobble still hard-codes that the SDK's descriptors do not say: which listings need no key
    (`llm/provider/list.go:42`), and Pi's wire names (D2). Each could become a request to the SDK. None is filed by
    this record. *(2026-11-20: the key-less list is `publicListings` at `list.go:41`.)*

### Questions for the owner

- **Q1. Learned refusals (D4).** Keep them across runs in the state directory (recommended)? Or rely on the SDK's
  learning within a process, as today?
- **Q2. Chat Completions servers in the ambient choice (D5).** Only when named (recommended)? Or also as ambient
  candidates after the built-ins, so a configured local server is picked when no built-in has a credential?
- **Q3. Images (D7, D8).** In this record as its last phases (recommended)? Or the adapter and the ACP prompt here,
  with `read`, MCP and `@path` in a later record? Or images in a record of their own?
- **Q4. Order.** After 0035's go-llmprovider-sdk phases, as D1 says (recommended)? Or the registry (D2) first, since
  it needs no new tag, ahead of 0035?

**The owner's answers, 2026-10-29.** Each question was put with three options, the recommended one first, and the
owner chose the recommended one each time:

- **Q1:** "Keep across runs". The others were "In-process only", and "Keep, and probe first", which calls
  `ProbeReasoning`, a billed generation, on a model's first use. D4 stands.
- **Q2:** "Only when named". The others were "Ambient after built-ins", and "Ambient only with a key". D5 stands.
- **Q3:** "All of it, last phases". The others were "Adapter and ACP here", which moves `read`, MCP and `@path` to a
  later record, and "A record of their own". D7 and D8 stand, as the PLAN's P6 and P7.
- **Q4:** "After 0035's llm phases". The others were "Registry (P1) first", and "After all of 0035", which also
  waits for its other libraries. D1 stands.

So this record is accepted. Its PLAN waits for the owner's approval of execution, and for 0035's go-llmprovider-sdk
phases (D1).

### Re-grounded 2026-11-20

Read again at gobble `main` `32bc0e7` and go-llmprovider-sdk tag `v1.31.0` (commit `0fbb761`, 2026-11-14). `go list -m`
prints `v1.31.0`. That tag is the latest published one. The SDK's `main` is ahead of it; a diff of `warning.go`,
`refusal.go`, `item.go`, `capabilities.go`, `providers/chatcompat`, `catalog/search.go`, `catalog/model_matcher.go`,
`providers/providers.go`, `catalog/lister.go` and `forcing.go` from the tag to that `main` is empty, so those commits
do not change this record's surface. D1's floor is unchanged: `v1.24.1` or later, with 0035's D3 row and D4 in. The
pin today is `v1.31.0`. Q1 to Q4 are not reopened.

The precondition was run again, as the PLAN's P0. `go test -count=1 -run TestWireOffline ./llm/provider/` passed
(`ok`, 0.668s; the test is `llm/provider/offline_test.go:25`). `git grep` finds neither `sdkEmptyAnswer` nor
`func offline` under `llm/provider`. Gobble's `*.go` uses none of `ReasoningRefusal`, `ImageItem`, `chatcompat`,
`WithReasoningRefusals` or `WarningImageDropped`.
*(Amended 2026-11-20, when P3 landed.)* That sentence is the measurement before the phase. After it,
`llm.ReasoningRefusal` is gobble's own type (`llm/event.go:79`) and `New` passes `Options.Refusals` to
`WithReasoningRefusals` (`llm/provider/provider.go:234`). `WarningImageDropped` has been a constant since P2.
`ImageItem` and `chatcompat` are still unused.
*(Amended 2026-11-20, when P4 landed.)* `chatcompat` is used, by `AddChatServers`. `ImageItem` is still unused.

What is still true, at the lines read this date:

- The registry is still built per call. `ambient.go` still calls `providers.Default().Descriptors()` at `:49`, `:61`,
  `:81`, `:93`, `:103`, `:152` and `:163`. `New` calls `providers.New` (`llm/provider/provider.go:238`), and the SDK's
  `providers.New` calls `Default()` (`providers/providers.go:54` at the tag). The wizard is given
  `Registry: providers.Default()` at `signin.go:346`.
- models.json's table is still the ten ids in `providerAPIs` (`llm/catalog/modelsjson.go:85`). An unknown key is still
  skipped at `unusable` (`:200`). `name` is still reported unused.
  *(Amended 2026-11-20, when P1 landed.)* The two bullets above are that measurement, at `32bc0e7`, before the phase.
  After it, the built-ins' registry is built once (`sync.OnceValue(providers.Default)` in `llm/provider/set.go`),
  and the functions in `ambient.go` read the descriptors copied once from it. `New` uses that registry's `New`,
  or the run's `Set` when `Options` carries one (`provider.go:240`). `SignIn` gives the wizard that registry
  (`signin.go:346`). `ParseModelsJSON` takes the caller's ids (`llm/catalog/modelsjson.go:149`). `providerAPIs`
  keeps Pi's wire names. An id the caller passes that the table does not name is kept only with no `api`.
  `name` is still reported unused.
  *(Amended 2026-11-20, when P4 landed.)* That sentence is the measurement before this phase. `name` is now the
  label of a Chat Completions server (`ChatServers`, `llm/catalog/modelsjson.go:207`). On a built-in it is not
  reported.
- `Listing` still drops the catalog (`llm/provider/list.go:111`). `Search` is still the id and label matcher (`:54`).
  The ACP picker passes that function (`internal/cli/agent.go:97`). `gobble models --search` uses it
  (`internal/cli/modelscmd.go:209`).
- `Capabilities` still sets only native streaming (`provider.go:332`). `EventDone` (`:426`) still yields usage and the
  finish reason, and does not read `Response.Warnings`.
  *(Amended 2026-11-20, when P2 landed.)* That sentence is the measurement before the phase. After it, `EventDone`
  (`llm/provider/provider.go:428`) yields one `llm.Warning` for each of the response's warnings, then usage and the
  finish reason. `llm` names `WarningEffortDegraded`, `WarningReasoningLearned`, `WarningImageDropped` and
  `WarningDeferralEmulated`. The agent logs every other kind at Warn, once per kind and model in the session, and
  logs `deferral_emulated` at Debug. Each prompt of a session shares one `WarningLog`. A forced-choice warning is
  logged at Warn and is not saved.
  *(Amended 2026-11-20, when P3 landed.)* A `WarningReasoningLearned`'s `Refused` is now saved.
  `yieldWarnings` (`llm/provider/provider.go:511`) copies it onto `llm.Warning` (`llm/event.go:92`). The agent's
  `note` (`agent/warning.go:40`) saves one on every sighting, through `Config.SaveRefusal` (`agent/agent.go:82`),
  which `acpserver` binds to the session's provider (`acpserver/agent.go:334`, set at `:794`). The CLI writes
  `<state>/reasoning-refusals.json` beside `recent-models.json` (`internal/cli/refusal.go:17`), one list per
  provider, ten kept, the same bound `recent-models.json` keeps, deduplicated, owner-only. `providerOptions`
  passes that provider's list (`internal/cli/modelsjson.go:158`). A file that cannot be read is ignored with one
  warning, and the provider is still built. `ProbeReasoning` is not called. A forced-choice warning is still
  logged at Warn and is not saved. The registry `New` call the P1 amendment cites at `:240` is now `:247`, and P2's
  `EventDone` citation `:428` is now `:435`.
- `toRequest` still has no `ContentImage` case. Its tool-result case is `ContentToolResult` at `:587`, and since
  0012-MADR it appends `ToolLoadItem`s after that output. An image in a tool message is an `ImageItem` immediately
  after the `FunctionCallOutputItem`, and those loads stay after the image. The probe of 2026-10-29 had no loads.
- `promptCapabilities.image` is still unset (`acpserver/agent.go:412`). `acpserver/agent_test.go:164` still asserts it
  false. `promptFiles` still leaves image blocks out (`acpserver/mentions.go:128`). History still carries `BlockText`
  only (`acpserver/history.go:273`, `:329`).
- `read`'s note is still `tool/builtin/read.go:48`. An MCP image is still `[image %s omitted]`
  (`mcpclient/tool.go:166`). `session.BlockImage` (`session/session.go:204`) and the compaction estimate
  (`compaction/estimate.go:67`) are unchanged. `golang.org/x/image` is not required.
- Rule 4 still keeps the SDK's root module behind `llm/provider`. The check is `internal/archtest/check.go:318` (the
  citation `:265` has moved). `authcontract` is the exception 0051-MADR added; this record does not widen it.
- `headerTransport` stays. When the caller passes no client it already wraps `llmprovider.NewHTTPClient()`
  (`provider.go:185`), which is the wrapping `v1.25.0` documents. D9 does not move it.
- Grok still refuses a GIF or a WebP before the network (`providers/grok/grok.go:266` at the tag). D7's sentence holds.

What `v1.25.0` through `v1.31.0` added that this record uses or leaves out:

- `warning.go` is unchanged since the lines P0 recorded on 2026-11-06. It has six kinds, not the three the table above
  names: `forced_tool_choice_degraded` (`:12`), `forced_tool_choice_learned` (`:17`), `deferral_emulated` (`:23`),
  `effort_degraded` (`:28`), `reasoning_learned` (`:34`), `image_dropped` (`:38`). D3 already passes any other kind
  through and logs it. The two forced-choice kinds are logged that way. Persisting `forced_tool_choice_learned` is
  D9. 0012-PLAN's native deferral has landed, so `deferral_emulated` is logged at Debug, once per session and model,
  as that plan's note of 2026-11-04 said once its P4 had landed.
- `catalog.Lister` (`catalog/lister.go:14`) and `chatcompat`'s `ListCatalog` (`chatcompat.go:455`) are `catalog.List`
  for a provider already built. They refuse `catalog.WithProfile` and `catalog.WithLimit`. D5's listing stays
  `catalog.List` with `chatcompat.WithID` and a base URL. A Chat Completions id is not sent those two options.
  `ListCatalog` is the same listing when the provider is already built.
- `chatcompat.New` is `:174`, `WithID` `:140`, `Discover` `:65`, `Descriptor` `descriptor.go:9`, `Catalog.Search`
  `catalog/search.go:205`, `Match.Why` `catalog/model_matcher.go:44`, `ImageItem` `item.go:79`,
  `Capabilities.Images` `capabilities.go:48`. Those are the lines at `v1.31.0`, the same ones P0 recorded at
  `v1.26.1` on 2026-11-06.
- Lines that moved in gobble since that P0 record, with the fact unchanged: `EventDone` `:426` (was `:348`); the
  wizard registry `:346` (was `:177`); `ContentToolResult` `:587` (was `:486`); the MCP image note `:166` (was
  `:153`).

### Consequences

* Good, because a provider the SDK adds, and any Chat Completions server the user names, need no gobble edit to the
  provider table.
* Good, because a registry lookup costs about a hundredth of what it costs today. A refused effort is learned once
  per state directory, not once per process.
* Good, because a lowered effort or a dropped image is said, not silent.
* Good, because image input lands as 0005-MADR and 0008-MADR specify it, closing #4's items 6 and 7.
* Good, because the search finds a model by what it can do, not only by its name.
* Bad, because images ride in the history: each later turn sends them again, within the 4.5 MB cap, until compaction
  drops the turn. That is Pi's behaviour, and the token estimate already counts them.
* Bad, because the session files grow by the images' base64.
* Bad, because how each service treats an image placed after a tool's output is unmeasured until the live check.
* Neutral, because `llm` and `tool` gain types and fields, additions `apidiff` accepts.
* Neutral, because models.json's `name` changes from "not used by gobble" to used.

### Confirmation

[0036-PLAN-adopt-go-llmprovider-sdk-new-capabilities.md](0036-PLAN-adopt-go-llmprovider-sdk-new-capabilities.md):

- **The registry:** a test fails when a package function builds a registry per call.
- **The table:** models.json accepts an id the registry adds.
- **The warnings and refusals:** a warning from a fake provider reaches the log once, and a learned refusal is sent
  back on the next `New`.
- **The servers:** a models.json server lists and answers through `chatcompat` against a local test server.
- **The search:** a facts query finds what `Catalog.Search` finds.
- **The images:** the four wires' requests hold the image where the probe saw it, and a text-only model ends the turn
  with an error.
- **Every check** is seen failing first on a planted break in a scratch copy.
- **Gates:** `make preflight` and the race detector on the three hosts.
- **Live, by the owner, billed and optional:** one image turn on Claude and one on an OpenAI vision model, and one
  Chat Completions server listing.

## Pros and Cons of the Options

### A. A record per feature, later

* Good, because each feature waits for the owner's wish.
* Bad, because the registry, which three of them need, would be decided three times or not at all.
* Bad, because #4 stays open with items the SDK shipped on 2026-10-26.

### B. One record, phase by phase

* Good, because the seams (registry, warnings) are decided once, before the features that use them.
* Good, because each phase reverts on its own.
* Bad, because it is many phases, and images alone touch `llm`, `tool`, the agent, ACP, MCP and the CLI.

### C. The same, images apart

* Good, because the smaller record lands sooner.
* Bad, because 0005-MADR and 0008-MADR already decided images. A record of their own would repeat those decisions,
  and would need the warnings and capability seams this record builds.

## More Information

- [gobble-cli#4](https://github.com/maccavelli/gobble-cli/issues/4), and go-llmprovider-sdk issue 9, its source.
- ~~The other session's `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` (proposed, uncommitted on
  2026-10-29): D1, D3, D4, D5, D18. A link replaces this text once it is committed.~~ *(2026-11-06: committed and
  accepted.)* [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md): D1, D3, D4, D5, D18; and its PLAN,
  [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md),
  P4 and P5.
- [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md): D9 and its note of
  2026-10-29.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md), [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md)
  D4 and D19, [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md) F11.
- go-llmprovider-sdk's records, in that repository: `0061-MADR-consumer-gaps-from-gobble-issue-9.md` (D4 to D8, the
  search amendment), `0062-MADR-learn-thinking-parameters-from-refusals.md` (D6), and its `CHANGELOG.md` sections
  `v1.23.0` and `v1.24.0`.
