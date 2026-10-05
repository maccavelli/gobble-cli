package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestHistoryRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "history")
	h, err := OpenHistory(path)
	if err != nil || h.Len() != 0 {
		t.Fatalf("OpenHistory(missing) = %v, %v; want empty", h, err)
	}
	entries := []string{"plain", "two\nlines", `back\slash`, `literal \n text`, "plain"}
	for _, e := range entries {
		h.Add(e)
	}
	h.Add("plain") // repeats the most recent: skipped
	h.Add("")      // empty: skipped
	if err := h.Err(); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantFile := "plain\ntwo\\nlines\nback\\\\slash\nliteral \\\\n text\nplain\n"
	if string(b) != wantFile {
		t.Fatalf("file = %q, want %q", b, wantFile)
	}

	again, err := OpenHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if again.Len() != len(entries) {
		t.Fatalf("Len = %d, want %d", again.Len(), len(entries))
	}
	for i, want := range entries {
		if got := again.At(len(entries) - 1 - i); got != want {
			t.Errorf("entry %d = %q, want %q", i, got, want)
		}
	}
}

func TestHistoryCapAndFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "history")
	var seed strings.Builder
	for i := range maxHistory + 4 {
		fmt.Fprintf(&seed, "entry %d\n", i)
	}
	if err := os.WriteFile(path, []byte(seed.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	h, err := OpenHistory(path)
	if err != nil {
		t.Fatal(err)
	}
	if h.Len() != maxHistory || h.At(h.Len()-1) != "entry 4" {
		t.Fatalf("loaded Len = %d, oldest %q; want %d and \"entry 4\"", h.Len(), h.At(h.Len()-1), maxHistory)
	}
	h.Add(fmt.Sprintf("entry %d", maxHistory+4))
	if h.Len() != maxHistory {
		t.Fatalf("Len = %d, want %d", h.Len(), maxHistory)
	}
	if got := h.At(h.Len() - 1); got != "entry 5" {
		t.Fatalf("oldest = %q, want \"entry 5\"", got)
	}
	des, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(des) != 1 || des[0].Name() != "history" {
		t.Fatalf("directory holds %v; the atomic replace must leave only the history file", des)
	}
	if runtime.GOOS != "windows" {
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if fi.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %v, want 0600", fi.Mode().Perm())
		}
	}
}

func TestHistorySaveError(t *testing.T) {
	h, err := OpenHistory(filepath.Join(t.TempDir(), "missing-dir", "history"))
	if err != nil {
		t.Fatal(err)
	}
	h.Add("x")
	if h.Err() == nil {
		t.Fatal("Err() = nil after saving into a missing directory")
	}
	if h.Len() != 1 {
		t.Fatal("a failed save must keep the entry in memory")
	}
}

func TestHistoryAtPanicsOutOfRange(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("At(0) on an empty history did not panic")
		}
	}()
	h := &FileHistory{}
	_ = h.At(0)
}
