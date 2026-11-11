package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/fsx"
)

// TestMain sets the walk ceiling at the test binary's temp roots,
// GOTMPDIR's, where t.TempDir puts a directory when it is set, and
// os.TempDir()'s, so no host file above them changes a test (0053-MADR D2,
// A1).
func TestMain(m *testing.M) {
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
