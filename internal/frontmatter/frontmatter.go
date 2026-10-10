// Package frontmatter reads the YAML frontmatter of skills and prompt
// templates, as Pi reads it (coding-agent/src/utils/frontmatter.ts). It is
// the one package that imports go.yaml.in/yaml/v3 (0005-MADR, amendment of
// 2026-10-10, item 7).
package frontmatter

import (
	"strings"

	"go.yaml.in/yaml/v3"
)

// Parse splits text into its frontmatter and its body, as Pi's
// parseFrontmatter does:
//   - a byte-order mark is dropped and line endings become "\n";
//   - frontmatter is present only when the text starts with "---" and a
//     later line starts "---"; its YAML lies between them;
//   - the body is what follows, trimmed, or the whole text when there is
//     no frontmatter;
//   - empty frontmatter, or YAML that is not a mapping, gives an empty map.
//
// A YAML error is returned with the body.
func Parse(text string) (map[string]any, string, error) {
	text = strings.TrimPrefix(text, "\ufeff")
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	fields := map[string]any{}
	if !strings.HasPrefix(text, "---") {
		return fields, text, nil
	}
	end := strings.Index(text[3:], "\n---")
	if end < 0 {
		return fields, text, nil
	}
	end += 3
	front, body := text[min(4, end):end], strings.TrimSpace(text[end+4:])
	if front == "" {
		return fields, body, nil
	}
	var v any
	if err := yaml.Unmarshal([]byte(front), &v); err != nil {
		return fields, body, err
	}
	if m, ok := v.(map[string]any); ok {
		fields = m
	}
	return fields, body, nil
}
