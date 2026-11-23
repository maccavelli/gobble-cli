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
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
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
	// origin is the issuer the discovery document and ID tokens name,
	// ti.URL when empty: a sign-in reaching the issuer through toIssuer
	// sees the SDK's own.
	origin string
	// scope, when set, is the scope its token responses grant: a ChatGPT plan
	// sign-in is refused a grant without chatgpt.tokens.use.direct.
	scope string

	mu        sync.Mutex
	nonce     string
	deny      bool
	revokeErr bool
	revoked   []string
	kiloPolls int
	// listStatus, when set, is the status every model listing answers with.
	listStatus int
	// profileStatus, when set, is the status Kilo's profile answers with;
	// otherwise it answers 200 with a personal account (0068-PLAN step 1a).
	profileStatus int
	// profileStall makes Kilo's profile answer nothing until the request
	// ends.
	profileStall bool
}

// profile makes Kilo's profile answer status: 401 refuses the key, 503 cannot
// say (go-llmprovider-sdk v1.32.0, signin.VerifyKilo).
func (ti *testIssuer) profile(status int) { ti.with(func() { ti.profileStatus = status }) }

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
		resp := map[string]any{"access_token": "access-1", "refresh_token": "refresh-1",
			"expires_in": 3600, "id_token": ti.sign(t, claims)}
		if ti.scope != "" {
			resp["scope"] = ti.scope
		}
		writeJSON(w, http.StatusOK, resp)
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
	// Kilo's profile, which the SDK reads to check a Kilo key from v1.32.0:
	// GET {origin}/api/profile (auth.KiloProfile).
	mux.HandleFunc("GET /api/profile", func(w http.ResponseWriter, r *http.Request) {
		ti.mu.Lock()
		status, stall := ti.profileStatus, ti.profileStall
		ti.mu.Unlock()
		switch {
		case stall:
			<-r.Context().Done()
		case status != 0:
			writeJSON(w, status, map[string]any{"error": map[string]string{"message": serviceText}})
		default:
			writeJSON(w, http.StatusOK, map[string]any{"user": map[string]string{"email": "person@example.test"},
				"organizations": []any{}, "hasPersonalAccount": true})
		}
	})
	// A provider's model listing: refused, with the service's own text, for a
	// key that begins "refused"; otherwise one model.
	mux.HandleFunc("GET /v1/", func(w http.ResponseWriter, r *http.Request) {
		ti.mu.Lock()
		status := ti.listStatus
		ti.mu.Unlock()
		if status != 0 {
			writeJSON(w, status, map[string]any{"error": map[string]string{"message": serviceText}})
			return
		}
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
		ClientID: auth.DefaultGrokOAuthClientID, TokenURL: ti.URL + "/oauth2/token", Kind: string(auth.SessionGrok)}
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

// refreshes are the requests among reqs that refresh a token.
func refreshes(reqs []seenRequest) []seenRequest {
	var out []seenRequest
	for _, r := range reqs {
		if v, _ := url.ParseQuery(r.form); r.path == "/oauth2/token" && v.Get("grant_type") == "refresh_token" {
			out = append(out, r)
		}
	}
	return out
}

// A stored Grok login past its expiry is refreshed under gobble's name by
// each of New's first request, List and ModelInfoWith, with the options'
// ClientName and ClientVersion, and under the SDK's with none. ModelInfoWith
// runs first on a fresh Sessions, no provider built before it, so only the
// SDK's hand-off from the WithClientInfo gobble passes can name its refresh;
// the listing it starts in the background is named too (0051-MADR D9, N6).
func TestStoredLoginNamedForRefresh(t *testing.T) {
	// each runs one call over a fresh expired login whose requests, all sent
	// to one in-process issuer, a recorder sees.
	calls := map[string]func(t *testing.T, o Options, rec *uaRecorder){
		"New": func(t *testing.T, o Options, _ *uaRecorder) {
			p, err := New("grok", o)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = drain(t, p, &llm.Request{Model: "grok-4.6", Messages: []llm.Message{text(llm.RoleUser, "hi")}}) //nolint:errcheck // the refresh is what is read
		},
		"List": func(t *testing.T, o Options, _ *uaRecorder) {
			_, _ = List(t.Context(), "grok", "", o) //nolint:errcheck // the refresh is what is read
		},
		"ModelInfoWith": func(t *testing.T, o Options, rec *uaRecorder) {
			c := &Cache{}
			o.Cache, o.Sessions = c, &Sessions{}
			_ = ModelInfoWith(o, "grok", "grok-4.6", nil)
			// The listing it started in the background reaches the issuer
			// before the cache's close ends it.
			listed := func() bool {
				return slices.ContainsFunc(rec.requests(), func(r seenRequest) bool { return r.method == "GET" && r.path == "/v1/models" })
			}
			for deadline := time.Now().Add(5 * time.Second); !listed() && time.Now().Before(deadline); {
				time.Sleep(10 * time.Millisecond)
			}
			if err := o.Sessions.Wait(t.Context()); err != nil {
				t.Fatal(err)
			}
			if err := c.Close(t.Context()); err != nil {
				t.Fatal(err)
			}
		},
	}
	for _, name := range []string{"New", "List", "ModelInfoWith"} {
		for _, named := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s named %t", name, named), func(t *testing.T) {
				ti := newTestIssuer(t)
				u, err := url.Parse(ti.URL)
				if err != nil {
					t.Fatal(err)
				}
				rec := &uaRecorder{}
				rec.atIssuer(ti)
				o := Options{Logins: expiredGrok(ti.URL + "/oauth2/token"), HTTPClient: &http.Client{Transport: toIssuer{u}},
					Retry: &llm.Retry{Off: true}}
				if named {
					o.ClientName, o.ClientVersion = "gobble", "1.2.3-test"
				}
				calls[name](t, o, rec)
				reqs := rec.requests()
				ref := refreshes(reqs)
				if len(ref) == 0 {
					t.Fatalf("%s: no refresh among %+v", name, reqs)
				}
				if named {
					wantNamed(t, name+"'s refresh", ref)
					if name == "ModelInfoWith" {
						wantNamed(t, name+"'s requests, its background listing among them", reqs)
						if !slices.ContainsFunc(reqs, func(r seenRequest) bool { return r.method == "GET" && r.path == "/v1/models" }) {
							t.Errorf("%s started no listing among %+v", name, reqs)
						}
					}
				} else {
					wantUnnamed(t, name+"'s refresh with no name", ref)
				}
			})
		}
	}
}

// Logout of a stored Grok login, and of a stored ChatGPT plan session, names
// gobble on the discovery and the revocation, whose paths and forms are those
// of a logout with no name, which sends the SDK's User-Agent (0051-MADR D9,
// N7).
func TestLogoutRevokesAsGobble(t *testing.T) {
	logins := map[string]func(ti *testIssuer) *Login{
		"grok": func(ti *testIssuer) *Login {
			return &Login{Access: "a", Refresh: "r-1", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
				ClientID: auth.DefaultGrokOAuthClientID, TokenURL: ti.URL + "/oauth2/token", Kind: string(auth.SessionGrok)}
		},
		"openai": func(ti *testIssuer) *Login {
			return &Login{Access: "a", Refresh: "plan-r", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
				ClientID: "oaiapp_gobbletest", TokenURL: ti.URL + "/oauth2/token", Resource: "https://api.openai.com/v1",
				Scope: "openid chatgpt.tokens.use.direct", HostID: "urn:uuid:host", Kind: string(auth.SessionChatGPTPlan)}
		},
	}
	for id, login := range logins {
		t.Run(id, func(t *testing.T) {
			run := func(o LoginOptions) []seenRequest {
				t.Helper()
				ti := newTestIssuer(t) // one issuer for each run: the SDK caches discovery by issuer
				rec := &uaRecorder{}
				o.HTTPClient = &http.Client{Transport: rec}
				store := &memLogins{}
				if err := store.Save(t.Context(), id, login(ti)); err != nil {
					t.Fatal(err)
				}
				if revoke, err := Logout(t.Context(), id, o, store); revoke != nil || err != nil {
					t.Fatalf("logout: %v, %v", revoke, err)
				}
				return rec.requests()
			}
			named := run(LoginOptions{ClientName: "gobble", ClientVersion: "1.2.3-test"})
			plain := run(LoginOptions{})
			wantNamed(t, "the named logout", named)
			wantUnnamed(t, "the unnamed logout", plain)
			var paths []string
			for _, r := range named {
				paths = append(paths, r.method+" "+r.path+" "+r.form)
			}
			var want []string
			for _, r := range plain {
				want = append(want, r.method+" "+r.path+" "+r.form)
			}
			if !slices.Equal(paths, want) {
				t.Errorf("the named logout sent %q, the unnamed %q; want the same requests", paths, want)
			}
			if !slices.ContainsFunc(named, func(r seenRequest) bool { return r.path == "/oauth2/revoke" }) ||
				!slices.ContainsFunc(named, func(r seenRequest) bool { return r.path == "/.well-known/openid-configuration" }) {
				t.Errorf("the named logout sent %+v, want a discovery and a revocation", named)
			}
		})
	}
}

// Logout revokes only a session gobble accepts: a stored OpenAI login of
// another application's kind (the Codex client's ChatGPT login) is deleted
// and never revoked, as gobble holds no right to revoke it; a Grok session
// and a ChatGPT plan session are revoked, then deleted (0051-MADR D2).
func TestLogoutRevokesOnlyAccepted(t *testing.T) {
	for _, c := range []struct {
		name, id string
		login    func(ti *testIssuer) *Login
		revoked  bool
	}{
		{"another application's ChatGPT login", "openai", func(ti *testIssuer) *Login {
			return &Login{Access: "a", Refresh: "codex-r", Expires: time.Now().Add(time.Hour), Issuer: ti.URL, ClientID: "app_codex",
				TokenURL: ti.URL + "/oauth2/token", Kind: string(auth.SessionChatGPT)}
		}, false},
		{"a Grok session", "grok", func(ti *testIssuer) *Login {
			return &Login{Access: "a", Refresh: "grok-r", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
				ClientID: auth.DefaultGrokOAuthClientID, TokenURL: ti.URL + "/oauth2/token", Kind: string(auth.SessionGrok)}
		}, true},
		{"a ChatGPT plan session", "openai", func(ti *testIssuer) *Login {
			return &Login{Access: "a", Refresh: "plan-r", Expires: time.Now().Add(time.Hour), Issuer: ti.URL, ClientID: "oaiapp_gobbletest",
				TokenURL: ti.URL + "/oauth2/token", Resource: "https://api.openai.com/v1", Kind: string(auth.SessionChatGPTPlan)}
		}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			ti := newTestIssuer(t)
			store := &memLogins{}
			login := c.login(ti)
			if err := store.Save(t.Context(), c.id, login); err != nil {
				t.Fatal(err)
			}
			revoke, err := Logout(t.Context(), c.id, LoginOptions{}, store)
			if revoke != nil || err != nil || store.logins[c.id] != nil {
				t.Fatalf("logout: %v, %v, stored %v; want it deleted", revoke, err, store.logins[c.id])
			}
			var revoked []string
			ti.with(func() { revoked = append(revoked, ti.revoked...) })
			if got := slices.Contains(revoked, login.Refresh); got != c.revoked || !c.revoked && len(revoked) != 0 {
				t.Errorf("revoked %q, want %s revoked: %v", revoked, login.Refresh, c.revoked)
			}
		})
	}
}

// Logout reads revocability from the method that signed the stored session in,
// as its kind names it, not from the provider's other methods: a stored
// ChatGPT plan session whose method is not revocable is deleted and never
// revoked, though the provider offers another method that is (0051-MADR D2).
func TestLogoutReadsTheStoredMethod(t *testing.T) {
	ti := newTestIssuer(t)
	store := &memLogins{}
	login := &Login{Access: "a", Refresh: "plan-r", Expires: time.Now().Add(time.Hour), Issuer: ti.URL, ClientID: "oaiapp_gobbletest",
		TokenURL: ti.URL + "/oauth2/token", Resource: "https://api.openai.com/v1", Kind: string(auth.SessionChatGPTPlan)}
	if err := store.Save(t.Context(), "openai", login); err != nil {
		t.Fatal(err)
	}
	saved := offeredFor
	t.Cleanup(func() { offeredFor = saved })
	offeredFor = func(id string) []authcontract.Method {
		return []authcontract.Method{
			{ID: "chatgpt_plan", Yields: authcontract.YieldsOAuthSession, ClientIdentity: authcontract.ClientAppRegistered, Revocable: false},
			{ID: "browser_oauth", Yields: authcontract.YieldsOAuthSession, ClientIdentity: authcontract.ClientVendorBorrowed, Revocable: true},
		}
	}
	if !Revocable("openai") {
		t.Fatal("the fixture's provider offers no revocable method")
	}
	revoke, err := Logout(t.Context(), "openai", LoginOptions{}, store)
	var revoked []string
	ti.with(func() { revoked = append(revoked, ti.revoked...) })
	if revoke != nil || err != nil || store.logins["openai"] != nil || len(revoked) != 0 {
		t.Errorf("logout: %v, %v, stored %v, revoked %q; want it deleted and never revoked", revoke, err, store.logins["openai"], revoked)
	}
}

// tokensSeen records the credentials each request to an issuer carried: its
// Authorization header and its form's refresh_token.
type tokensSeen struct {
	mu   sync.Mutex
	seen []string
}

// at makes ts see each request that reaches ti, whatever client sent it.
func (ts *tokensSeen) at(ti *testIssuer) {
	next := ti.Config.Handler
	ti.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm() //nolint:errcheck // a malformed form is recorded as it parsed
		ts.mu.Lock()
		ts.seen = append(ts.seen, r.Header.Get("Authorization"), r.Form.Get("refresh_token"))
		ts.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

// carried reports whether any request carried value.
func (ts *tokensSeen) carried(value string) bool {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return slices.ContainsFunc(ts.seen, func(s string) bool { return strings.Contains(s, value) })
}

// A held session adopts its own rotation from the store, through gobble's
// store, but refuses a store that now holds another sign-in (another client):
// its next refresh fails with an auth failure, gobble says LoginRejected's
// sentence, and neither of the other sign-in's tokens is sent (0068-MADR V5;
// go-llmprovider-sdk 0082-MADR).
func TestRefreshRefusesAnotherSignIn(t *testing.T) {
	ti := newTestIssuer(t)
	ts := &tokensSeen{}
	ts.at(ti)
	tokenURL := ti.URL + "/oauth2/token"
	a := &Login{Access: "a-access", Refresh: "a-refresh", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
		ClientID: auth.DefaultGrokOAuthClientID, TokenURL: tokenURL, Kind: string(auth.SessionGrok)}
	store := &memLogins{logins: map[string]*Login{"grok": a}}
	u, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	o := Options{Logins: store, Sessions: &Sessions{}, HTTPClient: &http.Client{Transport: toIssuer{u}}}
	held, err := o.login(t.Context(), "grok")
	if err != nil || held == nil {
		t.Fatalf("the held session: %v, %v", held, err)
	}
	// Its own rotations, twice: the second reads back the first, which gobble's
	// store kept.
	for i := range 2 {
		held.Invalidate()
		if tok, err := held.Token(t.Context()); err != nil || tok.Value != "access-1" {
			t.Fatalf("own rotation %d: %+v, %v; want access-1", i+1, tok, err)
		}
	}
	b := &Login{Access: "b-access", Refresh: "b-refresh", Expires: time.Now().Add(time.Hour), Issuer: ti.URL,
		ClientID: "another-client", TokenURL: tokenURL, Kind: string(auth.SessionGrok)}
	if err := store.Save(t.Context(), "grok", b); err != nil {
		t.Fatal(err)
	}
	held.Invalidate()
	p, err := New("grok", o)
	if err != nil {
		t.Fatal(err)
	}
	var got error
	for _, err := range p.Stream(t.Context(), &llm.Request{Model: "grok-4.6", Messages: []llm.Message{text(llm.RoleUser, "hi")}}) {
		if err != nil {
			got = err
			break
		}
	}
	if got == nil || !strings.HasPrefix(got.Error(), LoginRejected("grok")) || !errors.Is(got, llm.ErrAuth) {
		t.Errorf("a turn after the store took another sign-in: %v; want an auth failure led by %q", got, LoginRejected("grok"))
	}
	if ts.carried("b-access") || ts.carried("b-refresh") {
		t.Error("a request carried the other sign-in's token")
	}
}

// Pi's imported Grok login, stored with no kind, still refreshes after the
// bump: the SDK infers its kind from its issuer and client, and the rotation
// is saved (0068-MADR V6; a guard, plant P14).
func TestPiGrokLoginRefreshesAfterTheBump(t *testing.T) {
	_, o := signInIssuer(t)
	l, ok := PiLogin("grok", "pi-access", "pi-refresh", time.Now().Add(-time.Minute))
	if !ok {
		t.Fatal("no Pi login for grok")
	}
	store := &memLogins{logins: map[string]*Login{"grok": l}}
	sess := l.session("grok", store, o.HTTPClient, nil)
	tok, err := sess.Token(t.Context())
	if err != nil || tok.Value != "access-1" {
		t.Fatalf("Pi's grok login's refresh: %+v, %v; want access-1 from the issuer", tok, err)
	}
	if got, _ := store.Load(t.Context(), "grok"); got == nil || got.Access != "access-1" || got.Refresh != "refresh-1" {
		t.Errorf("stored %v, want the rotation saved", got)
	}
}
