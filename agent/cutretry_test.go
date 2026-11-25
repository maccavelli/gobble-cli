package agent_test

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"reflect"
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

// cutRetry is the session's defaults as the cut retry reads them.
func cutRetry() *llm.Retry {
	r := llm.DefaultRetry
	return &r
}

// cutAfter is a call that streams text and is then cut by err.
func cutAfter(text string, err error) llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: text}}, Err: err}
}

var errCut = fmt.Errorf("%w: stream: read: connection reset by peer", llm.ErrUnavailable)

// retries are the Retry events among evs, each with when it came.
func retriesOf(evs []timed) []waited {
	var out []waited
	for _, e := range evs {
		if r, ok := e.ev.(agent.Retry); ok {
			out = append(out, waited{r, e.at})
		}
	}
	return out
}

// timed is an event and when it came, from the turn's start.
type timed struct {
	ev agent.Event
	at time.Duration
}

// runTimed runs a turn of a on its clock, as runAll does, timing each event.
func runTimed(ctx context.Context, a *agent.Agent, env tool.Env) ([]timed, agent.End, error) {
	start := time.Now()
	var evs []timed
	var end agent.End
	for ev, err := range a.Run(ctx, env, nil, user("go")) {
		if err != nil {
			return evs, end, err
		}
		evs = append(evs, timed{ev, time.Since(start)})
		if e, ok := ev.(agent.End); ok {
			end = e
		}
	}
	return evs, end, nil
}

func eventsOf(evs []timed) []agent.Event {
	out := make([]agent.Event, len(evs))
	for i, e := range evs {
		out[i] = e.ev
	}
	return out
}

func cutStart(n int, d time.Duration) agent.Retry {
	return agent.Retry{Phase: agent.RetryStart, Cause: agent.RetryCut, Attempt: n, MaxAttempts: 3, Delay: d}
}

var cutEnd = agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryCut}

// A call an accidental failure cuts after any event is sent again, the same
// request, after the session's delay doubled at each, or the wait for the
// network; at most 3 in a row, the count back to 0 when a call ends; the cut
// part, when shown, is its own failed entry. Any other failure, a refusal, a
// length cut, a cut with no event, ends as before (0070-MADR D12; C1 to C5,
// C7, C8, C10, C11, C13 to C18, C26, C27).
func TestRetryAfterCut(t *testing.T) {
	t.Run("unavailable", cutUnavailable)
	t.Run("offline", cutOffline)
	t.Run("not_accidental", cutNotAccidental)
	t.Run("refusal", cutRefusal)
	t.Run("events_not_shown", cutEventsNotShown)
	t.Run("after_released_call", cutAfterReleasedCall)
	t.Run("no_event", cutNoEvent)
	t.Run("backoff", cutBackoff)
	t.Run("cancel_in_wait", cutCancelInWait)
	t.Run("then_empty", cutThenEmpty)
	t.Run("chain", cutChain)
	t.Run("cap", cutCap)
	t.Run("then_fails", cutThenFails)
	t.Run("then_length", cutThenLength)
	t.Run("then_overflow", cutThenOverflow)
	t.Run("cancelled", cutCancelled)
	t.Run("limit_refused_on_retry", cutLimitRefusedOnRetry)
	t.Run("limit_refused_after_resend", cutLimitRefusedAfterResend)
}

func cutUnavailable(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("half", errCut), llmtest.Text("whole"))
		evs, end, err := runTimed(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		if err != nil || end.StopReason != agent.StopEndTurn {
			t.Fatalf("err %v, end %s; want the turn to end well", err, end.StopReason)
		}
		want := []waited{{cutStart(1, 2*time.Second), 0}, {cutEnd, 2 * time.Second}}
		if got := retriesOf(evs); !slices.Equal(got, want) {
			t.Fatalf("retries %+v, want %+v", got, want)
		}
		kept := failedOf(eventsOf(evs))
		if len(kept) != 1 || textOf(kept[0].Message) != "half" || !errors.Is(kept[0].Err, llm.ErrUnavailable) {
			t.Fatalf("kept %+v, want the cut part", kept)
		}
		if got := texts(end.Messages); !slices.Equal(got, []string{"user:go", "assistant:half", "assistant:whole"}) || !end.Messages[1].Origin.Failed {
			t.Fatalf("End's messages %q (%+v), want the cut part marked failed, then whole", got, end.Messages)
		}
		if reqs := s.Requests(); len(reqs) != 2 || !reflect.DeepEqual(reqs[0], reqs[1]) {
			t.Fatalf("%d requests, want the same one twice", len(reqs))
		}
	})
}

func cutOffline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := &probing{Script: llmtest.NewScript(cutAfter("half", errOffline), llmtest.Text("whole")), start: time.Now(), answers: []error{nil}}
		evs, _, err := runTimed(t.Context(), agent.New(agent.Config{Provider: p, Retry: cutRetry()}), tool.Env{})
		want := []waited{{cutStart(1, 0), 0}, {cutEnd, 3 * time.Second}}
		if got := retriesOf(evs); err != nil || !slices.Equal(got, want) || !slices.Equal(p.probes(), []time.Duration{3 * time.Second}) {
			t.Fatalf("err %v, retries %+v, probes %v; want %+v after one probe", err, got, p.probes(), want)
		}
	})
}

func cutNotAccidental(t *testing.T) {
	for _, kind := range []error{llm.ErrWorkspace, llm.ErrQuota, llm.ErrRateLimited, llm.ErrContextOverflow, llm.ErrEmptyReply,
		llm.ErrIncomplete, llm.ErrAuth, llm.ErrNotPermitted, llm.ErrUnsupported, context.DeadlineExceeded, errors.New("other")} {
		s := llmtest.NewScript(cutAfter("half", fmt.Errorf("scripted: %w", kind)), llmtest.Text("never"))
		evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		if !errors.Is(err, kind) || len(s.Requests()) != 1 || len(failedOf(evs)) != 1 {
			t.Errorf("%s: err %v after %d requests; want the error, one request, the call kept", llm.Kind(kind), err, len(s.Requests()))
		}
	}
}

func cutRefusal(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "no"}, llm.Done{StopReason: llm.StopRefusal}}}, llmtest.Text("never"))
	_, end, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
	if err != nil || end.StopReason != agent.StopRefusal || len(s.Requests()) != 1 {
		t.Fatalf("err %v, end %s, %d requests; want the refusal, one request", err, end.StopReason, len(s.Requests()))
	}
}

func cutEventsNotShown(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{hiddenReasoning}, Err: errCut}, llmtest.Text("whole"))
		evs, end, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		if err != nil || len(s.Requests()) != 2 || len(failedOf(evs)) != 0 || !slices.Equal(texts(end.Messages), []string{"user:go", "assistant:whole"}) {
			t.Fatalf("err %v, %d requests, failed %+v, messages %q; want a retry and nothing kept", err, len(s.Requests()), failedOf(evs), texts(end.Messages))
		}
	})
}

func cutAfterReleasedCall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		call := llm.ToolCallDone{ID: "c1", Name: "read", Args: `{"path":"x"}`}
		s := llmtest.NewScript(llmtest.Turn{Events: []llm.Event{call}, Err: errCut}, llmtest.Text("whole"))
		evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Retry: cutRetry()}), tool.Env{Cwd: t.TempDir()})
		reqs := s.Requests()
		if err != nil || starts(evs) != 0 || len(reqs) != 2 || !reflect.DeepEqual(reqs[0], reqs[1]) {
			t.Fatalf("err %v, %d tool starts, %d requests; want no call run and the same request again", err, starts(evs), len(reqs))
		}
	})
}

func cutNoEvent(t *testing.T) {
	s := llmtest.NewScript(llmtest.Turn{Err: errCut}, llmtest.Text("never"))
	_, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
	if !errors.Is(err, llm.ErrUnavailable) || len(s.Requests()) != 1 {
		t.Fatalf("err %v after %d requests; want the error after one", err, len(s.Requests()))
	}
}

func cutBackoff(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("a", errCut), cutAfter("b", errCut), cutAfter("c", errCut), llmtest.Text("whole"))
		evs, _, err := runTimed(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		want := []waited{{cutStart(1, 2*time.Second), 0}, {cutEnd, 2 * time.Second}, {cutStart(2, 4*time.Second), 2 * time.Second},
			{cutEnd, 6 * time.Second}, {cutStart(3, 8*time.Second), 6 * time.Second}, {cutEnd, 14 * time.Second}}
		if got := retriesOf(evs); err != nil || !slices.Equal(got, want) {
			t.Fatalf("err %v, retries %+v; want %+v", err, got, want)
		}
	})
}

func cutCancelInWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		time.AfterFunc(time.Second, cancel)
		s := llmtest.NewScript(cutAfter("half", errCut), llmtest.Text("never"))
		evs, end, err := runAll(ctx, agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		kept := failedOf(evs)
		if err != nil || end.StopReason != agent.StopCancelled || len(kept) != 1 || len(s.Requests()) != 1 {
			t.Fatalf("err %v, end %s, kept %+v, %d requests; want a cancelled turn with the cut part kept", err, end.StopReason, kept, len(s.Requests()))
		}
		if n := len(evs); n < 2 || !isRetry(evs[n-3], agent.RetryStart) {
			t.Fatalf("events %+v: want the cut part kept before the wait", evs)
		}
	})
}

func cutThenEmpty(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("a", errCut), reasoningOnly(), cutAfter("b", errCut), cutAfter("c", errCut), cutAfter("d", errCut),
			llmtest.Text("whole"))
		_, end, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		if err != nil || end.StopReason != agent.StopEndTurn || len(s.Requests()) != 6 {
			t.Fatalf("err %v, end %s after %d requests; want the empty reply run again on its own count", err, end.StopReason, len(s.Requests()))
		}
	})
}

func cutChain(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("a", errCut), cutAfter("b", errCut), llmtest.Text("whole"))
		evs, _, err := runTimed(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		var attempts []int
		for _, w := range retriesOf(evs) {
			if w.Phase == agent.RetryStart {
				attempts = append(attempts, w.Attempt)
			}
		}
		if err != nil || !slices.Equal(attempts, []int{1, 2}) || len(failedOf(eventsOf(evs))) != 2 {
			t.Fatalf("err %v, attempts %v, kept %d; want attempts 1 and 2, both cut parts kept", err, attempts, len(failedOf(eventsOf(evs))))
		}
	})
}

func cutCap(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("a", errCut), cutAfter("b", errCut), cutAfter("c", errCut), cutAfter("d", errCut), llmtest.Text("never"))
		evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		kept := failedOf(evs)
		if !errors.Is(err, llm.ErrUnavailable) || len(s.Requests()) != 4 || len(kept) != 4 || textOf(kept[3].Message) != "d" {
			t.Fatalf("err %v, %d requests, %d kept; want the fourth cut to end the turn, each call kept once", err, len(s.Requests()), len(kept))
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// A call that ends ends the run: the next round may be cut 3 times again.
		s := llmtest.NewScript(cutAfter("a", errCut), cutAfter("b", errCut), llmtest.ToolCall("1", "read", `{"path":"x"}`),
			cutAfter("c", errCut), cutAfter("d", errCut), cutAfter("e", errCut), llmtest.Text("whole"))
		_, end, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Tools: builtin.Tools(), Retry: cutRetry()}), tool.Env{Cwd: t.TempDir()})
		if err != nil || end.StopReason != agent.StopEndTurn || len(s.Requests()) != 7 {
			t.Fatalf("err %v, end %s after %d requests; want the count reset by the call that ended", err, end.StopReason, len(s.Requests()))
		}
	})
}

func cutThenFails(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("half", errCut), cutAfter("again", errBad), llmtest.Text("never"))
		evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		kept := failedOf(evs)
		if !errors.Is(err, errBad) || len(s.Requests()) != 2 || len(kept) != 2 || !errors.Is(kept[1].Err, errBad) {
			t.Fatalf("err %v, %d requests, kept %+v; want the 400 to end the turn, its call kept", err, len(s.Requests()), kept)
		}
	})
}

func cutThenLength(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("half", errCut), llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "long"}, llm.Done{StopReason: llm.StopMaxTokens}}},
			llmtest.Text("never"))
		_, end, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry()}), tool.Env{})
		if err != nil || end.StopReason != agent.StopMaxTokens || len(s.Requests()) != 2 {
			t.Fatalf("err %v, end %s after %d requests; want max_tokens after two", err, end.StopReason, len(s.Requests()))
		}
	})
}

func cutThenOverflow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("half", errCut), llmtest.Turn{Err: llm.ErrContextOverflow}, llmtest.Text("whole"))
		cp := &compactor{answers: []compactAnswer{{}, {compacted: summary}}}
		_, end, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry(), Compact: cp.compact}), tool.Env{})
		if err != nil || end.StopReason != agent.StopEndTurn || !slices.Equal(cp.asks, []bool{false, true}) || len(s.Requests()) != 3 {
			t.Fatalf("err %v, end %s, asks %v, %d requests; want the overflow compacted and sent again", err, end.StopReason, cp.asks, len(s.Requests()))
		}
	})
}

func cutCancelled(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		p := &cancelSecond{Script: llmtest.NewScript(cutAfter("half", errCut), llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "who"},
			llm.TextDelta{Text: "le"}}}), cancel: cancel}
		_, end, err := runAll(ctx, agent.New(agent.Config{Provider: p, Retry: cutRetry()}), tool.Env{})
		if err != nil || end.StopReason != agent.StopCancelled {
			t.Fatalf("err %v, end %s; want a cancelled turn", err, end.StopReason)
		}
		if got := texts(end.Messages); !slices.Equal(got, []string{"user:go", "assistant:half", "assistant:who"}) || !end.Messages[1].Origin.Failed {
			t.Fatalf("End's messages %q (%+v), want the cut part kept failed, then the cancelled call", got, end.Messages)
		}
	})
}

func cutLimitRefusedOnRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("half", errCut), limitRefused(32000), llmtest.Text("whole"))
		_, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry(), OutputLimit: fixed}), tool.Env{})
		if got := sentCaps(s.Requests()); err != nil || !slices.Equal(got, []int{60000, 60000, 32000}) {
			t.Fatalf("err %v, caps %v; want the retry's refusal sent again once at 32,000", err, got)
		}
	})
}

func cutLimitRefusedAfterResend(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(limitRefused(32000), cutAfter("half", errCut), refused("max_tokens: 32000 > 16000, which is the maximum allowed number of output tokens for m1", nil),
			llmtest.Text("never"))
		_, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry(), OutputLimit: fixed}), tool.Env{})
		if got := sentCaps(s.Requests()); err == nil || !slices.Equal(got, []int{60000, 32000, 32000}) {
			t.Fatalf("err %v, caps %v; want the second refusal to end the turn", err, got)
		}
	})
}

// limitRefused is a refusal of a limit of 60,000, stating stated.
func limitRefused(stated int) llmtest.Turn {
	return refused(fmt.Sprintf("max_tokens: 60000 > %d, which is the maximum allowed number of output tokens for m1", stated), nil)
}

// fixed is an OutputLimit that asks 60,000, or a refused limit's stated
// value.
func fixed(req *llm.Request) llm.OutputLimit {
	if req.MaxTokens > 0 {
		return llm.OutputLimit{Tokens: req.MaxTokens, Basis: llm.BasisRefusal}
	}
	return llm.OutputLimit{Tokens: 60000, Basis: llm.BasisCeiling}
}

func isRetry(e agent.Event, phase agent.RetryPhase) bool {
	r, ok := e.(agent.Retry)
	return ok && r.Phase == phase && r.Cause == agent.RetryCut
}

// placed is a Script that records each call's place in a run of cut
// retries.
type placed struct {
	*llmtest.Script
	places []int
}

func (p *placed) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	p.places = append(p.places, llm.CutRetry(ctx))
	return p.Script.Stream(ctx, req)
}

// cancelSecond is a Script that cancels the turn after the first event of
// its second call.
type cancelSecond struct {
	*llmtest.Script
	cancel func()
	n      int
}

func (c *cancelSecond) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	c.n++
	n := c.n
	inner := c.Script.Stream(ctx, req)
	return func(yield func(llm.Event, error) bool) {
		for ev, err := range inner {
			if !yield(ev, err) {
				return
			}
			if n == 2 {
				c.cancel()
			}
		}
	}
}

// The call sent again after a cut is the cut call's request: the same
// messages, nothing of the attempt in them, and the limit it carried, which
// is not asked again (0070-MADR D12 (b) as amended, D7; C6, C25).
func TestRetryAfterCutRequest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := llmtest.NewScript(cutAfter("half", errCut), llmtest.Text("whole"))
		if _, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, System: "sys", Retry: cutRetry()}), tool.Env{}); err != nil {
			t.Fatal(err)
		}
		reqs := s.Requests()
		if len(reqs) != 2 || !reflect.DeepEqual(reqs[0], reqs[1]) {
			t.Fatalf("requests %+v, want the same one twice", reqs)
		}
	})
	t.Run("cut_retry_context", func(t *testing.T) {
		// The call sent again carries its place in the run of cut retries,
		// for its stream-health record (0070-MADR D17; C23).
		synctest.Test(t, func(t *testing.T) {
			p := &placed{Script: llmtest.NewScript(cutAfter("a", errCut), cutAfter("b", errCut), llmtest.Text("whole"))}
			if _, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: p, Retry: cutRetry()}), tool.Env{}); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(p.places, []int{0, 1, 2}) {
				t.Fatalf("places %v, want 0, 1, 2", p.places)
			}
		})
	})
	t.Run("keeps_limit", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			asks := 0
			less := func(*llm.Request) llm.OutputLimit {
				asks++
				return llm.OutputLimit{Tokens: 10000 - asks, Basis: llm.BasisWindow}
			}
			s := llmtest.NewScript(cutAfter("half", errCut), llmtest.Text("whole"))
			if _, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry(), OutputLimit: less}), tool.Env{}); err != nil {
				t.Fatal(err)
			}
			if got := sentCaps(s.Requests()); !slices.Equal(got, []int{9999, 9999}) || asks != 1 {
				t.Fatalf("caps %v after %d asks; want the first answer twice, asked once", got, asks)
			}
		})
	})
}

// The cut retry is off with retry.retryAfterCut off, with retries off, and
// with no retry settings: the cut ends the turn (0070-MADR D12 (f); C19).
func TestRetryAfterCutSetting(t *testing.T) {
	off := *cutRetry()
	off.AfterCut = false
	allOff := *cutRetry()
	allOff.Off = true
	for name, r := range map[string]*llm.Retry{"retryAfterCut off": &off, "retries off": &allOff, "no settings": nil} {
		s := llmtest.NewScript(cutAfter("half", errCut), llmtest.Text("never"))
		evs, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: s, Retry: r}), tool.Env{})
		if !errors.Is(err, llm.ErrUnavailable) || len(s.Requests()) != 1 || len(failedOf(evs)) != 1 {
			t.Errorf("%s: err %v after %d requests; want the cut to end the turn, its call kept", name, err, len(s.Requests()))
		}
	}
}

// cutEnds are the ends of the cut notices among evs, by their Err.
func cutEnds(evs []timed) []string {
	var out []string
	for _, w := range retriesOf(evs) {
		if w.Phase == agent.RetryEnd && w.Cause == agent.RetryCut {
			out = append(out, w.Err)
		}
	}
	return out
}

// When the call sent again after a cut fails before any event, the cut's
// notice ends at that failure, with its text, whichever path the turn takes
// next: an offline wait, a limit refusal's resend, an overflow's compaction
// (0070-MADR D12 (d); C12, the review of 2026-11-24).
func TestRetryAfterCutNoticeEnds(t *testing.T) {
	t.Run("offline", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			p := &probing{Script: llmtest.NewScript(cutAfter("half", errCut), llmtest.Turn{Err: errOffline}, llmtest.Text("whole")),
				start: time.Now(), answers: []error{nil}}
			evs, _, err := runTimed(t.Context(), agent.New(agent.Config{Provider: p, Retry: cutRetry()}), tool.Env{})
			if got := cutEnds(evs); err != nil || !slices.Equal(got, []string{errOffline.Error()}) {
				t.Fatalf("err %v, cut ends %q; want one, with the offline failure", err, got)
			}
		})
	})
	t.Run("limit_refused", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			refusal := limitRefused(32000)
			s := llmtest.NewScript(cutAfter("half", errCut), refusal, llmtest.Text("whole"))
			evs, _, err := runTimed(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry(), OutputLimit: fixed}), tool.Env{})
			if got := cutEnds(evs); err != nil || len(got) != 1 || !strings.Contains(got[0], "max_tokens: 60000 > 32000") {
				t.Fatalf("err %v, cut ends %q; want one, with the refusal", err, got)
			}
		})
	})
	t.Run("overflow", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := llmtest.NewScript(cutAfter("half", errCut), llmtest.Turn{Err: llm.ErrContextOverflow}, llmtest.Text("whole"))
			cp := &compactor{answers: []compactAnswer{{}, {compacted: summary}}}
			evs, _, err := runTimed(t.Context(), agent.New(agent.Config{Provider: s, Retry: cutRetry(), Compact: cp.compact}), tool.Env{})
			if got := cutEnds(evs); err != nil || !slices.Equal(got, []string{llm.ErrContextOverflow.Error()}) {
				t.Fatalf("err %v, cut ends %q; want one, with the overflow", err, got)
			}
		})
	})
}

// Every resend of a cut call is the cut call, its limit kept: the resend
// after an offline wait that follows a cut retry failed before any event
// asks for no limit again (0070-MADR D12 (b), Q8 with Q9 as the coordinator
// read them, 2026-11-24; C25).
func TestRetryAfterCutKeepsLimitAfterOffline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		asks := 0
		less := func(*llm.Request) llm.OutputLimit {
			asks++
			return llm.OutputLimit{Tokens: 10000 - asks, Basis: llm.BasisWindow}
		}
		p := &probing{Script: llmtest.NewScript(cutAfter("half", errCut), llmtest.Turn{Err: errOffline}, llmtest.Text("whole")),
			start: time.Now(), answers: []error{nil}}
		if _, _, err := runAll(t.Context(), agent.New(agent.Config{Provider: p, Retry: cutRetry(), OutputLimit: less}), tool.Env{}); err != nil {
			t.Fatal(err)
		}
		if got := sentCaps(p.Requests()); !slices.Equal(got, []int{9999, 9999, 9999}) || asks != 1 {
			t.Fatalf("caps %v after %d asks; want the cut call's limit three times, asked once", got, asks)
		}
	})
}

// An offline cut with no prober, or with no wait for the network, takes the
// short wait, the doubled BaseDelay, and is sent again (0070-MADR C29, the
// owner's "Keep the short wait and resend").
func TestRetryAfterCutOfflineNoWait(t *testing.T) {
	check := func(t *testing.T, p llm.Provider, s *llmtest.Script, r *llm.Retry) {
		t.Helper()
		evs, end, err := runTimed(t.Context(), agent.New(agent.Config{Provider: p, Retry: r}), tool.Env{})
		want := []waited{{cutStart(1, 2*time.Second), 0}, {cutEnd, 2 * time.Second}}
		if got := retriesOf(evs); err != nil || end.StopReason != agent.StopEndTurn || !slices.Equal(got, want) || len(s.Requests()) != 2 {
			t.Fatalf("err %v, end %s, retries %+v after %d requests; want %+v and the call sent again", err, end.StopReason, got, len(s.Requests()), want)
		}
	}
	t.Run("no_prober", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := llmtest.NewScript(cutAfter("half", errOffline), llmtest.Text("whole"))
			check(t, s, s, cutRetry())
		})
	})
	t.Run("no_offline_wait", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			s := llmtest.NewScript(cutAfter("half", errOffline), llmtest.Text("whole"))
			r := cutRetry()
			r.OfflineWait = 0
			check(t, &probing{Script: s, start: time.Now(), answers: []error{nil}}, s, r)
		})
	})
}
