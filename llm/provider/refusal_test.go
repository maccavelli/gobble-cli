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
		wire, id, model, body string
		want                  llm.StopReason
	}{
		{"messages", "claude", "claude-haiku-5-5",
			`{"id":"m","type":"message","role":"assistant","stop_reason":"refusal","content":[{"type":"text","text":"I can't help with that."}],"usage":{"input_tokens":12,"output_tokens":6}}`,
			llm.StopRefusal},
		{"chat completions", "together", "zai-org/GLM-5.3-Flash",
			`{"id":"c","choices":[{"index":0,"message":{"role":"assistant","content":"I can't help with that."},"finish_reason":"content_filter"}],"usage":{"prompt_tokens":12,"completion_tokens":6,"total_tokens":18}}`,
			llm.StopRefusal},
		{"messages, an ordinary reply", "claude", "claude-haiku-5-5",
			`{"id":"m","type":"message","role":"assistant","stop_reason":"end_turn","content":[{"type":"text","text":"Sunny."}],"usage":{"input_tokens":12,"output_tokens":2}}`,
			llm.StopEndTurn},
	} {
		t.Run(c.wire, func(t *testing.T) {
			w := &cannedWire{status: http.StatusOK, body: c.body}
			o := wireStream(t, c.id, c.model, w, wireRequest())
			if o.err != nil || o.stop != c.want || o.text == "" {
				t.Fatalf("err %v, stop %q, text %q; want %q with the text", o.err, o.stop, o.text, c.want)
			}
		})
	}
}
