package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// yamlLine finds yaml.v3's "line N: " in an error message.
var yamlLine = regexp.MustCompile(`line (\d+): `)

// Parse parses a settings file strictly (0018-MADR D3). It returns the root
// mapping, or nil with the diagnostics that make the file unusable: a
// syntax error or a tab, a duplicate key, a second document, or a root that
// is not a mapping. An empty or comments-only file is an empty mapping;
// CRLF is read as LF. (The plan's Read; renamed, since Read is the type of
// Options.Read.)
func Parse(file string, data []byte) (*yaml.Node, []Diagnostic) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	switch err := dec.Decode(&doc); {
	case errors.Is(err, io.EOF):
		return emptyMapping(), nil
	case err != nil:
		return nil, []Diagnostic{syntaxError(file, err)}
	}
	var second yaml.Node
	if err := dec.Decode(&second); !errors.Is(err, io.EOF) {
		line := second.Line
		if err != nil {
			line = lineOf(err)
		}
		return nil, []Diagnostic{errorAt(file, line, "", textSecondDocument)}
	}
	if len(doc.Content) == 0 {
		return emptyMapping(), nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, []Diagnostic{errorAt(file, root.Line, "", textNotMapping)}
	}
	if ds := duplicates(file, root); len(ds) > 0 {
		return nil, ds
	}
	return root, nil
}

func emptyMapping() *yaml.Node { return &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"} }

// syntaxError is yaml.v3's message at its line: "yaml: line N: text"
// becomes "<file>:N: text".
func syntaxError(file string, err error) Diagnostic {
	msg := strings.TrimPrefix(err.Error(), "yaml: ")
	line := lineOf(err)
	if m := yamlLine.FindStringIndex(msg); len(m) == 2 && m[0] == 0 {
		msg = msg[m[1]:]
	}
	return errorAt(file, line, "", msg)
}

func lineOf(err error) int {
	if m := yamlLine.FindStringSubmatch(err.Error()); m != nil {
		if n, convErr := strconv.Atoi(m[1]); convErr == nil {
			return n
		}
	}
	return 0
}

// duplicates finds a key a mapping repeats, at any depth, in yaml.v3's own
// words: decoding into a Node does not check them (0018-PLAN P2).
func duplicates(file string, n *yaml.Node) []Diagnostic {
	var out []Diagnostic
	var walk func(n *yaml.Node)
	walk = func(n *yaml.Node) {
		switch n.Kind {
		case yaml.MappingNode:
			seen := map[string]int{}
			for i := 0; i+1 < len(n.Content); i += 2 {
				k := n.Content[i]
				if first, ok := seen[k.Value]; ok {
					out = append(out, errorAt(file, k.Line, "", fmt.Sprintf("mapping key %q already defined at line %d", k.Value, first)))
				} else {
					seen[k.Value] = k.Line
				}
				walk(n.Content[i+1])
			}
		case yaml.SequenceNode:
			for _, c := range n.Content {
				walk(c)
			}
		}
	}
	walk(n)
	return out
}
