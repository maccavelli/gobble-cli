package fsx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// tree is a workspace root with a file, a sibling directory outside it
// holding a secret, and a symlink in the root that points at the sibling.
type tree struct {
	root, outside, link string
}

func newTree(t *testing.T) tree {
	t.Helper()
	base := t.TempDir()
	tr := tree{root: filepath.Join(base, "work"), outside: filepath.Join(base, "elsewhere")}
	for _, d := range []string{tr.root, tr.outside} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	write(t, filepath.Join(tr.root, "in.txt"), "inside\n")
	write(t, filepath.Join(tr.outside, "secret.txt"), "secret\n")
	tr.link = filepath.Join(tr.root, "link")
	if err := os.Symlink(tr.outside, tr.link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	return tr
}

func write(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestOutside(t *testing.T) {
	tr := newTree(t)
	w := Workspace{Roots: []string{tr.root}}
	cases := []struct {
		path    string
		outside bool
	}{
		{filepath.Join(tr.root, "in.txt"), false},
		{filepath.Join(tr.root, "new", "dir", "x.txt"), false},
		{tr.root, false},
		{filepath.Join(tr.root, "..", "elsewhere", "secret.txt"), true},
		{filepath.Join(tr.outside, "secret.txt"), true},
		{filepath.Join(tr.link, "secret.txt"), true},
		{filepath.Join(tr.link, "not-yet.txt"), true},
	}
	for _, tc := range cases {
		if got := w.Outside(tc.path); got != tc.outside {
			t.Errorf("Outside(%s) = %v, want %v", tc.path, got, tc.outside)
		}
	}
	if (Workspace{}).Outside(filepath.Join(tr.outside, "secret.txt")) {
		t.Error("a workspace with no roots confines nothing")
	}
}

func TestOutsideFoldsCaseOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("case folding is Windows'")
	}
	tr := newTree(t)
	w := Workspace{Roots: []string{strings.ToUpper(tr.root)}}
	if w.Outside(filepath.Join(tr.root, "in.txt")) {
		t.Fatal("a root spelled in another case does not contain its own file")
	}
}

func TestAdditionalRoot(t *testing.T) {
	tr := newTree(t)
	w := Workspace{Roots: []string{tr.root, tr.outside}}
	b, err := w.ReadFile(filepath.Join(tr.outside, "secret.txt"))
	if err != nil || string(b) != "secret\n" {
		t.Fatalf("read in an additional root = %q, %v", b, err)
	}
}

// Without approval a path outside the roots is refused, also through a
// symlink inside a root; with it, it is read.
func TestReadConfined(t *testing.T) {
	tr := newTree(t)
	w := Workspace{Roots: []string{tr.root}}
	if b, err := w.ReadFile(filepath.Join(tr.root, "in.txt")); err != nil || string(b) != "inside\n" {
		t.Fatalf("read inside = %q, %v", b, err)
	}
	for _, p := range []string{
		filepath.Join(tr.outside, "secret.txt"),
		filepath.Join(tr.root, "..", "elsewhere", "secret.txt"),
		filepath.Join(tr.link, "secret.txt"),
	} {
		b, err := w.ReadFile(p)
		if !errors.Is(err, ErrOutside) || len(b) != 0 {
			t.Errorf("read %s = %q, %v; want ErrOutside", p, b, err)
		}
	}
	w.AllowOutside = true
	if b, err := w.ReadFile(filepath.Join(tr.link, "secret.txt")); err != nil || string(b) != "secret\n" {
		t.Fatalf("approved read through the link = %q, %v", b, err)
	}
}

// The os.Root is what refuses a symlink escape: asked to open a path
// relative to the root that leaves it through the link, it fails, even
// when Outside is not consulted.
func TestRootRefusesEscape(t *testing.T) {
	tr := newTree(t)
	root, rel, err := Workspace{Roots: []string{tr.root}}.open(filepath.Join(tr.root, "in.txt"))
	if err != nil || rel != "in.txt" {
		t.Fatalf("open = %q, %v", rel, err)
	}
	defer root.Close() //nolint:errcheck // test
	if b, err := root.ReadFile(filepath.Join("link", "secret.txt")); err == nil {
		t.Fatalf("the root read %q through the link; want a refusal", b)
	}
}

func TestWriteFile(t *testing.T) {
	tr := newTree(t)
	w := Workspace{Roots: []string{tr.root}}
	nested := filepath.Join(tr.root, "a", "b", "new.txt")
	if err := w.WriteFile(nested, []byte("new\n")); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(nested); string(b) != "new\n" { //nolint:gosec // test file
		t.Fatalf("content = %q", b)
	}
	if err := w.WriteFile(filepath.Join(tr.link, "planted.txt"), []byte("x")); !errors.Is(err, ErrOutside) {
		t.Fatalf("write through the link = %v, want ErrOutside", err)
	}
	if _, err := os.Stat(filepath.Join(tr.outside, "planted.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the refused write left a file: %v", err)
	}
	entries, _ := os.ReadDir(filepath.Join(tr.root, "a", "b"))
	if len(entries) != 1 {
		t.Fatalf("directory holds %d entries, want only new.txt (no temporary file left)", len(entries))
	}
}

func TestWriteKeepsMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no Unix permission bits to keep")
	}
	tr := newTree(t)
	p := filepath.Join(tr.root, "run.sh")
	write(t, p, "#!/bin/sh\n")
	if err := os.Chmod(p, 0o755); err != nil { //nolint:gosec // test: an executable to keep executable
		t.Fatal(err)
	}
	if err := (Workspace{Roots: []string{tr.root}}).WriteFile(p, []byte("#!/bin/sh\necho\n")); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(p); info.Mode().Perm() != 0o755 {
		t.Fatalf("mode = %v, want 0755 kept", info.Mode().Perm())
	}
	fresh := filepath.Join(tr.root, "fresh.txt")
	if err := (Workspace{Roots: []string{tr.root}}).WriteFile(fresh, nil); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Stat(fresh); info.Mode().Perm() != newFileMode {
		t.Fatalf("new file mode = %v, want %v", info.Mode().Perm(), newFileMode)
	}
}

// A reader racing the writer sees the old content or the new, never a
// mixture or an empty file.
func TestWriteIsAtomic(t *testing.T) {
	tr := newTree(t)
	w := Workspace{Roots: []string{tr.root}}
	p := filepath.Join(tr.root, "big.txt")
	oldText, newText := bytes.Repeat([]byte("o"), 1<<20), bytes.Repeat([]byte("n"), 1<<20)
	if err := w.WriteFile(p, oldText); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	var bad []string
	var mu sync.Mutex
	go func() {
		defer close(done)
		deadline := time.Now().Add(500 * time.Millisecond)
		for time.Now().Before(deadline) {
			b, err := os.ReadFile(p) //nolint:gosec // test file
			if err != nil {
				continue // Windows refuses a read while the rename replaces the file
			}
			if !bytes.Equal(b, oldText) && !bytes.Equal(b, newText) {
				mu.Lock()
				bad = append(bad, string(b[:min(len(b), 8)]))
				mu.Unlock()
			}
		}
	}()
	for i := range 20 {
		next := newText
		if i%2 == 1 {
			next = oldText
		}
		if err := w.WriteFile(p, next); err != nil {
			t.Fatal(err)
		}
	}
	<-done
	if len(bad) > 0 {
		t.Fatalf("a reader saw %d partial files, starting %q", len(bad), bad[0])
	}
}
