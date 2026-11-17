package mcpclient

import (
	"maps"
	"slices"
	"strings"
)

// The rule of 0060-MADR D2: a tool whose input schema is an object with a
// root-level anyOf or oneOf is sent with each branch an object schema that
// carries the root's object keywords, which is what xAI's Grok accepts of a
// root union, and which accepts exactly the arguments the original does.
// A schema outside the rule is left as given, and flagged when it is the
// shape Grok refuses (D3).

const (
	// typeObject is the type name of an object schema.
	typeObject = "object"
	// kwAdditionalProperties is the keyword for the values a schema's
	// unnamed members may take.
	kwAdditionalProperties = "additionalProperties"
	// kwProperties is the keyword for schemas by member name.
	kwProperties = "properties"
	// kwPatternProperties is the keyword for schemas by member-name pattern.
	kwPatternProperties = "patternProperties"
)

// rootAllowed are the keywords a root may hold besides its one union for the
// rule to apply: the schema's own, and annotations (D2).
var rootAllowed = map[string]bool{
	"type": true, kwProperties: true, "required": true, kwAdditionalProperties: true, kwPatternProperties: true,
	"description": true, "title": true, "examples": true, "default": true, "deprecated": true, "$comment": true, "$schema": true,
}

// objectKeywords are the root's keywords a branch with none of them gets
// (D2 step 4): those that act on an object's own members.
var objectKeywords = []string{kwProperties, kwPatternProperties, kwAdditionalProperties}

// unionKeywords are the keywords that make a root a union.
var unionKeywords = []string{"anyOf", "oneOf"}

// rewriteRootUnion applies the rule of 0060-MADR D2 to the input schema s
// of a tool. It returns a schema that shares nothing with s, whether the
// rule changed it, and, when the rule does not apply to a root union with a
// branch that is not typed object, the reason the schema is left as given
// (D3), which D4 puts in the warning. s is never written to, and the root
// of the result is s's root but for its union. Applied to its own result
// the rule changes nothing.
func rewriteRootUnion(s map[string]any) (out map[string]any, rewritten bool, flag string) {
	out = cloneMap(s)
	key, branches, refusal := ruleApplies(s)
	if key == "" {
		return out, false, flagText(s, refusal)
	}
	kept := make([]any, 0, len(branches))
	for _, b := range branches {
		if keep, _ := objectBranch(b); keep {
			kept = append(kept, b)
		}
	}
	if len(kept) == 0 { // D2 step 2: nothing can match an object
		return out, false, flagText(s, "no branch can match an object")
	}
	norm := make([]any, len(kept))
	for i, b := range kept {
		norm[i] = objectify(b, s)
	}
	out[key] = norm
	return out, !equalValue(out, s), ""
}

// ruleApplies reports the union keyword the rule of D2 applies on and its
// branches, or, with "" for the keyword, why the rule does not apply ("" for
// a schema with no union, which is not a refusal). It checks the conditions
// in D2's order: the root's type, one union, the root's other keywords, and
// every branch.
func ruleApplies(s map[string]any) (key string, branches []any, refusal string) {
	var have []string
	for _, k := range unionKeywords {
		if _, ok := s[k]; ok {
			have = append(have, k)
		}
	}
	if len(have) == 0 {
		return "", nil, ""
	}
	if s["type"] != typeObject {
		return "", nil, `the root type is not exactly "object"`
	}
	if len(have) != 1 {
		return "", nil, "the root has both anyOf and oneOf"
	}
	for _, k := range slices.Sorted(maps.Keys(s)) {
		if k != have[0] && !rootAllowed[k] {
			return "", nil, `the root has the keyword "` + k + `", which the rule does not cover`
		}
	}
	branches, ok := s[have[0]].([]any)
	if !ok {
		return "", nil, "the root's " + have[0] + " is not a list"
	}
	for _, b := range branches {
		if _, why := objectBranch(b); why != "" {
			return "", nil, why
		}
	}
	return have[0], branches, ""
}

// objectBranch classifies one branch of a root union: whether it can match
// an object (a branch that cannot is removed, D2 step 1), or, with a reason,
// that it is a branch the rule does not cover.
func objectBranch(b any) (keep bool, refusal string) {
	switch b := b.(type) {
	case bool:
		return b, ""
	case map[string]any:
		if _, ok := b["$ref"]; ok {
			return false, "a branch uses $ref"
		}
		t, ok := b["type"]
		if !ok {
			return true, ""
		}
		switch t := t.(type) {
		case string:
			return t == typeObject, ""
		case []any:
			keep = false
			for _, n := range t {
				name, ok := n.(string)
				if !ok {
					return false, "a branch's type is not a type name or a list of them"
				}
				keep = keep || name == typeObject
			}
			return keep, ""
		}
		return false, "a branch's type is not a type name or a list of them"
	}
	return false, "a branch is not a schema"
}

// objectify is a kept branch as an object schema (D2 steps 3 and 4): a copy,
// typed "object", with the root's object keywords, descriptions removed, when
// it has none of its own. Nested unions are not touched (step 5).
func objectify(b any, root map[string]any) map[string]any {
	nb := map[string]any{}
	if m, ok := b.(map[string]any); ok {
		nb = cloneMap(m)
	}
	nb["type"] = typeObject
	own := false
	for _, k := range objectKeywords {
		_, ok := nb[k]
		own = own || ok
	}
	if !own {
		for _, k := range objectKeywords {
			if v, ok := root[k]; ok {
				nb[k] = stripDescriptions(cloneValue(v), k)
			}
		}
	}
	return nb
}

// hasUntypedBranch reports whether the root has a union with a branch that
// is not an object schema typed "object": the shape xAI refuses (D3, X1).
// An empty or malformed union has none.
func hasUntypedBranch(s map[string]any) bool {
	for _, k := range unionKeywords {
		branches, ok := s[k].([]any)
		if !ok {
			continue
		}
		for _, b := range branches {
			if m, ok := b.(map[string]any); !ok || m["type"] != typeObject {
				return true
			}
		}
	}
	return false
}

// flagText is the reason D4 puts in the warning for a root union the rule
// left as given because of why, when the union has a branch that is not
// typed object; "" otherwise.
func flagText(s map[string]any, why string) string {
	if !hasUntypedBranch(s) {
		return ""
	}
	var have []string
	for _, k := range unionKeywords {
		if branches, ok := s[k].([]any); ok && len(branches) > 0 {
			have = append(have, k)
		}
	}
	text := "its root " + strings.Join(have, " and ") + " has a branch that is not an object, which some providers refuse"
	if why == "" {
		return text
	}
	return text + "; left as given because " + why
}

// Keywords whose value is a schema, a list of schemas, or a map of names to
// schemas: the places stripDescriptions looks for a description. The others
// (enum, const, default, examples, required, type and so on) hold data or
// names, and a "description" in them is not a keyword.
var (
	schemaKeywords    = []string{"items", "additionalItems", "contains", "propertyNames", "not", "if", "then", "else", kwAdditionalProperties, "unevaluatedProperties", "unevaluatedItems", "contentSchema"}
	schemaListKeyword = []string{"allOf", "anyOf", "oneOf", "prefixItems"}
	schemaMapKeywords = []string{kwProperties, kwPatternProperties, "$defs", "definitions", "dependentSchemas", "dependencies"}
)

// stripDescriptions removes every description keyword from v, the value of
// the keyword kw of a schema, in place: v is a copy. A property named
// description is a name, not a keyword, and survives; so does a "description"
// inside enum, const, default or examples.
func stripDescriptions(v any, kw string) any {
	switch {
	case slices.Contains(schemaMapKeywords, kw):
		if m, ok := v.(map[string]any); ok {
			for name, sub := range m {
				m[name] = stripSchema(sub)
			}
		}
		return v
	case slices.Contains(schemaListKeyword, kw):
		if l, ok := v.([]any); ok {
			for i, sub := range l {
				l[i] = stripSchema(sub)
			}
		}
		return v
	}
	return stripSchema(v)
}

// stripSchema removes the description keyword of the schema v and of every
// schema inside it.
func stripSchema(v any) any {
	switch v := v.(type) {
	case map[string]any:
		delete(v, "description")
		for k, sub := range v {
			if slices.Contains(schemaKeywords, k) || slices.Contains(schemaListKeyword, k) || slices.Contains(schemaMapKeywords, k) {
				v[k] = stripDescriptions(sub, k)
			}
		}
		return v
	case []any: // items as a list of schemas, in the older drafts
		for i, sub := range v {
			v[i] = stripSchema(sub)
		}
		return v
	}
	return v
}

// cloneMap is a deep copy of m.
func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = cloneValue(v)
	}
	return out
}

// cloneValue is a deep copy of a decoded JSON value.
func cloneValue(v any) any {
	switch v := v.(type) {
	case map[string]any:
		return cloneMap(v)
	case []any:
		out := make([]any, len(v))
		for i, x := range v {
			out[i] = cloneValue(x)
		}
		return out
	}
	return v
}

// equalValue reports whether two decoded JSON values are equal.
func equalValue(a, b any) bool {
	switch a := a.(type) {
	case map[string]any:
		m, ok := b.(map[string]any)
		if !ok || len(m) != len(a) {
			return false
		}
		for k, v := range a {
			w, ok := m[k]
			if !ok || !equalValue(v, w) {
				return false
			}
		}
		return true
	case []any:
		l, ok := b.([]any)
		return ok && slices.EqualFunc(a, l, equalValue)
	}
	return a == b
}
