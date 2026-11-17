package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/maccavelli/go-tui-lib/v2/inline"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/llm/catalog"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/personality"
)

// signInWith signs in through the SDK's wizard; a test replaces it
// (0031-MADR D7).
var signInWith = provider.SignIn

// hostSignIn is what a sign-in on this host takes of the state directory
// (0051-PLAN P6 step 1a): the login store, with the issued clients beside it
// in <state>/clients.json, and the host id from <state>/host-id, created
// owner-only on first use. When the state directory cannot be resolved or the
// id cannot be read or made, the sign-in goes on with no host id and the
// logins alone, and warn says why, naming the error and never an id: the
// owner's ruling "Proceed, warn once", so no sign-in of grok, kilo or the
// plan fails over a file it may not use.
func hostSignIn(authFile string) (store provider.LoginStore, hostID string, warn error) {
	logins := auth.Logins{Path: authFile}
	dirs, _, err := systemDirs()
	if err != nil {
		return logins, "", fmt.Errorf("no host id: %w; the sign-in goes on without one", err)
	}
	store = auth.ClientLogins{Logins: logins, Clients: auth.Clients{Path: filepath.Join(dirs.State, "clients.json")}}
	hostID, err = auth.HostID(dirs.State)
	if err != nil {
		return store, "", fmt.Errorf("no host id: %w; the sign-in goes on without one", err)
	}
	return store, hostID, nil
}

// methodOf is --method's word as one of provider id's methods; another is
// a usage error naming them.
func methodOf(id, word string) (string, error) {
	ms := provider.Methods(id)
	if slices.Contains(ms, word) {
		return word, nil
	}
	return "", usageErrorf("%s signs in with --method %s, not %q", id, strings.Join(ms, " or "), word)
}

// signInHint is how the user signs in to provider id where they are: /login
// in the line session, gobble auth login elsewhere (0025-MADR D1, D6).
func (e *runEnv) signInHint(id string) string {
	if e.inLine {
		return "/login " + id
	}
	return "gobble auth login " + id
}

// signIn is /login's flow (0025-MADR D6; 0031-MADR D7): want, or a provider
// chosen from those that take a key, each with its credential's source;
// then the SDK's wizard, through q, asks how and signs in. It returns the
// provider signed in to. The sign-in opens a browser sign-in's URL in the
// host's browser and shows a device code with its expiry, as gobble auth
// login does (0051-MADR D3; 0051-PLAN P7): /login, gobble setup, the first
// run and gobble acp --login all come through here.
func (e *runEnv) signIn(ctx context.Context, q *questions, want string) (string, error) {
	id := want
	if id == "" {
		ids := provider.KeyProviders()
		opts := make([]option, len(ids))
		for i, p := range ids {
			opts[i] = option{provider.Label(p) + " (" + p + ")", e.credentialWords(p)}
		}
		i, err := q.choose("Sign in to which provider?", opts, 0)
		if err != nil {
			return "", err
		}
		id = ids[i]
	}
	o := provider.SignInOptions{ShowDeviceCode: e.showDeviceCode, OpenURL: e.openLoginURL}
	return id, e.signInTo(ctx, prompter{q}, id, o, nil)
}

// terminalQuestions are the questions a sign-in asks on e's terminal, read
// through the reader that owns stdin, a secret in raw mode; the caller has
// checked that there is a terminal (onTerminal).
func (e *runEnv) terminalQuestions(ctx context.Context) *questions {
	return &questions{ctx: ctx, in: inline.NewReader(e.stdin), out: e.out.err, raw: func() (func() error, error) { return editor.Raw(e.stdin) }}
}

// onTerminal reports whether e can ask a question: a terminal on stdin.
func (e *runEnv) onTerminal() bool { return e.out.caps.In && e.stdin != nil }

// signInTo signs in to provider id through the SDK's wizard, asking p, with
// o's method and callbacks (0031-MADR D7), and stores what it gives: a key
// in auth.json, a device's escaped so Pi's grammar gives it back unchanged;
// a login, which the wizard saved there through the login store. js, when
// not nil, takes the done step.
func (e *runEnv) signInTo(ctx context.Context, p provider.Prompter, id string, o provider.SignInOptions, js *jsonSteps) error {
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	o.Options = e.providerOptions(id, provider.Options{ClientName: appName, ClientVersion: identity().Version})
	store, hostID, warn := hostSignIn(path)
	if warn != nil {
		e.out.Warnf("%s", warn)
	}
	o.HostID = hostID
	c, err := signInWith(ctx, id, p, o, store)
	if err != nil {
		return err
	}
	if js != nil && c.Yields != "" {
		js.yields = c.Yields // what the SDK's done step says the sign-in left
	}
	switch {
	case c.Key == "":
		return e.saved(js, id, auth.TypeOAuth)
	case c.Device:
		c.Key = catalog.QuoteValue(c.Key)
	case provider.PublicListing(id):
		p.Notify("%s lists its models without a key, so the key was not checked.", provider.Label(id))
	}
	if err := auth.SetKey(ctx, path, id, c.Key, c.Workspace); err != nil {
		return failf("%v", err)
	}
	return e.saved(js, id, auth.TypeAPIKey)
}

// storedCredentials are the key providers auth.json holds a credential
// for, in the registry's order, each with its kind in words.
func storedCredentials() ([]string, map[string]string, error) {
	path := authPath()
	if path == "" {
		return nil, nil, failf("auth.json: the config directory cannot be resolved")
	}
	f, err := auth.Load(path)
	if err != nil {
		return nil, nil, failf("%v", err)
	}
	var ids []string
	kinds := map[string]string{}
	for _, id := range provider.KeyProviders() {
		if en, ok := f.Entry(id); ok {
			ids = append(ids, id)
			kinds[id] = "API key"
			if en.Type == auth.TypeOAuth {
				kinds[id] = "sign-in"
			}
		}
	}
	return ids, kinds, nil
}

// signOut is /logout's flow (0025-MADR D6): want, or a provider chosen from
// those auth.json holds a credential for, removed as gobble auth logout
// removes it, a Grok session revoked; the credentials are read again. It
// says what it did, and that a provider's variable, when set, still gives
// it a credential.
func (e *runEnv) signOut(ctx context.Context, q *questions, want string) (string, error) {
	id := want
	if id == "" {
		ids, kinds, err := storedCredentials()
		if err != nil {
			return "", err
		}
		if len(ids) == 0 {
			return "No stored credentials to remove: /logout removes what /login and gobble auth login saved.", nil
		}
		opts := make([]option, len(ids))
		for i, p := range ids {
			opts[i] = option{provider.Label(p) + " (" + p + ")", kinds[p]}
		}
		i, err := q.choose("Sign out of which provider?", opts, 0)
		if err != nil {
			return "", err
		}
		id = ids[i]
	}
	said, err := e.logout(ctx, id)
	if err != nil {
		return "", err
	}
	if said == "" {
		return fmt.Sprintf("No stored credential for %s to remove: /logout removes what /login and gobble auth login saved.", id), nil
	}
	e.refreshCredentials(id)
	if v := provider.CredentialVar(id); v != "" {
		if _, set := os.LookupEnv(v); set {
			said += fmt.Sprintf(" %s is still set, so %s keeps a credential.", v, id)
		}
	}
	return said, nil
}

// Bracketed paste's switches: off while a question reads, so a pasted key
// or URL arrives as typed text (0025-PLAN deviation 7).
const (
	pasteOff = "\x1b[?2004l"
	pasteOn  = "\x1b[?2004h"
)

// ask runs f with the line session's questions: through the reader that
// owns stdin, with bracketed paste off, and with the first Ctrl+C ending f
// rather than gobble. A Ctrl+C is errCancelled.
func (ls *lineSession) ask(f func(context.Context, *questions) error) error {
	e := ls.e
	ctx, cancel := context.WithCancel(e.ctx)
	defer cancel()
	if e.intr != nil {
		defer e.intr.during(cancel)()
	}
	if ls.pasting && e.out.caps.Out.TTY {
		io.WriteString(e.out.out, pasteOff)      //nolint:errcheck,gosec // paste mode is best effort
		defer io.WriteString(e.out.out, pasteOn) //nolint:errcheck // paste mode is best effort
	}
	err := f(ctx, &questions{ctx: ctx, in: ls.keys, out: e.out.err, raw: ls.raw})
	if err != nil && ctx.Err() != nil && e.ctx.Err() == nil {
		err = errCancelled
	}
	return err
}

// login is /login [provider] in the line session (0025-MADR D6): the
// sign-in; then the credentials read again, the provider listed, and its
// models shown by the agent's /models, numbered for /model N.
func (ls *lineSession) login(arg string, turn func(Prompt) error) error {
	e := ls.e
	want := ""
	if arg = strings.TrimSpace(arg); arg != "" {
		id, err := keyProvider(arg)
		if err != nil {
			e.out.Errorf("%v", err)
			return nil
		}
		want = id
	}
	var id string
	err := ls.ask(func(ctx context.Context, q *questions) (err error) {
		id, err = e.signIn(ctx, q, want)
		return err
	})
	switch {
	case errors.Is(err, errCancelled):
		e.out.Notef("Sign-in cancelled; nothing was saved.")
		return nil
	case err != nil:
		e.out.Errorf("%v", err)
		return nil
	}
	e.refreshCredentials(id)
	e.modelListing(provider.Selection{ID: id}).get()
	return turn(Prompt{Text: "/models " + id})
}

// logout is /logout [provider] in the line session (0025-MADR D6).
func (ls *lineSession) logout(arg string) error {
	e := ls.e
	want := ""
	if arg = strings.TrimSpace(arg); arg != "" {
		id, err := keyProvider(arg)
		if err != nil {
			e.out.Errorf("%v", err)
			return nil
		}
		want = id
	}
	var said string
	err := ls.ask(func(ctx context.Context, q *questions) (err error) {
		said, err = e.signOut(ctx, q, want)
		return err
	})
	switch {
	case errors.Is(err, errCancelled):
		e.out.Notef("Sign-out cancelled.")
		return nil
	case err != nil:
		e.out.Errorf("%v", err)
		return nil
	}
	return e.out.Resultf("%s\n", said)
}

// firstRunQuestion is the line session's first question when no provider
// has a credential (0025-MADR D7).
const firstRunQuestion = "No provider has a credential yet. Sign in to one now?"

// firstRun is the line session's offer when no provider has a credential
// (0025-MADR D7): /login's sign-in, before the first prompt. Declined, the
// session goes on as before, with the warning naming the variables.
func (ls *lineSession) firstRun() error {
	e := ls.e
	var yes bool
	err := ls.ask(func(_ context.Context, q *questions) (err error) {
		yes, err = q.confirm(firstRunQuestion, true)
		return err
	})
	if err != nil && !errors.Is(err, errCancelled) {
		return err
	}
	if !yes {
		e.out.Lead(ls.spoken(), personality.LeadNoCredential)
		e.out.Warnf("%s", e.noCredentialMessage())
		return nil
	}
	return ls.login("", ls.turn)
}

// SetupCmd is gobble setup: the sign-in the first run offers, at any time
// (0025-MADR D7).
type SetupCmd struct{}

// Run asks which provider to sign in to and how, on the terminal.
func (c *SetupCmd) Run(e *runEnv) error {
	e.ensureSettingsFile()
	if !e.onTerminal() {
		return usageErrorf("gobble setup asks its questions on a terminal; gobble auth login <provider> --method signs in without one")
	}
	id, err := e.signIn(e.ctx, e.terminalQuestions(e.ctx), "")
	switch {
	case errors.Is(err, errCancelled):
		return failf("setup cancelled; nothing was saved")
	case err != nil:
		return err
	}
	return e.out.Resultf("gobble models list --provider %s lists its models; /model in a session switches to one.\n", id)
}
