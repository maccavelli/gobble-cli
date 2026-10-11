package acpserver

import (
	"errors"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// session/delete answers method-not-found, and initialize advertises no
// session deletion: the SDK made DeleteSession an Agent method, and gobble
// offers none (0014-MADR D3, 0014-PLAN C5).
func TestDeleteSessionNotFound(t *testing.T) {
	c, _ := connectWith(t, llmtest.NewScript(), &acptest.Client{Allow: true})
	resp := initialize(t, c.Client)
	if resp.AgentCapabilities.SessionCapabilities.Delete != nil {
		t.Fatalf("session deletion advertised: %+v", resp.AgentCapabilities.SessionCapabilities)
	}
	id := openSession(t, c, t.TempDir())
	_, err := c.Client.DeleteSession(t.Context(), acp.DeleteSessionRequest{SessionId: id})
	re, ok := errors.AsType[*acp.RequestError](err)
	if !ok || re.Code != -32601 {
		t.Fatalf("session/delete: %v, want method not found", err)
	}
}
