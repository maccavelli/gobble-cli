package builtin

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/tool"
)

// waitDelay is how long a cancelled command's output pipes stay open after
// it is killed, so a grandchild holding them cannot hold the turn.
const waitDelay = 2 * time.Second

type bashIn struct {
	Command string `json:"command" jsonschema:"the command line, run by bash -c in the working directory"`
	Timeout int    `json:"timeout,omitzero" jsonschema:"seconds before the command is killed; 0 is no limit"`
}

// Bash is the bash tool: bash -c in the session's working directory, its
// combined output cut to the last 2,000 lines or 50 KB. A non-zero exit is
// an error result. 0005-PLAN F1 adds process groups, the full-output file
// and powershell.
func Bash() tool.Tool {
	return tool.New("bash", "Run a command with bash -c in the working directory. Output is the last 2000 lines or 50 KB.",
		func(ctx context.Context, in bashIn, env tool.Env) (tool.Result, error) {
			shell, err := findBash()
			if err != nil {
				return tool.Result{}, err
			}
			if in.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(in.Timeout)*time.Second)
				defer cancel()
			}
			var out bytes.Buffer
			cmd := exec.CommandContext(ctx, shell, "-c", in.Command) //nolint:gosec // G204: running the model's command is this tool's purpose, behind the session's approval
			cmd.Dir = env.Cwd
			cmd.Stdout, cmd.Stderr = &out, &out
			cmd.WaitDelay = waitDelay
			runErr := cmd.Run()
			text := tail(out.String())
			status := 0
			if exit, ok := errors.AsType[*exec.ExitError](runErr); ok {
				status = exit.ExitCode()
			} else if runErr != nil && ctx.Err() == nil {
				return tool.Result{}, fmt.Errorf("bash: %w", runErr)
			}
			if ctx.Err() != nil && in.Timeout > 0 && errors.Is(ctx.Err(), context.DeadlineExceeded) {
				r := tool.ErrorResult(fmt.Sprintf("%s\n[killed after %d s]", text, in.Timeout))
				r.Summary = summary(fmt.Sprintf("killed after %d s", in.Timeout))
				return r, nil
			}
			r := tool.TextResult(fmt.Sprintf("%s\n[exit status %d]", strings.TrimSuffix(text, "\n"), status))
			r.IsError = status != 0
			first, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
			r.Summary = summary(strings.TrimSpace(fmt.Sprintf("exit status %d: %s", status, first)))
			return r, nil
		},
		tool.WithKind(tool.KindExecute),
		tool.WithAnnotations(tool.Annotations{DestructiveHint: true, OpenWorldHint: true}),
		tool.WithDescribe(func(c tool.Call, _ tool.Env) string {
			var a bashIn
			if err := json.Unmarshal(c.Args, &a); err != nil || a.Command == "" {
				return "run ?"
			}
			return title("run " + a.Command)
		}),
	)
}

// findBash is bash on PATH. On Windows, System32's bash.exe is the WSL
// launcher, which would run the command in another system, so Git for
// Windows' bash beside git is preferred.
func findBash() (string, error) {
	if runtime.GOOS == "windows" {
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
	if runtime.GOOS == "windows" && strings.EqualFold(filepath.Base(filepath.Dir(p)), "System32") {
		return "", errors.New("bash: only the WSL launcher is on PATH; install Git for Windows")
	}
	return p, nil
}
