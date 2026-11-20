# Configuring models with models.json

`models.json` points a provider at a proxy or gateway, adds headers to its requests, adds models
gobble does not list, and corrects what gobble knows of a model: its context window, output cap,
price and thinking levels. It is Pi's file, in Pi's schema, so a Pi `models.json` mostly works
unchanged. What gobble does differently is listed under [What is not used](#what-is-not-used).

## Where it lives

It is `models.json` in gobble's config directory. `gobble config path` prints the directory on its
`config:` line. With no file, gobble uses what it ships with. gobble reads the file once, when it
starts, so restart gobble after you edit it.

## An example

```jsonc
{
  // gobble reads this file once, when it starts.
  "providers": {
    "anthropic": {
      "baseUrl": "https://llm-proxy.example/anthropic/v1",
      "headers": {"X-Team": "$TEAM_ID"},
      "modelOverrides": {"claude-haiku-5-5": {"contextWindow": 200000}},
    },
    "openrouter": {"baseUrl": "https://openrouter.ai/api/v1"},
    "openai": {"apiKey": "!pass show openai", "models": [{"id": "my-finetune", "contextWindow": 32000}]},
  },
}
```

`//` comments and trailing commas are accepted, as in Pi. `openrouter` has a base URL and no `api`,
so it is a Chat Completions server. This run does not name it, so gobble does not choose it. With no
credential set and no `pass` installed, gobble tries openai's `apiKey` command, and stops when it
fails, sending nothing (each line's timestamp is left out here):

```text
$ gobble -p hello
level=WARN msg=credentials message="models.json: provider openai: apiKey: the command \"pass show openai\" failed: exit status 127"
Error: openai's API key from models.json could not be resolved: models.json: provider openai: apiKey: the command "pass show openai" failed: exit status 127
```

Each warning names the provider and why. A file gobble cannot parse is ignored whole, with one
warning naming the error.

## Providers

A key under `providers` is one of gobble's provider ids: `openai`, `claude`, `gemini`, `grok`,
`opencode-zen`, `opencode-go`, `huggingface`, `kilo`, `together` and `ollama`. Pi's `anthropic`,
`google` and `xai` mean `claude`, `gemini` and `grok`; if both spellings are given, gobble's is used.
Any other key with a `baseUrl`, and an `api` of `openai-completions` or none, is a Chat Completions
server. Its `name`, when set, is the label; otherwise the key is. gobble chooses it only when
something names it: `--model`, `defaultProvider`, or the picker. A key with no `baseUrl` is skipped,
with a warning that it is not a provider go-llmprovider-sdk builds. A key whose `api` is anything
else is skipped, with a warning that gobble has no generic wire for that api.

`api` may be left out. When given for one of gobble's own providers, it must be Pi's name for that
provider's wire: `openai-responses` for openai and grok, `anthropic-messages` for claude,
`google-generative-ai` for gemini, `openai-completions` for huggingface, kilo, together and ollama,
and any of those four for opencode-zen and opencode-go. Anything else skips that provider. For a
key that is not one of those providers, `api` may be left out or be `openai-completions`.

## What each field does

- **`baseUrl`** replaces the provider's base URL, for its requests and its model listing. It is the
  base the request path is added to, so it carries the API version. **Claude's includes `/v1`**:
  `https://llm-proxy.example/anthropic/v1`, where Pi's file would say `https://api.anthropic.com`.
- **`headers`** are added to each request: the provider's, then a model's override's, then its
  definition's, a later one winning. A provider with headers keeps the same connection and reply
  time limits as one without.
- **`models`** adds a model to the provider's model choice, after the listed ones. A field it
  leaves out takes Pi's default: name the id, no reasoning, text input, no cost, a 128000-token
  window and a 16384-token output cap. An entry with an id gobble already knows replaces it.
- **`modelOverrides`** changes only the fields it gives, over what gobble knows of the model:
  `name`, `reasoning`, `thinkingLevelMap`, `input`, `cost`, `contextWindow`, `maxTokens` and
  `headers`. A cost's rates are each optional, and `tiers`, when given, replace the model's.
- **`thinkingLevelMap`** turns thinking levels on or off: a string turns a level on, and `null`
  turns it off. gobble uses the level's own name with the provider; the string is not used.
- **`apiKey`** is the provider's key when `auth.json` has none for it, ahead of the provider's
  variable ([credentials](credentials.md)). It takes the values below; a command runs each time the
  provider is built, and a value whose variables are unset counts as no key.

## Values

A header value, or `apiKey`, can name a variable or run a command, as in Pi:

- `$NAME` and `${NAME}` are environment variables, and `$$` and `$!` are a literal `$` and `!`.
- A value starting with `!` runs the rest as a command, through the same bash the bash tool uses,
  for at most 10 seconds, and uses what it prints.

Header values are resolved for each request and not cached. A variable that is not set, or a command that
fails or prints nothing, stops that request with an error naming the header and the variable or
command, never the value.

## What is not used

- **`compat`, `oauth`, `authHeader`, `inputLimits`, `promptCache`, `samplingParams`** and a
  model's own **`baseUrl`** are reported once per provider, and ignored.
- **`name`** is the label of a Chat Completions server. On one of gobble's own providers it is not used.
- A key that is not one of gobble's ten, and not a Pi alias of one, is skipped when it has no
  `baseUrl`, or when its `api` is set to something other than `openai-completions`. Pi's `opencode`
  is not an alias of `opencode-zen`. With a `baseUrl` and no other `api`, it is a Chat Completions
  server, like any other such key.
