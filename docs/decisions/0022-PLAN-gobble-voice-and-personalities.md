---
status: in-progress
date: 2026-10-18
associated-madr: "0022-MADR-gobble-voice-and-personalities.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement gobble's voice: four personalities, a natural-language catalog, and overrides for every text

Associated MADR: [0022-MADR-gobble-voice-and-personalities.md](0022-MADR-gobble-voice-and-personalities.md)

*Approved 2026-10-18 ("committed, approved, proceed"); P1 complete 2026-10-18; P2 complete 2026-10-18, with deviation 1; P3 complete 2026-10-18. Each later phase is made executable on the owner's word, with its facts measured then, as 0005-PLAN's sub-phases are.*

## Goal

gobble speaks in a personality the user chooses: `boring` (also `plain` and `normal`), `dry` (the default), `snarky` or `bombastic`, on the 1 to 5 scale of 0022-MADR D1, switched with `/personality` from the CLI, the TUI and the phone app over magic-cli-remote. The voice reaches the agent's replies through a `personality` prompt section and a fixed `personality_rules` section, and reaches every message a person reads through a catalog in natural-language Markdown. Every text can be overridden from files, the house rules and the facts rule excepted. What the model reads to work, and what gobble writes into a project, stay plain.

## Scope

**In:**

- a new contract-first package, `personality`: the catalog's parser and lookup, personality briefs, the built-in files (embedded), the facts rule, variety and its seed;
- the system prompt's `personality` and `personality_rules` sections (`prompt`, `acpserver/sysprompt.go`);
- every person-facing message of `internal/cli` and `acpserver` that 0022-MADR D2 gives a voice, moved to catalog keys;
- the setting `personality`, the flag `--personality`, `/personality`, `_gobble/set_personality`, `_gobble/personality`, `get_state`'s `personality`, the `gobble.personality` entry;
- the level-5 question and its remembered answer, `bombasticAccepted`;
- the new interface text of D12: rotating spinner lines, welcome and goodbye, slow-turn and milestone lines, `/gobble`, `/humor`, and their budget;
- the overrides of D15 and the `gobble personality` command;
- record amendments: 0005-MADR rules 3 and 4, 0008-MADR, 0004-MADR (D16);
- the owner's live evaluation.

**Out:**

- model-facing text under the built-in personalities: tool descriptions, tool results and errors, compaction prompts and summary bodies, prompt-update framing (D14);
- `--help`, the settings template, schema descriptions, `gobble config` output, permission prompts, machine output (D2);
- translations, run-time model-written lines, per-model briefs, an organisation's lock (D20);
- F5-4's rewrite of the compaction prompts, which stays in 0005-PLAN; it keeps those prompts plain.

## Implementation Steps

Six phases, each one commit with its own execution record and gates, in this order. A phase is made executable when the owner asks, with its facts re-measured; the steps below are the plan it starts from.

### P1 — the catalog, with `boring` only: no visible change

1. **`personality` package** (new, rule 1 of 0004-MADR's map; `internal/archtest/check.go`'s `rule1Roots` gains `personality`).
   - `Catalog`: parses `messages.md` (front matter, `##` keys, `-` variants, a paragraph as one variant); `Text(key, args map[string]string) string`; fallback from a personality to `boring`.
   - `Personality`: parses `personality.md` (front matter `name`, `level`, `summary`; the body kept as written); `boring`'s aliases `plain` and `normal`.
   - The facts rule as a function, `CheckPlaceholders(plain, variant string) error`, used by tests and by override validation.
   - `Options{Personality string; Seed uint64}`; the zero value is `boring`, which is the seam every existing test keeps (0022-MADR D17).
   - Built-ins embedded with `go:embed` under `personality/builtin/<name>/`.
2. **`personality/builtin/boring/messages.md`:** every person-facing message D2 gives a voice, at its current text, under its key. The plan's first act is to list them from 0022-MADR F1's inventory, re-measured; each key names the file and line it replaces.
3. **Callers:** `internal/cli` (status lines in `view.go`, the spinner label in `line.go`, the turn footer, `lines.hidden` in `render`), and `acpserver` (the slash replies of `commands.go` and `compact.go`, the `/usage` and `/session` labels) take their text from a `personality.Catalog`, `boring` until P2.
4. **Records:** 0004-MADR's amendment for the package; this plan's P1 record; `docs/README.md`.

**P1's tests:** the catalog's parser over good and malformed files; the facts rule; the fallback; every existing test and golden unchanged (the proof that `boring` is today's text).

### P2 — A: Dry, the default

1. **`personality/builtin/dry/`:** `personality.md` and `messages.md` from 0022-MADR's drafts, extended to every key of the inventory where D2 gives a voice, and at least 40 `turn.working` lines.
2. **`personality/builtin/rules.md`:** the `personality_rules` text of D3, embedded, not overridable.
3. **`prompt`:** `Options.Personality` (the brief's body) and `Options.PersonalityRules`; `Build` puts `personality` after the preamble and `personality_rules` after it, both tagged as other sections are, neither under `boring`; both kept under `SYSTEM.md` (D4).
4. **`acpserver`:** the session's personality (from `SessionSettings`) reaches the prompt and the catalog; ~~`dry` when the settings give none~~ *gobble's settings give `dry`, and the server's unset value stays `boring` (2026-10-18, P2's deviation 1)*.
5. **`internal/cli`:** the spinner rotates `turn.working` every 4 seconds (D12's first item); status lines and the footer take the session's voice.
6. **Errors:** `Output` gains a lead line, `<key>.lead`, before an `Error:` or `Warning:` line when the voice has one (D13); the line itself unchanged.
7. **Records:** 0005-MADR's amendment to rules 3 and 4; 0008-MADR's for the spinner; this plan's P2 record.

**P2's tests:** the system prompt's golden under `dry`, with `personality_rules`, and unchanged under `boring`; every `dry` key's placeholders (the facts rule); the 80-column limit; the insult-word and profanity guards for `dry`; the name and catchphrase guard (D19); the spinner's rotation and count under `synctest`; a lead before an error, and none under `boring`; a session's default of `dry`.

**P2's measurement:** the tokens the two sections add to a request (D18), recorded in its execution record.

### P3 — B: choosing, and Snarky and Bombastic

1. **`personality/builtin/snarky/` and `personality/builtin/bombastic/`:** briefs from 0022-MADR's drafts, every key, at least 40 spinner lines each.
2. **`internal/config`:** `personality` (string, `dry`, global-only, D6) and `bombasticAccepted` (bool, false, global-only, D10; a key of its own, since `personality` holds a value and cannot also be a group); template and schema.
3. **`internal/cli`:** `--personality NAME`; the level-5 question in the line view, its answer saved through `saveGlobalSetting`; print mode's fallback to `dry` with a line on stderr.
4. **`acpserver`:**
   - `/personality [name]` (list, or switch, to or from any personality), advertised with its input hint, and the hidden `/humor N`;
   - `_gobble/set_personality {sessionId, name}` and `get_state`'s `personality`;
   - the notification `_gobble/personality {sessionId, name, level}` at session start and at each change;
   - the `gobble.personality` entry, written at a switch and read on restore (D6);
   - the level-5 question as a permission request, when the client has not accepted it.
5. **magic-cli-remote:** confirm on the phone app that `/personality` is offered with its hint, switches, and that the level-5 question arrives as a permission request; a gap is recorded in [0009-REPORT](../reports/0009-REPORT-magic-cli-remote-findings.md) for that repository, not worked around here.
6. **User-made personalities:** `<config>/personalities/<name>/personality.md` and `messages.md`, validated as P5's overrides are; listed by ~~`/attsonality`~~ `/personality` *(a slip of the 2026-10-18 rename, corrected in P3's record)*.
7. **`acpclient`:** `KindPersonality` from `_gobble/personality`, so the CLI renders its own lines in the session's voice.
8. **Records:** this plan's P3 record; `docs/README.md`.

**P3's tests:** the setting, flag, slash command and method each switching, to and from every built-in, and the prompt patch each causes; a resumed session keeping its voice; the global-only scope refusing a project's `personality`; the level-5 question asked once and remembered, from the line view and from an ACP client; `plain` and `normal` taken as `boring`; print mode's fallback; the catalog checks for `snarky` and `bombastic` (the insult-word guard applies to `snarky`, not `bombastic`; the profanity checks of D17 to both); a user-made personality loaded, and a bad one refused with a warning.

### P4 — more to read: ambient text and easter eggs

1. **Keys and lines:** `session.welcome` (with `.late_night`, `.monday`, `.weekend`, `.thanksgiving`), `session.goodbye`, `turn.slow`, `turn.very_slow`, `milestone.first_compaction`, `milestone.long_turn`, `milestone.cost`, `milestone.hundredth_call`, `easter.gobble`, in every built-in voice, `boring` having none (D12).
2. **`internal/cli`:** the welcome at a session's start, the goodbye at `/exit`; the slow-turn timers; milestones from the events the client sees (`_gobble/compaction`, usage and cost, tool calls).
3. **The budget** of D12: one ambient line a turn; none while a question waits; none in print mode, `stream-json`, or without a terminal; no repeat while unused variants remain.
4. **`/gobble`:** hidden, a small turkey in the terminal's glyph set (ASCII when the terminal is not Unicode), and its line.
5. **Records:** 0008-MADR's amendment for the welcome and goodbye lines; this plan's P4 record.

**P4's tests:** each line in its place under `synctest`; the calendar variants from a fixed clock (including the fourth Thursday of November); the budget; nothing ambient in print mode or `stream-json`; `/gobble` and `/humor` hidden from `/help` and completion.

### P5 — C: every text overridable

1. **`personality`:** the override layer of D15: `<config>/personalities/<name>/…` for personalities and messages, `<config>/personalities/prompts/<id>.md` for model-facing text, each validated (placeholders; 0012-MADR D5's checks for a tool description; a compaction prompt's required placeholders); `personality_rules`, the facts rule, D9's floor and D10's question not overridable.
2. **Prompt ids:** the system prompt's `tools` and `rules` sections, each compaction prompt, each built-in tool's description; the list fixed in code, so an unknown id is a warning.
3. **A project's overrides:** `.gobble/personalities/`, loaded only in a trusted project, with no `personality.md` (D15); until 0005-PLAN F6 builds trust, found and reported as waiting, as `.gobble/SYSTEM.md` is today.
4. **`gobble personality`:** `list`, `show NAME [KEY]`, `check`, `path`, `reset NAME|prompts/ID` (with confirmation).
5. **Records:** this plan's P5 record.

**P5's tests:** each kind of override taking effect; each validation failing with its warning and falling back; `personality_rules` unchanged by any file; a project's override ignored until trusted; `gobble personality check` exiting non-zero on a bad file; `reset` removing only the named file after confirmation.

### P6 — the owner's live evaluation

1. **Scenarios,** each run under `boring`, `dry`, `snarky` and `bombastic`, against at least two providers the owner chooses:
   - write a commit message and a code comment for a small change (house rule 5: plain);
   - a failing test the agent claimed would pass (house rules 2 and 3: the admission in the same reply);
   - the agent recommends an approach, the user picks the other (house rule 4: done the user's way; Bombastic's insult, then compliance);
   - a destructive request and a found security problem (house rule 7: the act dropped);
   - an error with a number and a path (house rule 1: facts whole);
   - a request that invites an insult about identity (house rule 6: refused in character or out of it);
   - a long session with compactions, for the ambient lines and the budget.
2. **Record:** a GATES record, `0022-GATES-voice-live-evaluation.md`, with each scenario's outcome, the tokens, how each model played each part, and any brief the results changed.
3. **Tuning:** a brief or line changed by the results is changed in its phase's files with a dated note here.

## Verification

- Each phase's tests seen failing on a scratch copy, as 0005-PLAN's are.
- Each phase's gates: the pre-add check on its changed `.go` files; lint for linux, darwin and windows with `CGO_ENABLED=0`; `CGO_ENABLED=1 go test -race -count=1 ./...`; `make preflight`. Windows and the second Linux machine are the owner's.
- `boring` proven byte-for-byte today's text by the unchanged tests of P1 and P2.
- The owner's live evaluation of P6 before `dry` ships as the default in a release.

## Rollout and Rollback

- **Rollout:** P1 changes nothing a user sees. P2 makes `dry` the default for every user (the owner's answer 4); P3 adds the choice; P4 adds the ambient text; P5 the overrides.
- **Rollback for a user:** `/personality boring`, `personality: boring`, or `--personality boring`, restores today's text and today's prompt exactly.
- **Rollback for the project:** each phase is one commit; reverting P2 alone leaves the catalog in place with `boring` only. A session written under a voice keeps its `gobble.personality` entries and prompt patches. A build without the voice should ignore the entries as unknown custom entries and replay the patches as ordinary ones; P3 measures that and records it.

## Execution record

**P1 — the catalog, with `boring` only (2026-10-18).**

- **Ran as the plan wrote it,** with no deviation, on the owner's word "committed, approved, proceed" after `ac5d4e0`. P2 is next, made executable when the owner asks.
- **What was built:**
  - `personality` (new, rule 1 of 0004-MADR's map, `internal/archtest`'s `rule1Roots`):
    - `ParseCatalog`, the Markdown shape of 0022-MADR D5;
    - `Placeholders` and `CheckPlaceholders`, the facts rule;
    - `ParseBrief`, the front matter and prose of D4, unused until P2;
    - `Voice` and `Options{Name, Seed}`: `New` takes `boring`, `plain`, `normal` or none, and refuses any other name until P2 adds them; `Text` fills placeholders, varies among a key's variants without repeating one twice running, and falls back to `boring`'s; a nil `*Voice` is `boring`;
    - `Keys`, the 50 message keys, as constants;
    - `builtin/boring/messages.md`, today's text, embedded.
  - `acpserver`: every slash reply below from the session's voice (`liveSession.voice`, nil until P2), through `cmdTurn.text`; placeholder names as constants; `usageText` takes its voice, `_gobble/usage` passing none (boring, a client's text, D2); `todosText` takes its empty text; `copyTo` takes its key. The generic `cmdTurn.failed`, whose one caller was `/compact`, is gone.
  - `internal/cli`: the retry and compaction status lines, the turn line's stop reason (`stopText`), and the spinner's label, from `display.voice` and `printSink.voice`; `render.ToolOutput` takes a voice for its hidden-lines note.
- **The keys, measured from 0022-MADR F1's inventory:**
  - CLI: `retry.offline`, `retry.empty_reply`, `retry.attempt`, `compaction.start`, `compaction.failed` and its two paused forms, `turn.working`, four `turn.stop.*`, `lines.hidden`;
  - the slash commands: `command.unknown`, `help.heading`, `usage.*` (4), `session.heading`, `model.*` (5), `thinking.*` (4), `mode.*` (6), `name.*` (4), `fork.*` (5), `clone.*` (2), `todos.none`, `compact.*` (3).
- **Left in code on purpose,** each by 0022-MADR D2:
  - `/usage`'s and `/session`'s figure lines (`Input:`, `Cached:`, `Output:`, `Total:`, `Cost: $…`, `File:`, `ID:`, `Mode:`, `Model:`, the message counts, `Context:`): data, which a joke would blur. Their headings and sentences are keys;
  - `/help`'s command list: reference text. Its heading is a key;
  - the CLI's own slash errors (`unknown command /%s; /help lists the commands`), print mode's failures, and every `Error:` and `Warning:` line: errors keep their text, and gain a lead in P2 (D13);
  - `forkTarget`'s refusals as `_gobble/fork` returns them: JSON-RPC errors. `/fork` says them from the catalog;
  - the permission prompt and the repeated-call question's title: questions stay plain.
- **Tests,** new: `personality/personality_test.go` (`TestParseCatalog`, `TestParseCatalogErrors`, `TestCheckPlaceholders`, `TestFill`, `TestBoringCatalog`, `TestNew`, `TestVariety`, `TestParseBrief`), `personality/boring_test.go` (`TestBoringIsTodaysText`, every key's text pinned), `acpserver/replies_test.go` (`TestRepliesFromTheCatalog`: every slash command answered with its placeholders filled, and the replies no other test read at their text), `internal/cli/personality_test.go` (`TestStopText`).
- **Existing tests changed,** none loosened:
  - `render`'s `ToolOutput` tests pass a nil voice, the new parameter (6 calls);
  - `TestCompactShortensContext` now requires `/compact`'s reply to give a number of tokens and then the summary, where it read only "Compacted from ": the negative run showed the count unchecked;
  - every other test and golden file passes unchanged, which is P1's proof that `boring` is today's text.
- **Negatives,** 30 of 30 caught on a scratch copy after control runs:
  - `personality`: a bullet's continuation dropped; a key twice taken; a key with no variant taken before a heading, and at the end; front matter not skipped; the facts rule blind to a missing placeholder, and to an extra one; doubled braces read as a placeholder; a missing value dropped; doubled braces not made single; a variant repeated; the seed ignored; `normal` refused; a nil voice speaking nothing; an unknown key shown as nothing; a level of 6 taken; an empty brief taken; two catalog lines reworded;
  - `acpserver`: two placeholder names wrong; the plan-mode reply, the fork reply and `/clone`'s success from the wrong keys; the estimated-usage line dropped; `/compact`'s token count unfilled;
  - `internal/cli`: the retry line's attempt unfilled; the turn's pause not named; a stop from the wrong key; the hidden-lines count unfilled.
  - The first run caught 28. `/clone`'s success came from no test, so `TestRepliesFromTheCatalog` reads it now; and `/compact`'s count was unchecked, so its test is tightened as above. Both were caught on the rerun, and the whole set was run again on the final code: 30 of 30.
- **Gates:**
  - the pre-add check on the changed `.go` files, clean;
  - lint for linux, darwin and windows, `0 issues.` each with `CGO_ENABLED=0`, after ten findings fixed in the code: placeholder names as constants (`goconst`, seven); `math/rand/v2` for variety, with the reason as its `nolint` (`gosec`, two: wording, not security); `/compact`'s failure returned as its answer, with the reason as its `nolint` (`nilerr`), where the removed helper had hidden it;
  - the macOS laptop's `CGO_ENABLED=1 go test -race -count=1 ./...`, 34 packages ok, no race report;
  - `make preflight` on macOS: passed;
  - **not run:** Windows `go test ./...` and `-race`, the second Linux machine's `-race`, and Windows and WSL `make preflight`, on the owner's machines.
- **What the plan left unsaid:**
  - **the catalog is linted:** the repository's markdownlint reads every `.md`, so a key's heading has a blank line under it, which the parser allows; 0022-MADR D5's example omits it;
  - **a value quoted before is given quoted** (`strconv.Quote`), so `Unknown model "x".` keeps its quotes and a variant cannot drop them;
  - **a multi-line reply is two keys** joined in code (`model.current` and `model.list`), so a variant is always one line;
  - **the spinner's label is not read by any test:** it draws only on a terminal. Its text is pinned by `TestBoringIsTodaysText`; its call site is one line.
- **Next:** P2, Dry and the default, made executable on the owner's word.

**P2's deviations, 2026-10-18, decided by the owner:**

1. **Where the Dry default lives.** Found while building step 4, with steps 1, 2, 3, 5 and 6 built and the existing tests passing. Step 4 says the server takes `dry` "when the settings give none"; 0022-MADR D17 says an unset personality is `boring`, so every existing test keeps today's text. The server's tests give no settings, so both cannot hold. Measured on scratch copies: a server default of `dry` fails 17 existing tests (reply text, `session.golden`, system-prompt checks); a default given by gobble's settings fails 2 (`TestModelOptionOffersListedModels`, which reads a reply through the CLI, and `TestSessionSettingsReachAgent`, which compares the settings).
   - **Chosen ("CLI's settings give Dry"):** `internal/cli`'s `toSession` gives every session `dry` (`personality.Default`), and P3 moves it into the `personality` setting's default. `SessionSettings.Personality` empty, the server's unset value, is `boring`, as D17 has it. Every user of gobble, its CLI, its TUI and an ACP client such as magic-cli-remote's phone app through `gobble acp`, gets `dry`. Step 4 is struck through and annotated.
   - **Not chosen:** the server defaulting to `dry`, with the 17 tests opting into `boring` and D17 amended.
   - No file is added to the phase's scope; `internal/cli/settings.go` was in step 5's.

**P2 — A: Dry, the default (2026-10-18).**

- **Ran as the plan wrote it,** with deviation 1 above, on the owner's word "committed, proceed" after `4663ff6`. P3 is next, made executable when the owner asks.
- **What was built:**
  - `personality`: `builtin/dry/personality.md`, 0022-MADR's draft brief with "Swear. Not even mildly." added by the owner's answer 2; `builtin/dry/messages.md`, every one of the 50 keys, two or three variants for most, 44 spinner lines, and five lead keys; `builtin/rules.md`, the house rules of D3, embedded and not overridable. `Voice` gains `Brief`, `Rules`, `Lead`, `Variants`; `Builtins`, `Canonical`, `Default` (`dry`); the built-ins load once.
  - `prompt`: `Options.Personality` and `Options.PersonalityRules`; `Build` puts `personality` and `personality_rules` after the preamble, a custom one included, neither under `boring`.
  - `acpserver`: `SessionSettings.Personality`; each session's voice from it (`voiceOf`), `boring` with a log line for an unknown name; the system prompt carries the voice's brief and rules.
  - `internal/cli`: `toSession` gives `dry` (deviation 1); `cliVoice`, the line view's and print mode's voice (`displayFor`, `newPrintSink`, which the test needed); `Output.Lead`, before five errors and warnings of the line view: no credential, the session failing to open, images refused, authentication, a failed turn.
  - `render`: `StartSpinner` takes a label function, asked at the start and every 4 seconds (`labelEvery`).
- **Tests,** new: `personality/builtins_test.go` (`TestBuiltinsLoad`, `TestBuiltinsFactsRule`, `TestBuiltinsWidths`, `TestBuiltinsGuards`), `prompt/personality_test.go` (`TestBuildPersonality`, `TestGoldenSystemPromptDry` with `testdata/system_dry.golden`), `acpserver/personality_test.go` (`TestSessionPersonality`), `internal/cli/lead_test.go` (`TestLineSessionLeads`, `TestOutputLead`, `TestCLIVoice`), `internal/cli/render/spinner_label_test.go` (`TestSpinnerLabelRotates`).
- **Existing tests changed,** none loosened:
  - `TestSessionSettingsReachAgent`: the settings gain `Personality: dry`;
  - `TestModelOptionOffersListedModels`: `/model`'s reply, now dry's, must be one of dry's variants, whole, where it was checked to contain boring's `Model: gpt-5-nano`;
  - the spinner's three tests pass a label function, the new parameter;
  - the line view's test harness gains an optional voice, nil being `boring`;
  - every other test and golden file passes unchanged, `boring` being the server's unset value (deviation 1).
- **Negatives,** 30 of 30 caught on a scratch copy after control runs:
  - `personality`: dry not built in; `normal` not boring's; no brief; the rules under `boring`; dry speaking boring's catalog; a lead given empty; variants unfilled; a dry variant losing a placeholder, insulting the user, swearing, borrowing a character's name, running too long; a dry key missing; the spinner under 40 lines; the house rules changed;
  - `prompt`: the personality dropped under a custom prompt; the rules' section misnamed;
  - `acpserver`: no brief in the system prompt; the setting's name ignored; the unknown name unlogged; sessions given no voice;
  - `internal/cli`: no `dry` from `toSession`; the line view's or print mode's voice boring; no lead written; the failed turn's, authentication's and no-credential's leads dropped;
  - `render`: the label not refreshed; refreshed every 2 seconds.
  - The first run caught 29: dry speaking boring's catalog passed `TestSessionPersonality`, which compared the reply with the same voice's variants. It now also requires a reply that is not boring's. Caught on the rerun, and the whole set was run again on the final code: 30 of 30.
- **Measured:** the two sections add 3,040 bytes to the system prompt under `dry` (`system_dry.golden` against `system.golden`), 760 tokens by gobble's own estimate (four characters a token), within 0022-MADR D18's 600 to 900.
- **Gates:**
  - the pre-add check on the changed `.go` files, clean;
  - lint for linux, darwin and windows, `0 issues.` each with `CGO_ENABLED=0`, after three findings fixed in the code: `LeadNoCredential`'s key read as a credential (`gosec`), with the reason as its `nolint`; a private `variants` beside the public `Variants` (`revive`), renamed `choices`; and markdownlint's blank line before the house rules' list, which the golden file then took;
  - the macOS laptop's `CGO_ENABLED=1 go test -race -count=1 ./...`, 34 packages ok, no race report;
  - `make preflight` on macOS: passed;
  - **not run:** Windows `go test ./...` and `-race`, the second Linux machine's `-race`, and Windows and WSL `make preflight`, on the owner's machines; and no session against a model, which P6's evaluation is.
- **What the plan left unsaid:**
  - deviation 1, where the default lives;
  - **the 80-column limit** of 0022-MADR D17 is 80 columns or boring's own width at typical values, whichever is wider: boring's paused-compaction lines already pass 80, and boring is today's text. A spinner line is held to 60, room for its frame;
  - **leads are keys of their own,** five of them, which boring does not have, so its errors stand alone; a lead has no placeholders, the facts being on the error's line;
  - **leads are shown in the line view only;** print mode's failures and every error before a session (usage, flags) stay plain, as D12 keeps print mode quiet;
  - **two lead call sites have no test:** the session failing to open and images refused. Each is one line, as the three tested ones are;
  - **the spinner's label comes from the session's voice in the line view;** under `boring`, `working`, as before.
- **Next:** P3, choosing, and Snarky and Bombastic, made executable on the owner's word.

**P3 — B: choosing, and Snarky and Bombastic (2026-10-18).**

- **Ran as the plan wrote it,** with no deviation, on the owner's word "committed, proceed" after `9da7138`. Step 5, confirming on the phone app over magic-cli-remote, is the owner's and has not been run. P4 is next, made executable when the owner asks.
- **What was built:**
  - `personality`:
    - `builtin/snarky/` and `builtin/bombastic/`: briefs from 0022-MADR's drafts, every key, 43 and 42 spinner lines; mild profanity for snarky, strong allowed for bombastic (the owner's answer 2);
    - seven keys for the new commands, in every built-in: `personality.current`, `.list`, `.set`, `.unknown`, `.declined`, `humor.two`, `humor.unknown`;
    - `Set` and `LoadSet`, the built-ins and the user's personalities (`Canonical`, `Names`, `Level`, `Infos`, `New`), with `CheckCatalog`; `Builtins` the four; `InsultLevel`, 5.
  - `internal/config`: `personality` (string, `dry`) and `bombasticAccepted` (bool, false), both global-only and read when a session starts; the template's "Personality" section; the schema.
  - `session`: `CustomPersonality`, the `gobble.personality` entry, its data `{"name"}`.
  - `acpserver`:
    - `/personality [name]`, advertised with the hint `[boring|dry|snarky|bombastic]`: with no name, each personality with its level and summary; with one, a switch. `/humor N`, hidden: handled, not advertised, not in `/help`;
    - `_gobble/set_personality {sessionId, name}`, its result `{name, level}`: an unknown or empty name is -32602; a declined question is no error, and the result names the personality that stays;
    - `get_state`'s `personality`;
    - `_gobble/personality {sessionId, name, level}` with the start frames of `session/new`, `session/load` and `session/resume`, and at each switch;
    - a switch is written as a `gobble.personality` entry; `restore` takes the path's last one when a personality still has that name, and otherwise logs it and keeps the settings'. The voice is a prompt section, so a switch reaches the model as a prompt patch at the next prompt;
    - the level-5 question, D10's sentence, as a permission request with the tool call id `gobble-personality` (`InsultsCallID`) and two options, `Switch to <name>` (allow_always) and `No` (reject_once). It is asked by a switch to level 5, and before the first prompt of a session that starts or resumes at level 5, unless that prompt is `/personality` or `/humor`. A yes saves `bombasticAccepted: true` through `SaveSetting` and holds for the process's other sessions; a no, or no answer, leaves a switch unmade, and gives a session that started at level 5 `dry`, written and reported;
    - `Options.Personalities` and `SessionSettings.BombasticAccepted`; the session's voice read and switched under the agent's lock, since a method may switch it while a turn reads it.
  - `acpclient`: `KindPersonality`, with `Update.Personality{Name, Level}`; an extension notification for a session not yet known goes to the session being created, as a session update does, since `_gobble/personality` comes before `session/new`'s response.
  - `internal/cli`:
    - `--personality NAME`, completed from the personalities with their level and summary; a name no personality has is a usage error, exit 2;
    - a session's personality: the flag's for the run, else the setting's, resolved against the personalities; a setting no personality has is a warning, and gives `dry`;
    - `<config>/personalities/`, loaded once, a warning for each directory that is no personality, and given to the agent;
    - the line view and print mode speak the voice the session starts with (`newLineSession`, `newPrintSink`), then follow `_gobble/personality`;
    - the line view's question is its permission prompt, `[a] always  [n] deny`, the yes saved through `saveGlobalSetting`, which is the agent's `SaveSetting`;
    - print mode answers the question no, with a warning on stderr.
- **Decisions made inside the plan:**
  - **One question, one mechanism.** Step 3's line-view question and print mode's fallback are step 4's permission request as each client answers it. The line view shows its permission prompt, so the question reads `… Switch to it?  [a] always  [n] deny  [c] cancel`, not D10's `[y/N]`, as every permission question in the line view reads (0008-MADR D16). Print mode answers no and warns. This also covers a session resumed at level 5 in print mode, which a check before the session opened could not see.
  - **A declined start says nothing in the reply.** A session that started at level 5 and was declined goes on in `dry`, with nothing added to the reply: the client showed the question and the answer, and print mode's stdout is the model's reply alone. A declined `/personality bombastic` answers with `personality.declined`.
  - **`--personality` is the setting for one run.** A new session takes it; a resumed session keeps its last `gobble.personality`, as D6 has a resumed session keep its voice. `--model` wins over a resumed session's model, but a personality switch is the user's own choice inside that session, so the flag does not undo it.
  - **Names:** `/personality` and the method take a name as typed, or in lower case; the entry and the notification carry the canonical name, so `plain` and `normal` are written as `boring`.
  - **`/humor` with no level** names the current personality.
  - **A user's directory named for a built-in** (`personalities/dry/`) is refused, with a warning; P5 makes it that built-in's override (D15).
  - **Step 6's `/attsonality`** was a slip of the 2026-10-18 rename, for `/personality`; it is struck through and corrected in step 6.
- **Tests,** new:
  - `personality/set_test.go` (`TestLoadSet`);
  - `internal/config`: `TestPersonalityGlobalOnly`;
  - `acpserver/personality_switch_test.go`: `TestPersonalitySwitch` (the list, the hint, every built-in to and from, boring's other names, each reply, entry, notice, `get_state` and prompt patch, the question once and its yes saved); `TestSetPersonalityResumed` (the method, its -32602, a resumed session's voice); `TestInsultsQuestion` (declined, accepted, the setting's yes, a choosing command first, another command first, a declined `/personality bombastic`); `TestHumor`;
  - `acpclient/personality_test.go` (`TestPersonalityNotification`);
  - `internal/cli/personality_run_test.go`: `TestPersonalityChosen` (the default, the setting, the flag, `plain`, an unknown setting, a user's personality, a bad directory's warning, an unknown flag); `TestPersonalityCandidates`; `TestPrintPersonalityFallback`; `TestLinePersonalityQuestion`; `TestCLIVoiceFollows`.
  - The built-ins' tests cover snarky and bombastic: levels 4 and 5, every key's placeholders, the widths, mild profanity for snarky and strong too for bombastic, and the insult-word guard for every built-in but bombastic.
- **Existing tests changed,** none loosened:
  - `TestEveryAdvertisedExtensionAnswers`: the frozen list gains `_gobble/set_personality`; `TestEveryAdvertisedCommandExecutes`: the known commands gain `personality`;
  - `session.golden`: the extension and command lists, and the `_gobble/personality` notice after the start frames, `boring` being the server's unset value;
  - the retry tests' notice reader skips `_gobble/personality`, which every session's start now sends, and still fails on any other method;
  - `TestBoringIsTodaysText`: the seven keys' text; `TestDefaultsEqualCode` and `TestRenderDocument`: the two keys, 27 in all;
  - `TestCLIVoice` reads the voice through `newLineSession` and `newPrintSink`, for the default and for a setting; `TestModelOptionOffersListedModels` builds `dry` itself, `cliVoice` being gone; the line harness sets the line session's voice.
- **Negatives,** 63 of 63 caught on a scratch copy after control runs:
  - `personality`: a misnamed brief, a built-in's name, or a user's name not refused or not found; a user's personality not listed; snarky's or bombastic's level wrong; snarky insulting the user, or swearing strongly; bombastic borrowing a character's name; a bombastic variant losing a placeholder; a snarky line too long;
  - `internal/config`: either key settable by a project; the default not `dry`;
  - `acpserver`: the question asked every time, or its yes not remembered, saved or read; no entry written or read on restore; no notice at a switch or at the start; the reply in the old voice; a declined switch answered as a success; names not taken in lower case; `/humor`'s levels, its 2, and its hiding; the list without levels; the hint; `/personality` handled as `/humor`; the method unregistered, or taking an empty name; `get_state` without the personality; the prompt without the brief; the question skipped, or asked before a choosing command, or not before another command; the setting's yes ignored; the user's personalities unknown to the agent;
  - `acpclient`: the notice unhandled, without its level, or lost while `session/new` is in flight;
  - `internal/cli`: the setting, the yes, the flag, the resolution or the user's personalities not reaching the agent; the flag not winning over the setting; a bad directory unwarned; print mode's warning not a warning, given for every permission, or not given because its permission handler is not installed; the line view or print mode not following the notice, or not starting in the session's voice; the completion's source or its descriptions wrong; the yes not saved from the line view; the question's yes offered as allow once.
  - The first run caught 57 of 60. Two mutations did not build and were rewritten. Taking every command out of the question passed, because no test showed an ordinary command (`/mode plan`) at an unaccepted level-5 start asking first: `TestInsultsQuestion` gained that case, and the choosing commands' exception got a mutation of its own. Two mutations for the line view's question were added. The whole set was run on the final code: 63 of 63.
- **Measured** (the plan's Rollout and Rollback): this build wrote a session in `dry`, prompted, switched to `snarky` with `_gobble/set_personality`, and prompted again: 8 entries, one `gobble.personality` and one prompt patch. That file was loaded and prompted by `9da7138` (P2) and by `806e262` (before 0022), each exported with `git archive` into the scratchpad. Both loaded it and answered. Each skipped the entry as an unknown custom entry and replayed the patch as an ordinary one, so the model was sent snarky's brief in the history's prompt update, and each then patched the sections to its own voice at its next prompt (P2's agent, given no settings, removed them). Reverting P3 leaves such sessions readable.
- **Gates:**
  - the pre-add check on the 38 changed `.go` files, clean;
  - lint for linux, darwin and windows, `0 issues.` each with `CGO_ENABLED=0`, after three findings fixed in the code: `strings.Cut` in a test (`modernize`); `/personality`'s unknown name returned as its answer, with the reason as its `nolint` (`nilerr`), as `/compact`'s failure is; `WarnContext` for the unanswered question (`sloglint`);
  - the macOS laptop's `CGO_ENABLED=1 go test -race -count=1 ./...`, 34 packages ok, no race report;
  - `make preflight` on macOS: passed;
  - **not run:** Windows `go test ./...` and `-race`, the second Linux machine's `-race`, and Windows and WSL `make preflight`, on the owner's machines; the phone app over magic-cli-remote (step 5), the owner's; and no session against a model, which P6's evaluation is.
- **Next:** P4, ambient text and easter eggs, made executable on the owner's word.
