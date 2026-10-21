# Choosing a model

gobble can use the models of every provider you have a credential for, and a session can move
from one to another mid-conversation. Each session has a model and a thinking level. This page says
how they are chosen and switched, how `--model` finds the model you mean, and how to browse the
providers and their models. `--model` follows Pi's rules and Pi's messages, so a Pi `--model`
argument works unchanged.

## How a run's model is chosen

From the first that applies:

1. **`--model`**, with `--provider` when you give one. Every session the run opens starts on the
   model, a resumed one included. A model gobble cannot use is an error; the run does not fall to
   another.
2. **The `defaultProvider` and `defaultModel` settings.** `defaultModel` is a new session's model,
   its id exactly; it is used only with `defaultProvider`. When that provider has no credential, or
   its listing refused the key or found its quota spent, gobble warns and goes on to the next step.
3. **The newest of your recent models** whose provider has a credential and was not refused. gobble
   keeps the last ten models you chose, with `/model`, an editor's model menu or `--model`, in
   `recent-models.json` in its state directory.
4. **The first provider, in the SDK's registry order (gemini, openai, claude, grok, opencode-zen,
   opencode-go, huggingface, kilo, together), with a credential** from any of its sources
   ([credentials](credentials.md)), with the first model its listing offers. When several have one,
   the line session says so before the first prompt:

   ```text
   Starting on Gemini (gemini), the first provider with a key; /model switches, and the defaultProvider setting or --model chooses where a run starts.
   ```

`/model <model> --default` in a session switches to the model and saves it, and its provider, as
`defaultModel` and `defaultProvider` in your own settings file, so new runs start on it;
`/model --default` saves the session's model.

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

## Switching in a session

`/model` with nothing after it lists the models you can switch to, numbered: the session's model,
your recent ones, then a heading for each provider with its state (below) and a table of its
models, each with its window, its input and output price per million tokens, and whether it
thinks. Then:

- **`/model 7`** switches to entry 7 of that list.
- **`/model provider/id`**, such as `/model claude/claude-sonnet-5-5`, or an id only one provider
  offers, switches to it.
- **`/model claude`** switches to the model you last used of that provider, else its first.
- **`/model` with other words**, such as `/model sonnet`, lists the entries that match, keeping
  their numbers, so `/model N` picks one.

The conversation goes on, on the new model, another provider's included. The new model is sent the
conversation's text and tool calls; the earlier model's thinking, and the data a provider keeps
with it, are not, since another model cannot read them. When the new model's window is smaller
than the conversation, the switch says so and suggests `/compact`. A provider whose key was refused
is not offered, and `/model` to it says why.

Editors that speak ACP show the same models in their model menu, grouped by provider, your recent
ones first.

## Resuming with another model

A resumed session goes on with the model it last used, when that provider still has a credential
and was not refused; otherwise with the run's, and a warning. `--model` and `--thinking` win over a
session's saved choice under `-c`, `--session` and `--fork`. The change is written to the session,
as `/model` and `/thinking` write theirs, so the next resume keeps it. In a session, `/model` and
`/thinking` change the session's model and level; only `--default` changes where new runs start.

## Browsing the providers

`gobble providers` lists every provider gobble has, with where its credential comes from and how
you can sign in. It asks no provider anything:

```text
$ gobble providers
provider      name                credential                             sign-in
gemini        Gemini (Google)     none — gobble auth login gemini        API key
openai        OpenAI              none — gobble auth login openai        API key
claude        Claude (Anthropic)  none — gobble auth login claude        API key
grok          Grok (xAI)          none — gobble auth login grok          API key; browser or device sign-in
opencode-zen  OpenCode Zen        none — gobble auth login opencode-zen  API key
opencode-go   OpenCode Go         none — gobble auth login opencode-go   API key
huggingface   Hugging Face        none — gobble auth login huggingface   API key
kilo          Kilo Gateway        none — gobble auth login kilo          API key; device sign-in
together      Together AI         none — gobble auth login together      API key
ollama        Ollama (local)      runs on your machine, no key           —
```

A credential shows as `env: OPENAI_API_KEY`, `auth.json: key` or `auth.json: sign-in`, by where
gobble found it. `--check` lists each provider with a credential, and Ollama, and adds its state;
`--json` prints the same rows as JSON. A listing never spends a generation. The states:

- **live**: the provider listed its models just now;
- **stale**: the models it listed in an earlier run, with their age, while this run's listing
  runs or after it failed for a moment;
- **not checked**: the provider lists its models without a key, so the listing cannot say whether
  the key works;
- **not listed**: the listing failed, offline or unavailable; gobble offers the models it knows;
- **listing**: the listing has not answered yet;
- **key rejected**, **quota exhausted**, **not permitted**: the provider refused the key, with the
  HTTP status, and its models are not offered until a listing succeeds or you sign in again;
- **no credential**: none is set, with the variable to set and the `gobble auth login` to run.

In a session, `/providers` shows the same table, each provider with its state, and a turn the
provider refuses for its key, its quota or its permissions marks it so for the rest of the session.

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

`--provider ID` lists one provider's models, Ollama's included. A provider with no credential
shows the models gobble knows of it, and says so:

```text
$ gobble models list --provider claude sonnet
provider  model                       context  max-out  thinking  images
claude    claude-sonnet-4-5           200K     64K      yes       yes
claude    claude-sonnet-4-5-20250929  200K     64K      yes       yes
claude    claude-sonnet-4-6           1M       128K     yes       yes
claude    claude-sonnet-5             1M       128K     yes       yes
claude    claude-sonnet-5-5           1M       128K     yes       yes

claude has no credential: these are the models gobble knows of it. gobble auth login claude sets one.
```

`--all` lists every provider's models, the known ones of those with no credential, named in a note
after the table. In a session, `/models claude` lists one provider's models and `/models` every
provider's, words after either filtering them, numbered as `/model` numbers them; a provider you
cannot use yet shows its state and the models gobble knows of it.

`gobble models info <model>` takes what `--model` takes, and prints what gobble knows of that one
model, as above.

## Completion

With shell completion installed (`gobble completion --help` says how), `--provider` completes the provider ids, and
`--model` completes `provider/id` from the models gobble knows, then the thinking levels after a
colon. Completion asks no provider, so a model only the live listing has is not offered there.
