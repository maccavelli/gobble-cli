package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

// reprintAllocBudget is the most allocations drawing 1,000 rows of replies
// again at a new width may make: the count on go-tui-lib's Printer, the
// higher of a plain and a race-detector run, and a quarter more (0030-MADR
// D5, as amended by 0035-PLAN deviation 21). It is interim: 0035-PLAN's P18b
// restores 6,300 once replies are drawn by the library's own reprint.
const reprintAllocBudget = 8500

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
	tr := &transcript{}
	for range 100 {
		tr.open(v.replyAt).add(tr, string(reply))
	}
	var rows int
	n := testing.AllocsPerRun(10, func() {
		out, _ := tr.render(60, 1000)
		rows = strings.Count(out, "\n")
	})
	if rows != 1000 {
		t.Fatalf("%d rows drawn, want 1000", rows)
	}
	t.Logf("%v allocations for 1,000 rows", n)
	if n > reprintAllocBudget {
		t.Fatalf("%v allocations for 1,000 rows, more than %d", n, reprintAllocBudget)
	}
}
