package jsonl

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// lockHeld is lock's error when another process holds the lock.
type lockHeld struct{ pid int }

func (e *lockHeld) Error() string { return fmt.Sprintf("locked by pid %d", e.pid) }

// lock takes the exclusive writer lock on the sidecar path and writes this
// process's pid into it (0008-MADR D19 item 8). The lock is advisory and
// held for the life of the returned release; the file is left in place, so
// no two processes race to create and delete it.
func lock(path string) (release func() error, err error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600) //nolint:gosec // G304: the store's own lock file
	if err != nil {
		return nil, fmt.Errorf("session: lock: %w", err)
	}
	ok, err := tryLock(f)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("session: lock: %w", err), f.Close())
	}
	if !ok {
		pid := readPID(f)
		return nil, errors.Join(&lockHeld{pid: pid}, f.Close())
	}
	if err := writePID(f); err != nil {
		return nil, errors.Join(fmt.Errorf("session: lock: %w", err), unlock(f), f.Close())
	}
	return func() error { return errors.Join(unlock(f), f.Close()) }, nil
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
