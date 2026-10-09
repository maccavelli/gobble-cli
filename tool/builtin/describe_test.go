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
var sections = []string{"## Use when", "## Returns", "## Rules", "## Example"}

// required are the sections every description has (0012-MADR D5 item 6).
var required = []string{"## Use when", "## Returns"}

// notFor is a "Not for" bullet, and the tool it sends the reader to.
var notFor = regexp.MustCompile(`^- Not for [^:]+: use ([a-z_]+)`)

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
	// Every tool ToolsWith can build has exactly one template, and every
	// template a tool: powershell's is built on Windows only (F1c-1
	// deviation 2).
	want := Names()
	if !slices.Contains(want, "powershell") {
		want = append(want, "powershell")
	}
	slices.Sort(want)
	if !slices.Equal(names, want) {
		t.Fatalf("templates %v, want one for each tool, %v", names, want)
	}
	d := describeDataFor(Options{}.withDefaults())
	var tools []string
	for _, tl := range ToolsWith(Options{}) {
		tools = append(tools, tl.Spec().Name)
	}
	for _, name := range names {
		text, err := renderDescription(name, d)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		for _, problem := range descriptionShape(text, tools) {
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

// descriptionShape lists where a rendered description breaks D5's rules,
// tools being the names of the tools a "Not for" line may send the reader to.
func descriptionShape(text string, tools []string) []string {
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
	return append(out, agentFirst(text, tools)...)
}

// agentFirst lists where a description breaks D5 item 6: its required
// sections, three bullets at least, and a "Not for" line naming a tool that
// exists.
func agentFirst(text string, tools []string) []string {
	var out []string
	lines := strings.Split(text, "\n")
	for _, h := range required {
		if !slices.Contains(lines, h) {
			out = append(out, "no "+h+" section")
		}
	}
	bullets, nots := 0, 0
	for _, line := range lines {
		if strings.HasPrefix(line, "- ") {
			bullets++
		}
		if m := notFor.FindStringSubmatch(line); m != nil {
			nots++
			if !slices.Contains(tools, m[1]) {
				out = append(out, "a Not for line names "+m[1]+", which is not a tool here")
			}
		}
	}
	if bullets < 3 {
		out = append(out, "fewer than 3 bullets")
	}
	if nots == 0 {
		out = append(out, "no Not for line")
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
