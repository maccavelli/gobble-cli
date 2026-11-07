---
status: in-progress
date: 2026-11-06
associated-madr: "0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement adopting what go-llmprovider-sdk v1.23.0 and v1.24.0 added

Associated MADR: [0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md](0036-MADR-adopt-go-llmprovider-sdk-new-capabilities.md)

## Goal

- `llm/provider` builds the SDK's registry once, and a run's providers include the Chat Completions servers
  models.json names.
- The SDK's warnings are logged, and learned reasoning refusals are kept across runs.
- A live listing is searched by its facts.
- Images reach the model from the ACP prompt, `read`, MCP and `@path`, as 0005-MADR and 0008-MADR D19 specify.
- [gobble-cli#4](https://github.com/maccavelli/gobble-cli/issues/4)'s items 6 and 7 are done. Its other items are
  carried as the MADR's reconciliation table says.

## Scope

- **In:**
  - `llm` (`event.go`, `llm.go`, `doc.go`);
  - `llm/provider` (`provider.go`, `ambient.go`, `list.go`, `model.go`, `signin.go`, a new file for the run's set);
  - `llm/catalog` (`modelsjson.go`);
  - `agent` (warnings, tool-result images);
  - `tool` (`Result`, `Env`), `tool/builtin` (`read.go`, `sniff.go`, a new image preparation file);
  - `mcpclient/tool.go`;
  - `acpserver` (`agent.go`, `mentions.go`, `history.go`);
  - `acpclient` (an image `@path`);
  - `internal/cli` (the set's construction, `agent.go`, `models.go`, `modelscmd.go`, `choice.go`, `login.go`, the
    refusals file, `images.autoResize`);
  - `internal/config` (the key);
  - their tests;
  - `go.mod` and `go.sum` (`golang.org/x/image`, in P7 only);
  - records and docs: this pair; dated notes in 0005-MADR, 0008-MADR, 0018-MADR and 0031-MADR; `docs/architecture.md`;
    `docs/guides/models-json.md`; `docs/guides/choosing-a-model.md`; `README.md`; `docs/README.md`.
- **Out:** everything in MADR D9, and so everything `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md`
  decides.

## Order

- **Before P1:** gobble requires go-llmprovider-sdk `v1.24.1` with 0035's D3 row and D4 changes (MADR D1). P0 checks
  it. *(2026-11-06: `v1.24.1` or later; met at `v1.26.1`, see P0's record of that date.)*
- **The phases:**
  - P1 first;
  - P2 before P3, and before P6, whose dropped-image warning it carries;
  - P4 after P1;
  - P5 after P1;
  - P6 before P7.
  - P4 and P5 do not depend on each other, nor on P2 or P3.
- **Questions:** this plan assumes the MADR's recommended answers. Each answer that differs amends this plan before
  the phase it changes:
  - Q1 "no" drops P3;
  - Q2 "ambient too" changes P4's step on the choice;
  - Q3 moves P7, or P6 and P7, to a later record;
  - Q4 "registry first" moves P1 ahead of P0.

  *(2026-10-29: the owner chose the recommended answer to each of Q1–Q4. The MADR records them, under "The owner's
  answers". So no step above changes: P3 stays, P4 keeps its servers to being named, P6 and P7 stay, and P0 comes
  first.)*
- **The index:** `docs/README.md` holds the other session's uncommitted 0035 row. The owner chose on 2026-10-29 that
  this pair's index rows wait until that row is committed, so this session stages none of that file's changes.
  *(2026-11-06: that row is committed, and this pair's rows are in `docs/README.md`.)*
- **Status:** this plan stays `proposed` until the owner approves execution. Q1–Q4 are answered (above).
  *(2026-10-29: approved by the owner's "committed, proceed"; `in-progress`, waiting at P0. See P0's record.)*

## Implementation Steps

Every test below is first seen failing on the code as it stands, or on a stub, and its failure is quoted in the
execution record. Every planted break runs on a scratch copy, never in the tree. No test reaches a real service: each
serves its own replies, as `llm/provider/wire_test.go` and `wire_sse_test.go` do, or uses the SDK's `llmtest.Fake`,
which `provider_test.go` already imports.

### P0 — the precondition (MADR D1)

*(Amended 2026-11-06, to the tree as it is: 0035's go-llmprovider-sdk phases have landed and the pin has moved past
`v1.24.1`, so step 1 reads the floor, not an exact tag, and steps 4 and 5 are added. The MADR's D1 note of that date
says why.)*

1. `go list -m github.com/maccavelli/go-llmprovider-sdk` prints ~~`v1.24.1`~~ `v1.24.1` or later: the tag
   `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` D1 names last.
2. `go test -run TestWireOffline ./llm/provider/` passes.
3. `git grep -n sdkEmptyAnswer` and `git grep -n 'func offline'` find nothing in `llm/provider`, as 0035's D4 has it.
4. **The SDK's surface,** read at the pinned tag in the module cache or a clone: each name in the MADR's table "What
   the SDK added that gobble does not use" is still exported, `Response.Warnings` and its three kinds,
   `ReasoningRefusal` and `WithReasoningRefusals`, `chatcompat.New`, `WithID`, `Descriptor` and `Discover`,
   `Catalog.Search` and `Match.Why`, `ImageItem` and `Capabilities.Images`. And `git grep` over gobble's `*.go` finds
   none of them used, other than the package function `catalog.Search` the MADR names as today's search.
5. **The citations.** The MADR's and this plan's line numbers in gobble were measured at `ccd8aa8`. Each phase reads
   its own again at its start, and records the lines it found. A line that has moved is not a deviation; a fact that no
   longer holds is.
6. If any of steps 1 to 4 fails, stop: the MADR is amended (D1) before P1. Nothing is written in P0 but its record.

### P1 — one registry, and models.json's table from it (D2)

1. **Tests first:**
   - `llm/provider`, `TestRegistryBuiltOnce`: `testing.AllocsPerRun` of `Label("together")` and of
     `CredentialVar("openai")` is at most 2. Today it is about 85, from the registry each call builds.
   - `llm/provider`, `TestSetHoldsBuiltIns`: a nil set and the built-ins' set give the same ids, labels and methods.
   - `llm/catalog`, `TestModelsJSONTakesTheCallersIDs`: `ParseModelsJSON` given an id set that includes `newprov`
     keeps `"newprov": {"baseUrl": …}` with no `api`, and still checks Pi's wire names for the built-ins the table
     names. Today the hard-coded table skips `newprov`.
2. **The code:**
   - `llm/provider`:
     - `builtins = sync.OnceValue(providers.Default)`, read by every function in `ambient.go` and by `signin.go`;
     - the run's set type, with `IDs`, `Descriptor`, and `register` for P4. A nil set is `builtins()`.
     - `Options` and `SignInOptions` carry the set.
   - `llm/catalog`: `ParseModelsJSON` takes the SDK ids from its caller. `providerAPIs` keeps Pi's wire names only.
   - `internal/cli`: the set built once per run, beside `e.modelsJSON()`, and passed down.
3. **Planted breaks:**
   - `Label` back on `providers.Default()`: `TestRegistryBuiltOnce` fails;
   - the parser keyed by the old table: `TestModelsJSONTakesTheCallersIDs` fails.
4. **Records:** `docs/architecture.md`'s `llm/provider` and `llm/catalog` rows. Gates. Commit.

### P2 — the SDK's warnings (D3)

*(2026-11-04, from `0012-MADR-tool-schemas-descriptions-and-loading.md` D21, proposed: once gobble defers tools natively, `WarningDeferralEmulated` is expected on every request on most wires. If `0012-PLAN-native-tool-deferral.md` P4 has landed first, this phase logs that kind at Debug, once per session and model, and its other kinds as written. If this phase lands first, that plan's P4 adds it.)*

1. **Tests first:**
   - `llm/provider`, `TestWarningsYielded`: `Wrap` of an `llmtest.Fake` replying with `Warnings` of
     `effort_degraded` and an unknown kind yields two `llm.Warning` events, each with its kind, model and detail,
     before `Usage` and `Done`.
   - `agent`, `TestWarningLoggedOnce`: two turns that each warn `effort_degraded` for one model give one Warn record.
     A different model, or a different kind, gives another.
2. **The code:**
   - `llm.Warning` in the sealed set, and the kinds gobble names (`WarningEffortDegraded`, `WarningReasoningLearned`,
     `WarningImageDropped`) as `llm` constants;
   - the adapter's `EventDone` case yields them;
   - the agent logs them, deduplicated per session;
   - `llm/doc.go`'s mapping table gains the row;
   - each switch over `llm.Event` in gobble gains the case or ignores it.
3. **Planted breaks:**
   - the yield removed: `TestWarningsYielded` fails;
   - the deduplication removed: `TestWarningLoggedOnce` fails.
4. **Records:** `llm/doc.go`. Gates, with `apidiff` reporting `llm.Warning` as compatible. Commit.

### P3 — learned refusals across runs (D4; dropped if Q1 is no)

1. **Tests first:**
   - `internal/cli`, `TestRefusalsSaved`: a turn whose response warns `reasoning_learned` with `Refused`
     `{Model: m, Effort: high}` leaves that refusal in the state directory's file, once, however many turns repeat it.
   - `llm/provider`, `TestRefusalsPassedBack`: on a canned wire, `New` with a saved refusal for `openai`'s model `m`
     at `high` sends `m` a lower effort, never `high`, on its first request.
   - `TestRefusalsFileUnreadable`: a corrupt file is ignored with one warning, and the provider is built.
2. **The code:**
   - `Options.Refusals`, passed to `WithReasoningRefusals`;
   - the file in the state directory beside `recent-models.json`, written atomically, deduplicated, capped per
     provider;
   - P2's warning handler saves `reasoning_learned` refusals.
3. **Planted breaks:**
   - the option not passed: `TestRefusalsPassedBack` sees `high`;
   - no deduplication: `TestRefusalsSaved` sees two.
4. **Records:** `docs/architecture.md`, the state directory's files. Gates. Commit.

### P4 — Chat Completions servers from models.json (D5)

1. **Tests first:**
   - `llm/catalog`, `TestModelsJSONChatServer`:
     - `"local": {"baseUrl": "http://127.0.0.1:1/v1"}` is kept as a Chat Completions server, with `name` as its
       label;
     - `api: "openai-completions"` is kept;
     - `api: "anthropic-messages"` on an unknown key is skipped with the new reason;
     - a key with no `baseUrl` is skipped as today.
   - `llm/provider`, `TestChatServerStreams`: against an `httptest` server speaking Chat Completions over Server-Sent
     Events, a set holding `local` builds it with `New("local", …)`, and the text streams.
     - With no key the request has no `Authorization` header.
     - With models.json's `apiKey` resolved, it is `Bearer <key>`.
   - `TestChatServerLists`: `List("local", …)` returns the server's `/v1/models` ids, with models.json's models among
     the known ones.
   - `TestChatServerNotAmbient`:
     - `ChooseWith` with no `defaultProvider` never picks `local`;
     - with `defaultProvider: local` it does;
     - `--model local/m` resolves.
   - `internal/cli`: the picker lists `local` under its label, and `gobble models --provider local` lists its models.
2. **The code:**
   - `llm/catalog`: the kept servers, and `name` used;
   - `llm/provider`: the set registers each with `chatcompat.Descriptor` and a factory of `chatcompat.New(WithID,
     WithBaseURL)`;
   - `New`, `List` and `ModelInfo` resolve the id through the set;
   - `ChooseWith` and `KeyProviders` keep to key-taking built-ins, and an id that is named resolves through the set;
   - `internal/cli`: each server's credential, as for any id: `auth.json`, then models.json's `apiKey`.
3. **Planted breaks:**
   - the server not registered: `TestChatServerStreams` fails with an unknown provider;
   - ambient picking allowed: `TestChatServerNotAmbient` fails;
   - the bearer header dropped: `TestChatServerStreams` fails.
4. **Records:** `docs/guides/models-json.md`, where its `openrouter` example becomes a working server and the skip
   text is reworded; a dated note in 0005-MADR's unnamed-endpoints row. Gates. Commit.

### P5 — the facts search (D6)

1. **Tests first:**
   - `llm/provider`, `TestListingSearchByFacts`: a listing from a canned wire and a test models.dev document
     (`ModelMetadataURL`). `vision` finds the models whose input holds image, and `200k` those whose window is at
     least 200,000. `provider.Search` over the same ids finds neither.
   - `internal/cli`: `gobble models --search vision` and the picker's search return the listing's matches first, then
     the id matches among the models the listing does not hold.
2. **The code:**
   - `Listing` keeps its `catalog.Catalog`, unexported;
   - `provider.SearchListing(l Listing, query string) []string`, a name for the phase to confirm;
   - `internal/cli`'s listing cache hands its `Listing` to both searches;
   - `provider.Search` stays.
3. **Planted break:** the listing search replaced by the id search: `TestListingSearchByFacts` fails.
4. **Records:** `docs/guides/choosing-a-model.md`, the search's words. Gates. Commit.

### P6 — images in the adapter and the ACP prompt (D7; D8's ACP part)

1. **Tests first:**
   - `llm/provider`, `TestImagesToRequest`, through an `llmtest.Fake` with `Images: Supported`:
     - a user message's `ContentImage` is an `ImageItem` in place;
     - a tool message's image is an `ImageItem` right after its `FunctionCallOutputItem`;
     - the fake's recorded request shows both.
   - `TestImageWires`: on canned wires for `claude`, `openai`, `gemini` and a Chat Completions server, each request
     holds the image where the MADR's probe table says.
   - `TestCapabilitiesImageInput`: `ImageInput` follows `Capabilities.Images`.
   - `TestTextOnlyModelImage`: a model whose facts say text only ends the turn with an error matching
     `llm.ErrUnsupported`, and sends nothing.
   - `acpserver`:
     - `TestCapabilitiesAreHonest`: `image` is true for a default model that takes images, and false for a text-only
       one. Today's assertion that it is false is turned round.
     - `TestPromptImage`: an ACP image block reaches the provider.
     - `TestHistoryImageRoundTrip`: `session.BlockImage` round-trips through `history.go`.
     - `session.golden` unchanged.
2. **The code:**
   - `toRequest`'s image cases, and `Capabilities`;
   - `llm.go`'s comment;
   - `userMessage`, and `promptFiles` keeping image blocks;
   - `history.go` both ways;
   - `initialize`'s `promptCapabilities.image` from the configured default model.
3. **Planted breaks:**
   - the image dropped from `toRequest`: `TestImagesToRequest` fails;
   - the image put before the tool output: `TestImageWires` fails on Messages;
   - `image` advertised for a text-only default: `TestCapabilitiesAreHonest` fails.
4. **Records:** a dated note in 0008-MADR D19, that `promptCapabilities.image` follows the default model from here.
   Gates. Commit.

### P7 — images from `read`, MCP and `@path` (D8)

1. **Tests first:**
   - `tool/builtin`, `TestReadImage`:
     - a 3000×1000 PNG on a session whose model takes images returns one image, at most 2000 wide;
     - a JPEG with EXIF orientation 6 returns it turned;
     - a BMP returns a PNG;
     - a GIF returns its first frame as PNG or JPEG;
     - the base64 is at most 4.5 MB;
     - on a text-only model, the note.
   - `TestReadImageBroken`: a truncated PNG is an error naming the file, never a panic.
   - `mcpclient`, `TestMCPImageResult`: an `*mcp.ImageContent` result becomes image content, prepared the same way.
   - `agent`, `TestToolResultImage`: a tool result's images reach the request after its output.
   - `acpclient` / `internal/cli`: an image `@path` is sent as an ACP image block when the agent advertises images,
     and `ErrImagesUnsupported` otherwise. The agent's `@path` expansion does the same.
   - `internal/config`: `images.autoResize` is in the schema and template, default true. Off sends a small image
     unchanged, and one over 4.5 MB as the note.
2. **The code:**
   - `tool.Result.Images` and `tool.Env.Images`, both additions;
   - the image preparation, with `golang.org/x/image` added in this commit;
   - `read`, `mcpclient`, the agent's tool messages;
   - the CLI's `@path`;
   - the configuration key.
3. **Planted breaks:**
   - no resize: `TestReadImage` fails on the width;
   - EXIF ignored: it fails on the orientation;
   - a BMP sent as BMP: it fails;
   - the MCP image left as a note: `TestMCPImageResult` fails.
4. **Records:**
   - dated notes: 0005-MADR's `read` row and its `@` mentions, done here; 0018-MADR F11, `images.autoResize`
     unparked;
   - `docs/architecture.md`, `README.md`;
   - this plan `complete` once the owner's live checks are recorded or declined;
   - #4 closed when the owner asks in that turn.

   Gates. Commit.

## Verification

- Every new test seen failing first, its failure quoted. Planted breaks on a scratch copy, one at a time, each failing
  its test, and the restored copy passing.
- **Each phase:**
  - `make pre-add-check` on its Go files;
  - on macOS, the Linux machine and the Windows laptop: the tests, the race detector, `make preflight` (on the copies,
    to `verify-build-metadata`), `make check-records markdownlint`, and `make perf` within its budget;
  - `make probe-conhost` on the laptop.
- `apidiff` (in `make preflight`) reports only compatible changes to `llm` and `tool`.
- `internal/archtest` still refuses a go-llmprovider-sdk import outside `llm/provider`.
- `go mod tidy` is clean at every commit, and `golang.org/x/image` appears only from P7.
- The identifier scan of each phase's staged files finds no host name, account name or real-machine path.
- No test reaches a real service, a browser or a real sign-in.
- **Live, by the owner, billed and optional:**
  - one image turn on Claude and one on an OpenAI vision model, each with an image after a tool result;
  - one Chat Completions server's listing and turn.

  Each is recorded without any credential.

## Rollout and Rollback

One commit per phase. Reverting a phase reverts what it did:

| Revert | Restores |
| :--- | :--- |
| P7 | the image note, and `golang.org/x/image` leaves `go.mod` |
| P6 | text-only requests, and `image: false` |
| P5 | the search by id |
| P4 | the skip of unknown models.json keys |
| P3 | relearning in each process; the refusals file is left unread |
| P2 | silent warnings |
| P1 | the per-call registry |

P4 and P5 need P1; P3 needs P2; P7 needs P6. A later phase is reverted before the one it needs. No phase changes a
stored file's shape gobble already reads: `auth.json` and the session format are unchanged, and the refusals file is
new.

## Execution record

Approved by the owner's "committed, proceed", 2026-10-29. Q1–Q4 were answered the same day.

### P0 record (2026-10-29): waiting for 0035's go-llmprovider-sdk phases

P0's checks, run in the tree at `ccd8aa8`:

1. `go list -m github.com/maccavelli/go-llmprovider-sdk`: `github.com/maccavelli/go-llmprovider-sdk v1.22.2`, not
   `v1.24.1`. **Fails.**
2. `go test -count=1 -run TestWireOffline ./llm/provider/`: `ok`. It passes because the pin has not moved; at `v1.24.1`
   it fails without 0035's D3 row (0036-MADR, "How this was measured").
3. `git grep` finds `sdkEmptyAnswer` (`llm/provider/provider.go:523`) and `func offline` (`:548`): 0035's D4 is not
   in. **Fails.**

As step 4 says, P0 stops there, and nothing of P1 is written.

What P0 waits for: `0035-MADR-pin-current-sibling-releases-and-remove-gobble-copies.md` is now `accepted`. Its PLAN,
`0035-PLAN-pin-current-sibling-releases-and-remove-gobble-copies.md`, is `proposed`, written by another session, and,
like the MADR, not committed on this date. That PLAN moves go-llmprovider-sdk in its P4 (the pin and the offline row)
and P5 (the copies go). Both come after its P0–P3: the records, the docs, `check-libs`, and go-selfupdate-lib. Each of
its phases starts on the owner's word.

**The owner's choice, 2026-10-29** ("Record the wait"), of three put:

- the others were to amend this record to carry the pin and the copies itself, which would duplicate 0035-PLAN's P4
  and P5 in the same files;
- or to run P1 first on `v1.22.2`, reversing Q4's answer.

P0 runs again once 0035-PLAN's P5 has landed. MADR D1 is unchanged: the wait is the one it decided.

Found at the same check: the owner's "committed" had not landed. `HEAD` was still `ccd8aa8`, the reflog's last entry
was 08:29 that day, and this pair and the 0031 notes were still staged.

### P0 record, run again (2026-11-06): the precondition is met

On the owner's "update and fix 0036 P0 to current reality". Run in the tree at `d1d6a73`, equal to `origin/main`,
with the steps as amended that day.

1. `go list -m github.com/maccavelli/go-llmprovider-sdk`: `v1.26.1`, the tag 0035-MADR D1 names last as accepted. It
   is `v1.24.1` or later. **Met.** If 0035-PLAN's P32 lands first, it is `v1.27.0`, which a scratch clone showed
   changes none of this record's names (step 4) and none of gobble's tests.
2. `go test -count=1 -run TestWireOffline ./llm/provider/`: `ok … llm/provider 0.805s`. The test is in
   `llm/provider/offline_test.go:25`, and passes at a pin past `v1.24.1` because 0035's D3 row is in. **Met.**
3. `git grep -n sdkEmptyAnswer -- llm/provider` and `git grep -n 'func offline' -- llm/provider`: both exit 1, nothing
   found. 0035-PLAN's P4 (`bd79d2a`) and P5 (`43649b8`) landed on 2026-10-30. **Met.**
4. **The SDK's surface,** by `git grep` in a clone of go-llmprovider-sdk at `v1.26.1` and at `v1.27.0`, each with the
   same locations: `WarningEffortDegraded`, `WarningReasoningLearned`, `WarningImageDropped`
   (`llmprovider/warning.go:28`, `:34`, `:38`) and `Response.Warnings` (`llmprovider/item.go:140`);
   `ReasoningRefusal` (`refusal.go:32`) and `WithReasoningRefusals` (`:47`); `chatcompat.New`
   (`providers/chatcompat/chatcompat.go:174`, `:170` at `v1.24.1`), `WithID` (`:140`, was `:136`), `Descriptor`
   (`descriptor.go:9`) and `Discover` (`discover.go:65`, was `:63`); `Catalog.Search` (`catalog/search.go:205`) and
   `Match.Why` (`catalog/model_matcher.go:44`); `ImageItem` (`item.go:79`) and `Capabilities.Images`
   (`capabilities.go:48`). `git grep` over gobble's `*.go` for `Warnings`, `ReasoningRefusal`, `ImageItem`,
   `chatcompat` and `.Why` finds none; `.Search(` finds `catalog.Search` (`llm/provider/list.go:54`) and gobble's own
   tool search alone. **Met.**
5. **The citations,** spot-read, each fact unchanged and some lines moved since `ccd8aa8`: the adapter's `EventDone`
   is at `llm/provider/provider.go:348` (was `:337`); `ambient.go` still calls `providers.Default().Descriptors()` at
   `:49`, `:61`, `:81`, `:93`, `:103`, `:152` and `:163`, and `signin.go` passes `providers.Default()` at `:177` (was
   `:174`); models.json's skip message is `llm/catalog/modelsjson.go:201` (was `:196`); `toRequest`'s last case is
   `ContentToolResult` at `provider.go:486` (was `:443`), with no `ContentImage` case; `llm.Capabilities.ImageInput`'s
   comment (`llm/llm.go:45`) now names 0035-MADR D18 and 0036-MADR D7; `read`'s note is `tool/builtin/read.go:48`
   (was `:31`); MCP's `[image … omitted]` is `mcpclient/tool.go:153` (was `:159`). P1 to P7 read theirs again (step 5).

So P0 is met, and P1 may start on the owner's word. Nothing of P1 is written here. 0035-PLAN's P32 and this plan's P1
share only `README.md` and `docs/architecture.md`, on different lines, so either may land first and the second
rebases.
