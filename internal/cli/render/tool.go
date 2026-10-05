package render

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// Tool-call statuses, as ACP names them.
const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

const (
	// maxToolLines is the most output lines shown without --verbose.
	maxToolLines = 20
	// keepToolLines is how many lines are kept at each end when cut.
	keepToolLines = 10
)

// ToolHeader is one line: the arrow glyph, the sanitised title truncated to
// fit s.Width in display columns, and a status suffix.
func ToolHeader(title, status string, s term.Stream, g term.Glyphs) string {
	suffix, style := statusSuffix(status, g)
	head := g.Arrow + " "
	room := s.Width - displayWidth(head) - displayWidth(suffix) - 1
	t := truncateWidth(strings.Join(strings.Fields(term.Sanitize(title)), " "), max(room, 1), g.Ellipsis)
	line := term.Bold.Wrap(head, s.Color) + t
	if suffix != "" {
		line += " " + style.Wrap(suffix, s.Color)
	}
	return line
}

func statusSuffix(status string, g term.Glyphs) (string, term.Style) {
	switch status {
	case StatusCompleted:
		return g.Check, term.Green
	case StatusFailed:
		return g.Cross, term.Red
	case StatusPending, StatusInProgress:
		return g.Ellipsis, term.Dim
	}
	return "", 0
}

// ToolOutput sanitises out and, unless verbose, shows only the first and
// last ten lines of output longer than twenty, with a count of the rest.
func ToolOutput(out string, verbose bool, g term.Glyphs) string {
	out = strings.TrimSuffix(term.Sanitize(out), "\n")
	if out == "" {
		return ""
	}
	lines := strings.Split(out, "\n")
	if verbose || len(lines) <= maxToolLines {
		return out + "\n"
	}
	hidden := len(lines) - 2*keepToolLines
	var b strings.Builder
	for _, l := range lines[:keepToolLines] {
		b.WriteString(l + "\n")
	}
	fmt.Fprintf(&b, "%s %d lines hidden (--verbose shows all)\n", g.Ellipsis, hidden)
	for _, l := range lines[len(lines)-keepToolLines:] {
		b.WriteString(l + "\n")
	}
	return b.String()
}

// Diff colours a unified diff: added lines green, removed lines red, hunk
// headers cyan. File headers (+++ and ---) are bold.
func Diff(diff string, s term.Stream) string {
	if !s.Color {
		return diff
	}
	var b strings.Builder
	for line := range strings.Lines(diff) {
		body, nl := strings.CutSuffix(line, "\n")
		var st term.Style
		switch {
		case strings.HasPrefix(body, "+++"), strings.HasPrefix(body, "---"):
			st = term.Bold
		case strings.HasPrefix(body, "@@"):
			st = term.Cyan
		case strings.HasPrefix(body, "+"):
			st = term.Green
		case strings.HasPrefix(body, "-"):
			st = term.Red
		}
		b.WriteString(st.Wrap(body, true))
		if nl {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// ShortPath writes path relative to home as ~/…, with forward slashes, when
// it lies under home. filepath.Rel does the comparison, so Windows
// separators and volume names work. Other paths are returned unchanged.
func ShortPath(path, home string) string {
	if home == "" || !filepath.IsAbs(path) {
		return path
	}
	rel, err := filepath.Rel(home, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	if rel == "." {
		return "~"
	}
	return "~/" + filepath.ToSlash(rel)
}
