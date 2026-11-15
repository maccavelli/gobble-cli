package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"

	"github.com/maccavelli/gobble-cli/llm"
)

// toIssuer sends every request to the in-process issuer, whatever its
// host, so the SDK's own endpoints reach it and nothing leaves the machine.
type toIssuer struct{ target *url.URL }

func (t toIssuer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.URL.Scheme, r.URL.Host, r.Host = t.target.Scheme, t.target.Host, t.target.Host
	return http.DefaultTransport.RoundTrip(r)
}

// signInIssuer is the test issuer answering as xAI's, and the options that
// reach it.
func signInIssuer(t *testing.T) (*testIssuer, SignInOptions) {
	t.Helper()
	ti := newTestIssuer(t)
	ti.origin = auth.DefaultGrokOAuthIssuer
	u, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	return ti, SignInOptions{ClientName: "gobble", ClientVersion: "test", HTTPClient: &http.Client{Transport: toIssuer{u}}}
}

// script is a Prompter whose answers the test gives; a question it has no
// answer for fails the test.
type script struct {
	t       *testing.T
	mu      sync.Mutex
	choose  func(title string, choices []Choice) int
	secrets []string
	input   func(ctx context.Context, prompt string) (string, error)
	menus   [][]Choice
	inputs  int
	notes   []string
}

func (s *script) Choose(title string, choices []Choice, _ int) (int, error) {
	s.mu.Lock()
	s.menus = append(s.menus, choices)
	s.mu.Unlock()
	if s.choose == nil {
		s.t.Errorf("asked %q with no answer", title)
		return 0, errors.New("no answer")
	}
	return s.choose(title, choices), nil
}

func (s *script) Confirm(question string, _ bool) (bool, error) {
	s.t.Errorf("asked %q with no answer", question)
	return false, errors.New("no answer")
}

func (s *script) Input(ctx context.Context, prompt string) (string, error) {
	s.mu.Lock()
	s.inputs++
	s.mu.Unlock()
	if s.input == nil {
		s.t.Errorf("asked %q with no answer", prompt)
		return "", errors.New("no answer")
	}
	return s.input(ctx, prompt)
}

func (s *script) Secret(prompt string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.secrets) == 0 {
		s.t.Errorf("asked %q with no answer", prompt)
		return "", errors.New("no answer")
	}
	k := s.secrets[0]
	s.secrets = s.secrets[1:]
	return k, nil
}

func (s *script) Notify(format string, args ...any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.notes = append(s.notes, fmt.Sprintf(format, args...))
}

// labels are provider id's sign-in methods' labels in the SDK's registry,
// for the methods named.
func labels(t *testing.T, id string, ms ...llmprovider.AuthMethodID) []string {
	t.Helper()
	d, ok := providers.Default().Descriptor(llmprovider.ProviderID(id))
	if !ok {
		t.Fatalf("no descriptor for %s", id)
	}
	var out []string
	for _, m := range d.AuthMethods {
		if slices.Contains(ms, m.ID) {
			out = append(out, m.Label)
		}
	}
	return out
}

// Methods are gobble's ways to sign in to each provider: Grok's key, browser
// and device sign-ins; Kilo's key and device sign-in; a key alone for the
// rest, OpenAI's included until its plan sign-in (0031-PLAN P4a); none for
// a provider that takes no key.
func TestSignInMethods(t *testing.T) {
	for id, want := range map[string][]string{
		"grok":   {MethodAPIKey, MethodBrowser, MethodDevice},
		"kilo":   {MethodAPIKey, MethodDevice},
		"openai": {MethodAPIKey},
		"claude": {MethodAPIKey},
		"ollama": nil,
	} {
		if got := Methods(id); !slices.Equal(got, want) {
			t.Errorf("Methods(%s) = %q, want %q", id, got, want)
		}
	}
}

// The wizard's menu offers gobble's methods alone: Grok's without its token
// paste or CLI import. A provider with one method, or a forced method, asks
// no menu; a typed key is checked by the listing and returned.
func TestSignInMenu(t *testing.T) {
	_, o := signInIssuer(t)
	store := &memLogins{}
	s := &script{t: t, secrets: []string{"xai-good-key-0123456789"}, choose: func(_ string, cs []Choice) int { return 0 }}
	c, err := SignIn(t.Context(), "grok", s, o, store)
	if err != nil || c.Key != "xai-good-key-0123456789" || c.Device {
		t.Fatalf("grok's key: %+v, %v", c, err)
	}
	want := labels(t, "grok", llmprovider.AuthAPIKey, llmprovider.AuthBrowserOAuth, llmprovider.AuthDeviceCode)
	if len(s.menus) != 1 || len(want) != 3 {
		t.Fatalf("menus %v, want one of %q", s.menus, want)
	}
	var got []string
	for _, c := range s.menus[0] {
		got = append(got, c.Label)
	}
	if !slices.Equal(got, want) {
		t.Errorf("grok's menu %q, want %q", got, want)
	}

	for _, tc := range []struct{ id, method string }{{"openai", ""}, {"grok", MethodAPIKey}} {
		s := &script{t: t, secrets: []string{"sk-good-key-0123456789"}}
		o := o
		o.Method = tc.method
		if c, err := SignIn(t.Context(), tc.id, s, o, store); err != nil || c.Key != "sk-good-key-0123456789" || len(s.menus) != 0 {
			t.Errorf("%s, method %q: %+v, %v, menus %v", tc.id, tc.method, c, err, s.menus)
		}
	}
	if len(store.logins) != 0 {
		t.Errorf("a key saved a login: %v", store.logins)
	}
}

// browser is the test's browser: it reaches the sign-in's callback.
func browser(t *testing.T, ti *testIssuer) func(string) error {
	return func(u string) error {
		redirect, state := ti.authorizeParams(t, u)
		go func() {
			resp, err := http.Get(redirect + "?code=c1&state=" + url.QueryEscape(state)) //nolint:noctx // the test's browser
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
}

// Grok's browser sign-in saves the session once the browser reaches the
// callback. The paste prompt, read through a Prompter that gives up a read,
// is given up then, with no Enter asked; with NoPaste the Prompter is never
// read.
func TestSignInGrokBrowser(t *testing.T) {
	ti, o := signInIssuer(t)
	o.Method, o.OpenURL = MethodBrowser, browser(t, ti)
	store := &memLogins{}
	// The user presses Enter after 5 s, which a sign-in that cannot give up
	// its read waits for.
	s := &script{t: t, input: func(ctx context.Context, _ string) (string, error) {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(5 * time.Second):
			return "", nil
		}
	}}
	start := time.Now()
	c, err := SignIn(t.Context(), "grok", s, o, store)
	if err != nil || c.Key != "" || store.logins["grok"] == nil || store.logins["grok"].Access != "access-1" {
		t.Fatalf("grok's browser sign-in: %+v, %v, saved %v", c, err, store.logins)
	}
	if s.inputs != 1 || time.Since(start) > 4*time.Second {
		t.Errorf("the paste prompt was read %d times in %s, want once, given up", s.inputs, time.Since(start))
	}

	o.NoPaste = true
	quiet := &script{t: t}
	if _, err := SignIn(t.Context(), "grok", quiet, o, &memLogins{}); err != nil || quiet.inputs != 0 {
		t.Errorf("NoPaste: %v, read %d times", err, quiet.inputs)
	}
	if !slices.ContainsFunc(quiet.notes, func(n string) bool { return strings.Contains(n, "/oauth2/authorize?") }) {
		t.Errorf("the URL was not shown: %q", quiet.notes)
	}
}

// Grok's device sign-in gives ShowDeviceCode the URL, the code and an
// expiry, and saves the session; Kilo's gives its token as a key from the
// device, and saves nothing (0031-MADR D7).
func TestSignInDevice(t *testing.T) {
	ti, o := signInIssuer(t)
	o.Method = MethodDevice
	var shown []DeviceCode
	o.ShowDeviceCode = func(d DeviceCode) error { shown = append(shown, d); return nil }
	store := &memLogins{}
	if c, err := SignIn(t.Context(), "grok", &script{t: t}, o, store); err != nil || c.Key != "" || store.logins["grok"] == nil {
		t.Fatalf("grok's device sign-in: %+v, %v (%v), saved %v", c, err, errors.Unwrap(err), store.logins)
	}
	c, err := SignIn(t.Context(), "kilo", &script{t: t}, o, store)
	if err != nil || c.Key != "kilo-token-0123456789" || !c.Device || store.logins["kilo"] != nil {
		t.Fatalf("kilo's device sign-in: %+v, %v, saved %v", c, err, store.logins)
	}
	if len(shown) != 2 || shown[0].Code != "ABCD-1234" || shown[0].URL != ti.URL+"/device" || shown[0].Expires.IsZero() ||
		shown[1].Code != "KILO-1" || shown[1].Expires.IsZero() {
		t.Errorf("shown %+v", shown)
	}
	stop := errors.New("stop")
	o.ShowDeviceCode = func(DeviceCode) error { return stop }
	if _, err := SignIn(t.Context(), "grok", &script{t: t}, o, &memLogins{}); !errors.Is(err, stop) {
		t.Errorf("ShowDeviceCode's error: %v", err)
	}
}

// A typed key the listing refuses is asked for again, up to three; the
// notice and the error say so in gobble's words, with nothing of the
// service's text, and the error is llm.ErrAuth's class.
func TestSignInRefusedKeys(t *testing.T) {
	_, o := signInIssuer(t)
	o.Method = MethodAPIKey
	s := &script{t: t, secrets: []string{"refused-1", "refused-2", "refused-3"}}
	_, err := SignIn(t.Context(), "grok", s, o, &memLogins{})
	if err == nil || err.Error() != "grok refused 3 keys; none was saved" || !errors.Is(err, llm.ErrAuth) {
		t.Fatalf("three refusals: %v", err)
	}
	s = &script{t: t, secrets: []string{"refused-1", "xai-good-key-0123456789"}}
	c, err := SignIn(t.Context(), "grok", s, o, &memLogins{})
	if err != nil || c.Key != "xai-good-key-0123456789" {
		t.Fatalf("a refusal, then a good key: %+v, %v", c, err)
	}
	if !slices.ContainsFunc(s.notes, func(n string) bool { return strings.Contains(n, "refused") }) {
		t.Errorf("no notice of the refusal: %q", s.notes)
	}
	for _, n := range s.notes {
		if strings.Contains(n, serviceText) {
			t.Errorf("a notice shows the service's text: %q", n)
		}
	}
}

// seenRequest is one request a uaRecorder saw: what the tests read of it,
// and what a named run and an unnamed one must share.
type seenRequest struct {
	method, path, userAgent string
	form                    string // the form body and the query, sorted
}

// uaRecorder is an http.RoundTripper that records each request's method,
// path, User-Agent and form, then sends it on to next.
type uaRecorder struct {
	next http.RoundTripper
	mu   sync.Mutex
	seen []seenRequest
}

func (u *uaRecorder) RoundTrip(r *http.Request) (*http.Response, error) {
	form := r.URL.Query()
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		r.Body = io.NopCloser(bytes.NewReader(b))
		if v, err := url.ParseQuery(string(b)); err == nil {
			for k, vs := range v {
				form[k] = append(form[k], vs...)
			}
		}
	}
	u.mu.Lock()
	u.seen = append(u.seen, seenRequest{method: r.Method, path: r.URL.Path, userAgent: r.Header.Get("User-Agent"), form: form.Encode()})
	u.mu.Unlock()
	next := u.next
	if next == nil {
		next = http.DefaultTransport
	}
	return next.RoundTrip(r)
}

// atIssuer makes rec see each request that reaches ti, whatever client sent
// it: a session's refresh goes through the SDK's own client when no listing
// gave it one, so a transport in an option does not see it. Call it before
// the first request.
func (u *uaRecorder) atIssuer(ti *testIssuer) {
	next := ti.Config.Handler
	ti.Config.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm() //nolint:errcheck // a malformed form is recorded as it parsed
		u.mu.Lock()
		u.seen = append(u.seen, seenRequest{method: r.Method, path: r.URL.Path, userAgent: r.Header.Get("User-Agent"), form: r.Form.Encode()})
		u.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}

// requests is a copy of what the recorder has seen.
func (u *uaRecorder) requests() []seenRequest {
	u.mu.Lock()
	defer u.mu.Unlock()
	return slices.Clone(u.seen)
}

// wantNamed fails t for a request among reqs whose User-Agent does not begin
// "gobble/1.2.3-test (" and name the SDK's token, and for no request at all.
func wantNamed(t *testing.T, what string, reqs []seenRequest) {
	t.Helper()
	if len(reqs) == 0 {
		t.Errorf("%s: no request seen", what)
	}
	for _, r := range reqs {
		if !strings.HasPrefix(r.userAgent, "gobble/1.2.3-test (") || !strings.Contains(r.userAgent, "go-llmprovider-sdk/") {
			t.Errorf("%s: %s %s sent User-Agent %q, want gobble/1.2.3-test ( ... go-llmprovider-sdk/", what, r.method, r.path, r.userAgent)
		}
	}
}

// wantUnnamed fails t for a request among reqs whose User-Agent is not the
// SDK's own, and for no request at all.
func wantUnnamed(t *testing.T, what string, reqs []seenRequest) {
	t.Helper()
	if len(reqs) == 0 {
		t.Errorf("%s: no request seen", what)
	}
	for _, r := range reqs {
		if !strings.HasPrefix(r.userAgent, "go-llmprovider-sdk/") {
			t.Errorf("%s: %s %s sent User-Agent %q, want the SDK's own", what, r.method, r.path, r.userAgent)
		}
	}
}

// A sign-in through SignIn names gobble: Grok's device request carries
// referrer=gobble, its browser sign-in's authorize URL does, and every
// request of the device sign-in leads its User-Agent with gobble and the
// build's version, the SDK's token last. No originator is asserted: gobble
// sends none (0051-MADR D9, N11).
func TestSignInNamesGobble(t *testing.T) {
	_, o := signInIssuer(t)
	o.ClientVersion = "1.2.3-test"
	rec := &uaRecorder{next: o.HTTPClient.Transport}
	o.HTTPClient = &http.Client{Transport: rec}

	o.Method = MethodDevice
	o.ShowDeviceCode = func(DeviceCode) error { return nil }
	if _, err := SignIn(t.Context(), "grok", &script{t: t}, o, &memLogins{}); err != nil {
		t.Fatalf("grok's device sign-in: %v", err)
	}
	reqs := rec.requests()
	wantNamed(t, "the device sign-in", reqs)
	var device bool
	for _, r := range reqs {
		if r.path == "/oauth2/device/code" {
			device = true
			if v, _ := url.ParseQuery(r.form); v.Get("referrer") != "gobble" {
				t.Errorf("the device request's form %q, want referrer=gobble", r.form)
			}
		}
	}
	if !device {
		t.Errorf("no device request among %+v", reqs)
	}

	// The browser sign-in, cancelled once its URL is shown.
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var authorize string
	o.Method, o.NoPaste = MethodBrowser, true
	o.OpenURL = func(u string) error {
		authorize = u
		cancel()
		return nil
	}
	_, _ = SignIn(ctx, "grok", &script{t: t}, o, &memLogins{}) //nolint:errcheck // cancelled on purpose: only the URL is read
	u, err := url.Parse(authorize)
	if err != nil || authorize == "" {
		t.Fatalf("the browser sign-in gave no authorize URL: %q, %v", authorize, err)
	}
	if got := u.Query().Get("referrer"); got != "gobble" {
		t.Errorf("the authorize URL %q has referrer %q, want gobble", authorize, got)
	}
}
