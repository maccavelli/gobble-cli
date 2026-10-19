package builtin

import (
	"encoding/json/v2"
	"path/filepath"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

// Each file tool names its paths, resolved against the working directory,
// a move and a copy both of theirs, and a search with no path the working
// directory; bash and powershell name their whole command; todo names
// nothing (0005-PLAN F6-2 decision 3).
func TestSubjects(t *testing.T) {
	cwd := t.TempDir()
	abs := filepath.Join(t.TempDir(), "elsewhere.txt")
	in := func(p string) string { return filepath.Join(cwd, p) }
	cases := map[string]struct {
		args  string
		paths []string
		cmds  []string
		saves []string
	}{
		"read":   {args: `{"path":"a.txt"}`, paths: []string{in("a.txt")}},
		"write":  {args: `{"path":"sub/b.txt","content":"x"}`, paths: []string{in(filepath.Join("sub", "b.txt"))}},
		"edit":   {args: `{"path":` + quote(t, abs) + `,"edits":[{"oldText":"a","newText":"b"}]}`, paths: []string{abs}},
		"grep":   {args: `{"pattern":"x"}`, paths: []string{cwd}},
		"find":   {args: `{"pattern":"*.go","path":"src"}`, paths: []string{in("src")}},
		"tree":   {args: `{}`, paths: []string{cwd}},
		"move":   {args: `{"from":"a.txt","to":"b.txt"}`, paths: []string{in("a.txt"), in("b.txt")}},
		"copy":   {args: `{"from":"a.txt","to":` + quote(t, abs) + `}`, paths: []string{in("a.txt"), abs}},
		"delete": {args: `{"path":".env"}`, paths: []string{in(".env")}},
		"mkdir":  {args: `{"path":"d"}`, paths: []string{in("d")}},
		"bash":   {args: `{"command":"git status && rm -rf x"}`, cmds: []string{"git status", "rm -rf x"}, saves: []string{"git status *", "rm *"}},
		"todo":   {args: `{"todos":[]}`},
	}
	tools := append(Tools(), powershellWith(Options{}.withDefaults()))
	seen := map[string]bool{}
	for _, tl := range tools {
		name := tl.Spec().Name
		c, ok := cases[name]
		if name == "powershell" {
			// PowerShell is one raw subject (0005-PLAN F6-3 decision 8).
			c, ok = cases["bash"], true
			c.cmds, c.saves = []string{"git status && rm -rf x"}, []string{""}
		}
		if !ok {
			t.Errorf("%s: no case", name)
			continue
		}
		seen[name] = true
		var got tool.Subject
		if sj, ok := tl.(tool.Subjecter); ok {
			got = sj.Subject(tool.Call{Name: name, Args: []byte(c.args)}, tool.Env{Cwd: cwd})
		}
		if !slices.Equal(got.Paths, c.paths) || !slices.Equal(got.Commands, c.cmds) || !slices.Equal(got.Saves, c.saves) {
			t.Errorf("%s: subject %+v, want paths %q, commands %q, saves %q", name, got, c.paths, c.cmds, c.saves)
		}
	}
	for name := range cases {
		if !seen[name] {
			t.Errorf("%s: no such tool", name)
		}
	}
}

// quote is p as a JSON string.
func quote(t *testing.T, p string) string {
	t.Helper()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
