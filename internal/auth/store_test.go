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
	if err := SetKey(t.Context(), path, "openai", "new", ""); err != nil {
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
	if err := SetKey(t.Context(), path, "gemini", "g", ""); err != nil {
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

// SetKey keeps the fields gobble does not read, as SetOAuth does, writes a
// workspace and removes one it is not given, and drops an oauth login's
// fields (0035-MADR D5).
func TestStoreKeyKeepsUnknownFields(t *testing.T) {
	path := writeAuth(t, piFile)
	if err := SetKey(t.Context(), path, "cloudflare", "$CF_KEY2", "wrkspc_1"); err != nil {
		t.Fatal(err)
	}
	if err := SetKey(t.Context(), path, "anthropic", "sk", ""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"key": "$CF_KEY2"`, `"CLOUDFLARE_ACCOUNT_ID": "x"`, `"workspace": "wrkspc_1"`} {
		if !strings.Contains(string(got), want) {
			t.Errorf("%s not kept:\n%s", want, got)
		}
	}
	for _, gone := range []string{"r-not-real", "a-not-real", "acct", "1781000000000"} {
		if strings.Contains(string(got), gone) {
			t.Errorf("the oauth login's %s kept under an api_key:\n%s", gone, got)
		}
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if e, _ := f.Entry("cloudflare"); e != (Entry{Type: TypeAPIKey, Key: "$CF_KEY2", Workspace: "wrkspc_1"}) {
		t.Errorf("cloudflare: %+v", e)
	}
	if err := SetKey(t.Context(), path, "cloudflare", "$CF_KEY2", ""); err != nil {
		t.Fatal(err)
	}
	if f, err := Load(path); err != nil {
		t.Fatal(err)
	} else if e, _ := f.Entry("cloudflare"); e.Workspace != "" {
		t.Errorf("the workspace kept after SetKey gave none: %+v", e)
	}
}

// Writers of different providers, at once, each keep the other's entry:
// every change re-reads under the lock.
func TestStoreConcurrent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	var wg sync.WaitGroup
	errs := make(chan error, 20)
	for i := range 20 {
		wg.Go(func() { errs <- SetKey(t.Context(), path, fmt.Sprintf("p%d", i), "k", "") })
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

// A ChatGPT plan session's four fields are written as resource, scope,
// idToken and hostId, read back unchanged, and removed when unset; an API
// key written over the entry removes them too (0031-MADR D7, amended
// 2026-11-10).
func TestStoreOAuthPlanFields(t *testing.T) {
	path := writeAuth(t, "{}\n")
	e := Entry{Access: "a", Refresh: "r", Expires: 1781000000000, Issuer: "https://auth.openai.com", ClientID: "oaiapp_x",
		Resource: "https://api.openai.com/v1", Scope: "openid chatgpt.tokens.use.direct", IDToken: "id-secret",
		HostID: "urn:uuid:host", Kind: "chatgpt_plan", Consents: "background_use"}
	if err := SetOAuth(t.Context(), path, "openai", e); err != nil {
		t.Fatal(err)
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	want := e
	want.Type = TypeOAuth
	if got, _ := f.Entry("openai"); got != want {
		t.Errorf("openai after SetOAuth: %+v, want %+v", got, want)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{`"resource": "https://api.openai.com/v1"`, `"scope": "openid chatgpt.tokens.use.direct"`,
		`"idToken": "id-secret"`, `"hostId": "urn:uuid:host"`, `"kind": "chatgpt_plan"`,
		`"consents": "background_use"`} {
		if n := strings.Count(string(raw), s); n != 1 {
			t.Errorf("%q written %d times in:\n%s", s, n, raw)
		}
	}
	e.Resource, e.Scope, e.IDToken, e.HostID, e.Kind, e.Consents = "", "", "", "", "", ""
	if err := SetOAuth(t.Context(), path, "openai", e); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(path); strings.Contains(string(raw), "resource") || strings.Contains(string(raw), "idToken") ||
		strings.Contains(string(raw), "kind") || strings.Contains(string(raw), "consents") {
		t.Errorf("after a save without the plan's fields:\n%s", raw)
	}
	e.Resource, e.IDToken = "https://api.openai.com/v1", "id-secret"
	if err := SetOAuth(t.Context(), path, "openai", e); err != nil {
		t.Fatal(err)
	}
	if err := SetKey(t.Context(), path, "openai", "sk-k", ""); err != nil {
		t.Fatal(err)
	}
	if raw, _ := os.ReadFile(path); strings.Contains(string(raw), "idToken") || strings.Contains(string(raw), "resource") {
		t.Errorf("a key over the plan session kept its fields:\n%s", raw)
	}
}

// A key over an oauth entry leaves none of the session's members, kind and
// consents included, and keeps every other field (0052-PLAN P2-1).
func TestSetKeyDropsSessionMembers(t *testing.T) {
	path := writeAuth(t, `{
  "openai": {
    "type": "oauth",
    "access": "a",
    "refresh": "r",
    "expires": 1781000000000,
    "kind": "chatgpt_plan",
    "consents": "background_use",
    "note": "keep me"
  }
}
`)
	if err := SetKey(t.Context(), path, "openai", "sk-k", ""); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"kind", "consents", "chatgpt_plan", "background_use", "access", "refresh"} {
		if strings.Contains(string(raw), s) {
			t.Errorf("%q survived SetKey:\n%s", s, raw)
		}
	}
	for _, s := range []string{`"type": "api_key"`, `"key": "sk-k"`, `"note": "keep me"`} {
		if !strings.Contains(string(raw), s) {
			t.Errorf("no %q in:\n%s", s, raw)
		}
	}
}

// An oauth login written over an api_key entry leaves no key and no
// workspace behind, and keeps every other field as written (0052-MADR F25).
func TestSetOAuthDropsKey(t *testing.T) {
	path := writeAuth(t, `{
  "openai": {
    "type": "api_key",
    "key": "sk-dormant",
    "workspace": "ws-dormant",
    "note": "keep me"
  },
  "other": {
    "type": "api_key",
    "key": "sk-other",
    "workspace": "ws-other"
  }
}
`)
	e := Entry{Access: "a", Refresh: "r", Expires: 1781000000000, Issuer: "https://auth.openai.com", ClientID: "oaiapp_x"}
	if err := SetOAuth(t.Context(), path, "openai", e); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"sk-dormant", "ws-dormant"} {
		if strings.Contains(string(raw), s) {
			t.Errorf("%q survived SetOAuth:\n%s", s, raw)
		}
	}
	for _, s := range []string{`"note": "keep me"`, `"access": "a"`, `"clientId": "oaiapp_x"`,
		`"key": "sk-other"`, `"workspace": "ws-other"`} {
		if !strings.Contains(string(raw), s) {
			t.Errorf("no %q in:\n%s", s, raw)
		}
	}
	f, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := f.Entry("openai")
	if got.Key != "" || got.Workspace != "" || got.Type != TypeOAuth {
		t.Errorf("openai after SetOAuth: %+v", got)
	}
}

// The file is the user's alone: 0600 on Unix, the user's DACL on Windows,
// in a directory made 0700 when it is created.
func TestStoreMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new", "auth.json")
	if err := SetKey(t.Context(), path, "openai", "k", ""); err != nil {
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
			if err := SetKey(t.Context(), path, "gemini", "g", ""); err == nil {
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

// auth.json is configuration: invalid UTF-8 or a lone surrogate escape is
// refused, never repaired (0039-MADR D3; Confirmation 8). The backslash is
// made in code (0039-PLAN C4).
func TestAuthFileStrict(t *testing.T) {
	for name, body := range map[string]string{
		"invalid byte":   "{\"openai\":{\"type\":\"api_key\",\"key\":\"k\xff\"}}",
		"lone surrogate": strings.ReplaceAll(`{"openai":{"type":"api_key","key":"k~ud83d"}}`, "~", string(rune(92))),
	} {
		path := filepath.Join(t.TempDir(), "auth.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(path); err == nil {
			t.Errorf("%s: Load accepted it", name)
		}
	}
}

// A key holding an invalid byte is refused, and auth.json is left as it was:
// a credential is never written altered (0039-MADR D3; Confirmation 8).
func TestSetKeyInvalidUTF8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "auth.json")
	if err := SetKey(t.Context(), path, "openai", "good", ""); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := SetKey(t.Context(), path, "openai", "bad\xff", ""); err == nil {
		t.Fatal("SetKey accepted a key holding an invalid byte")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("auth.json changed: %q, %v; want %q", after, err, before)
	}
}
