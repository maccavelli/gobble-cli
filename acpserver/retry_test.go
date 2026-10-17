package acpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// prober is a Script whose Probe answers nil: the network is back at the
// first probe.
type prober struct{ *llmtest.Script }

func (prober) Probe(context.Context) error { return nil }

var errOffline = fmt.Errorf("%w: dial tcp 127.0.0.1:9: connect: connection refused", llm.ErrOffline)

func reasoningOnly() llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.ThinkingDelta{Text: "hmm"}, llm.Done{StopReason: llm.StopEndTurn}}}
}

// retryAgent is a connected agent whose provider is p, built with the
// retry settings it is given, which it records; its session settings come
// from settings when set, and SaveSetting records each write.
type retryAgent struct {
	c      *acptest.Conn
	client *acptest.Client

	mu    sync.Mutex
	built []llm.Retry
	saved []string
}

func newRetryAgent(t *testing.T, p llm.Provider, settings func(string) SessionSettings) *retryAgent {
	t.Helper()
	r := &retryAgent{client: &acptest.Client{}}
	ag := New(Options{Version: "1.2.3", Models: func() ModelChoice { return testModels }, Store: jsonl.New(t.TempDir(), jsonl.Options{}),
		Provider: func(_ string, retry llm.Retry) (llm.Provider, error) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.built = append(r.built, retry)
			return p, nil
		},
		Settings: settings,
		SaveSetting: func(path, value string) error {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.saved = append(r.saved, path+"="+value)
			return nil
		}})
	r.c = acptest.Connect(t, ag, r.client)
	ag.SetConnection(r.c.Agent)
	t.Cleanup(func() {
		if err := ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, r.c.Client)
	return r
}

// notices are the _gobble/retry notifications received, each decoded.
func (r *retryAgent) notices(t *testing.T) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, n := range r.client.Notices() {
		if n.Method != methodRetry {
			t.Fatalf("notice %s, want only %s", n.Method, methodRetry)
		}
		var m map[string]any
		if err := json.Unmarshal(n.Params, &m); err != nil {
			t.Fatal(err)
		}
		out = append(out, m)
	}
	return out
}

func notice(id acp.SessionId, phase, cause string, extra map[string]any) map[string]any {
	m := map[string]any{"sessionId": string(id), "phase": phase, "cause": cause}
	maps.Copy(m, extra)
	return m
}

func sameNotices(got, want []map[string]any) bool {
	return slices.EqualFunc(got, want, func(a, b map[string]any) bool { return fmt.Sprint(a) == fmt.Sprint(b) })
}

// Each wait of a turn reaches the client as _gobble/retry, a start before
// it and an end after it, for an offline call and for an empty reply
// (0005-PLAN F5-1 decision 5).
func TestRetryNotice(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := prober{llmtest.NewScript(llmtest.Turn{Err: errOffline}, llmtest.Text("back"), reasoningOnly(), llmtest.Text("ok"))}
		r := newRetryAgent(t, p, nil)
		id := newSession(t, r.c, t.TempDir(), nil).SessionId
		for _, text := range []string{"one", "two"} {
			if resp, err := prompt(t, r.c, id, text); err != nil || resp.StopReason != acp.StopReasonEndTurn {
				t.Fatalf("%s: %+v, %v", text, resp, err)
			}
		}
		want := []map[string]any{
			notice(id, "start", "offline", nil),
			notice(id, "end", "offline", nil),
			notice(id, "start", "empty_reply", map[string]any{"attempt": 1.0, "maxAttempts": 2.0, "delayMs": 2000.0}),
			notice(id, "end", "empty_reply", nil),
		}
		if got := r.notices(t); !sameNotices(got, want) {
			t.Fatalf("notices %v, want %v", got, want)
		}
	})
}

// The session's retry settings reach its provider's build and its turns:
// with retry off, an offline call fails the prompt at once, with no
// notice (0005-PLAN F5-1 decision 2).
func TestRetrySettingsReachSession(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		off := llm.Retry{Off: true, MaxRetries: 7, BaseDelay: time.Second, MaxDelay: 2 * time.Second, OfflineWait: time.Minute}
		r := newRetryAgent(t, prober{llmtest.NewScript(llmtest.Turn{Err: errOffline})},
			func(string) SessionSettings { return SessionSettings{Retry: off} })
		id := newSession(t, r.c, t.TempDir(), nil).SessionId
		start := time.Now()
		_, err := prompt(t, r.c, id, "go")
		if code, text := errText(t, err); code != -32603 || text != errOffline.Error() || time.Since(start) != 0 {
			t.Fatalf("prompt: %d %q after %s, want -32603 with the offline error at once", code, text, time.Since(start))
		}
		if len(r.built) != 1 || r.built[0] != off || len(r.notices(t)) != 0 {
			t.Fatalf("built with %+v, notices %v; want %+v once and none", r.built, r.notices(t), off)
		}
	})
}

// _gobble/set_auto_retry writes retry.enabled through SaveSetting and sets
// the session's: off, a turn takes no wait, and a wait under way ends at
// once; on again, the next turn waits. enabled must be given
// (0005-PLAN F5-1 decision 6).
func TestSetAutoRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		p := prober{llmtest.NewScript(llmtest.Turn{Err: errOffline}, llmtest.Turn{Err: errOffline}, llmtest.Text("back"))}
		r := newRetryAgent(t, p, nil)
		id := newSession(t, r.c, t.TempDir(), nil).SessionId
		if _, err := ext(t, r.c, "_gobble/set_auto_retry", map[string]any{"sessionId": id}); err == nil {
			t.Fatal("set_auto_retry with no enabled succeeded")
		} else if code, _ := errText(t, err); code != -32602 {
			t.Fatalf("no enabled: code %d, want -32602", code)
		}
		mustExt(t, r.c, "_gobble/set_auto_retry", map[string]any{"sessionId": id, "enabled": false})
		if _, err := prompt(t, r.c, id, "off"); err == nil || len(r.notices(t)) != 0 {
			t.Fatalf("retry off: err %v, notices %v; want the offline error and none", err, r.notices(t))
		}
		mustExt(t, r.c, "_gobble/set_auto_retry", map[string]any{"sessionId": id, "enabled": true})
		if resp, err := prompt(t, r.c, id, "on"); err != nil || resp.StopReason != acp.StopReasonEndTurn {
			t.Fatalf("retry on: %+v, %v", resp, err)
		}
		if want := []string{"retry.enabled=false", "retry.enabled=true"}; !slices.Equal(r.saved, want) {
			t.Fatalf("saved %q, want %q", r.saved, want)
		}
		if got := r.notices(t); len(got) != 2 {
			t.Fatalf("notices %v, want the second turn's start and end", got)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		r := newRetryAgent(t, prober{llmtest.NewScript(llmtest.Turn{Err: errOffline}, llmtest.Text("never"))},
			func(string) SessionSettings {
				return SessionSettings{Retry: llm.Retry{BaseDelay: time.Second, OfflineWait: time.Minute}}
			})
		id := newSession(t, r.c, t.TempDir(), nil).SessionId
		turnOff(t, r, id, "_gobble/set_auto_retry", map[string]any{"sessionId": id, "enabled": false})
	})
}

// _gobble/abort_retry ends a wait under way, and the prompt fails with the
// error that caused it; with no wait, it changes nothing (0005-PLAN F5-1
// decision 6).
func TestAbortRetry(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		r := newRetryAgent(t, prober{llmtest.NewScript(reasoningOnly(), llmtest.Text("ok"))}, nil)
		id := newSession(t, r.c, t.TempDir(), nil).SessionId
		mustExt(t, r.c, "_gobble/abort_retry", map[string]any{"sessionId": id})
		turnOff(t, r, id, "_gobble/abort_retry", map[string]any{"sessionId": id})
	})
}

// turnOff starts a prompt whose first call fails offline or replies empty,
// so it waits; calls method with params during the wait, before the wait's
// first probe or rerun; and checks the prompt fails, the wait's end saying
// retry cancelled.
func turnOff(t *testing.T, r *retryAgent, id acp.SessionId, method string, params map[string]any) {
	t.Helper()
	done := make(chan error, 1)
	go func() {
		_, err := prompt(t, r.c, id, "go")
		done <- err
	}()
	synctest.Wait() // the turn waits on its first empty reply, or on the network
	if got := r.notices(t); len(got) != 1 || got[0]["phase"] != "start" {
		t.Fatalf("before %s: notices %v, want one start", method, got)
	}
	mustExt(t, r.c, method, params)
	err := <-done
	if err == nil {
		t.Fatalf("the prompt succeeded after %s", method)
	}
	if got := r.notices(t); len(got) != 2 || got[1]["phase"] != "end" || got[1]["error"] != "retry cancelled" {
		t.Fatalf("after %s: notices %v, want the wait's end as retry cancelled", method, got)
	}
	var re *acp.RequestError
	if !errors.As(err, &re) || re.Code != -32603 {
		t.Fatalf("prompt error %v, want -32603", err)
	}
}
