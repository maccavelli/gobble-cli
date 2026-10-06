package cli

import (
	"context"
	"io"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpserver"
)

// agentServe is the agent gobble's own modes drive: gobble's agent, in this
// process, spoken to over ACP like any other client (0002-MADR rule 3).
// Tests replace it with a scripted agent.
var agentServe = func(e *runEnv) acpclient.ServeFunc {
	return func(ctx context.Context, in io.Reader, out io.Writer) error {
		return acpserver.Serve(ctx, in, out, acpserver.ServeOptions{
			Agent:  acpserver.Options{Version: identity().Version},
			Logger: e.log(),
		})
	}
}

// startAgent starts the agent and sends initialize. A dropped notification
// is reported once per drop on stderr; the session goes on.
func (e *runEnv) startAgent(ctx context.Context) (*acpclient.Conn, error) {
	conn, err := acpclient.Start(ctx, agentServe(e), acpclient.Options{
		Name:    "gobble",
		Version: identity().Version,
		Logger:  e.log(),
		OnDrop: func(method string, total uint64) {
			e.out.Warnf("dropped %d ACP notifications (%s)", total, method)
		},
	})
	if err != nil {
		return nil, failf("start the agent: %v", err)
	}
	return conn, nil
}
