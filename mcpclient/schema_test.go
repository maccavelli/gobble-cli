package mcpclient

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
)

// These tests hold 0060-MADR D2, D3 and its Confirmation 1: the rule's
// output, the shapes it leaves alone and flags, and that a rewritten schema
// accepts exactly what the original accepts, checked with the validator
// gobble's module graph already holds over a generated corpus.

// decode is s as the go-sdk's client hands a tool's schema over: a
// map[string]any of maps, lists, strings, float64s, bools and nils.
func decode(t testing.TB, s string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := jsonx.Unmarshal([]byte(s), &m); err != nil {
		t.Fatalf("not a JSON object: %v\n%s", err, s)
	}
	return m
}

// readData is a file of testdata, decoded.
func readData(t testing.TB, name string) map[string]any {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return decode(t, string(b))
}

// props7 is execute_pipeline's seven properties without their descriptions.
const props7 = `{"session_id":{"type":"string"},"target":{"type":"string"},"intent":{"type":"string"},"plan_hash":{"type":"string"},` +
	`"dry_run":{"type":"boolean"},"target_roles":{"type":"array","items":{"type":"string"}},"reject":{"type":"boolean"}}`

// props4 is the four of them the oneOf shape (S2) keeps.
const props4 = `{"session_id":{"type":"string"},"target":{"type":"string"},"intent":{"type":"string"},"dry_run":{"type":"boolean"}}`

// ruleCase is one schema and what the rule does with it. A case with want
// is rewritten into want. A case with none is returned as given; flag is
// then a part of the reason it is flagged with, or "" when it is not
// flagged. The ids are the MADR's table: S0 is the real schema, S1 the
// same without its descriptions, S2 to S12 the shapes.
type ruleCase struct {
	id, name string
	in, want string
	flag     string
	// named marks a case with a property called like a scope keyword: its
	// rewrite carries the name, which is not the keyword.
	named bool
}

func ruleCases() []ruleCase {
	p7 := props7
	return []ruleCase{
		{id: "S1", name: "the real schema without descriptions",
			in: fmt.Sprintf(`{"type":"object","properties":%s,"additionalProperties":false,"anyOf":[{"required":["target","intent"]},{"required":["session_id"]}]}`, p7),
			want: fmt.Sprintf(`{"type":"object","properties":%[1]s,"additionalProperties":false,"anyOf":[`+
				`{"type":"object","properties":%[1]s,"additionalProperties":false,"required":["target","intent"]},`+
				`{"type":"object","properties":%[1]s,"additionalProperties":false,"required":["session_id"]}]}`, p7)},
		{id: "S2", name: "oneOf in place of anyOf",
			in: fmt.Sprintf(`{"type":"object","properties":%s,"additionalProperties":false,"oneOf":[{"required":["target","intent"]},{"required":["session_id"]}]}`, props4),
			want: fmt.Sprintf(`{"type":"object","properties":%[1]s,"additionalProperties":false,"oneOf":[`+
				`{"type":"object","properties":%[1]s,"additionalProperties":false,"required":["target","intent"]},`+
				`{"type":"object","properties":%[1]s,"additionalProperties":false,"required":["session_id"]}]}`, props4)},
		{id: "S3", name: "no root type",
			in:   `{"properties":{"session_id":{"type":"string"},"target":{"type":"string"},"intent":{"type":"string"}},"anyOf":[{"required":["target","intent"]},{"required":["session_id"]}]}`,
			flag: `the root type is not exactly "object"`},
		{id: "S4", name: "a branch typed string is removed",
			in: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer"}},"anyOf":[{"required":["a"]},{"type":"string"}]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer"}},"anyOf":[` +
				`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer"}},"required":["a"]}]}`},
		{id: "S5", name: "a nested anyOf in a branch is not touched",
			in: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"number"}},"additionalProperties":false,` +
				`"anyOf":[{"anyOf":[{"required":["a"]},{"required":["b"]}]},{"required":["c"]}]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"number"}},"additionalProperties":false,"anyOf":[` +
				`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"number"}},"additionalProperties":false,"anyOf":[{"required":["a"]},{"required":["b"]}]},` +
				`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"number"}},"additionalProperties":false,"required":["c"]}]}`},
		{id: "S6", name: "$ref branches, with $defs at the root",
			in: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"number"}},` +
				`"$defs":{"A":{"required":["a"]},"B":{"required":["b"],"properties":{"b":{"minimum":0}}}},"anyOf":[{"$ref":"#/$defs/A"},{"$ref":"#/$defs/B"}]}`,
			flag: `the root has the keyword "$defs"`},
		{id: "S6b", name: "a $ref branch, nothing else outside the rule",
			in:   `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"anyOf":[{"$ref":"#/properties/a"},{"required":["b"]}]}`,
			flag: `a branch uses $ref`},
		{id: "S7", name: "branches with their own properties keep them",
			in: `{"type":"object","properties":{"mode":{"type":"string"},"n":{"type":"number"}},"anyOf":[` +
				`{"properties":{"mode":{"const":"count"},"n":{"type":"integer"}},"required":["mode","n"]},{"properties":{"mode":{"const":"all"}},"required":["mode"]}]}`,
			want: `{"type":"object","properties":{"mode":{"type":"string"},"n":{"type":"number"}},"anyOf":[` +
				`{"type":"object","properties":{"mode":{"const":"count"},"n":{"type":"integer"}},"required":["mode","n"]},` +
				`{"type":"object","properties":{"mode":{"const":"all"}},"required":["mode"]}]}`},
		{id: "S8", name: "a root typed object or null",
			in:   `{"type":["object","null"],"properties":{"a":{"type":"string"},"b":{"type":"string"}},"anyOf":[{"required":["a"]},{"required":["b"]}]}`,
			flag: `the root type is not exactly "object"`},
		{id: "S9", name: "true and false branches",
			in: `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[false,{"required":["a"]},true]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[` +
				`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]},{"type":"object","properties":{"a":{"type":"string"}}}]}`},
		{id: "S10", name: "every branch a non-object type",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[{"type":"string"},{"type":"number"}]}`,
			flag: `no branch can match an object`},
		{id: "S11", name: "patternProperties and additionalProperties false, oneOf",
			in: `{"type":"object","properties":{"a":{"type":"string"}},"patternProperties":{"^x_":{"type":"integer"}},"additionalProperties":false,"oneOf":[{"required":["a"]},{"required":["x_1"]}]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}},"patternProperties":{"^x_":{"type":"integer"}},"additionalProperties":false,"oneOf":[` +
				`{"type":"object","properties":{"a":{"type":"string"}},"patternProperties":{"^x_":{"type":"integer"}},"additionalProperties":false,"required":["a"]},` +
				`{"type":"object","properties":{"a":{"type":"string"}},"patternProperties":{"^x_":{"type":"integer"}},"additionalProperties":false,"required":["x_1"]}]}`},
		{id: "S12", name: "a branch typed object or null",
			in: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"additionalProperties":false,"anyOf":[{"type":["object","null"],"required":["a"]},{"required":["b"]}]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"additionalProperties":false,"anyOf":[` +
				`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"additionalProperties":false,"required":["a"]},` +
				`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}},"additionalProperties":false,"required":["b"]}]}`},

		// Beyond the MADR's table: the corners the rule's code can get wrong.
		{id: "X1", name: "a property named description is not a description",
			in: `{"type":"object","properties":{"description":{"type":"string","description":"the text"},"properties":{"type":"array","description":"a list"}},"anyOf":[{"required":["description"]},{"required":["properties"]}]}`,
			want: `{"type":"object","properties":{"description":{"type":"string","description":"the text"},"properties":{"type":"array","description":"a list"}},"anyOf":[` +
				`{"type":"object","properties":{"description":{"type":"string"},"properties":{"type":"array"}},"required":["description"]},` +
				`{"type":"object","properties":{"description":{"type":"string"},"properties":{"type":"array"}},"required":["properties"]}]}`},
		{id: "X2", name: "data inside default, enum, const and examples keeps its description key",
			in: `{"type":"object","properties":{"p":{"type":"object","description":"d1","default":{"description":"kept"},"enum":[{"description":"e"}],` +
				`"properties":{"q":{"type":"string","description":"d2","const":"x","examples":[{"description":"z"}]}}}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`,
			want: `{"type":"object","properties":{"p":{"type":"object","description":"d1","default":{"description":"kept"},"enum":[{"description":"e"}],` +
				`"properties":{"q":{"type":"string","description":"d2","const":"x","examples":[{"description":"z"}]}}}},"anyOf":[` +
				`{"type":"object","properties":{"p":{"type":"object","default":{"description":"kept"},"enum":[{"description":"e"}],"properties":{"q":{"type":"string","const":"x","examples":[{"description":"z"}]}}}},"required":["p"]},` +
				`{"type":"object","properties":{"p":{"type":"object","default":{"description":"kept"},"enum":[{"description":"e"}],"properties":{"q":{"type":"string","const":"x","examples":[{"description":"z"}]}}}},"required":["q"]}]}`},
		{id: "X3", name: "descriptions inside items, anyOf and additionalProperties of a property are removed in the copy",
			in: `{"type":"object","properties":{"l":{"type":"array","items":{"type":"string","description":"i"},"description":"l"},` +
				`"u":{"anyOf":[{"type":"string","description":"s"},{"type":"number"}]},"m":{"type":"object","additionalProperties":{"type":"integer","description":"v"}}},"additionalProperties":{"type":"string","description":"extra"},"anyOf":[{"required":["l"]},{"required":["u"]}]}`,
			want: `{"type":"object","properties":{"l":{"type":"array","items":{"type":"string","description":"i"},"description":"l"},` +
				`"u":{"anyOf":[{"type":"string","description":"s"},{"type":"number"}]},"m":{"type":"object","additionalProperties":{"type":"integer","description":"v"}}},"additionalProperties":{"type":"string","description":"extra"},"anyOf":[` +
				`{"type":"object","properties":{"l":{"type":"array","items":{"type":"string"}},"u":{"anyOf":[{"type":"string"},{"type":"number"}]},"m":{"type":"object","additionalProperties":{"type":"integer"}}},"additionalProperties":{"type":"string"},"required":["l"]},` +
				`{"type":"object","properties":{"l":{"type":"array","items":{"type":"string"}},"u":{"anyOf":[{"type":"string"},{"type":"number"}]},"m":{"type":"object","additionalProperties":{"type":"integer"}}},"additionalProperties":{"type":"string"},"required":["u"]}]}`},
		{id: "X4", name: "both anyOf and oneOf",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[{"required":["a"]}],"oneOf":[{"required":["a"]},{"required":["b"]}]}`,
			flag: `the root has both anyOf and oneOf`},
		{id: "X5", name: "a root keyword outside the rule, untyped branches",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"allOf":[{"required":["a"]}],"anyOf":[{"required":["a"]},{"required":["b"]}]}`,
			flag: `the root has the keyword "allOf"`},
		{id: "X6", name: "unevaluatedProperties at the root",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"unevaluatedProperties":false,"anyOf":[{"required":["a"]}]}`,
			flag: `the root has the keyword "unevaluatedProperties"`},
		{id: "X7", name: "outside the rule, but every branch is a typed object: not flagged",
			in: `{"type":"object","$defs":{"d":{"type":"string"}},"properties":{"a":{"type":"string"}},"anyOf":[{"type":"object","required":["a"]}]}`},
		{id: "X8", name: "no union at all",
			in: `{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}`},
		{id: "X9", name: "the rule applies and there is nothing to change",
			in: `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}]}`},
		{id: "X10", name: "an empty union",
			in: `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[]}`},
		{id: "X11", name: "a branch that is not a schema",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[{"required":["a"]},"b"]}`,
			flag: `a branch is not a schema`},
		{id: "X12", name: "a root type given as a list of one",
			in:   `{"type":["object"],"properties":{"a":{"type":"string"}},"anyOf":[{"required":["a"]}]}`,
			flag: `the root type is not exactly "object"`},
		{id: "X13", name: "an empty branch, and a branch with allOf",
			in: `{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":false,"anyOf":[{},{"allOf":[{"required":["a"]}]}]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":false,"anyOf":[` +
				`{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":false},` +
				`{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":false,"allOf":[{"required":["a"]}]}]}`},
		{id: "X14", name: "the root's annotations stay, and a branch's own properties keep their descriptions",
			in: `{"$schema":"https://json-schema.org/draft/2020-12/schema","title":"T","description":"D","$comment":"C","deprecated":false,"default":{},"examples":[{"a":"x"}],` +
				`"type":"object","properties":{"a":{"type":"string","description":"A"}},"required":["a"],"anyOf":[{"properties":{"a":{"type":"string","description":"own"}}},{"required":["a"]}]}`,
			want: `{"$schema":"https://json-schema.org/draft/2020-12/schema","title":"T","description":"D","$comment":"C","deprecated":false,"default":{},"examples":[{"a":"x"}],` +
				`"type":"object","properties":{"a":{"type":"string","description":"A"}},"required":["a"],"anyOf":[` +
				`{"type":"object","properties":{"a":{"type":"string","description":"own"}}},{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}]}`},
		{id: "X15", name: "a branch type that is not a name or a list of names",
			in:   `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[{"required":["a"]},{"type":3}]}`,
			flag: `a branch's type is not a type name or a list of them`},
		{id: "X16", name: "a root with no properties: a branch gets none",
			in:   `{"type":"object","anyOf":[{"required":["a"]}]}`,
			want: `{"type":"object","anyOf":[{"type":"object","required":["a"]}]}`},

		// The review of 2026-11-17 (0060-MADR, amendment): a branch with its
		// own properties under a root with additionalProperties keeps its own
		// and gets none of the root's three keywords, however many it lacks.
		{id: "X17", name: "own properties under a root with additionalProperties false: nothing copied into that branch",
			in: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"boolean"}},"additionalProperties":false,` +
				`"anyOf":[{"properties":{"a":{"const":"x"}},"required":["a"]},{"required":["b"]}]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"boolean"}},"additionalProperties":false,"anyOf":[` +
				`{"type":"object","properties":{"a":{"const":"x"}},"required":["a"]},` +
				`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"},"c":{"type":"boolean"}},"additionalProperties":false,"required":["b"]}]}`},

		// The review's counter-examples: with these keywords in the union or
		// in what the rule copies, a typed branch with copied properties
		// changes a verdict (TestCounterExamplesAreReal). They are left as
		// given and flagged, by name.
		{id: "R1", name: "unevaluatedProperties in a branch",
			in:   `{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"integer"},"c":{"type":"boolean"}},"anyOf":[{"unevaluatedProperties":false,"required":["a"]},{"required":["b"]}]}`,
			flag: `"unevaluatedProperties"`},
		{id: "R2", name: "a root property whose $ref points into the union",
			in:   `{"type":"object","properties":{"a":{"$ref":"#/anyOf/0"}},"anyOf":[{"required":["a"]},{"required":["zz"]}]}`,
			flag: `"$ref"`},
		{id: "R3", name: "a nested $ref that a removed branch shifts",
			in:   `{"type":"object","properties":{"a":{}},"anyOf":[{"type":"string"},{"properties":{"a":{"$ref":"#/anyOf/0"}},"required":["a"]}]}`,
			flag: `"$ref"`},
		{id: "R4", name: "$anchor in a root property",
			in:   `{"type":"object","properties":{"a":{"type":"string","$anchor":"x"}},"anyOf":[{"required":["a"]},{"required":["b"]}]}`,
			flag: `"$anchor"`},
		{id: "R5", name: "$id in a root property",
			in:   `{"type":"object","properties":{"a":{"type":"string","$id":"urn:a"}},"anyOf":[{"required":["a"]},{"required":["b"]}]}`,
			flag: `"$id"`},
		{id: "R6", name: "$dynamicAnchor in a root property",
			in:   `{"type":"object","properties":{"a":{"type":"string","$dynamicAnchor":"x"}},"anyOf":[{"required":["a"]},{"required":["b"]}]}`,
			flag: `"$dynamicAnchor"`},
		{id: "R7", name: "the known cost: a harmless self-reference is declined too",
			in:   `{"type":"object","properties":{"a":{"type":"string"},"n":{"$ref":"#"}},"anyOf":[{"required":["a"]},{"required":["n"]}]}`,
			flag: `"$ref"`},

		// A property called like a scope keyword is a name, not the keyword:
		// the rule applies, and the name goes into the copies.
		{id: "N1", name: "root properties called $ref, $id, $anchor and $dynamicRef", named: true,
			in: `{"type":"object","properties":{"$ref":{"type":"string"},"$id":{"type":"string"},"$anchor":{"type":"integer"},"$dynamicRef":{"type":"string"}},` +
				`"additionalProperties":false,"anyOf":[{"required":["$ref"]},{"required":["$id","$anchor"]}]}`,
			want: `{"type":"object","properties":{"$ref":{"type":"string"},"$id":{"type":"string"},"$anchor":{"type":"integer"},"$dynamicRef":{"type":"string"}},` +
				`"additionalProperties":false,"anyOf":[` +
				`{"type":"object","properties":{"$ref":{"type":"string"},"$id":{"type":"string"},"$anchor":{"type":"integer"},"$dynamicRef":{"type":"string"}},"additionalProperties":false,"required":["$ref"]},` +
				`{"type":"object","properties":{"$ref":{"type":"string"},"$id":{"type":"string"},"$anchor":{"type":"integer"},"$dynamicRef":{"type":"string"}},"additionalProperties":false,"required":["$id","$anchor"]}]}`},
		{id: "N2", name: "a branch's own properties called $ref and unevaluatedProperties", named: true,
			in: `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[` +
				`{"properties":{"$ref":{"type":"string"},"unevaluatedProperties":{"type":"boolean"}},"required":["$ref"]},{"required":["a"]}]}`,
			want: `{"type":"object","properties":{"a":{"type":"string"}},"anyOf":[` +
				`{"type":"object","properties":{"$ref":{"type":"string"},"unevaluatedProperties":{"type":"boolean"}},"required":["$ref"]},` +
				`{"type":"object","properties":{"a":{"type":"string"}},"required":["a"]}]}`},
		{id: "N3", name: "scope keywords as data inside default, enum and const are not keywords", named: true,
			in: `{"type":"object","properties":{"p":{"type":"object","default":{"$ref":"#","$id":"x"},"enum":[{"$anchor":"y"}],"const":{"unevaluatedProperties":false}}},` +
				`"anyOf":[{"required":["p"]},{"required":["q"]}]}`,
			want: `{"type":"object","properties":{"p":{"type":"object","default":{"$ref":"#","$id":"x"},"enum":[{"$anchor":"y"}],"const":{"unevaluatedProperties":false}}},"anyOf":[` +
				`{"type":"object","properties":{"p":{"type":"object","default":{"$ref":"#","$id":"x"},"enum":[{"$anchor":"y"}],"const":{"unevaluatedProperties":false}}},"required":["p"]},` +
				`{"type":"object","properties":{"p":{"type":"object","default":{"$ref":"#","$id":"x"},"enum":[{"$anchor":"y"}],"const":{"unevaluatedProperties":false}}},"required":["q"]}]}`},
	}
}

// ruleInput is the case's schema: the real schema's is the file's.
func ruleInput(t testing.TB, c ruleCase) map[string]any { return decode(t, c.in) }

// allCases are ruleCases and the real schema, S0, from its copied file.
func allCases(t testing.TB) []ruleCase {
	t.Helper()
	cs := ruleCases()
	b, err := os.ReadFile(filepath.Join("testdata", "magictools-execute-pipeline.json"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := os.ReadFile(filepath.Join("testdata", "magictools-execute-pipeline.rewritten.json"))
	if err != nil {
		t.Fatal(err)
	}
	return append([]ruleCase{{id: "S0", name: "the real schema, verbatim", in: string(b), want: string(w)}}, cs...)
}

func equalJSON(a, b any) bool { return reflect.DeepEqual(a, b) }

// The rule's output for each case is the expected JSON, or the schema is
// returned as given with its flag, and the input is never written to
// (0060-MADR D2, D3).
func TestRewriteRootUnion(t *testing.T) {
	for _, c := range allCases(t) {
		t.Run(c.id, func(t *testing.T) {
			in := ruleInput(t, c)
			before := ruleInput(t, c)
			out, rewritten, flag := rewriteRootUnion(in)
			if !equalJSON(in, before) {
				t.Fatalf("the rule wrote to its input:\n got %v\nwant %v", in, before)
			}
			if c.want == "" {
				if rewritten || !equalJSON(out, before) {
					t.Fatalf("%s: rewritten %v, out\n%v\nwant the schema as given\n%v", c.name, rewritten, out, before)
				}
				if c.flag == "" && flag != "" {
					t.Fatalf("%s: flagged %q, want not flagged", c.name, flag)
				}
				if c.flag != "" && !strings.Contains(flag, c.flag) {
					t.Fatalf("%s: flag %q, want one holding %q", c.name, flag, c.flag)
				}
				return
			}
			want := decode(t, c.want)
			if !rewritten || flag != "" || !equalJSON(out, want) {
				t.Fatalf("%s: rewritten %v, flag %q, out\n%v\nwant\n%v", c.name, rewritten, flag, out, want)
			}
		})
	}
}

// A flagged reason names the union and says why the schema was left alone,
// as 0060-MADR D4 words the warning's reason.
func TestFlagReasonWording(t *testing.T) {
	_, _, flag := rewriteRootUnion(decode(t, `{"type":["object","null"],"properties":{"a":{"type":"string"}},"oneOf":[{"required":["a"]}]}`))
	const want = `its root oneOf has a branch that is not an object, which some providers refuse; left as given because the root type is not exactly "object"`
	if flag != want {
		t.Fatalf("flag\n%q\nwant\n%q", flag, want)
	}
	_, _, both := rewriteRootUnion(decode(t, `{"type":"object","anyOf":[{"required":["a"]}],"oneOf":[{"required":["a"]}]}`))
	if !strings.HasPrefix(both, "its root anyOf and oneOf has a branch that is not an object") {
		t.Fatalf("flag %q", both)
	}
}

// The corpus of the preservation test, from a fixed seed: every subset of
// the schema's property names plus one unknown name, each with a value of
// its declared type; 2,000 objects over the same names and a pool of
// values; and nine values that are not objects.
func corpus(s map[string]any) []any {
	names := map[string]bool{"zz": true}
	var walk func(v any)
	walk = func(v any) {
		switch v := v.(type) {
		case map[string]any:
			if p, ok := v["properties"].(map[string]any); ok {
				for k := range p {
					names[k] = true
				}
			}
			if r, ok := v["required"].([]any); ok {
				for _, n := range r {
					if s, ok := n.(string); ok {
						names[s] = true
					}
				}
			}
			for _, x := range v {
				walk(x)
			}
		case []any:
			for _, x := range v {
				walk(x)
			}
		}
	}
	walk(s)
	keys := make([]string, 0, len(names))
	for k := range names {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	props := map[string]any{}
	if p, ok := s["properties"].(map[string]any); ok {
		props = p
	}
	typed := func(name string) any {
		p := map[string]any{} // an undeclared name takes a string
		if d, ok := props[name].(map[string]any); ok {
			p = d
		}
		switch p["type"] {
		case "integer":
			return float64(7)
		case "number":
			return 2.5
		case "boolean":
			return true
		case "array":
			return []any{"a"}
		case "object":
			return map[string]any{}
		}
		return "x"
	}
	out := []any{"x", "", float64(42), 1.5, true, nil, []any{}, []any{"a"}, []any{map[string]any{"a": "x"}}}
	for mask := range 1 << len(keys) {
		o := map[string]any{}
		for i, k := range keys {
			if mask&(1<<i) != 0 {
				o[k] = typed(k)
			}
		}
		out = append(out, o)
	}
	pool := []any{"x", "", float64(42), float64(0), float64(-3), 1.5, true, false, nil, []any{"a"}, []any{float64(1)}, map[string]any{}, map[string]any{"k": "v"}, "count", "all"}
	rng := rand.New(rand.NewPCG(20261117, 1))
	for range 2000 {
		o := map[string]any{}
		for _, i := range rng.Perm(len(keys))[:rng.IntN(len(keys)+1)] {
			o[keys[i]] = pool[rng.IntN(len(pool))]
		}
		out = append(out, o)
	}
	return out
}

// verdicts are jsonschema-go's verdict on each instance against s.
func verdicts(t testing.TB, s map[string]any, insts []any) []bool {
	t.Helper()
	b, err := jsonx.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	var sch jsonschema.Schema
	if err := jsonx.Unmarshal(b, &sch); err != nil {
		t.Fatalf("%v\n%s", err, b)
	}
	rs, err := sch.Resolve(nil)
	if err != nil {
		t.Fatalf("%v\n%s", err, b)
	}
	out := make([]bool, len(insts))
	for i, v := range insts {
		out[i] = rs.Validate(v) == nil
	}
	return out
}

// preserved checks that original and rewritten give the same verdict on
// every instance of the corpus, and returns how many were accepted. It
// fails when the corpus is no test: nothing accepted, or nothing refused.
func preserved(t testing.TB, original, rewritten map[string]any) (instances, accepted int) {
	t.Helper()
	insts := corpus(original)
	a, b := verdicts(t, original, insts), verdicts(t, rewritten, insts)
	diff := 0
	for i := range insts {
		if a[i] != b[i] {
			if diff == 0 {
				t.Errorf("instance %d, %v: the original says %v, the rewrite %v", i, insts[i], a[i], b[i])
			}
			diff++
		}
		if a[i] {
			accepted++
		}
	}
	if diff > 0 {
		t.Errorf("%d of %d verdicts differ", diff, len(insts))
	}
	if accepted == 0 || accepted == len(insts) {
		t.Errorf("the original accepts %d of %d instances: the corpus does not test it", accepted, len(insts))
	}
	return len(insts), accepted
}

// Input preservation: for every rewritten case, the original and the
// rewritten schema give the same verdict on every instance of the corpus
// (0060-MADR D2, and its measurements R1 and R2).
//
// It runs over every case, so a shape the rule must leave alone and a
// change that rewrites it fails here too: a case the rule does not rewrite
// has nothing to preserve, and is held by TestRewriteRootUnion.
func TestRewritePreservesInputs(t *testing.T) {
	total := 0
	for _, c := range allCases(t) {
		t.Run(c.id, func(t *testing.T) {
			in := ruleInput(t, c)
			out, rewritten, _ := rewriteRootUnion(in)
			if !rewritten {
				if c.want != "" {
					t.Fatalf("%s: not rewritten", c.name)
				}
				return // left as given: nothing to preserve
			}
			n, acc := preserved(t, in, out)
			t.Logf("%s: %d instances, %d accepted by both", c.name, n, acc)
			total += n
			// No schema the rule rewrites holds a keyword whose meaning depends
			// on where a schema sits or on its neighbours' annotations, named
			// here again, apart from the rule's own list, and searched for as a
			// member name anywhere. Only a case with a property called like one
			// may hold the name.
			if kw := memberAnywhere(out, testScopeNames); kw != "" && !c.named {
				t.Errorf("%s: the rewrite holds %q", c.name, kw)
			}
		})
	}
	t.Logf("%d instances in all", total)
}

// A schema the rule returns as given is the schema as given, byte for byte
// of its JSON, so what the provider refuses or accepts is what it did.
func TestRefusedShapesAreUntouched(t *testing.T) {
	n := 0
	for _, c := range allCases(t) {
		if c.want != "" {
			continue
		}
		n++
		in := ruleInput(t, c)
		out, rewritten, _ := rewriteRootUnion(in)
		a, errA := jsonx.Marshal(in)
		b, errB := jsonx.Marshal(out)
		if errA != nil || errB != nil || rewritten || string(a) != string(b) {
			t.Errorf("%s: rewritten %v\n got %s\nwant %s", c.id, rewritten, b, a)
		}
	}
	if n < 10 {
		t.Fatalf("%d cases, want the refused shapes of the table", n)
	}
}

// The precondition Grok states: every branch of every rewritten union has
// "type": "object", and a branch that had no object keyword carries the
// root's (0060-MADR K4, D2 step 4).
func TestRewrittenBranchesAreObjects(t *testing.T) {
	for _, c := range allCases(t) {
		if c.want == "" {
			continue
		}
		in := ruleInput(t, c)
		out, _, _ := rewriteRootUnion(in)
		for _, kw := range []string{"anyOf", "oneOf"} {
			branches, ok := out[kw].([]any)
			if !ok {
				continue
			}
			if len(branches) == 0 {
				t.Errorf("%s: %s has no branch", c.id, kw)
			}
			for i, b := range branches {
				m, ok := b.(map[string]any)
				if !ok || m["type"] != "object" {
					t.Errorf("%s: %s[%d] = %v, want a branch with type object", c.id, kw, i, b)
					continue
				}
				for _, k := range []string{"properties", "patternProperties", "additionalProperties"} {
					if _, rootHas := in[k]; rootHas && !hasObjectKeyword(m) {
						t.Errorf("%s: %s[%d] has no object keyword though the root has %s", c.id, kw, i, k)
					}
				}
			}
			// The real schema's branches had no object keyword: each carries the
			// root's properties, without descriptions.
			if c.id == "S0" || c.id == "S1" {
				for i, b := range branches {
					if got, want := mapAt(t, b)["properties"], decode(t, props7); !equalJSON(got, any(want)) {
						t.Errorf("%s: %s[%d].properties = %v", c.id, kw, i, got)
					}
				}
			}
		}
	}
}

// mapAt is v as an object, or fails the test.
func mapAt(t testing.TB, v any) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%v is not an object", v)
	}
	return m
}

// listAt is v as a list, or fails the test.
func listAt(t testing.TB, v any) []any {
	t.Helper()
	l, ok := v.([]any)
	if !ok {
		t.Fatalf("%v is not a list", v)
	}
	return l
}

func hasObjectKeyword(m map[string]any) bool {
	for _, k := range []string{"properties", "patternProperties", "additionalProperties"} {
		if _, ok := m[k]; ok {
			return true
		}
	}
	return false
}

// The root is untouched: every keyword of the rewritten root but the union
// equals the original's, descriptions included (0060-MADR D1).
func TestRootIsUntouched(t *testing.T) {
	checked := 0
	for _, c := range allCases(t) {
		if c.want == "" {
			continue
		}
		in := ruleInput(t, c)
		out, _, _ := rewriteRootUnion(in)
		if len(out) != len(in) {
			t.Errorf("%s: %d root keywords, want %d", c.id, len(out), len(in))
		}
		for k, v := range in {
			if k == "anyOf" || k == "oneOf" {
				continue
			}
			if !equalJSON(out[k], v) {
				t.Errorf("%s: root %s is\n%v\nwant\n%v", c.id, k, out[k], v)
			}
			checked++
		}
	}
	// The real schema's seven descriptions are among what was compared.
	out, _, _ := rewriteRootUnion(readData(t, "magictools-execute-pipeline.json"))
	for name, p := range mapAt(t, out["properties"]) {
		if d, ok := mapAt(t, p)["description"].(string); !ok || d == "" {
			t.Errorf("the root's %s has no description", name)
		}
	}
	if checked == 0 {
		t.Fatal("nothing compared")
	}
}

// Idempotence: the rule applied to its own output changes nothing and
// flags nothing new; for a schema it returns as given, it is the same
// twice (0060-MADR D2).
func TestRewriteIsIdempotent(t *testing.T) {
	for _, c := range allCases(t) {
		once, rewritten, flag := rewriteRootUnion(ruleInput(t, c))
		twice, again, flag2 := rewriteRootUnion(once)
		if !equalJSON(once, twice) || flag != flag2 {
			t.Errorf("%s: twice differs from once\nonce  %v\ntwice %v", c.id, once, twice)
		}
		if again {
			t.Errorf("%s: the second pass rewrote again (first pass %v)", c.id, rewritten)
		}
	}
}

// The rule shares nothing with its input: writing into what it returns
// leaves the input as it was.
func TestRewriteSharesNothing(t *testing.T) {
	for _, c := range allCases(t) {
		in := ruleInput(t, c)
		out, _, _ := rewriteRootUnion(in)
		scribble(out)
		if before := ruleInput(t, c); !equalJSON(in, before) {
			t.Errorf("%s: writing to the output changed the input", c.id)
		}
	}
}

// scribble writes into every map and list of v.
func scribble(v any) {
	switch v := v.(type) {
	case map[string]any:
		for _, x := range v {
			scribble(x)
		}
		v["scribble"] = true
	case []any:
		for i, x := range v {
			scribble(x)
			if _, ok := x.(string); ok {
				v[i] = "scribble"
			}
		}
	}
}

// newTool does not write to the go-sdk's own schema map: gobble's additions
// and the rule work on a copy (0060-MADR D1; G2 wrote into it).
func TestNewToolDoesNotWriteTheSDKsMap(t *testing.T) {
	for _, tc := range []struct{ name, schema string }{
		{"the real schema", ""},
		{"no type and no properties", `{"anyOf":[{"required":["a"]},{"required":["b"]}]}`},
		{"no properties", `{"type":"object","description":"x"}`},
		{"an object or null root", mcptest.NullableUnionSchema},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var m, before map[string]any
			if tc.schema == "" {
				m, before = readData(t, "magictools-execute-pipeline.json"), readData(t, "magictools-execute-pipeline.json")
			} else {
				m, before = decode(t, tc.schema), decode(t, tc.schema)
			}
			tl := &mcp.Tool{Name: "x", InputSchema: m}
			got := newTool(&conn{server: Server{Name: "fx"}}, tl, "mcp__fx__x")
			if !equalJSON(tl.InputSchema, any(before)) {
				t.Fatalf("the SDK's map was written to:\n got %v\nwant %v", tl.InputSchema, before)
			}
			if len(got.spec.InputSchema) == 0 {
				t.Fatal("no schema")
			}
		})
	}
}

// inputSchema keeps gobble's additions, and the rule runs after them: a
// root with no type is typed first, so a union under it is rewritten.
func TestInputSchemaAdditionsThenRule(t *testing.T) {
	spec, rewritten, flag := inputSchema(nil)
	if string(spec) != `{"properties":{},"type":"object"}` || rewritten || flag != "" {
		t.Fatalf("nil: %s %v %q", spec, rewritten, flag)
	}
	spec, rewritten, flag = inputSchema("not a schema")
	if string(spec) != `{"properties":{},"type":"object"}` || rewritten || flag != "" {
		t.Fatalf("a string: %s %v %q", spec, rewritten, flag)
	}
	// S3 as a server could send it, with no root type: gobble adds type
	// object, then the rule applies.
	s3 := decode(t, ruleCasesByID(t)["S3"].in)
	spec, rewritten, flag = inputSchema(s3)
	if !rewritten || flag != "" {
		t.Fatalf("S3: rewritten %v, flag %q", rewritten, flag)
	}
	var got map[string]any
	if err := jsonx.Unmarshal(spec, &got); err != nil {
		t.Fatal(err)
	}
	for i, b := range listAt(t, got["anyOf"]) {
		if mapAt(t, b)["type"] != "object" {
			t.Errorf("S3 branch %d not typed: %v", i, b)
		}
	}
	// The real schema: the spec is the expected rewrite; the same twice.
	spec, rewritten, flag = inputSchema(readData(t, "magictools-execute-pipeline.json"))
	want := readData(t, "magictools-execute-pipeline.rewritten.json")
	var have map[string]any
	if err := jsonx.Unmarshal(spec, &have); err != nil || !rewritten || flag != "" || !equalJSON(have, want) {
		t.Fatalf("the real schema: %v rewritten %v flag %q\n%s", err, rewritten, flag, spec)
	}
	again, _, _ := inputSchema(readData(t, "magictools-execute-pipeline.json"))
	if string(again) != string(spec) {
		t.Fatal("the spec is not deterministic")
	}
	// S8 goes out as given: its spec is its JSON.
	spec, rewritten, flag = inputSchema(decode(t, mcptest.NullableUnionSchema))
	var s8 map[string]any
	if err := jsonx.Unmarshal(spec, &s8); err != nil || rewritten || !strings.Contains(flag, "root anyOf") ||
		!equalJSON(s8, decode(t, mcptest.NullableUnionSchema)) {
		t.Fatalf("S8: %v rewritten %v flag %q\n%s", err, rewritten, flag, spec)
	}
}

func ruleCasesByID(t testing.TB) map[string]ruleCase {
	t.Helper()
	out := map[string]ruleCase{}
	for _, c := range allCases(t) {
		out[c.id] = c
	}
	return out
}

// The fixture server's copy of the real schema is the copy in testdata,
// which names its source.
func TestFixtureSchemaIsTheCopy(t *testing.T) {
	if !equalJSON(decode(t, mcptest.ExecutePipelineSchema), readData(t, "magictools-execute-pipeline.json")) {
		t.Fatal("mcptest.ExecutePipelineSchema differs from testdata/magictools-execute-pipeline.json")
	}
	b, err := os.ReadFile(filepath.Join("testdata", "magictools-execute-pipeline.source.txt"))
	if err != nil || !strings.Contains(string(b), "mcp-server-magictools") || !strings.Contains(string(b), "internal/handler/inventory.go") ||
		!strings.Contains(string(b), "368ac6a") {
		t.Fatalf("the source note must name the repository, path and commit: %v\n%s", err, b)
	}
}

// The fixture schema's shapes are what the MADR says: the real one is
// rewritten by the rule, the other flagged.
func TestFixtureShapes(t *testing.T) {
	_, rewritten, flag := rewriteRootUnion(decode(t, mcptest.ExecutePipelineSchema))
	if !rewritten || flag != "" {
		t.Errorf("execute_pipeline: rewritten %v, flag %q", rewritten, flag)
	}
	_, rewritten, flag = rewriteRootUnion(decode(t, mcptest.NullableUnionSchema))
	if rewritten || flag == "" {
		t.Errorf("pick_one: rewritten %v, flag %q", rewritten, flag)
	}
}
