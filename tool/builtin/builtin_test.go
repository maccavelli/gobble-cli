package builtin

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if got := strings.Join(Names(), ","); got != "read,write,edit,bash" {
		t.Fatalf("names %s", got)
	}
	for _, tl := range Tools() {
		s := tl.Spec()
		readOnly := s.Name == "read"
		if s.Annotations.ReadOnlyHint != readOnly || s.Annotations.DestructiveHint == readOnly {
			t.Errorf("%s annotations %+v: only read is read-only, the rest destructive", s.Name, s.Annotations)
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
	if r.IsError || len(r.Diffs) != 1 || r.Diffs[0].OldText != nil || r.Diffs[0].Path != filepath.Join(dir, "sub", "a.txt") {
		t.Fatalf("write = %+v", r)
	}
	if r = call(t, Read(), env, map[string]string{"path": "sub/a.txt"}); r.Text() != "one\ntwo\n" || r.Summary != "read sub/a.txt (2 lines)" {
		t.Fatalf("read = %+v, text %q", r, r.Text())
	}
	r = call(t, Edit(), env, map[string]string{"path": "sub/a.txt", "oldText": "two", "newText": "2"})
	if r.IsError || len(r.Diffs) != 1 || *r.Diffs[0].OldText != "one\ntwo\n" || r.Diffs[0].NewText != "one\n2\n" {
		t.Fatalf("edit = %+v", r)
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
		{"x", "y"}: "occurs 2 times",
		{"z", "y"}: "was not found",
		{"", "y"}:  "oldText is empty",
	} {
		r := call(t, Edit(), env, map[string]string{"path": "a", "oldText": args[0], "newText": args[1]})
		if !r.IsError || !strings.Contains(r.Text(), want) {
			t.Errorf("edit %q: %+v, want an error with %q", args, r, want)
		}
	}
	if r := call(t, Read(), env, map[string]string{"path": "missing"}); !r.IsError || !strings.Contains(r.Text(), "read missing:") {
		t.Errorf("read of a missing file = %+v", r)
	}
}

func TestBounds(t *testing.T) {
	var b strings.Builder
	for range 2500 {
		b.WriteString("line\n")
	}
	if got := head(b.String()); !strings.HasSuffix(got, "[truncated: showing the first 2000 of 2500 lines]\n") || strings.Count(got, "line\n") != 2000 {
		t.Fatalf("head kept %d lines", strings.Count(got, "line\n"))
	}
	if got := tail(b.String()); !strings.HasPrefix(got, "[truncated: showing the last 2000 of 2500 lines]\n") {
		t.Fatalf("tail = %.60q", got)
	}
	big := strings.Repeat(strings.Repeat("x", 99)+"\n", 1000) // 100 KB
	if got := head(big); len(got) > maxBytes+100 {
		t.Fatalf("head kept %d bytes", len(got))
	}
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

func TestBash(t *testing.T) {
	if _, err := findBash(); err != nil {
		t.Skipf("no usable bash: %v", err)
	}
	dir := t.TempDir()
	env := tool.Env{Cwd: dir}
	r := call(t, Bash(), env, map[string]string{"command": "echo hi; pwd"})
	if r.IsError || !strings.Contains(r.Text(), "hi\n") || !strings.HasSuffix(r.Text(), "[exit status 0]") || !strings.HasPrefix(r.Summary, "exit status 0: hi") {
		t.Fatalf("bash = %+v, text %q", r, r.Text())
	}
	if r = call(t, Bash(), env, map[string]string{"command": "echo no >&2; exit 3"}); !r.IsError || !strings.Contains(r.Text(), "no\n[exit status 3]") {
		t.Fatalf("failing bash = %+v, text %q", r, r.Text())
	}
	if r = call(t, Bash(), env, map[string]any{"command": "sleep 30", "timeout": 1}); !r.IsError || !strings.Contains(r.Text(), "killed after 1 s") {
		t.Fatalf("timed-out bash = %+v", r)
	}
}

// A cancelled context kills the command and ends the call at once.
func TestBashCancel(t *testing.T) {
	if _, err := findBash(); err != nil {
		t.Skipf("no usable bash: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)
	start := time.Now()
	_, err := Bash().Run(ctx, tool.Call{Name: "bash", Args: jsontext.Value(`{"command":"sleep 30"}`)}, tool.Env{Cwd: t.TempDir()})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("cancel took %v", d)
	}
}
