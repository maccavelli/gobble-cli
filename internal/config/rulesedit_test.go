package config

import (
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// fileRules are file's own rules, as "effect: text @line".
func fileRules(t *testing.T, file string) []string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	root, ds := Parse(file, data)
	if root == nil || len(ds) != 0 {
		t.Fatalf("parse: %v", ds)
	}
	r, ds := Resolve(Options{ConfigDir: filepath.Dir(file)})
	if len(ds) != 0 {
		t.Fatalf("resolve: %v", ds)
	}
	var out []string
	for _, p := range r.Settings.Permissions.Rules {
		e, x := p.Effect()
		out = append(out, e+": "+x+" @"+strconv.Itoa(p.Source.Line))
	}
	return out
}

// Appending to the shipped template uncomments its block, a later append
// adds a line after the last item, and every other line stays (0005-PLAN
// F6-2b decision E).
func TestAppendRuleTemplate(t *testing.T) {
	file := filepath.Join(t.TempDir(), "settings.yaml")
	if err := os.WriteFile(file, Template(), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AppendRule(file, PermissionRule{Allow: "bash(git status)"}); err != nil {
		t.Fatal(err)
	}
	if err := AppendRule(file, PermissionRule{Deny: "edit(/src/a: b #c.go)"}); err != nil {
		t.Fatal(err)
	}
	got := fileRules(t, file)
	if len(got) != 2 || !strings.HasPrefix(got[0], "allow: bash(git status) @") || !strings.HasPrefix(got[1], "deny: edit(/src/a: b #c.go) @") {
		t.Fatalf("rules %q", got)
	}
	data, _ := os.ReadFile(file)
	before := strings.Split(string(Template()), "\n")
	after := strings.Split(string(data), "\n")
	// Two template lines change ("# permissions:", "#   rules: []") into
	// three: the key, the list's key and its first item; the second append
	// adds one line.
	if len(after) != len(before)+2 {
		t.Fatalf("%d lines, want %d", len(after), len(before)+2)
	}
	for _, l := range before {
		if l == "# permissions:" || l == "#   rules: []" {
			continue
		}
		if !slices.Contains(after, l) {
			t.Errorf("line lost: %q", l)
		}
	}
	if !strings.Contains(string(data), "permissions:\n  rules:\n    - allow: bash(git status)\n    - deny: \"edit(/src/a: b #c.go)\"\n") {
		t.Errorf("block:\n%s", data)
	}
}

// A hand-written list keeps its comments through an append and a removal;
// a removal names the line and the rule, and refuses a line that no longer
// holds it; removing the last item leaves []; a flow list is written again
// as a block; a value that is not a list is refused (decisions E and H).
func TestAppendAndRemoveRule(t *testing.T) {
	file := filepath.Join(t.TempDir(), "settings.yaml")
	hand := "# mine\npermissions:\n  rules:\n    # the reads\n    - allow: read   # always\n    - deny: read(.env)\n    # shells\n    - ask: bash(*)\nsteeringMode: all\n"
	if err := os.WriteFile(file, []byte(hand), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AppendRule(file, PermissionRule{Allow: "todo"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(file)
	if want := strings.Replace(hand, "    - ask: bash(*)\n", "    - ask: bash(*)\n    - allow: todo\n", 1); string(data) != want {
		t.Fatalf("append:\n%s\nwant:\n%s", data, want)
	}
	if err := RemoveRule(file, 5, PermissionRule{Deny: "read(.env)"}); err == nil || !strings.Contains(err.Error(), "settings.yaml:5: no longer holds deny: read(.env)") {
		t.Fatalf("a stale line: %v", err)
	}
	if err := RemoveRule(file, 6, PermissionRule{Deny: "read(.env)"}); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(file)
	if want := "# mine\npermissions:\n  rules:\n    # the reads\n    - allow: read   # always\n    # shells\n    - ask: bash(*)\n    - allow: todo\nsteeringMode: all\n"; string(data) != want {
		t.Fatalf("remove:\n%s\nwant:\n%s", data, want)
	}
	for _, r := range []struct {
		line int
		rule PermissionRule
	}{{5, PermissionRule{Allow: "read"}}, {6, PermissionRule{Ask: "bash(*)"}}, {6, PermissionRule{Allow: "todo"}}} {
		if err := RemoveRule(file, r.line, r.rule); err != nil {
			t.Fatalf("remove %+v: %v", r, err)
		}
	}
	if got := fileRules(t, file); len(got) != 0 {
		t.Fatalf("rules %q, want none", got)
	}
	data, _ = os.ReadFile(file)
	if !strings.Contains(string(data), "steeringMode: all") || !strings.Contains(string(data), "# mine") {
		t.Errorf("the rest lost:\n%s", data)
	}

	if err := os.WriteFile(file, []byte("permissions:\n  rules: [{allow: read}]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AppendRule(file, PermissionRule{Ask: "write"}); err != nil {
		t.Fatal(err)
	}
	if got := fileRules(t, file); len(got) != 2 || got[0] != "allow: read @3" || got[1] != "ask: write @4" {
		t.Fatalf("flow list: %q", got)
	}
	if err := os.WriteFile(file, []byte("permissions:\n  rules: read\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := AppendRule(file, PermissionRule{Ask: "write"}); err == nil || !strings.Contains(err.Error(), "want a list of rules") {
		t.Fatalf("not a list: %v", err)
	}
}
