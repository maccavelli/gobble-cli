package acpclient_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// elicitAgent is a ScriptAgent that keeps its connection, to make requests
// of the client, and the client capabilities initialize gave it.
type elicitAgent struct {
	*acptest.ScriptAgent
	conn chan *acp.AgentSideConnection
	caps chan acp.ClientCapabilities
}

func (a *elicitAgent) Initialize(ctx context.Context, p acp.InitializeRequest) (acp.InitializeResponse, error) {
	a.caps <- p.ClientCapabilities
	return a.ScriptAgent.Initialize(ctx, p)
}

func (a *elicitAgent) serve(ctx context.Context, in io.Reader, out io.Writer) error {
	conn := acp.NewAgentSideConnection(a, out, in, acp.WithLogger(slog.New(slog.DiscardHandler)))
	a.conn <- conn
	select {
	case <-conn.Done():
		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}

// The client advertises no elicitation, answers elicitation/create with
// method-not-found, and ignores elicitation/complete: the SDK made both
// Client methods, and gobble's client offers neither (0014-MADR D3).
func TestElicitationNotFound(t *testing.T) {
	agent := &elicitAgent{ScriptAgent: &acptest.ScriptAgent{}, conn: make(chan *acp.AgentSideConnection, 1), caps: make(chan acp.ClientCapabilities, 1)}
	c, err := acpclient.Start(t.Context(), agent.serve, acpclient.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	var conn *acp.AgentSideConnection
	select {
	case conn = <-agent.conn:
	case <-ctx.Done():
		t.Fatal("the agent did not start")
	}
	select {
	case caps := <-agent.caps:
		if caps.Elicitation != nil {
			t.Fatalf("elicitation advertised: %+v", caps.Elicitation)
		}
	case <-ctx.Done():
		t.Fatal("no initialize")
	}
	_, err = conn.CreateElicitation(ctx, acp.CreateElicitationRequest{Url: &acp.CreateElicitationUrl{
		ElicitationId: "e1", Message: "Sign in", Mode: "url", Url: "https://example.com/login"}})
	re, ok := errors.AsType[*acp.RequestError](err)
	if !ok || re.Code != -32601 {
		t.Fatalf("elicitation/create: %v, want method not found", err)
	}
	if err := conn.CompleteElicitation(ctx, acp.CompleteElicitationNotification{ElicitationId: "e1"}); err != nil {
		t.Fatalf("elicitation/complete: %v", err)
	}
	if _, err := c.NewSession(ctx, "/w", func(acpclient.Update) {}); err != nil {
		t.Fatalf("the connection after elicitation/complete: %v", err)
	}
}
