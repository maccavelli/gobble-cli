package builtin

import (
	"encoding/json/jsontext"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

// project makes a repository in a temporary directory, the files named
// ("/" ending a directory), and returns its path.
func project(t *testing.T, files map[string]string) string {
	t.Helper()
	top := t.TempDir()
	files[".git/HEAD"] = "ref: refs/heads/main\n"
	for name, text := range files {
		p := filepath.Join(top, filepath.FromSlash(name))
		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(p, 0o750); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return top
}

// runIn runs tl with args in dir, and returns its result.
func runIn(t *testing.T, tl tool.Tool, dir, args string) tool.Result {
	t.Helper()
	r, err := tl.Run(t.Context(), tool.Call{ID: "c1", Name: tl.Spec().Name, Args: jsontext.Value(args)}, tool.Env{Cwd: dir, Roots: []string{dir}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func wantText(t *testing.T, r tool.Result, want string) {
	t.Helper()
	if r.IsError || r.Text() != want {
		t.Fatalf("result (isError %v):\n%s\nwant:\n%s", r.IsError, r.Text(), want)
	}
}

func wantError(t *testing.T, r tool.Result, want string) {
	t.Helper()
	if !r.IsError || !strings.Contains(r.Text(), want) {
		t.Fatalf("result (isError %v):\n%s\nwant an error containing %q", r.IsError, r.Text(), want)
	}
}

// grep's form: path:N: text, sorted by path, then line; ignore rules,
// binary files and .git skipped; hidden files searched.
func TestGrepFormat(t *testing.T) {
	dir := project(t, map[string]string{
		".gitignore":   "*.log\n",
		"b.go":         "package b\nfunc Hello() {}\n",
		"a/x.go":       "// hello\nfunc hello() {}\r\n",
		".hidden/h.go": "hello\n",
		"skip.log":     "hello\n",
		"bin.dat":      "hello\x00",
	})
	wantText(t, runIn(t, Grep(), dir, `{"pattern":"hello"}`), ".hidden/h.go:1: hello\na/x.go:1: // hello\na/x.go:2: func hello() {}")
	wantText(t, runIn(t, Grep(), dir, `{"pattern":"hello","ignoreCase":true,"glob":"*.go"}`),
		".hidden/h.go:1: hello\na/x.go:1: // hello\na/x.go:2: func hello() {}\nb.go:2: func Hello() {}")
	wantText(t, runIn(t, Grep(), dir, `{"pattern":"hello","glob":"a/*.go"}`), "a/x.go:1: // hello\na/x.go:2: func hello() {}")
	wantText(t, runIn(t, Grep(), dir, `{"pattern":"Hello()","literal":true}`), "b.go:2: func Hello() {}")
	wantText(t, runIn(t, Grep(), dir, `{"pattern":"Hello","path":"b.go"}`), "b.go:2: func Hello() {}")
}

// context lines are path-N- text, merged so no line shows twice.
func TestGrepContext(t *testing.T) {
	dir := project(t, map[string]string{"f.txt": "1\n2\nhit\n4\nhit\n6\n7\n8\n"})
	wantText(t, runIn(t, Grep(), dir, `{"pattern":"hit","context":1}`), "f.txt-2- 2\nf.txt:3: hit\nf.txt-4- 4\nf.txt:5: hit\nf.txt-6- 6")
}

// No match is a plain result; a bad pattern and a missing path are errors.
func TestGrepRefusals(t *testing.T) {
	dir := project(t, map[string]string{"notes.txt": "x\n"})
	wantText(t, runIn(t, Grep(), dir, `{"pattern":"absent"}`), "grep .: no matches for absent")
	wantError(t, runIn(t, Grep(), dir, `{"pattern":"(unclosed"}`), "grep: the pattern is not valid RE2 (missing closing )); escape it, or set literal")
	wantError(t, runIn(t, Grep(), dir, `{"pattern":"x","glob":"[x"}`), "the glob \"[x\" is malformed")
	wantError(t, runIn(t, Grep(), dir, `{"pattern":"x","path":"note"}`), "grep note: no such file or directory; did you mean notes.txt?")
}

// The notices: the limit, the output cap, cut lines and files too big.
func TestGrepNotices(t *testing.T) {
	long := strings.Repeat("y", 600)
	var many strings.Builder
	for range 1200 {
		many.WriteString("hit " + long + "\n")
	}
	dir := project(t, map[string]string{"few.txt": "hit\nhit\nhit\n", "many.txt": many.String(), "long.txt": "hit " + long + "\n"})
	r := runIn(t, Grep(), dir, `{"pattern":"hit","glob":"few.txt","limit":2}`)
	wantText(t, r, "few.txt:1: hit\nfew.txt:2: hit\n\n[2 matches shown, the limit; use limit=4 for more, or narrow the pattern or path]")
	r = runIn(t, Grep(), dir, `{"pattern":"hit","glob":"long.txt"}`)
	wantText(t, r, "long.txt:1: hit "+strings.Repeat("y", 496)+"…\n\n[some lines cut at 500 characters; read the file for the whole line]")
	r = runIn(t, Grep(), dir, `{"pattern":"hit","glob":"many.txt","limit":1000}`)
	if !strings.Contains(r.Text(), "[output cut at 50 KB; narrow the pattern or path]") || len(r.Text()) > 52*1024 {
		t.Fatalf("a 1000-match search of long lines gave %d bytes, ending %q", len(r.Text()), r.Text()[max(0, len(r.Text())-200):])
	}
	big := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(big, []byte(strings.Repeat("hit\n", maxGrepFileBytes/4+1)), 0o600); err != nil {
		t.Fatal(err)
	}
	r = runIn(t, Grep(), dir, `{"pattern":"hit","glob":"big.txt"}`)
	wantText(t, r, "grep .: no matches for hit\n\n[1 file over 8 MB was not searched]")
}

// find's form: paths from the search root, "/" after directories, sorted.
func TestFind(t *testing.T) {
	dir := project(t, map[string]string{
		".gitignore":            "dist/\n",
		"main.go":               "",
		"cmd/tool/main.go":      "",
		"cmd/tool/main_test.go": "",
		"docs/a.md":             "",
		"docs/guide/b.md":       "",
		"dist/out.go":           "",
		"node_modules/m.go":     "",
	})
	wantText(t, runIn(t, Find(), dir, `{"pattern":"*.go"}`), "cmd/tool/main.go\ncmd/tool/main_test.go\nmain.go")
	wantText(t, runIn(t, Find(), dir, `{"pattern":"docs/**/*.md"}`), "docs/a.md\ndocs/guide/b.md")
	wantText(t, runIn(t, Find(), dir, `{"pattern":"tool"}`), "cmd/tool/")
	wantText(t, runIn(t, Find(), dir, `{"pattern":"*.go","limit":1}`), "cmd/tool/main.go\n\n[1 paths shown, the limit; use limit=2 for more, or narrow the pattern or path]")
	wantText(t, runIn(t, Find(), dir, `{"pattern":"*.rs"}`), "find .: nothing matches *.rs")
	wantError(t, runIn(t, Find(), dir, `{"pattern":"*","path":"main.go"}`), "find main.go: not a directory")
}

// tree: directories first, files with sizes, depth, and the breadth-first
// budget with its marks and notices.
func TestTree(t *testing.T) {
	dir := project(t, map[string]string{
		"README.md":      "hello",
		"a/a1.txt":       "1",
		"a/a2.txt":       "22",
		"a/deep/d.txt":   "",
		"b/b1.txt":       "",
		"z.txt":          strings.Repeat("z", 2048),
		"node_modules/x": "",
	})
	wantText(t, runIn(t, Tree(), dir, `{}`), "./\n  a/\n    deep/\n    a1.txt (1 B)\n    a2.txt (2 B)\n  b/\n    b1.txt (0 B)\n  README.md (5 B)\n  z.txt (2.0 KB)")
	wantText(t, runIn(t, Tree(), dir, `{"depth":1}`), "./\n  a/\n  b/\n  README.md (5 B)\n  z.txt (2.0 KB)")
	wantText(t, runIn(t, Tree(), dir, `{"limit":5}`),
		"./\n  a/\n    deep/\n    … 2 more\n  b/ …\n  README.md (5 B)\n  z.txt (2.0 KB)\n\n[5 entries shown, the limit; use limit=10 for more, or tree a subdirectory]\n[1 directory not expanded; use a larger limit, or tree a subdirectory]")
	wantError(t, runIn(t, Tree(), dir, `{"path":"z.txt"}`), "tree z.txt: not a directory")
}

// A search path outside the roots is reported, so the call needs approval.
func TestSearchOutside(t *testing.T) {
	root, other := t.TempDir(), t.TempDir()
	env := tool.Env{Cwd: root, Roots: []string{root}}
	for _, tl := range []tool.Tool{Grep(), Find(), Tree()} {
		args := fmt.Sprintf(`{"pattern":"x","path":%q}`, other)
		c, ok := tl.(tool.Confined)
		if !ok {
			t.Fatalf("%s is not Confined", tl.Spec().Name)
		}
		if got := c.Outside(tool.Call{Name: tl.Spec().Name, Args: jsontext.Value(args)}, env); len(got) != 1 {
			t.Errorf("%s: Outside = %v, want the other directory", tl.Spec().Name, got)
		}
	}
}
