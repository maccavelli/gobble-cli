package acptest_test

import (
	"io"
	"log/slog"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// dial runs agent over pipes and returns an initialized SDK client with a
// session open.
func dial(t *testing.T, agent *acptest.ScriptAgent) (*acp.ClientSideConnection, acp.SessionId) {
	t.Helper()
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	go func() {
		err := agent.Serve(t.Context(), toAgentR, toClientW)
		toClientW.CloseWithError(err) //nolint:errcheck,gosec // always nil
	}()
	t.Cleanup(func() { toAgentW.Close() }) //nolint:errcheck,gosec // test cleanup
	c := acp.NewClientSideConnection(&acptest.Client{}, toAgentW, toClientR, acp.WithLogger(slog.New(slog.DiscardHandler)))
	if _, err := c.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber}); err != nil {
		t.Fatal(err)
	}
	s, err := c.NewSession(t.Context(), acp.NewSessionRequest{Cwd: "/w", McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	return c, s.SessionId
}

func prompt(t *testing.T, c *acp.ClientSideConnection, id acp.SessionId) (acp.PromptResponse, error) {
	t.Helper()
	return c.Prompt(t.Context(), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{acp.TextBlock("x")}})
}

func TestScriptAgentPlaysTurns(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{
		{StopReason: "max_tokens", Usage: `{"inputTokens":3,"outputTokens":5,"totalTokens":8}`},
		{ErrCode: -32000, ErrMessage: "provider down"},
		{Updates: []string{`{"sessionUpdate":"nope"`}},
	}}
	c, id := dial(t, agent)
	r, err := prompt(t, c, id)
	if err != nil || r.StopReason != "max_tokens" || r.Usage == nil || r.Usage.OutputTokens != 5 {
		t.Fatalf("turn 1 = %+v, %v", r, err)
	}
	if _, err := prompt(t, c, id); err == nil || !strings.Contains(err.Error(), "provider down") {
		t.Fatalf("turn 2 err = %v, want the scripted error", err)
	}
	if _, err := prompt(t, c, id); err == nil || !strings.Contains(err.Error(), `"update":0`) {
		t.Fatalf("turn 3 err = %v, want the bad update named", err)
	}
	if _, err := prompt(t, c, id); err == nil || !strings.Contains(err.Error(), "script exhausted after 3 turns") {
		t.Fatalf("turn 4 err = %v, want the script exhausted", err)
	}
	if got := agent.Cwds(); len(got) != 1 || got[0] != "/w" {
		t.Fatalf("cwds = %q", got)
	}
	if n := len(agent.Prompts()); n != 4 {
		t.Fatalf("%d prompts recorded, want 4", n)
	}
}
