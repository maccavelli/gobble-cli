package acpserver

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
	sdkacptest "github.com/maccavelli/acp-go-sdk2/acptest"
	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// What follows is the harness of the sign-in tests: an agent over two sign-in
// providers, a client that elicits from a script and records what it is asked,
// and a scripted run, which stands in for the SDK's engine (the relay is what
// these tests hold; the real x/signin runs in llm/provider and in
// internal/cli, because acpserver may not import it, archtest rule 4). The
// scripted run's steps have the shape signinflow gives them
// (go-llmprovider-sdk internal/signinflow/signinflow.go at v1.31.0): open_url
// with URL and, for a browser sign-in with a paste, Accepts; show_code with
// URL, Code and ExpiresAt, then waiting; paste with Accepts, Secret and Text;
// confirm with ConfirmID, Text, a false Default and Links.

var (
	formOnly = &acp.ElicitationCapabilities{Form: &acp.ElicitationFormCapabilities{}}
	withURL  = &acp.ElicitationCapabilities{Form: &acp.ElicitationFormCapabilities{}, Url: &acp.ElicitationUrlCapabilities{}}
)

const (
	goodCallback = "http://127.0.0.1:56121/callback?code=SECRET-CODE-7&state=st-1"
	authorizeURL = "https://auth.example/oauth2/authorize?client_id=c&state=st-1&a=1"
	deviceURL    = "https://auth.example/device"
)

// elicitReply is a client's scripted reply to one elicitation; hang leaves it
// open until the agent ends the request.
type elicitReply struct {
	resp acp.CreateElicitationResponse
	hang bool
}

func accept(content map[string]acp.ElicitationContentValue) elicitReply {
	r := acp.NewCreateElicitationResponseAccept()
	r.Accept.Content = content
	return elicitReply{resp: r}
}

func declineReply() elicitReply { return elicitReply{resp: acp.NewCreateElicitationResponseDecline()} }

func hangReply() elicitReply { return elicitReply{hang: true} }

func text(s string) acp.ElicitationContentValue { return acp.NewElicitationContentValueString(s) }

// elicitClient answers elicitation/create from script (the n-th request, from
// 0) and records the requests and completions; a request left open that the
// agent then ends is counted in ended.
type elicitClient struct {
	*acptest.Client
	script func(n int, req acp.CreateElicitationRequest) elicitReply

	mu        sync.Mutex
	reqs      []acp.CreateElicitationRequest
	completes []acp.ElicitationId
	ended     int
}

func (c *elicitClient) CreateElicitation(ctx context.Context, req acp.CreateElicitationRequest) (acp.CreateElicitationResponse, error) {
	c.mu.Lock()
	n := len(c.reqs)
	c.reqs = append(c.reqs, req)
	c.mu.Unlock()
	r := c.script(n, req)
	if r.hang {
		<-ctx.Done()
		c.mu.Lock()
		c.ended++
		c.mu.Unlock()
		return acp.CreateElicitationResponse{}, ctx.Err()
	}
	return r.resp, nil
}

func (c *elicitClient) CompleteElicitation(_ context.Context, n acp.CompleteElicitationNotification) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.completes = append(c.completes, n.ElicitationId)
	return nil
}

func (c *elicitClient) requests() []acp.CreateElicitationRequest {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.reqs)
}

func (c *elicitClient) completed() []acp.ElicitationId {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.completes)
}

func (c *elicitClient) endedCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ended
}

// waitFor polls cond for up to five seconds.
func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	for end := time.Now().Add(5 * time.Second); time.Now().Before(end); time.Sleep(2 * time.Millisecond) {
		if cond() {
			return
		}
	}
	t.Fatalf("timed out waiting for %s", what)
}

// messageOf, schemaOf and metaStepOf read what an elicitation asked.
func messageOf(r acp.CreateElicitationRequest) string {
	switch {
	case r.Form != nil:
		return r.Form.Message
	case r.Url != nil:
		return r.Url.Message
	}
	return ""
}

func scopeOf(r acp.CreateElicitationRequest) (*acp.ElicitationRequestScope, *acp.ElicitationSessionScope) {
	switch {
	case r.Form != nil:
		return r.Form.Request, r.Form.Session
	case r.Url != nil:
		return r.Url.Request, r.Url.Session
	}
	return nil, nil
}

// metaStepOf is the contract step in _meta.gobble.step of an elicitation,
// decoded strictly.
func metaStepOf(t *testing.T, r acp.CreateElicitationRequest) authcontract.Step {
	t.Helper()
	meta := map[string]any(nil)
	switch {
	case r.Form != nil:
		meta = r.Form.Meta
	case r.Url != nil:
		meta = r.Url.Meta
	}
	raw, ok := meta[gobble]
	if !ok {
		t.Fatalf("no _meta.gobble on %+v", r)
	}
	b, err := json.Marshal(raw)
	if err != nil {
		t.Fatal(err)
	}
	var m struct {
		Step authcontract.Step `json:"step"`
	}
	if err := json.Unmarshal(b, &m, json.RejectUnknownMembers(true)); err != nil {
		t.Fatalf("_meta.gobble.step: %v", err)
	}
	return m.Step
}

// scriptRun is a scripted sign-in run: the engine's stand-in.
type scriptRun struct {
	id     string
	method authcontract.Method

	steps chan authcontract.Step
	fin   chan struct{}

	mu       sync.Mutex
	seq      int
	asking   map[int]bool // asked and not yet answered
	answers  []authcontract.Answer
	refused  []authcontract.Answer // answers the engine refuses: not asked, or already answered
	canceled bool
	ended    bool
	err      error
	// key is the key a finished run returns, as a device sign-in that yields one
	// does.
	key string
	// onAnswer, when set, runs after an accepted answer, to emit more steps
	// or end the run.
	onAnswer func(r *scriptRun, a authcontract.Answer)
}

func newScriptRun(id string, m authcontract.Method) *scriptRun {
	return &scriptRun{id: id, method: m, steps: make(chan authcontract.Step, 32), fin: make(chan struct{}), asking: map[int]bool{}}
}

// emit sends step s, numbered as the engine numbers them.
func (r *scriptRun) emit(s authcontract.Step) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.emitLocked(s)
}

func (r *scriptRun) emitLocked(s authcontract.Step) {
	if r.ended {
		return
	}
	r.seq++
	s.SchemaVersion, s.Seq = authcontract.SchemaVersion, r.seq
	if asks(s.Kind) {
		r.asking[s.Seq] = true
	}
	r.steps <- s
}

// finish ends the run: err nil is done, else failed, as signinflow ends one.
func (r *scriptRun) finish(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.finishLocked(err)
}

func (r *scriptRun) finishLocked(err error) {
	if r.ended {
		return
	}
	r.ended, r.err = true, err
	r.seq++
	last := authcontract.Step{SchemaVersion: authcontract.SchemaVersion, Seq: r.seq, Kind: authcontract.StepDone, Yields: r.method.Yields}
	if err != nil {
		last = authcontract.Step{SchemaVersion: authcontract.SchemaVersion, Seq: r.seq, Kind: authcontract.StepFailed,
			Error: &authcontract.Failure{Kind: authcontract.ErrorKind(provider.ErrorKind(err)), Message: err.Error()}}
	}
	r.steps <- last
	close(r.steps)
	close(r.fin)
}

func (r *scriptRun) Method() authcontract.Method     { return r.method }
func (r *scriptRun) Steps() <-chan authcontract.Step { return r.steps }

func (r *scriptRun) Answer(_ context.Context, a authcontract.Answer) error {
	r.mu.Lock()
	if r.ended || !r.asking[a.Seq] {
		r.refused = append(r.refused, a)
		r.mu.Unlock()
		return errors.New("answer refused: invalid request")
	}
	delete(r.asking, a.Seq)
	r.answers = append(r.answers, a)
	on := r.onAnswer
	r.mu.Unlock()
	if on != nil {
		on(r, a)
	}
	return nil
}

func (r *scriptRun) Wait(ctx context.Context) (provider.SignInResult, error) {
	select {
	case <-r.fin:
		r.mu.Lock()
		defer r.mu.Unlock()
		if r.err != nil {
			return provider.SignInResult{}, r.err
		}
		return provider.SignInResult{Key: r.key, Yields: r.method.Yields}, nil
	case <-ctx.Done():
		return provider.SignInResult{}, provider.NewSignInError(r.id, ctx.Err())
	}
}

func (r *scriptRun) Cancel() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.ended {
		return
	}
	r.canceled = true
	r.finishLocked(provider.NewSignInError(r.id, context.Canceled))
}

func (r *scriptRun) answered() []authcontract.Answer {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.answers)
}

func (r *scriptRun) refusedAnswers() []authcontract.Answer {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.refused)
}

func (r *scriptRun) wasCanceled() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.canceled
}

// offeredMethod is provider id's offered method with contract id mid, from the
// real policy.
func offeredMethod(t *testing.T, id, mid string) authcontract.Method {
	t.Helper()
	for _, m := range provider.Offered(id) {
		if m.ID == mid {
			return m
		}
	}
	t.Fatalf("%s offers no %s", id, mid)
	return authcontract.Method{}
}

// deviceRun is a device sign-in's run: show_code, then waiting, as signinflow
// emits them; the test ends it with finish.
func deviceRun(id string, m authcontract.Method) *scriptRun {
	r := newScriptRun(id, m)
	r.emit(authcontract.Step{Kind: authcontract.StepShowCode, URL: deviceURL, Code: "ABCD-1234", ExpiresAt: time.Date(2026, 11, 16, 10, 30, 0, 0, time.UTC)})
	r.emit(authcontract.Step{Kind: authcontract.StepWaiting})
	return r
}

// pasteRun is a browser sign-in that takes a paste: open_url with accepts,
// then paste (secret). A callback equal to goodCallback ends it; anything else
// is a bad paste that brings a new paste step.
func pasteRun(id string, m authcontract.Method) *scriptRun {
	r := newScriptRun(id, m)
	paste := func(r *scriptRun) {
		r.emitLocked(authcontract.Step{Kind: authcontract.StepPaste, Accepts: m.Paste, Secret: true,
			Text: "Paste the URL the browser ends on, or its code"})
	}
	r.mu.Lock()
	r.emitLocked(authcontract.Step{Kind: authcontract.StepOpenURL, URL: authorizeURL, Accepts: m.Paste})
	paste(r)
	r.mu.Unlock()
	r.onAnswer = func(r *scriptRun, a authcontract.Answer) {
		if a.Paste == goodCallback {
			r.finish(nil)
			return
		}
		r.mu.Lock()
		paste(r)
		r.mu.Unlock()
	}
	return r
}

// loopbackRun is a browser sign-in with no paste: open_url with no accepts,
// which only the loopback completes (the test's finish).
func loopbackRun(id string, m authcontract.Method) *scriptRun {
	r := newScriptRun(id, m)
	r.emit(authcontract.Step{Kind: authcontract.StepOpenURL, URL: authorizeURL})
	return r
}

// confirmRun asks a consent, as a plan sign-in does before it opens a URL.
func confirmRun(id string, m authcontract.Method) *scriptRun {
	r := newScriptRun(id, m)
	no := false
	r.emit(authcontract.Step{Kind: authcontract.StepConfirm, ConfirmID: "chatgpt_plan.background_use",
		Text: "Allow background use?", Default: &no, Links: []authcontract.Link{{Rel: authcontract.LinkUsage, URL: "https://example.com/usage"}}})
	r.onAnswer = func(r *scriptRun, _ authcontract.Answer) { r.finish(nil) }
	return r
}

// standIn is the contract's own loopback-only shape (browser_loopback, no
// Paste, no paste feature), made from Grok's real browser_oauth: no built-in
// method at go-llmprovider-sdk v1.31.0 has it (0051-PLAN P6, the amendment of
// 2026-11-16).
func standIn(t *testing.T) authcontract.Method {
	m := offeredMethod(t, "grok", "browser_oauth")
	m.Completion = authcontract.CompletionBrowserLoopback
	m.Paste = authcontract.PasteNone
	m.Features = nil
	return m
}

// signInCall is one call of AuthOptions.SignIn.
type signInCall struct {
	ID, Method string
	Remote     bool
}

// siSpec configures a sign-in test's agent.
type siSpec struct {
	caps    *acp.ElicitationCapabilities
	has     map[string]bool
	holds   func(id string, m authcontract.Method) bool
	methods func(id string) []authcontract.Method
	// run makes the run for a call; nil is an error from SignIn.
	run    func(call signInCall) *scriptRun
	script func(n int, req acp.CreateElicitationRequest) elicitReply
	opts   func(*Options)
	// real, when set, starts the sign-in in place of run: the real engine,
	// through provider.StartSignIn.
	real func(ctx context.Context, id, method string, remote bool) (*provider.SignInRun, error)
}

// siEnv is a connected sign-in test.
type siEnv struct {
	t      *testing.T
	conn   *acptest.Conn
	client *elicitClient
	agent  *Agent
	log    *syncBuffer

	mu     sync.Mutex
	calls  []signInCall
	runs   []*scriptRun
	signed []string
}

var signInProviders = func() []acpext.AuthProvider {
	return []acpext.AuthProvider{
		{ID: "grok", Name: "Grok (xAI)", Command: "gobble auth login grok", Login: true},
		{ID: "kilo", Name: "Kilo Gateway", Command: "gobble auth login kilo", Login: true},
		{ID: "keyed", Name: "Keyed", Command: "gobble auth login keyed --api-key -"},
	}
}

func newSIEnv(t *testing.T, s siSpec) *siEnv {
	t.Helper()
	e := &siEnv{t: t, log: &syncBuffer{}}
	e.client = &elicitClient{Client: &acptest.Client{}, script: s.script}
	if e.client.script == nil {
		e.client.script = func(int, acp.CreateElicitationRequest) elicitReply { return accept(nil) }
	}
	methods := s.methods
	if methods == nil {
		methods = provider.Offered
	}
	auth := AuthOptions{
		Providers: signInProviders,
		Has:       func(id string) bool { return s.has[id] },
		Methods:   methods,
		Holds:     s.holds,
		Remote:    provider.Remote,
		Command:   func(id string, m authcontract.Method) string { return provider.MethodCommand(id, m.ID) },
		SignIn: func(ctx context.Context, id, method string, remote bool) (SignInRun, error) {
			call := signInCall{id, method, remote}
			e.mu.Lock()
			e.calls = append(e.calls, call)
			e.mu.Unlock()
			if s.real != nil {
				r, err := s.real(ctx, id, method, remote)
				if err != nil {
					return nil, err
				}
				return r, nil
			}
			if s.run == nil {
				return nil, errors.New("no run scripted")
			}
			r := s.run(call)
			if r == nil {
				return nil, provider.NewSignInError(id, errors.New("the engine would not start"))
			}
			e.mu.Lock()
			e.runs = append(e.runs, r)
			e.mu.Unlock()
			return r, nil
		},
		Signed: func(id string) {
			e.mu.Lock()
			e.signed = append(e.signed, id)
			e.mu.Unlock()
		},
	}
	o := Options{Version: "1.2.3", Auth: auth, Logger: slog.New(slog.NewTextHandler(e.log, &slog.HandlerOptions{Level: slog.LevelDebug})),
		Provider: func(string, llm.Retry) (llm.Provider, error) { return llmtest.NewScript(llmtest.Text("ok")), nil },
		Prompt:   PromptOptions{GOOS: "linux", cwd: "/work"}}
	if s.opts != nil {
		s.opts(&o)
	}
	e.agent = New(o)
	e.conn = acptest.Connect(t, e.agent, e.client)
	e.agent.SetConnection(e.conn.Agent)
	caps := acp.ClientCapabilities{Elicitation: s.caps}
	if _, err := e.conn.Client.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber,
		ClientInfo: &acp.Implementation{Name: "test", Version: "0.0.0"}, ClientCapabilities: caps}); err != nil {
		t.Fatal(err)
	}
	return e
}

// authMetaOf is _meta for an authenticate request: gobble's member.
func authMetaOf(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	return map[string]any{gobble: m}
}

// authenticate sends authenticate for id with _meta.gobble m.
func (e *siEnv) authenticate(ctx context.Context, id string, m map[string]any) error {
	// A request nothing answers fails the test at once instead of holding the
	// run to the package's timeout.
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	_, err := e.conn.Client.Authenticate(ctx, acp.AuthenticateRequest{MethodId: acp.AuthMethodId(id), Meta: authMetaOf(m)})
	if errors.Is(err, context.DeadlineExceeded) {
		e.t.Errorf("authenticate %s %v: no answer within 20s", id, m)
	}
	return err
}

// start sends authenticate in the background; the error comes on the channel.
func (e *siEnv) start(id string, m map[string]any) <-chan error {
	ch := make(chan error, 1)
	go func() { ch <- e.authenticate(e.t.Context(), id, m) }()
	return ch
}

// result waits for a started authenticate's answer.
func (e *siEnv) result(ch <-chan error) error {
	e.t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		e.t.Fatal("authenticate did not answer within 5 s")
		return nil
	}
}

func (e *siEnv) run(i int) *scriptRun {
	e.t.Helper()
	waitUntil(e.t, fmt.Sprintf("run %d", i), func() bool {
		e.mu.Lock()
		defer e.mu.Unlock()
		return len(e.runs) > i
	})
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.runs[i]
}

func (e *siEnv) callList() []signInCall {
	e.mu.Lock()
	defer e.mu.Unlock()
	return slices.Clone(e.calls)
}

func (e *siEnv) waitRequests(n int) []acp.CreateElicitationRequest {
	e.t.Helper()
	waitUntil(e.t, fmt.Sprintf("%d elicitations", n), func() bool { return len(e.client.requests()) >= n })
	return e.client.requests()
}

// authData is the data of an auth_required (or other) error.
func authData(t *testing.T, err error) (code int, d acpext.ErrorData) {
	t.Helper()
	re, ok := errors.AsType[*acp.RequestError](err)
	if !ok {
		t.Fatalf("err = %v, want a JSON-RPC error", err)
	}
	b, merr := json.Marshal(re.Data)
	if merr != nil {
		t.Fatal(merr)
	}
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatalf("error data %s: %v", b, err)
	}
	return re.Code, d
}

// wantAuthRequired asserts err is auth_required of kind kind for grok.
func wantAuthRequired(t *testing.T, err error, id string, kind acpext.ErrorKind) acpext.ErrorData {
	t.Helper()
	code, d := authData(t, err)
	if code != -32000 || d.Provider != id || d.Kind != kind {
		t.Fatalf("error = code %d %+v, want auth_required for %s of kind %q", code, d, id, kind)
	}
	return d
}

// noRelayGoroutines fails when a goroutine that sends a relay's elicitation is
// still running after five seconds: every one has ended when authenticate has
// answered, and the check polls, as a goroutine may take a moment to leave the
// stack after the client has the answer.
func noRelayGoroutines(t *testing.T) {
	t.Helper()
	var buf []byte
	i := -1
	for end := time.Now().Add(5 * time.Second); ; time.Sleep(2 * time.Millisecond) {
		buf = make([]byte, 1<<20)
		buf = buf[:runtime.Stack(buf, true)]
		if i = bytes.Index(buf, []byte("(*relay).start.func")); i < 0 {
			return
		}
		if time.Now().After(end) {
			break
		}
	}
	t.Errorf("a relay's elicitation goroutine is still running when the client has authenticate's answer:\n%s", buf[max(0, i-400):min(len(buf), i+200)])
}

// requestIDOf is the id of the first request of method in the client's
// recorded frames.
func requestIDOf(t *testing.T, e *siEnv, method string) jsontext.Value {
	t.Helper()
	for _, r := range e.conn.Recorder.Records() {
		if r.Direction != sdkacptest.Sent {
			continue
		}
		var f struct {
			ID     jsontext.Value `json:"id"`
			Method string         `json:"method"`
		}
		if json.Unmarshal(r.Frame, &f) == nil && f.Method == method {
			return f.ID
		}
	}
	t.Fatalf("no %s request recorded", method)
	return nil
}

// A shared case: grok's device sign-in over a form-only client that accepts
// every form, ended by the test.
func deviceSpec(t *testing.T, caps *acp.ElicitationCapabilities, script func(int, acp.CreateElicitationRequest) elicitReply) siSpec {
	return siSpec{caps: caps, script: script, run: func(c signInCall) *scriptRun {
		return deviceRun(c.ID, offeredMethod(t, c.ID, c.Method))
	}}
}

// Every elicitation is request-scoped, its requestId the authenticate
// request's own, never a session's (0051-MADR D4; U8).
func TestAuthenticateRequestScope(t *testing.T) {
	e := newSIEnv(t, deviceSpec(t, formOnly, func(int, acp.CreateElicitationRequest) elicitReply { return accept(nil) }))
	done := e.start("grok", map[string]any{"method": "device_code"})
	reqs := e.waitRequests(1)
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
	want := requestIDOf(t, e, "authenticate")
	for i, r := range reqs {
		scope, sess := scopeOf(r)
		if scope == nil || sess != nil {
			t.Fatalf("elicitation %d: scope %+v, session %+v, want request-scoped", i, scope, sess)
		}
		got, err := json.Marshal(scope.RequestId)
		if err != nil || string(got) != string(want) {
			t.Errorf("elicitation %d: requestId %s (%v), want the authenticate request's %s", i, got, err, want)
		}
	}
}

// A client with elicitation.url gets a device code as URL mode to the
// verification URL, its code and expiry in the message; elicitation/complete
// follows with that id when the poll ends; {} (0051-MADR D4's table).
func TestAuthenticateDeviceCodeURL(t *testing.T) {
	e := newSIEnv(t, deviceSpec(t, withURL, func(int, acp.CreateElicitationRequest) elicitReply { return accept(nil) }))
	done := e.start("grok", map[string]any{"method": "device_code"})
	reqs := e.waitRequests(1)
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
	u := reqs[0].Url
	if u == nil || u.Url != deviceURL || !strings.Contains(u.Message, "ABCD-1234") || !strings.Contains(u.Message, "10:30") {
		t.Fatalf("first elicitation %+v, want URL mode to %s with the code and expiry in its message", reqs[0], deviceURL)
	}
	if s := metaStepOf(t, reqs[0]); s.Kind != authcontract.StepShowCode || s.Code != "ABCD-1234" || s.Secret {
		t.Errorf("_meta.gobble.step = %+v, want the show_code step", s)
	}
	waitUntil(t, "elicitation/complete", func() bool { return len(e.client.completed()) > 0 })
	if got := e.client.completed(); len(got) != 1 || got[0] != u.ElicitationId {
		t.Errorf("elicitation/complete %v, want one with %q", got, u.ElicitationId)
	}
}

// A form-only client that leaves the form open: a form with no fields, URL
// and code in its message; the request ends, with $/cancel_request, when the
// poll ends (0051-MADR D4).
func TestAuthenticateDeviceCodeForm(t *testing.T) {
	e := newSIEnv(t, deviceSpec(t, formOnly, func(int, acp.CreateElicitationRequest) elicitReply { return hangReply() }))
	done := e.start("grok", map[string]any{"method": "device_code"})
	reqs := e.waitRequests(1)
	f := reqs[0].Form
	if f == nil || len(f.RequestedSchema.Properties) != 0 || !strings.Contains(f.Message, deviceURL) || !strings.Contains(f.Message, "ABCD-1234") {
		t.Fatalf("first elicitation %+v, want a form with no fields, URL and code in its message", reqs[0])
	}
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
	// The relay ends the open form and waits for its goroutine before it
	// answers authenticate: the SDK also cancels a child request when its handler
	// returns, but only after the response, so what tells the two apart is that
	// no goroutine of the relay's is left when the client has the response.
	noRelayGoroutines(t)
	waitUntil(t, "the open form to be cancelled", func() bool { return e.client.endedCount() == 1 })
	if len(e.client.completed()) != 0 {
		t.Errorf("elicitation/complete sent for a form: %v", e.client.completed())
	}
}

// grok's browser sign-in by paste for a form-only client: a form with no
// fields, the URL in its message; on its empty accept, a second form with one
// required string callback, whose answer is the paste step's, with its seq and
// Secret true, and completes the sign-in; the open_url's seq is never
// answered.
func TestAuthenticateBrowserPasteForm(t *testing.T) {
	paste := goodCallback
	e := newSIEnv(t, siSpec{caps: formOnly,
		script: func(n int, _ acp.CreateElicitationRequest) elicitReply {
			if n == 0 {
				return accept(nil)
			}
			return accept(map[string]acp.ElicitationContentValue{"callback": text(paste)})
		},
		run: func(c signInCall) *scriptRun { return pasteRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
	if err := e.authenticate(t.Context(), "grok", map[string]any{"method": "browser_oauth"}); err != nil {
		t.Fatal(err)
	}
	reqs := e.client.requests()
	if len(reqs) != 2 {
		t.Fatalf("%d elicitations, want the open_url's form then the paste form", len(reqs))
	}
	if f := reqs[0].Form; f == nil || len(f.RequestedSchema.Properties) != 0 || !strings.Contains(f.Message, authorizeURL) {
		t.Errorf("first elicitation %+v, want a form with no fields and the URL in its message", reqs[0])
	}
	if s := metaStepOf(t, reqs[0]); s.Kind != authcontract.StepOpenURL || s.Accepts != authcontract.PasteURLOrCode {
		t.Errorf("first step %+v, want open_url accepting url_or_code", s)
	}
	f := reqs[1].Form
	if f == nil || len(f.RequestedSchema.Properties) != 1 || !slices.Equal(f.RequestedSchema.Required, []string{"callback"}) || f.RequestedSchema.Properties["callback"].Kind() != "string" {
		t.Fatalf("second elicitation %+v, want a form of one required string callback", reqs[1])
	}
	if s := metaStepOf(t, reqs[1]); s.Kind != authcontract.StepPaste || !s.Secret {
		t.Errorf("second step %+v, want a secret paste step", s)
	}
	run := e.run(0)
	ans := run.answered()
	if len(ans) != 1 || ans[0].Seq != 2 || ans[0].Paste != paste || ans[0].Secret == nil || !*ans[0].Secret {
		t.Fatalf("answers %v, want one for the paste step's seq 2, secret", ans)
	}
	if r := run.refusedAnswers(); len(r) != 0 {
		t.Errorf("the engine refused answers %v: an open_url's seq was answered", r)
	}
}

// A form-only client's accept of a show_code form reaches no engine and the
// poll runs on; its decline cancels the sign-in with kind canceled and the
// poll stops (0051-MADR D4, "Shown steps are acknowledged, not answered").
func TestAuthenticateShownStepAcknowledged(t *testing.T) {
	e := newSIEnv(t, deviceSpec(t, formOnly, func(int, acp.CreateElicitationRequest) elicitReply { return accept(nil) }))
	done := e.start("grok", map[string]any{"method": "device_code"})
	e.waitRequests(1)
	run := e.run(0)
	// The accept has been answered; the sign-in is still running.
	select {
	case err := <-done:
		t.Fatalf("authenticate answered %v after the accept, want the poll to run on", err)
	case <-time.After(100 * time.Millisecond):
	}
	if a := run.answered(); len(a) != 0 || run.wasCanceled() {
		t.Fatalf("after the accept: answers %v, cancelled %v, want neither", a, run.wasCanceled())
	}
	run.finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}

	d := newSIEnv(t, deviceSpec(t, formOnly, func(int, acp.CreateElicitationRequest) elicitReply { return declineReply() }))
	err := d.authenticate(t.Context(), "grok", map[string]any{"method": "device_code"})
	wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureCanceled))
	if !d.run(0).wasCanceled() {
		t.Error("the sign-in was not cancelled by the decline: the poll runs on")
	}
}

// With elicitation.url: URL mode for the open_url, then the paste form.
func TestAuthenticateBrowserPasteURL(t *testing.T) {
	e := newSIEnv(t, siSpec{caps: withURL,
		script: func(n int, _ acp.CreateElicitationRequest) elicitReply {
			if n == 0 {
				return accept(nil)
			}
			return accept(map[string]acp.ElicitationContentValue{"callback": text(goodCallback)})
		},
		run: func(c signInCall) *scriptRun { return pasteRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
	if err := e.authenticate(t.Context(), "grok", map[string]any{"method": "browser_oauth"}); err != nil {
		t.Fatal(err)
	}
	reqs := e.client.requests()
	if len(reqs) != 2 || reqs[0].Url == nil || reqs[0].Url.Url != authorizeURL || reqs[1].Form == nil {
		t.Fatalf("elicitations %+v, want URL mode to the authorize URL, then the paste form", reqs)
	}
	if got := e.client.completed(); len(got) != 1 || got[0] != reqs[0].Url.ElicitationId {
		t.Errorf("elicitation/complete %v, want one for the URL elicitation", got)
	}
}

// A paste the engine refuses brings a new paste step: asked again, the paste
// never in the new message, two re-asks and three forms in all; a form of the
// third bad one ends it (the engine's kind, invalid_request, is held with the
// real engine in internal/cli).
func TestAuthenticateBadPasteAsksAgain(t *testing.T) {
	const bad = "BAD-PASTE-VALUE-9"
	e := newSIEnv(t, siSpec{caps: formOnly,
		script: func(n int, _ acp.CreateElicitationRequest) elicitReply {
			if n == 0 {
				return accept(nil)
			}
			return accept(map[string]acp.ElicitationContentValue{"callback": text(bad)})
		},
		run: func(c signInCall) *scriptRun {
			r := pasteRun(c.ID, offeredMethod(t, c.ID, c.Method))
			first := r.onAnswer
			r.onAnswer = func(r *scriptRun, a authcontract.Answer) {
				if len(r.answered()) >= 3 { // the third bad paste ends it
					r.finish(provider.NewSignInError(c.ID, context.DeadlineExceeded))
					return
				}
				first(r, a)
			}
			return r
		}})
	err := e.authenticate(t.Context(), "grok", map[string]any{"method": "browser_oauth"})
	wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureTimeout))
	reqs := e.client.requests()
	if len(reqs) != 4 { // the open_url's, then three paste forms
		t.Fatalf("%d elicitations, want the open_url's and three paste forms", len(reqs))
	}
	for i, r := range reqs[2:] {
		if strings.Contains(messageOf(r), bad) {
			t.Errorf("re-ask %d's message %q holds the paste", i+1, messageOf(r))
		}
		if !strings.Contains(messageOf(r), "did not complete") {
			t.Errorf("re-ask %d's message %q does not say the paste failed", i+1, messageOf(r))
		}
	}
	if strings.Contains(messageOf(reqs[1]), "did not complete") {
		t.Errorf("the first paste form's message %q reads as a re-ask", messageOf(reqs[1]))
	}
}

// A confirm step is a form with one boolean allow, default false; an accept
// of true and of false each become the answer; a decline cancels the sign-in,
// auth_required of kind canceled, and is never read as allow false.
func TestAuthenticateConfirm(t *testing.T) {
	m := offeredMethod(t, "openai", "chatgpt_plan")
	for _, allow := range []bool{true, false} {
		e := newSIEnv(t, siSpec{caps: formOnly,
			script: func(int, acp.CreateElicitationRequest) elicitReply {
				return accept(map[string]acp.ElicitationContentValue{"allow": acp.NewElicitationContentValueBoolean(allow)})
			},
			run: func(c signInCall) *scriptRun { return confirmRun("grok", m) }})
		if err := e.authenticate(t.Context(), "grok", map[string]any{"method": "browser_oauth"}); err != nil {
			t.Fatalf("allow %v: %v", allow, err)
		}
		f := e.client.requests()[0].Form
		allowProp, ok := f.RequestedSchema.Properties["allow"]
		if f == nil || !ok || allowProp.Boolean == nil || allowProp.Boolean.Default == nil || *allowProp.Boolean.Default || !slices.Equal(f.RequestedSchema.Required, []string{"allow"}) {
			t.Fatalf("elicitation %+v, want a form of one required boolean allow, default false", e.client.requests()[0])
		}
		if !strings.Contains(f.Message, "Allow background use?") || !strings.Contains(f.Message, "https://example.com/usage") {
			t.Errorf("message %q lacks the step's text and link", f.Message)
		}
		ans := e.run(0).answered()
		if len(ans) != 1 || ans[0].Confirm == nil || *ans[0].Confirm != allow || ans[0].Seq != 1 {
			t.Errorf("allow %v: answers %v, want one confirm %v for seq 1", allow, ans, allow)
		}
	}
	d := newSIEnv(t, siSpec{caps: formOnly, script: func(int, acp.CreateElicitationRequest) elicitReply { return declineReply() },
		run: func(c signInCall) *scriptRun { return confirmRun("grok", m) }})
	err := d.authenticate(t.Context(), "grok", map[string]any{"method": "browser_oauth"})
	wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureCanceled))
	if a := d.run(0).answered(); len(a) != 0 || !d.run(0).wasCanceled() {
		t.Errorf("a decline: answers %v, cancelled %v, want none and cancelled", a, d.run(0).wasCanceled())
	}
}

// A client with no elicitation gets auth_required naming the provider, a
// reason and the command, as before; nothing starts.
func TestAuthenticateNoElicitation(t *testing.T) {
	e := newSIEnv(t, deviceSpec(t, nil, nil))
	err := e.authenticate(t.Context(), "grok", nil)
	d := wantAuthRequired(t, err, "grok", "")
	if d.Reason != "no credential" || d.Command != "gobble auth login grok" {
		t.Errorf("data %+v, want reason \"no credential\" and the bare command", d)
	}
	if c := e.callList(); len(c) != 0 || len(e.client.requests()) != 0 {
		t.Errorf("calls %v, elicitations %d, want a sign-in never started", c, len(e.client.requests()))
	}
	// A client that offers only elicitation.url cannot ask forms either.
	u := newSIEnv(t, deviceSpec(t, &acp.ElicitationCapabilities{Url: &acp.ElicitationUrlCapabilities{}}, nil))
	wantAuthRequired(t, u.authenticate(t.Context(), "grok", nil), "grok", "")
	if c := u.callList(); len(c) != 0 {
		t.Errorf("a url-only client: calls %v, want none", c)
	}
}

// A credential present and no method named: {} with no elicitation; a method
// named that the stored credential is not signs in; one it is does not.
func TestAuthenticatePresent(t *testing.T) {
	held := map[string]bool{"device_code": true}
	e := newSIEnv(t, siSpec{caps: formOnly, has: map[string]bool{"grok": true},
		holds: func(_ string, m authcontract.Method) bool { return held[m.ID] },
		run:   func(c signInCall) *scriptRun { return deviceRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
	if err := e.authenticate(t.Context(), "grok", nil); err != nil {
		t.Fatalf("no method named: %v", err)
	}
	if err := e.authenticate(t.Context(), "grok", map[string]any{"method": "device_code"}); err != nil {
		t.Fatalf("the stored credential's own method: %v", err)
	}
	if c := e.callList(); len(c) != 0 || len(e.client.requests()) != 0 {
		t.Fatalf("calls %v, elicitations %d, want none while the credential is what was asked", c, len(e.client.requests()))
	}
	done := e.start("grok", map[string]any{"method": "browser_oauth"})
	e.waitRequests(1)
	c := e.callList()
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
	if len(c) != 1 || c[0].Method != "browser_oauth" {
		t.Errorf("calls %v, want a browser_oauth sign-in for a method the stored credential is not", c)
	}
}

// fresh asks for a sign-in whatever is stored: the device sign-in runs; the
// store's side of it (replaced only when it saves) is held in internal/cli.
func TestAuthenticateFresh(t *testing.T) {
	e := newSIEnv(t, siSpec{caps: formOnly, has: map[string]bool{"grok": true}, holds: func(string, authcontract.Method) bool { return true },
		run: func(c signInCall) *scriptRun { return deviceRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
	done := e.start("grok", map[string]any{"method": "device_code", "fresh": true})
	e.waitRequests(1)
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
	if got := e.callList(); len(got) != 1 {
		t.Fatalf("calls %v, want the fresh sign-in to run once", got)
	}
	e.mu.Lock()
	signed := slices.Clone(e.signed)
	e.mu.Unlock()
	if !slices.Equal(signed, []string{"grok"}) {
		t.Errorf("Signed called for %v, want grok once, after the credential saved", signed)
	}
	// fresh: false is the same as absent.
	if err := e.authenticate(t.Context(), "grok", map[string]any{"fresh": false}); err != nil {
		t.Errorf("fresh false: %v", err)
	}
	if got := e.callList(); len(got) != 1 {
		t.Errorf("calls %v after fresh false, want no new sign-in", got)
	}
}

// _meta.gobble is decoded strictly: a non-boolean fresh or remote, a member
// gobble does not know, a method not offered and a member of the wrong shape
// are -32602, and nothing starts (0051-MADR D4 step 1).
func TestAuthenticateFreshRefusesNonBoolean(t *testing.T) {
	e := newSIEnv(t, deviceSpec(t, formOnly, nil))
	for name, meta := range map[string]map[string]any{
		"fresh a string":  {"fresh": "yes"},
		"fresh a number":  {"fresh": 1},
		"fresh null":      {"fresh": nil},
		"method null":     {"method": nil},
		"remote null":     {"remote": nil},
		"remote a string": {"remote": "yes"},
		"remote a number": {"remote": 1},
		"unknown member":  {"fresh": true, "reconnect": true},
		"method a number": {"method": 5},
		"method unknown":  {"method": "no_such_method"},
	} {
		err := e.authenticate(t.Context(), "grok", meta)
		if code, d := authData(t, err); code != -32602 || d.Reason == "" {
			t.Errorf("%s: error code %d %+v, want -32602 with a reason", name, code, d)
		}
	}
	// _meta.gobble that is not an object.
	_, err := e.conn.Client.Authenticate(t.Context(), acp.AuthenticateRequest{MethodId: "grok", Meta: map[string]any{gobble: "fresh"}})
	if code, _ := authData(t, err); code != -32602 {
		t.Errorf("_meta.gobble a string: %v, want -32602", err)
	}
	if c := e.callList(); len(c) != 0 || len(e.client.requests()) != 0 {
		t.Errorf("calls %v, elicitations %d, want nothing started", c, len(e.client.requests()))
	}
}

// remote: true for a client that names no method (the owner's ruling of
// 2026-11-16, "Ask the client to choose"): with more than one method a remote
// person can complete, the client is asked by the one-field enum a local client
// gets, listing only those methods in the offered order, and choosing each
// starts it with remote true; with exactly one, that one starts with no
// question; a method named is not asked about. Offered
// [browser_oauth (by paste), device_code] at v1.31.0, and with the stand-in's
// loopback-only method first, it is not in the enum.
func TestAuthenticateOnlyRemoteMethods(t *testing.T) {
	browser, device := offeredMethod(t, "grok", "browser_oauth"), offeredMethod(t, "grok", "device_code")
	loopOnly := standIn(t)
	choose := func(label string) func(int, acp.CreateElicitationRequest) elicitReply {
		return func(n int, req acp.CreateElicitationRequest) elicitReply {
			if req.Form != nil {
				if p, ok := req.Form.RequestedSchema.Properties[fieldMethod]; ok && p.String != nil {
					return accept(map[string]acp.ElicitationContentValue{fieldMethod: text(label)})
				}
			}
			if n > 0 && metaStepOf(t, req).Kind == authcontract.StepPaste {
				return accept(map[string]acp.ElicitationContentValue{"callback": text(goodCallback)})
			}
			return accept(nil)
		}
	}
	run := func(c signInCall) *scriptRun {
		if c.Method == "device_code" {
			return deviceRun(c.ID, device)
		}
		return pasteRun(c.ID, browser)
	}
	for name, offered := range map[string][]authcontract.Method{
		"grok at the pinned facts":     {browser, device},
		"with the loopback-only first": {loopOnly, browser, device},
	} {
		for _, pick := range []authcontract.Method{browser, device} {
			e := newSIEnv(t, siSpec{caps: formOnly, script: choose(pick.Label), run: run,
				methods: func(string) []authcontract.Method { return offered }})
			done := e.start("grok", map[string]any{"remote": true})
			if pick.ID == "device_code" {
				e.waitRequests(2)
				e.run(0).finish(nil)
			}
			if err := e.result(done); err != nil {
				t.Fatalf("%s, choosing %s: %v", name, pick.ID, err)
			}
			reqs := e.client.requests()
			p := reqs[0].Form.RequestedSchema.Properties[fieldMethod]
			if p.String == nil || !slices.Equal(p.String.Enum, []string{browser.Label, device.Label}) {
				t.Errorf("%s: the choice %+v, want an enum of exactly the remote-capable methods %q", name, p, []string{browser.Label, device.Label})
			}
			if c := e.callList(); !slices.Equal(c, []signInCall{{"grok", pick.ID, true}}) {
				t.Errorf("%s, choosing %s: calls %v, want it started with remote true", name, pick.ID, c)
			}
		}
	}
	// The enum lists only what Remote accepts: with a Remote that refuses the
	// device sign-in, the browser sign-in is the only one, so it starts with no
	// question; the device sign-in is never offered to the remote client.
	{
		e := newSIEnv(t, siSpec{caps: formOnly, script: choose("unused"), run: run,
			opts: func(o *Options) { o.Auth.Remote = func(m authcontract.Method) bool { return m.ID != "device_code" } }})
		if err := e.authenticate(t.Context(), "grok", map[string]any{"remote": true}); err != nil {
			t.Fatal(err)
		}
		if c := e.callList(); !slices.Equal(c, []signInCall{{"grok", "browser_oauth", true}}) {
			t.Errorf("a Remote that refuses the device sign-in: calls %v, want browser_oauth with remote true", c)
		}
		for _, r := range e.client.requests() {
			if p, ok := r.Form.RequestedSchema.Properties[fieldMethod]; ok && p.String != nil {
				t.Errorf("one remote-capable method, and the client was asked to choose: %+v", p)
			}
		}
	}
	// A method named is not asked about, each driven as named with remote
	// passed on.
	for _, pick := range []authcontract.Method{browser, device} {
		e := newSIEnv(t, siSpec{caps: formOnly, script: choose("unused"), run: run})
		done := e.start("grok", map[string]any{"remote": true, "method": pick.ID})
		if pick.ID == "device_code" {
			e.waitRequests(1)
			e.run(0).finish(nil)
		}
		if err := e.result(done); err != nil {
			t.Fatalf("named %s: %v", pick.ID, err)
		}
		for _, r := range e.client.requests() {
			if p, ok := r.Form.RequestedSchema.Properties[fieldMethod]; ok && p.String != nil {
				t.Errorf("named %s: the client was asked to choose a method", pick.ID)
			}
		}
		if c := e.callList(); !slices.Equal(c, []signInCall{{"grok", pick.ID, true}}) {
			t.Errorf("named %s: calls %v, want it with remote true", pick.ID, c)
		}
	}
}

// remote with no method named and exactly one method a remote person can
// complete: that one starts, with no question. Through the stand-in, offered
// [browser_oauth (loopback only), device_code], it is the device sign-in. A
// local client is asked to choose between grok's two as before, and a client
// that did not advertise elicitation.form is not asked: auth_required with the
// command, as for a local one (0051-MADR D4 step 3).
func TestAuthenticateRemoteDefault(t *testing.T) {
	loopOnly := standIn(t)
	dev := offeredMethod(t, "grok", "device_code")
	e := newSIEnv(t, siSpec{caps: formOnly, methods: func(string) []authcontract.Method { return []authcontract.Method{loopOnly, dev} },
		run: func(c signInCall) *scriptRun { return deviceRun(c.ID, dev) }})
	done := e.start("grok", map[string]any{"remote": true})
	e.waitRequests(1)
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
	reqs := e.client.requests()
	if c := e.callList(); !slices.Equal(c, []signInCall{{"grok", "device_code", true}}) || len(reqs) != 1 || reqs[0].Form.RequestedSchema.Properties[fieldMethod].String != nil {
		t.Errorf("stand-in: calls %v, elicitations %+v, want the device sign-in with remote true and no method question", c, reqs)
	}
	// A local client is asked to choose between grok's two.
	l := newSIEnv(t, siSpec{caps: formOnly,
		script: func(n int, req acp.CreateElicitationRequest) elicitReply {
			if p, ok := req.Form.RequestedSchema.Properties[fieldMethod]; ok && p.String != nil {
				return accept(map[string]acp.ElicitationContentValue{fieldMethod: text(p.String.Enum[1])})
			}
			return accept(nil)
		},
		run: func(c signInCall) *scriptRun { return deviceRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
	done = l.start("grok", nil)
	reqs = l.waitRequests(2)
	l.run(0).finish(nil)
	if err := l.result(done); err != nil {
		t.Fatal(err)
	}
	p := reqs[0].Form.RequestedSchema.Properties[fieldMethod]
	want := []string{offeredMethod(t, "grok", "browser_oauth").Label, offeredMethod(t, "grok", "device_code").Label}
	if p.String == nil || !slices.Equal(p.String.Enum, want) {
		t.Errorf("a local client's choice %+v, want an enum of %q", p, want)
	}
	if c := l.callList(); !slices.Equal(c, []signInCall{{"grok", "device_code", false}}) {
		t.Errorf("local, chose the device sign-in: calls %v, want it with remote false", c)
	}
	// A remote client without elicitation.form is not asked: the same answer
	// as a local one (D4 step 3).
	n := newSIEnv(t, siSpec{caps: nil})
	err := n.authenticate(t.Context(), "grok", map[string]any{"remote": true})
	if code, d := authData(t, err); code != -32000 || d.Provider != "grok" || d.Command == "" || d.Kind != "" || d.Reason != "no credential" {
		t.Errorf("a remote client with no form: error code %d %+v, want auth_required with the command, reason %q and no kind (nothing sent to the client)", code, d, "no credential")
	}
	if c := n.callList(); len(c) != 0 {
		t.Errorf("a remote client with no form: calls %v, want none", c)
	}
}

// The stand-in of the contract's loopback-only shape, named by a local client:
// one open_url whose step has no accepts, no paste form after its accept, and
// the loopback's finish completes it; with elicitation.url, URL mode then
// elicitation/complete.
func TestAuthenticateLoopbackHasNoPaste(t *testing.T) {
	loop := standIn(t)
	spec := func(caps *acp.ElicitationCapabilities) siSpec {
		return siSpec{caps: caps, methods: func(string) []authcontract.Method { return []authcontract.Method{loop} },
			run: func(c signInCall) *scriptRun { return loopbackRun(c.ID, loop) }}
	}
	e := newSIEnv(t, spec(formOnly))
	done := e.start("grok", map[string]any{"method": "browser_oauth"})
	reqs := e.waitRequests(1)
	// What has happened, not what has not yet: the step shown is open_url with
	// no accepts, and an answer sent for it is refused by the engine.
	s := metaStepOf(t, reqs[0])
	if s.Kind != authcontract.StepOpenURL || s.Accepts != "" || s.URL != authorizeURL {
		t.Errorf("step %+v, want open_url with no accepts", s)
	}
	if err := e.run(0).Answer(t.Context(), authcontract.Answer{SchemaVersion: authcontract.SchemaVersion, Seq: s.Seq, Paste: goodCallback}); err == nil {
		t.Error("the engine took an answer to a step that has no accepts")
	}
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
	if n := len(e.client.requests()); n != 1 {
		t.Fatalf("%d elicitations by the sign-in's end, want only the open_url's", n)
	}
	if f := reqs[0].Form; f == nil || len(f.RequestedSchema.Properties) != 0 || !strings.Contains(f.Message, authorizeURL) {
		t.Errorf("elicitation %+v, want a form with no fields and the URL in its message", reqs[0])
	}
	if a := e.run(0).answered(); len(a) != 0 {
		t.Errorf("answers %v, want none", a)
	}
	u := newSIEnv(t, spec(withURL))
	done = u.start("grok", map[string]any{"method": "browser_oauth"})
	reqs = u.waitRequests(1)
	u.run(0).finish(nil)
	if err := u.result(done); err != nil {
		t.Fatal(err)
	}
	if reqs[0].Url == nil || reqs[0].Url.Url != authorizeURL {
		t.Fatalf("elicitation %+v, want URL mode to the authorize URL", reqs[0])
	}
	waitUntil(t, "elicitation/complete", func() bool { return len(u.client.completed()) == 1 })
}

// With remote: true the stand-in loopback-only method, named, is refused
// before anything starts: auth_required of kind unsupported, a reason and the
// host command; no elicitation; the engine never called. Without remote the
// same request reaches the engine, with remote false.
func TestAuthenticateRemoteRefusesLoopbackOnly(t *testing.T) {
	loop := standIn(t)
	e := newSIEnv(t, siSpec{caps: formOnly, methods: func(string) []authcontract.Method { return []authcontract.Method{loop} },
		run: func(c signInCall) *scriptRun { return loopbackRun(c.ID, loop) }})
	// The request shape magic-cli-remote sends on every sign-in the phone
	// starts: method, fresh and remote together; fresh must not skip the filter.
	for name, meta := range map[string]map[string]any{
		"method and remote":        {"method": "browser_oauth", "remote": true},
		"method, fresh and remote": {"method": "browser_oauth", "fresh": true, "remote": true},
	} {
		err := e.authenticate(t.Context(), "grok", meta)
		d := wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureUnsupported))
		if d.Command != "gobble auth login grok --method browser" || d.Reason == "" {
			t.Errorf("%s: data %+v, want a reason and the command gobble auth login grok --method browser", name, d)
		}
		if c := e.callList(); len(c) != 0 || len(e.client.requests()) != 0 {
			t.Fatalf("%s: calls %v, elicitations %d, want none", name, c, len(e.client.requests()))
		}
	}
	done := e.start("grok", map[string]any{"method": "browser_oauth"})
	e.waitRequests(1)
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
	if c := e.callList(); !slices.Equal(c, []signInCall{{"grok", "browser_oauth", false}}) {
		t.Errorf("without remote: calls %v, want the sign-in with remote false", c)
	}
}

// remote absent or false is a local sign-in: the engine is told remote false,
// which is how the listener stays (the option itself is held in llm/provider).
func TestAuthenticateLocalKeepsListener(t *testing.T) {
	for name, meta := range map[string]map[string]any{"absent": {"method": "device_code"}, "false": {"method": "device_code", "remote": false}} {
		e := newSIEnv(t, deviceSpec(t, formOnly, nil))
		done := e.start("grok", meta)
		e.waitRequests(1)
		e.run(0).finish(nil)
		if err := e.result(done); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if c := e.callList(); !slices.Equal(c, []signInCall{{"grok", "device_code", false}}) {
			t.Errorf("remote %s: calls %v, want remote false", name, c)
		}
	}
}

// A second authenticate for a provider while one runs is -32600, naming the
// running one; another provider's runs.
func TestAuthenticateOneAtATime(t *testing.T) {
	e := newSIEnv(t, deviceSpec(t, formOnly, func(int, acp.CreateElicitationRequest) elicitReply { return accept(nil) }))
	first := e.start("grok", map[string]any{"method": "device_code"})
	e.waitRequests(1)
	err := e.authenticate(t.Context(), "grok", map[string]any{"method": "device_code"})
	code, d := authData(t, err)
	if code != -32600 || !strings.Contains(d.Reason, "device_code") {
		t.Errorf("a second grok sign-in: code %d %+v, want -32600 naming the running one", code, d)
	}
	// The guard is the provider's, not the method's: another method of the same
	// provider is refused as well.
	err = e.authenticate(t.Context(), "grok", map[string]any{"method": "browser_oauth"})
	code, d = authData(t, err)
	if code != -32600 || !strings.Contains(d.Reason, "device_code") {
		t.Errorf("a second grok sign-in by another method: code %d %+v, want -32600 naming the running one", code, d)
	}
	if c := e.callList(); len(c) != 1 {
		t.Errorf("calls %v, want the second grok request never to reach the engine", c)
	}
	other := e.start("kilo", map[string]any{"method": "device_code"})
	waitUntil(t, "kilo's run", func() bool { return len(e.callList()) == 2 })
	e.run(0).finish(nil)
	e.run(1).finish(nil)
	for _, ch := range []<-chan error{first, other} {
		if err := e.result(ch); err != nil {
			t.Errorf("a sign-in: %v", err)
		}
	}
	// Once ended, the provider takes another.
	again := e.start("grok", map[string]any{"method": "device_code"})
	e.run(2).finish(nil)
	if err := e.result(again); err != nil {
		t.Errorf("after the first ended: %v", err)
	}
}

// A request the client makes while an elicitation waits is handled, and the
// answer then completes the sign-in: no lock is held across the wait (U8 (e)).
func TestAuthenticateWithCancelPending(t *testing.T) {
	release := make(chan struct{})
	e := newSIEnv(t, deviceSpec(t, formOnly, func(int, acp.CreateElicitationRequest) elicitReply {
		<-release
		return accept(nil)
	}))
	done := e.start("grok", map[string]any{"method": "device_code"})
	e.waitRequests(1)
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	sess, err := e.conn.Client.NewSession(ctx, acp.NewSessionRequest{Cwd: acptest.Abs("/work"), McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatalf("session/new while an elicitation waits: %v", err)
	}
	if err := e.conn.Client.Cancel(ctx, acp.CancelNotification{SessionId: sess.SessionId}); err != nil {
		t.Fatalf("session/cancel while an elicitation waits: %v", err)
	}
	close(release)
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
}

// The client's $/cancel_request cancels the sign-in and the pending
// elicitation, and nothing of the relay is left running.
func TestAuthenticateClientCancels(t *testing.T) {
	e := newSIEnv(t, deviceSpec(t, formOnly, func(int, acp.CreateElicitationRequest) elicitReply { return hangReply() }))
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- e.authenticate(ctx, "grok", map[string]any{"method": "device_code"}) }()
	e.waitRequests(1)
	cancel()
	if err := e.result(done); err == nil {
		t.Error("a cancelled authenticate answered success")
	}
	waitUntil(t, "the pending elicitation to be cancelled", func() bool { return e.client.endedCount() == 1 })
	waitUntil(t, "the sign-in to be cancelled", func() bool { return e.run(0).wasCanceled() })
	// The provider is free again once the handler has returned.
	waitUntil(t, "the sign-in to be forgotten", func() bool {
		e.agent.mu.Lock()
		defer e.agent.mu.Unlock()
		return len(e.agent.signing) == 0
	})
	again := e.start("grok", map[string]any{"method": "device_code"})
	e.waitRequests(2)
	e.run(1).finish(nil)
	if err := e.result(again); err != nil {
		t.Errorf("after the cancel: %v", err)
	}
	// The second form is ended by the sign-in's end; the test waits for that,
	// so the pipes do not close on the cancel still on its way.
	waitUntil(t, "the second form to be cancelled", func() bool { return e.client.endedCount() == 2 })
}

// A pasted callback is in the one answer to the engine and in no other place:
// not the agent's log (debug level), nor a frame but the client's answer,
// nor an error; each paste elicitation's step says its answer is secret, and
// the answer built for the engine carries Secret true.
func TestAuthenticateNeverLogsPaste(t *testing.T) {
	e := newSIEnv(t, siSpec{caps: formOnly,
		script: func(n int, _ acp.CreateElicitationRequest) elicitReply {
			if n == 0 {
				return accept(nil)
			}
			return accept(map[string]acp.ElicitationContentValue{"callback": text(goodCallback)})
		},
		run: func(c signInCall) *scriptRun { return pasteRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
	if err := e.authenticate(t.Context(), "grok", map[string]any{"method": "browser_oauth"}); err != nil {
		t.Fatal(err)
	}
	const secret = "SECRET-CODE-7"
	if strings.Contains(e.log.String(), secret) {
		t.Errorf("the agent's log holds the paste:\n%s", e.log.String())
	}
	holding := 0
	for _, r := range e.conn.Recorder.Records() {
		if strings.Contains(string(r.Frame), secret) {
			holding++
			if r.Direction != sdkacptest.Sent || !strings.Contains(string(r.Frame), `"result"`) {
				t.Errorf("a frame other than the client's answer holds the paste: %s", r.Frame)
			}
		}
	}
	if holding != 1 {
		t.Errorf("%d frames hold the paste, want the client's one answer", holding)
	}
	for _, r := range e.client.requests()[1:] {
		if s := metaStepOf(t, r); s.Kind == authcontract.StepPaste && !s.Secret {
			t.Errorf("a paste elicitation's step says its answer is not secret: %+v", s)
		}
	}
	ans := e.run(0).answered()
	if len(ans) != 1 || ans[0].Secret == nil || !*ans[0].Secret {
		t.Fatalf("answers %v, want one with Secret true", ans)
	}
	if got := fmt.Sprintf("%v %+v %#v", ans[0], ans[0], ans[0]); strings.Contains(got, secret) {
		t.Errorf("the answer formats with the paste: %s", got)
	}
}

// A key a device sign-in yields (Kilo's) reaches the host's store through the
// run alone: it is in no log line, no frame to the client and no error, and the
// answer is {} (0051-MADR D3, D4 "Secrets").
func TestAuthenticateKeyNeverLeaks(t *testing.T) {
	const key = "KEY-SECRET-9"
	e := newSIEnv(t, siSpec{caps: formOnly, run: func(c signInCall) *scriptRun {
		r := deviceRun(c.ID, offeredMethod(t, c.ID, c.Method))
		r.key = key
		return r
	}})
	done := e.start("grok", map[string]any{"method": "device_code"})
	e.waitRequests(1)
	e.run(0).finish(nil)
	if err := e.result(done); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(e.log.String(), key) {
		t.Errorf("the agent's log holds the key:\n%s", e.log.String())
	}
	for _, r := range e.conn.Recorder.Records() {
		if strings.Contains(string(r.Frame), key) {
			t.Errorf("a frame holds the key: %s", r.Frame)
		}
	}
}

// Each agent auth method's _meta.gobble.methods decodes strictly into
// []authcontract.Method and equals Offered, for every provider acpext lists:
// those that sign in and those that take a key alone.
func TestAuthMethodsMeta(t *testing.T) {
	e := newSIEnv(t, siSpec{opts: func(o *Options) { o.Auth.Providers = acpext.AuthProviders }})
	resp, err := e.conn.Client.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber})
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, m := range resp.AuthMethods {
		if m.Agent == nil {
			continue
		}
		id := string(m.Agent.Id)
		raw, ok := m.Agent.Meta[gobble].(map[string]any)
		if !ok {
			t.Fatalf("%s: _meta.gobble = %v, want an object", id, m.Agent.Meta[gobble])
		}
		b, err := json.Marshal(raw[acpext.MetaMethods])
		if err != nil {
			t.Fatal(err)
		}
		var got []authcontract.Method
		if err := json.Unmarshal(b, &got, json.RejectUnknownMembers(true)); err != nil {
			t.Fatalf("%s: methods: %v", id, err)
		}
		want := provider.Offered(id)
		if len(want) == 0 {
			t.Fatalf("%s: Offered is empty, the test holds nothing", id)
		}
		wb, _ := json.Marshal(want)
		gb, _ := json.Marshal(got)
		if !bytes.Equal(wb, gb) {
			t.Errorf("%s: methods\n%s\nwant\n%s", id, gb, wb)
		}
		seen++
	}
	if want := len(acpext.AuthProviders()); seen != want {
		t.Errorf("%d agent methods with methods, want %d", seen, want)
	}
}

// The frames of a form-only browser-paste sign-in match the golden and pass
// the SDK's ValidateGolden.
func TestAuthenticateGolden(t *testing.T) {
	e := newSIEnv(t, siSpec{caps: formOnly,
		script: func(n int, _ acp.CreateElicitationRequest) elicitReply {
			if n == 0 {
				return accept(nil)
			}
			return accept(map[string]acp.ElicitationContentValue{"callback": text(goodCallback)})
		},
		run: func(c signInCall) *scriptRun { return pasteRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
	if err := e.authenticate(t.Context(), "grok", map[string]any{"method": "browser_oauth"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := e.conn.Recorder.Wait(ctx, 8); err != nil {
		t.Fatal(err)
	}
	sdkacptest.Golden(t, "testdata/authenticate-elicitation.golden", acptest.HostIndependent(e.conn.Recorder.Records()))
	sdkacptest.ValidateGolden(t, e.conn.Recorder.Records())
}

// The provider is reserved before the client is asked which method to sign in
// by: a second request while the first waits on the method form is refused
// -32600 at once, and no second form is sent (D4, "One sign-in per provider").
// A form the client declines frees the provider.
func TestAuthenticateGuardComesBeforeTheMethodForm(t *testing.T) {
	release := make(chan struct{})
	device := offeredMethod(t, "grok", "device_code")
	var forms atomicCount
	e := newSIEnv(t, deviceSpec(t, formOnly, func(_ int, req acp.CreateElicitationRequest) elicitReply {
		if p, ok := req.Form.RequestedSchema.Properties[fieldMethod]; ok && p.String != nil {
			if forms.add() == 1 {
				<-release
				return accept(map[string]acp.ElicitationContentValue{fieldMethod: text(device.Label)})
			}
			return declineReply()
		}
		return accept(nil)
	}))
	first := e.start("grok", nil)
	waitUntil(t, "the method form", func() bool { return forms.get() == 1 })
	err := e.authenticate(t.Context(), "grok", nil)
	if code, d := authData(t, err); code != -32600 || !strings.Contains(d.Reason, "already running") {
		t.Errorf("a second request while the first is asked: code %d %+v, want -32600", code, d)
	}
	if n := forms.get(); n != 1 {
		t.Errorf("%d method forms were sent, want the second request refused before any", n)
	}
	close(release)
	e.waitRequests(2)
	e.run(0).finish(nil)
	if err := e.result(first); err != nil {
		t.Fatal(err)
	}
	// A declined method form releases the provider.
	err = e.authenticate(t.Context(), "grok", nil)
	wantAuthRequired(t, err, "grok", acpext.ErrorKind(authcontract.FailureCanceled))
	e.agent.mu.Lock()
	held := len(e.agent.signing)
	e.agent.mu.Unlock()
	if held != 0 {
		t.Errorf("%d sign-ins still reserved after a declined method form", held)
	}
}

// atomicCount is a counter the client's goroutine and the test share.
type atomicCount struct {
	mu sync.Mutex
	n  int
}

func (c *atomicCount) add() int { c.mu.Lock(); defer c.mu.Unlock(); c.n++; return c.n }
func (c *atomicCount) get() int { c.mu.Lock(); defer c.mu.Unlock(); return c.n }

// A remote client that names a method gobble does not drive over ACP gets the
// answer a local client gets: {} when a key is held, else the same
// auth_required, not the remote refusal (the owner's ruling of 2026-11-16, "The
// same answer as a local client"; D4 step 1's note).
func TestAuthenticateRemoteAPIKeyIsAsLocal(t *testing.T) {
	key := func(_ string, m authcontract.Method) bool { return m.Yields == authcontract.YieldsAPIKey }
	for _, held := range []bool{true, false} {
		e := newSIEnv(t, siSpec{caps: formOnly, has: map[string]bool{"grok": held}, holds: key,
			run: func(c signInCall) *scriptRun { return deviceRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
		var got, want error
		got = e.authenticate(t.Context(), "grok", map[string]any{"method": "api_key", "remote": true})
		want = e.authenticate(t.Context(), "grok", map[string]any{"method": "api_key"})
		if held {
			if got != nil || want != nil {
				t.Errorf("a key held: remote %v, local %v, want {} for both", got, want)
			}
		} else {
			gc, gd := authData(t, got)
			wc, wd := authData(t, want)
			if gc != wc || gd != wd || gd.Kind != "" || !strings.Contains(gd.Reason, "is not a sign-in gobble drives from an ACP client") {
				t.Errorf("no key: remote %d %+v, local %d %+v, want the same, not the remote refusal", gc, gd, wc, wd)
			}
		}
		if c := e.callList(); len(c) != 0 || len(e.client.requests()) != 0 {
			t.Errorf("held %v: calls %v, elicitations %d, want none", held, c, len(e.client.requests()))
		}
	}
}

// A sign-in that saved before the client's cancel arrived is a signed-in
// provider: the credential is stored, so Signed is told and the answer is not
// "cancelled" while the agent keeps the old credential (D4, "After a sign-in").
func TestAuthenticateCancelAfterSaveTellsTheAgent(t *testing.T) {
	var run *savedRun
	e := newSIEnv(t, siSpec{caps: formOnly, script: func(int, acp.CreateElicitationRequest) elicitReply { return hangReply() },
		run: func(c signInCall) *scriptRun {
			run = &savedRun{scriptRun: deviceRun(c.ID, offeredMethod(t, c.ID, c.Method))}
			return run.scriptRun
		},
		opts: func(o *Options) {
			inner := o.Auth.SignIn
			o.Auth.SignIn = func(ctx context.Context, id, method string, remote bool) (SignInRun, error) {
				if _, err := inner(ctx, id, method, remote); err != nil {
					return nil, err
				}
				return run, nil
			}
		}})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- e.authenticate(ctx, "grok", map[string]any{"method": "device_code"}) }()
	e.waitRequests(1)
	cancel()
	<-done
	waitUntil(t, "Signed", func() bool {
		e.mu.Lock()
		defer e.mu.Unlock()
		return slices.Equal(e.signed, []string{"grok"})
	})
}

// savedRun is a run that had saved its credential when the cancel arrived: its
// Wait ends in success once its context is done, as a Wait whose save completed
// first does.
type savedRun struct{ *scriptRun }

func (r *savedRun) Wait(ctx context.Context) (provider.SignInResult, error) {
	<-ctx.Done()
	return provider.SignInResult{Yields: r.method.Yields}, nil
}

// detachAuthenticate is next with authenticate run under a context its caller's
// cancel does not reach: the ACP SDK's public receiving middleware, written over
// the params type the SDK names so that this package imports no other encoding
// of JSON (the SDK's handler type is the one place it is not jsonx's).
func detachAuthenticate[P any](next func(context.Context, string, P) (any, *acp.RequestError)) func(context.Context, string, P) (any, *acp.RequestError) {
	return func(ctx context.Context, method string, params P) (any, *acp.RequestError) {
		if method == "authenticate" {
			ctx = context.WithoutCancel(ctx)
		}
		return next(ctx, method, params)
	}
}

// The relay's own end of an open form, apart from the SDK's cancel of
// the handler's context. A receiving middleware, the ACP SDK's public API, runs
// authenticate under a context the SDK does not cancel when the handler returns,
// so only the relay can end the form it opened: a form-only client leaves the
// device code's form open and the poll ends. It holds the relay's own guarantee,
// that every request it opened has ended when serve returns.
func TestAuthenticateRelayEndsItsFormWithoutTheSDK(t *testing.T) {
	var run *scriptRun
	runs := make(chan *scriptRun, 1)
	client := &elicitClient{Client: &acptest.Client{}, script: func(int, acp.CreateElicitationRequest) elicitReply { return hangReply() }}
	auth := AuthOptions{
		Providers: signInProviders,
		Has:       func(string) bool { return false },
		Methods:   provider.Offered,
		SignIn: func(_ context.Context, id, method string, _ bool) (SignInRun, error) {
			r := deviceRun(id, offeredMethod(t, id, method))
			runs <- r
			return r, nil
		},
	}
	o := Options{Version: "1.2.3", Auth: auth, Logger: slog.New(slog.DiscardHandler),
		Provider: func(string, llm.Retry) (llm.Provider, error) { return llmtest.NewScript(llmtest.Text("ok")), nil },
		Prompt:   PromptOptions{GOOS: "linux", cwd: "/work"}}
	agent := New(o)
	detach := acp.WithReceivingMiddleware(func(next acp.MethodHandler) acp.MethodHandler { return detachAuthenticate(next) })
	toAgentR, toAgentW := io.Pipe()
	toClientR, toClientW := io.Pipe()
	quiet := acp.WithLogger(slog.New(slog.DiscardHandler))
	ac := acp.NewAgentSideConnection(agent, toClientW, toAgentR, quiet, detach)
	cc := acp.NewClientSideConnection(client, toAgentW, toClientR, quiet)
	agent.SetConnection(ac)
	t.Cleanup(func() { _ = toAgentW.Close(); _ = toClientW.Close() })
	if _, err := cc.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber,
		ClientInfo: &acp.Implementation{Name: "test", Version: "0.0.0"}, ClientCapabilities: acp.ClientCapabilities{Elicitation: formOnly}}); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := cc.Authenticate(t.Context(), acp.AuthenticateRequest{MethodId: "grok", Meta: authMetaOf(map[string]any{"method": "device_code"})})
		done <- err
	}()
	select {
	case run = <-runs:
	case <-time.After(5 * time.Second):
		t.Fatal("no run")
	}
	waitUntil(t, "the form", func() bool { return len(client.requests()) == 1 })
	run.finish(nil)
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("authenticate did not answer")
	}
	waitUntil(t, "the open form to be cancelled by the relay", func() bool { return client.endedCount() == 1 })
}

// _meta.gobble: null is an absent container, as ACP reads a null one (the owner's
// ruling of 2026-11-16, "Refuse `method: null`; read `gobble: null` as absent"):
// with a credential held the request is answered {} and nothing starts, as one
// with no _meta at all; a null member inside it is -32602 (the test above).
func TestAuthenticateNullContainerIsAbsent(t *testing.T) {
	e := newSIEnv(t, siSpec{caps: formOnly, has: map[string]bool{"grok": true}, run: func(c signInCall) *scriptRun { return deviceRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
	if _, err := e.conn.Client.Authenticate(t.Context(), acp.AuthenticateRequest{MethodId: "grok", Meta: map[string]any{gobble: nil}}); err != nil {
		t.Errorf("_meta.gobble null: %v, want {} as for no _meta", err)
	}
	if err := e.authenticate(t.Context(), "grok", nil); err != nil {
		t.Errorf("no _meta: %v", err)
	}
	if c := e.callList(); len(c) != 0 || len(e.client.requests()) != 0 {
		t.Errorf("calls %v, elicitations %d, want nothing started", c, len(e.client.requests()))
	}
}

// The guard stops two sign-ins racing, so it applies to a request that would
// start one: while a fresh sign-in runs with a credential held, a request with
// no fresh and no method is answered {} at once, and one that would sign in is
// -32600 (the owner's ruling of 2026-11-16, "Keep `{}` for a request that needs
// no sign-in").
func TestAuthenticateGuardAppliesToSignInsOnly(t *testing.T) {
	e := newSIEnv(t, siSpec{caps: formOnly, has: map[string]bool{"grok": true}, holds: func(string, authcontract.Method) bool { return true },
		run: func(c signInCall) *scriptRun { return deviceRun(c.ID, offeredMethod(t, c.ID, c.Method)) }})
	first := e.start("grok", map[string]any{"method": "device_code", "fresh": true})
	e.waitRequests(1)
	if err := e.authenticate(t.Context(), "grok", nil); err != nil {
		t.Errorf("a request that needs no sign-in, while one runs: %v, want {}", err)
	}
	if err := e.authenticate(t.Context(), "grok", map[string]any{"method": "device_code"}); err != nil {
		t.Errorf("a named method the credential is, while one runs: %v, want {}", err)
	}
	err := e.authenticate(t.Context(), "grok", map[string]any{"method": "device_code", "fresh": true})
	if code, _ := authData(t, err); code != -32600 {
		t.Errorf("a second fresh sign-in: %v, want -32600", err)
	}
	if c := e.callList(); len(c) != 1 {
		t.Errorf("calls %v, want only the first sign-in", c)
	}
	e.run(0).finish(nil)
	if err := e.result(first); err != nil {
		t.Fatal(err)
	}
}
