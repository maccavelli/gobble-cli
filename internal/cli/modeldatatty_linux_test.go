package cli

import (
	"fmt"
	"os"
	"strconv"
	"syscall"

	"golang.org/x/sys/unix"
)

// openDataPty opens a pseudo-terminal's master and its slave, 120 by 30, for
// M28's interactive case. It is pty_linux_test.go's opener, which only the
// pty build tag compiles, for the default build.
func openDataPty() (master, slave *os.File, _ error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	fd := int(m.Fd()) //nolint:gosec // a descriptor fits an int
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		_ = m.Close() //nolint:errcheck // the error returned is the ioctl's
		return nil, nil, fmt.Errorf("unlock the pseudo-terminal: %w", err)
	}
	n, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		_ = m.Close() //nolint:errcheck // the error returned is the ioctl's
		return nil, nil, fmt.Errorf("the pseudo-terminal's number: %w", err)
	}
	s, err := os.OpenFile("/dev/pts/"+strconv.Itoa(n), os.O_RDWR|syscall.O_NOCTTY, 0)
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
