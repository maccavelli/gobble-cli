# Boring

The messages gobble has always shown, one per key. Every other personality's
messages fall back to these, and carry exactly their placeholders (0022-MADR
D5). Placeholders are names in braces; a value that was quoted before is
given quoted.

## retry.offline

waiting for the network

## retry.empty_reply

retrying: empty reply (attempt {attempt} of {max})

## retry.attempt

retrying: {cause} (attempt {attempt} of {max})

## compaction.start

compacting the conversation ({reason})

## compaction.failed

compaction failed: {error}

## compaction.failed.paused_turn

compaction failed: {error} (automatic compaction paused for this turn)

## compaction.failed.paused_session

compaction failed: {error} (automatic compaction paused until a compaction succeeds or the model changes)

## turn.working

working

## turn.stop.max_tokens

max_tokens

## turn.stop.max_turn_requests

max_turn_requests

## turn.stop.refusal

refusal

## turn.stop.cancelled

cancelled

## lines.hidden

{n} lines hidden (--verbose shows all)

## command.unknown

Unknown command /{name}. /help lists the commands.

## help.heading

Commands:

## usage.heading

Session usage

## usage.cost_unknown

Cost: unknown (no price for {model} in the catalog)

## usage.estimated

These counts are estimated: the provider did not report usage.

## usage.no_account

Account and rate-limit figures are not reported by {provider}.

## session.heading

Session Info

## model.none

No models are offered: no model provider is configured.

## model.current

Model: {model}

## model.list

Models: {models}

## model.unknown

Unknown model {model}. Available models: {models}.

## model.set

Model: {model}

## thinking.current

Thinking level: {level}

## thinking.list

Levels: {levels}

## thinking.unknown

Unknown thinking level {level}. Available levels: {levels}.

## thinking.set

Thinking level: {level}

## mode.current

Mode: {mode}

## mode.list

Modes: {modes}

## mode.unknown

Unknown mode {mode}. The modes are {modes}.

## mode.already_plan

Already in plan mode.

## mode.plan_on

Mode: plan. Files cannot change and commands cannot run until the user approves a plan.

## mode.set

Mode: {mode}

## name.current

Session name: {name}

## name.usage

Usage: /name <name>

## name.normalized

Session name was normalized from {from} to {to}.

## name.set

Session name set: {name}

## fork.none

No messages to fork from yet.

## fork.list

Send /fork <entry> to start a new session before one of these messages:

## fork.nothing_before

Nothing comes before that message; start a new session instead.

## fork.no_message

No user message {id} in this session. /fork lists them.

## fork.done

Forked to session {id}.

## clone.none

Nothing to clone yet

## clone.done

Cloned to session {id}.

## todos.none

no todo list

## compact.start

Compacting the conversation…

## compact.failed

Compaction failed: {error}

## compact.done

Compacted from {tokens} tokens.
