package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"testing"
	"time"
)

// A rename within a root is one; across roots, or when the rename fails as
// it does across devices, it is ErrCrossDevice, for the caller to copy.
func TestRename(t *testing.T) {
	base := t.TempDir()
	a, b := filepath.Join(base, "a"), filepath.Join(base, "b")
	mkdir(t, a)
	mkdir(t, b)
	w := Workspace{Roots: []string{a, b}}
	write(t, filepath.Join(a, "f"), "x")
	if err := w.Rename(filepath.Join(a, "f"), filepath.Join(a, "g")); err != nil {
		t.Fatalf("a rename within a root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(a, "g")); err != nil {
		t.Fatalf("the renamed file is missing: %v", err)
	}
	if err := w.Rename(filepath.Join(a, "g"), filepath.Join(b, "g")); !errors.Is(err, ErrCrossDevice) {
		t.Fatalf("a rename across roots gave %v, want ErrCrossDevice", err)
	}
	saved := rootRename
	defer func() { rootRename = saved }()
	rootRename = func(*os.Root, string, string) error { return &os.LinkError{Op: "rename", Err: syscall.EXDEV} }
	if err := w.Rename(filepath.Join(a, "g"), filepath.Join(a, "h")); !errors.Is(err, ErrCrossDevice) {
		t.Fatalf("a rename failing with EXDEV gave %v, want ErrCrossDevice", err)
	}
}

// CopyFile copies the bytes and the permission bits.
func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	w := Workspace{Roots: []string{dir}}
	from, to := filepath.Join(dir, "run.sh"), filepath.Join(dir, "sub", "run.sh")
	write(t, from, "#!/bin/sh\n")
	if err := os.Chmod(from, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := w.CopyFile(from, to); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(to); err != nil || string(b) != "#!/bin/sh\n" {
		t.Fatalf("the copy holds %q, %v", b, err)
	}
	if runtime.GOOS == "windows" {
		return // Windows has no Unix permission bits to keep
	}
	if info, err := os.Stat(to); err != nil || info.Mode().Perm() != 0o750 {
		t.Fatalf("the copy's mode is %v, %v; want 0750", info.Mode().Perm(), err)
	}
}

// CopyLink copies a link's target text without following it.
func TestCopyLink(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink("missing-target", filepath.Join(dir, "link")); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	w := Workspace{Roots: []string{dir}}
	if err := w.CopyLink(filepath.Join(dir, "link"), filepath.Join(dir, "copy")); err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(filepath.Join(dir, "copy")); err != nil || target != "missing-target" {
		t.Fatalf("the copy points at %q, %v", target, err)
	}
}

// MkdirAll, Remove and RemoveAll, through the root.
func TestDirectoryOps(t *testing.T) {
	dir := t.TempDir()
	w := Workspace{Roots: []string{dir}}
	deep := filepath.Join(dir, "a", "b", "c")
	if err := w.MkdirAll(deep); err != nil {
		t.Fatal(err)
	}
	if err := w.Remove(filepath.Join(dir, "a")); err == nil {
		t.Fatal("Remove of a directory that is not empty succeeded")
	}
	if err := w.RemoveAll(filepath.Join(dir, "a")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a is still there: %v", err)
	}
	if err := w.MkdirAll(filepath.Join(t.TempDir(), "x")); !errors.Is(err, ErrOutside) {
		t.Fatalf("an outside directory gave %v, want ErrOutside", err)
	}
}

// IsRoot is a root or a directory above one; Inside is a path or under it.
func TestIsRootAndInside(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "work")
	w := Workspace{Roots: []string{root}}
	for _, c := range []struct {
		abs  string
		want bool
	}{{root, true}, {base, true}, {filepath.Join(root, "sub"), false}} {
		if got := w.IsRoot(c.abs); got != c.want {
			t.Errorf("IsRoot(%q) = %v, want %v", c.abs, got, c.want)
		}
	}
	if !Inside(root, filepath.Join(root, "a")) || !Inside(root, root) || Inside(filepath.Join(root, "a"), root) {
		t.Error("Inside is wrong")
	}
}

// Two callers locking one pair in opposite orders both finish.
func TestLockPair(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a"), filepath.Join(dir, "b")
	var wg sync.WaitGroup
	for i := range 200 {
		wg.Go(func() {
			x, y := a, b
			if i%2 == 1 {
				x, y = b, a
			}
			unlock := LockPair(x, y)
			time.Sleep(time.Microsecond)
			unlock()
		})
	}
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("LockPair deadlocked")
	}
	unlock := LockPair(a, a) // the same path twice is one lock
	unlock()
}
