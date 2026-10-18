package config

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
)

// globalOnlyKeyword marks a key a project's file may not set.
const globalOnlyKeyword = "x-gobble-global-only"

// Schema is the JSON Schema of settings.yaml, generated from Settings and
// annotated from Keys: each key's description, default, enum, least value
// and global-only mark (0018-MADR D12). YAML editors read it as JSON Schema.
// It is indented, with a final newline, as testdata/settings.schema.json
// holds it.
func Schema() ([]byte, error) {
	s, err := jsonschema.For[Settings](nil)
	if err != nil {
		return nil, fmt.Errorf("config: schema: %w", err)
	}
	s.Schema = "https://json-schema.org/draft/2020-12/schema"
	s.Title = "gobble settings.yaml"
	s.Description = "gobble's settings. Every key is optional; a key left out takes its default."
	for _, k := range Keys {
		p, err := propertyAt(s, k.Path)
		if err != nil {
			return nil, err
		}
		p.Description = k.Doc
		def, err := json.Marshal(defaultValue(k))
		if err != nil {
			return nil, err
		}
		p.Default = def
		if k.Kind == KindEnum {
			p.Enum = make([]any, len(k.Enum))
			for i, e := range k.Enum {
				p.Enum[i] = e
			}
		}
		if k.Kind == KindInt {
			p.Minimum = new(float64(k.Min))
			if k.Max != 0 {
				p.Maximum = new(float64(k.Max))
			}
		}
		if k.Scope == ScopeGlobalOnly {
			if p.Extra == nil {
				p.Extra = map[string]any{}
			}
			p.Extra[globalOnlyKeyword] = true
		}
	}
	b, err := json.Marshal(s, json.Deterministic(true), jsontext.WithIndent("  "))
	if err != nil {
		return nil, fmt.Errorf("config: schema: %w", err)
	}
	return append(b, '\n'), nil
}

// propertyAt is the schema of the key at a dotted path.
func propertyAt(s *jsonschema.Schema, path string) (*jsonschema.Schema, error) {
	cur := s
	for part := range strings.SplitSeq(path, ".") {
		next, ok := cur.Properties[part]
		if !ok {
			return nil, fmt.Errorf("config: schema: Settings has no %s (at %s)", path, part)
		}
		cur = next
	}
	return cur, nil
}
