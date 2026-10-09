Create a directory, with any missing parents.

## Use when

- Preparing a directory before files are written into it, though write creates missing parents itself.
- Prefer it to `mkdir -p`, `New-Item -ItemType Directory` and `md` in a shell.
- Not for creating a file: use write.

## Returns

- A line such as `created docs/guides`, or that it already exists.

## Rules

- A directory that already exists is success; an existing file of that name is refused.
