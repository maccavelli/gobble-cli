package cli

import (
	"encoding/json/v2"
	"fmt"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
)

// systemDirs resolves the directory roles; tests replace it.
var systemDirs = appdirs.System

// ConfigCmd groups configuration commands. The config surface itself is
// undecided (0004-MADR amendment of 2026-10-04); only `path` exists.
type ConfigCmd struct {
	Path ConfigPathCmd `cmd:"" help:"Print the config, data, state and cache directories."`
}

// ConfigPathCmd prints the four role directories, in the shape of
// magic-cli-remote's `paths` command: `key: value` lines, or snake_case
// JSON. Resolution notes go to stderr as warnings. It creates nothing.
type ConfigPathCmd struct {
	JSON bool `name:"json" help:"Print JSON."`
}

type pathReport struct {
	Config string `json:"config"`
	Data   string `json:"data"`
	State  string `json:"state"`
	Cache  string `json:"cache"`
}

// Run prints the directories.
func (c *ConfigPathCmd) Run(e *runEnv) error {
	d, diags, err := systemDirs()
	if err != nil {
		return failf("%v", err)
	}
	for _, dg := range diags {
		e.out.Warnf("%s", dg.Message)
	}
	r := pathReport{Config: d.Config, Data: d.Data, State: d.State, Cache: d.Cache}
	if c.JSON {
		b, err := json.Marshal(r)
		if err != nil {
			return failf("config path: %v", err)
		}
		return e.out.Result(append(b, '\n'))
	}
	return e.out.Result(fmt.Appendf(nil, "config: %s\ndata:   %s\nstate:  %s\ncache:  %s\n", r.Config, r.Data, r.State, r.Cache))
}
