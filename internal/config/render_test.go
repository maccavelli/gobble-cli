package config

import (
	"path/filepath"
	"strings"
	"testing"
)

// renderLayout resolves a global file, an untrusted project file and one
// --set, for Render's tests.
func renderLayout(t *testing.T) (Resolved, string, string) {
	t.Helper()
	l := newLayout(t, "steeringMode: all\nmcp:\n  stopGraceMs: 100\n", "externalEditor: vim\n")
	set, err := ParseOverride("followUpMode=all")
	if err != nil {
		t.Fatal(err)
	}
	r, _ := Resolve(Options{ConfigDir: l.config, Cwd: l.project, Overrides: []Override{set}})
	return r, filepath.Join(l.config, "settings.yaml"), filepath.Join(l.project, ".gobble", "settings.yaml")
}

// The whole document is YAML in the template's order, every key present.
func TestRenderDocument(t *testing.T) {
	r, _, _ := renderLayout(t)
	want := `defaultProvider: ""
defaultModel: ""
cacheRetention: short
defaultThinkingLevel: off
steeringMode: all
followUpMode: all
sessionDir: ""
compaction:
  keepRecentTokens: 20000
retry:
  enabled: true
  maxRetries: 3
  baseDelayMs: 2000
  maxDelayMs: 60000
  offlineWaitMs: 300000
shellPath: ""
shellCommandPrefix: ""
tools:
  bash:
    useClientTerminal: false
    maxTimeoutMs: 600000
mcp:
  startupWaitMs: 10000
  stopGraceMs: 100
  stopTimeoutMs: 2000
externalEditor: ""
`
	if got := string(Render(r, "", false)); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	if got := string(Render(r, "mcp", false)); got != "mcp:\n  startupWaitMs: 10000\n  stopGraceMs: 100\n  stopTimeoutMs: 2000\n" {
		t.Fatalf("mcp:\n%s", got)
	}
	if got := string(Render(r, "steeringMode", false)); got != "all\n" {
		t.Fatalf("steeringMode: %q", got)
	}
}

// --origin names each value's file, line and layer, and lists the project
// layer that was read but not applied, with the keys it would set.
func TestRenderOrigin(t *testing.T) {
	r, global, project := renderLayout(t)
	got := string(Render(r, "", true))
	for _, line := range []string{
		"defaultThinkingLevel: off # default",
		"steeringMode: all # from " + global + ":1 (global)",
		"followUpMode: all # from --set",
		"  stopGraceMs: 100 # from " + global + ":3 (global)",
		`externalEditor: "" # default`,
		"# not applied: " + project + ": project settings apply once the project is trusted (0005-PLAN F6)",
		"#   it would set: externalEditor",
	} {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("no line %q in:\n%s", line, got)
		}
	}
	if got := string(Render(r, "mcp.stopGraceMs", true)); got != "100 # from "+global+":3 (global)\n" {
		t.Fatalf("one key: %q", got)
	}
}
