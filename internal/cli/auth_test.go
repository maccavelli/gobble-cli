package cli

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
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
// file's other entries; without --api-key -, a provider with no login is
// refused with its reason (0005-PLAN F4-3a decision 8, F4-3b decision 6).
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
	c.want(c.run("auth", "login", "openai"), ExitUsage, "", "openai's login waits for go-llmprovider-sdk's Sign in with ChatGPT for agents (0005-PLAN F4-3d); use --api-key -")
	c.want(c.run("auth", "login", "anthropic"), ExitUsage, "", "claude has no login; use --api-key -")
	c.want(c.runStdin(stdinFile(t, "k"), "auth", "login", "grok", "--api-key", "-", "--device"), ExitUsage, "", "--device is for a login")
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
	// A login is shown with its expiry; another provider's oauth entry says
	// gobble does not use it (0005-PLAN F4-3b decision 8).
	c.write(c.authFile(), `{"grok": {"type": "oauth", "access": "a-secret", "refresh": "r-secret", "expires": 1781000000000},
  "claude": {"type": "oauth", "access": "a-secret", "refresh": "r-secret", "expires": 1781000000000}}`)
	r = c.run("auth", "status")
	grok, claude := rowOf(r.stdout, "grok"), rowOf(r.stdout, "claude")
	if !strings.HasPrefix(grok, "grok          auth.json, an OAuth login, expires ") || strings.Contains(grok, "not used") ||
		!strings.HasSuffix(claude, ", not used (0005-PLAN F4-3b)") || strings.Contains(r.stdout+r.stderr, "secret") {
		t.Errorf("status with logins:\n%s%s", r.stdout, r.stderr)
	}
}

// rowOf is out's line for provider id, or "".
func rowOf(out, id string) string {
	for line := range strings.Lines(out) {
		if strings.HasPrefix(line, id+" ") {
			return strings.TrimSuffix(line, "\n")
		}
	}
	return ""
}

// gobble auth check is Pi's: ready, not_ready or invalid, with a reason,
// exiting 0, 1 or 2.
func TestAuthCheck(t *testing.T) {
	c := newCfgEnv(t)
	c.write(c.authFile(), `{"openai": {"type": "api_key", "key": "k"}, "claude": {"type": "oauth", "access": "a", "refresh": "r", "expires": 1},
  "grok": {"type": "oauth", "access": "a", "refresh": "r", "expires": 1}}`)
	c.want(c.run("auth", "check", "--provider", "openai"), ExitOK, "ready\n", "")
	c.want(c.run("auth", "check", "--provider", "kilo"), ExitFailure, "not_ready\n", "")
	c.want(c.run("auth", "check", "--model", "anthropic/claude-haiku-5-5"), ExitUsage, "invalid\n", "")
	c.want(c.run("auth", "check"), ExitUsage, "", "Auth checks require --provider <provider> or --model <model>")
	for args, want := range map[string]checkResult{
		"--provider openai": {Status: "ready", Provider: "openai", AuthType: "api_key"},
		"--provider kilo":   {Status: "not_ready", Provider: "kilo", Reason: "credentials_not_configured"},
		"--provider nope":   {Status: "not_ready", Provider: "nope", Reason: "provider_not_found"},
		"--provider claude": {Status: "invalid", Provider: "claude", Reason: "invalid_state"},
		"--provider grok":   {Status: "ready", Provider: "grok", AuthType: "oauth"},
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

// replace sets *v to fake for the test.
func replace[T any](t *testing.T, v *T, fake T) {
	old := *v
	*v = fake
	t.Cleanup(func() { *v = old })
}

// gobble auth login grok runs the browser login, printing its URL before
// opening it, or with --device the device login, showing its code; either
// saves the login (0005-PLAN F4-3b decision 6).
func TestAuthLoginGrok(t *testing.T) {
	c := newCfgEnv(t)
	var opened []string
	replace(t, &openURL, func(_ context.Context, u string) error { opened = append(opened, u); return nil })
	const authorize = "https://auth.x.ai/oauth2/authorize?state=s"
	replace(t, &browserLogin, func(ctx context.Context, id string, o provider.LoginOptions, store provider.LoginStore) error {
		if err := o.OpenURL(authorize); err != nil {
			return err
		}
		return store.Save(ctx, id, &provider.Login{Access: "a-browser", Refresh: "r", Issuer: "https://auth.x.ai"})
	})
	replace(t, &deviceLogin, func(ctx context.Context, id string, _ provider.LoginOptions, show func(provider.DeviceCode) error,
		store provider.LoginStore) error {
		if err := show(provider.DeviceCode{URL: "https://accounts.x.ai/device", Code: "ABCD-1234"}); err != nil {
			return err
		}
		return store.Save(ctx, id, &provider.Login{Access: "a-device", Refresh: "r"})
	})
	c.write(c.authFile(), `{"openai": {"type": "api_key", "key": "keep"}}`)

	c.want(c.run("auth", "login", "grok"), ExitOK, "Logged in to grok\n", "Open this URL to sign in to grok:\n  "+authorize+"\n")
	if len(opened) != 1 || opened[0] != authorize {
		t.Errorf("opened %q, want the authorize URL once", opened)
	}
	if got := c.read(c.authFile()); !strings.Contains(got, `"access": "a-browser"`) || !strings.Contains(got, `"key": "keep"`) {
		t.Errorf("auth.json after the browser login:\n%s", got)
	}
	c.want(c.run("auth", "login", "grok", "--device"), ExitOK, "Logged in to grok\n",
		"Open https://accounts.x.ai/device and enter the code ABCD-1234\n")
	if got := c.read(c.authFile()); !strings.Contains(got, `"access": "a-device"`) || len(opened) != 1 {
		t.Errorf("auth.json after the device login, %d opened:\n%s", len(opened), got)
	}
	replace(t, &browserLogin, func(context.Context, string, provider.LoginOptions, provider.LoginStore) error {
		return fmt.Errorf("grok login: %w", llm.ErrAuth)
	})
	if r := c.run("auth", "login", "grok"); r.code != ExitFailure || !strings.Contains(r.stderr, "grok login:") {
		t.Errorf("a failed login: exit %d, stderr %q", r.code, r.stderr)
	}
}

// gobble auth login kilo shows the device code and stores the key it gives,
// escaped, so Pi's grammar gives it back unchanged; a failed login stores
// nothing (0005-PLAN F4-3b decision 3 of the first entry).
func TestAuthLoginKilo(t *testing.T) {
	c := newCfgEnv(t)
	token := "!x$y"
	replace(t, &keyLogin, func(_ context.Context, _ string, _ provider.LoginOptions, show func(provider.DeviceCode) error) (string, error) {
		if err := show(provider.DeviceCode{URL: "https://app.kilo.ai/device", Code: "KILO-1"}); err != nil {
			return "", err
		}
		return token, nil
	})
	c.write(c.authFile(), `{"openai": {"type": "api_key", "key": "keep"}}`)
	c.want(c.run("auth", "login", "kilo"), ExitOK, "Saved API key for kilo\n", "Open https://app.kilo.ai/device and enter the code KILO-1\n")
	f, err := auth.Load(c.authFile())
	if err != nil {
		t.Fatal(err)
	}
	e, _ := f.Entry("kilo")
	got, err := catalog.ResolveValue(e.Key, func(string) (string, bool) { return "", false },
		func(string) (string, error) { return "", errors.New("ran a command") })
	if e.Type != auth.TypeAPIKey || e.Key != "$!x$$y" || got != token || err != nil {
		t.Errorf("kilo's entry %+v resolves to %q, %v; want %q stored escaped", e, got, err, token)
	}
	if o, _ := f.Entry("openai"); o.Key != "keep" {
		t.Errorf("openai's entry: %+v", o)
	}
	replace(t, &keyLogin, func(context.Context, string, provider.LoginOptions, func(provider.DeviceCode) error) (string, error) {
		return "", fmt.Errorf("kilo login: %w", llm.ErrAuth)
	})
	c.write(c.authFile(), `{}`)
	if r := c.run("auth", "login", "kilo"); r.code != ExitFailure || c.read(c.authFile()) != "{}" {
		t.Errorf("a failed login: exit %d, auth.json %q", r.code, c.read(c.authFile()))
	}
}

// gobble auth login --json prints decision 6's events, one JSON object a
// line on stdout, and nothing on stderr: the device and browser events,
// then saved; a failure is the error event, with its kind, the message
// gobble prints without --json, and the exit unchanged; a browser login
// offers no paste (0005-PLAN F4-3c decision 6).
func TestAuthLoginJSON(t *testing.T) {
	c := newCfgEnv(t)
	replace(t, &openURL, func(context.Context, string) error { return errors.New("no browser") })
	const authorize = "https://auth.x.ai/oauth2/authorize?state=s"
	var pasted []bool
	replace(t, &browserLogin, func(ctx context.Context, id string, o provider.LoginOptions, store provider.LoginStore) error {
		pasted = append(pasted, o.InputCode != nil)
		if err := o.OpenURL(authorize); err != nil {
			return err
		}
		return store.Save(ctx, id, &provider.Login{Access: "a-browser", Refresh: "r"})
	})
	replace(t, &deviceLogin, func(ctx context.Context, id string, _ provider.LoginOptions, show func(provider.DeviceCode) error,
		store provider.LoginStore) error {
		d := provider.DeviceCode{URL: "https://accounts.x.ai/device", Code: "ABCD-1234", Expires: time.Date(2026, 10, 15, 13, 0, 0, 0, time.FixedZone("", 3600))}
		if err := show(d); err != nil {
			return err
		}
		return store.Save(ctx, id, &provider.Login{Access: "a-device", Refresh: "r"})
	})
	replace(t, &keyLogin, func(_ context.Context, _ string, _ provider.LoginOptions, show func(provider.DeviceCode) error) (string, error) {
		return "k", show(provider.DeviceCode{URL: "https://app.kilo.ai/device", Code: "KILO-1"})
	})
	// A state directory that is a file fails the log file, whose warning
	// would reach stderr.
	c.write(filepath.Join(c.dir, "state-file"), "")
	t.Setenv("GOBBLE_STATE_DIR", filepath.Join(c.dir, "state-file"))
	for _, tc := range []struct {
		args  []string
		stdin string
		code  int
		lines []string
	}{
		{[]string{"grok", "--device"}, "", ExitOK, []string{
			`{"event":"device_code","provider":"grok","url":"https://accounts.x.ai/device","code":"ABCD-1234","expires":"2026-10-15T12:00:00Z"}`,
			`{"event":"saved","provider":"grok","type":"oauth"}`}},
		{[]string{"grok"}, "", ExitOK, []string{
			`{"event":"browser_url","provider":"grok","url":"` + authorize + `"}`,
			`{"event":"saved","provider":"grok","type":"oauth"}`}},
		{[]string{"kilo"}, "", ExitOK, []string{
			`{"event":"device_code","provider":"kilo","url":"https://app.kilo.ai/device","code":"KILO-1"}`,
			`{"event":"saved","provider":"kilo","type":"api_key"}`}},
		{[]string{"anthropic", "--api-key", "-"}, "sk-test-only\n", ExitOK, []string{`{"event":"saved","provider":"claude","type":"api_key"}`}},
		{[]string{"anthropic"}, "", ExitUsage, []string{`{"event":"error","provider":"claude","kind":"other","message":"claude has no login; use --api-key -"}`}},
		{[]string{"ollama"}, "", ExitUsage, []string{`{"event":"error","provider":"ollama","kind":"other","message":"unknown provider \"ollama\": one of ` +
			strings.Join(provider.KeyProviders(), ", ") + `"}`}},
	} {
		args := append([]string{"--log-level", "debug", "auth", "login", "--json"}, tc.args...)
		r := c.runStdin(stdinFile(t, tc.stdin), args...)
		if want := strings.Join(tc.lines, "\n") + "\n"; r.code != tc.code || r.stdout != want || r.stderr != "" {
			t.Errorf("%q: exit %d, stdout %q, stderr %q; want %d, %q, nothing", tc.args, r.code, r.stdout, r.stderr, tc.code, want)
		}
	}
	if got := c.read(c.authFile()); !strings.Contains(got, `"access": "a-browser"`) || !strings.Contains(got, `"key": "sk-test-only"`) {
		t.Errorf("auth.json:\n%s", got)
	}

	// A failed login's message is what gobble prints without --json.
	replace(t, &browserLogin, func(context.Context, string, provider.LoginOptions, provider.LoginStore) error {
		return fmt.Errorf("grok login: %w", llm.ErrAuth)
	})
	text := c.run("auth", "login", "grok")
	msg, ok := strings.CutPrefix(strings.TrimSuffix(text.stderr, "\n"), "Error: ")
	if text.code != ExitFailure || !ok {
		t.Fatalf("without --json: exit %d, stderr %q", text.code, text.stderr)
	}
	want, err := json.Marshal(map[string]string{"event": "error", "provider": "grok", "kind": "auth", "message": msg})
	if err != nil {
		t.Fatal(err)
	}
	r := c.run("auth", "login", "grok", "--json")
	var got, wantMap map[string]string
	if r.code != ExitFailure || r.stderr != "" || strings.Count(r.stdout, "\n") != 1 ||
		json.Unmarshal([]byte(r.stdout), &got) != nil || json.Unmarshal(want, &wantMap) != nil || !maps.Equal(got, wantMap) {
		t.Errorf("a failed login: exit %d, stdout %q, stderr %q; want 1 and %s", r.code, r.stdout, r.stderr, want)
	}

	// A settings warning goes to the log file, never stderr.
	t.Setenv("GOBBLE_STATE_DIR", filepath.Join(c.dir, "state"))
	if err := os.Remove(c.global()); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(c.global() + ".lock"); err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	if err := os.Mkdir(c.global()+".lock", 0o700); err != nil {
		t.Fatal(err)
	}
	r = c.runStdin(stdinFile(t, "k"), "auth", "login", "openai", "--api-key", "-", "--json")
	log, _ := os.ReadFile(filepath.Join(c.dir, "state", "logs", "gobble.log")) //nolint:errcheck // an unread log fails below
	if r.code != ExitOK || r.stderr != "" || !strings.Contains(string(log), "settings.yaml") {
		t.Errorf("a settings warning: exit %d, stderr %q, log %q", r.code, r.stderr, log)
	}

	// On a terminal, the browser login offers a paste only without --json.
	for _, j := range []bool{false, true} {
		var out, errw bytes.Buffer
		o := &Output{out: &out, err: &errw, caps: term.Caps{In: true}}
		e := &runEnv{ctx: t.Context(), out: o, stdin: stdinFile(t, ""), logs: &lazyLog{out: o, stderr: &errw}}
		pasted = nil
		replace(t, &browserLogin, func(ctx context.Context, id string, o provider.LoginOptions, store provider.LoginStore) error {
			pasted = append(pasted, o.InputCode != nil)
			return store.Save(ctx, id, &provider.Login{Access: "a", Refresh: "r"})
		})
		err := (&AuthLoginCmd{Provider: "grok", JSON: j}).Run(e)
		if cerr := e.logs.close(); cerr != nil {
			t.Error(cerr)
		}
		if err != nil || !slices.Equal(pasted, []bool{!j}) {
			t.Errorf("--json %t on a terminal: %v, paste offered %v", j, err, pasted)
		}
	}
}

// gobble auth logout grok revokes, then removes the login; a failed
// revocation is warned by its kind, and the login still removed (0005-PLAN
// F4-3b decision 7).
func TestAuthLogoutGrok(t *testing.T) {
	c := newCfgEnv(t)
	var revoke error
	logouts := 0
	replace(t, &logoutLogin, func(ctx context.Context, id string, _ provider.LoginOptions, store provider.LoginStore) (error, error) {
		logouts++
		return revoke, store.Delete(ctx, id)
	})
	login := `{"grok": {"type": "oauth", "access": "a", "refresh": "r-secret", "expires": 1}, "openai": {"type": "api_key", "key": "k"}}`
	c.write(c.authFile(), login)
	c.want(c.run("auth", "logout", "grok"), ExitOK, "Logged out of grok\n", "")
	if got := c.read(c.authFile()); strings.Contains(got, "grok") || !strings.Contains(got, "openai") || logouts != 1 {
		t.Errorf("after logout, %d revocations:\n%s", logouts, got)
	}
	revoke = fmt.Errorf("revoke: %w: 503 r-secret", llm.ErrUnavailable)
	c.write(c.authFile(), login)
	r := c.run("auth", "logout", "grok")
	c.want(r, ExitOK, "Logged out of grok\n", "Warning: grok's session could not be revoked (unavailable); it was removed from auth.json\n")
	if strings.Contains(r.stderr, "r-secret") || strings.Contains(c.read(c.authFile()), "grok") {
		t.Errorf("a failed revocation: stderr %q, auth.json %q", r.stderr, c.read(c.authFile()))
	}
}

// A login lists models as a key does: the listing is asked for with no key
// and the login store (0005-PLAN F4-3b decision 9).
func TestLoginListing(t *testing.T) {
	e, _ := modelEnv(t)
	if err := auth.SetOAuth(t.Context(), authPath(), "grok", auth.Entry{Access: "a", Refresh: "r", Expires: 1}); err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var got []string
	withLister(t, func(_ context.Context, id, key string, o provider.Options) (provider.Listing, error) {
		mu.Lock()
		defer mu.Unlock()
		got = append(got, fmt.Sprintf("%s key=%q logins=%t", id, key, o.Logins != nil))
		return provider.Listing{Live: true, Models: []string{"grok-4.6"}}, nil
	})
	if c := e.modelChoice(); c.Provider != "grok" || !slices.Equal(c.Models, []string{"grok-4.6"}) {
		t.Fatalf("model choice %+v, want grok's listing", c)
	}
	mu.Lock()
	defer mu.Unlock()
	if !slices.Equal(got, []string{`grok key="" logins=true`}) {
		t.Errorf("listed %q, want grok with its login", got)
	}
}

// A refused login says LoginRejected's sentence, and nothing of the
// provider's text (0005-PLAN F4-3b decision 8).
func TestLoginMessage(t *testing.T) {
	replace(t, &modelProvider, func(*runEnv) func(string) (llm.Provider, error) {
		return func(string) (llm.Provider, error) { return rejecting{}, nil }
	})
	c := newCfgEnv(t)
	c.write(c.authFile(), `{"grok": {"type": "oauth", "access": "a", "refresh": "r", "expires": 1}}`)
	r := c.run("-p", "hello")
	if r.code != ExitAuth || !strings.Contains(r.stderr, provider.LoginRejected("grok")) || strings.Contains(r.stderr, "sk-ech") {
		t.Errorf("a refused login: exit %d, stderr %q", r.code, r.stderr)
	}
}
