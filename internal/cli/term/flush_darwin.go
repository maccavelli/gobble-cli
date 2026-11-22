//go:build darwin

package term

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

// fread is <sys/fcntl.h>'s FREAD: TIOCFLUSH's argument for the read queue.
const fread = 0x1

// FlushInput empties what the terminal on f holds of its input, typed or
// pasted and not yet read: tcflush(fd, TCIFLUSH), which on macOS is the
// TIOCFLUSH ioctl with the read queue (0063-MADR D12, D18). On a pipe or a
// file it fails, at once.
func FlushInput(f *os.File) error {
	if f == nil {
		return errors.New("no standard input")
	}
	return unix.IoctlSetPointerInt(int(f.Fd()), unix.TIOCFLUSH, fread) //nolint:gosec // G115: a descriptor fits an int
}
