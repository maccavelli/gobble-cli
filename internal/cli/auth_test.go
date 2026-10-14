package cli

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/tool/builtin"
)

// runStdin is cfgEnv.run with stdin.
func (c *cfgEnv) runStdin(stdin *os.File, args ...string) result {
	c.t.Helper()
	var out, errw bytes.Buffer
	code := Main(c.t.Context(), args, stdin, &out, &errw)
	return result{code: code, stdout: out.String(), stderr: errw.String()}
}

func (c *cfgEnv) authFile() string { return filepath.Join(c.home, "auth.json") }

// gobble auth login stores a key read from stdin, in Pi's words, keeping the
// file's other entries; without --api-key -, it waits for F4-3b
// (0005-PLAN F4-3a decision 8).
func TestAuthLogin(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), `{"custom": {"type": "api_key", "key": "keep"}}`)
	r := c.runStdin(stdinFile(t, "sk-test-only\n"), "auth", "login", "openai", "--api-key", "-")
	c.want(r, ExitOK, "Saved API key for openai\n", "")
	got := c.read(c.authFile())
	if !strings.Contains(got, `"key": "sk-test-only"`) || !strings.Contains(got, `"custom": {`) || !strings.Contains(got, `"key": "keep"`) {
		t.Errorf("auth.json:\n%s", got)
	}
	c.want(c.runStdin(stdinFile(t, "k2"), "auth", "login", "anthropic", "--api-key", "-"), ExitOK, "Saved API key for claude\n", "")
	c.want(c.run("auth", "login", "openai"), ExitUsage, "", "until OAuth logins arrive (0005-PLAN F4-3b)")
	if r := c.run("auth", "login", "openai", "--api-key", "sk-literal"); r.code != ExitUsage ||
		!strings.Contains(r.stderr, apiKeyLiteral) || strings.Contains(r.stderr, "sk-literal") {
		t.Errorf("a literal: exit %d, stderr %q", r.code, r.stderr)
	}
	c.want(c.runStdin(stdinFile(t, "k"), "auth", "login", "ollama", "--api-key", "-"), ExitUsage, "", `unknown provider "ollama"`)
	c.want(c.runStdin(stdinFile(t, "  \n"), "auth", "login", "openai", "--api-key", "-"), ExitUsage, "", "no key was given")
}

// gobble auth logout removes the entry, in Pi's two messages, and says so
// when there is none; an OAuth entry is logged out of.
func TestAuthLogout(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), `{"openai": {"type": "api_key", "key": "k"}, "claude": {"type": "oauth", "access": "a", "refresh": "r", "expires": 1}}`)
	c.want(c.run("auth", "logout", "openai"), ExitOK,
		"Removed stored API key for openai. Environment variables and models.json config are unchanged.\n", "")
	c.want(c.run("auth", "logout", "openai"), ExitOK, "No stored credentials to remove.", "")
	c.want(c.run("auth", "logout", "anthropic"), ExitOK, "Logged out of claude\n", "")
	if got := c.read(c.authFile()); got != "{}\n" {
		t.Errorf("auth.json after both logouts: %q", got)
	}
}

// gobble auth status names each provider's source, and never a value.
func TestAuthStatus(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), `{"openai": {"type": "api_key", "key": "sk-stored-secret"},
  "claude": {"type": "oauth", "access": "a-secret", "refresh": "r-secret", "expires": 1781000000000}}`)
	c.write(filepath.Join(c.home, "models.json"), `{"providers": {"grok": {"apiKey": "!print-a-secret"}}}`)
	t.Setenv("GEMINI_API_KEY", "env-secret")
	r := c.run("auth", "status")
	if r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	for _, want := range []string{"openai        auth.json\n", "claude        auth.json, an OAuth login, expires ", "gemini        GEMINI_API_KEY\n",
		"grok          models.json\n", "kilo          none\n"} {
		if !strings.Contains(r.stdout, want) {
			t.Errorf("no %q in:\n%s", want, r.stdout)
		}
	}
	if strings.Contains(r.stdout+r.stderr, "secret") {
		t.Errorf("status printed a credential:\n%s%s", r.stdout, r.stderr)
	}
}

// gobble auth check is Pi's: ready, not_ready or invalid, with a reason,
// exiting 0, 1 or 2.
func TestAuthCheck(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), `{"openai": {"type": "api_key", "key": "k"}, "claude": {"type": "oauth", "access": "a", "refresh": "r", "expires": 1}}`)
	c.want(c.run("auth", "check", "--provider", "openai"), ExitOK, "ready\n", "")
	c.want(c.run("auth", "check", "--provider", "kilo"), ExitFailure, "not_ready\n", "")
	c.want(c.run("auth", "check", "--model", "anthropic/claude-haiku-5-5"), ExitUsage, "invalid\n", "")
	c.want(c.run("auth", "check"), ExitUsage, "", "Auth checks require --provider <provider> or --model <model>")
	for args, want := range map[string]checkResult{
		"--provider openai": {Status: "ready", Provider: "openai", AuthType: "api_key"},
		"--provider kilo":   {Status: "not_ready", Provider: "kilo", Reason: "credentials_not_configured"},
		"--provider nope":   {Status: "not_ready", Provider: "nope", Reason: "provider_not_found"},
		"--provider claude": {Status: "invalid", Provider: "claude", Reason: "invalid_state"},
	} {
		r := c.run(append([]string{"auth", "check", "--json"}, strings.Fields(args)...)...)
		var got checkResult
		if err := json.Unmarshal([]byte(r.stdout), &got); err != nil || got != want {
			t.Errorf("%s: %q (%v), want %+v", args, r.stdout, err, want)
		}
	}
}

// --api-key - reads the run's key from stdin for --model's provider, and
// never writes it; a literal, or no --model, is refused (decision 7).
func TestAPIKeyFlag(t *testing.T) {
	c := newCfgEnv(t)
	var got string
	old := ambientProvider
	ambientProvider = func(key func(string) (string, bool), _ string, o provider.Options) (llm.Provider, provider.Selection, error) {
		got, _ = key("openai")
		if o.CredentialSource != "--api-key" {
			got += " (source " + o.CredentialSource + ")"
		}
		return nil, provider.Selection{}, errors.New("recorded")
	}
	t.Cleanup(func() { ambientProvider = old })
	c.runStdin(stdinFile(t, "sk-flag-only\n"), "--api-key", "-", "--model", "openai/gpt-6-luna", "-p", "hello")
	if got != "sk-flag-only" {
		t.Errorf("the provider's key %q, want stdin's, from --api-key", got)
	}
	if _, err := os.Stat(c.authFile()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("auth.json exists after --api-key: %v", err)
	}
	if r := c.run("--api-key", "sk-literal", "--model", "openai/gpt-6-luna", "-p", "hello"); r.code != ExitUsage ||
		!strings.Contains(r.stderr, apiKeyLiteral) || strings.Contains(r.stderr, "sk-literal") {
		t.Errorf("a literal: exit %d, stderr %q", r.code, r.stderr)
	}
	c.want(c.runStdin(stdinFile(t, "k"), "--api-key", "-", "-p", "hello"), ExitUsage, "",
		"--api-key requires a model to be specified via --model or --provider/--model")
}

// A key whose command fails chooses its provider, as Pi's check counts it,
// but lists nothing, so no request goes without a key, and the CLI says the
// key could not be resolved, naming the command (deviation 3).
func TestUnresolvedKey(t *testing.T) {
	c := newCfgEnv(t)
	if _, err := builtin.BashPath(builtin.Options{}); err != nil {
		t.Skipf("no bash to run the key's command with: %v", err)
	}
	c.write(filepath.Join(c.home, "models.json"), `{"providers": {"openai": {"apiKey": "!exit 3"}}}`)
	l := &liveLister{ids: []string{"gpt-6-astra"}}
	withLister(t, l.list)
	r := c.run("-p", "hello")
	if r.code != ExitAuth || !strings.Contains(r.stderr, `openai's API key from models.json could not be resolved: `) ||
		!strings.Contains(r.stderr, `the command "exit 3" failed`) || strings.Contains(r.stderr, "rejected") {
		t.Errorf("exit %d, stderr %q", r.code, r.stderr)
	}
	if n := l.calls.Load(); n != 0 {
		t.Errorf("%d listings were made with no key, want none", n)
	}
}

// rejecting is a provider whose every turn fails as a refused key does, its
// text holding what a provider might echo.
type rejecting struct{}

func (rejecting) ID() string                     { return "openai" }
func (rejecting) Capabilities() llm.Capabilities { return llm.Capabilities{} }
func (rejecting) Stream(context.Context, *llm.Request) iter.Seq2[llm.Event, error] {
	return func(yield func(llm.Event, error) bool) {
		yield(nil, fmt.Errorf("%w: incorrect API key provided: sk-ech***oed", llm.ErrAuth))
	}
}

// A refused key says the provider rejected the key from its source, and
// nothing of the provider's text; with no key, the no-credential message
// stays (0005-PLAN F4-3a decision 9, deviation 1).
func TestRejectedKeyMessage(t *testing.T) {
	old := modelProvider
	modelProvider = func(*runEnv) func(string) (llm.Provider, error) {
		return func(string) (llm.Provider, error) { return rejecting{}, nil }
	}
	t.Cleanup(func() { modelProvider = old })
	c := newCfgEnv(t)
	t.Setenv("OPENAI_API_KEY", "test-only")
	r := c.run("-p", "hello")
	if r.code != ExitAuth || !strings.Contains(r.stderr, "openai rejected the API key from OPENAI_API_KEY") || strings.Contains(r.stderr, "sk-ech") {
		t.Errorf("a rejected key: exit %d, stderr %q", r.code, r.stderr)
	}
	t.Setenv("OPENAI_API_KEY", "")
	if r := c.run("-p", "hello"); r.code != ExitAuth || !strings.Contains(r.stderr, "no model credential: set one of") {
		t.Errorf("no key: exit %d, stderr %q", r.code, r.stderr)
	}
}
