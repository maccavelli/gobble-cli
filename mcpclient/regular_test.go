package mcpclient

import (
	"errors"
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

// Load and an edit refuse a FIFO as mcp.json at once; a missing file is no
// servers, and an edit creates it, as before (0037-MADR D10).
func TestConfigRefusesFIFO(t *testing.T) {
	dir := t.TempDir()
	if c := Load(filepath.Join(dir, "missing.json")); len(c.Servers) != 0 || len(c.Errors) != 0 {
		t.Errorf("Load of a missing file = %+v, want nothing", c)
	}
	created := filepath.Join(dir, "created.json")
	if _, err := Add(created, "a", []byte(`{"command": "x"}`)); err != nil {
		t.Errorf("Add to a missing file = %v", err)
	} else if _, err := os.Stat(created); err != nil {
		t.Errorf("Add to a missing file made none: %v", err)
	}
	path := filepath.Join(dir, "mcp.json")
	fifo(t, path)
	var c Config
	within(t, time.Second, func() { c = Load(path) })
	if len(c.Errors) != 1 || !strings.Contains(c.Errors[0], "not a regular file") {
		t.Errorf("Load of a FIFO: errors %q, want one naming \"not a regular file\"", c.Errors)
	}
	var err error
	within(t, time.Second, func() { _, err = Add(path, "a", []byte(`{"command": "x"}`)) })
	if !errors.Is(err, fsx.ErrNotRegular) {
		t.Errorf("Add to a FIFO = %v, want fsx.ErrNotRegular", err)
	}
}
