package llmtest

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
)

func drain(t *testing.T, p llm.Provider, req *llm.Request) ([]llm.Event, error) {
	t.Helper()
	var evs []llm.Event
	for ev, err := range p.Stream(t.Context(), req) {
		if err != nil {
			return evs, err
		}
		evs = append(evs, ev)
	}
	return evs, nil
}

func TestScriptPlaysTurnsInOrder(t *testing.T) {
	s := NewScript(Text("hello"), ToolCall("t1", "read", `{"path":"a"}`), RateLimited(2*time.Second))
	evs, err := drain(t, s, &llm.Request{Model: "m1"})
	if err != nil || len(evs) != 3 || evs[0] != (llm.TextDelta{Text: "hello"}) || evs[2] != (llm.Done{StopReason: llm.StopEndTurn}) {
		t.Fatalf("turn 1 = %#v, %v", evs, err)
	}
	evs, err = drain(t, s, &llm.Request{Model: "m2"})
	done, ok := evs[1].(llm.ToolCallDone)
	if err != nil || !ok || done.ID != "t1" || done.Name != "read" || done.Args != `{"path":"a"}` {
		t.Fatalf("turn 2 = %#v, %v", evs, err)
	}
	_, err = drain(t, s, &llm.Request{Model: "m3"})
	if apiErr, ok := errors.AsType[*llm.APIError](err); !ok || apiErr.Status != 429 || apiErr.RetryAfter != 2*time.Second ||
		!errors.Is(err, llm.ErrRateLimited) {
		t.Fatalf("turn 3 err = %v", err)
	}
	if _, err := drain(t, s, &llm.Request{}); !errors.Is(err, ErrExhausted) {
		t.Fatalf("after the script: %v, want ErrExhausted", err)
	}
	var models []string
	for _, r := range s.Requests() {
		models = append(models, r.Model)
	}
	if !slices.Equal(models, []string{"m1", "m2", "m3", ""}) || s.Remaining() != 0 {
		t.Fatalf("requests %q, remaining %d", models, s.Remaining())
	}
}

func TestScriptStopsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	s := NewScript(Text("hello"))
	var got []llm.Event
	var gotErr error
	for ev, err := range s.Stream(ctx, &llm.Request{}) {
		if err != nil {
			gotErr = err
			break
		}
		got = append(got, ev)
		cancel()
	}
	if len(got) != 1 || !errors.Is(gotErr, context.Canceled) {
		t.Fatalf("events %#v, err %v; want one event, then context.Canceled", got, gotErr)
	}
}

// The recorded request is a copy: changing the caller's after the call does
// not change what the script saw.
func TestScriptRecordsCopies(t *testing.T) {
	s := NewScript(Text("x"))
	req := &llm.Request{Messages: []llm.Message{{Role: llm.RoleUser, Content: []llm.Content{{Type: "text", Text: "before"}}}}}
	if _, err := drain(t, s, req); err != nil {
		t.Fatal(err)
	}
	req.Messages[0].Content[0].Text = "after"
	if got := s.Requests()[0].Messages[0].Content[0].Text; got != "before" {
		t.Fatalf("recorded request changed with the caller's: %q", got)
	}
}

func TestCapabilitiesAndID(t *testing.T) {
	s := NewScript().WithCapabilities(llm.Capabilities{ImageInput: true})
	if s.ID() != "llmtest" || !s.Capabilities().ImageInput {
		t.Fatalf("ID %q, caps %+v", s.ID(), s.Capabilities())
	}
}
