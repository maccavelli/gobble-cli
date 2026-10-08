package fsx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzResolve: whatever path a model gives, a read the workspace allows
// never returns the secret beside the root, reachable only by .. or by the
// symlink in the root.
func FuzzResolve(f *testing.F) {
	for _, seed := range []string{
		"in.txt", "../elsewhere/secret.txt", "link/secret.txt", "./link/../link/secret.txt",
		"@link/secret.txt", "a/../../elsewhere/secret.txt", "~", "file:///etc/passwd", "/c/x",
		"link\u00A0/secret.txt", "LINK/secret.txt",
	} {
		f.Add(seed)
	}
	base := f.TempDir()
	root, outside := filepath.Join(base, "work"), filepath.Join(base, "elsewhere")
	for _, d := range []string{root, outside} {
		if err := os.MkdirAll(d, 0o750); err != nil {
			f.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "in.txt"), []byte("inside"), 0o600); err != nil {
		f.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("SECRET"), 0o600); err != nil {
		f.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		f.Skipf("cannot create a symlink here: %v", err)
	}
	w := Workspace{Roots: []string{root}}
	f.Fuzz(func(t *testing.T, path string) {
		if strings.ContainsRune(path, 0) {
			return
		}
		abs := Resolve(root, path)
		b, err := w.ReadFile(abs)
		if err == nil && strings.Contains(string(b), "SECRET") {
			t.Fatalf("%q resolved to %s, and the workspace read the secret", path, abs)
		}
	})
}

// FuzzOutside: on a tree without symlinks, Outside agrees with a lexical
// containment check.
func FuzzOutside(f *testing.F) {
	for _, seed := range []string{"a", "../a", "a/../../b", ".", "..", "a/./b", "x/../x/../../y"} {
		f.Add(seed)
	}
	root := filepath.Join(f.TempDir(), "work")
	if err := os.MkdirAll(root, 0o750); err != nil {
		f.Fatal(err)
	}
	w := Workspace{Roots: []string{root}}
	f.Fuzz(func(t *testing.T, rel string) {
		if strings.ContainsRune(rel, 0) || filepath.IsAbs(rel) || filepath.VolumeName(rel) != "" {
			return
		}
		abs := filepath.Join(root, rel)
		r, err := filepath.Rel(root, abs)
		lexical := err == nil && (r == "." || filepath.IsLocal(r))
		if got := !w.Outside(abs); got != lexical {
			t.Fatalf("%q: inside = %v, lexical check says %v", rel, got, lexical)
		}
	})
}
