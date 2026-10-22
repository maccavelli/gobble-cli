//go:build unix

package term

import (
	"syscall"
	"testing"
	"time"
)

// SIGWINCH is a resize (0030-MADR D1).
func TestWatchResizeSignal(t *testing.T) {
	ch := resizeNotices(t.Context(), nil)
	if ch == nil {
		t.Fatal("no channel")
	}
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGWINCH); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatal("SIGWINCH was not noticed")
	}
}
