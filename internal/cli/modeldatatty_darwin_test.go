package cli

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// openDataPty opens a pseudo-terminal's master and its slave, 120 by 30, for
// M28's interactive case: /dev/ptmx, granted and unlocked, the slave's name
// from TIOCPTYGNAME. It is pty_darwin_test.go's opener, which only the pty
// build tag compiles, for the default build.
func openDataPty() (master, slave *os.File, _ error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	fd := int(m.Fd()) //nolint:gosec // a descriptor fits an int
	for _, req := range []uint{unix.TIOCPTYGRANT, unix.TIOCPTYUNLK} {
		if err := unix.IoctlSetInt(fd, req, 0); err != nil {
			_ = m.Close() //nolint:errcheck // the error returned is the ioctl's
			return nil, nil, fmt.Errorf("grant and unlock the pseudo-terminal: %w", err)
		}
	}
	name := make([]byte, 128)
	//nolint:gosec // G103: TIOCPTYGNAME fills name, as tty(4) says
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(unix.TIOCPTYGNAME), uintptr(unsafe.Pointer(&name[0]))); e != 0 {
		_ = m.Close() //nolint:errcheck // the error returned is the ioctl's
		return nil, nil, fmt.Errorf("the pseudo-terminal's name: %w", e)
	}
	n := 0
	for n < len(name) && name[n] != 0 {
		n++
	}
	s, err := os.OpenFile(string(name[:n]), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		_ = m.Close() //nolint:errcheck // the error returned is the open's
		return nil, nil, err
	}
	if err := unix.IoctlSetWinsize(int(s.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 30, Col: 120}); err != nil { //nolint:gosec // a descriptor fits an int
		_ = m.Close() //nolint:errcheck // the error returned is the ioctl's
		_ = s.Close() //nolint:errcheck // as above
		return nil, nil, fmt.Errorf("the pseudo-terminal's size: %w", err)
	}
	return m, s, nil
}
