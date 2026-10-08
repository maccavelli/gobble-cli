package builtin

import (
	"embed"
	"fmt"
	"strings"
	"text/template"
	"time"
)

//go:embed describe/*.md
var describeFS embed.FS

// describeData is what a description template may name (0012-MADR D5).
type describeData struct {
	MaxLines, MaxKB, MaxLineChars int
	DefaultTimeout, MaxTimeout    int // seconds
}

// describeDataFor is o's limits for the templates.
func describeDataFor(o Options) describeData {
	return describeData{
		MaxLines: maxLines, MaxKB: maxBytes / 1024, MaxLineChars: maxLineChars,
		DefaultTimeout: int(defaultTimeout / time.Second), MaxTimeout: int(o.MaxTimeout / time.Second),
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
