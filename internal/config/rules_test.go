package config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// effects is each rule as "effect: text @layer:line".
func effects(rules []PermissionRule) []string {
	var out []string
	for _, r := range rules {
		e, t := r.Effect()
		out = append(out, fmt.Sprintf("%s: %s @%s:%d", e, t, r.Source.Layer, r.Source.Line))
	}
	return out
}

// The rules of the global file, a trusted project's and --set's are joined
// in that order, each with its file and line; a rule that is not one is
// reported with its line and left out, and the rest of its layer applies;
// a value that is not a list gives none (0005-PLAN F6-2 decisions 2 and 4).
func TestRulesJoinAcrossLayers(t *testing.T) {
	cfg, project := t.TempDir(), t.TempDir()
	global := "permissions:\n  rules:\n    - ask: bash(*)\n    - allow: bash(git status*)\n    - maybe: read\n    - deny: read([a)\n"
	if err := os.WriteFile(filepath.Join(cfg, "settings.yaml"), []byte(global), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".gobble"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".gobble", "settings.yaml"), []byte("permissions:\n  rules:\n    - deny: read(.env)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	set, err := ParseOverride("permissions.rules=[{allow: todo}]")
	if err != nil {
		t.Fatal(err)
	}
	r, ds := Resolve(Options{ConfigDir: cfg, Cwd: project, Trusted: func(string) bool { return true }, Overrides: []Override{set}})
	want := []string{"ask: bash(*) @global:3", "allow: bash(git status*) @global:4", "deny: read(.env) @project:3", "allow: todo @--set:1"}
	if got := effects(r.Settings.Permissions.Rules); !slices.Equal(got, want) {
		t.Fatalf("rules %q, want %q", got, want)
	}
	var lines []string
	for _, d := range ds {
		lines = append(lines, d.String())
	}
	joined := strings.Join(lines, "\n")
	if len(ds) != 2 || !strings.Contains(joined, "settings.yaml:5: permissions.rules: ") || !strings.Contains(joined, `"maybe" is not allow, ask or deny`) ||
		!strings.Contains(joined, "settings.yaml:6: permissions.rules: ") || !strings.Contains(joined, "malformed glob") {
		t.Fatalf("diagnostics:\n%s", joined)
	}
	if err := os.WriteFile(filepath.Join(cfg, "settings.yaml"), []byte("permissions:\n  rules: read\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, ds = Resolve(Options{ConfigDir: cfg})
	if len(r.Settings.Permissions.Rules) != 0 || len(ds) != 1 || !strings.Contains(ds[0].String(), "want a list of rules, have a string") {
		t.Fatalf("not a list: %v, %v", r.Settings.Permissions.Rules, ds)
	}
}

// get --origin shows each rule with its file and line; --set takes a list
// and refuses a rule that is not one (decision 12).
func TestRulesRenderAndSet(t *testing.T) {
	cfg := t.TempDir()
	if err := os.WriteFile(filepath.Join(cfg, "settings.yaml"), []byte("permissions:\n  rules:\n    - allow: bash(git *)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r, _ := Resolve(Options{ConfigDir: cfg})
	got := string(Render(r, "permissions.rules", true))
	if want := "- allow: bash(git *) # from " + filepath.Join(cfg, "settings.yaml") + ":3 (global)\n"; got != want {
		t.Fatalf("render %q, want %q", got, want)
	}
	if _, err := ParseOverride("permissions.rules=[{allow: read, deny: write}]"); err == nil || !strings.Contains(err.Error(), "want a rule") {
		t.Fatalf("--set with a two-key rule: %v", err)
	}
}
