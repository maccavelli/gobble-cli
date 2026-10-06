package acptest

import (
	"io"
	"log/slog"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"
)

// closeTimeout bounds how long cleanup waits for both connections to stop.
const closeTimeout = 5 * time.Second

// Conn is an agent and a client joined by in-memory pipes.
type Conn struct {
	Client   *acp.ClientSideConnection
	Agent    *acp.AgentSideConnection
	Recorder *Recorder
}

// Pair connects agent to a new default Client and returns the client side
// and the recorder. The pipes close when the test ends.
func Pair(t testing.TB, agent acp.Agent) (*acp.ClientSideConnection, *Recorder) {
	t.Helper()
	c := Connect(t, agent, &Client{})
	return c.Client, c.Recorder
}

// Connect joins agent and client over two pipes, records every frame, and
// returns both sides. An agent that sends notifications needs c.Agent: set
// it on the agent before the first request. The pipes close, and both
// connections are awaited, when the test ends.
func Connect(t testing.TB, agent acp.Agent, client acp.Client) *Conn {
	t.Helper()
	rec := &Recorder{}
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	quiet := acp.WithLogger(slog.New(slog.DiscardHandler))
	c := &Conn{
		Agent:    acp.NewAgentSideConnection(agent, rec.tap(AgentToClient, toClientW), toAgentR, quiet),
		Client:   acp.NewClientSideConnection(client, rec.tap(ClientToAgent, toAgentW), toClientR, quiet),
		Recorder: rec,
	}
	t.Cleanup(func() {
		for _, err := range []error{toAgentW.Close(), toClientW.Close()} {
			if err != nil {
				t.Errorf("acptest: close pipe: %v", err)
			}
		}
		deadline := time.After(closeTimeout)
		for _, done := range []<-chan struct{}{c.Agent.Done(), c.Client.Done()} {
			select {
			case <-done:
			case <-deadline:
				t.Errorf("acptest: a connection did not stop within %v of its pipe closing", closeTimeout)
				return
			}
		}
	})
	return c
}
