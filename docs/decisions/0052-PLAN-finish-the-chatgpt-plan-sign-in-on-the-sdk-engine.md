---
status: in-progress
date: 2026-11-11
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

1. `internal/auth/store.go`: `Entry.Kind`, `Entry.Consents`, written `kind` and `consents` when set; `SetKey`'s removal
   list (`:176-177` [WIP]) gains both, so a key leaves no stale session member (MADR D6); `logins.go` and
   `llm/provider/login.go` carry them both ways.
2. `internal/auth/hostid.go`: `HostID(dir string) (string, error)`, reading `<state>/host-id`, or creating it with
   `O_CREATE|O_EXCL`, owner-only (`securePrivate`, as `auth.json`), a random `urn:uuid`.
3. `internal/auth/clients.go`: `Clients{Path}` with `Load(ctx, id) (ClientRecord, error)` and `Save(ctx, id,
   ClientRecord) error`, under `clients.json.lock`, with `auth.json`'s write; a zero record removes the member.
4. `llm/provider/login.go`: `sdkStore` implements `auth.ClientStore` when its `LoginStore` also implements gobble's
   `ClientStore`; `SignInOptions.HostID` reaches `wizard.Options.HostID` and `signin.Options.HostID`.
5. Tests, each seen failing:

   | Test | Asserts | Planted break |
   | :--- | :--- | :--- |
   | `TestLoginCarriesEverySessionField` (`llm/provider`) | every exported stored field of `auth.OAuthSession`, set to a distinct value, survives `loginOf`, `Logins.Save`, `auth.json`, `Logins.Load` and `session`; the list of fields is read by reflection, so a new SDK field fails it | `consents` not written by `oauthFields` |
   | `TestHostIDIsStable` (`internal/auth`) | two calls give one id; two goroutines racing create one file; the file is owner-only (0600; the DACL test's helper on Windows) | `O_EXCL` dropped, so the second write wins |
   | `TestClientsRoundTrip` (`internal/auth`) | save, load, a zero record removes; a session's `Delete` in `auth.json` leaves `clients.json` | `Delete` removing the client |
   | `TestClientStoreAdapter` (`llm/provider`) | `sdkStore` is an `auth.ClientStore` exactly when its store has one, and passes records both ways | the type assertion inverted |
   | `TestSetKeyDropsSessionMembers` (`internal/auth`, beside `TestSetOAuthDropsKey`) | an `oauth` entry with `kind` and `consents`, then `SetKey`: neither remains in the `api_key` entry, every other field kept | `consents` left out of `SetKey`'s removal list |

6. Gates (C6).

Acceptance: the tests pass, each plant failed.

### P3 — consent: wording, gating, the mark and the command (MADR D2, D3)

Prerequisites: P2; 0051-PLAN P3's tag. **Staged with 0051-PLAN P4 and committed with it, as one commit** (that phase
says why): the plan is offered only in the commit that honours its consent.

First, `llm/provider/policy.go` (0051-PLAN P4): `honouredConsents` gains `openai.ConsentBackgroundUse`.

1. `llm/provider/consent.go`: `ConsentText`, gobble's wording for `openai.ConsentBackgroundUse` (the WIP's six lines,
   the fifth naming `gobble auth consent openai background deny`, and the question), passed to `wizard.Options` and
   `signin.Options`; `PlanConsent(id string, store LoginStore) (plan, granted bool)`, from the stored session's kind and
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

### P4 — the plan sign-in on every surface, and the status (MADR D1, D7, D8)

Prerequisites: P2, P3, 0051-PLAN P4 to P7. Between P3's commit and this phase, a first plan sign-in under `--json`
fails closed until 0051-PLAN P5 refuses it with exit 2 (F52's behaviour, nothing saved).

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
  `acpext/v0.5.0`; 0051-PLAN P4 with this plan's P3, one commit; 0051-PLAN P5 to P7; P4; P5; P6; gobble `v0.6.0`,
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
