# gobble-cli documentation

`gobble-cli` is the repository for **gobble**, a coding agent written in Go that takes the best of several agent harnesses and makes it its own. This tree holds three kinds of document:

- reports that measured something;
- the decisions behind gobble's design;
- guides you follow to do a thing.

- **[architecture.md](architecture.md)** — how the pieces fit together, as they are now.
- **guides/** — task-shaped walkthroughs: [developing.md](guides/developing.md), on building, testing, the gates and adding a package;
  [models-json.md](guides/models-json.md), on pointing providers at a proxy and correcting models;
  [choosing-a-model.md](guides/choosing-a-model.md), on `--model`, `--thinking`, `defaultModel` and `gobble models`;
  [credentials.md](guides/credentials.md), on where keys come from, `auth.json` and `gobble auth`.
- **[decisions/](decisions/)** — numbered MADR/PLAN pairs.
- **[reports/](reports/)** — what was observed, deciding nothing.

## Records

| Number | Kind | Record | Status |
| :--- | :--- | :--- | :--- |
| 0001 | REPORT | [Go port feasibility of the Pi agent harness](reports/0001-REPORT-go-port-feasibility.md) | observation |
| 0002 | MADR | [v1 native magic-cli-remote CLI: Kong over ACP, ACP stdio, MCP client](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) | proposed (amended 2026-10-04 (0008); Phase 7's extension set 2026-10-07; config surface decided by 0018, note 2026-10-12) |
| 0002 | PLAN | [Implement the ACP core](decisions/0002-PLAN-cli-acp-headless-mcp-v1.md) | in progress (Phases 1–3 complete 2026-10-05, Phases 4–6 complete 2026-10-06, Phases 7–8 complete 2026-10-07 (CI green, run 37650274632); amended 2026-10-04 (0008), 2026-10-05, 2026-10-06, 2026-10-07 (Phases 7 and 8 made executable)) |
| 0003 | MADR | [The product is `gobble`: binary, module path, directories, wire names, read-only Pi bridge](decisions/0003-MADR-gobble-product-identity.md) | proposed (amended 2026-10-01; secrets row 2026-10-10; `settings.yaml` note 2026-10-12; Pi import note 2026-10-15) |
| 0004 | MADR | [One Go 1.27.1 module of contract-first packages, open standards at every boundary](decisions/0004-MADR-go-module-architecture.md) | proposed (decision text rewritten 2026-10-04; amended 2026-10-05; go-tui-lib note 2026-10-06; release pipeline amended 2026-10-07; toolchain go1.27.2 2026-10-08; settings are YAML, note 2026-10-12; 0021 note 2026-10-13; `personality` joins rule 1, 2026-10-18 (0022); `personality` beta and `internal/frontmatter` internal, 2026-10-19) |
| 0004 | PLAN | [Scaffold: module, contracts, import boundaries, toolchain, release](decisions/0004-PLAN-go-module-architecture.md) | in progress (Phase D, Phase 0, and Phase 1 complete; Phase 2 steps 2–4 complete 2026-10-05, and steps 1, 5 (in part), 6–9 and 11 through 0008-PLAN P2–P4; Phase 3 complete 2026-10-05; Phase 2 steps 10, 12, 13 not started; Phase 5 made executable and complete 2026-10-07; Phase 4 step 1 through 0002-PLAN Phase 8 2026-10-07; Phase 4 steps 2–4 made executable 2026-10-07 and complete 2026-10-07 (CI rehearsal green, run 37677754325); amended 2026-10-04 (0007); amended 2026-10-04 (0008); Phase 2 steps 2–4 amended 2026-10-05; fleet library pins refreshed 2026-10-06) |
| 0005 | MADR | [The v1 line: every portable Pi capability, tiered 1.0 / 1.x / exp](decisions/0005-MADR-v1-feature-scope.md) | accepted 2026-10-07 (decision text rewritten 2026-10-04; platform note 2026-10-07; reference-harness amendment 2026-10-07; native tools amendment 2026-10-08; 0012 amendment 2026-10-08; grep row's globs 2026-10-08; F3 amendment 2026-10-10; F4 amendment 2026-10-10; SDK `v1.7.0` amendment 2026-10-11; 0018 notes 2026-10-12; 0019 notes 2026-10-13; 0021 note 2026-10-13; subscription logins' terms checks 2026-10-14; Grok's login kept, second amendment 2026-10-14; 0016 note 2026-10-15; F5 notes on choices 7 and 9 2026-10-16; compaction from six harnesses, amendment 2026-10-18; F5-3 note on choice 7 2026-10-18; 0022 notes on rules 3 and 4, 2026-10-18; F5-5 notes on choice 11 and automatic titles, 2026-10-18; F6 notes on the Rules, Permission requests and Project trust rows, 2026-10-19; F6-2 notes on the Rules and Protected paths rows, 2026-10-19) |
| 0005 | PLAN | [Implement the v1.0.0 gate, the v1.x train, and `exp/`](decisions/0005-PLAN-v1-feature-scope.md) | in progress (approved 2026-10-07; decision text rewritten 2026-10-04; F4, F5, F8, F9 and F10 notes 2026-10-07; F10 step 6 `apidiff` note 2026-10-07; F1 made executable 2026-10-07; restructured into five sub-phases and F1a reworked 2026-10-07; F1a complete 2026-10-07; F1b complete 2026-10-07; native tools placed 2026-10-08; 0012 notes 2026-10-08; F1c made executable 2026-10-08; F1c-1 complete 2026-10-08; F1c-2 complete 2026-10-08; F1d made executable 2026-10-08; F1d-1 complete 2026-10-09; F1d-2 complete 2026-10-09; F1d-3 complete, and with it F1, 2026-10-09; F2 made executable 2026-10-09; F2-1 complete 2026-10-10; F2-2 complete, and with it F2, 2026-10-10; F3 made executable 2026-10-10; F3-1 complete 2026-10-10; F3-2 complete 2026-10-10; F3-3 complete 2026-10-10; F3-4 complete, and with it F3, 2026-10-10; F4 made executable 2026-10-10; F4-1 re-made for SDK `v1.7.0` 2026-10-11; F6 test file note 2026-10-12; F4-2 `defaultProvider` note 2026-10-13; F4-3 rejected-key note 2026-10-13; 0019 notes 2026-10-13; 0020 notes 2026-10-13; 0021 note 2026-10-13; F4-1a re-made at SDK `v1.12.0` 2026-10-13; F4-1a complete 2026-10-13; F4-2 re-made as three sub-phases, F4-2a and F4-2b complete 2026-10-13; F4-2c made executable and complete 2026-10-14, at SDK `v1.13.0`; F4-3 re-made as three sub-phases, F4-3a complete 2026-10-14; F4-3b's terms checks recorded and F4-3b made executable 2026-10-14, proposed; F4-3b re-made with Grok's login, and the ChatGPT login made F4-3d, 2026-10-14; F4-3b complete 2026-10-14, with the owner's live Grok login; F4-3c made executable as four sub-phases 2026-10-15; F4-3c-1 complete 2026-10-15, acp-go-sdk2 `v1.14.0`; F4-3c-2 complete 2026-10-15, ACP authentication; F4-3c-3 complete 2026-10-15, `--json` logins; F4-3c-4 complete, and with it F4-3c, 2026-10-15, the Pi import; F4-3c-4's note on a provider that takes no key corrected 2026-10-16; F5 re-made as five sub-phases, started before F4 is complete, and F5-1 made executable 2026-10-16; F5-1 complete 2026-10-16, retry gaps, deviation 1; F5-2 made executable 2026-10-17, proposed, and amended before approval 2026-10-18; F5-4 refined 2026-10-18; F5-2 complete 2026-10-18, automatic compaction, deviation 1; F5-3 made executable 2026-10-18, proposed; F5-3 complete 2026-10-18, loop safety; F5-4 item 8 added 2026-10-18, its prompts overridable (0022-PLAN P5); F5-4 complete 2026-10-18, the summary; F5-5 complete 2026-10-18, the small model's titles; F6 re-made as five sub-phases, F6-1 made executable 2026-10-19; F6-1 complete 2026-10-19, project trust; the docs brought up to date 2026-10-19; F6-2 made executable and approved 2026-10-19, after 0023) |
| 0005 | REPORT | [Compaction across six harnesses: what gobble's F5 takes](reports/0005-REPORT-compaction-across-six-harnesses.md) | observation (2026-10-17; the adopted items' lines read again 2026-10-18) |
| 0006 | MADR | [Adopt goose's terminal-CLI mechanics, not its product command tree](decisions/0006-MADR-goose-cli-port-candidates.md) | accepted 2026-10-04 (wider Phase 2 scope, `9d472a9`; amended 2026-10-04 (0008); body's proposal line and a host name corrected 2026-10-19) |
| 0007 | MADR | [Bring the repository up to the fleet scaffold: agent rules, lint and line-ending config, records tooling, portable Makefile](decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md) | accepted (amended 2026-10-04 (0008)) |
| 0007 | PLAN | [Bring gobble-cli up to the fleet repository scaffold](decisions/0007-PLAN-repository-scaffolding-to-fleet-standard.md) | completed (staged; the owner commits) |
| 0008 | MADR | [The native CLI mode: a lightweight line client on Kong, x/term and the standard library, natively integrated with magic-cli-remote](decisions/0008-MADR-native-cli-mode.md) | accepted (`tools.go` note 2026-10-07; spinner label from the personality, 2026-10-18 (0022); welcome, goodbye and ambient lines, 2026-10-18 (0022); numbered permission options, 2026-10-19 (0005-PLAN F6-1)) |
| 0008 | PLAN | [Implement the native CLI mode](decisions/0008-PLAN-native-cli-mode.md) | completed (2026-10-05; P8 emphasis follow-up 2026-10-05) |
| 0009 | REPORT | [magic-cli-remote findings from gobble work](reports/0009-REPORT-magic-cli-remote-findings.md) | observation (running; updated 2026-10-12, M11–M14) |
| 0010 | REPORT | [Harness design survey: Pi, opencode and Kilo across the v1 line](reports/0010-REPORT-harness-design-survey.md) | observation (2026-10-07) |
| 0011 | REPORT | [Native tools across six harnesses: which operations gobble implements itself](reports/0011-REPORT-native-tools-survey.md) | observation (2026-10-08; 88 verified claims; schemas, descriptions, loading and BM25 evaluated 2026-10-08) |
| 0012 | MADR | [Tool schemas, descriptions and loading: JSON Schema from Go types, sectioned Markdown descriptions, two exposure tiers, and an in-house BM25 `tool_search`](decisions/0012-MADR-tool-schemas-descriptions-and-loading.md) | accepted (2026-10-08; 38 verified claims; D15 proposed 2026-10-09, with 17 verified claims; F19's consequence contradicted and D15 on hold, 2026-10-09; no defect for D15 to fix, and D15 rejected, the same day; 0021 notes 2026-10-13; D3 amended and F20 added for P7, 2026-10-13; P7's Observed section, 2026-10-13) |
| 0012 | PLAN | [Tool schemas and descriptions: the contract, the five built-in tools, and the strict probe](decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md) | completed (P1–P6 and P8 complete 2026-10-08; P7 waits on go-llmprovider-sdk's per-tool strict flag, which `v1.7.0` has (2026-10-11), and follows 0005-PLAN F4-1a; P9 proposed, approved, paused and abandoned 2026-10-09, with nothing committed; P7's strict note from 0019-MADR D11, 2026-10-13; P7 written 2026-10-13, by the owner's choice, and complete 2026-10-13, with deviations 1–2) |
| 0013 | REPORT | [A data race in Go's encoding/json when one Marshal runs inside another](reports/0013-REPORT-go-json-nested-marshal-race.md) | observation (2026-10-08; reproduced with the standard library alone; drafted for upstream, not yet filed; amended 2026-10-09: the signature varies, and gobble removes its own nested marshal; amended again 2026-10-09: corrupted memory on this host or in the Go runtime, not `encoding/json`, and the upstream draft withdrawn; amended a third time 2026-10-09: the defect is this host's WSL; gobble keeps jsonschema-go's marshaller) |
| 0014 | MADR | [Move the ACP SDK from the fork to acp-go-sdk2 v0.14.0](decisions/0014-MADR-move-to-acp-go-sdk2.md) | accepted (2026-10-10; 0019 notes 2026-10-13) |
| 0014 | PLAN | [Move the ACP SDK from the fork to acp-go-sdk2 v0.14.0](decisions/0014-PLAN-move-to-acp-go-sdk2.md) | completed (P0 and P1, 2026-10-10) |
| 0014 | REPORT | [Integrating go-llmprovider-sdk as gobble's model layer](reports/0014-REPORT-llmprovider-sdk-integration.md) | observation (2026-10-10; 0019 gap-table note 2026-10-13; 0020 note 2026-10-13) |
| 0015 | REPORT | [The three sibling libraries: current state, pins, order, and whether to adopt them now](reports/0015-REPORT-sibling-library-pins-and-order.md) | observation (2026-10-11; 293 verified claims; go-llmprovider-sdk `v1.7.1`, go-selfupdate-lib `v1.15.1`, acp-go-sdk2 `v0.14.0`; recommends adopting now; 0019 notes 2026-10-13) |
| 0016 | MADR | [Take acp-go-sdk2 v0.15.0, close the gap it opens, and adopt the stable ACP surface gobble can honour](decisions/0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md) | proposed (2026-10-12; no PLAN yet; re-based by 0019-MADR D10, 2026-10-13; amended 2026-10-15: acp-go-sdk2 `v1.14.0` and ACP auth (A1 to A3), the owner's, built by 0005-PLAN F4-3c, complete 2026-10-15) |
| 0017 | MADR | [A todo list that keeps the agent working in order, stays live through the session, and works natively with magic-cli-remote](decisions/0017-MADR-todo-list-ordered-live-and-mcremote-native.md) | proposed (2026-10-12; no PLAN yet) |
| 0018 | MADR | [The configuration surface: a commented `settings.yaml` from a shipped template, edited in place so comments and layout survive, layered and trust-gated, with Kong binding flags on top](decisions/0018-MADR-configuration-surface.md) | accepted (2026-10-12, by the owner; revised before review to YAML and with `config import` dropped, at the owner's direction; six statements corrected at acceptance; 133 verified claims; execution notes on D8, D9, D14 and D18, and observed results, 2026-10-12; D19 `defaultProvider` note 2026-10-13; D19 `cacheRetention` note 2026-10-13; `permissions.rules` note 2026-10-19 (0005-PLAN F6)) |
| 0018 | PLAN | [The configuration surface: a commented `settings.yaml`, edited in place, layered and trust-gated](decisions/0018-PLAN-configuration-surface.md) | completed (2026-10-12; P1–P7, 24 deviations, each the owner's; execution record; P7, the deadline lock, an amendment) |
| 0019 | MADR | [Pin acp-go-sdk2 v0.17.0, go-llmprovider-sdk v1.10.2 and go-selfupdate-lib v1.15.2, and integrate each where the library now does the work](decisions/0019-MADR-sibling-libraries-current-pins-and-integration.md) | accepted (2026-10-12, by the owner, with the three open questions answered; F14 and D15, `defaultProvider`, 2026-10-13, P3 deviation 2; D4 note 2026-10-13, P4 deviation 4; F15 and D3's `v0.17.1` note, P4 deviation 11; D2's `v1.12.0` note from 0020, 2026-10-13; D11's P7 note, 2026-10-13) |
| 0019 | PLAN | [Pin the three sibling libraries at their latest tags, and integrate each](decisions/0019-PLAN-sibling-libraries-current-pins-and-integration.md) | completed (2026-10-13; P1–P5; P3 deviation 2, `defaultProvider`, 2026-10-13; P4 deviations 3–11, 2026-10-13; P5 deviations 12–14, 2026-10-13) |
| 0020 | MADR | [Pin go-llmprovider-sdk v1.11.0 and offer each provider's full curated catalog from a live listing](decisions/0020-MADR-full-model-catalog-listing.md) | accepted (2026-10-13, by the owner; gobble-cli issue 1; F12 and D1's `v1.12.0` note, P1 deviation 3) |
| 0020 | PLAN | [Pin go-llmprovider-sdk v1.11.0 and offer each provider's full curated catalog from a live listing](decisions/0020-PLAN-full-model-catalog-listing.md) | completed (2026-10-13; P1–P3; P1 deviations 1–3; P2 deviations 4–5) |
| 0021 | MADR | [Ask for prompt caching on agent turns, with one provider and session id per ACP session](decisions/0021-MADR-prompt-caching-per-session.md) | accepted (2026-10-13, by the owner; gobble-cli issue 2) |
| 0021 | PLAN | [Ask for prompt caching on agent turns, with one provider and session id per ACP session](decisions/0021-PLAN-prompt-caching-per-session.md) | completed (2026-10-13; P1–P4; P3 deviation 1) |
| 0022 | MADR | [gobble speaks in its own voice: four personalities on a sarcasm scale, written in natural-language files, with every text overridable](decisions/0022-MADR-gobble-voice-and-personalities.md) | accepted 2026-10-18 (the owner's design and answers of that date: four personalities, `/personality`, Dry by default; D12 amended 2026-10-18, P4's slow lines and goodbye; D15 amended 2026-10-18, compaction prompts wait for F5-4, settled by F5-4 the same day) |
| 0022 | GATES | [gobble's voice: the owner's live evaluation](reports/0022-GATES-voice-live-evaluation.md) | prepared 2026-10-18; waiting on the owner's runs (`openai`, `grok`) |
| 0022 | PLAN | [Implement gobble's voice: four personalities, a natural-language catalog, and overrides for every text](decisions/0022-PLAN-gobble-voice-and-personalities.md) | in progress (approved 2026-10-18; P1 complete 2026-10-18, the catalog with `boring` only; P2 complete 2026-10-18, Dry the default, deviation 1; P3 complete 2026-10-18, choosing, Snarky and Bombastic; P4 complete 2026-10-18, ambient text, deviations 1 and 2; P5 complete 2026-10-18, overrides, deviation 1; P6 prepared 2026-10-18, the owner's live evaluation in 0022-GATES) |
| 0023 | MADR | [Pin the current releases: acp-go-sdk2 v1.16.0, go-llmprovider-sdk v1.15.0, the x modules and the tool modules](decisions/0023-MADR-pin-current-releases.md) | accepted 2026-10-19 (golangci-lint kept at the fleet's `v2.14.0`; deviation 1's note on the metadata fetch, 2026-10-19) |
| 0023 | PLAN | [Implement pinning the current releases](decisions/0023-PLAN-pin-current-releases.md) | completed (2026-10-19; one deviation, the metadata fetch every listing makes, accepted) |

0003-MADR has no PLAN of its own. 0004-PLAN Phase 2 and 0005-PLAN F10 implement it.

## Build order

The plans run in sequence.

- **0004-PLAN.**
  - Run: Phase D, Phase 0 (`925ef0abf83e475c33b3ffae14685617b035639e`) and Phase 1 (`661b14e768407264a38687e2db99201cae2a04a1`).
  - Run: Phase 2's steps 2–4, and its steps 1, 5 (in part), 6–9 and 11 through [0008-PLAN](decisions/0008-PLAN-native-cli-mode.md), which is completed.
  - Run: Phase 3, on 2026-10-05.
  - Run: Phase 4, step 1 through 0002-PLAN Phase 8 and steps 2–4 on 2026-10-07.
  - Run: Phase 5 on 2026-10-07.
  - Not run: Phase 2's steps 10, 12 and 13, which wait on sessions and providers.
- **0002-PLAN.** Run: Phases 1–8, completed on 2026-10-07.
- **0005-PLAN.**
  - Run: F1a, the file tools, and F1b, bash and powershell, on 2026-10-07.
  - Run: F1c-1, search, and F1c-2, the file operations, on 2026-10-08.
  - Run: F1d-1, `todo`, F1d-2, `tool_search` and the MCP resource tools, and F1d-3, the client's file and terminal methods, on 2026-10-09. F1 is complete.
  - Run: F2-1, reading Pi's sessions of every version and entry type, and F2-2, the tree, on 2026-10-10. F2 is complete.
  - Run: F3-1, context files and the sectioned prompt, F3-2, skills, F3-3, prompt templates, and F3-4, `@path`, resource blocks and the prompt capabilities, on 2026-10-10. F3 is complete.
  - Run: F4-1a, the tool result's error flag, the wire tests and the live tests, on 2026-10-13
    (re-made at SDK `v1.12.0`; the pin and `NativeStreaming` done in 0019-PLAN P3).
  - Run: F4-2a, the models.dev snapshot and what it feeds, on 2026-10-13 (one deviation).
  - Run: F4-2b, `models.json`, on 2026-10-13 (two deviations).
  - Run: F4-2c, choosing a model, on 2026-10-14 at SDK `v1.13.0` (two deviations).
  - Run: F4-3a, keys, on 2026-10-14 (three deviations).
  - Run: F4-3b, the Grok and Kilo logins, on 2026-10-14 (two deviations), and the owner's live Grok login.
  - Run: F4-3c-1, acp-go-sdk2 `v1.14.0`, on 2026-10-15 (two deviations).
  - Run: F4-3c-2, ACP authentication, on 2026-10-15 (three deviations).
  - Run: F4-3c-3, `--json` logins, on 2026-10-15 (no deviation).
  - Run: F4-3c-4, the Pi import, on 2026-10-15 (no deviation). F4-3c is complete.
  - Next: F4-3d, the ChatGPT login, waits for an
    SDK release with OpenAI's sign-in for agents. F4-1b, live streaming, waits on an SDK release that retries a stream.
  - Run: F5-1, what the agent adds to the SDK's retry, on 2026-10-16 (one deviation), before F4 is complete, by the
    owner's choice of that date.
  - Run: F5-2, automatic compaction, on 2026-10-18 (one deviation).
  - Run: F5-3, loop safety, on 2026-10-18 (no deviation).
  - Run: F5-4, the summary, on 2026-10-18 (opencode v2's template, grok-build's two; prune not adopted; no deviation).
  - Run: F5-5, the small model, on 2026-10-18 (titles only; none while unset; at the first prompt, in the background; no deviation). F5 complete.
  - Made executable: F6, re-made as five sub-phases (trust, rules and requests, shell commands, modes,
    `request_permissions`), and F6-1, project trust, on 2026-10-19.
  - Run: F6-1, project trust, on 2026-10-19 (the home and filesystem-root limit kept; no deviation).
  - Docs brought up to date on 2026-10-19: `architecture.md`, the root README, `developing.md`,
    `choosing-a-model.md`, the stale package comments, and `personality`'s tier.
  - Made executable: F6-2, rules and permission requests, on 2026-10-19, as two sub-phases, F6-2a the engine and
    F6-2b the questions and stores; approved as written, to run after 0023-PLAN, which is complete.
  - Run: F6-2a, the rule engine, on 2026-10-19 (three deviations: acpclient carries the limit mark, the repeated-call
    question is marked, the outside title stays).
  - Made executable: F6-2b, the questions and the stores, on 2026-10-19; approved as written, with "once only" for a
    subject no rule can match exactly and the project store applying untrusted.
  - Run: F6-2b on 2026-10-19 (no deviation). F6-2 is complete.
  - Made executable: F6-3, shell commands, on 2026-10-19; approved as written, an "always" saving the arity prefix,
    shown in the question.
  - Run: F6-3 on 2026-10-19 (no deviation).
  - Made executable: F6-4, the modes, on 2026-10-19; approved as written, the user's rules winning over
    accept-edits and gobble's limits still asking in bypass.
  - Run: F6-4 on 2026-10-19 (no deviation). F6-5, `request_permissions`, is next on the owner's word. No release is
    tagged before F6-5, by the owner's choice of that date.
  - The SDKs: [0023-PLAN](decisions/0023-PLAN-pin-current-releases.md) pinned go-llmprovider-sdk `v1.15.0` and
    acp-go-sdk2 `v1.16.0` on 2026-10-19, so `gobble acp` answers a request piped before the end of its input. A
    provider's default model, the first of its live listing, may change with the SDK's ranking from `v1.14.0`, and every
    listing but Kilo's and Ollama's also fetches models.dev's metadata document, with no credential, at most once in 10
    minutes. The SDK's ChatGPT-plan sign-in, which F4-3d waits for, and `WithRetryNotify`, the retry notices F5-1 left
    for later, are each a sub-phase made executable on the owner's word. F4-1b still waits: `v1.15.0`'s retry still
    buffers a stream (`llmprovider/retry.go:79-80`).
- **0014-PLAN.** Run: P0 and P1, the move to acp-go-sdk2 `v0.14.0`, on 2026-10-10. Complete.
- **0018-PLAN.** Run: P1–P7, the configuration surface, on 2026-10-12, P7, the deadline lock, as an amendment.
  Complete. Put ahead of 0017 by the owner.
- **0019-PLAN.** Run: P1–P5 on 2026-10-12 and 2026-10-13. Complete. go-llmprovider-sdk `v1.11.0`, tagged after P3, is the next record's (its deviation 13).
- **0020-PLAN.** Run: P1–P3 on 2026-10-13. Complete; gobble-cli issue 1 closed.
- **0021-PLAN.** Run: P1–P4 on 2026-10-13. Complete.
- **0022-PLAN.**
  - Run: P1, the catalog with `boring` only, on 2026-10-18 (no deviation).
  - Run: P2, Dry the default, on 2026-10-18 (one deviation).
  - Run: P3, choosing, and Snarky and Bombastic, on 2026-10-18 (no deviation; the phone check is the owner's).
  - Run: P4, ambient text and easter eggs, on 2026-10-18 (two deviations, D12 amended).
  - Run: P5, every text overridable, on 2026-10-18 (one deviation: compaction prompts wait for 0005-PLAN F5-4,
    settled by F5-4 the same day).
  - Prepared: P6, the owner's live evaluation, in 0022-GATES (2026-10-18); waiting on the owner's runs against `openai` and `grok`.
- **0012-PLAN.** Run: P1–P6 and P8 on 2026-10-08. P7 ~~waits on go-llmprovider-sdk's per-tool strict flag~~ follows 0005-PLAN F4-1a, which pins go-llmprovider-sdk `v1.7.0` and its strict flag (2026-10-11).
  - Run: P7 on 2026-10-13: `strict: false` where a call may leave an argument out, no flag otherwise. Complete; the Claude, Gemini and Grok live tests are the owner's.
- `git ls-remote origin refs/heads/main` on 2026-10-08 returned `740f7b71450238e747dba99b4fa1eccd488ad79e`.

The plans:

1. **[0004-PLAN](decisions/0004-PLAN-go-module-architecture.md)** — source documentation in Phase D, then the scaffold in Phases 0–5. It replaces 0002-PLAN Phase 0.
2. **[0002-PLAN](decisions/0002-PLAN-cli-acp-headless-mcp-v1.md) Phases 1–8** — the ACP agent, Kong as an ACP client, the first tools, sessions, MCP, and native slash commands.
3. **[0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md)** — F1–F10 form the v1.0.0 gate. X1–X8 are the v1.x train and `exp/`.
4. **[0012-PLAN](decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md)** — tool schemas and descriptions for the built-in tools, and the strict probe.
5. **The pairs of 0014 and 0018 to 0022** — the move to acp-go-sdk2 (0014), the configuration surface (0018), the
   sibling libraries' pins (0019), the full model catalog (0020), prompt caching (0021) and gobble's voice (0022);
   each is in the run list above.

## I want to…

| I want to… | Start here |
| :--- | :--- |
| know where the project started: whether a complete Go port of Pi was feasible | [0001-REPORT](reports/0001-REPORT-go-port-feasibility.md) |
| know why ACP is the command API and how gobble sits under mcremote | [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) |
| see how gobble maps onto mcremote slash commands (`/compact`, `/usage`, …) | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (target table), [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md) (floor) |
| know why gobble cannot answer `session/set_model` | [0002-MADR](decisions/0002-MADR-cli-acp-headless-mcp-v1.md), second amendment |
| know where gobble keeps config, sessions and credentials | [0003-MADR](decisions/0003-MADR-gobble-product-identity.md) |
| know the licence | [0003-MADR](decisions/0003-MADR-gobble-product-identity.md) (Apache-2.0), [LICENSE](../LICENSE), [NOTICE](../NOTICE) |
| use my existing Pi skills, MCP servers and sessions with gobble | [0003-MADR](decisions/0003-MADR-gobble-product-identity.md) (read-only Pi bridge) |
| see the package map, public SDK tiers and import rules | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) |
| know which Go 1.25–1.27 features gobble uses, and for what | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (idiom list) |
| know which standards each boundary speaks (ACP, MCP, Agent Skills, OTel, OAuth, XDG) | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) |
| check whether a Pi feature made it into v1, and at which tier | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) |
| extend gobble (hooks, MCP servers, skills, packages, Go SDK) | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md), Extensibility |
| see what has to pass before `v1.0.0` is tagged | [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md), F10 |
| start building | [0004-PLAN](decisions/0004-PLAN-go-module-architecture.md) |
| know how gobble talks to models | [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (D10: `llm` facade over go-llmprovider-sdk) |
| know how `gobble update` and GitHub releases work | [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (self-update), [0004-PLAN](decisions/0004-PLAN-go-module-architecture.md) Phase 4 (assets) |
| see which sibling libraries exist today and what gobble will use | [architecture.md](architecture.md#dependencies), [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (2026-10-04: core TUI is go-tui-lib; self-update is go-selfupdate-lib), [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md) (F9, F10) |
| use the default terminal CLI, or the enhanced TUI | [architecture.md](architecture.md), [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (two terminal modes) |
| understand what this repository contains today | [architecture.md](architecture.md) |
| build gobble, run the gates, or add a package | [guides/developing.md](guides/developing.md) |
| point a provider at a proxy, add headers or models, or correct a model's window or price | [guides/models-json.md](guides/models-json.md) |
| choose a run's model and thinking level, or see the models gobble knows | [guides/choosing-a-model.md](guides/choosing-a-model.md) |
| store, check or remove an API key, or see where a key comes from | [guides/credentials.md](guides/credentials.md) |
| see which operations gobble runs as native tools rather than shell commands, and why | [0011-REPORT](reports/0011-REPORT-native-tools-survey.md), [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (amendment of 2026-10-08) |
| know why the race gate can fail in `internal/cli` once in a few hundred runs, and what gobble does about it | [0013-REPORT](reports/0013-REPORT-go-json-nested-marshal-race.md) |
| know which go-llmprovider-sdk, go-selfupdate-lib and acp-go-sdk2 releases gobble pins, what each next release changes, in what order gobble takes them, and whether to adopt them now | [0015-REPORT](reports/0015-REPORT-sibling-library-pins-and-order.md) |
| know which parts of ACP and acp-go-sdk2 gobble uses, which it leaves, and why | [0016-MADR](decisions/0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md), [0014-MADR](decisions/0014-MADR-move-to-acp-go-sdk2.md) (the move to sdk2) |
| know how gobble's tools are described to a model, which are always loaded, and how `tool_search` finds the rest | [0012-MADR](decisions/0012-MADR-tool-schemas-descriptions-and-loading.md), [0011-REPORT](reports/0011-REPORT-native-tools-survey.md) (evaluation) |
| see how other harnesses design what gobble builds, and which design gobble chose | [0010-REPORT](reports/0010-REPORT-harness-design-survey.md), [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (amendment of 2026-10-07) |
| retry failed model calls, wait for the network, or compact the conversation on its own | [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md) (F5-1, F5-2, F5-4), the `retry` and `compaction` blocks of `settings.yaml` |
| name sessions with a small model | [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md) (F5-5), [guides/choosing-a-model.md](guides/choosing-a-model.md) |
| allow, ask or deny a tool call by rule, answer "always" for a session, a project or everywhere, list and edit the rules with `/permissions`, or let print mode answer with `--yes` | [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md) (F6-2, F6-2b), the `permissions` block of `settings.yaml` |
| trust a project, so its own settings, `SYSTEM.md`, skills, prompts and personality files load | [0005-PLAN](decisions/0005-PLAN-v1-feature-scope.md) (F6-1), [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (Project trust), `gobble trust --help` |
| choose gobble's personality, or change its messages and model-facing text | [0022-MADR](decisions/0022-MADR-gobble-voice-and-personalities.md), `gobble personality --help` |
| see how six harnesses compact a conversation, and what gobble's F5 takes | [0005-REPORT](reports/0005-REPORT-compaction-across-six-harnesses.md), [0005-MADR](decisions/0005-MADR-v1-feature-scope.md) (amendment of 2026-10-18) |
| see each package's tier and role | [architecture.md](architecture.md#packages) |
| know where gobble's settings live, how the commented `settings.yaml` template grows, how global, project, `--set` and flags combine, which keys exist, and how to edit or validate them | [0018-MADR](decisions/0018-MADR-configuration-surface.md), [0004-MADR](decisions/0004-MADR-go-module-architecture.md) (Settings), [0003-MADR](decisions/0003-MADR-gobble-product-identity.md) (directories) |
| know how gobble's todo list keeps the agent in order, stays current through resume, cancel and compaction, and shows on magic-cli-remote's phone | [0017-MADR](decisions/0017-MADR-todo-list-ordered-live-and-mcremote-native.md), [0009-REPORT](reports/0009-REPORT-magic-cli-remote-findings.md) (M11–M14) |
| review what gobble work found about magic-cli-remote, and the changes it needs there | [0009-REPORT](reports/0009-REPORT-magic-cli-remote-findings.md) |
| know the rules agents follow in this repository | [AGENTS.md](../AGENTS.md) |
| know why the repository has the files and gates it has (AGENTS.md, lint configs, records check) | [0007-MADR](decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md) |
