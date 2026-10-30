package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// chunks is a reader that owns stdin, as inline.Reader does: each read
// gives the next chunk, then the end of input.
type chunks struct{ c []string }

func (r *chunks) Read(ctx context.Context, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if len(r.c) == 0 {
		return 0, io.EOF
	}
	n := copy(p, r.c[0])
	r.c = r.c[1:]
	return n, nil
}

// asked is questions over chunks, writing to a buffer, with a raw mode
// that counts its entries and restores.
func asked(t *testing.T, in ...string) (*questions, *bytes.Buffer, *[2]int) {
	t.Helper()
	var out bytes.Buffer
	var raws [2]int
	q := &questions{ctx: t.Context(), in: &chunks{in}, out: &out, raw: func() (func() error, error) {
		raws[0]++
		return func() error { raws[1]++; return nil }, nil
	}}
	return q, &out, &raws
}

// A choice is numbered and read by its number, the default on an empty
// answer, and asked again when the answer names none; one option is
// chosen without asking (0025-MADR D6).
func TestQuestionsChoose(t *testing.T) {
	q, out, _ := asked(t, "9\n", "2\n")
	i, err := q.choose("Which?", []option{{"Alpha", "a detail"}, {"Beta", ""}}, 0)
	if err != nil || i != 1 {
		t.Fatalf("choose = %d, %v", i, err)
	}
	for _, want := range []string{"Which?\n  1  Alpha  a detail\n  2  Beta\n", "Number [1]: ", `No choice "9": enter a number from 1 to 2.`} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the choice lacks %q:\n%s", want, out)
		}
	}
	q, _, _ = asked(t, "\n")
	if i, err := q.choose("Which?", []option{{"A", ""}, {"B", ""}}, 1); err != nil || i != 1 {
		t.Fatalf("an empty answer chose %d, %v; want the default", i, err)
	}
	q, out, _ = asked(t)
	if i, err := q.choose("Which?", []option{{"Only", ""}}, 0); err != nil || i != 0 || out.Len() != 0 {
		t.Fatalf("one option: %d, %v, wrote %q", i, err, out)
	}
	q, _, _ = asked(t)
	if _, err := q.choose("Which?", []option{{"A", ""}, {"B", ""}}, 0); !errors.Is(err, errCancelled) {
		t.Fatalf("the end of input gave %v, want cancelled", err)
	}
}

// A yes or no takes y, yes, n, no and the default, and asks again
// otherwise; a line keeps no bracketed-paste markers, and a \n after a \r
// ends no second line.
func TestQuestionsConfirmAndInput(t *testing.T) {
	q, out, _ := asked(t, "maybe\n", "N\n", "\n")
	if ok, err := q.confirm("Go?", true); err != nil || ok {
		t.Fatalf("confirm = %v, %v; want no", ok, err)
	}
	if !strings.Contains(out.String(), "Go? [Y/n]: ") || !strings.Contains(out.String(), "Answer y or n.") {
		t.Fatalf("confirm wrote %q", out)
	}
	if ok, err := q.confirm("Go?", true); err != nil || !ok {
		t.Fatalf("an empty answer gave %v, %v; want the default", ok, err)
	}
	q, _, _ = asked(t, "\x1b[200~https://localhost/cb?code=1\x1b[201~\r", "\nnext\n")
	if s, err := q.input("URL"); err != nil || s != "https://localhost/cb?code=1" {
		t.Fatalf("input = %q, %v", s, err)
	}
	if s, err := q.input("Again"); err != nil || s != "next" {
		t.Fatalf("the line after a \\r\\n = %q, %v", s, err)
	}
}

// A secret is read in raw mode, restored after, each character echoed as
// *: Backspace erases one, an escape sequence and paste markers are not
// part of it, and the value is written nowhere; Ctrl+C and the end of input
// cancel (0025-PLAN deviation 7).
func TestQuestionsSecret(t *testing.T) {
	q, out, raws := asked(t, "sk-ab", "\x7f", "c\x1b[200~d\x1b[201~", "\x1b[Dé\r")
	s, err := q.secret("Key")
	if err != nil || s != "sk-acdé" {
		t.Fatalf("secret = %q, %v", s, err)
	}
	if want := "Key: *****\b \b***\n"; out.String() != want {
		t.Fatalf("the echo %q, want %q", out, want)
	}
	// Raw mode is on before the prompt, so the terminal echoes nothing typed
	// after it.
	q, out, _ = asked(t, "k\r")
	q.raw = func() (func() error, error) {
		if out.Len() != 0 {
			t.Errorf("the prompt %q was written before raw mode", out)
		}
		return func() error { return nil }, nil
	}
	if _, err := q.secret("Key"); err != nil {
		t.Fatal(err)
	}
	if *raws != [2]int{1, 1} {
		t.Fatalf("raw mode entered %d, restored %d; want once each", raws[0], raws[1])
	}
	for _, in := range [][]string{{"sk\x03"}, {"sk"}} {
		q, out, raws := asked(t, in...)
		if _, err := q.secret("Key"); !errors.Is(err, errCancelled) || raws[1] != 1 || strings.Contains(out.String(), "sk") {
			t.Fatalf("%q: %v, restored %d, wrote %q", in, err, raws[1], out)
		}
	}
}

// questions are the sign-in's prompter: Input gives up when its context
// ends and leaves what is typed next to the next question; an empty secret
// cancels; with no terminal, every question is refused and a notice still
// shown (0031-MADR D7).
func TestQuestionsPrompter(t *testing.T) {
	qs, out, _ := asked(t, "\r", "https://localhost/cb?code=1\n")
	q := prompter{qs}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := q.Input(ctx, "Paste"); !errors.Is(err, context.Canceled) {
		t.Fatalf("an ended context: %v", err)
	}
	if s, err := q.Input(t.Context(), "Paste"); err != nil || s != "" {
		t.Fatalf("the next answer = %q, %v; want the empty line typed next", s, err)
	}
	if s, err := q.Input(t.Context(), "Paste"); err != nil || s != "https://localhost/cb?code=1" {
		t.Fatalf("the paste = %q, %v", s, err)
	}
	if i, err := q.Choose("Which?", []provider.Choice{{Label: "Only"}}, 0); err != nil || i != 0 {
		t.Fatalf("one choice: %d, %v", i, err)
	}
	q.Notify("note %d", 1)
	if !strings.Contains(out.String(), "Paste: ") || !strings.HasSuffix(out.String(), "note 1\n") {
		t.Fatalf("wrote %q", out)
	}
	qs, _, _ = asked(t, "\r")
	if _, err := (prompter{qs}).Secret("Key"); !errors.Is(err, errCancelled) {
		t.Fatalf("an empty secret: %v, want cancelled", err)
	}
	var notes bytes.Buffer
	u := unasked{out: &notes}
	if _, err := u.Secret("Key"); !errors.Is(err, errNoTerminal) {
		t.Fatalf("unasked: %v", err)
	}
	if _, err := u.Choose("Which?", nil, 0); !errors.Is(err, errNoTerminal) {
		t.Fatalf("unasked: %v", err)
	}
	u.Notify("Open %s", "https://x.test")
	if notes.String() != "Open https://x.test\n" {
		t.Fatalf("unasked's notice %q", notes.String())
	}
}

// keyLister answers claude's listing by its key: live, offline, or
// refused, a failure in the listing as provider.List gives it; every other
// listing fails offline.
func keyLister(good string) func(context.Context, string, string, provider.Options) (provider.Listing, error) {
	return func(_ context.Context, id, key string, _ provider.Options) (provider.Listing, error) {
		switch {
		case id != "claude":
			return provider.Listing{}, errOfflineList
		case key == good:
			return provider.Listing{Live: true, Source: "live", Models: []string{"claude-x1", "claude-x2"}, FetchedAt: time.Now()}, nil
		case key == "sk-test-OFFLINE":
			return provider.Listing{Models: provider.Models("claude"), Err: fmt.Errorf("list: %w", llm.ErrOffline)}, nil
		}
		return provider.Listing{Models: provider.Models("claude"), Err: &llm.APIError{Status: 401, Err: fmt.Errorf("%w: refused", llm.ErrAuth)}}, nil
	}
}

// signInFunc is provider.SignIn's shape.
type signInFunc = func(context.Context, string, provider.Prompter, provider.SignInOptions, provider.LoginStore) (provider.Credential, error)

// typedKeySignIn is a sign-in that asks for a key through the prompter and
// gives it, as the wizard's key method does; it records the options.
func typedKeySignIn(got *provider.SignInOptions) signInFunc {
	return func(_ context.Context, id string, p provider.Prompter, o provider.SignInOptions, _ provider.LoginStore) (provider.Credential, error) {
		*got = o
		key, err := p.Secret("Enter your " + provider.Label(id) + " API key")
		return provider.Credential{Key: key}, err
	}
}

// storedKey is provider id's api_key in auth.json under GOBBLE_HOME, or "".
func storedKey(t *testing.T, id string) string {
	t.Helper()
	f, err := auth.Load(authPath())
	if err != nil {
		t.Fatal(err)
	}
	en, _ := f.Entry(id)
	return en.Key
}

// /login claude signs in through the SDK's wizard, choosing no method:
// the key it asks for is read as a secret, stored, the credentials are read
// again, and claude's models are listed, numbered for /model N. No key is
// ever shown (0025-MADR D6; 0031-MADR D7).
func TestLoginSlash(t *testing.T) {
	withLister(t, keyLister("sk-test-GOOD-2"))
	var o provider.SignInOptions
	replace(t, &signInWith, typedKeySignIn(&o))
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	h.start(Prompt{})
	h.press("/login claude\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your Claude (Anthropic) API key: ") })
	h.press("sk-test-GOOD-2\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "claude/claude-x2") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if got := storedKey(t, "claude"); got != "sk-test-GOOD-2" {
		t.Fatalf("auth.json's claude key %q", got)
	}
	if o.Method != "" || o.ClientName != appName {
		t.Errorf("the sign-in's options %+v, want no method and gobble's client", o)
	}
	if !strings.Contains(h.stdout.String(), "Saved API key for claude") || !strings.Contains(h.stdout.String(), "Claude (Anthropic)** — live") {
		t.Fatalf("stdout:\n%s", h.stdout)
	}
	if all := h.stdout.String() + h.stderr.String(); strings.Contains(all, "sk-test") {
		t.Fatalf("a key was shown:\n%s", all)
	}
	if h.e.credentialGeneration("claude") != 1 || !h.e.hasCredential("claude") {
		t.Fatal("the credentials were not read again")
	}
	if off, on := strings.Index(h.stdout.String(), pasteOff), strings.LastIndex(h.stdout.String(), pasteOn); off < 0 || on < off {
		t.Fatalf("bracketed paste was not off while /login asked: off at %d, on at %d", off, on)
	}
}

// Ctrl+C while /login asks cancels the sign-in, not gobble; nothing is
// saved and the session goes on.
func TestLoginInterrupt(t *testing.T) {
	var o provider.SignInOptions
	replace(t, &signInWith, typedKeySignIn(&o))
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	h.start(Prompt{})
	h.press("/login claude\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "API key: ") })
	if !h.e.intr.take() {
		t.Fatal("the sign-in did not take the interrupt")
	}
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Sign-in cancelled; nothing was saved.") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatalf("the session ended with %v", err)
	}
	if got := storedKey(t, "claude"); got != "" {
		t.Fatalf("a key was stored: %q", got)
	}
}

// /login with no provider offers every provider that takes a key, with
// its credential's source; a sign-in the provider refused says so in
// gobble's words, with nothing saved.
func TestLoginMenuAndRefusals(t *testing.T) {
	var signedIn string
	replace(t, &signInWith, func(_ context.Context, id string, _ provider.Prompter, _ provider.SignInOptions, _ provider.LoginStore) (provider.Credential, error) {
		signedIn = id
		return provider.Credential{}, &provider.SignInError{ID: id, Keys: 3}
	})
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	h.start(Prompt{})
	h.press("/login\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
	n := slices.Index(provider.KeyProviders(), "claude") + 1
	for _, want := range []string{"Sign in to which provider?", fmt.Sprintf("  %d  Claude (Anthropic) (claude)", n), "none — /login claude", "Together AI (together)"} {
		if !strings.Contains(h.stderr.String(), want) {
			t.Errorf("the menu lacks %q:\n%s", want, h.stderr)
		}
	}
	h.press(fmt.Sprintf("%d\r", n))
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "claude refused 3 keys; none was saved") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if signedIn != "claude" || storedKey(t, "claude") != "" {
		t.Fatalf("signed in to %q; stored %q", signedIn, storedKey(t, "claude"))
	}
}

// The wizard's questions are the line session's: a menu numbered, a notice
// on its own line; a login the wizard saved is reported as one. A key from
// a public listing, typed, is noted as not checked (0031-MADR D7).
func TestLoginAsksThroughQuestions(t *testing.T) {
	replace(t, &signInWith, func(ctx context.Context, id string, p provider.Prompter, _ provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		if id == "kilo" {
			key, err := p.Secret("Enter your Kilo key")
			return provider.Credential{Key: key}, err
		}
		i, err := p.Choose("Choose how to authenticate with Grok:", []provider.Choice{{Label: "Key"}, {Label: "Browser"}, {Label: "Device", Detail: "a code"}}, 0)
		if err != nil || i != 2 {
			return provider.Credential{}, fmt.Errorf("chose %d: %w", i, err)
		}
		p.Notify("Open %s and enter the code %s", "https://example.test/device", "ABCD-1234")
		return provider.Credential{}, store.Save(ctx, id, &provider.Login{Access: "a", Refresh: "r"})
	})
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	h.start(Prompt{})
	h.press("/login grok\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
	if !strings.Contains(h.stderr.String(), "Choose how to authenticate with Grok:\n  1  Key\n  2  Browser\n  3  Device   a code\n") {
		t.Errorf("the menu:\n%s", h.stderr)
	}
	h.press("3\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Logged in to grok") })
	h.press("/login kilo\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your Kilo key: ") })
	h.press("kilo-typed\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Saved API key for kilo") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"Open https://example.test/device and enter the code ABCD-1234\n",
		provider.Label("kilo") + " lists its models without a key, so the key was not checked.\n"} {
		if !strings.Contains(h.stderr.String(), want) {
			t.Errorf("stderr lacks %q:\n%s", want, h.stderr)
		}
	}
	if storedKey(t, "kilo") != "kilo-typed" {
		t.Errorf("kilo's key %q", storedKey(t, "kilo"))
	}
}

// /logout lists only what auth.json holds, revokes a Grok session through
// gobble's logout, and says a variable still set keeps a credential; a
// second /logout of one already removed says there is none.
func TestLogoutSlash(t *testing.T) {
	var revoked string
	old := logoutLogin
	logoutLogin = func(ctx context.Context, id string, _ provider.LoginOptions, _ provider.LoginStore) (error, error) {
		revoked = id
		_, err := auth.Delete(ctx, authPath(), id)
		return nil, err
	}
	t.Cleanup(func() { logoutLogin = old })
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	t.Setenv("XAI_API_KEY", "test-only")
	t.Setenv("OPENAI_API_KEY", "test-only")
	writeFile(t, authPath(), `{"claude": {"type": "api_key", "key": "k"}, "grok": {"type": "oauth", "access": "a", "refresh": "r", "expires": 1}}`)
	h.start(Prompt{})
	h.press("/logout\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
	menu := h.stderr.String()
	if !strings.Contains(menu, "  1  Claude (Anthropic) (claude)  API key") || !strings.Contains(menu, "  2  Grok (xAI) (grok)") ||
		strings.Contains(menu, "  3  ") {
		t.Fatalf("the menu:\n%s", menu)
	}
	h.press("2\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Logged out of grok") })
	h.press("/logout claude\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Removed stored API key for claude") })
	h.press("/logout claude\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "No stored credential for claude to remove") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if revoked != "grok" || !strings.Contains(h.stdout.String(), "Logged out of grok XAI_API_KEY is still set, so grok keeps a credential.") {
		t.Fatalf("revoked %q; stdout:\n%s", revoked, h.stdout)
	}
}

// In the line session, a /model to a provider with no credential says how
// to sign in there: /login (0025-MADR D1).
func TestModelOffersLogin(t *testing.T) {
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	t.Setenv("OPENAI_API_KEY", "test-only")
	h.start(Prompt{})
	h.press("/model claude/claude-sonnet-5-5\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "claude is not usable") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.stdout.String(), "no credential — set ANTHROPIC_API_KEY, or /login claude") {
		t.Fatalf("stdout:\n%s", h.stdout)
	}
}

// With no credential at all, the line session offers the sign-in before
// its first prompt; taken, it signs in and lists the provider's models;
// declined, the session goes on with the warning (0025-MADR D7).
func TestFirstRunSetup(t *testing.T) {
	withLister(t, keyLister("sk-test-GOOD"))
	var o provider.SignInOptions
	replace(t, &signInWith, typedKeySignIn(&o))
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	h.answer = "\r"
	h.start(Prompt{})
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Sign in to which provider?") })
	h.press(fmt.Sprintf("%d\r", slices.Index(provider.KeyProviders(), "claude")+1))
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your Claude (Anthropic) API key: ") })
	h.press("sk-test-GOOD\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "claude/claude-x1") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h.stderr.String(), firstRunQuestion+" [Y/n]: ") || strings.Contains(h.stderr.String(), "Warning: no model credential") {
		t.Fatalf("stderr:\n%s", h.stderr)
	}

	d := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	d.start(Prompt{}) // declines
	waitUntil(t, func() bool { return strings.Contains(d.stderr.String(), "Warning: no model credential") })
	d.press("\x04")
	if err := d.wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(d.stderr.String(), firstRunQuestion) {
		t.Fatalf("the sign-in was not offered:\n%s", d.stderr)
	}
}

// gobble setup asks on a terminal, and is refused without one.
func TestSetupNeedsTerminal(t *testing.T) {
	r := runIn(t, t.TempDir(), "setup")
	if r.code != ExitUsage || !strings.Contains(r.stderr, "gobble setup asks its questions on a terminal") {
		t.Fatalf("setup without a terminal: %+v", r)
	}
}

// gobble auth login --method signs in with no menu: api-key reads the key
// from stdin; device runs the device sign-in; a method the provider does
// not have is refused, naming its methods (0025-MADR D6; 0031-MADR D7).
func TestAuthLoginMethod(t *testing.T) {
	c := newCfgEnv(t)
	c.want(c.runStdin(stdinFile(t, "sk-test-only\n"), "auth", "login", "claude", "--method", "api-key"), ExitOK, "Saved API key for claude\n", "")
	var methods []string
	replace(t, &signInWith, func(ctx context.Context, id string, _ provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		methods = append(methods, o.Method)
		return provider.Credential{}, store.Save(ctx, id, &provider.Login{Access: "a", Refresh: "r"})
	})
	c.want(c.run("auth", "login", "grok", "--method", "device"), ExitOK, "Logged in to grok\n", "")
	c.want(c.run("auth", "login", "grok", "--method", "browser"), ExitOK, "Logged in to grok\n", "")
	if !slices.Equal(methods, []string{provider.MethodDevice, provider.MethodBrowser}) {
		t.Fatalf("the sign-ins ran with %q", methods)
	}
	r := c.run("auth", "login", "grok", "--method", "nope")
	if r.code != ExitUsage || !strings.Contains(r.stderr, `grok signs in with --method api-key or browser or device, not "nope"`) {
		t.Fatalf("--method nope: %+v", r)
	}
	if r := c.run("auth", "login", "claude", "--method", "browser"); r.code != ExitUsage || !strings.Contains(r.stderr, "--method api-key") {
		t.Fatalf("claude --method browser: %+v", r)
	}
	if r := c.run("auth", "login", "grok", "--method", "device", "--device"); r.code != ExitUsage {
		t.Fatalf("--method with --device: %+v", r)
	}
}

// At start only the run's provider's failed listing warns; each failure is
// a record in the log file (0025-MADR D4; 0020-MADR D6).
func TestStartWarnsRunProviderOnly(t *testing.T) {
	e, warnings := modelEnv(t)
	for _, v := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "XAI_API_KEY"} {
		t.Setenv(v, "test-only")
	}
	settled(t, e)
	if n := strings.Count(warnings.String(), "model listing failed"); n != 1 || !strings.Contains(warnings.String(), "provider=openai") {
		t.Fatalf("%d warnings at start, want openai's alone:\n%s", n, warnings)
	}
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dirs.State, "logs", "gobble.log"))
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"openai", "claude", "grok"} {
		if !strings.Contains(string(b), `"provider":"`+id+`"`) {
			t.Errorf("the log file has no record of %s's listing:\n%s", id, b)
		}
	}
}
