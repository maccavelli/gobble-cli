# Choosing a model

gobble talks to one provider for a run, and each session has a model and a thinking level. This
page says how they are chosen, how `--model` finds the model you mean, and how to see the models
gobble knows. `--model` follows Pi's rules and Pi's messages, so a Pi `--model` argument works
unchanged.

## How a run's model is chosen

From the first that applies:

1. **`--model`**, with `--provider` when you give one. It names the provider for the whole run, and
   every session the run opens uses the model, a resumed one included.
2. **The `defaultProvider` and `defaultModel` settings.** `defaultModel` is a new session's model,
   its id exactly; it is used only with `defaultProvider`, and gobble warns once when it cannot
   use it.
3. **The first provider, in the SDK's registry order (gemini, openai, claude, grok, opencode-zen,
   opencode-go, huggingface, kilo, together), with a credential** from any of its sources
   ([credentials](credentials.md)), with the first model its listing offers.

A second model, `smallModel` (`provider/id`, set only in your own settings file), names each new session
from its first prompt; it is sent that prompt. Its provider needs a credential too, and gobble warns once when
it has none. Empty, the default, names no session.

`--thinking` sets the thinking level: `off`, `minimal`, `low`, `medium`, `high`, `xhigh` or `max`.
Without it, a new session starts at `defaultThinkingLevel`, and a resumed one keeps its own. A level
the model does not offer is clamped to the nearest one it does, higher first. gobble's providers
take no level above `xhigh` yet, so `max` is sent as `xhigh`.

## What `--model` accepts

- **`provider/id`**, such as `openai/gpt-6-luna`. Pi's provider names work too: `anthropic`,
  `google` and `xai` mean `claude`, `gemini` and `grok`.
- **An id alone**, such as `gpt-6-astra`, when one provider has it. When several do, gobble picks
  the one whose credential is set, and otherwise asks you to say which.
- **Part of an id or a name**, such as `sonnet`. An undated id is preferred over a dated one, such
  as `-20261001`, and then the highest id.
- **A thinking level after a colon**, such as `sonnet:high`. `--thinking`, when given, wins over it.
- **`--provider` with any of these**, to look in one provider only. `--provider` on its own is
  refused, since it needs a model.

gobble looks among the models it knows without asking any provider: each provider's curated
models, the models in its catalog that can call tools, and the ones `models.json` adds
([guide](models-json.md)). It resolves `anthropic/sonnet` like this:

```text
$ gobble models info anthropic/sonnet
provider  claude
model     claude-sonnet-5-5
name      Claude Sonnet 5.5
context   1000000
max-out   128000
input     text, image, pdf
thinking  low, medium, high, xhigh, max
cost      input $2, output $10, cache read $0.1, cache write $2.5, per million tokens
```

An id three providers have, with no credential set, needs its provider:

```text
$ gobble models info gpt-6-luna
Error: Model "gpt-6-luna" is ambiguous across providers: openai/gpt-6-luna, opencode-go/gpt-6-luna, opencode-zen/gpt-6-luna. No matching provider is authenticated. Use --provider or provider/model.
```

A model gobble does not know is an error, exit status 2:

```text
$ gobble --model nosuch-zz -p hello
Error: Model "nosuch-zz" not found. Use --list-models to see available models.
```

Under a provider gobble knows, an unknown id is used as given, with a warning, so a model newer
than gobble's catalog still works. gobble then knows nothing of its window, price or levels, unless
`models.json` describes it. Here no credential is set, so the run stops before any request:

```text
$ gobble --model openai/gpt-7:high -p hello
Warning: Model "gpt-7" not found for provider "openai". Using custom model id.
Error: no model credential: --model is openai/gpt-7; set OPENAI_API_KEY
```

## Resuming with another model

`--model` and `--thinking` win over a session's saved choice under `-c`, `--session` and `--fork`.
The change is written to the session, as `/model` and `/thinking` write theirs, so the next resume
keeps it. In a session, `/model` and `/thinking` change the model and level for that session only.

## Listing the models

`gobble models list` lists the models of the providers whose credential is set: the models gobble
knows, and those the provider's own listing adds. Words after it filter the list, each matching
the provider and id in order, as Pi's search does. `--list-models` does the same:

```text
$ gobble models list luna
provider  model         context  max-out  thinking  images
openai    gpt-5.6-luna  1.1M     128K     yes       yes
openai    gpt-6-luna    1.1M     128K     yes       yes
```

A fact gobble does not know prints as `-`. With no credential set, the list says which variables to
set:

```text
$ gobble models list
No models available. Set one of GEMINI_API_KEY, OPENAI_API_KEY, ANTHROPIC_API_KEY, XAI_API_KEY, OPENCODE_API_KEY, HF_TOKEN, KILO_API_KEY, TOGETHER_API_KEY.
```

`gobble models info <model>` takes what `--model` takes, and prints what gobble knows of that one
model, as above.

## Completion

With shell completion installed (`gobble completion --help` says how), `--provider` completes the provider ids, and
`--model` completes `provider/id` from the models gobble knows, then the thinking levels after a
colon. Completion asks no provider, so a model only the live listing has is not offered there.
