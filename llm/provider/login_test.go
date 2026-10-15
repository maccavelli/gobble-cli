package provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"

	"github.com/maccavelli/gobble-cli/llm"
)

// memLogins is a LoginStore in memory, counting refresh locks.
type memLogins struct {
	mu     sync.Mutex
	logins map[string]*Login
	locks  int
}

func (m *memLogins) Load(_ context.Context, id string) (*Login, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if l, ok := m.logins[id]; ok {
		c := *l
		return &c, nil
	}
	return nil, nil
}

func (m *memLogins) Save(_ context.Context, id string, l *Login) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.logins == nil {
		m.logins = map[string]*Login{}
	}
	c := *l
	m.logins[id] = &c
	return nil
}

func (m *memLogins) Delete(_ context.Context, id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.logins, id)
	return nil
}

func (m *memLogins) LockRefresh(context.Context, string) (func(), error) {
	m.mu.Lock()
	m.locks++
	m.mu.Unlock()
	return func() {}, nil
}

// testIssuer is an xAI-shaped OIDC issuer in process: discovery, an ES256
// JWKS, device and token endpoints whose ID tokens it signs, revocation, and
// Kilo's device endpoints. Nothing leaves the machine.
type testIssuer struct {
	*httptest.Server
	key *ecdsa.PrivateKey

	mu        sync.Mutex
	nonce     string
	deny      bool
	revokeErr bool
	revoked   []string
	kiloPolls int
}

func newTestIssuer(t *testing.T) *testIssuer {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ti := &testIssuer{key: key}
	mux := http.NewServeMux()
	ti.Server = httptest.NewServer(mux)
	t.Cleanup(ti.Close)
	writeJSON := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.MarshalWrite(w, v); err != nil {
			t.Error(err)
		}
	}
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"issuer": ti.URL, "authorization_endpoint": ti.URL + "/oauth2/authorize", "token_endpoint": ti.URL + "/oauth2/token",
			"device_authorization_endpoint": ti.URL + "/oauth2/device/code", "jwks_uri": ti.URL + "/jwks",
			"revocation_endpoint": ti.URL + "/oauth2/revoke", "id_token_signing_alg_values_supported": []string{"ES256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"keys": []map[string]string{{"kty": "EC", "crv": "P-256", "kid": "ec-1",
			"x": b64(key.X.FillBytes(make([]byte, 32))), "y": b64(key.Y.FillBytes(make([]byte, 32)))}}})
	})
	mux.HandleFunc("POST /oauth2/device/code", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"device_code": "dc", "user_code": "ABCD-1234",
			"verification_uri": ti.URL + "/device", "expires_in": 60, "interval": 1})
	})
	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		ti.mu.Lock()
		deny, nonce := ti.deny, ti.nonce
		ti.mu.Unlock()
		if deny {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "access_denied"})
			return
		}
		claims := map[string]any{"iss": ti.URL, "aud": r.FormValue("client_id"), "sub": "user-1",
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix()}
		if r.FormValue("grant_type") == "authorization_code" {
			claims["nonce"] = nonce
		}
		writeJSON(w, http.StatusOK, map[string]any{"access_token": "access-1", "refresh_token": "refresh-1",
			"expires_in": 3600, "id_token": ti.sign(t, claims)})
	})
	mux.HandleFunc("POST /oauth2/revoke", func(w http.ResponseWriter, r *http.Request) {
		ti.mu.Lock()
		defer ti.mu.Unlock()
		ti.revoked = append(ti.revoked, r.FormValue("token"))
		if ti.revokeErr {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	mux.HandleFunc("POST /api/device-auth/codes", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"code": "KILO-1", "verificationUrl": ti.URL + "/kilo", "expiresIn": 60})
	})
	mux.HandleFunc("GET /api/device-auth/codes/KILO-1", func(w http.ResponseWriter, _ *http.Request) {
		ti.mu.Lock()
		ti.kiloPolls++
		polls, deny := ti.kiloPolls, ti.deny
		ti.mu.Unlock()
		switch {
		case deny:
			w.WriteHeader(http.StatusForbidden)
		case polls == 1:
			w.WriteHeader(http.StatusAccepted)
		default:
			writeJSON(w, http.StatusOK, map[string]string{"token": "kilo-token-0123456789"})
		}
	})
	return ti
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// with changes or reads the issuer's state under its lock.
func (ti *testIssuer) with(f func()) {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	f()
}

// sign is claims as an ES256 JWT under the issuer's key.
func (ti *testIssuer) sign(t *testing.T, claims map[string]any) string {
	t.Helper()
	h, err := json.Marshal(map[string]string{"alg": "ES256", "typ": "JWT", "kid": "ec-1"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := b64(h) + "." + b64(c)
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, ti.key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + b64(append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...))
}

// A Login goes to the SDK's session and back unchanged, FedRAMP too, and
// the adapter is the SDK's TokenStore and RefreshLocker over the store.
func TestLoginStoreAdapter(t *testing.T) {
	in := &Login{Access: "secret-access", Refresh: "secret-refresh", Expires: time.UnixMilli(1781000000000), Issuer: "https://auth.x.ai",
		ClientID: "cid", AccountID: "acct", TokenURL: "https://auth.x.ai/oauth2/token", FedRAMP: true}
	store := &memLogins{}
	a := sdkStore{store}
	if err := a.Save(t.Context(), llmprovider.ProviderGrok, in.session("grok", nil, nil, nil)); err != nil {
		t.Fatal(err)
	}
	s, err := a.Load(t.Context(), llmprovider.ProviderGrok)
	if err != nil || s == nil {
		t.Fatalf("Load: %v, %v", s, err)
	}
	if got := loginOf(s); *got != *in {
		t.Errorf("round trip: %v, want %v", got, in)
	}
	if s.Store == nil {
		t.Error("a loaded session saves nowhere")
	}
	unlock, err := a.LockRefresh(t.Context(), llmprovider.ProviderGrok)
	if err != nil || store.locks != 1 {
		t.Fatalf("LockRefresh: %v, %d locks", err, store.locks)
	}
	unlock()
	if strings.Contains(in.String(), "secret") {
		t.Errorf("String shows a token: %s", in)
	}
	if got, _ := a.Load(t.Context(), llmprovider.ProviderOpenAI); got != nil {
		t.Errorf("a provider with no login: %v", got)
	}
}

// Grok's device login shows its code, waits for approval and saves the
// verified session; a denial is llm.ErrAuth and saves nothing.
func TestGrokDeviceLogin(t *testing.T) {
	ti := newTestIssuer(t)
	store := &memLogins{}
	var shown DeviceCode
	show := func(d DeviceCode) error { shown = d; return nil }
	if err := DeviceLogin(t.Context(), "grok", LoginOptions{Origin: ti.URL}, show, store); err != nil {
		t.Fatal(err)
	}
	if shown.Code != "ABCD-1234" || shown.URL != ti.URL+"/device" {
		t.Errorf("shown %+v", shown)
	}
	l := store.logins["grok"]
	if l == nil || l.Access != "access-1" || l.Refresh != "refresh-1" || l.Issuer != ti.URL || l.TokenURL != ti.URL+"/oauth2/token" ||
		l.ClientID != auth.DefaultGrokOAuthClientID || time.Until(l.Expires) < 50*time.Minute {
		t.Errorf("saved %v", l)
	}

	ti.with(func() { ti.deny = true })
	denied := &memLogins{}
	if err := DeviceLogin(t.Context(), "grok", LoginOptions{Origin: ti.URL}, show, denied); !errors.Is(err, llm.ErrAuth) {
		t.Errorf("a denial: %v, want llm.ErrAuth", err)
	}
	if len(denied.logins) != 0 {
		t.Errorf("a denial saved %v", denied.logins)
	}
	if err := DeviceLogin(t.Context(), "openai", LoginOptions{Origin: ti.URL}, show, denied); err == nil {
		t.Error("openai's OAuth login was not refused")
	}
	stop := errors.New("stop")
	ti.with(func() { ti.deny = false })
	if err := DeviceLogin(t.Context(), "grok", LoginOptions{Origin: ti.URL}, func(DeviceCode) error { return stop }, denied); !errors.Is(err, stop) ||
		len(denied.logins) != 0 {
		t.Errorf("show's error: %v, saved %v", err, denied.logins)
	}
}

// authorizeParams reads an authorize URL's redirect, state and nonce, and
// gives the issuer the nonce for its next ID token.
func (ti *testIssuer) authorizeParams(t *testing.T, raw string) (redirect, state string) {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	ti.with(func() { ti.nonce = q.Get("nonce") })
	return q.Get("redirect_uri"), q.Get("state")
}

// Grok's browser login gives OpenURL the authorize URL and saves the
// session when the browser reaches the callback, or when the user pastes
// the URL it ended on.
func TestGrokBrowserLogin(t *testing.T) {
	ti := newTestIssuer(t)
	store := &memLogins{}
	var opened string
	browser := func(u string) error {
		opened = u
		redirect, state := ti.authorizeParams(t, u)
		go func() {
			resp, err := http.Get(redirect + "?code=c1&state=" + url.QueryEscape(state)) //nolint:noctx // the test's browser
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
	if err := BrowserLogin(t.Context(), "grok", LoginOptions{Origin: ti.URL, OpenURL: browser}, store); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(opened, ti.URL+"/oauth2/authorize?") || store.logins["grok"] == nil || store.logins["grok"].Access != "access-1" {
		t.Errorf("opened %q, saved %v", opened, store.logins["grok"])
	}

	pasted := &memLogins{}
	urls := make(chan string, 1)
	o := LoginOptions{
		Origin:  ti.URL,
		OpenURL: func(u string) error { urls <- u; return nil },
		InputCode: func(ctx context.Context) (string, error) {
			select {
			case u := <-urls:
				redirect, state := ti.authorizeParams(t, u)
				return redirect + "?code=c2&state=" + url.QueryEscape(state), nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		},
	}
	if err := BrowserLogin(t.Context(), "grok", o, pasted); err != nil || pasted.logins["grok"] == nil {
		t.Errorf("a pasted redirect: %v, saved %v", err, pasted.logins)
	}
}

// Kilo's device login waits through a pending poll and gives the token as
// a key; a denial is llm.ErrAuth; only kilo has one. Each poll waits for
// the SDK's 3 s.
func TestKeyLogin(t *testing.T) {
	ti := newTestIssuer(t)
	var shown DeviceCode
	key, err := KeyLogin(t.Context(), "kilo", LoginOptions{Origin: ti.URL}, func(d DeviceCode) error { shown = d; return nil })
	if err != nil || key != "kilo-token-0123456789" {
		t.Fatalf("key %q, %v", key, err)
	}
	var polls int
	ti.with(func() { polls = ti.kiloPolls })
	if shown.Code != "KILO-1" || shown.URL != ti.URL+"/kilo" || polls != 2 {
		t.Errorf("shown %+v after %d polls", shown, polls)
	}
	ti.with(func() { ti.deny = true })
	if _, err := KeyLogin(t.Context(), "kilo", LoginOptions{Origin: ti.URL}, func(DeviceCode) error { return nil }); !errors.Is(err, llm.ErrAuth) {
		t.Errorf("a denial: %v, want llm.ErrAuth", err)
	}
	if _, err := KeyLogin(t.Context(), "openai", LoginOptions{Origin: ti.URL}, func(DeviceCode) error { return nil }); err == nil {
		t.Error("openai's key login was not refused")
	}
}

// Logout revokes the stored refresh token at the issuer, then deletes the
// login; a failed revocation is reported, and the login still deleted.
func TestLogout(t *testing.T) {
	ti := newTestIssuer(t)
	store := &memLogins{}
	login := &Login{Access: "a", Refresh: "r-1", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
		ClientID: auth.DefaultGrokOAuthClientID, TokenURL: ti.URL + "/oauth2/token"}
	if err := store.Save(t.Context(), "grok", login); err != nil {
		t.Fatal(err)
	}
	revoke, err := Logout(t.Context(), "grok", LoginOptions{}, store)
	if revoke != nil || err != nil || store.logins["grok"] != nil {
		t.Errorf("logout: %v, %v, stored %v", revoke, err, store.logins["grok"])
	}
	var revoked []string
	ti.with(func() { revoked = append(revoked, ti.revoked...) })
	if len(revoked) == 0 || revoked[0] != "r-1" {
		t.Errorf("revoked %q, want the refresh token", revoked)
	}
	ti.with(func() { ti.revokeErr = true })
	if err := store.Save(t.Context(), "grok", login); err != nil {
		t.Fatal(err)
	}
	revoke, err = Logout(t.Context(), "grok", LoginOptions{}, store)
	if revoke == nil || err != nil || store.logins["grok"] != nil {
		t.Errorf("a failed revocation: %v, %v, stored %v; want it reported and the login deleted", revoke, err, store.logins["grok"])
	}
}

// With no key, a listing goes with the stored login's token; with a key,
// with the key (0005-PLAN F4-3b decision 9).
func TestListLogin(t *testing.T) {
	c, o := cannedClient(nil)
	o.Logins = &memLogins{logins: map[string]*Login{"grok": {Access: "access-0", Refresh: "r", Expires: time.Now().Add(time.Hour),
		ClientID: "cid", TokenURL: "https://auth.test/oauth2/token"}}}
	if _, err := List(t.Context(), "grok", "", o); err != nil {
		t.Fatal(err)
	}
	if _, auth := c.requests(); len(auth) == 0 || auth[0] != "Bearer access-0" {
		t.Fatalf("listing with a login: authorization %q, want the login's token", auth)
	}
	n := len(c.seen)
	if _, err := List(t.Context(), "grok", "k", o); err != nil {
		t.Fatal(err)
	}
	if _, auth := c.requests(); len(auth) <= n || auth[n] != "Bearer k" {
		t.Errorf("listing with a key: authorization %q, want the key's", auth[n:])
	}
}

// Pi's Grok login becomes gobble's with the SDK's issuer and client id, which
// Pi signs in with and does not store; no other provider's login is
// imported (0005-PLAN F4-3c decision 7).
func TestPiLogin(t *testing.T) {
	exp := time.UnixMilli(1781000000000)
	l, ok := PiLogin("grok", "a", "r", exp)
	if !ok || l.Access != "a" || l.Refresh != "r" || !l.Expires.Equal(exp) ||
		l.Issuer != auth.DefaultGrokOAuthIssuer || l.ClientID != auth.DefaultGrokOAuthClientID || l.TokenURL != "" {
		t.Errorf("grok: %v, %t", l, ok)
	}
	for _, id := range []string{"xai", "kilo", "claude", "openai"} {
		if l, ok := PiLogin(id, "a", "r", exp); ok || l != nil {
			t.Errorf("%s: %v, %t; want none", id, l, ok)
		}
	}
}
