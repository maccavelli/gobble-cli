Read a file, or list a directory's entries.

## Use when

- Read a large file in a few big pieces rather than many small ones.

## Rules

- Lines come numbered as `N: text`. The numbers are not part of the file, so leave them out of an edit's `oldText`.
- At most {{.MaxLines}} lines or {{.MaxKB}} KB are shown, and a line longer than {{.MaxLineChars}} characters is cut. The last line of the output gives the `offset` to continue from.
- Binary files are refused, and images are not sent to the model yet.
