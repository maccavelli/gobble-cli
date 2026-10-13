package acpserver

import (
	"errors"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// initialize advertises no session deletion, so the client's SDK refuses to
// send session/delete, and a raw one is answered method-not-found by the
// agent's: the SDK made DeleteSession an Agent method, and gobble offers
// none (0014-MADR D3, 0014-PLAN C5; 0019-MADR D8).
func TestDeleteSessionNotFound(t *testing.T) {
	c, _ := connectWith(t, llmtest.NewScript(), &acptest.Client{Allow: true})
	resp := initialize(t, c.Client)
	if resp.AgentCapabilities.SessionCapabilities.Delete != nil {
		t.Fatalf("session deletion advertised: %+v", resp.AgentCapabilities.SessionCapabilities)
	}
	id := openSession(t, c, t.TempDir())
	req := acp.DeleteSessionRequest{SessionId: id}
	_, err := c.Client.DeleteSession(t.Context(), req)
	want := acp.CapabilityError{Method: acp.AgentMethodSessionDelete, Capability: "sessionCapabilities.delete"}
	if ce, ok := errors.AsType[*acp.CapabilityError](err); !ok || *ce != want {
		t.Fatalf("session/delete: %v, want the SDK's %+v", err, want)
	}
	_, err = acp.SendRequest[acp.DeleteSessionResponse](c.Client.Connection(), t.Context(), acp.AgentMethodSessionDelete, req)
	re, ok := errors.AsType[*acp.RequestError](err)
	if !ok || re.Code != -32601 {
		t.Fatalf("raw session/delete: %v, want method not found", err)
	}
}
