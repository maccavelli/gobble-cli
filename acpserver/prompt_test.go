package acpserver

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	acp "github.com/coder/acp-go-sdk"

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
	if first := ups[1].Content[0].Content; first == nil || first.Content.Text == nil || first.Content.Text.Text != "read notes.md (1 lines)" {
		t.Fatalf("first content block %+v, want the summary", ups[1].Content)
	}
	if got := agentText(updates); !strings.Contains(got, "the answer is 42") {
		t.Fatalf("reply %q lacks the file's contents", got)
	}
	if len(client.Permissions()) != 0 {
		t.Fatal("read asked permission; it is read-only")
	}
	if resp.Usage == nil || resp.Usage.TotalTokens != 37 {
		t.Fatalf("response usage %+v", resp.Usage)
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
	agent := New(Options{Version: "1.2.3", NewID: counterIDs(), Provider: func() (llm.Provider, error) {
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
