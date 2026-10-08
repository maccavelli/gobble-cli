Edit one file by replacing exact pieces of its text.

## Use when

- Changing part of an existing file.
- Not for creating a file, or rewriting most of one: use write.

## Returns

- A line such as `edited main.go: 2 replacements`.

## Rules

- Each `edits[].oldText` must occur exactly once in the file as it was before this call, and no two may overlap: put changes to nearby lines into one edit.
- Keep `oldText` short but unique, and copy it from the file without read's line numbers.
- Small differences in whitespace, indentation, quotes and escaping are forgiven, but a forgiving match far larger than `oldText` is refused.
- If another process changes the file during the edit, the edit fails: read the file again and resend it.

## Example

- `{"path": "main.go", "edits": [{"oldText": "retries := 3", "newText": "retries := 5"}]}`
