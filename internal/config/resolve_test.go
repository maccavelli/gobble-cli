package config

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/compaction"
)

// layout is a config directory and a project with a .git, holding the
// given global and project files ("" writes none).
type layout struct{ config, project string }

func newLayout(t *testing.T, global, project string) layout {
	t.Helper()
	root := t.TempDir()
	l := layout{config: filepath.Join(root, "config"), project: filepath.Join(root, "proj")}
	write := func(path, text string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(l.project, ".git", "HEAD"), "ref: refs/heads/main\n")
	if global != "" {
		write(filepath.Join(l.config, "settings.yaml"), global)
	}
	if project != "" {
		write(filepath.Join(l.project, ".gobble", "settings.yaml"), project)
	}
	return l
}

func trustAll(string) bool { return true }

// With no file, the settings are the defaults, and each default is the
// code's today (0018-MADR D14, 0018-PLAN C1). acpserver's start-up wait,
// mcpclient's stop timings and builtin's timeout cap are checked as
// literals here, and each package's own test pins its constant to the same
// literal (0018-PLAN P2 deviation 5): TestDefaultStartupWait,
// TestStopDefaults and TestDefaultMaxTimeout.
func TestDefaultsEqualCode(t *testing.T) {
	r, ds := Resolve(Options{})
	if len(ds) != 0 {
		t.Fatalf("diagnostics with no file: %v", ds)
	}
	s := r.Settings
	if s != Defaults() {
		t.Fatalf("Resolve with no file = %+v, want Defaults()", s)
	}
	checks := []struct {
		name      string
		got, want any
	}{
		{"defaultThinkingLevel", s.DefaultThinkingLevel, "off"},
		{"steeringMode", s.SteeringMode, "one-at-a-time"},
		{"followUpMode", s.FollowUpMode, "one-at-a-time"},
		{"keepRecentTokens", s.Compaction.KeepRecentTokens, int64(compaction.DefaultSettings.KeepRecentTokens)},
		{"useClientTerminal", s.Tools.Bash.UseClientTerminal, false},
		{"maxTimeout", s.MaxTimeout().String(), "10m0s"},
		{"startupWait", s.MCPStartupWait().String(), "10s"},
		{"stopGrace", s.MCPStopGrace().String(), "500ms"},
		{"stopTimeout", s.MCPStopTimeout().String(), "2s"},
		{"sessionDir", s.SessionDir, ""},
		{"shellPath", s.ShellPath, ""},
		{"shellCommandPrefix", s.ShellCommandPrefix, ""},
		{"externalEditor", s.ExternalEditor, ""},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}
	for _, k := range Keys {
		if r.Origins[k.Path].Layer != LayerDefault {
			t.Errorf("%s origin %v, want default", k.Path, r.Origins[k.Path])
		}
	}
}

// Every key has a Settings field of its kind's Go type, and its default.
func TestKeysMatchSettings(t *testing.T) {
	var s Settings
	for _, k := range Keys {
		if err := s.set(k.Path, defaultValue(k)); err != nil {
			t.Errorf("%s: %v", k.Path, err)
		}
		if v, ok := s.Value(k.Path); !ok || v != defaultValue(k) {
			t.Errorf("%s: Value = %v %v, want %v", k.Path, v, ok, defaultValue(k))
		}
	}
	if len(Keys) != 13 {
		t.Errorf("%d keys, want 0018-MADR D14's 13", len(Keys))
	}
}

// Layers stack: global over defaults, a trusted project over global,
// --set over both; each value's origin names its layer, file and line.
func TestLayersAndPrecedence(t *testing.T) {
	l := newLayout(t, "steeringMode: all\nfollowUpMode: all\nmcp:\n  stopGraceMs: 100\n",
		"followUpMode: one-at-a-time\nmcp:\n  stopGraceMs: 200\n")
	set, err := ParseOverride("mcp.stopGraceMs=300")
	if err != nil {
		t.Fatal(err)
	}
	r, ds := Resolve(Options{ConfigDir: l.config, Cwd: l.project, Trusted: trustAll, Overrides: []Override{set}})
	if len(ds) != 0 {
		t.Fatalf("diagnostics: %v", ds)
	}
	s := r.Settings
	if s.SteeringMode != "all" || s.FollowUpMode != "one-at-a-time" || s.MCP.StopGraceMs != 300 {
		t.Fatalf("settings %+v", s)
	}
	o := r.Origins
	if o["steeringMode"].Layer != LayerGlobal || o["steeringMode"].Line != 1 {
		t.Errorf("steeringMode origin %+v", o["steeringMode"])
	}
	if o["followUpMode"].Layer != LayerProject || !strings.HasSuffix(o["followUpMode"].File, filepath.Join(".gobble", "settings.yaml")) {
		t.Errorf("followUpMode origin %+v", o["followUpMode"])
	}
	if o["mcp.stopGraceMs"].Layer != LayerSession {
		t.Errorf("mcp.stopGraceMs origin %+v", o["mcp.stopGraceMs"])
	}
}

// A null clears a key back to the layer below; a null group clears every
// key under it (0018-MADR D4).
func TestNullClears(t *testing.T) {
	l := newLayout(t, "steeringMode: all\nmcp:\n  stopGraceMs: 100\n", "steeringMode: ~\nmcp: ~\n")
	r, ds := Resolve(Options{ConfigDir: l.config, Cwd: l.project, Trusted: trustAll})
	if len(ds) != 0 {
		t.Fatalf("diagnostics: %v", ds)
	}
	if r.Settings.SteeringMode != "all" || r.Settings.MCP.StopGraceMs != 100 {
		t.Fatalf("settings %+v: want the global values kept", r.Settings)
	}
}

// Until a project is trusted, its file is read and reported, but not
// applied, with D7's reason and the keys it would have set.
func TestProjectDisabledUntilTrusted(t *testing.T) {
	l := newLayout(t, "", "steeringMode: all\nmcp:\n  stopGraceMs: 9\n")
	r, ds := Resolve(Options{ConfigDir: l.config, Cwd: filepath.Join(l.project)})
	if r.Settings.SteeringMode != "one-at-a-time" || r.Settings.MCP.StopGraceMs != 500 {
		t.Fatalf("an untrusted project applied: %+v", r.Settings)
	}
	if len(r.Disabled) != 1 || r.Disabled[0].Reason != "project settings apply once the project is trusted (0005-PLAN F6)" ||
		!slices.Equal(r.Disabled[0].Keys, []string{"steeringMode", "mcp.stopGraceMs"}) {
		t.Fatalf("disabled %+v", r.Disabled)
	}
	if len(ds) != 1 || ds[0].Severity != SeverityWarning || !strings.HasSuffix(ds[0].String(), ": "+textUntrusted) {
		t.Fatalf("diagnostics %v", ds)
	}
}

// A trusted project's file applies.
func TestProjectTrustedApplies(t *testing.T) {
	l := newLayout(t, "", "steeringMode: all\n")
	var asked string
	trust := func(root string) bool { asked = root; return true }
	sub := filepath.Join(l.project, "a", "b")
	if err := os.MkdirAll(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	r, _ := Resolve(Options{Cwd: sub, Trusted: trust})
	if r.Settings.SteeringMode != "all" {
		t.Fatalf("a trusted project did not apply: %+v", r.Settings)
	}
	if asked != l.project {
		t.Fatalf("trust asked of %q, want the git root %q", asked, l.project)
	}
}

// A global-only key from a project is ignored, even when it is trusted,
// with D7's warning; from the global file it applies.
func TestGlobalOnlyIgnoredFromProject(t *testing.T) {
	l := newLayout(t, "shellCommandPrefix: set -e\n", "shellCommandPrefix: rm -rf ~\nshellPath: /evil\n")
	r, ds := Resolve(Options{ConfigDir: l.config, Cwd: l.project, Trusted: trustAll})
	if r.Settings.ShellCommandPrefix != "set -e" || r.Settings.ShellPath != "" {
		t.Fatalf("settings %+v", r.Settings)
	}
	var got []string
	for _, d := range ds {
		got = append(got, d.Path+": "+d.Text)
	}
	want := []string{"shellCommandPrefix: " + textGlobalOnly, "shellPath: " + textGlobalOnly}
	if !slices.Equal(got, want) {
		t.Fatalf("diagnostics %q, want %q", got, want)
	}
}

// A wrong type or rule is an error at its line, and the key falls back to
// the layer below.
func TestTypeAndRangeErrorsFallBack(t *testing.T) {
	l := newLayout(t, "steeringMode: some\nmcp:\n  stopGraceMs: \"5\"\n  stopTimeoutMs: -1\ntools:\n  bash:\n    useClientTerminal: [x]\n    maxTimeoutMs: 1.10\ncompaction: 3\nsessionDir: relative/dir\n", "")
	r, ds := Resolve(Options{ConfigDir: l.config})
	if r.Settings != Defaults() {
		t.Fatalf("settings %+v, want every key at its default", r.Settings)
	}
	want := []string{
		"settings.yaml:1: steeringMode: must be one of one-at-a-time, all",
		"settings.yaml:3: mcp.stopGraceMs: want an integer, have a string",
		"settings.yaml:4: mcp.stopTimeoutMs: must be at least 0",
		"settings.yaml:7: tools.bash.useClientTerminal: want a boolean, have a sequence",
		"settings.yaml:8: tools.bash.maxTimeoutMs: want an integer, have a number",
		"settings.yaml:9: compaction: want a mapping, have an integer",
		"settings.yaml:10: sessionDir: must be an absolute path, or start with ~",
	}
	if len(ds) != len(want) {
		t.Fatalf("%d diagnostics %v, want %d", len(ds), ds, len(want))
	}
	for i, d := range ds {
		if !strings.HasSuffix(d.String(), want[i]) || d.Severity != SeverityError {
			t.Errorf("diagnostic %d %q, want …%q", i, d.String(), want[i])
		}
	}
}

// An unknown key is a warning at its line, and is otherwise ignored.
func TestUnknownKeyReported(t *testing.T) {
	l := newLayout(t, "theme: dark\nmcp:\n  retries: 3\n", "")
	r, ds := Resolve(Options{ConfigDir: l.config})
	if r.Settings != Defaults() || len(ds) != 2 {
		t.Fatalf("%+v %v", r.Settings, ds)
	}
	if ds[0].Path != "theme" || ds[0].Line != 1 || ds[0].Text != textUnknown || ds[0].Severity != SeverityWarning ||
		ds[1].Path != "mcp.retries" || ds[1].Line != 3 {
		t.Fatalf("diagnostics %v", ds)
	}
}

// A credential-shaped key is an error, wherever it is, and its value is
// never printed.
func TestCredentialPathRefused(t *testing.T) {
	l := newLayout(t, "apiKey: sk-not-a-real-key\nmcp:\n  auth_token: hunter2\nproviderPassword: x\n", "")
	_, ds := Resolve(Options{ConfigDir: l.config})
	if len(ds) != 3 {
		t.Fatalf("diagnostics %v, want three", ds)
	}
	for _, d := range ds {
		s := d.String()
		if d.Text != textCredential || d.Severity != SeverityError || strings.Contains(s, "sk-not") || strings.Contains(s, "hunter2") {
			t.Fatalf("diagnostic %q", s)
		}
	}
	for _, p := range []string{"keepRecentTokens", "maxTimeoutMs", "tokens", "monkeys"} {
		if isCredentialPath(p) {
			t.Errorf("%s taken for a credential", p)
		}
	}
	for _, p := range []string{"apiKey", "x.api-key", "Secret", "githubToken", "db_password"} {
		if !isCredentialPath(p) {
			t.Errorf("%s not taken for a credential", p)
		}
	}
}

// Read selects session or process keys; zero selects both.
func TestReadFiltering(t *testing.T) {
	l := newLayout(t, "steeringMode: all\nshellCommandPrefix: set -e\n", "")
	session, _ := Resolve(Options{ConfigDir: l.config, Read: ReadSession})
	process, _ := Resolve(Options{ConfigDir: l.config, Read: ReadProcess})
	both, _ := Resolve(Options{ConfigDir: l.config})
	if session.Settings.SteeringMode != "all" || session.Settings.ShellCommandPrefix != "" {
		t.Errorf("session read %+v", session.Settings)
	}
	if process.Settings.SteeringMode != "one-at-a-time" || process.Settings.ShellCommandPrefix != "set -e" {
		t.Errorf("process read %+v", process.Settings)
	}
	if both.Settings.SteeringMode != "all" || both.Settings.ShellCommandPrefix != "set -e" {
		t.Errorf("both %+v", both.Settings)
	}
}

// An origin names the file and the key's line, through nesting and an
// alias, and outside a repository the project root is the working
// directory.
func TestOriginsAndLines(t *testing.T) {
	l := newLayout(t, "# head\n\nbase: &b 42\nmcp:\n  # c\n  stopGraceMs: *b\n", "")
	r, ds := Resolve(Options{ConfigDir: l.config})
	if r.Settings.MCP.StopGraceMs != 42 {
		t.Fatalf("alias not read: %+v %v", r.Settings, ds)
	}
	if o := r.Origins["mcp.stopGraceMs"]; o.Line != 6 || o.File != filepath.Join(l.config, "settings.yaml") {
		t.Fatalf("origin %+v", o)
	}
	plain := t.TempDir()
	if f, ok := ProjectFile(plain); !ok || f != filepath.Join(plain, ".gobble", "settings.yaml") {
		t.Fatalf("ProjectFile outside a repository = %q %v", f, ok)
	}
	if _, ok := ProjectFile(""); ok {
		t.Fatal("ProjectFile(\"\") found one")
	}
	if GlobalFile("") != "" || GlobalFile(l.config) != filepath.Join(l.config, "settings.yaml") {
		t.Fatal("GlobalFile")
	}
}
