package provider

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
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
	// origin is the issuer the discovery document and ID tokens name,
	// ti.URL when empty: a sign-in reaching the issuer through toIssuer
	// sees the SDK's own.
	origin string

	mu        sync.Mutex
	nonce     string
	deny      bool
	revokeErr bool
	revoked   []string
	kiloPolls int
}

// issuerKey is every test issuer's signing key: the SDK caches an issuer's
// keys by its URL, so issuers answering as one origin must sign alike.
var issuerKey = sync.OnceValues(func() (*ecdsa.PrivateKey, error) { return ecdsa.GenerateKey(elliptic.P256(), rand.Reader) })

func newTestIssuer(t *testing.T) *testIssuer {
	t.Helper()
	key, err := issuerKey()
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
			"issuer": ti.base(), "authorization_endpoint": ti.base() + "/oauth2/authorize", "token_endpoint": ti.base() + "/oauth2/token",
			"device_authorization_endpoint": ti.base() + "/oauth2/device/code", "jwks_uri": ti.base() + "/jwks",
			"revocation_endpoint": ti.base() + "/oauth2/revoke", "id_token_signing_alg_values_supported": []string{"ES256"},
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
		claims := map[string]any{"iss": ti.base(), "aud": r.FormValue("client_id"), "sub": "user-1",
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
	// A provider's model listing: refused, with the service's own text, for a
	// key that begins "refused"; otherwise one model.
	mux.HandleFunc("GET /v1/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), "refused") {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"message": serviceText}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": []map[string]string{{"id": "grok-4.6", "object": "model"}}})
	})
	return ti
}

// serviceText is a refusal's own text, which gobble never shows.
const serviceText = "SERVICE-TEXT-do-not-show"

// base is the issuer the discovery document and ID tokens name.
func (ti *testIssuer) base() string {
	if ti.origin != "" {
		return ti.origin
	}
	return ti.URL
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
	// A ChatGPT plan session's four fields go to the SDK and back, and its
	// ID token is as secret as its tokens (0031-MADR D7, amended 2026-11-10).
	plan := &Login{Access: "secret-access", Refresh: "secret-refresh", Expires: time.UnixMilli(1781000000000),
		Issuer: auth.DefaultOpenAIIssuer, ClientID: "oaiapp_gobbletest", Resource: "https://api.openai.com/v1",
		Scope: "openid chatgpt.tokens.use.direct", IDToken: "secret-id-token", HostID: "urn:uuid:host"}
	if err := a.Save(t.Context(), llmprovider.ProviderOpenAI, plan.session("openai", nil, nil, nil)); err != nil {
		t.Fatal(err)
	}
	ps, err := a.Load(t.Context(), llmprovider.ProviderOpenAI)
	if err != nil || ps == nil || !ps.PlanUsage() {
		t.Fatalf("Load a plan session: %v, %v", ps, err)
	}
	if got := loginOf(ps); *got != *plan {
		t.Errorf("a plan session's round trip: %v, want %v", got, plan)
	}
	if strings.Contains(plan.String(), "secret") {
		t.Errorf("String shows a secret: %s", plan)
	}
	if got, _ := a.Load(t.Context(), llmprovider.ProviderClaude); got != nil {
		t.Errorf("a provider with no login: %v", got)
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
