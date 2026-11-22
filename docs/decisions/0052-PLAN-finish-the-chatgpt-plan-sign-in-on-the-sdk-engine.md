---
status: in-progress
date: 2026-11-20
associated-madr: "0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Implement finishing the ChatGPT plan sign-in on the SDK engine

Associated MADR: [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)

## Goal

The owner's uncommitted 0031-PLAN P4b-2 work is sorted as MADR D1's table says, nothing of it lost; the ChatGPT plan
sign-in runs on go-llmprovider-sdk's engine with gobble's wording, classification and gating (D2), a command for the
consent (D3), a host-lifetime host id (D4), a reused issued client (D5), the session's kind and consents in `auth.json`
(D6), every finding of D7 fixed, `auth status --json`'s two new members (D8), and the one-time migration (D9). The plan
is complete when the MADR's Confirmation 1 to 8 is met.

This record is `proposed`. Nothing below starts before the owner accepts the MADR and approves this PLAN, whole or by
phase. The owner answered QA, QF and QG on 2026-11-10, each as recommended: QA (a), QF (a) and QG (a). P1 changes the owner's uncommitted work, and starts only on the owner's explicit word for
it.
*(2026-11-11: after Deviation 1 there is no uncommitted work for P1 to change. P1 re-applies parts of `1406bd6` to `main`, and still starts only on the owner's word.)*
*(2026-11-11, later: the owner accepted the MADR that day, and this PLAN is `in-progress`. The owner's words, "accept
and approve 0051 and 0052 when required", were confirmed to the cross-repository coordinator as: start the phases
that do not wait on the SDK. P0 is done (its record, under "Execution record"); it starts no other phase.)*

## Scope

**Base.** ~~`main` at `d6c10a5`, with the uncommitted work of 39 files (`git diff --stat`: +1010/−84).~~

*(2026-11-11, restated after Deviation 1.)* `main` at `17d1605`, level with `origin/main`; its tree holds none of that
work. The work is one commit, `1406bd6`, on the branch `wip/0031-p4b2-chatgpt-plan`, whose parent is `d6c10a5`: 39
files, +1010 and -84 lines (`git diff --stat d6c10a5 1406bd6`). The branch is also on origin: it was pushed on
2026-11-11, and `git ls-remote --heads origin` lists it at `1406bd6`. The tree holds six untracked records and nothing
else: this pair, the 0051 pair and the 0054 pair. `0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md` lands before
0051-PLAN's P2, so every phase here after P1 starts from `main` as that plan leaves it.

**In scope:**

* the 39 files of the uncommitted work, sorted (P1);
* `internal/auth/store.go`, `logins.go`, `clients.go` (new), `hostid.go` (new), `migrate.go` (new), and their tests;
* `llm/llm.go` or `llm/background.go` (new): `WithBackground`, `IsBackground`;
* `llm/provider/login.go`, `signin.go`, `provider.go` (the background mark, the plan session), `consent.go` (new), and
  their tests;
* `acpserver/agent.go`, `config.go`, `title.go`, `autocompact.go`, and their tests;
* `internal/cli/auth.go` (`auth consent`, the status members), `login.go`, `agent.go`, `credentials.go`, `main.go` (the
  migration's call), their tests, `internal/cli/testdata/auth-status.json.golden`;
* docs: `docs/guides/credentials.md` (the ChatGPT section, the consent, the files), `docs/architecture.md`, `README.md`,
  `docs/README.md`;
* records: this pair; the amendments the MADR lists (P0).
* *(2026-11-11, later; MADR D2's, D3's and D8's notes of that date.)* `llm/consent.go` (new: `ConsentError`), with its
  test; `acpserver/prompt.go`'s `errorData` (`ErrorData.Consent`, from acpext `v0.5.0`, 0051-PLAN P3) and
  `acpserver/errordata_test.go`; `llm/provider/policy.go`'s consent text (0051-PLAN P4); `consentTerms` in
  `internal/cli/auth.go`'s status row.

**Out of scope:** 0051's surfaces (its PLAN); go-llmprovider-sdk; magic-cli-remote; a release tag or a push without the
owner's word; reading the owner's real `auth.json`, `settings.yaml` or state directory (every test and probe uses a
scratch `GOBBLE_HOME`).

## Conventions every phase follows

C1 to C8 of [0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md),
and:

* **C9. The uncommitted work is never discarded by a git command.** No `git checkout --`, `restore`, `reset`, `stash`
  or `clean` on it. A hunk leaves the tree by the inverse edit, after P1's patch is written and checked.
* **C10. Real-shaped files only as copies.** The migration's tests build files of the owner's shapes in a temporary
  directory; no test or probe reads the owner's.

## Implementation Steps

### P0 — the records (on acceptance)

1. The MADR `accepted` with the owner's answers to QA, QF and QG, dated; this PLAN `in-progress`.
2. The owner pastes the 0031 amendment text (the scratch file `gobble-0031-amendment.md`, written with this pair) into
   `0031-MADR-sdk-current-releases-and-adoption.md` and `0031-PLAN-sdk-current-releases-and-adoption.md`. They are the
   owner's working files today, so the agent does not edit them.
3. Dated notes, each quoting what it amends: `0025-MADR-provider-choice-and-auth.md` D6; `0045-MADR-auth-status-json.md`
   D2 and D6; `0005-MADR-v1-feature-scope.md`'s OpenAI row (`:168`); and, under QA (a),
   `0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md` P5b split into P5b-1 (the `llm` half) and P5b-2
   (the wire half, with 0051-PLAN P4's acpext pin, its step 1).
4. `docs/README.md`: the 0052 MADR and PLAN rows.
5. `make check-records markdownlint`: exit 0.

Acceptance: every note quotes what it amends; the 0031 text is the owner's paste.

#### Deviation 1 (2026-11-11): the uncommitted work is archived on a branch and cleared from the tree now, before acceptance

**Found** by the owner: "Assess the dirt in the gobble-cli tree. It is chaos. I think most of it can be reverted,
unstaged, and deleted as superseded." The assessment, read at `d6c10a5` with the 39 files, sorted them as MADR D1's
table does:

* **0049-PLAN P5b's `llm` half,** to land now (QA (a)): `llm/errors.go`, `llm/provider/limit_test.go` and
  `acpserver/choice_test.go`, all three in whole; and `llm/provider/provider.go`'s `mapErrorAt`, `limitOf` and
  `planWindow` hunks. It builds on `HEAD`'s `llm/kind.go:14-17` constants and go-llmprovider-sdk v1.29.0's `Limit`
  (`llmprovider/api_limit.go:20-34`, `api_error.go:131`), with acpext unchanged.
* **The 0031 MADR and PLAN text,** to keep, with the paste of `gobble-0031-amendment.md`.
* **The other 33 files,** whose parts are "kept as is", "reworked" or "discarded" (MADR D1). They include the wire
  half, `acpserver/prompt.go`, `acpserver/errordata_test.go`, and `go.mod`/`go.sum`'s acpext v0.4.0.

**Decision (owner, 2026-11-11): "Proceed with the gobble cleanup",** the five steps as proposed:

1. Archive the whole uncommitted work on a local, unpushed branch, `wip/0031-p4b2-chatgpt-plan`, and as a patch.
   *(2026-11-11, later the same day: the branch was pushed; `git ls-remote --heads origin` lists it at `1406bd6`.)*
2. Land the `llm` half on `main`, recorded as 0049-PLAN P5b-1.
3. Land the 0031 text and the amendment paste on `main`.
4. Clear the other 33 files from the tree.
5. Amend this PLAN and 0051-PLAN P4 to take their parts from the archive.

**What changes here:**

* **C9 is amended.** The work is first committed whole to the archive branch, and kept as a patch whose SHA-256 is
  recorded *(2026-11-11: in the note under P1's heading; until then it was only in
  `0031-PLAN-sdk-current-releases-and-adoption.md:416-417`)*. Leaving the archive branch for `main` then takes the 33 files out of the tree. No hunk is lost: every one
  stays on the branch and in the patch. The inverse-edit rule stands for any work not archived.
* **P1 step 1** is done by this deviation. The patch is written, reverse-applies, and is scanned for identifiers and
  credentials.
* **P1 step 2 is changed.** The "kept as is" parts (the four plan fields in `internal/auth`, `provider.Login`'s fields,
  `MethodChatGPTPlan`, `AgentName` and their tests) leave the tree with the rest. P1 re-applies them from the archive
  branch, with `git show wip/0031-p4b2-chatgpt-plan:<path>` as their source. The P5b-1 bullet is done by this
  deviation, under 0049-PLAN.
* **P1 steps 3 to 5** are done by the clearing. Their "kept in the patch" is the archive branch and the patch.
* **The wire half** (step 5) is re-applied by 0051-PLAN P4 from the archive branch, with acpext v0.5.0.
* **P0 step 2** (the owner's 0031 paste) is done by this deviation's step 3, on the owner's word.

Files added to this phase: none.

### P1 — sort the uncommitted work (MADR D1), and what needs no SDK

**Read against Deviation 1 (note of 2026-11-11).** Steps 1 to 5 and step 8 below were written before Deviation 1 and
are kept as written. They no longer describe what happens:

* **Step 1 is done, by other means.** The work was committed whole as `1406bd6` on `wip/0031-p4b2-chatgpt-plan`, and
  written as a patch, before `main`'s tree was cleared. `git apply --check --reverse` cannot pass now, since the tree
  holds none of the work. The patch is `git diff --binary d6c10a5 1406bd6`: 90,118 bytes, SHA-256
  `8b89cbb7dc2da2d3caddb5f8148eede5084c668247ba510a0ce3cd8c08b71063`, the hash
  `0031-PLAN-sdk-current-releases-and-adoption.md:416-417` records. Run again on 2026-11-11 against the branch, that
  command gave the same size and the same hash. **The durable copy of the work is the branch at `1406bd6`, which
  origin also holds since that day's push.** The patch file can be made again from it at any time, and no path to a
  copy of it is recorded here.
* **Step 2.** Its last bullet, 0049-PLAN P5b-1, is on `main` as `97f13a5`. Its first three bullets are not "left in
  the tree": they left it with the rest, and P1 re-applies them from the branch (Deviation 1).
* **Steps 3 to 5** were done by the clearing. "Kept in the patch" reads "kept on the branch".
* **Step 7's `TestSortedTreeIsThePatch`** compares `main`, with the re-applied parts, against `1406bd6`, file by file,
  in place of "`HEAD` plus the patch".
* **Step 8's two commits** are one now; its note says why.

What P1 still has to do: re-apply step 2's first three bullets from the branch, then steps 6 to 8.
*(2026-11-11, later: done, staged and not committed; see "P1 record (2026-11-11)" at the end of "Execution record".
`TestSortedTreeIsThePatch` compares at the level of `git diff -U0` change groups, 99 of them, not whole hunks.)*

1. **Keep a copy first.** `git diff --binary` written to `<scratch>/0052-wip.patch`, `git diff --stat` beside it, and
   ~~the patch's SHA-256 recorded here~~ *(2026-11-11: it was recorded only in 0031-PLAN until the note above)*.
   `git apply --check --reverse <scratch>/0052-wip.patch` passes on the tree *(2026-11-11: it cannot now)*, which
   proves the patch is the tree's. The owner keeps the patch where they choose; it holds no credential (the identifier
   scan and a scan for `sk-`, `eyJ` and `refresh_token` values run on it first).
2. **Leave in the tree** (MADR D1, "kept as is"):
   * the four plan fields in `internal/auth/store.go` and `logins.go`, and `TestStoreOAuthPlanFields`;
   * the four fields in `provider.Login`, `session` and `loginOf`, and the plan half of `TestLoginStoreAdapter`;
   * `MethodChatGPTPlan`, its `methodIDs` entry and `AgentName`, inert until P4: `offers` at [HEAD] skips
     `AuthChatGPTPlan`;
   * under QA (a), 0049-PLAN P5b-1: `llm/errors.go`'s `Window`, `provider.go`'s `mapErrorAt` reset rule, `limitOf` and
     `planWindow`, `limit_test.go`'s five tests, `choice_test.go`'s case.
3. **Take out by the inverse edit, kept in the patch** for P3 and P4 to re-apply reworked: `HasLogin`'s openai,
   `HasPiLogin`, `Plan()`, `offers`' guards, `provider.go:107-112`, `loginWith`'s branch and help text, the status row's
   openai case, `PlanSession` and the gating hunks of `acpserver`, `signInWords`' case, `piimport.go:45`, and the WIP
   tests of those (`TestAuthLoginChatGPTPlan`, `TestAuthStatusOpenAINotAPlan`, `TestACPLoginOpenAI`,
   `TestPlanSessionOption`, `TestOpenAILoginIsAPlan`, `TestLogoutPlan`, `TestSignInChatGPTPlan`,
   `TestSignInChatGPTPlanPasted`, `TestTitleUnderPlan`, `TestAutoCompactUnderPlan`, and the edits to
   `TestAgentAuthOptions`, `TestProviderChoicesStates`, `TestProvidersJSON`, `TestAuthLogin`).
4. **Take out for good** (MADR D1, "discarded"): `planNotice`, `askPlanBackground`, `planBackgroundKey`, the `notice`
   event, and the `chatgptPlanBackground` setting in `internal/config` (`keys.go`, `settings.go`, `settings.yaml`,
   `testdata/settings.schema.json`, `render_test.go`, `resolve_test.go`) and `internal/cli/settings.go`.
5. **Under QA (a), the wire half waits:** `acpserver/prompt.go:308`'s `WindowMinutes`, the `weeklyErr` case and frame of
   `acpserver/errordata_test.go`, and `go.mod`/`go.sum`'s `acpext/v0.4.0` leave the tree with the inverse edit, for
   0051-PLAN P4's pin. The 0031 MADR and PLAN text stays (P0 step 2).
6. **F25, new.** `oauthFields` (`internal/auth/store.go:251-277` [WIP]) adds `key` and `workspace` to `remove`.
7. Tests, each seen failing (C2):

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestSetOAuthDropsKey` (`internal/auth`) | an `api_key` entry with `key` and `workspace`, then `SetOAuth`: neither remains, every other field kept | `key` left out of `remove` |
   | `TestStoreOAuthPlanFields` (kept) | as written | — (seen failing at [HEAD], where the fields do not exist) |
   | `TestSortedTreeIsThePatch` (a scratch script, not committed) | the tree plus the hunks taken out equals `HEAD` plus the patch, file by file | one hunk dropped from the patch |

8. Gates (C6). ~~Under QA (a) the owner commits two commits: 0049-PLAN P5b-1, then this phase's storage half and F25.~~
   *(2026-11-11: P5b-1 is already on `main` as `97f13a5`, "Committed on `main` by the agent, on the owner's word for
   0052-PLAN Deviation 1" (`0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md:839`). One commit
   remains for this phase: the storage half re-applied from the branch, and F25. It follows C6, the agent staging
   and the owner committing, unless the owner says otherwise as for Deviation 1.)*

Acceptance: the patch reverse-applies; the sort script reports no hunk lost; `make preflight` and `make race` pass on
three hosts.
*(2026-11-11: "the patch reverse-applies" cannot be met after Deviation 1. In its place:
`git diff --binary d6c10a5 1406bd6` has the SHA-256 that the note under P1's heading records, and the sort script
compares `main`, with the re-applied parts, against `1406bd6`.)*

### P2 — the session's fields, the host id and the client store (MADR D4, D5, D6)

Prerequisites: 0051-PLAN P2 (the SDK minor carrying its 0074 and 0075).

*(2026-11-13; the owner's rulings of that day, relayed by the cross-repository coordinator, on three questions put
by a read-only check of this phase against go-llmprovider-sdk. The text below is kept as written; this note and the
notes under steps 1, 4 and 5 say what each part now is, and "P2 split into P2-1 and P2-2 (2026-11-13)", after this phase,
gives the evidence and both parts in full.)* P2 is split. **P2-1** is built now on go-llmprovider-sdk `v1.29.0`,
with no SDK move: the owner's chosen option reads "P2-1 … is built now on v1.29.0 with no SDK move". **P2-2**
follows 0051-PLAN P2 and P2a, and the prerequisite above is P2-2's alone. Step 4 moves, whole, to
[0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
P6 step 1a.

1. `internal/auth/store.go`: `Entry.Kind`, `Entry.Consents`, written `kind` and `consents` when set; `SetKey`'s removal
   list (`:176-177` [WIP]) gains both, so a key leaves no stale session member (MADR D6); `logins.go` and
   `llm/provider/login.go` carry them both ways.
   *(2026-11-13.)* P2-1: `Entry.Kind` and `Entry.Consents`, their members, `oauthFields`, `SetKey`'s removal list
   (`internal/auth/store.go:176-177` on `main` at `8fa0b73`), `Logins.Load` and `Save`, and `provider.Login`'s two
   fields. P2-2: `llm/provider/login.go`'s `session` and `loginOf`, carrying them to and from the SDK's
   `OAuthSession.Kind` and `Consents`.
2. `internal/auth/hostid.go`: `HostID(dir string) (string, error)`, reading `<state>/host-id`, or creating it with
   `O_CREATE|O_EXCL`, owner-only (`securePrivate`, as `auth.json`), a random `urn:uuid`.
   *(2026-11-13: P2-1's.)*
3. `internal/auth/clients.go`: `Clients{Path}` with `Load(ctx, id) (ClientRecord, error)` and `Save(ctx, id,
   ClientRecord) error`, under `clients.json.lock`, with `auth.json`'s write; a zero record removes the member.
   *(2026-11-13: P2-1's. `ClientRecord` is gobble's own type here; `internal/auth` imports no SDK package.)*
4. `llm/provider/login.go`: `sdkStore` implements `auth.ClientStore` when its `LoginStore` also implements gobble's
   `ClientStore`; `SignInOptions.HostID` reaches `wizard.Options.HostID` and `signin.Options.HostID`.
   *(2026-11-13: moved, whole, to 0051-PLAN P6 step 1a, with `TestClientStoreAdapter` and its plant, by the owner's
   ruling of that day, the wizard's `HostID` included; neither P2-1 nor P2-2 does it. Its names are
   go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md`'s `auth.ClientStore` (D7) and
   `wizard.Options.HostID` and `signin.Options.HostID` (D6), none of them in `v1.29.0`, and no gobble code calls
   `signin.Start` before 0051-PLAN P6.)*
5. Tests, each seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestLoginCarriesEverySessionField` (`llm/provider`) | every exported stored field of `auth.OAuthSession`, set to a distinct value, survives `loginOf`, `Logins.Save`, `auth.json`, `Logins.Load` and `session`; the list of fields is read by reflection, so a new SDK field fails it | `consents` not written by `oauthFields` |
   | `TestHostIDIsStable` (`internal/auth`) | two calls give one id; two goroutines racing create one file; the file is owner-only (0600; the DACL test's helper on Windows) | `O_EXCL` dropped, so the second write wins |
   | `TestClientsRoundTrip` (`internal/auth`) | save, load, a zero record removes; a session's `Delete` in `auth.json` leaves `clients.json` | `Delete` removing the client |
   | `TestClientStoreAdapter` (`llm/provider`) | `sdkStore` is an `auth.ClientStore` exactly when its store has one, and passes records both ways | the type assertion inverted |
   | `TestSetKeyDropsSessionMembers` (`internal/auth`, beside `TestSetOAuthDropsKey`) | an `oauth` entry with `kind` and `consents`, then `SetKey`: neither remains in the `api_key` entry, every other field kept | `consents` left out of `SetKey`'s removal list |

   *(2026-11-13.)* `TestHostIDIsStable`, `TestClientsRoundTrip` and `TestSetKeyDropsSessionMembers` are P2-1's, as
   written. `TestLoginCarriesEverySessionField` is P2-2's, after the pin: its plant, "`consents` not written by
   `oauthFields`", moves to P2-1's extension of `TestStoreOAuthPlanFields`, and in P2-2 its plants are `Consents` left
   out of `loginOf` and `Kind` left out of `session`. `TestClientStoreAdapter` moves to 0051-PLAN P6 step 1a with
   step 4.

6. Gates (C6).

Acceptance: the tests pass, each plant failed.

#### P2 split into P2-1 and P2-2 (2026-11-13)

**Found** by a read-only check of 2026-11-13, made for the cross-repository coordinator, of this phase against
go-llmprovider-sdk as gobble pins it, `v1.29.0` (`go.mod:17`, read in the module cache), and as that repository's
`main` stood at `3f87e2b`, read with `git show` and `git grep`:

* `v1.29.0`'s `auth.OAuthSession` has no `Kind` and no `Consents` (`llmprovider/auth/tokenstore.go:21-78`); its
  `wizard.Options` has no `HostID`; it has no `llmprovider/x/signin`.
* At `3f87e2b`, `OAuthSession.Kind`, `auth.SessionKind` and `(*OAuthSession).SessionKind()` exist
  (`tokenstore.go:49`, `:88`; `oauth_session.go:388`): go-llmprovider-sdk
  `0074-MADR-one-sign-in-handle-and-method-facts.md` D5, committed there as `5d08770`, not released. None of the
  names of go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` exists yet (`Consents`,
  `Consent`, `StoredConsent`, `SetConsent`, `ClientRecord`, `ClientStore`, `ConsentText`, `wizard.Options.HostID`:
  `git grep` exit 1), nor does `llmprovider/x/signin`.
* Steps 2 and 3, and step 1's `internal/auth` half, name nothing of the SDK, which `internal/auth` may not import
  (`internal/archtest/check.go:308`). Step 1's `session` and `loginOf` need `Kind` and `Consents`; step 4 needs 0075's
  D6 and D7. The phase needs nothing of `authcontract`, nothing of 0051-PLAN P2 but its pin, and nothing of P2a.
* No gobble code calls `signin.Start` before 0051-PLAN P6, whose step 1 already sets `signin.Options.HostID` and
  `ClientStore`.

**Decision** (the owner, 2026-11-13, relayed by the cross-repository coordinator):

1. P2 is split into P2-1 and P2-2. P2-1 "is built now on v1.29.0 with no SDK move"; P2-2 follows 0051-PLAN P2 and
   P2a.
2. Step 4 moves wholesale to 0051-PLAN P6, the wizard's part included (that plan's P6 step 1a).
3. `TestLoginCarriesEverySessionField` lands in P2-2, after the pin.

**P2-1 — the stored members, the host id and the client file, on `v1.29.0`.** Prerequisites: P1, committed as
`cc1d55e`. It is one of the phases that do not wait on the SDK, which the owner approved on 2026-11-11.

1. `internal/auth/store.go`: `Entry.Kind` and `Entry.Consents`, strings, parsed from and written as `kind` and
   `consents`; `oauthFields` writes each when set and removes it when not, as it does `hostId`; `SetKey`'s removal
   list (`:176-177`) gains both. The values are the SDK's, kept as `scope` is: gobble does not read them here.
2. `internal/auth/logins.go`: `Logins.Load` and `Save` carry both to and from `provider.Login`, which gains `Kind` and
   `Consents` (the `Login` struct in `llm/provider/login.go` only; `session` and `loginOf` are P2-2's).
3. P2's step 2: `internal/auth/hostid.go`, `HostID(dir string) (string, error)`.
4. P2's step 3: `internal/auth/clients.go`, `Clients{Path}` with `Load` and `Save`, and gobble's `ClientRecord`.
5. Tests, each seen failing (C2):

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestStoreOAuthPlanFields` (`internal/auth`, extended) | also: `kind` and `consents` set on an `oauth` entry survive `SetOAuth` and `Load`, each written once; set empty by a later `SetOAuth`, they leave the file | `consents` left out of `oauthFields` |
   | `TestLoginsLoadSave` (`internal/auth`, extended) | also: a `provider.Login` with `Kind` and `Consents` set comes back equal from `Logins.Save` and `Load` | `Consents` left out of `Logins.Save` |
   | `TestHostIDIsStable` (`internal/auth`) | as P2's step 5 row | as P2's step 5 row |
   | `TestClientsRoundTrip` (`internal/auth`) | as P2's step 5 row | as P2's step 5 row |
   | `TestSetKeyDropsSessionMembers` (`internal/auth`) | as P2's step 5 row | as P2's step 5 row |

6. Gates (C6).

Acceptance: the tests pass, each plant failed; `go.mod` and `go.sum` unchanged; `git diff` leaves `session`,
`loginOf`, `sdkStore` and `llm/provider/signin.go` as they were.

What P2-1 leaves inert: nothing sets `Kind` or `Consents`, and nothing calls `HostID` or `Clients`, until later
phases. At `v1.29.0` a refresh saves through `loginOf`, which carries neither, so `oauthFields` can remove only what
nothing has set.

**P2-2 — the SDK's session carries them, after the pin.** Prerequisites: 0051-PLAN P2 (`go.mod` requires
go-llmprovider-sdk `v1.31.0`, with 0074 D5's `Kind` and 0075 D2's `Consents`) and P2a, each committed; P2-1.

1. `llm/provider/login.go`: `session` sets `OAuthSession.Kind`, as `auth.SessionKind`, and `Consents` from `Login`;
   `loginOf` reads both back.
2. Test, seen failing (C2): `TestLoginCarriesEverySessionField` (`llm/provider`), as P2's step 5 row asserts it. Planted
   breaks: `Consents` left out of `loginOf`; `Kind` left out of `session`.
3. Gates (C6).

Acceptance: the test passes, each plant failed; `go.mod` and `go.sum` unchanged from P2a.

**The gap, the owner's accepted cost.** From 0051-PLAN P2's commit to P2-2's, nothing guards the round trip of the
SDK's session through `Login` and `auth.json`. In that window `v1.31.0`'s sign-ins set `Kind` and gobble's `loginOf`
drops it, as it would have without the split, and `SessionKind()` infers the kind of a session saved without one
(0074 D5). No sign-in that answers a consent is offered in it: the first is 0051-PLAN P4, committed with P3, which
needs P2-2.
*(2026-11-15: the gap is closed by P2-2's commit. `loginOf` and `session` carry `Kind` and `Consents`, and
`TestLoginCarriesEverySessionField` guards the round trip of every stored field of the SDK's session, through
`auth.json`; see "P2-2 record (2026-11-15)".)*

Files added to this phase: none *(2026-11-15: P2-2 adds `llm/provider/export_test.go` and the test's file, by
"Deviation 2 (2026-11-15)" in its record)*. Moved out: step 4's (`llm/provider/login.go`'s `sdkStore` as `auth.ClientStore`;
`llm/provider/signin.go`'s `SignInOptions.HostID` and `wizard.Options.HostID`; `TestClientStoreAdapter`), to
0051-PLAN P6 step 1a.

### P3 — consent: wording, gating, the mark and the command (MADR D2, D3)

Prerequisites: P2; 0051-PLAN P3's tag. **Staged with 0051-PLAN P4 and committed with it, as one commit** (that phase
says why): the plan is offered only in the commit that honours its consent.
*(2026-11-15, the owner's ruling; 0051-PLAN's "P3 record (2026-11-15)".)* "0051-PLAN P3's tag" is `acpext/v0.5.0-rc.1`, the
candidate, served by the proxy; the final `acpext/v0.5.0` is 0051-PLAN P8a's. This phase's commit is the one 0051-PLAN P4
makes, on the branch `p3-acpext-v0.5.0` after 0051-PLAN P3's commit, and `main` moves to the branch's tip when every
gate is green on it. *(2026-11-15, the owner's ruling relayed by the coordinator, "Keep it on the branch until those phases land": the coordinator commits each approved phase on the branch `p3-acpext-v0.5.0` and pushes the branch; `main` stays at `9e21c8f` and is fast-forwarded only after 0051-PLAN P6 and 0052-PLAN P4 have landed on the branch and every gate is green, because until then a plan sign-in passes no host id and does not reuse its registered client. Consequence told to and accepted by the owner: CI runs on pushes to `main`, so until `main` moves the phases on the branch have only the local gates and the Linux and Windows runs behind them; `.github/workflows/ci.yml` also runs on `pull_request` and on `v*` tags, which a pull request from the branch would use.)* *(2026-11-15, later: the owner's ruling "Open a draft PR now", relayed by the coordinator. The coordinator's statement that CI runs only on pushes to `main` and tags was wrong: `.github/workflows/ci.yml` also runs on `pull_request`, which the review's finding showed. After P4 is committed and pushed the coordinator opens one draft pull request from `p3-acpext-v0.5.0` to `main`, so every later push to the branch runs CI; `main` does not move and nothing merges until the owner's word, and `main` is still fast-forwarded only after 0051 P6 and 0052 P4 land. The sentence above that the phases on the branch have only the local gates behind them holds only until that pull request is open.)*
*(2026-11-13: "P2" here is P2-1 and P2-2, "P2 split into P2-1 and P2-2 (2026-11-13)".)*

First, `llm/provider/policy.go` (0051-PLAN P4): `honouredConsents` gains `openai.ConsentBackgroundUse`.

1. `llm/provider/consent.go`: `ConsentText`, gobble's wording for `openai.ConsentBackgroundUse` (the WIP's six lines,
   the fifth naming `gobble auth consent openai background deny`, and the question), passed to `wizard.Options` and
   `signin.Options` *(2026-11-15, the owner's ruling "Accept; P6 builds the site", relayed by the coordinator: this phase builds `wizard.Options`' half; the `signin.Options` half is built with its call site in 0051-PLAN P6 step 1, whose `StartSignIn` sets `ConsentText`, and `TestConsentWordingOverACP` of P4 guards it)*; `PlanConsent(id string, store LoginStore) (plan, granted bool)`, from the stored session's kind and
   `Consent`; `SetConsent(ctx, store LoginStore, id, consent string, granted bool) error` over the SDK's
   `auth.SetConsent` through `sdkStore`, and `Consents(ctx, store LoginStore, id string)` listing the answered ones
   (MADR D3). They are `llm/provider`'s, the one package that may import the SDK beyond `authcontract`.
   *(2026-11-11, later; MADR D2's and D3's notes of that date.)* `PlanConsent` returns the consent's id with the
   answer: `func(id string, store LoginStore) (ConsentState, error)`, `ConsentState{Consent string; Plan, Granted
   bool}`, the id read from the offered method's `Consent` fact. `Consents` and `SetConsent` take the consent by
   the SDK's id, `chatgpt_plan.background_use`, with gobble's word `background` kept as its alias; the wording's fifth
   line keeps naming the alias, for a person at a terminal. The adapter wraps the SDK's refusal of a
   marked request on a plan session in `llm.ConsentError` with that id. `Offered` (`policy.go`) sets an honoured
   consent's `Text` to `ConsentText`'s wording and keeps its `Links`.
2. `llm`: `WithBackground(ctx)` and `IsBackground(ctx)`; `llm/provider`'s adapter adds `llmprovider.WithBackground`
   to a marked request's context.
   *(2026-11-11, later.)* And `llm/consent.go`: `ConsentError{Consent string; Err error}`, its `Error` that of `Err`,
   its `Unwrap` `Err`, which matches `ErrNotPermitted`, so `Kind` gives `not_permitted`.
3. `acpserver`: `Options.PlanConsent` replaces the WIP's `PlanSession`; `planGated(id)` asks it at each title and each
   automatic compaction; both mark their context with `llm.WithBackground`; `errPlanOverflow` names `/compact` and the
   command. No per-session copy of the answer.
   *(2026-11-11: `Options.PlanConsent` returns an error too, `func(id string) (plan, granted bool, err error)`, and an
   error is treated as no grant: the request is not made, and an overflow ends the turn with an error of its own that
   names `/compact` and the store's error, not `errPlanOverflow` (MADR D2's note of that date).)*
   *(2026-11-11, later: `Options.PlanConsent` is `func(id string) (ConsentState, error)`, acpserver's
   `ConsentState{Consent string; Plan, Granted bool}`, with the same rule for an error. `errPlanOverflow` is an
   `llm.ConsentError` with `ConsentState.Consent`, and `errorData` (`acpserver/prompt.go:286`) sets
   `ErrorData.Consent` from a `ConsentError` in the chain, so such a turn ends with `kind` `not_permitted` and
   `consent`. The unreadable-answer error wraps `llm.ErrNotPermitted` alone: `kind` `not_permitted`, no `consent`. An
   SDK refusal of a marked request takes one or the other by a fresh `PlanConsent` read when it comes back (MADR D2's
   note, its table).)*
4. `internal/cli/auth.go`: `AuthConsentCmd`, `gobble auth consent <provider> [<consent> allow|deny]`, listing through
   `provider.Consents` and changing through `provider.SetConsent` (step 1); `internal/cli` imports no SDK package for
   it; no session is a usage error, exit 2.
   *(2026-11-11, later; MADR D3's note: a non-interactive caller.)* No prompt and no read of stdin; `<consent>` is the
   SDK's consent id, `chatgpt_plan.background_use`, with `background` its alias; exit 0 stored or listed, 2 for each usage error the note lists, 1 for a store that cannot be read,
   parsed or locked; one stdout line naming the provider, the id and the answer after `allow` or `deny`; every error
   one line on stderr with stdout empty; no `--json`.
5. Tests, each seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestTitleUnderPlan` (reworked) | no title request without a grant; one with it | the gate reading `true` |
   | `TestAutoCompactUnderPlan` (reworked) | no automatic compaction without a grant, an overflow ends the turn naming `/compact` and the command; with it, one summary | `errPlanOverflow`'s old text |
   | `TestConsentAppliesFromTheNextRequest` (`acpserver`) | a session started with a grant, then a withdrawal through a second `Logins` on the same file: the next title or compaction is not made | the answer cached at the session's start |
   | `TestBackgroundRequestsAreMarked` (`acpserver`) | the title's and the compaction's requests reach the provider with `llm.IsBackground` true; a user's turn and `/compact` false | the title's context unmarked |
   | `TestMarkReachesTheSDK` (`llm/provider`) | a marked request on a plan session with no grant is refused `llm.ErrNotPermitted` and the `httptest` server receives nothing | the adapter dropping the mark |
   | `TestAuthConsentCommand` (`internal/cli`) | `allow` and `deny` change the stored answer; the list shows it; a provider with no session is exit 2 | `deny` writing nothing (F31's bug) |
   | `TestConsentWordingOnEverySurface` (`internal/cli`) | `auth login openai`, `/login openai`, setup, the first run and `acp --login openai`, each against `planIssuer`, show the wording before any URL | `/login` built without `ConsentText` |
   | *(2026-11-11, added)* `TestBackgroundRefusedWhenAnswerUnreadable` (`acpserver`) | a plan session with a grant, then a `PlanConsent` that returns an error, the test's `auth.json` having been replaced by text that does not parse: no title request and no automatic compaction are made; an overflow ends the turn with the unreadable-answer error, which gives the store's error and names `/compact`, and does not name `gobble auth consent`; `auth.json` is not written | an error from `PlanConsent` read as a grant; the overflow reported with `errPlanOverflow`'s text |
   | *(2026-11-11, extended)* `TestMarkReachesTheSDK` (`llm/provider`) | also: with the grant stored and `auth.json` then made unparseable, a marked request is refused `llm.ErrNotPermitted` and the `httptest` server receives nothing | the session handed to the SDK with no `Store`, so the SDK reads the session's own earlier `granted` |
   | *(2026-11-11, extended)* `TestAuthConsentCommand` (`internal/cli`) | also: with an `auth.json` that does not parse, the list and `allow` each exit 1 with the store's error and print no answer | the list printing "not asked" and exiting 0 |
   | *(2026-11-11, later, extended again)* `TestAuthConsentCommand` (`internal/cli`) | also, run with stdin closed and no terminal: `allow` by the id `chatgpt_plan.background_use` and by the word `background` store the same answer, each exit 0 with one stdout line and empty stderr; an unknown provider, an unknown consent, a provider with no session, a consent with no `allow` or `deny`, and `maybe` in its place each exit 2 with stdout empty and one stderr line; nothing waits for input | the id not accepted, so `chatgpt_plan.background_use` exits 2; an error printed on stdout |
   | *(2026-11-11, later, added)* `TestConsentErrorIsNotPermitted` (`llm`) | `errors.Is` of a `ConsentError` and `ErrNotPermitted` holds through a `fmt.Errorf("%w")` wrap, `errors.As` finds its `Consent`, and `Kind` gives `not_permitted` | `Unwrap` dropped, so `Kind` gives `other` |
   | *(2026-11-11, later, added)* `TestRefusedBackgroundCarriesConsent` (`acpserver`) | the gate's overflow refusal, and an overflow compaction that a stale grant lets through and the SDK fake refuses while the stored answer reads `declined`, each end the turn with `session/prompt`'s error whose `ErrorData` has `kind` `not_permitted`, `consent` `chatgpt_plan.background_use` and `provider` `openai`; the unreadable-answer overflow, and the SDK fake's refusal while `auth.json` does not parse, each end it with `kind` `not_permitted` and no `consent`; a title not made sends the client nothing | `errorData` not reading `ConsentError`, so `consent` is absent; `errPlanOverflow` a plain error, so `kind` is `other`; the unreadable-answer error built as a `ConsentError`, so `consent` is set |
   | *(2026-11-11, later, added)* `TestOfferedConsentTextIsGobbles` (`llm/provider`) | `Offered("openai")`'s `chatgpt_plan` consent has `Text` equal to `ConsentText`'s and the SDK's usage link; a first plan sign-in's `confirm` step has the same text and link | `Offered` passing the SDK's default text through |

6. Gates (C6).

Acceptance: the tests pass, each plant failed.
*(2026-11-15: built, staged with 0051-PLAN P4 on the branch `p3-acpext-v0.5.0`; see "P3 record (2026-11-15)", and 0051-PLAN's
"P4 record (2026-11-15)" for the gates.)*

### P4 — the plan sign-in on every surface, and the status (MADR D1, D7, D8)

Prerequisites: P2, P3, 0051-PLAN P4 to P7. Between P3's commit and this phase, a first plan sign-in under `--json`
fails closed until 0051-PLAN P5 refuses it with exit 2 (F52's behaviour, nothing saved).
*(2026-11-13: "P2" here is P2-1 and P2-2. `TestSignInChatGPTPlan`'s host id and client, and
`TestPlanAfterLogoutReusesClient`, also rest on P2's old step 4, now 0051-PLAN P6 step 1a, inside "0051-PLAN P4 to
P7".)*

1. Re-apply, reworked, what P1 step 3 took out: `TestSignInChatGPTPlan` and `TestSignInChatGPTPlanPasted` now pass the
   `confirm` step; `TestLogoutPlan`; `TestOpenAILoginIsAPlan` as `TestForeignOpenAISessionRefused`; the acpext pin is
   0051's `v0.5.0`. (0049-PLAN P5b-2's `WindowMinutes` and its test are not here: under QA (a) they land with
   0051-PLAN P4's acpext pin, its step 1.)
2. `internal/cli/auth.go`'s status row: `sessionKind` and `consents` (D8) *(2026-11-11: `sessionKind` is the value of
   `(*OAuthSession).SessionKind()`, go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D5, settled by
   the owner's ruling of that day)*; a session the resolver does not accept is
   `usable: false`, `oauth_unused`; the text form's consent words; the golden file regenerated and its diff reviewed.
   *(2026-11-11, later; MADR D8's note.)* And `consentTerms`, an array of `authcontract.Consent` with gobble's text and
   the SDK's links, on a row whose session is of the kind an offered method with consents signs in to.
3. `internal/cli/auth.go:27`'s help; `docs/guides/credentials.md`'s ChatGPT section (the notice's content, the files,
   the command, the phone through an ACP client, the host id); `README.md`; `docs/architecture.md`.
4. Tests, each seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestSignInChatGPTPlan` (reworked) | a first sign-in's steps begin `confirm` with gobble's wording, then the browser; the session saved with the answer, the host id from `<state>/host-id`, the client in `clients.json`; a second sign-in asks nothing and reuses the client | the host id not passed |
   | `TestPlanAfterLogoutReusesClient` | after `auth logout openai`, a plan sign-in names the stored client and sends no `agent_name_hint` (against `planIssuer`) | `clients.json` cleared by the logout |
   | `TestAuthStatusJSONPlan` | a plan session: `sessionKind` `chatgpt_plan`, `consents`; a foreign openai session: `usable: false`, `reason: "oauth_unused"`; the golden file | `Usable` from `Has` without `Accepts` |
   | *(2026-11-11, later, extended)* `TestAuthStatusJSONPlan` | also: a plan session with no answer has `consentTerms` (`id` `chatgpt_plan.background_use`, `required` true, `text` gobble's wording, the usage link) and no `consents`; a grok session has neither; an unparseable `auth.json` gives 0045-MADR D5's error object, no `providers`, exit 2, and nothing on stderr | `consentTerms` left out while the consent is unanswered; its `text` the SDK's default |
   | `TestForeignOpenAISessionRefused` | status, `auth check`, `authenticate`, provider choice and the provider build all refuse it; the logout deletes and does not revoke | the provider build refusing while status accepts |
   | `TestConsentWordingOverACP` (`acpserver`) | a form-only ACP client's `authenticate` for openai with `_meta.gobble.method: "chatgpt_plan"`, against `planIssuer`: the first elicitation is the `confirm` form, before any `open_url`, its `message` holding gobble's wording, equal to `_meta.gobble.step.text` (MADR Confirmation 2's ACP surface; `TestConsentWordingOnEverySurface` covers the five terminal ones) | `StartSignIn` built without `ConsentText`, so the SDK's default text is sent |

5. Gates (C6).

Acceptance: the tests pass, each plant failed; `grep -rn 'chatgptPlanBackground\|askPlanBackground\|planNotice'
--include=*.go` finds nothing.
*(2026-11-16: built, staged on the branch `p3-acpext-v0.5.0`; see "P4 record (2026-11-16)". The tests pass and each plant failed; the grep finds three lines, all `planNoticeLines`, which is a question for the owner in the record.)*
*(2026-11-16, later; the owner's ruling, relayed by the coordinator, "Amend the acceptance to a whole-word grep": the pattern is matched as whole words,
`grep -rnw -e chatgptPlanBackground -e askPlanBackground -e planNotice --include=*.go .`, which finds nothing (exit 1, run on the staged tree); `planNoticeLines`
is the kept wording of the consent's notice (P3), not the removed `planNotice`. No code changes for it.)*
*(2026-11-16, later; two more rulings of the owner's, relayed by the coordinator, amend P4's steps.)* **Step 1a, one place reads the kind.** "Keep it, and make it
the single place the kind is read": `provider.SessionKindOf` stays exported and is the one call into the SDK's `SessionKind()`; `storedMethod`, the adapter's
`planConsentOf` and `StartSignIn`'s `Previous` call it. `internal/auth` has no inference of its own (`AcceptsLogin` calls `provider.Accepts`), so no layering rule bends.
**Step 1b, the provider build refuses a foreign session.** "Add the check now, in P4": `Options.login` refuses a stored session `!Accepts(id, l)`, with the resolver's error and
words (`provider openai has an OAuth login in auth.json, which gobble does not use (0005-PLAN F4-3b)`, a plain error: the kind `auth check` gives it, `invalid_state`, is
`llm.Kind`'s `other`), so a resolver built before another process replaced the entry cannot hand a foreign session to the build. Tests: `TestNewRefusesAForeignLogin`
(`llm/provider`) and `TestStaleResolverCannotBuildAForeignSession` (`internal/cli`), each with its planted break.
*(2026-11-17, the second review; the owner's rulings, relayed by the coordinator.)* **Step 1c, the terminal tests keep the SDK's callback ports.** "Keep them until the SDK offers a no-listener option": `TestSignInChatGPTPlanTerminal` and `TestPlanAfterLogoutReusesClientTerminal` bind the wizard's callback port 1455, on both loopback families, or 1457 when 1455 is taken (the SDK's own ports), until go-llmprovider-sdk offers a wizard option to skip the listener; when both are in use they fail at once with a message that says so (`signInFailed`); they switch after the SDK release that has the option, a step for the phase that bumps the pin. **Step 1d, a held session is refused at the point of use, later.** The owner's rulings, relayed by the coordinator: "Refuse at the point of use", and then, when the builder stopped on it, "Ask the SDK for interfaces, then build as ruled". Step 1b's kind check is met at the load only; no code was written for the held session in P4. The point-of-use refusal waits for a go-llmprovider-sdk release in which its three concrete-type checks (`llmprovider/internal/chatgpt/chatgpt.go:50` and `:58`, `llmprovider/providers/openai/openai.go:380`, at `v1.31.0`) are interfaces, and is a named step of the gobble phase that bumps the SDK pin to it: a small gobble token source wraps the held session; on each token request it checks the session's current kind under the session's lock and, if the policy no longer accepts it, returns the resolver's error and drops the session; the reproduction, whose text is in "third review" below (not in the tree, because it fails by design today), is adopted into the tree by that phase. Until then a held session can adopt a foreign entry written into `auth.json` during the run (documented in `docs/architecture.md` and `docs/guides/credentials.md`). The request to the SDK is on the coordinator's ledger; it is not filed anywhere by this phase. See "second review and fixes", R2-B2. *(2026-11-17, later; the owner's ruling, relayed by the coordinator, on the SDK's draft record for the held session, "The SDK refuses to adopt another sign-in (Recommended)": the SDK's own session would refuse to adopt a stored session that is not the same sign-in, at the places it adopts from its store, and the token request would fail with the existing auth-failure error, with no interfaces and no new exports. That record is a proposed draft, not accepted and not released. If it is accepted and released, gobble's later step becomes a pin bump and a test that a held session fails, sends no foreign token and stays `chatgpt_plan`, with no wrapper; until then the documented limit stands as written.)*

### P5 — the one-time migration (MADR D9)

*(2026-11-20, re-grounded at `9332cef` before this phase starts. The steps below are unchanged. The facts the executor uses are the dated note on MADR D9 of the same day.)*

* The call is `run` in `internal/cli/main.go`, after `runEnv` is built and before `kctx.Run`. Completion mode returns in `Main` before `run`. No `auth.Load` runs before `kctx.Run`. An error from `Migrate` is a warning and the command continues with the file as it is.
* Kept fields stay the raw `member.value` bytes: remove `key` and `workspace` from an `oauth` entry, and do not rewrite that entry through `SetOAuth`. The consent write is only `CarryConsent`, after `auth.json.lock` is released.
* The backup uses the same owner-only write as `auth.json` (`securePrivate`), including the Windows DACL. C6 still runs on the two other hosts for that reason.
* Not this phase: the held-session refusal and the wizard's no-listener option. Both SDK records are accepted and neither is in the `v1.31.0` tag this repository requires (MADR D9's note of this date, and 0051-MADR's note of this date). 0060 is on `main` (`cde93be` and its fixes) and does not block this phase.
* **Tooling already in the tree, checked 2026-11-20.** This phase uses it and adds no script.
  * Gates stay this repository's. `make pre-add-check` is `scripts/go-precheck.sh`: gofmt, golangci-lint over both modules, and `go tool govulncheck`. The per-file golint step is retired (0064). `make preflight` is the gate list in the Makefile, including `scripts/check_staticcheck_ignored.py`, `scripts/verify_build_metadata.py`, `scripts/check_records.py --check-all` and `scripts/gen_icon.py --check`. `make check-test-isolation` is `scripts/check_test_isolation.py` and is not inside preflight; 0051's C6 still runs it. `make check-libs` is this repository's `scripts/check_libs.py`. There is no `check_libs.json` here, so the sibling go-shared `scripts/check_libs.py` is not this phase's library check, and this phase does not bump a library pin. `scripts/gen_catalog.py` refreshes the embedded catalog and is not a gate of this phase. 0033-PLAN, which would add `scripts/check_disclosure.py` here, is still proposed; this phase does not start it. The fleet `scripts/bump-libs.sh` finds every dependent and commits the pin itself; P8a's pin commit stays the `go get` that phase names.
  * Plants, once the code exists, go through the sibling go-llmprovider-sdk `tools/plant.py --root` pointed at this repository. It scratch-clones the working tree, untracked files included, and does not edit this tree. The red before `Migrate` exists is `go test` here: that sibling's `tools/redgreen.py` counts a package that does not build as a finding, not as the expected failure.
  * The identifier scan of the diff is that sibling's `tools/denyscan.py --root --changed`. It prints the kind of hit, not the identifier. A home-path scan of tracked files is the sibling go-shared `scripts/check_disclosure.py --repo .`. This repository's Makefile calls neither; both already implement the scan. On this date that whole-tree scan reports four findings already in the tree, none in this phase's diff: `docs/decisions/0024-PLAN-cli-mode-polish.md` lines 272 and 273, `permission/rules_test.go:178`, and `session/jsonl/store_test.go:99`. Each is a fixture or a quoted temporary path. This phase does not change those files. The scan this phase must pass is the diff scan.
  * The sibling go-tui-lib `tools/verify` default gates are that library's own targets (`modernize`, `release-check`, `apicheck`). This repository has none of them. This phase does not run that package. This repository has no `tools/` directory; this phase does not add one. The SDK's `tools/records.py` is that repository's index shape and is not used on `docs/README.md`.

*(2026-11-21, the readiness pass at `4c3e397`; records only. The steps below are unchanged; "Readiness pass for P5 and P6 (2026-11-21)", at the end of "Execution record", holds the evidence, the test map and the proposed tests.)*

* **Step 0, added: re-ground first.** Before any code, the executor re-reads every file, line and symbol this phase and MADR D9 cite at the phase's start commit, records each as holding or moved, and stops on a fact that no longer holds.
* **Wait for the open rows.** MADR "State table for what remains" rows M10, M12, M15, M16, M18, M19, M29, M33, M36, M41, M45 and M47 are open, put to the owner as Q1 to Q7. No code of this phase starts before they are answered and written into the table as decided rows.
* **The call's place at `4c3e397`.** 0063 changed `run`: `runEnv` is built at `internal/cli/main.go:90`, `e.ownListings()` is `:91`, `err = kctx.Run(e)` is `:92`, then `e.tellWaitWarn()` (`:93`, 0063-MADR D23, row R156) and `exitCode` (`:94`). "After `runEnv` is built and before `kctx.Run`" still holds: the call goes between `:91` and `:92`. Nothing before `:92` reads `auth.json` (`ownListings` is `internal/cli/models.go:90-94`). Completion mode still returns in `Main` at `:45-47`.
* **Step 1's `internal/archtest/check.go:308`.** `checkRule4` begins at `:304`; the rejection of a go-llmprovider-sdk import outside `llm/provider` is `:318-319` (unchanged since `9332cef`). `TestImportRules` is `internal/archtest/check_test.go:14`; its rule 4 cases (`:50-89`) have none for `internal/auth`, so step 3's new case is still needed.
* **Step 2's "openai's consent id".** `internal/cli` may not import `openai.ConsentBackgroundUse` (rule 4, `check.go:315`), and MADR D2's note says the id is never a constant of gobble's. The closure takes it from `llm/provider`: the `Consent` that `provider.PlanConsent` returns (`llm/provider/consent.go:115-130`), or the word `background`, which `provider.SetConsent` takes (`:174`). `SetConsent`'s signature holds (`consent.go:168`).
* **Step 3's owner-only check** is `appdirs.FileIsOwnerOnly`, as `TestStoreMode` uses it (`internal/auth/store_test.go:352-357`): `0600` on Linux and macOS, the user's DACL alone on Windows.
* **Exit criterion,** in place of the acceptance's wording alone: no blocker, every row tested, plants caught.
* *(2026-11-21, second draft.)* The open rows are now those of the MADR's "Second draft": M9, M10, M12, M15, M16, M18, M19, M29, M33, M36, M45, M56 to M64 and A8, put to the owner as that draft's Q1 to Q8; P5 waits for Q1 to Q7. Step 1's signature gains a lazy file-only `Log`, a `Now` and test hooks, decided now; see "Second draft of the readiness pass (2026-11-21)" at the end of "Execution record".
* *(2026-11-21, third draft.)* The open rows are unchanged; the questions are the MADR's third draft's Q1 to Q8, and P5 waits for Q1 to Q7. `Log` is `lazyLog`'s file-only view of its one file handler, the call site silences `auth status --json` and `auth login --json` before `Migrate`, and `internal/logging/logging.go` joins this phase; see "Third draft of the readiness pass (2026-11-21)" at the end of "Execution record".
* *(2026-11-21, the owner's answers.)* The owner answered Q1 to Q8, each as recommended, and approved the logging file, "Approve: add logging.go to P5 (Recommended)". No row of the MADR's state table is open, and nothing in the state table holds P5 back. Its steps, signature, files and tests are settled in "The owner's answers: P5 and P6 settled (2026-11-21)" at the end of "Execution record", which this phase executes, after its step 0, on the owner's word to start it.
* *(2026-11-22.)* Built in pass 1, unstaged: "P5 record (2026-11-22), pass 1" at the end of "Execution record". Its Deviation 5 adds `internal/auth/clients.go` and the MADR's rows M66 and M67. Every decided M row is tested, every plant is caught, and the gates pass on the three hosts.

1. `internal/auth/migrate.go`: `Migrate(ctx, MigrateOptions{AuthPath, StateDir, Settings func() (bool, bool),
   CarryConsent func(ctx context.Context) error, Log *slog.Logger}) (Report, error)`, MADR D9's steps in order; the
   backup named in UTC; `Report` names what it did. Step 5 calls `CarryConsent`, never the SDK: `internal/auth` may not
   import go-llmprovider-sdk (archtest rule 4, `internal/archtest/check.go:308`).
2. `internal/cli/main.go`: called once per process, before the first credential resolution, with `CarryConsent` a
   closure over `provider.SetConsent(ctx, auth.Logins{…}, "openai", openai's consent id, true)` (P3 step 1); an error
   is a warning and the run goes on with the file as it is.
3. Tests, on copies built in `t.TempDir()` of each shape the owner's file can have, each seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestMigrateDropsDormantKey` | an `oauth` entry with `key` and `workspace`: both gone, one Info line naming the provider and the backup, no value in the log | the log line printing the key |
   | `TestMigrateBacksUpFirst` | the backup holds the old bytes, owner-only, written before `auth.json` changes (a fault injected after the backup leaves `auth.json` unchanged) | the backup written after |
   | `TestMigrateIsIdempotent` | a second run writes nothing and makes no backup | the backup made on every run |
   | `TestMigrateAdoptsIDs` | a plan entry's `hostId` becomes `<state>/host-id`, its `clientId` and `hostId` `clients.json`; existing files are not overwritten | an existing `host-id` replaced |
   | `TestMigrateCarriesConsent` | `chatgptPlanBackground: true` and an unanswered plan session: `CarryConsent` is called once and the session reads `granted`; an answered session is not changed and `CarryConsent` is not called; `settings.yaml` is not written | a `declined` overwritten |
   | `TestImportRules` (`internal/archtest`, a new case) | `internal/auth` importing any go-llmprovider-sdk package is a rule 4 violation | a planted `import _ ".../go-llmprovider-sdk/llmprovider/auth"` in a scratch copy's `internal/auth/migrate.go`, which must fail it |
   | `TestMigrateKeepsEverythingElse` | every other member and field is byte-identical | `merge` dropping an unknown field |
   | `TestMigrateNothingToDo` | a file with no `oauth` entry, or no file: no write, no backup, no state file | a state file created anyway |

4. Gates (C6), the two other hosts included (the backup's DACL on Windows).

Acceptance: the tests pass, each plant failed.
*(2026-11-21: and the exit criterion, no blocker, every row tested, plants caught: every decided M row of the MADR's state table has its test, the proposed ones of "Readiness pass for P5 and P6 (2026-11-21)" included, each seen failing on its plant.)*

### P6 — acceptance and close-out (MADR Confirmation)

*(2026-11-20, before this phase starts. Probes L4 to L7 have not been run. There is no gobble `v0.7.0` tag. `v0.6.0` is an ancestor of `9332cef`. magic-cli-remote requires acp-go-sdk2 `v1.36.0`, the same tag this repository requires, which is what 0055-MADR D2 asks `v0.7.0` to meet. The owner tags; this note does not.)*

*(2026-11-21, the readiness pass at `4c3e397`; records only.)* **Step 0, added:** re-ground first, as P5's step 0, at this phase's start commit. Rows A1 to A7 of the MADR's state table are decided and mapped to their evidence in "Readiness pass for P5 and P6 (2026-11-21)"; A4's key switch has a proposed test, `TestKeySwitchKeepsHostAndClient`, added to this phase. Row A8 is open (Q8). L4 to L7 await the owner's live evidence; the kit they need is described in that entry, to be briefed after P5, and is not built by this pass. Step 4's "the owner tags gobble `v0.6.0`" reads `v0.7.0` (its note of 2026-11-12); `git tag -l` at `4c3e397` lists `v0.6.0` alone. **Exit criterion:** no blocker, every row tested, plants caught, with L4 to L7 recorded as fixed words.
*(2026-11-21, the owner's answer to Q8, "An owner-run L8 first (Recommended)".)* A8 is decided, and L8 joins the owner's live checks: it runs right after P5's commit, before any build carrying P5 runs on the owner's own home, so ahead of this phase's L4 to L7. Its expected words are the MADR's L8 row; what it needs is in "The owner's answers: P5 and P6 settled (2026-11-21)". The exit criterion reads "with L4 to L8 recorded as fixed words".

1. Every Confirmation item checked and recorded with its evidence.
2. **Owner-run probes**, each asked first, each on the host with a scratch `GOBBLE_HOME` (a copy of nothing real), no
   real `auth.json` read:
   * **L4, a first plan sign-in from another device through ACP elicitation:** a scratch form-only ACP client runs
     `gobble acp`; `authenticate` with `openai` and `_meta.gobble.method: "chatgpt_plan"`; the owner answers the
     confirm, opens the URL on the phone, signs in, and pastes the callback. *Owner-run; asks first; scratch home;
     quota: none (no generation); the sign-in registers one client under the owner's ChatGPT account.*
   * **L5, the terminal paste:** `gobble auth login openai` in the scratch home, the browser on another device, the
     callback pasted (0031-PLAN P4b-2 step 10's live check). *Owner-run; asks first; scratch home; quota: none.*
   * **L6, reuse after logout:** `gobble auth logout openai`, then L5 again; the record notes whether a new client was
     registered (the account's settings page), gated with SDK-2's probe. *Owner-run; asks first; scratch home; quota:
     none; may register a second client if OpenAI refuses reuse.*
     *(2026-11-12;
     [0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
     D9's live check, the owner's ruling on its Q3, "No separate test".)* The logout revokes the plan session under gobble's name (0051-PLAN P2a, N7), a
     revocation go-llmprovider-sdk's own
     `0079-PLAN-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md` probe does not send. The record notes that no
     "could not be revoked" warning was printed; a refusal is a deviation (C4).
   * **L7, the consent end to end:** `gobble auth consent openai background deny`, a short session: no title request
     (the run's log); `allow`, a new session: one title. *(2026-11-11, later: the `allow` is given by the id,
     `gobble auth consent openai chatgpt_plan.background_use allow`, as magic-cli-remote will run it, with no
     terminal attached; `deny` keeps the alias.)* *Owner-run; asks first; scratch home; quota: one short title
     generation on the owner's ChatGPT plan.*
3. Notes at completion: `0005-PLAN-v1-feature-scope.md` F4-3d and `0025-PLAN-provider-choice-and-auth.md` P6 done here.
4. The MADR's Completion section; this PLAN `complete`; the owner tags gobble `v0.6.0` with 0051's work.
   *(2026-11-12: "the owner tags gobble `v0.6.0` with 0051's work" now reads `v0.7.0`; gobble `v0.6.0` was cut
   earlier, from `main`, by
   [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md).)*

Acceptance: Confirmation 1 to 8 met, or each unmet item named with its reason.
*(2026-11-21: and the exit criterion, no blocker, every row tested, plants caught.)*

## Verification

Each command's exit status is captured before any filter.

* ~~`git apply --check --reverse <scratch>/0052-wip.patch` (P1, before any edit)~~ *(2026-11-11: it cannot pass after
  Deviation 1. In its place: `git diff --binary d6c10a5 1406bd6 | shasum -a 256` gives the hash the note under P1's
  heading records, and `git ls-remote --heads origin` lists `wip/0031-p4b2-chatgpt-plan` at `1406bd6`.)*
* `make pre-add-check FILES="<the phase's Go files>"`
* `make preflight` and `make race` (macOS); `go test -count=1 ./...` and `go test -race ./...` on scratch copies on the
  Linux machine and the Windows laptop
* `go test -count=1 -run 'TestSetOAuthDropsKey|TestSetKeyDropsSessionMembers|TestStoreOAuthPlanFields|TestHostID|TestClients|TestMigrate' ./internal/auth/`
* `go test -count=1 -run 'TestImportRules' ./internal/archtest/`; `go test -count=1 -run 'TestConsentWordingOverACP' ./acpserver/`
* `go test -count=1 -run 'TestLoginCarriesEverySessionField|TestClientStoreAdapter|TestMarkReachesTheSDK|TestSignInChatGPTPlan|TestPlanAfterLogout|TestLogoutPlan' ./llm/provider/`
  *(2026-11-13: `TestClientStoreAdapter` is 0051-PLAN P6 step 1a's; `TestLoginCarriesEverySessionField` is P2-2's.
  P2-1's extended `TestLoginsLoadSave` runs with the `./internal/auth/` line above, its pattern read with
  `|TestLoginsLoadSave` added.)*
  *(2026-11-16, P4's review: the pattern's `TestSignInChatGPTPlan`, `TestPlanAfterLogout` and `TestLogoutPlan` match nothing in
  `./llm/provider/`: `go test -list` of that pattern prints no test, so the line ran only the other three. Those tests are in
  `internal/cli` (below) or are `TestLogoutRevokesAsGobble`, `TestLogoutRevokesOnlyAccepted` and `TestLogoutReadsTheStoredMethod`
  here. The pattern that runs this phase's tests there is `go test -count=1 -run
  'TestLoginCarriesEverySessionField|TestClientStoreAdapter|TestMarkReachesTheSDK|TestLogoutRevokes|TestLogoutReadsTheStoredMethod|TestStartSignIn|TestNewRefusesAForeignLogin|TestSessionKindOf' ./llm/provider/`;
  `go test -list` of it prints 24 tests, each name in it among them.)*
* `go test -race -count=10 -run 'TestTitleUnderPlan|TestAutoCompactUnderPlan|TestConsentAppliesFromTheNextRequest|TestBackgroundRequestsAreMarked' ./acpserver/`
  *(2026-11-11: with `TestBackgroundRefusedWhenAnswerUnreadable` added to that run's pattern; P3's rows of that date.)*
* `go test -count=1 -run 'TestAuthConsentCommand|TestConsentWordingOnEverySurface|TestAuthStatusJSONPlan|TestForeignOpenAISessionRefused' ./internal/cli/`
  *(2026-11-16, P4's review: it lacked the plan sign-in tests. The pattern that runs this phase's is `go test -count=1 -run
  'TestAuthConsentCommand|TestConsentWordingOnEverySurface|TestAuthStatusJSON|TestForeignOpenAISessionRefused|TestSignInChatGPTPlanACP|TestSignInChatGPTPlanTerminal|TestPlanAfterLogoutReusesClientACP|TestPlanAfterLogoutReusesClientTerminal|TestStaleResolverCannotBuildAForeignSession|TestAdvertisedPlanConsentIsGobbles' ./internal/cli/`
  (`TestAuthStatusJSON` also matches `TestAuthStatusJSONPlan`, `TestAuthStatusJSONAgrees` and `TestAuthStatusJSONError`); `go test -list` of it prints 13 tests, each name in it among them. `TestConsentWordingOverACP` keeps its own line above.)*
* *(2026-11-11, later:)* `go test -count=1 -run 'TestConsentErrorIsNotPermitted' ./llm/`; `go test -count=1 -run
  'TestOfferedConsentTextIsGobbles' ./llm/provider/`; `TestRefusedBackgroundCarriesConsent` joins the `acpserver` race
  run's pattern above
* `make check-records markdownlint`
* Every plant of P1 to P5, quoted in the execution record.

## Rollout and Rollback

* **Order.** go-llmprovider-sdk's `authcontract/v0.1.0` and the minor carrying its 0074 and 0075 (released only after
  SDK-2's probe); this plan's P1 at any time after approval; 0051-PLAN P2; P2; 0051-PLAN P3 and the owner's
  `acpext/v0.5.0` *(2026-11-15: its candidate, `acpext/v0.5.0-rc.1`, here; the final tag is 0051-PLAN P8a's, after its
  P8)*; 0051-PLAN P4 with this plan's P3, one commit; 0051-PLAN P5 to P7; P4; P5; P6; gobble `v0.6.0`,
  tagged by the owner; then magic-cli-remote's cut-over, which relays the `confirm` step and reads
  `sessionKind` and `consents`.
  *(2026-11-11: before all of it, in this repository, `0054-PLAN-go-1-27-4-and-sibling-pin-refresh.md`: Go 1.27.4,
  acp-go-sdk2 `v1.36.0` and go-selfupdate-lib `v1.26.2`. In go-llmprovider-sdk, its Go-floor record, numbered 0077,
  lands before its 0076 minor (`v1.30.0`), which precedes the minor carrying 0074 and 0075; 0077 had no file there
  when this note was written. `v0.6.0` pins acp-go-sdk2 `v1.36.0`, in step with magic-cli-remote (0051-PLAN's rollout
  note of this date). `sessionKind` is the value of `(*OAuthSession).SessionKind()`, which the owner ruled the SDK
  exports.)*
  *(2026-11-11, later: two corrections. go-llmprovider-sdk's 0077 is a file now,
  `0077-MADR-go-1-27-4-for-go-2026-6859.md`. And magic-cli-remote's cut-over does more than read `sessionKind` and
  `consents`: by the owner's ruling, its daemon reads `gobble auth status --json` (`sessionKind`, `consents`,
  `consentTerms`), shows them on the phone, changes a consent by running `gobble auth consent <provider> <consent>
  allow|deny` non-interactively, and maps a refused background request, kind `not_permitted` with acpext's
  `ErrorData.Consent`, to a prompt offering to allow it (MADR D2's, D3's and D8's notes of that date).)*
  *(2026-11-12: "P4; P5; P6; gobble `v0.6.0`, tagged by the owner" now reads `v0.7.0`; gobble `v0.6.0` was cut
  earlier, from `main`, by
  [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md).)*
  *(2026-11-12: "`v0.6.0` pins acp-go-sdk2 `v1.36.0`, in step with magic-cli-remote" stays true, as 0051-PLAN's
  rollout note of this date says: the `v0.6.0` of
  [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md)
  meets it, and the lockstep runs on to `v0.7.0`.)*
  *(2026-11-13: "this plan's P1 at any time after approval; 0051-PLAN P2; P2" now reads "this plan's P1 and P2-1 at
  any time after approval; 0051-PLAN P2 and P2a; P2-2". P2's old step 4 is done in 0051-PLAN P6 step 1a
  ("P2 split into P2-1 and P2-2 (2026-11-13)").)*
* **Rollout.** The migration runs once, at the first `v0.6.0` start. A plan session never existed in a release, so
  only a build of the uncommitted work could have one; the migration covers it.
  *(2026-11-12: "The migration runs once, at the first `v0.6.0` start" now reads `v0.7.0`. "A plan session never
  existed in a release" stays true at `v0.6.0`, the release of
  [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md),
  which carries the four plan fields with no method that creates a plan session (0055-MADR's F5).)*
* **Rollback before a commit:** the phase's edits reversed by the inverse edit (C9); P1's patch restores the
  uncommitted work exactly.
  *(2026-11-11: after Deviation 1 nothing of that work is uncommitted, so there is nothing of it to lose. It is
  restored from the branch: `git show 1406bd6:<path>` for a file, `git diff --binary d6c10a5 1406bd6` for the whole.)*
* **Rollback after `v0.6.0`:** the previous binary reads `auth.json` with `kind` and `consents` (unknown members are
  allowed: `jsonx.StrictOptions`, `jsonx/jsonx.go:23-25`, rejects only invalid UTF-8 and duplicate names), ignores the
  state files, and keeps both members when it rewrites an entry (`merge` keeps unnamed fields). A dropped key is in the
  backup.
  *(2026-11-12: "Rollback after `v0.6.0`: the previous binary reads `auth.json` with `kind` and `consents`" now
  reads `v0.7.0`; the previous binary is then `v0.6.0`'s, cut by
  [0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md),
  whose store is `cc1d55e`'s.)*

## Execution record

~~Not started. The MADR is `proposed`.~~

*(2026-11-11.)* The MADR and this PLAN are `proposed`, and no phase is approved. Deviation 1 has nonetheless carried
out parts of P0 and P1, on the owner's word of 2026-11-11 ("Proceed with the gobble cleanup"). The true state:

| Part | State | Where |
| :--- | :--- | :--- |
| P0 step 1, the MADR `accepted` and this PLAN `in-progress` | not done | waits for the owner |
| P0 step 2, the owner's 0031 paste | done | `86ad4a7` (Deviation 1, step 3) |
| P0 step 3, 0049-PLAN's P5b split into P5b-1 and P5b-2 | done | `97f13a5`, which also holds 0049-PLAN's P5b-1 record |
| P0 step 3, the notes in 0025-MADR, 0045-MADR and 0005-MADR | not done | wait for acceptance |
| P0 steps 4 and 5, the `docs/README.md` rows and the check | not done | the rows are absent while this pair is uncommitted |
| P1 step 1, a copy of the work | done, by a commit and a patch | `1406bd6` on `wip/0031-p4b2-chatgpt-plan`, pushed to origin 2026-11-11; the patch's hash is in the note under P1 |
| P1 step 2, last bullet: 0049-PLAN P5b-1 | done | `97f13a5`, committed on `main` by the agent on the owner's word (`0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md:839`) |
| P1 step 2, first three bullets: the storage half, `MethodChatGPTPlan`, `AgentName` | not done | they left the tree with the rest; P1 re-applies them from the branch |
| P1 steps 3 to 5, the parts taken out | done by the clearing | `main`'s tree holds none of them (`git status --short` at `17d1605`) |
| P1 steps 6 to 8: F25, the tests, the gates, the commit | not started | |
| P2 to P6 | not started | |

*(2026-11-11, later: the table is the state before acceptance. The owner accepted the MADR that day, and P0 is done:
steps 1, 3, 4 and 5 by its record at the end of this section, and step 2 and step 3's 0049-PLAN split checked there as
already done, not repeated. P1 and the later phases are not started.)*

*(2026-11-11, later still: P1's steps 2 (first three bullets), 6, 7 and 8's gates are done and staged, and the owner
has not yet committed them; the "P1 record (2026-11-11)" below says what was run. P2 onward is not started.)*

The verification output is recorded where the work was done: 0049-PLAN's "P5b-1 record (2026-11-11)" for `97f13a5`,
and 0031-PLAN's closing note of P4b-2 for the archive and its hash.

### Corrections of 2026-11-11 (records only)

Directed by the owner on 2026-11-11 and checked against the tree first. Each is a dated note beside the text it
corrects.

* **The base:** `main` at `17d1605` with none of the work; the work on the branch at `1406bd6`, also on origin.
* **P1 against Deviation 1:** the note under P1's heading; step 1's hash claim; step 8's two commits; the acceptance;
  the Verification line; the rollback line.
* **The patch's SHA-256 and size,** recorded here for the first time, with the branch named as the durable copy.
* **The execution record's true state,** above.
* **P3:** an unreadable answer is no grant; one test added and two extended.
* **P4, the rollout:** `sessionKind` through `(*OAuthSession).SessionKind()`; 0054 first; the SDK's Go-floor record
  before its 0076 minor.

**Later on 2026-11-11, the owner's ruling relayed by the cross-repository coordinator,** that magic-cli-remote reads
and can change the consent (records only):

* **Scope:** `llm/consent.go`, `errorData`, the consent text in `policy.go`, `consentTerms`.
* **P3:** `PlanConsent` returns the consent's id; `llm.ConsentError`; `ErrorData.Consent` on a refused turn, and
  none on a turn refused because the answer could not be read; `gobble auth consent` takes the SDK's consent id, with
  `background` its alias, for a non-interactive caller; three tests added, one extended again.
* **P6 L7:** the `allow` given by the id.
* **P4:** `consentTerms`; `TestAuthStatusJSONPlan` extended.
* **Verification:** the new tests.
* **Rollout:** go-llmprovider-sdk's 0077 by its file name; what magic-cli-remote does with the consent.

### P0 record (2026-11-11)

C1: as [0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)'s
P0 record: branch `main`, `HEAD` `834b6d1`, `origin/main` `b6adb5a`, one commit ahead, the tree clean; this pair
`proposed`. Records only. P0 is staged with 0051-PLAN's P0 as one records change, and the owner commits. Every edit
asserted that its old text occurred once before it replaced it, and was read back. Lines below are as they are now.

1. **Acceptance.** The owner's word, typed in this repository's own session on 2026-11-11: "accept and approve 0051 and
   0052 when required"; confirmed to the cross-repository coordinator the same day as: accept 0051-MADR and 0052-MADR,
   and start their phases that do not wait on the SDK. The MADR's front matter is `status: accepted` (its `date` was
   already 2026-11-11). Its Decision Outcome strikes "This record is `proposed`, awaiting the owner's review of this
   draft." and states the acceptance in a dated note (`:162`), quoting the owner's words and naming the questions
   answered on 2026-11-10: QA (a), QF (a) and QG (a). "Revisions of 2026-11-11" gains one paragraph (`:585`). This PLAN
   is `in-progress`, with dated notes under Goal (`:24`) and under the table of "Execution record" (`:447`). No other
   question of this record is marked open or awaiting the owner. Two parts of it are the record's own proposals, which
   its notes say the owner's rulings did not cover, and which are accepted with it: D2's "What the user is shown" for
   an unreadable answer, and D3's "No `--json`" for `gobble auth consent`.
2. **The owner's 0031 paste: already done, by Deviation 1, step 3; checked, not repeated.** `git show --numstat
   86ad4a7` ("docs(decisions): supersede 0031 P4b-2 with 0051 and 0052 plans") adds 89 lines to
   `0031-MADR-sdk-current-releases-and-adoption.md` and 73 to `0031-PLAN-sdk-current-releases-and-adoption.md`, and
   changes 3 in `docs/README.md`. `git blame` gives `86ad4a7` for 0031-MADR D7's note "*Superseded in part 2026-11-10 by
   the owner's answers …*" (`:406-445`) and D8's note (`:464-467`), and for 0031-PLAN P4b-2's "*Superseded 2026-11-10,
   and not carried out as written.*" (`:309`) and its closing note with the archive's hash (`:416-417`). No 0031 file
   was edited.
3. **Amendment notes,** each dated 2026-11-11, beside the text it amends and quoting it:

   | Record | Where | The note |
   | :--- | :--- | :--- |
   | `0025-MADR-provider-choice-and-auth.md` | D6, the ChatGPT plan bullet, `:229` | quotes "a stable `HostID` in the state directory"; built by MADR D4 |
   | the same | D6, the terms bullet, `:235` | quotes "consent, asked once at the first plan session and saved"; refined by D2 and D3: on the session, read per request, changed by `gobble auth consent` |
   | `0045-MADR-auth-status-json.md` | D2, the `oauth_unused` reason, `:118` | enforced by D8: an entry the resolver does not accept is `usable: false`; quotes "It is there only when `usable` is false" |
   | the same | D6, `:157` | under "A field or a `source` or `reason` value may be added", D8 adds `sessionKind`, `consents` and `consentTerms` |
   | `0005-MADR-v1-feature-scope.md` | the OpenAI row, `:168` | quotes "API key or that backend's OAuth"; now "API key, or the ChatGPT plan sign-in under gobble's own name" (D7, F55) |

   * **0049-PLAN's P5b split: already done, checked, not repeated.** "#### P5b split into P5b-1 and P5b-2 (2026-11-11)"
     (`0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md:790-812`), every line of it from `97f13a5`
     (`git blame`), names P5b-1, the `llm` half at `acpext/v0.3.0`, and P5b-2, the wire half with 0051-PLAN P4's
     acpext pin, as QA (a) says. Its "(`proposed`)" for this MADR is the state on the day it was written; the file was
     not edited.
   * **Wording.** The MADR's "Records amended or superseded in part" gives each note's substance: the two quotations of
     0025-MADR, 0045-MADR's "D2 enforced (D8); D6 admits `sessionKind` and `consents`", and 0005-MADR's row text. The
     0045-MADR D6 note also names `consentTerms`, from D8's later note ("An addition under 0045-MADR D6"). None was
     written without a source in the MADR.
   * The MADR's "Records amended or superseded in part" gains a dated note that this step applied them (`:470`).
4. **`docs/README.md`:** the 0052 MADR and PLAN rows (`:119-120`), after 0051's, in the form of their neighbours.
5. **Checks:** run once over both P0s; their output is in 0051-PLAN's P0 record, step 4: `check_records.py
   --check-all` exit 0; `make check-records markdownlint` exit 0, "Summary: 0 issues in 0 files"; the planted broken
   link, in this MADR's link from the 0045-MADR note, seen failing; the identifier scan, nothing found.

Acceptance: every note quotes what it amends; the 0031 text is Deviation 1's paste, made on the owner's word
(`86ad4a7`). Met. P1 is not started.

### P1 record (2026-11-11)

C1: branch `main`, `HEAD` `5007c69`, three commits ahead of `origin/main` `b6adb5a`, the tree clean; the archive
branch `wip/0031-p4b2-chatgpt-plan` at `1406bd6` locally and on origin, its parent `d6c10a5`. P1 is read against
Deviation 1 and the note under P1's heading. The agent stages; the owner commits with `git commit --no-edit`. Nothing
was committed, pushed or tagged. Scratch logs are under `<TMP>` (the session's scratch directory).

1. **Step 2, first three bullets, re-applied from `1406bd6`.** `internal/auth` and `llm/provider` are unchanged
   between `d6c10a5` and `HEAD` apart from `provider.go` and `limit_test.go` (97f13a5), so each hunk applied to
   `main`'s code as it stands. No file was checked out from the branch.
   * `internal/auth/store.go`, `logins.go`, `store_test.go`: all hunks of the archive's diff of those three files
     (`git apply --check` exit 0, then applied): `Entry`'s `Resource, Scope, IDToken, HostID`; `parse`'s four JSON
     members; `SetKey`'s removal list; `oauthFields`' four names; `Logins.Load` and `Save`; `TestStoreOAuthPlanFields`.
   * `llm/provider/login.go`, three hunks by hand: `Login`'s four fields; `session`'s four; `loginOf`'s four.
     `Login.Plan()`, `HasLogin`'s openai and `HasPiLogin` (steps 3 and 4) were not brought back.
   * `llm/provider/login_test.go`, one hunk by hand: the plan half of `TestLoginStoreAdapter`. Its closing
     "a provider with no login" probe asks about Claude in place of OpenAI, as in the archive, since OpenAI now has a
     stored plan session in that test. The archive's imports, `testIssuer.scope`, `TestLogoutPlan` and
     `TestOpenAILoginIsAPlan` stay on the branch (step 3).
   * `llm/provider/signin.go`, four hunks by hand: `MethodChatGPTPlan`, `AgentName`, `methodIDs`' entry, and the
     wizard's `AgentName: AgentName` (MADR D1's `:201`). `offers` is unchanged, so the method stays inert: it skips
     `AuthChatGPTPlan`. `go build ./...` exit 0 and `go test -count=1 ./internal/auth/ ./llm/provider/` exit 0 followed.
2. **Step 6, F25.** `oauthFields` ends with `remove = append(remove, "key", "workspace")`, with a comment naming F25,
   and its doc comment says so. `SetOAuth` therefore leaves no key or workspace in an `oauth` entry.
3. **Step 7, each test seen failing on a scratch copy (C2); the tree was never dirtied.**

   | Test | Scratch copy | Failure seen |
   | :--- | :--- | :--- |
   | `TestSetOAuthDropsKey` (new, `internal/auth/store_test.go`) | the tree with `key` left out of `remove` | exit 1: `store_test.go:295: "sk-dormant" survived SetOAuth:`, the entry shown still holding `"key": "sk-dormant"`, and `store_test.go:310: openai after SetOAuth: {Type:oauth Key:sk-dormant …}`; passes in the tree |
   | `TestStoreOAuthPlanFields` (re-applied) | `git archive HEAD` plus this one test, no fields | exit 1, build failed: `store_test.go:225:3: unknown field Resource in struct literal of type Entry`, then `IDToken`, `HostID`, and `e.Resource undefined (type Entry has no field or method Resource)`; passes in the tree |
   | `TestSortedTreeIsThePatch` (a scratch script, not committed) | see below | see below |

   `TestSetOAuthDropsKey` builds an `api_key` entry with `key`, `workspace` and a `note`, beside another provider's
   entry with its own `key` and `workspace`; after `SetOAuth` neither dormant value is in the file, `note` and the
   written fields are, and the other provider's entry is unchanged.

   **The sort script** (`<TMP>/sort.py`; not in the repository). It reads every change group of
   `git diff -U0 d6c10a5 1406bd6` (99 groups in 39 files) and looks for the archive's lines in `HEAD` and in the
   tree. A group in both is "on main already", and the script checks that `97f13a5` or `86ad4a7` touched that file; a
   group in the tree only is "re-applied in P1"; a group in neither must be claimed by one `ACCOUNTING` entry naming
   the step that leaves it on the branch (3, 4 or 5), and an entry that claims no group, or a group no entry claims,
   fails it. Markdown is compared with `* ` bullets read as `- `, since the owner's paste (`86ad4a7`) was linted to
   dashes, and trailing space is ignored; with that, the archive's 0031 text is on `main` line for line.

   * Run on the tree, exit 0: `files 39`, `re-applied in P1 16`, `on main already 14`, `left on the branch 69`,
     `UNACCOUNTED 0`. The 14 are `llm/errors.go` (2), `provider.go` (8), `limit_test.go`, `choice_test.go` (97f13a5);
     the 0031 MADR and the 0031 PLAN (86ad4a7). The 16 are the files and hunks of item 1.
   * Planted break 1, one `ACCOUNTING` entry dropped (`offers`' guards, `--drop 4`), exit 1: `UNACCOUNTED 1` and
     `PROBLEM: llm/provider/signin.go group 5 (-0 +1, archive line 87): not in the tree, no ACCOUNTING entry`.
   * Planted break 2, on a scratch copy of the tree, the `AgentName: AgentName,` hunk removed, exit 1:
     `re-applied in P1 15`, `UNACCOUNTED 1`, `PROBLEM: llm/provider/signin.go group 8 (-0 +1, archive line 201): not
     in the tree, no ACCOUNTING entry`.
   * What it does not prove: that a group present in the tree is in the right place, only that its lines are there.
     The 69 left on the branch are the files and hunks that steps 3, 4 and 5 name: step 3 (reworked: `HasLogin`'s
     openai, `Login.Plan()`, `offers`' guards, `provider.go`'s refusal, `loginWith`, the status row, `PlanSession`,
     the gating in `acpserver`, their tests, and `planIssuer` and `planCallback` with `TestSignInChatGPTPlan` and
     `TestSignInChatGPTPlanPasted`); step 4 (discarded: the notice and the consent prompt, `HasPiLogin`,
     `piimport.go`, the `chatgptPlanBackground` setting in `internal/config` and its readers); step 5 (the wire half:
     `prompt.go`'s `WindowMinutes`, `weeklyErr`, acpext `v0.4.0` in `go.mod` and `go.sum`).
4. **Step 8, gates (C6), on the Mac.** Each exit status was taken before any filter; no proxy variable was set.
   * `make pre-add-check FILES="internal/auth/store.go internal/auth/logins.go internal/auth/store_test.go
     llm/provider/login.go llm/provider/signin.go llm/provider/login_test.go"`, exit 0: "No vulnerabilities found",
     "go-precheck: 6 file(s) clean (gofmt, golint, govulncheck)".
   * `make preflight`, exit 0, "preflight passed": gofmt, `go mod tidy`, pre-add-check, vet, staticcheck,
     golangci-lint, govulncheck, `check-acpext-deps`, `go fix`, archtest, apidiff, build metadata, `check-records`,
     `check-icon`, and markdownlint ("Summary: 0 issues in 0 files"). It ran before this record was written; the
     record's own checks are below.
   * `GOFLAGS=-count=1 make race`, exit 0, 39 `ok` lines (the root module and acpext); no failure, so
     `TestRuleStoreConcurrent` and every untouched test passed.
   * `make check-test-isolation`, exit 0: root and acpext modules `go test` exit 0, 0 failing, "files written above
     the temp roots: 0".
   * `go.mod`, `go.sum` and the `go` lines are unchanged; no golden file changed.
5. **The other two hosts,** each on a copy of the tree (947 files from `git ls-files -co --exclude-standard`, none
   missing, packed with `--no-mac-metadata`; the remote count was 947 and no `._*` file on each). Both hosts run Go
   1.27.4. The Linux machine got the archive through ssh's standard input; the Windows laptop by `scp`, then `tar` on
   the host, with `-o ControlPath=none` on every ssh. No proxy variable was set. Remote copies were removed
   afterwards, older ones left.

   | Host | `go test -count=1 ./...` | `CGO_ENABLED=1 go test -count=1 -race ./...` | `go -C acpext test -count=1 ./...` |
   | :--- | :--- | :--- | :--- |
   | Linux machine | exit 0, 38 `ok` | exit 0, 38 `ok` | exit 0, 1 `ok` |
   | Windows laptop | exit 0, 38 `ok` | exit 0, 38 `ok` | exit 0, 1 `ok` |

   `make preflight` is macOS-only and was not run there.
6. **Records.** This record and the three dated notes above; `python3 -B scripts/check_records.py --check-all`
   and `make check-records markdownlint` were run last, after this record was written: both exit 0, "Summary: 0
   issues in 0 files". The identifier scan of the diff looked for the local account name, the hosts' names and
   domain, the owner's e-mail address, `/Users/` and `/home/` paths and the scratch directory, and found nothing
   (`grep -c` gave 0).
7. **Not done.** The commit (the owner's); P2 onward; the archive's wire half, which waits for the SDK's tags and
   0051-PLAN P4. No deviation was needed: no hunk had to be reworked, no existing test or golden changed, no
   plant went uncaught.

Staged: the six Go files above and this PLAN. P1's acceptance, "the sort script reports no hunk lost; `make preflight`
and `make race` pass on three hosts", is met, with `make preflight` on the Mac alone as the PLAN says.

### Amendment of 2026-11-13 (records only)

The owner's rulings of 2026-11-13 on a read-only check of P2 against go-llmprovider-sdk, relayed by the
cross-repository coordinator. Read against this repository at `8fa0b73`, `main`, a clean tree. Each change is a dated
note beside the text it changes, or a new subsection; nothing is struck.

* **P2:** notes under its heading and under steps 1 to 5; "P2 split into P2-1 and P2-2 (2026-11-13)", new: the evidence, the
  rulings, P2-1, P2-2 and the gap between the pin and P2-2.
* **P3, P4:** notes on their prerequisites. **Verification:** where two tests now run. **Rollout:** the order.
* **0051-PLAN:** the gate before P2, P2a's place, P4's prerequisite, P6 step 1a (new, with `TestClientStoreAdapter`),
  the scope, the Verification and the rollout. **0051-MADR:** D9's "Where in the PLAN", and its revisions.
* **Not changed here:** the fleet sequence's `0006-PLAN-sequence-the-auth-contract-effort-across-four-repositories.md`,
  outside this repository, which the cross-repository coordinator amends.

~~P2-1 is not started.~~ *(2026-11-13: P2-1 is built and staged; see "P2-1 record (2026-11-13)" below.)*

### P2-1 record (2026-11-13)

C1: branch `main`, `HEAD` `4da8559`, the tree clean at the start. P2-1 is read against "P2 split into P2-1 and P2-2
(2026-11-13)". The agent stages; the owner commits with `git commit --no-edit`. Nothing was committed, pushed or
tagged. Scratch logs are under `<TMP>` (the session's scratch directory).

1. **Step 1, `internal/auth/store.go`.** `Entry.Kind` and `Entry.Consents`, strings; `parse` reads `kind` and
   `consents`; `oauthFields` writes each when set and removes it when not, as it does `hostId`; `SetKey`'s removal list
   gains both.
2. **Step 2, `internal/auth/logins.go` and `llm/provider/login.go`.** `Logins.Load` and `Save` carry both;
   `provider.Login` gains `Kind` and `Consents`, in the struct only. `session`, `loginOf`, `sdkStore` and
   `llm/provider/signin.go` are untouched; `go.mod` and `go.sum` are unchanged.
3. **Step 3, `internal/auth/hostid.go`.** `HostID(dir)`: reads `<dir>/host-id`, or creates it with
   `O_WRONLY|O_CREATE|O_EXCL`, `0o600` (`securePrivate` as well, for the Windows DACL), a random version 4
   `urn:uuid` from `crypto/rand`. A caller that loses the race reads the winner's id, waiting up to 5 s for the first
   byte, since the file exists before its content does. A file holding anything but a `urn:uuid` is an error and is
   not replaced.
4. **Step 4, `internal/auth/clients.go`.** `ClientRecord{ClientID, HostID}`, gobble's own type; `Clients{Path}` with
   `Load` and `Save`, under `clients.json.lock`, written with auth.json's `write`; a zero record removes the member;
   other providers' members are kept. `internal/auth` imports no SDK package.
5. **Tests.** `TestStoreOAuthPlanFields` and `TestLoginsLoadSave` extended; `TestHostIDIsStable`,
   `TestClientsRoundTrip` and `TestSetKeyDropsSessionMembers` new, as the table asks. Two further tests, not in the
   table: `TestHostIDRefusesOtherContent` and `TestClientsBadFile`.
6. **Plants, each seen failing on a scratch copy (C2); the tree was never dirtied.** Each plant built (exit 0) and
   failed the one test:

   | Test | Planted break | Failure seen |
   | :--- | :--- | :--- |
   | `TestStoreOAuthPlanFields` | `consents` left out of `oauthFields` | `store_test.go:247: "\"consents\": \"background_use\"" written 0 times in:` (and `:237`, `Consents:` empty against `Consents:background_use`) |
   | `TestLoginsLoadSave` | `Consents` left out of `Logins.Save` | `logins_test.go:24: Load: Login{...}, <nil>; want Login{...}` (`Login.String` omits the two fields, so both sides print alike) |
   | `TestHostIDIsStable` | `O_EXCL` dropped (`O_TRUNC` in its place) | `hostid_test.go:65: round 0: caller 1 got "urn:uuid:8de304f7-...", caller 0 got "urn:uuid:dd989827-..."; want one id` |
   | `TestClientsRoundTrip` | `Delete` removing the client | `clients_test.go:58: after the session's Delete: {ClientID: HostID:}, <nil>; want {ClientID:oaiapp_x HostID:urn:uuid:host}` |
   | `TestSetKeyDropsSessionMembers` | `consents` left out of `SetKey`'s removal list | `store_test.go:294: "consents" survived SetKey:` and `"background_use" survived SetKey:` |

   The `TestHostIDIsStable` plant was run again after the change under item 8, against the final `hostid.go`; the
   failure above is that run's. The restored copy passed.
7. **Gates (C6).** `make pre-add-check FILES=<the nine Go files>`: exit 0, "9 file(s) clean (gofmt, golint,
   govulncheck)", before and after item 8.
8. **Deviation (2026-11-13): gosec G304 on the host id's `os.OpenFile`.**
   * **Found:** the first `make preflight` on the Linux machine exited 2 at golangci-lint:
     `internal/auth/hostid.go:35:12: G304: Potential file inclusion via variable (gosec)`, at
     `os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)`. The finding was in the new file, so it did not
     predate P2-1; `scripts/go-precheck.sh` does not run gosec. No test had run.
   * **Resolutions put to the owner:** (1) create through a temp file and `os.Link`, which departs from the step's
     `O_EXCL`; (2) a `//nolint:gosec` line, which the repository's own convention uses and the owner's rules forbid
     for a gate failure; (3) route the create through an `fsx` helper, a file outside P2-1's list.
   * **Ruling (the owner, 2026-11-13, relayed by the cross-repository coordinator):** "filepath.Clean the path".
     `HostID` opens `filepath.Clean(path)`; `O_WRONLY|O_CREATE|O_EXCL` and `0o600` stand, so the step and the plant
     ("`O_EXCL` dropped") stand. No `nolint` was added.
   * **Seen to clear:** `golangci-lint run --config .golangci.yml ./internal/auth/...` exit 0, "0 issues", then the
     full `make preflight` below.
9. **Suites, after the last code change** (the `filepath.Clean` edit was the last):

   | Host | Command | Result |
   | :--- | :--- | :--- |
   | Linux machine | `make preflight` | exit 0, "preflight passed", 721 s |
   | Windows laptop | `go test -count=1 ./...` | exit 0, 38 `ok` |

   `make preflight` has no test stage; its tests are `archtest` and the lint. `go test -race` was not run: P2-1 adds
   no concurrency beyond `HostID`'s race test, which the plain run exercises.
10. **Records.** This record and the dated notes: `python3 -B scripts/check_records.py --check-all`,
    `make check-records`, the redaction dry run and an identifier scan of the added lines were run last; their results
    are given in the hand-off.
11. **Not done.** The commit (the owner's); P2-2; step 4's `sdkStore` and the wizard's `HostID` (0051-PLAN P6 step
    1a); anything that sets `Kind` or `Consents` or calls `HostID` or `Clients`, which stay inert.

Staged: the nine Go files and this PLAN. P2-1's acceptance, "the tests pass, each plant failed; `go.mod` and `go.sum`
unchanged; `session`, `loginOf`, `sdkStore` and `signin.go` as they were", is met.

### P2-2 record (2026-11-15)

C1: branch `main`, `HEAD` `4497fad`, the tree clean at the start, `origin/main` equal to it; `go.mod` requires
go-llmprovider-sdk `v1.31.0`. The agent stages; the owner commits. Nothing was committed, pushed or tagged. Scratch
logs are under `<TMP>`.

#### Deviation 2 (2026-11-15): the test needs a test-only export file

* **Found,** before any edit, when P2-2's step 2 was read against the tree. P2's step 5 row asks
  `TestLoginCarriesEverySessionField` (`llm/provider`) to carry every stored field through `loginOf`, `Logins.Save`,
  `auth.json`, `Logins.Load` and `session`. `Logins` is in `internal/auth`, and `internal/auth/logins.go:9` imports
  `llm/provider`, so a test in `package provider` that imports `internal/auth` is an import cycle. `session` and
  `loginOf` are unexported (`llm/provider/login.go`), so a test in `package provider_test`, which may import
  `internal/auth`, cannot call them. No exported function gives the round trip: `session` is reached only through
  `Logout`, `provider.go`'s session builder and `sdkStore`; `loginOf` only through `sdkStore.Save`, which is unexported.
  `llm/provider` had no `export_test.go`. The PLAN did not foresee it.
* **Resolutions put to the owner:** (1) a test-only `llm/provider/export_test.go` exposing `session` and `loginOf` to
  the external test package, the test written there against the real `internal/auth` store; (2) an in-package test
  over `memLogins`, which leaves `Logins` and `auth.json` out of the chain; (3) the test in `internal/auth`, which still
  needs the two functions exposed.
* **Ruling (the owner, 2026-11-15, put by the coordinator):** "Test-only export file": "Add
  llm/provider/export_test.go, a test-only file that exposes session and loginOf to the external test package, and
  write the test there against the real internal/auth store over a scratch auth.json. The plan's full chain is tested
  as written; one test-support file joins the phase's scope by a dated deviation; no production effect. The agent first
  confirms the import-rule check and the test-isolation check accept it." Rejected: "In-package test, no auth.json";
  "Put the test in internal/auth".
* **Files added to this phase:** `llm/provider/export_test.go`, and the test's own file in `llm/provider`.

1. **The SDK's declarations, read at `v1.31.0` (`llmprovider/auth/tokenstore.go`).** `:49` `Kind SessionKind`; `:56`
   `Consents   string`; `:101` `type SessionKind string`, with the constants at `:109-112` (`"chatgpt"`,
   `"chatgpt_plan"`, `"grok"`, `"kilo_device"`). `Consents` is a plain string, "space-separated `<id>=granted` or
   `<id>=declined` entries sorted by id", kept a string so the struct stays comparable. `provider.Login.Consents` is a
   `string`, so the mapping is a copy both ways, with no parsing; `Login.Kind` is a `string` and `OAuthSession.Kind` an
   `auth.SessionKind`, so it is a conversion both ways.
2. **Step 1, `llm/provider/login.go`.** `session` sets `Kind: auth.SessionKind(l.Kind)` and `Consents: l.Consents`;
   `loginOf` sets `Kind: string(s.Kind)` and `Consents: s.Consents`; the doc comments say so. No other production code
   changed; `sdkStore` and `signin.go` are untouched.
3. **Step 2, the test.** `TestLoginCarriesEverySessionField`, in `llm/provider/login_fields_test.go`
   (`package provider_test`), with `llm/provider/export_test.go` (`SessionOf` and `LoginOf`, the two unexported
   functions the external package needs; nothing else). The test:
   * reads the exported fields of `auth.OAuthSession` by reflection and sets each to a value of its own (a string
     `v<Field>`, `true`, a distinct millisecond time); `Provider` is set to the store key, since it is stored as the key;
   * saves that session through the SDK's own `FileTokenStore` and reads the file's members. **A field is "stored" when
     its member is in that file** (names compared lower case without underscores), so the SDK's code, not a list kept
     here, says which fields are stored. The collaborators `Store`, `HTTPClient` and `Logger` are interfaces and
     pointers and are not filled, by kind and not by name; any other exported field of a kind the test cannot fill fails
     it, and so does a file member that matches no field;
   * checks `Kind`, `Consents`, `Access` and `HostID` are among the stored fields, so the reflection cannot pass by
     finding nothing;
   * runs the stored fields through `LoginOf`, the real `internal/auth` `Logins.Save`, an `auth.json` in a temporary
     directory, `Logins.Load` and `SessionOf`, and fails naming each field that differs.
   `make archtest` and `make check-test-isolation` accepted a `provider_test` file importing `internal/auth`, on a scratch
   copy with a minimal such file, before the test was built on it (both exit 0; archtest reads `XTestImports`).
4. **Plants, each seen failing on a scratch copy (C2); the tree was never dirtied.** Each built (exit 0) and failed the
   one test, run again after the last edit to the test file; paths in the output are `<TMP>`-relative:

   | Planted break | Failure seen |
   | :--- | :--- |
   | the tree before the phase (`login.go` as committed) | `login_fields_test.go:123: OAuthSession.Kind: set vKind, came back ` and `:123: OAuthSession.Consents: set vConsents, came back ` |
   | `Consents` left out of `loginOf` | `login_fields_test.go:123: OAuthSession.Consents: set vConsents, came back ` |
   | `Kind` left out of `session` | `login_fields_test.go:123: OAuthSession.Kind: set vKind, came back ` |
   | `consents` left out of `oauthFields` (`internal/auth/store.go`, P2's row) | `login_fields_test.go:123: OAuthSession.Consents: set vConsents, came back ` |
   | the reflection's own job: `Scope` left out of `loginOf` | `login_fields_test.go:123: OAuthSession.Scope: set vScope, came back ` |

   Every existing test of `llm/provider` passes unchanged (`make race` ran the package).
5. **Gates (C6),** every one after the last Go file change (the last edit was `login_fields_test.go` at 10:06:08):

   | Gate | Where | Exit | Start to end |
   | :--- | :--- | :--- | :--- |
   | `make pre-add-check FILES=` the three Go files | this Mac | 0, "3 file(s) clean (gofmt, golint, govulncheck)" | 10:06:55 to 10:07:03 |
   | `make preflight` | the Linux machine, by `run-on-host` | 0, "preflight passed" | 10:07:20 to 10:19:09 *(2026-11-15, later: this line does not show that lint was clean; revive's `confusing-naming` finding is intermittent, Deviation 4, and this was one lint draw. Superseded by the gates of "P2-2 fix record" below.)* |
   | `make race` | this Mac, by `run-on-host --race` | 0, `llm/provider` and `acpserver` ran, the rest cached | 10:19:09 to 10:19:56 |
   | `make check-test-isolation` | the Linux machine, by `run-on-host` | 0, "0 failing", "files written above the temp roots: 0" | 10:19:56 to 10:21:24 |

   No WSL distro ran anything. No cross-host run of C6's P2, P4, P6 and P8 kind was named for this phase, and none was
   run: the change is two fields in two functions and a test, and the full-suite hosts above ran the package.
6. **Deviation 3 (2026-11-15): a gate failure in the new test, fixed in the file.** The first `make preflight` (the
   Linux machine, 09:54:05 to 10:05:36, exit 2) failed at golangci-lint:
   `llm/provider/login_fields_test.go:4:2: import 'encoding/json' is not allowed from list 'json': use jsonx, on
   encoding/json/v2 (0039-MADR D24) (depguard)`. The finding was in this phase's new file. It was fixed by importing
   `jsonx` and calling `jsonx.Unmarshal`, the repository's own rule, with no `nolint`; every gate above ran after it, and
   the plants were run again after it.
7. **Acceptance.** The test passes, and the tree before the phase and each of four planted breaks failed it, as above;
   `git diff --stat 4497fad -- go.mod go.sum` is empty (exit 0, no output).
8. **Docs (C9).** No user document is affected: nothing yet sets `Kind` or `Consents`, and `README.md`,
   `docs/architecture.md`, `docs/guides/credentials.md` and `AGENTS.md` do not describe either member, `loginOf`,
   `session` or `OAuthSession`. This repository keeps no changelog. This record, the note closing "The gap, the owner's
   accepted cost", the PLAN's date and `docs/README.md`'s row are the docs changed.
9. **Not done.** The commit (the owner's); step 4's `sdkStore` and the wizard's `HostID` (0051-PLAN P6 step 1a);
   anything that sets `Kind` or `Consents` outside the SDK's own sign-ins.

Last-modified times of the changed Go files: `llm/provider/login.go` 09:51:33, `llm/provider/export_test.go` 09:51:21,
`llm/provider/login_fields_test.go` 10:06:08. Staged: those three, this PLAN and `docs/README.md`. Provenance:
mainstream model, medium effort.

### P2-2 fix record (2026-11-15, after CI on `d82abd0`)

#### Deviation 4 (2026-11-15): `export_test.go`'s names differ from production names only by capitalization

* **Found** by CI on `d82abd0`, the commit of the five staged files above: `validate (ubuntu-24.04)` and `validate
  (macos-15)` failed in `make preflight` at lint with `llm/provider/login.go:142:6: confusing-naming: Method 'loginOf'
  differs only by capitalization to function 'LoginOf' in llm/provider/export_test.go (revive)`, then `make[1]: ***
  [lint] Error 1` and `make: *** [preflight] Error 2`. `validate (windows-2025)`, `test-isolation`, `sbom` and the Docs
  workflow passed.
* **Root cause.** The finding is intermittent: revive's `confusing-naming` reports against whichever of the two names
  it sees second, and files are linted concurrently, so the same tree is clean on some runs and red on others. The
  preflights recorded above each ran one lint and printed "0 issues"; each was one draw.
  * On this Mac at `d82abd0`, `make lint` through `run-heavy` exited 0, "0 issues", on a warm cache, and again with a
    fresh `GOLANGCI_LINT_CACHE`.
  * With a fresh `GOLANGCI_LINT_CACHE` for every run, `golangci-lint run --config .golangci.yml ./llm/provider/...` on
    the same tree gave 4 red runs of 6, each printing the CI line above and `1 issues: * revive: 1`, and 2 clean.
  * The Linux machine's lint is the CI pin: golangci-lint 2.14.0 built with go1.27.4, `make lint` reading the same
    `.golangci.yml`, with `run.tests: true`. `run-on-host` copies the whole working directory with `rsync -a --delete`
    (`node_modules` excluded), so the new files were in the copy. The record cannot say whether the Linux lint cache
    was warm for the package; either way that run was a single draw. The `.golangci.yml` exclusion for revive covers
    `_test.go` paths only, and this finding is reported at `login.go`, so it is not excluded.
  * `validate (windows-2025)` ran the same lint and was clean: one draw, with no more cause than that.
  * What the gate checked: that one lint run reported nothing. It does not check that the lint result is stable, and a
    warm cache returns an earlier result for a package whose files have not changed. A single clean `make preflight`
    is not evidence about a finding of this kind. The dev-host scripts, the Makefile and the lint config are
    unchanged.
* **Fix,** inside the phase's files: `export_test.go`'s two exports are renamed so that no name differs from a
  production name by capitalization alone: `SessionOf` is now `SDKSessionOf` and `LoginOf` is now
  `LoginFromSDKSession` (the production `session` and `loginOf` are unchanged), and `login_fields_test.go`'s two calls
  follow. No `nolint`.
  * Red, before: the loop above, 4 of 6 runs with the finding. Green, after: the same loop, 12 of 12 runs exit 0 with
    no `confusing-naming`.
* **Plants, run again on scratch copies after the rename** (the tree before the phase now means `login.go` as of
  `4497fad`, since `HEAD` holds the fix): the pre-phase `login.go` fails with `login_fields_test.go:123:
  OAuthSession.Kind: set vKind, came back ` and `:123: OAuthSession.Consents: set vConsents, came back `; the other
  four plants (`Consents` out of `loginOf`, `Kind` out of `session`, `consents` out of `oauthFields`, `Scope` out of
  `loginOf`) fail as before, each naming its field at `login_fields_test.go:123`.
* **Gates (C6), every one after the last file change** (`export_test.go` and `login_fields_test.go`, 10:42:39; the
  PLAN's own text came after, and `make check-records markdownlint` ran last):

  | Gate | Where | Exit | Start to end |
  | :--- | :--- | :--- | :--- |
  | `make pre-add-check FILES=` the three Go files | this Mac | 0, "3 file(s) clean" | 10:45:03 to 10:45:10 |
  | the lint loop, fresh cache each run | this Mac | 12 of 12 exit 0 | about 10:42 to 10:44 |
  | `make preflight` | this Mac, by `run-heavy` | 0, "preflight passed", golangci-lint "0 issues" | 10:47:13 to 10:55:21 |
  | `make race` | the Windows laptop (native, not a WSL distro), by `run-on-host --race` | 0, `llm/provider` ok | 10:55:21 to 10:59:49 |
  | `make check-test-isolation` | the Linux machine, by `run-on-host` | 0, "0 failing", "files written above the temp roots: 0" | 10:59:49 to 11:01:28 |
  | `make check-records markdownlint` | this Mac | 0, "Summary: 0 issues" | 11:01:28 to 11:01:32 |

  The Linux machine's preflight is not used as evidence for this change. The preflight's one lint draw is not the
  evidence either: the 12 fresh-cache runs are.
* **Last-modified times:** `llm/provider/login.go` 09:51:33 (unchanged by this fix), `llm/provider/export_test.go`
  10:42:39, `llm/provider/login_fields_test.go` 10:42:39.
* **Not verified:** whether a lint run of the whole tree, not `./llm/provider/...`, is stable 12 of 12; whether the
  Linux lint cache was warm on 2026-11-15 at 10:07; CI on the fixed commit, which a push is needed to show.
* **Note (2026-11-16): the root cause is completed in
  [0059-MADR-stop-excluding-confusing-naming-in-test-files.md](0059-MADR-stop-excluding-confusing-naming-in-test-files.md).**
  The finding is dropped, not only intermittent, when revive reports it on the test file: `.golangci.yml`'s exclusion of
  revive from `_test.go` paths removed it in the runs where the second name was the test file's, and a warm cache keeps
  whichever outcome its first run had. 0059-MADR D1 stops excluding `confusing-naming` in test files, which makes the
  verdict order-independent. The text above is unchanged.

### P3 record (2026-11-15)

Run by a mainstream model at medium effort, on the owner's approval of this PLAN's P3, as ONE change set with
[0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
P4 ("P4 record (2026-11-15)"), which holds the start state, the gates, the six lint runs, the files' times and the deviations
of both: the plan is offered only in the commit that honours its consent, and the two are staged together on the branch
`p3-acpext-v0.5.0`, after 0051-PLAN P3's commit and against `acpext/v0.5.0-rc.1`. Staged, not committed: the commit is made on
the branch by the coordinator, and `main` is fast-forwarded by the coordinator alone, only after it has verified the gates. *(2026-11-15, the owner's ruling relayed by the coordinator, "Keep it on the branch until those phases land": the coordinator commits each approved phase on the branch `p3-acpext-v0.5.0` and pushes the branch; `main` stays at `9e21c8f` and is fast-forwarded only after 0051-PLAN P6 and 0052-PLAN P4 have landed on the branch and every gate is green, because until then a plan sign-in passes no host id and does not reuse its registered client. Consequence told to and accepted by the owner: CI runs on pushes to `main`, so until `main` moves the phases on the branch have only the local gates and the Linux and Windows runs behind them; `.github/workflows/ci.yml` also runs on `pull_request` and on `v*` tags, which a pull request from the branch would use.)* *(2026-11-15, later: the owner's ruling "Open a draft PR now", relayed by the coordinator. The coordinator's statement that CI runs only on pushes to `main` and tags was wrong: `.github/workflows/ci.yml` also runs on `pull_request`, which the review's finding showed. After P4 is committed and pushed the coordinator opens one draft pull request from `p3-acpext-v0.5.0` to `main`, so every later push to the branch runs CI; `main` does not move and nothing merges until the owner's word, and `main` is still fast-forwarded only after 0051 P6 and 0052 P4 land. The sentence above that the phases on the branch have only the local gates behind them holds only until that pull request is open.)*

**What changed (staged),** by step.

1. **Policy.** `honouredConsents` is `openai.ConsentBackgroundUse` (`llm/provider/policy.go`), so `Offered("openai")` gives `chatgpt_plan`
   and `HasLogin("openai")` is true; `Offered` sets the consent's `Text` to `ConsentText`'s wording and keeps the SDK's usage link.
2. **Step 1.** `llm/provider/consent.go` (new): `ConsentText` (the WIP's six lines, the fifth naming `gobble auth consent openai background deny`,
   then the question), passed to `wizard.Options`; `ConsentState{Consent; Plan, Granted}` and `PlanConsent(id, store) (ConsentState, error)`,
   the id read from the offered method's `Consent` fact; `SetConsent(ctx, store, id, consent, granted)` over the SDK's `auth.SetConsent`
   through `sdkStore`, taking the SDK's id or gobble's word `background`; `Consents(ctx, store, id)` listing the stored session's
   method's consents with `granted`, `declined` or `not asked`; `ErrNoStoredSession` and `ErrUnknownConsent`.
3. **Step 2.** `llm/background.go` (new): `WithBackground`, `IsBackground`; `llm/consent.go` (new): `ConsentError{Consent; Err}`, whose `Unwrap`
   gives `Err`. The provider adapter adds `llmprovider.WithBackground` to a marked request's context, and wraps the SDK's refusal of
   a marked request (a `ErrNotPermitted` with no HTTP response behind it) on a plan session in `llm.ConsentError` with the
   consent's id.
4. **Step 3.** `acpserver`: `Options.PlanConsent func(id string) (ConsentState, error)` and acpserver's `ConsentState`;
   `backgroundGate` asks it at each title (`titleWanted`) and each automatic compaction; both mark their context with
   `llm.WithBackground`; `errPlanOverflow` is an `llm.ConsentError` naming `/compact` and
   `gobble auth consent <provider> chatgpt_plan.background_use allow`; an error is no grant (a title and a threshold compaction
   make nothing and log one line; an overflow ends the turn with `errPlanUnreadable`, which gives the store's error, names `/compact`
   and not the command, and is no `ConsentError`); `afterRefusal` reads the answer again when the SDK's refusal of an
   automatic compaction comes back, and decides the table's row; `errorData` sets `ErrorData.Consent` from a `ConsentError`.
   No per-session copy of the answer; no `chatgptPlanBackground` setting.
5. **Step 4.** `internal/cli/auth.go`: `AuthConsentCmd`, `gobble auth consent <provider> [<consent> allow|deny]`: exit 0 stored or
   listed, 2 for each usage error the MADR's note lists, 1 for a store that cannot be read, parsed or locked; one stdout line;
   every error one line on stderr with stdout empty; no prompt, no read of stdin, no `--json`. `credentials.go`'s `planConsent`
   answers `Options.PlanConsent` from `auth.json` through `provider.PlanConsent`; `internal/cli` imports no SDK package for it.

**Tests (C2): each seen failing on a scratch copy,** as 0051-PLAN's "P4 record (2026-11-15)" says for all of them:

| Test | Planted break | Failure seen (exit 1 each; `<TMP>` is the scratch copy) |
| :--- | :--- | :--- |
| `TestTitleUnderPlan` (P3) | the gate reading `true` (the answer taken as granted) | `plan_test.go:88: 1 title requests without consent, want none` |
| `TestAutoCompactUnderPlan` (P3) | `errPlanOverflow`'s old text (`or set chatgptPlanBackground to true`) | `plan_test.go:121: no consent, an overflow: err {"code":-32603,"message":"Internal error","data":{"consent":"chatgpt_plan.background_use","error":"the context is over the model's window, and automatic compaction runs under your Cha…` |
| `TestConsentAppliesFromTheNextRequest` (P3) | the answer cached at the session's start (read once) | `plan_test.go:157: after the withdrawal: 3 summaries, want still one` |
| `TestBackgroundRequestsAreMarked` (P3) | the title's context unmarked | `plan_test.go:218: the title's requests [{summary:false marked:false}], want one, marked` |
| `TestBackgroundRequestsAreMarked` (P3) | the automatic compaction's context unmarked | `plan_test.go:247: summaries: 0 marked and 2 unmarked, want the automatic one marked and /compact's not: [{summary:false marked:false} {summary:true marked:false} {summary:false marked:false} {summary:false marked:false} {summary:t…` |
| `TestMarkReachesTheSDK` (P3) | the adapter dropping the mark | `mark_test.go:86: err llm: api status 400: llmprovider: invalid request: openai HTTP 400: bad request (consent error false), 1 requests sent; want llm.ErrNotPermitted carrying chatgpt_plan.background_use and nothing sent` |
| `TestMarkReachesTheSDK` (P3) | the session handed to the SDK with no `Store` | `mark_test.go:86: err llm: api status 400: llmprovider: invalid request: openai HTTP 400: bad request (consent error false), 1 requests sent; want llm.ErrNotPermitted carrying chatgpt_plan.background_use and nothing sent` |
| `TestAuthConsentCommand` (P3) | `deny` writing nothing (F31's bug) | `consent_test.go:49: exit 0, stdout "openai chatgpt_plan.background_use granted\n", stderr ""; want 0, "openai chatgpt_plan.background_use declined\n", ""` |
| `TestAuthConsentCommand` (P3) | the id not accepted, so `chatgpt_plan.background_use` exits 2 | `consent_test.go:39: allow by id: exit 2, stdout "", stderr "Error: openai's stored sign-in has no consent \"chatgpt_plan.background_use\"\n"` |
| `TestAuthConsentCommand` (P3) | an error printed on stdout | `consent_test.go:79: a provider with none, listed: exit 2, stdout "provider: no stored sign-in to answer a consent for: kilo\n", stderr ""; want exit 2, nothing on stdout, one line naming "has no stored sign-in"` |
| `TestAuthConsentCommand` (P3) | the list printing "not asked" and exiting 0 on an unreadable store | `consent_test.go:79: a provider with none, listed: exit 0, stdout "kilo chatgpt_plan.background_use not asked\n", stderr ""; want exit 2, nothing on stdout, one line naming "has no stored sign-in"` |
| `TestConsentWordingOnEverySurface` (P3) | `ConsentText` dropped from the wizard's options, so the SDK's default text is shown | `consentwording_test.go:135: condition not met within 5 s` |
| `TestBackgroundRefusedWhenAnswerUnreadable` (P3) | an error from `PlanConsent` read as a grant | `plan_test.go:272: 1 title requests with an unreadable answer, want none` |
| `TestBackgroundRefusedWhenAnswerUnreadable` (P3) | the overflow reported with `errPlanOverflow`'s text | `plan_test.go:303: an unreadable answer, an overflow: err {"code":-32603,"message":"Internal error","data":{"consent":"chatgpt_plan.background_use","error":"the context is over the model's window, and automatic compaction runs unde…` |
| `TestConsentErrorIsNotPermitted` (P3) | `Unwrap` dropped, so `Kind` gives `other` | `consent_test.go:15: errors.Is(the turn: llm: not permitted: needs the consent, ErrNotPermitted) is false` |
| `TestRefusedBackgroundCarriesConsent` (P3) | `errorData` not reading `ConsentError`, so `consent` is absent | `plan_test.go:375: error -32603 {"error":"the context is over the model's window, and automatic compaction runs under your ChatGPT plan only with your consent: run /compact, or allow it with gobble auth consent fake chatgpt_plan.ba…` |
| `TestRefusedBackgroundCarriesConsent` (P3) | `errPlanOverflow` a plain error, so `kind` is `other` | `plan_test.go:375: error -32603 {"error":"the context is over the model's window, and automatic compaction runs under your ChatGPT plan only with your consent: run /compact, or allow it with gobble auth consent fake chatgpt_plan.ba…` |
| `TestRefusedBackgroundCarriesConsent` (P3) | the unreadable-answer error built as a `ConsentError`, so `consent` is set | `plan_test.go:375: error -32603 {"consent":"chatgpt_plan.background_use","error":"the context is over the model's window, and the stored answer for background use under your ChatGPT plan could not be read (auth.json: unreadable): r…` |
| `TestOfferedConsentTextIsGobbles` (P3) | `Offered` passing the SDK's default text through | `policy_test.go:153: the consent {ID:chatgpt_plan.background_use Required:true Text:You're using your ChatGPT plan: this app's requests count toward its usage, which you can manage on the usage page. Allow this app to also use your…` |
| `TestUnhonouredConsentKeepsMethodOut` (P3 (extra)) | `kept` ignoring a required consent | `policy_test.go:175: a method requiring a consent gobble does not honour was kept` |

`TestTitleUnderPlan` and `TestAutoCompactUnderPlan` run with and without a grant; `TestConsentAppliesFromTheNextRequest` for a compaction
and a title; `TestBackgroundRequestsAreMarked` for a title, an automatic compaction (marked), a user's turn and `/compact` (unmarked);
`TestBackgroundRefusedWhenAnswerUnreadable` for a title, a threshold compaction and an overflow, with `auth.json`'s stand-in
unchanged; `TestRefusedBackgroundCarriesConsent` for the table's two rows and the three variants of an SDK refusal;
`TestAuthConsentCommand` with stdin closed (`Main` is given no stdin) and no terminal, over `allow` and `deny` by id and by word, each usage
error, and an `auth.json` that does not parse; `TestMarkReachesTheSDK` over no answer, declined, granted, a grant whose store then
fails, an unmarked request and an API key; `TestConsentWordingOnEverySurface` below. On the tree before the phase, none of these
packages compiles with the new tests (0051-PLAN's record quotes the errors).

**Deviations** (0051-PLAN's "P4 record (2026-11-15)" has 1 to 7; these are this phase's).

8. **`TestConsentWordingOnEverySurface` against a recording issuer, not `planIssuer`, and each terminal surface cancels at the question.**
   `planIssuer` is `llm/provider`'s test helper and no other package can import it. The test uses a recording transport (`planRun`) that
   answers every request with an error, so the wizard goes on from the SDK's built-in endpoints for OpenAI, and asserts that nothing but a
   discovery request was made. `gobble auth login` and `gobble acp --login` are driven on a terminal stdin through `loginWith` and
   `acpLogin`, `gobble setup` through `signIn` with a scripted `questions` (what `SetupCmd.Run` calls after its terminal check), `/login`
   and the first run through the line harness. Each answers "Cancel the sign-in" at the consent's question, so no URL opens: an
   attempt to let one sign-in go on to its URL left the SDK's loopback listener and the paste read waiting. The test asserts
   gobble's wording was asked, as the wizard's title, on each, with no URL, and with no terminal, the same two commands put the
   wording to their prompter, which refuses with the no-terminal error, and `auth.json` is not written. The PLAN's plant,
   "`/login` built without `ConsentText`", has no such form: all five surfaces call `signInTo`, so the wording is set in one place,
   `provider.SignIn`; the plant is `ConsentText` dropped there, which fails the test.
9. **`TestConsentAppliesFromTheNextRequest`'s "second `Logins` on the same file".** `acpserver` imports no `internal/auth`, so the
   test's `PlanConsent` reads a file the test changes through another writer: the same property, the answer read at each
   request and in every process, with the plant "cached at the session's start".
10. **The consent command in the overflow's text is the SDK's id,** `gobble auth consent openai chatgpt_plan.background_use allow`, not gobble's word
    `background`: `acpserver` holds no alias table, and the id is accepted wherever the word is.
11. **`gobble auth consent` lists nothing for a stored session whose method has no consent** (Grok's), exit 0, as "one line per consent
    of the method" says; and it does not call `ensureSettingsFile`, so a failed run writes nothing.
12. **The gate is asked when an automatic compaction would otherwise be made,** not at every model call: the threshold is
    evaluated first. `afterRefusal`'s read is the MADR's "fresh `PlanConsent` read when it comes back".

**Docs step (C9),** with 0051-PLAN's: the same list, in "P4 record (2026-11-15)". `docs/guides/credentials.md` gained the ChatGPT plan
sign-in, its consent wording and where it is asked, the background requests it gates, `gobble auth consent`, and the `--json`
error kinds, and lost the sentence that said the sign-in came later.

**Acceptance.** The tests pass and each plant failed (above); every C6 gate is in 0051-PLAN's table. Met.

#### Review of 2026-11-15

The independent review's findings that are this phase's, with the owner's rulings in 0051-PLAN's "P4 record (2026-11-15)", "Review of 2026-11-15" (its plants,
gates and the files' times are there too). Nothing above is rewritten.

**Finding 1, no test exercised the real consent read: true.** The reviewer's plant, `Granted: granted || true` in `PlanConsent`, passed nine tests. New
`TestPlanConsentIsReadFromAuthJSON` (`internal/cli/consentstore_test.go`) writes a real `auth.json` under a scratch `GOBBLE_HOME` and reads it through
`runEnv.planConsent` and through `agentOptions().PlanConsent`, which must be set: a plan session with a granted, a declined and no answer, an `auth.json` that
does not parse (an error), a key entry, another application's OpenAI session that holds a granted answer (not a plan), and a provider with no such consent. Plants: the
reviewer's, `credentials.go`'s `planConsent` giving `Granted: true`, and `agentOptions` not setting `PlanConsent`; each fails it.

**Finding 3, one overflow refusal lacked the MADR's text: true.** 0052-MADR D2's table, row 1: "the gate refused an overflow's compaction for want of a grant (`errPlanOverflow`),
or the SDK refused one the gate let through and gobble's own read then finds no grant | `not_permitted` | `chatgpt_plan.background_use` | `errPlanOverflow`'s text: it
names `/compact` and the command". `afterRefusal`'s no-grant branch returned the SDK's error, wrapped by `overflowFailed`; it now returns `errPlanOverflow`, and the
overflow's own refusals (`errPlanOverflow`, `errPlanUnreadable`) are returned unwrapped, so `ErrorData.Error` is exactly the text. `TestRefusedBackgroundCarriesConsent`'s second
case asserts that text, in full, where it asserted the SDK's. *Does the row ask for a title case?* Yes: the row ends "a title not made sends the client nothing"; I had put it
in `TestTitleUnderPlan`. New `TestTitleNotMadeSendsNothing` holds it for a declined and an unreadable answer (the prompt ends as it would without a title, no title request, no
`session_info_update`).

**Finding 5, the wording test did not prove what `auth login` and `acp --login` write: true.** With `prompter.Choose` replacing the wording by "Allow?", only `/login`, the first
run and `setup` failed. The two surfaces' subtests now read what the surface wrote to its stderr (`wrote`) and require gobble's wording in it; they fail under the plant. The "no
terminal" subtest stays as it was: with no terminal the surface's `unasked` prompter writes nothing and fails closed, which the subtest asserts, and no wording is claimed
for it. No real terminal was needed: `loginWith` and `acpLogin` run on a terminal stdin and the test's stderr buffer.

**Finding 7, `TestOfferedConsentTextIsGobbles` lacked its row's second half: true.** Built: a first plan sign-in is run through `SignIn` against a transport that answers
nothing, the test cancels at the confirm step, and the question the wizard asked is required to be `Offered`'s consent text plus its usage link, which is where the wizard puts
the step's text and link. Plant: `ConsentText` dropped from the wizard options. *What step 1 says, what was built, whether it is a deviation:* step 1 says "`ConsentText` …
passed to `wizard.Options` and `signin.Options`". Built: `wizard.Options` only. `signin.Options` is constructed nowhere in this tree: its site is 0051-PLAN P6 step 1
(`StartSignIn`'s `signin.Options`, whose line there already names "`ConsentText` gobble's wording (0052-PLAN P3)"). So it is a deviation by omission, **deviation 13**: the second
site cannot exist before P6, which builds it; `TestConsentWordingOverACP` of 0052-PLAN P4 is its guard. Resolutions: (a) recommended: accept, P6 builds the site (cost: none); (b) a
stand-in `signin.Options` constructor here (code nothing calls, no test can fail). Doing nothing: none, while P6 does its step. For the owner, through the coordinator.

**Finding 9, the title gate read `auth.json` on every prompt: true.** `titleWanted` asked the gate before it checked the attempt and the name. It now checks them first and
asks the gate only when a title would otherwise be asked, so a gated first prompt does not use the attempt and the prompts after the first read nothing. New
`TestTitleGateReadsOnlyWhenATitleIsAsked` counts the gate's reads across three prompts of a session, for a granted and a declined answer: once. Plant: the gate asked first, 3 reads.

**Finding 10, `auth consent` could exit 1 after the answer was stored: true.** 0052-MADR D3: exit 1 is "the store cannot be read, parsed or locked … nothing written". The command read
`Consents` again after `SetConsent` only to map the word to the id. It now reads the consents once, before the write, finds the id there, writes by it and prints it. The store is a
seam (`consentStore`) so `TestAuthConsentNeverFailsAfterTheWrite` can make the file unreadable once a login was saved: exit 0, one stdout line, the answer stored. Plant: the second read restored, exit 1.

**Q-a, what does 0052-MADR D2 say the gate keys on?** "`acpserver.Options.PlanConsent func(id string) (plan, granted bool)` replaces `PlanSession` [WIP] and the per-session setting;
`internal/cli` answers it from the stored session (`provider.PlanConsent`, reading `auth.json` through `Logins` and the SDK's `Consent`). `planGated` asks it at every background
request". It keys on the stored session, not on the credential the request will use. The WIP's `PlanSession` keyed on the resolved credential (`id == "openai" && e.knownLogin(id)`) and
the MADR's table lists it as reworked, so that is a decision, and the code matches it: with `--api-key` and a plan session stored, the gate still reads the session. The SDK's own refusal
is narrower (a marked request on a plan session only). The MADR is silent on the key case itself; nothing changes.

**Q-c, deviation 9: is it a fact that `acpserver` may not import `internal/auth`?** No. `internal/archtest/check.go` has no rule that keeps `acpserver` from `internal/auth` (its rules for
`acpserver` are 2, ACP SDK; 4, the `authcontract` exception; and 12, acpext), and there is no import cycle: `go list -deps ./internal/auth` names `internal/filelock`, `internal/fsx`,
`jsonx`, `llm`, `llm/catalog` and `llm/provider`, none of which imports `acpserver`; `acpserver`'s tests already import `llm/provider`. Deviation 9's stated reason, "acpserver imports no
`internal/auth`", described what was imported, not a rule. **Deviation 9 is withdrawn:** `TestConsentAppliesFromTheNextRequest` now does what the PLAN says, with a real `auth.json`
under `auth.Logins`, `provider.PlanConsent` as the gate and a second `auth.Logins` withdrawing the answer (`provider.SetConsent`), for a compaction and a title; plant: the answer cached,
3 summaries where one is expected. It is not accepted as a deviation because it no longer exists.

**Deviations of this review:** 13, above. Deviation 8 (accepted subject to finding 5) is closed by finding 5. Deviations 10, 11 and 12: accepted by the owner on the independent review, as
the "Review of 2026-11-15" of 0051-PLAN's P4 record lists. Deviation 12 now also holds for the title (finding 9).

**Docs step (C9).** `docs/architecture.md` (`Remote`, `ErrorKind`) fixed; no guide or README sentence says Grok's browser sign-in is unavailable to a remote client or needs the agent's host:
`docs/guides/credentials.md:192` and `:205` say, for Grok and the ChatGPT plan, that on a machine whose browser cannot reach gobble the address the browser ends on is pasted, which is true at the
pinned facts. `gobble auth consent`'s section is unchanged (the id is mapped before the write, the exits and the output are as written).

#### Second review of 2026-11-15

The second-pass review's items that are this phase's, with the rulings, the plants and the gates in 0051-PLAN's "P4 record (2026-11-15)", "Second review of 2026-11-15". Nothing above is rewritten.

**Rulings** (the owner's, relayed by the coordinator): deviation 13, `ConsentText` to `signin.Options`: "Accept; P6 builds the site", written at step 1 above and at 0051-PLAN's P6 step 1; a key for the run
with a plan session stored: "Keep it, and document it", at 0052-MADR D2's dated note and in `docs/guides/credentials.md`; CI on the branch: "Open a draft PR now", and the annotations of the
sentences in this record that said the branch had only local gates behind it. The coordinator's check of deviations 6 and 9 under "Accept the eight; I check 6 and 9": 6 stands, 9 is withdrawn.

**Item 6, the granted-yet-refused overflow case was tested with gobble's own `refusal`: true.** In `TestRefusedBackgroundCarriesConsent` the stand-in is now the shape production gives acpserver for an
SDK refusal of a marked request: the SDK's text (go-llmprovider-sdk's `openai.checkBackground`: "not permitted: openai: a background request on a ChatGPT plan session needs the consent
chatgpt_plan.background_use, which the user has not granted (0075-MADR D5)"), wrapped by `llm/provider`'s `mapErrorAt` as `llm.ErrNotPermitted` ("llm: not permitted: …"), and by the adapter's `attempt` in an
`llm.ConsentError` with the consent's id (`llm/provider/provider.go`). The case asserts the whole text production gives for it, "the context is over the model's window, and compacting it failed: llm: not
permitted: llmprovider: not permitted: openai: …", where the substring check was. Plant: every overflow error returned unwrapped, which fails it (0051-PLAN's table).

**Q-a, the sentence in `docs/guides/credentials.md`** (in "Allowing background requests under a ChatGPT plan", after the paragraph on reading the answer at each request), verbatim: "It is the stored
sign-in's answer, whichever credential a run uses: with `--api-key -` for OpenAI while a ChatGPT plan sign-in is stored, titles and automatic compactions are held back until the consent is granted, although
those requests would go out with the key. An OpenAI key in the environment or in `models.json` is not used while that sign-in is stored, since `auth.json`'s entry owns the provider, so those requests go
out under the plan and the consent governs them as above." The code behind it, checked for each way a key reaches a run, is in 0052-MADR D2's note: `internal/cli/chat.go:69`, `internal/auth/resolve.go:77`, `:80-82`,
`:83`, `:94` and `Has` at `:109`, `:112`, `:115`, `:123`, `llm/provider/provider.go:99`, `internal/cli/credentials.go:22`, `internal/cli/agent.go:103`. The behaviour differs only for `--api-key`; an environment
variable and `models.json`'s key are not reached while an `auth.json` entry owns the provider; `settings.yaml` has no key that carries a credential.

### P4 record (2026-11-16)

Run by a mainstream model at medium effort, on the coordinator's brief, on the owner's approval of this PLAN's phases (the owner's statements this
PLAN and 0051-PLAN already quote; none is added here). Base: the branch `p3-acpext-v0.5.0` at `cecc892`, level with origin, tree clean; `go.mod`
requires acpext `v0.5.0-rc.1` and go-llmprovider-sdk `v1.31.0`, no `replace`, no `go.work`; `main` at `9e21c8f`. Staged, not committed; the
coordinator commits on the branch. P4 starts from P2, P3 and 0051-PLAN P4 to P7, all committed on the branch.

**Question 1, the acpext pin.** Settled by the records, so nothing here touches `go.mod` or `go.sum`. Step 1's "the acpext pin is 0051's `v0.5.0`"
was written before the candidate: 0051-PLAN C8's note of 2026-11-15 ("The root requires `acpext@v0.5.0-rc.1` (P4 step 1), by tag, with no `replace`
and no `go.work`. The final `acpext/v0.5.0` is the owner's, after P8 (P8a)", 0051-PLAN lines 131 to 136), the owner's ruling quoted at 0051-PLAN
`:1511-1512` ("P4 whole with 0052's consent phase is built on the branch against that candidate"), and P8a (`:780-796`: "One commit moves the root
from the candidate to the final tag") make the pin `v0.5.0-rc.1` until P8a, and 0051-PLAN P4 step 1 already made it (this PLAN's P3 note at the head
of "P3 — consent" says the same). The pin is not a step of this phase. `git diff` shows no change to `go.mod`, `go.sum` or `acpext/`.

**What was already true at `cecc892`** (each found by reading, then by running):

* `TestConsentWordingOnEverySurface`: `internal/cli/consentwording_test.go:132` (the five terminal surfaces, against a recording issuer; deviation 8 of the P3 record).
* The engine-level half of `TestSignInChatGPTPlan`, `TestSignInChatGPTPlanPasted` and `TestPlanAfterLogoutReusesClient`, over `planIssuer` and the real engine: `llm/provider/signinrun_test.go:149`
  (`TestStartSignInReusesClient`: after a logout that keeps `clients.json` the next sign-in names the stored client and the host id and sends no `agent_name_hint`), `:174` (`TestStartSignInPassesPrevious`: a second sign-in
  holds no `confirm`), `:204`, `:232` (`TestStartSignInConfirmIsGobblesWording`), `:323` (`TestHostIDReachesTheEngines`). `drive` (`:76`) completes `StartSignIn`'s engine, the ACP path's, by paste. *(2026-11-16, corrected in the review below: the first text said that `drive` "is `TestSignInChatGPTPlanPasted`'s sign-in". It is not: the archived tests ran `SignIn`, the terminal wizard behind `gobble auth login openai`, which no test completed; see "Review and fixes", B-1.)*
  What no test held was the surface above them: the host id read from `<state>/host-id`, the client in `<state>/clients.json`, and the logout of the CLI.
* `TestLogoutPlan`'s content: `llm/provider/login_test.go:457` (`TestLogoutRevokesAsGobble`, the openai case: discovery and revocation under gobble's name), `:512` (`TestLogoutRevokesOnlyAccepted`: a plan session revoked then deleted, a foreign OpenAI login deleted and never revoked),
  `:555` (`TestLogoutReadsTheStoredMethod`). Not rebuilt.
* `TestOpenAILoginIsAPlan`'s content, at the resolver: `internal/auth/resolve_test.go:153` (`TestResolverAcceptsOnlyOfferedKinds`), `llm/provider/policy.go:172` (`Accepts`). The WIP's refusal inside `Options.login` is not there and is not coming back (question 3).
* The status row already had `oauth_unused` for a session the resolver does not accept (`internal/cli/auth.go` `statusRow`, golden: `claude`), and the text form's "not used (0005-PLAN F4-3b)". It had none of `sessionKind`, `consents` or `consentTerms`.
* `auth consent`, `provider.Consents`, `SetConsent`, `PlanConsent`, `ConsentText`: P3.
* `TestConsentWordingOverACP` did not exist.

**What was built (staged),** by step.

1. **Step 1, the tests re-applied.** *(Renamed in the review: the ACP path's are `TestSignInChatGPTPlanACP` and `TestPlanAfterLogoutReusesClientACP`, `internal/cli/plansignin_test.go:130`, `:192`; the terminal's `TestSignInChatGPTPlanTerminal` and `TestPlanAfterLogoutReusesClientTerminal` are new, below.)* The ACP path's two tests are built on the CLI's own `startSignIn` (`internal/cli/agentauth.go`, `hostSignIn` in `login.go`) so the host id and the client store are
   production's; the engine is `provider.StartSignIn` over an issuer in process answering as OpenAI's (`loginissuer_test.go`: `newPlanLoginIssuer`, `newLoginIssuerAs`, and a revocation endpoint that records the token), the browser's side a paste (`planSignInRun`), remote, so no listener and no port but the issuer's own `httptest` one.
   `TestForeignOpenAISessionRefused` (`internal/cli/statusplan_test.go:266`). `TestLogoutPlan`: already true, above. *(`TestSignInChatGPTPlanPasted` was recorded here as already true; that was false, see "Review and fixes", B-1: its terminal form is `TestSignInChatGPTPlanTerminal`.)* The acpext pin: question 1.
2. **Step 2, the status row.** `internal/cli/auth.go`: `authStatusRow` gains `SessionKind`, `Consents`, `ConsentTerms` (`:587`), set by `sessionFacts` (`:648`, called from the oauth case, `:625`) from the stored session. `llm/provider/signinrun.go:71` `SessionKindOf(id, *Login) string`, `(*OAuthSession).SessionKind()`'s value (see "Questions for the owner": the one new exported name).
   A session the resolver does not accept is `usable: false`, `oauth_unused` as before, with `sessionKind` and no consent. The text form ends a plan session's cell with "background use allowed", "background use declined" or "background use not asked" (`consentWords`, from the consent's id). The golden `auth-status.json.golden` regenerated by `go test ./internal/cli -run '^TestAuthStatusJSON$' -update` (`main_test.go:23`, `golden`), and a second golden, `auth-status-plan.json.golden`, new.
3. **Step 3, docs.** `internal/cli/auth.go`'s help for `auth status` and `--json`; `docs/guides/credentials.md` (the status section: the text form's consent words, `sessionKind`, `consents`, `consentTerms`, an example row; the ChatGPT passage: the status and the phone; the ACP table: the `confirm` is first and is gobble's wording); `README.md`; `docs/architecture.md` (`llm/provider`'s `SessionKindOf`; `internal/cli`'s status row). Rows below.
4. **Step 4, tests.** `TestAuthStatusJSONPlan` (`statusplan_test.go:90`), `TestConsentWordingOverACP` (`acpserver/authconsent_test.go:28`), the three above.

**Question 3, `TestForeignOpenAISessionRefused`: where is each decision made?** One predicate. `provider.Accepts` (`llm/provider/policy.go:172`, `storedMethod`: the session's kind is the kind an offered method signs in to) is set as `Resolver.Accepts` (`internal/cli/credentials.go:60`, `auth.AcceptsLogin`, `internal/auth/resolve.go:38`), and read in `Resolver.Has` (`resolve.go:113`) and `Resolver.stored` (`:137-140`).
Status (`statusRow`: `Has`), `auth check` (`check`: `Credential`, an error for a foreign entry, so `invalid`, exit 2), `authenticate`'s present credential (`authOptions().Has`, `agentauth.go`: `credential`), the choice of provider (`ChooseWith(e.hasCredential)`) and the provider build (`modelProvider` and `providerFor`: `AmbientWith(e.credentialKey, ...)`, which finds no credential) all go through it. The logout is the sixth reader: `provider.Logout` (`login.go:102`) revokes only when `storedMethod` finds a revocable method.
What does not share it: `provider.New` / `Options.login` (`llm/provider/provider.go:98`) checks `HasLogin(id)` and loads whatever session is stored, with no kind check; the WIP's check there (`1406bd6:llm/provider/provider.go`) was taken out by P1 and its job moved to the resolver by 0051-MADR D2. Every route into `New` from gobble's code is gated by the resolver, so this is no gap for gobble; a caller outside the CLI that builds `New` with a `LoginStore` holding a foreign session would not be refused.
Making `New` share it is code beyond P4's steps (a new check in `llm/provider/provider.go` and its test), so it is not built; see the questions. The test holds the six surfaces as production wires them, and the plant "status accepts while the build refuses" fails it.

**Question 4, `consentTerms` and the words.** One source. `provider.ConsentText` (`llm/provider/consent.go:37`) is set on `Offered`'s consent (`policy.go:73`), on the wizard's options (`signin.go:361`) and on `StartSignIn`'s (`signinrun.go:95`). `consentTerms` is `provider.SignedInMethod(id, l).Consents` (`auth.go` `sessionFacts`), which is `Offered`'s method, so its `text` is `ConsentText`'s with the SDK's links kept. The `confirm` step of a sign-in carries `ConsentText` directly. The tests hold that they are the same words:
`TestAuthStatusJSONPlan` compares the status row's `text` with `provider.ConsentText` and requires the command that withdraws the consent, which the SDK's default lacks; `TestOfferedConsentTextIsGobbles` (P3) holds `Offered`'s against the real `confirm` step; `TestSignInChatGPTPlan` holds the first sign-in's `confirm` step `Text` against `ConsentText`; `TestConsentWordingOverACP` holds the ACP `confirm` form's message and `_meta.gobble.step.text`.
Plants: `terms-text-sdk-default` (below) fails the status test, `start-sign-in-without-consent-text` the ACP test.

**Question 5, the golden.** Regenerated by `go test ./internal/cli -run '^TestAuthStatusJSON$' -update`, the repository's own flag (`internal/cli/main_test.go:23`). The diff of `internal/cli/testdata/auth-status.json.golden`, whole:

```diff
@@ -25,7 +25,8 @@
       "source": "auth.json",
       "authType": "oauth",
       "usable": true,
-      "expires": "2026-06-09T10:13:20Z"
+      "expires": "2026-06-09T10:13:20Z",
+      "sessionKind": "grok"
     },
```

Line by line: the first `+` line is the old line with a comma, since a member follows it; the second is the Grok row's `sessionKind`, `grok`, inferred by the SDK because the fixture's entry stores no `kind` (step 2). The Claude row, which the resolver does not accept, gains nothing: the SDK infers no kind for it. Nothing else changed. The second golden, `auth-status-plan.json.golden`, is new: a plan session with no answer (`sessionKind`, `consentTerms` with gobble's wording and the usage link, no `consents`) and a Grok session (`sessionKind` alone), the other rows `none`.

**Tests, each seen failing on a scratch copy** (`git ls-files -co`, no `.git`; `HOME`, `USERPROFILE`, `XDG_*`, `GOBBLE_HOME` inside the scratch; `GOPROXY=off`; no proxy variable; the module and build caches the real ones; driver `~/.cache/claude-session-5a7234fa/scratch/gobble-0052-p4/plants.py`). Exit 1 and a `--- FAIL` each, never a build failure; the unbroken runs of four of the plants' test sets exit 0 (`.clean` rows).

| Test | Planted break | Failure seen (`<scratch>` is the scratch copy) |
| :--- | :--- | :--- |
| `TestSignInChatGPTPlanACP` | the host id not passed (`agentauth.go`: `HostID: hostID[:0]`) | `plansignin_test.go:151: the first authorize query map[agent_name_hint:[gobble] client_id:[dynamic_agent_client] … ext_agent_host_id:[urn:uuid:…]] (the SDK made an id of its own), want gobble's name and the host id` |
| `TestPlanAfterLogoutReusesClientACP` | `clients.json` cleared by the logout (`auth.go`) | `plansignin_test.go:226: clients.json after the logout holds {ClientID: HostID:}, <nil>; want the issued client kept` |
| `TestAuthStatusJSONPlan` | `Usable` from `Has` without `Accepts` (`row.Usable = true` in the oauth case) | `statusplan_test.go:178: the foreign row {… Usable:true Reason:oauth_unused …}, want usable false, oauth_unused, sessionKind chatgpt` |
| `TestAuthStatusJSONPlan` | `consentTerms` left out while the consent is unanswered | `statusplan_test.go:111: consentTerms [], want the one consent of the plan's method` |
| `TestAuthStatusJSONPlan` | its `text` the SDK's default (`Offered` not setting `Text`) | `statusplan_test.go:115: consentTerms[0] {ID:chatgpt_plan.background_use Required:true Text:You're using your ChatGPT plan: this app's requests count …` |
| `TestAuthStatusJSON` (golden) | `SessionKindOf` returning the stored `Kind` only, not the SDK's inferred kind | `auth_test.go:152: auth-status.json differs from testdata/auth-status.json.golden` |
| `TestForeignOpenAISessionRefused` | the provider build refusing while status accepts (status's `Usable` forced true) | `statusplan_test.go:253: status: usable true, reason ""; want not usable, oauth_unused` |
| `TestForeignOpenAISessionRefused` | every surface accepting (`Accepts` always true): six `Errorf`s, status, `auth check` (exit 0), authenticate, choice, run build, session build | `statusplan_test.go:253, :256, :259, :262, :265, :268` |
| `TestForeignOpenAISessionRefused` | the logout revoking a foreign session (`provider.Logout`) | `statusplan_test.go:283: the logout of a foreign session sent ["POST https://auth.openai.com/oauth/revoke"], want nothing` |
| `TestConsentWordingOverACP` | `StartSignIn` built without `ConsentText` | `authconsent_test.go:73: _meta.gobble.step.text "You're using your ChatGPT plan: this app's requests count toward its usage, …" (the SDK's default), want gobble's wording` |

The tests of the table's six rows are `TestSignInChatGPTPlan`, `TestPlanAfterLogoutReusesClient`, `TestAuthStatusJSONPlan` (twice: its first row and its extension), `TestForeignOpenAISessionRefused` and `TestConsentWordingOverACP`; the first two are `...ACP` and `...Terminal` in the review below, and `TestAuthStatusJSON` is the golden's test, not a row. Line numbers are those of the tree at the plant run after the lint fix (the first run, before `TestAuthStatusJSONPlan`'s subtests were moved into functions, gave the same failures at lines 5 earlier). Unbroken runs of the tests in four of the plants, on the same copy, exit 0.

**Earlier plants re-run** (this phase changed `llm/provider/signinrun.go` and `internal/cli/auth.go`; the drivers of 0051-PLAN P6's fix and P7, with their scratch path changed; each plant whose edit touches either file):

| Plant (driver) | Exit | Test that failed |
| :--- | :--- | :--- |
| `consent-text-omitted` (0051 P6 fix) | 1 | TestStartSignInPassesConsentText |
| `client-store-nil` (0051 P6 fix) | 1 | TestStartSignInReusesClient |
| `previous-never` (0051 P6 fix) | 1 | TestStartSignInPassesPrevious |
| `no-loopback-false` (0051 P6 fix) | 1 | TestStartSignInRemoteSetsNoLoopback |
| `identity-zero` (0051 P6 fix) | 1 | TestStartSignInNamesGobble |
| `host-id-not-copied` (0051 P6 fix) | 1 | TestHostIDReachesTheEngines |
| `start-deletes-stored` (0051 P6 fix) | 1 | TestStartSignInStoresOnlyInWait |
| `b1-save-original-session` (0051 P6 fix) | 1 | TestStartSignInSavesTheRenewedSession |
| `b2-refuse-403` (0051 P6 fix) | 1 | TestStartSignInChecksListingBeforeSaving |
| `b2-fail-on-list-error` (0051 P6 fix) | 1 | TestStartSignInSavesTheRenewedSession |
| `b2-fail-on-any-listing-error` (0051 P6 fix) | 1 | TestStartSignInChecksListingBeforeSaving |
| `b2-acp-403-refused` (0051 P6 fix) | 1 | TestAuthenticateRefusedListingStoresNothing |
| `b2-refusal-ignored` (0051 P6 fix) | 1 | TestStartSignInChecksListingBeforeSaving |
| `b2-renewal-refusal-ignored` (0051 P6 fix) | 1 | TestStartSignInRenewalOutlivesTheListing |
| `b3-no-ctx-check-before-save` (0051 P6 fix) | 1 | TestStartSignInCancelledDuringListingSavesNothing |
| `b3-no-ctx-check-at-all` (0051 P6 fix) | 1 | TestStartSignInCancelledDuringListingSavesNothing |
| `b3-acp-no-ctx-check` (0051 P6 fix) | 1 | TestAuthenticateCancelledDuringListingStoresNothing |
| `kilo-skip-restored` (0051 P6 fix) | 1 | TestStartSignInChecksKiloDeviceKey |
| `plan-listing-skipped` (0051 P6 fix) | 1 | TestStartSignInChecksPlanListing |
| `load-error-unwrapped` (0051 P6 fix) | 1 | TestStartSignInErrorsAreSignInErrors |
| `no-loopback-not-set-for-device` (0051 P6 fix) | 1 | TestStartSignInRemoteSetsNoLoopback |
| `wait-error-unwrapped` (0051 P6 fix) | 1 | TestStartSignInErrorsAreSignInErrors |
| `plan-login-deleted-before-save` (0051 P6 fix) | 1 | TestStartSignInPlanFailureLeavesAuthJSON |
| `auth-login-no-opener` (0051 P7) | 1 | loginsurfaces_test.go:68: |
| `expiry-dropped` (0051 P7) | 1 | loginsurfaces_test.go:130: |
| `expiry-in-utc` (0051 P7) | 1 | loginsurfaces_test.go:130: |
| `expiry-printed-when-zero` (0051 P7) | 1 | loginsurfaces_test.go:155: |

All 27 exit 1 with a `--- FAIL`, none a build failure; 23 of 0051-PLAN's P6 fix plants (each whose edit touches `llm/provider/signinrun.go`) and 4 of its P7 plants (each whose edit touches `internal/cli/auth.go`). The other P6 and P7 plants touch neither file.

**The acceptance's grep,** `grep -rn 'chatgptPlanBackground\|askPlanBackground\|planNotice' --include=*.go .`: exit 0, three lines, all `llm/provider/consent.go`'s `planNoticeLines` (`:15`, `:20`, `:39`), a P3 name for the notice's lines that contains `planNotice` and is not the WIP's `planNotice`. `chatgptPlanBackground` and `askPlanBackground` are found nowhere. As the acceptance is written it is not met; see the questions.

**Gates (C6),** after the last change to a Go file; each with start, end and exit status taken before any filter; `<scratch>` is this phase's scratch folder:

| Gate | Start | End | Exit |
| :--- | :--- | :--- | :--- |
| pre-add-check | 21:27:15 | 21:27:24 | 0 |
| race | 21:27:24 | 21:28:42 | 0 |
| check-test-isolation | 21:28:42 | 21:30:05 | 0 |
| check-acpext-deps | 21:30:05 | 21:30:05 | 0 |
| linux-race | 21:30:06 | 21:37:06 | 0 |
| windows-race | 21:37:06 | 21:39:33 | 0 |
| preflight | 21:39:33 | 21:47:20 | 0 |

The Linux and Windows rows are `run-on-host --race --need linux` and `--need windows` of `go test -race -count=1 ./...` and `go -C acpext test -race -count=1 ./...` (the Windows laptop's native shell, no WSL distro); `make preflight` is one cold lint run. A first run of every gate, 20:58 to 21:26, ended with preflight exit 2: `gocognit` on `TestAuthStatusJSONPlan` (cognitive complexity 56 of 50, `statusplan_test.go:90`) and `modernize` `embedlit` at `authconsent_test.go:32`, both in this phase's own tests. Fixed by moving two subtests into functions and by naming the `auth.Logins` value, no `//nolint`; the plants of those tests were run again, and every gate above was run again after that last change to a Go file (the first run's other gates were exit 0). Between the two runs the acceptance packages `go test ./internal/cli ./acpserver ./llm/provider ./internal/auth ./internal/archtest` passed.

**Files and times.** Staged files' modification times against the first gate's start:

All staged files were last written before the first gate of the second (final) run, which began at 21:27:15 on 2026-11-16 (the first run began at 20:58:58); the latest is 21:26:36.

| File | Modified |
| :--- | :--- |
| `acpserver/authconsent_test.go` | 21:26:36 |
| `internal/cli/statusplan_test.go` | 21:26:36 |
| `internal/cli/auth.go` | 20:48:02 |
| `docs/architecture.md` | 20:47:55 |
| `README.md`, `docs/guides/credentials.md` | 20:47:36 |
| `internal/cli/plansignin_test.go` | 20:43:49 |
| `internal/cli/testdata/auth-status-plan.json.golden` | 20:42:57 |
| `internal/cli/loginissuer_test.go` | 20:42:06 |
| `internal/cli/testdata/auth-status.json.golden` | 20:40:49 |
| `llm/provider/signinrun.go` | 20:40:08 |

The records (this PLAN, `docs/README.md`) were written after the gates and are checked by `make check-records markdownlint` below.

**Docs rows (C9).**

* `docs/guides/credentials.md` status section (the text form's consent words, the `sessionKind`, `consents` and `consentTerms` rows and an example row) — fixed; this is 0051-PLAN's row "`credentials.md:86-132` (`auth status --json`) — open: 0052-PLAN P4 step 2", which this phase closes (the dated note at that row in 0051-PLAN was added in the review below).
* `docs/guides/credentials.md`, the ChatGPT passage (`gobble auth status` shows the kind and the answer; the phone is asked the same question in the same words before any address; the `confirm`'s message and `_meta.gobble.step.text` are gobble's wording) — fixed; closes 0051-PLAN's rows "`credentials.md:189-300` — open: 0052-PLAN P4 step 3" and "`:200-215` — open for 0052-PLAN P4 step 3: the plan's consent through the phone and `TestConsentWordingOverACP`" (its P6 record, "Still open for 0052-PLAN P4"). The notice's content, the files (`auth.json`, `host-id`, `clients.json`), the command (`gobble auth consent`) and the host id were already there from P3, 0051-PLAN P6 and P7.
* `README.md` (the status sentence) — fixed.
* `docs/architecture.md` (`llm/provider`'s `SessionKindOf`; `internal/cli`'s status row) — fixed. The acpext requirement row stays open for 0051-PLAN P8a.
* `internal/cli/auth.go`'s help for `auth status` and `--json` — fixed. The `auth login` help already names the plan.
* `docs/README.md` rows 0052 PLAN — updated for this record.
* Searched: `README.md`, `AGENTS.md`, `docs/architecture.md`, `docs/guides/` (`credentials.md`, `developing.md`, `models-json.md`), `docs/README.md`; no other statement names the status row's members or contradicts them.

**Not verified.** No live provider, no real browser and no phone: every sign-in here runs against an issuer in process, the browser's part a paste or a scripted elicitation, so the real OpenAI sign-in, its listing check and the real phone are L4 to L6 of P6. The Windows run is the laptop's `go test -race ./...` of the root and `acpext`, not a WSL distro's.

**Questions for the owner** (each with evidence and three options in the phase's return): a new exported name, `provider.SessionKindOf`; the acceptance's grep against `planNoticeLines`; `provider.New` having no kind check.

Staged, not committed; the coordinator commits on the branch.

#### Review and fixes (2026-11-16)

**Provenance and verdict.** An independent strongest-tier reviewer at high effort, on the coordinator's brief, read the staged P4. It found the status row correct for every credential shape,
the foreign session refused on every surface, four of five wording surfaces held, ten race rounds clean, and ten of its twelve own breaks caught; two blockers (B-1, B-2), four should-fixes
(S-1 to S-4) and notes (N-2, N-4, N-5), and put three questions to the owner. This part was written by a mainstream model at medium effort on the coordinator's follow-up brief, base `cecc892`,
the same 13 files staged plus what is listed here. Staged, not committed. *(2026-11-17: the fixes of the second review are below.)*

**Two corrections the coordinator made, recorded plainly.**

1. *A new exported name was built before it was put to the coordinator.* The brief said to stop on a new exported name the plan does not call for; `provider.SessionKindOf` was built first and reported
   afterwards, on the reading that the plan's step 2 required the value. That reading was the agent's, not the plan's; the rule was to stop. The owner then ruled on it (ruling 1, below).
2. *The "already true" list held a false claim.* It said `drive` (`llm/provider/signinrun_test.go:76`) "is `TestSignInChatGPTPlanPasted`'s sign-in". It is not: the archived tests
   (`git show 1406bd6:llm/provider/signin_test.go`, `TestSignInChatGPTPlan` near `:287` and the pasted one near `:351`) ran `SignIn`, the wizard behind `gobble auth login openai`; `drive` runs `StartSignIn`,
   the engine the ACP path uses. It was found by the reviewer, who broke client reuse on the terminal (`llm/provider/signin.go`, `TokenStore: tokenStore(store)` replaced by a store without the client
   store) and saw both packages pass. The text above is corrected in place, with a dated note at it.

**The owner's three rulings of 2026-11-16, each the owner's ruling, relayed by the coordinator.**

1. "Keep it, and make it the single place the kind is read". Done: `git grep -n 'SessionKind()' -- '*.go' ':!*_test.go'` gives one line, `llm/provider/signinrun.go:79`, in `SessionKindOf` (`:75`).
   The other reads now call it: `storedMethod` (`llm/provider/consent.go:97`), the adapter's `planConsentOf` (`llm/provider/provider.go:444`, through `loginOf`) and `StartSignIn`'s `Previous`
   (`llm/provider/signinrun.go:111`). `internal/auth` holds no inference of its own: `AcceptsLogin` (`internal/auth/resolve.go:38`) builds a `provider.Login` and calls `provider.Accepts`,
   so no layering rule (archtest rule 4) is bent and no stop arose. Plants of 0051 P4, P6 and P7 that touch these files re-run: below. Test: `TestSessionKindOf` (`llm/provider/sessionkind_test.go:13`).
2. "Amend the acceptance to a whole-word grep". Done: a dated note on P4's acceptance line. `grep -rnw -e chatgptPlanBackground -e askPlanBackground -e planNotice --include=*.go .`: exit 1, nothing found.
3. "Add the check now, in P4". Done: `Options.login` refuses a stored session `!Accepts(id, l)` (`llm/provider/provider.go:107`) with the resolver's error and words, `provider openai has an OAuth login in auth.json,
   which gobble does not use (0005-PLAN F4-3b)`, a plain error, the resolver's own (`auth check` maps the resolver's error to `invalid` / `invalid_state` directly and never consults `llm.Kind`); `New` returns it as it is, naming the provider once *(2026-11-17: the first text had `New` prefix it with `provider openai: `, which named the provider twice)*. Tests:
   `TestNewRefusesAForeignLogin` (`llm/provider/foreignlogin_test.go:17`: another application's ChatGPT login, a login stored with no kind, a plan session accepted) and
   `TestStaleResolverCannotBuildAForeignSession` (`internal/cli/staleresolver_test.go:16`: a resolver built over a plan session, `auth.json`'s entry then replaced by a foreign one, the run's provider and a
   session's provider refused). Plants: the check removed, which fails both. The whole suite passes with it (`go test ./internal/cli ./acpserver ./llm/... ./internal/auth ./internal/archtest`, then the gates
   below): no test needed changing. *(2026-11-17, corrected in the second review: the first text said here that a provider built first, over a session a run's `Sessions` holds, and the entry replaced afterwards, "is not refused, since the run keeps the session it loaded; that is the cache's design", and treated the ruling as met. That was false and the ruling was only partly met: at the held session's next refresh the SDK reloads `auth.json` under the refresh lock and adopts a stored session whose refresh token differs, its kind included (go-llmprovider-sdk `v1.31.0` `llmprovider/auth/oauth_session.go:971`, `loadRotated` `:1007-1022`), so the held session becomes the foreign one and a build over it is not refused. The check refuses at the load; it does not refuse a session a run already holds.)* Dated amendments: P4's steps 1a and 1b, and a note at
   0051-MADR D2.

**Items.**

* **B-1, no test completed the plan sign-in in the terminal.** Built: `TestSignInChatGPTPlanTerminal` (`internal/cli/plansignin_test.go:307`) and `TestPlanAfterLogoutReusesClientTerminal` (`:356`), through
  `terminalPlan` (`:253`): `gobble auth login openai` on a terminal (`loginWith`, `signInTo`, `hostSignIn`: production's), `provider.SignIn` the real wizard against the plan issuer, a pipe for stdin, the
  consent allowed by typing `1`, the callback pasted when the browser opens (`openURL` replaced by a function that writes it); the SDK's own loopback listener is opened by the wizard and the paste wins it.
  They assert the consent's wording shown before the browser, the authorize query (gobble's name, `ext_agent_host_id` equal to `<state>/host-id`), the saved session's kind, the consent `granted`, the host id,
  `clients.json`'s record, a second sign-in over the stored session that asks nothing and signs in to the issued client with `id_token_hint` and no `agent_name_hint`, and after a logout (revoked, deleted) a
  new sign-in naming the stored client and the host with no `agent_name_hint` and the consent asked again. The ACP path's tests are renamed `TestSignInChatGPTPlanACP` (`:130`) and
  `TestPlanAfterLogoutReusesClientACP` (`:192`). Which of the plan's rows each holds: the plan's row `TestSignInChatGPTPlan` is held on both paths, by `...ACP` (the engine `StartSignIn`, what an ACP client's
  `authenticate` runs) and `...Terminal` (the wizard `SignIn`, what `gobble auth login`, `/login`, setup, the first run and `acp --login` run); `TestPlanAfterLogoutReusesClient` likewise; the archived
  `TestSignInChatGPTPlanPasted` is `...Terminal`'s paste, and the ACP path completes by paste only. Plants: the reviewer's (`wizard-client-store-dropped`) and the plan's (`terminal-host-id-not-passed`), below.
* **B-2, `docs/architecture.md`'s `internal/cli` row.** It now reads "a stored session of a kind no offered method signs in to (another application's ChatGPT login, not Grok's or the plan's) is `usable: false`,
  `oauth_unused`". The guide's `reason` row and the README say the same or less: `docs/guides/credentials.md`'s `reason` row names "another application's ChatGPT login", and its line 290 says "an OpenAI entry of another kind,
  another application's ChatGPT login"; neither says a Grok session is unusable.
* **S-1, the advertised method's wording.** `TestAdvertisedPlanConsentIsGobbles` (`internal/cli/advertised_test.go:17`): `authOptions().Methods("openai")`'s `chatgpt_plan` consent equals `provider.Offered("openai")`'s
  and `provider.ConsentText`'s, text and links. Plant `advertised-method-text-altered` (the wiring replaced by a function that alters the text).
* **S-2, `SessionKindOf`'s comment.** `llm/provider/signinrun.go:66-74`: a stored `Kind` comes back as it is, even one the SDK does not know; "" only for no login or one the SDK infers no kind for.
  `TestSessionKindOf` covers no login, a stored plan kind, an unknown stored kind, openai's and grok's login with none, and a provider with none. Plants `kind-unknown-dropped`, `kind-not-inferred-unit`.
* **S-3, record details.** The six-tests sentence, the `statusplan_test.go:266` citation, every `path:line` below re-checked at the end of this pass, and the dated notes at four rows of 0051-PLAN
  (`:1879`, `:1880`, `:2885`, `:2891`) saying each is closed by this phase and where. The record's "now closed" is therefore true.
* **S-4, the Verification lines.** Dated notes at the `./llm/provider/` and `./internal/cli/` lines: the old pattern's `TestSignInChatGPTPlan|TestPlanAfterLogout|TestLogoutPlan` matched no test in `./llm/provider/`
  (`go test -list` of it printed none); the patterns that run this phase's tests are given there, each checked with `go test -list` to match every name it claims (24 in `./llm/provider/`, 13 in `./internal/cli/`).
* **N-2, the `sessionKind` row** in `docs/guides/credentials.md` says it is absent when the SDK names no kind (the golden's `claude` row).
* **N-4, the foreign stand-in.** `statusPlanForeignNoKind` (`internal/cli/statusplan_test.go:191`): an `auth.json` written as text with no `kind`, the fields of Pi's entry shape and gobble's own recorded issuer
  and client (`internal/auth/store.go`'s `Entry`; no real file was read): `sessionKind` `chatgpt` by the SDK's inference, `usable: false`, `oauth_unused`, no consent. Plant `usable-from-has-without-accepts` fails it too.
* **N-5, the leak assertion.** `noTokens` and `textRow` (`internal/cli/statusplan_test.go:70`, `:80`): the subtests that print a session's row check the JSON and the text form for the fixtures' token strings (`secret`, `id-token`; the plan fixture stores an
  ID token); *(2026-11-17, corrected: the first text said every subtest checked both; the unreadable-store subtest, whose file holds no token, checks only the JSON.)* Plants `token-leaked-in-source` (a token appended to the row's `source`) and `token-leaked-in-text` (a refresh token appended to the text row).

**Plants of this phase, every one, on the final tree** (scratch copies, `GOPROXY=off`, no proxy variable; each exit 1 with a `--- FAIL`, none a build failure; the unbroken runs exit 0):

| Plant | Test | Break | Exit | Failing line |
| :--- | :--- | :--- | :--- | :--- |
| `host-id-not-passed` | `TestSignInChatGPTPlanACP`, `TestPlanAfterLogoutReusesClientACP` | the host id not passed (ACP path, `agentauth.go`) | 1 | `plansignin_test.go:153: the first authorize query map[agent_name_hint:[gobble] client_id:[dynamic_agent_client] code_challenge:[9907gqFV3ZUxyQ2o6lAtzi` |
| `clients-cleared-by-logout` | `TestPlanAfterLogoutReusesClientACP` | `clients.json` cleared by the logout | 1 | `plansignin_test.go:221: clients.json after the logout holds {ClientID: HostID:}, <nil>; want the issued client kept` |
| `usable-from-has-without-accepts` | `TestAuthStatusJSONPlan` | `Usable` from `Has` without `Accepts` | 1 | `statusplan_test.go:222: the foreign row {Provider:openai Usable:true Reason:oauth_unused AuthType:oauth SessionKind:chatgpt Consents:map[] ConsentTerm` |
| `terms-left-out-while-unanswered` | `TestAuthStatusJSONPlan` | `consentTerms` left out while unanswered | 1 | `statusplan_test.go:132: consentTerms [], want the one consent of the plan's method` |
| `terms-text-sdk-default` | `TestAuthStatusJSONPlan` | `text` the SDK's default | 1 | `statusplan_test.go:136: consentTerms[0] {ID:chatgpt_plan.background_use Required:true Text:You're using your ChatGPT plan: this app's requests count t` |
| `kind-not-inferred` | `TestAuthStatusJSON` (golden) | `SessionKindOf` returning the stored `Kind` only | 1 | `auth_test.go:152: auth-status.json differs from testdata/auth-status.json.golden:` |
| `status-accepts-build-refuses` | `TestForeignOpenAISessionRefused` | status accepts while the build refuses | 1 | `statusplan_test.go:298: status: usable true, reason ""; want not usable, oauth_unused` |
| `every-surface-accepts-foreign` | `TestForeignOpenAISessionRefused` | `Accepts` always true | 1 | `statusplan_test.go:298: status: usable true, reason ""; want not usable, oauth_unused` |
| `logout-revokes-foreign` | `TestForeignOpenAISessionRefused` | the logout revoking a foreign session | 1 | `statusplan_test.go:328: the logout of a foreign session sent ["POST https://auth.openai.com/oauth/revoke"], want nothing: it was not gobble's to revok` |
| `start-sign-in-without-consent-text` | `TestConsentWordingOverACP` | `StartSignIn` without `ConsentText` | 1 | `authconsent_test.go:73: _meta.gobble.step.text "You're using your ChatGPT plan: this app's requests count toward its usage, which you can manage on th` |
| `terminal-host-id-not-passed` | `TestSignInChatGPTPlanTerminal`, `TestPlanAfterLogoutReusesClientTerminal` | the host id not passed (terminal, `login.go`: the plan's) | 1 | `plansignin_test.go:325: the first authorize query map[agent_name_hint:[gobble] client_id:[dynamic_agent_client] code_challenge:[et8aAm8KOOSBkIb2_Pt_Ke` |
| `wizard-client-store-dropped` | `TestSignInChatGPTPlanTerminal`, `TestPlanAfterLogoutReusesClientTerminal` | the client store dropped on the wizard path (`signin.go`: the reviewer's) | 1 | `plansignin_test.go:337: clients.json holds {ClientID: HostID:}, <nil>; want the issued client and the host id` |
| `advertised-method-text-altered` | `TestAdvertisedPlanConsentIsGobbles` | the advertised `Methods` replaced by one that alters the text | 1 | `advertised_test.go:35: the advertised consent {ID:chatgpt_plan.background_use Required:true Text:altered Links:[{Rel:usage URL:https://chatgpt.com/set` |
| `new-accepts-foreign-login` | `TestNewRefusesAForeignLogin` | the `Options.login` check removed | 1 | `foreignlogin_test.go:39: the build gave <nil>, want the resolver's refusal of a foreign login` |
| `stale-resolver-builds-foreign` | `TestStaleResolverCannotBuildAForeignSession` | the `Options.login` check removed | 1 | `staleresolver_test.go:30: the run's provider over a foreign session gave <nil>, want the refusal "has an OAuth login in auth.json, which gobble does n` |
| `kind-unknown-dropped` | `TestSessionKindOf` | an unknown stored kind dropped | 1 | `sessionkind_test.go:32: SessionKindOf(openai) = "", want "a_future_kind"` |
| `kind-not-inferred-unit` | `TestSessionKindOf` | no inference (the stored `Kind` only) | 1 | `sessionkind_test.go:32: SessionKindOf(openai) = "", want "chatgpt"` |
| `token-leaked-in-source` | `TestAuthStatusJSONPlan` | a token appended to the row's `source` | 1 | `statusplan_test.go:120: auth status --json printed a credential:` |
| `token-leaked-in-text` | `TestAuthStatusJSONPlan` | a refresh token appended to the text row | 1 | `statusplan_test.go:147: auth status printed a credential:` |
| `kind-read-twice` | `TestStartSignInPassesPrevious` | `StartSignIn`'s `Previous` taken for the wrong kind | 1 | `signinrun_test.go:184: the second sign-in's steps [{1 1 confirm   0001-01-01 00:00:00 +0000 UTC  false chatgpt_plan.background_use gobble signs in to ` |

Twenty plants, twenty exit 1 with a `--- FAIL`, none a build failure. Unbroken runs on a scratch copy, nine of them (`host-id-not-passed`, `terminal-host-id-not-passed`, `usable-from-has-without-accepts`, `status-accepts-build-refuses`, `start-sign-in-without-consent-text`, `advertised-method-text-altered`, `new-accepts-foreign-login`, `kind-unknown-dropped`, `token-leaked-in-source`, covering every new or renamed test): exit 0 each. The driver is `~/.cache/claude-session-5a7234fa/scratch/gobble-0052-p4/plants.py`.

**Earlier plants re-run** (this pass changed `llm/provider/consent.go`, `provider.go` and `signinrun.go`; *(2026-11-17, corrected: the first text also named `internal/cli/auth.go`, which this pass did not change)* each plant of 0051-PLAN P4's, P6's, P6-fix's and P7's drivers whose edit touches `consent.go`, `provider.go`,
`signinrun.go`, `policy.go`, `login.go`, `resolve.go` or `internal/cli/auth.go`): 0051 P4: 24 of 24, 0051 P6 fix: 24 of 24, 0051 P7: 4 of 4, each exit 1 with a `--- FAIL`, none a build failure (drivers `p4plants.py`, `p6plants.py`, `p7plants.py`, the earlier phases' with their scratch path changed and, for `previous-never` and `start-deletes-stored` of the P6 fix, their anchors moved to the lines as step 1a leaves them: the same breaks). The five plants of the original P6 driver that are not in the P6 fix driver (`listing-refusal-ignored`, `save-before-listing-provider`, `save-before-listing-acp`, `unavailable-listing-fails-provider`, `unavailable-listing-fails-acp`) anchor on `signinrun.go` text the P6 fix removed (`if l.Err != nil && errors.Is(l.Err, llm.ErrAuth) {`, 0 matches): they are superseded by the fix's `b2-*` plants and were not run; the anchor check says so.

**Gates (C6),** after the last change to a Go file, each with start, end and exit status taken before any filter. A first start of this run (23:07:59) used the unstaged files only for its pre-add list and was stopped by the agent
before its `make race` ended; its rows are not counted. The list below is `git diff HEAD --name-only` plus the untracked files, `.go` only.

| Gate | Start | End | Exit |
| :--- | :--- | :--- | :--- |
| pre-add-check | 23:08:38 | 23:08:52 | 0 |
| race | 23:08:52 | 23:10:15 | 0 |
| check-test-isolation | 23:10:15 | 23:11:39 | 0 |
| check-acpext-deps | 23:11:39 | 23:11:40 | 0 |
| linux-race | 23:11:40 | 23:15:56 | 0 |
| windows-race | 23:15:56 | 23:20:37 | 0 |
| preflight | 23:20:37 | 23:28:50 | 0 |

The Linux and Windows rows are `run-on-host --race --need linux` and `--need windows` of `go test -race -count=1 ./...` and `go -C acpext test -race -count=1 ./...` (the laptop's native shell, no WSL distro); preflight is one cold lint run. The final `make check-records markdownlint` is below the table of files.

**Files and times.** Staged Go and golden files' modification times against the first gate's start (23:08:38): the latest is 22:48:15; none is later.

| File | Modified |
| :--- | :--- |
| `llm/provider/sessionkind_test.go` | 22:48:15 |
| `internal/cli/statusplan_test.go` | 22:48:15 |
| `internal/cli/advertised_test.go` | 22:48:15 |
| `internal/cli/staleresolver_test.go` | 22:47:30 |
| `llm/provider/foreignlogin_test.go` | 22:46:39 |
| `internal/cli/plansignin_test.go` | 22:46:17 |
| `llm/provider/signinrun.go` | 22:45:10 |
| `llm/provider/provider.go` | 22:45:10 |
| `llm/provider/consent.go` | 22:45:10 |
| `acpserver/authconsent_test.go` | 21:26:36 |
| `internal/cli/auth.go` | 20:48:02 |
| `internal/cli/testdata/auth-status-plan.json.golden` | 20:42:57 |
| `internal/cli/loginissuer_test.go` | 20:42:06 |
| `internal/cli/testdata/auth-status.json.golden` | 20:40:49 |

The docs and records were last written after the gates (the README, `docs/architecture.md` and `docs/guides/credentials.md` before them); `make check-records markdownlint` ran after the last of them.

**Docs rows.** `docs/architecture.md` (B-2, the `New` check, the single place for the kind) and `docs/guides/credentials.md` (N-2) fixed; 0051-PLAN's four rows annotated (S-3); 0051-MADR D2 annotated (ruling 3);
0052-PLAN's Verification and acceptance annotated (S-4, ruling 2). Searched: `README.md`, `AGENTS.md`, `docs/architecture.md`, `docs/guides/`, `docs/README.md`, `docs/decisions/0051-*`, `0052-*`.

**Not verified.** No live provider, real browser or phone. The terminal tests run the SDK's real wizard against an issuer in process; the wizard still binds its own loopback listener, which a paste beats.
The reviewer's break driver and outputs (`gobble-0052-p4-review/breaks.py`, `breaks.out`) were read, not run.

**Deviations or questions:** none open. The two corrections above are the agent's.

#### Second review and fixes (2026-11-17)

**Provenance and verdict.** A second independent strongest-tier reviewer at high effort, on the coordinator's brief, read the re-staged P4. Every first-round finding was fixed and each new test failed its own break; the pass had
introduced a data race (R2-B1), one ruling was only half met (R2-B2), and the new terminal tests bind real ports (R2-B3); three should-fixes (R2-S1, R2-S2 and the doubled provider name). The fixes were begun by one mainstream-tier
agent at medium effort, which ran out of working budget before the proof, the gates and the record, and finished by a second mainstream-tier agent at medium effort, each on the coordinator's brief; base `cecc892`, 21 files staged before
this pass. Staged, not committed.

**The correction the coordinator named.** The first review's record said of a provider built over a session a run's `Sessions` already holds, with `auth.json`'s entry replaced afterwards, that it "is not refused ... the cache's
design", and treated the owner's ruling "Add the check now, in P4" as met. That was false. The ruling was met at the load and not at use: at a held session's next refresh the SDK reloads the store under the refresh lock and adopts
a stored session whose refresh token differs, its kind included (go-llmprovider-sdk `v1.31.0`, `llmprovider/auth/oauth_session.go:971` `reloadOrRefresh` calling `loadRotated`, `:1007-1022`). The statement is corrected in place above,
at 0051-MADR D2's note and in `docs/architecture.md`; a ruling met in part is reported as met in part.

**The owner's rulings of 2026-11-17, each "the owner's ruling, relayed by the coordinator".**

1. "Refuse at the point of use" (a small gobble token source wraps the held session; on each token request it checks the session's current kind under the session's lock and, if the policy no longer accepts it, returns the
   resolver's error and drops the session; not chosen: checking when the held session is handed out; accepting it and documenting it). The builder stopped, because the SDK asserts its concrete session type and a wrapper would hide a
   plan session from it. Then, put the stop, the owner ruled: "Ask the SDK for interfaces, then build as ruled" (a go-llmprovider-sdk change replaces the concrete-type checks with interfaces, in its next release; gobble then builds the
   wrapper exactly as ruled; until that release a held session can still adopt a foreign entry, and the docs say so). This was not the option the coordinator recommended; the coordinator recommended refusing in gobble's own store adapter now.
   Not chosen: refusing in gobble's own store (the adapter refuses to hand the SDK a foreign entry on a reload and to overwrite one on a save); refusing in the store now and asking the SDK too.
   *(2026-11-17, later; the owner's ruling, relayed by the coordinator, on the SDK's draft record for the held session, "The SDK refuses to adopt another sign-in (Recommended)": the SDK's own session would refuse to adopt a stored session that is not the same sign-in, at the places it adopts from its store, and the token request would fail with the existing auth-failure error, with no interfaces and no new exports. That record is a proposed draft, not accepted and not released. If it is accepted and released, gobble's later step becomes a pin bump and a test that a held session fails, sends no foreign token and stays `chatgpt_plan`, with no wrapper; until then the documented limit stands as written.)*
2. "Keep them until the SDK offers a no-listener option" (the two terminal plan sign-in tests keep binding the SDK wizard's callback port, 1455, or 1457 when 1455 is taken). This was not the coordinator's recommendation.

**R2-B1, the data race.** Reproduced first by the reviewer: the `planConsentOfRacesARefresh` reproduction, run with `-race` on the staged tree, reported `DATA RACE` (14 lines: `loginOf` read by `planConsentOf` against `Invalidate`'s write).
`planConsentOf` called `SessionKindOf(id, loginOf(s))`, and `loginOf` copies a live session's fields with no lock. Fixed: `kindOfSession(s)` (`llm/provider/signinrun.go:87`) is the one call into the SDK's `SessionKind()`, which takes the
session's lock (`oauth_session.go:403`); `SessionKindOf` (`:75`) calls it through the login's own session, and `planConsentOf` (`llm/provider/provider.go:456`) calls it directly on the live session.
`git grep -n 'SessionKind()' -- '*.go' ':!*_test.go'`: one line, `llm/provider/signinrun.go:87`. The other caller of `loginOf` is `sdkStore.Save` (`llm/provider/login.go:157`), which the SDK calls with the refresh's own `next` session
(`oauth_session.go:352` and `:995`, a value the refresh built, not shared). Test: `TestPlanConsentOfReadsTheKindUnderTheSessionLock` (`llm/provider/planconsentrace_test.go:14`), run under `-race` by `make race` and the ten-round run below.
Plant `plan-consent-of-unlocked-read` (the unlocked read restored): `go test -race` exits 1 with 14 `DATA RACE` lines; unbroken: exit 0.

**R2-B2, the held session: not built; recorded.** No code is written for it in P4, by the owner's rulings above. Today a stored session of a kind no offered method signs in to is refused when gobble loads it; a session a run already holds can,
if another process writes another application's ChatGPT login into gobble's own `auth.json` while the run goes on, adopt it at its next refresh and send its token. The SDK asserts its own concrete session type in three places, in two
files, at `v1.31.0` (read with `git show v1.31.0:<path>`): `llmprovider/internal/chatgpt/chatgpt.go:50` and `:58`, and `llmprovider/providers/openai/openai.go:380`; the reload that adopts a foreign entry (`oauth_session.go`
`reloadOrRefresh`, `loadRotated`) asks the token store, not the token source. *(The first draft of this part also named `llmprovider/catalog/discovery.go:481`; at `v1.31.0` that line is an interface, `identityUser`, in `nameSession`, not a
concrete-type check, and is not one of the places.)* The point-of-use refusal is a named step of the gobble phase that bumps the SDK pin to the release with the interfaces. The request to the SDK is on the coordinator's ledger; it is not
filed in any record of that repository. The reviewer's reproduction, `TestReviewCachedSessionAdoptsAForeignEntry`, fails on this tree by design and is therefore not in it; its text, rewritten to use the in-process issuer, is in "third review" below, and the later phase adopts it. Recorded at 0052-PLAN P4's step 1d, 0051-MADR D2's note, `docs/architecture.md` (the `llm/provider` row) and `docs/guides/credentials.md`
(after the unused-sign-in sentence).

*(2026-11-17, later; the owner's ruling, relayed by the coordinator, on the SDK's draft record for the held session, "The SDK refuses to adopt another sign-in (Recommended)": the SDK's own session would refuse to adopt a stored session that is not the same sign-in, at the places it adopts from its store, and the token request would fail with the existing auth-failure error, with no interfaces and no new exports. That record is a proposed draft, not accepted and not released. If it is accepted and released, gobble's later step becomes a pin bump and a test that a held session fails, sends no foreign token and stays `chatgpt_plan`, with no wrapper; until then the documented limit stands as written.)*

**R2-B3, the terminal tests keep the SDK's ports.** `TestSignInChatGPTPlanTerminal` (`internal/cli/plansignin_test.go:313`) and `TestPlanAfterLogoutReusesClientTerminal` (`:359`) stay, by the second ruling above. When both ports are held the
failure surfaces from the SDK as `openai sign-in failed (other)`, whose `errors.Unwrap` chain holds `oauth: registered callback ports unavailable; use device-code login: listen tcp4 127.0.0.1:1455: bind: address already in use` and three more
`listen` lines; `signInFailed` (`:394`) tells a collision from any other failure by that text, after the failure and with no probe, and fails the test (never skips) with: "the first plan sign-in on the terminal failed because both callback
ports, 1455 and 1457, are in use on this machine: the SDK's wizard binds them for a terminal sign-in ... Nothing is wrong with gobble; free the ports and run again." Run here on a scratch copy of the final tree (`ports2.sh`): ports free,
both tests pass (listeners before: 0); with 1455 and 1457 held by `holdports.py` on both address families, both fail at once with that message (exit 1, FAIL 2), and the ports are free again afterwards. The reviewer measured: alone, 20 of 20 pass, and a listener census over those 20 runs saw only 1455 bound; with 1455 held, the wizard falls back to 1457 and passes. That a real Codex login starting while a test holds 1455 cannot bind it is reasoned, not measured. The wizard binds 1455 on both loopback families and 1457 only when 1455 is taken (`oauth_loopback.go`, `listenOpenAILoopback`, `v1.31.0`). The tests switch to a no-listener option in the phase that bumps the SDK pin to the release that has it; the request is on the
coordinator's ledger and filed nowhere. Written at P4's step 1c, at 0051-PLAN's note on the P6 ruling "Assert what gobble passes; use no real ports", and in `docs/guides/developing.md:86`.

**R2-S1, the consent comes first.** `terminalPlan` (`plansignin_test.go:255`) records the output written when the browser first opens, and the first-sign-in assertions require gobble's wording in that prefix. Plant `consent-after-browser`
(the browser opened once before the wizard asks): fails at `plansignin_test.go:322`.

**R2-S2, three false claims, corrected in place** (each with a dated note above): "this pass changed ... `internal/cli/auth.go`" (it did not); that `llm.Kind` `other` is what `auth check` derives `invalid` / `invalid_state` from
(`check` maps the resolver's error directly); and "every subtest checks the JSON and the text form" (the unreadable-store subtest, whose file holds no token, checks the JSON only).

**The error named the provider twice.** `New` returns the refusal as it is (`llm/provider/provider.go:217`; `foreignLoginError` `:125`, built at `:112`): `provider openai has an OAuth login in auth.json, which gobble does not use
(0005-PLAN F4-3b)`, once. `TestNewRefusesAForeignLogin` (`foreignlogin_test.go`) and `TestStaleResolverCannotBuildAForeignSession` (`staleresolver_test.go`) require that exact text. Plant `new-prefix-doubled`: fails at `foreignlogin_test.go:39`.

**Smaller items.** The comments of `statusPlanForeignNoKind` and `statusPlanForeign` (`internal/cli/statusplan_test.go:190`, `:212`) are each their own; the `sessionKind` row of `docs/guides/credentials.md` says "such as an OAuth entry for Claude".

**Plants, every one of this phase, on the final tree** (scratch copies of the staged tree; `HOME`, `XDG_*`, `GOBBLE_HOME` in the scratch, `GOPROXY=off`, no proxy variable; the race plant under `-race`): 23 plants, 23 exit 1 with a `--- FAIL` or, for
`plan-consent-of-unlocked-read`, `DATA RACE`; none a build failure. The 23: `host-id-not-passed`, `clients-cleared-by-logout`, `usable-from-has-without-accepts`, `terms-left-out-while-unanswered`, `terms-text-sdk-default`,
`status-accepts-build-refuses`, `every-surface-accepts-foreign`, `logout-revokes-foreign`, `kind-not-inferred`, `terminal-host-id-not-passed`, `wizard-client-store-dropped`, `advertised-method-text-altered`, `new-accepts-foreign-login`,
`stale-resolver-builds-foreign`, `kind-unknown-dropped`, `kind-not-inferred-unit`, `token-leaked-in-source`, `token-leaked-in-text`, `kind-read-twice`, `consent-after-browser`, `plan-consent-of-unlocked-read`, `new-prefix-doubled`,
`start-sign-in-without-consent-text`. The unbroken runs of each plant's tests: 23 of 23 exit 0. The plant driver broke on the way: three plants anchored on `return string(l.session(id, nil, nil, nil).SessionKind())`, which this pass changed
to `return kindOfSession(l.session(id, nil, nil, nil))`, so the driver's assertion on the anchor stopped it at the ninth plant; the three were re-anchored on the new line (the same breaks), and the driver now reports an anchor that does not match
and carries on. No test was changed for it. Driver: `~/.cache/claude-session-5a7234fa/scratch/gobble-0052-p4/plants.py`.

**Earlier plants re-run** (this pass changed `llm/provider/provider.go` and `signinrun.go`, and tests): 0051 P4: 24 of 24, 0051 P6 fix: 24 of 24, each exit 1 with a `--- FAIL`, none a build failure; they ran after
the last change to any Go file but the two lint fixes under "Gates" (the newest staged Go file was then written at 06:47:38, the two runs ended at 06:49:18 and 06:58:46; the lint fixes touched `New` in `provider.go` and the new race test,
no 0051 P6-fix or P7 plant breaks either, and the two 0051 P4 plants that break `provider.go` elsewhere were re-run after them). 0051 P7: 10 of 10 exit 1, none a build failure, and its five unbroken runs exit 0. Drivers `p4plants.py`,
`p6plants.py`, `p7plants.py` in the same folder.

**Gates (C6),** after the last change to a Go file, each with start, end and exit status taken before any filter, and the number of listeners on ports 1455 to 1457 on this Mac before it. `linux-race` and `windows-race` are
`run-on-host --race --need linux` and `--need windows` of `go test -race -count=1 ./...` and `go -C acpext test -race -count=1 ./...` (the laptop's native shell, no WSL distro), so the listeners' count before them is this Mac's, and what
those hosts' own runs showed is under "Not verified"; `race-10` is `go test -race -count=10 -timeout 60m ./internal/cli ./acpserver ./llm/provider` through `run-heavy` on this Mac; `preflight` is one cold lint run. The list for pre-add-check
is `git diff HEAD --name-only` plus untracked, `.go` only. The first two full runs of the gates were not counted: each ended with `make preflight` exit 2 on the cold lint, and the Go files changed after them. The
first: `llm/provider/provider.go:217` `errcheck` (`_, foreign := errors.AsType[*foreignLoginError](err)`) and `llm/provider/planconsentrace_test.go:30` `modernize` (`waitgroupgo`: a goroutine with its own `WaitGroup`). The second, after
the first fix (`errors.As` with a variable): `provider.go:218` `modernize` `errorsastype`. Fixed properly, no `//nolint`: `if foreign, ok := errors.AsType[*foreignLoginError](err); ok { return nil, foreign }` and `wg.Go(func() {...})`;
`make lint` then reported 0 issues, the plants touching those lines re-ran (`new-accepts-foreign-login`, `stale-resolver-builds-foreign`, `new-prefix-doubled`, `plan-consent-of-unlocked-read`, and 0051 P4's `mark-dropped` and
`mark-no-store`, which break `provider.go` elsewhere), all exit 1 with a `--- FAIL`, the race plant with 14 `DATA RACE` lines, and their unbroken runs exit 0; `new-prefix-doubled`'s anchor and break were adapted to the new code (it
wraps `foreign` in the doubled prefix) and, before that, the anchor miss produced a build failure which the re-run caught. The table below is the third run, which began after the last Go change. Its first `preflight` row ended exit 2 on
`markdownlint` alone (`docs/guides/credentials.md:309`, line length, a sentence of this pass's own, since wrapped); `preflight-2` is the whole of `make preflight` again, with `check-records` and `markdownlint` in it, after the wrap.

| Gate | Start | End | Exit | Listeners on 1455-1457 before |
| :--- | :--- | :--- | :--- | :--- |
| pre-add-check | 08:30:28 | 08:30:43 | 0 | 0 |
| race | 08:30:43 | 08:32:11 | 0 | 0 |
| check-test-isolation | 08:32:11 | 08:33:36 | 0 | 0 |
| check-acpext-deps | 08:33:36 | 08:33:36 | 0 | 0 |
| linux-race | 08:33:36 | 08:41:34 | 0 | 0 |
| windows-race | 08:41:34 | 08:44:02 | 0 | 0 |
| race-10 | 08:44:02 | 08:57:23 | 0 | 0 |
| preflight | 08:57:23 | 09:07:21 | 2 | 0 |
| preflight-2 | 09:07:52 | 09:17:49 | 0 | 0 |

**Files and times.** Staged Go and golden files' modification times against the first gate's start (08:30:28): the newest is 08:28:40.

| File | Modified |
| :--- | :--- |
| `llm/provider/provider.go` | 08:28:40 |
| `llm/provider/planconsentrace_test.go` | 07:52:39 |
| `internal/cli/plansignin_test.go` | 06:47:38 |
| `internal/cli/staleresolver_test.go` | 06:46:09 |
| `llm/provider/foreignlogin_test.go` | 06:46:03 |
| `internal/cli/statusplan_test.go` | 06:45:51 |
| `llm/provider/signinrun.go` | 06:42:52 |
| `llm/provider/sessionkind_test.go` | 22:48:15 |
| `internal/cli/advertised_test.go` | 22:48:15 |
| `llm/provider/consent.go` | 22:45:10 |
| `acpserver/authconsent_test.go` | 21:26:36 |
| `internal/cli/auth.go` | 20:48:02 |
| `internal/cli/testdata/auth-status-plan.json.golden` | 20:42:57 |
| `internal/cli/loginissuer_test.go` | 20:42:06 |
| `internal/cli/testdata/auth-status.json.golden` | 20:40:49 |

**Docs rows.** `docs/guides/developing.md:86` (the two tests' ports), `docs/architecture.md` (the `llm/provider` row: `New`'s check at load, and the open item), `docs/guides/credentials.md` (the `sessionKind` row, and the sentence after the unused-sign-in
paragraph) fixed; 0051-PLAN's note on the P6 ruling (the later note, now its own paragraph), 0051-MADR D2's notes and 0052-PLAN's step 1c and 1d written; the false statements above corrected in place. Searched: `README.md`, `AGENTS.md`,
`docs/architecture.md`, `docs/guides/`, `docs/README.md`, `docs/decisions/0051-*`, `0052-*`.

**Not verified.** The port behaviour was measured on this Mac only; the Linux and Windows host runs are in the table (both exit 0, so the two terminal tests passed there; their output, run without `-v`, did not say whether the ports were free). The Linux run waited in the host queue behind another
agent's measurement. No live provider, real browser or phone. The held-session
refusal is not built (the owner's rulings above), so a held session that adopts a foreign entry is not refused today.

**Stops and questions:** none open. Deviations: the corrections the coordinator named and listed above; the first draft's `discovery.go` cite. The held-session item waits for the SDK's interfaces by the owner's ruling.

#### Third review (2026-11-17)

**Verdict.** A third independent strongest-tier reviewer read the third staging and found it committable, with no blocker: the race fixed and held by a race test, the owner's rulings built, the held-session limit stated truthfully, every
one of its breaks failing, and its own ten race rounds clean. It left four small items and one note, done by the same mainstream-tier agent at medium effort, on the coordinator's brief.

1. **A measurement the reviewer did not make.** The text said the reviewer measured that "a real Codex login starting while a test holds 1455 cannot bind it"; the reviewer reasoned it. Corrected in place, as "reasoned, not measured", in
   0051-PLAN's note on the P6 ruling (the "2026-11-17, later" paragraph) and in R2-B3 above; `docs/guides/developing.md` and the other places do not repeat it.
2. **Which ports the wizard binds.** The text said the wizard binds ports 1455 and 1457. It binds 1455 on both loopback address families, and 1457 only when 1455 is taken (go-llmprovider-sdk `v1.31.0`, `llmprovider/auth/oauth_loopback.go`,
   `listenOpenAILoopback` and `listenLoopbackBothFamilies`; the reviewer's listener census over 20 runs saw only 1455). Corrected in `docs/guides/developing.md:86`, 0051-PLAN's note, 0052-PLAN's step 1c, R2-B3 and the second ruling's line.
   The failure message ("both callback ports, 1455 and 1457, are in use on this machine", `internal/cli/plansignin_test.go:401`) is true as it stands: it fires only when both are taken.
3. **The held-session reproduction is in the record, and calls no real service.** The reviewer's `cachedadopt_test.go` lived only in a cache folder, and pointed the plan session's token URL at OpenAI's real issuer: one of the reviewer's scratch
   probes, with a guard in place, made a single request to a real token endpoint, carrying only fake values. That is why the test below is rewritten to use the in-process issuer (`newTestIssuer`, `llm/provider/login_test.go:98`), every token URL
   being the issuer's own. It stays out of the tree because it fails today by design. It is the test the later phase, the one that bumps the SDK pin, makes pass, whatever mechanism that phase finds (the SDK-side refusal of the owner's later ruling below, or gobble's own point-of-use refusal). Before pasting it, it was run on a scratch copy of the
   final tree (`heldverify.py`) with `http.DefaultTransport` replaced by a transport that fails any request to a host other than the in-process issuer and counts it: requests outside the issuer 0 (and 0 to the issuer, since the session's token was current);
   the test fails, as intended, at its three assertions (the held plan session sent the foreign login's token; its kind became `chatgpt`; a provider was built over it).

```go
package provider

import (
	"context"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
)

// A run's Sessions holds the plan session it loaded; another process then
// replaces auth.json's entry with another application's ChatGPT login; at the
// held session's next refresh the SDK reloads the store under the refresh lock
// and adopts what it finds. The held session must not become the foreign one:
// when the point-of-use refusal is built, a provider built afterwards from the
// same Sessions is refused, and the held session's kind stays the plan's.
func TestHeldSessionRefusesAForeignEntry(t *testing.T) {
	ti := newTestIssuer(t) // in-process: every token URL below is its own
	tokenURL := ti.URL + "/oauth2/token"
	plan := &Login{Access: "plan-a", Refresh: "plan-r", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
		ClientID: "oaiapp_gobbletest", TokenURL: tokenURL, Resource: "https://api.openai.com/v1",
		Kind: string(auth.SessionChatGPTPlan)}
	store := &memLogins{logins: map[string]*Login{"openai": plan}}
	sessions := &Sessions{}
	if _, err := New("openai", Options{Logins: store, Sessions: sessions}); err != nil {
		t.Fatalf("the first build over the plan session: %v", err)
	}
	// Another process replaces the entry: another application's ChatGPT login, its access current.
	foreign := &Login{Access: "codex-a", Refresh: "codex-r", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
		ClientID: "app_codex", TokenURL: tokenURL, Kind: string(auth.SessionChatGPT)}
	if err := store.Save(context.Background(), "openai", foreign); err != nil {
		t.Fatal(err)
	}
	held, err := (Options{Logins: store, Sessions: sessions}).login(context.Background(), "openai")
	if err != nil || held == nil {
		t.Fatalf("the held session: %v, %v", held, err)
	}
	// A 401 on the plan token makes the provider invalidate it; the next Token refreshes.
	held.Invalidate()
	tok, err := held.Token(context.Background())
	if err == nil && tok.Value == "codex-a" {
		t.Errorf("the held plan session sent the foreign login's token")
	}
	if held.SessionKind() != auth.SessionChatGPTPlan {
		t.Errorf("the held session is now of kind %q, want %q", held.SessionKind(), auth.SessionChatGPTPlan)
	}
	if _, err := New("openai", Options{Logins: store, Sessions: sessions}); err == nil && held.SessionKind() == auth.SessionChatGPT {
		t.Errorf("a provider was built over the held session, now another application's ChatGPT login")
	}
}
```

4. **A gap in a test (the reviewer's N-2).** Nothing held that an error of the store that is not the foreign refusal is still wrapped by `New` with its `provider <id>: ` prefix: the reviewer's break (every error returned unwrapped) passed
   `llm/provider` and the `internal/cli` foreign tests. `TestNewWrapsAStoreError` (`llm/provider/foreignlogin_test.go:60`), with a login store whose `Load` fails with a fixed error, requires the error to start with `provider openai: ` and to match the
   store's error with `errors.Is`. Plant `new-error-unwrapped` (`return nil, fmt.Errorf("provider %s: %w", id, err)` becomes `return nil, err`): exit 1, failing at `foreignlogin_test.go:67`: the build's error is "auth.json is unreadable",
   want the prefix "provider openai: "; unbroken: exit 0.
5. **N-3, one plain note.** 0051-MADR D2's note of this phase read as an annotation inside an annotation. It was never committed, so it is rewritten as one dated note (2026-11-17) that says what is true today: refused at load; a held session
   can adopt a foreign entry at its next refresh.

**Gates, after the last change to a Go file** (item 4's test file; each with start, end and exit status taken before any filter, and the listeners on ports 1455 to 1457 on this Mac before it). The ten-round race run is not repeated: the change
since the second staging's ten clean rounds is one added deterministic test case with no goroutine, and the third reviewer's own ten rounds on the third staging were clean.

| Gate | Start | End | Exit | Listeners on 1455-1457 before |
| :--- | :--- | :--- | :--- | :--- |
| pre-add-check | 09:46:58 | 09:47:11 | 0 | 0 |
| race | 09:47:11 | 09:48:36 | 0 | 0 |
| check-test-isolation | 09:48:36 | 09:49:59 | 0 | 0 |
| check-acpext-deps | 09:49:59 | 09:50:00 | 0 | 0 |
| linux-race | 09:50:00 | 09:54:34 | 0 | 0 |
| windows-race | 09:54:34 | 09:57:55 | 0 | 0 |
| preflight | 09:57:55 | 10:05:17 | 0 | 0 |

`pre-add-check` took the changed Go files (`git diff HEAD --name-only` plus untracked, `.go` only). `linux-race` and `windows-race` are `run-on-host --race --need linux` and `--need windows` of `go test -race -count=1 ./...` and
`go -C acpext test -race -count=1 ./...` (the laptop's native shell, no WSL distro). The newest Go file written in this part, `llm/provider/foreignlogin_test.go`, is dated 09:44:28, before the first gate's start (09:46:58). The record edits after
the gates (the owner's later ruling, below) touch no Go file. `make check-records markdownlint` ran after the last record edit.

6. **The owner's ruling on the SDK's draft record (2026-11-17), records only.** The owner chose, for the SDK's draft record for the held session, that the SDK itself refuses the adoption ("The SDK refuses to adopt another sign-in (Recommended)"). That
   record is a proposed draft, not accepted and not released. One dated note beside each statement that a later gobble phase wraps the token source: 0052-PLAN step 1d (the P4 step amendments), the second ruling's list item 1 and R2-B2 in "second review and fixes",
   and 0051-MADR D2's note. The earlier rulings are not rewritten. Two documents promised a mechanism and now state only the limit: `docs/architecture.md` (the `llm/provider` row: gobble "has no refusal for it yet, and the fix is not released") and
   `docs/guides/credentials.md` (the sentence ends at "sends that login's token"). "Third review", item 3, now says the test is the one the later phase makes pass, "whatever mechanism that phase finds".

### Readiness pass for P5 and P6 (2026-11-21, records only)

Written by an agent on the coordinator's brief, after the owner released 0052 from hold that day: "Release it: readiness pass, P5, then L4-L7 kit (Recommended)". **Start state:** branch `main`, `HEAD` `4c3e397`, `git --no-optional-locks status --short` printed nothing. Records only: this PLAN and the MADR were edited, by additions and dated notes; no code, test, script, kit or other document; nothing staged or committed. Run: file reads, searches, git reads, `make check-records` and markdownlint, after this entry was written (their exit statuses are in the hand-back). No test, gate, build or live run.

The MADR's "State table for what remains: D9's migration and P6's acceptance (2026-11-21)" holds the rows: 43 decided, 4 out of scope, 13 open (Q1 to Q8), 4 awaiting live evidence. The notes of this date under P5's and P6's headings add each phase's step 0 (re-ground at the phase's start commit) and the exit criterion "no blocker, every row tested, plants caught".

**1. Re-grounding at `4c3e397`.** Every file, line and symbol P5, P6 and MADR D9 cite, against `9332cef` (the last re-grounding) and `4c3e397`. `git diff --stat 9332cef 4c3e397` changes nothing under `internal/auth/`, `internal/archtest/`, `internal/config/`, `internal/appdirs/`, `internal/filelock/` or `jsonx/`, and in `llm/provider/` only `chatserver_test.go`.

| Citation | Where cited | At `4c3e397` |
| :--- | :--- | :--- |
| `run`: the call after `runEnv` is built and before `kctx.Run` | P5's note of 2026-11-20; step 2 | holds; the lines moved with 0063 (`internal/cli/main.go:90`, `:91`, `:92`, then `tellWaitWarn` at `:93`), P5's note of this date |
| completion mode returns in `Main` before `run` | P5's note | holds, `main.go:45-47` |
| no `auth.Load` before `kctx.Run` | P5's note | holds (`ownListings`, `internal/cli/models.go:90-94`, reads nothing) |
| `member.value`, `SetOAuth`, `merge` | P5's note; D9 step 6 | hold (`internal/auth/store.go:61`, `:220`, `:293`); new: `encode` re-indents a one-line member (`:387-390`), the MADR's rows M9 and M10 |
| `securePrivate`, the Windows DACL | P5's note | holds: `secure_windows.go:7` calls `appdirs.SecurePrivateFile`; `secure_other.go:6` relies on `os.CreateTemp`'s `0600`; `write`, `store.go:412-425` |
| `internal/config/diag.go:44` | D9 step 5 and its note | holds, `textUnknown = "not used by this gobble"` |
| `HostID`, `Clients.Save` | D9's note | hold (`hostid.go:23`, the `O_EXCL` create at `:35`; `clients.go:47`, replacing a record at `:66`) |
| `filelock` is not re-entrant | D9's note | holds; `internal/filelock` unchanged |
| `provider.SetConsent` | D9 step 5; P5 step 2 | holds (`llm/provider/consent.go:168`); how the closure names the consent: P5's note of this date |
| archtest rule 4, `check.go:308` | MADR D3; D9's note; P5 step 1 | `checkRule4` at `:304`, the rejection at `:318-319`; unchanged since `9332cef` |
| `TestImportRules` | P5 step 3 | `internal/archtest/check_test.go:14`; no rule 4 case for `internal/auth` yet |
| the pins | D9's note | acpext moved to `v0.5.0` (`go.mod:20`, `95f71cc`); go-llmprovider-sdk `v1.31.0`, acp-go-sdk2 `v1.36.0` and authcontract `v0.1.0` hold |
| no gobble `v0.7.0` tag; `v0.6.0` exists | D9's note; P6's note | holds (`git tag -l` lists `v0.6.0` alone) |
| magic-cli-remote requires acp-go-sdk2 `v1.36.0` | P6's note | holds (its `go.mod:11`, read) |
| `scripts/go-precheck.sh`: gofmt, golangci-lint over both modules, govulncheck; golint retired | P5's tooling note | holds (`93ecb5e`, 0064). The sibling tools it names and the four whole-tree findings were not re-checked: running them is outside this pass's allowed commands |
| SDK-2, `TestLive_0075ClientReuse` | MADR D5 and "Not verified"; P6's L6 | passed on 2026-11-14, by go-llmprovider-sdk `0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md`, "Phase 6, the owner's run (2026-11-14)"; the MADR's D9 note of this date |
| `0005-PLAN-v1-feature-scope.md` F4-3d; `0025-PLAN-provider-choice-and-auth.md` P6 | P6 step 3 | hold (`:4857`; `:124`) |
| `_meta.gobble.method: "chatgpt_plan"`, `remote` | P6's L4 | hold (`acpserver/authconsent_test.go:57`) |
| `gobble auth consent <provider> [<consent> allow\|deny]` | P6's L7 | holds (`internal/cli/auth.go:411-415`) |
| 0060 on `main` | P5's note | not re-checked; no step of P5 cites it |

**Dependencies on 0063 and 0051** (their decided behaviour taken as given):

* **0063.** P5's call site is `run` as 0063 left it. A warning from `Migrate` is written before `kctx.Run`, so before any prompt; 0063's queued wait warning is written after it (`tellWaitWarn`). L5 depends on 0063: `gobble auth login`'s paste prompt reads through the one terminal input (0063-MADR D1), and a loopback that arrives first ends the paste's read; the kit runs L5 on a real terminal, not under a pipe.
* **0051.** L4 drives 0051's ACP `authenticate` by elicitation (0051-MADR D4: `_meta.gobble.method`, `remote`) with the client of 0051's L1 to L3 kit; L6's logout revokes under gobble's name (0051-MADR D9, its Q3 "No separate test"); acpext is `v0.5.0`. P5 needs nothing of 0051 beyond what P3 built on it (`provider.SetConsent`).

**2. The test map.** Every decided row of the MADR's state table, and the test that pins it.

| Rows | Test | Phase | State |
| :--- | :--- | :--- | :--- |
| M1 | `TestMigrateNothingToDo` | P5 | in step 3 |
| M4, M8 | `TestMigrateDropsDormantKey`, `TestMigrateKeepsEverythingElse` | P5 | in step 3 |
| M11 | `TestMigrateIsIdempotent` | P5 | in step 3 |
| M14, M17, M20 | `TestMigrateAdoptsIDs` | P5 | in step 3 |
| M24, M25, M31 | `TestMigrateCarriesConsent` (M31 also `TestUnknownKeyReported`, existing) | P5 | in step 3 |
| M34, M35, M38 | `TestMigrateBacksUpFirst` | P5 | in step 3 |
| M2, M3, M5, M6, M7, M9, M13, M21, M22, M23, M26, M27, M28, M30, M32, M34 (the name in UTC), M35 (a second plant), M37, M39, M40, M42, M43, M44, M46, M48 | the proposed tests below | P5 | dated addition |
| A1 | P1's sort script, run again; P4's whole-word grep | P6 | step 1 |
| A2, A3, A5 | the existing tests the MADR's rows name | P6 | step 1 |
| A4 | the four existing plan sign-in tests; the key switch: `TestKeySwitchKeepsHostAndClient` | P6 | existing; dated addition |
| A6, A7 | P5's tests and plants; the C6 gates at P6's start commit on three hosts | P5, P6 | step 1 |

**PLAN tests that map to no row:** `TestImportRules`'s new case (P5 step 3). It guards D9 step 5's layering rule, a property of the code, not an input or a state the migration meets; it stays. Every other test of P5's table maps to a row.

**3. Proposed tests (dated additions; P5 unless marked).** Each is seen failing on its plant, the single production change that must turn it red, on a scratch copy (C2). A test whose row depends on an open row (M13 on M12; M42 on M41) is written after the owner's answer, with the plant unchanged.

| Row | Test (package) | Asserts | Planted break |
| :--- | :--- | :--- | :--- |
| M2 | `TestMigrateNothingToDo`, extended (`internal/auth`) | an empty file and a white-space-only file: no write, no backup, no state file | a backup made for a file with no members |
| M3 | `TestMigrateUnreadableFileWritesNothing` (`internal/auth`) | not an object; a member with no string `type`; a duplicate member name; on Linux and macOS a file of mode `000`: an error naming `auth.json`; the file's bytes unchanged; no backup, no state file; `CarryConsent` not called | `Load`'s error dropped, and the migration going on as for an empty file |
| M5 | `TestMigrateDropsDormantKey`, extended | `key` alone; `workspace` alone: removed, one Info record each | removal only when both members are present |
| M6 | `TestMigrateNothingToDo`, extended | an `oauth` entry with neither member and no plan facts: no write, no backup | a backup made whenever an `oauth` entry exists |
| M7 | `TestMigrateKeepsEverythingElse`, extended | an `api_key` entry with `key` and `workspace` beside the edited entry: byte-identical | step 3 applied to every entry, whatever its type |
| M9 | `TestMigrateKeepsEverythingElse`, extended | another provider's member written on one line: byte-identical after step 3 | step 3 writing through `File.encode` as it stands |
| M13 | `TestMigrateSkipsForeignEntry` (`internal/auth`) | an openai `oauth` entry of stored kind `chatgpt`, with a `clientId` and a dormant `key`, the setting true: the key dropped; no `host-id`, no `clients.json`; `CarryConsent` not called | step 4 adopting from any openai `oauth` entry |
| M21 | `TestMigrateAdoptsIDs`, extended | `clients.json` holding another provider's record: kept, `openai`'s added | `clients.json` written with `openai`'s record alone |
| M22 | `TestMigrateBadClientsFileIsAnError` (`internal/auth`) | `clients.json` that is not JSON: an error; its bytes unchanged | the error from `Clients` dropped |
| M23 | `TestMigrateAdoptsIDs`, extended | a plan entry with both ids, no dormant key, the setting absent: the state files written, no backup | a backup made whenever any step has work |
| M26 | `TestMigrateCarriesConsent`, extended | the setting false; absent: `CarryConsent` not called | the carry made whenever the file was read, the value ignored |
| M27 | `TestMigrateCarriesConsent`, extended | the setting true with no openai entry; with an `api_key` entry: not called | the carry made whenever the setting is true |
| M28 | `TestMigrateUnreadableSettingsWritesNothing` (`internal/auth`) | `Settings` gives `(false, false)` with an unanswered plan session and a dormant key: an error; no write, no backup, no state file | `(false, false)` taken as `(false, true)` |
| M30 | `TestMigrateSettingsIsGlobalOnly` (`internal/cli`) | the `Settings` closure over a scratch home: the key true in a trusted project's `settings.yaml` only gives `(false, true)`; true in the global file gives `(true, true)`; no global file gives `(false, true)` | the closure reading the merged settings |
| M32 | `TestMigrateCarriesConsent`, extended | `CarryConsent` returns an error: `Migrate` returns it; the dormant key's removal stands; a second run calls `CarryConsent` again | `CarryConsent`'s error dropped |
| M34 | `TestMigrateBacksUpFirst`, extended | with the clock in a zone other than UTC, the name is `auth.json.bak-` and the UTC time | the name formatted in local time |
| M35 | `TestMigrateBacksUpFirst`, a second plant | as written, owner-only by `appdirs.FileIsOwnerOnly` on the three hosts | the backup created `0644` (Linux, macOS); the backup created without `securePrivate` (Windows) |
| M37 | `TestMigrateBackupFailureWritesNothing` (`internal/auth`) | a fault injected at the backup's write: an error; `auth.json`'s bytes unchanged; no state file; `CarryConsent` not called | `auth.json` written after a failed backup |
| M39 | `TestMigrateStateWriteFailureIsAnError` (`internal/auth`) | the state directory a regular file: an error; `auth.json` migrated and backed up; then, with a state directory, a second run writes the state files and makes no backup | the state write's error dropped |
| M40 | `TestMigrateCompletesAfterInterruption` (`internal/auth`) | a run stopped by a fault after `auth.json`'s write and before step 4: the next run adopts the ids and makes no backup | step 1 computing step 3's changes alone, so the second run stops at "None" |
| M42 | `TestMigrateConcurrentRunsWriteOnce` (`internal/auth`) | two `Migrate` calls racing on one file, under `-race`, ten rounds: one backup, one write of `auth.json`, the file migrated | the changes computed from a read made before the lock is taken |
| M43 | `TestMigrateHostIDRaceKeepsFirst` (`internal/auth`) | `Migrate`'s step 4 racing `HostID`: `host-id` holds one id, which neither replaces | `Migrate` writing `host-id` with `O_TRUNC` in place of `O_EXCL` |
| M44 | `TestCompletionModeDoesNotMigrate` (`internal/cli`) | `Main` with the completion environment set, over a scratch home holding a dormant key: `auth.json` byte-identical, no backup | the call made in `Main` before the completion check |
| M46 | `TestACPStartMigrates` (`internal/cli`) | `gobble acp` over a closed stdin, a scratch home with a dormant key: the key removed, a backup made, nothing on stdout from the migration | the call skipped for `acp` |
| M48 | `TestMigrateErrorIsAWarning` (`internal/cli`) | a scratch home whose state path is a regular file, with a plan entry holding both ids: `gobble auth status` exits 0, its rows on stdout and one warning line on stderr | `run` returning on `Migrate`'s error |
| A4 | `TestKeySwitchKeepsHostAndClient` (`internal/cli`, P6) | after a plan sign-in against the in-process issuer, `gobble auth login openai` with a key: `host-id` and `clients.json` unchanged, and a later plan sign-in names the stored client | the key's path clearing `clients.json`'s `openai` record |

The open rows' tests are written into this table, with their plants, when the owner's answers make them decided rows.

**4. The owner's live checks L4 to L7: what each needs, and the kit (described, not built).** The coordinator briefs the kit after P5 is committed. It takes the shape of 0051-PLAN's L1 to L3 kit ("P8 record (2026-11-21)").

* **What every check shares.** gobble is built from `main` at P5's commit into `<scratch>/live-0052/`, its version line recorded. The ACP client is 0051's form-only `acp-signin-client` (acp-go-sdk2 `v1.36.0`), extended to choose `chatgpt_plan` and, for L7, to send one fixed short prompt in a new session and count the `session_info_update` notifications that carry a title within a bounded wait. Each check is one bounded command the owner starts; nothing is left running. Each runs in a fresh scratch `GOBBLE_HOME` under `<scratch>`, with every `GOBBLE_<ROLE>_DIR` variable and every proxy variable unset; the runner refuses to start unless the resolved config directory is under `<scratch>` (`home_is_scratch=yes`). The owner's own `auth.json`, `settings.yaml` and state directory are never read or written. A result file per check holds fixed words and numbers only (`key=value`). No token, ID token, client id, host id, authorize address, pasted address or code is written by the kit anywhere: what must be compared (a client id before and after; the host id against `clients.json`'s) is compared in the runner's memory and written as `same` or `different`. The address to open is shown on the owner's terminal only. Every sign-in is the owner's, on another device. After each check: `backup_files=0` and `migration_warning=no` (a fresh home has nothing to migrate), `ports_bound` over the sign-in, and `scratch_home_deleted=yes` when the home is done with.
* **L4, a first plan sign-in from another device through ACP elicitation.** *The owner:* starts L4; answers the client's `confirm` form (allow, or decline, either recorded); opens the address shown on the phone; signs in there; pastes into the client's form the address the phone's browser lands on. One client is registered under the owner's ChatGPT account; no generation. *Scripted:* `initialize=ok`; `plan_method=found`; `authenticate` with `_meta.gobble.method: "chatgpt_plan"` and `remote: true`; `elicit_first=confirm`, before any `open_url`; `confirm_text=gobbles`, the form's message compared with `provider.ConsentText` of the same source tree; `authenticate=ok`; then the runner reads `gobble auth status --json`: `session_kind=chatgpt_plan`, `consent=granted` or `declined`, `usable=true`; `host_id_file=present`, `client_record=present`, `client_host_matches_host_id=same`; `ports_bound=none`. The home is kept for L7.
* **L5 and L6, the terminal paste, then reuse after a logout.** One runner, one fresh home. *The owner:* in a real terminal, starts it; it runs `gobble auth login openai` on that terminal; the owner chooses allow at the three-way `confirm`, opens the address shown on another device, signs in, and pastes the address the browser lands on. The runner then runs `gobble auth logout openai`, and the owner signs in again the same way. Last, the owner reads the account's page of authorised applications and types `yes` or `no` for a second gobble entry, and `yes` or `no` for whether the consent was asked again. No generation; L5 registers one client; L6 none if the client is reused. *Scripted:* L5: `login_exit=0`, `session_kind=chatgpt_plan`, `consent=granted`, `host_id_file=present`, `client_record=present`, `client_host_matches_host_id=same`, `ports_bound` (the wizard binds the host's loopback port for a terminal sign-in, P4 step 1c); L6: `logout_exit=0`, `logout_result=logged-out`, `revoke_warning=no` (a warning is a deviation, 0051-PLAN C4), `client_record_after_logout=kept`, `host_id_after_logout=same`, `login2_exit=0`, `client_reused=same` or `different`, `host_id=same`, `consent_asked_again` and `account_second_client` (the owner's words). SDK-2's probe passed (the MADR's D9 note of this date), so `different` is evidence the record has no outcome for beyond D5's fallback; the runner records it, and it goes to the owner as a deviation.
* **L7, the consent end to end.** In L4's home, which holds a plan session; a fresh home would need a sign-in and another registration. *The owner:* starts L7, having agreed to the quota. *Scripted:* `gobble auth consent openai background deny` with stdin from `/dev/null`: `deny_exit=0`, `deny_answer=declined`; a new ACP session and the fixed prompt, then a bounded wait for a title after the turn ends: `title_updates_after_deny=0`; `gobble auth consent openai chatgpt_plan.background_use allow` with stdin from `/dev/null` and no controlling terminal: `allow_exit=0`, `allow_answer=granted`, `allow_stderr_lines=0`; a new session and the same prompt: `title_updates_after_allow=1`. At its end the runner offers the owner `gobble auth logout openai` (`revoke_warning`), then deletes the home. **Quota, corrected:** the PLAN's L7 says "one short title generation"; each session's prompt is itself a request under the owner's plan, so L7 spends two short turns and one title. **Evidence:** the PLAN names "the run's log" for "no title request"; this pass did not check that gobble's log records a title request, so the kit's observable is the client's count of `session_info_update` carrying a title, the one `TestTitleNotMadeSendsNothing` holds offline, and the kit's builder checks whether the log can serve as well.
* **L8, only if the owner chooses Q8 (a).** *The owner:* copies their own `auth.json` and global `settings.yaml` into a fresh scratch home. *Scripted:* P5's build run twice there by a command that migrates (as Q1's answer says); `backup_made`, `backup_owner_only`, `keys_dropped=<count>`, `host_id_written`, `client_record_written`, `consent_carried`, `second_run_wrote=nothing`, `other_members_unchanged=yes` (compared in memory); then the home is deleted. No agent sees the copy.
* **Split of the work.** The owner: every sign-in, every paste and every choice at a `confirm`, the account page, and the agreement to L7's quota. An agent: building gobble and the client, each home's set-up and removal, driving the ACP protocol, `auth status --json` and `auth consent`, the comparisons and the result files, and reading the result files afterwards.

**5. Not done by this pass.** No code; no test run; no kit built; `docs/README.md` not edited (the row text is in the hand-back for the coordinator); nothing staged or committed. P5 waits for the owner's answers to Q1 to Q7, and P6's L8 for Q8.

### Second draft of the readiness pass (2026-11-21)

*(2026-11-21, a dated addition; records only.)* An independent review of the entry above was read by the coordinator, whose disposition directed this second draft by dated additions. The MADR's "Second draft (2026-11-21, after the independent record review)" holds the rows changed and added, the live checks' expected values and the eight questions; this entry holds the seams, the tests and the plants. **Counts:** 50 decided, 5 out of scope, 21 open (Q1 to Q8 of the second draft), 4 awaiting live evidence. The first draft's questions are superseded unasked, and every Q in this entry is the second draft's. Start state as above: `main` at `4c3e397`, records only, nothing staged.

**1. P5 step 1's signature and seams, decided now.** The tests of rows M34, M37, M38, M40, M42, M43 and M51 cannot be written against step 1's `MigrateOptions{AuthPath, StateDir, Settings, CarryConsent, Log *slog.Logger}`, and M55 needs a logger that is not opened before `--json` silences stderr. The step reads, by this note:

* `Migrate(ctx, MigrateOptions{AuthPath, StateDir string; Settings func() (bool, bool); CarryConsent func(ctx context.Context) error; Log func() *slog.Logger; Now func() time.Time}) (Report, error)`. `Log` is called only when there is a record to write, and gives a logger that writes to the log file only (M55); nil writes no record. `Now` gives the backup's time (M34); nil is `time.Now`. Q2's answer may add `PlanSession` (its option (a)) or an interface (its option (c)).
* Test hooks: one unexported package-level variable in `internal/auth`, `migrateHooks`, whose function fields are nil in production and are set only by `internal/auth`'s own tests, which are `package auth` (`internal/auth/store_test.go:1`), so no `export_test.go` is needed, unlike Deviation 2: `afterFirstRead` (between the read without the lock and the lock, M42), `beforeHostIDCreate` (between the check and the create, M43), `failBackup` (M37), `failAuthWrite` (M38), `afterAuthWrite` (a stop or a cancellation after `auth.json` is written and before step 4, M40, M51).

**2. Tests and plants fixed.**

* **`TestImportRules`' new case (no row).** Its plant was a real import in `migrate.go`, which the existing whole-tree case "rule 4 module" already catches (`internal/archtest/check_test.go:50`; `check.go:318-319`). The new case's plant is `checkRule4` exempting `internal/auth`. Since `internal/auth` already imports `llm/provider` (`clients.go:17`, `logins.go:9`), nothing pinned "step 5 calls the SDK only through `CarryConsent`": added `TestMigrateCallsOnlyItsHooks` (`internal/auth`), a source check with `go/parser` that `migrate.go` calls none of `provider.SetConsent`, `provider.Consents`, `provider.PlanConsent` and `provider.SessionKindOf`; plant: a call to `provider.SetConsent` added in `migrate.go`. If the owner answers Q2 with (b), this check is dropped, as that option says.
* **M42** `TestMigrateConcurrentRunsWriteOnce`: the second run is started from `migrateHooks.afterFirstRead` of the first, so the interleaving is forced, not timed. Plant, reworded: the changes written as computed from the read made before the lock, without re-reading under it.
* **M43** `TestMigrateHostIDRaceKeepsFirst`: `HostID` is run from `migrateHooks.beforeHostIDCreate`, so it creates the file between the migration's check and its create. Plant: the migration's create made without `O_EXCL` (check, then create). The first draft's plant, `O_TRUNC`, is caught without a race by M17's `TestMigrateAdoptsIDs`. If Q3 is answered (a), the create is `os.Link` and the plant is the link replaced by a rename.
* **M24** `TestMigrateCarriesConsentEndToEnd` (`internal/cli`, added): a scratch home with an unanswered plan session and the global setting true; after one command that migrates, `auth status --json` shows the consent `granted`. Plant: the closure passes `false`. Its command follows Q1, how the unanswered session is found follows Q2, and its entry has no `kind` member if Q6 is answered (a).
* **M14** `TestMigrateAdoptsIDs` also asserts `host-id` owner-only by `appdirs.FileIsOwnerOnly`. Second plant: the migration's own write of `host-id` made `0644` (Linux, macOS), or without `securePrivate` (Windows).
* **M4** `TestMigrateDropsDormantKey` holds two `oauth` entries with dormant keys and asserts one Info record per provider. Second plant: one record naming every provider.
* **M64**, D9 step 2's "or 5": a backup when only step 5 changes `auth.json` had no row or test. It is open (Q4); its test is written after the answer: `TestMigrateCarriesConsent`, extended, asserting a backup (D9 as written, Q4 (b) or (c)) or none (Q4 (a)).
* **M8** `TestMigrateKeepsEverythingElse`: its plant is named against the migration's own edit path, step 3's removal dropping an unknown field of the edited entry, since step 3 does not go through `merge` (M9).
* **M3** `TestMigrateUnreadableFileWritesNothing`: the file of mode `000` is skipped when the test runs as root.
* **M48** `TestMigrateErrorIsAWarning`: the fixture is a plan entry with both ids and a `clients.json` that does not parse (M22), so the state directory and the log file work and only the migration's warning reaches stderr; the command is one that migrates under Q1's answer. The first draft's fixture, the state path a regular file, would also make `lazyLog.open` print "no log file" (`internal/cli/main.go:391-400`) if the log opened.
* **M9** leaves the map of decided rows: it is open (Q5), and its proposed extension of `TestMigrateKeepsEverythingElse` is written after the answer, with its plant.
* **M47** is replaced by M55; its first draft named no test.

**3. Proposed tests for the rows decided in the second draft (P5).** Each seen failing on its plant on a scratch copy (C2).

| Row | Test (package) | Asserts | Planted break |
| :--- | :--- | :--- | :--- |
| M41 | `TestMigrateNothingToDoTakesNoLock` (`internal/auth`) | no `auth.json`, its directory absent: neither the directory nor `auth.json.lock` is created; a file with nothing to migrate and no lock file: no lock file created | the lock taken before the first read |
| M49 | `TestMigrateDropsDormantKey`, extended | an `oauth` entry under a provider name gobble does not know, with a dormant `key`: the key removed; no id adopted; `CarryConsent` not called | step 3 limited to the providers of `provider.KeyProviders()` (`llm/provider/ambient.go:83`) |
| M50 | `TestMigrateDropsDormantKey`, extended | `"key": ""`, `"key": null`, `"workspace": ""`: each member removed | the dormant member found by `e.Key != "" \|\| e.Workspace != ""` |
| M51 | `TestMigrateStopsWhenCancelled` (`internal/auth`) | a context already cancelled: the context's error, nothing written; a context cancelled from `migrateHooks.afterAuthWrite`: `auth.json` migrated and backed up, no state file, the context's error; the next run writes the state files and no backup | the steps going on after the cancellation |
| M51 | `TestMigrateSignalSkipsTheCommand` (`internal/cli`) | `Main` with a context cancelled by a signal's cause and a scratch home with a dormant key: the command does not run (nothing on stdout), and the exit status is 128 plus the signal | `run` going on to `kctx.Run` after a cancelled migration |
| M52 | `TestMigrateSkippedWithoutDirs` (`internal/cli`) | `GOBBLE_HOME` relative, which `appdirs` refuses (`internal/appdirs/dirs.go:157-161`): a migrating command prints the migration's skip warning once on stderr; `auth status --json` prints nothing of it on stderr. The commands follow Q1 | the skip warning written under `--json` |
| M53 | `TestMigrateSettingsIsGlobalOnly`, extended (`internal/cli`) | the global file holding `chatgptPlanBackground: "true"`, `yes`, `1`: `(false, true)` each | the closure taking the string `true` as true |
| M54 | `TestMigrateRunsOnAnyBuild` (`internal/cli`) | with the build's version a development string, as a `main` build reports, a migrating command drops a dormant key | the call gated on a tagged version |
| M55 | `TestMigrateSilentUnderJSONAndACP` (`internal/cli`) | the fixture of M48; `gobble auth status --json` (if Q1 lets it migrate), `gobble auth login --json` with a provider gobble does not know, and `gobble acp` over a closed stdin: stderr empty each; the log file holds one Warn record naming the error | the error written with `Output.Warnf` whatever the command; second plant: `Migrate` given the run's log opened before `kctx.Run`, with stderr attached |

The duplicated-key case of the review's finding 1m was not checked against the settings loader in this pass: a file the loader refuses is M28's.

**4. The live checks.** The MADR's L table now gives each check's expected values from the record. The kit described in the entry above already produces each of those words; L6's `consent_asked_again` and `account_second_client` are the owner's, and their expected values are `yes` and `no`. If Q8 is answered (a), L8 runs right after P5's commit, before any build carrying P5 runs on the owner's own home (M54).

**5. Not done.** No code; no test run; no kit built; `docs/README.md` not edited; nothing staged or committed. `make check-records` was run after this entry was written; its exit status is in the hand-back.

### Third draft of the readiness pass (2026-11-21)

*(2026-11-21, a dated addition; records only.)* The coordinator's delta check of the second draft found M55 not buildable as worded (a blocker for P5) and fixes to the tests and the questions. The MADR's "Third draft (2026-11-21, after the coordinator's delta check)" holds the rows replaced and added and the eight questions; this entry holds what changes in P5 and P6. **Counts:** 51 decided, 5 out of scope, 21 open (the third draft's Q1 to Q8), 4 awaiting live evidence. Every Q in this entry is the third draft's. Start state as above; records only, nothing staged.

**1. P5 step 1's `Log`, by the third draft's M55.** The second draft's "a logger that writes to the log file only" has nothing to build on: `lazyLog` holds one logger whose handler fans out to the file and to stderr (`internal/cli/main.go:382-396`; `internal/logging/logging.go:44-68`), and a second `logging.Open` would make two rolling writers on one file. The step reads, by this note:

* `lazyLog` keeps the file's handler that `logging.Open` builds and gives a file-only view of it, a `*slog.Logger` over that one handler; `MigrateOptions.Log` returns that view, opening the log on its first call. `internal/logging` exposes the file's handler for it (`internal/logging/logging.go:44-50`). **Files added to P5:** `internal/logging/logging.go` and its test, named here for the owner's approval of P5 with the questions.
  *(2026-11-21: approved by the owner, "Approve: add logging.go to P5 (Recommended)", relayed by the coordinator. The
  change exposes the existing file handler as a file-only view, and nothing else. Both files are in P5's file list of
  "The owner's answers: P5 and P6 settled (2026-11-21)".)*
* Before `Migrate`, the call site in `run` applies the whole silencing of `auth status --json` and `auth login --json`, read from the parsed command (`out.err = io.Discard`, `logs.stderr = nil`, as `internal/cli/auth.go:72-73` and `:522-523` do in `Run`). Under those two commands and `gobble acp`, a log file that cannot be opened writes nothing to stderr; today `lazyLog.open` writes "no log file" through `Output.Warnf` (`main.go:392`, `:400`).
* `Now` and `migrateHooks` stand as the second draft has them.

**2. Tests and plants changed.**

| Row | Test | Change | Planted break |
| :--- | :--- | :--- | :--- |
| M55 | `TestMigrateSilentUnderJSONAndACP` (`internal/cli`) | also with the log file unable to open (the state directory's `logs` made a regular file), the migration failing on M48's fixture: stderr empty under each of `auth status --json` (if Q1 lets it migrate), `auth login --json` and `gobble acp` | added: the "no log file" line reaching stderr under `auth status --json`; and, replacing the second draft's second plant, `Migrate`'s `Log` given the shared logger, so its Warn reaches stderr |
| M37 | `TestMigrateBackupFailureWritesNothing` (`internal/auth`) | `migrateHooks.failBackup` fires after bytes are written to the temporary file; asserts that neither a backup nor a temporary file remains, `auth.json`'s bytes unchanged, no state file, `CarryConsent` not called | added: the temporary file not removed on failure |
| M51 | `TestMigrateSignalSkipsTheCommand` (`internal/cli`) | runs `gobble version`, which prints whatever the context holds, and asserts its output is absent, besides the exit status of 128 plus the signal (`main.go:57-59` gives that status for any signal cause) | unchanged: `run` going on to `kctx.Run` after a cancelled migration |
| M52 | `TestMigrateSkippedWithoutDirs` (`internal/cli`) | counts the migration's skip line on stderr, not all of stderr | unchanged |
| M65 | `TestMigrateSettingsIsGlobalOnly`, extended (`internal/cli`) | a global `settings.yaml` holding `chatgptPlanBackground` twice: `(false, false)`. This replaces the second draft's note that the duplicated-key case was not checked | the closure taking the last value of a duplicated key |
| (no row) | `TestMigrateCallsOnlyItsHooks` (`internal/auth`), narrowed | asserts with `go/parser` that `migrate.go` does not import `llm/provider` at all, and calls no `internal/auth` function whose body uses it: `AcceptsLogin` (which calls `provider.Accepts`, `internal/auth/resolve.go:38-40`) and the methods of `Logins` and `ClientLogins`; `Clients`' methods use no `llm/provider` | added: `Migrate` calling `AcceptsLogin`; kept: a call to `provider.SetConsent` |
| M43 | `TestMigrateHostIDRaceKeepsFirst` | if Q3 is answered (a), the create is a rename under `host-id.lock`, not `os.Link` as the second draft supposed | under Q3 (a): the rename made without the check under `host-id.lock` |

**Note for the builder.** `migrateHooks` is package state, and the M42 hook fires once per run of the test: no test that sets `migrateHooks` uses `t.Parallel`, and each restores the hooks with `t.Cleanup`.

**3. The live checks.** The MADR's L4 to L7 expected lists now hold the kit's shared `backup_files=0` and `migration_warning=no`, which the kit already writes for every check. If Q8 is answered (a), L8 also copies the owner's `host-id` and `clients.json`, where present, into its scratch home, and compares them only in memory.

**4. Not done.** No code; no test run; no kit built; `docs/README.md` not edited; nothing staged or committed. `make check-records` was run after this entry was written; its exit status is in the hand-back.

### The owner's answers: P5 and P6 settled (2026-11-21)

*(2026-11-21, a dated amendment; records only.)* The owner answered the MADR's third-draft Q1 to Q8, each as recommended, relayed by the coordinator: Q1 "All but completion, status, check (Recommended)"; Q2 "A second hook, PlanSession (Recommended)"; Q3 "Validate; lock-and-rename host-id (Recommended)"; Q4 "Check-then-rename; back up key drops only (Recommended)"; Q5 "Keep members' bytes, not whitespace (Recommended)"; Q6 "Only into a kind-less entry; accept window (Recommended)"; Q7 "Resolve it; migrate the target (Recommended)"; Q8 "An owner-run L8 first (Recommended)". And the logging file: "Approve: add logging.go to P5 (Recommended)". The MADR's "The owner's answers (2026-11-21)" records them as rows; **counts:** 72 decided, 5 out of scope, 0 open, 5 awaiting live evidence (L4 to L8). This entry settles P5's steps, signature, files and tests, and L8; where an earlier entry of this record says otherwise for P5, this entry holds. Records only; nothing staged.

**1. P5's files.**

* `internal/auth/migrate.go` (new) and `internal/auth/migrate_test.go` (new).
* `internal/auth/hostid.go` (the `host-id.lock` helper; `HostID`'s create changed) and `internal/auth/hostid_test.go`.
* `internal/archtest/check_test.go` (rule 4's new case for `internal/auth`).
* `internal/cli/main.go` (the call, the exemption list, the silencing, `lazyLog`'s file-only view), `internal/cli/migrate.go` (new: the `Settings`, `PlanSession` and `CarryConsent` closures) and `internal/cli/migrate_test.go` (new).
* `internal/logging/logging.go` and `internal/logging/logging_test.go`: approved by the owner, "Approve: add logging.go to P5 (Recommended)"; the change exposes the existing file handler as a file-only view, and nothing else.
* Docs (C9): `docs/guides/credentials.md` (the migration, its backup, `host-id.lock`) and `docs/architecture.md` (`internal/auth`'s migration; `internal/logging`'s view).

**2. Step 1, the signature.** `Migrate(ctx context.Context, o MigrateOptions) (Report, error)`, with `MigrateOptions{AuthPath, StateDir string; Settings func() (bool, bool); PlanSession func(ctx context.Context) (plan, answered bool, err error); CarryConsent func(ctx context.Context) error; Log func() *slog.Logger; Now func() time.Time}`. `Log` gives `lazyLog`'s file-only view, opened at its first call; nil writes no record. `Now` gives the backup's time; nil is `time.Now`. `Report` names each provider whose key or workspace was dropped, the backup's file name, and whether `host-id`, the client record and the consent were written. Test hooks: `migrateHooks{afterFirstRead, beforeHostIDCreate, failBackup, failAuthWrite, afterAuthWrite}` and, in `hostid.go`, `hostIDHooks{afterRead}`, package variables nil in production, set only by `internal/auth`'s own tests (no `t.Parallel`; restored with `t.Cleanup`).

**3. Step 1, the order of `Migrate`.** The context is checked before each step; a write already begun completes (M51).

1. **The path** (Q7): the target is `filepath.EvalSymlinks(AuthPath)`, or `AuthPath` itself when no file is there; any other error is returned. The lock is always `AuthPath + ".lock"`, beside the given path.
2. **The read without the lock** (M41): the target is read; a file that cannot be read or parsed is an error (M3). The work is computed: step 3 has work when an `oauth` entry has a `key` or `workspace` member, whatever its value (M50); `PlanSession` is called when the file has an openai `oauth` entry; when the plan session is unanswered and its entry has no `kind` member (its presence in the entry's bytes, not its value: Q6), `Settings` is asked, and `(false, false)` is an error with nothing written (M28); step 5 has work when it then gives `(true, true)`; step 4 has work when the plan entry holds ids the state files lack. No work: return, with no lock taken and nothing created.
3. **The lock**: `migrateHooks.afterFirstRead`, then `auth.json.lock` taken (`authPolicy`, up to 30 s); the target re-read and the work computed again; none: release and return.
4. **The backup** (Q4), only when step 3 has work: a temporary file beside the target, owner-only before its first byte (`securePrivate`), written and synced (`migrateHooks.failBackup` after the bytes); the name `auth.json.bak-` and `Now().UTC()` as `YYYYMMDDTHHMMSSZ` beside the target; under the lock, the name checked free, else the temporary file removed and an error with nothing written (M36); then the rename. The temporary file is removed on any failure (M37).
5. **Step 3's write** (Q5): the migration's own encoder writes the top level as `encode` does (LF, two spaces, a final newline), each unchanged member's bytes verbatim, a one-line member included, and each edited entry with its `key` and `workspace` members removed by name and laid out as gobble's writes do; written through `write` (a temporary file and a rename) to the target (`migrateHooks.failAuthWrite` before the rename, `afterAuthWrite` after). One Info record per provider through `Log`, naming the provider and the backup's file name.
6. **Step 4** (Q3), when `PlanSession` reports a plan entry: its `hostId` must be a `urn:uuid:`, else an error and nothing adopted (M57); `host-id` placed by the helper of item 4 (`migrateHooks.beforeHostIDCreate` first); a `host-id` holding anything but a `urn:uuid` and not empty is an error (M19); a client record written by `Clients.Save` only when the entry holds both ids, the id the helper returns equals the entry's `hostId` (M18, M43), and `clients.json` has no `openai` record (M20).
7. **Step 5**: `auth.json.lock` released; then, when step 5 has work, `CarryConsent`; its error is returned (M32). Gobble does not guard the window between the check and the carry (M33).

**4. `host-id`** (Q3; MADR D4's note of this date). `hostid.go` gains `placeHostID(dir, id string) (string, error)`: it reads `host-id`; a valid id is returned as it is; a non-empty file that holds no `urn:uuid` is an error; for an absent or empty file, `hostIDHooks.afterRead`, then `host-id.lock` (`filelock`, `authPolicy`) is taken and the file read again: a valid id is returned, an invalid one is an error, and an absent or empty one is replaced by a temporary file, owner-only before its first byte and synced, renamed onto `host-id`; the id written is returned. `HostID(dir)` reads the file and, when it is absent or empty, returns `placeHostID(dir, newHostID())`. The `O_EXCL` create and `awaitHostID` are removed. The directory is still made `0700`.

**5. `run`** (Q1; M55). In `internal/cli/main.go`, after `e.ownListings()` and before `kctx.Run(e)`: no migration for `auth status` or `auth check` (`kctx.Command()`), with or without `--json`; for `auth login` with `--json` read from the parsed command, `out.err = io.Discard` and `logs.stderr = nil` first; when the config or the state directory cannot be resolved, the migration is skipped with one warning (M52), on stderr except under `auth login --json` and `acp`; an error from `Migrate` is one warning line on stderr (M48), or under `auth login --json` and `acp` a Warn record through the file-only view only; when the context is done after `Migrate`, `run` returns without `kctx.Run` (M51). `lazyLog` gains `fileOnly()`, a logger over the file handler that `internal/logging` now exposes, the log opened at its first use; `lazyLog.open`'s "no log file" warning is not written to stderr when `stderr` is nil (`acp`, and the silenced `--json`).

**6. The closures** (`internal/cli/migrate.go`; Q2). `Settings`: the global `settings.yaml` scalar read by its own parse; only the YAML boolean `true` gives `(true, true)`; a missing file `(false, true)`; a file that cannot be read or parsed, a duplicated key included, `(false, false)` (M53, M65). `PlanSession`: `provider.PlanConsent("openai", auth.Logins{Path})` gives the consent's id and `Plan`; `provider.Consents` gives that consent's answer, `answered` when it is not `not asked`; `ErrNoStoredSession` is `(false, false, nil)`; a store error is returned. `CarryConsent`: `provider.SetConsent(ctx, auth.Logins{Path}, "openai", <the id PlanConsent gave>, true)`. `internal/cli` imports no SDK package for them; `migrate.go` in `internal/auth` imports no `llm/provider`.

**7. Tests the answers add or change** (P5). Each seen failing on its plant on a scratch copy (C2). The tests of the earlier entries stand where this table does not change them.

| Rows | Test (package) | Asserts | Planted break |
| :--- | :--- | :--- | :--- |
| M45, M56 | `TestMigrateRunsExceptStatusAndCheck` (`internal/cli`) | a scratch home with a dormant key: `gobble auth status`, `gobble auth status --json` and `gobble auth check --provider openai --json` leave `auth.json` byte-identical, with no backup and no state file; `gobble version` then drops the key | `auth check` left out of the exemption list |
| M12 | `TestPlanSessionHook` (`internal/cli`) | the closure over a scratch `auth.json`: a plan entry with no `kind` member and no answer gives `(true, false)`; granted, `(true, true)`; another application's ChatGPT login, not a plan; no entry, `(false, false, nil)`; an unparseable file, an error | the closure deciding the plan from the stored `kind` alone |
| M12 | `TestMigrateCallsOnlyItsHooks` (`internal/auth`) | as the third draft narrowed it | as the third draft gives them |
| M29 | `TestMigrateUnreadableSettingsWritesNothing`, extended | `Settings` would give `(false, false)`, no unanswered plan session, a dormant key: `Settings` not called; the key dropped | `Settings` asked on every run that has work |
| M15, M16, M18, M19, M57 | `TestMigrateAdoptsIDs`, extended | `hostId` alone: `host-id` written, no record; `clientId` alone: nothing adopted; `host-id` holding another valid id: kept, no record; `host-id` holding text that is not a `urn:uuid`: an error, nothing adopted, the file unchanged; an entry's `hostId` that is not a `urn:uuid`: an error, nothing adopted | one plant each: a record with an empty `clientId`; a record with an empty `hostId`; the record written with the entry's `hostId`; the invalid file replaced; the entry's `hostId` copied unchecked |
| M58 | `TestHostIDReplacesEmptyFile` (`internal/auth`) | an empty `host-id`: `HostID` returns a new `urn:uuid`, which the file then holds; the migration over an empty `host-id` writes the plan entry's `hostId` | an empty file treated as present |
| M58, M43 | `TestHostIDLockedReplaceKeepsFirst` (`internal/auth`) | a second `HostID` started from `hostIDHooks.afterRead` of the first, over an absent and over an empty file: the file holds one id, and both calls return it | the rename made without the read again under `host-id.lock` |
| M43 | `TestMigrateHostIDRaceKeepsFirst` | `HostID` run from `migrateHooks.beforeHostIDCreate`: `host-id` holds `HostID`'s id; no client record | the same plant |
| (P2-1's) | `TestHostIDIsStable`, existing | as written | its plant was "`O_EXCL` dropped"; it is now the read again under `host-id.lock` skipped |
| M36 | `TestMigrateBackupNameTaken` (`internal/auth`) | `Now` fixed and a file at the backup's name: an error; `auth.json` byte-identical; that file unchanged; no temporary file left | the rename made over the existing name, with no check |
| M64, M59, M33, M62 | `TestMigrateCarriesConsent`, extended | a carry alone: no backup; a `CarryConsent` that fails on two runs: no backup on either; inside the `CarryConsent` stub, a `filelock.TryLock` of `auth.json.lock` succeeds; an unanswered plan entry with a `kind` member: not carried into; one with none: carried into | in turn: a backup made when only step 5 has work; `CarryConsent` called while `auth.json.lock` is held; the `kind` presence check dropped |
| M59 | `TestMigrateBacksUpFirst`, extended | `failAuthWrite` on two runs a second apart, then a run that succeeds: three backups, one per attempt | a backup skipped when one already exists |
| M60 | `TestMigrateDropsDormantKey`, extended | `Report` names each provider whose key was dropped and the backup's file name, with `Log` nil as well | `Report` naming nothing when there is no logger |
| M9 | `TestMigrateKeepsEverythingElse`, extended | an unchanged member written on one line: its bytes verbatim after step 3 | step 3 written through `File.encode` as it stands |
| M10, M61 | `TestMigrateLayout` (`internal/auth`) | a file with CRLF line ends, four-space indentation, no final newline, one compact unchanged member and one entry to edit: after step 3, LF between members, two-space indentation at the top level, a final newline, the compact member's bytes verbatim, the edited entry laid out as gobble's writes | the file's own CRLF kept at the top level; the edited entry written compact |
| M63 | `TestMigrateFollowsSymlink` (`internal/auth`; skipped where the OS will not let the test make a symbolic link) | `auth.json` a link to a file in another directory: the target migrated; the link still a link to it; the backup beside the target; nothing beside the link but `auth.json.lock` | the path not resolved, so the link is replaced by a file; the lock taken beside the target |
| M24 | `TestMigrateCarriesConsentEndToEnd` (`internal/cli`) | its plan entry has no `kind` member; the command is `gobble version`; then `auth status --json` shows the consent `granted` | the closure passing `false` |
| M48, M51 | `TestMigrateErrorIsAWarning`, `TestMigrateSignalSkipsTheCommand` | their command is `gobble version` | as before |
| M52, M55 | `TestMigrateSkippedWithoutDirs`, `TestMigrateSilentUnderJSONAndACP` | their `--json` case is `gobble auth login --json` with a provider gobble does not know; `auth status --json` does not migrate | the "no log file" line reaching stderr under `auth login --json`; the others as before |
| M55 | `TestFileOnlyView` (`internal/logging`) | the file-only view from `Open` writes a Warn record to the log file and nothing to the `Stderr` writer `Open` was given | the view built over the fan-out handler |

**8. L8** (Q8; the MADR's L8 row). It runs right after P5's commit, before any build carrying P5 runs on the owner's own home, and before P6's L4 to L7. *The owner:* copies their own `auth.json`, global `settings.yaml`, and `host-id` and `clients.json` where present, into a fresh scratch home, and starts L8. *Scripted:* the shared rules of the kit (the readiness entry's item 4: a fresh `GOBBLE_HOME`, every `GOBBLE_<ROLE>_DIR` and proxy variable unset, `home_is_scratch=yes`, fixed words only, no value written anywhere); a snapshot of the files taken in memory; `gobble version` run twice, which migrates under Q1 and reaches no service; each member compared in memory with the snapshot (Q5: by member); the words of the MADR's L8 row written; the home deleted. No agent sees the files or their values.

**9. P5's start.** No row is open, and P5's dependencies on the owner's answers are met. P5 starts on the owner's word to start it, with its step 0, the re-grounding at its start commit. Its exit criterion: no blocker, every row tested, plants caught.

**10. Not done.** No code; no test run; no kit built; `docs/README.md` not edited (the row text is in the hand-back); nothing staged or committed. `make check-records` was run after this entry was written; its exit status is in the hand-back.

### P5 record (2026-11-22), pass 1

Built by an agent on the coordinator's brief, on the owner's word relayed by the coordinator, "send an agent to work gobble P5". **Start state:** branch `main`, `HEAD` `0589454`. `git --no-optional-locks status --short` showed only another agent's files, written at the same time (`docs/decisions/0004-MADR-go-module-architecture.md` and new `docs/decisions/0065-*`); this phase does not touch them. Nothing staged or committed by this phase.

#### Deviation 5 (2026-11-22): the migration's warning doubles a command's own report of an unreadable `auth.json`; and the client record's check-then-save window

**Found** during pass 1, after every P5 test was green in `internal/auth`, `internal/logging` and `internal/archtest`. `go test ./internal/cli/` failed one test, `TestAuthConsentCommand` (`internal/cli/consent_test.go:84-95`). It was in the tree before P5 and pins 0052-MADR D3's note, "Every error is one line on stderr". With `auth.json` holding `{"openai": `, `gobble auth consent openai` and its `allow` and `deny` forms printed two lines: `Warning: auth.json migration: …invalid auth.json…`, then `Error: …invalid auth.json…`. The decided rows meet there and no row settles it:

* Q1 (a): `auth consent` migrates.
* M3: an unreadable `auth.json` is a migration error.
* M48: that error is one stderr warning line.
* D3: the command allows one error line.

The same doubling happens in every command that reads `auth.json` itself. Only D3's test pins it.

**Also found:** step 4's client record was written by `Clients.Load`, then `Clients.Save`, with no lock between the two. `Clients.Save` replaces a record, so a record another process saved in that window would be overwritten. That is against D9's note of 2026-11-20, "an existing client record is left". `internal/auth/clients.go` was outside P5's file list.

**Decisions (owner, 2026-11-22, relayed by the coordinator):**

1. The doubled report: **"Command reports it; log keeps it (Recommended)"**. An `auth.json` that cannot be read or parsed is reported by the command, not by the migration. `Migrate` still returns the error and writes nothing (M3 stands). The call site tells this case apart by a sentinel from `internal/auth` and writes it to the log's file-only view only, never to stderr. Every other migration error stays one stderr warning (M48). This is the MADR's row M66.
2. The client record's window: **"Guard it in P5 (Recommended)"**. The record's read-modify-save runs under `clients.json.lock`. This is the MADR's row M67.

**Files added to this phase:** `internal/auth/clients.go`, which gains a save-if-absent under `clients.json.lock`. `internal/auth/clients_test.go` is not changed; M67's test is in `internal/auth/migrate_test.go`.

**Tests added.** Each is seen failing first, each with its plant:

| Row | Test (package) | Asserts | Planted break |
| :--- | :--- | :--- | :--- |
| M66 | `TestMigrateUnreadableAuthJSONIsTheCommands` (`internal/cli`) | `auth.json` holding `{"openai": `: `gobble version` exits 0 with nothing on stderr; the log file holds one Warn record of the migration naming `auth.json`; `auth.json` unchanged. `TestAuthConsentCommand` passes unchanged | the call site writing the unreadable-file error to stderr, as any other |
| M66 | `TestMigrateUnreadableFileWritesNothing`, extended (`internal/auth`) | each unreadable case's error is `ErrUnreadable` (`errors.Is`) | the read's and the parse's errors returned without the sentinel |
| M67 | `TestMigrateClientRaceKeepsFirst` (`internal/auth`) | another process's `Clients.Save` of an `openai` record, run from `migrateHooks.beforeClientSave` between step 4's check and its save: that record is kept; `Report.ClientRecord` is false | the save made with no check, under the lock, that the record is still absent |

#### Pass 1, execution (2026-11-22)

**Step 0, re-grounding at `0589454`: nothing moved.** `git diff --stat 4c3e397 0589454` touches only this pair and `docs/README.md`. Every file, line and symbol that P5, D9 and the settled entry cite was re-read and holds. Among them: `internal/cli/main.go:45-47, 57-59, 87-94, 149-171, 350-353, 382-407`; `internal/auth/store.go:28, 61, 90-92, 220, 286, 293, 358-375, 387-390, 412-425`; `hostid.go:23, 35, 59-75, 86-87`; `clients.go:17, 47, 56-59, 66`; `logins.go:9, 15`; `resolve.go:38-40`; `secure_windows.go:7`; `secure_other.go:6`; `internal/logging/logging.go:12, 44-68`; `internal/cli/auth.go:72-73, 411, 522-523`; `llm/provider/consent.go:115-130, 168-179`; `internal/archtest/check.go:304, 308, 315, 318-319`; `check_test.go:14, 50`; `internal/config/diag.go:44`; `resolve_test.go:267`; `internal/appdirs/dirs.go:157-161`; `internal/cli/signal.go:91-92`; `models.go:90-94`; `llm/provider/ambient.go:83`; `internal/fsx/regular.go:35`; `acpserver/authconsent_test.go:57`; `SessionKindOf` at `llm/provider/signinrun.go:75`.

**What was built** (unstaged; the coordinator stages, the owner commits):

* `internal/auth/migrate.go` (new): `Migrate`, `MigrateOptions` (the settled signature), `Report`, `ErrUnreadable` (M66), and `migrateHooks`. `migrateHooks` holds the settled five hooks, plus `beforeClientSave` for M67 (Deviation 5). The steps are those of the settled entry's item 3.
* `internal/auth/hostid.go`: `placeHostID`, the rename under `host-id.lock`. `HostID` places through it. The `O_EXCL` create and `awaitHostID` are removed. `hostIDHooks{afterRead}` is added. `validHostID` is the one `urn:uuid` rule, used by `readHostID` and by the migration.
* `internal/auth/clients.go` (Deviation 5): `Save` and the new `saveIfAbsent` share one read-modify-write under `clients.json.lock` (`update`).
* `internal/logging/logging.go`: `OpenFileView`, the file-only view over the one file handler. `Open` is unchanged for its callers.
* `internal/cli/main.go`: `migrateAuth`, called between `e.ownListings()` and `kctx.Run`. It holds the exemption of `auth status` and `auth check`, the silencing under `auth login --json`, M48/M52/M55/M66's outputs, and M51's skip. `loginJSON` reads `--json` from the parsed command. `lazyLog` gains `fileOnly()`, and its "no log file" line is not written when the log has no stderr.
* `internal/cli/migrate.go` (new): the `Settings`, `PlanSession` and `CarryConsent` closures of the settled entry's item 6.
* Tests: `internal/auth/migrate_test.go` (new), `hostid_test.go`, `internal/archtest/check_test.go` (rule 4 for `internal/auth`), `internal/cli/migrate_test.go` (new), `internal/logging/logging_test.go`.
* Docs: `docs/guides/credentials.md` (the host id under `host-id.lock`; the files an unfinished sign-in leaves; a new section, "The one-time migration of `auth.json`") and `docs/architecture.md` (the `internal/logging`, `internal/auth` and `internal/cli` rows).

**Red first.** Every test was written before its code. `go test` ran against stubs: `Migrate` a no-op; `hostIDHooks` declared and unused; `OpenFileView` with a discarding view; the cli closures returning zero, and no call in `run`; `ErrUnreadable` declared and not wrapped; `beforeClientSave` placed between a `Load` and a `Save`. Each test failed for its stated reason, never a build failure: a missing backup or state file, an error not returned, a record not kept, stderr not empty, and so on. Seven tests pass against a no-op stub, so their only red is their plant: `TestMigrateNothingToDo`, `TestMigrateNothingToDoTakesNoLock`, `TestMigrateCallsOnlyItsHooks`, `TestImportRules`' new case (the existing whole-tree rule 4 already rejects the import), and `TestCompletionModeDoesNotMigrate`.

**Rows to tests,** as the MADR's state table, its answers and this record's Deviation 5 map them. M1, M2, M6: `TestMigrateNothingToDo`. M3, M66: `TestMigrateUnreadableFileWritesNothing`. M4, M5, M49, M50, M60: `TestMigrateDropsDormantKey`. M7, M8, M9: `TestMigrateKeepsEverythingElse`. M10, M61: `TestMigrateLayout`. M11: `TestMigrateIsIdempotent`. M13: `TestMigrateSkipsForeignEntry`. M14 to M21, M23, M57: `TestMigrateAdoptsIDs`. M22: `TestMigrateBadClientsFileIsAnError`. M24 to M27, M32, M33, M59, M62, M64: `TestMigrateCarriesConsent`. M24 end to end: `TestMigrateCarriesConsentEndToEnd`. M28, M29: `TestMigrateUnreadableSettingsWritesNothing`. M30, M53, M65: `TestMigrateSettingsIsGlobalOnly`. M12: `TestPlanSessionHook`, `TestMigrateCallsOnlyItsHooks`. M34, M35, M38, M59: `TestMigrateBacksUpFirst`. M36: `TestMigrateBackupNameTaken`. M37: `TestMigrateBackupFailureWritesNothing`. M39: `TestMigrateStateWriteFailureIsAnError`. M40: `TestMigrateCompletesAfterInterruption`. M41: `TestMigrateNothingToDoTakesNoLock`. M42: `TestMigrateConcurrentRunsWriteOnce`. M43: `TestMigrateHostIDRaceKeepsFirst`. M58: `TestHostIDReplacesEmptyFile`, `TestHostIDLockedReplaceKeepsFirst`. M44: `TestCompletionModeDoesNotMigrate`. M45, M56: `TestMigrateRunsExceptStatusAndCheck`. M46: `TestACPStartMigrates`. M48: `TestMigrateErrorIsAWarning`. M51: `TestMigrateStopsWhenCancelled`, `TestMigrateSignalSkipsTheCommand`. M52: `TestMigrateSkippedWithoutDirs`. M54: `TestMigrateRunsOnAnyBuild`. M55: `TestMigrateSilentUnderJSONAndACP`, `TestFileOnlyView`. M66: `TestMigrateUnreadableAuthJSONIsTheCommands`, with `TestAuthConsentCommand` unchanged. M67: `TestMigrateClientRaceKeepsFirst`. M31: `TestUnknownKeyReported` (existing). No decided M row is without a test.

**Plants.** Each was run through go-llmprovider-sdk's `tools/plant.py --root <this repo>` on a scratch clone. There is one spec per target test, whose command is `go test -count=1 -run '^<Test>$' <package>`, so "caught" means that test failed. The tool's own summary lines were all read, and no catch was a build failure. 79 plant runs: 78 caught by their target test; the 79th is the expected miss below. The first plant run, one spec per package, counted five plants as caught on a build failure (a variable they left unused: M13, M18, M25, M26, M62). They were fixed to compile and run again; the table is the second run.

| Plant | Caught by | Caught |
| :--- | :--- | :--- |
| rule 4: checkRule4 exempting internal/auth | `TestImportRules` | yes |
| M43/M58 the rename made without the read again under host-id.lock | `TestHostIDIsStable`, `TestHostIDLockedReplaceKeepsFirst`, `TestMigrateHostIDRaceKeepsFirst` | yes |
| M58 an empty host-id treated as present | `TestHostIDReplacesEmptyFile` | yes |
| M14 host-id written 0644 | `TestMigrateAdoptsIDs` | yes |
| M17/M18 an existing host-id replaced | `TestMigrateAdoptsIDs` | yes |
| M21 clients.json written with openai's record alone | `TestMigrateAdoptsIDs` | yes |
| M23 a backup made whenever any step has work | `TestMigrateAdoptsIDs` | yes |
| M15 a record with an empty clientId | `TestMigrateAdoptsIDs` | yes |
| M16 a record with an empty hostId | `TestMigrateAdoptsIDs` | yes |
| M18 the record written with the entry's hostId | `TestMigrateAdoptsIDs` | yes |
| M19 the invalid host-id replaced | `TestMigrateAdoptsIDs` | yes |
| M57 the entry's hostId copied unchecked | `TestMigrateAdoptsIDs` | yes |
| M38 the backup written after | `TestMigrateBacksUpFirst` | yes |
| M34 the name formatted in local time | `TestMigrateBacksUpFirst` | yes |
| M35 the backup created 0644 | `TestMigrateBacksUpFirst` | yes |
| M59 a backup skipped when one already exists | `TestMigrateBacksUpFirst` | yes |
| M37 auth.json written after a failed backup | `TestMigrateBackupFailureWritesNothing` | yes |
| M37 the temporary file not removed on failure | `TestMigrateBackupFailureWritesNothing` | yes |
| M36 the rename made over an existing name, with no check | `TestMigrateBackupNameTaken` | yes |
| M22 the error from Clients dropped | `TestMigrateBadClientsFileIsAnError` | yes |
| hooks only: Migrate calling AcceptsLogin | `TestMigrateCallsOnlyItsHooks` | yes |
| hooks only: a call to provider.SetConsent | `TestMigrateCallsOnlyItsHooks` | yes |
| M25 a declined answer overwritten | `TestMigrateCarriesConsent` | yes |
| M26 the carry made whenever the file was read, the value ignored | `TestMigrateCarriesConsent` | yes |
| M27 the carry made whenever the setting is true | `TestMigrateCarriesConsent` | yes |
| M32 CarryConsent's error dropped | `TestMigrateCarriesConsent` | yes |
| M64 a backup made when only step 5 has work | `TestMigrateCarriesConsent` | yes |
| M33 CarryConsent called while auth.json.lock is held | `TestMigrateCarriesConsent` | yes |
| M62 the kind presence check dropped | `TestMigrateCarriesConsent` | yes |
| M67 the save made with no check that the record is still absent | `TestMigrateClientRaceKeepsFirst` | yes |
| M40 step 1 computing step 3's changes alone | `TestMigrateCompletesAfterInterruption` | yes |
| M42 the changes written as computed from the read before the lock | `TestMigrateConcurrentRunsWriteOnce` | yes |
| M4 the log line printing the key | `TestMigrateDropsDormantKey` | yes |
| M4 one record naming every provider | `TestMigrateDropsDormantKey` | yes |
| M5 removal only when both members are present | `TestMigrateDropsDormantKey` | yes |
| M49 step 3 limited to a fixed list of gobble's providers | `TestMigrateDropsDormantKey` | yes |
| M50 the dormant member found by e.Key != "" \|\| e.Workspace != "" | `TestMigrateDropsDormantKey` | yes |
| M60 Report naming nothing when there is no logger | `TestMigrateDropsDormantKey` | yes |
| M63 the path not resolved | `TestMigrateFollowsSymlink` | yes |
| M63 the lock taken beside the target | `TestMigrateFollowsSymlink` | yes |
| M11 the backup made on every run | `TestMigrateIsIdempotent` | yes |
| M8 step 3's removal dropping an unknown field of the edited entry | `TestMigrateKeepsEverythingElse` | yes |
| M7 step 3 applied to every entry, whatever its type | `TestMigrateKeepsEverythingElse` | yes |
| M9 step 3 written through File.encode as it stands | `TestMigrateKeepsEverythingElse` | yes |
| M61 the file's own CRLF kept at the top level | `TestMigrateLayout` | yes |
| M10 the edited entry written compact | `TestMigrateLayout` | yes |
| M1/M2 a state file created anyway | `TestMigrateNothingToDo` | yes |
| M2 a backup made for a file with no members | `TestMigrateNothingToDo` | yes |
| M6 a backup made whenever an oauth entry exists | `TestMigrateNothingToDo` | yes |
| M41 the lock taken before the first read | `TestMigrateNothingToDoTakesNoLock` | yes |
| M13 step 4 adopting from any openai oauth entry | `TestMigrateSkipsForeignEntry` | yes |
| M39 the state write's error dropped | `TestMigrateStateWriteFailureIsAnError` | yes |
| M51 the steps going on after the cancellation | `TestMigrateStopsWhenCancelled` | yes |
| M3 Load's error dropped, the migration going on as for an empty file | `TestMigrateUnreadableFileWritesNothing` | yes |
| M66 the parse's error returned without the sentinel | `TestMigrateUnreadableFileWritesNothing` | yes |
| M28 (false, false) taken as (false, true) | `TestMigrateUnreadableSettingsWritesNothing` | yes |
| M29 Settings asked on every run that has work | `TestMigrateUnreadableSettingsWritesNothing` | yes |
| M46 the call skipped for acp | `TestACPStartMigrates` | yes |
| M66 the unreadable-file error written to stderr as any other | `TestAuthConsentCommand`, `TestMigrateUnreadableAuthJSONIsTheCommands` | yes |
| M44 the call made in Main before the completion check | `TestCompletionModeDoesNotMigrate` | yes |
| M24 the closure passing false | `TestMigrateCarriesConsentEndToEnd` | yes |
| M48 run returning on Migrate's error | `TestMigrateErrorIsAWarning` | yes |
| M55 the call site's Warn record through the shared logger | `TestMigrateErrorIsAWarning` | yes, by the test named; not by `TestMigrateSilentUnderJSONAndACP` (expected, see the note below) |
| M45/M56 auth check left out of the exemption list | `TestMigrateRunsExceptStatusAndCheck` | yes |
| M54 the call gated on a tagged version | `TestMigrateRunsOnAnyBuild` | yes |
| M30 the closure reading the project's settings.yaml | `TestMigrateSettingsIsGlobalOnly` | yes |
| M53 the closure taking the string true as true | `TestMigrateSettingsIsGlobalOnly` | yes |
| M65 the closure taking the last value of a duplicated key | `TestMigrateSettingsIsGlobalOnly` | yes |
| M51 run going on to kctx.Run after a cancelled migration | `TestMigrateSignalSkipsTheCommand` | yes |
| M55 the error written with Output.Warnf whatever the command | `TestMigrateSilentUnderJSONAndACP` | yes |
| M55 the silencing not applied before Migrate under auth login --json | `TestMigrateSilentUnderJSONAndACP` | yes |
| M55 the 'no log file' line reaching stderr when the log has no stderr | `TestMigrateSilentUnderJSONAndACP` | yes |
| M52 the skip warning written under --json | `TestMigrateSkippedWithoutDirs` | yes |
| M12 the closure deciding the plan from the stored kind alone | `TestPlanSessionHook` | yes |
| M55 the view built over the fan-out handler | `TestFileOnlyView` | yes |
| M35 the backup written without `securePrivate` (Windows; a scratch clone with the plant, run on the Windows laptop through `run-on-host`, since plant.py runs on this Mac) | `TestMigrateBacksUpFirst` (`backup owner-only false`) | yes |
| M14 `host-id` written without `securePrivate` (Windows, the same way) | `TestMigrateAdoptsIDs` (`host-id owner-only false`), `TestHostIDIsStable` | yes |

Plant notes:

* **M43.** `migrateHooks.beforeHostIDCreate` fires inside the placement, after `placeHostID`'s first read finds no id and before it takes `host-id.lock`. That is "between the check and the create". It is why `TestMigrateHostIDRaceKeepsFirst` catches the settled plant, "the rename made without the read again under `host-id.lock`" (the table above). A hook run before the placement would let the placement's own first read see `HostID`'s file, and that plant could not fail the test.
* **M55's shared-logger plant.** It was built as the call site's Warn record given to the shared logger instead of the file-only view: `Migrate` itself writes only Info records, which never reach stderr. The call site writes that record to the file-only view in every command. Under `auth login --json` and `gobble acp` the shared logger has no stderr either, so `TestMigrateSilentUnderJSONAndACP` passes with the plant (exit 0, as expected and recorded before the run). `TestMigrateErrorIsAWarning` catches it, with two stderr lines where one is allowed.
* **Plants worded against a name `migrate.go` may not use.** M49's "limited to `provider.KeyProviders()`" was built as a fixed list of providers, since `migrate.go` imports no `llm/provider`. M30's "the closure reading the merged settings" was built as the closure reading the project's `settings.yaml`, since the typed merged loader ignores the key.

**Gates.** Each exit status was captured directly; long output went to scratch files.

| Command | Host | Exit |
| :--- | :--- | :--- |
| `make pre-add-check FILES="<the 11 Go files>"` | this Mac | 0, after the lint fixes below (first runs: 2) |
| `go test -race -count=1 ./internal/auth/ ./internal/cli/` (`run-heavy`) | this Mac | 0 |
| `make preflight` (`run-heavy`; live tree, the other agent's records present) | this Mac | 0, `preflight passed` |
| `make check-test-isolation` (`run-heavy`) | this Mac | 0, `files written above the temp roots: 0` |
| `go test -count=1 ./...` (`run-on-host --need linux --race`; `--race` only keeps the run off the WSL distros, so it reaches the Linux machine) | the Linux machine | 0 |
| `go test -race -count=1 ./internal/auth/ ./internal/cli/` (`run-on-host --need linux --race`) | the Linux machine | 0 |
| `go test -count=1 ./...` (`run-on-host --need windows`) | the Windows laptop | 0 (second run; see below) |
| `go test -count=1 ./...` (`run-on-host --need linux`, which chose a distro) | the Windows laptop's `Ubuntu-24.04` (WSL) | 0; extra, not the Linux machine's run |
| go-llmprovider-sdk `tools/denyscan.py --root <this repo> --changed` | this Mac | 0, `deny-scan: 0 hit(s) in 18 file(s)` |
| `make check-records markdownlint` | this Mac | 0 (before and after these records) |

**The Windows laptop's first suite run** (before the lint fixes) failed one test: `TestSignInTypeAheadNotSentAsPrompt`, `login_test.go:856`, `auth.json: open …: The process cannot access the file because it is being used by another process`. That test's harness builds a `runEnv` itself and never calls `run`, so the migration does not run in it. Its poll of `auth.json` raced the key's save by rename. With P5, the second suite run passed, and `go test -count=20 -run TestSignInTypeAheadNotSentAsPrompt ./internal/cli/` passed. The same 20 runs at `0589454`, on a scratch clone, also passed. It is recorded as a Windows timing flake in a path P5 does not touch; it is not fixed here.

**Lint fixes inside the phase's files** (`make pre-add-check`'s first two runs):

* `contextcheck`: `placeHostIDAfter` takes the migration's `ctx`. `placeHostID` keeps the settled `(dir, id string)` and waits for the lock with `context.Background()`, under a `//nolint:contextcheck` doc directive, because `HostID`'s callers give it no context.
* The closures' two `provider.PlanConsent` calls carry a line `//nolint:contextcheck`, because `PlanConsent` takes no context.
* `goconst`: `jsonFlag`.
* `gocritic` `typeDefFirst`: `migrateAuth` moved below `runEnv`.
* `gocognit`: `TestMigrateAdoptsIDs`' and `TestMigrateCarriesConsent`'s subtests became top-level helpers (`adoptIDs…`, `carries…`); the subtest names are unchanged.

**Deviations:** Deviation 5 above, the only one. The build pass count is 1: the stop on Deviation 5 came mid-pass, and the code built and went green on its first build.

**Not done by this phase:**

* `docs/README.md`'s 0052 rows are not edited (outside this phase's files); the text is in the hand-back.
* P6 and the owner's L8 are not started; L8 runs after P5's commit.
* Nothing is staged or committed.

### P6 kits (2026-11-22)

*(Dated addition, by the coordinator.)* Both kits are owner-run and live outside the repository, in the session's
scratch folder; they build gobble at `dfaa1ac` from `git archive` and never run it outside a fresh scratch
`GOBBLE_HOME` whose resolution they verify first (`home_is_scratch`).

* **L8 kit** (`l8.py`, instructions, `test_l8.py`): built and self-tested on synthetic files only, 24 passed, 0 failed
  (re-run by the coordinator, the same result); five plants each turned a check to its failure word. It prints the MADR
  L8 row's words only. The kit cannot exercise M67 (the `clients.json` race) from outside the process; P5's race test
  holds it.
* **L7's title model, the owner's answer** (2026-11-22, relayed by the coordinator): "You name it at run time
  (Recommended)". After L4's sign-in and before the L7 sessions, the kit writes the scratch home's `settings.yaml` as
  `defaultProvider: openai`, `smallModel: openai/<id>`, `defaultModel` empty. `<id>` comes only from a required
  `--small-model <id>` argument the owner supplies, shape-checked, never guessed and with no default; without it, or
  malformed, the kit stops before the build (`check_failed=small_model_required` or `small_model_shape`). Recorded
  because no row said which model L7's title uses, and that model spends the owner's plan quota.
* **L4-L7 kit** (`l4l7.py`, instructions, `test_l4l7.py` with a stand-in gobble): one command on this Mac
  (`--small-model <id>`), two groups, each in its own scratch home: L4 then L7 in one home (L7 needs L4's sign-in), and
  L5 then L6. Re-run switch per group; results to one file of fixed words. Self-test: 94 passed, 0 failed, 19 plants
  each turning its check to its failure word; a real build and a real ACP `initialize` found the `chatgpt_plan` method.
  Not exercised: a real sign-in (the owner's). The owner's standing direction of 2026-11-22 applies: everything around
  the owner's own sign-ins and keypresses is scripted.
* **Disclosure by the kit builder.** While checking an output format, the builder ran the built gobble's `gobble config
  path --json` once outside a scratch home. `config path` is not one of the commands the migration skips
  (`internal/cli/main.go:359-362`), so that run could have migrated the owner's own `auth.json` ahead of L8. The
  coordinator checked names and times only, no contents: no `auth.json.bak-*` exists (so no dormant key was dropped),
  and the state folder holds no `host-id` or `clients.json` (so no ids were adopted); `auth.json`'s last change shares
  its second with `auth.json.grok.refresh.lock`, the pattern of a Grok token refresh. The evidence is that the migration
  changed nothing there; the exact time of the builder's run is not known, so this is not proof. L8 stands as planned.

* **The kits copy for the owner** (2026-11-22, the owner, relayed by the coordinator: "for the file copies, those
  should be scripted to copy into whatever my cwd is. i should not have to copy files that can easily be scripted.").
  This replaces item 8's "*The owner:* copies their own `auth.json` ... into a fresh scratch home": when the owner runs
  the L8 kit, the kit itself resolves the owner's config and state directories in Python as `internal/appdirs` does at
  `dfaa1ac` (never by running gobble), copies `auth.json` (required) and `settings.yaml`, `host-id`, `clients.json`
  where present, read-only on the source with symlinked sources refused, and prints only `copied=<names>`; one prompt
  remains ("Run the check now? [y/N]"). The copy runs in the owner's own process; no agent reads the files. Both kits make
  their scratch home under the owner's current directory (`--home-parent` overrides) and refuse a parent inside the
  owner's gobble directories (`home_parent_unsafe`). The L4-L7 kit also refuses a parent whose enclosing repository has
  a project `.gobble/settings.yaml` (`project_settings_present`), since `session/new` reads the project file of its
  working directory (`internal/config/resolve.go:54-63`, `acpserver/agent.go:131-132`); L8 runs only `version` and
  `config path`, which read no project file. Self-tests, re-run by the coordinator: L8 94 passed, 0 failed; L4-L7 123
  passed, 0 failed; plants on the directory resolver and the project check each caught.

### L8 result (2026-11-22)

*(Dated addition, by the coordinator.)* The owner ran the L8 kit on this Mac against copies of their own files, made by
the kit itself, and relayed its output verbatim (the kit prints fixed words only):

```text
built: gobble 0.0.0-dev (source commit dfaa1ac)
copied=auth.json,settings.yaml
home_is_scratch=yes
run1_exit=0
keys_dropped=0
backup_made=no
host_id=none
client_record=none
consent_carried=none
run2_exit=0
run2_wrote=nothing
run2_backup=no
other_members_unchanged=yes
migration_warning=no
scratch_home_deleted=yes
```

Every word is the expected one and no `check_failed` line was printed: L8 passes. The owner's `auth.json` carried no
dormant `key`/`workspace`, no plan entry with ids to adopt and no consent to carry, so the migration is a no-op on it:
the first run wrote no backup and changed nothing, and the second run wrote nothing. The owner had no `host-id` or
`clients.json` to copy. This agrees with the coordinator's earlier check of names and times only, recorded under "P6
kits (2026-11-22)". The MADR's L8 row moves from awaiting live evidence to decided at P6's close-out, with L4 to L7.
