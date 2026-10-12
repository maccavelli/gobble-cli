package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"

	"github.com/maccavelli/gobble-cli/internal/config"
)

// parseOverrides parses each --set PATH=VALUE (0018-MADR D10). An error is
// a usage error, in ParseOverride's words.
func parseOverrides(args []string) ([]config.Override, error) {
	out := make([]config.Override, 0, len(args))
	for _, a := range args {
		o, err := config.ParseOverride(a)
		if err != nil {
			return nil, usageErrorf("%v", err)
		}
		out = append(out, o)
	}
	return out, nil
}

// settingsOptions are Resolve's options for cwd: the global file, cwd's
// project file, and --set. No project is trusted until 0005-PLAN F6.
func (e *runEnv) settingsOptions(cwd string) (config.Options, error) {
	d, _, err := systemDirs()
	if err != nil {
		return config.Options{}, failf("%v", err)
	}
	return config.Options{ConfigDir: d.Config, Cwd: cwd, Overrides: e.overrides}, nil
}

// ensureSettingsFile creates <config>/settings.yaml from the template when
// it is missing (0018-MADR D9), as the first run of a command that loads
// settings does. A failure is a warning, and the run goes on.
func (e *runEnv) ensureSettingsFile() {
	d, _, err := systemDirs()
	if err != nil {
		return // the command reports the directories' error itself
	}
	file := config.GlobalFile(d.Config)
	if _, err := os.Stat(file); !errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err := config.Init(file); err != nil {
		if _, serr := os.Stat(file); serr == nil {
			return // another gobble created it first
		}
		e.warnf("create %s: %v", file, err)
	}
}

// warnf warns on stderr, or under `gobble acp`, whose stdio is the
// protocol, in the log only (0018-MADR D7).
func (e *runEnv) warnf(format string, args ...any) {
	if e.logs != nil && e.logs.stderr == nil {
		e.log().Warn("settings", slog.String("message", fmt.Sprintf(format, args...)))
		return
	}
	e.out.Warnf(format, args...)
}
