package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
)

// piFile is an auth.json as Pi writes one: JSON.stringify(…, null, 2), an
// oauth entry with fields gobble does not read, and an api_key with env.
const piFile = `{
  "anthropic": {
    "type": "oauth",
    "refresh": "r-not-real",
    "access": "a-not-real",
    "expires": 1781000000000,
    "accountId": "acct"
  },
  "cloudflare": {
    "type": "api_key",
    "key": "$CF_KEY",
    "env": {
      "CLOUDFLARE_ACCOUNT_ID": "x"
    }
  },
  "openai": {
    "type": "api_key",
    "key": "old"
  }
}
`

func writeAuth(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// A Set replaces its one entry and keeps every other byte for byte, in the
// file's order; entries parse as Pi's shape (0005-PLAN F4-3a decision 4).
func TestStoreRoundTrip(t *testing.T) {
	path := writeAuth(t, piFile)
	if err := Set(t.Context(), path, "openai", Entry{Type: TypeAPIKey, Key: "new"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(piFile, `"key": "old"`, `"key": "new"`, 1)
	if string(got) != want {
		t.Fatalf("after Set:\n%s\nwant:\n%s", got, want)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.Providers(), []string{"anthropic", "cloudflare", "openai"}) {
		t.Errorf("providers %q, want the file's order", f.Providers())
	}
	if e, _ := f.Entry("anthropic"); e != (Entry{Type: TypeOAuth, Expires: 1781000000000, Access: "a-not-real", Refresh: "r-not-real", AccountID: "acct"}) {
		t.Errorf("anthropic: %+v", e)
	}
	if e, _ := f.Entry("cloudflare"); e != (Entry{Type: TypeAPIKey, Key: "$CF_KEY"}) {
		t.Errorf("cloudflare: %+v", e)
	}
	if err := Set(t.Context(), path, "gemini", Entry{Type: TypeAPIKey, Key: "g"}); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); !strings.HasSuffix(string(got), "  },\n  \"gemini\": {\n    \"type\": \"api_key\",\n    \"key\": \"g\"\n  }\n}\n") {
		t.Errorf("a new entry is not laid out as Pi lays one out:\n%s", got)
	}
	if removed, err := Delete(t.Context(), path, "cloudflare"); err != nil || !removed {
		t.Fatalf("Delete: %v, %v", removed, err)
	}
	if removed, err := Delete(t.Context(), path, "cloudflare"); err != nil || removed {
		t.Errorf("a second Delete: %v, %v; want nothing removed", removed, err)
	}
	if f, err := Load(path); err != nil || !slices.Equal(f.Providers(), []string{"anthropic", "openai", "gemini"}) {
		t.Errorf("after Delete: %v, %v", f, err)
	}
}

// Writers of different providers, at once, each keep the other's entry:
// every change re-reads under the lock.
func TestStoreConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := range 20 {
		wg.Go(func() { errs <- Set(t.Context(), path, fmt.Sprintf("p%d", i), Entry{Type: TypeAPIKey, Key: "k"}) })
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if n := len(f.Providers()); n != 20 {
		t.Errorf("%d entries, want all 20: %q", n, f.Providers())
	}
}

// SetOAuth writes an oauth login's fields into the provider's entry and
// keeps every field it does not read; fedramp is there only when true, and
// another entry is untouched (0005-PLAN F4-3b decision 3).
func TestStoreOAuthRoundTrip(t *testing.T) {
	path := writeAuth(t, `{
  "grok": {
    "type": "oauth",
    "access": "a1",
    "refresh": "r1",
    "expires": 1,
    "note": "keep me"
  },
  "openai": {
    "type": "api_key",
    "key": "k"
  }
}
`)
	e := Entry{Access: "a2", Refresh: "r2", Expires: 1781000000000, Issuer: "https://auth.x.ai", ClientID: "cid",
		AccountID: "acct", TokenURL: "https://auth.x.ai/oauth2/token", FedRAMP: true}
	if err := SetOAuth(t.Context(), path, "grok", e); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := e
	want.Type = TypeOAuth
	if got, _ := f.Entry("grok"); got != want {
		t.Errorf("grok after SetOAuth: %+v, want %+v", got, want)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`"note": "keep me"`, `"fedramp": true`, `"clientId": "cid"`, "  \"openai\": {\n    \"type\": \"api_key\",\n    \"key\": \"k\"\n  }\n"} {
		if !strings.Contains(string(raw), s) {
			t.Errorf("no %q in:\n%s", s, raw)
		}
	}
	e.FedRAMP, e.AccountID = false, ""
	if err := SetOAuth(t.Context(), path, "grok", e); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(path); strings.Contains(string(raw), "fedramp") || strings.Contains(string(raw), "accountId") ||
		!strings.Contains(string(raw), `"note": "keep me"`) {
		t.Errorf("after a save without fedramp or an account:\n%s", raw)
	}
	if err := SetOAuth(t.Context(), path, "new", Entry{Access: "a", Refresh: "r", Expires: 2}); err != nil {
		t.Fatal(err)
	}
	if f, _ := Load(path); !slices.Equal(f.Providers(), []string{"grok", "openai", "new"}) {
		t.Errorf("providers %q, want new last", f.Providers())
	}
}

// The file is the user's alone: 0600 on Unix, the user's DACL on Windows,
// in a directory made 0700 when it is created.
func TestStoreMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "auth.json")
	if err := Set(t.Context(), path, "openai", Entry{Type: TypeAPIKey, Key: "k"}); err != nil {
		t.Fatal(err)
	}
	if ok, err := appdirs.FileIsOwnerOnly(path); err != nil || !ok {
		t.Errorf("owner-only %v, %v; want the user's alone", ok, err)
	}
	if runtime.GOOS == "windows" {
		return
	}
	for p, want := range map[string]os.FileMode{path: 0o600, filepath.Dir(path): 0o700} {
		if fi, err := os.Stat(p); err != nil || fi.Mode().Perm() != want {
			t.Errorf("%s: mode %v, %v; want %v", p, fi.Mode().Perm(), err, want)
		}
	}
}

// A file that is not auth.json is an error naming it, and a change leaves
// it untouched.
func TestStoreBadFile(t *testing.T) {
	for name, content := range map[string]string{
		"not JSON":      `{"openai": `,
		"not an object": `[]`,
		"no type":       `{"openai": {"key": "k"}}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := writeAuth(t, content)
			if _, err := Load(path); err == nil || !strings.Contains(err.Error(), path) {
				t.Errorf("Load: %v, want an error naming the file", err)
			}
			if err := Set(t.Context(), path, "gemini", Entry{Type: TypeAPIKey, Key: "g"}); err == nil {
				t.Error("Set over a bad file succeeded")
			}
			if got, _ := os.ReadFile(path); string(got) != content {
				t.Errorf("the file became %q", got)
			}
		})
	}
	if f, err := Load(filepath.Join(t.TempDir(), "absent.json")); err != nil || len(f.Providers()) != 0 {
		t.Errorf("no file: %v, %v; want an empty File", f, err)
	}
}
