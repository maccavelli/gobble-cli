package acpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os/exec"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// ServeOptions configure Serve.
type ServeOptions struct {
	Agent  Options
	Logger *slog.Logger // nil discards the connection's diagnostics
}

// Serve runs gobble's agent over in and out, the stdio of `gobble acp`,
// through the SDK's ServeAgent. It returns nil when in reaches end of file
// and the connection ended cleanly, as acp.EndedCleanly says (a close a
// notification overflow made is not clean), the connection's error
// otherwise, and context.Cause(ctx) when ctx is done first, after a
// graceful shutdown that answers a prompt in flight cancelled; each once the
// connection's handlers have returned, or 5 s have passed (0031-MADR D5;
// 0035-MADR D11). The agent closes after. Nothing but JSON-RPC is written
// to out.
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
	// client's bare stdio MCP command is resolved on PATH before the SDK
	// checks it, by the SDK's ResolveStdioCommands (0019-MADR D6). The
	// constructor binds the agent through SetAgentConnection before it
	// reads. Every prompt reaches the agent, whose claimTurn refuses a
	// second while one runs, rather than the SDK cancelling the first
	// (0016-MADR A1).
	err := acp.ServeAgent(ctx, agent, in, out, acp.WithLogger(logger),
		acp.WithInboundRewrite(acp.ResolveStdioCommands(exec.LookPath)),
		acp.WithPromptOverlapPolicy(acp.PromptOverlapConcurrent))
	return errors.Join(err, agent.Close())
}
