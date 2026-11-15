package cli

import (
	"strings"
	"testing"
)

// consentFixture is an auth.json holding a ChatGPT plan session, a Grok
// session, and a Claude key, every secret holding "secret".
const consentFixture = `{
  "openai": {"type": "oauth", "access": "a-secret", "refresh": "r-secret", "expires": 4102444800000, "issuer": "https://auth.openai.com",
    "clientId": "oaiapp_test", "resource": "https://api.openai.com/v1", "kind": "chatgpt_plan"},
  "grok": {"type": "oauth", "access": "a-secret", "refresh": "r-secret", "expires": 4102444800000},
  "claude": {"type": "api_key", "key": "sk-secret"}
}`

// oneErrorLine reports that r is a failure of the given exit code with
// nothing on stdout and one line on stderr.
func oneErrorLine(r result, code int) bool {
	return r.code == code && r.stdout == "" && strings.Count(strings.TrimSuffix(r.stderr, "\n"), "\n") == 0 && strings.HasPrefix(r.stderr, "Error: ")
}

// gobble auth consent lists a provider's consents with their answers and
// stores one, by the SDK's consent id or gobble's word for it, for a person
// or a program: it asks nothing and reads no stdin, it prints one line to
// stdout on success, and an error is one line on stderr with stdout empty,
// exit 2 for a usage error and 1 for a store that cannot be read
// (0052-MADR D3).
func TestAuthConsentCommand(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), consentFixture)
	const id = "chatgpt_plan.background_use"

	c.want(c.run("auth", "consent", "openai"), ExitOK, "openai "+id+" not asked\n", "")
	if r := c.run("auth", "consent", "openai"); r.stderr != "" || r.stdout != "openai "+id+" not asked\n" {
		t.Errorf("the list: stdout %q, stderr %q; want one line and nothing on stderr", r.stdout, r.stderr)
	}
	if r := c.run("auth", "consent", "openai", id, "allow"); r.code != ExitOK || r.stdout != "openai "+id+" granted\n" || r.stderr != "" {
		t.Errorf("allow by id: exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	c.want(c.run("auth", "consent", "openai"), ExitOK, "openai "+id+" granted\n", "")
	if got := c.read(c.authFile()); !strings.Contains(got, `"consents": "`+id+`=granted"`) || !strings.Contains(got, `"access": "a-secret"`) ||
		!strings.Contains(got, `"key": "sk-secret"`) {
		t.Errorf("auth.json after allow:\n%s", got)
	}
	if r := c.run("auth", "consent", "openai", "background", "deny"); r.code != ExitOK || r.stdout != "openai "+id+" declined\n" || r.stderr != "" {
		t.Errorf("deny by word: exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	c.want(c.run("auth", "consent", "openai"), ExitOK, "openai "+id+" declined\n", "")
	if got := c.read(c.authFile()); !strings.Contains(got, `"consents": "`+id+`=declined"`) {
		t.Errorf("auth.json after deny:\n%s", got)
	}
	if r := c.run("auth", "consent", "openai", "background", "allow"); r.code != ExitOK || r.stdout != "openai "+id+" granted\n" || r.stderr != "" {
		t.Errorf("allow by word: exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	c.want(c.run("auth", "consent", "openai"), ExitOK, "openai "+id+" granted\n", "")

	// A session whose sign-in has no consent lists none and takes none.
	if r := c.run("auth", "consent", "grok"); r.code != ExitOK || r.stdout != "" || r.stderr != "" {
		t.Errorf("grok's list: exit %d, stdout %q, stderr %q; want none", r.code, r.stdout, r.stderr)
	}

	// Each usage error is one line on stderr and nothing on stdout.
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"an unknown provider", []string{"auth", "consent", "nosuch", id, "allow"}, `unknown provider "nosuch"`},
		{"a consent the sign-in has not", []string{"auth", "consent", "openai", "nope", "allow"}, `has no consent "nope"`},
		{"a provider whose sign-in has none", []string{"auth", "consent", "grok", id, "allow"}, `has no consent`},
		{"a provider with no stored sign-in", []string{"auth", "consent", "claude", id, "allow"}, "has no stored sign-in"},
		{"a provider with none, listed", []string{"auth", "consent", "kilo"}, "has no stored sign-in"},
		{"no allow or deny", []string{"auth", "consent", "openai", "background"}, "needs allow or deny"},
		{"maybe for allow or deny", []string{"auth", "consent", "openai", "background", "maybe"}, `"maybe" is not allow or deny`},
	} {
		r := c.run(tc.args...)
		if !oneErrorLine(r, ExitUsage) || !strings.Contains(r.stderr, tc.want) {
			t.Errorf("%s: exit %d, stdout %q, stderr %q; want exit 2, nothing on stdout, one line naming %q", tc.name, r.code, r.stdout, r.stderr, tc.want)
		}
	}
	c.want(c.run("auth", "consent", "openai"), ExitOK, "openai "+id+" granted\n", "") // none of them wrote

	// A store that cannot be parsed is exit 1 with its error, whichever the
	// command, and nothing is printed as an answer.
	c.write(c.authFile(), `{"openai": `)
	for _, args := range [][]string{{"auth", "consent", "openai"}, {"auth", "consent", "openai", "background", "allow"}, {"auth", "consent", "openai", "background", "deny"}} {
		r := c.run(args...)
		if !oneErrorLine(r, ExitFailure) || strings.Contains(r.stdout+r.stderr, "not asked") || strings.Contains(r.stdout+r.stderr, "granted") {
			t.Errorf("%v with an auth.json that does not parse: exit %d, stdout %q, stderr %q; want exit 1, nothing on stdout, one line", args, r.code, r.stdout, r.stderr)
		}
	}
	if got := c.read(c.authFile()); got != `{"openai": ` {
		t.Errorf("an unreadable auth.json was written: %q", got)
	}
}
