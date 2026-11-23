package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var (
	// errHeld stands for a sharing violation: the only error the loop test's
	// predicate retries.
	errHeld = errors.New("held")
	// errOther is an open error the loop does not retry.
	errOther = errors.New("other")
)

// setReadSeams replaces ReadRegularFile's open, its predicate (errHeld
// only) and its window for one test, and puts them back after it.
func setReadSeams(t *testing.T, open openFunc, window time.Duration) {
	t.Helper()
	oldOpen, oldRetriable, oldWindow := openFile, retriable, readWindow
	openFile = open
	retriable = func(err error) bool { return errors.Is(err, errHeld) }
	readWindow = window
	t.Cleanup(func() { openFile, retriable, readWindow = oldOpen, oldRetriable, oldWindow })
}

// ReadRegularFile retries an open the predicate calls retriable until the
// window passes, returns the last error then, and returns any other error
// at once, a not-exist among them (0067-MADR R1, R2, R4, R5, R6).
func TestReadRegularFileRetryLoop(t *testing.T) {
	p := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(p, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	// Each step is one call's result: an error, or nil for the real open.
	// The last step repeats. A held step's error carries its call's number,
	// so the test can tell the last error from the first.
	cases := []struct {
		name      string
		steps     []error
		window    time.Duration
		wantCalls int // 0: more than one, and the window passed
		want      error
	}{
		{"released inside the window", []error{errHeld, errHeld, nil}, 5 * time.Second, 3, nil},
		{"held past the window", []error{errHeld}, 50 * time.Millisecond, 0, errHeld},
		{"held then gone", []error{errHeld, fs.ErrNotExist}, 5 * time.Second, 2, fs.ErrNotExist},
		{"held then another error", []error{errHeld, errOther}, 5 * time.Second, 2, errOther},
		{"another error first", []error{errOther}, 5 * time.Second, 1, errOther},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calls := 0
			setReadSeams(t, func(name string, flag int, perm fs.FileMode) (*os.File, error) {
				calls++
				step := c.steps[min(calls, len(c.steps))-1]
				switch {
				case step == nil:
					return os.OpenFile(name, flag, perm) //nolint:gosec // G304: the test's temp file
				case errors.Is(step, errHeld):
					return nil, &fs.PathError{Op: "open", Path: name, Err: fmt.Errorf("%w %d", errHeld, calls)}
				}
				return nil, &fs.PathError{Op: "open", Path: name, Err: step}
			}, c.window)
			var data []byte
			start := time.Now()
			err := bounded(t, 10*time.Second, func() error {
				var err error
				data, err = ReadRegularFile(p, ConfigLimit)
				return err
			})
			elapsed := time.Since(start)
			switch {
			case c.want == nil && (err != nil || string(data) != "hello"):
				t.Errorf("ReadRegularFile = %q, %v; want the content", data, err)
			case c.want != nil && !errors.Is(err, c.want):
				t.Errorf("ReadRegularFile error = %v; want %v", err, c.want)
			}
			if c.wantCalls == 0 {
				if calls < 2 || elapsed < c.window {
					t.Errorf("gave up after %d calls and %v; want more than one call and at least the %v window", calls, elapsed, c.window)
				}
				if want := fmt.Sprintf("%v %d", errHeld, calls); err == nil || !errors.Is(err, errHeld) || !strings.HasSuffix(err.Error(), want) {
					t.Errorf("error after the window = %v; want the last one, %q", err, want)
				}
				return
			}
			if calls != c.wantCalls {
				t.Errorf("open called %d times; want %d", calls, c.wantCalls)
			}
		})
	}
}
