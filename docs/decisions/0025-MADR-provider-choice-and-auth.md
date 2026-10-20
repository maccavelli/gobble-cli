---
status: proposed
date: 2026-10-20
decision-makers: repository owner
consulted: go-llmprovider-sdk v1.17.0 (llmprovider/providers, catalog, wizard, auth/oauth_plan.go, its 0043-MADR); acp-go-sdk2 v1.16.3 (SessionConfigSelectOptions); Pi 1402973a9, opencode 501b15af26, Kilo Code 2a1903f291 and goose 9920efa1 (0025-REPORT)
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

**What the other harnesses do,** and what the SDK already provides, is
[0025-REPORT-provider-ux-across-harnesses.md](../reports/0025-REPORT-provider-ux-across-harnesses.md): Pi, opencode,
Kilo Code and goose each switch model mid-session; Pi's replay rules are keyed by model; opencode and Kilo remember
recent models; goose remembers a model per provider; the SDK's `wizard.ConfigureLLM` runs a sign-in over a line-mode
`Prompter`, and its `catalog` already layers live, stale and static listings. Its replay filter is by wire format only
(`llmprovider/internal/wire/wire.go:42-44`), so two providers on one wire (OpenAI and Grok; Claude, Kilo and OpenCode)
would pass each other's opaque reasoning.

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

*(Revised 2026-10-20, before review, from [0025-REPORT-provider-ux-across-harnesses.md](../reports/0025-REPORT-provider-ux-across-harnesses.md),
at the owner's request to "evaluate and analyze against the proposed madr and plan … adjust the madr/plan as needed".
What changed from the first draft: the replay is keyed by model, not provider, and takes Pi's rules whole (D2);
`last-model.json` becomes a list of recent models, with each provider's last model (D3); dead keys are marked lazily
too, and a public listing is "not checked" (D4); the in-line picker and the provider states are specified (D5);
`/login` runs the SDK's wizard, a refused switch leads to it, and `auth login` takes `--method` (D6); first-run setup
(D7) and cycling a scoped set (D8) are new, each the owner's to take (Q3, Q5); a strip-and-retry safety net is offered
(Q4). D1's owner answers stand.)* *(2026-10-20: Q3, Q4 and Q5 answered by the owner: first-run setup offered;
one strip-and-retry; cycling as P7. Every owner question is answered.)*

- **D1. Any model of any credentialed provider, at start and mid-session** (the owner's answer: "Any model, any
  provider").
  - **One name for a model everywhere,** `provider/model[:thinking]`, resolved by the rules `--model` already uses
    (Pi's `resolveCliModel`, 0005-PLAN F4-2c): a bare id when exactly one offered provider has it; a fuzzy match that
    prefers an alias to a dated id; an ambiguity answered with the candidates. `/model`, the ACP option, `--model`,
    `defaultModel` and `enabledModels` (D8) take the same grammar.
  - `/model <ref>` switches; `/model <provider>` switches to that provider's last model (D3), else its first
    recommended one. The ACP `model` option is grouped (D5), values `provider/model`.
  - A switch builds the provider for the session and writes `model_change` with it; the conversation continues. When
    the new model's context window is smaller than the context in use, the switch says so and suggests `/compact`
    (goose). A switch to a provider marked dead (D4) is refused with its reason; to one with no credential, the line
    session offers `/login <provider>` (D6) rather than sending the user away to configure it.
  - A session restored from its log takes its last `model_change`'s model when that provider has a credential and is
    not marked dead; else the start choice (D3), with a note.
  - It amends 0021-MADR D4, 0020-MADR D3–D5, 0019-MADR D15 (the setting becomes a start preference) and 0005-PLAN
    F4-2c decision 4, each with a dated note.
- **D2. Replay on another model: Pi's rules** (the owner's Q1 answer, "drop it", made precise).
  - **Same model** is provider, wire format and model id all equal, as Pi's `isSameModel`
    (`transform-messages.ts:122-125`). A message from the same model is replayed as it is, signatures and encrypted
    reasoning included.
  - **From another model,** a switch within one provider included: its thinking is dropped with its signatures and
    encrypted reasoning; its text is sent bare, without `ProviderData`; its tool calls lose `ProviderData` and
    Gemini's thought signature (the SDK sends its skip marker for a call with none); call ids a wire refuses are left
    to the SDK's `RemapCallIDs`.
  - An errored or aborted assistant message is not replayed; a tool call left with no result gets a synthetic error
    result, "No result provided", before the next message (Pi, `transform-messages.ts:179-253`).
  - The log is never rewritten; the rules apply where the history becomes a request.
- **D3. The start, and what gobble remembers** (the owner's answer: "Your last choice").
  - `<state>/models.json` keeps `recent`, up to ten `provider/model` refs, newest first, the shape opencode and Kilo
    keep; a choice by `/model`, the ACP option or `--model` puts its ref first. A provider's last model is its newest
    entry there.
  - The start order: `--model` / `--provider`; then the `defaultProvider` and `defaultModel` settings; then `recent`'s
    newest whose provider has a credential and is not marked dead; then the registry order, with a note naming the
    provider and how to switch. `--model` and `--provider` still fail rather than fall (0019-MADR D15's reason); each
    other step falls to the next with a note.
  - A switch is the session's; saving a default is explicit: `/model <ref> --default` writes `defaultModel`, and its
    provider as `defaultProvider`, to the global settings, as Pi's Ctrl+S does.
- **D4. Each provider's state, by listing, with its reason** (the owner's answer: "Check by listing, mark").
  - At start every credentialed provider is listed once, concurrently, in the background, through the SDK's
    `catalog.List`, which never spends a generation; its last good listing is kept on disk (`WithLastGood`, in
    `<cache>/catalog/<provider>.json`), so the picker shows it at once and the live one replaces it.
  - The states, from `Catalog.Source` and `Catalog.Err`: **live** (with its model count); **stale** (a cached listing,
    with its age); **key rejected**, **quota exhausted**, **not permitted** (with the provider's reason; its models are
    not offered for a switch until `/login` or a later listing succeeds); **not listed** (offline or unavailable: its
    static models offered); **not checked** (OpenCode Zen and Go, Hugging Face and Kilo, whose listing needs no key and
    so cannot reject one); **no credential** (with its variable and `/login`).
  - A request that fails with `ErrAuth` or `ErrQuota` marks its provider then too, as opencode resolves lazily. No key
    is ever checked by a billed request.
- **D5. Browse providers and models, in the CLI and the session.**
  - **The picker, line mode:** `/model` with no argument prints a numbered list: the current model; the recent ones
    (five); then each provider under a heading with its state (D4), its models with the SDK menu's facts (context
    window, input and output price, thinking) and caveats (deprecated, no tools). The prompt reads a number to switch,
    or words to filter (`catalog.Search` per provider), or nothing to leave. `/model <words>` that is not a ref opens it
    filtered. The layout is 0024-MADR D2's.
  - **ACP:** the `model` option grouped, "Recent" first, then one group per provider with a usable state.
  - **Providers:** `/providers` and `gobble providers [--check] [--json]`: each of the ten, its state, its credential's
    source in Pi's words ("env: OPENAI_API_KEY", "auth.json: sign-in", "none — /login grok"), its sign-in methods.
  - **Catalogs:** `/models [provider] [words]` and `gobble models list [--provider ID] [--all]`; `--all` and a provider
    with no credential show the static catalog with the models.dev facts, marked so.
- **D6. Sign in where the user is** (the owner's answer: "/login + ChatGPT now").
  - `/login [provider]` in the line session runs the SDK's `wizard.ConfigureLLM` with gobble's own `Prompter` over the
    line editor (out of raw mode, 0024-MADR D1; `Secret` with no echo) and gobble's `auth.json` as its `TokenStore`
    (Pi's shape, through the existing `LoginStore` adapter). The provider list shows each one's state in Pi's words; the
    wizard checks a typed key by listing and asks again when it is refused. After it, the credential cache is
    refreshed, the provider is listed (D4), and the picker opens filtered to it (opencode, Pi).
  - `/logout [provider]` runs `wizard.Logout`: it lists stored credentials only, revokes an OpenAI or Grok session,
    deletes it, and says that an environment variable still applies (Pi's words).
  - `gobble auth login <provider> --method api-key|browser|device|chatgpt-plan` signs in with no menu (opencode's
    `--method`), for scripts and for a terminal with no menus.
  - ACP clients keep the `terminal` auth methods of 0005-PLAN F4-3c-2. This amends 0005-MADR's "`/login` and `/logout`
    are TUI-only".
  - **ChatGPT's plan sign-in, 0005-PLAN F4-3d, now:** `/login openai` and `--method chatgpt-plan` through
    `auth.LoginChatGPTPlan` with `AgentName` "gobble" and a stable `HostID` in the state directory; stored as an OAuth
    login; the openai provider sends its requests under the plan. Only the plan method is offered for OpenAI's account
    sign-in: the SDK's Codex-client-id methods (its browser and device OAuth, the Codex CLI import) are not, by
    0005-MADR's rule. It is browser-only; with no browser, the URL is printed and the code pasted back (`OpenURL`,
    `InputCode`).
  - **The terms, kept by gobble:** under a plan session, background calls (the session title, automatic compaction)
    wait for the user's consent, asked once at the first plan session and saved (the owner's Q2 answer); without it
    they are skipped (no title; compaction only on `/compact`). Requests go from the user's own process; nothing pools
    or shares the session.
  - Grok's sign-in is unchanged (0005-MADR, the owner's accepted risk) and joins `/login`. The Pi import still skips
    OpenAI OAuth, which Pi holds under Codex's client id.
- **D7. First-run setup** (owner question Q3). When no provider has a credential, the line session offers the
  wizard, `/login`'s, before its first prompt, instead of only "set one of …"; `gobble setup` runs it at any time.
  Declined, the session goes on as today.
- **D8. Cycling a few models** (owner question Q5). An `enabledModels` setting (Pi's globs, such as `anthropic/*` or
  `*sonnet*:high`) scopes a set; Ctrl+P in the line editor switches to the next model of the set, or of the recent
  ones when no set is given, each a D1 switch, and prints the model it switched to. Ctrl+P is free: `golang.org/x/term`
  binds only Ctrl+C, Ctrl+D, Ctrl+U and the arrow keys (`terminal.go:148-161`). Pi's Shift+Ctrl+P for the previous
  model sends the same byte as Ctrl+P on a byte-stream terminal, so "previous" takes another key, Alt+P where the
  terminal sends it as `ESC p`, measured on the three systems in the phase that builds it.

### Consequences

* Good, because every provider the user has a key or a sign-in for is usable, at start and mid-session, in the CLI
  and in ACP clients.
* Good, because a dead key is shown as such before the user picks it, not after a failed turn.
* Good, because a ChatGPT or Grok subscription can be used without an API key.
* Good, because the picker, the states and the sign-in reuse the SDK's catalog, search and wizard, and the replay
  rules are the ones gobble's Pi session format already expects.
* Good, because a missing credential leads to signing in, in the session, not to leaving it.
* Neutral, because a session start makes one listing request per credentialed provider, in the background, where it
  made one.
* Bad, because a conversation switched to another model loses the earlier model's thinking and its replay data;
  the new model sees the text and the tool calls only.
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

* **Q3. First-run setup (D7).** Offered: take it (recommended: a first run with no key leads into signing in); or
  leave the first run as today, with `/login` and `gobble setup` alone. **Answer (2026-10-20): offer setup.**
* **Q4. A strip-and-retry safety net.** When a provider rejects a request over a thinking signature despite D2, retry
  it once with every thinking block stripped (goose). Offered: take it (recommended: D2 should make it rare, and a
  failed turn is worse); or leave D2 alone, the error shown. **Answer (2026-10-20): retry once.**
* **Q5. Cycling a scoped set (D8).** Offered: take it, `enabledModels` and Ctrl+P (recommended for a terminal user who
  moves between a few models); or leave it for later, `/model` alone. **Answer (2026-10-20): yes, as P7.**

## More Information

- [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md) D15: `defaultProvider`.
- [0020-MADR-full-model-catalog-listing.md](0020-MADR-full-model-catalog-listing.md) D3–D5: the listing and `/model`.
- [0021-MADR-prompt-caching-per-session.md](0021-MADR-prompt-caching-per-session.md) D4: one provider per ACP session.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) and [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F4-2c, F4-3b (Grok), F4-3d (ChatGPT), `/login`.
- [0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md) D6: the plan sign-in left to F4-3d.
- The survey: [0025-REPORT-provider-ux-across-harnesses.md](../reports/0025-REPORT-provider-ux-across-harnesses.md).
- The plan: [0025-PLAN-provider-choice-and-auth.md](0025-PLAN-provider-choice-and-auth.md).
