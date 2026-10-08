Run a command with PowerShell (pwsh when installed, else Windows PowerShell), in the working directory or in `workdir`.

## Use when

- A command needs PowerShell: cmdlets, Windows paths, or the registry.
- Use `workdir` instead of `Set-Location`.
- Not for reading, editing or writing files: use read, edit and write rather than `Get-Content` and `Set-Content`.

## Returns

- The output, as the last {{.MaxLines}} lines or {{.MaxKB}} KB, ending with `[exit status N]`.
- When the output is longer, the whole of it is saved to a file the output names, which read can open.

## Rules

- `timeout` is in seconds: {{.DefaultTimeout}} when not given, at most {{.MaxTimeout}}. On timeout, the command and everything it started are stopped.
- Windows PowerShell 5.1 has no `&&` or `||`: separate commands with `;` and check `$LASTEXITCODE`.
