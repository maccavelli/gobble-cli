Run a command with bash, in the working directory or in `workdir`.

## Use when

- Building, testing, version control, and running other programs.
- Use `workdir` instead of `cd`.
- Not for reading, editing or writing files: use read, edit and write rather than `cat`, `sed` and `echo`.

## Returns

- stdout and stderr together, as the last {{.MaxLines}} lines or {{.MaxKB}} KB, ending with `[exit status N]`.
- When the output is longer, the whole of it is saved to a file the output names, which read can open.

## Rules

- `timeout` is in seconds: {{.DefaultTimeout}} when not given, at most {{.MaxTimeout}}. On timeout, the command and everything it started are stopped.
- Whatever the command leaves running is stopped when it ends, so it cannot start a server for later calls.
