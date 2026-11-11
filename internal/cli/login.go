package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
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
// provider signed in to.
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
	return id, e.signInTo(ctx, prompter{q}, id, provider.SignInOptions{}, nil)
}

// signInTo signs in to provider id through the SDK's wizard, asking p, with
// o's method and callbacks (0031-MADR D7), and stores what it gives: a key
// in auth.json, a device's escaped so Pi's grammar gives it back unchanged;
// a login, which the wizard saved there through the login store. ev, when
// not nil, takes the saved event.
func (e *runEnv) signInTo(ctx context.Context, p provider.Prompter, id string, o provider.SignInOptions, ev *loginEvents) error {
	path := authPath()
	if path == "" {
		return failf("auth.json: the config directory cannot be resolved")
	}
	o.Options = e.providerOptions(id, provider.Options{ClientName: appName, ClientVersion: identity().Version})
	c, err := signInWith(ctx, id, p, o, auth.Logins{Path: path})
	switch {
	case err != nil:
		return err
	case c.Key == "":
		return e.saved(ev, id, auth.TypeOAuth)
	case c.Device:
		c.Key = catalog.QuoteValue(c.Key)
	case provider.PublicListing(id):
		p.Notify("%s lists its models without a key, so the key was not checked.", provider.Label(id))
	}
	if err := auth.SetKey(ctx, path, id, c.Key, c.Workspace); err != nil {
		return failf("%v", err)
	}
	return e.saved(ev, id, auth.TypeAPIKey)
}

// planBackgroundKey is the setting that records the user's consent to
// background requests under their ChatGPT plan (0031-MADR D7, amended
// 2026-11-10; the Sign in with ChatGPT terms, section 3.1).
const planBackgroundKey = "chatgptPlanBackground"

// planNoticeLines are what gobble stores for a ChatGPT plan sign-in, where, and
// how to remove it, shown before the browser opens (the terms, sections 1,
// 2.3 and 2.4).
var planNoticeLines = []string{
	`gobble signs in to your ChatGPT plan under its own name, "gobble".`,
	"It stores the access, refresh and ID tokens, your account id and this host's id, in auth.json in gobble's " +
		"config directory on this machine, readable by you alone, and sends them only to OpenAI.",
	"This machine must be one only you control, never a shared or managed one. Do not copy auth.json to another machine.",
	"To remove the sign-in, run gobble auth logout openai, which also revokes it at OpenAI.",
	"Background requests, a session's title and automatic compaction, run under your plan only if you allow them; " +
		"gobble config set " + planBackgroundKey + " false turns them off at any time.",
	"On another device, open the address shown, sign in there, then paste here the address the browser lands on.",
}

// planNotice shows planNoticeLines on stderr, or as a notice event with --json.
func (e *runEnv) planNotice(ev *loginEvents, id string) error {
	if ev != nil {
		return ev.emit(loginEvent{Event: "notice", Provider: id, Message: strings.Join(planNoticeLines, " ")})
	}
	for _, line := range planNoticeLines {
		if _, err := fmt.Fprintln(e.out.err, line); err != nil {
			return fmt.Errorf("write stderr: %w", err)
		}
	}
	return nil
}

// askPlanBackground asks, once a plan sign-in has saved its session,
// whether gobble may make background requests under the plan, and saves a
// yes. With no terminal, or --json, it asks nothing and the consent stays
// unset: no background request is made under the plan.
func (e *runEnv) askPlanBackground(p provider.Prompter, ev *loginEvents) error {
	if ev != nil {
		return nil
	}
	if _, ok := p.(unasked); ok {
		return nil
	}
	yes, err := p.Confirm("Allow gobble to make background requests under your ChatGPT plan: naming sessions and "+
		"compacting them on its own?", false)
	if err != nil || !yes {
		return err
	}
	if err := saveGlobalSetting(planBackgroundKey, "true"); err != nil {
		return failf("%s: %v", planBackgroundKey, err)
	}
	return nil
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
	q := &questions{ctx: e.ctx, in: inline.NewReader(e.stdin), out: e.out.err, raw: func() (func() error, error) { return editor.Raw(e.stdin) }}
	id, err := e.signIn(e.ctx, q, "")
	switch {
	case errors.Is(err, errCancelled):
		return failf("setup cancelled; nothing was saved")
	case err != nil:
		return err
	}
	return e.out.Resultf("gobble models list --provider %s lists its models; /model in a session switches to one.\n", id)
}
