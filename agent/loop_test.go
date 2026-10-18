package agent_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/tool"
)

type pingIn struct {
	N int `json:"n"`
	M int `json:"m,omitzero"`
}

// ping is a read-only tool that records each n it runs with.
func ping(name string, ran *[]int) tool.Tool {
	return tool.New(name, "ping", func(_ context.Context, in pingIn, _ tool.Env) (string, error) {
		*ran = append(*ran, in.N)
		return "pong", nil
	}, tool.WithAnnotations(tool.Annotations{ReadOnlyHint: true}))
}

// asker is a policy that records the repeated-call questions it is asked,
// and answers them allow; every other call is allowed without a record.
type asker struct {
	allow bool
	asked []permission.Rule
}

func (a *asker) Decide(_ context.Context, r permission.Rule) (permission.Decision, error) {
	if r.Repeated == 0 {
		return permission.Decision{Allow: true}, nil
	}
	a.asked = append(a.asked, r)
	return permission.Decision{Allow: a.allow}, nil
}

// calls is a reply of tool calls, each id with its arguments.
func calls(name string, idArgs ...string) llmtest.Turn {
	var evs []llm.Event
	for i := 0; i+1 < len(idArgs); i += 2 {
		evs = append(evs, llm.ToolCallDone{ID: idArgs[i], Name: name, Args: idArgs[i+1]})
	}
	return llmtest.Turn{Events: append(evs, llm.Done{StopReason: llm.StopToolUse})}
}

// toolResults are End's tool results, as "<id>:<text>".
func toolResults(msgs []llm.Message) []string {
	var out []string
	for _, m := range msgs {
		for _, c := range m.Content {
			if c.Type == llm.ContentToolResult {
				out = append(out, c.ToolCallID+":"+c.Text)
			}
		}
	}
	return out
}

// A third identical call in a row, across the turn's replies, asks the
// policy before it runs; allowed, it runs, and the question comes again
// after three more (0005-PLAN F5-3 decision 1).
func TestRepeatedCallAsks(t *testing.T) {
	var ran []int
	p := &asker{allow: true}
	s := llmtest.NewScript(
		calls("ping", "a", `{"n":1}`, "b", `{"n":1}`),
		calls("ping", "c", `{"n":1}`, "d", `{"n":1}`, "e", `{"n":1}`, "f", `{"n":1}`),
		llmtest.Text("done"))
	a := agent.New(agent.Config{Provider: s, Tools: []tool.Tool{ping("ping", &ran)}, Policy: p})
	_, end, err := collect(t.Context(), t, a, tool.Env{}, "go")
	if err != nil || end.StopReason != agent.StopEndTurn || len(ran) != 6 {
		t.Fatalf("end %+v, %v, ran %v; want all six run", end.StopReason, err, ran)
	}
	var ids []string
	for _, r := range p.asked {
		ids = append(ids, r.CallID)
		if r.Repeated != 3 || r.Tool != "ping" || string(r.Input) != `{"n":1}` {
			t.Errorf("question %+v, want ping's call, repeated 3", r)
		}
	}
	if !slices.Equal(ids, []string{"c", "f"}) {
		t.Fatalf("asked about %v, want the third and the sixth", ids)
	}
}

// Refused, the call does not run: its result says so, the reply's later
// calls are cancelled, and the turn ends with no further model call.
func TestRepeatedCallStopped(t *testing.T) {
	var ran []int
	s := llmtest.NewScript(calls("ping", "a", `{"n":1}`, "b", `{"n":1}`, "c", `{"n":1}`, "d", `{"n":2}`), llmtest.Text("never"))
	a := agent.New(agent.Config{Provider: s, Tools: []tool.Tool{ping("ping", &ran)}, Policy: &asker{}})
	_, end, err := collect(t.Context(), t, a, tool.Env{}, "go")
	if err != nil || end.StopReason != agent.StopEndTurn || s.Remaining() != 1 || len(ran) != 2 {
		t.Fatalf("end %+v, %v, %d turns left, ran %v; want end_turn after two ran", end.StopReason, err, s.Remaining(), ran)
	}
	got := toolResults(end.Messages)
	want := []string{"a:pong", "b:pong", "c:The user stopped this call: the model made the same call 3 times in a row.",
		"d:The user cancelled the turn, so this call did not finish."}
	if !slices.Equal(got, want) {
		t.Fatalf("results %q, want %q", got, want)
	}
}

// Calls are identical by name and arguments as JSON, key order aside; a
// different call starts a new run; with no policy the turn goes on.
func TestRepeatedCallCounting(t *testing.T) {
	var ran []int
	p := &asker{allow: true}
	s := llmtest.NewScript(calls("ping",
		"a", `{"n":1}`, "b", `{"n":1}`, "c", `{"n":2,"m":1}`, "d", `{"m":1,"n":2}`, "e", `{ "n" : 2, "m" : 1 }`),
		llmtest.Text("done"))
	a := agent.New(agent.Config{Provider: s, Tools: []tool.Tool{ping("ping", &ran)}, Policy: p})
	if _, _, err := collect(t.Context(), t, a, tool.Env{}, "go"); err != nil {
		t.Fatal(err)
	}
	if len(p.asked) != 1 || p.asked[0].CallID != "e" {
		t.Fatalf("asked %+v, want only e, the third n=2, m=1 in a row", p.asked)
	}

	ran = nil
	s = llmtest.NewScript(calls("ping", "a", `{"n":1}`, "b", `{"n":1}`, "c", `{"n":1}`), llmtest.Text("done"))
	a = agent.New(agent.Config{Provider: s, Tools: []tool.Tool{ping("ping", &ran)}})
	if _, end, err := collect(t.Context(), t, a, tool.Env{}, "go"); err != nil || len(ran) != 3 || end.StopReason != agent.StopEndTurn {
		t.Fatalf("no policy: ran %v, %v; want all three", ran, err)
	}
}

// A call whose name is a tool's but for case and space runs that tool,
// and the reply carries the tool's name; two tools that match, or a
// deferred tool, are not chosen (0005-PLAN F5-3 decision 2).
func TestCallNameRepair(t *testing.T) {
	var ran []int
	s := llmtest.NewScript(calls(" PING ", "a", `{"n":1}`), llmtest.Text("done"))
	a := agent.New(agent.Config{Provider: s, Tools: []tool.Tool{ping("ping", &ran)}})
	evs, end, err := collect(t.Context(), t, a, tool.Env{}, "go")
	if err != nil || len(ran) != 1 {
		t.Fatalf("ran %v, %v; want ping run", ran, err)
	}
	for _, ev := range evs {
		if m, ok := ev.(agent.Message); ok && len(m.Message.Content) > 0 && m.Message.Content[0].Type == llm.ContentToolCall &&
			m.Message.Content[0].ToolName != "ping" {
			t.Fatalf("the recorded call names %q, want ping", m.Message.Content[0].ToolName)
		}
	}
	if call := s.Requests()[1].Messages[1].Content[0]; call.ToolName != "ping" {
		t.Fatalf("the next request sent the call as %q, want ping", call.ToolName)
	}
	if got := toolResults(end.Messages); !slices.Equal(got, []string{"a:pong"}) {
		t.Fatalf("results %q", got)
	}

	for _, c := range []struct {
		name  string
		tools []tool.Tool
		call  string
	}{
		{"two tools match", []tool.Tool{ping("Ping", &ran), ping("pinG", &ran)}, "PING"},
		{"a deferred tool", []tool.Tool{tool.New("lazy", "", func(context.Context, pingIn, tool.Env) (string, error) { return "", nil },
			tool.WithExposure(tool.ExposureDeferred))}, "LAZY"},
	} {
		s := llmtest.NewScript(calls(c.call, "a", `{"n":1}`), llmtest.Text("done"))
		a := agent.New(agent.Config{Provider: s, Tools: c.tools})
		_, end, err := collect(t.Context(), t, a, tool.Env{}, "go")
		if got := toolResults(end.Messages); err != nil || len(got) != 1 || !strings.Contains(got[0], "there is no tool named") {
			t.Errorf("%s: results %q, %v; want the unknown tool's", c.name, got, err)
		}
	}
}

// Three results in a row whose arguments were refused, across replies,
// end the turn once the reply's calls have run; any other result between
// them clears the count (0005-PLAN F5-3 decision 3).
func TestMalformedCallBreaker(t *testing.T) {
	var ran []int
	s := llmtest.NewScript(
		calls("ping", "a", `{"n":"x"}`, "b", `{"n":"y"}`),
		calls("ping", "c", `{"n":"z"}`, "d", `{"n":"w"}`),
		llmtest.Text("never"))
	a := agent.New(agent.Config{Provider: s, Tools: []tool.Tool{ping("ping", &ran)}})
	_, _, err := collect(t.Context(), t, a, tool.Env{}, "go")
	if err == nil || err.Error() != "the model sent invalid tool arguments 4 times in a row, the last for ping" || s.Remaining() != 1 {
		t.Fatalf("err %v, %d turns left; want the breaker after the second reply's calls", err, s.Remaining())
	}

	s = llmtest.NewScript(
		calls("ping", "a", `{"n":"x"}`, "b", `{"n":"y"}`, "c", `{"n":1}`),
		calls("ping", "d", `{"n":"z"}`, "e", `{"n":"w"}`),
		llmtest.Text("done"))
	a = agent.New(agent.Config{Provider: s, Tools: []tool.Tool{ping("ping", &ran)}})
	if _, end, err := collect(t.Context(), t, a, tool.Env{}, "go"); err != nil || end.StopReason != agent.StopEndTurn {
		t.Fatalf("a good call between: end %+v, %v; want the turn to go on", end.StopReason, err)
	}
}

// A reply stopped as a refusal ends the turn as one; an empty one is not
// run again as an empty reply (0005-PLAN F5-3 decision 5).
func TestRefusalStop(t *testing.T) {
	refused := func(text string) llmtest.Turn {
		var evs []llm.Event
		if text != "" {
			evs = append(evs, llm.TextDelta{Text: text})
		}
		return llmtest.Turn{Events: append(evs, llm.Done{StopReason: llm.StopRefusal})}
	}
	retry := llm.DefaultRetry
	for _, text := range []string{"I can't help with that.", ""} {
		s := llmtest.NewScript(refused(text), llmtest.Text("never"))
		a := agent.New(agent.Config{Provider: s, Retry: &retry})
		_, end, err := collect(t.Context(), t, a, tool.Env{}, "go")
		if err != nil || end.StopReason != agent.StopRefusal || s.Remaining() != 1 {
			t.Errorf("refusal %q: end %+v, %v, %d turns left; want refusal after one call", text, end.StopReason, err, s.Remaining())
		}
	}
}
