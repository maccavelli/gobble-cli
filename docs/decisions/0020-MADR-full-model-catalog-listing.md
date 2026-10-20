---
status: accepted
date: 2026-10-13
decision-makers: repository owner
consulted: none
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Pin go-llmprovider-sdk v1.11.0 and offer each provider's full curated catalog from a live listing

## Context and Problem Statement

The owner asked on 2026-10-13 to "update the pins to current lib and sdk versions" and to "assess the open issue and remediate it". Of the three sibling libraries, only go-llmprovider-sdk has a tag newer than gobble's pin: `v1.11.0`, against `v1.10.2` (0019-PLAN P3). acp-go-sdk2 `v0.17.1` and go-selfupdate-lib `v1.15.2` are pinned at their latest tags. The repository's one open issue, gobble-cli issue 1, asks gobble to list each provider's full curated catalog through that release instead of offering `Descriptor.StaticModels`: "gobble-cli is an agent harness, so its model picker should offer every model a provider serves. Today it offers only the SDK's static list, at most six models per provider, and refuses any other model."

The issue names its evidence at an older commit (`ad6ea57`, SDK `v1.2.1`); every claim below is re-measured at gobble's `main` after 0019-PLAN P4 (`17cd497`). The SDK's own decision is go-llmprovider-sdk's `0038-MADR-full-curated-catalog-by-default.md`, whose D7 names gobble-cli's adoption: "gobble-cli moves from `StaticModels` to `catalog.List` (or `ListModels`) with no limit."

### What was measured, not assumed

- **The tags** (`git ls-remote --tags`, 2026-10-13): acp-go-sdk2 `v0.17.1` latest; go-llmprovider-sdk `v1.10.1`, `v1.10.2`, `v1.11.0` (`d1fbad550fc8`), tagged 2026-10-13 09:54 (UTC-5); go-selfupdate-lib `v1.15.2` latest.
- **`python scripts/check_libs.py --lib go-llmprovider-sdk --keep`** at `v1.11.0`: get and tidy, the version after tidy, no other module, build, vet for linux, darwin and windows, gobble's tests, govulncheck and the advisories all ok. apidiff `v1.10.2`→`v1.11.0`: "Compatible changes: `catalog.WithLimit`: added; `wizard.Options.ModelLimit`: added", and nothing else. `go.mod` changes in one line; `go.sum` gains `h1:oCZmUe6hGMCZBS9lYqogTyndueF2CyWowJE6xxUqHoE=`, and its `go.mod` sum, `h1:en9k+ZhlMT41q/+FtXQrLE/3qDaBrWFr8TeLyZw2+Jg=`, is `v1.10.2`'s.
- **The live listing,** `catalog.List` at `v1.11.0` from a scratch module (`i1probe`, run by `q492_i1_probe.py`, which puts the owner's test key in the child's environment only and checks that no output contains it):

  | Provider | Profile | Metadata | Live | Recommended | Usable | Took | Head is the static six |
  | :--- | :--- | :--- | :--- | ---: | ---: | :--- | :--- |
  | openai | utility | off | yes | 73 | 73 | 1.306 s | yes |
  | openai | capable | off | yes | 73 | 73 | 548 ms | yes |
  | openai | utility | on | yes | 73 | 73 | 1.014 s | yes |
  | openai | capable | on | yes | 73 | 73 | 1.228 s | yes |
  | openai | utility, again | on | yes | 73 | 73 | 1.006 s | yes |
  | kilo | utility | off | yes | 272 | 285 | 1.911 s | yes |
  | kilo | capable | off | yes | 285 | 285 | 273 ms | no: `openai/gpt-6-astra` first |
  | kilo | utility | on | yes | 272 | 285 | 340 ms | yes |
  | kilo | capable | on | yes | 285 | 285 | 334 ms | no |

  Kilo's listing is public; it was listed with a placeholder credential, and the OpenAI key went to OpenAI only. A second run (`i1kilo`) later the same day counted 273 and 286, so the numbers move with the providers. Under the capable profile the static default `deepseek/deepseek-v4.1-flash` is still in Kilo's list, not first. OpenCode Zen answered the placeholder 401 and fell back to its static six, so the other open catalogs were not measured live; the issue's table gives the SDK's own counts (OpenCode Zen about 78, OpenCode Go 29, Hugging Face 100, Together 19, Claude 14, Gemini 12, Grok 8).
- **A failed listing degrades.** With a rejected key, OpenAI's listing returned `Live` false, the static six and `Err` "llmprovider: authentication failed: openai HTTP 401 invalid_api_key: Incorrect API key provided: sk-i1-no\*\*\*-key ..." in 126 ms. The provider's message echoes a masked fragment of the key it was sent.
- **Over a canned transport** (`i1canned`, `i1kilo/canned`, `i1kilo/zenurls`; no network):
  - an OpenAI listing of the static six, `gpt-5-mini`, `gpt-5-nano`, `text-embedding-4-large` and `tts-2` gives `Live` true and `Recommended` = the six, then `gpt-5-nano`, `gpt-5-mini`; the embedding and speech ids are left out, and an id the SDK does not know (`gpt-7-nova`) is left out too. The only request is `GET api.openai.com/v1/models`;
  - a 401 gives `Live` false and the static six; a listing that hangs past a 300 ms context gives the same, in 301 ms;
  - a Kilo listing of `kilo-auto/efficient` and `deepseek/deepseek-v4.1-flash` (`GET api.kilo.ai/api/gateway/models`) recommends only the second under the utility profile, and both, `kilo-auto/efficient` first, under the capable profile;
  - an OpenCode Zen listing requests `GET opencode.ai/zen/v1/models` and then `GET models.opencode.ai/api.json` with model metadata on, and only the first with it off. OpenAI and Kilo request no metadata either way. *(2026-10-13, 0020-PLAN P1 deviation 1: not "then". The SDK fetches the document concurrently with the listing and waits for it only when the listing answers; after a failed listing the request was made before `List` returned in 14 of 50 runs, and after an answering one in 50 of 50.)*
- **The metadata document** `https://models.opencode.ai/api.json`: 200, 394 571 bytes, 0.47 s (`curl --compressed`). The SDK caches it in the process (`llmprovider/catalog/model_metadata.go:23-27`).
- **gobble's code** at `17cd497`:
  - `llm/provider/ambient.go:117-126`: `Models` returns `slices.Clone(d.StaticModels)`; `:62-67`: `selection` takes `d.StaticModels[0]` as the model.
  - `internal/cli/agent.go:80-88`: `modelChoice` is `Choose`, then `ModelChoice{Provider: sel.ID, Models: provider.Models(sel.ID), Default: sel.Model}`; it is `acpserver.Options.Models` (`:43`) for every mode, since `gobble acp` (`internal/cli/acp.go:26`), the line session and `-p` (`agentServe`, `internal/cli/agent.go:117-121`) all serve `e.agentOptions()`.
  - `acpserver/config.go:86-92` publishes `choice.Models` as the `model` select; `:116-119` refuses any other value, "unknown model %q"; `acpserver/commands.go:289-297` prints every model for `/model` with no argument and refuses one outside the list.
  - The agent reads `a.models()` at `session/new` (`acpserver/agent.go:297`, `:323`), at restore (`:339`), for each assistant message's provider (`:553`), at load (`:741`, `:773`), at resume (`:809`), in `setConfig` (`config.go:113`) and in `/model` (`commands.go:284`). `Options.Models` is `func() ModelChoice` (`agent.go:59`), with no context.
  - A prompt's request carries the session's chosen model (`acpserver/agent.go:428`, `:461`; `agent/agent.go:351`), so the provider `Ambient` builds with `StaticModels[0]` answers whatever model the session chose.
  - `llm/provider/provider.go:41-46`: `New` sets `WithModelProbes(false)` and `WithoutModelMetadata()`, "Construction makes no network call".
  - `internal/cli/testmain_test.go:16-25` clears every credential variable before the tests, but `TestDefaultProviderSetting` sets `OPENCODE_API_KEY=test-only` and calls `e.modelChoice()` (`internal/cli/settings_test.go:255-266`).
  - `internal/archtest/check.go:190`, `:249`: only `llm/provider` may import the SDK, any of its packages.
  - `--offline` is reserved for 0005-PLAN X3 (`internal/cli/flags.go:52`, `:69`).
- **gobble's records:** 0005-PLAN F4-2 plans "the `model` option lists the catalog's models for the session's provider", from gobble's catalog; 0019-MADR D11 re-bases F4-2 on the SDK's `catalog.ModelInfo`; `docs/reports/0014-REPORT-llmprovider-sdk-integration.md` says configuring and `gobble models` "go through `wizard` and `catalog` via `llm/provider`" and "Do not implement a gobble-owned model catalog".

### Findings

- **F1. go-llmprovider-sdk `v1.11.0` is the only newer tag, and it passes every check.** Its exported API only grows, by `catalog.WithLimit` and `wizard.Options.ModelLimit`; gobble builds and tests on it unchanged.
- **F2. `v1.11.0` changes one default, and gobble does not reach it.** A listing's `Recommended` becomes the full curated catalog, the six best first (SDK 0038-MADR D1); the static lists stay at most six (D3). gobble calls neither `catalog.List` nor a provider's `ListModels`, so the bump alone changes nothing gobble does.
- **F3. gobble offers at most six models per provider, and refuses the rest.** The ACP picker, `set_config_option` and `/model` read `provider.Models`, which is `StaticModels`; a model outside it is "unknown". OpenAI serves 73 curated models and Kilo 285 under the capable profile.
- **F4. The capable profile changes the head of the open catalogs.** On Kilo it adds 13 models the utility profile leaves out (`kilo-auto/*`, dense models of 14B or less) and puts `openai/gpt-6-astra` first; on OpenAI it changes nothing.
- **F5. A listing costs about a second, and its failure is safe but its error text is not.** The measured listings took 0.27 to 1.9 s, bounded by the SDK at 10 s; a failure gives the static six with `Live` false. The provider's message can carry a masked fragment of the key, so it must not reach a log.
- **F6. Model metadata costs one 395 KB fetch per process, for the open catalogs only.** OpenCode Zen and Go, Hugging Face and Together rank with it; OpenAI and Kilo request none.
- **F7. A listing sends no generation.** `catalog.List` makes one `GET` of the provider's model list, plus the metadata fetch; probes belong to a provider's `ListModels`, and gobble's `New` turns them off. *(2026-10-19, 0023-PLAN deviation 1: from go-llmprovider-sdk `v1.14.0` the metadata fetch goes with every provider's listing but Kilo's and Ollama's, where it went with Hugging Face's, Together's and opencode's; a listing past the curated six is newest first; and a listing is served from an in-process cache for 10 minutes.)*
- **F8. The model choice is read many times per session, with no context, in every mode.** So a listing must be made once and kept, not made per read, and print mode lists too.
- **F9. The default model can come from the listing without changing the provider.** The session's chosen model is what each request names.
- **F10. One test would reach the network.** `TestDefaultProviderSetting` sets a credential and builds the model choice, so a real lister would list OpenCode Go with `test-only`.
- **F11. The records plan this, in pieces.** 0005-PLAN F4-2 owns the catalog, `models.json`, `gobble models list` and the model option's source; this record takes the option's source now, from the SDK, and leaves F4-2 the rest. `--offline` is X3's, and must turn the listing off when it lands.
- **F12. go-llmprovider-sdk `v1.12.0` was tagged after this record, and changes nothing gobble sends** *(2026-10-13, 0020-PLAN P1 deviation 3)*. `v1.12.0` (`57cbd79e0790`, 2026-10-13 14:32 UTC-5) adds request-side prompt caching (SDK `0039-MADR-request-side-prompt-caching.md`): a `Request.Cache` field, `CacheShort` or `CacheLong`, whose zero value sends every request as before, byte for byte (its D1, R45). gobble sets no such field. `check_libs.py --lib go-llmprovider-sdk --tag v1.12.0 --from v1.11.0`: every check ok; apidiff reports only `Request.Cache`, `CacheRetention`, `CacheShort` and `CacheLong` added; `llmprovider/catalog` is unchanged between the tags. `go.sum`: `h1:3nbuUj/BF/mRaXIEF2r1y+oSKB7KdZ6+T2arJZtmLv0=`, and the same `go.mod` sum.

## Decision Drivers

- **An agent harness offers every model a provider serves** (issue 1; the owner's request).
- **The library does the work** (0004-MADR): gobble builds no catalog of its own (0014-REPORT), and lists through the SDK's `catalog.List`.
- **No silent cost:** no generation to list; one listing per process; a failure is never worse than today's six.
- **No credential in a log** (the organisation's rule; F5).
- **No test reaches a provider** (0002-PLAN Phase 3).
- **The owner's choices of 2026-10-13:** the capable profile; model metadata on, once per process; `/model` prints every model; a new session starts on the listing's first model; print mode lists as every mode does; 0019-PLAN P5 before this record's work.

## Considered Options

- **A. `llm/provider.List` over `catalog.List`, kept once per process by `internal/cli`, started when the agent starts.**
- **B. A provider's `ListModels`,** on the provider `Ambient` builds.
- **C. Wait for 0005-PLAN F4-2,** and list there with `models.json` and `gobble models list`.
- **D. As A, with a disk cache** under gobble's cache directory and a time to live.
- **E. Pin `v1.11.0` with `catalog.WithLimit(catalog.MaxListed)`,** keeping the six.

## Decision Outcome

Chosen option: "A", because it offers the full catalog now, through the one package that may import the SDK, with no generation, one listing per process and the static six as its floor, and leaves F4-2 and X3 what they own.

### The decisions

- **D1. Require `github.com/maccavelli/go-llmprovider-sdk v1.11.0`,** with the sums measured above, in the commit that adds D2's listing, never `WithLimit`. *(2026-10-13, 0020-PLAN P1 deviation 3: `v1.12.0`, the latest tag, with F12's sums; gobble sets no `Request.Cache`, so its requests do not change. Setting it is a later record's.)* *(2026-10-19: `v1.15.0`, by [0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md).)*
- **D2. `llm/provider` gains `List(ctx, id, apiKey, o) (Listing, error)`.**
  - It calls `catalog.List` with `llmprovider.NewStaticToken(apiKey)`, `catalog.WithProfile(catalog.ProfileCapable)`, `Options`' client info and HTTP client, and model metadata on; no limit and no probe. It passes no logger, so the SDK logs nothing of a listing, and D6's line is the only one.
  - `Listing` is `{Models []string; Live bool; Err error}`: the listing's `Recommended` when it is live and not empty; otherwise the static models, `Live` false, and the reason in `Err`, wrapped as `New`'s errors are (`mapError`).
  - `Options` gains `ModelMetadataURL`, for tests, as `HTTPClient` is; `New` keeps model metadata and probes off.
  - `FailureKind(err)` names an error's class for a log line (`auth`, `rate-limited`, `quota`, `not-permitted`, `unavailable`, `timeout`, `canceled`, `empty`, `other`), never its text.
- **D3. `internal/cli` lists once per process, and starts when the agent starts.**
  - `agentOptions` starts the listing in the background for the provider `Choose` picks, when a credential is set; no credential, no listing.
  - The model choice waits for that first listing, then keeps it: a live listing for the process; a failed one gives the static models, and is retried in the background at most once a minute, so a long `gobble acp` recovers from a failure at start.
  - The lister is a package variable, `listModels`, so tests replace it.
  - *(2026-10-20, [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md) D4, 0025-PLAN P1: every credentialed provider is listed so, concurrently; the run's is waited
    for, the others are shown by their state, a listing still running by the one kept on disk.)*
- **D4. A new session starts on the listing's first model** when the listing is live, and on `StaticModels[0]` otherwise (the owner's choice). `Ambient` still builds the provider with `StaticModels[0]`; each request names the session's model (F9).
- **D5. The ACP `model` option, `set_config_option` and `/model` take the listed models, with no change in `acpserver`.** They already read `ModelChoice.Models`. `/model` with no argument prints every model, as today (the owner's choice). *(2026-10-20, [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md) D1, D5, 0025-PLAN P1: the option is grouped, recent first, then every usable provider, its values `provider/model`; `/model` prints a numbered list, and takes a number, a ref or an id one provider offers.)*
- **D6. A failed listing logs one warning per attempt,** "model listing failed; offering the curated static list", with the provider and `FailureKind`, never the error's text (F5).
- **D7. No test lists over the network.** `internal/cli`'s `TestMain` sets `listModels` to a lister that fails offline, and a test checks it; `llm/provider`'s tests list over canned transports.
- **D8. The records are corrected by dated notes:** 0019-MADR D2 (the pin moves to `v1.11.0` here); 0005-PLAN F4-2 (the model option's source is the SDK's listing from this record, and F4-2 adds `models.json`'s models and the metadata on top); 0005-PLAN X3 (`--offline` turns the listing off); 0014-REPORT's two lines; and `docs/architecture.md` and `README.md` name `v1.11.0`. The owner closes issue 1 when the work is committed.

### Consequences

- Good, because the picker, `set_config_option` and `/model` offer every known, working model of the chosen provider: 73 on OpenAI and 285 on Kilo, measured.
- Good, because a failed listing is today's behaviour: the static six, the static default.
- Good, because no generation is sent, and the listing is one request, plus one metadata fetch for the open catalogs.
- Neutral, because `acpserver`'s exported API and code do not change.
- Bad, because every process makes a listing when a credential is set: `-p` pays 0.3 to 2 s once, overlapped with its start, bounded at 10 s, and the open catalogs fetch 395 KB of metadata.
- Bad, because the default model depends on whether the listing succeeded: on Kilo a new session starts on `openai/gpt-6-astra` when it does, and `deepseek/deepseek-v4.1-flash` when it does not, and the strongest model is likely slower and dearer.
- Bad, because `/model` with no argument prints up to about 290 ids on one line.
- Bad, because the list is only as current as the process: a long `gobble acp` keeps its first live listing.
- Neutral, because the counts move with the providers; no test pins a live count.

### Confirmation

```text
go list -m github.com/maccavelli/go-llmprovider-sdk                       -> v1.11.0   (P1 deviation 3: v1.12.0)
git diff -U0 HEAD~ -- go.mod | grep -c '^[-+]\s'                           -> 2 (P1's commit)
grep -rn 'WithLimit' --include=*.go .                                      -> no match
go test ./llm/provider -run 'TestList'                                     -> ok: the full canned list, the capable profile, metadata on, the static fallback, FailureKind
go test ./internal/cli -run 'TestModelChoice|TestModelOption|TestListerIsOffline' -> ok
make archtest                                                              -> ok: the SDK only under llm/provider
the gates                                                                  -> lint ×3; go test on Windows and the macOS laptop; -race on Windows and the second Linux machine; Windows preflight
```

Each new test is seen failing on a scratch copy first (0020-PLAN's negatives), among them the source being today's `provider.Models`.

## Pros and Cons of the Options

### A — `List` over `catalog.List`, once per process (chosen)

- Good, because `catalog.List` is the SDK's listing, made for this (SDK 0038-MADR D7), and sends no generation.
- Good, because the listing is independent of building the provider, so `New`'s "no network call" holds.
- Good, because it is a small change: one function in `llm/provider`, one type in `internal/cli`, no `acpserver` change.
- Bad, because each process lists once, print mode included.

### B — a provider's `ListModels`

- Good, because it is the method the SDK's contract names, and it carries the provider's own options.
- Bad, because `New` turns model metadata off for generation, so the open catalogs would rank without it, or `New` would turn metadata on for every request.
- Bad, because the provider is built on its first use, the first prompt (`acpserver/agent.go:663-664`), after `session/new` has already published the option.

### C — wait for F4-2

- Good, because F4-2 brings `models.json`, the metadata view and `gobble models list` with it, in one design.
- Bad, because F4-2 is large and waits behind F4-1a and 0012-PLAN P7; the issue and the owner ask for the listing now.

### D — a disk cache with a time to live

- Good, because `-p` would rarely list, and every mode would start on the same list at once.
- Bad, because it adds a cache file, its invalidation and its tests, and a listing can differ per credential. The owner chose to list in every mode; a persistent catalog belongs with F4-2.

### E — `v1.11.0` with `WithLimit(MaxListed)`

- Good, because nothing changes for users.
- Bad, because it leaves issue 1 open and the picker at six. The strongest argument for it, no listing cost, is better met by A's once-per-process listing and static floor.

## More Information

### Evidence index

| Claim | Source |
| :--- | :--- |
| The three libraries' latest tags | `git ls-remote --tags`, 2026-10-13 |
| `v1.11.0` passes every check; apidiff's two additions; the sums | `check_libs.py --lib go-llmprovider-sdk --keep`, logs `gobble-check-libs-16fc4cr2` |
| The live counts, timings and heads | `q492_i1_probe.py` / `i1probe` (OpenAI with the test key, Kilo and OpenCode Zen with a placeholder) |
| Kilo's counts move | `i1kilo`, 273 and 286 |
| The masked key fragment in a 401's text | `i1probe`, the rejected-key run |
| The canned listings, requests, fallbacks and profiles | `i1canned`, `i1kilo/canned`, `i1kilo/zenurls` |
| The metadata document's size | `curl -s -o /dev/null -w ... --compressed https://models.opencode.ai/api.json` |
| The SDK's default change, static lists and probes | go-llmprovider-sdk `v1.11.0`, `docs/decisions/0038-MADR-full-curated-catalog-by-default.md` D1–D4, D7; `llmprovider/catalog/discovery.go:60-77`, `:79-108`; `config.go:82-99` |
| gobble's model choice and its readers | `llm/provider/ambient.go:62-67`, `:117-126`; `internal/cli/agent.go:39-46`, `:80-88`, `:117-121`; `acpserver/config.go:86-92`, `:113-119`; `acpserver/commands.go:283-303`; `acpserver/agent.go:59`, `:297`, `:323`, `:339`, `:553`, `:741`, `:773`, `:809` |
| Requests carry the session's model | `acpserver/agent.go:428`, `:461`; `agent/agent.go:351` |
| `New` makes no network call | `llm/provider/provider.go:38-46` |
| The test that would reach the network | `internal/cli/settings_test.go:255-266`; `internal/cli/testmain_test.go:16-25` |
| The SDK import rule | `internal/archtest/check.go:190`, `:249` |
| `--offline` is X3's | `internal/cli/flags.go:52`, `:69` |
| The issue | gobble-cli issue 1, opened 2026-10-13 |

### Related records

- [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md): D2 pinned `v1.10.2`; D11 re-bases F4-2 on the SDK's catalog; its PLAN's deviation 13 hands `v1.11.0` to this record.
- [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F4-2 (the catalog and the model option), X3 (`--offline`).
- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): the library does the work; the package map.
- [0014-REPORT-llmprovider-sdk-integration.md](../reports/0014-REPORT-llmprovider-sdk-integration.md): list through the SDK; no gobble-owned catalog.
- go-llmprovider-sdk, `docs/decisions/0038-MADR-full-curated-catalog-by-default.md`.

### Open questions for the plan

None. The owner answered the profile, the metadata, `/model`'s output, the default model, print mode and the order on 2026-10-13.
