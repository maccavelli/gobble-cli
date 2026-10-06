package acpserver

import (
	"bufio"
	"bytes"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

var testModels = ModelChoice{Provider: "fake", Models: []string{"m1", "m2"}, Default: "m1"}

// sessionAgent is an agent over the store in dir, on model p, joined to
// client. Its logs close when the test ends.
func sessionAgent(t *testing.T, dir string, p llm.Provider, client acp.Client) (*acptest.Conn, *Agent) {
	t.Helper()
	ag := New(Options{
		Version:  "1.2.3",
		Provider: func() (llm.Provider, error) { return p, nil },
		Models:   func() ModelChoice { return testModels },
		Store:    jsonl.New(dir, jsonl.Options{}),
	})
	c := acptest.Connect(t, ag, client)
	ag.SetConnection(c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, c.Client)
	return c, ag
}

func newSession(t *testing.T, c *acptest.Conn, cwd string, meta map[string]any) acp.NewSessionResponse {
	t.Helper()
	resp, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: cwd, McpServers: []acp.McpServer{}, Meta: meta})
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func sessionFiles(t *testing.T, dir string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(dir, "*", "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// A conversation is written, closed, and loaded by a new agent over the same
// store: the replay is compact, and the next prompt carries the history.
func TestLoadReplaysInANewAgent(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "notes.md"), []byte("forty-two\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	first := llmtest.NewScript(llmtest.ToolCall("r1", "read", `{"path":"notes.md"}`), llmtest.Text("it says forty-two"))
	c1, _ := sessionAgent(t, dir, first, &acptest.Client{})
	id := newSession(t, c1, cwd, nil).SessionId
	if _, err := prompt(t, c1, id, "read the notes"); err != nil {
		t.Fatal(err)
	}
	if _, err := c1.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}

	second := llmtest.NewScript(llmtest.Text("again"))
	client := &acptest.Client{}
	c2, _ := sessionAgent(t, dir, second, client)
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, n := range client.Updates() {
		u := n.Update
		switch {
		case u.UserMessageChunk != nil:
			kinds = append(kinds, "user:"+u.UserMessageChunk.Content.Text.Text)
		case u.AgentMessageChunk != nil:
			kinds = append(kinds, "agent:"+u.AgentMessageChunk.Content.Text.Text)
		case u.ToolCall != nil:
			kinds = append(kinds, fmt.Sprintf("tool:%s:%s", u.ToolCall.Title, u.ToolCall.Status))
		case u.UsageUpdate != nil:
			kinds = append(kinds, "usage")
		}
	}
	want := "user:read the notes,tool:read notes.md:completed,agent:it says forty-two,usage"
	if got := strings.Join(kinds, ","); got != want {
		t.Fatalf("replay %s\nwant   %s", got, want)
	}
	if _, err := prompt(t, c2, id, "and now?"); err != nil {
		t.Fatal(err)
	}
	llmtest.AssertTail(t, second.Requests()[0], llm.RoleUser, llm.RoleAssistant, llm.RoleTool, llm.RoleAssistant, llm.RoleUser)
}

// Resume continues the history with no replay.
func TestResumeWithoutReplay(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	c1, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Text("one")), &acptest.Client{})
	id := newSession(t, c1, cwd, nil).SessionId
	if _, err := prompt(t, c1, id, "first"); err != nil {
		t.Fatal(err)
	}
	if _, err := c1.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	second := llmtest.NewScript(llmtest.Text("two"))
	client := &acptest.Client{}
	c2, _ := sessionAgent(t, dir, second, client)
	if _, err := c2.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd}); err != nil {
		t.Fatal(err)
	}
	if n := len(client.Updates()); n != 0 {
		t.Fatalf("resume sent %d updates", n)
	}
	// The count and title the CLI prints (deviation of 2026-10-06).
	_, err := c2.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id})
	if err != nil {
		t.Fatal(err)
	}
	resumed, err := c2.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if g, _ := resumed.Meta["gobble"].(map[string]any); g == nil || g["title"] != "first" || g["messages"] != float64(2) { //nolint:errcheck // checked by the nil test
		t.Fatalf("resume _meta %v, want title first and 2 messages", resumed.Meta)
	}
	if _, err := prompt(t, c2, id, "second"); err != nil {
		t.Fatal(err)
	}
	llmtest.AssertTail(t, second.Requests()[0], llm.RoleUser, llm.RoleAssistant, llm.RoleUser)
}

func TestListSessions(t *testing.T) {
	dir, cwd, other := t.TempDir(), t.TempDir(), t.TempDir()
	c, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b")), &acptest.Client{})
	a := newSession(t, c, cwd, nil).SessionId
	b := newSession(t, c, other, map[string]any{"gobble": map[string]any{"name": "named\nsession"}}).SessionId
	live := newSession(t, c, cwd, nil).SessionId // no prompt: no file, still listed
	for _, id := range []acp.SessionId{a, b} {
		if _, err := prompt(t, c, id, "prompt of "+string(id)[:4]); err != nil {
			t.Fatal(err)
		}
	}
	resp, err := c.Client.ListSessions(t.Context(), acp.ListSessionsRequest{Cwd: &cwd})
	if err != nil {
		t.Fatal(err)
	}
	ids := map[acp.SessionId]acp.SessionInfo{}
	for _, s := range resp.Sessions {
		ids[s.SessionId] = s
	}
	if len(resp.Sessions) != 2 || ids[a].Title == nil || *ids[a].Title != "prompt of "+string(a)[:4] || ids[a].UpdatedAt == nil {
		t.Fatalf("list for cwd: %+v", resp.Sessions)
	}
	if _, ok := ids[live]; !ok {
		t.Fatal("a live session without a file is not listed")
	}
	all, err := c.Client.ListSessions(t.Context(), acp.ListSessionsRequest{})
	if err != nil {
		t.Fatal(err)
	}
	var named *acp.SessionInfo
	for i := range all.Sessions {
		if all.Sessions[i].SessionId == b {
			named = &all.Sessions[i]
		}
	}
	if len(all.Sessions) != 3 || named == nil || named.Title == nil || *named.Title != "named session" {
		t.Fatalf("list all: %+v", all.Sessions)
	}
}

// A header that cannot be read fails closed with an ACP error; a malformed
// middle line is skipped and the load succeeds.
func TestCorruptSessions(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	c, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Text("a")), &acptest.Client{})
	id := newSession(t, c, cwd, nil).SessionId
	if _, err := prompt(t, c, id, "x"); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	path := sessionFiles(t, dir)[0]
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{broken\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: id, Cwd: cwd}); err != nil {
		t.Fatalf("a malformed line failed the load: %v", err)
	}
	if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("not a header\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}})
	if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32602 || !strings.Contains(err.Error(), "not a valid session") {
		t.Fatalf("bad header: %v", err)
	}
}

// set_config_option changes the next request's model and thinking; a load
// in a new agent continues with them, with no set_config_option call.
func TestConfigOptionsPersist(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	first := llmtest.NewScript(llmtest.Text("a"))
	c1, _ := sessionAgent(t, dir, first, &acptest.Client{})
	resp := newSession(t, c1, cwd, nil)
	if len(resp.ConfigOptions) != 2 || resp.ConfigOptions[0].Select.CurrentValue != "m1" || resp.ConfigOptions[1].Select.CurrentValue != "off" {
		t.Fatalf("config options %+v", resp.ConfigOptions)
	}
	for _, kv := range [][2]string{{"model", "m2"}, {"thought_level", "high"}} {
		if _, err := c1.Client.SetSessionConfigOption(t.Context(), acp.SetSessionConfigOptionRequest{ValueId: &acp.SetSessionConfigOptionValueId{
			SessionId: resp.SessionId, ConfigId: acp.SessionConfigId(kv[0]), Value: acp.SessionConfigValueId(kv[1])}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := prompt(t, c1, resp.SessionId, "x"); err != nil {
		t.Fatal(err)
	}
	if r := first.Requests()[0]; r.Model != "m2" || r.Thinking != "high" {
		t.Fatalf("request model %q thinking %q, want m2 and high", r.Model, r.Thinking)
	}
	_, err := c1.Client.SetSessionConfigOption(t.Context(), acp.SetSessionConfigOptionRequest{ValueId: &acp.SetSessionConfigOptionValueId{
		SessionId: resp.SessionId, ConfigId: "model", Value: "nope"}})
	if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32602 {
		t.Fatalf("unknown model: %v", err)
	}
	if _, err := c1.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: resp.SessionId}); err != nil {
		t.Fatal(err)
	}

	second := llmtest.NewScript(llmtest.Text("b"))
	c2, _ := sessionAgent(t, dir, second, &acptest.Client{})
	loaded, err := c2.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: resp.SessionId, Cwd: cwd})
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ConfigOptions[0].Select.CurrentValue != "m2" || loaded.ConfigOptions[1].Select.CurrentValue != "high" {
		t.Fatalf("restored options %+v", loaded.ConfigOptions)
	}
	if _, err := prompt(t, c2, resp.SessionId, "y"); err != nil {
		t.Fatal(err)
	}
	if r := second.Requests()[0]; r.Model != "m2" || r.Thinking != "high" {
		t.Fatalf("after load: model %q thinking %q", r.Model, r.Thinking)
	}
}

// session/set_model is not a method: the SDK answers -32601, asserted, not
// tolerated (0002-PLAN Phase 4 accept).
func TestSetModelIsMethodNotFound(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	done := make(chan error, 1)
	go func() {
		done <- Serve(t.Context(), inR, outW, ServeOptions{Agent: Options{Version: "1.2.3"}})
		outW.Close() //nolint:errcheck,gosec // test pipe
	}()
	if _, err := io.WriteString(inW, `{"jsonrpc":"2.0","id":7,"method":"session/set_model","params":{"sessionId":"s","modelId":"m"}}`+"\n"); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(outR).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	var resp struct {
		ID    int `json:"id"`
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(line), &resp); err != nil || resp.ID != 7 || resp.Error.Code != -32601 {
		t.Fatalf("response %s: want -32601", line)
	}
	if err := inW.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

// A second agent cannot open a session the first is writing; it is told
// the holder's pid.
func TestSecondWriterIsRefused(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	c1, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Text("a")), &acptest.Client{})
	id := newSession(t, c1, cwd, nil).SessionId
	if _, err := prompt(t, c1, id, "x"); err != nil {
		t.Fatal(err)
	}
	c2, _ := sessionAgent(t, dir, llmtest.NewScript(), &acptest.Client{})
	_, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}})
	want := fmt.Sprintf("session %s is open in another gobble process (pid %d)", id, os.Getpid())
	if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32600 || !strings.Contains(err.Error(), want) {
		t.Fatalf("second writer: %v; want invalid request with %q", err, want)
	}
}

// checkingWriter sits between the agent and its pipe. When the agent writes
// the frame that finishes a tool call, it reads the session file then,
// before the frame can reach the client, and records whether the call's
// result was already on disk (0008-MADR D19 item 6). The SDK writes each
// message with one Write, so the check sees whole frames.
type checkingWriter struct {
	w   io.Writer
	dir string

	mu      sync.Mutex
	checked map[string]bool
}

func (c *checkingWriter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte(`"sessionUpdate":"tool_call_update"`)) && bytes.Contains(p, []byte(`"status":"completed"`)) {
		var frame struct {
			Params struct {
				Update struct {
					ToolCallID string `json:"toolCallId"`
				} `json:"update"`
			} `json:"params"`
		}
		if err := json.Unmarshal(p, &frame); err == nil {
			id := frame.Params.Update.ToolCallID
			c.mu.Lock()
			c.checked[id] = resultOnDisk(c.dir, id)
			c.mu.Unlock()
		}
	}
	return c.w.Write(p)
}

// resultOnDisk reports whether the session file holds call id's toolResult.
func resultOnDisk(dir, id string) bool {
	m, err := filepath.Glob(filepath.Join(dir, "*", "*.jsonl"))
	if err != nil || len(m) != 1 {
		return false
	}
	b, err := os.ReadFile(m[0])
	if err != nil {
		return false
	}
	for line := range strings.Lines(string(b)) {
		if strings.Contains(line, `"role":"toolResult"`) && strings.Contains(line, `"toolCallId":"`+id+`"`) {
			return true
		}
	}
	return false
}

func TestResultWrittenBeforeItsUpdate(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	model := llmtest.NewScript(llmtest.ToolCall("t1", "read", `{"path":"a.txt"}`), llmtest.Text("ok"))
	ag := New(Options{Version: "1.2.3", Provider: func() (llm.Provider, error) { return model, nil }, Store: jsonl.New(dir, jsonl.Options{})})
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	check := &checkingWriter{w: toClientW, dir: dir, checked: map[string]bool{}}
	quiet := acp.WithLogger(slog.New(slog.DiscardHandler))
	ag.SetConnection(acp.NewAgentSideConnection(ag, check, toAgentR, quiet))
	client := acp.NewClientSideConnection(&acptest.Client{}, toAgentW, toClientR, quiet)
	t.Cleanup(func() {
		toAgentW.Close()  //nolint:errcheck,gosec // test pipes
		toClientW.Close() //nolint:errcheck,gosec // test pipes
		ag.Close()        //nolint:errcheck,gosec // test cleanup
	})
	if _, err := client.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber}); err != nil {
		t.Fatal(err)
	}
	s, err := client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: cwd, McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Prompt(t.Context(), acp.PromptRequest{SessionId: s.SessionId, Prompt: []acp.ContentBlock{acp.TextBlock("read")}}); err != nil {
		t.Fatal(err)
	}
	check.mu.Lock()
	defer check.mu.Unlock()
	if onDisk, seen := check.checked["t1"]; !seen || !onDisk {
		t.Fatalf("terminal update written %v, result on disk then %v", seen, onDisk)
	}
}

// session/new writes nothing, nor does a prompt refused for want of a
// credential (0005-MADR: files are created lazily).
func TestNoFileWithoutConversation(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	ag := New(Options{Version: "1.2.3", Store: jsonl.New(dir, jsonl.Options{}),
		Provider: func() (llm.Provider, error) { return nil, fmt.Errorf("%w: no model credential", llm.ErrAuth) }})
	c := acptest.Connect(t, ag, &acptest.Client{})
	ag.SetConnection(c.Agent)
	t.Cleanup(func() { ag.Close() }) //nolint:errcheck,gosec // test cleanup
	initialize(t, c.Client)
	id := newSession(t, c, cwd, nil).SessionId
	if got := sessionFiles(t, dir); len(got) != 0 {
		t.Fatalf("files after session/new: %v", got)
	}
	if _, err := prompt(t, c, id, "x"); err == nil {
		t.Fatal("prompt without a credential succeeded")
	}
	if got := sessionFiles(t, dir); len(got) != 0 {
		t.Fatalf("files after a refused prompt: %v", got)
	}
}

// _meta.gobble chooses the id, names the session, and forks another; bad
// values are invalid params.
func TestNewSessionMeta(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	model := llmtest.NewScript(llmtest.Text("orig"), llmtest.Text("forked"))
	c, ag := sessionAgent(t, dir, model, &acptest.Client{})
	resp := newSession(t, c, cwd, map[string]any{"gobble": map[string]any{"sessionId": "my-id.1", "name": "Mine"}})
	if resp.SessionId != "my-id.1" {
		t.Fatalf("id %q", resp.SessionId)
	}
	if _, err := prompt(t, c, "my-id.1", "original"); err != nil {
		t.Fatal(err)
	}
	for _, meta := range []map[string]any{{"sessionId": "my-id.1"}, {"sessionId": "-bad"}, {"forkFrom": "nope"}} {
		_, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: cwd, McpServers: []acp.McpServer{}, Meta: map[string]any{"gobble": meta}})
		if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32602 {
			t.Errorf("meta %v: %v, want invalid params", meta, err)
		}
	}
	fork := newSession(t, c, cwd, map[string]any{"gobble": map[string]any{"forkFrom": "my-id.1"}}).SessionId
	if _, err := prompt(t, c, fork, "in the fork"); err != nil {
		t.Fatal(err)
	}
	llmtest.AssertTail(t, model.Requests()[1], llm.RoleUser, llm.RoleAssistant, llm.RoleUser)
	h, _, _, err := ag.store.Read(t.Context(), session.ID(fork))
	if err != nil || !strings.HasSuffix(h.ParentSession, "_my-id.1.jsonl") || !filepath.IsAbs(h.ParentSession) {
		t.Fatalf("fork header %+v, %v", h, err)
	}
	all, err := c.Client.ListSessions(t.Context(), acp.ListSessionsRequest{Cwd: &cwd})
	if err != nil {
		t.Fatal(err)
	}
	var title string
	for _, s := range all.Sessions {
		if s.SessionId == "my-id.1" && s.Title != nil {
			title = *s.Title
		}
	}
	if title != "Mine" {
		t.Fatalf("named session's title %q", title)
	}
}

// The replay of a failed call shows it failed, and a call with no recorded
// result (a turn cut short) shows failed too.
func TestReplayToolStatuses(t *testing.T) {
	if len(replayUpdates(replayState(nil), nil, "")) != 0 {
		t.Fatal("empty entries replayed something")
	}
	ts := session.Timestamp(time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC))
	call := agentMessageWithCalls("c1", "c2")
	entries := []session.Entry{
		{Type: session.TypeMessage, ID: "e1", Message: call},
		{Type: session.TypeMessage, ID: "e2", ParentID: new("e1"), Message: toolResultMessage("c1", "bash", "boom", true, ts)},
	}
	var got []string
	for _, u := range replayUpdates(replayState(entries), nil, "") {
		if u.ToolCall != nil {
			got = append(got, string(u.ToolCall.ToolCallId)+":"+string(u.ToolCall.Status))
		}
	}
	if strings.Join(got, ",") != "c1:failed,c2:failed" {
		t.Fatalf("statuses %v", got)
	}
}

func agentMessageWithCalls(ids ...string) *session.Message {
	m := &session.Message{Role: session.RoleAssistant}
	var blocks []session.Block
	for _, id := range ids {
		blocks = append(blocks, session.Block{Type: session.BlockToolCall, ID: id, Name: "bash", Arguments: []byte(`{}`)})
	}
	if err := m.SetBlocks(blocks); err != nil {
		panic(err)
	}
	return m
}
