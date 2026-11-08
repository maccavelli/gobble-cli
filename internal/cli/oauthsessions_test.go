package cli

import (
	"context"
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

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// issuerOnly passes a request for issuer, the loopback host of a test's
// token endpoint, to the real transport, and answers every other request
// with an error: a listing gives the static models and nothing reaches the
// network. The SDK gives a session with no client the listing's
// (go-llmprovider-sdk catalog/discovery.go:291, UseDefaultHTTPClient), so a
// refresh started by a listing goes through it.
type issuerOnly struct{ issuer string }

func (t issuerOnly) RoundTrip(r *http.Request) (*http.Response, error) {
	if t.issuer != "" && r.URL.Host == t.issuer {
		return http.DefaultTransport.RoundTrip(r)
	}
	return nil, errors.New("no network in tests")
}

// heldIssuer is a token endpoint on loopback whose refresh waits for
// release, then rotates the refresh token to "refresh-2"; hit is closed by
// the first refresh.
func heldIssuer(t *testing.T) (tokenURL string, hit chan struct{}, release func()) {
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

// storeGrok saves a Grok login in e's auth.json, expired when expired.
func storeGrok(t *testing.T, e *runEnv, tokenURL string, expired bool) {
	t.Helper()
	exp := time.Now().Add(time.Hour)
	if expired {
		exp = time.Now().Add(-time.Hour)
	}
	if err := e.logins().Save(t.Context(), "grok", &provider.Login{Access: "access-1", Refresh: "refresh-1", Expires: exp,
		ClientID: "cid", TokenURL: tokenURL}); err != nil {
		t.Fatal(err)
	}
}

// listGrok lists Grok with the run's options and its stored login, as the
// run's listing does, through issuerOnly: the run's session for Grok is
// held, and a token is asked for under ctx.
func listGrok(ctx context.Context, e *runEnv, tokenURL string) {
	host := ""
	if u, err := url.Parse(tokenURL); err == nil {
		host = u.Host
	}
	o := e.providerOptions("grok", provider.Options{Logins: e.logins(), HTTPClient: &http.Client{Transport: issuerOnly{issuer: host}}})
	_, _ = provider.List(ctx, "grok", "", o)
}

// Every provider.Options the run builds carries the run's sessions, as it
// carries its cache (0046-MADR D8).
func TestRunOptionsCarryTheSessions(t *testing.T) {
	e, _ := modelEnv(t)
	want := &e.sessions
	var built []*provider.Sessions
	old := ambientProvider
	ambientProvider = func(_ func(string) (string, bool), _ string, o provider.Options) (llm.Provider, provider.Selection, error) {
		built = append(built, o.Sessions)
		return nil, provider.Selection{}, errors.New("recorded")
	}
	t.Cleanup(func() { ambientProvider = old })
	_, _ = modelProvider(e)("s1", llm.DefaultRetry) // no credential: no provider is chosen
	t.Setenv("OPENAI_API_KEY", "test-only")
	e.refreshCredentials("openai")
	_, _ = modelProvider(e)("s2", llm.DefaultRetry)
	_, _ = providerFor(e)("openai", "s3", llm.DefaultRetry)
	if len(built) != 3 {
		t.Errorf("%d providers built, want 3", len(built))
	}
	for i, s := range built {
		if s != want {
			t.Errorf("provider %d built with sessions %p, want the run's %p", i, s, want)
		}
	}
	if s := e.providerOptions("openai", provider.Options{}).Sessions; s != want {
		t.Errorf("providerOptions' sessions %p, want the run's %p", s, want)
	}
	oldFacts := modelFacts
	var facts *provider.Sessions
	modelFacts = func(o provider.Options, id, model string, _ *catalog.ModelsJSON) llm.Model {
		facts = o.Sessions
		return llm.Model{Provider: id, ID: model}
	}
	t.Cleanup(func() { modelFacts = oldFacts })
	_ = e.agentOptions().ModelInfo("openai", "gpt-5-nano")
	if facts != want {
		t.Errorf("the ModelInfo hook's sessions %p, want the run's %p", facts, want)
	}
}

// A sign-in's refreshCredentials drops the provider's held session, so the
// run's next use loads the new login (0046-MADR D8).
func TestRefreshCredentialsDropsTheSession(t *testing.T) {
	e, _ := modelEnv(t)
	storeGrok(t, e, "https://auth.test/oauth2/token", false)
	listGrok(t.Context(), e, "https://auth.test/oauth2/token")
	if held := e.sessions.Held(); !slices.Equal(held, []string{"grok"}) {
		t.Fatalf("held %q after a Grok listing, want [grok]", held)
	}
	e.refreshCredentials("grok")
	if held := e.sessions.Held(); len(held) != 0 {
		t.Fatalf("held %q after refreshCredentials(grok), want none", held)
	}
}

// startRefresh holds the run's Grok session with an expired login and
// starts its refresh, which blocks at the issuer; the caller that asked
// stops waiting, and the refresh goes on.
func startRefresh(t *testing.T, e *runEnv, tokenURL string, hit chan struct{}) {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	listed := make(chan struct{})
	go func() {
		defer close(listed)
		listGrok(ctx, e, tokenURL)
	}()
	select {
	case <-hit:
	case <-time.After(5 * time.Second):
		t.Fatal("no refresh reached the issuer")
	}
	cancel()
	<-listed
}

// endListings waits for a sign-in refresh in flight, so its rotation is
// saved before the run ends (0046-MADR D8).
func TestEndListingsWaitsForARefresh(t *testing.T) {
	e, _ := modelEnv(t)
	tokenURL, hit, release := heldIssuer(t)
	storeGrok(t, e, tokenURL, true)
	startRefresh(t, e, tokenURL, hit)
	ended := make(chan struct{})
	go func() {
		defer close(ended)
		e.endListings(t.Context())
	}()
	select {
	case <-ended:
		t.Fatal("endListings returned while the refresh was blocked at the issuer")
	case <-time.After(200 * time.Millisecond):
	}
	release()
	select {
	case <-ended:
	case <-time.After(5 * time.Second):
		t.Fatal("endListings did not return once the refresh could end")
	}
	if l, err := e.logins().Load(t.Context(), "grok"); err != nil || l == nil || l.Refresh != "refresh-2" {
		t.Fatalf("stored Grok login %v (%v) after endListings, want the rotated refresh token, refresh-2", l, err)
	}
}

// A refresh that does not end holds endListings for refreshWaitBound at
// most, and the Warn record names the provider and says its rotation may
// not be saved (0046-MADR D8, Q3).
func TestEndListingsRefreshBound(t *testing.T) {
	e, warnings := modelEnv(t)
	old := refreshWaitBound
	refreshWaitBound = 50 * time.Millisecond
	t.Cleanup(func() { refreshWaitBound = old })
	tokenURL, hit, release := heldIssuer(t)
	storeGrok(t, e, tokenURL, true)
	t.Cleanup(func() { // ends the refresh before the home is removed
		release()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := e.sessions.Wait(ctx); err != nil {
			t.Error(err)
		}
	})
	startRefresh(t, e, tokenURL, hit)
	begin := time.Now()
	e.endListings(t.Context())
	took := time.Since(begin)
	if took < refreshWaitBound || took > 2*time.Second {
		t.Errorf("endListings took %s, want the refresh bound, %s", took, refreshWaitBound)
	}
	if w := warnings.String(); !strings.Contains(w, "its rotated login may not be saved") || !strings.Contains(w, "grok") {
		t.Errorf("warnings %q, want the record naming grok", w)
	}
}
