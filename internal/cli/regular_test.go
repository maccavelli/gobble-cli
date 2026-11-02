package cli

import (
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

// models.json as a FIFO is refused at once and warned, naming the reason;
// a missing one is none, quietly, as before (0037-MADR D10).
func TestModelsJSONRefusesFIFO(t *testing.T) {
	e, warnings := modelEnv(t)
	dir := t.TempDir()
	if mj := e.readModelsJSON(filepath.Join(dir, "missing.json")); mj != nil || warnings.String() != "" {
		t.Errorf("a missing models.json = %v, warnings %q; want none of either", mj, warnings)
	}
	path := filepath.Join(dir, "models.json")
	fifo(t, path)
	within(t, time.Second, func() {
		if mj := e.readModelsJSON(path); mj != nil {
			t.Errorf("a FIFO models.json = %v, want none", mj)
		}
	})
	if !strings.Contains(warnings.String(), "not a regular file") {
		t.Errorf("warnings %q, want one naming \"not a regular file\"", warnings)
	}
}

// config validate reports a FIFO auth.json at once, as a file it could not
// read; with no files it reports nothing (TestConfigValidateExits), as
// before (0037-PLAN deviation 7).
func TestConfigValidateRefusesFIFO(t *testing.T) {
	c := newCfgEnv(t)
	path := filepath.Join(c.home, "auth.json")
	fifo(t, path)
	var r result
	within(t, time.Second, func() { r = c.run("config", "validate") })
	c.want(r, ExitFailure, path+": could not read: open "+path+": not a regular file", "")
}

// Completion's models.json read returns at once on a FIFO, with no models,
// as on a missing file (0037-PLAN deviation 7).
func TestCompletionModelsJSONRefusesFIFO(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	if mj := completionModelsJSON(); mj != nil {
		t.Errorf("no models.json = %v, want none", mj)
	}
	fifo(t, filepath.Join(home, "models.json"))
	within(t, time.Second, func() {
		if mj := completionModelsJSON(); mj != nil {
			t.Errorf("a FIFO models.json = %v, want none", mj)
		}
	})
}
