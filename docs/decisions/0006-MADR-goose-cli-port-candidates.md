---
status: proposed
date: 2026-10-04
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Adopt goose's terminal-CLI mechanics, not its product command tree

## Context and Problem Statement

Gobble's default mode is a native terminal CLI. The library is Kong. TUI mode is a separate program surface and uses go-tui-lib; gobble does not reimplement a core TUI. Self-update is go-selfupdate-lib; gobble does not reimplement self-update. The language is Go 1.27.1, and new behavior comes from sibling libraries and the standard library, not from a local copy of another project's code.

Goose (`aaif-goose/goose`) ships a large terminal binary, `goose`, whose command tree, session prompt, completion, configure flow, skills listing, and updater are all in one clap program. The desktop UI is out of scope. The question this record answers is which of those terminal-CLI ideas are worth leveraging in gobble's native CLI, and which are not.

This file is a proposal. Nothing here is accepted. No PLAN is paired with it.

### What was measured, not assumed

The goose checkout on MAC420 was read, not executed.

- `git rev-parse HEAD` in `gitrepos/goose` returned `591edd47cf2cfea4957d720c607cf2a4def8673d`. `git rev-parse --abbrev-ref HEAD` returned `main`. `git remote get-url origin` returned `https://github.com/aaif-goose/goose.git`. `git status -sb` showed `## main...origin/main` and no dirty paths. `git log -1` subject was `fix(tom): read messages from config (#12634)`, committed `2026-10-02 15:03:24 +0000` (2026-10-02 10:03 CT). That is the tree assessed. It matches the requested prefix `591edd4`.
- The binary `goose` was not run. Behavior below is what the source says it will do, not a captured CLI transcript.
- `crates/goose-cli` is the binary. `Cargo.toml` names the bin `goose` at `src/main.rs` and depends on `clap`, `clap_complete`, `clap_complete_nushell`, `rustyline`, and `cliclack`. The `goose` library crate is a path dependency.
- `Cli` is a clap `Parser`. `command` is `Option<Command>`. `cli()` matches `None` to `handle_default_session` (`crates/goose-cli/src/cli.rs`).
- `handle_default_session` calls `handle_configure` and returns when `Config::global().exists()` is false. Otherwise it builds an interactive session (`cli.rs`).
- `Command` variants read in `cli.rs`: `Configure`, `Info`, `Doctor`, `Mcp` (feature `bundled-mcp`), `Acp`, `Roam` (feature `roaming`), `Serve` (feature `acp-http`), `Session` (visible alias `s`), `Run`, `Recipe`, `Skills`, `Plugin`, `Schedule` (feature `scheduler`, alias `sched`), `Gateway` (alias `gw`), `Update` (feature `update`), `Term`, `LocalModels` (feature `local-inference`, alias `lm`), `Completion`, `Review`, `ValidateExtensions` (`hide = true`), `McpProbe` (`hide = true`).
- `Session` and `Run` both flatten `SessionOptions`, `ExtensionOptions`, and `ModelOptions`. `Run` also flattens `InputOptions`, `RunBehavior`, and `OutputOptions` (`cli.rs`).
- `InputOptions` makes `instructions`, `input_text` (`--text`), and `recipe` mutually conflicting. Help text says `instructions` may be `-` for stdin (`cli.rs`).
- `OutputOptions.output_format` allows `text`, `json`, and `stream-json`, default `text`. `RunBehavior` has `interactive`, `no_session`, `resume`, and `stats` (`cli.rs`).
- `SessionCommand` is `List`, `Remove`, `Export`, `Import`, `Diagnostics`, `Rename`. Session flags include `resume`, `fork` (`requires = "resume"`), `edit` (`requires = "resume"`), and `history` (`requires = "resume"`) (`cli.rs`).
- `CompletionShell` is `Bash`, `Elvish`, `Fish`, `Powershell` (alias `pwsh`), `Nu` (alias `nushell`), `Zsh`. `cli()` writes the script to stdout via `shell.generate` (`cli.rs`). A unit test parses `completion nushell`.
- `get_input` in `crates/goose-cli/src/session/input.rs` uses rustyline. Empty input retries. `exit` and `quit` exit. A leading `/` goes to `handle_slash_command`. `None` from that function becomes `InputResult::Message` (the slash text is sent as a chat message).
- Slash names matched in `handle_slash_command`: `/exit`, `/quit`, `/?`, `/help`, `/t`, `/prompts`, `/prompt`, `/extension`, `/builtin`, `/mode`, `/model`, `/clear`, `/new`, `/compact`, `/skills`, `/summarize` (prints a rename warning, then compacts), `/r`, `/edit`. Help text also documents `/status`. Extra builtin lines are appended from `goose::agents::execute_commands::list_commands`, excluding a hardcoded documented set (`input.rs`).
- `crates/goose/src/slash_commands/slash_command.rs` `list_builtin_commands` is covered by a test that expects the names `prompts`, `prompt`, `compact`, `clear`, `skills`, `doctor`, `goal`, `grind`, `status`. `merge_command_sources` tests give builtins priority over recipes and skills, and recipes priority over skills, matched on a normalized name. This module's `list_acp_commands` is the ACP merge. The CLI parser in `input.rs` does not call it.
- `crates/goose-cli/src/session/paste.rs` states that rustyline uses bracketed paste off Windows, and that Windows console pastes arrive as a key-event burst. The module collapses a multi-line or long paste into a chip and restores it on submit.
- `handle_configure` in `crates/goose-cli/src/commands/configure.rs` returns an error when stdin is not a terminal, then branches on `config.exists()` into first-time setup or existing config. `Command::Configure` has no fields and no subcommand (`cli.rs`). The editor help string in `input.rs` tells the user to run `goose configure set`. That subcommand is not on `Command`.
- `handle_skills_list` in `crates/goose-cli/src/commands/skills.rs` calls `list_installed_skills`, sorts by name, and prints a width-capped table: name, description preview (50 characters), description tokens, content tokens, location.
- `list_installed_skills` in `crates/goose/src/skills/mod.rs` delegates to discovery. `all_skill_dirs_with_config` pushes, for a working directory, `.agents/skills`, `.goose/skills`, and `.claude/skills`, then project plugin skill dirs, then home `.agents/skills`, the goose config `skills` directory, home `.claude/skills`, home `.config/agents/skills`, and user plugin skill dirs. The comment on `all_skill_dirs` says project dirs come before global dirs. Files visited are named `SKILL.md`. `validate_skill_name` requires a non-empty name of at most 64 characters, ASCII lowercase, digits, and hyphens, and rejects a leading or trailing hyphen.
- `update` in `crates/goose-cli/src/commands/update.rs` bails when feature `disable-update` is set, bails on `riscv64` because no release artifacts are published, otherwise downloads `stable` or `canary` from `https://github.com/aaif-goose/goose/releases/download/{tag}/{asset}`, verifies SLSA provenance via Sigstore, extracts with path-traversal checks, replaces the current executable, copies DLLs on Windows, and if `reconfigure` is set runs the current executable with argument `configure`. `Cargo.toml` puts `update` in default features and defines an empty `disable-update` feature. The `update` feature pulls in `sigstore-verify` and `snap`.
- `handle_doctor` in `crates/goose-cli/src/commands/doctor.rs` builds a session with `no_session: true` and `interactive: true`, then calls `session.interactive(Some("/doctor".to_string()))`.
- `handle_info` in `crates/goose-cli/src/commands/info.rs`, when `check` is true, calls `check_provider` and returns an error if that check fails. The comment in that function says the non-zero status is so `goose info --check` can be a pre-flight verifier.
- `main.rs` on Windows calls `enable_windows_vt_processing` before the Tokio runtime. That function calls `console::Term::stdout()` and `console::Term::stderr()` `colors_supported()`. The comment says this sets `ENABLE_VIRTUAL_TERMINAL_PROCESSING` so spinners do not print as repeated lines. The CLI thread is started with `stack_size(8 * 1024 * 1024)`, an 8 MiB stack.
- `parse_run_input` in `cli.rs`: `--instructions` of `-` reads stdin with `.expect("Failed to read from stdin")` and keeps `input_opts.system`; any other path sets `additional_system_prompt` to `None` and calls `std::process::exit(1)` if the file cannot be read; `--text` keeps `system`.
- `Paths::path_root` reads `GOOSE_PATH_ROOT` and `validated_path_root` keeps only an absolute path (`crates/goose/src/config/paths.rs`).
- `setup_logging` in `crates/goose-cli/src/logging.rs` sets `console: false` and `json: true`. `prepare_log_directory(component, true)` joins a `%Y-%m-%d` directory, and `cleanup_old_logs` deletes directories older than `14 * 24 * 60 * 60` seconds (`crates/goose/src/logging.rs`). Neither file mentions redaction.
- `Agent::reply` calls `execute_command` only when `use_state_machine` is false (`crates/goose/src/agents/agent.rs`). The match is in `crates/goose/src/agents/execute_commands.rs`.
- `shutdown_signal` is defined twice under `cfg` in `crates/goose-cli/src/signal.rs`. A search for `shutdown_signal(` under `crates/` found no callers.
- `crates/goose-cli/src/bin/generate_manpages.rs` returns immediately on `target_os = "windows"`. Otherwise it walks the clap command and writes ROFF man pages.
- `crates/goose-cli/src/commands/term.rs` `Shell` is `Bash`, `Zsh`, `Fish`, `Nu`, `Powershell`. The bash and zsh templates export `AGENT_SESSION_ID`, alias `@goose` and `@g` to `term run`, and install a preexec hook that runs `term log` for other commands. Both have an optional command-not-found handler that sends the unknown command to `term run`. `TermCommand::Log` is `hide = true`.
- `ServePlatform` is `Cli` (default) or `Desktop`, mapped to `GoosePlatform::GooseCli` or `GoosePlatform::GooseDesktop` (`cli.rs`). That is the only desktop coupling read on this surface.

Inferences, separate from the measurements: this record proposes two mechanics and refuses several copies. It does not copy Rust. Kong's completion shells and Kong man-page support were not inspected and stay **[unverified]**. `docs/architecture.md` says go-selfupdate-lib gained opt-in prerelease channels in `v1.3.0`; that library's source was not opened. `docs/decisions/0003-MADR-gobble-product-identity.md` names `.agents/skills` and `~/.agents/skills`. A gobble token counter was not looked up **[unverified]**.

### Findings

F1. No subcommand is the product. `cli()` sends `None` to `handle_default_session`, which either configures (no config file) or starts an interactive session (`crates/goose-cli/src/cli.rs`). That default is a code fact. Copying it (bare goose-as-default-session) is rejected in D6. It is not proposed work.

F2. Interactive and one-shot entry points share flag groups. `Session` and `Run` both flatten `SessionOptions`, `ExtensionOptions`, and `ModelOptions` (`crates/goose-cli/src/cli.rs`). Shared flag groups are a code fact. They are not chosen work, and they are not a Phase 2 step.

F3. One-shot input is exclusive, and a file path drops `--system`. `instructions`, `--text` (`input_text`), and `--recipe` conflict on `InputOptions` (`crates/goose-cli/src/cli.rs`). `parse_run_input` in that file: a path passed to `--instructions` sets `additional_system_prompt` to `None`; `"-"` and `--text` keep `input_opts.system` by cloning it. A stdin read failure panics via `.expect("Failed to read from stdin")`. A missing instruction file calls `std::process::exit(1)`. `output_format` is `text`, `json`, or `stream-json`. `--interactive` continues into the session; `--no-session` skips persistence (`OutputOptions`, `RunBehavior`). One-shot exclusivity is a code fact. It is not chosen work and not a Phase 2 step. Do not copy the `.expect` panic. Recipes, sub-recipes, and `--render-recipe` are a goose product, not a CLI mechanic.

F4. Session bookkeeping is a real command family, and most of it is optional. `SessionCommand` covers list (text or json, working directory, limit, sort), remove, export (markdown, json, yaml, html), import of a goose export or a Claude Code, Codex, or Pi jsonl, diagnostics, and rename. `fork` and `edit` both require `resume` (`crates/goose-cli/src/cli.rs`). List, resume, and rename are code facts. They are not chosen work and not a Phase 2 step. Export, foreign-transcript import, html, yaml, diagnostics, fork, and edit-before-resume are not part of this proposal.

F5. Shell completion is a subcommand that prints a script. `CompletionShell` is bash, elvish, fish, powershell (alias `pwsh`), nu (alias `nushell`), and zsh. `CompletionShell::generate` writes the script through clap's generator, and `cli()` calls it for `Command::Completion`, writing stdout (`crates/goose-cli/src/cli.rs`). `--bin-name` defaults to `goose`. A unit test parses `completion nushell`. The proposed scope is a Kong subcommand that prints a script for bash, zsh, fish, and powershell only. Not nu or elvish until Kong is shown to emit them. Do not call clap_complete. Which of those shells Kong emits is **[unverified]**. `docs/decisions/0004-PLAN-go-module-architecture.md` Phase 2 already names a Kong `completion` subcommand; this finding does not add a second one.

F6. The interactive prompt is a line editor, not a TUI. `get_input` uses rustyline, keeps history, completes slash commands and file names (`crates/goose-cli/src/session/completion.rs`, `GooseCompleter`), clears a non-empty line on Ctrl+C, and exits on a second Ctrl+C or on EOF. Ctrl+J inserts a newline unless `GOOSE_CLI_NEWLINE_KEY` sets another character, and `m` and `c` are rejected (`crates/goose-cli/src/session/input.rs`). `/edit` opens `$GOOSE_PROMPT_EDITOR`, then `$VISUAL`, then `$EDITOR`. The line editor is a code fact. It is not chosen work and not a Phase 2 step. Do not get it by porting rustyline, and do not get it by calling go-tui-lib. TUI stays go-tui-lib.

F7. Windows paste is called out because bracketed paste is absent. `paste.rs` treats a queued key burst as a paste and does not submit each embedded newline (`crates/goose-cli/src/session/paste.rs`). Windows paste behavior is a code fact about goose. It is not chosen work and not a Phase 2 step. The chip UI is not a requirement.

F8. Slash handling is two matches, and they are not the same set. In `crates/goose-cli/src/session/input.rs`, `get_input` sends a leading `/` to `handle_slash_command`. The `_` arm returns `None`, and `get_input` turns that `None` into `InputResult::Message`, so the slash text is kept. In `crates/goose/src/agents/agent.rs`, `Agent::reply` calls `execute_command` before the model turn when `use_state_machine` is false. When `use_state_machine` is true, `reply` returns `reply_with_state_machine` first and this call does not run. That other path was not traced **[unverified]**. `execute_command` in `crates/goose/src/agents/execute_commands.rs` matches `prompts`, `prompt`, `compact`, `clear`, `skills`, `doctor`, `status`, `goal`, and `grind`, then a recipe, then a skill. `Ok(None)` covers both text that is not a slash command (`parse_slash_command` returns none) and a name that misses recipe and skill. The `Ok(None)` arm in `reply` stores the user message and continues the turn. A recipe or skill error string becomes an assistant message, not a user message. `handle_slash_command` does not match `doctor`, `goal`, or `grind`. The ACP list in `crates/goose/src/slash_commands/slash_command.rs` is neither of these matches. Proposed rule, not accepted and not a Phase 2 command: one registry outside the CLI binary, and an unknown slash is an error. Do not copy either match.

F9. `configure` is an interactive wizard with no non-interactive subcommand. It errors when stdin is not a terminal (`crates/goose-cli/src/commands/configure.rs`). `Command::Configure` takes no arguments (`crates/goose-cli/src/cli.rs`). The string `goose configure set` appears only in editor help (`input.rs`). TTY-only configure is a code fact. It is not chosen work and not a Phase 2 step. The provider-search wizard, telemetry consent, and Tetrate signup in that file are not candidates.

F10. `skills list` is a presentation over filesystem discovery. The command prints name, a 50-character description, two token counts, and a path (`crates/goose-cli/src/commands/skills.rs`). Discovery walks `SKILL.md` under project `.agents/skills`, `.goose/skills`, `.claude/skills`, then home and config dirs including `.agents/skills` and `.claude/skills` (`crates/goose/src/skills/mod.rs`). `validate_skill_name` is the agentskills-shaped constraint (lowercase, digits, hyphens, length at most 64). A gobble token counter was not looked up **[unverified]**. `docs/decisions/0003-MADR-gobble-product-identity.md` already names `.agents/skills` and `~/.agents/skills`. Adding `.goose/skills` only to match goose is not proposed. Porting goose's `skills` command is rejected in D6.

F11. `update` is a full installer, not a CLI wrapper. It selects a platform asset, downloads a GitHub release, checks Sigstore SLSA provenance, replaces the binary, and can re-exec `configure` (`crates/goose-cli/src/commands/update.rs`). The feature flag `disable-update` removes the capability (`crates/goose-cli/Cargo.toml`). Goose's Sigstore updater is rejected (D6). Self-update stays go-selfupdate-lib. `docs/architecture.md` says that library gained opt-in prerelease channels in `v1.3.0`. This pass did not open the library source. Do not add a canary flag to match goose.

F12. `term` is shell integration, not a subcommand of the REPL. Init scripts alias `@goose` and `@g`, log every other command via a hidden `term log`, and can send unknown commands to goose (`crates/goose-cli/src/commands/term.rs`). Consequence: do not adopt it for the native CLI. The strongest reason to want it, a per-terminal session without entering the REPL, does not outweigh logging every shell command.

F13. `doctor` is not a diagnostic command. It injects the string `/doctor` into an interactive session (`crates/goose-cli/src/commands/doctor.rs`). `info --check` actually calls the provider and returns an error on failure (`crates/goose-cli/src/commands/info.rs`). `info --check` is a code fact. It is not chosen work and not a Phase 2 step. Do not copy doctor-as-slash-injection.

F14. Help text is the source of man pages, and generation is skipped on Windows. `main` in `crates/goose-cli/src/bin/generate_manpages.rs` returns `Ok(())` immediately when `target_os` is `windows`. Man pages are rejected as chosen work (D6). Emitting ROFF from Kong was not checked **[unverified]**.

F15. The process enables Windows virtual-terminal processing on stdout and stderr before any UI runs (`enable_windows_vt_processing` in `crates/goose-cli/src/main.rs`). The same `main` then starts the CLI thread with an 8 MiB stack. Enabling that processing once, in Go, is proposed in D2. Do not depend on the Rust `console` crate. Do not copy the 8 MiB stack workaround.

F16. The same binary registers product and server commands that are not terminal-CLI mechanics: `acp`, `serve`, `roam`, `mcp`, `gateway`, `schedule`, `recipe`, `review`, `local-models`, and `plugin` (`crates/goose-cli/src/cli.rs`). `review` is a diff-review orchestrator. `serve` can label itself desktop via `ServePlatform`. Consequence: none of these are port candidates for native CLI mode.

F17. Operator commands are hidden from default help. `validate-extensions` and `mcp-probe` set `hide = true`, as does `term log` (`crates/goose-cli/src/cli.rs`, `crates/goose-cli/src/commands/term.rs`). Hidden commands are rejected as chosen work (D6). Do not copy `validate-extensions`, `mcp-probe`, or `term log`.

F18. An absolute `GOOSE_PATH_ROOT` replaces goose's config, data, and state roots. `Paths::path_root` reads that variable and `validated_path_root` keeps only an absolute path (`crates/goose/src/config/paths.rs`). Otherwise `get_dir` uses the `etcetera` strategy with top-level domain `Block` and app name `goose`. Gobble's directory table is already specified: config, data, state, cache, and secrets, plus `GOBBLE_HOME` and the per-role overrides (`docs/decisions/0003-MADR-gobble-product-identity.md`, Directories). `docs/decisions/0004-PLAN-go-module-architecture.md` Phase 2 step 3 is `internal/appdirs` for that table, including `config path`. Do not add a `GOOSE_PATH_ROOT`-style override. Keep that table.

F19. Goose CLI logging is a dated folder and a 14-day directory delete, with no redaction. `setup_logging` always sets `console: false` (`crates/goose-cli/src/logging.rs`). `build_logging_subscriber` calls `prepare_log_directory(component, true)`, which joins `%Y-%m-%d`, and `cleanup_old_logs` deletes subdirectories older than 14 days (`crates/goose/src/logging.rs`). A console layer is added only when `config.console` is true; the CLI call does not set that. Those two files do not redact. That `console: false` setting is not gobble's rule that `gobble acp` must not log to stderr. Phase 2 logging, already specified, is `slog`, a rolling file in the state directory, a redacting `ReplaceAttr`, and no stderr handler when the process is `gobble acp` (`docs/decisions/0004-PLAN-go-module-architecture.md` Phase 2 step 4). `docs/decisions/0005-MADR-v1-feature-scope.md` also says `slog` rolling files with redaction. Do not adopt date log folders or the 14-day cleanup. Keep that Phase 2 logging.

F20. Goose has no exit-code table. `std::process::exit(1)` is used from `parse_run_input` and elsewhere in `crates/goose-cli/src/cli.rs`, and from `crates/goose-cli/src/session/builder.rs`. `shutdown_signal` in `crates/goose-cli/src/signal.rs` has no callers in a `crates/` search. Gobble's planned codes are 0 ok, 1 runtime failure, 2 usage, 3 authentication, 4 cancelled, and 130 when SIGINT ended the process (`docs/decisions/0004-PLAN-go-module-architecture.md` Phase 2 step 6). `gobble update --check` maps update-available to 10 (`docs/decisions/0005-MADR-v1-feature-scope.md`, Self-update). `internal/cli/exit.go` is not in the tree, so the table is a document, not an implementation found in this pass. Keep those codes. Do not copy `process::exit(1)`, the stdin `.expect` panic, or `shutdown_signal`.

## Decision Drivers

- Kong is already chosen. A second command parser is out of the question.
- TUI mode already has a library (go-tui-lib). The native CLI must stay a line-oriented program.
- Self-update already has a library (go-selfupdate-lib). Goose's Sigstore updater is refused.
- Completion is already named as a Kong subcommand in `0004-PLAN` Phase 2. This record proposes the shell set and the print-a-script shape. It does not add a second command.
- Windows virtual-terminal processing is a startup fact in goose. The Go equivalent is proposed here and was not a step in the Phase 2 list that was read.
- One definition for a flag or a slash command, if those are ever accepted. Goose currently has two slash matches; that is the defect, not the model. The one-registry rule is not a Phase 2 command.
- Later mechanics stay undecided and out of Phase 2: one-shot exclusivity, shared flag groups, session list/resume/rename, the line editor, TTY-only configure, and `info --check`.
- No local copy of goose. A claim that was not re-opened in this pass is not restated as new fact.

## Considered Options

- A — Adopt the wider terminal mechanics (bare session, shared flags, one-shot, session list/resume/rename, line editor, TTY configure, skills list, `info --check`) and refuse the product command tree.
- B — Port the goose-cli command tree command for command, including `term`, `recipe`, `schedule`, `review`, and `update`.
- C — Take nothing from goose. Design the native CLI only from Kong's defaults.
- D — Ship shell integration (`term`) and a thin one-shot `run` first, and postpone the in-process session.
- E — Add clap, Cobra, or fang as a second command parser (rejected)
- F — Propose only a completion script (bash, zsh, fish, powershell) and Windows virtual-terminal processing. Leave the later mechanics undecided. Reject the path-root override, goose log layout, goose exit behavior, bare goose-as-default-session, man pages, hidden commands, term aliases, the Sigstore updater, and the product command tree.

## Decision Outcome

Option F is the proposed outcome. It is not accepted. Mac did not pick it. D1 and D2 are the proposed scope. They are not accepted work. D3 through D6 are rejections, not work to build. The undecided list is not a phase and not a PLAN.

### The decisions

D1. Proposed, not accepted. A completion subcommand prints a script, like `CompletionShell::generate` and `Command::Completion` in `crates/goose-cli/src/cli.rs`. Shells: bash, zsh, fish, and powershell only. Not nu or elvish until Kong is shown to emit them. Do not call clap_complete. Implement it with Kong. `docs/decisions/0004-PLAN-go-module-architecture.md` Phase 2 already names a Kong `completion` subcommand. This decision sets the shell set and the print-a-script shape. It does not add a second command.

D2. Proposed, not accepted. Enable Windows virtual-terminal processing on stdout and stderr once in `cmd/gobble/main.go` before any UI, for the same reason as `enable_windows_vt_processing` in `crates/goose-cli/src/main.rs`: without it, spinners print as repeated lines on Windows Console Host. Use the Go standard library or an existing sibling library. Do not depend on the Rust `console` crate. Do not copy the 8 MiB stack workaround (`stack_size(8 * 1024 * 1024)` in that `main`). This step is not in the Phase 2 list read in `0004-PLAN`. It is still only proposed.

D3. Rejected. No `GOOSE_PATH_ROOT`-style override. Keep the appdirs table and `config path` already specified in `docs/decisions/0003-MADR-gobble-product-identity.md` and in `0004-PLAN` Phase 2 step 3.

D4. Rejected. No date log folders and no 14-day cleanup. Keep Phase 2 logging: `slog`, a rolling file, redaction, and no stderr handler when the process is `gobble acp`. Goose `setup_logging` has no redaction and always sets `console: false`. That is not the acp rule.

D5. Rejected as a copy. Keep gobble exit codes 0, 1, 2, 3, 4, 130 on SIGINT, and 10 for `update --check`. Goose has no such table. Do not copy `process::exit(1)`, the stdin `.expect` panic in `parse_run_input`, or `shutdown_signal` (`crates/goose-cli/src/signal.rs` has no callers).

D6. Rejected. Do not adopt bare goose-as-default-session, man pages, hidden commands, term aliases, goose's Sigstore update, or these product commands: `acp`, `serve`, `roam`, `mcp`, `gateway`, `schedule`, `recipe`, `review`, `local-models`, `plugin`, and `skills`.

The F8 rule is proposed and not accepted: one registry outside the CLI binary, and an unknown slash is an error. That rule is not a Phase 2 command. Do not copy either goose match.

Undecided, not chosen work, and not Phase 2: one-shot exclusivity (F3), shared flag groups (F2), session list, resume, and rename (F4), the line editor (F6, F7), TTY-only configure (F9), and `info --check` (F13). Those sentences describe code that was read. They are not a commitment.

### Consequences

- Good: the proposed scope is two mechanics that were read in goose, without clap_complete, without the Rust `console` crate, and without the 8 MiB stack.
- Good: `0004-PLAN` Phase 2 already names Kong `completion`. D1 only narrows the shells.
- Bad: users coming from goose will not find `term`, recipes, a Sigstore updater, or a bare session that matches goose. That gap is intentional.
- Bad: the F8 rule, if it is later accepted, rejects sending an unknown slash line to the model. It is not part of D1 or D2, and it is not a Phase 2 command.
- Risk: D2 is easy to "solve" by depending on a wrapper shaped like the Rust `console` crate, or by copying the 8 MiB stack. Both are refused.
- The refusal most likely to be broken under time pressure is D6's updater line, by reimplementing goose's download inside gobble. That would also violate the existing self-update decision.

### Confirmation

These commands were not run. They are checks for D1 and D2 only, and only after this proposal is accepted. Kong completion coverage beyond bash, zsh, fish, and powershell stays **[unverified]**. The undecided mechanics have no check here.

```text
gobble completion bash
gobble completion powershell
  expected: a script on stdout, exit 0; nu and elvish are not required

Windows Console Host, before any UI
  expected: virtual-terminal processing is enabled on stdout and stderr; the module graph does not depend on the Rust console crate; the 8 MiB stack workaround is absent
```

## Pros and Cons of the Options

### F — Propose completion and Windows VT only (chosen, proposed, not accepted)

- Plus: the strongest argument is scope. Two mechanics were read end to end, and neither requires copying clap, the Rust `console` crate, or goose's product tree.
- Plus: D1 fits the Kong `completion` subcommand `0004-PLAN` Phase 2 already names, and only pins the shells.
- Minus: session shape, one-shot input, the line editor, configure, and `info --check` stay undecided. A later record has to pick them, or not.
- Minus: Mac has not accepted F. Status stays `proposed`.

### A — Adopt the wider terminal mechanics and refuse the product surface

- Plus: it would take the parts of goose that are about being a terminal program and still leave Kong, go-tui-lib, and go-selfupdate-lib in place.
- Plus: each of those mechanics maps to a file that was opened.
- Minus: it is not the proposed outcome. It would commit one-shot exclusivity, shared flags, session list/resume/rename, the line editor, TTY-only configure, and `info --check`, which this record leaves undecided and out of Phase 2.
- Minus: it would also copy bare goose-as-default-session, which D6 rejects.

### B — Port the goose-cli command tree command for command

- Plus: the strongest argument is fidelity. A user who knows `goose session --resume --fork`, `goose term init zsh`, and `goose update --canary` would find the same words, and the clap tree is a finished design rather than a guess.
- Plus: shared option groups, hidden commands, and man pages would arrive together because they fall out of one parser definition.
- Minus: it throws away Kong, reimplements self-update, and pulls servers, schedulers, recipes, and a diff-review orchestrator into the default CLI. That is a different product.
- Minus: goose's slash story is split across `input.rs` and `execute_command`. Copying the tree would copy that split.

### C — Take no goose CLI ideas

- Plus: the strongest argument is focus. Gobble's CLI constraints are already decided, and a blank Kong app cannot accidentally inherit goose's recipes, telemetry dialog, or shell hooks.
- Plus: it avoids a proposed-decision list that the owner has not accepted.
- Minus: it discards measured answers this record does use, including how completion is printed and why Windows Console Host needs virtual-terminal processing before any UI.
- Minus: D1 and D2 would have to be re-investigated later against the same tree.

### D — Shell integration first, in-process session later

- Plus: the strongest argument is that a terminal-native tool should live in the shell the user already has. `term init` plus `term run` gives a persistent session, aliases, and a prompt badge without building a line editor.
- Plus: it dodges the rustyline and Windows-paste problem entirely.
- Minus: the hook logs every command the user types (`term.rs`). That is a privacy and safety default, not an optional extra.
- Minus: it postpones the actual default mode. Gobble's native CLI is the program the user runs, not a set of shell functions. D6 rejects term aliases.

### E — Add clap, Cobra, or fang as a second command parser (rejected)

- Plus: none that survives contact with the existing Kong decision. A second parser would make D1 call clap_complete, which this record forbids.
- Minus: Kong is already the command parser. Completion and help stay on that definition.

## More Information

### Evidence index

| Claim | Source |
| --- | --- |
| Assessed tree is `591edd47cf2cfea4957d720c607cf2a4def8673d` on `main`, origin `https://github.com/aaif-goose/goose.git`, clean relative to `origin/main` | `git rev-parse HEAD`, `git rev-parse --abbrev-ref HEAD`, `git remote get-url origin`, `git status -sb` in `gitrepos/goose`, 2026-10-04 |
| Commit subject and time `2026-10-02 15:03:24 +0000` (10:03 CT) | `git log -1` |
| Bin `goose` depends on clap, clap_complete, rustyline, cliclack; features `update` and empty `disable-update` | `crates/goose-cli/Cargo.toml` |
| No subcommand starts configure or an interactive session | `crates/goose-cli/src/cli.rs` `handle_default_session`, `cli` match arm `None` |
| `Session` and `Run` share flattened option groups; run input sources conflict; output formats are `text`, `json`, `stream-json` | `crates/goose-cli/src/cli.rs` `SessionOptions`, `ExtensionOptions`, `ModelOptions`, `InputOptions`, `OutputOptions`, `RunBehavior` |
| A path to `--instructions` drops `--system`; `"-"` and `--text` keep it; stdin failure panics with `.expect` | `crates/goose-cli/src/cli.rs` `parse_run_input` |
| Session subcommands and resume, fork, edit, history flags | `crates/goose-cli/src/cli.rs` `SessionCommand`, `Command::Session` |
| Completion shells and stdout generation | `crates/goose-cli/src/cli.rs` `CompletionShell::generate`, `cli` match arm `Command::Completion` |
| CLI slash match returns `None` on `_`, and `get_input` turns that into `InputResult::Message` | `crates/goose-cli/src/session/input.rs` `get_input`, `handle_slash_command` |
| `Agent::reply` calls `execute_command` only when `use_state_machine` is false; the match names prompts, prompt, compact, clear, skills, doctor, status, goal, grind, then recipe, then skill; `Ok(None)` continues the turn | `crates/goose/src/agents/agent.rs` `reply`; `crates/goose/src/agents/execute_commands.rs` `execute_command` |
| State-machine slash path | **[unverified]** — `reply_with_state_machine` was not traced |
| In-session completion of prompts, modes, skills, models | `crates/goose-cli/src/session/completion.rs` `GooseCompleter` |
| Windows paste is a key-burst collapse, not bracketed paste | `crates/goose-cli/src/session/paste.rs` module comment |
| Library slash registry precedence and builtin names | `crates/goose/src/slash_commands/slash_command.rs` `merge_command_sources` and `lists_acp_safe_builtin_commands` |
| `configure` requires a TTY and has no subcommand; `configure set` is only a help string | `crates/goose-cli/src/commands/configure.rs` `handle_configure`; `crates/goose-cli/src/cli.rs` `Command::Configure`; `crates/goose-cli/src/session/input.rs` `print_editor_help` |
| Skills table columns | `crates/goose-cli/src/commands/skills.rs` `handle_skills_list` |
| Skill directory order and `SKILL.md`; name rules | `crates/goose/src/skills/mod.rs` `all_skill_dirs_with_config`, `scan_skills_from_dir`, `validate_skill_name`, `list_installed_skills` |
| Updater downloads `stable` or `canary`, verifies Sigstore SLSA, replaces the binary, optional reconfigure | `crates/goose-cli/src/commands/update.rs` `update` |
| `doctor` sends the string `/doctor` into a session | `crates/goose-cli/src/commands/doctor.rs` `handle_doctor` |
| `info --check` calls the provider and fails the process on error | `crates/goose-cli/src/commands/info.rs` `handle_info` |
| Windows VT processing on stdout and stderr; 8 MiB stack | `crates/goose-cli/src/main.rs` `enable_windows_vt_processing`, `stack_size(8 * 1024 * 1024)` |
| Man pages generated from the CLI definition, return immediately on Windows | `crates/goose-cli/src/bin/generate_manpages.rs` |
| `term` aliases, preexec logging, command-not-found handler | `crates/goose-cli/src/commands/term.rs` `BASH_CONFIG`, `ZSH_CONFIG`, `TermCommand` |
| Product commands registered beside the REPL, including hidden ones | `crates/goose-cli/src/cli.rs` `Command` |
| `GOOSE_PATH_ROOT` must be absolute | `crates/goose/src/config/paths.rs` `path_root`, `validated_path_root` |
| Dated log directory, 14-day cleanup, CLI `console: false`, no redaction in these files | `crates/goose-cli/src/logging.rs` `setup_logging`; `crates/goose/src/logging.rs` `prepare_log_directory`, `cleanup_old_logs` |
| `shutdown_signal` has no callers; `process::exit(1)` exists on the run path | `crates/goose-cli/src/signal.rs`; search under `crates/`; `crates/goose-cli/src/cli.rs` `parse_run_input`; `crates/goose-cli/src/session/builder.rs` |
| Gobble directory table and `GOBBLE_HOME` | `docs/decisions/0003-MADR-gobble-product-identity.md`, Directories |
| Phase 2 appdirs, logging, Kong `completion`, exit codes 0, 1, 2, 3, 4, 130 | `docs/decisions/0004-PLAN-go-module-architecture.md` Phase 2 |
| `update --check` exit 10 | `docs/decisions/0005-MADR-v1-feature-scope.md`, Self-update |
| `internal/cli/exit.go` not in the tree | directory listing, 2026-10-04 |
| Kong completion shells, Kong man pages, gobble token counter, `reply_with_state_machine` slash handling, go-selfupdate-lib source | **[unverified]** — not read in this pass. `docs/architecture.md` states prerelease channels in `v1.3.0` without this pass opening the library |

### Related records

No PLAN is part of this proposal. Do not create a `0006-PLAN`. This file is `docs/decisions/0006-MADR-goose-cli-port-candidates.md` in the gobble-cli repository. Status is proposed. It is not accepted.

Already decided, and not re-opened by this record: Kong is the CLI library; the default mode is the native terminal CLI and core TUI is go-tui-lib; self-update is go-selfupdate-lib; Go 1.27.1; sibling libraries and the standard library rather than local copies.

### Open questions for the plan

- Which sibling Go library, if any, provides line editing with history, completion, and Windows multi-line paste (F6, F7). Undecided. Not a Phase 2 step. If none does, a later record stops at that gap instead of porting `paste.rs`.
- Which completion shells Kong emits, in particular nushell and elvish (F5, D1). **[unverified]**. D1 does not add them until that is shown.
- Whether Kong or a sibling emits man pages (F14). **[unverified]**. D6 rejects man pages as chosen work.
- go-selfupdate-lib channels. `docs/architecture.md` says opt-in prerelease channels arrived in `v1.3.0`. This pass did not open that source. Do not invent a `--canary` flag to match goose.
- Whether a token counter already exists (F10). **[unverified]**. `.agents/skills` is already named in `0003-MADR`. Porting goose `skills` is rejected (D6).
- How `reply_with_state_machine` treats a slash line (F8). **[unverified]**.
- Whether one-shot `json` is one document or a stream. Goose has both `json` and `stream-json` (F3). Undecided. Not a Phase 2 step.
