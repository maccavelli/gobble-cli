---
status: proposed
date: 2026-10-18
associated-madr: "0022-MADR-gobble-voice-and-personalities.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement gobble's voice: four personalities, a natural-language catalog, and overrides for every text

Associated MADR: [0022-MADR-gobble-voice-and-personalities.md](0022-MADR-gobble-voice-and-personalities.md)

*Proposed: not approved for execution. Each phase is made executable on the owner's word, with its facts measured then, as 0005-PLAN's sub-phases are.*

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
4. **`acpserver`:** the session's personality (from `SessionSettings`) reaches the prompt and the catalog; `dry` when the settings give none.
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
6. **User-made personalities:** `<config>/personalities/<name>/personality.md` and `messages.md`, validated as P5's overrides are; listed by `/attsonality`.
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

None yet.
