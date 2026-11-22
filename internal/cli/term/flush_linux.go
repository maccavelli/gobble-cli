//go:build linux

package term

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// FlushInput empties what the terminal on f holds of its input, typed or
// pasted and not yet read: tcflush(fd, TCIFLUSH), which on Linux is the
// TCFLSH ioctl (0063-MADR D12, D18). On a pipe or a file it fails, at once.
func FlushInput(f *os.File) error {
	if f == nil {
		return errors.New("no standard input")
	}
	return unix.IoctlSetInt(int(f.Fd()), unix.TCFLSH, unix.TCIFLUSH) //nolint:gosec // G115: a descriptor fits an int
}
