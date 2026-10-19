# gobble-cli

**gobble** is a coding agent written in Go: a terminal CLI, an [Agent Client Protocol](https://agentclientprotocol.com)
agent for editors, and an embeddable Go SDK, in one pure-Go binary. The repository is `gobble-cli`; the
binary is `gobble`; the module is `github.com/maccavelli/gobble-cli`.

gobble is its own design. It takes the best of what other agent harnesses have already worked out —
[Pi](https://github.com/earendil-works/pi), [opencode](https://github.com/anomalyco/opencode),
[Kilo](https://github.com/Kilo-Org/kilocode), [goose](https://github.com/aaif-goose/goose),
[codex](https://github.com/openai/codex) and [grok-build](https://github.com/xai-org/grok-build) —
and makes each piece gobble's own, in its own words and to its own rules. Nothing is copied for its
own sake. Compatibility is kept only where it serves gobble's users: a Pi user's sessions, settings,
skills and MCP servers carry over through a read-only bridge.

**Documentation:** [docs/](docs/README.md)

## What it draws from

Each choice is argued, with source evidence, in the records:

| From | What gobble takes | Where it is decided |
| :--- | :--- | :--- |
| Pi | The baseline capability list, the tool shapes, session JSONL compatibility, and the read-only bridge for `~/.pi` assets | [0003-MADR](docs/decisions/0003-MADR-gobble-product-identity.md), [0005-MADR](docs/decisions/0005-MADR-v1-feature-scope.md) |
| opencode | Layered edit matching, numbered reads, last-match-wins permission rules, and the compaction template | [0010-REPORT](docs/reports/0010-REPORT-harness-design-survey.md); 0005-MADR, amendment of 2026-10-07 |
| Kilo | Hard permission ceilings, protected paths, loop circuit breakers, and notebook and document handling | the same |
| goose | Terminal-CLI mechanics, `tree`, and document extraction | [0006-MADR](docs/decisions/0006-MADR-goose-cli-port-candidates.md), [0011-REPORT](docs/reports/0011-REPORT-native-tools-survey.md) |
| codex | `apply_patch`, process sessions, `request_permissions`, and deferred tools behind a BM25 `tool_search` | 0011-REPORT; 0005-MADR, amendment of 2026-10-08 |
| grok-build | Gitignore-aware `tree` with an output budget, the `lsp` tool, `monitor`, and completion reminders | the same |

What is gobble's own: one ACP agent as the command API for every client, native file operations that
the permission rules and undo can see, a two-tier tool exposure that keeps the prompt cache stable, and
error messages written in one style throughout
([0012-MADR](docs/decisions/0012-MADR-tool-schemas-descriptions-and-loading.md), accepted).

## Status

Early, and built in the order [docs/README.md](docs/README.md#build-order) gives. What runs today:

- **`gobble`**, the native CLI mode: a line session in a terminal, or one prompt with `-p` or piped input.
- **`gobble acp`**: the ACP agent on stdio, for editors and magic-cli-remote. It has sessions, slash
  commands, modes, compaction by `/compact` and on its own, MCP servers and gobble's `_gobble/` extension
  methods. A client that offers files has file tools read and write its editor's text; with
  `tools.bash.useClientTerminal: true`, one that offers terminals runs bash in its terminal; each tool
  call names the files it touches. `gobble acp --login <provider>` is what an ACP client's terminal
  sign-in runs.
- **Tools:** `read`, `write` and `edit`, confined to the workspace roots; `grep`, `find` and `tree`,
  which follow the repository's `.gitignore` rules; `move`, `delete`, `copy` and `mkdir`, where every
  delete asks first; `todo`, a checklist the client shows as the plan; `tool_search`, which loads
  deferred tools; the MCP resource tools; `bash`, and `powershell` on Windows, run as process trees
  that stop whole.
- **Model calls that recover:** a failed call is retried by the provider; with the network down, a turn waits
  for it to come back and resends; an empty reply runs twice more (`retry.*` settings). Compaction runs on its own
  near the model's window and once on an overflow (`compaction.*`), on a summary template from opencode. Three
  identical calls in a row ask whether to go on, and three calls in a row whose arguments are refused end the turn.
- **Session titles:** with `smallModel` set, a small model names a new session from its first prompt.
- **Sessions** as JSONL files in Pi's v3 shapes, with every Pi entry type in the model's context as Pi puts it there. Pi's v1 and v2 files load,
  migrated in memory and never rewritten; `/clone` continues one as a v3 session.
- **The system prompt** in Pi's sections, from `AGENTS.md` and `CLAUDE.md` files, `SYSTEM.md` and `APPEND_SYSTEM.md`, or `--system-prompt`
  and `--append-system-prompt`; it stays fixed for a session, and a change reaches the model as a message.
- **Agent Skills** from the config directory's `skills/` and `~/.agents/skills`, listed in the prompt; `/skill:<name>` sends one, and
  `--no-skills` turns them off.
- **Prompt templates** from the config directory's `prompts/`, as Pi writes them: `/review concurrency` sends `review.md` with
  its arguments in place.
- **Project trust:** a project's own `.gobble/settings.yaml`, `SYSTEM.md`, `APPEND_SYSTEM.md`, skills, prompt templates and
  personality files load only once it is trusted: by `gobble trust`, `/trust`, the question gobble's terminal asks,
  `--approve` for one run, or `defaultProjectTrust: always`. `gobble untrust` and `--no-approve` refuse it.
- **Permission rules:** `permissions.rules` in `settings.yaml`, a trusted project's after it: `allow`, `ask` or
  `deny` of a tool, or of a tool and a pattern over its paths or its command, such as `deny: read(.env)` or
  `allow: bash(git status*)`; the last that matches decides. With none, a read-only built-in tool runs and every other
  call asks. A write under `.git` or `~/.ssh` is always refused, and a delete always asks. A question offers "once",
  "for this session", "for this project" or "everywhere" each way, and `[f]` rejects with a reason the model reads;
  `/permissions` lists, removes and adds rules. In print mode, `--yes` allows once what a rule asks. A bash
  command is split into its simple commands, each matched on its own, so `deny: bash(rm *)` reaches `make && rm -rf x`;
  an "always" covers the command's prefix, such as `git status *`, which the question names; a plain `rm`, `mv`, `cp`,
  `rmdir` or `mkdir` runs as gobble's own tool.
- **Modes:** `default`, `accept-edits` (edits without asking), `plan` (read-only until a plan is approved) and
  `bypass` (asks nothing but gobble's own limits, offered only with `permissions.allowBypass: true` in the global
  settings); `/mode`, `/plan` and `--mode`.
- **`request_permissions`:** after you decline a file call, the model can ask you for read or write access to paths,
  for the turn or the session, saying why; a grant answers the questions under those paths, and your deny rules and
  gobble's limits still hold.
- **Personalities:** gobble speaks as `boring`, `dry` (the default), `snarky` or `bombastic`, or one of your own;
  `/personality`, `--personality` and `gobble personality list|show|check|path|reset`. The messages in its catalog, and
  the system prompt's tools and rules sections, compaction's prompts and the built-in tools' descriptions, can be
  replaced from files.
- **`@path` in a prompt** attaches that file, read inside the workspace, ahead of the text; ACP clients can attach
  resources and links as well.
- **`gobble mcp add|remove|list`**, `gobble config path|get|set|unset|edit|init|update|validate|schema`,
  `gobble trust` and `gobble untrust`, `gobble version` and `gobble completion`. Settings live in a commented `settings.yaml` in the config directory; `gobble config path`
  shows where.
- **`models.json`** in the config directory, in Pi's schema: a provider's base URL and headers, added models, and
  corrections to a model's window, price and thinking levels ([guide](docs/guides/models-json.md)).
- **Choosing a model:** `--model` and `--provider` by Pi's rules, `--thinking` up to `max`, the
  `defaultModel` setting, and `gobble models list|info` ([guide](docs/guides/choosing-a-model.md)).
- **Credentials** in Pi's order: `--api-key -`, a Pi-style `auth.json`, `models.json`'s `apiKey`, then the
  provider's variable; `gobble auth login --api-key -|logout|status|check`, signing in to Grok or Kilo with
  `gobble auth login grok|kilo`, and `gobble auth import --from-pi` ([guide](docs/guides/credentials.md)).

Not built yet: `--offline`, which is refused until 0005-PLAN X3; the enhanced TUI
mode; and `gobble update`. The full v1 line, tiered into
a v1.0.0 gate, a v1.x train and `exp/`, is in [0005-MADR](docs/decisions/0005-MADR-v1-feature-scope.md).

| I want to… | Start here |
| :--- | :--- |
| see what this repository contains today | [architecture.md](docs/architecture.md) |
| see everything in the v1 line, and at which tier | [0005-MADR](docs/decisions/0005-MADR-v1-feature-scope.md) |
| see how other harnesses design what gobble builds, and which design gobble chose | [0010-REPORT](docs/reports/0010-REPORT-harness-design-survey.md), [0011-REPORT](docs/reports/0011-REPORT-native-tools-survey.md) |
| know how gobble talks to editors and magic-cli-remote | [0002-MADR](docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md) |
| know what the binary, directories and wire names are | [0003-MADR](docs/decisions/0003-MADR-gobble-product-identity.md) |
| see the package layout, SDK surface and Go idioms | [0004-MADR](docs/decisions/0004-MADR-go-module-architecture.md) |
| see the build order | [docs/README.md](docs/README.md#build-order) |
| build gobble, run the gates, or add a package | [developing.md](docs/guides/developing.md) |
| know where the project started: the Go port feasibility study of Pi | [0001-REPORT](docs/reports/0001-REPORT-go-port-feasibility.md) |

## Stack

- **Modes:** a native terminal CLI mode, the default, and an enhanced terminal TUI mode, not built yet.
- **CLI:** [Kong](https://github.com/alecthomas/kong) `v1.16.1`.
- **Protocols:** the [ACP Go SDK](https://github.com/maccavelli/acp-go-sdk2) `acp-go-sdk2` `v1.16.0`, and the
  [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk) `v1.8.0`.
- **Models:** [go-llmprovider-sdk](https://github.com/maccavelli/go-llmprovider-sdk) `v1.15.0`, behind
  gobble's own `llm` facade. Only `llm/provider` imports it.
- **Tool schemas:** JSON Schema 2020-12 from Go types, with [jsonschema-go](https://github.com/google/jsonschema-go) `v0.4.3`.
- **Settings, skills and templates:** YAML through [go.yaml.in/yaml/v3](https://github.com/yaml/go-yaml) `v3.0.5`.
- **Build identity and self-update:** [go-selfupdate-lib](https://github.com/maccavelli/go-selfupdate-lib)
  `v1.15.2`. gobble does not reimplement self-update.
- **TUI:** [go-tui-lib](https://github.com/maccavelli/go-tui-lib), not required yet. gobble does not
  reimplement core TUI; where a behaviour is not in the library yet, gobble waits rather than copying it.

Shipped binaries are pure Go (`CGO_ENABLED=0`).

## License

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE).
Third-party notices are in [NOTICE](NOTICE).
