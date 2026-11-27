package provider

import "golang.org/x/sys/unix"

// tcpsListen is a TCP socket's state once it listens (TCPS_LISTEN,
// <netinet/tcp_fsm.h>).
const tcpsListen = 1

// fdListening reports whether socket fd is a TCP socket in the listen state.
// macOS answers no SO_ACCEPTCONN query (ENOPROTOOPT), so the state is read
// from TCP_CONNECTION_INFO; a socket that is not TCP refuses that query.
func fdListening(fd int) bool {
	info, err := unix.GetsockoptTCPConnectionInfo(fd, unix.IPPROTO_TCP, unix.TCP_CONNECTION_INFO)
	return err == nil && info.State == tcpsListen
}
