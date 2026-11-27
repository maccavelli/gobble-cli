//go:build linux || darwin

package provider

import (
	"os"
	"strconv"
	"syscall"
)

// ownListeningPorts reads this process's open descriptors from /dev/fd and
// keeps the bound port of each listening TCP socket (fdListening). A
// descriptor that is not a socket, or that closed after the directory was
// read, is passed over.
func ownListeningPorts() (map[int]bool, error) {
	entries, err := os.ReadDir("/dev/fd")
	if err != nil {
		return nil, err
	}
	ports := map[int]bool{}
	for _, e := range entries {
		fd, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if p := fdPort(fd); p >= 0 && fdListening(fd) {
			ports[p] = true
		}
	}
	return ports, nil
}

// fdPort is the TCP or UDP port descriptor fd is bound to, or -1 when it is
// not an internet socket.
func fdPort(fd int) int {
	sa, err := syscall.Getsockname(fd)
	if err != nil {
		return -1
	}
	switch a := sa.(type) {
	case *syscall.SockaddrInet4:
		return a.Port
	case *syscall.SockaddrInet6:
		return a.Port
	}
	return -1
}
