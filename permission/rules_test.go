package permission

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func spec(t *testing.T, effect Mode, text, layer string) Spec {
	t.Helper()
	s, err := ParseSpec(effect, text)
	if err != nil {
		t.Fatal(err)
	}
	s.Source = RuleSource{Layer: layer}
	return s
}

// A rule is <tool> or <tool>(<pattern>), with an effect of allow, ask or
// deny; anything else is refused, naming the rule (0005-PLAN F6-2 decision 2).
func TestParseSpec(t *testing.T) {
	for _, c := range []struct {
		effect          Mode
		text, tool, pat string
	}{
		{ModeAllow, "read", "read", ""},
		{ModeAsk, " bash(git status*) ", "bash", "git status*"},
		{ModeDeny, "read(**/.env)", "read", "**/.env"},
		{ModeAllow, "mcp__github__*", "mcp__github__*", ""},
		{ModeAllow, "bash(echo (hi))", "bash", "echo (hi)"},
	} {
		s, err := ParseSpec(c.effect, c.text)
		if err != nil || s.Tool != c.tool || s.Pattern != c.pat || s.Effect != c.effect {
			t.Errorf("%q: %+v, %v", c.text, s, err)
		}
	}
	for _, c := range []struct {
		effect     Mode
		text, want string
	}{
		{"maybe", "read", `"maybe" is not allow, ask or deny`},
		{ModeAllow, "", "names no tool"},
		{ModeAllow, "(src/**)", "names no tool"},
		{ModeAllow, "read(src/**", "no closing )"},
		{ModeAllow, "read files(x)", "names no tool"},
		{ModeAllow, "read([a)", "malformed glob"},
	} {
		if _, err := ParseSpec(c.effect, c.text); !errors.Is(err, ErrBadRule) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v, want %q", c.text, err, c.want)
		}
	}
	if got := spec(t, ModeAllow, "bash(git *)", LayerGlobal).String(); got != "allow: bash(git *)" {
		t.Errorf("String %q", got)
	}
}

// opencode's wildcard: "*" any text, "?" one character, a trailing " *"
// optional.
func TestWildcard(t *testing.T) {
	for _, c := range []struct {
		in, pat string
		want    bool
	}{
		{"git status -s", "git status*", true},
		{"git", "git *", true},
		{"gitk", "git *", false},
		{"git log", "git *", true},
		{"mcp__github__issue", "mcp__github__*", true},
		{"mcp__gitlab__issue", "mcp__github__*", false},
		{"rm -rf a.b", "rm -rf a?b", true},
		{"a+b", "a+b", true},
		{"line1\nline2", "line1*", true},
	} {
		if got := wildcard(c.in, c.pat); got != c.want {
			t.Errorf("wildcard(%q, %q) = %v", c.in, c.pat, got)
		}
	}
}

// The last matching rule across the layers decides, the defaults before
// them, and no match asks (decisions 2 and 4).
func TestEvaluateLayers(t *testing.T) {
	root := filepath.FromSlash("/work/repo")
	at := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	e := Engine{Root: root,
		Default: func(r Rule) Mode {
			if r.ReadOnly {
				return ModeAllow
			}
			return ModeAsk
		},
		Layers: [][]Spec{
			{spec(t, ModeAsk, "bash(*)", LayerGlobal), spec(t, ModeAllow, "bash(git status*)", LayerGlobal), spec(t, ModeAllow, "edit(src/**)", LayerGlobal)},
			{spec(t, ModeDeny, "read(.env)", LayerProject), spec(t, ModeDeny, "edit(src/gen/**)", LayerProject)},
			{spec(t, ModeAllow, "bash(make test)", LayerStore)},
			{spec(t, ModeAllow, "edit(src/gen/keep.go)", LayerSession)},
		}}
	for _, c := range []struct {
		name  string
		r     Rule
		want  Mode
		layer string
	}{
		{"a read-only default", Rule{Tool: "read", Kind: SubjectPath, Subjects: []string{at("a.go")}, ReadOnly: true}, ModeAllow, LayerDefault},
		{"another tool's default", Rule{Tool: "write", Kind: SubjectPath, Subjects: []string{at("a.go")}}, ModeAsk, LayerDefault},
		{".env at any depth", Rule{Tool: "read", Kind: SubjectPath, Subjects: []string{at("x/y/.env")}, ReadOnly: true}, ModeDeny, LayerProject},
		{"a command's prefix", Rule{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"git status -s"}}, ModeAllow, LayerGlobal},
		{"any other command", Rule{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"rm -rf /"}}, ModeAsk, LayerGlobal},
		{"the store's", Rule{Tool: "bash", Kind: SubjectCommand, Subjects: []string{"make test"}}, ModeAllow, LayerStore},
		{"the global allow", Rule{Tool: "edit", Kind: SubjectPath, Subjects: []string{at("src/a.go")}}, ModeAllow, LayerGlobal},
		{"the project's deny over it", Rule{Tool: "edit", Kind: SubjectPath, Subjects: []string{at("src/gen/b.go")}}, ModeDeny, LayerProject},
		{"the session's allow over that", Rule{Tool: "edit", Kind: SubjectPath, Subjects: []string{at("src/gen/keep.go")}}, ModeAllow, LayerSession},
		{"the strictest of a move's paths", Rule{Tool: "edit", Kind: SubjectPath, Subjects: []string{at("src/a.go"), at("src/gen/b.go")}}, ModeDeny, LayerProject},
		{"the strictest, first", Rule{Tool: "edit", Kind: SubjectPath, Subjects: []string{at("src/gen/b.go"), at("src/a.go")}}, ModeDeny, LayerProject},
	} {
		if v := e.Evaluate(c.r); v.Effect != c.want || v.By.Source.Layer != c.layer {
			t.Errorf("%s: %+v, want %s from %s", c.name, v, c.want, c.layer)
		}
	}
}

// A path pattern is relative to the project root inside it, absolute outside
// it; a pattern on a tool with no subject never matches it (decision 3).
func TestEvaluateSubjects(t *testing.T) {
	root := filepath.FromSlash("/work/repo")
	tmp := filepath.FromSlash("/tmp/scratch/out.txt")
	e := Engine{Root: root, Layers: [][]Spec{{
		spec(t, ModeAllow, "write(/tmp/**)", LayerGlobal),
		spec(t, ModeAllow, "todo(x)", LayerGlobal),
		spec(t, ModeAllow, "todo", LayerGlobal),
		spec(t, ModeDeny, "tool_search(x)", LayerGlobal),
	}}}
	if v := e.Evaluate(Rule{Tool: "write", Kind: SubjectPath, Subjects: []string{tmp}}); v.Effect != ModeAllow {
		t.Errorf("an absolute pattern: %+v", v)
	}
	if v := e.Evaluate(Rule{Tool: "todo"}); v.Effect != ModeAllow {
		t.Errorf("a name-only rule: %+v", v)
	}
	if v := e.Evaluate(Rule{Tool: "tool_search"}); v.Effect != ModeAsk || v.By.Source.Layer != LayerDefault {
		t.Errorf("a pattern on a tool with no subject matched: %+v", v)
	}
}

// A path outside the roots is external_directory's, which asks by default
// and is allowed by a rule (decision 7).
func TestEvaluateExternalDirectory(t *testing.T) {
	out := filepath.FromSlash("/tmp/x/y.txt")
	r := Rule{Tool: "read", Kind: SubjectPath, Subjects: []string{out}, Outside: []string{out}, ReadOnly: true}
	allowRead := func(Rule) Mode { return ModeAllow }
	e := Engine{Default: allowRead}
	if v := e.Evaluate(r); v.Effect != ModeAsk || v.By.Tool != ExternalDirectory {
		t.Errorf("outside, no rule: %+v", v)
	}
	e.Layers = [][]Spec{{spec(t, ModeAllow, "external_directory(/tmp/**)", LayerGlobal)}}
	if v := e.Evaluate(r); v.Effect != ModeAllow || v.By.Tool != ExternalDirectory || v.By.Source.Layer != LayerGlobal {
		t.Errorf("outside, allowed: %+v; want the external_directory rule named", v)
	}
	e.Layers = append(e.Layers, []Spec{spec(t, ModeDeny, "read(**/y.txt)", LayerSession)})
	if v := e.Evaluate(r); v.Effect != ModeDeny || v.By.Tool != "read" {
		t.Errorf("outside, allowed, the read denied: %+v; want the deny", v)
	}
}

// gobble's limits come after the rules, and no rule lifts them: a write to a
// protected path is denied; a write to a configuration path and a delete
// ask; a deny stays a deny (decision 5).
func TestEvaluateLimits(t *testing.T) {
	root := filepath.FromSlash("/work/repo")
	home := filepath.FromSlash("/home/u")
	cfg := filepath.Join(home, ".config", "gobble")
	allowAll := [][]Spec{{spec(t, ModeAllow, "*", LayerGlobal)}}
	e := Engine{Root: root, Home: home, ConfigDir: cfg, Layers: allowAll}
	for _, c := range []struct {
		name, tool, path string
		want             Mode
		limit            bool
	}{
		{"under .git", "write", filepath.Join(root, ".git", "config"), ModeDeny, true},
		{"a nested .git", "edit", filepath.Join(root, "vendor", "x", ".git", "HEAD"), ModeDeny, true},
		{"under ~/.ssh", "copy", filepath.Join(home, ".ssh", "id"), ModeDeny, true},
		{"auth.json", "write", filepath.Join(cfg, "auth.json"), ModeDeny, true},
		{"trust.json", "delete", filepath.Join(cfg, "trust.json"), ModeDeny, true},
		{"permissions.json", "move", filepath.Join(cfg, "permissions.json"), ModeDeny, true},
		{"settings.yaml", "edit", filepath.Join(cfg, "settings.yaml"), ModeAsk, true},
		{"the project's .gobble", "write", filepath.Join(root, ".gobble", "settings.yaml"), ModeAsk, true},
		{"an AGENTS.md", "edit", filepath.Join(root, "sub", "AGENTS.md"), ModeAsk, true},
		{"the project's skills", "mkdir", filepath.Join(root, ".agents", "skills", "x"), ModeAsk, true},
		{"a delete", "delete", filepath.Join(root, "a.go"), ModeAsk, true},
		{"an ordinary write", "write", filepath.Join(root, "a.go"), ModeAllow, false},
		{"a read under .git", "read", filepath.Join(root, ".git", "config"), ModeAllow, false},
	} {
		v := e.Evaluate(Rule{Tool: c.tool, Kind: SubjectPath, Subjects: []string{c.path}})
		if v.Effect != c.want || (v.Limit != "") != c.limit || (c.limit && v.By.Source.Layer != LayerLimit) {
			t.Errorf("%s: %+v, want %s, limit %v", c.name, v, c.want, c.limit)
		}
	}
	e.Layers = [][]Spec{{spec(t, ModeDeny, "delete", LayerGlobal)}}
	if v := e.Evaluate(Rule{Tool: "delete", Kind: SubjectPath, Subjects: []string{filepath.Join(root, "a.go")}}); v.Effect != ModeDeny || v.Limit != "" {
		t.Errorf("a deny became a question: %+v", v)
	}
	// With no rule, the default's question is the limit's, so it is marked
	// and --yes leaves it (decision 8).
	e.Layers = nil
	for _, c := range []struct{ tool, path string }{{"delete", filepath.Join(root, "a.go")}, {"write", filepath.Join(root, "AGENTS.md")}} {
		if v := e.Evaluate(Rule{Tool: c.tool, Kind: SubjectPath, Subjects: []string{c.path}}); v.Effect != ModeAsk || v.Limit == "" {
			t.Errorf("%s %s with no rule: %+v, want the limit's question", c.tool, c.path, v)
		}
	}
}

// A tool whose last rule is a deny for every subject is hidden; one with a
// narrower deny, or a later allow, is not (decision 6).
func TestHidden(t *testing.T) {
	e := Engine{Layers: [][]Spec{
		{spec(t, ModeDeny, "mcp__github__*", LayerGlobal), spec(t, ModeDeny, "bash(rm *)", LayerGlobal), spec(t, ModeDeny, "powershell", LayerGlobal)},
		{spec(t, ModeAllow, "powershell", LayerProject)},
	}}
	for tool, want := range map[string]bool{"mcp__github__issue": true, "bash": false, "powershell": false, "read": false} {
		if got := e.Hidden(tool); got != want {
			t.Errorf("Hidden(%s) = %v", tool, got)
		}
	}
}
