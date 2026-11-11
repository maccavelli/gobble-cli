package provider

import (
	"context"
	"errors"
	"fmt"
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
	confirm func(question string) bool
	asked   []string
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
	s.mu.Lock()
	s.asked = append(s.asked, question)
	s.mu.Unlock()
	if s.confirm == nil {
		s.t.Errorf("asked %q with no answer", question)
		return false, errors.New("no answer")
	}
	return s.confirm(question), nil
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
// and device sign-ins; Kilo's key and device sign-in; OpenAI's key and its
// ChatGPT plan, never the Codex client's sign-ins (0031-MADR D7, amended
// 2026-11-10); a key alone for the rest; none for a provider that takes no
// key.
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

	// OpenAI's menu is its key and its ChatGPT plan, under gobble's own
	// name: never the Codex client's browser or device sign-in
	// (0031-MADR D7, amended 2026-11-10).
	s = &script{t: t, secrets: []string{"sk-good-key-0123456789"}, choose: func(string, []Choice) int { return 0 }}
	if c, err := SignIn(t.Context(), "openai", s, o, store); err != nil || c.Key != "sk-good-key-0123456789" {
		t.Fatalf("openai's key: %+v, %v", c, err)
	}
	want = labels(t, "openai", llmprovider.AuthAPIKey, llmprovider.AuthChatGPTPlan)
	if len(s.menus) != 1 || len(want) != 2 {
		t.Fatalf("menus %v, want one of %q", s.menus, want)
	}
	got = nil
	for _, c := range s.menus[0] {
		got = append(got, c.Label)
	}
	if !slices.Equal(got, want) {
		t.Errorf("openai's menu %q, want %q", got, want)
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

// planIssuer is the test issuer answering as OpenAI's, granting ChatGPT
// plan usage, and the options of a plan sign-in that reach it.
func planIssuer(t *testing.T) (*testIssuer, SignInOptions) {
	t.Helper()
	ti := newTestIssuer(t)
	ti.origin, ti.scope = auth.DefaultOpenAIIssuer, "openid profile email offline_access resource.invoke chatgpt.tokens.use.direct"
	u, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	return ti, SignInOptions{ClientName: "gobble", ClientVersion: "test", Method: MethodChatGPTPlan,
		HTTPClient: &http.Client{Transport: toIssuer{u}}}
}

// planCallback is where the issuer sends the browser after a plan sign-in:
// the code, the state, and the client it issued gobble.
func planCallback(t *testing.T, ti *testIssuer, authorize string) string {
	t.Helper()
	redirect, state := ti.authorizeParams(t, authorize)
	return redirect + "?code=plan-code&state=" + url.QueryEscape(state) + "&client_id=oaiapp_gobbletest"
}

// waitPaste is a paste prompt nobody answers: it gives up when the sign-in
// ends.
func waitPaste(ctx context.Context, _ string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

// OpenAI's ChatGPT plan sign-in, under gobble's own name, saves a plan
// session with its four fields; OpenAI's one-time notice is asked first.
// A second sign-in signs in to the client OpenAI issued again, with the ID
// token as its hint and the host kept, and asks no notice (0031-MADR D7,
// amended 2026-11-10).
func TestSignInChatGPTPlan(t *testing.T) {
	ti, o := planIssuer(t)
	var mu sync.Mutex
	var authorizes []string
	o.OpenURL = func(u string) error {
		mu.Lock()
		authorizes = append(authorizes, u)
		mu.Unlock()
		back := planCallback(t, ti, u)
		go func() {
			resp, err := http.Get(back) //nolint:noctx // the test's browser
			if err == nil {
				_ = resp.Body.Close()
			}
		}()
		return nil
	}
	store := &memLogins{}
	s := &script{t: t, confirm: func(string) bool { return true }, input: waitPaste}
	c, err := SignIn(t.Context(), "openai", s, o, store)
	l := store.logins["openai"]
	if err != nil || c.Key != "" || !l.Plan() {
		t.Fatalf("the plan sign-in: %+v, %v, saved %v", c, err, l)
	}
	if l.Resource != "https://api.openai.com/v1" || !strings.Contains(l.Scope, "chatgpt.tokens.use.direct") ||
		l.IDToken == "" || !strings.HasPrefix(l.HostID, "urn:uuid:") || l.ClientID != "oaiapp_gobbletest" {
		t.Errorf("the saved session: %v (resource %q scope %q host %q)", l, l.Resource, l.Scope, l.HostID)
	}
	if len(s.asked) != 1 || !strings.Contains(s.asked[0], "ChatGPT plan") {
		t.Errorf("asked %q, want OpenAI's one notice", s.asked)
	}
	if strings.Contains(l.String(), l.IDToken) || strings.Contains(l.String(), "access-1") {
		t.Errorf("String shows a secret: %s", l)
	}
	first, err := url.Parse(authorizes[0])
	if err != nil {
		t.Fatal(err)
	}
	if q := first.Query(); q.Get("agent_name_hint") != AgentName || q.Get("client_id") == "oaiapp_gobbletest" {
		t.Errorf("the first authorize: agent %q, client %q; want gobble's name and registration", q.Get("agent_name_hint"),
			q.Get("client_id"))
	}

	host, idToken := l.HostID, l.IDToken
	again := &script{t: t, input: waitPaste}
	if _, err := SignIn(t.Context(), "openai", again, o, store); err != nil {
		t.Fatalf("a second sign-in: %v", err)
	}
	second, err := url.Parse(authorizes[1])
	if err != nil {
		t.Fatal(err)
	}
	if q := second.Query(); q.Get("client_id") != "oaiapp_gobbletest" || q.Get("id_token_hint") != idToken {
		t.Errorf("the second authorize: client %q, hint %v; want the issued client and the ID token", q.Get("client_id"),
			q.Get("id_token_hint") != "")
	}
	if store.logins["openai"].HostID != host || len(again.asked) != 0 {
		t.Errorf("the second sign-in: host %q (was %q), asked %q", store.logins["openai"].HostID, host, again.asked)
	}
}

// A plan sign-in whose browser cannot reach the loopback, as a phone's
// cannot reach the host's: the user pastes the address the browser lands
// on, and the sign-in completes (0031-MADR D7, P4b-Q4 (a)).
func TestSignInChatGPTPlanPasted(t *testing.T) {
	ti, o := planIssuer(t)
	landed := make(chan string, 1)
	o.OpenURL = func(u string) error {
		landed <- planCallback(t, ti, u)
		return nil
	}
	var prompts []string
	s := &script{t: t, confirm: func(string) bool { return true }, input: func(ctx context.Context, prompt string) (string, error) {
		prompts = append(prompts, prompt)
		select {
		case u := <-landed:
			return u, nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}}
	store := &memLogins{}
	if _, err := SignIn(t.Context(), "openai", s, o, store); err != nil || !store.logins["openai"].Plan() {
		t.Fatalf("a pasted plan sign-in: %v, saved %v", err, store.logins["openai"])
	}
	if len(prompts) == 0 || !strings.Contains(prompts[0], "redirected URL") {
		t.Errorf("prompts %q, want the paste prompt", prompts)
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
