package provider

import (
	"net"
	"net/url"
	"os"
	"strconv"
	"testing"
)

// listensHere reports whether a socket of this process listens on TCP port
// port, at any address. It asks the operating system for this process's own
// sockets, so another program's listener on the port, which a dial would
// reach, is never taken for this process's (0068-MADR W3 and W15, as
// amended 2026-11-27): a Grok sign-in with no listener names a port drawn
// from 49152 to 65535, the dynamic range where every :0 bind on the host
// lands. Each operating system's lookup is ownListeningPorts.
func listensHere(port int) (bool, error) {
	ports, err := ownListeningPorts()
	return ports[port], err
}

// listensSince reports whether a socket of this process listens on TCP port
// port now that did not when before, ownListeningPorts' answer, was taken:
// a listener the code under test bound in between. A listener the test
// itself held before, such as its issuer's httptest server, which can sit on
// any port of the dynamic range, is not counted (0068-MADR W3, as amended
// 2026-11-27, pass 2).
func listensSince(before map[int]bool, port int) (bool, error) {
	held, err := listensHere(port)
	return held && !before[port], err
}

// listensHere answers for this process only: the network guard, which
// TestMain's parent process holds on the port HTTPS_PROXY names, is not this
// process's; a listener this test opens is, until it is closed (0068-MADR
// W15).
func TestListensHereSeesOnlyThisProcess(t *testing.T) {
	guard, err := url.Parse(os.Getenv("HTTPS_PROXY"))
	if err != nil || guard.Port() == "" {
		t.Fatalf("HTTPS_PROXY %q names no guard port (%v): this test runs behind TestMain's network guard", os.Getenv("HTTPS_PROXY"), err)
	}
	guardPort, err := strconv.Atoi(guard.Port())
	if err != nil {
		t.Fatal(err)
	}
	// Something holds the guard's port: this process cannot bind it. A dial
	// would show it too, but the guard counts every connection as a request
	// that tried to reach the network.
	if ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", guard.Host); err == nil {
		_ = ln.Close()
		t.Fatalf("bound the guard's address %s: nothing holds it", guard.Host)
	}
	if held, err := listensHere(guardPort); held || err != nil {
		t.Errorf("listensHere(%d), the guard's port, held by another process = %v, %v; want false", guardPort, held, err)
	}

	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("listener address %v is not TCP", ln.Addr())
	}
	if held, err := listensHere(addr.Port); !held || err != nil {
		t.Errorf("listensHere(%d), this test's own listener = %v, %v; want true", addr.Port, held, err)
	}
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	if held, err := listensHere(addr.Port); held || err != nil {
		t.Errorf("listensHere(%d) after its listener closed = %v, %v; want false", addr.Port, held, err)
	}
}
