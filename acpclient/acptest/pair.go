package acptest

import (
	"io"
	"log/slog"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// closeTimeout bounds how long cleanup waits for both connections to stop.
var closeTimeout = 5 * time.Second

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
// connections and their handlers are awaited, when the test ends; one that
// outlives closeTimeout fails the test (0031-MADR D5).
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
		for _, wait := range []func() error{c.Agent.Wait, c.Client.Wait} {
			if !handlersDone(wait, closeTimeout) {
				t.Errorf("acptest: a connection or its handlers did not stop within %v of its pipe closing", closeTimeout)
				return
			}
		}
	})
	return c
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
