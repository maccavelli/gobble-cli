package term

import (
	"os"
	"testing"
	"time"
)

// On a pipe, FlushInput fails, and at once: a pipe is no terminal, so it has
// no queue to empty (0063-PLAN P2 step 12; MADR D18, row R77). No file fails
// too.
func TestFlushInputOnAPipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = w.Close()
		_ = r.Close()
	})
	if _, err := w.WriteString("typed ahead\n"); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- FlushInput(r) }()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("FlushInput on a pipe gave no error")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("FlushInput on a pipe blocked")
	}
	if err := FlushInput(nil); err == nil {
		t.Fatal("FlushInput of no file gave no error")
	}
	// The pipe's bytes are still there: nothing was read or dropped.
	buf := make([]byte, 64)
	n, err := r.Read(buf)
	if err != nil || string(buf[:n]) != "typed ahead\n" {
		t.Fatalf("the pipe gave %q, %v after the flush", buf[:n], err)
	}
}
