package fsx

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolve(t *testing.T) {
	cwd := t.TempDir()
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	cases := []struct{ in, want string }{
		{"a.go", filepath.Join(cwd, "a.go")},
		{"@a.go", filepath.Join(cwd, "a.go")},
		{"dir/../b.go", filepath.Join(cwd, "b.go")},
		{"my\u00a0file.txt", filepath.Join(cwd, "my file.txt")},
		{"x\u3000y", filepath.Join(cwd, "x y")},
		{"~", home},
		{"~/notes.md", filepath.Join(home, "notes.md")},
		{filepath.Join(cwd, "abs.go"), filepath.Join(cwd, "abs.go")},
	}
	for _, tc := range cases {
		if got := Resolve(cwd, tc.in); got != tc.want {
			t.Errorf("Resolve(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestResolveFileURL(t *testing.T) {
	cwd := t.TempDir()
	want := filepath.Join(cwd, "u.go")
	url := "file://" + filepath.ToSlash(want)
	if runtime.GOOS == "windows" {
		url = "file:///" + filepath.ToSlash(want)
	}
	if got := Resolve(cwd, url); got != want {
		t.Fatalf("Resolve(%q) = %q, want %q", url, got, want)
	}
}

// Git Bash, MSYS, Cygwin and WSL drive paths are native paths on Windows,
// and left alone elsewhere (Pi normalizeWindowsShellPath).
func TestWindowsShellPath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"/c/Users/<user>/a.go", `C:\Users\<user>\a.go`},
		{"/mnt/d/src", `D:\src`},
		{"/cygdrive/e/", `E:\`},
		{"/c", `C:\`},
		{"//server/share", "//server/share"},
		{`/c\mixed`, `/c\mixed`},
		{"/usr/lib", "/usr/lib"},
	}
	for _, tc := range cases {
		if got := windowsShell(tc.in); got != tc.want {
			t.Errorf("windowsShell(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
	if got := normalize("linux", "/c/Users", true); got != "/c/Users" {
		t.Errorf("on linux /c/Users became %q", got)
	}
}

func TestReadVariants(t *testing.T) {
	got := ReadVariants("/s/Screenshot 2026-10-07 at 9.41.00 PM.png")
	if len(got) == 0 || got[0] != "/s/Screenshot 2026-10-07 at 9.41.00\u202fPM.png" {
		t.Fatalf("variants = %q, want the narrow-space AM/PM spelling first", got)
	}
	got = ReadVariants("/s/Capture d'e\u0301cran.png")
	want := "/s/Capture d\u2019e\u0301cran.png"
	found := false
	for _, v := range got {
		found = found || v == want
	}
	if !found {
		t.Fatalf("variants = %q, want %q among them", got, want)
	}
	if got := ReadVariants("/plain/a.go"); len(got) != 0 {
		t.Fatalf("variants of a plain path = %q, want none", got)
	}
}
