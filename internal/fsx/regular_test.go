package fsx

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fifo makes a FIFO at p with mkfifo(1), or skips where the platform
// cannot make one (0037-PLAN C10).
func fifo(t *testing.T, p string) {
	t.Helper()
	if runtime.GOOS == goosWindows {
		t.Skip("Windows has no FIFOs")
	}
	if out, err := exec.CommandContext(t.Context(), "mkfifo", p).CombinedOutput(); err != nil { //nolint:gosec // G204: a test's temporary path
		t.Skipf("cannot make a FIFO here: %v %s", err, out)
	}
}

// bounded runs f, failing the test when it has not returned by deadline.
func bounded(t *testing.T, deadline time.Duration, f func() error) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- f() }()
	select {
	case err := <-done:
		return err
	case <-time.After(deadline):
		t.Fatalf("still blocked after %v", deadline)
		return nil
	}
}

// Each regular-file reader reads a regular file, refuses a directory, and
// one over its limit with the limit named (0037-MADR D7).
func TestReadRegular(t *testing.T) {
	dir := t.TempDir()
	w := Workspace{Roots: []string{dir}}
	write(t, filepath.Join(dir, "small.txt"), "small\n")
	write(t, filepath.Join(dir, "big.txt"), strings.Repeat("b", 1025))
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	k, err := w.Walker(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close() //nolint:errcheck // read-only
	readers := map[string]func(name string) ([]byte, error){
		"Workspace.ReadRegular": func(name string) ([]byte, error) { return w.ReadRegular(filepath.Join(dir, name), 1024) },
		"Walker.ReadRegular":    func(name string) ([]byte, error) { return k.ReadRegular(name, 1024) },
		"ReadRegularFile":       func(name string) ([]byte, error) { return ReadRegularFile(filepath.Join(dir, name), 1024) },
	}
	for what, read := range readers {
		if b, err := read("small.txt"); err != nil || string(b) != "small\n" {
			t.Errorf("%s(small.txt) = %q, %v", what, b, err)
		}
		if _, err := read("sub"); !errors.Is(err, ErrNotRegular) || !errors.Is(err, fs.ErrInvalid) {
			t.Errorf("%s(sub) = %v, want ErrNotRegular, an fs.ErrInvalid", what, err)
		}
		_, err := read("big.txt")
		if !errors.Is(err, ErrTooLarge) || !errors.Is(err, fs.ErrInvalid) || !strings.Contains(err.Error(), "file too large: over 1 KiB") {
			t.Errorf("%s(big.txt) = %v, want ErrTooLarge naming the 1 KiB limit", what, err)
		}
	}
}

// Each regular-file reader refuses a FIFO at once (0037-MADR D7).
func TestReadRegularRefusesFIFO(t *testing.T) {
	dir := t.TempDir()
	fifo(t, filepath.Join(dir, "pipe"))
	w := Workspace{Roots: []string{dir}}
	k, err := w.Walker(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close() //nolint:errcheck // read-only
	readers := map[string]func() error{
		"Workspace.OpenRegular": func() error {
			f, err := w.OpenRegular(filepath.Join(dir, "pipe"))
			if err == nil {
				f.Close() //nolint:errcheck,gosec // read-only
			}
			return err
		},
		"Workspace.ReadRegular": func() error { _, err := w.ReadRegular(filepath.Join(dir, "pipe"), 1024); return err },
		"Walker.ReadRegular":    func() error { _, err := k.ReadRegular("pipe", 1024); return err },
		"ReadRegularFile":       func() error { _, err := ReadRegularFile(filepath.Join(dir, "pipe"), 1024); return err },
	}
	for what, read := range readers {
		if err := bounded(t, 2*time.Second, read); !errors.Is(err, ErrNotRegular) {
			t.Errorf("%s(pipe) = %v, want ErrNotRegular", what, err)
		}
	}
}
