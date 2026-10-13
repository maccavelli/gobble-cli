---
status: completed
date: 2026-10-13
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# PLAN 0019 — Pin acp-go-sdk2 v0.17.0, go-llmprovider-sdk v1.10.2 and go-selfupdate-lib v1.15.2, and integrate each where the library now does the work

Implements [0019-MADR-sibling-libraries-current-pins-and-integration.md](0019-MADR-sibling-libraries-current-pins-and-integration.md) decisions D1–D14, closing findings F1–F13. *(2026-10-13, P3 deviation 2: and D15, closing F14.)*

## Goal

When every phase is committed:

1. `go list -m` reports `github.com/maccavelli/acp-go-sdk2 v0.17.0`, `github.com/maccavelli/go-llmprovider-sdk v1.10.2` and `github.com/maccavelli/go-selfupdate-lib v1.15.2`, and `.github/workflows/ci.yml` calls both go-selfupdate-lib workflows at `436a7f5976a6815491c7384527fef90869574a1b # v1.15.2`. *(P4 deviation 11: `v0.17.1`, once tagged.)*
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
| P3 | `llm/provider/ambient_test.go`, `acpserver/usage_test.go` | `go.mod`, `go.sum`, `llm/provider/provider.go`, `llm/provider/provider_test.go`, `llm/doc.go`, `llm/llm.go`, `acpserver/history.go`, `docs/architecture.md`, `README.md`; *(deviation 2:)* `llm/provider/ambient.go`; `internal/config/keys.go`, `internal/config/settings.go`, `internal/config/settings.yaml`, `internal/config/merge.go`, `internal/config/testdata/settings.schema.json`, `internal/config/resolve_test.go`, `internal/config/render_test.go`; `internal/cli/agent.go`, `internal/cli/line.go`, `internal/cli/print.go`, `internal/cli/settings_test.go`; `docs/decisions/0018-MADR-configuration-surface.md`, `docs/decisions/0005-PLAN-v1-feature-scope.md`, `docs/README.md`, this plan and its MADR |
| P4 | `acpclient/acptest/abs.go`, `acpclient/acptest/abs_test.go`, `acpserver/rewrite.go`, `acpserver/rewrite_test.go`, `acpserver/list_test.go` | `go.mod`, `go.sum`; `acpclient/conn.go`, `acpclient/sessions.go`, `acpclient/conn_test.go`, `acpclient/sessions_test.go`, `acpclient/elicitation_test.go`; `acpclient/acptest/recorder.go`, `acpclient/acptest/script.go`, `acpclient/acptest/pair_test.go`, `acpclient/acptest/script_test.go`; `acpserver/serve.go`, `acpserver/agent.go`, `acpserver/mcp.go`, `acpserver/agent_test.go`, `acpserver/commands_test.go`, `acpserver/fixtures_test.go`, `acpserver/tree_test.go`, `acpserver/delete_test.go`, `acpserver/mcp_test.go`; `internal/cli/line_test.go`; `docs/architecture.md`, `README.md`; *(deviation 4:)* `acpserver/sysprompt.go`; *(deviation 7:)* `docs/decisions/0005-PLAN-v1-feature-scope.md`; *(deviation 10:)* `docs/README.md`; *(deviations 3 to 10:)* this plan and its MADR |
| P5 | — | *(deviation 15:)* `scripts/check_libs.py`; `docs/decisions/0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md`, `docs/decisions/0014-MADR-move-to-acp-go-sdk2.md`, `docs/decisions/0005-MADR-v1-feature-scope.md`, `docs/decisions/0005-PLAN-v1-feature-scope.md`, `docs/decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md`, `docs/reports/0014-REPORT-llmprovider-sdk-integration.md`, `docs/reports/0015-REPORT-sibling-library-pins-and-order.md`, `docs/README.md`, this plan (its execution record) |

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
   - ~~`llm/provider/ambient_test.go` (new), `TestAmbientDefaultModels`: for each of the nine key-taking providers, `Choose` with only that provider's `EnvVar` set returns its id and the model of MADR's `v1.10.2` column, as literals (gemini `gemini-3.5-flash-lite`, openai `gpt-6-luna`, claude `claude-haiku-5-5`, grok `grok-4.6`, opencode-zen and opencode-go `claude-haiku-5-5`, huggingface `deepseek-ai/DeepSeek-V4-Flash-0731`, kilo `deepseek/deepseek-v4.1-flash`, together `zai-org/GLM-5.3-Flash`), and a test fails naming the provider whose default moved. The values are read from the probe of MADR's table, never copied from the test's own output.~~ *(Deviation 2: impossible for opencode-go, which shares `OPENCODE_API_KEY` with opencode-zen and comes after it, so `Choose` never returns it (MADR F14).)*
   - *(Deviation 2, in its place:)* `llm/provider/ambient_test.go` (new):
     - `TestAmbientDefaultModels`: a table of the nine, each `{id, variable, model}` as literals (gemini `GEMINI_API_KEY` `gemini-3.5-flash-lite`, openai `OPENAI_API_KEY` `gpt-6-luna`, claude `ANTHROPIC_API_KEY` `claude-haiku-5-5`, grok `XAI_API_KEY` `grok-4.6`, opencode-zen and opencode-go `OPENCODE_API_KEY` `claude-haiku-5-5`, huggingface `HF_TOKEN` `deepseek-ai/DeepSeek-V4-Flash-0731`, kilo `KILO_API_KEY` `deepseek/deepseek-v4.1-flash`, together `TOGETHER_API_KEY` `zai-org/GLM-5.3-Flash`), read from the probe of MADR's table and from SDK `llmprovider/provider.go:62-74`, never from the test's own output. The table's ids must equal `KeyProviders()`, in order. For each, `Choose(env, id)` with only its variable set returns exactly `Selection{ID: id, Model: model, EnvVar: variable}`, else `t.Errorf` names the id. Then `Choose(env, "")` with only `OPENCODE_API_KEY` set returns opencode-zen (the registry order, F14).
     - `TestChoosePreferred`: `Choose(env, "openai")` with only `OPENCODE_API_KEY` set returns a zero `Selection` and an error for which `IsNoCredential` and `errors.Is(err, llm.ErrAuth)` hold and whose text contains `defaultProvider is openai; set OPENAI_API_KEY`; `Choose(env, "ollama")` and `Choose(env, "nope")`, with every variable set, each return an error naming the value, for which `IsNoCredential` is false.
   - `llm/provider/provider_test.go`: `TestNewReportsRetryCapabilities`: `New("openai", Options{APIKey: "test-only"})` reports `NativeStreaming` false, and `Wrap` of a stub reporting `Supported` reports true. `TestUsageCacheWrite`: a response with `InputTokens 100, CachedTokens 30, CacheWriteTokens 10` gives `CacheReadTokens 30, CacheWriteTokens 10`, and one with only `CacheWriteTokens 5` is not estimated.
   - `acpserver/usage_test.go` (new), `TestPiUsageExcludesCache`: `piUsage(llm.Usage{InputTokens: 100, OutputTokens: 20, CacheReadTokens: 30, CacheWriteTokens: 10})` stores `Input 60, CacheRead 30, CacheWrite 10, TotalTokens 120`, Pi's arithmetic (`openai-completions.ts:1544-1546` at Pi `312184e`); `InputTokens 5, CacheReadTokens 10` stores `Input 0`; an estimated usage still carries `{"estimated":true}`.
9. *(Deviation 2, MADR D15.)* **The `defaultProvider` key:**
   - **`llm/provider/ambient.go`:**
     - `Choose(env func(string) string, preferred string) (Selection, error)`. With `preferred` `""`, today's loop, unchanged. Otherwise the descriptor whose `ID` is `preferred`, `RequiresAPIKey` and a non-empty `EnvVar`: if `env(EnvVar)` is set, `Selection{ID, EnvVar, Model: StaticModels[0]}` (none when the list is empty); if not, `fmt.Errorf("%w: defaultProvider is %s; set %s", ErrNoCredential, preferred, EnvVar)`. No such descriptor: `fmt.Errorf("provider: defaultProvider %q is not a built-in provider that takes a key", preferred)`. The doc comment names 0019-MADR D15 and says it never falls back.
     - `Ambient(env func(string) string, preferred string, o Options)` passes `preferred` to `Choose`.
     - New `KeyProviders() []string`: the ids of the descriptors with `RequiresAPIKey` and an `EnvVar`, in registry order. New `CredentialVar(id string) string`: that id's `EnvVar` when it takes a key, else `""`.
     - `llm/provider/provider_test.go`'s `TestChoose` and `TestAmbientBuilds` call `Choose(…, "")` and `Ambient(…, "", …)`; their assertions are unchanged.
   - **`internal/config`:**
     - `keys.go`: first in `Keys`, `{Path: "defaultProvider", Kind: KindEnum, Enum: []string{"", "gemini", "openai", "claude", "grok", "opencode-zen", "opencode-go", "huggingface", "kilo", "together"}, Read: ReadProcess, Doc: "The provider gobble uses, with its first model; its credential variable must be set. Empty takes the first provider whose credential variable is set."}`; the comment above `Keys` becomes "0018-MADR D14's thirteen and 0019-MADR D15's defaultProvider, in the template's order."
     - `settings.go`: `DefaultProvider string` with `yaml:"defaultProvider" json:"defaultProvider,omitempty"`, first in `Settings`, and its `case` in `field`.
     - `settings.yaml`: a `Model` section ahead of `Sessions`, its banner as the others', holding:

       ```yaml
       # The provider gobble uses: gemini, openai, claude, grok, opencode-zen,
       # opencode-go, huggingface, kilo or together, with its first model.
       # Its credential variable must be set. Empty takes the first of them,
       # in that order, whose variable is set; opencode-zen and opencode-go
       # share OPENCODE_API_KEY, so only this setting reaches opencode-go.
       # defaultProvider: ""
       ```

     - `merge.go`: the enum message writes an empty value as `""` (a helper, `enumList`), so the message reads `must be one of "", gemini, …`, not `must be one of , gemini, …`.
     - `testdata/settings.schema.json`: regenerated with `go run ./internal/config/genschema`, never edited by hand.
     - Tests: `TestKeysMatchSettings` expects 14 keys, its message naming D14's 13 and D15's `defaultProvider`; `TestRenderDocument`'s document gains `defaultProvider: ""` as its first line; `TestDefaultsEqualCode` gains `{"defaultProvider", s.DefaultProvider, ""}`; `TestTypeAndRangeErrorsFallBack`'s file gains line 11, `defaultProvider: ollama`, and its list `settings.yaml:11: defaultProvider: must be one of "", gemini, openai, claude, grok, opencode-zen, opencode-go, huggingface, kilo, together`. The probe found no other test that the key changes.
   - **`internal/cli`:**
     - `agent.go`: `func (e *runEnv) defaultProvider() string`, the process settings' `DefaultProvider`, as `sessionStore` reads `SessionDir`; `modelProvider` calls `provider.Ambient(os.Getenv, e.defaultProvider(), …)`; `modelChoice` becomes a method, `e.modelChoice`, calling `provider.Choose(os.Getenv, e.defaultProvider())`, and `agentOptions` passes `Models: e.modelChoice`; `noCredentialMessage(preferred string)` returns `no model credential: defaultProvider is <id>; set <variable>` when `provider.CredentialVar(preferred)` is not empty, else today's text.
     - `line.go:72`, `:219` and `print.go:197` pass `e.defaultProvider()`.
     - `settings_test.go`, `TestDefaultProviderSetting`: the key's `Enum` equals `""` followed by `provider.KeyProviders()`; with `GOBBLE_HOME` holding `defaultProvider: opencode-go` and only `OPENCODE_API_KEY` set, after `e.processSettings("")` (as `gobble acp` does) `e.modelChoice()` is `{Provider: "opencode-go", Default: "claude-haiku-5-5", Models: provider.Models("opencode-go")}` and `modelProvider(e)()` builds a provider whose `ID()` is `opencode-go`; with `defaultProvider: openai` and only `OPENCODE_API_KEY` set, `-p x` exits `ExitAuth` with `Error: no model credential: defaultProvider is openai; set OPENAI_API_KEY` on stderr.
   - **Docs:** `docs/architecture.md:43`, `llm/provider`'s row, "with the ambient credential selection" becomes "with the ambient credential selection and the `defaultProvider` setting".
   - **Records:** 0018-MADR D19's F4 line (`:566`) gets a dated note that `defaultProvider` came forward in this phase and `defaultModel` and the model scope stay F4's; 0005-PLAN F4-2's "Choosing a model" (`:3274`) gets a dated note that F4-2 adds `defaultModel` and the model scope beside `defaultProvider`, and that `--provider` wins over it for its run; `docs/README.md`'s rows for 0005-PLAN, 0018-MADR, 0019-MADR and this plan name the amendment.

**P3's verification:**

```text
go list -m github.com/maccavelli/go-llmprovider-sdk                         -> v1.10.2
git diff -U0 HEAD -- go.mod | grep -c '^[-+]\s'                             -> 2
go test -count=1 ./llm/... ./acpserver/... ./internal/cli/...               -> ok
go test -count=1 ./llm/provider -run 'TestAmbientDefaultModels|TestNewReportsRetryCapabilities|TestUsageCacheWrite' -v -> each PASS
go test -count=1 ./acpserver -run TestPiUsageExcludesCache -v               -> PASS
git diff --exit-code acpserver/testdata/session.golden                      -> no diff
go test -count=1 ./llm/provider -run 'TestAmbientDefaultModels|TestChoosePreferred' -v   -> each PASS   (deviation 2)
go test -count=1 ./internal/config/... ./internal/cli/...                   -> ok                  (deviation 2)
go test -count=1 ./internal/cli -run TestDefaultProviderSetting -v          -> PASS                (deviation 2)
go run ./internal/config/genschema && git diff --exit-code internal/config/testdata/settings.schema.json -> no diff after staging
```

**P3's negatives,** on scratch copies: `New` reporting `p.Capabilities()` fails `TestNewReportsRetryCapabilities`; `usageOf` dropping `CacheWriteTokens` fails `TestUsageCacheWrite`; `piUsage` writing `Input: u.InputTokens` fails `TestPiUsageExcludesCache`; one literal of `TestAmbientDefaultModels` changed to `v1.2.1`'s value fails it, naming the provider. *(Deviation 2, also:)* `Choose` ignoring `preferred` fails `TestAmbientDefaultModels` on opencode-go and `TestChoosePreferred`; `Choose` falling back to the ambient loop when the preferred variable is unset fails `TestChoosePreferred`; `modelChoice` calling `Choose(os.Getenv, "")` fails `TestDefaultProviderSetting`; `"ollama"` added to the enum fails `TestDefaultProviderSetting`'s enum check; `enumList` dropped (the old `strings.Join`) fails `TestTypeAndRangeErrorsFallBack`.

### P3's deviations, 2026-10-13, decided by the owner

- **Deviation 2. The ambient choice cannot reach opencode-go, so step 8's default-model test was impossible as written.**
  - **Found:** writing `TestAmbientDefaultModels`. In go-llmprovider-sdk, opencode-zen and opencode-go share `OPENCODE_API_KEY` (`llmprovider/provider.go:69-70` at `v1.10.2`; `:63-64` at `v1.2.1`, the current pin, so it predates this phase), and opencode-zen comes first in the registry (`providers/providers.go:34-35`), so `Choose` with that variable set always returns opencode-zen (`llm/provider/ambient.go:27-37`). `Ambient` and `Choose` are gobble's only provider choosers (`internal/cli/agent.go:30`, `:82`; `internal/cli/line.go:72`), and `--provider` is refused until 0005-PLAN F4 (`internal/cli/flags.go:33`, `:69`). Read from both tags' sources in the module cache; MADR F14.
  - **Decided** ("make opencode-go selectable", then "defaultProvider setting"): the owner chose a real fix over pinning opencode-go's default through its descriptor alone, and the `defaultProvider` setting over the `--provider` and `--model` flags, since the setting reaches `gobble acp` too. MADR D15; step 8's first bullet is replaced, step 9 added, and the files in the scope table's P3 row marked "deviation 2" added. A scratch clone of `e8181b5` with the key added (`q461_defaultprovider_probe.py`) found exactly two tests the key changes, `TestRenderDocument` and `TestKeysMatchSettings`, both extended in step 9, never loosened (C3).

### P4 — acp-go-sdk2 `v0.17.0` (D3–D9; closes F2–F7)

1. `python scripts/check_libs.py --lib acp-go-sdk2` reports `v0.17.0` with no advisory and no new module; its test step fails with the 27 tests of MADR's table, which this phase fixes. Its 17 incompatible changes are reported, none in gobble's use. *(Deviation 11: rerun at `v0.17.1`, once tagged.)*
2. `go get github.com/maccavelli/acp-go-sdk2@v0.17.0`, `go mod tidy`; `go.sum` holds `h1:d+ZASkLAYayy4xPtvaLEQShRy6BYoX21RbZsFSjNus8=` and `h1:KqDRvShTbZEOpKOQOyRUnbeFqlyH20LXi7k3UOsgv0I=`. *(Deviation 11: `v0.17.1`, with its sums recorded here when it is pinned. Pinned 2026-10-13: `go.sum` holds `h1:G/MDUnda+mOgv5qNfVvoSkUDqAFNuCE2X9vPvui5gMo=` and, unchanged from `v0.17.0`, `h1:KqDRvShTbZEOpKOQOyRUnbeFqlyH20LXi7k3UOsgv0I=`.)*
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
     - `acpserver/agent_test.go:78`; `acpserver/commands_test.go:216`; `acpserver/fixtures_test.go:27` (`piCwd` becomes a `var`); `acpserver/tree_test.go:119`'s header line, built with the same helper; *(deviation 3:)* `acpserver/tree_test.go:49`, `treeAgent`'s resume, `Cwd: piCwd`;
     - `internal/cli/line_test.go:70` (the harness's default `cwd`).
   - `acpserver/settings_test.go:112`, `:124` and `acpserver/agent_test.go:180` call `Agent` methods directly, which the SDK does not validate, and stay.
4. **D4, host-independent goldens.** `acpclient/acptest/recorder.go`, `Golden`: before comparing or writing, the transcript's JSON strings that begin with the JSON-escaped form of `Abs("/")` are rewritten to their slash form (`"C:\\work"` → `"/work"`, `"C:\\work\\a.go"` → `"/work/a.go"`), by one `regexp` over string literals; off Windows `Abs("/")` is `/` and the rewrite is a no-op. `session.golden` is not regenerated; it must pass unchanged on all three systems. *(Deviation 4: the rewrite is not enough on Windows, since the golden's first `usage_update` estimates the prompt, whose cwd is two bytes longer there; `PromptOptions` gains the unexported `cwd`, and `TestSession` sets it to `/work`.)*
5. **D5, `acpserver/agent.go`, `ListSessions` (`:822-828`):** each stored row is built, then checked with `info.Validate()`; a row that fails is left out and logged once per session id, `a.logger.WarnContext(ctx, "session: not listed: its cwd is not absolute on this host", slog.String("session", …), slog.String("cwd", …))`, and the loop goes on. Live sessions are always absolute (their `cwd` came through the SDK), so they are not checked.
6. **D6, `acpserver/rewrite.go`** (new): `resolveBareCommands(ctx context.Context, method string, params any) error`. For `*acp.NewSessionRequest`, `*acp.LoadSessionRequest` and `*acp.ResumeSessionRequest`, each `McpServers[i].Stdio` whose `Command` is not `filepath.IsAbs` is resolved with `exec.LookPath`; a result that is absolute replaces `Command`; an error, or a relative result (`exec.ErrDot`), leaves it as it is, for the SDK to refuse. It never returns an error. `acpserver/serve.go:32` passes `acp.WithInboundRewrite(resolveBareCommands)` beside `acp.WithLogger(logger)`.
7. **D7, the checks the SDK owns:**
   - `acpclient/conn.go:125-127`: the version comparison is removed. After `Initialize`, an error that is a `*acp.ProtocolVersionError` is reported as `fmt.Errorf("agent speaks ACP protocol %d, want %d", pve.Answered, pve.Requested)`, so the message users see is unchanged.
   - `acpserver/mcp.go:111-114`: the `r.Sse` and `r.Acp` cases are removed; the `"an MCP server has no transport"` fallback stays.
   - `acpserver/agent.go:736-738`: the empty-`cwd` fallback is removed; `open` uses the request's `cwd`.
8. **D8, the tests the gates pre-empt** (C2):
   - `acpclient/acptest/script.go`: `ScriptAgent` gains `Capabilities *acp.AgentCapabilities`; `Initialize` answers it when non-nil, and today's value (`Image` from `a.Image`, `Close`) when nil.
   - `acpclient/sessions_test.go`, `TestAdditionalDirectoriesSent`: its agent advertises `AdditionalDirectories` and `Resume`; a second case with today's advertisement asserts `NewSessionWith` with directories fails with `*acp.CapabilityError{Method: "session/new", Capability: "sessionCapabilities.additionalDirectories"}` and that the agent recorded no request.
   - `acpclient/elicitation_test.go`, `TestElicitationNotFound`: the typed `conn.CreateElicitation` fails with a `*acp.CapabilityError` whose `Capability` is `elicitation.url`; a raw `acp.SendRequest` of the same request is answered -32602 by the client's SDK; gobble's client records no elicitation. ~~`elicitation/complete` and the later `NewSession` stay as they are.~~ *(Deviation 5: the SDK gates `elicitation/complete` as well; the typed call must fail with `*acp.CapabilityError` for `elicitation.url`, the notification is then sent raw, and the later `NewSession` stays.)*
   - `acpserver/delete_test.go`, `TestDeleteSessionNotFound`: the advertisement check stays; the typed `DeleteSession` fails with `*acp.CapabilityError{Method: "session/delete", Capability: "sessionCapabilities.delete"}`; a raw `acp.SendRequest` of `session/delete` is answered -32601, the original assertion, now through the agent's SDK.
   - `acpserver/mcp_test.go`, `TestUnofferedTransportsRefused`: the SSE and ACP servers are sent raw (`acp.SendRequest`, `session/new`) and answered -32602 with the SDK's `mcpCapabilities` reason; the stdio case keeps its bad name (`"bad name"`) with `Command: acptest.Abs("/bin/x")`, so gobble's own name check is what refuses it, -32602; the last assertion checks the SDK's text, `mcpCapabilities.sse`, in place of gobble's removed one.
   - `acpclient/conn_test.go`, `TestStartRefusesOtherProtocols`: unchanged; its message is kept by D7's mapping.
   - `internal/cli/line_test.go:163-168`, `TestLineSessionFirstPrompt`: after `waitUntil`, `if t.Failed() { t.FailNow() }`, so a failed wait stops instead of indexing an empty slice.
9. **D6's and D5's tests:**
   - `acpserver/commands_test.go`: `TestMcremoteScript`'s pipes and `call` move into `rawServe(t, Options) (call func(id int, method, params string) string, stop func())`, used by it and below.
   - `acpserver/rewrite_test.go` (new): `TestResolveBareCommands` sets `PATH` to a temp directory holding an executable `gobble-fake-mcp` (`.exe` on Windows; mode 0o755); a `NewSessionRequest` whose stdio command is `gobble-fake-mcp` is rewritten to its absolute path, one with `no-such-command-0019` is left alone, and an absolute command is left alone. `TestServeResolvesBareCommand`, through `rawServe`: `session/new` with the bare `gobble-fake-mcp` is answered with a session, and with `no-such-command-0019` is answered -32602 with `is not an absolute path`.
   - `acpserver/list_test.go` (new), `TestListSkipsNonAbsoluteCwd`: a `jsonl` store holds two sessions, written through `Store.Create` and one `Append` each, with header `cwd` `t.TempDir()` and `relative/dir`; `session/list` answers one session, the absolute one, and the log names the other once.
10. **D9, `acpclient/sessions.go`, `sessionError` (`:40-45`):** the reason is `data.reason`, else `data.error` (the SDK's -32602 detail), else the message. `acpclient/conn_test.go` gains `TestInvalidParamsCarriesReason`: an agent answering -32602 with `{"error":"NewSessionRequest: cwd: \"x\" is not an absolute path"}` gives an `ErrInvalidSession` whose text contains `cwd: "x" is not an absolute path`.
11. **Docs:** `docs/architecture.md:108` and `README.md:82` name acp-go-sdk2 `v0.14.0`; each becomes `v0.17.0`. *(Deviation 11: `v0.17.1`.)*
12. **magic-cli-remote** (D3; the owner's answer of 2026-10-12): with the staged tree built (`go build -o <scratch>/gobble.exe ./cmd/gobble`), magic-cli-remote's ACP client runs a session against `<scratch>/gobble.exe acp`: initialize, `session/new` in a temp directory, one prompt, close. Its transcript is kept; any capability, validation or -32602 error stops the phase as a deviation.

**P4's verification:**

```text
go list -m github.com/maccavelli/acp-go-sdk2                                -> v0.17.0   (deviation 11: v0.17.1)
git diff -U0 HEAD -- go.mod | grep -c '^[-+]\s'                             -> 2
go test -count=1 ./...                                                      -> ok on Windows; the gates run it on the second Linux machine and the macOS laptop
git diff --exit-code acpserver/testdata/session.golden                      -> no diff
grep -n 'ProtocolVersion != acp.ProtocolVersionNumber' acpclient/conn.go    -> no match
grep -n 'the SSE transport is not offered\|the ACP transport is not offered' acpserver/mcp.go -> no match
grep -rn 'Cwd: "/w\|"/work' --include=*_test.go acpclient acpserver internal/cli -> only acptest.Abs arguments
```

**P4's negatives,** on scratch copies: `Abs` returning `p` on Windows fails 22 tests there (the path set of MADR's table); `Golden` without its rewrite fails `TestSession` on Windows; `ListSessions` without the `Validate` check fails `TestListSkipsNonAbsoluteCwd`; `Serve` without `WithInboundRewrite` fails `TestServeResolvesBareCommand`; `resolveBareCommands` keeping a relative `LookPath` result fails `TestResolveBareCommands`; `sessionError` without `data.error` fails `TestInvalidParamsCarriesReason`; `TestDeleteSessionNotFound`'s raw request sent to an agent that implements `DeleteSession` fails it. *(Deviation 8: the relative-result negative fails `TestResolveBareCommands` only through its `execerrdot=0` case. Deviation 9: the agent must advertise deletion as well as implement it, since the SDK answers an unadvertised method first.)*

### P4's deviations, 2026-10-13, decided by the owner

- **Deviation 3. Step 3's inventory missed a fixture site.**
  - **Found:** after step 3 and 4, on Windows, `TestCloneKeepsLabels`, `TestForkDropsOffPathLabels` and `TestPromptFollowsTheLeaf` failed with -32602, `ResumeSessionRequest: cwd: "/work/project" is not an absolute path` (`tree_test.go:88`, `:104`, `:151`). `treeAgent` (`acpserver/tree_test.go:49`) resumes with `Cwd: h.Cwd`, the Pi fixture header's POSIX cwd, which the inventory does not list. `acptest.Abs(h.Cwd)` is wrong there: the `:119` header step 3 makes absolute would be made absolute twice, which failed `TestCloneMovesFirstKeptPastLabel` on a scratch copy.
  - **Decided** ("Cwd: piCwd"): `treeAgent` resumes with `Cwd: piCwd`, the host-absolute `/work/project` every one of its sessions is written at. Proved on a scratch copy (`q471_p4_golden_probe.py`): the four tree tests pass. No file added.
- **Deviation 4. The golden's usage estimate counts the prompt's cwd, so step 4's path rewrite alone cannot keep `session.golden` byte-identical on Windows.**
  - **Found:** after step 4, `TestSession` on Windows differed from the golden at line 6 only, `session/new`'s `usage_update`: `"used":151` against `150`. The estimate is the system prompt at four bytes a token, and the prompt's cwd section writes the cwd in slash form (`prompt/sections.go:134`): `C:/work` on Windows, `/work` elsewhere, two bytes more. `TestSession` already pins the prompt's `GOOS` to linux for the same reason (`acpserver/sysprompt.go:30-32`, 0005-PLAN F3-1 deviation 5).
  - **Decided** ("Test-only prompt cwd"): `PromptOptions` gains an unexported field, `cwd`, the cwd the prompt names in place of the session's, for tests only; `sections` passes `cmp.Or(o.cwd, s.cwd)`; `TestSession` sets `cwd: "/work"` beside `GOOS: "linux"`. The golden stays byte-identical and pins the prompt's size on every host; the field cannot be set outside the package. `acpserver/sysprompt.go` is added to P4's scope. Proved on the same scratch copy: `TestSession` passes on Windows.

- **Deviation 5. `v0.17.0` gates `elicitation/complete` too, so step 8's "stay as they are" was impossible.**
  - **Found:** with the create half rewritten, the typed `conn.CompleteElicitation` failed with `elicitation/complete: capability elicitation.url not advertised` (`elicitation_test.go:86`): the SDK gates the notification as it gates the request (SDK `agent_gen.go:633`). Writing the create half also showed that the SDK refuses to encode an unscoped URL elicitation ("CreateElicitationUrl must have exactly one member set"), so the request is scoped to a session, `Session: &acp.ElicitationSessionScope{SessionId: "s1"}`, and only the client's advertisement refuses it, as MADR F7 describes.
  - **Decided** ("Typed refusal + raw send"): the typed `CompleteElicitation` must fail with `*acp.CapabilityError{Method: "elicitation/complete", Capability: "elicitation.url"}`; the same notification is then sent raw through `conn.Connection().SendNotification`, and the later `NewSession` shows gobble's client ignored it and the connection goes on. No file added.

- **Deviation 6. Step 12's model turn could not run: the test key is rejected by the provider.**
  - **Found:** magic-cli-remote's client (a build-tagged test in a scratch clone of magic-cli-remote at `9b75e5f3`, on the pattern of its `live_runtime_test.go`; `q477_p4_mcr_session.py`) ran `initialize`, `session/new` in a temp directory, a prompt and close against `gobble.exe acp` with no capability, validation or -32602 error; the prompt was answered `auth_required`, which magic-cli-remote reports as "The model provider rejected the agent's credentials". The key in the owner's test file answers 401 `invalid_api_key` at `GET /v1/models` (`q479_key_status.py`, which prints the status only). So a model reply's chunks and its `usage_update` did not cross magic-cli-remote. A `/session` prompt, which gobble answers itself, streamed 370 bytes of agent text and completed. The harness's own first version passed on the refused turn, since it failed only on `error` events and stopped reading at `turn_complete`; it was fixed to fail on `turn_complete` with status `error` and to read on to the error event, and failed as it should.
  - **Corrected** (2026-10-13): the key was not rejected; the harness was wrong. The owner's test file holds an assignment, `OPENAI_API_KEY=<key>`, and `q477` and `q479` sent the whole line as the key, which the provider rightly answers 401. The value alone answers 200 (`q490_key_assignment.py`, which prints the status, the variable name and the value's length only). The harness now reads the value, and the model turn is asserted, no longer only logged.
  - **Decided** ("Rerun with a valid key"): step 12 is rerun with a working key before P4 is staged; the `/session` turn stays in the harness. No file added.
  - **Done** (2026-10-13, against `v0.17.1`): the session ran `initialize`, `session/new`, `/session` (370 bytes, no error), a model prompt answered "ok" with no error, and close; the transcript holds `available_commands`, `session_mode` ×2, `usage_update` ×2, `session_capabilities`, `session_config`, `session_status` ×4, `user_message` ×2, `assistant_message_chunk` ×2 and `turn_complete` ×2, so a model reply's chunks and its `usage_update` crossed magic-cli-remote. The key is in no output. Seen failing: with a key the provider rejects (`--bad-key`), the test fails on the model turn's auth error.
- **Deviation 7. A rejected key is reported as no key (pre-existing).**
  - **Found:** with a key set that the provider rejects, `gobble -p` exits 3 with `Error: no model credential: set one of GEMINI_API_KEY, OPENAI_API_KEY, …`, telling the user to set what they set: both cases reach the CLI as `acpclient.ErrAuthRequired`, and `internal/cli/print.go:197` and `line.go:219` print `noCredentialMessage`. Reproduced at HEAD `a7de4a8` in a scratch clone (`q480_head_rejected_key.py`), so P4 does not cause it.
  - **Corrected** (2026-10-13): the rejected key in these probes was the test file's whole `OPENAI_API_KEY=<key>` line (deviation 6's correction), which the provider rejects as any wrong key; the defect stands, since gobble reports any rejected key as no key.
  - **Decided** ("Note it for F4-3"): 0005-PLAN F4-3, which owns the credential order and its messages, gets a dated note naming the defect and its fix; P4 stays one library (C1). `docs/decisions/0005-PLAN-v1-feature-scope.md` is added to P4's scope for that note.

- **Deviation 8. The guard against a relative `LookPath` result was untested, so its negative was missed.**
  - **Found:** P4's negatives (`q482_p4_negatives.py`) dropped `&& filepath.IsAbs(path)` from `rewrite.go`, and `TestResolveBareCommands` still passed. By default `exec.LookPath` never returns a relative path with a nil error, only with `exec.ErrDot`, so the guard matters only under `GODEBUG=execerrdot=0`, which a user may set in gobble's environment. A scratch probe (`q483-probe`) showed that `t.Setenv("GODEBUG", "execerrdot=0")` with `PATH` `.` makes `LookPath` return `gobble-rel-mcp.exe` with a nil error.
  - **Decided** ("Add the execerrdot=0 case"): `TestResolveBareCommands` gains a case that changes into a temp directory holding `gobble-rel-mcp`, sets `PATH` to `.` and `GODEBUG` to `execerrdot=0`, and asserts the command is left as it is. Negative 5 must then fail it. No file added.
- **Deviation 9. `TestDeleteSessionNotFound`'s negative was aimed at the wrong mutation.**
  - **Found:** making gobble's `DeleteSession` handler answer success left the test passing: with deletion unadvertised, the agent's SDK answers the raw `session/delete` -32601 before any handler runs, so the raw assertion pins the SDK's gate, which a handler change cannot reach. gobble can break it only by advertising deletion, which the test's first check catches.
  - **Decided** ("Re-aim the negative"): the test stays as written; the negative becomes "the agent advertises and implements deletion" (`SessionCapabilities.Delete` advertised, the handler answering success), which must fail it. No file added.

- **Deviation 10. P4's records need their index rows.**
  - **Found:** P4 amends 0019-MADR (D4's note), this plan (deviations 3 to 9) and 0005-PLAN (the F4-3 note), and `AGENTS.md` requires `docs/README.md` to index every record in the same change; P4's file list did not name it.
  - **Decided** ("Update them in P4"): `docs/README.md` is added to P4's scope, and the three rows name the amendments; P5 still writes the final rows.

- **Deviation 11. acp-go-sdk2 `v0.17.0` races its own constructor (MADR F15).**
  - **Found:** the gates (`q428_gates_0019.py p4`): lint ×3, Windows `go test` (31 ok), Windows preflight and the macOS laptop's `go test` (31 ok) passed; `-race` failed on Windows (29 of 31; `acpclient`, `internal/cli`) and on the second Linux machine (29 of 31, one run a nil-pointer panic in `checkInbound`). Every report is the pair `agent_gen.go:33` (the constructor's assignment) and `:214` (the `initialize` handler's read), in the SDK's own files at `v0.17.0`; no gobble line is in either stack's racing access.
  - **Decided** ("Fix in acp-go-sdk2, pin v0.17.1"): the owner fixes the SDK — the connection is built without starting its reader, `asc.conn` and `csc.conn` are assigned, then reading starts, with a `-race` test that sends `initialize` at once — and tags `v0.17.1`. P4 then reruns step 1 at `v0.17.1`, pins it with its sums (recorded here when known), and reruns the gates; P4 waits, unstaged, until then. gobble does not guard around the SDK (0004-MADR). Steps 1, 2 and Goal 1 are annotated; no file added.
  - **Done** (2026-10-13): the owner tagged `v0.17.1` (`c77f25a45e33`) and closed the SDK issue. Step 1 rerun: `v0.17.1`, no advisory, no other module moved, build and vet ×3 ok; the test step fails with the same 27 tests as at `v0.17.0` (the two runs' failure names compared, no difference), since the clone is of HEAD without this phase's code; apidiff `v0.17.0`→`v0.17.1` 0 incompatible changes. Pinned with the sums in step 2's note; `go.mod` changes in 2 lines. Step 11 and the verification lines name `v0.17.1`; P5 step 2 and the 0016 deferral are annotated. Negatives rerun: 15 of 15 caught after the control. Gates rerun (`q428_gates_0019.py p4`): lint ×3 0 issues; Windows `go test` and `-race`, the second Linux machine's `-race` and the macOS laptop's `go test` 31 ok each, with no race report and no failure; Windows preflight passed.

### P5 — The records (D10–D13)

Each note is dated 2026-10-12 (or the day P5 runs), cites 0019-MADR's decision and the commit of the phase that carried it, and is added beside the text it corrects, which stays.

1. **`docs/decisions/0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md`:** the note of D10, under its frontmatter's first heading and at D1, D2, D3 and D7.
2. **`docs/decisions/0014-MADR-move-to-acp-go-sdk2.md`:** at F7 (`:48`), "`v0.17.0` is pinned by 0019-MADR D3"; at its Confirmation's -32601 lines for the delete and elicitation tests, "now asserted through the SDK's gates (0019-PLAN P4 step 8)". *(P4 deviation 11: the note names `v0.17.1`.)*
3. **`docs/decisions/0005-MADR-v1-feature-scope.md`:** at `:268`, "five raw binaries"; at `:587` and `:608`, "go-core-lib is go-selfupdate-lib, renamed 2026-10-03"; at `:598`, "fixed in go-selfupdate-lib `v1.5.1` (its 0010-MADR B5, B6)"; at `:966`, "superseded at go-llmprovider-sdk `v1.9.0`: `catalog.ModelInfo` (0019-MADR F9)". *(Deviation 12: "renamed 2026-10-02 in `v1.5.0`".)*
4. **`docs/decisions/0005-PLAN-v1-feature-scope.md`:** at the F4-1 amendment entry (`:3342`), "re-based on `v1.10.2` by 0019-MADR D2; the pin and a true `NativeStreaming` landed in 0019-PLAN P3, and F4-1a keeps the tool result's error flag and the wire tests"; at F4-2's entry, "re-made on `v1.9.0`'s `catalog.ModelInfo` (0019-MADR D11)"; at F4-3's, "the SDK's `TokenStore`, `RefreshLocker` and logins; terminal auth advertised before offered (0019-MADR D11, F6)"; at F10's (`:342`, `:374`, `:640`), D12's three corrections and "the newest go-selfupdate-lib at F10".
5. **`docs/decisions/0012-PLAN-tool-schemas-descriptions-and-loading.md`:** at `:1386`, D11's amendment of P7: `Strict` stays `nil` except for providers whose live test has shown `true` is accepted.
6. **`docs/reports/0014-REPORT-llmprovider-sdk-integration.md`:** at the gap table (`:279-288`), "strict and the error bit at `v1.7.0`, native streaming at `v1.6.0`, cache writes at `v1.10.0`; still open: deferral, `cache_control`, images, a generic Chat Completions provider (0019-MADR F9)".
7. **`docs/reports/0015-REPORT-sibling-library-pins-and-order.md`:** at `:163-166`, the three corrected defaults, measured by running the SDK (0019-MADR's table); at F2's and F6's `v1.8.0` metadata, "`v1.9.0`".
8. **`docs/README.md`:** the 0019 rows read `accepted` and `completed (<date>)`; the build-order list gains `**0019-PLAN.** Run: P1–P5 on <date>.` *(Deviation 14: and 0014-REPORT's missing row.)*
9. **This plan:** `status: completed`, and its `## Execution record (<date>)`: each phase's commit, gates and negatives, and what this plan predicted wrongly.

**P5's verification:**

```text
make check-records && make markdownlint                                     -> ok, 0 issues
python -I <scratch>/q261_lint_records.py --negative <every record P5 touches> -> only the planted violation
grep -c '0019-MADR' <each record of steps 1-7>                              -> at least 1 each
```

### P5's deviations, 2026-10-13

- **Deviation 12. Step 3's rename date is wrong.**
  - **Found:** go-selfupdate-lib's history has the module rename as `6deaa52`, "chore(module): rename go-core-lib to go-selfupdate-lib", dated 2026-10-02 and first released in `v1.5.0`, tagged 2026-10-02; `v1.4.1` is the last tag before it. Step 3's note text says 2026-10-03, the date of `v1.5.1`.
  - **Decided** ("Correct date and tag"): the notes read "renamed 2026-10-02 in `v1.5.0`". Step 3 is annotated. No file added.
- **Deviation 13. go-llmprovider-sdk `v1.11.0` was tagged after P3.**
  - **Found:** `v1.11.0` (`d1fbad55`) was tagged 2026-10-13 at 09:54 (UTC-5), after P3 pinned `v1.10.2`, so the whole-plan check "every library at its latest tag" cannot hold. `check_libs.py --lib go-llmprovider-sdk` passes at `v1.11.0`, every check ok; apidiff reports two additions, `catalog.WithLimit` and `wizard.Options.ModelLimit`. Its changed default, a listing's full curated catalog, reaches no gobble code, since gobble lists no models (gobble-cli issue 1).
  - **Decided** ("Record it; 0020 pins"): 0019 closes at `v1.10.2`, and the next record pins `v1.11.0` with issue 1's listing. The whole-plan verification line is annotated. No file added.
- **Deviation 14. 0014-REPORT has no row in the index.**
  - **Found:** `docs/reports/0014-REPORT-llmprovider-sdk-integration.md` is the only record `docs/README.md` does not link, since its commit `ad6ea57` (2026-10-11); `make check-records` does not check the index. P5 adds a note to it.
  - **Decided** ("Add its row in P5"): the index gains its row. No file added; the index is step 8's.
- **Deviation 15. `check_libs.py` crashes when a library is pinned at its latest tag (pre-existing).**
  - **Found:** the whole-plan `python scripts/check_libs.py` aborts at acp-go-sdk2, pinned at `v0.17.1`, its latest tag: `check_library` exports the API of the base and of the tag, both `v0.17.1`, and the second `mkdir` of `api-v0.17.1` raises `FileExistsError` (`scripts/check_libs.py:182`, called at `:223`). That is not the `RuntimeError` the step catches, so the run ends and the other two libraries are never checked. The script is unchanged since P1's `8b0bd54`; every earlier run had a newer tag. Unfixed, every full run crashes whenever any library is at its latest tag, the normal state after a pin.
  - **Decided** ("Skip API diff when pinned = tag"): the API step moves into `check_api`, which reports `api: pinned at <tag>; nothing to compare` and exports nothing when the base is the tag. A `--self-test` case runs that branch offline, seen failing on a scratch copy without it, and `--lib acp-go-sdk2` is run before and after. `scripts/check_libs.py` is added to P5's scope.
  - **Done** (2026-10-13): `--self-test` 9 of 9 ok; on a scratch copy without the branch the new case alone fails (8 ok, 1 FAIL), creating nothing. The full run, which crashed before, exits 0: acp-go-sdk2 and go-selfupdate-lib report `pinned at` their tags, and go-llmprovider-sdk passes every check at `v1.11.0`. The run checked the full set where `--lib acp-go-sdk2` alone was planned, since go-selfupdate-lib is pinned at its tag too.

## Verification (whole plan)

After P5:

```text
go list -m github.com/maccavelli/acp-go-sdk2 github.com/maccavelli/go-llmprovider-sdk github.com/maccavelli/go-selfupdate-lib
                                                   -> v0.17.0, v1.10.2, v1.15.2   (deviation 11: v0.17.1)
git diff 108d9f0 -- go.mod | grep -c '^[-+]\s'     -> 6
git diff --exit-code 108d9f0 -- acpserver/testdata/session.golden -> no diff
python scripts/check_libs.py                       -> every library at its latest tag; exit 0   (deviation 13: go-llmprovider-sdk v1.11.0, tagged after P3, is the next record's)
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
| A14 | *(2026-10-13, P3 deviation 2)* `defaultProvider` chooses each of the nine, opencode-go included, in every mode, and a set preference without its variable is an error naming it, never another provider | D15, F14 |

**A9 is the criterion most likely to be dropped.** Each rewritten test would also go green with a bare `err != nil`, and under time pressure that is the shortest path; C2 and P4 step 8 name what each must assert instead.

## Rollout and Rollback

- **Rollout:** one commit per phase, in order; P3 before 2026-10-23.
- **Rollback:** a phase reverts with `git revert` of its commit. P2, P3 and P4 are independent of each other's code, since each changes only its library's line and the code that library's change needs; P1's script is unaffected by any of them. P5 reverts as records only.

## Deferred (named, so they are not mistaken for oversights)

- **The entries D11 re-bases** (0012-PLAN P7, 0005-PLAN F4-1a's remainder, F4-2, F4-3, F4-1b): each runs as its own phase, on the newest tag that `check_libs.py` passes then.
- **`gobble update`** (F10): last in 0005-PLAN's order; this plan only corrects its entry.
- **0016-MADR's remaining decisions:** re-checked against `v0.17.0` before 0016's own PLAN is written. *(P4 deviation 11: against `v0.17.1`.)*
- **A streaming `WithRetry`:** a request to go-llmprovider-sdk, the owner's, in that repository.
- **`check_libs.py` in CI:** it needs the network and the libraries' advisories; whether it runs on a schedule is the owner's choice, after it has run by hand on a few tags.

## Execution record (2026-10-13)

Every phase ran, in order, with 0018-PLAN P7 between P1 and P2. The agent staged each phase; the owner committed each with `git commit --no-edit`.

| Phase | Commit | Landed |
| :--- | :--- | :--- |
| P1 | `8b0bd54` | `scripts/check_libs.py` and `make check-libs` (D14) |
| 0018-PLAN P7 | `ca314b8` | the edit lock's deadline, run before P2 at the owner's direction (0018-PLAN deviation 24) |
| P2 | `e8181b5` | go-selfupdate-lib `v1.15.2` and the two workflow `uses:` lines (D1) |
| P3 | `a7de4a8` | go-llmprovider-sdk `v1.10.2` (D2) and the `defaultProvider` setting (D15) |
| P4 | `17cd497` | acp-go-sdk2 `v0.17.1` (D3) and D4–D9 |
| P5 | the commit of these records | D10–D13's notes, deviations 12–15, `check_libs.py`'s pinned-at-the-tag fix, and this record |

**Gates.** Each phase's run of `q428_gates_0019.py`: lint for linux, darwin and windows, 0 issues; Windows `go test` and `-race`, the second Linux machine's `-race` and the macOS laptop's `go test`, 31 packages ok each, with no failure and no race report; Windows preflight passed. P4's first run failed `-race` on Windows and on Linux, 29 of 31 each, on the SDK's constructor race (deviation 11); its rerun on `v0.17.1` is the green one. Each staged index passed `q252_check_index.py` (gofmt, vet, test and check-records on the index alone); P1's first index run failed on `mcpclient`'s `TestAddConcurrent`, which 0018-PLAN P7 fixed.

**Negatives,** each on a scratch copy after a control run passed: P1 4 of 4 (3 of 4 before deviation 1); 0018-PLAN P7 3 of 3; P3 12 of 12; P4 15 of 15 (13 of 15 before deviations 8 and 9), and magic-cli-remote's session test failed with a key the provider rejects; P5's self-test case failed on a copy without its branch. P2 adds no check.

**magic-cli-remote** (step 12, A11): its ACP client ran `initialize`, `session/new`, `/session`, a model turn answered "ok", and close, against P4's agent on `v0.17.1`, with no capability, validation or -32602 error.

**Deviations:** 15, each the owner's decision, recorded under its phase: 1 (P1), 2 (P3), 3–11 (P4), 12–15 (P5).

**What this plan predicted wrongly:**

1. P1's negative 3: `go get` of go-selfupdate-lib `v1.5.0` and `go mod tidy` moved it back to `v1.15.2`, since `v1.5.0` lacks a package gobble's tests import, so the check tested another version until it checked the version after `tidy` (deviation 1).
2. The research's `vet` failure at `v1.5.0` came from running no `tidy`, not from the library.
3. P3 step 8 assumed `Choose` can reach all nine key-taking providers; opencode-go shares `OPENCODE_API_KEY` with opencode-zen and comes after it (deviation 2).
4. P4's inventory of POSIX paths missed `tree_test.go:49` (deviation 3).
5. That the path rewrite alone keeps the golden: its first usage estimate counts the prompt, whose `cwd` is longer on Windows (deviation 4).
6. That `elicitation/complete` is ungated (deviation 5).
7. Negatives 5 and 7 as worded: `LookPath` refuses a relative result by default, and the SDK answers an unadvertised method before gobble's handler (deviations 8, 9).
8. Negative 1's count: 32 tests fail with `Abs` off on Windows, not 22; the plan counted the table's path set only.
9. That the race gates pass on `v0.17.0`: neither P1's script nor the research ran `-race`, and `v0.17.0` races its own constructor (deviation 11, MADR F15).
10. The line numbers P5 cites drifted with P3's and P4's notes (0005-PLAN's F4-1 entry was no longer at `:3342`); each note is placed by the text it annotates.
11. Step 3's rename date (deviation 12).
12. That the latest tags hold until P5: go-llmprovider-sdk `v1.11.0` was tagged 74 minutes after P3 was committed (deviation 13).
13. That the index lists every record: 0014-REPORT had no row (deviation 14, pre-existing).
14. That `check_libs.py` runs with a library at its latest tag: P1's tests always had a newer tag, so the double export was never reached (deviation 15).

**The agent's own errors, found and fixed before staging:** P1's script printed failure text on success, and its scratch cleanup left read-only files on Windows while reporting them removed; step 12's harness first passed a turn that ended in error, and then sent the key file's whole `NAME=value` line as the key, which made deviation 6's premise wrong (its correction is recorded with it).

**Left open:** deviation 7's defect, a rejected key reported as no key, is 0005-PLAN F4-3's; `check_libs.py` runs no `-race`, which would have found deviation 11 before P4, and adding it is the owner's choice; and the Deferred items above.
