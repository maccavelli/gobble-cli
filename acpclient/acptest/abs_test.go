package acptest

import (
	"path/filepath"
	"strings"
	"testing"
)

// Abs makes a POSIX fixture path absolute on the host, keeping every
// element, so the SDK's host rule accepts it (0019-MADR D4).
func TestAbs(t *testing.T) {
	work := Abs("/work")
	if !filepath.IsAbs(work) {
		t.Fatalf("Abs(/work) = %q, not absolute on this host", work)
	}
	if !strings.HasSuffix(filepath.ToSlash(work), "/work") {
		t.Fatalf("Abs(/work) = %q, want it to end in /work", work)
	}
	nested := Abs("/work/project/a.go")
	if !filepath.IsAbs(nested) || filepath.Dir(filepath.Dir(nested)) != work ||
		!strings.HasSuffix(filepath.ToSlash(nested), "/work/project/a.go") {
		t.Fatalf("Abs(/work/project/a.go) = %q, want it under %q with every element", nested, work)
	}
}
