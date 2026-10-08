package builtin

import (
	"context"
	"errors"
	"os/exec"

	"github.com/maccavelli/gobble-cli/tool"
)

// utf8Prefix makes PowerShell write UTF-8, so non-ASCII output survives
// (0005-MADR, Tools, powershell row).
const utf8Prefix = "[Console]::OutputEncoding=[System.Text.Encoding]::UTF8;"

const powershellDescription = "Run a command with PowerShell (pwsh when installed, else Windows PowerShell), " +
	"in the working directory or in workdir (use workdir instead of Set-Location). " +
	"Output comes back as the last 2000 lines or 50 KB, ending with the exit status; " +
	"a longer output is saved whole to a file the output names, which read can open. " +
	"timeout is in seconds: 120 when not given, at most 600. " +
	"Windows PowerShell 5.1 has no && or ||; separate commands with ; and check $LASTEXITCODE."

// powershellWith is the powershell tool, on Windows: pwsh.exe, else
// powershell.exe, with no profile and no prompts, run by the shared runner.
func powershellWith(o Options) tool.Tool {
	sh := shell{
		name: "powershell",
		find: findPowerShell,
		args: func(command string) []string {
			return []string{"-NoLogo", "-NoProfile", "-NonInteractive", "-Command", utf8Prefix + command}
		},
	}
	return tool.New("powershell", powershellDescription,
		func(ctx context.Context, in shellIn, env tool.Env) (tool.Result, error) {
			return runShell(ctx, sh, o, in, env)
		},
		tool.WithKind(tool.KindExecute),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true, OpenWorldHint: true}),
		tool.WithOutside(shellOutside),
		tool.WithDescribe(shellDescribe),
		shellSchema(o),
	)
}

// findPowerShell prefers PowerShell 7 to Windows PowerShell 5.1.
func findPowerShell() (string, error) {
	for _, name := range []string{"pwsh.exe", "powershell.exe"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", errors.New("powershell: neither pwsh nor powershell is on PATH")
}
