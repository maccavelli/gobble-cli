// Package frontmatter reads the YAML frontmatter of skills and prompt
// templates, as Pi reads it (coding-agent/src/utils/frontmatter.ts). It is
// the one package that imports go.yaml.in/yaml/v3 (0005-MADR, amendment of
// 2026-10-10, item 7). IsSpace and Trim are JavaScript's whitespace, for
// the rest of what Pi reads in those files.
package frontmatter

import (
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
)

// IsSpace is JavaScript's whitespace, its \s and what its trim removes:
// Go's unicode.IsSpace with U+FEFF and without U+0085 (0005-PLAN F3-3
// deviation 4). Pi's files are read with it, so a skill or template trims
// as Pi trims it.
func IsSpace(r rune) bool {
	return r == '\ufeff' || r != '\u0085' && unicode.IsSpace(r)
}

// Trim is JavaScript's String.prototype.trim.
func Trim(s string) string { return strings.TrimFunc(s, IsSpace) }

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
	front, body := text[min(4, end):end], Trim(text[end+4:])
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
