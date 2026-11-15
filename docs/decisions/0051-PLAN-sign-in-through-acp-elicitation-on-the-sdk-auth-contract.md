---
status: in-progress
date: 2026-11-15
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
* *(2026-11-14, added by P2a's deviation 2 and C9; the owner's ruling of that day.)* `README.md`'s Credentials bullet,
  `docs/guides/credentials.md`'s "Signing in" and logout paragraphs, and the `llm/provider` row of
  `docs/architecture.md`: what name and version gobble sends, and where.
* *(2026-11-14, added by P2's deviations 1 and 2, and P7a; the owner's rulings of that day.)* `README.md:163` and
  `docs/architecture.md:192`, the SDK's version in each (P2, deviation 2); and P7a's `scripts/check_libs.py`
  (advisories by package).

**Out of scope:** the ChatGPT plan, its consent, host id, client and migration (0052); go-llmprovider-sdk's code and
records (the owner's, in that repository); magic-cli-remote; `auth status --json` (0052); a release tag or push
without the owner's word; posting on #16 before P8.
*(2026-11-13: but for 0052-PLAN P2's step 4, the host id's and the issued client's plumbing, which P6 step 1a now
does.)*

## Conventions every phase follows

* **C1. Start state.** Each phase re-reads `git status`, `HEAD`, `origin/main`, and the versions `go.mod` requires. A
  difference from this plan is a deviation. *(2026-11-15, the owner's ruling "Keep it on the branch until those phases land": while `main` waits, the start state of a phase on the branch is read against the branch's tip and `origin/p3-acpext-v0.5.0`, not `origin/main`; "P4 record (2026-11-15)", the question on `main`.)*
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
  *(2026-11-15, the owner's ruling after P3's finding; see "P3 record (2026-11-15)", "The sequence".)* C8 stands, 0042-MADR
  D5's rule unchanged, with the tag in two parts. P3's commit is on the branch `p3-acpext-v0.5.0`, not on `main`. The
  owner tags it `acpext/v0.5.0-rc.1` there, on the owner's own word. The root requires `acpext@v0.5.0-rc.1` (P4 step 1),
  by tag, with no `replace` and no `go.work`. The final `acpext/v0.5.0` is the owner's, after P8 (P8a). `main` moves
  once, to the branch's tip, when every C6 gate is green on that tip. *(2026-11-15, the owner's ruling relayed by the coordinator, "Keep it on the branch until those phases land": the coordinator commits each approved phase on the branch `p3-acpext-v0.5.0` and pushes the branch; `main` stays at `9e21c8f` and is fast-forwarded only after 0051-PLAN P6 and 0052-PLAN P4 have landed on the branch and every gate is green, because until then a plan sign-in passes no host id and does not reuse its registered client. Consequence told to and accepted by the owner: CI runs on pushes to `main`, so until `main` moves the phases on the branch have only the local gates and the Linux and Windows runs behind them; `.github/workflows/ci.yml` also runs on `pull_request` and on `v*` tags, which a pull request from the branch would use.)* *(2026-11-15, later: the owner's ruling "Open a draft PR now", relayed by the coordinator. The coordinator's statement that CI runs only on pushes to `main` and tags was wrong: `.github/workflows/ci.yml` also runs on `pull_request`, which the review's finding showed. After P4 is committed and pushed the coordinator opens one draft pull request from `p3-acpext-v0.5.0` to `main`, so every later push to the branch runs CI; `main` does not move and nothing merges until the owner's word, and `main` is still fast-forwarded only after 0051 P6 and 0052 P4 land. The sentence above that the phases on the branch have only the local gates behind them holds only until that pull request is open.)*
* **C9. Docs in the same phase.** *(2026-11-14, added; the owner's ruling on P2a's hand-back, relayed by the
  cross-repository coordinator. Asked what happens when a phase's docs step finds user docs outside the phase's own
  files that the phase makes false or leaves incomplete, the owner chose "Always in the same commit": "The agent fixes
  or adds the doc text in the same phase, the files joining the phase's scope by a dated deviation, without asking you
  each time. Docs never lag the code." Rejected: "False now; missing in the docs phase"; "Ask me each time". It binds
  every later phase of this PLAN.)* A phase's docs step names every document its change makes false, missing or
  incomplete, whatever file it lies in, and the phase fixes or adds the text in the same commit; each such file joins
  the phase's scope by a dated deviation in the record. Each sentence added is checked against the code and the
  phase's tests before it is kept.

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
   *(2026-11-15, the owner's ruling; "P3 record (2026-11-15)".)* Step 7 reads: the P3 commit is made on the branch
   `p3-acpext-v0.5.0`, and pushed there; on the owner's own word the owner tags it, annotated, `acpext/v0.5.0-rc.1`,
   pushes the tag, and then `go list -m github.com/maccavelli/gobble-cli/acpext@v0.5.0-rc.1` succeeds through the
   proxy. `acpext/v0.5.0` is not tagged in this phase (P8a). **The branch's P3 commit, checked out alone, fails two
   root tests, by construction:** `TestPromptQuotaData/prompt-rate-over-cap.error.json` and
   `TestRetryNoticeServerMembers` (both `acpserver`) read acpext's rewritten frames from `../acpext/testdata`, and the
   root's code still sends `rate-limited` until P4. This is the owner's accepted cost of the sequence. The commit
   fails `make race` and `make test`, and CI, until P4's commit follows it on the branch; `main` never holds it alone.

Acceptance: the tests pass, each plant failed; apidiff as step 5; the tag served.
*(2026-11-15: "the tag served" reads `acpext/v0.5.0-rc.1` served by the proxy; and "the tests pass" is acpext's own, on
every host, the root's two named tests excepted as step 7's note says.)*

### P4 — the policy, the resolver and the vocabulary in the root (MADR D2, D6)

Prerequisites: P2 and P3's tag; 0052-PLAN P1 (the WIP sorted) and P2 (the session's `kind` and `consents` in
`auth.json`, so no sign-in this phase enables can lose them).
*(2026-11-13: 0052-PLAN P2 is now its P2-1 and P2-2, both before this phase; its step 4 is P6 step 1a.)*
*(2026-11-15, the owner's ruling; "P3 record (2026-11-15)".)* "P3's tag" reads `acpext/v0.5.0-rc.1`, served by the
proxy: the prerequisite is the candidate, not the final tag. This phase is built on the branch `p3-acpext-v0.5.0`,
after P3's commit, whole and committed with 0052-PLAN P3 as one commit, as the header below says. Its evidence: P4's
steps 1 to 6 cannot be split. A respelling alone (steps 1, 4 and 4b) was tried on a scratch copy with the in-tree
`acpext` in place of the tag: 37 packages passed and `TestAcpextAuthProvidersAreTheRegistry` failed, because any
requirement past `v0.3.0` brings in `acpext/v0.4.0`'s openai entry (`Login: true`, the bare command), which the registry
matches only once the plan is offered (steps 2 and 5, with 0052-PLAN P3). `main` is fast-forwarded to the branch's tip
when every C6 gate is green on it; the later phases (P5 onward) continue on `main`. *(2026-11-15, the owner's ruling relayed by the coordinator, "Keep it on the branch until those phases land": the coordinator commits each approved phase on the branch `p3-acpext-v0.5.0` and pushes the branch; `main` stays at `9e21c8f` and is fast-forwarded only after 0051-PLAN P6 and 0052-PLAN P4 have landed on the branch and every gate is green, because until then a plan sign-in passes no host id and does not reuse its registered client. Consequence told to and accepted by the owner: CI runs on pushes to `main`, so until `main` moves the phases on the branch have only the local gates and the Linux and Windows runs behind them; `.github/workflows/ci.yml` also runs on `pull_request` and on `v*` tags, which a pull request from the branch would use.)* *(2026-11-15, later: the owner's ruling "Open a draft PR now", relayed by the coordinator. The coordinator's statement that CI runs only on pushes to `main` and tags was wrong: `.github/workflows/ci.yml` also runs on `pull_request`, which the review's finding showed. After P4 is committed and pushed the coordinator opens one draft pull request from `p3-acpext-v0.5.0` to `main`, so every later push to the branch runs CI; `main` does not move and nothing merges until the owner's word, and `main` is still fast-forwarded only after 0051 P6 and 0052 P4 land. The sentence above that the phases on the branch have only the local gates behind them holds only until that pull request is open.)*
*(Its "P5 onward continue on `main`" is annotated by that ruling: P5 and P6 continue on the branch, and so does 0052-PLAN P4.)*

**Committed with 0052-PLAN P3, as one commit.** `acpext/v0.5.0` keeps openai's `Login: true` and its bare command
(`acpext/auth.go:25`), so `TestAcpextAuthProvidersAreTheRegistry` holds only once openai's plan is offered, which the
consent rule allows only once 0052-PLAN P3 honours `chatgpt_plan.background_use`. No commit offers the plan without
gobble's wording and gate.

1. *(2026-11-15: `@v0.5.0-rc.1`; the final tag is P8a's.)* `go get github.com/maccavelli/gobble-cli/acpext@v0.5.0`, `go mod tidy`. With step 2's import, `tidy` makes
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
   | *(2026-11-11, later, added)* `TestRemoteMethods` (`llm/provider`) | per provider, the offered methods other than `api_key` that `Remote` accepts: grok `device_code`; kilo `device_code`; openai `chatgpt_plan`; grok's `browser_oauth` refused *(2026-11-15, the owner's ruling "Amend the records to the SDK's facts": grok's `browser_oauth` accepted at `v1.31.0`, where the row says refused; "P4 record (2026-11-15)", deviation 4)* | `Remote` reading `Completion` alone, so grok's `browser_oauth` is accepted; `Remote` refusing every browser completion, so openai loses `chatgpt_plan` *(2026-11-15: the first plant cannot fail the test at the pinned facts, `browser_oauth` being accepted by the rule; the plants run were `Remote` reading `Completion` alone, so a loopback-only method of the contract's own shape is accepted, and `Remote` refusing every browser completion, so grok loses `browser_oauth` and openai `chatgpt_plan`; "P4 record (2026-11-15)")* |
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
*(2026-11-15: built, staged with 0052-PLAN P3 on the branch `p3-acpext-v0.5.0`; see "P4 record (2026-11-15)". Two wordings
could not be followed as written and are deviations there: `provider.Login(id)` is `HasSignIn(id)` (1), and `Accepts` is
`provider.Accepts(id, *Login)` with `auth.AcceptsLogin` as the resolver's (2). One fact the PLAN's test row states does not hold at
go-llmprovider-sdk `v1.31.0`, Grok's `browser_oauth` being `browser_paste` with a paste, so `TestRemoteMethods` expects it accepted
(4, awaiting the owner's word).)*
*(2026-11-11: and `git grep -n '"empty_reply"\|"rate-limited"\|"not-permitted"\|"context-overflow"' -- '*.go'
'acpext/testdata'` finds nothing. `retry.empty_reply`, the personality key, stays and is not matched by it.)*

### P5 — `gobble auth login --json` prints the contract's steps (MADR D3, D5; QE (a))

1. `llm/provider/signin.go`: `SignInOptions.Steps func(authcontract.Step) error` replaces `ShowDeviceCode` and
   `OpenURL` for a non-terminal caller; `SignIn` passes `wizard.Options.Driver` = a driver that calls it and returns a
   zero answer, or, for `paste`, `confirm` and `form`, an error marked `unsupported`. `NoPastePrompt` stays set.
2. `internal/cli/auth.go`: `loginEvent` and its five events go; `jsonSteps` writes the `method` line, then each step; *(2026-11-15, carried from the review: a `failed` step's kind is `provider.ErrorKind` of the error `SignIn` returns, which wraps the SDK's errors in `provider.SignInError`; a caller of the SDK's sign-in that passed a raw SDK error to `ErrorKind` would get `other`, so a test in this phase holds that every error it reads is wrapped first, as 0051-MADR D6 reads it.)*
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

*(2026-11-15.)* **Open question for the owner, before P6 runs:** the three rows annotated below (`TestAuthenticateOnlyRemoteMethods`,
`TestAuthenticateGrokBrowserHasNoPaste`, `TestAuthenticateRemoteRefusesLoopbackOnly`) cannot pass as written at go-llmprovider-sdk `v1.31.0`, where Grok's
`browser_oauth` is `browser_paste` with a paste, so `Remote` accepts it (the owner's ruling "Amend the records to the SDK's facts"). What replaces them is
undecided: (a) keep the loopback-only rule's tests through a stand-in method of that shape offered to the relay, or (b) drop the rows because no offered method has
that shape, the rule staying in D4 untested. The plant of the first row, "the loopback method chosen by default", needs replacing under either. Recommended: (b) is
smaller and (a) holds the rule; the choice is the owner's.

1. `llm/provider/signinrun.go`: `StartSignIn(ctx, id, method string, o SignInOptions, store LoginStore) (*SignInRun,
   error)` over `signin.Start`, with `signin.Options` as MADR D3 lists it: `Identity: {Name: "gobble", Version}`,
   `AgentName: "gobble"`, `HostID` from `o.HostID` (0052-PLAN P2's `<state>/host-id`), `ClientStore` the `sdkStore`
   adapter when `store` has a client store (0052-PLAN P2 step 4; nil otherwise, which `x/signin` reads as none),
   `ConsentText` gobble's wording (0052-PLAN P3) *(2026-11-15, the owner's ruling "Accept; P6 builds the site", relayed by the coordinator: this is the `signin.Options` half of 0052-PLAN P3 step 1, which that phase could not build because this site is built here; 0052-PLAN P4's `TestConsentWordingOverACP` guards it. A second pointer, carried from the review of 2026-11-15: an error from `signin.Start` or `Wait` is wrapped in `provider.SignInError` before `ErrorKind` reads it, because `ErrorKind` takes the SDK's word only for a `*SignInError` or the SDK's denied and expired markers (0051-MADR D6), and a test in this phase holds it.)*, and `Previous` the session `store` holds for `id` when its kind is the
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
   *(2026-11-13, later; the owner's ruling of that day on who reads the host id: "0051 P6 step 1a".)* `internal/cli` fills
   `SignInOptions.HostID` from `auth.HostID(dirs.State)`, the state directory as `systemDirs`
   (`internal/cli/configpath.go:14`) resolves it, on both paths:
   * **the terminal path:** in `signInTo` (`internal/cli/login.go:70`), before it calls `signInWith`. Every terminal
     sign-in reaches it: `auth login` through `loginWith` (`internal/cli/auth.go:222`), and `/login`, `gobble setup`,
     the first run and, after P7, `gobble acp --login` through `signIn` (`internal/cli/login.go:62`). If P5 leaves
     `auth login --json` on a path that does not reach `signInTo`, that path fills it the same way;
   * **the ACP path:** in step 4's `AuthOptions.SignIn` (`internal/cli/agent.go`), before it calls
     `provider.StartSignIn`, which it calls through a package variable, `startSignInWith`, as it calls `signInWith`.

   `auth.HostID` is on `main` since 0052-PLAN P2-1 (`5147e09`): `func HostID(dir string) (string, error)`
   (`internal/auth/hostid.go:23`). It reads `<dir>/host-id`, or creates it once, owner-only (0052-MADR D4). In
   `llm/provider`, `SignIn` calls `wizard.ResolveCredential` (`llm/provider/signin.go:184`) through a package
   variable, as step 1 calls `signin.Start`, so a test can record the `wizard.Options` it was given.
   *Not ruled:* what a sign-in does when the state directory cannot be resolved or `auth.HostID` returns an error.
   The owner rules on it before P6 starts; this step names no behaviour for it until then.
   *(2026-11-13, later still; the owner's ruling of that day on the host id's failure case: "Proceed, warn once".)* The options, as put:
   (a) *recommended:* sign in with no host id and log one warning; the SDK then mints a per-sign-in id (0052-MADR's
   F24 behaviour) only in that failure case, and grok and kilo sign-ins never fail over a file they do not use; the
   cost is that a plan session made in that state carries a one-off id;
   (b) fail every sign-in, exit 1: strict, but a state directory that cannot be written blocks grok and kilo too;
   (c) fail only when the method is a plan sign-in, and proceed for the others: the most precise, but `internal/cli`
   would need the chosen method before the wizard's menu runs, or `llm/provider` would need to do the check itself.
   The owner chose (a). When `systemDirs` fails or `auth.HostID` returns an error, the sign-in proceeds with
   `SignInOptions.HostID` empty, and the failure is warned once for that sign-in, naming the error and never an id:
   on the terminal path through the run's warning output (`e.out.Warnf`, as `internal/cli/login.go:266` warns), on
   the ACP path in the agent's log at Warn. No sign-in, of grok, kilo or the plan, fails over the host-id file.

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
   | `TestAuthenticateOnlyRemoteMethods` | grok with no method named: the device sign-in, never `browser_oauth`; `browser_oauth` when named *(2026-11-15, the owner's ruling "Amend the records to the SDK's facts": at go-llmprovider-sdk `v1.31.0` Grok's `browser_oauth` is `Completion` `browser_paste` with `Paste` `url_or_code`, so `Remote` accepts it and it is not a loopback-only method, so with no method named a remote sign-in no longer has the device sign-in as the only method it can drive, and the row as written, "never `browser_oauth`", cannot pass at those facts. Its planted break, "the loopback method chosen by default", names a method that no longer exists among the offered ones. What replaces the row and its plant is an open question for the owner before P6 runs; see "Open question (2026-11-15)" under P6.)* | the loopback method chosen by default |
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
   | `TestAuthenticateGrokBrowserHasNoPaste` | grok's `browser_oauth`, named: one `open_url` elicitation whose `_meta.gobble.step` has no `accepts`, no `paste` form after its `accept`, and the loopback's finish completes it *(2026-11-15, the owner's ruling "Amend the records to the SDK's facts": at go-llmprovider-sdk `v1.31.0` Grok's `browser_oauth` is `Completion` `browser_paste` with `Paste` `url_or_code`, so `Remote` accepts it and it is not a loopback-only method, so the row as written, a Grok browser sign-in with no `accepts` and no `paste` form, cannot pass at those facts: its `browser_oauth` has both. Whether the no-paste shape keeps a test, through a stand-in method, or the row is dropped because no offered method has that shape is an open question for the owner before P6 runs; see "Open question (2026-11-15)" under P6.)* | the relay sending a `paste` form after every acknowledged `open_url` |

   `TestAuthenticateBadPasteAsksAgain` also counts: exactly two re-asks, three `paste` forms in all, before the
   `failed` step.

   *(2026-11-11, later, added, for MADR D2's, D3's and D4 step 1's notes of that date: the remote sign-in.)* Each seen
   failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestAuthenticateRemoteBindsNoPort` | a form-only client's `authenticate` for openai with `_meta.gobble.method: "chatgpt_plan"` and `remote: true`, against `planIssuer`, while the test holds `127.0.0.1:1455` and `:1457` (binding each, or finding it already bound, either of which leaves it unavailable): the sign-in starts, its authorize URL's `redirect_uri` names port 1455, a pasted callback completes it, and `{}` follows; the recorded `signin.Options.NoLoopback` is true | `remote` not passed on to `StartSignIn`, so the listener's bind fails on the held ports and no `open_url` form is sent |
   | `TestAuthenticateLocalKeepsListener` | the same `authenticate` with `remote` absent, and again with `remote: false`: the recorded `signin.Options.NoLoopback` is false. A recorder stands in for `signin.Start` here, so the test binds no real port on the host | `NoLoopback` set for every ACP sign-in |
   | `TestAuthenticateRemoteRefusesLoopbackOnly` *(2026-11-15, the owner's ruling "Amend the records to the SDK's facts": at go-llmprovider-sdk `v1.31.0` Grok's `browser_oauth` is `Completion` `browser_paste` with `Paste` `url_or_code`, so `Remote` accepts it and it is not a loopback-only method, so grok with `method: "browser_oauth"` and `remote: true` is not refused and the row as written cannot pass at those facts. Whether the refusal keeps a test, through a stand-in method, or the row is dropped because no offered method has that shape is an open question for the owner before P6 runs; see "Open question (2026-11-15)" under P6.)* | grok with `method: "browser_oauth"` and `remote: true`: `auth_required` with `ErrorData.Kind` `unsupported`, a `Reason`, and `Command` `gobble auth login grok --method browser`; no elicitation; the `signin.Start` variable never called; and the same request without `remote` reaches the recorder, which stands in for `Start`, with `NoLoopback` false | the relay's remote filter dropped, so `Start` is called (the recorded calls count one) |
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

   *(2026-11-13, later; the owner's ruling of that day on who reads the host id: "0051 P6 step 1a".)* Two more tests, each seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestHostIDReachesTheEngines` (`llm/provider`) | `SignInOptions{HostID: "urn:uuid:…-test"}`: the recorded `wizard.Options.HostID` of `SignIn` and the recorded `signin.Options.HostID` of `StartSignIn` both equal it | the id left empty: `SignInOptions.HostID` not copied, so both recorded options hold `""` |
   | `TestCLIFillsHostID` (`internal/cli`) | in a scratch `GOBBLE_HOME` whose state directory holds a `host-id`: `auth login grok` and `/login grok`, through a recording `signInWith`, and an ACP `authenticate` for grok, through a recording `startSignInWith`, each pass `SignInOptions.HostID` equal to that file's id; with no `host-id`, the first call creates one and both paths pass the same id | the id left empty: `signInTo` not setting it, and, as a second plant, the ACP `SignIn` not setting it |
   | *(2026-11-13, later still; the ruling "Proceed, warn once".)* `TestSignInWithoutHostID` (`internal/cli`) | a scratch `GOBBLE_HOME` whose state directory cannot be written (its parent made read-only on Unix; a deny-write ACE on Windows), and no `host-id`: `auth login grok` through a recording `signInWith`, and an ACP `authenticate` for grok through a recording `startSignInWith`, each complete the sign-in (the recorder is called once, the credential saved, exit 0 and `{}`) with `SignInOptions.HostID` empty, and each emits exactly one warning naming the host-id error | the error returned instead of warned, so the sign-in ends with the error, the recorder is never called and no warning is counted |

   The wizard's `HostID` has no test of its own in this phase, as it had none in 0052-PLAN P2:
   `TestStartSignInReusesClient` above asserts the host id on the ACP path, and 0052-PLAN P4's
   `TestSignInChatGPTPlan` asserts it on a terminal plan sign-in, its plant "the host id not passed".
   *(2026-11-13, later: it has one now, `TestHostIDReachesTheEngines` above, by the owner's ruling of that day.)*

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

### P7a — check_libs.py's advisories by package (added 2026-11-14; the owner's ruling on P2's observation)

Not started. Its file: `scripts/check_libs.py`, and its record in this PLAN. No Go code changes.

1. **A question first, for the executor.** `check_library` reads the repository's published security advisories
   (`advisories` and the loop in `check_library`, `scripts/check_libs.py`) and tests each vulnerable version range
   against the tag being checked, whatever package the advisory names. The script reads only `state` and
   `vulnerabilities[].vulnerable_version_range`; this PLAN has not read what the response carries for the package.
   Before writing code, read one real response, or the API's schema, and answer: does each `vulnerabilities[]` entry
   carry `package.name`, and for a Go module advisory is it the module path (for the contract,
   `.../go-llmprovider-sdk/authcontract`)? If it does not, stop and put the question to the owner: the step below cannot
   be written as it stands.
2. **If it does,** an advisory entry counts against a library only when its package name equals `lib.module`, so the
   SDK's advisories no longer fail the contract's entry and the contract's none fail the SDK's. What an entry that names
   no package does (counted for every library of the repository, as today, or not) is also the executor's question.
3. **`--self-test` gains a case** over fixed advisory data naming two packages of one repository, a range `< 1.2.0`
   for the SDK's path only: the contract at `v0.1.0` has no hit, and the SDK at `v1.1.0` has one. **Planted break:** the
   package filter removed, so the contract at `v0.1.0` reads the SDK's range as covering it; the case fails. Seen
   failing on a scratch copy (C2), quoted.
4. **Gates (C6):** `python3 scripts/check_libs.py --self-test`; `check_libs.py --lib go-llmprovider-sdk` and
   `--lib authcontract` against a commit (the checker clones the commit); `make check-records markdownlint`;
   `make preflight`, which lints the PLAN. The Go race and cross-host runs are not rerun for a Python-only change; the
   record says so with the files' last-modified times.
5. **P8's close-out waits for P7a.**

Acceptance: the question of step 1 answered in the record; the case passes and its plant failed; both `--lib` runs report
0 failing on the committed step.

### P8 — acceptance and close-out (MADR Confirmation)

0. *(2026-11-14.)* Not before P7a is done and recorded.

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
     fact changes. *Owner-run; asks first; scratch store; quota: none.* *(2026-11-15, the owner's ruling "Amend the records to the SDK's facts": the SDK's fact changed at `v1.31.0`, Grok's browser sign-in is `browser_paste`; L3 records the probe's outcome as written, and nothing else in this row changes.)*
3. `docs/architecture.md`'s rows; `README.md`'s sign-in section; `docs/README.md`'s status cells.
4. On the owner's word: #16 closed with the MADR D8 text (`gh issue close 16 --repo maccavelli/gobble-cli --comment
   …`).
5. The MADR's Completion section; this PLAN `complete`.

Acceptance: Confirmation 1 to 8 met, or each unmet item named with its reason.

### P8a — the final acpext tag (added 2026-11-15; the owner's ruling, "P3 record (2026-11-15)")

After P8 is accepted and before any gobble release (`v0.7.0`). Prerequisites: P8 recorded, the owner's word that the
repository is stable and correct.

1. **The owner tags `acpext/v0.5.0`**, annotated, message the tag name, pushed, on the commit `main` has then, if
   `git diff acpext/v0.5.0-rc.1 HEAD -- acpext/` is empty. If `acpext/` has changed since the candidate, on the commit
   that made the last such change, or on a commit after it, which the record names. Either way the tag names what
   `main` ships, and `go list -m github.com/maccavelli/gobble-cli/acpext@v0.5.0` succeeds through the proxy.
2. **apidiff from `acpext/v0.4.0` is re-run** then, as P3 step 5 ran it, on export data of `git archive acpext/v0.4.0`
   and of the commit tagged: the lists are P3 step 5's, with whatever `acpext/` gained since the candidate named as an
   addition or a deviation.
3. **One commit moves the root from the candidate to the final tag:** `go get
   github.com/maccavelli/gobble-cli/acpext@v0.5.0`, `go mod tidy`, then `docs/architecture.md`'s acpext row, and the
   gates of C6 (`make preflight`, `make race`, `make check-test-isolation`, and the two other hosts' `go test ./...`).
   The commit changes `go.mod` and `go.sum` and no Go file.

Acceptance: the tag served; the root requires `v0.5.0`; every C6 gate green; no `v0.5.0-rc.1` left in `go.mod` or in
a document that states what the root requires.

Rollback: a tag is never moved or deleted (P3's note under "Rollback after `acpext/v0.5.0`"); a fault in either tag
goes forward as the next patch, `acpext/v0.5.1`.

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
  *(2026-11-13, later: with `|TestHostIDReachesTheEngines` added to that pattern; and `go test -count=1 -run
  'TestCLIFillsHostID' ./internal/cli/`.)*
  *(2026-11-13, later still: that `./internal/cli/` pattern reads `'TestCLIFillsHostID|TestSignInWithoutHostID'`.)*
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
  *(2026-11-15: the same for `acpext/v0.5.0-rc.1`, which stays public whatever follows; a fault in it goes forward as
  `acpext/v0.5.0` or `v0.5.1`. Before the candidate is tagged, the branch's commit is reverted or amended.)*
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
* *(2026-11-13, later.)* The owner's ruling on who reads the host id, "0051 P6 step 1a": `internal/cli` fills
  `SignInOptions.HostID` from `auth.HostID` (`internal/auth/hostid.go:23`, on `main` since `5147e09`) on the terminal
  path (`signInTo`) and the ACP path (step 4's `SignIn`), and `wizard.ResolveCredential` is called through a package
  variable; `TestHostIDReachesTheEngines` and `TestCLIFillsHostID`, each with the plant "the id left empty"; the
  Verification. P6's prerequisites, read again against `5147e09`, hold. What a sign-in does when the host id cannot
  be had is not ruled. 0051-MADR needs no note: D3 already has `signin.Options.HostID` from `<state>/host-id`, and
  0052-MADR D4 passes the id on every plan sign-in.
* *(2026-11-13, later still.)* The owner's ruling on the host id's failure case, "Proceed, warn once", option (a):
  recorded in P6 step 1a with the three options as put; the sign-in proceeds with no host id and one warning, and no
  sign-in fails over the host-id file; `TestSignInWithoutHostID`, its plant "the error returned instead of warned";
  the Verification.

### P2 record (2026-11-14)

Run by a mainstream model at medium effort, on the owner's approval of this PLAN's P2, from `main`. Times are the
session machine's local time; the Linux machine's clock is UTC, shown as local in brackets.

**C1, the start state.** Branch `main`, `HEAD` and `origin/main` both `35fda07`, the tree clean (`git status
--short` printed nothing). `go.mod` required go-llmprovider-sdk `v1.29.0`, acpext `v0.3.0`, `go 1.27.4`
(0054-PLAN's moves already in).

**The gate before P2,** from a scratch directory, through the proxy: `go list -m
github.com/maccavelli/go-llmprovider-sdk/authcontract@v0.1.0` printed `github.com/maccavelli/go-llmprovider-sdk/authcontract v0.1.0`
(exit 0); `go list -m github.com/maccavelli/go-llmprovider-sdk@v1.31.0` printed `github.com/maccavelli/go-llmprovider-sdk v1.31.0`
(exit 0). Met.

1. **The requirement move.** `go get github.com/maccavelli/go-llmprovider-sdk@v1.31.0` (`go: upgraded
   github.com/maccavelli/go-llmprovider-sdk v1.29.0 => v1.31.0`), then `go mod tidy`, both exit 0. The `go.mod` diff is
   exactly two lines: the SDK's `v1.29.0` to `v1.31.0` in the direct block, and
   `github.com/maccavelli/go-llmprovider-sdk/authcontract v0.1.0 // indirect` added. No other requirement moved; the
   `go` line stays `1.27.4`. `go.sum` gains the SDK's `v1.31.0` pair and the contract's `v0.1.0` pair, and loses the
   `v1.29.0` pair. No deviation.
2. **`internal/archtest/check.go`.** `authcontractPath` (a constant with its doc), and rule 4's check skips an edge to
   a path under it from `acpserver` and `internal/cli`; every other edge to the SDK, and to the contract from any other
   package, is refused as before. Rule 1 is not edited.
3. **`internal/archtest/check_test.go`.** `TestImportRules` gains six cases: the three planted breaks (`rule 1 agent
   authcontract`, `rule 4 acpclient authcontract`, `rule 4 acpserver signin`, the last importing
   `go-llmprovider-sdk/llmprovider/x/signin`) and the three allowed imports (`llm/provider`, `acpserver`,
   `internal/cli`, each importing the contract, rule 4).
4. **`scripts/check_libs.py`.** `Library` gains `tag_prefix` and `name`; `name` defaults to `repo` (in
   `__post_init__`) and is the `--lib` choice and the key of the report, the clone, the logs and the API paths, while
   `repo` stays the key of the git URL and the advisories. `newest(tags, major, prefix)` keeps the tags under the
   prefix and strips it. The entry `Library("go-llmprovider-sdk", …, path=…/authcontract, tag_prefix="authcontract/",
   name="authcontract")` follows the SDK's. `--self-test` gains two cases: "the contract's tags and the SDK's are read
   apart" (`v1.40.0`, `authcontract/v0.2.0` give `v0.2.0` and `v1.40.0`) and "the --lib choices are unique and name the
   SDK and its contract". The module docstring names the contract.
5. **Seen failing (C2), on a scratch copy of the tree** (`<TMP>/tree`, the working tree untouched, `HOME`, `XDG_*` and
   `GOBBLE_HOME` in the scratch, `GOPROXY=off`):
   * archtest, `go test -count=1 ./internal/archtest/`, each exit 1; unbroken, exit 0:
     * `check.go` as at `HEAD` (the tree before the phase): `--- FAIL: TestImportRules/rule_4_acpserver_may_import_authcontract`,
       `check_test.go:268: unexpected edge: rule 4: github.com/maccavelli/gobble-cli/acpserver imports github.com/maccavelli/go-llmprovider-sdk/authcontract`,
       and the same for `rule_4_cli_may_import_authcontract` (`internal/cli`). The three break cases and the
       `llm/provider` case pass there: the refusals and that allowance already existed.
     * plant "rule 1 lets the contract through" (`return !under(rel, "llm/provider") && !pathUnder(imp, authcontractPath)`
       in `rule1Forbidden`): `--- FAIL: TestImportRules/rule_1_agent_authcontract`, `check_test.go:271: missing
       "go-llmprovider-sdk/authcontract" in:` (an empty violation list).
     * plant "rule 4's exception for every package" (the exception's condition made `true`): `--- FAIL:
       TestImportRules/rule_4_acpclient_authcontract`, `missing "go-llmprovider-sdk/authcontract" in:`.
     * plant "rule 4's exception admits the whole SDK" (its path test made the SDK's root): `--- FAIL:
       TestImportRules/rule_4_acpserver_signin`, `missing "go-llmprovider-sdk/llmprovider/x/signin" in:`.
   * `check_libs.py --self-test`, each exit 1; unbroken, exit 0 (17 cases ok):
     * plant "the prefix ignored" (`own = set(tags)` in `newest`): `self-test the contract's tags and the SDK's are
       read apart: FAIL`.
     * plant "the contract entry given no name": `self-test two libraries at one version do not share API paths:
       FAIL` and `self-test the --lib choices are unique and name the SDK and its contract: FAIL`.
     * extra, `api_paths` keyed by `repo` again: `self-test two libraries at one version do not share API paths: FAIL`.
     The existing case "two libraries at one version do not share API paths" compares `LIBRARIES[1]` and `[2]`, now the
     SDK and its contract, which one repository holds.
6. **`check_libs.py --lib go-llmprovider-sdk` and `--lib authcontract`** (network). `check_library` clones the
   repository's **committed** state (`git clone <repo>`), so run on the working tree before the owner commits, the SDK
   entry ends `go-llmprovider-sdk modules: FAIL new or changed module: github.com/maccavelli/go-llmprovider-sdk/authcontract`,
   exit 1 (the clone's `go.mod` still has `v1.29.0`; the contract is the move step 1 makes), every other check ok
   (19:05 to 19:09). That is **Finding 1** below. On a scratch clone of this repository with this phase's five files
   committed (commit made in the scratch only, hooks off), the verdicts are: `--lib go-llmprovider-sdk`, exit 0,
   19:11:20 to 19:16:45, "pinned v1.31.0, checking v1.31.0 … 0 failing checks"; `--lib authcontract`, exit 0, 19:16:45
   to 19:18:08, "pinned v0.1.0, checking v0.1.0 … 0 failing checks". Each: get and tidy, version, modules, build, vet
   (linux, darwin, windows), test, `make staticcheck`, `make lint`, govulncheck and advisories ok; api "pinned at …;
   nothing to compare". `make staticcheck` is clean with the SDK's deprecations in place, as step 5 expected.
7. **Gates (C6),** each exit status taken from the file it was redirected to before anything read it; the Go files were
   last modified 19:07:11, `scripts/check_libs.py` 19:09:02, `go.mod` and `go.sum` 19:22:41 (`make preflight`'s tidy
   check restores them with the same bytes), all before every gate below:

   | Gate | Where | Start to end | Exit |
   | :--- | :--- | :--- | :--- |
   | `make pre-add-check FILES="internal/archtest/check.go internal/archtest/check_test.go"` | this Mac | 19:18:24 to 19:18:32 | 0 ("2 file(s) clean (gofmt, golint, govulncheck)") |
   | `make preflight` | this Mac, under `run-heavy` | 19:22:39 to 19:32:16 (queued from 19:18:42 behind another repository's gate) | 0 ("preflight passed"; includes `make archtest`, `staticcheck` on three GOOS, `lint`, `govulncheck`, `go mod tidy` and acpext's `-diff`) |
   | `make race` | this Mac, under `run-heavy` | 19:32:16 to 19:33:08 | 0 (most packages `(cached)`, the race binaries' inputs unchanged since preflight; acpext `(cached)`) |
   | `make check-test-isolation` | this Mac, under `run-heavy` | 19:33:08 to 19:33:32 | 0 ("files written above the temp roots: 0") |
   | `go test -count=1 ./...`, `CGO_ENABLED=1 go test -race -count=1 ./...`, `go -C acpext test -count=1 ./...` | the Linux machine itself (`pick-host` excluded the WSL distro for a race run), scratch copy, `run-on-host --need linux --race` | 19:22:28 to 19:26:20 (UTC 01:22:28 to 01:26:20) | 0, 0, 0 |
   | the same three | the Windows laptop, scratch copy, `run-on-host --need windows --race`, command `sh -c` | 19:22:36 to 19:24:27 | 0, 0, 0 |
   | `python3 -B scripts/check_libs.py --self-test` | this Mac | 19:34:26 to 19:34:26, after the script's last edit (19:09:02) | 0 (17 cases ok) |

   The identifier scan and `make check-records markdownlint` ran last, over the staged diff with this record in it:
   `make check-records markdownlint` exit 0 ("Summary: 0 issues in 0 files"), run again after the record's last edit; the scan of the
   diff's added lines for the local account name, the host names, home, temporary and dev-host scratch paths,
   run-heavy ticket shapes and mail addresses found nothing (`grep` exit 1), and found the one line of a copy with a
   planted home path, host name and ticket (exit 0).
8. **Docs step.** `grep` over `README.md`, `docs/` (outside `docs/decisions/`), `AGENTS.md`, `scripts/` and the
   Makefile for the SDK pin, the import rules and `check_libs.py`'s choices. Two statements are false once this lands
   and lie outside the phase's files; they are open for the commit that carries them: `README.md:163` ("go-llmprovider-sdk
   `v1.29.0`") and `docs/architecture.md:192` (the same version, in the requirements table). Two more turn from true to
   false only with P4's first import, so they wait for it: `README.md:164` ("Only `llm/provider` imports it") and
   `docs/architecture.md:164` (rule 4's sentence); `docs/architecture.md:214`'s `make check-libs` sentence stays true.
   This repository keeps no changelog.
   *(2026-11-14, later: the first two are fixed in this commit, by deviation 2.)*

**Finding 1 (not foreseen by the PLAN).** `scripts/check_libs.py` takes its clone from the committed state, so P2's own
`go.mod` move, uncommitted, makes the SDK entry's `modules` check fail with the contract as "new or changed module".
The check is right on the committed tree and wrong about an uncommitted one: the step 5 acceptance reads "0 failing",
which holds on the committed phase (step 6). Resolutions, recommended first: (a) the owner commits P2, then reruns both
`--lib` commands, which is what the verdicts above rehearse on a scratch commit (no cost beyond the rerun); (b) the
script clones the working tree, which a change to the script's design would make and which this PLAN does not name;
(c) nothing: P2 is accepted on the scratch rehearsal. Not a workaround of the gate: the gate is not edited, and the
failing run is reported above.

**Observation (not a deviation).** The contract's entry reads the repository's security advisories, whose version ranges
are the SDK's; a range written for the SDK's `v1` would read as covering the contract's `v0.1.0`. None exists today
(the check passes). Filtering a vulnerability by its package would close it; this PLAN does not name it.

Acceptance: archtest and `check_libs.py --self-test` pass, each plant failed (items 5); `make preflight` passes with no
code change but `internal/archtest/check.go`, its test and `scripts/check_libs.py` (item 7); `go.mod` names the contract
`// indirect` only (item 1). Met, subject to the owner's word on Finding 1 for the two `--lib` runs. Staged with this
record; the owner commits. P2a and P3 are not started.
*(2026-11-14, later: "Met, subject to the owner's word on Finding 1" is resolved by deviation 1 below: the two `--lib`
runs are repeated against the owner's local commit, and the push waits for both to report 0 failing. Until then
that part of the acceptance is open, not met.)*
*(2026-11-14, later still: met. Both `--lib` runs reported 0 failing against `7120c3b`; see deviation 1.)*

**Deviations of P2 (2026-11-14, the owner's rulings, relayed by the cross-repository coordinator).**

1. **Order of step 5 against the commit (Finding 1).** The owner chose "Commit locally, check, then push": "I commit
   P2 without pushing, the two checker runs are repeated against that commit, and I push only if both report 0
   failing. If either fails, the commit stays local and it comes back to you. The gate is run as designed; only the
   order changes." Rejected: accepting the scratch rehearsal; changing the checker. The gate is not edited. The
   results of the two runs against the commit are recorded here when they are made (not yet made).
   *(2026-11-14, later: the commit was not the owner's own. P2 is committed locally as `7120c3b`, one commit ahead of
   `origin/main` (`35fda07`), made by the cross-repository coordinator under the owner's standing approval of
   2026-11-12 for commits of approved phases; so "I commit P2" in the option's text above, and "the owner's local
   commit" in the note under P2's acceptance, read: the coordinator. The two runs, each through `run-heavy`, one after
   the other, the exit status taken before the output was read, against `7120c3b`: `python3 -B scripts/check_libs.py
   --lib go-llmprovider-sdk --keep`, exit 0, 21:03:21 to 21:04:18, "0 failing checks"; `python3 -B
   scripts/check_libs.py --lib authcontract --keep`, exit 0, 21:04:18 to 21:05:13, "0 failing checks". Each ran get and
   tidy, version, modules, build, vet on linux, darwin and windows, test, staticcheck, lint, govulncheck and advisories,
   all ok, and api "pinned at ...; nothing to compare" (v1.31.0 and v0.1.0). The clone each checked was at `7120c3b`:
   `--keep` left the scratch clones, and `git rev-parse --short HEAD` in each printed `7120c3b`, its origin this
   repository. The scratch was then removed. The push is the coordinator's, after this record is committed.)*
2. **Two version statements join P2's scope.** The owner chose "In this commit": "Add the two lines to P2's scope by a
   dated deviation and correct them to v1.31.0 before the commit, so the docs move with the code. The Markdown checks
   are rerun; no code changes." Rejected: leaving them to gobble's docs plan. `README.md:163` (Stack, "Models") now
   reads "go-llmprovider-sdk `v1.31.0`, behind gobble's own `llm` facade. Only `llm/provider` imports it."; the
   `docs/architecture.md:192` row of the "Dependencies" table, which lists the direct requirements in `go.mod`, reads
   `github.com/maccavelli/go-llmprovider-sdk` `v1.31.0`. The table claims direct requirements only, and the contract is
   `// indirect`, so it gains no row. `README.md:164` and `docs/architecture.md:164` ("Only `llm/provider` imports
   it") stay: they turn false with P4's first import.
3. **An open item, and its step.** The owner chose "Record it, fix in its own step": "Note it in the plan as an open
   item and fix it as a small separate step with its own test and planted break (filter each advisory by the package
   it names), before the plan's close-out. P2 stays as the plan wrote it." Rejected: fixing it in P2; recording it
   only. The step is P7a, added below; it is not done.
4. **After the deviations' file changes,** `make check-records markdownlint`, `make preflight` (under `run-heavy`) and
   the identifier scan of the staged diff are rerun, below. No Go or Python file changed after the gates of item 7:
   `go.mod` and `go.sum` 19:22:41 (the same bytes), `internal/archtest/check.go` and `check_test.go` 19:07:11,
   `scripts/check_libs.py` 19:09:02; so the race and cross-host runs stand. Results: `make preflight` under
   `run-heavy`, exit 0 ("preflight passed"), 20:54:53 to 21:02:05; `make check-records markdownlint` exit 0 (20:54:50
   to 20:54:53, and again after this line); the identifier scan of the staged diff's added lines, nothing found
   (`grep` exit 1).


### P2a record (2026-11-14)

Run by a mainstream model at medium effort, on the owner's approval of this PLAN's P2a, from `main`. Times are the
session machine's local time. Staged, not committed; the owner (or the coordinator under the standing approval) commits.

**C1, the start state.** Branch `main`, `HEAD` and `origin/main` both `1e0a912`, the tree clean. `go.mod` requires
go-llmprovider-sdk `v1.31.0`, `authcontract v0.1.0` indirect (P2's `7120c3b`).

**Step 0, the sites at the base.** MADR D9's N1 to N11 read at `1e0a912`. Nothing changed in substance. Lines that
moved since `c47f4a7`, where `llm/provider/login.go` gained three lines above N7 (no commit since `35fda07` touched
`llm/` or `internal/cli/`):

| Site | MADR's place | Place at `1e0a912` |
| :--- | :--- | :--- |
| N7 `RevokeOAuthSession` call | `login.go:104` | `login.go:107` |
| N8 `l.session(string(p), a.s, nil, nil)` | `login.go:144` | `login.go:147` |
| `LoginOptions` | `login.go:54-56` | `login.go:57-59` |
| `HasLogin` (N11's "keeps a login for Grok only") | `login.go:67` | `login.go:70` |

Unmoved: N1 `signin.go:184`, N2 `signin.go:181`, N4 `provider.go:194`, N5 `list.go:71`, `:92`, N6 `model.go:157`, the hook
`agent.go:101-107` (`:102` the options line), `appName` `agent.go:26`, N7's caller `auth.go:302`, N9 `piimport.go:203`, N10
`auth.go:431`, `:548` and `acpserver/auth.go:59`.

The SDK at `v1.31.0` (module cache, `go list -m -f '{{.Dir}}'`):

* `wizard.Options.Identity` (`wizard/configure.go:139-151`): "Identity names the consumer's application to the identity
  providers on every sign-in the wizard runs ... The zero value names this module."
* `(*auth.OAuthSession).UseIdentity` (`llmprovider/auth/oauth_session.go:657`): `func (s *OAuthSession) UseIdentity(id Identity)`;
  a zero id changes nothing, and the first name given wins.
* `catalog.CachedModel` (`llmprovider/catalog/lookup.go:121-160`) has the line that hands the name over before the
  token is read, `lookup.go:141`, immediately before `lookup.go:142`'s `src.Token(ctx)`:

  ```go
  		nameSession(src, cfg)
  		if token, err := src.Token(ctx); err == nil {
  ```

  `nameSession` (`llmprovider/catalog/discovery.go:486-493`) calls `UseIdentity(auth.Identity{Name: cfg.clientName,
  Version: cfg.clientVersion})` on a source that takes one, when `WithClientInfo` named an application other than the
  module. `LookupModel`'s `listingModel` (`lookup.go:193`) and `CachedFacts` (`lookup.go:179`) have it too. So the
  SDK's amendment shipped in `v1.31.0`; no deviation.

**What changed (staged).**

1. `llm/provider/signin.go`: `identityOf(o Options) auth.Identity` (documented), and `SignIn` sets
   `wizard.Options.Identity` to `identityOf(o.Options)`; `SignIn`'s doc says every request names `ClientName` and
   `ClientVersion`. The `auth` import is added.
2. `llm/provider/model.go`: `ModelInfoWith` passes `o.ClientName` and `o.ClientVersion` to `modelInfoFrom` and so to
   `lookupModelWith`, which adds `llmprovider.WithClientInfo(clientName, clientVersion)` to the `catalog.CachedModel`
   call's options when `clientName` is set; `ModelInfoContext` passes none. Docs of `ModelInfoWith`, `modelInfoFrom` and
   `lookupModelWith` name the two fields. No `UseIdentity` is called there.
   `internal/cli/agent.go`: the `ModelInfo` hook's `provider.Options` gains `ClientName: appName, ClientVersion:
   identity().Version`; `modelFacts`' doc says so.
3. `llm/provider/login.go`: `LoginOptions.ClientName`, `ClientVersion` (documented as `Options`' are); `Logout` builds the
   session, calls `UseIdentity` on it, then `RevokeOAuthSession`; `Logout`'s doc says so. `internal/cli/auth.go`: the
   logout passes `LoginOptions{ClientName: appName, ClientVersion: identity().Version}`.
4. Tests: `TestSignInNamesGobble` (`signin_test.go`, with the shared `uaRecorder`, `wantNamed`, `wantUnnamed`),
   `TestStoredLoginNamedForRefresh` and `TestLogoutRevokesAsGobble` (`login_test.go`), `TestCLIPassesGobblesIdentity`
   (`auth_test.go`). No test asserts an `originator`.

**Names in the record, not deviations.**

* `lookupModelWith` and `modelInfoFrom` each gained two parameters (`clientName`, `clientVersion`), as step 2 words it
  ("passes ... on to `lookupModelWith`"). The four calls of `lookupModelWith` in the existing `llm/provider/model_test.go`
  (`:57`, `:113`, `:124`, `:127`) needed `"", ""` added for them: a compile change only, with no assertion touched. That
  file is one the phase's tests live beside; the PLAN lists "the test files the four tests live in", and this is the
  one named change to an existing test. Every other existing test of `llm/provider` and `internal/cli` is unchanged and
  passes.
* `TestStoredLoginNamedForRefresh` records its requests at the in-process issuer's handler (`uaRecorder.atIssuer`),
  and gives the options' `HTTPClient` a redirecting transport only. The table says each capturing transport is the
  options' `HTTPClient`; for `ModelInfoWith` that cannot see the refresh: the SDK refreshes a login's session that
  `CachedModel` names through the session's own client (the first run, with a recording `HTTPClient`, saw the listing's
  `GET /v1/models` and no `POST /oauth2/token`: `no refresh among [{method:GET path:/v1/models userAgent:gobble/1.2.3-test (...`).
  A handler at the issuer sees every request whatever sent it, which is the same evidence.
  *(2026-11-14, later: this is a deviation, not a name; see "Deviations of P2a", 1.)*
* `TestSignInNamesGobble`'s `Identity`-omitted plant also makes the Grok referrer `go-llmprovider-sdk`, the SDK's.

**Seen failing (C2), on a scratch copy of the tree** (`<TMP>/tree`, no `.git`, the working tree untouched; `HOME`,
`USERPROFILE`, `XDG_*` and `GOBBLE_HOME` in the scratch; `GOPROXY=off`; `go test -count=1 -run <test> <pkg>`). Unbroken:
`llm/provider` (the three) exit 0, `internal/cli` (the fourth) exit 0. Each plant, exit 1:

* `TestSignInNamesGobble`
  * `Identity: identityOf(o.Options),` removed: `signin_test.go:386: the device sign-in: GET /.well-known/openid-configuration sent User-Agent "go-llmprovider-sdk/v1.31.0 (darwin; arm64) go-llmprovider-sdk/v1.31.0", want gobble/1.2.3-test ( ... go-llmprovider-sdk/`
    (and the same for `POST /oauth2/device/code`, `POST /oauth2/token`, `GET /jwks`); `signin_test.go:392: the device request's form "client_id=...&referrer=go-llmprovider-sdk&scope=...", want referrer=gobble`;
    `signin_test.go:416: the authorize URL "https://auth.x.ai/oauth2/authorize?client_id=...&referrer=go-ll...` has referrer `go-llmprovider-sdk`, want gobble.
  * `identityOf` with an empty `Version`: `signin_test.go:386: the device sign-in: GET /.well-known/openid-configuration sent User-Agent "gobble/(devel) (darwin; arm64) go-llmprovider-sdk/v1.31.0", want gobble/1.2.3-test ( ... go-llmprovider-sdk/`
    (and the same for the other three requests).
* `TestStoredLoginNamedForRefresh`
  * `WithClientInfo` left out of `lookupModelWith`: only `ModelInfoWith_named_true` fails (`New` and `List` stay named, as the table
    says): `login_test.go:421: ModelInfoWith's refresh: POST /oauth2/token sent User-Agent "go-llmprovider-sdk/v1.31.0 (darwin; arm64) go-llmprovider-sdk/v1.31.0", want gobble/1.2.3-test ( ... go-llmprovider-sdk/`;
    `login_test.go:423: ... its background listing among them: GET /v1/models sent User-Agent "go-llmprovider-sdk/v1.31.0 ...`.
  * `ModelInfoWith` passing `"", ""` to `modelInfoFrom` (both calls): the same two failures, `ModelInfoWith_named_true` alone.
* `TestLogoutRevokesAsGobble`
  * `UseIdentity` removed: `login_test.go:458: the named logout: GET /.well-known/openid-configuration sent User-Agent "go-llmprovider-sdk/v1.31.0 (darwin; arm64) go-llmprovider-sdk/v1.31.0", want gobble/1.2.3-test ( ... go-llmprovider-sdk/`
    and the same for `POST /oauth2/revoke`.
  * `UseIdentity` after `RevokeOAuthSession`: the same two failures.
* `TestCLIPassesGobblesIdentity`
  * `auth.go`'s `provider.LoginOptions{}`: `auth_test.go:692: auth logout passed name "" and version "", want "gobble" and "0.0.0-dev"`.
  * the hook built without `ClientName`: `auth_test.go:692: the ModelInfo hook passed name "" and version "", want "gobble" and "0.0.0-dev"`.
  * extra, `internal/cli/login.go`'s `provider.Options{ClientName: ...}` made `provider.Options{}` (a pass-through that
    existed before the phase, which the test now pins): `auth_test.go:692: auth login passed name "" and version "", want "gobble" and "0.0.0-dev"`.

On the tree before the phase (`HEAD`'s five sources and `model_test.go`, with the new tests), where it compiles:
`TestSignInNamesGobble` fails, exit 1, with the same lines as the `Identity` plant (the sign-in's four requests named
`go-llmprovider-sdk/v1.31.0`, `referrer=go-llmprovider-sdk`, the authorize URL's referrer); `TestStoredLoginNamedForRefresh`
fails, exit 1, only in `ModelInfoWith_named_true`, as under the plant. `TestLogoutRevokesAsGobble` and
`TestCLIPassesGobblesIdentity` do not compile there (`unknown field ClientName in struct literal of type LoginOptions`;
`revoked.ClientName undefined (type provider.LoginOptions has no field or method ClientName)`), so the package's
other tests could not run until the first was dropped from the scratch copy.

**Gates (C6),** each exit status taken from the file it was redirected to before anything read it. The Go files were last
modified 21:24:37 (`login.go`, `model.go`, `agent.go`, `auth.go`), 21:24:56 (`signin.go`), 21:24:42 (`model_test.go`),
21:26:58 (`signin_test.go`), 21:27:46 (`login_test.go`), 21:27:27 (`auth_test.go`), all before every gate below:

| Gate | Where | Start to end | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check FILES=` the nine changed Go files | this Mac | 21:29:38 to 21:30:07 | 0 ("9 file(s) clean (gofmt, golint, govulncheck)") |
| `make preflight` | this Mac, under `run-heavy` | 21:30:30 to 21:37:58 | 0 ("preflight passed") |
| `make race` | this Mac, under `run-heavy` | 21:37:58 to 21:38:41 | 0 (`internal/cli` 35.2 s and `llm/provider` 12.1 s ran, not cached) |
| `make check-test-isolation` | this Mac, under `run-heavy` | 21:38:41 to 21:39:02 | 0 ("files written above the temp roots: 0") |
| `go test -count=1 ./...`, `CGO_ENABLED=1 go test -race -count=1 ./...`, `go -C acpext test -count=1 ./...` | the Linux machine itself (`pick-host` excluded the WSL distro for a race run), scratch copy, `run-on-host --need linux --race`, `sh -c` | 21:30:30 to 21:34:23 | 0, 0, 0 |
| the same three | the Windows laptop, scratch copy, `run-on-host --need windows --race`, `sh -c` | 21:30:30 to 21:32:38 | 0, 0, 0 |

C6 names the two cross-host runs for P2, P4, P6 and P8, not P2a. This phase edits `llm/provider` and `internal/cli`,
whose tests start loopback servers and a process-wide SDK cache, so both were run and both are recorded: they are extra,
not required by C6 for this phase. Heavy runs on this Mac went through `run-heavy` one at a time. The scratch plant
runs and one `-race` run of the three provider tests (before the plants) were single-package runs on this Mac.
*(2026-11-14, later: that `-race` run was a process deviation. By the global rules any `-race` run is a heavy run and
goes through `run-heavy` or `run-on-host`; this one did not. It is not repeated; see "Deviations of P2a", 3.)*

The identifier scan and `make check-records markdownlint` ran last, over the staged diff with this record in it; see the
results line at the end of this record.

**Acceptance.**

* The four tests pass and each plant failed, above. Met.
* Outside tests, `RevokeOAuthSession` is called only in `Logout`, after `UseIdentity`, and `UseIdentity` is called
  nowhere else: `grep -rn 'RevokeOAuthSession\|UseIdentity' --include=*.go .`, test files left out, finds
  `llm/provider/login.go:115` (`s.UseIdentity(...)`), `:116` (`auth.RevokeOAuthSession(ctx, s)`) and a doc comment at
  `llm/provider/model.go:156` ("UseIdentity itself", saying gobble calls none). Met.
* `go.mod` and `go.sum` unchanged from P2: `git diff --stat 1e0a912 -- go.mod go.sum` printed nothing. Met.

**Docs step.** `grep -rniE 'user-agent|referrer|client ?name|ClientInfo|revok|originator'` over `README.md`, `AGENTS.md`
and `docs/` outside `docs/decisions/`, and a read of `docs/guides/credentials.md` and the `llm/provider` row of
`docs/architecture.md`. Nothing states what name gobble sends on a sign-in, a refresh, a lookup or a revocation, so
nothing is made false; the user-facing description is missing, not wrong, and the credentials guide and the
architecture row are in the PLAN's docs phase's files, not this one's. This repository keeps no changelog.
*(2026-11-14, later: superseded by C9 and "Deviations of P2a", 2: the three documents are fixed in this phase.)*
The doc comments of every name added or changed are in the phase's files and written (items 1 to 3).

**Not verified.** A live refusal of the name by xAI is the owner-run L1's (P8), not this phase's. The Windows laptop's and
Linux machine's results are from scratch copies of the working tree, not of a commit.

**Deviations of P2a (2026-11-14, the owner's rulings on the hand-back, relayed by the cross-repository coordinator).**

1. **The capture in `TestStoredLoginNamedForRefresh`.** The step 4 table says "every capturing transport given as the
   options' `HTTPClient`". The test records at the in-process issuer's handler instead (`uaRecorder.atIssuer`, in
   `signin_test.go`), and gives the options' `HTTPClient` only a redirecting transport. The owner chose "Accept, as a
   recorded deviation": "Keep the issuer-side recording for that one test and relabel it in the plan as a dated
   deviation with the evidence (the failed first attempt, and the line that builds the session with no client). No code
   changes; it observes every request, so the evidence is at least as strong." Rejected: "Make the session use the
   options' client". Evidence: the first attempt, with a recording `HTTPClient`, failed `ModelInfoWith_named_true` with
   `login_test.go:410: ModelInfoWith: no refresh among [{method:GET path:/v1/models userAgent:gobble/1.2.3-test (darwin; arm64) go-llmprovider-sdk/v1.31.0 form:}]`:
   the listing went through the options' client and the refresh did not. The line that builds a stored login's session
   with no client is `llm/provider/provider.go:107`, `return l.session(id, o.Logins, nil, o.Logger), nil`, in
   `Options.login`, which `ModelInfoWith`, `New` and `List` all use; `(*Login).session` sets `HTTPClient` from that
   argument (`login.go`), so the session has none, and the SDK refreshes it through its own client. `grep -rn
   HTTPClient --include=*.go .` with the test files and `llm/provider/` left out finds nothing: no non-test code outside
   `llm/provider` passes a custom `HTTPClient` to these options.
2. **Three user documents join P2a's scope (C9).** The owner chose "Always in the same commit" (C9 above). The docs step
   found three places that did not say what name gobble sends, which this phase makes true; each is now added, checked
   against the code and the four tests: the sign-in, a stored login's refresh, a model lookup and a sign-out's discovery
   and revocation name gobble and the build's version in their User-Agent; the sign-in's authorize and device requests
   carry `referrer=gobble` (`TestSignInNamesGobble`); gobble sends no `originator` (0051-MADR D9, N11). Nothing is said
   of a vendor accepting the name: that is P8's owner-run probe.
   * `README.md`, the Credentials bullet of the stack list: three lines after "copies Pi's entries.".
   * `docs/guides/credentials.md`, "Signing in": a paragraph after the Kilo bullet; and the logout paragraph's sentence.
   * `docs/architecture.md`, the `llm/provider` row of the packages table: three sentences after "a `LoginStore` adapted
     to the SDK's token store.".
   The rows of the docs step are in the hand-back. The Kilo sign-in's requests are said nowhere: only Grok's were seen
   under test.
3. **A `-race` run outside the queue.** The single-package `-race` run of the three provider tests on this Mac, before
   the plants, did not go through `run-heavy` or `run-on-host`, which the global rules require of any `-race` run. It
   is recorded here as a process deviation and is not repeated; the `-race` evidence of the phase is `make race` under
   `run-heavy` and the Linux machine's and the Windows laptop's `run-on-host --race` runs.

**Results of the last gates (after the deviations' file changes).** No Go file changed after the gates above: the Go files'
last-modified times are those listed there (21:24:37 to 21:27:46), all before the race and cross-host runs, so
`make race`, `make check-test-isolation` and the Linux machine's and the Windows laptop's runs stand. Reran, each exit
status taken from its log before anything read it: `make preflight` under `run-heavy`, exit 0 ("preflight passed"),
22:03:41 to 22:10:57 (tracked Markdown changed; preflight lints it); `make check-records markdownlint`, exit 0
("Summary: 0 issues in 0 files"), run once before and once after this paragraph; the identifier scan of the staged
diff's added lines for the local account name, the host names, home, temporary and dev-host scratch paths, run-heavy
ticket shapes and mail addresses found nothing (`grep` exit 1), and found the one line of a copy with planted values
(count 1). The only text added after preflight began is this paragraph.

### P3 record (2026-11-15)

Run by a mainstream model at medium effort, on the owner's approval of this PLAN's P3 (steps 1 to 6), from `main`, then
moved to the branch `p3-acpext-v0.5.0` by the owner's ruling below. Times are the session machine's local time. Staged,
not committed; the coordinator commits on the branch under the owner's standing approval. Step 7 is the owner's, and
the tag waits for the owner's own word.

**C1, the start state.** Branch `main`, `HEAD` `9e21c8f`, the tree clean, `origin/main` equal to it, CI green on it.
`go.mod` requires acpext `v0.3.0`. The latest `acpext/` tag is `acpext/v0.4.0` (commit `d6c10a5`). The owner's answers
to QB and QC are in the MADR, "Questions answered by the owner (2026-11-10)": "The owner chose the recommended option of
each question below on 2026-11-10: QB (a), QC (i), QD (a) and QE (a), and O2 (a)"; QB (a) is "acpext's `ErrorKind` and
`RetryCause` constants take the contract's values at `acpext/v0.5.0`", and QC (i) is "provider list only, facts from
the contract (D7)". Neither is (b) nor (ii), so the phase runs as written.

**The kinds table.** Read from go-llmprovider-sdk `0073-MADR-publish-an-auth-contract-module.md` D3 and compared with
`authcontract@v0.1.0`'s `errorkind.go` in the module cache: the same eighteen values in the same order (canceled,
timeout, denied, expired, auth, quota, rate_limited, not_permitted, workspace, context_overflow, empty_answer,
incomplete, invalid_request, invalid_provider, offline, unavailable, unsupported, other). The table and the tagged code
agree. The list is copied into `acpext/types_test.go` (`contractKinds`); acpext imports no SDK.

**What changed (staged).**

1. `acpext/auth.go`: `AuthenticateMeta{Method string; Fresh bool; Remote bool}`, JSON `method`, `fresh`, `remote`, each
   `omitzero`; `MetaMethods = "methods"` and `MetaStep = "step"`; documented. `AuthProvider` unchanged.
2. `acpext/types.go`: `KindRateLimited`, `KindContextOverflow`, `KindEmptyReply`, `KindNotPermitted` and
   `RetryRateLimited`, `RetryEmptyReply`, `RetryNotPermitted` respelled (`rate_limited`, `context_overflow`,
   `empty_answer`, `not_permitted`); `RetryEmpty` removed; `KindDenied`, `KindExpired`, `KindInvalidRequest`,
   `KindInvalidProvider` added, so the kinds are the contract's eighteen; `ErrorData.Consent string`, JSON `consent`,
   `omitzero`, between `Command` and `Error`; the docs of `RetryCause`, `ErrorKind` and `ErrorData` say so.
3. `acpext/doc.go`: the first paragraph names `authenticate`'s `_meta.gobble`; the failed-turn paragraph names the
   contract's `ErrorKind` as the vocabulary, `Consent` and `rate_limited`.
4. Frames: `retry.start.notice.json` (`empty_answer`), `retry.start.provider.notice.json` and
   `prompt-rate-over-cap.error.json` (`rate_limited`); new `authenticate-meta.request.json` (`method`, `fresh` and
   `remote`) and `prompt-plan-consent.error.json` (`kind` `not_permitted`, `consent` `chatgpt_plan.background_use`,
   `provider` `openai`).
5. Tests: `roundtrip_test.go` (the two frames in the map; `TestErrorDataSortedMembers` lists `consent` between `command`
   and `error`; `TestKindsAndLimits` rewritten to the eighteen values, in the order of the old fourteen and then
   `denied`, `expired`, `invalid_request`, `invalid_provider`: its `kinds` and `wantKinds` lists, diff in the commit);
   `auth_test.go` (`TestAuthenticateMetaRoundTrip`: both codecs decode strictly, `method`, `fresh` and `remote`
   round-trip, an unknown member, a string or number `fresh` or `remote` and a number `method` are refused, `fresh` and
   `remote` false are not sent); `types_test.go`, new (`TestKindsAreTheContractsSpelling`: it reads the string constants
   of `ErrorKind` and of `RetryCause` from `types.go` with `go/parser`, so a constant missing from a list is still
   found; every value is `snake_case`; the `ErrorKind` set equals the table's eighteen; every `RetryCause` is in the
   table). The PLAN allows `types_test.go` "or puts its tests in" `roundtrip_test.go`; this is the first.
6. `docs/architecture.md`: one sentence in the acpext paragraph (see "Docs step").

**Deviations of P3.**

1. **`acpext/example_test.go` joins the scope (C9, dated 2026-11-15).** `ExampleRetryNotice` decodes `"cause":"rate-limited"`
   and compares it with `acpext.RetryRateLimited`; it prints `false` once the constant is respelled. An example is
   documentation. Changed to `rate_limited`; no assertion touched. Not named in the PLAN's scope or step 4.
2. **The root's tests read acpext's frames from the tree, so P3's commit alone fails two root tests.** Evidence: `make
   race` on the working tree, under `run-heavy`, this Mac, 12:08:11 to 12:08:57, exit 2:
   * `acpserver/errordata_test.go:203`, `TestPromptQuotaData/prompt-rate-over-cap.error.json`: the prompt error's data has
     `"kind":"rate-limited"`; the frame, read by `readFrame` (`errordata_test.go:75`, `../acpext/testdata/<name>`), says
     `rate_limited`;
   * `acpserver/retry_test.go:213`, `TestRetryNoticeServerMembers`: got `"cause":"rate-limited"`, want `"cause":"rate_limited"`
     (frame `retry.start.provider.notice.json`, read the same way).
   Every other package passed, and acpext's `-race` passed in the same run. Not pre-existing: CI was green on `9e21c8f`.
   Two more readers of `../acpext/testdata` (`internal/cli/limit_test.go:21`, `acpclient/prompterror_test.go:23`) read
   `prompt-quota.error.json`, which this phase does not change. The PLAN (Scope, and the coordinator's brief) said P3's
   `acpext/` changes "do not reach the root module's build". That holds for code (the root requires `v0.3.0`), not for the
   frames: **the coordinator's error, repeated from the PLAN's premise as fact; the finding refutes it for the frames.**
3. **The owner's rulings, in sequence.** Recorded as they happened.
   * The coordinator first offered two sequences: P3 on a branch with the tag there, and a small respelling step split
     out of P4 before `main` moves; or P3 and all of P4 on one branch.
   * The owner answered neither, in these words: "fix the shit, on main. in place. completely. no tag until the full
     repo codebase is stable and correct. why not this?"
   * The coordinator answered that this needs the root to build against the in-tree `acpext` without a tag, which
     `0042-MADR-acpext-extension-contract-module.md` ("The root requires acpext by tag. It never uses a `replace`
     directive, and no `go.work` is committed.") and `AGENTS.md:15` forbid, and asked whether the owner authorises a
     temporary `replace` as a dated exception. The owner chose "No: keep the tag-first rule": "0042 stands unchanged. Then
     main can only stay green by building P3 on a branch, tagging acpext/v0.5.0 early on that branch commit, and moving
     main once the root's respelling is in, which is the sequence you rejected." Rejected: "Yes: temporary replace, tag
     last".
   * The agent was asked whether the respelling (P4's steps 1, 4 and 4b) is separable. It was not: on a scratch copy with the
     in-tree `acpext` standing in for the tag (a `replace` in the scratch `go.mod` only, never in the tree), 37 packages
     passed and `internal/cli`'s `TestAcpextAuthProvidersAreTheRegistry` failed, 12:19:51 to 12:20:15, exit 1. Any
     requirement past `v0.3.0` brings in `acpext/v0.4.0`'s openai entry (`Login: true`, the bare command, commit
     `d6c10a5`), which the registry matches only once the plan is offered: P4 steps 2 and 5, with 0052-PLAN P3. P4's own
     header says so ("Committed with 0052-PLAN P3, as one commit"). **The coordinator had recommended a respelling-only
     step without reading that header: the coordinator's second error, beside its error about the frames.** Without
     step 2's import, `go mod tidy` still made `authcontract` a direct requirement, because the respelling's own
     code and tests import it (`llm/provider/list.go`, `TestErrorKindIsTheContracts`).
   * The owner's ruling on that finding: "Release-candidate tag, final tag last": "P3 is committed on a branch and, on your
     word, tagged acpext/v0.5.0-rc.1. P4 whole with 0052's consent phase is built on the branch against that candidate;
     main moves once, when every gate is green; the remaining phases continue on main. The final acpext/v0.5.0 is
     tagged only on your word when the plan's acceptance shows the repo stable, followed by one small commit moving the
     root from the candidate to the final tag. 0042's rule is kept. Cost: a candidate tag exists publicly, one extra
     commit at the end, and the branch's P3 commit alone fails two root tests." Rejected: "Final tag early, as the plan
     says"; "Revert the openai entry in v0.5.0". *(2026-11-15, the owner's ruling relayed by the coordinator, "Keep it on the branch until those phases land": the coordinator commits each approved phase on the branch `p3-acpext-v0.5.0` and pushes the branch; `main` stays at `9e21c8f` and is fast-forwarded only after 0051-PLAN P6 and 0052-PLAN P4 have landed on the branch and every gate is green, because until then a plan sign-in passes no host id and does not reuse its registered client. Consequence told to and accepted by the owner: CI runs on pushes to `main`, so until `main` moves the phases on the branch have only the local gates and the Linux and Windows runs behind them; `.github/workflows/ci.yml` also runs on `pull_request` and on `v*` tags, which a pull request from the branch would use.)* *(2026-11-15, later: the owner's ruling "Open a draft PR now", relayed by the coordinator. The coordinator's statement that CI runs only on pushes to `main` and tags was wrong: `.github/workflows/ci.yml` also runs on `pull_request`, which the review's finding showed. After P4 is committed and pushed the coordinator opens one draft pull request from `p3-acpext-v0.5.0` to `main`, so every later push to the branch runs CI; `main` does not move and nothing merges until the owner's word, and `main` is still fast-forwarded only after 0051 P6 and 0052 P4 land. The sentence above that the phases on the branch have only the local gates behind them holds only until that pull request is open.)*
   * Effect: the notes under C8, P3 step 7, P4's prerequisites and step 1, the new P8a, the Rollout and 0052-PLAN P3 (and
     its order line). **The branch's P3 commit, checked out alone, fails those two `acpserver` tests, by construction;
     the owner's accepted cost.**
4. **Does the tooling accept a pre-release tag of the nested module?** Yes; checked on a scratch copy, no tag created.
   * Go orders `v0.5.0-rc.1` after `v0.4.0` and before `v0.5.0` (semver: a pre-release is lower than its release, and
     higher than every earlier release). `golang.org/x/mod/semver.Compare` gave 1 for `v0.5.0-rc.1` against `v0.4.0`, -1
     against `v0.5.0`, and `IsValid` true. `@latest` of the module stays `v0.4.0` until a release tag is served, so a
     module that takes `@latest` is not moved by the candidate.
   * `scripts/check_libs.py` does not list acpext (`grep -c acpext scripts/check_libs.py` is 0): its `LIBRARIES` are the
     SDKs. Its `SEMVER` (`^v(\d+)\.(\d+)\.(\d+)$`) and `newest` skip pre-releases, which matters only for the SDKs'
     tags; `--self-test` exits 0.
   * `make check-acpext-deps` reads `go -C acpext list -deps -test`, not tags or `require`. `make apidiff` lists
     `git tag -l 'v*'` for `v1` or later, which no `acpext/` tag matches. `verify_build_metadata.py`'s `RELEASE_RE` reads
     `vX.Y.Z` tags only. `ci.yml` runs on `v*` tags and no `acpext/` tag matches. No `Makefile`, script or workflow reads
     the root's `require` line for acpext.
   * Not new: `make build`'s `git describe --tags` (`Makefile:91`) already sees `acpext/` tags.

**Seen failing (C2), on a scratch copy of the tree** (`<TMP>`, no `.git`; `HOME`, `USERPROFILE`, `XDG_*` and
`GOBBLE_HOME` in the scratch; `GOPROXY=off`; `go test -count=1 ./...` of acpext), run after the last Go change. Unbroken:
exit 0. Each plant, exit 1:

* `KindRateLimited` left `rate-limited`: `roundtrip_test.go:240: kind 2 is "rate-limited", want "rate_limited"`;
  `roundtrip_test.go:256: retry cause "rate_limited" has no kind of the same name`; `types_test.go:70: kind "rate-limited" is not
  snake_case`; and the `ErrorKind` constants differ from the contract's list (`types_test.go:75`).
* `KindDenied` removed: `FAIL ... [build failed]`, `./roundtrip_test.go:233:69: undefined: acpext.KindDenied` (the test
  lists name it). Plant 2b, the constant removed and the test's own references to it too: `TestKindsAreTheContractsSpelling`
  fails, the constants lacking `denied` against the contract's list.
* `AuthenticateMeta.Method` tagged `methodId`: `auth_test.go:51: encoding/json: decode the frame: json: unknown field
  "method"` (and the v2 codec's `unknown object member name "method"`); `TestFramesRoundTrip`: `authenticate-meta.request.json,
  encoding/json: decode: json: unknown field "method"`.
* `AuthenticateMeta.Fresh` tagged `force`: the same failures, `unknown field "fresh"`.
* `AuthenticateMeta.Remote` tagged `isRemote`: the same failures, `unknown field "remote"`.
* `ErrorData.Consent` declared after `Error`: `TestErrorDataSortedMembers`, `roundtrip_test.go:223: ErrorData encodes as
  {"attempts":3,"code":"c","command":"c","error":"e","consent":"chatgpt_plan.background_use",...`, want `consent`
  before `error`.
* `ErrorData.Consent` tagged `consentId`: `TestFramesRoundTrip`, `prompt-plan-consent.error.json, encoding/json: decode:
  json: unknown field "consent"` (and the v2 codec's); `TestErrorDataSortedMembers`, `"consentId":...`.

On the tree before the phase (HEAD's `acpext` with the new tests and frames added): it does not compile, `undefined:
acpext.AuthenticateMeta`, `unknown field Consent in struct literal of type acpext.ErrorData`, `undefined: acpext.KindDenied`.

**apidiff (step 5),** the root's `golang.org/x/exp/cmd/apidiff` built to scratch, export data of `git archive acpext/v0.4.0`
and of the tree's `acpext/`, exit 0, run after the last Go change.

* Incompatible changes: `KindContextOverflow`, `KindEmptyReply`, `KindNotPermitted`, `KindRateLimited`, `RetryEmptyReply`,
  `RetryNotPermitted`, `RetryRateLimited`: value changed; `RetryEmpty`: removed. Exactly step 2's constant values and
  `RetryEmpty`'s removal.
* Compatible changes: `AuthenticateMeta` (with `Remote`), `ErrorData.Consent`, `KindDenied`, `KindExpired`,
  `KindInvalidProvider`, `KindInvalidRequest`, `MetaMethods`, `MetaStep`. Step 1's additions, step 2's four kinds, and the
  two members of the 2026-11-11 note. Nothing on one side only. A wire change named here (0042-MADR D5).

**Gates (C6),** each exit status taken from the file its output was redirected to before anything read it. The changed
Go files were last modified at 11:21:11 (`types.go`, `example_test.go`), 11:21:28 (`auth.go`), 11:22:19 (`auth_test.go`,
`roundtrip_test.go`), 11:23:13 (`types_test.go`) and 11:24:22 (`doc.go`), before every gate below.

| Gate | Where | Start to end | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check FILES=` the seven changed Go files | this Mac | 11:24:57 to 11:25:07 | 0 ("7 file(s) clean (gofmt, golint, govulncheck)") |
| `go -C acpext test -count=1 ./...` and `go vet`, `run-on-host --need linux`, no `--race` (pick-host chose a WSL distro) | Linux WSL distro on the Windows laptop, scratch copy | 11:28:53 to 11:29:21 | 0 |
| `go -C acpext test -count=1 ./...`, then with `-race`, `run-on-host --need linux --race`, `sh -c` | the Linux machine, scratch copy | 11:29:44 to 11:30:02 | 0, 0 |
| the same two, `run-on-host --need windows --race`, `sh -c` | the Windows laptop, scratch copy | 11:30:13 to 11:30:37 | 0, 0 |
| `golangci-lint run --config .golangci.yml ./...` in `acpext`, six runs, each with a new `GOLANGCI_LINT_CACHE`, under `run-heavy` | this Mac | 12:03:11 to 12:03:39 | 0 six times, "0 issues." each; no finding |
| `make check-acpext-deps` | this Mac | 11:46:45 to 11:46:47 | 0 ("check-acpext-deps: ok") |
| `make race` | this Mac, under `run-heavy` | 12:08:11 to 12:08:57 | 2: the two `acpserver` tests of deviation 2; acpext's `-race` ok |
| `make check-test-isolation` | this Mac, under `run-heavy` | 12:40:48 to 12:41:10 (queued from 12:25:08) | 2: "root module: go test exit 1; 2 failing in 1 package(s)", `TestPromptQuotaData` and `TestRetryNoticeServerMembers` (deviation 2); "acpext module: go test exit 0"; "files written above the temp roots: 0" |
| `make preflight` | this Mac, under `run-heavy` | 12:43:29 to 12:51:45 (queued behind the isolation run) | 0 ("preflight passed"; `golangci-lint` "0 issues." for both modules, `apidiff` "no-op below v1.0.0", `check-records` and `markdownlint` ok) |

No Go file changed after those times, so the cross-host runs, the lint runs, `pre-add-check` and `check-acpext-deps` stand.
The root's `make race` is recorded red for the two tests as the intermediate state, by the owner's ruling; it was not
rerun, and no test was skipped or edited. The Linux WSL run is a non-race run; no WSL race result is used. The six lint
runs waited in the queue behind another job; the lint rule found intermittent on this repository on 2026-11-15
(`confusing-naming`) lives in the root module, which this phase does not change; acpext was clean six of six.
`make check-test-isolation` runs `go test` over both modules and so fails on the same two root tests and on nothing else; it also reports that nothing was written above the temp roots. `make preflight` runs no tests, and passed. `make check-records markdownlint` ran before and after this record was written, exit 0 ("Summary: 0 issues in 0 files"); the identifier scan of the staged diff's added lines is in the last line of this record.

**Acceptance.**

* The tests pass (acpext's, on this Mac, the Linux machine and the Windows laptop, with `-race` on the last two) and each
  plant failed. Met.
* apidiff as step 5, nothing on one side only. Met.
* "The tag served": not met, and not this phase's alone: the owner's, on the owner's own word, as
  `acpext/v0.5.0-rc.1` on the branch's P3 commit (step 7's note), with the final `acpext/v0.5.0` at P8a.

**Docs step (C9).** Searched `acpext/*.go`, `README.md`, `AGENTS.md`, `docs/architecture.md`, `docs/README.md`,
`docs/guides/` for the old kind and cause spellings, `RetryEmpty`, the count of kinds, and acpext's version and surface;
each hit checked against the code.

* `acpext/doc.go`, `acpext/types.go` (`RetryCause`, `ErrorKind`, the kinds' comment, `ErrorData`), `acpext/auth.go`: fixed
  or written.
* `acpext/example_test.go:12`: fixed (deviation 1).
* `docs/architecture.md:91-98` (the acpext paragraph): a sentence added, the contract's eighteen words and
  `AuthenticateMeta`.
* `docs/architecture.md:190`, the root's acpext requirement `v0.3.0`: open for P4 (it reads `v0.5.0-rc.1` there), and for
  P8a (`v0.5.0`).
* `docs/guides/credentials.md:320`, `--json`'s kinds `rate-limited` and `not-permitted`: open for P4 (`llm.Kind`'s
  respelling) and P5 (`--json` prints the contract's steps).
* `0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` joins the scope (C9, 2026-11-15, at the coordinator's review): dated notes at D1, D6, D7, D9's "Where in the PLAN", Confirmation 7 and QB, and a "Revisions of 2026-11-15" section with the owner's rulings and the finding. `0042-MADR` is not edited: the owner kept its rule.
* `docs/architecture.md:95`'s sentence reads "in the tree and in the `v0.5.0` version line (first tagged as `acpext/v0.5.0-rc.1`)", true before the candidate, after it and after the final tag.
* `AGENTS.md`, `README.md`, `docs/guides/developing.md`: nothing names a changed spelling, a kind count or an acpext
  version. This repository keeps no changelog.

**Not verified.** The two other hosts ran scratch copies of the working tree, not a commit. Plants ran with the tree's
`acpext` alone; the root's two tests above are the only root failure of `make race`, and no other root gate was run at
this commit but those in the table. A candidate's `go list` through the proxy waits for the owner's tag.

**What the owner must do for step 7** (on the branch, after the coordinator's commit, on the owner's own word): `git tag -a
acpext/v0.5.0-rc.1 -m acpext/v0.5.0-rc.1 <the branch's P3 commit>`, `git push origin acpext/v0.5.0-rc.1`, then `go list -m
github.com/maccavelli/gobble-cli/acpext@v0.5.0-rc.1` through the proxy. The tag form is that of `acpext/v0.4.0`:
annotated, its message the tag name.

The identifier scan of the staged diff's added lines, for the local account name, the host names, home, temporary and
dev-host scratch paths, run-heavy ticket shapes and mail addresses, found nothing (`grep` exit 1), and found the one line
of a copy with planted values (count 1). It ran after every text of this record was written but this paragraph.

*(2026-11-15, later: the candidate tag.)* On 2026-11-15 the coordinator asked the owner for the word on `acpext/v0.5.0-rc.1`; the
owner said "You may tag and push the tag". The coordinator created the annotated tag `acpext/v0.5.0-rc.1` (its message the tag
name) on `c1e9100`, the branch's P3 commit, and pushed it. `git ls-remote --tags origin 'acpext/v0.5.0-rc.1*'` lists the tag
object `0926ac244a8e3cd52f7835314b5937f16c1138b6` at `refs/tags/acpext/v0.5.0-rc.1` and its commit
`c1e9100a825f46ed98539785cdb969b48d69fc1f` at `refs/tags/acpext/v0.5.0-rc.1^{}`. `go list -m
github.com/maccavelli/gobble-cli/acpext@v0.5.0-rc.1`, run from a scratch directory through the module proxy, printed `… v0.5.0-rc.1`
on the first try, and `@latest` still printed `v0.4.0`. "The tag served", an acceptance item above, is met by the candidate; the
final tag is P8a's.

### P4 record (2026-11-15)

Run by a mainstream model at medium effort, on the owner's approval of this PLAN's P4 (steps 1 to 8, with 4a and 4b) and of
0052-PLAN's P3, as ONE change set: P4's header says "Committed with 0052-PLAN P3, as one commit", and
[0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
has the same commit's other half, "P3 record (2026-11-15)". Built on the branch `p3-acpext-v0.5.0`, after P3's commit.
Times are the session machine's local time. Staged, not committed. **The commit is made on the branch by the coordinator,
under the owner's standing approval, and `main` is fast-forwarded to the branch's tip by the coordinator alone, only
after it has verified the gates.** `main` and `origin/main` stayed at `9e21c8f`; no tag, push or merge was made. *(2026-11-15, the owner's ruling relayed by the coordinator, "Keep it on the branch until those phases land": the coordinator commits each approved phase on the branch `p3-acpext-v0.5.0` and pushes the branch; `main` stays at `9e21c8f` and is fast-forwarded only after 0051-PLAN P6 and 0052-PLAN P4 have landed on the branch and every gate is green, because until then a plan sign-in passes no host id and does not reuse its registered client. Consequence told to and accepted by the owner: CI runs on pushes to `main`, so until `main` moves the phases on the branch have only the local gates and the Linux and Windows runs behind them; `.github/workflows/ci.yml` also runs on `pull_request` and on `v*` tags, which a pull request from the branch would use.)* *(2026-11-15, later: the owner's ruling "Open a draft PR now", relayed by the coordinator. The coordinator's statement that CI runs only on pushes to `main` and tags was wrong: `.github/workflows/ci.yml` also runs on `pull_request`, which the review's finding showed. After P4 is committed and pushed the coordinator opens one draft pull request from `p3-acpext-v0.5.0` to `main`, so every later push to the branch runs CI; `main` does not move and nothing merges until the owner's word, and `main` is still fast-forwarded only after 0051 P6 and 0052 P4 land. The sentence above that the phases on the branch have only the local gates behind them holds only until that pull request is open.)*

**C1, the start state.** Branch `p3-acpext-v0.5.0`, `HEAD` `c1e9100` (P3's acpext changes), pushed, the tree clean; `main` and
`origin/main` at `9e21c8f`. `go.mod` required acpext `v0.3.0`, go-llmprovider-sdk `v1.31.0`, `authcontract` `v0.1.0`
`// indirect`. The candidate tag exists (the note under "P3 record (2026-11-15)"); from a scratch directory, through the
module proxy, `go list -m github.com/maccavelli/gobble-cli/acpext@v0.5.0-rc.1` printed `… v0.5.0-rc.1` on the first try and
`@latest` printed `v0.4.0`, both run again before step 1. `git ls-remote --tags origin 'acpext/v0.5.0-rc.1*'` lists the tag
object `0926ac244a8e3cd52f7835314b5937f16c1138b6` and, peeled, `c1e9100a825f46ed98539785cdb969b48d69fc1f`.
The two `acpserver` tests that read `../acpext/testdata` fail at `c1e9100`, as P3's deviation 2 says: on an unchanged
`git archive` of it, `TestPromptQuotaData/prompt-rate-over-cap.error.json` (`errordata_test.go:203`, the prompt error's data
`"kind":"rate-limited"` against the frame's `rate_limited`) and `TestRetryNoticeServerMembers` (`retry_test.go:213`) fail; they
pass in this tree, and neither was edited to pass. *(2026-11-15, the review: `acpserver/retry_test.go:194`, which `TestRetryNoticeServerMembers` uses, changes its input literal by one word, as step 4 lists; neither's expectation or frame changed. See "Review of 2026-11-15".)*
The lines each phase cites, re-read at this base: moved, with no change in substance, `llm/provider/signin.go`'s `offers` (`:77`
in the PLAN, `func offers` at `:86`, its comment above), the `SignInError` text (`:138`, now `:147`), `Logout`
(`login.go:93-104`, `func Logout` at `:108`), `authProviders` (`agent.go:381-392`, `func` at `:384`), and `prompt.go:308`'s
`WindowMinutes` place (after `d.Limit = …` at `:305`). At the PLAN's own lines, unchanged: `provider.go:273`, `list.go:106-133`,
`resolve.go:52`, `credentials.go:51`, `auth.go:199-223` and `:213`, `:84`, `:307`, `models.go:274` and `:316`, `piimport.go:45`,
`providerscmd.go:114`, `agent/event.go:103`, `agent/agent.go:697`, `view.go:247` and `:249`, `acpclient/update.go:118`,
`acpserver/prompt.go:108` and `:286`, `errordata_test.go:211`, `list_test.go:176-177`, `:188-189` and `:260`,
`workspace_test.go:22-23`, `limit_test.go:159-160` and `:182`, `acpserver/retry_test.go:122-123` and `:194`,
`prompt_test.go:467`, `acpclient/retry_test.go:80-82`, `internal/cli/retry_test.go:22-24` and `:33`. No site changed in substance.

**What changed (staged).**

1. **Step 1.** `go get github.com/maccavelli/gobble-cli/acpext@v0.5.0-rc.1`, `go mod tidy`. The whole `go.mod` diff is two lines:
   `github.com/maccavelli/gobble-cli/acpext v0.3.0` becomes `v0.5.0-rc.1`, and `github.com/maccavelli/go-llmprovider-sdk/authcontract
   v0.1.0` moves from the indirect block to the direct one, as step 1 says (the first import is `llm/provider/list.go`'s). `go.sum`
   changes only acpext's two lines. No other requirement moved; no `replace` and no `go.work`. 0049-PLAN P5b-2's wire half:
   `acpserver/prompt.go`'s `errorData` sets `WindowMinutes`; `errordata_test.go`'s `weeklyErr`, its case in `TestErrorData` and
   its frame in `TestPromptQuotaData` (`prompt-weekly.error.json`, in acpext's testdata since P3), re-applied from `1406bd6`
   unchanged.
2. **Step 2.** `llm/provider/policy.go` (new): `acceptedBorrowed`, `honouredConsents` (0052-PLAN P3 fills it: both here, in one
   commit), `Offered`, `kept`, `Remote`, and `HasLogin`, `HasKeyLogin`, `HasSignIn`, `Revocable`, `DefaultMethod`,
   `ImportsPiLogin`, `Accepts`, all derived from `Offered`. `offers`, `HasPiLogin`'s place and the `openai` checks are gone;
   `methodIDs` stays the word table, `Methods` reads `Offered`, and `skips` skips what `Offered` does not give.
3. **Step 3.** `internal/auth/resolve.go`: `Resolver.Accepts func(id string, e Entry) bool`, in place of `HasLogin`; `AcceptsLogin`
   (deviation 2); `credentials.go` passes it. `Logout` revokes only an accepted, revocable session and deletes any, and keeps
   P2a's `UseIdentity` before the revocation.
4. **Step 4.** `llm/kind.go` in the contract's spellings; `provider.ErrorKind` (in `list.go`) in place of `FailureKind` and
   `failureKinds`; every caller (`provider.go`, `signin.go`, `auth.go`, `models.go`) uses it. The three tests that called
   `FailureKind` assert `ErrorKind` with the same word (`list_test.go:176-177` `auth`, `:188-189` `timeout`, `workspace_test.go:22-23`
   `workspace`); `TestKindAgreesWithFailureKind` and `TestFailureKind` are removed with it, `TestErrorKindIsTheContracts` takes
   their place. **4a:** `loginWith` reads `DefaultMethod`, with a usage error naming `--api-key -` for none, and `--device`
   picks the offered device method; no per-id case remains. **4b:** `agent.RetryEmptyReply` is `"empty_answer"`; `retryLine`
   compares `string(acpext.RetryOffline)` and `string(acpext.RetryEmptyReply)`; `acpclient/update.go`'s doc names the new
   words. The tests that held an old word change with their values, each a one-word diff: `acpserver/retry_test.go:122-123`
   (`empty_answer`), `:194` (`rate_limited`), `acpserver/prompt_test.go:467`, `acpclient/retry_test.go:80-82`,
   `internal/cli/retry_test.go:22-24`, `llm/provider/limit_test.go:159-160`, and `llm/kind_test.go`'s four values.
5. **Steps 5 and 6.** `authProviders` sets `Login` from `provider.HasSignIn(id)` and the command from the default method; `acp.go`'s
   `--login` and `piimport.go` read `HasSignIn` and `ImportsPiLogin`; `signInWords` renders `Offered`'s labels.

Test values that changed with the offer, each named: `llm/provider/signin_test.go`'s `TestSignInMethods` (openai `api-key`,
`chatgpt-plan`) and `TestSignInMenu` (openai's forced `api-key` in place of the bare menu); `login_test.go`'s `TestLogout` and
`TestLogoutRevokesAsGobble` fixtures carry `Kind` `grok`, as a real sign-in's session does (a test issuer's URL is no issuer the SDK
infers a kind from), and `TestLogoutRevokesAsGobble` runs for `openai`'s plan session too; `internal/cli/auth_test.go`'s
`TestAuthLogin` loses its "not in gobble yet" line, which `TestLoginWithReadsTheDefaultMethod` replaces; `acp_test.go`'s
`TestAgentAuthOptions` and `choice_test.go`, `providerscmd_test.go` take openai's `Login` and the labels as `signInWords` gives them.

**Tests (C2): each seen failing on a scratch copy of the tree** (`<TMP>`, no `.git`; `HOME`, `USERPROFILE`, `XDG_*` and
`GOBBLE_HOME` in the scratch; `GOPROXY=off`; one `go test -count=1 -run '^(Test…)$' <package>` each, run after the last Go
change; unbroken, each passes). Each plant exits 1:

| Test | Planted break | Failure seen (exit 1 each; `<TMP>` is the scratch copy) |
| :--- | :--- | :--- |
| `TestOfferedMethods` (P4) | the borrowed-client rule dropped (`kept` no longer refuses a vendor-borrowed client) | `policy_test.go:41: Offered(openai) = ["api_key" "browser_oauth" "device_code" "chatgpt_plan"], want ["api_key" "chatgpt_plan"]` |
| `TestOfferedMethods` (P4) | `acceptedBorrowed` emptied | `policy_test.go:41: Offered(grok) = ["api_key"], want ["api_key" "browser_oauth" "device_code"]` |
| `TestOfferedMethods` (P4) | `honouredConsents` emptied | `policy_test.go:41: Offered(openai) = ["api_key"], want ["api_key" "chatgpt_plan"]` |
| `TestPredicatesFromOffered` (P4) | `HasKeyLogin` reading `oauth_session` | `policy_test.go:71: openai: HasLogin true, HasKeyLogin true, Revocable true, ImportsPiLogin false; Offered says true, false, true, false` |
| `TestRemoteMethods` (P4) | `Remote` reading `Completion` alone, so a loopback-only browser method is accepted | `policy_test.go:132: Remote(a loopback-only browser sign-in) = true, want false` |
| `TestRemoteMethods` (P4) | `Remote` refusing every browser completion | `policy_test.go:115: grok: Remote accepts ["device_code"], want ["browser_oauth" "device_code"]` |
| `TestEveryCommandAndWordIsAccepted` (P4) | openai's word list given `device` | `defaultmethod_test.go:119: openai: 3 method words for 2 offered methods` |
| `TestEveryCommandAndWordIsAccepted` (P4) | kilo's command given `--method browser` | `defaultmethod_test.go:78: kilo: "gobble auth login kilo --method browser" names a method gobble does not offer: kilo signs in with --method api-key or device, not "browser"` |
| `TestResolverAcceptsOnlyOfferedKinds` (P4) | `Accepts` ignoring the kind | `resolve_test.go:172: Credential(openai) = ok true, err <nil>; want usable false` |
| `TestLogoutRevokesOnlyAccepted` (P4) | revocation by `HasLogin` alone | `login_test.go:523: logout: oauth: revoke failed: 404 Not Found: 404 page not found, <nil>, stored Login(nil); want it deleted` |
| `TestErrorKindIsTheContracts` (P4) | `llm/kind.go` keeping `rate-limited` | `errorkind_test.go:53: ErrorKind(llm: rate limited) = "rate-limited", not a word of the contract` |
| `TestKindsAndLimitsMatchAcpext` (P4) | acpext pinned back to `v0.4.0` in the scratch copy | `acpserver/prompt.go:289:5: d.Consent undefined (type acpext.ErrorData has no field or method Consent)` |
| `TestKindsAndLimitsMatchAcpext` (P4) | `KindDenied` deleted from the scratch copy's acpext (a `replace` to its `acpext/`, in the scratch `go.mod` only) | `acpserver/errordata_test.go:245:10: undefined: acpext.KindDenied` |
| `TestLoginWithReadsTheDefaultMethod` (P4) | the `id == "openai"` case restored | `defaultmethod_test.go:39: gobble auth login openai: exit 2, stderr "Error: openai's ChatGPT sign-in is not in gobble yet (0031-PLAN P4b); use --api-key -\n", signed in by []; want "chatgpt-plan"` |
| `TestRetryCausesAreAcpexts` (P4) | `agent.RetryEmptyReply` left `"empty_reply"` | `retrycause_test.go:23: agent causes "offline" and "empty_reply", acpext's "offline" and "empty_answer"` |
| `TestRetryLines` (P4) | `retryLine` left comparing the literal `"empty_reply"` | `retry_test.go:40: view: stdout "", stderr "waiting for the network\nretrying: empty answer (attempt 1 of 2)\n"; want nothing and "waiting for the network\nretrying: empty reply (attempt 1 of 2)\n"` |
| `TestLogoutRevokesAsGobble` (P4) | the `UseIdentity` call moved inside a grok-only branch | `login_test.go:470: the named logout: GET /.well-known/openid-configuration sent User-Agent "go-llmprovider-sdk/v1.31.0 (darwin; arm64) go-llmprovider-sdk/v1.31.0", want gobble/1.2.3-test ( ... go-llmprovider-sdk/` |
| `TestErrorData` (the weekly case) and `TestPromptQuotaData` (`prompt-weekly.error.json`) (P4) | the `WindowMinutes` lines removed | `errordata_test.go:193: errorData` |

On the tree before the phase (HEAD's `acpext` and sources, with the new and changed test files added): `llm` does not compile,
`undefined: ConsentError`; `llm/provider`, `undefined: ErrorKind`; `internal/auth`, `r.Accepts undefined (type *Resolver has no
field or method Accepts)` and `undefined: AcceptsLogin`; `internal/cli`, `undefined: provider.DefaultMethod`,
`undefined: provider.Offered`, `undefined: provider.ConsentText`; `acpserver`, `undefined: ConsentState` and `unknown field
PlanConsent in struct literal of type Options`. `acpclient` and `agent` compile and pass there too: their tests change one
literal each, which each decodes or compares with itself.

**Gates (C6),** each exit status taken from the file its output was redirected to before anything read it. `make race`,
`make check-test-isolation`, `make preflight` and the lint runs went through `run-heavy`; the other two hosts through
`run-on-host --race`, `sh -c`, which kept the Linux run off the WSL distros.

| Gate | Where | Start to end | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check FILES=` the 51 changed Go files | this Mac | 14:02:17 to 14:03:06 | 0 ("51 file(s) clean (gofmt, golint, govulncheck)") |
| `golangci-lint run --config .golangci.yml` over the seven changed packages, six runs, each with a new `GOLANGCI_LINT_CACHE`, under `run-heavy` | this Mac | 14:03:20 to 14:04:33 | 0, 0, 0, 0, 0, 0; "0 issues." each |
| `make check-acpext-deps` | this Mac | 14:04:55 to 14:04:55 | 0 |
| `make race`, then `GOFLAGS=-count=1 make race`, under `run-heavy` | this Mac | 14:04:55 to 14:05:38, then 14:06:50 to 14:07:31 | 0, 0 (the root's packages and acpext; no test cache read in the second) |
| `make check-test-isolation`, under `run-heavy` | this Mac | 14:05:38 to 14:05:57 | 0 ("root module: go test exit 0; 0 failing", "acpext module: go test exit 0", "files written above the temp roots: 0") |
| `go test -count=1 ./...`, `CGO_ENABLED=1 go test -count=1 -race ./...`, `go -C acpext test -count=1 ./...`, one `run-on-host --need linux --race` | the Linux machine, scratch copy | 14:04:55 to 14:09:14 | 0 (`GOTEST-OK`, `RACE-OK`, `ACPEXT-OK`) |
| the same three, `run-on-host --need windows --race`, `sh -c` | the Windows laptop, scratch copy, Go 1.27.4 | 14:04:55 to 14:07:13 | 0 (`GOTEST-OK`, `RACE-OK`, `ACPEXT-OK`) |
| `make preflight`, under `run-heavy` | this Mac | 14:15:36 to 14:23:11 | 0 ("preflight passed") |

The six lint runs: run 1 14:03:20 to 14:03:33 exit 0, run 2 14:03:33 to 14:03:44 exit 0, run 3 14:03:45 to 14:03:57 exit 0, run 4 14:03:57 to 14:04:09 exit 0, run 5 14:04:10 to 14:04:21 exit 0, run 6 14:04:21 to 14:04:33 exit 0; **six of six clean, no finding of any
kind**, so there was nothing to fix. Besides the lint runs, no identifier this change adds differs from another in its package only by
capitalization: a scan of every top-level name of the seven packages, test files included, finds only five pairs, each
older than this change (`Selection`/`selection`, `RequestPermissionsTool`/`requestPermissionsTool`, `PromptError`/`promptError`,
`Start`/`start`, `ToolContent`/`toolContent`); the one pair this change touched, `Resolver`'s field `Accepts` and a method, was
renamed apart (`usesLogin`).

The changed Go files were last modified at the times below, before every gate in the table above ran (the gates started at
14:02:17 and later; the last Go edit was before 14:02:17):

```text
13:09:12  acpclient/update.go
13:09:12  internal/cli/view.go
13:09:12  llm/kind.go
13:09:12  llm/kind_test.go
13:09:30  agent/event.go
13:09:54  acpclient/retry_test.go
13:09:54  acpserver/prompt_test.go
13:09:54  acpserver/retry_test.go
13:09:54  internal/cli/retry_test.go
13:10:22  internal/cli/models.go
13:10:22  llm/provider/list.go
13:10:49  llm/provider/list_test.go
13:10:49  llm/provider/workspace_test.go
13:11:02  llm/provider/limit_test.go
13:18:45  llm/background.go
13:18:45  llm/consent.go
13:19:46  llm/provider/login.go
13:19:46  llm/provider/signin.go
13:20:28  llm/provider/consent.go
13:20:43  llm/provider/policy.go
13:22:22  internal/cli/acp.go
13:22:22  internal/cli/piimport.go
13:22:28  internal/cli/providerscmd.go
13:24:32  internal/cli/acp_test.go
13:24:32  internal/cli/auth_test.go
13:24:32  internal/cli/choice_test.go
13:24:32  internal/cli/providerscmd_test.go
13:24:32  llm/provider/signin_test.go
13:26:12  llm/provider/policy_test.go
13:26:38  llm/provider/errorkind_test.go
13:27:13  llm/provider/login_test.go
13:27:47  internal/auth/resolve_test.go
13:29:37  acpserver/retrycause_test.go
13:30:13  acpserver/errordata_test.go
13:31:02  acpserver/prompt.go
13:31:09  acpserver/agent.go
13:31:09  acpserver/title.go
13:33:21  acpserver/autocompact.go
13:33:21  internal/cli/agent.go
13:33:34  internal/cli/credentials.go
13:35:54  acpserver/autocompact_test.go
13:35:54  acpserver/title_test.go
13:38:41  llm/provider/mark_test.go
13:39:34  internal/cli/consent_test.go
13:48:29  llm/consent_test.go
13:48:33  internal/cli/auth.go
13:48:33  internal/cli/consentwording_test.go
13:49:47  llm/provider/provider.go
13:51:56  internal/cli/defaultmethod_test.go
14:00:35  acpserver/plan_test.go
14:01:50  internal/auth/resolve.go
```

The Linux and Windows copies hold the working tree, not a commit.

**Deviations.**

1. **`provider.Login(id)` cannot be declared.** Steps 2 and 5 name a predicate `Login`, "acpext's `Login` … from `provider.Login(id)`";
   `Login` is the stored login's type in `llm/provider` (`login.go:17`). The predicate is `HasSignIn(id)`: an offered method that is
   not an API key. Equivalent.
2. **`Accepts` takes a `*provider.Login`, and the bridge is `auth.AcceptsLogin`.** Step 3 has `credentials.go:51` pass
   `provider.Accepts`, with the resolver's `Accepts func(id string, e Entry) bool`. `internal/auth` imports `llm/provider` (`logins.go`), so
   `provider.Accepts` cannot take an `auth.Entry`. `provider.Accepts(id, *Login)` calls the SDK's `SessionKind()` on the session the
   `Login` builds, as the MADR says; `auth.AcceptsLogin(id, Entry)` (in `resolve.go`, which the PLAN names) builds a `Login` from the entry's `Kind`,
   `Issuer` and `Resource`, the three members the SDK's rules read, and `credentials.go` passes it.
3. **The pairing of an offered method with the session it signs in to is gobble's table** (`sessionKinds`, `policy.go`, three rows,
   keyed by provider and the method's client identity). The PLAN says "a session kind an offered method signs in to" and the
   SDK has no accessor for what a method will leave. Within the step; named because it is a table the PLAN does not show.
4. **The SDK's fact under `TestRemoteMethods` has changed. *(2026-11-15, later: ruled by the owner, "Amend the records to the SDK's facts", relayed by the coordinator; see "Review of 2026-11-15" below.)*** *Evidence.* The PLAN's row for `TestRemoteMethods`
   expects "grok's `browser_oauth` refused", and its plant "`Remote` reading `Completion` alone, so grok's `browser_oauth` is
   accepted". At go-llmprovider-sdk `v1.31.0` (module cache, `llmprovider/providers/grok/descriptor.go`) Grok's `browser_oauth`
   is `Completion` `browser_paste`, `Paste` `url_or_code`, not `browser_loopback` with no `Paste` as MADR D2's note says; the SDK's
   `0074-MADR-one-sign-in-handle-and-method-facts.md` has it as its table row C62, "now `browser_paste`, Phase 8 step 1", with C56 and
   C57 ("`NoLoopback` is refused for Grok") also removed by that step. `Remote`, which reads those two facts and no provider id,
   therefore accepts it, and the PLAN's row cannot pass as written whatever `Remote` does short of a provider-id rule. *Done,
   pending the owner's word:* `Remote` is the MADR's rule unchanged; `TestRemoteMethods` expects grok `browser_oauth` and `device_code`,
   kilo `device_code`, openai `chatgpt_plan`, and holds the rule's refusal on methods of the contract's own shape, a
   loopback-only browser method and a paste-marked one with no paste, which no built-in method has; its two plants are the
   PLAN's second (every browser completion refused) and the PLAN's first made to bite on that loopback method. MADR 0051 D2 carries a
   dated note. *Resolutions:* (a) recommended: accept this, amending D2's sentence "it refuses one" (cost: one note, already
   written; P6's relay may then drive Grok's browser sign-in for a remote client, which the SDK now supports); (b) refuse Grok's
   browser sign-in for a remote client by a provider-id rule (a workaround: the per-id check D2 exists to avoid, and a refusal the
   SDK's facts no longer state). *Doing nothing:* the PLAN's test row and the MADR's sentence state a fact the pinned SDK does not have.
5. **`auth status`'s `oauth_unused` reason** read `!provider.HasLogin(id)` (`auth.go`'s `statusRow`); `HasLogin("openai")` is now true,
   so an OpenAI entry of another application's login would be `usable: false` with no reason. It reads `!ok`, the resolver's own verdict.
   Not named by the PLAN (0052-PLAN P4 step 2 owns the rest of the status row).
6. **Tests the PLAN did not name:** `TestUnhonouredConsentKeepsMethodOut` (`policy_test.go`: the rules of `kept`, one plant), and
   `TestLoginWithReadsTheDefaultMethod` also asserts the `--device` refusal's generic text.
7. **Observation, no change: the provider build and a foreign OpenAI session.** `Options.login` (`provider.go`) loads whatever
   `auth.json` holds for an id whose `HasLogin` is true, as before; the resolver (`Accepts`) never hands a foreign session to it, so the
   CLI builds none, and 0052-PLAN P4's `TestForeignOpenAISessionRefused` owns the provider build's own refusal. Until then the
   window is a caller that builds a provider from a `Logins` store without the resolver.

**Docs step (C9),** with the docs of 0052-PLAN P3 in the same commit. Searched `README.md`, `AGENTS.md`, `docs/architecture.md`,
`docs/README.md`, `docs/guides/` for the old kind and cause spellings, `FailureKind`, "sign in", `auth login`, `acpext`, the
requirement table, rule 4, the `providers` table and `--method`'s words; each hit checked against the code and the tests. This
repository keeps no changelog.

Files that join the scope by C9, by this dated deviation (2026-11-15), without asking: `docs/guides/choosing-a-model.md` (the `providers`
table's sign-in words, which `signInWords` changes); `0051-MADR` (a dated note under D2, deviation 4); and
`0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md`, whose "P5b-2 record (2026-11-15)" step 1 of this phase
names. `README.md`, `docs/architecture.md`, `docs/guides/credentials.md` and `docs/README.md` are in the scope already.

The rows, one a finding; lines are as of this change (the brief's earlier numbers moved with the edits above them):

* `README.md:170` ("Only `llm/provider` imports it", the brief's `README.md:164`) — fixed: it says "but for its data-only `authcontract` module", which `acpserver` and `internal/cli` may import.
* `docs/architecture.md:167` (rule 4, the brief's `docs/architecture.md:164`) — fixed, the same sentence.
* `docs/architecture.md:194` (the root's acpext requirement, the brief's `:190`) — fixed: `v0.5.0-rc.1`.
* `docs/architecture.md:194` — open: P8a makes it `v0.5.0`.
* `docs/guides/credentials.md:366-369` (the `--json` error kinds, the brief's `:320`) — fixed: respelled, with `denied` and `expired`, and the plan sign-in's stop at its question under `--json`.
* `docs/guides/credentials.md:338-376` (`--json`'s events, and ACP `authenticate`) — open: P5 makes `--json` print the contract's steps and refuse a first plan sign-in with exit 2, and P6 makes `authenticate` sign in by elicitation.
* `README.md:109-120` (the credentials bullet) — fixed: ChatGPT plan offered, `--method chatgpt-plan`, `gobble auth consent`, the question and what it gates.
* `docs/architecture.md:24`, `:109`, `:110` and `:127` (the command list, the `llm`, `llm/provider` and `acpserver` rows) — fixed: `auth consent`, the contract's words, `WithBackground` and `ConsentError`, the policy, `ErrorKind`, `ConsentText`, the consent functions, the gate and `ErrorData.Consent`.
* `docs/guides/credentials.md:118` (`oauth_unused`) — fixed: a session of a kind gobble does not sign in to is not used.
* `docs/guides/credentials.md:152-153` (`--method`'s words) — fixed: `chatgpt-plan` added.
* `docs/guides/credentials.md:189-217` (Signing in: the third provider, its consent wording and where it is asked) — fixed; `:245-300` (logout of a plan session, "Allowing background requests under a ChatGPT plan", `gobble auth consent`) — fixed, new; `:306` (`/login`'s methods) and `:347-348` (`gobble acp --login`) — fixed.
* `docs/guides/credentials.md:189-300` (the ChatGPT section) — open: 0052-PLAN P4 step 3 adds the files it keeps, the host id and the issued client, and the phone through an ACP client; P6 the elicitation.
* `docs/guides/credentials.md:86-132` (`auth status --json`) — open: 0052-PLAN P4 step 2 adds `sessionKind`, `consents` and `consentTerms`.
* `docs/guides/choosing-a-model.md:137`, `:139` and `:143` (the `providers` table's sign-in words) — fixed.
* `AGENTS.md`, `docs/guides/developing.md`, `docs/guides/models-json.md`, `docs/guides/` otherwise — nothing names a changed spelling, a kind count, the offered methods or acpext's requirement; no change. This repository keeps no changelog.

`docs: searched README.md, AGENTS.md, docs/architecture.md, docs/README.md, docs/guides/*.md` (for `rate-limited`, `not-permitted`, `empty_reply`, `FailureKind`, `sign in`, `sign-in`, `auth login`, `ChatGPT`, `chatgpt`, `acpext`, `v0.3.0`, `Only llm/provider`, `--method`, `device sign-in`).


**Acceptance.**

* The tests pass and each plant failed: the table above, and every gate. Met.
* `grep -rn 'FailureKind\|id == "openai"\|HasPiLogin' --include=*.go .` finds nothing (exit 1). Met.
* `git grep -n '"empty_reply"\|"rate-limited"\|"not-permitted"\|"context-overflow"' -- '*.go' 'acpext/testdata'` finds nothing (exit 1);
  `retry.empty_reply`, the personality key, is not matched by it. Met.
* `TestAcpextAuthProvidersAreTheRegistry` passes unchanged. Met.
* The commit is the coordinator's, after 0052-PLAN P3's acceptance.

**Not verified.** The two other hosts ran scratch copies of the working tree, not a commit. Nothing was run against a live
service: C5, and the live sign-in is P8's. A real `auth.json` was never read: every test uses a scratch `GOBBLE_HOME`. The
SDK's behaviour on a ChatGPT plan sign-in past the consent question (the browser, the paste, the token exchange) is covered
by the SDK's own tests and by 0052-PLAN P4's `TestSignInChatGPTPlan`, not here: this change's tests end the sign-in at the question.

The identifier scan of the staged diff's added lines, for the local account name, the host names, home, temporary and
dev-host scratch paths, run-heavy ticket shapes and mail addresses, found nothing (`grep` exit 1), and found the one line of a copy with planted values (count 1). It ran after every text of this
record was written but this paragraph; `make check-records markdownlint` ran after it too, exit 0.

#### Review of 2026-11-15

An independent review of this change set found gaps against the plans and the MADRs; the coordinator relayed eleven findings and three questions.
This entry is this phase's: findings 2, 4, 6 (the `defaultmethod_test.go` half and the `retrycause_test.go` half), 8 and 11, deviation 6's
evidence, the question on the provider build (Q-b), and the rulings; the "Review of 2026-11-15" under 0052-PLAN's "P3 record (2026-11-15)"
has the rest. It annotates the record above and rewrites none of it, but for one sentence the review corrects: the record's C1 paragraph
says of the two `acpserver` tests that fail at `c1e9100` that neither "was edited to pass"; `acpserver/retry_test.go:194` changes
`TestRetryNoticeServerMembers`'s input literal, `Kind: "rate-limited"` to `"rate_limited"`, one word, as step 4 of "What changed" lists. Neither
test's expectation or frame was changed; the sentence is not repeated.

**The owner's rulings of 2026-11-15, relayed by the coordinator,** recorded in the same change set as the review's fixes:

* **Deviation 4** (Grok's `browser_oauth` and `Remote`): "Amend the records to the SDK's facts". D2's rule stays as decided; `Remote` and
  `TestRemoteMethods` stay as built; no code change. The note under 0051-MADR D2 says the owner ruled, and each passage of the MADR and the PLAN
  that still said Grok's browser sign-in is loopback-only with no paste is annotated with the ruling and the facts. The passages, verified
  line by line in this tree: 0051-MADR D2's note (`:331-337`); D3's note (`:365`) and its bullet on what cannot complete that way (`:384`);
  D4 step 1's refused-method bullet (`:426`); step 3's list of driven methods (`:445`), its rule (1) (`:451-454`) and its table row (`:470`);
  Confirmation 3's `_meta.gobble.remote` line (`:778`); the Not-verified question on D12 (`:919`); in this PLAN, P4's `TestRemoteMethods` row
  (`:425`), P6's rows `TestAuthenticateOnlyRemoteMethods` (`:580`), `TestAuthenticateGrokBrowserHasNoPaste` (`:598`) and
  `TestAuthenticateRemoteRefusesLoopbackOnly` (`:610`), and P8's L3 (`:715`). The coordinator's list was complete but for D3's two passages
  (`:365`, `:384`), the Not-verified question (`:919`), P4's own row (`:425`) and L3 (`:715`); the passages that say only "a method that only the loopback
  completes" (`:376`, `:382`, `:388`, `:948`) are generic and hold; 0052's pair has none (0052-MADR's one mention of Grok's browser, that it and
  its device sign-in yield one kind of session, holds). The three P6 rows cannot pass as written at the pinned facts, and what replaces them is an open question for the owner, recorded under P6 ("Open question (2026-11-15)"); no replacement is decided here.
* **The other deviations:** "Accept the eight; I check 6 and 9". Counting this record's and the "P3 record (2026-11-15)"'s numbers, the
  coordinator's list is 1, 2, 3, 5, 8, 10, 11, 12 (eight deviations) and 7 (an observation, not a deviation), nine numbers in all; accepted by the
  owner on the independent review, 8 subject to review finding 5, closed below. Deviation 4 is ruled above. Deviations 6 and 9 are not accepted
  yet: 6 is answered below (each test the plans did not name), and 9 is withdrawn below, its stated reason being false, and built as the PLAN
  says (see the "Review of 2026-11-15" of the "P3 record (2026-11-15)").
* **When `main` moves:** "Keep it on the branch until those phases land". The coordinator commits each approved phase on the branch
  `p3-acpext-v0.5.0` and pushes the branch; `main` stays at `9e21c8f` until 0051 P6 and 0052 P4 have landed on the branch and every gate is
  green. This replaces what "P4 record (2026-11-15)" and the "P3 record (2026-11-15)" say about the coordinator fast-forwarding `main` after P4.
  The consequence the owner was told and accepted: CI runs on pushes to `main`, so until then the phases on the branch have only the local
  gates and the Linux and Windows runs behind them. Evidence for the record: `.github/workflows/ci.yml` runs on `push` to `main` and `v*` tags
  and also on `pull_request`, so a pull request from the branch would run it. *(2026-11-15, later: the owner's ruling "Open a draft PR now", relayed by the coordinator. The coordinator's statement that CI runs only on pushes to `main` and tags was wrong: `.github/workflows/ci.yml` also runs on `pull_request`, which the review's finding showed. After P4 is committed and pushed the coordinator opens one draft pull request from `p3-acpext-v0.5.0` to `main`, so every later push to the branch runs CI; `main` does not move and nothing merges until the owner's word, and `main` is still fast-forwarded only after 0051 P6 and 0052 P4 land. The sentence above that the phases on the branch have only the local gates behind them holds only until that pull request is open.)*
  *Does any later phase, gate, script or doc assume it starts from `main`?* Found, each annotated or to be read with the ruling:
  (1) C1 ("re-reads `git status`, `HEAD`, `origin/main`"): annotated; the start state is read against the branch's tip and
  `origin/p3-acpext-v0.5.0`. (2) C8 and P4's header ("the later phases (P5 onward) continue on `main`"): annotated; P5, P6 and 0052-PLAN P4 continue
  on the branch. (3) The P3 record's quotation of the owner's earlier ruling ("the remaining phases continue on main") and 0051-MADR's
  "Revisions of 2026-11-15" ("`main` holds P3 only together with P4"): annotated. (4) P8a ("on the commit `main` has then",
  `git diff acpext/v0.5.0-rc.1 HEAD -- acpext/`): holds, since `main` holds P4 by P8a. (5) P7a, `scripts/check_libs.py`, `make check-libs`, the
  archtest and `make check-acpext-deps` read the tree and the network, never `main`: no annotation. (6) The verify-build-metadata gate names HEAD,
  not `main`: no annotation. (7) CI: no row of any acceptance names it; its absence on the branch is the consequence above. No gate or script changes.

**Finding 2, `ErrorKind` called the SDK first for every error: true.** 0051-MADR D6: "`provider.ErrorKind(err error) string` returns the contract's
word: `signin.ErrorKindOf` for an error of the SDK's sign-ins, else `llm.Kind`'s." The code (`llm/provider/list.go`) called `signin.ErrorKindOf` first
for every error and fell back on `llm.Kind` only for "other" and "". Probe: `mapError(&llmprovider.APIError{Status: 401, Kind: ErrQuotaExhausted,
Code: "MonthlyLimitError"})` gave `auth`, `llm.Kind` `quota` (0049-MADR D8's case). *Built as D6 says,* with one reading it fixes: an error of the SDK's
sign-ins is a `*SignInError`, which `SignIn` returns for every failure of the SDK's, or an error holding the SDK's denied or expired marker
(`auth.ErrSignInDenied`, `ErrSignInExpired`, the two the SDK's own doc says `ErrorKindOf` tells apart); any other error is `llm.Kind`'s. The step 4
sentence "`signin.ErrorKindOf` first, then `llm.Kind`" is read as D6 reads it. The criterion is the code's own types and the SDK's two markers,
so D6 could be built as written and no option was picked between readings; if the owner reads "an error of the SDK's sign-ins" more widely, the
criterion is `isSignInError`, one function. `SignInError`'s text takes `ErrorKind(e)`, so a denied sign-in still reads `(denied)`. New
`TestErrorKindAgreesWithLlmKind` (the agreement test the removed `TestKindAgreesWithFailureKind` was): for every class of `llm`, the 401 quota, an invalid
request and a plain error, `ErrorKind` equals `llm.Kind`; the 401 quota is `quota`; a sign-in error holding the denied marker is `denied`, one over an
invalid request `invalid_request` (which `llm.Kind` has no word for), one over a refusal `auth`.

**Finding 4, nothing tied `sessionKinds` to what the SDK stores: true.** `TestSignInGrokBrowser` and `TestSignInDevice` now assert
`Accepts("grok", <the login the SDK stored>)` after each real sign-in (`signin_test.go`); no test stores a plan session through the SDK before 0052-PLAN P4's
`TestSignInChatGPTPlan`, so the plan's row is not tied here, and `sessionKinds`'s `chatgpt_plan` word is held by the tests that hand-write it and by that
phase's test when it lands.

**Finding 6, vacuous assertions: true, both.** `defaultmethod_test.go`'s loop re-checked membership in the list it iterated; it now compares each
provider's `Methods` with the `--method` words of its offered methods from a table in the test, in the SDK's order (`api_key` `api-key`, `browser_oauth` `browser`,
`device_code` `device`, `chatgpt_plan` `chatgpt-plan`). `retrycause_test.go` fed the literal `"rate_limited"`; it now feeds `provider.ErrorKind` of a rate-limited
error, the word the adapter's `providerRetry` computes, so a respelling in `llm/kind.go` fails it.

**Finding 8, Logout read revocability per provider: true.** 0051-MADR D2: "The logout revokes when the stored session's method is `Revocable`, and
only for a session gobble accepts (D2's last bullet), never another app's (F54)." `Logout` now reads `storedMethod(id, l)`, the offered session method
whose kind the stored session's kind names, and revokes when that method is `Revocable`. Where several offered methods share a kind (Grok's browser
and device) the first is read; the SDK keeps no method id on a session (0052-MADR, "Where the report … differ"). A seam, `offeredFor`, lets
`TestLogoutReadsTheStoredMethod` offer a plan method that is not revocable beside a revocable method of another kind: the plan session is deleted and never revoked.

**Finding 11, `docs/architecture.md:110`: true.** It said `Remote` is "read from" `Offered`; it now says `Remote` reads the facts of one method
`Offered` gives, and the `ErrorKind` sentence says which words come from where.

**Q-b, which phase owns the refusal of a foreign session at provider build (`Options.login`, `provider.go:99`)?** 0052-MADR D1's table: "`HasLogin` gaining
openai, `offers`' guards, the `provider.go:107-112` refusal, … | **reworked** onto 0051-MADR D2's policy and the SDK's session kind". 0052-PLAN P4 step 1:
"Re-apply, reworked, what P1 step 3 took out: … `TestOpenAILoginIsAPlan` as `TestForeignOpenAISessionRefused`", and its row: "`TestForeignOpenAISessionRefused` | status,
`auth check`, `authenticate`, provider choice and the provider build all refuse it; the logout deletes and does not revoke | the provider build refusing while status
accepts". So 0052-PLAN P4 owns it, and no phase before it: until then `Options.login` loads what `auth.json` holds for an id whose `HasLogin` is true. On the resolver
reading a snapshot while `New` loads fresh: both are written nowhere in the plans; P4's test is the first to hold the two to the same verdict, and the plans'
wording ("the resolver … logout then agree (F54)", 0051-MADR D2) assumes it. Not a deviation of this change; deviation 7 stands as an observation.

**Deviation 6, the tests the plans did not name** (each with what it asserts and the plant that fails it, in the table below):
`TestUnhonouredConsentKeepsMethodOut` (the rules of `kept`: an unhonoured required consent, a token paste, a vendor import, a host-only method and an unaccepted borrowed
client keep a method out; plant `kept-ignores-consent`); `TestErrorKindAgreesWithLlmKind` (finding 2); `TestLogoutReadsTheStoredMethod` (finding 8);
the added assertions in `TestSignInGrokBrowser` and `TestSignInDevice` (finding 4); the generic-text assertion in `TestLoginWithReadsTheDefaultMethod` (the `--device`
refusal names only the id given and no ChatGPT sentence; plant `defaultmethod-openai-case`); and, 0052 P3's, `TestPlanConsentIsReadFromAuthJSON`,
`TestTitleNotMadeSendsNothing`, `TestTitleGateReadsOnlyWhenATitleIsAsked`, `TestAuthConsentNeverFailsAfterTheWrite` (its record). Each exists because a plan row or a MADR
sentence has no test of its own or a stronger one was needed; none loosens or replaces a named test.

**The plants of this review, each on a scratch copy of the tree** (`<TMP>`, `GOPROXY=off`, scratch `HOME`; each exits 1; the plants of the
earlier record were run again on this tree, all 53 failing, the list in this table being the new ones):

| Finding | Test | Planted break | Failure seen (exit 1) |
| :--- | :--- | :--- | :--- |
| 1 | `TestPlanConsentIsReadFromAuthJSON` | `PlanConsent` reading `Granted: granted || true` (the reviewer's plant) | `consentstore_test.go:89: the run's method: {Consent:chatgpt_plan.background_use Plan:true Granted:true}, <nil>; want plan true, granted false, error false, consent "chatgpt_plan.background_use"` |
| 1 | `TestPlanConsentIsReadFromAuthJSON` | `internal/cli`'s `planConsent` giving `Granted: true` | `consentstore_test.go:89: the run's method: {Consent:chatgpt_plan.background_use Plan:true Granted:true}, <nil>; want plan true, granted false, error false, consent "chatgpt_plan.background_use"` |
| 1 | `TestPlanConsentIsReadFromAuthJSON` | `agentOptions` not setting `PlanConsent` | `consentstore_test.go:84: the agent is given no PlanConsent` |
| 2 | `TestErrorKindAgreesWithLlmKind` | `ErrorKind` calling `signin.ErrorKindOf` first for every error (the earlier code) | `errorkind_test.go:93: ErrorKind(llm: workspace needed) = "other", llm.Kind "workspace"` |
| 2 | `TestErrorKindAgreesWithLlmKind, TestErrorKindIsTheContracts` | `ErrorKind` never using the SDK's word | `errorkind_test.go:76: a denied sign-in: ErrorKind = "other", want "denied"` |
| 3 | `TestRefusedBackgroundCarriesConsent` | `afterRefusal` keeping the SDK's text when the answer reads no grant | `plan_test.go:421: error -32603 {"consent":"chatgpt_plan.background_use","error":"Summarization failed: openai: a background request needs the consent","kind":"not_permitted","provider":"fake"}, want kind not_permitted, c…` |
| 4 | `TestSignInGrokBrowser, TestSignInDevice` | a wrong word in `sessionKinds` (Grok's row `chatgpt`) | `signin_test.go:219: Accepts(grok) refused the session the SDK stored: Login{Issuer:https://auth.x.ai ClientID:b1a00492-073a-47ea-816f-4c329264a828 AccountID: TokenURL:https://auth.x.ai/oauth2/token FedRAMP:false Expires:…` |
| 5 | `TestConsentWordingOnEverySurface` | `prompter.Choose` replacing the wording with "Allow?" | `consentwording_test.go:139: condition not met within 5 s` |
| 6 | `TestRetryCausesAreAcpexts` | `llm/kind.go` keeping `rate-limited`, so a provider retry's class is the old word | `retrycause_test.go:48: retry causes ["empty_answer" "rate-limited"], want "empty_answer" and "rate_limited"` |
| 6 | `TestEveryCommandAndWordIsAccepted` | openai's word list given `device` | `defaultmethod_test.go:118: openai: method words ["api-key" "chatgpt-plan" "device"], want ["api-key" "chatgpt-plan"] for its offered methods` |
| 7 | `TestOfferedConsentTextIsGobbles` | `ConsentText` dropped from the wizard options (the confirm step shows the SDK's text) | `policy_test.go:182: the confirm step asked "You're using your ChatGPT plan: this app's requests count toward its usage, which you can manage on the usage page. Allow this app to also use your plan for background requests…` |
| 8 | `TestLogoutReadsTheStoredMethod` | `Logout` reading `Revocable(id)` for the provider | `login_test.go:562: logout: <nil>, <nil>, stored Login(nil), revoked ["plan-r"]; want it deleted and never revoked` |
| 9 | `TestTitleGateReadsOnlyWhenATitleIsAsked` | the gate asked before the title's conditions | `plan_test.go:482: the gate was read 4 times over three prompts, want once` |
| 3 | `TestTitleNotMadeSendsNothing` | the gate reading a grant | `plan_test.go:447: 1 title requests, want none` |
| 10 | `TestAuthConsentNeverFailsAfterTheWrite` | `Consents` read again after `SetConsent` | `consentstore_test.go:44: allow by background: exit 1, stdout "", stderr "Error: auth.json: the file went bad after the write\n"; want the stored answer reported, exit 0` |

**Gates after the last code change** (the code changed after the first pass only by lint's `errcheck`: two `errors.AsType` calls read with `errors.As`; the
ten plants on `ErrorKind` and on the overflow's refusals were run again after it, all failing). The first gate started at 15:23:43 (the `pre-add-check` of the final tree, after an earlier pass at 15:20:27
that found the two `errcheck` findings below); every Go file below was last modified before it.

| Gate | Where | Start to end | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check FILES=` the 52 changed Go files | this Mac | 15:23:43 to 15:24:37 | 0 ("52 file(s) clean") |
| `golangci-lint run --config .golangci.yml`, seven packages, six runs, a new `GOLANGCI_LINT_CACHE` each, `run-heavy` | this Mac | 15:24:37 to 15:25:57 | 0, 0, 0, 0, 0, 0; "0 issues." each |
| `make check-acpext-deps` | this Mac | 15:26:14 to 15:26:14 | 0 |
| `make race`, then `GOFLAGS=-count=1 make race`, `run-heavy` | this Mac | 15:26:14 to 15:27:02, then 15:27:23 to 15:28:04 | 0, 0 |
| `make check-test-isolation`, `run-heavy` | this Mac | 15:27:02 to 15:27:23 | 0 |
| `go test ./...`, `-race ./...`, acpext's tests, `run-on-host --need linux --race` | the Linux machine, scratch copy | 15:26:13 to 15:30:05 | 0 |
| the same three, `--need windows --race`, `sh -c` | the Windows laptop, scratch copy | 15:26:13 to 15:28:37 | 0 |
| `make preflight`, `run-heavy` | this Mac | 15:30:18 to 15:38:19 | 0 ("preflight passed") |

The first lint pass of this round, before the final tree, found two `errcheck` findings (an `errors.AsType` result discarded in `list.go` and in
`autocompact.go`), in six runs of six; both were fixed in the files, without `nolint` and without touching the config, and the six runs above are the
final tree's, six of six clean. The changed Go files, by last-modified time:

```text
13:09:12  acpclient/update.go
13:09:12  internal/cli/view.go
13:09:12  llm/kind.go
13:09:12  llm/kind_test.go
13:09:30  agent/event.go
13:09:54  acpclient/retry_test.go
13:09:54  acpserver/prompt_test.go
13:09:54  acpserver/retry_test.go
13:09:54  internal/cli/retry_test.go
13:10:22  internal/cli/models.go
13:10:49  llm/provider/list_test.go
13:10:49  llm/provider/workspace_test.go
13:11:02  llm/provider/limit_test.go
13:18:45  llm/background.go
13:18:45  llm/consent.go
13:22:22  internal/cli/acp.go
13:22:22  internal/cli/piimport.go
13:22:28  internal/cli/providerscmd.go
13:24:32  internal/cli/acp_test.go
13:24:32  internal/cli/auth_test.go
13:24:32  internal/cli/choice_test.go
13:24:32  internal/cli/providerscmd_test.go
13:27:47  internal/auth/resolve_test.go
13:30:13  acpserver/errordata_test.go
13:31:02  acpserver/prompt.go
13:31:09  acpserver/agent.go
13:33:21  internal/cli/agent.go
13:33:34  internal/cli/credentials.go
13:35:54  acpserver/autocompact_test.go
13:35:54  acpserver/title_test.go
13:38:41  llm/provider/mark_test.go
13:39:34  internal/cli/consent_test.go
13:48:29  llm/consent_test.go
13:49:47  llm/provider/provider.go
14:01:50  internal/auth/resolve.go
14:49:18  acpserver/title.go
14:49:18  internal/cli/auth.go
14:49:18  llm/provider/consent.go
14:49:18  llm/provider/login.go
14:49:18  llm/provider/policy.go
14:49:18  llm/provider/signin.go
14:51:34  acpserver/plan_test.go
14:52:15  acpserver/retrycause_test.go
14:52:15  internal/cli/defaultmethod_test.go
14:52:40  llm/provider/policy_test.go
14:52:40  llm/provider/signin_test.go
14:53:27  llm/provider/errorkind_test.go
14:53:39  llm/provider/login_test.go
14:54:27  internal/cli/consentstore_test.go
14:54:27  internal/cli/consentwording_test.go
15:23:05  acpserver/autocompact.go
15:23:05  llm/provider/list.go
```

The identifier scan of the staged diff's added lines after this review (the same patterns as above) found nothing (`grep` exit 1) and found the one line of a
copy with planted values (count 1); it ran after every text of this entry but this paragraph, and `make check-records markdownlint` ran after it, exit 0.

#### Second review of 2026-11-15

The independent second-pass review closed every first-round finding and raised five items; the owner ruled on three more points. This entry is this phase's share; the
"Review of 2026-11-15" under 0052-PLAN's "P3 record (2026-11-15)" has the rest, and nothing above is rewritten but the sentence on the P6 rows the first entry had
(item 5).

**The coordinator's check under the owner's ruling "Accept the eight; I check 6 and 9"** (relayed by the coordinator): deviation 6 stands, checked against the list of the
unnamed tests and the reviewer's plants; deviation 9 is withdrawn on the evidence of Q-c, and so is not a deviation.

**The owner's rulings of 2026-11-15, relayed by the coordinator, recorded here and in 0051-MADR's "Revisions of 2026-11-15" (its second part), each with its annotations:**

* **CI on the branch:** "Open a draft PR now". The coordinator's earlier statement that CI runs only on pushes to `main` and tags was wrong: `.github/workflows/ci.yml`
  also runs on `pull_request` (the first review's finding). After P4 is committed and pushed the coordinator opens one draft pull request from `p3-acpext-v0.5.0` to
  `main`, so every later push to the branch runs CI; `main` does not move and nothing merges until the owner's word, and `main` is still fast-forwarded only after 0051 P6 and
  0052 P4 land. Every sentence of the first review that says the phases on the branch have only local gates behind them is annotated, dated, with this ruling (0051-MADR,
  this PLAN, 0052-PLAN: ten sites, each ending "holds only until that pull request is open").
* **Deviation 13**, `ConsentText` to `signin.Options`: "Accept; P6 builds the site". Written in both places: 0052-PLAN P3 step 1 and this PLAN's P6 step 1.
* **A key for the run with a plan session stored:** "Keep it, and document it". 0052-MADR D2 has the dated note, with the code lines; `docs/guides/credentials.md` the sentence,
  where the consent is explained.

**Item 4, annotations inside quotations: true, two.** The annotation about `main` sat inside the owner's quoted words at 0051-MADR's "Revisions of 2026-11-15" ("Release-candidate tag,
final tag last") and at this PLAN's "P3 record (2026-11-15)", where the same words are quoted. Both quotations are restored to `c1e9100`'s text and the annotation follows
each. Every annotation added in the three rounds was checked by script: for each paragraph of 0051-MADR, this PLAN, 0052-PLAN and 0049-PLAN, no annotation of 2026-11-15 is
preceded by an odd number of quotation marks in its paragraph (0 of them), and the diff against `c1e9100` of the two restored quotations is the one appended annotation.

**Item 5, the P6 row annotations decided more than the ruling: true.** The ruling covers facts. Rewritten in place: the annotations on `TestAuthenticateOnlyRemoteMethods`,
`TestAuthenticateGrokBrowserHasNoPaste` and `TestAuthenticateRemoteRefusesLoopbackOnly` now state only the pinned facts, the ruling, that the row as written cannot pass at
them and why; what replaces them (keep the loopback-only rule's tests through a stand-in method, or drop the rows because no offered method has that shape) is an open question
for the owner before P6 runs, added under P6 in the PLAN's form ("Open question (2026-11-15)", with the two options and no choice made). The first plant of `TestAuthenticateOnlyRemoteMethods`,
"the loopback method chosen by default", is annotated as needing replacement under either option. P4's `TestRemoteMethods` row names the plants actually run (`Remote` reading
`Completion` alone, so a loopback-only method of the contract's own shape is accepted; `Remote` refusing every browser completion), and says the row's own first plant cannot fail the
test at the pinned facts. The stand-in, "P6 builds to this annotation" and "`Command` the host command" phrases are gone; so is the first entry's sentence that P6's step 1 adds a seam or
its test builds the relay's offer directly.

**Item 7, nothing guarded `storedMethod`'s first-match choice: true.** New `TestSameKindMethodsAgreeOnRevocable` (`llm/provider/policy_test.go`) takes, for every built-in provider,
the offered session methods that share a session kind and requires them to agree on `Revocable` (a method with no kind in `sessionKinds` fails it too). Plant, through the `offeredFor`
seam (Grok's `browser_oauth` made non-revocable, `device_code` staying revocable): `policy_test.go:232: grok: browser_oauth (Revocable false) and device_code (Revocable true) sign in to the same grok session`.

**Item 8, `offeredFor` is half a seam: true; the reviewer's list of readers is right.** Its doc comment (`llm/provider/policy.go:40`) now says exactly which functions read it and that a
swap changes only those. The readers are `yields` (`policy.go:111`), which `HasLogin`, `HasKeyLogin`, `Revocable`, `ImportsPiLogin` (`:122`, `:126`, `:135`, `:157`) and `consent.go`'s
`consentMethod` and `storedMethod` (`:81`, `:101`) call, and through them `Accepts`, `Logout`, `PlanConsent`, `Consents`, `SetConsent` and the adapter's `planConsentOf`. `Offered`,
`DefaultMethod` (`:142`), `HasSignIn`, `Methods` and `skips` (`signin.go:86`, `:99`) read `Offered` directly. Not widened.

**Item 9, carried to P5 and P6, no code now.** `ErrorKind` takes the SDK's word only for a `*SignInError` (or the SDK's denied and expired markers), as 0051-MADR D6 reads
"an error of the SDK's sign-ins". At `v1.31.0` the SDK's sign-ins also return errors wrapping `llmprovider`'s sentinels, and plain errors; today every one reaches `ErrorKind`
wrapped by `signin.go`'s `SignIn`. A later direct caller that passed a raw SDK error, P5's `--json` driver or P6's `signin.Start`, would get `other`. It is D6 carried out, not
a new decision: dated pointers are at P5 step 2 and P6 step 1, saying that errors from those calls are wrapped in `SignInError` before `ErrorKind` reads them and that a test in
the phase holds it.

**Item 6 (the review's granted-yet-refused case) is 0052 P3's, in its record.** Its plant, every overflow error returned unwrapped, is in the table below.

**Plants of this round, each on a scratch copy** (each exits 1; the plants on `afterRefusal`, the overflow's refusals and `ErrorKind` were run again on this tree):

| Item | Test | Planted break | Failure seen |
| :--- | :--- | :--- | :--- |
| 6 | `TestRefusedBackgroundCarriesConsent` | every overflow error returned unwrapped (`autocompact.go`) | `plan_test.go:431: error -32603 {"error":"llm: not permitted: llmprovider: not permitted: openai: a background request on a ChatGPT plan session needs the consent chatgpt_plan.background_use, which the user has not granted (0075-MADR D5)",…}, want … naming "a background request on a ChatGPT plan session needs the consent"` |
| 7 | `TestSameKindMethodsAgreeOnRevocable` | two same-kind methods that disagree, through the `offeredFor` seam | `policy_test.go:232: grok: browser_oauth (Revocable false) and device_code (Revocable true) sign in to the same grok session` |

**Gates after the last code change** (test files are code; no Go file changed after the first, `pre-add-check`, at 16:11:44):

| Gate | Where | Start to end | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check FILES=` the 52 changed Go files | this Mac | 16:11:44 to 16:12:39 | 0 ("52 file(s) clean") |
| `golangci-lint run --config .golangci.yml`, seven packages, six runs, a new `GOLANGCI_LINT_CACHE` each, `run-heavy` | this Mac | 16:12:45 to 16:14:09 | 0, 0, 0, 0, 0, 0; "0 issues." each |
| `make check-acpext-deps` | this Mac | 16:14:17 to 16:14:17 | 0 |
| `make race`, then `GOFLAGS=-count=1 make race`, `run-heavy` | this Mac | 16:14:17 to 16:15:01, then 16:15:20 to 16:16:01 | 0, 0 |
| `make check-test-isolation`, `run-heavy` | this Mac | 16:15:01 to 16:15:20 | 0 |
| `go test ./...`, `-race ./...`, acpext's tests, `run-on-host --need linux --race` | the Linux machine, scratch copy | 16:14:17 to 16:18:31 | 0 |
| the same three, `--need windows --race`, `sh -c` | the Windows laptop, scratch copy | 16:14:17 to 16:16:35 | 0 |
| `make preflight`, `run-heavy` | this Mac | 16:18:47 to 16:26:48 | 0 ("preflight passed") |

The changed Go files' last-modified times, all before the first gate's start (16:11:44):

```text
13:09:12  acpclient/update.go
13:09:12  internal/cli/view.go
13:09:12  llm/kind.go
13:09:12  llm/kind_test.go
13:09:30  agent/event.go
13:09:54  acpclient/retry_test.go
13:09:54  acpserver/prompt_test.go
13:09:54  acpserver/retry_test.go
13:09:54  internal/cli/retry_test.go
13:10:22  internal/cli/models.go
13:10:49  llm/provider/list_test.go
13:10:49  llm/provider/workspace_test.go
13:11:02  llm/provider/limit_test.go
13:18:45  llm/background.go
13:18:45  llm/consent.go
13:22:22  internal/cli/acp.go
13:22:22  internal/cli/piimport.go
13:22:28  internal/cli/providerscmd.go
13:24:32  internal/cli/acp_test.go
13:24:32  internal/cli/auth_test.go
13:24:32  internal/cli/choice_test.go
13:24:32  internal/cli/providerscmd_test.go
13:27:47  internal/auth/resolve_test.go
13:30:13  acpserver/errordata_test.go
13:31:02  acpserver/prompt.go
13:31:09  acpserver/agent.go
13:33:21  internal/cli/agent.go
13:33:34  internal/cli/credentials.go
13:35:54  acpserver/autocompact_test.go
13:35:54  acpserver/title_test.go
13:38:41  llm/provider/mark_test.go
13:39:34  internal/cli/consent_test.go
13:48:29  llm/consent_test.go
13:49:47  llm/provider/provider.go
14:01:50  internal/auth/resolve.go
14:49:18  acpserver/title.go
14:49:18  internal/cli/auth.go
14:49:18  llm/provider/consent.go
14:49:18  llm/provider/login.go
14:49:18  llm/provider/signin.go
14:52:15  acpserver/retrycause_test.go
14:52:15  internal/cli/defaultmethod_test.go
14:52:40  llm/provider/signin_test.go
14:53:27  llm/provider/errorkind_test.go
14:53:39  llm/provider/login_test.go
14:54:27  internal/cli/consentstore_test.go
14:54:27  internal/cli/consentwording_test.go
15:23:05  acpserver/autocompact.go
15:23:05  llm/provider/list.go
16:07:49  acpserver/plan_test.go
16:08:15  llm/provider/policy.go
16:08:15  llm/provider/policy_test.go
```

The identifier scan of the staged diff's added lines after this round (the same patterns) found nothing (`grep` exit 1) and found the one line of a copy with planted values
(count 1); it ran after every text of this entry but this paragraph, and `make check-records markdownlint` ran after it, exit 0.
