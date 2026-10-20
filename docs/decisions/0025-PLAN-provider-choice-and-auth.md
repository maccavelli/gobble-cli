---
status: in-progress
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

## Execution record

**Deviation 1, 2026-10-20, P1, decided by the owner** ("Kind and status only"): MADR D4 shows a dead provider "with the
provider's reason", but 0020-MADR F5 found that "the provider's message can carry a masked fragment of the key, so it
must not reach a log", and D6 logs only `FailureKind`, "never the error's text".

- **Chosen:** the reason is gobble's words for the kind (`key rejected`, `quota exhausted`, `not permitted`, `not
  listed: offline`, …), with the HTTP status when the error carries one, and the way out (`/login <provider>`). MADR D4
  annotated.
- **Not chosen:** the provider's text with key-like parts masked.
- No file is added to P1's scope.

**Deviation 2, 2026-10-20, P1, decided by the owner** ("/model N, markdown"): MADR D5's picker "reads a number to switch,
or words to filter", but `/model` is an agent command, run by every ACP client, and cannot pause for a reply.

- **Chosen:** `/model` prints the list numbered, as markdown: the current model, the recent ones, then a heading per
  provider with its state and a table of its models (#, model, context, input and output price per million tokens,
  thinking); `/model N` picks entry N; `/model <words>` that is not a ref prints the matching entries with their
  numbers. The CLI renders the tables aligned (`internal/cli/render/table.go`), and editors render markdown. MADR D5
  annotated.
- **Not chosen:** an interactive prompt in the line session alone, with the printed list for ACP clients.
- No file is added to P1's scope.

### P1 execution record (2026-10-20)

**Run** on `main` at `a417f51`, on the owner's word "committed, proceed", with deviations 1 and 2 above.

**Clarification, not a deviation:** MADR D1 names models by `--model`'s rules, a fuzzy match included, and D5 has
`/model <words>` list the matches. Both hold by keeping each where it serves: `/model` switches on an entry's number, a
`provider/model` ref, or an id exactly one usable provider offers; any other words list their matches (D5), by the
SDK's matcher (`SearchModels`) and each word in the ref or the model's name. The fuzzy pick stays with `--model` and
`defaultModel`, which have no list to show. `llm/catalog`'s `parsePattern` picks one model for a short word such as
`m`, which in `/model` would switch where the user meant to filter.

**Steps:**

1. **Tests first,** each seen failing:
   - `llm/provider`: `TestListSourceAndUsable`, `TestListServesLastGood`, `TestPublicListing`, `TestSearch`, against
     stubs: `Source: FetchedAt:0001-01-01 … Usable:[]`; the 503 gave the static models, `Live:false`;
     `PublicListing("opencode-zen") = false` (and Go's, Hugging Face's, Kilo's); `Search mini = []`.
   - `acpserver`: `TestModelPicker`, `TestModelPick`, `TestModelOptionGrouped`, `TestRefusedKeyMarksProvider`, against
     the unchanged `/model`: `/model lacks "Model: fake/m1"` (it answered `Model: m1\nModels: m1, m2`), `/model 4:
     "Unknown model \"4\"…"`; and `TestStateOf`, whose eight failure cases each failed on a scratch copy whose
     `StateOf` answered `not listed` with no reason (`StateOf(… sk-pl***nt) = not listed "", want quota exhausted
     "(HTTP 429)"`, … `want not listed "(listing failed)"`).
   - `internal/cli`: `TestProviderChoicesStates`, `TestProviderChoicesConcurrent`, `TestRecentModels`,
     `TestLastGoodOnDisk`, `TestChoiceWiring`, against stubs: `providers [], want ["gemini" "openai" …]`, `recent [],
     want ["openai/m5" …]`, `claude while listing: {…State: …}, want the kept listing, stale`, `the agent remembers
     nothing or searches without the SDK's matcher`.
2. **The choice:**
   - `llm/provider`: `Listing` gains `Source`, `FetchedAt` and `Usable`; `Options` gains `LastGood` and `LastGoodAt`,
     handed to the SDK as `catalog.WithLastGood`; `PublicListing` (OpenCode Zen and Go, Hugging Face, Kilo, from the
     SDK's `discovery.go:1012`, `:1045`, `:1251`); `Search` over `catalog.Search`.
   - `acpserver` (`choice.go`): `ProviderState` and its nine states; `ProviderChoice`; `StateOf`, gobble's words and
     the HTTP status, never the error's text; `ModelChoice` gains `Providers` and `Recent`, a choice without
     `Providers` read as its own provider, live; `Options.Remember` and `Options.SearchModels`.
   - `internal/cli` (`choice.go`): every key provider in registry order; the run's listing waited for, the others
     peeked, a running one shown by its kept listing (`<cache>/catalog/<id>.json`, within 24 hours) as stale, else as
     listing with its static models; `<state>/models.json`, read once a run, ten kept, written owner-only through a
     temporary file; `--model` remembered; every credentialed provider's listing started with the agent.
3. **Lazy marking:** a turn failing with `ErrAuth`, `ErrQuota` or `ErrNotPermitted` marks its provider for the
   process (`acpserver/agent.go`, at the turn's failure).
4. **`/model` and the option:** `/model` prints the numbered markdown list (deviation 2), `/model N`, a ref, or an id
   one provider offers switches, other words list their matches, an unknown model names five; the option is grouped,
   `recent` then each usable provider, values refs; a bare id of the session's provider is still taken. A model of
   another provider answers "Switching to another provider's model is not available yet; this session's provider is
   …", until P2.
5. **Records:** dated notes in 0020-MADR D3 and D5 and 0005-PLAN F4-2c decision 4; this record.

**Tests changed with it,** to the refs D1 gives the option and `/model`, none loosened: `TestModelAndThinkingCommands`
(`/model m2` answers `Model: fake/m2`; `/model` lists `| 1 | fake/m1 |`), `TestConfigOptionsPersist`,
`TestOverrideNew`, `TestOverrideResume`, `TestOverrideFork` and `TestPiFixtureContext` (the option's current value a
ref), `TestModelOptionOffersListedModels` (the grouped option's values, the recent group left out, and the reply
naming `openai/gpt-5-nano`). The ACP golden is unchanged: its agent offers no model.

**Found and fixed in the phase:** two tests of mine raced their cleanup (a held listing wrote its kept file into a
home being removed); each now ends its listing first. Thirty runs of the new tests under the race detector passed.
Lint asked for a constant for "Recent", `strings.FieldsSeq`, and the repo's `G304` / `G101` annotations; done.

**In the terminal** (gobble built to a scratch directory, a scratch home, two dummy keys and a dead proxy, so no
request left the machine; every listing failed, so each provider showed its static models): with colour, `/model`
printed `The model is openai/gpt-6-luna.`, `Also available: 11 models from 2 providers`, then an aligned table per
provider (`|   1 | openai/gpt-6-luna    |    1.1M |   0.10 |    0.50 | off–max  |`), then the seven providers with no
credential, each `— set <VAR>, or gobble auth login <id>`. With `NO_COLOR` the markdown is printed raw, as 0008-MADR D7
has all agent text. Bold keeps its `**` markers in the line view, by 0008-PLAN P8's styling. Two things were seen and
are not P1's: an unclassified listing failure first showed no reason, now "(listing failed)" (`TestStateOf`); a
spinner's label runs into the reply's first line, which the capture's stripped escapes may cause, unmeasured.

**Gates:**

- the pre-add check on the nineteen changed `.go` files: `go-precheck: 19 file(s) clean (gofmt, golint, govulncheck)`;
  lint for linux, darwin and windows, `0 issues.` each;
- macOS: `go test -count=1 ./...` and `CGO_ENABLED=1 go test -race -count=1 ./...`, 35 packages ok each; `make
  preflight`, `preflight passed`;
- the Linux machine and the Windows laptop, over ssh with `-o BatchMode=yes -o ConnectTimeout=5`, on a copy of the
  working tree, Windows with `TMP` and `TEMP` outside the user's home: the tests and the race detector, 35 packages ok
  on each; `make preflight` through `apidiff` (golangci-lint `0 issues.`), stopping at `verify-build-metadata` with
  `git rev-parse HEAD` exiting 128 on a copy; `make check-records markdownlint`, "0 issues in 0 files".

**Not done in P1, as the plan says:** switching to another provider's model (P2); the start order (P3); `/providers`,
`/models`, `gobble providers` (P4); `/login` (P5). Until P5, a refused key's way out names `gobble auth login`.

**Deviation 3, 2026-10-20, P2, decided by the owner** ("Fix llmtest"): `llm/llmtest`'s `cloneRequest`
(`llm/llmtest/script.go:109-120`) rebuilt each recorded message from its role and content only, so a test saw no
`llm.Message.Origin`, the field P2 adds; the requests a provider receives were unaffected. `TestSwitchProvider` found
it: the other provider's recorded request had no origin on the first provider's answer.

- **Chosen:** `cloneRequest` copies the whole message and clones its content, so a recorded request keeps every field.
- **Not chosen:** checking the origin in the session's history instead, which would not show what reaches the
  provider.
- **File added to P2's scope:** `llm/llmtest/script.go`.

### P2 execution record (2026-10-20)

**Run** on `main` at `bdd8268`, on the owner's word "committed, proceed", with deviation 3 above.

**Steps:**

1. **Tests first,** each seen failing:
   - `llm/provider` (`replay_test.go`): `TestReplayOtherModel`, `TestReplayFailedAndOrphaned` and
     `TestStripThinkingAndSignatureRefused` failed on stubs that returned their input (`to b/mb:
     ["user:text" "assistant:thinking,text,tool_call" …] with provider data true`); `TestWireReplay` (`another model's
     thinking was sent: {… "signature":"sig-A" …}`) and `TestWireSignatureRetry` (`the retried request failed: … 400 …
     Invalid ` + "`signature`" + ` in ` + "`thinking`" + ` block`) on the unchanged `Stream`. `TestReplaySameModel` and
     `TestReplayUnknownOrigin` assert what stays the same, so each was seen failing on a scratch copy instead: every
     known origin treated as foreign (`same model: [… "assistant:text,tool_call" …], want … thinking … with its
     signatures`), and a zero origin treated as foreign (`unknown origin: …, want it whole`).
   - `acpserver` (`switch_test.go`): `TestSwitchProvider` (command and option), `TestSwitchToProviderLandsOnItsLast`,
     `TestSwitchWarnsOnSmallerWindow`, `TestSwitchBuildFails`, `TestRestoreSwitchedSession`, on P1's code (`/model
     other/x9: "Switching to another provider's model is not available yet…"`; the restored turn `llmtest: script
     exhausted`). `TestTokensShort`, for a window under a thousand shown as "0K", on a scratch copy (`tokensShort(5) =
     "0K", want "5"`).
   - `internal/cli`: `TestProviderForBuildsNamed`, on a scratch copy whose `providerFor` built the run's provider
     (`built "" for session "sess-7"…`).
2. **The origin:** `llm.Message` gains `Origin{Provider, Model, Failed}`. A turn's assistant messages are stamped with
   the session's provider and model at the turn's end, a cancelled turn's last one failed (`stamped`); a log's are
   stamped from its entries, `error` and `aborted` failed, and a Pi provider name mapped to gobble's (`anthropic` is
   `claude`, `canonicalProvider` over `catalog.ProviderID`), seen failing on a scratch copy that kept the name
   (`TestPiFixtureContext`: `restored … [claude/other medium]`).
3. **The replay** (`llm/provider/replay.go`), where the adapter builds a request: Pi's rules keyed by provider and
   model (a model's wire follows from those two); another model's thinking dropped and its text and calls without
   `ProviderData`; a failed turn not replayed, nor its calls' results; an unanswered call given "No result provided"
   before the next assistant or user message and at the end; a message with no origin sent as it was.
4. **Q4:** `Stream` makes each attempt through `attempt`; a 400 whose text names a signature or encrypted content
   (`signatureRefused`), before any event, is sent once more with every thinking block and signature stripped; any
   other 400 is not.
5. **The switch:** `liveSession.providerID`; `setConfig` takes a ref, builds another provider first through
   `Options.ProviderFor` (`buildProvider`), so a failure (`switchError`, "Cannot switch to … : …") leaves the session
   as it was, then records `model_change` with the provider; `sessionProvider` builds a switched session's provider
   through `ProviderFor`. `modelOf`, `levelsOf` and `qualified` take the provider, and every caller passes the
   session's (`provOf`): the thinking levels, the window, the price, `/usage`, `/session`, compaction, `usage_update`.
   `/model <provider>` lands on that provider's newest recent model, else its first; a switch to a smaller window than
   the context in use says so and suggests `/compact`. A restored session goes on with its last provider when it is
   usable, else with the run's and a warning (`provider=…`). `--model` still wins over a restored session.
6. **The CLI:** `providerFor` builds a named provider as `modelProvider` builds the run's: its key in Pi's order or its
   login, `models.json`'s base URL and headers, its credential's source, the session's id and retry settings.
7. **Records:** dated notes in 0021-MADR D4 and 0019-MADR D15; this record.

**Tests changed with it, none loosened:** `TestModelPick`'s three answers for another provider's model, P1's "not
available yet", are now the switch's own refusal for an agent with no `ProviderFor`; `TestPiFixtureContext`'s provider
is gobble's `claude`, which a Pi session names `anthropic`, so it now checks the mapping (`claude/claude-sonnet-4-5`
restored).

**The live check,** on the owner's credential variables, in a scratch home and working directory, with only provider
ids, model ids and outcomes kept:

```text
/model gemini/gemini-3.5-flash-lite   → Very good. The model is now gemini/gemini-3.5-flash-lite.
read hello.txt, its first word        → ▸ read hello.txt ✓ … Pineapple.        ($0.0025)
/model openai/gpt-6-luna              → openai/gpt-6-luna it is.
What was that word?                   → Pineapple                               ($0.0031)
/model openai/gpt-5.6-luna            → Very good. The model is now openai/gpt-5.6-luna.
Spell that word backwards             → elppaenip                               ($0.0042)
/session                              → Total: 8, Tools: 1 calls, 1 results
```

OpenAI took the conversation Gemini began, a tool call and its result included, with Gemini's data left out and no
error; the switch within OpenAI's models went on as well.

**Gates:**

- the pre-add check on the nineteen changed `.go` files, `19 file(s) clean`, after lint's asks: Pi's stop reasons as
  constants (`piAborted`, `piError`), `strconv.Itoa`, a builder in a test, and two providers' error texts passed
  through a variable so golint does not read them as gobble's; lint for linux, darwin and windows, `0 issues.` each;
- macOS: the tests and the race detector, 35 packages ok each; `make preflight`, `preflight passed`;
- the Linux machine and the Windows laptop, over ssh with `-o BatchMode=yes -o ConnectTimeout=5`, on a copy of the
  working tree, Windows with `TMP` and `TEMP` outside the user's home: the tests and the race detector, 35 packages ok
  on each; `make preflight` through `apidiff` (`0 issues.`), stopping at `verify-build-metadata` on a copy; `make
  check-records markdownlint`, "0 issues in 0 files".

**Not done in P2, as the plan says:** the start order (P3); browsing (P4); `/login` (P5). A switch to a provider with no
credential says it is not usable, with its variable; P5 offers `/login` there.

**Deviation 4, 2026-10-20, P3, decided by the owner** ("Rename to recent-models.json"): P1 kept the recent models in
`<state>/models.json`, as MADR D3 named it. gobble's custom models file is `models.json` in the config directory
(`internal/cli/modelsjson.go:40`), and with `GOBBLE_HOME` set the config and state directories are one
(`internal/appdirs/dirs.go:56-61`), so choosing a model rewrote the user's custom models; with the platforms' own
directories they are apart. P3's start-order test found it: its environment, under `GOBBLE_HOME`, warned "models.json:
providers: required; the file is ignored" after a recent model was remembered. P1 was never released, so no file needs
moving.

- **Chosen:** the recent models in `<state>/recent-models.json`; MADR D3 annotated; a test that a `models.json` beside
  it is left as it was.
- **Not chosen:** the same name in a subdirectory.
- No file is added to P3's scope beyond `internal/cli/choice.go` and `internal/cli/choice_test.go`, P1's, where the
  name is.

### P3 execution record (2026-10-20)

**Run** on `main` at `4377b6d`, on the owner's word "committed, proceed", with deviation 4 above.

**Steps:**

1. **Tests first,** each seen failing:
   - `internal/cli` (`start_test.go`): `TestStartOrder`, against a stub that kept today's start (`started on /, want
     claude/claude-2`; `started on openai/openai-1, want claude/claude-2`; `note "", want "Starting on OpenAI
     (openai)…"`); `TestStartNoteInLineSession` (`condition not met within 5 s`) before the line session printed the
     note. On scratch copies: `TestStartOrder` with the start ignoring a refused listing (`started on
     claude/claude-haiku-5-5, want openai/openai-1`); `TestStartNoCredential` with the setting dropped when nothing has
     a credential (`… set one of GEMINI_API_KEY, OPENAI_API_KEY, …`); `TestRecentModels` with the file named
     `models.json` again (`the custom models.json is now "{\n  \"recent\": [\n    \"openai/m5\", …`).
   - `acpserver`: `TestModelSaveDefault` (`/model fake/m2 --default: "Unknown model \"fake/m2 --default\"…"`).
2. **The start** (`internal/cli/start.go`), chosen once a process: `--model`'s provider; else the `defaultProvider`
   setting's, when it has a credential and its listing did not refuse it; else the newest recent model whose provider
   has a credential and was not refused; else the first provider with a key, in the registry's order, with a note when
   several had one. A setting or a recent model passed by is warned of; with no credential at all the start is the
   setting's, so the error names its variable as before. `chosenProvider` is `--model`'s or the start's; the
   `defaultModel` setting is read only when its `defaultProvider` is the start; a recent start takes its model when the
   provider still offers it.
3. **The note** in the line session, before the first prompt: "Starting on OpenAI (openai), the first provider with a
   key; /model switches, and the defaultProvider setting or --model chooses where a run starts." Seen in a
   pseudo-terminal with two dummy keys and a dead proxy, after one listing warning per provider (below).
4. **`/model <ref> --default`** switches and saves `defaultProvider` and `defaultModel` through the agent's
   `SaveSetting`, saying "Saved fake/m2 as the default: new runs start on it."; `/model --default` saves the session's
   model. The list's footer names `--default`.
5. **Records:** deviation 4 above and MADR D3's note; a dated note in 0019-MADR D15 (a setting no longer fails a run
   another provider can start; the error stays for `--model` and for no credential at all); this record.

**Tests changed with it, none loosened:** `TestDefaultProviderSetting`'s last check, 0019-MADR D15's error for a
`defaultProvider` with no credential, now checks D3's fall-through: the run starts on the first provider with a key
and warns. It reads the model choice, not a `-p` run, which would now reach a model.

**Found by the gates and fixed in the phase:**

- **A data race in the new and changed tests** (macOS and Windows race runs, `race detected during execution of test`
  in `TestDefaultProviderSetting`): their stderr was a `bytes.Buffer` that a background listing's warning writes to;
  both now use the package's `syncBuffer`. Twenty race runs then passed.
- **A timing flaw in P1's `TestLastGoodOnDisk`** (macOS, once: `the listing was handed [] at 0001-01-01…`): it read
  what claude's listing was handed before that listing's goroutine had begun; it now waits for it. Two hundred runs,
  and fifty under the race detector, passed.
- The first round's Linux logs were removed when the tree was copied again for the second round, before they were read;
  its exit codes showed a failure in the plain test run as well as the race run. The rounds after it are the record.

**Found, not done here (a question for the owner):** since P1 lists every credentialed provider, each listing that
fails writes its own warning at start ("model listing failed; offering the curated static list (provider: …)"), one
per provider, where there was one; offline, with eight keys, eight lines. 0020-MADR D6 asks for one warning per failed
listing. The states are also in `/model` and `/providers` (P4).

**Gates (the last round):**

- the pre-add check on the twelve changed `.go` files, `12 file(s) clean`, then on the three test files changed after
  it, clean; lint for linux, darwin and windows, `0 issues.` each, after golint's and revive's `else` after a return,
  and modernize's `slices.Contains` and `slices.Backward` in a test;
- macOS: the tests and the race detector, 35 packages ok each; `make preflight`, `preflight passed`;
- the Linux machine and the Windows laptop, over ssh with `-o BatchMode=yes -o ConnectTimeout=5`, on a copy of the
  working tree, Windows with `TMP` and `TEMP` outside the user's home: the tests and the race detector, 35 packages ok
  on each; `make preflight` through `apidiff` (`0 issues.`), stopping at `verify-build-metadata` on a copy; `make
  check-records markdownlint`, "0 issues in 0 files".

**Not done in P3, as the plan says:** browsing (P4), `/login` (P5), ChatGPT (P6), cycling (P7).
