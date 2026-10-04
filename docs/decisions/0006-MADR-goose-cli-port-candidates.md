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
- `main.rs` on Windows calls `console::Term` color detection before the Tokio runtime, with a comment that this sets `ENABLE_VIRTUAL_TERMINAL_PROCESSING` so spinners do not print as repeated lines. The CLI thread is started with an 8 MiB stack.
- `crates/goose-cli/src/bin/generate_manpages.rs` returns immediately on `target_os = "windows"`. Otherwise it walks the clap command and writes ROFF man pages.
- `crates/goose-cli/src/commands/term.rs` `Shell` is `Bash`, `Zsh`, `Fish`, `Nu`, `Powershell`. The bash and zsh templates export `AGENT_SESSION_ID`, alias `@goose` and `@g` to `term run`, and install a preexec hook that runs `term log` for other commands. Both have an optional command-not-found handler that sends the unknown command to `term run`. `TermCommand::Log` is `hide = true`.
- `ServePlatform` is `Cli` (default) or `Desktop`, mapped to `GoosePlatform::GooseCli` or `GoosePlatform::GooseDesktop` (`cli.rs`). That is the only desktop coupling read on this surface.

Inferences, separate from the measurements: gobble should copy the shape of a few of these commands and none of the Rust. Kong's completion shells, Kong man-page support, the go-selfupdate-lib channel API, and gobble's existing skill roots were not inspected. Those are marked **[unverified]** where a decision would depend on them.

### Findings

F1. No subcommand is the product. `cli()` sends `None` to `handle_default_session`, which either configures (no config file) or starts an interactive session (`crates/goose-cli/src/cli.rs`). Consequence: gobble's native CLI should do the same, and a missing config must not fall through into a session.

F2. Interactive and one-shot entry points share flag groups. `Session` and `Run` both flatten `SessionOptions`, `ExtensionOptions`, and `ModelOptions` (`crates/goose-cli/src/cli.rs`). Consequence: gobble should declare model, provider, and limit flags once and attach that set to both commands. Two hand-copied Kong flag lists will drift.

F3. One-shot input is exclusive and the output is machine-readable. `instructions` (help text: `-` means stdin), `--text`, and `--recipe` conflict. `output_format` is `text`, `json`, or `stream-json`. `--interactive` continues into the session; `--no-session` skips persistence (`crates/goose-cli/src/cli.rs`, `InputOptions`, `OutputOptions`, `RunBehavior`). Consequence: a one-shot command with text, file, and stdin, plus `text` and `json`, is worth having. Recipes, sub-recipes, and `--render-recipe` are a goose product, not a CLI mechanic.

F4. Session bookkeeping is a real command family, and most of it is optional. `SessionCommand` covers list (text or json, working directory, limit, sort), remove, export (markdown, json, yaml, html), import of a goose export or a Claude Code, Codex, or Pi jsonl, diagnostics, and rename. `fork` and `edit` both require `resume` (`crates/goose-cli/src/cli.rs`). Consequence: list, resume, rename, and export to markdown and json are the part worth leveraging. Foreign-transcript import, html, yaml, diagnostics, fork, and edit-before-resume are not part of this proposal.

F5. Shell completion is a subcommand that prints a script. Shells are bash, zsh, fish, powershell (`pwsh`), nushell (`nu`), and elvish, with `--bin-name` defaulting to `goose` (`crates/goose-cli/src/cli.rs`, `CompletionShell`). Consequence: gobble should grow `completion` on Kong, not by embedding clap. Which of those shells Kong can emit is **[unverified]**.

F6. The interactive prompt is a line editor, not a TUI. `get_input` uses rustyline, keeps history, completes slash commands and file names (`crates/goose-cli/src/session/completion.rs`, `GooseCompleter`), clears a non-empty line on Ctrl+C, and exits on a second Ctrl+C or on EOF. Ctrl+J inserts a newline unless `GOOSE_CLI_NEWLINE_KEY` sets another character, and `m` and `c` are rejected (`crates/goose-cli/src/session/input.rs`). `/edit` opens `$GOOSE_PROMPT_EDITOR`, then `$VISUAL`, then `$EDITOR`. Consequence: native CLI mode needs those behaviors. It must not get them by porting rustyline, and it must not get them by calling go-tui-lib.

F7. Windows paste is called out because bracketed paste is absent. `paste.rs` treats a queued key burst as a paste and does not submit each embedded newline (`crates/goose-cli/src/session/paste.rs`). Consequence: the line reader chosen for gobble has to make the same guarantee on Windows. The chip UI is not the requirement.

F8. Slash commands are two systems. The terminal parser is a match in `input.rs` and turns an unknown slash line into a user message. The library registry in `crates/goose/src/slash_commands/slash_command.rs` merges builtins, recipes, and skills with builtin, then recipe, then skill, and is what ACP lists. The names are not the same set (the library test includes `doctor`, `goal`, and `grind`; the CLI match does not). Consequence: gobble should have one registry owned outside the CLI binary. Goose's "unknown slash becomes a chat message" behavior is not worth copying.

F9. `configure` is an interactive wizard with no non-interactive subcommand. It errors when stdin is not a terminal (`crates/goose-cli/src/commands/configure.rs`). `Command::Configure` takes no arguments (`crates/goose-cli/src/cli.rs`). The string `goose configure set` appears only in editor help (`input.rs`). Consequence: gobble's `configure` should refuse a non-TTY with a non-zero exit. Flags and files are how non-interactive config is set. The provider-search wizard, telemetry consent, and Tetrate signup in that file are not candidates.

F10. `skills list` is a presentation over filesystem discovery. The command prints name, a 50-character description, two token counts, and a path (`crates/goose-cli/src/commands/skills.rs`). Discovery walks `SKILL.md` under project `.agents/skills`, `.goose/skills`, `.claude/skills`, then home and config dirs including `.agents/skills` and `.claude/skills` (`crates/goose/src/skills/mod.rs`). `validate_skill_name` is the agentskills-shaped constraint (lowercase, digits, hyphens, length at most 64). Consequence: a `skills list` of name, description, and path is worth having. Token columns need a counter gobble already has; that counter was not looked up **[unverified]**. Adding `.goose/skills` only to match goose is not.

F11. `update` is a full installer, not a CLI wrapper. It selects a platform asset, downloads a GitHub release, checks Sigstore SLSA provenance, replaces the binary, and can re-exec `configure` (`crates/goose-cli/src/commands/update.rs`). The feature flag `disable-update` removes the capability (`crates/goose-cli/Cargo.toml`). Consequence: do not port any of that. The only idea to keep is a single `update` subcommand that a build can omit, implemented by go-selfupdate-lib. Whether that library has a canary or channel switch is **[unverified]**.

F12. `term` is shell integration, not a subcommand of the REPL. Init scripts alias `@goose` and `@g`, log every other command via a hidden `term log`, and can send unknown commands to goose (`crates/goose-cli/src/commands/term.rs`). Consequence: do not adopt it for the native CLI. The strongest reason to want it, a per-terminal session without entering the REPL, does not outweigh logging every shell command.

F13. `doctor` is not a diagnostic command. It injects the string `/doctor` into an interactive session (`crates/goose-cli/src/commands/doctor.rs`). `info --check` actually calls the provider and returns an error on failure (`crates/goose-cli/src/commands/info.rs`). Consequence: copy the `info --check` shape. Do not copy doctor-as-slash-injection.

F14. Help text is the source of man pages, and generation is skipped on Windows (`crates/goose-cli/src/bin/generate_manpages.rs`). Consequence: gobble's command help should stay on the Kong definitions. Emitting ROFF from Kong was not checked **[unverified]** and is not a commitment.

F15. The process enables Windows virtual-terminal processing before any UI runs (`crates/goose-cli/src/main.rs`). Consequence: the native CLI should do the equivalent once at startup, using the standard library or an existing sibling, so ANSI status updates in place on Windows Console Host.

F16. The same binary registers product and server commands that are not terminal-CLI mechanics: `acp`, `serve`, `roam`, `mcp`, `gateway`, `schedule`, `recipe`, `review`, `local-models`, and `plugin` (`crates/goose-cli/src/cli.rs`). `review` is a diff-review orchestrator. `serve` can label itself desktop via `ServePlatform`. Consequence: none of these are port candidates for native CLI mode.

F17. Operator commands are hidden from default help. `validate-extensions` and `mcp-probe` set `hide = true`, as does `term log` (`crates/goose-cli/src/cli.rs`, `crates/goose-cli/src/commands/term.rs`). Consequence: the pattern is available if gobble later needs a debug command. It is not required to copy these three.

## Decision Drivers

- Native CLI mode is the default. Bare invocation has to do something a terminal user expects, which in goose is a session, not a help page.
- Kong is already chosen. A second command parser is out of the question.
- TUI mode already has a library. The native CLI must stay a line-oriented program.
- Self-update already has a library. Goose's updater is the thing most likely to be copied under time pressure, because it looks finished, and it is the thing this record refuses.
- One definition for a flag or a slash command. Goose currently has two slash lists; that is the defect, not the model.
- Machine-readable output for the one-shot path, because scripts cannot scrape a REPL.
- Windows is a first-class terminal (VT processing and paste), not a port to do later.
- No local copy of goose, and no behavior whose only source is an unread gobble file. This pass did not open `gobble-cli`.

## Considered Options

- A — Adopt the terminal mechanics in D1–D16 and refuse the product command tree.
- B — Port the goose-cli command tree command for command, including `term`, `recipe`, `schedule`, `review`, and `update`.
- C — Take nothing from goose. Design the native CLI only from Kong's defaults.
- D — Ship shell integration (`term`) and a thin one-shot `run` first, and postpone the in-process session.
- E — Add clap, Cobra, or fang as a second command parser (rejected)

## Decision Outcome

Option A is the proposed outcome. D1–D16 are proposed commitments. They are not accepted. Mac has not approved them. A later PLAN implements only the ones that are accepted, by these numbers.

### The decisions

D1. Keep Kong as the only command parser for the native CLI. Do not add a second command parser, and do not translate goose's derive macros into Go.

D2. With no subcommand, start an interactive session when configuration exists. When it does not, run configure and do not start a session.

D3. Define model, provider, and turn or tool limits once, and attach that flag set to both the interactive session command and the one-shot command.

D4. The one-shot command accepts a prompt as text, a file, or stdin (`-`), and those three sources are mutually exclusive. It prints `text` or `json`. It may take a flag to continue in the interactive session after the first turn. It does not grow a recipe runner.

D5. Session commands in this proposal are list, resume, rename, and export to markdown and json. Do not add foreign-transcript import, html or yaml export, diagnostics, fork, or edit-before-resume under this record.

D6. Add `completion <shell>` that writes a script to stdout, implemented with Kong, for bash, zsh, fish, and powershell. Add nushell or elvish only after Kong is shown to emit them.

D7. Interactive input uses a Go line-editing library, not a port of rustyline and not go-tui-lib. It must keep persistent history, complete slash commands, clear a non-empty line on Ctrl+C, offer a configurable insert-newline chord, and on Windows must not submit a multi-line paste once per embedded newline.

D8. Slash commands live in one registry outside the CLI binary. The native CLI consumes that registry. An unknown slash command is an error, not a chat message.

D9. `configure` requires a terminal. On a non-TTY it exits non-zero and says why. Non-interactive configuration is flags or a file, not a `configure` subcommand copied from goose. Do not port goose's provider wizard.

D10. `skills list` prints name, description, and path for discovered `SKILL.md` skills. Do not add a `.goose/skills` root for goose compatibility. Do not add token columns unless a token counter already exists in a sibling library.

D11. The `update` subcommand calls go-selfupdate-lib and nothing else. Do not port goose's asset table, Sigstore verification, archive extraction, or re-exec of configure. A build may omit the command. Do not add a canary flag until that library is shown to have a channel.

D12. Do not implement `term`, shell init scripts, command-not-found hooks, or per-terminal session aliases.

D13. Add a non-interactive `info` command with a check flag that talks to the configured provider and exits non-zero on failure. Do not implement `doctor` by injecting a slash string into a session.

D14. Enable Windows virtual-terminal processing once at process start in the native CLI.

D15. Do not port `acp`, `serve`, `roam`, `mcp`, `gateway`, `schedule`, `recipe`, `review`, `local-models`, or `plugin` as native CLI commands.

D16. Command help stays on the Kong definitions. Do not commit to man-page generation until a sibling or Kong itself is shown to emit them.

### Consequences

- Good: the native CLI gets a session, a one-shot, completion, a single slash list, a TTY-safe configure, a skills table, and a health check, without a second TUI or a second updater.
- Bad: users coming from goose will not find `term`, recipes, scheduled jobs, review, or `goose update`'s canary channel. That gap is intentional.
- Bad: D8 rejects goose's habit of sending unknown slash text to the model. People who liked that will call it a regression. It is a different product choice, made so a typo cannot spend a model call.
- Risk: D7 depends on a line-editing library this pass did not select. If none of the sibling libraries can do Windows paste and history, the plan has to say so and stop, not copy `paste.rs`.
- The commitment most likely to be broken under time pressure is D11. Goose's updater is self-contained and verifies provenance. Reimplementing "just the download" inside gobble would violate both this record and the existing self-update decision. The next most likely is D8, by hardcoding a second slash match in the CLI the way `input.rs` does.

### Confirmation

These commands were not run. They are the checks a later accepted plan would run against gobble. Kong completion coverage beyond bash, zsh, fish, and powershell stays **[unverified]** until the first command below is actually executed.

```text
gobble --help
  expected: interactive session is the default; help is available via a help flag or help command, not by refusing to start

gobble
  expected: non-zero and a TTY error when stdin is not a terminal and no config exists; otherwise an interactive prompt

gobble run --text "ping" --output-format json
gobble run - --output-format json < prompt.txt
  expected: one JSON document on stdout; combining --text and a file fails at parse time

gobble completion bash
gobble completion powershell
  expected: a script on stdout, exit 0

gobble skills list
  expected: columns name, description, path; no .goose/skills root created

gobble info --check
  expected: non-zero when the provider is missing or rejects the check

gobble update
  expected: delegates to go-selfupdate-lib; the module graph does not contain a Sigstore or goose release downloader
```

## Pros and Cons of the Options

### A — Adopt the terminal mechanics and refuse the product surface (chosen)

- Plus: matches the decisions already made (Kong, go-tui-lib, go-selfupdate-lib) and still takes the parts of goose that are actually about being a terminal program.
- Plus: each proposed commitment maps to a file that was opened, so a later plan can close findings by number.
- Minus: the result will not feel like goose. `term`, recipes, and the configure wizard are the parts a goose user notices first, and A leaves them behind.
- Minus: D6 and D16 stop at the point where Kong's capabilities were not measured.

### B — Port the goose-cli command tree command for command

- Plus: the strongest argument is fidelity. A user who knows `goose session --resume --fork`, `goose term init zsh`, and `goose update --canary` would find the same words, and the clap tree is a finished design rather than a guess.
- Plus: shared option groups, hidden commands, and man pages would arrive together because they fall out of one parser definition.
- Minus: it throws away Kong, reimplements self-update, and pulls servers, schedulers, recipes, and a diff-review orchestrator into the default CLI. That is a different product.
- Minus: goose's own slash story is split across `input.rs` and `slash_commands`. Copying the tree would copy that split.

### C — Take no goose CLI ideas

- Plus: the strongest argument is focus. Gobble's CLI constraints are already decided, and a blank Kong app cannot accidentally inherit goose's recipes, telemetry dialog, or shell hooks.
- Plus: it avoids a long proposed-decision list that the owner has not accepted.
- Minus: it discards measured answers to questions the native CLI still has, including what bare invocation does, how one-shot output is shaped, and how Windows paste and VT processing behave.
- Minus: those questions would be re-investigated later, against the same goose tree or against nothing.

### D — Shell integration first, in-process session later

- Plus: the strongest argument is that a terminal-native tool should live in the shell the user already has. `term init` plus `term run` gives a persistent session, aliases, and a prompt badge without building a line editor.
- Plus: it dodges the rustyline and Windows-paste problem entirely.
- Minus: the hook logs every command the user types (`term.rs`). That is a privacy and safety default, not an optional extra.
- Minus: it postpones the actual default mode. Gobble's native CLI is the program the user runs, not a set of shell functions.

## More Information

### Evidence index

| Claim | Source |
| --- | --- |
| Assessed tree is `591edd47cf2cfea4957d720c607cf2a4def8673d` on `main`, origin `https://github.com/aaif-goose/goose.git`, clean relative to `origin/main` | `git rev-parse HEAD`, `git rev-parse --abbrev-ref HEAD`, `git remote get-url origin`, `git status -sb` on MAC420, 2026-10-04 |
| Commit subject and time `2026-10-02 15:03:24 +0000` (10:03 CT) | `git log -1` |
| Bin `goose` depends on clap, clap_complete, rustyline, cliclack; features `update` and empty `disable-update` | `crates/goose-cli/Cargo.toml` |
| No subcommand starts configure or an interactive session | `crates/goose-cli/src/cli.rs` `handle_default_session`, `cli` match arm `None` |
| `Session` and `Run` share flattened option groups; run input sources conflict; output formats are `text`, `json`, `stream-json` | `crates/goose-cli/src/cli.rs` `SessionOptions`, `ExtensionOptions`, `ModelOptions`, `InputOptions`, `OutputOptions`, `RunBehavior` |
| Session subcommands and resume, fork, edit, history flags | `crates/goose-cli/src/cli.rs` `SessionCommand`, `Command::Session` |
| Completion shells and stdout generation | `crates/goose-cli/src/cli.rs` `CompletionShell`, `cli` match arm `Command::Completion` |
| Slash match list; unknown slash becomes a message; Ctrl+C and newline key | `crates/goose-cli/src/session/input.rs` `get_input`, `handle_slash_command` |
| In-session completion of prompts, modes, skills, models | `crates/goose-cli/src/session/completion.rs` `GooseCompleter` |
| Windows paste is a key-burst collapse, not bracketed paste | `crates/goose-cli/src/session/paste.rs` module comment |
| Library slash registry precedence and builtin names | `crates/goose/src/slash_commands/slash_command.rs` `merge_command_sources` and `lists_acp_safe_builtin_commands` |
| `configure` requires a TTY and has no subcommand; `configure set` is only a help string | `crates/goose-cli/src/commands/configure.rs` `handle_configure`; `crates/goose-cli/src/cli.rs` `Command::Configure`; `crates/goose-cli/src/session/input.rs` `print_editor_help` |
| Skills table columns | `crates/goose-cli/src/commands/skills.rs` `handle_skills_list` |
| Skill directory order and `SKILL.md`; name rules | `crates/goose/src/skills/mod.rs` `all_skill_dirs_with_config`, `scan_skills_from_dir`, `validate_skill_name`, `list_installed_skills` |
| Updater downloads `stable` or `canary`, verifies Sigstore SLSA, replaces the binary, optional reconfigure | `crates/goose-cli/src/commands/update.rs` `update` |
| `doctor` sends the string `/doctor` into a session | `crates/goose-cli/src/commands/doctor.rs` `handle_doctor` |
| `info --check` calls the provider and fails the process on error | `crates/goose-cli/src/commands/info.rs` `handle_info` |
| Windows VT processing at startup | `crates/goose-cli/src/main.rs` `enable_windows_vt_processing` |
| Man pages generated from the CLI definition, skipped on Windows | `crates/goose-cli/src/bin/generate_manpages.rs` |
| `term` aliases, preexec logging, command-not-found handler | `crates/goose-cli/src/commands/term.rs` `BASH_CONFIG`, `ZSH_CONFIG`, `TermCommand` |
| Product commands registered beside the REPL, including hidden ones | `crates/goose-cli/src/cli.rs` `Command` |
| Kong completion shells, Kong man pages, go-selfupdate-lib channels, gobble skill roots, gobble token counter | **[unverified]** — not read in this pass |

### Related records

No PLAN is part of this proposal. The house location for an accepted copy would be `docs/decisions/0006-MADR-goose-cli-port-candidates.md` inside gobble-cli. This draft is not that file. It was written only at `/workspace/0006-MADR-goose-cli-port-candidates.md` because another pass is rewriting the repository. No gobble-cli record was amended, and no relative link into that repository is used here.

Already decided, and not re-opened by this record: Kong is the CLI library; the default mode is the native terminal CLI and core TUI is go-tui-lib; self-update is go-selfupdate-lib; Go 1.27.1; sibling libraries and the standard library rather than local copies.

### Open questions for the plan

- Which sibling Go library, if any, provides line editing with history, completion, and Windows multi-line paste (F6, F7, D7). Not selected here. If none does, the plan stops at that gap instead of porting `paste.rs`.
- Which completion shells Kong emits, in particular nushell and elvish (F5, D6). **[unverified]**.
- Whether Kong or a sibling emits man pages (F14, D16). **[unverified]**.
- Whether go-selfupdate-lib has a channel distinct from "latest" (F11, D11). **[unverified]**. Do not invent a `--canary` flag to match goose.
- Whether gobble already discovers `.agents/skills` and `.claude/skills`, and whether a token counter already exists (F10, D10). **[unverified]** because `gobble-cli` was not opened.
- Whether one-shot `json` is one document or a stream. Goose has both `json` and `stream-json` (F3). This proposal commits only to `text` and `json`. Streamed events wait for a later record.
