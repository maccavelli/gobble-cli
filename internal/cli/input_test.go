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
			p, err := composeInput(t.Context(), strings.NewReader(tc.stdin), tc.stdinTTY, tc.words, files)
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

func TestComposeInputImage(t *testing.T) {
	p, err := composeInput(t.Context(), nil, true, []string{"@shot.png"}, fakeFiles(map[string][]byte{"shot.png": png}))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Images) != 1 || p.Images[0].MIME != "image/png" || !filepath.IsAbs(p.Images[0].Name) {
		t.Fatalf("Images = %+v", p.Images)
	}
}

func TestComposeInputErrors(t *testing.T) {
	_, err := composeInput(t.Context(), nil, true, []string{"@missing.md"}, fakeFiles(nil))
	if e, ok := errors.AsType[*exitError](err); !ok || e.code != ExitUsage || !strings.Contains(e.msg, "@missing.md: no such file") {
		t.Fatalf("missing @path: %v", err)
	}
	_, err = composeInput(t.Context(), nil, true, []string{"@locked.md"},
		func(string) ([]byte, error) { return nil, fs.ErrPermission })
	if e, ok := errors.AsType[*exitError](err); !ok || e.code != ExitFailure {
		t.Fatalf("unreadable @path: %v", err)
	}
	_, err = composeInput(t.Context(), failingReader{}, false, nil, fakeFiles(nil))
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
	if _, err := composeInput(ctx, pr, false, nil, fakeFiles(nil)); err == nil {
		t.Fatal("a cancelled read of stdin returned no error")
	}
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("device error") }
