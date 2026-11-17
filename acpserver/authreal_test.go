package acpserver

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// These tests run the real x/signin engine under the relay, through
// provider.StartSignIn, against issuers in process and a real auth.json:
// acpserver imports no SDK package (archtest rule 4) but llm/provider, which
// does. The issuer answers as xAI's; nothing leaves the machine.

// realEnv is a sign-in test over the real engine: auth.json in a temporary
// directory, and the issuers the engine's requests reach.
type realEnv struct {
	*siEnv
	file  string
	grok  *issuer
	httpc *http.Client
	// rawRuns are the engine runs the relay was given, in order.
	rawRuns []*provider.SignInRun
}

// startReal is siSpec.real's function over store and client.
func startReal(t *testing.T, store provider.LoginStore, client *http.Client) func(context.Context, string, string, bool) (*provider.SignInRun, error) {
	// The listing the sign-in checks its credential with is cached by the SDK
	// for ten minutes, per process: a cache of the test's own keeps another
	// test's listing from answering for the issuer.
	cache := &provider.Cache{}
	t.Cleanup(func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return func(ctx context.Context, id, method string, remote bool) (*provider.SignInRun, error) {
		o := provider.SignInOptions{ClientName: "gobble", ClientVersion: "1.2.3-test", HTTPClient: client, Cache: cache, Remote: remote}
		return provider.StartSignIn(ctx, id, method, o, store)
	}
}

func newRealEnv(t *testing.T, caps *acp.ElicitationCapabilities, spec siSpec) *realEnv {
	t.Helper()
	r := &realEnv{file: filepath.Join(t.TempDir(), "auth.json"), grok: newIssuer(t, grokOrigin)}
	r.httpc = &http.Client{Transport: byHost{"auth.x.ai": r.grok, "api.x.ai": r.grok}}
	spec.caps = caps
	start := startReal(t, auth.Logins{Path: r.file}, r.httpc)
	spec.real = func(ctx context.Context, id, method string, remote bool) (*provider.SignInRun, error) {
		run, err := start(ctx, id, method, remote)
		if err == nil {
			r.mu.Lock()
			r.rawRuns = append(r.rawRuns, run)
			r.mu.Unlock()
		}
		return run, err
	}
	r.siEnv = newSIEnv(t, spec)
	return r
}

// stored is auth.json's grok login, nil when none.
func (r *realEnv) stored() *provider.Login {
	r.t.Helper()
	l, err := auth.Logins{Path: r.file}.Load(r.t.Context(), "grok")
	if err != nil {
		r.t.Fatal(err)
	}
	return l
}

// fileBytes is auth.json as it is, nil for none.
func (r *realEnv) fileBytes() []byte {
	b, err := os.ReadFile(r.file)
	if err != nil {
		return nil
	}
	return b
}

// seed stores an old grok session, which a fresh sign-in must leave alone
// until its own saves.
func (r *realEnv) seed() {
	r.t.Helper()
	old := &provider.Login{Access: "OLD-ACCESS", Refresh: "OLD-REFRESH", Expires: time.Now().Add(time.Hour), Issuer: grokOrigin,
		ClientID: "b1a00492-073a-47ea-816f-4c329264a828", TokenURL: grokOrigin + "/oauth2/token", Kind: "grok"}
	if err := (auth.Logins{Path: r.file}).Save(r.t.Context(), "grok", old); err != nil {
		r.t.Fatal(err)
	}
}

// pasteScript is a form-only client's: it accepts the open_url form, giving
// the issuer its nonce from the authorize URL, then pastes what paste builds
// from the browser's redirect and state.
func pasteScript(t *testing.T, is *issuer, paste func(redirect, state string) string) func(int, acp.CreateElicitationRequest) elicitReply {
	var redirect, state string
	return func(n int, req acp.CreateElicitationRequest) elicitReply {
		if n == 0 {
			redirect, state = is.setNonce(t, metaStepOf(t, req).URL)
			return accept(nil)
		}
		return accept(map[string]acp.ElicitationContentValue{"callback": text(paste(redirect, state))})
	}
}

func goodPaste(redirect, state string) string {
	return redirect + "?code=c1&state=" + url.QueryEscape(state)
}

// The real device sign-in, through a form-only client: the SDK's show_code
// step as a form with the issuer's URL and code, and {} once the poll saved a
// session, which is auth.json's and which Signed is told of.
func TestAuthenticateRealGrokDevice(t *testing.T) {
	r := newRealEnv(t, formOnly, siSpec{})
	if err := r.authenticate(t.Context(), "grok", map[string]any{"method": "device_code"}); err != nil {
		t.Fatal(err)
	}
	reqs := r.client.requests()
	if len(reqs) != 1 || !strings.Contains(messageOf(reqs[0]), r.grok.URL+"/device") || !strings.Contains(messageOf(reqs[0]), "ABCD-1234") {
		t.Fatalf("elicitations %+v, want one form with the issuer's URL and code", reqs)
	}
	if l := r.stored(); l == nil || l.Access != "access-1" || l.Kind != "grok" {
		t.Errorf("auth.json holds %v, want the new grok session", l)
	}
	r.mu.Lock()
	signed := r.signed
	r.mu.Unlock()
	if len(signed) != 1 || signed[0] != "grok" {
		t.Errorf("Signed was called for %v, want grok once", signed)
	}
}

// grok's real browser sign-in for a remote client: the open_url step has
// accepts url_or_code, and the pasted callback completes it, with the sign-in
// started remote (the row of the amendment of 2026-11-16, at the pinned facts).
// Whether the engine then binds no port is the SDK's own test; gobble's part,
// that a remote sign-in sets no-loopback for every method, is held in
// llm/provider's TestStartSignInRemoteSetsNoLoopback (the owner's ruling of
// 2026-11-16, "Assert what gobble passes; use no real ports").
func TestAuthenticateRealGrokBrowserPasteRemote(t *testing.T) {
	r := newRealEnv(t, formOnly, siSpec{})
	r.client.script = pasteScript(t, r.grok, goodPaste)
	err := r.authenticate(t.Context(), "grok", map[string]any{"remote": true, "method": "browser_oauth"})
	if err != nil {
		t.Fatal(err)
	}
	reqs := r.client.requests()
	if len(reqs) != 2 {
		t.Fatalf("%d elicitations, want the open_url's and one paste form", len(reqs))
	}
	s := metaStepOf(t, reqs[0])
	if s.Kind != authcontract.StepOpenURL || s.Accepts != authcontract.PasteURLOrCode {
		t.Errorf("first step %+v, want open_url accepting url_or_code", s)
	}
	if c := r.callList(); len(c) != 1 || c[0].Method != "browser_oauth" || !c[0].Remote {
		t.Errorf("calls %v, want browser_oauth, remote", c)
	}
	if l := r.stored(); l == nil || l.Access != "access-1" {
		t.Errorf("auth.json holds %v, want the new session", l)
	}
}

// A pasted URL with this sign-in's state and error=access_denied ends it at
// once: no further paste form, auth_required of kind denied (0051-MADR D4
// step 3, rule 3).
func TestAuthenticatePastedDenialEnds(t *testing.T) {
	r := newRealEnv(t, formOnly, siSpec{})
	r.client.script = pasteScript(t, r.grok, func(redirect, state string) string {
		return redirect + "?error=access_denied&state=" + url.QueryEscape(state)
	})
	err := r.authenticate(t.Context(), "grok", map[string]any{"remote": true, "method": "browser_oauth"})
	wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureDenied))
	if n := len(r.client.requests()); n != 2 {
		t.Errorf("%d elicitations, want the open_url's and one paste form, no re-ask", n)
	}
	if l := r.stored(); l != nil {
		t.Errorf("a denied sign-in stored %v", l)
	}
}

// A paste the engine cannot use is asked for again, the paste never in the
// new message; the third ends it, failed, invalid_request (MADR D4 step 3,
// rule 2).
func TestAuthenticateRealBadPasteCap(t *testing.T) {
	const bad = "BADSTATE-SECRET-5"
	r := newRealEnv(t, formOnly, siSpec{})
	r.client.script = pasteScript(t, r.grok, func(redirect, _ string) string { return redirect + "?code=c1&state=" + bad })
	err := r.authenticate(t.Context(), "grok", map[string]any{"remote": true, "method": "browser_oauth"})
	wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureInvalidRequest))
	reqs := r.client.requests()
	if len(reqs) != 4 {
		t.Fatalf("%d elicitations, want the open_url's and three paste forms", len(reqs))
	}
	for i, q := range reqs[1:] {
		if strings.Contains(messageOf(q), bad) {
			t.Errorf("paste form %d's message holds the paste: %q", i+1, messageOf(q))
		}
	}
	if strings.Contains(r.log.String(), bad) {
		t.Errorf("the agent's log holds the paste:\n%s", r.log.String())
	}
	if strings.Contains(err.Error(), bad) {
		t.Errorf("the error holds the paste: %v", err)
	}
}

// A fresh sign-in that fails, is declined or is cancelled leaves auth.json as
// it was, byte for byte, and the issuer sees no revocation; one that
// completes replaces the session (0051-MADR D4 step 3).
func TestAuthenticateFreshKeepsOldOnFailure(t *testing.T) {
	fresh := map[string]any{"method": "device_code", "fresh": true}
	check := func(r *realEnv, before []byte, what string) {
		t.Helper()
		if got := r.fileBytes(); !bytes.Equal(got, before) {
			t.Errorf("%s: auth.json changed:\n%s\nwas\n%s", what, got, before)
		}
		if r.grok.seen("revoke") {
			t.Errorf("%s: the issuer saw a revocation", what)
		}
	}
	// The sign-in fails: the issuer denies it.
	r := newRealEnv(t, formOnly, siSpec{holds: func(string, authcontract.Method) bool { return true }, has: map[string]bool{"grok": true}})
	r.seed()
	before := r.fileBytes()
	r.grok.setDeny()
	err := r.authenticate(t.Context(), "grok", fresh)
	wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureDenied))
	check(r, before, "a denied fresh sign-in")

	// The client declines the show_code form.
	r = newRealEnv(t, formOnly, siSpec{holds: func(string, authcontract.Method) bool { return true }, has: map[string]bool{"grok": true},
		script: func(int, acp.CreateElicitationRequest) elicitReply { return declineReply() }})
	r.seed()
	before = r.fileBytes()
	err = r.authenticate(t.Context(), "grok", fresh)
	wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureCanceled))
	check(r, before, "a declined fresh sign-in")

	// The client cancels the request while the poll runs.
	r = newRealEnv(t, formOnly, siSpec{holds: func(string, authcontract.Method) bool { return true }, has: map[string]bool{"grok": true},
		script: func(int, acp.CreateElicitationRequest) elicitReply { return hangReply() }})
	r.seed()
	before = r.fileBytes()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- r.authenticate(ctx, "grok", fresh) }()
	r.waitRequests(1)
	cancel()
	if err := r.result(done); err == nil {
		t.Error("a cancelled authenticate answered success")
	}
	waitUntil(t, "the open form to be cancelled", func() bool { return r.client.endedCount() == 1 })
	check(r, before, "a cancelled fresh sign-in")

	// One that completes replaces it.
	r = newRealEnv(t, formOnly, siSpec{holds: func(string, authcontract.Method) bool { return true }, has: map[string]bool{"grok": true}})
	r.seed()
	if err := r.authenticate(t.Context(), "grok", fresh); err != nil {
		t.Fatal(err)
	}
	if l := r.stored(); l == nil || l.Access != "access-1" {
		t.Errorf("after a completed fresh sign-in auth.json holds %v, want the new session", l)
	}
}

// A listing the provider refuses fails the sign-in, kind auth; no network at
// all is offline, never other (0051-MADR D6).
func TestAuthenticateRealFailureKinds(t *testing.T) {
	r := newRealEnv(t, formOnly, siSpec{})
	r.grok.setRefuse()
	err := r.authenticate(t.Context(), "grok", map[string]any{"method": "device_code"})
	d := wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureAuth))
	if strings.Contains(d.Reason, "refused") && !strings.Contains(d.Reason, "sign-in") {
		t.Errorf("the reason %q holds the service's text", d.Reason)
	}

	o := newRealEnv(t, formOnly, siSpec{})
	o.grok.Close()
	err = o.authenticate(t.Context(), "grok", map[string]any{"method": "device_code"})
	d = wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureOffline))
	if d.Retryable == nil || !*d.Retryable {
		t.Errorf("an offline failure's retryable = %v, want true", d.Retryable)
	}
}

// stepsClosed reads ch until it closes and then yields its one false.
func stepsClosed(ch <-chan authcontract.Step) <-chan bool {
	out := make(chan bool, 1)
	go func() {
		for range ch {
		}
		out <- false
	}()
	return out
}

// callbackServers is how many goroutines serve a loopback sign-in's callback: the
// SDK starts one per listener in auth.serveCallbackListeners at go-llmprovider-sdk
// v1.31.0 (llmprovider/auth/oauth_loopback.go:530, its goroutine is the
// function's first closure, frame serveCallbackListeners.func1), and ends it
// when the sign-in ends. It reads the stacks, binds and dials nothing.
func callbackServers() int {
	buf := make([]byte, 1<<22)
	buf = buf[:runtime.Stack(buf, true)]
	return strings.Count(string(buf), "serveCallbackListeners.func1")
}

// A sign-in the client abandons leaves no listener: a local browser sign-in has
// the SDK serve its loopback callback from a goroutine, and cancelling the
// request ends it and the run, and no session is stored. No port is bound or
// dialled here: the test reads the stacks, as noRelayGoroutines does for the
// relay's own (the owner's ruling of 2026-11-16, "Assert what gobble passes; use
// no real ports"). The frame must be there while the sign-in runs, so that the
// test holds something, and gone, polled, after the abandon.
func TestAuthenticateAbandonedLeavesNothing(t *testing.T) {
	before := callbackServers()
	ready := make(chan struct{})
	var once sync.Once
	r := newRealEnv(t, formOnly, siSpec{script: func(int, acp.CreateElicitationRequest) elicitReply {
		once.Do(func() { close(ready) })
		return hangReply()
	}})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- r.authenticate(ctx, "grok", map[string]any{"method": "browser_oauth"}) }()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the open_url elicitation")
	}
	if n := callbackServers(); n <= before {
		t.Fatalf("%d callback servers while the sign-in runs, %d before it: the test holds nothing", n, before)
	}
	cancel()
	if err := r.result(done); err == nil {
		t.Error("a cancelled authenticate answered success")
	}
	r.mu.Lock()
	run := r.rawRuns[0]
	r.mu.Unlock()
	select {
	case <-stepsClosed(run.Steps()):
	case <-time.After(5 * time.Second):
		t.Error("the abandoned sign-in's run has not ended")
	}
	if _, err := run.Wait(t.Context()); err == nil || provider.ErrorKind(err) != string(authcontract.FailureCanceled) {
		t.Errorf("the abandoned sign-in's Wait = %v, want a canceled sign-in", err)
	}
	waitUntil(t, "the callback server's goroutine to end", func() bool { return callbackServers() <= before })
	waitUntil(t, "the open form to be cancelled", func() bool { return r.client.endedCount() == 1 })
	if l := r.stored(); l != nil {
		t.Errorf("an abandoned sign-in stored %v", l)
	}
}

// A sign-in's save and another writer's of the same auth.json leave a file
// that parses with both entries, deterministically: from the listing, which runs
// between the sign-in's token and its save, another writer takes auth.json's
// lock, reads the file and holds the lock until authenticate has answered, or
// a generous bound passes, and then writes from that earlier read. A save that
// skipped the lock completes while the lock is held, however late it is made,
// and the writer's write then loses the session every time; a save that takes
// the lock waits for the writer, so the bound only lengthens an honest run, to
// about the bound, and adds its entry to the file the writer left.
func TestAuthenticateSharesAuthJSON(t *testing.T) {
	r := newRealEnv(t, formOnly, siSpec{})
	held := make(chan struct{})
	answered := make(chan struct{})
	finished := make(chan error, 1)
	var once sync.Once
	r.grok.setOnList(func(*http.Request) {
		once.Do(func() {
			go func() {
				finished <- otherWriter(t.Context(), r.file, held, answered)
			}()
			<-held
		})
	})
	err := r.authenticate(t.Context(), "grok", map[string]any{"method": "device_code"})
	close(answered)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case werr := <-finished:
		if werr != nil {
			t.Fatal(werr)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the other writer never finished")
	}
	f, err := auth.Load(r.file)
	if err != nil {
		t.Fatalf("auth.json after both writers: %v", err)
	}
	if _, ok := f.Entry("grok"); !ok {
		t.Error("the sign-in's session is lost")
	}
	if _, ok := f.Entry("claude"); !ok {
		t.Error("the other writer's key is lost")
	}
}

// otherWriter is a writer of auth.json that is not gobble's sign-in: it takes
// the file's lock, reads the file, closes held, holds the lock until until is
// closed or three seconds pass, and writes the file from what it read with a key
// added, as a writer that read before another saved would.
func otherWriter(ctx context.Context, path string, held chan<- struct{}, until <-chan struct{}) error {
	release, err := filelock.Lock(ctx, path+".lock", filelock.Policy{First: 5 * time.Millisecond, Max: 50 * time.Millisecond, Deadline: 10 * time.Second})
	if err != nil {
		close(held)
		return err
	}
	read, rerr := os.ReadFile(path)
	if rerr != nil && !errors.Is(rerr, os.ErrNotExist) {
		close(held)
		return errors.Join(rerr, release())
	}
	close(held)
	select {
	case <-until:
	case <-time.After(3 * time.Second): // a locked save is waiting for us by now
	}
	f := map[string]jsontext.Value{}
	if len(read) > 0 {
		if err := json.Unmarshal(read, &f); err != nil {
			return errors.Join(err, release())
		}
	}
	f["claude"] = jsontext.Value(`{"type":"api_key","key":"sk-ant-test"}`)
	out, err := json.Marshal(f)
	if err != nil {
		return errors.Join(err, release())
	}
	return errors.Join(os.WriteFile(path, out, 0o600), release())
}

// The new credential is tried against the listing before anything is written
// (the owner's ruling of 2026-11-16, "Check first, save only if accepted"): a
// listing the provider refuses with 401 fails the sign-in with kind auth, not
// retryable, and leaves auth.json byte for byte as it was and the issuer sees
// no revocation, for a device sign-in and a browser sign-in by paste, with and
// without fresh, with a credential stored and with none. A listing that fails
// with a 503 does not fail the sign-in, as the wizard treats it
// (wizard/configure.go's stepRefusal): the session is saved.
func TestAuthenticateRefusedListingStoresNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		meta map[string]any
		by   func(*realEnv)
	}{
		{"device", map[string]any{"method": "device_code"}, func(*realEnv) {}},
		{"browser paste", map[string]any{"method": "browser_oauth", "remote": true}, func(r *realEnv) { r.client.script = pasteScript(r.t, r.grok, goodPaste) }},
	} {
		for _, fresh := range []bool{false, true} {
			for _, stored := range []bool{false, true} {
				name := tc.name + map[bool]string{true: " fresh", false: ""}[fresh] + map[bool]string{true: " stored", false: " none"}[stored]
				t.Run(name, func(t *testing.T) {
					meta := map[string]any{"fresh": fresh}
					maps.Copy(meta, tc.meta)
					r := newRealEnv(t, formOnly, siSpec{holds: func(string, authcontract.Method) bool { return false }, has: map[string]bool{"grok": stored}})
					tc.by(r)
					if stored {
						r.seed()
					}
					before := r.fileBytes()
					r.grok.setRefuse()
					err := r.authenticate(t.Context(), "grok", meta)
					d := wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureAuth))
					if d.Retryable != nil && *d.Retryable {
						t.Errorf("a refused listing's retryable = true, want false")
					}
					if got := r.fileBytes(); !bytes.Equal(got, before) {
						t.Errorf("auth.json changed:\n%s\nwas\n%s", got, before)
					}
					if r.grok.seen("revoke") {
						t.Error("the issuer saw a revocation")
					}
					r.mu.Lock()
					signed := r.signed
					r.mu.Unlock()
					if len(signed) != 0 {
						t.Errorf("Signed was called for %v after a refused sign-in", signed)
					}
				})
			}
		}
	}
	// Every other listing failure saves, as the wizard lets it (refusedCredential):
	// a refusal to permit (403), an outage (503), a bad request, a missing route
	// and a rate limit.
	for _, status := range []int{http.StatusForbidden, http.StatusServiceUnavailable, http.StatusBadRequest, http.StatusNotFound, http.StatusTooManyRequests} {
		t.Run(http.StatusText(status)+" listing saves", func(t *testing.T) {
			r := newRealEnv(t, formOnly, siSpec{})
			r.grok.setListStatus(status)
			if err := r.authenticate(t.Context(), "grok", map[string]any{"method": "device_code"}); err != nil {
				t.Fatalf("a %d listing failed the sign-in: %v", status, err)
			}
			if l := r.stored(); l == nil || l.Access != "access-1" {
				t.Errorf("auth.json holds %v, want the new session saved", l)
			}
		})
	}
}

// A sign-in whose request the client cancels while the listing is in flight
// stores nothing, with and without fresh, with a credential stored and with
// none: auth.json is byte for byte as it was, the issuer sees no revocation and
// the agent is not told of a new credential (D4 step 3). The listing's failure
// is no refusal, so what stops the save is the context.
func TestAuthenticateCancelledDuringListingStoresNothing(t *testing.T) {
	for _, fresh := range []bool{false, true} {
		for _, stored := range []bool{false, true} {
			name := map[bool]string{true: "fresh", false: "plain"}[fresh] + map[bool]string{true: " stored", false: " none"}[stored]
			t.Run(name, func(t *testing.T) {
				r := newRealEnv(t, formOnly, siSpec{holds: func(string, authcontract.Method) bool { return false }, has: map[string]bool{"grok": stored}})
				if stored {
					r.seed()
				}
				before := r.fileBytes()
				ctx, cancel := context.WithCancel(t.Context())
				defer cancel()
				listed := make(chan struct{}, 1)
				r.grok.setOnList(func(req *http.Request) {
					cancel() // the client's $/cancel_request, while the listing runs
					select {
					case listed <- struct{}{}:
					default:
					}
					select {
					case <-req.Context().Done():
					case <-time.After(5 * time.Second):
					}
				})
				err := r.authenticate(ctx, "grok", map[string]any{"method": "device_code", "fresh": fresh})
				if err == nil {
					t.Error("a cancelled authenticate answered success")
				}
				select {
				case <-listed:
				default:
					t.Fatal("the listing was never reached: the test holds nothing")
				}
				// The client has its answer at once; the agent's side ends later.
				waitUntil(t, "the sign-in to be forgotten", func() bool {
					r.agent.mu.Lock()
					defer r.agent.mu.Unlock()
					return len(r.agent.signing) == 0
				})
				if got := r.fileBytes(); !bytes.Equal(got, before) {
					t.Errorf("auth.json changed:\n%s\nwas\n%s", got, before)
				}
				if r.grok.seen("revoke") {
					t.Error("the issuer saw a revocation")
				}
				r.mu.Lock()
				signed := r.signed
				r.mu.Unlock()
				if len(signed) != 0 {
					t.Errorf("Signed was called for %v after a cancelled sign-in", signed)
				}
			})
		}
	}
}

// openaiOrigin is the origin OpenAI's issuer answers as to the SDK's plan sign-in
// (its default issuer), stated here as grokOrigin is.
const openaiOrigin = "https://auth.openai.com"

// A ChatGPT plan sign-in that the listing refuses (a 401), or that the client
// cancels during the listing, leaves auth.json byte for byte as it was, with a
// stored session in it, and the issuer sees no revocation; the issued client
// record in clients.json may exist, and holds no credential (the owner's ruling
// of 2026-11-16, "Accept it and say so in D4"). The real engine runs through
// provider.StartSignIn over the file-backed stores, as internal/cli builds them,
// and the test drives the steps as the relay would.
func TestStartSignInPlanFailureLeavesAuthJSON(t *testing.T) {
	for _, how := range []string{"refused", "cancelled"} {
		t.Run(how, func(t *testing.T) {
			dir := t.TempDir()
			file := filepath.Join(dir, "auth.json")
			logins := auth.Logins{Path: file}
			store := auth.ClientLogins{Logins: logins, Clients: auth.Clients{Path: filepath.Join(dir, "clients.json")}}
			old := &provider.Login{Access: "OLD-ACCESS", Refresh: "OLD-REFRESH", Expires: time.Now().Add(time.Hour), Issuer: openaiOrigin,
				ClientID: "app_test", TokenURL: openaiOrigin + "/oauth/token", Kind: "chatgpt_plan", Resource: "chatgpt"}
			if err := logins.Save(t.Context(), "openai", old); err != nil {
				t.Fatal(err)
			}
			before, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			is := newIssuer(t, openaiOrigin)
			is.scope = "openid chatgpt.tokens.use.direct"
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			listed := false
			if how == "refused" {
				is.setRefuse()
			} else {
				is.setOnList(func(req *http.Request) {
					listed = true
					cancel()
					select {
					case <-req.Context().Done():
					case <-time.After(5 * time.Second):
					}
				})
			}
			cache := &provider.Cache{}
			t.Cleanup(func() {
				if err := cache.Close(context.Background()); err != nil {
					t.Error(err)
				}
			})
			o := provider.SignInOptions{ClientName: "gobble", ClientVersion: "1.2.3-test", HTTPClient: &http.Client{Transport: toOne{is}}, Cache: cache,
				Remote: true, HostID: "urn:uuid:11111111-2222-4333-8444-555555555555"}
			run, err := provider.StartSignIn(ctx, "openai", "chatgpt_plan", o, store)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(run.Cancel)
			drivePlan(ctx, t, is, run)
			_, err = run.Wait(ctx)
			t.Logf("Wait: %v (%v)", err, errors.Unwrap(err))
			if err == nil {
				t.Fatalf("a %s plan sign-in succeeded", how)
			}
			if how == "cancelled" && !listed {
				t.Fatal("the listing was never reached: the test holds nothing")
			}
			if how == "refused" && !is.seen("GET /v1/") {
				t.Fatal("the plan's listing was not requested: the test holds nothing")
			}
			after, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(after, before) {
				t.Errorf("auth.json changed:\n%s\nwas\n%s", after, before)
			}
			if is.seen("revoke") {
				t.Error("the issuer saw a revocation")
			}
			if b, err := os.ReadFile(filepath.Join(dir, "clients.json")); err == nil {
				t.Logf("clients.json exists after a %s sign-in (%d bytes): it holds the issued client, no credential", how, len(b))
			}
		})
	}
}

// drivePlan reads a plan sign-in's steps to the end, answering as a person who
// allows the consent and pastes the browser's callback, naming the stored
// session's client, would.
func drivePlan(ctx context.Context, t *testing.T, is *issuer, run *provider.SignInRun) {
	t.Helper()
	var redirect, state string
	for s := range run.Steps() {
		var a authcontract.Answer
		switch s.Kind {
		case authcontract.StepConfirm:
			allow := true
			a = authcontract.Answer{SchemaVersion: authcontract.SchemaVersion, Seq: s.Seq, Confirm: &allow}
		case authcontract.StepOpenURL:
			redirect, state = is.setNonce(t, s.URL)
			continue
		case authcontract.StepPaste:
			cb := redirect + "?code=c1&state=" + url.QueryEscape(state) + "&client_id=app_test" // the stored session's client, signed in to again
			a = authcontract.Answer{SchemaVersion: authcontract.SchemaVersion, Seq: s.Seq, Paste: cb}
		default:
			continue
		}
		if err := run.Answer(ctx, a); err != nil {
			t.Fatalf("answering step %d: %v", s.Seq, err)
		}
	}
}
