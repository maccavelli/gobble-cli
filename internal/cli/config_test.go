package cli

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/config"
)

// cfgEnv is a fresh GOBBLE_HOME and a fresh working directory, which is
// also the project root (outside a repository, the working directory is).
type cfgEnv struct {
	t         *testing.T
	home, dir string
}

func newCfgEnv(t *testing.T) *cfgEnv {
	t.Helper()
	c := &cfgEnv{t: t, home: t.TempDir(), dir: t.TempDir()}
	t.Setenv("GOBBLE_HOME", c.home)
	t.Chdir(c.dir)
	return c
}

func (c *cfgEnv) run(args ...string) result {
	c.t.Helper()
	var out, errw bytes.Buffer
	code := Main(c.t.Context(), args, nil, &out, &errw)
	return result{code: code, stdout: out.String(), stderr: errw.String()}
}

func (c *cfgEnv) global() string  { return filepath.Join(c.home, "settings.yaml") }
func (c *cfgEnv) project() string { return filepath.Join(c.dir, ".gobble", "settings.yaml") }

func (c *cfgEnv) write(file, text string) {
	c.t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		c.t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(text), 0o600); err != nil {
		c.t.Fatal(err)
	}
}

func (c *cfgEnv) read(file string) string {
	c.t.Helper()
	b, err := os.ReadFile(file)
	if err != nil {
		c.t.Fatal(err)
	}
	return string(b)
}

func (c *cfgEnv) want(r result, code int, stdout, stderr string) {
	c.t.Helper()
	if r.code != code || (stdout != "" && !strings.Contains(r.stdout, stdout)) || (stderr != "" && !strings.Contains(r.stderr, stderr)) {
		c.t.Fatalf("exit %d, stdout %q, stderr %q; want %d, %q, %q", r.code, r.stdout, r.stderr, code, stdout, stderr)
	}
}

func missing(t *testing.T, file string) {
	t.Helper()
	if _, err := os.Stat(file); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("%s exists (%v)", file, err)
	}
}

// get prints the document, a group, or one value, from the files.
func TestConfigGet(t *testing.T) {
	c := newCfgEnv(t)
	r := c.run("config", "get")
	c.want(r, ExitOK, "defaultThinkingLevel: off\nsteeringMode: one-at-a-time\n", "")
	c.write(c.global(), "steeringMode: all\nmcp:\n  stopGraceMs: 100\n")
	c.want(c.run("config", "get", "steeringMode"), ExitOK, "all\n", "")
	if r := c.run("config", "get", "mcp"); r.stdout != "mcp:\n  startupWaitMs: 10000\n  stopGraceMs: 100\n  stopTimeoutMs: 2000\n" {
		t.Fatalf("get mcp: %q", r.stdout)
	}
	c.want(c.run("config", "get", "theme"), ExitUsage, "", `unknown setting "theme"`)
	c.write(c.global(), "steeringMode: some\n")
	c.want(c.run("config", "get", "steeringMode"), ExitOK, "one-at-a-time\n", "steeringMode: must be one of one-at-a-time, all")
}

// --origin names each value's file, line and layer, --set included, and
// the project layer not applied.
func TestConfigGetOrigin(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.global(), "steeringMode: all\n")
	c.write(c.project(), "externalEditor: vim\n")
	r := c.run("--set", "followUpMode=all", "config", "get", "--origin")
	for _, line := range []string{
		"steeringMode: all # from " + c.global() + ":1 (global)",
		"followUpMode: all # from --set",
		"defaultThinkingLevel: off # default",
		"# not applied: " + c.project() + ": project settings apply once the project is trusted (0005-PLAN F6)",
		"#   it would set: externalEditor",
	} {
		if !strings.Contains(r.stdout, line+"\n") {
			t.Errorf("no line %q in:\n%s", line, r.stdout)
		}
	}
}

// --json is the settings object, or one value; with --origin it is wrapped
// beside the origins and the layers not applied (0018-PLAN P4 deviation 11).
func TestConfigGetJSON(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.global(), "steeringMode: all\n")
	var doc struct {
		SteeringMode string `json:"steeringMode"`
		SessionDir   string `json:"sessionDir"`
		MCP          struct {
			StopGraceMs int64 `json:"stopGraceMs"`
		} `json:"mcp"`
	}
	r := c.run("config", "get", "--json")
	if err := json.Unmarshal([]byte(r.stdout), &doc, json.RejectUnknownMembers(false)); err != nil || doc.SteeringMode != "all" || doc.MCP.StopGraceMs != 500 {
		t.Fatalf("%v: %+v from %s", err, doc, r.stdout)
	}
	if !strings.Contains(r.stdout, `"sessionDir": ""`) {
		t.Fatalf("an empty string key is left out: %s", r.stdout)
	}
	c.want(c.run("config", "get", "steeringMode", "--json"), ExitOK, "\"all\"\n", "")
	var wrapped struct {
		Value    any                       `json:"value"`
		Settings map[string]any            `json:"settings"`
		Origins  map[string]map[string]any `json:"origins"`
		Disabled []any                     `json:"disabled"`
	}
	r = c.run("config", "get", "steeringMode", "--json", "--origin")
	if err := json.Unmarshal([]byte(r.stdout), &wrapped); err != nil || wrapped.Value != "all" || wrapped.Settings != nil ||
		wrapped.Origins["steeringMode"]["layer"] != "global" || wrapped.Origins["steeringMode"]["line"] != 1.0 || wrapped.Disabled == nil {
		t.Fatalf("%v: %+v from %s", err, wrapped, r.stdout)
	}
	wrapped = struct {
		Value    any                       `json:"value"`
		Settings map[string]any            `json:"settings"`
		Origins  map[string]map[string]any `json:"origins"`
		Disabled []any                     `json:"disabled"`
	}{}
	r = c.run("config", "get", "--json", "--origin")
	if err := json.Unmarshal([]byte(r.stdout), &wrapped); err != nil || wrapped.Settings["steeringMode"] != "all" || len(wrapped.Origins) != len(config.Keys) {
		t.Fatalf("%v: %+v from %s", err, wrapped, r.stdout)
	}
}

// set creates the global file from the template and uncomments the key;
// unset puts the template's line back.
func TestConfigSetUnset(t *testing.T) {
	c := newCfgEnv(t)
	c.want(c.run("config", "set", "steeringMode", "all"), ExitOK, "", "")
	if got := c.read(c.global()); !strings.Contains(got, "\nsteeringMode: all\n") || !strings.HasPrefix(got, "# gobble settings\n") {
		t.Fatalf("after set:\n%s", got)
	}
	c.want(c.run("config", "get", "steeringMode"), ExitOK, "all\n", "")
	c.want(c.run("config", "unset", "steeringMode"), ExitOK, "", "")
	if got := c.read(c.global()); got != string(config.Template()) {
		t.Fatalf("after unset, not the template:\n%s", got)
	}
}

// --project writes the project file, which only init --project or edit
// --project creates (0018-PLAN P4 deviation 10); the global file is left
// alone.
func TestConfigSetProject(t *testing.T) {
	c := newCfgEnv(t)
	c.want(c.run("config", "set", "--project", "steeringMode", "all"), ExitFailure, "", "gobble config init --project creates it")
	missing(t, c.project())
	c.want(c.run("config", "init", "--project"), ExitOK, "created "+c.project(), "")
	c.want(c.run("config", "set", "--project", "steeringMode", "all"), ExitOK, "", "")
	if !strings.Contains(c.read(c.project()), "\nsteeringMode: all\n") {
		t.Fatal("the project file was not written")
	}
	missing(t, c.global())
	c.want(c.run("config", "unset", "--project", "steeringMode"), ExitOK, "", "")
	if c.read(c.project()) != string(config.Template()) {
		t.Fatal("unset --project did not restore the template")
	}
}

// A global-only key is refused with --project, exit 2, and nothing is
// written.
func TestConfigSetGlobalOnlyProjectRefused(t *testing.T) {
	c := newCfgEnv(t)
	c.want(c.run("config", "init", "--project"), ExitOK, "", "")
	for _, k := range []string{"shellPath", "shellCommandPrefix", "sessionDir"} {
		c.want(c.run("config", "set", "--project", k, "~/x"), ExitUsage, "", k+": only the global settings file can set this")
	}
	if c.read(c.project()) != string(config.Template()) {
		t.Fatal("a refused set wrote the project file")
	}
	c.want(c.run("config", "set", "shellPath", "~/bin/bash"), ExitOK, "", "")
}

// An unknown path exits 2 before any file is created.
func TestConfigSetUnknownExit2(t *testing.T) {
	c := newCfgEnv(t)
	c.want(c.run("config", "set", "theme", "dark"), ExitUsage, "", `unknown setting "theme"`)
	c.want(c.run("config", "set", "apiKey", "x"), ExitUsage, "", "apiKey: credentials go in auth.json, not settings.yaml")
	missing(t, c.global())
}

// A bad value exits 2 in D7's words, and nothing is written.
func TestConfigSetBadValueExit2(t *testing.T) {
	c := newCfgEnv(t)
	c.want(c.run("config", "set", "mcp.stopGraceMs", "soon"), ExitUsage, "", "mcp.stopGraceMs: want an integer, have a string")
	c.want(c.run("config", "set", "steeringMode", "some"), ExitUsage, "", "steeringMode: must be one of one-at-a-time, all")
	missing(t, c.global())
}

// stubEditor makes openEditor write text into the file it opens, recording
// what the file held when it opened.
func stubEditor(t *testing.T, text string, opened *string) {
	t.Helper()
	old := openEditor
	openEditor = func(_ context.Context, _ func(string) string, _, file string, _ func(*exec.Cmd) error) error {
		b, err := os.ReadFile(file) //nolint:gosec // G304: the test's own file
		if err != nil {
			return err
		}
		*opened = string(b)
		return os.WriteFile(file, []byte(text), 0o600)
	}
	t.Cleanup(func() { openEditor = old })
}

// edit opens the file, created from the template when missing, and prints
// its diagnostics after; an error exits 1 (0018-PLAN P4 deviation 14).
func TestConfigEdit(t *testing.T) {
	c := newCfgEnv(t)
	var opened string
	stubEditor(t, "steeringMode: all\n", &opened)
	c.want(c.run("config", "edit"), ExitOK, "", "")
	if opened != string(config.Template()) || c.read(c.global()) != "steeringMode: all\n" {
		t.Fatalf("opened %q, file %q", opened, c.read(c.global()))
	}
	stubEditor(t, "steeringMode: some\ntheme: dark\n", &opened)
	r := c.run("config", "edit")
	c.want(r, ExitFailure, c.global()+":1: steeringMode: must be one of one-at-a-time, all\n"+c.global()+":2: theme: not used by this gobble\n", "")
	stubEditor(t, "theme: dark\n", &opened)
	c.want(c.run("config", "edit", "--project"), ExitOK, c.project()+":1: theme: not used by this gobble\n", "")
	if opened != string(config.Template()) {
		t.Fatal("edit --project did not create the project file from the template")
	}
	old := openEditor
	openEditor = func(context.Context, func(string) string, string, string, func(*exec.Cmd) error) error {
		return errors.New("editor vi: exit status 1")
	}
	t.Cleanup(func() { openEditor = old })
	c.want(c.run("config", "edit"), ExitFailure, "", "editor vi: exit status 1")
}

// init writes the template, creating the directory.
func TestConfigInit(t *testing.T) {
	c := newCfgEnv(t)
	c.want(c.run("config", "init"), ExitOK, "created "+c.global(), "")
	if c.read(c.global()) != string(config.Template()) {
		t.Fatal("init did not write the template")
	}
}

// init over an existing file exits 1 and leaves it alone.
func TestConfigInitExistsExit1(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.global(), "steeringMode: all\n")
	c.want(c.run("config", "init"), ExitFailure, "", "already exists")
	if c.read(c.global()) != "steeringMode: all\n" {
		t.Fatal("init changed an existing file")
	}
}

// update --dry-run prints the lines and writes nothing; update writes them.
func TestConfigUpdateDryRun(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.global(), "steeringMode: all\n")
	r := c.run("config", "update", "--dry-run")
	c.want(r, ExitOK, "+# defaultThinkingLevel: off\n", "")
	if c.read(c.global()) != "steeringMode: all\n" {
		t.Fatal("--dry-run wrote the file")
	}
	c.want(c.run("config", "update"), ExitOK, "+# defaultThinkingLevel: off\n", "")
	if got := c.read(c.global()); !strings.HasPrefix(got, "steeringMode: all\n") || !strings.Contains(got, "# defaultThinkingLevel: off\n") {
		t.Fatalf("after update:\n%s", got)
	}
	if r := c.run("config", "update"); r.code != ExitOK || r.stdout != "" {
		t.Fatalf("a second update: %+v", r)
	}
	c.want(c.run("config", "update", "--project"), ExitFailure, "", "gobble config init --project creates it")
}

// validate prints every diagnostic, by file and line, and exits 1 on an
// error, or with --strict on an unknown key.
func TestConfigValidateExits(t *testing.T) {
	c := newCfgEnv(t)
	if r := c.run("config", "validate"); r.code != ExitOK || r.stdout != "" {
		t.Fatalf("no files: %+v", r)
	}
	c.write(c.global(), "theme: dark\n")
	c.want(c.run("config", "validate"), ExitOK, c.global()+":1: theme: not used by this gobble\n", "")
	c.want(c.run("config", "validate", "--strict"), ExitFailure, "theme: not used by this gobble", "")
	c.write(c.global(), "mcp:\n  stopGraceMs: -1\nsteeringMode: some\n")
	r := c.run("config", "validate")
	c.want(r, ExitFailure, c.global()+":2: mcp.stopGraceMs: must be at least 1\n"+c.global()+":3: steeringMode: must be one of one-at-a-time, all\n", "")
	c.write(c.project(), "shellPath: /x\n")
	c.want(c.run("config", "validate"), ExitFailure, c.project()+":1: shellPath: only the global settings file can set this", "")
}

// schema prints the committed schema.
func TestConfigSchema(t *testing.T) {
	c := newCfgEnv(t)
	want, err := config.Schema()
	if err != nil {
		t.Fatal(err)
	}
	if r := c.run("config", "schema"); r.code != ExitOK || r.stdout != string(want) {
		t.Fatalf("exit %d, %d bytes", r.code, len(r.stdout))
	}
}

// A bad --set is a usage error before the command runs; a good one is the
// session layer.
func TestSetFlagUsageError(t *testing.T) {
	c := newCfgEnv(t)
	c.want(c.run("--set", "theme=dark", "config", "get"), ExitUsage, "", `unknown setting "theme"`)
	c.want(c.run("--set", "steeringMode", "config", "get"), ExitUsage, "", "--set steeringMode: want PATH=VALUE")
	c.want(c.run("--set", "mcp.stopGraceMs=x", "-p", "hi"), ExitUsage, "", "--set mcp.stopGraceMs: want an integer, have a string")
	c.want(c.run("--set", "steeringMode=all", "--set", "shellCommandPrefix=a=b", "config", "get", "shellCommandPrefix"), ExitOK, "a=b\n", "")
	missing(t, c.global())
}

// The first run of chat, acp and mcp creates the global file from the
// template.
func TestFirstRunCreatesTemplate(t *testing.T) {
	withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("ok")}}}})
	for _, args := range [][]string{{"-p", "hi"}, {"acp"}, {"mcp", "list"}} {
		c := newCfgEnv(t)
		if r := c.run(args...); r.code != ExitOK {
			t.Fatalf("%v: exit %d, stderr %q", args, r.code, r.stderr)
		}
		if c.read(c.global()) != string(config.Template()) {
			t.Fatalf("%v did not create the template", args)
		}
	}
}

// The read-only commands create nothing in a fresh home.
func TestReadOnlyCommandsCreateNothing(t *testing.T) {
	for _, args := range [][]string{
		{"version"}, {"config", "path"}, {"config", "get"}, {"config", "get", "--origin", "--json"},
		{"config", "validate"}, {"config", "schema"}, {"completion", "bash"},
	} {
		c := newCfgEnv(t)
		if r := c.run(args...); r.code != ExitOK {
			t.Fatalf("%v: exit %d, stderr %q", args, r.code, r.stderr)
		}
		for _, dir := range []string{c.home, c.dir} {
			if des, err := os.ReadDir(dir); err != nil || len(des) != 0 {
				t.Fatalf("%v created %v in %s (%v)", args, des, dir, err)
			}
		}
	}
}
