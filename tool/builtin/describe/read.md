Read a text file with numbered lines, or list a directory's entries.

## Use when

- You need a file's content, or a directory's entries, before acting on them.
- Prefer it to `cat`, `head`, `tail` and `Get-Content` in a shell.
- Not for searching files for text: use bash with `grep`.

## Returns

- Each line as `N: text`. The numbers are not part of the file: leave them out of an edit's `oldText`.
- For a directory, one entry per line, with a `/` after each directory.
- When the output is cut, a last line giving the `offset` to continue from.
- For a missing path, an error naming similar entries, when there are any.

## Rules

- At most {{.MaxLines}} lines or {{.MaxKB}} KB per call; a line over {{.MaxLineChars}} characters is cut.
- Read a large file in a few big pieces, not many small ones.
- Binary files are refused. An image returns a note instead of its content, for now.
