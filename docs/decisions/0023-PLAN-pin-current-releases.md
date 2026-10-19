---
status: in-progress
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
