package provider

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/tool"
)

// The deferral tests' tools: read, direct; tool_search, the search; and two
// deferred tools, request_permissions and lsp.
var (
	deferRead   = tool.Spec{Name: "read", Description: "Read a file.", InputSchema: jsontext.Value(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`)}
	deferSearch = tool.Spec{Name: "tool_search", Description: "Find deferred tools.", InputSchema: jsontext.Value(`{"type":"object","properties":{"query":{"type":"string"}},"required":["query"]}`)}
	deferPerms  = tool.Spec{Name: "request_permissions", Description: "Ask for more access.", Exposure: tool.ExposureDeferred,
		InputSchema: jsontext.Value(`{"type":"object","properties":{"network":{"type":"boolean"}}}`)}
	deferLSP = tool.Spec{Name: "lsp", Description: "Ask a language server.", Exposure: tool.ExposureDeferred,
		InputSchema: jsontext.Value(`{"type":"object","properties":{"file":{"type":"string"}},"required":["file"]}`)}
)

func specsOf(t *testing.T, specs ...tool.Spec) jsontext.Value {
	t.Helper()
	b, err := json.Marshal(specs)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// deferRequest is a request with request_permissions and lsp deferred. With
// loaded, a tool_search call's result has loaded request_permissions, which
// Tools then sends last, as a provider without deferral needs it.
func deferRequest(t *testing.T, loaded bool) *llm.Request {
	t.Helper()
	r := wireRequest()
	r.Tools = specsOf(t, deferRead, deferSearch)
	r.Deferred = specsOf(t, deferPerms, deferLSP)
	r.Search = deferSearch.Name
	if loaded {
		r.Tools = specsOf(t, deferRead, deferSearch, deferPerms)
		r.Messages = append(r.Messages,
			llm.Message{Role: llm.RoleAssistant, Content: []llm.Content{{Type: llm.ContentToolCall, ToolCallID: "s1", ToolName: deferSearch.Name, ToolArgs: `{"query":"permissions"}`}}},
			llm.Message{Role: llm.RoleTool, Content: []llm.Content{{Type: llm.ContentToolResult, ToolCallID: "s1", ToolName: deferSearch.Name,
				Text: "Loaded request_permissions.", Loads: []string{deferPerms.Name}}}},
		)
	}
	return r
}

// toolNamesOf is the SDK request's tool names, with "+deferred" or
// "+search" after each that has the flag.
func toolNamesOf(r *llmprovider.Request) []string {
	var out []string
	for _, tl := range r.Tools {
		n := tl.Name
		if tl.Deferred {
			n += "+deferred"
		}
		if tl.Search {
			n += "+search"
		}
		out = append(out, n)
	}
	return out
}

// toRequest sends the deferred tools as deferred and the search as the
// search, then a ToolLoadItem after the result that loaded each; a name the
// request does not defer gets none; and with nothing deferred, the new
// fields change nothing (0012-MADR D16, D17).
func TestToRequestDeferral(t *testing.T) {
	t.Run("nothing deferred", func(t *testing.T) {
		with := deferRequest(t, true)
		with.Deferred = nil
		without := deferRequest(t, true)
		without.Deferred, without.Search = nil, ""
		without.Messages[len(without.Messages)-1].Content[0].Loads = nil
		got, err := toRequest(with)
		if err != nil {
			t.Fatal(err)
		}
		want, err := toRequest(without)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("with Search and Loads but nothing deferred:\n%+v\nwant today's request:\n%+v", got, want)
		}
	})
	t.Run("loaded", func(t *testing.T) {
		sreq, err := toRequest(deferRequest(t, true))
		if err != nil {
			t.Fatal(err)
		}
		if got, want := toolNamesOf(sreq), []string{"read", "tool_search+search", "request_permissions+deferred", "lsp+deferred"}; !slices.Equal(got, want) {
			t.Fatalf("tools %q, want %q", got, want)
		}
		out := slices.IndexFunc(sreq.Input, func(it llmprovider.Item) bool {
			o, ok := it.(llmprovider.FunctionCallOutputItem)
			return ok && o.CallID == "s1"
		})
		if out < 0 || out+1 >= len(sreq.Input) {
			t.Fatalf("input %+v has no item after s1's result", sreq.Input)
		}
		if load, ok := sreq.Input[out+1].(llmprovider.ToolLoadItem); !ok || load != (llmprovider.ToolLoadItem{CallID: "s1", Name: deferPerms.Name}) {
			t.Fatalf("after s1's result: %#v, want its ToolLoadItem", sreq.Input[out+1])
		}
		if err := (llmprovider.Capabilities{Tools: llmprovider.Supported}).Check(sreq); err != nil {
			t.Fatalf("the SDK refuses the request: %v", err)
		}
	})
	t.Run("a load the request does not defer", func(t *testing.T) {
		r := deferRequest(t, true)
		res := &r.Messages[len(r.Messages)-1].Content[0]
		res.Loads = append(res.Loads, deferRead.Name)
		sreq, err := toRequest(r)
		if err != nil {
			t.Fatal(err)
		}
		var loads []string
		for _, it := range sreq.Input {
			if load, ok := it.(llmprovider.ToolLoadItem); ok {
				loads = append(loads, load.Name)
			}
		}
		if !slices.Equal(loads, []string{deferPerms.Name}) {
			t.Fatalf("loads %q, want only request_permissions", loads)
		}
		if err := (llmprovider.Capabilities{Tools: llmprovider.Supported}).Check(sreq); err != nil {
			t.Fatalf("the SDK refuses the request: %v", err)
		}
	})
}

// bodyTools is the tools array of a request body, as sent.
func bodyTools(t *testing.T, body string) string {
	t.Helper()
	var top map[string]jsontext.Value
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	return string(top["tools"])
}

// Each provider sends a load in the form F21 gives its wire and model:
// Claude 4.5 on (claude, and OpenCode Zen's messages route) and GPT-5.4 on
// (openai, and either gateway's responses route) natively, Grok through
// use_tool, every other wire appended; OpenCode Go sends claude- over Chat
// Completions. On the native wires and Grok the tools sent are the same
// before and after the load (0012-MADR D16, F21; 0012-PLAN deviation 2).
func TestWireDeferral(t *testing.T) {
	form := map[string]string{
		"claude": "messages", "opencode-zen": "messages", "opencode-go": "append",
		"openai": "responses", "grok": "dispatch",
		"gemini": "append", "huggingface": "append", "kilo": "append", "together": "append", "ollama": "append",
	}
	type wireCase struct{ id, model, form string }
	var cases []wireCase
	for _, w := range wireIDs {
		cases = append(cases, wireCase{w.id, w.model, form[w.id]})
	}
	cases = append(cases, wireCase{"opencode-go", "gpt-6-luna", "responses"})
	for _, w := range cases {
		t.Run(w.id+"/"+w.model, func(t *testing.T) {
			var bodies [2]string
			for i, loaded := range []bool{false, true} {
				c := &cannedWire{}
				if o := wireStream(t, w.id, w.model, c, deferRequest(t, loaded)); o.err != nil {
					t.Fatalf("loaded %v: %v", loaded, o.err)
				}
				oneRequest(t, c)
				_, b := c.requests()
				bodies[i] = b[0]
			}
			before, after := bodyTools(t, bodies[0]), bodyTools(t, bodies[1])
			switch f := w.form; f {
			case "messages", "responses", "dispatch":
				if before != after {
					t.Fatalf("%s: the tools changed at the load:\nbefore %s\nafter  %s", f, before, after)
				}
				switch f {
				case "messages":
					if !strings.Contains(before, `"defer_loading":true`) || !strings.Contains(bodies[1], `"tool_reference"`) {
						t.Fatalf("messages: no defer_loading in %s, or no tool_reference in %s", before, bodies[1])
					}
				case "responses":
					if !strings.Contains(bodies[1], `"tool_search_output"`) || !strings.Contains(bodies[1], deferPerms.Name) {
						t.Fatalf("responses: no tool_search_output loading request_permissions in %s", bodies[1])
					}
				case "dispatch":
					if !strings.Contains(before, `"use_tool"`) || strings.Contains(before, deferPerms.Name) {
						t.Fatalf("dispatch: tools %s, want use_tool and no request_permissions", before)
					}
				}
			case "append":
				if strings.Contains(before, deferPerms.Name) || strings.LastIndex(after, deferPerms.Name) < strings.LastIndex(after, deferSearch.Name) {
					t.Fatalf("append: before %s, after %s; want request_permissions only after the load, and last", before, after)
				}
			default:
				t.Fatalf("no form for %s", w.id)
			}
		})
	}
}

// A use_tool call on Grok naming a deferred tool reaches gobble as that
// tool's call (0012-MADR F21; the SDK's 0040-MADR D5).
func TestWireDeferralGrokCall(t *testing.T) {
	call := `{"type":"function_call","call_id":"c2","name":"use_tool","arguments":"{\"tool_name\":\"request_permissions\",\"tool_input\":{\"network\":true}}"}`
	reply := `{"id":"r","status":"completed","output":[` + call + `],"usage":{"input_tokens":12,"output_tokens":3,"total_tokens":15}}`
	c := &cannedWire{status: 200, body: reply,
		stream: sse("response.output_item.done", `{"type":"response.output_item.done","output_index":0,"item":`+call+`}`) +
			sse("response.completed", `{"type":"response.completed","response":`+reply+`}`)}
	var model string
	for _, w := range wireIDs {
		if w.id == "grok" {
			model = w.model
		}
	}
	o := wireStream(t, "grok", model, c, deferRequest(t, true))
	if o.err != nil {
		t.Fatal(o.err)
	}
	if len(o.calls) != 1 || o.calls[0] != `request_permissions {"network":true}` {
		t.Fatalf("calls %q, want request_permissions with its input", o.calls)
	}
}
