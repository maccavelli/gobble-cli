package cli

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// issuedClient is the client the test's plan issuer issues, which its callback
// names as OpenAI's registration does.
const issuedClient = "oaiapp_issued1"

// nonceOf is the test's browser reaching the authorize URL: the issuer then
// signs the ID token with that URL's nonce. It returns the redirect and the
// state the paste is built from, and opens nothing.
func (ti *loginIssuer) nonceOf(t *testing.T, authorize string) (redirect, state string) {
	t.Helper()
	u, err := url.Parse(authorize)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	ti.mu.Lock()
	ti.nonce = q.Get("nonce")
	ti.mu.Unlock()
	return q.Get("redirect_uri"), q.Get("state")
}

// planStart replaces the ACP path's sign-in with the real engine against the
// issuer in process, its options recorded, and a cache of the test's own that
// the SDK's listing check does not share with another test.
func planStart(t *testing.T, ti *loginIssuer) *[]provider.SignInOptions {
	t.Helper()
	target, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	cache := &provider.Cache{}
	t.Cleanup(func() {
		if err := cache.Close(context.Background()); err != nil {
			t.Error(err)
		}
	})
	seen := new([]provider.SignInOptions)
	replace(t, &startSignInWith, func(ctx context.Context, id, method string, o provider.SignInOptions, store provider.LoginStore) (*provider.SignInRun, error) {
		*seen = append(*seen, o)
		o.HTTPClient, o.Cache = &http.Client{Transport: loginTransport{target}}, cache
		return provider.StartSignIn(ctx, id, method, o, store)
	})
	return seen
}

// planSignInRun is a plan sign-in as the ACP path starts it (remote), read to
// its end as a person on another device would: the consent allowed, the
// authorize URL opened there, the address the browser lands on pasted with the
// issued client. It returns the steps in order and the authorize URL's query.
func planSignInRun(t *testing.T, e *runEnv, ti *loginIssuer) (steps []authcontract.Step, authorize url.Values) {
	t.Helper()
	run, err := e.startSignIn(t.Context(), "openai", "chatgpt_plan", true)
	if err != nil {
		t.Fatalf("starting the plan sign-in: %v", err)
	}
	var redirect, state string
	for s := range run.Steps() {
		steps = append(steps, s)
		switch s.Kind {
		case authcontract.StepConfirm:
			allow := true
			if err := run.Answer(t.Context(), authcontract.Answer{SchemaVersion: authcontract.SchemaVersion, Seq: s.Seq, Confirm: &allow}); err != nil {
				t.Fatalf("answering the consent: %v", err)
			}
		case authcontract.StepOpenURL:
			u, err := url.Parse(s.URL)
			if err != nil {
				t.Fatal(err)
			}
			authorize = u.Query()
			redirect, state = ti.nonceOf(t, s.URL)
		case authcontract.StepPaste:
			cb := redirect + "?code=c1&state=" + url.QueryEscape(state) + "&client_id=" + url.QueryEscape(issuedClient)
			if err := run.Answer(t.Context(), authcontract.Answer{SchemaVersion: authcontract.SchemaVersion, Seq: s.Seq, Paste: cb}); err != nil {
				t.Fatalf("answering the paste: %v", err)
			}
		}
	}
	if _, err := run.Wait(t.Context()); err != nil {
		t.Fatalf("the plan sign-in: %v (%v)", err, err)
	}
	return steps, authorize
}

// kindsOf are the kinds of steps, in order.
func kindsOf(steps []authcontract.Step) []authcontract.StepKind {
	var out []authcontract.StepKind
	for _, s := range steps {
		out = append(out, s.Kind)
	}
	return out
}

// stateDirFile is the file name in the state directory of the test's GOBBLE_HOME.
func stateDirFile(t *testing.T, name string) string {
	t.Helper()
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dirs.State, name)
}

// A first plan sign-in, over the ACP path (startSignIn, provider.StartSignIn: what an ACP client's authenticate runs; the
// terminal's is TestSignInChatGPTPlanTerminal), as an ACP client's authenticate starts it, asks the
// consent in gobble's wording before it shows the browser's address; it
// signs in as gobble with the host's id from <state>/host-id; and it saves
// the session with the answer, the host id and the issued client, which also
// goes to <state>/clients.json. A second sign-in over the stored session asks
// nothing, signs in to the issued client again, and keeps the host id
// (0052-MADR D2, D4, D5; 0051-PLAN P6 step 1a).
func TestSignInChatGPTPlanACP(t *testing.T) {
	c := newCfgEnv(t)
	ti := newPlanLoginIssuer(t)
	seen := planStart(t, ti)
	e := envHere(t)

	steps, first := planSignInRun(t, e, ti)
	if k := kindsOf(steps); len(k) < 2 || k[0] != authcontract.StepConfirm || k[1] != authcontract.StepOpenURL {
		t.Fatalf("a first sign-in's steps %v, want the confirm, then the browser", k)
	}
	if got, want := steps[0].Text, provider.ConsentText[planConsentID]; got != want || got == "" || steps[0].ConfirmID != planConsentID {
		t.Errorf("the confirm step: id %q text %q, want %s in gobble's wording", steps[0].ConfirmID, got, planConsentID)
	}

	b, err := os.ReadFile(hostIDFile(t))
	if err != nil {
		t.Fatalf("the host id file: %v", err)
	}
	host := strings.TrimSpace(string(b))
	if !strings.HasPrefix(host, "urn:uuid:") {
		t.Fatalf("<state>/host-id holds %q, want a urn:uuid", host)
	}
	if first.Get("agent_name_hint") != provider.AgentName || first.Get("ext_agent_host_id") != host {
		t.Errorf("the first authorize query %v, want gobble's name and the host id %s", first, host)
	}
	if len(*seen) != 1 || (*seen)[0].HostID != host || !(*seen)[0].Remote {
		t.Errorf("the sign-in's options %+v, want one remote call with the host id", *seen)
	}

	l, err := auth.Logins{Path: c.authFile()}.Load(t.Context(), "openai")
	if err != nil || l == nil {
		t.Fatalf("the saved session: %v, %v", l, err)
	}
	if l.Kind != "chatgpt_plan" || l.HostID != host || l.IDToken == "" {
		t.Errorf("the saved session: kind %q, host id %q, ID token held %v; want a plan session with the host id", l.Kind, l.HostID, l.IDToken != "")
	}
	state, err := provider.PlanConsent("openai", auth.Logins{Path: c.authFile()})
	if err != nil || !state.Plan || !state.Granted {
		t.Errorf("the saved answer %+v, %v, want granted on a plan session", state, err)
	}
	clients := auth.Clients{Path: stateDirFile(t, "clients.json")}
	rec, err := clients.Load(t.Context(), "openai")
	if err != nil || rec.ClientID != issuedClient || rec.HostID != host {
		t.Errorf("clients.json holds %+v, %v; want the issued client and the host id", rec, err)
	}

	steps, second := planSignInRun(t, e, ti)
	if slices.Contains(kindsOf(steps), authcontract.StepConfirm) {
		t.Errorf("a second sign-in's steps %v hold a confirm, want none: the session holds the answer", kindsOf(steps))
	}
	if second.Get("client_id") != issuedClient || second.Get("id_token_hint") == "" || second.Has("agent_name_hint") || second.Get("ext_agent_host_id") != host {
		t.Errorf("the second authorize query %v, want the issued client, the ID token as hint, the host id and no agent_name_hint", second)
	}
	if again, _ := (auth.Logins{Path: c.authFile()}).Load(t.Context(), "openai"); again == nil || again.HostID != host || again.Kind != "chatgpt_plan" {
		t.Errorf("the session after a second sign-in: %+v, want the host id kept", again)
	}
}

// Over the ACP path (the terminal's is TestPlanAfterLogoutReusesClientTerminal), after gobble auth logout openai, which revokes the plan session under
// gobble's name and deletes it, the next plan sign-in names the stored client
// and the host, sends no agent_name_hint, and asks the consent again (0052-MADR
// D5; 0051-PLAN P2a, N7).
func TestPlanAfterLogoutReusesClientACP(t *testing.T) {
	c := newCfgEnv(t)
	ti := newPlanLoginIssuer(t)
	planStart(t, ti)
	logoutAt(t, ti)
	e := envHere(t)
	planSignInRun(t, e, ti)

	r := c.run("auth", "logout", "openai")
	if r.code != ExitOK || r.stdout != "Logged out of openai\n" || strings.Contains(r.stderr, "could not be revoked") {
		t.Fatalf("auth logout openai: exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	ti.mu.Lock()
	revoked := slices.Clone(ti.revoked)
	ti.mu.Unlock()
	if !slices.Contains(revoked, "refresh-1") {
		t.Errorf("the issuer was sent %q, want the plan session's refresh token revoked", revoked)
	}
	if f, err := auth.Load(c.authFile()); err != nil {
		t.Fatal(err)
	} else if _, ok := f.Entry("openai"); ok {
		t.Error("the plan session is still in auth.json after the logout")
	}
	host, err := os.ReadFile(hostIDFile(t))
	if err != nil {
		t.Fatal(err)
	}
	rec, err := auth.Clients{Path: stateDirFile(t, "clients.json")}.Load(t.Context(), "openai")
	if err != nil || rec.ClientID != issuedClient || rec.HostID != strings.TrimSpace(string(host)) {
		t.Fatalf("clients.json after the logout holds %+v, %v; want the issued client kept", rec, err)
	}

	steps, again := planSignInRun(t, envHere(t), ti)
	if !slices.Contains(kindsOf(steps), authcontract.StepConfirm) {
		t.Errorf("the sign-in after a logout asked no consent: steps %v", kindsOf(steps))
	}
	if again.Get("client_id") != issuedClient || again.Has("agent_name_hint") || again.Get("ext_agent_host_id") != strings.TrimSpace(string(host)) {
		t.Errorf("the authorize query after the logout %v, want the stored client, the host and no agent_name_hint", again)
	}
}

// logoutAt replaces the tests' logout with production's, its network the plan
// issuer in process, so a logout revokes where the issuer can see it.
func logoutAt(t *testing.T, ti *loginIssuer) {
	t.Helper()
	target, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	replace(t, &logoutLogin, func(ctx context.Context, id string, o provider.LoginOptions, store provider.LoginStore) (revoke, err error) {
		o.HTTPClient = &http.Client{Transport: loginTransport{target}}
		return provider.Logout(ctx, id, o, store)
	})
}

// terminalPlan is gobble auth login openai on a terminal, the real wizard
// behind provider.SignIn against the plan issuer, as one person at it: each of
// answers is a line typed at a question, in order, and then, when the browser
// is opened, the callback the browser lands on is pasted (the wizard's own
// loopback listener is the SDK's; the paste wins it). It returns the authorize
// URL's query, what the wizard had put on stderr in all and when the browser was
// opened, and the error.
func terminalPlan(t *testing.T, e *runEnv, ti *loginIssuer, answers ...string) (authorize url.Values, shown, beforeURL string, err error) {
	t.Helper()
	target, perr := url.Parse(ti.URL)
	if perr != nil {
		t.Fatal(perr)
	}
	cache := &provider.Cache{}
	t.Cleanup(func() {
		if cerr := cache.Close(context.Background()); cerr != nil {
			t.Error(cerr)
		}
	})
	r, w, perr := os.Pipe()
	if perr != nil {
		t.Fatal(perr)
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})
	for _, a := range answers {
		if _, werr := w.WriteString(a + "\n"); werr != nil {
			t.Fatal(werr)
		}
	}
	e.out.caps.In, e.stdin = true, r
	opened := false
	replace(t, &openURL, func(_ context.Context, u string) error {
		parsed, uerr := url.Parse(u)
		if uerr != nil {
			return uerr
		}
		authorize = parsed.Query()
		if !opened { // the first time the browser opens
			opened, beforeURL = true, e.out.err.(*bytes.Buffer).String() // what the person had been shown by then
		}
		redirect, state := ti.nonceOf(t, u)
		cb := redirect + "?code=c1&state=" + url.QueryEscape(state) + "&client_id=" + url.QueryEscape(issuedClient)
		_, werr := w.WriteString(cb + "\n")
		return werr
	})
	replace(t, &signInWith, func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		o.HTTPClient, o.Cache = &http.Client{Transport: loginTransport{target}}, cache
		return provider.SignIn(ctx, id, p, o, store)
	})
	err = e.loginWith(t.Context(), "openai", false, nil)
	return authorize, e.out.err.(*bytes.Buffer).String(), beforeURL, err
}

// A first plan sign-in on the terminal (provider.SignIn, the wizard behind
// gobble auth login openai) asks the consent in gobble's wording before the
// browser opens, then completes by the pasted callback: it saves a plan
// session with the answer, the host id read from <state>/host-id, and the
// issued client, which also goes to <state>/clients.json; a second sign-in
// over the stored session asks no consent and signs in to the issued client
// again. This is the plan's TestSignInChatGPTPlan and TestSignInChatGPTPlanPasted
// on the terminal; TestSignInChatGPTPlanACP is the ACP path's (0052-MADR D2,
// D4, D5).
func TestSignInChatGPTPlanTerminal(t *testing.T) {
	c := newCfgEnv(t)
	ti := newPlanLoginIssuer(t)
	e := envHere(t)

	first, shown, beforeURL, err := terminalPlan(t, e, ti, "1")
	signInFailed(t, "the first plan sign-in on the terminal", err)
	want := provider.ConsentText[planConsentID]
	if !strings.Contains(beforeURL, want) {
		t.Errorf("when the browser was opened the terminal had shown\n%s\nwant gobble's wording in it first: the consent comes before the browser (the whole output: %q)", beforeURL, shown)
	}
	b, err := os.ReadFile(hostIDFile(t))
	if err != nil {
		t.Fatalf("the host id file: %v", err)
	}
	host := strings.TrimSpace(string(b))
	if first.Get("agent_name_hint") != provider.AgentName || first.Get("ext_agent_host_id") != host || !strings.HasPrefix(host, "urn:uuid:") {
		t.Errorf("the first authorize query %v, want gobble's name and the host id %q of <state>/host-id", first, host)
	}
	logins := auth.Logins{Path: c.authFile()}
	l, err := logins.Load(t.Context(), "openai")
	if err != nil || l == nil || l.Kind != "chatgpt_plan" || l.HostID != host || l.IDToken == "" {
		t.Fatalf("the saved session %+v, %v; want a plan session with the host id %q", l, err, host)
	}
	if state, err := provider.PlanConsent("openai", logins); err != nil || !state.Plan || !state.Granted {
		t.Errorf("the saved answer %+v, %v, want granted on a plan session", state, err)
	}
	clients := auth.Clients{Path: stateDirFile(t, "clients.json")}
	if rec, err := clients.Load(t.Context(), "openai"); err != nil || rec.ClientID != issuedClient || rec.HostID != host {
		t.Errorf("clients.json holds %+v, %v; want the issued client and the host id", rec, err)
	}

	second, shown, _, err := terminalPlan(t, envHere(t), ti)
	signInFailed(t, "the second plan sign-in on the terminal", err)
	if strings.Contains(shown, provider.ConsentText[planConsentID]) {
		t.Error("the second sign-in asked the consent again, though the session holds the answer")
	}
	if second.Get("client_id") != issuedClient || second.Get("id_token_hint") == "" || second.Has("agent_name_hint") || second.Get("ext_agent_host_id") != host {
		t.Errorf("the second authorize query %v, want the issued client, the ID token as hint, the host id and no agent_name_hint", second)
	}
}

// On the terminal, after gobble auth logout openai (revoked under gobble's
// name, then deleted), the next plan sign-in names the stored client and the
// host, sends no agent_name_hint, and asks the consent again: the plan's
// TestPlanAfterLogoutReusesClient on the terminal path (0052-MADR D5).
func TestPlanAfterLogoutReusesClientTerminal(t *testing.T) {
	c := newCfgEnv(t)
	ti := newPlanLoginIssuer(t)
	logoutAt(t, ti)
	_, _, _, err := terminalPlan(t, envHere(t), ti, "1")
	signInFailed(t, "the first plan sign-in on the terminal", err)
	r := c.run("auth", "logout", "openai")
	if r.code != ExitOK || r.stdout != "Logged out of openai\n" || strings.Contains(r.stderr, "could not be revoked") {
		t.Fatalf("auth logout openai: exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	b, err := os.ReadFile(hostIDFile(t))
	if err != nil {
		t.Fatal(err)
	}
	host := strings.TrimSpace(string(b))
	clients := auth.Clients{Path: stateDirFile(t, "clients.json")}
	if rec, err := clients.Load(t.Context(), "openai"); err != nil || rec.ClientID != issuedClient || rec.HostID != host {
		t.Fatalf("clients.json after the logout holds %+v, %v; want the issued client kept", rec, err)
	}
	again, _, beforeURL, err := terminalPlan(t, envHere(t), ti, "1")
	signInFailed(t, "the plan sign-in after the logout", err)
	if !strings.Contains(beforeURL, provider.ConsentText[planConsentID]) {
		t.Errorf("the sign-in after a logout had shown %q when the browser opened, want the consent asked first", beforeURL)
	}
	if again.Get("client_id") != issuedClient || again.Has("agent_name_hint") || again.Get("ext_agent_host_id") != host {
		t.Errorf("the authorize query after the logout %v, want the stored client, the host and no agent_name_hint", again)
	}
}

// signInFailed fails the test when a terminal sign-in failed, and says what
// that is: when the SDK's wizard could not bind its registered callback ports,
// 1455 and 1457, which it does for a terminal sign-in and an option to skip is
// not yet offered (0052-PLAN P4, the owner's ruling of 2026-11-17, relayed by
// the coordinator), the message names the collision, which is not a defect of
// gobble's. It is still a failure, never a skip: the sign-in did not happen.
func signInFailed(t *testing.T, what string, err error) {
	t.Helper()
	if err == nil {
		return
	}
	for e := err; e != nil; e = errors.Unwrap(e) {
		if strings.Contains(e.Error(), "registered callback ports unavailable") {
			t.Fatalf("%s failed because both callback ports, 1455 and 1457, are in use on this machine: the SDK's wizard binds them "+
				"for a terminal sign-in (another process holds them: a Codex login, or another run of these tests). Nothing is wrong with gobble; "+
				"free the ports and run again. The SDK said: %v", what, e)
		}
	}
	t.Fatalf("%s: %v", what, err)
}
