package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// Only a sharing violation is retried on Windows: access denied, a missing
// file and a plain error are not (0067-MADR R3, D1).
func TestSharingViolationPredicate(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{windows.ERROR_SHARING_VIOLATION, true},
		{&fs.PathError{Op: "open", Path: "x", Err: windows.ERROR_SHARING_VIOLATION}, true},
		{windows.ERROR_ACCESS_DENIED, false},
		{windows.ERROR_FILE_NOT_FOUND, false},
		{errors.New("plain"), false},
	}
	for _, c := range cases {
		if got := sharingViolation(c.err); got != c.want {
			t.Errorf("sharingViolation(%v) = %v; want %v", c.err, got, c.want)
		}
	}
}

// holdForReplace opens p as a replacing rename does, with DELETE access
// and every share, so os.Open, which does not share delete, is refused
// with a sharing violation while it is held (0067-MADR probe i).
func holdForReplace(t *testing.T, p string) windows.Handle {
	t.Helper()
	name, err := windows.UTF16PtrFromString(p)
	if err != nil {
		t.Fatal(err)
	}
	h, err := windows.CreateFile(name, windows.DELETE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	return h
}

// A read refused by a replace in the middle of its move succeeds once the
// holder lets go: the open's first refusal closes the holder, so no sleep
// and no second goroutine decide the outcome (0067-MADR R1, R9, R10).
func TestReadRegularFileSurvivesAHeldHandle(t *testing.T) {
	p := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(p, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	h := holdForReplace(t, p)
	released := false
	t.Cleanup(func() {
		if !released {
			windows.CloseHandle(h) //nolint:errcheck,gosec // the test's own handle
		}
	})
	var refusal error
	restore := SetOpenFile(func(name string, flag int, perm fs.FileMode) (*os.File, error) {
		f, err := os.OpenFile(name, flag, perm) //nolint:gosec // G304: the test's temp file
		if err != nil && !released {
			refusal = err
			released = true
			if cerr := windows.CloseHandle(h); cerr != nil {
				t.Errorf("close the holder: %v", cerr)
			}
		}
		return f, err
	})
	t.Cleanup(restore)
	data, err := bounded2(t, func() ([]byte, error) { return ReadRegularFile(p, ConfigLimit) })
	if !released || !errors.Is(refusal, windows.ERROR_SHARING_VIOLATION) {
		t.Fatalf("the first open was not refused with a sharing violation (%v): the holder does not hold", refusal)
	}
	if err != nil || string(data) != "hello" {
		t.Fatalf("ReadRegularFile under a held handle = %q, %v; want the content once the holder lets go", data, err)
	}
}

// bounded2 is bounded for a call that returns a value.
func bounded2(t *testing.T, f func() ([]byte, error)) ([]byte, error) {
	t.Helper()
	var data []byte
	err := bounded(t, 10*time.Second, func() error {
		var err error
		data, err = f()
		return err
	})
	return data, err
}
