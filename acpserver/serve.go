package acpserver

import (
	"context"
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
	conn := acp.NewAgentSideConnection(agent, out, in)
	agent.SetConnection(conn)
	if opts.Logger != nil {
		conn.SetLogger(opts.Logger)
	} else {
		conn.SetLogger(slog.New(slog.DiscardHandler))
	}
	select {
	case <-conn.Done():
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
