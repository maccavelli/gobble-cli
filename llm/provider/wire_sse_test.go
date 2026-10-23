package provider

import "strings"

// Canned streams in each wire's event format, for a request that asks to
// stream: "hello", or a get_weather call for Paris, with 12 tokens in and 3
// out, as the canned replies above (0031-PLAN P2). Each completed event
// carries the canned reply, where the wire's last event holds the response.
var wireStreams = map[string][2]string{
	"/responses": {
		sse("response.output_text.delta", `{"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"hello"}`) +
			sse("response.output_item.done", `{"type":"response.output_item.done","output_index":0,"item":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"hello"}]}}`) +
			sse("response.completed", `{"type":"response.completed","response":`+wireResponsesText+`}`),
		sse("response.output_item.done", `{"type":"response.output_item.done","output_index":0,"item":{"type":"function_call","call_id":"c1","name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}`) +
			sse("response.completed", `{"type":"response.completed","response":`+wireResponsesTool+`}`),
	},
	"/messages": {
		sse("message_start", `{"type":"message_start","message":{"id":"m","type":"message","role":"assistant","content":[],"usage":{"input_tokens":12,"output_tokens":0}}}`) +
			sse("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`) +
			sse("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"hello"}}`) +
			sse("content_block_stop", `{"type":"content_block_stop","index":0}`) +
			sse("message_delta", `{"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":3}}`) +
			sse("message_stop", `{"type":"message_stop"}`),
		sse("message_start", `{"type":"message_start","message":{"id":"m","type":"message","role":"assistant","content":[],"usage":{"input_tokens":12,"output_tokens":0}}}`) +
			sse("content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"c1","name":"get_weather","input":{}}}`) +
			sse("content_block_delta", `{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"city\":\"Paris\"}"}}`) +
			sse("content_block_stop", `{"type":"content_block_stop","index":0}`) +
			sse("message_delta", `{"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":3}}`) +
			sse("message_stop", `{"type":"message_stop"}`),
	},
	"/interactions": {
		sse("interaction.created", `{"event_type":"interaction.created","interaction":{"id":"i","status":"in_progress"}}`) +
			sse("step.start", `{"event_type":"step.start","index":0,"step":{"type":"model_output"}}`) +
			sse("step.delta", `{"event_type":"step.delta","index":0,"delta":{"type":"text","text":"hello"}}`) +
			sse("step.stop", `{"event_type":"step.stop","index":0}`) +
			sse("interaction.completed", `{"event_type":"interaction.completed","interaction":`+wireInterText+`}`) +
			sse("done", `[DONE]`),
		sse("interaction.created", `{"event_type":"interaction.created","interaction":{"id":"i","status":"in_progress"}}`) +
			sse("step.start", `{"event_type":"step.start","index":0,"step":{"type":"function_call","id":"c1","name":"get_weather","arguments":{"city":"Paris"}}}`) +
			sse("step.stop", `{"event_type":"step.stop","index":0}`) +
			sse("interaction.completed", `{"event_type":"interaction.completed","interaction":`+wireInterTool+`}`) +
			sse("done", `[DONE]`),
	},
	"/chat/completions": {
		sse("", `{"id":"c","choices":[{"index":0,"delta":{"role":"assistant","content":"hello"},"finish_reason":null}]}`) +
			sse("", `{"id":"c","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`) +
			sse("", `{"id":"c","choices":[],"usage":{"prompt_tokens":12,"completion_tokens":3,"total_tokens":15}}`) +
			sse("", `[DONE]`),
		sse("", `{"id":"c","choices":[{"index":0,"delta":{"role":"assistant","tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"get_weather","arguments":"{\"city\":\"Paris\"}"}}]},"finish_reason":null}]}`) +
			sse("", `{"id":"c","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`) +
			sse("", `{"id":"c","choices":[],"usage":{"prompt_tokens":12,"completion_tokens":3,"total_tokens":15}}`) +
			sse("", `[DONE]`),
	},
}

// sse is one Server-Sent Event: its name, when the wire names its events, and
// its data.
func sse(name, data string) string {
	if name == "" {
		return "data: " + data + "\n\n"
	}
	return "event: " + name + "\ndata: " + data + "\n\n"
}

// wireStreamFor is the canned stream for a request to path, text or tool.
func wireStreamFor(path string, tool bool) (string, bool) {
	for suffix, s := range wireStreams {
		if strings.HasSuffix(path, suffix) {
			if tool {
				return s[1], true
			}
			return s[0], true
		}
	}
	return "", false
}
