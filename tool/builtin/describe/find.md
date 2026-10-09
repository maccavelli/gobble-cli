Find files and directories whose names match a glob, under a directory.

## Use when

- Locating files by name or extension, such as `*_test.go` or `docs/**/*.md`.
- Prefer it to `find`, `fd`, `ls -R` and `Get-ChildItem -Recurse` in a shell.
- Not for searching inside files: use grep.

## Returns

- One path per line, relative to the search root, with a `/` after each directory, sorted by path.
- When nothing matches, a line saying so. That is not an error.

## Rules

- A glob without a `/` matches names at any depth; with a `/` it matches the path from the search root. `**` crosses directories, and `{a,b}` gives alternatives.
- Entries in `.gitignore`, `.git` and `node_modules` are skipped. Hidden entries are listed.
- At most `limit` paths and 50 KB are shown.
