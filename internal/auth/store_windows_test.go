package auth

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// holdBriefly opens p as a reader does, with os.Open, and closes it after
// 100 ms; wait returns once it is closed.
func holdBriefly(t *testing.T, p string) (wait func()) {
	t.Helper()
	reader, err := os.Open(p) //nolint:gosec // G304: the test's temp file
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		time.Sleep(100 * time.Millisecond)
		reader.Close() //nolint:errcheck,gosec // read-only
	}()
	return func() { <-done }
}

// A key's save replaces auth.json while a reader holds it: the replace is
// refused (Access is denied) until the reader lets go, inside the window,
// and then the file holds the new key (0067-MADR X1, D4).
func TestSetKeyReplaceSurvivesAReader(t *testing.T) {
	p := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(p, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wait := holdBriefly(t, p)
	err := SetKey(t.Context(), p, "claude", "sk-test-NEW", "")
	wait()
	if err != nil {
		t.Fatalf("SetKey under a reader: %v; want the save once the reader lets go", err)
	}
	f, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := f.Entry("claude"); !ok || e.Key != "sk-test-NEW" {
		t.Fatalf("auth.json after SetKey: %+v, %v; want the new key", e, ok)
	}
}

// clients.json is written as auth.json is, so its save survives a reader
// the same way (0067-MADR X3, D4).
func TestClientsSaveReplaceSurvivesAReader(t *testing.T) {
	c := Clients{Path: filepath.Join(t.TempDir(), "clients.json")}
	if err := os.WriteFile(c.Path, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wait := holdBriefly(t, c.Path)
	want := ClientRecord{ClientID: "oaiapp_new"}
	err := c.Save(t.Context(), "openai", want)
	wait()
	if err != nil {
		t.Fatalf("Clients.Save under a reader: %v; want the save once the reader lets go", err)
	}
	if got, err := c.Load(t.Context(), "openai"); err != nil || got != want {
		t.Fatalf("clients.json after Save: %+v, %v; want %+v", got, err, want)
	}
}
