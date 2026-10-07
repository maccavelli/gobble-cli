package mcpclient

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
)

// startFixture connects to the stdio fixture in mode (an extra variable
// set to "1", or none) and returns the manager and the pid file.
func startFixture(t *testing.T, mode string) (*Manager, string) {
	t.Helper()
	pids := filepath.Join(t.TempDir(), "pids")
	extra := []Pair{{Name: mcptest.EnvPIDs, Value: pids}}
	if mode != "" {
		extra = append(extra, Pair{Name: mode, Value: "1"})
	}
	m := Start(t.Context(), []Server{stdioFixture(t, "fx", extra...)}, Options{})
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	if err := m.Wait(ctx); err != nil {
		t.Fatal(err)
	}
	if st := m.Statuses()[0]; st.State != StateConnected {
		t.Fatalf("fixture %s: %s", st.State, st.Error)
	}
	return m, pids
}

// readPIDs is the server's and its child's pid.
func readPIDs(t *testing.T, path string) (server, child int) {
	t.Helper()
	b, err := os.ReadFile(path) //nolint:gosec // G304: the test's own file
	if err != nil {
		t.Fatal(err)
	}
	f := strings.Fields(string(b))
	if len(f) != 2 {
		t.Fatalf("pid file %q", b)
	}
	server, err = strconv.Atoi(f[0])
	if err == nil {
		child, err = strconv.Atoi(f[1])
	}
	if err != nil {
		t.Fatal(err)
	}
	return server, child
}

// goneWithin waits for each pid to exit, failing the test after d.
func goneWithin(t *testing.T, d time.Duration, pids map[string]int) {
	t.Helper()
	deadline := time.Now().Add(d)
	for name, pid := range pids {
		for alive(pid) {
			if time.Now().After(deadline) {
				t.Fatalf("the %s (pid %d) is still running %v after Close", name, pid, d)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}

// A server that ignores SIGTERM and stdin's end, and its child that ignores
// SIGTERM, are both gone after Close: SIGKILL to the group on Unix, the job
// on Windows (0002-PLAN Phase 8; 0005-PLAN F8's line).
func TestStopStubbornServer(t *testing.T) {
	m, pids := startFixture(t, mcptest.EnvStubborn)
	server, child := readPIDs(t, pids)
	start := time.Now()
	closed := make(chan error, 1)
	go func() { closed <- m.Close() }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(4 * time.Second):
		t.Fatal("Close did not return within 4 s: the stubborn server was not stopped")
	}
	t.Logf("Close took %v", time.Since(start))
	goneWithin(t, 4*time.Second-time.Since(start), map[string]int{"server": server, "child": child})
}

// A server that exits on stdin's end leaves no child running: the group is
// told once the server has exited, or the job is closed.
func TestStopLeavesNoOrphan(t *testing.T) {
	m, pids := startFixture(t, mcptest.EnvOrphan)
	server, child := readPIDs(t, pids)
	start := time.Now()
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Close took %v", time.Since(start))
	goneWithin(t, time.Second, map[string]int{"server": server, "child": child})
}

// A server that exits on stdin's end is stopped within the grace, with no
// error.
func TestStopPoliteServer(t *testing.T) {
	m, _ := startFixture(t, "")
	start := time.Now()
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	t.Logf("Close took %v", time.Since(start))
	if d := time.Since(start); d > time.Second {
		t.Fatalf("Close took %v, want the server to stop within the %v grace", d, stdinGrace)
	}
}
