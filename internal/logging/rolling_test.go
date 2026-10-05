package logging

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// clock returns a fake now that advances by step on every call.
func clock(start time.Time, step time.Duration) func() time.Time {
	t := start
	return func() time.Time {
		now := t
		t = t.Add(step)
		return now
	}
}

var t0 = time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)

func newTestSink(t *testing.T, limit int64, now func() time.Time, rename func(string, string) error, warn *bytes.Buffer) (*rolling, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "logs", "gobble.log")
	var w io.Writer // a nil *bytes.Buffer would be a non-nil io.Writer
	if warn != nil {
		w = warn
	}
	r, err := newRolling(path, w, limit, now, rename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	return r, path
}

func write(t *testing.T, r *rolling, s string) {
	t.Helper()
	if _, err := r.Write([]byte(s)); err != nil {
		t.Fatal(err)
	}
}

func line(i, size int) string {
	s := strings.Repeat(string(rune('a'+i%26)), size-1)
	return s + "\n"
}

// backups lists the backup files, oldest first by name.
func backups(t *testing.T, dir string) []string {
	t.Helper()
	des, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, d := range des {
		if d.Name() != "gobble.log" && strings.HasPrefix(d.Name(), "gobble-") {
			out = append(out, d.Name())
		}
	}
	slices.Sort(out)
	return out
}

func TestRotatesBeforeTheLimit(t *testing.T) {
	r, path := newTestSink(t, 100, clock(t0, time.Millisecond), os.Rename, nil)
	for i := range 4 {
		write(t, r, line(i, 40))
	}
	bs := backups(t, filepath.Dir(path))
	if len(bs) != 1 {
		t.Fatalf("backups = %v, want 1", bs)
	}
	if got := readFile(t, filepath.Join(filepath.Dir(path), bs[0])); got != line(0, 40)+line(1, 40) {
		t.Fatalf("backup = %q", got)
	}
	if got := readFile(t, path); got != line(2, 40)+line(3, 40) {
		t.Fatalf("active = %q", got)
	}
	if want := "gobble-2026-10-05T12-00-00.000.log"; bs[0] != want {
		t.Fatalf("backup name = %q, want %q", bs[0], want)
	}
}

func TestKeepsFiveBackups(t *testing.T) {
	r, path := newTestSink(t, 100, clock(t0, time.Second), os.Rename, nil)
	for i := range 20 {
		write(t, r, line(i, 60))
	}
	bs := backups(t, filepath.Dir(path))
	if len(bs) != maxBackups {
		t.Fatalf("backups = %v, want %d", bs, maxBackups)
	}
	// The newest backup holds the line before the active one.
	if got := readFile(t, filepath.Join(filepath.Dir(path), bs[len(bs)-1])); got != line(18, 60) {
		t.Fatalf("newest backup = %q, want line 18", got)
	}
}

func TestCollisionGetsASuffix(t *testing.T) {
	fixed := func() time.Time { return t0 }
	r, path := newTestSink(t, 100, fixed, os.Rename, nil)
	for i := range 9 {
		write(t, r, line(i, 60))
	}
	bs := backups(t, filepath.Dir(path))
	want := []string{
		"gobble-2026-10-05T12-00-00.000-3.log", "gobble-2026-10-05T12-00-00.000-4.log",
		"gobble-2026-10-05T12-00-00.000-5.log", "gobble-2026-10-05T12-00-00.000-6.log",
		"gobble-2026-10-05T12-00-00.000-7.log",
	}
	if !slices.Equal(bs, want) {
		t.Fatalf("backups = %v\nwant      %v (eight rotations in one millisecond, newest five kept)", bs, want)
	}
}

func TestPrunesByAge(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	old := "gobble-" + t0.Add(-30*24*time.Hour).Format(stampLayout) + ".log"
	recent := "gobble-" + t0.Add(-24*time.Hour).Format(stampLayout) + ".log"
	unrelated := []string{"other.txt", "gobble-not-a-stamp.log", "gobble-" + t0.Format(stampLayout) + "-x.log"}
	for _, n := range append([]string{old, recent}, unrelated...) {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := newRolling(filepath.Join(dir, "gobble.log"), nil, 100, clock(t0, time.Millisecond), os.Rename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	write(t, r, line(0, 60))
	write(t, r, line(1, 60)) // rotates, then prunes
	if _, err := os.Stat(filepath.Join(dir, old)); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a 30-day-old backup survived: %v", err)
	}
	for _, n := range append([]string{recent}, unrelated...) {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s was removed: %v", n, err)
		}
	}
}

func TestOversizedRecordGoesToAFreshFile(t *testing.T) {
	r, path := newTestSink(t, 100, clock(t0, time.Millisecond), os.Rename, nil)
	write(t, r, line(0, 50))
	big := line(1, 300)
	write(t, r, big)
	if got := readFile(t, path); got != big {
		t.Fatalf("active = %d bytes, want the 300-byte record alone", len(got))
	}
	write(t, r, line(2, 10))
	bs := backups(t, filepath.Dir(path))
	if len(bs) != 2 || readFile(t, filepath.Join(filepath.Dir(path), bs[1])) != big {
		t.Fatalf("backups = %v; the big record should be the newest backup", bs)
	}
}

func TestRotatesOnOpenAtTheLimit(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "logs")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "gobble.log")
	if err := os.WriteFile(path, []byte(line(0, 150)), 0o600); err != nil {
		t.Fatal(err)
	}
	r, err := newRolling(path, nil, 100, clock(t0, time.Millisecond), os.Rename)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := r.Close(); err != nil {
			t.Error(err)
		}
	})
	if bs := backups(t, dir); len(bs) != 1 {
		t.Fatalf("backups = %v, want the full file rotated on open", bs)
	}
	if got := readFile(t, path); got != "" {
		t.Fatalf("active = %q, want empty", got)
	}
}

func TestFailedRotationKeepsWritingAndWarnsOnce(t *testing.T) {
	var warn bytes.Buffer
	failing := func(string, string) error { return errors.New("sharing violation") }
	r, path := newTestSink(t, 100, clock(t0, time.Millisecond), failing, &warn)
	var want strings.Builder
	for i := range 5 {
		write(t, r, line(i, 60))
		want.WriteString(line(i, 60))
	}
	if got := readFile(t, path); got != want.String() {
		t.Fatalf("a line was lost: active holds %d bytes, want %d", len(got), want.Len())
	}
	if n := strings.Count(warn.String(), "log rotation failed"); n != 1 {
		t.Fatalf("warned %d times, want once: %q", n, warn.String())
	}
	if !strings.Contains(warn.String(), "sharing violation") {
		t.Fatalf("warning lacks the cause: %q", warn.String())
	}
}

func TestWriteAfterClose(t *testing.T) {
	r, _ := newTestSink(t, 100, clock(t0, time.Millisecond), os.Rename, nil)
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Write([]byte("x")); !errors.Is(err, os.ErrClosed) {
		t.Fatalf("Write after Close = %v, want os.ErrClosed", err)
	}
}
