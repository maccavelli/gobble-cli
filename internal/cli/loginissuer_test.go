package cli

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
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// loginGrokIssuer is xAI's issuer as the SDK's Grok sign-in names it
// (auth.DefaultGrokOAuthIssuer): the origin the issuer answers as. Only
// llm/provider may import the SDK's llmprovider packages (0004-MADR rule 4),
// so the test states it; a different value fails every sign-in here at its
// discovery.
const loginGrokIssuer = "https://auth.x.ai"

// loginIssuer is an xAI-shaped OIDC issuer in process, for a sign-in that runs
// the SDK's real wizard from gobble auth login: discovery, an ES256 JWKS,
// device and token endpoints whose ID tokens it signs, Kilo's device
// endpoints, and a model listing. It is the subset of llm/provider's
// testIssuer these tests use, which a package's test cannot import. Nothing
// leaves the machine: every request the SDK makes, whatever its host, is
// sent here.
type loginIssuer struct {
	*httptest.Server
	key *ecdsa.PrivateKey

	// kiloToken is the key Kilo's device sign-in gives.
	kiloToken string

	mu        sync.Mutex
	nonce     string
	deny      bool
	kiloPolls int
	// refuseListing, when set, makes the model listings answer 401 and, when
	// shouldRetry is not empty, with that x-should-retry.
	refuseListing bool
	shouldRetry   string
}

// loginIssuerKey is every loginIssuer's signing key: the SDK caches an
// issuer's keys by its URL, and these issuers answer as one origin.
var loginIssuerKey = sync.OnceValues(func() (*ecdsa.PrivateKey, error) { return ecdsa.GenerateKey(elliptic.P256(), rand.Reader) })

func newLoginIssuer(t *testing.T) *loginIssuer {
	t.Helper()
	key, err := loginIssuerKey()
	if err != nil {
		t.Fatal(err)
	}
	ti := &loginIssuer{key: key, kiloToken: "!kilo$token-0123456789"}
	mux := http.NewServeMux()
	ti.Server = httptest.NewServer(mux)
	t.Cleanup(ti.Close)
	write := func(w http.ResponseWriter, status int, v any) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.MarshalWrite(w, v); err != nil {
			t.Error(err)
		}
	}
	origin := loginGrokIssuer
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]any{
			"issuer": origin, "authorization_endpoint": origin + "/oauth2/authorize", "token_endpoint": origin + "/oauth2/token",
			"device_authorization_endpoint": origin + "/oauth2/device/code", "jwks_uri": origin + "/jwks",
			"revocation_endpoint": origin + "/oauth2/revoke", "id_token_signing_alg_values_supported": []string{"ES256"},
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]any{"keys": []map[string]string{{"kty": "EC", "crv": "P-256", "kid": "ec-1",
			"x": loginB64(key.X.FillBytes(make([]byte, 32))), "y": loginB64(key.Y.FillBytes(make([]byte, 32)))}}})
	})
	mux.HandleFunc("POST /oauth2/device/code", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]any{"device_code": "dc", "user_code": "ABCD-1234",
			"verification_uri": ti.URL + "/device", "expires_in": 60, "interval": 1})
	})
	mux.HandleFunc("POST /oauth2/token", func(w http.ResponseWriter, r *http.Request) {
		ti.mu.Lock()
		deny, nonce := ti.deny, ti.nonce
		ti.mu.Unlock()
		if deny {
			write(w, http.StatusBadRequest, map[string]string{"error": "access_denied"})
			return
		}
		claims := map[string]any{"iss": origin, "aud": r.FormValue("client_id"), "sub": "user-1",
			"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix()}
		if r.FormValue("grant_type") == "authorization_code" {
			claims["nonce"] = nonce
		}
		write(w, http.StatusOK, map[string]any{"access_token": "access-1", "refresh_token": "refresh-1",
			"expires_in": 3600, "id_token": ti.sign(t, claims)})
	})
	mux.HandleFunc("POST /api/device-auth/codes", func(w http.ResponseWriter, _ *http.Request) {
		write(w, http.StatusOK, map[string]any{"code": "KILO-1", "verificationUrl": ti.URL + "/kilo", "expiresIn": 60})
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
			write(w, http.StatusOK, map[string]string{"token": ti.kiloToken})
		}
	})
	// refused answers a listing 401, as a service does that refuses the new
	// credential; it reports whether it did.
	refused := func(w http.ResponseWriter) bool {
		ti.mu.Lock()
		refuse, retry := ti.refuseListing, ti.shouldRetry
		ti.mu.Unlock()
		if !refuse {
			return false
		}
		if retry != "" {
			w.Header().Set("X-Should-Retry", retry)
		}
		write(w, http.StatusUnauthorized, map[string]any{"error": map[string]string{"message": "the credential is refused"}})
		return true
	}
	mux.HandleFunc("GET /api/gateway/models", func(w http.ResponseWriter, _ *http.Request) {
		if !refused(w) {
			w.WriteHeader(http.StatusNotFound)
		}
	})
	mux.HandleFunc("GET /v1/", func(w http.ResponseWriter, _ *http.Request) {
		if refused(w) {
			return
		}
		write(w, http.StatusOK, map[string]any{"object": "list", "data": []map[string]string{{"id": "grok-4.6", "object": "model"}}})
	})
	return ti
}

func loginB64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// denying makes the issuer refuse every sign-in, as a user who says no does.
func (ti *loginIssuer) denying() {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.deny = true
}

// refusing makes the model listings, which a sign-in checks its credential
// with, answer 401, with x-should-retry set to shouldRetry when it is not empty.
func (ti *loginIssuer) refusing(shouldRetry string) {
	ti.mu.Lock()
	defer ti.mu.Unlock()
	ti.refuseListing, ti.shouldRetry = true, shouldRetry
}

// sign is claims as an ES256 JWT under the issuer's key.
func (ti *loginIssuer) sign(t *testing.T, claims map[string]any) string {
	t.Helper()
	h, err := json.Marshal(map[string]string{"alg": "ES256", "typ": "JWT", "kid": "ec-1"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	input := loginB64(h) + "." + loginB64(c)
	digest := sha256.Sum256([]byte(input))
	r, s, err := ecdsa.Sign(rand.Reader, ti.key, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	return input + "." + loginB64(append(r.FillBytes(make([]byte, 32)), s.FillBytes(make([]byte, 32))...))
}

// browse is the test's browser: it reaches the sign-in's callback with the
// authorize URL it was shown, as the SDK's loopback listener expects, and
// gives the issuer the URL's nonce for the ID token. It opens nothing.
func (ti *loginIssuer) browse(t *testing.T, authorize string) {
	t.Helper()
	u, err := url.Parse(authorize)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	ti.mu.Lock()
	ti.nonce = q.Get("nonce")
	ti.mu.Unlock()
	redirect, state := q.Get("redirect_uri"), q.Get("state")
	go func() {
		resp, err := http.Get(redirect + "?code=c1&state=" + url.QueryEscape(state)) //nolint:noctx // the test's browser
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
}

// loginTransport sends every request to the issuer, whatever its host.
type loginTransport struct{ target *url.URL }

func (lt loginTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme, r.URL.Host, r.Host = lt.target.Scheme, lt.target.Host, lt.target.Host
	return http.DefaultTransport.RoundTrip(r)
}

// signInAtIssuer is provider.SignIn, the real wizard, run against the issuer
// in place of the network, each call recorded as "provider method" in calls.
// It plays the browser for a browser sign-in: the URL a sign-in prints as an
// open_url step, after the options' Steps has been given it, or hands to the
// options' OpenURL, after that has, is reached at its callback.
func signInAtIssuer(t *testing.T, ti *loginIssuer, calls *[]string) signInFunc {
	t.Helper()
	target, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		*calls = append(*calls, id+" "+o.Method)
		o.HTTPClient = &http.Client{Transport: loginTransport{target}}
		if open := o.OpenURL; open != nil {
			o.OpenURL = func(u string) error {
				err := open(u)
				ti.browse(t, u)
				return err
			}
		}
		if steps := o.Steps; steps != nil {
			o.Steps = func(s authcontract.Step) error {
				if err := steps(s); err != nil {
					return err
				}
				if s.Kind == authcontract.StepOpenURL {
					ti.browse(t, s.URL)
				}
				return nil
			}
		}
		return provider.SignIn(ctx, id, p, o, store)
	}
}
