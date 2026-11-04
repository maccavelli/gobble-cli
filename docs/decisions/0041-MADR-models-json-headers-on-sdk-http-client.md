---
status: accepted
date: 2026-11-03
decision-makers: repository owner
consulted: none
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Send `models.json` headers over the SDK's own HTTP client, so a provider with headers keeps its bounded transport

## Context and Problem Statement

`models.json` can add headers to a provider's requests (`docs/guides/models-json.md`, "headers"; 0005-PLAN F4-2b,
deviation 1). gobble adds them in `llm/provider`. `Options.Headers` (`llm/provider/provider.go:63`) is a function that
resolves each request's headers for its model. `sdkOptions` (`:109`) wraps the provider's client in `headerTransport`
(`:126`), through `withHeaders` (`:150`), and hands the result to the SDK with `llmprovider.WithHTTPClient`.

`withHeaders` starts from `http.DefaultTransport` when it is given no client:

```go
func withHeaders(c *http.Client, headers func(ctx context.Context, model string) (map[string]string, error)) *http.Client {
	out := &http.Client{}
	base := http.DefaultTransport
	if c != nil {
		*out = *c
		if c.Transport != nil {
			base = c.Transport
		}
	}
	out.Transport = headerTransport{base: base, headers: headers}
	return out
}
```

Production never gives it a client: `Options.HTTPClient` is documented "for tests" (`:51`), and no production caller
sets it. So every provider whose `models.json` entry has `headers` sends its requests over `net/http`'s default
transport, not over the client the SDK builds for every other provider. Every SDK call of such a provider is affected,
because all three paths that build one go through `sdkOptions`:

* `New` (`provider.go:188`), for agent turns and session titles;
* `List` (`llm/provider/list.go:73`), for `gobble models` and the model choice;
* `SignIn` (`llm/provider/signin.go:175`), for `gobble login`'s check listing.

`internal/cli/modelsjson.go:121` sets `Headers` only when `models.json` gives the provider some (`headersFor`, `:98`).
A provider with none passes no client, and the SDK uses its own.

[0037-MADR-hardening-integrity-and-performance.md](0037-MADR-hardening-integrity-and-performance.md) found this
("What belongs elsewhere") and filed it upstream as
[go-llmprovider-sdk#10](https://github.com/maccavelli/go-llmprovider-sdk/issues/10). Its D35 says gobble adopts the
answer "when a tag carries it, through a record of its own". go-llmprovider-sdk `v1.25.0` carries it. gobble has
pinned that tag since `0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md` P25 (`2b1b262`), but has
not used it. [gobble-cli#10](https://github.com/maccavelli/gobble-cli/issues/10) asks for it. This is that record.

### What `http.DefaultTransport` loses

| | The SDK's client (`llmprovider.NewHTTPClient`) | `http.DefaultTransport` |
| :--- | :--- | :--- |
| Connect | 30 s | 30 s |
| TLS handshake | 10 s | 10 s |
| First byte of a reply (`ResponseHeaderTimeout`) | 300 s | **none** |
| Idle connections kept per host | 16 | **2** |
| Connections shared with the process's other SDK providers | yes | **no** |
| Proxy from `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY` | yes | yes |
| Total timeout | none, so a stream runs while it sends | none |

The row that matters is the first byte. A gateway that accepts the connection and never answers holds the turn until
its context ends. For an agent turn, that is when the user cancels it.

### What go-llmprovider-sdk `v1.25.0` provides

From `llmprovider/options.go` at `v1.25.0`:

* **`NewHTTPClient() *http.Client`** "returns the client a provider builds when it is given none: a new http.Client on
  each call, over the transport the process shares, so that every client shares its connections." Its bounds are the
  table's. "Its Transport is an http.RoundTripper to wrap, not an *http.Transport to configure: wrap it to add request
  headers, and give the client to WithHTTPClient."
* **`WithHTTPClient`** documents the same pattern: "To add request headers, wrap NewHTTPClient's Transport in an
  http.RoundTripper that sets them, and give that client; what the transport needs to know per request, such as the
  request's model, goes in the request's context."
* **`ExampleNewHTTPClient`** (`llmprovider/example_test.go:106`) shows it end to end.
* The SDK's decisions: its `0061-MADR-consumer-gaps-from-gobble-issue-9.md` D6 declined a per-request header option in
  favour of this pattern, and its `0066-MADR-wizard-v1-24-debugging-pass-findings.md` D2 exported the client.

gobble's `headerTransport` already is the wrapper the SDK describes. Its model also already travels in the request's
context, under `modelKey`, which `Stream` sets. Only the base it wraps is wrong.

## Decision Drivers

* A provider with `models.json` headers has the same transport, and so the same time limits, as one without.
* The SDK owns its transport's bounds (0004-MADR: wait for the library, do not copy it). gobble names no timeout.
* The SDK's documented pattern, not a second one of gobble's.
* `withHeaders` keeps meaning "the given client, plus headers", as `net/http` reads a client.
* The fix is shown by a test that fails without it.

## Considered Options

* **A. `withHeaders` starts from `llmprovider.NewHTTPClient()` when given no client.** A caller's own client is
  wrapped as it is, a nil `Transport` keeping `net/http`'s meaning.
* **B. As A, and a caller's client with a nil `Transport` also gets the SDK's transport.** This is the snippet
  gobble-cli#10 proposed.
* **C. gobble builds its own bounded `http.Transport`,** with the SDK's numbers.
* **D. An `http.Client.Timeout` on the header client.**
* **E. Leave it.**

## Decision Outcome

Chosen option: **"A. `withHeaders` starts from `llmprovider.NewHTTPClient()` when given no client"**, because it is the
SDK's documented pattern, it gives the production path the SDK's own transport and nothing else, and it changes
nothing for a client a caller chose. *Accepted 2026-11-03, by the owner's "committed, proceed to P1" after the records
commit `f71cc4a`.*

* **D1. The default client is the SDK's.** `withHeaders(nil, h)` wraps `llmprovider.NewHTTPClient()`'s `Transport` in
  `headerTransport`. Every provider built with `models.json` headers then sends over the transport the SDK shares with
  every other provider, whose limits are the SDK's. That covers all that `WithHTTPClient` covers: "its requests, its
  listings and a session's refresh". The SDK still sees a given client (`Settings.HTTPClientGiven`), as it does today,
  so it shares the client with an OAuth session exactly as before.
* **D2. A caller's client keeps its own transport.** `withHeaders(c, h)` copies `*c`, so `Timeout`, `Jar` and
  `CheckRedirect` carry over and `c` is not changed. It wraps `c.Transport`, or `http.DefaultTransport` when that is
  nil, since that is what a nil `Transport` means to `net/http` and what the SDK does with such a client. Only tests
  pass a client today (`Options.HTTPClient`, "for tests"), and each sets its own `Transport`.
* **D3. One shape for both.** `withHeaders` takes the client to wrap, `c` or `NewHTTPClient()`, and copies it the same
  way. The code reads as the SDK's example does:

  ```go
  func withHeaders(c *http.Client, headers func(ctx context.Context, model string) (map[string]string, error)) *http.Client {
  	if c == nil {
  		c = llmprovider.NewHTTPClient()
  	}
  	out := *c
  	base := out.Transport
  	if base == nil {
  		base = http.DefaultTransport
  	}
  	out.Transport = headerTransport{base: base, headers: headers}
  	return &out
  }
  ```

  `headerTransport`, `modelKey`, `requestModel` and `sdkOptions` are unchanged. So are the headers' resolution, the
  refusal of an unresolvable header as `llmprovider.ErrInvalidRequest`, and the closing of the unsent body.
* **D4. The test.** A new test in `llm/provider` checks `withHeaders` on three clients:
  * no client: the wrapped base is `llmprovider.NewHTTPClient().Transport`;
  * a client with its own transport: the base is that transport, the client's other fields are copied, and the
    caller's client is unchanged;
  * a client with a nil transport: the base is `http.DefaultTransport`.

  The first case is the fix. It is seen failing with `base := http.DefaultTransport` put back. The others pin D2, each
  seen failing on a break of its own. `TestWireHeaders` stays as it is: it covers the headers reaching the request and
  an unresolvable header ending it unsent.

  The first case compares the two `RoundTripper` values with `==`. That holds because the SDK returns "a new
  http.Client on each call, over the transport the process shares". Today that value is a comparable struct around
  one pointer. Should a later SDK change it to an uncomparable type, the comparison panics in the test, a loud
  failure, and this record is amended. The test does not read the SDK's bounds through its internal `Base()` method:
  they are the SDK's to test, and gobble does not depend on that method.
* **D5. The docs.**
  * `Options.Headers`' doc says the headers ride on the SDK's own client, or on `HTTPClient` when that is set.
  * `withHeaders`' doc says which client it copies.
  * `docs/guides/models-json.md`'s `headers` item says a provider with headers keeps the same connection and reply
    time limits as one without, without restating the SDK's numbers.
* **D6. The records.** 0037-MADR's "What belongs elsewhere" item for go-llmprovider-sdk#10, and its D35, each gain a
  dated note that this record adopts the SDK's answer.

### Consequences

* Good, because a provider with `models.json` headers now fails a reply that sends no first byte within the SDK's
  300 s, instead of waiting for the turn's context. A provider without headers already did.
* Good, because a provider with headers shares the SDK's pooled connections with the process's other providers, and
  keeps up to 16 idle connections per host instead of 2.
* Good, because gobble names no timeout of its own; a later SDK tag that changes a bound changes it here too.
* Neutral, because a gateway that takes longer than 300 s to its first byte now fails where it used to wait. That is
  the limit every other provider already has, the SDK's, chosen after the ChatGPT client
  (go-llmprovider-sdk's `0021-MADR`, D2).
* Neutral, because `mcpclient` keeps its own `headerTransport` over `http.DefaultTransport` (`mcpclient/conn.go:181`).
  MCP servers are not SDK providers; see More Information.
* Bad, because D4's first case depends on `NewHTTPClient` returning equal `RoundTripper` values across calls. That
  follows from its doc, not from a promise of `==`; D4 says what happens if it stops holding.

### Confirmation

1. `withHeaders(nil, h)`'s transport is a `headerTransport` whose base equals `llmprovider.NewHTTPClient().Transport`.
   The test of D4 shows it, and fails on a copy with `base := http.DefaultTransport` restored.
2. A caller's client keeps its transport and fields and is not changed, and a nil transport is wrapped as
   `http.DefaultTransport`. D4's other cases show it, each seen failing on a break of its own.
3. No production file of `llm/provider` names `http.DefaultTransport` but `withHeaders`' nil-`Transport` case.
4. `TestWireHeaders`, the rest of `go test ./...`, and `go test -race ./...` pass unchanged. No golden changes.
5. `make preflight` and `make race` pass on macOS, the Linux machine and the Windows laptop.

## Pros and Cons of the Options

### A. `NewHTTPClient()` for no client; a caller's client as `net/http` reads it

* Good, because the production path gets exactly the SDK's client, by the SDK's documented pattern.
* Good, because a client a caller chose means what `net/http` says it means, and what it means to the SDK without
  headers. Adding headers changes nothing else.
* Bad, because a test client with a nil `Transport` would still get no first-byte bound. No such client exists, and a
  test that wants one sets it.

### B. As A, and the SDK's transport for a caller's nil `Transport`

* Good, because no path of `withHeaders` reaches `http.DefaultTransport`.
* Bad, because the same client then means one thing with headers and another without: the SDK sends a nil-`Transport`
  client over `http.DefaultTransport`. Headers would change more than headers.
* Neutral, because no caller passes such a client, so in practice it differs from A only in what it promises.

### C. gobble's own bounded transport

* Good, because gobble would control its bounds.
* Bad, because it copies what the SDK owns (0004-MADR). Its numbers would drift from the SDK's, and its connections
  would not be shared with the other providers.

### D. An `http.Client.Timeout` on the header client

* Bad, because `Client.Timeout` bounds the whole exchange, body included. A streamed reply longer than the timeout
  would be cut off mid-stream. The SDK chose no total timeout for that reason.

### E. Leave it

* Good, because nothing changes.
* Bad, because a provider with headers keeps hanging on a gateway that never answers, the defect 0037-MADR found and
  gobble-cli#10 reports.

## More Information

### Not decided here

* **MCP's header client.** `mcpclient/conn.go:181` also wraps `http.DefaultTransport`, there to send an MCP server's
  configured headers over Streamable HTTP. MCP servers are not SDK providers, and `NewHTTPClient`'s bounds were set
  for LLM endpoints. An MCP server may hold a tool call's reply until the tool ends, which can take longer than 300 s.
  Whether that client takes bounds, and which, is a separate decision, as gobble-cli#10 notes.
* **Retries.** Whether the SDK's retry resends a request that timed out waiting for its first byte is the SDK's policy
  (`llmprovider.WithStreamRetry`), unchanged here and the same as for a provider without headers.

### Relationship to other records and issues

* [gobble-cli#10](https://github.com/maccavelli/gobble-cli/issues/10): this record carries it out. Its step 1, the pin,
  is done (0035-PLAN P25).
* [go-llmprovider-sdk#10](https://github.com/maccavelli/go-llmprovider-sdk/issues/10): closed by the SDK's `v1.25.0`.
* [0037-MADR-hardening-integrity-and-performance.md](0037-MADR-hardening-integrity-and-performance.md): its "What
  belongs elsewhere" item and D35 (D6).
* [0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md](0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md):
  D1, as amended on 2026-11-01, pinned `v1.25.0`.
* [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): `llm/provider` stays the only importer of
  go-llmprovider-sdk (archtest rule 4), and the library's behaviour is used, not copied.
* [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md): its note of
  2026-10-29 records that `headerTransport` stays, by the SDK's 0061-MADR D6; that is unchanged.
