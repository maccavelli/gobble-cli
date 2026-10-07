package acpserver

import (
	"bufio"
	"encoding/json/v2"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	acp "github.com/coder/acp-go-sdk"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
)

// textSince is the agent text of the updates after the first n.
func textSince(client *acptest.Client, n int) string {
	return agentText(client.Updates()[n:])
}

// slash sends one slash command and returns its text.
func slash(t *testing.T, c *acptest.Conn, client *acptest.Client, id acp.SessionId, text string) (string, acp.PromptResponse) {
	t.Helper()
	n := len(client.Updates())
	resp, err := prompt(t, c, id, text)
	if err != nil {
		t.Fatalf("%s: %v", text, err)
	}
	return textSince(client, n), resp
}

// advertised is the names of the session's available_commands_update.
func advertised(t *testing.T, client *acptest.Client) []string {
	t.Helper()
	for _, u := range client.Updates() {
		if a := u.Update.AvailableCommandsUpdate; a != nil {
			names := make([]string, len(a.AvailableCommands))
			for i, c := range a.AvailableCommands {
				names[i] = c.Name
			}
			return names
		}
	}
	t.Fatal("no available_commands_update")
	return nil
}

// Every advertised command executes when sent as a prompt: an answer,
// never "Unknown command", and never a model turn (advertisement ⊆
// handlers, 0002-MADR rule 1). The list comes from the wire, so a name
// advertised without a handler fails here.
func TestEveryAdvertisedCommandExecutes(t *testing.T) {
	model := llmtest.NewScript()
	c, client := connectWith(t, model, &acptest.Client{Allow: true})
	id := openSession(t, c, t.TempDir())
	names := advertised(t, client)
	if len(names) != len(commands) {
		t.Fatalf("advertised %v", names)
	}
	for _, name := range names {
		text, resp := slash(t, c, client, id, "/"+name)
		if resp.StopReason != acp.StopReasonEndTurn || text == "" || strings.Contains(text, "Unknown command") {
			t.Errorf("/%s: stop %s, text %q", name, resp.StopReason, text)
		}
	}
	if n := len(model.Requests()); n != 0 {
		t.Fatalf("commands made %d model requests", n)
	}
	for _, name := range names {
		if !strings.Contains(name, "/") && !slices.Contains([]string{"help", "compact", "usage", "context", "session", "model", "thinking", "mode", "plan", "name", "fork", "clone"}, name) {
			t.Errorf("unexpected command %s", name)
		}
	}
}

// /compact, /usage and /context answer with their own content and no
// model turn (0002-PLAN Phase 6 Accept); a slash that is not a command is
// prompt text.
func TestCommandsAnswer(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("hi there"), llmtest.Text("paths answered"))
	c, client := connectWith(t, model, &acptest.Client{})
	id := openSession(t, c, t.TempDir())
	if _, err := prompt(t, c, id, "hello"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ cmd, want string }{
		{"/usage", "Session usage"},
		{"/context", "Session Info"},
		{"/session", "Context: "},
		{"/compact", "Compaction failed: Nothing to compact (session too small)"},
		{"/help", "/compact [instructions] — Compact the session context"},
		{"/bogus", "Unknown command /bogus. /help lists the commands."},
	} {
		if text, _ := slash(t, c, client, id, tc.cmd); !strings.Contains(text, tc.want) {
			t.Errorf("%s: %q lacks %q", tc.cmd, text, tc.want)
		}
	}
	if n := len(model.Requests()); n != 1 {
		t.Fatalf("%d model requests after the commands, want 1", n)
	}
	if _, err := prompt(t, c, id, "/usr/local/bin is missing"); err != nil {
		t.Fatal(err)
	}
	if n := len(model.Requests()); n != 2 {
		t.Fatal("a path was taken for a command")
	}
}

// The session-start frames precede the session/new response on the wire,
// and session/load's follow its replay, before its response (2026-10-01
// amendment).
func TestStartFramesPrecedeTheResponse(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	c1, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Text("one")), &acptest.Client{})
	initialize(t, c1.Client)
	id := newSession(t, c1, cwd, nil).SessionId
	assertFramesBefore(t, c1.Recorder.Frames(), `"method":"session/new"`)
	if _, err := prompt(t, c1, id, "first"); err != nil {
		t.Fatal(err)
	}
	if _, err := c1.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	c2, _ := sessionAgent(t, dir, llmtest.NewScript(), &acptest.Client{})
	initialize(t, c2.Client)
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	assertFramesBefore(t, c2.Recorder.Frames(), `"method":"session/load"`)
}

// assertFramesBefore checks that, after the request whose frame contains
// method, the commands, mode and usage frames arrive before its response.
func assertFramesBefore(t *testing.T, frames []acptest.Frame, method string) {
	t.Helper()
	var reqID string
	at := -1
	for i, f := range frames {
		if f.Dir == acptest.ClientToAgent && strings.Contains(string(f.Raw), method) {
			var m struct {
				ID jsontextNumber `json:"id"`
			}
			if err := json.Unmarshal(f.Raw, &m); err != nil {
				t.Fatal(err)
			}
			reqID, at = string(m.ID), i
		}
	}
	if at < 0 {
		t.Fatalf("no %s request", method)
	}
	seen := map[string]bool{}
	for _, f := range frames[at+1:] {
		raw := string(f.Raw)
		for _, k := range []string{"available_commands_update", "current_mode_update", "usage_update"} {
			if strings.Contains(raw, `"sessionUpdate":"`+k+`"`) {
				seen[k] = true
			}
		}
		if f.Dir == acptest.AgentToClient && strings.Contains(raw, `"id":`+reqID+`,`) && strings.Contains(raw, `"result"`) {
			if len(seen) != 3 {
				t.Fatalf("%s answered after only %v", method, seen)
			}
			return
		}
	}
	t.Fatalf("no response to %s", method)
}

// jsontextNumber reads a JSON-RPC id as its literal text.
type jsontextNumber string

func (n *jsontextNumber) UnmarshalJSON(b []byte) error { *n = jsontextNumber(b); return nil }

// The script magic-cli-remote's acpagent sends completes with no
// unknown-method error, and set_model and an x.ai method answer -32601
// (0002-MADR fifth amendment, Confirmation changes).
func TestMcremoteScript(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	done := make(chan error, 1)
	model := llmtest.NewScript()
	go func() {
		done <- Serve(t.Context(), inR, outW, ServeOptions{Agent: Options{Version: "1.2.3", NewID: counterIDs(),
			Provider: func() (llm.Provider, error) { return model, nil }, Models: func() ModelChoice { return testModels }}})
		outW.Close() //nolint:errcheck,gosec // test pipe
	}()
	r := bufio.NewReader(outR)
	call := func(id int, method, params string) string {
		t.Helper()
		if _, err := io.WriteString(inW, `{"jsonrpc":"2.0","id":`+itoa(id)+`,"method":"`+method+`","params":`+params+"}\n"); err != nil {
			t.Fatal(err)
		}
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				t.Fatal(err)
			}
			var frame struct {
				ID     *int   `json:"id"`
				Method string `json:"method"`
			}
			if json.Unmarshal([]byte(line), &frame) == nil && frame.ID != nil && *frame.ID == id && frame.Method == "" {
				return line
			}
		}
	}
	for i, step := range []struct{ method, params string }{
		{"initialize", `{"protocolVersion":1,"clientCapabilities":{}}`},
		{"session/new", `{"cwd":"/w","mcpServers":[]}`},
		{"session/prompt", `{"sessionId":"s1","prompt":[{"type":"text","text":"/compact"}]}`},
		{"session/set_mode", `{"sessionId":"s1","modeId":"plan"}`},
		{"session/set_config_option", `{"sessionId":"s1","configId":"model","value":"m2"}`},
	} {
		if line := call(i+1, step.method, step.params); strings.Contains(line, `"error"`) {
			t.Fatalf("%s: %s", step.method, line)
		}
	}
	for i, method := range []string{"session/set_model", "_x" + ".ai/compact_conversation"} {
		if line := call(10+i, method, `{"sessionId":"s1"}`); !strings.Contains(line, `"code":-32601`) {
			t.Fatalf("%s: %s, want -32601", method, line)
		}
	}
	if err := inW.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
}

func itoa(n int) string {
	b, err := json.Marshal(n)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// No module source names an x.ai method: gobble speaks standard ACP and
// _gobble/ (0002-MADR). The needle is built so this file does not match
// itself.
func TestNoXAIMethods(t *testing.T) {
	needle := "_x" + ".ai/"
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == ".git" || d.Name() == "bin") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		b, err := os.ReadFile(path) //nolint:gosec // the module's own files
		if err != nil {
			return err
		}
		if strings.Contains(string(b), needle) {
			t.Errorf("%s names an x.ai method", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// /name sets the name with a session_info entry and reports it.
func TestNameCommand(t *testing.T) {
	c, client := connectWith(t, llmtest.NewScript(), &acptest.Client{})
	id := openSession(t, c, t.TempDir())
	if text, _ := slash(t, c, client, id, "/name"); text != "Usage: /name <name>" {
		t.Fatalf("no name: %q", text)
	}
	n := len(client.Updates())
	if text, _ := slash(t, c, client, id, "/name  my\nsession "); !strings.Contains(text, "Session name set: my session") {
		t.Fatalf("set: %q", text)
	}
	var title string
	for _, u := range client.Updates()[n:] {
		if si := u.Update.SessionInfoUpdate; si != nil && si.Title != nil {
			title = *si.Title
		}
	}
	if title != "my session" {
		t.Fatalf("session_info_update title %q", title)
	}
	if text, _ := slash(t, c, client, id, "/name"); text != "Session name: my session" {
		t.Fatalf("show: %q", text)
	}
}

// /thinking and /model change the next request, write the change, and
// report it as config_option_update.
func TestModelAndThinkingCommands(t *testing.T) {
	dir := t.TempDir()
	model := llmtest.NewScript(llmtest.Text("ok"))
	client := &acptest.Client{}
	c, _ := sessionAgent(t, dir, model, client)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	for _, tc := range []struct{ cmd, want string }{
		{"/thinking HIGH", "Thinking level: high"},
		{"/thinking loud", `Unknown thinking level "loud". Available levels: off, minimal, low, medium, high, xhigh.`},
		{"/model m2", "Model: m2"},
		{"/model nope", `Unknown model "nope". Available models: m1, m2.`},
		{"/model", "Model: m2\nModels: m1, m2"},
	} {
		if text, _ := slash(t, c, client, id, tc.cmd); text != tc.want {
			t.Errorf("%s: %q, want %q", tc.cmd, text, tc.want)
		}
	}
	updates := 0
	for _, u := range client.Updates() {
		if u.Update.ConfigOptionUpdate != nil {
			updates++
		}
	}
	if updates != 2 {
		t.Fatalf("%d config_option_updates, want 2", updates)
	}
	if _, err := prompt(t, c, id, "go"); err != nil {
		t.Fatal(err)
	}
	if r := model.Requests()[0]; r.Model != "m2" || r.Thinking != "high" {
		t.Fatalf("request model %q thinking %q", r.Model, r.Thinking)
	}
}

// /fork lists the user messages, then writes a new session holding the
// path before the chosen one; /clone copies the whole path. Each names
// the new session in _meta.gobble.switchTo, and the new session resumes.
func TestForkAndClone(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	client := &acptest.Client{}
	c, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Text("A1"), llmtest.Text("A2")), client)
	id := newSession(t, c, cwd, nil).SessionId
	for _, p := range []string{"first", "second"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	list, _ := slash(t, c, client, id, "/fork")
	var second string
	for line := range strings.SplitSeq(list, "\n") {
		if f := strings.Fields(line); len(f) == 2 && f[1] == "second" {
			second = f[0]
		}
	}
	if !strings.HasPrefix(list, "Send /fork <entry>") || second == "" {
		t.Fatalf("list %q", list)
	}
	text, resp := slash(t, c, client, id, "/fork "+second)
	forked := switchTo(t, resp)
	if text != "Forked to session "+forked+"." {
		t.Fatalf("fork: %q", text)
	}
	if got := userTexts(t, dir, forked); !slices.Equal(got, []string{"first"}) {
		t.Fatalf("forked session holds %v", got)
	}
	_, resp = slash(t, c, client, id, "/clone")
	if got := userTexts(t, dir, switchTo(t, resp)); !slices.Equal(got, []string{"first", "second"}) {
		t.Fatalf("clone holds %v", got)
	}
	if _, err := c.Client.ResumeSession(t.Context(), acp.ResumeSessionRequest{SessionId: acp.SessionId(forked), Cwd: cwd}); err != nil {
		t.Fatalf("resume the fork: %v", err)
	}
	firstID := strings.Fields(strings.Split(list, "\n")[1])[0]
	if text, _ := slash(t, c, client, id, "/fork "+firstID); text != "Nothing comes before that message; start a new session instead." {
		t.Fatalf("fork at the first message: %q", text)
	}
}

func switchTo(t *testing.T, resp acp.PromptResponse) string {
	t.Helper()
	g, _ := resp.Meta[gobble].(map[string]any) //nolint:errcheck // checked below
	id, _ := g["switchTo"].(string)            //nolint:errcheck // checked below
	if id == "" {
		t.Fatalf("response meta %v has no switchTo", resp.Meta)
	}
	return id
}

// userTexts are the user messages of the stored session id, in order.
func userTexts(t *testing.T, dir, id string) []string {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(dir, "*", "*_"+id+".jsonl"))
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
		if e.Message != nil && e.Message.Role == session.RoleUser {
			out = append(out, e.Message.Text())
		}
	}
	return out
}
