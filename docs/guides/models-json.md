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
- **`maxTokens`**, in a `models` entry or an override, is the output limit gobble asks for on every
  turn and every compaction summary for that model, reasoning included. See
  [The output limit](#the-output-limit).
- **`thinkingLevelMap`** turns thinking levels on or off: a string turns a level on, and `null`
  turns it off. gobble uses the level's own name with the provider; the string is not used.
- **`apiKey`** is the provider's key when `auth.json` has none for it, ahead of the provider's
  variable ([credentials](credentials.md)). It takes the values below; a command runs each time the
  provider is built, and a value whose variables are unset counts as no key.

## The output limit

Every turn and every compaction summary names an output limit, reasoning included
([0071-MADR](../decisions/0071-MADR-send-each-turn-the-models-output-cap-and-own-the-limit-policy.md)):

- **The model's cap**, from gobble's catalog or the provider's listing, held to **64,000** tokens. A
  model whose listed cap is larger, such as GLM-5.3's, asks for 64,000, so an answer longer than
  that is cut. To lift the ceiling for one model, set its `maxTokens`: a value you set is sent as
  you set it, with no ceiling.
- **A model no source gives a cap for** asks for 16,384, shown as assumed; a `models` entry without
  `maxTokens` takes the same 16,384.
- **Each call is held to the room the window leaves:** the window, less gobble's estimate of the
  context, less 4,096 tokens. With less room than 1,024, it asks for 1,024, or for the cap when the
  cap is smaller, so a cap below 1,024 is sent as it is; with no room, it sends the cap, so the
  service's refusal starts compaction. `compaction.reserveTokens` keeps that room free.
- **A refusal that states the room or the cap**, in the forms gobble reads, is sent again once at
  the value it states, whatever that is, from 1 up, before any compaction; every later call of that
  turn asks for at most that value. A refusal of the limit gobble cannot mend ends the turn with a
  line naming the limit asked and the `maxTokens` to set.
- **A refusal for credit or rate** (a 402, a 429 that outlasts the retries, or Kilo's 503 for an
  upstream's 402) of a call that asked for a limit is not mended or retried: the error text is the
  service's as before, and the terminal client adds a line after it naming the limit asked, that some
  services hold credit or rate against it, and the `maxTokens` to set. An ACP client reads the same
  line from the error data's `outputLimitNote`, and `-p --output-format json` carries it in the
  result's `errorData`; print mode's text output prints no such line.

To ask for less, for example where a gateway holds credit against the limit, set `maxTokens` lower:

```jsonc
{"providers": {"kilo": {"modelOverrides": {"z-ai/glm-5.3": {"maxTokens": 32000}}}}}
```

`/context` and `/session` show the last call's limit and why; `--stats` adds it as `out-limit`.

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
