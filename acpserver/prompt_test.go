package acpserver

import (
	"bufio"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// echoResult is a model that calls one tool, then answers with the result
// it was sent, so a test sees data flow from the tool to the reply.
type echoResult struct {
	mu   sync.Mutex
	call llm.ToolCallDone
	n    int
}

func (*echoResult) ID() string                     { return "echo-result" }
func (*echoResult) Capabilities() llm.Capabilities { return llm.Capabilities{} }

func (e *echoResult) Stream(_ context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	e.mu.Lock()
	e.n++
	first := e.n == 1
	e.mu.Unlock()
	return func(yield func(llm.Event, error) bool) {
		if first {
			if yield(e.call, nil) {
				yield(llm.Done{StopReason: llm.StopToolUse}, nil)
			}
			return
		}
		last := req.Messages[len(req.Messages)-1].Content[0]
		for _, ev := range []llm.Event{
			llm.TextDelta{Text: "The tool said: " + last.Text},
			llm.Usage{InputTokens: 30, OutputTokens: 7},
			llm.Done{StopReason: llm.StopEndTurn},
		} {
			if !yield(ev, nil) {
				return
			}
		}
	}
}

func toolArgs(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// openSession opens a session in cwd.
func openSession(t *testing.T, c *acptest.Conn, cwd string) acp.SessionId {
	t.Helper()
	initialize(t, c.Client)
	s, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: cwd, McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	return s.SessionId
}

func prompt(t *testing.T, c *acptest.Conn, id acp.SessionId, text string) (acp.PromptResponse, error) {
	t.Helper()
	return c.Client.Prompt(t.Context(), acp.PromptRequest{SessionId: id, Prompt: []acp.ContentBlock{acp.TextBlock(text)}})
}

// The tool calls of a turn, keyed by id, with each update in order.
func toolUpdates(updates []acp.SessionNotification) (starts []*acp.SessionUpdateToolCall, ups []*acp.SessionToolCallUpdate) {
	for _, n := range updates {
		if n.Update.ToolCall != nil {
			starts = append(starts, n.Update.ToolCall)
		}
		if n.Update.ToolCallUpdate != nil {
			ups = append(ups, n.Update.ToolCallUpdate)
		}
	}
	return starts, ups
}

func agentText(updates []acp.SessionNotification) string {
	var b strings.Builder
	for _, n := range updates {
		if c := n.Update.AgentMessageChunk; c != nil && c.Content.Text != nil {
			b.WriteString(c.Content.Text.Text)
		}
	}
	return b.String()
}

// A prompt needing read of a temp file gives a tool_call, then a terminal
// update, and a final message holding the file's contents (Phase 3 Accept).
func TestReadReachesTheReply(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("the answer is 42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	model := &echoResult{call: llm.ToolCallDone{ID: "r1", Name: "read", Args: toolArgs(t, map[string]string{"path": "notes.md"})}}
	c, client := connectWith(t, model, &acptest.Client{})
	id := openSession(t, c, dir)
	resp, err := prompt(t, c, id, "what is the answer?")
	if err != nil || resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("prompt = %+v, %v", resp, err)
	}
	updates := client.Updates()
	starts, ups := toolUpdates(updates)
	if len(starts) != 1 || starts[0].Title != "read notes.md" || starts[0].Status != acp.ToolCallStatusPending || starts[0].Kind != acp.ToolKindRead {
		t.Fatalf("tool_call = %+v", starts)
	}
	if len(ups) != 2 || *ups[0].Status != acp.ToolCallStatusInProgress || *ups[1].Status != acp.ToolCallStatusCompleted {
		t.Fatalf("tool_call_updates = %+v", ups)
	}
	if first := ups[1].Content[0].Content; first == nil || first.Content.Text == nil || first.Content.Text.Text != "read notes.md (1 line)" {
		t.Fatalf("first content block %+v, want the summary", ups[1].Content)
	}
	if got := agentText(updates); !strings.Contains(got, "the answer is 42") {
		t.Fatalf("reply %q lacks the file's contents", got)
	}
	if len(client.Permissions()) != 0 {
		t.Fatal("read asked permission; it is read-only")
	}
	//lint:ignore SA1019 acp-go-sdk2 keeps this unstable-schema name under v1 (0035-MADR D6)
	if resp.Usage == nil || resp.Usage.TotalTokens != 37 { //nolint:staticcheck // as the lint:ignore above
		//lint:ignore SA1019 acp-go-sdk2 keeps this unstable-schema name under v1 (0035-MADR D6)
		t.Fatalf("response usage %+v", resp.Usage) //nolint:staticcheck // as the lint:ignore above
	}
}

// A session's commands see the 0003 markers for that session (0005-PLAN
// F1b).
func TestCommandsSeeTheMarkers(t *testing.T) {
	model := &echoResult{call: llm.ToolCallDone{ID: "b1", Name: "bash", Args: toolArgs(t, map[string]string{
		"command": `echo "[$AI_AGENT] [$GOBBLE_SESSION_ID] [$GOBBLE_THINKING_LEVEL]"`,
	})}}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash on PATH")
	}
	c, client := connectWith(t, model, &acptest.Client{Allow: true})
	id := openSession(t, c, t.TempDir())
	resp, err := prompt(t, c, id, "run it")
	if err != nil || resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("prompt = %+v, %v", resp, err)
	}
	if got, want := agentText(client.Updates()), fmt.Sprintf("[gobble] [%s] [off]", id); !strings.Contains(got, want) {
		t.Fatalf("reply %q lacks %q", got, want)
	}
}

func TestMarkers(t *testing.T) {
	got := strings.Join(markers("s1", "", "anthropic", "m", "off"), " ")
	if want := "AI_AGENT=gobble GOBBLE_SESSION_ID=s1 GOBBLE_PROVIDER=anthropic GOBBLE_MODEL=m GOBBLE_THINKING_LEVEL=off"; got != want {
		t.Fatalf("markers = %q, want %q (an empty file is left out)", got, want)
	}
}

// A file in an additional directory is inside the workspace: read reads it
// without a permission request. The same file without the directory is
// outside, and the client is asked. A relative directory is refused
// (0005-PLAN F1a).
func TestAdditionalDirectories(t *testing.T) {
	base := t.TempDir()
	work, extra := filepath.Join(base, "work"), filepath.Join(base, "extra")
	for _, d := range []string{work, extra} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	notes := filepath.Join(extra, "notes.md")
	if err := os.WriteFile(notes, []byte("the answer is 42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run := func(dirs []string) (*acptest.Client, string) {
		t.Helper()
		model := &echoResult{call: llm.ToolCallDone{ID: "r1", Name: "read", Args: toolArgs(t, map[string]string{"path": notes})}}
		c, client := connectWith(t, model, &acptest.Client{})
		initialize(t, c.Client)
		s, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: work, AdditionalDirectories: dirs, McpServers: []acp.McpServer{}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := prompt(t, c, s.SessionId, "what is the answer?"); err != nil {
			t.Fatal(err)
		}
		return client, agentText(client.Updates())
	}
	client, reply := run([]string{extra})
	if len(client.Permissions()) != 0 || !strings.Contains(reply, "the answer is 42") {
		t.Fatalf("with the directory: %d permission requests, reply %q", len(client.Permissions()), reply)
	}
	client, reply = run(nil)
	if len(client.Permissions()) != 1 || strings.Contains(reply, "the answer is 42") {
		t.Fatalf("without the directory: %d permission requests, reply %q; want one, refused", len(client.Permissions()), reply)
	}

	c, _ := connect(t)
	initialize(t, c.Client)
	_, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: work, AdditionalDirectories: []string{"relative/dir"}, McpServers: []acp.McpServer{}})
	if err == nil || !strings.Contains(err.Error(), "not an absolute path") {
		t.Fatalf("relative directory: err = %v, want invalid params", err)
	}
}

// Every completed prompt is followed by usage_update, before the response:
// a client that shows the context bar requires it.
func TestUsageUpdateAfterEveryPrompt(t *testing.T) {
	c, client := connectWith(t, llmtest.NewScript(llmtest.Text("one"), llmtest.Text("two")), &acptest.Client{})
	id := openSession(t, c, t.TempDir())
	for i := range 2 {
		if _, err := prompt(t, c, id, "x"); err != nil {
			t.Fatal(err)
		}
		updates := client.Updates()
		last := updates[len(updates)-1].Update.UsageUpdate
		if last == nil {
			t.Fatalf("prompt %d: last update %+v is not usage_update", i+1, updates[len(updates)-1].Update)
		}
		if last.Used == 0 || last.Size != 0 {
			t.Fatalf("usage_update %+v: want used > 0 and size 0 until the catalog", last)
		}
	}
}

// A write asks permission. Refused, the file is not written and the call
// fails; allowed, it is written.
func TestWriteAsksPermission(t *testing.T) {
	for _, allow := range []bool{false, true} {
		t.Run(fmt.Sprintf("allow=%v", allow), func(t *testing.T) {
			dir := t.TempDir()
			model := llmtest.NewScript(llmtest.ToolCall("w1", "write", toolArgs(t, map[string]string{"path": "out.txt", "content": "hi"})), llmtest.Text("done"))
			c, client := connectWith(t, model, &acptest.Client{Allow: allow})
			id := openSession(t, c, dir)
			if _, err := prompt(t, c, id, "write it"); err != nil {
				t.Fatal(err)
			}
			perms := client.Permissions()
			if len(perms) != 1 || perms[0].ToolCall.ToolCallId != "w1" || perms[0].ToolCall.Title == nil || *perms[0].ToolCall.Title != "write out.txt" {
				t.Fatalf("permission requests %+v", perms)
			}
			_, statErr := os.Stat(filepath.Join(dir, "out.txt"))
			if allow == (statErr != nil) {
				t.Fatalf("file written %v", statErr == nil)
			}
			_, ups := toolUpdates(client.Updates())
			final := ups[len(ups)-1]
			want := acp.ToolCallStatusFailed
			if allow {
				want = acp.ToolCallStatusCompleted
				if len(final.Content) < 2 || final.Content[1].Diff == nil || final.Content[1].Diff.NewText != "hi" {
					t.Fatalf("allowed write content %+v, want the summary then the diff", final.Content)
				}
			}
			if *final.Status != want {
				t.Fatalf("final status %v, want %v", *final.Status, want)
			}
		})
	}
}

// Cancelling during a blocked bash ends the turn with cancelled.
func TestCancelBlockedBash(t *testing.T) {
	model := llmtest.NewScript(llmtest.ToolCall("b1", "bash", toolArgs(t, map[string]string{"command": "sleep 30"})))
	c, client := connectWith(t, model, &acptest.Client{Allow: true})
	id := openSession(t, c, t.TempDir())
	done := make(chan acp.PromptResponse, 1)
	go func() {
		resp, err := prompt(t, c, id, "sleep")
		if err != nil {
			t.Error(err)
		}
		done <- resp
	}()
	deadline := time.Now().Add(5 * time.Second)
	for {
		_, ups := toolUpdates(client.Updates())
		if len(ups) > 0 && *ups[0].Status == acp.ToolCallStatusInProgress {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("bash never started")
		}
		time.Sleep(10 * time.Millisecond)
	}
	start := time.Now()
	if err := c.Client.Cancel(t.Context(), acp.CancelNotification{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	select {
	case resp := <-done:
		if resp.StopReason != acp.StopReasonCancelled {
			t.Fatalf("stop reason %q, want cancelled", resp.StopReason)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the turn did not end within 5 s of session/cancel")
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("cancel took %v", d)
	}
}

// Titles are at most 60 runes and a terminal update's first block at most
// 400 bytes, however long the command and its output (D19 item 3).
func TestPhoneLegibleBounds(t *testing.T) {
	long := "echo " + strings.Repeat("abcdefghij ", 20) + "&& for i in $(seq 1 200); do echo line-$i-" + strings.Repeat("x", 40) + "; done"
	model := llmtest.NewScript(llmtest.ToolCall("b1", "bash", toolArgs(t, map[string]string{"command": long})), llmtest.Text("ok"))
	c, client := connectWith(t, model, &acptest.Client{Allow: true})
	id := openSession(t, c, t.TempDir())
	if _, err := prompt(t, c, id, "go"); err != nil {
		t.Fatal(err)
	}
	starts, ups := toolUpdates(client.Updates())
	if n := utf8.RuneCountInString(starts[0].Title); n > maxTitle {
		t.Fatalf("title is %d runes", n)
	}
	final := ups[len(ups)-1]
	first := final.Content[0].Content.Content.Text.Text
	if len(first) > maxSummary {
		t.Fatalf("first content block is %d bytes", len(first))
	}
	if len(final.Content) < 2 {
		t.Fatalf("the output does not follow the summary: %+v", final.Content)
	}
}

// With no credential, initialize and session/new succeed and a prompt
// fails with auth_required.
func TestNoCredentialIsAuthRequired(t *testing.T) {
	agent := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func(string, llm.Retry) (llm.Provider, error) {
		return nil, fmt.Errorf("%w: no model credential", llm.ErrAuth)
	}})
	c := acptest.Connect(t, agent, &acptest.Client{})
	agent.SetConnection(c.Agent)
	id := openSession(t, c, t.TempDir())
	_, err := prompt(t, c, id, "x")
	re, ok := errors.AsType[*acp.RequestError](err)
	if !ok || re.Code != -32000 {
		t.Fatalf("err = %v, want auth_required (-32000)", err)
	}
}

// A second prompt while one runs is refused.
func TestOnePromptAtATime(t *testing.T) {
	model := llmtest.NewScript(llmtest.ToolCall("b1", "bash", toolArgs(t, map[string]string{"command": "sleep 30"})))
	c, client := connectWith(t, model, &acptest.Client{Allow: true})
	id := openSession(t, c, t.TempDir())
	go func() { _, _ = prompt(t, c, id, "first") }() //nolint:errcheck // cancelled below
	// session/new's frames arrive first; the first prompt is running once
	// its tool call is reported.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if starts, _ := toolUpdates(client.Updates()); len(starts) > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	_, err := prompt(t, c, id, "second")
	if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32600 {
		t.Fatalf("second prompt err = %v, want invalid request", err)
	}
	if err := c.Client.Cancel(t.Context(), acp.CancelNotification{SessionId: id}); err != nil {
		t.Fatal(err)
	}
}

// A turn whose model reads a Latin-1 file completes: the next request
// carries the result, and the saved result holds U+FFFD for the invalid
// byte (0039-MADR defect 1; Confirmation 1).
func TestTurnReadsInvalidUTF8File(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(cwd, "notes.txt"), "caf\xe9 au lait\n")
	model := llmtest.NewScript(llmtest.ToolCall("r1", "read", `{"path":"notes.txt"}`), llmtest.Text("done"))
	c, _ := readAgent(t, dir, model)
	id := newSession(t, c, cwd, nil).SessionId
	resp, err := prompt(t, c, id, "read it")
	if err != nil || resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("prompt: %+v, %v", resp, err)
	}
	if reqs := model.Requests(); len(reqs) != 2 || !strings.Contains(requestToolResults(reqs[1]), "au lait") {
		t.Fatalf("the model's next request carries no read result (%d requests)", len(reqs))
	}
	if got := savedToolResult(t, savedEntries(t, dir, string(id)), "r1"); !strings.Contains(got, "caf"+replacement+" au lait") {
		t.Fatalf("saved result %q, want U+FFFD for the invalid byte", got)
	}
}

// A tool call cut off mid-arguments is sent with its text as rawInput, and
// the turn goes on to its reply (0039-MADR defect 3, D9; Confirmation 6).
func TestCutOffToolCallArguments(t *testing.T) {
	dir, cwd := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(cwd, "a.txt"), "hello\n")
	model := llmtest.NewScript(llmtest.ToolCall("r1", "read", `{"path":"a.tx`), llmtest.Text("after the call"))
	c, client := readAgent(t, dir, model)
	id := newSession(t, c, cwd, nil).SessionId
	resp, err := prompt(t, c, id, "go")
	if err != nil || resp.StopReason != acp.StopReasonEndTurn {
		t.Fatalf("prompt: %+v, %v", resp, err)
	}
	ups := client.Updates()
	starts, results := toolUpdates(ups)
	if len(starts) != 1 || len(results) == 0 {
		t.Fatalf("tool_call %d, tool_call_update %d; want the call and its result", len(starts), len(results))
	}
	var raw string
	if err := json.Unmarshal(starts[0].RawInput, &raw); err != nil || raw != `{"path":"a.tx` {
		t.Fatalf("rawInput %s, %v; want the arguments as a JSON string", starts[0].RawInput, err)
	}
	if got := agentText(ups); got != "after the call" {
		t.Fatalf("agent text %q, want the reply", got)
	}
}

// frameOf is the one JSON-RPC frame send writes on an agent-side
// connection, without its newline.
func frameOf(t *testing.T, send func(*acp.AgentSideConnection) error) string {
	t.Helper()
	toClientR, toClientW := io.Pipe()
	toAgentR, toAgentW := io.Pipe()
	t.Cleanup(func() {
		toAgentW.Close()  //nolint:errcheck,gosec // test cleanup
		toClientR.Close() //nolint:errcheck,gosec // test cleanup
	})
	conn := acp.NewAgentSideConnection(&acptest.ScriptAgent{}, toClientW, toAgentR, acp.WithLogger(slog.New(slog.DiscardHandler)))
	errs := make(chan error, 1)
	go func() { errs <- send(conn) }()
	line := make(chan string, 1)
	go func() {
		b, _ := bufio.NewReader(toClientR).ReadString('\n') //nolint:errcheck // an empty frame fails the test
		line <- strings.TrimSuffix(b, "\n")
	}()
	if err := <-errs; err != nil {
		return "error: " + err.Error()
	}
	return <-line
}

// gobble's own v2 encoding of each _gobble/* notification is spliced by the
// SDK into the frame the SDK writes from the Go value, byte for byte, for
// text with <, &, U+2028 and an invalid byte (0039-MADR D13,
// Confirmation 19).
func TestNotifyFramesUnchanged(t *testing.T) {
	for _, text := range []string{"plain", "a<b&c\u2028d", "x\xffy"} {
		for _, n := range []struct {
			method string
			params any
		}{
			{methodRetry, retryNotice{SessionID: "s1", Phase: "start", Cause: "rate-limited", Attempt: 1, MaxAttempts: 3, DelayMs: 2000, Error: text}},
			{methodCompaction, compactionNotice{SessionID: "s1", Phase: phaseEnd, Reason: reasonManual, Error: text, Paused: pauseSession}},
			{methodCompaction, compactionNotice{SessionID: "s1", Phase: phaseEnd, Reason: reasonManual, TokensBefore: 1234}},
			{methodPersonality, personalityNotice{SessionID: "s1", Name: text, Level: 3}},
		} {
			want := frameOf(t, func(c *acp.AgentSideConnection) error { return c.NotifyExtension(t.Context(), n.method, n.params) })
			if !strings.HasPrefix(want, `{"jsonrpc":"2.0","method":"`+n.method+`","params":{"sessionId":"s1"`) {
				t.Fatalf("the SDK wrote %q, not a notification", want)
			}
			got := frameOf(t, func(c *acp.AgentSideConnection) error {
				u := &updates{conn: c, ctx: t.Context(), session: "s1"}
				u.notify(n.method, n.params)
				return u.err
			})
			if got != want {
				t.Errorf("%s %q: notify wrote\n%s\nthe SDK writes\n%s", n.method, text, got, want)
			}
		}
	}
}
