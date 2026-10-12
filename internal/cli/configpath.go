package cli

import (
	"cmp"
	"encoding/json/v2"
	"fmt"
	"os"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
	"github.com/maccavelli/gobble-cli/internal/config"
)

// systemDirs resolves the directory roles; tests replace it.
var systemDirs = appdirs.System

// ConfigCmd groups the configuration commands: the settings surface of
// 0018-MADR (D12), and the directories.
type ConfigCmd struct {
	Get      ConfigGetCmd      `cmd:"" help:"Print the effective settings, or one of them."`
	Set      ConfigSetCmd      `cmd:"" help:"Set a setting in the settings file."`
	Unset    ConfigUnsetCmd    `cmd:"" help:"Remove a setting from the settings file."`
	Edit     ConfigEditCmd     `cmd:"" help:"Open the settings file in your editor."`
	Init     ConfigInitCmd     `cmd:"" help:"Create the settings file from the template."`
	Update   ConfigUpdateCmd   `cmd:"" help:"Add the settings this gobble knows to the file, as comments."`
	Validate ConfigValidateCmd `cmd:"" help:"Check the settings files."`
	Schema   ConfigSchemaCmd   `cmd:"" help:"Print the JSON Schema of settings.yaml."`
	Path     ConfigPathCmd     `cmd:"" help:"Print the directories and the settings files."`
}

// ConfigPathCmd prints the four role directories, in the shape of
// magic-cli-remote's `paths` command: `key: value` lines, or snake_case
// JSON, then the settings files (0018-MADR D12). Resolution notes go to
// stderr as warnings. It creates nothing.
type ConfigPathCmd struct {
	JSON bool `name:"json" help:"Print JSON."`
}

type pathReport struct {
	Config   string `json:"config"`
	Data     string `json:"data"`
	State    string `json:"state"`
	Cache    string `json:"cache"`
	Settings string `json:"settings"`
	Project  string `json:"project"` // "" when the working directory gives none
}

// Run prints the directories and the settings files.
func (c *ConfigPathCmd) Run(e *runEnv) error {
	d, diags, err := systemDirs()
	if err != nil {
		return failf("%v", err)
	}
	for _, dg := range diags {
		e.out.Warnf("%s", dg.Message)
	}
	r := pathReport{Config: d.Config, Data: d.Data, State: d.State, Cache: d.Cache, Settings: config.GlobalFile(d.Config)}
	if cwd, err := os.Getwd(); err == nil {
		r.Project, _ = config.ProjectFile(cwd)
	}
	if c.JSON {
		b, err := json.Marshal(r)
		if err != nil {
			return failf("config path: %v", err)
		}
		return e.out.Result(append(b, '\n'))
	}
	return e.out.Result(fmt.Appendf(nil, "config: %s\ndata:   %s\nstate:  %s\ncache:  %s\nsettings: %s\nproject:  %s\n",
		r.Config, r.Data, r.State, r.Cache, r.Settings, cmp.Or(r.Project, "none")))
}
