package cli

import (
	"context"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool/builtin"
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
// Sessions come from the session store (0002-PLAN Phase 4) and tools from
// the built-ins (0002-PLAN Phase 3, wired in Phase 4 Part B), and MCP
// servers from mcp.json (0002-PLAN Phase 5). Providers,
// models and their catalog have no data until 0005-PLAN F4, which is the
// true state, not a skip (0008-PLAN P5, step 4 and deviation 1).
func completionRegistry() complete.Registry {
	return complete.Registry{
		"path":      complete.PathSource(false),
		"dir":       complete.PathSource(true),
		"provider":  complete.ListSource(complete.NoCandidates, false),
		"session":   complete.ListSource(sessionCandidates, true),
		"model":     complete.ModelSource(complete.NoCandidates, thinkingLevels),
		"tools":     complete.ListSource(toolCandidates, false),
		"mcpserver": complete.ListSource(mcpServerCandidates, true),
	}
}

// sessionCandidates are the stored sessions' ids, newest first, each
// described as "name · date · cwd" (0008-MADR D17). The store is only read:
// completion creates nothing.
func sessionCandidates(ctx context.Context) iter.Seq[complete.Candidate] {
	return func(yield func(complete.Candidate) bool) {
		dirs, _, err := systemDirs()
		if err != nil {
			return
		}
		for sum, err := range jsonl.New(filepath.Join(dirs.Data, "sessions"), jsonl.Options{}).List(ctx, session.Filter{}) {
			if err != nil {
				return
			}
			title := sum.Name
			if title == "" {
				title = sum.First
			}
			desc := strings.Join([]string{strings.Join(strings.Fields(title), " "), sum.Modified.Local().Format("2006-01-02 15:04"), sum.Cwd}, " · ")
			if !yield(complete.Candidate{Value: string(sum.ID), Description: desc}) {
				return
			}
		}
	}
}

// toolCandidates are the built-in tools' names.
func toolCandidates(context.Context) iter.Seq[complete.Candidate] {
	return func(yield func(complete.Candidate) bool) {
		for _, n := range builtin.Names() {
			if !yield(complete.Candidate{Value: n}) {
				return
			}
		}
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
