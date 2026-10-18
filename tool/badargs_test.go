package tool_test

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

type countIn struct {
	N int64 `json:"n"`
}

// Each of a tool's four refusals of its arguments, by prepare, as not
// JSON, by the schema, and in decoding, is marked BadArguments; the tool's
// own error is not (0005-PLAN F5-3 decision 3).
func TestBadArguments(t *testing.T) {
	tl := tool.New("count", "", func(_ context.Context, in countIn, _ tool.Env) (string, error) {
		if in.N == 13 {
			return "", errors.New("unlucky")
		}
		return "ok", nil
	}, tool.WithPrepare(func(args jsontext.Value) (jsontext.Value, error) {
		if string(args) == `{"prepare":"no"}` {
			return nil, errors.New("count input is invalid")
		}
		return args, nil
	}))
	for _, c := range []struct {
		name, args string
		bad, err   bool
	}{
		{"prepare", `{"prepare":"no"}`, true, true},
		{"not JSON", `{"n":`, true, true},
		{"schema", `{"n":"x"}`, true, true},
		{"decoding", `{"n":1e30}`, true, true}, // an integer to the schema, too large for int64
		{"the tool's own error", `{"n":13}`, false, true},
		{"a good call", `{"n":1}`, false, false},
	} {
		r, err := tl.Run(t.Context(), tool.Call{Name: "count", Args: jsontext.Value(c.args)}, tool.Env{})
		if err != nil || r.BadArguments != c.bad || r.IsError != c.err {
			t.Errorf("%s: %+v (%q), %v; want BadArguments %v, IsError %v", c.name, r, r.Text(), err, c.bad, c.err)
		}
	}
}
