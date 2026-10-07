package acptest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"
)

// stubAgent answers initialize, opens one session, and replies to a prompt
// with one message chunk.
type stubAgent struct {
	conn *acp.AgentSideConnection
}

var _ acp.Agent = (*stubAgent)(nil)

func (*stubAgent) Initialize(context.Context, acp.InitializeRequest) (acp.InitializeResponse, error) {
	return acp.InitializeResponse{
		ProtocolVersion: acp.ProtocolVersionNumber,
		AgentInfo:       &acp.Implementation{Name: "stub", Version: "1.0.0"},
		AuthMethods:     []acp.AuthMethod{},
	}, nil
}

func (*stubAgent) NewSession(context.Context, acp.NewSessionRequest) (acp.NewSessionResponse, error) {
	return acp.NewSessionResponse{SessionId: "s1"}, nil
}

func (a *stubAgent) Prompt(ctx context.Context, p acp.PromptRequest) (acp.PromptResponse, error) {
	err := a.conn.SessionUpdate(ctx, acp.SessionNotification{
		SessionId: p.SessionId,
		Update:    acp.UpdateAgentMessageText("hello"),
	})
	return acp.PromptResponse{StopReason: acp.StopReasonEndTurn}, err
}

func (*stubAgent) Authenticate(context.Context, acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	return acp.AuthenticateResponse{}, nil
}

func (*stubAgent) Logout(context.Context, acp.LogoutRequest) (acp.LogoutResponse, error) {
	return acp.LogoutResponse{}, nil
}

func (*stubAgent) Cancel(context.Context, acp.CancelNotification) error { return nil }

func (*stubAgent) CloseSession(context.Context, acp.CloseSessionRequest) (acp.CloseSessionResponse, error) {
	return acp.CloseSessionResponse{}, nil
}

func (*stubAgent) ListSessions(context.Context, acp.ListSessionsRequest) (acp.ListSessionsResponse, error) {
	return acp.ListSessionsResponse{}, nil
}

func (*stubAgent) ResumeSession(context.Context, acp.ResumeSessionRequest) (acp.ResumeSessionResponse, error) {
	return acp.ResumeSessionResponse{}, nil
}

func (*stubAgent) SetSessionConfigOption(context.Context, acp.SetSessionConfigOptionRequest) (acp.SetSessionConfigOptionResponse, error) {
	return acp.SetSessionConfigOptionResponse{}, nil
}

func (*stubAgent) SetSessionMode(context.Context, acp.SetSessionModeRequest) (acp.SetSessionModeResponse, error) {
	return acp.SetSessionModeResponse{}, nil
}

func initialize(t *testing.T, c *acp.ClientSideConnection) acp.InitializeResponse {
	t.Helper()
	resp, err := c.Initialize(t.Context(), acp.InitializeRequest{
		ProtocolVersion: acp.ProtocolVersionNumber,
		ClientInfo:      &acp.Implementation{Name: "acptest", Version: "0.0.0"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

// Pair round-trips initialize against a stub agent, and the transcript
// matches its golden file (0004-PLAN Phase 3 Accept).
func TestPairInitialize(t *testing.T) {
	client, rec := Pair(t, &stubAgent{})
	resp := initialize(t, client)
	if resp.AgentInfo == nil || resp.AgentInfo.Name != "stub" || resp.ProtocolVersion != acp.ProtocolVersionNumber {
		t.Fatalf("initialize = %+v", resp)
	}
	frames := rec.Frames()
	if len(frames) != 2 || frames[0].Dir != ClientToAgent || frames[1].Dir != AgentToClient {
		t.Fatalf("frames = %+v, want one request and one response", frames)
	}
	rec.Golden(t, filepath.Join("testdata", "initialize.golden"))
}

// A prompt's session update reaches the default Client, and the recorder
// holds both directions in order.
func TestConnectRecordsNotifications(t *testing.T) {
	agent := &stubAgent{}
	client := &Client{}
	c := Connect(t, agent, client)
	agent.conn = c.Agent
	initialize(t, c.Client)
	if _, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: "/work", McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	resp, err := c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: "s1", Prompt: []acp.ContentBlock{acp.TextBlock("hi")}})
	if err != nil || resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("prompt = %+v, %v", resp, err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := c.Recorder.Wait(ctx, 7); err != nil {
		t.Fatal(err)
	}
	updates := client.Updates()
	if len(updates) != 1 || updates[0].Update.AgentMessageChunk == nil {
		t.Fatalf("updates = %+v", updates)
	}
	tr := c.Recorder.Transcript()
	for _, want := range []string{
		`agent→client {"jsonrpc":"2.0","method":"session/update"`,
		`client→agent {"id":3,"jsonrpc":"2.0","method":"session/prompt"`,
	} {
		if !strings.Contains(tr, want) {
			t.Errorf("transcript lacks %s:\n%s", want, tr)
		}
	}
}

func TestDefaultClientRefusals(t *testing.T) {
	c := &Client{}
	if _, err := c.ReadTextFile(t.Context(), acp.ReadTextFileRequest{}); err == nil {
		t.Fatal("ReadTextFile succeeded")
	}
	var re *acp.RequestError
	if _, err := c.CreateTerminal(t.Context(), acp.CreateTerminalRequest{}); !errors.As(err, &re) || re.Code != -32601 {
		t.Fatalf("CreateTerminal err = %v, want -32601", err)
	}
	resp, err := c.RequestPermission(t.Context(), acp.RequestPermissionRequest{})
	if err != nil || resp.Outcome.Cancelled == nil {
		t.Fatalf("RequestPermission = %+v, %v; want the cancelled outcome", resp, err)
	}
}

func TestRecorderJoinsSplitLines(t *testing.T) {
	r := &Recorder{}
	r.observe(ClientToAgent, []byte(`{"b":1,`))
	r.observe(AgentToClient, []byte("{\"x\":true}\n"))
	r.observe(ClientToAgent, []byte(`"a":2}`+"\n"+`{"c":3}`+"\n"))
	want := "agent→client {\"x\":true}\nclient→agent {\"a\":2,\"b\":1}\nclient→agent {\"c\":3}\n"
	if got := r.Transcript(); got != want {
		t.Fatalf("transcript = %q\nwant         %q", got, want)
	}
}

// fakeTB captures Golden's failure and supplies its artifact directory.
type fakeTB struct {
	testing.TB
	dir    string
	failed []string
}

func (*fakeTB) Helper()               { /* recorded failures need no helper frames */ }
func (f *fakeTB) ArtifactDir() string { return f.dir }
func (f *fakeTB) Errorf(format string, args ...any) {
	f.failed = append(f.failed, format)
	_ = args
}

// A wrong golden fails and leaves the actual transcript as an artifact
// (0004-PLAN Phase 3 Accept).
func TestGoldenWritesArtifact(t *testing.T) {
	t.Setenv("ACPTEST_UPDATE", "") // the mismatch path, even inside an update run
	client, rec := Pair(t, &stubAgent{})
	initialize(t, client)
	wrong := filepath.Join(t.TempDir(), "wrong.golden")
	if err := os.WriteFile(wrong, []byte("client→agent {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fake := &fakeTB{TB: t, dir: t.TempDir()}
	rec.Golden(fake, wrong)
	if len(fake.failed) != 1 || !strings.Contains(fake.failed[0], "differs") {
		t.Fatalf("failures = %q, want one mismatch", fake.failed)
	}
	got, err := os.ReadFile(filepath.Join(fake.dir, "wrong.golden.got"))
	if err != nil || string(got) != rec.Transcript() {
		t.Fatalf("artifact = %q, %v; want the actual transcript", got, err)
	}
	fake.failed = nil
	rec.Golden(fake, filepath.Join(t.TempDir(), "missing.golden"))
	if len(fake.failed) != 1 || !strings.Contains(fake.failed[0], "creates the golden") {
		t.Fatalf("a missing golden: failures = %q", fake.failed)
	}
}
