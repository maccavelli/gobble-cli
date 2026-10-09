Delete a file, or a directory with everything in it.

## Use when

- Removing files or directories that are no longer needed.
- Prefer it to `rm`, `rmdir`, `Remove-Item` and `del` in a shell.
- Not for emptying a file while keeping it: use write with empty `content`.

## Returns

- A line such as `deleted build/ (42 entries)`.

## Rules

- Every delete asks the user first.
- A directory that is not empty needs `recursive`.
- A workspace root is never deleted.
