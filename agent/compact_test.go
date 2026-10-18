package agent_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// compactor is a Config.Compact that records each ask's overflow flag, and
// answers the nth ask with answers[n], or nothing once they run out.
type compactor struct {
	asks    []bool
	answers []compactAnswer
}

type compactAnswer struct {
	compacted *agent.Compacted
	err       error
	during    func() // run before answering
}

func (c *compactor) compact(_ context.Context, overflow bool) (*agent.Compacted, error) {
	n := len(c.asks)
	c.asks = append(c.asks, overflow)
	if n >= len(c.answers) {
		return nil, nil
	}
	a := c.answers[n]
	if a.during != nil {
		a.during()
	}
	return a.compacted, a.err
}

// summary is a compaction's result: its history, and the system prompt it
// rebuilt.
var summary = &agent.Compacted{History: []llm.Message{user("SUMMARY")}, System: "rebuilt"}

// The hook is asked before each model call; when it compacts, the turn
// goes on from the history and the system prompt it returns, and End's
// Messages are what came after it (0005-PLAN F5-2 decisions 1 and 3,
// deviation 1).
func TestCompactBeforeEachCall(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("1", "read", `{"path":"x"}`), llmtest.Text("done"))
	c := &compactor{answers: []compactAnswer{{}, {compacted: summary}}}
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), System: "first", Compact: c.compact})
	_, end, err := collect(t.Context(), t, a, tool.Env{Cwd: t.TempDir()}, "go")
	if err != nil || end.StopReason != agent.StopEndTurn {
		t.Fatalf("end %+v, %v", end, err)
	}
	if !slices.Equal(c.asks, []bool{false, false}) {
		t.Fatalf("asks %v, want one before each of the two calls", c.asks)
	}
	if got := texts(s.Requests()[0].Messages); !slices.Equal(got, []string{"system:first", "user:go"}) {
		t.Fatalf("the first call sent %v", got)
	}
	if got := texts(s.Requests()[1].Messages); !slices.Equal(got, []string{"system:rebuilt", "user:SUMMARY"}) {
		t.Fatalf("the second call sent %v, want the rebuilt system prompt and the compacted history alone", got)
	}
	if got := texts(end.Messages); !slices.Equal(got, []string{"assistant:done"}) {
		t.Fatalf("End's messages %v, want only the reply after the compaction", got)
	}
}

// A call over the window is compacted and sent once more; a second
// overflow of the same call, or one the hook does not compact, ends the
// turn with the overflow (0005-PLAN F5-2 decision 2).
func TestOverflowCompactsOnce(t *testing.T) {
	overflow := llmtest.Turn{Err: llm.ErrContextOverflow}
	for _, c := range []struct {
		name    string
		turns   []llmtest.Turn
		answers []compactAnswer
		asks    []bool
		fails   bool
		sent    int
	}{
		{"compacted and sent again", []llmtest.Turn{overflow, llmtest.Text("ok")},
			[]compactAnswer{{}, {compacted: summary}}, []bool{false, true}, false, 2},
		{"a second overflow", []llmtest.Turn{overflow, overflow, llmtest.Text("never")},
			[]compactAnswer{{}, {compacted: summary}, {compacted: summary}}, []bool{false, true}, true, 2},
		{"not compacted", []llmtest.Turn{overflow, llmtest.Text("never")}, nil, []bool{false, true}, true, 1},
	} {
		s := llmtest.NewScript(c.turns...)
		cp := &compactor{answers: c.answers}
		a := agent.New(agent.Config{Provider: s, Compact: cp.compact})
		_, end, err := collect(t.Context(), t, a, tool.Env{}, "go")
		if c.fails != errors.Is(err, llm.ErrContextOverflow) || (!c.fails && err != nil) {
			t.Errorf("%s: err %v, want the overflow %v", c.name, err, c.fails)
		}
		if !slices.Equal(cp.asks, c.asks) || len(s.Requests()) != c.sent {
			t.Errorf("%s: asks %v, %d calls; want %v, %d", c.name, cp.asks, len(s.Requests()), c.asks, c.sent)
		}
		if !c.fails && !slices.Equal(texts(s.Requests()[1].Messages), []string{"system:rebuilt", "user:SUMMARY"}) {
			t.Errorf("%s: the resend sent %v, want the compacted history", c.name, texts(s.Requests()[1].Messages))
		}
		if !c.fails && !slices.Equal(texts(end.Messages), []string{"assistant:ok"}) {
			t.Errorf("%s: End's messages %v", c.name, texts(end.Messages))
		}
	}
}

// The hook's error ends the turn, before the call or after an overflow;
// a compaction the turn's cancel cut short ends it as cancelled.
func TestCompactError(t *testing.T) {
	boom := errors.New("boom")
	for _, c := range []struct {
		name    string
		turns   []llmtest.Turn
		answers []compactAnswer
		sent    int
	}{
		{"before the call", []llmtest.Turn{llmtest.Text("never")}, []compactAnswer{{err: boom}}, 0},
		{"after an overflow", []llmtest.Turn{{Err: llm.ErrContextOverflow}, llmtest.Text("never")}, []compactAnswer{{}, {err: boom}}, 1},
	} {
		s := llmtest.NewScript(c.turns...)
		cp := &compactor{answers: c.answers}
		a := agent.New(agent.Config{Provider: s, Compact: cp.compact})
		if _, _, err := collect(t.Context(), t, a, tool.Env{}, "go"); !errors.Is(err, boom) || len(s.Requests()) != c.sent {
			t.Errorf("%s: err %v after %d calls, want boom after %d", c.name, err, len(s.Requests()), c.sent)
		}
	}
	for _, overflow := range []bool{false, true} {
		ctx, cancel := context.WithCancel(t.Context())
		cut := compactAnswer{during: cancel, err: context.Canceled}
		answers := []compactAnswer{cut}
		turns := []llmtest.Turn{llmtest.Text("never")}
		if overflow {
			answers = []compactAnswer{{}, cut}
			turns = []llmtest.Turn{{Err: llm.ErrContextOverflow}, llmtest.Text("never")}
		}
		cp := &compactor{answers: answers}
		a := agent.New(agent.Config{Provider: llmtest.NewScript(turns...), Compact: cp.compact})
		_, end, err := collect(ctx, t, a, tool.Env{}, "go")
		if err != nil || end.StopReason != agent.StopCancelled {
			t.Errorf("cancelled in the compaction (overflow %v): end %+v, %v; want cancelled", overflow, end, err)
		}
		cancel()
	}
}
