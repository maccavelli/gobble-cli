Run a command with bash, in the working directory or in `workdir`.

## Use when

- Use `workdir` instead of `cd`.
- For files, prefer read, edit and write to `cat`, `sed` and `echo`.

## Rules

- stdout and stderr come back together, as the last {{.MaxLines}} lines or {{.MaxKB}} KB, ending with the exit status. A longer output is saved whole to a file the output names, which read can open.
- `timeout` is in seconds: {{.DefaultTimeout}} when not given, at most {{.MaxTimeout}}.
