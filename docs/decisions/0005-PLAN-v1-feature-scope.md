---
status: in-progress
date: 2026-10-08
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
[0003-MADR-gobble-product-identity.md](0003-MADR-gobble-product-identity.md)
is built here.

## Scope

In: the capability inventory, CLI surface, slash-command set, and Pi
bridge from 0005-MADR and 0003-MADR.

Prerequisites, not repeated here:

* [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md)
  (scaffold, harnesses, CI, release pipeline);
* [0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md)
  Phases 1–8 (ACP agent, Kong as an ACP client, basic loop, the first
  four tools, sessions, MCP, native slash commands, `_gobble/` extensions).

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
5. Implement `grep`, `find`, and ~~`ls`~~ in pure Go with a gitignore-aware
   walk. Use the limits from 0005-MADR.
   *(2026-10-08, 0005-MADR amendment "native tools": F1c also builds `move`, `delete`, `copy`, `mkdir` and `tree`; `tree` replaces `ls`.)*
   *(2026-10-08, 0012-MADR: F1c's tools follow D1, D2 and D5, through `tool.WithSchema` and `tool/builtin/describe/`, with the schemas 0012-MADR's table gives them.)*
   *(2026-10-08: F1c is made executable as two sub-phases, F1c-1 (the matcher, grep, find, `tree`) and F1c-2 (move, delete, copy, mkdir), with one in-house glob engine. See the Amendments entry "F1c made executable".)*
6. Add the per-real-path mutation queue and `os.Root` confinement over
   `cwd` plus the additional directories.
7. When the client has the capability, route file I/O through ACP
   `fs/read_text_file` and `fs/write_text_file`. Route `bash` through
   `terminal/*` when the setting is on.
8. Implement `todo`, publishing ACP `plan` updates; `tool_search` (BM25);
   and the MCP resource tools.
   *(2026-10-08, 0012-MADR: F1d builds D7's tiers, D8's families and D9's loading, D11's BM25 in `tool/toolsearch`, and D14's client data; the three resource tools take 0012-MADR's schemas.)*
   *(2026-10-08: F1d is made executable as three sub-phases, F1d-1 (`todo`), F1d-2 (`tool_search`, loading and the MCP resource tools) and F1d-3 (routing and `locations`). See the Amendments entry "F1d made executable".)*

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

*(2026-10-07: F1 runs as four sub-phases, F1a–F1d, by the owner's decisions of that date. Step 2's decode and resize wait for the provider SDK to carry images. See the Amendments entry "F1 made executable".)*
*(2026-10-07, 0005-MADR amendment "the best of several harnesses": F1 is five sub-phases, F1a–F1e; F1e is `apply_patch`. F1a is reworked before its commit. See the Amendments entry "F1 restructured; F1a reworked".)*

### F2 — sessions v3 (1.0; extends 0002-PLAN Phase 4)

1. Every v3 entry type, and v1/v2 migration on load.
2. `custom{gobble.*}` for gobble-only data. Unknown fields round-trip through
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

*(2026-10-07, 0005-MADR amendment "the best of several harnesses": choice 8: a frozen per-session context baseline, with changes appended as messages and per-turn state kept out of the system prompt. Nested `AGENTS.md` on read (choice 3) lands here.)*
*(2026-10-08, 0012-MADR D6 and D9: the `tools` section carries the cross-tool guidance, moved out of bash's description, and the categories of deferred tools; the tools list is part of the frozen baseline and grows only by appending.)*

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

*(2026-10-07, 0005-MADR amendment "the best of several harnesses": choice 11: provider-neutral reasoning variants offered over ACP as `thought_level`, and a small-model role for titles and summaries. The catalog's data stays the SDK's; gobble overlays only what it lacks.)*

Grounded in `go-llmprovider-sdk` as probed on 2026-09-30
(`origin/main` `3d4aff5f2be9363877aae0987755e6781f1cae98`; 0015-PLAN
in S7). Re-resolve the module version at the start of this phase and
record the pseudo-version here. Do not import the pre-S8 `Generate*`
API.

1. `llm/provider` is the only adapter. It:
   * constructs through `llmprovider/providers.New(id, opts...)`;
   * streams through `llmprovider.Stream` (synthesised events until a
     provider sets `NativeStreaming`);
   * wraps `llmprovider.WithRetry` for retryable kinds;
   * maps `llmprovider.Event` and `APIError` onto `llm`'s sealed events
     and sentinels as 0004-MADR specifies;
   * passes `WithClientInfo` from 0003-MADR (`gobble`, semver).
2. 1.0 ids on day one of this phase: every id `providers.New` already
   implements (`openai`, `claude`, `gemini`, `grok` on the probed
   commit). As 0015-PLAN S7 moves `opencode-zen`, `opencode-go`,
   `huggingface`, `kilo`, `together`, and `ollama` onto the contract,
   add each id in a follow-up commit of this phase, recording the SDK
   commit. Do not ship a gobble wrapper of the old API for those ids.
3. Record the pinned `go-llmprovider-sdk` pseudo-version here.
4. Typed error mapping is the 0004-MADR table. A 429 with `Retry-After`
   must round-trip through the adapter as `llm.ErrRateLimited` plus
   `APIError.RetryAfter`. Context overflow must round-trip as
   `llm.ErrContextOverflow` and must **not** be retried by `WithRetry`.
5. `llm/catalog`: an embedded snapshot and the `models.json` overlay, with
   `$ENV` and `!command` resolution. Overlay ids that the SDK does not
   implement yet fail closed with a clear error, not a silent skip.
6. Record the snapshot's source (Pi's generated catalog under MIT with a
   `NOTICE` entry, or another public catalog) here before it lands.
7. `internal/auth`: keyring with the 0600-file fallback; implements
   `llmprovider.TokenStore` so SDK `OAuthSession` rotations persist.
   Credential precedence, `gobble auth login|logout|status|token`, and ACP
   `authMethods`, `authenticate`, and `logout`. ChatGPT/Codex login uses
   the SDK `openai` backend; do not reimplement that loopback in gobble.
8. `gobble models list|info`.
9. Config option category `model` (0002-PLAN Phase 4, amended).

*(2026-10-07: 0002-PLAN Phase 7's step 4, `gobble auth` "as far as the chosen provider requires", is met by Phase 3's ambient credentials, by the owner's decision of that date. Step 7's `gobble auth login|logout|status|token` stays here. Until step 5's catalog, Phase 7's `_gobble/get_state` names the model as `{provider, id}`, and `_gobble/get_session_stats` reports `cost` 0 with no `contextUsage`. See that plan's Amendments entry "Phase 7 made executable".)*

**Accept:**

* For each 1.0 id, an adapter test against `httptest.NewTestServer`
  (or the SDK's own `llmtest.Run` harness behind the facade) covers:
  text, a tool call, usage on `done`, 429 with `Retry-After`, and
  context overflow. Because `NativeStreaming` is `Unsupported`, the
  text case asserts one `TextDelta` (or a small number of events after
  `Generate` returns), not a token stream.
* `live_<id>` tests exist for `openai`, `claude`, `gemini`, and `grok`,
  skipped without credentials, and have been run once by the owner.
  Record the date and outcome here.
* `go list -m github.com/maccavelli/go-llmprovider-sdk` prints the
  recorded pin. `go list -m` does not name `anthropic-sdk-go`,
  `openai-go`, or `google.golang.org/genai`.
* Nothing under `internal/cli` imports `go-llmprovider-sdk` (archtest
  rule 4).

### F5 — compaction and retry (1.0; extends 0002-PLAN Phase 6)

*(2026-10-07, 0005-MADR amendment "the best of several harnesses": choice 7: loop safety, meaning the doom-loop question, invalid-call repair, Kilo's circuit breakers, offline detection and a first-byte deadline. Choice 9: Pi's cut point stays; opencode's summary template, rolling updates and prune pass; Kilo's proactive threshold, empty-summary guard, too-large recovery and per-turn cap.)*

1. Retry with typed classification and back-off, honouring `Retry-After`.
   The provider-level retries are `llmprovider.WithRetry` from F4.
   This phase adds the agent-level policy (compaction on
   `llm.ErrContextOverflow`, the 2 s / 4 s / 8 s schedule and 60 s cap
   matching Pi, never retry exhausted quota). Do not duplicate the SDK's
   retry loop.
2. Auto-compaction trigger and cut-point rules. The iterative summary
   format. Split-turn merge. Overflow compacts and retries once.
3. `/compact [instructions]` and `_gobble/compact` share one code path, and
   both emit `usage_update`.

*(2026-10-06: 0002-PLAN Phase 6 delivers Pi's manual compaction, by the owner's decision of that date: step 2's cut-point rules, iterative summary format and split-turn merge, and step 3's `/compact`. This phase keeps step 1, step 2's automatic trigger and overflow compact-and-retry, and `_gobble/compact` (0002-PLAN Phase 7). See that plan's Amendments entry "Phase 6 made executable".)*

*(2026-10-07: 0002-PLAN Phase 7 delivers `_gobble/compact` on `/compact`'s code path (`compactSession`), with `usage_update`. Step 3 is complete. `_gobble/set_auto_compaction` and the retry methods wait for this phase's automatic compaction and retry.)*

**Accept:**

* `synctest` tests prove the back-off schedule (2 s, 4 s, 8 s, with the
  cap) and `Retry-After` precedence.
* A faux-provider session crosses the threshold and compacts. No cut ever
  lands on a tool result (property test).
* A forced overflow compacts once and then succeeds.

### F6 — permissions, modes, trust (1.0; extends 0002-PLAN Phase 6)

*(2026-10-08, 0005-MADR amendment "native tools": `request_permissions` lands here. A delete always asks, whatever an allow rule says.)*
*(2026-10-08, 0012-MADR D10: a bash command that is one plain `rm`, `rmdir`, `mv`, `cp` or `mkdir` with flags that map exactly runs as the native tool; anything else is permission-checked as a shell command. D7: `request_permissions` is deferred, and loads when a call is refused for a permission.)*

*(2026-10-07, 0005-MADR amendment "the best of several harnesses": choice 5: bash parsed with `mvdan.cc/sh` for per-sub-command patterns, arity-prefix "always", inert-operator masking and fail-closed raw patterns. Choice 6: opencode's last-match-wins rules, hidden denied tools, feedback on rejection, session and per-project "always", and Kilo's hard ceilings, protected paths and provenance. Choice 10: the two modes stay; no custom agents yet.)*

1. The `permission` engine: `allow`, `ask`, `deny` rules and the matcher
   grammar. Defaults come from tool annotations.
2. Modes `default`, `accept-edits`, `plan`, and `bypass` (behind
   `permissions.allowBypass`). A `current_mode_update` on every change.
3. `session/request_permission` with the four option kinds.
   `allow_always` persists a rule. `/permissions`.
4. Non-interactive resolution: `ask` becomes deny unless `--yes` or
   `bypass` is set.
5. Project trust (`trust.json`, the decision order, `gobble trust|untrust`,
   `/trust`) and the built-in protected paths.

*(2026-10-06: 0002-PLAN Phase 6 delivers modes `default` and `plan`, `/mode` and `/plan`, plan mode's refusal of non-read-only tools, and an `exit_plan_mode` tool that carries the plan through `session/request_permission` and publishes it as a `plan` update, by the owner's decision of that date. This phase keeps `accept-edits`, `bypass`, the rule engine and `allow_always`.)*

**Accept:**

* Table tests for rule precedence.
* A plan-mode session in which the model calls `edit` gets a rejection
  and no write.
* A permission round-trip through `acptest.Pair` records the
  `session/request_permission` frame.
* An untrusted project's `.gobble/settings.json` has no effect. A test
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
* A Go `hook.Hook` passed to `gobble.New` observes the same events. An
  `Example` shows it.

### F8 — MCP completion (1.0; extends 0002-PLAN Phase 5)

*(2026-10-08, 0012-MADR D7: an MCP server with no `exposure` key is deferred.)*

1. Pi's `mcp.json` schema, including `exposure` and the `toolExposure`
   globs, and the value interpolation.
2. The 64-character names with a hash suffix. The 20 KB middle truncation
   with the full-output file.
3. Lifecycle rules: 10 s start-up wait, retries, `list_changed`, reconnect,
   and the process-group stop.
4. OAuth: discovery, dynamic client registration, PKCE, a loopback
   callback with `CrossOriginProtection`, manual paste, and tokens kept in
   the secret store.
5. `gobble mcp get|login|logout` and `/mcp [status|reconnect]`.

*(2026-10-06: 0002-PLAN Phase 5 delivers part of this phase, by the owner's decision of that date: step 1's checks of the whole `mcp.json` schema and its `$NAME` / `${NAME}` values, with `!command` left here; step 2's 64-character names with the hash suffix; step 3's per-call `timeout`, with the progress reset left here. `exposure` and `toolExposure` are checked and kept there, and applied here. See that plan's Amendments entry "Phase 5 made executable".)*

*(2026-10-07: 0002-PLAN Phase 8 delivers step 3's process-group stop, by the owner's decision of that date. A stdio server runs in its own process group, or a kill-on-close Job object on Windows, and is stopped with Pi's sequence and timings. This phase's Accept line "A stdio server that ignores SIGTERM is killed with SIGKILL, including its child process" is tested there. This phase keeps the retries, `list_changed`, reconnect, the truncation and OAuth. See that plan's Amendments entry "Phase 8 made executable".)*

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

*(2026-10-07: 0002-PLAN Phase 7 delivers the steering and follow-up queues, in `agent` and over ACP: `_gobble/steer`, `_gobble/follow_up`, `_gobble/clear_queue` and the two queue modes. Step 1's Enter-steers and Alt+Enter-follows-up keys stay here, by the owner's decision of that date. A cancelled turn's response carries the cleared texts in `_meta.gobble.cleared`, for the editor to restore, as Pi's terminal does.)*

**Accept:**

* `teatest`-style golden-frame tests for the transcript, a tool card, and
  the permission dialog at 80×24 and 120×40.
* archtest confirms `internal/tui` does not import `agent`.
* A manual smoke run on macOS, Linux, and Windows Terminal is recorded
  here with its date.

### F10 — print mode, operations, release gate (1.0)

1. Print mode `-p`, piped stdin, `--output-format text|json|stream-json`.
2. The usage ledger, `/usage` (session plus today), and `gobble usage`.
3. `gobble doctor [--json]`.
4. `gobble update [--check]` through
   `github.com/maccavelli/go-core-lib/selfupdate` `v1.1.0`
   (`96b30961180671ab3697585951219001ecbb1c90`). Bind it as
   `selfupdate/example_test.go` does for a standalone program:
   `NewGitHubSource` (User-Agent from 0003, repository owner/name of
   this module's GitHub), `NewStrictVersionPolicy`,
   `NewExactAssetSelector` for the six Phase 4 platforms *(five since
   2026-10-07: darwin/amd64 is not a target, 0004-MADR amendment of that
   date; F10 reads them from the release spec)*,
   `NewStandaloneInstaller`, `NewTextReporter(os.Stderr)`,
   `NewTerminalConfirmer(os.Stdin, os.Stderr)`. `--check` is
   `Request.CheckOnly`; `--force` is `Request.Force`; `--yes` is
   `Request.Yes`. `Request.Product` is `gobble`. `Request.CurrentVersion`
   and `CurrentBuild` come from `internal/buildinfo`. Map `ExitCode` to
   process status (0, 10, 1). `internal/cli` is the only importer of
   `go-core-lib` (0004-MADR import-boundary rule 8).
5. The Pi bridge (0003): discovery sources, `session list --pi`,
   `session import`, `config import --from-pi`, `auth import --from-pi`,
   and `--no-pi-bridge`.
6. The **release gate**. Build a table in this PLAN listing every 1.0 row
   of 0005-MADR with its test name and the last CI result. Mark each row
   stable or beta per 0004-MADR. Promote stable packages. Run `apidiff`
   against the last pre-release tag and record its output.
   *(2026-10-07: the comparison is written here. Until it is, `make apidiff` is a no-op below `v1.0.0` and fails on a `v1.0.0` or later tag, naming it; 0004-PLAN Phase 5. It needs the previous tag, so CI then fetches tags, which its shallow checkout does not do today.)*

**Accept:**

* `stream-json` output of a fixture session equals the recorded ACP
  `session/update` params, line for line.
* `gobble update --check` against `selfupdatetest.GitHubServer` with a
  fixture named `gobble-linux-amd64` reports availability (exit 10) and
  writes nothing but reporter text to stderr. A planted archive name
  (`gobble-linux-amd64.tar.gz`) is not selected (`ErrIntegrity` or
  missing asset). `go list -m github.com/maccavelli/go-core-lib`
  prints `v1.1.0`.
* The Pi-user smoke test from 0005-MADR Confirmation passes, and the
  `~/.pi` fixture hash is unchanged.
* The gate table has no empty row.
* `v1.0.0` is **not** tagged by this plan. The owner asks for the tag
  explicitly in the turn that pushes it.

### X1 — sessions and context 1.x

*(2026-10-07, 0005-MADR amendment "the best of several harnesses": choice 12: project memory joins X1 as a step with its own MADR first, on Kilo's design (0010-REPORT §12).)*

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

*(2026-10-08, 0005-MADR amendment "native tools": step 3 is replaced by process sessions: a yield window on `bash` and `powershell`, `job_output`, `job_input` and `job_kill`, completion notices, and `monitor`. Its own MADR comes first, naming the pseudo-terminal library interactive jobs need.)*

*(2026-10-07, 0005-MADR amendment "the best of several harnesses": choice 12: step 1's shadow repository borrows the real repository's objects through `alternates`, takes a tree per step with a patch, and follows Kilo's checks: validate before restore, honest reports of failed restores, and a timeout on large repositories.)*

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

1. `agents/*.md` discovery (config, `.gobble/agents`, and packages).
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

*(2026-10-08, 0012-MADR D4 and D14: `gobble mcp serve` also serves the native file tools with their schemas, and each gains an `outputSchema`.)*

1. `gobble rpc`: the Pi JSONL RPC shim over an in-process ACP client.
   Commands follow the 0002-MADR mapping table.
2. `gobble mcp serve` with the `gobble_run` and `gobble_session_prompt` tools.
3. MCP sampling and elicitation.
4. TUI themes (Pi theme JSON) and `keybindings.json` with Pi action ids.

**Accept:**

* A replay of the command sequences documented in Pi's
  `docs/rpc-commands.md` through `gobble rpc` produces responses with the
  documented shapes.
* The official MCP go-sdk client calls `gobble_run` against a faux provider.

### X5 — providers, packages, operations 1.x

1. Bedrock, Vertex, and Azure **only if** `go-llmprovider-sdk` has grown
   those ids or options. If it has not, this step is skipped and the
   gap is recorded here; gobble does not add vendor LLM SDKs to close it.
2. Subscription OAuth beyond ChatGPT/Codex: one provider per step, each
   with a recorded check of the provider's terms, through the SDK's
   `OAuthSession`.
3. Unnamed OpenAI-compatible presets (Groq, Cerebras, DeepSeek,
   OpenRouter, Fireworks, Mistral, LM Studio, llama.cpp, vLLM) **only
   if** the SDK exposes a Chat Completions provider that takes a base
   URL, or adds those ids. gobble does not speak those wires itself.
4. `gobble models refresh`.
5. `gobble pkg install|remove|list|update`: git, path, and https-with-sha256
   sources; `gobble-package.json`; Pi `package.json` manifests with their
   extensions skipped; `packages.lock.json`.
6. Prompt-cache warming.
7. OpenTelemetry (GenAI semantic conventions).
8. The flight recorder and `gobble debug trace`.
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

### X7 — feedback after edits 1.x

*(2026-10-08, 0005-MADR amendment "native tools": the `lsp` navigation tool (definition, references, hover, implementation, symbols) shares X7's LSP client, and is deferred.)*

*(2026-10-07, 0005-MADR amendment "the best of several harnesses": choice 4, a new item: formatters and LSP diagnostics after edit, write and `apply_patch`, with its own MADR first. It needs an LSP client, server discovery and per-language formatter settings (0010-REPORT §1).)*

### X8 — documents and notebooks 1.x

*(2026-10-08, 0005-MADR amendment "native tools": a new item, with its own MADR first, naming its PDF and XLSX libraries. It adds text from PDF, DOCX, XLSX and PPTX in `read`, within bounded ranges, and a deferred `notebook` tool that reads and edits cells. Running cells waits for a kernel decision (0011-REPORT).)*

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
go test -tags live_openai,live_claude,live_gemini,live_grok ./llm/provider/...   # owner-run, credentials required
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
needs its own MADR/PLAN pair. The contract is the 2026-10-01 amendment of
[0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md)
and the target table in
[0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md):

1. `DefaultBin` `gobble`, `DefaultArgs` `[]string{"acp"}`.
2. The 0005-MADR target command table, registering only rows whose
   commands the pinned gobble release advertises.
3. Parameterize Compact / Fork / Rename / Usage / SetModel / SetThinking /
   Undo on `acpagent.Spec`. Until then those rows are `KindNative`.
   Model and thinking never use `session/set_model`.
4. `ConfigureSession` applies the start-up model through
   `SetConfigOption`. Map `session_info_update` to `session_title`.
5. Live-tagged probes for every native row, and `/settings` absent.
6. `KnownGoodVersion` from the gobble release notes.

## Execution record

~~None. This plan is proposed and has not been approved.~~

**Approved, 2026-10-07.** The owner approved the 0005 pair ("approve 0005, do F1"). 0005-MADR is `accepted`, and this PLAN is `in-progress`. F1 runs first, in the plan's order, as the entry "F1 made executable" under Amendments expands it.

**F1a, 2026-10-07 — complete (staged; the owner commits).** It ran as the entry "F1 made executable" wrote it, with deviations 1–3 decided by the owner. F1b, F1c and F1d have not run.

* **Files.**
  * `tool/tool.go` and its test: `Env.Roots`, `Env.AllowOutside`, `Confined`, `WithOutside` and `WithPrepare`.
  * `internal/fsx`: `resolve.go`, `workspace.go`, `write.go`, `lock.go`, `rename_windows.go` and `rename_other.go`, their tests, `fuzz_test.go`, and the fuzz seed.
  * `tool/builtin`:
    * `files.go` deleted;
    * new `read.go`, `write.go`, `edit.go`, `editmatch.go`, `truncate.go`, `sniff.go` and `paths.go`;
    * `builtin.go`;
    * tests: `builtin_test.go`, with new `read_test.go`, `edit_test.go`, `truncate_test.go` and `fuzz_test.go`.
  * `agent/agent.go` and its test.
  * acpserver:
    * `agent.go`: the capability, `liveSession.dirs`, `additionalDirs` and `env`;
    * `agent_test.go`, `prompt_test.go` and `extensions_test.go`;
    * `testdata/session.golden`: one change, `"additionalDirectories":{}` in `sessionCapabilities`.
  * `docs/architecture.md`, this PLAN, 0005-MADR's status, and `docs/README.md`.
* **Accept.**
  * Pi's read cases pass with its notices compared exactly:
    * `[Showing lines 1-2000 of 2500. Use offset=2001 to continue.]`;
    * `[Showing lines 1-243 of 500 (50.0KB limit). Use offset=244 to continue.]`, where line 243 was computed by hand in the test;
    * `[90 more lines in file. Use offset=11 to continue.]`;
    * `Offset 100 is beyond end of file (3 lines total)`;
    * the long-line hint;
    * each image kind's note.
  * Pi's edit cases pass: 28 of them, covering the fuzzy and CRLF suites (`tools.test.ts` "edit tool fuzzy matching" and "edit tool CRLF handling"), the three argument repairs, and the empty list.
  * Confinement:
    * `TestReadConfined` and `TestOutside`: `../` and a symlink out of the root are outside, and an approved read through the link works;
    * `TestRootRefusesEscape`: the `os.Root` refuses the link on its own;
    * `TestOutsideRoots` (agent): a read-only outside call is asked about with ` (outside the workspace)`, runs only when allowed, and is refused with no policy;
    * `TestAdditionalDirectories` (acpserver): a file in an advertised additional directory is read with no permission request; without the directory the client is asked; a relative directory is `invalid_params`.
  * `TestWriteIsAtomic` passes, with a concurrent reader over 20 replacements of 1 MiB. `TestWriteKeepsMode` passes in WSL and skips on Windows, which has no Unix permission bits.
  * `TestLockSerialises` passes.
  * Each fuzz target ran 30 s on Windows with no failure: `FuzzResolve`, `FuzzEditMatch`, and `FuzzOutside` after deviation 2's fix.
* **Gates.**
  * lint printed `0 issues.` for linux, darwin and windows.
  * Windows `go test ./...` had no `FAIL` or `panic:`.
  * `make preflight` printed `preflight passed` on Windows and in WSL.
  * WSL `go test -race ./...` exited 0.
* **Negative tests,** each on its own scratch copy, each failing as expected, and run again on the final code:
  * confinement through `filepath.Join` instead of `os.Root`, with lexical containment: `workspace_test.go:106: read …\work\link\secret.txt = "secret\n", <nil>; want ErrOutside`;
  * edits matched progressively: `TestEditPi/against_the_original … Found 2 occurrences of edits[1] in f.txt.`, and 17 cases in all;
  * CRLF not restored: `TestEditPi/CRLF_kept: file = "first\nREPLACED\nthird\n", want "first\r\nREPLACED\r\nthird\r\n"`;
  * the smart single quote step removed: `TestEditPi/smart_single_quotes: edit failed: Could not find the exact text in f.txt.`;
  * `fsx.Lock` a no-op: `lock_test.go:39: a acquired while a was held`;
  * `agent` ignoring `Confined`: `agent_test.go:190: allow false: rules = []; want one question about the outside read`;
  * the continuation offset off by one: `TestReadPi/line_limit: output lacks "…Use offset=2001 to continue.]"`.

  Two more failed first in the working tree, before their fixes:
  * `TestWriteIsAtomic` before the rename retry (deviation 1);
  * the `FuzzOutside` seed before `relativeFold`'s `IsLocal` (deviation 2).
* **What the entry predicted wrongly.**
  1. **Windows renames** (deviation 1), and **a stream name counted inside** (deviation 2). Both were found by tests written for this phase.
  2. **`acpserver/commands.go` did not change.** Fork writes a stored session that the client then opens with `session/load` or `resume`, and those carry `additionalDirectories`.
  3. **A limit of 0 is no limit.** Pi tells `limit: 0` from no limit, and reads nothing for it. Here an `int` with `omitzero` cannot tell them apart, and reading nothing asks for nothing.
  4. **Pi's messages break Go's error-string rules** (a capital, a final period). One documented helper, `piText`, carries them, instead of a lint exception at each of about a dozen sites.
  5. **Lint found 18 issues in the new code on the first full run.** They included a `windows` constant that clashed with the `x/sys/windows` import, `real` shadowing the builtin, and character-class regexes rewritten as `\x{…}` in raw strings. The fixes changed no behaviour, and every test and negative was run again after them.
  6. **A test's own assumption was wrong:** `TestOutsideRoots` first checked that the refusal did not contain `42`, and the temporary directory's name did. It now checks for the secret's whole sentence.
  7. **The identifier scan flagged a test path,** a Windows user-profile path in `TestWindowsShellPath` whose user was `x`. It is `<user>` now, as the repository's placeholders are.

## Amendments

**2026-09-30 — shared libraries as they exist.** Follows 0004-MADR and
0005-MADR of this date.

* F4 is a single `llm/provider` adapter over `go-llmprovider-sdk`
  (`providers.New`, `Stream`, `WithRetry`, `TokenStore`). Official
  vendor LLM SDKs are out. Native token streaming is not claimed.
* F5 uses `llmprovider.WithRetry` for provider retries and keeps
  compaction-on-overflow in gobble.
* F10 `gobble update` binds `go-core-lib/selfupdate` `v1.1.0` as the
  standalone example does.
* X5 no longer waits for a `go.mod` and streaming. Bedrock / Vertex /
  Azure and the unnamed OpenAI-compatible long tail land only if the
  SDK grows them.

**2026-10-01 — native agent of magic-cli-remote.** Companion checklist
follows 0002-MADR's fourth amendment (Spec-parameterized ops, title
mapping). X3 subagents stay `tool_call_update` on the parent: acpagent
drops child-session ids. No new phases.

**2026-10-01 — cross-repository assessment (MADR second amendment of this date).** No new phases. Changes to existing phases:

* **F1.**
  * Default active tools are `read, bash, edit, write, grep, find, ls`.
  * Image content goes to the model only when the SDK can send images; until then the text-note path is always used.
  * Add fixtures for the documented RE2-versus-ripgrep differences.
* **F2.** Replace "A corrupt line fails closed with an ACP error" with three accepts:
  * a malformed line is skipped with a diagnostic;
  * an unreadable header fails closed;
  * a truncated last line, written as Pi writes it, loads every complete entry.

  Further changes:
  * "Leaf pointer" is the in-memory index; the leaf is the last entry in file order.
  * Files are created lazily.
  * The golden fixtures are captured from real Pi output at `312184edb`, not copied from `docs/session-format.md`.
  * `model_change` and `thinking_level_change` are restored on load.
* **F3.**
  * Add the context-file candidates `AGENTS.MD` and `CLAUDE.MD`.
  * Add the `docs` and `addendum` sections, with `APPEND_SYSTEM.md` going into `addendum`.
  * Add `${@:-default}`, and the first-line description fallback.
  * Use the MADR's new skill discovery order.
* **F4.**
  1. **SDK pin.** Start from the development pin `940fee0`. Record each SDK commit built against. No gobble tag ships on a pseudo-version.
  2. **Day-one ids.** All 10 SDK ids are 1.0 on day one (step 2's "four, then the rest as S7 lands" is done).
  3. **Configuration.** The adapter sets:
     * `WithModelProbes(false)`;
     * `MaxOutputTokens` per model;
     * one provider per ACP session (`WithSessionID`);
     * its own `*http.Client`;
     * credentials passed explicitly, reading `ANTHROPIC_API_KEY`.
  4. **Error mapping.** Follow the MADR's mapping order. Handle `*RateLimitError` and `*IncompleteError` until S8.
  5. **Usage.** A token estimator marks `Usage.Estimated`.
  6. **Reference tables** in `llm/catalog`: the thinking-level map and the Pi↔SDK provider-id map.
  7. **Token store.** `internal/auth` implements `RefreshLocker`.
  8. **Catalog provenance.** Step 6 records the catalog snapshot's provenance as models.dev and/or OpenRouter, with their terms checked. It is not Pi's MIT tree.
  9. **Accept changes.**
     * Add a 429 case that arrives as `*RateLimitError`; it round-trips to `llm.ErrRateLimited` plus `RetryAfter`.
     * Add a quota case: `ErrQuotaExhausted` maps to `llm.ErrQuota`, not `llm.ErrRateLimited`.
     * Add a test that `doctor` and `models list` make no generate call.
     * Remove the `ANTHROPIC`→`CLAUDE` env assumption.
  10. **Live probe.** Run a Gemini tool-schema probe before the sanitizer is relied on.
* **F5.**
  * Provider retry is `WithRetry` with `RetryPolicy{MaxAttempts: 4, BaseDelay: 2s, MaxDelay: 60s}`.
  * The `synctest` accept asserts each delay inside its jitter bound (2–2.5 s, 4–5 s, 8–10 s), the 60 s cap, and that a `Retry-After` above the cap fails fast. It no longer asserts the exact steps.
  * Agent-level retry never wraps a streaming provider. When a `Streamer` exists, retry happens before the first event only.
* **F6.**
  * `permissions.allowBypass` defaults to `false`.
  * Leaving plan mode also publishes a `plan` update.
  * Add the default rule `bash(gobble update*)` → ask.
* **F8.**
  * Bridged Pi MCP servers with no `exposure` become `deferred`, and `tool_search` is activated.
  * `.pi/mcp.json` is read only when the project is trusted.
* **F10.**
  1. **Pin.** Re-resolve go-core-lib to the newest `v1.x`, with the same SHA in `go.mod` and in `uses:`.
  2. **Surface.** Bind the MADR's full update surface, with flags local to `update`. `--check` uses `Checker.Check`, apply uses `RunWith`, and the TUI uses `Start` / `Stream`.
  3. **Probes.** Use `NewVersionProber` and `NewImageVerifier`.
  4. **Install guards.** Add the Homebrew/symlink detection, and the refusal when `AI_AGENT=gobble` is set.
  5. **Readiness.** Add `gobble auth check [--json]`, and `--offline` / `GOBBLE_OFFLINE`.
  6. **Accept rewrite.** The update accept is rewritten as follows:
     * `gobble update --check` runs against `selfupdatetest.GitHubServer`, with `Request.Platform` set to `linux/amd64` and a fixture named `gobble-linux-amd64`, through `Checker.Check`. It exits 10.
     * Apply runs against a binary copied into a temp home with an explicit `TargetPolicy{ExecutablePath, AllowedRoots}`.
     * A planted `gobble-linux-amd64.tar.gz` gives "no exact asset". It is not `ErrIntegrity`.
     * A running-copy end-to-end test is modelled on go-core-lib `selfupdate/e2e_running_test.go`; that harness is unexported, so the pattern is copied, not imported.
  7. **Step 5 (bridge).** Follows 0003-MADR's amendment of this date: relocated installs, packages, JSONC, tolerant reads, and `trust import`.
  8. **Release gate.** The gate table adds the mcremote rows (MADR "Driven by mcremote") and the live probes from magic-cli-remote 0179 D10, run by the owner against the build that will be tagged, before the tag is pushed.
* **X1.** `/deep-research` is advertised with the skill.
* **X3.** `ask_user` and `_gobble/status` land here.
* **X5, step 2.** Subscription OAuth candidates are xAI Grok and Kilo device login, each with a recorded terms check. Anthropic Max, Gemini and Copilot subscription logins are out of the v1 line.
* **X5, step 6.** Cache warming stays blocked until the SDK supports cache hints and a cache read/write split. Record that as the skip reason if X5 runs first.
* **Companion work.** Replaced by magic-cli-remote `docs/decisions/0179-MADR-gobble-native-acp-provider.md` (D1–D10). This plan still does not mutate that repository.

**2026-10-02 — source-verified provider, update and TUI integration.** Follows
the amendments of this date in 0004-MADR and 0005-MADR. The observations
are go-llmprovider-sdk `67fc56e` (no tag), go-core-lib `v1.3.0`
(`f97c681`) and go-tui-lib `v0.1.0` (`5c57806`). This PLAN remains
`proposed`; no F phase has run.

* **F4 replaces its earlier usage and error assumptions.** Use the SDK's
  `Response.Usage` for reported input, output, reasoning and cached tokens.
  If every returned count is zero, estimate the turn and mark it in
  `llm.Usage` and the user-visible ledger. Otherwise treat the returned
  counts as reported, including zero components. The SDK has no
  cache-write count, so do not label one as reported. Match `*APIError`
  and classify with sentinels
  in the order from 0004-MADR. Remove the transitional
  `*RateLimitError` / `*IncompleteError` cases. Use OAuth and
  `RefreshLocker` from `llmprovider/auth`, not their pre-S8c path. Pass
  credentials and catalog options explicitly; do not use the SDK's opt-in
  environment helpers as gobble defaults. Keep `WithModelProbes(false)`.
  **Accept:** fixture responses from each of the four wire families yield
  reported counts, an all-zero response is marked estimated, and an `APIError`
  429 preserves `RetryAfter`; quota and overflow retain their more specific
  classification. The no-billed-probes accept remains.
* **F9 uses go-tui-lib for core TUI.** The enhanced terminal TUI is not
  the default; the default is the native terminal CLI. Before adding
  `go-tui-lib` to `go.mod`, inspect its newest tag, check the known
  `v0.1.0` defects from its proposed hardening PLAN, and choose a
  corrected tag. Core TUI is go-tui-lib. gobble does not reimplement core
  TUI. Where that tag lacks a core behaviour (transcript, tool cards,
  permission dialog, editor, pickers, or anything else that is core TUI),
  F9 waits for the library rather than copying it. `internal/tui` alone
  imports `go-tui-lib`; `_test.go` files may use `tuitest`. **Accept:**
  record the chosen tag and the defect check before committing F9. A
  fixture renders the ACP transcript and a permission dialog within a
  workspace at 80×24 and 120×40 only when the selected tag provides that
  behaviour. If it does not, F9 records the wait and does not land a
  local copy.
* **F10 pin and release policy.** Re-resolve the newest suitable stable
  `go-core-lib v1.x` tag when F10 starts; `v1.3.0` is today's tag, not a
  permanent pin. The workflow `uses:` SHA must be the peeled commit of
  the tag required in `go.mod`. Its channel API is available, but F10
  passes no channel and publishes only strict stable gobble tags. No
  `selfupdate/cli`, `buildinfo` or `updatetea` import is presumed. The
  `Checker.Check`, `RunWith` and `Start`/`Stream` paths and existing
  acceptance fixtures remain; record the selected tag and SHA in this
  PLAN when the phase runs.

**2026-10-02 — Phase D source-inventory correction.** The preceding F10
inventory was pinned to `v1.3.0`. `go-core-lib v1.3.1` now peels to
`351bd6a5ffbd4b352cacc5234f4a8c504c3c1b69`, and its released Go
packages remain `selfupdate` and `selfupdate/selfupdatetest`. Development
Commit `bf7221a5408984299a8511eb98f8f06eadd133bd` adds `buildinfo`,
which is not in that tag. F10 still re-resolves a stable tag and checks the
released API before importing it; its stable-only policy is unchanged.
The SDK's later `efd9c61` commit changes catalog and setup-wizard code but
does not change the F4-facing contract assessed at `67fc56e`. The TUI
library's later `7907590` commit changes documentation only; its released
source remains at `v0.1.0`.

**2026-10-03 — go-core-lib renamed go-selfupdate-lib.** Follows the source
corrections of this date in 0004-MADR and 0005-MADR. No new phases.
Changes to F10:

1. **Pin.** Re-resolve `github.com/maccavelli/go-selfupdate-lib` to its
   newest `v1.x` (`v1.5.0` or later), with the same SHA in `go.mod` and in
   `uses:`. The `uses:` line names `maccavelli/go-selfupdate-lib`.
2. **Surface.** Before binding, compare the released `selfupdate/cli` and
   `buildinfo` (since `v1.4.0`) with the MADR's update surface and with
   gobble's `internal/buildinfo`. Record the choice in F10's execution
   record.
3. **Accept rewrite.** The running-copy end-to-end test is modelled on
   go-selfupdate-lib `selfupdate/e2e_running_test.go`.

**2026-10-07 — F1 made executable (owner's decisions of 2026-10-07: four sub-phases; settings as Go options; images detected now, decoded and resized later; an in-house gitignore matcher).**

F1's eight steps name their tools but not their contracts. They are expanded here before execution. The steps above are kept as written; where this text differs, this text is the step to follow. F1a is expanded in full below. F1b, F1c and F1d are each expanded in an entry of their own when they start.

The facts, from two read-only surveys of 2026-10-07: Pi at `312184edb`, and this tree at `f6253fc`.
* **gobble's tools are 0002-PLAN Phase 3's minimum.** `tool/builtin` has `read {path}`, `write {path, content}`, `edit {path, oldText, newText}` and `bash {command, timeout?}` (`builtin.go:25-27`).
  * Paths are `filepath.Join(env.Cwd, path)`, and absolute paths are taken as given (`builtin.go:40-45`).
  * `tool.Env` holds only `Cwd` (`tool/tool.go:71-73`).
  * `internal/fsx` and `tool/toolsearch` are placeholders.
  * Nothing in the module uses `os.Root`, and there is no per-path lock.
  * acpserver ignores the client's capabilities in `initialize` (`acpserver/agent.go:199-207`), never reads `additionalDirectories`, and does not advertise them.
* **Pi's file tools** (`packages/coding-agent/src/core/tools/`):
  * `read {path, offset?, limit?}`: 1-based offset, and head truncation at 2000 lines or 50 KB (`truncate.ts:11-12`), with these notices (`read.ts:143-178`):
    * `[Showing lines s-e of N. Use offset=… to continue.]`
    * the same with ` (50.0KB limit)`
    * `[N more lines in file. …]`
    * the single-long-line hint
  * Images are found by magic bytes, not extension (`utils/mime.ts`).
  * `edit {path, edits:[{oldText, newText}]}` (`edit.ts`, `edit-diff.ts`):
    * each edit is matched against the original;
    * uniqueness is counted in fuzzy-normalised space;
    * overlaps are refused;
    * BOM and CRLF are kept;
    * the fuzzy fallback is NFKC, trailing whitespace, single quotes, double quotes, dashes, then special spaces, in that order (`edit-diff.ts:34-55`).
  * `prepareEditArguments` repairs three shapes: `edits` sent as a JSON string, a single edit object, and the legacy top-level `oldText`/`newText` (`edit.ts:103-134`).
  * `write` creates parent directories.
  * Edit and write share a mutation queue keyed by real path (`file-mutation-queue.ts:16-61`).
  * Pi has no confinement and no additional directories (`path-utils.ts`, `utils/paths.ts:76-107`).
* **0005-MADR's rows** (`:118-134`) add two things Pi lacks:
  * `os.Root` over `cwd` plus ACP `additionalDirectories`, where "a path outside those roots requires a permission decision";
  * the client's `fs/*` methods when advertised.
* **The pinned go-llmprovider-sdk `v1.2.1` carries text only.** `MessageItem` and `FunctionCallOutputItem` hold a `Text string` (`llmprovider/item.go:18-45`), and no image part exists. So no image can reach a model, which 0005-MADR's amendment of 2026-10-01 already says (`:570-573`).
* **No settings mechanism exists.** `internal/config` is a placeholder, and the config surface is undecided (0008-MADR). `shellPath`, `shellCommandPrefix`, `tools.bash.useClientTerminal` and `images.autoResize` have no home.
* **No MADR names a gitignore library.** 0005-MADR names `doublestar` for globs (`:123`) and asks only for "a gitignore-aware walk". `golang.org/x/text`, which holds `unicode/norm` for NFKC, is already a direct requirement.

The owner decided four questions on 2026-10-07:
1. **Four sub-phases, each one commit** with its own execution record:
   * **F1a, the file tools:** steps 1 and 3, step 2's detection only, and step 6.
   * **F1b, bash and powershell:** step 4.
   * **F1c, grep, find and ls:** step 5.
   * **F1d:** step 7 (ACP `fs/*` and `terminal/*` routing, and tool-call `locations`) and step 8 (`todo`, `tool_search`, the MCP resource tools).
2. **Settings are Go options for now.**
   * Typed fields on `builtin.Options` and `acpserver.Options`, which an embedder can set. The `gobble` binary uses the defaults.
   * Additional directories come from ACP `additionalDirectories`, which gobble starts advertising in F1a.
   * The user-facing settings wait for the config decision. They are named under Deferred.
3. **Images are detected now, and decoded and resized later.**
   * F1a sniffs the magic bytes by Pi's rules, and `read` answers an image with a text note.
   * Step 2's decode, EXIF orientation and resize, and `golang.org/x/image`, wait for the provider SDK to carry image content. They are still in the 1.0 gate.
4. **gitignore matching is in-house,** in `internal/fsx`, written to git's documented rules. It lands with F1c, which is the first phase to walk directories.

**F1a — the file tools (steps 1, 2 in part, 3 and 6).**

Choices made within the wording:
* **The roots.**
  * `tool.Env` gains `Roots []string` (the session `cwd` first, then the additional directories, all absolute) and `AllowOutside bool`.
  * acpserver advertises `sessionCapabilities.additionalDirectories`. It reads `additionalDirectories` on `session/new`, `load`, `resume` and fork, keeps them on the live session, and returns them in `session/list`'s `SessionInfo`.
  * A relative additional directory is refused with `invalid_params`, as the ACP schema requires absolute paths.
* **`internal/fsx`** is the confinement and the queue (0004-MADR's package map, `:252`):
  * `Resolve(cwd, path string) string` follows Pi's `resolvePath` order (`utils/paths.ts:76-107`):
    1. Unicode spaces become a plain space;
    2. a leading `@` is stripped;
    3. on Windows, `/c/…`, `/mnt/c/…` and `/cygdrive/c/…` become `C:\…`;
    4. `~` is expanded;
    5. `file://` URLs become paths;
    6. a relative path is joined to `cwd`, then cleaned.

    Pi's read-only fallbacks for a missing path come after these: the macOS `AM`/`PM` narrow space, NFD, and U+2019 (`path-utils.ts:86-118`).
  * `Workspace{Roots}` with `Outside(abs string) bool`. A path is inside when, after resolving the symlinks of its deepest existing ancestor, it is under some root.
  * `ReadFile` and `WriteFile`. An inside path is opened through an `os.Root` for its root, so a symlink or `..` that escapes is refused by the runtime too. An outside path is opened directly, and only when the call allows it.
  * `WriteFile` writes a temporary file beside the target and renames it into place (`Root.Rename`). It keeps an existing file's permission bits, and creates parent directories (`Root.MkdirAll`). A new file is `0o644`, as Pi's `writeFile` gives under the usual umask, rather than today's `0o600`.
  * `Lock(path) (unlock func())` serialises calls on one real path (`EvalSymlinks`, else the absolute path), process-wide. Different paths do not wait for each other.
* **`tool`, the contract:**
  * an optional interface `Confined{ Outside(call Call, env Env) []string }` returns the paths outside `env.Roots` that a call names;
  * `WithPrepare(func(jsontext.Value) (jsontext.Value, error))` rewrites a call's arguments before they are validated, for edit's repairs.
* **`agent`, the approval:** when a tool is `Confined` and a call names paths outside the roots, the call needs a decision even if it is read-only.
  * The rule's title gains ` (outside the workspace)`.
  * Only an allowed call runs, with `AllowOutside` set on its `Env`.
  * With no `Policy`, such a call is refused with `"<path> is outside the workspace (the working directory and the additional directories), and no permission policy can approve it."`. No policy means no one to ask, so the safe answer is no.
* **`read`** is Pi's tool:
  * the schema `{path, offset?, limit?}`, with Pi's descriptions;
  * the notices quoted above, worded as Pi words them, with Pi's line count (`split("\n")`, so a trailing newline adds an empty line);
  * CR and the BOM are kept;
  * an offset beyond the end gives Pi's message.
  * An image (JPEG, PNG that is not animated, GIF, WebP, BMP, by Pi's magic-byte rules) gives `Read image file [<mime>]` and `[The image was not sent: gobble cannot send images to models yet.]`. Pi's own note, "Current model does not support images", would be untrue here: the gap is the SDK's, not the model's.
* **`edit`** is Pi's multi-edit, with Pi's error messages:
  * `WithPrepare` makes the three repairs;
  * the BOM is split off, and CRLF is detected from the first `\n`;
  * matching is against the original: exact first, then the fuzzy chain in Pi's order (NFKC from `golang.org/x/text/unicode/norm`);
  * uniqueness is counted in fuzzy-normalised space;
  * overlaps are refused after sorting;
  * edits are applied in reverse, and only the touched lines are rewritten in fuzzy mode;
  * an edit that changes nothing gets Pi's no-change error.

  The model reads `Successfully replaced N block(s) in <path>.`. The client keeps today's ACP `diff` content and summary.
* **`write`:** `Successfully wrote to <path>` for the model, through `fsx.WriteFile` and `fsx.Lock`. Both mutating tools hold the lock from their read to their rename.
* **`truncate.go`** ports Pi's `truncateHead`, `truncateTail`, `truncateLine` and `formatSize` (`truncate.ts`), so F1b and F1c reuse them. `bash` keeps today's `tail` until F1b replaces it.
* **The display** is unchanged: the 0008-MADR D19 titles and summaries. The summaries follow the new texts.

**Files:**
* `tool/tool.go` and `tool/tool_test.go`;
* `internal/fsx`:
  * `doc.go`;
  * new `resolve.go`, `workspace.go`, `write.go` and `lock.go`;
  * new `rename_windows.go` and `rename_other.go` *(added by deviation 1)*;
  * new `testdata/fuzz/FuzzOutside/82c2975c430ac608` *(added by deviation 2)*;
  * their tests, and `fuzz_test.go`;
* `tool/builtin`:
  * `builtin.go`;
  * `files.go`, split into new `read.go`, `write.go` and `edit.go`;
  * new `editmatch.go`, `truncate.go` and `sniff.go`;
  * new `paths.go` *(added by deviation 3)*;
  * `builtin_test.go`, with new `read_test.go`, `edit_test.go`, `truncate_test.go` and `fuzz_test.go`;
* `agent/agent.go` and its tests;
* acpserver:
  * `agent.go`, `commands.go` (fork) and `prompt.go` (the `Env`);
  * their tests, and any golden whose bytes the new capability or texts change;
* tests elsewhere that assert the old tool texts, found by `go test ./...` and listed in the record;
* `docs/architecture.md`: the `internal/fsx`, `tool` and `tool/builtin` rows;
* this PLAN, and `docs/README.md`.

**Steps:**
1. `tool`: `Env.Roots`, `Env.AllowOutside`, `Confined` and `WithPrepare`, with tests.
2. `internal/fsx`, with table tests and two fuzz targets:
   * `FuzzResolve`: no path that `Workspace` calls inside resolves outside every root;
   * `FuzzOutside`: `Outside` agrees with a lexical check on paths without symlinks.
3. `tool/builtin`: truncate, then read, write and edit, with Pi's cases as table tests, and a `FuzzEditMatch` target (no panic; any result it accepts contains each `newText`).
4. `agent`: the outside-roots approval.
5. acpserver: the capability, the directories, and `Env.Roots`.
6. Docs and this entry's execution record.

**Accept:**
* `read`: an offset, a limit, the three notices, the long-line hint, an offset past the end, and each image kind's note. Pi's strings are compared exactly.
* `edit`: one edit and several; ambiguous, not found, overlap, empty and no-change; CRLF kept; BOM kept; each fuzzy normalisation; and the three argument repairs.
* `write`: parent directories, a kept mode, and an atomic replace (no partial file is ever observed by a concurrent reader in the test).
* Confinement:
  * a `../` path and a symlink out of `cwd` are outside;
  * a read-only call to one asks the policy, and runs only when allowed;
  * with no policy it is refused;
  * a path in an additional directory is inside.
* `fsx.Lock` serialises holders of one path, and does not block another path.
* acpserver advertises `additionalDirectories`, and a session made with one can read a file in it without a permission request.
* `FuzzResolve`, `FuzzOutside` and `FuzzEditMatch` each run for 30 s with no failure.

Each is shown failing first, on a scratch copy, with the failure quoted:
* the plan's own: confinement through `filepath.Join` instead of `os.Root` fails the symlink-escape test;
* edits matched progressively instead of against the original: the overlap test;
* CRLF not restored: the CRLF test;
* the smart-quote step removed from the fuzzy chain: its case;
* `fsx.Lock` made a no-op: the serialisation test;
* `agent` ignoring `Confined`: the outside-read approval test;
* `read`'s continuation offset off by one: its notice test.

**Verification:**
* `go test ./...` on Windows and in WSL, and WSL `go test -race ./...`;
* the three fuzz targets for 30 s each, on Windows;
* `golangci-lint` for linux, darwin and windows;
* `make preflight` on Windows and in WSL;
* the pre-add check on every changed Go file;
* the identifier and hidden-character scans.

The agent stages; the owner commits.

**Deferred, named:**
* to F1b, F1c and F1d: their steps, each expanded when it starts;
* to the provider SDK's image support, still in the 1.0 gate: step 2's decode, EXIF orientation and resize, `golang.org/x/image`, and images in MCP results (0002-PLAN Phase 5's deferral). `read`'s note changes then;
* to the config decision: user-facing `shellPath`, `shellCommandPrefix`, `tools.bash.useClientTerminal`, `images.autoResize`, and a CLI way to pass additional directories. Until then they are Go options, and additional directories come only from an ACP client;
* to F6: remembering an outside-root approval. Today's ACP policy offers allow-once and reject-once only (`acpserver/prompt.go:194-213`).

**F1a deviations, 2026-10-07.** Found while executing, each fixed before it was put to the owner, nothing staged until the owner decided. The owner kept all three on 2026-10-07.

1. **Windows refuses to rename over a file another process holds open.**
   * `TestWriteIsAtomic` failed: `renameat .big.txt.gobble-a00a09d243ca.tmp big.txt: Access is denied.` Its concurrent reader held the file, as an open editor would.
   * Decision: on Windows, a rename failing with `ERROR_ACCESS_DENIED` or `ERROR_SHARING_VIOLATION` is retried for up to 2 s, with pauses doubling from 1 ms to at most 100 ms. Go's `cmd/go` robustio does the same.
   * The write stays atomic. New files `internal/fsx/rename_windows.go` and `rename_other.go`.
2. **`FuzzOutside` found a Windows path that `Outside` called inside and `os.Root` would refuse.**
   * The failure: `":": inside = true, lexical check says false`, on its first run. The case-folding comparison (`relativeFold`) skipped the `filepath.IsLocal` check the Unix comparison makes, so a `:` element, which names a stream on Windows, counted as inside.
   * It failed safe, because `os.Root` refuses the name, but the two disagreed.
   * Decision: `relativeFold` checks `filepath.IsLocal` too. The fuzzer's saved input `internal/fsx/testdata/fuzz/FuzzOutside/82c2975c430ac608` is kept as a regression seed. It failed before the fix and passes after.
3. **`tool/builtin/paths.go`** holds the path helpers that were in `builtin.go` and `files.go`: `resolve`, `roots`, `workspace`, `outsidePath`, `shown`, `argPath`, `titlePath` and `unwrapPath`. Decision: keep it as its own file, which F1c's tools share.

**2026-10-07 — F1 restructured; F1a reworked to the reference rule (owner's choices of 2026-10-07; 0005-MADR amendment "the best of several harnesses, made gobble's own").**

The owner asked why F1a copied Pi rather than using it as a baseline, and asked for the best of Pi, opencode, Kilo and goose, made gobble's own. [0010-REPORT](../reports/0010-REPORT-harness-design-survey.md) holds the survey, and the MADR amendment holds the rule and twelve choices. F1a was staged and not committed, so it is reworked before its commit. Its entry above and its record below stay as the history of the first build.

**F1's sub-phases now.**
* **F1a** — file tools. Rework before commit, as below.
* **F1b** — bash and powershell, with opencode's `workdir` (choice 1). Its permission parsing is F6's (choice 5).
* **F1c** — grep, find and ls.
* **F1d** — todo, `tool_search`, the MCP resource tools, and ACP routing with `locations`.
* **F1e** — `apply_patch` (choice 1), a new sub-phase. It is offered to GPT-family models in place of edit and write, and is expanded in an entry of its own when it starts.

**F1a rework — what changes.**
1. **Messages are gobble's** (rule 3). `piText` is deleted. Errors read `<tool> <path>: <what>; <how to fix>`; for example:
   * `edit a.go: edits[1].oldText was not found; it must match the file exactly, or nearly (whitespace, quotes and indentation are forgiven)`
   * `edit a.go: edits[0].oldText occurs 3 times; include more surrounding lines so it occurs once`
   * `edit a.go: edits[0] and edits[1] overlap; merge them into one edit or choose separate regions`
   * `edit a.go: the edits change nothing`
   * `edit: edits needs at least one replacement`
   * `read a.go: offset 100 is past the end of the file (3 lines)`

   The model reads `edited a.go: 2 replacements` and `wrote a.go (12 lines)`. Tool descriptions are rewritten in gobble's words, with the same guidance.
2. **read is opencode's design** (choice 3).
   * Output is numbered, `N: text`, one line each. A CR at a line end and a leading BOM are not shown, since edit matches line-feed-normalised text.
   * A line over 2000 characters is cut, ending `… (line cut at 2000 characters)`. This replaces Pi's single-long-line `sed` hint.
   * Limits: 2000 lines and 50 KB. A final newline does not count as a line.
   * Footers:
     * `(lines 1-2000 of 2500; continue with offset=2001)`
     * `(lines 1-243 of 500, 50 KB limit; continue with offset=244)`
     * `(end of file, 500 lines)`
   * A file with a NUL byte, or more than 30% unprintable bytes, in its first 4 KB is refused: `read x.bin: a binary file; read cannot show it`. Images keep their note, and an animated PNG counts as an image, unlike Pi.
   * A missing file names up to three entries of its directory whose names contain, or are contained in, the name asked for, ignoring case: `read a.go: no such file; did you mean b/a_test.go?`.
   * A directory is listed: sorted, a `/` on directories, and paged by `offset` and `limit`. Its footer reads `(entries 1-500 of 812; continue with offset=501)`.
   * **Kilo's check already holds** in gobble's design. A path outside the roots is asked about (`Confined`) before `Run` reveals whether it exists, and inside the roots the workspace is the model's to see. No further check is added.
3. **edit matching is layered** (choice 2).
   * For each edit, in order: exact; then Pi's normalisation (which, as now, moves the call into normalised space with lines kept); then opencode's **line-trimmed**, **indentation-flexible** and **escape-normalised** strategies, run against the same base.
   * Each layer needs one candidate. Two or more is the "occurs N times" error.
   * Matching stays against the original, and overlaps are refused.
   * **The oversize guard.** A match found by any layer other than exact fails when its region has at least `max(oldLines+3, 2×oldLines)` lines or more than `max(len+500, 4×len)` bytes: `edit a.go: edits[0] matched a region much larger than its oldText; read the file again and resend the edit`.
4. **Compare-and-swap** (choice 2). Under the file's lock, the bytes edit read are compared with the file again just before the rename. A difference fails: `edit a.go: the file changed while the edit was being made; read it again and resend the edit`. That is a change by another process; gobble's own writers are already held off by the lock.
5. **Quirks are dropped** (rule 3).
   * `truncateLine` counts runes, not UTF-16 units.
   * `formatSize` is Go's `%.1f`, with no `toFixed` emulation.
   * There is no single-long-line hint (item 2 covers it).
   * Pi's other-spelling fallbacks for a missing path stay, because they find files macOS users paste. That is a usability reason, recorded here.
6. **Tests** check behaviour, and gobble's messages as gobble's own (rule 4). The ported Pi cases remain as behaviour cases. New cases:
   * each opencode strategy;
   * the oversize guard;
   * compare-and-swap, where a test writes the file between the read and the rename through a test hook;
   * binary refusal, "did you mean", directory reads, line numbering and the line cut.

**Files** — the F1a set as staged, and:
* `tool/builtin/editmatch.go`, with new `editstrategy.go`;
* `read.go`, with new `readdir.go`;
* `sniff.go` and `truncate.go`;
* their tests;
* the acpserver and agent tests that assert tool texts.

**Accept,** in addition to F1a's own:
* `go test ./tool/builtin/` covers every case in item 6.
* `FuzzEditMatch` keeps its property: an edit it accepts contains each `newText` and changes the content. It runs 30 s again.

Each is shown failing first, on a scratch copy:
* the line-trimmed strategy removed: its case;
* the oversize guard removed: its case;
* compare-and-swap removed: its case;
* line numbers off by one: the numbering case;
* binary detection removed: its case.

F1a's seven negatives are run again with their new expectations.

**Verification:** F1a's, run again in full.

The agent stages; the owner commits.

**F1a rework, 2026-10-07 — complete (staged with F1a; the owner commits).** It ran as the entry "F1 restructured; F1a reworked" wrote it.

* **Files.** The F1a set, and:
  * new `tool/builtin/editstrategy.go` and `readdir.go`;
  * `editmatch.go`, `edit.go`, `read.go`, `write.go`, `sniff.go`, `truncate.go` and `builtin.go` (`count`);
  * `internal/fsx/workspace.go` (`ReadDir`);
  * the builtin tests;
  * `acpserver/extensions_test.go` and `prompt_test.go`, for the tool texts;
  * the 0005 pair, [0010-REPORT](../reports/0010-REPORT-harness-design-survey.md), `docs/architecture.md` and `docs/README.md`.
* **Accept.**
  * `go test ./...` passed on Windows.
  * Read:
    * line numbering;
    * the footers;
    * the byte limit at line 238, computed by hand in the test;
    * the line cut;
    * BOM and CR hidden;
    * binary refusal;
    * "did you mean";
    * directory listing and paging;
    * every image kind, the animated PNG included.
  * Edit:
    * the 28 behaviour cases first written from Pi's suite, with gobble's messages;
    * one case for each of opencode's three strategies, for the oversize guard, and for ambiguity at every layer;
    * compare-and-swap through the `beforeWrite` test hook;
    * the three argument repairs.
  * `FuzzEditMatch`, `FuzzResolve` and `FuzzOutside` ran 30 s each (see Gates).
* **Negative tests,** on scratch copies, each failing at its own test.
  * F1a's seven, with their new expectations:
    * the `filepath.Join` confinement;
    * progressive matching;
    * CRLF;
    * smart quotes;
    * the lock;
    * `Confined`;
    * the continuation offset.
  * The rework's five:
    * the line-trimmed strategy removed: `TestEditStrategies/line-trimmed … oldText was not found`;
    * the oversize guard removed: `result = "edited f.txt: 1 replacement" (isError false)`;
    * compare-and-swap removed: `result "edited f.txt: 1 replacement", want "edit f.txt: the file changed while …"`;
    * numbering off by one: `output "50: Line 51\n…", want it to start with line 51`;
    * binary refusal removed: `result "1: PK\x03\x04\x00\x00data\n(end of file, 1 line)" (isError false)`.
* **What the entry predicted wrongly.**
  1. **Counts needed singular forms.** The first run showed `(end of file, 1 lines)`, and then `3 entrys`. `count(n, one, many)` gives both forms, and a one-line test case now covers it.
  2. **One negative's expected string was wrong.** Compare-and-swap's removal failed at the test's first assertion, before the line the harness looked for. The harness was corrected; the test was not.
  3. **Exact matching now comes before the fuzzy count.** Pi counts uniqueness in normalised space even when `oldText` occurs exactly once. gobble takes an exact unique match as the answer, because it is unambiguous. The behaviour cases pass unchanged.
  4. **Kilo's existence check needed no code.** gobble asks about an outside path before `Run`, and inside the roots the workspace is the model's to see.
  5. **Lint found one issue in the rework.** `ineffassign` flagged an assignment to `out` in `applyEdits` whose value was always overwritten. The branch now assigns once on each path. The twelve negatives were run again afterwards, and all twelve failed as before.
* **Gates.**
  * lint printed `0 issues.` for linux, darwin and windows.
  * Windows `go test ./...` had no `FAIL` or `panic:`.
  * `make preflight` printed `preflight passed` on Windows and in WSL.
  * WSL `go test -race ./...` exited 0.
  * `FuzzEditMatch`, `FuzzResolve` and `FuzzOutside` each ran 30 s on Windows with no failure.

**2026-10-07 — F1b made executable (owner's decisions of 2026-10-07: two shell tools; a 2-minute default timeout capped at 10 minutes; SIGTERM, 3 s, then SIGKILL; the full output in the state directory for 7 days).**

F1 step 4, under the rule of 0005-MADR's amendment "the best of several harnesses". The sources are 0010-REPORT §1 (Shell) and the F1 survey of Pi.

The facts:
* **`tool/builtin/bash.go` runs `bash -c` from `PATH`.** On Windows it prefers Git Bash and refuses System32's WSL launcher. A cancel kills the shell alone, through `exec.CommandContext`, with `WaitDelay` 2 s. Output collects in one unbounded buffer, and `tail` keeps the last 2000 lines or 50 KB. A timeout is optional, in seconds, with 0 for no limit. There is no `powershell` tool, no process group, no environment markers and no full-output file.
* **`mcpclient` already owns process trees.**
  * On Unix, `Setpgid` puts the server in its own group, which is signalled (`proc_unix.go`).
  * On Windows, a Job object with `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` holds it, and `TerminateJobObject` ends it (`proc_windows.go`).
  * Windows has no graceful signal for a tree, so its "terminate" is that kill.
* **The 0003 markers** are `AI_AGENT=gobble`, `GOBBLE_SESSION_ID`, `GOBBLE_SESSION_FILE`, `GOBBLE_PROVIDER`, `GOBBLE_MODEL` and `GOBBLE_THINKING_LEVEL` (0003-MADR, table). The temporary-file name is `gobble-bash-<id>.log`. `tool.Env` carries none of these.
* **How the model chooses tools.** The CLI builds its tools from `builtin.Tools()` (`internal/cli/agent.go:155`). acpserver falls back to `builtin.Tools()` when `Options.Tools` is nil.

The owner decided four questions on 2026-10-07:
1. **Two tools.** `bash` runs everywhere, with Git Bash on Windows. `powershell` exists on Windows only, preferring `pwsh.exe` to `powershell.exe`. They share one runner, and each has a description written for its shell. This is 0005-MADR's row.
2. **`timeout` is in seconds,** defaulting to 120 and capped at 600. The cap is a Go option.
3. **Stopping.** SIGTERM goes to the process group, and SIGKILL follows after 3 s if anything is left. On Windows the Job object kills the tree. mcpclient's process-tree code moves to a shared internal package.
4. **The full output** is `gobble-bash-<id>.log` in gobble's state directory, owner-only, and removed after 7 days. read may open files there without an outside-workspace question.

Choices made within the wording:
* **`internal/proctree`,** a new internal package, holds what mcpclient had:
  * `Prepare(*exec.Cmd)`;
  * `Attach(*os.Process) (*Tree, error)`;
  * `(*Tree).Terminate`, `Kill`, `Exited` and `Release`.

  mcpclient's `proc.go` calls it, and `proc_unix.go`, `proc_windows.go` and their tests move into it.
* **`builtin.Options`** carries:
  * `ShellPath`, which replaces the bash search;
  * `ShellCommandPrefix`, put before the command on a line of its own, as in Pi's `${prefix}\n${command}`;
  * `MaxTimeout`, default 600 s;
  * `OutputDir`, the full-output directory; the CLI passes `<state>/output`, and empty means the OS temp directory.

  `ToolsWith(Options)` builds the tools; `Tools()` is `ToolsWith(Options{})`. The CLI passes `OutputDir`. Settings stay Go options, by the owner's decision 2 of the entry "F1 made executable".
* **`bash` and `powershell` take** `{command, timeout?, workdir?}`.
  * `workdir` is opencode's: "use this instead of cd". It resolves against `cwd`. It is `Confined`, so a `workdir` outside the roots is asked about, and it must be an existing directory.
  * `powershell` runs `-NoLogo -NoProfile -NonInteractive -Command`, with the UTF-8 output prefix that 0005-MADR's row names: `[Console]::OutputEncoding=[System.Text.Encoding]::UTF8;`.
* **Environment.**
  * `tool.Env` gains `Environ []string`. acpserver fills it with the six 0003 markers for the session.
  * The child starts from `os.Environ()`. Inherited `GOBBLE_SESSION_*`, `GOBBLE_PROVIDER`, `GOBBLE_MODEL` and `GOBBLE_THINKING_LEVEL` are removed first, so a gobble run inside a gobble command sees its own session, not its parent's.
* **Output.**
  * stdout and stderr go to one writer. It holds the output in memory until it passes 2000 lines or 50 KB.
  * From then on everything goes to the log file, and only a tail of at most 100 KB stays in memory.
  * The model gets the last 2000 lines or 50 KB (`truncateTail`). When cut, the output begins `(output cut: the last N of M lines are shown; the whole output is in <path>)`.
  * The status line stays gobble's: `[exit status N]`, or `[stopped after N s; give a larger timeout, at most 600]`.
  * The first log written in a process sweeps that directory's `gobble-bash-*.log` files older than 7 days.
* **The read root.** read built by `ToolsWith` adds `OutputDir` to its own roots for reading. write and edit do not, so a model cannot write there without asking.
* **Stopping, in the runner.** When the context is done (a cancel or the timeout), the runner terminates the tree. After 3 s it kills it, and it waits at most `WaitDelay` 2 s more for pipes a grandchild still holds. A cancel stays an error from `Run`, which the agent turns into its cancelled text.

**Files:**
* new `internal/proctree`:
  * `doc.go`, `tree_unix.go` and `tree_windows.go`;
  * tests moved from mcpclient's;
  * a new tree test that a grandchild dies;
* `mcpclient/proc.go`; `proc_unix.go`, `proc_windows.go` ~~and their tests~~ deleted *(deviation 2: the two test files stay)*;
* `tool/tool.go` (`Env.Environ`) and its test;
* `tool/builtin`:
  * `builtin.go` (`Options`, `ToolsWith`) and `bash.go`;
  * new `shell.go` (the runner), `output.go` (the spilling writer and the sweep) and `powershell.go`;
  * `read.go` (the extra read root);
  * the tests;
* `acpserver/agent.go`: the markers in `env()`;
* `internal/cli/agent.go`: `ToolsWith` with `OutputDir`;
* `agent/agent_test.go`, `internal/cli/main_test.go` and `acpserver/agent_test.go` *(added by deviation 1)*;
* `docs/architecture.md`: the `internal/proctree`, `tool/builtin` and `mcpclient` rows;
* this PLAN and `docs/README.md`.

**Accept:**
* bash:
  * `echo` gives its output and `[exit status 0]`;
  * a failing command is an error result with its status;
  * `workdir` runs in that directory;
  * `ShellCommandPrefix` runs first;
  * the markers are in the child's environment, and an inherited `GOBBLE_SESSION_ID` is replaced.
* timeouts: none given is 120 s; 9999 is capped at 600; a 1 s timeout stops `sleep 30` within about 4 s with the `stopped after` line.
* process tree: a cancel ends a command's grandchild (`sleep 60 &` in a subshell). On Unix, a trapped SIGTERM handler runs before the kill.
* output: 3000 lines give the last 2000 with the cut notice, and the log file holds all 3000; a log 8 days old is swept, and one 6 days old is kept.
* read opens a file in `OutputDir` with no permission question. write to it still asks.
* powershell, on Windows only: `Write-Output hi` gives `hi`, and its UTF-8 prefix keeps `é` intact.
* mcpclient's stdio tests pass unchanged.

Each is shown failing first, on a scratch copy:
* the process group not set (`Prepare` a no-op): the grandchild test;
* no SIGTERM before SIGKILL: the trap test, on Unix in WSL;
* the default timeout dropped: its case;
* the markers not set: its case;
* the spill disabled: the log-file case;
* the sweep's age comparison inverted: the sweep case.

**Verification:**
* `go test ./...` on Windows and in WSL, and WSL `go test -race ./...`;
* `golangci-lint` for three GOOS;
* `make preflight` on Windows and in WSL;
* the pre-add check;
* the identifier and hidden-character scans.

The agent stages after the owner's commit of F1a; the owner commits.

**Deferred, named:**
* to F6: permission patterns from parsed commands (`mvdan.cc/sh`, choice 5);
* to F1d: running a command in the client's terminal (`terminal/*`);
* to X2: background commands;
* to the config decision: user-facing `shellPath`, `shellCommandPrefix` and the timeout cap.

**F1b deviations, 2026-10-07.** Both were decided by the owner on 2026-10-07.

1. **The Windows-only `powershell` tool made three tests depend on the OS.**
   * `agent/agent_test.go:68` hard-codes the four-tool list, and so does `internal/cli/main_test.go:90`.
   * acpserver's `TestSession` golden records the usage estimate, which counts tool names in the system prompt: `"used":115` on Windows against 112 elsewhere.
   * Decision: the two lists come from `builtin.Names()`. `TestSession` gives its agent an explicit tool set (read, write, edit, bash), so the golden is one file for every OS and stays at 112. The three test files join the phase.
2. **mcpclient's `proc_unix_test.go` and `proc_windows_test.go` are not tests of the moved code.** They are the `alive(pid)` helpers mcpclient's own `proc_test.go` uses. Decision: they stay. proctree has its own copies, and the plan's "and their tests" is struck.

**F1b, 2026-10-07 — complete (unstaged until the owner commits F1a, then staged; the owner commits).** It ran as the entry "F1b made executable" wrote it, with deviations 1 and 2.

* **Files.**
  * New `internal/proctree`: `doc.go`, `tree_unix.go`, `tree_windows.go`, `tree_test.go`, and the `alive` helpers.
  * `mcpclient/proc.go`; `proc_unix.go` and `proc_windows.go` deleted.
  * `tool/tool.go` (`Env.Environ`).
  * `tool/builtin`:
    * `builtin.go` (`Options`, `ToolsWith`, `goosWindows`);
    * `bash.go`, with new `shell.go`, `output.go` and `powershell.go`;
    * `read.go`, `readdir.go` and `paths.go` (the read root);
    * `builtin_test.go` and new `shell_test.go`.
  * `acpserver/agent.go` (`markers`) and its tests.
  * `internal/cli/agent.go` (`builtinTools`) and `main_test.go`.
  * `agent/agent_test.go`.
  * `docs/architecture.md`, this PLAN and `docs/README.md`.
* **Accept.**
  * bash, on Windows: echo, a failing status, workdir, a missing workdir, the prefix, and the markers with an inherited `GOBBLE_SESSION_ID` replaced.
  * The timeout rule: 0 is 120 s, 5 is 5 s, and 9999 is 600 s.
  * A 1 s timeout stopped `(sleep 2; echo late > marker) & sleep 30`, and the marker was never written.
  * The trap test ran in WSL: the TERM trap wrote its file before the kill.
  * A cancel ended a call in 0.2 s.
  * 3000 lines gave the last 2000 with the cut notice, and the log held all 3000.
  * The sweep removed an 8-day-old log and kept a 6-day-old one and another file.
  * read opens a file in the output directory without asking, and write still asks.
  * PowerShell gave `hi`, and `é` came through intact.
  * proctree's `Terminate` and `Kill` each ended a grandchild, on Windows and in WSL.
  * mcpclient's stdio tests passed unchanged on proctree.
  * acpserver: a session's bash saw `[gobble] [s1] [off]`.
* **Negative tests,** each on its own scratch copy:
  * `Prepare` a no-op, in WSL: `tree_test.go:90: grandchild … outlived Terminate`;
  * SIGKILL before SIGTERM, in WSL: `shell_test.go:111: the TERM trap did not run`;
  * the default timeout dropped: `commandTimeout(0) = 10m0s, want 2m0s`;
  * the markers not set: `reply "The tool said: [] [] []…" lacks "[gobble] [s1] [off]"`;
  * the spill disabled: `output starts "(output cut: … the whole output was not saved)"`;
  * the sweep's comparison inverted: `gobble-bash-old.log: kept true, want false`.
* **What the entry predicted wrongly.**
  1. **The powershell tool made three tests depend on the OS** (deviation 1).
  2. **mcpclient's two "tests" were helpers** (deviation 2).
  3. **PowerShell writes CRLF.** The model now reads command output with LF line ends; the saved file keeps the bytes as written.
  4. **A cut output with no saved file** printed an empty path. That cannot happen while the cut and the spill share one limit, but the notice now says "the whole output was not saved" rather than naming nothing. The spill negative found it.
  5. **The spill negative's expected string was wrong twice,** because the test fails at a different assertion each time. The harness was corrected, not the test.
  6. **Lint found four kinds of issue in the new code.**
     * `noctx` on the runner's `exec.Command`. The runner deliberately does not use `CommandContext`, which on cancel kills only the leader, so the line says why. The tests use `CommandContext`.
     * `goconst` for `windows`.
     * `unparam` for `readDir`'s unused `env`.
  7. **What a command leaves running ends with it.** On Unix, `Exited` sends SIGTERM to the group. On Windows, closing the Job object kills what is left. A model that starts `server &` loses it when the call returns. Background commands are X2's.
* **Gates.**
  * lint printed `0 issues.` for linux, darwin and windows.
  * Windows `go test ./...` had no `FAIL` or `panic:`.
  * `make preflight` printed `preflight passed` on Windows and in WSL. Its pre-add check covers every Go file.
  * WSL `go test -race ./...` exited 0, with `internal/proctree` and the trap test passing there.

**2026-10-08 — native tools placed (0005-MADR amendment "native tools"; owner's decision of 2026-10-08).**

The owner added eleven native tools or capabilities. [0011-REPORT](../reports/0011-REPORT-native-tools-survey.md) holds their evidence, 62 claims each checked against the sources. The MADR amendment holds each tool's contract. They are placed as follows, and each is expanded into an executable entry when its phase starts:

| Tool | Phase | Exposure |
| :--- | :--- | :--- |
| `move`, `delete`, `copy`, `mkdir` | F1c | direct |
| `tree` (replaces `ls`) | F1c | direct |
| `request_permissions` | F6 | ~~direct~~ deferred (0012-MADR D7) |
| `job_output`, `job_input`, `job_kill`; a yield window on `bash` and `powershell` | X2 | direct |
| `monitor` | X2 | deferred |
| `lsp` | X7 | deferred |
| `notebook` | X8 (new) | deferred |
| documents in `read` (PDF, DOCX, XLSX, PPTX) | X8 (new) | — |

F1c, the next sub-phase, therefore builds grep, find, the four file operations and `tree`, with the in-house gitignore matcher. 0010-REPORT §1 is corrected on the same date: Kilo's `read-extract.ts` converts DOCX and XLSX, not `.ipynb`, as 0011-REPORT's verification found.

**2026-10-08 — 0012-MADR: tool schemas, descriptions and loading.** [0012-PLAN](0012-PLAN-tool-schemas-descriptions-and-loading.md) builds D1, D2, D3 and D5 for the five built-in tools. The rest land where their tools are built, by the notes of this date at F1 steps 5 and 8, F3, F6, F8 and X4. `request_permissions` is deferred.

**2026-10-08 — F1c made executable** (the owner's decisions of 2026-10-08: two sub-phases; one in-house glob engine; the repository's own ignore files; Pi's ripgrep output form).

F1c is step 5, as the entry "native tools placed" widened it: grep, find, `tree`, move, delete, copy and mkdir, and the in-house gitignore matcher (decision 4 of the entry "F1 made executable"). It applies [0012-MADR](0012-MADR-tool-schemas-descriptions-and-loading.md) to every new tool: D1 and D2 schemas through `tool.WithSchema`, and D5 item 6 descriptions in `tool/builtin/describe/`.

**The facts,** checked on 2026-10-08 by a script in the session scratchpad (`q220_claims_f1c.py`): 15 claims, all passing. With the check inverted, all 15 failed.

- **Pi's grep:**
  - prints a match as `path:N: text` and a context line as `path-N- text` (`grep.ts:211-212`);
  - cuts a line at 500 characters (`truncate.ts:13`);
  - runs ripgrep with `--hidden` and no ignore override (`grep.ts:162`), so ripgrep's own ignore rules apply;
  - names the next limit when it hits one: "matches limit reached. Use limit=…" (`grep.ts:289`).
- **Pi's find** runs fd with `--glob --hidden` (`find.ts:182`). Without fd, it excludes `**/node_modules/**` and `**/.git/**` (`find.ts:126`).
- **Pi's ls** sorts by name (`ls.ts:109`).
- **opencode** groups matches under each file, as `Line N:` (`grep.ts:96`).
- **None of Pi, opencode or Kilo** sorts results, and none can turn ignore rules off. That is from a read-only survey of 2026-10-08; its citations C1–C9 are re-checked above.
- **0005-MADR's rows:**
  - grep names `doublestar` for globs (`:123`);
  - find includes hidden files, respects `.gitignore`, and skips `.git` and `node_modules` (`:124`).
- **A delete already always asks.**
  - acpserver's policy offers only "Allow" (allow-once) and "Reject" (`acpserver/prompt.go:201-202`), so no answer can stand for later calls.
  - A client with no prompt, such as print mode, rejects (`acpclient/conn.go:362`).
  - So 0005-MADR's "a delete always asks" holds until F6 adds rules, and F6 has to keep it.
- **read already lists one directory,** with a `/` after each directory (`tool/builtin/readdir.go:35`).
- **git's rules,** from `git-scm.com/docs/gitignore`, fetched 2026-10-08:
  - **Sources, highest precedence first:** a `.gitignore` in the path's directory or any parent up to the top of the working tree, with lower files overriding higher ones; then `$GIT_COMMON_DIR/info/exclude`; then `core.excludesFile`. "Within one level of precedence, the last matching pattern decides the outcome."
  - **Format:** blank lines and `#` comments; trailing spaces are ignored unless escaped; `!` negates. A separator at the start or middle anchors a pattern to its file's directory. A trailing `/` matches directories only. `*` and `?` never match `/`, and `[…]` is a range. `\` escapes.
  - **`**`:** leading `**/` matches in every directory, trailing `/**` everything inside, and `/**/` zero or more directories.
  - **Re-inclusion:** "It is not possible to re-include a file if a parent directory of that file is excluded."

**The owner's decisions, 2026-10-08:**

1. **Two sub-phases,** each one commit with its own record and gates:
   - **F1c-1, search:** the matcher, grep, find and `tree`;
   - **F1c-2, file operations:** move, delete, copy and mkdir.
2. **One in-house glob engine,** in `internal/fsx`. It is written to git's rules, and is also grep's and find's glob, with `{a,b}` alternatives added. 0005-MADR's `doublestar` is not required, and the grep row gains a note.
3. **The repository's own ignore files.** That is every `.gitignore` from the repository's top level down to each directory walked, and `.git/info/exclude`, in git's precedence. `core.excludesFile` is not read, so gobble reads no git configuration. `.git` and `node_modules` are always skipped, and hidden files are included.
4. **Pi's ripgrep output form** for grep, sorted by path and then by line, so the same search always gives the same output.

**F1c-1 — the matcher, grep, find and `tree`.**

Choices made within the wording:

- **`internal/fsx/glob.go`:**
  - `Glob`, compiled by `CompileGlob(pattern string) (Glob, error)` and matched by `(Glob) Match(rel string, dir bool) bool`. `rel` is slash-separated and relative.
  - The syntax is git's: `*`, `?`, `[…]` with `!` or `^` negation, `\` escapes, and the three `**` forms. On top of git's rules, `{a,b,…}` alternatives are expanded before matching; they nest, and an unclosed `{` is a literal.
  - A pattern that never matches, such as one ending in a lone `\`, compiles to a glob that matches nothing, as git says. A malformed `[` is an error, for tools to report.
- **`internal/fsx/ignore.go`:**
  - `Ignore` holds the rules of one walk. `(*Ignore) Ignored(rel string, dir bool) bool` applies them in git's precedence, with the last match winning within a file. A path under an ignored directory is ignored whatever a later `!` says.
  - **Where the rules come from.** The repository's top level is the nearest ancestor of the walk's root that holds a `.git` entry, a directory or a file (a worktree).
    - Each `.gitignore` from there down is read as the walk reaches its directory.
    - `.git/info/exclude` is read at the top level. ~~Where `.git` is a file, it is the `info/exclude` under the directory that file names.~~ *(F1c-1 deviation 1, 2026-10-08, the owner's decision: git reads `$GIT_COMMON_DIR/info/exclude`, and for a worktree the directory `.git` names is `.git/worktrees/<name>`. So where `.git` is a file, gobble reads the directory it names. If that directory holds a `commondir` file, gobble resolves it, relative to that directory, and reads `info/exclude` under the result, as git does. A test builds a fake worktree layout.)*
    - Without a repository, only the `.gitignore` files under the walk's root apply.
  - **Above the roots.** Ignore files in ancestors of a workspace root, up to the top level, are read with `os.ReadFile`, read-only. Their text is used to filter, and never reaches the model. This is the one read outside the roots that needs no permission, and the test names it.
- **`internal/fsx/walk.go`:**
  - `(Workspace) Walk(root string, visit func(rel string, d fs.DirEntry) error) error` walks under `os.Root`, sorted by name with byte order, so the walk is the same on every OS.
  - It skips `.git` and `node_modules` at every depth, and everything `Ignore` ignores. Hidden entries are included.
  - It does not follow symbolic links to directories. A link to a file is visited as a file, and reading it goes through `os.Root`, which refuses a target outside the root.
  - `visit` returning `fs.SkipDir` skips a directory. Any other error stops the walk.
- **The tools,** each confined (`tool.Confined` on its `path`) and direct. Their schemas are 0012-MADR's table, through `tool.WithSchema`, and each published bound is served within it or refused in the tool's own words (0012-MADR D2 rule 8).

  | Tool | Kind, annotations | Behaviour |
  | :--- | :--- | :--- |
  | `grep {pattern, path?, glob?, ignoreCase?, literal?, context?, limit?}` | search; read-only, idempotent | See the list below the table |
  | `find {pattern, path?, limit?}` | search; read-only, idempotent | `pattern` is a glob, matched as grep's `glob` is. The output is one path per line, relative to the search root, with a `/` after each directory, in walk order. The `limit` is 1–5000, default 1000, and a larger value is served at 5000. Output is capped at 50 KB. "No match" is a plain result: `find <path>: nothing matches <pattern>` |
  | `tree {path?, depth?, limit?}` | read; read-only, idempotent | The root's path and a `/`, then entries indented two spaces a level: directories first, then files, each group by name. A directory gets a trailing `/`, and a file its size from `stat` (`12 B`, `3.4 KB`, `1.2 MB`). See the list below the table |

  **grep in detail:**
  - **The pattern.** `pattern` is RE2 (Go's `regexp`), as 0005-MADR's report D13 notes. `literal` quotes it, and `ignoreCase` adds `(?i)`. An invalid pattern is refused with `grep: the pattern is not valid RE2 (<reason>); escape it, or set literal`.
  - **Which files.** `glob` filters by file name. A glob without a `/` matches a file's base name at any depth, and one with a `/` matches the path relative to the search root. A `path` that names a file searches that file.
  - **Skipped files:**
    - files whose first 8,000 bytes hold a NUL, as git's binary test does;
    - files over 8 MB. Their count is reported in a notice, so the model knows the search was partial.
  - **Output:**
    - a match line is `path:N: text`, and a context line `path-N- text` (Pi's form);
    - paths are relative to the search root, with `/` separators;
    - a line is cut at 500 characters, ending with `…`.
  - **Bounds:**
    - `context` is 0–10, and a larger value is served at 10;
    - `limit` is 1–1000, default 100, and a larger value is served at 1000;
    - output is capped at 50 KB.
  - **No match** is a plain result, not an error: `grep <path>: no matches for <pattern>`.

  **tree in detail:**
  - `depth` is 1–10, default 2, and `limit` is 1–2000 entries, default 500; larger values are served at their maxima.
  - **The budget is breadth-first** (grok-build, 0011 K2). Every entry of a level is counted before any entry of the next.
  - A directory left unexpanded when the budget runs out has an ellipsis, `…`, after its name.
- **The notices,** in gobble's form, `[<what>; <how to go on>]`, after a blank line:
  - `[100 matches shown, the limit; use limit=200 for more, or narrow the pattern or path]` (and the same with `paths` for find, and `entries` for tree, which says `use limit=… or a deeper path`);
  - `[output cut at 50 KB; narrow the pattern or path]`;
  - `[some lines cut at 500 characters; read the file for the whole line]`;
  - `[3 files over 8 MB were not searched]`;
  - for tree: `[N directories not expanded; use a larger limit, or tree a subdirectory]`.
- **A missing `path`** is an error naming similar entries, as read's is (`didYouMean`).
- **The descriptions,** under `tool/builtin/describe/`, exactly, each ending in one newline:
  - `grep.md`:

    ```markdown
    Search file contents for a regular expression, under a directory or in one file.

    ## Use when

    - Finding where a name, a string or a pattern appears across files.
    - Prefer it to `grep`, `rg` and `Select-String` in a shell.
    - Not for finding files by name: use find.

    ## Returns

    - One line per match, as `path:N: text`, and with `context` set, nearby lines as `path-N- text`.
    - Paths relative to the search root, sorted by path, then by line.
    - When nothing matches, a line saying so. That is not an error.

    ## Rules

    - `pattern` is RE2 syntax: no lookaround, and no backreferences. Set `literal` to search for the text as it is.
    - `glob` without a `/` matches file names at any depth, such as `*.go` or `*.{ts,tsx}`; with a `/` it matches the path from the search root.
    - Files in `.gitignore`, `.git`, `node_modules`, binary files, and files over 8 MB are skipped. Hidden files are searched.
    - At most `limit` matches and 50 KB are shown; a line over 500 characters is cut.
    ```

  - `find.md`:

    ```markdown
    Find files and directories whose names match a glob, under a directory.

    ## Use when

    - Locating files by name or extension, such as `*_test.go` or `docs/**/*.md`.
    - Prefer it to `find`, `fd`, `ls -R` and `Get-ChildItem -Recurse` in a shell.
    - Not for searching inside files: use grep.

    ## Returns

    - One path per line, relative to the search root, with a `/` after each directory, sorted by path.
    - When nothing matches, a line saying so. That is not an error.

    ## Rules

    - A glob without a `/` matches names at any depth; with a `/` it matches the path from the search root. `**` crosses directories, and `{a,b}` gives alternatives.
    - Entries in `.gitignore`, `.git` and `node_modules` are skipped. Hidden entries are listed.
    - At most `limit` paths and 50 KB are shown.
    ```

  - `tree.md`:

    ```markdown
    Show the layout of a directory as an indented tree, a few levels deep.

    ## Use when

    - Getting to know a project, or a directory, before reading or changing it.
    - Prefer it to `tree`, `ls -R` and `Get-ChildItem -Recurse` in a shell.
    - Not for one directory's entries: use read on the directory.

    ## Returns

    - The directory, then its entries indented two spaces a level: directories first, with a `/`, then files, with their sizes.
    - When the entry budget runs out, each unexpanded directory ends in an ellipsis, `…`, and a last line says how many.

    ## Rules

    - `depth` levels are shown, {{.TreeDepth}} by default. Each level is listed in full before the next is started.
    - Entries in `.gitignore`, `.git` and `node_modules` are skipped. Hidden entries are shown.
    - At most `limit` entries are shown.
    ```

  - **`read.md`'s "Not for" line** becomes `- Not for searching files for text: use grep.`, now that grep exists.
- **`describeData`** gains `TreeDepth` (the default `depth`, 2).
- **The shape test's tool set** is `ToolsWith(Options{})`, which now includes grep, find and `tree`, so their "Not for" lines are checked against real tools.

**F1c-1's files:**

| | Files |
| :--- | :--- |
| new | `internal/fsx/glob.go`, `glob_test.go`, `ignore.go`, `ignore_test.go`, `walk.go`, `walk_test.go`; `tool/builtin/grep.go`, `find.go`, `tree.go`, `search_test.go`; `tool/builtin/describe/grep.md`, `find.md`, `tree.md` |
| changed | `internal/fsx/fuzz_test.go` (adds `FuzzGlob`); `tool/builtin/builtin.go` (the three tools in `ToolsWith`); `tool/builtin/describe.go` (`TreeDepth`); `tool/builtin/describe/read.md`; `tool/builtin/builtin_test.go` (the tool list in `TestToolsAndAnnotations`: the names it expects grow by three, and grep, find and tree join read as read-only); `tool/builtin/schema_test.go` (`TestBoundsHold` gains the new bounds); `docs/architecture.md` (`internal/fsx` and `tool/builtin` rows); `README.md` (the Status list) |

*(F1c-1 deviation 2, 2026-10-08, the owner's decision.)* `tool/builtin/describe_test.go` joins F1c-1's changed files. `TestDescriptionShape` hard-coded its template set (`bash, edit, powershell, read, write`), so the three new templates broke it. The set is now derived: every tool `ToolsWith` can build, plus powershell's template where it is not built. Every tool has exactly one template, and every template a tool, and no later phase edits a list.

*(F1c-1 deviation 3, 2026-10-08, found at F1c-1's pre-add check; the owner's decisions.)*

- **What was found.** `govulncheck` fails every gate. Go 1.27.1's standard library has 10 reachable vulnerabilities, GO-2026-6603 to GO-2026-6617, in `net/http`, `net/http/internal/http2`, `net/textproto`, `crypto/tls` and `os`, all fixed in Go 1.27.2, released and stable. It is pre-existing: it reproduces on a clean export of `4504465`, which has none of F1c-1's changes.
- **The decisions:**
  - the toolchain moves to Go 1.27.2 in a commit of its own, before F1c-1's, with an amendment to 0004-MADR;
  - the agent updates both hosts' `GOTOOLCHAIN` pin to `go1.27.2`.
- **What follows.** F1c-1 is gated again on Go 1.27.2 before it is staged.

**F1c-1's tests,** each seen failing first on a scratch copy (0012-PLAN C3):

- **`glob_test.go`:**
  - every example in git's PATTERN FORMAT text: `doc/frotz/` against `doc/frotz` and `a/doc/frotz`; `frotz/` against `frotz` and `a/frotz`; `**/foo`; `**/foo/bar`; `abc/**`; `a/**/b` against `a/b`, `a/x/b` and `a/x/y/b`;
  - `*` and `?` not crossing `/`; ranges; escapes `\#`, `\!` and `\*`; a trailing `\` matching nothing;
  - braces, nested braces, and an unclosed `{`.
- **`ignore_test.go`:**
  - the last match winning; `!` re-including a file;
  - a file under an ignored directory staying ignored after `!`;
  - a lower `.gitignore` overriding a higher one;
  - `.git/info/exclude`, and a `.git` file naming another directory;
  - trailing spaces and their escape.
- **`walk_test.go`:**
  - byte-ordered names; `.git` and `node_modules` skipped; hidden entries included;
  - a directory symlink not followed (on systems where a test can make one, else skipped with the reason logged);
  - `fs.SkipDir`;
  - the one read above a root, an ancestor `.gitignore`.
- **`search_test.go`:**
  - grep's format, sorting, `context`, `ignoreCase`, `literal`, `glob` with and without `/`, a file `path`, the RE2 refusal, binary and over-size skips with their notice, no matches;
  - find's format and `/` suffix;
  - tree's breadth-first budget and its ellipsis marks, its sizes, and its order;
  - each tool's notices;
  - a missing path naming similar entries;
  - a `path` outside the roots reported by `Outside`.
- **`TestBoundsHold`:** grep `context` 50 and `limit` 5000, find `limit` 9000, and tree `depth` 50 and `limit` 9000, each served at its maximum.
- **`FuzzGlob`:** `CompileGlob` and `Match` never panic, and a pattern with no wildcard matches exactly itself.
- **The existing tests,** including 0012's convention and description tests, pass with no assertion edited (0012-PLAN C1).

**F1c-1's verification:**

```text
go test ./internal/fsx ./tool/builtin -count=1                  -> ok
go test ./internal/fsx -run '^$' -fuzz FuzzGlob -fuzztime 60s   -> ok (no failure in 60 s)
Probe A (scratch module specsize)                               -> eight tools; each description at most 1,200 bytes, each definition at most 2,000
the Stability rule's gates (0012-PLAN): pre-add check; lint for linux, darwin, windows; Windows go test; WSL preflight and race; Windows preflight
```

**F1c-2 — move, delete, copy and mkdir.**

Choices made within the wording of 0005-MADR's native-tools amendment:

- **`internal/fsx`** gains `Rename`, `Remove`, `RemoveAll`, `MkdirAll`, `Lstat` and `CopyFile` on `Workspace`, each under the `os.Root` of the root that holds the path (Go 1.25's `Root.Rename`, `Root.RemoveAll` and `Root.MkdirAll`).
- **Locks.** A move or a copy takes both paths' locks, in the order of their lock keys, so two operations can't deadlock.
- **`move {from, to, overwrite?}`.**
  - An existing `to` is refused unless `overwrite`.
  - Within one root it is `Root.Rename`. Across roots, or across devices (the rename's error is `EXDEV`), it copies, then removes the source. If the copy fails, the source is untouched and the partial copy is removed.
  - Kind `move`, destructive.
- **`delete {path, recursive?}`.**
  - A directory needs `recursive` unless it is empty.
  - A workspace root, or an ancestor of one, is refused: `delete <path>: a workspace root cannot be deleted`.
  - Kind `delete`, destructive. Every call asks (the fact above).
- **`copy {from, to, overwrite?}`.**
  - A file is written atomically through `fsx`, keeping its permission bits. A directory is copied entry by entry.
  - A symbolic link is copied as a link, with its target text unchanged and not followed.
  - An existing `to` is refused unless `overwrite`.
  - Kind `edit`, destructive.
- **`mkdir {path}`** creates the directory and its parents. An existing directory is success, and an existing file is refused. Kind `edit`.
- **Confinement.** Both paths of move and copy are reported by `Outside`, so either outside the roots needs approval.
- **Results,** in gobble's form:
  - `moved <from> to <to>`;
  - `deleted <path>` or `deleted <path> (N entries)`;
  - `copied <from> to <to>` or `copied <from> to <to> (N entries)`;
  - `created <path>` or `<path> already exists`.

  Each result's `Summary` is the same line.
- **Descriptions,** under `tool/builtin/describe/`, each ending in one newline:
  - `move.md`:

    ```markdown
    Move or rename a file or a directory.

    ## Use when

    - Renaming, or moving something to another directory, including across the workspace's directories.
    - Prefer it to `mv`, `Move-Item` and `ren` in a shell.
    - Not for keeping the original as well: use copy.

    ## Returns

    - A line such as `moved old.go to new.go`.

    ## Rules

    - An existing `to` is refused unless `overwrite` is set.
    - Across directories that are not on one device, it copies and then deletes, and a failed copy leaves the original as it was.
    ```

  - `delete.md`:

    ```markdown
    Delete a file, or a directory with everything in it.

    ## Use when

    - Removing files or directories that are no longer needed.
    - Prefer it to `rm`, `rmdir`, `Remove-Item` and `del` in a shell.
    - Not for emptying a file while keeping it: use write with empty `content`.

    ## Returns

    - A line such as `deleted build/ (42 entries)`.

    ## Rules

    - Every delete asks the user first.
    - A directory that is not empty needs `recursive`.
    - A workspace root is never deleted.
    ```

  - `copy.md`:

    ```markdown
    Copy a file, or a directory with everything in it.

    ## Use when

    - Duplicating a file or a directory, keeping the original.
    - Prefer it to `cp`, `Copy-Item` and `copy` in a shell.
    - Not for renaming or relocating: use move.

    ## Returns

    - A line such as `copied a.txt to b.txt`, with the entry count for a directory.

    ## Rules

    - An existing `to` is refused unless `overwrite` is set.
    - Permission bits are kept. A symbolic link is copied as a link, and not followed.
    ```

  - `mkdir.md`:

    ```markdown
    Create a directory, with any missing parents.

    ## Use when

    - Preparing a directory before files are written into it, though write creates missing parents itself.
    - Prefer it to `mkdir -p`, `New-Item -ItemType Directory` and `md` in a shell.
    - Not for creating a file: use write.

    ## Returns

    - A line such as `created docs/guides`, or that it already exists.

    ## Rules

    - A directory that already exists is success; an existing file of that name is refused.
    ```

**F1c-2's files:**

| | Files |
| :--- | :--- |
| new | `internal/fsx/ops.go`, `ops_test.go`; `tool/builtin/fileops.go`, `fileops_test.go`; `tool/builtin/describe/move.md`, `delete.md`, `copy.md`, `mkdir.md` |
| changed | `tool/builtin/builtin.go`; `tool/builtin/builtin_test.go` (the tool list); `docs/architecture.md`; `README.md` (the Status list) |

*(F1c-2 deviations, 2026-10-08, found while planning its code; the owner's decisions.)*

1. **mkdir's MCP hint.** `TestToolsAndAnnotations` asserted that every tool that is not read-only is destructive. mkdir only adds, and MCP defines `destructiveHint: false` as "only additive updates". So mkdir sets `DestructiveHint` false, and the test's rule becomes three-way: read, grep, find and tree are read-only; mkdir is additive; every other tool is destructive.
2. **The approval test's file.** `acpserver/fileops_test.go` joins F1c-2's new files. It drives a session through `acptest`'s client, calls delete, checks that a `session/request_permission` arrives, rejects it, and checks that the file is still there.
3. **A race in Go's encoding/json** *(found at F1c-2's gates)*.
   - **What was found.** The WSL `go test -race` run failed once, in `internal/cli`'s `TestSessionIDAndName`. The race is inside the standard library's `jsontext` encoder pool, where one `Marshal` runs inside another through a `MarshalJSON` method, as jsonschema-go's `Schema.MarshalJSON` does.
   - **It is pre-existing and outside gobble.** It fails about once in 200 runs on a clean `c4ca6f5`, and a reproducer with only the standard library races on Go 1.27.2 and 1.27.1. The evidence is in [0013-REPORT](../reports/0013-REPORT-go-json-nested-marshal-race.md).
   - **The owner's decision:** record it, report it upstream, and keep the race gate as it is. F1c-2's gate record says that its WSL race run failed on this defect.

**F1c-2's tests,** each seen failing first on a scratch copy:

- **move:** within a root; across roots; across devices, through a test seam on the rename that returns `EXDEV`; a failed cross-device copy leaving the source; `overwrite`.
- **delete:** a file; an empty directory; a non-empty directory with and without `recursive`; the root refusal; an ancestor of a root refused.
- **copy:** a file's mode kept; a directory tree; a symlink copied as a link (where a test can make one); `overwrite`.
- **mkdir:** a new path, an existing directory, and an existing file.
- **Two moves in opposite directions** run at once, and both finish: the lock order holds.
- **`Outside`** reports both paths of move and copy.
- **The agent asks:** through `acptest`, a delete call produces a `session/request_permission`, and with the reply "Reject" nothing is deleted.

**F1c-2's verification** is F1c-1's, plus `go test ./acpserver -run 'TestDelete' -count=1 -> ok`.

**Deferred, named:**

- ACP `locations` for these tools: F1d.
- The journal entries undo needs: X2.
- `tool_search`'s families, and `request_permissions`: F1d and F6.
- Reading `core.excludesFile`: not chosen (decision 3). It would need gobble to read git configuration.

**F1c-1, 2026-10-08 — complete (staged; the owner commits).** It ran as the entry "F1c made executable" wrote it, approved by the owner's "proceed" of 2026-10-08, with deviations 1–3 decided by the owner. F1c-2 has not run.

- **The deviations,** each recorded in the entry before it was fixed:
  1. A worktree's `info/exclude` is read through `commondir`, as git reads it.
  2. `TestDescriptionShape`'s template set is derived from the tools, not hard-coded.
  3. The toolchain moved to Go 1.27.2, in a commit of its own (`c4ca6f5`), with 0004-MADR's amendment.
     - **Why:** `govulncheck` had begun reporting 10 reachable standard-library vulnerabilities in Go 1.27.1. They reproduced on a clean export of `4504465`.
     - **The machines:** both hosts' `GOTOOLCHAIN` pin was updated, with the owner's approval.
     - **One slip:** this deviation's text was written into this PLAN after the records commit `6ef4ca9` had been staged for the last time, so it reaches the repository with F1c-1's commit instead.
- **What was built:**
  - **`internal/fsx`:** `glob.go`, `ignore.go` and `walk.go`, with their tests.
  - **`tool/builtin`:** `grep.go`, which also holds the helpers the three tools share, as the entry's file list has no other place for them; `find.go`; `tree.go`; `search_test.go`; and the three templates.
  - **Updated:** `builtin.go`, `describe.go`, `read.md`, `builtin_test.go`, `describe_test.go`, `schema_test.go` and `fuzz_test.go`; `docs/architecture.md` and `README.md`.
- **The existing tests.** Two expectations changed, both named in advance: the tool list in `TestToolsAndAnnotations` (the entry), and the template set (deviation 2). No other assertion was edited.
- **C3: every new check, seen failing on a scratch copy.** There were 14 mutations; each was asserted to land exactly once, and each log was read whole.
  - **The glob engine:**
    - a trailing `**` matching its own directory: `"abc/**" matching "abc" (dir true) = true, want false`;
    - git's `[!…]` left unrewritten: the `file[!0-9].txt` cases fail.
  - **The ignore rules:**
    - re-inclusion under an ignored directory: `no re-include under an ignored directory`;
    - the depth order removed: `Ignored("a.gen") = false, want true`;
    - `commondir` not read: `a worktree's exclude file is …`.
  - **The walker:**
    - `node_modules` walked;
    - the `.gitignore` above the root not read, so `a.log` is listed.
  - **grep:**
    - context lines repeated: `f.txt-4- 4` twice;
    - `:` for context lines;
    - `limit` unbounded: `want it served at 1000`.
  - **tree and find:**
    - tree's sort removed: `README.md` before `a/`;
    - find's `/` dropped: `cmd/tool`.
  - **The template set:** a missing template: `want one for each tool`.
  - **`FuzzGlob`:** a broken `Match`: `does not match itself`.
- **Fuzzing.** `FuzzGlob` ran 40,361,510 executions in 60 s with no failure.
- **The gates,** on Go 1.27.2, on the tree this commit lands on:
  - lint printed `0 issues.` for linux, darwin and windows;
  - Windows `go test ./...` had no `FAIL` or `panic:`;
  - WSL `make preflight` printed `preflight passed`, and WSL `go test -race ./...` exited 0;
  - Windows `make preflight` printed `preflight passed`.
- **Probe A** (Messages-shape definition bytes, on Windows):

  | Tool | Description | Definition |
  | :--- | ---: | ---: |
  | read | 846 | 1,352 |
  | grep | 975 | 1,910 |
  | find | 772 | 1,345 |
  | tree | 789 | 1,296 |
  | **all eight** | | **11,081** |

  Every description is within 1,200 bytes, and every definition within 2,000.

**What the entry predicted wrongly:**

1. **The worktree exclude file,** deviation 1.
2. **The hard-coded template set,** deviation 2.
3. **The toolchain,** deviation 3. Nothing in the entry could have foreseen it: the vulnerability database changed between F1c-1's start and its pre-add check.
4. **grep's definition budget.** The first draft of grep's parameter descriptions made its definition 2,072 bytes, over the 2,000-byte budget. They were shortened, since the entry fixed the template's text but not theirs, and the definition is 1,910.
5. **Lint.** The first full gate run failed golangci-lint with ten issues in the new code, the same on every GOOS:
   - errcheck 2: an unchecked `path.Match` result, and an unchecked type assertion;
   - nilerr 2: unreadable files skipped by returning nil with an error in scope, now in a helper, `candidate`, that has no error to return;
   - gocritic 4: three needless wrapper functions, and one condition;
   - modernize 1: `errors.AsType`;
   - revive 1: `walk` beside `Walk`, now `walkDir`.

   All ten were fixed, the 14 negatives were run again on the changed code (one anchor moved with the `path.Match` fix), and the gates were run again.
6. **The RE2 refusal's wording.** `the pattern is not valid RE2 (<reason>)` doubles the parenthesis when the reason ends in one, as `missing closing )` does. It is kept as the entry wrote it.
7. **The Write tool turned a byte order mark's Go escape into the character itself** in `ignore.go`, which `go build` refused. A script put the escape back.

**F1c-2, 2026-10-08 — complete (staged; the owner commits).** It ran as the entry "F1c made executable" wrote it, approved by the owner's "proceed" of 2026-10-08, with its deviations decided by the owner: 1 and 2 before its code was written, 3 at its gates. F1c is complete.

- **The deviations** (recorded under F1c-2's tests above):
  1. mkdir's MCP hint is additive, not destructive, and `TestToolsAndAnnotations`' rule is three-way.
  2. `acpserver/fileops_test.go` joins the files, for the end-to-end approval test.
  3. A race in Go's `encoding/json`, outside gobble: recorded in 0013-REPORT and reported upstream, with the race gate unchanged.
- **What was built:**
  - **`internal/fsx/ops.go`:** `Lstat`, `MkdirAll`, `Remove`, `RemoveAll`, `Rename` (with `ErrCrossDevice` for different roots and for `EXDEV`), `CopyFile` (keeping permission bits), `CopyLink`, `IsRoot`, `LockPair` and `Inside`, each through the root that holds the path.
  - **`tool/builtin/fileops.go`:** move, delete, copy and mkdir, each with a schema, a description template, and confinement of both paths for move and copy.
  - **Updated:** `builtin.go` (`ToolsWith`, in 0012-MADR D7's order), `builtin_test.go`, `docs/architecture.md` and `README.md`.
- **One addition of mine,** within the entry's wording: `moveCopy`, a package variable that only tests set. A builtin test cannot reach `fsx`'s rename seam, and it needs a cross-device copy that fails part way. It follows `edit.go`'s `beforeWrite`.
- **The approval test.** `TestDeleteAsks` drives a session through `acptest`'s client:
  - refused, the delete raises one `session/request_permission`, and the file stays;
  - allowed, the file is deleted.
- **C3: every new check, seen failing on a scratch copy.** There were 14 mutations, each asserted to land exactly once.
  - **move:**
    - no cross-device fallback;
    - a partial copy left behind;
    - `overwrite` ignored.
  - **delete:**
    - the root deletable;
    - `recursive` ignored;
    - delete marked read-only, so `TestDeleteAsks` saw `0 permission requests, want one`.
  - **copy:**
    - a link followed;
    - permission bits not kept. This one was run in WSL, since Windows has no permission bits.
  - **mkdir:** a file overwritten.
  - **The fsx operations:**
    - `LockPair` without its key order, which deadlocked;
    - `EXDEV` not mapped to `ErrCrossDevice`;
    - `IsRoot` without ancestors.
  - **Confinement:** move's and copy's `Outside` reporting one path.
  - **The annotations:** mkdir marked destructive.
- **Lint.** golangci-lint first reported two modernize findings, both in the new tests: `WaitGroup.Go` in place of `Add` and `Done`. Both were fixed.
- **The gates:** on Go 1.27.2:
  - lint printed `0 issues.` for linux, darwin and windows;
  - Windows `go test ./...` had no `FAIL` or `panic:`;
  - Windows and WSL `make preflight` each printed `preflight passed`.
  - **WSL `go test -race ./...` failed in its first run.** It failed once, in `internal/cli`'s `TestSessionIDAndName`, on the defect in Go's `encoding/json` that [0013-REPORT](../reports/0013-REPORT-go-json-nested-marshal-race.md) records (deviation 3). It reproduces on a clean `c4ca6f5`.
  - **A second full run exited 0,** with no race reported.
  - The race gate was not changed.
- **Probe A** (Messages-shape bytes, on Windows):

  | Tool | Description | Definition |
  | :--- | ---: | ---: |
  | move | 506 | 994 |
  | delete | 463 | 867 |
  | copy | 467 | 948 |
  | mkdir | 458 | 754 |
  | **all twelve** | | **14,644** |

**What the entry predicted wrongly:**

1. **mkdir's hint,** deviation 1: the existing test's rule did not fit an additive tool.
2. **The approval test's file,** deviation 2.
3. **The cross-device test seam.** The entry put it on the rename in `fsx`. The builtin move test also needs a copy that fails part way, so a second seam, `moveCopy`, was added in `tool/builtin`.
4. **The race gate.** The entry expected F1c-2's gates to pass. One WSL race run met a defect in the standard library, deviation 3, which predates F1c; the evidence and the upstream report are 0013-REPORT.

**2026-10-08 — F1d made executable** (the owner's decisions of 2026-10-08: three sub-phases; `tool_search` built now and dormant; found tools callable from the next model call).

F1d is step 7, the ACP `fs/*` and `terminal/*` routing and tool-call `locations`, and step 8, `todo`, `tool_search` and the MCP resource tools. With them come [0012-MADR](0012-MADR-tool-schemas-descriptions-and-loading.md)'s D7–D9, D11 and D14. Every new tool applies 0012-MADR D1, D2 and D5 item 6.

**The facts,** checked on 2026-10-08 by a script in the session scratchpad (`q239_claims_f1d.py`): 18 claims, all passing. With the check inverted, all 18 failed.

- **gobble's agent fixes its tools when it is built.**
  - The agent marshals its specs once (`agent/agent.go:78`), and every request sends them (`:238`).
  - acpserver builds a new agent for every prompt, with that prompt's tools (`acpserver/agent.go:368`).
  - So a tool list can change between prompts, never within one.
- **acpserver ignores the client's capabilities.** `Initialize` takes its request unnamed (`acpserver/agent.go:204`), so `fs` and `terminal` support is never stored.
- **Session entries.**
  - An entry's unknown members round-trip (`session/session.go:53`).
  - Pi writes extension state as `{"type":"custom", …, "customType", "data"}` (Pi `session-manager.ts:120-130`; gobble's round-trip fixture `session/session_test.go:26`).
  - There is no `custom` entry type in gobble yet.
- **acpserver already sends a plan update,** from plan mode's exit (`acpserver/modes.go:146`). The ACP SDK's plan statuses are pending, in_progress and completed (`types_gen.go:3399`).
- **The ACP SDK** has `ToolCallLocation` (`types_gen.go:6541`), and agent-side `ReadTextFile` (`agent_gen.go:478`) and `CreateTerminal` (`:501`).
- **mcpclient** lists a server's tools when it connects (`mcpclient/conn.go:138`), and never lists or reads resources: none of its 11 files does.
- **The other harnesses:**
  - Pi's `tool_search` sets BM25's `k1` (`tool-search/tool.ts:119`), and is off until something is deferred (`index.ts:14`).
  - Pi's found tools are "available from your next call" (`tool.ts:263`); codex's are exposed for "the next model call" (`tool_search_spec.rs:94`).
  - codex's `update_plan` answers "Plan updated" (`plan.rs:22`), and opencode's todo tool is `todowrite` (`todo.ts:3`). Both replace the whole list on every call: a read-only survey of 2026-10-08, whose citations above are re-checked.

**The owner's decisions, 2026-10-08:**

1. **Three sub-phases,** each one commit with its own record and gates:
   - **F1d-1:** `todo`, the `custom` entry type, and `/todos`;
   - **F1d-2:** `tool_search`, 0012-MADR's tiers, families and loading, and the MCP resource tools;
   - **F1d-3:** the client's capabilities, ACP `fs/*` and `terminal/*` routing, and `locations`.
2. **`tool_search` is built in F1d-2, dormant.** It is offered only while a tool is deferred (0005-MADR, `:533`), and nothing is deferred in 1.0 until F8 defers MCP tools and F6 adds `request_permissions`. It is tested with fixture tools.
3. **Found tools are callable from the next model call,** in the same turn. The agent's tool list may grow within a turn, append-only.

**F1d-1 — `todo`, the `custom` entry, and `/todos`.**

Choices made within the wording:

- **`session`** gains `TypeCustom = "custom"`, and `Entry` gains `CustomType string` and `Data jsontext.Value`, in Pi's shape. An unknown `customType` still round-trips.
- **`tool.Result`** gains `Plan []PlanItem`, with `PlanItem{Content, Status, Priority string}`. It is client data, like `Diffs` (0012-MADR D14), so the tool stays pure: it never reaches the session or the connection.
- **`todo {todos}`** (`tool/builtin/todo.go`):
  - **The list:** an array of `{content` req `minLength` 1, `status` req enum `pending`, `in_progress`, `completed`; `priority` enum `high`, `medium`, `low`, default `medium``}` (0012-MADR's table).
  - **Each call replaces the whole list,** as codex and opencode do. An empty array clears it.
  - **The result:** `todo list updated: 2 completed, 1 in progress, 3 pending`, or `todo list cleared`, with `Result.Plan` set.
  - **Kind and hints:** kind `think`; read-only and idempotent, since it changes no file and the user is never asked.
- **acpserver, on a tool result with `Plan`:**
  - it sends `session/update` `plan`, with the entries mapped one to one;
  - ~~it appends `custom{customType: "gobble.todo", data: {todos}}`.~~ *(Deviation 2: `prompt.go` only sends updates; entries are written by `writer.event` in `acpserver/agent.go`.)*
  - `writer.event` in `acpserver/agent.go` appends `custom{customType: "gobble.todo", data: {todos}}` right after the tool's result entry.
- **Replay.**
  - `session/load` sends the last `gobble.todo` entry's plan after the history.
  - `resume` restores nothing to the client.
  - Fork and clone copy the entry, as they copy every entry.
- **`/todos`** prints the current list as text: `[x]`, `[>]` or `[ ]`, then the content, one per line; or `no todo list`.
- **`todo.md`,** in `tool/builtin/describe/`, exactly:

  ```markdown
  Keep a short checklist of the steps of the current task, shown to the user as a plan.

  ## Use when

  - A task has three or more steps, or the user gave a list of things to do.
  - Update it as steps start and finish, so the user can follow progress.
  - Not for notes or results: answer in text, or use write for a file.

  ## Returns

  - A line counting the items, such as `todo list updated: 2 completed, 1 in progress, 3 pending`.

  ## Rules

  - Each call replaces the whole list: send every item, with its status.
  - Keep one item `in_progress` at a time, and mark it `completed` when it is done.
  - An empty `todos` clears the list.
  ```

  *(Deviation 4, 2026-10-08: the template is kept above as planned, but its third Use when bullet is written as `- Not for notes or results: use write for a file, or answer in text.`, because the description test reads a Not for line only when `use` and a tool's name follow the colon.)*

**F1d-1's files:**

| | Files |
| :--- | :--- |
| new | `tool/builtin/todo.go`, `todo_test.go`, `describe/todo.md`; `acpserver/todo_test.go` |
| changed | `session/session.go`, `session/session_test.go`; `tool/tool.go` (`Result.Plan`), `tool/tool_test.go`; `tool/builtin/builtin.go`, `builtin_test.go`; `acpserver/prompt.go` (the plan update and the entry), `acpserver/history.go` (replay), `acpserver/commands.go` (`/todos`); `docs/architecture.md`; `README.md` |

*(2026-10-08, F1d-1 deviations 2 and 3: `acpserver/agent.go` (the entry) and `acpserver/commands_test.go` (`todos` in its list of command names) join the changed files. `acpserver/prompt.go` carries the plan update only.)*

*(2026-10-08, F1d-1 deviation 5: `acpserver/testdata/session.golden` (the advertised commands) joins the changed files.)*

*(2026-10-09, F1d-1 deviation 6: `docs/reports/0013-REPORT-go-json-nested-marshal-race.md` (an amendment) and `docs/README.md` (its index rows) join the changed files.)*

**F1d-1's tests,** each seen failing first on a scratch copy:

- **`custom` entries:** written and read back, and an unknown `customType` still round-trips.
- **`todo`:**
  - the result line;
  - `Plan` set;
  - an empty list clears;
  - ~~a bad status is refused by the schema's enum, which the validator enforces, as an enum is structural.~~ *(Deviation 1: the validator checks only the type-derived schema.)*
  - a bad `status` or `priority` is refused by `todo`'s own code, in its own words (0012-MADR D2 rule 8).
- **Through `acptest`:**
  - a `todo` call produces one `session/update` `plan`, with the entries in order and their statuses and priorities;
  - the session file holds a `gobble.todo` entry;
  - `session/load` sends that plan again after the history;
  - `/todos` prints it.

**F1d-1's deviations, 2026-10-08,** found while reading its files before its code was written (1–3), while writing `todo` (4), at its first acpserver test run (5) and at its gates (6), and decided by the owner:

1. **The enum is not validated.**
   - **Found:** the test line said the validator enforces the `status` enum. It does not. `Run` validates against the schema derived from the Go type alone (`tool/tool.go:296-334`), and `WithSchema`'s enums are published only (`tool/tool.go:191-195`). jsonschema-go v0.4.3's `jsonschema` tag sets only a description (`infer.go:329-336`).
   - **Decided:** `todo`'s own code refuses a bad `status` or `priority` in its own words, as 0012-MADR D2 rule 8 says. The test is in `todo_test.go`. No file is added.
2. **Entries are written in `acpserver/agent.go`.**
   - **Found:** `acpserver/prompt.go` sends updates only. Session entries are written by `writer.event` (`acpserver/agent.go:479-499`).
   - **Decided:** `acpserver/agent.go` joins F1d-1's files. Its `ToolEnd` case appends the `gobble.todo` entry right after the tool's result entry.
3. **The command names are pinned.**
   - **Found:** `acpserver/commands_test.go:76` lists every command name, so `/todos` fails it.
   - **Decided:** `acpserver/commands_test.go` joins F1d-1's files, with `todos` added to that list.
4. **The template's Not for line.**
   - **Found:** `todo.md`'s planned bullet, `- Not for notes or results: answer in text, or use write for a file.`, fails the description test, whose Not for pattern needs `use` and a tool's name right after the colon (`tool/builtin/describe_test.go:21`).
   - **Decided:** the bullet is `- Not for notes or results: use write for a file, or answer in text.` The meaning is unchanged, and the test is unchanged. No file is added.
5. **The golden transcript pins the advertised commands.**
   - **Found:** with `/todos` added, `TestSession` failed at line 4 of `acpserver/testdata/session.golden`, whose `available_commands_update` lists every command. The one difference was the new `todos` entry; the file was unchanged in git, and no other fixture holds the list.
   - **Decided:** `acpserver/testdata/session.golden` joins F1d-1's files, regenerated with `ACPTEST_UPDATE=1 go test ./acpserver -run TestSession`, its diff checked to be that one entry.
6. **The race gate met 0013-REPORT's defect again, with a new signature.**
   - **Found:** WSL `go test -race ./...` failed once, in `internal/cli`'s `TestSessionValues`, in jsonschema-go's `orderedProperties.MarshalJSON` under `tool.New`'s `Marshal` of `tree`'s schema. No F1d-1 code is in the stack. The pair, `bytes.growSlice` against `bytes.(*Buffer).WriteByte`, was not the one 0013-REPORT records. A reproducer with no gobble code gave eight kinds of pair, in bursts. 15 runs of `internal/cli` without F1d-1, and 15 with it, gave no race, which at this rate cannot tell the trees apart. The evidence is in 0013-REPORT's amendment of 2026-10-09.
   - **Decided (2026-10-09):** F1d-1 lands with the failure recorded. 0013-REPORT is amended, correcting its claim that the pair never varies, and joins F1d-1's files with `docs/README.md`, whose index called the defect reported upstream. gobble's one nested marshal, in `tool.New`, is to be removed as 0012-PLAN P9, under a 0012-MADR amendment written and approved first. The race gate is unchanged.

A choice within the wording: `todos` publishes `minItems: 0`, because the schema test requires every array to state `minItems` (`tool/builtin/schema_test.go:72`) and an empty list clears.

**F1d-2 — `tool_search`, the tiers, and the MCP resource tools.**

Choices made within the wording:

- **`tool/toolsearch`** is the index (0012-MADR D11):
  - `New(specs []tool.Spec) *Index`, and `(*Index).Search(query string, limit int) []tool.Spec`.
  - **The fields and weights:** the name split on `_`, weight 3; the description's first sentence, weight 2; the rest of the description, 1; the schema's property names and descriptions, 1.
  - **Tokens and BM25:** lower-cased words on Unicode letter and digit boundaries, with an English stop-word list; `k1` 1.2 and `b` 0.75, as Pi's.
  - **An exact name** ranks first.
- **`tool`:**
  - `Spec` gains `Family string`, set by `WithFamily`;
  - `Result` gains `Load []string`, the names of tools to load.
- **`agent`** partitions its tools by `Spec.Exposure`:
  - **direct** tools are sent;
  - **deferred** tools are kept for loading;
  - **hidden** tools are neither sent nor found.
- **When a result carries `Load`,** the agent:
  - appends those tools, with their families, after the tools already sent, and re-marshals the specs before the next request in the same turn;
  - emits an event saying what it loaded.
- **acpserver records each load** as `custom{customType: "gobble.tools", data: {loaded}}`. At the next prompt it builds the agent with every tool the session's entries have loaded as direct, in load order: 0012-MADR D9's start-of-session loads.
- **`tool_search {query, limit?}`** (`tool/builtin/toolsearch.go`):
  - **The arguments:** `query` req `minLength` 1; `limit` 1–20, default 5.
  - **What it searches:** the agent gives it the deferred tools through `tool.Env.Deferred`, a new `[]tool.Spec`.
  - **The result:** `loaded 2 tools, callable from your next call:`, then one line per tool, `- name: first sentence`, with `Load` set. When nothing matches, `no deferred tool matches <query>`.
  - **When it is offered:** the agent adds it as a direct tool only when something is deferred.
- **The MCP resource tools** ~~(`mcpclient/resources.go`)~~ *(Deviation 2, 2026-10-09: built in `tool/builtin/resources.go` from a `ResourceSource` that `mcpclient` implements in `mcpclient/resources.go`, so D2's and D5's tests walk them.)*:
  - **mcpclient learns resources:** `resources/list`, `resources/templates/list` and `resources/read`, through the go-sdk's client session.
  - **The tools:** `list_mcp_resources {server?, cursor?}`, `list_mcp_resource_templates {server?, cursor?}` and `read_mcp_resource {server, uri}`, in family `mcp-resources`.
  - **They exist only in a session with a connected server that advertises resources,** and are loaded at the start of its first prompt. So they are never left deferred, and `tool_search` stays dormant.
- **Descriptions:** `tool_search.md` and the three resource tools' templates, in D5 item 6's shape. Their text is written with the code and reviewed in the commit, because their Returns lines depend on the go-sdk's result types, read when F1d-2 starts.

**F1d-2's files:**

| | Files |
| :--- | :--- |
| new | `tool/toolsearch/index.go`, `index_test.go`; `tool/builtin/toolsearch.go`, `toolsearch_test.go`, `describe/tool_search.md`; `mcpclient/resources.go`, `resources_test.go`, with templates for the three tools |
| changed | `tool/tool.go` (`Family`, `Result.Load`, `Env.Deferred`), `tool/tool_test.go`; `agent/agent.go`, `agent/agent_test.go`; `acpserver/agent.go`, `acpserver/mcp.go`, `acpserver/prompt.go`; `mcpclient/manager.go`, `mcpclient/conn.go`; `mcpclient/mcptest/` (a fixture server with resources); `docs/architecture.md` |

*(2026-10-09, F1d-2 deviations 1–3: `agent/event.go` and `tool/toolsearch/doc.go` join the changed files; `tool/builtin/resources.go`, `resources_test.go` and the three templates under `tool/builtin/describe/` join the new files, with `tool/builtin/builtin_test.go`, `describe_test.go` and `schema_test.go` changed; `mcpclient/resources.go` implements the source and has no templates; `internal/cli/agent.go` and its test join the changed files.)*

*(2026-10-09, F1d-2 deviation 5: `acpserver/toolload_test.go` joins the new files. Deviation 6: `README.md` joins the changed files.)*

**F1d-2's tests,** each seen failing first on a scratch copy:

- **The index:** the field weights; an exact name first; stop words dropped; `k1` and `b` changing scores as BM25 says.
- **The agent:**
  - deferred tools are not sent;
  - a `Load` adds a tool to the next request of the same turn, after the tools already there;
  - a family loads whole;
  - hidden tools are never offered;
  - `tool_search` is offered only while something is deferred.
- **acpserver:**
  - a load is recorded;
  - the next prompt starts with the loaded tool;
  - fork keeps it.
- **mcpclient,** against `mcptest`'s fixture server: the three resource methods; the tools appear only when the server advertises resources.

**F1d-2's deviations, 2026-10-09,** found by reading its files before any code was written (1–4) and while writing its code (5, 6), and decided by the owner:

1. **Two files the steps need.**
   - **Found:** the agent's events are declared in `agent/event.go`, not `agent/agent.go`. `tool/toolsearch/doc.go` says the package "builds no index".
   - **Decided:** both join F1d-2's files. The load event is declared beside the others, and the package documentation describes the index.
2. **The resource tools' D2 and D5 checks.**
   - **Found:** `TestSchemaConvention` and `TestDescriptionShape` walk only `tool/builtin`'s tools, and the template renderer is that package's own (`tool/builtin/describe.go`). Built in `mcpclient`, the three tools would have no check, though every new tool applies D1, D2 and D5 item 6.
   - **Decided:** `tool/builtin` builds the three tools from a small `ResourceSource` interface, which `mcpclient`'s manager implements. Their templates are under `tool/builtin/describe/`. The existing tests walk them, and nothing is duplicated.
3. **The CLI's tool flags.**
   - **Found:** `internal/cli/agent.go`'s `toolSet` accepts only built-in names and names starting `mcp__`. So `-t` and `--exclude-tools` refuse `list_mcp_resources` and the other two as unknown tools.
   - **Decided:** the CLI accepts the three names. They are filtered as MCP tools are, and listing one keeps the MCP servers on. `internal/cli/agent.go` and its test join the files.
4. **The race gate.**
   - **Found:** WSL's race detector on this host reports races that cannot happen, while the same probe is clean natively on Windows and on other machines (0013-REPORT, third amendment).
   - **Decided:** from F1d-2 on, while that holds, the race step runs in two places:
     - **for Windows,** `CGO_ENABLED=1 go test -race ./...` natively on this host, with MinGW's gcc;
     - **for Linux,** the same command over SSH on a second amd64 Linux machine, on a copy of the tree made from a git bundle.
   - WSL keeps `make preflight`, without its race run.
5. **The acpserver tests' file.**
   - **Found:** the steps require acpserver tests (a load recorded, the next prompt starting with it, fork keeping it), and the files table names no acpserver test file.
   - **Decided:** a new `acpserver/toolload_test.go` holds them, with a test of the resource family loading at the first prompt.
6. **README.md.**
   - **Found:** its status line says `tool_search` and the MCP resource tools are not built yet, and the files table leaves it out.
   - **Decided:** `README.md` joins the files. Its tools line gains them, and its status line keeps F1d-3's work only.

**F1d-3 — the client's capabilities, `fs/*` and `terminal/*` routing, and `locations`.**

Choices made within the wording:

- **acpserver stores** the client's `fs.readTextFile`, `fs.writeTextFile` and `terminal` capabilities from `initialize`.
- **`tool.Env` gains `Files`,** an interface with `ReadTextFile(path) (string, error)` and `WriteTextFile(path, content) error`.
  - acpserver sets it on a call when the client advertises the capability.
  - read, write and edit then do their text I/O through it.
  - **Confinement is unchanged:** every path is still resolved and checked by `fsx` before the client is asked.
  - Binary files, images and directories stay local.
- **`tool.Env` gains `Terminal`,** an interface over `CreateTerminal`, `TerminalOutput`, `WaitForTerminalExit`, `KillTerminal` and `ReleaseTerminal`.
  - bash uses it when the client advertises `terminal` and `builtin.Options.UseClientTerminal` is set. That setting is a Go option, by decision 2 of the entry "F1 made executable".
  - The tool call carries the terminal's reference, so the client shows it live.
  - Timeouts and cancellation still stop it, through `KillTerminal`.
- **`locations`:**
  - `tool.Result` gains `Locations []Location{Path, Line}`, and a tool may name its locations before it runs, through `tool.Locator`, as `Describer` names its title.
  - read, write, edit, move, delete, copy and mkdir name their paths. grep names its first 20 matches' files and lines.
  - acpserver puts them on `tool_call` and `tool_call_update`.

**F1d-3's files:**

| | Files |
| :--- | :--- |
| new | `acpserver/clientfs.go`, `acpserver/clientterm.go`, `acpserver/routing_test.go` |
| changed | `tool/tool.go` (`Env.Files`, `Env.Terminal`, `Result.Locations`, `Locator`), `tool/tool_test.go`; `tool/builtin/read.go`, `write.go`, `edit.go`, `shell.go`, `grep.go`, `fileops.go`, `builtin.go` (`UseClientTerminal`), and their tests; `acpserver/agent.go`, `acpserver/prompt.go`; `acpclient/acptest/client.go` (an optional file and terminal fake for tests); `docs/architecture.md` |

**F1d-3's tests,** each seen failing first on a scratch copy:

- **The file methods:**
  - a read through an `acptest` client that advertises `fs.readTextFile` produces `fs/read_text_file` and no direct disk read (0005-PLAN F1's own acceptance line);
  - a write produces `fs/write_text_file`;
  - a path outside the roots is still refused before the client is asked.
- **The terminal methods:**
  - bash with `UseClientTerminal` and a terminal-capable client produces `terminal/create`, `terminal/wait_for_exit`, `terminal/output` and `terminal/release`;
  - a timeout produces `terminal/kill`.
- **Locations** on `tool_call` and `tool_call_update` for read, edit and grep.

**Deferred, named:**

- **`request_permissions`'s event load:** F6.
- **MCP tools deferred by default:** F8. `tool_search` becomes live with them.
- **A model-visible `cancelled` todo status:** ACP has no such status, so it is left out.

**F1d-1, 2026-10-09 — complete.** Its code, tests and the `README.md` and `docs/architecture.md` lines were committed by the owner as `8c958e8`. This record, deviation 6, 0013-REPORT's amendment and the index follow in a records commit. It ran as the entry "F1d made executable" wrote it, approved by the owner's "Proceed to execute the plan" of 2026-10-08, with six deviations decided by the owner: 1 to 5 before the code each changed was written, 6 at its gates.

- **The deviations** (recorded under F1d-1's tests above):
  1. The status and priority enums are enforced by `todo`'s own code, not the validator, which checks only the type-derived schema.
  2. The `gobble.todo` entry is written by `writer.event` in `acpserver/agent.go`, which joins the files.
  3. `acpserver/commands_test.go` joins the files, for `todos` in its list of command names.
  4. `todo.md`'s Not for line reads `use write for a file, or answer in text`, the order the description test reads.
  5. `acpserver/testdata/session.golden` joins the files: regenerated, its diff exactly the one `todos` command.
  6. The race gate met 0013-REPORT's defect with a new signature. F1d-1 lands with it recorded. 0013-REPORT is amended, and joins the files with `docs/README.md`. gobble's nested marshal in `tool.New` goes to 0012-PLAN P9.
- **What was built:**
  - **`session`:** `TypeCustom`, `CustomTodo` (`gobble.todo`), and `Entry.CustomType` and `Entry.Data`, in Pi's shape.
  - **`tool`:** `Result.Plan`, `PlanItem`, and the plan's status and priority names. A nil `Plan` is no plan; an empty one clears.
  - **`tool/builtin/todo.go`:** `todo`, kind `think`, read-only and idempotent, with its template. Its schema publishes `minItems: 0`, the enums, `minLength` 1 on `content` and the default priority; its code refuses each breach in its own words.
  - **acpserver:**
    - `prompt.go` sends a result's plan as a `plan` update, after the call's terminal update;
    - `agent.go` records it as a `gobble.todo` entry, after the call's result;
    - `history.go` reads the active path's last list, and `session/load` sends it after the history unless it is empty;
    - `commands.go` adds `/todos`.
  - **Updated:** `builtin.go` (`todo` before the shells), `builtin_test.go`, `docs/architecture.md` and `README.md`.
  - **Records:** 0013-REPORT's amendment of 2026-10-09, and `docs/README.md`'s index. The index had called 0013's defect "reported upstream"; it has not been filed.
- **Choices of mine, within the entry's wording:**
  - `session/load` sends no plan for a cleared list, since a fresh client has none to clear.
  - `todo` refuses `todos: null`, which the validator passes, rather than reading it as a clear: `[]` clears.
  - The list is read along the active path, so a fork holds the list as it stood where it was forked. Fork and clone copy entries unchanged, so this follows from the code; no test drives it.
  - `commands.go`'s comment now says the command set is 0002-PLAN's frozen set plus what 0005-MADR's command table adds as phases land. 0005-MADR lists `/todos` for 1.0 (`:312`).
- **The acpserver tests,** through `acptest` with a JSONL store:
  - `TestTodoPlan`: one plan update with the entries in order, after the call's completed update; one `gobble.todo` entry; `/todos` printing `[x]`, `[>]` and `[ ]`; and `session/load`, in a new agent, sending the plan once, after the last agent message.
  - `TestTodoCleared`: a second, empty plan update; the empty list recorded; `/todos` saying `no todo list`; and no plan on load.
- **C3: every new check, seen failing on a scratch copy.** There were 19 mutations, each asserted to land exactly once:
  - **session:** the `customType` member renamed.
  - **tool:** the `plan` member renamed.
  - **todo:**
    - the status check removed, and the priority check removed, each serving the bad value;
    - the wrong default priority;
    - the counts swapped;
    - a cleared list returned with no plan;
    - the status enum not published;
    - `minItems` not published, which the schema convention test reported;
    - `todo` not read-only.
  - **acpserver:**
    - no plan update;
    - the plan sent before the terminal update;
    - no entry written;
    - no replay; the replay before the history; an empty list replayed;
    - `/todos`' in-progress mark changed;
    - `/todos` advertised with no handler;
    - the custom entry ignored by `replayState`.
  - Two cases first missed their expected text, though their tests failed: the log quotes the error with escaped quotes. They were re-pointed at the served plan, which is the defect, and both then passed.
- **The gates:** on Go 1.27.2, in three runs:
  - **The first run:**
    - lint printed `0 issues.` for linux, darwin and windows;
    - Windows `go test ./...` had no `FAIL` or `panic:`;
    - Windows and WSL `make preflight` each printed `preflight passed`;
    - **WSL `go test -race ./...` failed,** once, in `internal/cli`'s `TestSessionValues`, on the defect [0013-REPORT](../reports/0013-REPORT-go-json-nested-marshal-race.md) records (deviation 6).
  - **The second run,** after deviation 6's records:
    - lint, Windows `go test` and Windows `make preflight` passed as before;
    - **the WSL gate did not run to a result.** WSL's Go was moved from go1.27.1's directory to go1.27.2's while the run was going, and the gate's shell lost `go` part way through. That is no verdict, and is not counted as a pass.
  - **The third run,** of the WSL gate alone, with the same command, on an unchanged tree: WSL resolved `go1.27.2 linux/amd64`, `make preflight` printed `preflight passed`, and `go test -race ./...` exited 0 with no race reported.
  - The race gate was not changed.
- **Probe A** (Messages-shape bytes, on Windows):

  | Tool | Description | Definition |
  | :--- | ---: | ---: |
  | todo | 625 | 1297 |

**What the entry predicted wrongly:**

1. **It said the validator enforces an enum.** It does not, and 0012-MADR D2 rule 8, which the entry cites, says so. The claim was written from the schema rather than from `tool.New`.
2. **Its file list missed three files** that its own steps need: `acpserver/agent.go`, where entries are written, and two tests that pin the command list, `commands_test.go` and `session.golden`. The list was written from where updates are sent, not from where entries are written or from what pins the command set.
3. **Its exact template broke the description test** it was meant to pass. The template was not run through that test before the entry fixed its text.
4. **The race gate.** The entry expected F1d-1's gates to pass. The first race run met 0013-REPORT's defect with a signature that report said never varies (deviation 6). Separately, a WSL toolchain change during the second run left that run with no verdict.

**F1d-2, 2026-10-09 — complete (staged; the owner commits).** It ran as the entry "F1d made executable" wrote it, approved by the owner's "proceed to F1d-2" of 2026-10-09, with six deviations decided by the owner: 1–4 before any code was written, 5 and 6 while it was written.

- **The deviations** (recorded under F1d-2's tests above):
  1. `agent/event.go` and `tool/toolsearch/doc.go` join the files.
  2. The resource tools are built in `tool/builtin` from a `ResourceSource` that `mcpclient`'s manager implements, so D2's and D5's tests walk them.
  3. The CLI's `-t` and `--exclude-tools` accept the resource tools' names, filtered as MCP tools are.
  4. The race step runs natively on Windows and on a second amd64 Linux machine; WSL keeps `make preflight` alone.
  5. `acpserver/toolload_test.go` holds the acpserver tests.
  6. `README.md` joins the files.
- **What was built:**
  - **`tool`:** `ExposureHidden`; `Spec.Family` with `WithFamily`; `Env.Deferred`; `Result.Load`. The empty exposure stays direct.
  - **`tool/toolsearch`:** the BM25 index, with the name, first sentence, rest of the description and properties weighted 3, 2, 1 and 1; lower-cased words split at letter and digit boundaries; Pi's stop words; `k1` 1.2 and `b` 0.75; an exact name first; hidden specs not indexed.
  - **`agent`:**
    - direct tools are sent, deferred ones kept, hidden ones neither sent nor callable;
    - `tool_search` (`Config.Search`) is offered only while something is deferred;
    - a result's `Load` adds the tools, with their families, to the next request of the same turn, after the tools already sent, and emits `Loaded`;
    - `Config.Loaded` sends an earlier prompt's loads as direct, in load order;
    - each turn has its own copy of the tools, so loads never reach another session's turn.
  - **`tool/builtin`:** `tool_search` and the three resource tools, with their templates.
  - **`mcpclient`:** a server that advertises resources at `initialize` is listed; the manager lists resources and templates and reads contents, each bounded by the server's timeout; `mcptest` has a resource server.
  - **`acpserver`:**
    - a load is recorded as `custom{customType: "gobble.tools", data: {loaded}}`, and the next prompt starts with every load along the session's path;
    - the resource family loads at the first prompt that has it, and the load is recorded;
    - the system prompt names only the tools the first request sends (`agent.Agent.Offered`), so a deferred tool is never named as usable.
  - **`internal/cli`:** the resource tools' names are tool names.
  - **Updated:** `docs/architecture.md`'s agent, tool, tool/builtin, tool/toolsearch, acpserver and mcpclient rows; `README.md`.
- **Choices of mine, within the entry's wording:**
  - **`tool_search` reaches the agent as `Config.Search`,** built by acpserver, so the agent does not import `tool/builtin`.
  - **No camelCase split and no stemming** in the tokenizer. Pi has both, but the entry names only letter and digit boundaries, and Pi's stop words.
  - **The resource tools' arguments follow codex:** no `server` lists every server's first page, and a cursor needs its server.
  - **A resource read is cut at 50 KB,** the bound read and grep use.
  - **The `gobble.tools` name and its reader are in `acpserver/agent.go`,** beside the writer, rather than in `session` or `history.go`, which F1d-2 does not list.
- **The tests,** each seen failing on a scratch copy (C3): 22 mutations, each asserted to land exactly once:
  - **the index:** equal weights; no exact-name rank; stop words kept; hidden specs indexed; no length normalization;
  - **the agent:** deferred tools sent; `tool_search` always offered; a family not loaded whole; specs not encoded again after a load; hidden tools offered; no `Loaded` event; `Config.Loaded` ignored;
  - **`tool/builtin`:** `tool_search` setting no `Load`; a cursor accepted without a server; a blob shown as text;
  - **`mcpclient`:** every server treated as offering resources;
  - **acpserver:** a load not recorded; loads not replayed; no session-start load; the system prompt naming every tool;
  - **the CLI:** the resource names unknown; the resource tools unfiltered.
  - **Four cases first missed their expected text.** Three tests caught their defect at an earlier assertion, or printed escaped quotes, and were re-pointed at the text they print. Case 08's mutation left a variable unused and did not build, so it was rewritten to keep it used. All 22 then failed as expected; case 08's log shows the request without `c`.
- **Lint.** The first gate run's golangci-lint reported three findings, all in F1d-2's new code: `strings.EqualFold` in the index (gocritic), and a string built in a loop and a one-verb `Sprintf` in the resource tools (perfsprint). Both preflights failed on them alone. All three were fixed, and the gates and the 22 copies were run again.
- **The gates:** on Go 1.27.2, as deviation 4 sets them, after the lint fixes:
  - lint printed `0 issues.` for linux, darwin and windows;
  - Windows `go test ./...` had no `FAIL` or `panic:`;
  - **the race step:** `CGO_ENABLED=1 go test -race ./...` exited 0 with no race reported, natively on Windows and on a second amd64 Linux machine. The Windows run was the suite's first under the race detector there;
  - Windows and WSL `make preflight` each printed `preflight passed`, WSL's without a race run;
  - the 22 copies, run again, each failed as expected.
- **Probe A** (Messages-shape bytes, on Windows):

  | Tool | Description | Definition |
  | :--- | ---: | ---: |
  | tool_search | 685 | 1076 |
  | list_mcp_resources | 736 | 1117 |
  | list_mcp_resource_templates | 789 | 1179 |
  | read_mcp_resource | 602 | 982 |

**What the entry predicted wrongly:**

1. **Its file list missed six files, or groups,** that its own steps need: the event file, the placeholder package doc, a home for the resource tools that the D2/D5 tests reach, the CLI's tool flags, the acpserver tests' file, and `README.md`. The list was written from where the main code goes, not from what declares, documents, tests or names it.
2. **It put the resource tools in `mcpclient`** without asking how D2 and D5 would reach them there.
3. **Its gates named WSL's race run.** They were written before 0013-REPORT's third amendment showed that run cannot be trusted on this host, so this is a change of circumstance more than an error of the entry.
