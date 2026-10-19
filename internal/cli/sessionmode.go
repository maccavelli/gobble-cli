package cli

import (
	"context"
	"iter"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
)

// modeIDs are the session modes --mode takes (0005-PLAN F6-4 decision 1).
var modeIDs = []string{"default", "accept-edits", "plan", "bypass"}

// modeCandidates are --mode's values, in their order.
func modeCandidates(context.Context) iter.Seq[complete.Candidate] {
	return func(yield func(complete.Candidate) bool) {
		for _, m := range modeIDs {
			if !yield(complete.Candidate{Value: m}) {
				return
			}
		}
	}
}

// chooseMode takes --mode: the mode the run's sessions start in. bypass
// needs permissions.allowBypass in the global settings file, which a
// project's file cannot set (decisions 2 and 7).
func (e *runEnv) chooseMode(flag string) error {
	switch {
	case flag == "":
		return nil
	case !slices.Contains(modeIDs, flag):
		return usageErrorf("--mode %s: the modes are %s", flag, strings.Join(modeIDs, ", "))
	case flag == "bypass" && !e.sessionSettings("").Settings.Permissions.AllowBypass:
		return usageErrorf("--mode bypass needs permissions.allowBypass: true in the global settings file; " +
			"gobble config set permissions.allowBypass true sets it")
	}
	e.mode = flag
	return nil
}
