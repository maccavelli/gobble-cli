package config

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Render is `gobble config get`'s YAML (0018-MADR D12): the effective
// settings, or those under path, in the template's order; a key's path
// prints its value alone. With origin, each value carries a comment naming
// where it came from, and the document ends with the layers read but not
// applied, and why. internal/cli prints it without importing yaml.v3.
func Render(r Resolved, path string, origin bool) []byte {
	if k, ok := KeyOf(path); ok {
		v := valueOf(r, k, origin)
		return encode(v)
	}
	root := &yaml.Node{Kind: yaml.MappingNode}
	for _, k := range Keys {
		if path != "" && !strings.HasPrefix(k.Path, path+".") {
			continue
		}
		m := root
		segs := strings.Split(k.Path, ".")
		for _, seg := range segs[:len(segs)-1] {
			m = child(m, seg)
		}
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: segs[len(segs)-1]}, valueOf(r, k, origin))
	}
	if origin {
		root.FootComment = disabledComment(r.Disabled)
	}
	return encode(root)
}

// child is m's mapping under key, added when m has none.
func child(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	c := &yaml.Node{Kind: yaml.MappingNode}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, c)
	return c
}

// valueOf is k's effective value as a node, with its origin as a comment.
func valueOf(r Resolved, k Key, origin bool) *yaml.Node {
	v, _ := r.Settings.Value(k.Path)
	if rules, ok := v.([]PermissionRule); ok {
		return rulesNode(rules, origin)
	}
	n := scalarNode(v)
	if origin {
		n.LineComment = Origin(r.Origins[k.Path])
	}
	return n
}

// Origin is how get --origin names a value's source: "default", "from
// --set", or "from <file>:<line> (<layer>)".
func Origin(s Source) string {
	switch {
	case s.Layer == LayerDefault:
		return "default"
	case s.File == "" || s.Layer == LayerSession:
		return "from " + s.Layer.String()
	}
	return fmt.Sprintf("from %s:%d (%s)", s.File, s.Line, s.Layer)
}

func disabledComment(ds []DisabledLayer) string {
	var b strings.Builder
	for _, d := range ds {
		fmt.Fprintf(&b, "not applied: %s: %s\n", d.File, d.Reason)
		if len(d.Keys) > 0 {
			fmt.Fprintf(&b, "  it would set: %s\n", strings.Join(d.Keys, ", "))
		}
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func encode(n *yaml.Node) []byte {
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(n); err != nil {
		panic(err) // a node built from Keys always encodes
	}
	if err := enc.Close(); err != nil {
		panic(err)
	}
	return b.Bytes()
}
