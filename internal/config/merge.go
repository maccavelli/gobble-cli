package config

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Layer is where a value came from, lowest precedence first (0018-MADR D2).
type Layer int

// The layers.
const (
	LayerDefault Layer = iota
	LayerGlobal
	LayerProject
	LayerSession // --set
)

func (l Layer) String() string {
	switch l {
	case LayerGlobal:
		return "global"
	case LayerProject:
		return "project"
	case LayerSession:
		return "--set"
	}
	return "default"
}

// Source is a value's file, line and layer.
type Source struct {
	Layer Layer
	File  string
	Line  int
}

// leaf is one key's value in one layer. A null leaf clears the key back to
// the layer below (0018-MADR D4).
type leaf struct {
	value any
	null  bool
	src   Source
}

// leaves walks a layer's root mapping against Keys and returns each valid
// key's value, with diagnostics for the rest (0018-MADR D7): a wrong type
// or rule falls back to the layer below, an unknown key is ignored and
// kept, a global-only key in a project is ignored, a credential is refused.
func leaves(file string, root *yaml.Node, layer Layer) (map[string]leaf, []Diagnostic) {
	out := map[string]leaf{}
	var ds []Diagnostic
	var walk func(m *yaml.Node, prefix string)
	walk = func(m *yaml.Node, prefix string) {
		for i := 0; i+1 < len(m.Content); i += 2 {
			k, v := m.Content[i], resolveAlias(m.Content[i+1])
			path := prefix + k.Value
			src := Source{Layer: layer, File: file, Line: k.Line}
			key, known := KeyOf(path)
			switch {
			case isCredentialPath(path):
				ds = append(ds, errorAt(file, k.Line, path, textCredential))
			case known && layer == LayerProject && key.Scope == ScopeGlobalOnly:
				ds = append(ds, warningAt(file, k.Line, path, textGlobalOnly))
			case known:
				if isNull(v) {
					out[path] = leaf{null: true, src: src}
					continue
				}
				if key.Kind == KindRules {
					rules, rds := convertRules(file, v, layer)
					ds = append(ds, rds...)
					out[path] = leaf{value: rules, src: src}
					continue
				}
				val, d := convert(key, v)
				if d != "" {
					ds = append(ds, errorAt(file, v.Line, path, d))
					continue
				}
				out[path] = leaf{value: val, src: src}
			case isGroup(path) && isNull(v):
				// A null group clears every key under it to the layer below.
			case isGroup(path) && v.Kind == yaml.MappingNode:
				walk(v, path+".")
			case isGroup(path):
				ds = append(ds, errorAt(file, v.Line, path, "want a mapping, have "+kindOf(v)))
			default:
				ds = append(ds, warningAt(file, k.Line, path, textUnknown))
			}
		}
	}
	walk(root, "")
	return out, ds
}

func resolveAlias(n *yaml.Node) *yaml.Node {
	for n.Kind == yaml.AliasNode && n.Alias != nil {
		n = n.Alias
	}
	return n
}

func isNull(n *yaml.Node) bool { return n.Kind == yaml.ScalarNode && n.ShortTag() == "!!null" }

// kindOf names a node's YAML kind for D7's "have …".
func kindOf(n *yaml.Node) string {
	switch n.Kind {
	case yaml.MappingNode:
		return "a mapping"
	case yaml.SequenceNode:
		return "a sequence"
	}
	switch n.ShortTag() {
	case "!!int":
		return "an integer"
	case "!!float":
		return "a number"
	case "!!bool":
		return "a boolean"
	case "!!null":
		return "null"
	}
	return "a string"
}

// convert checks a non-null node against its key and returns its Go value,
// or D7's text for why it cannot be used.
func convert(k Key, n *yaml.Node) (any, string) {
	switch k.Kind {
	case KindRules:
		rules, ds := convertRules("", n, LayerSession)
		if len(ds) > 0 {
			return nil, ds[0].Text
		}
		return rules, ""
	case KindBool:
		var b bool
		if n.Kind != yaml.ScalarNode || n.Decode(&b) != nil {
			return nil, "want a boolean, have " + kindOf(n)
		}
		return b, ""
	case KindInt:
		var v int64
		// A float such as 1.10 decodes into an int64 as 1, so the tag
		// decides (0018-PLAN P2).
		if n.Kind != yaml.ScalarNode || n.ShortTag() != "!!int" || n.Decode(&v) != nil {
			return nil, "want an integer, have " + kindOf(n)
		}
		if v < k.Min {
			return nil, fmt.Sprintf("must be at least %d", k.Min)
		}
		if k.Max != 0 && v > k.Max {
			return nil, fmt.Sprintf("must be at most %d", k.Max)
		}
		return v, ""
	}
	if n.Kind != yaml.ScalarNode {
		return nil, "want a string, have " + kindOf(n)
	}
	s := n.Value
	switch {
	case k.Kind == KindEnum && !slices.Contains(k.Enum, s):
		return nil, "must be one of " + enumList(k.Enum)
	case k.Path == pathSessionDir && s != "" && !isAbsOrHome(s):
		return nil, "must be an absolute path, or start with ~"
	case k.Path == pathSmallModel && s != "" && !smallModelValid(s):
		return nil, "must be provider/id, with one of " + strings.Join(modelProviders(), ", ")
	}
	return s, ""
}

// enumList is an enum's values for a message, the empty one as "".
func enumList(values []string) string {
	shown := make([]string, len(values))
	for i, v := range values {
		shown[i] = cmp.Or(v, `""`)
	}
	return strings.Join(shown, ", ")
}

// modelProviders are the providers a model setting may name:
// defaultProvider's, without its empty value.
func modelProviders() []string {
	k, _ := KeyOf("defaultProvider")
	return slices.DeleteFunc(slices.Clone(k.Enum), func(p string) bool { return p == "" })
}

// smallModelValid reports whether s is provider/id, its provider one of
// modelProviders and its id not empty (0005-PLAN F5-5 decision 1).
func smallModelValid(s string) bool {
	p, id, ok := strings.Cut(s, "/")
	return ok && strings.TrimSpace(id) != "" && slices.Contains(modelProviders(), p)
}

// isAbsOrHome reports whether p is absolute, or starts with ~ for the home
// directory.
func isAbsOrHome(p string) bool {
	return filepath.IsAbs(p) || p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`)
}
