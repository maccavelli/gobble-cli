package cli

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// brokenPipeError writes to a real pipe whose reader has closed, so the test
// sees what the OS reports (EPIPE on Unix, ERROR_BROKEN_PIPE or
// ERROR_NO_DATA on Windows).
func brokenPipeError(t *testing.T) error {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := w.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
			t.Error(err)
		}
	})
	_, err = w.Write([]byte("x"))
	if err == nil {
		t.Fatal("writing to a pipe with no reader succeeded")
	}
	return err
}

func TestIsBrokenPipe(t *testing.T) {
	if err := brokenPipeError(t); !isBrokenPipe(err) {
		t.Fatalf("isBrokenPipe(%v) = false for a real broken pipe", err)
	}
	if isBrokenPipe(os.ErrClosed) || isBrokenPipe(errors.New("disk full")) {
		t.Fatal("isBrokenPipe accepted an unrelated error")
	}
}

type errWriter struct{ err error }

func (w errWriter) Write([]byte) (int, error) { return 0, w.err }

// A broken stdout ends the run quietly with 0 (0008-MADR D3).
func TestBrokenStdoutExitsZero(t *testing.T) {
	t.Setenv("GOBBLE_HOME", t.TempDir())
	var errw bytes.Buffer
	code := Main(t.Context(), []string{"version"}, nil, errWriter{brokenPipeError(t)}, &errw)
	if code != ExitOK || errw.Len() != 0 {
		t.Fatalf("exit %d, stderr %q; want 0 and nothing", code, errw.String())
	}
	code = Main(t.Context(), []string{"version"}, nil, errWriter{errors.New("disk full")}, &errw)
	if code != ExitFailure || !strings.Contains(errw.String(), "Error: write stdout: disk full") {
		t.Fatalf("other write error: exit %d, stderr %q", code, errw.String())
	}
}

func TestDiagnosticPrefixes(t *testing.T) {
	var out, errw bytes.Buffer
	o := &Output{out: &out, err: &errw}
	o.Errorf("bad %s", "thing")
	o.Warnf("careful")
	if got := errw.String(); got != "Error: bad thing\nWarning: careful\n" {
		t.Fatalf("stderr = %q", got)
	}
	if out.Len() != 0 {
		t.Fatalf("diagnostics reached stdout: %q", out.String())
	}
}
