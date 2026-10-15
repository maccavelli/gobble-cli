package acpserver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
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
	agent := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string) (llm.Provider, error) { return p, nil }})
	c := acptest.Connect(t, agent, client)
	agent.SetConnection(c.Agent)
	return c, client
}

// testAuth is ACP authentication over two test providers, one with a
// login and one with a key alone; only "keyed" has a credential (0005-PLAN
// F4-3c-2 deviation 1).
var testAuth = AuthOptions{
	Providers: func() []AuthProvider {
		return []AuthProvider{
			{ID: "grok", Name: "Grok (xAI)", Command: "gobble auth login grok", Login: true},
			{ID: "keyed", Name: "Keyed", Command: "gobble auth login keyed --api-key -"},
		}
	},
	Has: func(id string) bool { return id == "keyed" },
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
	// A fixed tool set: the default adds powershell on Windows, which would
	// change the usage estimate the golden records (0005-PLAN F1b,
	// deviation 1).
	// The prompt names linux and /work on every host, so the size is too
	// (0005-PLAN F3-1 deviation 5; 0019-PLAN P4 deviation 4).
	agent := New(Options{Version: "1.2.3", NewID: counterIDs(), Tools: []tool.Tool{builtin.Read(), builtin.Write(), builtin.Edit(), builtin.Bash()},
		Provider: func(string) (llm.Provider, error) { return llmtest.NewScript(llmtest.Text("hello gobble")), nil },
		Prompt:   PromptOptions{GOOS: "linux", cwd: "/work"}, Auth: testAuth})
	client := &acptest.Client{}
	c := acptest.Connect(t, agent, client)
	agent.SetConnection(c.Agent)
	resp := initialize(t, c.Client)
	if resp.ProtocolVersion != 1 {
		t.Fatalf("protocol version %d, want 1", resp.ProtocolVersion)
	}
	if resp.AgentInfo == nil || resp.AgentInfo.Name != "gobble" || resp.AgentInfo.Title == nil || *resp.AgentInfo.Title != "gobble" {
		t.Fatalf("agentInfo = %+v, want gobble/gobble", resp.AgentInfo)
	}
	sess, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: acptest.Abs("/work"), McpServers: []acp.McpServer{}})
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
// turned on MCP over streamable HTTP; 0005-PLAN F1a turned on additional
// directories.
func TestCapabilitiesAreHonest(t *testing.T) {
	c, _ := connect(t)
	got := initialize(t, c.Client).AgentCapabilities
	switch {
	case got.SessionCapabilities.AdditionalDirectories == nil:
		t.Error("additional directories are implemented (0005-PLAN F1a) and must be advertised")
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
	case !got.PromptCapabilities.EmbeddedContext:
		t.Error("resource blocks are read (0005-PLAN F3-4) and embeddedContext must be advertised")
	case got.PromptCapabilities.Image, got.PromptCapabilities.Audio:
		t.Error("image or audio advertised; llm carries neither")
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
	if _, err := c.Client.Authenticate(t.Context(), acp.AuthenticateRequest{MethodId: "x"}); requestCode(t, err) != -32602 {
		t.Fatal("authenticate with an unknown method: want -32602 (0005-PLAN F4-3c decision 4)")
	}
}

func TestDefaultIDsAreUUIDv7(t *testing.T) {
	a := New(Options{Version: "1.0.0"})
	resp, err := a.NewSession(t.Context(), acp.NewSessionRequest{Cwd: "/w"})
	if err != nil || !regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`).MatchString(string(resp.SessionId)) {
		t.Fatalf("session id %q, %v; want a UUIDv7", resp.SessionId, err)
	}
}

// served runs Serve with opts over pipes, with client on the other end, as
// `gobble acp` serves a client; the connection is initialized.
func served(t *testing.T, opts Options, client acp.Client) *acptest.Conn {
	t.Helper()
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- Serve(context.WithoutCancel(t.Context()), toAgentR, toClientW, ServeOptions{Agent: opts})
		_ = toClientW.Close() //nolint:errcheck // test pipe
	}()
	cc := acp.NewClientSideConnection(client, toAgentW, toClientR, acp.WithLogger(slog.New(slog.DiscardHandler)))
	t.Cleanup(func() {
		if err := toAgentW.Close(); err != nil {
			t.Error(err)
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Error("Serve did not return within 5 s of its input closing")
		}
	})
	initialize(t, cc)
	return &acptest.Conn{Client: cc}
}

// A second prompt during a turn is refused -32600 by gobble, and the first
// turn is not cancelled: Serve passes every prompt to the agent (0016-MADR
// A1).
func TestOverlapRefusesSecond(t *testing.T) {
	model := llmtest.NewScript(writeCall(t), llmtest.Text("a"))
	client := newHoldClient()
	c := served(t, Options{Version: "1.2.3", Provider: func(string) (llm.Provider, error) { return model, nil },
		Models: func() ModelChoice { return testModels }, Store: jsonl.New(t.TempDir(), jsonl.Options{})}, client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	done := startPrompt(t, c, id, "go")
	client.waitAsked(t)
	_, err := c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{acp.TextBlock("again")}})
	if code, text := errText(t, err); code != -32600 || text != "a prompt is already running" {
		t.Errorf("a second prompt: %d %q, want -32600 a prompt is already running", code, text)
	}
	client.answer <- true
	if resp := waitPrompt(t, done); resp.StopReason != acp.StopReasonEndTurn {
		t.Errorf("the first turn stopped %q, want end_turn: the second prompt must not cancel it", resp.StopReason)
	}
}

// slowStop is a model whose stream starts, then blocks until its context
// ends, then takes 300 ms to stop: a turn still unwinding when
// session/close arrives.
type slowStop struct{ started chan struct{} }

func (slowStop) ID() string                     { return "slow" }
func (slowStop) Capabilities() llm.Capabilities { return llm.Capabilities{} }
func (s slowStop) Stream(ctx context.Context, _ *llm.Request) iter.Seq2[llm.Event, error] {
	return func(yield func(llm.Event, error) bool) {
		s.started <- struct{}{}
		<-ctx.Done()
		time.Sleep(300 * time.Millisecond)
		yield(nil, ctx.Err())
	}
}

// session/close during a turn returns after the turn's last entry is
// written: the SDK cancels the turn, and CloseSession waits for it before
// closing the log (0016-MADR A1).
func TestCloseWaitsForTurn(t *testing.T) {
	dir := t.TempDir()
	store := jsonl.New(dir, jsonl.Options{})
	model := slowStop{started: make(chan struct{}, 1)}
	c, _ := sessionAgent(t, dir, model, &acptest.Client{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	done := startPrompt(t, c, id, "go")
	<-model.started
	start := time.Now()
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatalf("session/close: %v", err)
	}
	if took := time.Since(start); took > 2*time.Second {
		t.Errorf("session/close took %v: it waits for the turn's release, not for closeWait", took)
	}
	_, entries, _, err := store.Read(t.Context(), session.ID(id))
	if err != nil {
		t.Fatal(err)
	}
	last := entries[len(entries)-1]
	if last.Message == nil || last.Message.Role != "assistant" || last.Message.StopReason != "aborted" {
		t.Fatalf("the session's last entry after close: %s %+v, want the cancelled turn's aborted assistant message", last.Type, last.Message)
	}
	<-done
}

// The SDK's constructor binds the agent before it reads: no request finds
// it unbound (0016-MADR A1).
func TestConnAware(t *testing.T) {
	ag := New(Options{Version: "1.2.3"})
	in, _ := io.Pipe()
	conn := acp.NewAgentSideConnection(ag, io.Discard, in, acp.WithLogger(slog.New(slog.DiscardHandler)))
	t.Cleanup(func() { _ = conn.Close() }) //nolint:errcheck // test connection
	ag.mu.Lock()
	got := ag.conn
	ag.mu.Unlock()
	if got != conn {
		t.Fatalf("the agent's connection after the constructor: %p, want %p", got, conn)
	}
}

// When Serve's context ends it closes the connection, so its input is no
// longer read and its output ends (0016-MADR A1).
func TestServeClosesConnection(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, inR, outW, ServeOptions{Agent: Options{Version: "1.2.3"}}) }()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Serve: %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return within 5 s of its context ending")
	}
	if _, err := inW.Write([]byte("{}\n")); !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("a write to Serve's input after it returned: %v, want io.ErrClosedPipe", err)
	}
	if _, err := outR.Read(make([]byte, 1)); err == nil {
		t.Error("Serve's output still open after it returned")
	}
}
