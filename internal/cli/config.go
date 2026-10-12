package cli

import (
	"cmp"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/internal/config"
)

// openEditor is editor.Open; tests replace it.
var openEditor = editor.Open

// ConfigGetCmd prints the effective settings (0018-MADR D12).
type ConfigGetCmd struct {
	Path   string `arg:"" optional:"" complete:"setting" help:"A setting, or a group such as mcp; all of them when left out."`
	Origin bool   `help:"Show where each value comes from, and the layers not applied."`
	JSON   bool   `name:"json" help:"Print JSON."`
}

// ConfigSetCmd sets one setting in a settings file.
type ConfigSetCmd struct {
	Path    string `arg:"" complete:"setting" help:"The setting."`
	Value   string `arg:"" complete:"none" help:"Its value, as YAML."`
	Project bool   `help:"Write the project's .gobble/settings.yaml."`
}

// ConfigUnsetCmd removes one setting from a settings file.
type ConfigUnsetCmd struct {
	Path    string `arg:"" complete:"setting" help:"The setting."`
	Project bool   `help:"Edit the project's .gobble/settings.yaml."`
}

// ConfigEditCmd opens a settings file in the editor.
type ConfigEditCmd struct {
	Project bool `help:"Edit the project's .gobble/settings.yaml."`
}

// ConfigInitCmd creates a settings file from the template.
type ConfigInitCmd struct {
	Project bool `help:"Create the project's .gobble/settings.yaml."`
}

// ConfigUpdateCmd adds the settings a newer gobble knows, as comments.
type ConfigUpdateCmd struct {
	Project bool `help:"Update the project's .gobble/settings.yaml."`
	DryRun  bool `help:"Print what would be added, and write nothing."`
}

// ConfigValidateCmd checks the settings files.
type ConfigValidateCmd struct {
	Strict bool `help:"Fail on an unknown key too."`
}

// ConfigSchemaCmd prints the JSON Schema of settings.yaml.
type ConfigSchemaCmd struct{}

// settingsFile is the file a config command acts on: the global one, or
// with project, the working directory's project file.
func settingsFile(project bool) (string, config.Layer, error) {
	if !project {
		d, _, err := systemDirs()
		if err != nil {
			return "", 0, failf("%v", err)
		}
		return config.GlobalFile(d.Config), config.LayerGlobal, nil
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", 0, failf("working directory: %v", err)
	}
	file, ok := config.ProjectFile(cwd)
	if !ok {
		return "", 0, failf("no project directory")
	}
	return file, config.LayerProject, nil
}

// prepare makes file exist before a write: the global file from the
// template (0018-MADR D9); a project file only when create, for init and
// edit (0018-PLAN P4 deviation 10).
func prepare(file string, layer config.Layer, create bool) error {
	if _, err := os.Stat(file); !errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if layer == config.LayerProject && !create {
		return failf("%s does not exist; gobble config init --project creates it", file)
	}
	if err := config.Init(file); err != nil {
		return failf("%v", err)
	}
	return nil
}

// resolveHere resolves the settings for the working directory.
func (e *runEnv) resolveHere() (config.Resolved, []config.Diagnostic, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return config.Resolved{}, nil, failf("working directory: %v", err)
	}
	opts, err := e.settingsOptions(cwd)
	if err != nil {
		return config.Resolved{}, nil, err
	}
	r, ds := config.Resolve(opts)
	return r, ds, nil
}

// keysUnder are the keys at path, or under it as a group; every key for "".
func keysUnder(path string) []config.Key {
	var out []config.Key
	for _, k := range config.Keys {
		if path == "" || k.Path == path || strings.HasPrefix(k.Path, path+".") {
			out = append(out, k)
		}
	}
	return out
}

// Run prints the value, the group or the document.
func (c *ConfigGetCmd) Run(e *runEnv) error {
	keys := keysUnder(c.Path)
	if len(keys) == 0 {
		return usageErrorf("unknown setting %q", c.Path)
	}
	r, ds, err := e.resolveHere()
	if err != nil {
		return err
	}
	for _, d := range ds {
		e.out.Warnf("%s", d)
	}
	if !c.JSON {
		return e.out.Result(config.Render(r, c.Path, c.Origin))
	}
	b, err := getJSON(r, c.Path, keys, c.Origin)
	if err != nil {
		return failf("config get: %v", err)
	}
	return e.out.Result(append(b, '\n'))
}

type originJSON struct {
	Layer string `json:"layer"`
	File  string `json:"file,omitempty"`
	Line  int    `json:"line,omitempty"`
}

type disabledJSON struct {
	File   string   `json:"file"`
	Reason string   `json:"reason"`
	Keys   []string `json:"keys"`
}

// getJSON is get --json's form (0018-PLAN P4 deviation 11): the settings
// object, or a key's value; with origin, wrapped beside its origins and the
// layers not applied.
func getJSON(r config.Resolved, path string, keys []config.Key, origin bool) ([]byte, error) {
	var value any
	if len(keys) == 1 && keys[0].Path == path {
		value, _ = r.Settings.Value(path)
	} else {
		doc := map[string]any{}
		for _, k := range keys {
			v, _ := r.Settings.Value(k.Path)
			m := doc
			segs := strings.Split(k.Path, ".")
			for _, s := range segs[:len(segs)-1] {
				next, ok := m[s].(map[string]any)
				if !ok {
					next = map[string]any{}
					m[s] = next
				}
				m = next
			}
			m[segs[len(segs)-1]] = v
		}
		value = doc
		if path != "" {
			for s := range strings.SplitSeq(path, ".") {
				if m, ok := value.(map[string]any); ok {
					value = m[s]
				}
			}
		}
	}
	opts := []json.Options{json.Deterministic(true), jsontext.WithIndent("  ")}
	if !origin {
		return json.Marshal(value, opts...)
	}
	origins := map[string]originJSON{}
	for _, k := range keys {
		s := r.Origins[k.Path]
		origins[k.Path] = originJSON{Layer: s.Layer.String(), File: s.File, Line: s.Line}
	}
	disabled := []disabledJSON{}
	for _, d := range r.Disabled {
		disabled = append(disabled, disabledJSON{File: d.File, Reason: d.Reason, Keys: d.Keys})
	}
	wrapped := map[string]any{"origins": origins, "disabled": disabled}
	if len(keys) == 1 && keys[0].Path == path {
		wrapped["value"] = value
	} else {
		wrapped["settings"] = value
	}
	return json.Marshal(wrapped, opts...)
}

// Run checks the value, then writes it (D12: a bad value exits 2, a write
// failure 1).
func (c *ConfigSetCmd) Run(e *runEnv) error {
	file, layer, err := settingsFile(c.Project)
	if err != nil {
		return err
	}
	if err := config.Check(c.Path, c.Value, layer); err != nil {
		return usageErrorf("%v", err)
	}
	if err := prepare(file, layer, false); err != nil {
		return err
	}
	if err := config.Set(file, c.Path, c.Value); err != nil {
		return failf("%v", err)
	}
	return nil
}

// Run removes the setting: a template key goes back to its commented-out
// form, any other is deleted.
func (c *ConfigUnsetCmd) Run(e *runEnv) error {
	file, layer, err := settingsFile(c.Project)
	if err != nil {
		return err
	}
	if err := prepare(file, layer, false); err != nil {
		return err
	}
	if err := config.Unset(file, c.Path); err != nil {
		return failf("%v", err)
	}
	return nil
}

// Run opens the file in the editor chain, then prints its diagnostics; an
// error among them exits 1 (0018-PLAN P4 deviation 14).
func (c *ConfigEditCmd) Run(e *runEnv) error {
	file, layer, err := settingsFile(c.Project)
	if err != nil {
		return err
	}
	if err := prepare(file, layer, true); err != nil {
		return err
	}
	if err := openEditor(e.ctx, os.Getenv, file, e.runAttached); err != nil {
		return failf("%v", err)
	}
	_, ds, err := e.resolveHere()
	if err != nil {
		return err
	}
	ds = slices.DeleteFunc(ds, func(d config.Diagnostic) bool { return d.File != file })
	return e.printDiagnostics(ds, false)
}

// runAttached runs an editor on the process's terminal.
func (e *runEnv) runAttached(c *exec.Cmd) error {
	if e.stdin != nil {
		c.Stdin = e.stdin
	}
	c.Stdout, c.Stderr = e.out.out, e.out.err
	return c.Run()
}

// Run creates the file; an existing one exits 1 and is left alone.
func (c *ConfigInitCmd) Run(e *runEnv) error {
	file, _, err := settingsFile(c.Project)
	if err != nil {
		return err
	}
	if err := config.Init(file); err != nil {
		return failf("%v", err)
	}
	return e.out.Resultf("created %s\n", file)
}

// Run adds the template blocks the file lacks, printing them.
func (c *ConfigUpdateCmd) Run(e *runEnv) error {
	file, layer, err := settingsFile(c.Project)
	if err != nil {
		return err
	}
	if err := prepare(file, layer, false); err != nil {
		return err
	}
	_, preview, err := config.Update(file, c.DryRun)
	if err != nil {
		return failf("%v", err)
	}
	return e.out.Result(preview)
}

// Run prints every diagnostic of the global and project files.
func (c *ConfigValidateCmd) Run(e *runEnv) error {
	_, ds, err := e.resolveHere()
	if err != nil {
		return err
	}
	return e.printDiagnostics(ds, c.Strict)
}

// printDiagnostics prints ds one per line, by file and line, and exits 1
// on an error, or with strict on an unknown key (0018-MADR D7).
func (e *runEnv) printDiagnostics(ds []config.Diagnostic, strict bool) error {
	slices.SortStableFunc(ds, func(a, b config.Diagnostic) int {
		return cmp.Or(strings.Compare(a.File, b.File), a.Line-b.Line)
	})
	fail := false
	var b strings.Builder
	for _, d := range ds {
		b.WriteString(d.String() + "\n")
		fail = fail || d.Severity == config.SeverityError || (strict && d.Unknown())
	}
	if err := e.out.Result([]byte(b.String())); err != nil {
		return err
	}
	if fail {
		return &exitError{code: ExitFailure}
	}
	return nil
}

// Run prints the schema.
func (*ConfigSchemaCmd) Run(e *runEnv) error {
	b, err := config.Schema()
	if err != nil {
		return failf("%v", err)
	}
	return e.out.Result(b)
}
