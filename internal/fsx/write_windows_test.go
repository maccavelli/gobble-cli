package fsx

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// A replace refused for the whole window, by a reader holding the target
// with os.Open, gives up after the window with the refusal, and leaves the
// temporary file to its caller (0067-MADR X2, D4; 0067-PLAN deviation 1).
func TestReplaceFileGivesUp(t *testing.T) {
	dir := t.TempDir()
	name, tmp := filepath.Join(dir, "auth.json"), filepath.Join(dir, ".auth.json.tmp")
	for _, p := range []string{name, tmp} {
		if err := os.WriteFile(p, []byte(filepath.Base(p)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	old := renameWindow
	renameWindow = 50 * time.Millisecond
	t.Cleanup(func() { renameWindow = old })
	reader, err := os.Open(name) //nolint:gosec // G304: the test's temp file
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close() //nolint:errcheck // read-only
	start := time.Now()
	err = bounded(t, 10*time.Second, func() error { return ReplaceFile(tmp, name) })
	elapsed := time.Since(start)
	if !errors.Is(err, windows.ERROR_ACCESS_DENIED) {
		t.Fatalf("ReplaceFile under a reader = %v; want Access is denied", err)
	}
	if elapsed < renameWindow {
		t.Errorf("ReplaceFile gave up after %v; want at least the %v window", elapsed, renameWindow)
	}
	if _, err := os.Stat(tmp); err != nil {
		t.Errorf("the temporary file: %v; want it left for the caller", err)
	}
}
