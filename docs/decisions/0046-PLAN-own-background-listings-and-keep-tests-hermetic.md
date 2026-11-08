---
status: in-progress
date: 2026-11-07
associated-madr: "0046-MADR-own-background-listings-and-keep-tests-hermetic.md"
---
<!-- markdownlint-disable MD013 MD024 MD060 -->

# Implement giving a run's background listings an owner that waits for them, and keeping the tests off the network and out of other tests' homes

Associated MADR: [0046-MADR-own-background-listings-and-keep-tests-hermetic.md](0046-MADR-own-background-listings-and-keep-tests-hermetic.md)

## Goal

* No test reaches the network, and the test binary fails when one tries (MADR D3, D4).
* A `runEnv` ends and waits for the listings it starts. `run` does it for every command, and the tests do it before a
  home goes (D1, D2, as amended).
* A log opens nothing after `close` (D5), and reading a kept listing creates no directory (D6).

The plan is complete when the MADR's Confirmation 1 to 9 is met.

## Scope

**Base.** `main` at `7c57efb`, the MADR's commit, one ahead of `origin/main` (`b67cbbe`) when this plan was written.

**In scope,** by phase:

* P0: this plan; the MADR's D2 amendment of 2026-11-07; `docs/README.md`'s 0046 rows.
* P1, the network:
  * `internal/cli/agent.go`: the `modelInfo` variable;
  * `internal/cli/testmain_test.go`: the stub and the guard;
  * `llm/provider/model.go`: `ModelInfo`'s comment;
  * `docs/guides/developing.md`: a "What the tests may touch" section.
* P2, ownership:
  * `internal/cli/models.go`: the group, the context, `endListings`;
  * `internal/cli/main.go`: `runEnv`'s fields, and `run`;
  * `internal/cli/testmain_test.go`: the owner check;
  * the test files whose `runEnv`s start listings: `choice_test.go`, `models_test.go`, `start_test.go`,
    `line_test.go`, and each further file the owner check names. P2's record lists them all;
  * `docs/architecture.md`: `internal/cli`'s row.
* P3, the small fixes: `internal/cli/main.go` (`lazyLog`), `internal/cli/choice.go` (`readKept`), and their tests.
* P4: the records.
* *Added 2026-11-07 by the MADR's amendment of that date (D7 to D9, accepted; these phases proposed):*
  * P5, the run's cache (D7): `llm/provider/cache.go` (deviation 5), `llm/provider/provider.go` (`Options.Cache`,
    `sdkOptions`), `llm/provider/model.go`
    (`ModelInfo`'s form that takes `Options`), `internal/cli/main.go` (`runEnv`'s field), `internal/cli/models.go`
    (`endListings`), `internal/cli/agent.go`, `internal/cli/title.go` and `internal/cli/modelsjson.go` (the options the
    run builds), `internal/cli/testmain_test.go` (deviation 4), and their tests;
  * P6, the run's sessions (D8): a new `llm/provider/sessions.go`, `llm/provider/provider.go` (`Options.Sessions`,
    `login`), `internal/cli/main.go`, `internal/cli/models.go`, `internal/cli/credentials.go`
    (`refreshCredentials`), and their tests;
  * P7: acceptance and the records; `docs/architecture.md`'s `internal/cli` and `llm/provider` rows.

**Out of scope:**

* MADR option D, the directories resolved per `runEnv`;
* ~~production `ModelInfo` fetches ("Not decided here")~~ *(2026-11-07: P5 to P7, by the MADR's amendment)*;
* any package but `internal/cli` and the one comment in `llm/provider`. The whole module was measured: only
  `internal/cli` connects;
* go-llmprovider-sdk#12's implementation, which is the SDK's;
* a release tag.

## Conventions every phase follows

* **C1. Start state.** Each phase first re-reads `git status`, `HEAD` and `origin/main`. A difference from this plan is
  a deviation.
* **C2. Seen failing.** Every new test or check is seen failing on a scratch copy of the tree, never in the working
  tree: against the code as it was before the phase, and against a planted break of the new code. The record quotes
  each failure.
* **C3. Evidence.** Commands write to a scratch log, with their exit status taken before any filter.
* **C4. Deviations stop the phase,** and are recorded here, and in the MADR when a decision or fact changes, before
  work goes on.
* **C5. No real effects.** Every probe and test run off the tree points `HOME` and `XDG_*` into a scratch directory.
  A run that could reach the network goes through the recording proxy, which forwards nothing.
* **C6. Gates and commits.**
  * The Go files pass `make pre-add-check FILES=…`.
  * The tree passes `make preflight` and `make race`, and the identifier scan finds nothing.
  * The agent stages. The owner commits with `git commit --no-edit`, or tells the agent to. The disclosure guard
    runs before any push.
* **C7. Identifiers.** No hostname, account name or real-machine path in a committed file. Temporary paths in quoted
  output become `<TMP>`; the dev hosts are named by platform.

## Implementation Steps

### P0 — the records

1. This plan, the MADR's D2 amendment, and `docs/README.md`'s rows: the MADR's updated, the PLAN's added, both
   `proposed`.
2. `make check-records markdownlint` passes. Staged; the owner commits.

### P1 — the tests stay off the network (D3, D4)

**Gate:** P0 committed, and the owner's approval of this plan.

1. **The stub (D3).**
   * `internal/cli/agent.go`: `var modelInfo = provider.ModelInfo`, with a comment naming D3. The `ModelInfo` hook
     calls `modelInfo(...)`.
   * `testmain_test.go`: `offlineModelInfo`, which calls `provider.ModelInfo` with no key, no workspace and no login.
     `TestMain` sets `modelInfo = offlineModelInfo` beside `listModels = offlineList`.
   * `TestNoBrowserOrSignIn` gains `"modelInfo": same(modelInfo, offlineModelInfo)`.
2. **The guard (D4),** in `testmain_test.go`:
   * Before `m.Run`, `TestMain` listens on `127.0.0.1:0`, and serves each connection on its own goroutine: it reads
     the first line, records it, answers `HTTP/1.1 403 Forbidden`, and closes.
   * It sets `HTTPS_PROXY`, `HTTP_PROXY`, `https_proxy` and `http_proxy` to the listener, and clears `NO_PROXY` and
     `no_proxy`. This happens after `mcptest.ServeStdioIfAsked`, so the MCP fixture is unchanged.
   * After `m.Run`, it closes the listener. If anything was recorded, it prints "tests tried to reach the network:"
     and each host with its count to stderr, and exits 1, whatever `m.Run` returned.
   * *Deviation 2 (2026-11-07):* ~~it sets the proxy variables in `TestMain`~~. Those are read before `TestMain`
     runs, so the tests run in a child process that starts with them set (MADR D4, as amended):
     * `TestMain`, without the marker `GOBBLE_TEST_NETGUARD`, starts the listener and runs `os.Args[0]` with
       `os.Args[1:]`. The child's environment has the proxy variables, `NO_PROXY` and `no_proxy` cleared, and the
       marker. stdin, stdout and stderr pass through.
     * The parent exits with the child's code, or 1 when the guard recorded a host.
     * With the marker, `TestMain` first checks that `http.ProxyFromEnvironment` names `HTTPS_PROXY` for
       `https://example.com/`. If it does not, it prints why and exits 2. Then it runs the tests as before.
3. **The comment.** `llm/provider/model.go:55-63` says `ModelInfo` "never waits and makes no request of its own".
   With a login, the SDK asks the login for a token, which may refresh at the issuer before it returns (go-llmprovider-sdk
   `lookup.go:117-147`). The comment says so.
4. **Seen failing** (C2), on scratch copies:
   * the guard on today's stub-less code: the run exits 1 and names `api.openai.com`, `auth.x.ai`,
     `api.anthropic.com`, `api.x.ai` and `api.kilo.ai`. The counts are quoted;
   * `TestNoBrowserOrSignIn` with `TestMain`'s `modelInfo` line removed: it fails naming `modelInfo`;
   * the guard with a planted test that dials `https://example.com/` through `http.DefaultClient`: exit 1 naming
     `example.com:443`;
   * a planted test that reaches an `httptest.Server`: it passes, and the guard records nothing (loopback is not
     proxied);
   * *deviation 2:* the parent with `HTTPS_PROXY` left out of the child's environment: the child's check exits 2,
     naming the variable.
5. **The guide.** `docs/guides/developing.md` gains "### What the tests may touch", after "Golden transcripts":
   * `internal/cli`'s tests reach no network, and `TestMain`'s proxy fails the run naming any host one tried;
   * each test keeps to its own temporary directories;
   * a test that starts listings owns them (P2's `ownListings`).
6. **Gates** (C6). `make pre-add-check` on the changed Go files, `make preflight`, `make race`.
   * The proxy probe (`net_attrib.py`) over `internal/cli`: 0 tests connect (MADR Confirmation 4).
   * The whole module through the proxy (`net_probe.py`): 0 connections.
   * *Deviation 2:* the guard now overrides an outside proxy for `internal/cli`'s child process, so the probes'
     outside proxy sees that package's tests no longer. The guard's own report is that package's measure. The
     probes still measure every other package.
7. **Record.** P1's record here. Staged; the owner commits.

### P2 — a `runEnv` owns its listings (D1, D2)

**Gate:** P1 committed.

1. **The tests first** (`internal/cli/*_test.go`), each seen failing before the code:
   * `TestSettledWaitsForAStaleListing`. A kept claude listing is on disk, openai is the run's provider, and claude's
     listing blocks on a channel. `settled` runs on a goroutine and must not return while claude shows `StateStale`.
     After the channel closes, it returns. Today's `settled` returns at once.
   * `TestEndListingsWaits`. A listing blocks until its context is done, then answers live. `endListings` returns
     only after that listing has ended, and after its kept file is written. A listing started after `endListings`
     calls no lister.
   * `TestEndListingsBound`. A listing ignores its context and blocks on a channel the test closes in its cleanup.
     With `endListingsWait` set to 50 ms for the test, `endListings` returns at the bound and writes one Warn record
     naming the provider. The cleanup then closes the channel and waits for the listing, so the test leaves nothing
     running.
   * `TestRunEndsListings`. Through `Main`, a `-p` run with two keyed providers and a lister that blocks the second
     until its context is done, then records that it returned. When `Main` returns, the second listing has ended.
     Today `Main` returns with it blocked.
2. **The code** (`models.go`, `main.go`):
   * `runEnv` gains a listings owner: a mutex, a `sync.WaitGroup`, a context derived from `e.ctx` with its cancel
     function, and `owned` and `ended` flags.
   * `e.goListing(f func()) bool` starts `f` on the group, under the mutex, unless `ended`. When the `runEnv` is not
     owned, it first calls `unownedListing`, which is nil in production.
   * `newModelListing` takes the starter. The first fetch and any retry go through it, and `fetch` lists under the
     owner's context.
   * A listing not started because `ended` is set gives `Listing{Models: provider.Models(id), Err: errListingsEnded}`.
   * `e.endListings()` sets `ended` and cancels under the mutex, then waits for the group. It returns after
     `endListingsWait`, 6 s (MADR Q1), with one Warn record (`listings still running`, the providers), if the group
     has not finished.
   * `e.ownListings(cleanup func(func()))` sets `owned`, and with a non-nil `cleanup` registers `e.endListings` with
     it.
   * `run` calls `e.ownListings(nil)` before the command, and `e.endListings()` after it, before `logs.close`.
3. **The owner check** (D2 as amended), in `testmain_test.go`. `TestMain` sets `unownedListing` to record
   `runtime/debug.Stack()`. After `m.Run`, if anything was recorded, it prints each stack and exits 1.
4. **The test helpers.**
   * `ownListings(t testing.TB, e)` is `e.ownListings(t.Cleanup)`; `newLineHarness` takes a `testing.TB`. `modelEnv`, `envHere`, `startEnv`, `newModelHarness` and
     `newLineHarness` call it after their `TempDir` and `Setenv`.
   * Each further site the owner check names gets the same call. Its file joins P2's list, which is not a deviation:
     Scope names this.
   * `settled` waits until every listing in `e.listings` has ended and none is retrying.
   * The `modelListing` unit test (`models_test.go:150`) passes a starter on a group the test waits for.
5. **Seen failing** (C2), on scratch copies:
   * each step 1 test against the code before step 2, with its failure quoted;
   * the owner check with `ownListings` removed from `modelEnv`: the run exits 1, printing a stack through
     `modelEnv`'s callers;
   * `endListings` without its wait: `TestEndListingsWaits` fails;
   * `run` without its call: `TestRunEndsListings` fails.
6. **`docs/architecture.md`.** `internal/cli`'s row gains one sentence: a run's background listings end with it,
   each cancelled and waited for up to 6 s before the log closes.
7. **Gates** (C6). `make pre-add-check`, `make preflight`, `make race`, and `make perf`, since `run`'s end changed.
   * On the Linux and Windows dev hosts, the tree shipped: `go test -count=3 ./internal/cli/` and the race run.
8. **Record.** Staged; the owner commits.

### P3 — the log and the kept listings (D5, D6)

**Gate:** P2 committed.

1. **Tests first:**
   * `TestLogAfterCloseOpensNothing`. A `lazyLog` under a scratch `GOBBLE_HOME` that never logged is closed, and then
     logs. No `logs` directory exists after. Today it creates `logs/gobble.log`.
   * `TestReadKeptCreatesNothing`. `readKept` in a fresh home returns nothing, and leaves no `catalog` directory.
     Today it creates one.
2. **The code.**
   * `lazyLog` takes a mutex in `get` and `close`. `close` sets `closed`. A `get` after it returns a discarding
     logger and opens nothing.
   * `catalogFile` splits into `catalogPath(id)`, which creates nothing, used by `readKept`, and the directory's
     creation in `writeKept`.
3. **Seen failing** (C2): each test against the code before step 2. Then each planted break: the closed check
   removed; `readKept` back on `catalogFile`.
4. **Gates** (C6), with the dev hosts' `go test ./internal/cli/`.
5. **Record.** Staged; the owner commits.

### P4 — acceptance

**Gate:** P3 committed and pushed on the owner's word.

1. **The probes, again,** on the tree (C5). Each scratch probe's anchor is updated to the new code, and the record
   says how:
   * `settled_probe.py`, 200 runs: every listing ended (Confirmation 1);
   * `late_probe.py`, at 2 ms and 5 ms: every `writeKept` in its own test's home, and nothing under the scratch
     `HOME`'s defaults (Confirmation 2);
   * `late_probe4.py` under `-race`, twice: no late file (Confirmation 3);
   * `net_attrib.py`: 0 connections (Confirmation 4).
2. **The repetitions** (Confirmation 8).
   * `go test -race -count=60 -run '^TestLoginAsksThroughQuestions$' ./internal/cli/`, on macOS and on the Windows
     laptop.
   * `go test -count=3 ./internal/cli/` on all three platforms.
3. **CI** passes on its three runners (Confirmation 9).
4. **Records.**
   * The plan becomes `complete`, and the MADR `accepted`, with the `docs/README.md` rows to match.
   * The MADR's Confirmations are mapped to their evidence.
   * The 0039-PLAN deviations 14 and 18, and the 0037- and 0035-PLAN sightings, each gain a dated note pointing to
     this record.

### P5 — the run owns a `catalog.Cache` (MADR D7; amendment of 2026-11-07)

**Gate:** the amendment approved, Q3 to Q5 answered, and its records commit landed. The plan is then `in-progress`.

1. **The tests first,** each seen failing before the code (C2), against an `httptest` listing server, never a real
   service (C5):
   * `llm/provider`: `TestOptionsCacheReachesTheSDK`. `List` and the `Options` form of `ModelInfo` given
     `Options{Cache: c}` list into `c`: after `c.Close`, a second `List` given `c` fails with an error matching
     `llmprovider.ErrInvalidRequest`, while one without a cache still lists. Today `Options` has no `Cache`.
   * `internal/cli`: `TestRunOptionsCarryTheCache`. `modelProvider`, `providerFor`, `smallProvider`, the listing's
     options (`models.go:267`) and the login's options (`login.go:75`) each build `provider.Options` whose `Cache` is
     the run's. It replaces `ambientProvider` and `listModels` to read the options, as the existing option tests do.
   * `internal/cli`: `TestEndListingsClosesTheCache`. A `ModelInfo` fetch through the run's cache blocks until its
     request's context is done. `endListings` returns only after that request has ended. With the server never
     answering and the fetch made to ignore cancellation, `endListings` returns at D1's bound, with the Warn record
     naming the cache.
2. **The code:**
   * ~~`provider.Options` gains `Cache *catalog.Cache`; `sdkOptions` appends `catalog.WithCache(o.Cache)` when it is not
     nil.~~ *(Deviation 5: `llm/provider` gains `Cache`, holding the SDK's `catalog.Cache`, with `Close(ctx) error`;
     `provider.Options` gains `Cache *provider.Cache`; `sdkOptions` appends `catalog.WithCache` of the inner cache when
     it is not nil.)*
   * `provider.ModelInfoWith(o Options, id, model string, mj *gcatalog.ModelsJSON) llm.Model` takes the key, the
     workspace and the logins from `o`, and passes `WithCache` to `CachedModel`. `ModelInfo` calls it, with no cache,
     so its behaviour and signature are unchanged. ~~`modelFacts` keeps its type; its default becomes a closure the
     `runEnv` builds over `ModelInfoWith`, and D3's stub replaces it as before.~~ *(Deviation 4: `modelFacts` becomes
     `func(o provider.Options, id, model string, mj *catalog.ModelsJSON) llm.Model`, by default
     `provider.ModelInfoWith`. The hook builds the run's options: the key, the workspace, the logins when the provider
     has one, and the cache. D3's stub becomes `provider.ModelInfoWith(provider.Options{}, id, model, mj)`, pinned by
     identity as before.)*
   * `runEnv` gains `cache` ~~`catalog.Cache`~~ `provider.Cache` (deviation 5). `providerOptions` sets `o.Cache = &e.cache`. `providerFor`,
     `smallProvider`, the listing and the login call it on every path; `modelProvider` calls it only once a provider is
     chosen (`agent.go:39-40`), so it sets the cache itself before that branch.
   * `endListings`, after its group wait, calls `e.cache.Close` under a context ending at D1's bound (Q4), and adds
     the cache to the Warn record when `Close` returns `ctx.Err()`.
3. **Seen failing** (C2), on scratch copies: each step 1 test against the code before step 2; then each planted break:
   `sdkOptions` without `WithCache`; `providerOptions` without the cache; `endListings` without `Close`.
4. **Gates** (C6), with `make perf`, since `run`'s end changed; the dev hosts' `go test -count=3 ./internal/cli/
   ./llm/provider/` and the race run.
5. **Record.** Staged; the owner commits.

### P6 — one OAuth session per provider for the run (MADR D8)

**Gate:** P5 committed.

1. **The tests first,** each seen failing before the code, with an `httptest` issuer and a scratch `auth.json` (C5):
   * `llm/provider`: `TestSessionsShareOneSession`. Two `Options.login` calls for one provider through one
     `Sessions` return the same `*auth.OAuthSession`; without `Sessions`, two different ones, as today. `Drop(id)` makes
     the next call load the stored login again.
   * `llm/provider`: `TestSessionsWait`. A refresh that blocks on the issuer: `Sessions.Wait(ctx)` returns only after
     it, and the store then holds the rotated refresh token. With `ctx` ending first, it returns `ctx.Err()` naming the
     provider.
   * `internal/cli`: `TestRefreshCredentialsDropsTheSession`. After a sign-in's `refreshCredentials(id)`, the run's
     next provider for `id` is built over the new login, not the held one.
   * `internal/cli`: `TestEndListingsWaitsForARefresh`. A refresh blocked on the issuer when `endListings` runs:
     `endListings` returns after it, after the cache's `Close`. With the issuer never answering and the wait bound set
     to 50 ms for the test, it returns at the bound with a Warn record naming the provider and saying its rotation may
     not be saved; the test's cleanup then releases the issuer and waits, so nothing is left running.
2. **The code:**
   * `llm/provider/sessions.go`: `Sessions`, a mutex and a map of provider id to `*auth.OAuthSession`; `get(id,
     load)`, `Drop(id)`, and `Wait(ctx) error`, which waits for each held session in turn and joins the providers still
     refreshing into its error.
   * `Options` gains `Sessions *Sessions`. `login` returns `o.Sessions`' session for `id` when it is set, loading it
     once from `o.Logins` as today.
   * `runEnv` gains `sessions provider.Sessions`; `providerOptions` and the `ModelInfoWith` closure pass it.
   * `refreshCredentials` calls `e.sessions.Drop(id)` beside its `delete(e.listings, id)`.
   * `endListings`, after `Close`, calls `e.sessions.Wait` under its own bound, `refreshWaitBound` (Q3; 60 s, tests
     shorten it), and writes the Warn record when it returns an error.
3. **Seen failing** (C2): each step 1 test against the code before step 2; then each planted break: `login` ignoring
   `Sessions`; `refreshCredentials` without `Drop`; `endListings` without `Wait`; `Wait` under D1's 6 s instead of
   Q3's bound (`TestEndListingsWaitsForARefresh` with a refresh of 7 s on a scratch copy).
4. **Gates** (C6), with `make perf`; the dev hosts as in P5.
5. **Record.** Staged; the owner commits.

### P7 — acceptance of D7 to D9

**Gate:** P6 committed.

1. **The probes** of the MADR's Confirmation 10 and 11, on scratch copies (C5), each first at `3483725` to see it fail:
   * a `-p` run whose `ModelInfo` hook passes a token source to a blocking listing server: after `Main` returns, no
     request in flight, and the goroutine count back to its start;
   * a `-p` run with an OAuth login whose refresh blocks: `Main` waits for it, and `auth.json` holds the rotated token;
     past the bound, `Main` returns with the Warn record.
2. **D4's guard** still finds no outside connection across the module (Confirmation 12).
3. **Records.** The MADR's amendment becomes accepted and its Confirmations 10 to 13 are mapped to their evidence; the
   plan returns to `complete`; `docs/architecture.md`'s `internal/cli` row says a run's SDK cache and sign-in sessions
   end with it, and `llm/provider`'s row names `Options.Cache` and `Sessions`; `docs/README.md`'s rows.

## Verification

| MADR Confirmation | Where |
| :--- | :--- |
| 1. Every listing ended after `settled` | P2 step 1 (`TestSettledWaitsForAStaleListing`), P4 step 1 |
| 2. Writes stay in their own home | P4 step 1 |
| 3. No late file | P4 step 1 |
| 4. No test connects | P1 step 6, P4 step 1 |
| 5. The guard fails without the stub | P1 step 4 |
| 6. `endListings` waits, and returns at the bound | P2 steps 1 and 5 |
| 7. D5 and D6 | P3 |
| 8. The repetitions | P4 step 2 |
| 9. Gates and CI | each phase's gates, P4 step 3 |
| 10. A run's SDK fetches end with it | P5 step 1 (`TestEndListingsClosesTheCache`), P7 step 1 |
| 11. A refresh in flight is waited for, and its rotation saved | P6 step 1 (`TestEndListingsWaitsForARefresh`), P7 step 1 |
| 12. No test connects, D9's tests pass | P7 step 2 |
| 13. Gates on the three platforms | P5's and P6's gates |

## Rollout and Rollback

* **Rollout.** Each phase is one commit on `main`. P1 alone stops the network traffic, and can ship first if a release
  is wanted before P2.
* **Rollback.**
  * P1 reverts cleanly.
  * P2 reverts with its test changes, and P3 reverts alone.
  * If `make perf` or a user's exit shows the 6 s bound being reached in normal use, that is a deviation (C4). The
    bound is not raised or removed to hide it.
  * P5 reverts alone; P6 reverts alone, and reverting P5 reverts P6 first. Either brings back the SDK work `run`
    does not wait for (MADR G1 to G3), not a failure.

## Execution record

### Deviation 1 (2026-11-07): P1, `modelInfo` is `modelFacts`

* **Found** at P1 step 1. `go vet` refused the variable: `internal/cli/modelscmd.go:218:6: modelInfo redeclared in
  this block`. A function of that name already formats `gobble models info`.
* **Decision** (no prompt needed: a name, not a behaviour). The variable is `modelFacts`, and `TestMain`'s
  replacement is `offlineModelFacts`. MADR D3 is annotated. No file joins the phase.

### Deviation 2 (2026-11-07): P1, the proxy variables are read before `TestMain`

* **Found** at P1 step 4, on scratch copies. The `nostub` plant failed `TestNoBrowserOrSignIn`, but the guard
  recorded nothing. The `dial` plant's `GET https://example.com/` returned `err <nil>`: through the guard's 403 it
  would have failed, so it reached the site.
  * A planted test then printed `HTTPS_PROXY="http://127.0.0.1:<port>"; ProxyFromEnvironment gives <nil>`.
  * A scratch overlay of the toolchain's `net/http/transport.go` printed the stack of `envProxyOnce`'s first run:
    `net/http.ProxyFromEnvironment` ← `github.com/modelcontextprotocol/go-sdk/oauthex.proxyConfigured`
    (`oauth2.go:180`) ← `newDiscoveryTransport` (`:147`) ← `oauthex.init()` (`:28`). `HTTPS_PROXY` was empty.
  * It was pre-existing: the MCP SDK has done this since gobble linked it. It mattered only once D4 relied on the
    opposite.
* **Real effects:** none new, but two that happened. Before this was known, the `nostub` copy's run made the same
  outside attempts the tests had made on every run until then. The `dial` copy's `GET` reached `example.com`, a
  public test domain, with no credential.
* **Options put to the owner:**
  1. (recommended) a child process started with the variables set, and a check in the child;
  2. a test HTTP client for every provider, through a hook in `llm/provider`, which misses every other client;
  3. the variables set by the Makefile and CI, which leaves a plain `go test` unguarded.

  Dropping D4 was not offered.
* **Decision (owner, 2026-11-07): option 1,** "go with 1 and proceed". MADR D4 is amended. Step 2 and its seen-failing
  case are annotated above. No file joins the phase.
* **Not decided:** an issue on the MCP SDK about its start-up read, which waits for the owner's word.

### P1 record (2026-11-07)

On the owner's "committed and pushed. proceed." after P0 (`1477ddf`), and, after deviation 2, "go with 1 and
proceed". Start state (C1): `main` at `1477ddf`, equal to `origin/main`, the tree clean.

**Step 1, the stub.**

* `internal/cli/agent.go`: `var modelFacts = provider.ModelInfo` (deviation 1), and the `ModelInfo` hook calls it.
* `testmain_test.go`: `offlineModelFacts` calls `provider.ModelInfo(id, "", "", nil, model, mj)`, and `TestMain`
  sets it beside `listModels`.
* `TestNoBrowserOrSignIn` checks it by identity.

**Step 2, the guard,** as deviation 2 changed it (`testmain_test.go`):

* `TestMain` without `GOBBLE_TEST_NETGUARD` calls `runBehindNetGuard`. That starts `netGuard` on a
  `net.ListenConfig` loopback listener, and runs `os.Args[0]` with `os.Args[1:]` through `exec.CommandContext`.
  * The child's environment is `guardedEnv`: every proxy variable, `NO_PROXY`, `no_proxy` and the marker dropped,
    compared case-blind for Windows; the four proxy variables set to the guard; and the marker added.
  * The parent exits with the child's code. When the guard recorded a host, it prints
    `tests tried to reach the network (0046-MADR D4):` and `  <host> x<n>` for each, and returns 1.
* The child runs `checkNetGuard` first: `HTTPS_PROXY` set, and `http.ProxyFromEnvironment` naming it for
  `https://example.com/`. Otherwise it prints `network guard: …` and exits 2. Then it runs `TestMain` as before.
* `netGuard.answer` reads the request's first line, records `CONNECT`'s host, or a plain request's URL host,
  answers 403, and closes.

**Step 3, the comment.** `llm/provider/model.go`'s `ModelInfo` comment now says it waits for no listing, and that with
a login the SDK first asks the login for a token, which may refresh at the issuer.

**Step 4, seen failing,** by `scratchpad/p1_seenfail.py` (not committed), each on a fresh copy, with `HOME` and
`XDG_*` in the scratch directory. Run on the final code, after the lint fixes below:

| Case | Result |
| :--- | :--- |
| `TestMain`'s `modelFacts` line removed | exit 1: `modelFacts is not TestMain's: a test could open a browser or sign in`; and `tests tried to reach the network (0046-MADR D4):` `api.anthropic.com:443 x6`, `api.kilo.ai:443 x1`, `api.openai.com:443 x30`, `api.x.ai:443 x2`, `auth.x.ai:443 x12` (`api.x.ai` was 4 in an earlier run) |
| a planted `GET https://example.com/` | `Get "https://example.com/": Forbidden`; exit 1, `example.com:443 x1` |
| a planted `GET` of an `httptest` server | `<nil>`; exit 0, nothing recorded |
| the child's environment without `HTTPS_PROXY` | `network guard: HTTPS_PROXY is not set in the child`; the run fails |

The first draft, which set the variables in `TestMain`, failed the first two cases silently. That is deviation 2.

**Step 5, the guide.** `docs/guides/developing.md`, "### What the tests may touch", after "Golden transcripts".

**Step 6, gates.**

* `make pre-add-check FILES=` the three Go files: "3 file(s) clean (gofmt, golint, govulncheck)".
* `make preflight`: "preflight passed", after golangci-lint refused two drafts:
  * errorlint, on `%v` for an error in `checkNetGuard`, now split into a `%w` case and a value case;
  * noctx, on `exec.Command` and `net.Listen`, now `exec.CommandContext` and `(*net.ListenConfig).Listen`, each with
    `context.Background()`.
* `make race`: 39 ok.
* The whole module through `net_probe.py`'s outside proxy: `connections to the proxy: 0`. `internal/cli` now runs
  behind its own guard, which overrides the outside one. `tool/builtin`'s `TestBashLongOutput` failed under the
  probe's `HOME`, as it did before P1, and it passes without it.
* The Linux dev host and the Windows dev host (890 files each, Go 1.27.3): `go test -count=1 ./internal/cli/` ok,
  and `-race` ok. The planted `GET https://example.com/` on each: `Forbidden`, and the run failed naming
  `example.com:443 x1`, so the child and its environment work on both.
* The identifier scan of the changed files found nothing.

**Not done in P1:** `net_attrib.py`'s per-test count over `internal/cli` (step 6's first bullet). With the guard in
the binary, the outside proxy sees nothing of that package, so the guard's own report is its measure, and it was
empty in every run above.

**Step 7** is this record. Staged; the owner commits.

### Deviation 3 (2026-11-07): P2, how three of step 1's tests are seen failing

* **Found** at P2 step 5. Step 1 says each test is seen failing "before the code". `TestSettledWaitsForAStaleListing`,
  `TestEndListingsWaits` and `TestEndListingsBound` call what step 2 adds (`settledWithin`, `endListings`, `idle`,
  `errListingsEnded`, `endListingsWait`). Against the code before step 2 they do not build, which shows nothing.
* **Decision** (recorded, not prompted: it changes how a failure is shown, not what is checked). Each of the three is
  seen failing against a planted break of the code that makes its claim: the old `settled` condition, an
  `endListings` that does not wait, an `endListings` with no Warn record. `TestRunEndsListings` uses only helpers that
  were already there, so it is also seen failing against `HEAD`'s code. Step 5's plants are unchanged. No file joins
  the phase.

### P2 record (2026-11-07)

On the owner's "committed, proceed" after P1 (`6cf61d2`). Start state (C1): `main` at `6cf61d2`, one ahead of
`origin/main`, the tree clean.

**Step 2, the code.**

* `internal/cli/models.go`:
  * `listingOwner`: a mutex, a `sync.WaitGroup`, the listings' context and cancel, and `owned` and `ended`;
  * `e.listingCtx()`, `e.goListing(f)`, `e.ownListings(cleanup)` and `e.endListings()`;
  * `e.stillListing()` and `modelListing.idle()`;
  * `errListingsEnded`, `endListingsWait` (6 s), and the hook `unownedListing`, nil in gobble.
  * `newModelListing` takes the starter. `start` runs the first fetch through it, or in place when the run's
    listings have ended, when `fetch` answers `errListingsEnded` at once. `get`'s retry goes through it.
  * `fetch` lists under `e.listingCtx()`.
* `internal/cli/main.go`:
  * `runEnv.lists`;
  * `run` calls `e.ownListings(nil)` before the command, and `e.endListings()` after it, before `logs.close`.

**Step 3, the owner check** (`testmain_test.go`). The child sets `unownedListing = recordUnowned`, which keeps
`debug.Stack()`. After `m.Run`, it prints `<n> listing(s) started on a runEnv nobody owns; call ownListings(t, e) where
it is built (0046-MADR D2):` and the first five stacks, then exits 1.

**Step 4, the helpers.**

* `ownListings(t testing.TB, e)` (`models_test.go`). `modelEnv`, `envHere`, `startEnv`, `newLineHarness` and
  `newModelHarness` call it right after their log-closing cleanup, so it runs before that cleanup, before
  `GOBBLE_HOME` is restored, and before the home is removed.
* The first run with the check failed: `10 listing(s) started on a runEnv nobody owns`. A scratch copy printing every
  stack named seven sites in three tests:
  * `cache_test.go:64`, in `TestProviderForBuildsNamed`, twice. That is the test whose log record came after its end
    in the MADR's F5 probe;
  * `modelflags_test.go:107`, `:114`, `:123` and `:133`, in `TestDefaultModel`, through its helper;
  * `settings_test.go:291` and `:307`, in `TestDefaultProviderSetting`, two `runEnv`s, twice each.

  Each `runEnv` there now calls `ownListings` after its log-closing cleanup. The three files join P2's list, as Scope
  said they would.
* `settled` is `settledWithin(e, 2 s)`. That waits until `e.stillListing()` is empty, as well as until no provider is
  shown listing, and its failure names the listings not ended.
* `TestModelListingRetries` passes a starter on a `WaitGroup` it waits for in its cleanup.

**Step 1's tests,** written with step 2:

* `TestSettledWaitsForAStaleListing` (`choice_test.go`);
* `TestEndListingsWaits`, `TestEndListingsBound` and `TestRunEndsListings` (`models_test.go`).

`TestRunEndsListings`' first draft passed against `HEAD`'s code. `Main` cancels its own context as it returns
(`watchSignals`' `stop`), which also released claude's lister, and it marked itself done before the check. Its lister
now takes 100 ms to unwind after cancellation, as a request does. Only a run that waits sees it finish.

**Step 5, seen failing** (deviation 3), by `scratchpad/p2_seenfail.py` (not committed). Each case ran on a fresh copy
of the final code; `before-run` on `HEAD`:

| Case | Failure |
| :--- | :--- |
| `HEAD`'s code, with `TestRunEndsListings` added | `models_test.go:367: Main returned with claude's listing still running` |
| `settledWithin` without `stillListing` (today's `settled`) | `settled while claude's listing ran: {ID:claude … State:stale … Models:[c1] Age:1h0m0…s …}` |
| `endListings` returning after its cancel, without waiting | `endListings returned while claude's listing ran` |
| `endListings` without its Warn record | `warnings "", want the record naming claude` |
| `run` without `e.endListings()` | `Main returned with claude's listing still running` |
| `modelEnv` without `ownListings` | `20 listing(s) started on a runEnv nobody owns; …`; the stacks name `auth_test.go:632`, `choice_test.go:40`, `:53`, `:54`, `:92` and `models_test.go:313` |

**Step 6, `docs/architecture.md`.** `internal/cli`'s row: "A run's background model listings end with it: each is
cancelled and waited for, up to 6 s, before the log closes (0046-MADR D1)."

**Step 7, gates.**

* `make pre-add-check FILES=` the ten Go files: "10 file(s) clean (gofmt, golint, govulncheck)".
* `make preflight`: "preflight passed". The first run stopped at golangci-lint's gocritic: `typeDefFirst: definition
  of type 'modelListing' should appear before its methods`. `idle` now follows `newModelListing`.
* `make race`: 39 ok.
* `make perf`: passed:
  * `ReplyRender` 21.3 µs (baseline 30 µs);
  * `Reprint` 6.03 ms (9.4 ms);
  * `LineSessionStart` 1.13 ms (830 µs, within its budget; 1.03 ms at 0035-PLAN P31 and at 0008-PLAN's amendment);
  * `KeyDraw` 1.52 µs (1.5 µs).
* The Linux dev host and the Windows dev host, 890 files each: `go test -count=3 ./internal/cli/` ok, and `-race` ok.
* No test in any run of the package failed the network guard or the owner check.
* The identifier scan of the changed files found nothing.

**Step 8** is this record. Staged; the owner commits.

### P3 record (2026-11-07)

On the owner's "committed, proceed" after P2 (`0d489ed`). Start state (C1): `main` at `0d489ed`, two ahead of
`origin/main`, the tree clean. No deviation.

**Step 1, the tests,** first:

* `TestLogAfterCloseOpensNothing` (`main_test.go`) checks two logs. One is closed before it logged: a record after
  `close` leaves no `logs` directory. The other logged, then was closed: a record after `close` is not in the file.
* `TestReadKeptCreatesNothing` (`choice_test.go`): `readKept` in a fresh home finds nothing and leaves no `catalog`.
  `writeKept` then writes one, and `readKept` reads it.

**Step 2, the code.**

* `internal/cli/main.go`:
  * `lazyLog` loses its `sync.Once`, and takes a mutex over `opened`, `closed`, `logger` and `closer`.
  * `get` returns `discardLogger` once `closed` is set, and otherwise opens on first use. Opening is `open`, the old
    `once.Do` body.
  * `close` sets `closed`, closes the file once, and returns `nil` when called again.
* `internal/cli/choice.go`: `catalogPath(id)` resolves the path and creates nothing. `catalogFile` is `catalogPath`
  with its directory created, which only `writeKept` uses now. `readKept` uses `catalogPath`.

**Step 3, seen failing,** by `scratchpad/p3_seenfail.py` (not committed), each on a fresh scratch copy:

| Case | Failure |
| :--- | :--- |
| `main.go` and `choice.go` at `HEAD`, the new tests kept | `readKept made <TMP>/TestReadKeptCreatesNothing<n>/001/catalog: <nil>`; `a record after close made <TMP>/TestLogAfterCloseOpensNothing<n>/001/logs: <nil>` |
| `get` without its `closed` check | `a record after close made …/logs: <nil>` |
| `readKept` on `catalogFile` again | `readKept made …/catalog: <nil>` |

The same two failures first showed in the working tree, before step 2, with no planted change. They are quoted here
from the scratch runs (C2).

**Step 4, gates.**

* `make pre-add-check FILES=` the four Go files: "4 file(s) clean (gofmt, golint, govulncheck)".
* `make preflight`: "preflight passed".
* `make race`: 39 ok.
* `go test -count=1 ./internal/cli/`: ok, on the Linux dev host and on the Windows dev host, 890 files each.
* The whole package passed with the network guard and the owner check quiet. `lazyLog.open` warns through `Output`
  while holding the mutex, and no test deadlocked.
* The identifier scan of the changed files found nothing.

**Step 5** is this record. Staged; the owner commits. P4 waits for the push.

### P4 record (2026-11-07)

On the owner's "committed and pushed, proceed." after P3 (`198ae13`). Start state (C1): `main` at `198ae13`, equal to
`origin/main`, the tree clean. No deviation.

**Step 1, the probes again,** each on a scratch copy with `HOME` and `XDG_*` in the scratch directory. Their anchors
still matched the tree, so none needed updating.

* `settled_probe.py`, 200 runs. After `TestLastGoodOnDisk`'s last `settled`, claude's listing had ended in all 200:
  `136 x … claude stale, its listing ended: true` and `64 x … claude live, its listing ended: true`. Before P2 it had
  not ended in 5 of 100 (MADR F1).
* `late_probe.py`, at 2 ms and at 5 ms, four iterations each. `scratchpad/lp_check.py` checked the logs:
  * each iteration made exactly four writes, one of them phase two's claude (`["claude-1"]`), all in that
    iteration's own home;
  * nothing but Go's telemetry was under the scratch `HOME`'s defaults.

  `ok` at both delays. The same checker, on the 5 ms log from before P1, flags iterations 1 and 4 (`3 writes, 0 of
  phase two's claude`) and `cache/gobble/catalog/claude.json` under the defaults (MADR F2).
* `late_probe4.py` under `-race`, the whole package twice: `files under the fake default directories: none`, and no
  late file. Before P1 it found `auth.json.grok.refresh.lock` in `TestLoginAsksThroughQuestions`' home (MADR F3).
* MADR Confirmation 4 is the network guard's own report (deviation 2). No run of `internal/cli` in P1 to P4, here or
  on CI, printed `tests tried to reach the network`. The outside proxy saw 0 connections from the rest of the module
  in P1.

**Step 2, the repetitions,** on the final code:

* `CGO_ENABLED=1 go test -race -count=60 -run '^TestLoginAsksThroughQuestions$' ./internal/cli/`: ok on macOS, and ok
  on the Windows dev host.
* `go test -count=3 ./internal/cli/`: ok on macOS, on the Linux dev host and on the Windows dev host (890 files each).

**Step 3, CI.** Run `42401401130` on `198ae13` passed every job:

* `validate` on `ubuntu-24.04`, `macos-15` and `windows-2025`; `sbom`; `build`, and the identity check on all five
  platforms. `release` was skipped, with no tag, and the Docs workflow passed.
* In each `validate` job, `go test` gave `ok github.com/maccavelli/gobble-cli/internal/cli`, and `make preflight`
  gave `preflight passed`. `make race` gave `internal/cli` ok on `ubuntu-24.04` and `macos-15`; CI runs no race job
  on Windows.
* No log has the guard's or the owner check's report.

**Step 4, the records.**

* This plan is `complete`, and the MADR `accepted`, with the `docs/README.md` rows to match.
* Dated notes point here from 0039-PLAN's deviations 14 and 18, from 0037-PLAN's P7 G-local, and from 0035-PLAN's
  sighting.

**The MADR's Confirmations:**

| # | Met by |
| :--- | :--- |
| 1 | P2 `TestSettledWaitsForAStaleListing`; P4 step 1, 200 of 200 |
| 2 | P4 step 1, `late_probe.py` at 2 ms and 5 ms, and `lp_check.py` `ok` |
| 3 | P4 step 1, `late_probe4.py` under `-race` twice, no late file |
| 4 | The guard's report, empty in every run (deviation 2) |
| 5 | P1 step 4: without the stub, the guard named the five hosts and failed the run |
| 6 | P2 `TestEndListingsWaits` and `TestEndListingsBound`, each seen failing against a planted break |
| 7 | P3 `TestLogAfterCloseOpensNothing` and `TestReadKeptCreatesNothing`, each seen failing |
| 8 | P4 step 2 |
| 9 | each phase's gates, and CI run `42401401130` |

**Not done:**

* The issue on the MCP Go SDK about reading the proxy variables at package initialisation (deviation 2), which waits
  for the owner's word.
* go-llmprovider-sdk#12 is the SDK's to build. gobble does not wait for it.

### Amendment of 2026-11-07: P5 to P7, the SDK work a run starts (proposed)

* **Found,** on the owner's "write the pairs, - Using Cache.Close and OAuthSession.Wait from go-llmprovider-sdk#12,
  written up as an amendment to the 0046 records." (`main` at `f4c0775`, equal to `origin/main`). go-llmprovider-sdk
  `v1.28.0`, pinned by 0035-PLAN P34 (`3483725`), ships go-llmprovider-sdk#12 as `(*catalog.Cache).Close` and
  `(*auth.OAuthSession).Wait`. Read at that tag (`llmprovider/catalog/cache.go`, `llmprovider/auth/oauth_session.go`)
  and against gobble's call sites; the MADR's amendment of 2026-11-07 has the findings, G1 to G4.
* **Decision** (proposed, for the owner's approval with the MADR's Q3 to Q5):
  * the MADR gains D7 to D9 and Confirmations 10 to 13, and its "Not decided here" item is marked decided;
  * P5 (the run's cache), P6 (the run's sessions) and P7 (acceptance) are new, each its own commit, in that order;
  * this plan becomes `in-progress` on approval, until P7 returns it to `complete`.
* **Files added to scope:** Scope's P5 to P7 lists.
* **Answered** 2026-11-07, the owner's "Q3. 60s, Q4. as recommended, Q5. as recommended": the MADR's amendment is
  accepted. `refreshWaitBound` is 60 s (P6); `Close` takes the rest of D1's 6 s (P5); one `Sessions` per run (P6).
  P5 to P7 wait for the owner's word to execute.
* **Approved** 2026-11-07 by the owner's "committed, approved to proceed", after the records commit `1677155`. This
  plan is `in-progress` until P7.

### Deviation 4 (2026-11-07): P5, `modelFacts` takes `provider.Options`

* **Found** at P5's start (`HEAD` `a0baf9d`, the tree clean), before any code: step 2 said "`modelFacts` keeps its
  type; its default becomes a closure the `runEnv` builds over `ModelInfoWith`". Both cannot hold. `modelFacts` is a
  package variable, `func(id, apiKey, workspace string, logins provider.LoginStore, model string, mj
  *catalog.ModelsJSON) llm.Model` (`internal/cli/agent.go:75`), so its default cannot close over one `runEnv`, and none
  of its parameters can carry a cache. With its type kept, the `ModelInfo` hook (`agent.go:93-105`) cannot reach
  `CachedModel` through the run's cache. A gap in this plan's wording, not in the code.
* **Decision,** the owner's ("Take provider.Options"): `modelFacts` becomes `func(o provider.Options, id, model
  string, mj *catalog.ModelsJSON) llm.Model`, by default `provider.ModelInfoWith`. The hook builds the run's options.
  D3's stub becomes `provider.ModelInfoWith(provider.Options{}, id, model, mj)`, which passes no key, no login and no
  cache, and is still pinned by identity. P6 adds `Sessions` to the same options with no further signature change.
  Not chosen: a per-`runEnv` function, which `TestMain` could not replace for every run, weakening D3.
* **Files added to scope:** `internal/cli/testmain_test.go` (the stub) joins P5's list. The MADR's D3 has a dated
  note for its changed default.

### Deviation 5 (2026-11-08): P5, the run's cache is a `provider.Cache`

* **Found** while writing P5's tests, before any code: archtest rule 4 (`internal/archtest/check.go:308`) lets only
  `llm/provider` import go-llmprovider-sdk. Step 2 and the MADR's D7 had `runEnv` hold the SDK's `catalog.Cache`, and
  `Options.Cache` be a `*catalog.Cache`, which would put the SDK's type in `internal/cli`. `make archtest` refuses
  that edge, and `AGENTS.md` says such an edge is fixed in the code, or by amending the record and the rule together.
* **Decision,** the owner's ("provider.Cache wrapper"): `llm/provider` defines `Cache`, which holds the SDK's
  `catalog.Cache` and has `Close(ctx) error`. `Options.Cache` is a `*provider.Cache`, `runEnv` holds a
  `provider.Cache`, and `sdkOptions` passes the inner cache with `catalog.WithCache`. The boundary is unchanged. Not
  chosen: widening rule 4, which would amend 0004-MADR's package map for one type.
* **Files added to scope:** a new `llm/provider/cache.go`. The MADR's D7 has a dated note for the type.

### P5 record (2026-11-08)

The base is `a0baf9d` (F4-1b-1's commit), two ahead of `origin/main` (`f4c0775`), the tree clean. Deviations 4 and 5
above.

1. **The tests** (step 1), in two new files:
   * `llm/provider/sdkcache_test.go`, `TestOptionsCacheReachesTheSDK`: on the canned OpenAI listing, `List` given a
     `provider.Cache` lists live; after `Close`, `List` given it fails with an error matching
     `llmprovider.ErrInvalidRequest`; without it, lists live. Then `ModelInfoWith` with a key, an open cache and a
     transport whose requests wait for their context starts a background fetch, and after the cache's `Close` every
     fetch that started has returned.
   * `internal/cli/sdkcache_test.go`: `TestRunOptionsCarryTheCache` (`modelProvider` with no provider chosen and with
     one, `providerFor`, `providerOptions`, the listing's options and the `ModelInfo` hook's each carry `&e.cache`);
     `TestEndListingsClosesTheCache` (a fetch through the run's cache has returned when `endListings` does);
     `TestEndListingsCacheBound` (a fetch that ignores cancellation holds `endListings` for the bound, 50 ms in the
     test, and the Warn record names the cache).
   * **A slip, corrected before any code:** the provider test was first written with the `Write` tool to
     `llm/provider/cache_test.go`, which already existed (161 lines: `TestCacheWire`, `TestCacheUnknownRefused`,
     `TestSessionIDPerProvider` and their `wire400` helper), replacing it. `go vet` failed on `undefined: wire400`.
     The new test was copied to `sdkcache_test.go`, and the original restored from `HEAD`: `git diff --quiet HEAD --
     llm/provider/cache_test.go` exit 0. No other existing file was written; the three new files show as untracked.
2. **The code** (step 2, with deviations 4 and 5):
   * `llm/provider/cache.go`: `Cache`, holding the SDK's `catalog.Cache`, with `Close` and the unexported `option`.
   * `llm/provider/provider.go`: `Options.Cache`; `sdkOptions` appends the cache's `WithCache`.
   * `llm/provider/model.go`: `ModelInfoWith(o Options, id, model, mj)`, which passes `o`'s transport and cache to the
     lookup; `ModelInfo` calls it, its signature unchanged. `lookupModel` became `lookupModelWith`, taking the SDK
     options (gate fix 2 below).
   * `internal/cli`: `runEnv.cache` (`main.go`); `providerOptions` sets it (`modelsjson.go`); `modelProvider` sets it
     on every path, and the `ModelInfo` hook builds its options with it (`agent.go`); `modelFacts` is
     `provider.ModelInfoWith`, and `TestMain`'s stub passes empty options (`testmain_test.go`); `endListings` closes the
     cache after its group wait, both under one `endListingsWait` context, and its one Warn record adds `cache` when
     `Close` returns the context's error (`models.go`).
3. **Seen failing** (step 3), by a scratch script copying the tree per case, every case run with `HTTPS_PROXY` and
   `HTTP_PROXY` at a closed loopback port (C5):
   * **the control:** both packages exit 0;
   * **the code before P5,** `HEAD`'s files under P5's tests: both packages fail to build, `undefined: Cache`,
     `undefined: provider.ModelInfoWith`, `e.cache undefined`, `too many arguments in call to e.endListings`;
   * **`sdkOptions` without the cache:** `sdkcache_test.go:55: a listing given the closed cache: {… Live:true …},
     want the SDK's refusal, ErrInvalidRequest`;
   * **`ModelInfoWith` dropping its SDK options:** `sdkcache_test.go:69: the lookup started no background fetch`;
   * **`providerOptions` without the cache:** `sdkcache_test.go:78: provider 2 built with cache 0x0`, `:85:
     providerOptions' cache 0x0`, `:95: the listing's cache 0x0`;
   * **`modelProvider`'s unchosen path without it:** `sdkcache_test.go:78: provider 0 built with cache 0x0`;
   * **the hook without it:** `sdkcache_test.go:106: the ModelInfo hook's cache 0x0`;
   * **`endListings` without `Close`:** `sdkcache_test.go:119: after endListings, 0 of 1 fetches had returned`, and
     `:142: endListings took 8.625µs, want the bound, 50ms`, `:145: warnings "", want the record naming the cache`.
   * **A C5 slip:** the script's first run set no proxy variables. Its `ModelInfoWith`-without-SDK-options case let
     the SDK's background fetch use its default client towards `api.openai.com` with the test's fake key
     (`sk-facts-…`); `llm/provider`'s test binary has no guard. Whether a connection was made is not known. The script
     was changed to set the proxy variables for every case, and every case above is from that run.
4. **Found by the gates and fixed in the phase** (`golangci-lint`):
   * `contextcheck` at `main.go:89`: `endListings` built its bound from `context.Background()` while `run` has one.
     `endListings` now takes the caller's context and drops its cancellation (`context.WithoutCancel`), since `run`'s
     may be done as the run ends. `ownListings` only marks the listings owned; the tests' `ownListings(t, e)`
     registers `e.endListings(t.Context())` itself. `sloglint` then asked for `WarnContext`, which the record uses.
     Files: `internal/cli/models_test.go` (the helper and two calls) joins P5's.
   * `unparam` at `model.go:146`: `lookupModel` was left with test callers only, all passing `"openai"`. It is
     removed; `model_test.go`'s four calls use `lookupModelWith(…, nil, …)`. `llm/provider/model_test.go` joins P5's
     files.
5. **Gates** (step 4), after the fixes:
   * `make pre-add-check` on the twelve changed Go files: "12 file(s) clean (gofmt, golint, govulncheck)";
   * `make preflight`: "preflight passed", its 15 gates; `make race`: 39 ok;
   * `make perf` passed: `ReplyRender` 21.8 µs (baseline 30 µs), `Reprint` 5.99 ms (9.4 ms), `LineSessionStart`
     1.03 ms (830 µs, within its budget), `KeyDraw` 1.48 µs (1.5 µs).
   * The Linux machine and the Windows laptop, the fixed tree shipped to each, 893 files, no `._*` file: `go test`
     and race 38 ok each; `go test -count=3 ./internal/cli/ ./llm/provider/` ok on both; `make check-records
     markdownlint` 0 issues on both; `make perf` passed on both (the Linux machine: `ReplyRender` 96.0 µs of 110 µs,
     `Reprint` 18.7 ms of 29 ms, `LineSessionStart` 4.36 ms of 2.9 ms, over its baseline and within its budget,
     `KeyDraw` 4.47 µs of 5 µs; the Windows laptop: 25.1 µs of 54 µs, 4.09 ms of 12 ms, 5.00 ms of 6.9 ms, 1.28 µs
     of 1.4 µs); preflight passed `lint` and stopped at `verify-build-metadata` on both, as on every copy. The first
     shipped copy, before the gate fixes, had failed `lint` on both with the same two findings.
6. **Not done here:** P6 (the sessions) and P7 (acceptance), each its own commit; `docs/architecture.md`'s rows, which
   are P7's.

### P6 record (2026-11-08)

The base is `10c21c4` (P5's commit), one ahead of `origin/main` (`a0baf9d`), the tree clean. No deviation: the
additions below change no decision.

1. **The code** (step 2), written before its tests this time; C2 is met by step 3's runs against `HEAD`'s code and
   planted breaks:
   * `llm/provider/sessions.go`: `Sessions`, a mutex and a map of provider id to `*auth.OAuthSession`; `get(id,
     load)`, `Drop(id)`, `Wait(ctx)`, which waits for each held session in turn and joins `"<id>: <ctx error>"` for
     each still refreshing. **Added:** `Held() []string`, the providers held, sorted, so `internal/cli`'s tests can see
     the set without naming the SDK's session type (archtest rule 4).
   * `llm/provider/provider.go`: `Options.Sessions`; `login` returns the set's session when it is set, loading it once.
     `llm/provider/model.go`: `ModelInfoWith` passes `o.Sessions` to `login`.
   * `internal/cli`: `runEnv.sessions` (`main.go`); `providerOptions` sets it beside the cache (`modelsjson.go`);
     `modelProvider` on every path and the `ModelInfo` hook carry it (`agent.go`); `refreshCredentials` drops the
     provider's session (`credentials.go`); `endListings`, after the listings and the cache under D1's bound, waits for
     the sessions under `refreshWaitBound`, 60 s (Q3), from the same caller's context without its cancellation, and
     writes a Warn record, "a sign-in refresh was still running at the end of the run; its rotated login may not be
     saved", with the providers, when that wait ends first (`models.go`).
2. **The tests** (step 1), in `llm/provider/sessions_test.go` and `internal/cli/oauthsessions_test.go`
   (`internal/cli/sessions_test.go` already exists, so the new file takes another name):
   * `TestSessionsShareOneSession` and `TestSessionsWait`, as step 1 names them. **Added:**
     `TestModelInfoWithUsesTheSessions` (a login looked up through `ModelInfoWith` is held in the run's set; the
     cache is closed first, so the SDK asks for a token and starts no fetch), since no step 1 test would see
     `ModelInfoWith` drop `Sessions`.
   * `TestRefreshCredentialsDropsTheSession`, `TestEndListingsWaitsForARefresh` and `TestEndListingsRefreshBound` (the
     plan's `TestEndListingsWaitsForARefresh` bound case, as a test of its own). **Added:** `TestRunOptionsCarryTheSessions`,
     P5's options test for the sessions.
   * The issuer is a loopback token endpoint whose refresh waits to be released, then rotates the refresh token to
     `refresh-2`. A listing's transport passes the issuer's host to the real transport and fails every other request:
     the SDK gives a session with no client the listing's (go-llmprovider-sdk `catalog/discovery.go:291`,
     `UseDefaultHTTPClient`), so a transport that failed everything kept the refresh from the issuer, which the first run
     showed (`no refresh reached the issuer`).
   * **Fixed before the gates:** the first breaks run hung at "login ignores Sessions". A test that failed early left
     the issuer's handler waiting, and `httptest.Server.Close` waits for it. The issuer's release is now a `sync.Once`
     function its own cleanup calls before `Close`; the run was stopped (no process left behind) and run again, with
     `-timeout=120s` per test run.
3. **Seen failing** (step 3), by a scratch script copying the tree per case, every case with `HTTPS_PROXY` and
   `HTTP_PROXY` at a closed loopback port (C5) from its first run:
   * **the control:** both packages exit 0;
   * **the code before P6** under P6's tests: both fail to build, `undefined: Sessions`, `e.sessions undefined`,
     `o.Sessions undefined`;
   * **`login` ignoring `Sessions`:** `sessions_test.go:62: second login through the same Sessions: 0x… , want the
     first`, `:102: held [] after ModelInfoWith`, `:129: Wait with a refresh blocked: <nil>`; `oauthsessions_test.go:135:
     held [] after a Grok listing`, `:177: endListings returned while the refresh was blocked at the issuer`, `:214`,
     `:217`;
   * **`ModelInfoWith` dropping `Sessions`:** `sessions_test.go:102: held [] after ModelInfoWith with a login`;
   * **`Sessions.Wait` waiting for nothing:** `sessions_test.go:129: Wait with a refresh blocked: <nil>, want the
     deadline, naming grok`;
   * **`providerOptions` without the sessions:** `oauthsessions_test.go:109: provider 2 built with sessions 0x0`, `:113`,
     `:135`, `:177`;
   * **`modelProvider`'s unchosen path without them:** `oauthsessions_test.go:109: provider 0 built with sessions 0x0`;
   * **the hook without them:** `oauthsessions_test.go:124: the ModelInfo hook's sessions 0x0`;
   * **`refreshCredentials` without `Drop`:** `oauthsessions_test.go:139: held ["grok"] after refreshCredentials(grok),
     want none`;
   * **`endListings` without `Wait`:** `oauthsessions_test.go:177: endListings returned while the refresh was blocked`,
     `:214: endListings took 4.167µs, want the refresh bound, 50ms`, `:217: warnings "", want the record naming grok`
     (its first draft left `refreshing` unused and did not build; rewritten to keep it used);
   * **the session built without its store** (added, so the rotation assertions fail alone): `sessions_test.go:138:
     stored login … after Wait, want the rotated refresh token, refresh-2`, `oauthsessions_test.go:187: stored Grok login
     … after endListings, want the rotated refresh token`;
   * **Q3's bound against D1's**, step 3's 7 s refresh on scratch copies (the test's release delayed to 7 s, its
     final wait to 15 s): with `refreshWaitBound` it passes, `ok … 7.926s`; with the wait under `endListingsWait`,
     `oauthsessions_test.go:177: endListings returned while the refresh was blocked at the issuer` at 6.03 s.
4. **Gates** (step 4): `make lint` 0 issues; `make pre-add-check` on the ten changed Go files: "10 file(s) clean
   (gofmt, golint, govulncheck)".
   * `make preflight`: "preflight passed", its 15 gates; `make race`: 39 ok; `make perf` passed: `ReplyRender` 22.8 µs
     (baseline 30 µs), `Reprint` 5.77 ms (9.4 ms), `LineSessionStart` 1.09 ms (830 µs, within its budget), `KeyDraw`
     1.54 µs (1.5 µs).
   * The Linux machine and the Windows laptop, the tree shipped to each, 896 files, no `._*` file: `go test` and race 38
     ok each; `go test -count=3 ./internal/cli/ ./llm/provider/` ok on both; `make check-records markdownlint` 0 issues
     on both; `make perf` passed on both (the Linux machine: 77.9 µs of 110 µs, 19.1 ms of 29 ms, `LineSessionStart`
     4.10 ms of 2.9 ms, over its baseline and within its budget, 4.29 µs of 5 µs; the Windows laptop: 24.9 µs of 54 µs,
     3.98 ms of 12 ms, 5.14 ms of 6.9 ms, 1.28 µs of 1.4 µs); preflight stopped at `verify-build-metadata` on both, as on
     every copy.
5. **Not done here:** P7 (acceptance, the probes of Confirmation 10 and 11, `docs/architecture.md`'s rows, and the
   records' close).
