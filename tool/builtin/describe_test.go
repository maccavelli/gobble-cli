package builtin

import (
	"encoding/json/v2"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

// The section headings D5 allows, in their order.
var sections = []string{"## Use when", "## Rules", "## Example"}

// html is markup D5 keeps out: "<" before a letter or "/".
var html = regexp.MustCompile(`<[A-Za-z/]`)

// Every description template has 0012-MADR D5's shape and fits its budget,
// and every built-in tool's description is its rendered template.
func TestDescriptionShape(t *testing.T) {
	files, err := fs.Glob(describeFS, "describe/*.md")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, f := range files {
		names = append(names, strings.TrimSuffix(path.Base(f), ".md"))
	}
	if want := []string{"bash", "edit", "powershell", "read", "write"}; !slices.Equal(names, want) {
		t.Fatalf("templates %v, want %v", names, want)
	}
	d := describeDataFor(Options{}.withDefaults())
	for _, name := range names {
		text, err := renderDescription(name, d)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, problem := range descriptionShape(text) {
			t.Errorf("%s: %s", name, problem)
		}
	}
	for _, tl := range ToolsWith(Options{}) {
		s := tl.Spec()
		if want := description(s.Name, Options{}.withDefaults()); s.Description != want {
			t.Errorf("%s: the description is not its rendered template", s.Name)
		}
		def, err := json.Marshal(map[string]any{"name": s.Name, "description": s.Description, "input_schema": s.InputSchema})
		if err != nil {
			t.Fatal(err)
		}
		if len(def) > 2000 {
			t.Errorf("%s: the definition is %d bytes, over 2000", s.Name, len(def))
		}
	}
}

// descriptionShape lists where a rendered description breaks D5's rules.
func descriptionShape(text string) []string {
	var out []string
	first, _, _ := strings.Cut(text, "\n\n")
	if strings.Contains(first, "\n") || len(first) > 160 || strings.HasPrefix(first, "#") || !strings.HasSuffix(first, ".") {
		out = append(out, "the first paragraph is not one sentence of at most 160 bytes ending in a full stop")
	}
	next := 0
	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(line, "#") {
			i := slices.Index(sections, line)
			if i < next {
				out = append(out, "the heading "+line+" is not allowed, repeats, or is out of order")
			} else {
				next = i + 1
			}
		}
		if strings.HasPrefix(line, "|") || strings.HasPrefix(line, "* ") {
			out = append(out, "a table row or an asterisk list item: "+line)
		}
	}
	if html.MatchString(text) {
		out = append(out, "markup: <")
	}
	if len(text) > 1200 {
		out = append(out, "over 1200 bytes")
	}
	return out
}

// A template's limits come from the options, so a raised MaxTimeout is
// reported truly.
func TestDescriptionUsesOptions(t *testing.T) {
	if got := description("bash", Options{MaxTimeout: 20 * time.Minute}.withDefaults()); !strings.Contains(got, "at most 1200") {
		t.Fatalf("bash's description with a 20-minute maximum says %q", got)
	}
}
