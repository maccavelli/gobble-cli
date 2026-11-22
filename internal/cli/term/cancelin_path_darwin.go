//go:build darwin

package term

import (
	"syscall"
	"unsafe"
)

// maxPathLen is <sys/param.h>'s MAXPATHLEN, the most F_GETPATH writes.
const maxPathLen = 1024

// terminalPath is the path of the terminal open on fd, by fcntl(2)'s
// F_GETPATH (0063-MADR D22, Q25).
func terminalPath(fd int) (string, error) {
	buf := make([]byte, maxPathLen)
	//nolint:gosec // G103, G115: F_GETPATH fills buf, as fcntl(2) says
	_, _, e := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETPATH, uintptr(unsafe.Pointer(&buf[0])))
	if e != 0 {
		return "", e
	}
	n := 0
	for n < len(buf) && buf[n] != 0 {
		n++
	}
	return string(buf[:n]), nil
}
