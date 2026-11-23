---
status: complete
date: 2026-11-21
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
*(2026-11-15: built, staged on the branch `p3-acpext-v0.5.0` for the coordinator to commit; see "P5 record (2026-11-15)". One deviation was ruled by the owner, "Accept the bytes as built": the SDK's types written by `jsonx` carry a `"secret":false` and unescaped `&`, `<`, `>` (deviation 1); `docs/guides/credentials.md`'s `--json` section, the P4 record's open row, is rewritten.)* *(2026-11-16: a second review; the owner's ruling "Emit the failed step from the parser's own context" adds `internal/cli/main.go` to P5's files by deviation 2 of "Review of 2026-11-15 (P5), second part".)*

### P6 — ACP `authenticate` by elicitation (MADR D4)

*(2026-11-15.)* **Open question for the owner, before P6 runs:** the three rows annotated below (`TestAuthenticateOnlyRemoteMethods`,
`TestAuthenticateGrokBrowserHasNoPaste`, `TestAuthenticateRemoteRefusesLoopbackOnly`) cannot pass as written at go-llmprovider-sdk `v1.31.0`, where Grok's
`browser_oauth` is `browser_paste` with a paste, so `Remote` accepts it (the owner's ruling "Amend the records to the SDK's facts"). What replaces them is
undecided: (a) keep the loopback-only rule's tests through a stand-in method of that shape offered to the relay, or (b) drop the rows because no offered method has
that shape, the rule staying in D4 untested. The plant of the first row, "the loopback method chosen by default", needs replacing under either. Recommended: (b) is
smaller and (a) holds the rule; the choice is the owner's.

*(2026-11-15, the owner's ruling, relayed by the coordinator, answering the question above: "Keep the tests through a stand-in".)* Option (a). P6 tests the loopback-only refusal and the no-paste flow with a stand-in method of the contract's shape, offered to the relay through its options, and one row is restated for Grok's browser sign-in by paste for a remote client. The stand-in is not designed here and no row is rewritten here: P6's brief designs the stand-in and words the rows. Until then the three rows below keep their annotations of this date, each with the pointer that follows it.

*(2026-11-16, P6's run; the owner's ruling "Keep the tests through a stand-in", relayed by the coordinator, carried out. Written before anything was built; "P6 record (2026-11-16)" says what was built.)* **The stand-in and the rows.**

* **How the stand-in reaches the relay.** Through `AuthOptions.Methods`, `AuthOptions.Remote`, `AuthOptions.Command` and `AuthOptions.SignIn`, which step 2 makes injectable; `provider.Offered` is not changed and no production path exists for the test. The test's `Methods` returns Grok's `browser_oauth` taken from `provider.Offered("grok")` and rewritten to the contract's own loopback-only shape (`Completion` `browser_loopback`, `Paste` empty, no `paste` feature), then Grok's `device_code` unchanged; its `Remote` is `provider.Remote` and its `Command` is `provider.MethodCommand`, both production; its `SignIn` returns a scripted run (a type of the test) whose steps have the shape `signinflow` gives a browser sign-in with no paste: `open_url` with no `accepts`, then `done` when the test lets the loopback finish. A scripted run is a stand-in for the engine, not for the relay: every test of the relay's decisions uses it, and the real `x/signin` runs against an in-process issuer in `llm/provider` and in `internal/cli` (which, like `llm/provider`, may import what rule 4 allows there; `acpserver` may not import `x/signin`).
* **What `Remote` and D4 give for the default.** Grok's offered methods, in the SDK's order, are `api_key`, `browser_oauth`, `device_code` (`llmprovider/providers/grok/descriptor.go` at `v1.31.0`). D4 step 1: "With no method named, a remote sign-in takes the first offered method `Remote` accepts". `Remote(api_key)` is false (`Completion` `value`), and `Remote(browser_oauth)` is true (`browser_paste`, `Paste` `url_or_code`). The default for a remote Grok client is therefore `browser_oauth`, completed by a paste (`NoLoopback` set), not the device sign-in the row said. A local client with no method named is asked, by the enum of step 3, to choose between `browser_oauth` and `device_code`, and a local `browser_oauth` completes by the loopback or a paste.
* **The rows, restated at those facts.** Each keeps what it held.
  * `TestAuthenticateOnlyRemoteMethods`, restated for Grok's browser sign-in by paste for a remote client (the one row restated that the owner's "Keep the tests through a stand-in" allows): grok, `_meta.gobble.remote: true`, no method named, through `provider.Offered` and `provider.Remote`: `SignIn` is called for `browser_oauth` with `remote` true; the first elicitation is the `open_url` step with `accepts` `url_or_code`, then the paste form, and the pasted callback completes the sign-in and answers `{}`; `browser_oauth` named gives the same and `device_code` named gives the device sign-in. Planted break: `remote` not passed on to `SignIn` (the recorded call holds false).
  * `TestAuthenticateRemoteDefault`, which the 2026-11-15 annotation did not name and which cannot pass as written either ("grok ... the device sign-in"): through the stand-in, offered `[browser_oauth (loopback-only), device_code]`, `remote: true` and no method named: the device sign-in, never the loopback-only method, with `remote` true; with grok's real methods the same request drives `browser_oauth`. Planted break: `remote` ignored in the choice, so the client is asked to choose between the methods in the enum form that only a local client gets. *(Written first as "the default taken from `Offered` without `Remote`"; with the stand-in, a loopback-only method is never driven unless named, whatever the filter, so that break does not fail and was replaced as the run's reading of D4 step 3 settled the order of the checks.)*
  * `TestAuthenticateGrokBrowserHasNoPaste`, renamed `TestAuthenticateLoopbackHasNoPaste` because no Grok sign-in has that shape: the stand-in named by a local client: one `open_url` elicitation whose `_meta.gobble.step` has no `accepts`, no `paste` form after its `accept`, and the loopback's finish (the scripted run's `done`) completes it with `{}`; and, for a client with `elicitation.url`, URL mode then `elicitation/complete`. Planted break: the relay sending a `paste` form after every acknowledged `open_url`.
  * `TestAuthenticateRemoteRefusesLoopbackOnly`, through the stand-in: `method: "browser_oauth"` and `remote: true`: `auth_required` with `ErrorData.Kind` `unsupported`, a `Reason` saying the method completes only in a browser on the agent's host, and `Command` `gobble auth login grok --method browser` (what `provider.MethodCommand` gives for that method id); no elicitation; `SignIn` never called; the same request without `remote` reaches `SignIn` with `remote` false. Planted break: the relay's remote filter dropped, so `SignIn` is called.
* *(2026-11-16, later, the owner's ruling relayed by the coordinator: "Ask the client to choose". The two bullets above that read D4 step 1 as "take the first, no question" are amended, not rewritten: with no method named and `remote: true`, a client is sent the one-field enum form of step 3 when more than one offered method `Remote` accepts, the enum listing only those methods in the offered order, and with exactly one the relay starts that one with no question; a method named is never asked about. At `v1.31.0` Grok has two such methods, `browser_oauth` by paste and `device_code`, so a remote Grok client is asked. `TestAuthenticateOnlyRemoteMethods` and `TestAuthenticateRemoteDefault` hold this: the enum's members are exactly the remote-capable methods, in the offered order, each choice starts its method with `remote` true, a provider with one remote-capable method is not asked, and a remote client that did not advertise `elicitation.form` gets `auth_required` with the command, as a local one does (MADR D4 step 3). Planted breaks: `remote-takes-first` (the choice dropped for a remote client), `enum-not-remote-filtered`, `remote-no-form-signs-in`.)*
* **A name the PLAN did not give.** The host command of a refused method needs the CLI's word for the method (`--method browser`), which only `llm/provider` holds (`methodIDs`): `AuthOptions.Command func(id string, m authcontract.Method) string`, which `internal/cli/agent.go` points at the new `provider.MethodCommand(id, methodID string) string`. It is a seam of the same kind as `Remote`, and within step 2.

1. `llm/provider/signinrun.go`: `StartSignIn(ctx, id, method string, o SignInOptions, store LoginStore) (*SignInRun,
   error)` over `signin.Start`, with `signin.Options` as MADR D3 lists it: `Identity: {Name: "gobble", Version}`,
   `AgentName: "gobble"`, `HostID` from `o.HostID` (0052-PLAN P2's `<state>/host-id`), `ClientStore` the `sdkStore`
   adapter when `store` has a client store (0052-PLAN P2 step 4; nil otherwise, which `x/signin` reads as none),
   `ConsentText` gobble's wording (0052-PLAN P3) *(2026-11-15, the owner's ruling "Accept; P6 builds the site", relayed by the coordinator: this is the `signin.Options` half of 0052-PLAN P3 step 1, which that phase could not build because this site is built here; 0052-PLAN P4's `TestConsentWordingOverACP` guards it. A second pointer, carried from the review of 2026-11-15: an error from `signin.Start` or `Wait` is wrapped in `provider.SignInError` before `ErrorKind` reads it, because `ErrorKind` takes the SDK's word only for a `*SignInError` or the SDK's denied and expired markers (0051-MADR D6), and a test in this phase holds it.)*, and `Previous` the session `store` holds for `id` when its kind is the
   one the method signs in to. `Wait` returns `provider.SignInResult{Key string; Yields authcontract.Yields}` (MADR
   D3): it saves the session through `store` (`sdkStore`) or returns the key, then runs the listing check.
   *(2026-11-16, the owner's ruling, relayed by the coordinator: "Check first, save only if accepted". The order of that
   sentence is reversed: `Wait` runs the listing check with the new credential first, listing through the new session itself (or with the key
   as the listing's key), and saves the session through `store`, or returns the key, only when the listing accepts. A
   refusal (the listing's `401`, kind `auth`, not retryable) fails the sign-in and leaves what was stored byte for byte,
   with no revocation; a listing that fails otherwise (a `403`, a `503`, a failed renewal) does not, as in the wizard
   (`refusedCredential`), which saves after its listing loop; and, as the wizard's save does, `Wait` waits for any
   renewal the listing started and saves the renewed tokens, and saves nothing once its context is done. (The first
   build did none of those three; "P6 record (2026-11-16)", part "Review and fixes", says what was fixed.) The sentence's
   order contradicted MADR D4 step 3, "a sign-in that fails, is declined or is cancelled changes nothing stored", under
   `fresh`. Options not chosen: save, then restore the old credential on a refusal; keep the order and amend D4. "P6
   record (2026-11-16)" says what was built.)*
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
   by the MADR's table *(2026-11-15, the owner's ruling, relayed by the coordinator, "Accept the bytes as built", "P5 record (2026-11-15)", deviation 1: a step written to a client by `jsonx` carries `"secret":false` and does not escape `&`, `<` and `>`, unlike the SDK's own encoding and its vectors; it is accepted for this path too, and a test of the steps' bytes here must not hold them to the vectors' bytes)*; an `open_url` or `show_code` elicitation is acknowledged, not answered (MADR D4): its
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
   | `TestAuthenticateOnlyRemoteMethods` | grok with no method named: the device sign-in, never `browser_oauth`; `browser_oauth` when named *(2026-11-15, the owner's ruling "Amend the records to the SDK's facts": at go-llmprovider-sdk `v1.31.0` Grok's `browser_oauth` is `Completion` `browser_paste` with `Paste` `url_or_code`, so `Remote` accepts it and it is not a loopback-only method, so with no method named a remote sign-in no longer has the device sign-in as the only method it can drive, and the row as written, "never `browser_oauth`", cannot pass at those facts. Its planted break, "the loopback method chosen by default", names a method that no longer exists among the offered ones. What replaces the row and its plant is an open question for the owner before P6 runs; see "Open question (2026-11-15)" under P6.)* *(2026-11-15, the owner's ruling, relayed by the coordinator: "Keep the tests through a stand-in". The row is kept and tested through a stand-in method of the contract's shape; see the answer under "Open question (2026-11-15)" under P6. Its wording is for P6's brief.)* | the loopback method chosen by default |
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
   | `TestAuthenticateGrokBrowserHasNoPaste` | grok's `browser_oauth`, named: one `open_url` elicitation whose `_meta.gobble.step` has no `accepts`, no `paste` form after its `accept`, and the loopback's finish completes it *(2026-11-15, the owner's ruling "Amend the records to the SDK's facts": at go-llmprovider-sdk `v1.31.0` Grok's `browser_oauth` is `Completion` `browser_paste` with `Paste` `url_or_code`, so `Remote` accepts it and it is not a loopback-only method, so the row as written, a Grok browser sign-in with no `accepts` and no `paste` form, cannot pass at those facts: its `browser_oauth` has both. Whether the no-paste shape keeps a test, through a stand-in method, or the row is dropped because no offered method has that shape is an open question for the owner before P6 runs; see "Open question (2026-11-15)" under P6.)* *(2026-11-15, the owner's ruling, relayed by the coordinator: "Keep the tests through a stand-in". The row is kept and tested through a stand-in method of the contract's shape; see the answer under "Open question (2026-11-15)" under P6. Its wording is for P6's brief.)* | the relay sending a `paste` form after every acknowledged `open_url` |

   `TestAuthenticateBadPasteAsksAgain` also counts: exactly two re-asks, three `paste` forms in all, before the
   `failed` step.

   *(2026-11-11, later, added, for MADR D2's, D3's and D4 step 1's notes of that date: the remote sign-in.)* Each seen
   failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestAuthenticateRemoteBindsNoPort` | a form-only client's `authenticate` for openai with `_meta.gobble.method: "chatgpt_plan"` and `remote: true`, against `planIssuer`, while the test holds `127.0.0.1:1455` and `:1457` (binding each, or finding it already bound, either of which leaves it unavailable): the sign-in starts, its authorize URL's `redirect_uri` names port 1455, a pasted callback completes it, and `{}` follows; the recorded `signin.Options.NoLoopback` is true | `remote` not passed on to `StartSignIn`, so the listener's bind fails on the held ports and no `open_url` form is sent |
   | `TestAuthenticateLocalKeepsListener` | the same `authenticate` with `remote` absent, and again with `remote: false`: the recorded `signin.Options.NoLoopback` is false. A recorder stands in for `signin.Start` here, so the test binds no real port on the host | `NoLoopback` set for every ACP sign-in |
   | `TestAuthenticateRemoteRefusesLoopbackOnly` *(2026-11-15, the owner's ruling "Amend the records to the SDK's facts": at go-llmprovider-sdk `v1.31.0` Grok's `browser_oauth` is `Completion` `browser_paste` with `Paste` `url_or_code`, so `Remote` accepts it and it is not a loopback-only method, so grok with `method: "browser_oauth"` and `remote: true` is not refused and the row as written cannot pass at those facts. Whether the refusal keeps a test, through a stand-in method, or the row is dropped because no offered method has that shape is an open question for the owner before P6 runs; see "Open question (2026-11-15)" under P6.)* *(2026-11-15, the owner's ruling, relayed by the coordinator: "Keep the tests through a stand-in". The row is kept and tested through a stand-in method of the contract's shape; see the answer under "Open question (2026-11-15)" under P6. Its wording is for P6's brief.)* | grok with `method: "browser_oauth"` and `remote: true`: `auth_required` with `ErrorData.Kind` `unsupported`, a `Reason`, and `Command` `gobble auth login grok --method browser`; no elicitation; the `signin.Start` variable never called; and the same request without `remote` reaches the recorder, which stands in for `Start`, with `NoLoopback` false | the relay's remote filter dropped, so `Start` is called (the recorded calls count one) |
   | `TestAuthenticateRemoteDefault` | grok with no method named and `remote: true`: the device sign-in, and the recorded `NoLoopback` true | the default taken from `Offered` without `Remote` |
   | `TestAuthenticateRemoteRefusesNonBoolean` | `_meta.gobble.remote: "yes"` is -32602, and nothing starts | the value read as truthy |
   | `TestStartSignInRemoteSetsNoLoopback` (`llm/provider`) | `SignInOptions{Remote: true}` gives `signin.Options.NoLoopback` true, for `chatgpt_plan` and for `device_code`; without `Remote`, false; the terminal paths' `SignIn` never sets it | the option left false whatever `Remote` says |
   | `TestStartSignInRemoteDevice` (`llm/provider`) | a remote grok device sign-in against `testIssuer`, `NoLoopback` set, starts with no `ErrUnsupported` from `Start` and completes by the poll, and the `signin.Options` it ran with has `NoLoopback` true (MADR D3's note: the reading to confirm with the SDK) | `Remote` not passed through for a device method, so `NoLoopback` is false |

   A failure of `TestStartSignInRemoteDevice` that comes from the SDK's `Start`, not from a plant, is a deviation (C4):
   MADR D3's note says why, and it is settled in go-llmprovider-sdk's 0074 record, not here.

   *(2026-11-16, the review of P6; the owner's ruling, relayed by the coordinator, "Assert what gobble passes; use no real ports". The first row of this table,
   `TestAuthenticateRemoteBindsNoPort`, which holds ports 1455 and 1457, and the dial of the redirect's port that "P6 record (2026-11-16)" built in its place, are dropped: a probe of a
   real port could not fail on its defect. The row is restated: a remote sign-in sets `NoLoopback` for every offered method, held by `TestStartSignInRemoteSetsNoLoopback` at the seam where
   gobble starts the sign-in, and that a remote `authenticate` reaches that start with `Remote` true, held at the `AuthOptions.SignIn` seam in `acpserver` and in `internal/cli`
   (`TestAgentRemoteReachesTheEngine`); whether the engine then binds no port is the SDK's own test. Its planted break: `NoLoopback` not set for one method.)*

   *(2026-11-17, later: for the two terminal plan sign-in tests of 0052-PLAN P4, `TestSignInChatGPTPlanTerminal` and `TestPlanAfterLogoutReusesClientTerminal`, this ruling is not applied. By the owner's ruling of that day, relayed by the coordinator, "Keep them until the SDK offers a no-listener option", they bind the SDK wizard's callback port 1455, on both loopback families, or 1457 when 1455 is taken (go-llmprovider-sdk `v1.31.0`, `llmprovider/auth/oauth_loopback.go`, `listenOpenAILoopback`), until go-llmprovider-sdk offers a wizard option to skip the listener. The failure modes: alone, 20 of 20 pass, and the independent reviewer's listener census over those 20 runs saw only port 1455 bound; with 1455 held, the wizard falls back to 1457 and passes; with both held, they fail at once, with a message that names the collision (these three measured); that a real Codex login starting while a test holds 1455 cannot bind it is reasoned, not measured. The request for the option is on the coordinator's ledger for the SDK and is not filed in any record of that repository. The tests switch to the option in the phase that bumps the SDK pin to the release that has it. See 0052-PLAN P4's "second review and fixes".)*

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

*(2026-11-16: built and staged; see "P7 record (2026-11-16)".)*

### P7a — check_libs.py's advisories by package (added 2026-11-14; the owner's ruling on P2's observation)

Not started. Its file: `scripts/check_libs.py`, and its record in this PLAN. No Go code changes.
*(2026-11-16, "Yes, in P7a, by a dated amendment": the owner's ruling, relayed by the coordinator, on the 30-advisory page
limit. `advisories` asks GitHub's API for one page, 30 advisories by default, with no pagination, so a repository with more
than 30 advisories would have its later ones unread. As put to the owner: the function follows the API's pages until it
has every advisory, with a self-test case over fixed two-page data and a planted break (the second page dropped); same
function, same phase, one commit. P7a's scope grows by this one item: `advisories` reads every page, steps 3 and 4 cover
it, and the docs that describe the check say so (C9). Options not chosen: record it and fix it later; leave it. See "P7a
record (2026-11-16)".)*

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
   *(2026-11-16, "Count it against every library of that repository": the owner's ruling, relayed by the coordinator.
   As put to the owner: fail closed, as today: an advisory that does not say which module it covers is tested against
   both the SDK's and the contract's pinned tag, and the message says the entry names no known module, so the owner knows
   to fix the advisory's package name; a mis-typed advisory can never hide a vulnerability. Options not chosen: ignore
   it; fail the check outright whatever the version. "Names no known module" means: `package` or `package.name`
   missing, null or empty, or a name that equals the `module` of no library the script lists for that repository. Step
   1's answer is in "P7a record (2026-11-16)".)*
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

*(2026-11-20, before this phase starts. P7a's code is on `main` at `1305a61` (`feat(check-libs): match advisories by package and read all pages`), an ancestor of `9332cef`. The P7a record's "staged, not committed" described 2026-11-16. Step 0 is met. Probes L1 to L3 have not been run. Grok's `browser_oauth` at the go-llmprovider-sdk `v1.31.0` tag is still `Completion` `browser_paste` (`llmprovider/providers/grok/descriptor.go`). Issue #16 is open. This note does not close it. The gates are this repository's Makefile, as 0052-PLAN P5's tooling note of this date lists them. The sibling go-tui-lib `tools/verify` default gates are not these. P8a's pin commit is that phase's `go get`, not the fleet `scripts/bump-libs.sh`, which commits a pin in every dependent it finds.)*

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

*(2026-11-20: `git diff acpext/v0.5.0-rc.1 HEAD -- acpext/` is empty at `9332cef`, so a tag made on that commit names the same `acpext/` as the candidate. The root still requires `v0.5.0-rc.1`. The owner tags `acpext/v0.5.0`. The pin commit waits until that tag is fetchable.)*

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
* `docs/guides/credentials.md:338-376` (`--json`'s events, and ACP `authenticate`) — open: P5 makes `--json` print the contract's steps and refuse a first plan sign-in with exit 2, and P6 makes `authenticate` sign in by elicitation. *(2026-11-16: closed by P6, see its record.)*
* `README.md:109-120` (the credentials bullet) — fixed: ChatGPT plan offered, `--method chatgpt-plan`, `gobble auth consent`, the question and what it gates.
* `docs/architecture.md:24`, `:109`, `:110` and `:127` (the command list, the `llm`, `llm/provider` and `acpserver` rows) — fixed: `auth consent`, the contract's words, `WithBackground` and `ConsentError`, the policy, `ErrorKind`, `ConsentText`, the consent functions, the gate and `ErrorData.Consent`.
* `docs/guides/credentials.md:118` (`oauth_unused`) — fixed: a session of a kind gobble does not sign in to is not used.
* `docs/guides/credentials.md:152-153` (`--method`'s words) — fixed: `chatgpt-plan` added.
* `docs/guides/credentials.md:189-217` (Signing in: the third provider, its consent wording and where it is asked) — fixed; `:245-300` (logout of a plan session, "Allowing background requests under a ChatGPT plan", `gobble auth consent`) — fixed, new; `:306` (`/login`'s methods) and `:347-348` (`gobble acp --login`) — fixed.
* `docs/guides/credentials.md:189-300` (the ChatGPT section) — open: 0052-PLAN P4 step 3 adds the files it keeps, the host id and the issued client, and the phone through an ACP client; P6 the elicitation. *(2026-11-16: P6 part closed, see its record; 0052-PLAN P4 step 3 still open.)* *(2026-11-16, later: closed by 0052-PLAN P4: the status and the phone's consent in `docs/guides/credentials.md`'s ChatGPT passage; see 0052-PLAN's "P4 record (2026-11-16)", its docs rows.)*
* `docs/guides/credentials.md:86-132` (`auth status --json`) — open: 0052-PLAN P4 step 2 adds `sessionKind`, `consents` and `consentTerms`. *(2026-11-16, later: closed by 0052-PLAN P4: `sessionKind`, `consents`, `consentTerms` and the text form's consent words are in the status section; see 0052-PLAN's "P4 record (2026-11-16)".)*
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

### P5 record (2026-11-15)

Run by a mainstream model at medium effort, on the coordinator's brief, under the owner's approval of 2026-11-15, relayed by the coordinator:
"Run P5 then P6 without stopping". This entry is P5's only: the coordinator reviews and commits it on the branch `p3-acpext-v0.5.0`, and briefs P6
separately. Times are the session machine's local time. Staged, not committed; **the coordinator commits on the branch.** `main` and `origin/main`
stayed at `9e21c8f`; no tag, push or merge was made (the owner's ruling "Keep it on the branch until those phases land", C8). The same brief had one
record task, the owner's answer to P6's open question, which is under P6 ("Open question (2026-11-15)" and the pointer at each of its three rows).

**C1, the start state.** Branch `p3-acpext-v0.5.0`, `HEAD` `c4c5ec0`, the tree clean; `git ls-remote origin` gave `refs/heads/p3-acpext-v0.5.0` at
`c4c5ec0` and `refs/heads/main` at `9e21c8f`; draft pull request 18 open on that head (`gh pr view 18`, read; the checks `gh pr checks 18` listed first,
the build, five identity jobs, docs, sbom, test-isolation and both validate jobs, pass, and `release` is skipped). `go.mod` requires acpext `v0.5.0-rc.1`,
go-llmprovider-sdk `v1.31.0` and `authcontract` `v0.1.0`, direct; no `replace`, no `go.work`. The lines this phase cites, re-read at this base: `openLoginURL`
is at `internal/cli/auth.go:251` (the PLAN's `:251-261`), its `--json` branch inside it; `loginEvent`, `loginEvents`, `showDeviceCode` and `saved` carried the four
events (`device_code`, `browser_url`, `saved`, `error`); the PLAN's fifth, `notice`, was `[WIP]` and no committed code prints it. No site changed in substance.

**What changed (staged).**

1. **Step 1, `llm/provider/signin.go`.** `SignInOptions.Steps func(authcontract.Step) error`; `SignIn` passes `wizard.Options.Driver`, `stepDriver`, which gives
   each step that shows something (`open_url`, `show_code`, `waiting`, and a kind the contract does not know yet) to `Steps`, answers none, returns `ErrAsks` for
   `paste`, `confirm` and `form`, and swallows the SDK's last step, `done` or `failed`, which the caller states once it has stored the result (the SDK's `done` comes
   before the wizard's listing check and before the caller's `SetKey`). `ErrAsks` is an error of the SDK's kind `unsupported`; `SignIn` returns it inside a
   `*SignInError` whose text is gobble's own (`asksText`, naming a terminal and an ACP client). `NoPastePrompt` is set when `NoPaste` is or `Steps` is, so no
   paste step is printed. `ShowDeviceCode` and `OpenURL` stay for the terminal paths; the PLAN's "replaces ... for a non-terminal caller" is that, and the SDK
   does not call them when a driver is set. A caller's own error from `Steps` is marked `ownError`, so `SignIn` returns it as it is (`errBrokenPipe` reaches the
   command). **One defect the real-wizard test found in this step:** `SignIn` took a device sign-in's `Credential.Device` from `ShowDeviceCode`, which the SDK
   does not call under a driver, so Kilo's key would have come back with `Device` false and been stored unquoted; a `show_code` step now sets it
   (`TestSignInStepsDevice`, the first plant).
2. **Step 2, `internal/cli/auth.go`, `login.go`.** `loginEvent`, `loginEvents` and their four events are gone. `jsonSteps` writes the `method` line
   (`{"event":"method","provider":ID,"method":<authcontract.Method>}`), then each `step` line (`{"event":"step","provider":ID,"step":<authcontract.Step>}`),
   by `jsonx.Marshal`. The steps the SDK states keep its `seq`; the last, `done` or `failed`, is gobble's, with the next `seq`: `done` after the key or session is
   stored (`saved`), `failed` from the error the command ends with (`fail`). `Run` clears stderr and the log's stderr as before; `runWith` reads the key, a missing
   one being a usage error and so one failed step, then writes the method line for `--api-key -`, and `done` with `yields: "api_key"` after `SetKey`; `loginWith` resolves the method as before, then, under
   `--json`, writes its method line, runs `provider.CheckNoQuestion`, and signs in with `Steps` set and `NoPaste` true. A failed step's kind is `provider.ErrorKind` of
   the error `signInWith` returned, which is what `provider.SignIn` returns, a `*provider.SignInError` (the carried item of the review: `TestSignInStepsErrorsAreSignInErrors`
   holds it in `llm/provider`, `TestAuthLoginJSONFailureKind` through the command). A usage error is `invalid_request`, exit 2, `seq` 1, and no method line, as its
   command refused it before a method was chosen; a refusal for `ErrAsks` is `unsupported`, exit 2, after the method line. Before a sign-in starts,
   `provider.CheckNoQuestion` (new) refuses a method with a required consent the stored session of that method holds no answer to, which the SDK would ask as a
   `confirm`: today the ChatGPT plan's first sign-in. `provider.OfferedMethod` (new) is the offered method a `--method` word names, so the method line is the SDK's
   fact. `saved` takes `*jsonSteps`, and `signInTo` too.
3. **Step 3.** `openLoginURL`'s `--json` branch is gone, and so is `showDeviceCode`'s: `--json` calls neither, because it sets `Steps`, not the two hooks.
4. **Step 4.** The tests below. `TestAuthLoginJSON` is rewritten for the steps: it keeps the old test's three other checks (nothing on stderr, even at debug level
   and with a log file that cannot open; a settings warning goes to the log, never stderr; a paste is offered on a terminal only without `--json`), adds that `Steps`
   is asked for exactly under `--json`, and its table of expected events, which fed the old test a fake `signInWith` that called the hooks, goes to the tests below,
   which run the real wizard.
5. **Step 5, docs.** See the docs step below.
6. **Step 6.** The gates, below.

**Tests (C2): each seen failing on a scratch copy of the tree** (`<TMP>`, no `.git`; `HOME`, `USERPROFILE`, `XDG_*` and `GOBBLE_HOME` in the scratch; `GOPROXY=off`;
`GOFLAGS=-mod=mod`; the module and build caches the real ones, read only; one `go test -count=1 -timeout 120s -run '^(Test...)$' <package>` each, run after the last
Go change; unbroken, each passes: `clean.out`'s sixteen runs, exit 0 each). The tests the PLAN names are `TestAuthLoginJSONSteps`, `TestAuthLoginJSONKiloKey`,
`TestAuthLoginJSONAPIKey`, `TestAuthLoginJSONUsageError`, `TestAuthLoginJSONRefusesAQuestion`, `TestAuthLoginJSONNoHostBrowser` and `TestAuthLoginJSONMatchesVectors`,
all in `internal/cli/loginjson_test.go`; the real wizard runs in them against an issuer in process (`internal/cli/loginissuer_test.go`), Grok's and Kilo's device
endpoints and Grok's browser callback, `planSignIn` for the plan's first sign-in, and the contract's vectors are read from the module's source (`go list -m -f
{{.Dir}}`, at the pinned tag). Each asserts what the PLAN's row says; the lines are decoded strictly (`RejectUnknownMembers`) into the contract's own types, so a
line of an old event, or a member the contract does not have, fails first. Each plant exits 1; a plant that failed to compile or hung was replaced before this table.

| Test | Planted break | Failure seen (exit 1 each; `<TMP>` is the scratch copy) |
| :--- | :--- | :--- |
| `TestSignInStepsDevice` (P5) | `SignIn` not setting its device flag on a `show_code` step, so Kilo's key comes back with `Device` false | `steps_test.go:76: kilo's device sign-in: {Key:kilo-token-0123456789 Device:false Workspace:}, <nil>` |
| `TestSignInStepsDevice` (P5) | the driver not passed to the wizard (`Driver` nil), so the SDK calls `ShowDeviceCode` and `Steps` is given nothing | `steps_test.go:45: ShowDeviceCode was called with Steps set` |
| `TestSignInStepsBrowserAsksNoPaste` (P5) | `NoPastePrompt` not set by `Steps` | `steps_test.go:104: grok's browser sign-in: grok sign-in: this sign-in asks a question, which a program that reads its steps cannot answer; sign in at a terminal (gobble auth login), or from an ACP client, saved map[]` |
| `TestSignInStepsRefuseWhatAsks` (P5) | the driver answering a `paste`, `confirm` or `form` step with no answer in place of `ErrAsks` | `steps_test.go:140: a first plan sign-in: context deadline exceeded (kind timeout), want a *SignInError holding ErrAsks, unsupported` |
| `TestSignInStepsRefuseWhatAsks`, `TestSignInStepsErrorsAreSignInErrors`, `TestSignInStepsDevice` (P5) | the driver giving `Steps` the last step, `done` or `failed` | `steps_test.go:65: grok's steps [show_code waiting done], want show_code then waiting, and no done / steps_test.go:166: Steps was given [done failed open_url show_code waiting a_kind_not_known_yet], want the four steps that show something` |
| `TestSignInStepsErrorsAreSignInErrors` (P5) | `SignIn` returning the SDK's error unwrapped, not the `*SignInError` | `steps_test.go:211: the error *fmt.wrapErrors llm: authentication failed: llmprovider: authentication failed: oauth: Grok device authorization denied is not a *SignInError` |
| `TestSignInStepsErrorsAreSignInErrors` (P5) | `Steps`' own error not marked as the caller's, so `SignIn` wraps it | `steps_test.go:206: Steps' error came back as grok sign-in failed (other), want it as it is` |
| `TestSignInStepsRefuseWhatAsks` (P5) | `SignInError`'s text not special-casing `ErrAsks` | `steps_test.go:143: the refusal's text "openai sign-in failed (unsupported)", want gobble's, naming a terminal and an ACP client` |
| `TestCheckNoQuestion` (P5) | `CheckNoQuestion` returning nil for a consent with no answer | `steps_test.go:269: CheckNoQuestion = <nil>, refused false, want refused true` |
| `TestCheckNoQuestion` (P5) | the stored session's method not compared, so another application's answered session counts | `steps_test.go:269: CheckNoQuestion = <nil>, refused false, want refused true` |
| `TestOfferedMethodByWord` (P5) | `OfferedMethod` giving the first offered method for any word it knows | `steps_test.go:301: OfferedMethod(grok, device) = "api_key" "value", true; want "device_code" "device"` |
| `TestAuthLoginJSONSteps` | the old `device_code` event emitted beside the step | `loginjson_test.go:106: the line {"event":"device_code","provider":"grok","url":"http://127.0.0.1:<port>/device","code":"ABCD-1234"} does not decode into the contract's types: json: cannot unmarshal JSON string into Go cli.jsonLine: unknown object member name "url"` |
| `TestAuthLoginJSONKiloKey` | `yields` left empty | `loginjson_test.go:142: kilo's steps [{SchemaVersion:1 Seq:1 Kind:show_code URL:http://127.0.0.1:<port>/kilo Code:KILO-1 ExpiresAt:2026-11-15 18:01:52. ... alse ConfirmID: Text: Default:<nil> Links:[] Fields:[] Yields: Error:<nil> Meta:map[]}], want its code at the issuer and a done that yields device_key` |
| `TestAuthLoginJSONAPIKey` | the `saved` event kept | `loginjson_test.go:165: the line {"event":"saved","provider":"claude","type":"api_key"} does not decode into the contract's types: json: cannot unmarshal JSON string into Go cli.jsonLine: unknown object member name "type"` |
| `TestAuthLoginJSONUsageError` | exit 1 | `loginjson_test.go:198: ["ollama"]: exit 1, stdout "{\"event\":\"step\",\"provider\":\"ollama\",\"step\":{\"schema_version\":1,\"seq\":1,\"kind\":\"fai ...  claude, grok, opencode-zen, opencode-go, huggingface, kilo, together\",\"retryable\":false}}}\n", stderr ""; want exit 2, one line, nothing on stderr` |
| `TestAuthLoginJSONUsageError` | a usage error's kind left as `ErrorKind` gives it, `other` | `loginjson_test.go:204: ["ollama"]: the line {Event:step Provider:ollama Method:<nil> Step:0x21e927734000}, want one failed step, seq 1, invalid_request, a message that begins "unknown provider \"ollama\": one of "` |
| `TestAuthLoginJSONRefusesAQuestion` | the check before the sign-in dropped, so the refusal is left to the driver's backstop (the SDK asks the consent before it opens a URL, so "after the `open_url` step" cannot be built; this is the refusal after the sign-in has started) | `loginjson_test.go:246: the sign-in started 1 times and did [], want it refused before it began` |
| `TestAuthLoginJSONNoHostBrowser` | the old `openLoginURL` branch restored: `OpenURL` set and `Steps` not | `loginjson_test.go:287: the URL was shown [] and opened ["https://auth.x.ai/oauth2/authorize?client_id=b1a00492-073a-47ea-816f-4c329264a828&code_challe ... email+offline_access+grok-cli%3Aaccess+api%3Aaccess&state=mrENfAxcvgLUVHJXdZwS7rAde_GdCPk7HSczxebzT9o"]; want it shown once as a step and never opened` |
| `TestAuthLoginJSONSteps`, `TestAuthLoginJSONMatchesVectors` | a step's `expires_at` written as `expires` | `loginjson_test.go:106: the line {"event":"step","provider":"grok","step":{"schema_version":1,"seq":1,"kind":"show_code","url":"http://127.0.0.1:<port> ... ot decode into the contract's types: json: cannot unmarshal JSON string into Go authcontract.Step: unknown object member name "expires" within "/step" / loginjson_test.go:379: the show_code step {"event":"step","provider":"grok","step":{"schema_version":1,"seq":1,"kind":"show_code","url":"http://127.0.0.1:<port>/device","code":"ABCD-1234","expires":"<time>","secret":false}} has members [expires] that step-show-code.json does not` |
| `TestAuthLoginJSONFailureKind` (P5) | the failed step's kind read with `llm.Kind` in place of `provider.ErrorKind` | `loginjson_test.go:319: the last step is failed, seq 3, error {Kind:auth Message:grok sign-in failed (denied) Retryable:false}; want failed, seq 3, kind denied, the message "grok sign-in failed (denied)"` |
| `TestAuthLoginJSON` (rewritten) | the log lines left on stderr (`e.logs.stderr` not cleared) | `auth_test.go:526: a settings warning: exit 0, stderr "", log ""` |
| `TestAuthLoginJSON` (rewritten) | a paste offered under `--json` on a terminal | `auth_test.go:545: --json true on a terminal: <nil>, paste offered [true], steps asked for [true]` |

On the tree before the phase (HEAD's sources with the new and changed test files added): `llm/provider` does not compile, `o.Steps undefined (type SignInOptions
has no field or method Steps)`, `undefined: ErrAsks`, `undefined: stepDriver`, `undefined: CheckNoQuestion`; `internal/cli`, `o.Steps undefined (type
provider.SignInOptions has no field or method Steps)` (`before.out`: exit 1, 1).

The plants of the PLAN's rows that differ from the PLAN's wording: `TestAuthLoginJSONRefusesAQuestion`'s "the refusal after the `open_url` step" cannot be built as
worded, because the SDK asks the consent before it opens a URL; the plant is the check before the sign-in dropped, so the refusal comes from the driver after the
sign-in has started, which the test sees as the wizard having been run. `TestAuthLoginJSONNoHostBrowser`'s "the old `openLoginURL` branch restored" is the old
branch's two hooks set and `Steps` not: with a driver set the SDK calls neither, so that is the only way the opener is reached.

**Gates (C6),** each exit status taken from the file its output was redirected to before anything read it. `make race`, `make check-test-isolation`, `make
preflight` and the lint runs went through `run-heavy`; the other two hosts through `run-on-host --race`, `sh -c`, which kept the Linux run off the WSL distros (the
probe's own lines say so). **The first cross-host run found a failure the Mac's package runs had not:** `internal/archtest`'s `TestImportRules/rule_4_module`,
`rule 4: github.com/maccavelli/gobble-cli/internal/cli imports github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth`: `loginissuer_test.go` imported the SDK's
`llmprovider/auth` for one constant. Both hosts failed the same way, so it was not the hosts; the test now states the constant (`loginGrokIssuer`), and every gate
and plant below ran after that edit, the first run's results discarded (I stopped the runs in flight and removed their files).

| Gate | Where | Start to end | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check FILES=` the 7 changed Go files | this Mac | 17:54:20 to 17:54:32 | 0 |
| `golangci-lint run --config .golangci.yml ./internal/cli/ ./llm/provider/`, six runs, a new `GOLANGCI_LINT_CACHE` each, under `run-heavy` | this Mac | 17:54:32 to 17:55:22 | 0, 0, 0, 0, 0, 0 |
| `make check-acpext-deps` | this Mac | 17:55:23 to 17:55:23 | 0 |
| `make race`, then `GOFLAGS=-count=1 make race`, under `run-heavy` | this Mac | 17:55:23 to 17:56:17, then 17:56:17 to 17:57:09 | 0, 0 |
| `make check-test-isolation`, under `run-heavy` | this Mac | 17:57:09 to 17:57:40 | 0 |
| `go test -count=1 ./...`, `CGO_ENABLED=1 go test -count=1 -race ./...`, `go -C acpext test -count=1 ./...`, one `run-on-host --need linux --race`, `sh -c` | the Linux machine, scratch copy | 17:54:20 to 17:58:29 | 0 (GOTEST-OK, RACE-OK, ACPEXT-OK) |
| the same three, `run-on-host --need windows --race`, `sh -c` | the Windows laptop, scratch copy | 17:54:20 to 17:57:04 | 0 (GOTEST-OK, RACE-OK, ACPEXT-OK) |
| `make preflight`, under `run-heavy` | this Mac | 18:01:44 to 18:09:17 | 0 |

The six lint runs: run 1 17:54:32 to 17:54:40 exit 0, run 2 17:54:40 to 17:54:48 exit 0, run 3 17:54:49 to 17:54:57 exit 0, run 4 17:54:57 to 17:55:05 exit 0, run 5 17:55:06 to 17:55:14 exit 0, run 6 17:55:14 to 17:55:22 exit 0; **six of six clean, no finding of any kind**, so there was nothing to fix. No identifier this change adds differs from another in its
package only by capitalization: the lint's `confusing-naming` ran six times, and a scan of every top-level name of the two packages, test files included
(`casecheck.py`, which does not read methods), finds one pair, `Selection` and `selection` in `llm/provider/ambient.go`, older than this change.

The changed Go files were last modified at the times below, before every gate in the table above ran (the first gate started at 17:54:20):

```text
17:28:21  internal/cli/auth.go
17:28:21  internal/cli/login.go
17:33:07  llm/provider/signin.go
17:36:30  internal/cli/auth_test.go
17:47:18  llm/provider/steps_test.go
17:48:56  internal/cli/loginjson_test.go
17:53:54  internal/cli/loginissuer_test.go
```

The Linux and Windows copies hold the working tree, not a commit.

**Deviations.**

1. **Ruled by the owner, "Accept the bytes as built": the SDK's types written by `jsonx` differ from the SDK's own bytes.** *Evidence,* each run on a step whose URL holds `&`, with the encoders
   compared: `encoding/json` v1, `jsonx.Marshal`, `jsonx.Marshal` with `OmitEmptyWithLegacySemantics(true)`, and that with `jsontext.EscapeForHTML(true)` too. (i)
   `jsonx.Marshal` is `encoding/json/v2`; the contract's types carry v1 tags, and v2's `omitempty` omits only a value that encodes as an empty JSON value, which `false`
   is not, so the two `omitempty` bools of the contract's types print `false` under v2: `Step.Secret`, so every printed step line carries `"secret":false`, and
   `Field.Required` (form fields, which gobble never prints; run on a field: `{"key":"k","label":"L","type":"text","required":false}` under `jsonx`, without `required`
   under v1). The pointers `Step.Default`, `Answer.Secret` and `Answer.Confirm` are omitted when nil under both. (ii) `jsonx` does not escape `&`, `<` and `>` in strings,
   where v1 does, and the contract's README says its vectors are in Go's encoding, which escapes them: the URL `https://a.example/x?a=1&b=<2>` prints as
   `a=1&b=<2>` under `jsonx` and as `a=1\u0026b=\u003c2\u003e` under v1, and every printed `open_url` URL holds an `&`. (iii) With `OmitEmptyWithLegacySemantics(true)` alone, (i) is
   gone and (ii) is not; with `jsontext.EscapeForHTML(true)` as well, the bytes equal v1's for both samples. The lines decode strictly into the contract's types and mean the
   same (evolution rule 1: an absent member is its zero, and a JSON string reads `&` either way); only the bytes differ. `TestAuthLoginJSONMatchesVectors` first failed on (i).
   *Ruling (2026-11-15, the owner's ruling, relayed by the coordinator): "Accept the bytes as built".* The `--json` lines stay as `jsonx` writes them: they carry the contract's types and
   meanings, which is what 0051-MADR D5 and the PLAN's row require, and their bytes differ from the SDK's own encoding in the two ways above, `"secret":false` on every step and `&`, `<`, `>`
   not HTML-escaped. No `jsonx` change, no mirror struct, no SDK change. The owner was shown, and did not choose: the SDK-side `omitzero` on `Step.Secret` and `Field.Required` in an
   `authcontract` v0.1.1 (resolution (c)), and a `jsonx` option with legacy omit-empty semantics and HTML escaping (resolution (a)); resolution (d), the mirror struct, was counted a workaround. The
   same holds for P6's ACP path, where steps are written to a client: a pointer is at P6 step 3. In the tree: the encoder is as `jsonx` writes it; the vector test tolerates exactly a `"secret":false`
   at a step's top level (`memberDiff`) and nothing else, a plant of any other `false` member failing it; the guide says to read each line as the contract's `method` or `step` object, to ignore a member it
   does not know and not to depend on a member being absent. (The first version of this entry built resolution (b) before it stopped, and said so afterwards; "Review of 2026-11-15", item 9: a deviation stops
   the item. The ruling is the same as (b).) *Resolutions, as put:* (a) `jsonx` gains one option wrapping `encoding/json`'s `OmitEmptyWithLegacySemantics(true)`, and `jsonSteps.emit` passes it with
   `jsontext.EscapeForHTML(true)`; cost: a change in `jsonx` (outside this phase's files), one `depguard` exclusion for `^jsonx/` in `.golangci.yml`, a test; it gives the SDK's bytes; (b) accept it as
   built (chosen): cost: `"secret":false`, and unescaped `&`, `<` and `>`, on the wire, harmless to a decoder of either; (c) `omitzero` on `Step.Secret` and `Field.Required` in the contract module itself,
   an SDK release: it removes (i) at the source for every consumer and not (ii); cost: an SDK change and release, gobble's pin moving; (d) write the steps through a mirror struct of gobble's with `omitzero` tags:
   a workaround, a second copy of the contract's types that drifts from it. **0051-MADR asserts no byte equality with the SDK's vectors:** the sentences checked are the context's list of the contract's parts
   ("a schema, golden vectors and additive-only rules", `:80`), D5 (`:505-515`: "One JSON line first ... then one per step ... the last `done` or `failed`", nothing of bytes), and Confirmation 4
   (`:780-781`: "steps decode with the contract's golden vectors' types"), which decoding holds; no amendment is made.
2. **Tests the PLAN did not name,** each because a row or a sentence has no test of its own or a stronger one was needed; none loosens or replaces a named test:
   `TestSignInStepsDevice`, `TestSignInStepsBrowserAsksNoPaste`, `TestSignInStepsRefuseWhatAsks`, `TestSignInStepsErrorsAreSignInErrors` (the carried item, in
   `llm/provider/steps_test.go`), `TestCheckNoQuestion` (step 2's refusal, which row 5 tests only through the command), `TestOfferedMethodByWord`, and
   `TestAuthLoginJSONFailureKind` (a failure's kind and message through the real wizard). Also new, not tests: `internal/cli/loginissuer_test.go`, an issuer in
   process, the subset of `llm/provider`'s `testIssuer` that these tests use, because that one is in a package's tests and cannot be imported (a shared test package
   would move `testIssuer` out of `login_test.go`, outside this phase); `signInAtIssuer` is `provider.SignIn` with the HTTP client pointed at it and the browser played.
3. **Names the PLAN does not give,** in `llm/provider`: `ErrAsks`, `OfferedMethod` and `CheckNoQuestion` (step 2 needs the refusal and the offered method's facts where the
   SDK is imported, and `internal/cli` may not import the SDK, 0004-MADR rule 4, but for `authcontract`). Within the steps.
4. **Readings, each in the code and held by a test:** the last step's `seq` is the last printed step's plus one; `retryable` is the SDK's for the same error, `provider.Retryable`: the API
   error's own `Retryable()` when the error holds one (the service's `x-should-retry` and a final reply honoured), otherwise true for `rate_limited`, `offline` and `unavailable`
   (signinflow's `Failure`, unexported; "Review of 2026-11-15", item 3); the method line comes before a refusal for a question (the client knows which method was refused), after
   the key is read for `--api-key -`, and never before a usage error, a missing key being one ("Review of 2026-11-15", item 1); a failed step's `message` is the error's text, which
   gobble prints without `--json` and which holds nothing of the service's (`SignInError`'s own text, or a usage error's).

**Docs step (C9).** Searched `README.md`, `AGENTS.md`, `docs/architecture.md`, `docs/README.md`, `docs/guides/*.md` and the doc comments of the changed files for `--json`,
`auth login`, `device_code`, `browser_url`, `saved`, `ShowDeviceCode`, `OpenURL`, `SignInOptions`, `loginEvent`, `wizard` and the sign-in's no-terminal message; each hit
was checked against the code and the tests. Every sentence added was checked against the tests' output, which `docs/guides/credentials.md`'s example lines are, minus the
issuer's address and a time. No file joins the scope: both are in P5's list or C9's (`docs/architecture.md`, `docs/README.md`).

* `docs/guides/credentials.md:362-412` (the `--json` section, the P4 record's open row `:338-376`) rewritten, fixed: the `method` and `step` lines, each step kind, `done` after the
  store, `failed`'s `error`, the retired lines, the kinds, the exit codes (0; 2 for a usage error and for a refused question; 128 plus the signal for a sign-in a signal ended; 1 otherwise), the failure kinds as the open set they are, `retryable`, the plan sign-in's
  refusal and that an answered consent lets `--json` sign in, a command line that does not parse (text on stderr, exit 2, no JSON line) *(2026-11-16: now one `failed` step when `--json` is read first; second part, deviation 2)*, and example lines as gobble prints them; the guide says nothing of
  `"secret":false` while deviation 1 is open.
* `docs/guides/credentials.md:200-213` (the ChatGPT plan bullet said `--json` stops at the question with the no-terminal message) fixed: that is the no-terminal case; `--json` is refused before it starts, exit 2, unless the stored session holds an answer.
* `docs/architecture.md:110` (the `llm/provider` row) fixed: `SignInOptions.Steps`, the driver, `ErrAsks`, `CheckNoQuestion`, `OfferedMethod`.
* `internal/cli/auth.go:53` (`--json`'s help text, "Print the login's events as JSON lines") fixed: "Print the method and the sign-in's steps as JSON lines on stdout, and nothing on stderr."
* `docs/guides/credentials.md:189-300` (the ChatGPT section) and `:86-132` (`auth status --json`) stay open, as the P4 record says: 0052-PLAN P4 and P6 own them.
* `README.md`, `AGENTS.md`, `docs/README.md`'s matrices and the other guides: nothing names `auth login --json`, the retired lines or the hooks; no change. This repository keeps no changelog.

`docs: searched README.md, AGENTS.md, docs/architecture.md, docs/README.md, docs/guides/*.md` (for `--json`, `auth login`, `device_code`, `browser_url`, `saved`, `ShowDeviceCode`, `OpenURL`, `SignInOptions`, `wizard`, `no terminal`).

**Acceptance.**

* The tests pass and each plant failed: the table above, and every gate. Met.
* `TestAuthLoginJSON` is rewritten for the steps, and the rewrite is named in this record: step 4 above. Met.
* The commit is the coordinator's, on the branch.

**Not verified.** Nothing was run against a live service (C5; the live sign-ins are P8's). The two other hosts ran scratch copies of the working tree, not a commit. A real
`auth.json` was never read: every test uses a scratch `GOBBLE_HOME`. A first ChatGPT plan sign-in past its consent, the browser, the paste and the token exchange are not run
here (0052-PLAN P4's `TestSignInChatGPTPlan`); `TestAuthLoginJSONRefusesAQuestion` runs the sign-in of a plan session that holds an answer only as far as its `open_url` step,
which shows the SDK asked no `confirm` first, and no further. The method line's `consents` carry gobble's wording as `Offered` gives it; no reader of the lines
(magic-cli-remote) was run against them. `gobble auth login --json` was not run against a built binary here; the tests call `Main`.

The identifier scan of the staged diff's added lines, for the local account name, the host names, home, temporary and dev-host scratch paths, run-heavy ticket shapes and mail
addresses, found nothing (`grep` exit 1) and found the one line of a copy with planted values (count 1). A script found no annotation of 2026-11-15 preceded by an odd number of
quotation marks in its paragraph in this PLAN (0 before this change, 0 after). Both ran, and `make check-records markdownlint` (exit 0), after every text of this record but this
paragraph, and ran again after it.

#### Review of 2026-11-15 (P5)

An independent review of the staged P5 found no blocker in the sign-in or storage behaviour, and twelve items, relayed by the coordinator. Each was verified first; this entry says what it
found, what changed, and the plant of each new or changed test, run on a scratch copy (`<TMP>`, `GOPROXY=off`, `HOME`, `XDG_*` and `GOBBLE_HOME` in the scratch; one `go test -count=1 -run
'^(Test...)$'` each; exit 1 each). It annotates the record above and rewrites only the sentences that were false, each named below; deviation 1 is corrected in place, as the coordinator said,
because it is uncommitted. The coordinator's ruling on how deviation 1 was handled: a deviation stops the item and is not built one way and reported afterwards. The owner then ruled, relayed by the coordinator, "Accept the bytes as built": the encoder stays as `jsonx` writes it, recorded in deviation 1 with the evidence of item 9 and a pointer at P6 step 3;
items 5 and 9 below stand as written.

1. **A usage error has a method line: true, and the PLAN decides it the other way.** `gobble auth login openai --api-key - --json` with empty stdin is a usage error
   (`internal/cli/credentials.go:229`), exit 2, and its `failed` step came after a `method` line because `js.method` ran before `readAPIKey`. Step 2 reads: "`--api-key -` writes `method` and a
   `done` step with `yields: "api_key"`; a usage error is one `failed` step of `invalid_request`." A missing key is a usage error, so it is one `failed` step. *Changed:* `runWith`
   reads the key, then writes the method line (`internal/cli/auth.go:122-129`); `TestAuthLoginJSONAPIKey` asserts one line and no method line for the empty key; the guide's exit-code
   paragraph names the missing key as a usage error with no method line; the record's step 2 and reading 4 are corrected. Plant: the method line before the key is read (table).
2. **"Exit 1 for any failed step but two" is false for a signal: true.** I ran it, by the path `Main` takes (`watchSignals`' context cancelled with the signal as its cause, `run`, then
   `signalExit`): the sign-in prints a `failed` step of kind `canceled`, `seq` 3, not retryable; the command's own code is 1 and the process's is 143 for SIGTERM (130 for SIGINT, by
   `exitForSignal`; only SIGTERM is run, SIGINT being the same path with another number). *Changed:* `TestAuthLoginJSONEndedBySignal` (`loginjson_test.go`), and the guide says 128 plus the signal's number
   (`docs/guides/credentials.md:391-405`). Plant: no failed step when the context ended (table).
3. **`retryable` is held by no test and the rule is not the SDK's: true.** The contract's `Failure` says only "why a sign-in failed" and the PLAN names no rule; the SDK's `signinflow.Failure` (v1.31.0,
   `internal/signinflow/signinflow.go:560-572`) documents it as "whether the same sign-in can succeed later": the API error's own `Retryable()` when the error holds one (the service's `x-should-retry`
   and a final reply honoured), the kind table (`rate_limited`, `offline`, `unavailable`) otherwise. *Changed:* `provider.Retryable(err)` (`llm/provider/signin.go`) does that on the error `mapError` kept
   in the chain, and `jsonSteps.fail` calls it (`internal/cli/auth.go:238`); `TestRetryable` (`llm/provider`: a 503, a 503 marked final, offline, an auth failure, a cancelled sign-in) and
   `TestAuthLoginJSONRetryable` (through the command). *(Corrected 2026-11-16; "Review of 2026-11-15 (P5), second part", item 1: the text here first said no sign-in ends with an API error, which was false.)* I first tried a 503: with the issuer answering `503`
   marked final (`x-should-retry: false`) to the listing (`/api/gateway/models`, `/v1/models`), Kilo's device sign-in and Grok's both ended `done`, exit 0, and a 503 at Grok's device endpoint ends `other`, not an
   API error; a 503 is not a refusal of the credential. What does end a sign-in with an API error is the listing that checks the new credential refusing it with HTTP 401, which the wizard wraps in its refusal
   (`wizard/configure.go:499`, `:706-725` of go-llmprovider-sdk `v1.31.0`): a failed step of kind `auth`, the error a `*provider.SignInError` holding the API error. `TestAuthLoginJSONRetryable` runs the real wizard
   against that, for Grok and Kilo (second part). The first version used a stand-in `signInWith` returning what `provider.List` returns for a 503. Plants of `TestRetryable` (table): the API error not read,
   so the final 503 is read by its kind; the kind table cut, so offline is not retryable. The two plants of the first version of `TestAuthLoginJSONRetryable` are removed from the table, the test being replaced.
4. **"`done` after the credential is stored" had no test: true.** `TestAuthLoginJSONDoneFollowsTheStore`: `auth.json.lock` is a directory, so `SetKey` and the session's save cannot take the lock; for Kilo's key
   and for Grok's session the last step is `failed`, exit 1, and no `done` was printed. Plants: `done` before `SetKey` (key); the SDK's `done` given to `Steps` (session) (table). The first fails the key
   case and passes the session's, the second the reverse, each for its own reason.
5. **`membersBeyond` forgave any `false` at any depth: true** (`"default":false` on every step passed). *Changed:* `memberDiff` forgives one thing, a `"secret":false` at the step's top level (the owner's ruling "Accept the bytes as built" keeps it), and a missing top-level `text`, and is two-way. From the contract's types (`requiredMembers` reads the tags): required are `schema_version`, `seq` and `kind` of
   a `Step` and `kind`, `message` and `retryable` of its `Failure`; every other member is optional by type, and the vector's other members (`url`, `code`, `expires_at`, `yields`, `error`,
   `error.*`) are the facts of the step that the sign-in states, so gobble must print them: the test fails a step that lacks any member of its kind's vector but `text`, which gobble does not
   print because the SDK's steps from a real sign-in set none. Plants: `"default":false` on every step; `code` not printed (table).
6. **The guide's closed list of thirteen kinds: true.** `provider.ErrorKind` gives the SDK's word for a `SignInError` (`signin.ErrorKindOf`: all eighteen of the contract's, `invalid_request` and
   `invalid_provider` included) and `llm.Kind`'s otherwise (fourteen, `workspace`, `context_overflow`, `empty_answer` and `incomplete` among them); a usage error is `invalid_request`. The guide
   now lists the eighteen as what gobble can print, says which a person's refusal is, and that a program keeps a word it does not know (`docs/guides/credentials.md:391-399`).
7. **`done`'s `yields` came from the method line: true.** The contract's `StepDone` says "ends a sign-in that succeeded; Yields says what it left"; the SDK's own `done` carries
   `r.method.Yields` (`signinflow.go:348`), so at v1.31.0 the two are equal and no offered method differs; the PLAN says only `yields: "api_key"` for `--api-key -`. *Changed:* the driver captures the
   SDK's `done` yields (`Credential.Yields`), `signInTo` carries it into `jsonSteps` when it states one (`internal/cli/login.go:80-82`), and the method's is the default. `TestSignInStepsDevice`
   asserts the real wizard's `done` yields for Grok and Kilo; `TestAuthLoginJSONDoneYieldsWhatTheSignInSays` runs a stand-in `signInWith` that returns `access_only` against a method whose fact is
   `oauth_session`. Plants: the yields not captured; `done` stating the method's (table).
8. **A Kong parse error under `--json` prints text on stderr and no JSON line, exit 2: true** (`gobble auth login --json` with no provider: stdout empty, stderr `Error: expected "<provider>"`; run). **Stopped:
   the PLAN does not decide it.** *(2026-11-16: ruled by the owner, relayed by the coordinator, "Emit the failed step from the parser's own context"; built in "Review of 2026-11-15 (P5), second part", deviation 2, which also changes the guide's and the help text's wording quoted at the end of this item.)* Step 2: "a usage error is one `failed` step of `invalid_request`" speaks of the command's own usage errors, which `Run` produces after Kong has bound `--json`;
   a parse error happens before any `Run`, and whether `--json` is known then is a decision. Options: (a) after a parse failure `Main` looks in the arguments for `auth login` and `--json` and prints
   the failed step; cost: grammar knowledge in `main.go` (outside P5's files) that Kong already has, with edge cases (global flags between the words, `--json=false`, abbreviations, `--` ); (b) read
   the partial context Kong returns with its error to see whether `auth login` was selected and `--json` set; cost: unverified against Kong's behaviour on the error path, and the same file; (c) leave it
   and say so, which is the tree now. *Changed, whatever is ruled:* the guide says a command line gobble cannot parse is a usage error as text on stderr, exit 2, with no JSON line, and the help text says
   "and nothing on stderr once the command line parses" (`internal/cli/auth.go:53`, `docs/guides/credentials.md:362-368`). Recommended: (a) only if the owner wants it; (c) costs nothing and is true.
9. **Deviation 1's evidence was incomplete: true; corrected in place** (deviation 1 above). Each was run on a step whose URL is `https://a.example/x?a=1&b=<2>` and a field, by `encoding/json` v1, `jsonx.Marshal`, `jsonx.Marshal`
   with `OmitEmptyWithLegacySemantics(true)`, and that with `jsontext.EscapeForHTML(true)`: `jsonx` writes `"secret":false` and `&b=<2>`; v1 writes neither (`&`, `<`, `>`); the legacy option alone removes
   the `false` and not the escaping; both options give v1's bytes; `Field.Required` prints `"required":false` under `jsonx`, absent under v1. The contract's two `omitempty` bools are `Step.Secret` and
   `Field.Required` (the others, `Answer.Secret`, `Answer.Confirm` and `Step.Default`, are pointers). Resolution (c), `omitzero` in the contract module, is added, and (a) now includes the escaping option. The guide says to read each line as the contract's `method` or `step` object, to ignore a member it does not know and not to depend on one being absent, and that the bytes are not the vectors'.
10. **The carried item's record understated the exceptions: true.** `SignIn` with `Steps` returns unwrapped, besides Steps' own error: `ctx.Err()` (`signin.go`, after the wizard returns), which
    `ErrorKind` words `canceled` or `timeout` through `llm.Kind`, the words of the contract, right under 0051-MADR D6 ("`signin.ErrorKindOf` for an error of the SDK's sign-ins, else `llm.Kind`'s"), and
    held through the command by `TestAuthLoginJSONEndedBySignal`; a Prompter's own error, such as `errNoTerminal`, `llm.Kind` `other`, which is D6's reading (no SDK error is in it) and which `--json`
    never reaches, the driver refusing a question first; and the two errors before the wizard starts (`provider %s takes no key and has no sign-in`, `has no %s sign-in`), `other`, which
    `internal/cli` cannot reach (`keyProvider` and `methodOf` refuse first). *Changed:* `SignIn`'s doc comment lists them (`llm/provider/signin.go:276-289`), and the comment of
    `TestSignInStepsErrorsAreSignInErrors` says exactly what it holds (the wizard's errors once it has started, `ErrAsks`, and Steps' own) and what it does not; the record's step 2 paragraph says "the
    carried item" holds the first and not the others.
11. **`CheckNoQuestion` is looser than the SDK in two edges, known and backed by the driver.** (i) The SDK carries a stored answer only when the previous session is a plan session with a client id
    (`wizard/auth.go:279`); `CheckNoQuestion` reads the stored session's kind and answer. A plan session that holds an answer and no client id passes the check, the SDK asks, and the driver's backstop
    refuses it with `ErrAsks`: `TestAuthLoginJSONBackstopMatchesTheCheck` runs both and holds that the lines, exit and stderr are byte for byte the same, the wizard having started once in the second (plant:
    the driver answering the confirm, table). I found no case where the output differs. (ii) The SDK asks every consent of the method and `CheckNoQuestion` only the required ones; no offered method has a consent that is not
    required, so there is no run of it, and `TestSignInStepsRefuseWhatAsks` holds that the driver refuses any step that asks. No code change.
12. **One line of `SignIn`'s doc comment ran past the wrap: true; wrapped** (`llm/provider/signin.go:276-289`, with item 10's text).

The new and changed tests' plants (each exit 1):

| Test (the item) | Planted break | Failure seen (`<TMP>` is the scratch copy) |
| :--- | :--- | :--- |
| `TestRetryable` (3) | `Retryable` not reading the API error, so the final 503 is read by its kind | `steps_test.go:358: Retryable = true, want false (the kind unavailable)` |
| `TestRetryable` (3) | the kind table cut to `rate_limited`, so offline is not retryable | `steps_test.go:358: Retryable = false, want true (the kind offline)` |
| `TestAuthLoginJSONAPIKey`, `TestAuthLoginJSONUsageError` (1) | the method line written before the key is read | `loginjson_test.go:186: no key given: exit 2, stdout "{\"event\":\"method\",\"provider\":\"openai\",\"method\":{\"schema_version\":1,\"id\":\"api_key\" ... \",\"message\":\"--api-key -: no key was given\",\"retryable\":false}}}\n", stderr ""; want exit 2 and one failed invalid_request step, no method line` |
| `TestAuthLoginJSONEndedBySignal` (2) | no failed step when the context ended | `loginjson_test.go:589: 3 lines, want a method line and 3 steps: [{Event:method Provider:grok Method:0x51f0588fc360 Step:<nil>} {Event:step Provider:grok Method:<nil> Step:0x51f0588e40f0} {Event:step Provider:grok Method:<nil> Step:0x51f0588e41e0}]` |
| `TestAuthLoginJSONDoneFollowsTheStore` (4, a key) | `done` written before `SetKey` | `loginjson_test.go:541: a done step was printed though the store failed: {"event":"method","provider":"kilo","method":{"schema_version":1,"id":"device_ ... in with Kilo (device code)","completion":"device","yields":"device_key","client_identity":"none","refreshable":false,"revocable":false,"slot":"kilo"}}` |
| `TestAuthLoginJSONDoneFollowsTheStore` (4, a session) | the SDK's `done` step given to `Steps`, which prints it before the wizard saves | `loginjson_test.go:541: a done step was printed though the store failed: {"event":"method","provider":"kilo","method":{"schema_version":1,"id":"device_ ... in with Kilo (device code)","completion":"device","yields":"device_key","client_identity":"none","refreshable":false,"revocable":false,"slot":"kilo"}}` |
| `TestAuthLoginJSONMatchesVectors` (5) | `"default":false` printed on every step | `loginjson_test.go:351: the show_code step {"event":"step","provider":"grok","step":{"schema_version":1,"default":false,"seq":1,"kind":"show_code","url":"http://127.0.0.1:<port>/device","code":"ABCD-1234","expires_at":"<time>","secret":false}} has members [default] that step-show-code.json does not, and lacks [] that it has` |
| `TestAuthLoginJSONMatchesVectors` (5) | a `show_code` step's `code` not printed | `loginjson_test.go:351: the show_code step {"event":"step","provider":"grok","step":{"schema_version":1,"seq":1,"kind":"show_code","url":"http://127.0.0.1:<port>/device","expires_at":"<time>","secret":false}} has members [] that step-show-code.json does not, and lacks [code] that it has` |
| `TestSignInStepsDevice` (7) | the SDK's `done` yields not carried in `Credential.Yields` | `steps_test.go:65: grok's device sign-in: {Key: Device:false Workspace: Yields:}, <nil>, saved map[grok:Login{Issuer:https://auth.x.ai ClientID:b1a00492-073a-47ea-816f-4c329264a828 AccountID: TokenURL:https://auth.x.ai/oauth2/token FedRAMP:false Expires:<time>}]` |
| `TestAuthLoginJSONDoneYieldsWhatTheSignInSays` (7) | `done` stating the method's yields, not the credential's | `loginjson_test.go:626: the last line {Event:step Provider:grok Method:<nil> Step:0x30643a4d4000}, want a done step that yields access_only` |
| `TestAuthLoginJSONBackstopMatchesTheCheck` (11) | the driver answering a `confirm` with no answer: no backstop | `loginjson_test.go:671: the backstop printed exit 1, stdout "{\"event\":\"method\",\"provider\":\"openai\",\"method\":{\"schema_version\":1,\"id\":\"ch ... rogram that reads its steps cannot answer; sign in at a terminal (gobble auth login), or from an ACP client\",\"retryable\":false}}}\n"; want the same` |

All 35 plants of the record, the earlier ones too, were run again on the final tree, each exit 1, and the unbroken runs of their tests exit 0 (`plants-final.out`, `clean.out`). On the tree before the phase the new test
files do not compile, as before.

**First lint of this round,** run before the last code change: four findings, each in a file of this review (`gocognit` on `TestAuthLoginJSONMatchesVectors`, `gocritic` singleCaseSwitch in `signInTo`,
`modernize` `Type.Fields`, `staticcheck` QF1001), six of six alike. Fixed in the files (the vector test split into `compareWithVectors` and `requireMembers`), no `nolint`, no config change; the gates
below are the final tree's.

**Docs step (C9).** Searched the same files and words as the record, and for `exit`, `signal`, `retryable`, `kind`, `failed`: the rows are `docs/guides/credentials.md:362-421` (the `--json` section: no
method line for a usage error, the exit codes with a signal, the open set of kinds, `retryable`, the parse error, `"secret":false` removed) fixed; `docs/guides/credentials.md:200-213` unchanged since the record;
`internal/cli/auth.go:53` (the help text) fixed; `docs/architecture.md:110` (the `llm/provider` row) fixed: `Retryable` and the carried `Yields`; `README.md`, `AGENTS.md`, `docs/README.md`: no change but its row's
wording stays true. `docs: searched README.md, AGENTS.md, docs/architecture.md, docs/README.md, docs/guides/*.md`.

**Gates after the last code change** (test files are code); the first gate started at 20:00:42, every Go file below was last modified before it *(2026-11-16: the tree of the first part; the second part's table is the final tree's)*:

| Gate | Where | Start to end | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check FILES=` the 7 changed Go files | this Mac | 20:00:42 to 20:00:51 | 0 |
| `golangci-lint run --config .golangci.yml ./internal/cli/ ./llm/provider/`, six runs, a new `GOLANGCI_LINT_CACHE` each, under `run-heavy` | this Mac | 20:00:51 to 20:01:40 | 0, 0, 0, 0, 0, 0 |
| `make check-acpext-deps` | this Mac | 20:01:40 to 20:01:40 | 0 |
| `make race`, then `GOFLAGS=-count=1 make race`, under `run-heavy` | this Mac | 20:01:40 to 20:02:40, then 20:02:40 to 20:03:39 | 0, 0 |
| `make check-test-isolation`, under `run-heavy` | this Mac | 20:03:39 to 20:04:14 | 0 |
| `go test -count=1 ./...`, `CGO_ENABLED=1 go test -count=1 -race ./...`, `go -C acpext test -count=1 ./...`, one `run-on-host --need linux --race`, `sh -c` | the Linux machine, scratch copy | 20:00:42 to 20:05:27 | 0 (GOTEST-OK, RACE-OK, ACPEXT-OK) |
| the same three, `run-on-host --need windows --race`, `sh -c` | the Windows laptop, scratch copy | 20:00:42 to 20:03:37 | 0 (GOTEST-OK, RACE-OK, ACPEXT-OK) |
| `make preflight`, under `run-heavy` | this Mac | 20:09:48 to 20:17:01 | 0 |

The six lint runs: run 1 20:00:51 to 20:00:59 exit 0, run 2 20:01:00 to 20:01:08 exit 0, run 3 20:01:08 to 20:01:15 exit 0, run 4 20:01:16 to 20:01:23 exit 0, run 5 20:01:24 to 20:01:31 exit 0, run 6 20:01:32 to 20:01:40 exit 0, each "0 issues."

```text
17:36:30  internal/cli/auth_test.go
19:44:10  llm/provider/signin.go
19:52:17  internal/cli/loginissuer_test.go
19:52:49  internal/cli/auth.go
19:53:09  llm/provider/steps_test.go
19:59:57  internal/cli/login.go
19:59:57  internal/cli/loginjson_test.go
```

**Not verified.** As the record, and: SIGINT was not run (the same path as SIGTERM with another number); the Kong parse error is built in the second part; the reviewer's Kilo-listing case with a 503 does not end
a sign-in, and a listing that refuses the credential with HTTP 401 does, which the second part's test runs through the real wizard.

The identifier scan of the staged diff's added lines (the same patterns as the record) found nothing (`grep` exit 1) and found the one line of a copy with planted values (count 1); a script found no
annotation of 2026-11-15 preceded by an odd number of quotation marks in its paragraph in this PLAN; both ran, with `make check-records markdownlint` (exit 0), after every text of this entry
but this paragraph, and again after it. The guide, the architecture row and the README row changed after the last gate, which read Go files and the markdown gates only.

#### Review of 2026-11-15 (P5), second part

A second review of the staged P5 asked for four things, and the owner ruled on item 8 of the first part. Relayed by the coordinator; built and proved on 2026-11-16 by a mainstream model at medium effort
(an earlier agent built the code and was stopped for a host reboot before it ran the plants, the gates or wrote this entry; every plant and gate below was run again by this one). It annotates the entry
above and rewrites only the sentences that were false: item 3, item 8's "Stopped", its table of plants and its "Not verified", each marked in place.

**The owner's ruling on item 8, relayed by the coordinator (2026-11-15): "Emit the failed step from the parser's own context".** On a parse error where the selected command is `auth login` and `--json`
was seen, gobble prints one `failed` step of kind `invalid_request` and nothing on stderr, exit 2, as 0051-MADR D5 and the PLAN's step 2 say ("a usage error is one `failed` step of `invalid_request`").
The options the owner was shown and did not choose: leave it and amend D5; inspect the raw arguments. What Kong gives (read, Kong `v1.16.1`, `go.mod`; `kong.go:323-356`): every error `Parse` returns is
a `*kong.ParseError` whose `Context` is set, and `Context.Selected()` and `Context.FlagValue` give the selected command and the traced value of `--json`. What gobble prints, run on a binary built from this
tree with `HOME`, `XDG_*` and `GOBBLE_HOME` in the scratch: `auth login --json` exit 2, one `failed` line (`provider` empty, `invalid_request`, `expected "<provider>"`), stderr empty; `--json auth login`,
`auth login --json=false`, `auth login grok extra --json` and `auth login --bogus --json grok` exit 2, stdout empty, Kong's text on stderr. The context does not give the provider the person named (the
positional's target is empty at a parse error, even for `grok --method`), so the line's `provider` is empty.

**Deviation 2 (a file outside P5's list): `internal/cli/main.go` joins this phase,** by this dated deviation, whose resolution is the ruling. `parse` calls `jsonLoginParseFailure`
(`internal/cli/main.go:147`, called at `:187`) before it prints Kong's text: for a `*kong.ParseError` whose context selects `gobble auth login` with `--json` true, it writes the failed step through
`jsonSteps.fail`, the code every other failed step is written by (same `schema_version`, `seq` 1, `kind`, `message`, `retryable`; one encoder), and returns exit 2; every other parse error is printed
as before. The residue is stated, not hidden: a `--json` that comes after the argument Kong rejects is never read, so that command line still gets the text on stderr
(`docs/guides/credentials.md:362-368` says so in one sentence, and the help text for `--json`, `internal/cli/auth.go:53`, says what is true). **0051-MADR D5 says "Nothing goes to stderr":** the residue is a
case where something does. *(2026-11-16, the coordinator's instruction: the MADR joins this phase's files by this note. The owner's ruling stated the residue in the option it chose, so no new ruling is
needed; 0051-MADR carries a dated note at D5, after that sentence (`docs/decisions/0051-MADR-…md:516-520`), a pointer at D5's line on usage errors (`:509`), and the ruling in its "Revisions of 2026-11-15" section
(`:1103-1111`); D5's text stays as written. Searched there and found nothing else that asserts it: `stderr`, `usage error`, `parse`, `output-only`; Confirmation 4 and the Neutral consequence hold as written.)*

The commands that have a `--json` flag, and what each does on a parse error, which this change leaves as it was (exit 2, nothing on stdout, Kong's text and the `run 'gobble --help' for usage` line on
stderr), held by `TestAuthLoginJSONParseFailureLeavesTheRest` (`internal/cli/loginjson_test.go:701`):

| Command | `--json` | On a parse error |
| :--- | :--- | :--- |
| `auth login` | the steps as JSON lines | with `--json` read first: one `failed` step, exit 2, nothing on stderr (this change); otherwise Kong's text |
| `config get`, `config path`, `version`, `update`, `auth status`, `auth check`, `mcp list`, `providers` | JSON of the command's own | Kong's text on stderr, exit 2, nothing on stdout |

**The items of the second pass.**

1. **`TestAuthLoginJSONRetryable` used the wrong error shape, and the first part's premise was false: true.** I tried a 503, which is not a refusal of the credential: a listing that answers 503 does not end a
   sign-in (both ended `done`, exit 0), and a 503 at Grok's device endpoint ends `other`, not an API error. What does end a sign-in with an API error is the listing that checks the new credential refusing it with
   HTTP 401: the wizard wraps the listing's error in its refusal (`wizard/configure.go:499`, `:706-725` of go-llmprovider-sdk `v1.31.0`), so the failed step is kind `auth` and the error is a `*provider.SignInError`
   that holds the API error. *Changed:* the test (`internal/cli/loginjson_test.go:479`) runs the real wizard against the issuer whose listings answer 401 (`loginIssuer.refusing`,
   `internal/cli/loginissuer_test.go:162`), for Grok's device sign-in with no `x-should-retry`, `true` and `false`, and Kilo's with `true`; the failed step is `auth`, and `retryable` is the reply's word: false, true,
   false, true. The first part's item 3 and its "Not verified" are corrected in place, and its two plants of the old test are removed from its table. `TestRetryable` (`llm/provider`) stays, for the kinds that
   hold no API error.
2. **The signal sentence was incomplete: true.** `internal/cli/term/signals_unix.go:13` watches SIGINT, SIGTERM and SIGHUP, so 130, 143 and 129 (`internal/cli/exit.go:19`); `signals_windows.go` only an
   interrupt, 130. The guide says both (`docs/guides/credentials.md:404-405`). Not changed in code; the Windows run of the suite is among the gates. SIGINT and SIGHUP are not run: `TestAuthLoginJSONEndedBySignal`
   runs SIGTERM.
3. **A failure that is not a usage error can come before the method line: true.** A stdin that cannot be read under `--api-key -` (`internal/cli/credentials.go:215-226`) is one `failed` step, `seq` 1,
   exit 1, and no `method` line, the key being read first. The PLAN's step 2 says "`jsonSteps` writes the `method` line, then each step" and "`--api-key -` writes `method` and a `done` step": the order of a
   sign-in that starts, which this holds; it says nothing of a failure before a method is chosen. The guide now says so (`docs/guides/credentials.md:406-409`). New `TestAuthLoginJSONStdinReadError`
   (`internal/cli/loginjson_test.go:733`, stdin a directory): one failed step, `seq` 1, exit 1, no method line. Plant: the method line written before the key is read, which fails it with
   `TestAuthLoginJSONAPIKey` and `TestAuthLoginJSONUsageError` (table).
4. **Is `TestRetryable` isolated: yes, and what the check shows is limited.** `make check-test-isolation` on the final tree (08:35:42 to 08:36:25, exit 0) planted `AGENTS.md`, `.agents/skills/anc/SKILL.md`, `.git/HEAD` and `.gitignore` above two temp roots (`TMPDIR`, `TMP` and `TEMP`,
and `GOTMPDIR`, 131 characters each), ran both modules' suites (`go test` exit 0, 0 failing in each), and printed `files written above the temp roots: 0`. That script looks at its own root and the temp roots, not
at `HOME`. I looked there: `llm/provider`'s test binary, compiled by `go test -c`, run for `TestRetryable` alone (five cases, PASS) with `HOME`, `USERPROFILE`, `XDG_*` and `TMPDIR` in an empty scratch and its working
directory in another, left 0 files (`iso2.sh`); through `go test` in the same setting 7 files appear, all the go tool's own telemetry (`Library/Application Support/go/telemetry/local/`: `compile`, `vet`, `asm`, `go`
and `link` counters, `upload.token`, `weekends`), none the test's (`iso.sh`). What `provider.List` would write, and where: nothing. It passes no `LastGood`, so nothing of a listing is kept; with an API key it
loads no stored login (`llm/provider/provider.go:99`), so `auth.json` is not read; the SDK's catalog cache is in memory, the test passing no `Options.Cache` (`llm/provider/cache.go:10-17`); the only code in
go-llmprovider-sdk `v1.31.0` that writes a file is its token store (`llmprovider/auth/tokenstore_file.go`) and `ownerperm`, neither of which `List` reaches. Every request goes through the test's transport
(`Options.HTTPClient`): the replies are its 503s, which the test asserts the kind of (`unavailable`), a reply a real service would not give to a made-up key.

**The plants of this part, each on a scratch copy** (exit 1 each; `<TMP>` is the scratch copy):

| Test | Planted break | Failure seen |
| :--- | :--- | :--- |
| `TestAuthLoginJSONParseFailure` | the parse failure not written as a step: Kong's text only | `loginjson_test.go:682: stdout "" is not whole lines` |
| `TestAuthLoginJSONParseFailureLeavesTheRest` | `--json` read as true whatever its value (`--json=false` gets the step) | `loginjson_test.go:723: ["auth" "login" "--json=false"]: exit 2, stdout "{\"event\":\"step\",\"provider\":\"\",\"step\":{\"schema_version\":1,\"seq\":1 ... cted \\\"<provider>\\\"\",\"retryable\":false}}}\n", stderr ""; want exit 2, nothing on stdout, Kong's text beginning "Error: expected \"<provider>\""` |
| `TestAuthLoginJSONParseFailureLeavesTheRest` | the step written for the parse error of any command with a `--json` | `loginjson_test.go:723: ["config" "get" "--json" "--bogus"]: exit 2, stdout "{\"event\":\"step\",\"provider\":\"\",\"step\":{\"schema_version\":1,\"seq ... e\":\"unknown flag --bogus\",\"retryable\":false}}}\n", stderr ""; want exit 2, nothing on stdout, Kong's text beginning "Error: unknown flag --bogus"` |
| `TestAuthLoginJSONStdinReadError`, `TestAuthLoginJSONAPIKey`, `TestAuthLoginJSONUsageError` | the method line written before the key is read | `loginjson_test.go:184: no key given: exit 2, stdout "{\"event\":\"method\",\"provider\":\"openai\",\"method\":{\"schema_version\":1,\"id\":\"api_key\" ... \",\"message\":\"--api-key -: no key was given\",\"retryable\":false}}}\n", stderr ""; want exit 2 and one failed invalid_request step, no method line / loginjson_test.go:743: exit 1, stdout "{\"event\":\"method\",\"provider\":\"openai\",\"method\":{\"schema_version\":1,\"id\":\"api_key\",\"kind\":\"ap ... uthLoginJSONStdinReadError164700739/002: is a directory\",\"retryable\":false}}}\n", stderr ""; want exit 1 and one failed step, seq 1, no method line` |
| `TestAuthLoginJSONRetryable` (real wizard, refusing listing) | `retryable` always false | `loginjson_test.go:506: the last step is failed with error {Kind:auth Message:kilo sign-in failed (auth) Retryable:false}, want a failed step of kind auth with retryable true` |
| `TestAuthLoginJSONRetryable` (real wizard, refusing listing) | the kind table alone: `auth` is in none | `loginjson_test.go:506: the last step is failed with error {Kind:auth Message:kilo sign-in failed (auth) Retryable:false}, want a failed step of kind auth with retryable true` |
| `TestAuthLoginJSONRetryable` (real wizard, refusing listing) | `provider.Retryable` not reading the API error's own `Retryable()` | `loginjson_test.go:506: the last step is failed with error {Kind:auth Message:kilo sign-in failed (auth) Retryable:false}, want a failed step of kind auth with retryable true` |

All 39 plants of the record, the earlier ones too, were run again on the final tree, each exit 1, and the unbroken runs of their tests (26 runs) exit 0 (`plants-final.out`, `clean.out`).
On the tree before the phase (HEAD's sources with the new and changed test files added) neither `llm/provider` nor `internal/cli` compiles: `steps_test.go:47:4: o.Steps undefined (type SignInOptions has no field or method Steps)`, `undefined: ErrAsks`, `undefined: stepDriver`,
`c.Yields undefined (type Credential has no field or method Yields)`; `internal/cli`, `auth_test.go:537:68: o.Steps undefined (type provider.SignInOptions has no field or method Steps)` (`before.out`: exit 1, 1).

**Docs step (C9).** Searched `README.md`, `AGENTS.md`, `docs/architecture.md`, `docs/README.md`, `docs/guides/*.md` and the doc comments of the changed files for `--json`, `auth login`, `SIGHUP`, `parse`, `stderr`,
`signal`, `failed`: `docs/guides/credentials.md:362-368` (the parse failure and its residue), `:398-409` (the kinds' exit codes, the signals, a failure before a method), `internal/cli/auth.go:53` (the help text)
and `docs/README.md`'s row for this PLAN (the plant count) fixed; `docs/architecture.md:132` (the `internal/cli` row) names nothing of `--json`'s parse failure, and `:110` (the `llm/provider` row) is as the
record above left it, no change; `README.md`, `AGENTS.md`: no change. `docs: searched README.md, AGENTS.md, docs/architecture.md, docs/README.md, docs/guides/*.md`.

**Gates after the last code change** (the first started at 08:32:20; every Go file below was last modified before it):

| Gate | Where | Start to end | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check FILES=` the 8 changed Go files | this Mac | 08:32:20 to 08:32:35 | 0 |
| `golangci-lint run --config .golangci.yml ./internal/cli/ ./llm/provider/`, six runs, a new `GOLANGCI_LINT_CACHE` each, under `run-heavy` | this Mac | 08:32:35 to 08:33:24 | 0, 0, 0, 0, 0, 0 |
| `make check-acpext-deps` | this Mac | 08:33:25 to 08:33:25 | 0 |
| `make race`, then `GOFLAGS=-count=1 make race`, under `run-heavy` | this Mac | 08:33:25 to 08:34:34, then 08:34:34 to 08:35:42 | 0, 0 |
| `make check-test-isolation`, under `run-heavy` | this Mac | 08:35:42 to 08:36:25 | 0 |
| `go test -count=1 ./...`, `CGO_ENABLED=1 go test -count=1 -race ./...`, `go -C acpext test -count=1 ./...`, one `run-on-host --need linux --race`, `sh -c` | the Linux machine, scratch copy | 08:32:21 to 08:37:16 | 0 (GOTEST-OK, RACE-OK, ACPEXT-OK) |
| the same three, `run-on-host --need windows --race`, `sh -c` | the Windows laptop, scratch copy | 08:32:21 to 08:35:39 | 0 (GOTEST-OK, RACE-OK, ACPEXT-OK) |
| `make preflight`, under `run-heavy` | this Mac | 08:37:27 to 08:44:30 | 0 |

The six lint runs: run 1 08:32:35 to 08:32:44 exit 0, run 2 08:32:44 to 08:32:52 exit 0, run 3 08:32:53 to 08:33:01 exit 0, run 4 08:33:01 to 08:33:09 exit 0, run 5 08:33:09 to 08:33:16 exit 0, run 6 08:33:17 to 08:33:24 exit 0, each "0 issues." No identifier this change adds differs from another in its package only by capitalization (`confusing-naming` ran six times).

```text
11-15 17:36:30  internal/cli/auth_test.go
11-15 19:44:10  llm/provider/signin.go
11-15 19:53:09  llm/provider/steps_test.go
11-15 19:59:57  internal/cli/login.go
11-16 07:20:22  internal/cli/auth.go
11-16 07:21:27  internal/cli/loginissuer_test.go
11-16 07:22:07  internal/cli/loginjson_test.go
11-16 07:24:51  internal/cli/main.go
```

**Not verified.** The residue (a `--json` after the argument Kong rejects) is stated and held as it behaves, not changed. SIGINT and SIGHUP exits are by the same path as SIGTERM's, which is run; the Windows run
covers the Windows signal and parse paths only as the suite does. The provider's name at a parse error is not known to the context. No live service, and no reader of the lines (magic-cli-remote) was run
against them.

The identifier scan of the staged diff's added lines (the same patterns as the record: the local account name, host names, home, temporary and dev-host scratch paths, queue ticket shapes and mail addresses) found
nothing (`grep` exit 1) and found the one line of a copy with planted values (count 1). A script found no annotation of 2026-11-15 or 2026-11-16 preceded by an odd number of quotation marks in its
paragraph in this PLAN (0 at `HEAD`, 0 now). Both ran, with `make check-records markdownlint` (exit 0, 08:45:29 to 08:45:31), after every text of this entry but this paragraph, and again after it. No Go file
changed after the first gate started (08:32:20). The MADR note of 2026-11-16 (deviation 2) and its record changed no Go file, so no gate that reads Go files was run again for them; `make check-records markdownlint`, the
identifier scan and the quotation check were run again after them. The guide was rewrapped at 08:33, after the first gate started and before `make preflight` (08:37:27); the README row and this record changed after the last gate,
which reads Go files and, in preflight, the markdown gates only.

### P6 record (2026-11-16)

Built by one mainstream model at medium effort and finished by a second, also at medium effort, each on the coordinator's brief; the first ran out of working
budget after the code, its tests and a first run of the plants, and the second verified what was there, built the owner's two rulings of 2026-11-16, ran the
plants, the security checks and the gates, and wrote this entry. The owner's approval of 2026-11-15, relayed by the coordinator, is "Run P5 then P6 without
stopping". Three rulings bind P6, each the owner's, relayed by the coordinator: "Keep the tests through a stand-in" (2026-11-15; the amendment at the top of
P6), "Check first, save only if accepted" and "Ask the client to choose" (both 2026-11-16; dated notes at P6 step 1, at the amendment's bullets, and at MADR D3
and D4). Times are the session machine's local time. Staged, not committed; **the coordinator commits on the branch `p3-acpext-v0.5.0`.** `main` and
`origin/main` stayed at `9e21c8f`; no tag, push or merge was made (the owner's ruling "Keep it on the branch until those phases land", C8).

**C1, the start state.** Branch `p3-acpext-v0.5.0`, `HEAD` `4410333`, level with origin; nothing staged; the working tree held the first agent's unstaged work
(twelve modified files, nine new). `go.mod` requires acpext `v0.5.0-rc.1` and go-llmprovider-sdk `v1.31.0`, no `replace`, no `go.work`.

**What changed (staged).**

1. **Step 1, `llm/provider/signinrun.go`.** `StartSignIn` (`:75`) over `signin.Start` (the package variable `startSignIn`, `:19`) with `signin.Options` as MADR D3
   lists it: `Identity` from `identityOf`, `AgentName` `gobble`, `HostID`, `ClientStore` when the store has one, `ConsentText` gobble's wording, `NoLoopback` from
   `SignInOptions.Remote`, and `Previous` the stored session of the kind the method signs in to. Every error is a `*SignInError` (`NewSignInError`, `:48`).
   `SignInRun` has `Steps`, `Answer`, `Cancel` and `Wait`; **`Wait` (`finish`) is the owner's "Check first, save only if accepted":** the engine's credential
   is tried against the provider's listing first, a session listed through itself and a key as the listing's key (`checkListing`, `listWith`); only when the listing does not
   refuse it is the session saved through the store in one write, or the key returned for the caller to store. A refusal fails the sign-in with kind `auth`, not retryable, and
   nothing stored has changed, byte for byte, with no revocation. **What the wizard does on a listing failure that is no refusal**, read at go-llmprovider-sdk `v1.31.0`,
   `wizard/configure.go`: `stepRefusal(listErr)` is `refusedCredential(err) || refusedWorkspace(err)` (`:680`), and `refusedCredential` is
   `errors.Is(err, llmprovider.ErrAuthFailure) && !errors.Is(err, llmprovider.ErrNotPermitted)` (`:667-669`); the loop `for keys, workspaces := 1, 0; stepRefusal(listErr); {` (`:499`)
   is not entered for any other error, so the code after it, `credential.save(ctx, d)` and `return res, cat, nil` (`:508-513`), runs: a 403, a 503, a timeout or an empty list leaves
   the sign-in a success and the credential saved. *(The first build did not do this: it refused a 403, failed on `List`'s own error, listed through a copy of the session and
   saved the original, and saved after a cancel; "Review and fixes" below.)* Kilo's device key is returned for `internal/cli` to store and is listed with, since the SDK sends the
   key Kilo's listing is given. `MethodCommand` (`:56`) and `SignedInMethod` (`:66`) are the
   two helpers `internal/cli` needs. **Step 1a:** `llm/provider/login.go` (`ClientRecord`, `ClientStore`, `sdkClientStore`, `tokenStore`), `SignInOptions.HostID` and `Remote`
   (`signin.go:84`, `:89`) and the wizard's `HostID`; `internal/auth/clients.go:111` `ClientLogins` is the store with the issued clients beside the logins.
2. **Step 2, `acpserver`.** `AuthOptions` (`auth.go:19`: `Providers`, `Has`, `Methods`, `Holds`, `Remote`, `Command`, `SignIn`, `Signed`); `authMeta` (`auth.go:118`) decodes
   `_meta.gobble` strictly (`RejectUnknownMembers`, -32602 for an unknown member or a wrong type); `authenticateProvider` (`:161`) is D4's four cases; **ruling 2,
   "Ask the client to choose", is `chooseMethod` (`:248`)**: with no method named, one driven method (for a remote client, one `Remote` accepts) starts with no
   question, and more than one are asked by the enum form of step 3 (`askMethod`, `authsignin.go:61`) listing exactly those methods in the offered order; a method named
   is never asked; a client with no `elicitation.form` is asked nothing (`auth.go:190`, the same answer as for a local one, MADR D4 step 3). `refuseRemote` (`:210`) is the
   `unsupported` refusal of a named method `Remote` does not accept. `acpserver/authsignin.go`: `Agent.signIn` (`:127`) keeps one sign-in per provider; `relay.serve`
   (`:207`) reads the run's steps and sends each as a request-scoped elicitation, one at a time, acknowledges the steps that only show something (`reply`, `:315`),
   answers `paste`, `confirm` and `form` from the accepts with `Secret` taken from the step, ends the open request before it answers, sends `elicitation/complete` for URL
   mode (`completeURLs`, `:293`), caps a bad paste at three, and turns a decline or cancel into `canceled`; `failure` (`:117`) is `auth_required` with the SDK's kind and
   `retryable`.
3. **Step 4, `internal/cli`.** `agentauth.go`: `authOptions` (`:25`), `holds` (`:45`), `startSignIn` (`:65`, the host id and the clients through `hostSignIn`, `login.go:34`),
   `keyRun` (`:94`, a device key stored through `auth.SetKey`, escaped, and returned to no one). `signInTo` fills `HostID` on the terminal path too ("Proceed, warn once").
4. **Tests, docs and the record.** Below.

**The stand-in** is as the amendment at the top of P6 wrote it and was built that way: `standIn` in `acpserver/authsignin_test.go` is Grok's real `browser_oauth` from
`provider.Offered("grok")` rewritten to `browser_loopback` with no `Paste`, supplied through `AuthOptions.Methods`, with `Remote` `provider.Remote` and `Command`
`provider.MethodCommand`, both production; its scripted run (`scriptRun`) has the shape `signinflow` gives a browser sign-in with no paste. The relay's decisions are
tested through it and through scripted runs; the real `x/signin` engine runs against in-process issuers in `llm/provider` (`signinrun_test.go`), in `acpserver`
(`authreal_test.go`, through `provider.StartSignIn`) and in `internal/cli` (`agentauth_test.go`).

**Tests (C2): each seen failing on a scratch copy of the tree** (`<TMP>`, no `.git`; `HOME`, `USERPROFILE`, `XDG_*` and `GOBBLE_HOME` in the scratch; `GOPROXY=off`;
`GOFLAGS=-mod=mod`; the module and build caches the real ones; one `go test -count=1 -timeout 240s -run '^(...)$' <package>` each, with `-count=3` or `5` for the two plants
that need the repetition; four shards at a time, then the plants whose failures were port probes were not trusted and none depends on one; the unbroken runs of the same tests
on a scratch copy of the final tree, `clean.out`, are 14 runs, exit 0 each). Every plant's log holds a `--- FAIL`. Fifty plants were run in the first build; 49 failed as intended and one,
`form-never-cancelled`, did not (corrected below: the review's test fails it). The plants of the first agent's run that failed to compile (`acp-host-id-error-fails`, `kilo-key-unquoted`) were rewritten as behavioural
breaks that compile (the unused import kept by `_ = slog.String` and `_ = catalog.QuoteValue`); the unbroken tree changed after two plants' anchors (`scope-counter`,
`nonboolean-truthy`), which failed to compile once `json` left two files, and they were rewritten and re-run.

| Test | Planted break | Failure seen (exit 1 each; `<TMP>` is the scratch copy) |
| :--- | :--- | :--- |
| `TestSignInWithoutHostID` (P6) | the ACP sign-in failing over the host-id error (`acp-host-id-error-fails`) | `agentauth_test.go:144: an ACP sign-in with no host id: no host id: host-id: open <TMP>/TestSignInWithoutHostID1330353681/002/state-file/host-id: not a direct` |
| `TestCLIFillsHostID` (P6) | the ACP path not filling the host id (`acp-host-id-not-set`) | `agentauth_test.go:91: the ACP sign-in's options [{Options:{APIKey: CredentialSource: Model: ClientName:gobble ClientVersion:0.0.0-dev Logger:<nil> HTTPClient:<nil> ModelM ...` |
| `TestClientStoreAdapter` (P6) | the client-store adapter chosen when the store has none (`adapter-inverted`) | `signinrun_test.go:159: the client kept {ClientID: HostID:}, want the issued one and the host id` (3 tests failed) |
| `TestAuthenticateNeverLogsPaste` (P6) | the answer built without Secret (`answer-secret-nil`) | `authsignin_test.go:742: answers [Answer{SchemaVersion:1 Seq:2 Secret:<nil> Paste:[redacted]}], want one for the paste step's seq 2, secret` (2 tests failed) |
| `TestAuthenticateSharesAuthJSON` (P6) | auth.json's lock released before the read-modify-write, with a pause in it (`auth-json-unlocked`) | `authreal_test.go:385: the sign-in's session is lost` (3 tests failed) |
| `TestAuthenticateClientCancels` (P6) | the client's $/cancel_request not reaching the sign-in (`client-cancel-not-honoured`) | `authsignin_test.go:1292: timed out waiting for the pending elicitation to be cancelled` |
| `TestStartSignInReusesClient` (P6) | ClientStore not passed (`client-store-nil`) | `signinrun_test.go:159: the client kept {ClientID: HostID:}, want the issued one and the host id` (2 tests failed) |
| `TestAuthenticateConfirm` (P6) | a declined consent answered as allow false instead of ending the sign-in as canceled (`confirm-decline-as-false`) | `authsignin_test.go:875: err = <nil>, want a JSON-RPC error` |
| `TestStartSignInPassesConsentText` (P6) | ConsentText not passed to signin.Options (`consent-text-omitted`) | `signinrun_test.go:217: ConsentText map[], want gobble's wording` (2 tests failed) |
| `TestAuthenticateFreshKeepsOldOnFailure` (P6) | a declined form treated as an accept with no answer (`decline-ignored`) | `authreal_test.go:249: err = <nil>, want a JSON-RPC error` (3 tests failed) |
| `TestAuthenticateOnlyRemoteMethods` (P6) | the enum not filtered to what Remote accepts (`enum-not-remote-filtered`) | `authsignin_test.go:1049: {"code":-32000,"message":"Authentication required","data":{"command":"gobble auth login grok","kind":"canceled","provider":"grok","reason":"grok  ...` |
| `TestAuthenticateRelayEndsItsFormWithoutTheSDK` (the review's) | the open form not cancelled by the relay when the poll ends (`form-never-cancelled`; first built as `TestAuthenticateDeviceCodeForm`'s plant, which exited 0) | `authsignin_test.go:1677: timed out waiting for the open form to be cancelled by the relay` (3 of 3 runs) |
| `TestAuthenticateDeviceCodeForm` (P6) | the elicitation request detached from the sign-in's cancellation and never ended by the relay (`form-never-ended`) | `authreal_test.go:265: timed out waiting for the open form to be cancelled` (12 tests failed) |
| `TestAuthenticateFresh` (P6) | _meta.gobble.fresh ignored (`fresh-ignored`) | `authreal_test.go:240: err = <nil>, want a JSON-RPC error` (2 tests failed) |
| `TestAuthenticateGolden` (P6) | the golden's request id changed (`golden-requestid`) | `authsignin_test.go:1445: acptest: record 3 differs from the golden testdata/authenticate-elicitation.golden` |
| `TestAgentHolds` (P6) | a key credential never holding a key method (`holds-key-false`) | `agentauth_test.go:219: a key held is a key's, not a session's` |
| `TestHostIDReachesTheEngines` (P6) | HostID not passed to the wizard or the engine (`host-id-not-copied`) | `signinrun_test.go:346: wizard options [{Registry:<ptr> Providers:[] Existing:Result{Provider:openai} AllowEnv:false Discover:true DiscoverLimit:0s NeedFallbacks:false Pro ...` |
| `TestStartSignInNamesGobble` (P6) | Identity not set (`identity-zero`) | `signinrun_test.go:360: recorded options [{HTTPClient:<ptr> ClientID: Issuer: Identity:{Name: Version:} AgentName:gobble Previous:OAuthSession(nil) HostID: ClientStore:<ni ...` |
| `TestAuthenticateKeyNeverLeaks` (P6) | the relay logging the finished sign-in's result, which holds a device key (`key-logged`) | `authsignin_test.go:1374: the agent's log holds the key:` |
| `TestAgentSignInStoresKiloKey` (P6) | the device key stored unescaped (`kilo-key-unquoted`) | `agentauth_test.go:194: kilo's entry {Type:api_key Key:!kilo$token-0123456789 Expires:0 Access: Refresh: Issuer: ClientID: AccountID: TokenURL: FedRAMP:false Resource: Sco ...` |
| `TestAuthenticateRealFailureKinds and the kind tests` (P6) | every failure's kind fixed to auth (`kind-fixed-auth`) | `authreal_test.go:185: error = code -32000 {Attempts:0 Code: Command:gobble auth login grok Consent: Error: HTTPStatus:0 Kind:auth Limit: Provider:grok ProviderMessage: Re ...` (7 tests failed) |
| `TestAuthenticateRealFailureKinds` (P6) | a refused listing not failing the sign-in (`listing-refusal-ignored`) | `authreal_test.go:285: err = <nil>, want a JSON-RPC error` (2 tests failed) |
| `TestAuthenticateWithCancelPending` (P6) | the agent's lock held while the client is awaited (`lock-held-across-wait`) | `authsignin_test.go:1268: session/new while an elicitation waits: {"code":-32603,"message":"Internal error","data":{"error":"context deadline exceeded"}}` |
| `TestAuthenticatePresent` (P6) | a present credential answering {} whatever method is named (`method-ignored`) | `authreal_test.go:423: err = <nil>, want a JSON-RPC error` (2 tests failed) |
| `TestAuthMethodsMeta` (P6) | the methods member under another key (`methods-meta-wrong-key`) | `authsignin_test.go:1417: gemini: methods` (2 tests failed) |
| `TestAuthenticateDeviceCodeURL` (P6) | elicitation/complete never sent after a URL-mode step (`no-elicitation-complete`) | `authsignin_test.go:672: timed out waiting for elicitation/complete` (3 tests failed) |
| `TestAuthenticateNoElicitation` (P6) | a client with no elicitation.form signed in anyway (`no-elicitation-signs-in`) | `authsignin_test.go:886: error = code -32000 {Attempts:0 Code: Command:gobble auth login grok Consent: Error: HTTPStatus:0 Kind:unsupported Limit: Provider:grok ProviderMe ...` (2 tests failed) |
| `TestStartSignInRemoteSetsNoLoopback` (P6) | NoLoopback false for a remote sign-in (`no-loopback-false`) | `signinrun_test.go:257: openai chatgpt_plan remote true: recorded [{HTTPClient:<nil> ClientID: Issuer: Identity:{Name: Version:} AgentName:gobble Previous:OAuthSession(nil ...` (3 tests failed) |
| `TestAuthenticateOneAtATime` (P6) | a second sign-in for a provider allowed while one runs (`no-one-at-a-time`) | `authsignin_test.go:517: authenticate grok map[method:device_code]: no answer within 20s` |
| `TestAuthenticateFreshRefusesNonBoolean` (P6) | _meta.gobble decoded loosely, any value of fresh or remote truthy (`nonboolean-truthy`) | `authsignin_test.go:951: fresh false: {"code":-32000,"message":"Authentication required","data":{"command":"gobble auth login grok","kind":"canceled","provider":"grok","re ...` (3 tests failed) |
| `TestAuthenticateBrowserPasteForm` (P6) | an acknowledged open_url answered to the engine (`open-url-seq-answered`) | `authsignin_test.go:745: the engine refused answers [Answer{SchemaVersion:1 Seq:1 Secret:<nil>}]: an open_url's seq was answered` |
| `TestAuthenticateLoopbackHasNoPaste` (P6) | a paste form sent after every acknowledged open_url (`paste-after-every-open-url`) | `authreal_test.go:149: 3 elicitations, want the open_url's and one paste form` (9 tests failed) |
| `TestAuthenticateBadPasteAsksAgain` (P6) | the bad paste echoed in the re-ask's message (`paste-echoed-in-reask`) | `authreal_test.go:209: paste form 2's message holds the paste: "That did not complete the sign-in (http://127.0.0.1:<port>/callback?code=c1&state=BADSTATE-SECRET-5). That  ...` (2 tests failed) |
| `TestAuthenticateBrowserPasteForm` (P6) | the paste step not relayed (`paste-form-skipped`) | `authsignin_test.go:517: authenticate grok map[method:browser_oauth remote:true]: no answer within 20s` (7 tests failed) |
| `TestAuthenticateNeverLogsPaste` (P6) | the pasted callback logged at debug (`paste-logged`) | `authreal_test.go:213: the agent's log holds the paste:` (2 tests failed) |
| `TestStartSignInPassesPrevious` (P6) | Previous never set (`previous-never`) | `signinrun_test.go:185: the second sign-in's steps [{1 1 confirm   0001-01-01 00:00:00 +0000 UTC  false chatgpt_plan.background_use gobble signs in to your ChatGPT plan un ...` |
| `TestAuthenticateRemoteRefusesLoopbackOnly` (P6) | the remote filter on a named method dropped (`remote-filter-dropped`) | `authsignin_test.go:517: authenticate grok map[method:browser_oauth remote:true]: no answer within 20s` |
| `TestAuthenticateRemoteDefault` (P6) | a remote client with no elicitation.form asked or signed in (`remote-no-form-signs-in`) | `authsignin_test.go:1132: a remote client with no form: error code -32000 {Attempts:0 Code: Command:gobble auth login grok Consent: Error: HTTPStatus:0 Kind:unsupported Li ...` |
| `TestAuthenticateOnlyRemoteMethods` (P6) | remote not passed on to the sign-in (`remote-not-passed`) | `authreal_test.go:156: calls [{grok browser_oauth false}], want browser_oauth, remote` (3 tests failed) |
| `TestAuthenticateOnlyRemoteMethods` (P6) | a remote client with several remote-capable methods given the first, not asked (`remote-takes-first`) | `authsignin_test.go:1035: grok at the pinned facts: the choice {String:<nil> Number:<nil> Integer:<nil> Boolean:<nil> Array:<nil> Other:<nil>}, want an enum of exactly the ...` |
| `TestAuthenticateRefusedListingStoresNothing` (P6) | Wait saving the session before the listing check (`save-before-listing-acp`) | `authreal_test.go:428: auth.json changed:` |
| `TestStartSignInChecksListingBeforeSaving` (P6) | Wait saving the session before the listing check (`save-before-listing-provider`) | `signinrun_test.go:591: a refused sign-in stored Login{Issuer:https://auth.x.ai ClientID:b1a00492-073a-47ea-816f-4c329264a828 AccountID: TokenURL:https://auth.x.ai/oauth2/ ...` |
| `TestAuthenticateRequestScope` (P6) | every elicitation scoped to a request id other than the authenticate request's (`scope-counter`) | `authsignin_test.go:649: elicitation 0: requestId 424242 (<nil>), want the authenticate request's 2` (2 tests failed) |
| `TestStartSignInStoresOnlyInWait` (P6) | Start deleting the stored login (`start-deletes-stored`) | `signinrun_test.go:485: the stored login is gone` (2 tests failed) |
| `TestSignInWithoutHostID` (P6) | the terminal sign-in failing over the host-id error (`terminal-host-id-error-fails`) | `agentauth_test.go:120: exit 1, stderr "Error: no host id: host-id: open <TMP>/TestSignInWithoutHostID1287939981/002/state-file/host-id: not a directory; the ` |
| `TestCLIFillsHostID` (P6) | the terminal path not filling the host id (`terminal-host-id-not-set`) | `agentauth_test.go:77: the terminal sign-in's options [{Options:{APIKey: CredentialSource: Model: ClientName:gobble ClientVersion:0.0.0-dev Logger:<nil> HTTPClient:<nil> M ...` |
| `TestAuthenticateRefusedListingStoresNothing` (P6) | a listing that fails otherwise (503) failing the sign-in (`unavailable-listing-fails-acp`) | `authreal_test.go:447: a 503 listing failed the sign-in: {"code":-32000,"message":"Authentication required","data":{"command":"gobble auth login grok","kind":"unavailable" ...` |
| `TestStartSignInChecksListingBeforeSaving` (P6) | a listing that fails otherwise (503) failing the sign-in (`unavailable-listing-fails-provider`) | `signinrun_test.go:609: a 503 listing failed the sign-in: grok sign-in failed (unavailable)` |
| `TestAuthenticateRealGrokDevice and the form-only tests` (P6) | URL mode sent to a client that offers only elicitation.form (`url-mode-to-form-only`) | `authreal_test.go:120: {"code":-32000,"message":"Authentication required","data":{"command":"gobble auth login grok","kind":"unsupported","provider":"grok","reason":"the c ...` (27 tests failed) |
| `TestStartSignInErrorsAreSignInErrors` (P6) | Wait's error returned as the SDK's, not a *SignInError (`wait-error-unwrapped`) | `signinrun_test.go:436: a denied sign-in: *auth.signInError (llmprovider: authentication failed: oauth: Grok device authorization denied) is not a *SignInError` |

**`form-never-cancelled`, and how it is held.** The relay's `stop` (`authsignin.go`, in `serve`) cancels the open elicitation's request and waits for its goroutine before the relay
answers; the plant removes both. The first build found that the plant exited 0 when run against `TestAuthenticateDeviceCodeForm` and the other tests, because in production the ACP SDK
cancels a child request when its handler returns, so the two cannot be told apart *there*. That is true of production and of every test through the SDK's own connection; it is not
true of the relay's own guarantee, that every request it opened has ended when `serve` returns. The review showed a test can hold that guarantee with no production seam: the ACP
SDK's public receiving middleware (`acp.WithReceivingMiddleware`) runs `authenticate` under `context.WithoutCancel(ctx)`, so the SDK's cancel of the handler's children no longer reaches
the relay's request, and a form-only client that leaves the device code's form open sees it cancelled only if the relay cancels it. `TestAuthenticateRelayEndsItsFormWithoutTheSDK` is that
test; unbroken it passed 3 of 3, with `stop`'s cancel and wait removed it failed 3 of 3. Deviation 10 records the coordinator's decision on it. The earlier reading, that a cancel-order
assertion fails on the unbroken tree and a goroutine count passes with the plant, stands for tests that run through the SDK's own cancel.

**Security checks (the first brief's), each held by a named test that a plant fails.**

* *A secret is in no log line, error, error data or step beyond what the contract's step carries:* `TestAuthenticateNeverLogsPaste` and `TestAuthenticateRealBadPasteCap` (a pasted
  code: never in the log, a frame but the client's answer, an error, a re-ask; every paste step says `secret`; the answer is built with `Secret` true), plants `paste-logged`,
  `paste-echoed-in-reask`, `answer-secret-nil`; `TestAuthenticateKeyNeverLeaks` (a device key the run returns: in no log line and no frame), plant `key-logged`;
  `TestAgentSignInStoresKiloKey` (the key stored escaped, the result holding none), plant `kilo-key-unquoted`. A session's tokens never reach `acpserver`: `SignInResult` carries a
  key or nothing, so no test can assert their absence from a type that cannot hold them.
* *`fresh` leaves the stored credential untouched until the save, which is now after the listing:* `TestAuthenticateFreshKeepsOldOnFailure` (denied, declined, cancelled: `auth.json`
  byte-identical, no revocation) with plants `decline-ignored`, `fresh-ignored`; `TestAuthenticateRefusedListingStoresNothing` and `TestStartSignInChecksListingBeforeSaving` (a 401
  listing: kind `auth`, not retryable, the file as it was, no revocation, for a device and a browser-by-paste sign-in, with and without `fresh`, with a credential stored and none),
  plants `save-before-listing-acp`, `save-before-listing-provider`, `unavailable-listing-fails-acp`, `unavailable-listing-fails-provider`.
* *A cancelled or abandoned elicitation ends the sign-in and leaves no goroutine, listener or half-saved session:* `TestAuthenticateClientCancels`, `TestAuthenticateAbandonedLeavesNothing`
  (the loopback port freed, nothing stored), `TestAuthenticateDeviceCodeForm` (`noRelayGoroutines`), plants `client-cancel-not-honoured`, `form-never-ended`, `lock-held-across-wait`.
* *`_meta.gobble` decoded strictly:* `TestAuthenticateFreshRefusesNonBoolean` and `TestAuthenticateFresh`, plant `nonboolean-truthy`.
* *A remote client drives only methods `provider.Remote` accepts:* `TestAuthenticateRemoteRefusesLoopbackOnly` (`remote-filter-dropped`), `TestAuthenticateOnlyRemoteMethods` (`enum-not-remote-filtered`,
  `remote-takes-first`, `remote-not-passed`), `TestAuthenticateRemoteDefault` (`remote-no-form-signs-in`), `TestAuthenticateRealGrokBrowserPasteRemote` (the remote
  browser sign-in by paste completes, started remote) and `TestStartSignInRemoteSetsNoLoopback` (what gobble passes: no-loopback for every offered method; `no-loopback-false` and
  `no-loopback-not-set-for-device`; the owner's ruling "Assert what gobble passes; use no real ports" removed the two probes of real ports).
* *Concurrent `authenticate` calls do not corrupt the store:* `TestAuthenticateOneAtATime` (one sign-in per provider, by any method, plants `no-one-at-a-time` and
  `guard-per-method`) and `TestAuthenticateSharesAuthJSON`, which the review made deterministic: from the listing, which runs between the sign-in's token and its save, another writer
  takes `auth.json`'s lock, reads the file, holds it until `authenticate` has answered or 3 s pass, and writes from that earlier read; a save that skips the lock completes while the lock is held, however late, and the
  writer's write loses the session every time (plants `auth-json-unlocked` and `auth-json-unlocked-delayed`, ten of ten runs fail each; "Second review and last fixes" below). (The first build's version had a writer retaking the lock after 1 ms against a waiter that backs off up to 2 s, and
  its plant failed two of three runs.)
* *The teardown race the first agent noted.* Verdict: **in the tests and the ACP SDK, not in gobble.** Evidence: with the two waits for the client to have seen the cancel removed from
  `TestAuthenticateClientCancels` on a scratch copy, 21 of 60 runs failed with `acptest: a connection or its handlers did not stop within 5s of its pipe closing`. The cause, found by
  the review and checked against the code: when the input of a connection ends cleanly, the ACP SDK lets the requests already received run, under contexts that stay live, for up to
  `notificationQueueDrainTimeout`, 5 s (acp-go-sdk2 `connection.go:23`, `drainInput` at `:1768-1790`), and the test harness's cleanup waits `closeTimeout`, also 5 s
  (`acpclient/acptest/pair.go:16`, `:60`), for the connection to stop; the test client's hanging handler (`hangReply`) ends only when its request's context is cancelled, so a
  test that closes the pipes first leaves it blocked for the whole of the SDK's drain, which equals the harness's bound, and the cleanup fails or passes by a hair. (The first build's
  reading, that the SDK sends `$/cancel_request` after the handler's response, is the same fact seen from the other side.) Every test that uses `hangReply` waits for `endedCount`
  before it ends. Nothing in gobble's relay is involved; whether the SDK should cancel in-flight handlers when a pipe closes is for the SDK's owner.

**Deviations (dated 2026-11-16; each reason checked).**

1. **`planIssuer` did not exist.** The reason is true: `git grep planIssuer` at `HEAD` finds it only in this PLAN and in 0052-MADR's table ("kept"), in no `.go` file. It was built as
   `planIssuer` in `llm/provider/signinrun_test.go:53`, a `testIssuer` answering as OpenAI's issuer with the scope the plan sign-in is granted. C5's sentence that it exists was
   false at `HEAD`.
2. **`TestAuthenticateRemoteBindsNoPort` is held as `TestStartSignInRemoteBindsNoPort` in `llm/provider`.** True: `internal/archtest/check_test.go` ("rule 4 acpserver signin")
   fails an `acpserver` import of `x/signin`, and the plan's row needs the plan issuer's real engine. *(Corrected 2026-11-16, the owner's ruling "Assert what gobble passes; use no real ports": neither the row's holding of ports 1455 and 1457 nor `TestAuthenticateRealGrokBrowserPasteRemote`'s dial of the redirect's port is a test any more; the row is `TestStartSignInRemoteSetsNoLoopback` for every offered method and the `AuthOptions.SignIn` seam's `remote` true in `acpserver`.)*
3. **`TestSignInWithoutHostID` makes the state directory a regular file** where the row said a read-only parent and a deny-write ACE. True and no weaker: a read-only parent does not
   stop a process running as root, and the ACE is a Windows-only construct; a regular file fails `MkdirAll` the same way on every platform and the test still proves the sign-in
   goes on, warns once, names the host-id error and no id (plants `terminal-host-id-error-fails`, `acp-host-id-error-fails`).
4. **New names.** `AuthOptions.Holds` (needed: step 2's "the stored credential is what the named method yields" is read from the CLI's credential kind, which `acpserver` cannot read),
   `AuthOptions.Command` and `provider.MethodCommand` (needed: the refusal's command word is `llm/provider`'s `methodIDs`), `provider.SignedInMethod` (needed by `internal/cli`'s `holds`;
   first named `StoredMethod`, renamed because `revive`'s `confusing-naming` rejects an exported name that differs from `storedMethod` only by capitalization), `provider.NewSignInError`
   (needed: `SignInError.err` is unexported and `acpserver` builds `canceled`). Exported and unused by production: `SignInRun.Method()`, which MADR D3 names and the `acpserver`
   interface lists; the relay does not call it.
5. **Tests are in new files, not `auth_test.go`.** No rule requires either; `auth_test.go`'s three tests are unchanged and the new ones are 1,500 lines in `authsignin_test.go`,
   `authreal_test.go`, `authissuer_test.go`.
6. **`TestAuthenticateRemoteDefault` restated** though the 2026-11-15 annotation did not name it: true, its row said "the device sign-in" and cannot pass at `v1.31.0`; restated again for
   "Ask the client to choose".
7. **The first agent never ran a lint.** `make lint` found 23 findings on its code: `forbidigo` (`json.Unmarshal` in two places, now `jsonx`), `sloglint` (key-value logging; attrs, and
   `WarnContext` where a context is in scope), `contextcheck` (the relay's `start` now takes the context: `relay()` is `serve(ctx)`), `revive`'s `confusing-naming`
   (`authenticate` is `authenticateProvider`, `wait` is `finish`, `StoredMethod` is `SignedInMethod`), `unparam` (`chooseMethod`'s `caps`), `unused` (`openaiOrigin`), `modernize`
   (embedded literals, `maps.Copy`, a `strings.Builder`), `noctx` (the probes use `Dialer.DialContext`, the held port `ListenConfig.Listen`), `errcheck` (`errors.AsType`'s value used). Its
   `//nolint` comments in new test code were removed; none is left in what this phase adds.
8. **Defects in the first agent's tests, found by the plants and the gates and fixed:** `TestAuthenticateSharesAuthJSON` (above); `TestAuthenticateAbandonedLeavesNothing` read the
   redirect host the client's script set after `waitRequests(1)` returned, which the client's handler reaches first (a data race, an empty host under load; now a channel);
   `siEnv.authenticate` had no bound, so a planted break hung the package to its timeout and gave no `--- FAIL` (now 20 s); `TestStartSignInStoresOnlyInWait`'s snapshot dereferenced a
   deleted login (a panic; now a failed assertion). The three tests of a remote browser sign-in with no method named (`{"remote": true}`) now name `browser_oauth`, because the
   owner's ruling asks such a client to choose.
9. **The probes of a loopback port were removed.** *(Corrected 2026-11-16.)* The first build kept `TestAuthenticateRealGrokBrowserPasteRemote`'s dial of the redirect's port and
   `TestStartSignInRemoteBindsNoPort`'s holding of ports 1455 and 1457, and said they were as the PLAN's row asked and only sensitive to another process on the machine. The review found
   the worse defect: the dial passed three of three with `NoLoopback` forced false, because the redirect's port is one nothing listens on whether or not the engine bound it, so the
   probe could not fail on its defect and could fail with none. By the owner's ruling, relayed by the coordinator, "Assert what gobble passes; use no real ports", both are gone; the
   mechanism is held at the seam where gobble starts the sign-in (deviation 2's correction).
10. **The relay's explicit cancel of an open form (`stop`).** *(Corrected 2026-11-16; the coordinator's decision, not the owner's.)* The first build said `stop` "cannot be told from the SDK's
    cancel of a handler's children" and put the question to the owner, with options (a) keep it, (b) delete it, (c) a seam in `AuthOptions`. That was true of every test through the SDK's own
    connection and false of the relay's own guarantee: in production the SDK's cancel also ends the request, so `stop` is belt-and-braces there, but the relay's contract is that every request
    it opened has ended when `serve` returns, and `TestAuthenticateRelayEndsItsFormWithoutTheSDK` holds that with no seam, by running `authenticate` under a context the SDK does not cancel
    (the SDK's public receiving middleware). The coordinator, on the reviewer's evidence and the owner's standing rule that a check is unused until seen to fail, decided: keep `stop`,
    adopt the test, make the plant `form-never-cancelled` fail against it (it does, 3 of 3). Option (a) of the first build, with a test; no `AuthOptions` field.

**Gates (C6), the first build's; the review's are in "Review and fixes" below.** Each with its start and end time and its exit status taken before any filter; no Go file changed after the first started.

| Gate | Start | End | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check` on the 17 Go files (gofmt, golint, govulncheck) | 12:28:22 | 12:28:44 | 0 |
| cold `golangci-lint` run 1 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 12:28:44 | 12:29:12 | 0 |
| cold `golangci-lint` run 2 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 12:29:13 | 12:29:39 | 0 |
| cold `golangci-lint` run 3 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 12:29:40 | 12:30:07 | 0 |
| cold `golangci-lint` run 4 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 12:30:07 | 12:30:35 | 0 |
| cold `golangci-lint` run 5 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 12:30:36 | 12:31:02 | 0 |
| cold `golangci-lint` run 6 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 12:31:03 | 12:31:29 | 0 |
| `make race` (both modules, `-race`), through `run-heavy` | 12:31:30 | 12:32:46 | 0 |
| `make check-test-isolation` | 12:32:46 | 12:33:34 | 0 |
| `make check-acpext-deps` | 12:33:34 | 12:33:35 | 0 |
| `go test -race -count=1 ./...` and acpext's, `run-on-host --race --need linux`: the Linux machine, linux/amd64, not a WSL distro | 12:33:35 | 12:38:45 | 0 |
| `go test -race -count=1 ./...` and acpext's, `run-on-host --race --need windows` under `sh -c`: the Windows laptop, windows/amd64, native | 12:38:45 | 12:42:44 | 0 |
| `make preflight`, through `run-heavy` (gofmt, tidy, pre-add-check, vet, staticcheck, golangci-lint, govulncheck, check-acpext-deps, fix-check, archtest, apidiff, verify-build-metadata, check-records, check-icon, markdownlint) | 12:42:44 | 12:49:49 | 0 |

**Docs (C9).** `docs/guides/credentials.md`: "Signing in from an ACP client" (the `_meta.gobble` members, the answer order with both rulings, the steps' elicitations, the error kinds and
`retryable`, one sign-in at a time, cancel, the golden) and the ChatGPT passage (the host id, `clients.json`, the warning); `docs/architecture.md`'s `acpserver`, `internal/auth` and
`llm/provider` rows; `acpserver/doc.go`. Rows returned to the PLAN's list at its end:

* `docs/guides/credentials.md:371` ("Signing in from an ACP client", new) — fixed: the `_meta.gobble` members, the answer order with both rulings, the steps' elicitations, the error kinds and `retryable`, one sign-in at a time, cancel and the golden.
* `docs/guides/credentials.md:350-369` (the editor and magic-cli-remote section's opening) — fixed: the methods each agent method lists, and that a form client is signed in by `authenticate`; no key or token crosses ACP but what a person types or pastes.
* `docs/guides/credentials.md:338-376` (the P4 record's row, "`--json`'s events, and ACP `authenticate`") — fixed: P6 made `authenticate` sign in by elicitation.
* `docs/guides/credentials.md:200-215` (the ChatGPT passage) — fixed for P6: the question as a form for an ACP client, the host id, `clients.json`, and the one warning when the state directory cannot be used; open for 0052-PLAN P4 step 3: the plan's consent through the phone and `TestConsentWordingOverACP`. *(2026-11-16, later: the open part closed by 0052-PLAN P4: the form's message is gobble's wording, held by `TestConsentWordingOverACP`; see 0052-PLAN's "P4 record (2026-11-16)".)*
* `docs/architecture.md:19-23`, `:111` (`llm/provider`), `:128` (`acpserver`), `:144` (`internal/auth`) — fixed: `authenticate` by elicitation, `StartSignIn`, `ClientLogins` and `clients.json`.
* `README.md:45-47` — fixed: a client that offers `elicitation.form` is signed in by `authenticate`.
* `acpserver/doc.go` — fixed: the relay and where its engine lives.
* `docs/architecture.md:194` (the acpext requirement) — open: P8a makes it `v0.5.0`.
* `AGENTS.md`, `docs/README.md`'s other rows and the other guides — searched for `authenticate`, `elicit`, `sign in`, `acp --login`, `remote`; nothing else this phase makes false.
* Still open for 0052-PLAN P4: the phone's consent and `TestConsentWordingOverACP`, which hold `ConsentText` over ACP; this phase built the site (step 1) and `TestStartSignInPassesConsentText` holds that the engine is given gobble's wording. *(2026-11-16, later: closed by 0052-PLAN P4, which built `TestConsentWordingOverACP` (`acpserver/authconsent_test.go`); see its "P4 record (2026-11-16)".)*

**Not verified.** No live service: every sign-in ran against an issuer in process, so Grok's, Kilo's and OpenAI's real endpoints, a real client's rendering of the forms (Magic, an
editor) and the phone flow of magic-cli-remote were not run. The ChatGPT plan's listing check (the Codex backend) is exercised against the in-process plan issuer by `TestStartSignInChecksPlanListing` (a 401 refuses, a 403 saves); the real backend's answers were not seen. The Windows run
covers Windows' path and permission behaviour only as the suite does.

**Files (staged).** Modified: `README.md`, `acpserver/agent.go`, `acpserver/auth.go`, `acpserver/doc.go`, `docs/README.md`, `docs/architecture.md`,
`docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`, `docs/decisions/0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md`,
`docs/guides/credentials.md`, `internal/auth/clients.go`, `internal/cli/agent.go`, `internal/cli/login.go`, `llm/provider/login.go`, `llm/provider/login_test.go`,
`llm/provider/signin.go`. New: `acpserver/authsignin.go`, `acpserver/authsignin_test.go`, `acpserver/authissuer_test.go`, `acpserver/authreal_test.go`,
`acpserver/testdata/authenticate-elicitation.golden`, `internal/cli/agentauth.go`, `internal/cli/agentauth_test.go`, `llm/provider/signinrun.go`, `llm/provider/signinrun_test.go`.

The identifier scan of the staged diff's added lines (the local account name, host names and aliases, home, temporary and dev-host scratch paths, queue ticket shapes, mail addresses) found nothing (`grep` exit 1). The modification times of the staged files: every Go file and the golden at or before 12:21:57, the first gate having started at 12:28:22; `README.md`, `docs/architecture.md`, this PLAN and `docs/README.md` after the gates (12:50:24 to 12:51:27), which read no Go file after it; `make check-records markdownlint` ran (exit 0, 12:51:27 to 12:51:29) after every text of this entry but this paragraph, and again after it.

#### Review and fixes (2026-11-16)

**Provenance.** An independent reviewer, a strongest-tier model at high effort, read the staged P6 on the coordinator's brief and found it **must not be committed as it was staged**: five
blockers and a list of should-fix findings, each with a reproduction as a test. A mainstream agent at medium effort, on a second brief, adopted the reproductions as regression tests,
fixed the findings, built the owner's three rulings of the same day, re-ran every plant, and ran the gates again. Three rulings of the owner, relayed by the coordinator, and one decision
of the coordinator's, bind this part; each is named below by its exact label. Times are the session machine's local time. Staged, not committed.

**Each reproduction run against the staged production code** (the working tree's new tests, with the five staged production files and `list.go` as at `HEAD` on a scratch copy; logs
`repro-staged-*.log`): the regression tests for B1, B2, B3, the plan listing, Kilo's key, a `null` member, the guard order, a remote `api_key` and a cancel after the save all fail there
(for example `signinrun_test.go:691: stored Login{...}, want the rotated pair`, `:609: a 403 listing failed the sign-in`, `:762: a sign-in cancelled during its listing succeeded`,
`authsignin_test.go:988: fresh null: error code -32000 ...`, `:1507: a second request while the first is asked`, `:1552: a key held: remote {"code":-32000 ...`), and pass on the fixed tree.

| Finding | Fix | Test | Plant (failure seen) |
| :--- | :--- | :--- | :--- |
| **B1**, a spent refresh token saved | `llm/provider/signinrun.go:171` lists through the engine's own session (`checkListing`, `list.go:85` `listWith`), `:178` waits for the session's token as the wizard's `save` does, and `:187` saves that session | `TestStartSignInSavesTheRenewedSession` (rotated, renewal 503, renewal refused) | `b1-save-original-session` |
| **B2**, the refusals are not the wizard's | `signinrun.go:198` `refusedCredential`, applied to `List`'s error and to the listing's, `:209` | `TestStartSignInChecksListingBeforeSaving` and `TestAuthenticateRefusedListingStoresNothing` (the table below) | `b2-refuse-403`, `b2-acp-403-refused`, `b2-fail-on-list-error`, `b2-fail-on-any-listing-error`, `b2-refusal-ignored` |
| **B3**, a cancelled sign-in saves | `signinrun.go:182` `ctx.Err()` before the save; `internal/cli/agentauth.go:104` before `SetKey`; `acpserver/authsignin.go:274` and `:300`, the relay's cancel branches | `TestStartSignInCancelledDuringListingSavesNothing`, `TestAuthenticateCancelledDuringListingStoresNothing` (with and without `fresh`, stored and none), `TestKeyRunStoresNothingOnceCancelled`, `TestAuthenticateCancelAfterSaveTellsTheAgent` | `b3-no-ctx-check-before-save`, `b3-no-ctx-check-at-all`, `b3-acp-no-ctx-check`, `b3-keyrun-no-ctx-check`, `b3-cancel-branch-never-signed` |
| **B4**, the port probe cannot fail on its defect | both probes removed (ruling 3); `TestStartSignInRemoteSetsNoLoopback` over every offered method; the `AuthOptions.SignIn` seam's `remote` and `TestAgentRemoteReachesTheEngine` | those tests | `no-loopback-false`, `no-loopback-not-set-for-device`, `remote-not-passed`, `remote-not-passed-seam` |
| **B5**, statements the code did not make true | the `Wait` and `checkListing` comments, MADR D3's note, P6 step 1's note, `docs/architecture.md`'s `llm/provider` row, `docs/guides/credentials.md` (the `fresh` row, the cancel sentence, the `auth` kind), this record | `make check-records markdownlint` | none |
| `null` members | `acpserver/auth.go:128` | `TestAuthenticateFreshRefusesNonBoolean` (`fresh` and `remote` null) | `null-fresh-accepted` |
| Kilo's key not checked | `signinrun.go:209`: the skip only without a key | `TestStartSignInChecksKiloDeviceKey` (401 refuses, no key comes back; 403 saves) | `kilo-skip-restored` |
| the plan's listing untested | none needed | `TestStartSignInChecksPlanListing` (401 refuses, 403 saves) | `plan-listing-skipped` |
| `store.Load`'s error not a `*SignInError` | `signinrun.go:92` wraps it; the doc and the architecture row were already true of it | `TestStartSignInErrorsAreSignInErrors` (an unreadable store) | `load-error-unwrapped` |
| the form asked before the guard | `acpserver/auth.go:211` `reserve` before `chooseMethod`; `authsignin.go:130`, `:150`; released on every path | `TestAuthenticateGuardComesBeforeTheMethodForm` (the second request refused before any form; a declined form frees the provider) | `guard-after-form` |
| the guard is the agent process's | the guide says so | none | none |
| `TestAuthenticateOneAtATime` blind to a per-method guard | a second Grok request naming another method | `TestAuthenticateOneAtATime` | `guard-per-method`, `no-one-at-a-time` |
| `TestAuthenticateRemoteRefusesLoopbackOnly` lacks the phone's shape | `{method, fresh: true, remote: true}` added | that test | `remote-filter-skipped-under-fresh`, `remote-filter-dropped` |
| `stop` | kept; the test adopted | `TestAuthenticateRelayEndsItsFormWithoutTheSDK` | `form-never-cancelled` (fails 3 of 3) |
| timing: shared `auth.json` | a writer takes the lock from the listing, reads, holds it until `authenticate` answers or 3 s pass, writes from the read | `TestAuthenticateSharesAuthJSON` | `auth-json-unlocked` (the save's lock skipped: 10 of 10 runs fail) |
| timing: `noRelayGoroutines` | polls up to 5 s for the frame to go | the tests that call it | `form-never-ended` |
| timing: `TestAuthenticateLoopbackHasNoPaste` | asserts the step has no `accepts`, an answer for it is refused, and the elicitations at the sign-in's end are one | that test | `paste-after-every-open-url` |
| timing: `TestAuthenticateAbandonedLeavesNothing` | asserts the run has ended, canceled, and its steps closed, and, from the stacks, that the SDK's callback server is running while the sign-in does and gone after the abandon; no port is dialled ("Second review and last fixes" below) | that test | `abandon-not-ended` |
| the PLAN's amendment quoted a paraphrase | corrected in place | none | none |

**The rulings and the decision.**

* **Ruling 1, "The same answer as a local client"** (the owner, relayed by the coordinator). A remote client that names `api_key` gets `{}` when a key is held, otherwise the same
  `auth_required` a local client gets ("… is not a sign-in gobble drives from an ACP client"); step 1's remote refusal is limited to methods gobble drives (`acpserver/auth.go:192`),
  with a dated note at MADR D4 step 1. `TestAuthenticateRemoteAPIKeyIsAsLocal` (held and not held, remote against local, the two answers equal); plant `remote-api-key-refused-as-before`.
* **Ruling 2, "Accept it and say so in D4"** (the owner, relayed by the coordinator). "Nothing stored" means credentials; a dated note at D4 step 3; the guide says where a reader looks
  (`clients.json`). `TestStartSignInPlanFailureKeepsCredentialsAndMayKeepClient`: a plan sign-in refused by the listing, and one cancelled during it, leave the login store untouched and the
  issued client record in the client store. The test holds the stores' own state; the byte-level proof is `TestStartSignInPlanFailureLeavesAuthJSON`, built in the second review (below).
* **Ruling 3, "Assert what gobble passes; use no real ports"** (the owner, relayed by the coordinator). Both probes are gone; `TestStartSignInRemoteSetsNoLoopback` runs for every method
  `provider.Offered` gives openai, grok, kilo and claude, remote and local, each with a planted break; the row is restated by a dated note at MADR D3; deviations 2 and 9 are corrected above.
* **The coordinator's decision on `stop`** (not the owner's): keep `stop()`, adopt the test, make the plant fail against it, amend deviation 10. In production the ACP SDK's cancel also ends the
  request; the test holds the relay's own guarantee that every request it opened has ended when `serve` returns.

**The refusal table, as built** (`TestStartSignInChecksListingBeforeSaving`, `TestAuthenticateRefusedListingStoresNothing`, and for the plan and Kilo their own tests): a listing status **401
refuses** (kind `auth`, not retryable, the store as it was); **403, 503, 400, 404 and 429 save**; a renewal the issuer answers with a 503 saves the session as signed in (`List` returns that
error, which is no refusal); a renewal refused with a 400 `invalid_grant` refuses (the listing carries the authentication failure). **The workspace refusal cannot arise on this path:**
`ErrWorkspaceRequired` and `ErrWorkspaceNotFound` are Anthropic's alone (`llmprovider/api_error.go:603-618`, `providers/claude/claude.go:167` at go-llmprovider-sdk `v1.31.0`), and
`provider.Offered("claude")` offers no session sign-in that `authenticate` drives, so the wizard's prompt for a workspace has nowhere to be asked and `refusedWorkspace` is not applied.

**A cancel that arrives after the save completed.** The credential is stored, so the relay's cancel branch returns success (`authsignin.go:300`), which makes `signIn` call `Signed` and the
agent drop its stale shared session; the client, which has cancelled, has no answer to act on. D4's line is "After a sign-in gobble drops the run's shared session for that provider and moves its
credential generation", and step 3's "a sign-in that fails, is declined or is cancelled changes nothing stored" describes a cancel *before* the save. The reviewer's fix, call `Signed` when the run
reports success, is this. `TestAuthenticateCancelAfterSaveTellsTheAgent`; plant `b3-cancel-branch-never-signed`.

**The corrected statements.** MADR D3's note (order and "does the same as the wizard"); P6 step 1's note; this record's account of step 1 (the in-memory login, "no listing can refuse" Kilo's
key); `form-never-cancelled` and its table row; the security checks for the probes and `TestAuthenticateSharesAuthJSON`; the teardown race's cause (the ACP SDK's 5 s drain,
`connection.go:23`, `:1768-1790`, against the harness's 5 s, `acptest/pair.go:16`, `:60`); deviations 2, 9 and 10; "Not verified"; `docs/architecture.md`'s `llm/provider` row;
`docs/guides/credentials.md`'s `fresh` and `remote` rows, its answer order, its one-sign-in and cancel paragraph and the `auth` kind; `acpserver/doc.go`; `docs/README.md`'s row.

**New tests.** `TestStartSignInSavesTheRenewedSession`, `TestStartSignInCancelledDuringListingSavesNothing`, `TestStartSignInChecksPlanListing`, `TestStartSignInChecksKiloDeviceKey`,
`TestStartSignInPlanFailureKeepsCredentialsAndMayKeepClient` (`llm/provider`); `TestAuthenticateCancelledDuringListingStoresNothing`, `TestAuthenticateGuardComesBeforeTheMethodForm`,
`TestAuthenticateRemoteAPIKeyIsAsLocal`, `TestAuthenticateCancelAfterSaveTellsTheAgent`, `TestAuthenticateRelayEndsItsFormWithoutTheSDK` (`acpserver`); `TestKeyRunStoresNothingOnceCancelled`,
`TestAgentRemoteReachesTheEngine` (`internal/cli`). **Tests changed, and why:** `TestStartSignInChecksListingBeforeSaving` and `TestAuthenticateRefusedListingStoresNothing` (the one status
became the table); `TestStartSignInRemoteSetsNoLoopback` (every offered method); `TestStartSignInErrorsAreSignInErrors` (the store's error); `TestAuthenticateFreshRefusesNonBoolean`
(`null`); `TestAuthenticateRemoteRefusesLoopbackOnly` and `TestAuthenticateOneAtATime` (the added shapes); `TestAuthenticateSharesAuthJSON`, `noRelayGoroutines`,
`TestAuthenticateLoopbackHasNoPaste`, `TestAuthenticateAbandonedLeavesNothing` (timing); `TestAuthenticateRealGrokBrowserPasteRemote` (the probe removed). **Removed:** `TestStartSignInRemoteBindsNoPort`.
No assertion was loosened, and no test skipped. One production refactor for a test's sake: `keyRun` embeds the `acpserver.SignInRun` interface instead of `*provider.SignInRun`, so the
key-storing path can be tested with a stand-in of the shape `provider.SignInRun.Wait` returns (`signinrun.go`, `finish`); no new exported name.

**Plants of this round, run on scratch copies of the final tree** (`HOME`, `XDG_*`, `GOBBLE_HOME` in the scratch, `GOPROXY=off`, two shards at a time, no HTTP proxy variable for `llm/provider`).
The 45 plants of the previous round whose breaks still exist (`wait-error-unwrapped` re-anchored) and 22 new ones were run: **67 run, 66 seen failing with a `--- FAIL`, none a build failure, and one
that does not fail, `b2-renewal-refusal-ignored`** (below); the unbroken runs of the same tests, `clean.out`, are 33 runs, exit 0 each. Five plants of the previous round were replaced because the
code they broke no longer exists (`save-before-listing-provider`, `save-before-listing-acp`, `unavailable-listing-fails-provider`, `unavailable-listing-fails-acp` and `listing-refusal-ignored`); the
`b2-` and `b3-` plants above break the new code in their places. `auth-json-unlocked` was rewritten: the first build's break (the lock released just after it is taken) cannot fail the deterministic test,
because the save still waits for the writer's lock; the new break skips the lock (ten of ten fail), which is what the brief's test is for. Where the old plants' failure lines in the table above name an
`authreal_test.go` line, the line is as of the first build.

| Test | Planted break | Failure seen (exit 1 each) |
| :--- | :--- | :--- |
| `TestStartSignInSavesTheRenewedSession` | the listing run through a copy of the session, so the original, spent, session is saved (B1) (`b1-save-original-session`) | `signinrun_test.go:691: stored Login{Issuer:https://auth.x.ai ClientID:b1a00492-073a-47ea-816f-4c329264a828 AccountID: TokenURL:https://auth.x.ai/oauth ...` (1 `--- FAIL` lines) |
| `TestStartSignInChecksListingBeforeSaving` | a 403 (not permitted) refused as an authentication failure (B2) (`b2-refuse-403`) | `signinrun_test.go:609: a 403 listing failed the sign-in: grok sign-in failed (auth)` (1 `--- FAIL` lines) |
| `TestAuthenticateRefusedListingStoresNothing` | a 403 refused, through the relay and the real engine (B2) (`b2-acp-403-refused`) | `authreal_test.go:519: a 403 listing failed the sign-in: {"code":-32000,"message":"Authentication required","data":{"command":"gobble auth login grok", ...` (1 `--- FAIL` lines) |
| `TestStartSignInChecksListingBeforeSaving, TestStartSignInSavesTheRenewedSession` | any error of List failing the sign-in (B2) (`b2-fail-on-list-error`) | `signinrun_test.go:698: a renewal that failed with a 503 failed the sign-in: grok sign-in failed (unavailable)` (1 `--- FAIL` lines) |
| `TestStartSignInChecksListingBeforeSaving` | any error the listing carries failing the sign-in (B2) (`b2-fail-on-any-listing-error`) | `signinrun_test.go:609: a 403 listing failed the sign-in: grok sign-in failed (auth)` (1 `--- FAIL` lines) |
| `TestStartSignInChecksListingBeforeSaving` | a refusal the listing carries ignored (B2) (`b2-refusal-ignored`) | `signinrun_test.go:574: a sign-in whose listing was refused succeeded` (1 `--- FAIL` lines) |
| `TestStartSignInSavesTheRenewedSession` | a session the issuer refuses to renew saved (B1, B2) (`b2-renewal-refusal-ignored`) | `` (0 `--- FAIL` lines) |
| `TestStartSignInCancelledDuringListingSavesNothing` | no context check immediately before the save (B3) (`b3-no-ctx-check-before-save`) | `signinrun_test.go:762: a sign-in cancelled during its listing succeeded` (1 `--- FAIL` lines) |
| `TestStartSignInCancelledDuringListingSavesNothing` | no context check before the save and none in the renewal wait (B3) (`b3-no-ctx-check-at-all`) | `signinrun_test.go:762: a sign-in cancelled during its listing succeeded` (1 `--- FAIL` lines) |
| `TestAuthenticateCancelledDuringListingStoresNothing` | no context check before the save, through the relay and the real engine (B3) (`b3-acp-no-ctx-check`) | `authreal_test.go:573: auth.json changed:` (1 `--- FAIL` lines) |
| `TestKeyRunStoresNothingOnceCancelled` | no context check before SetKey (B3) (`b3-keyrun-no-ctx-check`) | `agentauth_test.go:258: a cancelled key sign-in gave {Key: Yields:device_key}, <nil>, want a canceled error and no key` (1 `--- FAIL` lines) |
| `TestAuthenticateCancelAfterSaveTellsTheAgent` | the relay's cancel branch never calling Signed for a sign-in that saved first (B3) (`b3-cancel-branch-never-signed`) | `authsignin_test.go:1591: timed out waiting for Signed` (1 `--- FAIL` lines) |
| `TestAuthenticateFreshRefusesNonBoolean` | a null fresh or remote read as absent (`null-fresh-accepted`) | `authsignin_test.go:988: fresh null: error code -32000 {Attempts:0 Code: Command:gobble auth login grok Consent: Error: HTTPStatus:0 Kind:canceled Limi ...` (1 `--- FAIL` lines) |
| `TestStartSignInChecksKiloDeviceKey` | Kilo's device key not listed with (the skip restored) (`kilo-skip-restored`) | `signinrun_test.go:868: the listing was not sent with the device key: the test holds nothing` (1 `--- FAIL` lines) |
| `TestStartSignInChecksPlanListing` | the ChatGPT plan's listing not checked (`plan-listing-skipped`) | `signinrun_test.go:810: the plan's sign-in was not listed: the test holds nothing` (1 `--- FAIL` lines) |
| `TestStartSignInErrorsAreSignInErrors` | StartSignIn's store.Load error not a *SignInError (`load-error-unwrapped`) | `signinrun_test.go:448: an unreadable store: *fmt.wrapError (provider grok: the stored sign-in: disk) is not a *SignInError` (1 `--- FAIL` lines) |
| `TestAuthenticateGuardComesBeforeTheMethodForm` | the one-sign-in guard taken after the method form (`guard-after-form`) | `authsignin_test.go:1507: a second request while the first is asked: code -32000 {Attempts:0 Code: Command:gobble auth login grok Consent: Error: HTTPS ...` (1 `--- FAIL` lines) |
| `TestAuthenticateOneAtATime` | the guard kept per method, not per provider (`guard-per-method`) | `authsignin_test.go:519: authenticate grok map[method:device_code]: no answer within 20s` (1 `--- FAIL` lines) |
| `TestAuthenticateRemoteRefusesLoopbackOnly` | the remote filter skipped under fresh (`remote-filter-skipped-under-fresh`) | `authsignin_test.go:519: authenticate grok map[fresh:true method:browser_oauth remote:true]: no answer within 20s` (1 `--- FAIL` lines) |
| `TestAuthenticateRemoteAPIKeyIsAsLocal` | a remote client naming api_key refused as unsupported (ruling 1) (`remote-api-key-refused-as-before`) | `authsignin_test.go:1552: a key held: remote {"code":-32000,"message":"Authentication required","data":{"command":"gobble auth login grok --method api- ...` (1 `--- FAIL` lines) |
| `TestStartSignInRemoteSetsNoLoopback` | no-loopback not set for one method (ruling 3) (`no-loopback-not-set-for-device`) | `signinrun_test.go:262: grok device_code remote true: recorded [{HTTPClient:<nil> ClientID: Issuer: Identity:{Name: Version:} AgentName:gobble Previous ...` (1 `--- FAIL` lines) |
| `TestAgentRemoteReachesTheEngine` | internal/cli not passing Remote to the engine (ruling 3) (`remote-not-passed-seam`) | `agentauth_test.go:281: remote true: the engine was started with [{Options:{APIKey: CredentialSource: Model: ClientName:gobble ClientVersion:0.0.0-dev  ...` (1 `--- FAIL` lines) |
| `TestAuthenticateRelayEndsItsFormWithoutTheSDK` | the relay not cancelling the form it opened (stop() removed) (`form-never-cancelled`) | `authsignin_test.go:1677: timed out waiting for the open form to be cancelled by the relay` (3 `--- FAIL` lines) |
| `TestAuthenticateSharesAuthJSON` | auth.json's lock released before the read-modify-write; ten runs (`auth-json-unlocked`) | `authreal_test.go:422: the sign-in's session is lost` (10 `--- FAIL` lines) |

**A plant that did not fail, and the panic note.** *(Corrected in the second review, below.)* `b2-renewal-refusal-ignored` removed the `errors.Is(err, llmprovider.ErrAuthFailure)` half of the token wait before the save
(`signinrun.go:178`), which mirrors the wizard's `save`, and exited 0 against this round's tests; the reviewer reached the half with an issuer whose token endpoint answers `invalid_grant` after 11 s, and
`TestStartSignInRenewalOutlivesTheListing` now fails the plant. A panic in the `Wait` goroutine is not recovered: the code in it (`finish`, `checkListing`, `listWith`, `keyRun.Wait`) does no indexing, conversion or
parsing of anything a client sends, and a pasted callback or a form answer goes to the SDK's own goroutine through `Answer`, which I did not audit.

**Gates (C6), after the last change to any Go file** (`acpserver/authsignin.go`'s `switch` to an `if` and the test's middleware helper, after a first lint run of the gates found two findings,
`depguard` on an `encoding/json` import in the new test and `gocritic` `singleCaseSwitch`; that run, `lint-1` to `lint-3` of the first attempt, exit 2, was stopped and the two fixed, and the gates
began again). Each with its start and end time and its exit status taken before any filter.

| Gate | Start | End | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check` on the changed Go files (gofmt, golint, govulncheck) | 14:38:05 | 14:38:27 | 0 |
| cold `golangci-lint` run 1 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 14:38:27 | 14:38:56 | 0 |
| cold `golangci-lint` run 2 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 14:38:56 | 14:39:24 | 0 |
| cold `golangci-lint` run 3 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 14:39:24 | 14:39:53 | 0 |
| cold `golangci-lint` run 4 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 14:39:54 | 14:40:22 | 0 |
| cold `golangci-lint` run 5 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 14:40:23 | 14:40:51 | 0 |
| cold `golangci-lint` run 6 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 14:40:51 | 14:41:20 | 0 |
| `make race` (both modules, `-race`), through `run-heavy` | 14:41:20 | 14:42:39 | 0 |
| `make check-test-isolation` | 14:42:39 | 14:43:39 | 0 |
| `make check-acpext-deps` | 14:43:39 | 14:43:39 | 0 |
| `go test -race -count=1 ./...` and acpext's, `run-on-host --race --need linux`: the Linux machine, linux/amd64, not a WSL distro | 14:43:39 | 14:51:45 | 0 |
| `go test -race -count=1 ./...` and acpext's, `run-on-host --race --need windows` under `sh -c`: the Windows laptop, windows/amd64, native | 14:51:45 | 14:54:07 | 0 |
| `go test -race -count=20 ./llm/provider ./acpserver ./internal/cli`, through `run-heavy` on the session machine, with the default 10 m `-timeout`: `panic: test timed out after 10m0s` in all three packages, no test failed; 20 rounds of each take longer than the bound (a gate that did not run, not one that passed) | 14:54:07 | 15:04:09 | 1 |
| `make preflight`, through `run-heavy` (gofmt, tidy, pre-add-check, vet, staticcheck, golangci-lint, govulncheck, check-acpext-deps, fix-check, archtest, apidiff, verify-build-metadata, check-records, check-icon, markdownlint) | 15:04:09 | 15:11:23 | 0 |
| the same, with `-timeout 60m` (the measured bound: one round of the three is 35, 22 and 46 s without `-race`) | 15:11:52 | 15:34:24 | 0 |

The 20-round race run: `llm/provider` (1114.258s); `acpserver` (910.212s); `internal/cli` (1350.514s), no `--- FAIL`, exit 0.

**Modification times of the staged files** (against the first gate, `pre-add-check`, started 14:38:05): the last Go or golden file, `acpserver/authsignin_test.go`, was written at 14:33:22, before the first gate.
Every Go file and the golden: `acpserver/agent.go` 09:36:26, `acpserver/auth.go` 13:59:53, `acpserver/authissuer_test.go` 14:04:52, `acpserver/authreal_test.go` 14:05:55, `acpserver/authsignin.go` 14:33:22, `acpserver/authsignin_test.go` 14:33:22, `acpserver/doc.go` 14:30:16, `acpserver/testdata/authenticate-elicitation.golden` 09:42:51, `internal/auth/clients.go` 09:37:12, `internal/cli/agent.go` 09:38:13, `internal/cli/agentauth.go` 14:06:22, `internal/cli/agentauth_test.go` 14:09:53, `internal/cli/login.go` 09:37:20, `llm/provider/list.go` 13:57:39, `llm/provider/login.go` 09:33:11, `llm/provider/login_test.go` 11:04:30, `llm/provider/signin.go` 14:30:43, `llm/provider/signinrun.go` 14:30:43, `llm/provider/signinrun_test.go` 14:08:05.
The text files, which no gate but `check-records` and `markdownlint` reads: `README.md` 12:50:24, `docs/README.md` 14:40:01, `docs/architecture.md` 14:11:42, `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` 14:13:43, `docs/decisions/0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` 14:40:18, `docs/guides/credentials.md` 14:43:44; `make check-records markdownlint` ran after the last record edit (see the return).

**Not verified (first review round).** No live service; the real endpoints of Grok, Kilo and OpenAI; a real client's rendering; the phone flow. (The plan sign-in's bytes, the `null` question and the token wait that were listed here are dealt with below.)

#### Second review and last fixes (2026-11-16)

**The second verdict.** The reviewer's second round found the code correct: B1, B2 and B3 are fixed, its own breaks of those fixes fail as they should, and its 20 race rounds passed. It was not yet
fit to commit: three false statements, three test items and, from the owner, three rulings remained. A mainstream agent at medium effort, on the coordinator's brief, fixed them. The review round's
own mistake is among them: this record said no `//nolint` was left in what the phase adds, and `llm/provider/signinrun_test.go` held two (`:639`, `:641`); they are gone and the staged Go diff's added lines
hold none (`grep -c nolint`, 0).

| Item | What was done | Test | Plant (failure seen) |
| :--- | :--- | :--- | :--- |
| 1, MADR D3's "no test dials a port" was false | `TestAuthenticateAbandonedLeavesNothing` (`acpserver/authreal_test.go`) no longer dials: `callbackServers` counts goroutines whose stack holds `serveCallbackListeners.func1` (the SDK's `llmprovider/auth/oauth_loopback.go:530` at `v1.31.0`, the function's first closure, seen in a real dump as present while the sign-in runs); present while it runs, gone, polled, after the abandon | that test | `abandon-not-ended`: `authreal_test.go:363: the abandoned sign-in's run has not ended` (1 `--- FAIL` lines) |
| 2, two `//nolint:errcheck` | the `RoundTripper` of `renewingIssuer` returns the read and parse errors (`signinrun_test.go`); count 0 | `make lint` | none |
| 3, the guide's "the one thing" | the guide names `host-id` (created by `hostSignIn`, `internal/cli/login.go:34`, through `auth.HostID`, before the engine starts) and `clients.json`, neither holding a credential; the D4 step 3 note corrected to match; `TestCLIFillsHostID` (`internal/cli/agentauth_test.go`) already holds that `host-id` exists after a sign-in that is then recorded as failing, so no assertion was added | `TestCLIFillsHostID` | `acp-host-id-not-set` (first round) |
| 4, `TestAuthenticateSharesAuthJSON` could pass with its defect | the writer holds the lock until `authenticate` has answered or 3 s pass, then writes from its earlier read (`authreal_test.go`, `otherWriter`) | that test | `auth-json-unlocked`: `authreal_test.go:416: the sign-in's session is lost` (10 `--- FAIL` lines); `auth-json-unlocked-delayed` (lock skipped and a 400 ms delay): `authreal_test.go:416: the sign-in's session is lost` (10 `--- FAIL` lines) |
| 5, the token-wait clause untested | `TestStartSignInRenewalOutlivesTheListing` (the reviewer's `TestR2RenewalOutlivesTheListing`, both cases, about 12 s each, `listingBound` + 1 s: the SDK bounds a listing at 10 s) | that test | `b2-renewal-refusal-ignored`: `signinrun_test.go:998: a renewal the issuer refused gave <nil> (kind other), want an auth failure` (1 `--- FAIL` lines) |
| 6, architecture row | `docs/architecture.md`'s `llm/provider` row: `Wait` fails only when the provider refuses the credential (a refused renewal is one) or its context is done, and saves nothing then (`signinrun.go`, `finish`) | `make check-records markdownlint` | none |
| 7, "had already completed" | the guide says "had begun", and the neighbouring sentences are the code's: the save is checked at `signinrun.go:182` and written after it; D4 step 3's note says the same | none | none |
| 8, byte proof for a failed plan sign-in | `TestStartSignInPlanFailureLeavesAuthJSON` (`authreal_test.go`): the issuer answers as OpenAI's with the plan grant's scope, every host routed to it (`toOne`), `provider.StartSignIn` over file-backed `auth.ClientLogins`, a stored `openai` session; a 401 on the listing, and separately a cancel during it: `auth.json` byte-identical, no revoke, and `clients.json` logged as present, holding no credential | that test | `plan-login-deleted-before-save`: `authreal_test.go:660: auth.json changed:` (1 `--- FAIL` lines) |
| A, `null` | `method`, `fresh` and `remote` refuse `null` (`acpserver/auth.go:128`); `_meta.gobble: null` is an absent container | `TestAuthenticateFreshRefusesNonBoolean` (`method null`), `TestAuthenticateNullContainerIsAbsent` | `null-method-accepted`: `authsignin_test.go:988: method null: error code -32000 {Attempts:0 Code: Command:gobble auth login grok Consent: Error: HTTPStatus:0 Kind:ca ...` (1 `--- FAIL` lines); `null-container-refused`: `authsignin_test.go:1688: _meta.gobble null: {"code":-32602,"message":"Invalid params","data":{"reason":"_meta.gobble is not an authenticate  ...` (1 `--- FAIL` lines) |
| B, Kilo's 401 | no code change; D4's dated note records the wizard's rule, the difference from Kilo's client (`packages/kilo-gateway/src/api/models.ts:227-245` at `58072bd696`, read) as not verified live, and that a question against go-llmprovider-sdk is open on the coordinator's ledger, not filed from here | none | none |
| C, the guard | no code change (the guard is taken after step 2, so a request that needs no sign-in is answered `{}`); D4's dated note; the guide says so | `TestAuthenticateGuardAppliesToSignInsOnly` | `guard-on-every-request`: `authsignin_test.go:1709: a request that needs no sign-in, while one runs: {"code":-32600,"message":"Invalid request","data":{"provider":"gro ...` (1 `--- FAIL` lines) |

**The owner's rulings, each the owner's ruling, relayed by the coordinator:** "Refuse `method: null`; read `gobble: null` as absent" (A); "Keep the wizard's rule: a 401 refuses" (B); "Keep `{}` for a request that needs no sign-in" (C). Dated notes at MADR D4 step 1,
step 3 and "One sign-in per provider", and the answered question in the MADR's "Revisions of 2026-11-16"; the guide's `_meta.gobble` section, one-sign-in paragraph and cancel paragraph say each.
The reservation's being held for as long as a client leaves the method form unanswered is said in the guide's one-sign-in paragraph and the D4 bullet.

**Plants of this round, on scratch copies of the final tree** (two shards at a time, `HOME`, `XDG_*`, `GOBBLE_HOME` in the scratch, `GOPROXY=off`, no HTTP proxy variable): **73 plants run, all 73 seen failing with a `--- FAIL`, none a build failure, none that does not fail** (the 67 of the first review round, `b2-renewal-refusal-ignored` now failing, and six new: `abandon-not-ended`, `auth-json-unlocked-delayed`, `null-method-accepted`, `null-container-refused`, `guard-on-every-request`, `plan-login-deleted-before-save`; `auth-json-unlocked` as before, now against the new test); the unbroken runs of the same tests, `clean2.out`, are 37, exit 0 each. `abandon-not-ended` (the sign-in started detached from the request and never cancelled by the relay) fails the test on the run still being open (`authreal_test.go:363`), before it reaches the final wait for the frame to go. That wait was then seen to fail on its own, on a scratch copy only (no Go file of the tree changed): under the same plant, with the run-ended assertions removed from the copy, `TestAuthenticateAbandonedLeavesNothing` failed 3 of 3 with `authreal_test.go:357: timed out waiting for the callback server's goroutine to end`; the same copy without the plant passed 3 of 3. So the frame tracks the engine's life, not the request's: it stays while the sign-in is not ended. The two assertions fail together under any break of gobble's that leaves the sign-in running, which is why the tree keeps one plant for both; the frame is real and counted (the assertion that it is present while the sign-in runs passes on the unbroken tree and on this plant).



**Gates (C6), after the last change to any Go file** (the last write of a Go file is at `llm/provider/signinrun_test.go` at 16:42:36, before the first gate (`pre-add-check`, started 17:09:01)), each with its start and end time and its exit status taken before any filter:

| Gate | Start | End | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check` on the changed Go files (gofmt, golint, govulncheck) | 17:09:01 | 17:09:26 | 0 |
| cold `golangci-lint` run 1 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 17:09:26 | 17:09:52 | 0 |
| cold `golangci-lint` run 2 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 17:09:52 | 17:10:19 | 0 |
| cold `golangci-lint` run 3 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 17:10:20 | 17:10:48 | 0 |
| cold `golangci-lint` run 4 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 17:10:48 | 17:11:16 | 0 |
| cold `golangci-lint` run 5 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 17:11:16 | 17:11:44 | 0 |
| cold `golangci-lint` run 6 of six (`make lint`, a fresh `GOLANGCI_LINT_CACHE`), through `run-heavy`: 0 issues in the root module and in acpext | 17:11:45 | 17:12:10 | 0 |
| `make race` (both modules, `-race`), through `run-heavy` | 17:12:11 | 17:13:38 | 0 |
| `make check-test-isolation` | 17:13:38 | 17:15:02 | 0 |
| `make check-acpext-deps` | 17:15:02 | 17:15:02 | 0 |
| `go test -race -count=1 ./...` and acpext's, `run-on-host --race --need linux`: the Linux machine, linux/amd64, not a WSL distro | 17:15:02 | 17:22:27 | 0 |
| `go test -race -count=1 ./...` and acpext's, `run-on-host --race --need windows` under `sh -c`: the Windows laptop, windows/amd64, native | 17:22:27 | 17:24:49 | 0 |
| `go test -race -count=20 -timeout 60m ./llm/provider ./acpserver ./internal/cli`, through `run-heavy` on the session machine | 17:24:49 | 17:51:25 | 0 |
| `make preflight`, through `run-heavy` | 17:51:25 | 17:58:59 | 0 |

The 20-round race run: `llm/provider` (1595.159s); `acpserver` (971.973s); `internal/cli` (1344.086s), no `--- FAIL`, no data race.

The staged Go files and the golden, by modification time: `acpserver/agent.go` 09:36:26, `acpserver/auth.go` 16:40:47, `acpserver/authissuer_test.go` 16:40:47, `acpserver/authreal_test.go` 16:42:36, `acpserver/authsignin.go` 14:33:22, `acpserver/authsignin_test.go` 16:40:47, `acpserver/doc.go` 14:30:16, `acpserver/testdata/authenticate-elicitation.golden` 09:42:51, `internal/auth/clients.go` 09:37:12, `internal/cli/agent.go` 09:38:13, `internal/cli/agentauth.go` 14:06:22, `internal/cli/agentauth_test.go` 14:09:53, `internal/cli/login.go` 09:37:20, `llm/provider/list.go` 13:57:39, `llm/provider/login.go` 09:33:11, `llm/provider/login_test.go` 11:04:30, `llm/provider/signin.go` 14:30:43, `llm/provider/signinrun.go` 14:30:43, `llm/provider/signinrun_test.go` 16:42:36. The text files, which only `check-records` and `markdownlint` read: `README.md` 12:50:24, `docs/README.md` 14:40:01, `docs/architecture.md` 16:41:40, `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` 16:41:40, `docs/decisions/0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` 16:45:00, `docs/guides/credentials.md` 16:41:40; `make check-records markdownlint` ran after the last record edit.

**Not verified.** No live service: the real endpoints of Grok, Kilo and OpenAI, whether real Kilo answers a 401 to a valid device key with no organisation (the owner's ruling B rests on the wizard's rule and Kilo's source, not on a
run), a real client's rendering, the phone flow. The plan sign-in's byte proof runs against an in-process issuer that grants the plan scope, not OpenAI's.

**Staged, not committed; the coordinator commits on the branch.**

### P7a record (2026-11-16)

Built by a mainstream model at medium effort, on the coordinator's brief. Step 1's question was answered earlier the same day by a lightweight agent, read-only, and
re-checked by the coordinator; this entry re-read that agent's evidence files before recording the answer, and corrects what they do not bear out (below). The phases
were approved by the owner; P7a was added on 2026-11-14 by the owner's ruling on P2's observation; on 2026-11-16 the owner said "proceed" to having step 1's question
answered, then ruled on the two points that answer raised, each "the owner's ruling, relayed by the coordinator": **"Count it against every library of that repository"**
(an entry that names no known module; dated note at step 2) and **"Yes, in P7a, by a dated amendment"** (the 30-advisory page limit; dated amendment at P7a's head). Times
are the session machine's local time. Staged, not committed; **the coordinator commits on the branch `p3-acpext-v0.5.0`.** `main` and `origin/main` stayed at `9e21c8f`;
no tag, push or merge was made. No Go file changed, and the `Library` table's entries are as they were.

**C1, the start state.** Branch `p3-acpext-v0.5.0`, `HEAD` `a25aacf`, level with `origin/p3-acpext-v0.5.0` (`git branch -vv`, ran), the tree clean (`git status --short`
printed nothing); `main` at `9e21c8f`.

**Step 1's answer (the question of step 1, answered).** Read from the evidence files of the earlier agent (`otel_advisories.json`, 13 advisories, 25 `vulnerabilities[]`
entries; `prom_advisories.json`, 6 advisories, 6 entries; `owner_sdk_advisories.json`, `gin_advisories.json`, `k8s_advisories.json`, each `[]`; `global_go_advisories.json`, 5
advisories, 6 entries) with a script of my own (ran), and the script's code (read):

* `advisories()` called `GET https://api.github.com/repos/<owner>/<repo>/security-advisories`, unauthenticated, with no query string and no pagination; GitHub's default is
  30 a page ("per_page", default 30, read on the endpoint's documentation page, below).
* In the public multi-module Go repository `open-telemetry/opentelemetry-go` (`otel_advisories.json`), all 25 entries carry `package.ecosystem: "go"` and a non-empty
  `package.name` that is a Go module path, and different entries of one advisory name different paths (`go.opentelemetry.io/otel/sdk`, `.../otel/trace`, `.../otel/baggage`,
  `.../otel/schema/v1.1`). None of the 25 names has a trailing slash, an upper-case letter or surrounding space. The 5 entries of `global_go_advisories.json` (GitHub's
  global advisory listing, not a repository's: no `state` key) are the same, e.g. `github.com/filebrowser/filebrowser/v2`.
* In `prometheus/prometheus` (`prom_advisories.json`) all 6 entries carry `package.ecosystem: ""` and a free-text name (`prometheus` or `Prometheus`, which differ by case between
  entries). GitHub's published schema marks `vulnerabilities`, `package` and `package.name` nullable: "`package`: required, object or null", "`vulnerabilities`: required,
  array of objects or null", "`name`: required, string or null" (`https://docs.github.com/en/rest/security-advisories/repository-advisories`, fetched 2026-11-16). No entry
  with a null package, a null name or null `vulnerabilities` occurs in any of the evidence files; that case is built from the schema, not seen in data.
* The SDK repository's own advisories are `[]` (`owner_sdk_advisories.json`; the two `--lib` runs below read the same empty list live), so nothing was observed for the contract's
  module. The two `module` values in the script's table are the SDK's path and its `/authcontract` path.
* So each entry normally carries `package.name`, and for a Go advisory it is the module path; the name is whatever the author typed and can be missing. Step 2 is built as
  written, with the owner's ruling for an entry that names no known module.
* The brief's version of this answer is borne out by the files, with two precisions: the "empty ecosystem and a free-text product name" project is `prometheus/prometheus`, whose names differ
  by case between entries (so a case-folding tolerance would have matched some and not others, a reason to count such names as unnamed rather than guess); and the null cases are the
  schema's, not observed.

**What changed (staged).** `scripts/check_libs.py` (line numbers at the staged file):

1. **Step 2, by package, with the ruling.** `advisory_scope` (`:292`) returns `"named"` when the entry's `package.name` equals `lib.module`, `"other"` when it equals the module
   of another library of the same repository, `"unnamed"` otherwise (`package` missing, null or not an object; `name` missing, null, empty or not a string; or a name that equals the
   module of no library the script lists for that repository). `advisory_hits` (`:310`) skips an `"other"` entry (`:321`), keeps the existing rules (state `published`, a
   `vulnerable_version_range` that `covers` the tag), and for an `"unnamed"` entry appends the message (`:324-326`):
   `advisory <GHSA> covers <tag>; its entry names no known module of <repo>, so it counts against every library there: fix the advisory's package name to the module path (<the repository's modules, sorted, comma-separated>)`.
   The same GHSA hit twice for one library (two entries) is listed once. **Exact comparison, no tolerance:** the data shows Go names are module paths, written exactly (none of the 25
   Go names has a trailing slash or a capital), and Go module paths are case-sensitive, so `github.com/Maccavelli/...` is a different path. A mis-typed name (an upper-case letter, a
   trailing slash) is therefore "names no known module", counted against both libraries with the message telling the owner to fix it: the fail-closed rule is the safety net, and a
   tolerance would be a rule the data does not call for and could match the wrong module. `check_library`'s inline loop is now `check_advisories(report, lib, tag)` (`:439`); `check_advisories`
   (`:331`) is the same try/except as before for the report's line.
2. **Ruling 2, every page.** `fetch_page` (`:262`) returns one page and the URL its `Link` header gives for `rel="next"` (`next_link`, `:254`); it raises on a network error, a bad status, unreadable
   JSON, or a body that is not a list. `advisories(lib, fetch=fetch_page)` (`:275`) starts at `.../security-advisories?per_page=100` (`PER_PAGE`, `:78`), follows each next link, at most
   `MAX_PAGES = 20` pages (`:79`; 2000 advisories), refuses a next link that is not under `https://api.github.com/` (`:287`), and raises `RuntimeError` when the bound is passed (`:289`).
   `fetch` is the one injected thing; there is no other production path for the test. **A failed page:** the exception of any page, the first or a later one, goes out of `advisories`
   (no `try` in it), and `check_advisories` (`:331-337`) turns `OSError`, `ValueError` (JSON, a non-list body, an off-host link, an unreadable range) and `RuntimeError` (the bound) into a failing
   `advisories` check: `could not read all the advisories, so none is taken as absent: <error>`. A partial list is never used: the first page's advisories are dropped with the failure, because the
   check fails whatever they held.
3. **Docs in the script.** The module docstring's step 4 and `--help`'s epilog (`:599`) say: every page is read; advisories are matched by module path; an entry that names no known module counts
   against every library of its repository; to have an advisory matched, write the package name as the module path.
4. **A suppression added, with its reason (the brief's rule).** `# noqa: S310` on the `urllib.request.Request` line (`:266`): with the URL now a variable, ruff's S310 flags it as it flags the
   `urlopen` below it, which already carried the same suppression; the URLs are guarded by `advisories()` (https://api.github.com only). The repository runs no Python linter (`Makefile` has no ruff; `make preflight`
   runs some `scripts/*.py` as checks and lints none), so this keeps the file's findings under `ruff check --select S,BLE,ANN,E,F --line-length 140` (ran; ruff is on this machine, not a gate) at the 7 it had at `HEAD`.
   `_raises` (`:537`) carries a `# noqa: BLE001` for the same reason as `pinned_at_tag_not_compared`'s.

**The pagination facts, with their source.** The endpoint's page (`https://docs.github.com/en/rest/security-advisories/repository-advisories`, fetched 2026-11-16): "per_page: The number of
advisories to return per page", default 30, no maximum stated; "before", "after": "A cursor, as given in the Link header. If specified, the query only searches for results before/after
this cursor"; no `page` parameter for this endpoint. The guide (`https://docs.github.com/en/rest/using-the-rest-api/using-pagination-in-the-rest-api`, fetched 2026-11-16): "You can use the `link`
header from the response to request additional pages of data."; "The URL for the next page is followed by `rel="next"`."; "In some cases, only a subset of these links are available."; "For most
endpoints, the maximum value of `per_page` is `100`." A real response (`gh api -i "repos/open-telemetry/opentelemetry-go/security-advisories?per_page=2"`, ran, read-only, 200) carried
`Link: <https://api.github.com/repositories/<id>/security-advisories?per_page=2&after=<cursor>>; rel="next"` and no `rel="last"`: the script follows `next` until it is absent, and never builds a
cursor itself. The documentation does not state that 100 is allowed for this endpoint, so `per_page=100` is a request GitHub may reduce; the script follows `next` either way. **A live multi-page run**
(ran, anonymous, read-only, `scratchpad/gobble-p7a/live.py`): `advisories()` against `open-telemetry/opentelemetry-go` with `PER_PAGE` set to 5 read pages of 5, 5 and 3 advisories, 13 distinct GHSA ids,
the count of `otel_advisories.json`.

**Self-test (`--self-test`, `advisory_cases`, `:461`; fixed data, no network).** 13 new cases, 30 in all, all `ok` (`st-final.txt`, 19:27:03, exit 0): (a) the SDK's range `< 1.2.0` for the SDK's path
alone does not hit the contract at `v0.1.0` and hits the SDK at `v1.1.0`; (c) an entry naming the contract does not hit the SDK and hits the contract; (b) entries with no package, no name, an empty name,
and the name `go-llmprovider-sdk` each hit both libraries with the message; a named entry's hit carries no such message; a differently cased path names no known module; another repository's path names no known module;
a draft advisory is not counted; (d) the advisory on the second page is found, through `check_advisories` as well as `advisories`; (e) a second page that fails to load fails the check ("could not read"),
and is not read as empty; an endless chain of next links stops at the bound; a next link off `api.github.com` is refused; the `Link` header's `rel="next"` is read (and `prev`, and an empty header, give none).

**Plants (C2), each on a scratch copy of the final script, `<TMP>` the scratch directory** (`plant.py` applies one replacement with an `assert` that it matches once, runs `python3 <TMP>/check_libs.py --self-test`; all exit 1, no traceback;
the unbroken script's 30 cases pass):

| Plant | Failing lines |
| :--- | :--- |
| the package filter removed (`scope == "other" or` dropped, `:321`): the contract at `v0.1.0` reads the SDK's range as covering it | `self-test an advisory range for the SDK's path alone does not hit the contract at v0.1.0: FAIL`; `self-test an entry naming the contract does not hit the SDK, and hits the contract: FAIL` |
| the unknown-module rule turned into "ignore" (`advisory_scope` ends `return "other"`) | `self-test an entry with no package, no name, an empty name or a name matching no library hits both libraries, with the message: FAIL`; `...the name is compared exactly: a differently cased path names no known module: FAIL`; `...a name that is no module of the repository (another repository's path) names no known module: FAIL` |
| the second page dropped (`advisories` returns after the first page) | `self-test the advisory on the second page is found: FAIL`; `...a second page that fails to load fails the check and is not read as empty: FAIL`; `...an endless chain of next links stops at the bound: FAIL`; `...a next link off api.github.com is refused: FAIL` |
| a failed page read as empty (`fetch` wrapped in `except OSError: return found`) | `self-test a second page that fails to load fails the check and is not read as empty: FAIL` |
| the page bound removed (`return found` for the `RuntimeError`) | `self-test an endless chain of next links stops at the bound: FAIL` |
| the off-host check removed | `self-test a next link off api.github.com is refused: FAIL` |

A first run of the last plant crashed the self-test with a traceback (the unrefused endless chain raised `RuntimeError` where the case expects `ValueError`); `_raises` now returns false for any other exception, so the plant
fails the case line as shown, and all six plants were rerun after that change.

**Gates (C6).** The script was last modified at 19:21:50, before the first row; each exit status was taken from the command's own `$?` before any filter. `--lib` runs went through `run-heavy`, which queues on the session machine.
The checker clones the committed `HEAD` (`a25aacf`), not the working tree (P2's Finding 1): the Go checks therefore test the committed gobble at the pinned tag, and the script that ran is the staged one. P7a changes no Go file, so
the staged script's `--lib` verdicts rest on that commit; the owner's "on the committed step" acceptance is met again by the coordinator's rerun after the commit, if wanted.

| Gate | Start | End | Exit |
| :--- | :--- | :--- | :--- |
| `python3 scripts/check_libs.py --self-test` (30 cases) | 19:27:03 | 19:27:03 | 0 |
| `run-heavy -- python3 scripts/check_libs.py --lib go-llmprovider-sdk`: `0 failing checks; scratch removed`; `go-llmprovider-sdk advisories: ok` (pinned and checking `v1.31.0`) | 19:21:57 | 19:24:16 | 0 |
| `run-heavy -- python3 scripts/check_libs.py --lib authcontract`: `0 failing checks; scratch removed`; `authcontract advisories: ok` (pinned and checking `v0.1.0`) | 19:24:16 | 19:26:33 | 0 |
| `run-heavy -- make preflight` (includes `check-records` and `markdownlint`; run before the two gate rows of this table were added to this record) | 19:28:07 | 19:41:10 | 0 |
| `make check-records markdownlint`, rerun after the last edit of this PLAN (the two gate rows) | 19:41:38 | 19:41:40 | 0 |

The Go race and the cross-host runs (Linux, Windows) were not rerun: the change is a Python script, two docs and this PLAN; no Go file, `go.mod` or `go.sum` changed, so the Go tests' inputs are the same as at `a25aacf`. The staged
files' modification times: `scripts/check_libs.py` 19:21:50; `docs/architecture.md` 19:22:13 (not read by any Go test; `docs/` is read by `make check-records` and markdownlint, which ran after it).

**Docs (C9).** Searched `README.md`, `AGENTS.md`, `Makefile`, `docs/architecture.md`, `docs/README.md`, `docs/guides/*.md` and `.github` for `check_libs`, `check-libs` and `advisor`. Rows:
`docs/architecture.md` (the `make check-libs` bullet of the preflight list): **fixed**, it now says every page is read, a failed page fails the check, matching is by module path, an entry that names no known
module counts against every library of its repository with a message to fix the advisory's package name, and how to write an advisory so it is matched. `docs/README.md`'s 0051 PLAN row: **fixed** (P7a built and staged).
The other hits are records (`docs/decisions/0004`, `0019`, `0021`, `0023`, `0031`, `0049`, `0050`, `0055`, and the PLAN's own earlier records) that describe what the script did when they were written, and stay as they are; the `Makefile`'s
`check-libs` target and `README.md`, `AGENTS.md` and the guides do not describe what it checks.

**Not verified.** The SDK repository has no published advisory, so nothing live matched the SDK's or the contract's module path: the matching rule is held by the self-test over fixed data built from the real shape of other repositories'
entries, and the live path was exercised against the SDK repository (an empty list) and, for several pages, against `open-telemetry/opentelemetry-go`, not against an advisory naming the contract. The null `package` and null `name`
cases come from the schema, not from an observed response. Whether GitHub honours `per_page=100` on this endpoint is not stated in its documentation (the 5-per-page run shows `per_page` is honoured below the default). An authenticated
caller would also see draft and triage advisories; the script is anonymous and filters on `state`, as before.

**Staged, not committed; the coordinator commits on the branch.**

### P7 record (2026-11-16)

Built by a mainstream model at medium effort, on the coordinator's brief. The phases were approved by the owner: P7 follows under the approval of 2026-11-11 for the phases
that do not wait on the SDK, the SDK's `v1.31.0` being released; the phases on this branch run in order, the coordinator committing each after review. This record quotes no
new owner statement. Times are the session machine's local time. Staged, not committed; **the coordinator commits on the branch `p3-acpext-v0.5.0`.** `main` and
`origin/main` stayed at `9e21c8f`; no tag, push or merge was made. No change in `acpext/`, the SDK, `go.mod` or `go.sum`.

**C1, the start state.** Branch `p3-acpext-v0.5.0`, `HEAD` `1305a61`, level with `origin/p3-acpext-v0.5.0` (`git rev-parse`, ran), the tree clean (`git status --short` printed
nothing); `main` at `9e21c8f`. `go.mod` requires acpext `v0.5.0-rc.1` and go-llmprovider-sdk `v1.31.0`, no `replace`, no `go.work`.

**What was already true at `1305a61`** (the brief's question 1; read at that commit, `git show 1305a61:<path>`):

* The plan's `internal/cli/acp.go:96-125` is `acpLogin`, at `acp.go:96` still; the plan's `internal/cli/login.go:48-63` is `signIn`, at `login.go:71` (P6 added `hostSignIn` above it).
* **Step 1 was not true.** `acpLogin` took the sign-in with `e.login(id, device)` (`acp.go:102`), which is `auth login`'s no-menu path (`loginWith`, the provider's default method); it did not call
  `signIn`. That is F53.
* **Step 2 was half true.** `gobble auth login` already passed the opener and the device code's text: `o.ShowDeviceCode = e.showDeviceCode` and `o.OpenURL = e.openLoginURL`
  (`auth.go:297-298`). `signIn` did not: `provider.SignInOptions{}` (`login.go:85`), so `/login`, `gobble setup` and the first run opened no browser and showed the device code
  through the SDK wrapper's default notice (`llm/provider/signin.go:320`), with no expiry. `showDeviceCode` (`auth.go:314`) printed no expiry either.
* Already true and relied on: `provider.DeviceCode.Expires` and its fill from the wizard's code (`signin.go:318`, `Expires: dc.Expiry`); `signInTo` fills `HostID` on the terminal path (P6).

**What was built (staged).**

1. **Step 1, `internal/cli/acp.go:99-112`.** `acpLogin` for a provider with a sign-in, on a terminal (`e.onTerminal()`, `login.go:100`), runs `e.signIn(e.ctx, e.terminalQuestions(e.ctx), id)` (`:108`):
   `/login`'s path, so the wizard asks its menu of the offered methods, the API key among them, through the terminal's questions. `errCancelled` is "sign-in cancelled; nothing was saved"
   (exit 1, as `gobble setup`'s). **Two cases keep the earlier path, `e.login(id, device)`:** `--device`, which names the method, and a run with no terminal to ask on (`:105`). This is
   not a loosening: with no terminal a menu cannot be asked, `TestACPLogin` runs `acp --login grok` with none and the "no terminal" subtest of `TestConsentWordingOnEverySurface` requires
   `acp --login openai` to fail closed with `errNoTerminal`, both as at `1305a61`. A provider with no sign-in keeps its key path (`readAPIKey`, `readWorkspace`, `auth.SetKey`), which F53 does not
   concern: its key was never lost. `terminalQuestions` (`login.go:95`) and `onTerminal` (`login.go:100`) are `gobble setup`'s construction of its questions, moved out of `SetupCmd.Run`
   (`login.go:325-328`) so both use it; unexported, no new seam. The `--login` flag's help (`acp.go:23`) says it shows `/login`'s menu on a terminal.
2. **Step 2, `internal/cli/login.go:88`.** `signIn` passes `provider.SignInOptions{ShowDeviceCode: e.showDeviceCode, OpenURL: e.openLoginURL}`: `/login`, `gobble setup`, the first run
   and now `acp --login` all come through `signIn`, so the four open the host's browser and show a device code as `auth login` does. **The device code's text**, `internal/cli/auth.go:317-325`:
   `Open <url> and enter the code <code> (expires 14:05)`, the time `d.Expires.In(e.now().Location()).Format("15:04")`; a code with a zero `Expires` names none (the SDK's never has one: question 2).

**Answers to the brief's questions 2 to 4.**

* **Question 2, the expiry.** The SDK's device step carries an absolute time and no duration: `wizard.DeviceCode.Expiry time.Time`, "when the code stops being accepted. It is never zero: a provider that gives none has a
  default, and OpenAI's is this module's own wait" (go-llmprovider-sdk `v1.31.0`, `wizard/device.go:24-27`, read), built from `DeviceLogin.Expiry` (`llmprovider/auth/oauth_device.go:88`, `:289`, `:379`: the deadline,
  from the issuer's `expires_in`, else `defaultDeviceExpiry`, 300 s, `:24`, `:326`). So there is an expiry to name for every provider that signs in by device code (Grok, Kilo); none is invented, and the zero case
  (a test's code) names none. `llm/provider` hands it on in `DeviceCode.Expires` (`signin.go:318`; `TestSignInDevice`'s `shown[0].Expires.IsZero()` check, `signin_test.go:251`, holds that it arrives non-zero from a real
  in-process issuer). **Local time, made testable:** the time is written in the zone of `runEnv.now`, gobble's clock (`main.go:90`, `clock = time.Now`, replaced by tests: `main.go:217`; `limitLine` writes its reset time the
  same way, `limit.go:48`), so in production it is the machine's local zone and a test that sets the clock's zone gets the same words on any machine. No new seam, no new exported name. (`modelEnv`'s `runEnv` has no
  `now`; a test that reaches `showDeviceCode` with an expiry sets one, as `TestDeviceCodeTextNamesExpiry` and `TestAuthLoginGrok` do through `clock` and `h.e.now`.)
* **Question 3, F53 and the key's path.** F53 is `0052-MADR`'s: `acpLogin` took the sign-in when `HasLogin`, so `acp --login openai` lost its key path (0031-PLAN P4b-2 step 4, "offers the plan and the API key"). **The key is stored
  by gobble's `signInTo`, not by the wizard**: `provider.SignIn` returns `Credential{Key: res.APIKey}` for a key chosen (`signin.go:380`) and `signInTo` calls `auth.SetKey` (`login.go:133`), escaping a device's. **The
  order on the terminal path is the wizard's already:** the wizard lists with a typed key and asks again while the listing refuses it ("a typed key checked by a listing and asked for again while refused", `signin.go:294-295`),
  and only a key it accepts comes back for `signInTo` to store. P6's "check first, save only if accepted" is `StartSignIn`'s `Wait` (the ACP path), which does the same on its side. No save order was changed.
* **Question 4, `acp --login` and the key.** MADR D3 says it: "**`gobble acp --login <id>` shows the wizard's menu** of the offered methods, the API key included, instead of the bare login
  (`internal/cli/acp.go:101-103`). This restores what 0031-PLAN P4b-2 step 4 asked, "`gobble acp --login openai` offers the plan and the API key" ... for every provider (F53)" (`0051-MADR` `:424-427`), and D3's terminal-surfaces
  bullet lists `gobble acp --login` among the terminal surfaces (`:419-420`). D4's "no key over ACP" is about `authenticate`, a request on the wire; `acp --login` is the terminal an ACP client opens, where a key is typed.

**Tests (C2), each seen failing on a scratch copy of the tree** (`<scratch>/gobble-p7/plants.py`; `git ls-files -co` copy with no `.git`; `HOME`, `USERPROFILE`, `XDG_*` and `GOBBLE_HOME` in the scratch;
`GOPROXY=off`; `GOFLAGS=-mod=mod`; the module and build caches the real ones; one `go test -count=1 -timeout 240s -run '^(<test>)$' ./internal/cli/` each, on the final tree). The unbroken runs of the same tests
are 5, exit 0 each (`clean.out`). Each plant exits 1 with a `--- FAIL`, none a build failure.

| Test | Asserts | Planted break | Failure seen |
| :--- | :--- | :--- | :--- |
| `TestACPLoginOffersMenu` (`acp_test.go:54`), "the menu": the real wizard (`provider.SignIn`, through `planSignIn`) over the test's plan issuer | `acp --login openai` on a terminal prints the wizard's menu, `1  OpenAI API key` and `2  Use your ChatGPT plan with this app`, and choosing the plan goes on to its consent | the bare login restored (`acp-bare-login-restored`: the terminal branch replaced by `e.login`) | `acp_test.go:64: acp --login openai on a terminal: context deadline exceeded, want the sign-in declined at the plan's consent` (the bare path asks no menu, so the test's answers do not fit and its 15 s bound ends it) |
| `TestACPLoginOffersMenu`, "the key" | on a terminal the sign-in is asked with no method forced, the opener and the device code's text set, and a key the menu's first choice gives is stored in `auth.json` by `signInTo` (F53) | the same plant | the same test fails in the same run (one `--- FAIL`); a stand-in is used here because a typed key needs the terminal's raw mode, which a pipe cannot give (below) |
| `TestLoginOpensBrowser` | `gobble auth login grok`, `/login grok`, `gobble setup` and the first run each call the opener once, with the authorize URL | `/login` passing no opener (`login-no-opener`) | `loginsurfaces_test.go:80: /login grok opened [], want the authorize URL once` |
| `TestLoginOpensBrowser` | the same | `auth login` passing no opener (`auth-login-no-opener`) | `loginsurfaces_test.go:68: gobble auth login grok opened [], want the authorize URL once` |
| `TestLoginOpensBrowser` | once, not twice | `/login`'s opener called twice (`login-opener-twice`) | `loginsurfaces_test.go:80: /login grok opened ["https://auth.x.ai/oauth2/authorize?state=s" "https://auth.x.ai/oauth2/authorize?state=s"], want the authorize URL once` |
| `TestDeviceCodeTextNamesExpiry` | `auth login grok --device` and `/login grok` name `(expires HH:MM)` in the clock's zone: 14:05 for a clock in UTC, 16:05 at +02:00, 09:05 at -05:00; a code with no expiry names none | the expiry dropped (`expiry-dropped`) | `loginsurfaces_test.go:130: a clock in UTC: exit 0, stderr "Open https://accounts.x.ai/device and enter the code ABCD-1234\n", want "... (expires 14:05)\n"` |
| `TestDeviceCodeTextNamesExpiry` | the zone is the clock's | the time written in UTC (`expiry-in-utc`) | `loginsurfaces_test.go:130: a clock in +02:00: exit 0, stderr "... (expires 14:05)\n", want "... (expires 16:05)\n"` |
| `TestDeviceCodeTextNamesExpiry` | `/login` shows the text too | `/login` given no device-code text (`login-no-device-text`) | `loginsurfaces_test.go:146: /login's stderr lacks "... (expires 09:05)\n"` |
| `TestDeviceCodeTextNamesExpiry` | no expiry, none named | the zero time printed (`expiry-printed-when-zero`) | `loginsurfaces_test.go:155: a code with no expiry: exit 0, stderr "... (expires 18:09)\n", want the words and no expiry` |

The stand-ins are shaped as production's: `opensBrowser` calls `o.OpenURL` only when the caller set one, as `provider.SignIn` gives the wizard an opener only then (`signin.go:326-328`), after telling the prompter the
URL; `showsDeviceCode` gives `o.ShowDeviceCode` a `provider.DeviceCode` with an `Expires` as `signin.go:318` builds it, or the default notice when none is set (`signin.go:319-322`). The opener is the test's
(`replace(t, &openURL, ...)`); no browser is opened and no port bound. `TestACPLoginOffersMenu`'s "the key" stand-in returns a fixed key after the menu's choice: what it holds is that the terminal path asks the
menu with no method forced, that `signInTo` stores what the sign-in gives (production code), and that the opener and the device text are passed; it does not hold the key's checking, which is the wizard's own (above),
nor its typing, since `terminalQuestions`'s raw mode needs a terminal and `editor.Raw` fails on a pipe: that gap is named under "Not verified".

**Tests changed, and why (none loosened).**

* `TestAuthLoginGrok` (`auth_test.go:~440`): its device assertion `... ABCD-1234\n` became `... ABCD-1234 (expires 12:00)\n`, with `clock` replaced by an instant in UTC. The stand-in's code expires at 13:00 at +01:00,
  which is 12:00 in UTC; without the clock the words depend on the machine's zone (it printed `07:00` here). The assertion is stricter.
* `TestConsentWordingOnEverySurface`, subtest "gobble acp --login" (`consentwording_test.go:210`): its answers `"3\n"` became `"2\n3\n"`. `acp --login openai` on a terminal now shows the menu first (the plan, "2"), then the
  consent's cancel ("3"). It asserts what it did: the wording asked before any URL, the sign-in declined, nothing saved. This test is 0052's (`c4c5ec0`), outside the brief's list of 14 login tests and `TestACPLogin`;
  with `"3\n"` alone it waited for an answer to the menu until the 10 minute bound.
* `TestACPLogin` and the 14 tests of `login_test.go`: unchanged, pass (`go test -count=1 ./internal/cli`, exit 0, and the race run below).
* Added: `TestACPLoginOffersMenu`, `TestLoginOpensBrowser`, `TestDeviceCodeTextNamesExpiry`, in `acp_test.go` and the new `loginsurfaces_test.go`.

**P6's plants that touch `internal/cli/login.go`, re-run** (P6's driver read; `llm/provider/signin.go` is unchanged, so its plants are not re-run): `terminal-host-id-not-set` (`TestCLIFillsHostID`) failed,
`agentauth_test.go:78: the terminal sign-in's options [...`, and `terminal-host-id-error-fails` (`TestSignInWithoutHostID`) failed, `agentauth_test.go:121: exit 1, stderr "Error: no host id: host-id: open <TMP>/...state-file/host-id: not a
directory; ...`; both unbroken, exit 0. P7 did not undo what they hold.

**What 0052-PLAN P4 expects of P7.** Its prerequisites are "P2, P3, 0051-PLAN P4 to P7", and step 4's `TestConsentWordingOnEverySurface` is the test of the five terminal surfaces (`/login`, `gobble setup`, the first run,
`gobble auth login`, `gobble acp --login`) that "covers" the consent's wording; that test exists (0052's `c4c5ec0`) and passes with `acp --login` through the menu. P7 delivers the terminal surfaces it rests on: `acp --login`
through `signIn`'s wizard path with the consent asked in gobble's wording, and the browser opened after it on all five. It does not deliver the ACP half (`TestConsentWordingOverACP`), which is P4 step 4's own, built on P6's `StartSignIn`.

**Gates (C6)**, after the last change to any Go file (the last write of a Go file is at 20:01:25, before the first gate started at 20:04:20; `make lint` is inside `make preflight`, the lint plan 0059 being complete: no six cold
runs). Each with its start and end time and its exit status taken before any filter. The two cross-host rows are the coordinator's addition to C6 for this phase (C6 names those hosts at P2, P4, P6 and P8; this phase changes how a browser is
opened and how a local time is printed, which differ by platform).

| Gate | Start | End | Exit |
| :--- | :--- | :--- | :--- |
| `make pre-add-check FILES=` the 7 Go files (gofmt, golint, govulncheck) | 20:04:20 | 20:04:30 | 0 |
| `make race` (both modules, `-race`), through `run-heavy` | 20:04:30 | 20:05:48 | 0 |
| `make check-test-isolation` | 20:05:48 | 20:07:12 | 0 |
| `make check-acpext-deps` | 20:07:13 | 20:07:13 | 0 |
| `go test -race -count=1 ./...` and acpext's, `run-on-host --race --need linux` (the coordinator's addition to C6): the Linux machine, not a WSL distro | 20:07:13 | 20:14:08 | 0 |
| `go test -race -count=1 ./...` and acpext's, `run-on-host --race --need windows` under `sh -c` (the coordinator's addition to C6): the Windows laptop, native | 20:14:08 | 20:17:39 | 0 |
| `make preflight`, through `run-heavy` (gofmt, tidy, pre-add-check, vet, staticcheck, one cold golangci-lint, govulncheck, check-acpext-deps, fix-check, archtest, apidiff, verify-build-metadata, check-records, check-icon, markdownlint) | 20:17:39 | 20:25:20 | 0 |
| `make check-records markdownlint`, after the last record edit | 20:25:32 | 20:25:34 | 0, and again after this row was filled |

**Docs (C9).** Rows:

* `docs/guides/credentials.md:192-196` (the Grok bullet) — fixed: the device code's line names its expiry in local time, and `/login`, `gobble setup` and the first-run offer open the browser and show the code as `auth login` does.
* `docs/guides/credentials.md:200` (the Kilo bullet) — fixed: the code is shown with its expiry as Grok's is.
* `docs/guides/credentials.md:~318` ("Signing in from a session") — fixed: a browser sign-in is opened in the browser; a device code is shown with its expiry.
* `docs/guides/credentials.md:~364` ("Signing in from an editor or magic-cli-remote") — fixed: `gobble acp --login` shows `/login`'s menu of methods on a terminal, `--device` or no terminal signs in with no menu, no key is asked over ACP itself.
* `docs/architecture.md:21` — fixed: `acp --login` shows `/login`'s menu.
* `README.md:45-47`, `README.md:119-122` — fixed: the same, and `/login` opens a browser sign-in and names a device code's expiry.
* `internal/cli/acp.go:23` (the `--login` help) and the doc comments of `acpLogin`, `signIn`, `showDeviceCode`, `terminalQuestions`, `onTerminal` — fixed; nothing exported was touched.
* `docs/guides/credentials.md:119` (`expires`, in UTC, RFC 3339: `auth status --json`'s row) and `:~533` (`--json`'s `show_code` step with `expires_at`) — not made false: both are machine output and stay in UTC/RFC 3339; the local time is the text form's.
* `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` (D4's note, above) — fixed: added.
* `docs/README.md`'s hooks row ("extend gobble (hooks, ...)") — left, as the brief says. `docs/README.md`'s 0051 PLAN row — fixed (P7 built and staged).
* open: none.
* docs: searched `README.md`, `AGENTS.md`, `docs/architecture.md`, `docs/README.md`, `docs/guides/*.md`, for `acp --login`, `/login`, `gobble setup`, `first run`, `first-run`, `expires`, `device code`, `browser`, `OpenURL`.

**A note added at MADR D4, outside P7's steps, on the coordinator's instruction.** One dated note (2026-11-16) in `0051-MADR`, at D4, directly after P6's note recording the owner's ruling "Keep the wizard's rule: a 401 refuses" (which stays as written). It is the coordinator's live verification, relayed, not a ruling of the owner's: Kilo's models listing with no organisation answers 200 whatever the Authorization header holds, so without an organisation the listing check cannot refuse a Kilo device key (and cannot detect a bad one), the case P6's note worried about does not arise there, and the open question against go-llmprovider-sdk is narrowed to whether the wizard should verify the key another way. The facts are the coordinator's and were not re-run here. `docs/guides/credentials.md` says nowhere that a Kilo device key is checked against the listing (searched for `kilo` with `list`, `check`, `refus`), so it is left. No code changed.

**Deviation (2026-11-16): the menu is not shown with `--device` or with no terminal.** MADR D3 says `acp --login` shows the wizard's menu "for every provider"; as built, with `--device` and with no terminal it signs in by the provider's default method with no menu (`acp.go:105`). This was built before it was put to the coordinator, who then recorded it as the coordinator's application of the plan's acceptance, not a ruling of the owner's, in a dated note at MADR D3's `acp --login` bullet. Reason: a menu cannot be asked without a terminal; `--device` names the method; `TestACPLogin` and the "no terminal" subtest of `TestConsentWordingOnEverySurface` have no terminal to ask on.

**The typed key's path through `signIn`.** An existing test holds it: `TestLoginSlash` (`login_test.go:226`) runs `/login claude` through the line session's `questions` (whose `raw` is the harness's), types a key at the secret prompt, and asserts the key is stored in `auth.json` (`:240`, `storedKey`) and never shown; `TestFirstRunSetup` (`:415`) does so from the provider menu's choice, and `TestLoginAsksThroughQuestions` (`:315`, `:351`) for Kilo. They use a stand-in for the wizard, as the new key subtest does. `acp --login` now reaches the same `signIn`, so only `terminalQuestions`' construction (three lines, `login.go:95`) is not exercised on a pipe.

**Not verified.** No real browser was opened and no live provider was signed in to: every opener and sign-in is the test's, and the menu is the real wizard over the test's plan issuer with every request refused but discovery.
A key typed through `acp --login`'s raw-mode `Secret` on a real terminal was not exercised: `editor.Raw` needs a terminal, so the key subtest's stand-in returns the key after the menu's choice (a seam for the raw mode in
`terminalQuestions` would be a name the plan does not give; not added). The expiry's wall-clock rendering was held for UTC, +02:00 and -05:00 clocks and the machine's own zone was not relied on; a zone with a half-hour or
daylight-saving offset was not tried. Whether Grok's or Kilo's real issuer sends `expires_in` was not checked (the SDK falls back to 300 s, `oauth_device.go:24`).

**Files (staged), with modification times** (the first gate started at 20:04:20): `internal/cli/acp.go` 20:01:25, `internal/cli/acp_test.go` 20:01:25, `internal/cli/auth.go` 19:51:52, `internal/cli/auth_test.go` 19:56:54,
`internal/cli/consentwording_test.go` 19:56:11, `internal/cli/login.go` 19:51:52, `internal/cli/loginsurfaces_test.go` 20:01:25; the text files, which only `check-records` and `markdownlint` read: `README.md` 19:59:10, `docs/README.md` 20:10:55, `docs/architecture.md` 19:59:10, `docs/guides/credentials.md` 19:59:10, `docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md` 20:10:35; this PLAN, written after the gates.

**Staged, not committed; the coordinator commits on the branch.**

### P8 record (2026-11-21)

Built by a mainstream model at medium effort, on the coordinator's brief; steps 1, 3 and 5 of P8, and step 4's text drafted. Step 2's probes were run by the owner and are recorded here, not
run by this phase. Times are the session machine's local time. Nothing staged or committed by the phase's checks; the records and docs below are edited in the working tree, **the owner commits**.
No tag, push, merge, issue close or branch change was made. No Go file changed.

**C1, the start state.** Branch `main`, `HEAD` `fa35986`, `git status --porcelain` printed nothing, before and after the checks. `go.mod` requires acpext `v0.5.0-rc.1` and
go-llmprovider-sdk `v1.31.0`.

**Step 2, the owner's live probes (recorded, not run here).** The kit is a scratch form-only ACP client, `acp-signin-client`, built from acp-go-sdk2 `v1.36.0`, at `<scratch>/live-0051`,
driving `gobble acp` built from `main` at `fa35986` (`gobble_version=0.6.0-56-gfa35986`). Each check ran in a fresh scratch `GOBBLE_HOME`, deleted at its end; the owner's own `auth.json` was never
read or written; the client sent `_meta.gobble.remote: true`; a port watch ran during each sign-in. The result files hold fixed words and numbers only. Read from them:

| Check | Mode and date | Outcome | Facts |
| :--- | :--- | :--- | :--- |
| L1, a grok device sign-in, form | `form`, 2026-11-21 06:57 | `signed-in` | `elicit_shown_form=1`, `elicit_shown_url=0`, `gobble_exit=0`, `client_exit=0`, `ports_bound=none` over `port_polls=351`; `gobble auth logout grok`: `logout_exit=0`, `logout_result=logged-out`, `revoke_warning=no` (C4 not triggered); `scratch_home_deleted=yes` |
| L2, the same, through a client offering `elicitation.url` | `url`, 2026-11-21 06:59 | `signed-in` | `elicit_shown_url=1`, `elicit_shown_form=0`, `url_complete=1`, `ports_bound=none` over `port_polls=74`; `logout_result=logged-out`, `revoke_warning=no`; `scratch_home_deleted=yes` |
| L3, U1 recorded | `sdk-probe`, 2026-11-21 07:43 | `passed` | the SDK's own `TestLive_0074GrokPaste` at go-llmprovider-sdk `v1.31.0`, commit `0fbb7613e9ac2e3420542a75b2f4ebffbe83854b`, from a scratch copy: `vet_exit=0`, `compile_exit=0`, `probe_exit=0`, `probe_test_result=PASS`, `probe_sign_in=completed-by-paste`, `probe_paste_shape=a bare code`, `probe_use=listed-12-models`, `probe_revocation=revoked-refresh-refused`, `probe_refusal_is_auth_failure=true` |

In both L1 and L2 `initialize=ok`, `device_method=found` and `authenticate=sent`, and `elicit_method_enum=0` and `elicit_declined_other=0`. L1 and L2 ran under a client with no request timeout, which is
what the MADR's "Not verified" first bullet says acceptance does; a client's own timeout is Magic's to measure.

**L3's first run failed, and why.** The first run ended `paste_refused_locally`. The coordinator's kit told the owner to paste a `127.0.0.1` address while xAI's page shows a code. The SDK's own 0074 record already said a bare
code; the error was the coordinator's, and the kit's wording was corrected before the second run, `L3.txt` as it stands. Grok's browser fact stays as the SDK has it, `browser_paste` at `v1.31.0` (the P4 record's deviation 4);
gobble changes nothing. No gobble code was involved in L3.

**Step 1, the commands, each run at `fa35986` with its exit status taken from the command itself before any filter** (output in scratch files; the decisive line quoted). Heavy runs ran one at a time, through `run-heavy` on this
Mac and `run-on-host --race` on the other two, which kept the Linux run off the WSL distros (`pick-host`: "excluded: race run; WSL distros report false data races").

| Command | Where | Start to end | Exit | Decisive line |
| :--- | :--- | :--- | :--- | :--- |
| `make archtest` | this Mac | 12:53:55 to 12:53:57 | 0 | `ok  	github.com/maccavelli/gobble-cli/internal/archtest` |
| `make check-acpext-deps` | this Mac | 12:53:57 | 0 | `check-acpext-deps: ok` |
| `python3 scripts/check_libs.py --self-test` | this Mac | 12:53:57 to 12:53:58 | 0 | `self-test a next link off api.github.com is refused: ok` (the last of the cases) |
| `make check-libs` (network) | this Mac | 12:53:58 to 13:02:38 | 0 | `0 failing checks; scratch removed`; each of acp-go-sdk2 `v1.36.0`, go-llmprovider-sdk `v1.31.0`, authcontract, go-selfupdate-lib and go-tui-lib `v2.12.1` ends `advisories: ok` |
| `go -C acpext test -count=1 ./...` | this Mac | 13:02:38 | 0 | `ok  	github.com/maccavelli/gobble-cli/acpext` |
| the verification's long `-run` pattern (`TestOffered\|TestPredicates\|TestEveryCommand\|TestResolverAccepts\|TestLogoutRevokes\|TestErrorKind\|TestKindsAndLimits\|TestLoginWithReadsTheDefaultMethod\|TestErrorData\|TestPromptQuotaData\|TestStartSignIn`) over `./llm/... ./internal/... ./acpserver/` | this Mac | 13:02:39 to 13:03:40 | 0 | `ok  	github.com/maccavelli/gobble-cli/llm/provider	59.669s`, `internal/cli`, `acpserver` ok |
| `-run 'TestAuthLoginJSON' ./internal/cli/` | this Mac | 13:03:40 to 13:04:09 | 0 | `ok  	github.com/maccavelli/gobble-cli/internal/cli	29.015s` |
| `-run 'TestRemoteMethods\|TestStartSignInRemote' ./llm/provider/` | this Mac | 13:04:09 | 0 | `ok` |
| `-run 'TestRetryCausesAreAcpexts\|TestRetryNotice' ./acpserver/`; `-run 'TestRetryLines' ./internal/cli/` | this Mac | 13:04:11 to 13:04:13 | 0, 0 | `ok`, `ok` |
| the D9 tests: `TestSignInNamesGobble\|TestStoredLoginNamedForRefresh\|TestLogoutRevokesAsGobble\|TestStartSignInNamesGobble` (`./llm/provider/`), `TestCLIPassesGobblesIdentity` (`./internal/cli/`) | this Mac | 13:04:13 to 13:04:17 | 0, 0 | `ok`, `ok` |
| the P6 step 1a tests: `TestClientStoreAdapter\|TestHostIDReachesTheEngines` (`./llm/provider/`), `TestCLIFillsHostID\|TestSignInWithoutHostID` (`./internal/cli/`) | this Mac | 13:04:17 to 13:04:20 | 0, 0 | `ok`, `ok` |
| `go test -race -count=20 -run 'TestAuthenticate\|TestAuthMethods\|TestNoACPLogout' ./acpserver/`, `run-heavy` | this Mac | 13:04:20 to 13:12:41 | 0 | `ok  	github.com/maccavelli/gobble-cli/acpserver	494.758s` |
| `make preflight`, `run-heavy` | this Mac | 13:12:41 to 13:13:12 | 0 | `preflight passed`; `golangci-lint` `0 issues.` for both modules, `govulncheck` `No vulnerabilities found.` for both, `apidiff: no-op below v1.0.0`, `markdownlint` `0 issues in 0 files` |
| `make race` (both modules, `-race`), `run-heavy` | this Mac | 13:13:12 to 13:14:41 | 0 | 39 `ok` lines; `ok  	github.com/maccavelli/gobble-cli/acpext	(cached)` |
| `make check-test-isolation`, `run-heavy` | this Mac | 13:14:42 to 13:16:05 | 0 | `check-test-isolation: ok` (`files written above the temp roots: 0`) |
| `go test -count=1 ./...`, `CGO_ENABLED=1 go test -count=1 -race ./...`, `go -C acpext test -count=1 ./...`, `run-on-host --need linux --race`, `sh -c` | the Linux machine (linux/amd64), scratch copy removed after | 13:16:05 to 13:22:58 | 0 | 77 `ok` lines, no `FAIL` |
| the same three, `run-on-host --need windows --race`, `sh -c` | the Windows test laptop (windows/amd64), scratch copy | 13:22:58 to 13:27:57 | 0 | 77 `ok` lines, no `FAIL` |
| `-v -run 'TestEveryCommand\|TestOffered\|TestPredicates\|TestResolverAccepts\|TestRemoteMethods' ./llm/provider/ ./internal/cli/` | this Mac | after the gates | 0 | `--- PASS: TestEveryCommandAndWordIsAccepted`, `TestOfferedMethods`, `TestPredicatesFromOffered`, `TestRemoteMethods`, `TestOfferedConsentTextIsGobbles`, `TestOfferedMethodByWord` |
| apidiff of acpext from `acpext/v0.4.0`: the root's `golang.org/x/exp/cmd/apidiff` built to scratch, export data of `git archive acpext/v0.4.0` and of the tree's `acpext/` | this Mac | after the gates | build, archive, both exports and the diff all 0 | the list below |

`make preflight`'s `pre-add-check` step printed "golangci-lint: skipped (GO_PRECHECK_SKIP_LINT=1)" and "govulncheck: skipped (GO_PRECHECK_SKIP_VULN=1)": the Makefile sets both because preflight runs
`golangci-lint` and `govulncheck` as their own steps, and each of those printed its result above. `make preflight` took 31 s, not the 8 minutes of earlier records, because the build and lint caches were warm;
its output lists every step. `go test -count=1`'s own runs are uncached; `make race` shows acpext `(cached)`, as `make race` does not pass `-count=1`, and the Linux and Windows runs and the 20-count run above did.

**apidiff from `acpext/v0.4.0`** (exit 0), the P3 record's list exactly:

* Incompatible: `KindContextOverflow`, `KindEmptyReply`, `KindNotPermitted`, `KindRateLimited`, `RetryEmptyReply`, `RetryNotPermitted`, `RetryRateLimited`: value changed; `RetryEmpty`: removed.
* Compatible: `AuthenticateMeta`, `ErrorData.Consent`, `KindDenied`, `KindExpired`, `KindInvalidProvider`, `KindInvalidRequest`, `MetaMethods`, `MetaStep`: added.

**Confirmation, item by item** (the MADR's "Completion (2026-11-21)" carries the same cells; the plants are the earlier records', not run again here, and no new test or gate was added by this phase):

| Item | Verdict | Evidence |
| :--- | :--- | :--- |
| 1. archtest refuses `authcontract` in a core package and in `acpclient` | met | `make archtest` exit 0; the P2 record's plants `rule 1 agent authcontract` and `rule 4 acpclient authcontract` (and `rule 4 acpserver signin`), each seen failing |
| 2. `provider.Offered`, the predicates, the word table | met | `TestOfferedMethods`, `TestPredicatesFromOffered`, `TestRemoteMethods`, `TestResolverAcceptsOnlyOfferedKinds` pass; the P4 record's plants |
| 3. loopback ACP tests, `fresh`, `remote` | met | `TestAuthenticate*` pass in the three full suites; `-race -count=20` exit 0; the P6 record's 73 plants; L1 and L2 live, `ports_bound=none` |
| 4. `auth login --json` | met | `TestAuthLoginJSON*` pass (exit 0); the P5 record's 39 plants |
| 5. no `FailureKind`, every kind an `authcontract.ErrorKind` | met | `grep -rn FailureKind --include=*.go .` matches two test names only (`TestAuthLoginJSONFailureKind`, `TestAuthenticateRealFailureKinds`), no type; `TestErrorKind`, `TestKindsAndLimits`, `TestErrorData` pass |
| 6. every acpext `Command` and offered word accepted by `gobble auth login` (issue #16's criterion) | met | `TestEveryCommandAndWordIsAccepted` passes; the P4 record's plants (openai given `device`, kilo given `--method browser`) |
| 7. `make preflight`, `make race` on three hosts; `make check-acpext-deps`; apidiff | met on the tree | the rows above; P8a repeats the gates and apidiff on the tagged commit, as the item's note says |
| 8. the owner's live probes recorded | met | L1, L2, L3 above |
| 9. (D9) gobble's name on every auth request | met | `TestSignInNamesGobble`, `TestStoredLoginNamedForRefresh`, `TestLogoutRevokesAsGobble`, `TestStartSignInNamesGobble`, `TestCLIPassesGobblesIdentity` pass; live, `revoke_warning=no` after L1 and L2 |

Items 1 to 8 are met. None is unmet.

**Step 5, the MADR's Completion.** "Completion (2026-11-21)" is added to the MADR after "Confirmation". It states that this PLAN stays `in-progress`. **This PLAN is not `complete`**: P8a is still to come, the owner's
`acpext/v0.5.0` tag and then one pin commit that moves the root from `v0.5.0-rc.1` to `v0.5.0`, after which this PLAN is set `complete` with P8a's record. The MADR's U1 bullet gains a dated note on L3.

**Step 4, issue #16, the owner's to run** (MADR D8's text, verbatim; the single quotes inside it are escaped for `sh`). Not run by this phase:

```sh
gh issue close 16 --repo maccavelli/gobble-cli --comment 'Closing as superseded by docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md,
from magic-cli-remote'\''s docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md.

What carries forward: clients get each provider'\''s methods as data, how each completes, what it yields, whose
client it signs in as and the consents it asks, as go-llmprovider-sdk'\''s authcontract.Method in each ACP agent
auth method'\''s _meta.gobble.methods, and gobble auth login --json'\''s first line. Not as
AuthProvider.Methods []string, which would make AuthProvider incomparable, and not as --method words alone.
A test asserts every acpext Command and every method gobble offers is accepted by gobble auth login.
acpext'\''s Login stays, documented as "has a method other than an API key", not "device code"; acpext/v0.5.0
carries the doc. The remote paste is answered by ACP elicitation inside authenticate, not by a --json
answer channel.'
```

**Docs (C9).** Rows:

* `docs/README.md:119` (0051 MADR row) — fixed: the 2026-11-21 acceptance, Confirmation 1 to 9 met and the Completion section.
* `docs/README.md:120` (0051 PLAN row) — fixed: "probes L1 to L3 not run" is struck and the 2026-11-21 state added (P8 run at `fa35986`, L1 to L3 passed, P8a still to come, issue #16 still open for the owner). *(2026-11-21, later: the owner closed #16, `CLOSED COMPLETED`; "P8a record (2026-11-21)".)*
* `README.md:46-47` (the `gobble acp --login` bullet, the sign-in section) — fixed: a remote client's browser sign-in for Grok completes by a pasted code or address, as the credentials guide says.
* `docs/architecture.md:198` (the acpext row of the version table) — open: stays `v0.5.0-rc.1`, true until P8a's pin commit, which changes it (P8a step 3). *(2026-11-21, later: fixed by P8a's pin commit, the row reads `v0.5.0`.)*
* `docs/architecture.md:17-23`, `:97-98`, `:128`, `:133` (sign-in rows) — checked, true now: no change.
* docs: searched `README.md`, `docs/architecture.md`, `docs/README.md` and `docs/guides/credentials.md` for `sign-in`, `authenticate`, `elicit`, `paste`, `127.0.0.1`, `bare code`, `url_or_code`, `acpext/v0.5.0`, `0051`.

**Not verified.** The final tag's gates and apidiff, P8a's. Magic's own client and its request timeout. No real console of the owner's was used; no token, code, address, account or e-mail was read: only
the three result files under `<scratch>/live-0051` were opened.

**Follow-ups (outside the records' cases).**

* The MADR's Consequences bullet of 2026-11-11, "a remote client cannot sign in to grok by browser: only its device method", is still as written; the line-882 annotation and the U1 note give the SDK's later fact. A dated note at that bullet is a one-line
  follow-up for the owner's next records pass.
* `make preflight`'s pre-add step reports lint and vulnerability skips by design; nothing to do.

**Gates after the last record edit (C6).** `make check-records markdownlint`: see the row filled below.

| Gate | Start | End | Exit |
| :--- | :--- | :--- | :--- |
| `make check-records markdownlint`, after the last record edit | 13:30:12 | 13:30:14 | 0 ("Summary: 0 issues in 0 files"; `check_records.py --check-all` printed nothing), and again after this row was filled |

### P8a record (2026-11-21)

Built by a mainstream model at medium effort, on the coordinator's message after the owner committed the P8 record as `4de3c45`, pushed it, tagged `acpext/v0.5.0` and closed #16. Steps 2 and 3 and the C6 gates are this phase's;
step 1 is the owner's, recorded below. Times are the session machine's local time. Edited in the working tree: `go.mod`, `go.sum`, `docs/architecture.md` and the records; **the owner commits**. No Go file changed. No tag, push, merge or
issue close was made by this phase.

**Start state.** Branch `main`, `HEAD` `4de3c45` (the P8 record), level with the remote, `git status --porcelain` printed nothing.

**Step 1, the tag (recorded, the owner's).** `acpext/v0.5.0` is an annotated tag (`git cat-file -t acpext/v0.5.0` prints `tag`) whose commit is `4de3c45` (`git rev-parse --short 'acpext/v0.5.0^{commit}'`). `git diff --quiet acpext/v0.5.0-rc.1 4de3c45 -- acpext/` exits 0,
so the tag names the same `acpext/` as the candidate. `go list -m github.com/maccavelli/gobble-cli/acpext@v0.5.0` prints `github.com/maccavelli/gobble-cli/acpext v0.5.0` through the proxy, exit 0. Issue #16 is `CLOSED COMPLETED` (the owner closed it with MADR D8's text).

**Step 2, apidiff from `acpext/v0.4.0` to `acpext/v0.5.0`** (exit 0): the root's `golang.org/x/exp/cmd/apidiff` built to scratch, export data of `git archive acpext/v0.4.0` and of `git archive acpext/v0.5.0`, each loaded from its own extracted `acpext/`.
The build, both archives, both exports and the diff each exit 0. The lists are P3 step 5's and the P8 record's, with nothing added and nothing removed, so there is no addition and no deviation to name:

* Incompatible: `KindContextOverflow`, `KindEmptyReply`, `KindNotPermitted`, `KindRateLimited`, `RetryEmptyReply`, `RetryNotPermitted`, `RetryRateLimited`: value changed; `RetryEmpty`: removed.
* Compatible: `AuthenticateMeta`, `ErrorData.Consent`, `KindDenied`, `KindExpired`, `KindInvalidProvider`, `KindInvalidRequest`, `MetaMethods`, `MetaStep`: added.

**Step 3, the pin commit's change (working tree).** `go get github.com/maccavelli/gobble-cli/acpext@v0.5.0` exit 0, `go mod tidy` exit 0. `git diff go.mod go.sum`: `go.mod` `acpext v0.5.0-rc.1` becomes `v0.5.0`; `go.sum` drops the two `v0.5.0-rc.1` lines and adds
`acpext v0.5.0 h1:gwrAqNx5EK4CfULMcPph2yFnVQUM7t6yw9ehtRwnBr4=` and its `/go.mod` line (`h1:yNoZruhuuNrybUQHCPtIZfRKGUDc7Hho/jHGqRFE6Cs=`, the same as the candidate's, since `acpext/go.mod` did not change). Nothing else in either file moved.
`git status --porcelain` before the records: `go.mod`, `go.sum`, `docs/architecture.md` only; no Go file.

**Gates (C6) on the changed tree**, each exit status taken from the command's own run before any filter:

| Gate | Where | Start to end | Exit | Decisive line |
| :--- | :--- | :--- | :--- | :--- |
| `grep -rn 'v0.5.0-rc.1' go.mod` | this Mac | 13:38:11 | 1 (no match, as required) | no output |
| `make check-acpext-deps` | this Mac | 13:38:11 to 13:38:12 | 0 | `check-acpext-deps: ok` |
| `make preflight`, `run-heavy` | this Mac | 13:38:12 to 13:39:08 | 0 | `preflight passed` |
| `make race`, `run-heavy` | this Mac | 13:39:08 to 13:40:28 | 0 | 39 `ok` lines |
| `make check-test-isolation`, `run-heavy` | this Mac | 13:40:28 to 13:42:22 | 0 | `check-test-isolation: ok` |
| `go test -count=1 ./...`, `CGO_ENABLED=1 go test -count=1 -race ./...`, `go -C acpext test -count=1 ./...`, `run-on-host --need linux --race`, `sh -c` | the Linux machine (not a WSL distro), scratch copy | 13:42:22 to 13:49:17 | 0 | 77 `ok` lines, no `FAIL` |
| the same three, `run-on-host --need windows --race`, `sh -c` | the Windows test laptop, scratch copy | 13:49:17 to 13:54:01 | 0 | 77 `ok` lines, no `FAIL` |
| `make check-libs` (network) | this Mac | 13:54:01 to 14:02:17 | 0 | `0 failing checks; scratch removed`, five libraries each `advisories: ok` |

**Documents that state what the root requires.** Searched `go.mod`, `go.sum`, `README.md`, `docs/architecture.md`, `docs/README.md`, `docs/guides/`, `.github/` for `v0.5.0-rc.1`: `docs/architecture.md:198` — fixed, `v0.5.0`;
`docs/architecture.md:97` — fixed: "first tagged as `acpext/v0.5.0-rc.1`, then `acpext/v0.5.0`" (history, not a requirement); the rest of `docs/decisions/` names the candidate as history and is not changed, except by the dated annotations here and in the
P8 record and the MADR. `go.mod` and `go.sum` hold no `v0.5.0-rc.1`.

**Acceptance (P8a).**

* the tag served: met (step 1; `go list -m ...@v0.5.0` exit 0).
* the root requires `v0.5.0`: met (`go.mod`).
* every C6 gate green: met (the table).
* no `v0.5.0-rc.1` left in `go.mod` or in a document that states what the root requires: met (the grep exit 1; `docs/architecture.md` fixed).

All four met. **This PLAN is set `complete`** (front matter `status: complete`, `date: 2026-11-21`; the repository has no `tools/records.py`, so by hand), and `docs/README.md`'s 0051 PLAN row reads `complete 2026-11-21`.
The MADR's Consequences bullet of 2026-11-11 on Grok's browser carries a one-line dated note (a records-only fact), and the P8 record's "still open" cells are annotated: #16 is closed.

**Not verified.** A gobble release (`v0.7.0`) is outside this PLAN. Magic's import of `acpext/v0.5.0` is that repository's.

**Gates after the last record edit (C6).** `make check-records markdownlint`: see the row below.

| Gate | Start | End | Exit |
| :--- | :--- | :--- | :--- |
| `make check-records markdownlint`, after the last record edit | 14:03:01 | 14:03:03 | 0 ("Summary: 0 issues in 0 files"), and again after this row was filled |

### Note (2026-11-23): v0.7.0 released

*(Dated addition, records only; the PLAN's status `complete` is unchanged. It closes the item this PLAN left outside
itself: the gobble release `v0.7.0` and the owner's tag.)* The release is 0055-PLAN's; its full note is that PLAN's
"Note (2026-11-23): v0.7.0 released".

* **The tag.** `v0.7.0` is an annotated tag, made by the owner on `861ba39` and pushed 2026-11-23. Tag CI run
  `45196405850` concluded `success`.
* **The Release.** <https://github.com/maccavelli/gobble-cli/releases/tag/v0.7.0>, published 2026-11-23T15:00:08Z by the
  release workflow, not a draft, with seven assets: `gobble-darwin-arm64`, `gobble-linux-amd64`, `gobble-linux-arm64`,
  `gobble-windows-amd64.exe`, `gobble-windows-arm64.exe`, `gobble.spdx.json` and `SHA256SUMS`.
* **The proxy.** `go list -m github.com/maccavelli/gobble-cli@v0.7.0` through proxy.golang.org resolves, `Version v0.7.0`.
* **Attestation.** `gh attestation verify gobble-darwin-arm64 --repo maccavelli/gobble-cli --signer-repo
  maccavelli/go-selfupdate-lib`: exit 0 (verified).
* **The pins at the tag** (`git show v0.7.0:go.mod`): acp-go-sdk2 `v1.36.0`, go-llmprovider-sdk `v1.31.0`
  (authcontract `v0.1.0`) and acpext `v0.5.0`.
* **developing.md's "After each `v*` release" checks,** run read-only on 2026-11-23:
  * The rule-suite query (`gh api 'repos/maccavelli/gobble-cli/rulesets/rule-suites?ref=refs/tags/v0.7.0&time_period=day'
    --jq '[.[] | select(.result == "bypass")] | length'`) printed `0`, exit 0. The tag push's own output was the
    owner's and is not read here.
  * `gh release view v0.7.0 --json isImmutable` printed `{"isImmutable":true}`, exit 0.
  * `gobble update --check` from the previous release: `v0.6.0`'s `gobble-darwin-arm64` was downloaded to a new scratch
    directory, its checksum matched `SHA256SUMS` (`OK`), and a copy was run from under a scratch home directory (the
    updater accepts only a binary under home) with the home and config directories set to scratch. It exit 10 and
    printed `gobble: update available: v0.6.0 -> v0.7.0`, with `selected ... target=v0.7.0 asset=gobble-darwin-arm64`.
  * **Not run:** the Windows sequence (`gobble acp` held open, `gobble update --yes`, then a plain `gobble update` that
    must exit 0 and leave no `.gobble.exe.selfupdate-*` file). No Windows host was used for this note. It is open, not
    met.
* **Next step, not done.** magic-cli-remote's G2 step (its 0246-PLAN: recapture its fixtures against `v0.7.0` and
  move 0246 out of `proposed`) is that repository's, on the owner's word. This note does not do it.
