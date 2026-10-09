Show the layout of a directory as an indented tree, a few levels deep.

## Use when

- Getting to know a project, or a directory, before reading or changing it.
- Prefer it to `tree`, `ls -R` and `Get-ChildItem -Recurse` in a shell.
- Not for one directory's entries: use read on the directory.

## Returns

- The directory, then its entries indented two spaces a level: directories first, with a `/`, then files, with their sizes.
- When the entry budget runs out, each unexpanded directory ends in an ellipsis, `…`, and a last line says how many.

## Rules

- `depth` levels are shown, {{.TreeDepth}} by default. Each level is listed in full before the next is started.
- Entries in `.gitignore`, `.git` and `node_modules` are skipped. Hidden entries are shown.
- At most `limit` entries are shown.
