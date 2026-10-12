---
status: proposed
date: 2026-10-12
decision-makers: repository owner
consulted: none
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# The configuration surface: a commented `settings.yaml` from a shipped template, edited in place so comments and layout survive, layered and trust-gated, with Kong binding flags on top

*(Revised twice on 2026-10-12, before review, at the owner's direction:*

1. *The first draft kept 0004-MADR's JSON `settings.json`. The owner then decided: "I definately want the config file to be yaml, commented, comments preserved, and a config template with sane defaults as we move forward adding additional config options", and "I am not concerned with importing pi settings".*
2. *The second revision kept `gobble config import --from-pi` in 0005-PLAN F10. The owner then directed: "drop the config import from pi".*

*Neither earlier draft was committed, so they are not kept here. Every claim below was checked at its cited line by a script (`q421_claims_0018.py`, "Confirmation").)*

## Context and Problem Statement

gobble has no configuration surface. Its `internal/config` package declares itself and nothing else ("will load layered settings", "Phase 1 declares the package only"; `internal/config/doc.go:1-5`). Two records leave the choice open:

- 0002-MADR, amendment of 2026-10-04: "The config surface is undecided and will be either a native Kong facility or a surface we write" (`0002-MADR-cli-acp-headless-mcp-v1.md:1040-1042`).
- 0004-MADR, the same amendment (`0004-MADR-go-module-architecture.md:956-958`).

Meanwhile work waits on it:

- persisting the queue modes, and the MCP stop timings (0002-PLAN);
- F1's user-facing shell settings, the bash timeout cap, and a CLI way to pass additional directories (0005-PLAN);
- F6's permission rules and trust, and F7's hooks key;
- the settings 0017-MADR D15 deferred.

The owner has put this ahead of 0017-MADR's todo work. They want:

- YAML;
- comments, to help users;
- comments that survive when gobble writes the file;
- a template with sane defaults, which grows as options are added.

This record decides how gobble finds, reads, merges, validates, overrides and edits its settings, how the template ships and grows, which keys exist when it lands, and how later phases add theirs.

### What was measured, not assumed

Read on 2026-10-12, read-only, at these commits:

- gobble `2088bc0`;
- the owner's mcp-server-magictools `e7ddbb8` and mcp-server-recall `6dab173`;
- pi `312184edb`, opencode `5683367e27`, kilocode `05bc9e15fb`, codex `16845bb8e`, goose `4617040`, grok-build `2bdd1d6`;
- Kong `v1.16.1`, from the module cache.

Two probes ran in scratch modules outside the repository, against `go.yaml.in/yaml/v3 v3.0.5` and `github.com/goccy/go-yaml v1.19.2`.

**How the owner's projects keep comments**

- **mcp-server-magictools** (`internal/config`):
  - **Template.** It ships a commented `config.yaml` as a Go string constant, `defaultConfigTemplate` (`templates.go:36-39`).
  - **Created when missing.** `New` writes that template, mode 0600, only when the file does not exist. It returns real read errors instead (`config.go:547-590`).
  - **Edits.** `ConfigStore.Apply` (`store.go:34-114`):
    - takes a lock on `<config>.lock`;
    - reads the file into a `yaml.v3` `Node` and applies a typed patch to the keys it owns: `Field[T]` with `PatchSet` or `PatchRemove` (`patch.go:3-26`; `patch_yaml.go:9-90`);
    - returns early when nothing changed;
    - validates the candidate;
    - writes through `atomicWriteFile`: temp file, chmod 0600, write, fsync, close, rename, then a sync of the parent directory (`atomic_write.go:9-61`).
  - **Comments.** The node helpers keep a key's comments by changing only its value node (`config.go:841-885`).
  - **Gaps:**
    - `setNodeValue` writes `fmt.Sprintf("%v", value)` with an empty tag (`:867-885`);
    - `setNodeSequence` replaces the list node, and the comments inside it (`:898-918`);
    - `SaveConfiguration` replaces a file that does not parse with an empty document, commented "If the existing file is corrupt, start fresh" (`:679-690`);
    - reads go through Viper with `AutomaticEnv` (`:541-545`).
- **mcp-server-recall:**
  - Its `FullConfigTemplate` is "a curated, natively-commented YAML template … Viper's automated write destroys YAML comments; writing this explicit template prevents that" (`cmd/mcp-server-recall/config_template.go:4-5`).
  - `configure` edits one key through a `yaml.v3` `Node`, forces block style and marshals again (`configure.go:53-219`, `:420-433`).
  - The template is written with four-space list indentation, the shape `yaml.v3` itself writes (`config_template.go:28-30`).

**What each way of editing keeps** (probe 1)

The input was a 30-line commented file shaped like gobble's settings:

- head comments;
- inline comments with extra spaces before them;
- six blank lines between sections;
- a nested mapping;
- a list with a comment between its items;
- a quoted `"1.10"`;
- a commented-out key.

The edit changed `compaction.reserveTokens` from 16384 to 8000.

| Method | Besides the value | Lines `diff` reports |
| :--- | :--- | :--- |
| `yaml.v3` `Node` round-trip, as magictools and recall do | 4 of 6 blank lines removed; the spaces before two inline comments collapsed | 7 |
| `goccy/go-yaml` AST (`parser.ParseComments`, `Path.ReplaceWithReader`) | the spaces before two inline comments collapsed; a trailing blank line added | 4 |
| A splice at the value's own position, from `yaml.v3`'s `Node.Line` and `Node.Column` | nothing | 1 |

Comments survived all three. Only the splice kept the file's layout.

**What `go.yaml.in/yaml/v3` does with YAML's hazards** (probe 2)

- A duplicate key is an error naming both lines: "line 2: mapping key "flag" already defined at line 1".
- `yes` and `on` decode as `true` into a `bool`. Into a `string`, `yes` stays `"yes"` and `1.10` stays `"1.10"`.
- A tab in the indentation is an error with its line: "found character that cannot start any token".
- **A second document (`---`) is ignored silently by `Unmarshal`.**
- A value node's position is exact (`line=2 column=18`), and its `LineComment` holds the inline comment.
- A syntax error gives a line, not a column: "line 2: mapping values are not allowed in this context".

**What gobble's records already decide**

- **Locations** (0003-MADR `:96`). The config directory is `$XDG_CONFIG_HOME/gobble`, or `%APPDATA%\gobble` on Windows. Its files include `mcp.json`, `models.json`, `keybindings.json` and `trust.json`. An amendment of 2026-10-12 (`:104-110`) names the settings file `settings.yaml`.
- **Overrides.** `GOBBLE_HOME` and `GOBBLE_{CONFIG,DATA,STATE,CACHE}_DIR` relocate the directories (`:110-113`). `internal/appdirs` implements the order (`internal/appdirs/dirs.go:50-132`).
- **The project directory** is `.gobble/`, loaded only for a trusted project (`:81`, `:115`).
- **0004-MADR's settings decisions** (`:361-393`):
  - a schema from the Go types, through `gobble config schema`;
  - a deep merge in which sequences and scalars replace;
  - `+name`/`-name`/`!glob` patches;
  - global-only keys;
  - project files only when trusted.

  Its JSON format and Pi key names were superseded on 2026-10-12 (`:369-375`). It left "how flags and `GOBBLE_*` environment variables bind on top of the merged result" unspecified.
- **The YAML library.** `go.yaml.in/yaml/v3 v3.0.5` is a requirement (`go.mod:19`), added for frontmatter (0005-MADR `:949`). Today `internal/frontmatter` is "the one package that imports" it (`internal/frontmatter/frontmatter.go:1-5`).
- **Kong.** Only `internal/cli` may import it (rule 9, `internal/archtest/check.go:332-342`).
- **The verbs** are `gobble config get|set|edit|path|validate|schema|import` (0005-MADR `:284`).
- **Keys records already name:**
  - `permissions.allow`, `ask` and `deny` (`:212`);
  - `defaultProjectTrust` (`:214`);
  - `hooks`, read from project settings only when trusted (`:243`);
  - `tools.bash.useClientTerminal`, default `false` (`:621`);
  - `permissions.allowBypass`, default `false` (`:623`).
- **F6 tests** that an untrusted project's settings file has no effect (0005-PLAN `:258-259`).
- **F7 runs a project's hooks only when trusted** (`:269`).
- **F10 step 5** lists `config import --from-pi` (`:357-359`), as 0003-MADR does (`:134`).

**What gobble's code has today**

- **Commands and flags:**
  - `gobble config` has one subcommand, `path [--json]` (`internal/cli/configpath.go:15-50`);
  - no flag has an `env:` tag (`internal/cli/flags.go:14-54`);
  - `--session-dir` gives a flat session directory (`internal/cli/sessions.go:39-47`).
- **How the agent's options are built.** `agentOptions` sets the version, provider, models, tools, store, MCP options and prompt options (`internal/cli/agent.go:37-49`).
  - `MCPOptions.StartupWait` is never set, so the default applies (`acpserver/mcp.go:34-36`).
  - The built-in tools get only `OutputDir` (`internal/cli/agent.go:200-207`).
  - `acpserver` applies compaction's defaults only when the whole `compaction.Settings` is zero (`acpserver/agent.go:151-153`).
- **The defaults the records park on configuration:**

  | Tunable | Value | Where | Parked by |
  | :--- | :--- | :--- | :--- |
  | queue modes | `one-at-a-time`, set by `newQueue` | `acpserver/queue.go:26-28`; `acpserver/agent.go:287`, `:731` | 0002-PLAN `:2005` |
  | MCP stop timings | stdin grace 500 ms, then a close timeout of 2 s | `mcpclient/proc.go:18-19`, `:83-85` | 0002-PLAN `:2250` |
  | `shellPath` | "": Git's bash on Windows, else `bash` on `PATH` | `tool/builtin/builtin.go:34-36` | 0005-PLAN `:959`, `:1204` |
  | `shellCommandPrefix` | "" | `tool/builtin/builtin.go:37-39` | the same |
  | bash timeout cap | 10 min | `tool/builtin/builtin.go:40-41`, `:55-61` | `:1204` |
  | `tools.bash.useClientTerminal` | `false` | `tool/builtin/builtin.go:46-49` | `:959` |
  | additional directories from the CLI | none: `acpclient.NewSessionOptions` has only `ID`, `ForkFrom` and `Name` | `acpclient/sessions.go:57-59` | `:959` |
  | `images.autoResize` | no consumer; waits on `llm` carrying images | — | `:959` |

- **Defaults not yet parked:**
  - The MCP start-up wait, 10 s (`acpserver/mcp.go:19`).
  - A new session's thinking level, `off` (`acpserver/agent.go:287`). A loaded or resumed session takes its own from its history. The levels are `off`, `minimal`, `low`, `medium`, `high` and `xhigh` (`acpserver/config.go:25`).
  - Compaction's `keepRecentTokens` of 20000, used by `compaction.Prepare` (`compaction/compaction.go:24`, `:62`; `acpserver/compact.go:65`).
  - Compaction's `reserveTokens`, which "leaves room for the reply in the automatic trigger (0005-PLAN F5)" (`compaction/compaction.go:15-16`). Automatic compaction is not built, so nothing reads it.
  - The editor chain: `GOBBLE_EDITOR`, `VISUAL`, `EDITOR`, then `notepad` or `vi` (`internal/cli/editor/external.go:48-58`), used by `/edit` through `editor.Compose` (`internal/cli/slash.go:27-28`).
- **Writes.**
  - The `mcp.json` editor keeps member order and indentation, and renames a temp file into place, with no lock (`mcpclient/file.go:16-198`).
  - The session store's lock is try-only, with a PID in the lock file (`session/jsonl/lock.go:21-38`), and OS-specific files beside it (`lock_unix.go`, `lock_windows.go`, `lock_other.go`).
- **Tests pin the defaults,** for example:
  - `compaction/compaction_test.go:90-97`;
  - `acpserver/queue_test.go:20-29`;
  - `mcpclient/proc_test.go:115`;
  - `tool/builtin/shell_test.go:77`.

**Kong** (`v1.16.1`)

- `kong.JSON` resolves one flag by its name (`resolver.go:30-62`).
- `kong.Configuration` adds one such resolver per file (`options.go:450-480`).
- `kong.DefaultEnvars` gives every flag a variable (`:520-545`).

No merge, trust gate, write path or schema exists in it, and a setting with no flag cannot reach code through it.

**The other harnesses**

- **Pi** (`packages/coding-agent/src/core/settings-manager.ts`):
  - its settings are strict JSON, with "Settings files are not validated" (`:225`, `:485`);
  - its merge is deep, with project values replacing (`:193-210`);
  - project settings load only when trusted (`:470-475`);
  - `defaultProjectTrust` is read from the global file alone (`:1097-1100`);
  - a write re-reads the file under a lock, then writes with `writeFileSync`, which is not atomic (`:336-350`).
- **opencode** writes `cli.json` under a lock, through a JSONC patch (`jsonc-parser` `modify`/`applyEdits`), to a temp file renamed into place (`packages/cli/src/config/config.ts:43-46`, `:56-59`, `:97-99`). That is a text edit guided by a parser, the idea of the splice above. Its `{env:VAR}` becomes `""` when the variable is unset (`packages/core/src/config/variable.ts:28-31`).
- **kilocode** refuses `{env:}` in an untrusted project: "environment references are not allowed in project config" (`packages/opencode/src/config/variable.ts:60-69`).
- **codex:**
  - loads an untrusted project's layers disabled, each with a `disabled_reason` (`codex-rs/config/src/loader/mod.rs:1113-1156`);
  - strips base URLs, providers, `notify` and `otel` even from trusted project files (`:89-101`);
  - fails a test when its committed schema drifts: "Run `just write-config-schema` to overwrite with your changes" (`codex-rs/core/src/config/schema_tests.rs:13-36`).
- **goose** replaces a corrupt file on its write path, logging "is corrupt: … Starting fresh" (`crates/goose/src/config/base.rs:506-518`).
- **grok-build** keeps trust in its home directory: "a cloned repo can never ship a … `trusted_folders.toml` that self-trusts its own checkout" (`crates/codegen/xai-grok-permission-rules/src/trust.rs:17-22`). It refuses roots that are too broad (`:436-438`).

### Findings

- **F1.** Kong's facility cannot be the configuration surface: it fills flags only. It can bind flags and their variables on top of the merged settings, which is the half 0004-MADR left open. So the answer to "native Kong facility or a surface we write" is both, split at that line.
- **F2.** The owner's YAML pattern keeps comments, but not layout. A `yaml.v3` round-trip removed four of six blank lines and collapsed inline-comment spacing on a one-value edit. `goccy/go-yaml` does better, but still rewrites spacing, and would be a new module.
- **F3.** A splice at `yaml.v3`'s node positions keeps the file byte for byte apart from the edited value. Parsing the result again proves the edit.
- **F4.** YAML has hazards JSON does not:
  - the reader must refuse a second document itself;
  - a written string must be quoted when its plain text would read back as another type.
- **F5.** Every project surveyed has a write path that can lose data:
  - magictools' `SaveConfiguration` and goose replace a file that does not parse;
  - magictools' sequence edit drops the comments inside a list;
  - Pi's write is not atomic.

  magictools' `ConfigStore.Apply` (lock, validate the candidate, temp file, fsync, rename, directory sync) is the discipline to keep.
- **F6.** A template that lists every key with its comment and default is the owner's "sane defaults". Two things follow:
  - **It must not drift** from the Go types as keys are added.
  - **Active keys pin today's defaults** in every user's file; magictools and recall write active keys. Commented-out keys show the default and leave it to gobble.
- **F7.** Trust is F6's, and not built. Loading a project's settings before F6 would trust every project, and ignoring them silently would hide why a setting has no effect.
- **F8.** Some keys are dangerous from a project file even when the project is trusted: what runs every command, where sessions are written, and what decides trust or bypass.
- **F9.** Mapping every key to an environment variable, as Viper's `AutomaticEnv` does in both owner projects, invites collisions. A one-run override is still needed, and gobble's `-c` is `--continue`.
- **F10.** The org rule (LKQ ADR 007) keeps secrets out of configuration files. Credentials belong in `auth.json` (F4-3).
- **F11.** Of the parked and unparked defaults, eight have consumers today. `compaction.reserveTokens` and `images.autoResize` have none until F5 and image support land, so they are not keys yet. `--add-dir` needs the CLI client to send additional directories, which it cannot today.
- **F12.** The owner is not concerned with importing Pi's settings, and has dropped it. Three records and one PLAN step name `config import --from-pi`.
- **F13.** The records disagree in three places this surface touches:
  - 0003-MADR `:111` names `PI_AGENT_DIR`, which its own erratum (`:266`) corrects to `PI_CODING_AGENT_DIR`;
  - `--offline`'s phase is F4 in `internal/cli/flags.go:67` and F10 in 0005-PLAN `:708`, but X3 in its latest entry, F4 made executable (`:3277`);
  - 0005-PLAN `:826` cites 0008-MADR for "the config surface is undecided", which 0008-MADR does not say; 0002-MADR and 0004-MADR do.

## Decision Drivers

- **The owner's direction:** YAML, commented, comments kept, a template with sane defaults that grows with the options, and no Pi settings import.
- **A write keeps the user's file:** its comments, blank lines, order and spacing, apart from the change asked for. It never replaces a file that does not parse.
- **A missing setting changes nothing** (F11).
- **Trust is honest** (F7), and dangerous keys never come from a project (F8).
- **Errors name the file and line,** and a key's path.
- **Module rules hold:** no Kong in `internal/config`, and no new module.
- **No secret in any settings file** (F10).
- **Small and closed:** no profiles, no system layer, no watcher, until a record asks.

## Considered Options

The format is the owner's decision. The options are about how comments and layout survive an edit.

- **A — A splice guided by `yaml.v3` positions,** with `go.yaml.in/yaml/v3`, already a requirement:
  - `yaml.v3` reads the file into a `Node` tree, for values and positions;
  - every edit is applied to the original bytes;
  - the result is parsed again and checked before it is written;
  - the template lists every key commented-out with its default.
- **B — The owner's current pattern:** a `yaml.v3` `Node` round-trip with typed patches (magictools' `ConfigStore`), and a template written in `yaml.v3`'s output shape so re-marshalling changes less (recall).
- **C — `github.com/goccy/go-yaml`'s comment-aware AST,** edited by path and printed.
- **D — Viper for reading, with node edits for writing,** as both owner projects do.
- **E — Regenerate the file from the template on every write,** filling in the user's values.

## Decision Outcome

Chosen: **A**. In the probe it kept the file exactly apart from the edited value, comments, blank lines and spacing included (F2, F3). It adds no module. It keeps magictools' write discipline, and closes that code's gaps (F5).

### The decisions

- **D1. The split.**
  - `internal/config` reads, merges, validates, explains and edits settings, and owns the template. It imports no CLI library.
  - `internal/cli` resolves the directories, asks `internal/config` for the effective settings, and applies flags and their named environment variables on top, with Kong's `env:` tags.
  - Kong's `Configuration` resolver is not used.
  - 0002-MADR's and 0004-MADR's "undecided" amendments get dated notes pointing here.
- **D2. The files and layers, lowest precedence first:**
  1. **Defaults:** the code's, unchanged.
  2. **Global:** `<config>/settings.yaml`.
  3. **Project:** `<root>/.gobble/settings.yaml`, where `<root>` is the git top-level of the working directory, or the working directory outside a repository. The walk never goes above it.
  4. **Session overrides:** `--set` (D10).
  5. **Flags and their environment variables** (D11).

  - **One file name per layer:** `settings.yaml`. Never `.yml`, and no JSON fallback.
  - **`mcp.json` and `models.json` stay JSON,** because other tools share their formats, and `gobble mcp add` writes `mcp.json`.
  - **`trust.json` and `auth.json` stay JSON,** because gobble writes them and users do not edit them.
  - **Keybindings will be `keybindings.yaml`** when F9 builds them, because they are a gobble-native file users edit.
- **D3. YAML, read strictly,** with `go.yaml.in/yaml/v3`. `internal/frontmatter`'s package comment, which calls it "the one package that imports" the library, is corrected in the same change.
  - **One document.** A second document is an error (F4).
  - **Errors, with the file and line:** a duplicate key, a tab in the indentation, or a syntax error.
  - **The root** must be a mapping. An empty file, or one holding only comments, is an empty mapping.
  - **Scalars** decode into their key's Go type. A boolean key accepts what `yaml.v3` reads as a boolean, including `yes` and `on`. A string key keeps its text as written.
  - **Anchors and aliases** are read. An edit that would change an aliased value is refused (D7).
  - **CRLF** line endings are read as LF. An edit keeps the file's own line endings.
- **D4. Merge** follows 0004-MADR:
  - mappings merge deeply;
  - sequences and scalars from a higher layer replace a lower one's;
  - `null` in a higher layer clears the key back to the layer below.

  The `+name`/`-name`/`!glob` patches apply only to sequences the schema marks as patchable; none of this record's keys is one. The merge works on the parsed trees, so every effective value keeps its file, its line and its layer, for `--origin` and for diagnostics.
- **D5. Trust gates the project layer, and says why.**
  - The project file is always read and validated, but applied only when `internal/config` is told the project is trusted.
  - Until F6 builds trust, nothing is trusted. The layer is disabled with the reason in D7.
  - `gobble config get --origin` and `gobble config validate` show a disabled layer, its reason, and each key it would have set.
  - F6 supplies the decision as one function value, `func(root string) bool`. This record builds no trust store.
- **D6. Global-only keys.** A project file never sets these, trusted or not; a project value is reported and ignored:
  - `sessionDir`, `shellPath` and `shellCommandPrefix` (this record's keys);
  - `defaultProjectTrust` and `permissions.allowBypass` (F6's, when added).

  The schema carries the mark, and each later key declares its own. `hooks` is not global-only: 0005-MADR and 0005-PLAN already run a trusted project's hooks (`0005-MADR-v1-feature-scope.md:243`; `0005-PLAN-v1-feature-scope.md:269`).
- **D7. Validation.** Diagnostics have these exact forms. `<path>` is the key's dotted path, such as `tools.bash.maxTimeoutMs`.

  | Case | Diagnostic | Effect |
  | :--- | :--- | :--- |
  | syntax error, tab or duplicate key | `<file>:<line>: <yaml.v3's message>` | the file reads as empty for the run; never written while it fails |
  | second document | `<file>:<line>: settings are one YAML document; a second one starts here` | the same |
  | root not a mapping | `<file>:<line>: settings must be a mapping of keys` | the same |
  | wrong type | `<file>:<line>: <path>: want <type>, have <the YAML kind>` | the key falls back to the layer below |
  | out of range, or not in its set | `<file>:<line>: <path>: must be <the rule>` | the same |
  | unknown key | `<file>:<line>: <path>: not used by this gobble` | ignored, and kept in the file |
  | global-only key in a project | `<file>:<line>: <path>: only the global settings file can set this` | ignored |
  | credential-shaped key (D15) | `<file>:<line>: <path>: credentials go in auth.json, not settings.yaml` | an error; ignored |
  | project layer before trust | `<file>: project settings apply once the project is trusted (0005-PLAN F6)` | the layer is disabled |
  | edit of an aliased value | `<path>: its value is a YAML alias; edit the anchor` | the edit is refused |
  | edit failing its check (D8) | `<file>: the edit to <path> could not be applied safely; the file is unchanged` | the edit is refused |

  - Warnings go to the log, and in the CLI to stderr once per run. `gobble acp` writes only to its log, since stdout is the protocol.
  - `gobble config validate [--strict]` prints every diagnostic, one per line, sorted by file and line. It exits 1 on any error, and with `--strict` on an unknown key too, and 0 otherwise.
  - A diagnostic prints a key's path and line, never its value.
- **D8. Edits are splices, checked before they are written.** An edit is a path and a value, or a removal. `internal/config` applies it to the file's bytes, in one of these cases:
  1. **An active scalar.** Replace exactly the value's bytes at its node's line and column. The key, every comment, the inline comment and the spacing before it are untouched.
  2. **A key that is commented out.** The line reads `#`, one space, then the key at its nesting indentation, a colon and a value, as the template writes every key. Remove the `#` and its one space from that line, and from each commented-out parent line above it in the same block, then set the value as in case 1. The prose comments above stay.
  3. **An absent key.** Insert `key: value` at the end of its parent mapping, at the indentation of its siblings, or two spaces deeper than the parent when it has none. Missing parents are inserted the same way.
  4. **A sequence or mapping value.** Replace the value's line range with block-style YAML at the key's indentation plus two spaces. The comments above the key stay. This is the only edit that rewrites more than one line of value.
  5. **A removal (`unset`).**
     - A key the template has: turn its lines back into the template's commented-out form for that key.
     - Any other key: delete its lines.

     The comments above it stay either way.

  Two rules apply to every case:

  - **Scalars are written plain** only when the plain text reads back as the same type and value. Otherwise they are double-quoted, so a string `1.10` is written `"1.10"`.
  - **The check.** The edited text is parsed again. It is written only if:
    - it parses as one mapping document;
    - the edited path reads back as the value set, or as absent after a removal;
    - every other path in the file reads back unchanged.

    Otherwise the edit is refused with D7's diagnostic, and the file is untouched.

  **The write:**
  1. take `<file>.lock` through `internal/filelock` (D18);
  2. re-read the file, apply the edit, and check it;
  3. write a temp file beside it, mode 0600, or the owner-only DACL on Windows through `internal/appdirs`'s helpers;
  4. fsync it, close it, rename it over the file, and sync the directory (magictools' `atomicWriteFile`).

  A file that does not parse is refused with its diagnostic and never replaced. A missing config directory is created with mode 0700. An existing one is used as it is.
- **D9. The template is the documentation, and it grows with the keys.**
  - **It is embedded.** `internal/config/settings.yaml` is embedded with `//go:embed`, and is exactly the text below.
  - **Creation.**
    - The global file is created from the template when it is missing, by:
      - the first run that loads settings (the default chat, `acp` and `mcp`);
      - `gobble config init`;
      - any `gobble config` command that writes.
    - These never create it, being read-only: `version`, `completion`, `config path`, `config get`, `config validate` and `config schema`. That keeps 0008-MADR's read-only rule for completion.
    - A project file is created only by `gobble config init --project`.
  - **Growth.** `gobble config update [--project] [--dry-run]` adds the template block of each template key the file lacks, active or commented-out:
    - after the block of the nearest earlier template key the file has, or after its section heading;
    - without changing any existing line, and printing what it added;
    - `--dry-run` prints the added lines and writes nothing;
    - it never runs on its own.
  - **Layout.** Every key appears once, commented-out, showing its default. Prose sits above each block, and no prose line has the form `name: value`, so D8's case 2 cannot mistake one for a key. A value of `""` for a string key means unset, so the code's default applies. Nested keys are written with their parents, all commented-out.

  ```yaml
  # gobble settings
  #
  # This file is optional, and so is every key in it. A key that stays
  # commented out takes the default shown, and a later gobble may change
  # that default. Remove the "# " in front of a key, and of the lines
  # it sits under, to set it.
  #
  # Where settings come from, lowest first:
  #   1. gobble's defaults
  #   2. this file
  #   3. a trusted project's .gobble/settings.yaml
  #   4. --set path=value on the command line, for one run
  #   5. command-line flags
  #
  # gobble config get --origin   shows what applies, and where it came from
  # gobble config validate       checks this file
  # gobble config update         adds the settings a newer gobble knows, as comments
  # gobble config schema         prints a JSON Schema for editors
  #
  # Credentials never go here. They belong in auth.json.

  # ---------------------------------------------------------------------
  # Sessions
  # ---------------------------------------------------------------------

  # The thinking level a new session starts with: off, minimal, low,
  # medium, high or xhigh. A resumed session keeps its own, and /thinking
  # changes it in a session.
  # defaultThinkingLevel: off

  # How messages queued while a turn runs reach the model.
  #   one-at-a-time   one queued message before each model call
  #   all             every queued message at once
  # Steering messages arrive mid-turn; follow-ups wait until the turn
  # would end. A client can change either for one session.
  # steeringMode: one-at-a-time
  # followUpMode: one-at-a-time

  # The directory session files are kept in, flat, as --session-dir
  # keeps them. An absolute path, and ~ is your home. Empty keeps them in
  # gobble's data directory, by working directory. Only this file can set
  # it, never a project's.
  # sessionDir: ""

  # ---------------------------------------------------------------------
  # Compaction
  # ---------------------------------------------------------------------

  # The recent conversation, in tokens, that /compact keeps word for
  # word. What comes before it is summarised.
  # compaction:
  #   keepRecentTokens: 20000

  # ---------------------------------------------------------------------
  # Tools
  # ---------------------------------------------------------------------

  # The bash the bash tool runs. Empty finds one: Git's bash on Windows,
  # bash on PATH elsewhere. ~ is your home. Only this file can set it.
  # shellPath: ""

  # A line run before every bash command, such as "set -euo pipefail".
  # Only this file can set it.
  # shellCommandPrefix: ""

  # Whether bash runs in the ACP client's terminal when the client offers
  # one, so the user can watch it there. Off, because a phone client
  # shows only "terminal <id>". And the longest timeout, in milliseconds,
  # a bash command may ask for: 600000 is 10 minutes.
  # tools:
  #   bash:
  #     useClientTerminal: false
  #     maxTimeoutMs: 600000

  # ---------------------------------------------------------------------
  # MCP servers (the servers themselves are listed in mcp.json)
  # ---------------------------------------------------------------------

  # In milliseconds: how long the first prompt waits for servers to
  # connect, after which a server still connecting joins a later prompt;
  # how long a stdio server has to exit once its input closes; and how
  # long after that before it is killed.
  # mcp:
  #   startupWaitMs: 10000
  #   stopGraceMs: 500
  #   stopTimeoutMs: 2000

  # ---------------------------------------------------------------------
  # Editor
  # ---------------------------------------------------------------------

  # The editor /edit and gobble config edit open. GOBBLE_EDITOR wins over
  # it, and it wins over VISUAL and EDITOR. Empty uses VISUAL, then
  # EDITOR, then notepad on Windows or vi elsewhere.
  # externalEditor: ""
  ```

  **Tests hold the template to the code:**
  - it parses, and is all comments;
  - every schema key appears exactly once, commented-out, with at least one prose line in its block;
  - the default each key shows equals its Go default, or `""` where the default is computed;
  - uncommenting every key gives effective settings equal to the defaults;
  - no template key is unknown to the schema;
  - no prose line has the form `name: value`.

  A key added to the Go types without its template block fails the build.
- **D10. One-run overrides: `--set path=value`, repeatable,** on the root, so `gobble acp --set steeringMode=all` works.
  - The value is parsed as one YAML scalar or flow value, then checked against the key's type. It lands in the session layer and is never written.
  - An unknown path or a bad value is a usage error, exit 2.
  - Global-only keys are accepted, since a user's command line is not a project file.
  - `-c` stays `--continue`.
- **D11. Flags and environment.**
  - A flag beats every file; its environment variable sits just below the flag.
  - Only variables records name are bound, as Kong `env:` tags: `GOBBLE_EDITOR` (`externalEditor`'s place in D9's chain), and `GOBBLE_OFFLINE` with `--offline` when that flag lands.
  - The directory variables are resolved before settings, by `internal/appdirs`.
  - No setting maps to an environment variable by default (F9). `--set` covers the rest.
  - `--session-dir` beats `sessionDir`. `PI_CODING_AGENT_SESSION_DIR` does not touch gobble's sessions: 0003-MADR has the read-only Pi bridge read it only to locate Pi's own sessions (`:270-278`), and no code reads it today.
- **D12. The commands:**
  - **`gobble config get [path] [--origin] [--json]`:** the effective value, or the whole effective document as YAML, with `--json` for scripts. `--origin` adds each value's file, line and layer, and the disabled layers with their reasons.
  - **`gobble config set path value [--project]`** and **`gobble config unset path [--project]`:** D8, on the global file, or the project's with `--project`.
    - The value is checked before anything is written.
    - A global-only key is refused with `--project`.
    - Writing a project file needs no trust; applying it does.
    - An unknown path or a bad value exits 2, and a write failure 1.
  - **`gobble config edit [--project]`:** creates the file from the template if missing, opens it in D9's editor chain, and validates it after the editor exits, printing the diagnostics.
  - **`gobble config init [--project]`:** creates the file from the template. If the file exists, it exits 1 and leaves it alone.
  - **`gobble config update [--project] [--dry-run]`:** D9.
  - **`gobble config path [--json]`:** today's four lines, plus `settings:` (the global file) and `project:` (the project file, or none outside a project).
  - **`gobble config validate [--strict]`:** D7.
  - **`gobble config schema`:** the JSON Schema that `github.com/google/jsonschema-go` generates from the `Settings` type, with each key's description, default, enum, range and global-only mark. It is committed at `internal/config/testdata/settings.schema.json`. A test fails when the generated schema differs, and names `go run ./internal/config/genschema` to regenerate it (codex).
- **D13. `gobble config import --from-pi` is dropped,** by the owner's direction. Pi's `settings.json` is never merged into, or converted to, gobble's settings. 0005-PLAN F10's read-only bridge may still read Pi's `sessionDir` from it, only to locate Pi's own sessions, as 0003-MADR has it (`:276`). These get dated notes striking the import, in this change:
  - 0003-MADR's bridge row (`:134`) and its consequence (`:174`);
  - 0004-MADR (`:363-366`, `:764`);
  - 0005-MADR's verbs (`:284`) and "No phone-home" row (`:262`);
  - 0005-PLAN F10 step 5 (`:357-359`).

  `auth import --from-pi` and `trust import --from-pi` are other records' decisions and stay.
- **D14. The keys this record lands.** Each has a consumer in the code today, and its default is today's:

  | Key | Type | Default | Consumer | Scope |
  | :--- | :--- | :--- | :--- | :--- |
  | `defaultThinkingLevel` | `off` \| `minimal` \| `low` \| `medium` \| `high` \| `xhigh` | `off` | a new session's thinking (`acpserver/agent.go:287`) | any |
  | `steeringMode` | `one-at-a-time` \| `all` | `one-at-a-time` | a new or opened session's queue (`acpserver/queue.go:26-28`) | any |
  | `followUpMode` | `one-at-a-time` \| `all` | `one-at-a-time` | the same | any |
  | `sessionDir` | absolute path, `~` expanded; "" unset | unset: `<data>/sessions`, by working directory | the store, flat as with `--session-dir` (`internal/cli/sessions.go:39-47`) | global-only |
  | `compaction.keepRecentTokens` | integer ≥ 1 | 20000 | `compaction.Prepare` (`acpserver/compact.go:65`) | any |
  | `shellPath` | path, `~` expanded; "" unset | unset: today's detection | `builtin.Options.ShellPath` | global-only |
  | `shellCommandPrefix` | string; "" none | none | `builtin.Options.ShellCommandPrefix` | global-only |
  | `tools.bash.useClientTerminal` | boolean | `false` | `builtin.Options.UseClientTerminal` | any |
  | `tools.bash.maxTimeoutMs` | integer ≥ 1 | 600000 | `builtin.Options.MaxTimeout` | any |
  | `mcp.startupWaitMs` | integer ≥ 0 | 10000 | `acpserver.MCPOptions.StartupWait` | any |
  | `mcp.stopGraceMs` | integer ≥ 0 | 500 | a stdio server's stdin grace (`mcpclient/proc.go:83`) | any |
  | `mcp.stopTimeoutMs` | integer ≥ 0 | 2000 | a stdio server's close timeout (`mcpclient/proc.go:85`) | any |
  | `externalEditor` | command line; "" unset | unset | the editor chain, after `GOBBLE_EDITOR` | any |

  - **Names** are lower camelCase, gobble's own. Several match Pi's, which is convenient but no longer a constraint.
  - **Queue modes** are per-session defaults. `_gobble/set_steering_mode` and `_gobble/set_follow_up_mode` still change one session and persist nothing; their frozen shapes stay (0002-PLAN Phase 7).
  - **`--add-dir <path>`** is the CLI's way to pass additional directories (0005-PLAN `:959`). It is a repeatable flag, not a key, since a directory list belongs to one run. Each path is made absolute against the working directory. `acpclient.NewSessionOptions` gains `AdditionalDirectories`, which `NewSessionWith` sends. `Conn.ResumeSession` gains a parameter carrying them, which its two callers pass (`internal/cli/sessions.go:61`; `internal/cli/slash.go:160`).
- **D15. No secrets.**
  - The schema has no credential key.
  - A key in any layer whose last path segment ends in `key`, `token`, `secret` or `password`, ignoring case and an `_` or `-` before the suffix, is D7's error.
  - `--set` and `config set` refuse such a path before anything is written.
- **D16. Settings are read at process start, and when a session starts** (`session/new`, `load`, `resume`). There is no file watcher. `/reload` (0005-MADR `:315`, 1.0, with no plan step yet) is not built here; when it is, it re-reads through this mechanism.
- **D17. How the settings reach the code.** `internal/cli` builds each consumer's value from the effective settings, with the code's default for every unset key:
  - **`agentOptions`** (`internal/cli/agent.go:37-49`) sets:
    - `Compaction` to a complete `compaction.Settings`: `keepRecentTokens`, and `compaction.DefaultSettings.ReserveTokens`, so the all-zero check at `acpserver/agent.go:151-153` never mixes defaults;
    - `MCP.StartupWait`;
    - new `acpserver.Options` fields: `SteeringMode`, `FollowUpMode` and `Thinking`, read where `newQueue` and `thinkingOff` are today (`acpserver/agent.go:287`, `:731`; `acpserver/queue.go:26-28`). A loaded or resumed session keeps its history's thinking.
  - **`builtinTools`** (`internal/cli/agent.go:200-207`) sets `ShellPath`, `ShellCommandPrefix`, `MaxTimeout` and `UseClientTerminal` on `builtin.Options`, beside `OutputDir`.
  - **`mcpclient.Options`** (`mcpclient/conn.go:39-51`) gains `StopGrace` and `StopTimeout`, whose zero values are today's constants (`mcpclient/proc.go:18-19`). `acpserver.MCPOptions` carries them to `mcpclient.Start` (`acpserver/mcp.go:123`).
  - **The store** (`internal/cli/agent.go:89-102`; `internal/cli/sessions.go:39-47`): `sessionDir` gives the flat store `--session-dir` gives, and the flag wins.
  - **The editor**: `editorCommand` (`internal/cli/editor/external.go:48`) takes the `externalEditor` value and tries it after `GOBBLE_EDITOR`.
- **D18. Packages.**
  - **`internal/config`** holds `Settings`, its schema, the reader, the merge, the diagnostics, the D8 editor and the embedded template.
  - **`internal/config/genschema`** is the generator `go run` uses.
  - **`internal/filelock`** holds the lock files moved from `session/jsonl` (`lock.go`, `lock_unix.go`, `lock_windows.go`, `lock_other.go`), with:
    - `TryLock`, today's behaviour, which `session/jsonl` keeps using;
    - `Lock`, which retries 10 times, 20 ms apart, as Pi does, for settings and for `mcp.json`.
  - **`mcpclient/file.go`** keeps its JSON editor, and takes `Lock` and an fsync before its rename.
  - No package outside `internal/cli` imports Kong, and none imports a new module.
- **D19. Later phases add their keys,** each in its own change, with the Go types, the schema, its drift test and the template block:
  - F4: `defaultProvider`, `defaultModel` and the model scope;
  - F5: `retry.*`, `compaction.enabled` and `compaction.reserveTokens`;
  - F6: `permissions.*` and `defaultProjectTrust`, and D5's trust function;
  - F7: `hooks`;
  - F9: `theme`, the TUI keys and `keybindings.yaml`;
  - F10: `bridge.pi`, which 0003-MADR names to turn the Pi bridge off (`:138`);
  - when `llm` carries images: `images.autoResize`;
  - 0017-MADR: the reminder settings its D15 deferred.

  D9's tests make each change carry its template block.
- **D20. The record contradictions of F13 are corrected now,** by dated notes in this change:
  - 0003-MADR `:111`'s `PI_AGENT_DIR` reads `PI_CODING_AGENT_DIR`, as its erratum says;
  - 0005-PLAN `:708` reads that `--offline`'s phase is X3, per `:3277`. `internal/cli/flags.go:67`'s comment is corrected in this record's plan, since it is code;
  - 0005-PLAN `:826`'s citation reads 0002-MADR and 0004-MADR.
- **D21. These are not done here, and why:**
  - **Viper** (option D): its `AutomaticEnv` maps every key to a variable (F9), and gobble reads YAML itself.
  - **`goccy/go-yaml`** (option C): a new module that still rewrites spacing.
  - **Profiles, a system or managed layer, a file watcher, and settings over ACP.** A managed layer, for an organisation's policy, belongs with F6's rules.
  - **`{env:}` and `{file:}` substitution in settings.** `models.json`'s `!cmd` and `$VAR` stay F4-2's.

### Consequences

- Good: users get one YAML file that documents itself. Every option appears with its default and its explanation, and new options arrive through `gobble config update`, which never rewrites a line the user wrote.
- Good: an edit by gobble changes exactly what was asked. Comments, blank lines and spacing survive, where the probe showed a `yaml.v3` round-trip does not.
- Good: the template cannot drift from the code, because its tests fail the build.
- Good: every parked tunable with a consumer has a key, with no behaviour change for a user who sets nothing.
- Good: no write can lose the user's file. Each edit is proved by parsing again, a file that does not parse is refused, and `mcp.json`'s editor gains a lock.
- Bad: the D8 editor is code gobble owns, with five cases each tested byte for byte. The check after each edit bounds the risk.
- Bad: the config directory holds YAML settings beside JSON `mcp.json`, `models.json` and state files (D2).
- Bad: until F6, project settings are read and reported but never applied.
- Bad: a settings change reaches new sessions, not a running one, until `/reload` exists.
- Neutral: Pi's settings are never imported, by the owner's choice. 0004-MADR's merge, schema, global-only and trust decisions stand.

### Confirmation

This record's claims:

```text
python -I q421_claims_0018.py             -> every claim ok (a quoted substring at each cited file:line)
python -I q421_claims_0018.py --invert    -> every claim fails (each quote altered), so the check can fail
```

Its implementation:

```text
go test ./internal/config/...                               -> ok:
                                                                 layers and precedence; merge incl. null-clears;
                                                                 a disabled project layer with D7's reason; global-only keys ignored from a project;
                                                                 a second document, a duplicate key, a tab and a non-mapping root, each at file:line;
                                                                 type and range errors by path and line; unknown keys kept; credential-shaped keys refused;
                                                                 schema drift; the template tests of D9
go test ./internal/config -run 'TestEdit'                   -> ok:
                                                                 each D8 case changes only its bytes (a byte diff against the expected file);
                                                                 "1.10" written quoted; an aliased value refused;
                                                                 a planted bad splice refused, with the file untouched;
                                                                 CRLF files kept CRLF; the lock retried; temp, fsync, rename
go test ./internal/filelock/... ./session/jsonl/... ./mcpclient/... -> ok; the store's and mcp.json's behaviour unchanged
go test ./internal/cli -run 'TestConfig'                    -> ok:
                                                                 get/--origin/--json, set/unset/--project, edit, init, update/--dry-run,
                                                                 path, validate/--strict, schema, --set, --add-dir;
                                                                 read-only commands create no file
go test ./acpserver ./tool/builtin ./compaction ./mcpclient -> ok, every existing default test unchanged
go test ./acpserver -run '^TestSession$'                    -> ok; session.golden unchanged
make archtest                                               -> ok; a scratch internal/config importing kong fails rule 9
make preflight                                              -> preflight passed (Windows and WSL)
```

Each new test is seen failing first, on a scratch copy:

- the trust gate removed;
- a global-only key let through;
- the post-edit check skipped, with a bad splice planted;
- the lock skipped;
- an in-place write;
- a template key removed;
- a template default changed;
- the second-document check removed;
- the committed schema changed by one byte.

## Pros and Cons of the Options

### A — A splice guided by `yaml.v3` positions (chosen)

- Good, because the probe changed exactly one line for a one-value edit, keeping comments, blank lines and spacing (F3).
- Good, because `go.yaml.in/yaml/v3` is already a requirement.
- Good, because parsing again after each edit turns any splice mistake into a refused write, not a damaged file.
- Good, because a commented-out template line becomes the active key in place, so the explanation stays beside the setting.
- Bad, because it is more code than a re-marshal: five edit cases, each tested byte for byte.

### B — The owner's current pattern: a `Node` round-trip with typed patches

- Good, because it is proven in the owner's own projects, is the least new code, and keeps every comment. This is the strongest argument for it.
- Good, because typed patches (`Field[T]`) state an edit's intent plainly.
- Bad, because every write reflows the file. The probe lost four of six blank lines and inline-comment spacing (F2). recall avoids part of this only by writing its template in `yaml.v3`'s own shape, which a user's hand edits do not keep.
- Bad, because a list edit replaces the list's node and its comments, and a `%v` value with an empty tag can change a string's type (F4, F5).

### C — `goccy/go-yaml`'s AST

- Good, because it kept blank lines in the probe, and it edits by path.
- Bad, because it still rewrote inline-comment spacing and added a trailing line.
- Bad, because it adds a module beside the `yaml.v3` gobble already has.

### D — Viper for reading, node edits for writing

- Good, because it is the owner's stack in two projects.
- Bad, because `AutomaticEnv` maps every key to a variable (F9), Viper folds key case, and it brings a large dependency tree. gobble would still need its own write path.

### E — Regenerate from the template on every write

- Good, because every write gives a clean file with current documentation.
- Bad, because the user's own comments and order are lost on every write, which is what the owner wants to avoid.

## More Information

### Evidence index

Every row is checked by `q421_claims_0018.py`, which reads a quoted substring at each cited line.

| Claim | Source |
| :--- | :--- |
| `internal/config` is a placeholder | `internal/config/doc.go:1-5` |
| The surface is undecided | `0002-MADR-cli-acp-headless-mcp-v1.md:1040-1042`; `0004-MADR-go-module-architecture.md:956-958` |
| magictools' template, creation, patches, store and atomic write | mcp-server-magictools `internal/config/templates.go:36-39`; `config.go:547-590`; `patch.go:3-26`; `patch_yaml.go:9-90`; `store.go:34-114`; `atomic_write.go:9-61` |
| magictools' gaps | `internal/config/config.go:541-545`, `:679-690`, `:867-885`, `:898-918` |
| recall's template and edit | mcp-server-recall `cmd/mcp-server-recall/config_template.go:4-5`, `:28-30`; `configure.go:53-219`, `:420-433` |
| Round-trip fidelity of three methods | probe 1: `probe-yaml/main.go`, `sample.yaml`, `probe-yaml.txt`, and `diff` of its three outputs |
| yaml.v3's handling of hazards | probe 2: `probe-yaml2/main.go`, `probe-yaml2.txt` |
| The decided locations, overrides, project directory and trust gate | `0003-MADR-gobble-product-identity.md:81`, `:96`, `:104-108`, `:110-115` |
| 0004-MADR's settings decisions and their 2026-10-12 note | `0004-MADR-go-module-architecture.md:361-393` |
| The YAML library and its scope | `go.mod:19`; `0005-MADR-v1-feature-scope.md:949`; `internal/frontmatter/frontmatter.go:1-5` |
| Rule 9; the verbs; the named keys; F6's and F7's tests; F10 step 5 | `internal/archtest/check.go:332-342`; `0005-MADR-v1-feature-scope.md:212`, `:214`, `:243`, `:284`, `:621`, `:623`; `0005-PLAN-v1-feature-scope.md:258-259`, `:269`, `:357-359` |
| The CLI's config, flags, store and agent options | `internal/cli/configpath.go:15-50`; `internal/cli/flags.go:14-54`; `internal/cli/sessions.go:39-47`; `internal/cli/agent.go:37-49`, `:89-102`, `:200-207` |
| The defaults and their consumers | `acpserver/queue.go:26-28`; `acpserver/agent.go:151-153`, `:287`, `:731`; `acpserver/config.go:25`; `acpserver/mcp.go:19`, `:34-36`, `:123`; `acpserver/compact.go:65`; `mcpclient/proc.go:18-19`, `:83-85`; `mcpclient/conn.go:39-51`; `tool/builtin/builtin.go:34-61`; `compaction/compaction.go:15-16`, `:24`, `:62`; `internal/cli/editor/external.go:48-58`; `internal/cli/slash.go:27-28` |
| No additional directories from the CLI | `acpclient/sessions.go:57-59` |
| The writers and the lock | `mcpclient/file.go:16-198`; `session/jsonl/lock.go:21-38` |
| Tests pin the defaults | `compaction/compaction_test.go:90-97`; `acpserver/queue_test.go:20-29`; `mcpclient/proc_test.go:115`; `tool/builtin/shell_test.go:77` |
| Kong's facility | Kong `v1.16.1` `resolver.go:30-62`; `options.go:450-480`, `:520-545` |
| Pi's settings | pi `packages/coding-agent/src/core/settings-manager.ts:193-210`, `:225`, `:336-350`, `:470-475`, `:485`, `:1097-1100` |
| opencode, Kilo, codex, goose, grok | opencode `packages/cli/src/config/config.ts:43-46`, `:56-59`, `:97-99`; `packages/core/src/config/variable.ts:28-31`; kilocode `packages/opencode/src/config/variable.ts:60-69`; codex `codex-rs/config/src/loader/mod.rs:89-101`, `:1113-1156`; `codex-rs/core/src/config/schema_tests.rs:13-36`; goose `crates/goose/src/config/base.rs:506-518`; grok-build `crates/codegen/xai-grok-permission-rules/src/trust.rs:17-22`, `:436-438` |
| The record contradictions; the Pi session variable | `0003-MADR-gobble-product-identity.md:111`, `:266`, `:270-278`; `internal/cli/flags.go:67`; `0005-PLAN-v1-feature-scope.md:708`, `:826`, `:3277` |
| The config-import mentions D13 strikes | `0003-MADR-gobble-product-identity.md:134`, `:174`; `0004-MADR-go-module-architecture.md:363-366`, `:764`; `0005-MADR-v1-feature-scope.md:262`, `:284`; `0005-PLAN-v1-feature-scope.md:357-359` |

### Related records

- [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): the settings decisions, whose JSON parts are superseded by note, and rule 9.
- [0003-MADR-gobble-product-identity.md](0003-MADR-gobble-product-identity.md): the directories, `.gobble/`, the Pi bridge, and the file list.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) and [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): the YAML library, the verbs, the named keys, F1's parked settings, F6, F7 and F10.
- [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md) and [0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md): the "undecided" amendment; the parked queue modes and MCP timings.
- [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md): the editor chain, `config path`'s output, and completion's read-only rule.
- [0012-MADR-tool-schemas-descriptions-and-loading.md](0012-MADR-tool-schemas-descriptions-and-loading.md): schemas from Go types.
- [0017-MADR-todo-list-ordered-live-and-mcremote-native.md](0017-MADR-todo-list-ordered-live-and-mcremote-native.md): the settings it defers here.

### Open questions for the plan

None. Every package, key, diagnostic, command, exit code and the template's text is decided above. The plan orders the work into phases and names their tests.
