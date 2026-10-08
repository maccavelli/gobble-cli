package mcpclient

import (
	"context"
	"io"
	"os/exec"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/gobble-cli/internal/proctree"
)

// Pi's stop timings (pkg-stdio.ts): the grace after stdin closes, and the
// wait after SIGTERM before SIGKILL.
const (
	stdinGrace   = 500 * time.Millisecond
	closeTimeout = 2 * time.Second
)

// stdioTransport runs a stdio MCP server as Pi does (0002-PLAN Phase 8): in
// its own process tree, a process group on Unix and a Job object on
// Windows, so stopping it stops what it started too. The JSON-RPC is the
// go-sdk's IOTransport over its pipes.
type stdioTransport struct {
	cmd *exec.Cmd
}

var _ mcp.Transport = (*stdioTransport)(nil)

// Connect starts the server and connects to it over stdin and stdout.
func (t *stdioTransport) Connect(ctx context.Context) (mcp.Connection, error) {
	stdout, err := t.cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stdin, err := t.cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	proctree.Prepare(t.cmd)
	if err := t.cmd.Start(); err != nil {
		return nil, err
	}
	tr, err := proctree.Attach(t.cmd.Process)
	if err != nil {
		// A server gobble could not stop does not run.
		_ = t.cmd.Process.Kill() //nolint:errcheck // the connection fails with err
		_ = t.cmd.Wait()         //nolint:errcheck // as above
		return nil, err
	}
	s := &stopper{stdin: stdin, tree: tr, done: make(chan struct{})}
	go func() {
		_ = t.cmd.Wait() //nolint:errcheck // an exit is what stopper waits for, not an error
		close(s.done)
	}()
	// The connection closes by closing stdin, not stdout (as the go-sdk's
	// CommandTransport does).
	return (&mcp.IOTransport{Reader: io.NopCloser(stdout), Writer: s}).Connect(ctx)
}

// stopper is the server's stdin. Closing it stops the server as Pi does:
// stdin closes; after stdinGrace the tree is terminated (SIGTERM on Unix,
// the job on Windows); after closeTimeout more, killed. Once the server has
// exited, on whichever step, the tree is told too, for children that
// outlive it.
type stopper struct {
	stdin io.WriteCloser
	tree  *proctree.Tree
	done  chan struct{} // closed when the server has exited
	once  sync.Once
	err   error
}

func (s *stopper) Write(p []byte) (int, error) { return s.stdin.Write(p) }

// Close stops the server. It returns stdin's close error only: an exit by
// signal is the asked-for outcome.
func (s *stopper) Close() error {
	s.once.Do(func() {
		s.err = s.stdin.Close()
		if !s.exitedWithin(stdinGrace) {
			s.tree.Terminate()
			if !s.exitedWithin(closeTimeout) {
				s.tree.Kill()
				<-s.done
			}
		}
		s.tree.Exited()
		s.tree.Release()
	})
	return s.err
}

func (s *stopper) exitedWithin(d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-s.done:
		return true
	case <-t.C:
		return false
	}
}
