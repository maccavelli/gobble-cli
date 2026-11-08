package provider

import (
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// blockingIssuer is a token endpoint in process whose refresh waits for
// release, then rotates the refresh token to "refresh-2". hit is closed by
// the first refresh. Nothing leaves the machine.
func blockingIssuer(t *testing.T) (url string, hit chan struct{}, release func()) {
	t.Helper()
	hit, released := make(chan struct{}), make(chan struct{})
	var once sync.Once
	release = func() { once.Do(func() { close(released) }) }
	var first sync.Once
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		first.Do(func() { close(hit) })
		select {
		case <-released:
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.MarshalWrite(w, map[string]any{"access_token": "access-2", "refresh_token": "refresh-2", "expires_in": 3600}); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(release) // runs before srv.Close, so a failed test never leaves the handler waiting
	return srv.URL + "/oauth2/token", hit, release
}

// expiredGrok is a store holding a Grok login whose access token has
// expired, so its first Token refreshes at tokenURL.
func expiredGrok(tokenURL string) *memLogins {
	return &memLogins{logins: map[string]*Login{"grok": {Access: "access-1", Refresh: "refresh-1",
		Expires: time.Now().Add(-time.Hour), ClientID: "cid", TokenURL: tokenURL}}}
}

// One Sessions gives a provider's calls one session, loaded once; without
// it, each call makes its own; Drop makes the next call load the stored
// login again (0046-MADR D8).
func TestSessionsShareOneSession(t *testing.T) {
	store := expiredGrok("https://auth.test/oauth2/token")
	s := &Sessions{}
	o := Options{Logins: store, Sessions: s}
	a, err := o.login(t.Context(), "grok")
	if err != nil || a == nil {
		t.Fatalf("first login: %v, %v", a, err)
	}
	if b, err := o.login(t.Context(), "grok"); err != nil || b != a {
		t.Fatalf("second login through the same Sessions: %p (%v), want the first, %p", b, err, a)
	}
	if held := s.Held(); !slices.Equal(held, []string{"grok"}) {
		t.Errorf("held %q, want [grok]", held)
	}
	plain := Options{Logins: store}
	p1, _ := plain.login(t.Context(), "grok")
	p2, _ := plain.login(t.Context(), "grok")
	if p1 == nil || p1 == p2 {
		t.Errorf("without Sessions: %p and %p, want two sessions", p1, p2)
	}
	s.Drop("grok")
	if len(s.Held()) != 0 {
		t.Errorf("held %q after Drop, want none", s.Held())
	}
	if c, err := o.login(t.Context(), "grok"); err != nil || c == nil || c == a {
		t.Errorf("a login after Drop: %p (%v), want a new session, not %p", c, err, a)
	}
}

// noNetwork answers every request with an error.
type noNetwork struct{}

func (noNetwork) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("no network in tests")
}

// ModelInfoWith asks for a login's token through the run's session: after
// it, the run's Sessions holds the provider's. Its cache is closed first, so
// the SDK asks for the token and starts no fetch (0046-MADR D8).
func TestModelInfoWithUsesTheSessions(t *testing.T) {
	store := expiredGrok("https://auth.test/oauth2/token")
	store.logins["grok"].Expires = time.Now().Add(time.Hour) // a valid token: no refresh
	c := &Cache{}
	if err := c.Close(t.Context()); err != nil {
		t.Fatal(err)
	}
	s := &Sessions{}
	_ = ModelInfoWith(Options{Logins: store, Sessions: s, Cache: c, HTTPClient: &http.Client{Transport: noNetwork{}}}, "grok", "grok-4.6", nil)
	if held := s.Held(); !slices.Equal(held, []string{"grok"}) {
		t.Fatalf("held %q after ModelInfoWith with a login, want [grok]", held)
	}
}

// Sessions.Wait waits for a held session's refresh in flight: with a
// context that ends first, it returns an error naming the provider and
// matching the context's; once the refresh ends, it returns nil, and the
// store holds the rotated refresh token (0046-MADR D8).
func TestSessionsWait(t *testing.T) {
	tokenURL, hit, release := blockingIssuer(t)
	store := expiredGrok(tokenURL)
	s := &Sessions{}
	sess, err := Options{Logins: store, Sessions: s}.login(t.Context(), "grok")
	if err != nil || sess == nil {
		t.Fatalf("login: %v, %v", sess, err)
	}
	gone, cancel := context.WithCancel(t.Context())
	go func() { _, _ = sess.Token(gone) }() // a caller that stops waiting; the refresh goes on
	select {
	case <-hit:
	case <-time.After(5 * time.Second):
		t.Fatal("no refresh reached the issuer")
	}
	cancel()
	short, stop := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer stop()
	if err := s.Wait(short); !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(fmt.Sprint(err), "grok") {
		t.Fatalf("Wait with a refresh blocked: %v, want the deadline, naming grok", err)
	}
	release()
	long, stop2 := context.WithTimeout(t.Context(), 5*time.Second)
	defer stop2()
	if err := s.Wait(long); err != nil {
		t.Fatalf("Wait once the refresh can end: %v", err)
	}
	if l, _ := store.Load(t.Context(), "grok"); l == nil || l.Refresh != "refresh-2" {
		t.Fatalf("stored login %+v after Wait, want the rotated refresh token, refresh-2", l)
	}
}
