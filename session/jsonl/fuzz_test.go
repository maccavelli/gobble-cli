package jsonl

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/maccavelli/gobble-cli/session"
)

// FuzzParse reads any bytes as a session file without panicking. Every
// entry it returns has a type and is not a header, and a v1 file's
// migrated ids are all different: v2 entries keep the ids Pi wrote, as
// Pi's migration does (0005-PLAN F2-1 deviation 2). It is seeded with
// every fixture.
func FuzzParse(f *testing.F) {
	seeds, err := filepath.Glob(filepath.Join(fixtures, "*.jsonl"))
	if err != nil || len(seeds) == 0 {
		f.Fatalf("no seeds: %v", err)
	}
	for _, p := range seeds {
		b, err := os.ReadFile(p)
		if err != nil {
			f.Fatal(err)
		}
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		h, entries, skipped, err := parse(b)
		if err != nil {
			return
		}
		if skipped < 0 {
			t.Fatalf("skipped %d", skipped)
		}
		seen := map[string]bool{}
		for i, e := range entries {
			if e.Type == "" || e.Type == session.TypeHeader {
				t.Fatalf("entry %d has type %q", i, e.Type)
			}
			if max(h.Version, 1) == 1 {
				if seen[e.ID] {
					t.Fatalf("migrated id %q twice", e.ID)
				}
				seen[e.ID] = true
			}
		}
	})
}
