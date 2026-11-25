package provider

import (
	"encoding/json/v2"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// The tests of a cut answer at go-llmprovider-sdk v1.33.0, through gobble's
// adapter (0071-MADR P9 to P12, P29; 0065-MADR R20, R22 to R25). Each reply
// is this file's own, in the shape of the SDK's cut-answer tests
// (llmprovider/providers/cut_answer_0088_test.go at v1.33.0); none is a
// recorded one.

// cutCall is a call of a cut reply: its id, name and arguments as the
// service sent them. A cut call's arguments are not whole JSON.
type cutCall struct{ id, name, args string }

// cutReply is a reply the output limit cut: its reasoning, text and calls.
type cutReply struct {
	reasoning, text string
	calls           []cutCall
}

// jsonOf is v as JSON.
func jsonOf(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// chatCut is r as a Chat Completions stream that ends finish_reason
// length.
func chatCut(t *testing.T, r cutReply) string {
	var b strings.Builder
	chunk := func(delta map[string]any, finish any) {
		b.WriteString(sse("", jsonOf(t, map[string]any{"id": "c", "model": "m",
			"choices": []any{map[string]any{"index": 0, "delta": delta, "finish_reason": finish}}})))
	}
	if r.reasoning != "" {
		chunk(map[string]any{"reasoning": r.reasoning}, nil)
	}
	if r.text != "" {
		chunk(map[string]any{"content": r.text}, nil)
	}
	for i, c := range r.calls {
		chunk(map[string]any{"tool_calls": []any{map[string]any{"index": i, "id": c.id, "type": "function",
			"function": map[string]any{"name": c.name, "arguments": c.args}}}}, nil)
	}
	chunk(map[string]any{}, "length")
	b.WriteString(sse("", jsonOf(t, map[string]any{"id": "c", "model": "m", "choices": []any{},
		"usage": map[string]any{"prompt_tokens": 3, "completion_tokens": 2}})))
	b.WriteString(sse("", "[DONE]"))
	return b.String()
}

// messagesCut is r as a Messages stream that ends stop_reason max_tokens;
// a tool_use that is the last block is the call the limit cut.
func messagesCut(t *testing.T, r cutReply) string {
	var b strings.Builder
	ev := func(v map[string]any) { b.WriteString(sse(v["type"].(string), jsonOf(t, v))) }
	ev(map[string]any{"type": "message_start", "message": map[string]any{"id": "m", "type": "message", "role": "assistant",
		"content": []any{}, "model": "claude-haiku-5-5", "usage": map[string]any{"input_tokens": 3, "output_tokens": 0}}})
	i := 0
	block := func(start, delta map[string]any) {
		ev(map[string]any{"type": "content_block_start", "index": i, "content_block": start})
		if delta != nil {
			ev(map[string]any{"type": "content_block_delta", "index": i, "delta": delta})
		}
		ev(map[string]any{"type": "content_block_stop", "index": i})
		i++
	}
	if r.text != "" {
		block(map[string]any{"type": "text", "text": ""}, map[string]any{"type": "text_delta", "text": r.text})
	}
	for _, c := range r.calls {
		block(map[string]any{"type": "tool_use", "id": c.id, "name": c.name, "input": map[string]any{}},
			map[string]any{"type": "input_json_delta", "partial_json": c.args})
	}
	ev(map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": "max_tokens"}, "usage": map[string]any{"output_tokens": 2}})
	ev(map[string]any{"type": "message_stop"})
	return b.String()
}

// responsesCutText is a Responses stream of text, then response.incomplete
// for max_output_tokens.
func responsesCutText(t *testing.T, text string) string {
	item := map[string]any{"type": "message", "role": "assistant", "status": "incomplete",
		"content": []any{map[string]any{"type": "output_text", "text": text}}}
	return sse("response.created", jsonOf(t, map[string]any{"type": "response.created", "response": map[string]any{"id": "r", "model": "gpt-6-luna"}})) +
		sse("response.output_text.delta", jsonOf(t, map[string]any{"type": "response.output_text.delta", "output_index": 0, "content_index": 0, "delta": text})) +
		sse("response.output_item.done", jsonOf(t, map[string]any{"type": "response.output_item.done", "output_index": 0, "item": item})) +
		sse("response.incomplete", jsonOf(t, map[string]any{"type": "response.incomplete", "response": map[string]any{"id": "r", "model": "gpt-6-luna",
			"status": "incomplete", "incomplete_details": map[string]any{"reason": "max_output_tokens"},
			"usage": map[string]any{"input_tokens": 3, "output_tokens": 2}}}))
}

// cutOutcome is what one stream through the adapter gave.
type cutOutcome struct {
	text     string
	calls    []string // name and arguments
	warnings []string // kinds
	dones    []llm.Done
	err      error
}

// cutStream streams req through provider id, model, over a wire that
// answers every streamed request with stream.
func cutStream(t *testing.T, id, model, stream string) cutOutcome {
	t.Helper()
	p, err := New(id, Options{APIKey: "k", Model: model, HTTPClient: &http.Client{Transport: &cannedWire{status: http.StatusOK, stream: stream}},
		Retry: &llm.Retry{Off: true}})
	if err != nil {
		t.Fatal(err)
	}
	req := assumedLimit(wireRequest())
	req.Model = model
	evs, err := drain(t, p, req)
	o := cutOutcome{err: err}
	for _, ev := range evs {
		switch ev := ev.(type) {
		case llm.TextDelta:
			o.text += ev.Text
		case llm.ToolCallDone:
			o.calls = append(o.calls, ev.Name+" "+ev.Args)
		case llm.Warning:
			o.warnings = append(o.warnings, ev.Kind)
		case llm.Done:
			o.dones = append(o.dones, ev)
		}
	}
	return o
}

// stop is the outcome's one Done's stop, or "" when there is not exactly
// one.
func (o cutOutcome) stop() llm.StopReason {
	if len(o.dones) != 1 {
		return ""
	}
	return o.dones[0].StopReason
}

// A Chat Completions answer the limit cut after a whole call: the whole call
// is yielded, the cut one dropped and named in the SDK's tool_call_cut
// warning, and the reply ends tool_use, so the turn runs the call (0071-MADR
// P9; 0065-MADR R24).
func TestAgentTurn_SalvageCompleteCalls_0065(t *testing.T) {
	o := cutStream(t, "kilo", "deepseek/deepseek-v4.1-flash", chatCut(t, cutReply{calls: []cutCall{
		{"c1", "read", `{"path":"x"}`}, {"c2", "read", `{"pa`}}}))
	if o.err != nil || !slices.Equal(o.calls, []string{`read {"path":"x"}`}) || o.stop() != llm.StopToolUse {
		t.Fatalf("calls %q, stop %q, err %v; want the whole call alone, then tool_use", o.calls, o.stop(), o.err)
	}
	// 0071 new names: begin
	if !slices.Equal(o.warnings, []string{llm.WarningToolCallCut}) {
		t.Errorf("warnings %q, want one %s", o.warnings, llm.WarningToolCallCut)
	}
	// 0071 new names: end
}

// A Responses answer cut while it wrote text ends max_tokens with the text
// (0071-MADR P10; 0065-MADR R22).
func TestResponsesCutTextStops_0071(t *testing.T) {
	o := cutStream(t, "openai", "gpt-6-luna", responsesCutText(t, "half an answ"))
	if o.err != nil || o.text != "half an answ" || o.stop() != llm.StopMaxTokens {
		t.Fatalf("text %q, stop %q, err %v; want the text, then max_tokens", o.text, o.stop(), o.err)
	}
}

// A cut with only reasoning is still the raw llm.ErrIncomplete, not a stop
// and not an empty reply (0071-MADR P11; 0065-MADR R23).
func TestReasoningOnlyCutUnchanged_0071(t *testing.T) {
	o := cutStream(t, "kilo", "deepseek/deepseek-v4.1-flash", chatCut(t, cutReply{reasoning: "thinking about it"}))
	if !errors.Is(o.err, llm.ErrIncomplete) || errors.Is(o.err, llm.ErrEmptyReply) || len(o.dones) != 0 {
		t.Fatalf("err %v, dones %v; want llm.ErrIncomplete, not an empty reply, and no Done", o.err, o.dones)
	}
}

// Kilo's window form is an overflow through the adapter, beneath an invalid
// request: the kind the turn reads its stated room from, or compacts on
// (0071-MADR P12; 0065-MADR R20).
func TestAgentTurn_KiloWindowFormCompacts_0065(t *testing.T) {
	body := `{"error":{"message":"This request exceeds the model's context window of 131072 tokens. ` +
		`Your request contains approximately 160002 tokens.","type":"invalid_request_error","code":400}}`
	o := wireStream(t, "kilo", "deepseek/deepseek-v4.1-flash", &cannedWire{status: http.StatusBadRequest, body: body}, wireRequest())
	if !errors.Is(o.err, llm.ErrContextOverflow) {
		t.Fatalf("err %v (%s), want llm.ErrContextOverflow", o.err, ErrorKind(o.err))
	}
}

// Text, then a call the limit cut, on Chat Completions and Messages: the
// text, the SDK's tool_call_cut warning, and max_tokens, so the turn ends
// with its text (0071-MADR P29, the owner's "End the turn, keep the text,
// notify (Recommended)").
func TestCutCallAfterText_0071(t *testing.T) {
	reply := cutReply{text: "Let me read it.", calls: []cutCall{{"c1", "read", `{"pa`}}}
	for _, c := range []struct{ name, id, model, stream string }{
		{"chat", "kilo", "deepseek/deepseek-v4.1-flash", chatCut(t, reply)},
		{"messages", "claude", "claude-haiku-5-5", messagesCut(t, reply)},
	} {
		t.Run(c.name, func(t *testing.T) {
			o := cutStream(t, c.id, c.model, c.stream)
			if o.err != nil || o.text != "Let me read it." || len(o.calls) != 0 || o.stop() != llm.StopMaxTokens {
				t.Fatalf("text %q, calls %q, stop %q, err %v; want the text, no call, max_tokens", o.text, o.calls, o.stop(), o.err)
			}
			// 0071 new names: begin
			if !slices.Equal(o.warnings, []string{llm.WarningToolCallCut}) {
				t.Errorf("warnings %q, want one %s", o.warnings, llm.WarningToolCallCut)
			}
			// 0071 new names: end
		})
	}
}

// A reply that is only a call the limit cut has nothing usable: the raw
// llm.ErrIncomplete, as before the SDK's tag (0071-MADR P29; 0065-MADR R25).
func TestOnlyCutCall_0071(t *testing.T) {
	o := cutStream(t, "kilo", "deepseek/deepseek-v4.1-flash", chatCut(t, cutReply{calls: []cutCall{{"c1", "read", `{"pa`}}}))
	if !errors.Is(o.err, llm.ErrIncomplete) || errors.Is(o.err, llm.ErrEmptyReply) || len(o.dones) != 0 || len(o.calls) != 0 {
		t.Fatalf("calls %q, dones %v, err %v; want no call, no Done, llm.ErrIncomplete", o.calls, o.dones, o.err)
	}
}
