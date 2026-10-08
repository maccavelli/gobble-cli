Edit one file by replacing text.

## Rules

- Each `edits[].oldText` must occur once in the file as it was before this call, and no two may overlap: put changes to nearby lines into one edit.
- Keep `oldText` short but unique, and copy it from the file without read's line numbers.
- Small differences in whitespace, indentation, quotes and escaping are forgiven, but a forgiving match far larger than `oldText` is refused.
