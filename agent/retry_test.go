package agent_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/tool"
)

// probing is a Script that is an llm.Prober: each Probe takes the next of
// answers, the last one again once they run out, and is timed from start.
type probing struct {
	*llmtest.Script
	start   time.Time
	mu      sync.Mutex
	answers []error
	at      []time.Duration
}

func (p *probing) Probe(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.at = append(p.at, time.Since(p.start))
	err := p.answers[min(len(p.at), len(p.answers))-1]
	return err
}

func (p *probing) probes() []time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.at)
}

var (
	errOffline = fmt.Errorf("%w: dial tcp 127.0.0.1:9: connect: connection refused", llm.ErrOffline)
	errDown    = errors.New("still down")
)

// waited is a Retry event and when it came, from the turn's start.
type waited struct {
	agent.Retry
	at time.Duration
}

// runRetry runs a turn of a, timing each Retry event from start, and
// calling during, when set, with each event.
func runRetry(ctx context.Context, t *testing.T, a *agent.Agent, start time.Time) ([]waited, agent.End, error) {
	t.Helper()
	var ws []waited
	var end agent.End
	for ev, err := range a.Run(ctx, tool.Env{}, nil, user("go")) {
		if err != nil {
			return ws, end, err
		}
		switch ev := ev.(type) {
		case agent.Retry:
			ws = append(ws, waited{ev, time.Since(start)})
		case agent.End:
			end = ev
		}
	}
	return ws, end, nil
}

func texts(msgs []llm.Message) []string {
	var out []string
	for _, m := range msgs {
		for _, c := range m.Content {
			if c.Type == llm.ContentText {
				out = append(out, string(m.Role)+":"+c.Text)
			}
		}
	}
	return out
}

// An offline call waits: it probes every 3 s, and the call is sent again
// once a probe answers, with one Retry start and one end around the wait.
// The wait ends at OfflineWait, on cancel and on an abort; with retry off,
// with no wait to take or a provider that cannot probe, the error ends the
// turn at once (0005-PLAN F5-1 decision 3).
func TestOfflineWait(t *testing.T) {
	start := waited{agent.Retry{Phase: agent.RetryStart, Cause: agent.RetryOffline}, 0}
	t.Run("back", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			p := &probing{Script: llmtest.NewScript(llmtest.Turn{Err: errOffline}, llmtest.Text("back")), start: time.Now(),
				answers: []error{errDown, errDown, nil}}
			a := agent.New(agent.Config{Provider: p, Retry: &llm.Retry{OfflineWait: time.Minute}})
			ws, end, err := runRetry(t.Context(), t, a, p.start)
			if err != nil {
				t.Fatal(err)
			}
			want := []waited{start, {agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryOffline}, 9 * time.Second}}
			if !slices.Equal(ws, want) || !slices.Equal(p.probes(), []time.Duration{3 * time.Second, 6 * time.Second, 9 * time.Second}) {
				t.Fatalf("retries %+v, probes %v; want %+v, probes at 3, 6 and 9 s", ws, p.probes(), want)
			}
			if got := texts(end.Messages); end.StopReason != agent.StopEndTurn || !slices.Equal(got, []string{"user:go", "assistant:back"}) ||
				len(p.Requests()) != 2 {
				t.Fatalf("end %s %q after %d requests", end.StopReason, got, len(p.Requests()))
			}
		})
	})
	t.Run("expired", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			p := &probing{Script: llmtest.NewScript(llmtest.Turn{Err: errOffline}), start: time.Now(), answers: []error{errDown}}
			a := agent.New(agent.Config{Provider: p, Retry: &llm.Retry{OfflineWait: 10 * time.Second}})
			ws, _, err := runRetry(t.Context(), t, a, p.start)
			want := []waited{start, {agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryOffline, Err: errOffline.Error()}, 10 * time.Second}}
			if !errors.Is(err, llm.ErrOffline) || !slices.Equal(ws, want) || len(p.probes()) != 3 {
				t.Fatalf("err %v, retries %+v, %d probes; want llm.ErrOffline, %+v, 3 probes", err, ws, len(p.probes()), want)
			}
		})
	})
	t.Run("cancelled", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			p := &probing{Script: llmtest.NewScript(llmtest.Turn{Err: errOffline}), start: time.Now(), answers: []error{errDown}}
			ctx, cancel := context.WithCancel(t.Context())
			time.AfterFunc(4*time.Second, cancel)
			a := agent.New(agent.Config{Provider: p, Retry: &llm.Retry{OfflineWait: time.Minute}})
			ws, end, err := runRetry(ctx, t, a, p.start)
			want := []waited{start, {agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryOffline, Err: "cancelled"}, 4 * time.Second}}
			if err != nil || end.StopReason != agent.StopCancelled || !slices.Equal(ws, want) {
				t.Fatalf("err %v, end %s, retries %+v; want a cancelled turn and %+v", err, end.StopReason, ws, want)
			}
		})
	})
	t.Run("aborted", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			p := &probing{Script: llmtest.NewScript(llmtest.Turn{Err: errOffline}), start: time.Now(), answers: []error{errDown}}
			abort := make(chan struct{})
			time.AfterFunc(4*time.Second, func() { close(abort) })
			a := agent.New(agent.Config{Provider: p, Retry: &llm.Retry{OfflineWait: time.Minute}, RetryAbort: abort})
			ws, _, err := runRetry(t.Context(), t, a, p.start)
			want := []waited{start, {agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryOffline, Err: "retry cancelled"}, 4 * time.Second}}
			if !errors.Is(err, llm.ErrOffline) || !slices.Equal(ws, want) {
				t.Fatalf("err %v, retries %+v; want llm.ErrOffline and %+v", err, ws, want)
			}
		})
	})
	for _, c := range []struct {
		name  string
		retry *llm.Retry
		probe bool
	}{
		{"retry off", &llm.Retry{Off: true, OfflineWait: time.Minute}, true},
		{"no wait", &llm.Retry{}, true},
		{"no settings", nil, true},
		{"no probe", &llm.Retry{OfflineWait: time.Minute}, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				s := llmtest.NewScript(llmtest.Turn{Err: errOffline}, llmtest.Text("never"))
				p := &probing{Script: s, start: time.Now(), answers: []error{nil}}
				var prov llm.Provider = s
				if c.probe {
					prov = p
				}
				ws, _, err := runRetry(t.Context(), t, agent.New(agent.Config{Provider: prov, Retry: c.retry}), p.start)
				if !errors.Is(err, llm.ErrOffline) || len(ws) != 0 || len(p.probes()) != 0 || time.Since(p.start) != 0 {
					t.Fatalf("err %v, retries %+v, probes %v after %s; want llm.ErrOffline at once", err, ws, p.probes(), time.Since(p.start))
				}
			})
		})
	}
}

var errEmptyAnswer = fmt.Errorf("%w: %w: responses: the answer has no content", llm.ErrEmptyReply, llm.ErrIncomplete)

// reasoningOnly is a reply with reasoning, no text and no calls.
func reasoningOnly() llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.ThinkingDelta{Text: "hmm"}, llm.Usage{InputTokens: 10, OutputTokens: 4}, llm.Done{StopReason: llm.StopEndTurn}}}
}

// An empty reply, reasoning only or the SDK's empty answer, runs again after
// 2 s and then 4 s; a third ends the turn. No empty reply reaches the
// turn's messages or a later request, and its usage is counted. A refusal
// and a reply cut at max tokens are not run again (0005-PLAN F5-1 decision
// 4).
func TestEmptyReplyRunsAgain(t *testing.T) {
	first := waited{agent.Retry{Phase: agent.RetryStart, Cause: agent.RetryEmptyReply, Attempt: 1, MaxAttempts: 2, Delay: 2 * time.Second}, 0}
	second := waited{agent.Retry{Phase: agent.RetryStart, Cause: agent.RetryEmptyReply, Attempt: 2, MaxAttempts: 2, Delay: 4 * time.Second}, 2 * time.Second}
	t.Run("then text", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			start := time.Now()
			s := llmtest.NewScript(reasoningOnly(), llmtest.Turn{Err: errEmptyAnswer}, llmtest.Text("ok"))
			ws, end, err := runRetry(t.Context(), t, agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}}), start)
			if err != nil {
				t.Fatal(err)
			}
			want := []waited{first, second, {agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryEmptyReply}, 6 * time.Second}}
			if !slices.Equal(ws, want) {
				t.Fatalf("retries %+v, want %+v", ws, want)
			}
			reqs := s.Requests()
			if len(reqs) != 3 || len(end.Messages) != 2 || !slices.Equal(texts(end.Messages), []string{"user:go", "assistant:ok"}) {
				t.Fatalf("%d requests, messages %q", len(reqs), texts(end.Messages))
			}
			for _, m := range reqs[2].Messages {
				if m.Role == llm.RoleAssistant {
					t.Fatalf("the third request carries an empty reply: %+v", reqs[2].Messages)
				}
			}
			if end.Usage.OutputTokens != 4+2 {
				t.Fatalf("usage %+v, want the reasoning reply's 4 out and ok's 2", end.Usage)
			}
		})
	})
	t.Run("three empty", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			start := time.Now()
			s := llmtest.NewScript(reasoningOnly(), reasoningOnly(), llmtest.Turn{Err: errEmptyAnswer})
			ws, _, err := runRetry(t.Context(), t, agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}}), start)
			const msg = "the provider gave no usable reply 3 times: llm: empty reply"
			want := []waited{first, second, {agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryEmptyReply, Err: msg}, 6 * time.Second}}
			if err == nil || err.Error() != msg || !errors.Is(err, llm.ErrEmptyReply) || !slices.Equal(ws, want) {
				t.Fatalf("err %v, retries %+v; want %q and %+v", err, ws, msg, want)
			}
		})
	})
	t.Run("aborted", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			start := time.Now()
			abort := make(chan struct{})
			time.AfterFunc(time.Second, func() { close(abort) })
			s := llmtest.NewScript(reasoningOnly(), llmtest.Text("never"))
			ws, _, err := runRetry(t.Context(), t, agent.New(agent.Config{Provider: s, Retry: &llm.Retry{}, RetryAbort: abort}), start)
			want := []waited{first, {agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryEmptyReply, Err: "retry cancelled"}, time.Second}}
			if err == nil || err.Error() != "the provider gave an empty reply: llm: empty reply" || !slices.Equal(ws, want) {
				t.Fatalf("err %v, retries %+v; want the empty reply's error and %+v", err, ws, want)
			}
		})
	})
	t.Run("after offline", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			p := &probing{Script: llmtest.NewScript(llmtest.Turn{Err: errOffline}, reasoningOnly(), llmtest.Text("ok")), start: time.Now(),
				answers: []error{nil}}
			ws, _, err := runRetry(t.Context(), t, agent.New(agent.Config{Provider: p, Retry: &llm.Retry{OfflineWait: time.Minute}}), p.start)
			want := []waited{
				{agent.Retry{Phase: agent.RetryStart, Cause: agent.RetryOffline}, 0},
				{agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryOffline}, 3 * time.Second},
				{first.Retry, 3 * time.Second},
				{agent.Retry{Phase: agent.RetryEnd, Cause: agent.RetryEmptyReply}, 5 * time.Second},
			}
			if err != nil || !slices.Equal(ws, want) {
				t.Fatalf("err %v, retries %+v; want %+v", err, ws, want)
			}
		})
	})
	for _, c := range []struct {
		name string
		turn llmtest.Turn
		stop agent.StopReason
	}{
		{"refusal", llmtest.Turn{Err: fmt.Errorf("%w: content_filter", llm.ErrIncomplete)}, ""},
		{"max tokens", llmtest.Turn{Events: []llm.Event{llm.Done{StopReason: llm.StopMaxTokens}}}, agent.StopMaxTokens},
		{"retry off", reasoningOnly(), agent.StopEndTurn},
	} {
		t.Run(c.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				start := time.Now()
				s := llmtest.NewScript(c.turn, llmtest.Text("never"))
				r := &llm.Retry{Off: c.name == "retry off"}
				ws, end, err := runRetry(t.Context(), t, agent.New(agent.Config{Provider: s, Retry: r}), start)
				if c.stop == "" && !errors.Is(err, llm.ErrIncomplete) || c.stop != "" && (err != nil || end.StopReason != c.stop) ||
					len(ws) != 0 || len(s.Requests()) != 1 {
					t.Fatalf("err %v, end %s, retries %+v after %d requests; want one request and no retry", err, end.StopReason, ws, len(s.Requests()))
				}
			})
		})
	}
}
