---
status: accepted
date: 2026-10-13
decision-makers: repository owner
consulted: none
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Ask for prompt caching on agent turns, with one provider and session id per ACP session

## Context and Problem Statement

gobble-cli issue 2 (2026-10-13) asks gobble to adopt go-llmprovider-sdk `v1.12.0`'s request-side prompt caching: "Request.Cache on agent turns, one session id per ACP session, cache-write usage". The SDK's decision is its `0039-MADR-request-side-prompt-caching.md`, which names gobble-cli as the consumer that "asks for 'cache control: Anthropic `cache_control`'" (its §4) and leaves the adoption to this repository (its 0039-PLAN, "Not done, and why").

The issue's evidence is from `origin/main` `108d9f0` with the SDK at `v1.2.1`. Every claim below is re-measured at `main` `007b63f`, which pins the SDK at `v1.12.0` (0020-PLAN P1, `1625013`). Two of the issue's eight points no longer hold there (F1). The owner also asked for an assessment of how the caching layer should be built, "eg: ring buffer, ram based cache, internal session cache", against the SDKs' APIs and gobble's use case; F9 to F12 and D6 answer it.

A prompt cache is the service's, not gobble's: the service keeps the processed prefix of a prompt and charges less to read it back. What gobble controls is whether it asks, which conversation a request belongs to, that the prefix it sends stays byte-identical from one model call to the next, and what it reports.

### What was measured, not assumed

- **The pins and the local clones** (2026-10-13): `go.mod:15-17` pins acp-go-sdk2 `v0.17.1`, go-llmprovider-sdk `v1.12.0` and go-selfupdate-lib `v1.15.2`, each the latest tag. The local go-llmprovider-sdk clone is one commit past `v1.12.0` (`3e81e3d`, which changes only `docs/`), so its code is `v1.12.0`'s.
- **The SDK's contract** (`llmprovider/contract.go`):
  - `CacheRetention` (`:95-117`): `""` asks for nothing, "the request is sent as it would be without the field"; `CacheShort` is "five minutes on Anthropic"; `CacheLong` "one hour on Anthropic. Where the service offers one lifetime, it is CacheShort's". Any other value is refused before the network (`:275-277`).
  - `Request.Cache` (`:210-212`). `Usage` (`:173-189`): `CachedTokens` and `CacheWriteTokens` are parts of `InputTokens` that do not overlap; `CacheWrite1hTokens` is the part of the writes with a one-hour lifetime, "Only Anthropic's Messages reports it".
- **What each provider sends** (SDK code): Claude and OpenCode's messages route call `messages.AddCache` (`providers/claude/claude.go:307`, `providers/opencode/opencode.go:531`), which adds a top-level `cache_control` and one breakpoint on the system prompt, or on the last tool when there is none, both with the same TTL (`internal/wire/messages/cache.go:20-42`). OpenAI, xAI, Together and OpenCode's responses route call `wire.AddPromptCacheKey` (`openai.go:269`, `grok.go:239`, `together.go:212`, `opencode.go:507`), which sends `prompt_cache_key` = the provider's session id "when the request asks for caching" (`internal/wire/cache.go:8-15`). Kilo sends OpenRouter's `cache_control` for a model whose id starts `anthropic/` (`kilo.go:320-321`). Gemini, Hugging Face and Ollama send nothing (their package docs).
- **The session id is per provider instance.** `WithSessionID` "sets the conversation id sent as x-opencode-session, Kilo's task id, and the ChatGPT backend's session-id. Empty keeps a random id, fixed for the provider's lifetime" (`llmprovider/identification.go:22-29`). A per-request id is out of the SDK's scope (its 0039-MADR D6).
- **A provider holds nothing to close.** `Provider` is `ID`, `Capabilities` and `Generate` (`contract.go:14-22`); no built-in provider starts a goroutine (`grep -rn 'go func' providers`, no match outside tests). Each instance gets its own `http.Client` over the one transport the process keeps, "so instances share connections" (`internal/transport/transport.go:45-51`, its 0028-MADR D-A2).
- **The SDK's live run** (its 0039-PLAN, Phase 4, 2026-10-13): a 7,086-token prompt sent twice per path with `Cache` set. Claude wrote 7,082 then read 7,082; with `CacheLong` it wrote 7,093, all of it `CacheWrite1hTokens`, then read; OpenCode Zen and Kilo's Claude wrote then read; OpenAI `gpt-6-luna` wrote 4,377 then read 4,377; Together read 4,544 on the second request; xAI read a 512-token prefix on both. In its first run, a "long" path read the entry a "short" path with the same prompt had written five seconds earlier: Anthropic's cache is keyed by the prefix, not by the session (its deviation 1).
- **gobble's request path** (`main` `007b63f`):
  - `llm.Request` (`llm/llm.go:29-34`) is `Model`, `Messages`, `Tools`, `Thinking`: no cache field.
  - A turn's model call builds it at `agent/agent.go:351`; the only other request is compaction's `summarise` (`compaction/compaction.go:301-305`), whose system prompt is compaction's own and whose user message is the serialised conversation.
  - `toRequest` (`llm/provider/provider.go:196-243`) joins the system messages into `Instructions` and decodes each tool's schema into `any`; it sets no `Cache`.
  - `New` (`llm/provider/provider.go:44-70`) passes no `WithSessionID`; `Options` (`:25-38`) has no session field.
  - `acpserver` builds one provider per Agent on first use and keeps it in `a.model` (`acpserver/agent.go:105`, `:664-679`), for every session's prompts (`:404`) and compactions (`acpserver/compact.go:21`, `:53`). `Options.Provider` is `func() (llm.Provider, error)` (`agent.go:56`).
  - A session leaves the Agent only on `session/close` (`agent.go:932`).
- **The cache counts in gobble:** `usageOf` maps `CachedTokens` and `CacheWriteTokens` (`provider.go:168-193`, since 0019-PLAN P3), and the all-zero estimate test includes the write. The agent sums both (`agent/agent.go:504-513`). Pi's stored usage keeps both (`acpserver/history.go:116-120`; `session/session.go:200-210`). `/session` and `_gobble/get_session_stats` show both (`acpserver/commands.go:148-223`, `extensions.go:496-507`). The ACP prompt response sets `CachedReadTokens` and not `CachedWriteTokens` (`acpserver/prompt.go:284-293`), though `acp.Usage` has both (acp-go-sdk2 `types_gen.go:19169-19188`, marked UNSTABLE; gobble already sends it, `agent.go:508`).
- **The prefix gobble sends** (`i2probe`, a scratch module that imports gobble's `llm/provider` and real built-in tools through a `replace` directive, with a transport that records each request and answers 400, so nothing reaches a service):
  - the 13 built-in tool specs are 16,780 bytes of JSON;
  - for claude, openai, kilo, opencode-zen, together and grok, ten model calls on one provider and one on a second instance, each from a freshly built `llm.Request`, gave **one** distinct body each: the tools, system prompt and history are byte-identical across calls and instances, and the tools keep the built-ins' order;
  - Kilo and OpenCode Zen send a 26-character session header per instance (`X-KiloCode-TaskId`, `x-opencode-session`), and a second instance sends a different one; the others send none today.
- **What `Cache` adds** (`i2probe/sdkcache`, the SDK alone, `WithSessionID` set to a UUIDv7 in the shape of a gobble session id, the same recording transport):

  | Provider | `Cache` `""` | `short` | `long` | Session header |
  | :--- | :--- | :--- | :--- | :--- |
  | claude | nothing | `cache_control` 5m, system block 5m | 1h, 1h | none |
  | openai, grok, together | nothing | `prompt_cache_key` = the session id | the same | none |
  | kilo (`anthropic/` model) | nothing | `cache_control` 5m | 1h | `X-KiloCode-TaskId` = the session id |
  | opencode-zen, `claude-haiku-5-5` | nothing | Anthropic's two markers, 5m | 1h | `x-opencode-session` = the session id |
  | opencode-zen, `gpt-6-luna` | nothing | `prompt_cache_key` = the session id | the same | the same |
  | opencode-zen, `deepseek-v4.1-flash` | nothing | nothing | nothing | the same |
  | gemini | nothing | nothing | nothing | none |

  Ten requests per row gave one body each. With an empty model, OpenCode Zen sent no marker; gobble never sends an empty model (the session's model is the choice's default, `acpserver/agent.go:297`, 0020-MADR D4).
- **The cost of a provider per session** (`i2probe`, `testing.Benchmark` of `provider.New`): about 7 µs, 15.2 KB and 100–102 allocations, for each of openai, claude, kilo, opencode-zen, together and grok.
- **What keeps the prefix stable in gobble:** the system prompt is a session's baseline; a later change is sent as a user message, "so the cached prefix holds" (`prompt/doc.go:8`; `acpserver/sysprompt.go:256-269`), and the prompt holds no date or time (no time call in `prompt/` or `acpserver/sysprompt.go`). The tools are the built-ins, then the MCP tools in server order and each server's own order (`mcpclient/manager.go:57-89`), then `exit_plan` in plan mode, then deferred tools in load order, appended (`acpserver/agent.go:446-456`; 0012-MADR D9).
- **Pi** (`pi` at `312184edb`): its option is `cacheRetention`, `none | short | long`, default `short`, with `PI_CACHE_RETENTION=long` for the hour (`packages/ai/src/api/anthropic-messages.ts:58-75`); its compaction asks for `"none"` under a fresh session id (`packages/coding-agent/src/core/compaction/compaction.ts:631-632`); its agent is built with the session's id, and each request carries it (`packages/coding-agent/src/core/sdk.ts:403-411`). Its warmer keeps one "active run" per session, the last request, and one timer, refreshing at 90 % of the TTL (`cache-warmer.ts:29-31`, `:141-163`) when the expected saving is at least $0.05 (`:20`), from each model's cache TTL and prices (`:39-46`, `:85`); its mode is the global-only `cacheWarming` setting, "because each refresh costs money" (`settings-manager.ts:180`). Pi has no user setting for the retention itself.
- **gobble's settings** (`internal/config/keys.go:58-89`): keys are dotted camelCase paths with a kind, a default and when they are read; a session key reaches `acpserver.SessionSettings` (`acpserver/config.go:31-45`) through `toSession` (`internal/cli/settings.go:84-92`). 0018-MADR D19: each later key comes "with the Go types, the schema, its drift test and the template block".
- **The records the issue names:** 0004-MADR's adapter configuration, item 6: "One provider instance per ACP session, so `WithSessionID` … is per session" (`0004-MADR-go-module-architecture.md:749`), not built; 0005-MADR: "Prompt-cache warming stays 1.x, and is blocked … The row lands only once the SDK supports Claude `cache_control` and the split" (`:578-581`); 0005-PLAN X5, step 6, the same block (`:725`, not `:719`); 0012-MADR F10 ("gobble cannot use provider-native deferral or cache control today", `:212-215`) and D13 (`:483-489`).

### Findings

- **F1. Two of the issue's points are already met.** The pin is `v1.12.0` (0020-PLAN P1), and `usageOf` reports cache writes, with the all-zero test covering them (0019-PLAN P3). What remains: no cache field, one provider and session id per Agent, the ACP prompt response's missing write count, and the records.
- **F2. One field asks every provider that can.** `Request.Cache` turns on Anthropic's markers on Claude, OpenCode's messages route and Kilo's Claude models, and the routing key on OpenAI, xAI, Together and OpenCode's responses route; nothing changes elsewhere, and the zero value changes nothing anywhere.
- **F3. Today every session in a process shares one cache identity.** The Agent's one provider has one random session id: Kilo's task id and OpenCode's session header are the same for every session, and once `Cache` is set, so is the routing key on OpenAI, xAI and Together. A second session's turns are then routed as the first's. 0004-MADR item 6 decided against this and was never built.
- **F4. A session id must come with a provider per session,** since the SDK takes it only at construction. A provider costs about 7 µs and 15 KB to build, holds nothing to close, and shares the process's connections, so one per session costs nothing measurable.
- **F5. gobble's prefix is already stable.** Its bodies are byte-identical across calls and provider instances; the system prompt is a session's baseline with patches sent as messages; the tools keep their order. So a cached prefix is read back on the next call of the same session, unless the tools change.
- **F6. The prefix changes in three known ways,** each a cache break from that request on, by design, since a service caches exact prefixes: a deferred tool's load (0012-MADR F10, D9), an MCP server's tools joining after start-up (`acpserver/mcp.go:153-166`), and entering or leaving plan mode (`exit_plan`). Anthropic also invalidates the message blocks when the thinking level or tool choice changes; D2's fixed breakpoint keeps the tools and system prompt cached across that.
- **F7. A compaction request should not ask.** Its prefix is its own system prompt and the serialised conversation, never sent again; a write that is never read costs 1.25 times the input on Anthropic. Pi asks for `"none"` there too.
- **F8. A single `-p` prompt still benefits.** One prompt's tool loop makes several model calls over the same prefix (`agent/agent.go:345-351`, per call), so the second call of a turn reads what the first wrote.
- **F9. No client-side cache structure is needed for the request path.** The service caches; gobble only has to say which session a request belongs to, and that state is the session itself: a provider kept on `liveSession` lives exactly as long as the session, needs no eviction, and is found through the sessions map the Agent already has.
- **F10. A ring buffer of recent requests has no reader.** The only consumer of past requests would be cache warming, and Pi's warmer keeps the last request alone; the per-call usage history is already in the session's entries, which `/session` and `_gobble/get_session_stats` read.
- **F11. A response cache would be wrong, and would never hit.** Each model call of a session carries a longer history than the last, so no two requests of a session are equal; the SDK's retry already re-sends a failed request. Serving a stored reply would also skip a model call that the agent loop and its tool side effects depend on.
- **F12. Warming is unblocked by the SDK and blocked by gobble's catalog.** Pi's warmer needs each model's cache TTL and prices to decide whether a refresh pays; gobble has neither until 0005-PLAN F4-2's catalog (0019-MADR D11). Without them a warmer can only re-send at a fixed interval, spending money on sessions that never return.
- **F13. `CacheWrite1hTokens` has no consumer in gobble yet.** Only Anthropic's Messages reports it, Pi's stored usage has no field for it, and its use is pricing a one-hour write at 2 times rather than 1.25 times, which needs F4-2's prices.

## Decision Drivers

- **The service does the caching** (0004-MADR: the library does the work): gobble asks, through the SDK's one field, and builds no cache of its own.
- **A request belongs to its session,** so the routing key and the gateways' session headers follow the conversation (0004-MADR item 6).
- **No cost the user did not choose:** never ask for a write that is never read (F7), and let the user choose the lifetime or turn the hint off.
- **The prefix stays stable**, as 0005-MADR's choice 8 and 0012-MADR D9 already require.
- **Nothing changes on the wire for `none`** (the SDK's R45), and nothing changes for a user who never sets the key, beyond the hint itself.
- **No test reaches a provider** (0002-PLAN Phase 3).
- **The owner's choices of 2026-10-13:** a session key `cacheRetention`, `none | short | long`, default `short`; the hint, the identity and the accounting now, with warming left to 0005-PLAN X5; `Options.Provider` takes the session id; and a billed OpenAI check of four requests.

## Considered Options

- **A. The hint on agent turns, and the session's provider kept on the session** (`liveSession`), with a `cacheRetention` key.
- **B. As A, with the providers in a separate in-memory cache,** keyed by session id, with LRU or TTL eviction.
- **C. As A, with a ring buffer of each session's recent requests and usage,** for warming and statistics.
- **D. A client-side response cache,** memoising replies to identical requests.
- **E. One provider per Agent, with the session id per request.**
- **F. As A, with cache warming now,** on a fixed TTL table.
- **G. Always on, with no setting.**

## Decision Outcome

Chosen option: "A", because it asks every provider that can through the SDK's one field, gives each conversation its own identity at no measurable cost, keeps the per-session state where the session already lives, and leaves warming and pricing to the catalog that can support them.

### The decisions

- **D1. `llm.Request` gains `Cache`.**
  - `llm` defines `type CacheRetention string` with `CacheNone CacheRetention = ""`, `CacheShort = "short"` and `CacheLong = "long"`, and `Request.Cache CacheRetention` (json `cache,omitzero`).
  - `toRequest` sets `llmprovider.Request.Cache` to the same value: the SDK's constants are the same strings. Any other value reaches the SDK, which refuses it before the network with `ErrInvalidRequest`, "unknown cache retention" (`contract.go:275-277`); gobble does not check it a second time, as 0019-MADR D7 stopped gobble checking what the SDK enforces.
  - `llm` stays free of the SDK (`internal/archtest`).
- **D2. Each model call of an agent turn asks; compaction never does.**
  - `agent.Config` gains `Cache llm.CacheRetention`, and `(t *turn) call` sets `Request.Cache` from it on every call of the turn (F8).
  - `compaction.summarise` builds its request without it (F7), and its `Model` gains no cache field.
- **D3. A session key, `cacheRetention`, chooses the hint.**
  - `none | short | long`, default `short`, read at session start (`ReadSession`), from any settings file or `--set`; it sits in the template's Model block after `defaultProvider`.
  - `none` sends the zero value. `long` asks for the service's longest lifetime: an hour on Anthropic's routes, the same as `short` elsewhere (the SDK's 0039-MADR D3).
  - `acpserver.SessionSettings` gains `CacheRetention string`, default `short`; a session keeps the value it started with, on `session/new`, load and resume alike.
  - It is added as 0018-MADR D19 says a key is added: its Go type, the schema, the drift test and the template block.
- **D4. One provider, and one session id, per ACP session.**
  - `llm/provider.Options` gains `SessionID`, sent as `llmprovider.WithSessionID` by `New`; `Ambient` passes it through.
  - `acpserver.Options.Provider` becomes `func(sessionID string) (llm.Provider, error)`. Each `liveSession` keeps its own provider, built on its first use (a prompt, `/compact` or `_gobble/compact`) with the session's ACP id, under `a.mu`, and kept for the session's life; `Agent.model` goes. A failed build is not kept, as today.
  - The provider leaves with the session on `session/close`, and needs no close of its own (F4).
  - A loaded or resumed session has its stored id, so it keeps its routing key across a restart; a fork or a clone has its own new id.
  - `internal/cli`'s `modelProvider` builds with `Options.SessionID` set to the id it is given.
- **D5. The ACP prompt response reports cache writes.** `acpUsage` sets `CachedWriteTokens` when the turn wrote, as it sets `CachedReadTokens`.
- **D6. No client-side cache structure** (F9–F11).
  - The per-session state is the session: its provider and its retention live on `liveSession`. There is no separate provider cache, no ring buffer, no response cache and nothing on disk.
  - If 0005-PLAN X5's warming lands, it keeps one "last cacheable request" per session, as Pi's warmer does, on `liveSession` beside the provider.
- **D7. Warming and one-hour pricing wait for the catalog** (F12, F13). 0005-PLAN X5's warming is no longer blocked by the SDK; it needs F4-2's per-model cache TTL and prices. `llm.Usage` gains no one-hour field until F4-2's cost work has a use for it.
- **D8. The records are corrected by dated notes:** 0004-MADR's adapter item 6 (built here); 0005-MADR `:578-581` and 0005-PLAN X5 step 6 (`:725`) (the SDK's block is lifted; the catalog's remains); 0012-MADR F10 and D13 (the cache-control request is met at go-llmprovider-sdk `v1.12.0`, and deferral is still a request); 0018-MADR D19 (`cacheRetention` is added here); and `llm/doc.go`'s request description. The owner closes issue 2 when the work is committed.
- **D9. One billed check against a live service,** approved by the owner: four requests to OpenAI `gpt-6-luna` through `gobble acp` with the owner's test key in the child's environment only, two turns of one session with `cacheRetention: short` and two of another with `none`. Claude's path rests on the SDK's recorded run.

### Consequences

- Good, because a session's second and later model calls read the prefix the first wrote, on every provider that caches: 0.1 times the input price per read on Anthropic, and OpenAI, xAI and Together route the session to one cache.
- Good, because each session has its own identity: Kilo's task id, OpenCode's session header and the routing key follow the conversation, and a resumed session keeps its key.
- Good, because compaction pays no write premium, and `none` restores today's requests byte for byte.
- Good, because the client reports the cache writes it is charged for.
- Neutral, because a provider per session costs about 7 µs and 15 KB, and shares the process's connections.
- Bad, because on Anthropic's routes the first call of a session pays 1.25 times its input for the write (2 times with `long`), which a session that ends after one call never recovers.
- Bad, because a prefix change (a deferred load, a late MCP server, plan mode) breaks the cache once each time, until the SDK has native deferral (0012-MADR D13).
- Bad, because `acpserver.Options.Provider` changes signature: its 18 call sites in `acpserver`'s tests change with it, as do `internal/cli`'s `modelProvider` and its two test uses. apidiff is a no-op below `v1.0.0` (`Makefile:163`, `:253-259`).
- Neutral, because a session that is never closed keeps its provider as it keeps the rest of its state.

### Confirmation

```text
go test ./llm/provider -run 'TestCache|TestSessionID'          -> ok: each wire's cache fields per retention, prompt_cache_key and the session headers = Options.SessionID, an unknown retention refused
go test ./acpserver -run 'TestCache|TestSessionProviders'      -> ok: a turn's calls carry the session's retention, a compaction carries none, two sessions build two providers with their own ids, cachedWriteTokens in the prompt response
go test ./internal/config -run 'TestKeys|TestRender|TestResolve|TestSchema' -> ok: cacheRetention with its default, template block and schema
go test ./internal/cli -run 'TestCacheRetentionSetting'        -> ok: the setting reaches the session's requests
make archtest                                                  -> ok: llm stays free of the SDK
the gates                                                      -> lint ×3; go test on Windows and the macOS laptop; -race on Windows and the second Linux machine; Windows preflight
the live check (D9)                                            -> short: a cache write, then a read; the key in no output
```

Each new test is seen failing on a scratch copy first (0021-PLAN's negatives).

## Pros and Cons of the Options

### A — the hint, and the provider on the session (chosen)

- Good, because the SDK's field does the work on every route, and the state gobble keeps is one field more on a struct it already has.
- Good, because the provider's lifetime is the session's, with no eviction policy to get wrong.
- Bad, because the provider signature changes, and with it every test that builds an Agent.

### B — a separate provider cache with eviction

- Good, because it could bound the number of providers in a long `gobble acp` with many sessions.
- Bad, because the sessions map already bounds them: a provider is 15 KB beside a session that holds its whole history; an LRU that evicted a live session's provider would hand that session a new id mid-conversation and break its routing.
- Bad, because it is a second map, a second lock and a second lifecycle to keep in step with the first.

### C — a ring buffer of recent requests

- Good, because a warmer or a statistics view could read recent calls from it.
- Bad, because nothing reads it today (F10): statistics come from the session's entries, and a warmer needs only the last request.
- Bad, because it would hold whole requests, histories included, in memory for every session.

### D — a client-side response cache

- Good, because a hit would cost nothing.
- Bad, because there are no hits (F11): a session's requests grow by at least one message each call.
- Bad, because a stored reply replayed into a tool loop would repeat side effects the user did not ask for.

### E — one provider per Agent, the id per request

- Good, because nothing in `acpserver` would change shape.
- Bad, because the SDK takes the session id only at construction (its 0039-MADR D6); it cannot be built today.

### F — warming now

- Good, because an idle interactive session would keep its cache past five minutes without `long`.
- Bad, because without the catalog's TTLs and prices it cannot tell a refresh that pays from one that does not (F12), and Pi itself makes the mode global-only because "each refresh costs money".

### G — always on, no setting

- Good, because there is nothing to configure, as in opencode, codex and grok-build (the SDK's 0039-MADR §3).
- Bad, because a user who pays per write, or runs one-shot prompts, could not turn it off, and `long` could not be chosen.

## More Information

### Evidence index

| Claim | Source |
| :--- | :--- |
| The pins, and the clones' state | `go.mod:15-17`; `git -C <clone> log -1`, 2026-10-13 |
| `CacheRetention`, `Request.Cache`, `Usage`'s cache counts | go-llmprovider-sdk `v1.12.0`, `llmprovider/contract.go:95-117`, `:173-189`, `:210-212`, `:275-277` |
| What each provider sends | `providers/claude/claude.go:307`; `providers/opencode/opencode.go:507`, `:531`; `openai.go:269`; `grok.go:239`; `together.go:212`; `kilo.go:320-321`; `internal/wire/cache.go:8-15`; `internal/wire/messages/cache.go:20-42` |
| The session id is per instance | `llmprovider/identification.go:22-29`; the SDK's 0039-MADR D6 |
| A provider holds nothing to close; shared connections | `contract.go:14-22`; `grep -rn 'go func' providers`; `internal/transport/transport.go:45-51` |
| The SDK's live run, and Anthropic's prefix keying | the SDK's `0039-PLAN-request-side-prompt-caching.md`, Phase 4, run 1, deviation 1, run 2 |
| gobble's request, provider and session code | `llm/llm.go:29-34`; `agent/agent.go:345-351`, `:504-513`; `compaction/compaction.go:301-305`; `llm/provider/provider.go:25-38`, `:44-70`, `:168-193`, `:196-243`; `acpserver/agent.go:56`, `:105`, `:404`, `:508`, `:664-679`, `:932`; `acpserver/compact.go:21`, `:53` |
| The ACP response's cache counts | `acpserver/prompt.go:284-293`; acp-go-sdk2 `v0.17.1` `types_gen.go:19163-19188` |
| One body per provider across calls and instances; the session headers | `i2probe` (scratchpad), 2026-10-13 |
| What `Cache` adds per provider and route | `i2probe/sdkcache` (scratchpad), 2026-10-13 |
| The cost of `provider.New` | `i2probe`, `testing.Benchmark` |
| The stable prefix and its three changes | `prompt/doc.go:8`; `acpserver/sysprompt.go:256-269`; `mcpclient/manager.go:57-89`; `acpserver/mcp.go:153-166`; `acpserver/agent.go:446-456`; 0012-MADR F10, D9 |
| Pi's retention, compaction, session id and warmer | `pi` `312184edb`: `packages/ai/src/api/anthropic-messages.ts:58-75`; `packages/coding-agent/src/core/compaction/compaction.ts:631-632`; `sdk.ts:403-411`; `cache-warmer.ts:20`, `:29-31`, `:39-46`, `:85`, `:141-163`; `settings-manager.ts:180` |
| How a key is added and read | `internal/config/keys.go:58-89`; `acpserver/config.go:31-45`; `internal/cli/settings.go:84-92`; 0018-MADR D19 |
| apidiff is a no-op below v1 | `Makefile:163`, `:253-259` |
| The records the issue names | `0004-MADR-go-module-architecture.md:749`; `0005-MADR-v1-feature-scope.md:578-581`; `0005-PLAN-v1-feature-scope.md:725`; `0012-MADR-tool-schemas-descriptions-and-loading.md:212-215`, `:483-489` |
| The issue | gobble-cli issue 2, opened 2026-10-13 |

### Related records

- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): the adapter configuration, item 6.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) and [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): prompt-cache warming (X5), the catalog (F4-2).
- [0012-MADR-tool-schemas-descriptions-and-loading.md](0012-MADR-tool-schemas-descriptions-and-loading.md): F10, D9, D13.
- [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md): D19, how keys are added.
- [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md) and [0020-MADR-full-model-catalog-listing.md](0020-MADR-full-model-catalog-listing.md): the pins this builds on.
- go-llmprovider-sdk, `docs/decisions/0039-MADR-request-side-prompt-caching.md` and its PLAN.

### Open questions for the plan

None. The owner answered the key and its default, the scope, the provider's shape and the live check on 2026-10-13.
