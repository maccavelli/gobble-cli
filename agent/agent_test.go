package agent_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	llmtest.AssertTools(t, req, builtin.Names()...)
	llmtest.AssertTail(t, req, llm.RoleSystem, llm.RoleUser)
}

// A read call runs, and its result is in the next request.
func TestToolRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("the secret is 42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := llmtest.NewScript(llmtest.ToolCall("c1", "read", args(t, map[string]string{"path": "notes.md"})), llmtest.Text("it is 42"))
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Policy: readOnly{t: t}})
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

// readOnly allows a call to a read-only tool naming paths only inside the
// workspace, and fails the test if asked about any other: the policy is
// asked about every call, and is told which are read-only (0005-PLAN F6-2
// decision 7).
type readOnly struct{ t *testing.T }

func (d readOnly) Decide(_ context.Context, r permission.Rule) (permission.Decision, error) {
	if !r.ReadOnly || len(r.Outside) != 0 || r.Kind != permission.SubjectPath || len(r.Subjects) == 0 {
		d.t.Errorf("asked about %+v; want a read-only call inside the workspace, with its paths", r)
		return permission.Decision{}, nil
	}
	return permission.Decision{Allow: true}, nil
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
		if len(p.rules) != 1 || p.rules[0].Tool != "write" || p.rules[0].CallID != "w1" || p.rules[0].Title != "write a.txt" ||
			p.rules[0].ReadOnly || p.rules[0].Kind != permission.SubjectPath || !slices.Equal(p.rules[0].Subjects, []string{filepath.Join(dir, "a.txt")}) {
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

// A read of a file outside the workspace roots is asked about although read
// is read-only, with the title saying so, and runs only when allowed. With
// no policy it is refused (0005-MADR "Confinement").
func TestOutsideRoots(t *testing.T) {
	base := t.TempDir()
	work, elsewhere := filepath.Join(base, "work"), filepath.Join(base, "elsewhere")
	for _, d := range []string{work, elsewhere} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	secret := filepath.Join(elsewhere, "secret.txt")
	if err := os.WriteFile(secret, []byte("the secret is 42\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := tool.Env{Cwd: work, Roots: []string{work}}
	run := func(p permission.Policy) llm.Content {
		t.Helper()
		s := llmtest.NewScript(llmtest.ToolCall("r1", "read", args(t, map[string]string{"path": secret})), llmtest.Text("ok"))
		a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Policy: p})
		if _, _, err := collect(t.Context(), t, a, env, "read it"); err != nil {
			t.Fatal(err)
		}
		sent := s.Requests()[1].Messages
		return sent[len(sent)-1].Content[0]
	}
	for _, allow := range []bool{false, true} {
		p := &policy{allow: allow}
		r := run(p)
		if len(p.rules) != 1 || p.rules[0].Tool != "read" || !strings.HasSuffix(p.rules[0].Title, " (outside the workspace)") ||
			!slices.Equal(p.rules[0].Outside, []string{secret}) || !slices.Equal(p.rules[0].Subjects, []string{secret}) {
			t.Fatalf("allow %v: rules = %+v; want one question about the outside read", allow, p.rules)
		}
		if allow != strings.Contains(r.Text, "the secret is 42") || allow == r.IsError {
			t.Fatalf("allow %v: result sent %+v", allow, r)
		}
	}
	r := run(nil)
	if !r.IsError || !strings.Contains(r.Text, "is outside the workspace") || strings.Contains(r.Text, "the secret is 42") {
		t.Fatalf("no policy: result sent %+v; want a refusal", r)
	}
	// An additional root makes the same file inside: asked as a read-only
	// call inside the workspace, and read.
	env.Roots = append(env.Roots, elsewhere)
	if r := run(readOnly{t: t}); r.IsError || !strings.Contains(r.Text, "the secret is 42") {
		t.Fatalf("inside an additional root: result sent %+v", r)
	}
}

// A shell call's subjects are its simple commands, each with its "always"
// pattern (0005-PLAN F6-3 decisions 2 and 4a); a read-only call with no
// path names the working directory; with no policy every inside call runs
// (0005-PLAN F6-2 decision 3).
func TestPolicySubjects(t *testing.T) {
	dir := t.TempDir()
	s := llmtest.NewScript(
		llmtest.ToolCall("b1", "bash", args(t, map[string]string{"command": "echo hi && git status"})),
		llmtest.ToolCall("f1", "find", args(t, map[string]string{"pattern": "*.go"})),
		llmtest.Text("ok"))
	p := &policy{}
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Policy: p})
	if _, _, err := collect(t.Context(), t, a, tool.Env{Cwd: dir}, "go"); err != nil {
		t.Fatal(err)
	}
	if len(p.rules) != 2 {
		t.Fatalf("rules = %+v", p.rules)
	}
	if b := p.rules[0]; b.Tool != "bash" || b.Kind != permission.SubjectCommand || !slices.Equal(b.Subjects, []string{"echo hi", "git status"}) ||
		!slices.Equal(b.Saves, []string{"echo *", "git status *"}) || b.ReadOnly {
		t.Errorf("bash rule = %+v", b)
	}
	if f := p.rules[1]; f.Tool != "find" || f.Kind != permission.SubjectPath || !slices.Equal(f.Subjects, []string{dir}) || !f.ReadOnly {
		t.Errorf("find rule = %+v", f)
	}
	s = llmtest.NewScript(llmtest.ToolCall("w1", "write", args(t, map[string]string{"path": "a.txt", "content": "x"})), llmtest.Text("ok"))
	if _, _, err := collect(t.Context(), t, agent.New(agent.Config{Provider: s, Tools: builtin.Tools()}), tool.Env{Cwd: dir}, "write"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err != nil {
		t.Fatalf("no policy: the write did not run: %v", err)
	}
}

// A plain shell file command runs as the native tool: its events, its
// policy question and its run are that tool's, and the model reads its
// result after a line naming it, as the answer to its bash call. With the
// native tool not offered, bash runs it (0012-MADR D10; 0005-PLAN F6-3
// decision 9).
func TestShellReroute(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("no bash on PATH")
	}
	dir := t.TempDir()
	write := func() {
		if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write()
	s := llmtest.NewScript(llmtest.ToolCall("b1", "bash", args(t, map[string]string{"command": "rm a.txt"})), llmtest.Text("ok"))
	p := &policy{allow: true}
	evs, _, err := collect(t.Context(), t, agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Policy: p}), tool.Env{Cwd: dir}, "rm it")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); err == nil {
		t.Fatal("a.txt is still there")
	}
	if len(p.rules) != 1 || p.rules[0].Tool != "delete" || p.rules[0].CallID != "b1" {
		t.Fatalf("rules %+v, want delete's question", p.rules)
	}
	var start agent.ToolStart
	for _, ev := range evs {
		if s, ok := ev.(agent.ToolStart); ok {
			start = s
		}
	}
	if start.Call.Name != "delete" || start.Call.ID != "b1" {
		t.Fatalf("start %+v", start.Call)
	}
	sent := s.Requests()[1].Messages
	if r := sent[len(sent)-1].Content[0]; r.ToolCallID != "b1" || r.ToolName != "bash" || !strings.HasPrefix(r.Text, "bash ran this as gobble's delete tool.\n") {
		t.Fatalf("result %+v", r)
	}
	write()
	without := slices.DeleteFunc(builtin.Tools(), func(tl tool.Tool) bool { return tl.Spec().Name == "delete" })
	s = llmtest.NewScript(llmtest.ToolCall("b1", "bash", args(t, map[string]string{"command": "rm a.txt"})), llmtest.Text("ok"))
	p = &policy{allow: true}
	if _, _, err := collect(t.Context(), t, agent.New(agent.Config{Provider: s, Tools: without, Policy: p}), tool.Env{Cwd: dir}, "rm it"); err != nil {
		t.Fatal(err)
	}
	if len(p.rules) != 1 || p.rules[0].Tool != "bash" {
		t.Fatalf("without delete: rules %+v, want bash's question", p.rules)
	}
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

// fakeQueue drains everything at each poll and counts the polls. Run
// yields on the caller's goroutine, so a test fills it from its event loop.
type fakeQueue struct {
	steering, followUps []llm.Message
	polls               int
}

func (q *fakeQueue) Steering() []llm.Message {
	q.polls++
	out := q.steering
	q.steering = nil
	return out
}

func (q *fakeQueue) FollowUps() []llm.Message {
	out := q.followUps
	q.followUps = nil
	return out
}

// lastText is the role and first text of a request's last message.
func lastText(req *llm.Request) (llm.Role, string) {
	m := req.Messages[len(req.Messages)-1]
	if len(m.Content) == 0 {
		return m.Role, ""
	}
	return m.Role, m.Content[0].Text
}

// runQueued runs a turn over q, calling during for each event first.
func runQueued(ctx context.Context, t *testing.T, s *llmtest.Script, q agent.Queue, env tool.Env, during func(agent.Event)) ([]agent.Event, agent.End) {
	t.Helper()
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Queue: q})
	var evs []agent.Event
	var end agent.End
	for ev, err := range a.Run(ctx, env, nil, user("go")) {
		if err != nil {
			t.Fatal(err)
		}
		evs = append(evs, ev)
		if during != nil {
			during(ev)
		}
		if e, ok := ev.(agent.End); ok {
			end = e
		}
	}
	return evs, end
}

func readEnv(t *testing.T) tool.Env {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "x"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return tool.Env{Cwd: dir}
}

// A steer queued while a tool runs enters after its result, before the
// next model call, and is reported as Queued first.
func TestSteeringAfterTools(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("r1", "read", `{"path":"x"}`), llmtest.Text("after"))
	q := &fakeQueue{}
	evs, end := runQueued(t.Context(), t, s, q, readEnv(t), func(ev agent.Event) {
		if _, ok := ev.(agent.ToolRun); ok {
			q.steering = append(q.steering, user("s1"))
		}
	})
	reqs := s.Requests()
	llmtest.AssertTail(t, reqs[1], llm.RoleUser, llm.RoleAssistant, llm.RoleTool, llm.RoleUser)
	if role, text := lastText(reqs[1]); role != llm.RoleUser || text != "s1" {
		t.Fatalf("second request ends with %s %q", role, text)
	}
	var order []string
	for _, ev := range evs {
		switch ev := ev.(type) {
		case agent.ToolEnd:
			order = append(order, "tool end")
		case agent.Queued:
			order = append(order, "queued "+ev.Message.Content[0].Text)
		case agent.Message:
			order = append(order, "message")
		}
	}
	if want := []string{"message", "tool end", "queued s1", "message"}; !slices.Equal(order, want) {
		t.Fatalf("events %v, want %v", order, want)
	}
	if end.StopReason != agent.StopEndTurn || len(end.Messages) != 5 || end.Messages[3].Content[0].Text != "s1" {
		t.Fatalf("end %+v", end)
	}
}

// A steer that arrives during a reply with no tool calls keeps the turn
// going.
func TestSteeringWithoutTools(t *testing.T) {
	s := llmtest.NewScript(llmtest.Text("one"), llmtest.Text("two"))
	q := &fakeQueue{}
	_, end := runQueued(t.Context(), t, s, q, tool.Env{}, func(ev agent.Event) {
		if d, ok := ev.(agent.TextDelta); ok && d.Text == "one" {
			q.steering = append(q.steering, user("more"))
		}
	})
	reqs := s.Requests()
	if len(reqs) != 2 {
		t.Fatalf("%d requests, want 2", len(reqs))
	}
	llmtest.AssertTail(t, reqs[1], llm.RoleUser, llm.RoleAssistant, llm.RoleUser)
	if end.StopReason != agent.StopEndTurn || len(end.Messages) != 4 {
		t.Fatalf("end %+v", end)
	}
}

// A follow-up waits until the turn would end: not after a tool result, but
// after the reply that asks for nothing.
func TestFollowUpWhenTurnEnds(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("r1", "read", `{"path":"x"}`), llmtest.Text("a"), llmtest.Text("b"))
	q := &fakeQueue{followUps: []llm.Message{user("f")}}
	_, end := runQueued(t.Context(), t, s, q, readEnv(t), nil)
	reqs := s.Requests()
	if len(reqs) != 3 {
		t.Fatalf("%d requests, want 3", len(reqs))
	}
	if role, _ := lastText(reqs[1]); role != llm.RoleTool {
		t.Fatalf("the follow-up entered after the tool result: second request ends with %s", role)
	}
	if role, text := lastText(reqs[2]); role != llm.RoleUser || text != "f" {
		t.Fatalf("third request ends with %s %q", role, text)
	}
	if end.StopReason != agent.StopEndTurn || len(end.Messages) != 6 {
		t.Fatalf("end %+v", end)
	}
}

// A steer queued before the turn enters after the prompt, before the first
// model call.
func TestSteeringAtStart(t *testing.T) {
	s := llmtest.NewScript(llmtest.Text("ok"))
	q := &fakeQueue{steering: []llm.Message{user("early")}}
	runQueued(t.Context(), t, s, q, tool.Env{}, nil)
	req := s.Requests()[0]
	llmtest.AssertTail(t, req, llm.RoleUser, llm.RoleUser)
	if _, text := lastText(req); text != "early" {
		t.Fatalf("first request ends with %q", text)
	}
}

// Nothing is polled once the turn is cancelled.
func TestNoPollAfterCancel(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("b1", "bash", `{"command":"sleep 30"}`))
	q := &fakeQueue{}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	_, end := runQueued(ctx, t, s, q, tool.Env{Cwd: t.TempDir()}, func(ev agent.Event) {
		if _, ok := ev.(agent.ToolRun); ok {
			q.steering = append(q.steering, user("late"))
			q.followUps = append(q.followUps, user("later"))
			time.AfterFunc(100*time.Millisecond, cancel)
		}
	})
	if end.StopReason != agent.StopCancelled || q.polls != 1 || len(q.steering) != 1 || len(q.followUps) != 1 {
		t.Fatalf("end %s, polls %d, left %d steering and %d follow-ups", end.StopReason, q.polls, len(q.steering), len(q.followUps))
	}
}

// stub is a read-only tool named name with the given exposure and family,
// returning a text result.
func stub(name string, exposure tool.Exposure, family string) tool.Tool {
	return tool.New(name, name+" does a thing.", func(context.Context, struct{}, tool.Env) (string, error) { return name + " ran", nil },
		tool.WithExposure(exposure), tool.WithFamily(family), tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true}))
}

// searchStub is a tool_search that loads load, and records the deferred
// specs each call is given.
func searchStub(load []string, seen *[][]string) tool.Tool {
	return tool.New("tool_search", "Find deferred tools.", func(_ context.Context, _ struct{}, env tool.Env) (tool.Result, error) {
		var names []string
		for _, s := range env.Deferred {
			names = append(names, s.Name)
		}
		*seen = append(*seen, names)
		r := tool.TextResult("loaded")
		r.Load = load
		return r, nil
	}, tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true}))
}

func exposureTools() []tool.Tool {
	return []tool.Tool{
		stub("a", "", ""),
		stub("b", tool.ExposureDeferred, "f"),
		stub("c", tool.ExposureDeferred, "f"),
		stub("d", tool.ExposureDeferred, ""),
		stub("h", tool.ExposureHidden, ""),
	}
}

// Deferred and hidden tools are not sent, and tool_search is offered only
// while something is deferred (0012-MADR D7, D9).
func TestExposureTiers(t *testing.T) {
	var seen [][]string
	s := llmtest.NewScript(llmtest.Text("ok"))
	a := agent.New(agent.Config{Provider: s, Tools: exposureTools(), Search: searchStub(nil, &seen)})
	if _, _, err := collect(t.Context(), t, a, tool.Env{}, "hi"); err != nil {
		t.Fatal(err)
	}
	llmtest.AssertTools(t, s.Requests()[0], "a", "tool_search")

	none := llmtest.NewScript(llmtest.Text("ok"))
	b := agent.New(agent.Config{Provider: none, Tools: []tool.Tool{stub("a", "", ""), stub("h", tool.ExposureHidden, "")}, Search: searchStub(nil, &seen)})
	if _, _, err := collect(t.Context(), t, b, tool.Env{}, "hi"); err != nil {
		t.Fatal(err)
	}
	llmtest.AssertTools(t, none.Requests()[0], "a")
}

// A load adds the tool and its whole family to the next request of the
// same turn, after the tools already sent; tool_search was given the
// deferred tools, hidden ones not among them; a Loaded event follows the
// call's ToolEnd; a later load appends again (0005-PLAN F1d-2).
func TestLoadInTurn(t *testing.T) {
	var seen [][]string
	s := llmtest.NewScript(
		llmtest.ToolCall("s1", "tool_search", `{}`),
		llmtest.ToolCall("b1", "b", `{}`),
		llmtest.Text("done"),
	)
	a := agent.New(agent.Config{Provider: s, Tools: exposureTools(), Search: searchStub([]string{"b", "nope"}, &seen)})
	evs, end, err := collect(t.Context(), t, a, tool.Env{}, "find it")
	if err != nil || end.StopReason != agent.StopEndTurn {
		t.Fatalf("turn: %v, %s", err, end.StopReason)
	}
	reqs := s.Requests()
	llmtest.AssertTools(t, reqs[0], "a", "tool_search")
	llmtest.AssertTools(t, reqs[1], "a", "tool_search", "b", "c")
	llmtest.AssertTools(t, reqs[2], "a", "tool_search", "b", "c")
	if len(seen) != 1 || !slices.Equal(seen[0], []string{"b", "c", "d"}) {
		t.Fatalf("tool_search saw %v; want the deferred b, c and d", seen)
	}
	var order []string
	for _, ev := range evs {
		switch ev := ev.(type) {
		case agent.ToolEnd:
			order = append(order, "end:"+ev.Call.Name)
		case agent.Loaded:
			order = append(order, "loaded:"+strings.Join(ev.Names, ","))
		}
	}
	if got := strings.Join(order, " "); got != "end:tool_search loaded:b,c end:b" {
		t.Fatalf("events %s", got)
	}
	if res := end.Messages[4].Content[0]; res.ToolName != "b" || res.IsError || res.Text != "b ran" {
		t.Fatalf("the loaded tool's call = %+v", res)
	}
}

// Tools loaded by an earlier prompt are sent as direct tools, after the
// others, in load order; a hidden tool cannot be called (0012-MADR D9).
func TestLoadedFromConfig(t *testing.T) {
	var seen [][]string
	s := llmtest.NewScript(llmtest.ToolCall("h1", "h", `{}`), llmtest.Text("ok"))
	a := agent.New(agent.Config{Provider: s, Tools: exposureTools(), Search: searchStub(nil, &seen), Loaded: []string{"d", "b"}})
	_, end, err := collect(t.Context(), t, a, tool.Env{}, "hi")
	if err != nil {
		t.Fatal(err)
	}
	llmtest.AssertTools(t, s.Requests()[0], "a", "d", "b", "tool_search")
	if res := end.Messages[2].Content[0]; !res.IsError || !strings.Contains(res.Text, `no tool named "h"`) {
		t.Fatalf("a call to the hidden tool = %+v", res)
	}
}

// Every model call of a turn carries the config's cache hint, the tool
// loop's later calls included; with none set, none asks (0021-MADR D2).
func TestCacheOnEveryCall(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, cache := range []llm.CacheRetention{llm.CacheLong, llm.CacheNone} {
		s := llmtest.NewScript(llmtest.ToolCall("c1", "read", args(t, map[string]string{"path": "notes.md"})), llmtest.Text("done"))
		a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Policy: readOnly{t: t}, Cache: cache})
		if _, _, err := collect(t.Context(), t, a, tool.Env{Cwd: dir}, "read it"); err != nil {
			t.Fatal(err)
		}
		reqs := s.Requests()
		if len(reqs) != 2 || reqs[0].Cache != cache || reqs[1].Cache != cache {
			t.Fatalf("cache %q: %d requests, want 2 each asking %q", cache, len(reqs), cache)
		}
	}
}
