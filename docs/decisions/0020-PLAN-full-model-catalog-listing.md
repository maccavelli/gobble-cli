---
status: completed
date: 2026-10-13
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# PLAN 0020 — Pin go-llmprovider-sdk v1.11.0 and offer each provider's full curated catalog from a live listing

Implements [0020-MADR-full-model-catalog-listing.md](0020-MADR-full-model-catalog-listing.md) decisions D1–D8, closing findings F1–F11, and gobble-cli issue 1.

## Goal

1. `go list -m github.com/maccavelli/go-llmprovider-sdk` reports `v1.11.0`; `go.mod` changes in that one requirement; `go.sum` holds `h1:oCZmUe6hGMCZBS9lYqogTyndueF2CyWowJE6xxUqHoE=` and `h1:en9k+ZhlMT41q/+FtXQrLE/3qDaBrWFr8TeLyZw2+Jg=`. *(P1 deviation 3: `v1.12.0`, with `h1:3nbuUj/BF/mRaXIEF2r1y+oSKB7KdZ6+T2arJZtmLv0=` and the same `go.mod` sum.)*
2. With a credential set and a listing that answers, the ACP `model` option offers the listing's models, the listing's first selected; `set_config_option` and `/model` accept any of them, and the next request names it. A test shows each over a canned listing, and fails with today's `provider.Models` as the source.
3. With the listing failing, the option offers the static models and `StaticModels[0]`, a session starts, and the warning names the provider and the failure's kind, never its text.
4. One listing per process: started when the agent starts, waited for by the first model choice, kept when live, retried at most once a minute when it failed. No generation is sent, and no test lists over the network.
5. The gates pass, and each new test has been seen failing on a scratch copy.

## Scope

### In scope (the only files any phase may touch)

| Phase | Decisions | Files |
| :--- | :--- | :--- |
| P1 | D1, D2 | `go.mod`, `go.sum`; `llm/provider/provider.go`; `llm/provider/list.go` (new); `llm/provider/list_test.go` (new); `llm/provider/ambient_test.go` (a comment); `docs/architecture.md`; `README.md` |
| P2 | D3–D7 | `internal/cli/agent.go`; `internal/cli/main.go` (two `runEnv` fields); `internal/cli/models.go` (new); `internal/cli/models_test.go` (new); `internal/cli/testmain_test.go` |
| P3 | D8 | `docs/decisions/0019-MADR-sibling-libraries-current-pins-and-integration.md`; `docs/decisions/0005-PLAN-v1-feature-scope.md`; `docs/reports/0014-REPORT-llmprovider-sdk-integration.md`; `docs/README.md`; this plan and its MADR |

### Out of scope

- **`acpserver`:** its option, `set_config_option` and `/model` already read `ModelChoice.Models` (MADR D5); its exported API does not change.
- **0005-PLAN F4-2:** `models.json`, the Pi-shaped catalog, `gobble models list`, `thought_level` per model, `size` and `cost`; F4-2 builds on this listing.
- **A disk cache of listings:** the owner chose to list in every mode; a persistent catalog is F4-2's.
- **`--offline`:** 0005-PLAN X3 turns the listing off when it lands (P3's note).
- **A model search over `Usable`:** nothing in gobble searches models yet.
- **Ollama:** `Choose` picks only providers that take a key; Ollama is not reached.

## Stability rule

Every phase ends with, in order:

```bash
export GOTOOLCHAIN=go1.27.2
make pre-add-check FILES="<the phase's .go files>"         # P3 has none
python -I <scratch>/q428_gates_0019.py 0020-<phase>         # lint for linux, darwin, windows; Windows go test; Windows -race;
                                                            # Windows make preflight; the second Linux machine's -race; the macOS laptop's go test
python -I <scratch>/q252_check_index.py 0020-<phase>        # gofmt, vet, test and check-records on the staged index alone
python -I <scratch>/q261_lint_records.py --negative <each record the phase touches>
```

- `q428_gates_0019.py` takes the phase as a label; it is reused unchanged. No gate runs in WSL. Host aliases stay in the scripts and are never written into a record.
- **Commits.** The agent stages each phase with `git add` and never commits. The owner commits each phase with `git commit --no-edit` before the next begins. No push and no tag without the owner's request in that turn.
- **Negatives** run on a scratch copy of the working tree, after a control run of the same tests passes, and each asserts that its edit landed.

## Cross-cutting contracts

- **C1. No test reaches a provider.** `llm/provider`'s tests list over canned transports; `internal/cli`'s lister is the offline one unless a test replaces it.
- **C2. No credential in output.** No log line, error a test prints, or record carries a key or a provider's error text; the warning carries `FailureKind` only.
- **C3. No generation to list.** `List` calls `catalog.List` only; probes stay off.
- **C4. No test is weakened.** `TestAmbientDefaultModels` and `TestDefaultProviderSetting` pass unchanged in their assertions; only `ambient_test.go`'s comment names the new tag.
- **C5. `acpserver` is unchanged** (`git diff --exit-code HEAD -- acpserver` at each phase).

**C2 is the contract most at risk:** logging `r.Err` is the obvious line to write, and OpenAI's 401 text carries a masked key fragment (MADR F5). P2's test plants a fragment and reads both the warning stream and the log file.

## Dependency and delivery order

| # | Phase | Why here |
| :--- | :--- | :--- |
| 1 | P1 — the pin and `List` | `List` needs `v1.11.0`'s default; the pin lands with its first use (AGENTS.md, Dependencies). |
| 2 | P2 — the model choice | Needs `List`. |
| 3 | P3 — the records | Cites P1's and P2's commits. |

0019-PLAN P5 is committed before P1 (the owner's order, 2026-10-13).

## Implementation Steps

### P1 — The pin and `List` (D1, D2; closes F1, F2, F5, F6, F7)

1. `python scripts/check_libs.py --lib go-llmprovider-sdk` reports `v1.11.0` and every check ok, `api: ok`, exit 0. *(2026-10-13, P1 deviation 3: `v1.12.0`.)*
2. `go get github.com/maccavelli/go-llmprovider-sdk@v1.11.0`, `go mod tidy`; `go.mod` changes in one line; `go.sum` holds the two sums of Goal 1, and `v1.10.2`'s module sum `h1:llPpBo+EKnQE4ksae+ZBK9xFOQ6z5NpK18H4cosmyJY=` is gone. *(2026-10-13, P1 deviation 3: `v1.12.0`.)*
3. **`llm/provider/provider.go`, `Options`:** after `HTTPClient`, add

   ```go
       // ModelMetadataURL replaces List's model metadata document, for
       // tests; New never fetches it.
       ModelMetadataURL string
   ```

4. **`llm/provider/list.go` (new):**

   ```go
   package provider

   import (
       "context"
       "errors"
       "fmt"

       "github.com/maccavelli/go-llmprovider-sdk/llmprovider"
       "github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"

       "github.com/maccavelli/gobble-cli/llm"
   )

   // errNothingListed is a listing that answered but recommended no model.
   var errNothingListed = errors.New("provider: the listing recommended no model")

   // Listing is one model listing of a provider (0020-MADR D2).
   type Listing struct {
       // Models are the provider's full curated catalog, ranked by the
       // capable profile, when Live; otherwise its static models.
       Models []string
       // Live reports whether Models came from the provider's listing.
       Live bool
       // Err is why the listing was not live; nil when it was.
       Err error
   }

   // List lists the built-in provider id's models with apiKey, through the
   // SDK's catalog.List with the capable profile, model metadata on and no
   // limit. It sends no generation and is bounded by the SDK at 10 s. It
   // passes no logger, so the SDK logs nothing of it. A listing that fails
   // or recommends nothing gives the static models, Live false and the
   // reason in Err; only an unknown id or a refused option is an error. o's
   // APIKey, Model and Logger are ignored.
   func List(ctx context.Context, id, apiKey string, o Options) (Listing, error) {
       opts := []llmprovider.Option{
           catalog.WithProfile(catalog.ProfileCapable),
           llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),
       }
       if o.HTTPClient != nil {
           opts = append(opts, llmprovider.WithHTTPClient(o.HTTPClient))
       }
       if o.ModelMetadataURL != "" {
           opts = append(opts, llmprovider.WithModelMetadataURL(o.ModelMetadataURL))
       }
       cat, err := catalog.List(ctx, llmprovider.ProviderID(id), llmprovider.NewStaticToken(apiKey), opts...)
       if err != nil {
           return Listing{}, fmt.Errorf("provider %s: list models: %w", id, mapError(err))
       }
       if !cat.Live || len(cat.Recommended) == 0 {
           reason := mapError(cat.Err)
           if reason == nil {
               reason = errNothingListed
           }
           return Listing{Models: Models(id), Err: reason}, nil
       }
       return Listing{Models: cat.Recommended, Live: true}, nil
   }

   // failureKinds name the errors FailureKind knows, most specific first.
   var failureKinds = []struct {
       err  error
       kind string
   }{
       {llm.ErrQuota, "quota"},
       {llm.ErrRateLimited, "rate-limited"},
       {llm.ErrAuth, "auth"},
       {llm.ErrNotPermitted, "not-permitted"},
       {llm.ErrUnavailable, "unavailable"},
       {context.DeadlineExceeded, "timeout"},
       {context.Canceled, "canceled"},
       {errNothingListed, "empty"},
   }

   // FailureKind names err's class for a log line, and never carries its
   // text, which a provider may fill with part of the credential
   // (0020-MADR F5): quota, rate-limited, auth, not-permitted, unavailable,
   // timeout, canceled, empty, or other.
   func FailureKind(err error) string {
       for _, k := range failureKinds {
           if errors.Is(err, k.err) {
               return k.kind
           }
       }
       return "other"
   }
   ```

5. **`llm/provider/list_test.go` (new),** package `provider`, each test seen failing first (P1's negatives):
   - a `canned` `http.RoundTripper` that records `METHOD host/path` and each request's `Authorization` header, answers its routes with their status and JSON body, and every other request 404 `{}`; a `hang` route blocks until the request's context ends;
   - **`TestListFullCatalog`:** an OpenAI listing (`GET api.openai.com/v1/models`, 200) of `gpt-6-luna`, `gpt-5.6-luna`, `gpt-4.1-mini`, `gpt-5.6-terra`, `gpt-6.1-sol`, `gpt-6-astra`, `gpt-5-mini`, `gpt-5-nano`, `text-embedding-4-large` and `tts-2`. `List(ctx, "openai", "sk-test", Options{HTTPClient: c})` gives `Live` true, `Err` nil, and `Models` exactly the six, then `gpt-5-nano`, `gpt-5-mini`. The requests are exactly `["GET api.openai.com/v1/models"]`, with `Authorization: Bearer sk-test`, so no generation and no metadata request;
   - **`TestListCapableProfile`:** a Kilo listing (`GET api.kilo.ai/api/gateway/models`, 200) of `kilo-auto/efficient` and `deepseek/deepseek-v4.1-flash`, in the shape `i1kilo/canned` used (MADR's evidence) gives `Models` exactly `["kilo-auto/efficient", "deepseek/deepseek-v4.1-flash"]`; the utility profile gives only the second;
   - **`TestListFetchesMetadata`:** an OpenCode Zen listing, with `ModelMetadataURL` `https://metadata.test/<the test's name>-<time.Now().UnixNano()>.json` so the SDK's in-process cache never holds it, records a request `GET metadata.test/<that path>` after `GET opencode.ai/zen/v1/models`; *(deviation 1: the Zen listing answers 200 with two models, and the test asserts `Live` and both requests, in any order; deviation 2: the path is `/<the test's name>-<pid>-<n>.json`, `n` a process-wide counter)*;
   - **`TestListFallsBack`:** the OpenAI listing answering 401 `{"error":{"message":"Incorrect API key provided: sk-pl***nt","code":"invalid_api_key"}}` gives `Live` false, `Models` equal to `Models("openai")`, `errors.Is(Err, llm.ErrAuth)` and `FailureKind(Err) == "auth"`; a hanging listing under a 300 ms context gives `Live` false, `FailureKind(Err) == "timeout"`, and returns within 2 s; `List(ctx, "nope", "k", Options{})` is an error;
   - **`TestFailureKind`:** each sentinel, wrapped with `fmt.Errorf("x: %w", ...)`, gives its kind; `errors.New("sk-pl***nt")` gives `other`, and no kind contains the error's text.
6. **`llm/provider/ambient_test.go:13`:** "default model at go-llmprovider-sdk v1.10.2" becomes "default model at go-llmprovider-sdk v1.10.2 and v1.11.0", since `v1.11.0` changes no static list (SDK 0038-MADR D3) and the test passed on it (step 1). *(P1 deviation 3: "v1.10.2 to v1.12.0"; the test passed at `v1.12.0` in its check too.)*
7. **Docs:** `docs/architecture.md:110` and `README.md:84` name `v1.10.2`; each becomes `v1.11.0`. *(2026-10-13, P1 deviation 3: `v1.12.0`.)*

**P1's negatives** (`q498_0020_p1_negatives.py`, a scratch copy, control first):

1. `List` adds `catalog.WithLimit(catalog.MaxListed)`: `TestListFullCatalog` fails, `Models` the six.
2. `List` without `catalog.WithProfile(catalog.ProfileCapable)`: `TestListCapableProfile` fails, without `kilo-auto/efficient`.
3. `List` adds `llmprovider.WithoutModelMetadata()`: `TestListFetchesMetadata` fails, no `metadata.test` request.
4. `List` returns `cat.Err` as its error when the listing is not live: `TestListFallsBack` fails.
5. `FailureKind` returns `err.Error()` for an unknown error: `TestFailureKind` fails.

**P1's verification:**

```text
go list -m github.com/maccavelli/go-llmprovider-sdk                         -> v1.11.0   (deviation 3: v1.12.0)
git diff -U0 HEAD -- go.mod | grep -c '^[-+]\s'                              -> 2
grep -c 'go-llmprovider-sdk v1.11.0' go.sum                                  -> 2   (deviation 3: v1.12.0)
grep -rn 'WithLimit\|WithLogger' llm/provider/list.go                        -> no match
go test -count=1 ./llm/provider                                              -> ok
make archtest                                                                -> ok
git diff --exit-code HEAD -- acpserver                                       -> no diff
```

### P1's deviations, 2026-10-13

- **Deviation 1. Step 5's `TestListFetchesMetadata` is flaky as written.**
  - **Found:** the plan's test lists OpenCode Zen with a failing listing and expects the metadata request after the listing's. The SDK starts the metadata fetch concurrently with the listing (`llmprovider/catalog/model_metadata.go`, `startModelMetadata`) and waits for it only when the listing answers, so after a failed listing `List` can return before the request is made. Measured (`i1kilo/zenwait`, 50 runs each, a metadata URL of its own per run): with the failing listing the request was made before `List` returned in 14 of 50; with a listing that answers 200, in 50 of 50. P1's negatives run saw it fail once, in negative 5, which does not touch it.
  - **Decided** ("Live Zen listing"): the canned Zen listing answers 200 with two models, so `List` waits for the metadata; the test asserts `Live` and both requests, in any order. Step 5 is annotated. No file added.
- **Deviation 2. Step 5's metadata URL is not unique on Windows.**
  - **Found:** the URL `<the test's name>-<time.Now().UnixNano()>` repeats: on the Windows host 10 000 consecutive `time.Now().UnixNano()` calls gave one distinct value (`i1clock`). Under `go test -count=50`, a run that reuses a URL is answered from the SDK's remembered failure, which is not retried for a minute, with no request: 44 of 50 runs failed. One `-count=1` run, as the gates use, passes.
  - **Decided** ("Process-wide counter"): the path is `/<the test's name>-<pid>-<n>.json`, `n` from an atomic counter, unique within the process by construction and across processes by the pid; shown with `-count=50`. Step 5 is annotated. No file added.

- **Deviation 3. go-llmprovider-sdk `v1.12.0` was tagged after this plan (MADR F12).**
  - **Found:** checking the pin before staging P1, `v1.12.0` (`57cbd79e0790`, 2026-10-13 14:32 UTC-5) is the latest tag. It adds `Request.Cache` for request-side prompt caching, whose zero value leaves every request unchanged (SDK 0039-MADR D1, R45); gobble sets no such field. `check_libs.py --lib go-llmprovider-sdk --tag v1.12.0 --from v1.11.0`: every check ok; apidiff: four additions only; `llmprovider/catalog` unchanged.
  - **Decided** ("v1.12.0 in P1"): P1 pins `v1.12.0`, with no other code change; `go.sum` holds `h1:3nbuUj/BF/mRaXIEF2r1y+oSKB7KdZ6+T2arJZtmLv0=` and `h1:en9k+ZhlMT41q/+FtXQrLE/3qDaBrWFr8TeLyZw2+Jg=`. Goal 1, steps 1, 2, 6 and 7, P1's and the whole plan's verification and P3 step 1 are annotated; P1's tests, the 50-run check, the negatives and the gates are rerun. Setting `Request.Cache` is deferred to its own record. No file added.

### P2 — The model choice (D3–D7; closes F3, F4, F8, F9, F10)

1. **`internal/cli/models.go` (new):** *(deviation 5: the warning logs typed attributes, `slog.String`, with `e.ctx`, as `line.go:183` does)*

   ```go
   package cli

   import (
       "os"
       "slices"
       "sync"
       "time"

       "github.com/maccavelli/gobble-cli/llm/provider"
   )

   // listModels lists a provider's models. TestMain replaces it with a
   // lister that fails offline, so no test lists over the network
   // (0020-MADR D7).
   var listModels = provider.List

   // listRetry is the least time between a failed listing and its retry
   // (0020-MADR D3).
   const listRetry = time.Minute

   // modelListing is the process's listing of one provider's models
   // (0020-MADR D3): started once, waited for by the first reader, kept
   // when live, and retried in the background at most once per listRetry
   // when it failed.
   type modelListing struct {
       fetch func() provider.Listing
       now   func() time.Time
       once  sync.Once
       ready chan struct{} // closed when the first listing has ended

       mu       sync.Mutex
       result   provider.Listing
       at       time.Time // when result was listed
       retrying bool
   }

   func newModelListing(fetch func() provider.Listing, now func() time.Time) *modelListing {
       return &modelListing{fetch: fetch, now: now, ready: make(chan struct{})}
   }

   // start begins the first listing, once.
   func (l *modelListing) start() {
       l.once.Do(func() {
           go func() {
               r := l.fetch()
               l.mu.Lock()
               l.result, l.at = r, l.now()
               l.mu.Unlock()
               close(l.ready)
           }()
       })
   }

   // get waits for the first listing, starts a retry when the kept one
   // failed listRetry or more ago, and returns the kept one.
   func (l *modelListing) get() provider.Listing {
       l.start()
       <-l.ready
       l.mu.Lock()
       defer l.mu.Unlock()
       if !l.result.Live && !l.retrying && l.now().Sub(l.at) >= listRetry {
           l.retrying = true
           go l.retry()
       }
       return l.result
   }

   // retry lists again and keeps the answer, live or not, with its time.
   func (l *modelListing) retry() {
       r := l.fetch()
       l.mu.Lock()
       defer l.mu.Unlock()
       l.result, l.at, l.retrying = r, l.now(), false
   }

   // modelListing is the listing of sel's models for this process, made
   // with sel's credential. A failed listing is the static models, and a
   // warning names the provider and the failure's kind, never its text
   // (0020-MADR D6).
   func (e *runEnv) modelListing(sel provider.Selection) *modelListing {
       e.listingsMu.Lock()
       defer e.listingsMu.Unlock()
       if l, ok := e.listings[sel.ID]; ok {
           return l
       }
       key := os.Getenv(sel.EnvVar)
       o := provider.Options{ClientName: appName, ClientVersion: identity().Version}
       l := newModelListing(func() provider.Listing {
           r, err := listModels(e.ctx, sel.ID, key, o)
           if err != nil {
               r = provider.Listing{Models: provider.Models(sel.ID), Err: err}
           }
           if !r.Live {
               e.log().Warn("model listing failed; offering the curated static list", "provider", sel.ID, "kind", provider.FailureKind(r.Err))
           }
           return r
       }, time.Now)
       if e.listings == nil {
           e.listings = map[string]*modelListing{}
       }
       e.listings[sel.ID] = l
       return l
   }

   // startModelListing starts the chosen provider's listing when a
   // credential is set, so the first session rarely waits for it
   // (0020-MADR D3).
   func (e *runEnv) startModelListing() {
       if sel, err := provider.Choose(os.Getenv, e.defaultProvider()); err == nil {
           e.modelListing(sel).start()
       }
   }

   // models is the listed models and the default: the listing's first when
   // it is live, else the static default (0020-MADR D4). List never gives a
   // live listing with no model; the length check keeps a replaced lister
   // from panicking here.
   func models(sel provider.Selection, l provider.Listing) ([]string, string) {
       if l.Live && len(l.Models) > 0 {
           return slices.Clone(l.Models), l.Models[0]
       }
       return slices.Clone(l.Models), sel.Model
   }
   ```

2. **`internal/cli/main.go`, `runEnv`:** after `now func() time.Time`, add

   ```go
       // listingsMu guards listings, the process's model listings by
       // provider (0020-MADR D3).
       listingsMu sync.Mutex
       listings   map[string]*modelListing
   ```

3. **`internal/cli/agent.go`:**
   - `agentOptions` (`:39`) begins with `e.startModelListing()`;
   - `modelChoice` (`:80-88`) becomes

     ```go
     // modelChoice is the chosen provider's listed models, the listing's
     // first the default, or its static models and StaticModels[0] when the
     // listing failed; none without a credential (0020-MADR D3, D4).
     func (e *runEnv) modelChoice() acpserver.ModelChoice {
         sel, err := provider.Choose(os.Getenv, e.defaultProvider())
         if err != nil {
             return acpserver.ModelChoice{}
         }
         ids, def := models(sel, e.modelListing(sel).get())
         return acpserver.ModelChoice{Provider: sel.ID, Models: ids, Default: def}
     }
     ```

4. **`internal/cli/testmain_test.go`:** `TestMain` sets `listModels = offlineList` before `m.Run()`, with

   ```go
   // errOfflineList is the tests' lister's answer: no test lists over the
   // network (0020-MADR D7).
   var errOfflineList = errors.New("tests list no models")

   func offlineList(context.Context, string, string, provider.Options) (provider.Listing, error) {
       return provider.Listing{}, errOfflineList
   }
   ```

   and its comment adds "and makes the model lister fail offline (0020-MADR D7)".
5. **`internal/cli/models_test.go` (new),** each test seen failing first (P2's negatives); a helper `withLister(t, fn)` sets `listModels` and restores it with `t.Cleanup`; each test sets `GOBBLE_HOME` to `t.TempDir()` and builds `runEnv` as `TestDefaultProviderSetting` does, with `logs: &lazyLog{out: out, stderr: logBuf}`:
   - **`TestListerIsOffline`:** `listModels(t.Context(), "openai", "k", provider.Options{})` gives an error matching `errOfflineList`;
   - **`TestModelChoiceFromListing`:** `OPENAI_API_KEY=test-only`; the lister answers `Listing{Live: true, Models: ["gpt-6-astra", "gpt-6-luna", "gpt-5-mini"]}`; `e.modelChoice()` is `{Provider: "openai", Models: those three, Default: "gpt-6-astra"}`, and the lister was called with `"openai"` and `"test-only"`;
   - **`TestModelChoiceListsOnce`:** a counting lister; `e.agentOptions()`, then `e.modelChoice()` five times from five goroutines, list once;
   - **`TestModelChoiceFallsBack`:** the lister answers an error wrapping `llm.ErrAuth` whose text holds `sk-pl***nt`; `e.modelChoice()` is `{openai, provider.Models("openai"), provider.Models("openai")[0]}`; `logBuf` holds `model listing failed; offering the curated static list`, `provider=openai` and `kind=auth`; neither `logBuf` nor `<state>/logs/gobble.log` holds `sk-pl`;
   - **`TestModelListingRetries`:** `newModelListing` with a fake clock and a fetch that fails first and succeeds after; `get()` at the start and at 59 s fetch once in all; at 60 s it starts one retry (a second `get()` meanwhile starts none), and once the retry has stored its answer, polled for up to 2 s, `get()` is live; later `get()`s at any time fetch no more;
   - **`TestNoCredentialNoListing`:** no credential; a counting lister; `e.agentOptions()` and `e.modelChoice()` give `ModelChoice{}` and no listing;
   - **`TestModelOptionOffersListedModels`:** `OPENAI_API_KEY=test-only`; the lister answers live `["gpt-6-luna", "gpt-5.6-luna", "gpt-4.1-mini", "gpt-5.6-terra", "gpt-6.1-sol", "gpt-6-astra", "gpt-5-nano", "gpt-5-mini"]`; an agent `acpserver.New(acpserver.Options{Models: e.modelChoice, Provider: <an llmtest script answering "ok">, Store: session.NewMemoryStore()})` over `acptest.Connect`:
     - `session/new`'s `model` option offers the eight, `gpt-6-luna` current;
     - `set_config_option` `model` = `gpt-5-mini` succeeds;
     - a prompt `/model gpt-5-nano` answers `Model: gpt-5-nano`;
     - a prompt `x` sends a request whose model is `gpt-5-nano`.
     *(Deviation 4: through `e.startAgent` and `acpclient`, not `acptest.Connect`: `/model gpt-5-nano`'s `config_option_update` offers the eight with `gpt-5-nano` current; `session/new`'s current value is `TestModelChoiceFromListing`'s default; `set_config_option` is `acpserver`'s `TestConfigOptionsPersist`.)*
6. `TestDefaultProviderSetting` (`internal/cli/settings_test.go:255-266`) passes unchanged: under the offline lister its choice is the static models, as it asserts (C4).

**P2's negatives** (`q499_0020_p2_negatives.py`, a scratch copy, control first):

1. `modelChoice` takes `provider.Models(sel.ID)` and `sel.Model` (today's source): `TestModelOptionOffersListedModels` fails (`gpt-5-mini` is -32602 "unknown model") and `TestModelChoiceFromListing` fails. *(Deviation 4: `/model gpt-5-nano` answers "Unknown model".)*
2. `models` returns `sel.Model` as the default when live: `TestModelChoiceFromListing` fails.
3. `get` lists on every call (no `once`, no kept result): `TestModelChoiceListsOnce` fails.
4. `get` never retries: `TestModelListingRetries` fails.
5. `get` retries on every call, without `listRetry`: `TestModelListingRetries` fails.
6. The warning logs `"err", r.Err`: `TestModelChoiceFallsBack` fails, `sk-pl` in the log.
7. `startModelListing` lists without `Choose`'s credential check: `TestNoCredentialNoListing` fails.
8. `TestMain` without `listModels = offlineList`: `TestListerIsOffline` fails. (This copy's run sends `k` to OpenAI's model list once; no other negative reaches the network.)

**P2's verification:**

```text
go test -count=1 ./internal/cli -run 'TestModel|TestListerIsOffline|TestNoCredentialNoListing|TestDefaultProviderSetting' -> ok
go test -count=1 -race ./internal/cli -run 'TestModel'                       -> ok (on Windows; the gates run -race everywhere)
git diff --exit-code HEAD -- acpserver                                       -> no diff
```

### P2's deviations, 2026-10-13

- **Deviation 4. Step 5's `TestModelOptionOffersListedModels` cannot import the SDK in `internal/cli`.**
  - **Found:** the planned test drives the agent over raw ACP through `acptest.Connect`, so `internal/cli`'s tests import `acp-go-sdk2`, and `go test ./...` fails `internal/archtest`'s `TestImportRules/rule_2_module`: "rule 2: github.com/maccavelli/gobble-cli/internal/cli imports github.com/maccavelli/acp-go-sdk2" (`internal/archtest/check.go:204-219`; only `acpserver` and `acpclient` may import it, tests included). The other six P2 tests passed, and 10 times under `-race`.
  - **Decided** ("Through gobble's acpclient"): the test runs the real in-process agent, `e.startAgent` (so `agentServe` and `e.agentOptions`), with the scripted model. `/model gpt-5-nano` answers `Model: gpt-5-nano`; the `config_option_update` it sends, whose params `acpclient` passes through, offers the eight listed models with `gpt-5-nano` current; the next request names `gpt-5-nano`. `set_config_option` reaches the same `setConfig`, which `acpserver`'s `TestConfigOptionsPersist` covers. C5 holds. Step 5 and negative 1 are annotated. No file added.
- **Deviation 5. Step 1's warning fails sloglint.**
  - **Found:** P2's gates: `golangci-lint` on linux, darwin and windows, and so Windows preflight, report "internal/cli/models.go:92:75: key-value pairs should not be used (sloglint)" (`.golangci.yml`, `sloglint`). Every test gate passed, 31 of 31 on each system with no race report. The repository logs with typed attributes (`internal/cli/line.go:183`).
  - **Decided** ("Typed attrs, house style"): `e.log().WarnContext(e.ctx, "model listing failed; offering the curated static list", slog.String("provider", sel.ID), slog.String("kind", provider.FailureKind(r.Err)))`. The output is the same, so `TestModelChoiceFallsBack` and negative 6 stand; P2's gates are rerun. Step 1 is annotated. No file added.

### P3 — The records (D8; closes F11)

Each note is dated the day P3 runs, cites 0020-MADR's decision and the commit of the phase that carried it, and is added beside the text it corrects, which stays.

1. **`docs/decisions/0019-MADR-sibling-libraries-current-pins-and-integration.md`, D2:** "go-llmprovider-sdk `v1.11.0` is pinned by 0020-MADR D1 (0020-PLAN P1, `<P1's commit>`)". *(P1 deviation 3: `v1.12.0`.)*
2. **`docs/decisions/0005-PLAN-v1-feature-scope.md`:** at F4-2's `acpserver` bullet ("the `model` option lists the catalog's models for the session's provider"): "the option's models are the SDK's listing from 0020-MADR (0020-PLAN P2, `<P2's commit>`); F4-2 adds `models.json`'s models and the metadata on top". At X3's entry: "`--offline` also turns off 0020-MADR's model listing, which then offers the static models".
3. **`docs/reports/0014-REPORT-llmprovider-sdk-integration.md`:** at the lines saying configuring and `gobble models` "go through `wizard` and `catalog` via `llm/provider`": "the ACP model option lists through `catalog` from 0020-MADR".
4. **`docs/README.md`:** the 0020 rows read `accepted` and `completed (<date>)`; the 0019-MADR, 0005-PLAN and 0014-REPORT rows note their 0020 notes; the build-order list gains `**0020-PLAN.** Run: P1–P3 on <date>.`
5. **This plan:** `status: completed`, and its `## Execution record (<date>)`: each phase's commit, gates and negatives, and what this plan predicted wrongly.

**P3's verification:**

```text
make check-records && make markdownlint                                     -> ok, 0 issues
python -I <scratch>/q261_lint_records.py --negative <every record P3 touches> -> only the planted violation
python -I <scratch>/q463_lint_vs_head.py docs/decisions/0005-PLAN-v1-feature-scope.md -> added 0
grep -c '0020-MADR' <each record of steps 1-3>                              -> at least 1 each
```

## Verification (whole plan)

After P3:

```text
go list -m github.com/maccavelli/go-llmprovider-sdk                         -> v1.11.0   (deviation 3: v1.12.0)
python scripts/check_libs.py                                                 -> every library at its latest tag; exit 0
git diff --exit-code 17cd497 -- acpserver                                    -> no diff
the gates                                                                    -> lint ×3, tests on three systems, -race on two, preflight
```

A live check, by hand, with the owner's test key in the child's environment only: `gobble acp` driven by a scratch client, as 0019-PLAN P4 step 12 drove it; `session/new`'s `model` option offers OpenAI's live list, more than six, `gpt-6-luna` first.

### Acceptance criteria (mapped to MADR Confirmation)

| # | Criterion | MADR |
| :--- | :--- | :--- |
| A1 | The pin and its sums; no `WithLimit` | D1 |
| A2 | `List` gives the full canned list with the capable profile and metadata on, sends one listing request, and falls back to the static models with a kind | D2, F5–F7 |
| A3 | The ACP option, `set_config_option`, `/model` and the next request take a listed model beyond the six | D5, F3 |
| A4 | A new session starts on the listing's first model when live, on `StaticModels[0]` when not | D4 |
| A5 | One listing per process; a failure retried at most once a minute | D3, F8 |
| A6 | The warning names the provider and the kind, never the text | D6, F5 |
| A7 | No test lists over the network | D7, F10 |
| A8 | `acpserver` unchanged | D5 |
| A9 | The records carry D8's notes | D8, F11 |

**A6 is the criterion most likely to be dropped.** The test that proves it reads two outputs, and it is the one that looks redundant; P2's negative 6 is what shows it can fail.

## Rollout and Rollback

- **Rollout:** one commit per phase, in order, after 0019-PLAN P5's commit.
- **Rollback:** P2 reverts with `git revert` of its commit, and the option is the static six again; P1 reverts the pin and `List` with no user-visible change, since nothing calls `List` before P2.

## Deferred (named, so they are not mistaken for oversights)

- **A persistent catalog, `gobble models list` and per-model metadata:** 0005-PLAN F4-2, on this listing.
- **`--offline`:** 0005-PLAN X3; P3 notes that it turns the listing off.
- **A shorter `/model` listing:** the owner chose every model; a search over `Usable` waits for a caller.
- **Request-side prompt caching** *(P1 deviation 3)*: go-llmprovider-sdk `v1.12.0`'s `Request.Cache`. gobble's 0005-MADR plans prompt-cache warming and its 0012-MADR D13 asked the SDK for `cache_control`; whether and how gobble sets it, and its cost, is its own record.
- **The SDK's own debug log of a listing:** `List` passes no logger (C2); if the SDK's logging is shown to redact, a later record may pass one.

## Execution record (2026-10-13)

Every phase ran, in order, after 0019-PLAN P5 (`1d52ed1`). The agent staged each; the owner committed each with `git commit --no-edit`.

| Phase | Commit | Landed |
| :--- | :--- | :--- |
| The records | `a0dd0e7` | this plan and its MADR, approved |
| P1 | `1625013` | go-llmprovider-sdk `v1.12.0` (D1, deviation 3) and `provider.List` (D2) |
| P2 | `a45dad6` | the model choice from the listing (D3–D7) |
| P3 | the commit of these records | D8's notes and this record |

**Gates.** Each phase's run of `q428_gates_0019.py`: lint for linux, darwin and windows, 0 issues; Windows `go test` and `-race`, the second Linux machine's `-race` and the macOS laptop's `go test`, 31 packages ok each, with no failure and no race report; Windows preflight passed. P2's first run failed lint on all three and preflight on sloglint (deviation 5); its rerun is the green one. Each staged index passed `q252_check_index.py`.

**Negatives,** each on a scratch copy after a control run passed: P1 5 of 5, each failing only its own test once deviations 1 and 2 were in; P2 8 of 8 (7 of 8 at first, below). P1's five list tests also passed 250 of 250 runs under `-count=50`, and P2's model tests 10 times under `-race`.

**The live check:** `gobble acp`, built from P2's tree, with the owner's test key in the child's environment only and a scratch client (`i1live`): `session/new`'s `model` option offered 73 OpenAI models, `gpt-6-luna` current, in 2.0 s, the listing included; the key was in no output and no file. **The whole plan:** `v1.12.0`; `check_libs.py` every library at its latest tag, exit 0; `acpserver` unchanged since `17cd497`.

**Issue 1** was closed on 2026-10-13, at the owner's request, after P2's commit.

**Deviations:** 5, each the owner's decision, recorded under its phase: 1–3 (P1), 4–5 (P2).

**What this plan predicted wrongly:**

1. That the metadata request follows the listing's: the SDK fetches it concurrently and waits for it only when the listing answers (deviation 1).
2. That the clock makes a unique URL: on Windows it did not move across 10 000 calls (deviation 2).
3. That `v1.11.0` stays the latest tag until P1: `v1.12.0` was tagged the same afternoon (deviation 3), the second SDK tag in a day to arrive mid-plan, after 0019-PLAN's deviation 13.
4. That a test in `internal/cli` may import the SDK: archtest rule 2 forbids it, tests included (deviation 4).
5. The warning's logging style: the repository's sloglint refuses key-value pairs (deviation 5).

**The agent's own error, found and fixed before staging:** P2's `TestModelListingRetries` first checked the fetch count straight after `get()`, before a retry's goroutine could run, so negative 5 (a retry on every read) passed it. The test now waits 200 ms for a retry that should not start; it then caught negative 5, and passed 20 of 20 runs.

**Left open:** `Request.Cache` (Deferred); `--offline` turning the listing off (0005-PLAN X3); a persistent catalog and `gobble models list` (0005-PLAN F4-2).
