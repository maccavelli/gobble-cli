package acpserver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
)

// Every delete asks: the call raises session/request_permission, a refusal
// leaves the file, and an approval deletes it (0005-PLAN F1c-2; F1c's
// facts: the policy offers no "always", so no earlier answer stands for a
// later delete).
func TestDeleteAsks(t *testing.T) {
	for _, allow := range []bool{false, true} {
		dir := t.TempDir()
		victim := filepath.Join(dir, "keep.txt")
		if err := os.WriteFile(victim, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		model := &echoResult{call: llm.ToolCallDone{ID: "d1", Name: "delete", Args: toolArgs(t, map[string]string{"path": "keep.txt"})}}
		c, client := connectWith(t, model, &acptest.Client{Allow: allow})
		initialize(t, c.Client)
		s, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: dir, McpServers: []acp.McpServer{}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := prompt(t, c, s.SessionId, "delete keep.txt"); err != nil {
			t.Fatal(err)
		}
		if n := len(client.Permissions()); n != 1 {
			t.Fatalf("allow %v: %d permission requests, want one", allow, n)
		}
		_, err = os.Stat(victim)
		if deleted := os.IsNotExist(err); deleted != allow {
			t.Fatalf("allow %v: deleted = %v (stat %v)", allow, deleted, err)
		}
	}
}
