---
status: accepted
date: 2026-11-11
decision-makers: repository owner
consulted: "magic-cli-remote's docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md (observed 2026-11-10, committed in that repository as 3e2f784b), its §6, §8.5, §9.2, ledger G1-G9 and G13-G16, and the owner's answers of 2026-11-10 (Q2-Q6, Q11, Q12, Q22, and SDK-1 to SDK-3 on the SDK drafts); go-llmprovider-sdk's proposed 0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md and 0074-MADR-one-sign-in-handle-and-method-facts.md (untracked in that repository); this repository at d6c10a5 with the owner's uncommitted 0031-PLAN P4b-2 work, read and not changed; corrected on 2026-11-11 against this repository at 17d1605 and its branch wip/0031-p4b2-chatgpt-plan at 1406bd6, go-llmprovider-sdk's drafts as dated 2026-11-11, and the owner's rulings of 2026-11-11 as the cross-repository session relayed them; and, later on 2026-11-11, the owner's ruling that magic-cli-remote reads and can change the consent inside its 0246 cut-over, with acpext ErrorData's `consent` member, as the cross-repository coordinator relayed them"
informed: "magic-cli-remote, which relays the plan's confirm step and reads auth status --json, and (2026-11-11, later) changes a consent through gobble auth consent; go-llmprovider-sdk"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Finish the ChatGPT plan sign-in on the SDK engine: fold in the uncommitted work, keep consent on the session, give the host its own id, and reuse the issued client

## Context and Problem Statement

0031-MADR D7's amendment of 2026-11-10 (accepted) offered OpenAI's "Sign in with ChatGPT for agents" in gobble, and
0031-PLAN P4b-2 began building it. ~~That work is uncommitted~~
*(2026-11-11: it was when this was written. It is now one commit, `1406bd6`, on the branch
`wip/0031-p4b2-chatgpt-plan`, local and on origin, and `main`'s tree holds none of it; the PLAN's Deviation 1. The
figures are now `git diff --stat d6c10a5 1406bd6`'s)*: 39 files, +1010/−84 (`git diff --stat`). The 0245 report
(magic-cli-remote `docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md`) found five defects in
it and four more around it (F24, F25, F30, F31, F52-F56), and the owner decided on 2026-11-10 not to land it separately
(Q6): it is folded into the redesign, and its defects are fixed here, not as P4b-2 deviations. The SDK now proposes to
own the consent's requirement, question and storage, the host id's plumbing and the issued client's reuse
(go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md`). This record decides gobble's half
of the plan sign-in on that engine. Its surfaces (the policy, `--json`, ACP elicitation, the vocabulary) are
[0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)'s.

"[HEAD]" is `d6c10a5`; "[WIP]" is the owner's uncommitted work as it stood on 2026-11-10, read and not changed. A
line with no mark is the same at both. *(2026-11-11: that work was archived whole as commit `1406bd6` on the local
branch `wip/0031-p4b2-chatgpt-plan`, and cleared from `main`'s tree, by the PLAN's Deviation 1. A "[WIP]" line is now
read with `git show 1406bd6:<path>`. The `llm` half of 0049-PLAN P5b landed on `main` as `97f13a5`.)*
*(2026-11-11, later: the branch was pushed that day, and `git ls-remote --heads origin` lists
`wip/0031-p4b2-chatgpt-plan` at `1406bd6`. `main` is `17d1605`, six commits of 0053 past `86ad4a7`, and its tree holds
none of the work. A "[HEAD]" line is read with `git show d6c10a5:<path>`. Wherever this record says the work "is
uncommitted" or "sits in the working tree", it describes 2026-11-10.)*

### What the uncommitted work does [WIP]

* **Storage.** `auth.Entry` gains `Resource, Scope, IDToken, HostID` (`internal/auth/store.go:47`), written as
  `resource`, `scope`, `idToken`, `hostId` (`:114-117`), removed by `SetKey` (`:176-177`) and written by `oauthFields`
  when set (`:263`); `Logins` carries them (`internal/auth/logins.go:41`, `:54`); `provider.Login` and its two copies
  too (`llm/provider/login.go:26`, `:131`, `:145`), its `String` hiding the ID token (`:33-42`). `Login.Plan()` is
  `Resource != ""` (`:30`).
* **The sign-in.** `MethodChatGPTPlan = "chatgpt-plan"` and `AgentName = "gobble"` (`llm/provider/signin.go:22`, `:27`,
  `:34`, `:201`); `HasLogin` gains openai (`login.go:73-75`); `offers` gains openai guards (`signin.go:86-100`);
  `loginWith` sends a bare `auth login openai` to the plan (`internal/cli/auth.go:209-214`).
* **The notice and the consent.** `planNoticeLines`, six lines (`internal/cli/login.go:101-110`), shown by `planNotice`
  (`:113-124`) before the sign-in in `loginWith` only (`auth.go:226-235`); `askPlanBackground` (`login.go:129-145`)
  asks after a saved sign-in and writes `chatgptPlanBackground: true` on a yes, nothing on a no (`:138-144`). The setting
  is new (`internal/config/keys.go:104-105`; `settings.go:24-27`, `:135`; `settings.yaml:143-152`), reaches a session at
  its start (`internal/cli/settings.go:104`; `acpserver/agent.go:496`, `:1096`), and gates the title and automatic
  compaction under a plan session (`acpserver/title.go:51`, `:61-78`; `autocompact.go:70-75`).
* **The resolver.** A non-plan openai login is refused at provider build (`llm/provider/provider.go:107-112`) but
  `auth status --json` gives `usable: true` with `reason: "oauth_unused"` (`internal/cli/auth.go:447`, `:465-467`).
* **Not auth, in the same tree.** 0049-PLAN P5b's mapping: `llm.APIError.Window` (`llm/errors.go:65`), `limitOf` and
  `planWindow` (`llm/provider/provider.go:768-809`), `ErrorData.WindowMinutes` (`acpserver/prompt.go:308`), their tests
  (`llm/provider/limit_test.go`, `acpserver/errordata_test.go`, `acpserver/choice_test.go`), and the pin to
  `acpext/v0.4.0` (`go.mod`). 0031-MADR P4b-Q3 (b) put them in "one root phase and one commit" with P4b-2
  (`0031-MADR-sdk-current-releases-and-adoption.md:357-359`).
* **The records.** 0031-MADR gains "The terms, read 2026-11-10" (`:363-404`): "own host only, disclosed"; "a notice at
  sign-in, before the browser opens"; "a setting, named in the notice", turned off "from the next request" (`:389-391`).
  0031-PLAN gains a P4b-2 record and its deviation 1 (`:350-393`).

### What is wrong with it (the report's findings, re-read here)

* **F30** the notice and the consent are on `auth login`'s path only: `/login`, `gobble setup` and the first run reach
  the sign-in through `signIn` → `signInTo` (`internal/cli/login.go:62`), with neither.
* **F31** the consent is asked at every plan sign-in; a "no" writes nothing, so an earlier "yes" survives
  (`login.go:138-144` [WIP]); P4b-Q2 (a) said "asked once" (`0031-MADR-…md:355` [WIP]); the setting is read when a
  session starts (`acpserver/agent.go:496` [WIP]), not "from the next request".
* **F52** a first plan sign-in under `--json` or with no terminal fails at the SDK's own `Confirm`, since `unasked`
  answers every question `errNoTerminal` (`internal/cli/auth.go:222` [WIP]).
* **F53** `gobble acp --login openai` lost its key path: `acpLogin` takes the sign-in when `HasLogin`
  (`internal/cli/acp.go:101-103`), against 0031-PLAN P4b-2 step 4, "offers the plan and the API key"
  (`0031-PLAN-…md:328`; *2026-11-11: `:347` on `main` since `86ad4a7`*).
* **F54** the resolver is plan-blind: `Resolver.Has` counts any `oauth` entry of a `HasLogin` provider
  (`internal/auth/resolve.go:103-104`), so status, `auth check`, `authenticate` and provider choice accept what the
  provider build refuses, and `Logout` revokes any such entry (`llm/provider/login.go:117` [WIP]): another app's token.
  0045-MADR D2 says `reason` "is there only when `usable` is false" (`0045-MADR-auth-status-json.md:116`).
* **F55** stale guidance: `docs/guides/credentials.md:208` ("Signing in to ChatGPT comes later"); the `AuthCmd` help,
  "or sign in to Grok or Kilo" (`internal/cli/auth.go:27`); 0005-MADR's OpenAI row, "API key or that backend's OAuth"
  (`0005-MADR-v1-feature-scope.md:168`).
* **F56** host behaviour: `--json` still opens the host's browser (`internal/cli/auth.go:257` [HEAD]); `/login` never
  does; text device codes omit their expiry.
* **F25** [HEAD] `SetOAuth` keeps a dormant `key` and `workspace`: `oauthFields` removes only unset session fields and
  `fedramp` (`internal/auth/store.go:242-268` [HEAD], `:251-277` [WIP]).
* **F24, F22, F23** the host id is the sign-in's, minted when there is no `Previous` (the SDK's `planHostID`), not the
  host's, which 0025-MADR D6 decided ("a stable `HostID` in the state directory",
  `0025-MADR-provider-choice-and-auth.md:224`); `SetKey` and `Delete` drop `clientId` and `hostId`
  (`internal/auth/store.go:176-177` [WIP], `:333-344` [WIP]), so the next plan sign-in registers another client.

### What the SDK proposes (go-llmprovider-sdk, `proposed`, 2026-11-10)

Its `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md`:

* D1: `AuthMethod.Consent`, `ConsentText`, `ConsentURL`; `openai.ConsentBackgroundUse = "chatgpt_plan.background_use"`.
* D2, D3: `OAuthSession.Consents` (`<id>=granted|declined`, space-separated), `(*OAuthSession).Consent(id) (granted,
  answered bool)`, `auth.SetConsent(ctx, store, provider, id, granted)` under the refresh lock; a refresh never loses
  an answer.
* D4: the plan sign-in starts with a `confirm` step when there is no plan session to reuse or it has no answer;
  `signin.Options.ConsentText` and `wizard.Options.ConsentText` give the app's wording; the answer is recorded only
  if the sign-in saves; the wizard renders it as a three-way `Select`.
* D5: `llmprovider.WithBackground(ctx)` and `IsBackground`; the openai provider refuses a marked request on a plan
  session without a granted consent, `ErrNotPermitted`, before the network.
* D6: `wizard.Options.HostID` and `signin.Options.HostID`, "gobble persists one in its state directory".
* D7: `auth.ClientRecord{ClientID, HostID}` and the optional `auth.ClientStore` interface (`LoadClient`, `SaveClient`);
  `ChatGPTPlanOptions.ClientID`; `signin.Options.ClientStore` for `x/signin`, while the wizard uses its
  `Options.TokenStore` when that store implements `ClientStore` (gobble passes `sdkStore{store}` there,
  `llm/provider/signin.go:193`; *2026-11-11: `:193` is the archive branch's line, `:181` on `main`*).

*(2026-11-11.)* Re-read as that record is dated 2026-11-11, with the owner's rulings of the day, which the SDK's
records are being revised to carry. D1 to D7 above are as written. Two names are added, and one is dropped:

* on its D5, `(*OAuthSession).StoredConsent(ctx, id) (granted, answered bool, err error)`: the openai provider reads
  the consent through it, and a read that fails is refused as not granted (D2's note of this date);
* on go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D5, the exported
  `(*OAuthSession).SessionKind()` (D8's note of this date);
* `wizard.Options.OAuthClients` is dropped from go-llmprovider-sdk `0076-MADR-auth-error-kinds-and-doc-drift.md` D4;
  this record never used it.

The owner answered the SDK drafts' questions on 2026-11-10: **SDK-1**, a "no" at the plan's confirm declines
background use only, and a third choice declines the sign-in; **SDK-2**, the release is gated on the owner-run probe
of client reuse without `id_token_hint`, the fallback keeping only the host id; **SDK-3**, `wizard.Options.Driver` may
name the v0 contract's types in the v1 `wizard` API (0051-MADR D3 is its caller, QE (a)).

## Decision Drivers

* **The owner's answers of 2026-11-10:**
  * **Q6**, "Fold into the redesign. The WIP is not landed separately. Its five defects … and G1-G9 are fixed
    inside the redesign's gobble pair(s)".
  * **Q2**, option C: "gobble holds the wording, the background classification, the gating and the setting UI".
  * **Q3**, "On the session: `OAuthSession` consents persisted by the token store, changed through an SDK `SetConsent`
    helper, and cleared with the session at sign-out".
  * **Q4**, "Before the browser opens, with the notice, as one `confirm` step. It is recorded only if the sign-in then
    saves."
  * **Q5**, the opt-in enforcement helper.
  * **Q11**, the plan from the phone "allowed, disclosed"; a pasted callback is a secret.
  * **Q12**, "One slot per provider, keep ids … a sign-out or switch never forces a new client registration".
  * **Q22**, "Drop them at migration, keeping an owner-only backup and logging one line per provider".
  * **SDK-1, SDK-2** above.
* **The terms as the owner read them** (0031-MADR's WIP amendment): own host only, disclosed; the notice before the
  browser; consent controllable, and withdrawn "from the next request".
* **gobble's records:** `auth.json` stays gobble's file in Pi's shape (0005-MADR amendment of 2026-10-10 item 3);
  `auth status --json` grows only by addition (0045-MADR D6); the SDK only in `llm/provider` (0004-MADR rule 4, as
  0051-MADR D1 amends it); a test seen failing before it is trusted (0004-MADR's archtest paragraph; 0049-PLAN's C2).
* **The owner's credentials are real data:** a one-time migration that backs up first, is idempotent, and writes
  nothing it cannot explain (the report's §9.2).

## Considered Options

* **A. Fold the work onto the SDK engine** — keep its storage and gating structure, rework its predicates, notice,
  resolver and setting onto 0051's policy and the SDK's consent, host id and client store, discard what the SDK now
  does, and fix every defect here.
* **B. Fix, then land the work as P4b-2, then migrate it** — the report's recommendation for Q6.
* **C. Discard the work and start again on the engine.**

## Decision Outcome

Chosen option: "A. Fold the work onto the SDK engine", because the owner chose it (Q6), and because the work's storage
half is what any design needs, while its CLI half is what 0051's policy and the SDK's consent replace. ~~This record is
`proposed`, awaiting the owner's review of this draft.~~

*(2026-11-11: accepted.)* On 2026-11-11 the owner typed, in this repository's own session, "accept and approve 0051 and
0052 when required", and confirmed it to the cross-repository coordinator the same day: accept 0051-MADR and
0052-MADR, and start their phases that do not wait on the SDK. This record is accepted as written, with its notes of 2026-11-11.
The questions it put were answered by the owner on 2026-11-10, each as recommended, and are kept under "Questions
answered by the owner (2026-11-10)": QA (a), QF (a) and QG (a). No question of this record is open. The records under
"Records amended or superseded in part" are amended by
[0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)'s P0, and its "P0 record
(2026-11-11)" says where.

### D1. What happens to each part of the uncommitted work

| Part [WIP] | Fate | Why |
| :--- | :--- | :--- |
| The four plan fields in `Entry`, their JSON names, `SetKey`'s removal list, `oauthFields` (`internal/auth/store.go:47`, `:114-117`, `:176-177`, `:263`); `Logins` (`logins.go:41`, `:54`); `provider.Login` and its copies (`llm/provider/login.go:26`, `:131`, `:145`) | **kept as is** | the SDK's session holds them (`tokenstore.go`'s record), and gobble's store must carry them |
| `TestStoreOAuthPlanFields`; the plan half of `TestLoginStoreAdapter`; `TestLogoutPlan`; `planIssuer`, `planCallback`, `TestSignInChatGPTPlan`, `TestSignInChatGPTPlanPasted` | **kept**, the last two **reworked** for the confirm step | they test what stays |
| `MethodChatGPTPlan`, its `methodIDs` entry, `AgentName` (`signin.go:22`, `:27`, `:34`, `:201`) | **kept** | the CLI word and the app's name |
| 0049-PLAN P5b's hunks (`llm/errors.go`, `provider.go`'s `limitOf` and `planWindow`, `prompt.go:308`, their tests) | **kept as is**, landed by QA's answer | not auth |
| `HasLogin` gaining openai, `offers`' guards, the `provider.go:107-112` refusal, `Login.Plan()`, `loginWith`'s openai branch, the status row's openai case, `PlanSession`'s `id == "openai"` (`internal/cli/agent.go:101`), `signInWords`' openai case | **reworked** onto 0051-MADR D2's policy and the SDK's session kind | per-id checks (F41) |
| `planNoticeLines`' wording | **reworked** into gobble's `ConsentText` (D2) | the SDK's `confirm` carries it to every surface |
| `planGated`, `errPlanOverflow` (`acpserver/title.go:61-78`, `autocompact.go:70-75`) | **reworked**: the gate reads the session's answer per request (D2) | F31 |
| `planNotice`, `askPlanBackground`, the `notice` `--json` event (`internal/cli/login.go:96-145`) | **discarded** | the SDK's `confirm` step replaces them (Q4) |
| The `chatgptPlanBackground` setting: `keys.go:104-105`, `settings.go:24-27`, `:135`, `settings.yaml:143-152`, `testdata/settings.schema.json`, the two config test edits, `internal/cli/settings.go:104`, `acpserver/config.go:101`, `liveSession.planBackground` (`acpserver/agent.go:286`, `:496`, `:1096`) | **discarded** (D3) | consent lives on the session (Q3) |
| `HasPiLogin` (`llm/provider/login.go:81`) and `piimport.go:45` | **discarded** | 0051-MADR D2's Pi rule |
| `TestAuthLoginChatGPTPlan`, `TestAuthStatusOpenAINotAPlan`, `TestACPLoginOpenAI`, `TestPlanSessionOption`, `TestOpenAILoginIsAPlan`, `TestTitleUnderPlan`, `TestAutoCompactUnderPlan`, the edits to `TestAgentAuthOptions`, `TestProviderChoicesStates`, `TestProvidersJSON` and `TestAuthLogin` | **reworked** to this record's rules | they encode the WIP's |
| The 0031 MADR and PLAN text | **kept**, amended by the owner from the text this record's author wrote to the scratch file named in the PLAN's P0 | the terms' reading is fact |

### D2. Consent: gobble's wording, classification and gating on the SDK's step and storage

* **Wording.** gobble passes `ConsentText{openai.ConsentBackgroundUse: …}` to `wizard.Options` and `signin.Options`.
  The text is `planNoticeLines` [WIP] with its fifth line changed to name D3's command, then the question "Allow gobble
  to make background requests under your ChatGPT plan: naming sessions and compacting them on its own?". It is shown
  as the `confirm` step before the browser (Q4), on every surface (0051-MADR D3): the wizard's three choices on a
  terminal (SDK-1), a boolean form over ACP (0051-MADR D4), a refusal under `--json` (0051-MADR D5).
* **Asked once.** The SDK shows the step only when no plan session exists or it holds no answer (go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D4).
  A "no" records `declined` (F31).
* **Classification.** Background requests are gobble's own: a session's title from `smallModel`
  (`acpserver/title.go`) and automatic compaction, at the threshold or on an overflow (`acpserver/autocompact.go`).
  The user's turns and `/compact` are not.
* **Gating, gobble's.** `acpserver.Options.PlanConsent func(id string) (plan, granted bool)` replaces `PlanSession`
  [WIP] and the per-session setting; `internal/cli` answers it from the stored session (`provider.PlanConsent`, reading
  `auth.json` through `Logins` and the SDK's `Consent`). `planGated` asks it at every background request, so a
  withdrawal applies from the next request in every process (0031-MADR's "from the next request" made true). Without a
  grant: no title; no automatic compaction; an overflow ends the turn with `errPlanOverflow`, reworded to name `/compact`
  and D3's command.
  *(2026-11-15, the owner's ruling "Keep it, and document it", relayed by the coordinator.)* The gate stays as decided, keyed on the stored session, not on the
  credential a run uses. Consequence: with `--api-key` for openai and a plan session stored, titles and automatic compactions are held back until the plan
  consent is granted, although those requests would go out with the key. Checked against the code, for each way a key reaches a run: only `--api-key` does this.
  `--api-key` sets the run's key (`internal/cli/chat.go:69`, `setRuntimeKey`), and the resolver takes it first (`internal/auth/resolve.go:77`; `Has`, `:109`), so the
  provider is built with the key (`llm/provider/provider.go:99`, `Options.login` returns none for a key); the gate is built from the stored session alone
  (`internal/cli/credentials.go:22`, `provider.PlanConsent`, wired at `internal/cli/agent.go:103`). An environment variable or `models.json`'s `apiKey` is reached only when
  `auth.json` has no entry for the provider (`resolve.go:80-82` returns the entry's verdict before `:83` and `:94`, and `Has` the same at `:112`, `:115`, `:123`), so with a plan
  session stored they are not used, the requests go out under the plan, and the gate governs them as it should. `settings.yaml` has no key that carries a credential
  (`internal/config/keys.go`). `docs/guides/credentials.md` says it where the consent is explained.
* **Enforcement, the SDK's, as well.** `llm` gains `WithBackground(ctx)` and `IsBackground(ctx)` (standard library, rule 1
  holds); the title and the automatic compaction mark their context; `llm/provider`'s adapter turns the mark into
  `llmprovider.WithBackground`, so a request that gobble's gate missed is refused before the network
  (go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D5).
* *(2026-11-11, added.)* **An answer that cannot be read is not a grant.** This follows the owner's ruling of that
  day on go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D5.
  * **The SDK.** 0075 gains `(*OAuthSession).StoredConsent(ctx, id) (granted, answered bool, err error)`. The openai
    provider reads the consent through it, and a read that fails is refused as not granted: `ErrNotPermitted`, before
    the network. For gobble the session's store is `sdkStore` over `auth.json`, so that read is of `auth.json`.
  * **gobble's gate follows the same rule.** `PlanConsent` returns an error as well: `func(id string) (plan, granted
    bool, err error)`. With an error the background request is not made, exactly as without a grant. gobble never
    reads a failed read as granted, and writes nothing because of one.
  * **What the user is shown.** This part is proposed with the record: the owner's ruling covers the refusal, not
    gobble's words. A title, or a compaction at the threshold, that is not made shows nothing, as the declined case
    does, and the run's log gets one line naming the provider and the store's error, never a token. An overflow has to
    end the turn, and its error is not `errPlanOverflow`. It says that the stored answer for background use under the
    ChatGPT plan could not be read, gives the store's error, and names `/compact`, which is the user's own request and
    is not gated. It does not name `gobble auth consent openai background allow`: the answer may already be `granted`,
    and that command would meet the same store.
  * **A marked request that gobble's gate missed,** and the SDK refuses, comes back as `llm.ErrNotPermitted` whichever
    reason the SDK had, a decline or a failed read. gobble shows the SDK's message, which names the consent id.
* *(2026-11-11, later; the owner's ruling of that day that magic-cli-remote reads the consent and can change it, inside
  its `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`.)* **A
  refused background request carries the consent's id,** in acpext's `ErrorData.Consent`, JSON `consent`
  ([0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)
  D7's note of this date), so a client can offer to allow it.
  * **What a client sees refused.** A title, or a compaction at the threshold, that is not made is no request of the
    client's, and stays silent, as above. What reaches a client is a turn that ends, as `session/prompt`'s error
    (`promptError`, `acpserver/prompt.go:272-278`, an internal error, -32603), and its `ErrorData` is one of two:

    | The turn ends because | `kind` | `consent` | `error` says | A client offers |
    | :--- | :--- | :--- | :--- | :--- |
    | the gate refused an overflow's compaction for want of a grant (`errPlanOverflow`), or the SDK refused one the gate let through and gobble's own read then finds no grant | `not_permitted` | `chatgpt_plan.background_use` | `errPlanOverflow`'s text: it names `/compact` and the command | to allow it |
    | the stored answer could not be read (the note above), whether gobble's read failed or the SDK refused and gobble's read then fails | `not_permitted` | absent | that the stored answer could not be read, with the store's error, naming `/compact` and not the command (the note above) | no allow: the error is shown |

    `provider` is `openai` in both. The rule is that `consent` is set only when allowing that consent would let the
    request through; an answer that cannot be read is not one, since an allow would meet the same store
    (`gobble auth consent` exits 1, D3's note). `kind` stays `not_permitted` for the second row, as the SDK refuses
    it (`ErrNotPermitted`, its D5 ruling), so it is never mistaken for a quota or a provider failure.
  * **How the id is carried.** `llm` gains `ConsentError{Consent string; Err error}`, standard library only (rule 1
    holds). Its `Unwrap` gives `Err`, which matches `llm.ErrNotPermitted`, so `llm.Kind` gives `not_permitted` with no
    change. `errorData` (`acpserver/prompt.go:286`) sets `ErrorData.Consent` from it. `errPlanOverflow` is a
    `ConsentError`; the unreadable-answer error is not, and wraps `llm.ErrNotPermitted` alone. The SDK's refusal of a
    marked request names the id only in its message, and does not say whether it read a decline or failed to read
    (go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D5), so `acpserver` decides the
    row from its own `PlanConsent` read made when that refusal comes back: an error gives the second row, otherwise
    the first. The id is never a constant of gobble's: it is the
    offered method's `Consent` fact (that record's D1), and `acpserver` learns it from `PlanConsent`, which now
    returns it: `func(id string) (ConsentState, error)`, `ConsentState{Consent string; Plan, Granted bool}`, in place of
    the earlier note's `(plan, granted bool, err error)`, with the same rule that an error is no grant.
  * **How the unreadable answer surfaces to magic-cli-remote.** By the table's second row: kind `not_permitted` with
    no `consent`, so Magic, which maps to an allow prompt only an error that carries `consent`, shows the error's
    text. `auth status --json` then gives 0045-MADR D5's error object and exit 2 (D8's note), and `gobble auth
    consent` exit 1 (D3's note), so neither offers a change that could not be stored. The behaviour of the earlier
    note is unchanged, and stays proposed with it.
  * **The text and link shown before an allow are gobble's.** `provider.Offered` (0051-MADR D2) sets each honoured
    consent's `Text` to gobble's wording, the `ConsentText` this record passes to the sign-ins, and keeps the SDK's
    `Links`, the usage page (`openai.ChatGPTUsageURL`, go-llmprovider-sdk
    `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D1). So the sign-in's `confirm` step (its `text` and
    `links`), each agent auth method's `_meta.gobble.methods[].consents[]` (0051-MADR D4), and D8's new
    `consentTerms` give one text and one link. A client that offers an allow shows them first: from the `confirm` step
    during a sign-in, and from `consentTerms` at any other time, which needs no ACP connection.

### D3. The setting is retired, and a command changes the answer

* `chatgptPlanBackground` is not landed. It exists only in the uncommitted work (`internal/config/keys.go:104` [WIP];
  absent at [HEAD]), so no release carries it.
* **`gobble auth consent <provider> [<consent> allow|deny]`** lists a provider's consents and their answers, or changes
  one through `provider.SetConsent(ctx, store LoginStore, id, consent string, granted bool) error` and
  `provider.Consents(ctx, store LoginStore, id string)`, two functions in `llm/provider` over the SDK's
  `auth.SetConsent` and `(*OAuthSession).Consent`. They live there because `llm/provider` is the one package that may
  import the SDK beyond `authcontract` (0004-MADR rule 4, as 0051-MADR D1 amends it; archtest's rule 4,
  `internal/archtest/check.go:308`), and `sdkStore`, the SDK's view of gobble's store, is unexported there; the command
  in `internal/cli` calls them. gobble's word for `chatgpt_plan.background_use` is `background`; the id stays the
  SDK's. A provider with no stored session is a usage error, exit 2. `auth status` shows the answer (D8).
  *(2026-11-11: a store that cannot be read or locked is not a usage error. `gobble auth consent` then exits 1 with
  the store's error and prints no answer, neither "not asked" nor an earlier one; D2's note of this date.)*
  *(2026-11-11, later; the owner's ruling that magic-cli-remote's daemon changes a consent by running this command
  non-interactively, D2's note.)* **What a non-interactive caller can rely on:**
  * **No question and no terminal.** The command never prompts, reads nothing from stdin, and behaves the same with no
    terminal attached.
  * **Arguments.** `<provider>` is a gobble provider id (`openai`). **`<consent>` is the SDK's consent id**
    (`chatgpt_plan.background_use`), the id `auth status --json`'s `consents` and `consentTerms` and
    `ErrorData.Consent` carry, so a client passes back the id it read and maps no word of gobble's. gobble's word,
    `background`, is kept as an alias of that id for a person at a terminal; the earlier sentence "gobble's word for
    `chatgpt_plan.background_use` is `background`; the id stays the SDK's" now reads with this: both are accepted,
    and the id is the one every machine-readable output names. The third argument is `allow` or `deny`; with no second
    and third, the consents are listed.
  * **Exit codes** (`internal/cli/exit.go`): 0, the answer is stored, or the list printed; 2, a usage error: an
    unknown provider, a consent its stored session's method does not have, a provider with no stored session, a
    second argument with no third, or a third other than `allow` or `deny`; 1, the store cannot be read, parsed or
    locked (the note above), nothing written.
  * **Output.** On 0 after `allow` or `deny`, one line on stdout naming the provider, the consent's id and the stored
    answer, `granted` or `declined`; a list prints one line per consent of the method, its answer `granted`,
    `declined` or `not asked`. Every error is one line on stderr, never a token, and stdout is then empty.
  * **No `--json`.** A client takes the result from the exit status and re-reads `auth status --json` (D8), the one
    machine-readable view of the answers; a second JSON shape of the same fact would have to be kept equal to it. This
    is this record's proposal, not part of the owner's ruling.
* **Why not a view** (the report's §6.4 (c)): two names for one fact, kept equal by code; and a key whose value lives in
  `auth.json` breaks the config model, where every key is a `settings.yaml` value with a layer and a scope
  (`internal/config/keys.go`), and `config set`, `unset` and `validate` act on the file.

### D4. A host id for the host's life (0025-MADR D6 built)

* `<state>/host-id` (`internal/appdirs`' `State`), one line, a random `urn:uuid` in the SDK's form, created once with
  an exclusive create, owner-only (0600; the protected DACL on Windows, as `auth.json`). A race between two processes
  keeps the first; the second reads it.
* Passed as `wizard.Options.HostID` and `signin.Options.HostID` on every plan sign-in. The state directory, not the
  config directory, because the notice tells the user not to copy `auth.json` to another machine, and a config
  directory is the one a user syncs.
* *(2026-11-21, the owner's answer to the state table's Q3, "Validate; lock-and-rename host-id (Recommended)".)* "Created
  once with an exclusive create" now reads: written to a temporary file, owner-only, and renamed into place under a lock
  file, `<state>/host-id.lock`, by the process that holds it and finds the name still absent or empty, by `HostID` and the
  migration alike. An empty `host-id`, left by a create that was interrupted, counts as absent and is replaced. "A race
  between two processes keeps the first; the second reads it" still holds: the second finds the first's id under the lock.
  A rename needs no hard link, so volumes without them keep working. "Owner-only (0600; the protected DACL on Windows)" is
  unchanged.

### D5. The issued client is kept apart from the session (Q12)

* `internal/auth.Clients{Path}` stores `{"openai": {"clientId": …, "hostId": …}}` in `<state>/clients.json`, with
  `auth.json`'s lock, private write and fsync. `sdkStore` implements the SDK's `auth.ClientStore` over it, so the wizard
  and `signin.Start` reuse the issued client after a logout or a switch to a key. The two find it differently
  (go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D7): the wizard asserts its
  `Options.TokenStore`, which is `sdkStore`; `x/signin` takes it only as `signin.Options.ClientStore`, nil meaning
  none, so `provider.StartSignIn`, the ACP path the phone uses, sets that option to the same adapter, with `HostID`
  and `Previous` (0051-MADR D3).
* `auth logout openai` deletes the session and leaves the record (Q12); a sign-out never forces a new registration.
* **Gated by SDK-2.** If the owner's probe shows OpenAI refusing an issued client without `id_token_hint`, the SDK stops
  passing `ClientID` and gobble's record keeps the host id only; nothing of gobble's changes.

### D6. `auth.json`'s `oauth` entry gains `kind` and `consents`

`Entry`, `Login` and both `sdkStore` directions carry the SDK's `OAuthSession.Kind` and `Consents`, written `kind` and
`consents` when set. Both are session members, so `SetKey`'s removal list (`internal/auth/store.go:176-177` [WIP])
gains them: a switch to a key leaves no stale kind or consent in the `api_key` entry, and the consent is cleared with
the session, as Q3 says. A field-copy test sets every exported, stored field of `auth.OAuthSession` and fails on any that a
round trip through `Login` and `auth.json` loses, so the next SDK field is not dropped silently (go-llmprovider-sdk
`0074-MADR-one-sign-in-handle-and-method-facts.md` D5's guard, on gobble's side).

### D7. The defects, each fixed here

| Finding | Fix |
| :--- | :--- |
| F30 | the `confirm` step reaches every surface (0051-MADR D3), and no gobble code shows a notice of its own |
| F31 | asked once, a "no" recorded, read per request (D2) |
| F52 | `--json` refuses a sign-in that would ask, before it starts, exit 2 (0051-MADR D5) |
| F53 | `gobble acp --login` shows the wizard's menu, the key included (0051-MADR D3) |
| F54 | the resolver accepts a session only of an offered kind (0051-MADR D2); the status obeys 0045-MADR D2 (D8); a foreign session is deleted, never revoked |
| F55 | `docs/guides/credentials.md`'s ChatGPT section, the `AuthCmd` help, 0005-MADR's OpenAI row, `README.md`, `docs/architecture.md` |
| F56 | no host browser under `--json` or ACP; `/login` opens it as `auth login` does; text device codes name their expiry (0051-MADR D3, D5) |
| F25 | `SetOAuth` removes `key` and `workspace` (`oauthFields`); D9 cleans what is already stored |

### D8. `auth status --json` gains `sessionKind` and `consents` (G15; additive under 0045-MADR D6)

* `sessionKind`: the stored session's kind, the SDK's `auth.SessionKind` value (`chatgpt_plan`, `grok`, …), when
  `authType` is `oauth`.
  *(2026-11-11: settled by the owner's ruling that the SDK exports `(*OAuthSession).SessionKind() SessionKind`
  (go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D5). `sessionKind` is that method's value for the stored
  session: its `Kind` when set, else the kind the SDK infers, so a session saved before `kind` existed still reports
  one. The values are `chatgpt`, `chatgpt_plan`, `grok` and `kilo_device`. gobble copies no inference rule, here or in
  0051-MADR D2.)*
  *(2026-11-11, later: the owner ruled that the auth contract carries a session-kind type with these values and their
  labels, added to go-llmprovider-sdk `0073-MADR-publish-an-auth-contract-module.md` before `authcontract/v0.1.0`.
  The member's values are unchanged; they are that type's values, and a client such as magic-cli-remote labels them
  from the contract. gobble's text form may take its words from the same labels; this record does not require it.)*
* `consents`: an object from consent id to `granted` or `declined`, only for answered ones, e.g.
  `{"chatgpt_plan.background_use": "declined"}`.
* An `oauth` entry the resolver does not accept is `usable: false` with `reason: "oauth_unused"`, as 0045-MADR D2 says.
* The text form says "background use allowed", "declined" or "not asked" for a plan session.
* *(2026-11-11, later; the owner's ruling that magic-cli-remote's daemon reads these members and shows them on the
  phone.)* The PLAN's rollout said that Magic "reads `sessionKind` and `consents`" (its note of this date corrects
  it). Magic's daemon does more: it reads them, shows them on the phone, changes an answer through `gobble auth consent` (D3's note), and maps a refused background
  request to a prompt offering to allow it (D2's note). For that caller the members, all on a row of `providers`
  beside 0045-MADR D2's `provider`, `source`, `variable`, `authType`, `usable`, `reason` and `expires`, are:
  * **`sessionKind`:** only when `authType` is `oauth`; one of `chatgpt`, `chatgpt_plan`, `grok` and `kilo_device` (the
    note above). The SDK may add kinds, so a value a client does not know is to be shown, not refused.
  * **`consents`:** only when the session holds an answer; an object from consent id to `granted` or `declined`. A
    consent with no member is "not asked".
  * **`consentTerms`, new:** only when the stored session is of the kind an offered method with consents signs in to,
    which among the built-ins is the plan session alone; an array of the contract's `Consent`
    (`authcontract.Consent`, go-llmprovider-sdk `0073-MADR-publish-an-auth-contract-module.md` D2): `id`, `required`,
    `text`, gobble's wording (D2's note of this date), and `links`, `[{rel, url}]`, the usage page. It lists every
    consent of that method, answered or not. Without it a client that reads only the CLI would have the text and link
    only from a sign-in's `confirm` step, which a later change of answer does not have. An addition under 0045-MADR
    D6.
  * **A store that cannot be read** gives 0045-MADR D5's answer: `{"error": {"file": "auth.json", "message": …}}` on
    stdout, no `providers`, exit 2. A client takes it as "the answers cannot be known", shows the message, and offers
    no change, since `gobble auth consent` meets the same store and exits 1 (D3's note). The two exits differ on
    purpose: 2 is `auth status --json`'s one error exit (0045-MADR Q2, answered 2026-11-07); 1 is the failure of a
    command asked to act.
  * **`--json` asks nothing and writes nothing to stderr** (0045-MADR D4; `internal/cli/auth.go:352-358`).

### D9. The one-time `auth.json` migration (G16, Q22)

Run by the first `v0.6.0` process that resolves credentials, before it reads `auth.json`; idempotent by content, so each
later run finds nothing to do and writes nothing.
*(2026-11-12: "Run by the first `v0.6.0` process that resolves credentials" now reads `v0.7.0`, the release
that carries this record's work. `v0.6.0` is the release of
[0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md),
cut before it.)*

1. Under `auth.json`'s lock, re-read the file and compute the changes of steps 3 to 5. None: stop.
2. **Back up** to `auth.json.bak-<UTC, YYYYMMDDTHHMMSSZ>` beside it, owner-only as `auth.json` is, before any write,
   when step 3 or 5 will change `auth.json`.
3. **Drop a dormant `key` or `workspace`** from each `oauth` entry (F25), and log one line per provider at Info, naming
   the provider and the backup's file name, never a value (Q22).
4. **Adopt the ids:** with no `<state>/host-id`, write the openai plan entry's `hostId` there; with no client record,
   write its `clientId` and `hostId` to `<state>/clients.json`.
5. **Carry a consent:** when the global `settings.yaml` holds `chatgptPlanBackground: true` (a build of the uncommitted
   work wrote it) and the openai plan session holds no answer, the consent is recorded granted through
   `MigrateOptions.CarryConsent`, a function `internal/cli` supplies from `provider.SetConsent` (D3). `internal/auth`,
   where the migration lives, may not import the SDK (archtest rule 4), so it calls the SDK only through that hook. The
   settings file is not edited; gobble reports the key as "not used by this gobble" (`internal/config/diag.go:44`).
6. Every other member and field is kept byte for byte (`member.value`).

*(2026-11-21, the owner's answers to the state table's Q1 to Q8, quoted in "The owner's answers (2026-11-21)" below. The
steps above are kept as written and read with these amendments.)*

* **Who runs it** (Q1, "All but completion, status, check (Recommended)"): "the first … process that resolves
  credentials" reads: every command that `run` dispatches, except `gobble auth status` and `gobble auth check`, which
  read the file as it is; completion mode never reaches `run`. 0045-MADR D4 is not amended.
* **The path** (Q7, "Resolve it; migrate the target (Recommended)"): `auth.json` is resolved with
  `filepath.EvalSymlinks`; the target is read, backed up beside the target, and written. The lock stays at the given
  path, `auth.json.lock` beside it.
* **Step 1:** a read without the lock comes first; the lock is taken, and the file re-read, only when that read finds
  work (the second draft's M41).
* **Step 2** (Q4, "Check-then-rename; back up key drops only (Recommended)"): "when step 3 or 5 will change `auth.json`"
  reads "when step 3 will change `auth.json`"; a carry alone makes no backup. The backup is a temporary file, owner-only,
  renamed to its name after a check, under `auth.json.lock`, that the name is free; if it is taken, the migration returns
  an error and writes nothing. A temporary file is removed on any failure.
* **Step 3** (Q4): the Info record of a dropped key can be lost at `--log-level warn` or `error`, or with no log file;
  the backup and the migration's `Report` are then the evidence.
* **Step 4** (Q2, "A second hook, PlanSession (Recommended)"; Q3, "Validate; lock-and-rename host-id (Recommended)"):
  the openai plan entry is the one `MigrateOptions.PlanSession` reports, a hook `internal/cli` builds from `llm/provider`.
  Its `hostId` is adopted only when it is a `urn:uuid:`, else nothing is adopted and the migration returns an error;
  `host-id` is written as D4's note of this date says; a client record is written only when the entry holds both ids and
  its `hostId` equals the host's id after step 4; a `host-id` holding anything but a `urn:uuid` (empty counts as absent)
  is an error, and nothing is adopted.
* **Step 5** (Q2; Q6, "Only into a kind-less entry; accept window (Recommended)"): the plan session and its answer come
  from `PlanSession`, and `Settings` is asked only when the plan session is unanswered. The consent is carried only into a
  plan entry that has no `kind` member (its presence is tested, the kind is not read). The window between the migration's
  check and `CarryConsent`, in which another process's `declined` would become `granted`, is accepted; the user
  withdraws a grant with `gobble auth consent openai background deny`. "`internal/auth` … calls the SDK only through that
  hook" holds, now of two hooks.
* **Step 6** (Q5, "Keep members' bytes, not whitespace (Recommended)"): "byte for byte" is members' bytes, not white
  space. Step 3's own write keeps each unchanged member's bytes, a one-line member included, lays the edited entry out
  as gobble's writes do, and writes the top level as `encode` does (LF, two spaces, a final newline), so a CRLF file or
  another indentation is normalised once. Step 5's write is gobble's ordinary write.

*(2026-11-20, re-grounded at `9332cef` before P5. The six steps above are unchanged.)*

* `internal/auth/migrate.go` is not in the tree. The pins are go-llmprovider-sdk `v1.31.0`, acp-go-sdk2 `v1.36.0` and acpext `v0.5.0-rc.1`. Remote tags end at those two SDK versions. magic-cli-remote requires the same acp-go-sdk2 tag. There is no gobble `v0.7.0` tag, and `v0.6.0` is an ancestor of `9332cef`.
* `internal/config/diag.go:44` is still `textUnknown = "not used by this gobble"`. The typed loader ignores an unknown key, and a diagnostic never carries the value (`internal/config/merge.go`, `internal/config/diag.go`), so `Settings` reads the global `settings.yaml` scalar itself. `(true, true)` means that file holds `chatgptPlanBackground: true`. A missing file is `(false, true)`. A file that cannot be read or parsed is `(false, false)`, and `Migrate` returns an error and writes nothing: the PLAN's existing error rule.
* F25 is already applied when an oauth entry is written: `oauthFields` removes `key` and `workspace` (`internal/auth/store.go`). Step 3 still removes them from an entry already stored.
* `HostID` creates a new id when `<state>/host-id` is absent (`internal/auth/hostid.go`). Step 4 writes the plan entry's `hostId` and does not call `HostID`. `Clients.Save` replaces a record (`internal/auth/clients.go`); an existing client record is left.
* `filelock` is not re-entrant (`internal/filelock`). `provider.SetConsent` locks the provider's refresh file, then `Logins.Save` locks `auth.json.lock` (`internal/auth/logins.go`, `internal/auth/store.go`). The re-read, the backup and the dormant-key write hold `auth.json.lock`, and that lock is released before `CarryConsent`.
* At go-llmprovider-sdk `v1.31.0`, a stored answer is `id=granted` or `id=declined` in `consents`, separated by spaces (`llmprovider/auth/consent.go`). The plan consent id is `chatgpt_plan.background_use`. No answer means that id has neither entry. `internal/auth` reads the stored string and does not import the SDK. `CarryConsent` is `provider.SetConsent` with that id and `granted` true.
  *(2026-11-21, second draft: "`internal/auth` reads the stored string" conflicts with D2's note of 2026-11-11, later,
  "The id is never a constant of gobble's": reading the stored string for that id puts the id in `internal/auth`'s code.
  Which holds is Q2 of the state table's second draft; until the owner answers it, P5 does not build on this sentence.)*
* Archtest rule 4 rejects a go-llmprovider-sdk import outside `llm/provider` in `checkRule4` (`internal/archtest/check.go`). The PLAN's `:308` is that function; the rejection is the later line of the same function.

*(2026-11-21, re-grounded at `4c3e397`, `main` after 0051's completion and 0063's merge, before P5. The six steps above are unchanged, and nothing below changes a decision.)*

* `internal/auth/`, `internal/archtest/`, `internal/config/`, `internal/appdirs/`, `internal/filelock/`, `jsonx/` and the non-test files of `llm/provider/` are unchanged since `9332cef` (`git diff --stat 9332cef 4c3e397`). Every fact of the note of 2026-11-20 about them holds, but those corrected below.
* The pins: go-llmprovider-sdk `v1.31.0`, acp-go-sdk2 `v1.36.0` and authcontract `v0.1.0`, as before; acpext is now `v0.5.0` (`go.mod:20`, `95f71cc`, 0051-PLAN P8a), no longer `v0.5.0-rc.1`. There is still no gobble `v0.7.0` tag. That the SDKs' remote tags end at those versions was not re-checked: this pass reached no outside service.
* Archtest rule 4: `checkRule4` begins at `internal/archtest/check.go:304`, and the rejection of a go-llmprovider-sdk import outside `llm/provider` is `:318-319`, the same at `9332cef`. The note's ":308 is that function" names the function; `:308` itself is its `relOf` skip. D3's `:308` reads the same way.
* `internal/auth` imports no go-llmprovider-sdk package, and it does import `llm/provider` (`internal/auth/logins.go:9`, `clients.go:17`; `AcceptsLogin` calls `provider.Accepts`, `resolve.go:38`). Step 5's "it calls the SDK only through that hook" is a rule of this record, not a consequence of the import graph: a call from `internal/auth` into `llm/provider` reaches the SDK with no import of it. How step 4 and step 5 find the plan session is the state table's open row M12.
* `File.encode` keeps a member's bytes as read only when they hold a line break; a member on one line is re-indented on every write (`internal/auth/store.go:387-390`), and `SetOAuth`, so `provider.SetConsent`, writes through it. Step 6's "byte for byte" therefore needs a write of the migration's own for step 3 (row M9); step 5's write is `SetConsent`'s (open row M10).
* `internal/cli` may not import `openai.ConsentBackgroundUse` (rule 4 admits only `authcontract` there, `check.go:315`), and D2's note says the id is never a constant of gobble's. The PLAN's P5 step 2, "openai's consent id", therefore reaches `internal/cli` through `llm/provider`: the `Consent` that `provider.PlanConsent` returns (`llm/provider/consent.go:115-130`), or the word `background`, which `provider.SetConsent` also takes (`:174`).
* SDK-2, D5's gate and the first bullet of "Not verified": go-llmprovider-sdk `0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md`, "Phase 6, the owner's run (2026-11-14)", records `TestLive_0075ClientReuse` passing, with "OpenAI accepted the client with no id_token_hint; the client and host id are unchanged". D5's fallback is not needed. Read in that repository's tree, not run here.

### State table for what remains: D9's migration and P6's acceptance (2026-11-21)

*(2026-11-21, a dated amendment; records only.)* This record was written before the owner's rule that a record which changes behaviour carries a state table deciding every case before code. P0 to P4 are built and on `main`; what remains is D9's migration (the PLAN's P5) and the acceptance with the owner's live checks L4 to L7 (P6). One row per input or state. A row is **decided** (the outcome a test checks, where the record or an owner answer decides it, and the test that pins it, "proposed" for a test the PLAN's readiness entry of this date adds), **out of scope** (with the reason), **open** (the record does not decide it; the case and today's code are stated, and the question is put to the owner below), or **open: awaits live evidence** (the outcome depends on the owner's live check). Rows are numbered M (the migration), A (P6's acceptance), O (out of scope) and L (the live checks). An "owner answer" is quoted by its label as the record writes it. This amendment changes no decision, no status and no owner answer; by the owner's rule the open rows go to the owner in one prompt before P5's code.

Counts: 43 decided, 4 out of scope, 13 open, 4 awaiting live evidence.
*(2026-11-21, second draft: these are the first draft's counts. "Second draft (2026-11-21, after the independent record review)", below, changes and adds rows and gives the counts that hold.)*

**The file as found**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| M1 | No `auth.json` | decided | Nothing is written: no backup, no state file, no log record, `CarryConsent` not called; `Report` says nothing was done. Whether the lock file and the config directory are created is M41 | D9 step 1 ("None: stop") | `TestMigrateNothingToDo` (P5) |
| M2 | `auth.json` empty, or white space only | decided | As M1: the store reads it as no entries (`internal/auth/store.go:90-92`) | D9 step 1 | proposed: `TestMigrateNothingToDo`, extended (P5) |
| M3 | `auth.json` that cannot be read or parsed: not an object, a member that is not an object with a string `type`, a duplicate member name, invalid UTF-8, a file the process may not read | decided | `Migrate` returns an error naming `auth.json`; nothing is written (no backup, no state file, no consent); the command runs on with the file as it is | the PLAN's P5 step 2 ("an error is a warning and the run goes on with the file as it is"); D9's note of 2026-11-20 ("returns an error and writes nothing") | proposed: `TestMigrateUnreadableFileWritesNothing` (P5) |
| M4 | An `oauth` entry holding `key` and `workspace` | decided | Both members leave that entry; one Info record per provider names the provider and the backup's file name, never a value; an Info record reaches the log file only, never stderr (`internal/logging/logging.go:12`) | D9 step 3; Q22 | `TestMigrateDropsDormantKey`, `TestMigrateKeepsEverythingElse` (P5) |
| M5 | An `oauth` entry holding `key` alone, or `workspace` alone | decided | The one present leaves the entry, with the provider's one Info record | D9 step 3 ("a dormant `key` or `workspace`") | proposed: `TestMigrateDropsDormantKey`, extended (P5) |
| M6 | An `oauth` entry with neither, and no work for steps 4 and 5 | decided | No write, no backup, no record | D9 steps 1 and 3 | proposed: `TestMigrateNothingToDo`, extended (P5) |
| M7 | An `api_key` entry holding `key` and `workspace` | decided | Kept byte for byte: step 3 acts on `oauth` entries only | D9 steps 3 and 6 | proposed: `TestMigrateKeepsEverythingElse`, its fixture extended (P5) |
| M8 | Unknown members; unknown fields in an entry; an entry of a type gobble does not use; the members' order; a member laid out over several lines | decided | Each kept byte for byte, in the file's order (`member.value`) | D9 step 6; the PLAN's P5 note of 2026-11-20 | `TestMigrateKeepsEverythingElse` (P5) |
| M9 | An unchanged member written on one line, when step 3 writes the file | decided | Kept byte for byte. `File.encode` re-indents a one-line member (`store.go:387-390`), so step 3 does not write through `encode` as it stands | D9 step 6 | proposed: `TestMigrateKeepsEverythingElse`, its fixture extended (P5) |
| M10 | The layout of the entry step 3 edits (its white space; its other fields' values are kept); and an unchanged one-line member when step 5 writes through `provider.SetConsent`, whose `SetOAuth` writes with `encode` | open (Q6) | Today: `merge` compacts the edited entry, and `encode` lays it out two spaces deep; every write gobble makes re-indents a one-line member | — | — |
| M11 | A second run after a completed migration | decided | Nothing written, no backup, no record, `CarryConsent` not called | D9 ("idempotent by content") | `TestMigrateIsIdempotent` (P5) |

**The plan entry, the ids and the state files**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| M12 | Telling the openai plan entry, and whether its consent is answered: an entry whose stored `kind` is `chatgpt_plan`; an entry with no `kind` (a build of the archived work wrote none, and only the SDK infers one) | open (Q3) | Today: the kind is read in one place, `provider.SessionKindOf` (`llm/provider/signinrun.go`), by the owner's ruling of 2026-11-16, "Keep it, and make it the single place the kind is read"; D2's note says the consent id is never a constant of gobble's; D9's note of 2026-11-20 has `internal/auth` read the stored `consents` string for that id; `internal/auth` already imports `llm/provider` | — | — |
| M13 | A foreign openai `oauth` entry: another application's ChatGPT login, its stored `kind` `chatgpt` | decided | Step 3 applies to it; no id is adopted from it; no consent is carried | D9 steps 3 to 5 ("the openai plan entry", "the openai plan session") | proposed: `TestMigrateSkipsForeignEntry` (P5) |
| M14 | The plan entry holds `hostId` and `clientId`; no `<state>/host-id`; no `clients.json` | decided | `host-id` holds the entry's `hostId`, one line, owner-only; `clients.json`'s `openai` record is the entry's `clientId` and `hostId`; `auth.json`'s entry keeps both members | D9 steps 4 and 6; D4; D5 | `TestMigrateAdoptsIDs` (P5) |
| M15 | The plan entry holds `hostId` and no `clientId` | open (Q4) | Today: no code; "write its `clientId` and `hostId`" assumes both | — | — |
| M16 | The plan entry holds `clientId` and no `hostId` | open (Q4) | Today: no code; as M15 | — | — |
| M17 | `<state>/host-id` exists | decided | Not overwritten | D9 step 4 ("with no `<state>/host-id`") | `TestMigrateAdoptsIDs` (P5) |
| M18 | `<state>/host-id` exists and differs from the plan entry's `hostId`, and `clients.json` has no `openai` record | open (Q4) | Today: no code. A record holding the entry's `hostId` would name a host id the file does not hold | — | — |
| M19 | `<state>/host-id` holds no `urn:uuid` | open (Q4) | Today: `HostID` returns an error and does not replace the file (`internal/auth/hostid.go:59-75`); step 4 does not call `HostID` | — | — |
| M20 | `clients.json` has an `openai` record | decided | Left as it is | D9 step 4; D9's note of 2026-11-20 ("an existing client record is left") | `TestMigrateAdoptsIDs` (P5) |
| M21 | `clients.json` has other providers' records and none for `openai` | decided | `openai`'s record is added; the others are kept | D9 step 4 ("with no client record"); `Clients.Save` | proposed: `TestMigrateAdoptsIDs`, extended (P5) |
| M22 | `clients.json` cannot be parsed | decided | `Migrate` returns an error; `clients.json` is not replaced | the PLAN's P5 step 2; `Clients.Save` refuses such a file (`internal/auth/clients.go:56-59`) | proposed: `TestMigrateBadClientsFileIsAnError` (P5) |
| M23 | Step 4 has work, and steps 3 and 5 have none | decided | The state files are written, and no backup is made | D9 step 2 ("when step 3 or 5 will change `auth.json`") | proposed: `TestMigrateAdoptsIDs`, extended (P5) |

**The consent carried**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| M24 | `chatgptPlanBackground: true` in the global `settings.yaml`, and an unanswered plan session | decided | `CarryConsent` is called once; the session then reads `granted`; `settings.yaml` is not written | D9 step 5 | `TestMigrateCarriesConsent` (P5) |
| M25 | The same, with the session answered `declined` or `granted` | decided | `CarryConsent` is not called; the answer is unchanged | D9 step 5 | `TestMigrateCarriesConsent` (P5) |
| M26 | `chatgptPlanBackground: false`, or absent | decided | `CarryConsent` is not called | D9 step 5 ("holds `chatgptPlanBackground: true`") | proposed: `TestMigrateCarriesConsent`, extended (P5) |
| M27 | `chatgptPlanBackground: true` and no plan session: no openai entry, an `api_key` entry, a foreign session | decided | `CarryConsent` is not called | D9 step 5 | proposed: `TestMigrateCarriesConsent`, extended (P5) |
| M28 | The global `settings.yaml` cannot be read or parsed, and there is an unanswered plan session | decided | `Settings` gives `(false, false)`; `Migrate` returns an error and writes nothing | D9's note of 2026-11-20 | proposed: `TestMigrateUnreadableSettingsWritesNothing` (P5) |
| M29 | The global `settings.yaml` cannot be read or parsed, and there is no unanswered plan session | open (Q3) | Today: no code. D9's note says "a file that cannot be read or parsed is `(false, false)`, and `Migrate` returns an error and writes nothing", without saying whether `Settings` is asked when step 5 has nothing to carry | — | — |
| M30 | `chatgptPlanBackground: true` in a project `settings.yaml` only | decided | Not read: `Settings` gives `(false, true)`, and `CarryConsent` is not called | D9 step 5 ("the global `settings.yaml`"); D9's note of 2026-11-20 | proposed: `TestMigrateSettingsIsGlobalOnly` (`internal/cli`, P5) |
| M31 | The key left in `settings.yaml` after the migration | decided | The file is not edited; gobble reports the key as "not used by this gobble" | D9 step 5; `internal/config/diag.go:44` | `TestMigrateCarriesConsent` (P5); `TestUnknownKeyReported` (`internal/config/resolve_test.go:267`, existing) |
| M32 | `CarryConsent` returns an error (the store unreadable or locked, the session gone) | decided | `Migrate` returns it; what steps 3 and 4 wrote stands; the next run calls `CarryConsent` again | the PLAN's P5 step 2; D9 ("idempotent by content") | proposed: `TestMigrateCarriesConsent`, extended (P5) |
| M33 | Another process records an answer for the plan consent after `Migrate` found none and before `CarryConsent` writes (`auth.json.lock` is released before it, D9's note of 2026-11-20) | open (Q7) | Today: `provider.SetConsent` stores its answer whatever the session holds (`llm/provider/consent.go:168-179`), so a `declined` given in that window becomes `granted` | — | — |

**Writes, failures and two processes**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| M34 | The backup: when it is made, where, and its name | decided | Made when step 3 or 5 will change `auth.json`, before any write, beside `auth.json`, named `auth.json.bak-<UTC, YYYYMMDDTHHMMSSZ>`, holding the old bytes | D9 step 2 | `TestMigrateBacksUpFirst` (P5); proposed: extended with a clock in a zone other than UTC (P5) |
| M35 | The backup's permissions on each platform | decided | Owner-only, as `auth.json`: created `0600` through a temporary file on Linux and macOS; the user's DACL alone before its first byte on Windows (`securePrivate`); checked with `appdirs.FileIsOwnerOnly`, as `TestStoreMode` checks `auth.json` | D9 step 2; the PLAN's P5 note of 2026-11-20 | `TestMigrateBacksUpFirst` (P5), on the three hosts (C6) |
| M36 | A file of the backup's name already exists (two runs in one second) | open (Q5) | Today: no code; the store's `write` replaces its target by a rename (`store.go:412-425`) | — | — |
| M37 | The backup cannot be written | decided | Nothing else is written: not `auth.json`, no state file, no consent; `Migrate` returns the error | D9 step 2 ("before any write") | proposed: `TestMigrateBackupFailureWritesNothing` (P5) |
| M38 | `auth.json` cannot be written after the backup | decided | `auth.json` is unchanged; the backup is kept; `Migrate` returns the error; the next run backs up again | D9 step 2; the PLAN's P5 step 3 | `TestMigrateBacksUpFirst` (P5) |
| M39 | A state file cannot be written (the state directory is not writable, or is not a directory) | decided | `Migrate` returns the error; `auth.json`'s changes and the backup stand; the next run that can write the state directory writes the state files and makes no new backup | the PLAN's P5 step 2; D9 ("idempotent by content") | proposed: `TestMigrateStateWriteFailureIsAnError` (P5) |
| M40 | A run stopped between its writes (killed, power lost) | decided | `auth.json` holds the old bytes or the new, never a mix, since it is replaced through a temporary file; the next run does what remains, and makes no backup when step 3 is already done | D9 step 1 and "idempotent by content"; 0005-PLAN F4-3a decision 4 (`auth.json` written through a temporary file) | proposed: `TestMigrateCompletesAfterInterruption` (P5) |
| M41 | Nothing to migrate, and `auth.json.lock` cannot be created (a read-only or permission-denied config directory) or another process holds it | open (Q1) | Today: `modify` creates the config directory and takes `auth.json.lock` before it reads (`store.go:358-375`), waiting up to 30 s (`authPolicy`, `:28`); D9 step 1 says "under `auth.json`'s lock, re-read the file" | — | — |
| M42 | Two gobble processes start at once over a file with a dormant key | decided | One backup and one write: the second process re-reads under the lock and finds nothing to do | D9 step 1 | proposed: `TestMigrateConcurrentRunsWriteOnce` (P5) |
| M43 | `Migrate` writes `<state>/host-id` while another process creates it (`HostID` at a sign-in, or a second `Migrate`) | decided | An exclusive create: the first file stays, and `Migrate` does not replace it | D4 ("created once with an exclusive create … A race between two processes keeps the first"); D9 step 4 | proposed: `TestMigrateHostIDRaceKeepsFirst` (P5) |

**The call, and the modes it runs in**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| M44 | Completion mode | decided | No migration: `Main` returns before `run` (`internal/cli/main.go:45-47`) | the PLAN's P5 note of 2026-11-20 ("Completion mode returns in `Main` before `run`") | proposed: `TestCompletionModeDoesNotMigrate` (`internal/cli`, P5) |
| M45 | A command that resolves no credential (`gobble version`, `gobble config path`), and whether such a command opens the log file | open (Q1) | Today: D9 runs the migration in "the first … process that resolves credentials"; the PLAN's P5 note puts the call in `run` before `kctx.Run`, for every command; `lazyLog` exists so that such commands create no state directory or log file (`main.go:350-353`) | — | — |
| M46 | `gobble acp` | decided | The migration runs once, before the agent serves; nothing of it reaches stdout; its Info records go to the log file (the log has no stderr under `acp`, `main.go:87-89`) | the PLAN's P5 note of 2026-11-20 (the call in `run`); D9 step 3 | proposed: `TestACPStartMigrates` (`internal/cli`, P5) |
| M47 | An error from `Migrate` under `--json` (`auth status --json`, `auth login --json`) and under `gobble acp` | open (Q2) | Today: a warning is a line on stderr (`Output.Warnf`); `auth status --json` writes nothing to stderr (0045-MADR D4; D8's last bullet), and neither does `auth login --json` (`main.go:138-148`) | — | — |
| M48 | An error from `Migrate` in any other command | decided | One warning line on stderr naming the error; the command runs, and its exit status is its own | the PLAN's P5 step 2 | proposed: `TestMigrateErrorIsAWarning` (`internal/cli`, P5) |

**P6's acceptance**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| A1 | Confirmation 1, D1's table | decided | P1's sort script, run again at P6's start commit, finds no change group of `1406bd6` unaccounted; P4's whole-word grep exits 1 | Confirmation 1; P4's acceptance as the owner amended it, "Amend the acceptance to a whole-word grep" | the sort script (P1, a scratch script) and the grep (P6) |
| A2 | Confirmation 2, the wording first on every surface | decided | gobble's wording before any URL on the five terminal surfaces and over ACP; `--json` refuses with exit 2 | Confirmation 2 | `TestConsentWordingOnEverySurface`, `TestSignInChatGPTPlanTerminal`, `TestConsentWordingOverACP`, `TestAuthLoginJSONRefusesAQuestion` (existing) |
| A3 | Confirmation 3, the consent changed from the next request | decided | As Confirmation 3 and its note | Confirmation 3 | `TestAuthConsentCommand`, `TestConsentAppliesFromTheNextRequest`, `TestPlanConsentIsReadFromAuthJSON`, `TestRefusedBackgroundCarriesConsent` (existing) |
| A4 | Confirmation 4, one host id across two sign-ins, a logout and a key switch; the client reused after a logout | decided | As Confirmation 4 | Confirmation 4 | `TestSignInChatGPTPlanACP`, `TestSignInChatGPTPlanTerminal`, `TestPlanAfterLogoutReusesClientACP`, `TestPlanAfterLogoutReusesClientTerminal` (existing); the key switch: proposed `TestKeySwitchKeepsHostAndClient` (`internal/cli`, P6) |
| A5 | Confirmation 5, `auth status --json` | decided | As Confirmation 5 and its note | Confirmation 5 | `TestAuthStatusJSONPlan`, `TestForeignOpenAISessionRefused`, `TestAuthStatusJSON` (existing) |
| A6 | Confirmation 6, the migration on copies of real-shaped files | decided | Rows M1 to M48 that are decided, each tested | Confirmation 6 | P5's tests and the proposed ones |
| A7 | Confirmation 7, each new test seen failing; the gates on three hosts | decided | Every plant of P5 and of the proposed tests fails its test on a scratch copy; the C6 gates pass at P6's start commit on the three hosts | Confirmation 7 | the plants and the gates (P5, P6) |
| A8 | The migration run on the owner's real `auth.json` before `v0.7.0` is tagged | open (Q8) | Today: Confirmation 6 uses copies of real-shaped files; no agent reads the owner's files (C10); the first run on the real file is the owner's first `v0.7.0` start ("Bad, because the migration writes the owner's real `auth.json` once") | — | — |

**Out of scope**

| Row | Input or state | Status | Reason |
| --- | --- | --- | --- |
| O1 | A held session that adopts a foreign entry; the wizard's no-listener option | out of scope | Not this phase: neither is in the go-llmprovider-sdk `v1.31.0` tag this repository requires (the PLAN's P5 note of 2026-11-20); each is a step of the phase that bumps the SDK pin (P4 steps 1c and 1d) |
| O2 | Restoring `auth.json` from a backup, or removing old backups | out of scope | Q22 asks that the backup be kept; this record gives gobble no command that reads or removes one |
| O3 | A `v0.6.0` binary run on a migrated `auth.json` | out of scope for P5's and P6's tests | The PLAN's "Rollback after `v0.6.0`" (as dated 2026-11-12) decides it by reading (`jsonx.StrictOptions`, `merge`); neither phase builds the earlier binary |
| O4 | An agent running the migration on the owner's real files | out of scope | The PLAN's Scope ("reading the owner's real `auth.json`, `settings.yaml` or state directory") and C10 |

**The owner's live checks (Confirmation 8)**

| Row | Input or state | Status | What the offline tests already pin, and what only the live check shows | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| L4 | A first plan sign-in from another device through ACP elicitation | open: awaits live evidence | Offline: the `confirm` form first, in gobble's wording; the host id and the client store passed (`TestConsentWordingOverACP`, `TestSignInChatGPTPlanACP`). Live: whether OpenAI's real sign-in, from the owner's phone through a form-only ACP client, completes by the pasted address and saves a `chatgpt_plan` session holding the answer given | the PLAN's P6 step 2, L4; 0051-MADR D4 | the kit's L4 result (P6) |
| L5 | `gobble auth login openai` on a real terminal, the browser on another device, the address pasted | open: awaits live evidence | Offline: `TestSignInChatGPTPlanTerminal`; the paste is read through 0063's one terminal input, and a loopback that arrives first ends the paste's read (0063-MADR D1, D5). Live: OpenAI's page and a real paste | the PLAN's P6 step 2, L5 | the kit's L5 result (P6) |
| L6 | `gobble auth logout openai`, then L5 again | open: awaits live evidence | Offline: `TestPlanAfterLogoutReusesClientTerminal`, `TestPlanAfterLogoutReusesClientACP`, `TestLogoutRevokesAsGobble`; SDK-2's probe passed on 2026-11-14 (D9's note of this date). Live: no "could not be revoked" warning (a refusal is a deviation, 0051-PLAN C4); whether the account's page shows a second client | the PLAN's P6 step 2, L6, and its note of 2026-11-12 | the kit's L6 result (P6) |
| L7 | The consent end to end | open: awaits live evidence | Offline: `TestTitleUnderPlan`, `TestTitleNotMadeSendsNothing`, `TestAuthConsentCommand`. Live: after `deny` by the word `background`, no title for a short session; after `allow` by the id with no terminal attached, one title | the PLAN's P6 step 2, L7, and its note of 2026-11-11 | the kit's L7 result (P6) |

**Questions for the owner (2026-11-21).** Each states the case and today's code, then three options, the recommended one first, each with its cost.
*(2026-11-21, second draft: these eight questions are superseded, unasked, by the second draft's Q1 to Q8 below. They are kept as written.)*
*(2026-11-21, third draft: the first draft's numbers against the questions that now hold, those of the third draft below. First-draft Q1 (M41, M45): M41 decided in the second draft; M45 is Q1. Q2 (M47): M47 replaced by M55, decided. Q3 (M12, M29) is Q2. Q4 (M15, M16, M18, M19) is Q3. Q5 (M36) is Q4. Q6 (M10) is Q5, with M9. Q7 (M33) is Q6. Q8 (A8) is Q8. The first draft's row text that names an old question number reads with this map.)*

* **Q1 (M41, M45). When the migration starts.** D9 says the first process that resolves credentials, under `auth.json`'s lock; the PLAN's P5 note puts the call in `run` for every command; `modify` makes the config directory and the lock file before it reads, and waits up to 30 s for the lock. (a) *Recommended:* the call in `run` for every command, reading `auth.json` first without the lock and taking the lock and re-reading (D9 step 1's "re-read") only when that read finds work; the log opened only when there is a record to write. Cost: one more read when there is work; D9 step 1 annotated. (b) The call in `run`, the lock taken first, as D9 step 1 reads. Cost: every command makes the config directory and `auth.json.lock`; a read-only config directory warns on every run; a held lock delays every start by up to 30 s. (c) The call only where credentials are resolved: the resolver's build and each command that reads `auth.json` itself (`auth consent`, `auth logout`, `authenticate`, `acp --login`), lock first. Cost: several call sites, and a later reader that forgets the call reads an unmigrated file.
* **Q2 (M47). Where an error from `Migrate` goes under `--json` and `gobble acp`.** Today a warning is a stderr line; `auth status --json` and `auth login --json` write nothing to stderr. (a) *Recommended:* under `--json` and `gobble acp`, a Warn record to the log file and nothing on stderr; elsewhere one stderr line (M48). Cost: a `--json` or ACP caller learns of it only from the log, or from the error its command then meets. (b) One stderr line everywhere. Cost: breaks 0045-MADR D4 and `auth login --json`'s empty stderr. (c) The log file only, everywhere. Cost: a person at a terminal is not told the file was not migrated.
* **Q3 (M12, M29). How `Migrate` tells the openai plan entry and its answer, and when it reads `settings.yaml`.** Today the kind is read only in `provider.SessionKindOf`; D2's note says the id is never a constant of gobble's; an entry a build of the archived work wrote has no `kind`. (a) *Recommended:* a second hook, `MigrateOptions.PlanSession func(ctx context.Context) (plan, answered bool, err error)`, which `internal/cli` builds from `llm/provider` (`SessionKindOf`, `Consents`); `Settings` asked only when the plan session is unanswered. Cost: one more field in P5 step 1's signature, and D9's note annotated. (b) `internal/auth` calls `llm/provider` itself, which it already imports; `Settings` as in (a). Cost: D9 step 5's "it calls the SDK only through that hook" stops being true and is annotated. (c) `internal/auth` reads the stored `kind` and `consents` with the id written in its code; no `kind`, no plan entry; `Settings` asked on every run with work. Cost: a gobble constant against D2's note; a kindless plan session keeps its ids and consent unmigrated; an unreadable `settings.yaml` stops every migration.
* **Q4 (M15, M16, M18, M19). The ids when one is missing, or the host's file disagrees.** No code today; D9 step 4 assumes both ids and no `host-id`. (a) *Recommended:* `host-id` from `hostId` alone when the file is absent; a client record only when the entry holds both ids and its `hostId` equals the host's id after step 4; a `host-id` holding no `urn:uuid` is an error, and nothing is adopted. Cost: such an entry's client is registered again at the next plan sign-in. (b) Adopt what is present, a record with an empty member included, and the entry's `hostId` even when the file differs. Cost: `clients.json` can name a host id `host-id` does not hold; whether the SDK reuses such a client is not known. (c) Adopt only a complete entry into a host with neither file; anything else adopts nothing, with no error. Cost: more new registrations, and a bad `host-id` goes unreported here.
* **Q5 (M36). The backup's name already exists.** No code today. (a) *Recommended:* an exclusive create; on a collision `Migrate` returns an error and writes nothing, and the next run makes the backup. Cost: a warning in that rare case. (b) A counter suffix on a collision. Cost: the name grows beyond D9 step 2's. (c) Replace the existing file. Cost: an earlier backup of other bytes can be lost.
* **Q6 (M10). "Byte for byte" and layout.** Today `merge` compacts an edited entry and `encode` re-indents a one-line member on every write, `SetConsent`'s included. (a) *Recommended:* step 6 means members' and fields' bytes, not white space: step 3's own write keeps every unchanged member's bytes (M9) and lays the edited entry out as gobble's writes do; step 5's write is gobble's ordinary write; step 6 annotated. Cost: a hand-compacted member is laid out again when a consent is carried, its values unchanged. (b) As (a), and `encode` changed to keep every member as read. Cost: a change to every `auth.json` write gobble makes, beyond D9. (c) Step 3 uses gobble's ordinary write too, and step 6 reads "values kept". Cost: M9 is reversed.
* **Q7 (M33). An answer recorded between `Migrate`'s check and `CarryConsent`.** Today `SetConsent` writes whatever the session holds; the SDK's `auth.SetConsent` has no conditional form, and `filelock` is not re-entrant. (a) *Recommended:* accept the window and say so in D9; the user can withdraw a grant with `gobble auth consent openai background deny`. Cost: none in code; the window is two calls in one start, and only a build of the archived work wrote the setting. (b) `CarryConsent` reads the answer again just before writing. Cost: narrows the window, does not close it. (c) A conditional set in the SDK, used here. Cost: a go-llmprovider-sdk record and release before P5.
* **Q8 (A8). Evidence of the migration on the owner's real file before the tag.** No check today. (a) *Recommended:* an owner-run L8 in the kit: the owner copies their own `auth.json` and global `settings.yaml` into a scratch home, runs P5's build there twice, and the kit reports fixed words only (backup made and owner-only, keys dropped by count, state files written, the second run wrote nothing, every other member unchanged by comparison). Cost: one more owner step; no agent sees the file. (b) No live check: Confirmation 6's copies stand, and the backup is the recovery. Cost: the first run on the real shapes is the release. (c) The owner inspects the backup after the first `v0.7.0` start. Cost: the check comes after the release.

#### Second draft (2026-11-21, after the independent record review)

*(2026-11-21, a dated addition; records only.)* An independent review of the first draft, read at `4c3e397`, found cases with no row, rows decided by accepted records but put as questions, options that contradicted a decided row or an accepted record, and tests that could not be written or could not fail. The coordinator's disposition of it, the same day: decide in rows the cases an accepted record or this record already settles; fix the tests and plants (the PLAN's "Second draft" entry); put the rest to the owner as at most eight self-contained questions. The rows of the first draft stand as written, except where a row below changes one; the first draft's questions are superseded by the eight below. Where a row below is decided on the review's finding rather than on a line of this record, its "Decided by" says so, for the owner's review.

**Counts, second draft:** 50 decided, 5 out of scope, 21 open (Q1 to Q8), 4 awaiting live evidence; 80 rows. M47 is replaced by M55 and is not counted.
*(2026-11-21, third draft: superseded by the third draft's counts below.)*

**Rows of the first draft whose status or text changes**

| Row | Change | Now |
| --- | --- | --- |
| M9 | decided → open, by the disposition: it and M10 are one question, since an option on M10 would reverse it | open (Q5) |
| M41 | open → decided. D9 step 1 says "under `auth.json`'s lock, re-read the file", so a read comes before the lock: with nothing to migrate no lock is taken and nothing is created (no config directory, no `auth.json.lock`), so a read-only config directory gives no warning and a held lock delays no start. With work, the lock is waited for up to 30 s (`authPolicy`, `internal/auth/store.go:28`). The OS releases a crashed process's lock (`flock`, `internal/filelock/lock_unix.go`; `LockFileEx`, `lock_windows.go`) | decided; test: proposed `TestMigrateNothingToDoTakesNoLock` (P5) |
| M45 | the log's opening is decided by M55; which commands migrate stays open | open (Q1) |
| M47 | replaced by M55. Its citation for `auth login --json` was wrong: the commands silence stderr and the log's stderr inside their own `Run`, `internal/cli/auth.go:72-73` (`auth login --json`) and `:522-523` (`auth status --json`), not `main.go:138-148` | superseded by M55 |
| M3 | its test's file of mode `000` is skipped when the test runs as root, who reads it | decided |
| M4 | its test holds two `oauth` entries with dormant keys, so that one record naming both providers fails it | decided |
| M8 | its plant is named against the migration's own edit path (M9), not `merge`: step 3's removal dropping an unknown field of the edited entry | decided |
| M13, M24 | M13 depends on Q2. M24 gains `TestMigrateCarriesConsentEndToEnd` (`internal/cli`), whose fixture follows Q1, Q2 and Q6 | decided |
| M14 | its test also asserts `host-id` owner-only, with a plant on the migration's own write of it | decided |
| M37 | a backup left partly written (a full disk) is removed: the backup is written through a temporary file (M35), removed on any failure, as `write` removes its own (`internal/auth/store.go:418-422`) | decided |
| M38 | it names the Windows case: another gobble process holding `auth.json` open without the lock can make the rename fail; the outcome is this row, with no retry | decided |
| M40 | a stray `.auth.json.*.tmp` left by a run killed inside `write` (`internal/auth/store.go:413-423`) is O5 | decided |
| M42 | its plant is reworded, and its interleaving is forced by a test hook (the PLAN's "Second draft") | decided; depends on nothing open now that M41 is decided |
| M43 | decided only for the create: when `Migrate` loses the race to a new id from `HostID`, the case is M18's (open, Q3) | decided, with M18 |
| M48 | depends on Q1 (which commands migrate, so which command its test runs) and on M55; its fixture becomes an unparseable `clients.json`, so the state directory and the log work and only the migration's warning is seen | decided |
| A8 | reframed by M54: any build carrying P5 migrates, so the question is evidence before the owner's first such build runs on the owner's own home | open (Q8) |
| L4 to L7 | an expected column, from the record (the table below) | open: awaits live evidence |

**Rows added**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| M49 | An `oauth` entry under a provider name gobble does not know, holding a dormant `key` | decided | Step 3 applies to it; its other members, the ids and any consent are untouched | D9 step 3 ("from each `oauth` entry") | proposed: `TestMigrateDropsDormantKey`, extended (P5) |
| M50 | An `oauth` entry whose `key` or `workspace` member is present with an empty string or `null` | decided | The member is removed whatever its value: removal is by name, as `oauthFields` removes them (`internal/auth/store.go:286`) | D9 step 3; F25 | proposed: `TestMigrateDropsDormantKey`, extended (P5) |
| M51 | A signal, or another cancellation of the run's context, during the migration | decided | The migration stops before its next step; a write already begun completes (the temporary file and the rename); `run` then returns without `kctx.Run`, and the exit status is 128 plus the signal (`signalExit`, `internal/cli/main.go:57-59`); the next run does what remains | D9 ("idempotent by content"); `watchSignals` (`internal/cli/signal.go:91-92`) and `filelock.Lock`'s return of the context's error; decided in a row by the coordinator's disposition of the review's finding 1i | proposed: `TestMigrateStopsWhenCancelled` (`internal/auth`), `TestMigrateSignalSkipsTheCommand` (`internal/cli`) (P5) |
| M52 | The config or the state directory cannot be resolved (`authPath()` is `""`, or `systemDirs` fails) | decided | The migration is skipped with one warning, as M48; under `auth status --json` and `auth login --json` nothing of it reaches stderr (M55). Which commands meet it follows Q1 | 0045-MADR D4, D5; 0051-MADR D5; M48; decided in a row by the coordinator's disposition of the review's finding 1l | proposed: `TestMigrateSkippedWithoutDirs` (`internal/cli`, P5) |
| M53 | `chatgptPlanBackground` in the global `settings.yaml` holding something other than the YAML boolean `true`: the string `"true"`, `yes`, `1` | decided | Only the YAML boolean `true` gives `(true, true)`; any other value gives `(false, true)`, and nothing is carried | D9's note of 2026-11-20 ("`(true, true)` means that file holds `chatgptPlanBackground: true`"); decided in a row by the coordinator's disposition of the review's finding 1m | proposed: `TestMigrateSettingsIsGlobalOnly`, extended (`internal/cli`, P5) |
| M54 | A build of `main` that carries P5 but is not the `v0.7.0` release | decided | It migrates as `v0.7.0` would: no step of D9 checks a version. D9's "the first `v0.7.0` process" names the release that carries the work, not a gate | D9 steps 1 to 6; decided in a row by the coordinator's disposition of the review's finding 1n | proposed: `TestMigrateRunsOnAnyBuild` (`internal/cli`, P5) |
| M55 | The migration under `auth status --json`, `auth login --json` and `gobble acp` (in place of M47) | decided | Nothing of the migration reaches stderr. The call site reads the command's `--json` from the parsed command before `Migrate` runs (as `jsonLoginParseFailure` reads it from the parser's context, `internal/cli/main.go:149-171`), and `Migrate`'s logger writes to the log file only and is opened lazily, at its first record, so the run's log is never opened with stderr attached before those commands turn it off (`internal/cli/auth.go:72-73`, `:522-523`). An error is a Warn record in the log file. Every other command, the other commands with a `--json` flag included (`gobble update --json` sends everything but its JSON to stderr), is M48 | 0045-MADR D4 ("stderr and log lines to stderr are turned off … The log file keeps them"); 0051-MADR D5; 0018-MADR D7 (`warnf`, `internal/cli/settings.go:183-191`); `main.go:87-89` | proposed: `TestMigrateSilentUnderJSONAndACP` (`internal/cli`, P5) |
| M56 | `gobble auth status` (with or without `--json`) and `gobble auth check`, which 0045-MADR D4 says write nothing to `auth.json` | open (Q1) | Today: no code. D9 and the PLAN's P5 note would run the migration, and step 5's `SetOAuth` write, inside them | — | — |
| M57 | The plan entry's `hostId` is not a `urn:uuid` | open (Q3) | Today: no code. Copied to `<state>/host-id`, it makes every later `HostID` fail, "does not hold a urn:uuid" (`internal/auth/hostid.go:71-72`) | — | — |
| M58 | `<state>/host-id` exists and is empty: a create interrupted between its `O_EXCL` open and its first byte (`hostid.go:35-45`) | open (Q3) | Today: step 4 would skip it as existing; `HostID` waits 5 s and fails "stayed empty" on every call (`:86-87`), so the plan sign-in fails for good | — | — |
| M59 | A carry or a write that fails and is retried at each start (M32, M38) | open (Q4) | Today: D9 step 2 backs up again at each retry; each backup holds the secrets, and none is removed (O2) | — | — |
| M60 | The Info record of a dropped key, at `--log-level warn` or `error`, or with no log file | open (Q4) | Today: Info is dropped below the level (`internal/logging/logging.go:12`, `:50`); with no log file `lazyLog.open` goes on with none (`internal/cli/main.go:391-404`), so Q22's line is lost | — | — |
| M61 | The file's top-level layout when step 3 writes it: CRLF line ends, an indentation other than two spaces, no final newline | open (Q5) | Today: `encode` writes LF, two spaces and a final newline (`internal/auth/store.go:381-396`) | — | — |
| M62 | Step 5 at a later start: `chatgptPlanBackground: true` stays in `settings.yaml` (M31), so step 5 fires at any later start that finds an unanswered plan session | open (Q6) | Today: no code; D9 makes the migration idempotent by content, and step 5's condition stays true while the key is there | — | — |
| M63 | `auth.json` is a symbolic link | open (Q7) | Today: `fsx.ReadRegularFile` follows it (`internal/fsx/regular.go:35`) and `write` renames over the path (`internal/auth/store.go:413-421`), so the link becomes a plain file and its target keeps the dormant key; where the backup goes ("beside it") is not decided | — | — |
| M64 | Only step 5 will change `auth.json` | open (Q4) | D9 step 2 decides a backup ("when step 3 or 5 will change `auth.json`"); with M59's retries such a backup is made at every start until the carry succeeds | — | — |
| O5 | A stray `.auth.json.*.tmp` after a run killed inside `write` | out of scope | As for every write of `auth.json` today (`internal/auth/store.go:413-423`); named in M40 | — | — |

**The owner's live checks, with what the record expects** (in place of the first draft's L table's last columns; the status is unchanged)

| Row | Live check | Status | Expected, from the record | Expected by |
| --- | --- | --- | --- | --- |
| L4 | A first plan sign-in from another device through ACP elicitation | open: awaits live evidence | `elicit_first=confirm`, `confirm_text=gobbles`, `authenticate=ok`, `session_kind=chatgpt_plan`, `consent=` the answer given, `host_id_file=present`, `client_record=present`, `client_host_matches_host_id=same`, `ports_bound=none`, `backup_files=0` | Confirmation 2; D2 ("before the browser", "recorded only if the sign-in saves"); D4; D5; D8; 0051-MADR D4 (`remote`) |
| L5 | `gobble auth login openai` on a real terminal, the address pasted | open: awaits live evidence | `login_exit=0`, `session_kind=chatgpt_plan`, `consent=granted` (the owner chooses allow), `host_id_file=present`, `client_record=present`, `client_host_matches_host_id=same` | D2; D4; D5; D8 |
| L6 | `gobble auth logout openai`, then L5 again | open: awaits live evidence | `logout_result=logged-out`, `revoke_warning=no`, `client_record_after_logout=kept`, `host_id_after_logout=same`, `client_reused=same`, `account_second_client=no`, `consent_asked_again=yes` | the PLAN's L6 and its note of 2026-11-12 (0051-PLAN C4); D4; D5 and Q12; SDK-2's probe; Q3 (consents cleared with the session at sign-out) and D2 (asked once per session) |
| L7 | The consent end to end | open: awaits live evidence | `deny_exit=0`, `deny_answer=declined`, `title_updates_after_deny=0`, `allow_exit=0`, `allow_answer=granted`, `allow_stderr_lines=0`, `title_updates_after_allow=1` | D2 (read per request); D3 and its note on a non-interactive caller; the PLAN's L7 and its note of 2026-11-11 |

**Questions for the owner, second draft (2026-11-21).** Eight, each self-contained: the case, today's code, then three options, the recommended one first, each with its cost and the rows it amends. No option contradicts a decided row or an accepted record (0045-MADR D4 and D5, 0051-MADR D5, 0018-MADR D7, archtest rule 4, the owner's ruling of 2026-11-16 on `SessionKindOf`).
*(2026-11-21, third draft: these questions are superseded, unasked, by the third draft's Q1 to Q8 below. The sentence above was untrue for Q1 (b), which amends 0045-MADR D4.)*

* **Q1 (M45, M56). Which commands run the migration.** D9 says "the first … process that resolves credentials"; the PLAN's P5 note puts the call in `run` before `kctx.Run`, for every command. 0045-MADR D4 says `gobble auth status` writes nothing to `auth.json`, and `gobble auth check` reads only. A command that has work waits up to 30 s for `auth.json.lock`, and a consent carry up to 60 s more for the provider's refresh lock (`internal/auth/logins.go:15`). With nothing to do, no lock is taken (M41), and the log opens only at a record (M55). In every option, a command that waits for a lock at a terminal lets text typed meanwhile reach its first prompt, since 0063 flushes only after a sign-in or the first run. (a) *Recommended:* every command in `run` but completion mode (M44), `auth status` and `auth check`, which read the file as it is; the first other command migrates. Cost: a host where only magic-cli-remote polls `auth status --json` stays unmigrated until another command runs, the dormant keys with it; one exemption list in `run`; M48's test runs another command. Amends M45, M56, M48's test. (b) Every command in `run`, `auth status` and `auth check` included, and 0045-MADR D4 amended by a dated note (and this record's "Records amended or superseded in part") to "writes nothing to `auth.json` but the one-time migration". Cost: a status poll that meets work can wait 30 s plus 60 s; an accepted record amended. Amends M45, M56, 0045-MADR D4. (c) Only where credentials are used or changed: `gobble acp`'s start (M46), the resolver's build for a run that sends a request (chat, print), and each command that changes `auth.json` (`auth login`, `logout`, `consent`, `import`); not `auth status`, `auth check`, `version` or `config`. Cost: several call sites, and a later reader that forgets the call meets an unmigrated file. Amends M45, M56, M48's test.
* **Q2 (M12, M29). How the migration tells the openai plan entry and whether its consent is answered, and when it reads `settings.yaml`.** The kind is read only in `provider.SessionKindOf`, by the owner's ruling of 2026-11-16, "Keep it, and make it the single place the kind is read", and D2's note says the consent id is never a constant of gobble's, so `internal/auth` neither infers the kind nor names the id. An entry a build of the archived work wrote has no `kind`. `internal/auth` already imports `llm/provider` (`internal/auth/logins.go:9`). (a) *Recommended:* a second hook, `MigrateOptions.PlanSession func(ctx context.Context) (plan, answered bool, err error)`, which `internal/cli` builds from `llm/provider` (`SessionKindOf`, `Consents`); `Settings` asked only when the plan session is unanswered, so an unreadable `settings.yaml` stops only a migration that has a consent to carry. Cost: one more field in P5 step 1's signature; D9's note of 2026-11-20 annotated (done, by this draft). Amends M12, M29. (b) `internal/auth` calls `llm/provider`'s `SessionKindOf` and `Consents` itself; `Settings` as in (a). Cost: D9 step 5's "it calls the SDK only through that hook" annotated as no longer true, and the PLAN's source check that the migration calls only its hooks dropped. Amends M12, M29, D9 step 5. (c) One injected interface, `provider.PlanConsent` (`llm/provider/consent.go:115-130`) and `provider.Consents` behind it, which `internal/cli` passes in; `Settings` as in (a). Cost: a larger seam (an interface type the migration owns) for the same facts as (a). Amends M12, M29.
* **Q3 (M15, M16, M18, M19, M57, M58). The host id and the client record when an id is missing, malformed or disagrees, or `host-id` is empty.** No code today. D9 step 4 assumes both ids and no `host-id`; `HostID` creates the file `O_EXCL` before its first byte, so an interrupted create leaves it empty, and `HostID` then fails on every call. (a) *Recommended:* the entry's `hostId` adopted only when it passes `readHostID`'s rule (a `urn:uuid:`), else nothing adopted and an error; `host-id` written to a temporary file and linked into place with `os.Link`, which fails if the name exists, by `Migrate` and by `HostID` alike, so no empty file is ever seen, and an empty one already there counts as absent; a client record only when the entry holds both ids and its `hostId` equals the host's id after step 4; a `host-id` holding anything but a `urn:uuid` is an error, and nothing is adopted. Cost: a new helper and a change to `HostID`'s create (D4 and D9 step 4 annotated); an entry missing an id, or registered under another host id, registers a new client at the next plan sign-in. Amends M15, M16, M18, M19, M57, M58, M43's plant. (b) Whatever is present adopted once it passes the `urn:uuid` rule, a record with an empty member included, and the entry's `hostId` in the record even when `host-id` differs; an empty `host-id` treated as absent and replaced. Cost: `clients.json` can name a host id `host-id` does not hold, and whether the SDK reuses such a client is not known; replacing an empty file races `HostID`'s create; D9 step 4 annotated. Amends the same rows. (c) Only a complete, valid entry adopted, into a host with neither file; anything else adopts nothing, with no error; an empty `host-id` left as it is. Cost: more new registrations; a bad `host-id` unreported; one kill at the wrong instant breaks the plan sign-in until the user deletes the file. Amends the same rows.
* **Q4 (M36, M59, M60, M64). The backup, and what a migration leaves as evidence.** No code today. D9 step 2 backs up whenever step 3 or 5 will change `auth.json`, so each retry of a failing carry or write makes another backup holding the secrets, and none is removed (O2); the Info record of a dropped key is lost at `--log-level warn` or `error`, or with no log file. In every option the backup is written through a temporary file that is removed on any failure, so no partial backup is left (M35, M37). (a) *Recommended:* the temporary file linked to the backup's name with `os.Link`, which fails if the name exists (then `Migrate` returns an error and writes nothing); a backup only when step 3 will change `auth.json`, none for a carry alone, which `gobble auth consent` can undo; the lost Info record accepted and said in D9, the backup and `Report` being the evidence. Cost: a warning on the rare collision; D9 steps 2 and 3 annotated; at those log levels a dropped key's only trace is its backup. Amends M36, M59, M60, M64. (b) A counter suffix on a collision; a backup at every run that changes `auth.json`, a carry included, but none when an existing backup holds the same bytes; the dropped-key record logged at Warn. Cost: the name's grammar grows beyond D9 step 2; each run compares with the existing backups; a stderr line at every terminal run that drops a key (M55's commands excepted). Amends the same rows. (c) A file of the name replaced; a backup at every such run, as D9 step 2 reads; the dropped-key record always written to the log file at Info by a handler of its own. Cost: an earlier backup of other bytes can be lost; secret-bearing backups pile up without bound; a special log handler. Amends the same rows.
* **Q5 (M9, M10, M61). What "byte for byte" keeps when step 3 and step 5 write.** D9 step 6 keeps every other member and field "byte for byte (`member.value`)". `merge` compacts an edited entry, and `encode` lays out the top level with LF, two spaces and a final newline, and re-indents a member on one line, on every write gobble makes, step 5's through `provider.SetConsent` included (`internal/auth/store.go:381-396`). (a) *Recommended:* step 6 keeps members' bytes, not white space: step 3's own write keeps each unchanged member's bytes, a one-line member included, lays the edited entry out as gobble's writes do, and writes the top level as `encode` does, so a CRLF file or another indentation is normalised once; step 5's write is gobble's ordinary write. Cost: a hand-compacted member is laid out again when a consent is carried, its values unchanged; D9 step 6 annotated. Amends M9, M10, M61, and L8's comparison (by member). (b) As (a), and `encode` changed to keep every member as read and the file's own line ends and final newline. Cost: a change to every write of `auth.json` gobble makes, beyond D9. Amends the same rows. (c) Step 3 uses gobble's ordinary write, `encode`, and step 6 reads "values kept". Cost: a one-line member is re-indented by step 3 itself; D9 step 6 annotated. Amends the same rows.
* **Q6 (M33, M62). When step 5 may carry the consent.** `chatgptPlanBackground: true` stays in `settings.yaml` (M31), so step 5 fires at any later start that finds an unanswered plan session (M62); and the lock is released before `CarryConsent`, whose `provider.SetConsent` writes whatever the session holds, so a `declined` another process records in between becomes `granted` (M33). The SDK's `auth.SetConsent` has no conditional form, and `filelock` is not re-entrant. (a) *Recommended:* carry only into a plan entry that has no `kind` member (only a build of the archived work wrote the setting and such a session; `v0.6.0` had no plan method, 0055-MADR F5; testing that the member is present is not reading the kind), and accept M33's window, said in D9 (the user withdraws a grant with `gobble auth consent openai background deny`). Cost: one presence check; M24 and M31 annotated; the window stays, two calls wide, in one start. Amends M24, M31, M33, M62. (b) A "migrated" marker in the state directory, after which step 5 never runs again, and `CarryConsent` reading the answer again just before it writes. Cost: a third state file to back up and explain; the window narrowed, not closed. Amends the same rows. (c) Step 5 at any start, as D9 reads, with a conditional set added to the SDK (answer only if unanswered, under the refresh lock), which closes M33. Cost: a consent never asked can be granted at any later start while the key stays; a go-llmprovider-sdk record and release before P5. Amends M33, M62.
* **Q7 (M63). A symbolic link at `auth.json`.** Today the read follows the link and the write replaces it with a plain file, leaving the dormant key in the target; where the backup goes is not decided. (a) *Recommended:* the path resolved with `filepath.EvalSymlinks`; the target read, backed up beside the target, and written. Cost: the migration then differs from gobble's ordinary writes, which replace a link (a follow-up record for those). Amends M63, and settles M34's "beside `auth.json`" for a link. (b) Migrated as today: the link replaced by a plain file, the backup beside the link, and one log record saying so. Cost: the dormant secret stays in the link's target. Amends M63. (c) A symlinked `auth.json` refused: `Migrate` returns an error (M48, M55) and writes nothing. Cost: the key stays until the user acts, with a warning at every migrating start. Amends M63.
* **Q8 (A8). Evidence of the migration on the owner's real file before the owner's first build carrying P5 runs at home.** Any build carrying P5 migrates (M54), not only `v0.7.0`. Confirmation 6 uses copies of real-shaped files, and no agent reads the owner's files (C10). (a) *Recommended:* an owner-run L8, right after P5's commit and before any P5 build runs on the owner's own home: the owner copies their own `auth.json` and global `settings.yaml` into a scratch home and runs P5's build there twice; the kit reports fixed words only (backup made and owner-only, keys dropped by count, state files written, the second run wrote nothing, every other member unchanged by comparison). Cost: one owner step, timed before any other use of a P5 build at home; no agent sees the file. Amends A8; adds L8 to P6's live checks. (b) No live check: Confirmation 6's copies stand, and the backup is the recovery. Cost: the first run on the real shapes is the owner's first P5 build at home. Amends A8. (c) The owner inspects the backup and the file after that first run. Cost: the check comes after the file is written. Amends A8.

#### Third draft (2026-11-21, after the coordinator's delta check)

*(2026-11-21, a dated addition; records only.)* The coordinator's delta check of the second draft found M55 not buildable as worded, a blocker for P5, and fixes to make before the questions reach the owner. The rows of the first and second drafts stand, except where a row below changes or replaces one; the second draft's questions are superseded, unasked, by the eight below.

**Counts, third draft:** 51 decided, 5 out of scope, 21 open (Q1 to Q8), 4 awaiting live evidence; 81 rows.

**Rows replaced or added**

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| M55 | The migration under `auth status --json`, `auth login --json` and `gobble acp` (replacing the second draft's M55, which could not be built: `lazyLog` holds one logger fanning out to the file's handler and stderr's, `internal/cli/main.go:382-396` and `internal/logging/logging.go:44-68`; nothing yields a file-only logger, and a second `logging.Open` would make two rolling writers on one file; the shared logger would put `Migrate`'s Warn on stderr beside M48's line; and `lazyLog.open` writes "no log file" through `Output.Warnf`, `main.go:392`, `:400`, before a command's `Run` discards `out.err`) | decided | Nothing of the migration reaches stderr. Before `Migrate`, the call site applies the command's whole silencing for `auth status --json` and `auth login --json`, read from the parsed command: `out.err = io.Discard` and `logs.stderr = nil`, as those commands do in their `Run` (`internal/cli/auth.go:72-73`, `:522-523`). `lazyLog` gains a file-only view of its one file handler, with no second `logging.Open`, and `Migrate`'s `Log` is that view, so a Warn record reaches the log file and never stderr. Under these two commands and `gobble acp`, a log file that cannot be opened writes nothing to stderr. Every other command is M48: its one warning line, not doubled by the log | 0045-MADR D4; 0051-MADR D5; 0018-MADR D7 (`internal/cli/settings.go:183-191`); `main.go:87-89`; the review's text, decided in a row by the coordinator's disposition of the delta check | proposed: `TestMigrateSilentUnderJSONAndACP`, extended (`internal/cli`, P5) |
| M65 | The `Settings` closure meets a duplicated `chatgptPlanBackground` key in the global `settings.yaml` | decided | Its own parse refuses the file: `(false, false)`, which is M28 | D9's note of 2026-11-20 (`Settings` reads the global file's scalar itself, not through the typed loader; "a file that cannot be read or parsed is `(false, false)`"); decided in a row by the coordinator's disposition of the delta check | proposed: `TestMigrateSettingsIsGlobalOnly`, extended (`internal/cli`, P5) |

**Rows whose text, test or expected values change**

| Row | Change |
| --- | --- |
| M37 | Also: once the backup is in place under its name (by Q4's answer), its temporary name is removed; a temporary file left by a run killed before that is O5. Its test's `failBackup` fires after bytes are written to the temporary file, and the test asserts that neither a backup nor a temporary file remains |
| O5 | Widened: a stray `.auth.json.*.tmp`, or a stray temporary file of the backup, after a run killed inside a write |
| M51 | Its command-level test runs `gobble version`, which prints whatever the context holds, and asserts that its output is absent: `Main` returns 128 plus the signal for any signal cause (`internal/cli/main.go:57-59`), so the exit status alone does not show that the command was skipped |
| M52 | Its test counts the migration's skip line on stderr, not all of stderr |
| M56 | 0045-MADR D4 covers `gobble auth status` only. No accepted record says `gobble auth check` writes nothing to `auth.json`; its exemption in Q1 (a) is this record's proposal |
| L4 to L7 | Each expected list also holds the kit's shared `backup_files=0` and `migration_warning=no` (L4's already held the first) |

**Questions for the owner, third draft (2026-11-21).** Eight, each self-contained: the case, today's code, then three options, the recommended one first, each with its cost and the rows it amends. No option contradicts a decided row. One option amends an accepted record and says so: Q1 (b), 0045-MADR D4, by a dated note. No option contradicts 0045-MADR D5, 0051-MADR D5, 0018-MADR D7, archtest rule 4 or the owner's ruling of 2026-11-16 on `SessionKindOf`.

* **Q1 (M45, M56). Which commands run the migration.** D9 says "the first … process that resolves credentials"; the PLAN's P5 note puts the call in `run` before `kctx.Run`, for every command. 0045-MADR D4 says `gobble auth status` writes nothing to `auth.json`; no accepted record says so of `gobble auth check`, so exempting it is this record's proposal. A command that has work waits up to 30 s for `auth.json.lock`, and a consent carry up to 60 s more for the provider's refresh lock (`internal/auth/logins.go:15`). With nothing to do, no lock is taken (M41), and the log opens only at a record (M55). In every option, a command that waits for a lock at a terminal lets text typed meanwhile reach its first prompt, since 0063 flushes only after a sign-in or the first run. (a) *Recommended:* every command in `run` but completion mode (M44), `auth status` and `auth check`, which read the file as it is; the first other command migrates. Cost: a host where only magic-cli-remote runs `auth status --json` or `auth check --json` stays unmigrated until another command runs, the dormant keys with it; one exemption list in `run`; M48's test runs another command. Amends M45, M56, M48's test. (b) Every command in `run`, `auth status` and `auth check` included, and 0045-MADR D4 amended by a dated note (and this record's "Records amended or superseded in part") to "writes nothing to `auth.json` but the one-time migration". Cost: a status poll that meets work can wait 30 s plus 60 s; `auth check --json`, which magic-cli-remote runs once per provider, concurrently (0045-MADR's option D, `0045-MADR-auth-status-json.md:69`), would migrate in those runs and print M48's stderr line there, since `auth check --json` does not silence stderr; an accepted record amended. Amends M45, M56, 0045-MADR D4. (c) Only where credentials are used or changed: `gobble acp`'s start (M46), the resolver's build for a run that sends a request (chat, print), and each command that changes `auth.json` (`auth login`, `logout`, `consent`, `import`); not `auth status`, `auth check`, `version` or `config`. Cost: several call sites, and a later reader that forgets the call meets an unmigrated file. Amends M45, M56, M48's test.
* **Q2 (M12, M29). How the migration tells the openai plan entry and whether its consent is answered, and when it reads `settings.yaml`.** The kind is read only in `provider.SessionKindOf`, by the owner's ruling of 2026-11-16, "Keep it, and make it the single place the kind is read", and D2's note says the consent id is never a constant of gobble's, so `internal/auth` neither infers the kind nor names the id. An entry a build of the archived work wrote has no `kind`. `internal/auth` already imports `llm/provider` (`internal/auth/logins.go:9`). (a) *Recommended:* a second hook, `MigrateOptions.PlanSession func(ctx context.Context) (plan, answered bool, err error)`, which `internal/cli` builds from `llm/provider` (`SessionKindOf`, `Consents`); `Settings` asked only when the plan session is unanswered, so an unreadable `settings.yaml` stops only a migration that has a consent to carry. Cost: one more field in P5 step 1's signature; D9's note of 2026-11-20 annotated (done, by the second draft). Amends M12, M29. (b) `internal/auth` calls `llm/provider`'s `SessionKindOf` and `Consents` itself; `Settings` as in (a). Cost: D9 step 5's "it calls the SDK only through that hook" annotated as no longer true, and the PLAN's source check that the migration calls only its hooks dropped. Amends M12, M29, D9 step 5. (c) One injected interface, `provider.PlanConsent` (`llm/provider/consent.go:115-130`) and `provider.Consents` behind it, which `internal/cli` passes in; `Settings` as in (a). Cost: a larger seam (an interface type the migration owns) for the same facts as (a). Amends M12, M29.
* **Q3 (M15, M16, M18, M19, M57, M58). The host id and the client record when an id is missing, malformed or disagrees, or `host-id` is empty.** No code today. D9 step 4 assumes both ids and no `host-id`; `HostID` creates the file `O_EXCL` before its first byte, so an interrupted create leaves it empty, and `HostID` then fails on every call. (a) *Recommended:* the entry's `hostId` adopted only when it passes `readHostID`'s rule (a `urn:uuid:`), else nothing adopted and an error; `host-id` written to a temporary file and moved into place under a lock file, `host-id.lock` (`filelock`), by `Migrate` and by `HostID` alike: the process holding the lock checks that the name is still absent or empty and only then renames its file there, so no empty file is ever seen, a leftover empty one is replaced, and two processes cannot each remove the other's new file (the second finds the first's id under the lock and reads it); a rename needs no hard link, so volumes without them (FAT, exFAT, some network shares), where `O_EXCL` works today, keep working; a client record only when the entry holds both ids and its `hostId` equals the host's id after step 4; a `host-id` holding anything but a `urn:uuid` is an error, and nothing is adopted. Cost: a new helper, a third lock file in the state directory, and a change to `HostID`'s create (D4 and D9 step 4 annotated); an entry missing an id, or registered under another host id, registers a new client at the next plan sign-in. Amends M15, M16, M18, M19, M57, M58, M43's plant. (b) Whatever is present adopted once it passes the `urn:uuid` rule, a record with an empty member included, and the entry's `hostId` in the record even when `host-id` differs; an empty `host-id` treated as absent and replaced with no lock. Cost: `clients.json` can name a host id `host-id` does not hold, and whether the SDK reuses such a client is not known; replacing an empty file with no lock races `HostID`'s create; D9 step 4 annotated. Amends the same rows. (c) Only a complete, valid entry adopted, into a host with neither file; anything else adopts nothing, with no error; an empty `host-id` left as it is. Cost: more new registrations; a bad `host-id` unreported; one kill at the wrong instant breaks the plan sign-in until the user deletes the file. Amends the same rows.
* **Q4 (M36, M59, M60, M64). The backup, and what a migration leaves as evidence.** This question bundles four choices, since splitting it would make more than eight questions: what happens when the backup's name is taken (M36), whether a carry alone is backed up (M64), whether retries add backups (M59), and what becomes of the dropped-key record that the log level or a missing log file loses (M60). No code today. D9 step 2 backs up whenever step 3 or 5 will change `auth.json`, so each retry of a failing carry or write makes another backup holding the secrets, and none is removed (O2). In every option the backup is written through a temporary file that is removed on any failure, so no partial backup is left (M35, M37), and it is made while step 2 holds `auth.json.lock`. (a) *Recommended:* the temporary file renamed to the backup's name only after a check, under `auth.json.lock`, that the name is free; if it is taken, `Migrate` returns an error and writes nothing; a backup only when step 3 will change `auth.json`, none for a carry alone, which `gobble auth consent` can undo; the lost Info record accepted and said in D9, the backup and `Report` being the evidence. Cost: a warning on the rare collision; the check excludes other gobble processes, which take the same lock, not other programs; a rename needs no hard link, so volumes without them (FAT, exFAT, some network shares) work, where the review's `os.Link` would fail there at every start; D9 steps 2 and 3 annotated; at those log levels a dropped key's only trace is its backup. Amends M36, M59, M60, M64. (b) A counter suffix on a collision; a backup at every run that changes `auth.json`, a carry included, but none when an existing backup holds the same bytes; the dropped-key record logged at Warn. Cost: the name's grammar grows beyond D9 step 2; each run compares with the existing backups; a stderr line at every terminal run that drops a key (M55's commands excepted). Amends the same rows. (c) A file of the name replaced; a backup at every such run, as D9 step 2 reads; the dropped-key record always written to the log file at Info by a handler of its own. Cost: an earlier backup of other bytes can be lost; secret-bearing backups pile up without bound; a special log handler. Amends the same rows.
* **Q5 (M9, M10, M61). What "byte for byte" keeps when step 3 and step 5 write.** D9 step 6 keeps every other member and field "byte for byte (`member.value`)". `merge` compacts an edited entry, and `encode` lays out the top level with LF, two spaces and a final newline, and re-indents a member on one line, on every write gobble makes, step 5's through `provider.SetConsent` included (`internal/auth/store.go:381-396`). (a) *Recommended:* step 6 keeps members' bytes, not white space: step 3's own write keeps each unchanged member's bytes, a one-line member included, lays the edited entry out as gobble's writes do, and writes the top level as `encode` does, so a CRLF file or another indentation is normalised once; step 5's write is gobble's ordinary write. Cost: a hand-compacted member is laid out again when a consent is carried, its values unchanged; D9 step 6 annotated. Amends M9, M10, M61, and L8's comparison (by member). (b) As (a), and `encode` changed to keep every member as read and the file's own line ends and final newline. Cost: a change to every write of `auth.json` gobble makes, beyond D9. Amends the same rows. (c) Step 3 uses gobble's ordinary write, `encode`, and step 6 reads "values kept". Cost: a one-line member is re-indented by step 3 itself; D9 step 6 annotated. Amends the same rows.
* **Q6 (M33, M62). When step 5 may carry the consent.** `chatgptPlanBackground: true` stays in `settings.yaml` (M31), so step 5 fires at any later start that finds an unanswered plan session (M62); and the lock is released before `CarryConsent`, whose `provider.SetConsent` writes whatever the session holds, so a `declined` another process records in between becomes `granted` (M33). The SDK's `auth.SetConsent` has no conditional form, and `filelock` is not re-entrant. (a) *Recommended:* carry only into a plan entry that has no `kind` member (only a build of the archived work wrote the setting and such a session; `v0.6.0` had no plan method, 0055-MADR F5; testing that the member is present is not reading the kind), and accept M33's window, said in D9 (the user withdraws a grant with `gobble auth consent openai background deny`). Cost: one presence check; M24 and M31 annotated; the window stays, two calls wide, in one start. Amends M24, M31, M33, M62. (b) A "migrated" marker in the state directory, after which step 5 never runs again, and `CarryConsent` reading the answer again just before it writes. Cost: a third state file to back up and explain; the window narrowed, not closed. Amends the same rows. (c) Step 5 at any start, as D9 reads, with a conditional set added to the SDK (answer only if unanswered, under the refresh lock), which closes M33. Cost: a consent never asked can be granted at any later start while the key stays; a go-llmprovider-sdk record and release before P5. Amends M33, M62.
* **Q7 (M63). A symbolic link at `auth.json`.** Today the read follows the link and the write replaces it with a plain file, leaving the dormant key in the target; where the backup goes is not decided. (a) *Recommended:* the path resolved with `filepath.EvalSymlinks`; the target read, backed up beside the target, and written; the lock stays at the given path (`path+".lock"`, `internal/auth/store.go:362`), so it still excludes every other gobble writer, which locks that same path. Cost: the migration then differs from gobble's ordinary writes, which replace a link (a follow-up record for those). Amends M63, and settles M34's "beside `auth.json`" for a link. (b) Migrated as today: the link replaced by a plain file, the backup beside the link, and one log record saying so. Cost: the dormant secret stays in the link's target. Amends M63. (c) A symlinked `auth.json` refused: `Migrate` returns an error (M48, M55) and writes nothing. Cost: the key stays until the user acts, with a warning at every migrating start. Amends M63.
* **Q8 (A8). Evidence of the migration on the owner's real file before the owner's first build carrying P5 runs at home.** Any build carrying P5 migrates (M54), not only `v0.7.0`. Confirmation 6 uses copies of real-shaped files, and no agent reads the owner's files (C10). (a) *Recommended:* an owner-run L8, right after P5's commit and before any P5 build runs on the owner's own home: the owner copies their own `auth.json`, global `settings.yaml` and state files (`host-id`, `clients.json`, where present) into a scratch home and runs P5's build there twice; the kit compares only in memory and reports fixed words only (backup made and owner-only, keys dropped by count, state files written or left, the second run wrote nothing, every other member unchanged). Cost: one owner step, timed before any other use of a P5 build at home; no agent sees the files. Amends A8; adds L8 to P6's live checks. (b) No live check: Confirmation 6's copies stand, and the backup is the recovery. Cost: the first run on the real shapes is the owner's first P5 build at home. Amends A8. (c) The owner inspects the backup and the files after that first run. Cost: the check comes after the file is written. Amends A8.

#### The owner's answers (2026-11-21)

*(2026-11-21, a dated amendment; records only.)* The owner answered the third draft's Q1 to Q8 on 2026-11-21, each as recommended, relayed by the coordinator. Each answer is the option (a) of its question, as the third draft writes it; for Q4, all four of its bundled choices.

| Question | The owner's answer | Option |
| --- | --- | --- |
| Q1 | "All but completion, status, check (Recommended)" | (a) |
| Q2 | "A second hook, PlanSession (Recommended)" | (a) |
| Q3 | "Validate; lock-and-rename host-id (Recommended)" | (a) |
| Q4 | "Check-then-rename; back up key drops only (Recommended)" | (a), its four choices |
| Q5 | "Keep members' bytes, not whitespace (Recommended)" | (a) |
| Q6 | "Only into a kind-less entry; accept window (Recommended)" | (a) |
| Q7 | "Resolve it; migrate the target (Recommended)" | (a) |
| Q8 | "An owner-run L8 first (Recommended)" | (a) |

Amended by them: D9's steps (the note of this date under step 6), D4 (its note of this date), and D9's note of 2026-11-20 on the stored `consents` string, whose sentence "`internal/auth` reads the stored string" is replaced by Q2's hook (the second draft's note on it is resolved). 0045-MADR D4 is not amended (Q1 (a)).

**Counts, with the answers:** 72 decided, 5 out of scope, 0 open, 5 awaiting live evidence (L4 to L8); 82 rows. The state table has no open row.

**The latest row for each row the answers decide.** The rows of the drafts stand as written; each row below is the one that holds.

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| M45 | Which commands run the migration | decided | Every command `run` dispatches, except `gobble auth status` and `gobble auth check` (with or without `--json`), which read `auth.json` as it is; completion mode never reaches `run` (M44) | Q1, "All but completion, status, check (Recommended)" | proposed: `TestMigrateRunsExceptStatusAndCheck` (`internal/cli`, P5) |
| M56 | `gobble auth status` and `gobble auth check` | decided | They do not migrate; `auth.json`, the state files and the log are as before them; 0045-MADR D4 stands | Q1 | proposed: `TestMigrateRunsExceptStatusAndCheck` (`internal/cli`, P5) |
| M12 | Telling the openai plan entry and whether its consent is answered | decided | From `MigrateOptions.PlanSession`, a hook `internal/cli` builds from `llm/provider` (`SessionKindOf`, `Consents`): a plan entry with no `kind` member is recognised by the SDK's inference; a foreign or absent entry is no plan; an unreadable `auth.json` is an error. `migrate.go` imports no `llm/provider` and names no consent id | Q2, "A second hook, PlanSession (Recommended)"; the owner's ruling of 2026-11-16 on `SessionKindOf`; D2's note | proposed: `TestPlanSessionHook` (`internal/cli`), `TestMigrateCallsOnlyItsHooks` (`internal/auth`) (P5) |
| M29 | The global `settings.yaml` cannot be read, and there is no unanswered plan session | decided | `Settings` is not asked; the migration goes on (a dormant key is dropped) | Q2 | proposed: `TestMigrateUnreadableSettingsWritesNothing`, extended (P5) |
| M15 | The plan entry holds a valid `hostId` and no `clientId` | decided | `host-id` written from it when absent or empty; no client record | Q3, "Validate; lock-and-rename host-id (Recommended)" | proposed: `TestMigrateAdoptsIDs`, extended (P5) |
| M16 | The plan entry holds `clientId` and no `hostId` | decided | Nothing adopted: no `host-id` from it, no client record | Q3 | proposed: `TestMigrateAdoptsIDs`, extended (P5) |
| M18 | `host-id` holds another valid id than the entry's `hostId` | decided | `host-id` kept; no client record | Q3 | proposed: `TestMigrateAdoptsIDs`, extended (P5) |
| M19 | `host-id` holds something other than a `urn:uuid`, and is not empty | decided | The migration returns an error; nothing is adopted; the file is not replaced | Q3 | proposed: `TestMigrateAdoptsIDs`, extended (P5) |
| M57 | The plan entry's `hostId` is not a `urn:uuid` | decided | The migration returns an error; nothing is adopted | Q3 | proposed: `TestMigrateAdoptsIDs`, extended (P5) |
| M58 | `host-id` exists and is empty | decided | It counts as absent: `HostID` and the migration replace it under `host-id.lock`; two processes keep one id, which both return | Q3; D4's note of this date | proposed: `TestHostIDReplacesEmptyFile`, `TestHostIDLockedReplaceKeepsFirst` (`internal/auth`, P5) |
| M36 | A file of the backup's name already exists | decided | Checked under `auth.json.lock` before the rename: the migration returns an error and writes nothing; the existing file is unchanged; no temporary file remains | Q4, "Check-then-rename; back up key drops only (Recommended)" | proposed: `TestMigrateBackupNameTaken` (P5) |
| M64 | Only step 5 will change `auth.json` | decided | No backup | Q4 | proposed: `TestMigrateCarriesConsent`, extended (P5) |
| M59 | Retries at each start | decided | A failing carry adds no backup (M64). A failing write of `auth.json` (M38) is retried at each start with a backup each time, since step 3 still has work; none is removed (O2) | Q4 | proposed: `TestMigrateCarriesConsent`, extended; `TestMigrateBacksUpFirst`, extended (P5) |
| M60 | The Info record of a dropped key lost at `--log-level warn` or `error`, or with no log file | decided | Accepted and said in D9; `Report` names each provider whose key was dropped and the backup's file name | Q4 | proposed: `TestMigrateDropsDormantKey`, extended (P5) |
| M9 | An unchanged one-line member when step 3 writes | decided | Kept byte for byte | Q5, "Keep members' bytes, not whitespace (Recommended)" | proposed: `TestMigrateKeepsEverythingElse`, extended (P5) |
| M10 | The edited entry's layout; a one-line member when step 5 writes | decided | The edited entry is laid out as gobble's writes do, two spaces deep; step 5's write is gobble's ordinary write, which may re-indent a one-line member | Q5 | proposed: `TestMigrateLayout` (P5) |
| M61 | The top-level layout: CRLF line ends, another indentation, no final newline | decided | Step 3 writes the top level as `encode` does: LF, two spaces, a final newline; every unchanged member's bytes kept | Q5 | proposed: `TestMigrateLayout` (P5) |
| M33 | Another process answers between the migration's check and `CarryConsent` | decided | Accepted, and said in D9: `CarryConsent` runs after `auth.json.lock` is released, and gobble does not guard the window | Q6, "Only into a kind-less entry; accept window (Recommended)" | proposed: `TestMigrateCarriesConsent`, extended (P5) |
| M62 | Step 5 at a later start | decided | The consent is carried only into a plan entry with no `kind` member; an unanswered plan entry that has one is not carried into | Q6 | proposed: `TestMigrateCarriesConsent`, extended (P5) |
| M63 | `auth.json` is a symbolic link | decided | Resolved with `filepath.EvalSymlinks`; the target is read, backed up beside the target and written; the link stays a link; the lock is `auth.json.lock` beside the given path | Q7, "Resolve it; migrate the target (Recommended)" | proposed: `TestMigrateFollowsSymlink` (P5) |
| A8 | Evidence of the migration on the owner's real files before the owner's first build carrying P5 runs at home | decided | L8 runs right after P5's commit and before any build carrying P5 runs on the owner's home; its result is recorded | Q8, "An owner-run L8 first (Recommended)" | the kit's L8 result (P6) |

**Rows the answers change in passing.** M34: a backup only when step 3 will change `auth.json`, beside the target for a link (Q4, Q7). M24: its end-to-end fixture's plan entry has no `kind` member (Q6). M31: the key left in `settings.yaml` carries nothing into an entry that has a `kind` member (Q6). M43: when the migration loses the race, `host-id` holds the other process's id and no client record is written (M18). M48 and M51: their command is `gobble version`, which migrates (Q1). M52 and M55: `auth status --json` does not migrate (Q1), so their `--json` case is `auth login --json`.

**L8, added to the owner's live checks**

| Row | Live check | Status | Expected, from the record | Expected by |
| --- | --- | --- | --- | --- |
| L8 | The migration run on copies of the owner's own `auth.json`, global `settings.yaml`, `host-id` and `clients.json` (where present), in a scratch home, by the owner, twice | open: awaits live evidence | `home_is_scratch=yes`; `run1_exit=0`; `keys_dropped=` a count (recorded); `backup_made=yes` when that count is above 0, else `no`; `backup_owner_only=yes` when one is made; `host_id`, `client_record` and `consent_carried` each `written`, `left` or `none` (recorded); `run2_exit=0`; `run2_wrote=nothing`; `run2_backup=no`; `other_members_unchanged=yes`; `migration_warning=no`; `scratch_home_deleted=yes` | Q8; D9 as amended this date; M8, M11; Q4 (a backup only for a dropped key) |

#### Amendment of 2026-11-22: two rows from P5's first build pass

*(2026-11-22, a dated amendment.)* P5's first pass found two cases no row decided (the PLAN's "Deviation 5 (2026-11-22)"). The owner answered both, relayed by the coordinator: **"Command reports it; log keeps it (Recommended)"** and **"Guard it in P5 (Recommended)"**. **Counts, with them:** 74 decided, 5 out of scope, 0 open, 5 awaiting live evidence (L4 to L8); 84 rows.

| Row | Input or state | Status | Outcome (what a test checks) | Decided by | Test (PLAN phase) |
| --- | --- | --- | --- | --- | --- |
| M66 | An `auth.json` that cannot be read or parsed (M3), under a command that migrates | decided | `Migrate` returns an error marked `auth.ErrUnreadable` and writes nothing (M3 stands). The call site writes it as one Warn record to the log's file-only view and nothing to stderr, whatever the command. A command that reads `auth.json` reports the file's error itself: D3's one error line for `auth consent`, the resolver's warning, `auth status --json`'s error object. A command that never reads the file, such as `gobble version`, prints nothing of it; the log keeps it | the owner, 2026-11-22, "Command reports it; log keeps it (Recommended)"; D3's note ("Every error is one line on stderr") | `TestMigrateUnreadableAuthJSONIsTheCommands` (`internal/cli`); `TestMigrateUnreadableFileWritesNothing`, extended (`internal/auth`); `TestAuthConsentCommand` (existing, unchanged) (P5) |
| M67 | Another process saves `clients.json`'s `openai` record between step 4's check and its save | decided | The record is saved only when, under `clients.json.lock`, the file still has no `openai` record. Another process's record is kept, and `Report.ClientRecord` is false | the owner, 2026-11-22, "Guard it in P5 (Recommended)"; D9's note of 2026-11-20 ("an existing client record is left") | `TestMigrateClientRaceKeepsFirst` (`internal/auth`, P5) |

**M48, annotated (2026-11-22):** "An error from `Migrate` in any other command" now excludes M66's case, an `auth.json` that cannot be read or parsed, which goes to the log file only. Every other migration error stays one stderr warning line.

**Rows whose test or plant differs from the tables, as built in P5 (2026-11-22, annotations; the PLAN's "P5 record (2026-11-22), pass 1" has the evidence):**

* **M43:** `migrateHooks.beforeHostIDCreate` fires inside the placement, after its first read finds no id and before `host-id.lock`. With it there, `TestMigrateHostIDRaceKeepsFirst` catches the plant "the rename made without the read again under `host-id.lock`".
* **M55:** the plant "`Migrate`'s `Log` given the shared logger" was built as the call site's Warn record through the shared logger, since `Migrate` writes only Info records. `TestMigrateErrorIsAWarning` catches it; `TestMigrateSilentUnderJSONAndACP` does not, since the log there has no stderr.
* **M49:** the plant "limited to `provider.KeyProviders()`" is a fixed list of providers, since `migrate.go` imports no `llm/provider`.
* **M30:** the plant "the closure reading the merged settings" is the closure reading the project's `settings.yaml`, since the typed loader ignores the key.
* **M24:** `TestMigrateCarriesConsent` checks "the session reads granted" through its stub `CarryConsent`. The SDK's own write is checked by `TestMigrateCarriesConsentEndToEnd`.
* **M35, and M14's second plant:** the Windows plants (written without `securePrivate`) ran on a scratch clone on the Windows laptop through `run-on-host`, not through `tools/plant.py`, which runs on this Mac. Both were caught on the owner-only assertion.
* **M14 to M21, M23 and M57's `TestMigrateAdoptsIDs`, and M24 to M64's `TestMigrateCarriesConsent`:** their subtests are top-level helpers (`adoptIDs…`, `carries…`), for the linter's complexity limit. The test and subtest names are as the tables give them.

#### Amendment of 2026-11-22: the live checks L4 to L8 decided

*(2026-11-22, a dated amendment; records only.)* The owner ran the live checks on this Mac, through the PLAN's kits ("P6 kits (2026-11-22)"), and the coordinator relayed their output verbatim. The kit prints fixed words only. This amendment moves rows L4, L5, L6, L7 and L8 from "open: awaits live evidence" to **decided**. It changes no decision, no status and no owner answer; the rows above stay as written, and this entry is the latest word on each. **Counts, with it:** 79 decided, 5 out of scope, 0 open, 0 awaiting live evidence; 84 rows. The status stays `accepted`.

| Row | Status | Evidence (the kit's words) | Decided by |
| --- | --- | --- | --- |
| L4 | decided | Run 1, group `l4-l7`: `elicit_first=confirm`, `confirm_text=gobbles`, `authenticate=ok`, `session_kind=chatgpt_plan`, `consent=granted`, `host_id_file=present`, `client_record=present`, `client_host_matches_host_id=same`, `ports_bound=none`, `backup_files=0`, `migration_warning=no`, `scratch_home_deleted=yes`. Every word is the row's expected one. | Run 1 |
| L5 | decided | Run 1, group `l5-l6`: `login_exit=0`, `session_kind=chatgpt_plan`, `consent=granted`, `host_id_file=present`, `client_record=present`, `client_host_matches_host_id=same`. Run 2 (`--only l5-l6`) printed the same words. Every word is the row's expected one. | Runs 1 and 2 |
| L6 | decided | See "L6, from runs 1 and 2 together" below. | Runs 1 and 2 together |
| L7 | decided | Run 1, group `l4-l7`: `deny_exit=0`, `deny_answer=declined`, `title_updates_after_deny=0`, `allow_exit=0`, `allow_answer=granted`, `allow_stderr_lines=0`, `title_updates_after_allow=1`. Every word is the row's expected one. The title model was named at run time by the owner's answer "You name it at run time (Recommended)" (the PLAN's "P6 kits (2026-11-22)"). | Run 1 |
| L8 | decided | The owner's run of the L8 kit on copies of their own `auth.json` and `settings.yaml`, made by the kit: `home_is_scratch=yes`, `run1_exit=0`, `keys_dropped=0`, `backup_made=no`, `host_id=none`, `client_record=none`, `consent_carried=none`, `run2_exit=0`, `run2_wrote=nothing`, `run2_backup=no`, `other_members_unchanged=yes`, `migration_warning=no`, `scratch_home_deleted=yes`; no `check_failed` line. Every word is the row's expected one. The owner's files held no dormant key, no plan entry with ids to adopt and no consent to carry, and no `host-id` or `clients.json`, so the migration was a no-op on them: the run shows no write, no backup and a second run that wrote nothing, and does not show a backup, a drop, an adoption or a carry on the owner's shapes (the copies of real-shaped files of Confirmation 6 hold those). | The PLAN's "L8 result (2026-11-22)" |

**L6, from runs 1 and 2 together.** Run 1 gave, for `l5-l6`: `logout_result=logged-out`, `revoke_warning=no`, `client_record_after_logout=kept`, `host_id_after_logout=same`, `client_reused=same`, `consent_asked_again=yes`, `backup_files=0`, `migration_warning=no`, `scratch_home_deleted=yes`, and `account_second_client=yes` with `check_failed=account_second_client`; the second sign-in completed (no `login2_exit` failure). The coordinator found run 1's `account_second_client` question **confounded**: each kit group registers its own client, so the account's page lists several gobble entries whatever L6 does. The owner chose "Fix the kit, re-run L5-L6 (Recommended)"; the kit now asks the count of connected apps named gobble after L5's sign-in and after L6's second sign-in, and passes when the count did not grow. Run 2 (`--only l5-l6`) gave every word as in run 1 except `account_second_client=no` (the count did not grow: the client was reused, no second entry) and `check_failed=login2_exit`: the second sign-in ended non-zero after the owner opened the address on another device and pasted the failed loopback address, as designed, and gobble's message was not captured. (A first attempt of run 2 printed `check_failed=usage` before doing anything, since the coordinator's command wrongly passed `--small-model` to the `l5-l6` group; nothing ran.) The row is decided as follows. `logout_result`, `revoke_warning`, `client_record_after_logout`, `host_id_after_logout`, `client_reused` and `consent_asked_again` are decided by both runs, which agree. `account_second_client=no` is decided by run 2 alone, the only run whose question was not confounded. Run 1 shows the second sign-in completing (client reused, no revoke warning, consent asked again). Run 2's `login2_exit` is recorded as a one-off whose cause was not captured; the record does not claim it was explained. Testing ended there on the owner's direction, in the owner's words, quoted exactly: "this is annoying as fuck. it works. why do we keep testing this over and over when we already know it works?"

**What the live checks settle in the rows and the Confirmation.** A8 (decided by Q8) has its evidence: L8 above. Confirmation 2 holds live over ACP (`elicit_first=confirm`, `confirm_text=gobbles`, no port bound); Confirmation 3 holds live for the consent end to end (L7); Confirmation 4 holds live across a logout (`host_id_after_logout=same`, `client_reused=same`; the key switch's test is not built, see the PLAN's "P6 record (2026-11-22)"); Confirmation 6's real-shape evidence is L8, a no-op run, as stated above; Confirmation 8, "the owner's live probes of the PLAN are recorded", is met for L4 to L8 by this amendment. Confirmations 1, 5 and 7 are not settled by live checks and are open in the PLAN's record.

#### Amendment of 2026-11-22: A1 closed, and the Windows flake handed off (the owner's rulings)

*(Dated addition, records only; the owner's rulings of 2026-11-22, relayed by the coordinator.)* Three rulings, each as recommended.

1. **A1's accounting: "Rely on P1's recorded run (Recommended)".** A1's "P1's sort script, run again at P6's start commit" is closed by P1's recorded run (`UNACCOUNTED 0`, the PLAN's "P1 record (2026-11-11)") and each later phase's tests. The script was a scratch script of P1 and no copy of it exists; it is not re-run and not rebuilt. What that does not prove is what P1's record already says it does not prove: that a group in the tree is in the right place.
2. **A1's grep: "No hit outside the migration (Recommended)".** A1 now reads: the whole-word grep `grep -rnw -e chatgptPlanBackground -e askPlanBackground -e planNotice --include=*.go .` has no hit outside the migration's three files, `internal/auth/migrate.go`, `internal/cli/migrate.go` and `internal/cli/migrate_test.go`. P5's migration (D9) must read the retired setting by its name to carry its value, so the grep cannot exit 1 any more; the evidence is P6 pass 2's run at `894b6d9`: exit 0 over the tree, and exit 1 once those three files are excluded (the PLAN's "P6 record, pass 2 (2026-11-22)", item 2).
3. **The Windows flake: "Investigate under its own record (Recommended)".** The test is `TestSignInTypeAheadNotSentAsPrompt` (`internal/cli/login_test.go:856`, 0063's). It failed once on the Windows laptop in P5's first suite run and once in P6 pass 2's run 1, each with `auth.json: ... The process cannot access the file because it is being used by another process`, and passed on the next run each time (P5 also passed `-count=20`). It is a known Windows-only flake, outside this record's scope, handed to a separate future record. It is not fixed here and no 0052 acceptance item waits on it.

### Consequences

* Good, because the plan sign-in ships with one disclosure on every surface, asked once, withdrawn from the next
  request, and gated twice.
* Good, because a host keeps one id, and a sign-out or a switch to a key stops registering new clients (if SDK-2's
  probe passes).
* Good, because `auth status --json` tells a client what kind of session it is and what the user allowed.
* Good, because the owner's dormant secrets leave `auth.json`, kept in an owner-only backup.
* Neutral, because the plan sign-in reaches users with the redesign, not before it (Q6's cost).
* ~~Bad, because the uncommitted work sits in the working tree until P1 sorts it: a risk of loss the PLAN's first step
  lowers with a patch file.~~ *(2026-11-11: no longer so. The PLAN's Deviation 1 committed the work whole as
  `1406bd6`, the branch is on origin, and `git diff --binary d6c10a5 1406bd6` gives the archived patch byte for
  byte (the PLAN's note under P1). What remains of this cost: the parts D1 keeps "as is" are off `main` until P1
  re-applies them from the branch.)*
* Bad, because gobble writes two new files in the state directory, each one more thing to back up and to explain.
* Bad, because the migration writes the owner's real `auth.json` once.

### Confirmation

1. Every WIP part of D1's table is where the table says, checked hunk by hunk against the P1 patch.
   *(2026-11-22: met by P1's recorded sort run and each later phase's tests, and by the grep as the owner amended it; see "Amendment of 2026-11-22: A1 closed, and the Windows flake handed off" under Decision Outcome.)*
2. A first plan sign-in on each surface (`auth login`, `/login`, setup, first run, `acp --login`, ACP `authenticate`)
   shows gobble's wording before any `open_url`, against `planIssuer`; `--json` refuses it with exit 2.
3. A "no" then a "yes" through `gobble auth consent` changes the next background request, in a second process too.
   *(2026-11-11, later: run with no terminal and the consent given by its id; and a turn refused for want of the
   consent ends with `ErrorData` of kind `not_permitted` and `consent` `chatgpt_plan.background_use`, one refused
   because the answer could not be read with kind `not_permitted` and no `consent`.)*
4. The host id is the same across two plan sign-ins, a logout and a key switch; the issued client is reused after a
   logout (offline, against `planIssuer`).
5. `auth status --json` for a foreign openai session is `usable: false`, `reason: "oauth_unused"`; for a plan session it
   has `sessionKind` and `consents`; the golden file changes only by those members.
   *(2026-11-11, later: and `consentTerms`, whose `text` is gobble's wording and whose link is the usage page.)*
6. The migration, run twice on copies of real-shaped files, writes once, backs up first with owner-only permissions,
   logs one line per dropped key, and keeps every other byte.
7. Each new test is seen failing on its planted break; `make preflight` and `make race` pass on three hosts.
8. The owner's live probes of the PLAN are recorded.
   *(2026-11-22: L4 to L8 are recorded, as fixed words, in "Amendment of 2026-11-22: the live checks L4 to L8 decided" under Decision Outcome. L6's second sign-in ended non-zero once, in run 2, with its cause not captured; the amendment says how the row was decided.)*

## Pros and Cons of the Options

### A. Fold the work onto the SDK engine

* Good, because nothing is built twice: the CLI half is replaced once, by the design that lasts.
* Good, because the work's tested storage half is kept.
* Bad, because the plan sign-in waits for the SDK's minor and 0051.

### B. Fix, then land, then migrate

* Good, because the plan reaches users soonest, and the uncommitted work is safe in a commit.
* Bad, because its CLI half, the notice placement and the setting would ship, and then be migrated again.
* Bad, because the owner chose against it (Q6).

### C. Discard and start again

* Good, because nothing of the WIP's per-id code survives by accident.
* Bad, because its storage half and its tests are right, and would be written again.

## More Information

### Records amended or superseded in part, applied by the 0052-PLAN's P0 on acceptance

*(2026-11-11: applied by the 0052-PLAN's P0. The 0031 paste and 0049-PLAN's P5b split were already on `main`, in
`86ad4a7` and `97f13a5` (the PLAN's Deviation 1), and were checked, not made again; the notes in 0025-MADR, 0045-MADR
and 0005-MADR were added, each dated and beside the text it amends.)*

* **`0031-MADR-sdk-current-releases-and-adoption.md`, D7** and its amendment of 2026-11-10: the owner pastes the
  text written to the scratch file the PLAN's P0 names (`gobble-0031-amendment.md`) *(2026-11-11: done, committed as
  `86ad4a7` by the PLAN's Deviation 1, step 3)*: P4b-2 folded here; P4b-Q2 (a),
  "Consent is asked once at the plan sign-in and saved as a setting" (`:355` [WIP]), superseded by D2 and D3; the WIP
  terms' "§2.4 consent control: a setting, named in the notice" (`:388-391` [WIP]) superseded by D3, a command named in
  the notice; "The phone sign-in (magic-cli-remote driving `gobble acp --login openai` …)" (`:392` [WIP]) superseded by
  0051-MADR D4; P4b-Q3 (b) as QA answers. D8 is carried out by 0051-MADR D2.
* **`0031-PLAN-sdk-current-releases-and-adoption.md`, P4b-2** (`:307-348`; *2026-11-11: `:307-421` on `main` since `86ad4a7`, its closing note included*): superseded by the 0051 and 0052 PLANs;
  its record and deviation 1 kept as history.
* **`0025-MADR-provider-choice-and-auth.md`, D6** (`:223-232`): "a stable `HostID` in the state directory" built by D4;
  "consent … asked once at the first plan session and saved" refined: on the session (D2).
* **`0045-MADR-auth-status-json.md`:** D2 enforced (D8); D6 admits `sessionKind` and `consents`; D7 unchanged.
* **`0005-MADR-v1-feature-scope.md`,** the OpenAI row (`:168`): "API key, or the ChatGPT plan sign-in under gobble's own
  name (0052-MADR)". `0005-PLAN-v1-feature-scope.md` F4-3d and `0025-PLAN-provider-choice-and-auth.md` P6: done by this
  record's PLAN, noted at its completion.
* **`0049-PLAN-structured-quota-and-rate-limit-errors-for-acp-clients.md`, P5b:** as QA answers.

### Records in other repositories this bears on (not changed here)

* go-llmprovider-sdk `docs/decisions/0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md`: gobble adopts D1 to
  D7; see the differences below.
* magic-cli-remote `docs/decisions/0074-MADR-remote-provider-auth-from-phone.md` D2 ("no vault"): unaffected; Magic
  relays the `confirm` step and stores no answer.
* *(2026-11-11, later.)* magic-cli-remote
  `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`: the owner chose
  that Magic both reads the consent and can change it, inside that cut-over record. Its daemon reads `gobble auth
  status --json` (`sessionKind`, `consents`, `consentTerms`), shows them on the phone, changes one by running `gobble
  auth consent <provider> <consent> allow|deny` (D3's note), and maps a refused background request, kind
  `not_permitted` with `ErrorData.Consent` (D2's note), to a prompt offering to allow it, showing `consentTerms`' text
  and link first. "No vault" still holds: the answer is stored by gobble on the host, never by Magic.

### Questions answered by the owner (2026-11-10)

The owner chose the recommended option of each question below on 2026-11-10: QA (a), QF (a) and QG (a). The options
are kept as they were put.

* **QA. 0049-PLAN P5b, coupled to P4b-2 by P4b-Q3 (b).** (a) *recommended:* split it; its `llm` and `llm/provider` half
  (`Window`, the reset rule, `planWindow`, their tests and `choice_test.go`'s case), which needs no acpext change, lands
  now at `acpext/v0.3.0` under 0049-PLAN; its wire half (`prompt.go:308`, the `prompt-weekly.error.json` test) lands
  with 0051-PLAN P4's acpext pin; (b) keep Q3 (b): P5b waits for this record's plan sign-in, weeks away; (c) tag an
  `acpext/v0.4.1` with openai's key-only entry back, pin it with P5b whole now, and have Magic take `v0.4.1`, against
  Q14's `v0.4.0`.
* **QF. The consent's control.** (a) *recommended:* retire the setting; `gobble auth consent` changes the answer (D3);
  (b) keep `chatgptPlanBackground` as a view that reads and writes the session; (c) no control but a new sign-in.
* **QG. Where the issued client is kept.** (a) *recommended:* `<state>/clients.json` (D5); (b) a reserved member of
  `auth.json`, which its parser refuses today, since every member must be an object with a string `type`
  (`internal/auth/store.go:62-64`, the check at `:112` [HEAD]; `:66-68`, `:120` [WIP]); (c) the openai entry keeps `clientId` and `hostId` after a logout, as a
  session-less entry the resolver must learn to skip.

### Where the report, the audit or the SDK drafts differ from the code

* **The report's §8.5 and §9.2 assume the setting migrates** (step 2 there). It never shipped (D3); D9 carries a value
  only a build of the uncommitted work could have written.
* **The report's G5 calls the resolver fix "plan-aware"**; the code's rule is broader (0051-MADR D2): a session of any
  kind gobble does not offer is refused, grok's too.
* **The report's G15 asks for "a consent field and the method that produced the credential"** (its §7.3). D8 gives a
  session's kind, not its method. A stored session records its kind (go-llmprovider-sdk
  `0074-MADR-one-sign-in-handle-and-method-facts.md` D5) and not the method that made it: grok's browser and device
  sign-ins yield one kind of session, and the SDK keeps no method id on it. The kind is also what the resolver and the
  policy read (0051-MADR D2). A `method` member would need a new stored field for no reader.
* **go-llmprovider-sdk `0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md`, as first drafted,** said gobble
  "implements `ClientStore` over its own `auth.json`". Under QG (a) it is `<state>/clients.json`; the SDK's interface
  does not care. Corrected there on 2026-11-10.
* **go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D7, as first drafted,** said
  `x/signin` is "given a `ClientStore`" without naming the field, and go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D3's `signin.Options` listed
  neither it nor `HostID` nor `ConsentText`. On 2026-11-10 D7 named `signin.Options.ClientStore`, said the wizard finds
  the store by asserting its `Options.TokenStore`, and go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D3 noted the three fields 0075 adds. D5 above relies
  on the wizard's rule: gobble's `sdkStore`, the wizard's `TokenStore`, implements `auth.ClientStore`.
* **The audit's §2.5 says the setting is read "from the next session"**; at [WIP] it is `ReadSession`
  (`internal/config/keys.go:104`), applied at a session's start (`acpserver/agent.go:496`): the audit is right, and D2
  removes the gap.

### Not verified, with the probe that settles it

* Whether OpenAI accepts the issued client after a revocation with no `id_token_hint` (SDK-2): the SDK's owner-run
  probe, `TestLive_0075ClientReuse` in go-llmprovider-sdk.
  *(2026-11-21: run by the owner on 2026-11-14 and passed, as go-llmprovider-sdk
  `0075-PLAN-chatgpt-plan-consent-host-id-and-client-reuse.md`, "Phase 6, the owner's run (2026-11-14)", records; D9's
  note of this date.)*
* Whether OpenAI regards a plan sign-in driven from the owner's phone, on the owner's own host, as within its terms
  (the report's U12): not a probe; the owner's stance "own host only, disclosed" (Q11) stands.
* Which client Pi's OpenAI login uses (U9; `0005-PLAN-v1-feature-scope.md:4664-4667` against
  `0025-MADR-provider-choice-and-auth.md:233-234`): read Pi's `packages/ai/src/auth/oauth/` sources. 0051-MADR D2's Pi
  rule skips an openai login either way.

### Revisions of 2026-11-11

Corrections the owner directed on 2026-11-11, each checked against the tree first. Every one is a dated note beside the
text it corrects; no decision of D1 to D9 changed but D2's and D3's handling of an unreadable answer, which is new and
marked as such. The record stays `proposed`.

* **The work is committed, and on origin.** It is `1406bd6` on `wip/0031-p4b2-chatgpt-plan`; `main`, at `17d1605`,
  holds none of it (Context; the "[HEAD]" paragraph; the first "Bad" consequence).
* **Anchors.** 0031-PLAN P4b-2 step 4's line is `:347` on `main`, and P4b-2 runs `:307-421`; `signin.go:193` is the
  archive branch's line, `:181` on `main`.
* **The SDK's 0075 and 0074 re-pinned,** with `StoredConsent` and `SessionKind()` ("What the SDK proposes").
* **D2, D3:** a stored answer that cannot be read is not a grant; what gobble shows then; `auth consent`'s exit.
* **D8:** `sessionKind` is `(*OAuthSession).SessionKind()`'s value.
* **More Information:** the 0031 paste is done, at `86ad4a7`.

**Later on 2026-11-11, the owner's ruling relayed by the cross-repository coordinator,** that magic-cli-remote reads
the consent and can change it, inside its 0246 cut-over. Each is a dated note beside the text it changes; the record
stays `proposed`.

* **D2:** a refused background request carries `ErrorData.Consent` with kind `not_permitted`, through
  `llm.ConsentError`; `PlanConsent` returns the consent's id; the unreadable answer is `not_permitted` with no
  `consent`, so a client offers no allow; `Offered` gives gobble's wording as the consent's text, the same as the
  sign-in's `confirm` step.
* **D3:** what `gobble auth consent` gives a non-interactive caller: the SDK's consent id as the argument, gobble's
  word kept as its alias, exit codes, stdout and stderr, and no `--json`.
* **D8:** the members a client reads and their values; `consentTerms`, new; the unreadable store's exit 2; what
  Magic does with them, where the PLAN's rollout said it "reads `sessionKind` and `consents`".
* **Confirmation 3 and 5; "Records in other repositories"; `informed`; `consulted`.**

**Accepted on 2026-11-11,** by the owner's word of that day (the note under Decision Outcome). The status is
`accepted`; no decision and no note changed with it.

### Readiness pass of 2026-11-21

Records only, at `4c3e397`, before P5. Added: D9's note of this date (re-grounding), the "State table for what remains"
with its eight questions for the owner, and a note on the first bullet of "Not verified". No decision, status or owner
answer changed. The PLAN's "Readiness pass for P5 and P6 (2026-11-21)" maps every decided row to its test.

The same day, after an independent review of that first draft, a second draft by dated additions: "Second draft
(2026-11-21, after the independent record review)" under the state table (rows changed and added, the live checks'
expected values, and eight questions in place of the first draft's), and a note on D9's note of 2026-11-20 against D2's
note. The PLAN's "Second draft of the readiness pass (2026-11-21)" holds the seams and the tests.

A third draft the same day, after the coordinator's delta check: "Third draft (2026-11-21, after the coordinator's delta
check)" (M55 replaced, M65 added, eight questions in place of the second draft's) and a map of the first draft's
question numbers; the PLAN's "Third draft of the readiness pass (2026-11-21)" holds the seams and tests it changes.

The owner answered Q1 to Q8 the same day, each as recommended: "The owner's answers (2026-11-21)" under the state
table, with the latest row for each row they decide and L8; dated notes on D4 and on D9's steps. No row is open. The
PLAN's "The owner's answers: P5 and P6 settled (2026-11-21)" holds the steps and tests.

### Close-out of the live checks (2026-11-22)

Records only. L4 to L8 are decided from the owner's runs of the PLAN's kits: "Amendment of 2026-11-22: the live checks L4 to L8 decided" under Decision Outcome, with Confirmation 8's note. No decision, status or owner answer changed. This record carries no Completion section: P6's other steps are open (the PLAN's "P6 record (2026-11-22)").

### P6, pass 2 (2026-11-22)

Records only. A4's key switch now has its test, `TestKeySwitchKeepsHostAndClient`, seen failing on its plant; Confirmation 5 was re-checked and holds; the C6 gates ran at `894b6d9` on three hosts (the PLAN's "P6 record, pass 2 (2026-11-22)" has the outputs). Two findings change no decision and are open: A1's whole-word grep exits 0, not 1, because P5's migration reads the retired setting's name (`internal/auth/migrate.go`, `internal/cli/migrate.go`, `internal/cli/migrate_test.go`), and the A1 sort script of P1 no longer exists; and `TestSignInTypeAheadNotSentAsPrompt` (0063's) failed once on the Windows laptop with a file-in-use error and passed on the second run. This record carries no Completion section.

### Completion (2026-11-22)

Every acceptance criterion of this record's Confirmation 1 to 8 is met, as the PLAN's "P6 record, pass 2 (2026-11-22)" and "P6 record, close-out (2026-11-22)" show: 1 by the owner's ruling on A1; 2, 3 and 5 by their existing tests (item 5 re-checked at `894b6d9`); 4 by the four plan sign-in tests and `TestKeySwitchKeepsHostAndClient`, seen failing on its plant; 6 and 7 by P5's tests and plants and the C6 gates on three hosts at `894b6d9` (one Windows run failed on a known flake and passed on the next, the flake handed to a record of its own); 8 by L4 to L8. One step is not done and is not this record's to do: **the owner's `v0.7.0` tag**, which waits on the SDK tag being adopted and on the acp-go-sdk2 lockstep with magic-cli-remote (0055-MADR D2). There is no `v0.7.0` tag (`git tag -l` lists `v0.6.0` last).

### Related

* PLAN: [0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md).
* [0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md),
  [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md),
  [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md),
  [0045-MADR-auth-status-json.md](0045-MADR-auth-status-json.md),
  [0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md).
