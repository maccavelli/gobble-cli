package builtin

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/maccavelli/gobble-cli/tool"
)

func call(t *testing.T, tl tool.Tool, env tool.Env, args any) tool.Result {
	t.Helper()
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	r, err := tl.Run(t.Context(), tool.Call{ID: "c1", Name: tl.Spec().Name, Args: jsontext.Value(b)}, env)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestToolsAndAnnotations(t *testing.T) {
	want := "read,write,edit,grep,find,tree,move,delete,copy,mkdir,bash"
	if runtime.GOOS == "windows" {
		want += ",powershell"
	}
	if got := strings.Join(Names(), ","); got != want {
		t.Fatalf("names %s, want %s", got, want)
	}
	for _, tl := range Tools() {
		s := tl.Spec()
		// Three kinds (F1c-2 deviation 1): read-only; additive, which mkdir
		// alone is (MCP: destructiveHint false is "only additive updates");
		// and destructive, every other tool.
		readOnly := s.Name == "read" || s.Name == "grep" || s.Name == "find" || s.Name == "tree"
		destructive := !readOnly && s.Name != "mkdir"
		if s.Annotations.ReadOnlyHint != readOnly || s.Annotations.DestructiveHint != destructive {
			t.Errorf("%s annotations %+v: read, grep, find and tree are read-only, mkdir is additive, the rest destructive", s.Name, s.Annotations)
		}
		if len(s.InputSchema) == 0 || s.Kind == "" {
			t.Errorf("%s spec %+v lacks a schema or a kind", s.Name, s)
		}
	}
}

func TestReadWriteEdit(t *testing.T) {
	dir := t.TempDir()
	env := tool.Env{Cwd: dir}
	r := call(t, Write(), env, map[string]string{"path": "sub/a.txt", "content": "one\ntwo\n"})
	if r.IsError || r.Text() != "wrote sub/a.txt (2 lines)" || len(r.Diffs) != 1 || r.Diffs[0].OldText != nil || r.Diffs[0].Path != filepath.Join(dir, "sub", "a.txt") {
		t.Fatalf("write = %+v, text %q", r, r.Text())
	}
	if r = call(t, Read(), env, map[string]string{"path": "sub/a.txt"}); r.Text() != "1: one\n2: two\n(end of file, 2 lines)" || r.Summary != "read sub/a.txt (2 lines)" {
		t.Fatalf("read = %+v, text %q", r, r.Text())
	}
	r = call(t, Edit(), env, map[string]any{"path": "sub/a.txt", "edits": []map[string]string{{"oldText": "two", "newText": "2"}}})
	if r.IsError || r.Text() != "edited sub/a.txt: 1 replacement" || len(r.Diffs) != 1 || *r.Diffs[0].OldText != "one\ntwo\n" || r.Diffs[0].NewText != "one\n2\n" {
		t.Fatalf("edit = %+v, text %q", r, r.Text())
	}
	b, err := os.ReadFile(filepath.Join(dir, "sub", "a.txt"))
	if err != nil || string(b) != "one\n2\n" {
		t.Fatalf("file = %q, %v", b, err)
	}
	r = call(t, Write(), env, map[string]string{"path": "sub/a.txt", "content": "x"})
	if r.Diffs[0].OldText == nil || *r.Diffs[0].OldText != "one\n2\n" {
		t.Fatalf("overwrite diff = %+v", r.Diffs)
	}
}

func TestEditNeedsOneMatch(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("x x"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := tool.Env{Cwd: dir}
	for args, want := range map[[2]string]string{
		{"x", "y"}: "edit a: oldText occurs 2 times; include more surrounding lines so it occurs once",
		{"z", "y"}: "edit a: oldText was not found; it must match the file exactly, or nearly (whitespace, quotes and indentation are forgiven)",
		{"", "y"}:  "edit a: oldText is empty",
	} {
		r := call(t, Edit(), env, map[string]any{"path": "a", "edits": []map[string]string{{"oldText": args[0], "newText": args[1]}}})
		if !r.IsError || r.Text() != want {
			t.Errorf("edit %q: %q, want the error %q", args, r.Text(), want)
		}
	}
	if r := call(t, Read(), env, map[string]string{"path": "missing"}); !r.IsError || !strings.Contains(r.Text(), "read missing:") {
		t.Errorf("read of a missing file = %+v", r)
	}
}

func TestBounds(t *testing.T) {
	if got := title(strings.Repeat("é", 100)); utf8.RuneCountInString(got) != maxTitle || !strings.HasSuffix(got, "…") {
		t.Fatalf("title has %d runes", utf8.RuneCountInString(got))
	}
	if got := summary(strings.Repeat("é", 300)); len(got) > maxSummary || !utf8.ValidString(got) {
		t.Fatalf("summary is %d bytes, valid %v", len(got), utf8.ValidString(got))
	}
}

func TestDescribeTitles(t *testing.T) {
	env := tool.Env{Cwd: filepath.Join(string(filepath.Separator), "w")}
	cases := map[tool.Tool]string{
		Read():  `{"path":"internal/cli/term.go"}`,
		Write(): `{"path":"a.go","content":""}`,
		Bash():  `{"command":"go test ./... && ` + strings.Repeat("x", 80) + `"}`,
	}
	for tl, args := range cases {
		got := tl.(tool.Describer).Describe(tool.Call{Args: jsontext.Value(args)}, env)
		if utf8.RuneCountInString(got) > maxTitle {
			t.Errorf("%s title %q is over %d runes", tl.Spec().Name, got, maxTitle)
		}
	}
	if got := Read().(tool.Describer).Describe(tool.Call{Args: jsontext.Value(`{"path":"internal/cli/term.go"}`)}, env); got != "read internal/cli/term.go" {
		t.Errorf("read title %q", got)
	}
}
