package fsx

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// repo makes a repository at base/repo: a .git directory, the files named,
// and returns its path. A name ending in "/" is a directory.
func repo(t *testing.T, base string, files map[string]string) string {
	t.Helper()
	top := filepath.Join(base, "repo")
	mkdir(t, filepath.Join(top, ".git", "info"))
	for name, text := range files {
		p := filepath.Join(top, filepath.FromSlash(name))
		if name[len(name)-1] == '/' {
			mkdir(t, p)
			continue
		}
		mkdir(t, filepath.Dir(p))
		write(t, p, text)
	}
	return top
}

// walked is every path Walk visits under dir, a "/" after directories.
func walked(t *testing.T, ws Workspace, dir string) []string {
	t.Helper()
	k, err := ws.Walker(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close() //nolint:errcheck // test
	var got []string
	err = k.Walk(func(rel string, d fs.DirEntry) error {
		if d.IsDir() {
			rel += "/"
		}
		got = append(got, rel)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

// Byte order; .git and node_modules skipped at every depth; hidden kept;
// .gitignore at each level and info/exclude applied.
func TestWalk(t *testing.T) {
	top := repo(t, t.TempDir(), map[string]string{
		".gitignore":          "*.log\nbuild/\n",
		".git/info/exclude":   "private.txt\n",
		".hidden":             "",
		"B.txt":               "",
		"a.txt":               "",
		"a.log":               "",
		"private.txt":         "",
		"build/out.bin":       "",
		"node_modules/x/y.js": "",
		"sub/.gitignore":      "!keep.log\n*.tmp\n",
		"sub/keep.log":        "",
		"sub/drop.log":        "",
		"sub/x.tmp":           "",
		"sub/node_modules/z":  "",
		"sub/.git":            "a submodule's .git file",
		"z.tmp":               "",
	})
	got := walked(t, Workspace{Roots: []string{top}}, top)
	want := []string{".gitignore", ".hidden", "B.txt", "a.txt", "sub/", "sub/.gitignore", "sub/keep.log", "z.tmp"}
	if !slices.Equal(got, want) {
		t.Fatalf("walked %q, want %q", got, want)
	}
}

// The one read outside the roots: a .gitignore above the workspace root, up
// to the repository's top level, filters the walk (0005-PLAN, entry "F1c
// made executable").
func TestWalkReadsAncestorIgnoreOutsideRoot(t *testing.T) {
	top := repo(t, t.TempDir(), map[string]string{
		".gitignore":      "*.log\n",
		"work/a.log":      "",
		"work/a.txt":      "",
		"work/deep/b.log": "",
		"work/deep/b.txt": "",
		"work/.gitignore": "deep/b.txt\n",
	})
	root := filepath.Join(top, "work")
	got := walked(t, Workspace{Roots: []string{root}}, root)
	want := []string{".gitignore", "a.txt", "deep/"}
	if !slices.Equal(got, want) {
		t.Fatalf("walked %q, want %q", got, want)
	}
}

// Without a repository, the .gitignore files under the walk's directory
// still apply.
func TestWalkWithoutRepository(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, ".gitignore"), "*.log\n")
	write(t, filepath.Join(dir, "a.log"), "")
	write(t, filepath.Join(dir, "a.txt"), "")
	if got, want := walked(t, Workspace{Roots: []string{dir}}, dir), []string{".gitignore", "a.txt"}; !slices.Equal(got, want) {
		t.Fatalf("walked %q, want %q", got, want)
	}
}

// fs.SkipDir skips a directory's contents, or the rest of a file's
// directory; fs.SkipAll ends the walk.
func TestWalkSkips(t *testing.T) {
	top := repo(t, t.TempDir(), map[string]string{"a/1": "", "a/2": "", "b/1": "", "b/2": "", "c": ""})
	k, err := Workspace{Roots: []string{top}}.Walker(top)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close() //nolint:errcheck // test
	var got []string
	err = k.Walk(func(rel string, d fs.DirEntry) error {
		got = append(got, rel)
		switch rel {
		case "a":
			return fs.SkipDir
		case "b/1":
			return fs.SkipDir
		case "c":
			return fs.SkipAll
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"a", "b", "b/1", "c"}; !slices.Equal(got, want) {
		t.Fatalf("visited %q, want %q", got, want)
	}
}

// A symbolic link to a directory is not followed.
func TestWalkDoesNotFollowLinks(t *testing.T) {
	base := t.TempDir()
	top := repo(t, base, map[string]string{"real/f": ""})
	if err := os.Symlink(filepath.Join(top, "real"), filepath.Join(top, "link")); err != nil {
		t.Skipf("cannot create a symlink here: %v", err)
	}
	got := walked(t, Workspace{Roots: []string{top}}, top)
	if want := []string{"link", "real/", "real/f"}; !slices.Equal(got, want) {
		t.Fatalf("walked %q, want %q", got, want)
	}
}

// A directory outside the roots is refused unless the call was approved.
func TestWalkerConfined(t *testing.T) {
	base := t.TempDir()
	root, other := filepath.Join(base, "root"), filepath.Join(base, "other")
	mkdir(t, root)
	mkdir(t, other)
	if _, err := (Workspace{Roots: []string{root}}).Walker(other); !errors.Is(err, ErrOutside) {
		t.Fatalf("an outside directory gave %v, want ErrOutside", err)
	}
	k, err := Workspace{Roots: []string{root}, AllowOutside: true}.Walker(other)
	if err != nil {
		t.Fatalf("an approved outside directory gave %v", err)
	}
	k.Close() //nolint:errcheck,gosec // test
}

// Ignored answers for one path, with the rules read so far and the
// directories above it.
func TestWalkerIgnored(t *testing.T) {
	top := repo(t, t.TempDir(), map[string]string{".gitignore": "*.log\n", "sub/.gitignore": "*.tmp\n", "sub/x": ""})
	k, err := Workspace{Roots: []string{top}}.Walker(top)
	if err != nil {
		t.Fatal(err)
	}
	defer k.Close() //nolint:errcheck // test
	for _, c := range []struct {
		rel  string
		want bool
	}{{"a.log", true}, {"sub/a.tmp", true}, {"a.tmp", false}, {"node_modules/x", true}, {".git/config", true}, {"sub/x", false}} {
		if got := k.Ignored(c.rel, false); got != c.want {
			t.Errorf("Ignored(%q) = %v, want %v", c.rel, got, c.want)
		}
	}
}
