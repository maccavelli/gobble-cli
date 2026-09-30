---
status: proposed
date: 2026-09-29
associated-madr: "0005-MADR-v1-feature-scope.md"
---
# Implement the v1 line: the v1.0.0 gate, the v1.x train, and `exp/`

Associated MADR: [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md)

## Goal

Every row of the 0005-MADR capability inventory is delivered at its tier:

* **1.0** rows are complete, and listed with their test evidence, before
  the `v1.0.0` tag;
* **1.x** rows land in minor releases;
* **exp** rows live under `exp/`, off by default.

The Pi bridge from
[0003-MADR-pigo-product-identity.md](0003-MADR-pigo-product-identity.md)
is built here.

## Scope

In: the capability inventory, CLI surface, slash-command set, and Pi
bridge from 0005-MADR and 0003-MADR.

Prerequisites, not repeated here:

* [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md)
  (scaffold, harnesses, CI, release pipeline);
* [0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md)
  Phases 1–8 (ACP agent, Cobra as an ACP client, basic loop, the first
  four tools, sessions, MCP, native slash commands, `_pigo/` extensions).

Where a phase below extends a 0002-PLAN phase, it says so. It does not
re-execute that phase.

Out: the 0005-MADR "Out of the v1 line" list. Changes in magic-cli-remote
(companion checklist at the end). Tagging or pushing without an explicit
ask in the same turn.

Order: phases F1–F10 are the 1.0 gate and run in order, except where a
phase says it is independent. Phases X1–X6 are the 1.x train and `exp/`.
They may start after F4 in any order.

## Implementation Steps

### F1 — tool parity (1.0; extends 0002-PLAN Phase 3)

1. Implement `read` fully (offsets, head truncation, continuation hints,
   the single long-line hint).
2. Images: sniff the format; decode png, jpeg, gif, webp, and bmp; apply
   EXIF orientation. Resize to 2000×2000 and 4.5 MB of base64, keeping the
   smaller of PNG and JPEG. If still too large, lower the JPEG quality,
   then the dimensions. Replace the image with a text note for text-only
   models.
3. Implement `edit` to Pi's semantics: multi-edit; uniqueness and overlap
   checked against the original; BOM and CRLF preserved; the fuzzy
   fallback in Pi's normalisation order. Return ACP `diff` content and
   `locations`.
4. Implement `bash` fully: tail truncation, the full-output file, process
   groups, kill on cancel and timeout, `shellPath`, `shellCommandPrefix`,
   and the 0003 environment markers. Implement `powershell` on windows.
5. Implement `grep`, `find`, and `ls` in pure Go with a gitignore-aware
   walk. Use the limits from 0005-MADR.
6. Add the per-real-path mutation queue and `os.Root` confinement over
   `cwd` plus the additional directories.
7. When the client has the capability, route file I/O through ACP
   `fs/read_text_file` and `fs/write_text_file`. Route `bash` through
   `terminal/*` when the setting is on.
8. Implement `todo`, publishing ACP `plan` updates; `tool_search` (BM25);
   and the MCP resource tools.

**Accept:**

* A fixture suite of Pi-documented cases: truncation at 2000 lines, 50 KB,
  and 500 characters; edit ambiguous, overlap, CRLF, BOM, and fuzzy cases;
  grep with `.gitignore`.
* Native fuzz targets for the edit matcher and the path resolver. The
  resolver fuzz asserts that no result escapes the roots.
* A test that an edit through an ACP client with `fs.writeTextFile`
  produces a `fs/write_text_file` frame and no direct disk write.
* The `os.Root` escape test fails on a copy that uses `filepath.Join`.
  Quote the failure.

### F2 — sessions v3 (1.0; extends 0002-PLAN Phase 4)

1. Every v3 entry type, and v1/v2 migration on load.
2. `custom{pigo.*}` for pigo-only data. Unknown fields round-trip through
   `jsontext.Value`.
3. The tree: leaf pointer, reset leaf, fork, clone, names, and
   `parentSession`.
4. `session/list`, `load`, `resume`, and `close` over the tree.
   `session_info_update` for names.

**Accept:**

* The samples in Pi's `docs/session-format.md` load, and an unchanged
  write reproduces them after JSON normalisation.
* A v1 file and a v2 file migrate.
* A corrupt line fails closed with an ACP error.
* The fuzz target for the JSONL decoder does not panic.

### F3 — context, prompts, skills, templates (1.0)

1. Context-file walk (`AGENTS.override.md`, `AGENTS.md`, `CLAUDE.md`),
   worktree de-duplication, `SYSTEM.md`, and `APPEND_SYSTEM.md`.
2. Sectioned system prompt. Section changes are stored as system-message
   patches.
3. Agent Skills discovery in the 0005 order, `/skill:<name>`, and the
   collision warning.
4. Prompt templates with Pi's substitution grammar. Each becomes a
   `/name` command.
5. `@path` mentions become `resource_link` or an embedded resource.
   Prompt capabilities `image` and `embeddedContext` are set to true.

**Accept:**

* The examples in Pi's `docs/prompt-templates.md` are table tests,
  including quoting and defaults. The template parser has a fuzz target.
* A skills directory with a name collision warns and loads the first skill
  found.
* A golden system prompt for a fixture project matches.

### F4 — providers, catalog, auth (1.0)

1. `llm/provider/anthropic`, `openai`, `google`, and `compat` (with
   presets), each streaming `iter.Seq2[llm.Event, error]`.
2. Record each SDK version pinned, in this PLAN as a dated note.
3. Typed error mapping in every adapter (`APIError`, `RetryAfter`,
   `ErrContextOverflow`, `ErrQuota`, `ErrAuth`).
4. `llm/catalog`: an embedded snapshot and the `models.json` overlay, with
   `$ENV` and `!command` resolution.
5. Record the snapshot's source (Pi's generated catalog under MIT with a
   `NOTICE` entry, or another public catalog) here before it lands.
6. `internal/auth`: keyring with the 0600-file fallback, the credential
   precedence order, `pigo auth login|logout|status|token`, and ACP
   `authMethods`, `authenticate`, and `logout`.
7. `pigo models list|info`.
8. Config option category `model` (0002-PLAN Phase 4, amended).

**Accept:**

* Each adapter passes a shared conformance suite against an
  `httptest.NewTestServer` fake of its wire: streaming text, thinking, a
  tool call, usage, 429 with `Retry-After`, and context overflow.
* `live_<provider>` tests exist and are documented, and have been run once
  by the owner with credentials. Record the date and outcome here.
* Nothing under `internal/cli` imports a vendor SDK (archtest).

### F5 — compaction and retry (1.0; extends 0002-PLAN Phase 6)

1. Retry with typed classification and back-off, honouring `Retry-After`.
2. Auto-compaction trigger and cut-point rules. The iterative summary
   format. Split-turn merge. Overflow compacts and retries once.
3. `/compact [instructions]` and `_pigo/compact` share one code path, and
   both emit `usage_update`.

**Accept:**

* `synctest` tests prove the back-off schedule (2 s, 4 s, 8 s, with the
  cap) and `Retry-After` precedence.
* A faux-provider session crosses the threshold and compacts. No cut ever
  lands on a tool result (property test).
* A forced overflow compacts once and then succeeds.

### F6 — permissions, modes, trust (1.0; extends 0002-PLAN Phase 6)

1. The `permission` engine: `allow`, `ask`, `deny` rules and the matcher
   grammar. Defaults come from tool annotations.
2. Modes `default`, `accept-edits`, `plan`, and `bypass` (behind
   `permissions.allowBypass`). A `current_mode_update` on every change.
3. `session/request_permission` with the four option kinds.
   `allow_always` persists a rule. `/permissions`.
4. Non-interactive resolution: `ask` becomes deny unless `--yes` or
   `bypass` is set.
5. Project trust (`trust.json`, the decision order, `pigo trust|untrust`,
   `/trust`) and the built-in protected paths.

**Accept:**

* Table tests for rule precedence.
* A plan-mode session in which the model calls `edit` gets a rejection
  and no write.
* A permission round-trip through `acptest.Pair` records the
  `session/request_permission` frame.
* An untrusted project's `.pigo/settings.json` has no effect. A test
  fails on a copy that loads it anyway.

### F7 — hooks (1.0)

1. The `hook` contract and exec-hook runner: JSON on stdin and stdout, a
   timeout, exit code 2 blocks.
2. Events `session_start`, `session_shutdown`, `input`,
   `before_agent_start`, `tool_call`, `tool_result`,
   `session_before_compact`, `session_compact`, `turn_end`,
   `agent_settled`, and `model_select`.
3. Hooks from project settings run only when the project is trusted.

**Accept:**

* A fixture hook blocks `bash(rm*)`, and the model sees the reason.
* A `tool_result` hook rewrites the output.
* A hook that hangs is killed at its timeout, and the turn continues with
  a logged error.
* A Go `hook.Hook` passed to `pigo.New` observes the same events. An
  `Example` shows it.

### F8 — MCP completion (1.0; extends 0002-PLAN Phase 5)

1. Pi's `mcp.json` schema, including `exposure` and the `toolExposure`
   globs, and the value interpolation.
2. The 64-character names with a hash suffix. The 20 KB middle truncation
   with the full-output file.
3. Lifecycle rules: 10 s start-up wait, retries, `list_changed`, reconnect,
   and the process-group stop.
4. OAuth: discovery, dynamic client registration, PKCE, a loopback
   callback with `CrossOriginProtection`, manual paste, and tokens kept in
   the secret store.
5. `pigo mcp get|login|logout` and `/mcp [status|reconnect]`.

**Accept:**

* Against the go-sdk's test server: a stdio fixture and an HTTP fixture;
  an OAuth fixture server completes a login; a `list_changed` refreshes
  the tool list.
* A stdio server that ignores SIGTERM is killed with SIGKILL, including
  its child process.

### F9 — TUI (1.0; independent of F5–F8 once F4 lands)

1. `internal/tui` on Charm v2 as an `acpclient` consumer:
   * transcript with streamed Markdown;
   * tool cards (collapsed or expanded) with diffs;
   * permission dialog;
   * model, thinking, and mode pickers;
   * session picker;
   * multi-line editor with history;
   * slash and `@path` completion;
   * `!cmd` and `!!cmd`;
   * Esc to interrupt;
   * steer on Enter while busy, follow-up on Alt+Enter;
   * Ctrl+G for the external editor;
   * a footer with model, mode, context %, and cost.
2. TUI-only commands: `/settings`, `/hotkeys`, `/quit`, `/copy`, `/new`,
   `/resume`, `/login`, `/logout`, `/theme`, `/changelog`, `/editor`.
3. Record the chosen Markdown renderer here.

**Accept:**

* `teatest`-style golden-frame tests for the transcript, a tool card, and
  the permission dialog at 80×24 and 120×40.
* archtest confirms `internal/tui` does not import `agent`.
* A manual smoke run on macOS, Linux, and Windows Terminal is recorded
  here with its date.

### F10 — print mode, operations, release gate (1.0)

1. Print mode `-p`, piped stdin, `--output-format text|json|stream-json`.
2. The usage ledger, `/usage` (session plus today), and `pigo usage`.
3. `pigo doctor [--json]`.
4. `pigo update` through `mcplib/selfupdate`.
5. The Pi bridge (0003): discovery sources, `session list --pi`,
   `session import`, `config import --from-pi`, `auth import --from-pi`,
   and `--no-pi-bridge`.
6. The **release gate**. Build a table in this PLAN listing every 1.0 row
   of 0005-MADR with its test name and the last CI result. Mark each row
   stable or beta per 0004-MADR. Promote stable packages. Run `apidiff`
   against the last pre-release tag and record its output.

**Accept:**

* `stream-json` output of a fixture session equals the recorded ACP
  `session/update` params, line for line.
* The Pi-user smoke test from 0005-MADR Confirmation passes, and the
  `~/.pi` fixture hash is unchanged.
* The gate table has no empty row.
* `v1.0.0` is **not** tagged by this plan. The owner asks for the tag
  explicitly in the turn that pushes it.

### X1 — sessions and context 1.x

1. Branch summaries, `/tree [id]`, `/label`, and `context_edit`.
2. Automatic titles.
3. Export `jsonl`, `md`, and self-contained `html`; `/export`; `/share`
   through `gh`.
4. `/archive` and `/delete`.
5. The TUI tree navigator.
6. MCP prompts as `/mcp:<server>:<prompt>`.
7. Built-in `/review`, `/init`, and the `deep-research` skill.

**Accept:** each new command is advertised only in the build that
contains its handler (the subset test). The companion-table rows are
updated in the release notes.

### X2 — checkpoints and background jobs 1.x

1. The file journal for `edit` and `write`. A shadow-index git snapshot
   when a git repository and `git` are present, using its own
   `GIT_INDEX_FILE` and refs under the data directory.
2. `/undo`, `/redo`, and `/diff [turn]`.
3. `bash{background:true}`, `job_output`, `job_kill`, `/ps`, and
   `/stop`. Jobs end when the session closes.

**Accept:**

* `/undo` after an edit restores the bytes exactly. After a bash
  `sed -i` in a git workspace, it restores the file.
* The user's `.git/index`, stash, and refs are byte-identical before and
  after (hash assertion). The test fails on a copy that uses the default
  index.
* A background job survives turn boundaries and dies on `session/close`.

### X3 — subagents, web, provider tools 1.x

1. `agents/*.md` discovery (config, `.pigo/agents`, and packages).
2. `task`: a child session through an in-process ACP connection,
   `parentSession` link, progress relayed as `tool_call_update`, parallel
   batches, and a depth limit.
3. `web_fetch` converting HTML to Markdown, with an open-world permission
   default.
4. Provider server tools passed through per model.
5. `/agents`.

**Accept:** a faux parent delegates two parallel tasks and receives both
results. The child sessions are listed with their parent. A task that
exceeds the depth limit gets a tool error.

### X4 — clients and servers 1.x

1. `pigo rpc`: the Pi JSONL RPC shim over an in-process ACP client.
   Commands follow the 0002-MADR mapping table.
2. `pigo mcp serve` with the `pigo_run` and `pigo_session_prompt` tools.
3. MCP sampling and elicitation.
4. TUI themes (Pi theme JSON) and `keybindings.json` with Pi action ids.

**Accept:**

* A replay of the command sequences documented in Pi's
  `docs/rpc-commands.md` through `pigo rpc` produces responses with the
  documented shapes.
* The official MCP go-sdk client calls `pigo_run` against a faux provider.

### X5 — providers, packages, operations 1.x

1. Bedrock, Vertex, and Azure through the SDK options.
2. Subscription OAuth: one provider per step, each with a recorded check
   of the provider's terms.
3. The go-llmprovider-sdk adapter once that module has a `go.mod` and
   streaming.
4. `pigo models refresh`.
5. `pigo pkg install|remove|list|update`: git, path, and https-with-sha256
   sources; `pigo-package.json`; Pi `package.json` manifests with their
   extensions skipped; `packages.lock.json`.
6. Prompt-cache warming.
7. OpenTelemetry (GenAI semantic conventions).
8. The flight recorder and `pigo debug trace`.
9. `/bug`.
10. The Homebrew tap and the `ko` image.
11. `default.pgo` from a recorded benchmark profile.

**Accept:**

* A Pi package fixture installs, and its skills and prompts load. Its
  `extensions` are listed as skipped.
* The lock file pins the sha256, and a tampered archive is refused.
* An OTLP test collector receives `chat` and `execute_tool` spans with
  `gen_ai.*` attributes.

### X6 — `exp/`

Each item is off by default, enabled by a setting, and needs a later MADR
to promote it:

* `exp/codemode` (wazero plus `quickjs-wasi`, Pi's script contract);
* `exp/wasmtool`;
* `exp/sandbox` (Seatbelt, Landlock);
* `exp/acpws` (ACP over WebSocket and Unix socket, guarded by
  `CrossOriginProtection`);
* `exp/a2a` (agent card and task endpoint);
* external ACP agents as subagents;
* virtual-model routing;
* the OCI package source;
* inline terminal images.

**Accept:** archtest rule 7 holds (no stable or beta package imports
`exp/`). Each `exp` package has one end-to-end test behind its setting.

## Verification

For each phase:

* the phase's accepts, with the evidence quoted in the execution record;
* `make preflight`;
* `go test -race ./...`;
* archtest.

Every new gate or assertion is shown failing on a planted defect in a
scratch copy before it is trusted, and the failure text is recorded.

Before the gate, from a clean clone:

```text
make preflight
go test ./...
go test -race ./...
go test -tags live_anthropic,live_openai,live_google ./llm/provider/...   # owner-run, credentials required
make release-dry-run
```

## Rollout and Rollback

* Each phase is one or more commits after pre-add checks. Rollback is
  `git revert` of the phase's commits.
* 1.x items ship in minor releases. A 1.x item that regresses is disabled
  by its setting in a patch release, not reverted across a release.
* `exp/` items are never on by default, so rolling one back is deleting
  its directory in a minor release.
* After `v1.0.0`, stable-tier API changes follow `apidiff`. A breaking
  change needs a new MADR and a module `v2`.

## Companion work (magic-cli-remote, not this tree)

Do not mutate magic-cli-remote under this plan. When it gains `IDPi`, it
needs its own MADR/PLAN pair. The contract is:

1. `DefaultBin` `pigo`, `DefaultArgs` `acp`.
2. The 0005-MADR target command table, registering only rows whose
   commands the pinned pigo release advertises.
3. Model and thinking: `KindNative` (`/model`, `/thinking`) until the Spec
   gains a hook that routes `OpSetModel` and `OpSetThinkingLevel` through
   `SetConfigOption`. Never through `session/set_model`: pigo's SDK cannot
   receive it (0002-MADR amendment of 2026-09-29).
4. Compact: `KindNative` until a Spec Compact hook calls `_pigo/compact`.
5. Live-tagged probes for every native row, and `/settings` absent.
6. `KnownGoodVersion` from the pigo release notes.

## Execution record

None. This plan is proposed and has not been approved.
