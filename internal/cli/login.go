package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/personality"
)

// The ways to sign in, as --method names them (0025-MADR D6).
const (
	methodAPIKey  = "api-key"
	methodBrowser = "browser"
	methodDevice  = "device"
)

// maxKeys is how many typed keys a sign-in tries before it stops on a
// refusal, as go-llmprovider-sdk's wizard does.
const maxKeys = 3

// signInMethod is one way to sign in to a provider; id is --method's word.
type signInMethod struct{ id, label string }

// signInMethods are provider id's ways to sign in, gobble's own (0025-MADR
// D6, deviation 6): an API key; Grok's browser and device sign-in; Kilo's
// device sign-in, which gives a key.
func signInMethods(id string) []signInMethod {
	label := provider.Label(id)
	ms := []signInMethod{{methodAPIKey, "Paste a " + label + " API key"}}
	switch {
	case provider.HasLogin(id):
		ms = append(ms, signInMethod{methodBrowser, "Sign in to " + label + " in a browser"},
			signInMethod{methodDevice, "Sign in to " + label + " with a code, on any device"})
	case provider.HasKeyLogin(id):
		ms = append(ms, signInMethod{methodDevice, "Sign in to " + label + " with a code, on any device; it gives a key"})
	}
	return ms
}

// methodOf is --method's word as one of provider id's methods; another is
// a usage error naming them.
func methodOf(id, word string) (string, error) {
	var ids []string
	for _, m := range signInMethods(id) {
		if m.id == word {
			return word, nil
		}
		ids = append(ids, m.id)
	}
	return "", usageErrorf("%s signs in with --method %s, not %q", id, strings.Join(ids, " or "), word)
}

// signInHint is how the user signs in to provider id where they are: /login
// in the line session, gobble auth login elsewhere (0025-MADR D1, D6).
func (e *runEnv) signInHint(id string) string {
	if e.inLine {
		return "/login " + id
	}
	return "gobble auth login " + id
}

// signIn is /login's flow (0025-MADR D6, deviation 6): want, or a provider
// chosen from those that take a key, each with its credential's source;
// then one of its methods; then its credential, through gobble's sign-ins.
// It returns the provider signed in to.
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
	ms := signInMethods(id)
	opts := make([]option, len(ms))
	for i, m := range ms {
		opts[i] = option{m.label, ""}
	}
	i, err := q.choose("How do you sign in to "+provider.Label(id)+"?", opts, 0)
	if err != nil {
		return "", err
	}
	if ms[i].id == methodAPIKey {
		return id, e.typedKey(ctx, q, id)
	}
	return id, e.loginWith(ctx, id, ms[i].id == methodDevice, nil, func(ctx context.Context) (string, error) {
		return (&questions{ctx: ctx, in: q.in, out: q.out}).line()
	})
}

// typedKey asks for provider id's API key and checks it with a listing,
// which spends no generation: a key the provider refuses is asked for
// again, up to maxKeys; one the listing cannot check is saved with a note.
// The key is stored in auth.json as gobble auth login --api-key - stores it.
func (e *runEnv) typedKey(ctx context.Context, q *questions, id string) error {
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	label := provider.Label(id)
	for keys := 1; ; keys++ {
		key, err := q.secret(label + " API key")
		if err != nil {
			return err
		}
		if key == "" {
			return errCancelled
		}
		r, err := listModels(ctx, id, key, e.providerOptions(id, provider.Options{ClientName: appName, ClientVersion: identity().Version}))
		if err != nil {
			r = provider.Listing{Err: err}
		}
		// A refused or failed listing is r.Err, as provider.List gives it.
		state, reason, _ := listingState(id, r, time.Now())
		switch {
		case state == acpserver.StateKeyRejected && keys == maxKeys:
			return failf("%s refused %d keys; none was saved", label, keys)
		case state == acpserver.StateKeyRejected:
			q.notify("%s refused the key%s. Enter another, or an empty line to stop.", label, prefixed(statusOf(r.Err)))
			continue
		case state == acpserver.StateNotChecked:
			q.notify("%s lists its models without a key, so the key was not checked.", label)
		case !r.Live:
			q.notify("The key could not be checked: %s; it is saved all the same.", strings.TrimSpace(string(state)+" "+reason))
		}
		if err := auth.Set(ctx, path, id, auth.Entry{Type: auth.TypeAPIKey, Key: key}); err != nil {
			return failf("%v", err)
		}
		return e.saved(nil, id, auth.TypeAPIKey)
	}
}

// statusOf is err's HTTP status in words, "HTTP 401", or "".
func statusOf(err error) string {
	if ae, ok := errors.AsType[*llm.APIError](err); ok && ae.Status > 0 {
		return fmt.Sprintf("HTTP %d", ae.Status)
	}
	return ""
}

// prefixed is s in parentheses after a space, or "".
func prefixed(s string) string {
	if s == "" {
		return ""
	}
	return " (" + s + ")"
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
	if !e.out.caps.In || e.stdin == nil {
		return usageErrorf("gobble setup asks its questions on a terminal; gobble auth login <provider> --method signs in without one")
	}
	var r editor.Reader
	r.Start(e.ctx, e.stdin)
	q := &questions{ctx: e.ctx, in: &r, out: e.out.err, raw: func() (func() error, error) { return editor.Raw(e.stdin) }}
	id, err := e.signIn(e.ctx, q, "")
	switch {
	case errors.Is(err, errCancelled):
		return failf("setup cancelled; nothing was saved")
	case err != nil:
		return err
	}
	return e.out.Resultf("gobble models list --provider %s lists its models; /model in a session switches to one.\n", id)
}
