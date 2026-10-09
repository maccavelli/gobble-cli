package builtin

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/tool"
)

// propertyName is 0012-MADR D2 rule 1: camelCase ASCII.
var propertyName = regexp.MustCompile(`^[a-z][a-zA-Z0-9]*$`)

// unportable are the keywords D2 rule 7 keeps out of every schema.
var unportable = []string{"$ref", "oneOf", "anyOf", "allOf", "not", "if", "then", "else", "patternProperties", "format"}

// Every built-in tool's published schema follows 0012-MADR D2, and bash's
// timeout maximum comes from the options.
func TestSchemaConvention(t *testing.T) {
	for _, o := range []Options{{}, {MaxTimeout: 20 * time.Minute}} {
		for _, tl := range ToolsWith(o) {
			s := tl.Spec()
			var root map[string]any
			if err := json.Unmarshal(s.InputSchema, &root); err != nil {
				t.Fatalf("%s: the schema is not JSON: %v", s.Name, err)
			}
			if root["type"] != "object" {
				t.Errorf("%s: the top-level type is %v, want object", s.Name, root["type"])
			}
			for _, problem := range convention(root, "$") {
				t.Errorf("%s: %s", s.Name, problem)
			}
			if s.Name == "bash" || s.Name == "powershell" {
				want := float64(o.withDefaults().MaxTimeout / time.Second)
				if got := decodedProperty(root, "timeout")["maximum"]; got != want {
					t.Errorf("%s: timeout maximum %v, want %v", s.Name, got, want)
				}
			}
		}
	}
}

// decodedProperty is a decoded schema's property name, or an empty map.
func decodedProperty(root map[string]any, name string) map[string]any {
	props, _ := root["properties"].(map[string]any)
	p, _ := props[name].(map[string]any)
	return p
}

// convention lists where node, at path, breaks D2's rules 1–7.
func convention(node map[string]any, path string) []string {
	var out []string
	for _, k := range unportable {
		if _, ok := node[k]; ok {
			out = append(out, fmt.Sprintf("%s uses %s", path, k))
		}
	}
	typ, ok := node["type"].(string)
	if !ok {
		out = append(out, fmt.Sprintf("%s: the type %v is not one type", path, node["type"]))
	}
	switch typ {
	case "object":
		out = append(out, objectConvention(node, path)...)
	case "array":
		if _, ok := node["minItems"]; !ok {
			out = append(out, path+": an array with no minItems")
		}
		if items, ok := node["items"].(map[string]any); ok {
			out = append(out, convention(items, path+"[]")...)
		}
	}
	return out
}

// objectConvention is convention for an object's own rules and its
// properties.
func objectConvention(node map[string]any, path string) []string {
	var out []string
	if node["additionalProperties"] != false {
		out = append(out, path+": additionalProperties is not false")
	}
	props, _ := node["properties"].(map[string]any)
	required := map[string]bool{}
	list, _ := node["required"].([]any)
	for _, r := range list {
		name, _ := r.(string)
		if _, ok := props[name]; !ok {
			out = append(out, fmt.Sprintf("%s: required %q is not a property", path, name))
		}
		required[name] = true
	}
	for name, p := range props {
		at := path + "." + name
		pm, _ := p.(map[string]any)
		if !propertyName.MatchString(name) {
			out = append(out, at+": the name is not camelCase ASCII")
		}
		if d, _ := pm["description"].(string); strings.TrimSpace(d) == "" {
			out = append(out, at+": no description")
		}
		if pm["type"] == "integer" {
			if _, ok := pm["minimum"]; !ok {
				out = append(out, at+": an integer with no minimum")
			}
			if _, ok := pm["default"]; !ok && !required[name] {
				out = append(out, at+": an optional integer with no default")
			}
		}
		out = append(out, convention(pm, at)...)
	}
	return out
}

// A call just past each published bound is served within it, or refused in
// the tool's own words (0012-MADR D2 rule 8).
func TestBoundsHold(t *testing.T) {
	dir := t.TempDir()
	var b strings.Builder
	for i := 1; i <= 3000; i++ {
		fmt.Fprintf(&b, "line %d\n", i)
	}
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "e.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	env := tool.Env{Cwd: dir, Roots: []string{dir}}
	run := func(t *testing.T, tl tool.Tool, args string) tool.Result {
		t.Helper()
		r, err := tl.Run(t.Context(), tool.Call{ID: "c1", Name: tl.Spec().Name, Args: jsontext.Value(args)}, env)
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	served := []struct{ name, args, want string }{
		{"read limit above the maximum", `{"path":"big.txt","limit":5000}`, "1-2000 of 3000"},
		{"read limit 0 is the default", `{"path":"big.txt","limit":0}`, "1-2000 of 3000"},
		{"read offset 0 is line 1", `{"path":"big.txt","offset":0,"limit":1}`, "1: line 1"},
	}
	for _, c := range served {
		t.Run(c.name, func(t *testing.T) {
			if r := run(t, Read(), c.args); r.IsError || !strings.Contains(r.Text(), c.want) {
				t.Fatalf("read %s = %+v; want it served, with %q", c.args, r, c.want)
			}
		})
	}
	refused := []struct {
		name string
		tl   tool.Tool
		args string
		want string
	}{
		{"edit with no edits", Edit(), `{"path":"e.txt","edits":[]}`, "edit: edits needs at least one replacement"},
		{"edit with an empty oldText", Edit(), `{"path":"e.txt","edits":[{"oldText":"","newText":"b"}]}`, "oldText is empty"},
		{"bash with an empty command", Bash(), `{"command":"  "}`, "bash: the command is empty; send the command to run"},
	}
	for _, c := range refused {
		t.Run(c.name, func(t *testing.T) {
			if r := run(t, c.tl, c.args); !r.IsError || !strings.Contains(r.Text(), c.want) {
				t.Fatalf("%s %s = %+v; want a refusal saying %q", c.tl.Spec().Name, c.args, r, c.want)
			}
		})
	}
	t.Run("timeout above the maximum, zero and negative", func(t *testing.T) {
		for _, c := range []struct {
			seconds int
			want    time.Duration
		}{{100000, 10 * time.Minute}, {0, defaultTimeout}, {-5, defaultTimeout}} {
			if got := commandTimeout(c.seconds, 10*time.Minute); got != c.want {
				t.Errorf("commandTimeout(%d) = %v, want %v", c.seconds, got, c.want)
			}
		}
	})
	// F1c-1's tools: each bound past its maximum is served at it.
	var hits strings.Builder
	for i := 1; i <= maxGrepLimit+1; i++ {
		fmt.Fprintf(&hits, "hit %d\n", i)
	}
	if err := os.WriteFile(filepath.Join(dir, "hits.txt"), []byte(hits.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Run("grep limit and context above their maximums", func(t *testing.T) {
		r := run(t, Grep(), `{"pattern":"hit","glob":"hits.txt","limit":5000}`)
		if r.IsError || !strings.Contains(r.Text(), "[1000 matches shown, the most there can be;") {
			t.Fatalf("grep limit 5000 = %.200q…; want it served at 1000", r.Text())
		}
		r = run(t, Grep(), `{"pattern":"^hit 500$","glob":"hits.txt","context":50}`)
		if lines := strings.Count(r.Text(), "\n") + 1; r.IsError || lines != 2*maxGrepContext+1 {
			t.Fatalf("grep context 50 showed %d lines; want it served at %d either side", lines, maxGrepContext)
		}
	})
	t.Run("find and tree limits and depth above their maximums", func(t *testing.T) {
		for _, c := range []struct {
			tl   tool.Tool
			args string
		}{{Find(), `{"pattern":"*.txt","limit":9000}`}, {Tree(), `{"depth":50,"limit":9000}`}} {
			if r := run(t, c.tl, c.args); r.IsError {
				t.Errorf("%s %s refused: %s", c.tl.Spec().Name, c.args, r.Text())
			}
		}
		for _, c := range []struct{ v, lo, hi, def, want int }{
			{9000, 1, maxFindLimit, defaultFindLimit, maxFindLimit},
			{9000, 1, maxTreeLimit, defaultTreeLimit, maxTreeLimit},
			{50, 1, maxTreeDepth, defaultTreeDepth, maxTreeDepth},
			{0, 1, maxTreeDepth, defaultTreeDepth, defaultTreeDepth},
		} {
			if got := clampInt(c.v, c.lo, c.hi, c.def); got != c.want {
				t.Errorf("clampInt(%d, %d, %d, %d) = %d, want %d", c.v, c.lo, c.hi, c.def, got, c.want)
			}
		}
	})
}
