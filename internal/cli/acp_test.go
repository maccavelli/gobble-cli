package cli

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// gobble acp --login runs what gobble auth login does, in the terminal an
// ACP client's terminal auth method opened, and serves nothing: a login
// for grok, a key read from stdin for claude (0005-PLAN F4-3c decision 5).
func TestACPLogin(t *testing.T) {
	c := newCfgEnv(t)
	logins := 0
	replace(t, &signInWith, func(ctx context.Context, id string, _ provider.Prompter, _ provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		logins++
		return provider.Credential{}, store.Save(ctx, id, &provider.Login{Access: "a", Refresh: "r"})
	})
	r := c.run("acp", "--login", "grok")
	if r.code != ExitOK || r.stdout != "Logged in to grok\n" || logins != 1 {
		t.Errorf("acp --login grok: exit %d, stdout %q, %d logins", r.code, r.stdout, logins)
	}
	c.want(c.runStdin(stdinFile(t, "sk-test-only\n"), "acp", "--login", "anthropic"), ExitOK, "Saved API key for claude\n", "")
	if got := c.read(c.authFile()); !strings.Contains(got, `"key": "sk-test-only"`) || !strings.Contains(got, `"grok"`) {
		t.Errorf("auth.json:\n%s", got)
	}
	c.want(c.run("acp", "--device"), ExitUsage, "", "--device is for --login")
	c.want(c.run("acp", "--login", "claude", "--device"), ExitUsage, "", "claude has no login; --device is for a login")
	c.want(c.run("acp", "--login", "ollama"), ExitUsage, "", `unknown provider "ollama"`)
	// The flags for served sessions are refused beside --login, which serves
	// none, rather than ignored (0049-MADR D12, as amended).
	for _, f := range [][]string{{"--mode", "plan"}, {"--provider", "openai"}, {"--model", "openai/gpt-zz"}, {"--thinking", "high"}} {
		c.want(c.run(append([]string{"acp", "--login", "grok"}, f...)...), ExitUsage, "",
			"Error: "+f[0]+" is for the sessions gobble acp serves; --login serves none")
	}
	if logins != 1 {
		t.Errorf("%d logins after the refusals, want the first alone", logins)
	}
}

// gobble acp's --provider, --model and --thinking choose the sessions'
// model and level as chat's do: the pick reaches the agent's Override, and
// chat's errors are usage errors here too (0049-MADR D12).
func TestACPModelFlags(t *testing.T) {
	e, _ := modelEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	c := &ACPCmd{Model: "openai/gpt-zz:low", Thinking: "high"}
	if err := c.choose(e); err != nil {
		t.Fatal(err)
	}
	if got := e.agentOptions().Override; got != (acpserver.Override{Provider: "openai", Model: "gpt-zz:low", Thinking: "high"}) {
		t.Errorf("override %+v, want openai/gpt-zz:low at high", got)
	}
	for _, tc := range []struct {
		name string
		f    ModelFlags
		msg  string
	}{
		{"an unknown model", ModelFlags{Model: "zzz-none"}, `Model "zzz-none" not found. Use --list-models to see available models.`},
		{"--provider alone", ModelFlags{Provider: "openai"}, "--provider requires --model"},
	} {
		err := (&ACPCmd{ModelFlags: tc.f}).choose(e)
		if ee, ok := errors.AsType[*exitError](err); !ok || ee.code != ExitUsage || ee.msg != tc.msg {
			t.Errorf("%s: %v, want the usage error %q", tc.name, err, tc.msg)
		}
	}
}

// The agent's ACP authentication lists every provider that takes a key, in
// the SDK's order, with its label and the command that stores its
// credential, and answers from gobble's credentials (0016-MADR A2).
func TestAgentAuthOptions(t *testing.T) {
	e, _ := modelEnv(t)
	o := e.agentOptions()
	ps := o.Auth.Providers()
	var ids []string
	for _, p := range ps {
		ids = append(ids, p.ID)
		want := "gobble auth login " + p.ID + " --api-key -"
		if p.Login {
			want = "gobble auth login " + p.ID
		}
		if p.Name != provider.Label(p.ID) || p.Command != want || p.Login != (p.ID == "grok" || p.ID == "kilo") {
			t.Errorf("%s: %+v", p.ID, p)
		}
	}
	if !slices.Equal(ids, provider.KeyProviders()) || len(ids) != 9 {
		t.Errorf("providers %q, want the nine key providers %q", ids, provider.KeyProviders())
	}
	t.Setenv("OPENAI_API_KEY", "test-only")
	if !o.Auth.Has("openai") || o.Auth.Has("claude") {
		t.Errorf("Has: openai %v, claude %v; want openai's variable found and claude none", o.Auth.Has("openai"), o.Auth.Has("claude"))
	}
}
