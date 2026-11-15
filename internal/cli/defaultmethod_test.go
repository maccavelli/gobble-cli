package cli

import (
	"context"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// A bare gobble auth login signs in by the provider's default offered
// method, the first that is not a key, and --device by its device method; a
// provider that offers neither is a usage error naming --api-key -, and no
// message names a provider but by the id given (0051-MADR D2; 0051-PLAN P4
// step 4a).
func TestLoginWithReadsTheDefaultMethod(t *testing.T) {
	c := newCfgEnv(t)
	var methods []string
	replace(t, &signInWith, func(_ context.Context, _ string, _ provider.Prompter, o provider.SignInOptions, _ provider.LoginStore) (provider.Credential, error) {
		methods = append(methods, o.Method)
		return provider.Credential{}, nil
	})
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"auth", "login", "grok"}, provider.MethodBrowser},
		{[]string{"auth", "login", "grok", "--device"}, provider.MethodDevice},
		{[]string{"auth", "login", "kilo"}, provider.MethodDevice},
		{[]string{"auth", "login", "kilo", "--device"}, provider.MethodDevice},
		{[]string{"auth", "login", "openai"}, provider.MethodChatGPTPlan},
		{[]string{"auth", "login", "openai", "--method", "chatgpt-plan"}, provider.MethodChatGPTPlan},
	} {
		methods = nil
		if r := c.run(tc.args...); r.code != ExitOK || !slices.Equal(methods, []string{tc.want}) {
			t.Errorf("gobble %s: exit %d, stderr %q, signed in by %q; want %q", strings.Join(tc.args, " "), r.code, r.stderr, methods, tc.want)
		}
	}
	c.want(c.run("auth", "login", "claude"), ExitUsage, "", "claude has no login; use --api-key -")
	c.want(c.run("auth", "login", "claude", "--device"), ExitUsage, "", "claude has no login; use --api-key -")
	r := c.run("auth", "login", "openai", "--device")
	if r.code != ExitUsage || r.stdout != "" || strings.TrimSpace(r.stderr) != "Error: openai has no device sign-in; --device is for one" ||
		strings.Contains(r.stderr, "ChatGPT") {
		t.Errorf("openai --device: exit %d, stdout %q, stderr %q; want the generic usage error", r.code, r.stdout, r.stderr)
	}
	if len(methods) != 1 {
		t.Errorf("a refused sign-in reached the wizard: %q", methods)
	}
}

// Every command acpext gives a client, and every one gobble builds from its
// registry, parses as gobble's own, and names a provider and a method gobble
// accepts; every offered method's word is one --method takes (issue #16's
// criterion; 0051-MADR D2).
func TestEveryCommandAndWordIsAccepted(t *testing.T) {
	for _, ap := range slices.Concat(acpext.AuthProviders(), authProviders()) {
		var root Root
		k, err := newParser(&root, io.Discard, io.Discard)
		if err != nil {
			t.Fatal(err)
		}
		args := strings.Fields(ap.Command)
		if len(args) < 2 || args[0] != "gobble" {
			t.Errorf("%s: command %q is not a gobble command", ap.ID, ap.Command)
			continue
		}
		kctx, err := k.Parse(args[1:])
		if err != nil {
			t.Errorf("%s: %q does not parse: %v", ap.ID, ap.Command, err)
			continue
		}
		login := root.Auth.Login
		if login.Method != "" {
			if _, err := methodOf(ap.ID, login.Method); err != nil {
				t.Errorf("%s: %q names a method gobble does not offer: %v", ap.ID, ap.Command, err)
			}
		}
		if kctx.Command() != "auth login <provider>" || login.Provider != ap.ID {
			t.Errorf("%s: %q parsed as %q for provider %q", ap.ID, ap.Command, kctx.Command(), login.Provider)
			continue
		}
		if _, err := keyProvider(login.Provider); err != nil {
			t.Errorf("%s: %q names a provider gobble does not take: %v", ap.ID, ap.Command, err)
		}
		switch {
		case login.APIKey != "":
			if login.APIKey != "-" || ap.Login {
				t.Errorf("%s: %q reads a key, but Login is %v", ap.ID, ap.Command, ap.Login)
			}
			if _, err := methodOf(ap.ID, provider.MethodAPIKey); err != nil {
				t.Errorf("%s: %q: %v", ap.ID, ap.Command, err)
			}
		default:
			if !ap.Login || provider.DefaultMethod(ap.ID) == "" {
				t.Errorf("%s: %q is a bare sign-in, but Login is %v and its default method %q", ap.ID, ap.Command, ap.Login, provider.DefaultMethod(ap.ID))
			}
			if _, err := methodOf(ap.ID, provider.DefaultMethod(ap.ID)); err != nil {
				t.Errorf("%s: %q: %v", ap.ID, ap.Command, err)
			}
		}
	}
	// Each offered method has the --method word the help and the guides give it, in
	// the SDK's order, and every word is one methodOf takes.
	words := map[string]string{"api_key": "api-key", "browser_oauth": "browser", "device_code": "device", "chatgpt_plan": "chatgpt-plan"}
	for _, id := range provider.KeyProviders() {
		var want []string
		for _, m := range provider.Offered(id) {
			w, ok := words[m.ID]
			if !ok {
				t.Errorf("%s: offered method %s has no --method word in this test", id, m.ID)
			}
			want = append(want, w)
		}
		if got := provider.Methods(id); !slices.Equal(got, want) {
			t.Errorf("%s: method words %q, want %q for its offered methods", id, got, want)
		}
		for _, w := range want {
			if _, err := methodOf(id, w); err != nil {
				t.Errorf("%s: method word %q: %v", id, w, err)
			}
		}
	}
}
