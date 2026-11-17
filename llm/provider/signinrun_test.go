package provider

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/catalog"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
	"github.com/maccavelli/go-llmprovider-sdk/wizard"
)

// memClients is memLogins that also keeps the issued clients apart from the
// sessions, as internal/auth's ClientLogins does: a Delete leaves them.
type memClients struct {
	memLogins
	cmu     sync.Mutex
	records map[string]ClientRecord
}

func (m *memClients) LoadClient(_ context.Context, id string) (ClientRecord, error) {
	m.cmu.Lock()
	defer m.cmu.Unlock()
	return m.records[id], nil
}

func (m *memClients) SaveClient(_ context.Context, id string, r ClientRecord) error {
	m.cmu.Lock()
	defer m.cmu.Unlock()
	if m.records == nil {
		m.records = map[string]ClientRecord{}
	}
	m.records[id] = r
	return nil
}

// planIssuer is the test issuer answering as OpenAI's issuer for a ChatGPT
// plan sign-in: it grants chatgpt.tokens.use.direct, and signs ID tokens as
// OpenAI's issuer. (The SDK's own tests keep their plan issuer to
// themselves.) The options reach it through toIssuer.
func planIssuer(t *testing.T) (*testIssuer, SignInOptions) {
	t.Helper()
	ti := newTestIssuer(t)
	ti.origin = auth.DefaultOpenAIIssuer
	ti.scope = "openid chatgpt.tokens.use.direct"
	u, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	return ti, SignInOptions{ClientName: "gobble", ClientVersion: "1.2.3-test", HTTPClient: &http.Client{Transport: toIssuer{u}}}
}

// signInRecord is what drive saw of a sign-in.
type signInRecord struct {
	steps []authcontract.Step
	urls  []*url.URL // the open_url steps' URLs, parsed
}

// drive reads a browser sign-in's steps to the end, answering as a person who
// allows the consent and pastes the browser's callback would: a confirm with
// allow, a paste of the authorize URL's redirect with a code, its state and
// the issued client issued. The nonce goes to the issuer for its ID token.
func drive(t *testing.T, ti *testIssuer, run *SignInRun, allow bool, issued string) signInRecord {
	t.Helper()
	var rec signInRecord
	var redirect, state string
	for s := range run.Steps() {
		rec.steps = append(rec.steps, s)
		switch s.Kind {
		case authcontract.StepConfirm:
			if err := run.Answer(t.Context(), authcontract.Answer{SchemaVersion: authcontract.SchemaVersion, Seq: s.Seq, Confirm: &allow}); err != nil {
				t.Fatalf("answering the consent: %v", err)
			}
		case authcontract.StepOpenURL:
			u, err := url.Parse(s.URL)
			if err != nil {
				t.Fatal(err)
			}
			rec.urls = append(rec.urls, u)
			redirect, state = ti.authorizeParams(t, s.URL)
		case authcontract.StepPaste:
			cb := redirect + "?code=c1&state=" + url.QueryEscape(state)
			if issued != "" {
				cb += "&client_id=" + url.QueryEscape(issued)
			}
			a := authcontract.Answer{SchemaVersion: authcontract.SchemaVersion, Seq: s.Seq, Paste: cb}
			if err := run.Answer(t.Context(), a); err != nil {
				t.Fatalf("answering the paste: %v", err)
			}
		}
	}
	return rec
}

// startPlan starts the plan sign-in over store, remote.
func startPlan(t *testing.T, o SignInOptions, store LoginStore) *SignInRun {
	t.Helper()
	run, err := StartSignIn(t.Context(), "openai", string(llmprovider.AuthChatGPTPlan), o, store)
	if err != nil {
		t.Fatalf("StartSignIn: %v", err)
	}
	t.Cleanup(run.Cancel)
	return run
}

// recordStart replaces the SDK's engine with one that records the options it
// is given and returns an error, for a test of what StartSignIn passes.
func recordStart(t *testing.T) *[]signin.Options {
	t.Helper()
	var mu sync.Mutex
	var got []signin.Options
	old := startSignIn
	t.Cleanup(func() { startSignIn = old })
	startSignIn = func(_ context.Context, _ llmprovider.ProviderID, _ llmprovider.AuthMethodID, o signin.Options) (*signin.SignIn, error) {
		mu.Lock()
		got = append(got, o)
		mu.Unlock()
		return nil, errors.New("recorded")
	}
	return &got
}

// waitPlan is a plan sign-in driven to its end and waited for.
func waitPlan(t *testing.T, ti *testIssuer, o SignInOptions, store LoginStore, allow bool, issued string) (signInRecord, SignInResult) {
	t.Helper()
	run := startPlan(t, o, store)
	rec := drive(t, ti, run, allow, issued)
	res, err := run.Wait(t.Context())
	if err != nil {
		t.Fatalf("the plan sign-in: %v (%v)", err, errors.Unwrap(err))
	}
	return rec, res
}

// After a logout that keeps clients.json, the next plan sign-in names the
// stored client and the host id and sends no agent_name_hint, the real engine
// against a plan issuer.
func TestStartSignInReusesClient(t *testing.T) {
	ti, o := planIssuer(t)
	o.Remote, o.HostID = true, "urn:uuid:11111111-2222-4333-8444-555555555555"
	store := &memClients{}
	first, res := waitPlan(t, ti, o, store, true, "oaiapp_issued1")
	if len(first.urls) != 1 || first.urls[0].Query().Get("agent_name_hint") != AgentName || res.Yields != authcontract.YieldsOAuthSession {
		t.Fatalf("the first sign-in: authorize %v, result %+v, want it to name the app", first.urls, res)
	}
	if got, _ := store.LoadClient(t.Context(), "openai"); got != (ClientRecord{ClientID: "oaiapp_issued1", HostID: o.HostID}) {
		t.Fatalf("the client kept %+v, want the issued one and the host id", got)
	}
	// A logout deletes the session and keeps the client.
	if err := store.Delete(t.Context(), "openai"); err != nil {
		t.Fatal(err)
	}
	second, _ := waitPlan(t, ti, o, store, true, "oaiapp_issued1")
	q := second.urls[0].Query()
	if q.Get("client_id") != "oaiapp_issued1" || q.Get("ext_agent_host_id") != o.HostID || q.Has("agent_name_hint") {
		t.Errorf("the second sign-in's authorize query %v, want the stored client, the host id and no agent_name_hint", q)
	}
}

// With a stored plan session that holds an answer, the steps hold no confirm
// and the authorize request carries id_token_hint; a stored session of
// another kind is no Previous.
func TestStartSignInPassesPrevious(t *testing.T) {
	ti, o := planIssuer(t)
	o.Remote, o.HostID = true, "urn:uuid:11111111-2222-4333-8444-555555555555"
	store := &memClients{}
	first, _ := waitPlan(t, ti, o, store, true, "oaiapp_issued1")
	if !slices.ContainsFunc(first.steps, func(s authcontract.Step) bool { return s.Kind == authcontract.StepConfirm }) {
		t.Fatal("the first sign-in asked no consent: the test holds nothing")
	}
	second, _ := waitPlan(t, ti, o, store, true, "oaiapp_issued1")
	if slices.ContainsFunc(second.steps, func(s authcontract.Step) bool { return s.Kind == authcontract.StepConfirm }) {
		t.Errorf("the second sign-in's steps %v hold a confirm, want none: the session holds the answer", second.steps)
	}
	if q := second.urls[0].Query(); q.Get("id_token_hint") == "" || q.Has("agent_name_hint") {
		t.Errorf("the second authorize query %v, want id_token_hint and no agent_name_hint", q)
	}

	// Another kind of session is not Previous.
	got := recordStart(t)
	other := &memLogins{logins: map[string]*Login{"openai": {Access: "a", Refresh: "r", Expires: time.Now().Add(time.Hour),
		Issuer: auth.DefaultOpenAIIssuer, ClientID: "app_test", TokenURL: auth.DefaultOpenAIIssuer + "/oauth/token"}}}
	if _, err := StartSignIn(t.Context(), "openai", string(llmprovider.AuthChatGPTPlan), o, other); err == nil {
		t.Fatal("the recorder gave no error")
	}
	if len(*got) != 1 || (*got)[0].Previous != nil {
		t.Errorf("a ChatGPT sign-in's session as Previous: %+v, want none", *got)
	}
}

// StartSignIn passes gobble's consent wording, the ID of its host and the
// client store, the app's identity and its name, as MADR D3 lists them.
func TestStartSignInPassesConsentText(t *testing.T) {
	got := recordStart(t)
	o := SignInOptions{ClientName: "gobble", ClientVersion: "1.2.3-test", HostID: "urn:uuid:test"}
	store := &memClients{}
	if _, err := StartSignIn(t.Context(), "openai", string(llmprovider.AuthChatGPTPlan), o, store); err == nil {
		t.Fatal("the recorder gave no error")
	}
	if len(*got) != 1 {
		t.Fatalf("%d calls, want 1", len(*got))
	}
	so := (*got)[0]
	if so.ConsentText["chatgpt_plan.background_use"] == "" || so.ConsentText["chatgpt_plan.background_use"] != ConsentText["chatgpt_plan.background_use"] {
		t.Errorf("ConsentText %v, want gobble's wording", so.ConsentText)
	}
	if so.AgentName != AgentName || so.HostID != "urn:uuid:test" || so.ClientStore == nil || so.NoLoopback {
		t.Errorf("options %+v, want the agent name, the host id and the client store, with a listener", so)
	}
	// A store with no client store passes none.
	*got = nil
	if _, err := StartSignIn(t.Context(), "openai", string(llmprovider.AuthChatGPTPlan), o, &memLogins{}); err == nil {
		t.Fatal("the recorder gave no error")
	}
	if (*got)[0].ClientStore != nil {
		t.Errorf("a store with no client store gave %+v", (*got)[0].ClientStore)
	}
}

// The real engine's confirm step shows gobble's wording, not the SDK's.
func TestStartSignInConfirmIsGobblesWording(t *testing.T) {
	ti, o := planIssuer(t)
	o.Remote = true
	rec, _ := waitPlan(t, ti, o, &memClients{}, true, "oaiapp_issued1")
	for _, s := range rec.steps {
		if s.Kind == authcontract.StepConfirm {
			if s.Text != ConsentText[s.ConfirmID] || s.Text == "" {
				t.Errorf("the confirm step's text %q, want gobble's wording", s.Text)
			}
			return
		}
	}
	t.Fatal("no confirm step")
}

// A remote sign-in sets NoLoopback for every method gobble offers, and a local
// one for none; the terminal paths' SignIn never sets it. The engine's own
// handling of the option is the SDK's test; this holds what gobble passes (the
// owner's ruling of 2026-11-16, "Assert what gobble passes; use no real
// ports").
func TestStartSignInRemoteSetsNoLoopback(t *testing.T) {
	got := recordStart(t)
	n := 0
	for _, id := range []string{"openai", "grok", "kilo", "claude", "xai"} {
		for _, m := range Offered(id) {
			n++
			for _, remote := range []bool{true, false} {
				*got = nil
				_, _ = StartSignIn(t.Context(), id, m.ID, SignInOptions{Remote: remote}, &memLogins{})
				if len(*got) != 1 || (*got)[0].NoLoopback != remote {
					t.Errorf("%s %s remote %v: recorded %+v, want NoLoopback %v", id, m.ID, remote, *got, remote)
				}
			}
		}
	}
	if n < 4 {
		t.Fatalf("%d offered methods were tried: the test holds nothing", n)
	}
	// SignIn takes no such option: Remote does not reach the wizard.
	var seen []wizard.Options
	oldResolve := resolveCredential
	t.Cleanup(func() { resolveCredential = oldResolve })
	resolveCredential = func(_ context.Context, _ wizard.Prompter, o wizard.Options, _ llmprovider.ProviderID) (wizard.Result, catalog.Catalog, error) {
		seen = append(seen, o)
		return wizard.Result{}, catalog.Catalog{}, errors.New("recorded")
	}
	_, _ = SignIn(t.Context(), "grok", &script{t: t}, SignInOptions{Remote: true}, &memLogins{})
	if len(seen) != 1 || seen[0].NoPastePrompt {
		t.Errorf("SignIn with Remote: wizard options %+v, want them as without it", seen)
	}
}

// A remote grok device sign-in against the issuer starts with no
// ErrUnsupported and completes by the poll, its options having NoLoopback.
func TestStartSignInRemoteDevice(t *testing.T) {
	ti, o := signInIssuer(t)
	_ = ti
	o.Remote = true
	var seen []signin.Options
	old := startSignIn
	t.Cleanup(func() { startSignIn = old })
	startSignIn = func(ctx context.Context, p llmprovider.ProviderID, m llmprovider.AuthMethodID, so signin.Options) (*signin.SignIn, error) {
		seen = append(seen, so)
		return old(ctx, p, m, so)
	}
	store := &memLogins{}
	run, err := StartSignIn(t.Context(), "grok", "device_code", o, store)
	if err != nil {
		t.Fatalf("a remote device sign-in: %v", err)
	}
	t.Cleanup(run.Cancel)
	var kinds []authcontract.StepKind
	for s := range run.Steps() {
		kinds = append(kinds, s.Kind)
	}
	if !slices.Equal(kinds, []authcontract.StepKind{authcontract.StepShowCode, authcontract.StepWaiting, authcontract.StepDone}) {
		t.Errorf("steps %v, want show_code, waiting, done", kinds)
	}
	res, err := run.Wait(t.Context())
	if err != nil || res.Yields != authcontract.YieldsOAuthSession || res.Key != "" {
		t.Fatalf("Wait = %+v, %v, want an oauth_session and no key", res, err)
	}
	if l, _ := store.Load(t.Context(), "grok"); l == nil || l.Access == "" {
		t.Errorf("the session was not saved: %v", l)
	}
	if len(seen) != 1 || !seen[0].NoLoopback {
		t.Errorf("recorded options %+v, want NoLoopback true", seen)
	}
}

// The host id reaches the wizard's options and the engine's.
func TestHostIDReachesTheEngines(t *testing.T) {
	const id = "urn:uuid:aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"
	var wiz []wizard.Options
	oldResolve := resolveCredential
	t.Cleanup(func() { resolveCredential = oldResolve })
	resolveCredential = func(_ context.Context, _ wizard.Prompter, o wizard.Options, _ llmprovider.ProviderID) (wizard.Result, catalog.Catalog, error) {
		wiz = append(wiz, o)
		return wizard.Result{}, catalog.Catalog{}, errors.New("recorded")
	}
	_, _ = SignIn(t.Context(), "openai", &script{t: t}, SignInOptions{HostID: id}, &memLogins{})
	eng := recordStart(t)
	_, _ = StartSignIn(t.Context(), "openai", "chatgpt_plan", SignInOptions{HostID: id}, &memLogins{})
	if len(wiz) != 1 || wiz[0].HostID != id || len(*eng) != 1 || (*eng)[0].HostID != id {
		t.Errorf("wizard options %+v, engine options %+v, want both to hold the host id", wiz, *eng)
	}
}

// The options carry gobble's identity, and every request of a sign-in names
// it: referrer=gobble on the device request, and a User-Agent that begins
// gobble/<version> ( on each.
func TestStartSignInNamesGobble(t *testing.T) {
	ti, o := signInIssuer(t)
	_ = ti
	got := recordStart(t)
	o.ClientVersion = "1.2.3-test"
	_, _ = StartSignIn(t.Context(), "grok", "device_code", o, &memLogins{})
	if len(*got) != 1 || (*got)[0].Identity != (auth.Identity{Name: "gobble", Version: "1.2.3-test"}) {
		t.Fatalf("recorded options %+v, want the identity gobble 1.2.3-test", *got)
	}
	var mu sync.Mutex
	type seen struct{ ua, body string }
	var reqs []seen
	base := o.HTTPClient.Transport
	o.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var b []byte
		if r.Body != nil {
			b, _ = io.ReadAll(r.Body)
			r.Body = io.NopCloser(bytes.NewReader(b))
		}
		mu.Lock()
		reqs = append(reqs, seen{r.Header.Get("User-Agent"), string(b)})
		mu.Unlock()
		return base.RoundTrip(r)
	})}
	// The recorder is replaced by the real engine again.
	startSignIn = signin.Start
	run, err := StartSignIn(t.Context(), "grok", "device_code", o, &memLogins{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(run.Cancel)
	for range run.Steps() {
	}
	if _, err := run.Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(reqs) == 0 {
		t.Fatal("no request seen")
	}
	device := false
	for _, r := range reqs {
		if !strings.HasPrefix(r.ua, "gobble/1.2.3-test (") {
			t.Errorf("User-Agent %q, want it to begin gobble/1.2.3-test (", r.ua)
		}
		if strings.Contains(r.body, "referrer=gobble") {
			device = true
		}
	}
	if !device {
		t.Error("no request carried referrer=gobble")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// Every error of a sign-in is a *SignInError, whose kind ErrorKind reads from
// the SDK: a denial, an offline failure (never other), a start refused and a
// cancelled sign-in (0051-MADR D6).
func TestStartSignInErrorsAreSignInErrors(t *testing.T) {
	wantKind := func(name string, err error, kind authcontract.ErrorKind) {
		t.Helper()
		if _, ok := errors.AsType[*SignInError](err); !ok {
			t.Errorf("%s: %T (%v) is not a *SignInError", name, err, err)
		}
		if got := ErrorKind(err); got != string(kind) {
			t.Errorf("%s: kind %q, want %q", name, got, kind)
		}
	}
	// A denial.
	ti, o := signInIssuer(t)
	ti.with(func() { ti.deny = true })
	o.Remote = true
	run, err := StartSignIn(t.Context(), "grok", "device_code", o, &memLogins{})
	if err != nil {
		t.Fatal(err)
	}
	for range run.Steps() {
	}
	_, err = run.Wait(t.Context())
	wantKind("a denied sign-in", err, authcontract.FailureDenied)
	// No network.
	// No network: a port nothing listens on, on the loopback, refuses.
	dead := newTestIssuer(t)
	deadURL, err := url.Parse(dead.URL)
	if err != nil {
		t.Fatal(err)
	}
	dead.Close()
	_, o = signInIssuer(t)
	o.HTTPClient = &http.Client{Transport: toIssuer{deadURL}}
	run, err = StartSignIn(t.Context(), "grok", "device_code", o, &memLogins{})
	if err != nil {
		wantKind("an offline start", err, authcontract.FailureOffline)
	} else {
		for range run.Steps() {
		}
		_, err = run.Wait(t.Context())
		wantKind("an offline sign-in", err, authcontract.FailureOffline)
	}
	// A store that cannot be read.
	_, err = StartSignIn(t.Context(), "grok", "device_code", SignInOptions{}, &unreadable{})
	wantKind("an unreadable store", err, authcontract.FailureOther)
	// A method gobble does not offer for the provider.
	_, err = StartSignIn(t.Context(), "grok", "no_such_method", SignInOptions{}, &memLogins{})
	wantKind("a method not offered", err, authcontract.FailureInvalidRequest)
	_, err = StartSignIn(t.Context(), "grok", "api_key", SignInOptions{}, &memLogins{})
	wantKind("a key method the SDK does not drive", err, authcontract.FailureUnsupported)
	// A cancel.
	_, o = signInIssuer(t)
	run, err = StartSignIn(t.Context(), "grok", "device_code", o, &memLogins{})
	if err != nil {
		t.Fatal(err)
	}
	run.Cancel()
	_, err = run.Wait(t.Context())
	wantKind("a cancelled sign-in", err, authcontract.FailureCanceled)
}

// A Kilo device sign-in returns its key with yields device_key, and saves no
// session; a session sign-in saves one, and nothing is stored before Wait
// returns: a fresh sign-in that is cancelled or fails leaves what was stored
// as it was (0051-MADR D4 step 3).
func TestStartSignInStoresOnlyInWait(t *testing.T) {
	ti, o := signInIssuer(t)
	_ = ti
	old := &Login{Access: "OLD-ACCESS", Refresh: "OLD-REFRESH", Expires: time.Now().Add(time.Hour), Issuer: auth.DefaultGrokOAuthIssuer,
		ClientID: auth.DefaultGrokOAuthClientID, Kind: string(auth.SessionGrok)}
	store := &memLogins{logins: map[string]*Login{"grok": old}}
	snapshot := func() Login {
		l, _ := store.Load(t.Context(), "grok")
		if l == nil {
			t.Fatal("the stored login is gone")
		}
		return *l
	}
	before := snapshot()
	run, err := StartSignIn(t.Context(), "grok", "device_code", o, store)
	if err != nil {
		t.Fatal(err)
	}
	if got := snapshot(); got != before {
		t.Fatalf("StartSignIn changed the stored login: %+v, want %+v", got, before)
	}
	run.Cancel()
	if _, err := run.Wait(t.Context()); err == nil {
		t.Fatal("a cancelled sign-in succeeded")
	}
	if got := snapshot(); got != before {
		t.Errorf("a cancelled sign-in changed the stored login: %+v", got)
	}
	// A completed one replaces it, in Wait.
	run, err = StartSignIn(t.Context(), "grok", "device_code", o, store)
	if err != nil {
		t.Fatal(err)
	}
	for range run.Steps() {
	}
	if got := snapshot(); got != before {
		t.Errorf("the steps ended and the stored login changed before Wait: %+v", got)
	}
	if _, err := run.Wait(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(); got.Access != "access-1" {
		t.Errorf("after Wait the stored login is %+v, want the new session", got)
	}
	// Kilo's key comes back, nothing saved.
	ks := &memLogins{}
	run, err = StartSignIn(t.Context(), "kilo", "device_code", o, ks)
	if err != nil {
		t.Fatal(err)
	}
	for range run.Steps() {
	}
	res, err := run.Wait(t.Context())
	if err != nil || res.Key != "kilo-token-0123456789" || res.Yields != authcontract.YieldsDeviceKey {
		t.Fatalf("kilo: %+v, %v, want its key with device_key", res, err)
	}
	if l, _ := ks.Load(t.Context(), "kilo"); l != nil {
		t.Errorf("a session was saved for a key sign-in: %v", l)
	}
}

// ownCache gives o a listing cache of the test's own: the SDK keeps a listing
// ten minutes per process, so another test's would answer for this issuer.
func ownCache(t *testing.T, o *SignInOptions) {
	t.Helper()
	c := &Cache{}
	o.Cache = c
	t.Cleanup(func() {
		if err := c.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
}

// The listing check comes before the save (the owner's ruling of 2026-11-16,
// "Check first, save only if accepted"): a listing the provider refuses with
// 401 fails Wait with kind auth, not retryable, and the store is exactly as it
// was, for a device sign-in and a browser sign-in by paste, with a login stored
// and with none; a 503 listing is no refusal and the session is saved, as the
// wizard saves after a listing that failed otherwise (wizard/configure.go,
// stepRefusal).
func TestStartSignInChecksListingBeforeSaving(t *testing.T) {
	for _, method := range []string{"device_code", "browser_oauth"} {
		for _, stored := range []bool{false, true} {
			t.Run(method+map[bool]string{true: " stored", false: " none"}[stored], func(t *testing.T) {
				ti, o := signInIssuer(t)
				o.Remote = true
				ownCache(t, &o)
				store := &memLogins{}
				var before Login
				if stored {
					before = Login{Access: "OLD-ACCESS", Refresh: "OLD-REFRESH", Expires: time.Now().Add(time.Hour), Issuer: auth.DefaultGrokOAuthIssuer,
						ClientID: auth.DefaultGrokOAuthClientID, Kind: string(auth.SessionGrok)}
					old := before
					store = &memLogins{logins: map[string]*Login{"grok": &old}}
				}
				ti.with(func() { ti.listStatus = http.StatusUnauthorized })
				run, err := StartSignIn(t.Context(), "grok", method, o, store)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(run.Cancel)
				drive(t, ti, run, true, "")
				_, err = run.Wait(t.Context())
				if err == nil {
					t.Fatal("a sign-in whose listing was refused succeeded")
				}
				if got := ErrorKind(err); got != string(authcontract.FailureAuth) {
					t.Errorf("kind %q, want auth", got)
				}
				if Retryable(err) {
					t.Errorf("a refused credential is retryable: %v", err)
				}
				got, _ := store.Load(t.Context(), "grok")
				switch {
				case stored && (got == nil || *got != before):
					t.Errorf("the stored login is %+v, want it as it was, %+v", got, before)
				case !stored && got != nil:
					t.Errorf("a refused sign-in stored %+v", got)
				}
			})
		}
	}
	// Every other listing failure saves, as the wizard's stepRefusal lets it:
	// a refusal to permit (403), an outage (503), a bad request, a missing
	// route and a rate limit (wizard/configure.go, refusedCredential).
	for _, status := range []int{http.StatusForbidden, http.StatusServiceUnavailable, http.StatusBadRequest, http.StatusNotFound, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status)+" listing saves", func(t *testing.T) {
			ti, o := signInIssuer(t)
			ownCache(t, &o)
			store := &memLogins{}
			ti.with(func() { ti.listStatus = status })
			run, err := StartSignIn(t.Context(), "grok", "device_code", o, store)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(run.Cancel)
			for range run.Steps() {
			}
			if _, err := run.Wait(t.Context()); err != nil {
				t.Fatalf("a %d listing failed the sign-in: %v", status, err)
			}
			if l, _ := store.Load(t.Context(), "grok"); l == nil || l.Access != "access-1" {
				t.Errorf("stored %+v, want the new session", l)
			}
		})
	}
}

// renewingIssuer is signInIssuer whose listing answers 401 to the first access
// token, as an issuer slow to propagate it would, so that the SDK renews the
// session before it lists again; refresh answers the renewal: a rotated pair
// when it returns "", else the status it returns with an error body. It
// returns the refresh tokens the renewals were sent.
func renewingIssuer(t *testing.T, refresh func(n int) (status int, body string)) (SignInOptions, func() []string) {
	t.Helper()
	ti, o := signInIssuer(t)
	ownCache(t, &o)
	u, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	base := toIssuer{u}
	var mu sync.Mutex
	var sent []string
	o.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		reply := func(status int, body string) (*http.Response, error) {
			return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Request: r, Body: io.NopCloser(strings.NewReader(body))}, nil
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/oauth2/token") {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			r.Body = io.NopCloser(bytes.NewReader(b))
			f, err := url.ParseQuery(string(b))
			if err != nil {
				return nil, err
			}
			if f.Get("grant_type") == "refresh_token" {
				mu.Lock()
				sent = append(sent, f.Get("refresh_token"))
				n := len(sent)
				mu.Unlock()
				return reply(refresh(n))
			}
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/") && r.Header.Get("Authorization") == "Bearer access-1" {
			return reply(http.StatusUnauthorized, `{"error":{"message":"token not yet valid"}}`)
		}
		return base.RoundTrip(r)
	})}
	return o, func() []string { mu.Lock(); defer mu.Unlock(); return slices.Clone(sent) }
}

// signedIn signs in to Grok by device code with o over store and returns Wait's
// error.
func signedIn(t *testing.T, o SignInOptions, store LoginStore) error {
	t.Helper()
	run, err := StartSignIn(t.Context(), "grok", "device_code", o, store)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(run.Cancel)
	for range run.Steps() {
	}
	_, err = run.Wait(t.Context())
	return err
}

// A refresh the listing causes rotates the refresh token, and what is saved is
// the rotated pair, not the spent one (the review's B1; the wizard saves the
// session it listed through, wizard/auth.go, save). A renewal the issuer
// answers with a 503 saves the original, as the wizard does; one it refuses
// with a 400 invalid_grant is a refused session and saves nothing.
func TestStartSignInSavesTheRenewedSession(t *testing.T) {
	t.Run("rotated", func(t *testing.T) {
		o, sent := renewingIssuer(t, func(int) (int, string) {
			return 200, `{"access_token":"access-2","refresh_token":"refresh-2","expires_in":3600,"token_type":"Bearer"}`
		})
		store := &memLogins{}
		if err := signedIn(t, o, store); err != nil {
			t.Fatalf("Wait: %v", err)
		}
		if got := sent(); !slices.Equal(got, []string{"refresh-1"}) {
			t.Fatalf("renewals were sent %q, want one with refresh-1: the test holds nothing", got)
		}
		l, _ := store.Load(t.Context(), "grok")
		if l == nil || l.Access != "access-2" || l.Refresh != "refresh-2" {
			t.Errorf("stored %+v, want the rotated pair access-2 and refresh-2, not the spent one", l)
		}
	})
	t.Run("renewal unavailable", func(t *testing.T) {
		o, sent := renewingIssuer(t, func(int) (int, string) { return 503, `{"error":"temporarily_unavailable"}` })
		store := &memLogins{}
		if err := signedIn(t, o, store); err != nil {
			t.Fatalf("a renewal that failed with a 503 failed the sign-in: %v", err)
		}
		if len(sent()) == 0 {
			t.Fatal("no renewal was sent: the test holds nothing")
		}
		if l, _ := store.Load(t.Context(), "grok"); l == nil || l.Refresh != "refresh-1" {
			t.Errorf("stored %+v, want the session as signed in", l)
		}
	})
	t.Run("renewal refused", func(t *testing.T) {
		o, sent := renewingIssuer(t, func(int) (int, string) { return 400, `{"error":"invalid_grant"}` })
		store := &memLogins{}
		err := signedIn(t, o, store)
		if len(sent()) == 0 {
			t.Fatal("no renewal was sent: the test holds nothing")
		}
		if err == nil {
			t.Fatal("a session the issuer refuses to renew was saved")
		}
		if l, _ := store.Load(t.Context(), "grok"); l != nil {
			t.Errorf("a refused renewal stored %+v", l)
		}
	})
}

// A sign-in whose context ends while the listing is in flight saves nothing,
// with a login stored or with none (the review's B3; D4 step 3): the listing's
// failure is no refusal, so the context is what stops the save.
func TestStartSignInCancelledDuringListingSavesNothing(t *testing.T) {
	for _, stored := range []bool{false, true} {
		t.Run(map[bool]string{true: "stored", false: "none"}[stored], func(t *testing.T) {
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
					cancel() // the client's cancel arrives while the listing runs
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				return base.RoundTrip(r)
			})}
			store := &memLogins{}
			var before Login
			if stored {
				before = Login{Access: "OLD-ACCESS", Refresh: "OLD-REFRESH", Expires: time.Now().Add(time.Hour), Issuer: auth.DefaultGrokOAuthIssuer,
					ClientID: auth.DefaultGrokOAuthClientID, Kind: string(auth.SessionGrok)}
				old := before
				store = &memLogins{logins: map[string]*Login{"grok": &old}}
			}
			run, err := StartSignIn(ctx, "grok", "device_code", o, store)
			if err != nil {
				t.Fatal(err)
			}
			for range run.Steps() {
			}
			_, err = run.Wait(ctx)
			if err == nil {
				t.Fatal("a sign-in cancelled during its listing succeeded")
			}
			if got := ErrorKind(err); got != string(authcontract.FailureCanceled) {
				t.Errorf("kind %q, want canceled", got)
			}
			got, _ := store.Load(t.Context(), "grok")
			switch {
			case stored && (got == nil || *got != before):
				t.Errorf("the stored login is %+v, want it as it was", got)
			case !stored && got != nil:
				t.Errorf("a cancelled sign-in stored %+v", got)
			}
		})
	}
}

// The ChatGPT plan's sign-in is listed too: a 401 refuses it and stores
// neither the session nor, in the store the sign-in was given, anything else of
// the credential; a 403 is no refusal and saves (the review's B5).
func TestStartSignInChecksPlanListing(t *testing.T) {
	for _, tc := range []struct {
		status  int
		refused bool
	}{{http.StatusUnauthorized, true}, {http.StatusForbidden, false}} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			ti, o := planIssuer(t)
			o.Remote = true
			ownCache(t, &o)
			ti.with(func() { ti.listStatus = tc.status })
			var mu sync.Mutex
			listed := 0
			inner := o.HTTPClient.Transport
			o.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/") {
					mu.Lock()
					listed++
					mu.Unlock()
				}
				return inner.RoundTrip(r)
			})}
			store := &memClients{}
			run := startPlan(t, o, store)
			drive(t, ti, run, true, "oaiapp_issued1")
			_, err := run.Wait(t.Context())
			mu.Lock()
			n := listed
			mu.Unlock()
			if n == 0 {
				t.Fatal("the plan's sign-in was not listed: the test holds nothing")
			}
			l, _ := store.Load(t.Context(), "openai")
			switch {
			case tc.refused && (err == nil || ErrorKind(err) != string(authcontract.FailureAuth)):
				t.Errorf("a refused plan listing gave %v (kind %s), want an auth failure", err, ErrorKind(err))
			case tc.refused && l != nil:
				t.Errorf("a refused plan sign-in stored %+v", l)
			case !tc.refused && (err != nil || l == nil):
				t.Errorf("a %d plan listing gave %v, stored %v, want the session saved", tc.status, err, l)
			}
		})
	}
}

// Kilo's device key is listed with: the SDK sends the key Kilo's public listing
// is given, and the wizard's device-key path lists with it and refuses a key
// the listing refuses (wizard/configure.go); so a 401 on that listing fails the
// sign-in and no key comes back, and a 403 does not.
func TestStartSignInChecksKiloDeviceKey(t *testing.T) {
	const key = "kilo-token-0123456789"
	for _, tc := range []struct {
		status  int
		refused bool
	}{{http.StatusUnauthorized, true}, {http.StatusForbidden, false}} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			ti, o := signInIssuer(t)
			ownCache(t, &o)
			u, err := url.Parse(ti.URL)
			if err != nil {
				t.Fatal(err)
			}
			base := toIssuer{u}
			var mu sync.Mutex
			withKey := 0
			o.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/api/device-auth/") && r.Header.Get("Authorization") == "Bearer "+key {
					mu.Lock()
					withKey++
					mu.Unlock()
					return &http.Response{StatusCode: tc.status, Header: http.Header{"Content-Type": {"application/json"}}, Request: r,
						Body: io.NopCloser(strings.NewReader(`{"error":{"message":"` + serviceText + `"}}`))}, nil
				}
				return base.RoundTrip(r)
			})}
			store := &memLogins{}
			run, err := StartSignIn(t.Context(), "kilo", "device_code", o, store)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(run.Cancel)
			for range run.Steps() {
			}
			res, err := run.Wait(t.Context())
			mu.Lock()
			n := withKey
			mu.Unlock()
			if n == 0 {
				t.Fatal("the listing was not sent with the device key: the test holds nothing")
			}
			switch {
			case tc.refused && (err == nil || res.Key != ""):
				t.Errorf("a refused device key gave %+v, %v, want no key and an error", res, err)
			case tc.refused && ErrorKind(err) != string(authcontract.FailureAuth):
				t.Errorf("kind %q, want auth", ErrorKind(err))
			case !tc.refused && (err != nil || res.Key != key):
				t.Errorf("a %d listing gave %+v, %v, want the key", tc.status, res, err)
			}
			if l, _ := store.Load(t.Context(), "kilo"); l != nil {
				t.Errorf("a session was saved for a key sign-in: %v", l)
			}
		})
	}
}

// sdkStore is the SDK's ClientStore exactly when its store is a ClientStore,
// and the records pass both ways.
func TestClientStoreAdapter(t *testing.T) {
	if _, ok := tokenStore(&memLogins{}).(auth.ClientStore); ok {
		t.Error("a store with no client store is an auth.ClientStore")
	}
	m := &memClients{}
	cs, ok := tokenStore(m).(auth.ClientStore)
	if !ok {
		t.Fatal("a store with a client store is not an auth.ClientStore")
	}
	if err := cs.SaveClient(t.Context(), llmprovider.ProviderOpenAI, auth.ClientRecord{ClientID: "c1", HostID: "h1"}); err != nil {
		t.Fatal(err)
	}
	if got := m.records["openai"]; got != (ClientRecord{ClientID: "c1", HostID: "h1"}) {
		t.Errorf("the store holds %+v, want the record written through the adapter", got)
	}
	m.records["openai"] = ClientRecord{ClientID: "c2", HostID: "h2"}
	if got, err := cs.LoadClient(t.Context(), llmprovider.ProviderOpenAI); err != nil || got != (auth.ClientRecord{ClientID: "c2", HostID: "h2"}) {
		t.Errorf("LoadClient = %+v, %v, want the store's record", got, err)
	}
	// Both stay the SDK's token store.
	if _, ok := tokenStore(m).(auth.RefreshLocker); !ok {
		t.Error("the client-store adapter is no RefreshLocker")
	}
}

// A plan sign-in that is refused by the listing, or cancelled during it, leaves
// the stored credentials untouched; the issued client record, which the SDK
// saves when the client is issued, may exist, and the next sign-in reuses it
// (the owner's ruling of 2026-11-16, "Accept it and say so in D4": "nothing
// stored" means credentials).
func TestStartSignInPlanFailureKeepsCredentialsAndMayKeepClient(t *testing.T) {
	for _, how := range []string{"refused", "cancelled"} {
		t.Run(how, func(t *testing.T) {
			ti, o := planIssuer(t)
			o.Remote, o.HostID = true, "urn:uuid:11111111-2222-4333-8444-555555555555"
			ownCache(t, &o)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			inner := o.HTTPClient.Transport
			o.HTTPClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if how == "cancelled" && r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/") {
					cancel()
					<-r.Context().Done()
					return nil, r.Context().Err()
				}
				return inner.RoundTrip(r)
			})}
			if how == "refused" {
				ti.with(func() { ti.listStatus = http.StatusUnauthorized })
			}
			store := &memClients{}
			run, err := StartSignIn(ctx, "openai", string(llmprovider.AuthChatGPTPlan), o, store)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(run.Cancel)
			drive(t, ti, run, true, "oaiapp_issued1")
			if _, err := run.Wait(ctx); err == nil {
				t.Fatalf("a %s plan sign-in succeeded", how)
			}
			if l, _ := store.Load(t.Context(), "openai"); l != nil || len(store.logins) != 0 {
				t.Errorf("the login store holds %+v, want it untouched", store.logins)
			}
			if got, _ := store.LoadClient(t.Context(), "openai"); got != (ClientRecord{ClientID: "oaiapp_issued1", HostID: o.HostID}) {
				t.Errorf("the client record is %+v, want the issued client kept for the next sign-in", got)
			}
		})
	}
}

// unreadable is a LoginStore whose Load fails.
type unreadable struct{ memLogins }

func (*unreadable) Load(context.Context, string) (*Login, error) { return nil, errors.New("disk") }

// listingBound is the SDK's bound on a model listing, 10 s (catalog.List, "bounded
// by the SDK at 10 s", llm/provider/list.go's List); the renewals below take
// longer, so the renewal outlives the listing and finishes after it returned.
const listingBound = 10 * time.Second

// A renewal that outlives the listing's bound is what the token wait before the
// save is for (the wizard's save does the same): one that then succeeds saves
// the rotated pair, and one the issuer refuses saves nothing, kind auth. Each
// takes about 12 s, which is its cost: listingBound plus a second.
func TestStartSignInRenewalOutlivesTheListing(t *testing.T) {
	for _, refused := range []bool{false, true} {
		t.Run(map[bool]string{false: "rotated", true: "refused"}[refused], func(t *testing.T) {
			ti, o := signInIssuer(t)
			ownCache(t, &o)
			u, err := url.Parse(ti.URL)
			if err != nil {
				t.Fatal(err)
			}
			base := toIssuer{u}
			renewals := new(counter)
			o.HTTPClient = &http.Client{Transport: lateRenewals(base, refused, renewals)}
			store := &memLogins{}
			err = signedIn(t, o, store)
			if renewals.get() == 0 {
				t.Fatal("no renewal was sent: the test holds nothing")
			}
			l, _ := store.Load(t.Context(), "grok")
			switch {
			case refused && (err == nil || ErrorKind(err) != string(authcontract.FailureAuth)):
				t.Errorf("a renewal the issuer refused gave %v (kind %s), want an auth failure", err, ErrorKind(err))
			case refused && l != nil:
				t.Errorf("a refused renewal stored %+v", l)
			case !refused && (err != nil || l == nil || l.Access != "access-2" || l.Refresh != "refresh-2"):
				t.Errorf("a late renewal gave %v, stored %+v, want the rotated pair", err, l)
			}
		})
	}
}

// counter is a count the transport's goroutines and the test share.
type counter struct {
	mu sync.Mutex
	n  int
}

func (c *counter) inc() { c.mu.Lock(); c.n++; c.mu.Unlock() }
func (c *counter) get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

// lateRenewals is the issuer's transport for a sign-in whose first access token
// the listing refuses, so that the SDK renews the session, and whose renewal
// answers listingBound plus a second later: with a rotated pair, or, refused,
// invalid_grant. Every other request goes to base.
func lateRenewals(base http.RoundTripper, refused bool, renewals *counter) roundTripFunc {
	return func(r *http.Request) (*http.Response, error) {
		reply := func(status int, body string) (*http.Response, error) {
			return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": {"application/json"}}, Request: r, Body: io.NopCloser(strings.NewReader(body))}, nil
		}
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/oauth2/token") {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				return nil, err
			}
			f, err := url.ParseQuery(string(b))
			if err != nil {
				return nil, err
			}
			if f.Get("grant_type") == "refresh_token" {
				renewals.inc()
				select {
				case <-r.Context().Done():
					return nil, r.Context().Err()
				case <-time.After(listingBound + time.Second):
				}
				if refused {
					return reply(http.StatusBadRequest, `{"error":"invalid_grant"}`)
				}
				return reply(http.StatusOK, `{"access_token":"access-2","refresh_token":"refresh-2","expires_in":3600,"token_type":"Bearer"}`)
			}
			r.Body = io.NopCloser(bytes.NewReader(b))
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v1/") && r.Header.Get("Authorization") == "Bearer access-1" {
			return reply(http.StatusUnauthorized, `{"error":{"message":"not yet"}}`)
		}
		return base.RoundTrip(r)
	}
}
