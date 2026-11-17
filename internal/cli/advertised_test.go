package cli

import (
	"slices"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The sign-in methods the agent advertises to an ACP client, as gobble wires
// them (authOptions().Methods), carry the ChatGPT plan's consent in gobble's
// wording and with the SDK's links: the same consent Offered gives, which holds
// provider.ConsentText. acpserver's own test supplies its Methods; this holds
// the wiring (0052-MADR D2, Confirmation 2's advertised method).
func TestAdvertisedPlanConsentIsGobbles(t *testing.T) {
	newCfgEnv(t)
	e := envHere(t)
	consentOf := func(ms []authcontract.Method) authcontract.Consent {
		t.Helper()
		for _, m := range ms {
			if m.ID == "chatgpt_plan" {
				if len(m.Consents) != 1 {
					t.Fatalf("chatgpt_plan advertises consents %+v, want its one", m.Consents)
				}
				return m.Consents[0]
			}
		}
		t.Fatal("openai advertises no chatgpt_plan method")
		return authcontract.Consent{}
	}
	got, want := consentOf(e.authOptions().Methods("openai")), consentOf(provider.Offered("openai"))
	if got.ID != want.ID || got.Required != want.Required || got.Text != want.Text || !slices.Equal(got.Links, want.Links) {
		t.Errorf("the advertised consent %+v, want Offered's %+v", got, want)
	}
	if got.Text != provider.ConsentText[got.ID] || got.Text == "" || len(got.Links) == 0 {
		t.Errorf("the advertised consent %+v, want gobble's wording and the SDK's links", got)
	}
}
