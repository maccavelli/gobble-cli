package render

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/cli/term"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden")

var (
	colorOn  = term.Stream{TTY: true, Color: true, Width: 80}
	colorOff = term.Stream{TTY: false, Color: false, Width: 80}
)

// golden compares got with testdata/name.golden. ESC is stored as <ESC> so
// the files stay readable. On a mismatch the output is written to the
// test's artifact directory and its path reported.
func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	vis := strings.ReplaceAll(got, "\x1b", "<ESC>")
	if *update {
		if err := os.WriteFile(path, []byte(vis), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if string(want) != vis {
		out := filepath.Join(t.ArtifactDir(), name+".got")
		if err := os.WriteFile(out, []byte(vis), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Fatalf("%s differs from %s; output written to %s\n--- got\n%s\n--- want\n%s", name, path, out, vis, want)
	}
}
