package cli

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/internal/config"
	"github.com/maccavelli/gobble-cli/personality"
)

// personalities are the built-ins and the user's, from
// <config>/personalities, loaded once; a directory there that is not a
// personality is left out, with a warning (0022-PLAN P3 step 6). Without a
// config directory they are the built-ins.
func (e *runEnv) personalities() *personality.Set {
	e.personalitiesOnce.Do(func() {
		dirs, _, err := systemDirs()
		if err != nil {
			return
		}
		set, errs := personality.LoadSet(filepath.Join(dirs.Config, "personalities"))
		for _, err := range errs {
			e.log().Warn("personality", slog.String("error", err.Error()))
		}
		e.personalitySet = set
	})
	return e.personalitySet
}

// personalityName is the personality a session starts with: --personality's
// for the run, else the setting's, by its canonical name. A setting no
// personality has is warned of, and gives dry (0022-MADR D6).
func (e *runEnv) personalityName(setting string) string {
	if e.personality != "" {
		return e.personality
	}
	if name := e.personalities().Canonical(setting); name != "" {
		return name
	}
	e.log().Warn("settings", slog.String("message",
		fmt.Sprintf("personality: no personality is named %q; sessions use %s", setting, personality.Default)))
	return personality.Default
}

// voiceNamed is the voice of the CLI's own lines for the named
// personality; nil, boring, for a name no personality has.
func (e *runEnv) voiceNamed(name string) *personality.Voice {
	v, err := e.personalities().New(personality.Options{Name: name})
	if err != nil {
		return nil
	}
	return v
}

// runVoice is the voice a session in cwd starts with, which the CLI's own
// lines speak until the agent's _gobble/personality says otherwise. The
// settings' diagnostics are the session's to report, when it starts, so
// they are not reported twice.
func (e *runEnv) runVoice(cwd string) *personality.Voice {
	quiet := func(string, ...any) {}
	return e.voiceNamed(e.personalityName(e.resolveSettings(cwd, config.ReadSession, quiet).Settings.Personality))
}

// choosePersonality takes --personality: a personality's name, or one of
// boring's, which the run's sessions start with.
func (e *runEnv) choosePersonality(flag string) error {
	if flag == "" {
		return nil
	}
	set := e.personalities()
	name := set.Canonical(flag)
	if name == "" {
		return usageErrorf("--personality %s: no personality has that name; the personalities are %s", flag, strings.Join(set.Names(), ", "))
	}
	e.personality = name
	return nil
}

// printPermission answers print mode's permission requests as the client
// answers with no one to ask: the first reject_once option, else
// reject_always, else cancelled. The level-5 question is answered no, and
// the session falls back to dry, which stderr says (0022-MADR D10).
func (e *runEnv) printPermission(_ context.Context, r acpclient.PermissionRequest) acpclient.PermissionAnswer {
	if r.Tool.ID == acpserver.InsultsCallID {
		e.out.Warnf("a level-5 personality asks first, and print mode cannot ask; this run uses %s. "+
			"Choose it once in a session to agree.", personality.Default)
	}
	for _, kind := range []string{"reject_once", "reject_always"} {
		for _, o := range r.Options {
			if o.Kind == kind {
				return acpclient.PermissionAnswer{OptionID: o.ID}
			}
		}
	}
	return acpclient.PermissionAnswer{}
}

// personalityCandidates are --personality's values: the personalities,
// each described by its level and summary.
func personalityCandidates(context.Context) iter.Seq[complete.Candidate] {
	return func(yield func(complete.Candidate) bool) {
		var set *personality.Set
		if dirs, _, err := systemDirs(); err == nil {
			set, _ = personality.LoadSet(filepath.Join(dirs.Config, "personalities")) //nolint:errcheck // completion offers what loads
		}
		for _, info := range set.Infos() {
			if !yield(complete.Candidate{Value: info.Name, Description: fmt.Sprintf("%d: %s", info.Level, info.Summary)}) {
				return
			}
		}
	}
}
