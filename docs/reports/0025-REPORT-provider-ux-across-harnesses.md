---
status: observation
date: 2026-10-20
subject: how Pi, opencode, Kilo Code and goose handle several providers, their model catalogs, model choice, switching mid-session, and sign-in; what go-llmprovider-sdk v1.17.0 already provides; and what 0025-MADR should take from them
examines: "Pi 1402973a9; opencode 501b15af26; Kilo Code 2a1903f291; goose 9920efa16d078777a4d9715b2f00d8800a4da188 (a shallow clone, read and deleted); go-llmprovider-sdk v1.17.0; gobble-cli d76e3d2 with 0025-MADR proposed"
---

# REPORT 0025 — Several providers, their catalogs, model choice and sign-in, across four harnesses and the SDK

<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

This report records findings and recommendations. It decides nothing. The owner asked on 2026-10-20 to "assess and
evaluate how kilo/opencode/pi/goose manage multi provider / catalogs / models. get ideas. find ergonomic and intuitive
solutions. evaluate and analyze against the proposed madr and plan. find the most idiomatic, llmprovider sdk
compatible, feature-rich, and cli optimized solutions." The decisions it informs are in
[0025-MADR-provider-choice-and-auth.md](../decisions/0025-MADR-provider-choice-and-auth.md)'s revision of 2026-10-20.

Each harness was read from source, by a read-only survey of its tree; paths below are relative to each repository.
Lines marked **(checked)** were read again before this report was written; the rest are the surveys' readings.

## 1. Model choice and the picker

| | Pi | opencode | Kilo Code | goose |
| :--- | :--- | :--- | :--- | :--- |
| Name of a model | `provider/id[:thinking]`, fuzzy, ambiguity lists candidates (`coding-agent/src/core/model-resolver.ts:230-307`) | `provider/model#variant` everywhere (`packages/schema/src/model.ts:27-34`) | `provider/model`, three fuzzy suggestions for an unknown id (`provider.ts:1426-1452`) | `/model <name>`, `/model --provider p [m]` (`crates/goose-cli/src/session/input.rs:295-321`) |
| Picker | fuzzy; current, default, then by provider; `[provider]` on each row; Tab scopes (`components/model-selector.ts:231-335`) | Favorites, Recent, then per provider; fuzzy; connect from the picker (`packages/tui/src/component/dialog-model.tsx:60-175`) | Favorites, Recent (10), Recommended, per provider; info panel at 108 columns (`packages/tui/src/kilocode/model-picker.ts:101-186`) | none in session; `goose configure` numbered list, "Search all models…" past 10 (`configure.rs:605-677`) |
| Remembered | `model_change` in the session; settings default only on Ctrl+S (`agent-session.ts:2507-2540`) | `state/model.json` `{recent[10], favorite[], variant}` (`packages/tui/src/model-preference.ts:45-60`) **(checked)** | the same, per agent too (`local.tsx:48`, `:199-205`) | per provider: `providers.<name>.model` (`crates/goose/src/config/providers.rs:8-97`) |
| Start order | `--model`; the session's last `model_change` if its provider has auth; settings default; a per-provider default table; first available (`core/sdk.ts:208-275`, `model-resolver.ts:537-610`) | per-agent choice; config; `--model`; first valid recent; first available (`local.tsx:184-218`) | the same, plus Kilo's server default | flag; the session's; recipe; config (`builder.rs:289-380`) |
| Cycling | Ctrl+P / Shift+Ctrl+P through scoped models (`enabledModels` globs), Shift+Tab thinking (`core/keybindings.ts:41-47`) | F2 recent, Ctrl+T variant | hidden keys for recent, favourites, variant | none |
| Switch, mid-session | any time | any time | any time | `/model`, warns when the context window shrinks; refused when credentials are missing, telling the user to leave (`session/mod.rs:883-1056`) |

**Findings.** A switch applies to the session; a saved default is a separate, explicit act (Pi, goose). Recent models
are the common memory (opencode, Kilo; gobble's owner chose "your last choice", which is recent's first). Remembering a
model per provider (goose) makes "switch to provider X" land on X's last model. Picking a provider's model where its
credential is missing should lead to signing in, not to leaving the session (goose's gap; opencode and Pi connect from
the picker).

## 2. Replaying a conversation on another model

- **Pi** (`packages/ai/src/api/transform-messages.ts:91-256`) **(checked, `:120-146`)**: "same model" is provider, API
  and model id all equal. Thinking from another model is dropped, signatures and encrypted reasoning with it ("sending
  readable thinking as plain text makes the new model treat it as part of the visible reply"); another model's text is
  rebuilt bare; a foreign tool call loses Gemini's `thoughtSignature` and its id is normalized per target API, its
  results renamed with it; errored and aborted assistant messages are skipped; an unanswered tool call gets a
  synthetic result "No result provided".
- **opencode** (`packages/core/src/session/runner/to-llm-message.ts:153-210`) and **Kilo** (`session/message-v2.ts:376-519`):
  provider metadata only for the same provider and model; otherwise reasoning is sent as plain text.
- **goose** (`crates/goose-provider-types/src/formats/anthropic.rs:146-159`; `crates/goose-providers/src/anthropic.rs:221-240`):
  drops another model's thinking for Anthropic; when the API rejects a thinking signature, it retries once with every
  thinking block stripped.
- **The SDK** filters reasoning by wire format only: `Replays(item, format)` is `item.Format == "" || item.Format ==
  format` (`llmprovider/internal/wire/wire.go:42-44`) **(checked)**. OpenAI and Grok share the Responses wire, Claude,
  Kilo and OpenCode the Messages wire, so their opaque reasoning would cross between them. It does rewrite call ids a
  wire refuses (`RemapCallIDs`, `llmprovider/internal/wire/callids.go`, 0026-MADR F8) **(checked)**, and fills Gemini's
  skip-signature marker for a call with none (`generatecontent.go:135-143`).
- **gobble** today replays every `ProviderData` it holds (`llm/provider/provider.go:384-393`) and has none of Pi's
  rules: no same-model test, no skip of an errored turn, no synthetic result **(checked)**.

**Finding.** Pi's rules, keyed by model rather than provider, are the ones gobble's Pi session format expects, and the
SDK leaves them to the caller except for call ids.

## 3. Catalogs and key checks

- **Catalog sources:** opencode and Kilo embed a models.dev snapshot, cache it on disk with a TTL and refresh it in
  the background (`packages/core/src/models-dev.ts:280-435`; Kilo `models-dev.ts:171-270`); goose the same with an
  ETag (`crates/goose/src/model_catalog.rs`); Pi a generated list plus a 4-hour remote overlay
  (`remote-catalog-provider.ts:14-146`).
- **The SDK already layers this:** `catalog.List` is live, cached 10 minutes, a stale listing up to 24 hours on a
  transient failure, the static catalog otherwise, with `Catalog.Err` classified and `Source` live, stale or static;
  `WithLastGood` keeps a listing across runs; `Static` and the models.dev `Metadata` need no credential; `Menu` carries
  rank, caveats and each model's facts (context window, costs, reasoning efforts, release, status).
- **Key checks:** none of the four validates a key at login (Pi `ai/src/auth/helpers.ts:9-31`; opencode
  `integration.ts:744-770`; Kilo none found). goose spends a 50-token streamed request (`provider_test.rs:7-46`). Kilo
  shows a per-provider red mark with a mapped reason ("Credentials rejected (HTTP 401) — sign in again",
  `cli dialog-provider.tsx:22-60`). The SDK's wizard checks a typed key by listing and re-asks on `ErrAuthFailure`
  (`wizard/configure.go:328`). OpenCode Zen and Go, Hugging Face and Kilo list without a key, so a listing cannot
  reject theirs.

## 4. Sign-in

- **Pi** `/login`: method first (account or API key), then providers with their state — "✓ configured", "✓ env:
  OPENAI_API_KEY", "• not configured" (`components/oauth-selector.ts:35-52`); after a login the provider's default
  model is selected (`interactive-mode.ts:6087-6180`); `/logout` lists stored credentials only and says the
  environment is unchanged (`:6024-6085`).
- **opencode** `auth login [target] [--method id] [--answer k=v]`, non-interactive with a target and a method
  (`packages/cli/src/commands/handlers/auth/login.ts:49-120`); after `/connect` the model picker opens filtered to the new
  provider (`dialog-integration.tsx:1248-1268`).
- **goose** asks for keys only in `goose configure`, outside the session; writes secrets to plain `secrets.yaml` when
  the keyring is unavailable, silently (`crates/goose/src/config/base.rs:1222-1240`); signs in to Copilot and ChatGPT
  with other products' client ids (`githubcopilot.rs:88`, `chatgpt_codex.rs:37-45`).
- **The SDK's wizard**: `wizard.ConfigureLLM(ctx, Prompter, Options)` asks for the provider, the credential (only the
  methods it can keep; the ChatGPT plan only with `AgentName`), checks a typed key by listing, then the model; the
  `Prompter` (Select, MultiSelect, Confirm, Input, Secret, Notify) has a line-mode `TextPrompter`; `wizard.Logout`
  confirms, revokes OpenAI and Grok sessions, and deletes (`wizard/prompter.go:68-92`) **(checked)**.

## 5. What to take, and what not

Take, in order of value for gobble:

1. Pi's replay rules, keyed by model; the SDK's call-id remap and Gemini marker cover the rest.
2. One name for a model everywhere, `provider/model[:thinking]`, which `--model` already parses by Pi's rules; a bare id
   when unique; candidates listed when not.
3. A switch is the session's; recent models are kept (ten, newest first) and the newest starts the next session; each
   provider's last model is where "switch to provider X" lands; saving a default is explicit.
4. Signing in from where the user is: `/login` from the picker and from a refused switch; after it, that provider's
   models offered at once.
5. The SDK's wizard and `Prompter` for `/login` and first-run setup, with gobble's own Prompter over the line editor.
6. The SDK's catalog layers as they are, with `WithLastGood` kept on disk so a start shows the last good listing at
   once.
7. Per-provider states with a reason, as Kilo shows them; "not checked" for a provider whose listing is public.
8. A warning when a switch shrinks the context window (goose).
9. Pi's cycling of a scoped set (`enabledModels` globs) on a key, for a terminal user who moves between a few models.
10. goose's one strip-and-retry when a provider rejects a thinking signature, as a safety net under rule 1.

Do not take: plaintext secrets as a silent fallback; another product's client id (gobble's rule, 0005-MADR; Grok's is
the owner's accepted risk); sign-in only outside the session; a billed request to check a key.
