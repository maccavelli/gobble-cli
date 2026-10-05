//go:build windows

package term

import (
	"os"
	"testing"
)

// A pipe is not a console: PrepareConsole skips it and reports no error.
// The live console check is internal/cli/editor's conhost suite.
func TestPrepareConsolePipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
		if err := w.Close(); err != nil {
			t.Error(err)
		}
	})
	restore, err := PrepareConsole(w, w)
	if err != nil {
		t.Fatalf("PrepareConsole(pipe) = %v, want nil", err)
	}
	restore()
	restore() // idempotent
}
