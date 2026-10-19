package builtin

import (
	"strings"
	"testing"
	"time"
)

// CheckDescription renders the user's template with the options' limits
// and holds it to 0012-MADR D5, for a built-in tool only; ToolsWith takes a
// description that passes, and keeps the built-in's for one that does not
// (0022-MADR D15).
func TestCheckDescription(t *testing.T) {
	const good = "Read a text file's lines, a page at a time.\n\n## Use when\n- You need a file's contents.\n" +
		"- Not for searching many files: use grep.\n\n## Returns\n- At most {{.MaxLines}} lines; a {{.MaxTimeout}} s limit.\n"
	got, err := CheckDescription("read", good, Options{MaxTimeout: 90 * time.Second})
	if err != nil || !strings.Contains(got, "At most 2000 lines; a 90 s limit.") || strings.HasSuffix(got, "\n") {
		t.Fatalf("%q, %v", got, err)
	}
	for _, c := range []struct{ name, tool, text, why string }{
		{"not a tool", "teleport", good, "no built-in tool is called teleport"},
		{"no Returns", "read", "Read a file.\n\n## Use when\n- a\n- b\n- Not for x: use grep.\n", "no ## Returns section"},
		{"two bullets", "read", "Read a file.\n\n## Use when\n- Not for x: use grep.\n\n## Returns\n- b\n", "fewer than 3 bullets"},
		{"an unknown tool", "read", strings.Replace(good, "use grep", "use teleport", 1), "names teleport"},
		{"a bad template", "read", good + "{{.Nope}}", "Nope"},
		{"unparsed", "read", good + "{{", "unclosed action"},
		{"over budget", "read", good + "\n## Rules\n- " + strings.Repeat("Long. ", 200) + "\n", "over 1200 bytes"},
	} {
		if _, err := CheckDescription(c.tool, c.text, Options{}); err == nil || !strings.Contains(err.Error(), c.why) {
			t.Errorf("%s: %v, want %q", c.name, err, c.why)
		}
	}
	ts := ToolsWith(Options{Descriptions: map[string]string{"read": good, "grep": "Search."}})
	for _, tl := range ts {
		s := tl.Spec()
		switch s.Name {
		case "read":
			if !strings.HasPrefix(s.Description, "Read a text file's lines") {
				t.Errorf("read's description %q", s.Description)
			}
		case "grep":
			if s.Description != description("grep", Options{}.withDefaults()) {
				t.Errorf("grep took a description that fails: %q", s.Description)
			}
		}
	}
}
