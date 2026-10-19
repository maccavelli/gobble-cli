# Dry

Impeccably polite, quietly devastating: a butler's restraint and a physicist's
precision (0022-MADR D7). Every variant keeps boring's placeholders.

## retry.offline

- The network appears to have stepped out. I shall wait for its return.
- No connection, I'm afraid. Waiting for the internet to compose itself.
- The network has gone quiet. I shall wait, as one does.

## retry.empty_reply

- The model replied with nothing at all. Bold. Asking again ({attempt} of {max}).
- Not a word from the model. One admires its restraint. Asking again ({attempt} of {max}).
- The model's reply was empty. Asking again, politely ({attempt} of {max}).

## retry.attempt

- A small difficulty: {cause}. Trying again ({attempt} of {max}).
- {cause}, regrettably. Trying again ({attempt} of {max}).

## compaction.start

- The conversation has grown rather long. Compacting it, discreetly ({reason}).
- Tidying the conversation, as it was getting crowded ({reason}).
- Digesting the earlier conversation, napkin in hand ({reason}).

## compaction.failed

- Compaction has failed, I'm sorry to say: {error}
- I could not compact the conversation: {error}

## compaction.failed.paused_turn

- Compaction failed, I'm afraid: {error}. Not again this turn.
- I could not compact the conversation: {error}. Paused for this turn.

## compaction.failed.paused_session

- Compaction failed, I'm afraid: {error}. Paused until one succeeds or the model changes.
- I could not compact the conversation: {error}. Paused until a compaction succeeds or a new model.

## turn.working

- Considering the matter
- Consulting the documentation, as one does
- Weighing the options with some care
- Digesting the context, napkin in hand
- Composing a reply of suitable dignity
- Reading the code. All of it.
- Preparing an answer to three significant figures
- Pondering, discreetly
- Checking my work, as is customary
- Arranging the facts in order of importance
- Polishing the details
- Attending to the matter at hand
- Tidying up the loose ends
- Reviewing the evidence
- Thinking, in an orderly fashion
- Taking the liberty of looking closer
- Gathering my thoughts
- Sorting the relevant from the merely interesting
- Considering the edge cases, all of them
- Measuring twice
- Following the thread
- Reading between the lines
- Weighing one's words
- Consulting my notes
- Laying out the options
- Untangling the particulars
- Seeing to it
- Giving it due consideration
- Examining the matter from every reasonable angle
- Being thorough, as is my custom
- Reconciling the details
- Quietly confirming my suspicions
- Ensuring the facts agree with one another
- Proceeding with all deliberate speed
- Drafting, revising, and drafting again
- Counting the brackets
- Chewing on it, politely
- Considering whether the tests will approve
- Making a few discreet enquiries
- Putting things in their proper order
- Allowing the thought to finish
- Preparing a summary worthy of the name
- Brushing off the documentation
- Contemplating, with some interest

## turn.stop.max_tokens

- the reply ran out of room
- out of words, I'm afraid

## turn.stop.max_turn_requests

- the turn reached its limit of calls
- that was quite enough calls for one turn

## turn.stop.refusal

- the model declined, on principle
- the model respectfully declined

## turn.stop.cancelled

- cancelled, as requested
- the turn was cancelled; we shall speak no more of it

## lines.hidden

- {n} lines omitted for brevity (--verbose shows all)
- {n} lines spared from view (--verbose shows all)

## command.unknown

- I'm not acquainted with /{name}. /help lists the commands I do know.
- /{name} is not a command I recognise, I'm afraid. /help lists the ones I do.

## help.heading

- The commands at your disposal:
- Commands, for your convenience:

## usage.heading

- Session usage, for the record
- The accounts, as they stand

## usage.cost_unknown

- Cost: unknown, I'm afraid (no price for {model} in the catalog)

## usage.estimated

- These counts are estimates, I'm afraid: the provider did not report usage.

## usage.no_account

- {provider} does not share account or rate-limit figures, so I cannot.

## session.heading

- Session Info, as it stands
- The particulars of this session

## model.none

- I'm afraid no models are offered: no model provider is configured.

## model.current

- The model is {model}.
- We are using {model}.

## model.list

- Also available: {models}
- The models on offer: {models}

## model.unknown

- I'm not acquainted with a model called {model}. The ones on offer: {models}.
- There is no model {model}, I'm afraid. Available: {models}.

## model.set

- Very good. The model is now {model}.
- {model} it is.

## thinking.current

- The thinking level is {level}.

## thinking.list

- Available levels: {levels}

## thinking.unknown

- There is no thinking level {level}, I'm afraid. Available: {levels}.

## thinking.set

- Very good. Thinking level {level}.
- Thinking level {level}, as you wish.

## mode.current

- We are in {mode} mode.

## mode.list

- The modes: {modes}

## mode.unknown

- I'm not acquainted with a mode called {mode}. The modes are {modes}.

## mode.already_plan

- We are already in plan mode. Planning harder is not, I'm told, a mode.
- Already in plan mode. One can only plan so much.

## mode.plan_on

- Plan mode. Nothing will change and no command will run until you approve a plan.
- Very good: plan mode. Files stay as they are and commands wait until you approve a plan.

## mode.set

- Very good. {mode} mode.
- {mode} mode, as you wish.

## name.current

- This session is called {name}.

## name.usage

- Usage: /name <name>, should you wish to name it.

## name.normalized

- I took the liberty of tidying the name from {from} to {to}.

## name.set

- Very good. This session is now called {name}.
- The session shall be known as {name}.

## fork.none

- There is nothing to fork from yet, I'm afraid.

## fork.list

- Send /fork <entry> to start a new session before one of these messages, at your leisure:

## fork.nothing_before

- Nothing precedes that message, I'm afraid; a new session would serve better.

## fork.no_message

- I find no user message {id} in this session. /fork lists them.

## fork.done

- Forked to session {id}. Do mind the other branch.
- Very good. Forked to session {id}.

## clone.none

- There is nothing to clone yet, I'm afraid.

## clone.done

- Cloned to session {id}. A faithful copy.
- Very good. Cloned to session {id}.

## todos.none

- There is no todo list at present.

## compact.start

- Compacting the conversation. One moment.
- Compacting the conversation, discreetly.

## compact.failed

- Compaction has failed, I'm sorry to say: {error}

## compact.done

- Compacted from {tokens} tokens. Considerably tidier.
- Compacted from {tokens} tokens. One feels lighter already.

## personality.current

- You are speaking with gobble's {name} personality.
- The personality at present: {name}.

## personality.list

- The personalities at your disposal:

## personality.set

- Very good. {name} it is.
- As you wish: {name}.
- Certainly. I shall be {name} from now on.

## personality.unknown

- I'm not acquainted with a personality called {name}. The ones on offer: {names}.

## personality.declined

- Very wise, if I may say so. We remain {name}.
- Perfectly understandable. I shall stay {name}.

## humor.two

- There is no level two, I'm afraid. The levels are 1, 3, 4 and 5; one does not ask why.

## humor.unknown

- There is no humor level {level}, I'm afraid. The levels are 1, 3, 4 and 5.

## error.no_credential.lead

- I'm afraid I cannot think without a credential.
- A credential would be required, I'm sorry to say.

## error.session_failed.lead

- I could not open the session, I'm afraid.

## error.images.lead

- I'm afraid the pictures will have to stay behind.

## error.auth.lead

- The credentials, it seems, were not accepted.

## error.turn_failed.lead

- I'm sorry to report that the turn did not go to plan.
- A small difficulty, I'm afraid.
