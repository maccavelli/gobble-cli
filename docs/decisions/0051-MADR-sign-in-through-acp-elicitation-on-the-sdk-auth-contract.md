---
status: accepted
date: 2026-11-12
decision-makers: repository owner
consulted: "magic-cli-remote's docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md (observed 2026-11-10, committed in that repository as 3e2f784b) and the owner's answers of 2026-11-10 to its Q1-Q22; go-llmprovider-sdk's proposed 0073-MADR-publish-an-auth-contract-module.md, 0074-MADR-one-sign-in-handle-and-method-facts.md, 0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md and 0076-MADR-auth-error-kinds-and-doc-drift.md (untracked in that repository); this repository at d6c10a5 with the owner's uncommitted 0031-PLAN P4b-2 work; acp-go-sdk2 v1.35.0 read from the module cache; magic-cli-remote at 995d442b and its proposed `0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`, whose O2 and O4 the owner answered on 2026-11-10; two independent reviews of these drafts, 2026-11-10; corrected on 2026-11-11 against this repository at 17d1605, go-llmprovider-sdk's four drafts as dated 2026-11-11, acp-go-sdk2 v1.36.0, and magic-cli-remote, whose origin/master is eefb7c25 (995d442b, cited here, is five commits behind it, and its local master is two ahead, bcde74ab and 14200b6e), with the owner's rulings of 2026-11-11 as the cross-repository session relayed them; and, later on 2026-11-11, the owner's rulings relayed by the cross-repository coordinator: go-llmprovider-sdk's 0074-MADR D4 reversed (NoLoopback is v1 API, with gobble its call site), `_meta.gobble.remote` on authenticate, and acpext ErrorData's `consent` member, with go-llmprovider-sdk's 0077-MADR-go-1-27-4-for-go-2026-6859.md, now a file there; and, on 2026-11-12, the owner's direction 'Where possible gobble auth would send gobble as the name/referrer not the sdk', go-llmprovider-sdk's 0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md (accepted 2026-11-12) and its PLAN (in progress), both read as committed there at f50466b, with that repository's code at the same commit, and this repository at c47f4a7; and the owner's rulings of 2026-11-12 on D9's Q1 to Q3, relayed by the cross-repository coordinator"
informed: "magic-cli-remote, which relays gobble's sign-in steps and imports acpext; go-llmprovider-sdk, whose x/signin and authcontract this consumes; maccavelli/gobble-cli#16"
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# Sign in through ACP elicitation on the SDK's auth contract: one policy over the SDK's method facts, the contract's steps on every surface, and one error vocabulary

## Context and Problem Statement

gobble signs in to three providers without an API key (grok, kilo and, in the owner's uncommitted work, openai's ChatGPT
plan). Every fact about those sign-ins is decided by gobble's own per-provider code, its machine-readable output is
device-code-shaped, and no ACP client can drive a sign-in at all. The 0245 report (magic-cli-remote
`docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md`, §5.5, §7.3 G10-G12, §7.4 A1-A3, §8.2)
found this; this record decides gobble's half of the redesign the owner chose on 2026-11-10. The ChatGPT plan itself,
its consent, host id and client, and the folding-in of the uncommitted work are
[0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)'s.

"[HEAD]" is `d6c10a5`, read with `git show HEAD:<path>`; "[WIP]" is the owner's uncommitted 0031-PLAN P4b-2 work as
it stood on 2026-11-10, since archived whole as `1406bd6` on the local branch `wip/0031-p4b2-chatgpt-plan` and read
with `git show 1406bd6:<path>` (0052-PLAN Deviation 1, 2026-11-11). A line cited without a mark is the same at both. acp-go-sdk2 lines are `v1.35.0`'s, the version both pin
(`go.mod:16`), read in the module cache.

*(2026-11-11, corrections.)* `main` is now `17d1605`, six commits of 0053 past `86ad4a7`, so a "[HEAD]" line is read
with `git show d6c10a5:<path>`, not `git show HEAD:<path>`. Of the files this record cites, two changed between
`d6c10a5` and `17d1605`, `llm/provider/provider.go` and `llm/provider/limit_test.go` (both in `97f13a5`), and the lines
cited in them are the same at both. The archive branch is no longer local only: it was pushed on 2026-11-11, and
`git ls-remote --heads origin` lists `wip/0031-p4b2-chatgpt-plan` at `1406bd6`. "The owner's uncommitted work" in this
record names that work as it stood on 2026-11-10; it is committed on that branch and is in no working tree. The
acp-go-sdk2 pin moves to `v1.36.0` before this record's PLAN reaches P2
([0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md));
the note at the end of U8 gives each cited line at that tag.

### What gobble does today

* **Facts are per-id predicates.** `HasLogin` is `id == "grok"` [HEAD] (`llm/provider/login.go:62`), grok or openai
  [WIP] (`:73-75`); `HasKeyLogin` is kilo's (`:66` [HEAD], `:85` [WIP]); [WIP] `HasPiLogin` is grok's (`:81`). `offers`
  (`llm/provider/signin.go:77` [HEAD], `:86-100` [WIP], with `openai := id == …` at `:87`) decides which SDK methods
  gobble offers. `signInWords` (`internal/cli/providerscmd.go:114`) words them, with [WIP] a fifth openai case at
  `:118`. 0031-MADR D8 already decided "`HasLogin` and `HasKeyLogin` read `Descriptor.AuthMethods`"
  (`0031-MADR-sdk-current-releases-and-adoption.md:406` [WIP], `:363` [HEAD]; *2026-11-11: `:446-447` on `main` since `86ad4a7`, whose supersession notes lengthened the section before it*), and its 2026-10-29 note says they "are still hard-coded".
* **`--json` is output-only and device-shaped.** `loginEvent` (`internal/cli/auth.go:158-167`) has five events,
  `device_code`, `browser_url`, `saved`, `error` and [WIP] `notice` (0005-PLAN F4-3c decision 6,
  `0005-PLAN-v1-feature-scope.md:5137-5143`; 0016-MADR A3). With `--json`, `offersPaste` is false
  (`internal/cli/auth.go:227` [HEAD], `:240` [WIP]) and every question returns `errNoTerminal` through `unasked`
  (`:218` [HEAD], `:222` [WIP]), so a browser or plan sign-in cannot finish for a remote driver (report F8, F12,
  F52). `openLoginURL` still opens the host's browser under `--json` (`:257` [HEAD], `:270` [WIP]; F56).
* **ACP `authenticate` is presence-only.** `Authenticate` answers `{}` when `Has(id)`, else `auth_required`
  (`acpserver/auth.go:59-70`). 0016-MADR A2 decided this (quoted under "Records amended"). No ACP client can
  learn a provider's methods, nor answer a paste or a question (gobble audit §5.4).
* **Two failure vocabularies.** `FailureKind` (`llm/provider/list.go:106-133`) names ten words, `rate-limited`,
  `not-permitted`, `empty` among them, for `--json`'s `error` event (`internal/cli/auth.go:84`), log lines
  (`internal/cli/models.go:274`, `:316`) and a provider retry's cause (`llm/provider/provider.go:273` [HEAD]).
  `llm.Kind` (`llm/kind.go:25-56`) names fourteen, which acpext's `ErrorKind` (`acpext/types.go:337-358`) and
  `RetryCause` (`acpext/types.go:27-44`) spell on the wire. 0049-MADR D3 kept `FailureKind` "as it is"
  (`0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md:200-202`).
* **acpext's auth surface is a list and a boolean.** `AuthProvider{ID, Name, Command string; Login bool}`
  (`acpext/auth.go:9-12`). Its doc says "A provider with a sign-in may still take a key: its command offers both"
  (`:6-8`), which is false: `gobble auth login <id>` never shows a menu (`internal/cli/auth.go:192-198` [HEAD]) and
  `gobble acp --login <id>` branches to the sign-in before any key (`internal/cli/acp.go:101-103`) (F19). At
  `acpext/v0.4.0`, openai has `Login: true` (`acpext/auth.go:25`), which magic-cli-remote reads as "device code"
  (issue #16).
* **The SDK may not be imported outside `llm/provider`.** archtest rule 4 refuses any import under
  `github.com/maccavelli/go-llmprovider-sdk` elsewhere (`internal/archtest/check.go:308`), and rule 1 in the core
  packages (`:249`); 0004-MADR rule 4 (`0004-MADR-go-module-architecture.md:475`). The contract module the SDK
  proposes sits under that prefix (go-llmprovider-sdk `0073-MADR-publish-an-auth-contract-module.md` D1).

### What the SDK proposes (go-llmprovider-sdk records, all `proposed`, 2026-11-10)

*(2026-11-11: the four records are still `proposed` and untracked there, and are now dated 2026-11-11, each with a
"Revisions of 2026-11-11" section. The three bullets below are kept as written on 2026-11-10; what they leave out, and
what the owner ruled that day, follows them.)*

* `0073-MADR-publish-an-auth-contract-module.md`: the nested module `github.com/maccavelli/go-llmprovider-sdk/authcontract`,
  standard library only, tagged `authcontract/vX.Y.Z` from `v0.1.0`, with `Method`, `Step`, `Answer`, `Failure`,
  `ErrorKind` (`snake_case` values, constants `FailureCanceled` … `FailureOther`), `StepKind` (`open_url`,
  `show_code`, `paste`, `confirm`, `form`, `waiting`, `done`, `failed`), a schema, golden vectors and additive-only
  rules (D2-D7).
* `0074-MADR-one-sign-in-handle-and-method-facts.md`: comparable facts on `llmprovider.AuthMethod` (`Kind`,
  `Completion`, `Paste`, `FirstPaste`, `Yields`, `ClientIdentity`, `Slot`, `Refreshable`, `Revocable`; D1);
  `signin.Methods(d) []authcontract.Method` (D2); `signin.Start(ctx, p, m, signin.Options) (*signin.SignIn, error)`
  with `Method`, `Steps`, `Answer`, `Wait`, `Cancel` under `llmprovider/x/signin` (D3); `OAuthSession.Kind`
  (D5); the wizard rendering the same steps, and `wizard.Options.Driver` (D6); `signin.ErrorKindOf` and
  `auth.ErrSignInDenied`/`ErrSignInExpired` (D8).
* `0076-MADR-auth-error-kinds-and-doc-drift.md`: `auth.Identity{Name, Version}` on the sign-ins and
  `wizard.Options.Identity` (D3, D4).

*(2026-11-11.)* The same records re-read as dated 2026-11-11, with the owner's rulings of that day, which those records
are being revised to carry. Decisions are cited by id, since their lines are moving:

* **The contract's types** (0073-MADR D2) also hold what the first bullet leaves out: kind `oauth` (`KindOAuth`),
  completion `at_start` (`CompletionAtStart`), `Method.command` ("what to run on the host; shown, never run"),
  `Step.secret`, which this record names only under U8's caveat 1 and in D4, and `Answer.secret`, a `*bool` that the
  reader holding the step sets, nil meaning not known. None of the thirteen built-in methods of 0074-MADR D1's table has
  kind `oauth` or completion `at_start`, so D2's policy table and D4's step table have no row for either, and decide
  nothing about them.
* **The `Failure*` constant names are kept** (0073-MADR D3; the owner's ruling): `FailureCanceled` … `FailureOther`,
  as the first bullet has them.
* **A paste is offered by the method's `Paste` fact** (0074-MADR D3), **a sign-in takes three pastes at most, and a
  pasted denial ends it** (its D4). The owner confirmed these three rules. D4 below states them outright.
* **`(*OAuthSession).SessionKind()` is exported** (0074-MADR D5; the owner's ruling). D2 below reads the kind through
  it.
* **`NoLoopback` is experimental only** (0074-MADR D4; the owner's ruling): `signin.Options.NoLoopback` stays, and the
  two v1 option fields, `OAuthFlowOptions.NoLoopback` and `ChatGPTPlanOptions.NoLoopback`, are dropped. gobble sets it
  nowhere (D3).
  *(2026-11-11, later the same day: reversed by the owner. `OAuthFlowOptions.NoLoopback` and
  `ChatGPTPlanOptions.NoLoopback` are v1 API after all, and `signin.Options.NoLoopback` passes through to them
  (go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D4, as that record now reads). gobble is their
  call site: it sets the option for an ACP sign-in whose `authenticate` carries `_meta.gobble.remote: true` (D3's note
  and D4 step 1's of this date). "gobble sets it nowhere" is superseded.)*
* **`wizard.Options.OAuthClients` is dropped** from 0076-MADR D4 (the owner's ruling). `auth.Identity` (its D3) and
  `wizard.Options.Identity` (its D4) stay, and are all this record uses of 0076.
* **0075 gains `(*OAuthSession).StoredConsent(ctx, id) (granted, answered bool, err error)`** (its D5; the owner's
  ruling): the openai provider reads the consent through it, and a read that fails is refused as not granted. 0052-MADR
  D2 follows it.
* **Deferred past the cut-over** (the owner's ruling): one revocation table, and one error kind for a `LockRefresh`
  whose `ctx` ends (0076-MADR, "Found on review, not decided here"). Neither is relied on: D2's logout reads the
  method's `Revocable` fact (0074-MADR D1), and nothing in this record or 0052-MADR asserts a kind for a lock that is
  not taken.
* **A Go-floor record, numbered 0077,** is being written in that repository and lands before its 0076 minor,
  `v1.30.0` (the PLAN's rollout). It had no file there when this note was written, so it is named by number only.
  *(2026-11-11, later: it is a file now, go-llmprovider-sdk `0077-MADR-go-1-27-4-for-go-2026-6859.md`, with its PLAN.)*

### U8, settled: ACP request-scoped elicitation works during `authenticate` at acp-go-sdk2 `v1.35.0`

Read 2026-11-10 in `go list -m -f '{{.Dir}}' github.com/maccavelli/acp-go-sdk2` run in this repository.

* **(a) A nested request while `authenticate` is pending.** An inbound request runs its handler on its own
  goroutine (`connection.go:1948-2016`), with a context carrying its id (`:1949`). The reader goes on reading, and
  hands a response to the waiting call (`handleResponse`, `:2451-2488`). `AgentSideConnection.CreateElicitation`
  sends `elicitation/create` and waits for its answer (`agent_gen.go:882-887`; `sendRequest`,
  `connection.go:2668-2694`). The dispatch passes that context to `Agent.Authenticate` (`agent_gen.go:237-248`).
  Nothing requires a session: `gateMessage` checks only the negotiated capabilities (`types_gen.go:25422-25436`).
* **(b) Stable, with this shape.** `schema/schema.json` (the stable schema, version `1.25.0` in `schema/version`)
  defines `CreateElicitationRequest` (`:1350`), `ElicitationRequestScope` (`:1487`, "tied to a specific JSON-RPC
  request outside of a session (e.g., during auth/configuration phases before any session is started)", required
  `requestId`) and `ElicitationCapabilities` (`:4883`); `meta.json:28` lists `elicitation/create` as a client method.
  Go types: `CreateElicitationForm{Message, RequestedSchema, Session | Request}` (`types_gen.go:3476-3502`),
  `CreateElicitationUrl{ElicitationId, Message, Url, Session | Request}` (`:3601-3629`), `ElicitationRequestScope{RequestId}`
  (`:5847-5853`), and the answer `Accept{Content} | Decline | Cancel` (`:4225-4244`). URL mode is closed with the
  notification `elicitation/complete{elicitationId}` (`:2518-2529`; `agent_gen.go:857-862`).
* **(c) The id reaches the handler.** `acp.RequestIDFromContext(ctx)` returns the id of "the request the connection
  made ctx for", "Receiving middleware and handlers can read it" (`middleware.go:133-155`), as raw JSON that
  `acp.RequestId` decodes (`types_gen.go:12499-12526`).
* **(d) The gate.** The client advertises `clientCapabilities.elicitation.form` and/or `.url`. The SDK refuses a
  form without `elicitation.form` and a URL without `elicitation.url` as `*CapabilityError` before sending
  (`types_gen.go:25493-25516`), and `elicitation/complete` without `.url` (`:25382-25388`). gobble keeps the
  client's capabilities from `initialize` (`acpserver/agent.go:415` [HEAD], `:427` [WIP]).
* **(e) No deadlock.** Writes are serialised by one writer goroutine (`agent_gen.go:15-19`), and the elicitation is
  written before `authenticate`'s answer, since the handler returns only after it. A response waits only for the
  notifications received before it (`connection.go:2684`). The one client notification gobble handles is
  `session/cancel`, whose handler takes `a.mu` and cancels a turn (`acpserver/agent.go:1044-1051` [HEAD], `:1057-1064`
  [WIP]); it waits for no authentication. **So the relay must never hold `a.mu` while it waits for the client:** a
  `session/cancel` queued behind it would block the notification worker, and the elicitation's answer, which waits
  for that worker, would never be handed over (D4 says so; the PLAN tests it). `OverflowBlock`, the default (`connection.go:1018-1023`), pauses reading only
  while the notification queue is full, and forbids only a notification handler that waits for a later request
  (`:446-450`). gobble sets no `WithMaxConcurrentRequests` (`acpserver/serve.go:45-47`), and responses take no request
  slot. A client's `$/cancel_request` for `authenticate` cancels the handler's context (`:2494-2518`), and a
  context that ends sends `$/cancel_request` for the pending elicitation (`:3054-3072`); the input's end fails it
  (`:3045-3053`).
* **Caveats, none a blocker.** (1) The stable form schema has no secret format: `StringFormat` is `email`, `uri`,
  `date`, `date-time` (`types_gen.go:18110-18129`). The contract carries it instead, as `Step.secret`
  (go-llmprovider-sdk `docs/decisions/0073-MADR-publish-an-auth-contract-module.md` D2), and the step reaches the client
  in `_meta.gobble.step`; a form-only client that reads no `_meta` sees an ordinary string field. (2) No acp-go-sdk2 test sends a request-scoped elicitation from
  `authenticate` (`ElicitationRequestScope` appears in `types_gen.go` and `schema_gen_test.go` only); gobble's own
  loopback test is the proof (Confirmation). (3) gobble's own client refuses elicitation
  (`acpclient/conn.go:373-375`; `acpclient/acptest/client.go:43-46`), so the test uses a client of its own. (4)
  magic-cli-remote advertises form only and cancels a request-scoped form today
  (`internal/provider/acpagent/elicitation.go:29-48` in that repository); magic-cli-remote `0191-MADR-answer-agent-elicitation-forms-on-the-phone.md` is its to amend, and its
  proposed `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md` D8 does:
  Magic relays a request-scoped form on a sign-in connection of its own, and keeps advertising form only. The owner
  answered that record's O4 on 2026-11-10: form only, so Magic gets D4's right-hand column, and `_meta.gobble.step`
  carries each step to the phone; 0191's URL rules are unchanged. D4's left-hand column stays for any client that
  advertises `elicitation.url`.

*(2026-11-11: the pin moves to acp-go-sdk2 `v1.36.0`, and U8 holds there.)* On 2026-11-11 the owner approved moving
gobble-cli and magic-cli-remote together from `v1.35.0` to `v1.36.0` before gobble's `v0.6.0`.
*(2026-11-12: "moving gobble-cli and magic-cli-remote together from `v1.35.0` to `v1.36.0` before gobble's
`v0.6.0`" stays true: both moved before it. `v0.6.0` is the release of
[0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md](0055-MADR-release-v0-6-0-now-and-the-sign-in-redesign-as-v0-7-0.md),
cut from `main` before this record's work; this record's release, the ACP sign-in redesign, is `v0.7.0`.)*
[0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md)
decides it for this repository, and its PLAN makes the move before this record's PLAN reaches P2. The lines above are
`v1.35.0`'s and are kept. At `v1.36.0`, read with `git show v1.36.0:<file>` in a clone of acp-go-sdk2:

* **Unchanged.** `connection.go` and `middleware.go` do not differ between the two tags. `agent_gen.go` differs only in
  two doc-comment links, at `:970` and `:995`, so each `agent_gen.go` range cited above (`:15-19`, `:237-248`,
  `:857-862`, `:882-887`) is the same text at the same lines. `meta.json:28` is the same.
* **`types_gen.go` moved.** Each block is the same text at its new place, but for one doc-comment link in each of the
  four rows marked †:

  | Cited above, at `v1.35.0` | At `v1.36.0` |
  | :--- | :--- |
  | `gateMessage`, `:25422-25436` | `:25731-25745` |
  | the form and URL gate, `:25493-25516` | `:25802-25825` |
  | the `elicitation/complete` gate, `:25382-25388` | `:25691-25697` |
  | `CreateElicitationForm`, `:3476-3502`, declared at `:3479` † | `:3474-3500`, declared at `:3477` |
  | `CreateElicitationUrl`, `:3601-3629`, declared at `:3604` † | `:3599-3627`, declared at `:3602` |
  | `ElicitationRequestScope`, `:5847-5853`, declared at `:5851` | `:5845-5851`, declared at `:5849` |
  | the answer, `:4225-4244` | `:4223-4242` |
  | `CompleteElicitationNotification`, `:2518-2529` † | `:2518-2529` |
  | `RequestId`, declared at `:12489`, its decoding at `:12499-12526` | declared at `:12513`, `:12523-12550` |
  | `StringFormat`, `:18110-18129` | `:18291-18310` |
  | `LogoutRequest`, `:8312-8322`, declared at `:8315`, cited in D4 † | `:8323-8333`, declared at `:8326` |

* **`schema/schema.json` moved:** `CreateElicitationRequest` from `:1350` to `:1352`, `ElicitationRequestScope` from
  `:1487` to `:1489`, `ElicitationCapabilities` from `:4883` to `:4959`. The stable schema's version, in
  `schema/version`, is `1.31.0`, where it was `1.25.0`.
* **The probe.** The cross-repository session of 2026-11-11 reports that a probe of a request-scoped elicitation sent
  from inside `authenticate` passed on both tags. It was not re-run for this note; gobble's own loopback test
  (Confirmation 3) stays the proof.
* **One behaviour changes, outside sign-in.** `v1.36.0` gates `systemPrompt` on `session/new`, `session/load` and
  `session/resume`: an agent that does not advertise `sessionCapabilities.systemPrompt` answers `-32602`, where
  `v1.35.0` dropped the field unread. What gobble does about it is 0054-MADR D5's question, not this record's.

## Decision Drivers

* **The owner's answers of 2026-11-10** (the report's §11), binding here:
  * **Q7, ACP elicitation now**, "native sign-in inside ACP `authenticate` through request-scoped
    `elicitation/create`, instead of a CLI `--answers` channel"; it names 0016-MADR A2's amendment and U8.
  * **Q8**, the stdlib-only contract module whose types "SDK, gobble and Magic all use".
  * **Q1**, the engine under `llmprovider/x/signin`, outside the SDK's v1 promise until promoted.
  * **Q10**, per-app policy on borrowed client ids: gobble "refuses Codex's id and accepts the Grok CLI's as risk".
  * **Q14**, Magic takes acpext `v0.4.0` now; "the long-term shape of acpext's auth publication is not decided by
    this answer: keep it as an open item in the gobble MADR, recommending 'provider list only, facts from the
    contract'".
  * **Q15**, one error vocabulary: "gobble retires `FailureKind` in `--json`, and acpext/llm `ErrorKind` aligns".
  * **Q16**, close #16 "once the replacing gobble MADR exists", with the report's §13 text adjusted.
  * **Q21**, retire the old `--json` events at the cut-over.
* **The owner's answer of 2026-11-10 to the Magic draft's O2** (magic-cli-remote
  `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`; the options as put
  are under More Information): "amend gobble 0051 so `authenticate` accepts `_meta.gobble.fresh` and runs the sign-in
  even when a credential exists. The old credential is replaced only if the new one saves."
* **The owner's standing constraints:** the agent owns its flows; Magic relays; the reusable bulk lives in the SDK;
  "no gobble-specific features or baggage" in the SDK; the daemon and the app change in lockstep with no shim.
* **gobble's records:** no OAuth code of gobble's own (`0005-MADR-v1-feature-scope.md:1057-1058`); `auth.json` is
  gobble's (0005-MADR amendment of 2026-10-10 item 3); the SDK only in `llm/provider` (0004-MADR rule 4); acpext
  static, stdlib-only and tested against the root (0042-MADR D1, D3, A1); "No module may be required without a
  MADR in this repository that names it" (`AGENTS.md`).
* **Every surface renders the same steps,** so a disclosure or a consent cannot be on one path and missing from
  another (F30).

## Considered Options

* **A. ACP elicitation on `signin.Start`, the contract's steps everywhere** — a policy filter over the SDK's
  facts; `authenticate` relays each step as request-scoped elicitation when the client offers it; `--json` prints the
  steps; terminals render them through the wizard; one error vocabulary.
* **B. A CLI answer channel** — the report's recommendation, `--json` steps with an opt-in `--answers` on stdin;
  `authenticate` stays presence-only; ACP elicitation later.
* **C. acpext-carried sign-in** — `_gobble/auth` notifications and a `_gobble/auth/answer` request in acpext, as the
  gobble audit's §C sketched for clients without elicitation.

## Decision Outcome

Chosen option: "A. ACP elicitation on `signin.Start`", because it is the owner's choice of 2026-11-10 (Q7, with Q8,
Q10, Q14, Q15, Q21), U8 shows the pinned SDK carries it on the stable schema, and it gives one step model on every
surface with no channel of gobble's own. ~~This record is `proposed`, awaiting the owner's review of this draft.~~

*(2026-11-11: accepted.)* On 2026-11-11 the owner typed, in this repository's own session, "accept and approve 0051 and
0052 when required", and confirmed it to the cross-repository coordinator the same day: accept 0051-MADR and
0052-MADR, and start their phases that do not wait on the SDK. This record is accepted as written, with its notes of 2026-11-11.
The questions it put were answered by the owner on 2026-11-10, each as recommended, and are kept under "Questions
answered by the owner (2026-11-10)": QB (a), QC (i), QD (a) and QE (a), and O2 (a), which magic-cli-remote's
`0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md` put. No question of this record is
open. The records under "Records amended or superseded in part" are amended by
[0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md)'s P0, and its
"P0 record (2026-11-11)" says where.

### D1. Two new requirements, and the contract allowed at gobble's edge

* **Requirements, each in the commit of its first import.** The root requires go-llmprovider-sdk at the minor that
  carries its 0074 and 0075 (`v1.31.0` if its 0076 ships as `v1.30.0`, go-llmprovider-sdk
  `0074-PLAN-one-sign-in-handle-and-method-facts.md` Phase 8), in the 0051-PLAN's P2, since `llm/provider` already
  imports the SDK. That SDK minor requires the contract module, so `go mod tidy` lists
  `github.com/maccavelli/go-llmprovider-sdk/authcontract` `// indirect` from then on, at the tag go-llmprovider-sdk
  `0073-PLAN-publish-an-auth-contract-module.md` makes (`authcontract/v0.1.0`). It becomes a direct requirement in the
  0051-PLAN's P4, the commit of gobble's first import of it (`llm/provider/policy.go`), as `AGENTS.md`
  "Dependencies" asks: a requirement is added in the commit that adds its first import, and `go mod tidy` is clean at
  every commit. This record names both modules, as `AGENTS.md` asks. No `replace`, no committed `go.work`.
* **0004-MADR rule 4 is amended:** `authcontract` is a data module with no dependency, so `llm/provider`,
  `acpserver` and `internal/cli` may import it, as they import acpext; no core package (rule 1) may, and the rest of
  go-llmprovider-sdk stays `llm/provider`'s alone. In archtest, rule 4's check (`internal/archtest/check.go:308`)
  gains the exception for `acpserver` and `internal/cli`. Rule 1's check (`:249`) keeps its code: its roots
  (`:182-186`) are the core packages and `llm`, and its SDK case already admits `llm/provider` alone, so it refuses
  `authcontract` in `agent` as it stands; its test gains that case. Both are seen failing on a planted import from
  `agent` and from `acpclient`.
* **`llmprovider/x/signin` stays inside `llm/provider`,** behind gobble's own handle (D3). Its experimental status
  (go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D9) reaches no other package.

### D2. One policy filter over the SDK's method facts (G11; 0031-MADR D8 carried out)

`provider.Offered(id) []authcontract.Method` is `signin.Methods(d)` with gobble's policy applied. The policy is one
table in `llm/provider/policy.go`, read from facts, with no provider id but the accepted-risk list:

| Rule | From | Fact read |
| :--- | :--- | :--- |
| No token paste, no vendor import | gobble writes no vendor store; the `offers` comment (`llm/provider/signin.go:72-76` [HEAD]) | `Kind` |
| No host-only method | gobble runs no host command for a sign-in | `Kind` |
| No borrowed client unless accepted | `0005-MADR-v1-feature-scope.md:1057-1058`; its second amendment of 2026-10-14 (grok); Q10 | `ClientIdentity`, and the list `acceptedBorrowed = {"grok"}` |
| No method with a required consent gobble does not honour | a consent needs gobble's wording and gate first (0052-MADR D2) | `Consents`, and the list `honouredConsents`, empty until 0052-PLAN P3 adds `chatgpt_plan.background_use` |

Every predicate is derived from `Offered`:

* `HasLogin(id)`: an offered method yields `oauth_session`. `HasKeyLogin(id)`: one yields `device_key`.
* The Pi import takes an `oauth` entry only for a provider whose offered session method is `vendor_borrowed` and
  accepted, the client Pi used; `HasPiLogin` [WIP] is not added.
* The logout revokes when the stored session's method is `Revocable`, and only for a session gobble accepts (D2's
  last bullet), never another app's (F54).
* `--method` words stay one table, `methodIDs` (`llm/provider/signin.go:23` [HEAD]), the CLI's vocabulary. A bare
  `gobble auth login <id>` takes the first offered method that is not `api_key`, in the descriptor's order.
* `signInWords` renders the offered methods' `Label`s; `providers --json`'s `signIn` stays display text.
* acpext's `Login` is derived as "an offered method other than `api_key`".
* **The resolver accepts a stored session only of an offered kind.** `auth.Resolver.HasLogin func(id string) bool`
  (`internal/auth/resolve.go:52`) becomes `Accepts func(id string, e Entry) bool`, true when the entry's session
  kind (`OAuthSession.Kind`, inferred by the SDK's rules when empty, go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D5) is the kind an
  offered method signs in to. `Has`, `Credential`, `auth status`, `auth check`, `authenticate`, provider choice and
  logout then agree (F54); the plan's own use is 0052-MADR's.
  *(2026-11-11: settled. The owner ruled that the SDK exports `(*OAuthSession).SessionKind() SessionKind`
  (go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D5): `Kind` when set, else the kind the SDK infers by its own
  rules. `provider.Accepts`, in `llm/provider`, calls that method on the session `sdkStore` builds from the entry.
  "Inferred by the SDK's rules when empty" above means that call: gobble copies none of the rules, since a copy could
  drift from the SDK's.)*

*(2026-11-11, later; the owner's ruling of that day on `NoLoopback`, D3's note.)* **A remote sign-in is offered only
what completes without the host's listener.** `provider.Remote(m authcontract.Method) bool` is true for a method whose
`Completion` is `device`, or whose completion is a browser one and whose `Paste` fact is set; it reads those two facts
and no provider id. When `authenticate` carries `_meta.gobble.remote: true`, the methods the relay may drive (D4 step
3) are those of `Offered` that `Remote` accepts. Of the built-in offered methods it refuses one, grok's `browser_oauth`
(`browser_loopback`, no `Paste`; go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D1, D3).
`Offered`, and so `_meta.gobble.methods` (D4), does not change with it: `initialize` comes before any `authenticate`,
so gobble cannot know there whether a later sign-in is remote, and a client reads the same two facts, `completion` and
`paste`, from each method. What a client that names a refused method gets is D4 step 1's note. *(Also of this date:
an honoured consent's `text` in `Offered` is gobble's wording, not the SDK's default;
[0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
D2's note of 2026-11-11, later.)*

### D3. One engine per kind of surface

* **ACP `authenticate`** drives `signin.Start` through one gobble handle in `llm/provider`:
  `provider.StartSignIn(ctx, id, method string, o SignInOptions, store LoginStore) (*SignInRun, error)`, with
  `Method() authcontract.Method`, `Steps() <-chan authcontract.Step`, `Answer(ctx, authcontract.Answer) error`,
  `Wait(ctx) (SignInResult, error)` and `Cancel()`. `SignInResult{Key string; Yields authcontract.Yields}` is gobble's
  own type, so that `acpserver`, which may import `authcontract` but not `x/signin` (D1), can name it: `Wait` saves a
  session through `store` and returns `Yields` `oauth_session` with no key, or returns the key with `Yields`
  `device_key` for the caller to store (as `signInTo` does, `internal/cli/login.go:70-91` [HEAD]), then runs the
  listing check the wizard runs (`Discover`, `llm/provider/signin.go:179` [HEAD]). It never opens a browser on the
  host (F56). `signin.Options` carries `Identity: auth.Identity{Name: "gobble", Version}` (go-llmprovider-sdk
  `0076-MADR-auth-error-kinds-and-doc-drift.md` D3), `AgentName: "gobble"`, and what the SDK's plan sign-in needs to
  treat an ACP sign-in as the same host and client as a terminal one (go-llmprovider-sdk
  `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D4, D6, D7): `HostID` from `<state>/host-id`
  (0052-MADR D4); `ClientStore`, the `sdkStore` adapter over `<state>/clients.json` when its store has one (0052-MADR
  D5), since `x/signin` takes the client store only as this option and nil means none; `ConsentText`, gobble's wording
  (0052-MADR D2); and `Previous`, the stored session loaded through `store` when its kind is the one the method signs
  in to, so a re-sign-in with an answer asks no consent again and sends `id_token_hint`.
  *(2026-11-11: that is the whole list. gobble sets `signin.Options.NoLoopback` nowhere: over ACP it drives a `device`
  or a `browser_paste` sign-in, or grok's `browser_loopback` when named (D4 step 3), and the last has no paste that
  could complete it without a listener. By the owner's ruling of 2026-11-11 the option stays, experimental with
  `x/signin`, and the two v1 option fields of go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D4 are
  dropped; nothing here used them. `wizard.Options.OAuthClients` is dropped from
  go-llmprovider-sdk `0076-MADR-auth-error-kinds-and-doc-drift.md` D4 by the same ruling; gobble never set it. `auth.Identity`,
  which this bullet sets, and `wizard.Options.Identity` stay (its D3, D4).)*
  *(2026-11-11, later: the note's first two sentences, "that is the whole list" and "gobble sets
  `signin.Options.NoLoopback` nowhere", and its "the two v1 option fields … are dropped", are superseded by the owner's
  reversal of go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D4 that day; the rest of the note
  stands.)* **gobble sets `signin.Options.NoLoopback` for a remote sign-in.** An `authenticate` whose
  `_meta.gobble.remote` is `true` (D4 step 1) says that the person completing the sign-in is not on the agent's host,
  so the host's loopback listener can never receive the callback. `SignInOptions` gains `Remote bool`, set from that
  member and from nothing else, and `provider.StartSignIn` then sets `NoLoopback`, which `x/signin` passes to
  `OAuthFlowOptions.NoLoopback` or `ChatGPTPlanOptions.NoLoopback`: no listener is bound, the redirect URI names the
  method's first registered port, and only a paste completes the sign-in. Ports 1455 and 1457 stay free for other
  agents on the host. Everything else keeps the listener: every terminal sign-in (the wizard's, below), and an ACP
  `authenticate` with `remote` absent or `false`, a local client's, whose browser sign-in still completes by the
  loopback or by a paste.
  * **What cannot complete that way is the SDK's to say** (its D4): in `x/signin`, `NoLoopback` with a method whose
    `Paste` fact is empty is `ErrUnsupported`, as for grok's `browser_oauth`, and with `NoPaste` as well
    `ErrInvalidRequest`, each returned by `Start` before any request. gobble meets neither: D2's remote rule and D4
    step 1 refuse such a method before `Start` is called, and the ACP path never sets `NoPaste`.
  * **One reading to confirm with the SDK.** Its D4 states the `ErrUnsupported` rule for "a method whose `Paste` fact
    is empty", in a bullet about browser sign-ins; a device method has no `Paste` fact either, and binds no listener
    in any case. gobble sets `NoLoopback` on a remote device sign-in too, as the ruling says, and reads that rule as
    not applying to it. The PLAN's `TestStartSignInRemoteDevice` holds the reading; a refusal there stops P6 as a
    deviation, to be settled in go-llmprovider-sdk's record, not by gobble leaving the option off.
* **`gobble auth login --json`** keeps `wizard.ResolveCredential` (0031-MADR D7, P4a), with `wizard.Options.Driver`
  set to a driver of gobble's that prints each step (D5) and gives no answer, in place of today's `ShowDeviceCode` and
  `OpenURL` hooks (`llm/provider/signin.go:159-171` [HEAD]). The wizard's listing check and save stay as they are. The
  owner accepted `Driver` in the wizard's v1 API on 2026-11-10 (the SDK drafts' SDK-3); this is its caller (QE).
* **Terminal surfaces** (`gobble auth login` without `--json`, `/login`, `gobble setup`, the first-run offer,
  `gobble acp --login`) keep `wizard.ResolveCredential` with the wizard's own renderer of the same steps
  (go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D6), so every disclosure and question reaches every terminal path by construction
  (F30). The SDK's `OpenURL` is set on all of them, so `/login` and `auth login` both open the host's browser (F56).
  A device code's text names its expiry (F56).
* **`gobble acp --login <id>` shows the wizard's menu** of the offered methods, the API key included, instead of the
  bare login (`internal/cli/acp.go:101-103`). This restores what 0031-PLAN P4b-2 step 4 asked, "`gobble acp --login
  openai` offers the plan and the API key" (`0031-PLAN-sdk-current-releases-and-adoption.md:328`; *2026-11-11: `:347` on `main` since `86ad4a7`, `:328` being the line at `d6c10a5` and on the archive branch*), for every
  provider (F53).

*(2026-11-12; D9.)* The terminal surfaces and `--json` above name gobble to the identity provider as the ACP bullet
does: `provider.SignIn` sets `wizard.Options.Identity` to the same value, which it does not at `c47f4a7` (D9, N1).

### D4. ACP `authenticate` signs in by elicitation (amends 0016-MADR A2)

* **Advertised methods.** Each agent auth method keeps its id, name and description (`acpserver/auth.go:32-41`) and
  gains `_meta.gobble.methods`: the provider's `Offered` methods, each an `authcontract.Method`. A client learns how
  each completes, what it yields, whose client it signs in as, and its consents, before it asks.
  *(2026-11-11, later.)* A consent's `text` there is gobble's wording, the one the sign-in's `confirm` step shows, not
  the SDK's default, and its `links` are the SDK's: `Offered` sets it (D2's note of this date), so a client that shows
  a consent outside a sign-in shows the same words and link as the `confirm` step does.
* **`authenticate` with a provider's id:**
  1. `_meta.gobble.method`, when present, names a method by its contract id (`chatgpt_plan`, `device_code`, …);
     one not offered is -32602. `_meta.gobble.fresh`, when `true`, asks for a sign-in whatever is stored (O2); any
     other value but `false` or absence is -32602.
     *(2026-11-11, later; the owner's rulings of that day.)* **`_meta.gobble.remote`**, when `true`, says that the
     person completing this sign-in is not on the agent's host; absent or `false` means local; any other value is
     -32602, as for `fresh`. magic-cli-remote sends it on every sign-in the phone starts, beside `method` and `fresh`
     (its `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md` D8).
     With it, gobble sets `NoLoopback` (D3's note) and drives only the methods D2's remote rule accepts:
     * **A named method that is offered but that rule refuses** (grok's `browser_oauth`) is answered before anything
       starts: `auth_required` with `ErrorData{Provider, Kind: "unsupported", Reason, Command}`, `Reason` saying that
       the method completes only in a browser on the agent's host, and `Command` the host command that runs it,
       `gobble auth login grok --method browser`. No elicitation is sent, `signin.Start` is not called, and no port
       is bound. `unsupported` is the contract's word for it (go-llmprovider-sdk
       `0073-MADR-publish-an-auth-contract-module.md` D3), the one `signin.ErrorKindOf` gives the SDK's own
       `ErrUnsupported` for the same case (its 0074-MADR D4, D8). A method not offered at all stays -32602, as above:
       that request names nothing gobble signs in with, where this one names a method that cannot complete from
       where its person is.
     * **With no method named,** a remote sign-in takes the first offered method `Remote` accepts, and step 3's enum
       lists only those.
  2. **A credential is present** (gobble's order, as today), **`_meta.gobble.fresh` is not `true`, and** no method
     was named, or the stored credential is what the named method yields (a key for a key; a session of the kind it
     signs in to, so grok's browser and device sign-ins count as one): `{}`, as A2 decided.
  3. **Otherwise, a client that advertised `elicitation.form`:** gobble signs in, relaying the steps below, and
     answers `{}` once the credential is saved. **With `fresh`, the stored credential is left exactly as it is until
     the new one saves** (O2): the sign-in neither deletes nor revokes it first; `Wait`'s save through `store`, or
     the key's `auth.SetKey`, replaces it in one locked write of `auth.json`; a sign-in that fails, is declined or is
     cancelled changes nothing stored. The replaced session is not revoked (a revocation is a logout's, QD). The methods driven are those whose `Completion` is `device` or
     `browser_paste`; `browser_loopback` (grok's browser, go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D1) only when named, since
     it completes only where the browser reaches gobble's host *(2026-11-11, later: and only for a local sign-in;
     step 1's note)*. With several, gobble asks with a form whose one
     field is an enum of their labels. An API key is never asked over ACP.
     *(2026-11-11, stated outright; three rules of go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md`
     D3 and D4 that the owner confirmed that day.)*
     **(1) Grok's `browser_loopback` sign-in has no `accepts` and no `paste` step.** A browser sign-in offers a paste
     only when its method's `Paste` fact is set, and grok's `browser_oauth` has none, whatever `NoPaste` says. A named
     grok browser sign-in is therefore the table's "`open_url` without `accepts`" row and nothing more: only the
     loopback completes it, and an answer sent for it is refused (`ErrInvalidRequest`).
     **(2) The cap is three pastes in all, two re-asks.** Each of the first two bad pastes brings a new `paste` step;
     a third bad one ends the sign-in `failed`, `invalid_request`.
     **(3) A pasted denial is not a bad paste.** A pasted URL that carries this sign-in's `state` and
     `error=access_denied` ends the sign-in at once, `failed` with kind `denied`, with no re-ask; `authenticate` then
     answers `auth_required` with `ErrorData.Kind` `denied`. Another `error` value ends it the same way with kind
     `other`.
  4. **Otherwise:** `auth_required` with `ErrorData{Provider, Reason, Command}`, as today.
* **The steps, as elicitation** (each request-scoped, `requestId` from `RequestIDFromContext`; each carries
  `_meta.gobble.step`, the `authcontract.Step`, so a contract-aware client renders it natively and reads from its
  `secret` field that the answer is a secret, U8 caveat 1). The column is chosen by the client's advertised
  capabilities, since acp-go-sdk2 refuses URL mode to a client without `elicitation.url` (U8 (d)):

| Step | Client offers `elicitation.url` | Client offers `elicitation.form` only |
| :--- | :--- | :--- |
| `open_url` with `accepts` | URL mode to the authorize URL; on its `accept`, the `paste` step's form | a form with no fields, the URL and the paste instruction in `message`; on its `accept`, the `paste` step's form |
| `open_url` without `accepts` (named loopback) *(2026-11-11: grok's `browser_oauth`, which never has `accepts`)* | URL mode; `elicitation/complete` when the loopback finishes | a form with no fields, the URL in `message` |
| `show_code` | URL mode to the verification URL, the code and expiry in `message`; `elicitation/complete` when the poll finishes | a form with no fields, URL, code and expiry in `message` |
| `paste` (a re-ask after a bad paste included) *(2026-11-11: two re-asks at most, three pastes in all)* | a form, one required string `callback`, its title the step's `text` | the same |
| `confirm` | a form, one boolean `allow`, default false (consent must be express), `message` the step's `text` and links | the same |
| `form` | a form whose properties are the fields (`select` as an enum) | the same |
| `waiting` | nothing is sent | nothing is sent |
| `done` / `failed` | `authenticate` answers `{}` / `auth_required` with `ErrorData.Kind` the failure's kind | the same |

* **Shown steps are acknowledged, not answered** (the owner's O4 answer, 2026-11-10, is that Magic is a form-only
  client; go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D3's answer rule). An `open_url` or
  `show_code` elicitation, URL mode or a form with no fields, asks the client to show something, not to answer it:
  * its `accept` builds no `authcontract.Answer` and sends nothing to `SignInRun.Answer`; the sign-in runs on. For an
    `open_url` with `accepts`, the relay then sends the `paste` step's form. A client may accept at once, as Magic
    does once it has shown the step, or when its user has opened the page;
  * its `decline` or `cancel` cancels the sign-in, which answers `auth_required` with kind `canceled`;
  * while it is still open, a device poll or a loopback that finishes ends it: gobble sends `elicitation/complete`
    for URL mode, and for a form ends the request's context, which sends `$/cancel_request` (U8 (e)).
* **Answers.** The `accept` of a `paste`, `confirm` or `form` step's form becomes the `authcontract.Answer` for that
  step's own `seq`: `callback` as `Paste`, `allow` as `Confirm`, the properties as `Fields`. An `open_url`'s `seq` is
  never answered, so the SDK's refusal of an answer to a step that does not ask (`ErrInvalidRequest`) is never met.
  `decline` and `cancel` cancel the sign-in, which answers `auth_required` with kind `canceled`. A loopback that wins
  while a paste form waits cancels the form (its context ends, U8 (e)).
* **One sign-in per provider at a time;** a second `authenticate` for it while one runs is -32600 naming the
  running one. A client's `$/cancel_request` cancels the sign-in.
* **No lock is held while the client is awaited.** The relay takes `a.mu` only to read the client's capabilities and
  to register or remove its provider's sign-in, never across `CreateElicitation` (U8 (e)).
* **Secrets.** A pasted callback is never logged, never written but to the provider's flow, and never echoed in a
  re-ask (Q11; go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D4). The relay builds each `authcontract.Answer` with `Secret` taken from
  the step it sent for that `seq`, so the contract's own redaction masks it (go-llmprovider-sdk `0073-MADR-publish-an-auth-contract-module.md` D2, D6 rule
  5).
* **After a sign-in** gobble drops the run's shared session for that provider and moves its credential generation
  (`internal/cli/credentials.go:81`; 0046-MADR D8), as a terminal sign-in does.
* **No ACP `logout`** (QD). ACP's `LogoutRequest` carries only `_meta` and "Terminates the current authenticated
  session" (`types_gen.go:8312-8322`), which for an agent of nine providers names none.

### D5. `gobble auth login --json` prints the contract's steps, output-only (amends 0016-MADR A3, 0005-PLAN F4-3c decision 6)

* One JSON line first, `{"event":"method","provider":<id>,"method":<authcontract.Method>}`, then one per step,
  `{"event":"step","provider":<id>,"step":<authcontract.Step>}`, the last `done` or `failed`. A failure's
  `step.error.kind` is the contract's (D6). A usage error is one `failed` step, `invalid_request`.
* `--api-key -` with `--json` prints the method line for `api_key` and a `done` step with `yields: "api_key"`.
* **No answer channel** (Q7 chose elicitation over `--answers`). `--json` runs with no paste (`NoPaste`), so no
  `paste` step is emitted. A method whose sign-in would ask a `confirm` or a `form` is refused before it starts, exit
  2, with a `failed` step of kind `unsupported` whose message names the ways that can ask (a terminal, an ACP client).
* **The five old events retire** (Q21): `device_code`, `browser_url`, `saved`, `error` and [WIP] `notice`.
* Nothing goes to stderr, the host's browser is not opened, and exit codes are unchanged.

### D6. One error vocabulary (Q15; amends 0049-MADR D3)

* **`FailureKind` retires.** `provider.ErrorKind(err error) string` returns the contract's word: `signin.ErrorKindOf`
  for an error of the SDK's sign-ins, else `llm.Kind`'s. Its callers are `--json` (D5), the log lines, the
  `SignInError` text (`llm/provider/signin.go:134-139` [HEAD]) and a provider retry's cause
  (`llm/provider/provider.go:273` [HEAD]).
* **`llm.Kind` returns the contract's spellings.** `llm` imports no SDK (rule 1), so it keeps plain strings and a
  test in `llm/provider` holds every word it returns inside `authcontract`'s `ErrorKind` constants:

| gobble's word today | The contract's word |
| :--- | :--- |
| `rate-limited`, `not-permitted`, `context-overflow` | `rate_limited`, `not_permitted`, `context_overflow` |
| `empty_reply` | `empty_answer` (gobble's `ErrEmptyReply` is the SDK's `ErrEmptyAnswer`, `llm/provider/provider.go:643-655` [HEAD]) |
| `empty` (`FailureKind` and `RetryEmpty` only: a listing that recommended nothing) | `other` |
| `workspace`, `quota`, `incomplete`, `auth`, `offline`, `unavailable`, `unsupported`, `timeout`, `canceled`, `other` | unchanged |

* **acpext's `ErrorKind` and `RetryCause` take the contract's values** at `acpext/v0.5.0` (QB). acpext's `ErrorKind`
  holds every one of the contract's eighteen words, since `ErrorData.Kind` now also carries a sign-in's failure (D4):
  its fourteen, respelled as the table says, and four new constants, `KindDenied` (`denied`), `KindExpired`
  (`expired`), `KindInvalidRequest` (`invalid_request`) and `KindInvalidProvider` (`invalid_provider`).
  `TestKindAgreesWithFailureKind` (`llm/provider/limit_test.go:180-189`) gives way to a test that every word
  `provider.ErrorKind` can return is an acpext constant.
* *(2026-11-11, found on review: the agent's own causes.)* `_gobble/retry`'s `cause` is not acpext's constant on the
  wire. `acpserver/prompt.go:108` sends `acpext.RetryCause(ev.Cause)`, a conversion of `agent.RetryCause`
  (`agent/event.go:97-104`), whose `RetryEmptyReply` is the literal `"empty_reply"`; and `internal/cli/view.go:249`
  compares that literal. With acpext's value alone respelled, an empty reply's wait would still say `empty_reply`. One
  vocabulary means the agent's word moves too: `agent.RetryEmptyReply` becomes `"empty_answer"`, and the CLI compares
  acpext's constants. `agent` may not import acpext (archtest rule 12), so a test in `acpserver` ties the two (the
  PLAN's P4 step 4b). **Kept, with the reason:** the Go names (`llm.ErrEmptyReply`, `agent.RetryEmptyReply`,
  `acpext.KindEmptyReply`, `acpext.RetryEmptyReply`) are identifiers, not wire words; and the personality key
  `retry.empty_reply` (`personality/keys.go:12`) is a heading in the built-in and in users' `messages.md` files, a
  file format of its own that is not an error kind and that renaming would break.

### D7. acpext: the doc now, and the long-term shape the owner chose (Q14, QC)

* **Now:** magic-cli-remote takes `acpext/v0.4.0` (its work).
  *(2026-11-11: not yet. magic-cli-remote's `go.mod` still requires acpext `v0.3.0`, at its origin/master `eefb7c25`
  and on its local master. The bump is its `docs/decisions/0179-PLAN-gobble-native-acp-provider.md` P7, commit 7a,
  which is still `proposed`. "Now" is the timing the owner chose in Q14, not a state reached.)*
  acpext's `AuthProvider` doc (A1) says what is true:
  `Command` runs the provider's bare sign-in, or stores a key from stdin; `Login` means "has a method other than an
  API key", never "device code". It is committed now and published with the next tag.
* **`AuthProvider` stays comparable,** its four fields unchanged, so `slices.Equal` in
  `TestAcpextAuthProvidersAreTheRegistry` (`internal/cli/acpext_test.go:104-107`) and Magic's use compile.
* **Long term, answered by the owner on 2026-11-10 (QC (i)):** "provider list only, facts from the contract". Q14 had
  left it open for this record. acpext keeps the static
  list for a client that has not connected (0042-MADR D3), and gains only ACP extension names: `AuthenticateMeta{Method
  string; Fresh bool}` for `_meta.gobble.method` and `_meta.gobble.fresh` (O2), and the key names of
  `_meta.gobble.methods` and `_meta.gobble.step`. The facts
  are `authcontract.Method`, which acpext never names, so 0042-MADR A1's standard-library rule and
  `check-acpext-deps` stand. These, A1 and D6's values tag `acpext/v0.5.0` (report A3).
* *(2026-11-11, later; the owner's rulings of that day, with the two names the cross-repository coordinator fixed.)*
  `acpext/v0.5.0` also carries two members. `AuthenticateMeta` gains `Remote bool`, JSON `remote`, for
  `_meta.gobble.remote` (D4 step 1), so it is `AuthenticateMeta{Method string; Fresh bool; Remote bool}`. The members
  of `_meta.gobble` this record defines are then:

  | Where | Member | Value | In acpext |
  | :--- | :--- | :--- | :--- |
  | the `authenticate` request | `method` | a contract method id (`chatgpt_plan`, `device_code`, …); absent, gobble chooses (D4 step 1) | `AuthenticateMeta.Method` |
  | the `authenticate` request | `fresh` | `true`: sign in whatever is stored; absent or `false`: not (O2) | `AuthenticateMeta.Fresh` |
  | the `authenticate` request | `remote` | `true`: the person completing this sign-in is not on the agent's host; absent or `false`: local (D3's and D4 step 1's notes) | `AuthenticateMeta.Remote` |
  | each agent auth method in `initialize` | `methods` | `[]authcontract.Method`, `Offered`'s (D4) | the key name `MetaMethods` |
  | each elicitation of a sign-in | `step` | the `authcontract.Step` relayed (D4) | the key name `MetaStep` |

  **`ErrorData` gains `Consent string`, JSON `consent`.** It is set, with `Kind` `not_permitted`, when a request is
  refused because a consent is not granted, and its value is the consent's id: for the ChatGPT plan, the SDK's
  `openai.ConsentBackgroundUse`, `chatgpt_plan.background_use` (go-llmprovider-sdk
  `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` D1). It is set only when allowing that consent would let
  the request through: a request refused because the stored answer could not be read is `not_permitted` with no
  `consent`. Where gobble sets it is
  [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md)
  D2's note of this date. It goes in `ErrorData`'s sorted order, between `command` and `error`, which the type's doc
  ("Its fields are in their JSON names' sorted order") and `TestErrorDataSortedMembers` (`acpext/roundtrip_test.go:203`)
  hold, and it is tagged `omitzero`, as every member of `ErrorData` is (`acpext/types.go:400-416`); for a string that
  omits exactly the empty value that `omitempty` would. Both are additions, compatible under apidiff, as
  `WindowMinutes` was (`6cee0af`).

### D8. Issue #16 closes as superseded, on acceptance (Q16)

On the owner's word after acceptance, #16 is closed with this text (the report's §13 text, adjusted):

```text
Closing as superseded by docs/decisions/0051-MADR-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md,
from magic-cli-remote's docs/reports/0245-REPORT-non-api-key-auth-across-magic-gobble-and-the-sdk.md.

What carries forward: clients get each provider's methods as data, how each completes, what it yields, whose
client it signs in as and the consents it asks, as go-llmprovider-sdk's authcontract.Method in each ACP agent
auth method's _meta.gobble.methods, and gobble auth login --json's first line. Not as
AuthProvider.Methods []string, which would make AuthProvider incomparable, and not as --method words alone.
A test asserts every acpext Command and every method gobble offers is accepted by gobble auth login.
acpext's Login stays, documented as "has a method other than an API key", not "device code"; acpext/v0.5.0
carries the doc. The remote paste is answered by ACP elicitation inside authenticate, not by a --json
answer channel.
```

### D9. gobble names itself on every auth request the SDK lets it name (added and accepted 2026-11-12)

*(2026-11-12. Added by amendment; D1 to D8 stand as written. The direction is the owner's. The owner ruled the same
day on the three questions this section first put, kept with their options under "Questions for the owner
(2026-11-12), ruled the same day": Q1 "New phase P2a", Q2 "Amend SDK 0079", Q3 "No separate test". D9 is accepted
2026-11-12 as it reads here. It was drafted and ruled on one day, before any commit, so its text is the ruled one;
the draft's own choice for Q2, gobble naming the session itself, survives as that question's option (a).)*

The owner, on 2026-11-12: "Where possible gobble auth would send gobble as the name/referrer not the sdk".

go-llmprovider-sdk carries the SDK's half as `0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md`,
accepted 2026-11-12, with its PLAN in progress; both are committed there at `f50466b`, and they ship in `v1.31.0`, the
minor D1 requires, with that repository's 0074 and 0075. What it gives a consumer:

* a sign-in keeps its `auth.Identity` on the session it returns, in memory, never stored (its D2);
* `(*auth.OAuthSession).UseIdentity(auth.Identity)` names a session that has no identity; the first name given wins
  (its D2);
* a refresh, a revocation, and the discovery a Grok or plan revocation reads first send the session's identity as their
  User-Agent (its D3);
* the openai and grok providers and `catalog`'s listings hand `WithClientInfo`'s name and version to the session they
  use (its D4);
* the ChatGPT backend's `originator` is `WithClientInfo`'s name (its D1);
* the wizard names its own requests and its logout with `Options.Identity` (its D5);
* the User-Agent keeps the SDK as its trailing token, `<app>/<ver> (<os>; <arch>) go-llmprovider-sdk/<ver>`, by that
  owner's ruling 3, the form [0003-MADR-gobble-product-identity.md](0003-MADR-gobble-product-identity.md) item 2
  already records for provider requests.

Its Consequences name one gobble-side gap: gobble's `Logout` builds its own session to revoke (its C38) and must call
`UseIdentity` itself. Read against this repository at `c47f4a7`, that gap is one of three.

**The name gobble passes.** `ClientName` is `appName`, `const appName = "gobble"` (`internal/cli/agent.go:26`), passed
with `identity().Version` at `internal/cli/login.go:75`, `agent.go:37`, `agent.go:56`, `title.go:37` and
`models.go:293`. `identity` is `buildinfo.Identity` (`internal/cli/version.go:13`), whose `Version` "always parses as
SemVer" (`internal/buildinfo/buildinfo.go:24-26`).

#### Every gobble site that reaches an SDK auth or identity surface, at `c47f4a7`

| Id | Site | Where | Text at `c47f4a7` | Under `v1.31.0` |
| :--- | :--- | :--- | :--- | :--- |
| N1 | terminal and `--json` sign-ins, through the wizard (D3) | `llm/provider/signin.go:184` | `res, _, err := wizard.ResolveCredential(ctx, w, wizard.Options{` | **gobble acts.** The options (`:184-198`) set `AgentName` but no `Identity`, whose zero value names the SDK (go-llmprovider-sdk `wizard/configure.go:137-143`). Grok's authorize and device `referrer`, the User-Agent of every sign-in request, and the identity the session keeps are then the SDK's. The wizard's sign-ins take their name from `Options.Identity` alone (go-llmprovider-sdk `wizard/auth.go:292`, `:725`), never from `ProviderOptions` |
| N2 | the wizard's own requests: the listing check, the Kilo profile, the Ollama check | `llm/provider/signin.go:181` | `llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),` | **Covered, today and after.** `ProviderOptions` names gobble; 0079-MADR D5 puts `Identity` first and lets this option win, with the same name |
| N3 | the ACP sign-in, planned | D3 above; the PLAN's P6 step 1 | `Identity: auth.Identity{Name: "gobble", Version}` | **Covered by the plan.** 0079-MADR D2 carries it into the session `Wait` saves. P6 gains a test that pins it |
| N4 | a provider's refresh: every turn, title and session switch | `llm/provider/provider.go:194`, the session from `:107` | `llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),` | **Covered automatically** by 0079-MADR D4: the grok provider, and the openai provider for a plan session, hands the name to the session |
| N5 | a listing's refresh | `llm/provider/list.go:92`, its options at `:71` | `cat, err := catalog.List(ctx, llmprovider.ProviderID(id), src, opts...)` | **Covered automatically:** `catalog.List` reads the token through `listingToken` (go-llmprovider-sdk `llmprovider/catalog/discovery.go:259`), where 0079-MADR D4 hands the name over |
| N6 | the agent's model facts, a lookup that can refresh | `llm/provider/model.go:157`, called through the hook at `internal/cli/agent.go:101-107` | `info, _ := catalog.CachedModel(ctx, llmprovider.ProviderID(id), src, model, opts...)` | **The SDK acts, and gobble passes the name.** At `f50466b`, `CachedModel` asks the session for its token itself (go-llmprovider-sdk `llmprovider/catalog/lookup.go:138`), not through `listingToken`, so 0079-MADR D4 does not reach it; by the owner's ruling on Q2, a pending amendment of go-llmprovider-sdk `0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md` puts it through D4's hand-off. That hand-off names the session only from a `WithClientInfo` among the call's options, and gobble passes none: the lookup's options are `WithoutModelMetadata`, `sdkOptions`' client and base URL, a workspace and the layers (`model.go:150-157`; `sdkOptions` holds no `WithClientInfo`), and the hook's `provider.Options` at `agent.go:102` sets no `ClientName`. Today a refresh it starts names gobble only when a provider sharing the run's `Sessions` named the session first, which nothing orders |
| N7 | `gobble auth logout`: the revocation, and Grok's discovery before it | `llm/provider/login.go:104`, called from `internal/cli/auth.go:302` | `if rerr := auth.RevokeOAuthSession(ctx, l.session(id, nil, o.HTTPClient, nil)); rerr != nil {` | **gobble acts** (0079-MADR C38): the session is gobble's own, no provider uses it, and `LoginOptions` (`login.go:54-56`) carries no name |
| N8 | sessions the SDK loads through gobble's store | `llm/provider/login.go:144` | `return l.session(string(p), a.s, nil, nil), nil` | **No action.** A refresh that adopts a sibling's rotation keeps the refreshing session's identity (0079-MADR D2, "Kept"); P6's `Previous` is read for its fields, and the sign-in's own `Identity` names its requests |
| N9 | Pi's login, imported | `internal/cli/piimport.go:203` | `login, ok := provider.PiLogin(id, entry.Access, entry.Refresh, exp)` | **Nothing to name.** It writes `auth.json` and sends no request; its first refresh is N4's, N5's or N6's |
| N10 | `auth status`, `auth check`, ACP `authenticate` | `internal/cli/auth.go:431`; `:548`; `acpserver/auth.go:59` | `func (e *runEnv) statusRow(id string) authStatusRow {`; `c, ok, err := e.credentials().Credential(id)`; `func (a *Agent) Authenticate(_ context.Context, p acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {` | **Nothing to name.** Each reads `auth.json` and sends nothing; `internal/auth` makes no HTTP request. `authenticate`'s sign-in is N3 |
| N11 | the ChatGPT `originator` | none | none | **Nothing to name at `c47f4a7`.** gobble keeps a login for Grok only (`llm/provider/login.go:67`) and offers OpenAI's plan, whose requests carry no `originator` (0079-MADR D1, D6). A Codex session, which D2's policy does not offer, would take `WithClientInfo`'s name with no change here |

#### What gobble does

* **N1. The wizard names gobble.** `provider.SignIn` sets `wizard.Options.Identity` to `auth.Identity{Name:
  o.ClientName, Version: o.ClientVersion}`, the name and version that already lead its listing's User-Agent (N2). Every
  terminal surface and `--json` (D3, D5) passes through it. With `internal/cli`'s values, above, that is `{Name:
  "gobble", Version: <the build's>}`, the value D3 gives the ACP sign-in.
* **N3 reads the same values.** D3's `Identity: auth.Identity{Name: "gobble", Version}` is built from `SignInOptions`'
  `ClientName` and `ClientVersion`, so every gobble sign-in names one application at one version.
* **N4, N5: nothing.** The SDK's hand-off names the session from the `WithClientInfo` gobble already passes.
* **N6. gobble passes its name to the lookup; the SDK names the session.** `ModelInfoWith` passes `o.ClientName` and
  `o.ClientVersion` to `lookupModelWith`, which adds `llmprovider.WithClientInfo(o.ClientName, o.ClientVersion)` to the
  options of its `catalog.CachedModel` call (`llm/provider/model.go:150-157`) when `ClientName` is set, and the agent's `ModelInfo` hook
  (`internal/cli/agent.go:102`) sets `ClientName` and `ClientVersion`, as the provider's options do. gobble calls no
  `UseIdentity` for the model facts (the owner's ruling on Q2). With the SDK's pending amendment of its 0079-MADR D4,
  `CachedModel` and `LookupModel`'s `listingModel` hand that name to the session before they read its token, and the
  listing a lookup starts in the background names gobble too. If the minor gobble pins at P2 does not carry that
  amendment, P2a's step 0 finds it, and it is a deviation (C4), not a reason for gobble to name the session itself.
* **N7. The logout names gobble.** `LoginOptions` gains `ClientName` and `ClientVersion`; `Logout` calls `UseIdentity`
  with them on the session it builds, before `RevokeOAuthSession`; `internal/cli/auth.go:302` passes `appName` and
  `identity().Version`. D2's rule of which sessions are revoked, made in the PLAN's P4, keeps the call, so a plan
  session revoked there is named the same way.
* **Where in the PLAN.** The SDK minor is pinned by the PLAN's P2 (D1; the fleet sequence's pin table says the same),
  not by its P4: P4 makes `authcontract` a direct requirement and pins acpext `v0.5.0`, committed with 0052-PLAN P3.
  What D9 changes needs only P2's minor, and lands as one new phase, P2a, on its own commit after P2's and before
  0052-PLAN P2 (the owner's ruling on Q1). From P2 on, gobble's providers and listings already name its refreshes (N4,
  N5) with no change of gobble's; no release falls between P2 and P2a.

#### What stays the SDK's, or borrowed, and why

* **Grok's OAuth client id stays the Grok CLI's** (`auth.DefaultGrokOAuthClientID`, `llm/provider/login.go:83`). xAI
  refuses a client id it has not registered, "Unknown or disabled client"
  ([0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md) `:1096`, its amendments of 2026-10-14), and offers no
  registration. It is the borrowed client D2 accepts as risk (`acceptedBorrowed`; Q10). Not possible.
* **OpenAI's browser and device sign-ins keep Codex's client id,** and gobble does not offer them (D2's borrowed-client
  rule; Q10, "refuses Codex's id"). gobble's OpenAI sign-in is the plan, whose client OpenAI issues to gobble under
  `AgentName`, "gobble" (`llm/provider/signin.go:27`; go-llmprovider-sdk 0043-MADR D7, as 0079-MADR D6 cites it), and
  which gobble keeps and reuses (0052-MADR D5). gobble borrows nothing of OpenAI's.
* **The User-Agent's trailer,** `go-llmprovider-sdk/<ver>`, stays on every request (that owner's ruling 3; 0003-MADR
  item 2).
* **A refresh by another process that shares `auth.json`** names that process. Every gobble at or past P2a names
  gobble. A gobble built before it, such as an installed `v0.6.0`, refreshes and revokes under the SDK's name, since the
  identity is never stored (0079-MADR D2); that ends as hosts take `v0.7.0`. Pi shares no file with gobble: an import
  copies its login (N9).
* **The issuer's keys** are fetched under the sign-in that first misses the SDK's process-wide cache (0079-MADR D6).
  Every gobble sign-in is named, so that is gobble's name too.
* **gobble sends no `originator`** (N11), so no gobble test asserts one. Grok's `referrer` and the User-Agent are what
  gobble's traffic shows.

#### The live check

go-llmprovider-sdk's `0079-PLAN-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md` Phase 5, owner-run
before `v1.31.0`, signs in, lists, refreshes and revokes under "gobble" at xAI (a device sign-in) and at OpenAI (a
Codex browser sign-in and the ChatGPT backend). It runs none of gobble's code, and it sends no plan refresh or plan
revocation. By the owner's ruling on Q3, "No separate test", gobble adds no probe of its own: the PLAN's owner-run L1
(P8) ends with `gobble auth logout grok`, a revocation through N7, and 0052-PLAN P6's L6 ends with its logout of a plan
session; each records that the revocation was not refused. A refusal of the name on either is a deviation (C4).

#### The SDK's part of N6, pending

`catalog.CachedModel` (go-llmprovider-sdk `llmprovider/catalog/lookup.go:138`) and `LookupModel`'s `listingModel`
(`:112`, `:187`) read the session's token directly, outside `listingToken`, so 0079-MADR D4's `catalog` hand-off does
not reach a refresh they start, for any consumer. The owner ruled on 2026-11-12, "Amend SDK 0079": go-llmprovider-sdk
fixes it at the source, by a dated amendment of
`0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md` and its PLAN that puts both through D4's
hand-off. That amendment is a later step in that repository and is pending; this record does not change it. gobble's
part is to pass `WithClientInfo` on its call (N6, above).

### Consequences

* Good, because every fact about a method is read from the SDK, and gobble's five openai checks [WIP], three
  predicates and `offers` give way to one policy table (F41).
* Good, because an ACP client, Magic included once it amends magic-cli-remote `0191-MADR-answer-agent-elicitation-forms-on-the-phone.md`, finishes a browser or plan sign-in
  from another device with a pasted callback, and answers the plan's consent, with no channel of gobble's own.
* Good, because one vocabulary of failure kinds runs from the SDK through `--json`, `ErrorData` and
  `_gobble/retry`.
* Good, because terminals and remote clients see the same steps, so a disclosure cannot be skipped on one path.
* Neutral, because `--json` stays output-only: a method that asks a question is refused there, with exit 2.
* Bad, because `--json`'s events, acpext's kind values and `_gobble/retry`'s causes change on the wire, a break for
  any consumer but Magic, which moves in lockstep (QB).
* Bad, because gobble depends on an experimental SDK package (`x/signin`) until the SDK promotes it, and must follow
  its changes in each SDK minor.
* Bad, because `authenticate` can now run for minutes; a client with a short request timeout cancels it.
* Bad, because a form-only client gets the authorize URL as text, not as a link it opens (U8 caveat 4), unless it
  reads `_meta.gobble.step`, as Magic does.
* Neutral, because a fresh sign-in (O2) replaces a stored session without revoking it at its issuer; revocation stays
  `gobble auth logout`'s (QD).
* *(2026-11-11, later.)* Good, because a sign-in from a phone binds no port on the host, so it never contends with
  another agent there for 1455 or 1457, and cannot wait on a listener its person can never reach.
* *(2026-11-11, later.)* Bad, because a remote client cannot sign in to grok by browser: only its device method, until
  the SDK states a paste for grok (U1, its D12 probe); a client that names the browser method gets `unsupported`.
* *(2026-11-12; D9.)* Good, because every request gobble sends to an identity provider names gobble where the vendor
  lets it: Grok's `referrer`, and the User-Agent of every sign-in, refresh, revocation and revocation discovery.
* *(2026-11-12; D9.)* Neutral, because Grok's client id stays the Grok CLI's, which xAI requires, and the User-Agent
  keeps the SDK's trailer.
* *(2026-11-12; D9.)* Bad, because a gobble built before the PLAN's P2a that shares `auth.json` still refreshes and
  revokes under the SDK's name: the name is never stored.
* *(2026-11-12; D9, the owner's ruling on Q2.)* Neutral, because a refresh the model facts start (N6) names gobble only
  once go-llmprovider-sdk's pending amendment of its 0079 records ships in the minor gobble pins; gobble's part, the
  `WithClientInfo` on the call, lands in P2a either way.

### Confirmation

1. archtest refuses `authcontract` in a core package and in `acpclient`, each seen failing on a planted import.
2. `provider.Offered` gives, per provider, exactly the methods of the 0051-PLAN's table, each predicate and the
   word table agree with it, and each policy rule is seen failing on a planted break.
3. A loopback ACP test (`acpclient/acptest.Connect` with an eliciting client) signs in through each step mapping
   of D4, form-only and with URL, passes the SDK's `acptest.ValidateGolden`, and is seen failing when the
   `requestId` is wrong, the capability is not offered, a decline is read as an answer, or an `open_url`'s
   acknowledgement is sent to the engine as an answer for its `seq`. With a credential stored,
   `_meta.gobble.fresh: true` signs in, a failed or cancelled fresh sign-in leaves `auth.json` byte for byte as it
   was, and a completed one replaces the credential; seen failing when the stored credential is removed before the
   sign-in starts.
   *(2026-11-11, later.)* With `_meta.gobble.remote: true`, a browser sign-in binds no port and completes by a paste,
   and a named grok `browser_oauth` is refused `unsupported` before `signin.Start` is called; with `remote` absent or
   `false`, `NoLoopback` is not set; a non-boolean `remote` is -32602; each seen failing on the PLAN's planted breaks.
4. `gobble auth login --json` prints only `method` and `step` lines whose steps decode with the contract's golden
   vectors' types, and never one of the five old events.
5. No `FailureKind` remains; every word `provider.ErrorKind` and `llm.Kind` return is an `authcontract.ErrorKind`.
6. A test asserts every acpext `Command` and every offered method's word is accepted by `gobble auth login`
   (issue #16's criterion), seen failing on a planted `--method device` for openai.
7. `make preflight` and `make race` pass on macOS, the Linux machine and the Windows laptop; `make check-acpext-deps`
   passes; apidiff of acpext from `acpext/v0.4.0` lists only D6's and D7's changes.
8. The owner's live probes of the PLAN are recorded with their outcomes.
9. *(2026-11-12; D9.)* A capturing transport sees `referrer=gobble` on Grok's authorize URL and device request, and a
   User-Agent that begins `gobble/<the build's version> (` and names `go-llmprovider-sdk/` on every sign-in (N1, N3),
   refresh (N4 to N6), revocation and revocation discovery (N7); each seen failing on the PLAN's planted breaks.

## Pros and Cons of the Options

### A. ACP elicitation on `signin.Start`

* Good, because it is ACP's own, stable mechanism (U8), and any ACP client, not only Magic, can drive it.
* Good, because the steps are the contract's, so Magic relays them with no mapping of gobble's.
* Good, because `authenticate`'s answer says, at last, that the credential exists.
* Neutral, because a client that offers no elicitation keeps today's `auth_required`.
* Bad, because Magic must accept request-scoped forms (magic-cli-remote `0191-MADR-answer-agent-elicitation-forms-on-the-phone.md` D1, D6), and would need URL mode for a link rather than text, which its owner declined (its O4, answered "form only" on 2026-11-10).
* Bad, because a long-running `authenticate` is unusual, and clients' timeouts are untested.

### B. A CLI answer channel

* Good, because Magic already runs gobble's CLI for sign-in (magic-cli-remote `0179-MADR-gobble-native-acp-provider.md` D3), so no ACP change is
  needed there.
* Good, because a script can drive it without ACP.
* Bad, because it is a channel of gobble's own, which another ACP client cannot use.
* Bad, because the owner chose against it (Q7).

### C. acpext-carried sign-in

* Good, because it works for a client that offers no elicitation.
* Bad, because it re-invents in `_gobble/*` what ACP's stable elicitation does.
* Bad, because only a client that imports acpext can drive it.

## More Information

### Records amended or superseded in part, applied by the 0051-PLAN's P0 on acceptance

*(2026-11-11: applied by the 0051-PLAN's P0, each as a dated note beside the text it amends. 0031-MADR D8's note was
already there, from the owner's paste in `86ad4a7` (0052-PLAN Deviation 1), and was not made again. 0042-MADR,
unchanged under QC (i), gains a note of QC's answer and one of `acpext/v0.5.0`'s content.)*

* **`0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md`, A2** (`:473-487`). Its bullets "**`authenticate`** with
  such an id answers `{}` when that provider has a credential from any source in gobble's order (0005-PLAN F4-3a
  decision 5). Otherwise it is `auth_required`, its data naming the provider and the command. No secret crosses ACP."
  are amended by D4: absent a credential, a client that offers elicitation is signed in, and a pasted callback
  crosses ACP as a secret, used once. "**No `logout` over ACP.**" stands (QD).
* **The same record, A3** (`:488-489`), "`gobble auth login … --json` prints the login's events as JSON lines":
  amended by D5, the contract's steps.
* **`0005-PLAN-v1-feature-scope.md`, F4-3c decision 6** (`:5137-5143`), the five `--json` events: superseded by D5.
* **`0004-MADR-go-module-architecture.md`, rule 4** (`:475`), "Only `llm/provider` imports
  `github.com/maccavelli/go-llmprovider-sdk`": amended by D1 for `authcontract`.
* **`0031-MADR-sdk-current-releases-and-adoption.md`, D8** (`:405-406` [WIP], `:362-363` [HEAD]; *2026-11-11: `:446-447` on `main` since `86ad4a7`*), "`HasLogin` and `HasKeyLogin` read
  `Descriptor.AuthMethods`": carried out by D2. Its D7 is amended as 0052-MADR says.
* **`0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md`, D3** (`:190-204`), "`llm/provider.FailureKind`
  … is left as it is": superseded by D6; its "acpext's `ErrorKind` constants are these names" holds with the
  contract's spellings.
* **`0042-MADR-acpext-extension-contract-module.md`:** unchanged under QC (i). Under (ii) its A1 (`:292-296`) and D1
  "Only the standard library" (`:142`) would be amended.
* **`0045-MADR-auth-status-json.md`:** D2 and D6 as 0052-MADR says; D7 unchanged.
* **`0025-MADR-provider-choice-and-auth.md`, D6:** its "`--method api-key|browser|device|chatgpt-plan`" words stand
  (D2).

### Records in other repositories this bears on (not changed here)

* magic-cli-remote `docs/decisions/0191-MADR-answer-agent-elicitation-forms-on-the-phone.md` D1 ("session-scoped
  form mode only") and D6 (`elicitation.form` advertised): Magic's to amend for request-scoped forms and URL mode.
* magic-cli-remote `docs/decisions/0179-MADR-gobble-native-acp-provider.md` D3 (phone auth through gobble's CLI):
  Magic's to amend for `authenticate`.
* magic-cli-remote `docs/decisions/0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`
  (proposed) D8: Magic's side of D4, on a sign-in connection of its own, advertising form only, sending
  `_meta.gobble.method` and, on every sign-in the phone starts, `_meta.gobble.fresh: true`; it amends 0179 D3 and 0191
  D1.
* go-llmprovider-sdk `docs/decisions/0074-MADR-one-sign-in-handle-and-method-facts.md`: see "Where the SDK drafts
  differ" below.

### Questions answered by the owner (2026-11-10)

The owner chose the recommended option of each question below on 2026-11-10: QB (a), QC (i), QD (a) and QE (a), and
O2 (a), which magic-cli-remote's draft put to this repository. The options are kept as they were put.

* **QB. acpext's kind values.** (a) *recommended:* acpext's `ErrorKind` and `RetryCause` constants take the
  contract's values at `acpext/v0.5.0`, released with Magic's cut-over, which requires a gobble at or above this
  release; (b) keep acpext's hyphenated values on the wire, and hold a mapping to the contract by a root test, so
  only `--json` uses the contract's words; (c) add the contract's spellings as new constants beside the old,
  emitting the new and deprecating the old for one minor — a compatibility path the owner's lockstep rule does not
  ask for.
* **QC. acpext's long-term auth shape.** (i) *recommended:* provider list only, facts from the contract (D7); (ii)
  acpext imports `authcontract` and carries `[]authcontract.Method`, amending 0042-MADR A1 and `check-acpext-deps`;
  (iii) acpext mirrors per-method facts as its own comparable value type (issue #16's direction), a second copy of
  the contract.
* **QD. ACP `logout`.** (a) *recommended:* out of scope; sign-out stays `gobble auth logout <provider>`; (b) advertise
  `logout`, signing out every provider; (c) advertise it with `_meta.gobble.provider` naming one, an extension ACP
  does not define.
* **QE. The engine of the two non-terminal surfaces.** (a) *recommended:* the wizard's `Driver` for `--json`, which
  keeps P4a's wizard path, its listing check and its save, and gives the `Driver` the owner accepted (SDK-3) its
  caller; `signin.Start` for ACP, as Q7 and go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D3 intend; (b) `signin.Start` for both, so one handle
  serves both and `Driver` has no gobble caller; (c) the wizard's `Driver` for both, so ACP's sign-in also runs the
  wizard's menu logic, which `authenticate` does not need.
* **O2, put by magic-cli-remote `0246-MADR-take-provider-auth-types-from-the-sdk-auth-contract-in-one-cut-over.md`. A fresh gobble sign-in over a stored session.** The owner chose (a) on
  2026-11-10, with the rule that the old credential is replaced only if the new one saves (D4 steps 1 to 3). (a)
  *recommended:* gobble-cli accepts `_meta.gobble.fresh: true` on `authenticate`, signing in whatever is stored; (b)
  Magic runs `gobble auth logout <id>` first, which revokes the old session at the issuer before the new one exists;
  (c) accept it: a phone re-sign-in needs the session cleared on the host.

### Where the report, the audit or the SDK drafts differ from the code

* **The report's §5.5 and §8.4 put ACP elicitation "later"** and recommend `--answers`; the owner chose otherwise
  (Q7), and U8 shows nothing in the pinned SDK delays it.
* **The report's §5.5 says gobble's `--json` "takes answers on stdin"**; under Q7 it does not (D5).
* **The audit's §5.2 says Magic consumes `ErrorData.Command` "(per brief; not re-verified)"**; the report's §2.5
  corrects it: Magic reads `Limit` and `Kind` only.
* **The audit's §B proposes acpext `Consent: "chatgptPlanBackground"`**, gobble's setting name; the report's §6.3 and
  go-llmprovider-sdk `0075-MADR-chatgpt-plan-consent-host-id-and-client-reuse.md` use `chatgpt_plan.background_use`, which this record follows.
* **go-llmprovider-sdk `0074-PLAN-one-sign-in-handle-and-method-facts.md`'s exported-names table, as first
  drafted,** justified `wizard.Options.Driver` by "gobble's `--json` and ACP `authenticate` paths, which run
  `wizard.ResolveCredential`", and `signin.Start` by "gobble's `--json` steps (Q21) and its ACP elicitation (Q7)":
  each surface claimed by both. Under QE (a) gobble calls `Driver` for `--json` and `signin.Start` for ACP only. The
  table was corrected to say so on 2026-11-10.
* **go-llmprovider-sdk `0073-MADR-publish-an-auth-contract-module.md` D2, as first drafted,** said "No step needs a
  schema construct that form mode lacks", and mapped `open_url` and `show_code` onto URL mode. A `paste` is a secret
  and the stable form schema has no secret format (U8 caveat 1); URL mode cannot reach a form-only client such as
  Magic (U8 (d), caveat 4). Its D2 now carries `secret` on the step and leaves the ACP mapping to D4 here, both
  columns of it, on 2026-11-10.
* **The contract's `ErrorKind` has no word for gobble's `empty`** (a listing that recommended nothing,
  `llm/provider/list.go:119`); D6 maps it to `other`.
* **The archtest prefix rule** (`internal/archtest/check.go:249`, `:308`) would refuse `authcontract` in
  `acpserver`; neither the report nor the SDK drafts name it. D1 amends it.

### Not verified, with the probe that settles it

* Whether a client's request timeout lets a ten-minute `authenticate` run: magic-cli-remote's ACP client is Magic's
  to measure; gobble's acceptance runs `gobble acp` under a scratch client with no timeout.
* Whether acp-go-sdk2 logs a request's params at any level gobble sets (`acpserver/serve.go:45`): the PLAN's test
  greps the run's log for a planted callback.
* Whether accounts.x.ai offers a paste (U1): go-llmprovider-sdk `0074-MADR-one-sign-in-handle-and-method-facts.md` D12's owner-run probe; until it does,
  grok's browser stays `browser_loopback`.

### Revisions of 2026-11-11

Corrections the owner directed on 2026-11-11, each checked against the tree first. Every one is a dated note beside the
text it corrects. No decision of D1 to D8 is reversed; D6's note carries its respelling to the agent's own retry
cause, which D6 as written did not reach. The record stays `proposed`.

* **The base and the archive.** `main` is `17d1605`; the archive branch is also on origin (the note after the "[HEAD]"
  paragraph).
* **Anchors that `86ad4a7` moved.** 0031-MADR D8 is at `:446-447`; 0031-PLAN P4b-2 step 4's line is `:347`.
* **The SDK records re-pinned** to their text of 2026-11-11 and the owner's rulings of that day ("What the SDK
  proposes"): the contract's `oauth`, `at_start`, `Method.command` and `Answer.secret`; the kept `Failure*` names;
  `SessionKind()`; `NoLoopback`; `OAuthClients`; `StoredConsent`; what is deferred; the Go-floor record.
* **U8 at acp-go-sdk2 `v1.36.0`,** by
  [0054-MADR-go-1-27-4-and-sibling-pin-refresh.md](0054-MADR-go-1-27-4-and-sibling-pin-refresh.md): each cited line, the schema version,
  and the `systemPrompt` gate.
* **D2:** the session's kind is read through `(*OAuthSession).SessionKind()`.
* **D3:** gobble sets no `NoLoopback` and no `OAuthClients`.
* **D4:** grok's browser sign-in has no paste; three pastes in all; a pasted denial ends the sign-in `denied`.
* **D6:** the agent's own retry cause takes the contract's word; the PLAN gains the files and a test.
* **D7:** magic-cli-remote has not taken `acpext/v0.4.0` yet.
* **`consulted`:** magic-cli-remote's revisions corrected.

**Later on 2026-11-11, the owner's rulings relayed by the cross-repository coordinator.** Each is a dated note beside
the text it changes; the record stays `proposed`.

* **`NoLoopback`** (go-llmprovider-sdk 0074-MADR D4 reversed that day): the "What the SDK proposes" bullet; D3's note,
  where gobble now sets it for a remote sign-in, superseding "gobble sets it nowhere"; D2's remote rule; D4 step 1's
  `_meta.gobble.remote` and the `unsupported` refusal of a named method that only the loopback completes; step 3;
  Confirmation 3; two Consequences.
* **`_meta.gobble.remote` and `ErrorData.Consent`** (D7): `AuthenticateMeta.Remote`, the table of `_meta.gobble`
  members, and the `consent` member, both in `acpext/v0.5.0`; `consent` is never set for an answer that could not be
  read.
* **The consent's text** (D2's note, D4's advertised methods): gobble's wording in `_meta.gobble.methods`, the same as
  the sign-in's `confirm` step.
* **The SDK's Go-floor record** is a file now: go-llmprovider-sdk `0077-MADR-go-1-27-4-for-go-2026-6859.md`.
* **`consulted`:** these rulings.

**Accepted on 2026-11-11,** by the owner's word of that day (the note under Decision Outcome). The status is
`accepted`; no decision and no note changed with it.

### Revisions of 2026-11-12

The owner's direction of 2026-11-12, "Where possible gobble auth would send gobble as the name/referrer not the sdk",
with go-llmprovider-sdk `0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md`, read against this
repository at `c47f4a7`. Each change is a dated note or a new section. No decision of D1 to D8 is reversed. The record
stays `accepted`. ~~D9 is proposed until the owner answers Q1 to Q3 below and approves the PLAN's P2a.~~
*(2026-11-12, later: the owner answered Q1 to Q3 that day, each noted under its question below, and D9 is accepted
2026-11-12. D9's text, not yet committed, was brought to the rulings in place: N6's row and its bullet, the PLAN's
place for P2a, the live check, and the SDK's part of N6.)*

* **D9, new:** gobble's sites (N1 to N11), what gobble does at N1, N6 and N7, what stays the SDK's or borrowed, the
  live check, and the SDK's part of N6, pending in go-llmprovider-sdk.
* **D3:** a note that the wizard's sign-ins carry gobble's `Identity` too.
* **Consequences:** four bullets. **Confirmation:** item 9.
* **Front matter:** `date` and `consulted`.
* **Lines kept in place.** Every insertion is below `:350`, which go-llmprovider-sdk's 0079-MADR cites (its C41,
  pinned at `c47f4a7`).

### Questions for the owner (2026-11-12), ruled the same day

The recommended option of each is first. The options are kept as they were put; the owner's ruling follows each.

* **Q1. Where the gobble change lands.** (a) *recommended:* a new phase, P2a, on P2's commit and before 0052-PLAN P2:
  small, and right after the pin that provides `UseIdentity`; it costs one more commit, and a line in the fleet
  sequence's stage 7 (`0006-PLAN-sequence-the-auth-contract-effort-across-four-repositories.md`, outside this
  repository); (b) inside P2: no commit between the pin and the naming, but P2's acceptance, "no code change but
  `internal/archtest/check.go`, its test and `scripts/check_libs.py`", is amended, and the pin commit carries behaviour;
  (c) inside P4, committed with 0052-PLAN P3, where `Logout` is rewritten anyway: one pass fewer over `login.go`, but
  the effort's largest commit grows, and `main` revokes under the SDK's name from P2 to P4.
  *(2026-11-12, the owner's ruling: "New phase P2a", option (a). P2a lands after P2, on its own commit.)*
* **Q2. The model-facts refresh (N6).** (a) *recommended:* gobble names the session in `Options.login`, and
  `ModelInfoWith` passes `WithClientInfo`: one place names every session gobble builds, whatever the SDK hands over; it
  costs a call that repeats the SDK's hand-off for N4 and N5; (b) go-llmprovider-sdk extends its 0079-MADR D4 to
  `CachedModel` and `LookupModel`, and gobble passes `WithClientInfo` alone: nothing repeated, but it costs an SDK
  record change and a phase on the `v1.31.0` chain, which is the effort's critical path; (c) N6 is left as it is, and
  recorded: no work, but a refresh the model facts start can name the SDK.
  *(2026-11-12, the owner's ruling: "Amend SDK 0079", option (b). The SDK fixes the gap at the source: `CachedModel`
  and `LookupModel`'s `listingModel` go through its D4 hand-off, by a dated amendment of go-llmprovider-sdk
  `0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md` and its PLAN, a later step there and
  pending. gobble calls no `UseIdentity` for the model facts. The hand-off names the session only from a
  `WithClientInfo` among the call's options, and gobble passes none at `llm/provider/model.go:150-157`, so gobble's
  part is that option on the call (N6).)*
* **Q3. The live check.** (a) *recommended:* none of gobble's own: L1 (the PLAN's P8) ends with a logout, and
  0052-PLAN P6's L6 logs a plan session out, each under gobble's name and each recording that the revocation was not
  refused; no extra sign-in and no quota; (b) a dedicated probe in P2a, at xAI and OpenAI: proves gobble's three sites
  live before the rest of the effort, but costs the owner two more sign-ins and repeats the SDK's Phase 5; (c) none at
  all, the SDK's Phase 5 standing and L1 and L6 unchanged: no cost, but a refusal of gobble's name on the plan's
  revocation, which Phase 5 does not send, would show only as a warning that no record asks anyone to read.
  *(2026-11-12, the owner's ruling: "No separate test", option (a). No dedicated gobble probe; L1 and L6 each end with
  a logout and record that the revocation was not refused.)*

### Related

* PLAN: [0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md](0051-PLAN-sign-in-through-acp-elicitation-on-the-sdk-auth-contract.md).
* The ChatGPT plan, its consent, host id, client and the WIP:
  [0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md](0052-MADR-finish-the-chatgpt-plan-sign-in-on-the-sdk-engine.md).
* [0042-MADR-acpext-extension-contract-module.md](0042-MADR-acpext-extension-contract-module.md),
  [0045-MADR-auth-status-json.md](0045-MADR-auth-status-json.md),
  [0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md](0049-MADR-structured-quota-and-rate-limit-errors-for-acp-clients.md),
  [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md).
* maccavelli/gobble-cli#16.
* *(2026-11-12)* The application's name past the sign-in (D9): go-llmprovider-sdk
  `docs/decisions/0079-MADR-name-the-app-on-refresh-revocation-and-the-chatgpt-originator.md` and its PLAN; gobble's
  name on its requests, [0003-MADR-gobble-product-identity.md](0003-MADR-gobble-product-identity.md).
