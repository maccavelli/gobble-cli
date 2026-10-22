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

// On a terminal, gobble's messages wrap at spaces to stderr's width, at
// most 120 columns, continued two columns in; off a terminal each is one
// line, as it was (0028-MADR D6).
func TestMessagesWrapOnATerminal(t *testing.T) {
	warning := "no model credential: set one of " + strings.Repeat("SOME_PROVIDER_API_KEY, ", 8) + "or sign in"
	for _, tc := range []struct {
		name  string
		tty   bool
		width int
		want  int // the widest row; 0 is one line
	}{
		{"a terminal 60 wide", true, 60, 60},
		{"a terminal 200 wide", true, 200, 120},
		{"not a terminal", false, 60, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var errw bytes.Buffer
			o := &Output{out: &bytes.Buffer{}, err: &errw}
			o.caps.Err.TTY, o.caps.Err.Width = tc.tty, tc.width
			o.Warnf("%s", warning)
			o.Errorf("%s", warning)
			got := errw.String()
			if tc.want == 0 {
				if got != "Warning: "+warning+"\nError: "+warning+"\n" {
					t.Errorf("stderr %q, want each message on one line", got)
				}
				return
			}
			rows := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
			widest := 0
			for _, r := range rows {
				widest = max(widest, len(r))
				if !strings.HasPrefix(r, "Warning: ") && !strings.HasPrefix(r, "Error: ") && !strings.HasPrefix(r, "  ") {
					t.Errorf("row %q is neither a message's first nor continued two columns in", r)
				}
			}
			if widest > tc.want || widest < tc.want-25 {
				t.Errorf("rows %q: widest %d, want at most %d and close to it", rows, widest, tc.want)
			}
			if words := strings.Join(strings.Fields(got), " "); words != "Warning: "+warning+" Error: "+warning {
				t.Errorf("words %q changed", words)
			}
		})
	}
}
