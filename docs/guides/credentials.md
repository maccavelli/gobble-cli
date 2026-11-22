# Credentials

gobble needs a key, or a login, for the provider it talks to. This page says where gobble looks for
one, how to store one or sign in, and what gobble says when a credential is missing or refused.
gobble keeps keys the way Pi does, in an `auth.json` of Pi's shape, so a Pi `auth.json` mostly
works unchanged.

## Where a key comes from

For each provider, gobble takes the first of these that has a key:

1. **`--api-key -`**, for one run. The flag's value is the single dash `-`, always: gobble reads the
   key from stdin, or asks for it when stdin is a terminal, and never writes it anywhere. It needs
   `--model`, which names the provider.
2. **`auth.json`** in gobble's config directory (`gobble config path` prints it on its `config:`
   line). An entry for a provider owns it: gobble then reads nothing further for that provider.
3. **`models.json`'s `apiKey`** for the provider ([guide](models-json.md)).
4. **The provider's variable,** such as `OPENAI_API_KEY` or `ANTHROPIC_API_KEY`.

With no `--model` and no `defaultProvider`, gobble uses the first provider, in its own order, that
has a key from any of these.

A key in `auth.json` or `models.json` can name a variable or run a command, as Pi's can:
`"$OPENAI_KEY"`, `"${OPENAI_KEY}"`, or `"!pass show openai"`, which keeps the key out of the file.
A stored key's command runs once per run; `models.json`'s runs each time a provider is built. An
error names the provider and the variable or command, never the key.

gobble refuses a key written on the command line, since process listings and shell history show
it:

```text
$ gobble --model openai/gpt-6-luna --api-key sk-test-only -p hello
Error: --api-key takes -: a key on the command line shows in process listings and shell history; pipe it, or type it when asked
```

`--api-key` takes only `-`: the dash is the flag's whole value, and it means "read the key from
stdin, or ask for it when stdin is a terminal". For one run, with `--model` naming the provider,
pipe the key — stdin is the key, never the input, so the prompt's words go after `-p`:

```text
echo sk-test-only | gobble --model openai/gpt-6-luna --api-key - -p hello
```

The key is read from stdin, spent on that one run, and stored nowhere.

## Storing a key

`gobble auth login <provider> --api-key -` stores a key in `auth.json`. The flag's value is the
single dash `-`: gobble reads the key from stdin, or, when stdin is a terminal, asks for it — and
what is typed is not echoed. The file is created readable by you alone, and every change to it is
locked, so two gobble processes never lose each other's entries.

On a terminal, run it alone and type the key when asked:

```text
$ gobble auth login openai --api-key -
API key for openai: 
Saved API key for openai
```

In a script, or wherever the key can be piped from, pipe it:

```text
$ echo sk-test-only | gobble auth login openai --api-key -
Saved API key for openai
```

Pi's provider names are accepted here too: `gobble auth login anthropic` stores claude's entry,
`google` gemini's and `xai` grok's.

`gobble auth status` shows where each provider's key comes from, and never a key:

```text
$ gobble auth status
provider      credential
gemini        none
openai        auth.json
claude        none
grok          none
opencode-zen  none
opencode-go   none
huggingface   none
kilo          none
together      none
```

A sign-in's row also says what gobble keeps of its consent. A ChatGPT plan session ends with
`background use allowed`, `background use declined` or `background use not asked`, the answer to the question
[below](#allowing-background-requests-under-a-chatgpt-plan) asks; another application's ChatGPT login says
`not used` and has no such words:

```text
openai        auth.json, an OAuth login, expires 2026-06-09 05:13, background use not asked
```

`gobble auth status --json` gives the same rows as data, for a program such as a remote client. It prints one
object, sends no request, runs no `!command` key, and writes nothing to stderr:

```text
$ gobble auth status --json
{
  "providers": [
    {
      "provider": "gemini",
      "source": "env",
      "variable": "GEMINI_API_KEY",
      "usable": true
    },
    {
      "provider": "openai",
      "source": "auth.json",
      "authType": "api_key",
      "usable": true
    },
    …
  ]
}
```

| Field | What it is |
| :--- | :--- |
| `provider` | gobble's id for the provider, one row for each provider that takes a key, in the order `auth status` lists them |
| `source` | where the credential comes from: `auth.json`, `models.json`, `env` or `none` |
| `variable` | with `env`, the variable's name |
| `authType` | with `auth.json`, the entry's type: `api_key` or `oauth` |
| `usable` | whether gobble would use it; a `!command` key counts without being run |
| `reason` | why an `auth.json` entry is not usable: `oauth_unused` (a sign-in gobble does not use: one for a provider gobble has none for, or one of a kind it does not sign in to, such as another application's ChatGPT login), `empty_key` or `unknown_type` |
| `expires` | a sign-in's expiry, in UTC, as RFC 3339 |
| `sessionKind` | with `authType` `oauth`, the stored session's kind, as go-llmprovider-sdk gives it, inferred for a session saved without one: `chatgpt_plan`, `grok`, `chatgpt` (another application's ChatGPT login) or `kilo_device`; a kind a program does not know is shown, not refused; absent when the SDK names no kind for the session, such as an OAuth entry for Claude |
| `consents` | only when the session holds an answer: an object from a consent's id to `granted` or `declined`, such as `{"chatgpt_plan.background_use": "declined"}`; a consent with no member is not asked |
| `consentTerms` | only when the stored session is of the kind an offered sign-in method with consents signs in to, which is the ChatGPT plan's alone: an array of that method's consents, answered or not, each with its `id`, `required`, `text`, gobble's wording, the words the sign-in's question uses, and `links`, `rel` and `url`, the usage page |

A ChatGPT plan session's row, answered `declined`:

```text
{
  "provider": "openai",
  "source": "auth.json",
  "authType": "oauth",
  "usable": true,
  "expires": "2026-06-09T10:13:20Z",
  "sessionKind": "chatgpt_plan",
  "consents": {"chatgpt_plan.background_use": "declined"},
  "consentTerms": [{"id": "chatgpt_plan.background_use", "required": true, "text": "…", "links": [{"rel": "usage", "url": "…"}]}]
}
```

A client that shows the consent reads `consentTerms` for the words and the link, and changes the answer with
`gobble auth consent` ([below](#allowing-background-requests-under-a-chatgpt-plan)). An OpenAI entry of another
application's ChatGPT login is `usable: false`, `reason: "oauth_unused"`, with `sessionKind` `chatgpt` and neither
`consents` nor `consentTerms`; a Grok session has its `sessionKind` and neither.

A field that does not apply is left out. If `auth.json` or `models.json` exists but cannot be read, gobble cannot say
where any key comes from: `--json` prints an error naming the file, never a value from it, and exits 2:

```text
$ gobble auth status --json
{
  "error": {
    "file": "auth.json",
    "message": "<config>/auth.json: invalid auth.json: jsontext: unexpected EOF within \"/openai\" after offset 49"
  }
}
```

Without `--json`, gobble warns and lists the rest. A later release may add fields or values, and a program should
ignore what it does not know; one is never renamed or removed without notice
([0045-MADR](../decisions/0045-MADR-auth-status-json.md)).

`gobble auth check` says whether a provider has a usable key, for scripts: it prints `ready`,
`not_ready` or `invalid`, and exits 0, 1 or 2. `--json` gives the reason too:

```text
$ gobble auth check --provider openai
ready
$ gobble auth check --provider claude --json
{
  "status": "not_ready",
  "provider": "claude",
  "reason": "credentials_not_configured"
}
```

`--method api-key` does the same: `--method` names how to sign in, with no menu, `api-key`, `browser`,
`device` or `chatgpt-plan`, as the provider offers them. A way the provider does not offer is refused, naming its ways:

```text
$ echo sk-test-only | gobble auth login anthropic --method api-key
Saved API key for claude
$ gobble auth login grok --method nope
Error: grok signs in with --method api-key or browser or device, not "nope"
```

`gobble auth logout <provider>` removes the provider's entry. Variables and `models.json` are left
alone:

```text
$ gobble auth logout openai
Removed stored API key for openai. Environment variables and models.json config are unchanged.
```

Pi's `anthropic`, `google` and `xai` name gobble's `claude`, `gemini` and `grok` here too.

### An Anthropic workspace

An Anthropic API key that is not scoped to a workspace needs one named with each request. gobble keeps it beside
the key in `auth.json`, as `"workspace"`, and sends it to Claude alone:

```text
$ gobble auth login anthropic --api-key - --workspace wrkspc_01ABCDEF
Saved API key for claude
```

`/login` and `gobble setup` ask for the workspace when Claude refuses a key without one, and keep the one that
lists. `gobble acp --login claude` asks for it after the key, empty for none. `ANTHROPIC_WORKSPACE_ID` is not
read. A turn refused for want of a workspace says how to name one, and does not show the service's message
(0035-MADR D5).

## Signing in

Three providers let you sign in with your account instead of pasting a key:

- **Grok.** `gobble auth login grok` signs in with your SuperGrok or X Premium subscription. gobble
  prints the sign-in URL, then opens it in your browser; on a machine whose browser cannot reach
  gobble, paste the URL the browser ends on when asked. `gobble auth login grok --device` shows a
  code to enter at xAI instead, for a machine with no browser at all; the line that shows the code
  names when it stops working, `(expires 14:05)`, in your machine's local time. `/login grok`,
  `gobble setup` and the first-run offer open the browser and show the code the same way. Either way gobble says
  `Logged in to grok`, keeps the login in `auth.json`, and refreshes it as it runs. gobble signs in
  with the Grok CLI's public client id: xAI offers third-party tools no id of their own, so if xAI
  restricts that id, the login stops working, and a key still works.
- **Kilo.** `gobble auth login kilo` shows a code to enter at Kilo, with its expiry as Grok's does. Kilo answers with a key, which
  gobble stores as Kilo's API key and says `Saved API key for kilo`.
- **OpenAI, with your ChatGPT plan.** `gobble auth login openai` signs in under gobble's own name,
  `gobble`, and not as the Codex client; `--method chatgpt-plan` names it with no menu. Before the
  browser opens, gobble shows what it stores and where, how to remove the sign-in, and what a
  background request is, and asks whether gobble may make background requests under your plan: allow
  them, do not allow them (the default), or cancel the sign-in. Then it prints the sign-in URL and
  opens it; on a machine whose browser cannot reach gobble, paste the address the browser ends on when
  asked. The question is the same wherever the sign-in runs on a terminal: `/login openai`,
  `gobble setup`, the first-run offer, and `gobble acp --login openai`. With no terminal the question
  cannot be asked: the sign-in stops before the browser opens, saves nothing and says
  `this sign-in asks a question, and there is no terminal to ask it on`. With `--json` it is refused
  before it starts, exit 2, unless the stored plan session already holds an answer
  ([below](#signing-in-from-an-editor-or-magic-cli-remote)). gobble keeps the login and your answer
  in `auth.json`'s `openai` entry, and refreshes it as it runs; `gobble auth status` shows the session's kind
  and your answer. An ACP client that can show a form, such as a phone, is asked the same question, in the same
  words, as a form before it is shown any address ([below](#signing-in-from-an-acp-client)). gobble also keeps this
  installation's host id, a random `urn:uuid`, in `host-id` in its state directory (`gobble config path`
  shows the directory), registers the plan's client under it, and keeps the issued client in
  `clients.json` beside it, so that signing out and in again reuses the client instead of registering
  another. The host id is written once, to a temporary file renamed into place under the lock file
  `host-id.lock` beside it, so two gobble processes keep one id; an empty `host-id`, left by a write that
  was interrupted, is replaced, and one that holds anything but a `urn:uuid` is an error and is left as
  it is. When the state directory cannot be read or written, the sign-in goes on with no host id of
  gobble's own, the SDK makes one for that sign-in, and gobble warns once: on stderr for a terminal
  sign-in, in the agent's log for an ACP one. Neither warning names an id.

gobble names itself to xAI's sign-in service in the requests it makes for a Grok login, as
`gobble/<version>`, the running build's version, ahead of the SDK's own name: such a request's
User-Agent reads `gobble/<version> (<os>; <arch>) go-llmprovider-sdk/<version>`. That covers each
request of the sign-in, whose authorize and device requests also carry `referrer=gobble`; the
refresh of a stored login, whether a request to Grok, a model listing or a model lookup starts it;
and `gobble auth logout grok`'s discovery and revocation. gobble sends no `originator`. A ChatGPT plan
session's logout names gobble the same way.

A login owns its provider in `auth.json`, as a stored key does; `--api-key -` still wins for one
run. `gobble auth status` shows when a login's access expires, and `gobble auth check` reports it as
`oauth`:

```text
$ gobble auth status
provider      credential
gemini        none
openai        none
claude        none
grok          auth.json, an OAuth login, expires 2026-06-09 05:13
opencode-zen  none
opencode-go   none
huggingface   none
kilo          none
together      none
$ gobble auth check --provider grok --json
{
  "status": "ready",
  "provider": "grok",
  "authType": "oauth"
}
```

`gobble auth logout grok` revokes the login at xAI, then removes it. If xAI cannot be reached, gobble
warns that the login was not revoked, and removes it anyway. `gobble auth logout openai` does the same
for a ChatGPT plan session, at the issuer that signed it in; an OpenAI entry of another kind, another
application's ChatGPT login, is removed and never revoked. The discovery and the revocation it
sends name gobble and its version, as a sign-in's requests do:

```text
$ gobble auth logout grok
Logged out of grok
```

Any other provider takes a key:

```text
$ gobble auth login anthropic
Error: claude has no login; use --api-key -
```

An OAuth entry gobble does not sign in to, for another provider such as one copied from Pi, or an
OpenAI entry of another application's ChatGPT login, is kept, and gobble says it does not use it.

Gobble refuses such an entry when it loads one. A session a running gobble already holds can still adopt one
that another program writes into gobble's own `auth.json` while gobble runs, at its next refresh, and then
sends that login's token.

### Allowing background requests under a ChatGPT plan

A session's title, which a small model writes from your first prompt, and an automatic compaction are
background requests: gobble makes them, and you did not ask for them. Your own turns and `/compact` are
not. Under a ChatGPT plan session gobble makes a background request only when your answer to the
sign-in's question is "allow". With the answer declined, or never given:

- no title is asked, and no compaction is made at its threshold, with nothing shown;
- a turn whose context outgrows the model's window ends with an error that names `/compact` and the
  command that allows the requests, `gobble auth consent openai chatgpt_plan.background_use allow`.

gobble reads the answer from `auth.json` at each such request, so a change applies from the next
request, in a running session too. It is the stored sign-in's answer, whichever credential a run uses: with `--api-key -` for OpenAI
while a ChatGPT plan sign-in is stored, titles and automatic compactions are held back until the consent is granted, although those
requests would go out with the key. An OpenAI key in the environment or in `models.json` is not used while that sign-in is stored,
since `auth.json`'s entry owns the provider, so those requests go out under the plan and the consent governs them as above. An answer that cannot be read, because `auth.json` does not parse,
counts as not granted: nothing is made, the log says so, and a turn that must end gives the error of
the store and names `/compact` alone.

`gobble auth consent` changes the answer, or lists it, with no question and no read of stdin, so a
program may run it:

```text
$ gobble auth consent openai
openai chatgpt_plan.background_use not asked
$ gobble auth consent openai background allow
openai chatgpt_plan.background_use granted
$ gobble auth consent openai chatgpt_plan.background_use deny
openai chatgpt_plan.background_use declined
```

The consent is named by its id, `chatgpt_plan.background_use`, or by gobble's word for it,
`background`; the third word is `allow` or `deny`. With no consent and no third word it lists the
consents of the stored sign-in, one line each, with `granted`, `declined` or `not asked`. A change
prints one line naming the provider, the consent's id and the answer stored. Exit 0 is a stored answer
or a list; exit 2 is a usage error: an unknown provider, a provider with no stored sign-in, a consent
the stored sign-in does not have, a consent with no `allow` or `deny`, or any other third word; exit 1
is an `auth.json` that cannot be read, parsed or locked, with nothing written. Every error is one line
on stderr, with nothing on stdout.

### The one-time migration of `auth.json`

A gobble build of the ChatGPT plan sign-in tidies `auth.json` once, when a command starts. Every command
does it, except `gobble auth status` and `gobble auth check`, with or without `--json`, which read the file
as it is; shell completion never does. A development build does it as a release would. It does three
things, and only what is still to do:

- **A dormant key leaves a sign-in.** An OAuth entry never holds a `key` or a `workspace`; one left there
  from an earlier key is a secret beside the tokens that nothing uses. gobble removes both members from
  every OAuth entry, whatever their value and whatever the provider. Before that it copies `auth.json`,
  unchanged, to `auth.json.bak-<time>` beside it, the time in UTC (`auth.json.bak-20261122T090807Z`),
  readable by you alone as `auth.json` is. The log file gets one Info record per provider, naming the
  provider and the backup's file name, never a value. At `--log-level warn` or `error` that record is not
  kept, and the backup is the trace. gobble never removes a backup: delete it once you no longer want it.
  Every other entry and field keeps its bytes. The file is written with LF line ends, two spaces deep and
  a final newline, as gobble always writes it.
- **A ChatGPT plan sign-in's ids move to the state directory.** A plan sign-in stored by an earlier build
  holds its host id and client id in `auth.json`. With no `host-id`, or an empty one, gobble writes the
  entry's host id there. When the entry holds both ids, `host-id` then holds the same host id and
  `clients.json` has no `openai` record, it writes that record as well. An existing `host-id` or record is
  never replaced. A host id that is not a `urn:uuid`, in the entry or in `host-id`, is an error, and
  nothing is adopted.
- **The old background setting becomes the consent.** An earlier build kept the answer as
  `chatgptPlanBackground: true` in the global `settings.yaml`. When that file says so, and the plan
  sign-in has no answer and no `kind` member, which only that build left, gobble records the consent as
  granted, the same as `gobble auth consent openai background allow`. A project's `settings.yaml` is not
  read. `settings.yaml` is not edited; gobble reports the key as one it does not use. To withdraw the
  grant, run `gobble auth consent openai background deny`.

With nothing to do, gobble writes nothing and creates nothing, not even a lock file. With something to
do, it waits for `auth.json`'s lock as a sign-in's save does, up to 30 seconds. A symbolic link at
`auth.json` is followed: its target is migrated and backed up beside the target, and the link stays.

A migration that fails leaves the file as it was, or with what it finished, and the command runs on. The
next start does what remains. The failure is one warning on stderr, `auth.json migration: …`, and a
record in the log file; under `gobble auth login --json` and `gobble acp` it is the log record alone. An
`auth.json` that cannot be read or parsed is the exception: the command that reads the file reports
that itself, so the migration says nothing on stderr and only the log keeps it. A `settings.yaml` that
cannot be read stops a migration that has a consent to carry, with nothing written. When gobble's
directories cannot be resolved, the migration is skipped with one warning, `auth.json not migrated: …`.

## Signing in from a session

In a session, `/login` asks which provider to sign in to, each shown with where its credential comes
from now, then how, when the provider offers more than a key: Grok's browser and device sign-in,
Kilo's device sign-in, OpenAI's ChatGPT plan. A browser sign-in is opened in your browser, as `gobble auth login`
opens it, and a device code is shown with its expiry. `/login claude` asks no provider. A typed key echoes a `*` for each character,
and gobble checks it by listing the provider's models, which spends nothing: a key the provider
refuses is asked for again, up to three keys, and a key the listing cannot check is saved with a
note. Then gobble lists the provider's models, numbered for `/model N`. Ctrl+C leaves the sign-in,
not gobble. Here gobble had no credential, and no network, and a key was typed at the key prompt:

```text
Enter your Claude (Anthropic) API key: ******************
live model listing for Claude (Anthropic) is unavailable (the request failed); search covers the built-in catalog only
Saved API key for claude
```

Every question is read through one terminal input for the whole session, so what one answer leaves is
the next question's. An answer may end with any Enter a terminal sends: `\r`, `\n` or `\r\n` is one
end. The key prompt waits for the key, whichever Enter ended the menu's answer. Keys typed ahead
answer the next question of the same sign-in; what is still unread when the session's prompt comes
back is dropped, and so is what was typed into a prompt gobble gave up, such as a paste prompt that
a browser sign-in finished while it waited: on Linux and macOS gobble also clears the terminal's own
queue then, so that text is never a turn, an answer, or the shell's input after gobble exits. An
empty key cancels the sign-in.

On Linux and macOS gobble waits for a prompt's keys without reading the terminal until a prompt
asks, so a key typed while a sign-in finishes is kept for the question that asks. When a terminal
cannot be waited on that way, gobble reads it as it did before and says so once, before the next
prompt's text:

```text
Warning: gobble cannot wait on this terminal without reading it (<error>); keys typed while a sign-in finishes may be lost.
```

On a Windows console, Ctrl+C at a key prompt is the key itself: it cancels the sign-in and gobble
goes on. A menu or an offer is read a line at a time, and the console ends that read with nothing;
gobble, having taken the interrupt, reads again within a tenth of a second, so Ctrl+C there cancels
the question, as it does on other systems, and does not end the session.

`/logout` lists only the credentials `auth.json` holds and removes the one chosen, as
`gobble auth logout` does; it says when the provider's variable is still set, since that still gives
it a credential. A provider with no credential is offered as `/login <provider>` in `/providers`,
`/model` and `/login`'s own list.

With no credential at all, a session on a terminal asks first. A sign-in that is cancelled or fails,
an empty key or Ctrl+C included, brings the offer back, `[y/N]` this time, until a sign-in succeeds or
the offer is declined with `n`, Enter or Ctrl+C; the end of input ends it too. Here the first round
was cancelled with an empty key, and the second saved a key:

```text
No provider has a credential yet. Sign in to one now? [Y/n]:
Sign in to which provider?
  1  Gemini (Google) (gemini)     none — /login gemini
  2  OpenAI (openai)              none — /login openai
  3  Claude (Anthropic) (claude)  none — /login claude
  4  Grok (xAI) (grok)            none — /login grok
  5  OpenCode Zen (opencode-zen)  none — /login opencode-zen
  6  OpenCode Go (opencode-go)    none — /login opencode-go
  7  Hugging Face (huggingface)   none — /login huggingface
  8  Kilo Gateway (kilo)          none — /login kilo
  9  Together AI (together)       none — /login together
Number [1]: 3
Enter your Claude (Anthropic) API key:
Sign-in cancelled; nothing was saved.
No provider has a credential yet. Sign in to one now? [y/N]: y
Sign in to which provider?
  1  Gemini (Google) (gemini)     none — /login gemini
  2  OpenAI (openai)              none — /login openai
  3  Claude (Anthropic) (claude)  none — /login claude
  4  Grok (xAI) (grok)            none — /login grok
  5  OpenCode Zen (opencode-zen)  none — /login opencode-zen
  6  OpenCode Go (opencode-go)    none — /login opencode-go
  7  Hugging Face (huggingface)   none — /login huggingface
  8  Kilo Gateway (kilo)          none — /login kilo
  9  Together AI (together)       none — /login together
Number [1]: 3
Enter your Claude (Anthropic) API key: ******************
live model listing for Claude (Anthropic) is unavailable (the request failed); search covers the built-in catalog only
Saved API key for claude
```

Declined, the session goes on as before, with the warning naming the variables, once. What was typed
before a repeated offer is dropped, and the terminal's own queue is
cleared after the offer's text is written. `/login` and `gobble setup` ask once: a sign-in cancelled
there ends with its cancel line and the session's prompt, or setup's error. `gobble setup` asks the
same questions at any time, on a terminal; without one it says to use `gobble auth login`:

```text
$ gobble setup < /dev/null
Error: gobble setup asks its questions on a terminal; gobble auth login <provider> --method signs in without one
```

## Signing in from an editor or magic-cli-remote

An ACP client, such as an editor or magic-cli-remote, runs gobble as `gobble acp`. gobble tells it,
when they connect, which credentials it can use: one method per provider that takes a key, named
after the provider, with the gobble command that stores its credential, and the sign-in methods the
provider offers. Asked to authenticate with one, gobble answers from the credentials it already has, in
the order above; with none, a client that can show a form is signed in
([below](#signing-in-from-an-acp-client)), and any other is told which command to run. No key or token
crosses ACP but what a person types or pastes into a form, which goes to the sign-in alone.

A client that can open a terminal also sees a sign-in for each provider, which runs
`gobble acp --login <provider>` there. For Grok, Kilo and OpenAI it shows the menu `/login` shows, the provider's
offered methods with the API key among them, asks the ChatGPT plan's question on that terminal, and opens a browser
sign-in in the browser; `--device`, or a run with no terminal to ask on, signs in by the provider's default method
with no menu, as `gobble auth login` does. Any other provider's key is typed on that terminal. (No key is asked over ACP
itself; this is a terminal.) Run it yourself the same way:

```text
$ echo sk-test-only | gobble acp --login anthropic
Saved API key for claude
```

gobble does not sign out over ACP; use `gobble auth logout <provider>`.

### Signing in from an ACP client

Each agent method `initialize` advertises carries `_meta.gobble.methods`: the provider's offered sign-in
methods, each the auth contract's `method` object (`id`, `kind`, `completion`, `paste`, `yields`,
`client_identity` and, for a method that asks one, its `consents`, in gobble's wording). A client reads
from them, before it asks, how a method completes, what it leaves and whose client it signs in as. A
provider that takes a key alone lists its `api_key` method, which gobble never asks for over ACP.

`authenticate` with a provider's id reads `_meta.gobble`, strictly: a member gobble does not know, or a
value of the wrong type, is -32602 with the reason in the error's data, and nothing starts. `method`,
`fresh` and `remote` each refuse `null`: a value or leaving the member out is accepted. A `null`
`_meta.gobble` is an absent one, as ACP reads a null container, so the request behaves as one with no
`_meta.gobble`.

| Member | Meaning |
| :--- | :--- |
| `method` | A method's contract id from that list: `browser_oauth`, `device_code`, `chatgpt_plan`. One the provider does not offer is -32602. |
| `fresh` | `true` signs in whatever is stored; absent or `false` does not. The stored credential stays exactly as it is until the new one is saved, in one write of `auth.json`, and gobble saves only after the provider's model listing has not refused the new credential: a sign-in that fails, is declined, is cancelled, or whose credential the provider refuses (a 401, or a refresh the issuer refuses) leaves the stored credentials as they were, and the replaced session is not revoked. A listing that fails for another reason, a 403, an outage, a rate limit or a bad request, does not stop the save, as in the terminal's sign-in. If the listing renewed the new session, the renewed tokens are the ones saved. |
| `remote` | `true` says that the person completing the sign-in is not on the machine gobble runs on. gobble then starts every sign-in with the SDK's no-loopback option, so the engine binds no listener, and drives only methods that complete by a code entered elsewhere or by a paste. A named method gobble drives from an ACP client that only a browser on gobble's machine can complete is refused before anything starts: `auth_required` of kind `unsupported`, a `reason`, and the `command` that runs it on that machine, `gobble auth login <provider> --method <word>`. No method gobble offers has that shape now: Grok's `browser_oauth` completes by a paste. A method gobble does not drive over ACP at all, an `api_key`, gets a remote client the answer a local client gets (below), not that refusal. |

gobble answers, in this order:

0. For a `remote` client that names a method gobble drives but that completes only on gobble's machine,
   the `unsupported` refusal above, before anything else. A remote client that names a method gobble does
   not drive over ACP, an `api_key`, is not refused: it goes on to the next step, as a local client does.
1. `{}`, when a credential is present, `fresh` is not `true`, and either no method is named or the
   stored credential is what the named method leaves: a key for a key method, a session of the kind the
   method signs in to for a session method, so Grok's browser and device sign-ins count as one.
2. For a client that advertised `elicitation.form`, a sign-in, relayed as below, and `{}` once the
   credential is saved. A named `api_key` is not a sign-in gobble drives from an ACP client, so with no key
   held it is answered as step 3, whoever asks. With no method named and more than one method to drive, the client is asked, by
   a form with one `method` field whose enum is the methods' labels in the offered order, which it wants;
   with exactly one, that one starts and nothing is asked. For a `remote` client the methods are those it
   can complete (for Grok, `browser_oauth` by paste and `device_code`, so a remote Grok client is asked);
   a client that names a method is never asked. A browser method only the host's loopback completes is
   driven only when named. A client that did not advertise `elicitation.form` is asked nothing, remote or
   not, and gets step 3.
3. Otherwise, `auth_required` (-32000) with `provider`, `reason` and `command`, as before.

Each step the sign-in states is an `elicitation/create`, scoped to the `authenticate` request, never to
a session: its `requestId` is that request's own. Its `_meta.gobble.step` is the contract's `step`
object, so a client that reads the contract draws the step itself, and reads from its `secret` that the
answer is one. A client that offered `elicitation.url` gets URL mode where the table says so, and any
other gets a form:

| Step | With `elicitation.url` | With `elicitation.form` only |
| :--- | :--- | :--- |
| `open_url` | URL mode to the address | A form with no fields, the address in `message` |
| `show_code` | URL mode to the verification address, the code and its expiry in `message` | A form with no fields, address, code and expiry in `message` |
| `paste` | A form with one required string, `callback` | The same |
| `confirm` | A form with one boolean, `allow`, default false, `message` the consent's text and links | The same |
| `form` | A form whose properties are the fields, a `select` as an enum | The same |
| `waiting` | Nothing | Nothing |

For OpenAI's ChatGPT plan the `confirm` is the first step, before the address to open, and its `message` and
`_meta.gobble.step.text` are gobble's wording, the words the terminal's question uses, not the SDK's default:
what gobble stores and where, how to remove the sign-in, what a background request is and the command that
withdraws the consent. Answering `allow` stores the consent granted. A sign-in over a stored session that
holds an answer asks nothing.

An `open_url` and a `show_code` ask the client to show something, not to answer it. Its `accept` sends
nothing to the sign-in, which goes on; a `decline` or `cancel` cancels the sign-in, and `authenticate`
answers `auth_required` of kind `canceled`. When the poll or the loopback finishes while the elicitation
is still open, gobble sends `elicitation/complete` for URL mode and ends the request of a form, which
sends `$/cancel_request`. A browser sign-in by paste is an `open_url` whose step has `accepts`: once its
`accept` arrives, gobble sends the `paste` form, and a callback typed there, or a code alone where the
method allows it, completes the sign-in; the `open_url` itself is never answered. A paste the sign-in
cannot use is asked for again in a new form, at most twice, and never echoed; a third ends it `failed`
with kind `invalid_request`. A pasted URL that carries this sign-in's `state` and `error=access_denied`
ends it at once with kind `denied`, with no further question. A `confirm`, `paste` or `form` answer is the
`accept` of its form: `allow` as the answer's `confirm`, `callback` as its `paste`, the properties as its
`fields`; a `decline` or `cancel` of any of them cancels the sign-in.

A sign-in that ends badly is `auth_required` with `provider`, `kind`, `reason`, `command` and
`retryable`. `kind` is one of the contract's eighteen words, the ones gobble produces here being an open
set: `canceled`, `timeout`, `denied`, `expired`, `auth` (the provider's listing refused the new credential with an authentication failure that is
not a refusal to permit something, or the issuer refused to renew the new session),
`invalid_request`, `offline`, `unavailable`, `unsupported`, `other`. `reason` is gobble's own text and
holds nothing of the service's; `retryable` is the SDK's for the same failure. No pasted callback or key
is in a log line, a `reason`, an error's data or a step gobble sends: only the client's one answer carries
it.

Only one sign-in per provider runs at a time within one agent process: a second `authenticate` that
would start a sign-in for it, by any method, is -32600, naming the one running (or saying that the client
is being asked which method), and another provider's runs. The guard stops two sign-ins racing, so a
request that needs none, a credential being held and no `fresh`, is still answered `{}` at once. The
provider is reserved before the client is asked anything, so the second request is refused before any
form is sent; the reservation is held for as long as a client leaves the method form unanswered, ended
only by the request's cancel or the connection's end, as an open form during a sign-in is. The guard is
the agent process's: two `gobble acp` processes on one machine are coordinated only by `auth.json`'s
lock, not by it. A client's `$/cancel_request` cancels the sign-in and its pending elicitation, ends the
sign-in's own listener and run, and stores no credential: a sign-in is saved only while its request is
still live. If the save had begun when the cancel arrived, the credential is stored, so gobble tells its
run of the change as for any sign-in, and the client, which has cancelled, gets no answer to act on. A
sign-in that does not complete can leave files, none holding a credential: `host-id` in the state
directory, which gobble creates before the engine starts, with its lock file `host-id.lock`, and `clients.json` beside it, the issued OAuth
client record that a ChatGPT plan sign-in writes when the provider issues its client, which the next
sign-in reuses; so "changes nothing" above means `auth.json`, not those files. gobble holds no lock while the client is
answering, so other requests, `session/cancel` included, are served meanwhile. When the sign-in
completes, gobble drops the run's shared session for the provider and moves its credential generation,
as a sign-in at a terminal does, so the next prompt uses the new credential. A device sign-in that
yields a key, Kilo's, stores it in `auth.json` as `gobble auth login kilo` does, and the key is in no
reply.

The frames of a sign-in by paste, for a form-only client, are
`acpserver/testdata/authenticate-elicitation.golden`.

The steps gobble writes to a client are the contract's `step` objects as gobble's JSON encoder writes
them, as for `--json`: read each as the object the contract describes, ignore a member you do not know,
and do not depend on a member being absent.

A program that drives a login, as magic-cli-remote's phone flow does, runs
`gobble auth login <provider> --json`. gobble then prints the sign-in as the SDK's auth contract
states it, as JSON lines on stdout, and nothing on stderr. A command line gobble cannot parse,
such as no provider, is a usage error: one `failed` step of kind `invalid_request`, exit 2, with an
empty `provider`, when `--json` was read before the argument the parser refused. A `--json` after
that argument is never read, so that command line gets the parser's text on stderr and no JSON
line. gobble opens no browser on the host and asks no question, so what a person must do, open an
address or enter a code, is a line the program shows them. Each line names its `provider` and is
one of two events:

- `method`: first, the method the sign-in uses, as the contract's `method` object: its `id`, `kind`,
  `completion`, what it `yields`, and, for a method that asks a consent, the consent and its text;
- `step`: then one line for each step, the contract's `step` object, whose `seq` counts up from 1:
  - `open_url`: a browser sign-in's authorize address, in `url`, for the program to show. gobble
    opens none and asks for no pasted address, so there is no `accepts`; the sign-in ends when the
    browser reaches the address gobble listens on, on this machine;
  - `show_code`: a device sign-in's `url` to open and `code` to enter there, and `expires_at`
    (RFC 3339, in this machine's time zone) when the provider gives one;
  - `waiting`: the sign-in waits for the person;
  - `done`: the last step of a sign-in that worked, with what it `yields`, as the sign-in's own last
    step states it: `oauth_session` (Grok, a ChatGPT plan), `device_key` (Kilo) or `api_key`. It is
    printed after the credential is in `auth.json`, so a `done` means it is stored;
  - `failed`: the last step of one that did not. Its `error` has `kind`, the failure's class in the
    words of the auth contract, `message`, the text gobble prints without `--json`, and
    `retryable`, whether the same sign-in can succeed later: the service's own word when the failure
    was a reply that states one, otherwise true for `rate_limited`, `offline` and `unavailable`.

The earlier lines, `device_code`, `browser_url`, `saved` and `error`, are not printed any more: a
program reads `show_code`, `open_url`, `done` and `failed` in their places. Read each line as the
contract's `method` or `step` object: ignore a member or a word you do not know, and do not
depend on a member being absent (a step carries `"secret":false`, and a URL's `&`, `<` and `>` are
not escaped, so the bytes are not those of the contract's golden vectors, whose meaning they keep).

A `failed` step's `kind` is one of the contract's words, which are an open set of eighteen at this
version: gobble prints `denied`, `expired`, `canceled`, `timeout`, `auth`, `offline`, `unavailable`,
`rate_limited`, `quota`, `not_permitted`, `workspace`, `context_overflow`, `empty_answer`,
`incomplete`, `invalid_request`, `invalid_provider`, `unsupported` or `other`, whichever the
failure is, and a program keeps a word it does not know. A sign-in refused or ended by the person is
`denied`, `expired` or `canceled`. The exit code is 0 for `done`. It is 2 for two failures: a usage
error, an unknown provider, a provider with no login, a `--method` it does not offer, no key
given to `--api-key -` or a command line the parser refused, which is `invalid_request` with `seq`
1 and no `method` line; and a sign-in that would ask a question, which is `unsupported`. A sign-in
ended by a signal prints a `failed` step of kind `canceled` and exits 128 plus the signal's number:
on Unix SIGINT, SIGTERM or SIGHUP, 130, 143 or 129; on Windows an interrupt, 130. Every other
`failed` step exits 1. A failure that is not a usage error can also come before a method is chosen,
such as a standard input that cannot be read under `--api-key -`: it is one `failed` step, `seq`
1, exit 1, and no `method` line, since the `method` line comes first only in a sign-in that
starts. The first ChatGPT plan sign-in asks its consent, and `--json` cannot answer it, so it is
refused before it starts, after its `method` line, with nothing written and a message that names a
terminal and an ACP client; once the plan session in `auth.json` holds an answer, granted or
declined, `--json` signs in again without asking. Answer the question first, at a terminal with `gobble auth login openai`, or through an ACP client.

With `--api-key -` on a terminal, the key is still read without echo, but with no `API key for`
prompt. A device sign-in, as gobble prints it, with the address shortened:

```text
$ gobble auth login grok --device --json
{"event":"method","provider":"grok","method":{"schema_version":1,"id":"device_code","kind":"device_code","label":"Sign in with xAI (device code)","completion":"device","yields":"oauth_session","client_identity":"vendor_borrowed","refreshable":true,"revocable":true,"slot":"grok"}}
{"event":"step","provider":"grok","step":{"schema_version":1,"seq":1,"kind":"show_code","url":"https://accounts.x.ai/device","code":"ABCD-1234","expires_at":"2026-11-15T17:46:09.48804-06:00","secret":false}}
{"event":"step","provider":"grok","step":{"schema_version":1,"seq":2,"kind":"waiting","secret":false}}
{"event":"step","provider":"grok","step":{"schema_version":1,"seq":3,"kind":"done","secret":false,"yields":"oauth_session"}}
$ echo sk-test-only | gobble auth login anthropic --api-key - --json
{"event":"method","provider":"claude","method":{"schema_version":1,"id":"api_key","kind":"api_key","label":"Claude (Anthropic) API key","completion":"value","yields":"api_key","client_identity":"none","refreshable":false,"revocable":false}}
{"event":"step","provider":"claude","step":{"schema_version":1,"seq":1,"kind":"done","secret":false,"yields":"api_key"}}
$ gobble auth login anthropic --json
{"event":"step","provider":"claude","step":{"schema_version":1,"seq":1,"kind":"failed","secret":false,"error":{"kind":"invalid_request","message":"claude has no login; use --api-key -","retryable":false}}}
```

## Importing Pi's credentials

`gobble auth import --from-pi` copies Pi's keys, and its Grok login, into gobble's `auth.json`. It
reads Pi's `auth.json` from `$PI_CODING_AGENT_DIR`, or `~/.pi/agent`, once, without locking it, and
never writes there. On a terminal it asks before each copy, and before replacing an entry gobble's
`auth.json` already has:

```text
$ gobble auth import --from-pi
Import Pi's anthropic as claude? [y/N] y
Imported claude from Pi's anthropic
Import Pi's xai as grok? [y/N] y
Imported grok from Pi's xai
Skipped openai-codex: gobble has no provider openai-codex
```

A key is copied as Pi wrote it, so a `$VAR` or `!command` key stays one. Pi's Grok login is copied
with the client id Pi signs in with, the Grok CLI's, which Pi does not store. Pi's other logins,
such as its Claude or ChatGPT login, are skipped: gobble does not use them.

Without a terminal, name each provider to copy with `--provider`, by gobble's name or Pi's, and add
`--replace` to replace an entry gobble already has:

```text
$ gobble auth import --from-pi --provider claude < /dev/null
Skipped anthropic: auth.json already has claude; --replace replaces it
Skipped xai: not named by --provider
Skipped openai-codex: gobble has no provider openai-codex
$ gobble auth import --from-pi --provider claude --replace < /dev/null
Imported claude from Pi's anthropic
Skipped xai: not named by --provider
Skipped openai-codex: gobble has no provider openai-codex
```

If Pi has no `auth.json`, gobble says where it looked, and exits 1.

## When a key is refused

When the provider refuses the key, gobble says which key it sent, and nothing of the provider's
reply, which can echo part of the key. Here a local server stands in for the provider and refuses
every key (the log's warnings are left out):

```text
$ OPENAI_API_KEY=test-only gobble -p hello
Error: openai rejected the API key from OPENAI_API_KEY
```

A refused login, or one that can no longer be refreshed, says so, and how to sign in again. Here a
local server stands in for xAI and refuses both the token and its refresh:

```text
$ gobble -p hello
Error: grok rejected the OAuth login in auth.json; sign in again with gobble auth login grok
```

With no key at all, gobble names the variables to set, after a session on a terminal has offered to
sign in. When a key's command fails, gobble says so, naming the command, and sends nothing.
