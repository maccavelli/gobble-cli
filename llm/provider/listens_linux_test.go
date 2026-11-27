package provider

import "syscall"

// fdListening reports whether socket fd accepts connections: SO_ACCEPTCONN,
// which Linux sets on a stream socket once it listens.
func fdListening(fd int) bool {
	on, err := syscall.GetsockoptInt(fd, syscall.SOL_SOCKET, syscall.SO_ACCEPTCONN)
	return err == nil && on != 0
}
