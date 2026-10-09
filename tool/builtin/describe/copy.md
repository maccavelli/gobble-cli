Copy a file, or a directory with everything in it.

## Use when

- Duplicating a file or a directory, keeping the original.
- Prefer it to `cp`, `Copy-Item` and `copy` in a shell.
- Not for renaming or relocating: use move.

## Returns

- A line such as `copied a.txt to b.txt`, with the entry count for a directory.

## Rules

- An existing `to` is refused unless `overwrite` is set.
- Permission bits are kept. A symbolic link is copied as a link, and not followed.
