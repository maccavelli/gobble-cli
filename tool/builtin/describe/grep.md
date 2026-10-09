Search file contents for a regular expression, under a directory or in one file.

## Use when

- Finding where a name, a string or a pattern appears across files.
- Prefer it to `grep`, `rg` and `Select-String` in a shell.
- Not for finding files by name: use find.

## Returns

- One line per match, as `path:N: text`, and with `context` set, nearby lines as `path-N- text`.
- Paths relative to the search root, sorted by path, then by line.
- When nothing matches, a line saying so. That is not an error.

## Rules

- `pattern` is RE2 syntax: no lookaround, and no backreferences. Set `literal` to search for the text as it is.
- `glob` without a `/` matches file names at any depth, such as `*.go` or `*.{ts,tsx}`; with a `/` it matches the path from the search root.
- Files in `.gitignore`, `.git`, `node_modules`, binary files, and files over 8 MB are skipped. Hidden files are searched.
- At most `limit` matches and 50 KB are shown; a line over 500 characters is cut.
