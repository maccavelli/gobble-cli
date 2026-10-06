package acpclient_test

import (
	"bufio"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// start runs agent in process and closes the connection when the test ends.
func start(t *testing.T, agent *acptest.ScriptAgent, opts acpclient.Options) *acpclient.Conn {
	t.Helper()
	c, err := acpclient.Start(t.Context(), agent.Serve, opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	return c
}

// collector gathers a session's updates.
type collector struct {
	mu      sync.Mutex
	updates []acpclient.Update
}

func (c *collector) on(u acpclient.Update) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.updates = append(c.updates, u)
}

func (c *collector) all() []acpclient.Update {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]acpclient.Update(nil), c.updates...)
}

func TestTranslate(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{
		`{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"hello"}}`,
		`{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":"hmm"}}`,
		`{"sessionUpdate":"agent_message_chunk","content":{"type":"image","data":"AA==","mimeType":"image/png"}}`,
		`{"sessionUpdate":"tool_call","toolCallId":"t1","title":"read a.go","status":"pending"}`,
		`{"sessionUpdate":"tool_call_update","toolCallId":"t1","status":"completed","content":[` +
			`{"type":"content","content":{"type":"text","text":"package a"}},` +
			`{"type":"diff","path":"/w/a.go","oldText":"x","newText":"y"},` +
			`{"type":"terminal","terminalId":"term-1"}]}`,
		`{"sessionUpdate":"plan","entries":[{"content":"step","priority":"high","status":"in_progress"}]}`,
		`{"sessionUpdate":"usage_update","used":1200,"size":200000,"cost":{"amount":0.25,"currency":"USD"}}`,
		`{"sessionUpdate":"available_commands_update","availableCommands":[]}`,
	}}}}
	c := start(t, agent, acpclient.Options{Name: "test", Version: "0.0.0"})
	var col collector
	s, err := c.NewSession(t.Context(), "/w", col.on)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "go"}); err != nil {
		t.Fatal(err)
	}
	got := col.all()
	wantKinds := []acpclient.Kind{acpclient.KindAgentText, acpclient.KindThought, acpclient.KindAgentText, acpclient.KindToolCall,
		acpclient.KindToolCallUpdate, acpclient.KindPlan, acpclient.KindUsage, acpclient.KindOther}
	if len(got) != len(wantKinds) {
		t.Fatalf("got %d updates, want %d: %+v", len(got), len(wantKinds), got)
	}
	for i, k := range wantKinds {
		if got[i].Kind != k {
			t.Errorf("update %d: kind %v, want %v", i, got[i].Kind, k)
		}
		if !strings.Contains(string(got[i].Params), `"sessionId":"s1"`) || !strings.Contains(string(got[i].Params), `"sessionUpdate":`) {
			t.Errorf("update %d: params %s lack the session and the kind", i, got[i].Params)
		}
	}
	if got[0].Text != "hello" || got[1].Text != "hmm" || got[2].Block != "image" {
		t.Errorf("chunks = %+v %+v %+v", got[0], got[1], got[2])
	}
	if tc := got[3].Tool; tc.ID != "t1" || tc.Title != "read a.go" || tc.Status != "pending" {
		t.Errorf("tool call = %+v", tc)
	}
	up := got[4].Tool
	if up.Title != "" || up.Status != "completed" || len(up.Content) != 3 || up.Content[0].Text != "package a" ||
		up.Content[1].Diff == nil || *up.Content[1].Diff.OldText != "x" || up.Content[1].Diff.NewText != "y" || up.Content[2].Terminal != "term-1" {
		t.Errorf("tool call update = %+v", up)
	}
	if p := got[5].Plan; len(p) != 1 || p[0] != (acpclient.PlanEntry{Content: "step", Status: "in_progress", Priority: "high"}) {
		t.Errorf("plan = %+v", p)
	}
	if u := got[6].Context; u.Used != 1200 || u.Size != 200000 || u.Cost == nil || u.Cost.Amount != 0.25 || u.Cost.Currency != "USD" {
		t.Errorf("usage = %+v", u)
	}
	if !strings.Contains(string(got[7].Params), `"available_commands_update"`) {
		t.Errorf("other update's params = %s", got[7].Params)
	}
}

// The same prompt sent by acpclient and by the SDK's own client carries the
// same params (0002-PLAN Phase 2 step 3).
func TestPromptParamsMatchSDKClient(t *testing.T) {
	const want = `{"prompt":[{"text":"hi","type":"text"}],"sessionId":"s1"}`

	ours := &acptest.ScriptAgent{Turns: []acptest.Turn{{}}}
	c := start(t, ours, acpclient.Options{})
	s, err := c.NewSession(t.Context(), "/w", func(acpclient.Update) {})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "hi"}); err != nil {
		t.Fatal(err)
	}

	theirs := &acptest.ScriptAgent{Turns: []acptest.Turn{{}}}
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	go func() {
		err := theirs.Serve(t.Context(), toAgentR, toClientW)
		toClientW.CloseWithError(err) //nolint:errcheck,gosec // always nil
	}()
	t.Cleanup(func() { toAgentW.Close() }) //nolint:errcheck,gosec // test cleanup
	sdk := acp.NewClientSideConnection(&acptest.Client{}, toAgentW, toClientR, acp.WithLogger(slog.New(slog.DiscardHandler)))
	if _, err := sdk.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber}); err != nil {
		t.Fatal(err)
	}
	sess, err := sdk.NewSession(t.Context(), acp.NewSessionRequest{Cwd: "/w", McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sdk.Prompt(t.Context(), acp.PromptRequest{SessionId: sess.SessionId, Prompt: []acp.ContentBlock{acp.TextBlock("hi")}}); err != nil {
		t.Fatal(err)
	}

	got, sdkGot := ours.Prompts(), theirs.Prompts()
	if len(got) != 1 || len(sdkGot) != 1 || got[0] != sdkGot[0] || got[0] != want {
		t.Fatalf("acpclient sent %q, the SDK client sent %q; want both %s", got, sdkGot, want)
	}
}

func TestImagesUnsupported(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{}}}
	c := start(t, agent, acpclient.Options{})
	s, err := c.NewSession(t.Context(), "/w", func(acpclient.Update) {})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Prompt(t.Context(), acpclient.Prompt{Text: "see", Images: []acpclient.Image{{MIME: "image/png", Data: []byte{1}}}})
	if !errors.Is(err, acpclient.ErrImagesUnsupported) {
		t.Fatalf("err = %v, want ErrImagesUnsupported", err)
	}
	if n := len(agent.Prompts()); n != 0 {
		t.Fatalf("%d prompts reached the agent; the image must stop the turn before it is sent", n)
	}

	withImages := &acptest.ScriptAgent{Image: true, Turns: []acptest.Turn{{}}}
	c2 := start(t, withImages, acpclient.Options{})
	s2, err := c2.NewSession(t.Context(), "/w", func(acpclient.Update) {})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s2.Prompt(t.Context(), acpclient.Prompt{Images: []acpclient.Image{{MIME: "image/png", Data: []byte{1}}}}); err != nil {
		t.Fatal(err)
	}
	if p := withImages.Prompts(); len(p) != 1 || !strings.Contains(p[0], `"data":"AQ==","mimeType":"image/png","type":"image"`) {
		t.Fatalf("prompts = %q, want one base64 image block", p)
	}
}

func TestCancel(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{WaitCancel: true}}}
	c := start(t, agent, acpclient.Options{})
	s, err := c.NewSession(t.Context(), "/w", func(acpclient.Update) {})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan acpclient.Result, 1)
	go func() {
		r, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "wait"})
		if err != nil {
			t.Error(err)
		}
		done <- r
	}()
	waitFor(t, func() bool { return len(agent.Prompts()) == 1 })
	if err := s.Cancel(t.Context()); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-done:
		if r.StopReason != "cancelled" {
			t.Fatalf("stop reason %q, want cancelled", r.StopReason)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the turn did not end after session/cancel")
	}
	if err := s.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	want := []string{"initialize", "session/new", "session/prompt", "session/cancel", "session/close"}
	if got := agent.Methods(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("methods %v, want %v", got, want)
	}
}

// A full notification queue drops the newest notification and reports it,
// and the connection keeps going.
func TestDropReported(t *testing.T) {
	updates := make([]string, 4)
	for i := range updates {
		updates[i] = `{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"x"}}`
	}
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: updates}, {}}}
	dropped := make(chan uint64, 8)
	c := start(t, agent, acpclient.Options{MaxQueued: 1, OnDrop: func(_ string, total uint64) { dropped <- total }})
	release := make(chan struct{})
	var once sync.Once
	s, err := c.NewSession(t.Context(), "/w", func(acpclient.Update) { <-release })
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "flood"})
		done <- err
	}()
	select {
	case n := <-dropped:
		if n < 1 {
			t.Fatalf("drop total %d", n)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no drop reported with a queue of 1 and a blocked handler")
	}
	once.Do(func() { close(release) })
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "still there"}); err != nil {
		t.Fatalf("the connection did not survive the drop: %v", err)
	}
	if c.Dropped() == 0 {
		t.Fatal("Dropped() = 0 after a reported drop")
	}
}

// An agent that answers initialize with another protocol is refused.
func TestStartRefusesOtherProtocols(t *testing.T) {
	serve := func(_ context.Context, in io.Reader, out io.Writer) error {
		sc := bufio.NewScanner(in)
		if sc.Scan() {
			if _, err := io.WriteString(out, `{"jsonrpc":"2.0","id":1,"result":{"protocolVersion":2,"agentCapabilities":{},"authMethods":[]}}`+"\n"); err != nil {
				return err
			}
		}
		for sc.Scan() {
		}
		return sc.Err()
	}
	_, err := acpclient.Start(t.Context(), serve, acpclient.Options{})
	if err == nil || !strings.Contains(err.Error(), "agent speaks ACP protocol 2, want 1") {
		t.Fatalf("err = %v", err)
	}
}

// Updates sent before the session/new answer reach the new session's
// callback, as Phase 6's available_commands_update will.
func TestUpdatesBeforeNewSessionReturns(t *testing.T) {
	agent := &acptest.ScriptAgent{SessionUpdates: []string{`{"sessionUpdate":"available_commands_update","availableCommands":[]}`}}
	c := start(t, agent, acpclient.Options{})
	var col collector
	if _, err := c.NewSession(t.Context(), "/w", col.on); err != nil {
		t.Fatal(err)
	}
	if got := col.all(); len(got) != 1 || !strings.Contains(string(got[0].Params), "available_commands_update") {
		t.Fatalf("updates = %+v, want the one sent before session/new returned", got)
	}
}

// auth_required (-32000) is ErrAuthRequired; another error is not.
func TestAuthRequired(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{ErrCode: -32000, ErrMessage: "Authentication required"}, {ErrCode: -32603, ErrMessage: "boom"}}}
	c := start(t, agent, acpclient.Options{})
	s, err := c.NewSession(t.Context(), "/w", func(acpclient.Update) {})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "x"}); !errors.Is(err, acpclient.ErrAuthRequired) {
		t.Fatalf("err = %v, want ErrAuthRequired", err)
	}
	if _, err := s.Prompt(t.Context(), acpclient.Prompt{Text: "x"}); err == nil || errors.Is(err, acpclient.ErrAuthRequired) {
		t.Fatalf("err = %v, want a plain failure", err)
	}
}

func TestAgentInfo(t *testing.T) {
	c := start(t, &acptest.ScriptAgent{Image: true}, acpclient.Options{})
	if a := c.Agent(); a.Name != "script" || a.Version != "0.0.0" || !a.Image || a.LoadSession {
		t.Fatalf("agent = %+v", a)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met within 5 s")
		}
		time.Sleep(5 * time.Millisecond)
	}
}
