# Credentials

gobble needs a key, or a login, for the provider it talks to. This page says where gobble looks for
one, how to store one or sign in, and what gobble says when a credential is missing or refused.
gobble keeps keys the way Pi does, in an `auth.json` of Pi's shape, so a Pi `auth.json` mostly
works unchanged.

## Where a key comes from

For each provider, gobble takes the first of these that has a key:

1. **`--api-key -`**, for one run. gobble reads the key from stdin, or asks for it when stdin is a
   terminal, and never writes it anywhere. It needs `--model`, which names the provider.
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

## Storing a key

`gobble auth login <provider> --api-key -` stores a key in `auth.json`, read from stdin or typed
when asked. The file is created readable by you alone, and every change to it is locked, so two
gobble processes never lose each other's entries:

```text
$ echo sk-test-only | gobble auth login openai --api-key -
Saved API key for openai
```

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

`gobble auth logout <provider>` removes the provider's entry. Variables and `models.json` are left
alone:

```text
$ gobble auth logout openai
Removed stored API key for openai. Environment variables and models.json config are unchanged.
```

Pi's `anthropic`, `google` and `xai` name gobble's `claude`, `gemini` and `grok` here too.

## Signing in

Two providers let you sign in with your account instead of pasting a key:

- **Grok.** `gobble auth login grok` signs in with your SuperGrok or X Premium subscription. gobble
  prints the sign-in URL, then opens it in your browser; on a machine whose browser cannot reach
  gobble, paste the URL the browser ends on when asked. `gobble auth login grok --device` shows a
  code to enter at xAI instead, for a machine with no browser at all. Either way gobble says
  `Logged in to grok`, keeps the login in `auth.json`, and refreshes it as it runs. gobble signs in
  with the Grok CLI's public client id: xAI offers third-party tools no id of their own, so if xAI
  restricts that id, the login stops working, and a key still works.
- **Kilo.** `gobble auth login kilo` shows a code to enter at Kilo. Kilo answers with a key, which
  gobble stores as Kilo's API key and says `Saved API key for kilo`.

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
warns that the login was not revoked, and removes it anyway:

```text
$ gobble auth logout grok
Logged out of grok
```

Signing in to ChatGPT comes later, through OpenAI's sign-in for third-party agents (0005-PLAN
F4-3d); until then, use an OpenAI API key. Any other provider takes a key:

```text
$ gobble auth login openai
Error: openai's login waits for go-llmprovider-sdk's Sign in with ChatGPT for agents (0005-PLAN F4-3d); use --api-key -
$ gobble auth login anthropic
Error: claude has no login; use --api-key -
```

An OAuth entry for any other provider, such as one copied from Pi, is kept, and gobble says it does
not use it.

## Signing in from an editor or magic-cli-remote

An ACP client, such as an editor or magic-cli-remote, runs gobble as `gobble acp`. gobble tells it,
when they connect, which credentials it can use: one method per provider that takes a key, named
after the provider, with the gobble command that stores its credential. Asked to authenticate with
one, gobble answers from the credentials it already has, in the order above; it says which command
to run when it has none. No key or token crosses ACP.

A client that can open a terminal also sees a sign-in for each provider, which runs
`gobble acp --login <provider>` there: a login for Grok and Kilo, a key typed for any other. Run it
yourself the same way:

```text
$ echo sk-test-only | gobble acp --login anthropic
Saved API key for claude
```

gobble does not sign out over ACP; use `gobble auth logout <provider>`.

A program that drives a login, as magic-cli-remote's phone flow does, runs
`gobble auth login <provider> --json`. gobble then prints what the login shows as JSON lines on
stdout, and nothing on stderr; the exit code is the same as without `--json`. Each line is one
event, naming its `provider`:

- `device_code`: the `url` to open, the `code` to enter there, and `expires`, in UTC, when the
  provider gives one;
- `browser_url`: the `url` of a browser login. gobble still opens it, and asks for no pasted URL;
- `saved`: the credential is stored, its `type` `api_key` or `oauth`;
- `error`: the login failed. `kind` is its class: `auth`, `timeout`, `canceled`, `unavailable`,
  `quota`, `rate-limited`, `not-permitted`, or `other`. `message` is what gobble prints without
  `--json`.

With `--api-key -` on a terminal, the key is still read without echo, but with no `API key for`
prompt:

```text
$ echo sk-test-only | gobble auth login anthropic --api-key - --json
{"event":"saved","provider":"claude","type":"api_key"}
$ gobble auth login anthropic --json
{"event":"error","provider":"claude","kind":"other","message":"claude has no login; use --api-key -"}
```

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

With no key at all, gobble names the variables to set. When a key's command fails, gobble says so,
naming the command, and sends nothing.
