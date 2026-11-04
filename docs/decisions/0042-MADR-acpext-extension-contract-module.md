---
status: proposed
date: 2026-11-03
decision-makers: repository owner
consulted: "gobble-cli at 67f72fb, read from code; acp-go-sdk2 v1.31.2, read from source; magic-cli-remote at 6b331cf7, its 0179-MADR and 0179-REPORT"
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Publish gobble's ACP extension contract as the nested module `acpext`, which gobble serves from and magic-cli-remote imports

## Context and Problem Statement

gobble's ACP agent (`acpserver`) adds a `_gobble/*` extension surface to the standard protocol. magic-cli-remote
registers gobble as a provider (magic-cli-remote `docs/decisions/0179-MADR-gobble-native-acp-provider.md`). To do that
it calls two of those requests, handles two of the notifications, and reads gobble's auth methods and error data.

Today nothing outside `acpserver` can name those shapes in Go. The types are unexported, so every client writes the
JSON by hand. magic-cli-remote's assessment says so as its risk R3
(magic-cli-remote `docs/reports/0179-REPORT-gobble-native-acp-integration-assessment.md`):

> gobble's extension result types are unexported (`compactResult` and others, `extensions.go:132-289`), so this
> repository would hand-write the JSON. That is how 0179's `instructions` came to differ from gobble's
> `customInstructions` (M9)

Its recommendation was: "gobble publishes a small nested module (for example `acpext`) with method names and types,
depending only on `acp-go-sdk2`, and this repository imports it. Vendor extensions do not belong in the generic SDK."

On 2026-11-03 magic-cli-remote's rewritten 0179 left that out. The owner then asked for it: "in addition to acp auth
and phone configuration working, i want the acpext module."

The question is how gobble publishes its extension contract so that a client imports it, and how gobble keeps
serving exactly what it publishes.

### What gobble has today

Read at `67f72fb`.

* **Twenty requests.** The table is `acpserver/extensions.go:28-49`:
  * steer, follow_up and clear_queue;
  * set_steering_mode and set_follow_up_mode;
  * compact, usage, set_session_name, fork and clone;
  * get_state, get_messages, get_entries, get_last_assistant_text, get_session_stats and get_fork_messages;
  * set_auto_retry, abort_retry, set_auto_compaction and set_personality.
* **Three notifications:**
  * `_gobble/retry` (`acpserver/extensions.go:53`; params `retryNotice`, `acpserver/prompt.go:80-89`);
  * `_gobble/compaction` (`acpserver/autocompact.go:20`; `compactionNotice`, `:31-38`, with its phases, reasons and
    pauses at `:23-29` and `:40-47`);
  * `_gobble/personality` (`acpserver/personality.go:22`; `personalityNotice`, `:24-29`).
* **The advertisement.** `initialize` puts the request names under `agentCapabilities._meta.gobble.extensions`
  (`capabilitiesMeta`, `acpserver/agent.go:371` and `:393-396`). `_meta.gobble` has other members too, in questions,
  grants, mentions and session metadata, which gobble's server and client exchange
  (0039-MADR D15, eight writers and seven readers).
* **gobble's JSON is its own.** Every `_gobble/*` payload gobble sends or receives is encoded and decoded by gobble,
  with `encoding/json/v2` under `jsonx`'s data policy, and handed to the SDK as a raw value (0039-MADR D10, D13).
  Error data is too (D14). No ACP payload is decoded with `RejectUnknownMembers` (D19).
* **Params and results.** Each is an unexported struct in `acpserver/extensions.go:80-121` and `:141-640`, plus
  `setPersonalityParams` and `personalityResult` (`acpserver/personality.go:280-291`). Four of them hold gobble's own
  types:
  * `compactResult.Details` is a `compaction.Details` (`:224`);
  * `forkResult.SessionID` is a `session.ID` (`:385`);
  * `messagesResult.Messages` is `[]*session.Message` (`:504`);
  * `entriesResult.Entries` is `[]session.Entry` (`:516`).
* **Auth.**
  * One `agent` method per provider that takes a credential, its id the provider id. With `auth.terminal`, a
    `terminal` method per provider is added too, `<provider>-login` (`acpserver/auth.go:30-66`; 0016-MADR A2).
  * `AuthProvider{ID, Name, Command, Login}` is exported, but only as `acpserver`'s input
    (`acpserver/auth.go:16-19`). The list itself is built in `internal/cli/agent.go:370-381`, from
    `llm/provider.KeyProviders`, `Label`, `HasLogin` (grok) and `HasKeyLogin` (kilo).
* **Error data.** Its keys are `reason` and `error` (`acpserver/agent.go:45-46`):
  * `authenticate`'s -32000 data is `{provider, reason, command}` (`acpserver/auth.go:80`);
  * a prompt without a credential gets -32000 with data `{error}` (`acpserver/prompt.go:285`);
  * a request refused mid-turn gets -32600 with data `{reason}`.
* **gobble already keeps the contract twice.**
  * `acpclient` decodes the three notifications with its own constants and types (`acpclient/update.go:119-195`,
    constants at `:127`, `:159` and `:186`).
  * It sends `_gobble/set_session_name` with its own `setNameParams` (`acpclient/sessions.go:105-120`).
  * It routes the notifications through `acp.ExtensionMux` (`acpclient/conn.go:428-470`).
  * Nothing but the tests on each side ties the two copies together.

### What constrains the shape

* **One module today.** `docs/decisions/0004-MADR-go-module-architecture.md` chose one module. It rejected a
  `go.work` multi-module repository: "cross-module changes need coordinated tags, and the fleet has no multi-module
  release tooling" (`:556-563`). Its confirmation is that `go list -m` shows one module (`:517`).
* **`go install` is a supported install path.** `go install github.com/maccavelli/gobble-cli/cmd/gobble@latest`
  (0003-MADR `:71`, 0005-MADR `:275`). `go install pkg@version` refuses a main module whose `go.mod` has `replace`
  directives, and it ignores `go.work`. So a module gobble requires must be reachable at a published version.
* **Releases.** The release workflow runs on tags matching `v*` (`.github/workflows/ci.yml:14`). `make apidiff` looks
  only at tags matching `v*` (`Makefile:259-266`). A tag `acpext/v0.1.0` matches neither.
* **Dependencies need a record.** AGENTS.md says no module may be required without a MADR that names it, and that
  `go mod tidy` is clean at every commit.
* **Import boundaries.** `internal/archtest` forbids acp-go-sdk2 in the core tier (rule 1), and allows it only under
  `acpserver` and `acpclient` (rule 2) (`internal/archtest/check.go:235-269`).
* **Raw JSON has one type.**
  * acp-go-sdk2's public API spells opaque JSON `json.RawMessage`: 920 uses in `types_gen.go`.
  * In Go 1.27, `encoding/json` declares `type RawMessage = jsontext.Value`, an alias, so the two are one type
    (0039-MADR `:135`). gobble spells it `jsontext.Value` (0039-MADR D11).
  * The SDK's `ExtensionMux` decodes a typed `Handle[P, R]` with `encoding/json` (`extensions.go:53-75`).
* **Both repositories are public.** gobble-cli and acp-go-sdk2 are, so the module proxy can serve a tagged nested
  module with no `GOPRIVATE`.
* **magic-cli-remote may not add go-llmprovider-sdk** (its `docs/decisions/0185-MADR-migrating-to-go-llmprovider-sdk-is-out-of-scope.md`).
  A package inside gobble's root module would bring gobble's whole requirement graph with it.

## Decision Drivers

* There is one definition of every `_gobble/*` name and shape. gobble serves from it, and `acpclient` and
  magic-cli-remote decode with it.
* An importer gets acp-go-sdk2 and the standard library, nothing else.
* The wire does not change. Moving the types does not change a byte gobble sends; the golden and extension tests prove
  it.
* `go install …/cmd/gobble@version` keeps working.
* Vendor extensions stay out of the generic SDK (0179-REPORT R3).

## Considered Options

* **A. A nested module `acpext/`, tagged `acpext/vX.Y.Z`.** The root module requires it at a tag, and `acpserver` and
  `acpclient` use its types.
* **B. A package `acpext` inside the root module.** magic-cli-remote would then require
  `github.com/maccavelli/gobble-cli`.
* **C. A separate repository** for the contract.
* **D. Types in acp-go-sdk2.**
* **E. No module.** Each client keeps hand-written JSON, pinned by golden fixtures (magic-cli-remote's 0179 as
  rewritten on 2026-11-03).

## Decision Outcome

Chosen option: **"A. A nested module `acpext/`"**, because it is the only one that gives an importer the contract
with nothing but the SDK, keeps the contract in the repository that serves it, and lets gobble serve from the same
types. Its cost is the coordinated tag that 0004 named; D5 below sets the order that keeps that cost small.

### D1. The module

* **Path and layout.** The directory is `acpext/`, the module `github.com/maccavelli/gobble-cli/acpext`, and the
  package `acpext`. It is one package, with no `internal/` and no sub-packages.
* **`go.mod`:**
  * `go 1.27.3`, as the root module, for fleet lockstep (0004's accepted cost);
  * a single requirement, `github.com/maccavelli/acp-go-sdk2`, at the lowest tag acpext needs. At creation that is
    the root's pin, `v1.31.2`. A later SDK bump raises it only when acpext uses something new; MVS gives every
    importer its own pin.
* **Imports.** Only the standard library and acp-go-sdk2. A gate checks it (D6).
* **`acpext/LICENSE`** is a copy of the root `LICENSE`. A nested module's zip holds only its own directory, so the
  module proxy and pkg.go.dev do not see the root's file.

### D2. What it holds

The contract as gobble serves it at the commit that creates acpext, with no addition and no rename.

* **Method names:**
  * a constant for each of the twenty requests and three notifications, named `Method` plus the method in Go case
    (`MethodSetSessionName`, `MethodClone`, `MethodRetry`, …);
  * `Requests()`, the request names sorted, which is what `initialize` advertises;
  * `Notifications()`, the notification names sorted.
* **The advertisement:**
  * `MetaKey = "gobble"`, the key under `_meta`;
  * `CapabilitiesMeta{Extensions []string}`, its value in `initialize`'s `agentCapabilities._meta`.

  The other `_meta.gobble` members stay `acpserver`'s and `acpclient`'s. They are traffic between gobble's own server
  and client, which no other client reads today.
* **Params and results.** Each of today's unexported structs becomes an exported type with the same fields, JSON
  names and `omitempty`/`omitzero` options. A type is named for its method (`SetSessionNameParams`, `CloneResult`,
  `RetryNotice`, …). A shape that several methods share keeps one name (`SessionParams`). The PLAN's P1 lists each
  type against its source line.
* **The four fields that hold gobble's own types** become `jsontext.Value`, gobble's spelling of the SDK's
  `json.RawMessage` (the same type):
  * `CompactResult.Details`;
  * each element of `MessagesResult.Messages`;
  * each element of `EntriesResult.Entries`.

  `ForkResult.SessionID` and `CloneResult.SessionID` become `acp.SessionId`, the same string on the wire. Their
  doc comments say the raw fields carry gobble's session-file shapes, which acpext does not freeze.
* **String values** become typed constants:
  * the notification phases (`start`, `end`);
  * the compaction reasons (`manual`, `threshold`, `overflow`) and pauses (`turn`, `session`);
  * the retry causes;
  * the queue modes (`one-at-a-time`, `all`);
  * the queue dispositions.

  The PLAN reads each value from the code.
* **Auth, D3.**

acpext holds names, types and fixed lists, and no behaviour: no connection, no handler, no encoder and no `_meta`
reader. Its types carry JSON tags only, which `encoding/json` and `encoding/json/v2` read alike.

* gobble keeps encoding and decoding them through `jsonx`, under 0039's policies.
* Another client calls with the SDK (`ClientSideConnection.CallExtension`, `acp.SendRequest`) and decodes with its
  own encoder, `acp.Handle` or `acp.ExtensionMux`.

### D3. The auth contract in acpext

* **`AuthProvider{ID, Name, Command string; Login bool}`** moves from `acpserver` into acpext, with the same fields
  and meaning. Its `ID` is the provider id and so the `agent` method id. `Login` is true where the provider has a
  sign-in rather than a key alone.
* **`AuthProviders()`** is the static list gobble advertises, in registry order. Today that is nine providers: gemini,
  openai, claude, grok, opencode-zen, opencode-go, huggingface, kilo and together. Only grok and kilo have `Login`.
  * The list is static so a client can show gobble's providers before any `initialize`. magic-cli-remote's phone
    settings sheet needs that (its 0179 D3).
  * acpext cannot import `llm/provider`, so the root module derives the same list at run time, from the SDK
    registry, as today. A root test asserts the two are equal (D4).
* **`LoginSuffix = "-login"`**, the terminal method id's suffix.
* **`ErrorData{Reason, Error, Provider, Command string}`.** These are the `data` members of gobble's errors, each
  `omitzero`: -32000 from `authenticate`, -32000 from a prompt without a credential, and -32600 for a mid-turn
  refusal. The code is the SDK's, and acpext adds no error constructor.

### D4. gobble serves and reads from acpext

* **`acpserver`:**
  * Its private structs and method constants give way to acpext's.
  * The `extensions` table is keyed by acpext's constants.
  * `Capabilities` uses `MetaKey` and `CapabilitiesMeta`.
  * `AuthOptions.Providers` returns `[]acpext.AuthProvider`.
  * Error data is built from `ErrorData`.
* **`acpclient`.** `update.go`'s three constants and the decoding behind `Retry`, `Compaction` and `Personality`, and
  `sessions.go`'s `setNameParams`, give way to acpext's. `acpclient`'s exported `Update` types stay as they are. They
  are its API to the CLI and TUI, not the wire.
* **What proves the wire did not change.** `acpserver/testdata/session.golden` and the extension tests of both
  packages pass unchanged. A new root test asserts:
  * the `extensions` table's keys equal `acpext.Requests()`;
  * every method `acpserver` notifies is in `acpext.Notifications()`;
  * `acpext.AuthProviders()` equals the list `internal/cli` builds from the registry.

  Each assertion is seen failing on a planted break.
* **archtest.** acpext imports the SDK, so it is placed with it: only `acpserver`, `acpclient` and `internal/cli` may
  import acpext. `internal/cli` is the place `AuthProviders` is compared against the registry. The rule is seen
  failing on a planted import from a core package.
* **Not changed.** gobble's own `HandleExtensionMethod` dispatch stays. Moving it onto `acp.Handle` would change how
  bad params are refused, and that is a wire change for another record.

### D5. Versions and release order

* **Tags** are `acpext/vMAJOR.MINOR.PATCH`, Go's form for a module in a subdirectory. The first is `acpext/v0.1.0`.
  Tags and pushes are the owner's, on an explicit word, as for every tag.
* **v0 until gobble's v1.** A change to the contract is a new minor tag, named in a record as a wire change (0014-MADR
  D5). acpext moves to v1 with gobble's own v1.0.0, when `make apidiff` starts to apply.
* **The root requires acpext by tag.** It never uses a `replace` directive, and no `go.work` is committed. Both would
  break `go install …@version`, or disagree with it. `go.work` and `go.work.sum` go into `.gitignore`, so a local
  workspace for working on both at once never lands.
* **Order of a change.**
  1. The acpext change is committed, pushed and tagged.
  2. The root commit that needs it runs `go get …/acpext@<tag>` and adapts.
  3. A gobble release then carries both, and `go version -m` on the binary names the acpext version it serves.

  The first change, the creation, follows the same order: the PLAN's P1, then its P2.

### D6. Gates and tooling

* **Every module's gates.** `make preflight`, `make test` and `make race` run over the root and `acpext/`:
  * gofmt, `go mod tidy`, vet, staticcheck for the three OSes, golangci-lint, govulncheck, `go fix -diff` and tests;
  * each in the module's own directory (`go -C acpext …`).

  `gofmt -l .` and `pre-add-check` already reach `acpext/` files.
* **`check-acpext-deps`,** in `preflight`, fails when `go -C acpext list -deps` names a non-standard package outside
  acp-go-sdk2 and acpext itself. It is seen failing on a planted import.
* **CI.** The existing jobs run the same per-module targets, so CI and `make preflight` agree.
* **Documentation, updated in the phase that makes them true:**
  * AGENTS.md's "one module" sentence;
  * `docs/architecture.md`'s module and package map;
  * a dated note on 0004-MADR. Its Confirmation's `go list -m` still shows one module, since `acpext/` is outside
    the root.

### Consequences

* Good, because gobble's server, gobble's client and magic-cli-remote decode one definition. A renamed field becomes a
  compile error at the next `go get`, not a silent mismatch.
* Good, because an importer's graph gains acpext and the SDK it already has. magic-cli-remote's `go.mod` gains one
  requirement, and no go-llmprovider-sdk.
* Good, because `AuthProviders()` gives a client gobble's providers with no session open, and gobble's own test keeps
  it equal to the registry.
* Good, because `acpclient`'s second copy of the notification shapes goes away.
* Neutral, because the wire is unchanged, and the existing golden tests are the proof.
* Bad, because a contract change now needs two commits and a tag, in order, before gobble can serve it. 0004 named
  this cost.
* Bad, because the static provider list must be updated with the registry. The equality test fails the root's build
  until it is, which turns an SDK provider addition into an acpext tag.
* Bad, because the `jsontext.Value` fields leave those shapes to the client.

### Confirmation

1. `go -C acpext list -m all` names acpext, acp-go-sdk2 and the SDK's own requirements, and nothing else.
2. `make check-acpext-deps` passes, and was seen failing on a planted import.
3. The root's `go.mod` requires `github.com/maccavelli/gobble-cli/acpext` at a tag, with no `replace`. `go install
   github.com/maccavelli/gobble-cli/cmd/gobble@<commit>` succeeds from an empty module cache, and `go version -m`
   names acpext.
4. `acpserver/testdata/session.golden` and the extension tests of `acpserver` and `acpclient` pass with no change to
   their expected frames.
5. D4's root test passes, and each of its three assertions was seen failing on a planted break.
6. archtest's acpext rule passes, and was seen failing on a planted import.
7. A scratch module outside the tree that requires `acpext@v0.1.0` from the proxy builds, decodes a recorded
   `_gobble/retry` frame into `RetryNotice`, and its `go list -m all` shows no go-llmprovider-sdk.

## Pros and Cons of the Options

### A. A nested module `acpext/`

* Good, because the importer gets the contract and the SDK, nothing more.
* Good, because the types live beside the code that serves them, and gobble serves from them.
* Good, because `go install` keeps working: the root requires a published tag.
* Bad, because a change needs an acpext tag before the root can use it (D5).
* Bad, because the Makefile and CI must run each gate per module.

### B. A package inside the root module

* Good, because there is one module, one tag and no ordering.
* Bad, because magic-cli-remote would require all of gobble-cli. go-llmprovider-sdk, go-tui-lib, Kong and the rest
  would enter its module graph and `go.sum`, against 0185's intent.
* Bad, because the contract could be versioned only with gobble's releases. A pre-release fix to a type would mean
  pinning gobble's main branch, which magic-cli-remote's 0186 D9 forbids.

### C. A separate repository

* Good, because there is a plain module with plain tags.
* Bad, because the contract moves away from the code that serves it. Each change is then a cross-repository change,
  with its own records, CI and releases, for a package of types and constants.

### D. Types in acp-go-sdk2

* Bad, because vendor extensions do not belong in the generic SDK (0179-REPORT R3), and the SDK would release for
  gobble's changes.

### E. No module

* Good, because there is no dependency and no tag ordering.
* Bad, because every client keeps hand-written JSON, which is how the M9 mismatch happened. Fixtures catch a drift
  only at the next capture.
* Bad, because gobble's own two copies stay.
* Bad, because the owner asked for the module.

## More Information

### Not decided here

* **Moving `acpserver`'s dispatch onto `acp.Handle`/`ExtensionMux`.** It changes how bad params are refused, a wire
  change.
* **New methods,** such as a credential write or a provider status over ACP. Each is a wire change with its own
  record.
* **Freezing the shapes behind the `jsontext.Value` fields** (session messages and entries, compaction details).
* **The other `_meta.gobble` members** (0039-MADR D15), should a client other than gobble's need them.
* **acpext v1,** which waits for gobble's v1.0.0.

### Relationship to other records and issues

* [0004-MADR-go-module-architecture.md](0004-MADR-go-module-architecture.md): one module chosen. This record adds a
  second, nested one for the contract alone, and the PLAN adds a dated note there.
* [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md): A2, the
  auth methods acpext's D3 publishes.
* [0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md](0039-MADR-one-json-v2-encoder-and-invalid-utf8-policy.md):
  D10, D13 and D14, gobble's encoding of the payloads acpext types; D15, the typed `_meta.gobble`; D19, no
  unknown-member rejection. acpext changes none of them.
* `0002-PLAN-cli-acp-headless-mcp-v1.md` (Phase 7), `0005-PLAN-v1-feature-scope.md` (F5-1, F5-2) and
  [0022-MADR-gobble-voice-and-personalities.md](0022-MADR-gobble-voice-and-personalities.md): where the methods
  acpext names were decided.
* magic-cli-remote `docs/decisions/0179-MADR-gobble-native-acp-provider.md`: the importer, amended the same day to
  require acpext.
* magic-cli-remote `docs/reports/0179-REPORT-gobble-native-acp-integration-assessment.md`: R3, the recommendation
  this record takes.
