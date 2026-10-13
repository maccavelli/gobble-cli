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

// Policy is how Lock waits for a lock another process holds: it tries
// First apart, doubling the gap up to Max, until Deadline has passed since
// its first try. A First of zero or less is one try.
type Policy struct{ First, Max, Deadline time.Duration }

// EditPolicy is for an edit of a settings file or mcp.json: Pi's first
// tries, 20 ms apart, backing off to 100 ms, for up to 2 s (0018-MADR D18,
// amended 2026-10-12: Pi's 10 tries, about 200 ms, were outlasted under
// load).
var EditPolicy = Policy{First: 20 * time.Millisecond, Max: 100 * time.Millisecond, Deadline: 2 * time.Second}

// Lock is TryLock, tried again under p while another process holds the
// lock. Its last wait is cut to end at p.Deadline from the first try, where
// it tries once more, so it returns the last try's *HeldError no sooner
// than p.Deadline (0018-PLAN P7, deviation 24); it returns ctx's error when
// ctx ends first.
func Lock(ctx context.Context, path string, p Policy) (release func() error, err error) {
	start, wait := time.Now(), p.First
	for {
		release, err := TryLock(path)
		var held *HeldError
		left := p.Deadline - time.Since(start)
		if !errors.As(err, &held) || wait <= 0 || left <= 0 {
			return release, err
		}
		t := time.NewTimer(min(wait, left))
		select {
		case <-ctx.Done():
			t.Stop()
			return nil, ctx.Err()
		case <-t.C:
		}
		wait = min(2*wait, max(p.Max, p.First))
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
