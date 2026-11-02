package auth

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

// auth.json as a FIFO is refused at once, naming the file; a missing one is
// empty, as before (0037-MADR D10).
func TestLoadRefusesFIFO(t *testing.T) {
	dir := t.TempDir()
	if f, err := Load(filepath.Join(dir, "missing.json")); err != nil || len(f.members) != 0 {
		t.Errorf("Load of a missing file = %v, %v; want empty, nil", f, err)
	}
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no FIFOs")
	}
	path := filepath.Join(dir, "auth.json")
	if out, err := exec.CommandContext(t.Context(), "mkfifo", path).CombinedOutput(); err != nil { //nolint:gosec // G204: a test's temporary path
		t.Skipf("cannot make a FIFO here: %v %s", err, out)
	}
	done := make(chan error, 1)
	go func() {
		_, err := Load(path)
		done <- err
	}()
	select {
	case err := <-done:
		if !errors.Is(err, fsx.ErrNotRegular) || !strings.HasPrefix(err.Error(), "auth.json: ") {
			t.Errorf("Load of a FIFO = %v, want fsx.ErrNotRegular naming auth.json", err)
		}
	case <-time.After(time.Second):
		t.Fatal("still blocked after 1s")
	}
}
