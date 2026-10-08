Run a command with PowerShell (pwsh when installed, else Windows PowerShell), in the working directory or in `workdir`.

## Use when

- Use `workdir` instead of `Set-Location`.

## Rules

- Output comes back as the last {{.MaxLines}} lines or {{.MaxKB}} KB, ending with the exit status. A longer output is saved whole to a file the output names, which read can open.
- `timeout` is in seconds: {{.DefaultTimeout}} when not given, at most {{.MaxTimeout}}.
- Windows PowerShell 5.1 has no `&&` or `||`: separate commands with `;` and check `$LASTEXITCODE`.
