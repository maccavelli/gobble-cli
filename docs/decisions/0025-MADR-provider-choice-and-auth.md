---
status: proposed
date: 2026-10-20
decision-makers: repository owner
consulted: go-llmprovider-sdk v1.17.0 (llmprovider/providers, catalog, auth/oauth_plan.go, its 0043-MADR); acp-go-sdk2 v1.16.3 (SessionConfigSelectOptions)
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Any provider, any model: choose and switch providers, browse their catalogs, and sign in from the session

## Context and Problem Statement

The owner reported on 2026-10-20, running the `v0.1.1` release: "it reads all of the available API-KEY/TOKEN env vars
and they appear, but i can not change providers. it is stuck on google models … i need to be able to both configure
provider authentication, and also when running gobble change providers and select from their models. ideally, if
there are existing keys/tokens in my env vars, i should be able to use any model provided by any provider that has an
active, funded key or token. also, i need to be able to configure auth regardless of the env vars, to setup oauth for
openai/chatgpt and grok to use their subscriptions instead of api-keys. i can not browse providers, i can not browse
catalogs per provider, i can not configure auth for providers. … it is a core, basic, required, necessary and
essential feature set". Each fact below was measured that day at `d76e3d2`.

**One provider per process, chosen by registry order.**

- `ChooseWith(has, preferred)` (`llm/provider/ambient.go:38-55`) takes `preferred` when it has a credential, else
  fails naming its variable; with none preferred it takes the first provider, in the SDK's registry order, that needs
  a key and has a credential. The order is gemini, openai, claude, grok, opencode-zen, opencode-go, huggingface, kilo,
  together, then ollama (`llmprovider/providers/providers.go`). With `GEMINI_API_KEY` set, Gemini wins whatever else is
  set: what the owner saw.
- `preferred` is `--provider` / `--model`'s provider, else the `defaultProvider` setting (`internal/cli/modelflags.go:53`),
  a process key read once at start (0019-MADR D15).
- The credential resolver and `auth.json` are read once a process (`internal/cli/credentials.go:31-52`); each session
  builds its provider once and keeps it (`acpserver/agent.go:862-879`).

**One provider's models in the session.**

- `modelChoice()` (`internal/cli/agent.go:162-182`) is `ModelChoice{Provider, Models, Default}`: one provider's live
  listing or static list (`acpserver/config.go:155-162`).
- `/model` accepts only an exact entry of that list (`acpserver/commands.go:343-364`); `provider/model` is not
  parsed. The ACP `model` option is the same list (`config.go:168-177`, `:196-234`). No command, option or `_gobble/`
  method switches provider (`commands.go:115`, `config.go:23-26`, `extensions.go:29-50`).
- The records say so: "One provider, and one session id, per ACP session" (0021-MADR D4); the listing is "for the
  provider `Choose` picks" and `/model` takes "the listed models" (0020-MADR D3, D5); "The chosen provider is the
  run's" (0005-PLAN F4-2c decision 4).

**Browsing.** `gobble models list` (`internal/cli/modelscmd.go:58-99`) is one flat table of the providers with a
credential, with no provider filter and nothing for a provider without one. There is no list of providers. In a
session there is neither.

**Auth.**

- `gobble auth login <provider>` stores an API key in `auth.json` (`--api-key -`), or signs in to Grok (browser or
  `--device`) or Kilo (device, returning a key) (`internal/cli/auth.go:166-201`; `llm/provider/login.go:71-126`).
  `auth status` lists each provider's credential source; `auth check` checks presence only, with no network call
  (`auth.go:423-446`).
- OpenAI is refused: "openai's login waits for go-llmprovider-sdk's Sign in with ChatGPT for agents (0005-PLAN F4-3d);
  use --api-key -" (`auth.go:197-198`). The SDK has had it since `v1.15.0`: `auth.LoginChatGPTPlan(ctx,
  ChatGPTPlanOptions{AgentName, HostID, Previous, HTTPClient, OpenURL, InputCode})` (`llmprovider/auth/oauth_plan.go:49-94`),
  a browser PKCE flow on a loopback redirect, with no device flow; its session is a plan session, billed to the plan.
  0023-MADR D6 left it to "0005-PLAN F4-3d's … made executable on the owner's word".
- `oauth_plan.go:90-93`: "What OpenAI's terms ask of the app itself, the SDK cannot do for it: requests from the
  user's own runtime, consent before any background use, the plan used only for the app the user connected, no
  pooling or sharing, and no charge." gobble's background calls are the session title (`acpserver/title.go`) and
  automatic compaction (0005-PLAN F5-2).
- There is no `/login` in a session; 0005-MADR made `/login` and `/logout` TUI-only (`0005-MADR-v1-feature-scope.md:323`),
  and the TUI is not built.

**Keys that may be dead.** The SDK has no balance or validity call. `catalog.List(ctx, id, tokenSrc, …)` lists one
provider with one request and reports a failure in `Catalog.Err`, which gobble maps to `llm.ErrAuth`, `ErrQuota`,
`ErrNotPermitted` and the rest (`llm/provider/provider.go:460-469`). Whether a key with no funds fails a listing is
not measured.

**Switching provider mid-conversation.** A session's history carries provider-specific replay data: thinking
signatures and encrypted reasoning in `ProviderData` (`llm/llm.go:106-113`; `llm/provider/provider.go:284-321`,
`:384-393`), and in the log `thinkingSignature` and `thoughtSignature` (`session/session.go:186-191`). Another provider
refuses or mis-reads them. Each assistant message records its `provider` and `model` (`session/session.go:147`;
`acpserver/history.go:185`, `:201`), so a replay can tell whose data is whose.

**ACP.** A select option may group its values (`acp.NewSessionConfigSelectOptionsGrouped`, acp-go-sdk2 `v1.16.3`
`helpers_gen.go:346-349`), so a client can show models under their providers.

## Decision Drivers

* The owner's requirement: every provider with a working credential usable, any of its models, chosen at start and
  switched mid-session; providers and their catalogs browsable; auth configurable apart from the environment,
  ChatGPT and Grok subscriptions included.
* The owner's answers of 2026-10-20 (below, D1–D5): switch inside the conversation; start on the last choice; check
  keys by listing and mark the dead; `/login` in the session and ChatGPT sign-in now.
* Never send work to a provider the user did not choose (0019-MADR D15's reason), and never send one provider's
  replay data to another.
* OpenAI's terms for a plan sign-in, which gobble itself must keep.
* One engine for the line session and ACP clients (0008-MADR D13), so an editor and the phone app get the same choices.

## Considered Options

* **A. Every credentialed provider in the session,** switchable mid-conversation, with browsing and in-session sign-in.
* **B. A new session per provider:** picking another provider's model starts a fresh session on it.
* **C. Start-time choice only:** a picker or prompt at start, the session fixed to it as today.

## Decision Outcome

Chosen option: "A. Every credentialed provider in the session", because it is what the owner asked for and chose on
2026-10-20, and the session log already records each message's provider, which a switch needs. *Proposed: the owner
decides at review.* *(2026-10-20: Q1 and Q2 answered by the owner: another provider's thinking is dropped; consent
under a plan is asked once and saved.)*

- **D1. A session's models are every credentialed provider's** (the owner's answer: "Any model, any provider").
  - The model choice becomes a list of providers, each with its models and its state. A model is named
    `provider/model` wherever it can be ambiguous; a bare id is accepted when exactly one provider has it.
  - `/model` with no argument shows the current model and every provider's models, grouped; `/model provider/model` or
    a unique bare id switches. The ACP `model` option is grouped by provider, its values `provider/model`.
  - Switching to another provider's model builds that provider for the session and writes `model_change` with the new
    provider; the conversation continues. The provider of a session restored from its log is its last
    `model_change`'s, when that provider has a credential; else the start choice's (D2), with a note.
  - **Replay across providers:** a message produced by another provider is sent without its provider-specific data:
    thinking signatures, encrypted reasoning and `ProviderData` are dropped, and that provider's thinking text is
    dropped too, as visible text it was never meant to be (owner question Q1). Text, tool calls and tool results are
    kept. The log is not rewritten.
  - It amends 0021-MADR D4, 0020-MADR D3–D5, 0019-MADR D15 (the setting is the start's preference, not the session's
    only provider) and 0005-PLAN F4-2c decision 4, each with a dated note.
- **D2. A new session starts on the last choice** (the owner's answer: "Your last choice").
  - The order: `--model` / `--provider`; then the `defaultProvider` and `defaultModel` settings; then the last model
    the user chose, kept in `<state>/last-model.json` as `provider/model` and written when `/model`, the ACP option or
    `--model` chooses; then, with none, today's registry order, with a note naming the provider and how to switch
    (`/model`, `--model`, `defaultProvider`).
  - A start choice whose provider has no credential, or whose key is marked dead (D3), falls to the next in that order
    with a note; `--model` and `--provider` still fail rather than fall (0019-MADR D15's reason).
- **D3. Keys are checked by listing, and a dead one is marked** (the owner's answer: "Check by listing, mark").
  - At start, every provider with a credential is listed once, concurrently, in the background (one request each, the
    SDK's `catalog.List`, as 0020-MADR D3 lists one today). A provider whose listing fails with `ErrAuth`, `ErrQuota`
    or `ErrNotPermitted` is marked with that state and its reason, and its models are not offered for a switch until a
    later `/login` or listing succeeds; any other failure (offline, unavailable) keeps its static models, marked
    "not listed".
  - The session does not wait for the listings: a switch to a provider not yet listed waits for its listing, at most
    the listing's own deadline.
  - "Funded" is what a listing shows and no more; a key that lists but cannot generate shows at its first request.
- **D4. Browse providers and catalogs, in the CLI and in the session.**
  - `gobble providers` lists every provider: id, name, auth methods (API key, sign-in), the credential's source
    (`auth status`'s), and with `--check` D3's state. `gobble models list --provider ID` lists one provider's catalog,
    its static models when it has no credential, marked so.
  - In a session: `/providers` (the same list, with D3's states) and `/models [provider] [search]` (one provider's
    models, or every provider's, with a search). Both are agent commands, so ACP clients have them too.
- **D5. Sign in from the session, and ChatGPT's plan sign-in** (the owner's answer: "/login + ChatGPT now").
  - `/login [provider]` and `/logout <provider>` in the line session: with no provider, a numbered list to choose
    from; then that provider's methods: an API key typed with no echo, Grok's browser or device sign-in, Kilo's device
    sign-in, ChatGPT's plan sign-in. The credential goes to `auth.json` as `gobble auth login` stores it, the
    credential cache is refreshed, the provider is listed (D3) and joins the session's models. They are line-session
    commands, since they need the terminal; ACP clients keep the `terminal` auth methods of 0005-PLAN F4-3c-2. This
    amends 0005-MADR's "`/login` and `/logout` are TUI-only".
  - **ChatGPT's plan sign-in, 0005-PLAN F4-3d, now:** `gobble auth login openai` (and `/login openai`) offers it beside
    the API key, through `auth.LoginChatGPTPlan` with `AgentName` "gobble" and a stable `HostID` kept in the state
    directory; the session is stored in `auth.json` as an OAuth login, and the openai provider sends its requests
    under the plan. It is browser-only; on a machine without one, the sign-in URL is printed for another device
    (`OpenURL`), and the code is pasted back (`InputCode`).
  - **The terms, kept by gobble:** under a plan session, background calls — the session title and automatic
    compaction — are made only after the user consents once, asked at the first plan session and saved; without
    consent they are skipped (no title; compaction only on `/compact`) (owner question Q2). Requests go from the
    user's own process only; nothing pools or shares the session.
  - Grok's sign-in is unchanged (0005-MADR, the owner's accepted risk) and joins `/login`. The Pi import still skips
    OpenAI OAuth, which Pi holds under Codex's client id.

### Consequences

* Good, because every provider the user has a key or a sign-in for is usable, at start and mid-session, in the CLI
  and in ACP clients.
* Good, because a dead key is shown as such before the user picks it, not after a failed turn.
* Good, because a ChatGPT or Grok subscription can be used without an API key.
* Neutral, because a session start makes one listing request per credentialed provider, in the background, where it
  made one.
* Bad, because a conversation switched to another provider loses the earlier provider's thinking and its replay
  data; the new provider sees the text and the tool calls only.
* Bad, because four accepted records change by amendment, and the session engine's one-provider assumption is
  removed across `acpserver`, `internal/cli` and `llm/provider`.
* Bad, because the plan sign-in brings duties gobble must keep (consent before background use), and the flow has no
  device mode.

### Confirmation

The PLAN's tests for each phase, each seen failing first; a live run per phase on the owner's keys where a provider is
involved (listing, switching, sign-in), recorded without any credential; the gates on macOS, the Linux machine and
the Windows laptop.

## Pros and Cons of the Options

### A. Every credentialed provider in the session

* Good, because it meets every part of the requirement in one model.
* Bad, because it is the largest change, and a cross-provider replay must drop data correctly.

### B. A new session per provider

* Good, because a session keeps one provider and its replay data stays whole.
* Bad, because switching loses the conversation, which the owner did not choose.

### C. Start-time choice only

* Good, because it is small.
* Bad, because it does not let the user switch or browse in the session, which the owner asked for.

## Owner questions

* **Q1. Another provider's thinking on a switch.** Offered: drop it with its signatures (recommended: thinking was
  never visible text, and a provider may refuse another's); or keep its text as plain text in the replay.
  **Answer (2026-10-20): drop it.**
* **Q2. Background calls under a ChatGPT plan.** Offered: ask once at the first plan session and save the answer
  (recommended); or never make background calls under a plan (no titles, no automatic compaction); or make them
  without asking (against the terms the SDK quotes, so not recommended). **Answer (2026-10-20): ask once, and save
  the answer.**

## More Information

- [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md) D15: `defaultProvider`.
- [0020-MADR-full-model-catalog-listing.md](0020-MADR-full-model-catalog-listing.md) D3–D5: the listing and `/model`.
- [0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md) D4: one provider per ACP session.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) and [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F4-2c, F4-3b (Grok), F4-3d (ChatGPT), `/login`.
- [0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md) D6: the plan sign-in left to F4-3d.
- The plan: [0025-PLAN-provider-choice-and-auth.md](0025-PLAN-provider-choice-and-auth.md).
