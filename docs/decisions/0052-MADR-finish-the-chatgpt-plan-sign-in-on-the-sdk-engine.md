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

### Related

* PLAN: [0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-PLAN-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md).
* [0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md),
  [0031-MADR-sdk-current-releases-and-adoption.md](0031-MADR-sdk-current-releases-and-adoption.md),
  [0025-MADR-provider-choice-and-auth.md](0025-MADR-provider-choice-and-auth.md),
  [0045-MADR-auth-status-json.md](0045-MADR-auth-status-json.md),
  [0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md).
