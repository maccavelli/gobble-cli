package cli

import (
	"context"
	"encoding/json/v2"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// planConsentID is the SDK's id of the ChatGPT plan's background-use consent,
// the key of provider.ConsentText.
const planConsentID = "chatgpt_plan.background_use"

// planLogin is a ChatGPT plan session as gobble stores one: its kind set, the
// way provider.StartSignIn's save leaves it.
func planLogin() *provider.Login {
	return &provider.Login{Access: "plan-access-secret", Refresh: "plan-refresh-secret", Expires: time.UnixMilli(1781000000000),
		Issuer: loginOpenAIIssuer, ClientID: "oaiapp_test", TokenURL: loginOpenAIIssuer + "/oauth/token",
		Resource: "https://api.openai.com/v1", Scope: "openid chatgpt.tokens.use.direct", HostID: "urn:uuid:11111111-2222-4333-8444-555555555555",
		IDToken: "plan-id-token",
		Kind:    "chatgpt_plan"}
}

// foreignLogin is another application's ChatGPT login, such as the Codex
// client's: a session of a kind gobble offers no sign-in to.
func foreignLogin() *provider.Login {
	return &provider.Login{Access: "codex-access-secret", Refresh: "codex-refresh-secret", Expires: time.UnixMilli(1781000000000),
		Issuer: loginOpenAIIssuer, ClientID: "app_codex", TokenURL: loginOpenAIIssuer + "/oauth/token", Kind: "chatgpt"}
}

// grokLogin is a Grok session.
func grokLogin() *provider.Login {
	return &provider.Login{Access: "grok-access-secret", Refresh: "grok-refresh-secret", Expires: time.UnixMilli(1781000000000),
		Issuer: loginGrokIssuer, ClientID: "b1a00492-073a-47ea-816f-4c329264a828", TokenURL: loginGrokIssuer + "/oauth2/token", Kind: "grok"}
}

// saveLogin stores l as provider id's login in c's auth.json, as gobble does.
func saveLogin(t *testing.T, c *cfgEnv, id string, l *provider.Login) {
	t.Helper()
	if err := (auth.Logins{Path: c.authFile()}).Save(t.Context(), id, l); err != nil {
		t.Fatal(err)
	}
}

// statusPlanRow is the members of a row of auth status --json that the plan
// sign-in adds to 0045-MADR D2's.
type statusPlanRow struct {
	Provider     string                 `json:"provider"`
	Usable       bool                   `json:"usable"`
	Reason       string                 `json:"reason"`
	AuthType     string                 `json:"authType"`
	SessionKind  string                 `json:"sessionKind"`
	Consents     map[string]string      `json:"consents"`
	ConsentTerms []authcontract.Consent `json:"consentTerms"`
}

// noTokens fails when what gobble printed holds a token or an ID token: every
// token the fixtures store carries "secret", and the ID token's "id-token".
func noTokens(t *testing.T, what string, outputs ...string) {
	t.Helper()
	for _, o := range outputs {
		if strings.Contains(o, "secret") || strings.Contains(o, "id-token") {
			t.Errorf("%s printed a credential:\n%s", what, o)
		}
	}
}

// textRow is provider id's row of c's auth status, which holds no token.
func textRow(t *testing.T, c *cfgEnv, id string) string {
	t.Helper()
	r := c.run("auth", "status")
	noTokens(t, "auth status", r.stdout, r.stderr)
	return rowOf(r.stdout, id)
}

// statusRowsOf is c's auth status --json, by provider, and its raw output.
func statusRowsOf(t *testing.T, c *cfgEnv) (map[string]statusPlanRow, result) {
	t.Helper()
	r := c.run("auth", "status", "--json")
	noTokens(t, "auth status --json", r.stdout, r.stderr)
	var out struct {
		Providers []statusPlanRow `json:"providers"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &out); err != nil {
		t.Fatalf("auth status --json: %v:\n%s", err, r.stdout)
	}
	rows := map[string]statusPlanRow{}
	for _, row := range out.Providers {
		rows[row.Provider] = row
	}
	return rows, r
}

// A plan session's row has its kind, its consent's terms in gobble's wording
// with the SDK's usage link, and the answers the session holds; a foreign
// openai session is not usable, whatever it holds; a grok session has a kind
// and no consent; the text form says "background use" allowed, declined or
// not asked; and a store that cannot be read is 0045-MADR D5's error object
// (0052-MADR D8).
func TestAuthStatusJSONPlan(t *testing.T) {
	pkg, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("a plan session with no answer", func(t *testing.T) {
		c := newCfgEnv(t)
		saveLogin(t, c, "openai", planLogin())
		saveLogin(t, c, "grok", grokLogin())
		rows, r := statusRowsOf(t, c)
		if r.code != ExitOK || r.stderr != "" {
			t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
		}
		plan := rows["openai"]
		if !plan.Usable || plan.AuthType != "oauth" || plan.SessionKind != "chatgpt_plan" || plan.Reason != "" {
			t.Errorf("the plan row %+v, want usable, oauth, sessionKind chatgpt_plan", plan)
		}
		if len(plan.Consents) != 0 {
			t.Errorf("consents %v, want none while the consent is unanswered", plan.Consents)
		}
		if len(plan.ConsentTerms) != 1 {
			t.Fatalf("consentTerms %+v, want the one consent of the plan's method", plan.ConsentTerms)
		}
		terms := plan.ConsentTerms[0]
		if terms.ID != planConsentID || !terms.Required || terms.Text != provider.ConsentText[planConsentID] || terms.Text == "" {
			t.Errorf("consentTerms[0] %+v, want id %s, required, and gobble's wording", terms, planConsentID)
		}
		if !strings.Contains(terms.Text, "gobble auth consent openai background deny") {
			t.Errorf("the wording %q is not gobble's: it lacks the command that withdraws the consent", terms.Text)
		}
		if len(terms.Links) == 0 || terms.Links[0].Rel != authcontract.LinkUsage || !strings.HasPrefix(terms.Links[0].URL, "https://") {
			t.Errorf("the consent's links %+v, want the SDK's usage link", terms.Links)
		}
		if grok := rows["grok"]; grok.SessionKind != "grok" || len(grok.Consents) != 0 || len(grok.ConsentTerms) != 0 {
			t.Errorf("the grok row %+v, want its kind and no consent", grok)
		}
		if got := textRow(t, c, "openai"); !strings.HasSuffix(got, ", background use not asked") {
			t.Errorf("the text row %q, want it to say background use not asked", got)
		}
		if got := textRow(t, c, "grok"); strings.Contains(got, "background") {
			t.Errorf("grok's text row %q mentions a consent", got)
		}
		t.Chdir(pkg) // newCfgEnv's directory has no testdata
		golden(t, "auth-status-plan.json", r.stdout)
	})

	for _, answer := range []struct {
		word    string
		granted bool
		status  string
		text    string
	}{{"allow", true, "granted", "background use allowed"}, {"deny", false, "declined", "background use declined"}} {
		t.Run(answer.word, func(t *testing.T) {
			c := newCfgEnv(t)
			saveLogin(t, c, "openai", planLogin())
			if err := provider.SetConsent(t.Context(), auth.Logins{Path: c.authFile()}, "openai", "background", answer.granted); err != nil {
				t.Fatal(err)
			}
			plan := func() statusPlanRow {
				rows, _ := statusRowsOf(t, c)
				return rows["openai"]
			}()
			if got := plan.Consents[planConsentID]; got != answer.status || len(plan.Consents) != 1 {
				t.Errorf("consents %v, want %s: %s", plan.Consents, planConsentID, answer.status)
			}
			if len(plan.ConsentTerms) != 1 {
				t.Errorf("consentTerms %+v, want the terms listed whether or not answered", plan.ConsentTerms)
			}
			if got := textRow(t, c, "openai"); !strings.HasSuffix(got, ", "+answer.text) {
				t.Errorf("the text row %q, want it to end %q", got, answer.text)
			}
		})
	}

	t.Run("a foreign openai session", statusPlanForeign)
	t.Run("a foreign openai session stored with no kind", statusPlanForeignNoKind)
	t.Run("a store that cannot be read", statusPlanUnreadable)
}

// statusPlanForeignNoKind is TestAuthStatusJSONPlan's foreign openai session stored with no kind.
func statusPlanForeignNoKind(t *testing.T) {
	// A session stored before kind existed has none: Pi's entry shape, the
	// tokens, the expiry and the account (internal/auth/store.go's Entry), and
	// the issuer gobble's own sign-in recorded (0005-PLAN F4-3b decision 3). Its
	// kind is the SDK's inference, and as another application's login it is
	// not gobble's.
	c := newCfgEnv(t)
	c.write(c.authFile(), `{"openai": {"type": "oauth", "access": "codex-access-secret", "refresh": "codex-refresh-secret", "expires": 1781000000000, "accountId": "acct-1", "issuer": "https://auth.openai.com", "clientId": "app_codex"}}`)
	rows, r := statusRowsOf(t, c)
	if r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	foreign := rows["openai"]
	if foreign.Usable || foreign.Reason != "oauth_unused" || foreign.SessionKind != "chatgpt" || len(foreign.ConsentTerms) != 0 {
		t.Errorf("the row of a session with no stored kind %+v, want usable false, oauth_unused, sessionKind chatgpt by inference", foreign)
	}
	if got := textRow(t, c, "openai"); !strings.HasSuffix(got, ", not used (0005-PLAN F4-3b)") {
		t.Errorf("the text row %q, want it not used", got)
	}
}

// statusPlanForeign is TestAuthStatusJSONPlan's foreign openai session, stored with its kind.
func statusPlanForeign(t *testing.T) {
	c := newCfgEnv(t)
	saveLogin(t, c, "openai", foreignLogin())
	rows, r := statusRowsOf(t, c)
	if r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	foreign := rows["openai"]
	if foreign.Usable || foreign.Reason != "oauth_unused" || foreign.AuthType != "oauth" || foreign.SessionKind != "chatgpt" {
		t.Errorf("the foreign row %+v, want usable false, oauth_unused, sessionKind chatgpt", foreign)
	}
	if len(foreign.Consents) != 0 || len(foreign.ConsentTerms) != 0 {
		t.Errorf("the foreign row %+v holds consents, which only gobble's own session has", foreign)
	}
	if got := textRow(t, c, "openai"); !strings.HasSuffix(got, ", not used (0005-PLAN F4-3b)") || strings.Contains(got, "background") {
		t.Errorf("the text row %q, want it not used and with no consent", got)
	}
}

// statusPlanUnreadable is TestAuthStatusJSONPlan's auth.json that does not parse.
func statusPlanUnreadable(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), "this is not json")
	r := c.run("auth", "status", "--json")
	noTokens(t, "auth status --json over an unreadable store", r.stdout, r.stderr)
	var out struct {
		Providers []statusPlanRow `json:"providers"`
		Error     *statusError    `json:"error"`
	}
	if err := json.Unmarshal([]byte(r.stdout), &out); err != nil {
		t.Fatalf("auth status --json: %v:\n%s", err, r.stdout)
	}
	if r.code != ExitUsage || r.stderr != "" || len(out.Providers) != 0 || out.Error == nil || out.Error.File != "auth.json" {
		t.Errorf("exit %d, stderr %q, stdout %s; want exit 2, nothing on stderr, no providers and the error object", r.code, r.stderr, r.stdout)
	}
}

// recordedTransport is an HTTP transport that records each request and
// answers none: a logout of a session gobble does not own must send nothing.
type recordedTransport struct{ seen *[]string }

func (r recordedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	*r.seen = append(*r.seen, req.Method+" "+req.URL.String())
	return nil, errors.New("the test's transport answers nothing")
}

// A stored openai session that is not a ChatGPT plan session gobble signed in
// is refused everywhere the resolver is: status, auth check, authenticate's
// present credential, the choice of provider, and the build of the run's
// provider and of a session's; and the logout deletes it without revoking
// it, as gobble holds no right to revoke another application's session
// (0051-MADR D2; 0052-MADR D7, F54). The same surfaces accept a plan session,
// so the refusal is the session's kind and not the surface's.
func TestForeignOpenAISessionRefused(t *testing.T) {
	surfaces := func(t *testing.T, c *cfgEnv) (status statusPlanRow, check result, has, chosen, built, builtFor bool) {
		t.Helper()
		rows, _ := statusRowsOf(t, c)
		status = rows["openai"]
		check = c.run("auth", "check", "--provider", "openai", "--json")
		e := envHere(t)
		has = e.authOptions().Has("openai")
		sel, err := provider.ChooseWith(nil, e.hasCredential, "")
		chosen = err == nil && sel.ID == "openai"
		_, err = modelProvider(e)(nil, "session-1", llm.Retry{})
		built = err == nil
		_, err = providerFor(e)(nil, "openai", "session-2", llm.Retry{})
		builtFor = err == nil
		return status, check, has, chosen, built, builtFor
	}

	t.Run("a plan session is accepted on every surface", func(t *testing.T) {
		c := newCfgEnv(t)
		saveLogin(t, c, "openai", planLogin())
		status, check, has, chosen, built, builtFor := surfaces(t, c)
		if !status.Usable || check.code != ExitOK || !has || !chosen || !built || !builtFor {
			t.Errorf("a plan session: status usable %v, check exit %d (%s), authenticate has %v, chosen %v, run build %v, session build %v; want all accepted",
				status.Usable, check.code, strings.TrimSpace(check.stdout), has, chosen, built, builtFor)
		}
	})

	t.Run("a foreign session is refused on every surface", func(t *testing.T) {
		c := newCfgEnv(t)
		saveLogin(t, c, "openai", foreignLogin())
		status, check, has, chosen, built, builtFor := surfaces(t, c)
		if status.Usable || status.Reason != "oauth_unused" {
			t.Errorf("status: usable %v, reason %q; want not usable, oauth_unused", status.Usable, status.Reason)
		}
		if check.code == ExitOK || strings.Contains(check.stdout, `"ready"`) {
			t.Errorf("auth check: exit %d, %s; want it refused", check.code, strings.TrimSpace(check.stdout))
		}
		if has {
			t.Error("authenticate counts the foreign session as the provider's credential")
		}
		if chosen {
			t.Error("the choice of provider took openai on a foreign session")
		}
		if built {
			t.Error("the run's provider was built on a foreign session")
		}
		if builtFor {
			t.Error("a session's provider was built on a foreign session")
		}
	})

	t.Run("the logout deletes and does not revoke", func(t *testing.T) {
		c := newCfgEnv(t)
		saveLogin(t, c, "grok", grokLogin())
		saveLogin(t, c, "openai", foreignLogin())
		var requests []string
		replace(t, &logoutLogin, func(ctx context.Context, id string, o provider.LoginOptions, store provider.LoginStore) (revoke, err error) {
			o.HTTPClient = &http.Client{Transport: recordedTransport{&requests}} // production's logout, its network a recorder
			return provider.Logout(ctx, id, o, store)
		})
		c.want(c.run("auth", "logout", "openai"), ExitOK, "Logged out of openai\n", "")
		if len(requests) != 0 {
			t.Errorf("the logout of a foreign session sent %q, want nothing: it was not gobble's to revoke", requests)
		}
		f, err := auth.Load(c.authFile())
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := f.Entry("openai"); ok {
			t.Error("the foreign session is still in auth.json")
		}
		if _, ok := f.Entry("grok"); !ok {
			t.Error("the logout of openai removed grok's session")
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(c.authFile()), "clients.json")); err == nil {
			t.Error("the logout of a foreign session wrote clients.json")
		}
	})
}
