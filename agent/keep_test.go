package agent_test

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// textOf is m's text blocks, joined.
func textOf(m llm.Message) string {
	var b strings.Builder
	for _, c := range m.Content {
		if c.Type == llm.ContentText {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

// thinkingOf is m's thinking blocks' text, joined.
func thinkingOf(m llm.Message) string {
	var b strings.Builder
	for _, c := range m.Content {
		if c.Type == llm.ContentThinking {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

// callsOf is m's tool calls, as id:name.
func callsOf(m llm.Message) []string {
	var out []string
	for _, c := range m.Content {
		if c.Type == llm.ContentToolCall {
			out = append(out, c.ToolCallID+":"+c.ToolName)
		}
	}
	return out
}

// failedOf are the failed Messages among evs, in order.
func failedOf(evs []agent.Event) []agent.Message {
	var out []agent.Message
	for _, ev := range evs {
		if m, ok := ev.(agent.Message); ok && m.Err != nil {
			out = append(out, m)
		}
	}
	return out
}

// starts counts the ToolStart events among evs.
func starts(evs []agent.Event) int {
	n := 0
	for _, ev := range evs {
		if _, ok := ev.(agent.ToolStart); ok {
			n++
		}
	}
	return n
}

// runAll runs a turn of a, collecting every event, the End, and the error.
func runAll(ctx context.Context, a *agent.Agent, env tool.Env) ([]agent.Event, agent.End, error) {
	var evs []agent.Event
	var end agent.End
	for ev, err := range a.Run(ctx, env, nil, user("go")) {
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

// errBad is a failure no wait or retry takes: an HTTP 400.
var errBad = &llm.APIError{Status: 400, Err: errors.New("scripted bad request")}

// hiddenReasoning is an encrypted reasoning item alone: an event that shows
// nothing.
var hiddenReasoning = llm.ThinkingDelta{ProviderData: jsontext.Value(`{"encrypted":"e1"}`)}

// The call a failed turn ends on is always emitted, as a Message carrying
// the error, before the error: with what it showed, its thinking and its
// complete calls, not run, or empty; the turn's own failures, the reruns run
// out or an aborted wait, and a Responses cut are kept the same way; a
// cancel in the empty reply's wait keeps the shown attempt as a cancelled
// one (0070-MADR D1 to D3, D11; G1 to G4, G8 to G10, G16, G33, G36, G37).
func TestFailedCallKept(t *testing.T) {
	incomplete := fmt.Errorf("%w: max_output_tokens", llm.ErrIncomplete)
	call := llm.ToolCallDone{ID: "c1", Name: "read", Args: `{"path":"x"}`}
	for _, c := range []struct {
		name     string
		turn     llmtest.Turn
		want     error
		text     string
		thinking string
		calls    []string
	}{
		{"text", llmtest.Turn{Events: []llm.Event{llm.ThinkingDelta{Text: "so"}, llm.TextDelta{Text: "half"}}, Err: errBad}, errBad, "half", "so", nil},
		{"nothing_shown", llmtest.Turn{Err: errBad}, errBad, "", "", nil},
		{"calls_only", llmtest.Turn{Events: []llm.Event{call}, Err: errBad}, errBad, "", "", []string{"c1:read"}},
		{"text_and_calls", llmtest.Turn{Events: []llm.Event{llm.ThinkingDelta{Text: "so"}, llm.TextDelta{Text: "half"}, call}, Err: errBad},
			errBad, "half", "so", []string{"c1:read"}},
		{"responses_cut_text", llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "half"}}, Err: incomplete}, llm.ErrIncomplete, "half", "", nil},
		{"responses_cut_thinking", llmtest.Turn{Events: []llm.Event{llm.ThinkingDelta{Text: "so"}}, Err: incomplete}, llm.ErrIncomplete, "", "so", nil},
		{"responses_cut_calls", llmtest.Turn{Events: []llm.Event{call}, Err: incomplete}, llm.ErrIncomplete, "", "", []string{"c1:read"}},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := llmtest.NewScript(c.turn, llmtest.Text("never"))
			a := agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Retry: &llm.Retry{AfterCut: true, OfflineWait: time.Minute}})
			evs, _, err := runAll(t.Context(), a, tool.Env{Cwd: t.TempDir()})
			if !errors.Is(err, c.want) {
				t.Fatalf("err %v, want %v", err, c.want)
			}
			kept := failedOf(evs)
			if len(kept) != 1 || !errors.Is(kept[0].Err, c.want) {
				t.Fatalf("failed messages %+v, want one carrying %v", kept, c.want)
			}
			m := kept[0].Message
			if textOf(m) != c.text || thinkingOf(m) != c.thinking || !slices.Equal(callsOf(m), c.calls) {
				t.Errorf("kept text %q, thinking %q, calls %v; want %q, %q, %v", textOf(m), thinkingOf(m), callsOf(m), c.text, c.thinking, c.calls)
			}
			if n := starts(evs); n != 0 || len(s.Requests()) != 1 {
				t.Errorf("%d tool starts, %d requests; want no call run and one request", n, len(s.Requests()))
			}
		})
	}
	t.Run("thinking", keptThinking)
	t.Run("reruns_exhausted", keptRerunsExhausted)
	t.Run("wait_aborted", keptWaitAborted)
	t.Run("overflow_after_text", keptOverflowAfterText)
	t.Run("cancel_in_wait", keptCancelInWait)
}

func keptThinking(t *testing.T) {
	// Thinking is shown: the call is not run again for its empty answer.
	s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.ThinkingDelta{Text: "hmm"}}, Err: errEmptyAnswer}, llmtest.Text("never"))
	evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}}), tool.Env{})
	kept := failedOf(evs)
	if !errors.Is(err, llm.ErrEmptyReply) || len(s.Requests()) != 1 || len(kept) != 1 || thinkingOf(kept[0].Message) != "hmm" {
		t.Fatalf("err %v after %d requests, kept %+v; want the thinking kept and one request", err, len(s.Requests()), kept)
	}
}

func keptRerunsExhausted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(reasoningOnly(), reasoningOnly(), reasoningOnly())
		evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}}), tool.Env{})
		kept := failedOf(evs)
		if !errors.Is(err, llm.ErrEmptyReply) || len(kept) != 3 {
			t.Fatalf("err %v, %d kept; want the three attempts kept", err, len(kept))
		}
		const last = "the provider gave no usable reply 3 times: llm: empty reply"
		if kept[0].Err.Error() != "the provider gave an empty reply: llm: empty reply" || kept[2].Err.Error() != last {
			t.Errorf("kept errors %q, %q; want the empty reply's, then %q", kept[0].Err, kept[2].Err, last)
		}
	})
}

func keptWaitAborted(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		abort := make(chan struct{})
		time.AfterFunc(time.Second, func() { close(abort) })
		s := llmtest.NewScript(reasoningOnly(), llmtest.Text("never"))
		evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}, RetryAbort: abort}), tool.Env{})
		kept := failedOf(evs)
		if !errors.Is(err, llm.ErrEmptyReply) || len(kept) != 1 || kept[0].Err.Error() != "the provider gave an empty reply: llm: empty reply" ||
			thinkingOf(kept[0].Message) != "hmm" {
			t.Fatalf("err %v, kept %+v; want the shown attempt kept with the empty reply's error", err, kept)
		}
	})
}

func keptOverflowAfterText(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "half"}}, Err: llm.ErrContextOverflow}, llmtest.Text("never"))
	cp := &compactor{answers: []compactAnswer{{}, {compacted: summary}}}
	evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Compact: cp.compact}), tool.Env{})
	kept := failedOf(evs)
	if !errors.Is(err, llm.ErrContextOverflow) || !slices.Equal(cp.asks, []bool{false}) || len(kept) != 1 || textOf(kept[0].Message) != "half" {
		t.Fatalf("err %v, asks %v, kept %+v; want no compaction and the shown text kept", err, cp.asks, kept)
	}
}

func keptCancelInWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		time.AfterFunc(time.Second, cancel)
		s := llmtest.NewScript(reasoningOnly(), llmtest.Text("never"))
		evs, end, err := runAll(ctx, agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}}), tool.Env{})
		if err != nil || end.StopReason != agent.StopCancelled {
			t.Fatalf("err %v, end %s; want a cancelled turn", err, end.StopReason)
		}
		var msgs []agent.Message
		for _, ev := range evs {
			if m, ok := ev.(agent.Message); ok {
				msgs = append(msgs, m)
			}
		}
		if len(msgs) != 1 || msgs[0].Err != nil || msgs[0].StopReason != "" || thinkingOf(msgs[0].Message) != "hmm" {
			t.Fatalf("messages %+v, want the shown attempt, no error, its stop cleared", msgs)
		}
		if n := len(end.Messages); n != 2 || thinkingOf(end.Messages[1]) != "hmm" {
			t.Errorf("End's messages %+v, want the prompt and the shown attempt", end.Messages)
		}
	})
}

// An attempt the turn replaces after it showed something is kept as its own
// failed entry, in the turn's messages marked failed and in no request: a
// thinking-only reply run again, whether the rerun answers, fails before
// any event, or overflows and is sent again after a compaction (0070-MADR
// D4; G7, G34, G35).
func TestReplacedAttemptsKept(t *testing.T) {
	t.Run("rerun_succeeds", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := llmtest.NewScript(reasoningOnly(), llmtest.Text("ok"))
			evs, end, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}}), tool.Env{})
			if err != nil {
				t.Fatal(err)
			}
			kept := failedOf(evs)
			if len(kept) != 1 || thinkingOf(kept[0].Message) != "hmm" || !errors.Is(kept[0].Err, llm.ErrEmptyReply) {
				t.Fatalf("kept %+v, want the shown attempt with the empty reply's error", kept)
			}
			if len(end.Messages) != 3 || !end.Messages[1].Origin.Failed || thinkingOf(end.Messages[1]) != "hmm" || textOf(end.Messages[2]) != "ok" {
				t.Fatalf("End's messages %+v, want the prompt, the attempt marked failed, and ok", end.Messages)
			}
			for _, m := range s.Requests()[1].Messages {
				if m.Role == llm.RoleAssistant {
					t.Fatalf("the rerun carries the replaced attempt: %+v", s.Requests()[1].Messages)
				}
			}
		})
	})
	t.Run("rerun_fails_before_event", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := llmtest.NewScript(reasoningOnly(), llmtest.Turn{Err: errBad})
			evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}}), tool.Env{})
			kept := failedOf(evs)
			if !errors.Is(err, errBad) || len(kept) != 2 || thinkingOf(kept[0].Message) != "hmm" || !errors.Is(kept[1].Err, errBad) ||
				len(kept[1].Message.Content) != 0 {
				t.Fatalf("err %v, kept %+v; want the shown attempt, then the empty last call with the 400", err, kept)
			}
		})
	})
	t.Run("overflow_resend", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := llmtest.NewScript(reasoningOnly(), llmtest.Turn{Err: llm.ErrContextOverflow}, llmtest.Text("ok"))
			cp := &compactor{answers: []compactAnswer{{}, {compacted: summary}}}
			evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}, Compact: cp.compact}), tool.Env{})
			if err != nil {
				t.Fatal(err)
			}
			kept := failedOf(evs)
			if len(kept) != 1 || thinkingOf(kept[0].Message) != "hmm" {
				t.Fatalf("kept %+v, want the shown attempt", kept)
			}
			if got := texts(s.Requests()[2].Messages); !slices.Equal(got, []string{"system:rebuilt", "user:SUMMARY"}) {
				t.Fatalf("the resend after the compaction sent %v", got)
			}
			for _, m := range s.Requests()[2].Messages {
				if thinkingOf(m) != "" {
					t.Fatalf("the resend carries the replaced attempt: %+v", s.Requests()[2].Messages)
				}
			}
		})
	})
}

// When a stream fails because the turn was cancelled, the cancel wins: the
// turn ends cancelled, with no error and no failed message (0070-MADR D10,
// G13).
func TestCancelWinsOverFailure(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	p := &cancelling{Script: llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "half"}}, Err: errBad}), cancel: cancel}
	evs, end, err := runAll(ctx, agent.New(agent.Config{Provider: p}), tool.Env{})
	if err != nil || end.StopReason != agent.StopCancelled || len(failedOf(evs)) != 0 {
		t.Fatalf("err %v, end %s, failed %+v; want a cancelled turn", err, end.StopReason, failedOf(evs))
	}
}

// cancelling is a Script that cancels the turn before its stream's failure.
type cancelling struct {
	*llmtest.Script
	cancel func()
}

func (c *cancelling) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	inner := c.Script.Stream(ctx, req)
	return func(yield func(llm.Event, error) bool) {
		for ev, err := range inner {
			if err != nil {
				c.cancel()
			}
			if !yield(ev, err) {
				return
			}
		}
	}
}

// A length cut and a refusal are completed replies: their message carries
// no error (0070-MADR D11; G15, G17).
func TestLengthCutIsNotFailed(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "long"}, llm.Done{StopReason: llm.StopMaxTokens}}})
	evs, end, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: &llm.Retry{AfterCut: true}}), tool.Env{})
	if err != nil || end.StopReason != agent.StopMaxTokens || len(failedOf(evs)) != 0 || len(s.Requests()) != 1 {
		t.Fatalf("err %v, end %s, failed %+v, %d requests; want a completed length cut", err, end.StopReason, failedOf(evs), len(s.Requests()))
	}
}

func TestRefusalIsNotFailed(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "no"}, llm.Done{StopReason: llm.StopRefusal}}})
	evs, end, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: &llm.Retry{AfterCut: true}}), tool.Env{})
	if err != nil || end.StopReason != agent.StopRefusal || len(failedOf(evs)) != 0 || len(s.Requests()) != 1 {
		t.Fatalf("err %v, end %s, failed %+v, %d requests; want a completed refusal", err, end.StopReason, failedOf(evs), len(s.Requests()))
	}
}
