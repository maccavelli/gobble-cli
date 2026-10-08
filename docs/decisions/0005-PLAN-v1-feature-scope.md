---
status: in-progress
date: 2026-10-07
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

*(2026-10-07, 0005-MADR amendment "the best of several harnesses": choice 4, a new item: formatters and LSP diagnostics after edit, write and `apply_patch`, with its own MADR first. It needs an LSP client, server discovery and per-language formatter settings (0010-REPORT §1).)*

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
