package acpserver

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"log/slog"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// AuthOptions are what ACP authentication offers. Without Providers, it
// offers nothing.
type AuthOptions struct {
	// Providers are the providers that take a credential, in order.
	Providers func() []acpext.AuthProvider
	// Has reports whether provider id has a credential now, from any source
	// in gobble's order.
	Has func(id string) bool
	// Methods are provider id's offered sign-in methods, which
	// _meta.gobble.methods advertises and authenticate drives
	// (0051-MADR D4). Nil offers none, so authenticate signs in to nothing.
	Methods func(id string) []authcontract.Method
	// Holds reports whether the credential gobble has for provider id is
	// what method m yields: a key for a key method, a session of the kind m
	// signs in to for a session method (0051-MADR D4 step 2).
	Holds func(id string, m authcontract.Method) bool
	// Remote reports whether method m can be completed by a person who is not
	// on the agent's host (provider.Remote). Nil accepts no method for a
	// remote sign-in.
	Remote func(m authcontract.Method) bool
	// Command is the host command that signs in to provider id with method
	// m, which a refused remote sign-in names (provider.MethodCommand).
	Command func(id string, m authcontract.Method) string
	// SignIn starts provider id's sign-in by the method whose contract id is
	// method; remote says that the person is not on the agent's host
	// (provider.StartSignIn). Nil means authenticate signs in to nothing.
	SignIn func(ctx context.Context, id, method string, remote bool) (SignInRun, error)
	// Signed is told that provider id's credential changed by a sign-in, to
	// drop the run's shared session and move its credential generation. Nil
	// is told nothing.
	Signed func(id string)
}

// SignInRun is a sign-in in progress, as provider.StartSignIn gives it:
// acpserver names the contract's types and gobble's own result and no type
// of the SDK's x/signin (archtest rule 4). Wait saves what the sign-in
// yields and returns the result; Cancel ends the sign-in.
type SignInRun interface {
	Method() authcontract.Method
	Steps() <-chan authcontract.Step
	Answer(ctx context.Context, a authcontract.Answer) error
	Wait(ctx context.Context) (provider.SignInResult, error)
	Cancel()
}

func (a *Agent) authProviders() []acpext.AuthProvider {
	if a.auth.Providers == nil {
		return nil
	}
	return a.auth.Providers()
}

// offered is provider id's offered sign-in methods.
func (a *Agent) offered(id string) []authcontract.Method {
	if a.auth.Methods == nil {
		return nil
	}
	return a.auth.Methods(id)
}

// authMethods are initialize's authMethods: an agent method per provider,
// whose _meta.gobble.methods are the provider's offered sign-in methods, then,
// for a client that offers a terminal, a terminal method per provider, which
// the client runs as `gobble acp --login <provider>` (0016-MADR A2; 0051-MADR
// D4). None is an empty list, never null.
func (a *Agent) authMethods(terminal bool) []acp.AuthMethod {
	ps := a.authProviders()
	methods := []acp.AuthMethod{}
	for _, p := range ps {
		m := acp.NewAuthMethodAgent(acp.AuthMethodId(p.ID), p.Name)
		m.Agent.Description = new("Uses the credential gobble has for " + p.Name + "; store one with " + p.Command)
		if ms := a.offered(p.ID); len(ms) > 0 {
			// A struct of strings and the contract's own types cannot fail to
			// encode under jsonx's options; were it to, the method goes
			// without the member and the client reads no facts.
			if meta, err := jsonx.SetMember(nil, gobble, map[string][]authcontract.Method{acpext.MetaMethods: ms}); err == nil {
				m.Agent.Meta = meta
			} else {
				a.logger.Warn("auth: methods not advertised", slog.String("provider", p.ID), slog.String("error", err.Error()))
			}
		}
		methods = append(methods, m)
	}
	if !terminal {
		return methods
	}
	for _, p := range ps {
		name := "Store an API key for " + p.Name
		if p.Login {
			name = "Sign in to " + p.Name
		}
		m := acp.NewAuthMethodTerminal(acp.AuthMethodId(p.ID+acpext.LoginSuffix), name)
		m.Terminal.Args = []string{"--login", p.ID}
		methods = append(methods, m)
	}
	return methods
}

// authMeta is _meta.gobble of an authenticate request, decoded strictly: a
// member gobble does not know, or one of the wrong type, is an error, and the
// request is -32602 (0051-MADR D4 step 1). Absent is the zero value.
func authMeta(meta map[string]any) (acpext.AuthenticateMeta, error) {
	var m acpext.AuthenticateMeta
	raw, ok := meta[gobble]
	if !ok {
		return m, nil
	}
	// A null is no value: method, fresh and remote each take a value or
	// absence, so a null of any is refused, which the decoder alone reads as
	// absent. A null _meta.gobble itself is an absent one, as ACP reads a null
	// container (the owner's ruling of 2026-11-16, "Refuse `method: null`; read
	// `gobble: null` as absent").
	if obj, isObj := raw.(map[string]any); isObj {
		for _, k := range []string{"method", "fresh", "remote"} {
			if v, present := obj[k]; present && v == nil {
				return m, fmt.Errorf("%s is null: want true or false", k)
			}
		}
	}
	b, err := jsonx.Marshal(raw)
	if err != nil {
		return m, err
	}
	err = jsonx.Unmarshal(b, &m, json.RejectUnknownMembers(true))
	return m, err
}

// noCredential is auth_required as it has always been: the provider has no
// credential, and the command that stores one.
func noCredential(ap acpext.AuthProvider, reason string) error {
	return acp.NewAuthRequired(errData(acpext.ErrorData{Provider: ap.ID, Reason: reason, Command: ap.Command}))
}

// authInvalid is -32602 with a reason.
func authInvalid(format string, args ...any) error {
	return acp.NewInvalidParams(errData(acpext.ErrorData{Reason: fmt.Sprintf(format, args...)}))
}

// Authenticate answers an agent method (0016-MADR A2, amended by 0051-MADR
// D4): {} when its provider has a credential that _meta.gobble does not ask
// to replace; otherwise, for a client that offers elicitation.form, a
// sign-in, relayed as elicitations, whose result {} follows once the
// credential is saved; otherwise auth_required naming the provider and the
// command that stores one. No credential crosses ACP but the one a person
// types into a form, which the sign-in alone receives. An unknown id is
// -32602; the SDK refuses a terminal method before this runs.
func (a *Agent) Authenticate(ctx context.Context, p acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	for _, ap := range a.authProviders() {
		if string(p.MethodId) == ap.ID {
			return acp.AuthenticateResponse{}, a.authenticateProvider(ctx, ap, p.Meta)
		}
	}
	return acp.AuthenticateResponse{}, authInvalid("unknown auth method %q", string(p.MethodId))
}

// authenticateProvider is Authenticate for provider ap, by MADR D4's four cases.
func (a *Agent) authenticateProvider(ctx context.Context, ap acpext.AuthProvider, rawMeta map[string]any) error {
	meta, err := authMeta(rawMeta)
	if err != nil {
		return authInvalid("_meta.%s is not an authenticate request's members: %v", gobble, err)
	}
	offered := a.offered(ap.ID)
	var named *authcontract.Method
	if meta.Method != "" {
		for i := range offered {
			if offered[i].ID == meta.Method {
				named = &offered[i]
			}
		}
		if named == nil {
			return authInvalid("%s has no sign-in method %q", ap.ID, meta.Method)
		}
	}
	// 1. A named method that gobble drives and that a remote person cannot
	// complete is refused before anything starts. A method gobble does not
	// drive over ACP gets the answer a local client gets, whoever asks: {} when
	// the credential is held, else step 3's (the owner's ruling of 2026-11-16,
	// "The same answer as a local client").
	if named != nil && meta.Remote && drivable(*named, true) && !a.remoteOK(*named) {
		return a.refuseRemote(ap, *named)
	}
	// 2. A credential is present, and nothing asks for another.
	has := a.auth.Has != nil && a.auth.Has(ap.ID)
	if has && !meta.Fresh && (named == nil || a.auth.Holds != nil && a.auth.Holds(ap.ID, *named)) {
		return nil
	}
	// 3. A client that elicits: sign in.
	caps := a.elicitation()
	if a.auth.SignIn == nil || caps == nil || caps.Form == nil {
		reason := "no credential"
		if has {
			reason = "signing in again needs a client that offers elicitation.form"
		}
		return noCredential(ap, reason)
	}
	// One sign-in per provider: the provider is reserved before the client is
	// asked anything, and released on every path out.
	release, err := a.reserve(ap)
	if err != nil {
		return err
	}
	defer release()
	m, err := a.chooseMethod(ctx, ap, offered, named, meta.Remote)
	if err != nil {
		return err
	}
	a.reserved(ap, m)
	return a.signIn(ctx, ap, m, meta.Remote, *caps)
}

// remoteOK reports whether a remote person can complete method m.
func (a *Agent) remoteOK(m authcontract.Method) bool { return a.auth.Remote != nil && a.auth.Remote(m) }

// refuseRemote answers a named method that completes only on the agent's host
// to a remote client: auth_required of kind unsupported, naming the host
// command that runs it (0051-MADR D4 step 1).
func (a *Agent) refuseRemote(ap acpext.AuthProvider, m authcontract.Method) error {
	cmd := ap.Command
	if a.auth.Command != nil {
		cmd = a.auth.Command(ap.ID, m)
	}
	return acp.NewAuthRequired(errData(acpext.ErrorData{
		Provider: ap.ID, Kind: acpext.ErrorKind(authcontract.FailureUnsupported), Command: cmd,
		Reason: m.Label + " completes only in a browser on the machine gobble runs on, so it cannot be done from here",
	}))
}

// elicitation is what the client offers of elicitation, or nil.
func (a *Agent) elicitation() *acp.ElicitationCapabilities {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.client.Elicitation
}

// drivable reports whether authenticate signs in by method m: a device sign-in
// or a browser sign-in a paste can complete; an API key is never asked over
// ACP, and a browser sign-in only the host's loopback completes only when
// named (0051-MADR D4 step 3).
func drivable(m authcontract.Method, named bool) bool {
	switch m.Completion {
	case authcontract.CompletionDevice, authcontract.CompletionBrowserPaste:
		return true
	case authcontract.CompletionBrowserLoopback:
		return named
	}
	return false
}

// chooseMethod is the method to sign in by: the named one, when gobble drives
// it; with none named, the only method that is driven (and, for a remote
// client, that a remote person can complete), else the one the client chooses
// by an enum of those methods' labels, in the offered order (0051-MADR D4
// steps 1 and 3; the owner's ruling of 2026-11-16, "Ask the client to
// choose").
func (a *Agent) chooseMethod(ctx context.Context, ap acpext.AuthProvider, offered []authcontract.Method, named *authcontract.Method, remote bool) (authcontract.Method, error) {
	if named != nil {
		if !drivable(*named, true) {
			return authcontract.Method{}, noCredential(ap, named.Label+" is not a sign-in gobble drives from an ACP client")
		}
		return *named, nil
	}
	var cands []authcontract.Method
	for _, m := range offered {
		if drivable(m, false) && (!remote || a.remoteOK(m)) {
			cands = append(cands, m)
		}
	}
	switch {
	case len(cands) == 0:
		return authcontract.Method{}, noCredential(ap, "no sign-in gobble drives from an ACP client")
	case len(cands) == 1:
		return cands[0], nil
	}
	return a.askMethod(ctx, ap, cands)
}

// _ keeps the compiler's check that SignInRun's methods are provider's.
var _ SignInRun = (*provider.SignInRun)(nil)
