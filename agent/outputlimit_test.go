package agent_test

import (
	"fmt"
	"log/slog"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// limiter is a Config.OutputLimit that answers each request by its
// messages' count, so two requests of one turn get two values, and a
// request that already names a limit, a refused one's stated room or cap,
// with it, basis refusal. It records what each request named when asked.
type limiter struct {
	asked []int
}

func (l *limiter) limit(req *llm.Request) llm.OutputLimit {
	l.asked = append(l.asked, req.MaxTokens)
	if req.MaxTokens > 0 {
		return llm.OutputLimit{Tokens: req.MaxTokens, Basis: llm.BasisRefusal}
	}
	return llm.OutputLimit{Tokens: 60000 + len(req.Messages), Basis: llm.BasisCeiling}
}

func sentCaps(reqs []*llm.Request) []int {
	out := make([]int, len(reqs))
	for i, r := range reqs {
		out[i] = r.MaxTokens
	}
	return out
}

// Each model call of a turn asks for its own limit, with its own request
// (0071-MADR D2, D4, L10).
func TestCapPerCall_0071(t *testing.T) {
	s := llmtest.NewScript(llmtest.ToolCall("1", "read", `{"path":"x"}`), llmtest.Text("done"))
	l := &limiter{}
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), OutputLimit: l.limit})
	if _, _, err := collect(t.Context(), t, a, tool.Env{Cwd: t.TempDir()}, "go"); err != nil {
		t.Fatal(err)
	}
	reqs := s.Requests()
	if len(reqs) != 2 {
		t.Fatalf("%d calls, want 2", len(reqs))
	}
	for i, r := range reqs {
		if r.MaxTokens != 60000+len(r.Messages) {
			t.Errorf("call %d sent %d, want its own %d", i+1, r.MaxTokens, 60000+len(r.Messages))
		}
	}
	if reqs[0].MaxTokens == reqs[1].MaxTokens {
		t.Errorf("caps %v: each call asks for its own", sentCaps(reqs))
	}
	// Nil sends no limit.
	plain := llmtest.NewScript(llmtest.Text("done"))
	if _, _, err := collect(t.Context(), t, agent.New(agent.Config{Provider: plain}), tool.Env{}, "go"); err != nil || plain.Requests()[0].MaxTokens != 0 {
		t.Errorf("no OutputLimit: sent %d, %v; want none", plain.Requests()[0].MaxTokens, err)
	}
}

// The call sent again after an overflow's compaction asks for its limit
// again, over the compacted request (0071-MADR D8 (b), L11).
func TestOverflowResendRecomputesCap_0071(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Err: llm.ErrContextOverflow}, llmtest.Text("ok"))
	cp := &compactor{answers: []compactAnswer{{}, {compacted: summary}}}
	l := &limiter{}
	a := agent.New(agent.Config{Provider: s, Compact: cp.compact, OutputLimit: l.limit})
	if _, _, err := collect(t.Context(), t, a, tool.Env{}, "a long prompt"); err != nil {
		t.Fatal(err)
	}
	reqs := s.Requests()
	if len(reqs) != 2 || !slices.Equal(cp.asks, []bool{false, true}) {
		t.Fatalf("%d calls, asks %v; want the overflow compacted and the call sent again", len(reqs), cp.asks)
	}
	if reqs[1].MaxTokens != 60000+len(reqs[1].Messages) || slices.Equal(texts(reqs[0].Messages), texts(reqs[1].Messages)) {
		t.Errorf("the resend sent %d over %v, want its own limit", reqs[1].MaxTokens, texts(reqs[1].Messages))
	}
	if !slices.Equal(l.asked, []int{0, 0}) {
		t.Errorf("asked %v, want two fresh asks", l.asked)
	}
}

// refused is a 400 whose provider message is msg.
func refused(msg string, wrap error) llmtest.Turn {
	err := fmt.Errorf("invalid request: %s", msg)
	if wrap != nil {
		err = fmt.Errorf("%w: %s", wrap, msg)
	}
	return llmtest.Turn{Err: &llm.APIError{Status: 400, Message: msg, Err: err}}
}

// A call refused for its limit, in a form that states the room or the cap,
// is sent once more at it, basis refusal, before any compaction; refused
// again, the turn ends with the refusal; a room at or below 0 is an
// overflow of the input, which compacts (0071-MADR D8 (a), (b), L39).
func TestCapRefusalResentOnce_0071(t *testing.T) {
	anthropic := func(input, sent int) llmtest.Turn {
		return refused(fmt.Sprintf("input length and `max_tokens` exceed context limit: %d + %d > 200000", input, sent), nil)
	}
	// The first request carries the system prompt and the prompt: the
	// limiter asks for 60,002.
	openRouter := refused("This endpoint's maximum context length is 131072 tokens. However, you requested about 160002 tokens (100000 of text input, 60002 in the output).",
		llm.ErrContextOverflow)
	capForm := refused("max_tokens: 60002 > 32000, which is the maximum allowed number of output tokens for m1", nil)
	for _, c := range []struct {
		name  string
		turns []llmtest.Turn
		caps  []int  // each call's limit
		asks  []bool // the compaction's asks
		fails bool
	}{
		{"anthropic_window", []llmtest.Turn{anthropic(184000, 60002), llmtest.Text("ok")}, []int{60002, 16000}, []bool{false}, false},
		{"openrouter_window", []llmtest.Turn{openRouter, llmtest.Text("ok")}, []int{60002, 31072}, []bool{false}, false},
		{"cap_form", []llmtest.Turn{capForm, llmtest.Text("ok")}, []int{60002, 32000}, []bool{false}, false},
		{"refused_again", []llmtest.Turn{anthropic(184000, 60002), anthropic(190000, 16000), llmtest.Text("never")}, []int{60002, 16000},
			[]bool{false}, true},
		{"input_overflow", []llmtest.Turn{anthropic(210000, 60002), llmtest.Text("ok")}, []int{60002, 60002}, []bool{false, true}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := llmtest.NewScript(c.turns...)
			cp := &compactor{answers: []compactAnswer{{}, {compacted: &agent.Compacted{History: []llm.Message{user("SUMMARY")}, System: "rebuilt"}}}}
			l := &limiter{}
			a := agent.New(agent.Config{Provider: s, System: "sys", Compact: cp.compact, OutputLimit: l.limit})
			_, _, err := collect(t.Context(), t, a, tool.Env{}, "go")
			if (err != nil) != c.fails {
				t.Fatalf("err %v, want failed %v", err, c.fails)
			}
			if got := sentCaps(s.Requests()); !slices.Equal(got, c.caps) {
				t.Errorf("caps %v, want %v", got, c.caps)
			}
			if !slices.Equal(cp.asks, c.asks) {
				t.Errorf("compaction asks %v, want %v", cp.asks, c.asks)
			}
		})
	}
}

// Once a refusal has stated a room or cap M in a turn, every later call of
// the turn asks for at most M: its cap is computed again, then held to M,
// after an overflow's compaction, an offline wait, an empty reply's run
// again and a tool round alike; a computed cap below M is sent as computed
// (0071-MADR D8 (a) as amended, L44, the owner's "Keep the stated value
// (Recommended)"). The waits run in a synctest bubble, on its clock.
func TestStatedCapHeldForTurn_0071(t *testing.T) {
	refusal := refused("input length and `max_tokens` exceed context limit: 184000 + 60002 > 200000", nil) // a room of 16,000
	overflow := llmtest.Turn{Err: llm.ErrContextOverflow}
	compacted := func() *compactor {
		return &compactor{answers: []compactAnswer{{}, {compacted: &agent.Compacted{History: []llm.Message{user("SUMMARY")}, System: "rebuilt"}}}}
	}
	for _, c := range []struct {
		name     string
		turns    []llmtest.Turn
		computed []int // the fresh asks' answers, in order; the last repeats
		caps     []int
		probes   bool // the provider probes, for the offline wait
	}{
		{"overflow_resend", []llmtest.Turn{refusal, overflow, llmtest.Text("ok")}, []int{60002}, []int{60002, 16000, 16000}, false},
		{"offline_wait", []llmtest.Turn{refusal, {Err: errOffline}, llmtest.Text("ok")}, []int{60002}, []int{60002, 16000, 16000}, true},
		{"empty_reply", []llmtest.Turn{refusal, llmtest.Text(""), llmtest.Text("ok")}, []int{60002}, []int{60002, 16000, 16000}, false},
		{"next_call", []llmtest.Turn{refusal, llmtest.ToolCall("1", "read", `{"path":"x"}`), llmtest.Text("ok")}, []int{60002},
			[]int{60002, 16000, 16000}, false},
		{"computed_below", []llmtest.Turn{refusal, overflow, llmtest.Text("ok")}, []int{60002, 9000}, []int{60002, 16000, 9000}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				fresh := 0
				limit := func(req *llm.Request) llm.OutputLimit {
					if req.MaxTokens > 0 {
						return llm.OutputLimit{Tokens: req.MaxTokens, Basis: llm.BasisRefusal}
					}
					v := c.computed[min(fresh, len(c.computed)-1)]
					fresh++
					return llm.OutputLimit{Tokens: v, Basis: llm.BasisCeiling}
				}
				s := llmtest.NewScript(c.turns...)
				var p llm.Provider = s
				if c.probes {
					p = &probing{Script: s, start: time.Now(), answers: []error{nil}}
				}
				a := agent.New(agent.Config{Provider: p, System: "sys", Tools: builtin.Tools(), Compact: compacted().compact, OutputLimit: limit,
					Retry: &llm.Retry{OfflineWait: time.Minute}})
				if _, _, err := collect(t.Context(), t, a, tool.Env{Cwd: t.TempDir()}, "go"); err != nil {
					t.Fatalf("the turn failed: %v", err)
				}
				if got := sentCaps(s.Requests()); !slices.Equal(got, c.caps) {
					t.Errorf("caps %v, want %v", got, c.caps)
				}
			})
		})
	}
}

// resultIDs are the ids of the tool results msgs carry.
func resultIDs(msgs []llm.Message) []string {
	var out []string
	for _, m := range msgs {
		for _, c := range m.Content {
			if c.Type == llm.ContentToolResult {
				out = append(out, c.ToolCallID)
			}
		}
	}
	return out
}

// A reply the output limit cut after a whole tool call, as the SDK reports
// it from v1.33.0: the whole call runs, the turn makes its next call, the
// cut call has no result, and the SDK's tool_call_cut warning is logged once,
// at Warn (0071-MADR P9; 0065-MADR R24).
func TestAgentTurn_SalvageCompleteCalls_0065(t *testing.T) {
	cut := llmtest.Turn{Events: []llm.Event{
		llm.ToolCallDone{ID: "1", Name: "read", Args: `{"path":"x"}`},
		llm.Warning{Kind: llm.WarningToolCallCut, Model: "m", Detail: "the output limit cut call 2 (read)"},
		llm.Usage{InputTokens: 3, OutputTokens: 2},
		llm.Done{StopReason: llm.StopToolUse},
	}}
	s := llmtest.NewScript(cut, llmtest.Text("done"))
	h := &records{level: slog.LevelDebug}
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Logger: slog.New(h)})
	_, end, err := collect(t.Context(), t, a, tool.Env{Cwd: t.TempDir()}, "go")
	if err != nil {
		t.Fatal(err)
	}
	reqs := s.Requests()
	if len(reqs) != 2 || end.StopReason != agent.StopEndTurn {
		t.Fatalf("%d calls, stop %q; want the whole call run and a second call", len(reqs), end.StopReason)
	}
	if got := resultIDs(reqs[1].Messages); !slices.Equal(got, []string{"1"}) {
		t.Errorf("the next call carries results %q, want the whole call's alone", got)
	}
	if got := h.count(slog.LevelWarn); got != 1 {
		t.Errorf("%d Warn records, want the warning once", got)
	}
}

// Kilo's window form, an overflow from go-llmprovider-sdk v1.33.0: a stated
// room of at least 1 is one resend at it, with no summary; the input alone
// past the window is a summary, then the turn sent again (0071-MADR P12,
// L39, L11; 0065-MADR R20).
func TestAgentTurn_KiloWindowFormCompacts_0065(t *testing.T) {
	kilo := func(total int) llmtest.Turn {
		return refused(fmt.Sprintf("This request exceeds the model's context window of 131072 tokens. "+
			"Your request contains approximately %d tokens.", total), llm.ErrContextOverflow)
	}
	for _, c := range []struct {
		name  string
		first llmtest.Turn
		caps  []int  // each call's limit
		asks  []bool // the compaction's asks
	}{
		// 100,000 of input and the 60,002 asked: a room of 31,072.
		{"room", kilo(160002), []int{60002, 31072}, []bool{false}},
		// 200,000 of input alone passes the window.
		{"over", kilo(260002), []int{60002, 60002}, []bool{false, true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := llmtest.NewScript(c.first, llmtest.Text("ok"))
			cp := &compactor{answers: []compactAnswer{{}, {compacted: &agent.Compacted{History: []llm.Message{user("SUMMARY")}, System: "rebuilt"}}}}
			l := &limiter{}
			a := agent.New(agent.Config{Provider: s, System: "sys", Compact: cp.compact, OutputLimit: l.limit})
			if _, _, err := collect(t.Context(), t, a, tool.Env{}, "go"); err != nil {
				t.Fatalf("the turn failed: %v", err)
			}
			if got := sentCaps(s.Requests()); !slices.Equal(got, c.caps) {
				t.Errorf("caps %v, want %v", got, c.caps)
			}
			if !slices.Equal(cp.asks, c.asks) {
				t.Errorf("compaction asks %v, want %v", cp.asks, c.asks)
			}
		})
	}
}

// Text, then a call the output limit cut, which the SDK dropped: the turn
// ends max_tokens with its text kept, runs nothing, and logs the warning
// once, at Warn (0071-MADR P29, the owner's "End the turn, keep the text,
// notify (Recommended)").
func TestCutCallAfterText_0071(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{
		llm.TextDelta{Text: "Let me read it."},
		llm.Warning{Kind: llm.WarningToolCallCut, Model: "m", Detail: "the output limit cut call 1 (read)"},
		llm.Usage{InputTokens: 3, OutputTokens: 2},
		llm.Done{StopReason: llm.StopMaxTokens},
	}})
	h := &records{level: slog.LevelDebug}
	a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Logger: slog.New(h)})
	_, end, err := collect(t.Context(), t, a, tool.Env{Cwd: t.TempDir()}, "go")
	if err != nil {
		t.Fatal(err)
	}
	if n := len(s.Requests()); n != 1 || end.StopReason != agent.StopMaxTokens {
		t.Fatalf("%d calls, stop %q; want one call, max_tokens", n, end.StopReason)
	}
	if got := texts(end.Messages); !slices.Contains(got, "assistant:Let me read it.") {
		t.Errorf("the turn's messages %q lack the reply's text", got)
	}
	if got := h.count(slog.LevelWarn); got != 1 {
		t.Errorf("%d Warn records, want the warning once", got)
	}
}
