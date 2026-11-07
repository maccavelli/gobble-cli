---
status: proposed
date: 2026-11-07
decision-makers: repository owner
consulted: "this repository at b67cbbe (main, 2026-11-07); go-llmprovider-sdk v1.27.0's llmprovider/catalog (lookup.go, listing_cache.go, cache.go, discovery.go); Go 1.27.3's net/http and httpproxy; CI run 42387920951 (v0.3.0, attempt 1) and the eight earlier failed CI runs; 0035-, 0037- and 0039-PLAN's records of the same failure; probes on scratch copies of the tree, recorded under More Information"
informed: ""
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Give a run's background listings an owner that waits for them, and keep the tests off the network and out of other tests' homes

## Context and Problem Statement

The `v0.3.0` tag's CI run `42387920951` failed on `windows-2025` in `TestRecentModelsV1File`. The test's own checks
passed. Its teardown failed:

```text
--- FAIL: TestRecentModelsV1File/recent-models-v1.json (0.00s)
    testing.go:1619: TempDir RemoveAll cleanup: unlinkat <TMP>\TestRecentModelsV1Filerecent-models-v1.json4165991242\001\catalog: The directory is not empty.
```

So `validate` failed and the release was skipped. The same teardown failure has been recorded before, and in every
case it was left for "a record of its own":

* in `TestLoginAsksThroughQuestions`, on the Windows laptop and under `-race` on macOS (1 in 30 runs), by
  [0039-PLAN-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-PLAN-one-json-v2-encoder-and-invalid-utf8-policy.md),
  deviations 14 and 18;
* in `TestLastGoodOnDisk`, as `… /catalog: directory not empty`, by
  [0037-PLAN-permission-and-file-tool-hardening.md](0037-PLAN-permission-and-file-tool-hardening.md) (P7, G-local) and
  by [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md)
  (`check_libs.py`'s run of gobble's tests).

None of the eight earlier failed CI runs had it. This was its first appearance in CI.

0039-PLAN deviation 14 had one unverified candidate: "a model listing the harness does not wait for, logging after
`logs.close`". This record identifies the writers, measures them, and decides how to stop them. The investigation
found more than one bug, and one of them is not a teardown flake: **nine tests try to reach real model providers.**

### How the code works (at `b67cbbe`)

* **Directories are resolved on every use.** `systemDirs` is `appdirs.System` (`internal/cli/configpath.go:14`), which
  reads `os.Getenv` each time (`internal/appdirs/dirs.go:81-87`). With `GOBBLE_HOME` set, the config, data, state and
  cache directories are all that one directory (`dirs.go:56-61`). Without it, they are the platform defaults:
  `~/.cache/gobble`, `~/.local/state/gobble` and the rest on macOS and Linux, and `%LOCALAPPDATA%\gobble\…` on Windows
  (`dirs.go:89-133`). 25 calls in 16 files resolve directories this way.
* **A provider's listing runs on a goroutine nothing owns.**
  * `modelListing.start` runs the first fetch on a new goroutine (`internal/cli/models.go:52-64`).
  * `get` may start a retry on another (`:66-78`), after a failure and at least a minute.
  * Nothing records these goroutines, and nothing waits for them. `runEnv` has no field for them, and `run` closes
    the log and returns without them (`internal/cli/main.go:85-90`).
  * `agentOptions` starts every credentialed provider's listing (`internal/cli/agent.go:74-75`,
    `models.go:175-184`). Only the run's provider's listing is waited for.
* **A live listing is kept on disk by the listing goroutine.** `fetch` calls `writeKept` before the listing's `ready`
  closes (`models.go:156-161`).
  * `writeKept` → `catalogFile` → `<cache>/catalog/<id>.json` (`internal/cli/choice.go:188-230`). The directory is
    resolved when the write runs, not when the listing started.
  * `writePrivate` writes a temporary file and renames it (`choice.go:232-250`).
  * `readKept` goes through `catalogFile` too, so reading a kept listing creates the `catalog` directory (`:195`).
* **The log opens on its first record.** `lazyLog.get` opens `<state>/logs/gobble.log` inside a `sync.Once`
  (`main.go:240-268`).
  * `close` reads `l.closer` with no synchronisation, and does nothing if the log never opened (`:270-275`).
  * A first record after `close` therefore opens a log file, in the directories of that moment, that is never closed.
* **Model facts may fetch in the background, with no owner.**
  * The agent's `ModelInfo` hook calls `provider.ModelInfo` (`agent.go:93-99`). That passes `context.Background()`
    (`llm/provider/model.go:64-65`) and, for a stored OAuth login, the login, to the SDK's `catalog.CachedModel`.
  * `CachedModel` asks the token source for a token at once (go-llmprovider-sdk `llmprovider/catalog/lookup.go:135`).
    For an OAuth login, that may refresh, under gobble's refresh lock `auth.json.<id>.refresh.lock`
    (`internal/auth/logins.go:67-75`).
  * It then starts a background listing fetch (`lookup.go:140`). That fetch runs under
    `context.WithoutCancel(ctx)` and the SDK's 10 s `modelListingTimeout` (`listing_cache.go:218-244`,
    `discovery.go:118`).
  * The SDK counts these fetches in an unexported `WaitGroup`, "so tests can wait" (`cache.go:62-63`): its own tests
    only. It exports no way to wait for them.
  * With a nil token source it reads no listing and starts no fetch (`lookup.go:134`).
* **What the tests stub.** `TestMain` clears every credential variable, makes `listModels` fail offline, and
  replaces the browser opener and the sign-ins (`internal/cli/testmain_test.go:59-70`). It does not stub
  `provider.ModelInfo`'s SDK path.
* **How a test's home is torn down.**
  * `modelEnv` does `t.Setenv("GOBBLE_HOME", t.TempDir())` (`internal/cli/models_test.go:52-66`). The `TempDir`
    removal is registered first, so cleanup restores `GOBBLE_HOME`, to unset between tests, *before* it removes the
    home.
  * `t.Context()`, the `runEnv`'s `ctx` in the tests, is cancelled just before the cleanups run.
* **What `settled` waits for.** `settled` waits until the model choice shows no provider as `StateListing`
  (`internal/cli/choice_test.go:36-47`). But a provider that is not the run's, whose listing is still running and
  which has a kept listing on disk, is shown as `StateStale` (`choice.go:61-62`).

### Findings

**F1. `TestLastGoodOnDisk` ends with claude's listing still running, and `settled` cannot see it.**

* The test's second phase lists claude with a kept listing on disk (`choice_test.go:195-240`). openai is first in
  the registry, so it is the run's provider, and claude is read through `peek`. While claude lists, it is shown
  `StateStale`, not `StateListing`.
* So `settled(t, e)` after `close(release)` returns at once. The comment on line 239, "claude's listing ends before
  the home is removed", does not hold.
* Measured without any delay, right after that `settled`: claude's listing had not ended in 5 of 100 runs (state
  `stale`, `ready` not closed). In the other 95, it had ended: 70 `stale`, 25 `live`.

**F2. Such a listing writes into whatever home is set when it finishes: the test's own, the next test's, or the
user's real cache.**

* A scratch copy whose `writeKept` waited before resolving its directory, and reported the path, ran
  `TestLastGoodOnDisk` and `TestRecentModelsV1File` in turn. `HOME` and `XDG_*` pointed into the scratch directory.
* **With 2 ms**, in each of 4 iterations, phase two's claude listing (`["claude-1"]`) wrote
  `<scratch HOME>/cache/gobble/catalog/claude.json`. That is the platform default, written while `GOBBLE_HOME` was
  unset between tests. On a developer's machine it is the real `~/.cache/gobble/catalog/claude.json`.
* **With 5 ms**, it wrote into the next iteration's `TestLastGoodOnDisk` home twice, and into the default cache twice.
* The CI failure is the same write landing in the roughly one-millisecond window while `TestRecentModelsV1File`'s home
  is set. The temporary file (`.claude.json.*.tmp`), or the file itself, appears in that home's `catalog` after
  `RemoveAll` has read it, so the directory is not empty when its turn comes.
* This Mac's real `~/.cache/gobble/catalog` held only real listings when read. No test-written file was found, so
  the default-directory write is shown possible, not shown to have happened.

**F3. A test's OAuth refresh lock appears in its home after the test ends.**

* A scratch copy whose test helpers listed each home just before its removal, waited 100 ms and listed it again,
  ran the whole package twice under `-race`.
* It found one late file: `auth.json.grok.refresh.lock`, in `TestLoginAsksThroughQuestions`' home. That is the
  home 0039-PLAN's deviations 14 and 18 found not empty.
* The lock is taken by the SDK's token refresh for grok's stored login, which F4's path starts. On Windows a file
  still open cannot be removed, which `RemoveAll` reports as "The directory is not empty".

**F4. Nine tests try to reach real model providers.**

* A local proxy that records each connection's first line and forwards nothing (403, close) was set as `HTTPS_PROXY`
  and `HTTP_PROXY`. Each top-level test of `internal/cli` then ran alone.
* 9 of 264 tests connected:

  | Test | Connections |
  | :--- | :--- |
  | `TestSmallModelTitles` | `api.openai.com` ×14 |
  | `TestLoginMessage` | `auth.x.ai` ×12 |
  | `TestModelOptionOffersListedModels` | `api.openai.com` ×7 |
  | `TestRejectedKeyMessage` | `api.openai.com` ×4 |
  | `TestLoginSlash` | `api.anthropic.com` ×3 |
  | `TestFirstRunSetup` | `api.anthropic.com` ×3 |
  | `TestLoginAsksThroughQuestions` | `api.x.ai` ×3, `api.kilo.ai` ×1 |
  | `TestAPIKeyFlag` | `api.openai.com` ×2 |
  | `TestModelOffersLogin` | `api.openai.com` ×2 |

* The table is one run, 51 attempts. The whole module's tests, run together, made 52, all from `internal/cli`.
  Counts vary by one between runs: `TestSmallModelTitles` gave 14 in one and 15 in another.
* With `provider.ModelInfo` cut to return before its SDK lookup, a scratch copy made **0**. With the title model's
  provider cut instead, the 52 stayed. Every attempt is `ModelInfo` → `CachedModel`: a token, and a refresh at
  `auth.x.ai`, or a background listing.
* Without the proxy, these are real TLS connections to OpenAI, Anthropic, xAI and Kilo. They carry the tests' fake
  keys and logins. That breaks the rule `TestMain` states, that no test reaches a real provider (0002-PLAN Phase 3,
  0020-MADR D7). The suite passes only because the fetches fail quietly in the background, logged at Debug.
* Whether a run on CI completed such a request was not observed: the runners have the network, and the logs do not
  record it.

**F5. A log record after `close` opens a log file that is never closed.** This is from the code, not observed:

* `lazyLog.get` opens the file on the first record, wherever the directories are at that moment.
* `close` does nothing when nothing was opened, and reads `l.closer` with no lock.
* In the probes, no first record came after `close`. One record came after its test ended, to a log already open
  (`TestProviderForBuildsNamed`). Under `-race`, the whole package ran three times with no record after the test's
  end.

**F6. Reading a kept listing creates the `catalog` directory.** `readKept` → `catalogFile` →
`appdirs.EnsurePrivateDir` (`choice.go:195`), so `gobble providers --check` and the model choice create
`<cache>/catalog` even when nothing is kept.

### What is not wrong

* The production `GOBBLE_HOME` does not change inside a process, so per-use resolution is only a hazard for writers
  that outlive their test.
* The commands that start listings and read them, `providers --check` and `models list`, wait for each one
  (`internal/cli/providerscmd.go:34-50`, `internal/cli/modelscmd.go:94-103`).
* `TestMain`'s offline lister keeps `listModels` off the network; F4 goes around it, not through it.
* `TestBashLongOutput` failed once in the whole-module proxy probe, and passed three times without the probe's
  environment. It is not part of this record.

## Decision Drivers

* **A test touches only its own temporary directories:** never another test's home, never the user's real
  directories.
* **No test reaches the network,** and an escape is a failure, not a silent Debug line.
* **Background work has an owner** that ends it and waits for it before the run's directories, log and context go.
* **No workaround.** Nothing is retried, slept on, or skipped, and no assertion is loosened.
* **The smallest change that removes each cause,** with each fix seen failing first.

## Considered Options

* **A.** No change: rerun on a teardown failure, as before.
* **B.** Test-only fixes: `settled` waits for the listings, the helpers wait for theirs at cleanup, and `TestMain`
  stubs `ModelInfo`'s fetch.
* **C.** B, plus a `runEnv` that owns its listings, ends them and waits for them; a log that opens nothing after
  `close`; reads that create no directory; and a `TestMain` network guard that fails the run on any outside
  connection.
* **D.** C, plus resolving the directories once per `runEnv` instead of on every use.

## Decision Outcome

Chosen option: **C**, because it removes each cause where it lives.
* The listing goroutines get an owner, which the product lacks and the tests can only work around.
* `ModelInfo`'s fetch is stubbed where `TestMain` stubs the other outside effects.
* The network rule becomes a check instead of a hope.
* D's change of 25 calls is not needed once nothing outlives its `runEnv`.

### D1. A `runEnv` owns its listings

* Each `modelListing` goroutine, the first fetch and any retry, runs under a group the `runEnv` holds, and under a
  context derived from the run's that the `runEnv` can cancel.
* `e.endListings()` cancels that context, then waits for the group. A cancelled listing is not live, so it keeps
  nothing on disk.
* `run` calls it after the command and before `logs.close` (`main.go:86-89`). So `gobble -p` and every other command
  end their background listings before the log closes and the process exits. Today they are cut off at exit, which
  can leave a `writePrivate` temporary file (not observed).
* The wait is bounded. A listing honours its context (the SDK's `List` does, and its own timeout is 10 s). After
  6 s (Q1), `endListings` returns anyway and writes one Warn record naming the providers still listing. The bound
  protects a user's exit from a listing that ignores cancellation; it is not a retry or a sleep.

### D2. The tests end what they start, before their home goes

* Every helper that builds a `runEnv` registers `e.endListings` as a cleanup after its `TempDir` and `Setenv`. That
  covers `modelEnv`, `envHere`, `startEnv`, `newModelHarness` and the line harnesses. So it runs first: before
  `GOBBLE_HOME` is restored, before the log closes, before the home is removed.
* `runEnv`s built inside `Main` (`c.run`, `runIn`) get D1's call in `run`.
* `settled` waits until every listing the `runEnv` has started has ended, not until the shown state stops being
  `StateListing`. The comment at `choice_test.go:239` then holds.

### D3. Model facts in the tests make no request

* `internal/cli` gets `modelInfo`, a package variable that is `provider.ModelInfo` by default. The `ModelInfo` hook
  (`agent.go:93-99`) calls it.
* `TestMain` replaces it with one that passes no key and no login. The SDK then reads only the snapshot and
  `models.json`, and starts no fetch (`lookup.go:134`).
* A test pins the replacement by identity, as `TestNoBrowserOrSignIn` pins the others (`testmain_test.go:40-52`).

### D4. The test binary fails on any outside connection

* `TestMain` starts a loopback listener and sets `HTTPS_PROXY` and `HTTP_PROXY` to it, and clears `NO_PROXY`, before
  `m.Run`. The listener records each connection's first line and answers 403.
* Go never proxies `localhost` or a loopback address (`httpproxy/proxy.go:178-187`), so `httptest` servers are
  unaffected. The proxy variables are read once per process (`net/http/transport.go:1036-1040`), so `TestMain` sets
  them before any request.
* After `m.Run`, if anything was recorded, `TestMain` prints each host and count and exits 1.
* This catches a client that honours the proxy variables, as the SDK's does (F4's probe). A client that sets no
  proxy would go around it; none was found.

### D5. A log opens nothing after `close`

* `lazyLog` takes a mutex for `get` and `close`. After `close`, `get` returns a logger that discards: no file is
  opened, and a record is lost rather than written to a directory the run no longer owns.
* With D1, no listing logs after `close` in a normal run, so this only guards against a future writer.

### D6. Reading a kept listing creates no directory

* `readKept` resolves the path without `EnsurePrivateDir`. Only `writeKept` creates `<cache>/catalog`.

### Consequences

* Good, because the teardown failures in `TestRecentModelsV1File`, `TestLastGoodOnDisk` and
  `TestLoginAsksThroughQuestions` lose their writers: F1 by D1 and D2, F3 by D3.
* Good, because no test can write into the user's real gobble cache or state (F2), or reach a real provider (F4).
  D4 turns a future escape into a failing run that names the host.
* Good, because a command's background listings now end with it, instead of being cut off at exit.
* Neutral, because a run's exit can wait for a listing to answer its cancellation, up to 6 s (Q1).
* Bad, because the `runEnv` gains a group and a cancel function, and every test helper must register one more
  cleanup. A helper that forgets is caught only by D4 or a teardown failure, so the PLAN lists the helpers and checks
  each.
* Bad, because D4's guard depends on clients honouring the proxy variables.

### Confirmation

1. After `TestLastGoodOnDisk`'s last `settled`, every listing it started has ended, in 200 runs. Seen failing first
   with today's `settled`: F1's 5 in 100.
2. F2's probe, rerun on the fixed tree with the 2 ms and 5 ms delays, shows every `writeKept` in its own test's home,
   and nothing under the scratch `HOME`'s default directories.
3. F3's probe, rerun under `-race` twice over the whole package, finds no late file.
4. F4's per-test probe finds 0 of the package's tests connecting.
5. D4's guard fails the run, naming `api.openai.com` and the others, on a scratch copy with D3's stub removed. It
   passes on the tree.
6. `endListings` is covered by a test: a listing that blocks until its context is done ends when `endListings`
   runs, and writes nothing after it returns. A listing that ignores its context makes `endListings` return at the
   bound, with its Warn record. Each is seen failing first.
7. D5 is covered: a record after `close` opens no file. D6 too: `readKept` with no kept listing leaves no `catalog`
   directory. Each is seen failing first.
8. `go test -race -count=60 -run '^TestLoginAsksThroughQuestions$'` passes on macOS and on the Windows laptop, and
   the whole package passes `-count=3` on all three platforms.
9. `make preflight` and `make race` pass, and CI passes on its three runners.

## Pros and Cons of the Options

### A. No change

* Good, because it costs nothing now.
* Bad, because it fails a release at random, as it did `v0.3.0`'s first run.
* Bad, because tests keep reaching real providers with fake credentials, and can write into a developer's real cache.

### B. Test-only fixes

* Good, because no product code changes.
* Bad, because the tests can wait only for listings they can see. The `runEnv`s that `Main` builds have none to show
  them, and a command's listings are still cut off at exit.
* Bad, because the network rule stays unchecked: the next unstubbed path escapes silently, as `ModelInfo`'s did.

### C. B, with ownership, the guard and the small fixes

* Good, because each cause is fixed at its source, and D4 checks the rule the tests rely on.
* Neutral, because product code changes: `runEnv`, `run`, `lazyLog`, `readKept`, and the `ModelInfo` hook's
  indirection.
* Bad, because a run's exit may wait up to 6 s (Q1).

### D. C, with the directories resolved once per `runEnv`

* Good, because even a writer that escaped D1 would write into its own run's home.
* Bad, because it changes 25 calls in 16 files, and many tests set `GOBBLE_HOME` after building their `runEnv`, so
  they would need reordering.
* Bad, because after C nothing outlives its `runEnv`, so it buys nothing the tests can measure.

## More Information

### Owner questions

* **Q1. D1's bound.** *Recommended: 5 s.* That is longer than a cancelled listing takes to return, and shorter than
  the SDK's own 10 s timeout. Not chosen: no bound, which lets a stuck listing hold the exit for up to 10 s; or 1 s,
  which may cut a slow cancellation short on a loaded host.
* **Q2. An SDK issue.** go-llmprovider-sdk could export a way to wait for `CachedModel`'s background fetches, or an
  option to start none. gobble does not need it after D3, since the tests pass no token source.
  *Recommended: file it on the owner's word,* as a convenience for other consumers. Not chosen: not filing it.

*Answered 2026-11-07: "Q1. 6s, Q2. file a detailed, accurate, actionable issue in the go-llmprovider-sdk repo." D1's
bound is 6 s. The issue is [go-llmprovider-sdk#12](https://github.com/maccavelli/go-llmprovider-sdk/issues/12). It
proposes `(*catalog.Cache).Close(ctx) error`, which stops a cache's new background work and waits for what is
running. A caller opts in by owning a `Cache`. It also proposes `(*auth.OAuthSession).Wait(ctx) error` for a
detached refresh. gobble's D3 does not wait for it.*

### The probes

Each ran on a scratch copy of the tree at `b67cbbe`, with `HOME` and `XDG_*` pointing into the scratch directory, so
nothing reached the real gobble directories. The scripts are scratch files, not committed:

* **`settled_probe.py`:** after `TestLastGoodOnDisk`'s last `settled`, whether claude's `ready` had closed; 100 runs
  (F1).
* **`late_probe.py`:** `writeKept` waits 0, 2, 5 or 30 ms and reports its models and path (F2).
* **`late_probe3.py`:** flags kept-listing reads and writes, and log records, made after their test's context was
  done; the whole package three times, and three times under `-race` (F5).
* **`late_probe4.py`:** each helper's home listed before its removal, and again 100 ms later; the whole package once,
  and twice under `-race` (F3).
* **`net_probe.py` and `net_attrib.py`:** a recording proxy that forwards nothing; the whole module, then each
  `internal/cli` test alone (F4).
* **`net_bisect.py`:** the same with `ModelInfo`, the title provider, or both cut (F4).

### Not decided here

* Whether a production run's `ModelInfo` should own its background fetches too. In production they warm a cache and
  write nothing to disk; the SDK times each out.
* D's per-`runEnv` directories.

### Related records

* [0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md) Phase 3: no test reaches a real
  provider.
* [0020-MADR-full-model-catalog-listing.md](0020-MADR-full-model-catalog-listing.md) D3 and D7: the listing, and the
  tests' offline lister.
* [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md) D4: kept listings.
* [0039-PLAN-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-PLAN-one-json-v2-encoder-and-invalid-utf8-policy.md)
  deviations 14 and 18,
  [0037-PLAN-permission-and-file-tool-hardening.md](0037-PLAN-permission-and-file-tool-hardening.md) P7, and
  [0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md):
  the earlier sightings.
