package cli

import (
	"errors"
	"os"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
)

// CompletionCmd prints a shell's registration script (0008-MADR D17).
type CompletionCmd struct {
	Shell string `arg:"" enum:"bash,zsh,fish,powershell" help:"Shell: ${enum}."`
}

// Help is the command's detailed help: installation, and completion mode.
func (*CompletionCmd) Help() string {
	return `Generate the script when the shell starts, so it always matches the binary:

  bash        add to ~/.bashrc:                   source <(gobble completion bash)
  zsh         add to ~/.zshrc after compinit:     source <(gobble completion zsh)
  fish        add to ~/.config/fish/config.fish:  gobble completion fish | source
  PowerShell  add to $PROFILE:                    gobble completion powershell | Out-String | Invoke-Expression

The script calls gobble back in completion mode: GOBBLE_COMPLETE=<shell> with
GOBBLE_COMPLETE_PROTOCOL=1 makes gobble print candidates for the words it is
given, then exit. Completion mode reads only local files, writes nothing, and
answers within 150 ms. Set GOBBLE_COMPLETE_DEBUG=<file> to log each request.`
}

// Run prints the script.
func (c *CompletionCmd) Run(e *runEnv) error {
	s, err := complete.Script(c.Shell, "gobble")
	if err != nil {
		return failf("%v", err)
	}
	return e.out.Result([]byte(s))
}

// thinkingLevels are --thinking's values, offered after "provider/id:".
var thinkingLevels = []string{"off", "minimal", "low", "medium", "high", "xhigh"}

// completionRegistry names the source behind each complete:"<name>" tag.
// The dynamic sources have no data yet, which is the true state, not a
// skip: sessions arrive in 0002-PLAN Phase 4, providers, models and their
// catalog in 0005-PLAN F4, and tools in 0002-PLAN Phase 3 (0008-PLAN P5,
// step 4 and deviation 1).
func completionRegistry() complete.Registry {
	return complete.Registry{
		"path":     complete.PathSource(false),
		"dir":      complete.PathSource(true),
		"provider": complete.ListSource(complete.NoCandidates, false),
		"session":  complete.ListSource(complete.NoCandidates, true),
		"model":    complete.ModelSource(complete.NoCandidates, thinkingLevels),
		"tools":    complete.ListSource(complete.NoCandidates, false),
	}
}

// completionMode reports whether the process was started by a completion
// script.
func completionMode(getenv func(string) string) bool {
	v := getenv(complete.EnvShell)
	return v != "" && v != "0"
}

// clearCompletionEnv removes the completion variables, so no child process
// gobble starts can inherit them (0008-MADR D17).
func clearCompletionEnv() error {
	var errs []error
	for _, n := range complete.EnvNames {
		errs = append(errs, os.Unsetenv(n))
	}
	return errors.Join(errs...)
}
