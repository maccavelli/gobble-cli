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
		ClientID: "cid", AccountID: "acct", TokenURL: "https://auth.x.ai/oauth2/token", FedRAMP: true,
		Kind: "chatgpt_plan", Consents: "background_use"}
	if err := l.Save(t.Context(), "grok", in); err != nil {
		t.Fatal(err)
	}
	got, err := l.Load(t.Context(), "grok")
	if err != nil || got == nil || *got != *in {
		t.Fatalf("Load: %v, %v; want %v", got, err, in)
	}
	if err := SetKey(t.Context(), l.Path, "openai", "k", ""); err != nil {
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

// A refresh waits for another's lock past the longest the SDK holds it:
// three attempts of 15 s, with 200 ms and 400 ms between them
// (go-llmprovider-sdk's 0068-MADR D3), so a waiter does not give up on a
// refresh that is still running.
func TestRefreshWaitOutlastsAHold(t *testing.T) {
	const longestHold = 3*15*time.Second + 200*time.Millisecond + 400*time.Millisecond
	if refreshPolicy.Deadline <= longestHold {
		t.Errorf("refreshPolicy.Deadline = %v, want more than a refresh's longest hold, %v", refreshPolicy.Deadline, longestHold)
	}
}
