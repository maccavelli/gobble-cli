---
status: in-progress
date: 2026-10-13
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# PLAN 0021 — Ask for prompt caching on agent turns, with one provider and session id per ACP session

Implements [0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md) decisions D1–D9, closing findings F1–F13, and gobble-cli issue 2.

## Goal

1. A model call of an agent turn carries `llm.Request.Cache` from the session's `cacheRetention` (default `short`), on every call of the turn; a compaction request carries none; `none` sends what gobble sends today.
2. Each ACP session builds its own provider with its ACP session id, on its first prompt or compaction, and keeps it for its life: two sessions send two session ids (`prompt_cache_key`, `x-opencode-session`, `X-KiloCode-TaskId`), and a reloaded session sends its own again.
3. The ACP prompt response reports `cachedWriteTokens` beside `cachedReadTokens`.
4. `cacheRetention` is a session key (`none | short | long`, default `short`) with its Go type, schema, drift test and template block.
5. The records carry D8's notes, and four billed OpenAI requests show a cache write, then a read, through `gobble acp`.
6. The gates pass, and each new test has been seen failing on a scratch copy.

## Scope

### In scope (the only files any phase may touch)

| Phase | Decisions | Files |
| :--- | :--- | :--- |
| P1 | D1, D4 (the option) | `llm/llm.go`; `llm/doc.go`; `llm/provider/provider.go`; `llm/provider/cache_test.go` (new) |
| P2 | D2, D4, D5, D6 | `agent/agent.go`, `agent/agent_test.go`; `acpserver/agent.go`, `acpserver/compact.go`, `acpserver/config.go`, `acpserver/prompt.go`; `acpserver/cache_test.go` (new), `acpserver/usage_test.go`, `acpserver/settings_test.go`, and the 15 `acpserver` test files that build an `Options.Provider` (`agent_test.go`, `commands_test.go`, `compact_test.go`, `fixtures_test.go`, `list_test.go`, `mcp_test.go`, `prompt_test.go`, `rewrite_test.go`, `routing_test.go`, `session_test.go`, `settings_test.go`, `sysprompt_test.go`, `templates_test.go`, `toolload_test.go`, `tree_test.go`); `internal/cli/agent.go`, `internal/cli/sessions_test.go`, `internal/cli/settings_test.go`, `internal/cli/cache_test.go` (new) |
| P3 | D3 | `internal/config/keys.go`, `internal/config/settings.go`, `internal/config/settings.yaml`, `internal/config/testdata/settings.schema.json`, `internal/config/resolve_test.go`, `internal/config/render_test.go`; `internal/cli/settings.go`, `internal/cli/cache_test.go` |
| P4 | D7, D8, D9 | `docs/decisions/0004-MADR-go-module-architecture.md`, `docs/decisions/0005-MADR-v1-feature-scope.md`, `docs/decisions/0005-PLAN-v1-feature-scope.md`, `docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md`, `docs/decisions/0018-MADR-configuration-surface.md`; `docs/README.md`; this plan and its MADR |

### Out of scope

- **Cache warming:** 0005-PLAN X5, which needs F4-2's per-model TTL and prices (MADR D7, F12).
- **A one-hour write count in `llm.Usage`:** F4-2's cost work (MADR D7, F13).
- **Native tool deferral:** still a request to go-llmprovider-sdk (0012-MADR D13).
- **Choosing the retention per session from the client** (an ACP config option): the key is read at session start.
- **OpenAI's explicit `prompt_cache_options` and turning GPT-5.6's automatic caching off:** out of the SDK's scope (its 0039-MADR D6).

## Stability rule

Every phase ends with, in order:

```bash
export GOTOOLCHAIN=go1.27.2
make pre-add-check FILES="<the phase's .go files>"         # P4 has none
python -I <scratch>/q428_gates_0019.py 0021-<phase>         # lint for linux, darwin, windows; Windows go test; Windows -race;
                                                            # Windows make preflight; the second Linux machine's -race; the macOS laptop's go test
python -I <scratch>/q252_check_index.py 0021-<phase>        # gofmt, vet, test and check-records on the staged index alone
python -I <scratch>/q261_lint_records.py --negative <each record the phase touches>
```

- `q428_gates_0019.py` takes the phase as a label and is reused unchanged. No gate runs in WSL. Host aliases stay in the scripts and are never written into a record.
- **Commits.** The agent stages each phase with `git add` and never commits. The owner commits each phase with `git commit --no-edit` before the next begins. No push and no tag without the owner's request in that turn.
- **Negatives** run on a scratch copy of the working tree, after a control run of the same tests passes; each asserts that its edit landed, and each runs only its own tests.

## Cross-cutting contracts

- **C1. No test reaches a provider.** `llm/provider`'s tests use a transport that records each request and answers 400, which the SDK does not retry; `acpserver`'s and `internal/cli`'s use `llmtest` scripts.
- **C2. `none` is today's wire.** A request with `Cache` `""` carries no `cache_control` and no `prompt_cache_key` on any provider (P1's test asserts it per row).
- **C3. The compaction request never asks.** `compaction` gains no cache field.
- **C4. No credential in output.** The live check (P4) puts the key in the child's environment only and checks every output and every file of its temporary home.
- **C5. No test is weakened.** The 18 `Options.Provider` call sites change only their signature; every assertion stays.

**C3 is the contract most at risk:** compaction uses the session's provider, so a later change that sets `Cache` on the provider rather than on the request would reach compaction silently; P2's negative 2 is what shows the test can see it.

## Dependency and delivery order

| # | Phase | Why here |
| :--- | :--- | :--- |
| 1 | P1 — the field and the session id in `llm` | Everything else sets or reads them. |
| 2 | P2 — the turn's hint and the session's provider | Needs P1; `internal/cli` must compile with `acpserver`'s new signature in the same commit. |
| 3 | P3 — the `cacheRetention` key | Needs P2's `SessionSettings.CacheRetention`. |
| 4 | P4 — the records and the live check | Cites P1–P3's commits; the live check needs P2's write count. |

## Implementation Steps

### P1 — The field and the session id (D1, D4's option; closes F1, F2, F4)

1. **`llm/llm.go`:** before `// Request is one model call.`, add

   ```go
   // CacheRetention asks the model service to cache a request's prompt, and
   // for how long (0021-MADR D1). CacheNone asks for nothing; CacheShort is
   // the service's short lifetime, five minutes on Anthropic; CacheLong its
   // longest, an hour on Anthropic, and CacheShort's where it has one. The
   // values are go-llmprovider-sdk's.
   type CacheRetention string

   // The retentions.
   const (
       CacheNone  CacheRetention = ""
       CacheShort CacheRetention = "short"
       CacheLong  CacheRetention = "long"
   )
   ```

   and in `Request`, after `Thinking`: `` Cache CacheRetention `json:"cache,omitzero"` `` with the comment `// Cache asks the service to cache the prompt; CacheNone asks for nothing.`
2. **`llm/doc.go`:** after the `Usage` paragraph (`:47-50`), add: "Request.Cache asks the service to cache the prompt; an agent turn sets it and a compaction does not (0021-MADR D2)."
3. **`llm/provider/provider.go`:**
   - `Options`, after `ModelMetadataURL`:

     ```go
     // SessionID names the conversation: the routing key a cached request
     // carries, and OpenCode's and Kilo's session headers (0021-MADR D4).
     // Empty keeps the SDK's random id, fixed for the provider.
     SessionID string
     ```

   - `New`, after the `HTTPClient` option: `if o.SessionID != "" { opts = append(opts, llmprovider.WithSessionID(o.SessionID)) }`.
   - `toRequest`: `out := &llmprovider.Request{Model: req.Model, Reasoning: reasoningFor(req.Thinking), Cache: llmprovider.CacheRetention(req.Cache)}`.
4. **`llm/provider/cache_test.go` (new),** package `provider`, each test seen failing first:
   - `wire400`, an `http.RoundTripper` that records each request's decoded JSON body and its `x-opencode-session` and `X-KiloCode-TaskId` headers, and answers 400 `{"error":{"message":"test"}}`;
   - **`TestCacheWire`**, one row per (provider, model, retention), `provider.New(id, Options{APIKey: "k", SessionID: "sess-1", HTTPClient: …})` and one `Stream` of a request with a system message and a user message:

     | Provider, model | `Cache` | Body | Header |
     | :--- | :--- | :--- | :--- |
     | claude, `claude-haiku-5-5` | `""` | no `cache_control`; `system` a string | — |
     | claude, `claude-haiku-5-5` | `short` | `cache_control` `{"type":"ephemeral","ttl":"5m"}`; `system[0].cache_control` the same | — |
     | claude, `claude-haiku-5-5` | `long` | both with `"ttl":"1h"` | — |
     | openai, `gpt-6-luna` | `""` | no `prompt_cache_key` | — |
     | openai, `gpt-6-luna` | `short` | `prompt_cache_key` = `sess-1` | — |
     | kilo, `anthropic/claude-haiku-5.5` | `short` | `cache_control` with `"ttl":"5m"` | `X-KiloCode-TaskId: sess-1` |
     | opencode-zen, `claude-haiku-5-5` | `long` | `cache_control` and `system[0].cache_control` with `"ttl":"1h"` | `x-opencode-session: sess-1` |
     | gemini, `gemini-3.5-flash-lite` | `short` | no `cache_control`, no `prompt_cache_key` | — |

   - **`TestCacheUnknownRefused`:** `Cache: "forever"` gives an error matching `llm`'s wrapping of the SDK's `ErrInvalidRequest` whose text holds `unknown cache retention`, and the transport saw no request;
   - **`TestSessionIDPerProvider`:** two opencode-zen providers with `SessionID` `a` and `b` send `x-opencode-session` `a` and `b`; one with none sends a non-empty header that is neither.

**P1's negatives** (`q511_0021_p1_negatives.py`):

1. `toRequest` without `Cache`: `TestCacheWire`'s `short` and `long` rows fail.
2. `toRequest` sends `CacheShort` for every non-empty value: the `long` rows fail.
3. `New` without `WithSessionID`: `TestSessionIDPerProvider` and the `prompt_cache_key` and header rows fail.
4. `toRequest` maps an unknown value to `""`: `TestCacheUnknownRefused` fails.

**P1's verification:**

```text
go test -count=1 ./llm/provider -run 'TestCache|TestSessionID'               -> ok
go test -count=1 ./llm/... ./agent/... ./acpserver/... ./internal/cli/...    -> ok (nothing sets Cache yet, so nothing else changes)
make archtest                                                                -> ok: llm imports no SDK
```

### P2 — The turn's hint and the session's provider (D2, D4, D5, D6; closes F3, F5–F11)

1. **`agent/agent.go`:**
   - `Config`, after `Thinking`:

     ```go
     // Cache asks the service to cache each model call's prompt
     // (0021-MADR D2); CacheNone asks for nothing.
     Cache llm.CacheRetention
     ```

   - `(t *turn) call` (`:351`): `req := &llm.Request{Model: t.a.cfg.Model, Messages: msgs, Tools: t.specs, Thinking: t.a.cfg.Thinking, Cache: t.a.cfg.Cache}`.
2. **`acpserver/config.go`:**
   - `SessionSettings`, after `Thinking`:

     ```go
     // CacheRetention is the prompt cache the session's turns ask for:
     // none, short or long (0021-MADR D3).
     CacheRetention string
     ```

   - `sessionSettings` (`:51-63`): `s.CacheRetention = cmp.Or(s.CacheRetention, cacheShort)`, with `const cacheShort = "short"` beside `thinkingOff`;
   - a new function:

     ```go
     // cacheOf is a cacheRetention setting's hint: none asks for nothing;
     // short and long are llm's values, as any other is, for the SDK to
     // refuse (0021-MADR D1, D3).
     func cacheOf(setting string) llm.CacheRetention {
         if setting == "none" {
             return llm.CacheNone
         }
         return llm.CacheRetention(setting)
     }
     ```

3. **`acpserver/agent.go`:**
   - `Options.Provider` (`:52-56`): the type is `func(sessionID string) (llm.Provider, error)`, and its comment adds: "It is called once per session, with the session's ACP id, so each session has its own provider and session id (0021-MADR D4)."
   - `Agent`: `provider func(sessionID string) (llm.Provider, error)`; the `model llm.Provider // built on the first prompt` field (`:105`) goes.
   - `liveSession`, after `model`:

     ```go
     // provider is the session's own model provider, built on its first
     // use with the session's id; cache is the hint its turns send
     // (0021-MADR D3, D4).
     provider llm.Provider
     cache    llm.CacheRetention
     ```

   - both constructors (`:297`, `:741`) add `cache: cacheOf(ss.CacheRetention)`;
   - `modelProvider` (`:663-679`) becomes

     ```go
     // sessionProvider builds s's provider on first use, with s's id, and
     // keeps it for the session (0021-MADR D4). A failed build is not kept.
     func (a *Agent) sessionProvider(s *liveSession) (llm.Provider, error) {
         a.mu.Lock()
         defer a.mu.Unlock()
         if s.provider != nil {
             return s.provider, nil
         }
         if a.provider == nil {
             return nil, ErrNoProvider
         }
         p, err := a.provider(string(s.id))
         if err != nil {
             return nil, err
         }
         s.provider = p
         return p, nil
     }
     ```

   - the prompt (`:404`) calls `a.sessionProvider(s)`, and its `agent.Config` (`:457-467`) adds `Cache: s.cache`.
4. **`acpserver/compact.go`:** `:21` calls `a.sessionProvider(t.s)`, `:53` `a.sessionProvider(s)`; nothing else changes (C3).
5. **`acpserver/prompt.go`, `acpUsage`** (`:284-293`): after the read, `if u.CacheWriteTokens > 0 { out.CachedWriteTokens = new(count(u.CacheWriteTokens)) }`.
6. **The 18 call sites** (C5): in the 15 `acpserver` test files of the scope, `Provider: func() (llm.Provider, error)` becomes `Provider: func(string) (llm.Provider, error)`, by a script that asserts 18 replacements and no other edit.
7. **`internal/cli/agent.go`:**

   ```go
   // ambientProvider builds the ambient provider; a test replaces it to see
   // the options it is given.
   var ambientProvider = provider.Ambient

   var modelProvider = func(e *runEnv) func(sessionID string) (llm.Provider, error) {
       return func(sessionID string) (llm.Provider, error) {
           p, _, err := ambientProvider(os.Getenv, e.defaultProvider(), provider.Options{ClientName: appName, ClientVersion: identity().Version,
               Logger: e.log(), SessionID: sessionID})
           return p, err
       }
   }
   ```

   and its comment adds "Each session's provider names the session (0021-MADR D4)."; `internal/cli/sessions_test.go`'s `withModel` and `internal/cli/settings_test.go:269` take the new signature (`modelProvider(e)("s1")`).
8. **Tests,** each seen failing first:
   - **`agent/agent_test.go`, `TestCacheOnEveryCall`:** a script of a `read` tool call, then text; `Config{Cache: llm.CacheLong}` sends both requests with `CacheLong`, and a `Config` without it sends both with `""`, as `TestToolRoundTrip` runs a turn.
   - **`acpserver/cache_test.go` (new):**
     - **`TestSessionProvidersHaveTheirIDs`:** an `Options.Provider` that records each id and returns one script; two sessions, prompts to the first, the second, then the first again, record exactly `[s1, s2]`; after `session/close` and `session/load` of the first, one more prompt records `[s1, s2, s1]`.
     - **`TestTurnAsksForCache`:** for `Settings` `nil` (the default), `{CacheRetention: "long"}` and `{CacheRetention: "none"}`, a prompt that calls `read` then answers sends two requests, each with `short`, `long` or `""`.
     - **`TestCompactionAsksNoCache`:** as `TestCompactShortensContext` (`compact_test.go:58-97`): two prompts, `/compact`, a third prompt; requests 0, 1 and 3 carry `short`, and request 2, the summary, `""`.
   - **`acpserver/usage_test.go`, `TestACPUsageCacheWrites`:** `acpUsage(llm.Usage{InputTokens: 100, OutputTokens: 5, CacheReadTokens: 30, CacheWriteTokens: 10})` has `CachedReadTokens` 30 and `CachedWriteTokens` 10; with no write, `CachedWriteTokens` is nil.
   - **`acpserver/settings_test.go`, `TestNilSettingsIsToday`:** `today` and `want` gain `CacheRetention: "short"`.
   - **`internal/cli/cache_test.go` (new), `TestModelProviderSessionID`:** with `ambientProvider` replaced by one that records `o.SessionID`, `modelProvider(e)("sess-9")` records `sess-9`.

**P2's negatives** (`q512_0021_p2_negatives.py`):

1. The turn's `agent.Config` without `Cache`: `TestTurnAsksForCache` fails.
2. `compaction.summarise`'s request with `Cache: llm.CacheShort`: `TestCompactionAsksNoCache` fails.
3. `sessionProvider` keeps the provider on the Agent (one for all sessions): `TestSessionProvidersHaveTheirIDs` fails (`[s1]`).
4. `sessionProvider` builds a provider on every call: `TestSessionProvidersHaveTheirIDs` fails (`[s1, s2, s1, …]`).
5. `acpUsage` without the write: `TestACPUsageCacheWrites` fails.
6. `cacheOf` maps `none` to `short`: `TestTurnAsksForCache`'s `none` row fails.
7. `(t *turn) call` sets `Cache` only on a turn's first call: `TestCacheOnEveryCall` fails.
8. `modelProvider` without `SessionID`: `TestModelProviderSessionID` fails.

**P2's verification:**

```text
go test -count=1 ./agent/ -run TestCacheOnEveryCall                                    -> ok
go test -count=1 ./acpserver -run 'TestSessionProviders|TestTurnAsksForCache|TestCompactionAsksNoCache|TestACPUsage|TestNilSettings' -> ok
go test -count=1 ./internal/cli -run TestModelProviderSessionID                        -> ok
grep -rn 'func() (llm.Provider, error)' --include=*.go acpserver internal/cli           -> no match
grep -n 'a.model\b' acpserver/*.go                                                    -> no match
git diff --exit-code HEAD -- compaction                                                -> no diff (C3)
```

### P3 — The `cacheRetention` key (D3; closes F7, F8)

1. **`internal/config/keys.go`:** after `defaultProvider`'s entry,

   ```go
   {Path: "cacheRetention", Kind: KindEnum, Default: "short", Enum: []string{"none", "short", "long"},
       Read: ReadSession, Doc: "How long the model service keeps a session's prompt cached: none, short (five minutes on Anthropic's models) or long (an hour)."},
   ```

   and the `Keys` comment names "0021-MADR D3's cacheRetention".
2. **`internal/config/settings.go`:** `Settings` gains `` CacheRetention string `yaml:"cacheRetention" json:"cacheRetention,omitempty"` `` after `DefaultProvider`, and `field` a `case "cacheRetention": return &s.CacheRetention`.
3. **`internal/config/settings.yaml`,** in the Model block after `defaultProvider`'s:

   ```yaml
   # How long the model service keeps this session's prompt cached, so
   # the next request reads it back: none, short or long. On Anthropic's
   # models short is five minutes and long an hour; elsewhere both ask
   # the same. A write costs more than plain input there (1.25 times, or
   # 2 times for long), and a read a tenth. none asks for nothing.
   # cacheRetention: short
   ```

4. **`internal/config/testdata/settings.schema.json`:** regenerated, `go run ./internal/config/genschema`.
5. **`internal/config/resolve_test.go`:** the defaults table gains `{"cacheRetention", s.CacheRetention, "short"}`; the count is 15, "0018-MADR D14's 13, 0019-MADR D15's defaultProvider and 0021-MADR D3's cacheRetention". **`internal/config/render_test.go`:** `want` gains `cacheRetention: short` after `defaultProvider: ""`.
6. **`internal/cli/settings.go`, `toSession`:** `CacheRetention: s.CacheRetention,`.
7. **`internal/cli/cache_test.go`, `TestCacheRetentionSetting`,** each run through `runIn` with `withModel` and a script answering text: with no settings file the request carries `short`; with `cacheRetention: long` in `<home>/settings.yaml`, `long`; with `--set cacheRetention=none`, `""`; with `cacheRetention: forever`, the request carries `short`, the key's default, and the log holds the warning `cacheRetention: must be one of none, short, long` (a session key's diagnostics go to the log, `internal/cli/settings.go:57-63`).

**P3's negatives** (`q513_0021_p3_negatives.py`):

1. `toSession` without `CacheRetention`: `TestCacheRetentionSetting`'s `long` and `none` rows fail (they get `short`).
2. The key's default `long`: `resolve_test.go`'s defaults table and `TestCacheRetentionSetting`'s first row fail.
3. The template without the block: the package's template drift test fails.
4. The key without `Read: ReadSession`: `TestCacheRetentionSetting`'s `long` row fails.

**P3's verification:**

```text
go test -count=1 ./internal/config ./internal/cli -run 'TestKeys|TestRender|TestResolve|TestSchema|TestCacheRetentionSetting' -> ok
go run ./internal/config/genschema && git diff --exit-code internal/config/testdata/settings.schema.json  -> no diff after staging
gobble config get cacheRetention                                                        -> short
```

### P4 — The records and the live check (D7, D8, D9; closes F12, F13)

Each note is dated the day P4 runs, cites 0021-MADR's decision and the commit of the phase that carried it, and is added beside the text it corrects, which stays.

1. **`docs/decisions/0004-MADR-go-module-architecture.md`,** the adapter item "One provider instance per ACP session" (`:749`): "built in 0021-PLAN P2 (`<P2's commit>`, 0021-MADR D4)".
2. **`docs/decisions/0005-MADR-v1-feature-scope.md`,** "Prompt-cache warming stays 1.x, and is blocked" (`:578-581`): "go-llmprovider-sdk `v1.12.0` sends cache hints and splits reads from writes, and gobble asks on every turn (0021-MADR D2); warming now waits for 0005-PLAN F4-2's per-model cache TTL and prices (0021-MADR D7)".
3. **`docs/decisions/0005-PLAN-v1-feature-scope.md`,** X5 step 6 (`:725`), within 200 characters a line: "the SDK's block is lifted at `v1.12.0` (0021-MADR D7); the skip reason becomes F4-2's TTL and prices".
4. **`docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md`:** at F10 (`:212`), "cache control: gobble asks from 0021-PLAN P2 (`<P2's commit>`); deferral is still missing"; at D13's cache-control item (`:485`), "met at go-llmprovider-sdk `v1.12.0` (0021-MADR D1, D2)".
5. **`docs/decisions/0018-MADR-configuration-surface.md`,** D19 (`:565`): "`cacheRetention` was added by 0021-PLAN P3 (`<P3's commit>`, 0021-MADR D3), as this decision says a key is added".
6. **The live check** (`q514_0021_live.py`, D9): `gobble acp` built from the working tree, `OPENAI_API_KEY` from the owner's test file in the child's environment only, `GOBBLE_HOME` a temporary directory; a scratch ACP client sends `initialize`, `session/new` in a temporary directory, then the prompt "Reply with the single word: ok." twice, and prints each prompt response's `inputTokens`, `cachedReadTokens` and `cachedWriteTokens`. Session A runs with no settings (`short`); session B, in a second process, with `cacheRetention: none` in its home's `settings.yaml`, and a temporary directory of its own, so that its system prompt differs from A's. Four model requests in all.
   - **A must show** a cache write on the first prompt and a read on the second (`cachedReadTokens` > 0); a first prompt with no write, which OpenAI reported on `gpt-6-luna` in the SDK's run, is a deviation.
   - **B is recorded, not asserted:** OpenAI caches on its own, with or without the key.
   - The key is in no output and in no file under either home.
7. **`docs/README.md`:** the 0021 rows read `accepted` and `completed (<date>)`; the 0004-MADR, 0005-MADR, 0005-PLAN, 0012-MADR and 0018-MADR rows note their 0021 notes; the build-order list gains `**0021-PLAN.** Run: P1–P4 on <date>.`
8. **This plan:** `status: completed`, and its `## Execution record (<date>)`: each phase's commit, gates and negatives, the live check's counts, and what this plan predicted wrongly.

**P4's verification:**

```text
make check-records && make markdownlint                                     -> ok, 0 issues
python -I <scratch>/q261_lint_records.py --negative <every record P4 touches except the two 0005 records> -> only the planted violation
python -I <scratch>/q463_lint_vs_head.py docs/decisions/0005-MADR-v1-feature-scope.md   -> added 0
python -I <scratch>/q463_lint_vs_head.py docs/decisions/0005-PLAN-v1-feature-scope.md   -> added 0
grep -c '0021-MADR' <each record of steps 1-5>                              -> at least 1 each
```

## Verification (whole plan)

After P4:

```text
go list -m github.com/maccavelli/go-llmprovider-sdk                         -> v1.12.0 (no pin change)
git diff --exit-code 007b63f -- go.mod go.sum                                -> no diff
git diff --exit-code 007b63f -- compaction                                   -> no diff (C3)
python scripts/check_libs.py                                                 -> every library at its latest tag; exit 0
the gates                                                                    -> lint ×3, tests on three systems, -race on two, preflight
the live check                                                               -> session A: a write, then a read
```

### Acceptance criteria (mapped to MADR Confirmation)

| # | Criterion | MADR |
| :--- | :--- | :--- |
| A1 | Each wire's cache fields per retention, from gobble's request; `""` sends none | D1, F2, C2 |
| A2 | `Options.SessionID` reaches the routing key and the gateways' headers; two providers, two ids | D4, F3, F4 |
| A3 | Every model call of a turn carries the session's retention; compaction carries none | D2, F7, F8, C3 |
| A4 | One provider per session, built once, rebuilt with the same id after a reload | D4, F9 |
| A5 | The ACP prompt response reports cache writes | D5 |
| A6 | `cacheRetention` with its default, schema, drift test and template, reaching the request | D3 |
| A7 | No ring buffer, provider cache, response cache or disk cache added | D6, F9–F11 |
| A8 | The records carry D8's notes | D7, D8 |
| A9 | The live check shows a write, then a read, and no key in any output | D9, C4 |

**A3 is the criterion most likely to be dropped:** asserting the hint on a turn's first call passes even if a later call of the same turn loses it, and those later calls are where a `-p` run's saving is (F8); `TestCacheOnEveryCall` and P2's negative 7 hold it.

## Rollout and Rollback

- **Rollout:** one commit per phase, in order. Users get `short` from P2's commit, through `acpserver`'s default; P3 makes it a setting.
- **Rollback:** `cacheRetention: none` turns the hint off without a revert. A phase reverts with `git revert` of its commit: P3 alone leaves `short` fixed; P2 restores one provider per Agent and no hint; P1 is then unused.

## Deferred (named, so they are not mistaken for oversights)

- **Prompt-cache warming:** 0005-PLAN X5, after F4-2's per-model TTL and prices; it keeps one "last cacheable request" per session on `liveSession` (MADR D6).
- **`CacheWrite1hTokens`:** F4-2's cost work, which can price a one-hour write.
- **Native deferral:** the SDK's, 0012-MADR D13; until then each mid-session load breaks the cache once.
- **A per-session retention chosen from the client:** no ACP config option is added; a later record may add one.
