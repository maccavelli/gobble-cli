package filelock

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// HeldError is TryLock's error when another process holds the lock. PID is
// the holder's, or 0 when its lock file does not say.
type HeldError struct{ PID int }

func (e *HeldError) Error() string { return fmt.Sprintf("locked by pid %d", e.PID) }

// TryLock takes the exclusive lock on path, creating the file 0600, and
// writes this process's pid into it (0008-MADR D19 item 8). It never waits:
// a lock another process holds is a *HeldError. The lock is advisory and
// held until release is called.
func TryLock(path string) (release func() error, err error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600) //nolint:gosec // G304: a lock file the caller names
	if err != nil {
		return nil, err
	}
	ok, err := lockFile(f)
	if err != nil {
		return nil, errors.Join(err, f.Close())
	}
	if !ok {
		pid := readPID(f)
		return nil, errors.Join(&HeldError{PID: pid}, f.Close())
	}
	if err := writePID(f); err != nil {
		return nil, errors.Join(err, unlockFile(f), f.Close())
	}
	return func() error { return errors.Join(unlockFile(f), f.Close()) }, nil
}

// Lock is TryLock tried up to attempts times, wait apart, while another
// process holds the lock: Pi's 10 tries 20 ms apart, for an edit of a
// settings file (0018-MADR D18). After the last try it returns that try's
// *HeldError; when ctx ends first, ctx's error. Fewer than one attempt is
// one.
func Lock(ctx context.Context, path string, attempts int, wait time.Duration) (release func() error, err error) {
	for i := 1; ; i++ {
		release, err := TryLock(path)
		var held *HeldError
		if !errors.As(err, &held) || i >= attempts {
			return release, err
		}
		t := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		case <-t.C:
		}
	}
}

func readPID(f *os.File) int {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0
	}
	b, err := io.ReadAll(io.LimitReader(f, 32))
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(b))) //nolint:errcheck // an unreadable pid is reported as 0
	return pid
}

func writePID(f *os.File) error {
	if err := f.Truncate(0); err != nil {
		return err
	}
	_, err := f.WriteAt([]byte(strconv.Itoa(os.Getpid())+"\n"), 0)
	return err
}
