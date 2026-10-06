//go:build unix

package jsonl

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// tryLock takes flock's exclusive lock without waiting. ok is false when
// another open file holds it.
func tryLock(f *os.File) (ok bool, err error) {
	err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB) //nolint:gosec // a file descriptor fits in int
	if errors.Is(err, unix.EWOULDBLOCK) {
		return false, nil
	}
	return err == nil, err
}

func unlock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN) //nolint:gosec // as above
}
