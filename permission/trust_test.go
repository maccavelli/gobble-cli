package permission

import (
	"encoding/json/v2"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

// realDir is dir's real path, as the store keys it: t.TempDir is under a
// link on macOS.
func realDir(t *testing.T, dir string) string {
	t.Helper()
	p, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func store(t *testing.T) TrustStore {
	t.Helper()
	return TrustStore{Path: filepath.Join(t.TempDir(), "config", TrustFile), Home: t.TempDir()}
}

func mkdir(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	if err := os.MkdirAll(p, 0o750); err != nil {
		t.Fatal(err)
	}
	return p
}

// Pi's order, a row per step (0005-PLAN F6-1 decision 2): each step
// decides only when every step before it did not.
func TestDecideTrustOrder(t *testing.T) {
	yes, no := true, false
	saved := func(trusted bool) func(string) (bool, string, bool, error) {
		return func(string) (bool, string, bool, error) { return trusted, "/saved", true, nil }
	}
	none := func(string) (bool, string, bool, error) { return false, "", false, nil }
	answer := func(trusted bool) func(string) (bool, bool) {
		return func(string) (bool, bool) { return trusted, true }
	}
	noAnswer := func(string) (bool, bool) { return false, false }
	for _, c := range []struct {
		name string
		in   TrustInput
		want TrustDecision
	}{
		{"--approve over a saved no", TrustInput{Flag: &yes, Needs: true, Saved: saved(false), Default: DefaultNever}, TrustDecision{Trusted: true, Source: SourceFlag}},
		{"--no-approve over nothing to trust", TrustInput{Flag: &no, Saved: saved(true)}, TrustDecision{Source: SourceFlag}},
		{"nothing to trust over a saved no", TrustInput{Saved: saved(false), Default: DefaultNever}, TrustDecision{Trusted: true, Source: SourceNothing}},
		{"saved yes over never", TrustInput{Needs: true, Saved: saved(true), Default: DefaultNever}, TrustDecision{Trusted: true, Source: SourceSaved, At: "/saved"}},
		{"saved no over always", TrustInput{Needs: true, Saved: saved(false), Default: DefaultAlways, Ask: answer(true)}, TrustDecision{Source: SourceSaved, At: "/saved"}},
		{"always", TrustInput{Needs: true, Saved: none, Default: DefaultAlways, Ask: answer(false)}, TrustDecision{Trusted: true, Source: SourceDefault}},
		{"never", TrustInput{Needs: true, Saved: none, Default: DefaultNever, Ask: answer(true)}, TrustDecision{Source: SourceDefault}},
		{"ask, answered yes", TrustInput{Needs: true, Saved: none, Default: DefaultAsk, Ask: answer(true)}, TrustDecision{Trusted: true, Source: SourceAnswer}},
		{"ask, answered no", TrustInput{Needs: true, Saved: none, Default: DefaultAsk, Ask: answer(false)}, TrustDecision{Source: SourceAnswer}},
		{"ask, no answer", TrustInput{Needs: true, Saved: none, Default: DefaultAsk, Ask: noAnswer}, TrustDecision{Source: SourceAsk}},
		{"ask, no one to ask", TrustInput{Needs: true, Default: DefaultAsk}, TrustDecision{Source: SourceAsk}},
	} {
		got, err := DecideTrust(c.in)
		if err != nil || got != c.want {
			t.Errorf("%s: %+v, %v; want %+v", c.name, got, err, c.want)
		}
	}
	boom := errors.New("unreadable")
	if _, err := DecideTrust(TrustInput{Needs: true, Saved: func(string) (bool, string, bool, error) { return false, "", false, boom }}); !errors.Is(err, boom) {
		t.Errorf("a store that cannot be read: %v", err)
	}
}

// The nearest saved decision wins, the root's own or a parent's (Pi's
// lookup), and Forget lets the next one up decide.
func TestTrustLookupNearest(t *testing.T) {
	s := store(t)
	base := realDir(t, t.TempDir())
	parent := mkdir(t, base, "work")
	child := mkdir(t, parent, "repo")
	deep := mkdir(t, child, "sub", "dir")
	sibling := mkdir(t, parent, "other")
	if _, _, ok, err := s.Lookup(child); ok || err != nil {
		t.Fatalf("an empty store decided: %v %v", ok, err)
	}
	for _, d := range []struct {
		dir     string
		trusted bool
	}{{parent, true}, {child, false}} {
		if _, err := s.Save(t.Context(), d.dir, d.trusted); err != nil {
			t.Fatal(err)
		}
	}
	for _, c := range []struct {
		dir     string
		trusted bool
		at      string
	}{{child, false, child}, {deep, false, child}, {sibling, true, parent}, {parent, true, parent}} {
		trusted, at, ok, err := s.Lookup(c.dir)
		if !ok || err != nil || trusted != c.trusted || at != c.at {
			t.Errorf("Lookup(%s) = %v %s %v %v, want %v %s", c.dir, trusted, at, ok, err, c.trusted, c.at)
		}
	}
	if at, removed, err := s.Forget(t.Context(), child); !removed || err != nil || at != child {
		t.Fatalf("Forget = %s %v %v", at, removed, err)
	}
	if trusted, at, ok, _ := s.Lookup(deep); !ok || !trusted || at != parent {
		t.Errorf("after Forget, Lookup(deep) = %v %s %v, want the parent's", trusted, at, ok)
	}
	if _, removed, err := s.Forget(t.Context(), child); removed || err != nil {
		t.Errorf("a second Forget removed %v, %v", removed, err)
	}
}

// A home directory and a filesystem root are never saved (decision 3),
// and nothing is written for them.
func TestTrustSaveTooBroad(t *testing.T) {
	s := store(t)
	root := string(filepath.Separator)
	if runtime.GOOS == "windows" {
		root = filepath.VolumeName(os.TempDir()) + `\`
	}
	for _, dir := range []string{s.Home, root} {
		if _, err := s.Save(t.Context(), dir, true); !errors.Is(err, ErrTooBroad) {
			t.Errorf("Save(%s) = %v, want ErrTooBroad", dir, err)
		}
	}
	if _, err := os.Stat(s.Path); !os.IsNotExist(err) {
		t.Errorf("the file was written: %v", err)
	}
	if !TooBroad(realDir(t, s.Home), s.Home) || TooBroad(mkdir(t, realDir(t, s.Home), "repo"), s.Home) {
		t.Error("TooBroad: home itself refused, a project under it allowed")
	}
}

// A root reached through a link is saved, and found, by its real path.
func TestTrustSymlinkedRoot(t *testing.T) {
	s := store(t)
	target := mkdir(t, realDir(t, t.TempDir()), "repo")
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	at, err := s.Save(t.Context(), link, true)
	if err != nil || at != target {
		t.Fatalf("Save = %s %v, want %s", at, err, target)
	}
	m, err := s.Load()
	if err != nil || len(m) != 1 || !m[target] {
		t.Fatalf("saved %v %v", m, err)
	}
	for _, dir := range []string{target, link} {
		if trusted, at, ok, _ := s.Lookup(dir); !ok || !trusted || at != target {
			t.Errorf("Lookup(%s) = %v %s %v, want the real path's", dir, trusted, at, ok)
		}
	}
}

// The file is Pi's shape: an object, sorted, two-space indent, mode 0600.
func TestTrustFileShape(t *testing.T) {
	s := store(t)
	base := realDir(t, t.TempDir())
	b, a := mkdir(t, base, "b"), mkdir(t, base, "a")
	for _, d := range []struct {
		dir     string
		trusted bool
	}{{b, false}, {a, true}} {
		if _, err := s.Save(t.Context(), d.dir, d.trusted); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(s.Path)
	if err != nil {
		t.Fatal(err)
	}
	q := func(p string) string { j, _ := json.Marshal(p); return string(j) }
	want := "{\n  " + q(a) + ": true,\n  " + q(b) + ": false\n}\n"
	if string(got) != want {
		t.Errorf("file:\n%s\nwant:\n%s", got, want)
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(s.Path); err != nil || info.Mode().Perm() != 0o600 {
			t.Errorf("mode %v %v, want 0600", info.Mode().Perm(), err)
		}
	}
}

// Two saves at once lose neither.
func TestTrustConcurrentSaves(t *testing.T) {
	s := store(t)
	base := realDir(t, t.TempDir())
	var dirs []string
	for i := range 16 {
		dirs = append(dirs, mkdir(t, base, string(rune('a'+i))))
	}
	var wg sync.WaitGroup
	for _, d := range dirs {
		wg.Go(func() {
			if _, err := s.Save(t.Context(), d, true); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	m, err := s.Load()
	if err != nil || len(m) != len(dirs) {
		t.Errorf("saved %v %v, want all %d", m, err, len(dirs))
	}
}
