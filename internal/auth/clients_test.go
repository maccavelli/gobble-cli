package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
)

// A record is saved and loaded; a zero record removes its member and no
// other; a session's Delete in auth.json leaves the client (0052-PLAN P2-1).
func TestClientsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	c := Clients{Path: filepath.Join(dir, "clients.json")}
	if got, err := c.Load(t.Context(), "openai"); err != nil || got != (ClientRecord{}) {
		t.Fatalf("Load before any save: %+v, %v; want the zero record", got, err)
	}
	openai := ClientRecord{ClientID: "oaiapp_x", HostID: "urn:uuid:host"}
	other := ClientRecord{ClientID: "other_x"}
	for id, r := range map[string]ClientRecord{"openai": openai, "other": other} {
		if err := c.Save(t.Context(), id, r); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := c.Load(t.Context(), "openai"); err != nil || got != openai {
		t.Errorf("Load openai: %+v, %v; want %+v", got, err, openai)
	}
	raw, err := os.ReadFile(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`"clientId": "oaiapp_x"`, `"hostId": "urn:uuid:host"`} {
		if !strings.Contains(string(raw), s) {
			t.Errorf("no %q in:\n%s", s, raw)
		}
	}
	if ok, err := appdirs.FileIsOwnerOnly(c.Path); err != nil || !ok {
		t.Errorf("owner-only %v, %v; want the user's alone", ok, err)
	}
	if runtime.GOOS != "windows" {
		if fi, err := os.Stat(c.Path); err != nil || fi.Mode().Perm() != 0o600 {
			t.Errorf("%s: %v, %v; want 0600", c.Path, fi, err)
		}
	}

	// A session's Delete, and a key over it, leave the client.
	authPath := filepath.Join(dir, "auth.json")
	if err := SetOAuth(t.Context(), authPath, "openai", Entry{Access: "a", Refresh: "r", ClientID: "oaiapp_x"}); err != nil {
		t.Fatal(err)
	}
	if err := (Logins{Path: authPath}).Delete(t.Context(), "openai"); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Load(t.Context(), "openai"); err != nil || got != openai {
		t.Errorf("after the session's Delete: %+v, %v; want %+v", got, err, openai)
	}

	if err := c.Save(t.Context(), "openai", ClientRecord{}); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Load(t.Context(), "openai"); err != nil || got != (ClientRecord{}) {
		t.Errorf("after a zero save: %+v, %v; want the zero record", got, err)
	}
	if got, err := c.Load(t.Context(), "other"); err != nil || got != other {
		t.Errorf("Load other: %+v, %v; want %+v", got, err, other)
	}
	if raw, _ := os.ReadFile(c.Path); strings.Contains(string(raw), "openai") {
		t.Errorf("the member stayed:\n%s", raw)
	}
}

// A file that is not an object of records is an error naming it, and a
// save leaves it as it was.
func TestClientsBadFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "clients.json")
	const content = `{"openai": `
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	c := Clients{Path: path}
	if _, err := c.Load(t.Context(), "openai"); err == nil || !strings.Contains(err.Error(), path) {
		t.Errorf("Load: %v, want an error naming the file", err)
	}
	if err := c.Save(t.Context(), "openai", ClientRecord{ClientID: "x"}); err == nil {
		t.Error("Save over a bad file succeeded")
	}
	if got, _ := os.ReadFile(path); string(got) != content {
		t.Errorf("the file became %q", got)
	}
}
