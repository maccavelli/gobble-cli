package provider_test

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"maps"
	"strings"
	"testing"
)

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
