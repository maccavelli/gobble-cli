---
status: accepted
date: 2026-11-23
decision-makers: repository owner
consulted: "the owner\'s answers of 2026-11-23 to Q1 and Q2, relayed by the coordinator (see "Owner rulings"); go-llmprovider-sdk's CHANGELOG.md `[v1.32.0]` (tag `v1.32.0`, 2026-11-22) and the records it cites (0080-MADR, 0081-MADR, 0082-MADR, 0083-MADR, 0084-MADR, 0085-MADR, 0087-MADR, each in that repository); gobble's call sites read at `84dad4a`"
informed: none
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Adopt go-llmprovider-sdk v1.32.0: what gobble must change, may use, and can ignore

Associated PLAN: [0068-PLAN-adopt-go-llmprovider-sdk-v1-32-0.md](0068-PLAN-adopt-go-llmprovider-sdk-v1-32-0.md)

## Context and Problem Statement

`go.mod:16` requires go-llmprovider-sdk `v1.31.0`. `v1.32.0` is tagged (2026-11-22). Its changelog lists eleven consumer-facing items. Four of
them change what gobble does or says whether or not gobble asks for it; the rest are options gobble may use, or code gobble does not run. This
record decides, per item and by what the code and the records say, which is which, so the pin move and what it forces are one reviewed change.

Facts read at `84dad4a` (every line is re-read by the PLAN's Phase 0):

* **F1. gobble's terminal sign-in is the SDK wizard.** `provider.SignIn` (`llm/provider/signin.go:345`) calls `wizard.ResolveCredential` with
  `Discover: true`, `KiloDeviceKey: true`, `NoPastePrompt: o.NoPaste || o.Steps != nil` and no `NoLoopback` or `NoBrowserQuestion`. `gobble auth login`
  and `/login` reach it with a real prompter (`internal/cli/auth.go:296-303`, `login.go:86`); a script with no terminal sets `NoPaste` and gets
  `unasked`; `--json` sets `Steps` (a driver). So at `v1.32.0` the new "where is the browser" question is asked exactly for a browser sign-in
  (OpenAI, Grok, the ChatGPT plan) on a terminal with paste offered, and for no other call.
* **F2. The ACP sign-in is `x/signin`, not the wizard.** `StartSignIn` (`llm/provider/signinrun.go:98-123`) sets `NoLoopback: o.Remote`
  (0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md D3's note; `TestStartSignInRemoteSetsNoLoopback`). The wizard's new option
  is a second door onto the same no-listener sign-in; gobble's ACP path already uses the first.
* **F3. 0052 deferred exactly this.** 0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md row O1 and 0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md
  (the ruling of 2026-11-17, "keep them until the SDK offers a no-listener option") keep two terminal tests, `TestSignInChatGPTPlanTerminal` and
  `TestPlanAfterLogoutReusesClientTerminal` (`internal/cli/plansignin_test.go`), binding the SDK's callback ports 1455/1457;
  `docs/guides/developing.md:87-92` says so. The option now exists, and the same ruling (0052-PLAN step 1c, 2026-11-17) says the tests "switch after the SDK release
  that has the option, a step for the phase that bumps the pin": the switch is ruled, not open. A test of those two ends by a paste on stdin
  (`terminalPlan`, `plansignin_test.go:283-294`), so none reaches the listener today.
* **F3a. 0052 also gave this pin its held-session test.** 0052-PLAN step 1d and its "Third review" (2026-11-17) say that when the SDK refuses to adopt another sign-in, gobble's step
  is "a pin bump and a test that a held session fails, sends no foreign token and stays `chatgpt_plan`", the reviewer's `TestReviewCachedSessionAdoptsAForeignEntry` adopted into the tree.
  That is 0082-MADR (go-llmprovider-sdk), in `v1.32.0`. Two statements become false at the bump: `docs/architecture.md:111` ("no refusal for it yet" in the `llm/provider` row)
  and `docs/guides/credentials.md:312` ("can still adopt one").
* **F4. 0060 deferred the typed schema error.** 0060-MADR-normalise-root-level-unions-in-mcp-tool-schemas.md D6 says a gobble record for the typed-error
  fallback is drafted "when the SDK's release is tagged ... and gobble's pin moves to that release", with two prerequisites still open (whether a history
  holding calls to a dropped tool is accepted; how a drop meets `tool_search`).
* **F5. Two gobble tests pin the old words.** `llm/provider/wire_union_test.go:159` and `acpserver/mcp_test.go:525` assert, over the real SDK Grok
  provider against a fake endpoint, `"... llmprovider: invalid request: grok HTTP 400 ..."`. `v1.32.0` begins that error "llmprovider: tool schema refused"
  (changelog, `ErrToolSchemaRefused`). `signatureRefused` (`llm/provider/replay.go:128`) still reads the error's text; `llm.Kind` has no sentinel for the new kind, so
  gobble's class stays `other` and `errors.Is(err, llmprovider.ErrInvalidRequest)` still holds (the new kind is "beneath" it).
* **F6. `internal/cli/login.go:133` tells a person that a key typed for Kilo "was not checked".** `PublicListing` includes `kilo`
  (`llm/provider/list.go:49`). With 0080-MADR (go-llmprovider-sdk) the wizard, under `Discover` and with no organization, checks a Kilo key against
  Kilo's profile and refuses it on a 401, so for Kilo that notice becomes false. `TestLoginAsksThroughQuestions` (`internal/cli/login_test.go:361`) pins it.
* **F7. gobble's ACP check is the SDK's named call site for `signin.VerifyKilo`.** `checkListing` (`llm/provider/signinrun.go:232`) skips a
  public-listing provider with no key and otherwise lists models, "as the wizard's Discover does". A Kilo device key is listed with, and Kilo's listing
  answers every credential, so the check cannot refuse it. go-llmprovider-sdk 0080-MADR D10 says "its call site (R55) is gobble's ACP check (G1), by a gobble-cli record".
* **F8. The save is now under the refresh lock.** gobble's `Logins.LockRefresh` (`internal/auth/logins.go:72`) waits up to 60 s on `filelock.Lock`, which
  tries once and then returns `ctx.Err()` when the context is done and the lock is held (`internal/filelock/lock.go:61-76`; `lock_test.go:81`). That is the
  "held" signal the SDK's `RefreshLocker` documentation names (and names gobble's locker as meeting).
* **F9. `Login` keeps every field that names a sign-in** (`Kind`, `Issuer`, `ClientID`, `AccountID`; the provider is the entry's key),
  through `Logins.Load` and `Save` and `loginOf` (`login.go:136-142`; `TestLoginCarriesEverySessionField`, `login_fields_test.go:34`). Pi's imported Grok login
  carries the SDK's issuer and client and no `Kind` (`PiLogin`, `login.go:73-83`), which the SDK infers.
* **F10. A whitespace-only key can still reach the SDK.** `internal/auth/resolve.go:95` and `:124` take an environment variable when it is `!= ""`, and `New`
  passes a non-empty `o.APIKey` to `WithAPIKey` (`provider.go:218`). At `v1.32.0` `NewStaticToken` stores such a key as `""` (0085-MADR D9): Claude, Gemini, Grok,
  Hugging Face, OpenAI and Together then refuse at `New` ("<id> api key is required", `ErrInvalidRequest`), where the service refused it before; Kilo and OpenCode go anonymous.
* **F11. gobble's `SignInError.Keys` counts asks, not refusals.** `wizardPrompter.Secret` increments on every call (`signin.go:464-470`), and `Keys` is set when the
  wizard's error matches `ErrAuthFailure` (`signin.go:372`). A blank secret is now asked again (0085-MADR D1 to D5), and a refused key followed by three blanks ends with an error that
  matches no sentinel.
* **F14. The consent test meets the question.** `TestConsentWordingOnEverySurface` (`internal/cli/consentwording_test.go:131`) runs the real wizard on five terminal surfaces
  against the plan issuer and answers the method menu, then the consent. At `v1.32.0` the browser question comes between them: setup's scripted "3" answers a two-choice menu, and
  `/login` waits for consent text that never comes.
* **F15. The Kilo device sign-in now reads `/api/profile`, and neither test issuer serves it.** Under `KiloDeviceKey` the wizard reads the profile once to check the key (changelog,
  Changed). `loginIssuer` (`internal/cli/loginissuer_test.go:89-190`) and `signInIssuer` (`llm/provider`) answer the device poll and `/api/gateway/models` only, so the profile is a 404, which
  is "cannot say" and a warning on stderr (a `Notify`; `--json` uses `unasked`, whose notices go to stderr). `TestAuthLoginJSONKiloKey` (`loginjson_test.go:135`) and the kilo cases of
  `TestAuthLoginJSONRetryable` (`:480`) and `TestAuthLoginJSONDoneFollowsTheStore` (`:515`) assert empty stderr.
* **F16. The key sources.** `--api-key -` trims and refuses blank (`credentials.go:228`); a `!command` value is trimmed and a blank output is an error (`llm/catalog/value.go:28`).
  A literal or `$VAR` value in `auth.json`'s `key` (`resolve.go:141`, `Has` `:113`) or `models.json` (`:83-91`, `:115-121`), and an environment variable (`:95`, `:124`), are used when non-empty,
  so spaces pass.
* **F17. Spaces typed at gobble's own terminal reach the SDK.** `prompter.Secret` cancels only on `""` (`prompt.go:69-75`), so `"   "` is returned to the wizard, which now notices and asks again.
* **F12. gobble does not use `TextPrompter`** (no non-test reference), the only caller of 0087's fix; its terminal input is its own (0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md).
* **F13. Release rule.** 0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md: "Below `v1.0.0` a minor bump is the release that may add or change behaviour; a patch is a fix."
  Tags exist through `v0.7.0`, which carries 0067 (commit `5962276`). SDK 0088 and 0089 (output limits) are not in `v1.32.0` and not tagged; they are 0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md and 0066-MADR-set-each-session-model-up-from-discovered-limits.md's, which need a second pin bump later.

## Decision Drivers

* A pin move must not leave a gobble statement false (F6) or a test pinning words the SDK no longer says (F5).
* Use an SDK option only where gobble's own records already want it (F3), or where an owner decision is recorded here; never change what a person sees by accident.
* The SDK's changelog says whose job each change is; gobble does not re-implement what the SDK now does (AGENTS.md: gobble waits for the library).
* Every changed behaviour has a test seen failing before the change and a plant that turns it red.
* The release number follows 0055's rule (F13), by evidence.
* The typed-error fallback has open prerequisites (F4) and stays its own record.

## Considered Options

* **A. Adopt in one record: bump the pin, fix what the bump forces (F5, F6, F11), use what gobble's records already wait for (F3, F7), and put the two real choices to the owner (Q1, Q2).** One reviewed change; the owner chooses where behaviour is a product choice.
* **B. Bump the pin and fix only what turns red.** Smallest diff; leaves F6 false in production (no test asserts the new truth), F7 unadopted, and the Kilo wrong-key and blank-key behaviours unrecorded.
* **C. Split: one record for the pin and forced fixes, another for the options.** Two reviews and two passes through the same sign-in tests; the option decisions (Q1) change the same tests the bump turns red, so they would be written twice.
* **D. Stay on `v1.31.0`.** Keeps 0082's refresh-adoption and 0081's cancelled-save defects (a refresh can send another sign-in's token on this route) and blocks 0060 D6's record.

## Decision Outcome

Chosen option: "A", because the forced changes and the optional adoptions touch the same tests and files, and the SDK's own records name gobble's ACP check and the 0052/0060 deferrals as the call sites that follow this release.

### The decisions

* **D1. The pin.** `go.mod` requires go-llmprovider-sdk `v1.32.0` (and `go.sum`). `authcontract` stays `v0.1.0` (the SDK's own `go.mod` requires `v0.1.0`);
  `golang.org/x/sys v0.49.0` and `x/term v0.47.0` are already above the SDK's `v0.47.0` and `v0.43.0`, so no other requirement moves unless `go mod tidy` says so,
  and the PLAN records its diff. No module is added; the module is already named by 0031-MADR-sdk-current-releases-and-adoption.md and the records after it.
* **D2. Must change.** (a) The two schema-refusal texts (F5) are asserted in their new words, and a test pins the new typed kind and the tool's name
  (`errors.Is(err, llmprovider.ErrToolSchemaRefused)`, `APIError.Tool`), so 0060 D6's record starts from a pinned fact. (b) The Kilo "not checked" notice is no longer given for Kilo
  (F6); a new predicate says which public-listing providers have their key checked at sign-in, and `PublicListing` and `listingState` (a listing's state) do not change.
  (c) `SignInError.Keys` counts the secrets that were not blank (F11), so "refused N keys" says what it says. (d) The held-session test of F3a is adopted, and `architecture.md:111` and `credentials.md:312` are corrected. (e) Every test the bump turns red is edited by a row, including F14 and F15: both test issuers gain a profile handler (200 by default, 401 or 503 on request).
* **D3. May use, and gobble does.** (a) `signin.VerifyKilo` joins `checkListing` for a Kilo key, with the run's SDK options (`sdkOptions`: the header-wrapped client, the base URL, the cache; they are valid by construction) and no organization of gobble's own: a 401 is a refusal as the
  wizard's is; "cannot say" passes with no notice (the ACP path has no warning channel and refuses only on a refusal, as `checkListing` documents); the caller's ended context is returned; and an
  error matching `ErrInvalidRequest` (options the SDK rejects, with no request sent) **fails the sign-in** with that error, because passing it would skip the check silently in production (F7).
  (b) The wizard's browser question is taken as the SDK asks it (Q1). (c) The two terminal plan tests switch to no listener, as 0052 ruled (F3), by answering "On another device"; no production field is added. (d) `wizard`'s `NoBrowserQuestion` and `NoLoopback` are not set by gobble.
* **D4. Can ignore, with the reason on record.** `TextPrompter`'s late-LF settle (F12); the error body's 1 MiB read (gobble maps by the SDK's kinds, so a large quota body becomes
  `llm.ErrQuota` by the same table); Together's listing header; the SDK's port-collision advice text (gobble's `SignInError` shows none of the SDK's text, `signin.go:150-157`, so
  the new advice never reaches a person and is not asserted); the typed schema-error fallback and the `signature` misfire (F4, F5), which stay 0060 D6's record, now unblocked on its pin;
  the schemeless pasted redirect (0083-MADR D11) is taken as the SDK reads it, with one test through the ACP engine.
* **D5. Takes the SDK's rules, no gobble change, each pinned.** The cancelled-sign-in save rule and the save under the refresh lock (F8), the refresh-adoption refusal (F9), and the
  blank-key refusal at the prompt (a blank secret asked again) are the SDK's; gobble's locker, store fields and prompter already meet their conditions (F8, F9, `prompter.Secret`'s empty answer cancels, `prompt.go:69-75`). Tests pin that they hold through gobble's store and prompter.
* **D6. The release.** The next release is a minor, `v0.8.0` (F13): a person sees a new question (Q1), a wrong Kilo key refused at sign-in, and a changed error text; none of that is "a fix" to
  a gobble defect. This record does not cut it; the tag is the owner's. 0067 shipped in `v0.7.0`; 0065 and 0066 wait for SDK 0088 and 0089 (untagged) and a second pin bump, so they do not share this release unless that tag exists first.

### Owner questions

Two questions were put to the owner as three options each; both are answered ("Owner rulings"). The terminal plan tests' switch is not a question: 0052 ruled it (F3).

## Consequences

* Good, because a refresh can no longer send another sign-in's token on gobble's route, a cancelled sign-in saves nothing, and a wrong Kilo key stops at sign-in.
* Good, because 0052's ports caveat (`developing.md:87-92`) can go once both plan tests switch, and 0060 D6's record can start.
* Bad, because a person at a terminal sees one more question on a browser sign-in (Q1), the one visible cost of taking the option.
* Bad, because the Kilo profile is one more request, with the SDK's own 10 s bound, on a Kilo sign-in over ACP.
* Neutral, because the exported surface gobble imports only grows; `apidiff` is a no-op below `v1.0.0` (`Makefile:177`).

### Confirmation

1. Every row of the state table that is `decided` has its test, seen failing at `v1.31.0` (the PLAN's Phase 1) and passing after the bump (Phase 2); each plant turns its test red.
2. `go list -m github.com/maccavelli/go-llmprovider-sdk` prints `v1.32.0`, `go mod tidy` leaves no diff, `make preflight` passes on the three hosts.
3. `grep -rn "not checked\|was not checked" internal/cli` finds the notice only for providers whose key is unchecked; no document says `v1.31.0` of what is now `v1.32.0`.
4. The dated notes in 0052-MADR O1 and 0060-MADR D6 exist; the index rows are in `docs/README.md`.

## State table

"Today" is `84dad4a` with the SDK at `v1.31.0`. No row is open: Q1 and Q2 were answered on 2026-11-23 ("Owner rulings").

**Pin and surface**

| Row | Input or state | Status | Outcome | Decided by / test |
| :--- | :--- | :--- | :--- | :--- |
| S1 | the pin and what `go mod tidy` does | decided | `v1.32.0`; no other requirement moves unless tidy says so; recorded | D1; gate (`go list -m`, tidy) |
| S2 | an exported name gobble uses changed or removed | decided | none: the changelog adds, and gobble builds unchanged | D1; the build at the bump |
| S3 | `apidiff`, `archtest` | out of scope | no-op below `v1.0.0` / no new import edge (`x/signin` and `wizard` are already imported by `llm/provider` only) | `Makefile:177`; archtest in preflight |

**The wizard's browser question (0083-MADR)**

| Row | Input or state | Status | Outcome | Decided by / test |
| :--- | :--- | :--- | :--- | :--- |
| W1 | terminal, browser sign-in (OpenAI, Grok, plan), paste offered | decided | asked, the SDK's question (Q1, "Yes, take the SDK's question (Recommended)", 2026-11-23) | `TestSignInAsksWhereTheBrowserIs` |
| W2 | answer "On this machine", an empty line | decided | Q1: the listener binds (the test dials the authorize URL's redirect address while the sign-in waits and connects), `OpenURL` is called | same test |
| W3 | answer "On another device" | decided | Q1: no listener (the same dial is refused; Grok's sign-in is used, which listens on an OS-chosen port, so no test holds 1455 or 1457 and the plan tests in parallel cannot collide), `OpenURL` not called, the URL shown, a paste completes it | `TestSignInAnotherDeviceNeedsNoListener` |
| W4 | `NoPaste` (no terminal) or `Steps` (`--json`, ACP-driver) | decided | no question: the SDK asks none under `NoPastePrompt` or a driver; unchanged behaviour | `TestSignInNoPasteAsksNoBrowserQuestion` (a guard: green at the base; plant P12) |
| W5 | a device or API-key method | decided | no question; this is the SDK's rule and no gobble line decides it, so the test is a guard with no plant | `TestSignInDeviceAsksNoBrowserQuestion` |
| W6 | scripted prompters of gobble's tests that meet a browser sign-in | decided | Q1: each answers the question (index 0) where it reaches it: `TestSignInGrokBrowser` (`signin_test.go:194`), and the five terminal surfaces of `TestConsentWordingOnEverySurface` (F14), whose answers become method, browser question, then consent | the tests the PLAN's Phase 0 finds red |
| W7 | the two terminal plan tests | decided | switch to no listener, as 0052 ruled (F3) and Q1 makes possible: each answers "On another device" and finishes by the paste they already use; `signInFailed` and `developing.md`'s caveat go | `TestSignInChatGPTPlanTerminal`, `TestPlanAfterLogoutReusesClientTerminal` |
| W8 | the ACP sign-in with `Remote` | decided | unchanged: `x/signin` `NoLoopback` | `TestStartSignInRemoteSetsNoLoopback` (exists) |
| W9 | a schemeless pasted redirect address in the ACP paste step | decided | taken as the address with every check; a bare host and port is refused and asked again | D4; `TestStartSignInPasteWithoutScheme` |
| W10 | input ends at, or is exhausted before, the question | out of scope | the SDK's two rules, tested in the SDK; gobble's line session maps EOF as it did (0063) | SDK 0083-MADR |
| W11 | ports 1455 and 1457 both taken: the new advice | out of scope | gobble's `SignInError` shows no SDK text; not asserted | D4 |
| W12 | the SDK's "Sign-in cancelled" / "timed out" notice beside gobble's own cancel line | decided | gobble's line (`login.go:272`) stays and is asserted once; any SDK notice is shown through `Notify` unasserted | existing `login_test.go:277`, `:878`; named in Not verified |
| W13 | "On another device", then three empty pastes | decided | the sign-in ends with an error matching `ErrInvalidRequest` (SDK changelog, Changed); gobble's text is "<id> sign-in failed (invalid_request)"; nothing saved | `TestSignInAnotherDeviceEmptyPastes` (Q1 a) |
| W14 | Ctrl+C (the prompter's cancel) at the browser question | decided | returned as the caller's own error, nothing saved, no URL opened; gobble's cancel line shows once | `TestSignInCancelAtBrowserQuestion` (Q1 a) |

**Save, cancel, refresh (0081, 0082)**

| Row | Input or state | Status | Outcome | Decided by / test |
| :--- | :--- | :--- | :--- | :--- |
| V1 | the run's context ends before the save | decided | nothing saved; the context's error returned | D5; `TestSignInCancelledSavesNothing` |
| V2 | save while no process holds the refresh lock | decided | saved; `Logins.LockRefresh` gives the lock with a done context | D5; `TestLogins_LockRefreshFreeWithDoneContext` |
| V3 | the lock is held by another process | decided | the locker answers at once with an error wrapping `context.Canceled`; the wizard says it waits, waits to 60 s, then asks to try again through gobble's `Confirm` | D5; `TestLogins_LockRefreshHeldWithDoneContext` (the real `Logins`), `TestSignInWaitsForAHeldRefreshLock` (a test locker over `memLogins`: it pins `sdkStore.LockRefresh` and the prompter, not `Logins`) |
| V4 | V3 with no terminal (`unasked`) | decided | the store's error, nothing saved | `TestSignInHeldLockUnasked` |
| V5 | a session whose store now holds another sign-in (other client or account) | decided | the refresh fails with an auth failure; gobble says `LoginRejected`'s sentence; the other sign-in's token is not sent | D5; `TestRefreshRefusesAnotherSignIn` |
| V6 | Pi's imported Grok login (no `Kind`) refreshes | decided | accepted, as the SDK infers its kind | D5; `TestPiGrokLoginRefreshesAfterTheBump` (a guard; plant P14) |
| V7 | `Logins` drops a field that names a sign-in | decided | cannot happen today; stays pinned | `TestLoginCarriesEverySessionField` (exists) |
| V8 | a held ChatGPT plan session while another process writes another application's login into `auth.json` (0052-PLAN step 1d) | decided | at its next refresh the token request fails with an auth failure, sends no foreign token, and the session stays `chatgpt_plan`; docs corrected (F3a) | D2 (d); `TestReviewCachedSessionAdoptsAForeignEntry` adopted from 0052-PLAN's "Third review"; plant P17 |

**Keys (0085, 0087)**

| Row | Input or state | Status | Outcome | Decided by / test |
| :--- | :--- | :--- | :--- | :--- |
| K1 | a whitespace-only environment variable (`resolve.go:95`, `:124`) | decided | none: treated as no key (Q2, "Treat it as no key (Recommended)", 2026-11-23); `Credential` and `Has` agree | `TestBlankEnvKeyIsUnset` |
| K1a | `auth.json` `key`, literal or `$VAR` resolving to spaces (`:141`, `Has` `:113`) | decided | treated as no key (Q2) | `TestBlankStoredKeyIsUnset` |
| K1b | `models.json` key, literal or `$VAR` resolving to spaces (`:83-91`, `:115-121`) | decided | treated as no key (Q2) | `TestBlankModelsJSONKeyIsUnset` |
| K1c | a `!command` whose output is blank | decided | an error already (`value.go:28`); unchanged | the existing `ResolveValue` tests (Phase 0 names them) |
| K1d | `--api-key -` with spaces | decided | refused already (`credentials.go:228`); unchanged | the existing `--api-key` tests (Phase 0 names them) |
| K2 | the terminal's empty secret | decided | still cancels (`prompter.Secret`) | `prompt.go:69-75`; `TestLFThenCRIsTwoEnds` (exists, `lineend_test.go:166`) |
| K3 | a whitespace-only secret from another prompter | decided | the wizard asks again, up to three asks; gobble's `Keys` counts only non-blank secrets | D2 (c); `TestSignInKeysCountsNonBlank` |
| K4 | a refused key then three blanks | decided | the error matches no sentinel: text "<id> sign-in failed (other)", `Keys` 0 | D2 (c); `TestSignInRefusedThenBlanks` |
| K5 | `TextPrompter`'s late LF | out of scope | gobble does not use it (F12) | grep, Phase 0 |
| K6 | spaces typed at gobble's own terminal (F17) | decided | reach the wizard, which notices and asks again, up to three asks, then an error with no sentinel | `TestSignInTerminalSpacesAskedAgain` over the real `prompter` |

**Kilo (0080)**

| Row | Input or state | Status | Outcome | Decided by / test |
| :--- | :--- | :--- | :--- | :--- |
| L1 | terminal, Kilo key typed, no organization, profile answers 401 | decided | refused, asked again up to three; ends `kilo refused N keys; none was saved` | D3; `TestSignInKiloWrongKey` |
| L2 | the profile cannot answer (403, 5xx, connection, stall) | decided | one warning "could not verify the Kilo key (...); going on without verifying it", the key is kept | `TestSignInKiloUnverifiedKeyWarns` |
| L3 | the "not checked" notice for a Kilo key | decided | not given for Kilo; still given for OpenCode Zen and Go and Hugging Face | D2 (b); `TestUncheckedKeyNotice` |
| L4 | Kilo device key at a terminal (`KiloDeviceKey`) | decided | the profile is read once to check the key (changelog, Changed); a 401 refuses where it was a warning | `TestSignInDevice` (kilo case, edited first) |
| L5 | Kilo device key over ACP (`SignInRun.finish`), the run's real options (Cache and Headers set) | decided | `VerifyKilo`: profile 401 refuses with `ErrAuthFailure` and nothing is saved; cannot say passes; ended context returned within about a second while the profile stalls | D3 (a); `TestSignInRunKiloKeyVerified`; the double answers the models listing 200 and the profile 401, so only `VerifyKilo` can refuse |
| L6 | a Kilo key valid only for an organization, typed with none | out of scope | the SDK's own recorded, unprobed limit; gobble sets no organization | SDK 0085-MADR |
| L7 | `listingState` for a Kilo listing | decided | unchanged: a public listing is "not checked" as a listing | `TestPublicListing` (exists) |
| L8 | `auth login kilo --json` (a driver): the profile is read, and cannot answer | decided | the SDK's warning goes to stderr (`unasked` writes there); stdout's steps are unchanged; both test issuers serve a profile, so the tests that assert empty stderr stay valid | F15; `TestAuthLoginJSONKiloKey`, the kilo cases of `TestAuthLoginJSONRetryable` and `TestAuthLoginJSONDoneFollowsTheStore` (edited: the issuers); `TestAuthLoginJSONKiloUnverifiedWarnsOnStderr` (profile 503) |
| L9 | options `VerifyKilo` rejects (`ErrInvalidRequest`, no request) | decided | the sign-in fails with that error and saves nothing; never "passes" | D3 (a); the fourth case of `TestSignInRunKiloKeyVerified` (a base URL that does not parse; if the run fails earlier, the case is the seam's, named by Phase 0) |

**Errors**

| Row | Input or state | Status | Outcome | Decided by / test |
| :--- | :--- | :--- | :--- | :--- |
| E1 | Grok's refused tool schema | decided | text begins "llmprovider: tool schema refused"; `errors.Is` `ErrToolSchemaRefused` and `ErrInvalidRequest`; `APIError.Tool` names the tool; `llm.Kind` stays `other` | D2 (a); `wire_union_test.go:159` and `mcp_test.go:525` edited first (red at the base); `TestToolSchemaRefusalIsTyped`, which names `ErrToolSchemaRefused` and `APIError.Tool`, is written after the bump (they do not exist at `v1.31.0`), the one test of this record with no red at the base; its plant is P8 |
| E2 | the typed fallback, the `signature` misfire | out of scope | 0060-MADR D6's own record, open prerequisites | F4 |
| E3 | a quota body between 64 KiB and 1 MiB | out of scope | the SDK's kind; gobble maps by the same table | D4 |
| E4 | Together's listing header | out of scope | not gobble's path | D4 |

**Release and records**

| Row | Input or state | Status | Outcome | Decided by / test |
| :--- | :--- | :--- | :--- | :--- |
| R1 | which release | decided | minor, `v0.8.0` (F13); not cut here | D6 |
| R2 | pin statements and now-false statements in docs and comments | decided | `architecture.md:200`, `README.md:175` moved; `architecture.md:111` and `credentials.md:312` (F3a) rewritten; a `v1.31.0` comment is re-read, and moved only where `v1.32.0` still says it | Phase 3 grep |
| R3 | records that wait for this | decided | dated notes in 0052-MADR O1 and 0052-PLAN steps 1c and 1d, and 0060-MADR D6 | Phase 3 |
| R4 | SDK 0088 and 0089 | out of scope | not in `v1.32.0`; 0065 and 0066's | F13 |

Counts: 3 + 14 + 8 + 10 + 9 + 4 + 4 = 52 rows; 43 decided, 9 out of scope (S3, W10, W11, K5, L6, E2, E3, E4, R4), 0 open.

## Owner rulings

*(2026-11-23, the owner's answers, relayed by the coordinator; the labels are exact.)*

* **Q1, "Yes, take the SDK's question (Recommended)".** A browser sign-in at a terminal with paste offered asks where the browser is, as the SDK does; "On another device" binds no listener and a paste finishes it. Decides W1, W2, W3, W6; W7 follows. The other options were: never ask (`NoBrowserQuestion`), and gobble sets no-listener itself when it detects a remote shell.
* **Q2, "Treat it as no key (Recommended)".** A key of only spaces from an environment variable, `auth.json` or `models.json` is no key, at every source, like an empty one. Decides K1, K1a, K1b. The other options were: leave it to the SDK, and refuse it with gobble's own message naming the source.

No question remains open, and the record is `accepted` (2026-11-23) by the state-table rule.

## Pros and Cons of the Options

### A. One record

* Good, because the forced fixes and the options share tests (F5, F6, W6).
* Bad, because it is larger than a pin bump, and three owner choices gate it.

### B. Bump and fix only the reds

* Good, because it is the smallest diff.
* Bad, because the Kilo notice stays false (F6) and no test would say so.

### C. Split

* Good, because each review is smaller.
* Bad, because the sign-in tests change twice.

### D. Stay on v1.31.0

* Good, because nothing changes.
* Bad, because a refresh can adopt another sign-in's session and 0060's follow-up waits.

## More Information

### Relation to other records

* 0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md O1 and 0052-PLAN's steps 1c and 1d: the no-listener option and the held-session test it deferred to this pin bump (a dated note, R3).
* 0060-MADR-normalise-root-level-unions-in-mcp-tool-schemas.md D6: its precondition, the pin, is met by this record's build; its other prerequisites stay open.
* 0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md: the ACP path (F2) is unchanged.
* 0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md: the version rule (F13).
* 0065-MADR-output-limit-ownership-between-gobble-and-the-sdk.md, 0066-MADR-set-each-session-model-up-from-discovered-limits.md: SDK 0088 and 0089, not in `v1.32.0`; a second pin bump.

### Not verified

* The SDK's profile URL (`GET {origin}{prefix}/api/profile`) and the body a fake endpoint must send for 200, and what redirect address a no-listener sign-in names (W3's proof dials it), are read from the SDK in the PLAN's Phase 0; no live call is made, ever, by this record.
* Whether the SDK's "Sign-in cancelled" notice appears beside gobble's own cancel line (W12) is read from a test's output, not a live run.
* Which existing tests the bump turns red beyond those named here: the PLAN's Phase 0 lists them on a scratch clone before any file of the tree changes; a red outside this record's rows is a deviation.
