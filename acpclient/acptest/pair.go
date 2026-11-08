package acptest

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
	sdkacptest "github.com/maccavelli/acp-go-sdk2/acptest"
)

// closeTimeout bounds how long cleanup waits for both connections to stop.
var closeTimeout = 5 * time.Second

// Conn is an agent and a client joined by in-memory pipes.
type Conn struct {
	Client *acp.ClientSideConnection
	Agent  *acp.AgentSideConnection
	// Recorder holds the client's side of the wire, as acp-go-sdk2's
	// acptest.ConnectRecorded records it: a Sent record went to the agent,
	// a Received one came from it, in the order the client wrote and read
	// them. acptest.ValidateGolden reads records so (0016-MADR D6).
	Recorder *sdkacptest.Recorder
}

// Pair connects agent to a new default Client and returns the client side
// and the recorder. The pipes close when the test ends.
func Pair(t testing.TB, agent acp.Agent) (*acp.ClientSideConnection, *sdkacptest.Recorder) {
	t.Helper()
	c := Connect(t, agent, &Client{})
	return c.Client, c.Recorder
}

// Connect joins agent and client over two pipes, records every frame on the
// client's side with acp-go-sdk2's acptest.Recorder, and returns both sides.
// An agent that sends notifications needs c.Agent: set it on the agent
// before the first request. The pipes close, and both connections and their
// handlers are awaited, when the test ends; one that outlives closeTimeout
// fails the test (0031-MADR D5).
func Connect(t testing.TB, agent acp.Agent, client acp.Client) *Conn {
	t.Helper()
	rec := &sdkacptest.Recorder{}
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	quiet := acp.WithLogger(slog.New(slog.DiscardHandler))
	c := &Conn{
		Agent:    acp.NewAgentSideConnection(agent, toClientW, toAgentR, quiet),
		Client:   acp.NewClientSideConnection(client, rec.Writer(toAgentW), rec.Reader(toClientR), quiet),
		Recorder: rec,
	}
	t.Cleanup(func() {
		for _, err := range []error{toAgentW.Close(), toClientW.Close()} {
			if err != nil {
				t.Errorf("acptest: close pipe: %v", err)
			}
		}
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		for _, conn := range []*acp.Connection{c.Agent.Connection(), c.Client.Connection()} {
			if err := conn.WaitContext(ctx); errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("acptest: a connection or its handlers did not stop within %v of its pipe closing", closeTimeout)
				return
			}
		}
	})
	return c
}
