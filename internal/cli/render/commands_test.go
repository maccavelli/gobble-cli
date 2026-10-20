package render

import (
	"strings"
	"testing"
)

// helpSections is a fixed list: a short entry, one whose description
// wraps, and one too wide for the name column.
var helpSections = []CommandSection{
	{Heading: "Commands", Commands: []CommandEntry{
		{Name: "help", Description: "Show this list of commands"},
		{Name: "edit", Hint: "[text]", Description: "Write the prompt in your editor, starting from text; it is sent when the editor closes"},
	}},
	{Heading: "Agent commands", Commands: []CommandEntry{
		{Name: "permissions", Hint: "[remove N | allow|ask|deny RULE]", Description: "List the rules in force, remove rule N, or add a rule (session, project or global)"},
	}},
}

// At 60 columns: two columns, the descriptions at one column, a wrap
// indented to it, and a too-wide entry's description on the next line
// (0024-MADR D2).
func TestCommandListAligned(t *testing.T) {
	want := `Commands
  /help         Show this list of commands
  /edit [text]  Write the prompt in your editor, starting
                from text; it is sent when the editor closes

Agent commands
  /permissions [remove N | allow|ask|deny RULE]
                List the rules in force, remove rule N, or
                add a rule (session, project or global)
`
	if got := CommandList(helpSections, 60, false); got != want {
		t.Fatalf("at 60 columns:\n%s\nwant:\n%s", got, want)
	}
}

// Below 40 columns every description is on its own lines, indented four,
// and an entry wider than the terminal wraps too.
func TestCommandListNarrow(t *testing.T) {
	want := `Commands
  /help
    Show this list of commands
  /edit [text]
    Write the prompt in your
    editor, starting from text;
    it is sent when the editor
    closes

Agent commands
  /permissions [remove N |
    allow|ask|deny RULE]
    List the rules in force,
    remove rule N, or add a rule
    (session, project or global)
`
	if got := CommandList(helpSections, 32, false); got != want {
		t.Fatalf("at 32 columns:\n%s\nwant:\n%s", got, want)
	}
}

// At every width no line is wider than the terminal or ends in a space.
func TestCommandListFits(t *testing.T) {
	for _, width := range []int{100, 80, 60, 45, 40, 32, 20} {
		for i, line := range strings.Split(strings.TrimSuffix(CommandList(helpSections, width, false), "\n"), "\n") {
			if w := displayWidth(line); w > width {
				t.Errorf("width %d, line %d is %d columns: %q", width, i+1, w, line)
			}
			if strings.HasSuffix(line, " ") {
				t.Errorf("width %d, line %d ends in a space: %q", width, i+1, line)
			}
		}
	}
}

// With colour on only the command name is bold, and the columns are those
// of the plain list.
func TestCommandListColor(t *testing.T) {
	got := CommandList(helpSections, 60, true)
	if !strings.Contains(got, "  \x1b[1m/help\x1b[0m         Show this list of commands\n") {
		t.Fatalf("no bold name at the plain list's columns:\n%q", got)
	}
	if strings.Contains(got, "\x1b[1mCommands") || strings.Contains(got, "\x1b[1m[text]") {
		t.Fatalf("more than the name is bold:\n%q", got)
	}
}

// A description the user wrote cannot move the cursor or colour the
// terminal: it is sanitized before it is laid out.
func TestCommandListSanitizes(t *testing.T) {
	got := CommandList([]CommandSection{{Heading: "Agent commands", Commands: []CommandEntry{
		{Name: "tpl", Description: "Run it\x1b[2J now"},
	}}}, 60, false)
	if strings.Contains(got, "\x1b") {
		t.Fatalf("an escape reached the list: %q", got)
	}
}
