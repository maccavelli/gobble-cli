package permission

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
)

// A leading "/" anchors a path pattern at the root, as gitignore's does;
// without it a name matches at any depth (0005-PLAN F6-2b decision C).
func TestPatternAnchored(t *testing.T) {
	root := filepath.FromSlash("/work/repo")
	e := Engine{Root: root, Layers: [][]Spec{{spec(t, ModeDeny, "write(/README.md)", LayerGlobal), spec(t, ModeDeny, "write(NOTES.md)", LayerGlobal)}}}
	for _, c := range []struct {
		rel  string
		want Mode
	}{{"README.md", ModeDeny}, {"docs/README.md", ModeAsk}, {"NOTES.md", ModeDeny}, {"docs/NOTES.md", ModeDeny}} {
		r := Rule{Tool: "write", Kind: SubjectPath, Subjects: []string{filepath.Join(root, filepath.FromSlash(c.rel))}}
		if v := e.Evaluate(r); v.Effect != c.want {
			t.Errorf("%s: %+v, want %s", c.rel, v, c.want)
		}
	}
}

// An "always" saves rules that match the call's subjects as they were
// matched and nothing else; an allow also saves external_directory for
// each outside path, and a deny the tool's rule alone; a subject holding a
// pattern's character cannot be saved (decisions A and C).
func TestRemember(t *testing.T) {
	root := filepath.FromSlash("/work/repo")
	out := filepath.FromSlash("/tmp/x/y.txt")
	outShown := filepath.ToSlash(out)
	e := Engine{Root: root}
	texts := func(rules []Spec) []string {
		var s []string
		for _, r := range rules {
			s = append(s, r.String())
		}
		return s
	}
	for _, c := range []struct {
		name   string
		effect Mode
		r      Rule
		want   []string
	}{
		{"a path from the root", ModeAllow, Rule{Tool: "edit", Kind: SubjectPath, Subjects: []string{filepath.Join(root, "src", "a.go")}},
			[]string{"allow: edit(/src/a.go)"}},
		{"a root-level name, anchored", ModeDeny, Rule{Tool: "write", Kind: SubjectPath, Subjects: []string{filepath.Join(root, "README.md")}},
			[]string{"deny: write(/README.md)"}},
		{"a move's two paths", ModeAllow, Rule{Tool: "move", Kind: SubjectPath, Subjects: []string{filepath.Join(root, "a"), filepath.Join(root, "b")}},
			[]string{"allow: move(/a)", "allow: move(/b)"}},
		{"an outside path, allowed", ModeAllow, Rule{Tool: "read", Kind: SubjectPath, Subjects: []string{out}, Outside: []string{out}},
			[]string{"allow: read(" + outShown + ")", "allow: external_directory(" + outShown + ")"}},
		{"an outside path, denied", ModeDeny, Rule{Tool: "read", Kind: SubjectPath, Subjects: []string{out}, Outside: []string{out}},
			[]string{"deny: read(" + outShown + ")"}},
		{"a command whole", ModeAllow, Rule{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"git status -s && make test"}},
			[]string{"allow: bash(git status -s && make test)"}},
		{"a tool with no subject", ModeDeny, Rule{Tool: "mcp__fx__echo"}, []string{"deny: mcp__fx__echo"}},
	} {
		rules, ok := e.Remember(c.effect, c.r)
		if got := texts(rules); !ok || !slices.Equal(got, c.want) {
			t.Errorf("%s: %q, %v, want %q", c.name, got, ok, c.want)
			continue
		}
		// Each saved rule decides the call it was saved for, and its
		// effect is the answer's.
		saved := Engine{Root: root, Default: func(Rule) Mode { return ModeAsk }, Layers: [][]Spec{rules}}
		if v := saved.Evaluate(c.r); v.Effect != c.effect {
			t.Errorf("%s: the saved rules give %+v", c.name, v)
		}
	}
	for _, r := range []Rule{
		{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"find . -name '*.go'"}},
		{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"ls a?b"}},
		{Tool: "edit", Kind: SubjectPath, Subjects: []string{filepath.Join(root, "a[1].go")}},
		{Tool: "edit", Kind: SubjectPath, Subjects: []string{filepath.Join(root, "{a,b}.go")}},
		{Tool: "read", Kind: SubjectPath, Subjects: []string{filepath.Join(root, "ok.go")}, Outside: []string{filepath.FromSlash("/tmp/*.log")}},
	} {
		if rules, ok := e.Remember(ModeAllow, r); ok {
			t.Errorf("%+v: saved %v; want no rule, since none matches it alone", r, rules)
		}
	}
	// A command's arity pattern is saved in place of its text, once for two
	// subjects that share it; a glob in the prefix keeps the call to "once"
	// (0005-PLAN F6-3 decision 4a).
	arity := Rule{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"git status -s", "git status", "ls $X"},
		Saves: []string{"git status *", "git status *", ""}}
	if rules, ok := e.Remember(ModeAllow, arity); !ok || !slices.Equal(texts(rules), []string{"allow: bash(git status *)", "allow: bash(ls $X)"}) {
		t.Errorf("arity: %q, %v", texts(rules), ok)
	}
	for _, save := range []string{"git *.go *", "make t? *", "x[1] *", "{a,b} *"} {
		if rules, ok := e.Remember(ModeAllow, Rule{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"x"}, Saves: []string{save}}); ok {
			t.Errorf("%q: saved %v; want once only", save, rules)
		}
	}
	// A saved command matches itself alone.
	rules, _ := e.Remember(ModeAllow, Rule{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"git status"}})
	if v := (Engine{Layers: [][]Spec{rules}}).Evaluate(Rule{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"git status -s"}}); v.Effect != ModeAsk {
		t.Errorf("the saved command matched a longer one: %+v", v)
	}
}

// permissions.json keeps each root's rules by its real path, owner-only,
// in the order saved, once each; a root's rules are its own, and a parent's
// do not apply; removing the last drops the root; a bad rule is named and
// the rest load (decisions 10 and E).
func TestRuleStore(t *testing.T) {
	dir := t.TempDir()
	s := RuleStore{Path: filepath.Join(dir, PermissionsFile)}
	root := filepath.Join(dir, "proj")
	child := filepath.Join(root, "sub")
	if err := os.MkdirAll(child, 0o700); err != nil {
		t.Fatal(err)
	}
	if got, err := s.Rules(root); err != nil || len(got) != 0 {
		t.Fatalf("no file: %v, %v", got, err)
	}
	a, d := spec(t, ModeAllow, "edit(/src/a.go)", ""), spec(t, ModeDeny, "bash(rm -rf build)", "")
	if _, err := s.Add(t.Context(), root, a, d); err != nil {
		t.Fatal(err)
	}
	at, err := s.Add(t.Context(), root, a)
	if err != nil {
		t.Fatal(err)
	}
	real, _ := filepath.EvalSymlinks(root)
	if at != real {
		t.Errorf("saved at %q, want the real path %q", at, real)
	}
	got, err := s.Rules(root)
	if err != nil || len(got) != 2 || got[0].String() != "allow: edit(/src/a.go)" || got[1].String() != "deny: bash(rm -rf build)" ||
		got[0].Source != (RuleSource{Layer: LayerStore, File: s.Path}) {
		t.Fatalf("rules %v, %v", got, err)
	}
	if got[0].Source.String() != s.Path+" (project store)" {
		t.Errorf("source %q", got[0].Source)
	}
	if sub, err := s.Rules(child); err != nil || len(sub) != 0 {
		t.Errorf("a child root took its parent's rules: %v, %v", sub, err)
	}
	data, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  " + jsonString(real) + ": [\n    {\n      \"allow\": \"edit(/src/a.go)\"\n    },\n    {\n      \"deny\": \"bash(rm -rf build)\"\n    }\n  ]\n}\n"
	if string(data) != want {
		t.Errorf("file:\n%s\nwant:\n%s", data, want)
	}
	if info, err := os.Stat(s.Path); err != nil || runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, %v", info.Mode(), err)
	}
	if ok, err := s.Remove(t.Context(), root, d); !ok || err != nil {
		t.Fatalf("remove: %v, %v", ok, err)
	}
	if ok, _ := s.Remove(t.Context(), root, d); ok {
		t.Error("removed a rule twice")
	}
	if ok, _ := s.Remove(t.Context(), root, a); !ok {
		t.Fatal("remove the last")
	}
	if m, _ := s.Load(); len(m) != 0 {
		t.Errorf("a root with no rules kept: %v", m)
	}
	if err := os.WriteFile(s.Path, []byte(`{`+jsonString(real)+`: [{"allow": "read"}, {"allow": "x", "deny": "y"}, {"ask": "read([a)"}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = s.Rules(root)
	if len(got) != 1 || err == nil || !strings.Contains(err.Error(), "want one of allow, ask or deny") || !strings.Contains(err.Error(), "malformed glob") {
		t.Errorf("bad rules: %v, %v", got, err)
	}
}

// Saves from many goroutines all land, under the file's lock.
func TestRuleStoreConcurrent(t *testing.T) {
	dir := t.TempDir()
	s := RuleStore{Path: filepath.Join(dir, PermissionsFile)}
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Go(func() {
			if _, err := s.Add(t.Context(), dir, Spec{Effect: ModeAllow, Tool: "bash", Pattern: "make t" + string(rune('0'+i))}); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if got, err := s.Rules(dir); err != nil || len(got) != 8 {
		t.Fatalf("%d rules, %v; want 8", len(got), err)
	}
}

// jsonString is p as JSON writes it.
func jsonString(p string) string {
	return `"` + strings.ReplaceAll(p, `\`, `\\`) + `"`
}
