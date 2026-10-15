package acpserver

import (
	"errors"
	"slices"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// authConn is an agent over testAuth, joined to a client that offers a
// terminal or not, initialized.
func authConn(t *testing.T, terminal bool) (*acptest.Conn, acp.InitializeResponse) {
	t.Helper()
	agent := New(Options{Version: "1.2.3", Auth: testAuth})
	c := acptest.Connect(t, agent, &acptest.Client{})
	agent.SetConnection(c.Agent)
	resp, err := c.Client.Initialize(t.Context(), acp.InitializeRequest{
		ProtocolVersion:    acp.ProtocolVersionNumber,
		ClientInfo:         &acp.Implementation{Name: "test", Version: "0.0.0"},
		ClientCapabilities: acp.ClientCapabilities{Auth: acp.AuthCapabilities{Terminal: terminal}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return c, resp
}

// methodsOf names each method "agent:<id>" or "terminal:<id> <args>".
func methodsOf(ms []acp.AuthMethod) []string {
	var out []string
	for _, m := range ms {
		switch {
		case m.Agent != nil:
			out = append(out, "agent:"+string(m.Agent.Id)+" "+m.Agent.Name)
		case m.Terminal != nil:
			out = append(out, "terminal:"+string(m.Terminal.Id)+" "+m.Terminal.Name+" "+strings.Join(m.Terminal.Args, " "))
		}
	}
	return out
}

// Every client sees an agent method per provider, with its label and the
// command that stores its credential; a client that offers a terminal also
// sees a terminal method per provider, running gobble acp --login
// (0016-MADR A2).
func TestAuthMethods(t *testing.T) {
	_, resp := authConn(t, false)
	if got, want := methodsOf(resp.AuthMethods), []string{"agent:grok Grok (xAI)", "agent:keyed Keyed"}; !slices.Equal(got, want) {
		t.Errorf("without auth.terminal: %q, want %q", got, want)
	}
	if d := resp.AuthMethods[1].Agent.Description; d == nil || *d != "Uses the credential gobble has for Keyed; store one with gobble auth login keyed --api-key -" {
		t.Errorf("keyed's description: %v", d)
	}
	_, resp = authConn(t, true)
	want := []string{"agent:grok Grok (xAI)", "agent:keyed Keyed",
		"terminal:grok-login Sign in to Grok (xAI) --login grok", "terminal:keyed-login Store an API key for Keyed --login keyed"}
	if got := methodsOf(resp.AuthMethods); !slices.Equal(got, want) {
		t.Errorf("with auth.terminal: %q, want %q", got, want)
	}
	if none := New(Options{}).authMethods(true); none == nil || len(none) != 0 {
		t.Errorf("no providers: %v, want an empty list", none)
	}
}

// authenticate answers {} for a provider with a credential, auth_required
// naming the provider and its command for one without, -32602 for an
// unknown id; a terminal id is refused before gobble's handler, and no
// credential crosses ACP (0016-MADR A2).
func TestAuthenticate(t *testing.T) {
	c, _ := authConn(t, true)
	auth := func(id string) error {
		_, err := c.Client.Authenticate(t.Context(), acp.AuthenticateRequest{MethodId: acp.AuthMethodId(id)})
		return err
	}
	if err := auth("keyed"); err != nil {
		t.Errorf("keyed, with a credential: %v", err)
	}
	err := auth("grok")
	re, ok := errors.AsType[*acp.RequestError](err)
	if !ok {
		t.Fatalf("grok, with none: %v, want auth_required", err)
	}
	data, _ := re.Data.(map[string]any) //nolint:errcheck // checked below
	if re.Code != -32000 || data["provider"] != "grok" || data["command"] != "gobble auth login grok" || data["reason"] != "no credential" {
		t.Errorf("grok, with none: %+v, want auth_required naming grok and its command", re)
	}
	if code, _ := errText(t, auth("nope")); code != -32602 {
		t.Errorf("an unknown method: %d, want -32602", code)
	}
	if err := auth("grok-login"); !errors.Is(err, acp.ErrTerminalAuthMethod) {
		t.Errorf("a terminal method: %v, want the SDK's ErrTerminalAuthMethod", err)
	}
}

// gobble advertises no auth.logout, so logout is refused: signing out is
// gobble auth logout (0016-MADR A2).
func TestNoACPLogout(t *testing.T) {
	c, resp := authConn(t, true)
	if resp.AgentCapabilities.Auth.Logout != nil {
		t.Errorf("auth.logout advertised: %+v", resp.AgentCapabilities.Auth)
	}
	if _, err := c.Client.Logout(t.Context(), acp.LogoutRequest{}); err == nil {
		t.Error("logout answered")
	}
}
