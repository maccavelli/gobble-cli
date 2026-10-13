---
status: completed
date: 2026-10-12
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# PLAN 0018 — The configuration surface: a commented `settings.yaml`, edited in place, layered and trust-gated

Implements [0018-MADR-configuration-surface.md](0018-MADR-configuration-surface.md) decisions D1–D21, closing findings F1–F13. The MADR was accepted on 2026-10-12. Its record notes (D1's, D13's and D20's) are in the same docs change as this plan, so no phase repeats them.

## Goal

When this plan is done, these hold, and each is shown by a command in its phase:

1. With no `settings.yaml`, no `--set` and no new flag, gobble behaves exactly as before. Every existing test passes unchanged, except the two named in C1. `acpserver/testdata/session.golden` is byte-identical.
2. The first run of `gobble`, `gobble acp` or `gobble mcp …` creates `<config>/settings.yaml` from the embedded template, which is all comments. `version`, `completion`, `config path`, `config get`, `config validate` and `config schema` create no file.
3. Each of D14's thirteen keys, set in the global file, reaches its consumer. A session key set in a project's `.gobble/settings.yaml` is reported as disabled until F6 supplies trust. `--set` beats the files, and a flag beats `--set`.
4. `gobble config set` and `unset` change only the bytes their path needs. Comments, blank lines, spacing and line endings elsewhere are byte-identical, which a byte-diff test proves for each of D8's five cases. A file that does not parse is never written.
5. `gobble config update` adds the template blocks a file lacks, and changes no existing line.
6. The committed schema matches the generated one. The template matches the key table, by D9's six tests.
7. `--add-dir` sends additional directories on `session/new` and `session/resume`.
8. Every gate of the stability rule passes on Windows, ~~on WSL~~ *(not WSL, by the owner's direction of 2026-10-12 in the stability rule)* and on the second Linux machine, and each new test has been seen failing on a scratch copy.

## Scope

### In scope (the only files any phase may touch)

| Phase | New | Changed |
| :--- | :--- | :--- |
| P1 | `internal/filelock/doc.go`, `lock.go`, `lock_unix.go`, `lock_windows.go`, `lock_other.go`, `lock_test.go`; `internal/fsx/git.go`, `git_test.go` | *(deviation 1: `internal/fsx/write.go`, `internal/fsx/workspace_test.go`)* `session/jsonl/store.go`; `session/jsonl/lock.go`, `lock_unix.go`, `lock_windows.go`, `lock_other.go` (deleted); `prompt/context.go`; `mcpclient/file.go`, `mcpclient/file_test.go`; `mcpclient/conn.go`, `mcpclient/proc.go`, `mcpclient/proc_test.go` |
| P2 | `internal/config/keys.go`, `settings.go`, `read.go`, `merge.go`, `resolve.go`, `diag.go`, `secret.go`, `override.go`, `schema.go`, `settings.yaml`, `template.go`, `genschema/main.go`, `testdata/settings.schema.json`, ~~`testdata/*.yaml` fixtures~~ *(deviation 4: none; the fixtures are built in the tests)*, `read_test.go`, `resolve_test.go`, `override_test.go`, `schema_test.go`, `template_test.go` | `internal/config/doc.go`; `internal/frontmatter/frontmatter.go` (package comment only); *(deviation 2: `.golangci.yml`; deviation 5: `mcpclient/proc_test.go`, `tool/builtin/shell_test.go`, `acpserver/mcp_test.go`)* |
| P3 | `internal/config/edit.go`, `update.go`, `edit_test.go`, `update_test.go`, `testdata/edit/*` | — |
| P4 | `internal/cli/config.go`, `internal/cli/config_test.go`, `internal/cli/settings.go`; `internal/config/render.go`, `render_test.go` | `internal/cli/root.go`, `internal/cli/configpath.go`, `internal/cli/main.go`, `internal/cli/main_test.go`, `internal/cli/chat.go`, `internal/cli/acp.go`, `internal/cli/mcp.go`, `internal/cli/flags.go`, `internal/cli/completion.go`; *(deviation 9: `internal/cli/editor/external.go`, `internal/cli/editor/external_test.go`; deviation 13: `internal/config/edit.go`, `internal/config/diag.go`, `internal/config/edit_test.go`, `internal/config/resolve_test.go`)* |
| P5 | `acpserver/settings_test.go`, `internal/cli/settings_test.go` | `acpserver/agent.go`, `acpserver/queue.go`, `acpserver/mcp.go`, `acpserver/config.go`; `internal/cli/agent.go`, `internal/cli/sessions.go`, `internal/cli/slash.go`, `internal/cli/editor/external.go`, `internal/cli/editor/external_test.go`; `acpclient/sessions.go`, `acpclient/sessions_test.go`; `internal/cli/flags.go`; *(P4 deviation 12: `internal/cli/settings.go`; deviation 15: `acpserver/compact.go`, `internal/cli/mcp.go`, `internal/cli/chat.go`, `internal/cli/acp.go`, `internal/cli/config.go`, `internal/cli/config_test.go`; deviation 19: `internal/config/keys.go`, `internal/config/testdata/settings.schema.json`, `internal/config/resolve_test.go`, `internal/config/edit_test.go`; deviation 20: `internal/cli/main.go`, `internal/cli/slash_test.go`)* |
| P6 | — | `docs/architecture.md`, `README.md`, this plan (its execution record), `docs/README.md` |

A phase that needs a file not listed here stops, and the file is raised as a deviation.

### Out of scope

- **F6's trust store and `/trust`.** P2's trust function is a parameter that always answers "untrusted" until F6 (MADR D5).
- **F4, F5, F7 and F9's keys** (MADR D19), and `bridge.pi` (F10).
- **`/reload`** (MADR D16).
- **`--offline`'s binding to `GOBBLE_OFFLINE`** (X3, MADR D11).
- **Changes to `mcp.json`'s or `models.json`'s format** (MADR D2).
- **Anything in magic-cli-remote.**

## Stability rule

Every phase ends with:

```bash
export GOTOOLCHAIN=go1.27.2
GO_PRECHECK_SKIP_VULN=1 make pre-add-check FILES="<the phase's .go files>"
go test ./...
make preflight
```

and with 0005-PLAN F1d-2's deviation 4 gates:

- lint for linux, darwin and windows;
- Windows `go test ./...`;
- `CGO_ENABLED=1 go test -race ./...` natively on Windows with MinGW's gcc, and over SSH on the second amd64 Linux machine, on a copy made from a git bundle;
- ~~Windows and WSL `make preflight`, WSL's without its race run.~~ *(2026-10-12, during P2, the owner directed: "run linux race gates via ssh on [the second Linux machine] not on the WSL it is flaky". From P2 on, `make preflight` runs on Windows only, and no gate runs in WSL. Linux coverage is the race run over SSH, which also runs every test on Linux, and lint and staticcheck for `GOOS=linux`, which preflight already runs.)*

A crash in `internal/cli`'s tests on Windows is checked for Bitdefender's injected DLLs before it is debugged (the host's known cause). A race report in WSL is not a gobble defect while 0013-REPORT's third amendment holds.

**Commits.** The agent stages each phase with `git add` and never commits. The owner commits each phase with `git commit --no-edit` before the next begins. No push and no tag without the owner's request in that turn.

## Cross-cutting contracts

- **C1. No behaviour change without a settings file.**
  - Every existing test passes unedited, except:
    - `internal/cli/main_test.go`'s `TestConfigPath`, which gains the two new lines (P4);
    - `TestLaterFlagsAreRejected`'s `offline` case, whose message names X3 (P4, MADR D20).
  - Every default is the code's: a test that pins a default (MADR "Tests pin the defaults") is never edited to pass.
- **C2. gobble writes a settings file only through D8's editor, D9's `init` and `update`, and D9's first-run creation.**
  - Nothing writes on read. A read-only command creates nothing.
  - A file that does not parse is never written.
- **C3. No new module.** `go.mod` and `go.sum` do not change in any phase, and `go mod tidy` stays clean.
- **C4. Imports.**
  - Kong only in `internal/cli` (rule 9).
  - `go.yaml.in/yaml/v3` only in `internal/config` and `internal/frontmatter`.
  - `internal/config` imports no `acpserver`, `acpclient` or `internal/cli`.
- **C5. Every new check is seen failing first,** on a scratch copy in the session scratchpad, never by dirtying the tree, and the planted edit is asserted to have landed.
- **C6. The wire does not change.** `session.golden` is never regenerated.
- **C7. No secret is printed.** Diagnostics name paths and lines, never values (MADR D7, D15).

C1 is most at risk. Wiring a default through settings (P5) can shift it, by a zero value or a unit (milliseconds versus a `time.Duration`), and editing the pinning test to match would hide exactly that.

## Dependency and delivery order

P1 → P2 → P3 → P4 → P5 → P6, one commit each.

- P1 moves shared code with no behaviour change.
- P2 and P3 build `internal/config` with no consumer.
- P4 adds the commands and loading, still with no consumer.
- P5 wires the consumers.
- P6 is docs.

Each phase builds and passes on its own.

## Implementation Steps

### P1 — Shared foundations: the lock, the git finder, `mcp.json`'s writer, MCP stop timings (D8, D17, D18; closes F5's lock gap)

1. **`internal/filelock`** (new). Move `session/jsonl/lock.go`, `lock_unix.go`, `lock_windows.go` and `lock_other.go` into it, renaming the package `filelock`. The exported API:

   ```go
   // HeldError is TryLock's error when another process holds the lock. PID is the holder's, or 0 when unreadable.
   type HeldError struct{ PID int }
   func (e *HeldError) Error() string // "locked by pid N"
   // TryLock takes the exclusive lock on path (created 0600), writes this pid into it, and never waits.
   func TryLock(path string) (release func() error, err error)
   // Lock is TryLock retried while another process holds it: attempts tries, wait apart (Pi: 10 and 20 ms).
   func Lock(ctx context.Context, path string, attempts int, wait time.Duration) (release func() error, err error)
   ```

   - `TryLock` is today's `lock`, with errors unwrapped.
   - `Lock` returns the last `*HeldError` after the final attempt, and `ctx.Err()` if `ctx` ends first.
   - `doc.go` states the package (internal) and its use by `session/jsonl`, `mcpclient` and `internal/config`.
2. **`session/jsonl/store.go`** `:135` and `:332` call `filelock.TryLock`. Errors are wrapped as today, `fmt.Errorf("session: lock: %w", err)`, and `errors.AsType[*filelock.HeldError]` replaces `*lockHeld`, so `session.ErrLocked`'s text is unchanged. The four `session/jsonl/lock*.go` files are deleted.
3. **`internal/filelock/lock_test.go`:**
   - `TestTryLockHeld`: a second `TryLock` on the same path, through a second open in the same process, gets `*HeldError` with this process's pid. flock and `LockFileEx` both lock per open file, which is how `session/jsonl`'s `TestWriterLock` already tests a second writer (`session/jsonl/store_test.go:103-125`).
   - `TestLockRetries`: `Lock` succeeds after the holder releases within the attempts, and returns `*HeldError` when it does not.
   - `TestLockContext`: a cancelled `ctx` returns `context.Canceled`.
4. **`internal/fsx/git.go`** (new). Move `gitPaths` and `absUnder` from `prompt/context.go:121-160`, exported as `GitRepo(dir string) (repoDir, commonGitDir string, ok bool)`, and unexported `absUnder`, with their comments. `prompt/context.go:102` calls `fsx.GitRepo`.
5. **`internal/fsx/git_test.go`:** `TestGitRepo` covers:
   - a `.git` directory;
   - a worktree's `.git` file with `gitdir:` and `commondir`;
   - a `.git` directory without `HEAD` (not a repository);
   - a plain directory up to the root (none).

   `prompt`'s existing `TestWorktreeShadowing` and the other `context_test.go` tests pass unchanged.
6. **`mcpclient/file.go`:**
   - `Add` and `Remove` take `filelock.Lock(ctx, path+".lock", 10, 20*time.Millisecond)` around read-edit-write. `Add` and `Remove` gain no context parameter: they pass `context.Background()`, since the CLI command's context is not needed for a 200 ms bound.
   - ~~`writeFile` (`:177-198`) is replaced by `fsx.Workspace{}.WriteFile(path, data)`.~~ *(Deviation 1: `writeFile` is replaced by `fsx.WriteFileMode(path, data, 0o600)`, which keeps a new `mcp.json` at 0600.)*
   - The file's own temp-name pattern goes.
7. **`mcpclient/file_test.go`:** `TestAddRemoveKeepTheFile`, `TestAddCreatesTheFile` and `TestAddRefusesAMalformedFile` pass unchanged. A new `TestAddConcurrent` runs 8 goroutines each adding one server, and finds all 8 in the file.
8. **`mcpclient/conn.go`'s `Options`** gains:

   ```go
   // StopGrace is how long a stdio server has to exit once its stdin closes; zero is 500 ms (Pi's).
   StopGrace time.Duration
   // StopTimeout is how long after StopGrace's terminate before the tree is killed; zero is 2 s (Pi's).
   StopTimeout time.Duration
   ```

   `transport` (`:163-196`) passes them into `stdioTransport`, and `stopper` holds them. `proc.go`'s `stdinGrace` and `closeTimeout` become the defaults applied when a field is zero, and `Close` (`:80-94`) uses the stopper's values.
9. **`mcpclient/proc_test.go`:** the existing `:115` test passes unchanged. A new `TestStopGraceOption` sets `StopGrace: 50*time.Millisecond` on a server that ignores stdin close, and the tree is terminated within 50 ms plus a 1 s slack.

**P1's verification:**

```text
go test ./internal/filelock/... ./internal/fsx/... ./prompt/... ./session/... ./mcpclient/...  -> ok
go test ./...                                                                               -> ok, unchanged elsewhere
go vet ./...                                                                                -> ok
```

**P1's negatives,** on scratch copies:

- `Lock` made to try once fails `TestLockRetries`;
- `filelock.TryLock` skipping the flock fails `TestTryLockHeld`;
- `fsx.GitRepo` ignoring `commondir` fails `TestGitRepo`'s worktree case;
- `Add` without the lock fails `TestAddConcurrent` in at least one of 20 runs (`-count=20`);
- `stopper.Close` using the constant fails `TestStopGraceOption`.

### P1's deviations, 2026-10-12, decided by the owner

- **Deviation 1. `fsx`'s writer would loosen `mcp.json`'s mode.**
  - **Found:** step 6 would swap `mcpclient`'s writer for `fsx.Workspace{}.WriteFile`, which creates a new file 0644 (`internal/fsx/write.go:13-15`, `:61`). The old writer's `os.CreateTemp` creates it 0600, and `mcpclient/file_test.go:93-94` pins that. `mcp.json` can hold literal header and environment values, such as tokens. The step as written would loosen a new file's mode, and fail a test that pins a default (C1). Both the test and the modes are unchanged in HEAD; the conflict is the step's.
  - **Decided** ("fsx gains a mode"):
    - `internal/fsx/write.go` gains `WriteFileMode(abs string, data []byte, newMode fs.FileMode) error`, a write outside any workspace through `writeAtomic` with the new-file mode as a parameter. `Workspace.WriteFile` keeps 0644, so file tools are unchanged.
    - `mcpclient` writes `mcp.json` with 0600, and keeps today's mode while gaining the fsync and the Windows rename retry. P3's settings writes use the same function with 0644 (MADR D8).
    - `internal/fsx/workspace_test.go` gains `TestWriteFileModeNewAndExisting`: a new file takes the given mode, and an existing one keeps its own.
    - 0018-MADR D8 and D18 get a dated note naming the function.
    - Files added: `internal/fsx/write.go`, `internal/fsx/workspace_test.go`.

### P2 — `internal/config`: keys, reading, merging, trust, diagnostics, overrides, schema and template (D2–D7, D9's template, D10's parsing, D13–D15, D19's mechanism; closes F1–F4, F6–F11)

1. **`keys.go`** holds the one table every other file reads:

   ```go
   type Kind int // String, Bool, Int, Enum, Path, Command
   type Scope int // Any, GlobalOnly
   type Read int // Session, Process
   type Key struct {
       Path    string   // dotted, as D14
       Kind    Kind
       Default string   // as the template shows it: "off", "20000", "false", ""
       Enum    []string // for Enum
       Min     int64    // for Int: the least value
       Scope   Scope
       Read    Read
       Doc     string   // the schema description: the template's first prose sentence
   }
   var Keys = []Key{ /* the thirteen of D14, in the template's order */ }
   ```

   The entries are exactly D14's table: path, type, default, range, scope and read.
2. **`settings.go`:**
   - `Settings`, with nested structs, `yaml` tags (camelCase) and `json` tags of the same names;
   - `Defaults() Settings`, equal to today's code defaults: thinking `off`, queues `one-at-a-time`, `keepRecentTokens` 20000, bash `MaxTimeout` 600000 ms, `useClientTerminal` false, MCP 10000, 500 and 2000 ms, empty strings for the rest;
   - helper methods converting milliseconds to `time.Duration` (`MaxTimeout()`, `MCPStartupWait()`, `MCPStopGrace()`, `MCPStopTimeout()`).
3. **`read.go`:** ~~`Read(file string, data []byte) (*yaml.Node, []Diagnostic)`~~ *(Deviation 3: `Parse(file string, data []byte) (*yaml.Node, []Diagnostic)`, since step 1's type `Read` takes the name)*:
   - decodes with `yaml.NewDecoder`, reading the first document;
   - a second `Decode` that does not return `io.EOF` gives D7's second-document diagnostic at that document's line;
   - a decode error gives `<file>:<line>: <message>`, the line parsed from `yaml.v3`'s `line N:` prefix;
   - a non-mapping root gives D7's diagnostic;
   - an empty or comments-only file is an empty mapping;
   - CRLF input is accepted;
   - a file that fails yields a nil node, and the caller treats the layer as empty.
4. **`diag.go`:** `Diagnostic{File string; Line int; Path string; Severity Severity; Text string}`, and `String()` giving each D7 form exactly. `Severity` is `Error` or `Warning`. Unknown keys and global-only keys in a project are warnings; the rest are errors.
5. **`merge.go`:** ~~`validate(node, layer)`~~ *(Deviation 3: `leaves(file, root, layer)`)* walks the mapping against `Keys`:
   - wrong kind: D7's type diagnostic;
   - out of range or not in the enum: D7's rule diagnostic;
   - unknown path: D7's unknown diagnostic, and kept in the file;
   - a global-only key in a project layer: D7's diagnostic, and ignored;
   - a credential-shaped path (`secret.go`, D15): D7's error;
   - aliases resolved by `yaml.v3`.

   It returns the valid leaves with `Source{Layer, File, Line}`. ~~`merge(layers...)` applies D4~~ *(Deviation 3: `Resolve` applies each layer's leaves in turn, which is D4)*: mappings deep, scalars replace, `null` clears to the layer below. The merged leaves decode into `Settings` over `Defaults()`.
6. **`secret.go`:** `isCredentialPath(path string) bool`, true when the last segment, lowercased with `_` and `-` removed, ends in `key`, `token`, `secret` or `password`.
7. **`resolve.go`:**

   ```go
   type Options struct {
       ConfigDir string                // <config>; "" reads no global file
       Cwd       string                // the session's or the process's working directory; "" reads no project file
       Read      Read                  // which keys apply: Session or Process
       Trusted   func(root string) bool // nil: nothing is trusted (until F6)
       Overrides []Override            // from --set
   }
   type Resolved struct {
       Settings Settings
       Origins  map[string]Source      // per key path: the layer that set it
       Disabled []DisabledLayer        // {File, Reason}
   }
   func Resolve(o Options) (Resolved, []Diagnostic)
   func GlobalFile(configDir string) string                // <config>/settings.yaml
   func ProjectFile(cwd string) (path string, ok bool)     // <root>/.gobble/settings.yaml, root by fsx.GitRepo or cwd
   ```

   - **Layers:** defaults, then global, then project (when `Trusted(root)`; otherwise read, validated and listed in `Disabled` with D7's reason), then overrides.
   - **Read filtering.** Only keys whose `Read` matches `o.Read` are applied. *(Deviation 3: the constants are `ReadSession` and `ReadProcess`, and a zero `Read` applies both, for P4's `config get` and `config validate`.)* For `Read == Process` under `gobble acp`, the caller passes `Cwd == ""`, so no project file is read (MADR D16).
8. **`override.go`:** `ParseOverride(s string) (Override, error)` splits at the first `=`.
   - The path must be in `Keys`, else `unknown setting "<path>"`.
   - The value is decoded as one YAML flow node, then checked like a file value. A failure gives `--set <path>: <rule>`.
9. **`schema.go`:** `Schema() ([]byte, error)`:
   - `jsonschema.For[Settings]` (`github.com/google/jsonschema-go`), then each property's `Description`, `Default`, `Enum` and `Minimum` from `Keys`;
   - an `x-gobble-global-only: true` extension on global-only keys;
   - `additionalProperties: false`;
   - indented, with a final newline.
10. **`genschema/main.go`** writes `Schema()` to `internal/config/testdata/settings.schema.json`. `testdata/settings.schema.json` is generated by `go run ./internal/config/genschema` and committed.
11. **`settings.yaml`** is the MADR D9 template text, byte for byte, with LF endings and the two-space indentation removed. **`template.go`:** `//go:embed settings.yaml` as `Template() []byte`.
12. **`doc.go`** replaces the placeholder text with the package's job (MADR D18). `internal/frontmatter/frontmatter.go`'s package comment reads "one of two packages that import `go.yaml.in/yaml/v3`, with `internal/config`".
13. **Tests.**
    - **`read_test.go`**, ~~over `testdata/`~~ *(deviation 4: over fixtures built in the test code)*:
      - `TestReadEmptyAndComments`;
      - `TestReadSecondDocument` ~~(`two-docs.yaml`, line 3)~~ *(deviation 4: at line 2, the `---` line where the second document starts)*;
      - `TestReadDuplicateKey`;
      - `TestReadTab`;
      - `TestReadSyntax`;
      - `TestReadNonMapping`;
      - `TestReadCRLF`.
    - **`resolve_test.go`:**
      - `TestDefaultsEqualCode`: `Resolve` with no files equals `Defaults()`, and each default equals its consumer's constant: `compaction.DefaultSettings.KeepRecentTokens`, `acpserver`'s `DefaultStartupWait` through a literal, the `mcpclient` constants, and `builtin`'s 10 minutes; *(deviation 5: the last three through literals too, being unexported, and each consumer's own test pins its constant to the same literal: `TestStopDefaults` in `mcpclient/proc_test.go`, `TestDefaultMaxTimeout` in `tool/builtin/shell_test.go`, `TestDefaultStartupWait` in `acpserver/mcp_test.go`)*
      - `TestLayersAndPrecedence`;
      - `TestNullClears`;
      - `TestProjectDisabledUntilTrusted` (exact reason text);
      - `TestProjectTrustedApplies` (a `Trusted` stub returning true);
      - `TestGlobalOnlyIgnoredFromProject`;
      - `TestTypeAndRangeErrorsFallBack`;
      - `TestUnknownKeyReported`;
      - `TestCredentialPathRefused`;
      - `TestReadFiltering`: a process key is not applied with `Read == Session`, and the converse;
      - `TestOriginsAndLines`.
    - **`override_test.go`:** valid, unknown path, bad value, `=` in the value, and a flow list.
    - **`schema_test.go`:** `TestSchemaCommitted` compares `Schema()` with the committed file. On a difference it fails with `the committed schema is stale: run go run ./internal/config/genschema`.
    - **`template_test.go`**, D9's six tests:
      - `TestTemplateParsesAsComments`;
      - `TestTemplateHasEveryKeyOnce`;
      - `TestTemplateDefaultsMatch`;
      - `TestTemplateUncommentedIsDefaults`;
      - `TestTemplateHasNoUnknownKey`;
      - `TestTemplateProseIsNotAKey`.

    The template's key lines are found by D8 case 2's line shape.

**P2's verification:**

```text
go test ./internal/config/...                     -> ok
go run ./internal/config/genschema && git diff --exit-code internal/config/testdata/settings.schema.json -> no diff
go list -deps ./internal/config | grep -c kong     -> 0
go mod tidy && git diff --exit-code go.mod go.sum -> no diff
```

**P2's negatives,** on scratch copies, each failing its test:

- the second-document check removed;
- the trust gate inverted;
- a global-only key let through;
- the secret check removed;
- one byte changed in the committed schema;
- a template block deleted;
- a template default changed to `20001`;
- a prose line reworded as `name: value`.

### P2's deviations, 2026-10-12, decided by the owner

- **Deviation 2. `genschema` is a second `main`, and lint forbids its streams.**
  - **Found:** step 10's `genschema/main.go` reports a failure on `os.Stderr` and exits with `os.Exit`. `.golangci.yml:63-64` forbids both through forbidigo (0008-MADR D18) and exempts only `cmd/gobble/main.go` (`:140-142`), so lint fails on linux, darwin and windows with four findings. The rule exists so the shipped binary writes only through `cli.Main`'s injected writers; `genschema` is a generator a developer runs, never shipped. The conflict is the step's: the lint configuration is unchanged in HEAD.
  - **Decided** ("exempt it in lint"): `.golangci.yml` gains a forbidigo exclusion for `internal/config/genschema/main\.go`, beside `cmd/gobble/main.go`'s. `genschema` keeps its command and its message. File added: `.golangci.yml`.
- **Deviation 3. The plan's names collide, or name helpers the code does not have.**
  - **Found:** step 1 defines the type `Read`, and step 3 the function `Read`, in one package, which does not compile. Step 5 names `validate` and `merge`; the walk is `leaves`, and the layering is `Resolve`'s, with the same behaviour. P4 (step 2) writes `config.Session` and `config.Process`, and `config get` and `config validate` need every key, which neither value gives.
  - **Decided** ("amend the plan to the code"): the function is `Parse`, the walk `leaves`, the constants `ReadSession` and `ReadProcess`, and a zero `Read` applies every key. Steps 3, 5 and 7, P3's step 1 and P4's step 2 are annotated. No file added.
- **Deviation 4. The read fixtures are built in the tests.**
  - **Found:** step 13 reads its cases from `testdata/` files. `.gitattributes` (`* text=auto eol=lf`, 0007-MADR D3) makes a committed CRLF fixture impossible, and the other cases are a line or two each. `two-docs.yaml` "line 3" assumed a fixture with a line before the document; the diagnostic names the `---` line, where D7 says "a second one starts here".
  - **Decided** ("inline, amend the plan"): every fixture is built in the test code, and `testdata/*.yaml` leaves P2's scope. `TestReadSecondDocument` expects line 2 of `steeringMode: all\n---\nfollowUpMode: all\n`. No file added.
- **Deviation 5. Three of the four defaults were pinned on one side only.**
  - **Found:** `TestDefaultsEqualCode` is to compare with "the `mcpclient` constants, and `builtin`'s 10 minutes". They are unexported (`mcpclient/proc.go:20-21`, `tool/builtin/builtin.go:56`), so `internal/config` can hold only literals. No test pins their values; they are used only as bounds (`mcpclient/proc_test.go:110`, `:139`) and as an argument (`tool/builtin/shell_test.go:77`). Nor is `acpserver.DefaultStartupWait` pinned by any test. A change to any one would leave `config.Defaults()` disagreeing with the code, with every test passing, against C1.
  - **Decided** ("pin both sides"): each consumer's test pins its constant to the literal `internal/config` holds, and names the key: `TestStopDefaults` (`stdinGrace` 500 ms, `closeTimeout` 2 s), `TestDefaultMaxTimeout` (10 minutes) and `TestDefaultStartupWait` (10 s). A change on either side then fails a test that names the other. Files added: `mcpclient/proc_test.go`, `tool/builtin/shell_test.go`, `acpserver/mcp_test.go`.

### P3 — The editor and `update` (D8, D9's growth; closes F2, F3, F5)

1. **`edit.go`:**

   ```go
   type Edit struct {
       Path   string     // a key in Keys
       Value  *yaml.Node // nil for a removal
   }
   // Apply returns data with e applied, or an error naming why it could not be applied safely. It never touches a file.
   func Apply(file string, data []byte, e Edit) ([]byte, error)
   // Set and Unset lock <file>.lock, read, Apply, check, and write through fsx.Workspace{}.WriteFile.
   // (P1 deviation 1: through fsx.WriteFileMode(file, data, 0o644).)
   func Set(file, path, value string) error
   func Unset(file, path string) error
   ```

   `Apply` implements D8's five cases, on the file's lines with their own endings:
   1. **An active scalar.** Find the value node by path. Its raw extent starts at `Node.Line` and `Node.Column`:
      - for a plain scalar, it runs to the start of an inline comment (a space, then `#`) or the line's end, without trailing spaces;
      - for a quoted scalar, it runs to its closing quote, honouring escapes.

      Replace that extent with `render(value)`. A literal or folded scalar is handled as case 4.
   2. **A commented-out key.** Find a line whose text after one `#` and one space is the key at its nesting indentation (two spaces per level) followed by `:`. Every parent must be either active, with the line inside its extent, or commented in the same form directly above it within the same comment block. Remove the `#` and the one space after it from that line and from those commented parents, then apply case 1 to it. If no such line exists, use case 3.
   3. **An absent key.** Insert `<indent><key>: <render(value)>` after `lastLine(parent)`. The indentation is the parent's children's, or the parent's plus two. Missing parents are inserted the same way, in path order.
   4. **A mapping, sequence, or block scalar value.** Replace lines `Node.Line` to `lastLine(node)` with block-style YAML at the key's indentation plus two spaces.
   5. **A removal.**
      - A key in `Keys`: replace its lines with the template's commented form, taken from the embedded template at the key's indentation.
      - Otherwise: delete its lines.
      - *(Deviation 7: a parent left with no active key is then commented too, in the same form, and so on upward, so that unsetting and setting a key again round-trips through case 2. A parent that is not a template group, and is left empty, is deleted.)*

      The comment lines above it stay.

   The helpers:
   - `lastLine(n)` is the largest line among `n` and its descendants, plus the line count of any multi-line scalar.
   - `render(v)` writes a scalar plain only when `yaml.v3` re-reads the plain text as the same tag and value; otherwise double-quoted. *(Named `renderScalar` since P4, where revive's confusing-naming rule refuses it beside P4's exported `Render`.)*
   - An aliased value, or a path through an alias, is refused with D7's alias diagnostic.

   **The check.** After `Apply`, the result is passed to ~~`Read`~~ `Parse` *(P2 deviation 3)*. It must parse as one mapping document, the path must re-read as the value (or as absent), and every other leaf path must re-read equal, compared by tag and value. *(Deviation 8: except an ancestor of the path that was `key:` with nothing after it, which may become the mapping that holds the path, as case 3 makes it. An explicit `~` or `null` stays a value the check protects.)* Anything else returns D7's "could not be applied safely" error, and the file is untouched.
2. **`update.go`:**

   ```go
   // Init writes Template() to file; an existing file is an error.
   func Init(file string) error
   // Update adds the template's blocks for keys the file lacks, without changing an existing line.
   func Update(file string, dryRun bool) (added []string, preview []byte, err error)
   ```

   - **Blocks.** A template block is a run of non-blank lines: its prose and its commented key lines. Sections are the ruler-and-heading comment lines.
   - **A block none of whose keys appears in the file,** active or commented: insert it after the last line of the nearest earlier block the file has, or after its section heading, with one blank line around it. If the file lacks the section, append the section heading and the block at the end.
   - **A block that is partly present:** insert each missing key's commented line after its nearest earlier sibling's line, at its indentation, without prose.
   - **`dryRun`** returns the added lines as `preview`, a `+`-prefixed listing, and writes nothing.
   - `Update` takes the lock and writes like `Set`, with its own check that every line of the old file is still present, in order.
3. **`edit_test.go`**, each a byte-diff against an expected file in `testdata/edit/`:
   - `TestEditActiveScalar`, keeping an inline comment and the spaces before it;
   - `TestEditCommentedKey` (top-level and nested);
   - `TestEditAbsentKey` (inserting parents);
   - `TestEditSequence`;
   - `TestEditBlockScalar`;
   - `TestUnsetTemplateKey`;
   - `TestUnsetUnknownKey`;
   - `TestEditQuotesAmbiguous` (`1.10` and `yes` into a string key become `"1.10"` and `"yes"`);
   - `TestEditCRLFKept` *(deviation 6: its CRLF input and expected bytes are built in the test from an LF fixture)*;
   - *(deviation 7: `TestUnsetLastChildCommentsParent`, and an unset-then-set round trip back to the original bytes)*;
   - `TestEditAliasRefused`;
   - `TestEditUnparsableRefused` (the file is unchanged, by bytes and by modification time);
   - `TestEditCheckRefusesBadSplice`, through a test-only hook that corrupts the splice, so the check returns its error;
   - `TestSetLocksAndWrites`.
4. **`update_test.go`:**
   - `TestUpdateAddsMissingBlocks`;
   - `TestUpdatePartialBlock`;
   - `TestUpdateMissingSection`;
   - `TestUpdateDryRunWritesNothing`;
   - `TestUpdateKeepsEveryLine` (old lines kept in order);
   - `TestInitRefusesExisting`.

**P3's verification:**

```text
go test ./internal/config/... -run 'TestEdit|TestUnset|TestUpdate|TestInit|TestSet' -> ok
go test ./internal/config/...                                                      -> ok
```

**P3's negatives,** on scratch copies:

- case 1 replacing the whole line, losing the inline comment, fails `TestEditActiveScalar`;
- `render` always plain fails `TestEditQuotesAmbiguous`;
- the check skipped makes `TestEditCheckRefusesBadSplice` write the corrupt file and fail;
- `Update` changing an existing line fails `TestUpdateKeepsEveryLine`;
- line endings normalised to LF fail `TestEditCRLFKept`.

### P3's deviations, 2026-10-12, decided by the owner

- **Deviation 6. A CRLF fixture cannot be a committed file.**
  - **Found:** step 3 makes each edit test a byte-diff against a file in `testdata/edit/`. `.gitattributes` (`* text=auto eol=lf`, 0007-MADR D3) turns a CRLF fixture and its expected file into LF on checkout, so `TestEditCRLFKept` would pass while testing LF. This is the cause of P2's deviation 4.
  - **Decided** ("CRLF case in code"): `TestEditCRLFKept` builds its input and expected bytes from an LF fixture, with each `\n` made `\r\n`. Every other edit test keeps its `testdata/edit/` files. No file added.
- **Deviation 7. Unsetting a parent's last active key left a parent the editor could not reuse.**
  - **Found:** case 5 as written turns only the key's line back into its template form. Unsetting `mcp.stopGraceMs` from `mcp:\n  stopGraceMs: 100\n` leaves `mcp:\n#   stopGraceMs: 500\n`. The empty `mcp:` is null, which D4 reads as absent, so the meaning is safe, but case 2 then cannot use the commented line: a null parent has no extent for it to sit inside. A later set falls to case 3 and inserts a second, active line beside the stale commented one, and a later case-2 search finds two candidates.
  - **Decided** ("comment the parent too"): after a removal, a parent left with no active key is commented in the template's form too, and so on upward. A parent that is not a template group, and is left empty, is deleted. Unsetting and then setting a key restores the file's bytes. Tests added: `TestUnsetLastChildCommentsParent`, and the round trip in `TestUnsetTemplateKey`. No file added.
- **Deviation 8. D8's check refused the insertion D8's case 3 describes.**
  - **Found:** case 3 inserts a key "two spaces deeper than the parent when it has none". In block YAML a parent with no children is `mcp:` with nothing after it, which reads as null; the insertion makes it a mapping. The check requires every other path to read back unchanged, so `set mcp.stopGraceMs 5` on `mcp:` was refused (`testdata/edit/empty.in.yaml`, found by the test added for case 1's empty value). The code is new; the conflict is between two sentences of MADR D8.
  - **Decided** ("empty parent only"): the check treats an ancestor of the edited path that was `key:` with nothing after it as part of the edit; it may become the mapping that holds the path. An explicit `~` or `null`, which D4 reads as clearing the layer below, is a value the user wrote, and the check still refuses to replace it. Tests: the `empty` case of `TestEditActiveScalar`, and `TestEditExplicitNullParentRefused`. No file added.

### P4 — The CLI: loading, first-run creation, `--set` and the `config` commands (D1, D9's creation, D10–D12, D20's code comment; closes F1, F9, F12)

1. **`internal/cli/root.go`:** `Root` gains

   ```go
   Set []string `name:"set" sep:"none" complete:"none" placeholder:"PATH=VALUE" help:"Override a setting for this run; repeatable."`
   ```

   and `Config ConfigCmd` keeps its place.
2. **`internal/cli/settings.go`** (new):
   - `runEnv` gains `overrides []config.Override`, parsed in `main.go`'s `run` from `root.Set` before the command runs. A parse error is a usage error, exit 2, with `ParseOverride`'s text.
   - *(Deviation 12: the next two methods are written in P5, step 4, with their callers; in P4 they would have none, and the `unused` linter fails them.)*
   - `runEnv.processSettings(cwd string) config.Resolved` is resolved once with ~~`Read: config.Process`~~ `Read: config.ReadProcess` *(P2 deviation 3)*, cached, and its diagnostics printed through `e.out.Warnf` once. Under `gobble acp` it goes to the log through `e.log()` (MADR D7).
   - `runEnv.sessionSettings(cwd string) config.Resolved` resolves with ~~`Read: config.Session`~~ `Read: config.ReadSession` *(P2 deviation 3)* on every call. Its diagnostics go to the log.
   - `runEnv.ensureSettingsFile()` writes `config.Template()` to `config.GlobalFile(dirs.Config)` when the file is missing, through `config.Init`. A failure is a warning, and the run continues.
3. **`chat.go`, `acp.go` and `mcp.go`'s command `Run`s** call `e.ensureSettingsFile()` first: the chat command, `ACPCmd`, and each `MCP…Cmd`. `VersionCmd`, `CompletionCmd` and `ConfigPathCmd` do not. Completion mode returns before `run` already (`internal/cli/main.go:37-42`).
4. **`internal/cli/config.go`** (new) holds `ConfigCmd`'s subcommands, D12's:
   - `Get{Path string arg optional; Origin bool; JSON bool}`;
   - `Set{Path, Value string args; Project bool}`;
   - `Unset{Path string arg; Project bool}`;
   - `Edit{Project bool}`;
   - `Init{Project bool}`;
   - `Update{Project bool; DryRun bool}`;
   - `Validate{Strict bool}`;
   - `Schema{}`.

   Every value-taking argument has a `complete:` tag (0008-MADR D17): `complete:"setting"` names a new completion source. It is registered in `internal/cli/completion.go`'s `completionRegistry` and lists `config.Keys`' paths. The exits are D12's: usage errors 2, write failures 1.
   - **`get --json`** *(deviation 11)*: without `--origin`, the effective `Settings` object, or the one path's value. With `--origin`, `{"settings": …}` or `{"value": …}`, beside `"origins": {path: {layer, file, line}}` and `"disabled": [{file, reason, keys}]`.
   - **A write to a missing file** *(deviation 10)*: the global file is created from the template first (D9). A missing project file is created by `init --project` and `edit --project` only; `set`, `unset` and `update` with `--project` exit 1, naming `gobble config init --project`.
   - **`get`:** without `--json`, output is YAML, from a new `config.Render(r Resolved, path string, origin bool) []byte` in `internal/config/render.go`. `internal/cli` must not import `yaml.v3` (C4). `render_test.go` checks the YAML and the `--origin` annotations, written as `# from <file>:<line> (<layer>)` comments.
   - **`--project`** uses `config.ProjectFile(cwd)` from the working directory. Outside a repository, the project root is the working directory.
   - **`edit`** runs D17's editor chain on the file, then prints `Validate`'s diagnostics. *(Deviation 9: through a new `editor.Open(ctx, env, file, run) error` in `internal/cli/editor/external.go`, called through a package variable the test stubs, as `/edit` stubs `compose`.)*
5. **`internal/cli/configpath.go`:**
   - `ConfigCmd`'s comment cites 0018-MADR, and it gains the subcommands of step 4.
   - `ConfigPathCmd` prints today's four lines unchanged, then `settings: <config>/settings.yaml` and `project:  <path>`, or `project:  none` when the working directory gives none.
   - `--json` gains `"settings"` and `"project"`.
   - It still creates nothing.
6. **`internal/cli/flags.go:67`:** `offline` maps to `plan0005X3 = "0005-PLAN X3"` (MADR D20), and `TestLaterFlagsAreRejected`'s expectation for it follows.
7. **`internal/cli/config_test.go`** (new):
   - `TestConfigGet`, `TestConfigGetOrigin`, `TestConfigGetJSON`;
   - `TestConfigSetUnset`, `TestConfigSetProject`, `TestConfigSetGlobalOnlyProjectRefused`;
   - `TestConfigSetUnknownExit2`, `TestConfigSetBadValueExit2`;
   - `TestConfigEdit`, through the slash tests' editor stub;
   - `TestConfigInit`, `TestConfigInitExistsExit1`;
   - `TestConfigUpdateDryRun`;
   - `TestConfigValidateExits` (0, 1, and `--strict`);
   - `TestConfigSchema`;
   - `TestSetFlagUsageError`;
   - `TestFirstRunCreatesTemplate` (chat print mode with a scripted agent, and `acp` with empty stdin);
   - `TestReadOnlyCommandsCreateNothing` (`version`, `config path`, `config get`, `config validate`, `config schema`, each in a fresh `GOBBLE_HOME`).
8. **`internal/cli/main_test.go`:** `TestConfigPath`'s `want` gains the two lines (C1's first exception), and its "creates nothing" check stays.

**P4's verification:**

```text
go test ./internal/cli/... -run 'TestConfig|TestSetFlag|TestFirstRun|TestReadOnly|TestLaterFlags' -> ok
go test ./internal/cli/...                                                                      -> ok
go run ./cmd/gobble config path; go run ./cmd/gobble config validate                           -> four lines plus two; exit 0
```

**P4's negatives,** on scratch copies:

- `ensureSettingsFile` called from `ConfigPathCmd` fails `TestReadOnlyCommandsCreateNothing`;
- `--set` errors ignored fail `TestSetFlagUsageError`;
- `--project` writing a global-only key fails `TestConfigSetGlobalOnlyProjectRefused`;
- `validate --strict` exiting 0 on an unknown key fails `TestConfigValidateExits`.

### P4's deviations, 2026-10-12, decided by the owner

- **Deviation 9. The editor chain could not open a named file.**
  - **Found:** `internal/cli/editor` exports only `Compose` (`external.go:19`), which writes a temp `gobble-prompt-*.md`, runs the editor and returns the text with its trailing newline trimmed; `editorCommand` (`:48`) is unexported. MADR D12 has `config edit` open the settings file itself. `external.go` is in P5's file list, not P4's.
  - **Decided** ("add editor.Open"): `external.go` gains `Open(ctx, env, file, run) error`, which runs `editorCommand`'s editor on the named file without a shell. `config edit` calls it through a package variable, which `TestConfigEdit` stubs. `external_test.go` gains `TestOpenRunsEditorOnFile`. Files added: `internal/cli/editor/external.go`, `internal/cli/editor/external_test.go`. P5 still adds `externalEditor` to the same chain.
- **Deviation 10. MADR D9 and D12 disagreed on who creates a project file.**
  - **Found:** D9 says a project file is created only by `gobble config init --project`; D12 says `gobble config edit [--project]` creates the file from the template if missing.
  - **Decided** ("edit --project creates too"): `init --project` and `edit --project` create a missing project file, both explicit requests. `set`, `unset` and `update` with `--project` on a missing project file exit 1, naming `gobble config init --project`. MADR D9 gets a dated note. No file added.
- **Deviation 11. `config get --json` had no defined shape with `--origin`.**
  - **Found:** D12 says `--origin` adds each value's file, line and layer, and the disabled layers, without a JSON form.
  - **Decided** ("object, wrapped with --origin"): step 4's `get --json` form. `TestConfigGetJSON` checks both. No file added.
- **Deviation 12. Step 2's two resolvers had no caller in P4.**
  - **Found:** `runEnv.processSettings` and `runEnv.sessionSettings` are called only by P5's consumers. The `unused` linter (`.golangci.yml:25`) fails an unexported method with no caller, so P4's lint gate could not pass as written.
  - **Decided** ("move both to P5"): P4's `settings.go` holds `overrides` and `ensureSettingsFile`; the two resolvers are written in P5's step 4, with their callers. P4 has no settings consumer, as planned. File added to P5: `internal/cli/settings.go`.
- **Deviation 13. `internal/config` gave the CLI no way to check a value, or to recognise an unknown key.**
  - **Found:** D12 has a bad value exit 2 and a write failure 1, checked before anything is written. `config.Set` returns one plain error for both; `valueNode` is unexported (`internal/config/edit.go`). `validate --strict` must fail on an unknown key, whose text is an unexported constant (`internal/config/diag.go:40`), as is the global-only text (`:41`) that `--project` must refuse with.
  - **Decided** ("in edit.go and diag.go"): `edit.go` gains `Check(path, value string, layer Layer) error`: an unknown path, a credential path, a global-only key for `LayerProject`, or a bad value, in D7's words. `Set` keeps its own checks. `diag.go` gains `Diagnostic.Unknown() bool`. Tests: `TestCheck` in `edit_test.go`, `TestDiagnosticUnknown` in `resolve_test.go`. Files added: `internal/config/edit.go`, `internal/config/diag.go`, `internal/config/edit_test.go`, `internal/config/resolve_test.go`.
- **Deviation 14. `config edit`'s exit status was not defined.**
  - **Found:** D12 has `config edit` validate the file after the editor exits and print the diagnostics, without an exit status.
  - **Decided** ("1 on an error, as validate"): `config edit` exits 1 when the edited file has an error diagnostic, and 0 otherwise; the file stays as the user saved it. `TestConfigEdit` checks both. No file added.

### P5 — The consumers and `--add-dir` (D14, D16, D17; closes F11)

1. **`acpserver/config.go`** gains:

   ```go
   // SessionSettings are the settings a session takes when it starts (0018-MADR D16).
   type SessionSettings struct {
       SteeringMode, FollowUpMode string
       Thinking                   string
       Compaction                 compaction.Settings
       MCPStartupWait, MCPStopGrace, MCPStopTimeout time.Duration
   }
   ```

   `acpserver.Options` gains `Settings func(cwd string) SessionSettings`. Nil gives `defaultSessionSettings()`, equal to today's values. Any zero field of a returned value is filled from that default, so a partial value never mixes defaults (D17).
2. **`acpserver/agent.go`:**
   - `NewSession` (`:244`) and `open` (`:706`) call `a.sessionSettings(cwd)` once.
   - ~~`newQueue` takes `SteeringMode` and `FollowUpMode` (`acpserver/queue.go:26-28`).~~ *(Deviation 18: `newQueue()` is unchanged; the session's queue takes `SteeringMode` and `FollowUpMode` through the queue's own `setSteeringMode` and `setFollowUpMode`.)*
   - `NewSession`'s `think:` takes `Thinking` (`:287`). `open` keeps `thinkingOff` before `restore`, which sets the history's level (`:731`).
   - `liveSession` keeps its `compaction.Settings` and MCP durations, used by `compactSession` (`acpserver/compact.go:65`) and by `startMCP` and `mcpTools` (`acpserver/mcp.go:123`, `:133-146`), in place of `a.compaction` and `a.mcp.StartupWait`. `Options.Compaction` and `MCPOptions.StartupWait` stay as the fallbacks, so the SDK's callers see no change.
3. **`acpserver/settings_test.go`** (new):
   - `TestSessionSettingsApplied`: a stub `Settings` returning `all`, `all`, `low`, `keepRecentTokens` 100 and a 1 ms start-up wait. A new session's `_gobble/get_state` reports the queue modes and thinking. `/compact`'s cut honours 100, through `compaction.Prepare`. The MCP wait is shortened, by a server that never answers.
   - `TestNilSettingsIsToday`: a nil function gives today's defaults.
   - `TestResumedKeepsHistoryThinking`.
4. **`internal/cli/agent.go`:**
   - *(P4 deviation 12: `internal/cli/settings.go` gains `runEnv.processSettings` and `runEnv.sessionSettings` here, as P4's step 2 describes them.)*
   - `agentOptions` sets `Settings: func(cwd string) acpserver.SessionSettings { return toSession(e.sessionSettings(cwd)) }`.
   - `builtinTools` (`:200-207`) sets `ShellPath` (with `~` expanded), `ShellCommandPrefix`, `MaxTimeout` and `UseClientTerminal` from `e.processSettings(cwd)`, beside `OutputDir`. *(Deviation 15: `toolSet` takes the built-in tools as an argument, and `chat.go` resolves the working directory before the tool flags, so they can take `processSettings(cwd)`. Deviation 16: `acp.go` sets the agent's tools from the process settings with no project file (D16), keeping `builtin`'s own `OutputDir` default, so `gobble acp` without a settings file behaves as today.)*
   - `internal/cli/mcp.go`'s `list` (`:231`) passes the process settings' `StopGrace` and `StopTimeout`.
5. **`internal/cli/sessions.go`'s `configureStore`** (`:39-47`): `sessionDir`, when set and `--session-dir` is not, gives the same flat `jsonl.New(dir, jsonl.Options{Flat: true, …})`. `--session-dir` wins. `--no-session` still wins over both. *(Deviation 21: in `internal/cli/agent.go`'s `sessionStore`, which MADR D17 also cites, so `gobble acp` honours it too; `configureStore` still sets the flag's store first, so the flag wins.)*
6. **`internal/cli/editor/external.go`:** `editorCommand(env, goos, configured string)` tries `GOBBLE_EDITOR`, then `configured`, then `VISUAL` and `EDITOR`, then `notepad` or `vi`. `Compose` gains a `configured string` parameter. `internal/cli/slash.go:28`'s `compose` and its call at `:88` pass `e.processSettings(cwd).Settings.ExternalEditor`. `external_test.go` adds `TestEditorChainConfigured`. *(Deviation 17: P4's `editor.Open` gains the same `configured` parameter, and `config edit` passes the process settings' `externalEditor`, as the template says it does.)*
7. **`--add-dir`:**
   - `SharedFlags` gains

     ```go
     AddDir []string `name:"add-dir" sep:"none" complete:"dir" placeholder:"DIR" help:"Another workspace directory for this run; repeatable."`
     ```

     Each value is made absolute against `--cwd` or the working directory, and must be a directory, else a usage error.
   - `acpclient.NewSessionOptions` gains `AdditionalDirectories []string`, sent by `NewSessionWith`.
   - `acpclient` gains `ResumeSessionOptions{AdditionalDirectories []string}` and `Conn.ResumeSessionWith(ctx, id, cwd string, opts ResumeSessionOptions, on func(Update))`. `ResumeSession` calls it with zero options.
   - `internal/cli/sessions.go:61`, `:84` and `internal/cli/slash.go:160` pass the directories.
   - Tests:
     - `acpclient/sessions_test.go` adds `TestAdditionalDirectoriesSent`, ~~over `acptest`'s recorder~~ *(deviation 22: recorded by a test agent)*, for new and resume;
     - `internal/cli/settings_test.go` adds `TestAddDirAbsoluteAndChecked`.
8. **`internal/cli/settings_test.go`** (new):
   - `TestProcessSettingsReachTools`: `shellCommandPrefix` and `tools.bash.maxTimeoutMs` in a temp global file reach `builtin.Options`, through a test hook on `builtinTools`.
   - `TestSessionDirSetting`.
   - `TestSetBeatsFile`.
   - `TestFlagBeatsSet`: `--session-dir` over `--set sessionDir=…`.
   - `TestACPIgnoresProjectProcessKeys`.

**P5's verification:**

```text
go test ./acpserver/... ./acpclient/... ./internal/cli/... ./mcpclient/... -> ok
go test ./acpserver -run '^TestSession$' && git diff --exit-code acpserver/testdata/session.golden -> ok, no diff
go test ./...                                                              -> ok; every default-pinning test unedited (C1)
```

**P5's negatives,** on scratch copies:

- ~~`newQueue` ignoring its arguments~~ *(deviation 18: the session's queue modes not set)* fails `TestSessionSettingsApplied`;
- `Settings` called at each prompt rather than at session start fails it too, by a counting stub;
- `sessionDir` beating `--session-dir` fails `TestFlagBeatsSet`;
- `editorCommand` ignoring `configured` fails `TestEditorChainConfigured`;
- `--add-dir` not sent on resume fails `TestAdditionalDirectoriesSent`.

### P5's deviations, 2026-10-12, decided by the owner

- **Deviation 15. P5's steps change six files its list leaves out.**
  - **Found:** step 2 has `compactSession` (`acpserver/compact.go:65`) take the session's compaction settings; step 4 has `mcp list` (`internal/cli/mcp.go:234`) pass the stop timings. `builtinTools` takes no settings, and its only callers are `SharedFlags.toolSet` (`internal/cli/agent.go:169`, `:172`), which `chat.go` calls before it resolves the working directory (`:28-32`). `gobble acp` never calls it: `acpserver` falls back to `builtin.Tools()` with zero options, so its sessions would ignore the bash keys, which `TestACPIgnoresProjectProcessKeys` presumes apply. Deviation 17 needs `config.go` and `config_test.go`.
  - **Decided** ("add all six"): `toolSet` takes the built-in tools as an argument; `chat.go` resolves the working directory before the tool flags; `acp.go` sets the agent's tools (deviation 16). Files added: `acpserver/compact.go`, `internal/cli/mcp.go`, `internal/cli/chat.go`, `internal/cli/acp.go`, `internal/cli/config.go`, `internal/cli/config_test.go`.
- **Deviation 16. Building acp's tools through `builtinTools` would move its output directory.**
  - **Found:** `builtinTools` sets `OutputDir` to `<state>/output`; `gobble acp` uses `builtin`'s default, the OS temp directory's `gobble-output` (`tool/builtin/builtin.go:62-63`). Moving it changes acp without a settings file, against C1.
  - **Decided** ("keep acp's output dir"): acp's tools take the process settings and keep `builtin`'s own `OutputDir` default. Aligning the two directories is its own change, not this record's. No file added.
- **Deviation 17. `externalEditor` did not reach `config edit`.**
  - **Found:** the template (MADR D9) says `externalEditor` is the editor `/edit` and `gobble config edit` open; step 6 threads it into `Compose` only, not P4's `editor.Open` (deviation 9).
  - **Decided** ("Open takes it too"): `Open` gains `configured`, as `Compose` does, and `config edit` passes the process settings' `externalEditor`. `TestOpenRunsEditorOnFile` and `TestConfigEdit`'s stub follow. Files: in deviation 15.
- **Deviation 18. `newQueue`'s new signature broke tests outside the list.**
  - **Found:** step 2 gives `newQueue` the two modes; `acpserver/extensions_test.go:588` and `acpserver/queue_test.go:21`, `:51` call `newQueue()`. The queue already has `setSteeringMode` and `setFollowUpMode` (`acpserver/queue.go:60-70`), which `_gobble/set_queue_mode` uses.
  - **Decided** ("keep newQueue(), set modes"): `newQueue()` is unchanged; `NewSession` and `open` build the session's queue through `sessionQueue(ss)`, which sets both modes with the existing setters. No file added.
- **Deviation 19. Three keys accepted a value that could not take effect.**
  - **Found:** MADR D14 makes `mcp.startupWaitMs`, `mcp.stopGraceMs` and `mcp.stopTimeoutMs` integers ≥ 0. A zero duration means "the default" in `acpserver`'s `sessionSettings` (D17's "any zero field is filled") and in `mcpclient` (`cmp.Or(t.grace, stdinGrace)`, `mcpclient/proc.go:57`, from P1), so a user's 0 would silently become 10 s, 500 ms or 2 s.
  - **Decided** ("minimum 1 ms"): the three keys' least value is 1; MADR D14 gets a dated note. Expectations of "must be at least 0" become "at least 1". Files added: `internal/config/keys.go`, `internal/config/testdata/settings.schema.json` (regenerated), `internal/config/resolve_test.go`, `internal/config/edit_test.go`, `internal/cli/config_test.go`.
- **Deviation 20. Two more files had to change.**
  - **Found:** "resolved once, cached" (P4 step 2) needs a field on `runEnv`, which `internal/cli/main.go` declares. Step 6 changes `Compose`'s signature, which `internal/cli/slash_test.go:87` stubs.
  - **Decided** ("add both"): `runEnv` gains `process`, the once-resolved process settings; the stub takes the new parameter, with no assertion changed. Files added: `internal/cli/main.go`, `internal/cli/slash_test.go`.
- **Deviation 21. `sessionDir` belongs to the store, in both modes.**
  - **Found:** step 5 applies `sessionDir` in `configureStore`, which only the chat command runs; MADR D17 cites `sessionStore` (`internal/cli/agent.go:89-102`) as well, which `gobble acp` uses.
  - **Decided** ("sessionStore, both modes"): `sessionStore` gives the flat store for `sessionDir`; `configureStore` still sets `--session-dir`'s and `--no-session`'s first, so the flags win. `TestSessionDirSetting` covers both modes. No file added.
- **Deviation 22. `acptest`'s recorder cannot tap `acpclient`.**
  - **Found:** the recorder taps only `acptest.Connect`'s pair, which hands back a raw SDK client, not an `acpclient.Conn`.
  - **Decided** ("record at the agent"): `TestAdditionalDirectoriesSent`'s agent, embedding `acptest.ScriptAgent`, records each request's `additionalDirectories`; the assertions are the step's. No file added.

### P6 — Docs and the execution record (D2, D9)

1. **`docs/architecture.md`:**
   - the `internal/config` row reads "Settings: a commented `settings.yaml` from an embedded template, layered global, project (trusted) and `--set`, edited in place; the JSON Schema";
   - a new `internal/filelock` row: "An exclusive lock file between processes, for session files, `mcp.json` and settings";
   - the `internal/fsx` row gains "the repository finder".
2. **`README.md` `:59`:** "`gobble mcp add|remove|list`, `gobble config path|get|set|unset|edit|init|update|validate|schema`, `gobble version` and `gobble completion`", and one line under it: "Settings live in a commented `settings.yaml` in the config directory; `gobble config path` shows where."
3. **This plan:**
   - `status: completed`;
   - an `## Execution record (YYYY-MM-DD)`, naming each phase run, its gates' results, each negative and how it failed, and what this plan predicted wrongly;
   - 0018-MADR gains `## Observed — execution results`.
4. **`docs/README.md`:** the 0018 rows' status.

**P6's verification:**

```text
make check-records && make markdownlint -> ok, 0 issues
```

The records' own lint runs through the scratchpad's `q261_lint_records.py`, since `markdownlint` excludes MADR and PLAN files.

### P6's deviations, 2026-10-12, decided by the owner

- **Deviation 23. The deviation lists failed MD029, from P2 on.**
  - **Found:** the records' own lint (`q261_lint_records.py`, which copies the records beside a config that does not exclude them, and was seen failing on a planted MD012) reported 21 MD029 errors in this plan: each phase's deviation entries were an ordered list numbered from the plan-wide count, 2 to 22, and MD029 wants a list to start at 1. This plan as committed at P2 (`1ac178c`) already had four, and P5's commit had all 21; `make markdownlint` excludes records, and no earlier phase ran the records' own lint, so none saw them. 0018-MADR was clean.
  - **Decided** ("bulleted, numbers kept"): each entry is a bullet, `- **Deviation N. …**`, with its nested lines moved left to the bullet's indentation; the numbers, and so every "deviation N" in the records, code and tests, are unchanged. No file added.

## Verification (whole plan)

After P6:

```text
go test ./...                                     -> ok
CGO_ENABLED=1 go test -race ./...                 -> ok, natively on Windows and on the second Linux machine
make preflight                                    -> preflight passed (Windows ~~and WSL~~; WSL dropped 2026-10-12, see the stability rule)
git diff --exit-code acpserver/testdata/session.golden go.mod go.sum -> no diff
go list -deps ./internal/config | grep -c alecthomas/kong -> 0
python -I q421_claims_0018.py                     -> 133/133 claims hold (0018-MADR's citations, ~~re-run in case code moved~~ read from 81f52fd, the tree they were measured on; see the execution record)
```

### Acceptance criteria (mapped to MADR Confirmation)

| # | Criterion | MADR |
| :--- | :--- | :--- |
| A1 | With no settings, every existing test passes unedited but C1's two; `session.golden` byte-identical | D14, D17, Confirmation |
| A2 | Layers, precedence, `null`-clears and read filtering, by `resolve_test.go` | D2, D4, D16 |
| A3 | A project layer is disabled with D7's exact reason until trusted; global-only keys ignored from it | D5, D6 |
| A4 | A second document, a duplicate key, a tab and a non-mapping root each give file:line; type and range errors fall back | D3, D7 |
| A5 | Each D8 case changes only its bytes, by byte-diff; an unparsable file and a bad splice are refused with the file untouched | D8 |
| A6 | `update` adds missing blocks and keeps every line; `init` refuses an existing file | D9, D12 |
| A7 | D9's six template tests pass; the committed schema matches | D9, D12 |
| A8 | `--set` parses, refuses unknown paths with exit 2, beats the files, and loses to flags | D10, D11 |
| A9 | Read-only commands create nothing; the first run creates the template | D9 |
| A10 | Each of the thirteen keys reaches its consumer; `--add-dir` reaches the agent on new and resume | D14, D17 |
| A11 | Credential-shaped keys are refused, and no diagnostic prints a value | D15, C7 |
| A12 | Kong only in `internal/cli`; YAML only in `internal/config` and `internal/frontmatter`; no new module | D18, C3, C4 |
| A13 | Every gate passes; every negative seen failing | Stability rule, C5 |

A1 is the criterion most likely to be dropped under pressure. When P5 wires a default through a unit or a zero value, editing the test that pins it is the quickest way back to green, and it would hide the regression C1 exists to catch.

## Rollout and Rollback

- **Rollout:** the owner commits each phase.
  - Users see the template on their next run, all commented, so nothing changes until they set a key.
  - magic-cli-remote needs nothing.
- **Rollback:** revert the phase commits in reverse order.
  - A `settings.yaml` left behind is ignored by the reverted gobble, which reads no settings file.
  - `<config>/settings.yaml.lock` and `<config>/mcp.json.lock` files left behind are harmless, and can be deleted.

## Deferred (named, so they are not mistaken for oversights)

- **F6's trust store, `/trust` and `gobble trust`:** they supply P2's `Trusted` function; until then project settings are reported, not applied.
- **The later keys** (MADR D19): each arrives with its phase, carrying its template block under D9's tests:
  - F4's model keys;
  - F5's retry keys, `compaction.enabled` and `reserveTokens`;
  - F6's permission keys and `defaultProjectTrust`;
  - F7's `hooks`;
  - F9's theme and `keybindings.yaml`;
  - F10's `bridge.pi`;
  - `images.autoResize`;
  - 0017-MADR's reminder settings.
- **`/reload`** (MADR D16): a 1.0 command with no plan step yet; it will re-read through `config.Resolve`.
- **`GOBBLE_OFFLINE`:** bound with `--offline` in X3.
- **A user guide for settings:** the template is the documentation (MADR D9). A separate `docs/guides/` page waits for the owner to ask for one.
- **A system or managed layer, profiles and a file watcher** (MADR D21).

## Execution record (2026-10-12)

All six phases ran, in order, each committed by the owner before the next began: P1 `3547163`, P2 `1ac178c`, P3 `c42dd45`, P4 `0611155`, P5 `f9850cf`, and P6 with this record. Twenty-three deviations were raised, each with its evidence and a choice of real fixes; the owner chose each time, the PLAN carries each as a dated entry under its phase, and the original steps are struck through or annotated in place. The MADR carries notes for deviations 1, 7, 8, 10 and 19.

### The gates, phase by phase

Every phase ended with lint for linux, darwin and windows, Windows `go test ./...`, `CGO_ENABLED=1 go test -race ./...` natively on Windows and over SSH on the second Linux machine, Windows `make preflight`, the pre-add check on the phase's Go files, and a check of the staged index alone (gofmt, vet, test, `check-records` on an export of what was staged).

| Phase | Lint ×3 | Windows test | Race, Windows / Linux | Preflight | Pre-add | Index check |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| P1 | 0 issues (second run) | ok | 30 ok / 30 ok, no race | passed (Windows and WSL) | 17 files clean | exit 0 |
| P2 | 0 issues (third run) | 31 ok | 31 ok / 31 ok, no race | passed | 21 files clean | exit 0 |
| P3 | 0 issues (second run) | 31 ok | 31 ok / 31 ok, no race | passed | 4 files clean | exit 0 |
| P4 | 0 issues (second run) | 31 ok | 31 ok / 31 ok, no race | passed | 20 files clean | exit 0 |
| P5 | 0 issues (second run) | 31 ok | 31 ok / 31 ok, no race | passed; `apidiff` accepts the `acpclient` and `acpserver` additions | 25 files clean | exit 0 |
| P6 | 0 issues | 31 ok | 31 ok / 31 ok, no race | passed | no Go file | exit 0; both records pass their own lint |

From P2 on, no gate ran in WSL, by the owner's direction recorded in the stability rule.

### The negatives

Every negative ran on a scratch copy of the tree, never the working tree, after control runs on the unmutated copy had passed, and each mutation was asserted to have landed exactly once. A negative counted only when it failed the tests named for it; one that failed to build was rewritten, since a build failure proves nothing.

- **P1, 6 of 6:** `Lock` trying once fails `TestLockRetries` ("locked by pid"); `TryLock` without the flock fails `TestTryLockHeld`; `GitRepo` ignoring `commondir` fails `TestGitRepo`'s worktree case; `Add` without the lock fails `TestAddConcurrent` (at `-count=20`, one run kept 1 of 8 servers); the stopper's constant grace fails `TestStopGraceOption` (`Close` took 502 ms against a 400 ms bound); `WriteFileMode` ignoring its mode fails `TestWriteFileModeNewAndExisting` (a new file at 0644).
- **P2, 13 of 13:** the plan's eight (the second-document check removed, the trust gate inverted, a global-only key let through, the credential check removed, one byte of the committed schema changed, a template block deleted, a template default changed to 20001, a prose line reworded as `name: value`), a capitalised prose line that only the looser pattern catches, and each side of deviation 5's pins: `mcpclient`'s grace, `builtin`'s cap, `acpserver`'s start-up wait, and `internal/config`'s own default.
- **P3, 14 of 14:** the plan's five (case 1 replacing the whole line, `render` always plain, the check skipped, `Update` changing a line, line endings normalised to LF), `Update` overwriting a line with its own check left on (refused by `Update` itself, not by the test's line check), inserted lines always LF, deviation 7's rule off, case 2 off, the alias refusal off, `Set` without the lock, deviation 8's ancestor rule off, case 3 and the check both accepting any null parent, and `Init` overwriting.
- **P4, 12 of 12:** the plan's four (`config path` creating the settings file, `--set` errors ignored, `--project` letting a global-only key through, `validate --strict` exiting 0 on an unknown key), `set` writing before it checks, acp not creating the file, `set --project` creating a project file, `edit` exiting 0 on an error, `--offline` left on F4, `editor.Open` dropping the file, `Check` ignoring the layer, and `Diagnostic.Unknown` matching every error. Run twice, before and after the lint fixes.
- **P5, 15 of 15:** the plan's five (the session's queue modes not set, `Settings` called at each prompt, `sessionDir` beating `--session-dir`, `editorCommand` ignoring `configured`, `--add-dir` not sent on resume), `/compact` ignoring the session's settings, the MCP wait ignoring them, a new session ignoring `Settings`' thinking, a loaded session taking it, `sessionDir` not applied, acp's tools ignoring the settings, acp reading the project's process keys, `toSession` dropping `keepRecentTokens`, `--add-dir` unchecked, and a zero MCP timing accepted.

### What this plan predicted wrongly

- **The file lists.** P5's list left out eight files its own steps change (deviations 15 and 20), and its decisions added four more (19); P4's left out `internal/cli/editor/external.go` and its test (9), and `internal/config`'s `edit.go`, `diag.go` and two tests (13); P2's needed `.golangci.yml` (2) and three consumer tests (5). A file list written before the code is read for every caller is a guess.
- **Names and signatures.** Steps 1 and 3 of P2 both named `Read` in one package (3). P5 changed `newQueue`'s signature without counting its test callers (18). P4 defined two methods whose only callers were in P5, which the `unused` linter refuses (12).
- **Contradictions inside the records.** D8's case 3 and D8's check disagreed (8); D9 and D12 disagreed on who creates a project file (10); D14 allowed a zero that D17's fill rule and `mcpclient` turn into the default (19).
- **The fixtures.** Byte-exact CRLF fixtures cannot be committed under `.gitattributes`' `eol=lf` (4, 6).
- **The tests that could not fail.** `TestLaterFlagsAreRejected` reads its expectation from the map it checks, so it could not catch the `offline` mapping; P4 added an explicit assertion, seen failing. P2's `git diff --exit-code` on the generated schema cannot fail while the file is untracked; it was run after staging, and the regeneration was also checked by hash. P1's `TestStopGraceOption` bound of 1 s would not have caught the constant grace on Windows, where the job's terminate kills at the grace; it was tightened to 10 ms options and a 400 ms bound.
- **This plan's own verification line.** "Re-run q421 in case code moved" cannot hold: the citations describe the tree as measured, and P1 moves the cited code by design. The checkers now read gobble's files from `81f52fd`, the measured tree; 133 of 133 and 35 of 35 claims hold, and 0 when inverted.
- **Lint.** Every phase's first lint run found findings in the new code: 2 in P1, 11 in P2, then one more after a fix, and a handful in each later phase. Each was fixed in the code, with two exceptions: `genschema`'s streams, by the exclusion the owner chose (deviation 2), and one `os.ReadFile`, by the justified `//nolint:gosec // G304` the repository already uses.
- **The editor's details.** A bare `key:` is a decoded null with its tag filled in, so a branch keyed on an empty tag never ran; the `empty` fixture now covers it. yaml.v3 tags `yes`, `on` and `off` as strings but decodes them into a boolean, so a string with one of those values is written quoted, as the plan's `TestEditQuotesAmbiguous` expects.
- **The records' lint.** The stability rule runs `make markdownlint`, which excludes MADR and PLAN files by design, so no phase before P6 linted this plan; the deviation lists had failed MD029 since P2 (deviation 23). The records' own lint is now run on both records, with a planted violation seen failing.
- **What the plan's tests left uncovered.** The CLI's mapping of settings into `acpserver.SessionSettings` had no test; P5 added `TestSessionSettingsReachAgent`, seen failing.

### C1, checked over the whole plan

Against `81f52fd`, the existing tests that changed are: `TestConfigPath` (its two new lines) and `TestLaterFlagsAreRejected` (an assertion added), C1's two named exceptions; the call sites of `Compose`, `editorCommand` and `Open`, and `/edit`'s `compose` stub, which deviations 9, 17 and 20 changed; `mcpclient`'s `startFixture` helper, which P1 gave an options parameter; and the four "must be at least 0" texts that deviation 19's rule replaced. No assertion was loosened, and no test that pins a default was edited. `acpserver/testdata/session.golden`, `go.mod` and `go.sum` are byte-identical to `81f52fd`. The real settings file of the host the tests ran on was absent before and after every phase's suites, so no test wrote outside its temporary directories.

### Acceptance

A1 to A13 hold, by the gates and tests above. A1, the criterion this plan named as the one most likely to be dropped, held: every default is pinned on both sides (deviation 5), and none of those tests was edited.
