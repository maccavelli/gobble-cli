//go:build linux && pty

package cli

// The Linux opener of the pseudo-terminal cases in pty_unix_test.go
// (2026-11-21, MADR D22, Q26: the cases moved there word for word; this file
// keeps what is Linux's).

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// openPty opens a pseudo-terminal's master and its slave, 120 by 30.
func openPty() (master, slave *os.File, _ error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	fd := int(m.Fd()) //nolint:gosec // a descriptor fits an int
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		m.Close()
		return nil, nil, fmt.Errorf("unlock the pseudo-terminal: %w", err)
	}
	n, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		m.Close()
		return nil, nil, fmt.Errorf("the pseudo-terminal's number: %w", err)
	}
	s, err := os.OpenFile("/dev/pts/"+strconv.Itoa(n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		m.Close()
		return nil, nil, err
	}
	if err := unix.IoctlSetWinsize(int(s.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 30, Col: 120}); err != nil { //nolint:gosec // a descriptor fits an int
		m.Close()
		s.Close()
		return nil, nil, fmt.Errorf("the pseudo-terminal's size: %w", err)
	}
	return m, s, nil
}

// queued is how many bytes the terminal on fd holds unread (FIONREAD).
func queued(fd int) int {
	n, err := unix.IoctlGetInt(fd, unix.TIOCINQ)
	if err != nil {
		return -1
	}
	return n
}
