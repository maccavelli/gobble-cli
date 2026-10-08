package builtin

import (
	"strconv"

	"github.com/google/jsonschema-go/jsonschema"
)

// property is s's property name. A missing one is a programming error that
// the first test building the tool finds.
func property(s *jsonschema.Schema, name string) *jsonschema.Schema {
	p, ok := s.Properties[name]
	if !ok {
		panic("builtin: the schema has no property " + name)
	}
	return p
}

// bound publishes an integer property's minimum, its maximum when maximum
// is above 0, and its default (0012-MADR D2 rule 6). The tool's code
// enforces them (rule 8).
func bound(s *jsonschema.Schema, name string, minimum, maximum, def int) {
	p := property(s, name)
	p.Minimum = new(float64(minimum))
	if maximum > 0 {
		p.Maximum = new(float64(maximum))
	}
	p.Default = []byte(strconv.Itoa(def))
}

// nonEmpty publishes a string property's minLength of 1.
func nonEmpty(s *jsonschema.Schema, name string) { property(s, name).MinLength = new(1) }
