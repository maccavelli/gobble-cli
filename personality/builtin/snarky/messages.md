# Snarky

Deadpan, a little exhausted, and painfully aware of the odds (0022-MADR D8).
Mild profanity only, and never at the user. Every variant keeps boring's
placeholders.

## retry.offline

- The internet left. Waiting for it to come back. It usually does.
- Network's gone. Waiting. Patiently. Mostly.
- No connection. Waiting for the internet to remember us.

## retry.empty_reply

- The model said nothing. Relatable. Asking again ({attempt} of {max}).
- Empty reply. Bold strategy. Asking again ({attempt} of {max}).
- Nothing came back. Trying again ({attempt} of {max}).

## retry.attempt

- {cause}. Of course. Trying again ({attempt} of {max}).
- {cause}, again. Retrying ({attempt} of {max}).

## compaction.start

- Context is full. Digesting the old parts to make room ({reason}).
- Too much conversation. Chewing some of it down ({reason}).
- Out of room. Compacting before something falls off ({reason}).

## compaction.failed

- Compaction didn't take: {error}
- Compaction failed. Naturally: {error}

## compaction.failed.paused_turn

- Compaction didn't take: {error}. Not trying again this turn.
- Compaction failed: {error}. Giving it a rest this turn.

## compaction.failed.paused_session

- Compaction didn't take: {error}. Paused until one works or the model changes.
- Compaction failed: {error}. Off until a compaction works or the model changes.

## turn.working

- Chewing on it
- Thinking. Allegedly.
- Reading your code. No comment yet.
- Calculating the odds this works
- Pretending this is easy
- Digesting tokens, slowly
- Waiting for the model to have an idea
- Consulting the void
- Untangling it, probably
- Doing the part nobody sees
- Reading the docs nobody reads
- Counting the ways this could break
- Thinking harder than the tests did
- Gobbling tokens
- Staring into the stack trace
- Considering a career change
- Running the numbers. They're grim.
- Hoping this one compiles
- Looking for the bug. There's always one.
- Reading YAML. Again.
- Doing math. Reluctantly.
- Making it work. Then making it right.
- Untangling someone's regex
- Following the imports down the hole
- Weighing the odds
- Finding where the edge cases hide
- Chewing slowly. It's a big one.
- Waiting on the model's inner monologue
- Building a mental model. Badly.
- Squinting at the diff
- Asking the code nicely
- Checking whether this ever worked
- Reading between the brackets
- Hunting for the off-by-one
- Thinking. Don't rush me.
- Reconsidering everything
- Digesting the context, with indigestion
- Working. Mostly.
- Assuming the worst, as usual
- Bargaining with the type checker
- Tracing the call nobody documented
- Wondering who wrote this. Probably fine.
- Doing the thing

## turn.stop.max_tokens

- ran out of words
- out of tokens, mid-thought

## turn.stop.max_turn_requests

- hit the call limit. Enough.
- too many calls for one turn

## turn.stop.refusal

- the model said no. It has boundaries.
- refused. Fair enough.

## turn.stop.cancelled

- cancelled. Probably for the best.
- cancelled

## lines.hidden

- {n} lines hidden. You're welcome. (--verbose shows all)
- {n} more lines nobody needed (--verbose shows all)

## command.unknown

- /{name} isn't a thing. /help lists the things that are.
- No /{name}. /help has the real ones.

## help.heading

- Commands. The ones that exist:
- What you can actually type:

## usage.heading

- Session usage. Brace yourself.
- The damage so far

## usage.cost_unknown

- Cost: unknown (no price for {model} in the catalog). Ignorance is cheap.

## usage.estimated

- These counts are estimates: the provider did not report usage. Educated guesses.

## usage.no_account

- {provider} doesn't share account or rate-limit figures. Mysterious.

## session.heading

- Session Info. The paperwork.
- Session Info

## model.none

- No models on offer: no model provider is configured. That's a problem.

## model.current

- Model: {model}. For now.
- Using {model}.

## model.list

- Also on the menu: {models}
- Models: {models}

## model.unknown

- No model called {model}. The real ones: {models}.

## model.set

- Model: {model}. Let's see how it does.
- Switched to {model}.

## thinking.current

- Thinking level: {level}. Adjust expectations.

## thinking.list

- Levels: {levels}

## thinking.unknown

- No thinking level {level}. Pick one of: {levels}.

## thinking.set

- Thinking level: {level}. We'll see.
- Thinking level {level}. Noted.

## mode.current

- Mode: {mode}.

## mode.list

- Modes: {modes}

## mode.unknown

- No mode called {mode}. The modes are {modes}.

## mode.already_plan

- Already in plan mode. Extra planning doesn't stack.
- Still in plan mode. Planning harder won't help.

## mode.plan_on

- Plan mode. Nothing changes and nothing runs until you approve a plan.
- Plan mode. Look, don't touch, until a plan is approved.

## mode.set

- Mode: {mode}.
- {mode} mode. Fine.

## name.current

- Session name: {name}.

## name.usage

- Usage: /name <name>. It's free.

## name.normalized

- Cleaned the name up from {from} to {to}. You're welcome.

## name.set

- Session name: {name}. Very official.
- Named it {name}.

## fork.none

- Nothing to fork from yet. Say something first.

## fork.list

- Send /fork <entry> to start a new session before one of these messages:

## fork.nothing_before

- Nothing comes before that message. Start a new session instead.

## fork.no_message

- No user message {id} here. /fork lists the ones that exist.

## fork.done

- Forked to session {id}. Alternate timeline open.
- Forked to session {id}.

## clone.none

- Nothing to clone yet.

## clone.done

- Cloned to session {id}. Twins.
- Cloned to session {id}.

## todos.none

- No todo list. Living dangerously.

## compact.start

- Compacting. Chewing slowly.
- Compacting the conversation.

## compact.failed

- Compaction failed: {error}

## compact.done

- Compacted from {tokens} tokens. Down to the parts that matter, probably.
- Compacted from {tokens} tokens. Lighter now.

## personality.current

- Personality: {name}. As if you had to ask.
- Currently {name}.

## personality.list

- Personalities. Pick your poison:

## personality.set

- {name} now. This should be interesting.
- Switched to {name}.
- Fine. {name}.

## personality.unknown

- No personality called {name}. The real ones: {names}.

## personality.declined

- Smart. Staying {name}.
- No bombast, then. Still {name}.

## humor.two

- There is no level two. Level two is a myth. The levels are 1, 3, 4 and 5.

## humor.unknown

- Humor level {level} doesn't exist. The levels are 1, 3, 4 and 5.

## error.no_credential.lead

- No credential, no thoughts. It's in the contract.

## error.session_failed.lead

- The session didn't open. Of course it didn't.

## error.images.lead

- The pictures aren't coming along.

## error.auth.lead

- The credential got rejected. Damn.

## error.turn_failed.lead

- Well, that didn't work.
- That turn went sideways.
