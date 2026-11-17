package acpserver

import (
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
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// grokOrigin is the origin xAI's issuer answers as to the SDK's Grok sign-in
// (its default issuer): only llm/provider may import the SDK (archtest rule
// 4), so the test states it, and a different value fails every sign-in at its
// discovery.
const grokOrigin = "https://auth.x.ai"

// issuer is an OIDC issuer in process answering as origin: discovery, an
// ES256 JWKS, device and token endpoints whose ID tokens it signs, a model
// listing, and a log of every request. It is the subset of llm/provider's
// testIssuer the relay's tests use, which another package's test cannot import.
// The real engine runs against it, through provider.StartSignIn, with no
// request leaving the machine.
type issuer struct {
	*httptest.Server
	origin string
	key    *ecdsa.PrivateKey

	mu     sync.Mutex
	nonce  string
	deny   bool
	refuse bool
	// listStatus, when set, is the status every model listing answers with.
	listStatus int
	// onList, when set, runs in the listing handler before it answers: a test's
	// hook between a sign-in's token and its save.
	onList   func(*http.Request)
	requests []string
	// scope, when set, is the scope its token responses grant: a ChatGPT plan
	// sign-in is refused a grant without chatgpt.tokens.use.direct.
	scope string
}

var issuerKey = sync.OnceValues(func() (*ecdsa.PrivateKey, error) { return ecdsa.GenerateKey(elliptic.P256(), rand.Reader) })

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func newIssuer(t *testing.T, origin string) *issuer {
	t.Helper()
	key, err := issuerKey()
	if err != nil {
		t.Fatal(err)
	}
	is := &issuer{origin: origin, key: key}
	mux := http.NewServeMux()
	is.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		is.mu.Lock()
		is.requests = append(is.requests, r.Method+" "+r.URL.Path)
		is.mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(is.Close)
	write := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.MarshalWrite(w, v); err != nil {
			t.Error(err)
		}
	}
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]any{
			"issuer": origin, "authorization_endpoint": origin + "/oauth2/authorize", "token_endpoint": origin + "/oauth2/token",
			"device_authorization_endpoint": origin + "/oauth2/device/code", "jwks_uri": origin + "/jwks",
			"revocation_endpoint": origin + "/oauth2/revoke", "id_token_signing_alg_values_supported": []string{"ES256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]any{"keys": []map[string]string{{"kty": "EC", "crv": "P-256", "kid": "ec-1",
			"x": b64(key.X.FillBytes(make([]byte, 32))), "y": b64(key.Y.FillBytes(make([]byte, 32)))}}})
	})
	mux.HandleFunc("POST /oauth2/device/code", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]any{"device_code": "dc", "user_code": "ABCD-1234",
			"verification_uri": is.URL + "/device", "expires_in": 60, "interval": 1})
	})
	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		is.mu.Lock()
		deny, nonce := is.deny, is.nonce
		is.mu.Unlock()
		if deny {
			write(w, http.StatusBadRequest, map[string]string{"error": "access_denied"})
			return
		}
		claims := map[string]any{"iss": origin, "aud": r.FormValue("client_id"), "sub": "user-1",
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix()}
		if r.FormValue("grant_type") == "authorization_code" {
			claims["nonce"] = nonce
		}
		resp := map[string]any{"access_token": "access-1", "refresh_token": "refresh-1",
			"expires_in": 3600, "id_token": is.sign(t, claims)}
		if is.scope != "" {
			resp["scope"] = is.scope
		}
		write(w, http.StatusOK, resp)
	})
	mux.HandleFunc("GET /v1/", func(w http.ResponseWriter, r *http.Request) {
		is.mu.Lock()
		refuse, status, hook := is.refuse, is.listStatus, is.onList
		is.mu.Unlock()
		if hook != nil {
			hook(r)
		}
		if status != 0 {
			write(w, status, map[string]any{"error": map[string]string{"message": "the listing failed"}})
			return
		}
		if refuse {
			write(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"message": "the credential is refused"}})
			return
		}
		write(w, http.StatusOK, map[string]any{"object": "list", "data": []map[string]string{{"id": "grok-4.6", "object": "model"}}})
	})
	return is
}

// setNonce is the test's browser reaching the authorize URL: the issuer then
// signs the ID token with that nonce. It returns the redirect and the state.
func (is *issuer) setNonce(t *testing.T, authorize string) (redirect, state string) {
	t.Helper()
	u, err := url.Parse(authorize)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	is.mu.Lock()
	is.nonce = q.Get("nonce")
	is.mu.Unlock()
	return q.Get("redirect_uri"), q.Get("state")
}

func (is *issuer) setDeny() {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.deny = true
}

func (is *issuer) setListStatus(code int) {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.listStatus = code
}

func (is *issuer) setOnList(f func(*http.Request)) {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.onList = f
}

func (is *issuer) setRefuse() {
	is.mu.Lock()
	defer is.mu.Unlock()
	is.refuse = true
}

// seen reports whether the issuer was sent a request whose method and path
// are in the log.
func (is *issuer) seen(substr string) bool {
	is.mu.Lock()
	defer is.mu.Unlock()
	return slices.ContainsFunc(is.requests, func(r string) bool { return strings.Contains(r, substr) })
}

// errNoRoute is a request to a host no test issuer answers as.
var errNoRoute = errors.New("no route: the test reaches only its issuers")

// sign is claims as an ES256 JWT under the issuer's key.
func (is *issuer) sign(t *testing.T, claims map[string]any) string {
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
	r, s, err := ecdsa.Sign(rand.Reader, is.key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + b64(append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...))
}

// byHost sends each request to the issuer that answers as its host, and any
// other to nobody: no request leaves the machine.
type byHost map[string]*issuer

func (h byHost) RoundTrip(r *http.Request) (*http.Response, error) {
	is, ok := h[r.URL.Host]
	if !ok {
		return nil, &url.Error{Op: r.Method, URL: r.URL.String(), Err: errNoRoute}
	}
	u, err := url.Parse(is.URL)
	if err != nil {
		return nil, err
	}
	r = r.Clone(r.Context())
	r.URL.Scheme, r.URL.Host, r.Host = u.Scheme, u.Host, u.Host
	return http.DefaultTransport.RoundTrip(r)
}

// toOne sends every request to one issuer, whatever host it names: the ChatGPT
// plan's sign-in reaches OpenAI's issuer and its backend, and no request leaves
// the machine.
type toOne struct{ is *issuer }

func (t toOne) RoundTrip(r *http.Request) (*http.Response, error) {
	u, err := url.Parse(t.is.URL)
	if err != nil {
		return nil, err
	}
	r = r.Clone(r.Context())
	r.URL.Scheme, r.URL.Host, r.Host = u.Scheme, u.Host, u.Host
	return http.DefaultTransport.RoundTrip(r)
}
