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
	e.ensureSettingsFile()
	g := givenFlags(k)
	// With --api-key -, piped stdin is the key, never the input (0005-PLAN F4-3a).
	stdinPiped := e.stdin != nil && !e.out.caps.In && !g["api-key"]
	if err := c.validate(g, stdinPiped, hasAtPath(c.Prompt)); err != nil {
		return err
	}
	// --list-models is gobble models list, the prompt words its search, as
	// Pi's --list-models [search] (0005-MADR; 0005-PLAN F4-2c).
	if c.ListModels {
		return e.printModels(strings.Join(c.Prompt, " "))
	}
	mode := resolveMode(c.Print, e.out.caps.In, e.out.caps.Out.TTY)
	// The working directory comes first: the process settings, and so the
	// built-in tools' options, are its (0018-PLAN P5 deviation 15).
	cwd, base, err := resolveCwd(c.Cwd)
	if err != nil {
		return err
	}
	if e.pick, err = e.pickModel(&c.SharedFlags); err != nil {
		return err
	}
	if err := e.choosePersonality(c.Personality); err != nil {
		return err
	}
	// --api-key -'s key is the run's, for --model's provider, and is never
	// written; read from stdin, it leaves stdin no input (0005-PLAN F4-3a).
	keyFromStdin := false
	if g["api-key"] {
		key, err := e.readAPIKey(e.pick.Provider)
		if err != nil {
			return err
		}
		e.setRuntimeKey(e.pick.Provider, key)
		keyFromStdin = !e.out.caps.In
	}
	choice, err := c.toolSet(e.builtinTools(e.processSettings(cwd).Settings))
	if err != nil {
		return err
	}
	e.tools, e.mcpOff, e.mcpAllow = choice.builtins, choice.mcpOff, choice.mcpAllow
	if c.AddDir, err = resolveAddDirs(base, c.AddDir); err != nil {
		return err
	}
	if e.prompt, err = c.promptOptions(base); err != nil {
		return err
	}
	if err := c.configureStore(e, base); err != nil {
		return err
	}
	stdin, stdinTTY := e.stdinReader(), e.out.caps.In
	if keyFromStdin {
		stdin = nil
	}
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
