package provider

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
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
	// fail, when set, is the error a menu titled title gives instead of an
	// answer, nil to answer it (a person's Ctrl+C at it).
	fail func(title string) error
	// confirm, when set, answers a yes-or-no question.
	confirm func(question string) (bool, error)
	menus   [][]Choice
	titles  []string
	inputs  int
	notes   []string
}

func (s *script) Choose(title string, choices []Choice, _ int) (int, error) {
	s.mu.Lock()
	s.menus = append(s.menus, choices)
	s.titles = append(s.titles, title)
	s.mu.Unlock()
	if s.fail != nil {
		if err := s.fail(title); err != nil {
			return 0, err
		}
	}
	if s.choose == nil {
		s.t.Errorf("asked %q with no answer", title)
		return 0, errors.New("no answer")
	}
	return s.choose(title, choices), nil
}

func (s *script) Confirm(question string, _ bool) (bool, error) {
	if s.confirm != nil {
		return s.confirm(question)
	}
	s.t.Errorf("asked %q with no answer", question)
	return false, errors.New("no answer")
}

// asked are the titles of the menus s was asked, in order.
func (s *script) asked() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.titles)
}

// noted are the notices s was given, in order.
func (s *script) noted() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.notes)
}

// browserQuestion is the question go-llmprovider-sdk's wizard asks, from
// v1.32.0, before a browser sign-in at a terminal with a paste offered, and
// its two answers (0083-MADR D3; 0068-MADR W1).
const (
	browserQuestion  = "Where is the browser you will sign in with?"
	browserHere      = "On this machine"
	browserElsewhere = "On another device: I will paste the address back"
)

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
// and device sign-ins; Kilo's key and device sign-in; OpenAI's key and ChatGPT
// plan sign-in; a key alone for the rest; none for a provider that takes no
// key (0051-MADR D2).
func TestSignInMethods(t *testing.T) {
	for id, want := range map[string][]string{
		"grok":   {MethodAPIKey, MethodBrowser, MethodDevice},
		"kilo":   {MethodAPIKey, MethodDevice},
		"openai": {MethodAPIKey, MethodChatGPTPlan},
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

	for _, tc := range []struct{ id, method string }{{"openai", MethodAPIKey}, {"grok", MethodAPIKey}} {
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
	// The browser is on this machine: the first answer to the SDK's question
	// (0068-MADR W6).
	s := &script{t: t, choose: func(string, []Choice) int { return 0 }, input: func(ctx context.Context, _ string) (string, error) {
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
	if got := s.asked(); !slices.Equal(got, []string{browserQuestion}) {
		t.Errorf("the sign-in asked %q, want the SDK's question where the browser is, once", got)
	}
	if s.inputs != 1 || time.Since(start) > 4*time.Second {
		t.Errorf("the paste prompt was read %d times in %s, want once, given up", s.inputs, time.Since(start))
	}
	// The session the SDK saved is one gobble accepts, by the kind the SDK
	// stored (0051-MADR D2).
	if !Accepts("grok", store.logins["grok"]) {
		t.Errorf("Accepts(grok) refused the session the SDK stored: %+v", store.logins["grok"])
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
	if !Accepts("grok", store.logins["grok"]) {
		t.Errorf("Accepts(grok) refused the session the SDK stored: %+v", store.logins["grok"])
	}
	if len(shown) != 2 || shown[0].Code != "ABCD-1234" || shown[0].URL != ti.URL+"/device" || shown[0].Expires.IsZero() ||
		shown[1].Code != "KILO-1" || shown[1].Expires.IsZero() {
		t.Errorf("shown %+v", shown)
	}
	// Kilo's profile refusing the device key refuses the sign-in: the SDK
	// reads it once to check the key (0068-MADR L4).
	ti.profile(http.StatusUnauthorized)
	if c, err := SignIn(t.Context(), "kilo", &script{t: t}, o, store); err == nil || c.Key != "" || !errors.Is(err, llm.ErrAuth) {
		t.Errorf("kilo's device key the profile refuses: %+v, %v; want no key and an auth failure", c, err)
	}
	ti.profile(0)
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

// shownURL is the authorize URL among the notices s was given, or "".
func shownURL(s *script) string {
	for _, n := range s.noted() {
		for f := range strings.FieldsSeq(n) {
			if strings.Contains(f, "/oauth2/authorize?") {
				return strings.Trim(f, ".,;()<>\"'")
			}
		}
	}
	return ""
}

// dial reports whether something listens at host, a redirect address's.
func dial(ctx context.Context, host string) error {
	c, err := (&net.Dialer{Timeout: time.Second}).DialContext(ctx, "tcp", host)
	if err == nil {
		_ = c.Close()
	}
	return err
}

// browserRecord is what a test's browser saw: how often it was opened,
// whether the redirect address answered a dial while the sign-in waited, and
// whether a listener of this process, not there when before was taken, was
// on its port then (listensSince).
type browserRecord struct {
	mu      sync.Mutex
	opened  int
	dialErr error
	before  map[int]bool
	held    bool
	heldErr error
	// getErr is why the browser's GET of the callback failed, nil while it
	// has not (0068-MADR W17).
	getErr error
}

// browserFailed records why the browser's GET of the callback failed.
func (b *browserRecord) browserFailed(err error) {
	b.mu.Lock()
	b.getErr = err
	b.mu.Unlock()
}

// browserErr is why the browser's GET of the callback failed, or nil.
func (b *browserRecord) browserErr() error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.getErr
}

// callbackBrowser is a test's browser for "On this machine": it records, in
// rec, whether the redirect address answers a dial and whether a listener of
// this process that was not there before is on its port, then GETs the
// callback with client. A GET that fails, or that the callback answers with
// a status other than 200, is recorded and ends the sign-in through end, so
// the test fails at once naming it rather than after the SDK's 600 s wait
// (0068-MADR W17).
func callbackBrowser(t *testing.T, ti *testIssuer, rec *browserRecord, client *http.Client, end context.CancelFunc) func(string) error {
	t.Helper()
	return func(u string) error {
		redirect, state := ti.authorizeParams(t, u)
		ru, err := url.Parse(redirect)
		if err != nil {
			return err
		}
		rec.listenedAt(ru)
		derr := dial(t.Context(), ru.Host)
		rec.mu.Lock()
		rec.opened++
		rec.dialErr = derr
		rec.mu.Unlock()
		go func() {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet,
				redirect+"?code=c1&state="+url.QueryEscape(state), nil)
			if err == nil {
				var resp *http.Response
				if resp, err = client.Do(req); err == nil {
					_ = resp.Body.Close()
					if resp.StatusCode != http.StatusOK {
						err = fmt.Errorf("the callback answered %s", resp.Status)
					}
				}
			}
			if err != nil {
				rec.browserFailed(err)
				end()
			}
		}()
		return nil
	}
}

// newBrowserRecord takes this process's listening ports before the sign-in
// starts.
func newBrowserRecord(t *testing.T) *browserRecord {
	t.Helper()
	before, err := ownListeningPorts()
	if err != nil {
		t.Fatalf("this process's listening ports: %v", err)
	}
	return &browserRecord{before: before}
}

func (b *browserRecord) get() (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.opened, b.dialErr
}

// listened is what listensHere answered for the redirect port.
func (b *browserRecord) listened() (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.held, b.heldErr
}

// listenedAt records whether a listener of this process that was not there
// before the sign-in is on redirect address ru's port.
func (b *browserRecord) listenedAt(ru *url.URL) {
	port, err := strconv.Atoi(ru.Port())
	held := false
	if err == nil {
		held, err = listensSince(b.before, port)
	}
	b.mu.Lock()
	b.held, b.heldErr = held, err
	b.mu.Unlock()
}

// waitForCtx is an Input that answers nothing until its context ends, as a
// person who presses nothing.
func waitForCtx(ctx context.Context, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

// A browser sign-in at a terminal with a paste offered asks where the browser
// is, with "On this machine" first; that answer binds the listener (the
// redirect address answers a dial while the sign-in waits, and listensHere
// finds this process listening on its port) and opens the browser, which
// completes it (0068-MADR W1, W2; Q1; W2 as amended 2026-11-27). A failed GET
// of the browser ends the sign-in and fails the test at once, naming it
// (W17).
func TestSignInAsksWhereTheBrowserIs(t *testing.T) {
	ti, o := signInIssuer(t)
	o.Method = MethodBrowser
	rec := newBrowserRecord(t)
	ctx, end := context.WithCancel(t.Context())
	defer end()
	o.OpenURL = callbackBrowser(t, ti, rec, &http.Client{}, end)
	store := &memLogins{}
	s := &script{t: t, choose: func(string, []Choice) int { return 0 }, input: waitForCtx}
	c, err := SignIn(ctx, "grok", s, o, store)
	if berr := rec.browserErr(); berr != nil {
		t.Fatalf("the test browser's GET of the callback failed, which ended the sign-in (%v): %v", err, berr)
	}
	if got := s.asked(); !slices.Equal(got, []string{browserQuestion}) {
		t.Errorf("the sign-in asked %q, want the question where the browser is", got)
	} else if m := s.menus[0]; len(m) != 2 || m[0].Label != browserHere || m[1].Label != browserElsewhere {
		t.Errorf("the question's choices %+v, want %q then %q", m, browserHere, browserElsewhere)
	}
	if opened, derr := rec.get(); opened != 1 || derr != nil {
		t.Errorf("the browser was opened %d times and the redirect address answered %v; want once, and a listener", opened, derr)
	}
	if held, herr := rec.listened(); !held || herr != nil {
		t.Errorf("this process listened on the redirect port: %v (%v); want a listener of the sign-in's", held, herr)
	}
	if err != nil || c.Key != "" || store.logins["grok"] == nil {
		t.Fatalf("the sign-in: %+v, %v, saved %v", c, err, store.logins)
	}
}

// When the test browser's GET of the callback fails, as it did under the
// Windows laptop's loopback exhaustion ("connectex: Only one usage of each
// socket address"), the browser records the error and ends the sign-in: it
// returns context.Canceled within seconds and saves nothing, never after the
// SDK's 600 s. The 30 s bound only turns a sign-in left waiting into this
// test's failure (0068-MADR W17; the owner's "Make the hang a fast failure
// (Recommended)").
func TestSignInBrowserFailureEndsTheSignIn(t *testing.T) {
	ti, o := signInIssuer(t)
	o.Method = MethodBrowser
	rec := newBrowserRecord(t)
	bounded, unbound := context.WithTimeout(t.Context(), 30*time.Second)
	defer unbound()
	ctx, end := context.WithCancel(bounded)
	defer end()
	refused := errors.New("the test transport refuses every request")
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, refused })}
	o.OpenURL = callbackBrowser(t, ti, rec, client, end)
	store := &memLogins{}
	s := &script{t: t, choose: func(string, []Choice) int { return 0 }, input: waitForCtx}
	_, err := SignIn(ctx, "grok", s, o, store)
	if berr := rec.browserErr(); !errors.Is(berr, refused) {
		t.Errorf("the browser recorded %v; want its GET's error, %v", berr, refused)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("the sign-in ended with %v; want context.Canceled, the browser ending it, not the 30 s bound", err)
	}
	if store.logins["grok"] != nil {
		t.Errorf("saved %v", store.logins)
	}
}

// "On another device" binds no listener (while the sign-in waits, no socket
// of this process that was not there before it started listens on the
// redirect address's port), does not open the browser, shows the URL, and a
// pasted callback completes the sign-in. Grok's sign-in is used, so no test
// holds 1455 or 1457 (0068-MADR W3; Q1). The port is drawn from 49152 to
// 65535, the dynamic range where every :0 bind on the host lands, so a dial
// there can reach another program's listener, and this process's own issuer
// can sit on it: the question is this process's sockets, against those it
// held before the sign-in (listensSince; W3 as amended 2026-11-27).
func TestSignInAnotherDeviceNeedsNoListener(t *testing.T) {
	ti, o := signInIssuer(t)
	o.Method = MethodBrowser
	rec := newBrowserRecord(t)
	o.OpenURL = func(string) error {
		rec.mu.Lock()
		rec.opened++
		rec.mu.Unlock()
		return nil
	}
	store := &memLogins{}
	s := &script{t: t, choose: func(string, []Choice) int { return 1 }}
	s.input = func(context.Context, string) (string, error) {
		u := shownURL(s)
		if u == "" {
			return "", errors.New("no URL was shown before the paste")
		}
		redirect, state := ti.authorizeParams(t, u)
		ru, err := url.Parse(redirect)
		if err != nil {
			return "", err
		}
		rec.listenedAt(ru)
		return redirect + "?code=c1&state=" + url.QueryEscape(state), nil
	}
	c, err := SignIn(t.Context(), "grok", s, o, store)
	if got := s.asked(); !slices.Equal(got, []string{browserQuestion}) {
		t.Errorf("the sign-in asked %q, want the question where the browser is", got)
	}
	opened, _ := rec.get()
	if held, herr := rec.listened(); opened != 0 || held || herr != nil {
		t.Errorf("the browser was opened %d times and this process listened on the redirect port: %v (%v); want neither: no listener", opened, held, herr)
	}
	if err != nil || store.logins["grok"] == nil {
		t.Fatalf("the sign-in by paste: %+v, %v, saved %v", c, err, store.logins)
	}
}

// Where no paste is offered, with no terminal (NoPaste) or a step driver
// (--json, an ACP driver), the SDK asks no such question (0068-MADR W4; a
// guard, plant P13).
func TestSignInNoPasteAsksNoBrowserQuestion(t *testing.T) {
	ti, o := signInIssuer(t)
	o.Method = MethodBrowser
	quiet := o
	quiet.OpenURL, quiet.NoPaste = browser(t, ti), true
	s := &script{t: t}
	if _, err := SignIn(t.Context(), "grok", s, quiet, &memLogins{}); err != nil || len(s.asked()) != 0 {
		t.Errorf("NoPaste: %v, asked %q; want a sign-in and no question", err, s.asked())
	}
	driven := o
	open := browser(t, ti)
	driven.Steps = func(st authcontract.Step) error {
		if st.Kind == authcontract.StepOpenURL {
			return open(st.URL)
		}
		return nil
	}
	s = &script{t: t}
	if _, err := SignIn(t.Context(), "grok", s, driven, &memLogins{}); err != nil || len(s.asked()) != 0 {
		t.Errorf("a step driver: %v, asked %q; want a sign-in and no question", err, s.asked())
	}
}

// A device sign-in asks no question where the browser is: Grok's and Kilo's,
// with the method named, ask nothing (0068-MADR W5; the SDK's rule, a guard
// with no plant).
func TestSignInDeviceAsksNoBrowserQuestion(t *testing.T) {
	_, o := signInIssuer(t)
	o.Method = MethodDevice
	o.ShowDeviceCode = func(DeviceCode) error { return nil }
	for _, id := range []string{"grok", "kilo"} {
		s := &script{t: t}
		if _, err := SignIn(t.Context(), id, s, o, &memLogins{}); err != nil || len(s.asked()) != 0 {
			t.Errorf("%s's device sign-in: %v, asked %q; want no question", id, err, s.asked())
		}
	}
}

// "On another device", then three empty pastes: the sign-in ends with an error
// of the kind invalid_request, in gobble's words, and nothing is saved
// (0068-MADR W13).
func TestSignInAnotherDeviceEmptyPastes(t *testing.T) {
	_, o := signInIssuer(t)
	o.Method = MethodBrowser
	o.OpenURL = func(string) error { return nil }
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	store := &memLogins{}
	empties := 0
	s := &script{t: t, choose: func(string, []Choice) int { return 1 }}
	s.input = func(ctx context.Context, _ string) (string, error) {
		if empties < 3 {
			empties++
			return "", nil
		}
		<-ctx.Done()
		return "", ctx.Err()
	}
	_, err := SignIn(ctx, "grok", s, o, store)
	if err == nil || !errors.Is(err, llmprovider.ErrInvalidRequest) || ErrorKind(err) != "invalid_request" ||
		err.Error() != "grok sign-in failed (invalid_request)" {
		t.Errorf("three empty pastes: %v (kind %q); want grok sign-in failed (invalid_request)", err, ErrorKind(err))
	}
	if len(store.logins) != 0 {
		t.Errorf("saved %v", store.logins)
	}
}

// Ctrl+C at the question where the browser is: the prompter's own error comes
// back as it is, nothing is saved and no browser is opened (0068-MADR W14).
func TestSignInCancelAtBrowserQuestion(t *testing.T) {
	_, o := signInIssuer(t)
	o.Method = MethodBrowser
	rec := &browserRecord{}
	o.OpenURL = func(string) error {
		rec.mu.Lock()
		rec.opened++
		rec.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	interrupted := errors.New("the person pressed Ctrl+C")
	s := &script{t: t, choose: func(string, []Choice) int { return 0 }, input: waitForCtx, fail: func(title string) error {
		if title == browserQuestion {
			return interrupted
		}
		return nil
	}}
	store := &memLogins{}
	_, err := SignIn(ctx, "grok", s, o, store)
	if err != interrupted { //nolint:errorlint // the caller's own error, returned as it is
		t.Errorf("Ctrl+C at the question: %v, want the prompter's own error as it is", err)
	}
	if opened, _ := rec.get(); opened != 0 || len(store.logins) != 0 {
		t.Errorf("opened %d times, saved %v; want neither", opened, store.logins)
	}
}

// A sign-in whose context ends during the model listing saves nothing and
// returns the context's error (0068-MADR V1; go-llmprovider-sdk 0081-MADR).
func TestSignInCancelledSavesNothing(t *testing.T) {
	ti, o := signInIssuer(t)
	ownCache(t, &o)
	u, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	base := toIssuer{u}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	o.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/") {
			cancel() // the person's cancel arrives while the listing runs
			<-r.Context().Done()
			return nil, r.Context().Err()
		}
		return base.RoundTrip(r)
	})}
	o.Method = MethodDevice
	o.ShowDeviceCode = func(DeviceCode) error { return nil }
	store := &memLogins{}
	_, err = SignIn(ctx, "grok", &script{t: t}, o, store)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("a sign-in cancelled during its listing: %v, want the context's error", err)
	}
	if len(store.logins) != 0 {
		t.Errorf("a cancelled sign-in saved %v", store.logins)
	}
}

// errHeldTooLong is a test locker's own give-up, past its window.
var errHeldTooLong = errors.New("the refresh lock is still held")

// heldLocker is memLogins whose refresh lock another process holds until
// release: a try with a done context fails at once wrapping its error, as
// auth.RefreshLocker documents for a held lock (and gobble's Logins does), and a
// wait gives up after window with errHeldTooLong. With releaseAfter set, the
// hold ends that long after the first wait begins.
type heldLocker struct {
	*memLogins
	window       time.Duration
	releaseAfter time.Duration
	release      chan struct{}

	mu    sync.Mutex
	held  bool
	tries int
	once  sync.Once
}

func (h *heldLocker) LockRefresh(ctx context.Context, _ string) (func(), error) {
	h.mu.Lock()
	h.tries++
	held := h.held
	h.mu.Unlock()
	if !held {
		return func() {}, nil
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("held by another process: %w", err)
	}
	if h.releaseAfter > 0 {
		time.AfterFunc(h.releaseAfter, h.let)
	}
	timer := time.NewTimer(h.window)
	defer timer.Stop()
	select {
	case <-h.release:
		return func() {}, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("held by another process: %w", ctx.Err())
	case <-timer.C:
		return nil, errHeldTooLong
	}
}

// let ends the other process's hold.
func (h *heldLocker) let() {
	h.once.Do(func() {
		h.mu.Lock()
		h.held = false
		h.mu.Unlock()
		close(h.release)
	})
}

// A new sign-in is saved under the store's refresh lock: while another
// process's refresh holds it, the wizard says it waits, and saves once it is
// released (0068-MADR V3; go-llmprovider-sdk 0082-MADR).
func TestSignInWaitsForAHeldRefreshLock(t *testing.T) {
	_, o := signInIssuer(t)
	o.Method = MethodDevice
	o.ShowDeviceCode = func(DeviceCode) error { return nil }
	store := &heldLocker{memLogins: &memLogins{}, window: 10 * time.Second, releaseAfter: 300 * time.Millisecond,
		release: make(chan struct{}), held: true}
	s := &script{t: t}
	if _, err := SignIn(t.Context(), "grok", s, o, store); err != nil {
		t.Fatalf("the sign-in: %v", err)
	}
	if !slices.ContainsFunc(s.noted(), func(n string) bool { return strings.Contains(n, "waiting for another process to finish refreshing") }) {
		t.Errorf("no notice that the save waits: %q", s.noted())
	}
	store.mu.Lock()
	tries := store.tries
	store.mu.Unlock()
	if l, _ := store.Load(t.Context(), "grok"); l == nil || tries < 2 {
		t.Errorf("saved %v after %d tries of the lock; want the session saved once the lock was taken", l, tries)
	}
}

// With the lock held past the store's own bound and no answer to the
// question whether to try again, the sign-in ends with the store's error and
// saves nothing (0068-MADR V4).
func TestSignInHeldLockUnasked(t *testing.T) {
	_, o := signInIssuer(t)
	o.Method = MethodDevice
	o.ShowDeviceCode = func(DeviceCode) error { return nil }
	store := &heldLocker{memLogins: &memLogins{}, window: 200 * time.Millisecond, release: make(chan struct{}), held: true}
	s := &script{t: t, confirm: func(string) (bool, error) { return false, errors.New("no terminal to ask on") }}
	_, err := SignIn(t.Context(), "grok", s, o, store)
	if !errors.Is(err, errHeldTooLong) {
		t.Errorf("the sign-in: %v, want the store's error", err)
	}
	if l, _ := store.Load(t.Context(), "grok"); l != nil {
		t.Errorf("saved %v", l)
	}
}

// SignInError.Keys counts the keys that were typed, not the asks: a blank
// one asked again is not a key (0068-MADR K3).
func TestSignInKeysCountsNonBlank(t *testing.T) {
	_, o := signInIssuer(t)
	o.Method = MethodAPIKey
	s := &script{t: t, secrets: []string{"refused-1", "   ", "refused-2", "refused-3"}}
	_, err := SignIn(t.Context(), "grok", s, o, &memLogins{})
	if err == nil || err.Error() != "grok refused 3 keys; none was saved" || !errors.Is(err, llm.ErrAuth) {
		t.Errorf("three refused keys and a blank: %v, want grok refused 3 keys; none was saved", err)
	}
}

// A refused key, then three blanks: the error matches no sentinel, so it is
// "other", and no key is counted (0068-MADR K4; go-llmprovider-sdk 0085-MADR).
func TestSignInRefusedThenBlanks(t *testing.T) {
	_, o := signInIssuer(t)
	o.Method = MethodAPIKey
	s := &script{t: t, secrets: []string{"refused-1", "", " ", "\t"}}
	_, err := SignIn(t.Context(), "grok", s, o, &memLogins{})
	se, ok := errors.AsType[*SignInError](err)
	if err == nil || err.Error() != "grok sign-in failed (other)" || errors.Is(err, llm.ErrAuth) || !ok || se.Keys != 0 {
		t.Errorf("a refused key, then three blanks: %v; want grok sign-in failed (other), no auth failure, no keys counted", err)
	}
}

// A typed Kilo key that Kilo's profile refuses is asked for again, up to
// three, and none is saved (0068-MADR L1; go-llmprovider-sdk 0080-MADR).
func TestSignInKiloWrongKey(t *testing.T) {
	ti, o := signInIssuer(t)
	ti.profile(http.StatusUnauthorized)
	o.Method = MethodAPIKey
	s := &script{t: t, secrets: []string{"kilo-wrong-1", "kilo-wrong-2", "kilo-wrong-3"}}
	c, err := SignIn(t.Context(), "kilo", s, o, &memLogins{})
	if err == nil || err.Error() != "kilo refused 3 keys; none was saved" || !errors.Is(err, llm.ErrAuth) || c.Key != "" {
		t.Errorf("three keys the profile refuses: %+v, %v; want kilo refused 3 keys; none was saved", c, err)
	}
}

// A profile that cannot say is one warning, and the key is kept (0068-MADR
// L2).
func TestSignInKiloUnverifiedKeyWarns(t *testing.T) {
	ti, o := signInIssuer(t)
	ti.profile(http.StatusServiceUnavailable)
	o.Method = MethodAPIKey
	s := &script{t: t, secrets: []string{"kilo-key-0123456789"}}
	c, err := SignIn(t.Context(), "kilo", s, o, &memLogins{})
	if err != nil || c.Key != "kilo-key-0123456789" {
		t.Fatalf("a key the profile cannot check: %+v, %v; want the key", c, err)
	}
	n := 0
	for _, note := range s.noted() {
		if strings.Contains(note, "could not verify the Kilo key") {
			n++
		}
	}
	if n != 1 {
		t.Errorf("%d warnings that the key was not verified in %q, want one", n, s.noted())
	}
}
