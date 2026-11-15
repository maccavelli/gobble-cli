package cli

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// failsAfterSave is a LoginStore over auth.json that cannot be read once a
// login was saved through it, as a file that went bad after the write.
type failsAfterSave struct {
	provider.LoginStore
	saved bool
}

func (f *failsAfterSave) Load(ctx context.Context, id string) (*provider.Login, error) {
	if f.saved {
		return nil, errors.New("auth.json: the file went bad after the write")
	}
	return f.LoginStore.Load(ctx, id)
}

func (f *failsAfterSave) Save(ctx context.Context, id string, l *provider.Login) error {
	f.saved = true
	return f.LoginStore.Save(ctx, id, l)
}

// gobble auth consent finds the consent's id before it writes, so nothing it
// reads afterwards can fail an answer already stored: exit 1 means nothing
// was written (0052-MADR D3).
func TestAuthConsentNeverFailsAfterTheWrite(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), consentFixture)
	replace(t, &consentStore, func(path string) provider.LoginStore { return &failsAfterSave{LoginStore: auth.Logins{Path: path}} })
	for _, word := range []string{"background", "chatgpt_plan.background_use"} {
		r := c.run("auth", "consent", "openai", word, "allow")
		if r.code != ExitOK || r.stdout != "openai chatgpt_plan.background_use granted\n" || r.stderr != "" {
			t.Errorf("allow by %s: exit %d, stdout %q, stderr %q; want the stored answer reported, exit 0", word, r.code, r.stdout, r.stderr)
		}
	}
	if got := c.read(c.authFile()); !strings.Contains(got, `"consents": "chatgpt_plan.background_use=granted"`) {
		t.Errorf("auth.json after the allow:\n%s", got)
	}
}

// The consent the agent gates on is read from auth.json at each request,
// through provider.PlanConsent over auth.Logins, and the agent is given it:
// a plan session with a granted, a declined or no answer; an auth.json that
// does not parse; a key; another application's OpenAI session; a provider
// with no such consent (0052-MADR D2).
func TestPlanConsentIsReadFromAuthJSON(t *testing.T) {
	const id = "chatgpt_plan.background_use"
	plan := func(consents string) string {
		return `{"openai": {"type": "oauth", "access": "a", "refresh": "r", "expires": 4102444800000, "issuer": "https://auth.openai.com",
  "clientId": "oaiapp_test", "resource": "https://api.openai.com/v1", "kind": "chatgpt_plan"` + consents + `}}`
	}
	for _, tc := range []struct {
		name, file, provider string
		want                 [3]bool // Plan, Granted, an error
		consent              string
	}{
		{"granted", plan(`, "consents": "` + id + `=granted"`), "openai", [3]bool{true, true, false}, id},
		{"declined", plan(`, "consents": "` + id + `=declined"`), "openai", [3]bool{true, false, false}, id},
		{"no answer", plan(``), "openai", [3]bool{true, false, false}, id},
		{"an auth.json that does not parse", `{"openai": `, "openai", [3]bool{false, false, true}, id},
		{"a key", `{"openai": {"type": "api_key", "key": "sk-secret"}}`, "openai", [3]bool{false, false, false}, id},
		{"another application's session", `{"openai": {"type": "oauth", "access": "a", "refresh": "r", "expires": 4102444800000,
  "issuer": "https://auth.openai.com", "kind": "chatgpt", "consents": "` + id + `=granted"}}`, "openai", [3]bool{false, false, false}, id},
		{"a provider with no such consent", plan(`, "consents": "` + id + `=granted"`), "grok", [3]bool{false, false, false}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e, _ := modelEnv(t)
			if err := os.WriteFile(authPath(), []byte(tc.file), 0o600); err != nil {
				t.Fatal(err)
			}
			o := e.agentOptions()
			if o.PlanConsent == nil {
				t.Fatal("the agent is given no PlanConsent")
			}
			for name, read := range map[string]func(string) (acpserver.ConsentState, error){"the run's method": e.planConsent, "the agent's option": o.PlanConsent} {
				st, err := read(tc.provider)
				if got := [3]bool{st.Plan, st.Granted, err != nil}; got != tc.want || st.Consent != tc.consent {
					t.Errorf("%s: %+v, %v; want plan %v, granted %v, error %v, consent %q", name, st, err, tc.want[0], tc.want[1], tc.want[2], tc.consent)
				}
			}
		})
	}
}
