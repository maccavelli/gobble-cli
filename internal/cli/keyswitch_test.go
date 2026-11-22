package cli

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/internal/auth"
)

// A switch from a ChatGPT plan session to an API key (gobble auth login
// openai --api-key -) replaces the session and leaves the host's files alone:
// <state>/host-id and <state>/clients.json are byte for byte what they were,
// the entry is a key and holds no plan member, and a later plan sign-in asks
// the consent again but names the stored client and the same host id, with no
// agent_name_hint (0052-MADR D4, D5, D6; Confirmation 4's "key switch", row A4).
func TestKeySwitchKeepsHostAndClient(t *testing.T) {
	c := newCfgEnv(t)
	ti := newPlanLoginIssuer(t)
	planStart(t, ti)
	e := envHere(t)
	planSignInRun(t, e, ti)

	hostBefore, err := os.ReadFile(hostIDFile(t))
	if err != nil {
		t.Fatalf("the host id file: %v", err)
	}
	host := strings.TrimSpace(string(hostBefore))
	clientsFile := stateDirFile(t, "clients.json")
	clientsBefore, err := os.ReadFile(clientsFile)
	if err != nil {
		t.Fatalf("clients.json: %v", err)
	}
	if rec, err := (auth.Clients{Path: clientsFile}).Load(t.Context(), "openai"); err != nil || rec.ClientID != issuedClient || rec.HostID != host {
		t.Fatalf("clients.json before the switch holds %+v, %v; want the issued client and the host id", rec, err)
	}

	c.want(c.runStdin(stdinFile(t, "sk-switch-only\n"), "auth", "login", "openai", "--api-key", "-"), ExitOK, "Saved API key for openai\n", "")

	f, err := auth.Load(c.authFile())
	if err != nil {
		t.Fatal(err)
	}
	entry, ok := f.Entry("openai")
	if !ok || entry.Type != auth.TypeAPIKey || entry.Key != "sk-switch-only" ||
		entry.Kind != "" || entry.Consents != "" || entry.HostID != "" || entry.IDToken != "" || entry.Access != "" {
		t.Errorf("the entry after the switch is %+v (found %v), want a key and no plan member", entry, ok)
	}
	if hostAfter, err := os.ReadFile(hostIDFile(t)); err != nil || !bytes.Equal(hostAfter, hostBefore) {
		t.Errorf("host-id after the switch is %q, %v; want it unchanged", hostAfter, err)
	}
	if clientsAfter, err := os.ReadFile(clientsFile); err != nil || !bytes.Equal(clientsAfter, clientsBefore) {
		t.Errorf("clients.json after the switch is %q, %v; want it unchanged (the issued client kept)", clientsAfter, err)
	}

	steps, again := planSignInRun(t, e, ti)
	if !slices.Contains(kindsOf(steps), authcontract.StepConfirm) {
		t.Errorf("the plan sign-in after the switch asked no consent: steps %v", kindsOf(steps))
	}
	if again.Get("client_id") != issuedClient || again.Has("agent_name_hint") || again.Get("ext_agent_host_id") != host {
		t.Errorf("the authorize query after the switch %v, want the stored client, the host %s and no agent_name_hint", again, host)
	}
}
