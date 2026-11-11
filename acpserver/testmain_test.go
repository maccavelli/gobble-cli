package acpserver

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/fsx"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
)

// TestMain lets this test binary serve as the stdio MCP fixture, then sets
// the walk ceiling at its temp roots, GOTMPDIR's, where t.TempDir puts a
// directory when it is set, and os.TempDir()'s, so no host file above them
// changes a session's system prompt or skills (0053-MADR D2, A1).
func TestMain(m *testing.M) {
	mcptest.ServeStdioIfAsked()
	if err := fsx.SetWalkCeiling(os.Getenv("GOTMPDIR"), os.TempDir()); err != nil {
		fmt.Fprintf(os.Stderr, "walk ceiling: %v\n", err)
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// The walks stop at the test binary's temp roots: TestMain set the ceiling
// (0053-MADR D2, A1).
func TestWalksBounded(t *testing.T) {
	var want []string
	for _, d := range []string{os.Getenv("GOTMPDIR"), os.TempDir()} {
		if d == "" {
			continue
		}
		abs, err := filepath.Abs(d)
		if err != nil {
			t.Fatal(err)
		}
		want = append(want, abs)
	}
	if got := fsx.WalkCeiling(); !slices.Equal(got, want) {
		t.Fatalf("walk ceiling %q, want %q: TestMain must call fsx.SetWalkCeiling (0053-MADR D2)", got, want)
	}
}
