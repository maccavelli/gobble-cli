package builtin

import (
	"embed"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/maccavelli/gobble-cli/tool"
)

//go:embed describe/*.md
var describeFS embed.FS

// describeData is what a description template may name (0012-MADR D5).
type describeData struct {
	MaxLines, MaxKB, MaxLineChars int
	DefaultTimeout, MaxTimeout    int // seconds
	TreeDepth                     int // tree's default depth
}

// describeDataFor is o's limits for the templates.
func describeDataFor(o Options) describeData {
	return describeData{
		MaxLines: maxLines, MaxKB: maxBytes / 1024, MaxLineChars: maxLineChars,
		DefaultTimeout: int(defaultTimeout / time.Second), MaxTimeout: int(o.MaxTimeout / time.Second),
		TreeDepth: defaultTreeDepth,
	}
}

// renderDescription is describe/<name>.md rendered with d, without its
// final newline.
func renderDescription(name string, d describeData) (string, error) {
	t, err := template.ParseFS(describeFS, "describe/"+name+".md")
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, d); err != nil {
		return "", err
	}
	return strings.TrimRight(b.String(), "\n"), nil
}

// descSections are the section headings D5 allows, in their order.
var descSections = []string{"## Use when", "## Returns", "## Rules", "## Example"}

// descRequired are the sections every description has (0012-MADR D5 item 6).
var descRequired = []string{"## Use when", "## Returns"}

// descNotFor is a "Not for" bullet, and the tool it sends the reader to.
var descNotFor = regexp.MustCompile(`^- Not for [^:]+: use ([a-z_]+)`)

// descMarkup is markup D5 keeps out: "<" before a letter or "/".
var descMarkup = regexp.MustCompile(`<[A-Za-z/]`)

// descriptionProblems lists where a rendered description breaks D5's rules,
// tools being the names of the tools a "Not for" line may send the reader to.
func descriptionProblems(text string, tools []string) []string {
	var out []string
	first, _, _ := strings.Cut(text, "\n\n")
	if strings.Contains(first, "\n") || len(first) > 160 || strings.HasPrefix(first, "#") || !strings.HasSuffix(first, ".") {
		out = append(out, "the first paragraph is not one sentence of at most 160 bytes ending in a full stop")
	}
	next := 0
	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(line, "#") {
			i := slices.Index(descSections, line)
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
	if descMarkup.MatchString(text) {
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
	for _, h := range descRequired {
		if !slices.Contains(lines, h) {
			out = append(out, "no "+h+" section")
		}
	}
	bullets, nots := 0, 0
	for _, line := range lines {
		if strings.HasPrefix(line, "- ") {
			bullets++
		}
		if m := descNotFor.FindStringSubmatch(line); m != nil {
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

// CheckDescription renders text, the user's description template for the
// built-in tool name, with o's limits, as the built-in's is, and checks it
// against 0012-MADR D5, a "Not for" line naming one of the tools this
// package builds (0022-MADR D15). It returns the rendered description.
func CheckDescription(name, text string, o Options) (string, error) {
	o = o.withDefaults()
	if !slices.Contains(Names(), name) && name != powershellName {
		return "", fmt.Errorf("no built-in tool is called %s", name)
	}
	t, err := template.New(name).Parse(text)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	if err := t.Execute(&b, describeDataFor(o)); err != nil {
		return "", err
	}
	rendered := strings.TrimRight(b.String(), "\n")
	tools := slices.Concat(Names(), []string{powershellName, ToolSearchName, ListResourcesName, ListResourceTemplatesName, ReadResourceName})
	if problems := descriptionProblems(rendered, tools); len(problems) > 0 {
		return "", errors.New(strings.Join(problems, "; "))
	}
	return rendered, nil
}

// redescribe gives each tool in ts the description o.Descriptions has for
// it, when that passes CheckDescription; the caller has warned of any that
// does not.
func redescribe(ts []tool.Tool, o Options) []tool.Tool {
	for i, t := range ts {
		text, ok := o.Descriptions[t.Spec().Name]
		if !ok {
			continue
		}
		if d, err := CheckDescription(t.Spec().Name, text, o); err == nil {
			ts[i] = tool.Redescribed(t, d)
		}
	}
	return ts
}

// description is a built-in tool's rendered description. A broken template
// is a programming error that the first test building the tool finds, as
// tool.New's schema panic is.
func description(name string, o Options) string {
	s, err := renderDescription(name, describeDataFor(o))
	if err != nil {
		panic(fmt.Sprintf("builtin: %s description: %v", name, err))
	}
	return s
}
