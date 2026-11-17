package mcpclient

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"testing"
)

// These tests hold the amendment of 2026-11-17 to 0060-MADR D2, after the
// independent review: the rule refuses a union or properties that hold a
// keyword whose meaning depends on where a schema sits or on its neighbours'
// annotations, wherever it is in them, and does not mistake a property of
// that name for the keyword.

// testScopeNames are the nine keywords of the amendment, written out again
// here so that the tests do not take the rule's list on trust.
var testScopeNames = []string{"$ref", "$dynamicRef", "$recursiveRef", "$id", "$anchor", "$dynamicAnchor", "$recursiveAnchor",
	"unevaluatedProperties", "unevaluatedItems"}

// scopeValue is a value each of testScopeNames takes, in its own keyword.
var scopeValue = map[string]string{"$ref": `"#/properties/p"`, "$dynamicRef": `"#x"`, "$recursiveRef": `"#"`, "$id": `"urn:x"`,
	"$anchor": `"x"`, "$dynamicAnchor": `"x"`, "$recursiveAnchor": `true`, "unevaluatedProperties": `false`, "unevaluatedItems": `false`}

// memberAnywhere is the first of names that v, a decoded schema, has as a
// member name at any depth, or "": a search of the text of the schema, with no
// idea of a keyword.
func memberAnywhere(v any, names []string) string {
	switch v := v.(type) {
	case map[string]any:
		for _, k := range slices.Sorted(maps.Keys(v)) {
			if slices.Contains(names, k) {
				return k
			}
			if f := memberAnywhere(v[k], names); f != "" {
				return f
			}
		}
	case []any:
		for _, x := range v {
			if f := memberAnywhere(x, names); f != "" {
				return f
			}
		}
	}
	return ""
}

// scopePositions are the places a schema sits in a root union or in the
// keywords the rule copies: each wraps the schema %s in the position, inside a
// root the rule applies to when %s is harmless (its control: a title).
// "Copied" positions are in the root's properties, patternProperties or
// additionalProperties; the others are in a branch of the union.
var scopePositions = []struct{ name, tmpl string }{
	// Inside what the rule copies, at the root.
	{"root properties", `{"type":"object","properties":{"p":%s},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root patternProperties", `{"type":"object","patternProperties":{"^x":%s},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root additionalProperties", `{"type":"object","properties":{"p":{}},"additionalProperties":%s,"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property items", `{"type":"object","properties":{"p":{"type":"array","items":%s}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property prefixItems", `{"type":"object","properties":{"p":{"prefixItems":[%s]}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property allOf", `{"type":"object","properties":{"p":{"allOf":[%s]}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property anyOf", `{"type":"object","properties":{"p":{"anyOf":[%s]}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property oneOf", `{"type":"object","properties":{"p":{"oneOf":[%s]}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property not", `{"type":"object","properties":{"p":{"not":%s}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property if", `{"type":"object","properties":{"p":{"if":%s}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property then", `{"type":"object","properties":{"p":{"then":%s}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property else", `{"type":"object","properties":{"p":{"else":%s}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property dependentSchemas", `{"type":"object","properties":{"p":{"dependentSchemas":{"x":%s}}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property propertyNames", `{"type":"object","properties":{"p":{"propertyNames":%s}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property contains", `{"type":"object","properties":{"p":{"contains":%s}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property $defs", `{"type":"object","properties":{"p":{"$defs":{"d":%s}}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property definitions", `{"type":"object","properties":{"p":{"definitions":{"d":%s}}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},
	{"root property, three deep", `{"type":"object","properties":{"p":{"type":"object","properties":{"q":{"type":"object","properties":{"r":%s}}}}},"anyOf":[{"required":["p"]},{"required":["q"]}]}`},

	// Inside the union.
	{"branch itself", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},%s]}`},
	{"branch properties", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"properties":{"q":%s}}]}`},
	{"branch patternProperties", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"patternProperties":{"^x":%s}}]}`},
	{"branch additionalProperties", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"additionalProperties":%s}]}`},
	{"branch items", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"items":%s}]}`},
	{"branch prefixItems", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"prefixItems":[%s]}]}`},
	{"branch allOf", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"allOf":[%s]}]}`},
	{"branch anyOf", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"anyOf":[%s]}]}`},
	{"branch oneOf", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"oneOf":[%s]}]}`},
	{"branch not", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"not":%s}]}`},
	{"branch if", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"if":%s}]}`},
	{"branch then", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"then":%s}]}`},
	{"branch else", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"else":%s}]}`},
	{"branch dependentSchemas", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"dependentSchemas":{"x":%s}}]}`},
	{"branch propertyNames", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"propertyNames":%s}]}`},
	{"branch contains", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"contains":%s}]}`},
	{"branch $defs", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"$defs":{"d":%s}}]}`},
	{"branch definitions", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"definitions":{"d":%s}}]}`},
	{"branch, three deep", `{"type":"object","properties":{"p":{}},"anyOf":[{"required":["p"]},{"properties":{"q":{"type":"object","properties":{"r":{"allOf":[%s]}}}}}]}`},
	{"branch of a oneOf root", `{"type":"object","properties":{"p":{}},"oneOf":[{"required":["p"]},{"properties":{"q":%s}}]}`},
}

// Each of the nine keywords, in each position of a union or of the keywords
// the rule copies, at any depth, leaves the schema as given and flagged by
// the keyword's name; the same schema with a harmless title in its place is
// rewritten, so that the position is one the rule reaches. The nine sit at
// the keyword of a schema, as {"keyword": value}: every position holds that
// schema (0060-MADR D2, amendment of 2026-11-17).
func TestScopeKeywordsRefusedEverywhere(t *testing.T) {
	for _, pos := range scopePositions {
		control := fmt.Sprintf(pos.tmpl, `{"title":"t"}`)
		if _, rewritten, flag := rewriteRootUnion(decode(t, control)); !rewritten || flag != "" {
			t.Errorf("%s: the control is not rewritten (flag %q): the position is not one the rule reaches\n%s", pos.name, flag, control)
		}
		for _, kw := range testScopeNames {
			t.Run(pos.name+"/"+kw, func(t *testing.T) {
				in := fmt.Sprintf(pos.tmpl, `{"`+kw+`":`+scopeValue[kw]+`}`)
				out, rewritten, flag := rewriteRootUnion(decode(t, in))
				if rewritten || !equalJSON(out, decode(t, in)) {
					t.Fatalf("rewritten %v, out\n%v\nwant the schema as given\n%s", rewritten, out, in)
				}
				if !strings.Contains(flag, kw) || !strings.Contains(flag, "its root ") {
					t.Fatalf("flag %q, want one naming %q", flag, kw)
				}
			})
		}
	}
}

// A property called like one of the nine, at the root, in a branch or deep,
// in a name list, or data in a default, an enum, a const or an example, is
// not the keyword: the rule applies, and the name is carried into the copies.
func TestScopeKeywordNamesAreNotKeywords(t *testing.T) {
	for _, kw := range testScopeNames {
		t.Run(kw, func(t *testing.T) {
			q := `"` + kw + `"`
			in := `{"type":"object","properties":{` + q + `:{"type":"string"},` +
				`"p":{"type":"object","properties":{` + q + `:{"type":"integer"}},"default":{` + q + `:1},"enum":[{` + q + `:1}],"const":{` + q + `:1},"examples":[{` + q + `:1}]}},` +
				`"anyOf":[{"properties":{` + q + `:{"type":"string"}},"required":[` + q + `]},{"required":["p"]}]}`
			out, rewritten, flag := rewriteRootUnion(decode(t, in))
			if !rewritten || flag != "" {
				t.Fatalf("a property called %s: rewritten %v, flag %q\n%s", kw, rewritten, flag, in)
			}
			n, acc := preserved(t, decode(t, in), out)
			if acc == 0 || n == 0 {
				t.Fatal("empty corpus")
			}
		})
	}
}

// The review's counter-examples are real: a hand-written rewrite as D2 stood
// before the amendment, copying the root's properties into each typed branch,
// changes a verdict on the instance (or, for an anchor or an id, makes the
// schema invalid), which is why the rule now declines them.
func TestCounterExamplesAreReal(t *testing.T) {
	p3 := `{"a":{"type":"string"},"b":{"type":"integer"},"c":{"type":"boolean"}}`
	for _, tc := range []struct {
		id, naive, inst string
		original, now   bool
	}{
		{"R1", `{"type":"object","properties":` + p3 + `,"anyOf":[` +
			`{"type":"object","properties":` + p3 + `,"unevaluatedProperties":false,"required":["a"]},{"type":"object","properties":` + p3 + `,"required":["b"]}]}`,
			`{"a":"x"}`, false, true},
		{"R2", `{"type":"object","properties":{"a":{"$ref":"#/anyOf/0"}},"anyOf":[` +
			`{"type":"object","properties":{"a":{"$ref":"#/anyOf/0"}},"required":["a"]},{"type":"object","properties":{"a":{"$ref":"#/anyOf/0"}},"required":["zz"]}]}`,
			`{"a":"x"}`, true, false},
		{"R3", `{"type":"object","properties":{"a":{}},"anyOf":[{"type":"object","properties":{"a":{"$ref":"#/anyOf/0"}},"required":["a"]}]}`,
			`{"a":"x"}`, true, false},
	} {
		c := ruleCasesByID(t)[tc.id]
		inst := decode(t, tc.inst)
		if got := verdicts(t, decode(t, c.in), []any{inst})[0]; got != tc.original {
			t.Errorf("%s: the original says %v on %s, want %v", tc.id, got, tc.inst, tc.original)
		}
		if got := verdicts(t, decode(t, tc.naive), []any{inst})[0]; got != tc.now {
			t.Errorf("%s: the naive rewrite says %v on %s, want %v", tc.id, got, tc.inst, tc.now)
		}
	}
}
