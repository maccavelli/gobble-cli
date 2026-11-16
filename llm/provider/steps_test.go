package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/x/signin"
)

// stepLog records the steps SignInOptions.Steps was given.
type stepLog struct {
	mu    sync.Mutex
	steps []authcontract.Step
}

func (l *stepLog) add(s authcontract.Step) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.steps = append(l.steps, s)
	return nil
}

func (l *stepLog) kinds() []authcontract.StepKind {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]authcontract.StepKind, len(l.steps))
	for i, s := range l.steps {
		out[i] = s.Kind
	}
	return out
}

// withSteps are o's options for a caller that only reads steps: the steps go
// to l, and the terminal's two hooks fail the test if they are called.
func withSteps(t *testing.T, o SignInOptions, l *stepLog) SignInOptions {
	t.Helper()
	o.Steps = l.add
	o.ShowDeviceCode = func(DeviceCode) error { t.Error("ShowDeviceCode was called with Steps set"); return nil }
	o.OpenURL = func(string) error { t.Error("OpenURL was called with Steps set"); return nil }
	return o
}

// A device sign-in gives Steps the contract's steps in order, show_code with
// the URL, the code and its expiry, then waiting, each with the SDK's seq,
// and never the last one, which the caller states; the terminal's hooks are
// not called, and nothing is asked (0051-MADR D3, D5). It is the SDK's own
// wizard, run against the issuer.
func TestSignInStepsDevice(t *testing.T) {
	ti, o := signInIssuer(t)
	o.Method = MethodDevice
	store := &memLogins{}
	var l stepLog
	s := &script{t: t}
	if c, err := SignIn(t.Context(), "grok", s, withSteps(t, o, &l), store); err != nil || c.Key != "" || store.logins["grok"] == nil || c.Yields != authcontract.YieldsOAuthSession {
		t.Fatalf("grok's device sign-in: %+v, %v, saved %v", c, err, store.logins)
	}
	if got := l.kinds(); len(got) != 2 || got[0] != authcontract.StepShowCode || got[1] != authcontract.StepWaiting {
		t.Fatalf("grok's steps %v, want show_code then waiting, and no done", got)
	}
	show := l.steps[0]
	if show.SchemaVersion != authcontract.SchemaVersion || show.Seq != 1 || l.steps[1].Seq != 2 ||
		show.Code != "ABCD-1234" || show.URL != ti.URL+"/device" || show.ExpiresAt.IsZero() {
		t.Errorf("grok's show_code %+v, waiting seq %d", show, l.steps[1].Seq)
	}

	var kilo stepLog
	c, err := SignIn(t.Context(), "kilo", s, withSteps(t, o, &kilo), store)
	if err != nil || c.Key != "kilo-token-0123456789" || !c.Device || c.Yields != authcontract.YieldsDeviceKey {
		t.Fatalf("kilo's device sign-in: %+v, %v", c, err)
	}
	if got := kilo.kinds(); len(got) != 2 || got[0] != authcontract.StepShowCode || kilo.steps[0].Code != "KILO-1" || kilo.steps[0].URL != ti.URL+"/kilo" {
		t.Errorf("kilo's steps %+v", kilo.steps)
	}
}

// A browser sign-in with Steps asks for no paste, though the caller did not say
// NoPaste: its open_url step has no accepts, the Prompter is never read, and
// the callback alone completes it.
func TestSignInStepsBrowserAsksNoPaste(t *testing.T) {
	ti, o := signInIssuer(t)
	o.Method = MethodBrowser
	var l stepLog
	opts := withSteps(t, o, &l)
	play := browser(t, ti)
	opts.Steps = func(s authcontract.Step) error {
		if err := l.add(s); err != nil {
			return err
		}
		if s.Kind == authcontract.StepOpenURL {
			return play(s.URL)
		}
		return nil
	}
	store := &memLogins{}
	s := &script{t: t}
	if _, err := SignIn(t.Context(), "grok", s, opts, store); err != nil || store.logins["grok"] == nil {
		t.Fatalf("grok's browser sign-in: %v, saved %v", err, store.logins)
	}
	n := 0
	for _, st := range l.steps {
		switch st.Kind {
		case authcontract.StepOpenURL:
			n++
			if st.Accepts != authcontract.PasteNone || !strings.Contains(st.URL, "/oauth2/authorize?") {
				t.Errorf("the open_url step %+v, want the authorize URL and no accepts", st)
			}
		case authcontract.StepPaste:
			t.Errorf("a paste step was printed: %+v", st)
		}
	}
	if n != 1 || s.inputs != 0 {
		t.Errorf("%d open_url steps, the Prompter read %d times; want one and none", n, s.inputs)
	}
}

// A sign-in that asks a question ends with ErrAsks, an error of the kind
// unsupported in a *SignInError whose text names a terminal and an ACP client,
// and Steps has been given nothing: the first plan sign-in's confirm comes
// before its open_url. The paste, the confirm and the form refuse the same way
// when the driver is handed them, the steps that show something pass through
// and the last step is swallowed.
func TestSignInStepsRefuseWhatAsks(t *testing.T) {
	var l stepLog
	s := &script{t: t}
	o := withSteps(t, SignInOptions{Method: MethodChatGPTPlan, HTTPClient: &http.Client{Transport: refusingTransport{}}}, &l)
	// A sign-in that did not refuse would wait for a browser no one opens; the
	// deadline makes that a failure, not a hang.
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	_, err := SignIn(ctx, "openai", s, o, &memLogins{})
	se, ok := errors.AsType[*SignInError](err)
	if !ok || !errors.Is(err, ErrAsks) || ErrorKind(err) != "unsupported" {
		t.Fatalf("a first plan sign-in: %v (kind %s), want a *SignInError holding ErrAsks, unsupported", err, ErrorKind(err))
	}
	if msg := se.Error(); !strings.Contains(msg, "terminal") || !strings.Contains(msg, "ACP client") || strings.Contains(msg, "refusingTransport") {
		t.Errorf("the refusal's text %q, want gobble's, naming a terminal and an ACP client", msg)
	}
	if len(l.steps) != 0 {
		t.Errorf("Steps was given %v before the refusal", l.kinds())
	}

	d := stepDriver{steps: l.add}
	for _, k := range []authcontract.StepKind{authcontract.StepPaste, authcontract.StepConfirm, authcontract.StepForm} {
		if _, err := d.SignInStep(t.Context(), "grok", authcontract.Step{Kind: k}); !errors.Is(err, ErrAsks) {
			t.Errorf("the %s step: %v, want ErrAsks", k, err)
		}
	}
	for _, k := range []authcontract.StepKind{authcontract.StepDone, authcontract.StepFailed} {
		if _, err := d.SignInStep(t.Context(), "grok", authcontract.Step{Kind: k}); err != nil {
			t.Errorf("the %s step: %v, want it swallowed", k, err)
		}
	}
	for _, k := range []authcontract.StepKind{authcontract.StepOpenURL, authcontract.StepShowCode, authcontract.StepWaiting, "a_kind_not_known_yet"} {
		if _, err := d.SignInStep(t.Context(), "grok", authcontract.Step{Kind: k}); err != nil {
			t.Errorf("the %s step: %v, want it passed on", k, err)
		}
	}
	if got := l.kinds(); len(got) != 4 || got[3] != "a_kind_not_known_yet" {
		t.Errorf("Steps was given %v, want the four steps that show something", got)
	}
}

// What the wizard fails with, once it has started, comes back from a sign-in
// with Steps as a *SignInError, so ErrorKind reads the SDK's word for it and not
// llm.Kind's (0051-MADR D6): a refused device sign-in is denied, a refusal to
// answer is unsupported. It holds the two device sign-ins' refusals, and that
// Steps' own error comes back as it is. It does not hold the other errors
// SignIn returns unwrapped, which SignIn's doc comment lists: the context's own
// error (ErrorKind: canceled or timeout; TestAuthLoginJSONEndedBySignal holds
// canceled through the command), a Prompter's error such as the no-terminal one
// (llm.Kind: other), and the two errors before the wizard starts (other).
func TestSignInStepsErrorsAreSignInErrors(t *testing.T) {
	stop := errors.New("the caller's own error")
	cases := []struct {
		name string
		id   string
		deny bool
		fn   func(authcontract.Step) error
		kind string
		own  bool
	}{
		{name: "grok's device sign-in, denied", id: "grok", deny: true, kind: "denied"},
		{name: "kilo's device sign-in, denied", id: "kilo", deny: true, kind: "denied"},
		{name: "Steps' own error", id: "grok", fn: func(authcontract.Step) error { return stop }, own: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ti, o := signInIssuer(t)
			o.Method = MethodDevice
			ti.with(func() { ti.deny = tc.deny })
			var l stepLog
			opts := withSteps(t, o, &l)
			if tc.fn != nil {
				opts.Steps = tc.fn
			}
			_, err := SignIn(t.Context(), tc.id, &script{t: t}, opts, &memLogins{})
			if err == nil {
				t.Fatal("the sign-in succeeded")
			}
			if tc.own {
				if _, wrapped := errors.AsType[*SignInError](err); wrapped || !errors.Is(err, stop) {
					t.Errorf("Steps' error came back as %v, want it as it is", err)
				}
				return
			}
			if _, wrapped := errors.AsType[*SignInError](err); !wrapped {
				t.Errorf("the error %T %v is not a *SignInError", err, err)
			}
			if got, want := ErrorKind(err), tc.kind; got != want || got != string(signin.ErrorKindOf(err)) {
				t.Errorf("ErrorKind = %q, the SDK's %q, want %q", got, signin.ErrorKindOf(err), want)
			}
			for _, k := range l.kinds() {
				if k == authcontract.StepFailed || k == authcontract.StepDone {
					t.Errorf("Steps was given the last step %s", k)
				}
			}
		})
	}
}

// failingLogins is a LoginStore that cannot be read.
type failingLogins struct{ LoginStore }

var errUnreadable = errors.New("the store cannot be read")

func (failingLogins) Load(context.Context, string) (*Login, error) { return nil, errUnreadable }

// CheckNoQuestion refuses, before a sign-in starts, the ChatGPT plan's when
// the stored session of that method holds no answer to its required consent;
// an answer, granted or declined, lets it pass, as the SDK asks nothing then
// (0052-MADR D2). Another application's ChatGPT session, a key and no session
// at all hold no answer to it; a method with no consent passes; a store that
// cannot be read is its own error.
func TestCheckNoQuestion(t *testing.T) {
	const consent = "chatgpt_plan.background_use"
	plan := func(consents string) *Login {
		return &Login{Access: "a", Refresh: "r", Expires: time.Now().Add(time.Hour), Issuer: auth.DefaultOpenAIIssuer, ClientID: "oaiapp_test",
			Resource: "https://api.openai.com/v1", Kind: string(auth.SessionChatGPTPlan), Consents: consents}
	}
	other := &Login{Access: "a", Refresh: "r", Expires: time.Now().Add(time.Hour), Issuer: auth.DefaultOpenAIIssuer, ClientID: "app_test",
		Kind: string(auth.SessionChatGPT), Consents: consent + "=granted"}
	for _, tc := range []struct {
		name, id, method string
		stored           *Login
		refused          bool
	}{
		{"no stored session", "openai", MethodChatGPTPlan, nil, true},
		{"a plan session with no answer", "openai", MethodChatGPTPlan, plan(""), true},
		{"a plan session that granted", "openai", MethodChatGPTPlan, plan(consent + "=granted"), false},
		{"a plan session that declined", "openai", MethodChatGPTPlan, plan(consent + "=declined"), false},
		{"another application's session", "openai", MethodChatGPTPlan, other, true},
		{"openai's key sign-in", "openai", MethodAPIKey, nil, false},
		{"grok's device sign-in", "grok", MethodDevice, nil, false},
		{"a method grok does not offer", "grok", MethodChatGPTPlan, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &memLogins{}
			if tc.stored != nil {
				if err := store.Save(t.Context(), tc.id, tc.stored); err != nil {
					t.Fatal(err)
				}
			}
			err := CheckNoQuestion(t.Context(), tc.id, tc.method, store)
			if refused := err != nil; refused != tc.refused {
				t.Fatalf("CheckNoQuestion = %v, refused %t, want refused %t", err, refused, tc.refused)
			}
			if tc.refused {
				if _, ok := errors.AsType[*SignInError](err); !ok || !errors.Is(err, ErrAsks) || ErrorKind(err) != "unsupported" {
					t.Errorf("the refusal %v (kind %s), want a *SignInError holding ErrAsks, unsupported", err, ErrorKind(err))
				}
			}
		})
	}
	if err := CheckNoQuestion(t.Context(), "openai", MethodChatGPTPlan, failingLogins{&memLogins{}}); !errors.Is(err, errUnreadable) || errors.Is(err, ErrAsks) {
		t.Errorf("an unreadable store: %v, want its own error, not a refusal", err)
	}
}

// OfferedMethod finds the offered method a --method word names, as the SDK's
// facts give it, and nothing for a word the provider does not offer or gobble
// does not know.
func TestOfferedMethodByWord(t *testing.T) {
	for _, tc := range []struct {
		id, word, wantID string
		wantCompletion   authcontract.Completion
	}{
		{"grok", MethodDevice, "device_code", authcontract.CompletionDevice},
		{"grok", MethodBrowser, "browser_oauth", authcontract.CompletionBrowserPaste},
		{"openai", MethodAPIKey, "api_key", authcontract.CompletionValue},
		{"openai", MethodChatGPTPlan, "chatgpt_plan", authcontract.CompletionBrowserPaste},
		{"grok", MethodChatGPTPlan, "", ""},
		{"kilo", MethodBrowser, "", ""},
		{"grok", "nonsense", "", ""},
	} {
		m, ok := OfferedMethod(tc.id, tc.word)
		if ok != (tc.wantID != "") || m.ID != tc.wantID || m.Completion != tc.wantCompletion && ok {
			t.Errorf("OfferedMethod(%s, %s) = %q %q, %t; want %q %q", tc.id, tc.word, m.ID, m.Completion, ok, tc.wantID, tc.wantCompletion)
		}
	}
}

// serviceDown answers every request 503, marked final (x-should-retry false)
// when final is set.
type serviceDown struct{ final bool }

func (s serviceDown) RoundTrip(r *http.Request) (*http.Response, error) {
	h := http.Header{"Content-Type": {"application/json"}}
	if s.final {
		h.Set("X-Should-Retry", "false")
	}
	return &http.Response{StatusCode: http.StatusServiceUnavailable, Header: h, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"busy"}}`)), Request: r}, nil
}

// Retryable is the SDK's rule for a failed step: the API error's own
// Retryable when the error holds one, so a 503 the service marks final is not,
// though its kind, unavailable, is; and the kind table's otherwise, true for
// rate_limited, offline and unavailable only (go-llmprovider-sdk v1.31.0,
// signinflow.Failure). The errors are what production builds: List's, mapError's
// of the SDK's.
func TestRetryable(t *testing.T) {
	list := func(final bool) error {
		l, err := List(t.Context(), "openai", fmt.Sprintf("sk-retry-%t-%d", final, time.Now().UnixNano()), Options{HTTPClient: &http.Client{Transport: serviceDown{final}}})
		if err == nil {
			err = l.Err // a listing that fails past retry is reported in the Listing
		}
		return err
	}
	for _, tc := range []struct {
		name string
		err  error
		kind string
		want bool
	}{
		{"a 503", list(false), "unavailable", true},
		{"a 503 marked final", list(true), "unavailable", false},
		{"offline", mapError(fmt.Errorf("dial: %w", llmprovider.ErrOffline)), "offline", true},
		{"an auth failure", mapError(fmt.Errorf("refused: %w", llmprovider.ErrAuthFailure)), "auth", false},
		{"a cancelled sign-in", context.Canceled, "canceled", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Fatal("the listing succeeded, so it holds no error to read")
			}
			if got := ErrorKind(tc.err); got != tc.kind {
				t.Errorf("ErrorKind = %q, want %q", got, tc.kind)
			}
			if got := Retryable(tc.err); got != tc.want {
				t.Errorf("Retryable = %t, want %t (the kind %s)", got, tc.want, tc.kind)
			}
		})
	}
}
