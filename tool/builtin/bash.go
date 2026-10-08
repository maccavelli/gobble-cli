package builtin

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/maccavelli/gobble-cli/tool"
)

const bashDescription = "Run a command with bash, in the working directory or in workdir (use workdir instead of cd). " +
	"stdout and stderr come back together, as the last 2000 lines or 50 KB, ending with the exit status; " +
	"a longer output is saved whole to a file the output names, which read can open. " +
	"timeout is in seconds: 120 when not given, at most 600. " +
	"For files, prefer read, edit and write to cat, sed and echo."

// Bash is the bash tool with the default options.
func Bash() tool.Tool { return bashWith(Options{}.withDefaults()) }

// bashWith is the bash tool: bash -c, or o.ShellPath, with
// o.ShellCommandPrefix before the command, run by the shared runner
// (shell.go).
func bashWith(o Options) tool.Tool {
	sh := shell{
		name: "bash",
		find: func() (string, error) {
			if o.ShellPath != "" {
				return o.ShellPath, nil
			}
			return findBash()
		},
		args: func(command string) []string {
			if o.ShellCommandPrefix != "" {
				command = o.ShellCommandPrefix + "\n" + command
			}
			return []string{"-c", command}
		},
	}
	return tool.New("bash", bashDescription,
		func(ctx context.Context, in shellIn, env tool.Env) (tool.Result, error) {
			return runShell(ctx, sh, o, in, env)
		},
		tool.WithKind(tool.KindExecute),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true, OpenWorldHint: true}),
		tool.WithOutside(shellOutside),
		tool.WithDescribe(shellDescribe),
	)
}

// findBash is bash on PATH. On Windows, System32's bash.exe is the WSL
// launcher, which would run the command in another system, so Git for
// Windows' bash beside git is preferred.
func findBash() (string, error) {
	if runtime.GOOS == goosWindows {
		if git, err := exec.LookPath("git"); err == nil {
			for _, rel := range []string{`..\bin\bash.exe`, `..\..\bin\bash.exe`} {
				if p, err := exec.LookPath(filepath.Join(filepath.Dir(git), rel)); err == nil {
					return p, nil
				}
			}
		}
	}
	p, err := exec.LookPath("bash")
	if err != nil {
		return "", errors.New("bash: no bash on PATH")
	}
	if runtime.GOOS == goosWindows && strings.EqualFold(filepath.Base(filepath.Dir(p)), "System32") {
		return "", errors.New("bash: only the WSL launcher is on PATH; install Git for Windows")
	}
	return p, nil
}
