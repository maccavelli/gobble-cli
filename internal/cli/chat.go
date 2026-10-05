package cli

import (
	"log/slog"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

// ChatCmd is the default command (0008-MADR D1).
type ChatCmd struct {
	SharedFlags
	Prompt []string `arg:"" optional:"" complete:"none" help:"Prompt words; @path attaches a file."`
}

// Run validates the flags, resolves the mode and composes the input. The
// agent arrives in 0002-PLAN Phase 2; until then every run ends there.
func (c *ChatCmd) Run(k *kong.Context, e *runEnv) error {
	g := givenFlags(k)
	stdinPiped := e.stdin != nil && !e.out.caps.In
	if err := c.validate(g, stdinPiped, hasAtPath(c.Prompt)); err != nil {
		return err
	}
	mode := resolveMode(c.Print, e.out.caps.In, e.out.caps.Out.TTY)
	stdin, stdinTTY := e.stdinReader(), e.out.caps.In
	if c.File != "" {
		b, err := os.ReadFile(c.File) //nolint:gosec // G304: the user names the file to read
		if err != nil {
			return usageErrorf("--file %s: %v", c.File, err)
		}
		stdin, stdinTTY = strings.NewReader(string(b)), false
	}
	prompt, err := composeInput(e.ctx, stdin, stdinTTY, c.Prompt, os.ReadFile)
	if err != nil {
		return err
	}
	if mode == ModePrint && prompt.Text == "" && len(prompt.Images) == 0 {
		return usageErrorf("no prompt: give words, @path, --file or piped input")
	}
	e.log().DebugContext(e.ctx, "chat",
		slog.String("mode", mode.String()), slog.Int("prompt_bytes", len(prompt.Text)), slog.Int("images", len(prompt.Images)))
	return usageErrorf("the agent is not available yet (0002-PLAN Phase 2)")
}

// givenFlags is the set of flags named on the command line. Kong records
// each in the parse path; defaults and environment values are not there.
func givenFlags(k *kong.Context) given {
	g := given{}
	for _, p := range k.Path {
		if p.Flag != nil && !p.Resolved {
			g[p.Flag.Name] = true
		}
	}
	return g
}
