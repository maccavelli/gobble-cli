package tool_test

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"

	"github.com/maccavelli/gobble-cli/tool"
)

type readIn struct {
	Path  string `json:"path" jsonschema:"the file to read"`
	Limit int    `json:"limit,omitzero"`
}

func readTool(seen *readIn) tool.Tool {
	return tool.New("read", "read a file", func(_ context.Context, in readIn, env tool.Env) (string, error) {
		*seen = in
		if in.Path == "missing" {
			return "", errors.New("no such file")
		}
		return "contents of " + env.Cwd + "/" + in.Path, nil
	}, tool.WithKind(tool.KindRead), tool.WithDescribe(func(c tool.Call, _ tool.Env) string { return "read " + string(c.Args) }))
}

// The schema comes from the input type: path is required with its
// description, limit is optional, and nothing else is allowed.
func TestNewDerivesSchema(t *testing.T) {
	var seen readIn
	spec := readTool(&seen).Spec()
	s := string(spec.InputSchema)
	for _, want := range []string{`"type":"object"`, `"required":["path"]`, `"description":"the file to read"`, `"additionalProperties":false`} {
		if !strings.Contains(s, want) {
			t.Errorf("schema %s lacks %s", s, want)
		}
	}
	if spec.Name != "read" || spec.Kind != tool.KindRead {
		t.Errorf("spec = %+v", spec)
	}
}

type listIn struct {
	Items []string `json:"items" jsonschema:"the items"`
	Limit int      `json:"limit,omitzero" jsonschema:"how many"`
}

// The published schema has no null types and carries WithSchema's bounds,
// while Run validates against the Go type's schema alone, so the tool's
// code sees the arguments it sees today (0012-MADR D2 rules 5 and 8).
func TestPublishedSchema(t *testing.T) {
	var seen listIn
	tl := tool.New("list", "list", func(_ context.Context, in listIn, _ tool.Env) (string, error) {
		seen = in
		return "ok", nil
	}, tool.WithSchema(func(s *jsonschema.Schema) {
		s.Properties["items"].MinItems = new(1)
		s.Properties["limit"].Maximum = new(float64(10))
	}))
	s := string(tl.Spec().InputSchema)
	for _, want := range []string{`"items":{"type":"array"`, `"minItems":1`, `"maximum":10`} {
		if !strings.Contains(s, want) {
			t.Errorf("schema %s lacks %s", s, want)
		}
	}
	if strings.Contains(s, `"null"`) {
		t.Errorf("schema %s offers null", s)
	}
	r, err := tl.Run(t.Context(), tool.Call{Name: "list", Args: jsontext.Value(`{"items":[],"limit":50}`)}, tool.Env{})
	if err != nil || r.IsError || seen.Limit != 50 {
		t.Fatalf("run = %+v, %v, seen %+v; want the call to reach the tool, whose code enforces the published bounds", r, err, seen)
	}
}

func TestRun(t *testing.T) {
	var seen readIn
	tl := readTool(&seen)
	env := tool.Env{Cwd: "/w"}
	cases := []struct {
		name, args, want string
		isError          bool
	}{
		{"valid", `{"path":"a.go","limit":3}`, "contents of /w/a.go", false},
		{"fn error is the model's to read", `{"path":"missing"}`, "no such file", true},
		{"missing required field", `{}`, "invalid arguments", true},
		{"unknown field", `{"path":"a","extra":1}`, "invalid arguments", true},
		{"wrong type", `{"path":3}`, "invalid arguments", true},
		{"not JSON", `{"path":`, "not JSON", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := tl.Run(t.Context(), tool.Call{ID: "c1", Name: "read", Args: jsontext.Value(tc.args)}, env)
			if err != nil {
				t.Fatal(err)
			}
			if r.IsError != tc.isError || !strings.Contains(r.Text(), tc.want) {
				t.Fatalf("result = %+v (text %q); want isError %v and %q", r, r.Text(), tc.isError, tc.want)
			}
		})
	}
	if seen.Limit != 0 && seen.Path != "missing" {
		t.Fatalf("decoded %+v", seen)
	}
}

// A done context is an error, not a result.
func TestRunCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	tl := tool.New("wait", "wait", func(ctx context.Context, _ struct{}, _ tool.Env) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	if _, err := tl.Run(ctx, tool.Call{Name: "wait"}, tool.Env{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestDescribe(t *testing.T) {
	var seen readIn
	d, ok := readTool(&seen).(tool.Describer)
	if !ok || d.Describe(tool.Call{Args: jsontext.Value(`{"path":"x"}`)}, tool.Env{}) != `read {"path":"x"}` {
		t.Fatal("WithDescribe's function is not used")
	}
	plain := tool.New("noop", "", func(context.Context, struct{}, tool.Env) (string, error) { return "", nil })
	if got := plain.(tool.Describer).Describe(tool.Call{}, tool.Env{}); got != "noop" {
		t.Fatalf("default title %q, want the name", got)
	}
}

// Outside is WithOutside's function, and no paths without it.
func TestOutside(t *testing.T) {
	confined := tool.New("read", "", func(context.Context, struct{}, tool.Env) (string, error) { return "", nil },
		tool.WithOutside(func(_ tool.Call, env tool.Env) []string { return env.Roots }))
	got := confined.(tool.Confined).Outside(tool.Call{}, tool.Env{Roots: []string{"/elsewhere"}})
	if len(got) != 1 || got[0] != "/elsewhere" {
		t.Fatalf("Outside = %q, want WithOutside's answer", got)
	}
	plain := tool.New("noop", "", func(context.Context, struct{}, tool.Env) (string, error) { return "", nil })
	if got := plain.(tool.Confined).Outside(tool.Call{}, tool.Env{Roots: []string{"/x"}}); got != nil {
		t.Fatalf("default Outside = %q, want none", got)
	}
}

// WithPrepare rewrites the arguments before validation, and its error is
// the model's to read, word for word.
func TestPrepare(t *testing.T) {
	var seen readIn
	tl := tool.New("read", "", func(_ context.Context, in readIn, _ tool.Env) (string, error) {
		seen = in
		return "ok", nil
	}, tool.WithPrepare(func(args jsontext.Value) (jsontext.Value, error) {
		if string(args) == `{"file":"a"}` {
			return jsontext.Value(`{"path":"a"}`), nil
		}
		if string(args) == `{}` {
			return nil, errors.New("read input is invalid")
		}
		return args, nil
	}))
	r, err := tl.Run(t.Context(), tool.Call{Args: jsontext.Value(`{"file":"a"}`)}, tool.Env{})
	if err != nil || r.IsError || seen.Path != "a" {
		t.Fatalf("result = %+v, %v, decoded %+v; want the rewritten arguments", r, err, seen)
	}
	r, err = tl.Run(t.Context(), tool.Call{}, tool.Env{})
	if err != nil || !r.IsError || r.Text() != "read input is invalid" {
		t.Fatalf("result = %+v (text %q), %v; want the prepare error verbatim", r, r.Text(), err)
	}
}

// A Result output is used as it is, and a struct is its JSON.
func TestOutputs(t *testing.T) {
	res := tool.New("r", "", func(context.Context, struct{}, tool.Env) (tool.Result, error) {
		return tool.Result{Output: jsontext.Value(`"x"`), Title: "t", Summary: "s"}, nil
	})
	r, err := res.Run(t.Context(), tool.Call{}, tool.Env{})
	if err != nil || r.Title != "t" || r.Summary != "s" || r.Text() != "x" {
		t.Fatalf("result = %+v, %v", r, err)
	}
	type out struct {
		N int `json:"n"`
	}
	st := tool.New("s", "", func(context.Context, struct{}, tool.Env) (out, error) { return out{N: 2}, nil })
	r, err = st.Run(t.Context(), tool.Call{}, tool.Env{})
	if err != nil || r.Text() != `{"n":2}` {
		t.Fatalf("result = %+v, %v", r, err)
	}
}

// A result's plan reaches the caller as the tool left it: a list in order,
// and an empty list, which clears, still set.
func TestPlan(t *testing.T) {
	for _, plan := range [][]tool.PlanItem{
		{{Content: "a", Status: tool.PlanCompleted, Priority: tool.PriorityHigh}, {Content: "b", Status: tool.PlanPending, Priority: tool.PriorityMedium}},
		{},
	} {
		tl := tool.New("p", "", func(context.Context, struct{}, tool.Env) (tool.Result, error) {
			r := tool.TextResult("ok")
			r.Plan = plan
			return r, nil
		})
		r, err := tl.Run(t.Context(), tool.Call{}, tool.Env{})
		if err != nil || r.Plan == nil || !slices.Equal(r.Plan, plan) {
			t.Fatalf("plan %v = %+v, %v", plan, r.Plan, err)
		}
	}
	b, err := json.Marshal(tool.Result{Plan: []tool.PlanItem{{Content: "a", Status: tool.PlanInProgress, Priority: tool.PriorityLow}}})
	if err != nil || string(b) != `{"plan":[{"content":"a","status":"in_progress","priority":"low"}]}` {
		t.Fatalf("encoded %s, %v", b, err)
	}
}

// WithLocations names a call's locations through Locator; a tool without
// it names none; a result's locations are encoded with their lines, a
// line of 0 left out (0005-PLAN F1d-3).
func TestLocations(t *testing.T) {
	tl := tool.New("l", "", func(context.Context, struct{}, tool.Env) (string, error) { return "", nil },
		tool.WithLocations(func(c tool.Call, env tool.Env) []tool.Location {
			return []tool.Location{{Path: env.Cwd + "/" + c.ID, Line: 3}}
		}))
	if got := tl.(tool.Locator).Locations(tool.Call{ID: "x"}, tool.Env{Cwd: "/w"}); len(got) != 1 || got[0] != (tool.Location{Path: "/w/x", Line: 3}) {
		t.Fatalf("locations %+v", got)
	}
	if got := readTool(new(readIn)).(tool.Locator).Locations(tool.Call{}, tool.Env{}); got != nil {
		t.Fatalf("a tool without WithLocations names %+v", got)
	}
	b, err := json.Marshal(tool.Result{Locations: []tool.Location{{Path: "/a", Line: 2}, {Path: "/b"}}})
	if err != nil || string(b) != `{"locations":[{"path":"/a","line":2},{"path":"/b"}]}` {
		t.Fatalf("encoded %s, %v", b, err)
	}
}

// A tool's exposure and family reach its spec, and a result's loads and an
// environment's deferred tools are encoded by those names (0005-PLAN F1d-2).
func TestExposureAndFamily(t *testing.T) {
	tl := tool.New("f", "", func(context.Context, struct{}, tool.Env) (string, error) { return "", nil },
		tool.WithExposure(tool.ExposureDeferred), tool.WithFamily("files"))
	if s := tl.Spec(); s.Exposure != tool.ExposureDeferred || s.Family != "files" {
		t.Fatalf("spec exposure %q, family %q", s.Exposure, s.Family)
	}
	if s := readTool(new(readIn)).Spec(); s.Exposure != "" || s.Family != "" {
		t.Fatalf("a tool without the options: exposure %q, family %q; want both empty, which is direct", s.Exposure, s.Family)
	}
	b, err := json.Marshal(tool.Result{Load: []string{"a", "b"}})
	if err != nil || string(b) != `{"load":["a","b"]}` {
		t.Fatalf("result encoded %s, %v", b, err)
	}
	b, err = json.Marshal(tool.Env{Deferred: []tool.Spec{{Name: "a", Exposure: tool.ExposureDeferred, Family: "x"}}})
	if err != nil || string(b) != `{"deferred":[{"name":"a","exposure":"deferred","family":"x"}]}` {
		t.Fatalf("env encoded %s, %v", b, err)
	}
}
