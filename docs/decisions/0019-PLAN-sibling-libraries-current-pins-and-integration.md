---
status: proposed
date: 2026-10-12
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# PLAN 0019 — Pin acp-go-sdk2 v0.17.0, go-llmprovider-sdk v1.10.2 and go-selfupdate-lib v1.15.2, and integrate each where the library now does the work

Implements [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md) decisions D1–D14, closing findings F1–F13.

## Goal

When every phase is committed:

1. `go list -m` reports `github.com/maccavelli/acp-go-sdk2 v0.17.0`, `github.com/maccavelli/go-llmprovider-sdk v1.10.2` and `github.com/maccavelli/go-selfupdate-lib v1.15.2`, and `.github/workflows/ci.yml` calls both go-selfupdate-lib workflows at `436a7f5976a6815491c7384527fef90869574a1b # v1.15.2`.
2. `go.mod` and `go.sum` differ from `108d9f0` in those three requirements and their six sums, and nothing else.
3. `go test -count=1 ./...` passes on Windows, natively on the second Linux machine and on the macOS laptop; `go test -race ./...` passes natively on Windows and on the second Linux machine; `make preflight` passes on Windows.
4. `acpserver/testdata/session.golden` is byte-identical to `108d9f0`, and the same golden passes on all three systems.
5. `python scripts/check_libs.py` reports each library at its latest tag, with no incompatible change, no new module and no advisory, and exits 0; it exits 1 on each of its planted negatives.
6. magic-cli-remote's ACP session opens, prompts and closes against the agent of P4's commit.
7. No test is skipped, and no assertion loosened, to reach 3 (C2).

## Scope

### In scope (the only files any phase may touch)

| Phase | New | Changed |
| :--- | :--- | :--- |
| P1 | `scripts/check_libs.py` | `Makefile` |
| P2 | — | `go.mod`, `go.sum`, `.github/workflows/ci.yml`, `docs/architecture.md`, `README.md` |
| P3 | `llm/provider/ambient_test.go`, `acpserver/usage_test.go` | `go.mod`, `go.sum`, `llm/provider/provider.go`, `llm/provider/provider_test.go`, `llm/doc.go`, `llm/llm.go`, `acpserver/history.go`, `docs/architecture.md`, `README.md` |
| P4 | `acpclient/acptest/abs.go`, `acpclient/acptest/abs_test.go`, `acpserver/rewrite.go`, `acpserver/rewrite_test.go`, `acpserver/list_test.go` | `go.mod`, `go.sum`; `acpclient/conn.go`, `acpclient/sessions.go`, `acpclient/conn_test.go`, `acpclient/sessions_test.go`, `acpclient/elicitation_test.go`; `acpclient/acptest/recorder.go`, `acpclient/acptest/script.go`, `acpclient/acptest/pair_test.go`, `acpclient/acptest/script_test.go`; `acpserver/serve.go`, `acpserver/agent.go`, `acpserver/mcp.go`, `acpserver/agent_test.go`, `acpserver/commands_test.go`, `acpserver/fixtures_test.go`, `acpserver/tree_test.go`, `acpserver/delete_test.go`, `acpserver/mcp_test.go`; `internal/cli/line_test.go`; `docs/architecture.md`, `README.md` |
| P5 | — | `docs/decisions/0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md`, `docs/decisions/0014-MADR-move-to-acp-go-sdk2.md`, `docs/decisions/0005-MADR-v1-feature-scope.md`, `docs/decisions/0005-PLAN-v1-feature-scope.md`, `docs/decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md`, `docs/reports/0014-REPORT-llmprovider-sdk-integration.md`, `docs/reports/0015-REPORT-sibling-library-pins-and-order.md`, `docs/README.md`, this plan (its execution record) |

A phase that needs a file not listed here stops, and the file is raised as a deviation.

### Out of scope

- **Building what D11 hands to its owners:** 0012-PLAN P7 (the strict flag), 0005-PLAN F4-1a's error flag and wire tests, F4-2 (the catalog on `ModelInfo`), F4-3 (credentials) and F4-1b (live streaming). This plan only re-bases their entries (P5).
- **`gobble update`** (0005-PLAN F10, D12). Only its entry is corrected.
- **0016-MADR's D4–D6 and D8–D13** (notices, compaction updates, conformance goldens, discriminators). They stay 0016's, re-based by D10's note.
- **The request to go-llmprovider-sdk for a streaming `WithRetry`** (D11): the owner files it in that repository; this plan changes nothing there.
- **The `go-core-lib` path string** in `internal/archtest/check.go` (`:146`, `:155`): harmless, and archtest's rules are 0004-MADR's to change.

## Stability rule

Every phase ends with, in order:

```bash
export GOTOOLCHAIN=go1.27.2
make pre-add-check FILES="<the phase's .go files>"         # P1, P2 and P5 have none
python -I <scratch>/q428_gates_0019.py <phase>              # lint for linux, darwin, windows; Windows go test; Windows -race;
                                                            # Windows make preflight; the second Linux machine's -race; the macOS laptop's go test
python -I <scratch>/q252_check_index.py <phase>-0019        # gofmt, vet, test and check-records on the staged index alone
python -I <scratch>/q261_lint_records.py --negative <each record the phase touches>
```

- `q428_gates_0019.py` is `q428_gates_0018.py` with one gate added: the macOS laptop's `go test -count=1 ./...`, run as the second Linux machine's run is, on a tarball in a fresh `/tmp` directory removed afterwards. No gate runs in WSL (0018-PLAN's stability rule, the owner's direction of 2026-10-12). The host aliases stay in the scripts and are never written into a record.
- **Commits.** The agent stages each phase with `git add` and never commits. The owner commits each phase with `git commit --no-edit` before the next begins. No push and no tag without the owner's request in that turn.

## Cross-cutting contracts

- **C1. One library per commit.** P2, P3 and P4 each change one requirement line in `go.mod`, with its two `go.sum` lines, and nothing else in either file.
- **C2. No test is weakened.** A test that asserted gobble's own refusal asserts the SDK's refusal of the same input (D7, D8): an error type and its fields, or the code the peer receives. None is skipped on any system, and none is deleted.
- **C3. Every default is pinned on both sides.** 0018-PLAN's pin tests (`TestStopDefaults`, `TestDefaultMaxTimeout`, `TestDefaultStartupWait`, `TestDefaultsEqualCode`) stay unedited, and P3's default-model test is a new pin, never a loosened one.
- **C4. No module is added.** Each phase's `go mod tidy` leaves `go.mod` with the pin's line changed only, and `scripts/check_libs.py` fails on a new module.
- **C5. Every new check is seen failing,** on a scratch copy, before it is trusted.

**C2 is the contract most at risk.** P4 rewrites five tests whose assertions the SDK now pre-empts. Swapping an `errors.AsType[*acp.RequestError]` for `err != nil` would turn each green and stop testing anything; each rewrite therefore names the type, the field values and the receiving side's code it asserts.

## Dependency and delivery order

| Order | Phase | Why here |
| ---: | :--- | :--- |
| 1 | P1 — `scripts/check_libs.py` | Each pin phase runs it first, on the tag it is about to pin. |
| 2 | P2 — go-selfupdate-lib `v1.15.2` | No Go change; the smallest commit. |
| 3 | P3 — go-llmprovider-sdk `v1.10.2` | Must land before 2026-10-23 (MADR D2). |
| 4 | P4 — acp-go-sdk2 `v0.17.0` | The largest commit; nothing after it depends on it but P5. |
| 5 | P5 — records | Notes cite the commits of P2 to P4. |

## Implementation Steps

### P1 — The compatibility checks, as a repository script (D14; closes F12)

1. **`scripts/check_libs.py`** (new), in the style of `scripts/check_records.py` (a module docstring naming its record, `argparse`, functions with one job each, `main()` under `if __name__ == "__main__":`; stdlib only):
   - **Libraries:** a table of the three modules, `github.com/maccavelli/<repo>`, each with its packages to test (`./acpclient/... ./acpserver/... ./internal/cli/...`; `./llm/... ./acpserver/... ./internal/cli/...`; `./internal/buildinfo/... ./internal/cli/... ./internal/archtest/...`).
   - **Options:** `--lib NAME` (one library; default all three), `--tag vX.Y.Z` (default: the latest semver tag), `--from vX.Y.Z` (default: the version `go list -m` reports in the repository), `--keep` (keep the scratch clones), `--self-test`.
   - **The latest tag:** `git ls-remote --tags https://github.com/maccavelli/<repo>`, semver tags only (`^v\d+\.\d+\.\d+$`), sorted numerically; never a branch or pseudo-version.
   - **For each library, in a fresh `tempfile.mkdtemp()` directory outside the working tree:**
     1. `git clone --quiet <repo root> <tmp>/clone`;
     2. `go get <module>@<tag>`, `go mod tidy`;
     3. the `go.mod` diff: parse `go.mod` before and after (`require` lines) and fail when any module other than `<module>` is added, removed or changed (`new or changed module: <path>`); *(deviation 1: and `go list -m <module>` must report `<tag>` after `tidy`, else the library fails with `tidy moved it to <version>: <tag> lacks a package gobble imports`, and its other checks are not run, since they would test another version;)*
     4. `go build ./...`; `go vet ./...` with `GOOS` linux, darwin and windows; `go test -count=1 <packages>`;
     5. apidiff: `go build -o <tmp>/apidiff golang.org/x/exp/cmd/apidiff` from the repository; `go mod download -json <module>@<from>` and `@<tag>`; `apidiff -m -w <tmp>/<v>.exp <module>` run in each version's module directory; `apidiff -m <from>.exp <tag>.exp`; parse the text before `Compatible changes:` and fail on any line there that starts with a hyphen (`incompatible: <line>`), since apidiff's exit status is not a verdict. For a module below v1 the incompatible changes are reported and do not fail, as 0004-MADR expects breaks there; the report says so;
     6. `go run golang.org/x/vuln/cmd/govulncheck ./...` in the clone (the repository's tool pin);
     7. the advisories: `GET https://api.github.com/repos/maccavelli/<repo>/security-advisories` with `urllib.request`, unauthenticated (it answers 200 today), and fail when a published advisory's `vulnerable_version_range` covers `<tag>` (`advisory <ghsa_id> covers <tag>`). Ranges are parsed for `=`, `<`, `<=`, `>=`, `>` and comma-joined conjunctions, which is every form the endpoint returns.
   - **Output:** one line per library and check, `ok` or the failure; each command's whole output in `<tmp>/logs/<lib>-<step>.log`, whose directory is printed. Exit 1 on any failure.
   - **No credential is read, printed or sent.** The environment passed to each command drops every variable whose name contains `KEY`, `TOKEN`, `SECRET` or `PASSWORD`, as the research scripts did.
   - **`--self-test`** runs the parsers on fixed inputs: an apidiff text with one incompatible line fails and one with none passes; a `go.mod` pair adding `example.com/x v1.0.0` fails and one changing only the module passes; the range `= 0.15.0` covers `v0.15.0` and not `v0.16.0`, and `< 1.2.0, >= 1.0.0` covers `v1.1.0` and not `v1.2.0`.
2. **`Makefile`:** a `check-libs` target, `$(PYTHON) scripts/check_libs.py`, with the comment `# The sibling libraries against their latest tags (0019-MADR D14); needs the network, so not in preflight.` It is added to `.PHONY` and to no other target.

**P1's verification:**

```text
python scripts/check_libs.py --self-test                                    -> exit 0, every parser case as listed
python scripts/check_libs.py                                                -> each library: tag v0.17.0 / v1.10.2 / v1.15.2;
                                                                               acp-go-sdk2 test fails (F3, F7) and its 17 below-v1 breaks are reported;
                                                                               the other two: every check ok; exit 1 because of acp-go-sdk2's tests
python scripts/check_libs.py --lib go-llmprovider-sdk                       -> exit 0
python scripts/check_libs.py --lib go-selfupdate-lib                        -> exit 0
```

**P1's negatives,** each run as a real invocation, not a self-test:

- `--lib acp-go-sdk2 --tag v0.15.0` fails with `advisory GHSA-579q-9j9v-7w8w covers v0.15.0`;
- `--lib go-llmprovider-sdk --from v1.10.2 --tag v1.2.1` fails with incompatible changes, reported as such (a downgrade removes API), so a v1 break fails the run;
- ~~`--lib go-selfupdate-lib --tag v1.5.0` fails at `go vet` (`release_test.go:9:2: no required module provides package …/selfupdate/releasespec`), which `go build` alone passes;~~ *(Deviation 1: `go mod tidy` repairs that import by moving the library to its newest tag before `go vet` runs, so the run fails on the version check instead: `go-selfupdate-lib version: FAIL tidy moved it to v1.15.2: v1.5.0 lacks a package gobble imports`.)*
- a copy of the script whose `go.mod` parser is replaced with one that ignores other modules makes the `--self-test` case "adding `example.com/x`" pass, which `--self-test` reports as a failed case.

### P1's deviations, 2026-10-12, decided by the owner

- **Deviation 1. `go mod tidy` could change the version being checked.**
  - **Found:** negative 3 (`--lib go-selfupdate-lib --tag v1.5.0`) did not fail at `go vet`. `go get` downgraded the clone to `v1.5.0` (its log: "downgraded github.com/maccavelli/go-selfupdate-lib v1.9.0 => v1.5.0"), but `v1.5.0` has no `selfupdate/releasespec`, which `internal/cli/release_test.go:9` imports, so `go mod tidy` moved the library to `v1.15.2`, the newest tag providing it; the clone's `go.mod:17` read `v1.15.2`. Every later step tested `v1.15.2` while the report said `v1.5.0`; only the API step, in its own scratch module, was at `v1.5.0`. The research run had seen `vet` fail because it ran no `tidy`. The defect is in the new script, and the step as written could not catch it.
  - **Decided** ("check the version after tidy"): after `get` and `tidy`, `go list -m <module>` must report the tag, else that library fails with `tidy moved it to <version>: <tag> lacks a package gobble imports` and its other checks are skipped. Negative 3 expects that line. No file added.

### P2 — go-selfupdate-lib `v1.15.2` (D1; closes F10)

1. `python scripts/check_libs.py --lib go-selfupdate-lib` exits 0 at `v1.15.2`. Its log directory is kept for the execution record.
2. `go get github.com/maccavelli/go-selfupdate-lib@v1.15.2`, `go mod tidy`. `go.mod:17` reads `github.com/maccavelli/go-selfupdate-lib v1.15.2`; `go.sum` holds `h1:FuQ++o9ojyQUDWR7g7WvLKYBzyNplztF3KrziyynlFE=` and the `go.mod` sum `h1:yZNc7rTDTKgGS2DaqNKRS9ph09bsIsP6ymJBCiV1Kfo=`.
3. `.github/workflows/ci.yml:145` and `:157`: `@39b12945fad311174252038745f5a88f71c4c66c # v1.9.0` becomes `@436a7f5976a6815491c7384527fef90869574a1b # v1.15.2`, in both lines.
4. `docs/architecture.md:111`, the go-selfupdate-lib row of its dependency table, and `README.md:88`, the line after "Build identity and self-update": `v1.9.0` becomes `v1.15.2`.

**P2's verification:**

```text
go list -m github.com/maccavelli/go-selfupdate-lib                          -> github.com/maccavelli/go-selfupdate-lib v1.15.2
git diff -U0 HEAD -- go.mod | grep -c '^[-+]\s'                             -> 2
grep -c '436a7f5976a6815491c7384527fef90869574a1b # v1.15.2' .github/workflows/ci.yml -> 2
grep -c '39b12945fad311174252038745f5a88f71c4c66c' .github/workflows/ci.yml  -> 0
python scripts/verify_build_metadata.py                                     -> 7 of 7
go test -count=1 ./internal/buildinfo/... ./internal/cli/... ./internal/archtest/...   -> ok
```

### P3 — go-llmprovider-sdk `v1.10.2` (D2; closes F8, F13)

1. `python scripts/check_libs.py --lib go-llmprovider-sdk` exits 0 at `v1.10.2`.
2. `go get github.com/maccavelli/go-llmprovider-sdk@v1.10.2`, `go mod tidy`; `go.sum` holds `h1:llPpBo+EKnQE4ksae+ZBK9xFOQ6z5NpK18H4cosmyJY=` and `h1:en9k+ZhlMT41q/+FtXQrLE/3qDaBrWFr8TeLyZw2+Jg=`.
3. **`llm/provider/provider.go`, `New` (`:63-65`):** the retry wrapper is built once and its capabilities are the adapter's:

   ```go
   // The retry wrapper buffers the stream, so its capabilities, not the
   // inner provider's, are what gobble can deliver (0019-MADR D2).
   w := llmprovider.WithRetry(p, retryPolicy)
   return &adapter{p: w, id: p.ID(), caps: w.Capabilities()}, nil
   ```

   `Wrap` keeps the provider's own capabilities, since it adds no retry.
4. **`usageOf` (`:164-176`):** the non-zero test also reads `u.CacheWriteTokens`, and the reported `llm.Usage` sets `CacheWriteTokens: int64(u.CacheWriteTokens)`.
5. **`llm/doc.go:47-50`:** "Cache-write stays unknown until the SDK has a count for it." becomes "Cache-write is the SDK's count, from go-llmprovider-sdk `v1.10.0`." **`llm/llm.go:21`:** "NativeStreaming stays false while the SDK reports streaming unsupported." becomes "NativeStreaming stays false while the SDK's retry wrapper buffers the stream (0005-PLAN F4-1b)."
6. **`acpserver/history.go`, `piUsage` (`:115-117`):** Pi's `input` excludes cached tokens, so:

   ```go
   // Pi's input is the uncached input; llm's InputTokens, as the SDK's,
   // includes cache reads and writes (0019-MADR D2, F13).
   in := max(0, u.InputTokens-u.CacheReadTokens-u.CacheWriteTokens)
   out := &session.Usage{Input: in, Output: u.OutputTokens, CacheRead: u.CacheReadTokens, CacheWrite: u.CacheWriteTokens,
       Reasoning: u.ReasoningTokens, TotalTokens: u.InputTokens + u.OutputTokens}
   ```

7. **Docs:** `docs/architecture.md:110` and `README.md:84` name go-llmprovider-sdk `v1.2.1`; each becomes `v1.10.2`.
8. **Tests:**
   - `llm/provider/ambient_test.go` (new), `TestAmbientDefaultModels`: for each of the nine key-taking providers, `Choose` with only that provider's `EnvVar` set returns its id and the model of MADR's `v1.10.2` column, as literals (gemini `gemini-3.5-flash-lite`, openai `gpt-6-luna`, claude `claude-haiku-5-5`, grok `grok-4.6`, opencode-zen and opencode-go `claude-haiku-5-5`, huggingface `deepseek-ai/DeepSeek-V4-Flash-0731`, kilo `deepseek/deepseek-v4.1-flash`, together `zai-org/GLM-5.3-Flash`), and a test fails naming the provider whose default moved. The values are read from the probe of MADR's table, never copied from the test's own output.
   - `llm/provider/provider_test.go`: `TestNewReportsRetryCapabilities`: `New("openai", Options{APIKey: "test-only"})` reports `NativeStreaming` false, and `Wrap` of a stub reporting `Supported` reports true. `TestUsageCacheWrite`: a response with `InputTokens 100, CachedTokens 30, CacheWriteTokens 10` gives `CacheReadTokens 30, CacheWriteTokens 10`, and one with only `CacheWriteTokens 5` is not estimated.
   - `acpserver/usage_test.go` (new), `TestPiUsageExcludesCache`: `piUsage(llm.Usage{InputTokens: 100, OutputTokens: 20, CacheReadTokens: 30, CacheWriteTokens: 10})` stores `Input 60, CacheRead 30, CacheWrite 10, TotalTokens 120`, Pi's arithmetic (`openai-completions.ts:1544-1546` at Pi `312184e`); `InputTokens 5, CacheReadTokens 10` stores `Input 0`; an estimated usage still carries `{"estimated":true}`.

**P3's verification:**

```text
go list -m github.com/maccavelli/go-llmprovider-sdk                         -> v1.10.2
git diff -U0 HEAD -- go.mod | grep -c '^[-+]\s'                             -> 2
go test -count=1 ./llm/... ./acpserver/... ./internal/cli/...               -> ok
go test -count=1 ./llm/provider -run 'TestAmbientDefaultModels|TestNewReportsRetryCapabilities|TestUsageCacheWrite' -v -> each PASS
go test -count=1 ./acpserver -run TestPiUsageExcludesCache -v               -> PASS
git diff --exit-code acpserver/testdata/session.golden                      -> no diff
```

**P3's negatives,** on scratch copies: `New` reporting `p.Capabilities()` fails `TestNewReportsRetryCapabilities`; `usageOf` dropping `CacheWriteTokens` fails `TestUsageCacheWrite`; `piUsage` writing `Input: u.InputTokens` fails `TestPiUsageExcludesCache`; one literal of `TestAmbientDefaultModels` changed to `v1.2.1`'s value fails it, naming the provider.

### P4 — acp-go-sdk2 `v0.17.0` (D3–D9; closes F2–F7)

1. `python scripts/check_libs.py --lib acp-go-sdk2` reports `v0.17.0` with no advisory and no new module; its test step fails with the 27 tests of MADR's table, which this phase fixes. Its 17 incompatible changes are reported, none in gobble's use.
2. `go get github.com/maccavelli/acp-go-sdk2@v0.17.0`, `go mod tidy`; `go.sum` holds `h1:d+ZASkLAYayy4xPtvaLEQShRy6BYoX21RbZsFSjNus8=` and `h1:KqDRvShTbZEOpKOQOyRUnbeFqlyH20LXi7k3UOsgv0I=`.
3. **D4, host-absolute fixtures.** `acpclient/acptest/abs.go` (new):

   ```go
   // Abs is a POSIX fixture path, such as "/work", as an absolute path on
   // the host: itself off Windows, and on Windows under the system drive,
   // "C:\work". acp-go-sdk2 checks a path it receives with the host's rule
   // from v0.16.0 (0019-MADR D4).
   func Abs(p string) string
   ```

   On Windows it returns the system drive, `os.Getenv("SystemDrive")` or `C:` when that is empty, followed by `filepath.FromSlash(p)`; elsewhere `p`. `abs_test.go` checks that `Abs("/work")` is absolute on the host, that its slash form ends in `/work`, and that `Abs` of a nested path keeps every element.
   - Every fixture path of the inventory below becomes `acptest.Abs(...)`; a path inside a raw JSON string is written with `json.Marshal(acptest.Abs(p))`:
     - `acpclient/acptest/pair_test.go:114`; `acpclient/acptest/script_test.go:29`, `:61`;
     - `acpclient/conn_test.go:62`, `:73`, `:131`, `:151`, `:170`, `:174`, `:178`, `:190`, `:204`, `:219`, `:264`, `:318`, `:330`;
     - `acpclient/elicitation_test.go:80`; `acpclient/sessions_test.go:63`, `:127-139`;
     - `acpserver/agent_test.go:78`; `acpserver/commands_test.go:216`; `acpserver/fixtures_test.go:27` (`piCwd` becomes a `var`); `acpserver/tree_test.go:119`'s header line, built with the same helper;
     - `internal/cli/line_test.go:70` (the harness's default `cwd`).
   - `acpserver/settings_test.go:112`, `:124` and `acpserver/agent_test.go:180` call `Agent` methods directly, which the SDK does not validate, and stay.
4. **D4, host-independent goldens.** `acpclient/acptest/recorder.go`, `Golden`: before comparing or writing, the transcript's JSON strings that begin with the JSON-escaped form of `Abs("/")` are rewritten to their slash form (`"C:\\work"` → `"/work"`, `"C:\\work\\a.go"` → `"/work/a.go"`), by one `regexp` over string literals; off Windows `Abs("/")` is `/` and the rewrite is a no-op. `session.golden` is not regenerated; it must pass unchanged on all three systems.
5. **D5, `acpserver/agent.go`, `ListSessions` (`:822-828`):** each stored row is built, then checked with `info.Validate()`; a row that fails is left out and logged once per session id, `a.logger.WarnContext(ctx, "session: not listed: its cwd is not absolute on this host", slog.String("session", …), slog.String("cwd", …))`, and the loop goes on. Live sessions are always absolute (their `cwd` came through the SDK), so they are not checked.
6. **D6, `acpserver/rewrite.go`** (new): `resolveBareCommands(ctx context.Context, method string, params any) error`. For `*acp.NewSessionRequest`, `*acp.LoadSessionRequest` and `*acp.ResumeSessionRequest`, each `McpServers[i].Stdio` whose `Command` is not `filepath.IsAbs` is resolved with `exec.LookPath`; a result that is absolute replaces `Command`; an error, or a relative result (`exec.ErrDot`), leaves it as it is, for the SDK to refuse. It never returns an error. `acpserver/serve.go:32` passes `acp.WithInboundRewrite(resolveBareCommands)` beside `acp.WithLogger(logger)`.
7. **D7, the checks the SDK owns:**
   - `acpclient/conn.go:125-127`: the version comparison is removed. After `Initialize`, an error that is a `*acp.ProtocolVersionError` is reported as `fmt.Errorf("agent speaks ACP protocol %d, want %d", pve.Answered, pve.Requested)`, so the message users see is unchanged.
   - `acpserver/mcp.go:111-114`: the `r.Sse` and `r.Acp` cases are removed; the `"an MCP server has no transport"` fallback stays.
   - `acpserver/agent.go:736-738`: the empty-`cwd` fallback is removed; `open` uses the request's `cwd`.
8. **D8, the tests the gates pre-empt** (C2):
   - `acpclient/acptest/script.go`: `ScriptAgent` gains `Capabilities *acp.AgentCapabilities`; `Initialize` answers it when non-nil, and today's value (`Image` from `a.Image`, `Close`) when nil.
   - `acpclient/sessions_test.go`, `TestAdditionalDirectoriesSent`: its agent advertises `AdditionalDirectories` and `Resume`; a second case with today's advertisement asserts `NewSessionWith` with directories fails with `*acp.CapabilityError{Method: "session/new", Capability: "sessionCapabilities.additionalDirectories"}` and that the agent recorded no request.
   - `acpclient/elicitation_test.go`, `TestElicitationNotFound`: the typed `conn.CreateElicitation` fails with a `*acp.CapabilityError` whose `Capability` is `elicitation.url`; a raw `acp.SendRequest` of the same request is answered -32602 by the client's SDK; gobble's client records no elicitation. `elicitation/complete` and the later `NewSession` stay as they are.
   - `acpserver/delete_test.go`, `TestDeleteSessionNotFound`: the advertisement check stays; the typed `DeleteSession` fails with `*acp.CapabilityError{Method: "session/delete", Capability: "sessionCapabilities.delete"}`; a raw `acp.SendRequest` of `session/delete` is answered -32601, the original assertion, now through the agent's SDK.
   - `acpserver/mcp_test.go`, `TestUnofferedTransportsRefused`: the SSE and ACP servers are sent raw (`acp.SendRequest`, `session/new`) and answered -32602 with the SDK's `mcpCapabilities` reason; the stdio case keeps its bad name (`"bad name"`) with `Command: acptest.Abs("/bin/x")`, so gobble's own name check is what refuses it, -32602; the last assertion checks the SDK's text, `mcpCapabilities.sse`, in place of gobble's removed one.
   - `acpclient/conn_test.go`, `TestStartRefusesOtherProtocols`: unchanged; its message is kept by D7's mapping.
   - `internal/cli/line_test.go:163-168`, `TestLineSessionFirstPrompt`: after `waitUntil`, `if t.Failed() { t.FailNow() }`, so a failed wait stops instead of indexing an empty slice.
9. **D6's and D5's tests:**
   - `acpserver/commands_test.go`: `TestMcremoteScript`'s pipes and `call` move into `rawServe(t, Options) (call func(id int, method, params string) string, stop func())`, used by it and below.
   - `acpserver/rewrite_test.go` (new): `TestResolveBareCommands` sets `PATH` to a temp directory holding an executable `gobble-fake-mcp` (`.exe` on Windows; mode 0o755); a `NewSessionRequest` whose stdio command is `gobble-fake-mcp` is rewritten to its absolute path, one with `no-such-command-0019` is left alone, and an absolute command is left alone. `TestServeResolvesBareCommand`, through `rawServe`: `session/new` with the bare `gobble-fake-mcp` is answered with a session, and with `no-such-command-0019` is answered -32602 with `is not an absolute path`.
   - `acpserver/list_test.go` (new), `TestListSkipsNonAbsoluteCwd`: a `jsonl` store holds two sessions, written through `Store.Create` and one `Append` each, with header `cwd` `t.TempDir()` and `relative/dir`; `session/list` answers one session, the absolute one, and the log names the other once.
10. **D9, `acpclient/sessions.go`, `sessionError` (`:40-45`):** the reason is `data.reason`, else `data.error` (the SDK's -32602 detail), else the message. `acpclient/conn_test.go` gains `TestInvalidParamsCarriesReason`: an agent answering -32602 with `{"error":"NewSessionRequest: cwd: \"x\" is not an absolute path"}` gives an `ErrInvalidSession` whose text contains `cwd: "x" is not an absolute path`.
11. **Docs:** `docs/architecture.md:108` and `README.md:82` name acp-go-sdk2 `v0.14.0`; each becomes `v0.17.0`.
12. **magic-cli-remote** (D3; the owner's answer of 2026-10-12): with the staged tree built (`go build -o <scratch>/gobble.exe ./cmd/gobble`), magic-cli-remote's ACP client runs a session against `<scratch>/gobble.exe acp`: initialize, `session/new` in a temp directory, one prompt, close. Its transcript is kept; any capability, validation or -32602 error stops the phase as a deviation.

**P4's verification:**

```text
go list -m github.com/maccavelli/acp-go-sdk2                                -> v0.17.0
git diff -U0 HEAD -- go.mod | grep -c '^[-+]\s'                             -> 2
go test -count=1 ./...                                                      -> ok on Windows; the gates run it on the second Linux machine and the macOS laptop
git diff --exit-code acpserver/testdata/session.golden                      -> no diff
grep -n 'ProtocolVersion != acp.ProtocolVersionNumber' acpclient/conn.go    -> no match
grep -n 'the SSE transport is not offered\|the ACP transport is not offered' acpserver/mcp.go -> no match
grep -rn 'Cwd: "/w\|"/work' --include=*_test.go acpclient acpserver internal/cli -> only acptest.Abs arguments
```

**P4's negatives,** on scratch copies: `Abs` returning `p` on Windows fails 22 tests there (the path set of MADR's table); `Golden` without its rewrite fails `TestSession` on Windows; `ListSessions` without the `Validate` check fails `TestListSkipsNonAbsoluteCwd`; `Serve` without `WithInboundRewrite` fails `TestServeResolvesBareCommand`; `resolveBareCommands` keeping a relative `LookPath` result fails `TestResolveBareCommands`; `sessionError` without `data.error` fails `TestInvalidParamsCarriesReason`; `TestDeleteSessionNotFound`'s raw request sent to an agent that implements `DeleteSession` fails it.

### P5 — The records (D10–D13)

Each note is dated 2026-10-12 (or the day P5 runs), cites 0019-MADR's decision and the commit of the phase that carried it, and is added beside the text it corrects, which stays.

1. **`docs/decisions/0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md`:** the note of D10, under its frontmatter's first heading and at D1, D2, D3 and D7.
2. **`docs/decisions/0014-MADR-move-to-acp-go-sdk2.md`:** at F7 (`:48`), "`v0.17.0` is pinned by 0019-MADR D3"; at its Confirmation's -32601 lines for the delete and elicitation tests, "now asserted through the SDK's gates (0019-PLAN P4 step 8)".
3. **`docs/decisions/0005-MADR-v1-feature-scope.md`:** at `:268`, "five raw binaries"; at `:587` and `:608`, "go-core-lib is go-selfupdate-lib, renamed 2026-10-03"; at `:598`, "fixed in go-selfupdate-lib `v1.5.1` (its 0010-MADR B5, B6)"; at `:966`, "superseded at go-llmprovider-sdk `v1.9.0`: `catalog.ModelInfo` (0019-MADR F9)".
4. **`docs/decisions/0005-PLAN-v1-feature-scope.md`:** at the F4-1 amendment entry (`:3342`), "re-based on `v1.10.2` by 0019-MADR D2; the pin and a true `NativeStreaming` landed in 0019-PLAN P3, and F4-1a keeps the tool result's error flag and the wire tests"; at F4-2's entry, "re-made on `v1.9.0`'s `catalog.ModelInfo` (0019-MADR D11)"; at F4-3's, "the SDK's `TokenStore`, `RefreshLocker` and logins; terminal auth advertised before offered (0019-MADR D11, F6)"; at F10's (`:342`, `:374`, `:640`), D12's three corrections and "the newest go-selfupdate-lib at F10".
5. **`docs/decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md`:** at `:1386`, D11's amendment of P7: `Strict` stays `nil` except for providers whose live test has shown `true` is accepted.
6. **`docs/reports/0014-REPORT-llmprovider-sdk-integration.md`:** at the gap table (`:279-288`), "strict and the error bit at `v1.7.0`, native streaming at `v1.6.0`, cache writes at `v1.10.0`; still open: deferral, `cache_control`, images, a generic Chat Completions provider (0019-MADR F9)".
7. **`docs/reports/0015-REPORT-sibling-library-pins-and-order.md`:** at `:163-166`, the three corrected defaults, measured by running the SDK (0019-MADR's table); at F2's and F6's `v1.8.0` metadata, "`v1.9.0`".
8. **`docs/README.md`:** the 0019 rows read `accepted` and `completed (<date>)`; the build-order list gains `**0019-PLAN.** Run: P1–P5 on <date>.`
9. **This plan:** `status: completed`, and its `## Execution record (<date>)`: each phase's commit, gates and negatives, and what this plan predicted wrongly.

**P5's verification:**

```text
make check-records && make markdownlint                                     -> ok, 0 issues
python -I <scratch>/q261_lint_records.py --negative <every record P5 touches> -> only the planted violation
grep -c '0019-MADR' <each record of steps 1-7>                              -> at least 1 each
```

## Verification (whole plan)

After P5:

```text
go list -m github.com/maccavelli/acp-go-sdk2 github.com/maccavelli/go-llmprovider-sdk github.com/maccavelli/go-selfupdate-lib
                                                   -> v0.17.0, v1.10.2, v1.15.2
git diff 108d9f0 -- go.mod | grep -c '^[-+]\s'     -> 6
git diff --exit-code 108d9f0 -- acpserver/testdata/session.golden -> no diff
python scripts/check_libs.py                       -> every library at its latest tag; exit 0
the gates                                          -> lint ×3, tests on three systems, -race on two, preflight
```

### Acceptance criteria (mapped to MADR Confirmation)

| # | Criterion | MADR |
| :--- | :--- | :--- |
| A1 | The three pins and the two workflow lines | D1–D3 |
| A2 | `go.mod` changes in three requirement lines only; no module added | D1–D3, C4 |
| A3 | Tests pass on Windows, Linux and macOS; `-race` on Windows and Linux; preflight on Windows | D4, Confirmation |
| A4 | `session.golden` byte-identical, on every system | D4 |
| A5 | Every provider's ambient model is pinned to the SDK's; the streaming flag is false; cache writes are reported | D2 |
| A6 | Pi's `input` and `totalTokens` in stored sessions | D2, F13 |
| A7 | A non-absolute stored `cwd` is left out of `session/list` | D5 |
| A8 | A bare stdio command resolves; an unresolvable one is -32602 | D6 |
| A9 | gobble's duplicate checks are gone, and the tests assert the SDK's refusals | D7, D8 |
| A10 | A refused value is named in the client's error | D9 |
| A11 | magic-cli-remote's session against P4's agent | D3 |
| A12 | The records carry D10–D13's notes | D10–D13 |
| A13 | `check_libs.py` passes, and each of its negatives fails | D14 |

**A9 is the criterion most likely to be dropped.** Each rewritten test would also go green with a bare `err != nil`, and under time pressure that is the shortest path; C2 and P4 step 8 name what each must assert instead.

## Rollout and Rollback

- **Rollout:** one commit per phase, in order; P3 before 2026-10-23.
- **Rollback:** a phase reverts with `git revert` of its commit. P2, P3 and P4 are independent of each other's code, since each changes only its library's line and the code that library's change needs; P1's script is unaffected by any of them. P5 reverts as records only.

## Deferred (named, so they are not mistaken for oversights)

- **The entries D11 re-bases** (0012-PLAN P7, 0005-PLAN F4-1a's remainder, F4-2, F4-3, F4-1b): each runs as its own phase, on the newest tag that `check_libs.py` passes then.
- **`gobble update`** (F10): last in 0005-PLAN's order; this plan only corrects its entry.
- **0016-MADR's remaining decisions:** re-checked against `v0.17.0` before 0016's own PLAN is written.
- **A streaming `WithRetry`:** a request to go-llmprovider-sdk, the owner's, in that repository.
- **`check_libs.py` in CI:** it needs the network and the libraries' advisories; whether it runs on a schedule is the owner's choice, after it has run by hand on a few tags.
