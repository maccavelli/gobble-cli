package acpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"

	acp "github.com/maccavelli/acp-go-sdk2"
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
		return agent.Close()
	case <-ctx.Done():
		return errors.Join(context.Cause(ctx), conn.Close(), agent.Close())
	}
}
