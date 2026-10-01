---
status: proposed
date: 2026-10-01
decision-makers: repository owner
consulted: 0001-REPORT-go-port-feasibility.md, 0002-MADR-cli-acp-headless-mcp-v1.md
informed: magic-cli-remote (companion provider Spec), go-llmprovider-sdk, go-core-lib
---
# The product is `pigo`: its own binary, module path, directories, wire names, and a read-only bridge to `~/.pi`

## Context and Problem Statement

[0001-REPORT-go-port-feasibility.md](../reports/0001-REPORT-go-port-feasibility.md)
left identity open (D17: binary name, config directory, User-Agent, module
path), together with home-directory sharing (D11) and provenance (D18).
[0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md)
used `pi-go` as a placeholder binary name and `_pi/` as its ACP extension
prefix. It also said no release may be tagged until identity is decided.

The owner has now named the runtime binary: **`pigo`**.

TypeScript Pi owns the name `pi`, the `~/.pi` tree, the `PI_*` environment,
the npm scope `@earendil-works`, and `pi.dev`. The upstream licence is MIT
(0001-REPORT F16). A second product that writes into `~/.pi` or answers to
`PI_*` would corrupt or confuse an installed TypeScript Pi. Users of that
Pi, though, already have skills, `AGENTS.md` files, `mcp.json` servers,
`models.json` providers, and session history. A clean break would throw all
of that away.

The problem: fix every name the binary presents to the operating system,
the network, and the protocols it speaks, and decide how it reads Pi's data
without owning it.

## Decision Drivers

* Owner-stated binary name `pigo`.
* No write ever lands in `~/.pi` or `.pi/`: TypeScript Pi must keep working
  beside pigo on the same machine.
* A Pi user's skills, context files, MCP servers, custom models, and
  sessions should be usable on day one.
* Fleet convention: module paths are `github.com/maccavelli/<repository>`
  (fleet `docs/0005-MADR-github-origin-consolidation.md`). XDG directories
  are resolved in-tree, as magic-cli-remote does in `internal/appdirs`.
* Go idiom: a package's name matches the last element of its import path,
  and repositories named `go-x` or `x-go` keep their code in a
  subdirectory (`github.com/google/go-cmp/cmp`).
* Protocol hygiene: ACP `agentInfo`, MCP `clientInfo`, the HTTP
  `User-Agent`, and ACP extension method names all identify the product
  that is actually speaking.

## Considered Options

* `pigo` identity with XDG directories and a read-only Pi bridge
* `pigo` binary, but share `~/.pi/agent` and `PI_*` with TypeScript Pi
* `pigo` identity with no Pi compatibility at all
* Rename the repository and module to `pigo` so the module root is the package

## Decision Outcome

Chosen option: "`pigo` identity with XDG directories and a read-only Pi
bridge". It gives pigo its own name, directories, and wire names, and it
still lets a Pi user carry their data across without risking the
TypeScript install.

### Names

| Surface | Value |
|---|---|
| Binary | `pigo` (`cmd/pigo`) |
| Module path | `github.com/maccavelli/pi-go` |
| Embedding facade package | `github.com/maccavelli/pi-go/pigo` (package `pigo`) |
| Install | `go install github.com/maccavelli/pi-go/cmd/pigo@latest` |
| Display name | `pigo` (lower case, everywhere) |
| ACP `agentInfo` | `{name: "pigo", title: "pigo", version: <semver>}` |
| MCP `clientInfo` | `{name: "pigo", version: <semver>}` |
| HTTP `User-Agent` | `pigo/<semver> (<GOOS>; <GOARCH>) go/<go version>` |
| ACP extension namespace | `_pigo/` (replaces 0002's `_pi/`) |
| Session-entry custom types | `pigo.<kind>` (for example `pigo.checkpoint`) |
| Environment prefix | `PIGO_` |
| Child-process markers | `AI_AGENT=pigo`, `PIGO_SESSION_ID`, `PIGO_SESSION_FILE`, `PIGO_PROVIDER`, `PIGO_MODEL`, `PIGO_THINKING_LEVEL` |
| Temp-file prefix | `pigo-` (for example `pigo-bash-<id>.log`) |
| Project directory | `.pigo/` |

The module root holds no package. Library code lives in named
subdirectories (see
[0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md)),
following the `go-cmp/cmp` pattern.

### Directories

pigo resolves XDG Base Directory locations itself on every Unix, macOS
included, so that dotfile users find everything under `~/.config`.
Windows uses Known Folders.

| Role | Unix and macOS | Windows | Holds |
|---|---|---|---|
| config | `$XDG_CONFIG_HOME/pigo` (default `~/.config/pigo`) | `%APPDATA%\pigo` | `settings.json`, `mcp.json`, `models.json`, `keybindings.json`, `trust.json`, `skills/`, `prompts/`, `agents/`, `hooks/`, `themes/`, `SYSTEM.md`, `APPEND_SYSTEM.md`, `AGENTS.md` |
| data | `$XDG_DATA_HOME/pigo` (default `~/.local/share/pigo`) | `%LOCALAPPDATA%\pigo\data` | `sessions/`, `packages/`, `checkpoints/`, `usage/` ledger |
| state | `$XDG_STATE_HOME/pigo` (default `~/.local/state/pigo`) | `%LOCALAPPDATA%\pigo\state` | `logs/`, `mcp.log`, prompt history, flight-recorder dumps |
| cache | `$XDG_CACHE_HOME/pigo` (default `~/.cache/pigo`) | `%LOCALAPPDATA%\pigo\cache` | model-catalog refresh, package downloads, large tool outputs |
| secrets | OS keyring (service `pigo`); fallback `<config>/auth.json` at mode 0600 | Windows Credential Manager; fallback DPAPI file | provider keys, OAuth tokens, `mcp-auth` |

`PIGO_HOME=<dir>` collapses all four roles into one directory, for
portable installs and tests. It is the analogue of Pi's `PI_AGENT_DIR`.
`PIGO_CONFIG_DIR`, `PIGO_DATA_DIR`, `PIGO_STATE_DIR` and `PIGO_CACHE_DIR`
override one role each.

Project resources live in `.pigo/` and load only for a trusted project,
using the same rule as Pi's `.pi/`. Directories named by open standards
are read too: `AGENTS.md` walking up from `cwd`, and `.agents/skills` /
`~/.agents/skills` from the Agent Skills convention.

### The read-only Pi bridge

pigo **reads** these Pi locations at the lowest precedence. It **never
writes** to them. It never takes a lock in `~/.pi` either, which also
avoids contention with Pi's `proper-lockfile` locks.

| Pi resource | pigo behaviour |
|---|---|
| `~/.pi/agent/skills/`, `.pi/skills/` (trusted) | discovered as skills after pigo's and the `.agents` directories; a name collision warns and the first skill found wins (Pi's rule) |
| `~/.pi/agent/prompts/`, `.pi/prompts/` | discovered as prompt templates |
| `~/.pi/agent/AGENTS.md`, `SYSTEM.md`, `APPEND_SYSTEM.md` | used only when the pigo config directory has no file of that name |
| `~/.pi/agent/mcp.json`, `.pi/mcp.json` | merged under pigo's `mcp.json`; a server that pigo also names comes from pigo |
| `~/.pi/agent/models.json` | merged under pigo's `models.json` |
| `~/.pi/agent/sessions/**.jsonl` | listed by `pigo session list --pi`; copied with `pigo session import`; never modified in place |
| `~/.pi/agent/settings.json` | not merged automatically. `pigo config import --from-pi` writes a pigo `settings.json` from it once, reporting every key it dropped. |
| `~/.pi/agent/auth.json` | not read. Credentials are copied only by an explicit `pigo auth import --from-pi`, which asks per provider. |
| `PI_*` environment | ignored |

`bridge.pi = false` in settings (or `--no-pi-bridge`) turns the bridge off.
`pigo doctor` reports which bridge sources were found.

### Provenance and licence

pigo is a **specification-derived rewrite**. Pi's public documentation and
its source are read as a behavioural specification: tool limits, session
format, template syntax, merge rules. Go code is written fresh, not
transcribed. pigo is released under the Apache License 2.0. `LICENSE`
is the fleet copy (201 lines, appendix unfilled), `cmp`-identical to
`go-core-lib` and `magic-cli-remote`. A `NOTICE` file credits Pi
(MIT, Copyright (c) 2025 Mario Zechner) and marks any data files copied
verbatim, such as theme JSON or prompt text. Those files keep the upstream
notice. It also names `github.com/maccavelli/go-core-lib` and
`github.com/maccavelli/go-llmprovider-sdk`, both Apache License 2.0.

### Consequences

* Good, because TypeScript Pi and pigo coexist: pigo never writes to Pi's
  tree or reads Pi's credential file uninvited.
* Good, because a Pi user's skills, templates, context files, MCP servers,
  custom models, and history work on the first run.
* Good, because every wire name (`agentInfo`, `clientInfo`, `User-Agent`,
  `_pigo/`) identifies pigo. Provider and MCP-server logs can tell it apart
  from Pi.
* Good, because `go install …/cmd/pigo@latest` and the binary name agree,
  and the `go-cmp/cmp` pattern keeps package names equal to their
  directory names.
* Neutral, because the import path `github.com/maccavelli/pi-go/pigo`
  repeats the product name once. A later repository rename to `pigo` is
  possible before `v1.0.0`: GitHub redirects old URLs, and the module has
  no dependents yet.
* Neutral, because macOS users who expect `~/Library/Application Support`
  will find pigo under `~/.config`, the convention of Go CLI peers such as
  `gh`.
* Bad, because two copies of settings can drift for a user who runs both
  products. `config import` is a one-time copy, not a sync.
* Bad, because the licence position (D18) has had no legal review.
  Apache-2.0 and the Pi MIT notice in `NOTICE` state the grant; it is
  unreviewed.

### Confirmation

* A test with `PIGO_HOME` unset and a fake `$HOME` asserts each role path
  above on linux, darwin, and windows.
* A bridge test builds a fake `~/.pi/agent` tree, runs discovery and
  `session import`, and asserts that the tree's hash is unchanged afterwards.
  The same test is shown failing on a deliberately broken bridge that
  writes a lock file.
* `pigo version --json`, the ACP `initialize` response, an MCP
  `initialize` request captured from a fixture server, and an `httptest`
  provider request all carry the name `pigo` and the same semver.
* `grep -rn '"_pi/' --include=*.go .` is empty; every extension method
  starts with `_pigo/`.
* `LICENSE` is Apache-2.0 (`cmp`-identical to `go-core-lib/LICENSE`)
  and `NOTICE` names Pi's MIT copyright line, `go-core-lib`, and
  `go-llmprovider-sdk`.

## Pros and Cons of the Options

### `pigo` identity with XDG directories and a read-only Pi bridge

* Good, because it gives migration value with zero write risk to Pi.
* Good, because it follows the fleet's `appdirs` precedent and the XDG
  standard.
* Bad, because the bridge is extra code with its own tests and precedence
  rules.

### `pigo` binary, but share `~/.pi/agent` and `PI_*` with TypeScript Pi

* Good, because there would be one settings file, one session list, and one
  set of credentials.
* Bad, because two products would write the same files under different lock
  implementations, and a pigo schema extension would break Pi's parser or
  the reverse.
* Bad, because `PI_*` variables would configure two programs with different
  semantics.

### `pigo` identity with no Pi compatibility at all

* Good, because it is the simplest: no bridge and no import commands.
* Bad, because every Pi user starts from zero, even for resources that are
  open standards (Agent Skills, `AGENTS.md`, MCP configuration).

### Rename the repository and module to `pigo` so the module root is the package

* Good, because `import "github.com/maccavelli/pigo"` would be the
  cleanest possible path.
* Neutral, because GitHub redirects a renamed repository, and there are no
  dependents today.
* Bad, because it is an outward-facing action on the hosting account that
  this decision does not need. The owner can still take it before
  `v1.0.0` without changing anything else here.

## More Information

* Decides report D11 (home directory), D17 (identity), and D18
  (provenance) for this repository.
* Amends [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md):
  the binary is `pigo` (not `pi-go`), `pigo acp` is the headless entry,
  and the extension prefix is `_pigo/`.
* Implemented by Phase 2 of
  [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md)
  (directories, build info, names) and by the bridge phase of
  [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md).
* Amended 2026-09-30: `NOTICE` also names `go-core-lib` (Apache-2.0).
  `go-llmprovider-sdk` has no LICENSE in tree; none is invented.
* Amended 2026-10-01: the owner directed Apache-2.0 for this
  repository, `go-llmprovider-sdk`, and `go-core-lib`. pigo's `LICENSE`
  is that fleet copy. `NOTICE` names Pi (MIT) and both sibling
  libraries (Apache-2.0). The 2026-09-30 "do not invent" line for
  `go-llmprovider-sdk` is closed by that repository's
  `0018-MADR-apache-2-license.md`. `LICENSE` and `NOTICE` land in this
  change, ahead of 0004-PLAN Phase 0.
* Pi facts come from Pi at `312184edb`: `docs/configuration.md`,
  `docs/environment-variables.md`, `docs/skills.md`, `docs/security.md`,
  and `core/resource-loader.ts`.

### Amendment (2026-10-01, second): the bridge measured against Pi's code

Still `proposed`. Pi was re-read in code at `312184edb`. Paths below are under `packages/coding-agent/` unless stated otherwise. These three are unchanged:

* the names;
* the directories;
* the "never write to `~/.pi`" rule.

**Corrections**

1. **Environment variable names.** Pi's directory variables are `PI_CODING_AGENT_DIR` and `PI_CODING_AGENT_SESSION_DIR` (`src/config.ts:535-537`). They are not `PI_AGENT_DIR` or `PI_SESSION_DIR`. `PIGO_HOME` is the analogue of `PI_CODING_AGENT_DIR`. 0001-REPORT's configuration section has the same error.
2. **User-Agent.** go-llmprovider-sdk fixes the User-Agent on provider requests as `<app>/<ver> (<os>; <arch>) go-llmprovider-sdk/<ver>` (`identification.go:89-92`). pigo passes `WithClientInfo("pigo", <semver>)`. Provider requests therefore carry that form. Every other HTTP request pigo makes uses this record's form: MCP, `web_fetch`, `pigo update`, and the catalog refresh. The Confirmation's `httptest` provider check asserts the SDK form with `pigo/<semver>` as the product token.

**Decision change: relocated Pi installs (was "`PI_*` environment: ignored")**

The bridge reads exactly two `PI_*` variables, and only to **locate** Pi's tree:

* `PI_CODING_AGENT_DIR` (the agent directory; default `~/.pi/agent`);
* `PI_CODING_AGENT_SESSION_DIR` (sessions).

It also honours Pi's `sessionDir` setting from Pi's `settings.json`, but **as a location only**; no other Pi setting is merged. No other `PI_*` variable has any effect.

Without this, users who relocated Pi get an empty bridge. Precedence follows Pi: `PI_CODING_AGENT_SESSION_DIR` > `sessionDir` > `<agent>/sessions`. A custom session directory is **flat**, and Pi filters it by the header's `cwd` (`src/core/session-manager.ts:1794-1795,1875-1876`). The default layout is grouped by cwd: `sessions/--<cwd>--/<ts>_<uuidv7>.jsonl`.

**Bridge additions (rows added to the bridge table)**

| Pi resource | pigo behaviour |
|---|---|
| Installed Pi packages: `<agent>/npm/node_modules/*/`, `<agent>/git/**/`, and project `.pi/npm`, `.pi/git` (trusted) | Their `package.json` `pi.{skills,prompts,themes}` entries are discovered at bridge precedence. `pi.extensions` are listed and skipped (TypeScript). |
| `.pi/mcp.json`, `.pi/prompts/` | **Trusted projects only**. Pi gates both on trust, and stdio servers run commands. The original table omitted the qualifier. |
| `.pi/SYSTEM.md`, `.pi/APPEND_SYSTEM.md` (trusted) | Used only when the project has no `.pigo/` file of that name. |
| `AGENTS.MD`, `CLAUDE.MD` | Context-file candidates beside the `.md` spellings (`src/core/resource-loader.ts:185`). Applies to discovery in general, not only to the bridge. |
| `<agent>/themes/`, `<agent>/keybindings.json` | Read by the TUI at bridge precedence (0005 X4). |
| `<agent>/agents/*.md` | Read as subagent definitions (0005 X3). Pi's example reads `name`, `description`, `tools`, `model`. |
| `<agent>/trust.json` (`Record<absPath, boolean\|null>`) | Not merged. `pigo trust import --from-pi` copies decisions once. |
| `<agent>/mcp-auth.json` | Not read. `pigo mcp login` re-authenticates; `pigo auth import --from-pi --mcp` copies tokens per server on request. |

**Format rules the bridge must follow**

* **`models.json` is JSONC.** Pi parses it with `stripJsonComments` (`src/core/model-config.ts:297`). `encoding/json/v2` rejects comments, so the bridge strips them first. Values support `!cmd` (10 s, uncached), `$NAME` and `${NAME}`, with the `$$` and `$!` escapes. `mcp.json` carries `{mcpServers, autoEnableCodemode}`.
* **Session reads are tolerant and in-memory.**
  * Pi appends to session files with no lock (`appendFileSync`, `session-manager.ts:1175-1182`). A half-written last line is therefore normal. Pi skips malformed lines (`:355-370`), and the bridge does too, with a diagnostic.
  * Pi **rewrites** a file after migrating it (`:1092-1093`). pigo migrates in memory only and never rewrites a Pi file.
  * Only `version` ≤ 3 is read. Pi's agent package already defines a format 4 (`{"v":4,"kind":"header"}`); those files are skipped with a diagnostic.
* **The leaf is the last entry in file order.** Pi persists no leaf pointer (`session-manager.ts:1103-1121`).
* **`parentSession` paths are absolute paths into Pi's tree.** `pigo session import` rewrites them to the imported pigo copy when the parent was imported too. Otherwise it keeps the original path as `custom{pigo.import.parentSession}`.
* **Pi's custom entry types are preserved verbatim on import.** These are `pi.virtual-model-state` and `codemode-store`.

**Confirmation added**

* The bridge test also covers:
  * a relocated agent directory via `PI_CODING_AGENT_DIR`;
  * a flat custom session directory;
  * a `models.json` with comments;
  * a session file whose last line is truncated (it lists, and imports the complete entries).

  The `~/.pi` hash is unchanged afterwards.
* The bridge test fails on a copy that rewrites a migrated v2 file.
