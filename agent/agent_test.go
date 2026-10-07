package agent_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

func user(text string) llm.Message {
	return llm.Message{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: text}}}
}

func args(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// collect runs a turn and returns its events and the End.
func collect(ctx context.Context, t *testing.T, a *agent.Agent, env tool.Env, prompt string) ([]agent.Event, agent.End, error) {
	t.Helper()
	var evs []agent.Event
	var end agent.End
	for ev, err := range a.Run(ctx, env, nil, user(prompt)) {
		if err != nil {
			return evs, end, err
		}
		evs = append(evs, ev)
		if e, ok := ev.(agent.End); ok {
			end = e
		}
	}
	return evs, end, nil
}

func TestTextTurn(t *testing.T) {
	s := llmtest.NewScript(llmtest.Text("hello"))
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), System: "be brief"})
	evs, end, err := collect(t.Context(), t, a, tool.Env{}, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if end.StopReason != agent.StopEndTurn || len(end.Messages) != 2 || end.Messages[1].Content[0].Text != "hello" || end.Usage.OutputTokens != 5 {
		t.Fatalf("end = %+v", end)
	}
	if _, ok := evs[0].(agent.TextDelta); !ok {
		t.Fatalf("first event %T", evs[0])
	}
	req := s.Requests()[0]
	llmtest.AssertSystem(t, req, "be brief")
	llmtest.AssertTools(t, req, "read", "write", "edit", "bash")
	llmtest.AssertTail(t, req, llm.RoleSystem, llm.RoleUser)
}

// A read call runs, and its result is in the next request.
func TestToolRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("the secret is 42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := llmtest.NewScript(llmtest.ToolCall("c1", "read", args(t, map[string]string{"path": "notes.md"})), llmtest.Text("it is 42"))
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Policy: denyAll{t: t}})
	evs, end, err := collect(t.Context(), t, a, tool.Env{Cwd: dir}, "what is the secret?")
	if err != nil {
		t.Fatal(err)
	}
	var start agent.ToolStart
	var done agent.ToolEnd
	for _, ev := range evs {
		switch ev := ev.(type) {
		case agent.ToolStart:
			start = ev
		case agent.ToolEnd:
			done = ev
		}
	}
	if start.Title != "read notes.md" || done.Result.IsError || !strings.Contains(done.Result.Text(), "the secret is 42") {
		t.Fatalf("start %+v, end %+v", start, done)
	}
	reqs := s.Requests()
	llmtest.AssertTail(t, reqs[1], llm.RoleUser, llm.RoleAssistant, llm.RoleTool)
	result := reqs[1].Messages[len(reqs[1].Messages)-1].Content[0]
	if result.Type != llm.ContentToolResult || result.ToolCallID != "c1" || !strings.Contains(result.Text, "the secret is 42") {
		t.Fatalf("tool result sent = %+v", result)
	}
	call := reqs[1].Messages[len(reqs[1].Messages)-2].Content[0]
	if call.Type != llm.ContentToolCall || call.ToolName != "read" || call.ToolCallID != "c1" {
		t.Fatalf("tool call sent = %+v", call)
	}
	if end.StopReason != agent.StopEndTurn || len(end.Messages) != 4 {
		t.Fatalf("end = %+v", end)
	}
}

// denyAll fails the test if asked: read is read-only and is never asked.
type denyAll struct{ t *testing.T }

func (d denyAll) Decide(_ context.Context, r permission.Rule) (permission.Decision, error) {
	d.t.Errorf("asked about %s; read-only tools are not asked", r.Tool)
	return permission.Decision{}, nil
}

type policy struct {
	allow bool
	rules []permission.Rule
}

func (p *policy) Decide(_ context.Context, r permission.Rule) (permission.Decision, error) {
	p.rules = append(p.rules, r)
	return permission.Decision{Allow: p.allow, Reason: "test"}, nil
}

func TestPolicy(t *testing.T) {
	for _, allow := range []bool{false, true} {
		dir := t.TempDir()
		s := llmtest.NewScript(llmtest.ToolCall("w1", "write", args(t, map[string]string{"path": "a.txt", "content": "x"})), llmtest.Text("ok"))
		p := &policy{allow: allow}
		a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Policy: p})
		if _, _, err := collect(t.Context(), t, a, tool.Env{Cwd: dir}, "write"); err != nil {
			t.Fatal(err)
		}
		_, statErr := os.Stat(filepath.Join(dir, "a.txt"))
		if allow == (statErr != nil) {
			t.Fatalf("allow %v: file exists %v", allow, statErr == nil)
		}
		if len(p.rules) != 1 || p.rules[0].Tool != "write" || p.rules[0].CallID != "w1" || p.rules[0].Title != "write a.txt" {
			t.Fatalf("rules = %+v", p.rules)
		}
		sent := s.Requests()[1].Messages
		result := sent[len(sent)-1].Content[0]
		if allow == result.IsError || !allow && !strings.Contains(result.Text, "declined") {
			t.Fatalf("allow %v: result sent %+v", allow, result)
		}
	}
}

type refuse struct{}

func (refuse) Decide(context.Context, permission.Rule) (permission.Decision, error) {
	return permission.Decision{Refusal: "Plan mode is read-only."}, nil
}

// A refusal made without asking is the whole result the model reads.
func TestPolicyRefusal(t *testing.T) {
	dir := t.TempDir()
	s := llmtest.NewScript(llmtest.ToolCall("w1", "write", args(t, map[string]string{"path": "a.txt", "content": "x"})), llmtest.Text("ok"))
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Policy: refuse{}})
	if _, _, err := collect(t.Context(), t, a, tool.Env{Cwd: dir}, "write"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err == nil {
		t.Fatal("a refused write ran")
	}
	sent := s.Requests()[1].Messages
	if r := sent[len(sent)-1].Content[0]; !r.IsError || r.Text != "Plan mode is read-only." {
		t.Fatalf("result %+v", r)
	}
}

func TestUnknownTool(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("x", "nope", "{}"), llmtest.Text("sorry"))
	a := agent.New(agent.Config{Provider: s})
	if _, _, err := collect(t.Context(), t, a, tool.Env{}, "go"); err != nil {
		t.Fatal(err)
	}
	sent := s.Requests()[1].Messages
	if r := sent[len(sent)-1].Content[0]; !r.IsError || !strings.Contains(r.Text, `no tool named "nope"`) {
		t.Fatalf("result %+v", r)
	}
}

// Cancelling during a running bash ends the turn as cancelled, and every
// call keeps a result.
func TestCancelDuringTool(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("b1", "bash", args(t, map[string]string{"command": "sleep 30"})))
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools()})
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	start := time.Now()
	var end agent.End
	for ev, err := range a.Run(ctx, tool.Env{Cwd: t.TempDir()}, nil, user("sleep")) {
		if err != nil {
			t.Fatal(err)
		}
		switch ev := ev.(type) {
		case agent.ToolRun:
			time.AfterFunc(100*time.Millisecond, cancel)
		case agent.End:
			end = ev
		}
	}
	if end.StopReason != agent.StopCancelled || time.Since(start) > 5*time.Second {
		t.Fatalf("end %+v after %v", end, time.Since(start))
	}
	last := end.Messages[len(end.Messages)-1]
	if last.Role != llm.RoleTool || last.Content[0].ToolCallID != "b1" || !last.Content[0].IsError {
		t.Fatalf("last message %+v", last)
	}
}

func TestMaxCalls(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("1", "read", `{"path":"x"}`), llmtest.ToolCall("2", "read", `{"path":"x"}`), llmtest.Text("never"))
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), MaxCalls: 2})
	_, end, err := collect(t.Context(), t, a, tool.Env{Cwd: t.TempDir()}, "loop")
	if err != nil || end.StopReason != agent.StopMaxTurnRequests || s.Remaining() != 1 {
		t.Fatalf("end %+v, err %v, remaining %d", end, err, s.Remaining())
	}
}

func TestProviderError(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Err: llm.ErrAuth})
	a := agent.New(agent.Config{Provider: s})
	if _, _, err := collect(t.Context(), t, a, tool.Env{}, "x"); !errors.Is(err, llm.ErrAuth) {
		t.Fatalf("err = %v, want llm.ErrAuth", err)
	}
}

// Reasoning with provider data is kept as its own block, for replay.
func TestThinkingBlocks(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{
		llm.ThinkingDelta{Text: "let me "},
		llm.ThinkingDelta{Text: "think"},
		llm.ThinkingDelta{ProviderData: jsontext.Value(`{"signature":"s"}`)},
		llm.TextDelta{Text: "done"},
		llm.Done{StopReason: llm.StopEndTurn},
	}})
	a := agent.New(agent.Config{Provider: s})
	_, end, err := collect(t.Context(), t, a, tool.Env{}, "x")
	if err != nil {
		t.Fatal(err)
	}
	c := end.Messages[1].Content
	if len(c) != 2 || c[0].Type != llm.ContentThinking || c[0].Text != "let me think" || string(c[0].ProviderData) != `{"signature":"s"}` || c[1].Text != "done" {
		t.Fatalf("assistant content %+v", c)
	}
}
