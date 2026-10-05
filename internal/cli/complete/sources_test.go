package complete

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func collect(t *testing.T, c Completer, prefix string) ([]Candidate, Directive) {
	t.Helper()
	seq, d := c.Complete(t.Context(), prefix)
	return slices.Collect(seq), d
}

func valuesOf(cs []Candidate) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.Value
	}
	return out
}

func pathFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, f := range []string{"alpha.txt", "Beta.md", "dés.txt", ".hidden"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range []string{"sub", ".git"} {
		if err := os.Mkdir(filepath.Join(dir, d), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return dir + string(filepath.Separator)
}

func TestPathSource(t *testing.T) {
	dir := pathFixture(t)
	sep := string(filepath.Separator)
	cs, d := collect(t, PathSource(false), dir)
	want := []string{dir + "alpha.txt", dir + "Beta.md", dir + "dés.txt", dir + "sub" + sep}
	if !sameSet(valuesOf(cs), want) {
		t.Fatalf("files = %q, want %q", valuesOf(cs), want)
	}
	if d&DirectiveNoSpace == 0 {
		t.Fatal("paths must not get a trailing space")
	}
	for _, c := range cs {
		wantKind := KindFile
		if strings.HasSuffix(c.Value, sep) {
			wantKind = KindDir
		}
		if c.Kind != wantKind {
			t.Errorf("%s: kind %v, want %v", c.Value, c.Kind, wantKind)
		}
	}

	if cs, _ := collect(t, PathSource(false), dir+"dé"); !slices.Equal(valuesOf(cs), []string{dir + "dés.txt"}) {
		t.Fatalf("multibyte prefix: %q", valuesOf(cs))
	}
	if cs, _ := collect(t, PathSource(false), dir+".g"); !slices.Equal(valuesOf(cs), []string{dir + ".git" + sep}) {
		t.Fatalf("a dot prefix shows hidden entries: %q", valuesOf(cs))
	}
	cs, d = collect(t, PathSource(true), dir)
	if !slices.Equal(valuesOf(cs), []string{dir + "sub" + sep}) || d&DirectiveFilterDirs == 0 {
		t.Fatalf("dirs only = %q, directive %d", valuesOf(cs), d)
	}
	cs, _ = collect(t, PathSource(false), dir+"beta")
	if runtime.GOOS == "windows" {
		if !slices.Equal(valuesOf(cs), []string{dir + "Beta.md"}) {
			t.Fatalf("Windows matches names case-insensitively: %q", valuesOf(cs))
		}
	} else if len(cs) != 0 {
		t.Fatalf("case-sensitive match expected: %q", valuesOf(cs))
	}
	if cs, _ := collect(t, PathSource(false), filepath.Join(dir, "missing")+sep); len(cs) != 0 {
		t.Fatalf("a missing directory gave %q", valuesOf(cs))
	}
}

func TestPathSourceSlashAndHome(t *testing.T) {
	dir := pathFixture(t)
	home := strings.TrimSuffix(dir, string(filepath.Separator))
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cs, _ := collect(t, PathSource(false), "~/al")
	if !slices.Equal(valuesOf(cs), []string{"~/alpha.txt"}) {
		t.Fatalf("~ prefix = %q", valuesOf(cs))
	}
	cs, _ = collect(t, PathSource(true), "~/")
	if !slices.Equal(valuesOf(cs), []string{"~/sub/"}) {
		t.Fatalf("a typed / is kept as the separator: %q", valuesOf(cs))
	}
}

func TestPathSourceStopsAt200(t *testing.T) {
	dir := t.TempDir()
	for i := range 250 {
		if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%03d", i)), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if cs, _ := collect(t, PathSource(false), dir+string(filepath.Separator)); len(cs) != maxPathCandidates {
		t.Fatalf("got %d candidates, want %d", len(cs), maxPathCandidates)
	}
}

func TestEnumAndModelSources(t *testing.T) {
	cs, d := collect(t, EnumSource([]string{"", "off", "low"}), "")
	if !slices.Equal(valuesOf(cs), []string{"off", "low"}) || d&DirectiveKeepOrder == 0 {
		t.Fatalf("enum = %q, %d", valuesOf(cs), d)
	}
	m := ModelSource(listOf("a/x", "b/y"), []string{"low", "high"})
	if cs, d := collect(t, m, "a/"); !slices.Equal(valuesOf(cs), []string{"a/x", "b/y"}) || d&DirectiveNoSpace == 0 {
		t.Fatalf("models = %q, %d (no space: a ':' level may follow)", valuesOf(cs), d)
	}
	if cs, _ := collect(t, m, "a/x:"); !slices.Equal(valuesOf(cs), []string{"a/x:low", "a/x:high"}) {
		t.Fatalf("levels = %q", valuesOf(cs))
	}
	if cs, _ := collect(t, ListSource(NoCandidates, true), ""); len(cs) != 0 {
		t.Fatalf("NoCandidates gave %q", valuesOf(cs))
	}
}
