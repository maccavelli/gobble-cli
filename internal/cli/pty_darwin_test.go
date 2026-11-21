//go:build darwin && pty

package cli

// The macOS opener of the pseudo-terminal cases in pty_unix_test.go
// (2026-11-21, MADR D22, Q26).

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// openPty opens a pseudo-terminal's master and its slave, 120 by 30:
// /dev/ptmx, granted and unlocked, the slave's name from TIOCPTYGNAME.
func openPty() (master, slave *os.File, _ error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	fd := int(m.Fd()) //nolint:gosec // a descriptor fits an int
	for _, req := range []uint{unix.TIOCPTYGRANT, unix.TIOCPTYUNLK} {
		if err := unix.IoctlSetInt(fd, req, 0); err != nil {
			m.Close()
			return nil, nil, fmt.Errorf("grant and unlock the pseudo-terminal: %w", err)
		}
	}
	name := make([]byte, 128)
	//nolint:gosec // G103: TIOCPTYGNAME fills name, as tty(4) says
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&name[0]))); e != 0 {
		m.Close()
		return nil, nil, fmt.Errorf("the pseudo-terminal's name: %w", e)
	}
	n := 0
	for n < len(name) && name[n] != 0 {
		n++
	}
	s, err := os.OpenFile(string(name[:n]), os.O_RDWR|syscall.O_NOCTTY, 0)
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

// fionread is <sys/filio.h>'s FIONREAD, _IOR('f', 127, int).
const fionread = 0x4004667f

// queued is how many bytes the terminal on fd holds unread (FIONREAD).
func queued(fd int) int {
	n, err := unix.IoctlGetInt(fd, fionread)
	if err != nil {
		return -1
	}
	return n
}
