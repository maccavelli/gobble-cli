package acpserver

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// counterIDs makes s1, s2, … so transcripts are deterministic.
func counterIDs() func() string {
	n := 0
	return func() string {
		n++
		return fmt.Sprintf("s%d", n)
	}
}

func connect(t *testing.T) (*acptest.Conn, *acptest.Client) {
	t.Helper()
	return connectWith(t, llmtest.NewScript(llmtest.Text("hello gobble")), &acptest.Client{})
}

// connectWith joins an agent on model p, with the built-in tools, to client.
func connectWith(t *testing.T, p llm.Provider, client *acptest.Client) (*acptest.Conn, *acptest.Client) {
	t.Helper()
	agent := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func() (llm.Provider, error) { return p, nil }})
	c := acptest.Connect(t, agent, client)
	agent.SetConnection(c.Agent)
	return c, client
}

func initialize(t *testing.T, c *acp.ClientSideConnection) acp.InitializeResponse {
	t.Helper()
	resp, err := c.Initialize(t.Context(), acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientInfo:      &acp.Implementation{Name: "test", Version: "0.0.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// Initialize → NewSession → Prompt → CloseSession through the SDK's client,
// against a golden transcript (0002-PLAN Phase 1 step 4; the scripted model
// of Phase 3 replaced the echo).
func TestSession(t *testing.T) {
	c, client := connect(t)
	resp := initialize(t, c.Client)
	if resp.ProtocolVersion != 1 {
		t.Fatalf("protocol version %d, want 1", resp.ProtocolVersion)
	}
	if resp.AgentInfo == nil || resp.AgentInfo.Name != "gobble" || resp.AgentInfo.Title == nil || *resp.AgentInfo.Title != "gobble" {
		t.Fatalf("agentInfo = %+v, want gobble/gobble", resp.AgentInfo)
	}
	sess, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: "/work", McpServers: []acp.McpServer{}})
	if err != nil || sess.SessionId != "s1" {
		t.Fatalf("new session = %+v, %v", sess, err)
	}
	pr, err := c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: sess.SessionId,
		Prompt: []acp.ContentBlock{acp.TextBlock("hello"), acp.TextBlock("gobble")}})
	if err != nil || pr.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("prompt = %+v, %v", pr, err)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: sess.SessionId}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := c.Recorder.Wait(ctx, 13); err != nil {
		t.Fatal(err)
	}
	// session/new's three frames (0002-PLAN Phase 6), then the reply's
	// chunk and its usage_update.
	updates := client.Updates()
	if len(updates) != 5 || updates[0].Update.AvailableCommandsUpdate == nil || updates[1].Update.CurrentModeUpdate == nil ||
		updates[2].Update.UsageUpdate == nil || updates[3].Update.AgentMessageChunk == nil ||
		updates[3].Update.AgentMessageChunk.Content.Text == nil ||
		updates[3].Update.AgentMessageChunk.Content.Text.Text != "hello gobble" || updates[4].Update.UsageUpdate == nil {
		t.Fatalf("updates = %+v, want the session's frames, the reply's chunk, then usage_update", updates)
	}
	c.Recorder.Golden(t, filepath.Join("testdata", "session.golden"))
}

// The advertised set is exactly what is implemented. A later phase that
// turns a capability on changes this test on purpose (0002-PLAN amendment
// of 2026-10-05). Phase 4 turned on loading, listing and resuming; Phase 5
// turned on MCP over streamable HTTP.
func TestCapabilitiesAreHonest(t *testing.T) {
	c, _ := connect(t)
	got := initialize(t, c.Client).AgentCapabilities
	switch {
	case !got.LoadSession:
		t.Error("loadSession is implemented (Phase 4) and must be advertised")
	case got.SessionCapabilities.List == nil, got.SessionCapabilities.Resume == nil:
		t.Error("session list and resume are implemented (Phase 4) and must be advertised")
	case got.SessionCapabilities.Close == nil:
		t.Error("session close is implemented and must be advertised")
	case got.SessionCapabilities.Fork != nil, got.SessionCapabilities.Delete != nil:
		t.Error("session fork or delete advertised; neither is implemented")
	case !got.McpCapabilities.Http:
		t.Error("MCP over streamable HTTP is implemented (Phase 5) and must be advertised")
	case got.McpCapabilities.Sse, got.McpCapabilities.Acp:
		t.Error("MCP over SSE or ACP advertised; neither is offered")
	case got.PromptCapabilities.Image, got.PromptCapabilities.Audio, got.PromptCapabilities.EmbeddedContext:
		t.Error("prompt capabilities advertised before Phase 3")
	}
}

var semver = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

func TestVersionIsSemVer(t *testing.T) {
	for _, v := range []string{"1.2.3", "0.0.0-dev+4e25c8a01234.dirty"} {
		agent := New(Options{Version: v})
		resp, err := agent.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: 1})
		if err != nil || !semver.MatchString(resp.AgentInfo.Version) {
			t.Fatalf("agentInfo.version = %q, %v", resp.AgentInfo.Version, err)
		}
	}
}

func requestCode(t *testing.T, err error) int {
	t.Helper()
	var re *acp.RequestError
	if !errors.As(err, &re) {
		t.Fatalf("err = %v, want a JSON-RPC error", err)
	}
	return re.Code
}

func TestErrors(t *testing.T) {
	c, _ := connect(t)
	initialize(t, c.Client)
	_, err := c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: "nope", Prompt: []acp.ContentBlock{acp.TextBlock("x")}})
	if code := requestCode(t, err); code != -32602 {
		t.Fatalf("prompt to an unknown session: code %d, want -32602", code)
	}
	_, err = c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: "nope"})
	if code := requestCode(t, err); code != -32602 {
		t.Fatalf("close of an unknown session: code %d, want -32602", code)
	}
	_, err = c.Client.SetSessionMode(t.Context(), acp.SetSessionModeRequest{SessionId: "nope", ModeId: "plan"})
	if code := requestCode(t, err); code != -32602 {
		t.Fatalf("session/set_mode on an unknown session: code %d, want -32602 (Phase 6 implements it)", code)
	}
	if _, err := c.Client.Authenticate(t.Context(), acp.AuthenticateRequest{MethodId: "x"}); requestCode(t, err) != -32601 {
		t.Fatal("authenticate is not offered")
	}
}

func TestDefaultIDsAreUUIDv7(t *testing.T) {
	a := New(Options{Version: "1.0.0"})
	resp, err := a.NewSession(t.Context(), acp.NewSessionRequest{Cwd: "/w"})
	if err != nil || !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(string(resp.SessionId)) {
		t.Fatalf("session id %q, %v; want a UUIDv7", resp.SessionId, err)
	}
}
