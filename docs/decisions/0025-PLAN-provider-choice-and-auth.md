---
status: proposed
date: 2026-10-20
associated-madr: "0025-MADR-provider-choice-and-auth.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement any provider, any model, browsing and in-session sign-in

Associated MADR: [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md)

*Proposed. It starts after the owner approves it, on `main` at or after `d76e3d2`. The MADR's Q1 and Q2 are answered
(2026-10-20): another provider's thinking is dropped on a switch; consent under a plan is asked once and saved.
The phases run in order, each one commit, the owner's; each starts by measuring again what it relies on, and a fact
that differs is a deviation, stopped and asked.*

## Goal

With several credentials, every provider whose key or sign-in works is usable: chosen at start (D2), switched
mid-conversation to any of its models (D1), its state shown (D3), browsable with its catalog in the CLI and the
session (D4), and signed in to from the session, ChatGPT's plan and Grok's subscription included (D5).

## Scope

- **In:** `llm/provider` (listing every provider, the choice, the replay filter); `acpserver` (the model choice, `/model`,
  the `model` option, `/providers`, `/models`, the switch, consent under a plan); `internal/cli` (the start choice,
  `last-model.json`, `gobble providers`, `models list --provider`, `/login`, `/logout`, the plan sign-in, the
  credential cache); `internal/auth` (an OpenAI plan login in `auth.json`); their tests; `README.md`,
  `docs/architecture.md`, `docs/README.md`; the dated notes in 0019-MADR, 0020-MADR, 0021-MADR, 0005-MADR and
  0005-PLAN.
- **Out:** any new SDK release (the SDK at `v1.17.0` has what is needed); the TUI; Anthropic, Gemini and Copilot
  subscription sign-ins (0005-MADR: out of v1); a funds or balance check (the SDK has none); changing the Pi import's
  rules.

## Implementation Steps

### P1 — every credentialed provider's models, with their state (D1's data, D3)

1. **Tests first,** each seen failing on today's code:
   - `internal/cli`: with scripted listings for three providers with credentials and one without, the model choice
     holds the three, each with its models and state; one whose listing returns `ErrAuth` is marked
     `key rejected` with its reason and offers no models; one offline keeps its static models, marked `not listed`;
     the listings run concurrently (a slow one does not delay the others' results).
   - `acpserver`: `/model` with no argument lists every provider's models under its name, the marked ones with their
     state; the ACP `model` option is grouped by provider, values `provider/model`.
2. **The choice:** `acpserver.ModelChoice` becomes a list of `ProviderChoice{ID, Label, Models, State, Reason}` with the
   session's current `provider/model`; `internal/cli`'s `modelChoice` lists each credentialed provider through the
   existing listing path, once per process (0020-MADR D3, now per provider), in the background.
3. **`/model` and the option** read the new choice; a value is `provider/model`; a bare id resolves when exactly one
   offered provider has it, else the answer names the candidates.
4. **Records:** dated notes in 0020-MADR D3–D5 and 0005-PLAN F4-2c decision 4.

*P1 does not switch provider yet:* choosing another provider's model answers that switching comes with P2, and the
phase's commit says so.

### P2 — switching provider mid-conversation (D1's switch and replay)

1. **Tests first:**
   - `llm/provider` or `acpserver`'s replay: a history whose assistant messages came from provider A, replayed to B,
     has no `ProviderData`, signatures or encrypted reasoning from A, and (by Q1's answer) no A thinking; replayed to A
     it is unchanged; text, tool calls and tool results are kept in both.
   - `acpserver`: `/model b/model-x` in a session on A builds B, writes `model_change` with provider B, and the next
     turn goes to B with the filtered history; the ACP option does the same; a marked provider is refused with its
     reason.
   - a session restored from a log whose last `model_change` names B starts on B when B has a credential, else on the
     start choice with a note.
2. **The switch:** the live session's provider becomes rebuildable; a switch closes nothing of the session; the
   replay filter runs where the history is turned into the request.
3. **Records:** dated notes in 0021-MADR D4 and 0019-MADR D15.
4. **Live check** on the owner's keys, recorded without any credential: one conversation that starts on one provider,
   switches to a second after a tool call, and goes on.

### P3 — the start choice and the last choice (D2)

1. **Tests first,** on `internal/cli`'s start choice with a fake state directory: the order `--model`/`--provider`,
   then `defaultProvider`/`defaultModel`, then `last-model.json`, then the registry order with its note; a last choice
   whose provider lost its credential or is marked falls through with a note; `--model` to a provider with no
   credential still fails.
2. **`last-model.json`** in the state directory, `{"model":"provider/model"}`, written 0600 when `/model`, the option
   or `--model` chooses; an unreadable file is ignored with a warning.
3. **The note** on a registry-order start: which provider, and how to switch.

### P4 — browsing (D4)

1. **Tests first:** `gobble providers` lists all ten providers with their auth methods and credential sources, and
   with `--check` D3's states; `gobble models list --provider claude` with no credential lists its static models,
   marked; `/providers` and `/models [provider] [search]` answer in the session and are advertised to ACP clients;
   `/help` lists them.
2. **The commands** in `internal/cli` (Kong) and `acpserver`'s command table, with D3 of 0024-MADR's description rule.
3. **Records:** `README.md` and `docs/architecture.md`.

### P5 — `/login` and `/logout` in the session (D5, the session part)

1. **Tests first:** `/login` with no provider shows the numbered providers and their methods; `/login claude` reads a
   key with no echo (the editor's hidden input), stores it in `auth.json`, refreshes the credential cache, lists
   claude, and claude's models are then offered by `/model`; `/logout claude` removes it and claude is no longer
   offered; Grok's and Kilo's sign-ins run through the existing login functions, out of raw mode (0024-MADR D1).
2. **The credential cache** (`credsOnce`, `internal/cli/credentials.go:31-52`) becomes refreshable after a login or a
   logout; the model choice re-lists the provider that changed.
3. **Records:** a dated note in 0005-MADR (`/login`, `/logout` in the line session).

### P6 — ChatGPT's plan sign-in (D5, 0005-PLAN F4-3d)

1. **Measure first:** the SDK's `LoginChatGPTPlan`, `OAuthSession.PlanUsage`, and how the openai provider uses a plan
   session at `v1.17.0`; OpenAI's terms page the SDK's 0043-MADR cites, read again, and its date recorded. A
   difference from the MADR is a deviation.
2. **Tests first,** against the SDK's test issuer if it has one, else a scripted one: `gobble auth login openai` (and
   `/login openai`) offers the plan sign-in and the API key; the plan session is stored in `auth.json` as OAuth; the
   openai provider is built with it; `HostID` is stable across runs; the refusal at `internal/cli/auth.go:197-198` is
   gone; consent (Q2's answer) is asked once, saved, and without it no title or automatic compaction request is made
   under the plan.
3. **The sign-in** with `AgentName` "gobble", `OpenURL` printing the URL when no browser opens, `InputCode` reading the
   pasted code.
4. **Records:** 0005-PLAN F4-3d's execution record; a dated note in 0005-MADR's ChatGPT row; `README.md`.
5. **Live check** by the owner's own sign-in (the agent cannot sign in for the owner): the owner runs `gobble auth
   login openai`, then one turn on a plan model; recorded without any credential.

## Verification

- Every new test seen failing first, on the unchanged code or on a scratch copy, its failure quoted in the record.
- Each phase: the pre-add check on its changed `.go` files; lint for linux, darwin and windows; `go test -count=1
  ./...`, `CGO_ENABLED=1 go test -race -count=1 ./...` and `make preflight` on macOS, the Linux machine and the Windows
  laptop, the remote two on a copy of the tree; `check-records` and `markdownlint`.
- Live checks (P2, P5, P6) on the owner's credentials, run so that no key, token or sign-in URL reaches a record, a log
  shown here, or this conversation: only provider ids, model ids, states and outcomes are recorded.

## Rollout and Rollback

One commit per phase, the owner's. Each phase reverts on its own, from the last back: P6 restores the refusal, P5 the
CLI-only login, P4 removes the commands, P3 the registry-order start, P2 the one-provider session, P1 the one
provider's model list. `auth.json` entries written by P5 and P6 are Pi's shape, read by the reverted code as before
(an OpenAI OAuth entry is then skipped, as the Pi import skips one).
