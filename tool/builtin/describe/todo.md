Keep a short checklist of the steps of the current task, shown to the user as a plan.

## Use when

- A task has three or more steps, or the user gave a list of things to do.
- Update it as steps start and finish, so the user can follow progress.
- Not for notes or results: use write for a file, or answer in text.

## Returns

- A line counting the items, such as `todo list updated: 2 completed, 1 in progress, 3 pending`.

## Rules

- Each call replaces the whole list: send every item, with its status.
- Keep one item `in_progress` at a time, and mark it `completed` when it is done.
- An empty `todos` clears the list.
