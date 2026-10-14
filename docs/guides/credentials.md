# Credentials

gobble needs a key for the provider it talks to. This page says where gobble looks for one, how to
store one, and what gobble says when a key is missing or refused. gobble keeps keys the way Pi does,
in an `auth.json` of Pi's shape, so a Pi `auth.json` mostly works unchanged.

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

Pi's `anthropic`, `google` and `xai` name gobble's `claude`, `gemini` and `grok` here too. Signing in
with an account, rather than a key, comes later (0005-PLAN F4-3b); an OAuth entry in `auth.json` is
kept, and gobble says it does not use it yet.

## When a key is refused

When the provider refuses the key, gobble says which key it sent, and nothing of the provider's
reply, which can echo part of the key. Here a local server stands in for the provider and refuses
every key (the log's warnings are left out):

```text
$ OPENAI_API_KEY=test-only gobble -p hello
Error: openai rejected the API key from OPENAI_API_KEY
```

With no key at all, gobble names the variables to set. When a key's command fails, gobble says so,
naming the command, and sends nothing.
