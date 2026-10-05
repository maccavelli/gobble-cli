---
status: accepted
date: 2026-10-04
decision-makers: repository owner
consulted: goose-cli source (aaif-goose/goose 591edd4); Pi coding-agent source (earendil-works/pi f5d2004 and the fork at 312184e); go-tui-lib, go-selfupdate-lib, go-llmprovider-sdk, magic-cli-remote, mcp-server-magictools and prepare-commit-msg sources; Kong v1.16.1; golang.org/x/term v0.46.0
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# The native CLI mode: a lightweight line client on Kong, x/term and the standard library, built from goose's terminal behaviours and Pi's machine contracts, and natively integrated with magic-cli-remote

## Context and Problem Statement

gobble has two terminal modes. The native terminal CLI mode is the default. The enhanced TUI mode is separate and is built on go-tui-lib ([0002-MADR](0002-MADR-cli-acp-headless-mcp-v1.md), "CLI as ACP client"; [0005-MADR](0005-MADR-v1-feature-scope.md), "Clients and modes"). The next build step is [0004-PLAN](0004-PLAN-go-module-architecture.md) Phase 2: the process edge, identity, and the Kong surface accepted in [0006-MADR](0006-MADR-goose-cli-port-candidates.md) D7–D14.

The owner asked on 2026-10-04 for this record. The request was to research the goose CLI and Pi CLI sources for reusable components and features that port over, ground the record in facts, and make the CLI mode "as clean, lightweight, yet feature-rich as possible".

**Second pass, 2026-10-04.** The owner asked, the same day, to:
- expand shell completion into gobble's own native, idiomatic Go completion, because Kong has none;
- make the code optimised, idiomatic Go 1.27.1, with `slog` and modern features enforced;
- reassess magic-cli-remote's ACP provider surfaces and code to "fully support, optimize for, and adhere to" them, because gobble "is going to be the native agent harness for magic-cli-remote going forward", with "native integration across all levels";
- test the Windows Console Host behaviour on this Windows host.

Changes to magic-cli-remote stay out of scope for now. Measurements 10–14, findings F19–F30 and decisions D17–D20 come from this pass. Decisions it changes keep their first text, marked superseded, with the new text inline.

Three things make this a decision rather than a list:

- **The earlier records do not agree on the CLI's shape.** They disagree on bare `gobble`, on the one-shot command's name, on the output formats, and on the exit codes (F1–F4).
- **The import rules decide what "lightweight" can mean.** Rule 5 keeps Charm out of `internal/cli` (F5), so the line client's toolkit is fixed: Kong, `golang.org/x/term`, and the standard library.
- **Two accepted commitments cannot be met as written.**
  - 0006 D7 says "Kong completion" for four shells, and Kong has no completion generator (F6).
  - 0006 D12 needs a line editor, and none of the siblings has one (F7).

### What was measured, not assumed

All reading was read-only. Nothing was executed except the commands named here, each on a scratch copy or in the module cache, and none in any repository's tree.

1. **goose.** `gitrepos/goose` was read at `591edd47cf2cfea4957d720c607cf2a4def8673d` (2026-10-02), with `git status -sb` clean against `origin/main`. Licence: Apache-2.0 (`LICENSE`; `Cargo.toml:14`). The CLI crate is `crates/goose-cli`. Line references below are relative to `crates/goose-cli/src/` unless a path says otherwise.
2. **Pi.** Shallow clones of upstream `earendil-works/pi` at `f5d2004` (2026-10-04, `@earendil-works/pi-coding-agent` 1.0.2) and of the fork at `312184e` (2026-09-29, 0.99.0, the commit [0001-REPORT](../reports/0001-REPORT-go-port-feasibility.md) examined) were made into the session scratchpad and read.
   - Licence: MIT, `LICENSE`, byte-identical in both.
   - The two trees differ in 330 files. Every CLI contract file is byte-identical between them: `src/modes/print-mode.ts`, `src/modes/json-event.ts`, `src/modes/rpc/*`, `src/core/output-guard.ts`, `src/cli/initial-message.ts`, `src/cli/file-processor.ts`, `docs/json.md` and `docs/cli.md`'s mode section.
   - Paths below are relative to `packages/coding-agent/` at `f5d2004`.
3. **Kong.** `go mod download github.com/alecthomas/kong@v1.16.1` was run into the module cache, and gobble's `go.mod` was not touched. Its `go.mod` has no runtime requirement; it requires only `alecthomas/assert/v2` and `alecthomas/repr`, which are test libraries. A case-insensitive search for `complet` across its `*.go` files matched one line, an English comment in `options.go:121`. The same search for `powershell`, `zsh` and `bash` matched nothing. The newest tag is `v1.16.1` (`0678fd30af8b`).
4. **Kong completion add-ons**, from the GitHub API:
   - `jotaen/kong-completion` `v0.0.14` (2026-04-30, MIT) lists Bash, Zsh and Fish in its README, and not PowerShell.
   - `willabides/kongplete` `v0.2.0` (2021-01-13, MIT) is built on `posener/complete`.
5. **`golang.org/x/term` v0.46.0** (module cache), `terminal.go`:
   - `Terminal` has a pluggable `History` interface (`:41`), an `AutoCompleteCallback` (`:70`) and `SetBracketedPasteMode` (`:993`) with `ErrPasteIndicator` (`:986`). The line limit is `maxLineLength = 4096` (`:366`).
   - Outside a paste, Ctrl+C returns `"", io.EOF` whatever the line holds (`:832-834`). Ctrl+D on an empty line also returns `io.EOF` (`:827-830`).
   - Inside a bracketed paste, each Enter still ends a `ReadLine`, and the line is flagged `ErrPasteIndicator` (`:836-870`).
   - On Windows, `makeRaw` sets `ENABLE_VIRTUAL_TERMINAL_INPUT` (`term_windows.go:30-31`).
6. **Line-editor modules**, from the GitHub API, giving last push, licence and stars:

   | Module | Last push | Licence | Stars |
   |---|---|---|---|
   | `chzyer/readline` | 2025-06-20 | MIT | 2318 |
   | `ergochat/readline` | 2025-01-21 | MIT | 53 |
   | `peterh/liner` | 2023-06-23 | MIT | 1098 |
   | `reeflective/readline` | 2026-09-30 | Apache-2.0 | 150 |
   | `nyaosorg/go-readline-ny` | 2026-09-07 | MIT | 36 |
   | `c-bata/go-prompt` | 2025-08-12 | MIT | 5502 |

   No sibling `go.mod` requires any of these.
7. **gobble's import rules**, `internal/archtest/check.go`:
   - Rule 5 (`:253-275`) lets only `internal/tui` import `charm.land/…`, `github.com/charmbracelet/…` or any `github.com/maccavelli/go-tui-lib` package. The one exception is `tuitest` in tests.
   - Rule 6 (`:277-289`) forbids `internal/cli` from importing `agent`.
   - Rule 2 (`:200-218`) keeps `github.com/coder/acp-go-sdk` in `acpserver`, `acpclient` and the root `tools.go`.
   - Rule 8 (`:306-322`) keeps go-selfupdate-lib in `internal/cli`.
   - No rule mentions `alecthomas/kong`.
8. **go-tui-lib.** The local checkout's `main` is a one-file tree. The library is on `origin/main` at `45ead16` (2026-10-04), `v0.2.0-5-g45ead16`; the newest tag is `v0.2.0`.
   - Top-level packages are `glyph`, `layout`, `termcap`, `theme`, `tuitest` and `workspace`. There is no `command` or `stream` directory.
   - Its accepted records plan these packages:
     - `command`, which imports `when` and `bubbletea` (its 0006-MADR §1);
     - `command/kongcmd`, a nested module importing `command`, `theme`, `glyph` and Kong, that "generates" completion "for bash, zsh, fish and PowerShell" (0006-MADR A1);
     - `safetext`, standard library plus `glyph`, and `stream` with a `Plain` renderer importing `x/ansi` and `workspace` (its 0009-MADR §1).
   - Its 0006-PLAN and 0009-PLAN are `status: proposed`.
9. **Sibling CLI code**, read-only at the commits in the Evidence index:
   - magic-cli-remote splits shutdown signals by OS. On Windows it uses `os.Interrupt` only, with the comment that Windows never delivers SIGTERM (`internal/cli/signals_windows.go:7-17`).
   - mcp-server-magictools `internal/ui/init_windows.go:14-32` sets VT processing on stdout and stderr, but discards the `SetConsoleMode` error and always returns nil.
   - prepare-commit-msg `internal/ui/open.go:13,39` opens a URL with no shell and validates it.
   - go-llmprovider-sdk `wizard.ConfigureLLM` and `TextPrompter` (`configure.go:132`, `text_prompter.go:62`) form a line-oriented configure flow that masks secrets.
   - go-selfupdate-lib `selfupdate/cli` maps contradictory flags to exit 1 (`command.go:39-44`). gobble's 0004-PLAN amendment of 2026-10-01 wants exit 2.
   - magic-cli-remote's `0157-MADR` rolling-log design is `status: proposed` and has no code in that tree.
   - No sibling has a Go line editor, a NO_COLOR helper outside Charm, or a Markdown renderer outside go-tui-lib's planned TUI packages.
10. **Windows Console Host, measured on this host** (Windows 11 25H2, build 26200.9457; `conhost.exe` 10.0.26100.1). A scratch Go program built against `x/term` v0.46.0 and `x/sys` v0.48.0 ran in a new `conhost.exe` window; its `GetConsoleWindow` class was `ConsoleWindowClass`. Key events were injected with `WriteConsoleInputW`. A real paste was triggered with conhost's Paste command (`WM_COMMAND 0xFFF1`), so no keystrokes went to another window. The clipboard was saved first and restored after.
    - Before raw mode, the input mode was `0x01f7` and the output mode `0x0003`. After `term.MakeRaw` and a `SetConsoleMode` adding VT processing: input `0x03f0` (VT input on, processed input off) and output `0x0007` (VT processing on). The `SetConsoleMode` error was nil.
    - **Injected events, read from `CONIN$`:**
      - Ctrl+C arrived as the single byte `03`, in one read.
      - Enter arrived as `0d`.
      - `é` and U+1F600 arrived as UTF-8 `c3 a9 f0 9f 98 80`.
      - A burst of `a⏎b⏎c⏎` arrived in one read: `61 0d 62 0d 63 0d`.
    - **A real three-line paste with DECSET 2004 on** arrived in one read as `ESC[200~line1\rline2\rline3ESC[201~`. Conhost sends bracketed paste, and it turns the clipboard's CRLF into CR.
    - **The same paste through `term.Terminal.ReadLine`** with `SetBracketedPasteMode(true)` returned `line1`, then `line2`, each with `ErrPasteIndicator`. `line3` had no terminating CR, so it stayed in the edit buffer.
    - **Closing `CONIN$` while another goroutine was blocked in `Read` hung the process.** The first probe never exited and had to be stopped. With one long-lived reader and no close, the second and third probes exited at once.
11. **Go 1.27.1 API** (`$GOROOT/api/go1.25.txt`, `go1.26.txt`, `go1.27.txt` of the installed `go1.27.1 windows/amd64`). 0004-MADR's idiom list matches the toolchain:
    - 1.25: `sync.WaitGroup.Go`, `testing/synctest.Test`/`Wait`, `slog.GroupAttrs`, the wider `os.Root` API, `testing.T.Output`.
    - 1.26: `errors.AsType`, `slog.NewMultiHandler`, `testing.T.ArtifactDir`, `os.Process.WithHandle`.
    - 1.27: `encoding/json/v2` and `jsontext`, the `uuid` package with `NewV7`, `strings.CutLast`/`bytes.CutLast`, `testing/synctest.Sleep`, `httptest.NewTestServer`.
    - `slog.DiscardHandler` is present (`go doc log/slog.DiscardHandler`).
12. **`slog` and idiom enforcement, measured.** golangci-lint 2.14.0 (built with go1.27.1) has `sloglint`, `forbidigo`, `depguard`, `modernize`, `intrange`, `copyloopvar`, `usestdlibvars`, `perfsprint`, `usetesting`, `noctx`, `contextcheck` and `fatcontext` (`golangci-lint linters`).
    - A scratch module was configured with sloglint (`no-mixed-args`, `attr-only`, `no-global: all`, `context: scope`, `static-msg`, `key-naming-case: snake`, `forbidden-keys: time, level, msg, source`), depguard (deny `log`, logrus, zap, zerolog) and forbidigo (deny `fmt.Print*`, `print`, `println`, `log.*`, `os.Stdout`, `os.Stderr`, `os.Exit`).
    - `golangci-lint config verify` exited 0.
    - Each plant failed with its rule named:
      - `import 'log' is not allowed` (depguard);
      - `fmt.Println`, `log.Printf`, `os.Stderr` and `os.Exit` forbidden (forbidigo);
      - `default logger should not be used`;
      - `key-value pairs and attributes should not be mixed`;
      - `message should be a string literal or a constant`;
      - `keys should be written in snake_case`;
      - `InfoContext should be used instead`;
      - `the "msg" key is forbidden`.
    - A clean control line was not reported.
13. **magic-cli-remote, second pass,** at `9778cbc1`. These add to, and in one place correct, 0002-MADR's fifth amendment, which read `64282e29`/`772c041e`.
    - **The record names.** magic-cli-remote's provider record is `docs/decisions/0179-MADR-pigo-native-acp-provider.md`, `status: proposed`, created as `pigo` in `72588004` (2026-10-01). `git log --all` finds no `0179-MADR-gobble-…` file, and "gobble" occurs nowhere in its records or Go code. 0179 D8 names `provider.IDPigo`, wire id `"pigo"`, `DefaultBin: "pigo"` and `_pigo/compact`, `_pigo/usage`, `_pigo/set_session_name` and `_pigo/fork` (`0179…:250-271`).
    - **The resolver is unchanged.** 0179 D2 (`OpReporter`) and D3 (strict fallback) are not implemented: `command.Table` is a plain map (`internal/command/command.go:149`), and neither `strict` nor `OpReporter` occurs in `internal/`.
    - **Delivery.** The SDK fork's inbound queue is 1,024 notifications with `OverflowDropNewest`, so any notification can be dropped while the daemon is stalled, with one "dropped" notice per mid-turn session and no re-delivery (`0167-MADR…` D17, D18; `acpagent/session.go:582-603`). Daemon-side, `usage_update` and `available_commands` are dropped when the events channel is full; chunks are retried every 50 ms (`session.go:1263-1305`, `:1427-1437`). Chunks are coalesced in 80 ms windows with an 8 KiB cap, and non-terminal `tool_call_update`s per tool id are merged, latest wins (`0024-MADR-stream-coalescing.md`; `internal/chunkbuf/chunkbuf.go:28-33`, `:84-110`).
    - **Tool and permission text.** Tool cards carry `ToolName` (the tool-call title, else kind) and a `Text` summary cut to 400 bytes for `tool_call` and 600 for updates (`session.go:1668-1705`). The summary turns a diff into `"diff <path>"` and terminal content into `"terminal <id>"`, drops raw JSON, and ignores locations (`:2556-2609`). Permission detail is the same 400-byte summary (`:2113-2118`).
    - **Config options and modes.** Only ungrouped `select` options and booleans are forwarded, without the category (`session.go:1953-1958`). Agent-advertised modes are never marked dangerous (`:1815-1826`).
    - **Replay on `session/load`.** Replayed events are added to the daemon's history only when no event with the same (type, text, tool id, agent session id) exists, and are never broadcast (`internal/session/manager.go:270-272`, `:369-401`, `:1483`). Before the pump attaches, the 256-slot events channel drops the oldest events (`session.go:1446-1461`). The daemon's own 32 MiB ring and disk file are the transcript of record (`manager.go:49`, `store.go:265-341`).
    - **Limits.** Inbound frames are capped at 1 MiB, and the phone's frame budget is a fixed `1<<20` (`internal/ws/server.go:298-300`; `apps/mobile/lib/data/protocol/frame_budget.dart:7`). Outbound broadcast and replay are not size-checked (0181, `proposed`).
    - **Phone presentation.** The context chip turns colour at 75% and 90% (`apps/mobile/lib/features/chat/chat_screen.dart:3326-3380`). Tool cards are one-line status tiles that expand to the text summary, with no diff view (`chat_bubble.dart:291-331`). The composer offers only `remote_commands` marked available (`chat_screen.dart:606-645`).
    - **Process lifecycle** (`0179…:371-384`): one process per session plus an optional warm spare, claimed only when argv and cwd match and re-armed at turn end (`acpagent.go:157-162`, `:791-794`, `:810-817`). `initialize` and `session/new` have 30 s, `session/load` 120 s; close is bounded at 2 s and then the process group is killed.
    - **Version pin.** `agentInfo.version` is compared on major.minor.patch, ignoring a leading `v` and any pre-release or build suffix. An unparseable version never matches and logs a warning (`internal/provider/version.go:8-40`; `acpagent/version.go:18-56`).
14. **Pi does not lock session files.** `packages/coding-agent/src/core/session-manager.ts` has no lock. Pi locks only `auth-storage.ts`, `settings-manager.ts` and `trust-manager.ts` (search for `lock` at `f5d2004`). magic-cli-remote runs one `gobble acp` process per session (measurement 13), and the CLI resumes sessions from the same store (D12). So without a lock, two gobble processes can append to one session file.

### Findings

F1. **Bare `gobble` is specified twice, differently.**
- 0005-MADR "CLI surface" says `gobble [prompt…]` is "native terminal CLI, the default; print mode with -p or piped stdin".
- 0006-MADR D6, confirmed by its 2026-10-04 amendment, rejects "bare session (goose-as-default-session)".
- What D6 measured in goose is a specific coupling. `None` calls `handle_default_session`, which runs the configure wizard when no config exists and otherwise opens a session (`cli.rs:2767-2805`).
- Pi's bare `pi` on a terminal opens its interactive mode (`main.ts:112-123`).
- Consequence: the default mode needs a bare entry point. What 0006 rejected is goose's auto-configure coupling, not the entry point itself. D1 resolves this.

F2. **The one-shot command has three names.**
- [0002-PLAN](0002-PLAN-cli-acp-headless-mcp-v1.md) Phase 2 step 2 says `gobble prompt "…"`.
- 0005-MADR says `gobble -p "…"` or piped stdin.
- 0004-PLAN Phase 2 step 5 says "a session command and a one-shot command that share flags".
- Pi selects print mode with `--print`, or when stdin or stdout is not a terminal (`main.ts:112-123`).
- goose uses a separate `run` command (`cli.rs:983-1002`), and makes you pass `-i -` to read stdin (`cli.rs:219-300`).
- Consequence: one command with Pi's mode resolution gives one flag set by construction, which is what 0006 D9 wanted, and it removes two of the three names. D1.

F3. **The one-shot output formats disagree.**
- 0005-MADR "Clients and modes" lists `--output-format text|json|stream-json`, with `stream-json` being "each ACP `session/update` params object as one JSONL line after a header line".
- 0006 D10, confirmed in its amendment, says "text or json only … Do not add `stream-json`".
- Pi's `--mode json` is a stream: a session header record and then every event (`print-mode.ts:108-127`, `json-event.ts:1-61`, `docs/json.md:15-27`). Pi users treat it as a machine contract.
- goose's `--output-format` has all three (`cli.rs:302-331`).
- Consequence: a streaming machine format is the one Pi feature a script cannot get any other way. D9 resolves this.

F4. **The exit-code rules disagree.**
- 0004-PLAN Phase 2 step 6 says 0 ok, 1 runtime, 2 usage, 3 authentication, 4 cancelled, and 130 when SIGINT ended the process. The 0004-PLAN 2026-10-01 amendment adds 2 for contradictory update flags, and 4 or 130 for a cancelled update.
- 0005-MADR "CLI flags" takes Pi's print-mode rule: 1 on error or abort, 129 on SIGHUP, 143 on SIGTERM (`print-mode.ts:50-66`, `:145-147`).
- In Pi's json mode, a failed assistant turn still exits 0 (`docs/cli-integration.md:46`).
- Consequence: the 128-plus-signal convention is a superset of both tables. The one real conflict is "abort", and D10 resolves it.

F5. **Rule 5 makes the CLI mode Charm-free (measurement 7).**
- `internal/cli` cannot import lipgloss, glamour, colorprofile, `x/ansi` or bubbletea. It cannot import any go-tui-lib package either, including `glyph`, which imports only the standard library.
- Consequence: the CLI mode's dependencies are Kong, `x/term` and `x/sys`. Colour, width, sanitising and Markdown styling in that mode are gobble code or standard-library code.
- This matches the owner's "lightweight", and it keeps CLI mode separate from TUI mode. D2.

F6. **Kong has no completion generator, and 0006 D7 cannot be met by Kong (measurements 3, 4).**
- The only maintained add-on covers three of the four required shells.
- go-tui-lib's planned `kongcmd` covers all four. But it has no code, it imports `theme`, which pulls in Charm, and rule 5 forbids it. Its 0006-PLAN is proposed.
- 0006 D6 rejects hidden commands, which rules out the usual `__complete` callback design.
- Consequence: a static generator that walks `kong.Model` is the only route that meets D7 with no new module and no hidden command. D11.

F7. **No sibling has a line editor, and `x/term`'s `Terminal` meets most of 0006 D12 (measurements 5, 6).**
- It already provides history, a completion hook and bracketed paste. Its only module dependency is `x/sys`, which gobble already requires (indirectly, `v0.48.0`). Seven siblings require `x/term` directly at `v0.43.0`.
- It misses D12 in two places:
  - Ctrl+C ends the read with `io.EOF` instead of clearing the line;
  - a multi-line paste returns once per line.
- Both can be closed outside the editor, in a reader that maps the Ctrl+C byte and a loop that coalesces `ErrPasteIndicator` lines.
- The third-party editors add a module, are stale, are large, or are prompt frameworks with their own rendering (measurement 6).
- Consequence: build on `x/term` with a thin adapter. D6.

F8. **Pi has no line-mode REPL at all.**
- Its modes are `interactive` (full-screen), `print`, `json` and `rpc` (`main.ts:112-123`; `src/modes/` holds `interactive/`, `print-mode.ts`, `json-event.ts` and `rpc/`).
- Its built-in slash commands are handled only in `modes/interactive/interactive-mode.ts:3157-3299`.
- goose's interactive session is a line REPL (`session/input.rs:116-218`, `session/mod.rs:561-643`).
- Consequence: gobble's line session follows goose's interaction design, and its machine-facing contracts follow Pi's.

F9. **goose's streaming Markdown is a stable-prefix buffer, not a renderer.**
- `MarkdownBuffer.push` releases only text that cannot change meaning when more arrives: no open fence, table, heading, inline code, emphasis, link or image. It keeps a resumable checkpoint (`session/streaming_buffer.rs:225`, `:292`, `:317`, `:328`).
- It is flushed before every tool event (`session/output.rs:385-483`).
- The released text is then printed as Markdown *source* with syntax colouring through bat (`session/output.rs:1137-1179`). The markers are kept and nothing is reflowed. Off a terminal it is raw text (`:1150-1152`).
- Pi's print mode writes the final text raw, with no ANSI (`print-mode.ts:139-156`).
- Consequence: colouring Markdown source in place, from a stable-prefix buffer, gives clean streaming with no layout engine and no reflow. Copying the output still yields valid Markdown. D7.

F10. **goose sanitises before printing, which a model-driven terminal needs.**
- `sanitize_terminal_line` strips ANSI and control characters except tab from tool output (`session/output.rs:651-698`).
- Approval text also escapes the bidirectional controls U+061C, U+200E, U+200F, U+202A–U+202E and U+2066–U+2069 as `\uXXXX` (`:666-667`, `:1847-1848`).
- The window title is sanitised (`:1542-1555`).
- go-tui-lib's planned `safetext` is the same idea, standard library plus `glyph` (measurement 8), and is not shipped.
- Consequence: every model- or tool-originated string is sanitised before it reaches the terminal. D8.

F11. **goose's turn and prompt control is cheap and high-value.**
- At the prompt, Ctrl+C clears a non-empty line. On an empty line it shows a hint, and a second Ctrl+C exits (`session/input.rs:49-86`; `session/completion.rs:515-541`).
- During a turn, a per-turn Ctrl+C task cancels the stream (`session/mod.rs:1319-1325`, `:1529-1535`).
- After a cancel, dangling tool calls get an error result, "Interrupted by the user to make a correction", so the transcript stays valid for providers (`session/mod.rs:1646-1723`, `:1669-1675`).
- Consequence: the line client maps Ctrl+C during a turn to ACP `session/cancel`. Transcript repair is the agent's job, not the client's. D5 and D13.

F12. **goose shows the context budget without a status bar.**
- Before every prompt it prints a 20-cell bar: green under 50%, yellow under 85%, red above. It shows `N%` and `12k/200k` (`session/output.rs:1565-1613`).
- It prints elapsed time per turn (`session/mod.rs:791-802`, `:2633-2642`), and `--stats` prints time to first token and tokens per second to stderr (`:2052-2125`).
- ACP `usage_update` carries the same data (0005-MADR "Driven by mcremote").
- Consequence: the CLI mode gets a footer's information as one line per turn. D7.

F13. **goose's tool output is bounded and readable.**
- Each call gets a `▸ tool` header (`session/output.rs:589-603`).
- On a terminal, output over 20 lines shows the first 10 and the last 10, dimmed and sanitised. Off a terminal it is raw (`:720-761`).
- Parameters are truncated to the terminal width (`:1354-1433`).
- Defects not to copy:
  - the truncation hint says "/toggle to show all", and no `/toggle` exists in any crate (`:751`);
  - the width check compares bytes with columns (`:1361`);
  - path shortening splits only on `/` (`:1435-1482`).
- Consequence: D7 ports the bounds and fixes the three defects.

F14. **goose's external-editor compose is small, safe and portable.**
- The editor comes from `$GOOSE_PROMPT_EDITOR`, then `$VISUAL`, then `$EDITOR` (`session/editor.rs:15-229`).
- The command is split into argv without a shell, keeping Windows backslashes unless quotes are present (`:131-139`), and run with no shell (`:142-172`).
- Consequence: `/edit` is the multi-line compose path in a line editor that holds one line. D6.

F15. **Pi's machine contracts are precise and portable.**
- Mode resolution: `--mode rpc` gives rpc, `--mode json` gives json, and `--print` or a non-terminal stdin or stdout gives print (`main.ts:112-123`).
- Every non-interactive mode diverts incidental stdout writes to stderr and keeps stdout for the result (`main.ts:129-131`, `:652-655`; `core/output-guard.ts:45-108`).
- Diagnostics are prefixed `Error:` or `Warning:` on stderr (`main.ts:99-105`).
- Piped stdin is read whole and trimmed (`main.ts:80-97`).
- `@path` attaches a text file as `<file name="ABS">\n…\n</file>\n`, or attaches an image (`cli/file-processor.ts:25-88`).
- The first prompt is assembled from stdin, file text and the first message with `parts.join("")` (`cli/initial-message.ts:40`), so stdin `foo` and prompt `bar` become `foobar`. That is a defect not to copy.
- Consequence: D3 and D4 take the contracts and fix the join.

F16. **Pi's session and model flags are the vocabulary Pi users know.** 0005-MADR "CLI flags" already adopts most of them at 1.0:
- `-c/--continue`, `--session <path|id|prefix>`, `--session-id`, `--fork`, `--no-session`, `--session-dir`, `-n/--name`;
- `--provider` (which requires `--model` since 1.0.2, `main.ts:469-474`), `--model provider/id[:thinking]`, `--thinking`, `--api-key`;
- `-t/--tools`, `--exclude-tools`, `--no-tools`.

These sit in `args.ts:71-262` and `main.ts:252-536`. Pi's multi-letter short forms (`-nt`, `-xt`, `-nbt`, `-nc`, `-na` and the like) are not POSIX short options. Consequence: D12.

F17. **Several Phase 2 inputs need correcting before they are built.**
- **SIGTERM on Windows.** 0004-PLAN Phase 2 step 1 registers `syscall.SIGTERM`, which Windows never delivers (measurement 9).
- **No rolling-log code to copy.** Step 4 cites magic-cli-remote 0157 as a "pattern", and 0157 has no code (measurement 9).
- **VT and `NO_COLOR` in goose.** goose's VT enabling runs through `console::colors_supported()`, which returns before enabling when `NO_COLOR` is set or the stream is not a terminal (cargo registry `console-0.16.6/src/windows_term/mod.rs:62-72`, `:133-140`). With `NO_COLOR`, goose's spinners repeat on Console Host, and the console mode is never restored.
- Consequence: D2 and D14.

F18. **The configure and credential flows exist in the provider SDK.**
- `wizard.ConfigureLLM` with `TextPrompter` is line-oriented and masks secrets.
- `auth.StartDeviceOAuth` suits a line terminal.
- `catalog.List` is bounded at 10 s with a static fallback (measurement 9).
- Rule 4 requires them to be reached through `llm/provider`.
- Consequence: 0006 D13's `configure` and D14's `info --check` are wiring, not new code. D15.

F19. **Console Host sends bracketed paste, and `x/term` splits it (measurement 10).**
- D6's [unverified] premise is now measured. Conhost on this build answers DECSET 2004 with `ESC[200~ … ESC[201~` and CR line ends.
- So the Windows burst-detection fallback is not needed.
- `Terminal.ReadLine` still returns one line per CR inside a paste, flagged `ErrPasteIndicator`, and leaves an unterminated last line in the buffer. Coalescing is gobble's job.
- Ctrl+C reaches Go as `0x03`, and UTF-8 survives.
- Consequence: D6 is revised, and the fallback is struck.

F20. **Closing the console input while a read is pending hangs the process (measurement 10).**
- Consequence: one goroutine owns stdin for the process lifetime, and gobble never closes `os.Stdin` or `CONIN$` to stop a read. It restores the console modes and exits. D6.

F21. **The Go 1.27.1 idioms 0004-MADR lists are real (measurement 11), and `slog` discipline is machine-enforceable (measurement 12).**
- Each rule was seen to fail on its own plant: std `log`, `fmt.Print*`, direct `os.Stdout`/`os.Stderr`/`os.Exit`, the global logger, mixed arguments, a dynamic message, non-snake_case keys, a missing context, and the reserved keys.
- Consequence: D18 makes these lint rules, not guidance.

F22. **gobble's records name a magic-cli-remote record that does not exist (measurement 13).**
- 0002-MADR's fifth amendment says the mcremote half of the contract is `0179-MADR-gobble-native-acp-provider.md` with `provider.IDGobble`. The real file is `0179-MADR-pigo-native-acp-provider.md`, with `IDPigo`, `DefaultBin: "pigo"` and `_pigo/*` methods.
- gobble's identity is `gobble` and its extension prefix is `_gobble/` (0003-MADR; 0002-MADR "TypeScript Pi RPC → v1 mapping").
- Consequence: 0002's citation is corrected (D14). The rename in magic-cli-remote is a companion change (D20).

F23. **Any ACP notification gobble sends to mcremote can be dropped, with no re-delivery (measurement 13).**
- Which one is dropped depends on the daemon, not on gobble. The daemon also merges non-terminal tool updates and coalesces chunks.
- Consequence: gobble's stream must be **snapshot-shaped**. Every update states its whole current value, so any single loss is repaired by the next update. Terminal states and the end of a turn are never left to a single delta. Gobble also keeps the update count low. D19.

F24. **What the phone shows of a tool or a permission is its title plus the first 400 or 600 bytes of a summary that skips diffs, terminals and raw JSON (measurement 13).**
- Consequence: gobble writes a meaningful tool-call title and puts a short, self-contained text block first, before any diff or terminal content. D19.

F25. **Replay on `session/load` is dedupe-sensitive and lossy at the head (measurement 13).**
- The daemon already keeps its own transcript and needs replay only to fill gaps.
- Replayed events are matched on exact text, so differently chunked text duplicates. A long replay loses its first events to the 256-slot channel.
- Consequence: gobble replays compactly: one chunk per message, and each tool call once, in its final state. D19.

F26. **The phone shows config options only when they are ungrouped selects or booleans, and it does not yet read categories (measurement 13).**
- Consequence: gobble declares `model` and `thought_level` as ungrouped select options and sets their category, so they work on the phone's config sheet today and under 0179 D6 later. D19.

F27. **The phone colours context use at 75% and 90%.** D7 took goose's 50% and 85% (measurement 13; F12).
- Consequence: one product should use one threshold. D7's thresholds become the phone's.

F28. **Under mcremote, each `gobble acp` process lives for one session, is killed 2 s after close, and may be started in advance as a spare (measurement 13).**
- Consequence:
  - start-up before `initialize` must be cheap and network-free;
  - argv must not vary per session;
  - each session entry is written before it is acknowledged, so a kill loses nothing written;
  - stdin EOF ends the process.

  D19.

F29. **Two gobble processes can write one session file (measurement 14).**
- A phone-driven `gobble acp` and a terminal `gobble -c` in the same directory resolve to the same store.
- Consequence: an exclusive per-session writer lock. D19.

F30. **Dynamic completion needs the shell to call the program back, and an environment variable can do that without a hidden command.**

The mature implementations:

| Implementation | How the shell calls back | Shells and descriptions | Problems |
|---|---|---|---|
| Cobra `v1.10.2` | A hidden `__complete` command (`completions.go:31-34`, `:236`) | All four shells, with descriptions and a directive bitmask (`:56-95`, `:258-293`) | `eval`s the user's typed line in all four shells (`bash_completionsV2.go:84`, `zsh_completions.go:151`, `fish_completions.go:59`, `powershell_completions.go:129`) |
| clap `clap_complete-v4.6.9` | No subcommand: it checks an environment variable, `COMPLETE=<shell>` (`env/mod.rs:215-252`) | — | Its PowerShell script runs `Invoke-Expression` on the typed line (`env/shells.rs:336-369`) |
| posener/complete | bash's `COMP_LINE`/`COMP_POINT` | No PowerShell | — |
| kongplete `v0.4.0`, kong-completion `v0.0.14` | Through posener v1 | No PowerShell, no descriptions | Neither handles Kong's `default:"withargs"` root, negatable flags or passthrough, so both would complete gobble's D1 root wrongly |

Kong v1.16.1 exposes what a completer needs:
- the model: `Kong.Model`, and `Node` with `Children`, `Flags`, `Positional`, `DefaultCmd`, `Aliases`, `Hidden` and `Passthrough` (`model.go:41-60`);
- `Flag` with `Short`, `Aliases`, `Negated` and `Hidden`, and `Value.EnumSlice`, `IsBool` and `IsCounter` (`model.go:258-359`, `:422-433`);
- raw custom tags through `Tag.Get` (`tag.go:358-364`);
- the `withargs` rule, which a completer must copy (`context.go:399-403`, `:569-576`).

`kong.Trace` runs mapper `Decode` and stops at the first error, so it is a test oracle, not an engine (`context.go:102-116`). `Node.AllFlags` has a side effect, setting `flag.Active` (`model.go:95-110`).

The PowerShell hazards:
- **Empty arguments.** An empty argument to a native executable must be special-cased on 5.1, or under `$PSNativeCommandArgumentPassing = Windows`, which is pwsh 7.6's default on this host (measured).
- **Output encoding.** 5.1 decodes native output with `[Console]::OutputEncoding`. That is 65001 on this host but follows the OEM code page (437 here) by default.

Consequence: D17.

## Decision Drivers

- **Lightweight.**
  - The CLI mode adds Kong and `golang.org/x/term`, and nothing else, to `internal/cli`. *(2026-10-04, while writing the PLAN: plus `golang.org/x/text`, for its `width` package only, the Go team's East Asian Width tables, used by D7's table layout and truncation in place of a hand-written table. magic-cli-remote already requires `v0.42.0`.)*
  - No Charm, no syntax-highlighting engine and no Markdown AST library.
  - No network before the first prompt, and no automatic update check (0005-MADR, Self-update: "`gobble acp`, print mode and the TUI never check on their own").
- **Clean.**
  - Stdout carries only results.
  - One flag set and one registry for slash commands.
  - Every escape sequence that reaches the terminal is either gobble's own or was sanitised.
  - Copying output yields valid Markdown.
- **Feature-rich where it is cheap:** streaming Markdown, bounded tool output, a context-budget line, history, completion, external-editor compose, two-stage Ctrl+C, and a streaming machine format.
- **One command API.** The CLI is an ACP client and never calls the agent loop (0002-MADR; archtest rule 6). Anything the CLI shows comes from ACP `session/update`.
- **Canonical code first.** Import a sibling where the rules allow it (go-selfupdate-lib, the provider SDK through `llm/provider`). Copy a sibling only where no import is possible. Port behaviour, not Rust or TypeScript.
- **Reconcile, don't stack.** Where earlier records disagree (F1–F4), this record chooses once and says which text it supersedes.
- **Native magic-cli-remote integration** (owner, 2026-10-04). gobble is magic-cli-remote's native agent harness. Its ACP stream, its terminal presentation and its session store are shaped by what magic-cli-remote measurably does. They never depend on detecting it.
- **Enforced, not advised.** A Go idiom or logging rule that matters is a lint rule seen to fail on a plant (F21).

## Considered Options

- **A. A native line client on Kong, `x/term` and the standard library: goose's terminal interaction, Pi's machine contracts, and Markdown coloured in place from a stable-prefix stream.**
- **B. Allow Charm in `internal/cli`** (lipgloss, glamour, and bubbles' textinput) and share go-tui-lib's theme with the CLI mode.
- **C. Follow Pi literally.** No line REPL: bare `gobble` on a terminal opens the TUI, and the CLI mode is print and json only.
- **D. A third-party editor and renderer:** `reeflective/readline` or `chzyer/readline` for input, `goldmark` to parse Markdown, and `chroma` to highlight code.

## Decision Outcome

Chosen option: **A**. It is the only option that keeps both the accepted default mode (a native line CLI, 0002 and 0005) and the import rules, adds no module beyond Kong and `x/term`, and still carries the features the research ranked highest in goose and Pi.

### The decisions

**D1. One default command with Pi's mode resolution. This supersedes F1's and F2's conflicting text.** gobble's root has a Kong default command, `chat`, marked `default:"withargs"`. Its shared flag set is the D12 set. It resolves the mode as follows:

| Condition | Mode |
|---|---|
| `-p/--print` given, or stdin is not a terminal, or stdout is not a terminal | print (D9) |
| otherwise | line session (D5) |

Further rules:
- A prompt given in a line session is sent as the first turn, and the session continues.
- **Bare `gobble` on a terminal opens the line session.** It never runs the configure wizard. With no usable credential it still opens, and prints one `Warning:` line naming `gobble configure`. A turn sent without a credential fails with the authentication error, and in print mode the process exits 3 (D10).
- This keeps 0006 D6's rejection of goose's auto-configure coupling, and replaces D6's "bare session" line with this rule.
- **`gobble prompt` (0002-PLAN Phase 2 step 2) is replaced by `gobble -p`.** 0004-PLAN Phase 2's "session command and one-shot command" become this single command.
- The TUI stays explicit. It is never chosen by this resolution (0005-MADR).

**D2. The terminal layer, decided once at start.** A small package `internal/cli/term` holds the following.
- **Capabilities.**
  - Per-stream `IsTerminal`, from `x/term`.
  - Colour is off when the stream is not a terminal, when `NO_COLOR` is non-empty, or when `TERM=dumb`. It is forced on by `CLICOLOR_FORCE=1`.
  - Width comes from `term.GetSize`, then `COLUMNS`, then 80.
  - Glyphs use Unicode, with an ASCII fallback for `TERM=dumb` or when stdout is not UTF-8-capable.
- **Styles.** 16-colour SGR only: bold, dim, italic, underline and the eight foreground colours. There are no true-colour themes in this mode. Themes are a TUI feature (0005-MADR, 1.x).
- **Windows.**
  - Before any output, enable `ENABLE_VIRTUAL_TERMINAL_PROCESSING` on stdout and stderr, whatever `NO_COLOR` says, and restore the saved modes on exit.
  - Copy mcp-server-magictools' `EnableVirtualTerminalProcessing`, with its discarded `SetConsoleMode` error returned (F17).
  - This refines 0006 D8.
- **Signals.**
  - Interrupt plus SIGTERM and SIGHUP on Unix; Interrupt only on Windows. This is magic-cli-remote's per-OS split.
  - It corrects 0004-PLAN Phase 2 step 1 (F17).

**D3. Output discipline (Pi, F15).**
- Stdout carries the result only: the transcript in a line session, and the D9 format in print mode.
- Diagnostics go to stderr, prefixed `Error:` or `Warning:`.
- Logging never writes to stdout (0004-PLAN Phase 2 step 4 already holds for `gobble acp`).
- A broken pipe (`EPIPE`) on stdout ends the process quietly with status 0, as goose's list commands do (`commands/session.rs:170-176`).
- The stdout writer is the only path to `os.Stdout`, enforced by a test that greps `internal/cli` for other writes.

**D4. Input composition (Pi, F15, with the join fixed).**
- The first prompt is assembled from three parts, joined by a blank line (`"\n\n"`), not `""`:
  1. piped stdin, read whole and trimmed;
  2. each `@path`, wrapped as `<file name="ABS">\n…\n</file>`;
  3. the positional words.
- **Positional words are joined with single spaces into one prompt**, so `gobble fix the failing test` is one request. This departs from Pi's one message per positional (`print-mode.ts:135-137`).
- An image `@path` becomes an ACP image content block.
- A missing `@path` is a usage error, exit 2.
- 0006 D10's exclusivity (text, file or stdin) applies to the explicit `--file` form only. Composition above is the default path.
- A failed stdin read is an error, never a panic (0006 D10).

**D5. The line session: goose's interaction design over ACP (F8, F11).**
- **Prompt and exit.** The prompt is `> `. An empty line is ignored. `/exit`, `/quit` and Ctrl+D on an empty line exit 0.
- **Ctrl+C at the prompt** clears a non-empty line. On an empty line it prints the hint `Press Ctrl+C again to exit`, and a second Ctrl+C within 2 s exits 0.
- **Ctrl+C during a turn** sends ACP `session/cancel` and returns to the prompt. A second Ctrl+C while cancelling exits 130.
- **Turn display.**
  - A spinner on stderr while waiting, terminal only. It writes `\r` and `ESC[K` from one goroutine.
  - Then the D7 stream.
  - Then the D7 status line.
- **Start-up.** No network call before the first prompt.

**D6. The line editor: `x/term` `Terminal` plus a thin adapter (F7, F14).** This meets 0006 D12.
- **Ctrl+C.** A reader in front of the terminal maps the Ctrl+C byte, `0x03`, to the D5 behaviour, so Ctrl+C never reaches `Terminal` as EOF.
- **Paste.**
  - Bracketed paste is on. Lines returned with `ErrPasteIndicator` are coalesced, and the paste is submitted once, when the next non-paste Enter arrives.
  - ~~On Windows Console Host, bracketed paste under `ENABLE_VIRTUAL_TERMINAL_INPUT` is **[unverified]**.~~ *(Superseded 2026-10-04 by measurement 10 and F19.)*
  - ~~If the PLAN's probe shows it is absent, the adapter detects a burst instead. It treats input as a paste when `GetNumberOfConsoleInputEvents` reports queued events at Enter. This reimplements the idea in goose's `session/paste.rs` through `x/sys/windows`, with no chip UI.~~ *(Superseded 2026-10-04: conhost sends bracketed paste, so there is no burst fallback.)*
  - **2026-10-04, measured (F19):**
    - Console Host sends `ESC[200~ … ESC[201~` with CR line ends.
    - `Terminal.ReadLine` returns each pasted CR-terminated line with `ErrPasteIndicator`, and leaves the unterminated last line in the edit buffer.
    - The adapter appends each flagged line to a pending paste and shows `… N pasted lines` on stderr. The next Enter submits the pending paste, a newline, and the edited last line, as one prompt.
    - There is no burst detection on any platform.
- **Owning stdin (F20).**
  - One goroutine reads stdin for the life of the process and feeds the editor through a channel.
  - gobble never closes `os.Stdin` or `CONIN$` to stop a read. It restores the console mode with `term.Restore` and returns its exit code to `cmd/gobble`.
  - A test asserts that no code path in `internal/cli` calls `Close` on the stdin file.
- **Multi-line input.**
  - A line ending in `\` continues on a `… ` prompt.
  - `/edit [text]` opens `$GOBBLE_EDITOR`, then `$VISUAL`, then `$EDITOR`, then `notepad` on Windows or `vi` elsewhere.
  - The command is split into argv without a shell, using goose's quote-aware rule (`session/editor.rs:131-139`), on a temp file under the state directory that is deleted afterwards.
- **History.**
  - The file is `<state>/history`, mode 0600, one entry per line, with `\n` and `\\` escaped.
  - It keeps at most 1,000 entries and is saved after every submission.
- **Completion.** Through `AutoCompleteCallback`: slash names from the D13 registry, and `@path` from `os.ReadDir`.

**D7. Rendering: coloured source from a stable-prefix stream, with bounded tool output (F9, F12, F13).**
- **Markdown streaming.**
  - Assistant text goes through a Go stable-prefix buffer. It reimplements the behaviour of goose's `MarkdownBuffer`: it releases only text no later chunk can restyle, and it is flushed before each tool event.
  - Released lines are coloured in place. Headings are bold; emphasis, inline code and fences are dimmed or coloured; the fence language label is shown; list and quote markers are coloured.
  - Markers are kept. Text is not reflowed or wrapped, because the terminal wraps.
  - GFM tables are re-laid out with aligned columns, measured in display width.
  - With colour off, the text passes through raw, as in Pi's print mode.
- **Thinking** is hidden unless `--show-thinking` is set, and is then dimmed.
- **Tool calls.**
  - Each ACP `tool_call` prints `▸ <title>`, with its parameters truncated to the width in display columns, not bytes.
  - `tool_call_update` prints the status.
  - Output over 20 lines shows the first 10 and the last 10 with `… N lines hidden (--verbose shows all)`. The hint names a real flag.
  - Diff content prints as a unified diff with `+` and `-` coloured.
  - Paths are shortened under `~` with `filepath`, so Windows separators work.
- **Plan updates** print as a checklist.
- **Status line.** After each turn, one dim line on stderr gives elapsed time, a 20-cell context bar ~~(green under 50%, yellow under 85%, red above)~~, `used/limit` tokens, and the cost when known. The data comes from ACP `usage_update`. `--stats` adds time to first token and tokens per second.
  - *2026-10-04 (F27):* the bar is green under 75%, yellow from 75% and red from 90%, the thresholds of the phone's context chip. So the terminal and the phone show the same warning at the same moment.
- **No syntax highlighting of code in this mode.** It would need `chroma` (option D). It can be revisited by a later record if asked for.

**D8. Sanitise everything that did not come from gobble (F10).**
- Model text, tool output, tool titles, paths and session names are stripped of escape sequences and control characters except tab and newline before they are styled.
- Permission text also escapes the bidirectional controls as `\uXXXX`.
- The terminal title, if set, is sanitised.
- This is gobble code in `internal/cli/term`, standard library only.
- When go-tui-lib ships `safetext` under a tag, a later record may narrow rule 5 to admit it and replace this code. That is the canonical-library rule, not a reason to wait.

**D9. Print mode: three formats over the ACP vocabulary. This supersedes 0006 D10's "text or json only".** `--output-format` takes one of three values:

| Format | Output |
|---|---|
| `text` (default) | The final assistant text of the last turn, raw, no ANSI |
| `json` | One object, `{text, stopReason, usage, cost, sessionId}`, as 0005-MADR specifies |
| `stream-json` | A header line `{"type":"session","id","cwd","timestamp"}`, then each ACP `session/update` params object as one JSONL line. LF framing, with a final `{"type":"result",…}` object of the `json` shape |

- No private event dialect (0005-MADR).
- In non-interactive mode an `ask` decision resolves to deny unless `--yes` or mode `bypass` (0005-MADR).

**D10. One exit-code table. This supersedes the conflicting text in F4.**

| Code | Meaning |
|---|---|
| 0 | Success, including declined and current updates |
| 1 | Runtime failure, including a turn whose `stopReason` is an error or a provider-side abort, in every output format |
| 2 | Usage: bad or contradictory flags, a missing `@path` |
| 3 | No usable credential |
| 4 | Cancelled by the user at a gobble prompt (a permission or confirmation answered Cancel) |
| 10 | `update --check` found an update |
| 128 + signal | 130 SIGINT, 129 SIGHUP, 143 SIGTERM |

- Pi's json-mode "failed turn exits 0" is not copied.
- go-selfupdate-lib's `cli.Command` is not called. gobble calls `Flags.Request` and `Run` and maps errors to this table (F18; measurement 9).

**D11. Shell completion is a gobble generator over `kong.Model`. This makes 0006 D7 implementable.** *(Superseded 2026-10-04 by D17. The static-script design and "dynamic values are out of this record" no longer hold; the text below is kept as first written.)*
- `gobble completion bash|zsh|fish|powershell` prints a static script, walking the Kong model's commands, flags, aliases and `enum` values.
- There is no hidden completion command (0006 D6), no new module, and no clap.
- Dynamic values such as session ids are out of this record.
- If go-tui-lib's `kongcmd` ships Charm-free, a later record may adopt it.

**D12. The shared flag set (F16; it implements 0005-MADR "CLI flags" for the default command).**
- **Prompt and output:** `-p/--print`, `--output-format`, `--file <path>` (the explicit exclusive input of D4), `--show-thinking`, `--stats`, `--verbose`.
- **Sessions:** `-c/--continue`, `--session <path|id|prefix>`, `--session-id <id>`, `--fork <path|id>`, `--no-session`, `--session-dir`, `-n/--name`.
- **Models:** `--provider` (requires `--model`), `--model provider/id[:thinking]`, `--thinking`, `--api-key`.
- **Tools:** `-t/--tools`, `--exclude-tools`, `--no-tools`.
- **Prompts and context:** `--system-prompt`, `--append-system-prompt` (`--system-prompt` is kept for file input, 0006 D10), `--no-context-files`.
- **Trust, approval and environment:** `--approve`/`--no-approve`, `--yes`, `--offline`, `--cwd`, `--log-level`.
- **Rules.**
  - Long forms only for Pi's multi-letter shorts.
  - Pi's conflict checks are kept, for example `--fork` with `--session` is a usage error.
  - Flags whose behaviour needs a later plan are parsed and rejected with exit 2 and "not yet available" until that plan lands. They are never silently ignored.

**D13. Slash commands in the line session: one registry.**
- The registry is the agent's ACP `available_commands_update`, which by 0002-MADR's rule lists only commands the agent executes. It is unioned with five client-local commands: `/help`, `/exit`, `/quit`, `/edit` and `/new`.
- `/new` is `session/new` in process, 0002-MADR rule 3.
- An unknown `/name` is an error printed locally and is never sent to the model. This adopts 0006 F8's proposed rule for the line session.
- `/skill:<name>` and template commands work because the agent advertises them.

**D14. Phase 2 corrections, recorded here and executed by amending 0004-PLAN.**
- **Signals.** Step 1's signal list becomes D2's per-OS list.
- **Rolling log.** Step 4's rotator is written from 0157-MADR's design. It is not "copied from a pattern": 0157 has no code. The parameters are 10 MiB per file, 5 backups and 28 days, as 0157-MADR states.
- **Kong usage rules,** from go-tui-lib's probed findings (its 0006-MADR A1):
  - `kong.Name("gobble")` and `kong.Writers(stdout, stderr)` on every parser;
  - an `Exit` that panics a sentinel, which `cli.Main` recovers into a D10 code;
  - never `kong.Parse`, `Fatalf` or `FatalIfErrorf`.
- **archtest rule 9.** Only `internal/cli` may import `github.com/alecthomas/kong`. 0004-MADR states this rule, and no rule enforces it (measurement 7).
- **0002-MADR's fifth amendment citation (F22).** A dated correction replaces `0179-MADR-gobble-native-acp-provider.md` and `provider.IDGobble` with the file and names that exist, `0179-MADR-pigo-native-acp-provider.md` and `provider.IDPigo`. It notes that the rename to `gobble` is the D20 companion item. The amendment's other facts stand; measurement 13 confirms them at `9778cbc1`.

**D15. Reuse map. Import where the rules allow it; copy otherwise; port behaviour, never code, from goose and Pi.**

| Need | Source | How |
|---|---|---|
| `update`, `version` | go-selfupdate-lib `selfupdate/cli` (`Flags`, `Request`, `Run`), `buildinfo` | import in `internal/cli` (rule 8) |
| `configure`, login, model list, `info --check` | go-llmprovider-sdk `wizard`, `llmprovider/auth`, `llmprovider/catalog` | import through `llm/provider` (rule 4) |
| URL opening for login | prepare-commit-msg `internal/ui/open.go` | copy, about 35 lines, without its stdout print |
| Directory roots, owner-only files | magic-cli-remote `internal/appdirs` | copy into gobble's `internal/appdirs` on the 0003-MADR table |
| Windows VT | mcp-server-magictools `internal/ui/init_windows.go` | copy, error returned (D2) |
| Signal split | magic-cli-remote `internal/cli/signals_*.go` | copy (D2) |
| `config path` output shape | magic-cli-remote `internal/cli/paths.go` | follow (`key: value`, `--json` snake_case) |
| Line editor | `golang.org/x/term` | import (D6) |
| Stable-prefix Markdown, sanitising, two-stage Ctrl+C, tool bounds, context bar, editor compose | goose (Apache-2.0) | reimplement the behaviour; no code copied, so no notice owed |
| Mode resolution, stdout discipline, input composition, session and model flags, print formats | Pi (MIT) | reimplement the contract; the `NOTICE` already credits Pi |

Copies from Apache-2.0 siblings carry a provenance comment naming the source file. prepare-commit-msg has no licence file. It has the same owner, and the copy's provenance comment says so.

**D16. Phasing.** This record decides behaviour. The plans own the order.
- **0004-PLAN Phase 2.** No agent exists yet, so this phase builds everything except the ACP wiring:
  - the Kong root and the D1 default command, which parses flags and validates them;
  - D2, D3, D4, D6, D10, ~~D11~~ D17 (2026-10-04), D14 and D18;
  - the D7 stable-prefix buffer, Markdown styler, table layout and tool-output bounding, and the D8 sanitiser, each tested on fixtures.
- **0002-PLAN Phase 2 ("Kong is an ACP client").** Wire D1, D5, D7 and D9 to `acpclient`.
- **2026-10-04, D19 by owner.**
  - Items 1, 2, 3, 5 and 6 belong to the agent-side phases that implement them: 0002-PLAN Phases 1, 3, 4 and 6.
  - Item 4 belongs to the phase that adds config options.
  - Item 8 (the session lock) belongs to 0002-PLAN Phase 4, sessions.
  - Item 9 belongs here.
  - Each of those plans gains a dated entry citing D19.
- **0002-PLAN Phase 6.** D13's union with `available_commands_update`, and the permission prompt.
- **The permission prompt.** One line on stderr, `[y] allow once  [a] always  [n] deny  [N] never  [c] cancel`. It is built from the ACP options offered, with bidi escaping and Ctrl+C meaning cancel. Its text and keys are settled in that plan.

**D17. Native, dynamic shell completion, written in Go over Kong's model (F30). It supersedes D11 and implements 0006 D7.**
- **Two entry points, neither hidden.**
  1. `gobble completion bash|zsh|fish|powershell` prints a registration script. It is a visible Kong command. Its help documents the installation (`source <(gobble completion bash)`, `gobble completion fish | source`, `gobble completion powershell | Out-String | Invoke-Expression` in `$PROFILE`), and advises generating the script at shell start, so script and binary never disagree.
  2. The script calls gobble back in **completion mode**. The process switches into it when `GOBBLE_COMPLETE=<shell>` and `GOBBLE_COMPLETE_PROTOCOL=1` are set. This is an environment-variable mode, not a command, so it does not breach 0006 D6. `gobble completion --help` documents it.
  - A protocol mismatch outputs only the error directive. The script then asks the user to re-source it.
- **How words reach gobble.** The current word is always the last word, already cut at the cursor, and it may be empty.

  | Shell | How the words are passed |
  |---|---|
  | zsh and fish | argv after `--` |
  | bash | `GOBBLE_COMPLETE_LINE`, `GOBBLE_COMPLETE_POINT` and `GOBBLE_COMPLETE_CUR`. Go splits `line[:point]` with a POSIX lexer that does no expansion and tolerates an open quote, completes the whole token, and strips the prefix that bash's `=`/`:` word breaks already show |
  | PowerShell | `GOBBLE_COMPLETE_WORDS`, joined with U+001F. Nothing goes in argv, so the 5.1 and 7.x empty-argument handling never applies |

- **Output.**
  - UTF-8, exit 0, written once from a buffer.
  - One candidate per line, as `value` or `value<TAB>description`. Values replace the whole current token, including a `--flag=` prefix.
  - The last line is `:<directive>`, using Cobra's bit values so the meaning is familiar: 1 error, 2 no space, 4 no file completion, 16 directories only, 32 keep order.
  - Control characters are stripped from values and descriptions (D8), and descriptions are one line of at most 80 runes.
  - Files and directories are completed by gobble itself, with directories ending in the OS separator and no space. The scripts never fall back to the shell's file completion, so behaviour is the same on every shell and platform.
- **Scripts.** Each is a `text/template` file under `internal/cli/complete/scripts/`, embedded with `//go:embed`. None uses `eval` or `Invoke-Expression`.

  | Shell | Script |
  |---|---|
  | bash | A `_gobble` function: `mapfile`, `compopt -o nospace`/`-o nosort`, quoting with `printf %q`, registered with `complete -F _gobble gobble`. No bash-completion package required |
  | zsh | `#compdef gobble` with the sourcing guard, `_describe` (with `-V` and `-S ''` per directive), and `:` escaped in values |
  | fish | `commandline -xpc`, falling back to `-opc` for fish 3, plus `commandline -ct`. Registered with `complete -c gobble -f [-k]`, plus the trick that avoids a trailing space |
  | PowerShell, one script for 5.1 and 7 | `Register-ArgumentCompleter -Native -CommandName gobble, gobble.exe`. Words come from `$commandAst.CommandElements`, cut at `$cursorPosition`. The executable is resolved with `Get-Command -CommandType Application`. It sets `[Console]::OutputEncoding` to UTF-8 and restores it, along with `$global:LASTEXITCODE` and the `GOBBLE_COMPLETE*` variables. Results are `CompletionResult` with typed kinds (`ParameterName`, `ParameterValue`, `ProviderItem`, `ProviderContainer`) and a non-empty tooltip, with single quotes doubled for values with spaces or special characters. Plain strings outside FullLanguage mode |

- **The Go side.** A package `internal/cli/complete`, standard library and Kong only:

  ```go
  type Candidate struct {
      Value, Description string
      Kind               Kind // Command, Flag, Value, File, Dir
  }
  type Directive uint8 // Cobra-compatible bits
  type Completer interface {
      Complete(ctx context.Context, prefix string) (iter.Seq[Candidate], Directive)
  }
  type CompleterFunc func(ctx context.Context, prefix string) (iter.Seq[Candidate], Directive)
  ```

  - **Finding the flag or argument.** The walker follows clap's state machine over `kong.Model`:
    - children and aliases, skipping hidden ones;
    - the `withargs` default command, with Kong's two rules for it;
    - long and short names, one-character aliases, and `--no-<name>` or custom negations, copying Kong's unexported rule;
    - `--name=value` and short clusters;
    - `--`, passthrough, and the positional index.

    It never calls `Node.AllFlags` or `kong.Trace`, because both have side effects.
  - **Where values come from.** A field's values come from a custom tag `complete:"<name>"`, read with `Tag.Get`, that names a registered `Completer`. Otherwise they come from `EnumSlice()`. `complete:"none"` means "no values, no files".
  - **A coverage test** fails when a flag that takes a value has neither an `enum` nor a `complete` tag. So no flag silently lacks completion.
- **Values (D12's flag set).**
  - **Static:** `--output-format`, `--thinking`, `--log-level`, `--approve` forms and `completion <shell>` (enums); `--provider` (the provider registry).
  - **Dynamic, from local files only:**
    - `--session` and `--fork`: ids and name prefixes from the session store, newest first with keep-order, each described as `name · date · cwd`;
    - `--model`: `provider/id` from the on-disk catalog cache, then thinking levels after `:`;
    - `-t/--tools` and `--exclude-tools`: tool names, completed after the last comma with no space.
  - **Paths:** files for `--file`, `--system-prompt` and `--append-system-prompt`; directories for `--session-dir` and `--cwd`.
  - **Nothing:** `--api-key`, `--session-id`, `-n/--name`.
- **Safety and speed.**
  - Completion mode starts in `cli.Main` straight after `kong.New` builds the model and before `Parse`. It runs before configuration load, logging, credentials, `x/term`, update checks and any network use.
  - It is read-only. It never creates a directory, never refreshes the model catalog, and never reads secrets or environment values into candidates.
  - It has one deadline of 150 ms through `context.WithTimeout`. When that expires it returns what it has, keep-order, without an error.
  - Directory scans stop early through `iter.Seq`, at most 200 candidates.
  - Session files are read only to a capped header.
  - stderr stays silent unless `GOBBLE_COMPLETE_DEBUG=<file>` names a file to write to.
  - The `GOBBLE_COMPLETE*` variables are removed from the environment before any child process starts.
- **Tests.**
  - Engine table tests (`model + words → candidates, directive`). They cover the `withargs` root, `--model=an`, short clusters, `--no-approve`, `--`, aliases, hidden items and the cut at the cursor, and use `kong.Trace` as the oracle for the command each prefix reaches.
  - Protocol tests that run the built binary with an explicit environment, including U+001F words and multibyte paths.
  - Golden scripts, updated with `-update`.
  - Syntax checks: `bash -n`, `zsh -n`, `fish --no-execute`, and the PowerShell parser under both `pwsh` and `powershell.exe`.
  - Real-shell tests:
    - bash: a `_gobble` call with `COMP_*` set, checking `COMPREPLY`;
    - fish: `complete -C`;
    - PowerShell: `TabExpansion2` under 5.1 and 7 on Windows;
    - zsh: the syntax check plus an optional `zpty` harness.

    The bash test settles whether `COMP_POINT` counts bytes or characters **[unverified]**.
  - Each gate is seen to fail on a scratch copy with a corrupted golden, a removed `complete` tag, and a protocol mismatch.
  - The latency baseline of D18 is a completion request for `--session`.

**D18. Go 1.27.1 idioms and `log/slog` are enforced by lint, module-wide (F21).**
- **Logging.**
  - `log/slog` only. Every call goes through an injected `*slog.Logger`, never the default or global one, and uses the `…Context` methods.
  - Arguments are `slog.Attr` values only. Messages are constants. Keys are snake_case. `time`, `level`, `msg` and `source` are never used as keys.
  - `internal/logging` builds the one handler: `slog.NewMultiHandler` over the rolling file (D14) and stderr (never under `gobble acp`), with the redacting `ReplaceAttr` (0004-PLAN Phase 2 step 4).
  - Tests use `slog.DiscardHandler` or a recording handler.
- **User output is not logging.** D3's `Error:`/`Warning:` lines and every rendered byte go through the injected writers. `fmt.Print*`, `print`, `println`, `log.*` and direct `os.Stdout`, `os.Stderr` or `os.Exit` are forbidden outside `cmd/gobble/main.go`, which passes the writers in and exits with `cli.Main`'s code.
- **Rules in `.golangci.yml`** (the configuration of measurement 12, which was seen to fail on each plant):
  - `sloglint` with `no-mixed-args`, `attr-only`, `no-global: all`, `context: scope`, `static-msg`, `key-naming-case: snake` and `forbidden-keys: [time, level, msg, source]`;
  - a `depguard` rule denying `log`, logrus, zap and zerolog;
  - `forbidigo` for the identifiers above, excluding `cmd/gobble/main.go`;
    ~~*(Amended 2026-10-04, 0008-PLAN P1 deviation: forbidigo runs with `analyze-types: false`. In type-aware mode it missed `os.Stderr.WriteString(…)` and `os.Stdout.Write(…)`, because it matches a method call as one expression and does not report the stream inside it. Measurement 12's plants did not include that form. Literal matching catches it, at the cost that an import alias of `os`, `fmt` or `log` is not caught.)*~~ *(Withdrawn 2026-10-04: a misdiagnosis. golangci-lint reports one issue per line by default (`uniq-by-line`), and the plants wrote `_, _ = os.Stderr.WriteString(…)`, which errcheck reports under `check-blank`, so that report hid forbidigo's. With forbidigo alone and `--uniq-by-line=false`, `analyze-types: true` reports all six forms, an error-checked `os.Stderr.WriteString` included. The measured configuration stands, with `analyze-types: true`.)*
  - `modernize`, `intrange`, `copyloopvar`, `usestdlibvars`, `perfsprint`, `usetesting`, `noctx`, `contextcheck` and `fatcontext`.

  This goes beyond the fleet set adopted in 0007-MADR. The PLAN records the divergence in 0007-MADR as a dated note. The `go fix -diff` gate stays.
- **Idioms, used on purpose in CLI-mode code** (0004-MADR's idiom list, confirmed in measurement 11):
  - `context.Context` first;
  - `iter.Seq` for completion candidates and registry walks;
  - `errors.AsType` and `errors.Join`;
  - `sync.WaitGroup.Go` for the reader, spinner and renderer goroutines;
  - `testing/synctest` for every timer: the two-stage Ctrl+C window, the spinner, and the completion timeout;
  - `encoding/json/v2` for `--json`, `stream-json` and the completion protocol;
  - `strings.CutLast` where a suffix is split;
  - `t.ArtifactDir()` for golden-output diffs.
- **Start-up cost.**
  - No `init()` with side effects.
  - The Kong model is built once per process.
  - Nothing before argument parsing touches the network or the disk beyond reading configuration.
  - The PLAN records `gobble --version` and a completion request as wall-clock baselines on this host.

**D19. Native magic-cli-remote integration, at every level gobble owns (F22–F29).** These rules are gobble's side of the contract. They hold for every ACP client, because gobble selects behaviour by capability and never by client name (0002-MADR fifth amendment). They are tuned to what magic-cli-remote does today, as measured at `9778cbc1`.
1. **Handshake.**
   - `agentInfo` is `{name: "gobble", title: "gobble", version}`. `version` always parses as semver: the release tag without `v`, or `0.0.0-dev+<revision>` for other builds, never `(devel)`. So the `KnownGoodVersion` pin compares and never fails on format.
   - Capabilities:
     - `loadSession: true`;
     - `sessionCapabilities.list` and `close`;
     - `promptCapabilities.image` follows 0005-MADR: false until the SDK's image input lands, then true when the configured default model accepts images. `initialize` precedes any session, so this cannot be per model. An image sent to a model that cannot take one ends the turn with a readable error and is never dropped silently. `audio: false`. `embeddedContext: true`;
     - `mcpCapabilities.http: true`, `sse: false`.

     The rest is 0002-MADR and 0005-MADR as they stand.
2. **Snapshot-shaped updates (F23).**
   - `plan` and `available_commands_update` always carry the full list.
   - `usage_update` carries absolute `used`/`size`/`cost`, sent only when a value changed.
   - Every `tool_call` reaches a terminal status, `completed` or `failed`, in its own update, never folded into a later chunk.
   - `current_mode_update` and `config_option_update` repeat the full current value.
   - Assistant and thought text is coalesced agent-side, at most one chunk per 50 ms or 4 KiB. This keeps the notification count far below the daemon's 1,024-slot queue.
3. **Phone-legible tools and permissions (F24).**
   - A tool-call `title` is a short human phrase, at most 60 runes, such as `edit internal/cli/term.go` or `run go test ./...`.
   - The first content block is a text summary of at most 400 bytes that stands alone. Diffs, terminal output and locations follow it.
   - A permission request's tool call follows the same rule, so the phone's 400-byte cut shows the decision, not a diff header.
   - The plan behind a plan-mode exit is also published as a `plan` update (0005-MADR).
4. **Config options (F26).** `model` and `thought_level` are ungrouped `select` options with `category` set and the model list as `options`. They appear on the phone's config sheet today, and 0179 D6 reads them later.
5. **Compact replay (F25).** `session/load` replays:
   - one `user_message_chunk` per user message;
   - one `agent_message_chunk` per assistant message, with its whole text;
   - one `tool_call` per call, in its final state;
   - then the snapshot updates of item 2.

   It sends no per-token deltas, which keeps replay small enough to survive the daemon's 256-slot channel and to dedupe on exact text.
6. **Process lifecycle (F28).**
   - `gobble acp` answers `initialize` with no network call and no MCP connect; MCP connects on `session/new`.
   - argv is `gobble acp` with nothing per session, so a warm spare matches.
   - Each session entry is written and flushed before the update that reports it is sent.
   - stdin EOF or `session/close` ends the process within the 2 s bound.
7. **Command names.** Every name in `available_commands_update` matches `[A-Za-z0-9][A-Za-z0-9_-]*`, mcremote's `isCommandName`, so it can appear in `remote_commands`. `/skill:<name>` and template commands still run as prompt text (0002-MADR fifth amendment). The phone-visible aliases stay as 0005-MADR lists them.
8. **One session store across modes (F29).**
   - A session started from the phone can be continued in the terminal with `gobble -c` or `--session`, and the reverse.
   - The process that appends to a session file holds an exclusive advisory lock on it: `flock` on Unix and `LockFileEx` on Windows, through `x/sys`.
   - A second writer gets exit 1 and `Error: session <id> is open in another gobble process (pid N)`.
   - A reader, such as `session list` or an export, takes no lock.
9. **The terminal mirrors the phone.**
   - The line session's tool display (D7) matches the phone's card: title, status, collapsed detail.
   - Its context colours match the phone's chip (F27).
   - Its slash completion offers only commands that execute. That is magic-cli-remote 0023's "advertisement is capability" rule, and gobble's D13.

**D20. Companion changes in magic-cli-remote, recorded and not made here.** Each needs a record in that repository first, and the owner put them out of scope for now.
- **Rename 0179's provider to gobble:** `IDGobble`, wire id `"gobble"`, `DefaultBin: "gobble"`, and `_gobble/compact`, `_gobble/usage`, `_gobble/set_session_name`, `_gobble/fork` in place of `_pigo/*` (F22).
- **Implement 0179 D2 and D3** (Spec-reported operations, strict fallback) before gobble is registered. Until then a `KindNative` row falls back to grok vendor calls (0002-MADR fifth amendment).
- **Implement 0179 D5:** session title, `usage_update.cost`, tool `locations` and diff text, and config categories and groups. Then the phone shows what gobble already sends.
- **Implement 0179 D6 and D7:** `ConfigureSession` after `session/load`, model catalog from `configOptions`, and `DangerousModeIDs`.
- **The deferred items in 0179's "More Information":** the `!` gate, a steer path, stdio MCP, `clientInfo` in `initialize`, per-provider environment, and question delivery.
- **Size checks on outbound broadcast and replay** (0181), and dedupe of replayed events by message identity rather than exact text (F25).

### Consequences

- **Good:** the CLI mode's module cost is Kong (no runtime dependencies) and `x/term` (only `x/sys`, already in the graph). That is measurements 3 and 5.
- **Good:** every conflict in F1–F4 has one answer, and the text it replaces is named.
- **Good:** the features the research ranked highest come with it:
  - from goose: stable-prefix streaming, two-stage Ctrl+C, sanitising, bounded tool output, the context bar and editor compose;
  - from Pi: mode resolution, stdout discipline, input composition, the session and model flags, and a streaming machine format.
- **Good:** copying a reply out of the terminal yields valid Markdown, because markers are kept.
- **Neutral:** Pi users lose the multi-letter short flags and Pi's json event names. gobble's stream is ACP's, as 0005-MADR already decided.
- **Neutral:** code blocks are not syntax-highlighted in this mode.
- **Bad:**
  - D1 replaces 0006 D6's "bare session" line;
  - D9 replaces 0006 D10's "text or json only";
  - D10 replaces Pi's json-mode exit 0.

  Each of these reverses a recorded position. The owner must accept them knowingly.
- **Good (2026-10-04):**
  - completion is dynamic on all four shells with no hidden command and no new module (D17);
  - `slog` and the Go 1.27.1 idioms are lint rules seen to fail (D18);
  - gobble's ACP stream survives magic-cli-remote's drops, cuts and replay dedupe by construction (D19);
  - the terminal and the phone show the same thresholds and the same commands.
- **Neutral (2026-10-04):** D18 goes beyond the fleet's lint set, which 0007-MADR aligned to. The difference is recorded, and a fleet-wide adoption is for another record.
- **Bad (2026-10-04):**
  - full integration still waits on magic-cli-remote changes (D20), which are out of scope;
  - until 0179 D2, D3 and D5 land, the phone shows less than gobble sends, and a `KindNative` row can still fall back to a grok vendor call.
- **Bad:** three small pieces of gobble code are not canonical library code: the D8 sanitiser, the D7 styler and the D11 generator. Each is a candidate to swap for go-tui-lib's `safetext`, `stream` or `kongcmd` once those exist Charm-free.
- **Risk: the sanitiser or the context bar is the first thing dropped to save time.** D8 is the one most likely to be cut, because nothing visibly breaks without it until a hostile tool output arrives.

### Confirmation

These are checks for the PLAN, not commands run for this record.

```text
gobble --print "x" </dev/null        stdout: final text only; stderr: diagnostics; no ANSI on stdout
echo hi | gobble ask                 print mode (stdin not a terminal); prompt is "hi\n\nask"
gobble                               on a terminal: line session; never the configure wizard
gobble -p --output-format stream-json …   first line has type "session"; every later line but the last is an ACP session/update params object
gobble completion bash|zsh|fish|powershell   a script on stdout, exit 0; go.mod gains no completion module
NO_COLOR=1 gobble …                  no SGR sequences on stdout or stderr; Windows VT still enabled
GOBBLE_COMPLETE=bash … --session    candidates from the session store, newest first, ':32' directive; nothing on stderr; no file created (D17)
completion, all shells              golden scripts; bash -n, zsh -n, fish --no-execute, PowerShell parser clean; TabExpansion2 on 5.1 and 7 returns typed results (D17)
completion coverage test            fails on a scratch copy with one `complete` tag removed (D17)
golangci-lint                       sloglint, depguard and forbidigo plants fail as in measurement 12; the tree is clean (D18)
acp handshake                       agentInfo.version parses as semver on a dev build; initialize sends no network request (D19)
replay                              session/load emits one chunk per message and one tool_call per call (D19)
session lock                        a second writer of the same session exits 1 with the pid message; a reader succeeds (D19)
conhost paste                       measurement 10's probe passes in CI-equivalent form on Windows: 3-line paste submits once (D6)
archtest                             internal/cli imports no charm.land/, charmbracelet/ or go-tui-lib path; only internal/cli imports kong
sanitiser test                       ESC[2J, OSC 0 and U+202E in tool output are neutralised (seen to fail on an unsanitised copy)
line editor test                     Ctrl+C clears a non-empty line; a 3-line bracketed paste submits once
exit codes                           2 for --fork with --session; 130 for SIGINT; 1 for a turn ending in error, in json and stream-json too
```

## Pros and Cons of the Options

### A — Native line client on Kong, x/term and the standard library (chosen)

- Good, because it is the only option that respects rule 5 and the accepted default mode at once.
- Good, because its module cost is measured: Kong has no runtime requirements, and `x/term` needs only `x/sys`, which is already present.
- Good, because the behaviours it takes are the ones ranked highest in two independent code reads, and each is cited to a line range.
- Bad, because the editor is single-line with continuation and an external editor, not a full multi-line editor with vi mode.
- Bad, because gobble writes its own sanitiser, styler and completion generator.

### B — Allow Charm in `internal/cli`

- Good. The strongest argument: one theme and one renderer (glamour) for both modes, a richer input widget, and go-tui-lib's `safetext` and `stream` usable the day they ship.
- Bad, because a bubbletea input widget is a TUI. The line between the two modes, which 0005-MADR draws, disappears.
- Bad, because it widens rule 5 for every Charm module, not just one package, and makes the default mode carry the TUI's dependency graph.

### C — Follow Pi literally: no line REPL

- Good. The strongest argument: Pi has no line REPL (F8). This is the smallest CLI, and Pi users would find exactly what they know.
- Bad, because it contradicts the accepted default mode, a native terminal CLI (0002, 0005), and 0006 D12's line editor.
- Bad, because a terminal user would then need the TUI for any conversation, which is the heavier mode.

### D — Third-party editor, goldmark and chroma

- Good. The strongest argument: vi mode, true multi-line editing and highlighted code blocks, without writing them.
- Bad, because the candidate editors are stale (`peterh/liner`, 2023), only lightly maintained (`chzyer/readline`), or framework-sized (`reeflective/readline`, `c-bata/go-prompt`). See measurement 6.
- Bad, because `goldmark` and `chroma` would add parsing and highlighting modules to the default mode, where coloured source (F9) already gives clean output.

## More Information

### Evidence index

| Claim | Source |
|---|---|
| goose commit, licence | `gitrepos/goose` `591edd47cf2cfea4957d720c607cf2a4def8673d`; `LICENSE`; `Cargo.toml:14` |
| goose bare command runs configure or a session | `crates/goose-cli/src/cli.rs:2767-2805` |
| goose `run` and its input and output flags; stdin needs `-i -` | `cli.rs:219-331`, `:983-1002` |
| goose stable-prefix buffer | `session/streaming_buffer.rs:225`, `:292`, `:317`, `:328` |
| goose flush before tool events; bat on Markdown source; raw off a terminal | `session/output.rs:385-483`, `:1137-1179`, `:1150-1152` |
| goose sanitising, bidi escaping, title | `session/output.rs:651-698`, `:666-667`, `:1542-1555`, `:1847-1848` |
| goose tool header, bounds, the `/toggle` hint defect, width and path defects | `session/output.rs:589-603`, `:720-761`, `:751`, `:1354-1433`, `:1361`, `:1435-1482` |
| goose context bar, elapsed time, `--stats` | `session/output.rs:1565-1613`; `session/mod.rs:791-802`, `:2052-2125`, `:2633-2642` |
| goose two-stage Ctrl+C and hints; per-turn cancel; interrupt repair | `session/input.rs:49-86`; `session/completion.rs:515-541`; `session/mod.rs:1319-1325`, `:1529-1535`, `:1646-1723`, `:1669-1675` |
| goose external editor | `session/editor.rs:15-229`, `:131-139`, `:142-172` |
| goose EPIPE-tolerant list output | `commands/session.rs:170-176` |
| goose VT enabling depends on NO_COLOR and terminal checks | cargo registry `console-0.16.6/src/windows_term/mod.rs:62-72`, `:133-140` |
| Pi commits, licence, unchanged contract files | upstream `f5d2004`, fork `312184e`; `LICENSE`; normalised tree diff of both clones |
| Pi mode resolution | `src/main.ts:112-123` |
| Pi stdout takeover and output guard | `src/main.ts:129-131`, `:652-655`; `src/core/output-guard.ts:45-108` |
| Pi diagnostics prefixes; stdin read and trim | `src/main.ts:99-105`, `:80-97` |
| Pi `@file` wrapping; `join("")` | `src/cli/file-processor.ts:25-88`; `src/cli/initial-message.ts:40` |
| Pi print output and exits | `src/modes/print-mode.ts:50-66`, `:135-156` |
| Pi json stream shape | `src/modes/print-mode.ts:108-127`; `src/modes/json-event.ts:1-61`; `docs/json.md:15-27` |
| Pi json failed turn exits 0 | `docs/cli-integration.md:46` |
| Pi has no line REPL; built-in slashes only in the TUI | `src/modes/` listing; `src/modes/interactive/interactive-mode.ts:3157-3299` |
| Pi session and model flags; `--provider` requires `--model` | `src/cli/args.ts:71-262`; `src/main.ts:252-536`, `:469-474` |
| Kong v1.16.1: no completion code, no runtime requirements | module cache `kong@v1.16.1` `go.mod`, `*.go` search |
| kong-completion and kongplete shells | GitHub API, README of each |
| `x/term` Terminal: history, completion, paste, Ctrl+C, Windows VT input | module cache `x/term@v0.46.0` `terminal.go:41`, `:70`, `:366`, `:827-870`, `:986`, `:993`; `term_windows.go:30-31` |
| Line-editor module activity | GitHub API, 2026-10-04 |
| Import rules 2, 5, 6 and 8; no Kong rule | `internal/archtest/check.go:200-218`, `:253-275`, `:277-289`, `:306-322` |
| go-tui-lib state and planned packages | `git -C go-tui-lib` `origin/main` `45ead16`, tag `v0.2.0`; its `docs/decisions/0006-MADR-command-registry.md` §1 and A1, `0009-MADR-streaming-content-engine.md` §1, `0006-PLAN`/`0009-PLAN` frontmatter |
| Kong usage rules (Name, Writers, Exit sentinel) | go-tui-lib `0006-MADR-command-registry.md` A1, "Rules" |
| Windows signal split | magic-cli-remote `9778cbc1` `internal/cli/signals_windows.go:7-17` |
| VT helper and its swallowed error | mcp-server-magictools `e7ddbb8` `internal/ui/init_windows.go:14-32` |
| URL opener | prepare-commit-msg `4b5dab4` `internal/ui/open.go:13,39` |
| Configure wizard, device login, catalog | go-llmprovider-sdk `1ca1e7d` `wizard/configure.go:132`, `wizard/text_prompter.go:62`, `llmprovider/auth/oauth_device.go:106`, `llmprovider/catalog/discovery.go:63` |
| selfupdate `cli.Command` maps contradictions to 1 | go-selfupdate-lib `a65e12b` `selfupdate/cli/command.go:39-44` |
| 0157 rolling log has no code | magic-cli-remote `docs/decisions/0157-MADR-*` frontmatter; no `internal/logging/rotate.go` on any branch |
| Earlier gobble positions | 0002-MADR "CLI as ACP client"; 0002-PLAN Phase 2; 0004-PLAN Phase 2 and amendments; 0005-MADR "Clients and modes", "CLI surface", "CLI flags", "Self-update"; 0006-MADR D6–D14 |
| Binary-size effect of each option | **[unverified]**: not measured; the PLAN measures `go build` size before and after |
| ~~Windows Console Host bracketed paste under VT input~~ | ~~**[unverified]**~~; measured 2026-10-04, see the row below |
| Console Host modes, Ctrl+C byte, UTF-8, bursts, bracketed paste, `ReadLine` split, close hang | Measurement 10: scratch probes in a `conhost.exe` window (`ConsoleWindowClass`), Windows 11 25H2 build 26200.9457, conhost 10.0.26100.1 |
| Go 1.25–1.27 API used by D18 | Measurement 11: `$GOROOT/api/go1.25.txt`, `go1.26.txt`, `go1.27.txt`; `go doc log/slog.DiscardHandler` |
| Lint rules enforce slog and output discipline | Measurement 12: scratch module, `golangci-lint config verify` and `run` with planted violations |
| magic-cli-remote 0179 is `pigo`, proposed; D2/D3 unimplemented | `docs/decisions/0179-MADR-pigo-native-acp-provider.md:1-10`, `:250-271`; `git log --follow` (`72588004`); `internal/command/command.go:149` |
| Notification drop, coalescing, tool-update merge | magic-cli-remote `0167-MADR…` D17–D18; `acpagent/session.go:582-603`, `:1263-1305`, `:1427-1437`; `internal/chunkbuf/chunkbuf.go:28-33`, `:84-110`; `0024-MADR-stream-coalescing.md` |
| 400/600-byte tool and permission summaries; diff and terminal reduced | `acpagent/session.go:1668-1705`, `:2113-2118`, `:2556-2609` |
| Config options ungrouped only, no category; modes never dangerous | `acpagent/session.go:1953-1958`, `:1815-1826` |
| Replay dedupe and head loss | `internal/session/manager.go:270-272`, `:369-401`, `:1483`; `acpagent/session.go:1446-1461` |
| Phone context thresholds 75/90; tool tiles; composer uses available commands | `apps/mobile/lib/features/chat/chat_screen.dart:3326-3380`, `:606-645`; `chat_bubble.dart:291-331` |
| Process model, timeouts, spare matching | `0179…:371-384`; `acpagent.go:157-162`, `:791-794`, `:810-817` |
| Version pin parsing | magic-cli-remote `internal/provider/version.go:8-40`; `acpagent/version.go:18-56` |
| Pi has no session-file lock | Pi `f5d2004` `packages/coding-agent/src/core/session-manager.ts` (no lock); locks in `auth-storage.ts`, `settings-manager.ts`, `trust-manager.ts` |
| Completion protocols compared | clap `clap_complete-v4.6.9` `env/mod.rs:215-281`, `env/shells.rs:31-62`, `:336-369`; Cobra `v1.10.2` `completions.go:31-34`, `:56-95`, `:258-298`, script files as cited in F30; posener/complete `v2.1.0`; kongplete `v0.4.0`; kong-completion `v0.0.14` |
| Kong model and trace facts for D17 | Kong `v1.16.1` `model.go:41-60`, `:95-110`, `:258-359`, `:422-433`; `tag.go:358-364`; `negatable.go:1-20`; `context.go:102-116`, `:399-403`, `:569-576` |
| PowerShell encodings and argument passing on this host | `powershell.exe`: `[Console]::OutputEncoding` 65001, `$OutputEncoding` 20127, OEMCP 437; `pwsh` 7.6.6: `$PSNativeCommandArgumentPassing` `Windows` |
| bash `COMP_POINT` units | **[unverified]**: the D17 real-shell test settles it |
| Whether chalk 6 honours `NO_COLOR` in Pi | **[unverified]**: Pi's own source never reads it; not needed for D2 |

### Related records

- [0002-MADR-cli-acp-headless-mcp-v1.md](0002-MADR-cli-acp-headless-mcp-v1.md) and [0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md): the CLI is an ACP client. D1 replaces 0002-PLAN Phase 2's `gobble prompt` with `gobble -p`.
- [0004-PLAN-go-module-architecture.md](0004-PLAN-go-module-architecture.md): Phase 2 builds the D16 first part. D14 lists its corrections.
- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md): print mode, CLI surface and flags. This record implements them for the default command.
- [0006-MADR-goose-cli-port-candidates.md](0006-MADR-goose-cli-port-candidates.md):
  - D7 is made implementable by ~~D11~~ D17 (2026-10-04);
  - D6's "bare session" line is replaced by D1;
  - D10's "text or json only" is replaced by D9;
  - D8, D12 and D13 are refined by D2, D6 and D15.
- [0007-MADR-repository-scaffolding-to-fleet-standard.md](0007-MADR-repository-scaffolding-to-fleet-standard.md): D18 adds linters beyond the fleet set it aligned to. The PLAN records that as a dated note there.
- magic-cli-remote `docs/decisions/0179-MADR-pigo-native-acp-provider.md` (`proposed`): the mcremote half of the contract. D19 is gobble's half, and D20 lists what 0179 must change.
- [0001-REPORT-go-port-feasibility.md](../reports/0001-REPORT-go-port-feasibility.md): it examined Pi at `312184e`. Its env-var name `PI_AGENT_DIR` is `PI_CODING_AGENT_DIR` (`src/config.ts:546`), and its session-backend package row is stale upstream.

### Open questions for the plan

These are execution details. Every decision above is settled.

- **The amendments that must land with this record's PLAN**, as dated entries:
  - 0006-MADR (D1, D9, D11);
  - 0004-PLAN Phase 2 (D1, D14, D16);
  - 0002-PLAN Phase 2 (D1's `gobble -p`);
  - 0002-MADR fifth amendment (D14's 0179 citation);
  - the 0002-PLAN phases named in D16 for D19;
  - 0007-MADR (D18's lint divergence).
- ~~**The Windows Console Host paste probe (D6),** run on conhost and on Windows Terminal, with the burst fallback built only if the probe shows bracketed paste is absent.~~ *(Answered 2026-10-04 by measurement 10 for Console Host. Windows Terminal was not probed; the PLAN repeats measurement 10's probe under `wt.exe`.)*
- **Latency baselines (D18):** `gobble --version` and a `--session` completion request, measured on this host.
- **`go build` size** of `cmd/gobble` before and after Phase 2, recorded as the "lightweight" baseline.
- **`docs/architecture.md:32` still cites go-tui-lib `v0.1.0`;** the current tag is `v0.2.0`. This is a documentation correction for the plan that next touches that file.
