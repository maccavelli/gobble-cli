package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/llm"
)

// processOnce holds the process settings, resolved once a run.
type processOnce struct {
	once sync.Once
	r    config.Resolved
}

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

// processSettings are the process keys' settings (0018-MADR D16): resolved
// once, for the first cwd asked, and cached, so their diagnostics print
// once a run. `gobble acp` asks with "", which reads no project file.
func (e *runEnv) processSettings(cwd string) config.Resolved {
	e.process.once.Do(func() {
		e.process.r = e.resolveSettings(cwd, config.ReadProcess, e.warnf)
	})
	return e.process.r
}

// sessionSettings are the session keys' settings for a session in cwd,
// resolved each time a session starts; their diagnostics go to the log.
func (e *runEnv) sessionSettings(cwd string) config.Resolved {
	return e.resolveSettings(cwd, config.ReadSession, func(format string, args ...any) {
		e.log().Warn("settings", slog.String("message", fmt.Sprintf(format, args...)))
	})
}

// resolveSettings resolves read's keys for cwd, reporting each diagnostic;
// without a config directory it is the defaults.
func (e *runEnv) resolveSettings(cwd string, read config.Read, report func(string, ...any)) config.Resolved {
	opts, err := e.settingsOptions(cwd)
	if err != nil {
		return config.Resolved{Settings: config.Defaults()}
	}
	opts.Read = read
	r, ds := config.Resolve(opts)
	for _, d := range ds {
		report("%s", d)
	}
	return r
}

// toSession is a session's settings as acpserver takes them (0018-MADR
// D17): compaction is complete, so it never mixes defaults (0005-PLAN F5-2
// decision 8); the personality is the setting's, dry by default
// (0022-PLAN P2's deviation 1, P3), which agentOptions resolves.
func toSession(s config.Settings) acpserver.SessionSettings {
	return acpserver.SessionSettings{
		SteeringMode: s.SteeringMode, FollowUpMode: s.FollowUpMode, Thinking: s.DefaultThinkingLevel,
		CacheRetention: s.CacheRetention,
		Compaction: compaction.Settings{
			ReserveTokens: int(s.Compaction.ReserveTokens), KeepRecentTokens: int(s.Compaction.KeepRecentTokens),
			ThresholdPercent: int(s.Compaction.ThresholdPercent), ThresholdTokens: int(s.Compaction.ThresholdTokens),
		},
		MCPStartupWait: s.MCPStartupWait(), MCPStopGrace: s.MCPStopGrace(), MCPStopTimeout: s.MCPStopTimeout(),
		Retry: retryOf(s.Retry), AutoCompactionOff: !s.Compaction.Enabled, Personality: s.Personality,
		BombasticAccepted: s.BombasticAccepted,
	}
}

// retryOf is the retry settings as a session takes them (0005-PLAN F5-1).
func retryOf(r config.Retry) llm.Retry {
	ms := func(n int64) time.Duration { return time.Duration(n) * time.Millisecond }
	return llm.Retry{Off: !r.Enabled, MaxRetries: int(r.MaxRetries), BaseDelay: ms(r.BaseDelayMs), MaxDelay: ms(r.MaxDelayMs),
		OfflineWait: ms(r.OfflineWaitMs)}
}

// expandHome expands a leading ~ to the home directory, as the path keys
// allow (0018-MADR D14); without a home directory the path is unchanged.
func expandHome(p string) string {
	rest, ok := strings.CutPrefix(p, "~")
	if !ok || (rest != "" && rest[0] != '/' && rest[0] != '\\') {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	return filepath.Join(home, rest)
}

// resolveAddDirs makes each --add-dir absolute against base (the --cwd
// directory, or the working directory for ""), and checks it is a
// directory (0018-MADR D14).
func resolveAddDirs(base string, dirs []string) ([]string, error) {
	out := make([]string, 0, len(dirs))
	for _, d := range dirs {
		abs, err := resolvePath(base, d)
		if err != nil {
			return nil, usageErrorf("--add-dir %s: %v", d, err)
		}
		if fi, err := os.Stat(abs); err != nil || !fi.IsDir() {
			return nil, usageErrorf("--add-dir %s: not a directory", d)
		}
		out = append(out, abs)
	}
	return out, nil
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
