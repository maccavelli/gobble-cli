---
status: proposed
date: 2026-11-23
associated-madr: "0068-MADR-adopt-go-llmprovider-sdk-v1-32-0.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement Adopting go-llmprovider-sdk v1.32.0

Associated MADR: [0068-MADR-adopt-go-llmprovider-sdk-v1-32-0.md](0068-MADR-adopt-go-llmprovider-sdk-v1-32-0.md)

*(2026-11-23, second draft, after the independent review: the consent test, both test issuers' profile, the held-session test 0052 gave this pin, the ruled switch of the plan tests, every key source, the VerifyKilo options rule, plants that can turn their tests red, and the named exceptions to red-first.)*

## Goal

After this PLAN, `go.mod` requires go-llmprovider-sdk `v1.32.0`; the schema-refusal texts, the Kilo notice and `SignInError.Keys` say what the new SDK makes true; a Kilo key is verified at the profile on the terminal and over ACP; the browser question is asked as the SDK asks it (Q1, ruled 2026-11-23); a blank key is no key (Q2, ruled 2026-11-23); the two terminal plan tests no longer bind the SDK's ports (0052's ruling); a held plan session is tested to refuse a foreign entry; and every decided row of the MADR has a test and a plant that turns it red.

This PLAN is `proposed`. The MADR is `accepted` (2026-11-23) with no open row: Q1 "Yes, take the SDK's question (Recommended)" and Q2 "Treat it as no key (Recommended)". Nothing below Phase 0 step 3 starts before the owner approves this PLAN and says `proceed`. Nothing is committed by the agent: each phase ends with its files **staged**, gates recorded; the owner commits.

## Scope

### In scope

* `go.mod`, `go.sum` (the pin only, plus what `go mod tidy` demands).
* `llm/provider`: `signin.go` (`Secret`'s count), `signinrun.go` (`checkListing` and `VerifyKilo`), `list.go` or `policy.go` (the predicate for L3).
* `internal/cli`: `login.go` (the notice); `internal/auth/resolve.go` (Q2).
* Tests: `llm/provider/*_test.go`, `internal/auth/*_test.go`, `internal/cli/*_test.go` (including the two test issuers' profile handler), `acpserver/mcp_test.go`.
* Docs (Phase 3): `README.md:175`, `docs/architecture.md:200` and `:111`, `docs/guides/developing.md` (ports paragraph), `docs/guides/credentials.md` (`:312`, the Kilo check, the question), the dated notes in 0052 and 0060, the index rows.

### Out of scope

* The typed schema-error fallback and the `signature` misfire (0060-MADR D6's own record). SDK 0088 and 0089 (0065, 0066; a second pin bump). Cutting `v0.8.0` or any tag. `TextPrompter`. An organization for Kilo.
* A change to `acpext` or to any other module.

## Rules for Every Phase

* **Re-ground first.** Re-read each `path:line` the MADR cites at the phase's start commit and record each moved line.
* **Red, then green.** A test marked red runs at the phase's base (the pin still `v1.31.0`) and must fail on its assertion, never on a compile error. Every Phase 1 test uses only gobble's own API and compiles at `v1.31.0`. **One test is written after the bump, by name, because it uses SDK names that do not exist at `v1.31.0`:** `TestToolSchemaRefusalIsTyped` (E1: `ErrToolSchemaRefused`, `APIError.Tool`); it has no red at the base and is held by its plant. The guards (named in the table) pass at the base and are held by their plants.
* **Plants.** One production change each, made on a scratch clone, never in the tree; each must turn its named test red; the record quotes the failing line. A plant that does not turn its test red is a deviation: the test or the plant is wrong.
* **Heavy runs** (`make preflight`, `make race`, `go test ./...`, any `-race`) go through `~/.agents/skills/available-dev-hosts/scripts/pick-host`, then `run-on-host` (with `--race` where `-race` is hidden in a `make` target, never `-race` on WSL), one at a time per host; a single package's `go test` without `-race` may run on the session host.
* **No live call, no sign-in, no key.** Every provider answer is a test double (`HTTPClient` or an in-process issuer); the network guard of `internal/cli` stays on.
* **Stop conditions**, each a stop and a prompt: a test outside the MADR's rows turns red at the bump; a row cannot be met as written; a third build pass of one phase.
* **Exit criterion of every phase:** no blocker, every row tested, plants caught.

## Implementation Steps

### Phase 0: re-ground and discover (records and a scratch clone only)

1. Re-read at HEAD: `go.mod:16`; `llm/provider/signin.go:95`, `:150-157`, `:345-375`, `:416-470`; `signinrun.go:98-123`, `:177`, `:205`, `:232`; `list.go:46-54`; `login.go:73-83`, `:133`, `:136-215`; `provider.go:149-163`, `:218`, `:737-754`, `:820-860`; `replay.go:119-140`; `internal/cli/auth.go:281-304`, `login.go:86`, `:133`, `:272`; `internal/cli/prompt.go:69-75`; `internal/cli/credentials.go:228`; `internal/auth/logins.go:72`, `resolve.go:77-150`; `llm/catalog/value.go:28`; `internal/filelock/lock.go:61-76`; `wire_union_test.go:159`; `mcp_test.go:525`; `login_test.go:361`; `consentwording_test.go:131`; `loginissuer_test.go:89-190`; `loginjson_test.go:135`, `:480`, `:515`; `plansignin_test.go:255-300`, `:390`; `developing.md:87-92`; `architecture.md:111`, `:200`; `credentials.md:312`; `README.md:175`; 0052-PLAN's step 1c and 1d and "Third review". Record each moved line.
2. **Discovery on a scratch clone** (a `git clone` into the scratchpad; the tree is never touched): bump the pin there, `go mod tidy`, `go build ./...`, `go vet ./...`. Record the `go.mod`/`go.sum` diff (D1). Run the suite there through `run-on-host` (no `-race`) and record the tests that turn red. The MADR expects exactly: `TestSignInGrokBrowser` and any scripted browser sign-in in `signin_test.go`, `steps_test.go`, `signinrun_test.go`; the five terminal subtests of `TestConsentWordingOnEverySurface`; the two terminal plan tests; the refusal tests of `wire_union_test.go` and `mcp_test.go`; `TestAuthLoginJSONKiloKey` and the kilo cases of `TestAuthLoginJSONRetryable` and `TestAuthLoginJSONDoneFollowsTheStore` (a profile 404 warns on stderr); the Kilo cases of `TestSignInDevice` and `TestStartSignInChecksKiloDeviceKey` if their issuers lack a profile; `TestLoginAsksThroughQuestions` (the Kilo notice). Write the list with each test's reason. A red outside it is a deviation: stop and prompt (the MADR gains a row).
3. Read the SDK (module cache of the scratch clone) for: the Kilo profile's request (`GET {origin}{prefix}/api/profile`) and the 200 body the doubles must send; the redirect address a no-listener sign-in names (W3's proof dials it) and that Grok's listener is on an OS-chosen port; the exact error `VerifyKilo` returns for rejected options. Grep for `TextPrompter` and `NoLoopback` in non-test code: expected none and `signinrun.go:107` only (K5, W8).
4. `docs/guides/credentials.md`, `docs/architecture.md`: grep for the sign-in questions, the Kilo check, the ports and the held-session sentences; note which lines Phase 3 changes.

Exit: citations re-grounded; the reds-by-bump list recorded and equal to the MADR's expectation; nothing but the records changed.

### Phase 1: red tests at `v1.31.0` (the pin unchanged)

**Step 1a: the profile handler, behaviour unchanged.** `loginIssuer` (`internal/cli/loginissuer_test.go`) and `signInIssuer` (`llm/provider`) gain `GET /api/profile`: 200 with the body Phase 0 step 3 recorded by default, 401 or 503 when the test asks (`ti.profile(status)`). At `v1.31.0` nothing reads it, so every existing test still passes; this is the seam that lets Phase 2's tests assert empty stderr. **Step 1b: the red tests.** Each compiles at the base and must fail on its assertion.

| Row | Test | File | Shape | Red at the base because |
| :--- | :--- | :--- | :--- | :--- |
| W1, W2 | `TestSignInAsksWhereTheBrowserIs` | `llm/provider/signin_test.go` | Grok browser sign-in over the in-process issuer, a `script` that records menus; "On this machine" answered; while the sign-in waits the test dials the authorize URL's `redirect_uri` and connects; `OpenURL` called; saved | no such menu is asked |
| W3 | `TestSignInAnotherDeviceNeedsNoListener` | same | answer index 1 on Grok; `OpenURL` not called; the URL in the notes; the same dial is refused (no listener; Grok's listener is on an OS-chosen port, so no test holds 1455 or 1457 and parallel plan tests cannot collide); a pasted callback completes it | the question is absent and a listener is bound |
| W4 | `TestSignInNoPasteAsksNoBrowserQuestion` | same | `NoPaste` and `Steps` each: no `Choose` call | a guard (green at the base); plant P13 |
| W5 | `TestSignInDeviceAsksNoBrowserQuestion` | same | Grok device and Kilo device: no `Choose` beyond the method menu | a guard with **no plant**: the rule is the SDK's and no gobble line decides it |
| W6 | `TestSignInGrokBrowser` and the five terminal subtests of `TestConsentWordingOnEverySurface` answer the question | `signin_test.go`, `internal/cli/consentwording_test.go` | edited first: the script gains `choose` and asserts two menus; the consent test's answers become method, browser question (index 0), consent; its `shownFirst` check still reads the consent before any URL | the base asks one menu only; the new assertion fails (the consent test's extra answer is read as the consent at the base) |
| W7 | the two terminal plan tests switch | `internal/cli/plansignin_test.go` | each answers "On another device" and pastes (the test already ends by a paste on stdin); `signInFailed` and its port message are removed | at the base the answer is read as the consent, the sign-in fails on that assertion |
| W9 | `TestStartSignInPasteWithoutScheme` | `llm/provider/signinrun_test.go` | ACP paste step given `<host:port>/<path>?code=..&state=..` with no scheme | refused at the base |
| W13 | `TestSignInAnotherDeviceEmptyPastes` | `signin_test.go` | answer index 1, then three empty pastes: error matches `ErrInvalidRequest`, `ErrorKind` `invalid_request`, nothing saved | the base asks no question and reads the empty paste differently |
| W14 | `TestSignInCancelAtBrowserQuestion` | same | the prompter cancels at the question: its own error returned, nothing saved, no URL opened | no question at the base |
| V1 | `TestSignInCancelledSavesNothing` | same | context cancelled during the model listing; `memLogins` holds nothing; error is the context's | a new session is saved at the base (0081) |
| V2 | `TestLogins_LockRefreshFreeWithDoneContext` | `internal/auth/logins_test.go` | a cancelled context, nobody holds the lock: `LockRefresh` returns an unlock | a guard (green at the base); plant P9 |
| V3 | `TestLogins_LockRefreshHeldWithDoneContext`, `TestSignInWaitsForAHeldRefreshLock` | `logins_test.go`; `signin_test.go` (a test locker over `memLogins`) | held lock: error wraps `context.Canceled`; the wizard's notice says it waits; released after a moment: saved | no waiting notice at the base; saved under no lock (the first is a guard, plant P10) |
| V4 | `TestSignInHeldLockUnasked` | `signin_test.go` | an unanswering prompter, lock held past a short window: the store's error, nothing saved | saved at the base |
| V5 | `TestRefreshRefusesAnotherSignIn` | `llm/provider/login_test.go` | a session from login A; the store then holds login B (other client id); a due refresh: auth failure, `LoginRejected`'s sentence, the issuer saw no refresh of B's token | adopted at the base |
| V6 | `TestPiGrokLoginRefreshesAfterTheBump` | same | `PiLogin("grok", ...)` stored with no kind, refreshed against the issuer: succeeds, rotation saved | a guard (green at the base); plant P14 |
| V8 | `TestReviewCachedSessionAdoptsAForeignEntry`, adopted from 0052-PLAN's "Third review", renamed `TestHeldPlanSessionRefusesAForeignEntry` | `internal/cli` or `llm/provider`, where that text puts it | a held `chatgpt_plan` session; another application's login written to the store; the next token request fails with an auth failure, sends no foreign token, the session stays `chatgpt_plan` | the base adopts it (the reproduction fails by design today) |
| K1, K1a, K1b | `TestBlankEnvKeyIsUnset`, `TestBlankStoredKeyIsUnset`, `TestBlankModelsJSONKeyIsUnset` | `internal/auth/resolve_test.go` | a value of spaces from an environment variable, `auth.json` (`$VAR` and literal) and `models.json` (`$VAR` and literal): `Credential` and `Has` agree it is no key | resolved as a key at the base |
| K3 | `TestSignInKeysCountsNonBlank` | `signin_test.go` | secrets `refused-1`, `"   "`, then refused ones; `Keys` equals the number of non-blank secrets the script returned | blank counted at the base |
| K4 | `TestSignInRefusedThenBlanks` | same | `refused-1` then three blanks: text `grok sign-in failed (other)`, `errors.Is` `llm.ErrAuth` false, `Keys` 0 | the base ends in a count that includes blanks |
| K6 | `TestSignInTerminalSpacesAskedAgain` | `internal/cli/login_test.go` | the real `prompter` over `questions` fed `"   \n"` three times: asked again, then an error with no sentinel | the base returns the spaces as a key |
| L1 | `TestSignInKiloWrongKey` | `signin_test.go` | Kilo typed key, the issuer's profile answers 401: refused, asked again, ends `kilo refused 3 keys; none was saved` | the base accepts the key |
| L2 | `TestSignInKiloUnverifiedKeyWarns` | same | profile 503: one notice "could not verify the Kilo key", key returned | no notice at the base |
| L3 | `TestUncheckedKeyNotice` | `internal/cli/login_test.go` | `loginAsksThroughQuestions` split: the notice for `opencode-zen`, `opencode-go`, `huggingface`; none for `kilo` | the notice is given for Kilo |
| L4 | `TestSignInDevice` kilo case asserts a 401 profile refuses the device key | `signin_test.go` | edited first | accepted at the base |
| L5, L9 | `TestSignInRunKiloKeyVerified` | `signinrun_test.go` | four cases over a Kilo device-key run with `Cache` and `Headers` set in the options and a double that answers the **models listing 200 and the profile 401** (so only a profile read can refuse): profile 401 -> `ErrAuthFailure`, no save; profile 503 -> passes; the profile stalls and the context is cancelled -> `Wait` returns within about a second; a base URL that does not parse -> the sign-in fails with `ErrInvalidRequest` and saves nothing (if the run fails earlier, Phase 0 names the seam) | the 401 case passes the key at the base; the stall case waits on nothing; the fourth is not reached |
| L8 | `TestAuthLoginJSONKiloUnverifiedWarnsOnStderr`; `TestAuthLoginJSONKiloKey` and the kilo cases of the other two keep asserting empty stderr | `internal/cli/loginjson_test.go` | profile 503: the SDK's warning on stderr, stdout's steps unchanged; the others run on an issuer whose profile answers 200 | no warning at the base for the new test |
| E1 | `wire_union_test.go:159` and `mcp_test.go:525` edited to the new text | `llm/provider/wire_union_test.go`, `acpserver/mcp_test.go` | text begins "llm: api status 400: llmprovider: tool schema refused: grok HTTP 400 ..." | the base says "invalid request" |

For each red test: run it (`go test -run '^<name>$' <pkg>` at the base), record the failing assertion line. A test that passes at the base is only a named guard.

### Phase 2: the bump and the changes

1. `go.mod`: `github.com/maccavelli/go-llmprovider-sdk v1.32.0`; `go mod tidy`; the diff recorded (D1).
2. `llm/provider/signin.go`: `wizardPrompter.Secret` counts only secrets that are not blank once trimmed (K3, K4; D2 c). The browser question reaches `Choose` unchanged (`Select` already forwards); gobble sets neither `NoBrowserQuestion` nor `NoLoopback`. The two plan tests' switch (W7) is edited here if Phase 1 left it.
3. `llm/provider/signinrun.go`: `checkListing`, for Kilo with a key, calls `signin.VerifyKilo(ctx, key, "", o.sdkOptions()...)`: an error matching `ErrAuthFailure` and not `ErrNotPermitted` is returned (`refusedCredential`); an error matching `ErrInvalidRequest` is returned as the sign-in's failure; the ended context's error is returned; nil passes (L5, L9). The comment citing `v1.31.0` is re-read and moved only where `v1.32.0` still says it.
4. `llm/provider/list.go` (or `policy.go`): a predicate `KeyCheckedAtSignIn(id)` true for `kilo`; `internal/cli/login.go:133` notes "not checked" for `PublicListing(id) && !KeyCheckedAtSignIn(id)`. `PublicListing` and `choice.go:116` unchanged (L7).
5. `internal/auth/resolve.go`: one helper, a key empty once trimmed is none, applied at the environment variable (`:95`, `:124`), `auth.json`'s `key` (`:141`, `Has` `:113`) and `models.json`'s resolved value (`:83-91`, `:115-121`) (Q2). `--api-key -` and `!command` are unchanged (K1c, K1d).
6. `TestToolSchemaRefusalIsTyped` is written (E1, named exception): text, `errors.Is` both kinds, `APIError.Tool` is the tool, `llm.Kind` `other`.
7. The held-session test of V8 passes; the docs sentences it falsifies are Phase 3's.
8. Only after steps 1 to 7: each Phase 1 test runs again and passes; the guards still pass; a `-count=3` run of the sign-in packages shows no flake.

**Plants** (scratch clone, one production change each):

| Plant | Production change | Turns red |
| :--- | :--- | :--- |
| P1 | `Secret` counts every ask again | `TestSignInKeysCountsNonBlank` |
| P2 | `checkListing` skips `VerifyKilo` | `TestSignInRunKiloKeyVerified` (401 case: the listing answers 200, so nothing else refuses) |
| P3 | `VerifyKilo`'s error is dropped (`return nil`) | `TestSignInRunKiloKeyVerified` (401 case) |
| P4 | `VerifyKilo` called with `context.Background()` | `TestSignInRunKiloKeyVerified` (stall case: `Wait` returns only after the SDK's own 10 s, not within about a second; the ended-context return alone cannot catch it) |
| P5 | `KeyCheckedAtSignIn` returns false | `TestUncheckedKeyNotice` |
| P6 | `loginOf` drops `ClientID` | `TestRefreshRefusesAnotherSignIn`, `TestLoginCarriesEverySessionField` |
| P7 | `rejected` no longer leads with `LoginRejected` | `TestRefreshRefusesAnotherSignIn` |
| P8 | `mapErrorAt` returns `errors.New(err.Error())` for an `APIError` | `TestToolSchemaRefusalIsTyped` (`errors.Is`, `Tool`) |
| P9 | `Logins.LockRefresh` checks `ctx.Err()` before the first try | `TestLogins_LockRefreshFreeWithDoneContext` |
| P10 | `Logins.LockRefresh` returns an error that does not wrap the context's when held | `TestLogins_LockRefreshHeldWithDoneContext` only: `llm/provider` cannot use `auth.Logins` (`internal/auth` imports it), so `TestSignInWaitsForAHeldRefreshLock` cannot see this plant |
| P11 | `sdkStore.LockRefresh` returns a no-op unlock instead of the store's | `TestSignInWaitsForAHeldRefreshLock` |
| P12 | `SignIn` returns the wizard's result when `ctx.Err() != nil` | `TestSignInCancelledSavesNothing` |
| P13 | `NoPastePrompt` forced `false` | `TestSignInNoPasteAsksNoBrowserQuestion` |
| P14 | `PiLogin` sets a wrong `Issuer` | `TestPiGrokLoginRefreshesAfterTheBump` |
| P15 | `checkListing` treats an `ErrInvalidRequest` as a pass | `TestSignInRunKiloKeyVerified` (fourth case) |
| P16a-c | the resolver's blank check removed at the environment, `auth.json` and `models.json` source in turn | `TestBlankEnvKeyIsUnset`, `TestBlankStoredKeyIsUnset`, `TestBlankModelsJSONKeyIsUnset` |
| P17 | `NoBrowserQuestion: true` set | `TestSignInAsksWhereTheBrowserIs`, `TestSignInAnotherDeviceNeedsNoListener` |
| P18 | `loginOf` drops `Resource` (the plan session's marker) | `TestHeldPlanSessionRefusesAForeignEntry`; if it does not turn red, Phase 2 chooses the production change that does and records the deviation |

**Gates** (one at a time per host; `pick-host` first, `run-on-host`, `--need windows` for the Windows laptop as 0067 did):

* The sign-in packages' tests (`./llm/provider/ ./internal/auth/ ./internal/cli/ ./acpserver/`) on macOS, Linux and the Windows laptop; `-count=3` on `internal/cli` once.
* `make preflight` on the three hosts; then `make check-records markdownlint`.
* `python3 scripts/check_records.py --check-all` before the files are staged (exit 0 at drafting).

**Acceptance (Phase 2):** every decided row of the MADR has its test seen failing at the base (or is a named guard or named exception, with its plant caught) and passing now; no existing test is edited without a row naming it; files staged, not committed.

### Phase 3: docs and close-out

1. `README.md:175` and `docs/architecture.md:200`: `v1.32.0`. `docs/architecture.md:111` (the `llm/provider` row) and `docs/guides/credentials.md:312`: the held-session limit is gone; they say a session refuses another sign-in's entry at its next refresh and the request fails. `docs/guides/developing.md:87-92`: the ports paragraph removed (both plan tests switched; W7). `docs/guides/credentials.md`: the browser question (Q1) and that a Kilo key is checked at sign-in. No other change unless Phase 0 step 4 finds one.
2. Dated notes: 0052-MADR O1 and 0052-PLAN's steps 1c and 1d ("done by 0068-PLAN"); 0060-MADR D6 and 0060-PLAN's "still future" ("its pin precondition is met by 0068-PLAN").
3. A grep that no document or comment says `v1.31.0` of what is now `v1.32.0` (comments that name `v1.31.0` as the version a fact was read at stay, by name) and that no document says a held session can still adopt an entry.
4. The MADR's Confirmation 1 to 4 checked; MADR `accepted` after the owner's answers; this PLAN `complete` after the gates and the owner's approval. The `v0.8.0` tag is told to the owner; this PLAN does not tag.

## Verification

Each command with its exit status captured before any filter, long output redirected to a scratch file: the Phase 0 reds-by-bump list; every Phase 1 red line; the passes; the plants P1 to P18 (the failing line quoted, the clone discarded); the three-host gates; `check_records.py`. Heavy runs through `run-on-host` or `run-heavy`; `run-heavy --status` is read before each.

## Rollout and Rollback

No flag. Rollback is reverting the staged files or the owner's commit; the pin returns to `v1.31.0`, and the two refresh defects (0081, 0082) return with it. A person who has signed in under `v1.32.0` keeps a valid `auth.json` (no stored field changes).

## Execution Record

### Drafting (2026-11-23)

Records only, at `84dad4a`; second draft after the independent review. Read: the SDK's CHANGELOG `[v1.32.0]` and its records 0080, 0082, 0083, 0084 and 0085 where cited; gobble's call sites. No tool run beyond reads, searches and `python3 scripts/check_records.py` (`--next` printed 0068; `--check-all` exit 0). No phase has started; nothing is approved.
