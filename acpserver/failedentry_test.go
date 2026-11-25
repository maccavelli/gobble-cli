package acpserver

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
)

// errScripted is a failure no wait or retry takes: an HTTP 400.
var errScripted = &llm.APIError{Status: http.StatusBadRequest, Err: errors.New("scripted, not retried")}

// shownThenFailed is a call that shows thinking and text, reports its
// usage, and fails with errScripted.
func shownThenFailed() llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.ThinkingDelta{Text: "so"}, llm.TextDelta{Text: "half"},
		llm.Usage{InputTokens: 10, OutputTokens: 3}}, Err: errScripted}
}

// lastAssistant is the last assistant message among entries.
func lastAssistant(t *testing.T, entries []session.Entry) *session.Message {
	t.Helper()
	for _, e := range slices.Backward(entries) {
		if m := e.Message; m != nil && m.Role == session.RoleAssistant {
			return m
		}
	}
	t.Fatal("no assistant entry")
	return nil
}

// A call a failure ends is written in Pi's shape: an assistant entry with
// its thinking and text, stopReason error, errorMessage the error's text,
// and its usage, priced (0070-MADR D2, D3; G18).
func TestFailedEntryShape(t *testing.T) {
	dir := t.TempDir()
	c, _ := sessionAgent(t, dir, llmtest.NewScript(shownThenFailed()), &acptest.Client{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "go"); err == nil {
		t.Fatal("the turn did not fail")
	}
	m := lastAssistant(t, savedEntries(t, dir, string(id)))
	blocks, err := m.Blocks()
	if err != nil {
		t.Fatal(err)
	}
	var kinds []string
	for _, b := range blocks {
		kinds = append(kinds, b.Type+":"+b.Thinking+b.Text)
	}
	if m.StopReason != "error" || m.ErrorMessage != errScripted.Error() || strings.Join(kinds, ",") != "thinking:so,text:half" ||
		m.Usage == nil || m.Usage.Output != 3 || m.Usage.TotalTokens != 13 {
		t.Fatalf("entry %+v blocks %v; want stopReason error, the error's text, thinking then text, its usage", m, kinds)
	}
}

// The entry of a turn that ends on a refusal of its limit keeps the text the
// live error showed, 0071's line included (0070-MADR D2 as amended, Q10;
// G41).
func TestFailedEntryKeepsLimitLine(t *testing.T) {
	h := newLimitHarness(t, SessionSettings{}, map[string]llm.Model{"fake/m1": {ContextWindow: 200000, MaxTokens: 32000}},
		apiError(400, "fake", "max_tokens: must be at most 8192 for this model", nil))
	id := h.session()
	_, err := prompt(t, h.c, id, "hello")
	if err == nil || !strings.Contains(err.Error(), "gobble asked for at most 32,000") {
		t.Fatalf("the turn answered %v, want the limit's line", err)
	}
	m := lastAssistant(t, h.live(id).tree.Path())
	if m.StopReason != "error" || !strings.Contains(m.ErrorMessage, "max_tokens: must be at most 8192") ||
		!strings.Contains(m.ErrorMessage, "; gobble asked for at most 32,000 output tokens (stated); to send less, set maxTokens for m1") {
		t.Fatalf("entry stopReason %q errorMessage %q, want the live error's text with the limit's line", m.StopReason, m.ErrorMessage)
	}
}

// A workspace refusal's entry keeps the advice the live error shows, not the
// service's words (0070-MADR D2 as amended, Q10; G41).
func TestFailedEntryKeepsWorkspaceAdvice(t *testing.T) {
	refused := &llm.APIError{Status: 403, Err: fmt.Errorf("%w: service said secret-ish things", llm.ErrWorkspace)}
	dir := t.TempDir()
	c, _ := sessionAgent(t, dir, llmtest.NewScript(llmtest.Turn{Err: refused}), &acptest.Client{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "go"); err == nil || !strings.Contains(err.Error(), workspaceAdvice) {
		t.Fatalf("the turn answered %v, want the workspace advice", err)
	}
	if m := lastAssistant(t, savedEntries(t, dir, string(id))); m.ErrorMessage != workspaceAdvice {
		t.Fatalf("errorMessage %q, want the live error's %q", m.ErrorMessage, workspaceAdvice)
	}
}

// When the failed call's entry cannot be appended, the prompt answers the
// call's error, the append's is logged at warn, and the live history is
// the file's (0070-MADR D5; G20).
func TestFailedEntryAppendFails(t *testing.T) {
	var done atomic.Bool
	var logs bytes.Buffer
	store := func(o *Options) {
		o.Store = failAt{Store: o.Store, done: &done, match: func(e session.Entry) bool {
			return e.Message != nil && e.Message.StopReason == "error"
		}}
	}
	c := promptAgent(t, t.TempDir(), llmtest.NewScript(shownThenFailed()), PromptOptions{}, &logs, store)
	id := newSession(t, c, t.TempDir(), nil).SessionId
	_, err := prompt(t, c, id, "go")
	if err == nil || !strings.Contains(err.Error(), "scripted, not retried") || strings.Contains(err.Error(), "scripted append failure") {
		t.Fatalf("the prompt answered %v, want the call's error", err)
	}
	if !done.Load() || !strings.Contains(logs.String(), "level=WARN") || !strings.Contains(logs.String(), "scripted append failure") {
		t.Fatalf("logs %q: want the append's error at warn", logs.String())
	}
}

// turnRequests runs the turns of first, then a second prompt, and returns
// the second prompt's request, as failedTurnRequests does for a turn that
// may also end well or be cancelled; during, when set, runs while the first
// prompt is under way, with the session id.
func turnRequests(t *testing.T, reload bool, first *llmtest.Script, client *orderedClient, during func(acp.SessionId, *acptest.Conn)) *llm.Request {
	t.Helper()
	dir, cwd := t.TempDir(), t.TempDir()
	if client == nil {
		client = newOrderedClient()
	}
	c, _ := sessionAgent(t, dir, first, client)
	sid := newSession(t, c, cwd, nil).SessionId
	done := startPrompt(t, c, sid, "go")
	if during != nil {
		during(sid, c)
	}
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		t.Fatal("the first prompt did not end")
	}
	second := first
	if reload {
		if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: sid}); err != nil {
			t.Fatal(err)
		}
		second = llmtest.NewScript(llmtest.Text("after"))
		c, _ = sessionAgent(t, dir, second, &acptest.Client{})
		if _, err := c.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: sid, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := prompt(t, c, sid, "again"); err != nil {
		t.Fatal(err)
	}
	reqs := second.Requests()
	return reqs[len(reqs)-1]
}

// sentOnWire reports whether text is in req as the claude wire sends it.
func sentOnWire(t *testing.T, req *llm.Request, text string) bool {
	t.Helper()
	return bytes.Contains(wireBodies(t, resumeWires[0], []*llm.Request{req})[0], []byte(text))
}

// After a failure no retry takes, the live session's next request is the
// reloaded one's, and neither sends the failed call (0070-MADR D7; G19).
func TestShownFailureHistoryIsTheFiles(t *testing.T) {
	script := func() *llmtest.Script { return llmtest.NewScript(shownThenFailed(), llmtest.Text("after")) }
	isErr := func(err error) bool { return err != nil }
	live := failedTurnRequests(t, false, script(), nil, isErr)
	reloaded := failedTurnRequests(t, true, script(), nil, isErr)
	sameRequest(t, live, reloaded)
	if sentOnWire(t, live, "half") || sentOnWire(t, reloaded, "half") {
		t.Errorf("the next request sends the failed call's text")
	}
}

// signedThinking is a thinking-only reply whose block is signed, so the
// claude wire would send it again were it not marked failed.
func signedThinking() llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.ThinkingDelta{Text: "pondering"},
		llm.ThinkingDelta{ProviderData: jsontext.Value(`{"signature":"sig-P","format":"messages"}`)}, llm.Done{StopReason: llm.StopEndTurn}}}
}

// A shown thinking-only reply run again is kept as a failed entry: the live
// history equals the reloaded one, and neither sends it (0070-MADR D4; G7).
func TestReplacedAttemptHistoryIsTheFiles(t *testing.T) {
	script := func() *llmtest.Script {
		return llmtest.NewScript(signedThinking(), llmtest.Text("ok"), llmtest.Text("after"))
	}
	live := turnRequests(t, false, script(), nil, nil)
	reloaded := turnRequests(t, true, script(), nil, nil)
	sameRequest(t, live, reloaded)
	if sentOnWire(t, live, "pondering") || sentOnWire(t, reloaded, "pondering") {
		t.Errorf("the next request sends the replaced attempt")
	}
}

// A cancel in the empty reply's wait after a shown attempt keeps it as a
// cancelled entry, aborted: the live history equals the reloaded one, and
// neither sends it (0070-MADR D2; G33).
func TestCancelInWaitHistoryIsTheFiles(t *testing.T) {
	script := func() *llmtest.Script { return llmtest.NewScript(signedThinking(), llmtest.Text("after")) }
	run := func(reload bool) *llm.Request {
		client := newOrderedClient()
		return turnRequests(t, reload, script(), client, func(id acp.SessionId, c *acptest.Conn) {
			client.waitNotice(t, "start")
			if err := c.Client.Cancel(t.Context(), acp.CancelNotification{SessionId: id}); err != nil {
				t.Fatal(err)
			}
		})
	}
	live, reloaded := run(false), run(true)
	sameRequest(t, live, reloaded)
	if sentOnWire(t, live, "pondering") || sentOnWire(t, reloaded, "pondering") {
		t.Errorf("the next request sends the cancelled attempt")
	}
}

// orderedClient is an acptest.Client that also keeps its agent chunks and
// _gobble/retry notices in one list, in the order they came, with when.
type orderedClient struct {
	*acptest.Client
	mu      sync.Mutex
	log     []logged
	notices chan string // each retry notice's phase
}

type logged struct {
	what string
	at   time.Time
}

func newOrderedClient() *orderedClient {
	return &orderedClient{Client: &acptest.Client{Allow: true}, notices: make(chan string, 16)}
}

func (o *orderedClient) SessionUpdate(ctx context.Context, n acp.SessionNotification) error {
	if c := n.Update.AgentMessageChunk; c != nil && c.Content.Text != nil {
		o.mu.Lock()
		o.log = append(o.log, logged{"chunk:" + c.Content.Text.Text, time.Now()})
		o.mu.Unlock()
	}
	return o.Client.SessionUpdate(ctx, n)
}

func (o *orderedClient) HandleExtensionMethod(ctx context.Context, method string, params jsontext.Value) (any, error) {
	if method == acpext.MethodRetry {
		var n acpext.RetryNotice
		if err := json.Unmarshal(params, &n); err == nil {
			o.mu.Lock()
			o.log = append(o.log, logged{"retry:" + string(n.Phase) + ":" + string(n.Cause), time.Now()})
			o.mu.Unlock()
			select {
			case o.notices <- string(n.Phase):
			default:
			}
		}
	}
	return o.Client.HandleExtensionMethod(ctx, method, params)
}

// waitNotice waits for a retry notice of phase.
func (o *orderedClient) waitNotice(t *testing.T, phase string) {
	t.Helper()
	timeout := time.After(10 * time.Second)
	for {
		select {
		case p := <-o.notices:
			if p == phase {
				return
			}
		case <-timeout:
			t.Fatalf("no %s retry notice", phase)
		}
	}
}

// entries is the ordered log.
func (o *orderedClient) entries() []logged {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]logged(nil), o.log...)
}

// loadReplay writes a session by first's turns over one prompt, loads it
// into a new agent, and returns the load's replay as kind:text lines.
func loadReplay(t *testing.T, first llm.Provider, cancelAt func(*acptest.Conn, acp.SessionId)) []string {
	t.Helper()
	dir, cwd := t.TempDir(), t.TempDir()
	c1, _ := sessionAgent(t, dir, first, &acptest.Client{Allow: true})
	id := newSession(t, c1, cwd, nil).SessionId
	if cancelAt != nil {
		cancelAt(c1, id)
	} else {
		_, _ = prompt(t, c1, id, "go") //nolint:errcheck // the turn's failure is the case under test
	}
	if _, err := c1.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	client := &acptest.Client{}
	c2, _ := sessionAgent(t, dir, llmtest.NewScript(), client)
	if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, n := range client.Updates() {
		switch u := n.Update; {
		case u.AgentMessageChunk != nil:
			out = append(out, "agent:"+u.AgentMessageChunk.Content.Text.Text)
		case u.ToolCall != nil:
			out = append(out, "tool:"+string(u.ToolCall.Status))
		}
	}
	return out
}

// A load replays a failed entry's text, then Pi's "Error: …" line on its
// own paragraph, or the line alone when it has no text, and each call it
// holds as failed; an aborted entry shows no line (0070-MADR D3, D8; G4,
// G23, G24, G38, G40).
func TestLoadReplaysFailedEntry(t *testing.T) {
	line := "Error: " + errScripted.Error()
	t.Run("text", func(t *testing.T) {
		got := loadReplay(t, llmtest.NewScript(shownThenFailed()), nil)
		if len(got) < 1 || got[0] != "agent:half" {
			t.Fatalf("replay %q, want the text first", got)
		}
	})
	t.Run("error_line", func(t *testing.T) {
		got := loadReplay(t, llmtest.NewScript(shownThenFailed()), nil)
		if strings.Join(got, "|") != "agent:half|agent:\n\n"+line {
			t.Fatalf("replay %q, want the text, then %q on its own paragraph", got, line)
		}
	})
	t.Run("no_text", func(t *testing.T) {
		got := loadReplay(t, llmtest.NewScript(llmtest.Turn{Err: errScripted}), nil)
		if strings.Join(got, "|") != "agent:"+line {
			t.Fatalf("replay %q, want the line alone", got)
		}
	})
	t.Run("calls", func(t *testing.T) {
		got := loadReplay(t, llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.ToolCallDone{ID: "c1", Name: "read", Args: `{"path":"x"}`}},
			Err: errScripted}), nil)
		if strings.Join(got, "|") != "agent:"+line+"|tool:failed" {
			t.Fatalf("replay %q, want the line and the call failed", got)
		}
	})
	t.Run("aborted", func(t *testing.T) {
		p := &holdAfterText{started: make(chan struct{})}
		got := loadReplay(t, p, func(c *acptest.Conn, id acp.SessionId) {
			resp, err := promptCancelled(t, c, id, p.started)
			if err != nil || resp.StopReason != acp.StopReasonCancelled {
				t.Fatalf("the cancelled turn answered %+v, %v", resp, err)
			}
		})
		if strings.Join(got, "|") != "agent:cut" {
			t.Fatalf("replay %q, want the text and no line for an aborted entry", got)
		}
	})
}

// holdAfterText streams "cut", says so on started, and holds until the
// turn is cancelled.
type holdAfterText struct{ started chan struct{} }

func (*holdAfterText) ID() string                     { return "hold" }
func (*holdAfterText) Capabilities() llm.Capabilities { return llm.Capabilities{} }
func (h *holdAfterText) Stream(ctx context.Context, _ *llm.Request) iter.Seq2[llm.Event, error] {
	return func(yield func(llm.Event, error) bool) {
		if !yield(llm.TextDelta{Text: "cut"}, nil) {
			return
		}
		close(h.started)
		<-ctx.Done()
		yield(nil, ctx.Err())
	}
}

// promptCancelled sends a prompt and cancels it once started is closed.
func promptCancelled(t *testing.T, c *acptest.Conn, id acp.SessionId, started <-chan struct{}) (acp.PromptResponse, error) {
	t.Helper()
	done := startPrompt(t, c, id, "go")
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("the turn did not start")
	}
	if err := c.Client.Cancel(t.Context(), acp.CancelNotification{SessionId: id}); err != nil {
		t.Fatal(err)
	}
	select {
	case r := <-done:
		return r.resp, r.err
	case <-time.After(10 * time.Second):
		t.Fatal("the prompt did not end")
	}
	return acp.PromptResponse{}, nil
}

// _gobble/get_last_assistant_text reads a failed entry: its text, or null
// for one with none, never the reply before it (0070-MADR D9; G26, G39).
func TestLastAssistantTextAfterFailure(t *testing.T) {
	c, _ := sessionAgent(t, t.TempDir(), llmtest.NewScript(llmtest.Text("first answer"), shownThenFailed()), &acptest.Client{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "one"); err != nil {
		t.Fatal(err)
	}
	_, _ = prompt(t, c, id, "two") //nolint:errcheck // the failure is the case under test
	if got := mustExt(t, c, "_gobble/get_last_assistant_text", map[string]any{"sessionId": string(id)})["text"]; got != "half" {
		t.Fatalf("last assistant text %v, want the failed entry's half", got)
	}
}

func TestLastAssistantTextAfterEmptyFailure(t *testing.T) {
	c, _ := sessionAgent(t, t.TempDir(), llmtest.NewScript(llmtest.Text("first answer"), llmtest.Turn{Err: errScripted}), &acptest.Client{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, c, id, "one"); err != nil {
		t.Fatal(err)
	}
	_, _ = prompt(t, c, id, "two") //nolint:errcheck // the failure is the case under test
	if got, ok := mustExt(t, c, "_gobble/get_last_assistant_text", map[string]any{"sessionId": string(id)})["text"]; !ok || got != nil {
		t.Fatalf("last assistant text %v (%v), want null for the empty failed entry", got, ok)
	}
}

// A failed entry's usage counts in the session's totals (0070-MADR D9; G28).
func TestFailedEntryCounted(t *testing.T) {
	failed := &session.Message{Role: session.RoleAssistant, StopReason: "error", ErrorMessage: "cut",
		Usage: &session.Usage{Input: 10, Output: 3, TotalTokens: 13}}
	ok := &session.Message{Role: session.RoleAssistant, StopReason: "stop", Usage: &session.Usage{Input: 20, Output: 5, TotalTokens: 25}}
	got := sessionTotals([]session.Entry{{Type: session.TypeMessage, Message: failed}, {Type: session.TypeMessage, Message: ok}})
	if got.input != 30 || got.output != 8 {
		t.Fatalf("totals in %d out %d, want the failed entry's usage counted: 30 and 8", got.input, got.output)
	}
}

// The failed entry is the leaf: the next prompt is its child (0070-MADR D9;
// G29).
func TestForkAfterFailedEntry(t *testing.T) {
	c, _ := sessionAgent(t, t.TempDir(), llmtest.NewScript(shownThenFailed(), llmtest.Text("after")), &acptest.Client{})
	id := newSession(t, c, t.TempDir(), nil).SessionId
	_, _ = prompt(t, c, id, "one") //nolint:errcheck // the failure is the case under test
	if _, err := prompt(t, c, id, "two"); err != nil {
		t.Fatal(err)
	}
	var failedID, parent string
	for _, e := range mustExt(t, c, "_gobble/get_entries", map[string]any{"sessionId": string(id)})["entries"].([]any) {
		e := e.(map[string]any)
		m, _ := e["message"].(map[string]any)
		if m == nil {
			continue
		}
		if m["stopReason"] == "error" {
			failedID, _ = e["id"].(string)
		}
		if m["role"] == "user" && failedID != "" && parent == "" {
			parent, _ = e["parentId"].(string)
		}
	}
	if failedID == "" || parent != failedID {
		t.Fatalf("the second prompt's parent %q, want the failed entry %q", parent, failedID)
	}
}
