package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// symlink makes link point at target, or skips where the platform cannot
// (0037-PLAN C10).
func symlink(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
}

// isSymlink fails the test unless p is still a symlink.
func isSymlink(t *testing.T, p string) {
	t.Helper()
	info, err := os.Lstat(p)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("%s is no longer a symlink (mode %v) after the write, want the link kept", filepath.Base(p), info.Mode())
	}
}

// A write through CLAUDE.md -> AGENTS.md writes AGENTS.md and keeps the
// link, in a workspace and outside one (0037-MADR D6).
func TestWriteThroughSymlink(t *testing.T) {
	tr := newTree(t)
	agents, claude := filepath.Join(tr.root, "AGENTS.md"), filepath.Join(tr.root, "CLAUDE.md")
	write(t, agents, "old\n")
	symlink(t, "AGENTS.md", claude)
	if err := (Workspace{Roots: []string{tr.root}}).WriteFile(claude, []byte("new\n")); err != nil {
		t.Fatal(err)
	}
	isSymlink(t, claude)
	if b, _ := os.ReadFile(agents); string(b) != "new\n" { //nolint:gosec // test file
		t.Fatalf("AGENTS.md = %q, want the new bytes", b)
	}

	dir := t.TempDir()
	conf, link := filepath.Join(dir, "settings.json"), filepath.Join(dir, "linked.json")
	write(t, conf, "{}\n")
	symlink(t, "settings.json", link)
	if err := WriteFileMode(link, []byte("{\"a\":1}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	isSymlink(t, link)
	if b, _ := os.ReadFile(conf); string(b) != "{\"a\":1}\n" { //nolint:gosec // test file
		t.Fatalf("settings.json = %q, want the new bytes", b)
	}
}

// A link in a root to a file outside every root is refused, and neither the
// link nor its target changes.
func TestWriteSymlinkOutsideRoot(t *testing.T) {
	tr := newTree(t)
	secret := filepath.Join(tr.outside, "secret.txt")
	link := filepath.Join(tr.root, "notes.md")
	symlink(t, secret, link)
	if err := (Workspace{Roots: []string{tr.root}}).WriteFile(link, []byte("planted\n")); !errors.Is(err, ErrOutside) {
		t.Fatalf("write through the link = %v, want ErrOutside", err)
	}
	isSymlink(t, link)
	if b, _ := os.ReadFile(secret); string(b) != "secret\n" { //nolint:gosec // test file
		t.Fatalf("the target = %q, want it unchanged", b)
	}
}

// A link whose target does not exist yet is followed too: the write creates
// the target inside the roots, refuses one outside them, and gives up on a
// loop, keeping the link each time (0037-PLAN deviation 5).
func TestWriteThroughDanglingSymlink(t *testing.T) {
	tr := newTree(t)
	w := Workspace{Roots: []string{tr.root}}
	agents, claude := filepath.Join(tr.root, "AGENTS.md"), filepath.Join(tr.root, "CLAUDE.md")
	symlink(t, "AGENTS.md", claude)
	if err := w.WriteFile(claude, []byte("new\n")); err != nil {
		t.Fatal(err)
	}
	isSymlink(t, claude)
	if b, err := os.ReadFile(agents); err != nil || string(b) != "new\n" { //nolint:gosec // test file
		t.Fatalf("AGENTS.md = %q, %v; want it made with the new bytes", b, err)
	}

	none, escape := filepath.Join(tr.outside, "none.md"), filepath.Join(tr.root, "escape.md")
	symlink(t, none, escape)
	if err := w.WriteFile(escape, []byte("planted\n")); !errors.Is(err, ErrOutside) {
		t.Fatalf("write through the dangling link out = %v, want ErrOutside", err)
	}
	isSymlink(t, escape)
	if _, err := os.Lstat(none); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("the refused write made its target: %v", err)
	}

	a, b := filepath.Join(tr.root, "a.md"), filepath.Join(tr.root, "b.md")
	symlink(t, "b.md", a)
	symlink(t, "a.md", b)
	if err := w.WriteFile(a, []byte("x")); err == nil {
		t.Fatal("write through a -> b -> a succeeded, want an error")
	}
	isSymlink(t, a)
	isSymlink(t, b)
}

// Writing one of two hard links writes the file both name, and keeps them
// one file (0037-MADR Q1 (a)).
func TestWriteHardLinkInPlace(t *testing.T) {
	tr := newTree(t)
	a, b := filepath.Join(tr.root, "a"), filepath.Join(tr.root, "b")
	write(t, a, "old\n")
	if err := os.Link(a, b); err != nil {
		t.Fatal(err)
	}
	if err := (Workspace{Roots: []string{tr.root}}).WriteFile(a, []byte("new\n")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(b); string(got) != "new\n" { //nolint:gosec // test file
		t.Fatalf("b = %q, want the new bytes", got)
	}
	ia, err := os.Stat(a)
	if err != nil {
		t.Fatal(err)
	}
	ib, err := os.Stat(b)
	if err != nil {
		t.Fatal(err)
	}
	n, err := links(osFS{}, a, ia)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(ia, ib) || n != 2 {
		t.Fatalf("after the write a and b are one file: %v, with %d names; want one file with 2", os.SameFile(ia, ib), n)
	}
}

// A file with one name is still replaced by a rename: its inode changes.
func TestWriteNewFileAtomic(t *testing.T) {
	if runtime.GOOS == goosWindows {
		t.Skip("the inode a rename changes is a Unix notion")
	}
	tr := newTree(t)
	p := filepath.Join(tr.root, "in.txt")
	before, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if err := (Workspace{Roots: []string{tr.root}}).WriteFile(p, []byte("new\n")); err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(before, after) {
		t.Fatal("in.txt kept its inode, want a new file renamed into place")
	}
	if b, _ := os.ReadFile(p); string(b) != "new\n" { //nolint:gosec // test file
		t.Fatalf("in.txt = %q, want the new bytes", b)
	}
}
