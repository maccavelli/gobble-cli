package provider

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/llmprovider"
	"github.com/maccavelli/go-llmprovider-sdk/llmprovider/providers"
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

// SignInOptions configure SignIn. The embedded Options carry the listing
// that checks the credential: its client name and version, its HTTP client,
// headers and base URL; HTTPClient also carries the sign-in's own requests.
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
}

// Credential is what a sign-in gives: an API key to store, or none when it
// saved a login in the store.
type Credential struct {
	Key string
	// Device reports that Key came from a device sign-in, Kilo's.
	Device bool
	// Workspace is the Anthropic workspace the wizard asked for, for a
	// Claude key not scoped to one (0035-MADR D5).
	Workspace string
}

// offers reports whether gobble offers the SDK's method m for provider id:
// a key everywhere; a browser sign-in where gobble keeps the login (Grok's);
// a device sign-in there, and Kilo's, which gives a key; and OpenAI's
// ChatGPT plan sign-in. OpenAI's browser and device sign-ins are the Codex
// client's, which sign in under another app's name, and a token paste or a
// vendor CLI's login is not gobble's (0031-MADR D7, amended 2026-11-10).
func offers(id string, m llmprovider.AuthMethodID) bool {
	openai := id == string(llmprovider.ProviderOpenAI)
	switch m {
	case llmprovider.AuthAPIKey:
		return true
	case llmprovider.AuthBrowserOAuth:
		return HasLogin(id) && !openai
	case llmprovider.AuthDeviceCode:
		return HasLogin(id) && !openai || HasKeyLogin(id)
	case llmprovider.AuthChatGPTPlan:
		return openai
	}
	return false
}

// Methods are provider id's ways to sign in, as --method names them, in the
// SDK's order; none for a provider that takes no key.
func Methods(id string) []string {
	d, ok := keyProvider(id)
	if !ok {
		return nil
	}
	if len(d.AuthMethods) == 0 {
		return []string{MethodAPIKey}
	}
	var out []string
	for _, m := range d.AuthMethods {
		for word, sdk := range methodIDs {
			if sdk == m.ID && offers(id, m.ID) {
				out = append(out, word)
			}
		}
	}
	return out
}

// skips are the SDK's methods SignIn leaves out of provider id's menu: those
// gobble does not offer, and, when method is forced, every other one. The
// option applies to every provider of a call, so it is computed for id
// alone (0031-PLAN deviation 5).
func skips(d llmprovider.Descriptor, method string) []llmprovider.AuthMethodID {
	var out []llmprovider.AuthMethodID
	for _, m := range d.AuthMethods {
		if !offers(string(d.ID), m.ID) || method != "" && m.ID != methodIDs[method] {
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
	if e.Keys > 0 {
		return fmt.Sprintf("%s refused %d keys; none was saved", e.ID, e.Keys)
	}
	return fmt.Sprintf("%s sign-in failed (%s)", e.ID, FailureKind(e.err))
}

func (e *SignInError) Unwrap() error { return e.err }

// SignIn signs in to provider id through go-llmprovider-sdk's wizard
// (0031-MADR D7): its methods, as gobble offers them, asked through p; a
// session saved in store; a typed key checked by a listing and asked for
// again while refused. gobble's own provider menu chooses id. An error of
// p's, ShowDeviceCode's or OpenURL's is returned as it is; any other is a
// *SignInError.
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
	listing := append([]llmprovider.Option{
		llmprovider.WithClientInfo(o.ClientName, o.ClientVersion),
		llmprovider.WithoutModelMetadata(),
	}, o.sdkOptions()...)
	res, _, err := wizard.ResolveCredential(ctx, w, wizard.Options{
		Registry:         providers.Default(),
		Existing:         wizard.Result{Provider: d.ID, BaseURL: o.BaseURL, Workspace: o.Workspace},
		Discover:         true,
		ProviderOptions:  listing,
		TokenStore:       sdkStore{store},
		HTTPClient:       o.HTTPClient,
		OpenURL:          open,
		KiloDeviceKey:    true,
		ShowDeviceCode:   show,
		SkipMethods:      skips(d, o.Method),
		NoEndpointPrompt: true,
		NoPastePrompt:    o.NoPaste,
		AgentName:        AgentName,
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
		return Credential{}, nil
	}
	return Credential{Key: res.APIKey, Device: device, Workspace: res.Workspace}, nil
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
