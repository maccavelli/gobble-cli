package cli

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// png is the smallest file http.DetectContentType reports as image/png.
var png = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0dIHDR")

func fakeFiles(files map[string][]byte) func(string) ([]byte, error) {
	return func(name string) ([]byte, error) {
		if b, ok := files[filepath.Base(name)]; ok {
			return b, nil
		}
		return nil, fs.ErrNotExist
	}
}

// A relative @path resolves against --cwd's directory; an absolute one
// stays as it is.
func TestComposeInputBase(t *testing.T) {
	base := t.TempDir()
	var asked []string
	read := func(name string) ([]byte, error) {
		asked = append(asked, name)
		return []byte("x"), nil
	}
	other := filepath.Join(t.TempDir(), "b.md")
	if _, err := composeInput(t.Context(), nil, true, []string{"@a.md", "@" + other}, base, read); err != nil {
		t.Fatal(err)
	}
	if len(asked) != 2 || asked[0] != filepath.Join(base, "a.md") || asked[1] != other {
		t.Fatalf("read %q; want %q then %q", asked, filepath.Join(base, "a.md"), other)
	}
}

func TestComposeInput(t *testing.T) {
	abs := func(n string) string {
		p, err := filepath.Abs(n)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	files := fakeFiles(map[string][]byte{"notes.md": []byte("line one\nline two\n"), "shot.png": png, "pic.jpg": []byte("not really")})
	cases := []struct {
		name     string
		stdin    string
		stdinTTY bool
		words    []string
		want     string
		images   int
	}{
		// The Pi join("") defect, fixed: parts are joined by a blank line.
		{"stdin then words", "hi\n", false, []string{"ask"}, "hi\n\nask", 0},
		{"words joined by single spaces", "", true, []string{"fix", "the", "test"}, "fix the test", 0},
		{"stdin is trimmed", "  \n hi \n\n", false, nil, "hi", 0},
		{"a terminal stdin is not read", "ignored", true, []string{"x"}, "x", 0},
		{"empty stdin is skipped", "", false, []string{"x"}, "x", 0},
		{"@path is wrapped", "", true, []string{"@notes.md", "summarise"},
			"<file name=\"" + strings.ReplaceAll(abs("notes.md"), `\`, `\\`) + "\">\nline one\nline two\n</file>\n\nsummarise", 0},
		{"image by content", "", true, []string{"@shot.png", "what is this"}, "what is this", 1},
		{"image by extension", "", true, []string{"@pic.jpg"}, "", 1},
		{"a lone @ is a word", "", true, []string{"@", "x"}, "@ x", 0},
		{"all three parts in order", "ctx", false, []string{"q", "@notes.md"},
			"ctx\n\n<file name=\"" + strings.ReplaceAll(abs("notes.md"), `\`, `\\`) + "\">\nline one\nline two\n</file>\n\nq", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := composeInput(t.Context(), strings.NewReader(tc.stdin), tc.stdinTTY, tc.words, "", files)
			if err != nil {
				t.Fatal(err)
			}
			if p.Text != tc.want {
				t.Fatalf("Text = %q\nwant   %q", p.Text, tc.want)
			}
			if len(p.Images) != tc.images {
				t.Fatalf("Images = %d, want %d", len(p.Images), tc.images)
			}
		})
	}
}

// The composed input keeps its stdin and attached files apart from its
// words, and the client sends them first, marked so the agent does not
// scan them for @path, then the words, an @ inside a quoted argument
// included; a typed line goes as it is (0005-PLAN F3-4 deviation 1).
func TestComposeInputContext(t *testing.T) {
	files := fakeFiles(map[string][]byte{"notes.md": []byte("see @README.md\n")})
	p, err := composeInput(t.Context(), strings.NewReader("piped @x\n"), false, []string{"explain @main.go", "@notes.md"}, "", files)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Context) != 2 || p.Context[0] != "piped @x" || !strings.HasSuffix(p.Context[1], ">\nsee @README.md\n</file>") || p.Words != "explain @main.go" {
		t.Fatalf("Context %q, Words %q", p.Context, p.Words)
	}
	if want := strings.Join(append(append([]string{}, p.Context...), p.Words), "\n\n"); p.Text != want {
		t.Fatalf("Text = %q\nwant   %q", p.Text, want)
	}
	cp := clientPrompt(p)
	if len(cp.Context) != 2 || cp.Context[1] != p.Context[1] || cp.Text != "explain @main.go" {
		t.Fatalf("client prompt %+v", cp)
	}
	if typed := clientPrompt(Prompt{Text: "fix @a.go"}); typed.Context != nil || typed.Text != "fix @a.go" {
		t.Fatalf("typed line %+v", typed)
	}
	if words, err := composeInput(t.Context(), nil, true, []string{"just", "words"}, "", files); err != nil ||
		clientPrompt(words).Context != nil || clientPrompt(words).Text != "just words" {
		t.Fatalf("words alone: %+v, %v", words, err)
	}
}

func TestComposeInputImage(t *testing.T) {
	p, err := composeInput(t.Context(), nil, true, []string{"@shot.png"}, "", fakeFiles(map[string][]byte{"shot.png": png}))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Images) != 1 || p.Images[0].MIME != "image/png" || !filepath.IsAbs(p.Images[0].Name) {
		t.Fatalf("Images = %+v", p.Images)
	}
}

func TestComposeInputErrors(t *testing.T) {
	_, err := composeInput(t.Context(), nil, true, []string{"@missing.md"}, "", fakeFiles(nil))
	if e, ok := errors.AsType[*exitError](err); !ok || e.code != ExitUsage || !strings.Contains(e.msg, "@missing.md: no such file") {
		t.Fatalf("missing @path: %v", err)
	}
	_, err = composeInput(t.Context(), nil, true, []string{"@locked.md"}, "",
		func(string) ([]byte, error) { return nil, fs.ErrPermission })
	if e, ok := errors.AsType[*exitError](err); !ok || e.code != ExitFailure {
		t.Fatalf("unreadable @path: %v", err)
	}
	_, err = composeInput(t.Context(), failingReader{}, false, nil, "", fakeFiles(nil))
	if e, ok := errors.AsType[*exitError](err); !ok || e.code != ExitFailure || !strings.Contains(e.msg, "read stdin") {
		t.Fatalf("failed stdin read must be an error, not a panic: %v", err)
	}
}

func TestComposeInputCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	pr, pw := io.Pipe()
	t.Cleanup(func() {
		if err := pw.Close(); err != nil {
			t.Error(err)
		}
	})
	if _, err := composeInput(ctx, pr, false, nil, "", fakeFiles(nil)); err == nil {
		t.Fatal("a cancelled read of stdin returned no error")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("device error") }
