---
status: proposed
date: 2026-10-20
associated-madr: "0025-MADR-provider-choice-and-auth.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement any provider, any model, browsing and in-session sign-in

Associated MADR: [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md)

*Proposed. It starts after the owner approves it, on `main` at or after `d76e3d2`. The MADR's questions are all
answered (2026-10-20): another model's thinking is dropped on a switch (Q1); consent under a plan is asked once and
saved (Q2); first-run setup is offered (Q3, P5); one strip-and-retry (Q4, P2); cycling (Q5, P7).
The phases run in order, each one commit, the owner's; each starts by measuring again what it relies on, and a fact
that differs is a deviation, stopped and asked.*

*(Revised 2026-10-20, before review, with the MADR's revision from
[0025-REPORT-provider-ux-across-harnesses.md](../reports/0025-REPORT-provider-ux-across-harnesses.md): seven phases
where there were six; the memory file and the picker move into P1; Pi's replay rules into P2; the SDK's wizard into
P5; first-run setup into P5 and cycling into P7, each only if the owner takes it (MADR Q3, Q5).)*

## Goal

With several credentials, every provider whose key or sign-in works is usable: chosen at start (D2), switched
mid-conversation to any of its models (D1), its state shown (D3), browsable with its catalog in the CLI and the
session (D4), and signed in to from the session, ChatGPT's plan and Grok's subscription included (D5).

## Scope

- **In:** `llm/provider` (listing every provider, the states, the replay rules); `acpserver` (the model choice,
  `/model`, the grouped `model` option, `/providers`, `/models`, the switch, consent under a plan); `internal/cli` (the
  start choice, `<state>/models.json`, the listing cache, the line picker, `gobble providers`, `models list
  --provider/--all`, `/login`, `/logout`, `gobble setup`, `auth login --method`, the wizard's `Prompter`, the plan
  sign-in, the credential cache); `internal/cli/editor` (Ctrl+P, P7 only); `internal/auth` (an OpenAI plan login in
  `auth.json`); their tests; `README.md`,
  `docs/architecture.md`, `docs/README.md`; the dated notes in 0019-MADR, 0020-MADR, 0021-MADR, 0005-MADR and
  0005-PLAN.
- **Out:** any new SDK release (the SDK at `v1.17.0` has what is needed); the TUI; Anthropic, Gemini and Copilot
  subscription sign-ins (0005-MADR: out of v1); a funds or balance check (the SDK has none); changing the Pi import's
  rules.

## Implementation Steps

### P1 — every credentialed provider's models, their states, and the picker (D1's names, D3's memory, D4, D5's picker)

1. **Tests first,** each seen failing on today's code:
   - `internal/cli`: with scripted listings for providers in each D4 state (live, stale, key rejected, quota
     exhausted, not listed, not checked, no credential), the model choice holds each with its state and reason, and
     offers models only from the usable ones; the listings run concurrently; a last good listing on disk is shown before
     the live one replaces it.
   - `<state>/models.json`: a choice puts its ref first, ten kept, duplicates removed, written 0600; an unreadable file
     is ignored with a warning.
   - the ref grammar: `provider/model[:thinking]`, a unique bare id, a fuzzy match, an ambiguity with its candidates,
     through the resolver `--model` uses.
   - `acpserver`: `/model` with no argument prints the numbered picker (current, recent, each provider under its state
     with the menu's facts), within the terminal width (0024-MADR D2); `/model <words>` filters; the ACP `model` option
     is grouped, "Recent" first.
2. **The choice:** `acpserver.ModelChoice` becomes the session's current ref plus a list of `ProviderChoice{ID, Label,
   State, Reason, Models, Source, Age}`; `internal/cli` lists each credentialed provider through `catalog.List` with
   `WithLastGood` on `<cache>/catalog/<id>.json`, once a process, in the background.
3. **A request's `ErrAuth` or `ErrQuota`** marks its provider in the choice (D4's lazy marking).
4. **Records:** dated notes in 0020-MADR D3–D5 and 0005-PLAN F4-2c decision 4.

*P1 does not switch provider:* a ref to another provider's model answers that switching comes with P2.

### P2 — switching, and replay on another model (D1's switch, D2)

1. **Tests first:**
   - the replay (where history becomes a request): Pi's rules, case by case, mirroring `transform-messages.ts`'s:
     same model kept whole; another model's thinking dropped with signatures and encrypted reasoning; its text bare;
     its tool calls without `ProviderData` or thought signature; an errored or aborted turn skipped; an unanswered call
     given "No result provided"; a switch within one provider counted as another model.
   - `acpserver`: `/model b/model-x` on A builds B, writes `model_change` with B, and the next turn goes to B through
     the rules; `/model b` lands on B's last model; a dead provider is refused with its reason; a smaller context window
     warns and suggests `/compact`; the ACP option does the same; a restored session follows D1's rule.
2. **The switch:** the live session's provider becomes rebuildable, under the session's lock; the replay rules run in
   `llm/provider`'s request building, given the target's provider, wire format and model.
3. **Q4 (taken):** a request refused over a thinking signature is retried once with every thinking block
   stripped; a test scripts the refusal.
4. **Records:** dated notes in 0021-MADR D4 and 0019-MADR D15.
5. **Live check** on the owner's keys, recorded without any credential: a conversation starts on one provider, makes a
   tool call, switches to a second, goes on, and switches within the second provider to another model.

### P3 — the start order and the saved default (D3)

1. **Tests first,** with a fake state directory: the order `--model`/`--provider`, settings, `recent`'s newest usable,
   registry order with its note; a dead or uncredentialed recent falls through with a note; `--model` to a provider
   with no credential still fails; `/model <ref> --default` writes `defaultModel` and `defaultProvider` to the global
   settings.
2. **The start** in `internal/cli`, with the note.

### P4 — browsing (D5)

1. **Tests first:** `gobble providers` lists the ten with state, source in Pi's words and sign-in methods, `--check`
   lists them first, `--json` is stable; `gobble models list --provider claude` with no credential and `--all` show the
   static catalog marked so; `/providers` and `/models [provider] [words]` answer in the session, are advertised to ACP
   clients, and are in `/help`.
2. **The commands** in `internal/cli` and `acpserver`, with 0024-MADR D3's description rule.
3. **Records:** `README.md`, `docs/architecture.md`.

### P5 — signing in where the user is (D6; D7, Q3 taken)

1. **Tests first:**
   - gobble's `Prompter` over the line editor: `Select` numbered, `Secret` with no echo, `Confirm`, `Input`, `Notify`,
     each out of raw mode;
   - `/login` with no provider runs `wizard.ConfigureLLM` with the providers' states; `/login claude` with a key the
     scripted listing refuses asks again; one it accepts is stored in `auth.json`, the cache refreshed, claude listed,
     and the picker opened filtered to it;
   - `/logout` lists stored credentials only, revokes a scripted Grok session, and says the environment still applies;
   - `gobble auth login claude --method api-key` with the key on stdin needs no menu; an unknown method is refused with
     the provider's methods;
   - a `/model` to a provider with no credential offers `/login <provider>`;
   - D7: with no credential the session offers the wizard before its first prompt; declined, it goes on; `gobble
     setup` runs it.
2. **The credential cache** (`internal/cli/credentials.go:31-52`) becomes refreshable after a login or a logout.
3. **Records:** a dated note in 0005-MADR (`/login`, `/logout` in the line session).

### P6 — ChatGPT's plan sign-in (D6, 0005-PLAN F4-3d)

1. **Measure first:** the SDK's `LoginChatGPTPlan`, `OAuthSession.PlanUsage`, the openai provider under a plan session,
   and whether `wizard.ConfigureLLM` offers the plan with `AgentName` set and leaves out the Codex-id methods when
   asked; OpenAI's terms page as the SDK's 0043-MADR cites it, read again, its date recorded. A difference from the
   MADR is a deviation.
2. **Tests first,** against a scripted issuer: `/login openai` and `--method chatgpt-plan` offer the plan and the API key
   only; the session is stored as OAuth; the openai provider is built with it; `HostID` is stable; the refusal at
   `internal/cli/auth.go:197-198` is gone; consent is asked once, saved, and without it no title or automatic
   compaction request is made under the plan.
3. **The sign-in** with `AgentName` "gobble", `OpenURL` printing the URL when no browser opens, `InputCode` reading the
   pasted code.
4. **Records:** 0005-PLAN F4-3d's execution record; a dated note in 0005-MADR's ChatGPT row; `README.md`.
5. **Live check** by the owner's own sign-in: `gobble auth login openai --method chatgpt-plan`, then one turn on a plan
   model; recorded without any credential.

### P7 — cycling a scoped set (D8, Q5 taken)

1. **Measure first:** the bytes Ctrl+P and Alt+P send on macOS Terminal, a Linux terminal and Windows Terminal, through
   the line editor's callback; a key that does not reach it is a deviation.
2. **Tests first:** `enabledModels` globs select the set from the offered models; Ctrl+P switches to the next, wrapping,
   and prints it; with no set it cycles `recent`; the previous key does the reverse.
3. **The key** in `internal/cli/editor`'s callback, and the `enabledModels` setting with its schema and template entry
   (0018-MADR D19's rule for a new key).

## Verification

- Every new test seen failing first, on the unchanged code or on a scratch copy, its failure quoted in the record.
- Each phase: the pre-add check on its changed `.go` files; lint for linux, darwin and windows; `go test -count=1
  ./...`, `CGO_ENABLED=1 go test -race -count=1 ./...` and `make preflight` on macOS, the Linux machine and the Windows
  laptop, the remote two on a copy of the tree; `check-records` and `markdownlint`.
- Live checks (P2, P6) on the owner's credentials, run so that no key, token or sign-in URL reaches a record, a log
  shown here, or this conversation: only provider ids, model ids, states and outcomes are recorded.

## Rollout and Rollback

One commit per phase, the owner's. Each phase reverts on its own, from the last back: P7 removes the key and the
setting, P6 restores the refusal, P5 the CLI-only login, P4 removes the commands, P3 the registry-order start, P2 the
one-provider session, P1 the one provider's model list. `<state>/models.json` and `<cache>/catalog/` are ignored by
the reverted code. `auth.json` entries written by P5 and P6 are Pi's shape, read by the reverted code as before
(an OpenAI OAuth entry is then skipped, as the Pi import skips one).
