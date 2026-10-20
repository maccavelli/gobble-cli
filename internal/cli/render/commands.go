package render

import (
	"strings"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// CommandEntry is one command of a list: its name without the slash, its
// argument hint, and what it does.
type CommandEntry struct {
	Name, Hint, Description string
}

// CommandSection is a heading and the commands under it.
type CommandSection struct {
	Heading  string
	Commands []CommandEntry
}

const (
	commandIndent  = 2  // an entry under its heading
	commandGap     = 2  // between the name column and the description
	commandNameMax = 32 // the widest the name column grows
	commandNarrow  = 40 // below it, descriptions go on their own lines
	commandDeep    = 4  // a description on its own lines, and an entry's wrap
)

// CommandList lays sections out for a terminal width columns wide, as
// /help prints them (0024-MADR D2). Each heading is at column 0 and each
// entry, the command and its hint, two columns in. The descriptions start
// at one column, after the widest entry that fits the name column, and
// wrap at spaces back to it; an entry wider than that column has its
// description on the next lines. Below 40 columns every description is on
// its own lines, four columns in. No line is wider than width or ends in a
// space. With color, the command's name is bold. What the list holds is
// sanitized, since a template's description is the user's text.
func CommandList(sections []CommandSection, width int, color bool) string {
	var b strings.Builder
	nameCol := nameColumn(sections, width)
	for i, s := range sections {
		if i > 0 {
			b.WriteString("\n")
		}
		for _, line := range wrapIndented(strings.Fields(term.Sanitize(s.Heading)), width, 0, commandDeep) {
			b.WriteString(line + "\n")
		}
		for _, c := range s.Commands {
			for _, line := range commandLines(c, nameCol, width, color) {
				b.WriteString(line + "\n")
			}
		}
	}
	return b.String()
}

// entryOf is the command as typed, with its hint.
func entryOf(c CommandEntry) string {
	entry := "/" + term.Sanitize(c.Name)
	if hint := term.Sanitize(c.Hint); hint != "" {
		entry += " " + hint
	}
	return entry
}

// nameColumn is the width of the widest entry that fits the name column, at
// most 32 and at most half the room, or 0 below 40 columns, where every
// description goes on its own lines.
func nameColumn(sections []CommandSection, width int) int {
	if width < commandNarrow {
		return 0
	}
	limit := min(commandNameMax, (width-commandIndent-commandGap)/2)
	col := 0
	for _, s := range sections {
		for _, c := range s.Commands {
			if w := displayWidth(entryOf(c)); w <= limit {
				col = max(col, w)
			}
		}
	}
	return col
}

// commandLines lays out one command: its entry beside its description, or
// above it when the entry is wider than the name column.
func commandLines(c CommandEntry, nameCol, width int, color bool) []string {
	entry := entryOf(c)
	desc := strings.Fields(term.Sanitize(c.Description))
	var lines []string
	if ew := displayWidth(entry); nameCol > 0 && ew <= nameCol {
		descCol := commandIndent + nameCol + commandGap
		lines = wrapIndented(desc, width, descCol, descCol)
		head := strings.Repeat(" ", commandIndent) + entry
		if len(lines) == 0 {
			lines = []string{head}
		} else {
			lines[0] = head + strings.Repeat(" ", descCol-commandIndent-ew) + lines[0][descCol:]
		}
	} else {
		descCol := commandDeep
		if nameCol > 0 {
			descCol = commandIndent + nameCol + commandGap
		}
		lines = append(wrapIndented(strings.Fields(entry), width, commandIndent, commandDeep),
			wrapIndented(desc, width, descCol, descCol)...)
	}
	name := "/" + term.Sanitize(c.Name)
	if rest, ok := strings.CutPrefix(lines[0][commandIndent:], name); ok {
		lines[0] = lines[0][:commandIndent] + term.Bold.Wrap(name, color) + rest
	}
	return lines
}

// wrapIndented fills lines of at most width columns with words, the first
// indented first columns and the rest rest columns. A word wider than its
// line is cut. It returns no lines for no words.
func wrapIndented(words []string, width, first, rest int) []string {
	var lines []string
	var line strings.Builder
	indent, used := first, 0
	flush := func() {
		lines = append(lines, strings.Repeat(" ", indent)+line.String())
		line.Reset()
		indent, used = rest, 0
	}
	for _, w := range words {
		ww := displayWidth(w)
		if used > 0 && used+1+ww > max(width-indent, 1) {
			flush()
		}
		for used == 0 && ww > max(width-indent, 1) {
			head, tail := cutWidth(w, max(width-indent, 1))
			line.WriteString(head)
			used = displayWidth(head)
			flush()
			w, ww = tail, displayWidth(tail)
		}
		if used > 0 {
			line.WriteByte(' ')
			used++
		}
		line.WriteString(w)
		used += ww
	}
	if used > 0 {
		flush()
	}
	return lines
}

// cutWidth splits s after at most w columns, keeping at least one rune in
// head so that a cut always makes progress.
func cutWidth(s string, w int) (head, tail string) {
	used := 0
	for i, r := range s {
		rw := displayWidth(string(r))
		if used+rw > w && i > 0 {
			return s[:i], s[i:]
		}
		used += rw
	}
	return s, ""
}
