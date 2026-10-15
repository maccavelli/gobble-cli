package acpserver

import (
	"context"
	"fmt"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// AuthProvider is a provider ACP authentication names: its id, which is
// its agent method's; its label; the gobble command that stores its
// credential; and whether it has a login rather than a key alone
// (0016-MADR A2).
type AuthProvider struct {
	ID, Name, Command string
	Login             bool
}

// AuthOptions are what ACP authentication offers. Without Providers, it
// offers nothing.
type AuthOptions struct {
	// Providers are the providers that take a credential, in order.
	Providers func() []AuthProvider
	// Has reports whether provider id has a credential now, from any source
	// in gobble's order.
	Has func(id string) bool
}

// loginSuffix ends a terminal method's id: <provider>-login.
const loginSuffix = "-login"

func (a *Agent) authProviders() []AuthProvider {
	if a.auth.Providers == nil {
		return nil
	}
	return a.auth.Providers()
}

// authMethods are initialize's authMethods: an agent method per provider,
// then, for a client that offers a terminal, a terminal method per
// provider, which the client runs as `gobble acp --login <provider>`
// (0016-MADR A2). None is an empty list, never null.
func (a *Agent) authMethods(terminal bool) []acp.AuthMethod {
	ps := a.authProviders()
	methods := []acp.AuthMethod{}
	for _, p := range ps {
		m := acp.NewAuthMethodAgent(acp.AuthMethodId(p.ID), p.Name)
		m.Agent.Description = new("Uses the credential gobble has for " + p.Name + "; store one with " + p.Command)
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
		m := acp.NewAuthMethodTerminal(acp.AuthMethodId(p.ID+loginSuffix), name)
		m.Terminal.Args = []string{"--login", p.ID}
		methods = append(methods, m)
	}
	return methods
}

// Authenticate answers an agent method: {} when its provider has a
// credential, else auth_required naming the provider and the command that
// stores one. No credential crosses ACP. An unknown id is -32602; the SDK
// refuses a terminal method before this runs (0016-MADR A2).
func (a *Agent) Authenticate(_ context.Context, p acp.AuthenticateRequest) (acp.AuthenticateResponse, error) {
	for _, ap := range a.authProviders() {
		if string(p.MethodId) != ap.ID {
			continue
		}
		if a.auth.Has != nil && a.auth.Has(ap.ID) {
			return acp.AuthenticateResponse{}, nil
		}
		return acp.AuthenticateResponse{}, acp.NewAuthRequired(map[string]any{"provider": ap.ID, keyReason: "no credential", "command": ap.Command})
	}
	return acp.AuthenticateResponse{}, acp.NewInvalidParams(map[string]any{keyReason: fmt.Sprintf("unknown auth method %q", p.MethodId)})
}
