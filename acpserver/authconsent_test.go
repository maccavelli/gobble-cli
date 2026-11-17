package acpserver

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// planConsent is the SDK's id of the ChatGPT plan's background-use consent.
const planConsent = "chatgpt_plan.background_use"

// A form-only ACP client's authenticate for openai with method chatgpt_plan,
// the real engine against the plan issuer in process, is asked the consent
// first, as a form whose message holds gobble's wording, equal to the
// step's text in _meta.gobble.step, and only then is shown the browser's
// address: the phone's consent is gobble's, not the SDK's default
// (0052-MADR D2; Confirmation 2's ACP surface, whose five terminal surfaces
// internal/cli's TestConsentWordingOnEverySurface covers).
func TestConsentWordingOverACP(t *testing.T) {
	dir := t.TempDir()
	is := newIssuer(t, openaiOrigin)
	is.scope = "openid chatgpt.tokens.use.direct"
	logins := auth.Logins{Path: filepath.Join(dir, "auth.json")}
	store := auth.ClientLogins{Logins: logins, Clients: auth.Clients{Path: filepath.Join(dir, "clients.json")}}
	var redirect, state string
	e := newSIEnv(t, siSpec{
		caps: formOnly,
		real: startReal(t, store, &http.Client{Transport: toOne{is}}),
		opts: func(o *Options) {
			o.Auth.Providers = func() []acpext.AuthProvider {
				return []acpext.AuthProvider{{ID: "openai", Name: "OpenAI", Command: "gobble auth login openai", Login: true}}
			}
		},
		script: func(n int, req acp.CreateElicitationRequest) elicitReply {
			switch step := metaStepOf(t, req); step.Kind {
			case authcontract.StepConfirm:
				return accept(map[string]acp.ElicitationContentValue{"allow": acp.NewElicitationContentValueBoolean(true)})
			case authcontract.StepOpenURL:
				redirect, state = is.setNonce(t, step.URL)
				return accept(nil)
			case authcontract.StepPaste:
				return accept(map[string]acp.ElicitationContentValue{"callback": text(goodPaste(redirect, state) + "&client_id=" + url.QueryEscape("oaiapp_issued1"))})
			}
			t.Errorf("elicitation %d: a step this sign-in has no answer for: %+v", n, req)
			return declineReply()
		},
	})
	if err := e.authenticate(t.Context(), "openai", map[string]any{"method": "chatgpt_plan", "remote": true}); err != nil {
		t.Fatal(err)
	}
	reqs := e.client.requests()
	if len(reqs) < 2 {
		t.Fatalf("%d elicitations, want the consent, the browser's address and a paste", len(reqs))
	}
	first := metaStepOf(t, reqs[0])
	if first.Kind != authcontract.StepConfirm || first.ConfirmID != planConsent {
		t.Fatalf("the first step %+v, want the consent: the form came before the browser", first)
	}
	want := provider.ConsentText[planConsent]
	if want == "" {
		t.Fatal("gobble has no wording for the consent: the test holds nothing")
	}
	if first.Text != want {
		t.Errorf("_meta.gobble.step.text %q, want gobble's wording %q", first.Text, want)
	}
	if reqs[0].Form == nil || !strings.Contains(reqs[0].Form.Message, want) {
		t.Errorf("the consent form's message %q, want gobble's wording in it", messageOf(reqs[0]))
	}
	if second := metaStepOf(t, reqs[1]); second.Kind != authcontract.StepOpenURL {
		t.Errorf("the second step %+v, want the browser's address after the consent", second)
	}
	l, err := logins.Load(t.Context(), "openai")
	if err != nil || l == nil || l.Kind != "chatgpt_plan" {
		t.Errorf("the saved session %+v, %v, want a plan session: the sign-in did not complete", l, err)
	}
}
