---
status: complete
date: 2026-11-23
associated-madr: "0068-MADR-adopt-go-llmprovider-sdk-v1-32-0.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement Adopting go-llmprovider-sdk v1.32.0

Associated MADR: [0068-MADR-adopt-go-llmprovider-sdk-v1-32-0.md](0068-MADR-adopt-go-llmprovider-sdk-v1-32-0.md)

*(2026-11-23, second draft, after the independent review: the consent test, both test issuers' profile, the held-session test 0052 gave this pin, the ruled switch of the plan tests, every key source, the VerifyKilo options rule, plants that can turn their tests red, and the named exceptions to red-first.)*

*(2026-11-23, approved: the owner, relayed by the coordinator: "Approve both; 0069 then 0068 (Recommended)". 0069 was built and committed first (`f6c6b47`); this plan starts at `f6c6b47`. Status `in-progress`. The agent stages nothing and commits nothing; the owner stages and commits.)*

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
| K6 | ~~`TestSignInTerminalSpacesAskedAgain`~~ `TestSignInTerminalSpacesCancel` (deviation 2) | `internal/cli/login_test.go` | ~~the real `prompter` over `questions` fed `"   \n"` three times: asked again, then an error with no sentinel~~ the real `prompter` over `questions` fed `"   \n"` three times: one ask, `errCancelled`, nothing saved | ~~the base returns the spaces as a key~~ a guard, green at the base and after; plant P19 |
| L1 | `TestSignInKiloWrongKey` | `signin_test.go` | Kilo typed key, the issuer's profile answers 401: refused, asked again, ends `kilo refused 3 keys; none was saved` | the base accepts the key |
| L2 | `TestSignInKiloUnverifiedKeyWarns` | same | profile 503: one notice "could not verify the Kilo key", key returned | no notice at the base |
| L3 | `TestUncheckedKeyNotice` | `internal/cli/login_test.go` | `loginAsksThroughQuestions` split: the notice for `opencode-zen`, `opencode-go`, `huggingface`; none for `kilo` | the notice is given for Kilo |
| L4 | `TestSignInDevice` kilo case asserts a 401 profile refuses the device key | `signin_test.go` | edited first | accepted at the base |
| L5, L9 | `TestSignInRunKiloKeyVerified` | `signinrun_test.go` | four cases over a Kilo device-key run with `Cache` and `Headers` set in the options and a double that answers the **models listing 200 and the profile 401** (so only a profile read can refuse): profile 401 -> `ErrAuthFailure`, no save; profile 503 -> passes; the profile stalls and the context is cancelled -> `Wait` returns within about a second; a base URL that does not parse -> the sign-in fails with `ErrInvalidRequest` and saves nothing (if the run fails earlier, Phase 0 names the seam) | the 401 case passes the key at the base; the stall case waits on nothing; the fourth is not reached |
| L8 | ~~`TestAuthLoginJSONKiloUnverifiedWarnsOnStderr`~~ `TestAuthLoginJSONKiloUnverifiedKeepsStderrEmpty` (amended 2026-11-23, the owner's "Pin what happens (Recommended)"); `TestAuthLoginJSONKiloKey` and the kilo cases of the other two keep asserting empty stderr | `internal/cli/loginjson_test.go` | ~~profile 503: the SDK's warning on stderr, stdout's steps unchanged~~ profile 503: exit 0, a done step yielding `device_key`, the key stored, stderr empty; profile 401: a failed step of kind `auth`, nothing stored; the others run on an issuer whose profile answers 200 | the 401 case: the base reads no profile, so it saves the key; the 503 case is a guard, green at the base |
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
| P6 | `loginOf` drops `ClientID` | ~~`TestRefreshRefusesAnotherSignIn`,~~ `TestLoginCarriesEverySessionField` (*2026-11-23, Phase 2: `TestRefreshRefusesAnotherSignIn` cannot see it: the SDK takes a stored record holding the session's own refresh token as its own, so a store that drops the client still serves its one process (go-llmprovider-sdk CHANGELOG `[v1.32.0]`, Security); run, not caught, with the default client and with one of the test's own*) |
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
| P18 | ~~`loginOf` drops `Resource` (the plan session's marker)~~ (run, not caught: no save runs in the test); *2026-11-23, deviation 1: `sdkStore.Load` drops the stored entry (returns no session), if that turns the test red; it does, at `foreignlogin_test.go:108`* | `TestHeldPlanSessionRefusesAForeignEntry`; if it does not turn red, Phase 2 chooses the production change that does and records the deviation |
| P19 | *2026-11-23, deviation 2:* `questions.readSecret` returns the line untrimmed | `TestSignInTerminalSpacesCancel` |

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

### Phase 0, pass 1 (2026-11-23, at `f6c6b47`): stopped at its exit

**Step 1, re-ground.** Every cited line holds at `f6c6b47` except these moves, none of which changes a fact: `consentwording_test.go:131` is the doc comment, the function is at `:132`; `loginjson_test.go:480` is inside `TestAuthLoginJSONRetryable` (`:479`) and `:515` is the doc comment of `TestAuthLoginJSONDoneFollowsTheStore` (`:517`); `login_test.go:361` is the pinned notice inside `TestLoginAsksThroughQuestions` (`:325`); `login_fields_test.go:34` is `llm/provider/login_fields_test.go:34`; `internal/cli/login.go:133` is the `case`, its notice at `:134`; `plansignin_test.go:390` is the doc comment of `signInFailed` (`:394`). The held-session reproduction of 0052-PLAN's "Third review" item 3 is named `TestHeldSessionRefusesAForeignEntry` in that text, in `package provider`; this plan's V8 renames it `TestHeldPlanSessionRefusesAForeignEntry`.

**Step 2, discovery on a scratch clone** of `f6c6b47`. `go get github.com/maccavelli/go-llmprovider-sdk@v1.32.0` exit 0, `go mod tidy` exit 0; the diff is the SDK line of `go.mod` (`v1.31.0` to `v1.32.0`) and its two `go.sum` lines, nothing else (D1, S1). `go build ./...` exit 0, `go vet ./...` exit 0 (S2). `go test -count=1 ./...` through `run-on-host` (no `-race`) exit 1; `internal/cli` hit the 10-minute timeout, so `llm/provider`, `acpserver` and `internal/cli` were also run on this Mac on the clone, `internal/cli` once more with the hanging tests skipped and each of those alone under a 45 s timeout. The reds the bump causes:

| Test | Package | How it fails | Row |
| :--- | :--- | :--- | :--- |
| `TestSignInGrokBrowser` | `llm/provider` | `signin_test.go:65`: asked "Where is the browser you will sign in with?" with no answer | W6, expected |
| `TestOfferedConsentTextIsGobbles` | `llm/provider` | `policy_test.go:183`: its scripted plan sign-in captures the browser question as the consent's title | W6's input; **not in step 2's list** |
| `TestWireRootUnionSchema` | `llm/provider` | `wire_union_test.go:163`: the text now reads "llmprovider: tool schema refused" | E1, expected |
| `TestMCPUnionSchemaProviderRefusal` | `acpserver` | `mcp_test.go:527`: the same | E1, expected |
| `TestConsentWordingOnEverySurface` | `internal/cli` | hangs (timed out in its `gobble auth login` subtest) | W6 and F14, expected |
| `TestSignInChatGPTPlanTerminal`, `TestPlanAfterLogoutReusesClientTerminal` | `internal/cli` | each hangs at `plansignin_test.go:298` (the sign-in waits on the browser question) | W7, expected |
| `TestACPLoginOffersMenu/the_menu` | `internal/cli` | `acp_test.go:64`: "sign-in cancelled; nothing was saved", want declined at the consent (its scripted "2\n3\n" now meets the browser question) | W6's input; **not in step 2's list** |

Expected by step 2 and **not** red at the bump: `TestAuthLoginJSONKiloKey` and the kilo cases of `TestAuthLoginJSONRetryable` and `TestAuthLoginJSONDoneFollowsTheStore` (stderr stays empty), the kilo case of `TestSignInDevice`, `TestStartSignInChecksKiloDeviceKey` (its transport answers the profile as it answers the listing), and `TestLoginAsksThroughQuestions` (its sign-in is a test double; only the L3 change turns it red).

**Step 3, the SDK at `v1.32.0`.** The profile is `GET {origin}/api/profile` (`auth.KiloProfile`, `kiloendpoint.Route(base, "profile")`); a 200 body is `{"user":{"email":...},"organizations":[],"selectedOrganizationId":"","hasPersonalAccount":true}`. `VerifyKilo` returns `signin: verify the Kilo key: <ResolveOptions' error>`, matching `ErrInvalidRequest`, for options it rejects, before any request. A no-listener Grok sign-in names `http://127.0.0.1:<port>/callback` with a port drawn from 49152 to 65535 and never bound; a listening one binds `127.0.0.1:0` (OS-chosen); OpenAI's no-listener names `127.0.0.1:1455/auth/callback`, unbound. `refusedCredential` is still at `wizard/configure.go:884`, now through `signinflow.Refused`, the same predicate. No non-test `TextPrompter`; `NoLoopback` only at `signinrun.go:107` (K5, W8 hold).

**Why the phase stops.** A probe on the clone (a scratch test, not in the tree) ran `gobble auth login kilo --json` against `loginIssuer` with the profile answering 404, then 503: both read `GET /api/profile`, both exit 0 with stdout's four lines unchanged, and **stderr is empty both times**. Under a step driver the SDK's "could not verify the Kilo key" warning does not reach gobble's `unasked` prompter. The same sign-in through `provider.SignIn` with a scripted prompter and no driver gives that warning as a notice. So F15's premise and row L8's outcome ("the SDK's warning goes to stderr") are false, and `TestAuthLoginJSONKiloUnverifiedWarnsOnStderr` cannot be written as the row states: a row that cannot be met as written. Separately, two reds are in W6's input but not in step 2's list. Nothing in the tree has changed beyond the records; Phase 1 has not started. The stop goes to the owner through the coordinator.

#### Deviation 1 (2026-11-23): L8 and F15 amended, two reds under W6, the plant numbering

* **Found:** Phase 0 pass 1, above.
* **Decided:** the owner, relayed by the coordinator: L8 "Pin what happens (Recommended)": F15 and L8 amended in the MADR; L8's test is `TestAuthLoginJSONKiloUnverifiedKeepsStderrEmpty`, profile 503 (exit 0, a done step yielding `device_key`, the key stored, stderr empty) and a 401 case (a failed step of kind `auth`, nothing stored); Phase 1's L8 row is struck and rewritten. W6 "Yes, under W6 (Recommended)": `TestOfferedConsentTextIsGobbles` (`llm/provider/policy_test.go`) and `TestACPLoginOffersMenu/the_menu` (`internal/cli/acp_test.go`) join W6's edits, each answering index 0 at the browser question; with them Phase 0 step 2's list equals the reds seen, and the expected reds that did not turn red (listed above) stay as they are. The coordinator, from evidence: this PLAN's plant numbers are authoritative (the MADR's W4 and V8 annotated to P13 and P18); P18 becomes `sdkStore.Load` dropping the stored entry if that, on a scratch clone, turns `TestHeldPlanSessionRefusesAForeignEntry` red; otherwise it is reported.
* **Files added to the scope:** `llm/provider/policy_test.go`, `internal/cli/acp_test.go` (W6's edits only).
* **Phase 0 exit:** met with this deviation; Phase 1 starts at `f6c6b47` with the records changed.

### Phase 1, pass 1 (2026-11-23, at `f6c6b47`, the pin `v1.31.0`): stopped on row K6

Step 1a: `testIssuer` (`llm/provider/login_test.go`) and `loginIssuer` (`internal/cli/loginissuer_test.go`) serve `GET /api/profile`, 200 with a personal account by default, `profile(status)` for 401 or 503, and (`testIssuer` only) `profileStall`. Step 1b: every test of the table written; each package builds and vets at `v1.31.0`. Each test was run alone (`go test -count=1 -timeout 120s -run '^<name>$'`) at the base; the failing line of each:

| Row | Test | At `v1.31.0` |
| :--- | :--- | :--- |
| W6 | `TestSignInGrokBrowser` | red, `signin_test.go:256`: asked [], want the question once |
| W6 | `TestOfferedConsentTextIsGobbles` | red, `policy_test.go:185`: asked the consent alone |
| W6 | `TestConsentWordingOnEverySurface` | red: the `/login` subtest's wait for the question fails, and the `gobble auth login` subtest then hangs (the base reads the question's `1` as the consent's allow) to the 120 s timeout |
| W6 | `TestACPLoginOffersMenu` | red, `acp_test.go:66`: context deadline exceeded, want declined at the consent |
| W1, W2 | `TestSignInAsksWhereTheBrowserIs` | red, `signin_test.go:553`: asked [] |
| W3 | `TestSignInAnotherDeviceNeedsNoListener` | red, `:599` asked [], `:602` the browser opened once and the redirect address answered |
| W4 | `TestSignInNoPasteAsksNoBrowserQuestion` | guard, green |
| W5 | `TestSignInDeviceAsksNoBrowserQuestion` | guard, green |
| W7 | `TestSignInChatGPTPlanTerminal`, `TestPlanAfterLogoutReusesClientTerminal` | red, `plansignin_test.go:355`, `:375`, `:384`, `:403`: the browser was opened; the consent read "2" (deny) |
| W9 | `TestStartSignInPasteWithoutScheme` | red, `signinrun_test.go:1114`: the code exchanged was the whole pasted text (pass 2 of this test, below) |
| W13 | `TestSignInAnotherDeviceEmptyPastes` | red, `:673`: context deadline exceeded, want invalid_request |
| W14 | `TestSignInCancelAtBrowserQuestion` | red, `:704`, `:707`: deadline, and the browser opened |
| V1 | `TestSignInCancelledSavesNothing` | red, `:739`: a cancelled sign-in saved the session |
| V2, V3 | `TestLogins_LockRefreshFreeWithDoneContext`, `TestLogins_LockRefreshHeldWithDoneContext` | guards, green |
| V3 | `TestSignInWaitsForAHeldRefreshLock` | red, `:813` no waiting notice, `:819` saved after 0 tries of the lock (pass 2, below) |
| V4 | `TestSignInHeldLockUnasked` | red, `:834` no error, `:837` saved |
| V5 | `TestRefreshRefusesAnotherSignIn` | red, `login_test.go:683` (a 405 for the other sign-in's request, not `LoginRejected`), `:686` a request carried the other sign-in's token |
| V6 | `TestPiGrokLoginRefreshesAfterTheBump` | guard, green |
| V8 | `TestHeldPlanSessionRefusesAForeignEntry` | red at its four assertions, `foreignlogin_test.go:105`, `:108`, `:111`, `:114` |
| K1, K1a, K1b | `TestBlankEnvKeyIsUnset`, `TestBlankStoredKeyIsUnset`, `TestBlankModelsJSONKeyIsUnset` | red, `resolve_test.go:204`, `:217`, `:219`, `:228`, `:230`: resolved as a key |
| K1c, K1d, K2 | `TestResolveValue` (`llm/catalog/value_test.go:31`, `"!   "` printed nothing), `TestAuthLogin` (`internal/cli/auth_test.go:56`, `--api-key -` with spaces), `TestLFThenCRIsTwoEnds` | existing, green |
| K3 | `TestSignInKeysCountsNonBlank` | red, `:841`: no error (the base took the blank as a key) |
| K4 | `TestSignInRefusedThenBlanks` | red, `:854`: no error |
| K6 | `TestSignInTerminalSpacesAskedAgain` | red, `login_test.go:987`, and **red after the bump too** (below) |
| L1 | `TestSignInKiloWrongKey` | red, `:867`: the key accepted |
| L2 | `TestSignInKiloUnverifiedKeyWarns` | red, `:889`: no warning |
| L3 | `TestUncheckedKeyNotice`, `TestLoginAsksThroughQuestions` | red, `login_test.go:959`, `:367`: the notice given for Kilo |
| L4 | `TestSignInDevice` | red, `:304`: the device key the profile refuses accepted |
| L5, L9 | `TestSignInRunKiloKeyVerified` | red, `signinrun_test.go:1162`, `:1170`, `:1174`: refused, stalled and bad-options cases all passed the key at once |
| L8 | `TestAuthLoginJSONKiloUnverifiedKeepsStderrEmpty` | red, `loginjson_test.go:780` (the 401 case stored the key, exit 0); the three kilo `--json` tests green |
| E1 | `TestWireRootUnionSchema`, `TestMCPUnionSchemaProviderRefusal` | red, `wire_union_test.go:165`, `mcp_test.go:528`: "invalid request" |

**Pass 2 of two new tests (their own defects, not rework of a row).** `TestStartSignInPasteWithoutScheme` passed at the base: the in-process issuer takes any code, and the base read the schemeless text as a bare code; the test now requires the exchanged code to be the callback's `c1` (red at the base, above). `TestSignInWaitsForAHeldRefreshLock` released its hold 300 ms after the test began, before the device sign-in's first poll (about a second) reached the save; the hold now ends 300 ms after the save's wait begins.

**A preview, not Phase 2.** On a scratch copy of the tree, with the pin at `v1.32.0`, the production changes of steps 2 to 5 and `TestToolSchemaRefusalIsTyped`, every test above passes except `TestSignInTerminalSpacesAskedAgain`. Nothing of Phase 2 is in the tree.

**Why the pass stops: K6 cannot be met as written.** F17 says `prompter.Secret` cancels only on `""`, so spaces reach the wizard. At a terminal the secret is read in raw mode by `questions.readSecret`, which returns `strings.TrimSpace` of the line (`internal/cli/prompt.go:276`): a key of spaces is the empty key, and `prompter.Secret` cancels it (`prompt.go:69-75`). A probe on the scratch copy at `v1.32.0`: `prompter{q}.Secret` over `"   \n"` gave `"", cancelled`; `provider.SignIn("claude", ...)` over three such lines asked once (`Enter your Claude (Anthropic) API key: ***`) and returned `cancelled`, with no "no key was entered" notice. Spaces typed at gobble's own terminal never reach the SDK; they cancel the sign-in as K2's empty key does. The stop goes to the owner through the coordinator.

#### Deviation 2 (2026-11-23): K6 and F17 amended to what happens

* **Found:** Phase 1 pass 1, above (`internal/cli/prompt.go:276` and `:69-75`; the coordinator read both lines and confirmed them).
* **Decided:** the owner, relayed by the coordinator: "Record what happens (Recommended)". F17 and K6 are amended in the MADR (struck, not rewritten): at gobble's terminal a key of spaces is the empty key and cancels the sign-in, as K2's does. The test is `TestSignInTerminalSpacesCancel`, the real `prompter` over three lines of spaces: one ask, `errCancelled`, nothing saved; a guard, green before and after the bump, held by plant P19 (`readSecret` returns the line untrimmed), run on a scratch clone. Phase 1's K6 row is struck and rewritten; P19 joins the plants.
* **Files added to the scope:** none; `internal/cli/prompt.go` is a plant's target only, on a scratch clone.
* **Next:** Phase 1 pass 2 (K6's test replaced and seen green at the base), then Phase 2 and Phase 3 as written.

### Phase 1, pass 2 (2026-11-23, at `4e07a74`, the pin `v1.31.0`)

HEAD moved to `4e07a74` (the owner's commit of 0069's T8 fix; no file of this plan). `TestSignInTerminalSpacesAskedAgain` replaced by `TestSignInTerminalSpacesCancel` (deviation 2): green at the base, as a guard. Every other Phase 1 test stands as pass 1 recorded it. Phase 1's exit is met: each decided row has its test, red at the base on an assertion, or a named guard.

### Phase 2, pass 1 (2026-11-23, at `4e07a74`)

* **Step 1, the pin.** `go get github.com/maccavelli/go-llmprovider-sdk@v1.32.0` exit 0, `go mod tidy` exit 0. `go.mod`: the one SDK line `v1.31.0` to `v1.32.0`; `go.sum`: its two lines; nothing else moves (D1, S1). `go build ./...` exit 0, `go vet ./...` exit 0 (S2).
* **Steps 2 to 5, as written.** `wizardPrompter.Secret` counts a secret only when it is not blank once trimmed (K3; `secrets`' comment says so). `checkListing` calls `signin.VerifyKilo(ctx, key, "", o.sdkOptions()...)` first for a Kilo key and returns any error it gives (a refusal, the caller's ended context, or rejected options); the `v1.31.0` comments at `signin.go:172` and `signinrun.go:218` moved to `v1.32.0`, which still says both. `KeyCheckedAtSignIn` (`list.go`), true for `kilo`, and `login.go:133`'s notice for `PublicListing(id) && !KeyCheckedAtSignIn(id)`; `PublicListing` unchanged. `resolve.go`: `blank`, applied to the variable (`Credential`, `Has`), `auth.json`'s key (written blank, or resolving to blank; `Has` through `storedKeyBlank`, which runs no command), and `models.json`'s resolved value; `!command` and `--api-key -` unchanged.
* **Step 6.** `TestToolSchemaRefusalIsTyped` in `llm/provider/wire_union_test.go` (the named exception), green.
* **Step 8.** Each row's test run alone after the bump: all green (the forty-seven tests of the Phase 1 table, the guards and the existing tests named for K1c, K1d, K2, W8, W12, L7, V7).
* **Plants** through go-llmprovider-sdk's `tools/plant.py --root <this repository>`, one spec per plant, each run with its named test alone on a scratch clone of the tree:

| Plant | Test | Result |
| :--- | :--- | :--- |
| P1 | `TestSignInKeysCountsNonBlank` | caught, `signin_test.go:849`: "grok refused 4 keys" (its first spec broke the build by leaving `strings` unused, not a catch; re-run with a change that compiles) |
| P2 | `TestSignInRunKiloKeyVerified` | caught, `signinrun_test.go:1175`, `:1183`, `:1187` (refused, stalled, bad options) |
| P3 | same | caught, `:1175`, `:1187` |
| P4 | same | caught, `:1183`: canceled after 10.0 s, want about a second |
| P5 | `TestUncheckedKeyNotice` | caught, `login_test.go:959` (kilo) |
| P6 | `TestLoginCarriesEverySessionField` | caught, `login_fields_test.go:123`; `TestRefreshRefusesAnotherSignIn` does not see it (run twice, with the SDK's default client and with one of the test's own, not caught): the SDK takes a record holding the session's own refresh token as its own (CHANGELOG `[v1.32.0]`, Security), so the PLAN's row is annotated |
| P7 | `TestRefreshRefusesAnotherSignIn` | caught, `login_test.go:683` |
| P8 | `TestToolSchemaRefusalIsTyped` | caught, `wire_union_test.go:265`, `:268` |
| P9 | `TestLogins_LockRefreshFreeWithDoneContext` | caught, `logins_test.go:91` |
| P10 | `TestLogins_LockRefreshHeldWithDoneContext` | caught, `logins_test.go:115` |
| P11 | `TestSignInWaitsForAHeldRefreshLock` | caught, `signin_test.go:813`, `:819` |
| P12 | `TestSignInCancelledSavesNothing` | caught, `signin_test.go:736` |
| P13 | `TestSignInNoPasteAsksNoBrowserQuestion` | caught, `signin_test.go:79`, `:619` |
| P14 | `TestPiGrokLoginRefreshesAfterTheBump` | caught, `login_test.go:705` (no token URL at the planted issuer; nothing sent) |
| P15 | `TestSignInRunKiloKeyVerified/bad_options` | caught |
| P16a-c | `TestBlankEnvKeyIsUnset`, `TestBlankStoredKeyIsUnset`, `TestBlankModelsJSONKeyIsUnset` | caught, `resolve_test.go:204`, `:219`, `:228` |
| P17 | `TestSignInAsksWhereTheBrowserIs`; `TestSignInAnotherDeviceNeedsNoListener` | caught, each alone: `signin_test.go:553`; `:599` |
| P18 | `TestHeldPlanSessionRefusesAForeignEntry` | `loginOf` dropping `Resource`: not caught (no save runs in the test); deviation 1's `sdkStore.Load` returning no session: caught, `foreignlogin_test.go:108` |
| P19 | `TestSignInTerminalSpacesCancel` | caught, `login_test.go:983`, `:986`: "claude sign-in failed (other)", the key asked 3 times (the SDK's own blank-key rule, once the terminal no longer trims) |

The W5 guard has no plant, as the PLAN says. `make lint` first found one issue in a test of this plan (`login_test.go:980`, modernize `embedlit`), fixed in the literal; `make lint` then 0 issues in both modules; P19 re-run on the fixed test, caught.

**Gates.** Each through `run-on-host` (`pick-host` first), one at a time per host; `-race` with `--race`, so not in WSL. Pass 1 of `make preflight` failed on every host at golangci-lint on that one issue; pass 2, after the fix, is below. The other gates ran once, on trees that already held every change but that one literal.

| Gate | This Mac | The Linux machine | The Windows laptop |
| :--- | :--- | :--- | :--- |
| `make preflight` (pass 2) | 0 | 0 | ~~**not run there**~~ 0 (deviation 3) |
| `go test -count=1 ./...` | 0 | 0 | ~~**not run there**~~ 0 (deviation 3) |
| `go test -race -count=1` over `llm/provider`, `internal/auth`, `internal/cli`, `acpserver` | 0 | 0 | ~~**not run there**~~ 0, native (deviation 3) |
| `make check-test-isolation` | 0 | 0 | ~~**not run there**~~ 0 (deviation 3) |
| `go test -count=3 ./internal/cli/` | | 0 | |
| `make pre-add-check` (gofmt, golangci-lint, govulncheck) | 0, 743 files clean, "No vulnerabilities found." | | |
| `make check-records markdownlint` | 0, 0 issues | | |
| identifier scan (`denyscan.py --changed`) | 0 hits in 36 files | | |

Each preflight's `check-go-precheck` step passed, T8 included (0069's fix at `4e07a74`). **The Windows laptop ran none of them.** Each of its five runs (the four gates and preflight's pass 2) failed to copy the tree: the laptop's SSH shell parsed `run-on-host`'s copy command as Windows PowerShell ("The token '&&' is not a valid statement separator in this version"). `run-on-host` then ran the command on this Mac under its local queue, exit 0, and said so. 0069's record shows the laptop's Git Bash running preflight earlier the same day, so the laptop's state changed in between. This is an unexpected state, a stop of this pass (below).

### Phase 3, pass 1 (2026-11-23)

* **Step 1, the docs.** `README.md` and `docs/architecture.md`'s requirement table: `v1.32.0`. `docs/architecture.md`'s `llm/provider` row: the held session refuses another sign-in's entry at its next refresh (F3a, V8); `StartSignIn`'s Kilo key is checked at Kilo's profile, and rejected options fail the sign-in (D3). `docs/guides/credentials.md`: a key of only white space is no key at any source (Q2); a Kilo key, from the device or typed, is checked at Kilo's profile (L1, L2, L4); the question where the browser is, before a browser sign-in at a terminal (Q1); the held-session sentence (`:312`, F3a); spaces typed at the terminal cancel, as an empty key does (K6 as amended). `docs/guides/developing.md`: the ports paragraph removed (W7).
* **Step 2, the dated notes.** 0052-MADR O1; 0052-PLAN steps 1c and 1d; 0060-MADR D6; 0060-PLAN's "still future".
* **Step 3, the grep.** No document outside the records says a held session can still adopt an entry, or that the plan tests hold ports 1455 and 1457. `v1.31.0` stays only where a comment or a record names the version a fact was read at (`acpserver/authreal_test.go:318`, `authsignin_test.go:36`, `:384`, `:1007`, `policy_test.go:101`, `steps_test.go:327`, `loginjson_test.go:471`, and the records), or as test data (`llm/catalog/modelsjson_test.go`, `chatserver_test.go`).
* **Step 4, the MADR's Confirmation.** 1: every decided row has its test, red at the base or a named guard or the named exception, green after; every plant caught (P6 by `TestLoginCarriesEverySessionField` alone; P18 by deviation 1's change). 2: `go list -m` prints `v1.32.0`; preflight's tidy check is clean; preflight passes on this Mac, the Linux machine and (after deviation 3) the Windows laptop. 3: the "not checked" notice is at `internal/cli/login.go:134` alone, for `PublicListing && !KeyCheckedAtSignIn`. 4: the dated notes and the index row exist. MADR annotations where a test's name or place differs: V8 (`TestHeldPlanSessionRefusesAForeignEntry`, `foreignlogin_test.go`), E1 (`wire_union_test.go`), K1c and K1d (the existing tests named).
* **Not done.** The Windows gates. Nothing staged or committed: the owner stages and commits. The PLAN stays `in-progress` until the Windows gates pass and the owner approves; the `v0.8.0` tag is the owner's.

#### Deviation 3 (2026-11-23): the Windows laptop's gates had run on this Mac

* **Found:** each of the five "Windows" runs of Phase 2's gates (preflight passes 1 and 2, `go test ./...`, `-race`, `check-test-isolation`) logged `run-on-host: the copy to <the laptop> failed (ssh -o exited 1); running the command here under the local run-heavy and reporting this`, after the laptop's SSH shell refused the copy command: "The token '&&' is not a valid statement separator in this version" (Windows PowerShell). Their exit statuses are this Mac's and do not count as the laptop's.
* **Cause:** the laptop's OpenSSH default shell was Windows PowerShell, not Git Bash's `bash.exe`.
* **Decided:** the owner, relayed by the coordinator: "Restore Git Bash as default (Recommended)". The coordinator set the laptop's OpenSSH `DefaultShell` back to Git Bash's `bash.exe`; a fresh connection printed `shell=/usr/bin/bash`, MINGW64.
* **Next:** the four gates re-run on the laptop through `run-on-host --need windows` (`-race` natively, not in WSL), each counted only when its log shows the run on the laptop and no fallback line.
* **Done (2026-11-23, at `313e99e`, the owner's commit of 0069's follow-up at 15:51, with this plan's files; the first copy began at 15:51:27):** each log names the laptop as `target`, platform `windows/amd64`, and ends `remote status 0`, with no fallback line:

| Gate | Exit | It ran on the laptop |
| :--- | :--- | :--- |
| `make preflight` | 0 | the tree copied by `scp+tar`; staticcheck built as `staticcheck.exe` for `GOOS=windows`; golangci-lint "0 issues." in both modules; `check-go-precheck` ran under `C:/PROGRA~1/Git/usr/bin/bash.EXE`, T8, T8b and the second-gofmt case `ok`, PL5 and PL5b caught |
| `go test -count=1 ./...` | 0 | 38 packages `ok` |
| `go test -race -count=1` over the four touched packages | 0 | `pick-host: need=windows race=True`, natively; `llm/provider` 119 s, `internal/auth` 4 s, `internal/cli` 161 s, `acpserver` 86 s |
| `make check-test-isolation` | 0 | as above |

The Windows gates are met. Phase 2's exit (no blocker, every row tested, plants caught) and Phase 3's are met; the PLAN stays `in-progress` for the owner's approval, and nothing is staged or committed by the agent.

### 2026-11-23, close-out check at `2e4dd60`

The owner committed the 33 files as `2e4dd60`. CI run 45280487393 on `2e4dd60` passed every job on attempt 1 (validate on the three OSes, test-isolation, sbom, build, the five identity jobs). The coordinator's clean-clone gate on `313e99e` plus the 33 files exited 0 on gofmt, pre-add-check, `go test ./...`, `-race` on the four touched packages, preflight, check-test-isolation, check-records, markdownlint and denyscan.

Acceptance criteria checked against this PLAN's text:

* Phase 0 exit (citations re-grounded, reds-by-bump list equal to the MADR's expectation): met, with deviation 1.
* Phase 1 exit (every row tested, red at the base or a named guard): met, pass 2.
* Phase 2 acceptance (every decided row's test seen failing at the base and passing now; plants caught; no test edited without a row; files staged): met, per the Phase 2 record and its deviation 3.
* Phase 2 step 8 (`-count=3` run of the sign-in packages shows no flake): **not met as recorded.** The Gates bullet ran `go test -count=3 ./internal/cli/` once on Linux only. `./llm/provider/`, `./internal/auth/` and `./acpserver/` have no `-count=3` run, although step 8 names the sign-in packages and the Gates bullet names the same four packages.
* Phase 3 steps 1 to 4 (docs, dated notes, grep, MADR Confirmation 1 to 4, owner approval): met, with the owner's commit as the approval.

Status stays `in-progress`. The PLAN is not marked `complete` until step 8's `-count=3` run is recorded for the sign-in packages, or the owner rules that the internal/cli run satisfies it.

### 2026-11-23, step 8 met; 0068 complete

The coordinator ran `go test -count=3 ./llm/provider/ ./internal/auth/ ./internal/cli/ ./acpserver/` at `2e4dd60` on the session Mac through `run-heavy` (started 2026-11-23T16:55:59-06:00, status 0, elapsed 353 s): `ok llm/provider 352.321s`, `ok internal/auth 10.151s`, `ok internal/cli 210.128s`, `ok acpserver 104.220s`. The Gates bullet asked for `-count=3` on `internal/cli` only, while Phase 2 step 8 names all four sign-in packages; this run covers both readings.

Step 8 is met. Re-checked: Phase 0, Phase 1, Phase 2 acceptance and Phase 3 steps 1 to 4 are met as recorded above. Status is `complete`, 2026-11-23.
