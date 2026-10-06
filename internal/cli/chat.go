package cli

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
)

// ChatCmd is the default command (0008-MADR D1).
type ChatCmd struct {
	SharedFlags
	Prompt []string `arg:"" optional:"" complete:"none" help:"Prompt words; @path attaches a file."`
}

// Run validates the flags, resolves the mode and the working directory,
// composes the input, then runs print mode or the line session over ACP
// (0002-PLAN Phase 2).
func (c *ChatCmd) Run(k *kong.Context, e *runEnv) error {
	g := givenFlags(k)
	stdinPiped := e.stdin != nil && !e.out.caps.In
	if err := c.validate(g, stdinPiped, hasAtPath(c.Prompt)); err != nil {
		return err
	}
	mode := resolveMode(c.Print, e.out.caps.In, e.out.caps.Out.TTY)
	tools, err := c.toolSet()
	if err != nil {
		return err
	}
	e.tools = tools
	cwd, base, err := resolveCwd(c.Cwd)
	if err != nil {
		return err
	}
	if err := c.configureStore(e, base); err != nil {
		return err
	}
	stdin, stdinTTY := e.stdinReader(), e.out.caps.In
	if c.File != "" {
		path, err := resolvePath(base, c.File)
		if err != nil {
			return usageErrorf("--file %s: %v", c.File, err)
		}
		b, err := os.ReadFile(path) //nolint:gosec // G304: the user names the file to read
		if err != nil {
			return usageErrorf("--file %s: %v", c.File, err)
		}
		stdin, stdinTTY = strings.NewReader(string(b)), false
	}
	prompt, err := composeInput(e.ctx, stdin, stdinTTY, c.Prompt, base, os.ReadFile)
	if err != nil {
		return err
	}
	if mode == ModePrint && prompt.Text == "" && len(prompt.Images) == 0 {
		return usageErrorf("no prompt: give words, @path, --file or piped input")
	}
	e.log().DebugContext(e.ctx, "chat",
		slog.String("mode", mode.String()), slog.Int("prompt_bytes", len(prompt.Text)), slog.Int("images", len(prompt.Images)))
	if mode == ModePrint {
		return runPrint(e, c, prompt, cwd, base)
	}
	return runLine(e, c, prompt, cwd, base)
}

// resolveCwd is the session's working directory: --cwd made absolute, which
// must be a directory, or the process's own. base is what relative @path
// and --file resolve against: the --cwd directory, or "" for the process's
// working directory. gobble never changes its own directory.
func resolveCwd(dir string) (cwd, base string, err error) {
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", "", failf("working directory: %v", err)
		}
		return wd, "", nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", "", usageErrorf("--cwd %s: %v", dir, err)
	}
	if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
		return "", "", usageErrorf("--cwd %s: not a directory", dir)
	}
	return abs, abs, nil
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
