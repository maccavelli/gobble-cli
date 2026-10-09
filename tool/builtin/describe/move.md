Move or rename a file or a directory.

## Use when

- Renaming, or moving something to another directory, including across the workspace's directories.
- Prefer it to `mv`, `Move-Item` and `ren` in a shell.
- Not for keeping the original as well: use copy.

## Returns

- A line such as `moved old.go to new.go`.

## Rules

- An existing `to` is refused unless `overwrite` is set.
- Across directories that are not on one device, it copies and then deletes, and a failed copy leaves the original as it was.
