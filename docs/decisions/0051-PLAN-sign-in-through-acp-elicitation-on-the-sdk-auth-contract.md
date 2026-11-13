---
status: in-progress
date: 2026-11-13
associated-madr: "0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement signing in through ACP elicitation on the SDK's auth contract

Associated MADR: [0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)

## Goal

gobble reads every sign-in fact from go-llmprovider-sdk through one policy filter (MADR D2); ACP `authenticate` signs
a client in through request-scoped elicitation when it offers one (D4); `gobble auth login --json` prints the auth
contract's steps and nothing else (D5); the terminals render the same steps through the wizard (D3); one error
vocabulary runs from the SDK to the wire (D6); acpext's doc is true and its next tag carries what D6 and D7 add; #16 is
closed (D8). The plan is complete when the MADR's Confirmation 1 to 8 is met.

This record is `proposed`. Nothing below starts before the owner accepts the MADR and approves this PLAN, whole or by
phase. The owner answered QB to QE on 2026-11-10, each as recommended: QB (a), QC (i), QD (a) and QE (a); and the
same day O2 (a), put by magic-cli-remote `0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`: `authenticate` takes `_meta.gobble.fresh` (MADR D4); and that record's O4, form only: Magic
advertises `elicitation.form` alone, so it receives D4's right-hand column, with `open_url` and `show_code`
acknowledged, not answered.
*(2026-11-11: the owner accepted the MADR that day, and this PLAN is `in-progress`. The owner's words, "accept and
approve 0051 and 0052 when required", were confirmed to the cross-repository coordinator as: start the phases that
do not wait on the SDK, which this plan's rollout names as P0, P1 and 0052-PLAN P1. P0 is done (its record, under
"Execution record"); it starts no other phase.)*
*(2026-11-12; MADR D9.)* And gobble names itself on every request it sends to an identity provider that the SDK lets
it name: its sign-ins, its refreshes, its revocations. The new phase P2a does it, by the owner's rulings of that day
on the MADR's Q1 to Q3; it is part of this PLAN, after P2 on its own commit.

## Scope

**Base.** ~~`main` at `86ad4a7` (0049-PLAN P5b-1 and the 0031 supersession landed on 2026-11-11), with a clean tree.~~
*(2026-11-11, restated.)* `main` at `17d1605`, level with `origin/main`: six commits of 0053 past `86ad4a7`
(`4fc6ef4`, `6bcb898`, `957a943`, `8a7262b`, `ac757a0`, `17d1605`). They touch `acpserver/testmain_test.go`,
`internal/cli/testmain_test.go`, `Makefile` and `.github/workflows/ci.yml`, among others; of the files this plan
lists in scope they touch only `docs/architecture.md` and `docs/README.md`. The tree holds six untracked records and
nothing else: this pair, the 0052 pair and the 0054 pair. `main`'s CI is red at `17d1605`
([0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md), F1), and
[0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md](0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md)
lands before this plan's P2: it moves the `go` line to `1.27.4`, acp-go-sdk2 to `v1.36.0` and go-selfupdate-lib to
`v1.26.2`. The base each phase re-reads (C1) is `main` as that plan leaves it.
The owner's 0031-PLAN P4b-2 work is archived on the ~~local~~ branch `wip/0031-p4b2-chatgpt-plan` (`1406bd6`)
*(2026-11-11: also on origin since that day's push; `git ls-remote --heads origin` lists it at `1406bd6`)*, by
[0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
Deviation 1. "[WIP]" lines are read there, with `git show 1406bd6:<path>`; where this plan re-applies a part of that
work, the branch is its source.

**In scope:**

* `go.mod`, `go.sum`: go-llmprovider-sdk's minor (P2), `github.com/maccavelli/go-llmprovider-sdk/authcontract` made
  direct with its first import and acpext `v0.5.0` (P4).
* `acpserver/prompt.go` and `acpserver/errordata_test.go`'s `weeklyErr` case and `prompt-weekly.error.json` frame:
  0049-PLAN P5b-2, the wire half the owner's QA puts in this plan's P4 (0052-PLAN P1 step 5 takes it out of the tree
  until then).
* `internal/archtest/check.go`, `check_test.go`: rule 4's exception (MADR D1).
* `scripts/check_libs.py`: the nested module's tags.
* `acpext/auth.go`, `acpext/types.go`, `acpext/auth_test.go`, `acpext/types_test.go`, acpext's frames under
  `acpext/testdata/`: the doc (A1), `AuthenticateMeta` and the key names (D7), the kind values (D6, QB).
* `llm/kind.go`, `llm/kind_test.go`; `llm/provider/policy.go` (new), `signin.go`, `login.go`, `list.go`,
  `provider.go` (the retry's cause), `signinrun.go` (new: `StartSignIn`, `SignInRun`), and their tests.
* `internal/auth/resolve.go`, `resolve_test.go`: `Accepts`.
* `internal/cli/auth.go`, `login.go`, `acp.go`, `agent.go`, `credentials.go`, `providerscmd.go`, `piimport.go`,
  `models.go` (log kinds), their tests, `internal/cli/acpext_test.go`.
* `acpserver/auth.go`, `acpserver/authsignin.go` (new: the relay), `acpserver/auth_test.go`,
  `acpserver/errordata_test.go`, `acpserver/testdata/authenticate-elicitation.golden` (new).
* Docs: `docs/guides/credentials.md` (`--json`, ACP sign-in), `docs/architecture.md` (the rows of `llm/provider`,
  `acpserver`, `acpext`, the new requirement), `README.md`'s sign-in section, `docs/README.md` (the two rows).
* Records: this pair; the amendments the MADR lists (P0).
* *(2026-11-11, added to the scope; MADR D6's note of that date.)* The retry cause's other holders, which the
  respelling of P3 and P4 does not reach as listed above:
  * `agent/event.go` (`RetryEmptyReply`'s value) and `internal/cli/view.go` (`retryLine`'s two comparisons), with
    `acpclient/update.go`'s doc of `Retry` (P4 step 4b);
  * `acpext/roundtrip_test.go`, whose `TestKindsAndLimits` (`:222-250`) lists the old spellings (P3). The
    `acpext/types_test.go` named above does not exist today: P3 creates it, or puts its tests in this file;
  * the tests and frames that hold an old word as an expected value, each changed with its value and named in the
    record: `acpserver/retry_test.go` (`:122-123`, `:194`), `acpserver/prompt_test.go` (`:467`),
    `acpclient/retry_test.go` (`:80-82`), `internal/cli/retry_test.go` (`:22-24`), `llm/provider/limit_test.go`
    (`:159-160`), and `acpext/testdata/retry.start.notice.json`, `retry.start.provider.notice.json` and
    `prompt-rate-over-cap.error.json`.
* *(2026-11-11, later; MADR D2, D3, D4 and D7, notes of that date.)* The remote sign-in and the consent member:
  `acpext/auth.go`'s `AuthenticateMeta.Remote` and `acpext/types.go`'s `ErrorData.Consent`, with `acpext/doc.go`,
  their frames and `acpext/roundtrip_test.go`'s `TestErrorDataSortedMembers` (P3); `llm/provider/policy.go`'s `Remote`
  (P4); `llm/provider/signinrun.go`'s `SignInOptions.Remote` and the relay's remote path in `acpserver/authsignin.go`
  (P6). Where `ErrorData.Consent` is set is
  [0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
  P3's.
* *(2026-11-12, added; MADR D9.)* gobble's name past the sign-in (P2a): `llm/provider/signin.go` (the wizard's
  `Identity`), `model.go` (`ModelInfoWith`, `lookupModelWith`), `login.go` (`LoginOptions`, `Logout`) and their tests; `internal/cli/auth.go` (the logout's options) and `internal/cli/agent.go` (the `ModelInfo` hook), with
  their tests.
* *(2026-11-13, added; the owner's ruling of that day, 0052-PLAN's "P2 split into P2-1 and P2-2 (2026-11-13)".)* 0052-PLAN P2's
  step 4, moved to P6 step 1a: `llm/provider/login.go`'s `sdkStore` as go-llmprovider-sdk's `auth.ClientStore` and
  gobble's `ClientStore`; `llm/provider/signin.go`'s `SignInOptions.HostID` and `wizard.Options.HostID`; and
  `TestClientStoreAdapter`.

**Out of scope:** the ChatGPT plan, its consent, host id, client and migration (0052); go-llmprovider-sdk's code and
records (the owner's, in that repository); magic-cli-remote; `auth status --json` (0052); a release tag or push
without the owner's word; posting on #16 before P8.
*(2026-11-13: but for 0052-PLAN P2's step 4, the host id's and the issued client's plumbing, which P6 step 1a now
does.)*

## Conventions every phase follows

* **C1. Start state.** Each phase re-reads `git status`, `HEAD`, `origin/main`, and the versions `go.mod` requires. A
  difference from this plan is a deviation.
* **C2. Seen failing.** Every new test, assertion and gate is seen failing on a scratch copy of the tree, never in the
  working tree: on each planted break the phase names, and, where it compiles, on the tree before the phase. The
  record quotes each failure, temporary paths as `<TMP>`.
* **C3. Evidence.** Commands write to a scratch log, exit status taken before any filter. Scratch runs set `HOME`,
  `USERPROFILE`, `XDG_*` and `GOBBLE_HOME` into the scratch directory, and the proxies at `127.0.0.1:9`, except where
  0049-PLAN C3 exempts `llm` and `llm/provider` (`GOPROXY=off`).
* **C4. Deviations stop the phase,** recorded here with a date, and in the MADR when a decision or fact changes.
* **C5. No real effects.** Tests stay on loopback: `llm/provider`'s `testIssuer` and `planIssuer`, go-llmprovider-sdk's
  `sdktest` fakes, `acptest.Connect`. Live sign-ins are P8's, owner-run.
* **C6. Gates and commits.** Every changed Go file passes `make pre-add-check FILES=…`; the tree passes `make
  preflight` and `make race`; the identifier scan of the diff finds nothing. The Linux machine and the Windows laptop
  run `go test ./...`, `-race`, and acpext's tests on a scratch copy at P2, P4, P6 and P8. The agent stages; the owner
  commits with `git commit --no-edit`; the disclosure guard runs before any push.
  *(2026-11-11: since 0053 landed, the gates also include `make check-test-isolation`, and CI runs it as the
  `test-isolation` job. Every test this plan adds keeps to its temp root
  (`0053-MADR-keep-tests-off-host-files-above-their-temp-root.md` D4).)*
* **C7. Identifiers.** No hostname, account name, email address or real-machine path in a committed file.
* **C8. Order of the modules** (0042-MADR D5). P3's acpext change is committed, pushed and tagged `acpext/v0.5.0` by
  the owner, and served by the module proxy, before P4 requires it.

## Implementation Steps

### P0 — the records (on acceptance)

1. The MADR `accepted` with the owner's answers to QB to QE, dated; this PLAN `in-progress`.
2. Dated amendment notes, each quoting the text it amends, in:
   * `0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md` A2 and A3 (MADR D4, D5);
   * `0005-PLAN-v1-feature-scope.md` F4-3c decision 6 (D5);
   * `0004-MADR-go-module-architecture.md` rule 4, and its "Dependencies" naming `authcontract` (D1);
   * `0031-MADR-sdk-current-releases-and-adoption.md` D8 (carried out by D2);
   * `0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md` D3 (D6);
   * `0042-MADR-acpext-extension-contract-module.md`: a note of QC's answer, and of `acpext/v0.5.0`'s content.
3. `docs/README.md`: the 0051 MADR and PLAN rows (hand-written index, `scripts/check_records.py:4-6`).
4. `make check-records markdownlint`: exit 0.

Acceptance: every note quotes what it amends; `check-records` passes.

### P1 — acpext's doc, now (MADR D7, report A1)

1. `acpext/auth.go:3-8`: `AuthProvider`'s doc says what is true: `ID` is the agent auth method's id; `Command` runs the
   provider's bare sign-in, or reads a key from stdin; `Login` means "has a way to sign in other than an API key",
   never "a device code"; a client reads each method's facts from the agent's `_meta.gobble.methods` once connected.
2. No code, no test change, no tag: the doc reaches pkg.go.dev with `acpext/v0.5.0` (P3).
3. `make pre-add-check FILES="acpext/auth.go"`, `make check-acpext-deps`.

Acceptance: `go -C acpext test -count=1 ./...` passes unchanged; `git diff` touches `acpext/auth.go`'s comment only.

### Gate before P2: the SDK's tags

`go list -m github.com/maccavelli/go-llmprovider-sdk/authcontract@v0.1.0` and `go list -m
github.com/maccavelli/go-llmprovider-sdk@<the minor carrying 0074 and 0075>` both succeed through the proxy, run from a
scratch directory. Until then P2 waits; P1 and 0052-PLAN P1 do not.
*(2026-11-13: nor does 0052-PLAN P2-1, which is built on `v1.29.0` with no SDK move; 0052-PLAN's "P2 split into P2-1 and P2-2 (2026-11-13)".)*

### P2 — the pins and the boundary (MADR D1)

1. `go get github.com/maccavelli/go-llmprovider-sdk@<minor>`, `go mod tidy`. gobble imports no `authcontract` package
   yet, so it is not required directly here (`AGENTS.md` "Dependencies": a requirement is added in the commit of its
   first import, and `go mod tidy` is clean at every commit); the SDK minor requires it, so `tidy` lists
   `github.com/maccavelli/go-llmprovider-sdk/authcontract v0.1.0 // indirect`. A requirement that moves besides these
   two is a deviation. P4 makes it direct.
   *(2026-11-11: "these two" is read against the base 0054-PLAN leaves. Before this phase,
   `0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md` has moved the `go` line to `1.27.4`, acp-go-sdk2 from `v1.35.0` to
   `v1.36.0` and go-selfupdate-lib from `v1.26.0` to `v1.26.2`. Those three are that record's, are already in `go.mod`
   when this step runs, and are not this step's deviation. Anything else that moves here is, the `go` line included,
   should the SDK's minor carry a floor above `1.27.4`.)*
2. `internal/archtest/check.go`: `authcontractPath = "github.com/maccavelli/go-llmprovider-sdk/authcontract"`; rule 4
   (`:308`) lets `acpserver` and `internal/cli` import it besides `llm/provider`, and no other package; the rest of the
   SDK stays `llm/provider`'s. Rule 1 (`:249`) is not edited: it runs only in its roots (`:182-186`, `inRule1`), where
   its SDK case already admits `llm/provider` alone and refuses `authcontract` in `agent` and the other core packages
   (MADR D1).
3. `internal/archtest/check_test.go`: `TestImportRules` gains three cases. **Planted breaks:** `agent` importing
   `authcontract` (rule 1); `acpclient` importing it (rule 4); `acpserver` importing `llmprovider/x/signin` (rule 4).
   Each is a violation; the three allowed imports are not.
4. `scripts/check_libs.py`: `Library` gains `tag_prefix` and a `name`, the `--lib` choice, which defaults to `repo`
   (`--lib`'s choices are `lib.repo` today, `scripts/check_libs.py:399`, so two entries of one repository would
   share a choice); a `Library("go-llmprovider-sdk", …, path=…/authcontract, tag_prefix="authcontract/",
   name="authcontract")` entry reads `authcontract/vX.Y.Z` tags only. `--self-test` gains two cases: a tag list
   `v1.40.0`, `authcontract/v0.2.0` picks `v0.2.0` for the contract and `v1.40.0` for the SDK; and the `--lib` choices
   are unique and include `go-llmprovider-sdk` and `authcontract`. **Planted breaks:** the prefix ignored, so the
   contract's latest reads `v1.40.0`; the contract entry given no `name`, so two choices are `go-llmprovider-sdk`.
5. `python3 scripts/check_libs.py --lib go-llmprovider-sdk` and `--lib authcontract` (each needs the network): 0
   failing. go-llmprovider-sdk
   deprecates `AuthMethod.Interactive` and `HeadlessOK` (go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D7); gobble reads neither (`grep -rn
   'HeadlessOK\|\.Interactive\b' --include=*.go` found nothing on 2026-11-10), so `make staticcheck` stays clean.
6. Gates (C6).

Acceptance: archtest and `check_libs.py --self-test` pass, each plant failed; `make preflight` passes with no code
change but `internal/archtest/check.go`, its test and `scripts/check_libs.py`; `go.mod` names the contract
`// indirect` only.

### P2a — gobble's name past the sign-in (MADR D9; added 2026-11-12, the owner's)

*Added on 2026-11-12 by MADR D9, on the owner's direction of that day: "Where possible gobble auth would send gobble as
the name/referrer not the sdk". The owner ruled the same day on the MADR's Q1, "New phase P2a": this phase is part of
this PLAN, `in-progress`, and lands after P2, on its own commit, before 0052-PLAN P2. Like every phase after P1, it
waits for the gate before P2, the SDK's tags. It was drafted and ruled before any commit, so its text is the ruled
one: by the ruling on Q2, "Amend SDK 0079", gobble calls no `UseIdentity` for the model facts.*

*(2026-11-13: "before 0052-PLAN P2" reads "before 0052-PLAN P2-2". The owner split that phase that day, and its
P2-1 needs no SDK move, so it may land before P2; 0052-PLAN's "P2 split into P2-1 and P2-2 (2026-11-13)".)*

Prerequisites: P2 committed, so `go.mod` requires go-llmprovider-sdk `v1.31.0`, which carries go-llmprovider-sdk
`0079-PLAN-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md`: `(*auth.OAuthSession).UseIdentity`, the
session's identity on a refresh and a revocation, and the providers' and listings' hand-offs. For N6 it must also
carry that repository's pending amendment of
`0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md` (the owner's ruling on Q2), which puts
`catalog.CachedModel` and `LookupModel`'s `listingModel` through the hand-off. This phase moves no requirement.

0. **The sites at the phase's base** (C1). MADR D9's N1 to N11 are read again at P2's commit; a line that moved is
   recorded with its new place, and a site that changed in substance is a deviation (C4). go-llmprovider-sdk's
   `wizard.Options.Identity`, `UseIdentity` and `catalog.CachedModel` are read at `v1.31.0` in the module cache, and
   the record quotes the line where `CachedModel` hands `WithClientInfo`'s name to the session before it reads the
   token. If `v1.31.0` has no such line, the SDK's amendment did not ship in it: that is a deviation (C4), and gobble
   does not name the session itself in its place.
1. **N1, N3:** `llm/provider/signin.go`: an unexported `identityOf(o Options) auth.Identity`, `{Name: o.ClientName,
   Version: o.ClientVersion}`; `SignIn` sets `wizard.Options.Identity` to it. P6's `StartSignIn` takes
   `signin.Options.Identity` from the same helper (P6 step 1's note).
2. **N6:** `llm/provider/model.go`: `ModelInfoWith` passes `o.ClientName` and `o.ClientVersion` on to
   `lookupModelWith`, which adds `llmprovider.WithClientInfo(o.ClientName, o.ClientVersion)` to the options of its
   `catalog.CachedModel` call (`:150-157`) when `ClientName` is set; the doc of each names the two fields. No
   `UseIdentity` is called: the SDK's hand-off names the session from that option. `internal/cli/agent.go:102`: the
   `ModelInfo` hook's `provider.Options` gains `ClientName: appName, ClientVersion: identity().Version`.
   `Options.login` (`provider.go:98-112`) is not changed (N4, N5 are the SDK's).
3. **N7:** `llm/provider/login.go`: `LoginOptions` gains `ClientName, ClientVersion string`, documented as
   `Options`' are; `Logout` calls `UseIdentity` with them on the session it builds, before `RevokeOAuthSession`.
   `internal/cli/auth.go:302` passes `appName` and `identity().Version`.
4. Tests, each seen failing (C2), every capturing transport given as the options' `HTTPClient`:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestSignInNamesGobble` (`llm/provider`) | grok's device sign-in through `SignIn` against `testIssuer`, with `ClientName` "gobble" and `ClientVersion` "1.2.3-test": the device request's form carries `referrer=gobble`; every request of the sign-in has a User-Agent that begins `gobble/1.2.3-test (` and names `go-llmprovider-sdk/`. Grok's browser sign-in, its URL taken from the `OpenURL` hook and the sign-in then cancelled: the authorize URL carries `referrer=gobble` | `Identity` left out of `wizard.Options`; `identityOf` giving an empty `Version`, so the User-Agent names another version |
   | `TestStoredLoginNamedForRefresh` (`llm/provider`) | a stored grok login past its expiry, with `ClientName` "gobble" and `ClientVersion` "1.2.3-test": the refresh that `New`'s first request, `List`, and `ModelInfoWith` each start sends a User-Agent that begins `gobble/1.2.3-test (`; `ModelInfoWith` runs first on a fresh `Sessions`, with no provider built before it, so only the SDK's hand-off from the `WithClientInfo` gobble passes can name its refresh; the background listing it starts is named too; with `ClientName` empty, each is the SDK's own User-Agent | `WithClientInfo` left out of `lookupModelWith`'s options, so `ModelInfoWith`'s refresh and its background listing name the SDK (`New`'s and `List`'s stay named, and the record says so); `ModelInfoWith` not passing `ClientName` on to `lookupModelWith` |
   | `TestLogoutRevokesAsGobble` (`llm/provider`) | `Logout` of a stored grok login against `testIssuer`, with `LoginOptions{ClientName: "gobble", ClientVersion: "1.2.3-test"}`: the discovery and the revocation each send a User-Agent that begins `gobble/1.2.3-test (` and names `go-llmprovider-sdk/`; their URLs and bodies equal those of the same logout with no name, which sends the SDK's User-Agent | `UseIdentity` removed; `UseIdentity` called after `RevokeOAuthSession` |
   | `TestCLIPassesGobblesIdentity` (`internal/cli`) | with `signInWith`, `logoutLogin` and `modelFacts` replaced by recorders: `auth login grok`, `auth logout grok` and the agent's `ModelInfo` hook each pass `ClientName` "gobble" and `ClientVersion` equal to `identity().Version` | `internal/cli/auth.go:302` passing `provider.LoginOptions{}`; the hook built without `ClientName` |

   No test asserts an `originator`: gobble sends none (MADR D9, N11). A failure of `TestStoredLoginNamedForRefresh`'s
   `ModelInfoWith` case with no plant in place is the SDK's amendment missing (step 0), a deviation (C4); the case is
   not loosened, and gobble does not call `UseIdentity` to pass it. Every existing test of `llm/provider` and
   `internal/cli` passes unchanged; one that needs a change is named in the record with why, or is a deviation (C4).
5. Gates (C6), with `make check-test-isolation`. Heavy runs go through
   `~/.agents/skills/available-dev-hosts/scripts/run-heavy` or `run-on-host`, one at a time per host.

Acceptance: the tests pass, each plant failed; outside tests, `RevokeOAuthSession` is called only in `Logout`, after
`UseIdentity`, and `UseIdentity` is called nowhere else; `go.mod` and `go.sum` unchanged from P2. The commit is the
owner's, its own, after P2's.

### P3 — acpext `v0.5.0` (MADR D6, D7; QB (a), QC (i) as recommended)

*If the owner answers QB (b), step 2 is not done; if QC (ii) or (iii), this phase is rewritten by amendment before it
starts.*

1. `acpext/auth.go`: `AuthenticateMeta{Method string; Fresh bool}`, its members' JSON names `method` and `fresh`, each
   with `omitzero`, the value of `_meta.gobble` on `authenticate` (MADR D4, D7; the owner's O2); constants `MetaMethods = "methods"` and `MetaStep = "step"`, the members of `_meta.gobble` on an
   agent auth method and on an elicitation, whose values are `[]authcontract.Method` and `authcontract.Step` (named in
   the doc, never imported). `AuthProvider` unchanged.
   *(2026-11-11, later: and `Remote bool`, JSON `remote`, `omitzero`, for `_meta.gobble.remote`: `AuthenticateMeta{Method
   string; Fresh bool; Remote bool}` (MADR D4 step 1, D7, notes of that date).)*
2. `acpext/types.go`: `KindRateLimited = "rate_limited"`, `KindNotPermitted = "not_permitted"`, `KindContextOverflow =
   "context_overflow"`, `KindEmptyReply = "empty_answer"`; `RetryRateLimited`, `RetryNotPermitted`, `RetryEmptyReply`
   the same; `RetryEmpty` removed (its source, `FailureKind`'s `empty`, becomes `other`, MADR D6). Four new kinds a
   sign-in's `auth_required` can carry (MADR D4, D6): `KindDenied = "denied"`, `KindExpired = "expired"`,
   `KindInvalidRequest = "invalid_request"`, `KindInvalidProvider = "invalid_provider"`, so the kinds are the
   contract's eighteen. The doc names the contract's `ErrorKind` as the vocabulary.
   *(2026-11-11, later; MADR D7's note.)* `ErrorData` gains `Consent string`, JSON `consent`, `omitzero`, between
   `Command` and `Error` in the sorted order: set with `Kind` `not_permitted` when a request is refused because a
   consent is not granted, its value the consent's id (`chatgpt_plan.background_use` for the ChatGPT plan), and never
   for a request refused because the stored answer could not be read. The type's doc and `acpext/doc.go`'s
   failed-turn paragraph say when it is set.
3. Frames under `acpext/testdata/` holding a changed value are rewritten, and one new frame,
   `authenticate-meta.request.json`.
   *(2026-11-11, later: that frame carries `method`, `fresh` and `remote`; and a second new frame,
   `prompt-plan-consent.error.json`, a turn refused with `kind` `not_permitted`, `consent`
   `chatgpt_plan.background_use` and `provider` `openai`, joins the round-trip and coverage tests.)*
   *(2026-11-11: two things this phase also carries. `acpext/roundtrip_test.go`'s `TestKindsAndLimits` (`:222-250`)
   lists the fourteen old spellings and is rewritten to step 2's values, named in the record with its diff. And
   `acpext/go.mod`'s `go` line is `1.27.4` at this tag, moved by `0054-MADR-go-1-27-4-and-sibling-pin-refresh.md` D2
   with no tag of its own: it is no API change, so step 5's apidiff does not list it, and the tag's notes name it as
   the module's new floor.)*
4. Tests: `TestAuthenticateMetaRoundTrip` (strict decode, unknown member refused as the module's other types; `fresh`
   round-trips, and a non-boolean `fresh` is refused);
   `TestKindsAreTheContractsSpelling` (every `RetryCause` value is `snake_case` and in a list copied from
   go-llmprovider-sdk `0073-MADR-publish-an-auth-contract-module.md` D3's table, and the `ErrorKind` constants are
   exactly that table's eighteen, since acpext imports no SDK). **Planted breaks:** `KindRateLimited` left
   `rate-limited`; `KindDenied` removed; `AuthenticateMeta.Method` tagged `methodId`; `AuthenticateMeta.Fresh` tagged
   `force`.
   *(2026-11-11, later.)* `TestAuthenticateMetaRoundTrip` also: `remote` round-trips, and a non-boolean `remote` is
   refused. `TestErrorDataSortedMembers` (`acpext/roundtrip_test.go:203`) lists `consent` between `command` and
   `error`, and `prompt-plan-consent.error.json` decodes strictly into `ErrorData` with `Consent` set. **Planted
   breaks:** `AuthenticateMeta.Remote` tagged `isRemote`; `ErrorData.Consent` declared after `Error`, so the sorted
   order fails; `Consent` tagged `consentId`, so the frame's strict decode fails.
5. apidiff from `acpext/v0.4.0`, as 0049-PLAN P1 step 5 ran it (`0049-PLAN-…md:166`), on export data of `git archive
   acpext/v0.4.0` and the tree's `acpext/`: incompatible changes are exactly the constant values of step 2 and
   `RetryEmpty`'s removal; compatible, the additions of step 1 and step 2's four new kinds. A wire change named here
   (0042-MADR D5).
   *(2026-11-11, later: the compatible list also holds `AuthenticateMeta.Remote` and `ErrorData.Consent`; nothing
   else moves.)*
6. `make check-acpext-deps`: `ok`. acpext's tests on the Linux machine and the Windows laptop.
7. **The owner's:** commit, push, annotated tag `acpext/v0.5.0` on that commit, push; then `go list -m
   github.com/maccavelli/gobble-cli/acpext@v0.5.0` succeeds through the proxy.

Acceptance: the tests pass, each plant failed; apidiff as step 5; the tag served.

### P4 — the policy, the resolver and the vocabulary in the root (MADR D2, D6)

Prerequisites: P2 and P3's tag; 0052-PLAN P1 (the WIP sorted) and P2 (the session's `kind` and `consents` in
`auth.json`, so no sign-in this phase enables can lose them).
*(2026-11-13: 0052-PLAN P2 is now its P2-1 and P2-2, both before this phase; its step 4 is P6 step 1a.)*

**Committed with 0052-PLAN P3, as one commit.** `acpext/v0.5.0` keeps openai's `Login: true` and its bare command
(`acpext/auth.go:25`), so `TestAcpextAuthProvidersAreTheRegistry` holds only once openai's plan is offered, which the
consent rule allows only once 0052-PLAN P3 honours `chatgpt_plan.background_use`. No commit offers the plan without
gobble's wording and gate.

1. `go get github.com/maccavelli/gobble-cli/acpext@v0.5.0`, `go mod tidy`. With step 2's import, `tidy` makes
   `github.com/maccavelli/go-llmprovider-sdk/authcontract` a direct requirement (MADR D1): the commit of its first
   import.
   * **0049-PLAN P5b-2, the wire half (the owner's QA, 2026-11-10).** Re-applied from the archive branch
     `wip/0031-p4b2-chatgpt-plan` (`1406bd6`; 0052-PLAN Deviation 1), unchanged:
     `acpserver/prompt.go`'s `errorData` sets `WindowMinutes` to `int(ae.Window / time.Minute)` when `Window` is
     positive (`:308` [WIP]); `acpserver/errordata_test.go`'s `weeklyErr` and its two cases, `TestErrorData`'s "a
     stated weekly window" and `TestPromptQuotaData`'s `prompt-weekly.error.json` frame. Seen failing on a scratch copy
     with the `WindowMinutes` lines removed (both cases name `windowMinutes` 10080). 0049-PLAN's record gains its P5b-2
     entry, and ~~the owner's 0049 issue (#3)~~ *(2026-11-11: #15)* may close after this commit.
     *(2026-11-11: #3 is not 0049's. `gh issue view` gives #3 as "Adopt go-llmprovider-sdk v1.15.0: retry notices
     (SDK issue 3) and the ChatGPT plan sign-in (F4-3d)", #14 as the request 0049-MADR answers, and #15 as "Adopt
     go-llmprovider-sdk v1.29.0: provider reset instants and stated limits (0049-PLAN P5)"; all three are open.
     P5b-2 is the last part of 0049-PLAN P5, so #15 is the issue this commit may close, on the owner's word. Whether
     #14 and #3 then close is for 0049-PLAN's and 0052-PLAN's close-out to say.)*
2. `llm/provider/policy.go`: `acceptedBorrowed = []string{"grok"}`; `honouredConsents`, empty (0052-PLAN P3 fills it);
   `Offered(id string) []authcontract.Method` from
   `signin.Methods` and the MADR D2 table; `HasLogin`, `HasKeyLogin`, the Pi rule (`piimport.go:45`), `Revocable`, the
   bare method (`DefaultMethod(id)`) and `Login` derived from it. `offers` (`signin.go:77` [HEAD]) and the [WIP]
   `HasPiLogin` and `openai` checks of `signin.go` go. `methodIDs` stays the word table.
   *(2026-11-11, later; MADR D2's note.)* `Remote(m authcontract.Method) bool`: `Completion` `device`, or a browser
   completion with a `Paste` fact. And `Offered` sets an honoured consent's `Text` to gobble's wording, keeping its
   `Links` (0052-PLAN P3 step 1's note).
3. `internal/auth/resolve.go`: `HasLogin func(id string) bool` (`:52`) becomes `Accepts func(id string, e Entry) bool`;
   `internal/cli/credentials.go:51` passes `provider.Accepts`, true for an `oauth` entry whose session kind is one an
   offered method signs in to (the SDK's `auth.SessionKind`, inferred when the entry has none). *(2026-11-11: read by calling
   `(*OAuthSession).SessionKind()`, which the owner ruled the SDK exports, go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md`
   D5; gobble copies no inference rule. MADR D2's note of that date.)* `Logout`
   (`llm/provider/login.go:93-104` [HEAD]) revokes only an accepted, revocable session, and deletes any.
   *(2026-11-12; MADR D9, N7.)* The rewrite keeps P2a's `UseIdentity` call before the revocation, for every session it
   revokes. `TestLogoutRevokesAsGobble` (P2a) passes, and gains a case once this commit offers the plan: a stored plan
   session's revocation, and the discovery before it, send a User-Agent that begins `gobble/`. **Planted break:** the
   call moved inside a grok-only branch, so the plan session is revoked under the SDK's name.
4. `llm/kind.go`: the contract's spellings (MADR D6's table). `llm/provider`: `ErrorKind(err) string`,
   `signin.ErrorKindOf` first, then `llm.Kind`; `FailureKind` and `failureKinds` (`list.go:106-133`) go; their callers
   (`provider.go:273` [HEAD], `signin.go:138` [HEAD], `internal/cli/auth.go:84`, `:307` [HEAD], `models.go:274`,
   `:316`) use `ErrorKind`. Three tests that call `FailureKind` assert `ErrorKind` instead, each with the same word,
   since `auth` and `timeout` are spelled alike in both: `llm/provider/list_test.go:176-177` (`auth`), `:188-189`
   (`timeout`) and `llm/provider/workspace_test.go:22-23` (`workspace`); each is named in the record with its diff.
4a. `internal/cli/auth.go`'s `loginWith` (`:199-223` [HEAD]): its switch on `HasKeyLogin`, `HasLogin` and `case id ==
   "openai"` (`:213` [HEAD]) becomes `provider.DefaultMethod(id)`: a provider whose default method is none is a usage
   error naming `--api-key -`, and `--device` picks the offered device method. No per-id case remains.
4b. *(2026-11-11, added; MADR D6's note of that date.)* **The agent's own causes take the same words.**
   `_gobble/retry`'s `cause` is `acpext.RetryCause(ev.Cause)` (`acpserver/prompt.go:108`), a conversion of
   `agent.RetryCause`, so for the turn's own waits acpext's constant never reaches the wire.
   * `agent/event.go:103`: `RetryEmptyReply RetryCause = "empty_answer"`. `RetryOffline` is `offline` on both sides and
     stays. `agent` may not import acpext (archtest rule 12), so the value is a literal, held by the test below.
   * `internal/cli/view.go:247` and `:249`: `retryLine` compares `string(acpext.RetryOffline)` and
     `string(acpext.RetryEmptyReply)` in place of the two literals, so a later respelling cannot leave the CLI behind
     (`internal/cli` may import acpext, rule 12).
   * A provider retry's cause is `agent.RetryCause(pr.Kind)` (`agent/agent.go:697`), and `pr.Kind` is step 4's
     `provider.ErrorKind` word, so it moves with step 4. `retryLine` shows it with `_` as a space (`view.go:252`): the
     CLI's line reads "rate limited" where it read "rate-limited".
   * `acpclient/update.go:118`'s doc of `Retry` names the new words.
   * Not changed: the personality key `retry.empty_reply` and the Go identifiers (MADR D6's note says why).
   * The tests the scope's note of this date lists change with their values, each named in the record with its diff.
5. `internal/cli/agent.go:381-392` [HEAD]: `authProviders` sets `Login` from `provider.Login(id)`, the command from
   `DefaultMethod`.
6. `providerscmd.go:114`: `signInWords` renders the offered labels.
7. Tests, each seen failing (C2):

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestOfferedMethods` (`llm/provider`) | per provider, the offered method ids: grok `api_key`, `browser_oauth`, `device_code`; kilo `api_key`, `device_code`; openai `api_key`, `chatgpt_plan` (its consent honoured by 0052-PLAN P3, committed with this phase); the six key-only providers `api_key`; ollama none | the borrowed-client rule dropped (openai gains `browser_oauth`); `acceptedBorrowed` emptied (grok loses two); `honouredConsents` emptied (openai loses `chatgpt_plan`) |
   | `TestPredicatesFromOffered` | `HasLogin`, `HasKeyLogin`, `Login`, `Revocable`, `DefaultMethod` and the Pi rule equal what `Offered` says, for every key provider | `HasKeyLogin` reading `oauth_session` |
   | *(2026-11-11, later, added)* `TestRemoteMethods` (`llm/provider`) | per provider, the offered methods other than `api_key` that `Remote` accepts: grok `device_code`; kilo `device_code`; openai `chatgpt_plan`; grok's `browser_oauth` refused | `Remote` reading `Completion` alone, so grok's `browser_oauth` is accepted; `Remote` refusing every browser completion, so openai loses `chatgpt_plan` |
   | `TestEveryCommandAndWordIsAccepted` (`internal/cli`) | each `acpext.AuthProviders()` `Command` parses with Kong and its provider and method pass `methodOf`; each offered method's word passes `methodOf` (issue #16's criterion) | openai's word list given `device`; kilo's command given `--method browser` |
   | `TestResolverAcceptsOnlyOfferedKinds` (`internal/auth`) | an openai `oauth` entry of the ChatGPT (Codex) kind is not usable; a grok session is; a kilo `api_key` is | `Accepts` ignoring the kind |
   | `TestLogoutRevokesOnlyAccepted` (`llm/provider`) | a foreign-kind openai entry is deleted and never revoked (the `testIssuer` sees no revoke); a grok session is revoked then deleted | revocation by `HasLogin` alone |
   | `TestErrorKindIsTheContracts` (`llm/provider`) | every word `ErrorKind` and `llm.Kind` return, over every sentinel of `llm` and the SDK's, is an `authcontract.ErrorKind` constant; `denied` and `expired` from `auth.ErrSignInDenied`, `ErrSignInExpired` | `llm/kind.go` keeping `rate-limited` |
   | `TestKindsAndLimitsMatchAcpext` (`acpserver/errordata_test.go:226` [WIP], `:211` [HEAD], extended) | every `llm.Kind` value and every word `provider.ErrorKind` can return is an acpext `ErrorKind`, and every acpext `ErrorKind` an `authcontract.ErrorKind` | acpext pinned back to `v0.4.0` in the scratch copy; `KindDenied` deleted from the scratch copy's acpext |
   | `TestLoginWithReadsTheDefaultMethod` (`internal/cli`) | `auth login grok` signs in by the default offered method, `--device` by the device one; `auth login claude` is the usage error naming `--api-key -`; no message names openai | the `id == "openai"` case restored |
   | *(2026-11-11, added)* `TestRetryCausesAreAcpexts` (`acpserver`) | `string(agent.RetryOffline)` equals `string(acpext.RetryOffline)` and `string(agent.RetryEmptyReply)` equals `string(acpext.RetryEmptyReply)`; a turn that waits for an empty reply sends `_gobble/retry` with `cause` `empty_answer` (`TestRetryNotice`, its expected frames changed), and a provider retry's notice has `cause` `rate_limited` | `agent.RetryEmptyReply` left `"empty_reply"` in the scratch copy |
   | *(2026-11-11, changed)* `TestRetryLines` (`internal/cli/retry_test.go:33`, its updates respelled) | an empty reply's wait, received as `cause` `empty_answer`, shows the personality's empty-reply line, not the generic one | `retryLine` left comparing the literal `"empty_reply"`, so the line reads "retrying: empty answer …" |

   `TestKindAgreesWithFailureKind` (`llm/provider/limit_test.go:182`) and `TestFailureKind` (`list_test.go:260`) are
   removed with `FailureKind`, which MADR D6 retires; the three tests of step 4 change as it says;
   `TestAcpextAuthProvidersAreTheRegistry` passes unchanged.
8. Gates (C6), the two other hosts included.

Acceptance: the tests pass, each plant failed; `grep -rn 'FailureKind\|id == "openai"\|HasPiLogin' --include=*.go`
finds nothing outside the records. The commit is the owner's, after 0052-PLAN P3's acceptance.
*(2026-11-11: and `git grep -n '"empty_reply"\|"rate-limited"\|"not-permitted"\|"context-overflow"' -- '*.go'
'acpext/testdata'` finds nothing. `retry.empty_reply`, the personality key, stays and is not matched by it.)*

### P5 — `gobble auth login --json` prints the contract's steps (MADR D3, D5; QE (a))

1. `llm/provider/signin.go`: `SignInOptions.Steps func(authcontract.Step) error` replaces `ShowDeviceCode` and
   `OpenURL` for a non-terminal caller; `SignIn` passes `wizard.Options.Driver` = a driver that calls it and returns a
   zero answer, or, for `paste`, `confirm` and `form`, an error marked `unsupported`. `NoPastePrompt` stays set.
2. `internal/cli/auth.go`: `loginEvent` and its five events go; `jsonSteps` writes the `method` line, then each step;
   `--api-key -` writes `method` and a `done` step with `yields: "api_key"`; a usage error is one `failed` step of
   `invalid_request`. Before a sign-in starts, a method whose `Consents` hold a required one the stored session has not
   answered (0052-MADR) is refused, exit 2, a `failed` step of `unsupported` naming a terminal and an ACP client.
3. `openLoginURL`'s `--json` branch (`:251-261` [HEAD]) goes: no host browser under `--json`.
4. Tests, each seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestAuthLoginJSONSteps` | grok's device sign-in against `testIssuer`: a `method` line, `show_code` (URL, code, `expires_at`), `waiting`, `done`; each line decodes strictly into the contract's types; no `device_code`, `browser_url`, `saved`, `error` or `notice` | the old `device_code` event emitted beside the step |
   | `TestAuthLoginJSONKiloKey` | kilo's device sign-in ends `done` with `yields: "device_key"`, and `auth.json` holds the quoted key | `yields` left empty |
   | `TestAuthLoginJSONAPIKey` | `--api-key - --json`: `method` (`api_key`), `done` (`api_key`); exit 0 | the `saved` event kept |
   | `TestAuthLoginJSONUsageError` | an unknown provider: one `failed` step, `invalid_request`, exit 2 | exit 1 |
   | `TestAuthLoginJSONRefusesAQuestion` | a method whose sign-in asks a `confirm` is refused before any `open_url`, exit 2, kind `unsupported` | the refusal after the `open_url` step |
   | `TestAuthLoginJSONNoHostBrowser` | the `openURL` seam is never called under `--json` | the old `openLoginURL` branch restored |
   | `TestAuthLoginJSONMatchesVectors` | each step kind gobble emits decodes into the same Go type as the contract's golden vector of that kind, read from the module's source at the pinned tag (go-llmprovider-sdk `0073-MADR-publish-an-auth-contract-module.md` D5) | a step's `expires_at` written as `expires` |

5. `docs/guides/credentials.md`: the `--json` section rewritten for the steps, with the retired events listed.
6. Gates (C6).

Acceptance: the tests pass, each plant failed; `TestAuthLoginJSON` (`internal/cli/auth_test.go`) is rewritten for the
steps, and the rewrite is named in this record.

### P6 — ACP `authenticate` by elicitation (MADR D4)

1. `llm/provider/signinrun.go`: `StartSignIn(ctx, id, method string, o SignInOptions, store LoginStore) (*SignInRun,
   error)` over `signin.Start`, with `signin.Options` as MADR D3 lists it: `Identity: {Name: "gobble", Version}`,
   `AgentName: "gobble"`, `HostID` from `o.HostID` (0052-PLAN P2's `<state>/host-id`), `ClientStore` the `sdkStore`
   adapter when `store` has a client store (0052-PLAN P2 step 4; nil otherwise, which `x/signin` reads as none),
   `ConsentText` gobble's wording (0052-PLAN P3), and `Previous` the session `store` holds for `id` when its kind is the
   one the method signs in to. `Wait` returns `provider.SignInResult{Key string; Yields authcontract.Yields}` (MADR
   D3): it saves the session through `store` (`sdkStore`) or returns the key, then runs the listing check.
   *(2026-11-11, later; MADR D3's note.)* `SignInOptions.Remote bool` sets `signin.Options.NoLoopback`; nothing else
   sets it, and no terminal path sets `Remote`. `signin.Start` is called through a package variable, so a test can
   record the `signin.Options` it was given.
   *(2026-11-12; MADR D9, N3.)* `signin.Options.Identity` is P2a's `identityOf(o)`: `{Name: "gobble", Version}` is
   `SignInOptions`' `ClientName` and `ClientVersion`, which `internal/cli/agent.go` sets to `appName` and
   `identity().Version` as every other caller does. go-llmprovider-sdk's 0079-MADR D2 keeps it on the session `Wait`
   saves.
   *(2026-11-13.)* "0052-PLAN P2 step 4" above reads "step 1a"; `o.HostID` is the value 0052-PLAN P2-1's
   `HostID` returns.

   **1a.** *(2026-11-13, added: 0052-PLAN P2's step 4, moved here whole, the wizard's part included, by the owner's ruling
   of that day; 0052-PLAN's "P2 split into P2-1 and P2-2 (2026-11-13)".)* As that step wrote it: `llm/provider/login.go`: `sdkStore`
   implements go-llmprovider-sdk's `auth.ClientStore` when its `LoginStore` also implements gobble's `ClientStore`,
   whose records are 0052-PLAN P2-1's `internal/auth.Clients`; `SignInOptions.HostID` reaches `wizard.Options.HostID`,
   in `SignIn` (`llm/provider/signin.go`, the terminal surfaces' wizard), and `signin.Options.HostID`, in step 1's
   `StartSignIn`. The wizard finds the client store by asserting its `TokenStore`, which is `sdkStore`; `x/signin`
   takes it only as `signin.Options.ClientStore` (step 1; go-llmprovider-sdk
   `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D7).
   *Prerequisites, checked 2026-11-13.* `auth.ClientRecord`, `auth.ClientStore`, `wizard.Options.HostID`, and
   `signin.Options.HostID` and `ClientStore`, are in the exported-name table of go-llmprovider-sdk
   `0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md` ("Exported names (R55)"), built by its Phase 4 and
   released with 0074 in `v1.31.0` (its Phase 7; the fleet sequence's T3). P2 pins `v1.31.0` and this phase comes
   after P2 (Rollout), so they hold; this phase also follows 0052-PLAN P2-1, whose `HostID` and `Clients` it uses.
   None of these names was on go-llmprovider-sdk's `main` at `3f87e2b`; a `v1.31.0` without one is a deviation
   (C4).

2. `acpserver/auth.go`: `AuthOptions` gains `Methods func(id string) []authcontract.Method` (for `_meta.gobble.methods`)
   and `SignIn func(ctx context.Context, id, method string) (SignInRun, error)`, `SignInRun` an interface of
   `acpserver`'s with `StartSignIn`'s methods, `Wait` returning `provider.SignInResult`, so `acpserver` names no
   `x/signin` type (P2's archtest plant). `authMethods` (`:32-53`) sets `_meta.gobble.methods`.
3. `acpserver/authsignin.go`: `Authenticate` (`:59-70`) as MADR D4's four cases, `_meta.gobble` decoded strictly into
   `acpext.AuthenticateMeta`, and `Fresh` skipping case 2 while leaving the stored credential untouched until `Wait`
   saves; the relay reads
   `acp.RequestIDFromContext`, decodes it into `acp.RequestId`, builds `ElicitationRequestScope`, and maps each step
   by the MADR's table; an `open_url` or `show_code` elicitation is acknowledged, not answered (MADR D4): its
   `accept` builds no `Answer`, and for an `open_url` with `accepts` the relay then sends the `paste` step's form,
   whose answer carries the `paste` step's `seq`; URL mode only when `a.client.Elicitation.Url` is set; `elicitation/complete` after a URL
   elicitation whose step ended without an answer; one sign-in per provider (a map under `a.mu`); a client's
   `$/cancel_request` cancels it; `CredentialGeneration` and the run's shared session are refreshed through a new
   `AuthOptions.Signed func(id string)`, which `internal/cli/agent.go` points at `refreshCredentials`.
   *(2026-11-11, later; MADR D4 step 1's note.)* `AuthenticateMeta.Remote` is decoded with the rest, strictly. With it
   `true`, `AuthOptions.SignIn` is called with `remote` set, so `StartSignIn` gets `Remote`; the methods driven are
   those `provider.Remote` accepts (a new `AuthOptions.Remote func(authcontract.Method) bool`, which
   `internal/cli/agent.go` points at `provider.Remote`); a named method that is offered but refused is answered
   `auth_required` with `ErrorData{Provider, Kind: "unsupported", Reason, Command}` before `SignIn` is called. The
   `SignIn` signature becomes `func(ctx context.Context, id, method string, remote bool) (SignInRun, error)`.
4. `internal/cli/agent.go:133-136` [HEAD]: `AuthOptions` gains `Methods: provider.Offered` and `SignIn` over
   `StartSignIn` with `auth.Logins{Path: authPath()}`, storing a key through `auth.SetKey` as `signInTo` does.
5. Tests in `acpserver/auth_test.go`, with a test client that embeds `acptest.Client`, answers `CreateElicitation` from
   a script, and records each request; each seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestAuthenticateRequestScope` | every elicitation is request-scoped, its `requestId` equal to the `authenticate` request's, never session-scoped | the scope built from a counter |
   | `TestAuthenticateDeviceCodeURL` | a client with `elicitation.url`: `show_code` as URL mode to the verification URL, the code and expiry in `message`; `elicitation/complete` with that id when the poll ends; `{}` | no `elicitation/complete` |
   | `TestAuthenticateDeviceCodeForm` | a form-only client that leaves the form open: a form with no fields, URL and code in `message`, cancelled by `$/cancel_request` when the poll ends; `{}` | the form never cancelled |
   | `TestAuthenticateBrowserPasteForm` | a form-only client: a form with no fields, the URL and the paste instruction in `message`; on its empty `accept`, a second form with one required string `callback`; that form's answer reaches `SignInRun.Answer` with the `paste` step's `seq` and `Secret` true, and completes the sign-in; the `open_url` step's `seq` is never answered | URL mode sent to a form-only client (the SDK's `CapabilityError` surfaces); the callback sent as the answer for the `open_url` step's `seq`, which the SDK refuses `ErrInvalidRequest` |
   | `TestAuthenticateShownStepAcknowledged` | a form-only client that accepts a `show_code` form at once: no `Answer` reaches the engine, the poll runs on, and `{}` follows; one that declines it: the sign-in is cancelled, `auth_required` with kind `canceled`, and the poll stops | the `accept` turned into an `Answer`; the `decline` ignored, so the poll runs on |
   | `TestAuthenticateBrowserPasteURL` | with `elicitation.url`: URL mode, then the paste form | the paste form skipped |
   | `TestAuthenticateBadPasteAsksAgain` | a callback with a wrong `state` is asked for again, the paste never in the new `message`; the third ends it `failed`, `invalid_request` | the paste echoed in the re-ask |
   | `TestAuthenticateConfirm` | a `confirm` step is a form with one boolean `allow`, default false; `accept` true and false each become the answer; `decline` cancels the sign-in, `auth_required` with kind `canceled` | `decline` read as `allow: false` |
   | `TestAuthenticateNoElicitation` | a client with no `elicitation`: `auth_required` with provider, reason and command, as today | a sign-in started |
   | `TestAuthenticatePresent` | a credential present and no method named: `{}` with no elicitation; a named method other than the stored one's signs in | `_meta.gobble.method` ignored |
   | `TestAuthenticateFresh` | a grok session stored and `_meta.gobble.fresh: true`: the device sign-in runs; the stored session is replaced only once the new one saves | `fresh` ignored, so `{}` with no elicitation |
   | `TestAuthenticateFreshKeepsOldOnFailure` | the same, with the sign-in failing, declined, and cancelled by `$/cancel_request`: `auth.json` is byte for byte as before, and the issuer sees no revoke | the stored session deleted when the fresh sign-in starts (MADR D4 step 3) |
   | `TestAuthenticateFreshRefusesNonBoolean` | `_meta.gobble.fresh: "yes"` is -32602, and nothing starts | the value read as truthy |
   | `TestAuthenticateOnlyRemoteMethods` | grok with no method named: the device sign-in, never `browser_oauth`; `browser_oauth` when named | the loopback method chosen by default |
   | `TestAuthenticateOneAtATime` | a second `authenticate` for a provider while one runs is -32600; another provider's runs | no lock |
   | `TestAuthenticateWithCancelPending` | a `session/cancel` for an open session sent while an elicitation waits is handled, and the elicitation's answer then completes the sign-in within 5 s (MADR U8 (e)) | `a.mu` held across `CreateElicitation` |
   | `TestAuthenticateClientCancels` | the client's `$/cancel_request` cancels the sign-in and the pending elicitation | the handler's context not passed to `CreateElicitation` |
   | `TestAuthenticateNeverLogsPaste` | the agent's log and every frame but the one answer hold no pasted callback; every `paste` elicitation's `_meta.gobble.step` has `secret: true`, and the `Answer` the relay builds has `Secret` true (MADR D4, "Secrets") | the paste logged at Debug; the relay's `Answer` built with `Secret` nil and logged through a formatter that reads nil as false |
   | `TestAuthMethodsMeta` | each agent auth method's `_meta.gobble.methods` decodes strictly into `[]authcontract.Method` and equals `Offered` | the meta under `gobble.method` |
   | `TestStartSignInReusesClient` (`llm/provider`) | against `planIssuer`, after a logout that keeps `clients.json`: the ACP path's plan sign-in names the stored client and the host id, and sends no `agent_name_hint` | `ClientStore` left nil in `signin.Options` |
   | `TestStartSignInPassesPrevious` (`llm/provider`) | with a stored plan session that has an answer, `StartSignIn`'s steps hold no `confirm` and the authorize request carries `id_token_hint`; with a stored session of another kind, `Previous` is nil | `Previous` never set |
   | `TestAuthenticateGolden` | the frames of a form-only browser-paste sign-in match `acpserver/testdata/authenticate-elicitation.golden` and pass the SDK's `acptest.ValidateGolden` | a frame's `requestId` changed in the golden |

   `TestAuthMethods`, `TestAuthenticate` and `TestNoACPLogout` (`acpserver/auth_test.go:50`, `:73`, `:109`) pass
   unchanged: a client without elicitation sees today's wire.
   *(2026-11-11, added, for MADR D4's three rules of that date.)* Two more tests, each seen failing, and one more
   assertion:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestAuthenticatePastedDenialEnds` | a pasted URL with this sign-in's `state` and `error=access_denied` ends the sign-in at once: no further `paste` form, and `auth_required` with `ErrorData.Kind` `denied` | the relay answering `auth_required` with a fixed kind `auth` in place of the `failed` step's own |
   | `TestAuthenticateGrokBrowserHasNoPaste` | grok's `browser_oauth`, named: one `open_url` elicitation whose `_meta.gobble.step` has no `accepts`, no `paste` form after its `accept`, and the loopback's finish completes it | the relay sending a `paste` form after every acknowledged `open_url` |

   `TestAuthenticateBadPasteAsksAgain` also counts: exactly two re-asks, three `paste` forms in all, before the
   `failed` step.

   *(2026-11-11, later, added, for MADR D2's, D3's and D4 step 1's notes of that date: the remote sign-in.)* Each seen
   failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestAuthenticateRemoteBindsNoPort` | a form-only client's `authenticate` for openai with `_meta.gobble.method: "chatgpt_plan"` and `remote: true`, against `planIssuer`, while the test holds `127.0.0.1:1455` and `:1457` (binding each, or finding it already bound, either of which leaves it unavailable): the sign-in starts, its authorize URL's `redirect_uri` names port 1455, a pasted callback completes it, and `{}` follows; the recorded `signin.Options.NoLoopback` is true | `remote` not passed on to `StartSignIn`, so the listener's bind fails on the held ports and no `open_url` form is sent |
   | `TestAuthenticateLocalKeepsListener` | the same `authenticate` with `remote` absent, and again with `remote: false`: the recorded `signin.Options.NoLoopback` is false. A recorder stands in for `signin.Start` here, so the test binds no real port on the host | `NoLoopback` set for every ACP sign-in |
   | `TestAuthenticateRemoteRefusesLoopbackOnly` | grok with `method: "browser_oauth"` and `remote: true`: `auth_required` with `ErrorData.Kind` `unsupported`, a `Reason`, and `Command` `gobble auth login grok --method browser`; no elicitation; the `signin.Start` variable never called; and the same request without `remote` reaches the recorder, which stands in for `Start`, with `NoLoopback` false | the relay's remote filter dropped, so `Start` is called (the recorded calls count one) |
   | `TestAuthenticateRemoteDefault` | grok with no method named and `remote: true`: the device sign-in, and the recorded `NoLoopback` true | the default taken from `Offered` without `Remote` |
   | `TestAuthenticateRemoteRefusesNonBoolean` | `_meta.gobble.remote: "yes"` is -32602, and nothing starts | the value read as truthy |
   | `TestStartSignInRemoteSetsNoLoopback` (`llm/provider`) | `SignInOptions{Remote: true}` gives `signin.Options.NoLoopback` true, for `chatgpt_plan` and for `device_code`; without `Remote`, false; the terminal paths' `SignIn` never sets it | the option left false whatever `Remote` says |
   | `TestStartSignInRemoteDevice` (`llm/provider`) | a remote grok device sign-in against `testIssuer`, `NoLoopback` set, starts with no `ErrUnsupported` from `Start` and completes by the poll, and the `signin.Options` it ran with has `NoLoopback` true (MADR D3's note: the reading to confirm with the SDK) | `Remote` not passed through for a device method, so `NoLoopback` is false |

   A failure of `TestStartSignInRemoteDevice` that comes from the SDK's `Start`, not from a plant, is a deviation (C4):
   MADR D3's note says why, and it is settled in go-llmprovider-sdk's 0074 record, not here.

   *(2026-11-12, added; MADR D9, N3.)* One more test, seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestStartSignInNamesGobble` (`llm/provider`) | with `ClientName` "gobble" and `ClientVersion` "1.2.3-test": the recorded `signin.Options.Identity` is `{Name: "gobble", Version: "1.2.3-test"}`; a grok device sign-in against `testIssuer`, its requests recorded by a capturing transport, sends `referrer=gobble` in the device request's form and a User-Agent that begins `gobble/1.2.3-test (` on each request | `signin.Options.Identity` left zero; its `Version` a literal |

   *(2026-11-13, added with step 1a; moved from 0052-PLAN P2 step 5.)* One more test, seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestClientStoreAdapter` (`llm/provider`) | `sdkStore` is an `auth.ClientStore` exactly when its store has one, and passes records both ways | the type assertion inverted |

   The wizard's `HostID` has no test of its own in this phase, as it had none in 0052-PLAN P2:
   `TestStartSignInReusesClient` above asserts the host id on the ACP path, and 0052-PLAN P4's
   `TestSignInChatGPTPlan` asserts it on a terminal plan sign-in, its plant "the host id not passed".

6. `docs/guides/credentials.md`: "Signing in from an ACP client".
7. Gates (C6), the two other hosts included.

Acceptance: the tests pass, each plant failed, `go test -race -count=20 -run TestAuthenticate ./acpserver/` passes;
`session.golden` unchanged.

### P7 — the terminal surfaces (MADR D3)

1. `internal/cli/acp.go:96-125`: `acpLogin` runs `/login`'s path (`signIn`, `internal/cli/login.go:48-63`) for the
   provider: the wizard's menu of offered methods, the API key included.
2. `internal/cli/login.go`: `/login`, `gobble setup` and the first run pass the host browser opener as
   `SignInOptions.OpenURL`, as `auth login` does; the device code's text names its expiry, "(expires 14:05)" in local
   time.
3. Tests, each seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestACPLoginOffersMenu` | `gobble acp --login openai` shows the offered methods, the key among them; choosing the key stores it (F53) | the bare login path restored |
   | `TestLoginOpensBrowser` | `/login grok` and `gobble auth login grok` both call the opener once | `/login` passing no opener |
   | `TestDeviceCodeTextNamesExpiry` | the text names the code's expiry | the expiry dropped |

4. Gates (C6).

Acceptance: the tests pass, each plant failed; `TestACPLogin` (`internal/cli/acp_test.go`) and the 14 tests of
`internal/cli/login_test.go` pass, any changed one named here with why.

### P8 — acceptance and close-out (MADR Confirmation)

1. Every Confirmation item checked and recorded with its evidence.
2. **Owner-run probes**, each asked first, each in a scratch `GOBBLE_HOME` on the host, nothing of the owner's real
   `auth.json` read or written:
   * **L1, a grok device sign-in through ACP elicitation:** a scratch form-only ACP client runs `gobble acp`, calls
     `authenticate` with `grok`, and shows the form; the owner enters the code on another device. *Owner-run; asks
     first; scratch home; quota: none (no generation).*
     *(2026-11-11, later: the client sends `_meta.gobble.remote: true`, as magic-cli-remote will, and the record notes
     that no port was bound on the host during the sign-in.)*
     *(2026-11-12; MADR D9's live check, the owner's ruling on Q3, "No separate test".)* L1 ends with `gobble auth logout grok` in the same scratch home,
     which revokes the probe's session under gobble's name (N7). The record notes that no "could not be revoked"
     warning was printed. A refusal is a deviation (C4).
   * **L2, the same through a client offering `elicitation.url`.** *Owner-run; asks first; scratch home; quota: none.*
   * **L3, U1 recorded:** the SDK's own probe of grok's browser from another device (go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md`
     D12). gobble records its outcome and changes nothing; grok's browser stays `browser_loopback` until the SDK's
     fact changes. *Owner-run; asks first; scratch store; quota: none.*
3. `docs/architecture.md`'s rows; `README.md`'s sign-in section; `docs/README.md`'s status cells.
4. On the owner's word: #16 closed with the MADR D8 text (`gh issue close 16 --repo maccavelli/gobble-cli --comment
   …`).
5. The MADR's Completion section; this PLAN `complete`.

Acceptance: Confirmation 1 to 8 met, or each unmet item named with its reason.

## Verification

Each command's exit status is captured before any filter.

* `make pre-add-check FILES="<the phase's Go files>"`
* `make preflight` and `make race` (macOS); `go test -count=1 ./...` and `go test -race ./...` on scratch copies on the
  Linux machine and the Windows laptop
* `make archtest`; `python3 scripts/check_libs.py --self-test`; `make check-libs` (network)
* `make check-acpext-deps`; `go -C acpext test -count=1 ./...`; apidiff of acpext from `acpext/v0.4.0` (P3 step 5)
* `go test -count=1 -run 'TestOffered|TestPredicates|TestEveryCommand|TestResolverAccepts|TestLogoutRevokes|TestErrorKind|TestKindsAndLimits|TestLoginWithReadsTheDefaultMethod|TestErrorData|TestPromptQuotaData|TestStartSignIn' ./llm/... ./internal/... ./acpserver/`
* `go test -count=1 -run 'TestAuthLoginJSON' ./internal/cli/`
* `go test -race -count=20 -run 'TestAuthenticate|TestAuthMethods|TestNoACPLogout' ./acpserver/`
* *(2026-11-11, later:)* `go test -count=1 -run 'TestRemoteMethods|TestStartSignInRemote' ./llm/provider/`; the
  `TestAuthenticateRemote` and `TestAuthenticateLocalKeepsListener` tests run in the line above
* *(2026-11-11, added with P4 step 4b:)* `go test -count=1 -run 'TestRetryCausesAreAcpexts|TestRetryNotice' ./acpserver/`;
  `go test -count=1 -run 'TestRetryLines' ./internal/cli/`
* *(2026-11-12, added with P2a; MADR D9:)* `go test -count=1 -run
  'TestSignInNamesGobble|TestStoredLoginNamedForRefresh|TestLogoutRevokesAsGobble|TestStartSignInNamesGobble'
  ./llm/provider/`; `go test -count=1 -run 'TestCLIPassesGobblesIdentity' ./internal/cli/`
* *(2026-11-13, added with P6 step 1a:)* `go test -count=1 -run 'TestClientStoreAdapter' ./llm/provider/`
* `make check-records markdownlint`
* Every plant of P2 to P7, quoted in the execution record.

## Rollout and Rollback

* **Order across repositories.**
  1. go-llmprovider-sdk: `authcontract/v0.1.0` (go-llmprovider-sdk `0073-PLAN-publish-an-auth-contract-module.md`); its 0076 minor (`v1.30.0`) comes first, now (the owner's DF, 2026-11-10); the minor
     carrying 0074 and 0075 (`v1.31.0` if 0076 is `v1.30.0`).
     *(2026-11-11: in that repository its Go-floor record, numbered 0077, lands before the 0076 minor. It had no
     file there when this note was written, so it is named by number only.)*
     *(2026-11-11, later: it is a file now, go-llmprovider-sdk `0077-MADR-go-1-27-4-for-go-2026-6859.md`, with
     `0077-PLAN-go-1-27-4-for-go-2026-6859.md`.)*
  2. gobble: P0, P1 and 0052-PLAN P1 at any time; P2 after step 1; *(2026-11-12: P2a, MADR D9, after P2 on its own commit;)*
     0052-PLAN P2; P3 and the owner's `acpext/v0.5.0`;
     P4 with 0052-PLAN P3, one commit; P5 to P7; 0052-PLAN P4 to P6; then gobble's next release, `v0.6.0`, tagged by
     the owner.
     *(2026-11-11: first of all, `0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md`: Go 1.27.4, acp-go-sdk2 `v1.36.0`
     and go-selfupdate-lib `v1.26.2`, before P2 and before any tag. And `v0.6.0` pins acp-go-sdk2 `v1.36.0`, in step
     with magic-cli-remote: the owner ruled on 2026-11-11 that `v0.6.0` is the gobble release that must pin the same
     acp-go-sdk2 tag as magic-cli-remote, which is its `docs/decisions/0179-MADR-gobble-native-acp-provider.md` gate 3,
     "Both repositories on one acp-go-sdk2 tag". If either repository's pin moves again before `v0.6.0`, it moves in
     both, or the release waits.)*
     *(2026-11-12: "then gobble's next release, `v0.6.0`, tagged by the owner" now reads `v0.7.0`: gobble's
     `v0.6.0` was cut earlier, from `main`, by
     [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md).)*
     *(2026-11-12: "And `v0.6.0` pins acp-go-sdk2 `v1.36.0`, in step with magic-cli-remote" and "`v0.6.0` is the
     gobble release that must pin the same acp-go-sdk2 tag" stay true, and are met by the `v0.6.0` of
     [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md).
     `v0.7.0` must pin magic-cli-remote's tag again, read at the Gate G2 of magic-cli-remote `0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`,
     and "If either repository's pin moves again before `v0.6.0`, it moves in both, or the release waits" runs on to
     `v0.7.0` (0055-MADR's D2).)*
     *(2026-11-13: "P0, P1 and 0052-PLAN P1 at any time" reads "P0, P1, 0052-PLAN P1 and 0052-PLAN P2-1 at any
     time", and "0052-PLAN P2" after P2a reads "0052-PLAN P2-2". 0052-PLAN P2's step 4 is P6 step 1a; 0052-PLAN's "P2 split into P2-1 and P2-2 (2026-11-13)".)*
  3. magic-cli-remote: `acpext/v0.4.0` ~~now~~ with its gobble mapping patched (Q14; its
     `docs/decisions/0179-PLAN-gobble-native-acp-provider.md` P7); at its cut-over
     (`docs/decisions/0246-PLAN-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`),
     `authcontract`, `acpext/v0.5.0`, and a gobble at or above `v0.6.0`, with magic-cli-remote `0191-MADR-answer-agent-elicitation-forms-on-the-phone.md` and magic-cli-remote `0179-MADR-gobble-native-acp-provider.md` amended;
     daemon and app together.
     *(2026-11-11: "now" has not happened. magic-cli-remote's `go.mod` requires acpext `v0.3.0` and acp-go-sdk2
     `v1.35.0` at its origin/master `eefb7c25`; the acpext bump is its 0179-PLAN P7, commit 7a, still `proposed`. It
     moves acp-go-sdk2 to `v1.36.0` by a record of its own, in step with 0054 here.)*
     *(2026-11-12: "a gobble at or above `v0.6.0`" now reads `v0.7.0`, the redesign's release. `v0.6.0` is the
     release of [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md),
     cut before it, and does not carry the redesign.)*
* **Rollout.** A client that offers no elicitation sees today's ACP wire. `--json` changes at once for every consumer;
  Magic is the only known one, and moves with `v0.6.0`.
  *(2026-11-12: "Magic is the only known one, and moves with `v0.6.0`" now reads `v0.7.0`, the redesign's
  release. `v0.6.0` is the release of
  [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md),
  and leaves `--json` as `v0.5.0` had it.)*
* **Rollback before a tag:** revert the phase's commit.
* **Rollback after `acpext/v0.5.0`:** a tag is never moved or deleted; a fix goes forward as `acpext/v0.5.1`.
* **Rollback after gobble `v0.6.0`:** the previous binary still reads `auth.json` (unknown members round-trip,
  `internal/auth/store.go`'s `member.value`); Magic's cut-over is reverted with it.
  *(2026-11-12: "Rollback after gobble `v0.6.0`: the previous binary still reads `auth.json`" now reads
  `v0.7.0`; the previous binary is then `v0.6.0`'s, cut by
  [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md).)*

## Execution record

~~Not started. The MADR is `proposed`.~~
*(2026-11-11, later: the MADR is `accepted` and this PLAN `in-progress`. P0 is done; see "P0 record (2026-11-11)" at
the end of this section. P1 and the later phases are not started.)*

### Corrections of 2026-11-11 (records only; no phase has started)

Directed by the owner on 2026-11-11 and checked against the tree first. Each is a dated note beside the text it
corrects.

* **Scope, the base:** `main` at `17d1605`; the archive branch also on origin; 0054 first.
* **Scope, added files:** `agent/event.go`, `internal/cli/view.go`, `acpclient/update.go`, `acpext/roundtrip_test.go`
  and the tests and frames that hold a respelled word. Without them `_gobble/retry` would keep `empty_reply` on the
  wire while acpext said `empty_answer`.
* **C6:** `make check-test-isolation`, since 0053.
* **P2 step 1:** what "these two" means once 0054 has moved three other lines.
* **P3:** `TestKindsAndLimits`; acpext's `go 1.27.4` line at `acpext/v0.5.0`.
* **P4 step 1:** the issue is #15, not #3.
* **P4 step 3:** the session's kind through `(*OAuthSession).SessionKind()`.
* **P4 step 4b and two test rows:** the agent's causes tied to acpext's.
* **P6:** two tests and one count for the paste rules.
* **Rollout:** 0054 first; the SDK's Go-floor record before its 0076 minor; `v0.6.0` pins acp-go-sdk2 `v1.36.0` in
  step with magic-cli-remote; magic-cli-remote has not taken `acpext/v0.4.0` yet.
  *(2026-11-12: "`v0.6.0` pins acp-go-sdk2 `v1.36.0` in step with magic-cli-remote" stays true, as the
  rollout's note of this date says: the `v0.6.0` of
  [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md)
  meets it, and the lockstep runs on to `v0.7.0`, the redesign's release.)*

**Later on 2026-11-11, the owner's rulings relayed by the cross-repository coordinator** (records only; no phase has
started):

* **Scope:** `AuthenticateMeta.Remote`, `ErrorData.Consent`, `provider.Remote`, `SignInOptions.Remote` and the relay's
  remote path.
* **P3:** `Remote` and `Consent` in acpext `v0.5.0`, a frame for each, their tests and plants, and apidiff's list.
* **P4 step 2 and its tests:** `provider.Remote`; an honoured consent's text is gobble's; `TestRemoteMethods`.
* **P6 steps 1 and 3 and its tests:** `NoLoopback` for a remote sign-in; the `unsupported` refusal of a named method
  only the loopback completes; seven tests.
* **P8 L1:** sent as a remote sign-in.
* **Verification:** the new tests.
* **Rollout:** go-llmprovider-sdk's 0077 is cited by its file name.

### P0 record (2026-11-11)

C1: branch `main`, `HEAD` `834b6d1` (0054's close-out), `origin/main` `b6adb5a`, one commit ahead, the tree clean
(`git status --short` printed nothing). This pair and the 0052 pair were `proposed`. Records only: no source, test,
`go.mod` or CI file changed. P0 is staged with 0052-PLAN's P0 as one records change, and the owner commits. Every edit
asserted that its old text occurred once before it replaced it, and was read back. Lines below are as they are now.

1. **Acceptance.** The owner's word, typed in this repository's own session on 2026-11-11: "accept and approve 0051 and
   0052 when required"; confirmed to the cross-repository coordinator the same day as: accept 0051-MADR and 0052-MADR,
   and start their phases that do not wait on the SDK. The MADR's front matter is `status: accepted` (its `date` was
   already 2026-11-11). Its Decision Outcome strikes "This record is `proposed`, awaiting the owner's review of this
   draft." and states the acceptance in a dated note (`:258`), quoting the owner's words and naming the questions
   answered on 2026-11-10: QB (a), QC (i), QD (a), QE (a) and O2 (a). "Revisions of 2026-11-11" gains one paragraph
   (`:813`). This PLAN is `in-progress`, with a dated note under Goal (`:25`) and one at the head of this section. Both
   records, and the 0052 pair, were read for any other question marked open or awaiting the owner: none was found.
   Q14's "keep it as an open item in the gobble MADR" (Decision Drivers) is QC, answered (i).
2. **Amendment notes,** each dated 2026-11-11, beside the text it amends and quoting it:

   | Record | Where | The note |
   | :--- | :--- | :--- |
   | `0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md` | A2, the `authenticate` bullet, `:480` | quotes "`authenticate` with such an id answers `{}` … No secret crosses ACP."; amended by MADR D4: absent a credential, an eliciting client is signed in, and a pasted callback crosses ACP as a secret, used once; "No `logout` over ACP." stands (QD (a)) |
   | the same | A3, `:495` | quotes "prints the login's events as JSON lines"; amended by D5: a `method` line, then the contract's `step` lines |
   | `0005-PLAN-v1-feature-scope.md` | F4-3c decision 6, `:5144` | quotes "**`--json` events** are JSON lines on stdout"; superseded by D5: the four events retire, and the kind is the contract's |
   | `0004-MADR-go-module-architecture.md` | the import-boundary rule 4, `:478` | quotes "Only `llm/provider` imports `github.com/maccavelli/go-llmprovider-sdk`"; amended by D1: `authcontract` may also be imported by `acpserver` and `internal/cli`, never by a rule-1 package |
   | the same | Confirmation, the `go list -m` bullet, `:548` | quotes "After F4 it names … After F10 it names …"; from P2 the list names `authcontract`, `// indirect` until P4; D1 names it, as `AGENTS.md` ("Dependencies") asks |
   | `0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md` | D3, the `FailureKind` bullet, `:203` | quotes "`llm/provider.FailureKind` (`list.go:106-133`) is left as it is"; superseded by D6 |
   | the same | D3, the acpext bullet, `:209` | quotes "acpext's `ErrorKind` constants are these names"; holds, with the contract's spellings (D6's table) |
   | `0042-MADR-acpext-extension-contract-module.md` | A1, `:307` | QC (i), answered 2026-11-10: A1 unchanged; quotes "So acpext has no requirement and no `go.sum`" |
   | the same | D3, after the `ErrorData` bullet, `:208` | `acpext/v0.5.0`'s content: `AuthProvider`'s corrected doc, `AuthenticateMeta`, `MetaMethods`, `MetaStep`, `Consent` added to "`ErrorData{Command, Error, Provider, Reason, SessionID string}`", the contract's kind spellings |

   Read against the list as P0 wrote it:
   * **`0031-MADR-sdk-current-releases-and-adoption.md` D8: already done, so not repeated.** D8's text is at `:446-447`,
     and its note of 2026-11-10 at `:464-467` reads: "`HasLogin` and `HasKeyLogin` are to be derived from
     go-llmprovider-sdk's method facts through gobble's one policy filter, `provider.Offered`, by D2 of
     `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` (`proposed`). That decision carries out
     this bullet". `git blame` gives `86ad4a7` for each of its lines, the owner's 0031 paste
     ([0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
     Deviation 1, step 3). No 0031 file was edited; that note's "(`proposed`)" is the MADR's state on the day it was
     written.
   * **0004-MADR's "Dependencies".** 0004-MADR has no section of that name. "Dependencies" is `AGENTS.md`'s section, "No
     module may be required without a MADR in this repository that names it", which MADR D1 cites. The note went to the
     place where 0004-MADR names the modules gobble requires: its Confirmation's `go list -m` bullet.
   * **Wording.** The MADR's "Records amended or superseded in part" gives the substance of the 0016, 0005-PLAN, 0004
     rule 4 and 0049 notes. It gives none for 0042, which it calls "unchanged under QC (i)", nor for 0004's module list;
     those three notes were written from MADR D1 and D7.
   * The MADR's "Records amended or superseded in part" gains a dated note that this step applied them (`:676`).
3. **`docs/README.md`:** the 0051 MADR and PLAN rows (`:117-118`), after 0050's PLAN row, in the form of their
   neighbours: the MADR's or PLAN's title as a link to the file, then the status cell.
4. **Checks,** each exit status taken before any filter, run once over both P0s with every file in place:
   * `python3 -B scripts/check_records.py --check-all`: exit 0, no output.
   * `make check-records markdownlint`: exit 0, "Summary: 0 issues in 0 files". markdownlint's globs leave out the
     `*MADR*` and `*PLAN*` files; `check_records.py` checks their names, pairing and links.
   * Seen failing: on a scratch copy of `docs/` and `scripts/`, the 0045-MADR note's link to 0052-MADR shortened to a
     file that does not exist made `check_records.py --check-all` exit 1, with "broken relative link:
     docs/decisions/0045-MADR-auth-status-json.md:118: 0052-MADR-finish-the-chatgpt-plan-sign-in.md". The copy's other
     failures were the links to the root's `LICENSE` and `NOTICE`, which were not copied.
   * The identifier scan of the diff's added lines, for the local account name, the host's names, the org mail domain
     and real home and temporary paths: nothing found (`grep` exit 1). On a copy of those lines with a planted home
     path and host name it found one line (exit 0).

Acceptance: every note quotes what it amends, and `check-records` passes. Met. P1 is not started.
*(2026-11-11, later: the owner committed P0 as `011b63a`.)*

### P1 record (2026-11-11)

Run by the cross-repository coordinator on the owner's approval of this PLAN, from `main` at `011b63a`, clean tree.

1. `acpext/auth.go:3-8`, `AuthProvider`'s doc, now reads: "AuthProvider is a provider gobble's ACP authentication
   names (0016-MADR A2). ID is the id of the agent auth method gobble advertises for it. Name is its label. Command runs
   the provider's bare sign-in, or, ending in "--api-key -", reads a key from standard input. Login means the provider
   has a way to sign in other than an API key; it never means a device code, and a provider with a sign-in may still
   take a key, as OpenAI's offers its ChatGPT plan sign-in and its API key (gobble-cli 0031-MADR D7). A client reads
   each sign-in method's facts from the agent's _meta.gobble.methods once connected (gobble-cli 0051-MADR D7)."
2. No code, test or tag. The doc reaches pkg.go.dev with `acpext/v0.5.0` (P3).
3. `make pre-add-check FILES="acpext/auth.go"`: exit 0, "go-precheck: 1 file(s) clean (gofmt, golint, govulncheck)."
   `make check-acpext-deps`: exit 0, "check-acpext-deps: ok". `go -C acpext test -count=1 ./...`: exit 0,
   "ok github.com/maccavelli/gobble-cli/acpext".
4. `git diff -U0 -- acpext/auth.go`, filtered to changed lines that are not `//` comments: none (`grep` exit 1). The
   diff is 9 lines added and 6 removed, all in the comment.

Acceptance: `go -C acpext test` passes unchanged, and the diff touches `acpext/auth.go`'s comment only. Met. Staged
with this record; the owner commits. P2 waits for the SDK's tags (the gate before P2).

### Amendment of 2026-11-12 (records only; no phase started)

The owner's direction of 2026-11-12, "Where possible gobble auth would send gobble as the name/referrer not the sdk",
carried here after go-llmprovider-sdk's
`0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md` (accepted 2026-11-12) and its PLAN (in
progress), both read at `f50466b` in that repository. Read against this repository at `c47f4a7`, `main`, a clean tree.
Each change is a dated note beside the text it extends, or a new phase; nothing is struck.

* **MADR D9, new:** gobble's eleven sites (N1 to N11); three need gobble's code: the wizard's sign-ins carry no
  `Identity` (N1), the model facts' lookup passes no `WithClientInfo` for the SDK to hand over (N6), and the logout's
  session is gobble's own (N7).
* **The SDK pin is P2's, not P4's.** P2 step 1 runs `go get github.com/maccavelli/go-llmprovider-sdk@<minor>`, and the
  fleet sequence's pin table gives the move from `v1.29.0` to `v1.31.0` to "gobble 0051 P2". P4, committed with
  0052-PLAN P3, makes `authcontract` direct and pins acpext `v0.5.0`. So P2a follows P2.
* **Goal, Scope:** a note and a bullet. **P2a, new:** a step 0, three steps, four tests with their planted breaks. **P4 step 3:**
  the rewrite keeps the call, and a plan case. **P6 step 1:** the identity from P2a's helper, and
  `TestStartSignInNamesGobble`. **P8 L1:** ends with a logout. **Verification:** the new tests. **Rollout:** P2a's
  place.
* **0052-PLAN P6's L6:** a note that its logout records the plan session's revocation under gobble's name.
* ~~**Open:** MADR Q1 (P2a's place), Q2 (the model facts) and Q3 (the live check), each with a recommended option. P2a
  starts only on the owner's approval of it.~~
  *(2026-11-12, later: the owner ruled that day. Q1, "New phase P2a": P2a is part of this PLAN, after P2 on its own
  commit. Q2, "Amend SDK 0079": go-llmprovider-sdk puts `CachedModel` and `LookupModel`'s `listingModel` through its
  0079-MADR D4 hand-off, by a pending dated amendment of
  `0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md`; gobble calls no `UseIdentity` for
  the model facts and passes `WithClientInfo` on its lookup (P2a step 2, `TestStoredLoginNamedForRefresh` and its
  plants reworked to it, step 0's check of the SDK's minor added). Q3, "No separate test": no gobble probe; L1 and L6
  each end with a logout and record that the revocation was not refused. MADR D9 is accepted 2026-11-12. P2a's text,
  not yet committed, was brought to the rulings in place.)*
* **Not changed:** the fleet sequence's
  `0006-PLAN-sequence-the-auth-contract-effort-across-four-repositories.md` stage 7, which lists 0051 P2 then
  0052-PLAN P2; its owner adds P2a between them.

### Amendment of 2026-11-13 (records only; no phase started)

The owner's rulings of 2026-11-13, relayed by the cross-repository coordinator, on a read-only check of
[0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
P2 against go-llmprovider-sdk `v1.29.0` and that repository's `main` at `3f87e2b`. Read against this repository at
`8fa0b73`, `main`, a clean tree. Each change is a dated note beside the text it extends, or a new step; nothing is
struck. The evidence and the rulings are in 0052-PLAN's "P2 split into P2-1 and P2-2 (2026-11-13)".

* **0052-PLAN P2 is split.** P2-1, the stored members, the host id and the client file, is built now on `v1.29.0`
  with no SDK move. P2-2, the SDK session's `Kind` and `Consents` through `session` and `loginOf`, with
  `TestLoginCarriesEverySessionField`, follows P2 and P2a. From P2's pin to P2-2 nothing guards that round trip, the
  owner's accepted cost.
* **0052-PLAN P2's step 4 is P6 step 1a,** the wizard's `HostID` included, with `TestClientStoreAdapter` and its
  plant. Its prerequisites, go-llmprovider-sdk 0075's names in `v1.31.0`, were checked against P6's place.
* **Here:** Scope (a bullet, and a note on Out of scope); the gate before P2; P2a's place; P4's prerequisite; P6 step
  1's note, step 1a and one test; Verification; Rollout. **0051-MADR:** D9's "Where in the PLAN" and "Revisions of
  2026-11-13".
* **Not changed:** the fleet sequence's
  `0006-PLAN-sequence-the-auth-contract-effort-across-four-repositories.md` stages 4G and 7; its owner amends them.
