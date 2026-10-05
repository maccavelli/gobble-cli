---
status: completed
date: 2026-10-05
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# PLAN 0008 — The native CLI mode: Kong root, terminal layer, line editor, native completion, renderer, and enforced Go 1.27.1 idioms

Implements [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) decisions D1–D4, D6–D8, D10, D12, D14 and D16–D18, D19 item 9, and D20 (as records). It closes findings F1–F7, F9–F17, F19–F21, F23–F24 (terminal side), F27 and F30.

Four decisions are not built here. Each is handed by a dated amendment (P0) to the plan that owns the agent or ACP wiring:
- D5 (the line session loop) and D9 (print-mode formats) go to 0002-PLAN Phase 2.
- D13 (the slash registry union) goes to 0002-PLAN Phase 6.
- D19 items 1–8 (agent-side ACP behaviour) go to 0002-PLAN Phases 1, 3, 4 and 6.
- D15's configure, login and update reuse stays in 0004-PLAN Phase 2 steps 12–13 and 0005-PLAN F10.

## Goal

Every item below is an observable state when this plan is done:

1. `gobble` builds as a binary from `cmd/gobble`.
   - `gobble version --json` prints `{"name":"gobble","version",…}`.
   - `gobble config path` prints the four role directories.
   - `gobble acp` exits 2 with "not yet implemented".
   - `gobble`, `gobble -p "x"` and `echo x | gobble` resolve the D1 mode, compose D4 input, then exit 2 with `Error: the agent is not available yet (0002-PLAN Phase 2)`.
2. Every D12 flag parses and is validated. Contradictions exit 2, and a flag whose behaviour needs a later plan exits 2 with "not yet available".
3. `gobble completion bash|zsh|fish|powershell` prints a script. With `GOBBLE_COMPLETE` set, the binary answers completion requests in under 150 ms with D17's protocol. Real bash, zsh, fish, pwsh 7 and Windows PowerShell 5.1 complete flags, enum values and paths.
4. The line editor (`internal/cli/editor`) passes its tests: two-stage Ctrl+C, a coalesced bracketed paste, history, continuation and `/edit`. Measurement 10's conhost probe, rebuilt as a live test, passes on this host.
5. `internal/cli/render` turns recorded assistant-text and tool-event fixtures into the D7 output: a stable-prefix stream, Markdown coloured in place, aligned tables, bounded tool output, and a 75/90 status line. Colour off yields the raw text.
6. `.golangci.yml` carries D18's rules, and `make preflight` is green on Windows (Git Bash) and Linux (WSL). Every new rule and gate has been seen to fail on a plant.
7. archtest rule 9 (Kong only in `internal/cli`) exists and has been seen to fail.
8. The dated amendments of P0 are in 0002-MADR, 0002-PLAN, 0004-PLAN, 0006-MADR and 0007-MADR. `docs/architecture.md` cites go-tui-lib `v0.2.0`.

## Scope

### In scope (the only files any phase may touch)

**Records**
- `docs/decisions/0008-PLAN-native-cli-mode.md` (this file)
- `docs/decisions/0008-MADR-native-cli-mode.md`: status and the Observed section only ~~*(2026-10-04: also the dated D18 amendment that records the P1 forbidigo deviation)*~~ *(Withdrawn 2026-10-04 with that deviation; the strike-through it leaves in D18 is the only MADR edit.)* *(2026-10-05: and the dated D15 note of the owner's buildinfo decision.)*
- `docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md`, `docs/decisions/0002-PLAN-cli-acp-headless-mcp-v1.md`, `docs/decisions/0004-PLAN-go-module-architecture.md`, `docs/decisions/0006-MADR-goose-cli-port-candidates.md`, `docs/decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md`: dated amendments only
- `docs/README.md`: index rows and statuses
- `docs/architecture.md`: the go-tui-lib version line only

**Build and gates**
- `.golangci.yml`
- `Makefile`
- `scripts/verify_build_metadata.py` *(added 2026-10-05, P4 deviation)*
- `go.mod`, `go.sum`
- `internal/archtest/check.go`, `internal/archtest/check_test.go`

**Process edge**
- `cmd/gobble/main.go` (new)
- `cmd/gobble/doc.go`: the comment only

**`internal/cli`** (the package comment in `doc.go`, plus new files)
- `doc.go`, `main.go`, `root.go`, `chat.go`, `mode.go`, `flags.go`, `input.go`, `exit.go`, `signal.go`, `output.go`, `version.go`, `configpath.go`, `acp.go`, `completion.go`
- `brokenpipe_windows.go`, `brokenpipe_other.go` *(added 2026-10-05, P4 deviation)*
- the tests beside each

**`internal/cli/term`** (new)
- `doc.go`, `caps.go`, `style.go`, `sanitize.go`, `console_windows.go`, `console_other.go`, `signals_unix.go`, `signals_windows.go`, and tests

**`internal/cli/editor`** (new)
- `doc.go`, `editor.go`, `reader.go`, `keys.go`, `paste.go`, `history.go`, `external.go`, and tests
- `conhost_windows_test.go`, build tag `conhost`

**`internal/cli/complete`** (new)
- `doc.go`, `candidate.go`, `walk.go`, `lex.go`, `protocol.go`, `sources.go`, `scripts.go`
- `scripts/bash.tmpl`, `scripts/zsh.tmpl`, `scripts/fish.tmpl`, `scripts/powershell.tmpl`
- `testdata/*.golden`, and tests
- `shells_test.go`, build tag `shells`

**`internal/cli/render`** (new)
- `doc.go`, `mdstream.go`, `mdstyle.go`, `table.go`, `tool.go`, `status.go`, `spinner.go`
- `testdata/*`, and tests

### Out of scope

- **Agent and ACP wiring.** `acpclient`, `acpserver`, the agent loop, sessions and the session lock. These are D5, D9, D13 and D19 items 1–8, handed to 0002-PLAN in P0.
- **`internal/buildinfo`, `internal/appdirs`, `internal/logging`.** These are 0004-PLAN Phase 2 steps 2–4, and they are **preconditions** of P4 (see Dependency and delivery order).
- **`session list|resume|rename`, `configure`, `info --check`.** These stay as 0004-PLAN Phase 2 steps 10, 12 and 13. They add commands to the root P4 creates.
- **`ci.yml`.** 0004-PLAN Phase 4. This plan's gates run through `make preflight` and the two new make targets.
- **The TUI** (`internal/tui`) and anything under go-tui-lib.
- **Every sibling repository, magic-cli-remote included** (D20 is recorded, not executed).

## Stability rule

- **Branch and commits.** Work stays on the current branch, which is `main`. The agent creates no branch, stages each phase's files with `git add`, and never runs `git commit` or `git push`. The owner commits and pushes (owner, 2026-10-04).
- **Every phase ends with these, exit statuses captured before any filter:**
  1. `LOG=$SCRATCH/0008-pN-win.log; make preflight > "$LOG" 2>&1; GATE=$?` in Git Bash. Expected: `GATE=0`, last line `preflight passed`.
  2. The same preflight in WSL Ubuntu 24.04 on a copy of the working tree, with the log written inside Linux (0007-PLAN execution record, prediction error 4):

     ```text
     MSYS_NO_PATHCONV=1 wsl.exe -d Ubuntu-24.04 --exec bash -lc 'rm -rf ~/scratch/gobble-0008 && cp -a /mnt/c/<path-to>/gobble-cli ~/scratch/gobble-0008 && cd ~/scratch/gobble-0008 && make preflight > ~/scratch/0008-preflight.log 2>&1; rc=$?; cat ~/scratch/0008-preflight.log; exit $rc'
     ```

     Expected: exit 0.
  3. `git diff --check` and `git diff --cached --check main` are both clean.
  4. `git status --porcelain` lists only In-scope files.
- **Staged Go files** pass `make pre-add-check FILES="…"` first.
- **On a failure** this plan does not fix, stop and prompt: give the evidence, give resolutions and no workaround, and say what doing nothing would cost. No gate is weakened.

## Cross-cutting contracts

- **C1. No Charm, no go-tui-lib, no ACP SDK under `internal/cli/...`** (rules 2 and 5). Only `internal/cli/...` imports Kong (new rule 9).
- **C2. New module requirements are exactly:**
  - `github.com/alecthomas/kong v1.16.1`;
  - `golang.org/x/term v0.46.0`;
  - `golang.org/x/text v0.42.0`, for `width` only (P6), the Go team's East Asian Width tables, rather than a hand-written table; magic-cli-remote already requires the same version;
  - `golang.org/x/sys` (already present at v0.48.0), promoted to direct.

  `go mod tidy -diff` is clean after every phase.
- **C3. Output discipline (D3, D18).** No package writes to `os.Stdout` or `os.Stderr` except through the writers `cli.Main` receives. `cmd/gobble/main.go` is the only file that names them. forbidigo enforces this.
  ~~*(2026-10-04: forbidigo runs with `analyze-types: false`, because the type-aware mode missed method calls on the streams. See the P1 deviation.)*~~ *(Withdrawn 2026-10-04: see the P1 note.)*
- **C4. Every gate is seen to fail before it is trusted.** Plants go in scratch copies, never in the tree. The failure text is quoted in the Execution record.
- **C5. No test skips for a missing tool.** The `shells` and `conhost` suites fail when a shell or a console is absent.

  **This is the contract most at risk.** zsh and fish are not installed on this host's Windows or WSL side today (measured 2026-10-04), and a `t.Skip` would make P5 green at once. The precondition in P5 is to install them, not to skip.
  *(2026-10-04: fish has been installed in WSL and on a Linux host, and zsh is on the macOS host. P5 runs each shell where it exists, and a shell missing from its assigned host fails the suite.)*
- **C6. Completion mode is read-only and network-free** (D17). A test runs it with `HOME`, `XDG_*` and `GOBBLE_HOME` pointing at an empty temp directory, and asserts that the directory is still empty afterwards.
- **C7. No identifiers** in any added file or record. Paths in this plan are written `<path-to>`.

## Dependency and delivery order

```text
P0 records ─┬─ P1 lint + archtest ─ P2 terminal layer ─ P3 line editor ─┐
            │                                                           ├─ P5 completion ─ P6 renderer ─ P7 close-out
            └── 0004-PLAN Phase 2 steps 2, 3, 4 (buildinfo, appdirs, logging) ── P4 Kong root ┘
```

- P1–P3 can run now.
- **P4 is blocked until 0004-PLAN Phase 2 steps 2–4 have run.** `version` needs `internal/buildinfo`, `config path` needs `internal/appdirs`, and `cli.Main` builds its logger with `internal/logging`.
- P5 needs P4's Kong root. P6 needs P2's term layer.
- 0004-PLAN Phase 2 steps 10, 12 and 13 run after P4.

## Implementation Steps

`$SCRATCH` is the session scratchpad. Helper scripts are written there with the Write tool and run by path, and none is staged.

### P0 — Records: amendments, statuses, one doc fix (D1, D9, D14, D16, D17, D18, D19, D20; closes F1–F4 on paper, F22)

1. **`docs/decisions/0008-MADR-native-cli-mode.md`:** set `status: accepted`. This step runs only once the owner has approved this plan.
2. **`docs/decisions/0006-MADR-goose-cli-port-candidates.md`:** append this, with the execution date:

   ```markdown

   ## Amendment — YYYY-MM-DD: superseded in part by 0008-MADR

   [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) changes three positions of this record. The text above is kept as it was decided.

   * **D6, "bare session".** Replaced by 0008-MADR D1. Bare `gobble` on a terminal opens the line session and never runs the configure wizard. The goose coupling D6 measured (configure when no config exists) stays rejected.
   * **D10, "text or json only … Do not add `stream-json`".** Replaced by 0008-MADR D9. Print mode has `text`, `json` and `stream-json`, and `stream-json` speaks the ACP vocabulary, as 0005-MADR specifies.
   * **D7, "Kong completion".** Kong has no completion generator (0008-MADR F30, measurement 3). D7 is implemented by 0008-MADR D17: native dynamic completion for bash, zsh, fish and PowerShell, through an environment-variable mode and no hidden command.
   * D8, D12 and D13 are refined by 0008-MADR D2, D6 and D15. In particular, Console Host sends bracketed paste, so D12's Windows paste needs no burst detection (0008-MADR F19).
   ```
3. **`docs/decisions/0004-PLAN-go-module-architecture.md`:** append this, with the execution date:

   ```markdown

   **YYYY-MM-DD — Phase 2 split with 0008-PLAN (0008-MADR D14, D16).**
   [0008-PLAN-native-cli-mode.md](0008-PLAN-native-cli-mode.md) executes Phase 2 steps 1, 5 (the root and the `version`, `completion`, `config path`, `acp`-stub and default commands), 6, 7, 8, 9 and 11. Steps 2, 3 and 4 run here first, because they are 0008-PLAN P4's preconditions. Steps 10, 12 and 13 run here after 0008-PLAN P4 has created the root.

   Changes to the steps as written:

   * **Step 1.** `cmd/gobble/main.go` calls `cli.Main(context.Background(), …)`. `cli.Main` owns signal handling: `signal.Notify` with the per-OS list (Interrupt, SIGTERM and SIGHUP on Unix; Interrupt only on Windows), and `context.WithCancelCause` with a typed cause, so the exit code can be 128 plus the signal (0008-MADR D2, D10). SIGTERM is not registered on Windows, which never delivers it.
   * **Step 4.** The rolling file sink is written from magic-cli-remote 0157-MADR's design: 10 MiB per file, 5 backups, 28 days. That record has no code to copy (0008-MADR F17).
   * **Step 5.** The one-shot and session commands are one default command, `chat`, with Pi's mode resolution and `-p/--print` (0008-MADR D1). Kong is used with `kong.Name`, `kong.Writers` and an `Exit` that panics a sentinel. `kong.Parse` and `FatalIfErrorf` are never called (0008-MADR D14).
   * **Step 6.** The exit table adds 128 plus the signal: 130, 129 and 143 (0008-MADR D10).
   * **Step 11.** The editor is `golang.org/x/term` behind gobble's adapter (0008-MADR D6).
   * **Phase 4.** `make verify-build-metadata` is implemented by 0008-PLAN P4, because creating `cmd/gobble/main.go` trips the Phase 0 placeholder (`Makefile`, `verify-build-metadata`). Phase 4 keeps the CI wiring.
   * **Phase 0 step 7.** The lint set gains 0008-MADR D18's rules (sloglint, depguard, forbidigo, modernize, intrange, copyloopvar, usestdlibvars, perfsprint, usetesting, noctx, contextcheck, fatcontext).
   * **archtest rule 9.** Only `internal/cli/...` may import Kong. Added by 0008-PLAN P1.
   ```
4. **`docs/decisions/0002-PLAN-cli-acp-headless-mcp-v1.md`:** append under its `## Amendments` heading (create the heading at the end of the file if it is absent), with the execution date:

   ```markdown

   **YYYY-MM-DD — native CLI mode and magic-cli-remote integration (0008-MADR).**

   * **Phase 2, step 2.** `gobble prompt "…"` is replaced by `gobble -p "…"` on the default `chat` command (0008-MADR D1). This phase also wires:
     * the line session (D5);
     * the renderer of `internal/cli/render` (D7);
     * the print formats `text`, `json` and `stream-json` (D9);
     * Ctrl+C during a turn as `session/cancel`.
   * **Phase 1.** `initialize` follows 0008-MADR D19 item 1:
     * `agentInfo.version` is parseable semver on every build;
     * capabilities are as listed there;
     * no network call before `initialize` returns (item 6).
   * **Phase 3.** Agent output follows D19 items 2 and 3:
     * snapshot-shaped updates;
     * chunks coalesced at 50 ms or 4 KiB;
     * terminal tool statuses sent alone;
     * tool titles of at most 60 runes, with a self-contained summary of at most 400 bytes first.
   * **Phase 4.** Sessions follow D19 items 5 and 8:
     * compact replay on `session/load`;
     * an exclusive per-session writer lock (`flock` / `LockFileEx`), with a second writer failing with exit 1 and the owning pid;
     * readers take no lock.
   * **Phase 6.**
     * The line session's slash registry is the agent's `available_commands_update` united with `/help`, `/exit`, `/quit`, `/edit` and `/new`. An unknown name is a local error (D13).
     * Advertised names match `[A-Za-z0-9][A-Za-z0-9_-]*` (D19 item 7).
     * `model` and `thought_level` are ungrouped select config options with a category (D19 item 4).
     * The permission prompt is D16's one-line prompt.
   ```
5. **`docs/decisions/0002-MADR-cli-acp-headless-mcp-v1.md`:** append, with the execution date:

   ```markdown

   ### Amendment (YYYY-MM-DD): the magic-cli-remote record is named `pigo` there

   The fifth amendment says the magic-cli-remote half of the contract is `0179-MADR-gobble-native-acp-provider.md`, with `provider.IDGobble`. At magic-cli-remote `9778cbc1` the file is `docs/decisions/0179-MADR-pigo-native-acp-provider.md` (`status: proposed`), created as `pigo` in `72588004`. It names `provider.IDPigo`, wire id `"pigo"`, `DefaultBin: "pigo"` and `_pigo/*` methods, and "gobble" occurs nowhere in that repository (0008-MADR F22, measurement 13). The other facts of the fifth amendment hold at `9778cbc1`. Renaming 0179 to gobble, with `_gobble/*` methods, is a companion change in magic-cli-remote (0008-MADR D20). The rename is out of scope here.
   ```
6. **`docs/decisions/0007-MADR-repository-scaffolding-to-fleet-standard.md`:** append, with the execution date:

   ```markdown

   ## Amendment (YYYY-MM-DD): lint set beyond the fleet's

   [0008-MADR-native-cli-mode.md](0008-MADR-native-cli-mode.md) D18 adds sloglint, depguard, forbidigo, modernize, intrange, copyloopvar, usestdlibvars, perfsprint, usetesting, noctx, contextcheck and fatcontext to `.golangci.yml`. This record's D7 aligned the file to mcplib's set. The additions are a deliberate divergence. Proposing them fleet-wide is a separate record.
   ```
7. **`docs/architecture.md`:** in the sibling-sources table, the go-tui-lib row cites `v0.1.0`. Add the sentence `Newest tag at YYYY-MM-DD: v0.2.0 (0224d0e).` to the row's "Current observed surface" cell, and leave the row's other text unchanged.
8. **`docs/README.md`:**
   - Add the 0008 PLAN row: `| 0008 | PLAN | [Implement the native CLI mode](decisions/0008-PLAN-native-cli-mode.md) | in progress |`.
   - Set the 0008 MADR row to `accepted`.
   - Append `amended YYYY-MM-DD (0008)` to the status cells of 0002 MADR and PLAN, 0004 PLAN, 0006 MADR and 0007 MADR.

**Verification:**
- `make check-records` exits 0.
- `make markdownlint` exits 0.
- `git diff --cached --stat` lists only the eight record files.
- The struck or superseded text in each amended record is still present: `grep -c 'bare session' docs/decisions/0006-MADR-goose-cli-port-candidates.md` is unchanged from before.

### P1 — Lint enforcement and archtest rule 9 (D18, D14; closes F21)

1. **`.golangci.yml`.** Make the three edits measured on 2026-10-04 by `$SCRATCH/p1_lint_probe.py`, whose anchors assert exactly one match each:
   - **Enable list:** after `    - whitespace`, insert in this order: `contextcheck`, `copyloopvar`, `depguard`, `fatcontext`, `forbidigo`, `intrange`, `modernize`, `noctx`, `perfsprint`, `sloglint`, `usestdlibvars`, `usetesting`.
   - **Settings:** before `    errcheck:`, insert the `depguard.rules.logging`, `forbidigo` and `sloglint` blocks of 0008-MADR measurement 12. Each `desc` and `msg` ends `(0008-MADR D18)`.
     ~~*(Deviation 2026-10-04, owner's decision: with `analyze-types: true`, forbidigo does not report a method call on the stream, so `os.Stderr.WriteString("x")` and `os.Stdout.Write(nil)` passed lint on a scratch copy, while `fmt.Fprintln(os.Stderr, …)`, `w := os.Stderr` and `return os.Stderr` failed. Widening the pattern to `os.Stderr.X` did not change that. The forbidigo block is written with `analyze-types: false`, which reported all six forms, and the clean tree stayed at `0 issues.` Literal matching means an import alias such as `o "os"` is not caught, and a local identifier named `log` is. 0008-MADR D18 is amended to match.)*~~ *(Withdrawn 2026-10-04: a misdiagnosis. golangci-lint reports one issue per line by default (`uniq-by-line`), and the plants wrote `_, _ = os.Stderr.WriteString(…)`, which errcheck reports under `check-blank`, so that report hid forbidigo's. With forbidigo alone and `--uniq-by-line=false`, `analyze-types: true` reports all six forms, an error-checked `os.Stderr.WriteString` included. The measured configuration stands, with `analyze-types: true`.)*
   - **Exclusions:** after the existing `path: _test\.go` rule, add two rules: `forbidigo` for `path: _test\.go`, because example tests print by design, and `forbidigo` for `path: cmd/gobble/main\.go`.
2. **`internal/archtest/check.go`.**
   - Change `func loadGraph(dir string)` to `func loadGraph(ctx context.Context, dir string)` and `exec.Command(` to `exec.CommandContext(ctx, `. These are the only two findings the new set reports on today's tree, both `noctx` (measured).
   - Add rule 9:

     ```go
     func checkRule9(pkgs []modPkg) []Violation {
         var vs []Violation
         for _, e := range edges(pkgs) {
             rel, ok := relOf(e.from)
             if !ok || !pathUnder(e.to, "github.com/alecthomas/kong") {
                 continue
             }
             if !under(rel, "internal/cli") {
                 vs = add(vs, 9, e)
             }
         }
         return vs
     }
     ```

     Add `case 9: return checkRule9(pkgs)` to `checkRule`, and include 9 in the loop that runs every rule.
3. **`internal/archtest/check_test.go`.**
   - In `loadReal`, use `exec.CommandContext(t.Context(), …)` and `loadGraph(t.Context(), …)`.
   - Add three table cases to `TestImportRules`:
     - `{name: "rule 9 module", rule: 9}`;
     - `{name: "rule 9 agent kong", rule: 9, want: "alecthomas/kong", pkgs: …agent imports github.com/alecthomas/kong}`;
     - `{name: "rule 9 cli may import kong", rule: 9, pkgs: …internal/cli/complete imports github.com/alecthomas/kong}`.

**Verification:**
- `golangci-lint config verify -c .golangci.yml` exits 0.
- For `GOOS=linux`, `darwin` and `windows` with `CGO_ENABLED=0`, `golangci-lint run ./...` reports `0 issues.`.
- `go test ./internal/archtest/...` passes.
- **Negative tests on a scratch copy:**
  - measurement 12's planted file under `hook/zz_planted.go` fails with each of the 13 messages listed there;
  - a scratch `agent/zz.go` importing Kong fails archtest with `rule 9: …/agent imports github.com/alecthomas/kong`.
- Then the Stability rule.

### P2 — Terminal layer (D2, D8; closes F5, F10, F17 signal part, F20 part)

1. **`go.mod`:** `go get golang.org/x/term@v0.46.0`, then `go mod tidy`. Expected: `golang.org/x/term v0.46.0` and `golang.org/x/sys v0.48.0` in the direct `require` block.
2. **`internal/cli/term/caps.go`**, standard library plus `x/term`:

   ```go
   type Stream struct{ TTY, Color bool; Width int }
   type Caps struct{ Out, Err Stream; In bool; Unicode bool }
   func Detect(env func(string) string, in, out, err *os.File) Caps
   ```

   - `TTY` comes from `term.IsTerminal`.
   - `Color` is `TTY && env("NO_COLOR") == "" && env("TERM") != "dumb"`, or `env("CLICOLOR_FORCE") == "1"`.
   - `Width` comes from `term.GetSize`, then `COLUMNS`, then 80.
   - `Unicode` is false when `TERM=dumb`, or when no `LC_ALL`/`LC_CTYPE`/`LANG` contains `UTF-8`/`utf8` on Unix. On Windows it is true.
   - `env` is injected, so tests need no process environment.
3. **`internal/cli/term/style.go`:**
   - `type Style uint16` with bits `Bold`, `Dim`, `Italic`, `Underline` and the eight foreground colours.
   - `func (s Style) Wrap(text string, on bool) string` emits SGR and `ESC[0m` only when `on`.
   - `type Glyphs struct{ Bullet, Arrow, Check, Cross, Ellipsis, Bar, BarEmpty string }`, with `Unicode` (`•`, `▸`, `✓`, `✗`, `…`, `━`, `╌`) and `ASCII` (`*`, `>`, `+`, `x`, `...`, `#`, `-`).
4. **`internal/cli/term/sanitize.go`:**
   - `func Sanitize(s string) string` removes:
     - ESC-introduced sequences: CSI, OSC up to BEL or ST, and DCS, SOS, PM and APC up to ST;
     - C0 controls except `\t` and `\n`;
     - DEL;
     - C1 controls U+0080–U+009F.
   - `func EscapeBidi(s string) string` rewrites U+061C, U+200E, U+200F, U+202A–U+202E and U+2066–U+2069 as `\uXXXX`.
   - Both are table-tested: `ESC[2J`, `ESC]0;title BEL`, a C1 CSI (U+009B), U+202E, and a clean string that passes unchanged.
5. **`internal/cli/term/console_windows.go`** (`//go:build windows`), copied in shape from mcp-server-magictools `internal/ui/init_windows.go` with the error returned and the modes restored:

   ```go
   // PrepareConsole enables VT processing on stdout and stderr when they are
   // consoles, and returns a func that restores the saved modes. A handle
   // that is not a console is skipped, not an error. Provenance:
   // mcp-server-magictools internal/ui/init_windows.go (Apache-2.0).
   func PrepareConsole(out, err *os.File) (restore func(), _ error)
   ```

   It sets `ENABLE_VIRTUAL_TERMINAL_PROCESSING|ENABLE_PROCESSED_OUTPUT` for each stream whose `GetConsoleMode` succeeds, collects `SetConsoleMode` errors with `errors.Join`, and restores in reverse. **`console_other.go`** (`//go:build !windows`) returns a no-op.
6. **`internal/cli/term/signals_unix.go`** returns `[]os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGHUP}`. **`signals_windows.go`** returns `[]os.Signal{os.Interrupt}`, with magic-cli-remote's comment on Windows and SIGTERM as provenance.
7. **`internal/cli/term/doc.go`:** a package comment with `Stability: internal`.

**Verification:**
- `go test ./internal/cli/term/...` passes on Windows and in WSL.
- The sanitiser test is seen to fail on a scratch copy whose `Sanitize` returns its input.
- A Windows test runs `PrepareConsole` on a pipe and gets `nil`. The live console check is P3's `conhost` suite.
- Then the Stability rule.

### P3 — Line editor (D6; closes F7, F14, F19, F20)

1. **`internal/cli/editor/reader.go`:**
   - `type Reader struct` owns stdin. `Start(ctx, in io.Reader)` launches **one** goroutine, through `sync.WaitGroup.Go`, that reads 512-byte chunks into a channel for the life of the process.
   - `Read(p)` serves `Terminal` from that channel.
   - There is no `Close` of the underlying file anywhere in the package. A test greps the package's non-test files for `.Close()` on the stdin argument and fails if one appears (F20).
2. **`internal/cli/editor/keys.go`:** the reader wrapper maps byte `0x03` outside a paste to an editor event, never to `Terminal`:
   - ~~a non-empty line → clear the line (send `0x15`, Ctrl+U, to `Terminal`);~~
     *(Deviation 2026-10-05, owner's decision. As written this step cannot be built. `x/term` keeps the line private, so a byte-stream wrapper cannot tell an empty line from a non-empty one. Its Ctrl+U also erases only left of the cursor (`terminal.go:612`, `eraseNPreviousChars(t.pos)`). Instead, the wrapper rewrites `0x03` to a private key that reaches `AutoCompleteCallback`, which sees the whole line. A non-empty line is replaced by `""`. On an empty line the editor prints the notice and arms the window. A second press inside the window makes the wrapper's next read yield Ctrl+D on the empty line, so `ReadLine` returns `io.EOF` and `ReadPrompt` returns `ErrExit`. The observable behaviour, and the negative test below, are unchanged. 0008-MADR D6 states the behaviour, not the mechanism, so no MADR amendment.)*
   - a non-empty line → the callback replaces it with `""`;
   - an empty line → print `Press Ctrl+C again to exit` on the error writer and arm a 2 s window;
   - a second `0x03` inside the window → `ErrExit`.

   The window is tested with `testing/synctest`.
3. **`internal/cli/editor/paste.go`:**
   - `SetBracketedPasteMode(true)` on start.
   - A `ReadLine` result carrying `term.ErrPasteIndicator` is appended to `pending`, and `… N pasted lines` is printed on the error writer.
   - The next plain Enter returns `strings.Join(append(pending, line), "\n")` as one submission.
   - Tests feed `ESC[200~line1\rline2\rline3ESC[201~` followed by `\r` and assert one submission, `"line1\nline2\nline3"`. That is measurement 10's exact byte stream.
4. **`internal/cli/editor/editor.go`:**

   ```go
   type Editor struct{ /* Terminal, Reader, history, completer */ }
   func New(in io.Reader, out, errw io.Writer, opts Options) *Editor
   type Options struct{ Prompt, Continue string; History History; Complete func(line string, pos int) (string, int, bool) }
   func (e *Editor) ReadPrompt(ctx context.Context) (string, error)
   ```

   - `Prompt` is `"> "` and `Continue` is `"… "`, or `"... "` with ASCII glyphs.
   - A line ending in `\` continues.
   - The empty line is ignored.
   - `/exit`, `/quit` and Ctrl+D on an empty line return `io.EOF`.
   - `AutoCompleteCallback` is wired to `Options.Complete`.
   - *(Added 2026-10-05, owner's decision: no phase put the console into raw mode or turned bracketed paste off at exit.)* `func Raw(f *os.File) (restore func() error, err error)` wraps `x/term` `MakeRaw`/`Restore`, and `(*Editor).Restore()` writes `ESC[?2004l`. It is deliberately not named `Close`, so the no-close guard of step 1 stays meaningful. The conhost suite uses both, and 0002-PLAN Phase 2's line session calls them.
5. **`internal/cli/editor/history.go`:**
   - Implements `term.History` over a file: one entry per line, `\n` written as `\\n` and `\` as `\\\\`.
   - Capped at 1,000 entries, file mode 0600.
   - Written by atomic replace after every add: a temp file in the same directory, then `os.Rename`.
   - The path is passed in, `<state>/history`; P4 supplies it from `internal/appdirs`.
6. **`internal/cli/editor/external.go`:**
   - ~~`func Compose(ctx, env func(string) string, initial string, run func(*exec.Cmd) error) (string, error)`.~~
     *(Deviation 2026-10-05, owner's decision: the signature had no directory, so the state-directory rule below could not be met.)* `func Compose(ctx, env func(string) string, dir, initial string, run func(*exec.Cmd) error) (string, error)`. The file is `os.CreateTemp(dir, "gobble-prompt-*.md")`, an unpredictable name.
   - The editor command comes from `GOBBLE_EDITOR`, `VISUAL` or `EDITOR`, falling back to `notepad` on Windows and `vi` elsewhere.
   - The command is split by goose's quote-aware rule: split on whitespace unless a quote is present, and keep backslashes.
   - It runs through `exec.CommandContext` with no shell, on a temp file in the state directory that is removed afterwards.
7. **`internal/cli/editor/conhost_windows_test.go`** (`//go:build windows && conhost`): rebuilds measurement 10 as a test.
   - It starts the test binary itself in a new `conhost.exe` window, with the clipboard saved and restored.
   - It triggers Paste with `WM_COMMAND 0xFFF1` to the console window, which the child reports in a ready file.
   - It asserts that the child's `Editor` returned one submission, `line1\nline2\nline3`, and that an injected Ctrl+C on a non-empty line cleared it.
   - `make probe-conhost` runs it.

**Verification:**
- `go test ./internal/cli/editor/...` passes, `-race` included, in WSL.
- `make probe-conhost` passes on this host and quotes the console window class.
- **Negative tests on scratch copies:**
  - with coalescing removed, the paste test reports three submissions;
  - with the Ctrl+C mapping removed, `ReadPrompt` returns `io.EOF` on a non-empty line;
  - the no-close grep fails on a copy that closes stdin.
- Then the Stability rule.

### P4 — Kong root and process edge (D1, D3, D4, D10, D12, D14; closes F1–F4, F15–F17)

**Precondition:** 0004-PLAN Phase 2 steps 2, 3 and 4 are done (`internal/buildinfo`, `internal/appdirs`, `internal/logging`). If they are not, P4 stops and reports.
*(Met 2026-10-05: commit `55f35ba`.)*

**Deviations, 2026-10-05, the owner's decisions before P4 ran:**

1. **Broken pipe on Windows.** Step 4 checks `syscall.EPIPE`, but Windows reports a broken stdout pipe as `ERROR_BROKEN_PIPE` or `ERROR_NO_DATA`. The check moves to `isBrokenPipe(err) bool` in `brokenpipe_windows.go` and `brokenpipe_other.go`, both added to the scope, so D3's exit 0 holds on every OS.
2. **`verify-build-metadata` in Python.** Step 13's check builds twice, parses `version --json`, and compares it with `git rev-parse HEAD` and the commit time. That logic goes in `scripts/verify_build_metadata.py` (standard library only, like `check_records.py`), added to the scope. The Makefile target calls it.
3. **`completion` before P5.** Step 8's grammar has `Completion`, whose command is P5's. P4's `completion.go` holds a stub: `gobble completion <bash|zsh|fish|powershell>` parses its argument and exits 2 with `Error: gobble completion is not yet available (0008-PLAN P5)`. This is D12's "never silently ignored" rule. P5 replaces the stub.

**Choices made within the steps' wording:**

- **The logger is opened lazily,** on a command's first log call. `version`, `config path`, `--help` and the stubs never create the state directory or the log file; `chat` and `acp` open it when they log. `acp` gets no stderr handler.
- **"Not yet available" names the owning plan for each flag:**
  - `--output-format`, `--show-thinking`, `--stats`, `--verbose` and `--cwd`: 0002-PLAN Phase 2;
  - `-t/--tools`, `--exclude-tools` and `--no-tools`: 0002-PLAN Phase 3;
  - `-c/--continue`, `--session`, `--session-id`, `--fork`, `--no-session`, `--session-dir` and `-n/--name`: 0002-PLAN Phase 4;
  - `--system-prompt`, `--append-system-prompt` and `--no-context-files`: 0005-PLAN F3;
  - `--provider`, `--model`, `--thinking`, `--api-key` and `--offline`: 0005-PLAN F4;
  - `--approve/--no-approve` and `--yes`: 0005-PLAN F6.

  `-p/--print` and `--file` work in P4: mode resolution and input composition. `--log-level` is a root flag. Contradictions are checked first, so `--fork` with `--session` is still the D12 usage error, not "not yet available".

1. **`go.mod`:** `go get github.com/alecthomas/kong@v1.16.1`, then `go mod tidy`.
2. **`internal/cli/exit.go`:**
   - Constants `ExitOK = 0`, `ExitFailure = 1`, `ExitUsage = 2`, `ExitAuth = 3`, `ExitCancelled = 4`, `ExitUpdateAvailable = 10`.
   - `func exitForSignal(s os.Signal) int` returns 130 for Interrupt, 129 for SIGHUP and 143 for SIGTERM.
   - `type exitPanic struct{ code int }` is the Kong `Exit` sentinel.
3. **`internal/cli/signal.go`:**
   - `func watchSignals(parent context.Context, sigs []os.Signal, notify func(chan<- os.Signal, ...os.Signal)) (ctx context.Context, stop func())` uses `notify` (`signal.Notify` in production) on a buffered channel and `context.WithCancelCause(parent)`, and cancels with `signalCause{sig}` on the first signal.
   - `func signalExit(ctx context.Context) (int, bool)` uses `errors.AsType[signalCause](context.Cause(ctx))`.
4. **`internal/cli/output.go`:**
   - `type Output struct{ out, err io.Writer; caps term.Caps }`, with `Errorf` and `Warnf` writing `Error: …` or `Warning: …` to `err`.
   - `Result(p []byte)` writes to `out`. A `syscall.EPIPE` error from it makes `Main` return 0 quietly.
5. **`internal/cli/flags.go`:** the D12 shared flag set as one embedded struct `SharedFlags`.
   - Every value-taking field carries `enum:"…"` or `complete:"<source>"`; `complete:"none"` for `--api-key`, `--session-id` and `--name`.
   - Long forms only, apart from `-p`, `-c`, `-n` and `-t`.
   - `validate()` returns usage errors, and so exit 2:
     - `--provider` without `--model`;
     - `--fork` with `--session`, `-c` or `--no-session`;
     - `--session-id` with `--session` or `-c`;
     - `--file` together with piped stdin or an `@path` word. 0006 D10's exclusivity applies to the explicit `--file` form (0008-MADR D4).
   - It returns "not yet available" (exit 2) for every flag whose behaviour needs 0002 or 0005, never ignoring one silently.
6. **`internal/cli/mode.go`:** `func resolveMode(print, stdinTTY, stdoutTTY bool) Mode` returns `ModePrint` or `ModeLine`. It is table-tested over all eight combinations.
7. **`internal/cli/input.go`:** `func composeInput(ctx, stdin io.Reader, stdinTTY bool, words []string, readFile func(string) ([]byte, error)) (Prompt, error)`.
   - Stdin is read whole and trimmed.
   - `@path` words are wrapped as `<file name="ABS">\n…\n</file>`. Images are detected by extension and content sniffing and become an image part.
   - The other words are joined with single spaces.
   - The three parts are joined with `"\n\n"`.
   - A missing `@path` is exit 2.
   - Tested: stdin `hi` with words `ask` gives `hi\n\nask` (the Pi `join("")` defect fixed).
8. **`internal/cli/root.go`:** the grammar.

   ```go
   type Root struct {
       Chat       ChatCmd       `cmd:"" default:"withargs" help:"Start a session, or run one prompt with -p or piped input."`
       Version    VersionCmd    `cmd:"" help:"Print version information."`
       Completion CompletionCmd `cmd:"" help:"Print a shell completion script."`
       Config     ConfigCmd     `cmd:"" help:"Inspect configuration."`
       ACP        ACPCmd        `cmd:"" name:"acp" help:"Run the ACP agent on stdio (not yet implemented)."`
       LogLevel   string        `enum:"debug,info,warn,error" default:"info" help:"Log level."`
   }
   ```

9. **`internal/cli/main.go`:** `func Main(ctx context.Context, args []string, stdin *os.File, stdout, stderr io.Writer) (code int)`.
   - It builds `kong.New(&root, kong.Name("gobble"), kong.Description(…), kong.Writers(stdout, stderr), kong.Exit(func(c int) { panic(exitPanic{c}) }), kong.UsageOnError(), kong.Bind(…))` once.
   - **Completion mode** (P5) is checked straight after `kong.New` and before `Parse`.
   - It calls `k.Parse(args)`, recovers `exitPanic`, and maps a `*kong.ParseError` to 2. It never calls `kong.Parse` or `FatalIfErrorf`.
   - It runs the selected command with the `Output`, `term.Caps`, logger and signal context.
   - When `signalExit` reports a signal, that code wins.
10. **`internal/cli/chat.go`:** `ChatCmd` embeds `SharedFlags` and `Prompt []string arg:"" optional:""`.
    - `Run` validates the flags, resolves the mode and composes the input, then returns `ExitUsage` with `Error: the agent is not available yet (0002-PLAN Phase 2)`.
    - In line mode on a terminal it prints the same line before any editor starts.
11. **`version.go`, `configpath.go`, `acp.go`:**
    - `version [--json]` prints `{name, version, commit, date, go, os, arch}` from `internal/buildinfo`, written with `encoding/json/v2`.
    - `config path [--json]` prints `config: …`, `data: …`, `state: …` and `cache: …`, or snake_case JSON, from `internal/appdirs`, following magic-cli-remote `paths.go`'s shape.
    - `acp` returns 2 with `Error: gobble acp is not yet implemented (0002-PLAN Phase 1)`, and writes nothing to stdout.
12. **`cmd/gobble/main.go`:**

    ```go
    func main() {
        restore, err := term.PrepareConsole(os.Stdout, os.Stderr)
        code := cli.Main(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr)
        if err != nil && code == cli.ExitOK {
            code = cli.ExitFailure
        }
        restore()
        os.Exit(code)
    }
    ```

    `cmd/gobble/doc.go`'s comment changes from "Phase 1 is a compile stub…" to "Package main is the gobble command. It prepares the console and runs cli.Main."
13. **`Makefile`:**
    - ~~Add `VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)`, `COMMIT`, `DATE` (the commit time, per the 0004-PLAN 2026-10-01 amendment), `BUILD_KIND ?= local` and `LDFLAGS` with the four `-X` variables of `internal/buildinfo`.~~
      *(Changed 2026-10-05 by the owner's buildinfo decision, 0004-MADR amendment of that date.)* Add `VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.0.0-dev)`, `BUILD_KIND ?= local` and `LDFLAGS` with the two `-X` variables of go-selfupdate-lib `buildinfo`, `VersionVar` and `KindVar`. Commit and date come from `vcs.revision` and `vcs.time`, which `go build` records in a checkout. The `verify-build-metadata` check below asserts them against `git rev-parse HEAD` and the commit time.
    - Add the target `bin/gobble$(BIN_EXT)`.
    - Replace the `verify-build-metadata` placeholder with a check. It builds with `LDFLAGS` into a temp directory, asserts that `version --json` reports the stamped `version`, `commit` and `date`, and asserts that a build without `-ldflags` reports a parseable `0.0.0-dev` version (D19 item 1).
14. **Tests:**
    - Kong tests run `Main` with argument tables.
    - Exit-code tests cover usage cases, `--fork` with `--session`, and a missing `@path`.
    - `version --json` is checked against a golden file.
    - `resolveMode` is table-tested.
    - `composeInput` is tested.
    - A signal test calls `watchSignals` with a fake `notify` that delivers `os.Interrupt`, `syscall.SIGHUP` or `syscall.SIGTERM` (the last two on Unix only). It asserts that `signalExit` returns 130, 129 and 143. It is seen to fail on a scratch copy that cancels without a cause.

**Verification:**
- `go build ./cmd/gobble` succeeds.
- The three Goal item 1 commands behave as stated.
- `make verify-build-metadata` passes.
- **Negative tests:**
  - a scratch build without `LDFLAGS` fails the stamped assertions;
  - a scratch copy whose `Exit` returns instead of panicking fails the `--help` test, because parsing continues after help (go-tui-lib 0006-MADR A1);
  - a scratch file outside `internal/cli` importing Kong fails rule 9.
- Then the Stability rule.

### P5 — Native completion (D17; closes F6, F30)

~~**Precondition:** zsh and fish are installed in WSL Ubuntu 24.04. The owner runs `sudo apt-get install -y zsh fish`, and `zsh --version` and `fish --version` print versions. They were measured absent on 2026-10-04. P5 does not start without them (C5).~~ *(Superseded 2026-10-04 by the owner: zsh is on the macOS host, and fish is installed in WSL and on a Linux host.)*

**Precondition (2026-10-04):** each host in the table below has its shells (C5).

   Real-shell hosts, measured 2026-10-04:

   | Host | Shells |
   |---|---|
   | Windows (this host) | Git Bash, pwsh 7.6.6, Windows PowerShell 5.1 |
   | WSL Ubuntu 24.04 | bash 5.2, fish 3.7.0 (installed on that date) |
   | A Linux host, Ubuntu 26.04 | bash, fish 4.2.1 (installed on that date) |
   | The macOS host | zsh, as the owner states |

   fish 3.7.0 exercises the script's `commandline -opc` fallback; fish 4.2.1 exercises `-xpc`.

**Deviations, 2026-10-05, the owner's decisions before P5 ran:**

1. **`--provider`.** Step 4's "static provider registry" does not exist until 0005-PLAN F4, and only `llm/provider` may import the SDK (rule 4). `ProviderSource` is registered with a lister that returns no candidates, as step 4 already does for sessions, models and tools. F4 supplies the real list.
2. **PowerShell result kinds.** D17's typed PowerShell results cannot be told apart from `value[\tdescription]` lines. When `GOBBLE_COMPLETE=powershell`, each line is `value\tdescription\tkind`, where kind is `command`, `flag`, `value`, `file` or `dir`. Other shells keep two fields. 0008-MADR D17 is amended.
3. **The 80-rune cap applies to descriptions only.** Step 5 says "values and descriptions … are capped at 80 runes", but a cut value completes to a token that does not exist. Values are sanitised and never cut.
4. **bash 3.2.** macOS's `/bin/bash` is 3.2, which has no `mapfile` and no `compopt`. The script reads results with a `while IFS= read -r` loop and calls `compopt` only when it exists. On 3.2, completing a directory therefore adds a trailing space, and the order may be re-sorted. The macOS shell test runs both `/bin/bash` (3.2) and Homebrew bash. 0008-MADR D17 is amended. *(Measured 2026-10-05 in an interactive bash in a pty, typing `gobble dé x` and Tab: `COMP_POINT` counts characters in bash 4.4.18, 5.0, 5.1, 5.2.21, 5.3.9 and 5.3.20, and bytes in 3.2.57. 4.0 and 4.2 did not build with a current compiler, so 4.0–4.3 are unmeasured. The script cuts `COMP_LINE` in bytes when `BASH_VERSINFO[0] < 4`, otherwise in characters.)*

**Choices made within the steps' wording:**

- `cli.Main` removes every `GOBBLE_COMPLETE*` variable from its own environment as soon as completion mode is ruled out, so no child it starts later can inherit one (step 7).
- The registry of `complete:` sources is built in `internal/cli/completion.go`. The coverage test lives beside it, because `internal/cli/complete` cannot import `internal/cli`. Positional arguments are covered too: chat's prompt words carry `complete:"none"`.
- Remote hosts are run from a copy of the working tree sent over SSH into a scratch directory there. No checkout and no commit are made on those hosts.

1. **`internal/cli/complete/candidate.go`:** the types of 0008-MADR D17, namely `Candidate`, `Kind`, the `Directive` bits (`Error=1, NoSpace=2, NoFileComp=4, FilterDirs=16, KeepOrder=32`), `Completer` and `CompleterFunc`, plus `type Registry map[string]Completer`.
2. **`internal/cli/complete/walk.go`:** `func Resolve(app *kong.Application, words []string) Target`, where `Target` names the command node, the flag awaiting a value or the positional index, and the prefix. It implements D17's state machine:
   - children and aliases, hidden ones skipped;
   - Kong's two `withargs` rules (`context.go:399-403`, `:569-576`);
   - short and long names, and one-character aliases as `-x`;
   - negation: `--no-<name>`, or `--<custom>` when `Negatable` is not `_`. These are copied from Kong's unexported `negatableFlagName`, with a provenance comment;
   - `--name=value` and short clusters;
   - `--`, passthrough, and the positional index.

   It never calls `Node.AllFlags` or `kong.Trace`.
3. **`internal/cli/complete/lex.go`:** `func SplitLine(line string, point int) (words []string, cur string)`. It is a POSIX lexer handling single quotes, double quotes and backslash, with no expansion, and it tolerates an unterminated quote. `point` is measured in bytes, and a multibyte test settles bash's unit (MADR [unverified]).
4. **`internal/cli/complete/sources.go`:**
   - `EnumSource` reads `Value.EnumSlice()`. `PathSource(dirsOnly bool)` reads the directory of the prefix with `os.ReadDir`, stopping early through `iter.Seq` at 200 entries, and appends the OS separator and `NoSpace` to directories.
   - `ProviderSource` reads the static provider registry.
   - `SessionSource`, `ModelSource` and `ToolSource` take a lister interface. Until 0002-PLAN Phase 4 and the provider phases supply listers, they are registered with listers that return no candidates. That is the true state, because no sessions or catalogs exist yet. They are not a skip.
   - Tags: `complete:"path"`, `"dir"`, `"provider"`, `"session"`, `"model"`, `"tools"` and `"none"`.
5. **`internal/cli/complete/protocol.go`:** `func Run(ctx context.Context, app *kong.Application, reg Registry, env func(string) string, args []string, out io.Writer) int`.
   - It checks `GOBBLE_COMPLETE` and `GOBBLE_COMPLETE_PROTOCOL=1`.
   - It reads words from argv after `--` (zsh and fish), from `GOBBLE_COMPLETE_LINE`/`POINT`/`CUR` (bash), or from `GOBBLE_COMPLETE_WORDS` split on U+001F (PowerShell).
   - It applies `context.WithTimeout(ctx, 150*time.Millisecond)`.
   - It writes `value[\tdescription]` lines and `:<directive>` to a buffer, then once to `out`.
   - Values and descriptions pass `term.Sanitize` and are capped at 80 runes.
   - Debug output goes only to the file named by `GOBBLE_COMPLETE_DEBUG`.
   - It always returns 0. A protocol mismatch writes only `:1`.
6. **`internal/cli/complete/scripts/*.tmpl` and `scripts.go`:** the four templates of 0008-MADR D17, embedded with `//go:embed scripts/*.tmpl` and rendered with `text/template`, taking `{{.Name}}` and `{{.Protocol}}`. No `eval`, no `Invoke-Expression`.
7. **`internal/cli/completion.go`:**
   - `CompletionCmd{Shell string arg:"" enum:"bash,zsh,fish,powershell"}` prints the rendered script.
   - `cli.Main` calls `complete.Run` before `Parse` when `GOBBLE_COMPLETE` is set and not `0`. It removes the `GOBBLE_COMPLETE*` variables from the environment of any child it starts.
8. **Tests:**
   - Engine table tests: the `withargs` root, `--model=an`, `-cn`, `--no-approve`, `--`, aliases, a hidden flag, and the cut at the cursor. `kong.Trace` is the oracle for the reached command.
   - Protocol tests: the built binary with an explicit `Env` covering U+001F words, a multibyte path and the C6 empty-home check.
   - Golden scripts: `testdata/*.golden`, refreshed with `-update`.
   - Syntax checks: `bash -n`, `fish --no-execute` (WSL and the Linux host) and `zsh -n` (the macOS host). PowerShell `[System.Management.Automation.Language.Parser]::ParseInput` must report zero errors under `pwsh` and under `powershell.exe` (5.1).
   - A coverage test: every value-taking flag has an `enum` or a `complete` tag.
   - `shells_test.go` (`//go:build shells`), run by `make completion-shells`:
     - bash: `_gobble` with `COMP_*` set, checking `COMPREPLY`;
     - fish: `complete -C 'gobble --output-format '`;
     - zsh: `zsh -n` plus a `compadd` stub harness;
     - PowerShell: `TabExpansion2 -inputScript 'gobble --output-format ' -cursorColumn 24` under `pwsh` and `powershell.exe`, asserting `text`, `json` and `stream-json` with result type `ParameterValue`.

**Verification:**
- `go test ./internal/cli/complete/...` passes.
- `make completion-shells` passes on every host in the P5 table:
  - Windows: bash, pwsh, powershell.exe;
  - WSL: bash, fish 3.7.0;
  - the Linux host: bash, fish 4.2.1;
  - the macOS host: zsh, bash.
  - A host is run by copying the tree there, as the WSL preflight does, or from a checkout at the staged commit.
- The `--session` completion request's wall time is recorded, median of 20 runs.
- **Negative tests:**
  - a corrupted golden fails;
  - a scratch copy with one `complete` tag removed fails the coverage test;
  - `GOBBLE_COMPLETE_PROTOCOL=2` returns only `:1`;
  - a scratch completer that writes a file under `HOME` fails C6.
- Then the Stability rule.

### P6 — Renderer (D7, D8, D19 item 9; closes F9, F12, F13, F24 terminal side, F27)

1. **`internal/cli/render/mdstream.go`:** `type Stream struct`, with `Push(chunk string) string` and `Flush() string`. It reimplements the behaviour of goose's `MarkdownBuffer`: it releases only text that cannot change meaning (no open fence, table, heading, inline code, emphasis, link or image), and keeps a resumable scan checkpoint. Its fixtures are recorded chunk sequences, including a fence split across chunks, a table arriving row by row, and `**` split across chunks.
2. **`internal/cli/render/mdstyle.go`:** `func Style(lines string, s term.Stream) string` colours headings bold, inline code and fences dim, shows the fence language label, and colours list and quote markers. Markers are kept, and nothing is reflowed. When `!s.Color` the input is returned byte for byte.
3. **`internal/cli/render/table.go`:** GFM table re-layout with alignment from `:`, measured in display width.
   - First `go get golang.org/x/text@v0.42.0`.
   - `func displayWidth(s string) int` counts 2 for a rune whose `width.LookupRune(r).Kind()` is `width.EastAsianWide` or `width.EastAsianFullwidth`, 0 for a combining mark (`unicode.Mn`, `unicode.Me`) or a zero-width joiner, and 1 otherwise.
   - The same function measures the tool-header truncation in item 4.
   - Tested with ASCII, CJK and an emoji with a joiner.
4. **`internal/cli/render/tool.go`:**
   - A header line: glyph `▸` plus `term.Sanitize(title)`, truncated to the width in display columns.
   - A status suffix.
   - Output over 20 lines shows the first 10 and the last 10, plus `… N lines hidden (--verbose shows all)`.
   - A unified diff with `+` and `-` coloured.
   - Paths shortened under `~` with `filepath.Rel`, so Windows separators work.
5. **`internal/cli/render/status.go`:** `func StatusLine(elapsed time.Duration, used, size int64, cost *float64, s term.Stream, g term.Glyphs) string`. It draws a 20-cell bar: green under 75%, yellow from 75%, red from 90%, which are the phone's thresholds. Tested at 74.9%, 75% and 90%.
6. **`internal/cli/render/spinner.go`:** one goroutine started with `sync.WaitGroup.Go`, writing `\r` + frame + `ESC[K` to the error writer only when `Err.TTY`. It stops on context cancel. Its timing is tested with `testing/synctest`.

**Verification:**
- `go test ./internal/cli/render/...` passes against golden files.
- **Negative tests:**
  - a scratch `Stream` that releases text at once splits a fence in a golden diff;
  - a threshold changed to 85% fails the 75%/90% tests;
  - the 20-line bound removed fails the tool golden.
- Then the Stability rule.

### P7 — Close-out (records)

1. Set this PLAN's `status: completed`. Append `## Execution record (YYYY-MM-DD)` with:
   - each phase's staged files;
   - both hosts' preflight results;
   - every negative test's quoted failure;
   - the completion and `--version` latency baselines;
   - `go build` size of `cmd/gobble` before P4 (no binary) and after P6;
   - what the plan predicted wrongly.
2. Append `## Observed — execution results (YYYY-MM-DD)` to 0008-MADR. Record whether the D17 budget held, whether conhost behaved as in measurement 10 under the real editor, and bash's `COMP_POINT` unit.
3. In `docs/README.md`, set the 0008 PLAN row to `completed`.

**Verification:** `make check-records` and `make markdownlint` exit 0. Then the Stability rule.

## Verification (whole plan)

From a copy of the final tree, on Windows (Git Bash) and in WSL:

```text
make preflight                        # exit 0, "preflight passed"
make probe-conhost                    # Windows only; exit 0
make completion-shells                # Windows: bash, pwsh, powershell.exe; WSL: bash, fish 3; Linux host: bash, fish 4; macOS host: zsh, bash
go run ./cmd/gobble version --json    # {"name":"gobble",…}
go run ./cmd/gobble -p "x"; echo $?   # stderr "Error: the agent is not available yet …"; 2
GOBBLE_COMPLETE=fish GOBBLE_COMPLETE_PROTOCOL=1 go run ./cmd/gobble -- --output-format ''   # text, json, stream-json, then ":4"
go mod tidy -diff                     # empty
git ls-files 'internal/cli/**' | xargs grep -l 'charm.land\|charmbracelet\|go-tui-lib\|acp-go-sdk'   # no output
```

### Acceptance criteria (mapped to MADR Confirmation)

| # | Criterion | MADR |
|---|---|---|
| A1 | Bare `gobble` on a terminal resolves to line mode and never starts a wizard; `-p` or piped input resolves to print mode | Confirmation 3; D1 |
| A2 | Composed input from stdin `hi` and words `ask` is `hi\n\nask` | Confirmation 2; D4 |
| A3 | Stdout carries no diagnostics; `Error:`/`Warning:` go to stderr; EPIPE exits 0 | Confirmation 1; D3 |
| A4 | Exit 2 for `--fork` with `--session`; 130 for SIGINT | Confirmation 10; D10 |
| A5 | `gobble completion <shell>` prints a script for all four shells, and `go.mod` gains no completion module | Confirmation 5; D17 |
| A6 | A dynamic completion request answers within 150 ms, writes nothing to stderr, and creates no file | Confirmation (completion); D17, C6 |
| A7 | Real-shell completion passes in bash, zsh, fish, pwsh 7 and PowerShell 5.1 | Confirmation (completion, all shells); D17 |
| A8 | The coverage test fails when a `complete` tag is removed | Confirmation (coverage); D17 |
| A9 | `NO_COLOR=1` yields no SGR, and Windows VT is still enabled | Confirmation 6; D2 |
| A10 | The sanitiser neutralises `ESC[2J`, OSC 0 and U+202E, and fails on an unsanitised copy | Confirmation 8; D8 |
| A11 | Ctrl+C clears a non-empty line; a 3-line bracketed paste submits once; the conhost probe passes | Confirmation 9 and the conhost row; D6 |
| A12 | sloglint, depguard and forbidigo plants fail; the tree is clean | Confirmation (golangci-lint); D18 |
| A13 | archtest: no Charm or go-tui-lib under `internal/cli`; Kong only under `internal/cli` | Confirmation 7; D14 |
| A14 | The status bar turns at 75% and 90% | D7, F27 |
| A15 | `version --json` on a dev build reports a parseable `0.0.0-dev` version | Confirmation (acp handshake), the build half; D19 item 1 |
| A16 | P0's amendments exist and keep the text they supersede | D14, D16, D20 |

The handshake, replay and session-lock rows of the MADR's Confirmation are checked by the 0002-PLAN phases that P0 hands them to.

**The criterion most likely to be dropped is A7.** Installing zsh and fish, and driving `TabExpansion2` under two PowerShell editions, is slow and fiddly. Golden scripts alone look like coverage. They are not: only a real shell shows whether `COMP_WORDBREAKS`, zsh quoting or PowerShell's argument passing break the protocol.

## Rollout and Rollback

- There are no users and no release. Each phase is staged on the current branch, and the owner commits.
- **Rollback:** `git revert` of the owner's commit for a phase. Later phases depend on earlier ones as the delivery order shows. Reverting P4 removes the binary and restores the `verify-build-metadata` placeholder.
- Nothing is pushed, tagged or released by this plan.

## Deferred (named, so they are not mistaken for oversights)

- **D5 and D9 wiring, D13, D19 items 1–8.** These are 0002-PLAN phases, handed over by P0, because they need the agent and `acpclient`.
- **Real session, model and tool completion sources.** The listers arrive with 0002-PLAN Phase 4 (sessions) and the provider and tool phases. P5 ships the sources and their tags.
- **Code syntax highlighting in the line client.** That needs `chroma` (0008-MADR option D) and a later record.
- **Adopting go-tui-lib's `safetext`, `stream` or `kongcmd`.** That waits until they ship Charm-free under a tag, and a record narrows rule 5 (0008-MADR D8, D17).
- **Windows Terminal (`wt.exe`) paste probe.** The conhost suite covers Console Host. Windows Terminal is documented to support bracketed paste, and probing it is a follow-up line in the Execution record if `make probe-conhost` is extended.
- **CI wiring of `completion-shells` and `probe-conhost`.** 0004-PLAN Phase 4 owns `ci.yml`. A `conhost` window in a CI runner is **[unverified]**.
- **Companion changes in magic-cli-remote (D20).** These are out of scope by the owner's direction.

## Execution record (2026-10-05, interim)

P7 writes the final record. This interim one keeps the evidence of the phases that have run.

### Phases

| Phase | State | Staged files |
|---|---|---|
| P0 | ran 2026-10-04; committed by the owner in `4499e89` | the eight record files, plus this PLAN |
| P1 | ran; in `4499e89` | `.golangci.yml`, `internal/archtest/check.go`, `internal/archtest/check_test.go` |
| P2 | ran; in `4499e89` | `go.mod`, `go.sum`, `internal/cli/term/*` (13 files) |
| P3 | ran 2026-10-05; staged | `Makefile` (`probe-conhost`), `internal/cli/editor/*` (12 files) |
| P6 | ran 2026-10-05; staged | `go.mod`, `go.sum` (`golang.org/x/text v0.42.0`), `internal/cli/render/*` (14 Go files, 4 testdata files) |
| P4 | ~~**not run**~~ ran 2026-10-05 after 0004-PLAN Phase 2 steps 2–4 (`55f35ba`); staged | `go.mod`, `go.sum` (Kong `v1.16.1`), `Makefile`, `scripts/verify_build_metadata.py`, `cmd/gobble/main.go`, `cmd/gobble/doc.go`, `internal/cli/*` (19 Go files, 2 goldens), `internal/buildinfo` (the version fix) |
| P5 | ran 2026-10-05; staged | `Makefile` (`completion-shells`), `internal/cli/complete/*` (11 Go files, 4 script templates, 4 goldens), `internal/cli/completion.go` and its test, `internal/cli/main.go`, `chat.go`, `main_test.go` |
| P7 | **not run** | Needs P4 and P5. |

After each of P1, P2, P3 and P6, `make preflight` printed `preflight passed` and exited 0 on Windows (Git Bash) and in WSL Ubuntu 24.04 on a copy of the tree. `git diff --check` and `git diff --cached --check` were clean. `go test -race ./internal/...` passed in WSL, and `go test ./...` passed on Windows. `go mod tidy -diff` was empty. No file under `internal/cli` names Charm, go-tui-lib, the ACP SDK or Kong.

### Negative tests (C4), each on a scratch copy

- **P1, lint.** A planted `hook/zz_planted.go`, run with `--uniq-by-line=false`, failed with:
  - `import 'log' is not allowed from list 'logging'` (depguard);
  - `use of fmt.Println forbidden`, `use of log.Printf forbidden`, `use of os.Stderr forbidden`, `use of os.Stdout forbidden` and `use of os.Exit forbidden` (forbidigo);
  - `default logger should not be used`, `key-value pairs and attributes should not be mixed`, `key-value pairs should not be used`, `message should be a string literal or a constant`, `keys should be written in snake_case`, `InfoContext should be used instead` and `the "msg" key is forbidden and should not be used` (sloglint);
  - `for loop can be changed to use an integer range` (intrange) and `net/http.Get must not be called` (noctx).
- **P1, archtest.** `agent/zz.go` importing Kong failed `TestImportRules/rule_9_module` with `rule 9: github.com/maccavelli/gobble-cli/agent imports github.com/alecthomas/kong`.
- **P2.** A `Sanitize` that returns its input failed 14 subtests of `TestSanitize`, among them `Sanitize("a\x1b[2Jb") = "a\x1b[2Jb", want "ab"`.
- **P3.**
  - Coalescing removed: `submissions = ["line1" "line2" "line3"] (then err EOF), want ["line1\nline2\nline3"]`.
  - Ctrl+C mapping removed: `submissions = [] (then err EOF), want ["xyz"]`, which is `io.EOF` on a non-empty line.
  - Closing stdin in `reader.go`: `reader.go calls Close: stdin is never closed (0008-MADR F20)`.
  - The live suite with coalescing removed: `child submissions = ["line1" "line2" "<nil>"], want ["line1\nline2\nline3" "xy" "<nil>"]`.
- **P6.**
  - A `Stream` that releases everything at once failed 21 of the 29 fixtures, among them `fence_split_across_chunks_inside_a_string_literal`, `table_arriving_row_by_row` and `bold_split_mid-word`.
  - `warnAt = 85`: `75%_is_yellow` failed, with `StatusLine = "1s • <ESC>[32m…" want colour "<ESC>[33m"`.
  - The 20-line bound removed: `tool_output_25 differs from testdata\tool_output_25.golden`.

### The live Console Host suite

`make probe-conhost` passed three times out of three, each run taking 0.11 s, with `console window class "ConsoleWindowClass"`. The child's `Editor` returned `line1\nline2\nline3` as one submission for a real conhost paste, and Ctrl+C cleared `abc` before `xy` was submitted. Console Host behaved as in measurement 10 under the real editor.

### What the plan predicted wrongly

1. **P1 step 2** says to "include 9 in the loop that runs every rule". There is no such loop: `TestImportRules`'s table is the loop, and the `rule 9 module` case runs rule 9 against the real graph.
2. **A false alarm in P1, withdrawn.** A suspected forbidigo gap (see the struck note in P1) was golangci-lint's default one-issue-per-line reporting: errcheck's `check-blank` finding hid forbidigo's on the same line. Plants are now run with `--uniq-by-line=false`. Measurement 12's configuration stands unchanged.
3. **P3 steps 2, 4 and 6** could not be built as written. They are recorded as dated deviations in place, with the owner's decisions.
4. **`os/exec` cannot start `conhost.exe`.** It always sets `STARTF_USESTDHANDLES`, and conhost started that way exits at once without running its child. The suite calls `windows.CreateProcess` with a plain `STARTUPINFO` and `CREATE_NEW_CONSOLE`; measurement 10 had used `Start-Process`.
5. **History cost.** 1,005 atomic rewrites took 5.1 s on this Windows host, so the cap test seeds the file once. One rewrite per submission is the production cost.
6. **The authoring tool turned `\u` escapes of U+2000 and above into literal characters** in the Go sources, bidi controls included. The tests build those runes from code points, and every new file was searched for bidi and zero-width characters (none found).
7. **`make preflight` runs no tests.** The Stability rule's preflight gates are not a test run, so `go test` was run separately on both hosts after every phase.
8. **`golang.org/x/text`** was already in the module graph at v0.3.0 (indirect, through tooling). `go get` raised it to v0.42.0, and only the direct requirement line changed in `go.mod`.

### Choices made inside the plan's wording

- `Caps.Unicode` follows the plan's literal rule: any of `LC_ALL`, `LC_CTYPE` or `LANG` naming UTF-8 is enough. That is not POSIX precedence.
- `PrepareConsole` sets only `ENABLE_VIRTUAL_TERMINAL_PROCESSING|ENABLE_PROCESSED_OUTPUT`, as the plan says. magictools also sets `ENABLE_WRAP_AT_EOL_OUTPUT`.
- `render.Style` dims the fence lines and inline code, and leaves the code inside a fence unstyled. Syntax highlighting is deferred.
- The `… N pasted lines` notice is printed once per submission, with N counting the line ended by Enter.

### P4, 2026-10-05

- **Deviations.** Three were decided by the owner before P4 ran and are recorded in P4: per-OS broken-pipe files, the build check in Python, and a `completion` stub. A fourth was found during the run: step 2's version kept Go's VCS pseudo-version. The owner chose to fix it now; it is recorded in 0004-PLAN's step-2 entry and in 0009-REPORT M7.
- **Gates.**
  - `make preflight` printed `preflight passed` on Windows and in WSL. `verify-build-metadata` is now real, and passed all seven checks on both hosts.
  - `go test -race ./...` passed in WSL, and `go test ./...` on Windows.
  - `golangci-lint` reported `0 issues.` for linux, darwin and windows.
  - The pre-add check was clean for all 26 files.
- **Goal item 1, by hand, with an isolated `GOBBLE_HOME`.**
  - `version --json` printed `{"name":"gobble","version":"0.0.0-dev+55f35bac850a.dirty",…}`.
  - `config path` printed the four roles.
  - `acp` exited 2 and wrote nothing to stdout.
  - `-p x`, piped `hi` with `ask`, and a bare run all exited 2 with `Error: the agent is not available yet (0002-PLAN Phase 2)`. With stdin at end of file, a bare run is print mode with no prompt, so it says that instead.
  - `--fork a --session b` gave the D12 conflict, and `--model x` gave `is not yet available (0005-PLAN F4)`.
- **Baselines** on this host:
  - the stamped `bin/gobble.exe` is 10,230,272 bytes (9.76 MiB);
  - `gobble version` takes a median of 10.2 ms over 20 runs (min 9.3, max 28.8).
- **Negative tests**, each on a scratch copy, each failing as expected:
  - the stamped build without `LDFLAGS`: `FAIL stamped version: got '0.0.0-dev+55f35bac850a.dirty', want '1.2.3'`;
  - a Kong `Exit` that returns: `[--help]: exit 2, stderr "Error: expected one of \"chat\", …"`, because parsing went on after help (go-tui-lib 0006-MADR A1);
  - cancelling without a cause: `signalExit = 0, false; want 130, true`, and the same for 129 and 143;
  - an `acp` start-up banner: `gobble acp wrote to stdout: "gobble acp starting\n"` (0004-PLAN Phase 2 Accept);
  - the broken-pipe check disabled: `exit 1, stderr "Error: write stdout: write |1: The pipe is being closed."`. That is Windows' `ERROR_NO_DATA`, and it confirms deviation 1;
  - Pi's `join("")`: `Text = "hiask"`;
  - an eager logger: `version created [d logs/]`;
  - pseudo-versions kept: `Version = "1.2.4-0.20261005131429-e825cdafd332", want "0.0.0-dev+4e25c8a01234"`;
  - Kong imported from `agent` (rule 9, now with the real module): `rule 9: github.com/maccavelli/gobble-cli/agent imports github.com/alecthomas/kong`.
- **What the plan predicted wrongly.**
  1. **`kong.UsageOnError()` does nothing here.** It affects only `FatalIfErrorf`, which gobble never calls (`options.go:391`). It was left out. `Main` prints `Error: <parse error>` and `Error: run 'gobble --help' for usage` itself.
  2. **`cmd/gobble/doc.go` held Phase 1's stub `func main()`**, not only a comment. The stub was removed with the comment, because `main.go` now defines `main`.
  3. **Go 1.24+ stamps a pseudo-version.** This is deviation 4 above.
  4. **Kong checks an enum even when the flag is empty and not required** (`context.go:200`), so `--thinking` lists `""` among its values.
  5. **Print mode reads piped stdin to the end, as D1 and Pi require.** A non-terminal stdin that never closes, such as an agent harness's shell, makes `gobble` or `gobble -p x` wait. Passing `</dev/null` avoids that. This is the behaviour the plan specifies, recorded because it surprised the first manual run.
- **Choices within the steps' wording:**
  - the logger opens lazily, so `version`, `config path` and `--help` create nothing (tested);
  - every "not yet available" message names its owning plan;
  - `version` without `--json` prints `gobble <version>` and then `commit:`, `date:` and `go:` lines.

### P5, 2026-10-05

- **Deviations.** Four were decided by the owner before P5 ran and are recorded in P5: an empty provider lister, a third field for PowerShell, the description-only cap, and bash 3.2 support. 0008-MADR D17 is amended for the second and fourth.
- **Gates.**
  - `make preflight` printed `preflight passed` on Windows and in WSL.
  - `go test -race ./...` passed in WSL.
  - `golangci-lint` reported `0 issues.` for linux, darwin and windows, and with `-tags shells`.
  - The pre-add check was clean for all 36 files.
- **`make completion-shells`**, on every host in the P5 table, with each shell's version as the suite logged it:

  | Host | Shells, all passing |
  |---|---|
  | Windows | Git Bash 5.3.15, pwsh 7.6.6, Windows PowerShell 5.1.26100 |
  | WSL Ubuntu 24.04 | bash 5.2.21, fish 3.7.0 (the `commandline -opc` fallback) |
  | the Linux host | bash 5.3.9, fish 4.2.1 (`-xpc`) |
  | the macOS host | zsh 5.9, Homebrew bash 5.3.20, `/bin/bash` 3.2.57 |

  The macOS host has GNU Make 3.81, and the target ran there unchanged.
- **The bash `COMP_POINT` unit (0008-MADR D17 [unverified]) is settled.** It counts characters from bash 4.4 on and bytes in 3.2, as measured in deviation 4. The suite's mid-line case (`gobble --file dé --verbose`, cursor after `dé`) passes on all four bash versions above.
- **Baselines** on Windows:
  - a `--session` completion request takes a median of 9.7 ms over 20 runs (min 9.2, max 13.5), against the 150 ms deadline;
  - the stamped binary is 10,860,032 bytes (10.36 MiB), up from 9.76 MiB after P4.
- **Negative tests**, each on a scratch copy, each failing as expected:
  - a corrupted golden: `fish script differs from testdata\fish.golden`;
  - `complete:"session"` removed: `chat session takes a value but has neither enum nor complete tag`;
  - the protocol check removed: `protocol_mismatch: output = "chat\nrun\nshell\nversion\n:4\n", want ":1\n"`;
  - a completer writing under `HOME`: `completion mode wrote under the home directory (C6): [… cache.json]`;
  - the walker's withargs rule 2 removed: `Resolve reached "gobble", kong.Trace reached "chat"`;
  - a single file kept NoSpace: `TestProtocolPathSpacing` failed on `notes.md`;
  - bash 3.2 cutting in characters, on the macOS host: `"gobble --file dé --verbose": COMPREPLY = ["acp" "chat" "completion" "config" "version"], want ["dés.txt"]`. Only `bash_3.2` failed, and Homebrew bash 5.3 passed.
- **What the plan predicted wrongly.**
  1. **The `COMP_POINT` unit depends on the bash version.** The plan expected one unit to settle. The script has to branch on `BASH_VERSINFO`.
  2. **Path completion's NoSpace is decided after collection.** A single file left takes a space and a single directory does not. The first fish run caught this: fish's no-space trick was offering a file twice, as `dés.txt` and `dés.txt.`. The fish script also skips the trick after `@=/:.,`, as Cobra's does.
  3. **bash passes the whole line,** so the program name is dropped in Go. The first binary test caught it; the unit tests had passed only because the stray `gobble` word entered the withargs command.
  4. **The PowerShell harness must not print through the console's code page.** The completer decodes gobble's UTF-8 itself, then restores the user's code page. Results are written to a UTF-8 file instead.
  5. **The engine oracle needed care.** `kong.Trace` selects the default command when nothing is named. Accepting "root" in that case was right only when every earlier word is a flag. The first, looser comparison let the withargs-rule negative pass, and was tightened.
  6. **The `TabExpansion2` cursor column** in step 8 is 24; for `'gobble --output-format '` it is that string's length, 23. The suite uses the length.
  7. **`powershell.exe` on this host resolves through a scoop shim.** It reports Windows PowerShell 5.1.26100, so 5.1 is what ran.

## Execution record (2026-10-05, final)

Every phase has run. This section closes the plan; the interim record above holds each phase's evidence and is kept as written.

### Commits

The agent staged each phase, and the owner committed and pushed:

| Commit | Contents |
|---|---|
| `4499e89` | P0 records, P1 lint and archtest rule 9, P2 terminal layer |
| `4e25c8a` | P3 line editor, P6 renderer |
| `55f35ba` | 0004-PLAN Phase 2 steps 2–4 (`buildinfo`, `appdirs`, `logging`), P4's precondition |
| `6e53d05` | P4 Kong root and process edge, with the step-2 version fix and 0009-REPORT |
| `98073db` | P5 native completion |
| (this change) | P7 close-out records |

### Whole-plan verification, at `98073db`

| Command | Result |
|---|---|
| `make preflight` | `preflight passed` on Windows and in WSL; `verify-build-metadata` passed all seven checks |
| `make probe-conhost` | passed, `console window class "ConsoleWindowClass"` |
| `make completion-shells` | passed on Windows: Git Bash 5.3.15, pwsh 7.6.6, Windows PowerShell 5.1. WSL, the Linux host and the macOS host were not re-run at this commit. Their last runs passed in P5 on the same code, which no later change touched: WSL bash 5.2.21 and fish 3.7.0; the Linux host bash 5.3.9 and fish 4.2.1; the macOS host zsh 5.9, bash 5.3.20 and `/bin/bash` 3.2.57 |
| `go run ./cmd/gobble version --json` | `{"name":"gobble","version":"0.0.0-dev","commit":"","date":"","go":"1.27.1",…}`; `go run` records no VCS stamp |
| `go run ./cmd/gobble -p "x"` | `Error: the agent is not available yet (0002-PLAN Phase 2)` and `exit status 2` |
| `GOBBLE_COMPLETE=fish … -- --output-format ''` | `text`, `json`, `stream-json`, then `:36` |
| `go mod tidy -diff` | empty |
| the forbidden-import grep over `internal/cli` | no output |

### Acceptance criteria

| # | Criterion | Result and evidence |
|---|---|---|
| A1 | Line mode on a terminal; `-p` or piped input is print mode | met: `TestResolveMode` (all eight cases), `TestPipedInputIsComposed` (`"mode":"print"`) |
| A2 | `hi` and `ask` compose to `hi\n\nask` | met: `TestComposeInput`; Pi's `join("")` mutation gave `"hiask"` |
| A3 | Stdout carries no diagnostics; EPIPE exits 0 | met: `TestMainExitCodes`, `TestDiagnosticPrefixes`, `TestBrokenStdoutExitsZero` (real broken pipes, `ERROR_NO_DATA` on Windows) |
| A4 | Exit 2 for `--fork` with `--session`; 130 for SIGINT | met: `TestMainExitCodes`, `TestWatchSignals` (130, 129, 143) |
| A5 | Scripts for four shells; no completion module | met: `TestCompletionScriptsPrinted`; `go.mod` gained Kong, `x/term`, `x/text` and go-selfupdate-lib only |
| A6 | A request answers within 150 ms, silent on stderr, creating nothing | met: median 9.7 ms; `TestBinaryCompletionMode` (C6 empty home, empty stderr); `TestProtocolDeadline` |
| A7 | Real-shell completion in bash, zsh, fish, pwsh 7 and PowerShell 5.1 | met: on all four hosts, including bash 3.2 and fish 3 and 4 |
| A8 | The coverage test fails when a `complete` tag is removed | met: `chat session takes a value but has neither enum nor complete tag` |
| A9 | `NO_COLOR=1` yields no SGR; Windows VT still enabled | met: `TestColorOK`. `PrepareConsole` runs before `cli.Main` and never reads `NO_COLOR` |
| A10 | The sanitiser neutralises `ESC[2J`, OSC 0 and U+202E, and fails on an unsanitised copy | met: `TestSanitize`, `TestEscapeBidi`; the mutation failed 14 subtests |
| A11 | Ctrl+C clears a line; a 3-line paste submits once; conhost passes | met: `TestReadPrompt`, `TestConhost` |
| A12 | sloglint, depguard and forbidigo plants fail; the tree is clean | met: P1 negatives; `0 issues.` for three GOOS |
| A13 | No Charm or go-tui-lib under `internal/cli`; Kong only there | met: rules 5 and 9, the rule-9 negative, the grep above |
| A14 | The status bar turns at 75% and 90% | met: `TestStatusLineThresholds`; the `warnAt = 85` mutation failed |
| A15 | A dev build reports a parseable `0.0.0-dev` version | met: `verify-build-metadata`, after the step-2 pseudo-version fix (0009-REPORT M7) |
| A16 | P0's amendments exist and keep what they supersede | met: P0; `bare session` occurs 5 times in 0006-MADR, up from 4 |

A7 was named the criterion most likely to be dropped. It was not: it needed SSH runs on two more hosts, a bash 3.2 code path, and a measurement of `COMP_POINT` in interactive bash.

### Baselines

| Measure | Value |
|---|---|
| `cmd/gobble` before P4 | no binary |
| stamped binary after P4 | 10,230,272 bytes (9.76 MiB) |
| stamped binary after P5, final | 10,860,032 bytes (10.36 MiB) |
| `gobble version`, median of 20 | 10.2 ms |
| `--session` completion request, median of 20 | 9.7 ms |

P6 ran before P4, so "after P6" in step 1 has no binary of its own. The final size is the one after P5.

### What the plan predicted wrongly, in addition to the phase lists above

1. **The whole-plan verification block.**
   - It expected `:4` after the `--output-format` values. Enum sources keep their declared order, so the directive is `:36` (keep order and no file completion).
   - It expected `go run …; echo $?` to show 2. `go run` exits 1 for any non-zero child and prints `exit status 2`, so the code is checked in that line, or with the built binary.
2. **The delivery order.** It drew P6 after P5. P6 needed only P2's term layer and ran with P3, before P4. Nothing depended on the drawn order.
3. **Scope.** Twelve deviations were taken to the owner, each before the affected code was written or fixed: one in P1 (withdrawn), three in P3, three in P4, four in P5, and the step-2 version defect found in P4. Each is recorded in place, with the files it added to the scope.

### Deferred, unchanged

The plan's Deferred list stands. Two items are added:

- **bash 4.0–4.3's `COMP_POINT` unit** is unmeasured: those releases would not build with a current compiler. The script treats them as counting characters.
- **The magic-cli-remote findings** M1–M7 wait in 0009-REPORT for the owner's review.
