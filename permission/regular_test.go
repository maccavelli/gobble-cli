package permission

import (
	"errors"
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

// permissions.json and the trust file each refuse a FIFO at once, naming
// the file; a missing one is empty, as before (0037-MADR D10).
func TestStoresRefuseFIFO(t *testing.T) {
	stores := map[string]func(path string) (int, error){
		PermissionsFile: func(path string) (int, error) {
			m, err := RuleStore{Path: path}.Load()
			return len(m), err
		},
		TrustFile: func(path string) (int, error) {
			m, err := TrustStore{Path: path}.Load()
			return len(m), err
		},
	}
	for name, load := range stores {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			if n, err := load(filepath.Join(dir, "missing.json")); n != 0 || err != nil {
				t.Errorf("Load of a missing file = %d, %v; want none, nil", n, err)
			}
			path := filepath.Join(dir, name)
			fifo(t, path)
			var err error
			within(t, time.Second, func() { _, err = load(path) })
			if !errors.Is(err, fsx.ErrNotRegular) || !strings.HasPrefix(err.Error(), name+": ") {
				t.Errorf("Load of a FIFO = %v, want fsx.ErrNotRegular naming %s", err, name)
			}
		})
	}
}
