package cli

import (
	"context"
	"errors"
	"iter"
	"os"
	"path/filepath"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
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
var thinkingLevels = []string{"off", "minimal", "low", "medium", "high", "xhigh", "max"}

// completionRegistry names the source behind each complete:"<name>" tag.
// Sessions come from the session store (0002-PLAN Phase 4) and tools from
// the built-ins (0002-PLAN Phase 3, wired in Phase 4 Part B), and MCP
// servers from mcp.json (0002-PLAN Phase 5). Providers are the key
// providers, and models the offline model set (0005-PLAN F4-2c).
func completionRegistry() complete.Registry {
	return complete.Registry{
		"path":      complete.PathSource(false),
		"dir":       complete.PathSource(true),
		"provider":  complete.ListSource(providerCandidates, true),
		"session":   complete.ListSource(sessionCandidates, true),
		"model":     complete.ModelSource(modelCandidates, thinkingLevels),
		"tools":     complete.ListSource(toolCandidates, false),
		"mcpserver": complete.ListSource(mcpServerCandidates, true),
		"setting":   complete.ListSource(settingCandidates, true),
		// The personalities, built-in and the user's (0022-PLAN P3).
		"personality": complete.ListSource(personalityCandidates, true),
		// The session modes (0005-PLAN F6-4).
		"mode": complete.ListSource(modeCandidates, true),
	}
}

// settingCandidates are the settings' paths, in the template's order, each
// described by its documentation (0018-MADR D12).
func settingCandidates(context.Context) iter.Seq[complete.Candidate] {
	return func(yield func(complete.Candidate) bool) {
		for _, k := range config.Keys {
			if !yield(complete.Candidate{Value: k.Path, Description: k.Doc}) {
				return
			}
		}
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

// providerCandidates are the providers --provider takes, in registry order,
// each described by its credential variable.
func providerCandidates(context.Context) iter.Seq[complete.Candidate] {
	return func(yield func(complete.Candidate) bool) {
		for _, p := range provider.KeyProviders() {
			if !yield(complete.Candidate{Value: p, Description: provider.CredentialVar(p)}) {
				return
			}
		}
	}
}

// modelCandidates are the offline model set as provider/id, each described
// by its name, with models.json read and its diagnostics dropped, since
// completion prints nothing else (0005-PLAN F4-2c decision 10).
func modelCandidates(ctx context.Context) iter.Seq[complete.Candidate] {
	return func(yield func(complete.Candidate) bool) {
		for _, m := range offlineModels(ctx, completionModelsJSON()) {
			if ctx.Err() != nil || !yield(complete.Candidate{Value: m.Provider + "/" + m.ID, Description: m.Name}) {
				return
			}
		}
	}
}

// completionModelsJSON is <config>/models.json, or nil when there is none
// or it cannot be read.
func completionModelsJSON() *catalog.ModelsJSON {
	dirs, _, err := systemDirs()
	if err != nil {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(dirs.Config, "models.json")) //nolint:gosec // G304: the config directory's own models.json
	if err != nil {
		return nil
	}
	mj, _, err := catalog.ParseModelsJSON(data, provider.SDKVersion())
	if err != nil {
		return nil
	}
	return mj
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
