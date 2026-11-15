---
status: in-progress
date: 2026-11-15
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

### P5 — the one-time migration (MADR D9)

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

### P6 — acceptance and close-out (MADR Confirmation)

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
* `go test -race -count=10 -run 'TestTitleUnderPlan|TestAutoCompactUnderPlan|TestConsentAppliesFromTheNextRequest|TestBackgroundRequestsAreMarked' ./acpserver/`
  *(2026-11-11: with `TestBackgroundRefusedWhenAnswerUnreadable` added to that run's pattern; P3's rows of that date.)*
* `go test -count=1 -run 'TestAuthConsentCommand|TestConsentWordingOnEverySurface|TestAuthStatusJSONPlan|TestForeignOpenAISessionRefused' ./internal/cli/`
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
