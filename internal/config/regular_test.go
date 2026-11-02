package config

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

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// fifo makes a FIFO at p with mkfifo(1), or skips where the platform
// cannot make one (0037-PLAN C10).
func fifo(t *testing.T, p string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no FIFOs")
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.CommandContext(t.Context(), "mkfifo", p).CombinedOutput(); err != nil { //nolint:gosec // G204: a test's temporary path
		t.Skipf("cannot make a FIFO here: %v %s", err, out)
	}
}

// within runs f, failing the test when it has not returned by deadline.
func within(t *testing.T, deadline time.Duration, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
	case <-time.After(deadline):
		t.Fatalf("still blocked after %v", deadline)
	}
}

// refused is the diagnostic about file whose text holds reason, or fails.
func refused(t *testing.T, ds []Diagnostic, file, reason string) {
	t.Helper()
	for _, d := range ds {
		if d.File == file && d.Severity == SeverityError && strings.Contains(d.Text, reason) {
			return
		}
	}
	t.Fatalf("diagnostics %v, want an error about %s naming %q", ds, file, reason)
}

// A FIFO as the untrusted project's settings.yaml is refused at once, with
// a diagnostic, not read (0037-MADR D10; F9).
func TestResolveRefusesFIFO(t *testing.T) {
	l := newLayout(t, "", "")
	file := filepath.Join(l.project, ".gobble", "settings.yaml")
	fifo(t, file)
	var ds []Diagnostic
	within(t, time.Second, func() { _, ds = Resolve(Options{ConfigDir: l.config, Cwd: l.project}) })
	refused(t, ds, file, "not a regular file")
}

// A settings file over 1 MiB is refused with the limit named, before it is
// read (0037-MADR D10; F9).
func TestResolveRefusesLarge(t *testing.T) {
	line := "# " + strings.Repeat("x", 1021) + "\n"
	l := newLayout(t, strings.Repeat(line, 2048), "")
	file := filepath.Join(l.config, "settings.yaml")
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	_, ds := Resolve(Options{ConfigDir: l.config, Cwd: l.project})
	runtime.ReadMemStats(&after)
	if n := after.TotalAlloc - before.TotalAlloc; n >= 4<<20 {
		t.Errorf("Resolve of a 2 MiB settings.yaml allocated %d bytes, want under 4 MiB", n)
	}
	refused(t, ds, file, "file too large: over 1 MiB")
}

// A project settings.yaml linked to /dev/zero is refused, not read until
// memory runs out (0037-MADR D10; F9).
func TestResolveDevZero(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no /dev/zero")
	}
	l := newLayout(t, "", "")
	file := filepath.Join(l.project, ".gobble", "settings.yaml")
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("/dev/zero", file); err != nil {
		t.Skipf("cannot make a symlink here: %v", err)
	}
	var ds []Diagnostic
	within(t, time.Second, func() { _, ds = Resolve(Options{ConfigDir: l.config, Cwd: l.project}) })
	refused(t, ds, file, "not a regular file")
}

// Each settings editor refuses a FIFO at once, and a missing file is
// fs.ErrNotExist, as before (0037-PLAN P7 step 3).
func TestEditorsRefuseFIFO(t *testing.T) {
	editors := map[string]func(file string) error{
		"Update": func(file string) error { _, _, err := Update(file, false); return err },
		"Set":    func(file string) error { return Set(file, "steeringMode", "all") },
		"AppendRule": func(file string) error {
			return AppendRule(file, PermissionRule{Allow: "bash(ls)"})
		},
	}
	for name, run := range editors {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if err := run(filepath.Join(dir, "missing.yaml")); !errors.Is(err, fs.ErrNotExist) {
				t.Errorf("%s of a missing file = %v, want fs.ErrNotExist", name, err)
			}
			file := filepath.Join(dir, "settings.yaml")
			fifo(t, file)
			var err error
			within(t, time.Second, func() { err = run(file) })
			if !errors.Is(err, fsx.ErrNotRegular) {
				t.Errorf("%s of a FIFO = %v, want fsx.ErrNotRegular", name, err)
			}
		})
	}
}
