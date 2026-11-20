package provider

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/auth"
	"github.com/maccavelli/go-llmprovider-sdk/wizard"
)

// The ways to sign in gobble offers, as --method names them (0025-MADR D6).
const (
	MethodAPIKey  = "api-key"
	MethodBrowser = "browser"
	MethodDevice  = "device"
	// MethodChatGPTPlan is OpenAI's Sign in with ChatGPT for agents, under
	// gobble's own name (0031-MADR D7, amended 2026-11-10).
	MethodChatGPTPlan = "chatgpt-plan"
)

// AgentName is the name gobble signs in under, where a sign-in shows the
// app's own: OpenAI's ChatGPT plan sign-in (its terms, section 1).
const AgentName = "gobble"

// methodIDs are gobble's methods as the SDK's.
var methodIDs = map[string]llmprovider.AuthMethodID{
	MethodAPIKey:      llmprovider.AuthAPIKey,
	MethodBrowser:     llmprovider.AuthBrowserOAuth,
	MethodDevice:      llmprovider.AuthDeviceCode,
	MethodChatGPTPlan: llmprovider.AuthChatGPTPlan,
}

// Choice is one answer to a sign-in's question: its label and a detail
// shown beside it.
type Choice struct{ Label, Detail string }

// Prompter is what a sign-in asks the user through (0031-MADR D7). Input
// gives up when ctx ends and leaves no read running that could take a later
// answer. Notify may be called while Input waits.
type Prompter interface {
	Choose(title string, choices []Choice, def int) (int, error)
	Confirm(question string, def bool) (bool, error)
	Input(ctx context.Context, prompt string) (string, error)
	Secret(prompt string) (string, error)
	Notify(format string, args ...any)
}

// SignInOptions configure SignIn. The embedded Options carry the run's
// provider set (nil is the built-ins) and the listing that checks the
// credential: its client name and version, its HTTP client, headers and
// base URL; HTTPClient also carries the sign-in's own requests.
type SignInOptions struct {
	Options
	// Method is one of Methods(id); "" lets the user choose.
	Method string
	// NoPaste starts no paste prompt in a browser sign-in, for a caller with
	// no terminal: the browser's return alone completes it.
	NoPaste bool
	// ShowDeviceCode is given a device sign-in's code; an error from it ends
	// the sign-in. Nil tells the user through the Prompter.
	ShowDeviceCode func(DeviceCode) error
	// OpenURL is given a browser sign-in's URL, after the Prompter was told it.
	OpenURL func(string) error
	// Steps is given each step of the sign-in as the SDK's auth contract
	// states it, for a caller with no terminal, such as one that prints them:
	// the steps that show something (open_url, show_code, waiting), in order,
	// never the last one, done or failed, which the caller states once it has
	// stored the result. It takes the place of ShowDeviceCode and OpenURL,
	// which are then not called. No paste is asked, and a step that asks
	// (paste, confirm or form) ends the sign-in with ErrAsks, as an error of
	// the kind unsupported. An error from Steps ends the sign-in and is
	// returned as it is (0051-MADR D3, D5).
	Steps func(authcontract.Step) error
	// HostID is this installation's host id, which the ChatGPT plan's
	// sign-in registers its client with and keeps for the host's life; the
	// caller reads it from <state>/host-id (0051-PLAN P6 step 1a, 0052-MADR
	// D4). Empty lets the SDK make a one-off id, which a caller that could
	// not read the file warns of.
	HostID string
	// Remote says that the person completing StartSignIn's sign-in is not
	// on the agent's host: the engine is told to bind no listener, so a paste
	// alone completes a browser sign-in (0051-MADR D3's note of 2026-11-11;
	// whether it binds none is the SDK's). No terminal
	// path sets it, and SignIn ignores it.
	Remote bool
}

// resolveCredential is the wizard's sign-in; a test replaces it to record
// the options it was given.
var resolveCredential = wizard.ResolveCredential

// Credential is what a sign-in gives: an API key to store, or none when it
// saved a login in the store.
type Credential struct {
	Key string
	// Device reports that Key came from a device sign-in, Kilo's.
	Device bool
	// Workspace is the Anthropic workspace the wizard asked for, for a
	// Claude key not scoped to one (0035-MADR D5).
	Workspace string
	// Yields is what the SDK's last step, done, says the sign-in left, when
	// SignIn ran with Steps: the caller states it in its own done step. Empty
	// otherwise.
	Yields authcontract.Yields
}

// Methods are provider id's offered ways to sign in (Offered), as --method
// names them, in the SDK's order; none for a provider that takes no key.
func Methods(id string) []string {
	var out []string
	for _, m := range Offered(id) {
		if w := wordOf(m.ID); w != "" {
			out = append(out, w)
		}
	}
	return out
}

// skips are the SDK's methods SignIn leaves out of provider id's menu: those
// the policy does not offer, and, when method is forced, every other one.
// The option applies to every provider of a call, so it is computed for id
// alone (0031-PLAN deviation 5).
func skips(d llmprovider.Descriptor, method string) []llmprovider.AuthMethodID {
	have := Offered(string(d.ID))
	var out []llmprovider.AuthMethodID
	for _, m := range d.AuthMethods {
		keep := slices.ContainsFunc(have, func(o authcontract.Method) bool { return o.ID == string(m.ID) })
		if !keep || method != "" && m.ID != methodIDs[method] {
			out = append(out, m.ID)
		}
	}
	return out
}

// SignInError is a sign-in the provider refused, or that failed. Its text is
// gobble's and holds nothing of the service's; the error it wraps keeps
// llm's class (0031-MADR D7).
type SignInError struct {
	ID string
	// Keys is how many typed keys were refused, when that ended it.
	Keys int
	err  error
}

func (e *SignInError) Error() string {
	switch {
	case e.Keys > 0:
		return fmt.Sprintf("%s refused %d keys; none was saved", e.ID, e.Keys)
	case errors.Is(e.err, ErrAsks):
		return fmt.Sprintf("%s sign-in: %s", e.ID, asksText)
	}
	return fmt.Sprintf("%s sign-in failed (%s)", e.ID, ErrorKind(e))
}

func (e *SignInError) Unwrap() error { return e.err }

// asksText is why a sign-in that asks a question cannot run for a caller that
// only reads its steps: gobble's words, naming the ways that can ask
// (0051-MADR D5).
const asksText = "this sign-in asks a question, which a program that reads its steps cannot answer; " +
	"sign in at a terminal (gobble auth login), or from an ACP client"

// Retryable reports whether the same sign-in can succeed later, as the SDK's
// own failed step states it for the same error: the API error's Retryable
// when err holds one, which honours the service's x-should-retry and a final
// reply, and otherwise whether the kind is rate_limited, offline or
// unavailable (signinflow.Failure at go-llmprovider-sdk v1.31.0).
func Retryable(err error) bool {
	if ae, ok := errors.AsType[*llmprovider.APIError](err); ok {
		return ae.Retryable()
	}
	switch ErrorKind(err) {
	case string(authcontract.FailureRateLimited), string(authcontract.FailureOffline), string(authcontract.FailureUnavailable):
		return true
	}
	return false
}

// ErrAsks is a sign-in that must ask a question (a consent's confirm, a
// form, a paste) of a caller that only reads steps, SignInOptions.Steps'.
// It is an error of the SDK's kind unsupported, and SignIn returns it inside
// a *SignInError whose text is asksText, so ErrorKind reads it as the
// contract does.
var ErrAsks = fmt.Errorf("%w: a sign-in asks a question", llmprovider.ErrUnsupported)

// asked is the *SignInError of provider id's sign-in refused for ErrAsks.
func asked(id string) error {
	return &SignInError{ID: id, err: mapError(ErrAsks)}
}

// stepDriver is the wizard's step driver for SignInOptions.Steps: it gives
// each step that shows something to steps, answers none, refuses the steps
// that ask, and swallows the last step, which the caller states after the
// result is stored (0051-MADR D3).
type stepDriver struct {
	steps func(authcontract.Step) error
	// done, when not nil, is given what the SDK's done step says the sign-in
	// yielded.
	done func(authcontract.Yields)
}

func (d stepDriver) SignInStep(_ context.Context, _ llmprovider.ProviderID, s authcontract.Step) (authcontract.Answer, error) {
	switch s.Kind {
	case authcontract.StepPaste, authcontract.StepConfirm, authcontract.StepForm:
		return authcontract.Answer{}, ErrAsks
	case authcontract.StepDone:
		if d.done != nil {
			d.done(s.Yields)
		}
		return authcontract.Answer{}, nil
	case authcontract.StepFailed:
		return authcontract.Answer{}, nil
	}
	if err := d.steps(s); err != nil {
		return authcontract.Answer{}, &ownError{err}
	}
	return authcontract.Answer{}, nil
}

// OfferedMethod is provider id's offered method that --method's word names,
// as the SDK's facts give it.
func OfferedMethod(id, word string) (authcontract.Method, bool) {
	sdk, ok := methodIDs[word]
	if !ok {
		return authcontract.Method{}, false
	}
	for _, m := range Offered(id) {
		if m.ID == string(sdk) {
			return m, true
		}
	}
	return authcontract.Method{}, false
}

// CheckNoQuestion refuses, before it starts, a sign-in of provider id by
// --method's word method that would ask a consent the stored session has
// not answered, for a caller that only reads steps and cannot answer
// (0051-MADR D5; 0052-MADR D2): the offered method has a required consent
// that provider id's stored session of that method holds no answer to, a
// session the SDK would otherwise carry the answer from. The refusal is a
// *SignInError holding ErrAsks. A method with no such consent, an
// unoffered one, and a sign-in the stored session has answered pass. A store
// that cannot be read is its error.
func CheckNoQuestion(ctx context.Context, id, method string, store LoginStore) error {
	m, ok := OfferedMethod(id, method)
	if !ok {
		return nil
	}
	for _, c := range m.Consents {
		if !c.Required {
			continue
		}
		answered, err := consentAnswered(ctx, store, id, m, c.ID)
		if err != nil {
			return err
		}
		if !answered {
			return asked(id)
		}
	}
	return nil
}

// consentAnswered reports whether provider id's stored session, signed in by
// method m, holds an answer, granted or declined, to consent.
func consentAnswered(ctx context.Context, store LoginStore, id string, m authcontract.Method, consent string) (bool, error) {
	l, err := store.Load(ctx, id)
	if err != nil {
		return false, err
	}
	stored, ok := storedMethod(id, l)
	if !ok || stored.ID != m.ID {
		return false, nil
	}
	_, answered := l.session(id, nil, nil, nil).Consent(consent)
	return answered, nil
}

// identityOf is the application o names to the identity providers: its
// ClientName and ClientVersion as the SDK's auth.Identity. SignIn gives it to
// the wizard, which names every sign-in request, and Grok's authorize and
// device referrer, with it (0051-MADR D9, N1). The zero Options name the SDK.
func identityOf(o Options) auth.Identity {
	return auth.Identity{Name: o.ClientName, Version: o.ClientVersion}
}

// SignIn signs in to provider id through go-llmprovider-sdk's wizard
// (0031-MADR D7): its methods, as gobble offers them, asked through p; a
// session saved in store; a typed key checked by a listing and asked for
// again while refused. gobble's own provider menu chooses id. An error of
// p's, ShowDeviceCode's, OpenURL's or Steps' is returned as it is; any other
// is a *SignInError, so ErrorKind reads the SDK's word for it (0051-MADR D6).
// Four errors are not: the context's own error when it ended the sign-in, which
// ErrorKind words canceled or timeout; a Prompter's error, such as a sign-in
// that asks with no terminal, which is whatever llm.Kind says of it, other
// for most; and two errors before the wizard starts, a provider that takes no
// key and a method it does not offer, which are other and which a caller that
// checks Methods first never meets. Every request of the sign-in, and the
// session it keeps, name o's ClientName and ClientVersion (identityOf;
// 0051-MADR D9).
func SignIn(ctx context.Context, id string, p Prompter, o SignInOptions, store LoginStore) (Credential, error) {
	d, ok := keyProvider(id)
	if !ok {
		return Credential{}, fmt.Errorf("provider %s takes no key and has no sign-in", id)
	}
	if o.Method != "" && !slices.Contains(Methods(id), o.Method) {
		return Credential{}, fmt.Errorf("provider %s has no %s sign-in", id, o.Method)
	}
	w := &wizardPrompter{ctx: ctx, p: p}
	var device bool
	show := func(dc wizard.DeviceCode) error {
		device = true
		c := DeviceCode{URL: dc.URL, Code: dc.Code, Expires: dc.Expiry}
		if o.ShowDeviceCode == nil {
			p.Notify("Open %s and enter the code %s", c.URL, c.Code)
			return nil
		}
		return w.mark(o.ShowDeviceCode(c))
	}
	var open func(string) error
	if o.OpenURL != nil {
		open = func(u string) error { return w.mark(o.OpenURL(u)) }
	}
	var driver wizard.StepDriver
	var yielded authcontract.Yields
	if o.Steps != nil {
		// The SDK calls ShowDeviceCode for a device sign-in only without a
		// driver, so a show_code step is what says the key came from one.
		driver = stepDriver{func(s authcontract.Step) error {
			if s.Kind == authcontract.StepShowCode {
				device = true
			}
			return o.Steps(s)
		}, func(y authcontract.Yields) { yielded = y }}
	}
	listing := append([]llmprovider.Option{
		llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),
		llmprovider.WithoutModelMetadata(),
	}, o.sdkOptions()...)
	res, _, err := resolveCredential(ctx, w, wizard.Options{
		Registry:         o.Set.registry(),
		Existing:         wizard.Result{Provider: d.ID, BaseURL: o.BaseURL, Workspace: o.Workspace},
		Discover:         true,
		ProviderOptions:  listing,
		TokenStore:       tokenStore(store),
		HostID:           o.HostID,
		HTTPClient:       o.HTTPClient,
		OpenURL:          open,
		KiloDeviceKey:    true,
		ShowDeviceCode:   show,
		SkipMethods:      skips(d, o.Method),
		NoEndpointPrompt: true,
		NoPastePrompt:    o.NoPaste || o.Steps != nil,
		Driver:           driver,
		AgentName:        AgentName,
		ConsentText:      ConsentText,
		Identity:         identityOf(o.Options),
	}, d.ID)
	if err != nil {
		if own, ok := errors.AsType[*ownError](err); ok {
			return Credential{}, own.err
		}
		if ctx.Err() != nil {
			return Credential{}, ctx.Err()
		}
		e := &SignInError{ID: id, err: mapError(err)}
		if errors.Is(err, llmprovider.ErrAuthFailure) && !device {
			e.Keys = w.secrets()
		}
		return Credential{}, e
	}
	if res.Kind == wizard.CredOAuth {
		return Credential{Yields: yielded}, nil
	}
	return Credential{Key: res.APIKey, Device: device, Workspace: res.Workspace, Yields: yielded}, nil
}

// ownError is an error of the caller's, a Prompter's or a callback's, which
// SignIn returns as it is.
type ownError struct{ err error }

func (e *ownError) Error() string { return e.err.Error() }
func (e *ownError) Unwrap() error { return e.err }

// wizardPrompter is a Prompter as the wizard's, and as one that can give up
// a read (wizard.InputerContext), so a browser sign-in the callback finished
// asks no Enter. A menu of one is answered without asking.
type wizardPrompter struct {
	ctx context.Context
	p   Prompter

	mu   sync.Mutex
	keys int
}

// mark is err as the caller's own, or nil.
func (w *wizardPrompter) mark(err error) error {
	if err == nil {
		return nil
	}
	return &ownError{err}
}

// secrets is how many secrets were asked.
func (w *wizardPrompter) secrets() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.keys
}

func (w *wizardPrompter) Select(title string, choices []wizard.Choice, def int) (int, error) {
	if len(choices) == 1 {
		return 0, nil
	}
	cs := make([]Choice, len(choices))
	for i, c := range choices {
		cs[i] = Choice{Label: c.Label, Detail: c.Detail}
	}
	i, err := w.p.Choose(title, cs, def)
	return i, w.mark(err)
}

func (w *wizardPrompter) MultiSelect(title string, _ []wizard.Choice, _ []int) ([]int, error) {
	return nil, fmt.Errorf("provider: a sign-in asks no multiple choice (%q)", title)
}

func (w *wizardPrompter) Confirm(question string, def bool) (bool, error) {
	ok, err := w.p.Confirm(question, def)
	return ok, w.mark(err)
}

func (w *wizardPrompter) Input(prompt, def string) (string, error) {
	return w.InputContext(w.ctx, prompt, def)
}

func (w *wizardPrompter) InputContext(ctx context.Context, prompt, def string) (string, error) {
	s, err := w.p.Input(ctx, prompt)
	switch {
	case err != nil && ctx.Err() != nil:
		return "", ctx.Err() // given up, as the wizard asked
	case err != nil:
		return "", w.mark(err)
	case s == "":
		return def, nil
	}
	return s, nil
}

func (w *wizardPrompter) Secret(prompt string) (string, error) {
	w.mu.Lock()
	w.keys++
	w.mu.Unlock()
	s, err := w.p.Secret(prompt)
	return s, w.mark(err)
}

func (w *wizardPrompter) Notify(_ wizard.Level, format string, args ...any) {
	w.p.Notify(format, args...)
}

var (
	_ wizard.Prompter       = (*wizardPrompter)(nil)
	_ wizard.InputerContext = (*wizardPrompter)(nil)
)
