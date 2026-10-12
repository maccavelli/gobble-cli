package config

import (
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Override is one --set path=value (0018-MADR D10): a known key and its
// checked value, never written to a file.
type Override struct {
	Path  string
	value any
	null  bool
}

// ParseOverride parses --set's argument. The value is one YAML scalar or
// flow value, checked as a file value is; an empty value is the empty
// string. Its errors are usage errors.
func ParseOverride(arg string) (Override, error) {
	path, text, ok := strings.Cut(arg, "=")
	if !ok || path == "" {
		return Override{}, fmt.Errorf("--set %s: want PATH=VALUE", arg)
	}
	if isCredentialPath(path) {
		return Override{}, fmt.Errorf("--set %s: %s", path, textCredential)
	}
	k, known := KeyOf(path)
	if !known {
		return Override{}, fmt.Errorf("unknown setting %q", path)
	}
	if text == "" {
		if k.Kind == KindInt || k.Kind == KindBool {
			return Override{}, fmt.Errorf("--set %s: want %s, have an empty value", path, want(k))
		}
		return Override{Path: path, value: ""}, nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(text), &doc); err != nil || len(doc.Content) == 0 {
		return Override{}, errors.Join(fmt.Errorf("--set %s: the value is not YAML", path), err)
	}
	n := doc.Content[0]
	if isNull(n) {
		return Override{Path: path, null: true}, nil
	}
	v, d := convert(k, n)
	if d != "" {
		return Override{}, fmt.Errorf("--set %s: %s", path, d)
	}
	return Override{Path: path, value: v}, nil
}

func want(k Key) string {
	switch k.Kind {
	case KindInt:
		return "an integer"
	case KindBool:
		return "a boolean"
	}
	return "a string"
}
