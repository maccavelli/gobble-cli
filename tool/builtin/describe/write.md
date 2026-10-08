Write a whole file: create it with any missing parent directories, or replace all of its content.

## Use when

- Creating a new file.
- Replacing a file whose content changes throughout.
- Not for changing part of an existing file: use edit, which sends less and leaves the rest of the file exactly as it was.

## Returns

- A line such as `wrote main.go (12 lines)`.

## Rules

- The write is atomic: a reader sees the old file or the new one, never a mix.
- `content` is the whole file. Anything not in it is gone after the write.
