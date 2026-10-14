---
status: in-progress
date: 2026-10-13
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

*(2026-10-09: F2 is made executable as two sub-phases, F2-1 (migration, every entry type's context, the fixtures and the fuzz target) and F2-2 (the tree), with v1 and v2 migrated in memory only. See the Amendments entry "F2 made executable".)*

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

*(2026-10-10: F3 is made executable as four sub-phases, F3-1 (context files and the sectioned prompt), F3-2 (skills), F3-3 (prompt templates) and F3-4 (`@path` and the prompt capabilities). See the Amendments entry "F3 made executable", and 0005-MADR's amendment of this date.)*

### F4 — providers, catalog, auth (1.0)

*(2026-10-10: F4 is made executable as three sub-phases, F4-1 (the SDK and the adapters), F4-2 (the catalog, `models.json` and choosing a model) and F4-3 (credentials). See the Amendments entry "F4 made executable", and 0005-MADR's second amendment of this date.)*

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
* An untrusted project's ~~`.gobble/settings.json`~~ `.gobble/settings.yaml` has no effect. A test
  fails on a copy that loads it anyway. *(2026-10-12: the file is YAML, per [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md). Its D5 reads the project layer but applies it only when F6's trust says so.)*

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
   *(2026-10-13, 0019-MADR D12: go-core-lib is go-selfupdate-lib, renamed 2026-10-02 in `v1.5.0`. F10 binds the newest go-selfupdate-lib at F10,
   `v1.15.2` today (0019-PLAN P2, `e8181b5`), whose Windows "Access is denied" defect is fixed from `v1.5.1`. The five binaries are noted above.)*
5. The Pi bridge (0003): discovery sources, `session list --pi`,
   `session import`, ~~`config import --from-pi`~~ *(dropped 2026-10-12 by the owner; 0018-MADR D13)*, `auth import --from-pi`,
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
  *(2026-10-13, 0019-MADR D12: `go list -m github.com/maccavelli/go-selfupdate-lib` prints the newest tag at F10.)*
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

*(2026-10-13, 0020-MADR D8: `--offline` also turns off 0020-MADR's model listing, which then offers the static models.)*

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
  *(2026-10-13, 0019-MADR D12: go-selfupdate-lib, renamed from go-core-lib, at its newest tag at F10.)*
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
  5. **Readiness.** Add `gobble auth check [--json]`, and `--offline` / `GOBBLE_OFFLINE`. *(2026-10-12: `--offline`'s phase is X3, by the later entry "F4 made executable" of 2026-10-10; 0018-MADR D20.)*
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
  *(2026-10-13, 0021-MADR D7: the SDK's block is lifted at `v1.12.0`; the skip reason becomes F4-2's per-model cache TTL and prices.)*
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
* **No settings mechanism exists.** `internal/config` is a placeholder, and the config surface is undecided (~~0008-MADR~~ 0002-MADR and 0004-MADR, amendments of 2026-10-04; corrected 2026-10-12, 0018-MADR D20). `shellPath`, `shellCommandPrefix`, `tools.bash.useClientTerminal` and `images.autoResize` have no home.
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
  - ~~Binary files, images and directories stay local.~~ *(Deviation 1, 2026-10-09: directories, by a stat, and images, by their extension, stay local. Every other file is read through the client, and its text is checked for binary content as bytes are now.)*
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

**F1d-3's deviations, 2026-10-09,** found by reading its files before any code was written (1) and while writing its docs (2), and decided by the owner:

1. **read cannot both keep binary files local and make no direct disk read.**
   - **Found:** read tells text from images and other binary files by sniffing a file's first bytes (`tool/builtin/sniff.go:17`, `:70`), after reading it (`read.go:50`). `fs/read_text_file` returns a string. The step says binary files stay local, and F1's acceptance line says a read through a capable client makes no direct disk read; the type cannot be known without reading.
   - **Decided:** classify without reading the content:
     - a directory, found by a stat, and an image, found by its extension, stay local, as today;
     - every other file is read through the client, and its text is checked for binary content as bytes are now; a binary file is refused.
   - **So:** a binary file with an unfamiliar extension is read by the client, then refused. No non-image file is read from disk.
2. **README.md.**
   - **Found:** its status line says the client's file and terminal methods are not built yet, and the files table leaves it out.
   - **Decided:** `README.md` joins F1d-3's files. Its status line drops F1d, and `gobble acp`'s line describes the routing.

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

**F1d-3, 2026-10-09 — complete (staged; the owner commits). With it, F1 is complete.** It ran as the entry "F1d made executable" wrote it, approved by the owner's "approved to proceed" of 2026-10-09, with two deviations decided by the owner: 1 before any code was written, 2 while its docs were written.

- **The deviations** (recorded under F1d-3's tests above):
  1. read classifies without reading: directories and images stay local, every other file is read through the client and then checked for binary content.
  2. `README.md` joins the files.
- **What was built:**
  - **`tool`:** `TextFiles` and `Terminal`, carried as `Env.Files` and `Env.Terminal`; `Result.Locations` of `Location{Path, Line}`; the `Locator` interface, with `WithLocations`.
  - **`tool/builtin`:**
    - read, write and edit do their text I/O through `textIO`. That is the client, when it offers files and the path is inside the roots; otherwise the workspace;
    - a path is stat-ed through the workspace before the client is asked, or for a new file its parent is made through it, so confinement is unchanged;
    - edit's compare-and-swap reads the client's text twice;
    - bash runs in the client's terminal when `Options.UseClientTerminal` is set and the client offers one: create, wait for exit, kill on cancel or timeout, read the output, release. The result has a local run's shape;
    - read, write, edit, move, delete, copy and mkdir name their paths; grep names its first 20 matches' files and lines.
  - **acpserver:**
    - `initialize` keeps the client's capabilities;
    - `clientFiles` and `clientTerminal` carry the calls over `fs/*` and `terminal/*`;
    - `tool_call` carries a call's locations from its Locator, and `tool_call_update` those of its result;
    - a running call carries the terminal it created, while it runs and in its last update.
  - **`acpclient/acptest`:** the client can serve files (`Files`, with `FileCalls`) and scripted terminals (`Terminals`, with `Calls` and `Requests`). Left unset, it still answers "method not found".
  - **Updated:** `docs/architecture.md`'s tool, tool/builtin, acpserver and acpclient/acptest rows; `README.md`.
- **Choices of mine, within the entry's wording:**
  - **`Env.Files` is set only when the client offers both reading and writing.** A client offering one would have edits read from its editor and written to disk.
  - **The interfaces take a context,** so a cancel reaches the client.
  - **A path outside the roots,** approved, and gobble's output directory are read and written here, not through the client.
  - **Only bash uses the client's terminal,** as the step says; powershell runs here.
  - **The client's terminal is asked to keep 4 MiB of output.** The model sees the same tail as a local run, and that output is saved as a local run's is.
  - **acpserver finds a call's locations from the turn's own tools,** so `agent` is unchanged. The running call is the one whose `ToolRun` came last, since a turn runs its calls one at a time.
- **The tests:**
  - `acpserver/routing_test.go`, through acptest:
    - a read through a client that offers files asks `fs/read_text_file` and shows the client's text, not the disk's;
    - an image stays local; an approved path outside the roots is read here;
    - a write asks `fs/write_text_file`; an edit reads twice, then writes, and the disk is unchanged;
    - bash makes `create`, `wait_for_exit`, `output` and `release`, and its tool call carries the terminal;
    - a timeout makes `kill`;
    - read and edit name their files at `tool_call`, and grep its matches at `tool_call_update`.
  - **`TestClientFilesConfined`,** in `tool/builtin`: a symbolic link inside the roots that leaves them is refused for a read, a write and a new file under a linked directory, and the client is never asked. The links were created on Windows, and the test ran whole.
- **C3: every new check, seen failing on a scratch copy.** There were 15 mutations, each asserted to land exactly once, and all 15 failed as expected on the first run:
  - no routing; images routed; an outside path routed;
  - both confinement guards removed (the routing's link resolution and the stat before the ask: either alone refuses the link, so the copy removes both);
  - write kept local; edit's second read taken from disk;
  - the client terminal ignored; no kill on timeout; the terminal not carried;
  - no `tool_call` locations; no grep locations; read's line dropped; the Locator answering nothing;
  - `Env.Files` not set; `Env.Terminal` not set.
- **The gates:** on Go 1.27.2, as F1d-2's deviation 4 sets them, all on the first run:
  - lint printed `0 issues.` for linux, darwin and windows;
  - Windows `go test ./...` had no `FAIL` or `panic:`;
  - `CGO_ENABLED=1 go test -race ./...` exited 0 with no race reported, natively on Windows and on a second amd64 Linux machine;
  - Windows and WSL `make preflight` each printed `preflight passed`, WSL's without a race run.

**What the entry predicted wrongly:**

1. **It asked read to keep binary files local and to read nothing from disk,** which together cannot be done. Its test line said "no direct disk read" without saying how read would know a file was binary.
2. **Its file list left out `README.md`,** as F1d-2's did.

**2026-10-09 — F2 made executable** (the owner's decisions of 2026-10-09: two sub-phases; v1 and v2 migrated in memory only; Pi's context rules for every entry type; fixtures written by Pi's own code).

F2 is steps 1–4 and their accept list, as the cross-repository amendment of 2026-10-01 changed them. 0002-PLAN Phase 4 built most of steps 2–4: entries round-trip, lazy files, the leaf as the last entry, fork, clone, names, `parentSession`, and the four ACP session methods. F2 adds migration, the context of every entry type, labels across fork and clone, and the operations that move the leaf.

**The facts,** checked on 2026-10-09 by a script in the session scratchpad (`q302_claims_f2.py`): 34 claims, all passing. With the check inverted, all 34 failed.

- **gobble today:**
  - It reads v3 only (`session/jsonl/store.go:198`). A malformed line is skipped and counted (`:208`), a torn last line is kept on its own line by the next write (`:178`, `:290`), and an unreadable header fails closed (`:194`).
  - The model's context holds messages and the newest compaction only; every other type "takes no part in context" (`acpserver/history.go:158`). The projection knows three roles (`compaction/estimate.go:167`), and a turn starts only at a user message or a compaction (`:117`).
  - `/session`'s totals add usage from compactions and assistant messages (`acpserver/commands.go:158`).
  - A session's name is the latest `session_info` that has one (`session/memory.go:127`).
  - Fork and clone copy the path's entries as they are (`acpserver/commands.go:474`), and `session/new`'s `forkFrom` copies the stored entries (`acpserver/agent.go:248`).
  - `Entry` has no member for `context_edit`'s or `label`'s target (`session/session.go`).
  - 0002-PLAN Phase 6 did not port Pi's advance over `context_edit` omissions, because gobble had no such entries (0002-PLAN `:1512`). Pi's files have them.
- **Pi at `312184edb`** (paths under `packages/coding-agent/src/core/`):
  - **Migration** (`session-manager.ts:287-347`):
    - v1 to v2 gives each entry a random 8-hex id (`:297`) and the previous entry as its parent;
    - a compaction's `firstKeptEntryIndex` counts the parsed lines, header included (`:305`), and becomes `firstKeptEntryId`;
    - v2 to v3 renames the message role `hookMessage` to `custom` (`:327`);
    - Pi then rewrites the file (`:1092`).
  - **Context** (`buildSessionProjection`, `:543-574`, and `convertToLlm`, `messages.ts:148-196`):
    - a `custom_message` becomes a user message of its content;
    - a `branch_summary` with a summary becomes a user message, `BRANCH_SUMMARY_PREFIX` + summary + `BRANCH_SUMMARY_SUFFIX` (`messages.ts:19-24`);
    - a `bashExecution` message becomes a user message in `bashExecutionToText`'s form, unless `excludeFromContext` (`messages.ts:82-98`, `:154`);
    - a `context_edit` with a `null` replacement drops its target (`:523`); otherwise it replaces the content of a user, assistant, tool-result or custom message. The last edit of a target wins (`:553`).
  - **Compaction** estimates, cuts and starts turns at the new roles (`compaction/compaction.ts:300-375`), and advances over an omitted recovery suffix (`:846`).
  - **The tree:**
    - the default leaf is the last entry (`:403`), and a `null` leaf is the empty path (`:397-399`);
    - `resetLeaf` makes the next entry a root (`:1591`), and `branchWithSummary` names `root` when there is no leaf (`:1610`);
    - `createBranchedSession` leaves labels out, re-chains the path, and appends the path's labels again (`:1632-1712`).
  - **The name** is the latest `session_info` in file order, trimmed; an empty one clears it (`:1325`).
  - **Session stats** count `usage` entries, branch summaries' and compactions' usage, and tool results' usage (`agent-session.ts:4093-4116`).
  - **Writers of v1 and v2:** Pi wrote v1 until `c58d5f20a` (`c58d5f20a^` has no version constant, and stores compactions by index) and v2 until `c6fc08453` (`c6fc08453^` has `CURRENT_SESSION_VERSION = 2`). `312184edb` writes v3 only.
  - **Pi's own fixtures** (`test/fixtures/*.jsonl`) are v1 files of real sessions, carrying their author's paths and conversations, so they are not copied.
- **Two Pi defects, recorded and not followed:**
  - `forkFrom` copies the source's entries without migrating them (`:1823`; `loadEntriesFromFile`, `:627-670`, does not migrate). A v1 source makes a v3 file of entries with no ids.
  - agent-core's v3 importer lists the types it accepts and throws on any other (`packages/agent/src/harness/session/jsonl/legacy-v3.ts:204-216`). `usage` and `context_edit` are not in the list.
- **The second amd64 Linux machine** has node v24.21.0 and npm 11.19.0. Its `/tmp` is a 512 MB tmpfs (`ssh … node --version; npm --version; df -h /tmp`, 2026-10-09).

**The owner's decisions, 2026-10-09:**

1. **Two sub-phases,** each one commit with its own record and gates:
   - **F2-1, reading:** v1 and v2 migration, every entry type's context, the fixtures, and the fuzz target;
   - **F2-2, the tree:** the leaf operations (move, reset, branch with a summary), and labels across fork and clone, as Pi keeps them.
2. **Migration is in memory only.** gobble never rewrites a file on load (0005-MADR, "Sessions (changed)"). An older file is read and migrated each time it is loaded. Appending to it is refused; it can be forked or cloned to a new v3 file.
3. **Pi's context rules apply to every entry type,** so the model sees what Pi's would.
4. **The fixtures are written by Pi's own code,** at the commits that wrote each version, with synthetic content.

**F2-1 — reading: migration, every entry type, fixtures and fuzzing.**

Choices made within the wording:

- **`session`:**
  - **Types and roles:** `TypeUsage`, `TypeContextEdit`, `TypeBranchSummary`, `TypeCustomMessage` and `TypeLabel`; `RoleCustom`, `RoleBashExecution`, `RoleBranchSummary`, `RoleCompactionSummary` and `RoleSystem`.
  - **`Entry` gains Pi's members** for the remaining types: `TargetID` (`targetId`), `Replacement jsontext.Value` (`replacement`, where `null` is kept and differs from absent), `FromID` (`fromId`), `Content jsontext.Value` (`content`), `Display *bool`, `Label *string`, and `usage`'s `Kind`, `Model` and `Note`.
  - ~~**`Message` gains the members of Pi's other roles:** `CustomType`, `Display *bool`, `Details`, `Summary`, `FromID`, `TokensBefore *int64`; and `bashExecution`'s `Command`, `Output`, `ExitCode *int`, `Cancelled *bool`, `Truncated *bool`, `FullOutputPath` and `ExcludeFromContext *bool`.~~ *(F2-1 deviation 4, 2026-10-10: Pi writes `null` and `""` for these two.)*
  - **`Message` gains the members of Pi's other roles:** `CustomType`, `Display *bool`, `Details`, `Summary`, `FromID`, `TokensBefore *int64`; and `bashExecution`'s `Command`, `Output *string`, `ExitCode jsontext.Value`, `Cancelled *bool`, `Truncated *bool`, `FullOutputPath` and `ExcludeFromContext *bool`.
  - **Pointers** are used wherever Pi writes `false` or `""`, so the value round-trips.
  - **`Migrate(version int, entries []Entry) ([]Entry, error)`:**
    - **v1:** entry n, counting from 1, gets the id `fmt.Sprintf("%08x", n)` and the previous entry as its parent. A compaction's `firstKeptEntryIndex`, counting the header as 0 as Pi does, becomes the `firstKeptEntryId` of that entry, and the index is removed. An index at the header, or past the end, leaves no id, as Pi leaves none.
    - **v2, and v1 after it:** a message with the role `hookMessage` gets the role `custom`.
    - **Above v3:** an error.
    - **The ids are fixed, not random as Pi's are,** because the file is never rewritten: every load must give the same ids, since `/fork <entry>` and `_gobble/get_entries`' `since` name them.
  - **`Name(entries) string`** is Pi's rule: the latest `session_info` in file order, trimmed; empty is no name.
  - **`ErrOldVersion{ID, Version}`**, whose text is `session <id> is a version <n> file, which gobble reads but does not rewrite; /clone it to continue in a new session`.
- **`session/jsonl`:**
  - A header of version 1, 2 or 3 is read, an absent version being 1 as now. The entries are migrated in memory. Any other version fails closed as now.
  - A log opened on an older file refuses `Append` with `ErrOldVersion` and writes nothing.
  - `List` lists older files.
- **`compaction`:**
  - **The projection is exported** as `Project(ctx []session.Entry) []Projected`, with `Projected{Entry session.Entry; Messages []*session.Message}`. Each message is in Pi's `AgentMessage` shape:
    - a message entry gives its message, with `null` content read as empty (`session-manager.ts:444-449`);
    - a `custom_message` gives a `custom` message (`customType`, `content`, `display`, `details`, and its timestamp in milliseconds);
    - a `branch_summary` with a summary gives a `branchSummary` message (`summary`, `fromId`);
    - the leading compaction gives a `compactionSummary`, and an older compaction in the kept range gives nothing, both as now.
  - **Context edits:** the last edit of each target among the context entries applies. A `null` replacement drops the target's messages. Any other replaces the content of a user, assistant, tool-result or custom message, a string becoming one text block for an assistant or tool result.
  - **Estimates, cut points and turn starts** take Pi's roles (`compaction.ts:300-375`):
    - a custom message and a tool result count their content; a bash execution, its command and output; a branch or compaction summary, its summary;
    - cut points are user, assistant, bash execution, custom, branch summary and compaction summary messages;
    - turn starts are the same, without assistant messages.
  - **The omission advance is ported** (`compaction.ts:832-850`). This replaces 0002-PLAN Phase 6's choice of not porting it, and that line gets a dated note.
  - **The summary prompt** serialises the converted messages, as Pi converts before it serialises (`compaction.ts:724`).
- **`acpserver`:**
  - **The model's history** is built from the projection, converted as `convertToLlm` converts:
    - a `custom` message becomes a user message of its content;
    - a `branchSummary`, a user message of Pi's prefixed text;
    - a `compactionSummary`, as now;
    - a `bashExecution`, a user message of `bashExecutionToText`'s text, unless it is excluded.
    - Images are left out, as a user message's are today (step 2 of F1 waits for the provider SDK).
  - **The name** is `session.Name`, in `replayState` and in `Summarize`.
  - **`_gobble/get_messages`** returns the projection's messages, in Pi's shapes.
  - **`session/load`'s replay:**
    - a custom message whose `display` is true, as one `user_message_chunk` of its text;
    - a branch summary, as one `agent_message_chunk`, `Branch summary:\n\n<summary>`;
    - a bash execution the model sees, as one `user_message_chunk` of its text.
    - A hidden custom message, and an excluded bash execution, are not replayed.
  - **`/session`'s totals** count what Pi's stats count: `usage` entries, and the usage of branch summaries, compactions, assistant messages and tool results.
  - **An append refused with `ErrOldVersion`** fails its request as ACP invalid request, with the error's text, through one helper in `acpserver/agent.go`. That covers a prompt (`agent.go:371`), `/name` (`commands.go:370`), `/compact` (`compact.go:75`), and a model or thinking change (`config.go:88`).
  - **`/fork`, `/clone`, `_gobble/fork` and `_gobble/clone` still work** on an older file, writing v3 files of the migrated entries. So does `session/new`'s `forkFrom`, which copies the migrated entries, not the unmigrated ones Pi's `forkFrom` copies.
- **A Pi system message is kept and not sent.** A `system` message entry and a compaction's `systemMessage` carry Pi's prompt and tool state. gobble writes neither, and F3 owns the prompt (0002-PLAN Phase 4). Both round-trip, take no part in gobble's history or estimate, and are not replayed. This is the one place F2 does not apply Pi's rule.
- **The fixtures,** in `session/testdata/pi/`:
  - **Where they are written:** on the second amd64 Linux machine, so every recorded path is under `/tmp` and names no account.
    - Each Pi checkout is a `git archive` of the local Pi repository: `c58d5f20a^` for v1, `c6fc08453^` for v2, and `312184edb` for v3.
    - Each is copied with `scp` into a directory under the home directory, as `/tmp` is too small, installed with `npm ci --ignore-scripts`, and removed afterwards.
    - Sessions are written under `/tmp/gobble-f2/sessions`, with the cwd `/work/project`.
  - **The generators** are three TypeScript files in the scratchpad's `f2/gen/`:
    - `v1.ts` and `v2.ts`, run with their checkout's `tsx`;
    - `v3.test.ts`, run with its checkout's `vitest`, since `312184edb` has no `tsx`.
    - They call SessionManager's own methods, with invented prompts, answers and tool calls, and no real conversation.
  - **The files:**
    - **`v1.jsonl`:** model and thinking changes, a user message, an assistant message with text, thinking and a tool call, its result, a compaction whose index names a later message, then more messages. With it, `v1-branched.jsonl`, from `createBranchedSessionFromEntries`.
    - **`v2.jsonl`:** a `hookMessage`, a bash execution, a `custom` entry, labels, a compaction, a custom message, `branch`, `branchWithSummary` and `resetLeaf`. With it, `v2-branched.jsonl`, from `createBranchedSession`.
    - **`v3.jsonl`:** every type:
      - messages: user, assistant, tool result, custom, and two bash executions, one excluded;
      - `model_change`, `thinking_level_change`, `usage` and `compaction`;
      - `context_edit` twice, once with a replacement and once with `null`;
      - `branch_summary`, `custom`, and `custom_message` twice, once with `display` true and once false;
      - `label`, set and cleared;
      - `session_info`, set and cleared;
      - `branch` and `resetLeaf`.
    - With it, **`v3-branched.jsonl`,** from `createBranchedSession` of a labelled path.
    - For each file, **`<name>.context.json`**, from Pi at `312184edb`: the messages of `SessionManager.open(copy).buildSessionContext()`, and `convertToLlm` of them.
    - For v1 and v2, **`<name>.migrated.jsonl`:** the copy Pi rewrote when it opened it.
  - **Before they are staged,** the identifier scan and the hidden-character scan run over them, and the record lists each file with its size.
- **The fuzz target** is `FuzzParse` in `session/jsonl`, seeded with every fixture. It runs as a gate: `go test -run '^$' -fuzz FuzzParse -fuzztime 60s ./session/jsonl`.

**F2-1's files:**

| | Files |
| :--- | :--- |
| new | `session/migrate.go`, `migrate_test.go`; `session/testdata/pi/` (the fixtures above); `session/jsonl/fixtures_test.go`, `fuzz_test.go`; `compaction/project_test.go`; `acpserver/fixtures_test.go` |
| changed | `session/session.go`, `session_test.go`, `memory.go`; `session/jsonl/store.go`, `store_test.go`; `compaction/estimate.go`, `compaction.go`, `serialize.go`, `compaction_test.go`; `acpserver/history.go`, `extensions.go`, `commands.go`, `agent.go`, `compact.go`, `config.go`; `docs/architecture.md`; `README.md`; `docs/decisions/0002-PLAN-cli-acp-headless-mcp-v1.md` (the dated note at `:1512`) |

*(2026-10-10, F2-1 deviations 2 and 3: `session/jsonl/testdata/fuzz/FuzzParse/` (the seed) joins the new files, and `session/doc.go` the changed files.)*

**F2-1's tests,** each seen failing first on a scratch copy:

- **`session`:**
  - `Migrate`: v1's ids and chain; an index to a later entry, to the header, and past the end; `hookMessage` to `custom`; version 4 refused.
  - `Name`: the latest wins, it is trimmed, and an empty one clears.
- **`session/jsonl`, against the fixtures:**
  - each v3 file loads with no line skipped, and every entry, written again, equals its line after JSON normalisation (both decoded into `any` and compared);
  - each v1 and v2 file loads, and its entries equal Pi's `.migrated.jsonl` with ids mapped by position. Its bytes are unchanged after loading and after `Close`;
  - an `Append` to an older file returns `ErrOldVersion`, and the file is unchanged;
  - on copies of `v3.jsonl`:
    - a malformed line is skipped with a warning;
    - an unreadable header fails closed;
    - a last line cut short loads every complete entry.
  - ~~`FuzzParse` does not panic. Every entry it returns has a type and is not a header, and an older file's ids are unique.~~ *(F2-1 deviation 2, 2026-10-10: only v1's ids are assigned by migration.)*
  - `FuzzParse` does not panic. Every entry it returns has a type and is not a header, and a v1 file's migrated ids are unique. Its first failing input is kept as a seed under `session/jsonl/testdata/fuzz/FuzzParse/`.
- **`compaction`:**
  - `Project` of each fixture's path equals its `.context.json` messages after JSON normalisation;
  - the estimate of each new role;
  - the omission advance, and a suffix it must not advance over.
- **`acpserver`, through `acptest`:**
  - loading `v3.jsonl` gives a history whose roles and texts are those of `.context.json`'s converted messages, and `_gobble/get_messages` returns its messages;
  - the replay's chunks for a custom message, a branch summary and a bash execution, and none for the hidden ones;
  - the model and thinking level are restored from the file's changes;
  - a prompt to the v1 session fails with `ErrOldVersion`'s text, and the file is unchanged;
  - `/clone` of the v1 session, and `session/new`'s `forkFrom` of it, write v3 files of the migrated entries;
  - `/session`'s totals count a `usage` entry.

**F2-1's deviations,** found by reading Pi's code for the fixture generators before any code was written (1, 2026-10-09) at the fuzz target's first run (2, 2026-10-10), and on checking the files touched against the entry (3 and 4, 2026-10-10), and decided by the owner:

1. **Pi's v2 `createBranchedSession` is not v3's.**
   - **Found:** at `c6fc08453^` (`session-manager.ts:918-1000`) it leaves the label entries out without re-chaining, so a kept entry can name a removed label as its parent, and it gives the labels it writes again new timestamps. At `312184edb` (`:1632-1712`) it re-chains, and keeps each label's timestamp. F2-2's test compared `Branched` with both, and `Branched` follows v3.
   - **Decided:** F2-2 compares `Branched` with `v3-branched.jsonl` only. `v2-branched.jsonl` stays a v2 fixture, which F2-1's tests load, and its path ends at the entry whose parent was a label. No file is added.

2. **Only v1's ids are migration's.**
   - **Found:** `FuzzParse` failed within 3 seconds, on a v2 header followed by two entries with no id: `migrated id "" twice`. Migration gives v1's entries their ids. v2's entries keep the ids Pi wrote, and Pi's v2-to-v3 step assigns none (`session-manager.ts:316-330`), so the test line's "an older file's ids are unique" holds for v1 only.
   - **Decided:** the property checks unique ids for v1 input only, and v2 data stays as Pi wrote it. The failing input is kept as a regression seed in `session/jsonl/testdata/fuzz/FuzzParse/`, Go's place for it, which joins F2-1's files.

3. **`session/doc.go`.**
   - **Found:** its package comment says entry types gobble "does not act on yet" round-trip through `Unknown`, and that the format read is v3. After F2-1 both are false, and the files table leaves it out.
   - **Decided:** `session/doc.go` joins F2-1's changed files. Its comment says Pi's entry types are typed, and v1 and v2 are migrated in memory and never rewritten. No code changes.

4. **Two `bashExecution` members cannot be `*int` and a plain string.**
   - **Found:** Pi writes `"exitCode": null` for a command a signal ended, and `"output": ""` for one with no output. `*int` reads `null` as nil and leaves the member out when written, and an `omitzero` string leaves `""` out, so neither line round-trips.
   - **Decided:** `ExitCode` is a `jsontext.Value`, kept raw, and `Output` is a `*string`. `session_test.go`'s round-trip lines include both values.

**F2-2 — the tree.**

Choices made within the wording:

- **`session.Tree`** is a session's entries with a leaf that can move, as Pi's SessionManager keeps them:
  - `NewTree(entries)` puts the leaf on the last entry in file order, as a load does.
  - `Leaf()`, and `Path()`: from the leaf to the root, or nothing when there is no leaf. A missing parent, or a cycle, ends the path, as now.
  - **`Next(e Entry) Entry`** stamps an entry as a child of the leaf, with a new id. **`Add(e Entry)`** records it and moves the leaf to it. They are separate so a failed write leaves the tree as it was.
  - **`Branch(id) error`** moves the leaf to an entry, and **`ResetLeaf()`** removes it, so the next entry is a root.
  - ~~**`BranchSummary(from, summary string, details jsontext.Value) Entry`** is Pi's `branchWithSummary`: a `branch_summary` whose parent is `from` (none for the root), whose `fromId` is the old leaf or `root`, and which becomes the leaf.~~ *(F2-2 deviation 1, 2026-10-10: an unknown `from` is an error, and the leaf moves only through `Add`.)*
  - **`BranchSummary(from, summary string, details jsontext.Value) (Entry, error)`** is Pi's `branchWithSummary`: it returns a stamped `branch_summary` whose parent is `from` (none for the root, `""`) and whose `fromId` is the old leaf or `root`. An unknown `from` is an error, as Pi throws. The summary becomes the leaf through `Add`, once its write has succeeded, as `Next`'s entries do.
  - **`Entries()`** is every entry in file order. The tree holds a mutex, so the session's readers may call it, and `Path`, while an entry is added.
  - **`Labels() []Label`**, with `Label{TargetID, Label, Timestamp}`, resolves the label entries in file order, as Pi's map does:
    - the latest label of each target, with its timestamp;
    - an empty or absent label clears it;
    - targets in the order their label was first set since it was last cleared.
  - **`Branched(leaf string) []Entry`** is Pi's `createBranchedSession` (`session-manager.ts:1632-1712`):
    - the path to `leaf`, without its label entries, each entry's parent being the entry before it;
    - a compaction's `firstKeptEntryId` that named a label moves to the next entry kept;
    - then one label entry for each label whose target is in that path, chained, with the label's timestamp and a new id.
- **acpserver:**
  - **A live session holds a `Tree`.** It replaces the session's leaf and id set, and `append` is `Next`, the log's `Append`, then `Add`.
  - **Every path is the tree's.** That covers the prompt's loaded tools, `/compact`, the estimates, `/todos`, `_gobble/get_state` and the replay. With nothing moving the leaf yet, these are the paths of today.
  - **Fork and clone write `Branched`:** `/clone` and `_gobble/clone` of the leaf, and `/fork` and `_gobble/fork` of the parent of the message they name.
  - **`session/load` and `session/resume`** restore the tree with its leaf on the last entry. `session/list` and `session/close` are unchanged.
- **Nothing in F2-2 moves the leaf from a command.** `/tree <id>`, with its generated branch summary, and `/label` are X1 step 1 (0005-MADR, "Tree navigation and labels", 1.x). They will call `Branch`, `ResetLeaf`, `BranchSummary` and a label append.

**F2-2's files:**

| | Files |
| :--- | :--- |
| new | `session/tree.go`, `tree_test.go`; `acpserver/tree_test.go` |
| changed | `acpserver/agent.go`, `commands.go`, `compact.go`, `extensions.go`, `history.go`, `prompt.go`; `docs/architecture.md` |

*(2026-10-10, F2-2 deviation 2: `session/session.go` joins the changed files.)*

**F2-2's tests,** each seen failing first on a scratch copy:

- **`session.Tree`:**
  - after `Branch`, the next entry starts a sibling branch, and `Path` follows the leaf;
  - after `ResetLeaf`, `Path` is empty and the next entry is a root;
  - `BranchSummary` from the root has no parent and `fromId` `root`; from an entry, that parent and the old leaf;
  - `NewTree` of the entries puts the leaf on the last in file order, whatever moved it before;
  - `Labels` over set, reset, clear and set again;
  - ~~`Branched` of `v2.jsonl`'s and `v3.jsonl`'s leaves equals Pi's `v2-branched.jsonl` and `v3-branched.jsonl`, with the new label entries' ids mapped by position.~~ *(F2-1 deviation 1, 2026-10-09: v2's `createBranchedSession` does not re-chain, so only v3 is compared.)*
  - `Branched` of `v3.jsonl`'s leaf equals Pi's `v3-branched.jsonl`, with the new label entries' ids mapped by position. `v2-branched.jsonl` loads as a v2 file, and its path ends at the entry whose parent was a label.
- **acpserver:**
  - `/clone` of a labelled session writes its labels at the end, for targets on the path only;
  - `/fork` leaves out a label whose target is off the path;
  - a compaction whose first kept entry was a label keeps the entry after it.

**F2-2's deviations, 2026-10-10,** found by reading its files before any code was written (1) and on checking the files touched against the entry (2), and decided by the owner:

1. **The tree's API.**
   - **Found:**
     - `BranchSummary` returned an `Entry` with no error, but Pi's `branchWithSummary` throws on an unknown `from` (`session-manager.ts:1607-1609`). Its summary "becomes the leaf" before any write, unlike `Next` and `Add`.
     - acpserver's replay state needs every entry in file order, for Pi's name, the labels and the ids taken, beside the path. The tree offered `Leaf` and `Path` only, and nothing made it safe for the session's readers during an append.
   - **Decided:** `BranchSummary` returns `(Entry, error)`, and its summary becomes the leaf through `Add`. The tree gains `Entries()` and an internal mutex. No file is added.

2. **`session/session.go`.**
   - **Found:** `Path` and the tree must walk alike: a missing parent or a cycle ends the path, and a last entry with no id is still the leaf, as Pi's `leaf ??= entries[last]` makes it. The files table leaves `session/session.go` out.
   - **Decided:** `session/session.go` joins F2-2's changed files. `Path`'s loop is an unexported `pathFrom(entries, byID, start)`, which `Path` and `Tree` both call. `Path`'s behaviour and API are unchanged.

**The gates, for each sub-phase,** as F1d-2's deviation 4 sets them, with the fuzz run added:

- lint for linux, darwin and windows;
- Windows `go test ./...`;
- `CGO_ENABLED=1 go test -race ./...`, natively on Windows and on the second amd64 Linux machine;
- Windows and WSL `make preflight`, WSL's without a race run;
- for F2-1, `FuzzParse` for 60 seconds.

**How F2's accept list is met:**

- **The samples in Pi's `docs/session-format.md`** are read as the generated fixtures. Those samples are not valid JSON (0005-MADR, "Sessions (changed)"), and the 2026-10-01 amendment moved the golden files to real Pi output.
- **"Captured from real Pi output at `312184edb`"** holds for v3. v1 and v2 are captured at the last commits that wrote them, since `312184edb` writes v3 only.
- **A v1 file and a v2 file migrate:** they equal Pi's own migration.
- **A malformed line, an unreadable header and a cut-short last line:** the three `session/jsonl` tests.
- **The fuzz target:** `FuzzParse`.
- **The leaf is the last entry in file order:** `NewTree`'s test.
- **Model and thinking changes restored on load:** the acpserver test of `v3.jsonl`.

**Deferred, named:**

- **`/tree`, `/tree <id>` with a branch summary, `/label`, `_gobble/get_tree` and the TUI's tree navigator:** X1 steps 1 and 5. They call the tree's operations.
- **Writing `context_edit` entries:** X1 step 1.
- **Pi import** (`gobble session list --pi`, `gobble session import`): 1.x, through the 0003-MADR bridge.
- **Appending to a v1 or v2 file:** refused by decision 2. A clone continues it.
- **Images in the model's history:** step 2 of F1, with the provider SDK.
- **Pi's two defects** are recorded here only. Reporting them upstream is the owner's call.

**F2-1, 2026-10-10 — complete (staged; the owner commits).** It ran as the entry "F2 made executable" wrote it, approved by the owner's "approved to proceed to F2-1" of 2026-10-09, with four deviations decided by the owner: 1 before any code was written, 2 at the fuzz target's first run, 3 and 4 on checking the files touched against the entry. F2-2 has not run.

- **The deviations** (recorded under F2-1's deviations above):
  1. F2-2 compares `Branched` with `v3-branched.jsonl` only, since Pi's v2 `createBranchedSession` does not re-chain.
  2. The fuzz property checks unique ids for v1 input only; its first failing input is kept as a seed.
  3. `session/doc.go` joins the files, for its package comment.
  4. `bashExecution`'s `ExitCode` is a `jsontext.Value` and its `Output` a `*string`, so `null` and `""` round-trip.
- **The fixtures,** 16 files in `session/testdata/pi/`, were written on the second amd64 Linux machine by Pi's own SessionManager:
  - **at `c58d5f20a^`** with its `tsx`: `v1.jsonl` (2563 bytes) and `v1-branched.jsonl` (1909);
  - **at `c6fc08453^`** with its `tsx`: `v2.jsonl` (6550) and `v2-branched.jsonl` (5407);
  - **at `312184edb`** with its `vitest`: `v3.jsonl` (8764) and `v3-branched.jsonl` (7372); then, for all six, `<name>.context.json` (v1 3425, v1-branched 4022, v2 7744, v2-branched 5719, v3 7977, v3-branched 7977), and, for the four older ones, Pi's own migration, `<name>.migrated.jsonl` (v1 2917, v1-branched 2143, v2 6545, v2-branched 5402).
  - Each checkout was a `git archive` of the local Pi repository, installed with `npm ci --ignore-scripts` under the home directory there, and removed afterwards with `/tmp/gobble-f2`; a listing then found neither.
  - The identifier scan and the hidden-character scan found nothing in them.
- **What was built:**
  - **`session`:** the types `usage`, `context_edit`, `branch_summary`, `custom_message` and `label`, and the roles `custom`, `bashExecution`, `branchSummary`, `compactionSummary` and `system`; their members on `Entry` and `Message`; `Migrate`; `Name`; `ErrOldVersion`. `Summarize` takes its name from `Name`.
  - **`session/jsonl`:** versions 1 to 3 are read, and older entries migrated; a log on an older file refuses `Append`, and its `Writable` says so before any work.
  - **`compaction`:** `Project` and `Projected`, the context in Pi's message shapes with context edits applied; `Convert`, Pi's `convertToLlm`; `BashText` and `BranchSummaryText`; the new roles in the estimate, the cut points and the turn starts; usage after a context edit untrusted; the omission advance; the summary prompt serialising converted messages.
  - **`acpserver`:**
    - the history, `_gobble/get_messages` and `_gobble/get_last_assistant_text` come from the projection;
    - the replay shows a shown custom message, a branch summary and a bash execution the model sees, and nothing hidden;
    - `/session`'s totals count Pi's;
    - `appendFailed` answers an older file's refused append as invalid request, for a prompt, `/name`, `/compact` and a config change;
    - `/compact` checks `Writable` before asking for a summary.
  - **Updated:** `docs/architecture.md`'s session, session/jsonl, compaction and acpserver rows; `README.md`'s sessions line; and 0002-PLAN Phase 6's two context choices, each with a dated note.
- **Choices of mine, within the entry's wording:**
  - **`/compact` is refused before the summary is asked for,** so an older session's refusal costs no model call. `fileLog.Writable` is a method of an unexported type, found by an interface assertion, so the package's API is unchanged.
  - **`_gobble/get_last_assistant_text` reads the projection,** so an edited answer is the edited text, as the model sees it.
  - **A Pi `system` message takes no part in the cut point's omission check either:** an entry gobble does not project is invisible, not omitted.
  - **0002-PLAN's note covers both superseded lines** of Phase 6's choices, the omission advance and the context-invisible types.
  - **Three tests were added** where a mutation found no check: a custom message as a cut point and turn start; usage after a context edit untrusted; a hidden custom message not replayed. The fixture's only hidden message is also dropped by a null edit, so it could not show the last.
- **The tests:**
  - `session`: `TestMigrateV1`, `TestMigrateV2`, `TestName`, and `TestPiShapesRoundTrip`'s new lines of every type, with `null`, `false` and `""`.
  - `session/jsonl`: `TestFixturesRoundTrip`, `TestFixturesMigrate`, `TestFixtureDamage` and `FuzzParse`; `TestBadHeaderFailsClosed` now names version 4.
  - `compaction`: `TestProjectMatchesPi` (all six fixtures, Pi's messages and Pi's conversion), `TestEstimateNewRoles`, `TestProjectEdits`, `TestCutAtNewRoles`, `TestEstimateAfterEdit` and `TestOmissionAdvance`.
  - `acpserver`: `TestPiFixtureContext`, `TestPiFixtureTotals`, `TestPiFixtureOlder` and `TestReplayHidesHidden`.
- **C3: every new check, seen failing on a scratch copy.** There were 27 mutations, each asserted to land exactly once, and all 27 failed as expected on the first run, none by a build failure:
  - migration: v1's ids shifted; the index counted without the header; no `hookMessage` rename; the append not refused; the entries not migrated; the name not trimmed; duplicate v1 ids, which `FuzzParse`'s seeds caught;
  - the round trip: `replacement`'s `null` dropped by `omitempty`;
  - the projection: edits ignored; no `custom_message`; no `branch_summary`; an excluded bash execution sent; `bashExecutionToText`'s wording; the branch summary's suffix;
  - estimates and cuts: a custom message not estimated; a bash execution's output not estimated; no omission advance; an outside replacement ignored; unsent input ignored; a custom message not a turn start; usage trusted after an edit;
  - acpserver: the history not converted; a hidden custom message replayed; `get_messages` empty; `usage` entries not counted; the refusal internal, not invalid request; `/compact` not refused before its summary.
- **The gates:** on Go 1.27.2, as F1d-2's deviation 4 sets them, with the fuzz run added.
  - **The first run failed lint** on all three systems, and with it both preflights: `goconst` found the new code comparing `"null"` four times (`compaction/compaction.go:214`, `compaction/estimate.go:192`). Every other gate passed. One helper, `missing`, now makes the four tests; the tests, the 27 negatives and every gate were run again on that tree.
  - **On the second run:**
    - lint printed `0 issues.` for linux, darwin and windows;
    - Windows `go test ./...` had no `FAIL` or `panic:`;
    - `CGO_ENABLED=1 go test -race ./...` exited 0 with no race reported, natively on Windows and on a second amd64 Linux machine;
    - Windows and WSL `make preflight` each printed `preflight passed`, WSL's without a race run;
    - `go test -run '^$' -fuzz FuzzParse -fuzztime 60s ./session/jsonl` passed.

**What the entry predicted wrongly:**

1. **Its F2-2 test assumed Pi's v2 and v3 branching agree.** They do not.
2. **Its fuzz property assumed migration gives every older file its ids.** Only v1's.
3. **Its files table left out `session/doc.go`.**
4. **Two of its member types could not round-trip** Pi's `null` and `""`.
5. **Three of its behaviours had no test that a mutation could fail;** the tests are added above.

**F2-2, 2026-10-10 — complete (staged; the owner commits). With it, F2 is complete.** It ran as the entry "F2 made executable" wrote it, approved by the owner's "approved to proceed to F2-2" of 2026-10-10, with two deviations decided by the owner: 1 before any code was written, 2 on checking the files touched against the entry.

- **The deviations** (recorded under F2-2's deviations above):
  1. `BranchSummary` returns `(Entry, error)`, and its summary becomes the leaf through `Add`; the tree gains `Entries()` and a mutex.
  2. `session/session.go` joins the files: `Path` and `Tree` share one walk, `pathFrom`.
- **What was built:**
  - **`session.Tree`:** `NewTree`, `Leaf`, `Entries`, `Path`, `Next`, `Add`, `Branch`, `ResetLeaf`, `BranchSummary`, `Labels` and `Branched`, with `Label`. The leaf is kept as an index, so a last entry with no id is still the leaf, as Pi's `leaf ??= entries[last]` makes it.
  - **acpserver:**
    - a live session holds a `Tree` in place of its leaf and id set; `append` stamps with `Next`, writes, then calls `Add`;
    - every path is the tree's: the prompt's loaded tools, `/compact`, the estimates, `/todos`, `/context`, `/name`, `_gobble/get_state`, `get_messages`, `get_last_assistant_text`, the replay and `resumedMeta`, through `liveSession.state`;
    - `/clone`, `_gobble/clone`, `/fork` and `_gobble/fork` write `Branched`; `pathTo` is gone;
    - `_gobble/get_entries`' `leafId` is the tree's leaf, Pi's `getLeafId`.
  - **Updated:** `docs/architecture.md`'s session and acpserver rows.
- **Choices of mine, within the entry's wording:**
  - **`replayState(entries)` stays,** the state of stored entries with the leaf on the last, and `replayPath(entries, path)` is a live session's. `acpserver/session_test.go` calls the first, and is not in the files.
  - **A move of the leaf leaves the session's history to its caller to rebuild,** as Pi's `navigateTree` refreshes its context. The history is built at a load, a compaction and each turn. F2-2 moves the leaf from no command, and X1's `/tree` will rebuild it; `TestPromptFollowsTheLeaf` does what that caller will do.
  - **`get_entries`' leaf** was not named by the entry. It is the tree's, so the leaf a client sees is the one the next entry follows.
  - **Two tests were tightened** where a mutation found no check: a label set again keeps its place, and the entry after a written prompt is its child.
- **The tests:**
  - `session`: `TestTreeLeaf`, `TestBranchSummary`, `TestLabels`, `TestBranchedMatchesPi` (Pi's own `v3-branched.jsonl`, entry for entry), `TestV2BranchedPathEnds` and `TestBranchedMovesFirstKept`.
  - `acpserver`: `TestCloneKeepsLabels`, `TestForkDropsOffPathLabels`, `TestCloneMovesFirstKeptPastLabel` and `TestPromptFollowsTheLeaf`.
- **C3: every new check, seen failing on a scratch copy.** There were 20 mutations, each asserted to land exactly once. 19 failed as expected on the first run. One, the fork copying the leaf's path, was a broken experiment: it left a variable unused, and the copy did not build. It was rewritten to fork from the leaf and failed as expected; no run then failed by a build error.
  - the tree: no parent stamped; `Branch` not moving; `ResetLeaf` a no-op; a load's leaf on the first entry; `Add` not moving; a summary always from `root`; an unknown `from` accepted;
  - labels: an empty label kept; a label set again moved last;
  - branching: no re-chaining; the first kept entry not moved; off-path labels kept; a label's timestamp lost;
  - acpserver: `/clone` and `/fork` copying paths, not `Branched`; `append` ignoring the leaf; the state taken from the last entry; `get_entries`' leaf the last entry; no `Add`; `restore` with an empty tree.
- **The gates:** on Go 1.27.2, as F1d-2's deviation 4 sets them, all on the first run:
  - lint printed `0 issues.` for linux, darwin and windows;
  - Windows `go test ./...` had no `FAIL` or `panic:`;
  - `CGO_ENABLED=1 go test -race ./...` exited 0 with no race reported, natively on Windows and on a second amd64 Linux machine;
  - Windows and WSL `make preflight` each printed `preflight passed`, WSL's without a race run.

**What the entry predicted wrongly:**

1. **Its `BranchSummary` could not refuse an unknown entry,** and moved the leaf before the write.
2. **Its tree offered no entries in file order,** which the replay state needs, and no safety for concurrent readers.
3. **Its files table left out `session/session.go`,** and named `acpserver/prompt.go`, which needed no change.
4. **Its tests left three behaviours unpinned:** a moved leaf followed by a prompt, `get_entries`' leaf, and a label's place when set again. The tests are added or tightened above.

**2026-10-10 — F3 made executable** (the owner's decisions of 2026-10-10: four sub-phases; the prompt kept in `custom{gobble.prompt}`; nothing that needs trust loaded before F6; `@path` expanded by the agent, with `image` false; the Pi bridge left to F10; nested `AGENTS.md` once per session; no `docs` section yet; `go.yaml.in/yaml/v3` for frontmatter).

F3 is steps 1–5 and their accept list, with the 2026-10-01 amendment's F3 changes, choices 3 and 8 of the 2026-10-07 amendment, and 0012-MADR D6 and D9's notes. 0005-MADR's amendment of this date records the decisions that change its rows.

**The facts,** checked on 2026-10-10 by a script in the session scratchpad (`q319_claims_f3.py`): 41 claims, all passing. With the check inverted, all 41 failed. Pi paths are under `packages/coding-agent/src/core/` unless named otherwise.

- **gobble today:**
  - The system prompt is one constant paragraph, built from the cwd and the tool names (`acpserver/prompt.go:44`), and rebuilt at every prompt (`acpserver/agent.go:410`).
  - A prompt's user message keeps its text blocks only (`acpserver/agent.go:621`), and `initialize` advertises no prompt capabilities.
  - The CLI rejects `--system-prompt`, `--append-system-prompt` and `--no-context-files` as not yet available (`internal/cli/flags.go:63`). It already inlines `@path` arguments as `<file>` blocks (`internal/cli/input.go:75`).
  - `prompt` and `skill` are placeholders (`prompt/doc.go:4`, `skill/doc.go:4`). A command name is `[A-Za-z0-9][A-Za-z0-9_-]*` (`command/doc.go:6`), so `/skill:<name>` runs as prompt text (0008-MADR D19 item 7, `:707`).
  - `go.mod` has no YAML module.
- **Pi's context files:**
  - five candidates, of which the first found in a directory wins (`resource-loader.ts:185`);
  - the agent directory's file first, then each directory from the filesystem root down to the cwd (`:232-270`);
  - a nested worktree shadows the main repository's file of the same name (`:214-230`);
  - each file is rendered as `<project_instructions path="…">` (`system-prompt.ts:72-79`).
- **Pi's `SYSTEM.md` and `APPEND_SYSTEM.md`:** the trusted project copy, else the agent directory's (`resource-loader.ts:1201-1227`). `SYSTEM.md` replaces the preamble and drops `tools`, `rules` and `docs`. `APPEND_SYSTEM.md` is the `addendum` section (`system-prompt.ts:143-163`).
- **Pi's sections:**
  - the order is `preamble`, `tools`, `rules`, `docs`, `addendum`, `project_context`, `skills`, `cwd`, then custom ones (`system-prompt.ts:121-180`);
  - every section but the preamble is wrapped as `<name>…</name>` (`:177`), and names match `^[a-z][a-z0-9_-]*$` (`:52`);
  - a patch holds the changed sections, `null` for a removed one (`:204-216`);
  - a mid-conversation update is framed `Updated system prompt section "<name>":` or `Removed system prompt section "<name>".` (`ai/src/utils/text.ts:35-36`);
  - a provider without mid-conversation system messages gets every patch folded into the leading prompt (`ai/src/utils/transcript.ts:108-120`).
- **Pi's skills** (`skills.ts`):
  - a name is at most 64 characters, and a description at most 1024;
  - a collision is a diagnostic, and the first skill found wins;
  - `disable-model-invocation` keeps a skill out of the prompt;
  - the list is `<available_skills>`.
  - `/skill:<name>` sends the body in a `<skill name="…" location="…">` block (`agent-session.ts:2062-2086`).
  - Project `.agents/skills` is searched from the cwd up to the repository root (`package-manager.ts:468-487`).
- **Pi's templates** (`prompt-templates.ts`):
  - arguments split on whitespace, with `"` and `'` quotes and no escapes (`:25-56`);
  - the substitution grammar is one regular expression (`:75`);
  - a template without a description takes its first non-empty line, cut to 60 characters (`:131-140`).
- **Pi has no ACP.** Its `resource_link` handling is for MCP tool results (`coding-agent/src/extensions/mcp/tools.ts:169`), and its `@file` is a CLI argument, inlined as a `<file>` block (`coding-agent/src/cli/file-processor.ts`).

**The owner's decisions, 2026-10-10:**

1. **Four sub-phases,** each one commit with its own record and gates:
   - **F3-1:** context files, `SYSTEM.md` and `APPEND_SYSTEM.md`, the sectioned prompt with its frozen baseline and patches, the `tools` section, the three CLI flags, and nested `AGENTS.md` on read;
   - **F3-2:** skills, `/skill:<name>`, and `go.yaml.in/yaml/v3`;
   - **F3-3:** prompt templates, and the fuzz target;
   - **F3-4:** `@path` in prompt text, resource blocks, and the prompt capabilities.
2. **The prompt is kept in `custom{gobble.prompt}` entries,** a baseline and patches (0005-MADR, amendment of this date, item 1).
3. **Nothing that needs trust loads before F6.** It is found, and a note names it.
4. **The agent expands `@path`.** `embeddedContext` is true, and `image` false until `llm` carries images.
5. **The Pi bridge is F10's.**
6. **A nested `AGENTS.md` is added once per session.**
7. **No `docs` section** until a release installs documentation.
8. **`go.yaml.in/yaml/v3`** v3.0.5 parses frontmatter.

**F3-1 — context files, the sectioned prompt, and its frozen baseline.**

Choices made within the wording:

- **`prompt`** (the package, beta):
  - **Sections:** `Section{Name, Text string}`, with `Text` stored as Pi stores it: the preamble bare, every other section wrapped as `<name>\n…\n</name>`.
    - `Build(Options) []Section` gives Pi's order, without `docs`: `preamble`, `tools`, `rules`, `addendum`, `project_context`, `skills`, `cwd`.
    - `Text([]Section) string` joins the sections with blank lines, as `getSystemMessageText` does.
  - **The preamble** is gobble's own sentence, `You are gobble, a coding agent. You help the user with software engineering tasks in their project.`, or the custom prompt. With a custom prompt, `tools` and `rules` are left out, as Pi leaves them out.
  - ~~**`tools`:**~~
    - ~~one line per tool, `- <name>: <the first sentence of its description>`;~~
    - ~~then 0012-MADR D6's three cross-tool rules: prefer move, delete, copy and mkdir to shell commands; read a file before you edit it; a delete always asks;~~
    - ~~when anything is deferred, the line `tool_search can find more tools for: <families, comma-separated>.`~~
    - ~~Plan mode's exit tool is not listed, so a mode switch does not patch the prompt.~~
    - *(F3-1 deviation 2, 2026-10-10: no per-tool lines, so the sections need no MCP wait.)*
  - **`tools`,** built from the agent's own tools, not the MCP servers':
    - 0012-MADR D6's three cross-tool rules: prefer move, delete, copy and mkdir to shell commands; read a file before you edit it; a delete always asks;
    - when anything is deferred, the line `tool_search can find more tools for: <families, comma-separated>.`;
    - then Pi's closing line, worded for a list it no longer follows: `Each request describes the tools you can call; you may also have tools from MCP servers and the project.`
  - **`rules`:**
    - Pi's shell rule, when bash or powershell is offered and grep, find and `tree` are not (`system-prompt.ts:101-109`, with `tree` for `ls`);
    - then, in order: `A call to a tool that changes something may be declined by the user; then do not retry it, and say what you would have done.`; `Commands run on <GOOS>.`; Pi's `Be concise in your responses` and `Show file paths clearly when working with files`; and `Answer in Markdown.`
  - **`addendum`** is the append text. **`project_context`** is Pi's rendering of the context files. **`skills`** is F3-2's, and empty until then. **`cwd`** is the working directory with `/` for `\`.
  - **Patches:** `Patch{Name string; Text *string}`, where nil `Text` removes the section.
    - `Diff(prev, cur []Section) []Patch` lists changed and new sections in `cur`'s order, then removed ones in `prev`'s.
    - `Apply(base []Section, patches []Patch) []Section` replaces a section in its place, adds a new one at the end, and removes on nil, as Pi's replay does.
    - `Update([]Patch) string` is Pi's framing.
  - **Context files:** `ContextFiles(configDir, cwd string) ([]File, []Diagnostic)` follows Pi's walk:
    - the five candidate names, first per directory;
    - the config directory's file, then the filesystem root down to the cwd;
    - duplicate paths dropped, and a nested worktree's shadowing applied;
    - a byte-order mark stripped;
    - an unreadable file a diagnostic, not an error.
  - **System files:** `SystemFiles(configDir, cwd string) (system, appended string, waiting []string)` reads `SYSTEM.md` and `APPEND_SYSTEM.md` from the config directory. It names the project's `.gobble/SYSTEM.md` and `.gobble/APPEND_SYSTEM.md` in `waiting` when they exist.
  - **Nested files:** `Nested(cwd, path string) []File` finds the context files in each directory below the cwd down to `path`'s, root-most first. The cwd's own files and its ancestors' are already in `project_context`.
- **`session`** gains `CustomPrompt = "gobble.prompt"`, and `CustomContext = "gobble.context"`.
  - A baseline's data is `{"baseline":[{"name","text"},…]}`. A patch's is `{"patch":[{"name","text"|null},…]}`. Arrays keep the order, which a JSON object would not.
  - A context entry's data is `{"paths":[…]}`.
- **`compaction`:** `Project` gives a `gobble.prompt` patch entry one user message, `prompt.Update` of its patch. A baseline entry gives nothing, since it is the system prompt.
- **`tool`:**
  - `Env` gains `Instructions func(path string) []Instruction`, with `Instruction{Path, Text string}`;
  - `Result` gains `Instructions []string`, the paths added.
  - read, after a text read inside the cwd, appends each instruction's text to its result and names them in `Result.Instructions`.
- **acpserver:**
  - **`Options` gains `Prompt PromptOptions{ConfigDir, System, Append string; NoContextFiles bool}`.** `System` and `Append` are the texts the CLI resolved; each, when set, replaces its file's text, as Pi's flags do (`resource-loader.ts:645-655`).
  - **At each prompt:**
    - the sections are built from the files read then;
    - a session with no baseline on its path gets one, written before the user's message;
    - otherwise the current sections are the latest baseline with its later patches applied, and a difference is written as a patch before the user's message. Its update message joins this turn's history.
  - **The system prompt sent** is `prompt.Text` of the latest baseline. It replaces `systemPrompt` and `systemPromptOf`, and the estimates use it.
  - ~~**`/compact`** writes a baseline right after its compaction entry.~~ *(F3-1 deviation 3, 2026-10-10: before it, so the compaction stays last.)*
  - **`/compact`** writes a baseline right before its compaction entry, so the compaction stays the path's last entry, as `compaction.Prepare`'s "Already compacted" check reads it.
  - **Fork before a session's first message** is refused when the path before the message holds no user or assistant message, the condition lazy files need, rather than when the message has no parent.
  - **A diagnostic** (an unreadable file, a file waiting for trust) is logged once per session through `Options.Logger`.
  - **`Env.Instructions`** returns the nested files not already added. An added file is one whose path a `gobble.context` entry names among the context entries of the session's path, so a compaction that drops the entry drops the file with it. The writer appends `custom{gobble.context, {paths}}` after a result that names any.
- **internal/cli:**
  - `--system-prompt` and `--append-system-prompt` take text or a path, read as a file when one exists there, as Pi's `resolvePromptInput` does (`resource-loader.ts:167-182`);
  - `--no-context-files` sets `NoContextFiles`;
  - the three leave `laterFlags`.

**F3-1's files:**

| | Files |
| :--- | :--- |
| new | `prompt/sections.go`, `prompt/context.go`, `prompt/sections_test.go`, `prompt/context_test.go`, `prompt/testdata/` (a fixture project and `system.golden`); `acpserver/sysprompt.go`, `acpserver/sysprompt_test.go` |
| changed | `prompt/doc.go`; `session/session.go`; `compaction/estimate.go`, `compaction/project_test.go`; `tool/tool.go`, `tool/tool_test.go`; `tool/builtin/read.go`, `tool/builtin/builtin_test.go`; `acpserver/agent.go`, `acpserver/prompt.go`, `acpserver/compact.go`, `acpserver/commands.go`, `acpserver/history.go`; `internal/cli/agent.go`, `internal/cli/flags.go`, and their tests; `docs/architecture.md`; `README.md` |

*(2026-10-10, F3-1 deviation 1: `internal/cli/chat.go` and `internal/cli/main.go` join the changed files.)*

*(2026-10-10, F3-1 deviation 4: `acpserver/mcp_test.go`, `acpserver/toolload_test.go`, `acpserver/tree_test.go`, `acpserver/testdata/session.golden` and `internal/cli/sessions_test.go` join the changed files.)*

*(2026-10-10, F3-1 deviation 5: `acpserver/agent_test.go` joins the changed files.)*

**F3-1's tests,** each seen failing first on a scratch copy:

- **`prompt`:**
  - **The golden prompt:** a fixture project, with a config directory's `AGENTS.md`, a root `AGENTS.override.md` beside an `AGENTS.md`, a `CLAUDE.md` in a subdirectory, and an empty directory between them, builds `testdata/system.golden`. This is F3's own accept line.
  - **The walk:** a nested worktree's shadowing, and the five candidates' precedence.
  - **`SYSTEM.md`** replaces the preamble and drops `tools` and `rules`; **`APPEND_SYSTEM.md`** is `addendum`; a project copy waits.
  - **`Diff` and `Apply`:** a replaced section keeps its place, a new one goes last, a removed one is nil; Apply of Diff is the new sections.
  - **`Update`** gives Pi's two framings.
- **acpserver, through `acptest`:**
  - a session's first prompt writes a baseline, and the request's system message is its text;
  - after an `AGENTS.md` edit, the next prompt writes a patch, the system message is unchanged, and the history carries the update message;
  - a resumed session sends the same system prompt;
  - `/compact` writes a new baseline holding the patch;
  - a read below the cwd adds its nested `AGENTS.md` once, a second read adds nothing, and a resumed session adds nothing either.
- **internal/cli:** the three flags reach the agent, and a text value and a file value both work.

**F3-1's deviations, 2026-10-10,** found by reading its files before any code was written (1, 2), at its first test run (3, 4) and at its first gate run (5), and decided by the owner:

1. **The CLI flags reach the agent through two more files.**
   - **Found:** `ChatCmd.Run` sets the run environment from the flags (`internal/cli/chat.go:32`), and the environment is declared in `internal/cli/main.go:130`. The files table names `agent.go` and `flags.go` only.
   - **Decided:** `internal/cli/main.go` gains a field for the prompt options, and `chat.go` sets it beside the tool choice. Both join F3-1's files.
2. **The `tools` section lists no tools.**
   - **Found:** the plan writes the prompt entry before the user's message, as Pi does. Listing each tool needed the MCP servers' tools, and on a session's first prompt they wait up to the start-up limit (`acpserver/mcp.go:133-148`). That would move the wait before the user's message is recorded (`acpserver/agent.go:371`). Writing the entry after the message instead makes a live turn and its replay disagree: `agent.Run` takes the prompt as the turn's last message, so the live update would come before the prompt and the replayed one after it.
   - **Decided:** the section holds D6's rules, the deferred categories and a closing line, built from the agent's own tools. The provider already receives every tool's spec. The sections need no MCP wait, so the prompt entry is written before the user's message, and the user's message is still recorded before the wait. No file is added.

3. **Two defects of the new entries.**
   - **Found:** at the first test run, after the code was written:
     - a fork before a session's first message went ahead, because `forkTarget` refuses only a message with no parent (`acpserver/commands.go`), and the first message's parent is now the baseline. The fork would write a session holding only a baseline, which lazy files never create, then switch to it;
     - a second `/compact` answered `Nothing to compact` instead of Pi's `Already compacted`, because `compaction.Prepare` checks whether the path's last entry is a compaction (`compaction/compaction.go:52`), and the plan wrote the new baseline after it.
   - **Decided:** a fork is refused when the path before the message holds no user or assistant message. `/compact` writes its baseline just before the compaction entry. Both are in `acpserver/commands.go` and `acpserver/compact.go`, already in the files.
4. **Five tests check what F3-1 changes on purpose.**
   - **Found:** at the same run:
     - `acpserver/mcp_test.go` and `acpserver/toolload_test.go`, in two tests, look for tool names in the system prompt, which deviation 2 removed;
     - `acpserver/tree_test.go` expects the prompt's parent to be the moved leaf, which is now the baseline, itself the leaf's child;
     - `internal/cli/sessions_test.go`'s `TestFork` expects the source to hold 2 entries, now 3 with the baseline;
     - `acpserver/testdata/session.golden` holds the old prompt's size in its `usage_update`.
   - **Decided:**
     - the prompt tests keep their tool and loading checks, and drop the names from their system-prompt checks, keeping "the deferred tool is not named";
     - the leaf test expects the baseline as the moved leaf's child, and the prompt as the baseline's;
     - `TestFork` counts the source's entries before and after the fork, rather than fixing them at 2;
     - the golden is regenerated with `ACPTEST_UPDATE=1`, its diff checked to be that one size.
   - The five files join F3-1's.
5. **The golden transcript depended on the host.**
   - **Found:** at the first gate run, the Linux race run failed `TestSession`: its golden records the `usage_update` size, which counts the system prompt. The rule `Commands run on <GOOS>.` made it 151 on Windows and 150 on Linux. The old prompt named the OS too, but its rounding gave 112 on both.
   - **Decided:** `PromptOptions` gains `GOOS`, the system the prompt names, where `""` is the host's. `TestSession`'s agent sets it to `linux`, so the golden is the same on every host and still checks the exact size. `acpserver/agent_test.go` joins the files.

**F3-2 — skills.**

Choices made within the wording:

- **`internal/frontmatter`:** `Parse(text) (map[string]any, string, error)`, through `go.yaml.in/yaml/v3`.
  - The text must start with `---`, and the frontmatter ends at the first `\n---`. CRLF is normalised, and the body is trimmed, as Pi's `utils/frontmatter.ts` does.
  - It is the only package that imports the module.
- **`skill`:**
  - `Skill{Name, Description, FilePath, BaseDir string; DisableModelInvocation bool}`;
  - `Load(sources []Source) ([]Skill, []Diagnostic)` follows Pi's rules:
    - a directory holding `SKILL.md` is a skill, and is not searched further;
    - dotfiles and `node_modules` are skipped, and F1c's ignore matcher applies;
    - a name falls back to the directory's;
    - Pi's warnings, in Pi's words (`skills.ts:92-127`);
    - a skill with no description is left out;
    - a path seen twice by its real path is dropped;
    - a name collision keeps the first, with Pi's diagnostic, `name "<name>" collision`.
  - `Format([]Skill) string` is Pi's `formatSkillsForPrompt` with the read tool, leaving out `disable-model-invocation` skills.
  - `Expand(text string, []Skill) (string, bool)` is Pi's `/skill:<name> [args]`: the name ends at the first space, and the body goes in Pi's `<skill>` block.
- **The order of the sources** is the one 0005-MADR's "Skill discovery order (changed)" gives:
  1. `.gobble/skills`, which waits for trust;
  2. the project's `.agents/skills`, from the cwd up to the repository root, which waits for trust;
  3. the config directory's `skills/`, loaded;
  4. `~/.agents/skills`, loaded;
  5. the Pi bridge, empty until F10.
- **acpserver:**
  - the `skills` section is `skill.Format` of the loaded skills, when read is offered;
  - a prompt starting `/skill:` is expanded before the model turn, and the expansion is the user's message, as in Pi;
  - a collision is logged once per session.
- **`--no-skills`** (0005-MADR's global flags) sets no sources.
- **`go.mod` and `go.sum`** gain `go.yaml.in/yaml/v3 v3.0.5`, in this commit.

**F3-2's files:**

| | Files |
| :--- | :--- |
| new | `internal/frontmatter/frontmatter.go`, `frontmatter_test.go`; `skill/skill.go`, `skill_test.go`, `skill/testdata/`; `acpserver/skills_test.go` |
| changed | `skill/doc.go`; `go.mod`, `go.sum`; `acpserver/agent.go`, `acpserver/sysprompt.go`; `internal/cli/agent.go`, `internal/cli/flags.go`, and their tests; `docs/architecture.md`; `README.md` |

*(2026-10-10, F3-2 deviation 1: `internal/fsx/ignore.go` and its test join the changed files.)*

*(2026-10-10, F3-2 deviation 2: `internal/fsx/walk.go` joins the changed files.)*

**F3-2's tests,** each seen failing first on a scratch copy:

- **`skill`:**
  - a skills directory with a name collision warns and loads the first skill found (F3's own accept line);
  - Pi's validation warnings, word for word;
  - no description means not loaded;
  - a duplicate real path is dropped;
  - the scan rules: a `SKILL.md` root, dotfiles, `node_modules`;
  - `Format` against Pi's text, and a hidden skill left out;
  - `Expand` with and without arguments, and an unknown name left as typed.
- **`internal/frontmatter`:** a block scalar, quoted strings, a nested map, CRLF, and no frontmatter.
- **acpserver:**
  - the `skills` section in the prompt;
  - `/skill:<name> args` sends Pi's block;
  - a project `.agents/skills` is not loaded, and its note is logged.

**F3-2's deviations, 2026-10-10,** found by reading its files before any code was written (1) and at its first gate run (2), and decided by the owner:

1. **The ignore matcher could not be used from `skill`.**
   - **Found:** Pi's skill scan reads `.gitignore`, `.ignore` and `.fdignore` in each directory it walks, each file's patterns prefixed with its directory (`skills.ts:16`, `:24-65`). `internal/fsx`'s `Ignore` takes rules only through an unexported `add` (`internal/fsx/ignore.go:60`), and its walker reads `.gitignore` in a repository only (`walk.go:48-55`).
   - **Decided:** `internal/fsx` exports `Ignore.Add(base, depth, text)`, with the rules and precedence `add` has. The skill scan keeps one `Ignore` per source, and adds each directory's three files as it walks, as Pi does. `internal/fsx/ignore.go` and its test join the files.
2. **`Add` beside `add` failed lint.**
   - **Found:** at the first gate run, `revive`'s `confusing-naming` rejected the exported `Ignore.Add` beside the unexported `add` it wrapped (`internal/fsx/ignore.go:67`), as two methods differing only by case, on all three systems, and with it both preflights.
   - **Decided:** the unexported `add` becomes `Add`, so there is one method, and its three callers in `internal/fsx/walk.go` and the test helper call it. `internal/fsx/walk.go` joins the files.

**F3-3 — prompt templates.**

Choices made within the wording:

- **`prompt`:**
  - `Template{Name, Description, ArgumentHint, Content, FilePath string}`;
  - `LoadTemplates(dirs []string) ([]Template, []Diagnostic)` follows Pi's rules:
    - each directory's direct `.md` children only, without dotfiles or `node_modules`;
    - the name is the file name without `.md`;
    - a frontmatter error skips the file, with a warning;
    - a missing description takes the first non-empty line, cut to 60 characters with `...`;
    - a name collision keeps the first, with `name "/<name>" collision`.
    - *(2026-10-10, F3-3 deviation 3: and each directory's `.gitignore`, `.ignore` and `.fdignore` apply, as Pi's `collectAutoPromptEntries` applies them.)*
  - `ParseArgs(s string) []string` is Pi's `parseCommandArgs`. Whitespace is Go's `unicode.IsSpace` and U+FEFF, which is JavaScript's `\s`.
    *(2026-10-10, F3-3 deviation 4: ~~Go's `unicode.IsSpace` and U+FEFF~~ is not JavaScript's `\s`, which leaves out U+0085. Whitespace is JavaScript's own set, from `internal/frontmatter`.)*
  - `Substitute(content string, args []string) string` is Pi's `substituteArgs`, with the same regular expression and no recursive substitution.
  - `ExpandTemplate(text string, []Template) (string, bool)` is Pi's `expandPromptTemplate`.
- **The directories:** `.gobble/prompts`, which waits for trust; the config directory's `prompts/`, loaded; the Pi bridge, empty until F10.
- **acpserver:**
  - a command that no built-in handles, and a template matches, is expanded and run as a model prompt, so a built-in wins over a template of its name, with Pi's collision message logged;
    *(2026-10-10, F3-3 deviation 1: Pi has ~~a collision message~~ none for this. gobble logs Pi's template-collision wording, `name "/<name>" collision`, naming the built-in as the winner, once per session, and does not advertise the template.)*
  - *(2026-10-10, F3-3 deviation 2: a queued steering or follow-up message is expanded as it is queued, a skill and then a template, as Pi expands it.)*
  - `available_commands_update` adds each template whose name is a command name (`command/doc.go:6`), with its description, and its `argument-hint` as the input hint;
  - the set is the one found when the session starts.

**F3-3's files:**

| | Files |
| :--- | :--- |
| new | `prompt/template.go`, `prompt/template_test.go`, `prompt/template_fuzz_test.go`, `prompt/testdata/prompts/`; `acpserver/templates_test.go` |
| changed | `acpserver/agent.go`, `acpserver/commands.go`, `acpserver/sysprompt.go`; `docs/architecture.md`; `README.md` |

*(2026-10-10, F3-3 deviation 2: `acpserver/extensions.go` and `acpserver/extensions_test.go` join the changed files.)*

*(2026-10-10, F3-3 deviation 4: `internal/frontmatter/frontmatter.go`, `frontmatter_test.go`, `skill/skill.go` and `skill/skill_test.go` join the changed files.)*

*(2026-10-10, F3-3 deviation 6: `prompt/context.go` joins the changed files.)*

**F3-3's tests,** each seen failing first on a scratch copy:

- **The table tests** (F3's own accept line):
  - `docs/prompt-templates.md`'s example, `/review`, `/review concurrency` and `/review "API compatibility"`, with the outputs Pi's code gives;
  - every case of Pi's `test/prompt-templates.test.ts`, with its expected values, quoting and defaults among them.
- **`FuzzTemplate`:** `ParseArgs` and `Substitute` never panic. Content without `$` is returned unchanged, and an argument's own `$1` is never substituted. Its gate is `go test -run '^$' -fuzz FuzzTemplate -fuzztime 60s ./prompt`.
- **acpserver:**
  - a template runs as a prompt, with its arguments;
  - it is advertised with its hint;
  - a built-in of the same name wins;
  - a project template is not loaded.

*(2026-10-10, F3-3 deviation 5: Pi's invalid-YAML case is ported with yaml.v3's own message in place of Pi's `line 1, column 14`.)*

**F3-3's deviations, 2026-10-10,** found by reading its files and Pi's before any code was written (1 to 5) and while its code was written (6), and decided by the owner:

1. **Pi has no message for a template a built-in shadows.**
   - **Found:** Pi's one template collision is between two templates, `name "/<name>" collision` (`resource-loader.ts:1148-1170`). A template with a built-in's name is logged by nothing; Pi's autocomplete still lists it, filtering only extension commands (`interactive-mode.ts:750-761`), and the built-in runs.
   - **Decided:** gobble logs Pi's template-collision wording, naming the built-in as the winner, once per session, and does not advertise the template, so the advertised commands stay ones that run. No files change.
2. **A queued message was never expanded.**
   - **Found:** Pi expands `/skill:` and templates in a message as it queues it (`agent-session.ts:1922-1924`, `:2106-2107`). gobble's `enqueue` queues the text as typed (`acpserver/extensions.go:146-158`), so a queued `/skill:<name>`, since F3-2, and a queued template reach the model unexpanded.
   - **Decided:** `enqueue` expands the text as Pi does, a skill and then a template, before it queues it. `acpserver/extensions.go` and its test join the files.
3. **The plan's scan rules left out Pi's ignore files.**
   - **Found:** Pi's `collectAutoPromptEntries` applies the prompts directory's `.gitignore`, `.ignore` and `.fdignore` (`package-manager.ts:489-516`, `:261-279`).
   - **Decided:** `LoadTemplates` applies them, through F3-2's `fsx.Ignore.Add`. No files change.
4. **JavaScript's whitespace is not Go's.**
   - **Found:** JavaScript's `\s` and `.trim()` include U+FEFF and leave out U+0085; Go's `unicode.IsSpace` and `strings.TrimSpace` do the reverse. The entry named Go's set with U+FEFF, which keeps U+0085. F3-2 trims with `strings.TrimSpace` where Pi trims with `.trim()` (`skill/skill.go:184`, `:262`, `:276`; `internal/frontmatter/frontmatter.go:35`), read in the committed files.
   - **Decided:** `internal/frontmatter` exports JavaScript's whitespace, `IsSpace` and `Trim`, beside the parser that already serves both packages, and the template code and F3-2's four trims use it, each with a test. `internal/frontmatter/frontmatter.go`, `frontmatter_test.go`, `skill/skill.go` and `skill/skill_test.go` join the files.
5. **Pi's invalid-YAML message is its library's.**
   - **Found:** Pi's test expects `line 1, column 14` in the warning (`test/prompt-templates.test.ts:626-652`), the wording of JavaScript's `yaml`. `go.yaml.in/yaml/v3` v3.0.5, the library 0005-MADR's amendment of 2026-10-10 chose (item 7), gives `yaml: mapping values are not allowed in this context` for that frontmatter, with no position, as a probe outside the repository showed.
   - **Decided:** the case asserts all else Pi's does, a warning with the file's path, the file skipped and the valid sibling loaded, and yaml.v3's message exactly. No files change.
6. **A template collision's winner had nowhere to go.**
   - **Found:** `LoadTemplates` returns `prompt.Diagnostic`, which holds a path and a message only (`prompt/context.go:18-20`). Pi's collision carries the winner's path beside its message (`resource-loader.ts:1155-1165`), and F3-2's skill diagnostics log theirs.
   - **Decided:** `prompt.Diagnostic` gains `Winner`, the file or built-in a collision kept, a compatible change; the session log names it, for a template collision and a built-in's alike. `prompt/context.go` joins the files. The queued-message test lives in `acpserver/templates_test.go`, so `acpserver/extensions_test.go`, which deviation 2 named, does not change.

**F3-4 — `@path`, resource blocks, and the prompt capabilities.**

Choices made within the wording:

- **acpserver's user message** is built from all of a prompt's blocks, in the order Pi's CLI uses: the files, then the text.
  - **`@path` in text:**
    - *(2026-10-10, F3-4 deviation 1: in ~~every prompt's text~~ ~~every prompt's text but the CLI's composed first prompt, which carries `_meta.gobble.mentions: false` because the CLI expanded its own `@path` words~~ every text block but one marked `_meta.gobble.mentions: false`, as the CLI marks the stdin and the files it attached; and in a queued message, deviation 2.)*
    - an `@` at the start or after whitespace, followed by `"…"` or by non-space characters;
    - trailing `.,;:!?)]}` is dropped when the name without it is a file and the name with it is not;
    - resolved against the cwd, and confined by `fsx` to the roots;
    - expanded only when it names an existing regular file of valid UTF-8 with no NUL byte; anything else is left as typed;
    - each file once per prompt, as `<file name="<absolute path>">\n<text>\n</file>`.
    - The text keeps the `@path` as typed.
  - **An embedded text resource** is `<file name="<uri>">…</file>`. A blob is `[binary resource <uri> (<mime type>) omitted]`.
  - **A `resource_link`** is `<name>: <uri>`, as Pi renders a link in MCP content (`packages/mcp/src/protocol/content.ts:78-102`), for the model to read with its tools.
  - **An image block** is left out, as now.
- **The user entry** holds the expanded message, so a resumed session gives the model what it had.
- **`initialize`** advertises `promptCapabilities{embeddedContext: true, image: false, audio: false}`.

**F3-4's files:**

| | Files |
| :--- | :--- |
| new | `acpserver/mentions.go`, `acpserver/mentions_test.go` |
| changed | `acpserver/agent.go`, `acpserver/agent_test.go`; `acpserver/testdata/session.golden` (the capabilities); `docs/architecture.md`; `README.md` |

*(2026-10-10, F3-4 deviation 1: `acpclient/conn.go`, `acpclient/conn_test.go`, `internal/cli/input.go`, `internal/cli/input_test.go` and `internal/cli/print.go` join the changed files.)*

*(2026-10-10, F3-4 deviation 2: `acpserver/extensions.go` and `acpserver/extensions_test.go` join the changed files.)*

*(2026-10-10, F3-4 deviation 5: `internal/cli/main_test.go` joins the changed files.)*

**F3-4's tests,** each seen failing first on a scratch copy:

- `@path`, `@"a b.txt"` and `@path.` are expanded; an `@` inside a word, a missing file, a directory, a binary file, and a path outside the roots are left as typed; a file named twice is expanded once.
- An embedded text resource, a blob and a `resource_link` give their texts.
- `initialize`'s capabilities are as listed.
- *(2026-10-10, F3-4 deviations 1 and 2:)* ~~a prompt with `_meta.gobble.mentions: false` is not expanded, and the CLI's composed first prompt carries it~~ a text block marked `_meta.gobble.mentions: false` is not scanned, and the CLI marks its stdin and attached files but not its words, so a quoted `@path` among them is expanded; a queued message is expanded.

**F3-4's deviations, 2026-10-10,** found by reading its files and Pi's before any code was written (1 and 2), by its first test (3), while closing 0014-PLAN (4), and at its first test run after (5), and decided by the owner:

1. **The CLI's own files would be scanned for `@path`.**
   - **Found:** the CLI expands its first prompt's `@path` words itself (0008-MADR D4): after stdin, each text file is wrapped as `<file name="ABS">` inside the one text block (`internal/cli/input.go:31-80`). An agent pass over every prompt's text would scan those files' contents and stdin, so a decorator such as `@property`, or `see @README.md` in an attached document, would attach further workspace files no one named. Pi has no agent-side pass; its CLI expands only its arguments (`cli/file-processor.ts`).
   - **Decided:** ~~the CLI's composed first prompt carries `_meta.gobble.mentions: false`, through a new `acpclient.Prompt.NoMentions`, and the agent does not expand `@path` in a prompt that carries it. The line session's typed lines, and every other client's prompts, are expanded as the entry says.~~ *(Revisited the same day, before code: the CLI counts a word as `@path` only when the whole argument starts with `@` (`internal/cli/flags.go:149`), so `gobble -p "explain @main.go"` is one plain word, and a per-prompt mark would leave it unexpanded by either side.)* The mark is per text block: the CLI sends its first prompt's stdin and attached files as text blocks before its words, in 0008-MADR D4's order, each marked `_meta.gobble.mentions: false` through a new `acpclient.Prompt.Context`, and the agent scans only the text blocks without the mark. The words, the line session's typed lines, and every other client's prompts are expanded as the entry says. Commands and expansions read the blocks joined by a blank line, as before; the model gets one text part per block, as it does for any client that sends several. 0005-MADR's amendment of 2026-10-10, item 2, is amended to match. `acpclient/conn.go`, `acpclient/conn_test.go`, `internal/cli/input.go`, `internal/cli/input_test.go` and `internal/cli/print.go` join the files.
2. **A queued message had no `@path` pass.**
   - **Found:** since F3-3's deviation 2, a queued steering or follow-up message is expanded as a prompt is, a skill and then a template (`acpserver/extensions.go`, `enqueue`); the entry expands `@path` in prompts only. Pi has no `@path` pass to follow.
   - **Decided:** the queued text is given its `<file>` blocks as it is queued, as a prompt's text is. `acpserver/extensions.go` and `acpserver/extensions_test.go` join the files.
3. **The ACP SDK cannot send a text block's `_meta`.**
   - **Found:** writing deviation 1's per-block mark, a test sending a text block marked `_meta.gobble.mentions: false` through `acpclient` showed the agent receiving the block without it. The fork's `ContentBlock.MarshalJSON` rebuilds each variant from a fixed key list and never writes `_meta` or `annotations` (`acp-go-sdk@v0.13.6-mcr.2` `types_gen.go:1473-1600`).
   - **Decided:** the owner directed that gobble move from the fork to their rewrite, `github.com/maccavelli/acp-go-sdk2` at `v0.14.0`, which keeps both. That move is its own record, [0014-MADR-move-to-acp-go-sdk2.md](0014-MADR-move-to-acp-go-sdk2.md), and F3-4 waits until [0014-PLAN-move-to-acp-go-sdk2.md](0014-PLAN-move-to-acp-go-sdk2.md) has run. F3-4's code written so far, `acpclient.Prompt.Context` and its test, is set aside by 0014-PLAN's P0 and re-applied when F3-4 resumes.
4. **The dependency table missed a direct requirement.**
   - **Found:** closing 0014-PLAN, `docs/architecture.md`'s table of `go.mod`'s direct requirements has no row for `go.yaml.in/yaml/v3`, which F3-2 made direct (`go.mod:19`, `v3.0.5`). It predates F3-4: F3-2's execution record lists the file as updated, with a row for `internal/frontmatter` but not for the module.
   - **Decided,** by the owner on 2026-10-10: F3-4 adds the row. No files join: `docs/architecture.md` is already among F3-4's.
5. **A CLI test asserted the old single block.**
   - **Found:** at F3-4's first test run, `internal/cli/main_test.go:252`, `TestPipedInputIsComposed`, asserted one text block `"text":"hi\n\nask"`; the CLI now sends the stdin marked `_meta.gobble.mentions: false`, then the words, as deviation 1 decided. The test predates F3-4. Its log assertion, 7 bytes, still holds, since `Prompt.Text` keeps the whole input.
   - **Decided:** the test asserts the two blocks exactly, the marked `hi` and the unmarked `ask`. `internal/cli/main_test.go` joins the files.

**The gates, for each sub-phase,** as F1d-2's deviation 4 sets them: lint for linux, darwin and windows; Windows `go test ./...`; `CGO_ENABLED=1 go test -race ./...` natively on Windows and on the second amd64 Linux machine; Windows and WSL `make preflight`, WSL's without a race run. F3-3 adds its fuzz run.

**How F3's accept list is met:**

- **The examples in `docs/prompt-templates.md` as table tests, including quoting and defaults,** with Pi's own test cases: F3-3. The fuzz target is `FuzzTemplate`.
- **A skills directory with a name collision warns and loads the first skill found:** F3-2.
- **A golden system prompt for a fixture project matches:** F3-1's `system.golden`.

**Deferred, named:**

- **The project's own `SYSTEM.md`, `APPEND_SYSTEM.md`, skills and prompts:** F6, with trust. F3 finds them and says so.
- **The Pi bridge's skills, prompts and context file:** F10 step 5.
- **The `docs` section:** the release that installs documentation.
- **Images in prompts:** `llm`'s image input, with the provider SDK.
- **`--skill`, `--prompt-template` and `--no-prompt-templates`:** 1.x (0005-MADR, CLI flags).
- **`gobble skills list|show|new` and `gobble prompts list`:** not in F3's steps. They wait for an amendment that places them.
- **MCP prompts as commands, and the built-in templates:** X1 steps 6 and 7.

**F3-1, 2026-10-10 — complete (staged; the owner commits).** It ran as the entry "F3 made executable" wrote it, approved by the owner's "approved to proceed to F3-1" of 2026-10-10, with five deviations decided by the owner: 1 and 2 before any code was written, 3 and 4 at its first test run, 5 at its first gate run. F3-2, F3-3 and F3-4 have not run.

- **The deviations** (recorded under F3-1's deviations above):
  1. `internal/cli/chat.go` and `main.go` join the files, to carry the prompt flags.
  2. The `tools` section lists no tools, so the sections need no MCP wait, and the prompt entry comes before the user's message.
  3. A fork is refused when nothing before its message is a conversation, and `/compact`'s baseline goes just before its compaction entry.
  4. Five tests that checked the old prompt or entry counts are updated to the new shape.
  5. `PromptOptions.GOOS` names the system in the prompt, so `TestSession`'s golden holds on every host.
- **What was built:**
  - **`prompt`:**
    - `Section`, `Patch` and `Options`; `Build`, in Pi's order without `docs`; `Text`, `Diff`, `Apply` and `Update`;
    - `BaselineData`, `PatchData` and `ParseData`, the entry data, as arrays;
    - `ContextFiles`, with Pi's five candidates, its walk and its worktree shadowing; `Nested`; `SystemFiles`, with a project's copies waiting for trust; `Instruction`, Pi's `<project_instructions>` block.
  - **`session`:** `CustomPrompt` and `CustomContext`.
  - **`compaction`:** a `gobble.prompt` patch projects as one user message, its update text.
  - **`tool`:** `Env.Instructions`, `Instruction` and `Result.Instructions`. read appends what `Env.Instructions` gives after a text read, and names the files.
  - **acpserver:**
    - `PromptOptions`;
    - `writePrompt` writes the baseline or patch before the user's message; its update message joins the history, live and on replay alike;
    - the system prompt sent is the latest baseline's;
    - `/compact` rebases;
    - the estimates use `systemText`;
    - diagnostics are logged once per session;
    - `Env.Instructions` gives the nested files the path's context does not hold, and the writer records a `gobble.context` entry;
    - the old constant prompt is gone.
  - **internal/cli:** `--system-prompt`, `--append-system-prompt` and `--no-context-files` reach the agent, a value read as a file when one is there; the config directory goes with them, for `gobble acp` too.
  - **Updated:** `docs/architecture.md`'s prompt and acpserver rows; `README.md`.
- **Choices of mine, within the entry's wording:**
  - **D6's three rules appear only for the tools there are:** the shell rule only with a file operation and a shell, read-before-edit only with both, the delete rule only with delete.
  - **Pi's closing line is reworded** for a section that lists no tools: `Each request describes the tools you can call; you may also have tools from MCP servers and the project.`
  - **The import is aliased `sysprompt` in `acpserver`,** where a test helper is named `prompt`.
  - **A flag's relative path resolves against `--cwd`,** as an `@path` does.
  - **The golden test copies the fixture into a temporary directory,** so the walk to the filesystem root meets no file of this repository. It writes the fixture's root as `<root>`, with `/` separators, so the golden holds on every system.
- **The tests:**
  - `prompt`: `TestGoldenSystemPrompt` (F3's accept line), `TestBuildCustomAndAppend`, `TestDiffApply`, `TestEntryData`, `TestUpdate`, `TestCandidates`, `TestWorktreeShadowing`, `TestConfigFirstAndUnreadable`, `TestNested` and `TestSystemFiles`.
  - `compaction`: `TestProjectPromptEntries`. `tool/builtin`: `TestReadInstructions`.
  - acpserver: `TestPromptBaselineAndPatch`, `TestCompactRebases`, `TestNestedInstructionsOnce` and `TestPromptOptionsAndTrust`.
  - internal/cli: `TestPromptFlags`.
  - `TestConfigFirstAndUnreadable`'s refusal had to match either spelling of `AGENTS.md`: on a case-insensitive file system `AGENTS.MD` is the same file, as it is for Pi.
- **C3: every new check, seen failing on a scratch copy.** There were 26 mutations, each asserted to land exactly once, and all 26 failed as expected on the first run, none by a build failure:
  - the walk: the override not first; cwd first; no config file; no shadowing; the byte-order mark kept;
  - the sections: a custom prompt keeping the rules; backslashes in `cwd`; no delete rule;
  - patches: no removals; a replaced section moved; Pi's wording changed; a removal's `null` dropped;
  - acpserver: no patch; the system prompt following the patch; the update not in the live history; no rebase; the patch not projected; held files ignored; no `gobble.context` entry; context files forced; the trust note logged every time; the `System` option ignored; a fork refused only by its parent; the baseline after the compaction;
  - read: no instructions appended; the CLI: a flag's file not read.
- **The gates,** on Go 1.27.2, as F1d-2's deviation 4 sets them.
  - **The first run failed** lint on all three systems, with four findings in the new code, and with it both preflights. The Linux race run failed `TestSession` (deviation 5). Windows `go test` and race passed.
  - **The fixes:** a `gosec` G703 on reading the user's own `.git` file is excused with its reason; `strings.SplitSeq`; a `strings.Builder` for read's appended instructions; a constant log message, its reason an attribute. The negatives' read case moved to the builder's line, and all 26 were run again.
  - **On the second run:**
    - lint printed `0 issues.` for linux, darwin and windows;
    - Windows `go test ./...` had no `FAIL` or `panic:`;
    - `CGO_ENABLED=1 go test -race ./...` exited 0 with no race reported, natively on Windows and on a second amd64 Linux machine;
    - Windows and WSL `make preflight` each printed `preflight passed`, WSL's without a race run.

**What the entry predicted wrongly:**

1. **Its files table left out `internal/cli/chat.go` and `main.go`,** through which the CLI's flags reach the agent.
2. **Its `tools` section and its entry order could not both hold:** listing every tool needed the MCP wait before the user's message, and writing the entry after the message would have split a live turn from its replay.
3. **It did not foresee that a new entry before the first message changes two rules** that read a session's shape: fork's "nothing comes before" and compaction's "already compacted".
4. **It did not name the tests that checked the old prompt and entry counts.**
5. **It put the host's system into the prompt** without seeing that a golden counts the prompt's size.

**F3-2, 2026-10-10 — complete (staged; the owner commits).** It ran as the entry "F3 made executable" wrote it, approved by the owner's "approved to proceed to F3-2" of 2026-10-10, with two deviations decided by the owner: 1 before any code was written, 2 at its first gate run. F3-3 and F3-4 have not run.

- **The deviations** (recorded under F3-2's deviations above):
  1. `internal/fsx` exports `Ignore.Add`, so the skill scan honours `.gitignore`, `.ignore` and `.fdignore` with the one matcher.
  2. `add` became `Add`, one method, and `internal/fsx/walk.go`'s callers call it.
- **What was built:**
  - **`internal/frontmatter`:** `Parse`, Pi's `parseFrontmatter`, through `go.yaml.in/yaml/v3` v3.0.5, now a direct requirement in `go.mod`.
  - **`skill`:** `Skill`, `Source`, `Diagnostic`; `Load`, with Pi's scan, its warnings word for word, its real-path de-duplication and its first-wins collisions; `Format`; `Expand`.
  - **acpserver:** `PromptOptions.Home` and `NoSkills`; the skills from the config directory's `skills/` and `~/.agents/skills`; the `skills` section when read is offered; `/skill:` expanded into the user's message; the project's `.gobble/skills` and `.agents/skills` noted as waiting for trust.
  - **internal/cli:** `--no-skills`, and the home directory passed on.
  - **Updated:** `docs/architecture.md`'s skill row and a row for `internal/frontmatter`; `README.md`.
- **Choices of mine, within the entry's wording:**
  - **The layouts are Pi's:** `.gobble/skills` and the config directory's `skills/` scan as Pi's own directories do, a loose `.md` file a skill only at the top; both `.agents/skills` directories as Pi's `.agents` ones do, a loose file a skill only below the top (`package-manager.ts:369-448`).
  - **A loose `.md` file with no name of its own is named after its directory,** as Pi names it (`skills.ts:310-321`).
  - **A collision's diagnostic keeps Pi's words,** `name "<name>" collision`, and names the skill that won in its own field, which the log shows.
  - **The home directory's own `.agents/skills` is never a project directory,** by the real home directory even when `Home` is unset, as Pi excludes it.
  - **A directory reached twice through links is searched once,** so a link loop cannot hang the scan. Pi has no such guard.
  - **A non-mapping YAML frontmatter reads as no fields,** as Pi's casts give none.
- **The tests:**
  - `skill`: `TestLoadCollision` (F3's accept line), `TestValidation`, `TestScan`, `TestFormat` and `TestExpand`.
  - `internal/frontmatter`: `TestParse`. `internal/fsx`: `TestAdd`.
  - acpserver: `TestSkillsSection`, `TestSkillCommand` and `TestProjectSkillsWait`. internal/cli: `TestNoSkillsFlag`.
- **C3: every new check, seen failing on a scratch copy.** There were 24 mutations, each asserted to land exactly once.
  - **21 failed as expected on the first run. Three did not:**
    - one mutation, ignore files not read, left a variable unused, and the copy did not build. It was a broken experiment, rewritten to add empty rules, and then failed as expected;
    - removing the real-path de-duplication failed nothing. The scan's own directory guard already stops a link inside one source, and no test reached the same file from two sources. `TestScan` gained that case;
    - removing the CRLF normalisation failed nothing, because YAML tolerated the stray `\r`. `TestParse` gained a body with CRLF inside it, and lone CR line endings.
  - All 24 were run again, and all failed as expected, none by a build failure:
    - the scan: the last of two names winning; no real-path de-duplication; a skill's directory searched further; dotfiles searched; `node_modules` searched; one layout for both; no ignore files;
    - the skill rules: a skill with no description loaded; consecutive hyphens allowed; lengths in bytes, not JavaScript's units; a loose file named after itself;
    - output: no XML escaping; hidden skills listed; arguments untrimmed; the block's wording;
    - frontmatter: the byte-order mark kept; CRLF kept; a scalar an error;
    - acpserver: no skills section; `NoSkills` ignored; no expansion; the project's skills loaded;
    - the CLI flag not passed; `Ignore.Add` a no-op.
- **The gates,** on Go 1.27.2, as F1d-2's deviation 4 sets them.
  - **The first run failed** lint on all three systems, and with it the preflights: `revive` rejected `Add` beside `add` (deviation 2), and `revive` and `gocritic` rejected a variable named `real`, the built-in's name, now `loaded`. Windows `go test` and race passed. The run was stopped there, before the Linux race step began, and nothing was left on that machine.
  - **The fixes** moved two negatives' anchors, and all 24 were run again, failing as expected.
  - **On the second run:**
    - lint printed `0 issues.` for linux, darwin and windows;
    - Windows `go test ./...` had no `FAIL` or `panic:`;
    - `CGO_ENABLED=1 go test -race ./...` exited 0 with no race reported, natively on Windows and on a second amd64 Linux machine;
    - Windows and WSL `make preflight` each printed `preflight passed`, WSL's without a race run.

**What the entry predicted wrongly:**

1. **It named F1c's matcher for a use its API did not offer,** and that walker reads `.gitignore` only. Exporting it as a second method then failed lint, and `walk.go` had to change too.
2. **Its tests would not have caught two behaviours it asked for:** the real-path de-duplication, and the CRLF normalisation. The negatives showed it, and the tests are added.

**F3-3, 2026-10-10 — complete (staged; the owner commits).** It ran as the entry "F3 made executable" wrote it, approved by the owner's "committed and pushed, proceed" of 2026-10-10, with six deviations decided by the owner: 1 to 5 before any code was written, 6 while it was written. F3-4 has not run.

- **The deviations** (recorded under F3-3's deviations above):
  1. A template a built-in shadows is logged in Pi's template-collision wording, with the built-in as the winner, and is not advertised.
  2. A queued message is expanded as it is queued, a skill and then a template.
  3. A prompts directory's `.gitignore`, `.ignore` and `.fdignore` apply.
  4. `internal/frontmatter` exports JavaScript's whitespace, `IsSpace` and `Trim`, and F3-2's four trims use it.
  5. Pi's invalid-YAML case is ported with yaml.v3's message.
  6. `prompt.Diagnostic` gains `Winner`.
- **What was built:**
  - **`prompt`:** `Template`; `LoadTemplates`, with Pi's scan, its warnings and its first-wins collisions; `ParseArgs`; `Substitute`, one pass with Pi's pattern; `ExpandTemplate`; `Diagnostic.Winner`.
  - **`internal/frontmatter`:** `IsSpace` and `Trim`, and the body trimmed by them. **`skill`:** a description's blankness, the arguments and the body trimmed by them.
  - **acpserver:**
    - the config directory's `prompts/` loaded when a session starts, and each template with a command name advertised after the built-ins, with its argument hint;
    - a prompt expanded as Pi expands it, a skill and then a template, and any other command run as before;
    - a queued message expanded the same way;
    - a template with a built-in's name left out and logged;
    - the project's `.gobble/prompts` noted as waiting for trust.
  - **Updated:** `docs/architecture.md`'s `prompt` and `acpserver` rows; `README.md`.
- **Choices of mine, within the entry's wording:**
  - **Pi's cases were ported by a converter.** It read every literal `expect` of `substituteArgs` (80) and `parseCommandArgs` (16) from Pi's test file and wrote each as a table row, with its test's name and its line, and listed the nine whose arguments are computed, which were ported by hand. Its first run skipped two literal cases for a trailing comma it did not parse, which its listing showed, and both were then ported.
  - **A template whose name is not a command name** (`a.b.md`) runs from `/a.b`, as Pi's `expandPromptTemplate` matches any name, and is not advertised.
  - **Text before a command's `/` is dropped before expansion,** as `command.Parse` drops it, so `/review` after a space runs the template; Pi's requires the `/` first.
  - **A body without frontmatter is not trimmed,** as Pi's `extractFrontmatter` returns it, so its trailing newline reaches the model.
  - **A number too large for an int** is larger than any argument list, as JavaScript's `parseInt` makes it; a surrogate pair cut by the 60-unit description leaves U+FFFD; bytes that are not UTF-8 pass through `ParseArgs` as they came.
  - **Three lines no test could fail were left out:** Pi's `node_modules` rule, since no `.md` file has that name; trimming the arguments' leading whitespace, which `ParseArgs` skips; and a branch running a built-in before expansion, since no template a built-in shadows is kept.
  - **A shadowed template's log line** is `name "/<name>" collision (the built-in command /<name> is loaded)`; `/help` lists the built-ins only.
- **The tests:**
  - `prompt`: `TestSubstitute`, `TestSubstituteComputed`, `TestSubstituteLargeNumbers`, `TestParseArgs`, `TestParseArgsWhitespace`, `TestParseAndSubstitute`, `TestExpandTemplate`, `TestDocsExample` (F3's accept line, from `testdata/prompts/review.md`), `TestArgumentHint`, `TestInvalidFrontmatter`, `TestDescriptionFallback`, `TestTemplateScan`, `TestTemplateCollision`, and `FuzzTemplate`.
  - `internal/frontmatter`: `TestIsSpace` and a `TestParse` case. `skill`: `TestBlankDescription` and two `TestExpand` cases.
  - acpserver: `TestTemplateRuns`, `TestTemplateAdvertised`, `TestTemplatesFixedAtStart`, `TestProjectTemplatesWait` and `TestQueuedExpansion`.
- **C3: every new check, seen failing on a scratch copy.** There were 47 mutations, each asserted to land exactly once.
  - **46 failed as expected on the first run. One did not:** loading the project's `.gobble/prompts` did run its template, but the test's model had no reply to give, so the test stopped on the script's exhaustion before its own check. It was a broken experiment: the model now answers once, and the mutation then failed with the test's message.
  - All 47 were run again, and all failed as expected, none by a build failure:
    - the scan and the loader: the last of two names winning; dotfiles loaded; no ignore files; a directory read; a link not followed; the description not cut; cut in bytes; a blank line chosen; Go's trim of a line; the hint dropped; the description ignored; a file with bad YAML loaded;
    - `ParseArgs`: quotes kept; an empty argument; Go's whitespace; bytes that are not UTF-8 replaced;
    - `Substitute`: a second, recursive pass; a default ignored; `${@:0}` not read as `${@:1}`; a slice's length ignored; `$0` read as an argument; a large number read as 0; `$@` not every argument;
    - `ExpandTemplate`: Go's whitespace ending the name; a prefix matching; a template with no name matching;
    - `FuzzTemplate`'s own properties, on its seeds: an argument substituted in turn; text without `$` changed; an empty argument;
    - acpserver: a shadowed template kept; its collision not logged; the winner not shown; no expansion; the space before `/` kept; templates not advertised; any name advertised; no hint; the project's templates loaded; no trust note; the queue unexpanded; templates reloaded at each prompt; templates not kept at the start;
    - JavaScript's whitespace: Go's `IsSpace`; Go's trim of a body, of a skill's description, its arguments and its body.
- **The gates,** on Go 1.27.2, as F1d-2's deviation 4 sets them, with F3-3's fuzz run. All passed on the first run:
  - lint printed `0 issues.` for linux, darwin and windows;
  - Windows `go test ./...` had no `FAIL` or `panic:`;
  - `CGO_ENABLED=1 go test -race ./...` exited 0 with no race reported, natively on Windows and on a second amd64 Linux machine, 29 packages each, none cached;
  - Windows and WSL `make preflight` each printed `preflight passed`, WSL's without a race run;
  - `go test -run '^$' -fuzz FuzzTemplate -fuzztime 60s ./prompt` passed, 26,379,888 runs, and wrote no failing input.

**What the entry predicted wrongly:**

1. **It named a Pi message that does not exist,** for a template a built-in shadows.
2. **Its whitespace rule was wrong,** keeping U+0085, and F3-2 had trimmed with Go's rule throughout.
3. **Its scan rules left out Pi's ignore files,** and its acpserver steps the queue, which Pi expands too.
4. **One of Pi's cases cannot be ported word for word:** its YAML message is its library's.
5. **Its file list missed `prompt/context.go`,** where the diagnostic it returns is defined.

**F3-4, 2026-10-10 — complete (staged; the owner commits), and with it F3.** It ran as the entry "F3 made executable" wrote it, approved by the owner's "approved to proceed to F3-4" of 2026-10-10, paused by deviation 3 for [0014-PLAN-move-to-acp-go-sdk2.md](0014-PLAN-move-to-acp-go-sdk2.md), and resumed by the owner's "approved to proceed with F3-4, apply the patch". Five deviations were decided by the owner.

- **The deviations** (recorded under F3-4's deviations above):
  1. The CLI marks its stdin and attached files as text blocks `_meta.gobble.mentions: false`, and the agent scans only unmarked text blocks; per block, not per prompt, so a quoted `"explain @main.go"` expands.
  2. A queued message is given its files too.
  3. The ACP SDK could not send a block's `_meta`; gobble moved to acp-go-sdk2 `v0.14.0` under 0014, and F3-4's first code waited in a scratchpad patch, re-applied cleanly on `a2e22d8`.
  4. `docs/architecture.md` gains the `go.yaml.in/yaml/v3` row, at the owner's direction.
  5. `TestPipedInputIsComposed` asserts the two blocks.
- **What was built:**
  - **`acpserver/mentions.go`:** `mentionsIn`, `mentionFiles`, `mentionPath`, `fileBlock` (Pi's CLI block), `unscanned`, `promptFiles` and `resourceText`.
  - **acpserver:** the user message is the files, then the text, and the user entry holds both; a queued message gets its files ahead of its expansion; `initialize` advertises `promptCapabilities{embeddedContext: true}`.
  - **acpclient:** `Prompt.Context`, each sent as a marked text block ahead of the text.
  - **internal/cli:** `Prompt.Context` and `Prompt.Words` beside the whole `Text`, and `clientPrompt` sending them.
  - **Updated:** `docs/architecture.md`'s `acpserver` and `acpclient` rows and the dependency row; `README.md`.
- **Choices of mine, within the entry's wording:**
  - **The @path words are read from the prompt's own text,** before any skill or template expansion, so a template's or skill's body never attaches files; `/fix @a.go` attaches `a.go`.
  - **An empty file gives nothing, and a byte-order mark is dropped,** as Pi's CLI skips an empty file and strips a BOM. A file's text is kept whole, trailing newline included, as Pi's block has it.
  - **A file is read from disk through `fsx`,** not through the client's `fs/read_text_file`, as the entry says; a file is named once per prompt by its resolved path.
  - **Trailing punctuation goes one character at a time** while the name with it is no file, so `@a.txt).` finds `a.txt`; a quoted name keeps its punctuation.
  - **A blob's missing MIME type reads `unknown type`,** Pi's words (`content.ts:96`). An audio block is left out with images, since neither is advertised.
  - **The files are their own text part, ahead of the text's parts,** in Pi's order; the model gets one text part per block, as from any client that sends several.
- **The tests:**
  - acpserver: `TestMentionsIn`, `TestMentionFiles`, `TestPromptFiles`, `TestPromptCarriesFiles`, `TestQueuedMentions`, and `TestCapabilitiesAreHonest` updated; `session.golden`'s one changed line is `promptCapabilities`, `{}` to `{"embeddedContext":true}`, read on a scratch copy before the tree's golden was regenerated.
  - acpclient: `TestPromptContext`. internal/cli: `TestComposeInputContext`, and `TestPipedInputIsComposed` updated.
- **C3: every new check, seen failing on a scratch copy.** There were 25 mutations, each asserted to land once.
  - **23 failed as expected on the first run. Two were broken experiments,** each leaving a variable unused so the copy did not build: `afterSpace` in the first, `ok` in the twelfth. Both were rewritten to keep the variable in use. The runner also stopped on a test's output holding the invalid-UTF-8 file's byte, and now decodes with replacement.
  - All 25 were run again, and all failed as expected, none by a build failure:
    - the scan: an `@` inside a word; an empty quote;
    - the files: punctuation kept; a quoted name's punctuation dropped; no confinement; a file named twice; a binary file, one not UTF-8, an empty one, and a BOM kept;
    - the blocks: a marked block scanned; the mark misread; resources dropped; a blob's type blank; a link unnamed;
    - the message: files after the text; no files; the queue without files;
    - the capabilities: `embeddedContext` unadvertised; an image advertised;
    - the CLI and client: the context unsplit, unsent, and unmarked on the wire; a client block unmarked; an empty context sent.
  - **One check could not be made to fail here:** `mentionPath`'s regular-file test. A directory fails at the read anyway, and the case it guards, a named pipe that would block the read, cannot be made on Windows. It stays, for Linux and macOS.
- **The gates,** on Go 1.27.2, as F1d-2's deviation 4 sets them: all passed on the first run:
  - lint printed `0 issues.` for linux, darwin and windows;
  - Windows `go test ./...` had no `FAIL` or `panic:`;
  - `CGO_ENABLED=1 go test -race ./...` exited 0 with no race reported, natively on Windows and on a second amd64 Linux machine, 29 packages each, none cached;
  - Windows and WSL `make preflight` each printed `preflight passed`, WSL's without a race run.

**What the entry predicted wrongly:**

1. **It took the CLI's prompt as plain text,** and the CLI already attaches files inside its text, so a pass over every prompt would have scanned them (deviation 1).
2. **It assumed the SDK could carry a block's mark,** which the fork could not; the SDK moved first (deviation 3, 0014).
3. **Its files left out the queue, and a CLI test that pinned the old wire** (deviations 2 and 5).

**2026-10-10 — F4 made executable** (the owner's decisions of 2026-10-10: three sub-phases; go-llmprovider-sdk `v1.3.2`; a models.dev snapshot for the catalog; a Pi-style `auth.json`, the keyring left to 1.x; ChatGPT, Grok and Kilo logins, each behind a recorded terms check; the small-model role left to F5; Pi's `--model` rules; a one-run key read from stdin; `--offline` still refused; ACP logins as terminal auth methods).

F4 is steps 1–9 and their accept list, with choice 11 of the 2026-10-07 amendment and the 2026-10-02 amendment's F4 notes. 0005-MADR's second amendment of this date, and a note in 0003-MADR, record the decisions that change their rows.

**The facts,** checked on 2026-10-10 by a script in the session scratchpad (`q383_claims_f4.py`): 46 claims, all passing. With the check inverted, all 46 failed. SDK paths are under `go-llmprovider-sdk@v1.3.2/llmprovider/`; Pi paths under `packages/coding-agent/src/` unless named otherwise.

- **gobble today:**
  - `llm/provider` builds a provider with probes and the metadata fetch off, the client info set, and `WithRetry` around it (`llm/provider/provider.go:43-66`); errors map to `llm`'s sentinels with the status and `RetryAfter` kept (`:268-283`). Its tests use the SDK's fake, not a wire (`provider_test.go`).
  - The only provider choice is ambient: the first whose standard variable is set (`llm/provider/ambient.go:24-37`), which the CLI uses for the agent and its model list (`internal/cli/agent.go:27-31`, `:78-84`).
  - `--provider`, `--model`, `--thinking`, `--api-key` and `--offline` parse and are refused as not yet available (`internal/cli/flags.go:66-67`).
  - `llm/catalog` and `internal/auth` are placeholders (`llm/catalog/doc.go:4`, `internal/auth/doc.go:4`). `initialize` lists no auth methods, and `authenticate` and `logout` answer method-not-found (`acpserver/agent.go:233`, `:934-941`).
  - The `model` and `thought_level` config options exist (`acpserver/config.go:36-45`). `usage_update`'s size is 0 until a catalog gives one (`acpserver/prompt.go:296`).
- **go-llmprovider-sdk:** `go.mod` pins `v1.2.1` (`go.mod:16`). `v1.3.2` is tagged 2026-10-07, and gobble builds and passes its `llm`, `acpserver` and `internal/cli` tests on it unchanged, on a scratch copy.
  - Ten provider ids (`constants.go`), none streaming natively and none taking images (`item.go:19-22`). *(2026-10-11: from `v1.6.0` every one streams natively; see the Amendments entry "F4-1 re-made for go-llmprovider-sdk `v1.7.0`".)*
  - **No model has limits or cost:** the catalog package lists ids and ranks them (`catalog/models_catalog.go`), and its `Metadata` carries reasoning efforts only (`catalog/model_metadata.go`).
  - **Auth** (`auth/`): a `TokenStore` of `Load`, `Save` and `Delete` (`tokenstore.go`); `LoginBrowserOAuth` for `openai` and `grok` (`oauth_loopback.go`); `StartDeviceOAuth` for `openai`, `grok` and `kilo` (`oauth_device.go`); a file token store; no keyring. The flows' default client ids are the vendor CLIs' own (`auth/doc.go`).
  - ~~A tool has no strict flag (`provider.go:17-21`), so 0012-PLAN P7 still waits.~~ *(2026-10-11: `v1.7.0` has `Tool.Strict *bool`, and P7 follows F4-1a; see the Amendments entry "F4-1 re-made for go-llmprovider-sdk `v1.7.0`".)*
- **Pi** (`312184edb`):
  - `models.json`: `providers` only at the top level, each with `baseUrl`, `apiKey`, `api`, `headers`, `models[]` and `modelOverrides` (`core/model-config.ts:229-244`); `//` comments and trailing commas stripped (`:297`, `utils/json.ts:2-6`); a bad file is ignored whole and a bad provider falls back to the built-in, each with a diagnostic (`core/model-config.ts:288-311`, `core/model-runtime.ts:308-314`).
  - Values: `!command` and `$NAME`/`${NAME}`, resolved at request time, a command uncached with a 10 s timeout (`core/resolve-config-value.ts`).
  - Credentials, in order: the run's `--api-key`, the `auth.json` entry (which owns its provider), the `models.json` key, then the environment (`ai/src/auth/resolve.ts:46-93`). `auth.json` is 0600, locked for every write (`core/auth-storage.ts:25`, `:449-471`).
  - `--list-models [search]` prints `provider model context max-out thinking images` for providers with credentials (`cli/list-models.ts`). `--model` matches an exact id, then a substring, preferring an alias to a dated id; `:level` is split from the last colon; an unknown id under a known provider is a custom model (`core/model-resolver.ts:406-606`).
  - `pi auth` has `print-api-key`, `print-bearer-token` and `check` (`cli/auth-command.ts:18-118`). There is no small-model setting.
- **models.dev** (`https://models.dev/api.json`, fetched 2026-10-10, 5,377,832 bytes): 226 providers. Each model has `limit.context`, `limit.output`, `cost` (with `cache_read` and `cache_write`), `modalities.input`, `reasoning` and, for most, `reasoning_options`. It has every SDK provider but local Ollama, under the keys `openai`, `anthropic`, `google`, `xai`, `opencode`, `opencode-go`, `huggingface`, `kilo` and `togetherai`. The repository is `anomalyco/models.dev`, MIT, per GitHub's licence API on 2026-10-10.

**The owner's decisions, 2026-10-10:**

1. **Three sub-phases,** each one commit with its own record and gates: **F4-1**, the SDK and the adapters; **F4-2**, the catalog, `models.json` and choosing a model; **F4-3**, credentials.
2. ~~**go-llmprovider-sdk `v1.3.2`.**~~ *(2026-10-11: `v1.7.0`, and the latest tag at each later sub-phase. `v1.3.2` was not the latest tag when this was written; see the Amendments entry "F4-1 re-made for go-llmprovider-sdk `v1.7.0`".)*
3. **The catalog's data is a dated models.dev snapshot,** embedded and credited in `NOTICE`; `models.json` overlays it. Refresh stays 1.x.
4. **Credentials live in a Pi-style `auth.json`.** The OS keyring is 1.x.
5. **ChatGPT, Grok and Kilo logins at 1.0,** each landing only after a recorded check of the vendor's terms for third-party clients.
6. **The small-model role is F5's,** with its first consumers.
7. **`--model` resolves by Pi's rules,** in Pi's words.
8. **A one-run key is read from stdin:** `--api-key -`. A literal is refused, since a key in argv shows in process listings and shell history (the organisation's secrets rule).
9. **`--offline` stays refused,** re-pointed to X3, which brings `web_fetch`, its first network consumer.
10. **ACP logins are terminal auth methods,** running `gobble auth login <provider>`.

**F4-1 — the SDK and the adapters.** *(2026-10-11: re-made as F4-1a and F4-1b, on `v1.7.0`; see the Amendments entry "F4-1 re-made for go-llmprovider-sdk `v1.7.0`". The text below stands where that entry does not change it.)*

Choices made within the wording:

- ~~**`go.mod`** requires `go-llmprovider-sdk v1.3.2`; the record names its sums.~~ *(2026-10-11: `v1.7.0`.)*
- **The ten ids are 1.0:** `openai`, `claude`, `gemini`, `grok`, `opencode-zen`, `opencode-go`, `huggingface`, `kilo`, `together` and `ollama`, each as `providers.New` builds it.
- **A wire test per id,** through `New(id, Options{HTTPClient})` against an `httptest` server for its wire. The client's transport sends every request to the test server, so `Options` gains nothing:
  - a text reply gives one text event and the usage the wire reported;
  - a tool call gives its call;
  - a 429 with `Retry-After: 120` returns at once as `llm.ErrRateLimited`, with `APIError.RetryAfter` 120 s, since the wait is past the policy's 60 s cap;
  - a context overflow is `llm.ErrContextOverflow` after exactly one request.
- **The live tests,** `TestLiveOpenAI`, `TestLiveClaude`, `TestLiveGemini` and `TestLiveGrok`, send one short prompt each, and run only with `GOBBLE_LIVE=1` and the provider's variable set. The owner runs them once; the record names the date and outcome, and F4's accept waits for that, not F4-1's commit.

**F4-1's files:**

| | Files |
| :--- | :--- |
| new | `llm/provider/wire_test.go`, `llm/provider/live_test.go`, `llm/provider/testdata/wire/` |
| changed | `go.mod`, `go.sum`; `docs/architecture.md`; `README.md` |

**F4-1's tests,** each seen failing first on a scratch copy: the four wire cases for each of the ten ids; the live tests' skip without `GOBBLE_LIVE`.

**F4-2 — the catalog, `models.json`, and choosing a model.**
*(2026-10-13, 0019-MADR D11: re-made on go-llmprovider-sdk `v1.9.0`'s `catalog.ModelInfo`, `LookupModel` and `WithModelLayer`,
with gobble's snapshot passed as a layer, in place of a gobble-side resolver.)*
*(2026-10-13: made executable as F4-2a, F4-2b and F4-2c; see the Amendments entry "F4-2 re-made on go-llmprovider-sdk
`v1.12.0`'s model metadata, as three sub-phases". The text below stands where that entry does not change it.)*

Choices made within the wording:

- **`llm/catalog`:**
  - `Model{Provider, ID, Name, ContextWindow, MaxTokens, Input []string, Reasoning bool, Levels []string, Cost{Input, Output, CacheRead, CacheWrite}}`, Pi's `Model` shape for what gobble uses, cost in dollars per million tokens.
  - `data/models.json`, embedded, made by `scripts/gen_catalog.py` from a models.dev `api.json`: the nine keys above mapped to the SDK's ids (`anthropic` to `claude`, `google` to `gemini`, `xai` to `grok`, `opencode` to `opencode-zen`, `togetherai` to `together`, the rest by name), with the snapshot's date and source in the file. The script checks its input and output, and nothing fetches at run time.
  - A model's `Levels` follow Pi's rule (`ai/src/models.ts:1215-1247`): `off` only without reasoning; otherwise the levels models.dev's `reasoning_options` give, as Pi's generator reads them (`scripts/models-dev-reasoning-options.ts`); a level not offered clamps to the next higher, then the next lower, as Pi clamps.
  - Ollama has no snapshot entry: its models are the SDK's list, with Pi's defaults for a custom model (128000 context, 16384 out, no cost).
- **`models.json`**, `<config>/models.json`, in Pi's schema:
  - a provider key must be an SDK id, and `api`, when given, must be that id's own; anything else fails closed for that provider, with a diagnostic naming the SDK version;
  - `baseUrl` goes to the SDK's `WithBaseURL`; `headers` are added by a transport on the provider's client; `models[]` add or replace models with Pi's defaults; `modelOverrides` merge as Pi merges them (`core/model-config.ts:163-190`);
  - `apiKey` and header values resolve at request time as Pi resolves them, `!command` through the bash tool's shell with a 10 s timeout, uncached;
  - `//` comments and trailing commas are accepted as Pi accepts them; a bad file or a bad provider is ignored with a diagnostic, as in Pi.
- **Choosing a model:** `--provider`, `--model` and `--thinking` are accepted. `--model` resolves by Pi's rules and words; `--thinking` beats a `:level` suffix.
  *(2026-10-13: the `defaultProvider` setting came forward in 0019-PLAN P3 (0019-MADR D15), and `Choose` takes it.
  F4-2 adds `defaultModel` ~~and the model scope beside it~~, and `--provider` wins over it for its run.)*
  *(2026-10-14: the model scope, Pi's `enabledModels`, waits for `--models` at 1.x, as 0005-MADR's flag table says;
  see the Amendments entry "F4-2c made executable", decision 11.)*
- **`gobble models list [search]`**, and `--list-models` as its alias, print Pi's columns for providers with a credential. **`gobble models info <model>`** prints one model's fields.
- **acpserver:** the `model` option lists the catalog's models for the session's provider; `thought_level` offers that model's levels; `usage_update` carries the context window as `size`; `_gobble/get_session_stats` reports `cost` from the catalog, and `contextUsage`.
  *(2026-10-13, 0020-MADR D5: the option's models are the SDK's listing from 0020-MADR (0020-PLAN P2, `a45dad6`);
  F4-2 adds `models.json`'s models and the metadata on top.)*
- **`--offline`** stays refused, now naming X3.

**F4-2's files:**

| | Files |
| :--- | :--- |
| new | `llm/catalog/catalog.go`, `catalog_test.go`, `llm/catalog/data/models.json`, `llm/catalog/modelsjson.go`, `modelsjson_test.go`, `llm/catalog/resolve.go`, `resolve_test.go`; `scripts/gen_catalog.py`; `internal/cli/models.go`, `models_test.go` |
| changed | `llm/catalog/doc.go`; `llm/provider/provider.go` (base URL, headers); `acpserver/config.go`, `acpserver/prompt.go`, `acpserver/extensions.go` and their tests; `internal/cli/flags.go`, `internal/cli/agent.go`, `internal/cli/root.go` and their tests; `NOTICE`; `docs/architecture.md`; `README.md` |

**F4-2's tests,** each seen failing first on a scratch copy:

- **The catalog:** a known model's fields from the snapshot; Pi's level rule and clamping; Ollama's defaults; the generator refusing input without the expected keys.
- **`models.json`:** Pi's merge for each field kind; an unknown provider or `api` failing closed; `$NAME`, `${NAME}`, `$$` and `!command` resolved at request time, and a missing variable or a failing command an error; comments; a bad file and a bad provider each a diagnostic.
- **`--model`:** each of Pi's resolution steps and errors, with Pi's words; `:level` from the last colon; `--thinking` winning.
- **`gobble models list`** with a search and with none, and **`info`**; the ACP options, `size` and `cost`.

**F4-3 — credentials.**
*(2026-10-13, 0019-MADR D11, F6: F4-3 uses the SDK's `TokenStore`, `RefreshLocker` and logins, and advertises terminal auth
before it offers one, since acp-go-sdk2's gates otherwise fail `initialize`.)*
*(2026-10-14: re-made as F4-3a, F4-3b and F4-3c; see the Amendments entry "F4-3 re-made as three sub-phases, F4-3a
made executable". The text below stands where that entry does not change it.)*

Choices made within the wording:

- **`internal/auth`:** `<config>/auth.json` in Pi's shape, `{"<provider>": {"type": "api_key", "key": …} | {"type": "oauth", "access", "refresh", "expires", …}}`, the SDK session's own fields (`issuer`, `clientId`, `accountId`, `tokenUrl`) kept beside Pi's. It is created 0600 in a 0700 directory, and on Windows with a DACL for the user alone, through `golang.org/x/sys/windows`, already a requirement. Every write takes a lock file, re-reads, merges and renames into place, as Pi's `modify` does. It implements the SDK's `TokenStore`, so a rotated OAuth token is saved.
- **The order** is Pi's: `--api-key -`, then the `auth.json` entry, which owns its provider, then the `models.json` key, then the provider's variable.
- **`gobble auth`:**
  - `login <provider>`: for `openai`, `grok` and `kilo` the SDK's browser flow, or its device flow with `--device`, each only after its terms check below; for any provider, `--api-key -` stores a key read from stdin;
  - `logout <provider>` removes its entry; `status` lists each provider's credential source and expiry, never a secret;
  - `token <provider> [--min-expiry D]` prints a usable bearer token, refreshing it first, as Pi's `print-bearer-token`;
  - `check [--json]` exits 0 with a credential, 1 without, 2 on error, as Pi's;
  - `import --from-pi` copies Pi's `auth.json` entries for SDK ids, asking for each (0003-MADR).
- **The terms checks:** before each login lands, the record names the vendor's current terms for third-party clients, their URL and date, and whether a client may use the vendor CLI's public client id. A login the terms do not allow stays out, 1.x, and goes to the owner.
- **ACP:** `initialize` lists a terminal auth method for each login, running `gobble auth login <provider>`; `authenticate` answers once that provider has a credential, else `auth_required`; `logout` removes the stored credential of the provider the agent uses.
- **`--api-key`:** `-` reads the key from stdin, or a prompt on a terminal; a literal value is refused with the reason.
- *(2026-10-13, 0019-PLAN P4 deviation 7.)* **A rejected key is not reported as no key.** Today a key the provider
  refuses (401) and no key both reach the CLI as `acpclient.ErrAuthRequired`, and `-p` and the line session print
  "no model credential: set one of …" (`internal/cli/print.go:197`, `line.go:219`). F4-3's `auth_required` says
  whether a credential was found and from which source, and the CLI then says the provider rejected it, naming the
  source; a test covers each case.

**F4-3's files:**

| | Files |
| :--- | :--- |
| new | `internal/auth/store.go`, `store_test.go`, `store_windows.go`, `store_other.go`, `internal/auth/lock.go`, `lock_test.go`, `internal/auth/resolve.go`, `resolve_test.go`; `internal/cli/auth.go`, `auth_test.go` |
| changed | `internal/auth/doc.go`; `llm/provider/provider.go` (token source); `acpserver/agent.go` and its test; `acpserver/testdata/session.golden` (the auth methods); `internal/cli/flags.go`, `internal/cli/agent.go`, `internal/cli/root.go` and their tests; `docs/architecture.md`; `README.md` |

**F4-3's tests,** each seen failing first on a scratch copy:

- **The store:** the file's mode on Unix and its DACL on Windows; two processes writing at once both kept; an OAuth rotation saved through `TokenStore`; a missing file is no credential.
- **The order:** each source winning over the ones after it, and an `auth.json` entry owning its provider.
- **`gobble auth`** with a scripted SDK flow: `login` by browser and device, `--api-key -`, `logout`, `status` never printing a secret, `token`, `check`'s three exits, `import --from-pi` asking per provider.
- **ACP:** the auth methods; `authenticate` with and without a credential; `logout`.
- **`--api-key`:** `-` read from stdin; a literal refused.

**The gates, for each sub-phase,** as F1d-2's deviation 4 sets them: lint for linux, darwin and windows; Windows `go test ./...`; `CGO_ENABLED=1 go test -race ./...` natively on Windows and on the second amd64 Linux machine; Windows and WSL `make preflight`, WSL's without a race run.

**How F4's accept list is met:**

- **A wire test for each 1.0 id,** text, a tool call, usage, a 429 with `Retry-After`, and an overflow: F4-1. *(2026-10-11: and the reset-header and error-flag cases, F4-1a; a stream case, F4-1b.)*
- **The live tests,** run once by the owner, with the date and outcome recorded: F4-1, then the owner.
- **The pin printed by `go list -m`,** and no vendor SDK: F4-1, and checked at each sub-phase.
- **Nothing under `internal/cli` imports go-llmprovider-sdk:** archtest rule 4, unchanged.

**Deferred, named:**

- **The OS keyring:** 1.x, the owner's decision 4.
- **Catalog refresh, and `gobble models refresh`:** 1.x (0005-MADR's row).
- **The small-model role:** F5.
- **`--offline`:** X3.
- **Reading the Codex and Grok CLIs' own credential files** (the SDK's vendor CLI sessions): not in F4's steps; it waits for an amendment that places it.
- **Image input, ~~native streaming,~~ cache-write accounting ~~and 0012-PLAN P7~~:** each waits for the SDK. *(2026-10-11: native streaming is F4-1b, and P7 follows F4-1a; see the Amendments entry "F4-1 re-made for go-llmprovider-sdk `v1.7.0`".)*

**2026-10-11 — F4-1 re-made for go-llmprovider-sdk `v1.7.0`** (the owner's decisions of 2026-10-11: pin `v1.7.0`, and take the latest tag at each later sub-phase; bring a tool result's error flag, a true `NativeStreaming` and live streaming into F4-1; the stream's retry lives in the SDK).
*(2026-10-13, 0019-MADR D2: re-based on `v1.10.2`. The pin and a truthful `NativeStreaming`, the retry wrapper's, landed in 0019-PLAN P3 (`a7de4a8`);
F4-1a keeps the tool result's error flag and the wire tests.)*

**Why.** The SDK released four minor versions after the facts of "F4 made executable" were taken. The owner asked for them to be evaluated against F4. That entry's decision 2 and 0005-MADR's second amendment of 2026-10-10, item 1, call `v1.3.2` the latest tag. It was not: `v1.4.0` was tagged on 2026-10-09 at 17:54 UTC, the day before. The error is corrected here and in 0005-MADR's amendment of 2026-10-11.

**The facts,** checked on 2026-10-11 by a script in the session scratchpad (`q388_claims_f41.py`): 38 claims, all passing. With the check inverted, all 38 failed. SDK paths are under `go-llmprovider-sdk@v1.7.0/llmprovider/`.

- **The releases:** `v1.4.0` (2026-10-09), `v1.5.0` (2026-10-10), `v1.6.0` and `v1.7.0` (2026-10-11), each a minor release. `v1.7.0` requires Go 1.27.2, as gobble does.
- **gobble on `v1.7.0`,** on a scratch clone: `go get` and `go mod tidy` change `go.mod` and `go.sum` only, and no other requirement moves; `go build ./...`, `go vet ./...` and `go test -count=1 ./...` (29 packages) pass with no code change.
- **The F4 entry's twelve SDK claims,** re-run at `v1.7.0`: one changes, "a tool has no strict flag". The rest hold: no model has limits or cost, there is no keyring, the token store and the logins are as recorded, and no item carries an image. `Usage` is unchanged, so there is still no cache-write count.
- **`v1.4.0`** (its 0028-MADR):
  - an overflow message that states its split, with the completion alone at or past the limit, is `ErrInvalidRequest`, not `ErrContextOverflow` (`context_overflow.go:69`);
  - with no `Retry-After`, a 429's `RetryAfter` comes from the reset headers: `x-ratelimit-reset-requests` and `-tokens` as Go durations (OpenAI, Hugging Face), and Claude's `anthropic-ratelimit-*-reset` as RFC 3339 times (`internal/transport/transport.go:236-246`);
  - a wait past `MaxDelay` still returns at once (`retry.go:151-156`);
  - a 403 that names no credential is `ErrNotPermitted`, which gobble already maps (`llm/provider/provider.go:261`).
- **`v1.5.0` and `v1.6.0`** (its 0031-MADR): every built-in provider streams natively. Eight declare `NativeStreaming: Supported`, and OpenCode's two `BestEffort` (`providers/*/*.go`); at `v1.3.2` none did. `WithRetry` buffers, and reports `Unsupported` (`retry.go:95-99`). A caller that wants text as it is generated streams the provider itself, and handles a failure there (`retry.go:60-66`).
- **`v1.7.0`** (its 0032-MADR): `Tool.Strict *bool` (`provider.go:26`) and `FunctionCallOutputItem.IsError` (`item.go:48`), each sent where 0032-MADR's measurement found the wire's field. `v1.3.2` had neither.
- **gobble today:**
  - `New` takes its capabilities from the provider it wraps, not from the `WithRetry` result (`llm/provider/provider.go:65`, `:81-83`). From `v1.6.0`, `Capabilities().NativeStreaming` is therefore true while `Stream` is buffered.
  - The adapter drops a tool result's `IsError` (`provider.go:219`), which `llm` carries (`llm/llm.go:73`). A failed tool call reaches Claude and Gemini as a success.
  - The agent loop shows each delta as it arrives (`agent/agent.go:365-369`), and a failed call ends the turn with nothing added to the history (`:249-252`). A script can yield events and then an error (`llm/llmtest/script.go:76-87`).
  - `llm` has no `ErrInvalidRequest`: an invalid request is an `*llm.APIError` with its status (`llm/errors.go`).

**The owner's decisions, 2026-10-11:**

1. **Pin `v1.7.0`.** Each later sub-phase starts by pinning the SDK's latest tag. On it, it runs `go build ./...`, `go vet ./...`, `go test -count=1 ./...` and the F4 claims, and records any claim that changed as a deviation.
2. **F4-1 is two commits.** F4-1a is the pin, the wire tests and two fixes. F4-1b is live streaming, on the SDK release of decision 3.
3. **The stream's retry lives in the SDK.** gobble asks go-llmprovider-sdk for this, as a record of its own there:
   - `WithRetry`'s result implements `Streamer`, and reports the inner provider's `NativeStreaming`;
   - a failure before the first event is retried under the policy's rules: the same retryable test, waits and cap, and the once-only resend of a request that was written;
   - after the first event, every event passes through, and a failure ends the stream with no `EventDone`, as 0031-MADR D4 has it.

   F4-1b pins the first release that has it. Until then, `Stream` stays buffered and `NativeStreaming` says so.
4. **0012-PLAN P7 follows F4-1a,** made executable as an amendment to 0012-PLAN first, as P7's own text requires.
5. **F4-2 follows F4-1a.** It does not wait for F4-1b, which lands before F4's accept.

**F4-1a — the pin, the wire, and two fixes.**
*(2026-10-13: re-made at go-llmprovider-sdk `v1.12.0`; see the Amendments entry "F4-1a re-made at go-llmprovider-sdk `v1.12.0`".
The pin and the truthful `NativeStreaming` are done (0019-PLAN P3); the rest is planned there.)*

- **`go.mod`** requires `go-llmprovider-sdk v1.7.0`; the record names its sums.
- **A true `NativeStreaming`:** `New` keeps the capabilities of the `WithRetry` result, so `Capabilities().NativeStreaming` is false until F4-1b; `Wrap`, which adds no retry, keeps the provider's own. `llm/llm.go:21` reads "NativeStreaming is false while the provider's retry buffers the stream."
- **A failed tool result is sent as one:** `toRequest` builds `llmprovider.FunctionCallOutputItem{CallID: c.ToolCallID, Output: c.Text, IsError: c.IsError}`.
- **The wire tests of "F4 made executable",** with one change: the overflow case's message states no split, so `v1.4.0` still classifies it as overflow.
- **The cases `v1.4.0` and `v1.7.0` add,** in the same `wire_test.go`:
  - `openai` and `huggingface`: a 429 with no `Retry-After` and `x-ratelimit-reset-requests: 90s` returns at once as `llm.ErrRateLimited`, with `RetryAfter` 90 s, after one request;
  - `claude`: the same, with `anthropic-ratelimit-requests-reset` 90 s ahead as an RFC 3339 time, and `RetryAfter` between 89 s and 90 s;
  - `openai`: an overflow message stating `(N in the messages, M in the completion)` with M at the limit is an `*llm.APIError` with status 400, and not `llm.ErrContextOverflow`;
  - `claude`: a tool result with `IsError` reaches the body as `tool_result` with `is_error: true`; `openai`: its output reaches the body unchanged.
- **Unit tests** in `provider_test.go`: `New` over a fake that declares `Supported` reports `NativeStreaming` false; `toRequest` carries `IsError` onto the item.
- **The live tests** of "F4 made executable", unchanged.

**F4-1a's files:**

| | Files |
| :--- | :--- |
| new | `llm/provider/wire_test.go`, `llm/provider/live_test.go`, `llm/provider/testdata/wire/` |
| changed | `go.mod`, `go.sum`; `llm/provider/provider.go`, `llm/provider/provider_test.go`; `llm/llm.go`; `docs/architecture.md`; `README.md` |

**F4-1b — live streaming.**

- **`go.mod`** requires the SDK release of decision 3, or a later tag, by decision 1.
- **`New`** keeps the `WithRetry` result's capabilities, which on that release report `Supported`; `llm/llm.go:21` reads "NativeStreaming is true when text arrives as the model writes it."
- **`llm/doc.go:20-22`:** text and thinking arrive as deltas while the model writes; calls arrive whole, as `ToolCallDone`; `ToolCallStart` and `ToolCallDelta` stay unproduced.
- **Wire cases, for each of the ten ids,** in `wire_test.go`, against a streamed reply:
  - two or more text deltas arrive before `Done`, and their concatenation is the reply's text;
  - a 503 before the first event is retried, and the stream then succeeds, after two requests;
  - a failure after the first delta ends the stream with the mapped error, with no `Done`, after one request.
- **`agent/agent_test.go`:** `TestMidStreamFailure`. A script yields `TextDelta{Text: "par"}`, then `llm.ErrUnavailable`. The collected events hold that delta, the error is `llm.ErrUnavailable`, and there is no `agent.Message` or `agent.End`.

**F4-1b's files:**

| | Files |
| :--- | :--- |
| new | `llm/provider/testdata/wire/` stream replies |
| changed | `go.mod`, `go.sum`; `llm/provider/provider.go`, `llm/provider/wire_test.go`; `llm/llm.go`, `llm/doc.go`; `agent/agent_test.go`; `docs/architecture.md` |

**Tests,** each seen failing first on a scratch copy: every case above.

**How F4's accept list is met,** in addition to "F4 made executable":

- **Live text over ACP and in the CLI:** F4-1b, through the agent loop's existing deltas.
- **A failed tool result reaches the model as one:** F4-1a.

**Deferred, named:**

- **Image input and cache-write accounting:** each still waits for the SDK.
- **Call-argument deltas** (`ToolCallStart`, `ToolCallDelta`): the SDK yields calls whole (0031-MADR D2), so they wait for an SDK release that streams them.

**2026-10-13 — F4-1a re-made at go-llmprovider-sdk `v1.12.0`** (the owner's choices of 2026-10-13: re-plan F4-1a to what remains;
the live tests take the repository's `live_<provider>` build tags, and the agent runs `TestLiveOpenAI` once).

**Why.** The build order's next step, "F4-1a, the SDK pin, the wire tests and two fixes", is half done:
0019-PLAN P3 (`a7de4a8`) pinned the SDK and made `NativeStreaming` the retry wrapper's,
and 0020-PLAN P1 (`1625013`) and 0021-PLAN P1 (`c522a7b`) moved the pin to `v1.12.0`, the latest tag.
The error flag, the wire tests and the live tests remain.

**The facts,** measured on 2026-10-13 against `main` `8ff38dd` and go-llmprovider-sdk `v1.12.0`
(scratch probes `i2probe/f41a` and `i2probe/iserror`, offline, over an in-process transport that answers by path):

- **Pins:** acp-go-sdk2 `v0.17.1`, go-llmprovider-sdk `v1.12.0`, go-selfupdate-lib `v1.15.2`, each the latest tag;
  decision 1 of 2026-10-11 needs no pin change.
- **Done:** `New` keeps the `WithRetry` result's capabilities (`llm/provider/provider.go:75-76`),
  `TestNewReportsRetryCapabilities` holds it (`provider_test.go:119-131`), and `llm/llm.go:21` says why.
- **Not done:** `toRequest` drops a tool result's `IsError`: it builds `FunctionCallOutputItem{CallID, Output}`
  (`llm/provider/provider.go:231`), though `llm.Content.IsError` exists (`llm/llm.go:89`).
  No `wire_test.go` or live test of a provider's reply exists; `strict_live_test.go` (0012-PLAN P5) is the only live test.
- **The F4 claims at `v1.12.0`:** the ten ids hold (`constants.go`); no item carries an image; there is no keyring;
  `TokenStore` is `Load`, `Save` and `Delete`, with `RefreshLocker` beside it; the browser and device logins hold.
  The claims that changed are 0019-MADR's F9 and F11: native streaming, the strict flag, model limits and cost, cache writes.
- **Each id's wire,** as gobble's `New` builds it: `openai` and `grok` `/v1/responses`; `claude` `/v1/messages`;
  `gemini` `/v1beta/interactions`; `opencode-zen` with a Claude model `/zen/v1/messages`;
  `opencode-go` `/zen/go/v1/chat/completions`; `huggingface`, `together` and `ollama` `/v1/chat/completions`;
  `kilo` `/api/gateway/chat/completions`.
- **The cases, through gobble's `llm/provider` today, for all ten ids,** one request each:
  - a text reply gives its text and the usage the wire reported; a tool reply gives its call;
  - a 429 with `Retry-After: 120` is `llm.ErrRateLimited` with `RetryAfter` 120 s, returned at once (the policy's cap is 60 s);
  - an overflow (`prompt is too long`, code `context_length_exceeded`) is `llm.ErrContextOverflow`;
  - a 429 with a reset header 90 s ahead and no `Retry-After` gives `RetryAfter` 90 s;
  - an overflow message stating its split (`100 in the messages, 1000 in the completion`) is an `*llm.APIError`
    with status 400, and not `llm.ErrContextOverflow`.
- **The error flag on the wire** (the SDK alone, a failed `FunctionCallOutputItem`): `claude`, `gemini` and
  `opencode-zen`'s messages route send `"is_error":true`; the other seven send the output text unchanged and no flag,
  as their package docs say.
- **The live-test convention:** a `live_<provider>` build tag, and `llmtest.LiveCredential`, which skips when the
  provider's variable is unset (`llm/llmtest/live.go:8-19`; `strict_live_test.go:1`). `GOBBLE_LIVE`, which the
  2026-10-10 entry names, is read by no test.

**The decisions:**

1. **No pin change.** F4-1a runs on `v1.12.0`; if a newer tag exists when it starts, decision 1 of 2026-10-11 applies.
2. **A failed tool result is sent as one:** `toRequest` builds
   `llmprovider.FunctionCallOutputItem{CallID: c.ToolCallID, Output: c.Text, IsError: c.IsError}`.
3. **The wire tests run in process.** A transport on `Options.HTTPClient` answers each request by its path with a canned
   reply in the wire's shape, as the SDK's own `wirecase` does; no `httptest` server and no `testdata/wire/` directory.
   The replies are constants in the test file.
4. **The live tests take the repository's convention:** one file per provider, tagged `live_openai`, `live_claude`,
   `live_gemini` and `live_grok`, each skipping without its variable. The agent runs `TestLiveOpenAI` once, one billed
   request with the owner's test key in the child's environment only; the record names which ran, when, and the outcome.
5. **0012-PLAN P7 and F4-2 follow F4-1a,** as decisions 4 and 5 of 2026-10-11 say.

**F4-1a's steps:**

1. **`llm/provider/provider.go:231`:** `FunctionCallOutputItem{CallID: c.ToolCallID, Output: c.Text, IsError: c.IsError}`.
2. **`llm/provider/provider_test.go`, `TestToolResultError`:** a history whose tool result has `IsError` maps to an item
   with `IsError` true, and one without to false.
3. **`llm/provider/wire_test.go` (new),** package `provider`:
   - `cannedWire`, an `http.RoundTripper` that records each request's method, path, headers and body, and answers by
     the path's suffix: `/responses`, `/messages`, `/interactions` and `/chat/completions`, each with a text reply or a
     `get_weather` call, and a 404 for any other path;
   - **`TestWireCases`,** for each of the ten ids with its model above, one subtest per case, each after exactly one request:
     text (`hello`, usage 12 in and 3 out); a tool call (`get_weather`, arguments `{"city":"Paris"}`);
     `Retry-After: 120` (`llm.ErrRateLimited`, `RetryAfter` 120 s); overflow (`llm.ErrContextOverflow`);
   - **`TestWireRateLimitResets`:** `openai` and `huggingface` with `x-ratelimit-reset-requests: 90s`, and `claude` with
     `anthropic-ratelimit-requests-reset` 90 s ahead as an RFC 3339 time, each alone: `llm.ErrRateLimited`, `RetryAfter`
     90 s (between 89 s and 90 s for the RFC 3339 form), one request;
   - **`TestWireOverflowSplit`:** `openai` with the split message is an `*llm.APIError` of status 400, not `llm.ErrContextOverflow`;
   - **`TestWireToolResultError`:** a conversation ending in a failed `get_weather` result: `claude`, `gemini` and
     `opencode-zen` (`claude-haiku-5-5`) send `"is_error":true`; `openai` and `huggingface` send `no such city` and no flag.
4. **The live tests (new):** `live_openai_test.go` (`//go:build live_openai`, `TestLiveOpenAI`, `gpt-6-luna`),
   `live_claude_test.go` (`claude-haiku-5-5`), `live_gemini_test.go` (`gemini-3.5-flash-lite`) and `live_grok_test.go`
   (`grok-4.6`), each in package `provider_test`, each sending "Reply with the single word: ok." through `provider.New`
   with `llmtest.LiveCredential`'s key, and asserting a non-empty reply and non-zero usage; the shared `liveTurn` helper is
   in `live_test.go`, tagged `live_openai || live_claude || live_gemini || live_grok`. Errors are reported with the key redacted.
5. **The live run:** `go test -tags live_openai -count=1 -run '^TestLiveOpenAI$' -v ./llm/provider`, once, with
   `OPENAI_API_KEY` from the owner's test file in the child's environment only, every output checked not to contain it.

**F4-1a's negatives,** each on a scratch copy after a control run passes:

1. `toRequest` without `IsError`: `TestToolResultError` and `TestWireToolResultError`'s three flagged rows fail.
2. `mapError` without the `ErrRateLimited` sentinel: `TestWireCases`' ten `Retry-After` subtests and `TestWireRateLimitResets` fail.
3. `mapError` without the `ErrContextOverflow` sentinel: the ten overflow subtests fail.
4. `mapError` dropping `RetryAfter`: the `Retry-After` and reset subtests fail on the wait.
5. A live test without its build tag: `go vet ./llm/provider` then compiles it, and its `LiveCredential` skip is shown
   by `go test -tags live_openai -run TestLiveOpenAI -v ./llm/provider` with `OPENAI_API_KEY` unset: `--- SKIP`.

**F4-1a's verification:**

```text
go test -count=1 ./llm/provider -run 'TestWire|TestToolResultError'           -> ok
go vet -tags 'live_openai live_claude live_gemini live_grok' ./llm/provider    -> exit 0
go test -tags live_claude -count=1 -run TestLiveClaude -v ./llm/provider       -> --- SKIP: set ANTHROPIC_API_KEY to run it
the gates                                                                      -> lint ×3, tests on three systems, -race on two, preflight
the live run                                                                   -> --- PASS: TestLiveOpenAI; the key in no output
```

**F4-1a's files:**

| | Files |
| :--- | :--- |
| new | `llm/provider/wire_test.go`, `llm/provider/live_test.go`, `live_openai_test.go`, `live_claude_test.go`, `live_gemini_test.go`, `live_grok_test.go` |
| changed | `llm/provider/provider.go`, `llm/provider/provider_test.go`; this plan; `docs/README.md` |

F4-1a is one commit, with its own record of what ran. F4-1b is unchanged: it waits for a streaming `WithRetry`.

**F4-1a's execution record (2026-10-13).**

- **Ran as planned,** on go-llmprovider-sdk `v1.12.0`, still the latest tag when it started; no deviation.
- **Tests:** `TestToolResultError`; `TestWireCases`, 40 subtests (the ten ids, each text, a tool call, `Retry-After: 120`
  and an overflow); `TestWireRateLimitResets` (3); `TestWireOverflowSplit`; `TestWireToolResultError` (5). All pass, each
  case after one request.
- **The live tests,** without their keys, each skip naming the variable; `go vet` and `golangci-lint` with the four tags: 0 issues.
- **Negatives,** 5 of 5 caught after control runs: `toRequest` without `IsError`; `mapError` without the rate-limit sentinel;
  without the overflow sentinel; dropping `RetryAfter`; the live helper without `LiveCredential`'s skip, which then fails.
- **The live run:** `TestLiveOpenAI` passed on 2026-10-13, one billed request to `gpt-6-luna`; the key was in no output.
  `TestLiveClaude`, `TestLiveGemini` and `TestLiveGrok` have not run: no key for them here; the owner runs them.
- **Gates:** lint for linux, darwin and windows, 0 issues; Windows `go test` and `-race`, the second Linux machine's `-race` and
  the macOS laptop's `go test`, 31 packages ok each, no race report; Windows preflight passed.
- **The agent's own error, fixed before the commit:** the canned transport first copied its reply headers with their keys as
  written, so `http.Header.Get` missed the lowercase reset headers and the SDK retried four times; it now canonicalises them,
  as a server's headers arrive.
- **Next:** 0012-PLAN P7, then F4-2 (decisions 4 and 5 of 2026-10-11).

**2026-10-13 — F4-2 re-made on go-llmprovider-sdk `v1.12.0`'s model metadata, as three sub-phases**
(the owner's choices of 2026-10-13: F4-2a, F4-2b and F4-2c, each one commit; the snapshot made by a Python script and
checked by the SDK's own parser; the snapshot passed as a caller layer, its ranking above the listing asked of the SDK).

**Why.** F4-2's text of 2026-10-10 designs a gobble-side resolver over a models.dev snapshot. 0019-MADR D11 re-bases it
on the SDK's model metadata, which the SDK's 0033-MADR D3 designed for gobble: an embedded models.dev copy, parsed and
passed as a layer, offline. That text was never made executable. F4-2 is also one commit for five pieces of work.

**The facts,** measured on 2026-10-13 against `main` `f7c6d60` and go-llmprovider-sdk `v1.12.0`, still the latest tag
(`git ls-remote --tags`; decision 1 of 2026-10-11 needs no pin change):

- **gobble today:**
  - `llm/catalog` is a declared package only (`llm/catalog/doc.go:1-5`); nothing embeds model data, and no
    `models.json` is read.
  - `llm.Model` holds `Provider`, `ID` and `Name`, filled by nothing (`llm/llm.go:101-107`).
  - Only `catalog.List` and `catalog.WithProfile` of the SDK's catalog are used (`llm/provider/list.go:35-58`);
    `New` sets `WithoutModelMetadata` (`provider.go:51-52`).
  - ACP's `thought_level` offers one fixed list, off to xhigh, whatever the model (`acpserver/config.go:42`, `:105-111`).
  - `usage_update` sends `used` alone, `size` 0 (`acpserver/prompt.go:298-308`), from four places
    (`agent.go:511`, `commands.go:518`, `compact.go:38`, `extensions.go:241`).
  - An assistant message's `usage.cost` is all zero (`acpserver/history.go:102-125`).
  - `_gobble/get_session_stats` reports `cost` 0 and no `contextUsage` (`acpserver/extensions.go:479-508`);
    `/usage` and `/context` print that prices and the window wait for F4 (`acpserver/commands.go:237`, `:278`).
  - `--provider`, `--model` and `--thinking` are refused (`internal/cli/flags.go:68-71`); there is no `--list-models`,
    no `gobble models` and no `defaultModel` setting (`internal/config/keys.go:62-68`).
- **The SDK's model metadata** (paths under `go-llmprovider-sdk@v1.12.0/llmprovider/catalog/`):
  - `ModelInfo` has the window, input and output caps, modalities, reasoning (support, toggle, efforts, budget),
    prices in USD per million tokens with tiers, release date and status; `Has` tells a known zero from an unknown
    (`model_info.go:17-60`, `:199`).
  - `ParseMetadata` reads a models.dev-format document and makes no request; `Metadata` is a `Layer`
    (`metadata_document.go:22-31`, `:67-78`).
  - `LookupModel` and `CachedModel` merge models.dev, then the provider's listing (when given a credential), then each
    caller layer, a later one winning field by field (`lookup.go:50-157`). `CachedModel` never waits.
  - So a caller layer outranks the listing: the dated snapshot's window and prices beat a fresher listing's. The
    listing fills only what the snapshot lacks.
  - `catalog` cannot be imported outside `llm/provider` (`internal/archtest/check.go:238-254`, rule 4).
- **models.dev,** `https://models.dev/api.json`, fetched 2026-10-14 02:10 UTC: 5,621,182 bytes, 228 providers.
  - The nine sections the SDK reads (`openai`, `anthropic`, `google`, `xai`, `opencode`, `opencode-go`, `huggingface`,
    `kilo`, `togetherai`) are 515,687 bytes compact; kilo's 390 models are half of it.
  - The SDK's `Encode` of the parsed document is 515,649 bytes.
  - Licence: MIT, "Copyright (c) 2025 models.dev" (GitHub's licence API for `anomalyco/models.dev`, 2026-10-13).
- **Coverage,** through the SDK's parser and `CachedModel` with the document as a layer and no network (scratch module
  `f42probe`): all 49 curated models of the nine key providers have a window and a price. Ollama has no section.
  For example `gpt-6-luna`: window 1,050,000, output 128,000, efforts none, low, medium, high, xhigh and max,
  $0.10 in and $0.50 out, one tier.
- **Pi** (`312184edb`):
  - **levels:** a model that does not reason has `off` alone. One that reasons, with no effort list, has off to high.
    With efforts, `off` when `none` is among them, and each of minimal to max that is listed
    (`ai/src/models.ts:1215-1226`, `ai/scripts/models-dev-reasoning-options.ts:18-30`). Pi's generator then overrides
    some vendors by hand (`ai/scripts/generate-models.ts`), which gobble does not copy.
  - **clamping:** an unoffered level goes to the next higher offered, else the next lower (`ai/src/models.ts:1228-1247`).
    On a model change the session's level is clamped, and saved to the transcript only when it changes
    (`coding-agent/src/core/agent-session.ts:2527-2533`).
  - **cost:** per category, `rate / 1e6 × tokens`, the tier chosen by input plus cache reads and writes
    (`ai/src/models.ts:1193-1213`). A 1-hour cache write costs twice the input rate; llm's usage does not split the
    two kinds of write, so gobble prices every write at the cache-write rate.
  - **stats:** `cost` is the sum of each message's `usage.cost.total`. `contextUsage` is
    `{tokens, contextWindow, percent}`, omitted without a window (`agent-session.ts:4119-4180`).
- **A difference recorded, not fixed:** models.dev's `null` effort is `none` in the SDK (`metadata_document.go:188-190`),
  which Pi drops. A model listing `null` therefore offers `off` in gobble and not in Pi.

**The decisions:**

1. **Three sub-phases, each one commit with its own record and gates.** F4-2a is executable below. F4-2b and F4-2c keep
   the 2026-10-10 text where this entry does not change it, and each is made executable as an amendment before it runs.
2. **The snapshot is data in `llm/catalog`, parsed in `llm/provider`.** `scripts/gen_catalog.py` writes
   `llm/catalog/data/models.json`, the nine sections, and `llm/catalog/data/source.json`, its source and time;
   models.dev's licence is kept beside them and credited in `NOTICE`. `llm/provider` parses it once with
   `catalog.ParseMetadata`, and a test there is the format's judge.
3. **One lookup, never waiting:** `provider.ModelInfo` is `catalog.CachedModel` with `WithoutModelMetadata`, the
   snapshot as `WithModelLayer`, and the session's credential as the listing's token source when there is one.
4. **The pure rules are `llm/catalog`'s:** Pi's level rule and clamp, and Pi's per-category cost, over `llm.Model`.
5. **F4-2a feeds what exists:** `thought_level` per model, with Pi's clamp; `usage_update`'s `size` and `cost`;
   each assistant message's and compaction's `usage.cost`; the stats' `cost` and `contextUsage`; `/usage` and `/context`.
6. **The ranking is asked of the SDK.** The owner files a request against go-llmprovider-sdk for a caller layer that
   ranks below the provider's listing; until a release has one, the snapshot outranks the listing.

**F4-2a — the snapshot, the lookup, and what they feed.**

1. **`scripts/gen_catalog.py` (new).** `python3 scripts/gen_catalog.py --input <api.json>` or `--fetch`:
   - reads the document (`--fetch`: `https://models.dev/api.json`, 60 s timeout); refuses, exit 1 naming the key, a
     document missing any of the nine sections or a section without `models`;
   - writes `llm/catalog/data/models.json`: those nine sections alone, `json.dumps(..., sort_keys=True,
     separators=(",", ":"))`, LF, one final newline;
   - writes `llm/catalog/data/source.json`: `{"source": <url or "file">, "fetched": <UTC, RFC 3339 to the minute>,
     "sha256": <the input's>, "providers": {<key>: <model count>}}`, indented 2;
   - prints one line per section, and nothing fetches at run time.
2. **`llm/catalog/data/LICENSE.models.dev` (new):** models.dev's MIT licence, verbatim.
3. **`llm/catalog/catalog.go` (new),** no SDK import:
   - `//go:embed data/models.json data/source.json`;
   - `Snapshot() []byte`, a copy of the document; `Source() (url string, fetched time.Time)`, from `source.json`;
   - `Levels(m llm.Model) []string`: Pi's rule above over `m.Reasoning` and `m.Efforts`; with `m.ReasoningKnown` false,
     today's list, off to xhigh, so a model the snapshot lacks offers what it offers now;
   - `Clamp(levels []string, level string) string`: Pi's clamp, over the order off, minimal, low, medium, high, xhigh, max;
   - `Price(m llm.Model, input, output, cacheRead, cacheWrite int64) Charge`, with `Charge{Input, Output, CacheRead,
     CacheWrite, Total float64}`: Pi's formula, the tier by `input + cacheRead + cacheWrite`; zero when `m.Priced` is false.
4. **`llm/llm.go`:** `Model` gains, after `Name`, `ContextWindow`, `MaxTokens int` (0 unknown); `Input []string`;
   `Reasoning`, `ReasoningKnown bool`; `Efforts []string`; `Cost ModelCost`; `Priced bool`. `ModelCost{Input, Output,
   CacheRead, CacheWrite float64; Tiers []CostTier}` and `CostTier{Above int; Input, Output, CacheRead, CacheWrite float64}`
   are new, in USD per million tokens. Its comment "The catalog package fills real rows later." becomes
   "llm/provider fills it from the catalog (0005-PLAN F4-2a)."
5. **`llm/provider/model.go` (new):**
   - the snapshot parsed once (`sync.OnceValues` over `catalog.ParseMetadata(bytes.NewReader(gcatalog.Snapshot()))`);
     ~~a parse error leaves every model unknown and is logged once;~~ *(F4-2a deviation 1: a parse error leaves every
     model unknown, and `SnapshotErr() error` reports it for the CLI to log; the unexported helper is `lookupModel`)*;
   - `ModelInfo(id, apiKey, model string) llm.Model`: `catalog.CachedModel(context.Background(), id, src, model,
     llmprovider.WithoutModelMetadata(), catalog.WithModelLayer(meta))`, `src` the key as a static token when not empty;
   - each `llm.Model` field set only where `Has` says the field is known; `Priced` when input and output prices are.
6. **`acpserver`:**
   - `Options.ModelInfo func(provider, model string) llm.Model`; nil gives the zero `Model`, so today's behaviour holds;
   - `config.go`: `thought_level`'s values are `catalog.Levels` of the session's model, its current value clamped.
     A model change clamps the session's level and, when it changes, appends a `thinking_level_change` after the
     `model_change`, as Pi does. A level the model does not offer is refused, as an unknown level is today;
   - `prompt.go`: `usedUpdate` becomes `(a *Agent) contextUpdate(s, used)`, with `size` the window when known and `cost`
     the session's total in USD when the model is priced. The four call sites take it; `usageUpdate` goes;
   - `history.go` and `compact.go`: an assistant message's and a compaction's `usage.cost` is `catalog.Price` of its model;
   - `commands.go`: `sessionTotals` sums `usage.cost.total`. `/usage` prints `Cost: $<total, 3 places>`, or
     `Cost: unknown (no price for <provider>/<model> in the catalog)`. `/context` prints `Context: <used> of <window>
     tokens (<pct>%)`, or today's line without the F4 parenthesis when the window is unknown;
   - `extensions.go`: the stats' `cost` is that total, and `contextUsage` is `{tokens, contextWindow, percent}` with the
     window known, else absent.
7. **`internal/cli/agent.go`:** `agentOptions` sets `ModelInfo` to `provider.ModelInfo` with the chosen provider's key,
   read from its variable as `modelProvider` reads it. *(F4-2a deviation 1:)* when `provider.SnapshotErr()` is set,
   `agentOptions` logs it once through `e.log()`, as an `slog.Any` attribute.
8. **Records:** `NOTICE` credits models.dev's data with its licence; `docs/architecture.md`'s `llm/catalog` row reads
   "The embedded models.dev snapshot and the level and price rules (0005-PLAN F4-2a)."; this plan; `docs/README.md`.

**F4-2a's tests,** each seen failing first on a scratch copy:

- `llm/catalog`: `TestSnapshot` (the nine keys; `source.json`'s fields and counts match the document); `TestLevels`
  (no reasoning; reasoning with no efforts; efforts with and without `none`; with `max`; reasoning unknown);
  `TestClamp` (up, then down; unknown level); `TestPrice` (each category; a tier crossed and not; unpriced is zero).
- `llm/provider`: `TestSnapshotParses` (`ParseMetadata` accepts the embedded document, each of the nine ids lists
  models); `TestModelInfo` over a canned two-model document (fields known and unknown, `Priced`, efforts, tiers;
  an unknown model is zero).
- `acpserver`, with a `ModelInfo` of two canned models: `thought_level`'s values per model and the clamp's two entries;
  `usage_update`'s `size` and `cost`; an assistant message's `usage.cost`; the stats' `cost` and `contextUsage`;
  `/usage` and `/context`, priced and unknown. The existing tests pass unedited, since they pass no `ModelInfo`.
- `scripts/gen_catalog.py` on a scratch copy of the document without `kilo` exits 1 naming it.

**F4-2a's verification:**

```text
python3 scripts/gen_catalog.py --fetch                   -> nine lines; data/models.json and data/source.json written
go test -count=1 ./llm/... ./acpserver/...                -> ok
make archtest                                             -> ok (no SDK import outside llm/provider)
go test -count=1 ./acpserver -run 'TestSession|TestGolden' -> ok, unedited
the gates                                                 -> lint ×3, tests on three systems, -race on two, preflight
```

**F4-2a's files:**

| | Files |
| :--- | :--- |
| new, data | `scripts/gen_catalog.py`; `llm/catalog/data/models.json`, `data/source.json`, `data/LICENSE.models.dev` |
| new, code | `llm/catalog/catalog.go`, `catalog_test.go`; `llm/provider/model.go`, `model_test.go`; `acpserver/catalog_test.go` |
| changed, code | `llm/llm.go`; `llm/catalog/doc.go`; `internal/cli/agent.go` |
| changed, acpserver | `agent.go`, `config.go`, `prompt.go`, `history.go`, `compact.go`, `commands.go`, `extensions.go` |
| changed, records | `NOTICE`; `docs/architecture.md`; this plan; `docs/README.md` |

**F4-2a deviation 1 (2026-10-13), found by the gates' lint.**

- **What was found.** `golangci-lint` printed 4 issues, the same for linux, darwin and windows, in lines F4-2a added,
  and `make preflight` failed on them:
  - `llm/catalog/catalog.go:43:27`: ``string `off` has 3 occurrences, make it a constant (goconst)``;
  - `llm/provider/model.go:45:6`: `Method 'modelInfo' differs only by capitalization to function 'ModelInfo' (revive)`;
  - `llm/provider/model.go:21:3` and `:21:88`: `default logger should not be used` and `key-value pairs should not
    be used (sloglint)`, for step 5's "logged once".
- **The owner's decision (2026-10-13):** `"off"` becomes a constant; the helper is renamed `lookupModel`; a snapshot the
  SDK cannot parse leaves models unknown, `provider.SnapshotErr()` reports it, and `internal/cli`'s `agentOptions` logs
  it once through its own logger. Steps 5 and 7 are annotated above; no file is added to the phase's scope.

**F4-2b — `models.json`.** As the 2026-10-10 text, re-based: `models.json` is a second caller layer, added after the
snapshot, so it outranks it; `models[]` adds models to the choice; `baseUrl` and `headers` reach the provider. Made
executable as an amendment before it runs.
*(2026-10-13: made executable; see the Amendments entry "F4-2b made executable".)*

**F4-2c — choosing a model.** As the 2026-10-10 text: `--provider`, `--model` and `--thinking` by Pi's rules,
`defaultModel` beside `defaultProvider`, `gobble models list` and `info`, `--list-models`, completion, and `max` in the
`--thinking` and `defaultThinkingLevel` enums. Made executable as an amendment before it runs.
*(2026-10-14: made executable; see the Amendments entry "F4-2c made executable".)*

**Deferred, named:**

- **A caller layer ranked below the listing:** the owner's request to go-llmprovider-sdk (decision 6).
- **1-hour cache writes at twice the input rate:** needs llm's usage to split the two kinds of write, which needs the SDK's.
- **Refreshing the snapshot, and writing back a copy the SDK fetched:** 1.x, as decision 3 of 2026-10-10 says.
- **Pi's per-vendor level overrides:** not copied; a model whose levels are wrong is corrected by `models.json` (F4-2b).

**F4-2a's execution record (2026-10-13).**

- **Ran on go-llmprovider-sdk `v1.12.0`,** still the latest tag, with one deviation, the owner's decision (F4-2a deviation 1).
- **The snapshot:** `gen_catalog.py --fetch` at 2026-10-14 02:22 UTC wrote the nine sections, 782 models (anthropic 17,
  google 40, huggingface 81, kilo 390, openai 53, opencode 122, opencode-go 37, togetherai 29, xai 13).
- **Tests:**
  - `llm/catalog`: `TestSnapshot`, `TestLevels`, `TestClamp` and `TestPrice` pass;
  - `llm/provider`: `TestSnapshotParses` and `TestModelInfo` pass;
  - `acpserver`: `TestCatalogThinkingLevels`, `TestCatalogPriced` and `TestCatalogUnknown` pass;
  - every existing test, the session golden file's included, passes unedited.
- **Negatives,** 15 of 15 caught on a scratch copy after control runs, each with its named tests and text, each built:
  - Levels offering `off` without `none`; Clamp without its upward search; a tier at its threshold; Price ignoring `Priced`;
    `Snapshot` returning the embedded bytes; the snapshot without kilo (`TestSnapshot` and `TestSnapshotParses`);
  - `modelOf` dropping reasoning, or the tiers;
  - a model change without the clamp's entry; levels ignoring the catalog; `usage_update` without the window; an
    unpriced message; the stats without `contextUsage`; `/usage` pricing an unpriced model;
  - the generator on a document without kilo: exit 1, "the document has no 'kilo' section with models", data unchanged.
  - All 15 ran again after deviation 1's fix, and were caught again.
- **Gates,** on deviation 1's tree: the pre-add check, 15 files clean; lint for linux, darwin and windows, `0 issues.`;
  Windows `go test ./...` and `-race`, the second Linux machine's `-race` and the macOS laptop's `go test`, 32 packages ok
  each, no race report; Windows `make preflight`, `preflight passed`.
- **What the plan predicted wrongly, or left unsaid:**
  - Deviation 1: step 5's "logged once" needed the default slog logger, which the lint configuration bans, and the
    first draft failed goconst and revive. A plan's Go is not lint-checked until it is code.
  - `TestClamp` as first written had no gap in its levels, so a Clamp that never searched upward passed it. The agent
    added the gap case before running the negatives; the plan's "up, then down" now has a case that tells them apart.
  - `/thinking` lists and accepts the model's levels too: it is a path of `setConfig`, as step 6's refusal implies.
  - `configOptions` became an `Agent` method, since its levels need the catalog.
- **Next:** F4-2b, made executable as an amendment first.

**2026-10-13 — F4-2b made executable** (the owner's choices of 2026-10-13: `apiKey` is read but waits for F4-3, with a
notice; the ten SDK ids and 0004-MADR's Pi aliases; the model facts honoured and the rest reported; a `!command` run by
the bash tool's shell).

**The facts,** measured on 2026-10-13 against `main` after F4-2a, go-llmprovider-sdk `v1.12.0`, and Pi `312184edb`:

- **Pi's `models.json`** (paths under `packages/coding-agent/src/`):
  - **the schema** is `{"providers": {<id>: provider}}`. A provider has `name`, `baseUrl`, `apiKey`, `api`, `oauth`,
    `headers`, `compat`, `authHeader`, `models[]` and `modelOverrides` (`core/model-config.ts:229-244`).
  - **A model definition** has `id`, `name`, `api`, `baseUrl`, `reasoning`, `thinkingLevelMap`, `input`, `inputLimits`,
    `cost` (with `tiers` of `inputTokensAbove` and four rates), `promptCache`, `contextWindow`, `maxTokens`, `samplingParams`,
    `headers` and `compat`. An override has the same fields but `id`, `api` and `baseUrl`, its cost rates each optional (`:188-227`).
  - **Loading:** no file is no config. A file that cannot be read, parsed or validated is ignored whole, with an error
    naming the file (`:281-320`). Comments and trailing commas are stripped first (`utils/json.ts:2-6`), and a BOM.
  - **Merging** (`core/provider-composer.ts`):
    - a provider entry with none of `baseUrl`, `headers`, `compat`, `modelOverrides`, `models`, `apiKey`, `oauth` or
      `authHeader` is an error (`:290-304`);
    - `baseUrl` applies to the provider's models (`:306-311`);
    - each `models[]` entry replaces a model of its id or is added, with defaults: name the id, reasoning false, input
      text, cost zero, window 128000, output 16384 (`:192-230`, `:312-318`). A window or output of 0 or less is an error.
    - `modelOverrides` apply last, field by field. A cost's rates merge one by one, its tiers replace, and a
      `thinkingLevelMap` merges over the model's (`:163-190`, `:534-554`).
  - **Headers:** the provider's, then the override's, then the definition's, a later one winning (`:376-382`, `:489-508`).
  - **Values** (`core/resolve-config-value.ts`):
    - `!` runs the rest as a shell command and takes its trimmed stdout. Pi uses its configured shell on Windows, else
      `execSync`, with a 10 s timeout (`:153-206`);
    - otherwise `$NAME` and `${NAME}` are environment variables, and `$$` and `$!` are a literal `$` and `!` (`:28-78`);
    - headers and keys resolve at request time, uncached, and a missing variable or a failed command is an error naming
      it (`:221-251`, `:271-282`).
  - **Pi's `api` names:** openai and xai `openai-responses`; anthropic `anthropic-messages`; google `google-generative-ai`;
    huggingface and together `openai-completions`; opencode and opencode-go several (`ai/src/providers/*.ts`).
  - **Pi's base URLs:** the same as the SDK's but Anthropic's, which is `https://api.anthropic.com`
    (`ai/src/providers/anthropic.ts:47`), where the SDK's is `https://api.anthropic.com/v1` (SDK `providers/claude/claude.go:95`).
- **The SDK:** `WithBaseURL` replaces a provider's base, and drops a trailing slash (`llmprovider/options.go:57-59`).
  There is no header option: `WithHTTPClient` is the way in (`:41-43`). A caller layer is a `catalog.Layer`. A later
  `WithModelLayer` ranks above an earlier one, and a value the layer marks with `Known` is its own (`catalog/lookup.go:13-31`).
- **gobble today:**
  - nothing reads `models.json`, and no JSON reader strips comments;
  - `provider.Options` has no base URL or header field (`llm/provider/provider.go:25-43`);
  - the listing is built in `internal/cli/models.go:79-103` and the choice in `internal/cli/agent.go` (`modelChoice`);
  - the bash tool's shell is `findBash`, or the `shellPath` setting, unexported (`tool/builtin/bash.go:20-69`).

**The decisions:**

1. **`<config>/models.json`, read once per process** by `internal/cli`, in Pi's schema. Its whole-file and per-provider
   diagnostics are logged through the CLI's logger, as the settings' are.
2. **Keys:** the ten SDK ids, and `anthropic`, `google` and `xai` as `claude`, `gemini` and `grok` (0004-MADR's table).
   Any other key, or an `api` other than the id's own, fails closed for that provider: it is skipped, with a diagnostic
   naming the SDK version. An id's own `api` is Pi's name for its wire: `openai-responses` for openai and grok;
   `anthropic-messages` for claude; `google-generative-ai` for gemini; `openai-completions` for huggingface, kilo,
   together and ollama; any of the four for opencode-zen and opencode-go.
3. **Honoured:**
   - `baseUrl` is the SDK's base for that provider (`WithBaseURL`), for its turns and its listing. Claude's includes
     `/v1`, which a Pi file's does not; the guide says so;
   - `headers`, provider and model, Pi's order;
   - `models[]` and `modelOverrides` with `name`, `reasoning`, `thinkingLevelMap`, `input`, `cost` with tiers,
     `contextWindow`, `maxTokens` and `headers`, merged as Pi merges them, and Pi's defaults for a defined model.
4. **Reported, not used:** `compat`, `oauth`, `authHeader`, `inputLimits`, `promptCache` and `samplingParams` each get one
   diagnostic per provider. `apiKey` is parsed and validated, and its diagnostic says it takes effect with F4-3.
5. **Values:** header values resolve at request time by Pi's grammar, uncached. A `!command` runs through the bash
   tool's shell (~~`builtin.FindBash`~~ `builtin.BashPath`, deviation 2), `-c`, 10 s, its stdout trimmed and stderr discarded. A missing variable, a failed
   or empty command, or a timeout fails that request, naming the header and the variable or command, never a value.
6. **The facts are a second caller layer,** after the snapshot, so `models.json` outranks it; `models[]` ids join the
   model choice after the listing's.

**F4-2b's steps:**

1. **`llm/catalog/modelsjson.go` (new),** no SDK import:
   - `ParseModelsJSON(data []byte) (*ModelsJSON, []string, error)`: strips a BOM, then `//` comments and trailing commas
     with Pi's two expressions (Go's `regexp` takes both); decodes into typed structs with `encoding/json/v2`; returns a
     whole-file error for bad JSON, a missing or non-object `providers`, or a field of the wrong type. It returns one
     diagnostic per skipped provider or reported field, as decisions 2 and 4 say;
   - `(*ModelsJSON).BaseURL(id) string`, `Headers(id, model string) map[string]string` (the raw values, Pi's order),
     `Models(id) []string` (the `models[]` ids, in order), and `Facts(id, model string) (Facts, bool)`;
   - `Facts` holds each honoured field as a pointer, or `nil` where the file says nothing. A `models[]` entry fills
     every field, Pi's defaults included, and its override is applied over it. `LevelMap map[string]*string` is the
     merged `thinkingLevelMap`.
2. **`llm/catalog/value.go` (new):** `ResolveValue(v string, lookup func(string) (string, bool), run func(string) (string,
   error)) (string, error)`, Pi's grammar above. Its errors name the variables or the command, and never a value.
3. **`llm/llm.go`:** `Model` gains `LevelMap map[string]bool`, the levels `models.json` turns on (true) or off (false).
   **`llm/catalog/catalog.go`:** `Levels` applies it after Pi's rule, when the model reasons, keeping Pi's order.
4. **`llm/provider/model.go`:** `ModelInfo(id, apiKey, model string, mj *gcatalog.ModelsJSON) llm.Model`. A nil `mj` is
   today's lookup. Otherwise a `modelsJSONLayer` (`Name` "models.json"; `Model` converts `Facts` to a `catalog.ModelInfo`
   marked `Known` field by field) is added after the snapshot, and `LevelMap` is set from `Facts`.
5. **`llm/provider/provider.go`:** `Options` gains `BaseURL string`, sent as `WithBaseURL`, and `Headers func(ctx
   context.Context) (map[string]string, error)`. `New` and `List` wrap the client's transport (`o.HTTPClient`'s, else
   `http.DefaultTransport`) so that each request takes `Headers(req.Context())`; an error ends that request, unsent.
   *(F4-2b deviation 1: ~~`Headers func(ctx context.Context)`~~ is `Headers func(ctx context.Context, model string)`.
   The adapter's `Stream` puts `req.Model` in the request's context, and the transport passes it; a listing passes "".
   An unresolvable header is wrapped in the SDK's `ErrInvalidRequest`, so `WithRetry` does not resend it.)*
6. *(F4-2b deviation 2: the export is `BashPath`, since revive refuses a `FindBash` beside the private `findBash`.)*
   **`tool/builtin/bash.go`:** ~~`FindBash(o Options) (string, error)`~~ `BashPath(o Options) (string, error)`, `o.ShellPath` or `findBash()`, exported for
   `models.json`'s commands.
7. **`internal/cli/modelsjson.go` (new):**
   - `(e *runEnv) modelsJSON() *catalog.ModelsJSON`: reads `<config>/models.json` once. No file gives nil, quietly; a
     whole-file error gives nil and one warning; each diagnostic is one warning, `e.log().Warn("models.json",
     slog.String("message", d))`;
   - `runCommand(ctx, command)`: ~~`builtin.FindBash`~~ `builtin.BashPath(e.builtinOptions())` (deviation 2), `exec.CommandContext` with a 10 s timeout,
     stdin closed and stderr discarded; a non-zero exit or empty stdout is an error naming the command;
   - `headersFor(id, model)`: the provider's `Headers` func over `catalog.ResolveValue` with `os.LookupEnv` and `runCommand`.
8. **`internal/cli/agent.go` and `models.go`:** `modelProvider` and the listing pass `BaseURL` and `Headers` for their id.
   `agentOptions`' `ModelInfo` passes `e.modelsJSON()`. `modelChoice` appends `Models(sel.ID)` after the listing's ids,
   each once.
9. **Records:** `docs/guides/models-json.md` (new), the user's page, with an example run against a scratch config;
   `README.md`'s feature list gains `models.json`; `docs/architecture.md`'s `llm/catalog` row names it; this plan;
   `docs/README.md`.

**F4-2b deviation 1 (2026-10-13), found writing step 5.**

- **What was found.** A provider is built once per session with its first model, and `/model` changes the model its
  requests carry. A `Headers func(ctx)` cannot tell which model a request is for, so the per-model headers decision 3
  honours would be the first model's after a change.
- **The owner's decision (2026-10-13):** `Headers` takes the request's model, which `Stream` puts in the request's
  context. Step 5 is annotated above; no file is added to the phase's scope.

**F4-2b deviation 2 (2026-10-13), found by the gates' lint.**

- **What was found.** `golangci-lint` printed 6 issues, the same for linux, darwin and windows, in lines F4-2b added,
  and `make preflight` failed on them:
  - `llm/catalog/modelsjson.go:78-82`: goconst, for Pi's four `api` names, repeated in the wire table;
  - `llm/provider/model.go:140`: `embedlit: embedded field type can be removed from struct literal (modernize)`;
  - `tool/builtin/bash.go:56`: `Method 'findBash' differs only by capitalization to function 'FindBash' (revive)`.
- **The owner's decision (2026-10-13):** the four names become constants; the tier literal sets its rates directly;
  step 6's export is `BashPath`. Step 6 is annotated above; no file is added to the phase's scope.

**F4-2b's tests,** each seen failing first on a scratch copy:

- `llm/catalog`:
  - `TestParseModelsJSON`: comments, trailing commas and a BOM; a whole-file error for bad JSON and for a wrong type;
    an unknown key, a wrong `api`, an empty entry and a window of 0 each skip their provider with a diagnostic; each
    reported field, and `apiKey`, a diagnostic;
  - `TestFacts`: a `models[]` entry with Pi's defaults; an override over the snapshot's fields, with a cost merged rate by
    rate and tiers replaced; an override over a `models[]` entry; `LevelMap` merged;
  - `TestModelsJSONHeaders`: Pi's order; `TestResolveValue`: `$NAME`, `${NAME}`, `$$`, `$!`, a literal, a missing
    variable, a command and a failed one, and no error text holding a resolved value;
  - `TestLevels` gains a `LevelMap` case each way.
- `llm/provider`: `TestModelInfoModelsJSON` over a canned snapshot (`models.json` wins; a `models[]` model has Pi's
  defaults); `TestWireBaseURL` and `TestWireHeaders` over `cannedWire` (the request's host and path; the header sent;
  a failing `Headers` sends nothing).
- `internal/cli`: `TestModelsJSONLoad` (no file; a bad file warned once; diagnostics warned); `TestRunCommand` with the
  bash it finds, skipped with its reason where there is none; `TestModelChoiceModelsJSON` (`models[]` ids after the
  listing's, once each).
- Every existing test passes unedited.

**F4-2b's verification:**

```text
go test -count=1 ./llm/... ./internal/cli/... ./tool/builtin/...   -> ok
the guide's example, against a scratch config directory            -> the output the guide shows
the gates                                                           -> lint ×3, tests on three systems, -race on two, preflight
```

**F4-2b's files:**

| | Files |
| :--- | :--- |
| new, code | `llm/catalog/modelsjson.go`, `modelsjson_test.go`, `value.go`, `value_test.go`; `internal/cli/modelsjson.go`, `modelsjson_test.go` |
| changed, code | `llm/llm.go`; `llm/catalog/catalog.go`, `catalog_test.go`; `llm/provider/model.go`, `model_test.go`, `provider.go`, `list.go`, `wire_test.go` |
| changed, cli and tools | `internal/cli/agent.go`, `models.go`; `tool/builtin/bash.go` |
| new and changed, records | `docs/guides/models-json.md`; `README.md`; `docs/architecture.md`; this plan; `docs/README.md` |

**Deferred, named:**

- **`apiKey`:** F4-3, in Pi's order (decision 4).
- **`compat`, `oauth`, `authHeader`, `inputLimits`, `promptCache`, `samplingParams`:** each waits for an SDK option or a
  gobble feature that uses it; the diagnostic names it.
- **Pi's `opencode` key, and custom providers:** not aliased. `opencode` is `opencode-zen` here, and a provider the SDK
  does not build has no wire.
- **Reloading `models.json` in a running process:** Pi's `/reload` has no gobble counterpart yet.

**F4-2b's execution record (2026-10-13).**

- **Ran on go-llmprovider-sdk `v1.12.0`,** with two deviations, each the owner's decision: the request's model reaches
  `Headers` through the context (deviation 1), and lint's six issues were fixed, the export named `BashPath`
  (deviation 2).
- **Tests:**
  - `llm/catalog`: `TestParseModelsJSON`, `TestFacts`, `TestModelsJSONHeaders` and `TestResolveValue` pass, and
    `TestLevels` passes with its two level-map cases;
  - `llm/provider`: `TestModelInfoModelsJSON`, `TestWireBaseURL` (openai, claude, huggingface) and `TestWireHeaders` pass.
    `TestModelsJSONIDs` also passes: it holds `llm/catalog`'s ten ids to the SDK's built-in providers;
  - `internal/cli`: `TestModelsJSONLoad` (no file, a bad file, a skipped provider), `TestRunCommand` (run, not skipped:
    Git for Windows' bash was found), `TestHeadersFor` and `TestModelChoiceModelsJSON` pass;
  - every existing test passes unedited.
- **Negatives,** 18 of 18 caught on a scratch copy after control runs, each built, each failing its named test with its text:
  - trailing commas kept; no Pi aliases; any `api` accepted; no Pi defaults; an override's rates dropped; headers in the
    wrong order; `$$` not an escape; a command's output in its error; the level map ignored;
  - the layer dropping the window; no level map on the model; no base URL; a header failure retried; no model in the
    context;
  - `models.json`'s models not offered; read on every call; a failed command's exit ignored; a header's name missing
    from its error.
  - All 18 ran again after deviation 2's fix, and were caught again.
- **The guide's example** ran against a scratch `GOBBLE_HOME` with every credential variable cleared, on the final tree.
  It printed the two warnings and the missing-credential error the guide shows, and no request left the machine.
- **Gates,** on deviation 2's tree: the pre-add check, 17 files clean; lint for linux, darwin and windows, `0 issues.`;
  Windows `go test ./...` and `-race`, the second Linux machine's `-race` and the macOS laptop's `go test`, 32 packages ok
  each, no race report; Windows `make preflight`, `preflight passed`.
- **What the plan predicted wrongly, or left unsaid:**
  - Deviation 1: a provider outlives its first model, so step 5's `Headers func(ctx)` could not give per-model headers.
  - Deviation 2: the plan's code failed lint again (goconst, modernize, revive), as F4-2a's did.
  - An unresolvable header is wrapped in the SDK's `ErrInvalidRequest`. Without it, `WithRetry` resent the request
    after waiting (negative 13); the plan's "an error ends that request" needed this to hold.
  - The parsed file is cached per path, not on `runEnv`, so `main.go` stays out of the phase, and each test's config
    directory has its own read.
  - `provider.SDKVersion` reads the SDK's version from the build information, for the diagnostics decision 2 names.
  - A model's own `baseUrl` and the provider's `name` are reported as unused, with the fields decision 4 lists.
  - Two of the agent's own test errors were fixed before the commit. A literal BOM in a Go string, which gofmt refuses,
    became `\uFEFF`. `TestFacts` dereferenced its facts unchecked, so negatives 4 and 5 failed by panic, not by its
    message; it now checks each pointer.
- **Next:** F4-2c, made executable as an amendment first.

**2026-10-14 — F4-2c made executable** (the owner's choices of 2026-10-14: the offline model set, with the live
listing added to `gobble models list`; Pi's custom id for an id the set lacks; the flags winning over a resumed
session's choice, and recorded; the model scope left to 1.x).

**The facts,** measured on 2026-10-14 against `main` `2792461`, go-llmprovider-sdk `v1.12.0` and `v1.13.0`, and Pi
`312184edb`:

- **Pi's choice of a model** (paths under `packages/coding-agent/src/` unless named otherwise):
  - **`resolveCliModel`** (`core/model-resolver.ts:406-606`) resolves `--model` over every model, credential or not
    (`:418-420`), in this order:
    1. no `--model` is no choice; no models at all is `No models available. Check your installation or add models to
       models.json.` (`:414-427`);
    2. `--provider`, without case, must name a provider, else `Unknown provider "<p>". Use --list-models to see
       available providers/models.` (`:429-442`);
    3. without `--provider`, a prefix before the first `/` that names a provider is taken as one (`:449-463`);
    4. still without one, an exact id or `provider/id` across every provider: one match wins; of several, the sole one
       whose provider has a credential, else `Model "<m>" is ambiguous across providers: <list>. <hint> Use --provider
       or provider/model.` (`:470-504`);
    5. with `--provider`, a `<provider>/` prefix on the model is dropped (`:506-512`);
    6. `parseModelPattern` over that provider's models, or all of them (`:514-517`): an exact match, unique and without
       case, as `provider/id`, as provider and id, then as a bare id (`:88-130`); else the models whose id or name holds
       the pattern, an alias (no `-YYYYMMDD` suffix, or `-latest`) before a dated id, the highest id by `localeCompare`
       (`:74-81`, `:136-166`); else, at the last `:`, a level from off to max splits off and the rest is tried again,
       and any other suffix is no match (`:204-257`);
    7. a provider taken from the prefix gives way to the sole credentialed provider holding the whole input as an id,
       when its own has no credential (`:525-540`); with no match in it, the whole input is tried across every provider
       (`:548-568`);
    8. with a provider and still no match, a custom model: the id, with a `:level` split off unless `--thinking` is
       given, and `Model "<id>" not found for provider "<p>". Using custom model id.` (`:570-597`). Pi copies the
       provider's default model's facts onto it (`:175-189`);
    9. otherwise `Model "<m>" not found. Use --list-models to see available models.` (`:599-605`).
  - **`--thinking` wins over a `:level` suffix** (`main.ts:484-489`, `:515-518`). The levels are off, minimal, low,
    medium, high, xhigh and max (`cli/args.ts:60`).
  - **A resumed session:** `--model` wins over the session's model (`core/sdk.ts:198-216`), and `--thinking` over its
    level, clamped to the model's and recorded only when it changes (`main.ts:844-847`,
    `core/agent-session.ts:2527-2549`). Pi records no `model_change` for it (`core/sdk.ts:424-435`).
  - **The settings:** `defaultModel` is used only with `defaultProvider`, as an exact id, and only when that provider
    has a credential; a miss falls through, silently, to the providers' defaults (`core/model-resolver.ts:675-705`).
    `enabledModels`, the scope `--models` also sets, narrows the choice (`main.ts:803-807`).
  - **`--list-models [search]`** (`cli/list-models.ts`) lists the models of the providers with a credential. Each of
    the search's words, split at spaces and `/`, must match `<provider> <id>` as a subsequence, without case, or with
    its letters and digits swapped (`tui/src/fuzzy.ts:12-94`, `:100-138`). The rows are sorted by provider, then id
    (`:57-62`), in the columns `provider model context max-out thinking images`, two spaces apart, the counts as `200K`
    and `1M` (`:14-24`, `:64-114`). No match prints `No models matching "<search>"`.
  - **Pi's generator keeps only the models that call tools** (`ai/scripts/generate-models.ts:1441`, and each
    provider's loop after it).
- **The snapshot,** through the SDK's parser (scratch module `f42cprobe`, three runs): 677 of its 782 models call tools.
  The nine key providers' tool-calling models and their 49 curated ones make 726 candidates, each with a name.
  Parsing, enumerating and one `ModelInfo` each took 8.8 ms warm on Windows (3.6 ms, 1.0 ms and 4.2 ms), inside
  completion's 150 ms (`internal/cli/complete/protocol.go:26`).
- **gobble today:**
  - `--provider`, `--model` and `--thinking` are refused (`internal/cli/flags.go:68-71`), and `--provider` without
    `--model` is a usage error (`:81-82`). `--thinking`'s enum stops at xhigh (`:35`), as `defaultThinkingLevel`'s
    does (`internal/config/keys.go:67`), and completion's levels (`internal/cli/completion.go:48`).
  - The provider is `defaultProvider`'s, or the first with a credential, at five places (`internal/cli/agent.go:34`,
    `:37`, `:103`, `models.go:108`, `line.go:72`). `noCredentialMessage` names `defaultProvider` (`agent.go:263-268`),
    at three places (`line.go:73`, `:219`, `print.go:197`).
  - A new session starts with the model choice's default and `defaultThinkingLevel` (`acpserver/agent.go:311-313`);
    a forked, loaded or resumed one with its history's (`:314-321`, `:350-358`, `:758-762`). `setConfig` refuses a
    model outside the choice (`acpserver/config.go:144-146`). A turn clamps the level to the model's (`agent.go:444`).
  - Completion's provider and model sources list nothing (`internal/cli/completion.go:60-62`). There is no
    `gobble models` (`internal/cli/root.go:6-15`).
  - `max` reaches the SDK as xhigh (`llm/provider/provider.go:357-360`, `llm/doc.go:45`): the SDK has four efforts
    (SDK `llmprovider/contract.go:61-83`).
  - `runEnv` is declared in `internal/cli/main.go:136-164`, so a run's choice puts that file in scope.
  - Adding a key edits two config tests that pin the key list: `TestRenderDocument` (`render_test.go:23-54`) and
    `TestKeysMatchSettings` (`resolve_test.go:89-102`). Every other test that names a key matches part of a line.
- **go-llmprovider-sdk `v1.13.0`** is the latest tag (`git ls-remote --tags`, 2026-10-14), nine commits after
  `v1.12.0`, all native tool deferral (the SDK's 0040-MADR): `Tool.Deferred` and `Tool.Search`, `ToolLoadItem` and
  `WarningDeferralEmulated`, each additive (GitHub's compare API). On a scratch copy of the tree with the pin moved,
  `go build`, `go vet` and `go test ./...` pass, 32 packages ok, nothing edited.

**The decisions:**

1. **The pin moves to `v1.13.0`,** as decision 2 of 2026-10-10, amended 2026-10-11, asks of each sub-phase. Its tool
   deferral is 0012-MADR's to adopt, not F4-2c's.
2. **The offline model set** is each key provider's curated models, then the snapshot's models that call tools, then
   `models.json`'s `models[]`, each once. `--model`, `defaultModel`, `gobble models info` and completion resolve
   against it, with no request. `gobble models list` adds each credentialed provider's live listing.
3. **`--provider` and `--model` resolve by `resolveCliModel`'s steps and words,** over the offline set. 0004-MADR's Pi
   names (`anthropic`, `google`, `xai`) are gobble's ids, and "a credential" is its variable set (F4-3 adds
   `auth.json`). An id the set lacks, under a known provider, is Pi's custom id with Pi's warning. gobble does not copy
   another model's facts onto it, so the catalog does not know it unless `models.json` describes it. A resolution
   error is a usage error, exit 2.
4. **The chosen provider is the run's:** the agent's, the model choice's, the listing's and the credential message's,
   in place of `defaultProvider`'s. The chosen model joins the model choice when the listing lacks it.
5. **The flags win over every session the run opens.** A new session starts with them. A forked, loaded or resumed one
   is set to them after its history, and each change is recorded as a `model_change` or a `thinking_level_change`, as
   `/model` and `/thinking` record one. The level is clamped to the model's, Pi's way. Pi records the level only;
   gobble records both, so a later resume keeps what the run chose.
6. **`--thinking` wins over a `:level` suffix,** and max joins `--thinking`, `defaultThinkingLevel` and completion. It
   reaches the SDK as xhigh until the SDK has a max effort.
7. **`defaultModel`** is a process setting beside `defaultProvider`. With `defaultProvider` set, it is a new
   session's model when the model choice or the offline set holds it. Without `defaultProvider`, or naming a model
   neither holds, it is ignored with one warning; Pi ignores it silently.
8. **`gobble models list [search]`, and `--list-models` as its alias with the prompt words as the search,** print Pi's
   rows and columns; a fact the catalog does not know prints `-`. With no credential set, `No models available. Set
   one of <variables>.`, exit 0, as Pi exits.
9. **`gobble models info <model>`** resolves its argument as `--model` does, and prints that model's facts from the
   offline set.
10. **Completion:** `--provider` offers the key providers, and `--model` the offline set as `provider/id`, with
    `models.json` read and its diagnostics dropped. The levels after `:` include max.
11. **The model scope waits.** Pi's `enabledModels` comes with `--models`, at 1.x (0005-MADR's flag table); the
    2026-10-13 note on choosing a model is annotated.

**F4-2c's steps:**

1. **`go.mod`, `go.sum`:** `go get github.com/maccavelli/go-llmprovider-sdk@v1.13.0`, then `go mod tidy`.
2. **`llm/catalog/resolve.go` (new),** no SDK import:
   - `ProviderID(name string) (string, bool)`: an SDK id or a Pi name, without case, as gobble's id;
   - `type Resolution struct { Model llm.Model; Level string; Custom bool; Warning string }`;
   - `ResolveModel(provider, model, thinking string, models []llm.Model, authed func(string) bool) (Resolution,
     error)`: the facts' steps 1–9 in order, with Pi's words in its errors and warning. `Level` is a `:level`
     suffix's, and a custom model has only `Provider` and `ID`.
3. **`llm/provider/model.go`:** `KnownModels(id string) []string` is `Models(id)`, then the snapshot's models for id
   that set `Tools`, sorted, each once. Its body is ~~`knownModels(id string, meta *catalog.Metadata)`~~
   `toolModels(id string, meta *catalog.Metadata)`, for the test. *(F4-2c deviation 1: revive refuses `knownModels`
   beside `KnownModels`; `ModelInfoContext(ctx, id, apiKey, model, mj)` is added, and `lookupModel` takes the ctx.)*
4. **`acpserver`:**
   - `config.go`: `type Override struct { Provider, Model, Thinking string }`, the run's choice, each field empty when
     not given. `(a *Agent) applyOverride(ctx context.Context, s *liveSession) error` does nothing for a zero
     `Override`. A model that differs from the session's is set, and a `model_change` naming `Override.Provider` is
     appended. Then the level: `catalog.Clamp(a.levelsOf(model), Thinking)` when `Thinking` is set, else the current
     level clamped when the model changed; a level that differs is set and appended as a `thinking_level_change`;
   - `agent.go`: `Options.Override`. `NewSession` starts a session with `cmp.Or(Override.Model, Default)` and
     `cmp.Or(Override.Thinking, ss.Thinking)`, as `defaultThinkingLevel` starts one today, and calls `applyOverride`
     after a fork's `restore`. `open`, for load and resume, calls it after `restore`.
5. **`internal/config`:**
   - `keys.go`: after `defaultProvider`, `{Path: "defaultModel", Kind: KindString, Read: ReadProcess, Doc: "The model a
     new session starts with, of defaultProvider's, its id exactly. Empty takes the provider's first listed model;
     without defaultProvider it is ignored."}`; max ends `defaultThinkingLevel`'s enum;
   - `settings.go`: `DefaultModel`, after `DefaultProvider`, and its `field` case;
   - `settings.yaml`: after `defaultProvider`'s block, `# The model a new session starts with, of defaultProvider's,`,
     `# its id exactly, as gobble models list shows it. Empty takes the`, `# provider's first listed model; without`,
     `# defaultProvider it is ignored. --model wins over it for a run.` and `# defaultModel: ""`.
     `defaultThinkingLevel`'s first line ends `off, minimal, low,`, and its second `medium, high, xhigh or max. A
     resumed session keeps its own, and`, the rest rewrapped;
   - `testdata/settings.schema.json`: regenerated, `go run ./internal/config/genschema`.
6. **`internal/cli`:**
   - `flags.go`: max ends `--thinking`'s enum and help. `ListModels bool` with `name:"list-models"` and `help:"List
     models, as gobble models list; the prompt words are its search."`. `provider`, `model` and `thinking` leave
     `laterFlags`;
   - `modelflags.go` (new): `type modelPick struct { Provider, Model, Thinking string }`.
     `(e *runEnv) pickModel(f *SharedFlags) (modelPick, error)` is `catalog.ResolveModel` over
     `offlineModels(e.modelsJSON())` with `hasCredential`; its warning goes through `e.out.Warnf`, its error is a
     usage error, and its `Thinking` is `cmp.Or(f.Thinking, Level)`. ~~`offlineModels(mj *catalog.ModelsJSON)
     []llm.Model`~~ `offlineModels(ctx context.Context, mj *catalog.ModelsJSON) []llm.Model` is decision 2's set,
     each through ~~`provider.ModelInfo`~~ `provider.ModelInfoContext` with no key *(F4-2c deviation 1)*. `(e *runEnv) chosenProvider()
     string` is the pick's provider, else `defaultProvider`'s. `(e *runEnv) defaultModel(id string, ids []string)
     string` is decision 7, its warning given once. `(e *runEnv) noCredentialMessage() string` is `no model
     credential: --model is <provider>/<model>; set <VAR>` for a pick, and today's two texts otherwise;
   - `main.go`: `runEnv` gains `pick modelPick` and `defaultModelOnce sync.Once`;
   - `chat.go`: after `validate`, `--list-models` runs the list with `strings.Join(c.Prompt, " ")` and returns. After
     `resolveCwd`, `e.pick, err = e.pickModel(&c.SharedFlags)`;
   - `agent.go`, `models.go`, `line.go`, `print.go`: the five places take `e.chosenProvider()`, and the three messages
     are `e.noCredentialMessage()`. `agentOptions` sets `Override` from `e.pick`. `modelChoice` takes
     `defaultModel`'s default, and appends the pick's model and the default when its ids lack them;
   - `modelscmd.go` (new): `ModelsCmd` with `List` and `Info`. `list`'s rows are each credentialed key provider's
     offline set and live listing (`e.modelListing`, every listing started before the first is read), each once, the
     facts from `provider.ModelInfo` with no key; Pi's search, sort, columns and counts, trailing spaces trimmed.
     `info` prints `provider`, `model`, `name`, `context`, `max-out`, `input`, `thinking` (`catalog.Levels`), `cost`,
     and a `tier` line for each tier, aligned, `unknown` for what the catalog does not know;
   - `root.go`: `Models ModelsCmd` with `cmd:""` and `help:"List the models gobble knows, or show one."`;
   - `completion.go`: max ends `thinkingLevels`. `provider` lists `providerCandidates`, each key provider described by
     its variable; `model` lists `modelCandidates`, decision 10's.
7. **Records:** `docs/guides/choosing-a-model.md` (new): how a run's provider, model and level are chosen, `--model`'s
   forms, resuming, `defaultModel`, `gobble models` and completion, each example run against a scratch config;
   `README.md`'s features gain choosing a model; `docs/architecture.md`'s `llm/catalog` row gains "and Pi's model
   resolution (F4-2c)"; this plan; `docs/README.md`.

**F4-2c deviation 1 (2026-10-14), found by the gates' lint.**

- **What was found.** `golangci-lint` printed 11 issues, the same for linux, darwin and windows, all in lines F4-2c added,
  so `make preflight` would fail on them:
  - `internal/cli/completion.go:124:34`: ``Function `offlineModels->ModelInfo->lookupModel` should pass the context
    parameter (contextcheck)``: completion's 150 ms context never reached the SDK's lookup, which used
    `context.Background()`;
  - goconst, `provider` and `model` 3 times each in `internal/cli` (`completion.go:61`, `:63`, `modelscmd.go:105`);
  - gocritic regexpSimplify twice (`modelscmd.go:180-181`, `[0-9]` as `\d`) and equalFold four times
    (`llm/catalog/resolve.go:249`, `:261`, `:272`, `:291`);
  - revive, `llm/provider/model.go:96`: `Method 'knownModels' differs only by capitalization to function 'KnownModels'`.
- **The owner's decision (2026-10-14):** `provider.ModelInfoContext(ctx, id, apiKey, model, mj)` is added, and
  `lookupModel` takes the context; `ModelInfo` keeps its signature and passes `context.Background()`, so
  `acpserver`'s option is unchanged. `lookupModel`'s four calls in F4-2a's and F4-2b's tests (`model_test.go`) pass
  `t.Context()`, edited for the new parameter alone: a wrapper keeping the old signature had only those calls, and
  unparam refused it. `offlineModels(ctx, mj)` takes `e.ctx` in `pickModel`, `gobble models list` and
  `info`, and completion's context in `modelCandidates`. The column labels become constants; `\d`; `strings.EqualFold`;
  the helper is `toolModels`. Steps 3 and 6 are annotated above; no file is added to the phase's scope.

**F4-2c deviation 2 (2026-10-14), found re-running F4-2b's guide example after step 1.**

- **What was found.** `docs/guides/models-json.md:38` shows the example's output as `not a provider go-llmprovider-sdk
  v1.12.0 builds`. The example re-run on F4-2c's tree, against a scratch `GOBBLE_HOME` with every credential cleared,
  prints `v1.13.0`: the diagnostic names the version gobble was built with (`provider.SDKVersion`), so step 1 staled it.
- **The owner's decision (2026-10-14):** the guide writes the version as `<sdk-version>`, as it writes `<config>` for the
  path, with one sentence saying gobble names the go-llmprovider-sdk version it was built with, so a later pin move
  cannot stale it. `docs/guides/models-json.md` is added to F4-2c's scope.

**F4-2c's tests,** each seen failing first on a scratch copy:

- `llm/catalog`: `TestResolveModel`, a case for each step and its words: an exact `provider/id`, without case; a bare
  id held by one provider, and by several with one, none and two credentialed; an unknown `--provider`; `--provider`
  with a `provider/` prefix on the model; a substring of an id, and of a name; an alias before a dated id, and the
  latest dated id; a `:level` suffix, and an unknown suffix; a Pi name as the prefix; a taken provider giving way to a
  credentialed raw id, and the whole input across providers; a custom id, its level split off, and kept in the id
  with `--thinking`; not found; no models. `TestProviderID`.
- `llm/provider`: `TestKnownModels` over a canned document: the curated models first, then the tool-calling ones
  sorted, a model without tools left out, each once.
- `acpserver`: `TestOverrideNew` (the first config options carry the override, and no entry is written);
  `TestOverrideResume` (one `model_change` and one `thinking_level_change`; a second resume with the same override
  writes none); `TestOverrideFork`; `TestOverrideClamp` (max, on a model without it, gives its highest level).
- `internal/config`: `TestRenderDocument` and `TestKeysMatchSettings` take the sixteenth key, edited because they pin
  the key list; `TestSchemaCommitted` passes on the regenerated schema.
- `internal/cli`: `TestPickModel` (a pick reaches `Override` and the model choice; the custom id's warning; Pi's
  errors, exit 2); `TestNoCredentialMessagePick`; `TestDefaultModel` (used; ignored without `defaultProvider`, warned
  once; unknown, warned once); `TestModelsList` (no credential; a canned listing, offline and live rows sorted, `-`
  for an unknown fact; a search; no match); `TestListModelsFlag` (`models list`'s output); `TestModelsInfo` (a
  `models.json` model's facts; an unknown model, exit 2); `TestFuzzyMatch`; `TestTokenCount`; `TestModelCandidates`;
  `TestProviderCandidates`; `TestThinkingLevelsMatchTheSetting` (`defaultThinkingLevel`'s enum is `thinkingLevels`).
- Every other existing test passes unedited. `TestLaterFlags` reads `laterFlags`, and `TestThinkingLevelsMatchTheFlag`
  the enum, so both hold as written.

**F4-2c's verification:**

```text
go test -count=1 ./llm/... ./acpserver/... ./internal/...   -> ok
the guide's examples, against a scratch GOBBLE_HOME          -> the output the guide shows
the gates                                                    -> lint ×3, tests on three systems, -race on two, preflight
```

The guide's examples run with every credential variable cleared, except `OPENAI_API_KEY=test-only` where a listing is
shown, and a scratch `models.json` pointing openai at `http://127.0.0.1:9/v1`, so the listing fails on the machine and
no request leaves it.

**F4-2c's files:**

| | Files |
| :--- | :--- |
| new, code | `llm/catalog/resolve.go`, `resolve_test.go`; `acpserver/override_test.go`; `internal/cli/modelflags.go`, `modelflags_test.go`, `modelscmd.go`, `modelscmd_test.go` |
| changed, code | `go.mod`, `go.sum`; `llm/provider/model.go`, `model_test.go`; `acpserver/agent.go`, `config.go` |
| changed, config | `internal/config/keys.go`, `settings.go`, `settings.yaml`, `testdata/settings.schema.json`, `render_test.go`, `resolve_test.go` |
| changed, cli | `internal/cli/flags.go`, `main.go`, `chat.go`, `agent.go`, `models.go`, `line.go`, `print.go`, `root.go`, `completion.go` |
| new and changed, records | `docs/guides/choosing-a-model.md`; `README.md`; `docs/architecture.md`; this plan; `docs/README.md` |

**Deferred, named:**

- **`enabledModels` and `--models`:** 1.x (decision 11).
- **Ollama:** `--provider` and `defaultProvider` take the providers with a key (0019-MADR D15). Ollama waits for a
  keyless choice, and is an unknown provider until then.
- **max as its own effort:** it needs an SDK effort above xhigh (decision 6), a request for the owner to file with the
  ranking request of 2026-10-13.
- **The SDK's native tool deferral:** 0012-MADR's, whose `tool_search` it would carry (decision 1).
- **A custom id with another model's facts:** Pi's, not copied (decision 3); `models.json` describes a model instead.
- **`/model` by Pi's rules, and a choice saved to the settings:** Pi's picker and its `persist` belong to the TUI (F9).
- **`gobble models list --json`:** no consumer yet.
- **`--api-key`:** F4-3.

**F4-2c's execution record (2026-10-14).**

- **Ran on go-llmprovider-sdk `v1.13.0`** (step 1), with two deviations, each the owner's decision: lint's 11 issues
  fixed, the context passed through `ModelInfoContext` (deviation 1); the `models.json` guide's SDK version written as
  `<sdk-version>` (deviation 2).
- **Tests:**
  - `llm/catalog`: `TestResolveModel` (22 cases, one per step and word), `TestResolveModelEmpty` and `TestProviderID`;
  - `llm/provider`: `TestKnownModels`;
  - `acpserver`: `TestOverrideNew`, `TestOverrideResume`, `TestOverrideFork` and `TestOverrideClamp`;
  - `internal/cli`: `TestPickModel`, `TestNoCredentialMessagePick`, `TestDefaultModel`, `TestThinkingLevelsMatchTheSetting`,
    `TestModelsList`, `TestListModelsFlag`, `TestModelsInfo`, `TestFuzzyMatch`, `TestTokenCount`, `TestModelCandidates`
    and `TestProviderCandidates`;
  - all 19 pass, each seen by name in a `-v` run. `TestRenderDocument` and `TestKeysMatchSettings` were edited for the
    sixteenth key, as planned, and `lookupModel`'s four calls in `model_test.go` for its context (deviation 1). Every
    other existing test passes unedited.
- **Negatives,** 30 of 30 caught on a scratch copy after control runs, each built, each failing its named tests with its
  text:
  - `llm/catalog`: no Pi names; no credentialed tie-break; no prefix stripped; no alias preferred; no level split; a
    taken provider never giving way; a custom id ignoring `--thinking`; names not searched; Pi's hint reworded;
  - `llm/provider`: models without tools kept; curated models twice;
  - `acpserver`: no `model_change`; the override's level unclamped; no clamp on a model change; not applied on resume;
    not applied to a new session; a level recorded unchanged;
  - `internal/cli`: the pick not offered; no override; `defaultModel` used without `defaultProvider`, and warned each
    time; the credential message ignoring `--model`; the live listing left out; rows unsorted; an unknown count as 0;
    no letters and digits swapped; `--list-models` without its search; info without tiers; completion without
    `models.json`; the setting without max.
  - The first run caught 26. Three misses were the harness's (an expected text another case caught, and a mutation
    that left an import unused, so the copy did not build), and were corrected. One was the test's:
    `TestOverrideClamp` read only `thought_level`, which clamps for display, so an unclamped recorded level passed it.
    It, and `TestOverrideResume`, now check the recorded level; all 30 ran again and were caught.
- **The guides' examples** ran against a scratch `GOBBLE_HOME` with every credential variable cleared, on the final
  code; the listing examples set `OPENAI_API_KEY=test-only` with openai's base URL at `http://127.0.0.1:9/v1`, so the
  listing failed on the machine and no request left it. `choosing-a-model.md` shows their output, without timestamps.
  F4-2b's example, re-run, printed `v1.13.0`, which deviation 2 addresses.
- **Gates:** the pre-add check, 24 files clean; lint for linux, darwin and windows, `0 issues.`; Windows `go test ./...`
  and `-race`, the second Linux machine's `-race` and the macOS laptop's `go test`, 32 packages ok each, no race report;
  Windows `make preflight`, `preflight passed`.
- **What the plan predicted wrongly, or left unsaid:**
  - Deviation 1: the plan's code failed lint a fourth time, and contextcheck found a real gap: completion's 150 ms
    deadline never reached the SDK's lookup.
  - Deviation 2: moving the pin staled a guide that quotes a diagnostic naming the SDK's version.
  - `gpt-6-luna` is held by openai, opencode-go and opencode-zen, so the bare id is Pi's ambiguity error without a
    credential; the guide shows it.
  - Pi's errors are capitalised sentences, which revive's `error-strings` refuses from `errors.New`; they are a
    `catalog.ResolutionError`, so the words stay Pi's.
  - `--provider` also strips its own spelling as a prefix, so `--provider anthropic --model anthropic/x` works; Pi's
    strips only its canonical name, which for these providers is the same word.
  - `catalog.CompareIDs`, the case-folded order standing for Pi's `localeCompare`, is exported for `gobble models
    list`'s sort.
- **Next:** F4-3, credentials, made executable as an amendment first.

**2026-10-14 — F4-3 re-made as three sub-phases, F4-3a made executable** (the owner's choices of 2026-10-14: three
sub-phases; the vendors' terms checked when F4-3b is made executable; `gobble auth token` for OAuth alone, and no command
printing a stored key; a stored key in Pi's value grammar).

**The facts,** measured on 2026-10-14 against `main` `dce2a97`, go-llmprovider-sdk `v1.13.0` (still the latest tag,
`git ls-remote --tags`), acp-go-sdk2 `v0.17.1` and Pi `312184edb`:

- **gobble today:**
  - `internal/auth` is a declared package only (`internal/auth/doc.go:1-5`), imported by nothing.
  - A key is read only from the provider's variable: `provider.Choose` and `Ambient` take `os.Getenv`
    (`llm/provider/ambient.go:31-48`, `:91-102`), at `internal/cli/agent.go:36`, `:39`, `:63`, `:110`, `models.go:85`,
    `:108`, `modelflags.go:40-42`, `modelscmd.go:65` and `line.go:72`. `provider.Options.APIKey` is the only credential
    field (`llm/provider/provider.go:26-29`), sent as `WithAPIKey` (`:128-130`).
  - `--api-key` is parsed and refused (`internal/cli/flags.go:66-72`). A rejected key and no key both reach the CLI as
    `acpclient.ErrAuthRequired` (`acpclient/conn.go:283-284`) and print `noCredentialMessage` (`line.go:217-220`,
    `print.go:196-197`), the defect 0019-PLAN P4 deviation 7 noted.
  - `models.json`'s `apiKey` is parsed and reported as waiting for F4-3 (`llm/catalog/modelsjson.go:169-170`), which
    `TestParseModelsJSON` pins (`modelsjson_test.go:66`) and the guide shows (`docs/guides/models-json.md:37-38`, `:88`).
  - **0015-REPORT's lock correction is already met:** `3547163` moved `session/jsonl`'s lock to `internal/filelock`
    (`Lock(ctx, path, Policy)`, `TryLock`, `EditPolicy`, `internal/filelock/lock.go:24-59`), which `internal/config` and
    `mcpclient` use for their edits. `internal/appdirs` has `EnsurePrivateDir`, `FileIsOwnerOnly`, and on Windows
    `SecurePrivateFile` (`ensure_unix.go:24`, `ensure_windows.go:26`, `security_windows.go:279`, `:300`).
  - archtest rule 4 lets only `llm/provider` import go-llmprovider-sdk (`internal/archtest/check.go:238-254`), so
    `internal/auth` cannot implement the SDK's `TokenStore`; an adapter in `llm/provider` must.
- **go-llmprovider-sdk `v1.13.0`** (paths under `llmprovider/`):
  - `auth.TokenStore` is `Load`, `Save` and `Delete` of `*OAuthSession` by provider, and `RefreshLocker` an optional
    `LockRefresh`, so a refresh token is never spent twice (`auth/tokenstore.go:132-145`). `OAuthSession` has `Issuer`,
    `ClientID`, `AccountID`, `FedRAMP` and `TokenURL` besides the tokens (`:21-64`); its JSON redacts the tokens, so it
    is no file format (`:92-110`).
  - Browser logins exist for openai and grok, device logins for openai, grok and kilo (`auth/oauth_loopback.go:91`,
    `:184-204`; `auth/oauth_device.go:106-117`). The default client ids are the Codex and Grok CLIs' (`oauth_constants.go:5-13`),
    which the SDK's own doc calls "borrowed" (`auth/doc.go:8-11`). Kilo's device login gives a key (`kilo_device.go:119-122`).
  - Only openai, grok and kilo take an OAuth session; claude, gemini, huggingface, opencode-zen, opencode-go and together
    take a key alone (e.g. `providers/claude/claude.go:149-151`).
  - A 401 or 403 is `ErrAuthFailure` on an `*APIError` (`api_error.go:183-184`); a missing key is `ErrInvalidRequest` at
    `New` (`providers/openai/openai.go:148`). There is no "no credential" sentinel.
- **Pi** (paths under `packages/coding-agent/src/` unless named):
  - **`auth.json`** is one object keyed by provider (`core/auth-storage.ts:17`): `{"type": "api_key", "key", "env"?}` or
    `{"type": "oauth", "access", "refresh", "expires", ...}`, each provider's extra fields kept (`ai/src/auth/types.ts:17-34`).
    The file is created `0o600`, the mode "only on creation so administrator-managed modes and ACLs remain intact", in a
    directory made `0o700` (`core/auth-storage.ts:24-25`, `:59`, `:65`). Every change takes `auth.json`'s lock, re-reads,
    merges the one provider and writes the whole file (`:449-482`); the async lock waits up to 30 s (`:116-155`).
  - **The order** (`ai/src/auth/resolve.ts:28-92`, `core/runtime-credentials.ts:24-28`, `core/provider-composer.ts:426-451`):
    the run's `--api-key`, kept in memory, never written; then the stored entry, which "owns the provider: ambient/env is
    consulted only when nothing is stored"; then `models.json`'s `apiKey`, a value whose variables are unset counting as
    none; then the provider's variable.
  - **Values:** a stored key goes through `resolveConfigValue`, a `!command` run once and cached; `models.json`'s key
    through `resolveConfigValueOrThrow`, uncached (`core/resolve-config-value.ts:145-151`, `:208-230`).
  - **`--api-key <key>`** is a literal from argv (`cli/args.ts:118-119`), and needs a model: "--api-key requires a model
    to be specified via --model, --provider/--model, or --models" (`main.ts:821-829`).
  - **`pi auth`** has `check`, `print-api-key` and `print-bearer-token` (`cli/auth-command.ts:51-58`); login and logout
    are the TUI's `/login` and `/logout`, saying `Saved API key for <name>`, `Removed stored API key for <name>.
    Environment variables and models.json config are unchanged.` and, with none, `No stored credentials to remove. ...`
    (`modes/interactive/interactive-mode.ts:5873`, `:5898`, `:5924`). `check` reports `ready`, `not_ready` or `invalid`,
    with a reason, and exits 0, 1 or 2 (`cli/auth-check.ts:8-51`, `main.ts:188-202`).
  - Pi has no Google login, and imports no other tool's credentials.
- **acp-go-sdk2 `v0.17.1`:** a terminal auth method listed without the client's `auth.terminal` fails `initialize` on both
  sides (`types_gen.go:19956-19968`); `authenticate` refuses a terminal method (`:19920-19927`); `logout` needs
  `agentCapabilities.auth.logout` (`:19805-19810`). gobble's client sends no capabilities (`acpclient/conn.go:121-124`).
  This is F4-3c's, as 0016-MADR D11 amends it.

**The decisions:**

1. **Three sub-phases,** each one commit with its own record and gates, each made executable as an amendment first:
   - **F4-3a — keys:** `auth.json`, the order, `--api-key -`, `models.json`'s `apiKey`, `gobble auth login --api-key -`,
     `logout`, `status` and `check`, and a rejected key reported as one;
   - **F4-3b — OAuth logins:** the terms checks, recorded before any login is built (decision 2); the browser and device
     logins; `gobble auth token`; an `oauth` entry used, through an `llm/provider` adapter that implements the SDK's
     `TokenStore` and `RefreshLocker` over `auth.json`, keeping `FedRAMP` (0015-REPORT F5);
   - **F4-3c — ACP and import:** ACP auth as 0016-MADR D11 amends it, and `gobble auth import --from-pi`.
2. **The terms checks** are F4-3b's first step: OpenAI's, xAI's and Kilo's current terms for third-party clients, each
   URL, its date and what it says. A login they do not allow goes to 1.x and back to the owner.
3. **`gobble auth token <provider> [--min-expiry D]`** prints an OAuth bearer token, as Pi's `print-bearer-token`, and
   lands with F4-3b. No command prints a stored key, and `check` has no `--credentials`.
4. **`auth.json`** is `<config>/auth.json`, in Pi's shape. Each entry, and each field gobble does not read, is kept as
   written. It is created owner-only (0600; on Windows the user's DACL, `appdirs.SecurePrivateFile`), in a directory
   made 0700 only when it is created, so an existing directory's mode stays as it is, as Pi's does. Every change takes
   `auth.json.lock` through `internal/filelock` (30 s, as Pi's), re-reads, sets or removes the one provider, and renames a
   temporary file into place. A file that cannot be parsed is an error naming it, and is never overwritten.
5. **The order is Pi's:** `--api-key -`, for the run only and never written; then `auth.json`'s entry, which owns its
   provider; then `models.json`'s `apiKey`; then the provider's variable. In F4-3a an `oauth` entry owns its provider
   and fails closed, naming F4-3b. One lookup replaces each place a variable is read today.
6. **Values:** a stored key takes Pi's grammar through `catalog.ResolveValue` and the bash tool's shell, a command's
   result cached for the process; `models.json`'s key the same grammar, uncached, resolved when the provider is built.
   A `models.json` key whose variables are unset is no key. An error names the provider and the variable or command,
   never a value.
7. **`--api-key -`** needs a model, in Pi's words without `--models` (1.x). It reads the key from stdin, all of it,
   trimmed, when stdin is not a terminal, so the prompt must then come from words, `--file` or `@path`; on a terminal it
   asks, without echo. A literal is refused with the reason.
8. **`gobble auth`:**
   - `login <provider> --api-key -` stores a key, saying Pi's `Saved API key for <provider>`. Without `--api-key` it is
     refused, naming F4-3b, until the OAuth logins land;
   - `logout <provider>` removes the entry, in Pi's two messages, `/logout` read as `gobble auth logout`;
   - `status` lists each key provider's source (`auth.json`, `models.json`, its variable, or none) and an OAuth entry's
     expiry, never a value;
   - `check (--provider P | --model M) [--json]` is Pi's `check`: `ready`, `not_ready` or `invalid`, a reason, exit 0, 1
     or 2.
9. **A rejected key is reported as one:** `provider.Options.CredentialSource` names the key's source, and an
   `ErrAuthFailure` from the provider's response is `llm.ErrAuth` saying `<provider> rejected the API key from <source>`.
   The CLI prints that when the run's provider had a key, and today's message when it had none.

**F4-3a's steps:**

1. **`internal/auth/store.go` (new):**
   - `type Entry struct { Type, Key string; raw map[string]jsontext.Value }`: an entry's `type` and, for an `api_key`,
     its `key`; `raw` keeps every field, read or not, so a write gives back what it read;
   - `type File struct` and `Load(path string) (*File, error)`: no file is an empty `File`; bad JSON, a top level that is
     not an object, or an entry without a string `type` is an error naming the file, as Pi's
     ``Invalid auth.json credential for provider "<p>"``;
   - `(*File) Entry(provider string) (Entry, bool)` and `Providers() []string`;
   - `Set(ctx, path, provider string, e Entry) error` and `Delete(ctx, path, provider string) (bool, error)`: under
     `filelock.Lock(ctx, path+".lock", authPolicy)`, `authPolicy` 20 ms first, 2 s most, 30 s in all; re-read; change the
     one provider; write a temporary file in the same directory, owner-only before its first byte, and rename it over
     `path`. The directory is made 0700 when absent.
2. **`internal/auth/secure_windows.go`, `secure_other.go` (new):** `securePrivate(path string) error`, the file's owner-only
   protection: `appdirs.SecurePrivateFile` on Windows; elsewhere the 0600 the file was opened with, so nothing more.
3. **`internal/auth/resolve.go` (new):**
   - `type Source string` with `SourceFlag` ("--api-key"), `SourceFile` ("auth.json"), `SourceModelsJSON` ("models.json"),
     and a variable's name for one from the environment;
   - `type Credential struct { Key string; Source Source }`;
   - `type Resolver struct { Runtime map[string]string; File *File; ModelsJSONKey func(id string) string; Var func(id
     string) string; LookupEnv func(string) (string, bool); Run func(string) (string, error) }`;
   - `(*Resolver) Credential(id string) (Credential, bool, error)`: decision 5's order and decision 6's values; an `oauth`
     entry is the error `provider <id> has an OAuth login in auth.json; gobble uses OAuth logins from 0005-PLAN F4-3b`;
     a stored key's command result cached in the `Resolver`.
4. **`llm/catalog/modelsjson.go`:** `(*ModelsJSON) APIKey(id string) string`, the raw value; the `apiKey` diagnostic goes.
5. **`llm/provider`:**
   - `ambient.go`: `ChooseWith(has func(id string) bool, preferred string) (Selection, error)` and `AmbientWith(key func(id
     string) (string, bool), preferred string, o Options)`, the same rules over a credential lookup; `Choose` and `Ambient`
     become them over the variables, so their tests stand;
   - `provider.go`: `Options.CredentialSource string`; `mapError` wraps an `*APIError` whose kind is `ErrAuthFailure` as
     `fmt.Errorf("%w: %s rejected the API key from %s", llm.ErrAuth, id, source)` when the source is set.
6. **`internal/cli`:**
   - `credentials.go` (new): `(e *runEnv) credentials() *auth.Resolver`, built once: `auth.json` read once, `--api-key`'s
     key, `models.json`'s keys, the variables, and `runCommand`; `(e *runEnv) credential(id string) (auth.Credential, bool)`,
     a resolution error warned once; `readAPIKey(stdin, tty)` for `--api-key -`;
   - `agent.go`, `models.go`, `modelflags.go`, `modelscmd.go`, `line.go`: each variable read takes `e.credential`; the
     provider is built with `AmbientWith` and `CredentialSource`, so `ambientProvider` is `provider.AmbientWith`;
     `hasCredential` becomes `e.hasCredential`;
   - `flags.go`: `api-key` leaves `laterFlags`; `validate` refuses `--api-key` other than `-` (`--api-key takes -: a key on
     the command line shows in process listings and shell history; pipe it, or type it when asked`) and `--api-key`
     without `--model` (`--api-key requires a model to be specified via --model or --provider/--model`);
   - `chat.go`: after `validate`, `--api-key -` reads the key before the input is composed, and stdin then carries none;
   - `line.go`, `print.go`: on `ErrAuthRequired`, the agent's message when the run's provider had a key, else
     `noCredentialMessage`;
   - `auth.go` (new): `AuthCmd` with `Login`, `Logout`, `Status` and `Check`, as decision 8; `root.go` gains
     `Auth AuthCmd` (`cmd:""`, `help:"Store, remove and check model credentials."`); `main.go`'s `runEnv` gains the
     resolver and its once.
7. **Records:** `docs/guides/credentials.md` (new): where keys come from, in order, `auth.json`, `--api-key -`,
   `gobble auth`, and what a rejected key says, each example run against a scratch config; `docs/guides/models-json.md`:
   `apiKey` takes effect, and the example's output loses its `apiKey` line; `README.md`'s features gain credentials;
   `docs/architecture.md`'s `internal/auth` row; this plan; `docs/README.md`.

**F4-3a's tests,** each seen failing first on a scratch copy:

- `internal/auth`: `TestStoreRoundTrip` (an unread provider and an unread field kept byte for byte through a `Set`);
  `TestStoreConcurrent` (two writers, of different providers, both kept); `TestStoreMode` (0600 on Unix, owner-only on
  Windows, through `appdirs.FileIsOwnerOnly`); `TestStoreBadFile` (an error naming the file; the file untouched);
  `TestResolveOrder` (each source winning over the ones after it; an `auth.json` entry owning its provider over a set
  variable; an `oauth` entry failing closed; a `models.json` key with an unset variable as none; a stored `!command` run
  once).
- `llm/provider`: `TestChooseWith` (a stored key chooses a provider whose variable is unset); `TestWireRejectedKey` (a 401
  through `cannedWire` names the provider and the source, and is `llm.ErrAuth`).
- `llm/catalog`: `TestParseModelsJSON`'s `apiKey` case edited, now that the field is used: no diagnostic, and `APIKey`
  gives the value.
- `internal/cli`: `TestAPIKeyFlag` (`-` from stdin; a literal refused; without `--model` refused; never written);
  `TestAuthLogin`, `TestAuthLogout` (Pi's messages; the file's other entries kept); `TestAuthStatus` (each source, and no
  value in the output); `TestAuthCheck` (the three statuses, exits and `--json`); `TestRejectedKeyMessage` (the agent's
  message with a key, the old one without).
- `internal/cli`'s `cache_test.go` is edited for `ambientProvider`'s new type alone: its replacement
  (`cache_test.go:22-27`) takes `AmbientWith`'s signature, and checks what it checked.
- Every other existing test passes unedited.

**F4-3a's verification:**

```text
go test -count=1 ./internal/... ./llm/... ./acpserver/...    -> ok
the guides' examples, against a scratch GOBBLE_HOME           -> the output they show, no request leaving the machine
the gates                                                     -> lint ×3, tests on three systems, -race on two, preflight
```

**F4-3a's files:**

| | Files |
| :--- | :--- |
| new, code | `internal/auth/store.go`, `store_test.go`, `secure_windows.go`, `secure_other.go`, `resolve.go`, `resolve_test.go`; `internal/cli/credentials.go`, `auth.go`, `auth_test.go` |
| changed, code | `internal/auth/doc.go`; `llm/catalog/modelsjson.go`, `modelsjson_test.go`; `llm/provider/ambient.go`, `ambient_test.go`, `provider.go`, `wire_test.go` |
| changed, cli | `internal/cli/flags.go`, `main.go`, `chat.go`, `agent.go`, `models.go`, `modelflags.go`, `modelscmd.go`, `line.go`, `print.go`, `root.go`, `cache_test.go` |
| new and changed, records | `docs/guides/credentials.md`; `docs/guides/models-json.md`; `README.md`; `docs/architecture.md`; this plan; `docs/README.md` |

**Deferred, named:**

- **OAuth logins, `token`, and an `oauth` entry used:** F4-3b, after its terms checks (decisions 1-3).
- **ACP auth and `gobble auth import --from-pi`:** F4-3c (decision 1).
- **`--models`:** 1.x, so `--api-key`'s message leaves it out.
- **Pi's `api_key` `env` field** (provider-scoped values, such as Cloudflare's ids): kept as written, unused; no SDK
  provider takes one.
- **The OS keyring:** 1.x, as decision 4 of 2026-10-10 says.
