package auth

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// A login saved through Logins loads back as it was, and a provider with no
// oauth entry has none (0005-PLAN F4-3b decision 3).
func TestLoginsLoadSave(t *testing.T) {
	l := Logins{Path: filepath.Join(t.TempDir(), "auth.json")}
	in := &provider.Login{Access: "a", Refresh: "r", Expires: time.UnixMilli(1781000000000), Issuer: "https://auth.x.ai",
		ClientID: "cid", AccountID: "acct", TokenURL: "https://auth.x.ai/oauth2/token", FedRAMP: true}
	if err := l.Save(t.Context(), "grok", in); err != nil {
		t.Fatal(err)
	}
	got, err := l.Load(t.Context(), "grok")
	if err != nil || got == nil || *got != *in {
		t.Fatalf("Load: %v, %v; want %v", got, err, in)
	}
	if err := Set(t.Context(), l.Path, "openai", Entry{Type: TypeAPIKey, Key: "k"}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"openai", "kilo"} {
		if got, err := l.Load(t.Context(), id); got != nil || err != nil {
			t.Errorf("Load(%s): %v, %v; want none", id, got, err)
		}
	}
	if err := l.Delete(t.Context(), "grok"); err != nil {
		t.Fatal(err)
	}
	if got, err := l.Load(t.Context(), "grok"); got != nil || err != nil {
		t.Errorf("after Delete: %v, %v", got, err)
	}
}

// A refresh's lock is its own: a Save while it is held, as the SDK's refresh
// makes, does not wait for it; a second refresh does (0005-PLAN F4-3b
// decision 4).
func TestRefreshLockSeparate(t *testing.T) {
	l := Logins{Path: filepath.Join(t.TempDir(), "auth.json")}
	unlock, err := l.LockRefresh(t.Context(), "grok")
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()
	if err := l.Save(ctx, "grok", &provider.Login{Access: "a", Refresh: "r"}); err != nil {
		t.Fatalf("Save while the refresh lock is held: %v", err)
	}
	short, stop := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer stop()
	if second, err := l.LockRefresh(short, "grok"); err == nil {
		second()
		t.Error("a second refresh lock was taken while the first was held")
	}
	if other, err := l.LockRefresh(ctx, "kilo"); err != nil {
		t.Errorf("another provider's refresh lock: %v", err)
	} else {
		other()
	}
}
