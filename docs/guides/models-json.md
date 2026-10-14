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

`//` comments and trailing commas are accepted, as in Pi. Run with no credential set, gobble
reports the two entries it cannot use yet, then stops at the missing key (each line's timestamp is
left out here, and `<sdk-version>` is the go-llmprovider-sdk version gobble was built with, which
the warning names):

```text
$ gobble -p hello
level=WARN msg=models.json message="provider \"openai\": apiKey takes effect with gobble's credentials (0005-PLAN F4-3); until then the provider's variable is used" file=<config>/models.json
level=WARN msg=models.json message="provider \"openrouter\": not a provider go-llmprovider-sdk <sdk-version> builds (one of claude, gemini, grok, huggingface, kilo, ollama, openai, opencode-go, opencode-zen, together); skipped" file=<config>/models.json
Error: no model credential: set one of GEMINI_API_KEY, OPENAI_API_KEY, ANTHROPIC_API_KEY, XAI_API_KEY, OPENCODE_API_KEY, HF_TOKEN, KILO_API_KEY, TOGETHER_API_KEY
```

Each warning names the provider and why. A file gobble cannot parse is ignored whole, with one
warning naming the error.

## Providers

A key under `providers` is one of gobble's provider ids: `openai`, `claude`, `gemini`, `grok`,
`opencode-zen`, `opencode-go`, `huggingface`, `kilo`, `together` and `ollama`. Pi's `anthropic`,
`google` and `xai` mean `claude`, `gemini` and `grok`; if both spellings are given, gobble's is used.
Any other key is skipped with a warning, since gobble has no wire for it.

`api` may be left out. When given, it must be Pi's name for the provider's own wire:
`openai-responses` for openai and grok, `anthropic-messages` for claude, `google-generative-ai` for
gemini, `openai-completions` for huggingface, kilo, together and ollama, and any of those four for
opencode-zen and opencode-go. Anything else skips the provider.

## What each field does

- **`baseUrl`** replaces the provider's base URL, for its requests and its model listing. It is the
  base the request path is added to, so it carries the API version. **Claude's includes `/v1`**:
  `https://llm-proxy.example/anthropic/v1`, where Pi's file would say `https://api.anthropic.com`.
- **`headers`** are added to each request: the provider's, then a model's override's, then its
  definition's, a later one winning.
- **`models`** adds a model to the provider's model choice, after the listed ones. A field it
  leaves out takes Pi's default: name the id, no reasoning, text input, no cost, a 128000-token
  window and a 16384-token output cap. An entry with an id gobble already knows replaces it.
- **`modelOverrides`** changes only the fields it gives, over what gobble knows of the model:
  `name`, `reasoning`, `thinkingLevelMap`, `input`, `cost`, `contextWindow`, `maxTokens` and
  `headers`. A cost's rates are each optional, and `tiers`, when given, replace the model's.
- **`thinkingLevelMap`** turns thinking levels on or off: a string turns a level on, and `null`
  turns it off. gobble uses the level's own name with the provider; the string is not used.

## Values

A header value can name a variable or run a command, as in Pi:

- `$NAME` and `${NAME}` are environment variables, and `$$` and `$!` are a literal `$` and `!`.
- A value starting with `!` runs the rest as a command, through the same bash the bash tool uses,
  for at most 10 seconds, and uses what it prints.

Values are resolved for each request and not cached. A variable that is not set, or a command that
fails or prints nothing, stops that request with an error naming the header and the variable or
command, never the value.

## What is not used

- **`apiKey`** is read and checked, and takes effect with gobble's credentials (0005-PLAN F4-3).
  Until then the provider's standard variable is the key, and gobble warns once.
- **`compat`, `oauth`, `authHeader`, `name`, `inputLimits`, `promptCache`, `samplingParams`** and a
  model's own **`baseUrl`** are reported once per provider, and ignored.
- Providers other than gobble's ten, Pi's `opencode` among them (gobble's is `opencode-zen`), are
  skipped.
