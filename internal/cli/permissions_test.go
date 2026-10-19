package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/permission"
)

// The agent's global save appends to the global settings.yaml, created
// from the template when it is missing, and gives the file's rules with
// their lines; its removal takes the rule off at its line, and refuses a
// rule from another file; the store is permissions.json in the config
// directory (0005-PLAN F6-2b decision E).
func TestPermissionOptions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	o := permissionOptions()
	if o.Store.Path != filepath.Join(home, permission.PermissionsFile) {
		t.Fatalf("store %q", o.Store.Path)
	}
	add := func(effect permission.Mode, text string) []permission.Spec {
		t.Helper()
		s, err := permission.ParseSpec(effect, text)
		if err != nil {
			t.Fatal(err)
		}
		rules, err := o.AddGlobal(s)
		if err != nil {
			t.Fatal(err)
		}
		return rules
	}
	add(permission.ModeAllow, "bash(git status)")
	rules := add(permission.ModeDeny, "edit(/a.go)")
	file := filepath.Join(home, "settings.yaml")
	if len(rules) != 2 || rules[1].String() != "deny: edit(/a.go)" || rules[1].Source.File != file || rules[1].Source.Line != rules[0].Source.Line+1 ||
		rules[0].Source.Layer != permission.LayerGlobal {
		t.Fatalf("rules %+v", rules)
	}
	data, err := os.ReadFile(file)
	if err != nil || !strings.Contains(string(data), "# The rules that decide whether a tool call runs") {
		t.Fatalf("not the template: %v", err)
	}
	rules, err = o.RemoveGlobal(rules[0])
	if err != nil || len(rules) != 1 || rules[0].String() != "deny: edit(/a.go)" {
		t.Fatalf("remove: %+v, %v", rules, err)
	}
	other := rules[0]
	other.Source.File = filepath.Join(home, "elsewhere.yaml")
	if _, err := o.RemoveGlobal(other); err == nil || !strings.Contains(err.Error(), "not in "+file) {
		t.Fatalf("another file's rule: %v", err)
	}
}
