package builtin

import (
	"encoding/json/v2"
	"io/fs"
	"path"
	"slices"
	"strings"
	"testing"
	"time"
)

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
	// Every tool ToolsWith can build has exactly one template, and every
	// template a tool: powershell's is built on Windows only (F1c-1
	// deviation 2).
	want := catalogNames()
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Fatalf("templates %v, want one for each tool, %v", names, want)
	}
	d := describeDataFor(Options{}.withDefaults())
	tools := catalogNames()
	for _, name := range names {
		text, err := renderDescription(name, d)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, problem := range descriptionProblems(text, tools) {
			t.Errorf("%s: %s", name, problem)
		}
	}
	for _, tl := range catalog(Options{}) {
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

// A template's limits come from the options, so a raised MaxTimeout is
// reported truly.
func TestDescriptionUsesOptions(t *testing.T) {
	if got := description("bash", Options{MaxTimeout: 20 * time.Minute}.withDefaults()); !strings.Contains(got, "at most 1200") {
		t.Fatalf("bash's description with a 20-minute maximum says %q", got)
	}
}
