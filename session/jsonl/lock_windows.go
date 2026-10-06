//go:build windows

package jsonl

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockOffset is where the locked byte range starts: far past any pid the
// file holds, because LockFileEx's locks are mandatory and would otherwise
// keep a second process from reading the holder's pid.
const lockOffset = 1 << 40

func overlapped() *windows.Overlapped {
	return &windows.Overlapped{Offset: uint32(lockOffset & 0xFFFFFFFF), OffsetHigh: uint32(lockOffset >> 32)}
}

// tryLock takes an exclusive lock on one byte at lockOffset without
// waiting. ok is false when another handle holds it.
func tryLock(f *os.File) (ok bool, err error) {
	err = windows.LockFileEx(windows.Handle(f.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, overlapped())
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}
	return err == nil, err
}

func unlock(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, overlapped())
}
