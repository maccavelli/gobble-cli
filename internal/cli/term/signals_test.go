package term

import (
	"os"
	"runtime"
	"slices"
	"testing"
)

// Interrupt everywhere; SIGTERM and SIGHUP only where the platform delivers
// them (0008-MADR D10).
func TestShutdownSignals(t *testing.T) {
	got := ShutdownSignals()
	if !slices.Contains(got, os.Interrupt) {
		t.Fatalf("ShutdownSignals() = %v, want os.Interrupt in it", got)
	}
	want := 3
	if runtime.GOOS == "windows" {
		want = 1
	}
	if len(got) != want {
		t.Fatalf("ShutdownSignals() = %v, want %d signals on %s", got, want, runtime.GOOS)
	}
}
