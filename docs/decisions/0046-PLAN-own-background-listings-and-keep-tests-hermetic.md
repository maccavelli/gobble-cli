---
status: proposed
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

**Out of scope:**

* MADR option D, the directories resolved per `runEnv`;
* production `ModelInfo` fetches ("Not decided here");
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
     proxied).
5. **The guide.** `docs/guides/developing.md` gains "### What the tests may touch", after "Golden transcripts":
   * `internal/cli`'s tests reach no network, and `TestMain`'s proxy fails the run naming any host one tried;
   * each test keeps to its own temporary directories;
   * a test that starts listings owns them (P2's `ownListings`).
6. **Gates** (C6). `make pre-add-check` on the changed Go files, `make preflight`, `make race`.
   * The proxy probe (`net_attrib.py`) over `internal/cli`: 0 tests connect (MADR Confirmation 4).
   * The whole module through the proxy (`net_probe.py`): 0 connections.
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

## Rollout and Rollback

* **Rollout.** Each phase is one commit on `main`. P1 alone stops the network traffic, and can ship first if a release
  is wanted before P2.
* **Rollback.**
  * P1 reverts cleanly.
  * P2 reverts with its test changes, and P3 reverts alone.
  * If `make perf` or a user's exit shows the 6 s bound being reached in normal use, that is a deviation (C4). The
    bound is not raised or removed to hide it.

## Execution record

Not started.
