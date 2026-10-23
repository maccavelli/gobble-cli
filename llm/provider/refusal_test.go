package provider

import (
	"net/http"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// A reply the service stops for safety, or the model refuses, stops as
// llm.StopRefusal, its text kept, on the wires that report it in their
// finish reason; an ordinary reply still stops as end_turn (0005-PLAN F5-3
// decision 5).
func TestWireRefusal(t *testing.T) {
	for _, c := range []struct {
		wire, id, model, body, stream string
		want                          llm.StopReason
	}{
		{"messages", "claude", "claude-haiku-5-5",
			`{"id":"m","type":"message","role":"assistant","stop_reason":"refusal","content":[{"type":"text","text":"I can't help with that."}],"usage":{"input_tokens":12,"output_tokens":6}}`,
			sse("message_start", `{"type":"message_start","message":{"id":"m","type":"message","role":"assistant","content":[],"usage":{"input_tokens":12,"output_tokens":0}}}`) + sse("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`) + sse("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"I can't help with that."}}`) + sse("content_block_stop", `{"type":"content_block_stop","index":0}`) + sse("message_delta", `{"type":"message_delta","delta":{"stop_reason":"refusal"},"usage":{"output_tokens":6}}`) + sse("message_stop", `{"type":"message_stop"}`),
			llm.StopRefusal},
		{"chat completions", "together", "zai-org/GLM-5.3-Flash",
			`{"id":"c","choices":[{"index":0,"message":{"role":"assistant","content":"I can't help with that."},"finish_reason":"content_filter"}],"usage":{"prompt_tokens":12,"completion_tokens":6,"total_tokens":18}}`,
			sse("", `{"id":"c","choices":[{"index":0,"delta":{"role":"assistant","content":"I can't help with that."},"finish_reason":null}]}`) + sse("", `{"id":"c","choices":[{"index":0,"delta":{},"finish_reason":"content_filter"}]}`) + sse("", `{"id":"c","choices":[],"usage":{"prompt_tokens":12,"completion_tokens":6,"total_tokens":18}}`) + sse("", `[DONE]`),
			llm.StopRefusal},
		{"messages, an ordinary reply", "claude", "claude-haiku-5-5",
			`{"id":"m","type":"message","role":"assistant","stop_reason":"end_turn","content":[{"type":"text","text":"Sunny."}],"usage":{"input_tokens":12,"output_tokens":2}}`,
			sse("message_start", `{"type":"message_start","message":{"id":"m","type":"message","role":"assistant","content":[],"usage":{"input_tokens":12,"output_tokens":0}}}`) + sse("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`) + sse("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Sunny."}}`) + sse("content_block_stop", `{"type":"content_block_stop","index":0}`) + sse("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}`) + sse("message_stop", `{"type":"message_stop"}`),
			llm.StopEndTurn},
	} {
		t.Run(c.wire, func(t *testing.T) {
			w := &cannedWire{status: http.StatusOK, body: c.body, stream: c.stream}
			o := wireStream(t, c.id, c.model, w, wireRequest())
			if o.err != nil || o.stop != c.want || o.text == "" {
				t.Fatalf("err %v, stop %q, text %q; want %q with the text", o.err, o.stop, o.text, c.want)
			}
		})
	}
}
