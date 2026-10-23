package acpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// ServeOptions configure Serve.
type ServeOptions struct {
	Agent  Options
	Logger *slog.Logger // nil discards the connection's diagnostics
}

// endWait bounds how long Serve waits for the connection's handlers.
const endWait = 5 * time.Second

// Serve runs gobble's agent over in and out, the stdio of `gobble acp`. It
// returns nil when in reaches end of file, in's error when reading it
// fails, and context.Cause(ctx) when ctx is done first; each only once the
// connection's handlers have returned, or endWait has passed (0031-MADR
// D5). Nothing but JSON-RPC is written to out.
func Serve(ctx context.Context, in io.Reader, out io.Writer, opts ServeOptions) error {
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if opts.Agent.Logger == nil {
		opts.Agent.Logger = logger
	}
	agent := New(opts.Agent)
	// The logger goes in at construction: the connection starts reading in
	// inside the constructor, so a later SetLogger would race with it. A
	// client's bare stdio MCP command is resolved before the SDK checks it
	// (0019-MADR D6). The constructor binds the agent through
	// SetAgentConnection before it reads. Every prompt reaches the agent,
	// whose claimTurn refuses a second while one runs, rather than the SDK
	// cancelling the first (0016-MADR A1).
	conn := acp.NewAgentSideConnection(agent, out, in, acp.WithLogger(logger), acp.WithInboundRewrite(resolveBareCommands),
		acp.WithPromptOverlapPolicy(acp.PromptOverlapConcurrent))
	select {
	case <-conn.Done():
		return errors.Join(ended(conn), agent.Close())
	case <-ctx.Done():
		cerr := conn.Close()
		return errors.Join(context.Cause(ctx), cerr, ended(conn), agent.Close())
	}
}

// ended waits for conn's handlers and says why it ended, when that is a
// fault: nil at the end of its input or when this side closed it, the
// input's error when reading it failed (0031-MADR D5).
func ended(conn *acp.AgentSideConnection) error {
	if !handlersDone(conn.Wait, endWait) {
		return fmt.Errorf("the connection's handlers did not stop within %v of its end", endWait)
	}
	if err := conn.Err(); !errors.Is(err, io.EOF) && !errors.Is(err, acp.ErrConnectionClosed) {
		return err
	}
	return nil
}

// handlersDone waits for wait, at most limit, and reports whether it
// returned: a handler that ignores its context would hold a connection's
// Wait forever (0031-MADR D5).
func handlersDone(wait func() error, limit time.Duration) bool {
	done := make(chan struct{})
	go func() {
		_ = wait() //nolint:errcheck // the connection's end is read from Err
		close(done)
	}()
	timer := time.NewTimer(limit)
	defer timer.Stop()
	select {
	case <-done:
		return true
	case <-timer.C:
		return false
	}
}
