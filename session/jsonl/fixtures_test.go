package jsonl

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/session"
)

// The fixtures are written by Pi's own SessionManager, v1 and v2 at the
// last commits that wrote them and v3 at 312184edb, with invented content
// (0005-PLAN "F2 made executable"). <name>.migrated.jsonl is Pi's own
// migration of an older file.
const fixtures = "../testdata/pi"

var (
	current = []string{"v3", "v3-branched"}
	older   = []string{"v1", "v1-branched", "v2", "v2-branched"}
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(fixtures, name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func lines(b []byte) [][]byte { return bytes.Split(bytes.TrimSuffix(b, []byte("\n")), []byte("\n")) }

// install writes a fixture's bytes into a new store directory as Pi names
// the file, and returns the store, the session's id and the file's path.
func install(t *testing.T, body []byte, o Options) (*Store, session.ID, string) {
	t.Helper()
	var h session.Header
	if err := json.Unmarshal(lines(body)[0], &h); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	p := filepath.Join(dir, EncodeCwd(h.Cwd), FileName(h))
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return New(dir, o), h.ID, p
}

// normal is a JSON value decoded into any, so two encodings compare
// whatever their member order.
func normal(t *testing.T, b []byte) any {
	t.Helper()
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		t.Fatalf("%s: %v", b, err)
	}
	return v
}

// Each v3 file loads whole, and each entry, written again, is its line.
func TestFixturesRoundTrip(t *testing.T) {
	for _, name := range current {
		body := fixture(t, name+".jsonl")
		s, id, _ := install(t, body, Options{})
		h, entries, skipped, err := s.Read(t.Context(), id)
		if err != nil || skipped != 0 {
			t.Fatalf("%s: %v, %d skipped", name, err, skipped)
		}
		ls := lines(body)
		if len(entries) != len(ls)-1 {
			t.Fatalf("%s: %d entries for %d lines", name, len(entries), len(ls))
		}
		for i, v := range append([]any{h}, entriesAny(entries)...) {
			out, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := normal(t, out), normal(t, ls[i]); !reflect.DeepEqual(got, want) {
				t.Errorf("%s line %d changed:\n got %s\nwant %s", name, i+1, out, ls[i])
			}
		}
	}
}

// Each older file migrates as Pi migrates it, with the ids mapped by
// position, and is never written: not by loading, closing, or an append,
// which is refused.
func TestFixturesMigrate(t *testing.T) {
	for _, name := range older {
		body := fixture(t, name+".jsonl")
		s, id, path := install(t, body, Options{})
		l, err := s.Open(t.Context(), id)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		entries := l.Entries()
		pi := lines(fixture(t, name+".migrated.jsonl"))[1:]
		if len(entries) != len(pi) {
			t.Fatalf("%s: %d entries, Pi has %d", name, len(entries), len(pi))
		}
		ids := map[string]string{} // Pi's id → gobble's
		for i, line := range pi {
			var e session.Entry
			if err := json.Unmarshal(line, &e); err != nil {
				t.Fatal(err)
			}
			ids[e.ID] = entries[i].ID
		}
		for i, line := range pi {
			out, err := json.Marshal(entries[i])
			if err != nil {
				t.Fatal(err)
			}
			if got, want := normal(t, out), mapIDs(normal(t, line), ids); !reflect.DeepEqual(got, want) {
				t.Errorf("%s entry %d:\n got %s\nwant %s (ids mapped)", name, i+1, out, line)
			}
		}
		err = l.Append(t.Context(), userEntry("eeee0001", entries[len(entries)-1].ID, "more"))
		old, ok := errors.AsType[*session.ErrOldVersion](err)
		if !ok || old.ID != id || !strings.Contains(err.Error(), "/clone it to continue in a new session") {
			t.Errorf("%s: append %v, want ErrOldVersion", name, err)
		}
		if err := l.Close(); err != nil {
			t.Fatal(err)
		}
		if after, err := os.ReadFile(path); err != nil || !bytes.Equal(after, body) {
			t.Errorf("%s: the file changed (%v)", name, err)
		}
		if l.Header().Version == session.Version || len(l.Entries()) != len(pi) {
			t.Errorf("%s: header version %d, %d entries after the refused append", name, l.Header().Version, len(l.Entries()))
		}
	}
}

// mapIDs replaces Pi's ids in an entry with gobble's.
func mapIDs(v any, ids map[string]string) any {
	m, ok := v.(map[string]any)
	if !ok {
		return v
	}
	for _, k := range []string{"id", "parentId", "firstKeptEntryId", "targetId", "fromId"} {
		if s, ok := m[k].(string); ok {
			if g, ok := ids[s]; ok {
				m[k] = g
			}
		}
	}
	return m
}

// On copies of v3.jsonl: a malformed line is skipped with a warning, a
// header that cannot be read fails closed, and a last line cut short, as an
// interrupted write leaves it, loses that entry only.
func TestFixtureDamage(t *testing.T) {
	body := fixture(t, "v3.jsonl")
	ls := lines(body)
	n := len(ls) - 1

	malformed := bytes.Join(append(append([][]byte{}, ls[:3]...), append([][]byte{[]byte(`{"type":`)}, ls[3:]...)...), []byte("\n"))
	var log bytes.Buffer
	s, id, _ := install(t, append(malformed, '\n'), Options{Logger: slog.New(slog.NewTextHandler(&log, nil))})
	if _, entries, skipped, err := s.Read(t.Context(), id); err != nil || skipped != 1 || len(entries) != n {
		t.Errorf("malformed: %d entries, %d skipped, %v", len(entries), skipped, err)
	}
	if !strings.Contains(log.String(), "malformed lines skipped") {
		t.Errorf("no warning: %q", log.String())
	}

	headless := bytes.Join(append([][]byte{[]byte(`{"type":"session","id":`)}, ls[1:]...), []byte("\n"))
	dir := t.TempDir()
	p := filepath.Join(dir, EncodeCwd("/work/project"), "2026-10-10T00-00-00-000Z_cut.jsonl")
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, headless, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := New(dir, Options{}).Read(t.Context(), "cut"); !errors.Is(err, session.ErrCorrupt) {
		t.Errorf("unreadable header: %v, want ErrCorrupt", err)
	}

	last := ls[len(ls)-1]
	cut := append(bytes.Join(ls[:len(ls)-1], []byte("\n")), '\n')
	cut = append(cut, last[:len(last)/2]...)
	s, id, _ = install(t, cut, Options{})
	if _, entries, skipped, err := s.Read(t.Context(), id); err != nil || skipped != 1 || len(entries) != n-1 {
		t.Errorf("cut short: %d entries, %d skipped, %v; want %d and 1", len(entries), skipped, err, n-1)
	}
}
