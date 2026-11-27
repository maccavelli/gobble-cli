package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// noticePaths are the repository paths NOTICE names: its words with a
// slash, less URLs and module paths (whose first element is a host).
func noticePaths(text string) []string {
	var paths []string
	for w := range strings.FieldsSeq(text) {
		w = strings.Trim(w, "().,;:")
		first, _, ok := strings.Cut(w, "/")
		if !ok || strings.Contains(w, "://") || strings.Contains(first, ".") {
			continue
		}
		paths = append(paths, w)
	}
	return paths
}

// M49: every file NOTICE names exists, so a licence or data file that moves
// cannot leave NOTICE pointing at nothing (0072-MADR D15).
func TestNoticeNamesExistingFiles_0072(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "NOTICE"))
	if err != nil {
		t.Fatal(err)
	}
	paths := noticePaths(string(b))
	if len(paths) < 4 {
		t.Fatalf("NOTICE names %q; want at least the four files of its models.dev and opencode credits", paths)
	}
	for _, p := range paths {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(p))); err != nil {
			t.Errorf("NOTICE names %s: %v", p, err)
		}
	}
}
