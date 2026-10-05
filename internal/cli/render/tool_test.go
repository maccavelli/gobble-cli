package render

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

func TestToolHeader(t *testing.T) {
	narrow := term.Stream{Width: 24}
	cases := []struct {
		name, title, status string
		s                   term.Stream
		want                string
	}{
		{"completed", "edit internal/cli/term.go", StatusCompleted, colorOff, "▸ edit internal/cli/term.go ✓"},
		{"failed", "run go test ./...", StatusFailed, colorOff, "▸ run go test ./... ✗"},
		{"running", "read file", StatusInProgress, colorOff, "▸ read file …"},
		{"no status", "read file", "", colorOff, "▸ read file"},
		{"truncated to the width", "edit a/very/long/path/to/some/file.go", StatusCompleted, narrow, "▸ edit a/very/long/pa… ✓"},
		{"sanitised and folded", "evil\x1b[2J title\nsecond line", "", colorOff, "▸ evil title second line"},
		{"coloured", "x", StatusCompleted, colorOn, "\x1b[1m▸ \x1b[0mx \x1b[32m✓\x1b[0m"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ToolHeader(tc.title, tc.status, tc.s, term.Unicode)
			if got != tc.want {
				t.Fatalf("ToolHeader = %q, want %q", got, tc.want)
			}
			if tc.s.Width < 80 && displayWidth(got) > tc.s.Width {
				t.Fatalf("header is %d columns, wider than %d", displayWidth(got), tc.s.Width)
			}
		})
	}
}

func numbered(n int) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	return b.String()
}

func TestToolOutputGolden(t *testing.T) {
	golden(t, "tool_output_25", ToolOutput(numbered(25), false, term.Unicode))
}

func TestToolOutputBounds(t *testing.T) {
	if got := ToolOutput(numbered(20), false, term.Unicode); got != numbered(20) {
		t.Fatalf("20 lines were cut:\n%s", got)
	}
	if got := ToolOutput(numbered(25), true, term.Unicode); got != numbered(25) {
		t.Fatalf("verbose output was cut:\n%s", got)
	}
	if got := ToolOutput(numbered(21), false, term.ASCII); !strings.Contains(got, "... 1 lines hidden (--verbose shows all)\n") {
		t.Fatalf("21 lines: %q", got)
	}
	if got := ToolOutput("ok\x1b]0;title\x07\n", false, term.Unicode); got != "ok\n" {
		t.Fatalf("output not sanitised: %q", got)
	}
	if got := ToolOutput("", false, term.Unicode); got != "" {
		t.Fatalf("empty output rendered as %q", got)
	}
}

func TestDiff(t *testing.T) {
	in := "--- a/x.go\n+++ b/x.go\n@@ -1 +1 @@\n-old\n+new\n same"
	want := "\x1b[1m--- a/x.go\x1b[0m\n\x1b[1m+++ b/x.go\x1b[0m\n\x1b[36m@@ -1 +1 @@\x1b[0m\n" +
		"\x1b[31m-old\x1b[0m\n\x1b[32m+new\x1b[0m\n same"
	if got := Diff(in, colorOn); got != want {
		t.Fatalf("Diff = %q\nwant   %q", got, want)
	}
	if got := Diff(in, colorOff); got != in {
		t.Fatalf("colour off changed the diff: %q", got)
	}
}

func TestShortPath(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home", "user")
	cases := []struct{ in, want string }{
		{filepath.Join(home, "proj", "main.go"), "~/proj/main.go"},
		{home, "~"},
		{filepath.Join(filepath.Dir(home), "other", "f"), filepath.Join(filepath.Dir(home), "other", "f")},
		{filepath.Join(home+"x", "f"), filepath.Join(home+"x", "f")},
		{"relative/path", "relative/path"},
	}
	for _, tc := range cases {
		if got := ShortPath(tc.in, home); got != tc.want {
			t.Errorf("ShortPath(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if got := ShortPath(filepath.Join(home, "f"), ""); got != filepath.Join(home, "f") {
		t.Errorf("no home: %q", got)
	}
	if runtime.GOOS == "windows" {
		if got := ShortPath(`C:\Home\example\src\a.go`, `c:\home\example`); got != "~/src/a.go" {
			t.Errorf("Windows path, other case: %q", got)
		}
		if got := ShortPath(`D:\src\a.go`, `C:\Home\example`); got != `D:\src\a.go` {
			t.Errorf("other volume: %q", got)
		}
	}
}
