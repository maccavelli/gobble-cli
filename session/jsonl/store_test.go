package jsonl

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/session"
)

func header(id session.ID, cwd string) session.Header {
	return session.Header{Type: session.TypeHeader, Version: session.Version, ID: id, Timestamp: "2024-12-03T14:00:00.000Z", Cwd: cwd}
}

func userEntry(id, parent, text string) session.Entry {
	e := session.Entry{Type: session.TypeMessage, ID: id, Timestamp: "2024-12-03T14:00:01.000Z",
		Message: &session.Message{Role: session.RoleUser, Content: jsontext.Value(`"` + text + `"`), Timestamp: 1733234401000}}
	if parent != "" {
		e.ParentID = &parent
	}
	return e
}

func model(id, parent string) session.Entry {
	return session.Entry{Type: session.TypeModelChange, ID: id, ParentID: &parent, Timestamp: "2024-12-03T14:00:00.500Z", Provider: "openai", ModelID: "gpt-4.1"}
}

func files(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() {
			rel, _ := filepath.Rel(dir, p) //nolint:errcheck // under dir
			out = append(out, filepath.ToSlash(rel))
		}
		return err
	})
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return out
}

// A session is a file only once it has a user or assistant message; the
// file is named and placed as Pi names and places it.
func TestLazyCreationAndLayout(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, Options{})
	ctx := t.Context()
	cwd := filepath.Join(string(filepath.Separator)+"srv", "work", "proj")
	l, err := s.Create(ctx, header("019a0000-0000-7000-8000-000000000001", cwd))
	if err != nil {
		t.Fatal(err)
	}
	thinking := session.Entry{Type: session.TypeThinkingLevelChange, ID: "aaaa0001", Timestamp: "2024-12-03T14:00:00.100Z", ThinkingLevel: "high"}
	if err := l.Append(ctx, thinking); err != nil {
		t.Fatal(err)
	}
	if got := files(t, dir); len(got) != 0 {
		t.Fatalf("files before any message: %v", got)
	}
	if err := l.Append(ctx, userEntry("aaaa0002", "aaaa0001", "hi")); err != nil {
		t.Fatal(err)
	}
	want := EncodeCwd(cwd) + "/2024-12-03T14-00-00-000Z_019a0000-0000-7000-8000-000000000001.jsonl"
	got := files(t, dir)
	if len(got) != 2 || got[0] != want || got[1] != want+".lock" {
		t.Fatalf("files %v, want %s and its lock", got, want)
	}
	if err := l.Append(ctx, model("aaaa0003", "aaaa0002")); err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(want)))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(b), "\n"), "\n")
	if len(lines) != 4 || !strings.HasPrefix(lines[0], `{"type":"session","version":3,`) ||
		!strings.Contains(lines[1], `"parentId":null`) || !strings.Contains(lines[2], `"parentId":"aaaa0001"`) {
		t.Fatalf("file:\n%s", b)
	}
}

func TestEncodeCwd(t *testing.T) {
	for cwd, want := range map[string]string{
		"/srv/work/proj":  "--srv-work-proj--",
		`C:\Users\u\proj`: "--C--Users-u-proj--",
		`\\srv\share`:     `---srv-share--`,
	} {
		if got := EncodeCwd(cwd); got != want {
			t.Errorf("EncodeCwd(%q) = %q, want %q", cwd, got, want)
		}
	}
}

// A second writer gets ErrLocked with the first's pid; a reader needs no
// lock; closing releases the writer.
func TestWriterLock(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()
	s := New(dir, Options{})
	l := create(t, s, "s1", "/w")
	other := New(dir, Options{}) // a second store, as a second process opens it
	_, err := other.Open(ctx, "s1")
	locked, ok := errors.AsType[*session.ErrLocked](err)
	if !ok || locked.ID != "s1" || locked.PID != os.Getpid() {
		t.Fatalf("second writer: %v", err)
	}
	if !strings.Contains(err.Error(), "session s1 is open in another gobble process (pid ") {
		t.Fatalf("message %q", err.Error())
	}
	if _, entries, _, err := other.Read(ctx, "s1"); err != nil || len(entries) != 1 {
		t.Fatalf("reader: %d entries, %v", len(entries), err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	l2, err := other.Open(ctx, "s1")
	if err != nil {
		t.Fatalf("open after close: %v", err)
	}
	if err := l2.Close(); err != nil {
		t.Fatal(err)
	}
}

func create(t *testing.T, s *Store, id session.ID, cwd string) session.Log {
	t.Helper()
	l, err := s.Create(t.Context(), header(id, cwd))
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Append(t.Context(), userEntry("bbbb0001", "", "first prompt")); err != nil {
		t.Fatal(err)
	}
	return l
}

// A malformed middle line is skipped and counted; a torn last line is kept
// apart from the next entry, which a reader then sees.
func TestMalformedAndTornLines(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()
	s := New(dir, Options{})
	l := create(t, s, "s1", "/w")
	path := l.(*fileLog).path
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{not json}\n" + `{"type":"message","id":"cccc0001","parentId":"bbbb0001","timestamp":"x","message":{"role":"user","content":"tor`); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	w, err := s.Open(ctx, "s1")
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Append(ctx, userEntry("dddd0001", "bbbb0001", "after")); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	_, entries, skipped, err := s.Read(ctx, "s1")
	if err != nil || skipped != 2 || len(entries) != 2 || entries[1].ID != "dddd0001" {
		t.Fatalf("read: %d entries, %d skipped, %v", len(entries), skipped, err)
	}
}

// A file whose header is unreadable, or of a version above 3, fails closed
// (v1 and v2 are migrated: 0005-PLAN F2).
func TestBadHeaderFailsClosed(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, EncodeCwd("/w"))
	if err := os.MkdirAll(sub, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"x_bad1.jsonl": `{"type":"message","id":"a"}` + "\n",
		"x_bad2.jsonl": "garbage\n",
		"x_v4.jsonl":   `{"type":"session","version":4,"id":"v4","timestamp":"t","cwd":"/w"}` + "\n",
	} {
		if err := os.WriteFile(filepath.Join(sub, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	s := New(dir, Options{})
	for _, id := range []session.ID{"bad1", "bad2", "v4"} {
		if _, err := s.Open(t.Context(), id); !errors.Is(err, session.ErrCorrupt) {
			t.Errorf("open %s: %v, want ErrCorrupt", id, err)
		}
	}
	_, err := s.Open(t.Context(), "v4")
	if err == nil || !strings.Contains(err.Error(), "unsupported session version 4") {
		t.Fatalf("v4: %v", err)
	}
	if n := countList(t, s, session.Filter{}); n != 0 {
		t.Fatalf("listed %d invalid sessions", n)
	}
}

func countList(t *testing.T, s session.Store, f session.Filter) int {
	t.Helper()
	n := 0
	for _, err := range s.List(context.Background(), f) {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	return n
}

func TestListAndFind(t *testing.T) {
	dir := t.TempDir()
	ctx := t.Context()
	s := New(dir, Options{})
	for _, c := range []struct {
		id  session.ID
		cwd string
	}{{"a1", "/w"}, {"b2", "/w"}, {"c3", "/other"}} {
		if err := create(t, s, c.id, c.cwd).Close(); err != nil {
			t.Fatal(err)
		}
	}
	if n := countList(t, s, session.Filter{Cwd: "/w"}); n != 2 {
		t.Fatalf("listed %d for /w", n)
	}
	if n := countList(t, s, session.Filter{}); n != 3 {
		t.Fatalf("listed %d in all", n)
	}
	if _, err := s.Create(ctx, header("a1", "/w")); !errors.Is(err, session.ErrExists) {
		t.Fatalf("duplicate: %v", err)
	}
	if _, err := s.Open(ctx, "nope"); !errors.Is(err, session.ErrNotFound) {
		t.Fatalf("unknown: %v", err)
	}
	// A flat store keeps files side by side, and filters by the header's cwd.
	flat := New(t.TempDir(), Options{Flat: true})
	if err := create(t, flat, "f1", "/w").Close(); err != nil {
		t.Fatal(err)
	}
	if err := create(t, flat, "f2", "/x").Close(); err != nil {
		t.Fatal(err)
	}
	if n := countList(t, flat, session.Filter{Cwd: "/x"}); n != 1 {
		t.Fatalf("flat listed %d for /x", n)
	}
	if got := files(t, flat.Dir()); len(got) != 4 || strings.Contains(got[0], "/") {
		t.Fatalf("flat files %v", got)
	}
}

// A created session never written leaves nothing, and frees its id.
func TestCloseUnwritten(t *testing.T) {
	dir := t.TempDir()
	s := New(dir, Options{})
	l, err := s.Create(t.Context(), header("z9", "/w"))
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Close(); err != nil {
		t.Fatal(err)
	}
	if got := files(t, dir); len(got) != 0 {
		t.Fatalf("files %v", got)
	}
	if _, err := s.Create(t.Context(), header("z9", "/w")); err != nil {
		t.Fatalf("id not freed: %v", err)
	}
}
