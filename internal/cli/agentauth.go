package cli

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// startSignInWith starts an ACP client's sign-in through the SDK's engine; a
// test replaces it (0051-PLAN P6 step 1a).
var startSignInWith = provider.StartSignIn

// authOptions are what ACP authentication offers: the providers and their
// credentials as gobble auth check finds them (0016-MADR A2), the offered
// sign-in methods, and the sign-in itself, which stores what it gives as gobble
// auth login does (0051-MADR D4).
func (e *runEnv) authOptions() acpserver.AuthOptions {
	return acpserver.AuthOptions{
		Providers: authProviders,
		Has: func(id string) bool {
			_, ok := e.credential(id)
			return ok
		},
		Methods: provider.Offered,
		Holds:   e.holds,
		Remote:  provider.Remote,
		Command: func(id string, m authcontract.Method) string { return provider.MethodCommand(id, m.ID) },
		SignIn:  e.startSignIn,
		Signed:  e.refreshCredentials,
	}
}

// holds reports whether the credential gobble has for provider id is what
// method m yields: a key for a key method; for a session method, a stored
// session of the kind m signs in to, so Grok's browser and device sign-ins
// count as one (0051-MADR D4 step 2).
func (e *runEnv) holds(id string, m authcontract.Method) bool {
	c, ok := e.credential(id)
	if !ok {
		return false
	}
	if !c.Login {
		return m.Yields == authcontract.YieldsAPIKey || m.Yields == authcontract.YieldsDeviceKey
	}
	l, err := auth.Logins{Path: authPath()}.Load(context.Background(), id)
	if err != nil || l == nil {
		return false
	}
	sm, ok := provider.SignedInMethod(id, l)
	return ok && sm.Yields == m.Yields && sm.ClientIdentity == m.ClientIdentity
}

// startSignIn is authenticate's SignIn: provider.StartSignIn over auth.json,
// with the host id and the issued clients (a failure to read the id is one
// warning in the agent's log, and the sign-in goes on without it), and the key
// a device sign-in yields stored through auth.SetKey, as signInTo stores it.
func (e *runEnv) startSignIn(ctx context.Context, id, method string, remote bool) (acpserver.SignInRun, error) {
	path := authPath()
	if path == "" {
		return nil, errors.New("auth.json: the config directory cannot be resolved")
	}
	store, hostID, warn := hostSignIn(path)
	if warn != nil {
		e.log().WarnContext(ctx, "sign-in", slog.String("message", warn.Error()))
	}
	o := provider.SignInOptions{Options: e.providerOptions(id, provider.Options{ClientName: appName, ClientVersion: identity().Version}), HostID: hostID, Remote: remote}
	run, err := startSignInWith(ctx, id, method, o, store)
	if err != nil {
		return nil, err
	}
	return &keyRun{SignInRun: run, id: id, path: path}, nil
}

// keyRun is a sign-in run (a *provider.SignInRun) whose Wait also stores the key a device
// sign-in yields, escaped so Pi's grammar gives it back unchanged, and
// returns it to no one: the result a client's authenticate sees holds no key.
type keyRun struct {
	acpserver.SignInRun
	id, path string

	once sync.Once
	res  provider.SignInResult
	err  error
}

func (r *keyRun) Wait(ctx context.Context) (provider.SignInResult, error) {
	r.once.Do(func() {
		r.res, r.err = r.SignInRun.Wait(ctx)
		if r.err != nil || r.res.Key == "" {
			return
		}
		key := catalog.QuoteValue(r.res.Key)
		r.res.Key = ""
		// A key is stored only for a sign-in still wanted: the wizard saves
		// nothing once its context is done (0051-MADR D4 step 3).
		if err := ctx.Err(); err != nil {
			r.res, r.err = provider.SignInResult{}, provider.NewSignInError(r.id, err)
			return
		}
		if err := auth.SetKey(ctx, r.path, r.id, key, ""); err != nil {
			r.res, r.err = provider.SignInResult{}, provider.NewSignInError(r.id, err)
		}
	})
	return r.res, r.err
}
