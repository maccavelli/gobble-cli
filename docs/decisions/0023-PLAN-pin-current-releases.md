---
status: complete
date: 2026-10-19
associated-madr: "0023-MADR-pin-current-releases.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement pinning the current releases

Associated MADR: [0023-MADR-pin-current-releases.md](0023-MADR-pin-current-releases.md)

*Approved on the owner's word of 2026-10-19, "Approve 0023", with golangci-lint kept at `v2.14.0` (step 7 is not run), and to run before 0005-PLAN F6-2; it starts after the owner commits the records of that date.*

## Goal

Every direct requirement, tool module and CI action at its current release (MADR D1, D2; D3 if the owner takes it),
with the tests the upgrade needs (D4, D5) and the records and docs that name a version brought up to date (D7).

## Scope

- **In:** `go.mod`, `go.sum`; `.github/workflows/ci.yml`'s `actions/upload-artifact`; `Makefile` and `ci.yml`'s
  golangci-lint only under D3; a test for D4 in `acpserver` or `internal/cli`; `llm/provider/list_test.go`; the records
  and docs of D7.
- **Out:** adopting any new SDK feature (MADR D6); F6-2, whose entry waits for the owner separately.

## Implementation Steps

1. **Measure again,** since tags move: `make check-libs` and `go list -m -u all`. A tag newer than the MADR's is a
   deviation: stop and ask.
2. **D4's test first, seen failing:** at `v1.14.0`, a test of `acpserver.Serve` over an in-memory pipe writes
   `initialize`, then closes the input, and requires the answer; it fails today (no answer before `Done()`).
3. **acp-go-sdk2 `v1.16.0`:** `go get`, `go mod tidy`; D4's test passes; `go test ./acpserver/... ./acpclient/...
   ./internal/cli/...`, and the ACP goldens unchanged.
4. **go-llmprovider-sdk `v1.15.0`:** `go get`, `go mod tidy`; find in the SDK's source which change orders OpenAI's
   curated listing as `v1.15.0` does, and cite it; bring `TestListFullCatalog`'s expected order to it, the test's
   intent (the full catalog, live, from one request with the key) unchanged; `go test ./llm/... ./acpserver/...
   ./internal/cli/...`.
5. **The x and tool modules:** `go get golang.org/x/sys@v0.49.0 golang.org/x/term@v0.47.0 golang.org/x/text@v0.43.0
   golang.org/x/vuln@v1.9.0 honnef.co/go/tools@v0.8.2 golang.org/x/exp@<the MADR's pseudo-version>`, `go mod tidy`;
   the `go.mod` diff is the MADR's, no module added.
6. **CI:** `actions/upload-artifact` to `v7.0.2` by its commit SHA, read from the tag, in both places.
7. **D3, only on the owner's word:** golangci-lint `v2.15.0` in `Makefile` and `ci.yml`, a dated note in 0007-MADR D5,
   and lint run with it on three systems.
8. **Records and docs (D7):** dated notes in 0019-MADR D2 and 0020-MADR D1; `docs/architecture.md`'s dependency table,
   `README.md`'s Stack, `docs/README.md`'s SDK line and index rows; this plan's execution record.

## Verification

- `make check-libs`: every library at its latest tag, nothing to compare.
- D4's test failing at `v1.14.0` on a scratch copy and passing at `v1.16.0`; `TestListFullCatalog` failing with the old
  order at `v1.15.0`.
- The pre-add check on the changed `.go` files; lint for linux, darwin and windows; `go test -count=1 ./...`,
  `CGO_ENABLED=1 go test -race -count=1 ./...` and `make preflight` on macOS, the Linux machine and the Windows laptop,
  over ssh on a copy of the working tree, as 0005-PLAN F6-1's were.
- The architecture check and the link check of 0005-PLAN's "The docs brought up to date", run again.

## Rollout and Rollback

One commit, after the owner's approval. Rollback is reverting it: `go.mod` and `go.sum` return to `30b8a4b`'s pins, and
D4's test with them, since it fails at `v1.14.0`.

## Execution record

**Run on 2026-10-19,** on the owner's word "committed, proceed" after `8b9fac3`, as the steps say, with deviation 1
below. Step 7 was not run: golangci-lint stays at the fleet's `v2.14.0` (the owner's choice).

**Steps:**

1. **Measured again:** `go list -m -u` gave the MADR's tags, every one; `actions/upload-artifact`'s latest release is
   still `v7.0.2`. No deviation.
2. **`TestServeAnswersBeforeEndOfInput`** (`acpserver/serve_test.go`): `Serve` over a reader holding one `initialize`
   request and then end of file, which must answer it before returning. At `v1.14.0` it failed 20 runs of 20, `Serve`
   returning with nothing written: acp-go-sdk2 issue 8.
3. **acp-go-sdk2 `v1.16.0`:** the test passed 20 of 20; `acpserver`, `acpclient` and `internal/cli` passed, with
   `testdata/session.golden` unchanged.
4. **go-llmprovider-sdk `v1.15.0`:** `TestListFullCatalog`'s order is the SDK's: from `v1.14.0`, the models past the
   curated six are newest first by release date, then by id (go-llmprovider-sdk `llmprovider/catalog/menu.go:75-94`,
   `:120-150`, its 0041-MADR D3), where `v1.13.0` kept the uncapped curation's order (`discovery.go:207-226`). The two
   undated models of the test's listing now sort by id. The new order failed at `v1.13.0`. Then deviation 1.
5. **The x and tool modules:** the `go.mod` diff is the MADR's scratch clone's, line for line; no module added.
6. **CI:** `actions/upload-artifact` from `043fb46d…` (`v7.0.1`, as resolved today) to `cf430e030ddbb5b0abf93d22962f4752f3646cd9`
   (`v7.0.2`), at both places.
7. Not run (the owner's choice).
8. **Records and docs:** this record; dated notes in the MADR, 0019-MADR D2 and 0020-MADR D1 and F7;
   `docs/architecture.md`'s dependency table, `README.md`'s Stack, `docs/README.md`'s index rows and SDK line.

**Deviation 1, 2026-10-19, decided by the owner** ("Accept, test it"): with the order corrected, `TestListFullCatalog`
failed its next check, "requests [GET api.openai.com/v1/models, GET models.opencode.ai/api.json] with 2 authorization
headers". `check-libs` had not shown it, the order check failing first.

- **Found:** from `v1.14.0` every provider's listing but Kilo's and Ollama's also fetches the model metadata document
  (`discovery.go:283-296`, `listOnce`), where `v1.13.0` fetched it only for Hugging Face, Together and opencode
  (`discovery.go:265-282`, `:671`). The document's request carries no `Authorization` header: a scratch copy logged
  `GET api.openai.com/v1/models auth="Bearer sk-test"` and `GET models.opencode.ai/api.json auth=""`. It is cached
  in-process for 10 minutes, then revalidated by ETag (`model_metadata.go:30`, `:445-508`). From `v1.14.0` a listing is
  also served from an in-process cache, keyed by provider, base URL and a fingerprint of the credential
  (`cachedOrFetched`, `listing_cache.go`), which gobble's own once-a-process listing (`internal/cli/models.go:83-88`,
  0020-MADR D3) already made invisible to a user.
- **Chosen:** accept the fetch, which the owner's choice of 2026-10-13, "model metadata on, once per process", already
  covers, and test it: `TestListFullCatalog` requires the listing with the key and the metadata document with no
  `Authorization` at all.
- **Not chosen:** `WithoutModelMetadata()` for listings, which would rank without the document's facts; keeping the
  provider SDK at `v1.13.0`.
- **Tests changed with it,** for the SDK's caches, which outlive a test in one process: `TestListFullCatalog` and
  `TestListFetchesMetadata` use a document URL and a key of each run's own, so `-count=3` passes; at `v1.15.0` with
  fixed keys a second run made no listing request. At `v1.13.0` the fixed keys passed `-count=3`.
- No file is added to the step's scope.

**Deviation 2, 2026-10-19, decided by the owner** ("Fix it in 0023"): the Linux machine's race run failed
`TestWireRateLimitResets/claude` (`llm/provider/wire_test.go`) once: "retry-after 1m28.999564465s … want … in [1m29s,
1m30s]".

- **Found:** the test wrote Claude's reset header as `time.Now()+90s` in RFC 3339, which keeps whole seconds, so the
  fraction of a second it drops, with the call's own time, can leave the wait just under its 89 s floor. It is not the
  upgrade's: the subtest alone, 20,000 times under the race detector on macOS, failed 5 times at `v1.13.0` and 4 at
  `v1.15.0`, each `1m28.9998…s`.
- **Chosen:** the header is a whole second 91 s ahead of the second the test starts in,
  `time.Now().Truncate(time.Second).Add(91 * time.Second)`, so the wait is in (90 s less the call's time, 91 s], and
  the check is [89 s, 91 s]. Seen failing on a copy whose header was 87 s ahead; then 20,000 runs under the race
  detector: 0 failures, on macOS at `v1.15.0`.
- **Not chosen:** leaving it to a later record, with the race gate failing about one run in 4,000.
- `llm/provider/wire_test.go` is added to the scope.

**Tests,** new or changed, none loosened: `TestServeAnswersBeforeEndOfInput` (new); `TestListFullCatalog` (the SDK's
order; the document requested without a credential; a key and URL of each run's own); `TestListFetchesMetadata` (a key
of each run's own); `TestWireRateLimitResets` (Claude's header from a whole second, the bounds [89 s, 91 s]).

**Negatives:** `TestServeAnswersBeforeEndOfInput` failed 20 of 20 at `v1.14.0`; `TestListFullCatalog` failed with the
new order at `v1.13.0`, with the old order at `v1.15.0` (`check-libs`), and, on a scratch copy whose `List` passes
`WithoutModelMetadata()`, with "requests [GET api.openai.com/v1/models], want the listing with the key and the
metadata document".

**Gates:**
- the pre-add check on the three changed `.go` files (`acpserver/serve_test.go`, `llm/provider/list_test.go`,
  `llm/provider/wire_test.go`), clean;
- lint for linux, darwin and windows, `0 issues.` each with `CGO_ENABLED=0`;
- macOS: `CGO_ENABLED=1 go test -race -count=1 ./...`, 35 packages ok, and `make preflight`, `preflight passed`;
- the Linux machine and the Windows laptop, over ssh with `-o BatchMode=yes -o ConnectTimeout=5`, on a copy of the
  working tree in a fresh temporary directory: `go test -count=1 ./...` and `CGO_ENABLED=1 go test -race -count=1
  ./...`, 35 packages ok on each, Windows with `TMP` and `TEMP` outside the user's home; `make preflight`, every step
  through `apidiff` passed on each (golangci-lint `0 issues.`), and `check-records` and `markdownlint` after it;
  `verify-build-metadata` needs a commit the copy does not have, and passed on macOS. The first Linux race run, before
  deviation 2, failed only `TestWireRateLimitResets/claude`;
- `make check-libs`: every library at its pinned tag, the latest; it clones the committed repository
  (`scripts/check_libs.py:236-238`), whose `TestListFullCatalog` still held `v1.13.0`'s order, so its test step fails
  until this change is committed.

**Not done here, named:** the SDKs' new features (MADR D6); golangci-lint `v2.15.0` (MADR D3, the owner's choice).
