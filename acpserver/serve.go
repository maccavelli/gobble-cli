package acpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"

	acp "github.com/coder/acp-go-sdk"
)

// ServeOptions configure Serve.
type ServeOptions struct {
	Agent  Options
	Logger *slog.Logger // nil discards the connection's diagnostics
}

// Serve runs gobble's agent over in and out, the stdio of `gobble acp`. It
// returns nil when in reaches end of file, and context.Cause(ctx) when ctx
// is done first. Nothing but JSON-RPC is written to out.
func Serve(ctx context.Context, in io.Reader, out io.Writer, opts ServeOptions) error {
	agent := New(opts.Agent)
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	// The logger goes in at construction: the connection starts reading in
	// inside the constructor, so a later SetLogger would race with it.
	conn := acp.NewAgentSideConnection(agent, out, in, acp.WithLogger(logger))
	agent.SetConnection(conn)
	select {
	case <-conn.Done():
		return agent.Close()
	case <-ctx.Done():
		return errors.Join(context.Cause(ctx), agent.Close())
	}
}
