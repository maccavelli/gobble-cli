---
status: proposed
date: 2026-10-06
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
5. Project trust (`trust.json`, the decision order, `gobble trust|untrust`,
   `/trust`) and the built-in protected paths.

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
2. The usage ledger, `/usage` (session plus today), and `gobble usage`.
3. `gobble doctor [--json]`.
4. `gobble update [--check]` through
   `github.com/maccavelli/go-core-lib/selfupdate` `v1.1.0`
   (`96b30961180671ab3697585951219001ecbb1c90`). Bind it as
   `selfupdate/example_test.go` does for a standalone program:
   `NewGitHubSource` (User-Agent from 0003, repository owner/name of
   this module's GitHub), `NewStrictVersionPolicy`,
   `NewExactAssetSelector` for the six Phase 4 platforms,
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

None. This plan is proposed and has not been approved.

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
