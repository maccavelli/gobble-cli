package provider_test

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	sdktest "github.com/maccavelli/go-llmprovider-sdk/llmprovider/llmtest"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/tool"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// wantStrict is each built-in tool's strict flag (0012-PLAN P7): "false"
// where a call may leave an argument out, "unset" where it may not.
var wantStrict = map[string]string{
	"read": "false", "write": "unset", "edit": "unset", "grep": "false", "find": "false",
	"tree": "false", "move": "false", "delete": "false", "copy": "false", "mkdir": "unset",
	"todo": "false", "bash": "false", "powershell": "false",
}

// flag names a Strict value: "unset", "false" or "true".
func flag(p *bool) string {
	if p == nil {
		return "unset"
	}
	return strconv.FormatBool(*p)
}

// builtinSpecs is the JSON of builtin.Tools()' specs, as the agent sends
// them; it fails t when they do not encode.
func builtinSpecs(t *testing.T) jsontext.Value {
	t.Helper()
	var specs []tool.Spec
	for _, tl := range builtin.Tools() {
		specs = append(specs, tl.Spec())
	}
	b, err := json.Marshal(specs)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// checkStrict compares the strict values a Responses reply reports for the
// tools named in sent with wantStrict: false where false was sent, and
// true where none was, since OpenAI then makes the tool strict (Probe D).
// It returns one line per mismatch, and per tool missing or unexpected.
func checkStrict(got map[string]bool, sent []string) []string {
	var lines []string
	for _, name := range sent {
		want := wantStrict[name] != "false"
		v, ok := got[name]
		switch {
		case !ok:
			lines = append(lines, name+": not in the reply")
		case v != want:
			lines = append(lines, fmt.Sprintf("%s: strict=%v, want %v", name, v, want))
		}
	}
	for _, name := range slices.Sorted(maps.Keys(got)) {
		if !slices.Contains(sent, name) {
			lines = append(lines, name+": in the reply but not sent")
		}
	}
	return lines
}

// Each built-in tool reaches the SDK with its wantStrict flag (0012-PLAN
// P7).
func TestStrictBuiltins(t *testing.T) {
	f := sdktest.NewFake("fake", llmprovider.Capabilities{Tools: llmprovider.Supported}).ReplyText("ok")
	req := &llm.Request{
		Messages: []llm.Message{{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: "ok?"}}}},
		Tools:    builtinSpecs(t),
	}
	for _, err := range provider.Wrap(f).Stream(t.Context(), req) {
		if err != nil {
			t.Fatal(err)
		}
	}
	tools := f.Requests()[0].Tools
	if n := len(builtin.Tools()); len(tools) != n {
		t.Fatalf("%d tools reached the SDK, want %d", len(tools), n)
	}
	for _, tl := range tools {
		want, ok := wantStrict[tl.Name]
		if !ok {
			t.Errorf("%s: no expectation in wantStrict", tl.Name)
			continue
		}
		if got := flag(tl.Strict); got != want {
			t.Errorf("%s: strict %s, want %s", tl.Name, got, want)
		}
	}
}

// checkStrict accepts a reply that matches, and names a mismatched tool and
// a missing one.
func TestCheckStrict(t *testing.T) {
	sent := []string{"read", "write"}
	for _, c := range []struct {
		name string
		got  map[string]bool
		want []string
	}{
		{"match", map[string]bool{"read": false, "write": true}, nil},
		{"mismatch", map[string]bool{"read": false, "write": false}, []string{"write: strict=false, want true"}},
		{"missing", map[string]bool{"write": true}, []string{"read: not in the reply"}},
	} {
		if got := checkStrict(c.got, sent); !slices.Equal(got, c.want) {
			t.Errorf("%s: checkStrict = %q, want %q", c.name, got, c.want)
		}
	}
}

// replyTool is a tool as a Responses reply reports it.
type replyTool struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
}

// strictOf reads a Responses reply, an event stream or one response
// object, and returns the strict value it reports for each tool
// (0012-MADR D3).
func strictOf(body []byte) (map[string]bool, error) {
	byName := func(tools []replyTool) map[string]bool {
		m := make(map[string]bool, len(tools))
		for _, t := range tools {
			m[t.Name] = t.Strict
		}
		return m
	}
	for line := range strings.SplitSeq(string(body), "\n") {
		data, ok := strings.CutPrefix(strings.TrimRight(line, "\r"), "data: ")
		if !ok {
			continue
		}
		var ev struct {
			Type     string `json:"type"`
			Response struct {
				Tools []replyTool `json:"tools"`
			} `json:"response"`
		}
		if json.Unmarshal([]byte(data), &ev) == nil && ev.Type == "response.created" && len(ev.Response.Tools) > 0 {
			return byName(ev.Response.Tools), nil
		}
	}
	var obj struct {
		Tools []replyTool `json:"tools"`
	}
	if json.Unmarshal(bytes.TrimSpace(body), &obj) == nil && len(obj.Tools) > 0 {
		return byName(obj.Tools), nil
	}
	return nil, errors.New("no tools in the reply")
}

// strictOf reads response.created's tools from a stream, and the tools of
// a whole response object, and says when a reply names none.
func TestStrictOf(t *testing.T) {
	want := map[string]bool{"record": true, "other": false}
	stream := `event: response.created
data: {"type":"response.created","response":{"tools":[{"type":"function","name":"record","strict":true},{"type":"function","name":"other","strict":false}]}}

event: response.completed
data: {"type":"response.completed","response":{"tools":[{"type":"function","name":"record","strict":false},{"type":"function","name":"other","strict":true}]}}
`
	object := `{"id":"resp_1","object":"response","model":"m","tools":[{"type":"function","name":"record","strict":true},{"type":"function","name":"other","strict":false}]}`
	for name, body := range map[string]string{"stream": stream, "object": object} {
		got, err := strictOf([]byte(body))
		if err != nil || !maps.Equal(got, want) {
			t.Errorf("%s: strictOf = %v, %v; want %v", name, got, err, want)
		}
	}
	if _, err := strictOf([]byte(`data: {"type":"response.completed","response":{}}`)); err == nil {
		t.Error("a reply naming no tools gave no error")
	}
}
