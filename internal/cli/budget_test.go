package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	tuitranscript "github.com/maccavelli/go-tui-lib/v2/transcript"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// reprintAllocBudget is the most allocations drawing 1,000 rows of replies
// again at a new width may make: 0030-PLAN P4's measurement after its
// rework, and a quarter more (0030-MADR D5, as amended). go-tui-lib's
// reprint keeps within it (0035-PLAN deviation 21, restored at P18b).
const reprintAllocBudget = 6300

// terminalRows is how many rows of a terminal columns wide lines s takes:
// a line wider than that takes more than one, as go-tui-lib counts a
// reprint's rows (0035-MADR D14 step 5).
func terminalRows(s string, columns int) int {
	rows := 0
	for line := range strings.Lines(s) {
		rows += max(1, (ansi.StringWidth(strings.TrimSuffix(line, "\n"))+columns-1)/columns)
	}
	return rows
}

// Drawing a conversation of 1,000 rows of replies again at 60 columns, as a
// reprint does, allocates at most reprintAllocBudget times. A count, unlike
// a time, is the same on every host, so this runs with every test
// (0030-MADR D5).
func TestReprintAllocationBudget(t *testing.T) {
	reply, err := os.ReadFile("render/testdata/reply.txt")
	if err != nil {
		t.Fatal(err)
	}
	s := term.Stream{TTY: true, Color: true, Width: 100}
	v := newView(&Output{caps: term.Caps{Out: s, Err: s}}, display{}, stepClock())
	tr := newTranscript(100, tuitranscript.WithMaxRows(1000))
	keepReplies(tr, v, string(reply), 100)
	var out string
	n := testing.AllocsPerRun(10, func() { out = reprintAt(tr, 60) })
	shown := reprinted(out)
	notice, rest, _ := strings.Cut(shown, "\n")
	if rows := terminalRows(rest, 60); rows != 1000 || !strings.HasPrefix(notice, "(earlier output") {
		t.Fatalf("%d terminal rows drawn, want the notice and 1000", rows)
	}
	t.Logf("%v allocations for 1,000 rows", n)
	if n > reprintAllocBudget {
		t.Fatalf("%v allocations for 1,000 rows, more than %d", n, reprintAllocBudget)
	}
}
