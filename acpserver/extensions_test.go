package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
)

// holdClient is a test client whose permission answer waits, so a test can
// hold a turn in the middle of a tool call and call methods meanwhile.
type holdClient struct {
	acptest.Client
	asked  chan struct{}
	answer chan bool
}

func newHoldClient() *holdClient {
	return &holdClient{asked: make(chan struct{}, 8), answer: make(chan bool, 8)}
}

func (h *holdClient) RequestPermission(ctx context.Context, r acp.RequestPermissionRequest) (acp.RequestPermissionResponse, error) {
	h.asked <- struct{}{}
	select {
	case allow := <-h.answer:
		if allow {
			return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(r.Options[0].OptionId)}, nil
		}
	case <-ctx.Done():
	}
	return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeCancelled()}, nil
}

// waitAsked waits for the held turn's permission request.
func (h *holdClient) waitAsked(t *testing.T) {
	t.Helper()
	select {
	case <-h.asked:
	case <-time.After(5 * time.Second):
		t.Fatal("no permission request within 5 s")
	}
}

type promptResult struct {
	resp acp.PromptResponse
	err  error
}

// startPrompt sends a prompt and returns where its answer arrives.
func startPrompt(t *testing.T, c *acptest.Conn, id acp.SessionId, text string) <-chan promptResult {
	t.Helper()
	done := make(chan promptResult, 1)
	go func() {
		resp, err := c.Client.Prompt(context.WithoutCancel(t.Context()), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{acp.TextBlock(text)}})
		done <- promptResult{resp, err}
	}()
	return done
}

func waitPrompt(t *testing.T, done <-chan promptResult) acp.PromptResponse {
	t.Helper()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("prompt: %v", r.err)
		}
		return r.resp
	case <-time.After(10 * time.Second):
		t.Fatal("the prompt did not end within 10 s")
	}
	return acp.PromptResponse{}
}

// ext calls a _gobble/ method and decodes its result into a map.
func ext(t *testing.T, c *acptest.Conn, method string, params map[string]any) (map[string]any, error) {
	t.Helper()
	raw, err := c.Client.CallExtension(t.Context(), method, params)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("%s result %s: %v", method, raw, err)
	}
	return out, nil
}

func mustExt(t *testing.T, c *acptest.Conn, method string, params map[string]any) map[string]any {
	t.Helper()
	out, err := ext(t, c, method, params)
	if err != nil {
		t.Fatalf("%s: %v", method, err)
	}
	return out
}

// errText is a JSON-RPC error's code and the reason or error its data
// carries.
func errText(t *testing.T, err error) (int, string) {
	t.Helper()
	var re *acp.RequestError
	if !errors.As(err, &re) {
		t.Fatalf("err = %v, want a JSON-RPC error", err)
	}
	data, _ := re.Data.(map[string]any) //nolint:errcheck // no data is no text
	for _, k := range []string{keyReason, "error"} {
		if s, ok := data[k].(string); ok {
			return re.Code, s
		}
	}
	return re.Code, ""
}

// fileMessages are the session file's messages after the header, as
// "role: text".
func fileMessages(t *testing.T, dir string, id acp.SessionId) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*", "*_"+string(id)+".jsonl"))
	if err != nil || len(files) != 1 {
		t.Fatalf("session %s files %v, %v", id, files, err)
	}
	b, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(b)), "\n")[1:] {
		var e session.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		if e.Message != nil {
			out = append(out, e.Message.Role+": "+e.Message.Text())
		}
	}
	return out
}

func writeCall(t *testing.T) llmtest.Turn {
	t.Helper()
	return llmtest.ToolCall("w1", "write", toolArgs(t, map[string]string{"path": "a.txt", "content": "x"}))
}

func lastRequestText(req *llm.Request) (llm.Role, string) {
	m := req.Messages[len(req.Messages)-1]
	if len(m.Content) == 0 {
		return m.Role, ""
	}
	return m.Role, m.Content[0].Text
}

// A steer sent while a write waits for permission enters after the write's
// result, before the next model call; it is written to the session and
// reported as user_message_chunk before the reply to it (0002-PLAN Phase 7
// Accept).
func TestSteerDuringPrompt(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	model := llmtest.NewScript(writeCall(t), llmtest.Text("ok"))
	client := newHoldClient()
	c, _ := sessionAgent(t, dir, model, client)
	id := newSession(t, c, cwd, nil).SessionId
	done := startPrompt(t, c, id, "write it")
	client.waitAsked(t)
	got := mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "  and then stop  "})
	if got["disposition"] != "queued" {
		t.Fatalf("steer = %v", got)
	}
	state := mustExt(t, c, "_gobble/get_state", map[string]any{"sessionId": id})
	if state["isStreaming"] != true || state["pendingMessageCount"] != float64(1) {
		t.Fatalf("state during the turn %v", state)
	}
	client.answer <- true
	if resp := waitPrompt(t, done); resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("stop %s", resp.StopReason)
	}
	reqs := model.Requests()
	llmtest.AssertTail(t, reqs[1], llm.RoleUser, llm.RoleAssistant, llm.RoleTool, llm.RoleUser)
	if role, text := lastRequestText(reqs[1]); role != llm.RoleUser || text != "and then stop" {
		t.Fatalf("second request ends with %s %q", role, text)
	}
	var order []string
	for _, n := range client.Updates() {
		switch u := n.Update; {
		case u.UserMessageChunk != nil && u.UserMessageChunk.Content.Text != nil:
			order = append(order, "user "+u.UserMessageChunk.Content.Text.Text)
		case u.AgentMessageChunk != nil && u.AgentMessageChunk.Content.Text != nil:
			order = append(order, "agent "+u.AgentMessageChunk.Content.Text.Text)
		}
	}
	if want := []string{"user and then stop", "agent ok"}; !slices.Equal(order, want) {
		t.Fatalf("message chunks %q, want %q", order, want)
	}
	want := []string{"user: write it", "assistant: ", "toolResult: " + toolResultText(t, cwd), "user: and then stop", "assistant: ok"}
	if got := fileMessages(t, dir, id); !slices.Equal(got, want) {
		t.Fatalf("file messages %q, want %q", got, want)
	}
}

// toolResultText is what write reports for a.txt in cwd.
func toolResultText(t *testing.T, cwd string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(cwd, "a.txt"))
	if err != nil || string(b) != "x" {
		t.Fatalf("a.txt %q, %v", b, err)
	}
	return "wrote a.txt (1 line)"
}

// A follow-up waits for the reply that asks for nothing, inside the same
// prompt; in "all" mode both steers enter at once.
func TestFollowUpAndModes(t *testing.T) {
	model := llmtest.NewScript(writeCall(t), llmtest.Text("a"), llmtest.Text("b"))
	client := newHoldClient()
	c, _ := sessionAgent(t, t.TempDir(), model, client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	mustExt(t, c, "_gobble/set_steering_mode", map[string]any{"sessionId": id, "mode": "all"})
	done := startPrompt(t, c, id, "go")
	client.waitAsked(t)
	mustExt(t, c, "_gobble/follow_up", map[string]any{"sessionId": id, "message": "f1"})
	mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "s1"})
	mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "s2"})
	client.answer <- true
	resp := waitPrompt(t, done)
	reqs := model.Requests()
	if resp.StopReason != acp.StopReasonEndTurn || len(reqs) != 3 {
		t.Fatalf("stop %s after %d requests", resp.StopReason, len(reqs))
	}
	if got := requestTexts(reqs[1]); !slices.Equal(got[len(got)-2:], []string{"s1", "s2"}) {
		t.Fatalf("second request texts end %q", got)
	}
	if role, text := lastRequestText(reqs[2]); role != llm.RoleUser || text != "f1" {
		t.Fatalf("third request ends with %s %q", role, text)
	}
	state := mustExt(t, c, "_gobble/get_state", map[string]any{"sessionId": id})
	if state["steeringMode"] != "all" || state["followUpMode"] != "one-at-a-time" || state["pendingMessageCount"] != float64(0) {
		t.Fatalf("state %v", state)
	}
}

// In one-at-a-time mode, the default, one steer enters per model call.
func TestSteerOneAtATime(t *testing.T) {
	model := llmtest.NewScript(writeCall(t), llmtest.Text("a"), llmtest.Text("b"))
	client := newHoldClient()
	c, _ := sessionAgent(t, t.TempDir(), model, client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	done := startPrompt(t, c, id, "go")
	client.waitAsked(t)
	mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "s1"})
	mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "s2"})
	client.answer <- true
	waitPrompt(t, done)
	reqs := model.Requests()
	if len(reqs) != 3 {
		t.Fatalf("%d requests, want 3", len(reqs))
	}
	if _, text := lastRequestText(reqs[1]); text != "s1" {
		t.Fatalf("second request ends with %q", text)
	}
	if _, text := lastRequestText(reqs[2]); text != "s2" {
		t.Fatalf("third request ends with %q", text)
	}
}

// clear_queue returns what was queued, and none of it reaches the model.
func TestClearQueue(t *testing.T) {
	model := llmtest.NewScript(writeCall(t), llmtest.Text("a"))
	client := newHoldClient()
	c, _ := sessionAgent(t, t.TempDir(), model, client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	done := startPrompt(t, c, id, "go")
	client.waitAsked(t)
	mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "s"})
	mustExt(t, c, "_gobble/follow_up", map[string]any{"sessionId": id, "message": "f"})
	got := mustExt(t, c, "_gobble/clear_queue", map[string]any{"sessionId": id})
	if b, err := json.Marshal(got, json.Deterministic(true)); err != nil || string(b) != `{"followUp":["f"],"steering":["s"]}` {
		t.Fatalf("clear_queue = %s, %v", b, err)
	}
	client.answer <- true
	waitPrompt(t, done)
	if reqs := model.Requests(); len(reqs) != 2 || slices.Contains(requestTexts(reqs[1]), "s") {
		t.Fatalf("%d requests, last texts %q", len(reqs), requestTexts(reqs[len(reqs)-1]))
	}
}

// A cancelled turn empties both queues into _meta.gobble.cleared, and the
// next prompt sees neither (owner's decision of 2026-10-07).
func TestCancelClearsQueue(t *testing.T) {
	model := llmtest.NewScript(writeCall(t), llmtest.Text("next"))
	client := newHoldClient()
	c, _ := sessionAgent(t, t.TempDir(), model, client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	done := startPrompt(t, c, id, "go")
	client.waitAsked(t)
	mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "s"})
	mustExt(t, c, "_gobble/follow_up", map[string]any{"sessionId": id, "message": "f"})
	if err := c.Client.Cancel(t.Context(), acp.CancelNotification{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	client.answer <- false
	resp := waitPrompt(t, done)
	if resp.StopReason != acp.StopReasonCancelled {
		t.Fatalf("stop %s", resp.StopReason)
	}
	b, err := json.Marshal(resp.Meta, json.Deterministic(true))
	if err != nil || string(b) != `{"gobble":{"cleared":{"followUp":["f"],"steering":["s"]}}}` {
		t.Fatalf("cancelled response _meta %s, %v", b, err)
	}
	if _, err := prompt(t, c, id, "again"); err != nil {
		t.Fatal(err)
	}
	if got := requestTexts(model.Requests()[1]); slices.Contains(got, "s") || slices.Contains(got, "f") || got[len(got)-1] != "again" {
		t.Fatalf("next request texts %q", got)
	}
}

// gobble's own commands and empty text are refused; other slash text is
// queued, and a steer queued while idle enters after the next prompt.
func TestQueueRefusals(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("ok"))
	c, _ := sessionAgent(t, t.TempDir(), model, &acptest.Client{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	for _, tc := range []struct{ method, message, want string }{
		{"_gobble/steer", "/compact now", `Command "/compact" cannot be queued. Send it as a prompt when the turn ends.`},
		{"_gobble/follow_up", "   ", "Message cannot be empty"},
	} {
		_, err := ext(t, c, tc.method, map[string]any{"sessionId": id, "message": tc.message})
		if code, text := errText(t, err); code != -32602 || text != tc.want {
			t.Fatalf("%s %q: %d %q, want -32602 %q", tc.method, tc.message, code, text, tc.want)
		}
	}
	if _, err := ext(t, c, "_gobble/set_steering_mode", map[string]any{"sessionId": id, "mode": "some"}); requestCode(t, err) != -32602 {
		t.Fatal("an unknown queue mode was accepted")
	}
	if _, err := ext(t, c, "_gobble/steer", map[string]any{"sessionId": "nope", "message": "x"}); requestCode(t, err) != -32602 {
		t.Fatal("an unknown session was accepted")
	}
	mustExt(t, c, "_gobble/steer", map[string]any{"sessionId": id, "message": "/usr/local/bin is missing"})
	if _, err := prompt(t, c, id, "go"); err != nil {
		t.Fatal(err)
	}
	if got := requestTexts(model.Requests()[0]); !slices.Equal(got, []string{"go", "/usr/local/bin is missing"}) {
		t.Fatalf("request texts %q", got)
	}
}

// _gobble/compact is /compact's compaction, answered with Pi's
// CompactionResult; the next prompt starts from the summary.
func TestExtensionCompact(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text(big(100)), llmtest.Text(big(100)), llmtest.Text("## Goal\nSUMMARY"), llmtest.Text("after"))
	client := &acptest.Client{}
	c := compactAgent(t, session.NewMemoryStore(), model, client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	_, err := ext(t, c, "_gobble/compact", map[string]any{"sessionId": id})
	if code, text := errText(t, err); code != -32600 || text != "Nothing to compact (session too small)" {
		t.Fatalf("empty session compact: %d %q", code, text)
	}
	u1, u2 := "one "+big(100), "two "+big(100)
	for _, p := range []string{u1, u2} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	n := len(client.Updates())
	res := mustExt(t, c, "_gobble/compact", map[string]any{"sessionId": id, "customInstructions": "keep the API"})
	if res["summary"] != "## Goal\nSUMMARY" || res["firstKeptEntryId"] == "" || res["tokensBefore"].(float64) <= 0 || res["details"] == nil {
		t.Fatalf("compact = %v", res)
	}
	after := client.Updates()[n:]
	if agentText(after) != "" || len(after) != 1 || after[0].Update.UsageUpdate == nil {
		t.Fatalf("updates of _gobble/compact: %d, text %q", len(after), agentText(after))
	}
	if p := model.Requests()[2].Messages[1].Content[0].Text; !strings.Contains(p, "Additional focus: keep the API") {
		t.Fatalf("summary prompt %q", p)
	}
	msgs := mustExt(t, c, "_gobble/get_messages", map[string]any{"sessionId": id})["messages"].([]any)
	if first := msgs[0].(map[string]any); first["role"] != "compactionSummary" || first["summary"] != "## Goal\nSUMMARY" {
		t.Fatalf("first message after compaction %v", first)
	}
	_, err = ext(t, c, "_gobble/compact", map[string]any{"sessionId": id})
	if code, text := errText(t, err); code != -32600 || text != "Already compacted" {
		t.Fatalf("second compact: %d %q", code, text)
	}
	if _, err := prompt(t, c, id, "three"); err != nil {
		t.Fatal(err)
	}
	got := requestTexts(model.Requests()[3])
	if len(got) != 4 || got[0] != "The conversation history before this point was compacted into the following summary:\n\n<summary>\n## Goal\nSUMMARY\n</summary>" || got[1] != u2 || got[3] != "three" {
		t.Fatalf("context after compact: %d messages, %.80q", len(got), got)
	}
}

// The writing methods are refused while a turn runs.
func TestWritersRefusedDuringTurn(t *testing.T) {
	model := llmtest.NewScript(writeCall(t), llmtest.Text("a"))
	client := newHoldClient()
	c, _ := sessionAgent(t, t.TempDir(), model, client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	done := startPrompt(t, c, id, "go")
	client.waitAsked(t)
	for method, params := range map[string]map[string]any{
		"_gobble/compact":          {"sessionId": id},
		"_gobble/set_session_name": {"sessionId": id, "name": "x"},
		"_gobble/fork":             {"sessionId": id, "entryId": "x"},
		"_gobble/clone":            {"sessionId": id},
	} {
		_, err := ext(t, c, method, params)
		if code, text := errText(t, err); code != -32600 || text != "a prompt is already running" {
			t.Errorf("%s during a turn: %d %q", method, code, text)
		}
	}
	client.answer <- true
	waitPrompt(t, done)
}

// set_session_name is /name's path; fork and clone write sessions that
// session/load opens.
func TestExtensionNameForkClone(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	client := &acptest.Client{}
	c, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b")), client)
	id := newSession(t, c, cwd, nil).SessionId
	for _, p := range []string{"first", "second"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	mustExt(t, c, "_gobble/set_session_name", map[string]any{"sessionId": id, "name": "  My name\n"})
	ups := client.Updates()
	if u := ups[len(ups)-1].Update.SessionInfoUpdate; u == nil || u.Title == nil || *u.Title != "My name" {
		t.Fatalf("last update %+v", ups[len(ups)-1].Update)
	}
	_, err := ext(t, c, "_gobble/set_session_name", map[string]any{"sessionId": id, "name": " "})
	if code, text := errText(t, err); code != -32602 || text != "Session name cannot be empty" {
		t.Fatalf("empty name: %d %q", code, text)
	}
	forks := mustExt(t, c, "_gobble/get_fork_messages", map[string]any{"sessionId": id})["messages"].([]any)
	if len(forks) != 2 || forks[1].(map[string]any)["text"] != "second" {
		t.Fatalf("fork messages %v", forks)
	}
	first, second := forks[0].(map[string]any)["entryId"].(string), forks[1].(map[string]any)["entryId"].(string)
	_, err = ext(t, c, "_gobble/fork", map[string]any{"sessionId": id, "entryId": first})
	if code, text := errText(t, err); code != -32602 || text != nothingBefore {
		t.Fatalf("fork before the first message: %d %q", code, text)
	}
	_, err = ext(t, c, "_gobble/fork", map[string]any{"sessionId": id, "entryId": "zz"})
	if code, text := errText(t, err); code != -32602 || text != `No user message "zz" in this session.` {
		t.Fatalf("fork of no message: %d %q", code, text)
	}
	fork := mustExt(t, c, "_gobble/fork", map[string]any{"sessionId": id, "entryId": second})
	clone := mustExt(t, c, "_gobble/clone", map[string]any{"sessionId": id})
	if fork["text"] != "second" {
		t.Fatalf("fork = %v", fork)
	}
	for _, tc := range []struct {
		res  map[string]any
		want []string
	}{{fork, []string{"first"}}, {clone, []string{"first", "second"}}} {
		nid := acp.SessionId(tc.res["sessionId"].(string))
		if _, err := c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: nid, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
			t.Fatalf("load %s: %v", nid, err)
		}
		if got := userTexts(t, dir, string(nid)); !slices.Equal(got, tc.want) {
			t.Fatalf("session %s user messages %q, want %q", nid, got, tc.want)
		}
	}
}

// The reads answer from the log and the live session.
func TestExtensionReads(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	client := &acptest.Client{}
	c, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Text("a"), llmtest.Text("  b  ")), client)
	id := newSession(t, c, cwd, map[string]any{"gobble": map[string]any{"name": "named"}}).SessionId
	if got := mustExt(t, c, "_gobble/get_last_assistant_text", map[string]any{"sessionId": id}); got["text"] != nil {
		t.Fatalf("last text of a new session %v", got)
	}
	for _, p := range []string{"first", "second"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	state := mustExt(t, c, "_gobble/get_state", map[string]any{"sessionId": id})
	model, _ := state["model"].(map[string]any) //nolint:errcheck // checked below
	switch {
	case model["provider"] != "fake" || model["id"] != "m1":
		t.Errorf("model %v", state["model"])
	case state["thinkingLevel"] != thinkingOff, state["isStreaming"] != false, state["isCompacting"] != false:
		t.Errorf("state %v", state)
	case state["sessionName"] != "named", state["messageCount"] != float64(4), state["mode"] != modeDefault, state["cwd"] != cwd:
		t.Errorf("state %v", state)
	case !strings.HasSuffix(state["sessionFile"].(string), "_"+string(id)+".jsonl"), state["autoCompactionEnabled"] != false:
		t.Errorf("state %v", state)
	}
	var roles []string
	for _, m := range mustExt(t, c, "_gobble/get_messages", map[string]any{"sessionId": id})["messages"].([]any) {
		roles = append(roles, m.(map[string]any)["role"].(string))
	}
	if !slices.Equal(roles, []string{"user", "assistant", "user", "assistant"}) {
		t.Errorf("message roles %q", roles)
	}
	all := mustExt(t, c, "_gobble/get_entries", map[string]any{"sessionId": id})
	entries := all["entries"].([]any)
	firstID := entries[0].(map[string]any)["id"].(string)
	if all["leafId"] != entries[len(entries)-1].(map[string]any)["id"] {
		t.Errorf("leafId %v", all["leafId"])
	}
	if later := mustExt(t, c, "_gobble/get_entries", map[string]any{"sessionId": id, "since": firstID})["entries"].([]any); len(later) != len(entries)-1 {
		t.Errorf("entries since the first: %d of %d", len(later), len(entries))
	}
	_, err := ext(t, c, "_gobble/get_entries", map[string]any{"sessionId": id, "since": "zz"})
	if code, text := errText(t, err); code != -32602 || text != "Entry not found: zz" {
		t.Errorf("unknown since: %d %q", code, text)
	}
	if got := mustExt(t, c, "_gobble/get_last_assistant_text", map[string]any{"sessionId": id}); got["text"] != "b" {
		t.Errorf("last assistant text %v", got)
	}
	stats := mustExt(t, c, "_gobble/get_session_stats", map[string]any{"sessionId": id})
	tokens := stats["tokens"].(map[string]any)
	if stats["userMessages"] != float64(2) || stats["assistantMessages"] != float64(2) || stats["totalMessages"] != float64(4) || stats["cost"] != float64(0) {
		t.Errorf("stats %v", stats)
	}
	usage, _ := slash(t, c, client, id, "/usage")
	got := mustExt(t, c, "_gobble/usage", map[string]any{"sessionId": id})
	if got["text"] != usage {
		t.Errorf("_gobble/usage text %q, /usage %q", got["text"], usage)
	}
	if want := "Input: " + thousands(int64(tokens["input"].(float64))) + "\n"; !strings.Contains(usage, want) {
		t.Errorf("/usage %q lacks the stats' input %q", usage, want)
	}
}

// Every method initialize advertises, read from the wire, is handled; the
// set is the frozen table; anything else is method-not-found.
func TestEveryAdvertisedExtensionAnswers(t *testing.T) {
	c, _ := sessionAgent(t, t.TempDir(), llmtest.NewScript(), &acptest.Client{})
	resp, err := c.Client.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber})
	if err != nil {
		t.Fatal(err)
	}
	meta, _ := resp.AgentCapabilities.Meta["gobble"].(map[string]any) //nolint:errcheck // checked below
	raw, _ := meta["extensions"].([]any)                              //nolint:errcheck // checked below
	var names []string
	for _, n := range raw {
		names = append(names, n.(string))
	}
	frozen := []string{
		"_gobble/clear_queue", "_gobble/clone", "_gobble/compact", "_gobble/follow_up", "_gobble/fork",
		"_gobble/get_entries", "_gobble/get_fork_messages", "_gobble/get_last_assistant_text", "_gobble/get_messages",
		"_gobble/get_session_stats", "_gobble/get_state", "_gobble/set_follow_up_mode", "_gobble/set_session_name",
		"_gobble/set_steering_mode", "_gobble/steer", "_gobble/usage",
	}
	if !slices.Equal(names, frozen) {
		t.Fatalf("advertised %q, want the frozen %q", names, frozen)
	}
	id := newSession(t, c, t.TempDir(), nil).SessionId
	for _, name := range names {
		_, err := ext(t, c, name, map[string]any{"sessionId": id, "message": "m", "mode": "all", "name": "n", "entryId": "e"})
		if err != nil && requestCode(t, err) == -32601 {
			t.Errorf("%s is advertised and answers method-not-found", name)
		}
	}
	for _, name := range []string{"_gobble/nope", "_nope"} {
		if _, err := ext(t, c, name, map[string]any{"sessionId": id}); err == nil || requestCode(t, err) != -32601 {
			t.Errorf("%s: %v, want -32601", name, err)
		}
	}
}

// claimTurn admits one claimant at a time: of 50 claims made at once on one
// session, none released until all have tried, exactly one succeeds and
// the rest are refused (0002-PLAN Phase 7, deviations 2 and 3).
func TestClaimTurnAdmitsOne(t *testing.T) {
	a := New(Options{})
	s := &liveSession{id: "s1", queue: newQueue()}
	const n = 50
	var (
		start, tried sync.WaitGroup
		mu           sync.Mutex
		won          int
		refused      []error
	)
	start.Add(1)
	tried.Add(n)
	var all sync.WaitGroup
	for range n {
		all.Go(func() {
			start.Wait()
			_, release, err := a.claimTurn(t.Context(), s)
			mu.Lock()
			if err == nil {
				won++
			} else {
				refused = append(refused, err)
			}
			mu.Unlock()
			tried.Done()
			tried.Wait()
			if release != nil {
				release()
			}
		})
	}
	start.Done()
	all.Wait()
	if won != 1 || len(refused) != n-1 {
		t.Fatalf("%d of %d claims succeeded, want 1", won, n)
	}
	for _, err := range refused {
		if code, text := errText(t, err); code != -32600 || text != "a prompt is already running" {
			t.Fatalf("refusal %d %q", code, text)
		}
	}
	if s.cancel != nil {
		t.Fatal("the turn was not released")
	}
}

// set_config_option, get_state and prompts at once share the model and
// thinking level under the agent's lock (0002-PLAN Phase 7, deviation 1);
// go test -race fails this without it.
func TestModelChoiceUnderLock(t *testing.T) {
	turns := make([]llmtest.Turn, 10)
	for i := range turns {
		turns[i] = llmtest.Text("ok")
	}
	c, _ := sessionAgent(t, t.TempDir(), llmtest.NewScript(turns...), &acptest.Client{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	var wg sync.WaitGroup
	wg.Go(func() {
		for i := range 30 {
			model := []string{"m1", "m2"}[i%2]
			if _, err := c.Client.SetSessionConfigOption(t.Context(), acp.SetSessionConfigOptionRequest{ValueId: &acp.SetSessionConfigOptionValueId{
				SessionId: id, ConfigId: acp.SessionConfigId(configModel), Value: acp.SessionConfigValueId(model)}}); err != nil {
				t.Error(err)
			}
		}
	})
	wg.Go(func() {
		for range 30 {
			if _, err := ext(t, c, "_gobble/get_state", map[string]any{"sessionId": id}); err != nil {
				t.Error(err)
			}
		}
	})
	wg.Go(func() {
		for range 10 {
			if _, err := c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{acp.TextBlock("x")}}); err != nil {
				t.Error(err)
			}
		}
	})
	wg.Wait()
}
