---
status: in-progress
date: 2026-11-03
associated-madr: "0041-MADR-models-json-headers-on-sdk-http-client.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement sending `models.json` headers over the SDK's own HTTP client, so a provider with headers keeps its bounded transport

Associated MADR: [0041-MADR-models-json-headers-on-sdk-http-client.md](0041-MADR-models-json-headers-on-sdk-http-client.md)

## Goal

`withHeaders` wraps `llmprovider.NewHTTPClient()` when it is given no client. A provider with `models.json` headers
then sends over the SDK's shared, bounded transport, as every other provider does. A test shows it and fails without
it, and the docs and records say so. The plan is complete when the MADR's Confirmation 1 to 5 is met.

## Scope

**Base.** `main` at `04e1ea5`, go-llmprovider-sdk `v1.25.0`, Go 1.27.3.

**In scope:**

* `llm/provider/provider.go`: `withHeaders`, its doc, and the docs of `Options.Headers` and `Options.HTTPClient`;
* `llm/provider/wire_test.go`: one new test;
* `docs/guides/models-json.md`: the `headers` item;
* `docs/decisions/0037-MADR-hardening-integrity-and-performance.md`: two dated notes;
* this pair and `docs/README.md`.

**Out of scope** (the MADR's "Not decided here"):

* `mcpclient`'s header client;
* the SDK's retry policy;
* any timeout of gobble's own;
* `headerTransport`'s resolution of headers, `modelKey`, `sdkOptions`, and `internal/cli/modelsjson.go`'s
  `headersFor`, all unchanged.

No module is added or moved; `go.mod` and `go.sum` do not change.

## Conventions every phase follows

0039-PLAN's C1 to C11 apply, as written there:

* the start state, and line numbers re-read;
* scratch work, and long output to a log;
* G-local: `make pre-add-check`, `make preflight` and `make race`;
* G-hosts: the Linux machine and the Windows laptop;
* deviations stopped and recorded;
* every new test seen failing on a broken copy;
* `git commit --no-edit` by the owner;
* no identifier of a real machine in a committed file.

No phase touches `internal/cli`, so `make perf` and `make probe-conhost` are not run.

## Implementation Steps

### P0 — the records

Files: this pair; `docs/README.md`.

1. `docs/README.md` gains the pair's two rows, after 0040's.
2. `make check-records markdownlint` passes.
3. The identifier scan over the staged records finds nothing.
4. Staged; the owner commits.

### P1 — `withHeaders` over the SDK's client (MADR D1 to D6)

Files: `llm/provider/provider.go`, `llm/provider/wire_test.go`, `docs/guides/models-json.md`,
`docs/decisions/0037-MADR-hardening-integrity-and-performance.md`, this plan, `docs/README.md`.

1. **The facts,** re-read at the phase's start; a difference is a deviation:
   * `withHeaders` (`llm/provider/provider.go:150`) is as the MADR quotes it;
   * its only caller is `sdkOptions` (`:109`), whose callers are `New` (`:188`), `List` (`list.go:73`) and `SignIn`
     (`signin.go:175`);
   * no production file sets `Options.HTTPClient` or `SignInOptions.HTTPClient`;
   * `llmprovider.NewHTTPClient` exists at the pinned tag, with the doc the MADR quotes.
2. **The code** (D1 to D3): `withHeaders` becomes the MADR's D3 listing. `headerTransport`, `requestModel` and
   `sdkOptions` are not touched. `provider.go`'s imports do not change: it already imports `net/http` and
   `llmprovider`.
3. **The docs in code** (D5):
   * `withHeaders`' doc: a copy of c, or of the SDK's own client (`llmprovider.NewHTTPClient`) when c is nil, whose
     transport adds headers; a nil `Transport` in c is `net/http`'s default, as `http.Client` reads it. It cites the
     MADR;
   * `Options.Headers`' doc gains a sentence: the headers ride on the SDK's own client, or on `HTTPClient` when set,
     so a provider with headers keeps the SDK's transport and its limits (0041-MADR D1);
   * `Options.HTTPClient`'s doc keeps "for tests", and adds that `Headers` wraps it.
4. **The test** (D4): `TestWithHeadersClient` in `llm/provider/wire_test.go`, after `TestWireHeaders`. It reads the
   wrapped base through a helper that fails unless the client's transport is a `headerTransport`, and has three
   subtests:
   * `"no client"`: `withHeaders(nil, h)`'s base `==` `llmprovider.NewHTTPClient().Transport`;
   * `"own transport"`: given `c := &http.Client{Transport: rt, Timeout: 7 * time.Second}`, the base is `rt`; the
     result's `Timeout` is 7 s; the result is not `c`; and `c.Transport` is still `rt`;
   * `"nil transport"`: given `&http.Client{}`, the base is `http.DefaultTransport`.

   `h` resolves no headers; the test sends no request. `wire_test.go` gains the import
   `github.com/maccavelli/go-llmprovider-sdk/llmprovider`; it already imports `net/http` and `time`. The test sits
   beside `TestWireHeaders`, which covers the same option end to end. **Seen failing (C9),** each on its own scratch copy, with the
   failure quoted in the record:
   * the old base, `base := http.DefaultTransport` for a nil client, fails `"no client"`;
   * a fresh `&http.Client{}` in place of the copy of `*c` fails `"own transport"` on `Timeout`;
   * `c.Transport = headerTransport{…}; return c` fails `"own transport"` on the caller's client being changed;
   * `llmprovider.NewHTTPClient().Transport` for a nil `c.Transport` (the MADR's option B) fails `"nil transport"`.
5. **Assert by search,** in `llm/provider`'s non-test files:
   * `http.DefaultTransport` appears once, in `withHeaders`' nil-`Transport` case;
   * `NewHTTPClient` appears once, in `withHeaders`.
6. **`go doc ./llm/provider`** is diffed against the base. Only `Options.Headers`' and `Options.HTTPClient`'s field
   docs change; no exported name is added or removed.
7. **The guide** (D5): `docs/guides/models-json.md`'s `headers` item gains a sentence. A provider with headers keeps
   the same connection and reply time limits as a provider without them. No number is restated.
8. **The records** (D6): 0037-MADR's "What belongs elsewhere" item for go-llmprovider-sdk#10, and its D35, each gain a
   dated italic note naming this record. Their text above the note is unchanged.
9. **Nothing else changes:** `TestWireHeaders` passes unchanged; `go test ./...` and `go test -race ./...` pass; after
   the gates, `git status` shows only the phase's files, so no golden changed.
10. G-local, then G-hosts. **Commit.**

### P2 — close-out

Files: this plan, the MADR if a fact changed, `docs/README.md`.

1. A comment for gobble-cli#10, written into the execution record: what changed, with P1's commit, and that the issue
   is closed. Posting it and closing the issue wait for the owner's word.
2. The disclosure guard over the outgoing commits, its output recorded. Nothing is pushed.
3. This plan's status becomes `complete`; the MADR's becomes `accepted` if it was not already.

## Verification

| MADR Confirmation | Phase | Check |
| :--- | :--- | :--- |
| 1 the SDK's client for no client, seen failing | P1 | `TestWithHeadersClient/no_client`, step 4's first break |
| 2 a caller's client as `net/http` reads it, seen failing | P1 | the other two subtests, step 4's other breaks |
| 3 no other `http.DefaultTransport` | P1 | step 5's search |
| 4 nothing else changes | P1 | step 9; step 6's `go doc` diff |
| 5 the gates | P1 | G-local, G-hosts |

## Rollout and Rollback

* **One commit for the code,** P1. Reverting it restores `http.DefaultTransport` under headers. Nothing else depends on
  it.
* **User-visible:** a provider with `models.json` headers now fails a reply whose first byte takes more than the SDK's
  300 s, as a provider without headers already does, instead of waiting until the turn is cancelled. No message or
  option is added.
* **No configuration, data, wire or module change.**

## Execution record

Each phase appends its record here.

**Approved 2026-11-03,** the owner's "committed, proceed to P1".

### P0 record (2026-11-03): the records

The pair and its two `docs/README.md` rows: `make check-records markdownlint` passed, and the identifier scan found
nothing. The owner committed them as `f71cc4a`.

### P1 record (2026-11-03): `withHeaders` over the SDK's client

The base is `f71cc4a`. No deviation.

1. **The facts** (step 1), re-read at the base, each as the plan states (the line numbers are the base's; P1's doc
   lines move `provider.go`'s down by 3 to 6):
   * `withHeaders` (`llm/provider/provider.go:150`) was as the MADR quotes it;
   * its only caller is `sdkOptions` (`:113`), whose callers are `New` (`:188`), `List` (`list.go:73`) and `SignIn`
     (`signin.go:175`);
   * no production file sets `Options.HTTPClient` or `SignInOptions.HTTPClient`. `signin.go:182` passes `o.HTTPClient`
     on, and `login.go:112` passes `LoginOptions.HTTPClient`, which `internal/cli/auth.go:302` leaves nil;
   * `llmprovider.NewHTTPClient` is at the pinned tag's `llmprovider/options.go:69`.
2. **The code** (step 2): `withHeaders` is the MADR's D3 listing, now at `provider.go:156`. `headerTransport`,
   `requestModel`, `sdkOptions` and the imports are unchanged.
3. **The docs in code** (step 3): `withHeaders`' doc names the SDK's own client for a nil `c` and `net/http`'s default
   for a nil `Transport`, citing 0041-MADR D1 and D2. `Options.Headers` says the headers ride on the SDK's own client,
   or on `HTTPClient` when set (0041-MADR D1). `Options.HTTPClient` adds "Headers, when set, wraps it." The citations
   use the number, as the package's other comments do.
4. **The test** (step 4): `TestWithHeadersClient` (`llm/provider/wire_test.go`, after `TestWireHeaders`), with its
   subtests `no_client`, `own_transport` and `nil_transport`. `wire_test.go` imports `llmprovider`, in the SDK's group,
   as `provider_test.go` does. `go test -count=1 -run 'TestWithHeadersClient|TestWireHeaders' -v ./llm/provider/`:
   both pass.

   **Seen failing (C9),** each break on a scratch copy of the tree (`git ls-files`, copied outside it), the copy
   restored and passing after the last. The quoted lines are `go test`'s:
   * the old code, `http.DefaultTransport` for a nil client: `--- FAIL: TestWithHeadersClient/no_client`,
     `wire_test.go:377: base *http.Transport, want the SDK's shared transport, the same transport.scoped value`;
   * a fresh `&http.Client{…}` in place of the copy of `*c`: `--- FAIL: TestWithHeadersClient/own_transport`,
     `wire_test.go:388: Timeout 0s, want the caller's 7s`;
   * the caller's client changed in place: `--- FAIL: TestWithHeadersClient/own_transport`,
     `wire_test.go:391: the caller's client was changed: transport provider.headerTransport`;
   * the MADR's option B, the SDK's transport for a nil `Transport`: `--- FAIL: TestWithHeadersClient/nil_transport`,
     `wire_test.go:396: base transport.scoped, want http.DefaultTransport`.

   The first run's message printed the whole `*http.Transport`; it now names the two types. The four breaks were run
   again after that change, with the same results.
5. **Assert by search** (step 5), in `llm/provider`'s non-test files: `NewHTTPClient` once (`provider.go:158`), and
   `http.DefaultTransport` once (`:163`, the nil-`Transport` case).
6. **`go doc -all ./llm/provider`** (step 6), diffed against a `git archive` of the base: two hunks, `Options.HTTPClient`'s
   and `Options.Headers`' field docs. No exported name was added or removed.
7. **The guide** (step 7): `docs/guides/models-json.md`'s `headers` item adds "A provider with headers keeps the same
   connection and reply time limits as one without."
8. **The records** (step 8): 0037-MADR's "What belongs elsewhere" item for go-llmprovider-sdk#10 and its D35 each
   have a dated italic note naming this record; D35's also names 0040-MADR for acp-go-sdk2#13.
9. **Nothing else changes** (step 9): `go test -count=1 ./llm/provider/ ./internal/cli/...` passed. After the gates,
   `git status` showed only the phase's files, so no golden changed.
10. **G-local:** `make pre-add-check FILES="llm/provider/provider.go llm/provider/wire_test.go"`: "2 file(s) clean
    (gofmt, golint, govulncheck)"; `make preflight`: passed; `make race`: 38 ok.
11. **G-hosts,** on the Linux machine and the Windows laptop, with 826 files shipped to each: `go test` and race passed
    on both, 38 ok, the new test among them; `make check-records markdownlint` passed on both; preflight stops at
    `verify-build-metadata`, as on every copy.
