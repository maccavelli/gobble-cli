---
status: proposed
date: 2026-10-12
decision-makers: repository owner
consulted: none
informed: magic-cli-remote
---
<!-- markdownlint-disable MD013 MD024 MD033 MD036 MD060 -->

# A todo list that keeps the agent working in order, stays live through the session, and works natively with magic-cli-remote

## Context and Problem Statement

gobble has a `todo` tool. [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md) F1d-1 built it on 2026-10-09:

- each call replaces the whole list;
- the server sends the list as an ACP `plan` update, records it as a `gobble.todo` session entry, and replays it on `session/load`;
- `/todos` prints it.

magic-cli-remote ("mcremote") and its Android app show an ACP `plan` as the session's "Todos" panel. The owner wants three things:

- the list correct in real time, as the session runs, through resume, cancel and compaction;
- the agent using the list to stay on task and work in the right order, as a supplement to the MADR and PLAN workflow;
- the list working natively with mcremote, the way grok's does.

This record decides the todo list's model, its rules, its lifecycle in the server, and its wire contract. It also records what mcremote would have to change for a cold-started phone to keep the list. It changes nothing in mcremote, which is the owner's separate repository.

### What was measured, not assumed

All of this was read on 2026-10-12, read-only. The sources were:

- gobble at `2088bc0`;
- magic-cli-remote at `9b75e5f3`;
- grok-build at `2bdd1d6`, opencode at `5683367e27`, kilocode at `05bc9e15fb`, codex at `16845bb8e`, goose at `4617040` and pi at `312184edb`;
- acp-go-sdk2's schema at `v0.15.0`.

Each claim below was read in its file. Four research passes surveyed the harnesses, and each of their load-bearing claims was re-read at the cited line before it went in here.

**gobble today**

- **The tool** (`tool/builtin/todo.go:15-93`):
  - input is `{todos: [{content, status, priority?}]}`;
  - statuses are `pending`, `in_progress` and `completed`; priorities are `high`, `medium` and `low`, defaulting to `medium`;
  - the code refuses a `null` list, empty content, and a status or priority outside those sets;
  - the model gets back counts only: `todo list updated: N completed, N in progress, N pending`.
- **What the tool does not check:** it has no item ids, no length or count limit, and no `cancelled` status. Nothing enforces "one item in progress" or any order. The description asks for one in progress (`tool/builtin/describe/todo.md:16`), but the code never checks it.
- **Live and recorded.** After the call's terminal `tool_call_update`, `planUpdate` sends the list, entries in order and no `_meta` (`acpserver/prompt.go:111-113`, `:206-212`). The writer appends a `gobble.todo` custom entry after the tool result (`acpserver/agent.go:632-634`; `acpserver/history.go:381-401`).
- **Load and resume.**
  - `session/load` replays the path's last list once, after the history, when it is not empty (`acpserver/history.go:167-176`, `:278-284`).
  - `session/resume` sends nothing (`acpserver/agent.go:786-800`).
  - gobble's CLI resumes with `session/resume`, also after `/fork` and `/clone` (`internal/cli/sessions.go:61`; `internal/cli/slash.go:160`). So after any of those, the CLI and any resuming client show no list until the model calls `todo` again.
- **Plan mode.**
  - `exit_plan_mode` sends `planEntries(plan)` as a `plan` update *before* it asks the user (`acpserver/modes.go:146`).
  - `planEntries` (`:179-193`) makes every non-blank line an entry, pending and medium, with list markers stripped. Headings and prose become todo items.
  - Nothing records that list. `/todos`, a reload and the next `todo` call all ignore it.
  - On "Keep planning", the unapproved list stays on the client.
  - The `todo` tool is offered in plan mode, since it is read-only and `planPolicy` gates only calls that need approval (`acpserver/modes.go:90-105`).
- **Compaction.**
  - A `custom` entry never reaches the model. Only `gobble.prompt` patches do (`compaction/estimate.go:214-219`).
  - So after `/compact`, the model's context has neither its todo calls nor its list, though `/todos` and a reload still show the list.
  - A Pi `custom_message` does reach the model, and its `display` decides whether replay shows it (`compaction/estimate.go:200-205`; replay hides one whose `display` is not true, `acpserver/history.go:310-313`).
- **Turn end.** `Prompt` sends a usage update and returns on `agent.End` (`acpserver/agent.go:488-502`). Nothing looks at the list there, so a cancelled turn leaves its item `in_progress`.
- **The CLI** renders content and status only, and drops priority (`internal/cli/view.go:126-133`). `/todos` prints `[x]`, `[>]` and `[ ]` (`acpserver/history.go:405-421`).
- **Tests.** `acpserver/todo_test.go` covers a live update, the recorded entry, `/todos`, a load replay (`TestTodoPlan`, `:74`), and a cleared list (`TestTodoCleared`, `:124`). It does not cover resume, fork, compaction, plan mode, cancel, or more than one item in progress.
- **The system prompt.** Its `tools` section has cross-tool rules (`prompt/sections.go:143-173`), and none of them is about the todo list.
- **Resync.** `_gobble/get_state` returns sixteen fields of session state, and the list is not among them (`acpserver/extensions.go:349-384`). No method returns the list. The extension list advertised in `initialize` is recorded in `acpserver/testdata/session.golden:2`.

**magic-cli-remote and the Android app**

- **Which providers it runs.** mcremote registers `fake`, `grok`, `opencode`, `codex` and `kilo` (`internal/provider/provider.go:64-72`). Its gobble provider, 0179-MADR, is `status: proposed` (0009-REPORT M1 and M2). grok runs on the generic stdio ACP provider, `internal/provider/acpagent`, which is where gobble would run.
- **The plan path is generic.**
  - `case u.Plan != nil` emits `event.TypePlan`, and `plan_removed` emits an empty list (`internal/provider/acpagent/session.go:1740-1756`).
  - `acpcommon.PlanEntries` copies `content`, `status` and `priority`, and turns unknown values into `pending` or `medium`. It drops `_meta` (`internal/provider/acpcommon/plan.go:10-48`).
  - Nothing in mcremote's plan path is grok-specific.
- **Delivery and retention.**
  - `TypePlan` is control traffic, so it is never dropped under back-pressure (`internal/event/event.go:199-204`).
  - An empty list goes on the wire with no `entries` key, which clients read as "clear" (`:624-632`).
  - In retention, a plan is `ClassProgress`, evicted early (`internal/event/retention.go:24-28`, `:82-86`).
- **Replay.**
  - Events from `session/load` go into history but are never broadcast live (`internal/session/manager.go:1479-1484`).
  - The replay duplicate check keys on type, text, tool id and agent session only (`:369-379`, `:388-399`, applied at `:270`). A plan has no text and no tool id, so a replayed plan matches any plan already in the ring and is dropped, whatever its entries.
- **opencode and Kilo get a resync.** Their todos arrive as `todo.updated`, plus a `GET /session/{id}/todo` resync that runs even while a turn is busy (`internal/provider/opencode/todo.go:22-92`; `resync.go:21-37`). The reason is in mcremote's 0020-MADR, line 1027: "SSE can lose mid-turn `todo.updated` frames; the phone should not wait for tree-idle to recover a 10-step list". ACP providers have no such resync.
- **Cancelled items.** mcremote's convention for opencode and Kilo (0020-MADR KD5, line 247) shows a cancelled todo as `pending`, with content prefixed `"(cancelled) "`. Its reason: "not `completed` — false 'done' misleads multi-step UX" (`internal/provider/opencode/todo.go:55-57`).
- **The app** (`apps/mobile`, Flutter):
  - It replaces the list on each event, and treats an identical list as a no-op (`lib/data/chat/transcript_reducer.dart:41-48`, `:958-968`).
  - It batches plan events at 32 ms.
  - Its "Todos" panel (`lib/features/widgets/work_items_panel.dart`) shows a done/total bar and `N done, M remaining · <the first in_progress item>`. In-progress items have a spinner; only `high` priority gets an icon; completed items are struck through.
  - The transcript cache does not keep the plan: `hasControlState` leaves it out (`lib/data/chat/chat_models.dart:577-582`), and the cache writes items and modes only (`lib/data/chat/transcript_cache.dart:320-331`).

**What the other harnesses do**

- **grok-build** works natively with mcremote. It sends a standard ACP `plan` with the whole list straight after each `todo_write` result (`crates/codegen/xai-grok-shell/src/session/acp_session_impl/tool_calls.rs:3049-3051`). Beyond that:
  - **Items** are `{id, content?, status?}` with `merge` (`crates/codegen/xai-grok-tools/src/implementations/grok_build/todo/mod.rs:207-248`).
  - **Cancelled** is sent as `completed` with `_meta.cancelled` (`crates/codegen/xai-grok-shell/src/tools/todo.rs:48-73`). mcremote drops `_meta`, so the phone shows a cancelled grok item as done.
  - **Turn end:** a transient frame shows leftover in-progress items as `completed`. It "must not be persisted or replayed" (`acp_session_impl/turn_end.rs:8-64`).
  - **After compaction,** grok puts the active list in a reminder: "This is your task list from before the conversation was compacted — it is still active. Keep working through the items below…" (`crates/common/xai-grok-compaction/src/reminder.rs:216-222`).
- **opencode** removed its todo tool on 2026-07-09 (`7feefb697f`, "refactor: remove todo tool (#35989)"); the commit gives no reason. Its V1 migration now lists `todowrite` among removed tools (`packages/core/src/database/v1-migration.bun.ts:901`). Before that, its ACP layer built `plan` from the tool's output text, and sent `cancelled` as `completed` (opencode `v1.4.11` `acp/agent.ts:374-397`).
- **kilocode** keeps the tool. Its description adds three sequencing rules, and keeps opencode's rule on verification (`packages/opencode/src/tool/todowrite.txt:24-30`):
  - "After creating a list, call this tool before starting the first item."
  - "After completing each item, call this tool before starting the next item, with the completed item marked `completed` and the next item marked `in_progress`."
  - "Do not complete multiple items or continue through multiple steps without updating the list between them."
  - "Mark `completed` only after the required work is actually done, including any required verification. Never based on intent."
- **codex's** `update_plan` replaces the whole list and returns "Plan updated". Its prompt rules (`codex-rs/protocol/src/prompts/base_instructions/default.md:271-275`) are:
  - "short list of 1-sentence steps (no more than 5-7 words each)";
  - "There should always be exactly one `in_progress` step until everything is done";
  - "If all steps are complete, ensure you call `update_plan` to mark all steps as `completed`."

  The code checks none of these. codex refuses `update_plan` in plan mode (`codex-rs/core/src/tools/handlers/plan.rs:87-111`), and never writes the plan to its rollout (`codex-rs/rollout/src/policy.rs:209`).
- **goose** keeps free-text notes in session metadata, and re-injects them each turn in a `<turn-context>` message, only when they changed (`crates/goose/src/agents/platform_extensions/todo.rs:165-180`; `state_machine/inference_preparation.rs:79-93`). It does not send them as an ACP `plan`.
- **pi** has no todo tool in core. Its `plan-mode` example parses only numbered lines under "Plan:" into steps. Before each run it injects a hidden message: "Remaining steps: … Execute each step in order" (`packages/coding-agent/examples/extensions/plan-mode/index.ts:230-245`). That message is a `customType` message with `display: false`, the same entry kind gobble already writes and replays.
- **Order in code.** None of the six enforces order or a single in-progress item in code. All rely on prompt text.

**The ACP schema** (`schema/schema.json`, 1.25.0):

- `Plan.entries`: "the agent must send a complete list of all entries with their current status. The client replaces the entire plan with each update".
- `PlanEntry` requires `content`, `priority` and `status`, and has `_meta`. Its statuses are `pending`, `in_progress` and `completed`, with no `cancelled`.
- `plan_update` and `plan_removed`, which carry plan ids and Markdown or file plans, are unstable-only (0016-MADR, What was measured).

### Findings

- **F1.** gobble already emits exactly what makes grok's todos work in mcremote: a standard `plan` with the whole list, in order, straight after the call. The live path needs no new transport. What is missing is the list's correctness outside a single call, the agent's discipline in using it, and a way for a client to recover it.
- **F2.** The list goes wrong at four points:
  - **resume** sends none, so the CLI shows none after resume, `/fork` or `/clone`;
  - **cancel** leaves an item spinning `in_progress`;
  - **compaction** removes the list from the model's context;
  - **plan mode** shows an unrecorded, noisy list, and leaves it showing after "Keep planning".
- **F3.** The tool says "one in progress" and checks nothing; it has no order rule, no `cancelled`, and no limits. The other harnesses show that prompt text alone does not keep a model in order: none of the six checks it in code.
- **F4.** The model loses its list at compaction. grok's reminder and Pi's hidden message show the remedy, and gobble already has the entry kind that carries it: a `custom_message` with `display: false`, which reaches the model and is not replayed.
- **F5.** A phone that cold-starts loses the list. The cause is entirely in mcremote: the ring evicts plan events early, the replay duplicate check drops replayed plans, the app cache leaves the plan out, and ACP providers have no resync. gobble can make it less likely but cannot fix it. It can offer what mcremote's opencode path already uses, a resync read, so mcremote has the same remedy for gobble.
- **F6.** ACP has no `cancelled` status. grok sends `completed`, which the phone shows as done. mcremote's own rule for the other providers is `pending` plus a "(cancelled) " prefix. Following mcremote's rule keeps all providers alike on the phone.
- **F7.** The unstable `plan_update` and `plan_removed` would not reach mcremote: it handles only `plan_removed`, and advertises no `plan` capability. 0016-MADR D12 also keeps unstable surfaces out. The stable `plan` is the only shape that works everywhere.
- **F8.** gobble's MADR and PLAN workflow has ordered, named steps: phases `P1…` with numbered steps. A todo list built from those names gives the user and the phone a live view of the plan's execution in the plan's own words. It also gives the order rule something concrete to hold the agent to.

## Decision Drivers

- **Accuracy on every client, at every moment.** The list a client shows is the list the session holds: after a call, a resume, a reload, a cancel, a compaction, and leaving plan mode.
- **Order the agent cannot drift from.** The list's order is the order of work, and the tool refuses a list that contradicts it. Prompt text alone has not held other harnesses to it (F3).
- **Native on mcremote today, with no mcremote change required** for the live path. Changes mcremote could make are recorded for the owner, not assumed.
- **Stable ACP only** (0016-MADR D12; F7).
- **gobble's own entry kinds**, so the session file stays Pi's format: `custom` for the list, and `custom_message` for what the model must see.
- **The wire is the contract** ([0014-MADR-move-to-acp-go-sdk2.md](0014-MADR-move-to-acp-go-sdk2.md) D5). A frame that changes is named here, and nothing else changes.
- **Borrow what has been shown to work:**
  - grok's live frame and its post-compaction reminder;
  - Kilo's sequencing rules;
  - codex's step size, and its refusal in plan mode;
  - Pi's hidden "remaining steps" message;
  - mcremote's own treatment of cancelled items.

## Considered Options

- **A — Keep the whole-list tool and make it exact:**
  - order, one item in progress, `cancelled`, and limits checked in code;
  - a server-side lifecycle covering resume, turn end, cancel, compaction and plan mode;
  - the stable `plan`, with mcremote's treatment of cancelled items;
  - a resync read;
  - written rules that tie the list to MADR and PLAN execution.
- **B — grok's model:** items with ids, `merge` by id, a transient turn-end cleanup that shows leftovers as completed, and `cancelled` as `completed` plus `_meta`.
- **C — goose's model:** free-text notes, re-injected into the model's context every turn when they change, with no ACP `plan`.
- **D — The unstable `plan_update` and `plan_removed`,** with plan ids and Markdown plans.
- **E — Prompt text only,** codex-style: better instructions, no checks in code, and no lifecycle changes.

## Decision Outcome

Chosen: **A**.

- It keeps the one shape mcremote already shows live (F1, F7).
- It fixes every point where the list goes wrong (F2).
- It holds the agent to the list's order in code, where every other harness has relied on prompt text alone (F3).
- It restores the list to the model after compaction (F4).
- It gives mcremote the same resync it already has for opencode (F5), and follows mcremote's convention for cancelled items (F6).

### The decisions

- **D1. The list stays whole: each `todo` call replaces it.** The input stays `{todos: [{content, status, priority?}]}`. Statuses become `pending`, `in_progress`, `completed` and `cancelled`; priorities stay `high`, `medium` and `low`, defaulting to `medium`. Items have no ids: the list's order and its content identify them, as in ACP's `plan`, opencode, Kilo and codex.
- **D2. The list's order is the order of work, and the tool checks it.** A list is valid only when it reads top to bottom as: done (`completed` or `cancelled`), then at most one `in_progress`, then the rest `pending`. The tool refuses anything else, with an error naming the items and the fix:
  - two in progress: `todo: items 2 and 4 are both in_progress; keep one in progress, and mark the other pending or completed`;
  - out of order: `todo: item 3 ("…") is completed, but item 2 ("…") above it is pending; the list is the order of work: move item 3 above item 2, finish item 2 first, or mark it cancelled`.

  To change the order of work, the model reorders the list, and the reordering is visible to the user. A list of only `pending` items is valid: it has just been made.
- **D3. Limits.** At most 50 items. Each item's content is one line of at most 200 runes, after trimming. A longer list or item is refused, with the limit named in the error. The description asks for steps of a few words (codex's "5-7 words").
- **D4. The result keeps the model oriented.** The result text is the counts, then the current item and the next one. For example: `todo list updated: 2 completed, 1 in progress, 3 pending. Now: 3. add the resolver. Next: 4. wire the flag.` When every item is done, it says so: `todo list complete: 5 completed, 1 cancelled`. The model is not asked to repeat the list to the user, since the client shows it (codex's rule).
- **D5. The description and the system prompt carry the discipline.** `tool/builtin/describe/todo.md` is rewritten. Its "Use when" section keeps the three-step threshold and adds "the user gave a list, or a written plan is being carried out". It gains a "Skip when" section: a single step, or a question. Its "Rules" become:
  - make the list before starting the work, and update it before starting each item and as soon as it is done (Kilo);
  - one item in progress, always the first one not done (D2);
  - mark an item `completed` only when the work is done and verified, never on intent (Kilo and opencode);
  - mark an item that no longer applies `cancelled`, and say why in the reply;
  - if blocked, keep the item `in_progress` and add an item for the blocker;
  - when every item is done, send the list with all of them `completed` (codex).

  It also gains a "Plans" section (F8). When carrying out a written plan, whether a PLAN file's phases and steps or a plan the user approved, the list is that plan's steps:
  - in its order, named as the plan names them, for example `P2 step 3 — add the resolver`;
  - a step the plan does not cover is a deviation: add it as its own item, and tell the user before doing it;
  - do not skip or reorder a plan's steps without saying why.

  The system prompt's `tools` section gains one line when `todo` is offered: `- For work of three or more steps, keep the todo list current: update it before each step and as soon as it is done.`
- **D6. A cancelled item on the wire.** In the ACP `plan`, a `cancelled` item is sent with:
  - status `pending`;
  - content prefixed `"(cancelled) "`;
  - `_meta: {"gobble": {"status": "cancelled"}}`.

  This is mcremote's rule for opencode and Kilo (F6). A client that reads gobble's `_meta` can show the true status; gobble's own CLI does (D12). Every other status is sent as it is, and every entry carries its priority.
- **D7. Resume sends the list.** `session/resume` sends the path's last non-empty list as a `plan` update before it answers, with the session-start frames. This covers gobble's CLI after resume, `/fork` and `/clone`. `session/load` keeps its replay: the last list, once, after the history.
- **D8. Turn end keeps the client current.** When a prompt's turn ends (`agent.End`) and the session has a non-empty list, the server re-sends it as the turn's last update, before the response. This happens whether or not the list changed. That keeps the latest list at the end of every turn, which is where a reconnecting client's history page looks (F5). A cancelled turn (`StopCancelled`) first sets its `in_progress` item back to `pending`, records the list as a `gobble.todo` entry, and then sends it, so no item spins after a cancel.
- **D9. A finished list is cleared at the next prompt.** When a prompt starts and the last list has every item `completed` or `cancelled`, the server records an empty `gobble.todo` entry and sends an empty `plan` before the turn runs. The user saw the finished list at the last turn's end. New work then starts with no list, until the model makes one. A list with any item not done is never cleared by the server.
- **D10. Compaction gives the model its list back.** When `compactSession` writes its compaction entry and the list has an item not done, it writes a `custom_message` after the entry:
  - `customType: "gobble.todo_reminder"`, `display: false`;
  - content: `Your todo list from before the conversation was compacted. It is still active: keep working through it in order, and keep it current with the todo tool.`, followed by the list in `/todos`'s form.

  The model sees it, because a `custom_message` is part of its context (`compaction/estimate.go:200-205`). No client sees it, because replay hides a message whose `display` is not true. The list itself is not reset. This applies to `/compact` and `_gobble/compact`, and to F5's automatic compaction when that is built.
- **D11. Plan mode feeds the list.**
  - **`exit_plan_mode`'s list** is parsed from list items only: lines that start with a bullet, a checkbox or `N.` at the plan's shallowest list indent. Headings, prose and nested items are left out. A plan with no list items sends no `plan` update.
  - **The preview stays.** The list is still sent before the question, so the phone shows the plan while the user decides (0005-MADR, "Plan visible on the phone").
  - **On "Start implementing",** the list is recorded as a `gobble.todo` entry, all `pending`. The tool's answer tells the model `The todo list now holds the plan's N steps: mark step 1 in_progress, and keep the list current as you work.`
  - **On "Keep planning",** the server re-sends the list as it was before the preview: the last recorded one, or empty. Nothing is recorded.
  - **The `todo` tool in plan mode** is refused, as codex refuses `update_plan` there, with: `In plan mode, put the steps in the plan; they become the todo list when the user approves it.`
- **D12. gobble's own client shows the list exactly.**
  - `acpclient`'s `PlanEntry` gains `Cancelled`, read from `_meta.gobble.status` with the `"(cancelled) "` prefix removed;
  - the CLI's plan rendering shows a cancelled item struck through and dimmed, and marks a `high` item, as the phone does;
  - `/todos` prints `[-]` for cancelled and `!` after the mark for high priority.
- **D13. A resync read, `_gobble/get_todos {sessionId}`.** It returns `{todos: [{content, status, priority}]}`, the path's last list in gobble's own statuses, `cancelled` included. It is answered while a prompt runs, as the other reads are. It is the ACP counterpart of mcremote's opencode `GET /session/{id}/todo` (F5).
  - It is added to the extensions advertised in `initialize`. So `session.golden:2` changes in exactly that array, by the one name. No other byte of any golden changes.
  - It is the seventeenth method. The sixteen frozen ones (0002-PLAN Phase 7) are unchanged.
- **D14. The magic-cli-remote findings go to [0009-REPORT-magic-cli-remote-findings.md](../reports/0009-REPORT-magic-cli-remote-findings.md),** by its rule, in the same change as this record:
  - **M11:** the replay duplicate check drops replayed plans.
  - **M12:** a cold-started phone loses the plan. The ring evicts plan events early, the app cache leaves the plan out, and nothing gives a reconnecting client the latest one.
  - **M13:** grok's cancelled items show as done, because `_meta` is dropped.
  - **M14:** ACP providers have no todo resync; gobble offers `_gobble/get_todos` once this record is built.
- **D15. These are not done here, and why:**
  - **Item ids and `merge`** (grok). ACP's `plan` and the phone replace the whole list, so ids would exist only inside gobble. D2's order rule needs the whole list anyway.
  - **A per-turn reminder** (goose), and **a turn-end gate that keeps the agent going while items remain** (grok's opt-in TodoGate). Each changes how long a turn runs, and needs a setting. gobble has no configuration surface yet (`internal/config` is a placeholder). They belong to the record that builds it.
  - **Sub-agents** (0005-PLAN X3). When built, a child session is not offered `todo` by default, as opencode and Kilo deny `todowrite` to sub-agents. A child's list never replaces the parent's, matching mcremote's rule (its 0051-MADR D7). This is a constraint on X3, recorded there by a dated note.
  - **The unstable `plan_update` and `plan_removed`** (F7; 0016-MADR D12).

### Consequences

- Good: the list on every client matches the session's at every point F2 names. On mcremote that holds with no mcremote change, for a phone that stays connected.
- Good: the agent is held to the list's order by the tool itself. The order is visible, so a user watching the phone sees each step go from pending to in progress to completed, in the plan's words.
- Good: after compaction the model keeps working from its list rather than re-deriving it.
- Good: mcremote gets the same remedy for gobble it built for opencode, a resync read, when it wants it.
- Bad: D2 refuses lists that other harnesses accept. A model used to free-form lists may need a retry. The error names the fix, but the count of refused calls is **[unverified]** until the plan's live run measures it.
- Bad: D8 adds one `plan` frame to every turn of a session with a list. The phone treats an identical list as a no-op, so the cost is the frame alone.
- Bad: D9 clears a finished list on the next prompt. A user who wanted to keep it on screen loses it then.
- Neutral: a cold-started phone can still miss the list until mcremote acts on M11–M14 (F5). D8 makes it more likely the latest list is in the history page it fetches.
- Neutral: the session file stays Pi's format. `gobble.todo` stays a `custom` entry, and the reminder is a `custom_message`, both of which Pi reads and ignores or shows by `display`.

### Confirmation

```text
go test ./tool/builtin -run 'TestTodo'                         -> ok: D1 statuses, D2 order and one-in-progress errors (exact text), D3 limits, D4 result text
go test ./acpserver -run 'TestTodoPlan|TestTodoCleared'        -> ok, unchanged
go test ./acpserver -run 'TestTodoCancelledOnTheWire'          -> ok: pending, "(cancelled) " prefix, _meta.gobble.status
go test ./acpserver -run 'TestResumeSendsTodos'                -> ok: plan sent before the resume answer; none for an empty list
go test ./acpserver -run 'TestTurnEndResendsTodos|TestCancelDemotesInProgress' -> ok
go test ./acpserver -run 'TestFinishedListClearedAtNextPrompt' -> ok
go test ./acpserver -run 'TestCompactionTodoReminder'          -> ok: in the model's context, not in a load replay
go test ./acpserver -run 'TestPlanModeTodos'                   -> ok: list items only; recorded on approve; restored on keep-planning; todo refused in plan mode
go test ./acpserver -run 'TestGetTodos'                        -> ok, also while a prompt runs
go test ./acpclient -run 'TestTranslatePlanCancelled'          -> ok
go test ./acpserver -run '^TestSession$'                       -> ok; git diff acpserver/testdata/session.golden shows only "_gobble/get_todos" added to line 2's extensions
make preflight                                                 -> preflight passed (Windows and WSL)
```

Each new test is seen failing first, on a scratch copy, with its rule removed or inverted:

- order unchecked;
- resume silent;
- no turn-end send;
- no demotion on cancel;
- no reminder;
- prose parsed as plan steps;
- no restore on keep-planning;
- cancelled sent as `completed`.

## Pros and Cons of the Options

### A — Whole list, exact (chosen)

- Good, because the live path is the one mcremote already shows, so the phone needs nothing new.
- Good, because order and a single in-progress item are checked in code, which none of the six harnesses does.
- Good, because every point where the list goes wrong is fixed in gobble's server, where the session state lives.
- Good, because it reuses gobble's entry kinds and Pi's format.
- Bad, because D2 is stricter than any harness the model may know, so a model can be refused until it learns the rule.
- Bad, because it adds a lifecycle at five points (resume, turn start, turn end, cancel, compaction), each needing its own test.

### B — grok's model

- Good, because `merge` by id sends less on a long list, and changing one status needs only the id. This is the strongest argument for it.
- Good, because grok's model is the one known to work on mcremote.
- Bad, because what makes grok work on mcremote is its `plan` frame, not its ids (F1). gobble already sends that frame.
- Bad, because its turn-end cleanup shows unfinished items as completed, and its cancelled items arrive on the phone as done (F6).
- Bad, because a merged list has no single sent value to check order against, so D2 would need the server to rebuild the list before checking.

### C — goose's notes

- Good, because notes that survive compaction and re-enter the context every turn keep the model oriented without any tool discipline.
- Bad, because free text is not a list: no status, no order, and nothing for the phone's panel, since goose sends no `plan`.
- Bad, because a block injected each turn costs tokens on every turn, and changes what is cached.

### D — The unstable plan updates

- Good, because plan ids and Markdown plans fit a PLAN file's phases better than a flat list.
- Bad, because mcremote handles neither `plan_update` nor the `plan` capability, so nothing would show on the phone (F7).
- Bad, because 0016-MADR keeps gobble off unstable surfaces.

### E — Prompt text only

- Good, because it is the smallest change: a better description, and nothing else.
- Bad, because the evidence of six harnesses is that prompt text does not hold order (F3).
- Bad, because it fixes none of F2's points where the list goes wrong, and none of F4 or F5.

## More Information

### Evidence index

| Claim | Source |
| :--- | :--- |
| The tool's input, checks and result; no order, limits or cancelled | `tool/builtin/todo.go:15-93`; `tool/builtin/describe/todo.md:1-17`; `tool/tool.go:187-200` |
| The live plan frame, and the recorded entry | `acpserver/prompt.go:111-113`, `:206-212`; `acpserver/agent.go:632-634`; `acpserver/history.go:381-401` |
| Load replays the last list; resume sends none | `acpserver/history.go:167-176`, `:278-284`; `acpserver/agent.go:786-800` |
| The CLI resumes, also after fork and clone | `internal/cli/sessions.go:61`; `internal/cli/slash.go:160` |
| exit_plan_mode's preview, its parser, no record; todo allowed in plan mode | `acpserver/modes.go:90-105`, `:137-175`, `:179-193` |
| `custom` entries never reach the model; `custom_message` does; replay hides it | `compaction/estimate.go:200-205`, `:214-219`; `acpserver/history.go:310-313` |
| Turn end touches no list | `acpserver/agent.go:488-502` |
| CLI drops priority; `/todos` marks | `internal/cli/view.go:126-133`; `acpserver/history.go:405-421` |
| What `todo_test.go` covers | `acpserver/todo_test.go:74`, `:124` |
| `get_state` has no list; the golden lists the extensions | `acpserver/extensions.go:349-384`; `acpserver/testdata/session.golden:2` |
| The sixteen frozen methods | `0002-PLAN-cli-acp-headless-mcp-v1.md:1831-1842`; 0009-REPORT M9 |
| mcremote's providers; 0179 proposed | magic-cli-remote `internal/provider/provider.go:64-72`; `docs/decisions/0179-MADR-pigo-native-acp-provider.md:2` |
| The generic plan path, and its field mapping | `internal/provider/acpagent/session.go:1740-1756`; `internal/provider/acpcommon/plan.go:10-48` |
| Plan is control traffic; empty means clear; progress-class retention | `internal/event/event.go:199-204`, `:624-632`; `internal/event/retention.go:24-28`, `:82-86` |
| Replay not broadcast; replay duplicate key ignores entries | `internal/session/manager.go:270`, `:369-379`, `:388-399`, `:1479-1484` |
| opencode/Kilo todo resync and its reason; cancelled convention | `internal/provider/opencode/todo.go:22-92`, `:55-57`; `resync.go:21-37`; `docs/decisions/0020-MADR-opencode-session-tree.md:247`, `:1027` |
| The app's reducer, panel and cache | `apps/mobile/lib/data/chat/transcript_reducer.dart:41-48`, `:958-968`; `lib/features/widgets/work_items_panel.dart`; `lib/data/chat/chat_models.dart:577-582`; `lib/data/chat/transcript_cache.dart:320-331` |
| grok's live frame, schema, cancelled mapping, turn-end cleanup, compaction reminder | grok-build `xai-grok-shell/src/session/acp_session_impl/tool_calls.rs:3049-3051`; `xai-grok-tools/.../todo/mod.rs:207-248`; `xai-grok-shell/src/tools/todo.rs:48-73`; `acp_session_impl/turn_end.rs:8-64`; `xai-grok-compaction/src/reminder.rs:216-222` |
| opencode removed its tool; its old ACP mapping | opencode `7feefb697f` (2026-07-09); `packages/core/src/database/v1-migration.bun.ts:901`; `v1.4.11` `acp/agent.ts:374-397` |
| Kilo's sequencing and verification rules | kilocode `packages/opencode/src/tool/todowrite.txt:24-30` |
| codex's rules, plan-mode refusal, no rollout | codex `codex-rs/protocol/src/prompts/base_instructions/default.md:271-275`; `core/src/tools/handlers/plan.rs:87-111`; `rollout/src/policy.rs:209` |
| goose's notes and per-turn re-injection | goose `crates/goose/src/agents/platform_extensions/todo.rs:165-180`; `state_machine/inference_preparation.rs:79-93` |
| pi's hidden "remaining steps" message | pi `packages/coding-agent/examples/extensions/plan-mode/index.ts:230-245` |
| ACP `plan` semantics; no cancelled; unstable plan updates | acp-go-sdk2 `v0.15.0` `schema/schema.json` (`Plan`, `PlanEntry`, `PlanEntryStatus`); `schema.unstable.json`; 0016-MADR |

### Related records

- [0005-MADR-v1-feature-scope.md](0005-MADR-v1-feature-scope.md): the `todo` row, and "Plan visible on the phone". [0005-PLAN-v1-feature-scope.md](0005-PLAN-v1-feature-scope.md): F1d-1, which built the tool, and X3, which D15 constrains.
- [0002-PLAN-cli-acp-headless-mcp-v1.md](0002-PLAN-cli-acp-headless-mcp-v1.md): the frozen `_gobble/` methods (Phase 7).
- [0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md](0016-MADR-acp-go-sdk2-v0.15.0-and-stable-surface.md): stable surfaces only, and the golden conformance check that would cover D13's frame.
- [0009-REPORT-magic-cli-remote-findings.md](../reports/0009-REPORT-magic-cli-remote-findings.md): M11–M14 (D14).
- magic-cli-remote `docs/decisions/0020-MADR-opencode-session-tree.md` (KD5, the resync), `0051-MADR-auto-approve-chat-noise.md` (D7, a child's list) and `0179-MADR-pigo-native-acp-provider.md` (gobble's provider, proposed).

### Open questions for the plan

None that is a decision. The plan settles these by reading the code:

- where D9's prompt-start clear and D8's turn-end send sit relative to `startFrames`, the queue and the usage update in `Prompt`;
- the exact text of the rewritten `todo.md`, held to D5's rules and to 0012-MADR's description format.
