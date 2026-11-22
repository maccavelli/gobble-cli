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
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
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

// asked is questions over chunks, through a terminal input with no
// terminal to flush (0063-PLAN P2 step 4), writing to a buffer, with a raw
// mode that counts its entries and restores.
func asked(t *testing.T, in ...string) (*questions, *bytes.Buffer, *[2]int) {
	t.Helper()
	var out bytes.Buffer
	var raws [2]int
	q := &questions{ctx: t.Context(), in: newTerminalInput(&chunks{in}, nil), out: &out, raw: func() (func() error, error) {
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
	eachEnter(t, loginMenuAndRefusals)
}

func loginMenuAndRefusals(t *testing.T, enter string) {
	var signedIn string
	replace(t, &signInWith, func(_ context.Context, id string, _ provider.Prompter, _ provider.SignInOptions, _ provider.LoginStore) (provider.Credential, error) {
		signedIn = id
		return provider.Credential{}, &provider.SignInError{ID: id, Keys: 3}
	})
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	h.enter = enter
	h.start(Prompt{})
	h.press("/login\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
	n := slices.Index(provider.KeyProviders(), "claude") + 1
	for _, want := range []string{"Sign in to which provider?", fmt.Sprintf("  %d  Claude (Anthropic) (claude)", n), "none — /login claude", "Together AI (together)"} {
		if !strings.Contains(h.stderr.String(), want) {
			t.Errorf("the menu lacks %q:\n%s", want, h.stderr)
		}
	}
	h.press(fmt.Sprintf("%d%s", n, enter))
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
	eachEnter(t, loginAsksThroughQuestions)
}

func loginAsksThroughQuestions(t *testing.T, enter string) {
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
	h.enter = enter
	h.start(Prompt{})
	h.press("/login grok\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
	if !strings.Contains(h.stderr.String(), "Choose how to authenticate with Grok:\n  1  Key\n  2  Browser\n  3  Device   a code\n") {
		t.Errorf("the menu:\n%s", h.stderr)
	}
	h.press("3" + enter)
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
	eachEnter(t, firstRunSetup)
}

func firstRunSetup(t *testing.T, enter string) {
	withLister(t, keyLister("sk-test-GOOD"))
	var o provider.SignInOptions
	replace(t, &signInWith, typedKeySignIn(&o))
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	h.enter = enter
	h.answer = enter
	h.start(Prompt{})
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Sign in to which provider?") })
	h.press(fmt.Sprintf("%d%s", slices.Index(provider.KeyProviders(), "claude")+1, enter))
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
	// A12 (row R71): a sign-in that succeeds ends the first run; the offer
	// was asked once.
	if n := offerCount(h); n != 1 {
		t.Fatalf("the offer was asked %d times; want once:\n%s", n, h.stderr)
	}

	d := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	d.enter = enter
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

// eachEnter runs f once per line end a terminal sends for Enter at a
// question read out of raw mode: CR, LF (a POSIX terminal's cooked Enter)
// and CRLF (a Windows console's line read) (0063-PLAN P1 step 2).
func eachEnter(t *testing.T, f func(t *testing.T, enter string)) {
	t.Helper()
	for _, e := range []struct{ name, enter string }{{"CR", "\r"}, {"LF", "\n"}, {"CRLF", "\r\n"}} {
		t.Run(e.name, func(t *testing.T) { f(t, e.enter) })
	}
}

// errKeyRecorded is what recordingPrompter gives the sign-in once a key was
// typed: the real wizard stops there, before it lists anything.
var errKeyRecorded = errors.New("the test stopped the sign-in once the key was read")

// recordedSecret is what one Secret gave.
type recordedSecret struct {
	key string
	err error
}

// recordingPrompter wraps a sign-in's prompter and records what each
// Secret returned, so a test runs the real provider.SignIn and sees the key
// it was given. A key read is not handed on: the sign-in ends with
// errKeyRecorded, so nothing is listed or saved.
type recordingPrompter struct {
	provider.Prompter
	mu      sync.Mutex
	secrets []recordedSecret
}

func (r *recordingPrompter) Secret(prompt string) (string, error) {
	s, err := r.Prompter.Secret(prompt)
	r.mu.Lock()
	r.secrets = append(r.secrets, recordedSecret{s, err})
	r.mu.Unlock()
	if err != nil {
		return s, err
	}
	return "", errKeyRecorded
}

func (r *recordingPrompter) recorded() []recordedSecret {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.secrets)
}

// realSignIn replaces signInWith with the real provider.SignIn behind a
// recordingPrompter, which it returns.
func realSignIn(t *testing.T) *recordingPrompter {
	t.Helper()
	rp := &recordingPrompter{}
	replace(t, &signInWith, func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		rp.mu.Lock()
		rp.Prompter = p
		rp.mu.Unlock()
		return provider.SignIn(ctx, id, rp, o, store)
	})
	return rp
}

// wantKey waits for the first secret the sign-in read, and checks it is
// key, read with no error.
func wantKey(t *testing.T, rp *recordingPrompter, key string) {
	t.Helper()
	waitUntil(t, func() bool { return len(rp.recorded()) > 0 })
	got := rp.recorded()
	if len(got) == 0 {
		t.Fatal("no secret was asked")
	}
	if got[0].key != key || got[0].err != nil {
		t.Fatalf("secret = %q, %v; want %q", got[0].key, got[0].err, key)
	}
}

// providerNumber is id's number in the provider menu.
func providerNumber(id string) int { return slices.Index(provider.KeyProviders(), id) + 1 }

// offerCount counts the first-run offer in the harness's stderr.
func offerCount(h *lineHarness) int { return strings.Count(h.stderr.String(), firstRunQuestion) }

// warningCount counts the no-credential warning in the harness's stderr.
func warningCount(h *lineHarness) int {
	return strings.Count(h.stderr.String(), "Warning: no model credential")
}

// methodItem is the number of the item naming an API key in the last menu
// printed in s.
func methodItem(t *testing.T, s string) string {
	t.Helper()
	menu := s[strings.LastIndex(s[:strings.LastIndex(s, "Number [")], "\n\n")+1:]
	for line := range strings.Lines(menu) {
		f := strings.Fields(line)
		if len(f) > 1 && strings.Contains(strings.ToLower(line), "api key") {
			if _, err := fmt.Sscan(f[0], new(int)); err == nil {
				return f[0]
			}
		}
	}
	t.Fatalf("no API key item in the menu:\n%s", menu)
	return ""
}

// endSession closes the harness's input and waits for the session's end.
func endSession(t *testing.T, h *lineHarness) error {
	t.Helper()
	if err := h.keys.Close(); err != nil {
		t.Fatal(err)
	}
	return h.wait()
}

// The first run's key prompt, after the offer and the provider menu, waits
// for the key, with each line end (the owner's report). Red for the
// defect with CRLF: the secret is cancelled before the key is pressed.
func TestFirstRunKeyOnlyProviderWaits(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		rp := realSignIn(t)
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.enter, h.answer = enter, "y"+enter
		h.start(Prompt{})
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
		h.press(strconv.Itoa(providerNumber("gemini")) + enter)
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your Gemini (Google) API key: ") })
		h.press("sk-test-typed\r")
		wantKey(t, rp, "sk-test-typed")
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
	})
}

// As above, through a method menu: OpenAI's, answered by the item naming
// an API key. Red for the defect with CRLF.
func TestFirstRunMethodMenuProviderWaits(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		rp := realSignIn(t)
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.enter, h.answer = enter, "y"+enter
		h.start(Prompt{})
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
		h.press(strconv.Itoa(providerNumber("openai")) + enter)
		waitUntil(t, func() bool { return strings.Count(h.stderr.String(), "Number [") == 2 })
		h.press(methodItem(t, h.stderr.String()) + enter)
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your OpenAI API key: ") })
		h.press("sk-test-typed\r")
		wantKey(t, rp, "sk-test-typed")
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
	})
}

// /login with no provider: the menu, then the key prompt waits. Red for
// the defect with CRLF. /login gemini, with no menu, waits: a guard.
func TestLoginNoArgumentMenuWaits(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		rp := realSignIn(t)
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.enter = enter
		h.start(Prompt{})
		h.press("/login\r")
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
		h.press(strconv.Itoa(providerNumber("gemini")) + enter)
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your Gemini (Google) API key: ") })
		h.press("sk-test-typed\r")
		wantKey(t, rp, "sk-test-typed")
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("named", func(t *testing.T) {
		rp := realSignIn(t)
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.start(Prompt{})
		h.press("/login gemini\r")
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your Gemini (Google) API key: ") })
		h.press("sk-test-typed\r")
		wantKey(t, rp, "sk-test-typed")
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
	})
}

// A5 (row R21): /login openai, with a method menu: the key prompt waits
// after it. Red for the defect with CRLF.
func TestLoginProviderMethodMenuWaits(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		rp := realSignIn(t)
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.enter = enter
		h.start(Prompt{})
		h.press("/login openai\r")
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
		h.press(methodItem(t, h.stderr.String()) + enter)
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your OpenAI API key: ") })
		h.press("sk-test-typed\r")
		wantKey(t, rp, "sk-test-typed")
		if err := endSession(t, h); err != nil {
			t.Fatal(err)
		}
	})
}

// pipeEnv is a runEnv on a terminal whose input is a pipe: caps.In true,
// stdin the pipe's read end, raw mode stubbed by termRaw. It returns the
// pipe's write end, where a test types.
func pipeEnv(t *testing.T) (*runEnv, *os.File, *syncBuffer) {
	t.Helper()
	t.Setenv("GOBBLE_HOME", t.TempDir())
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = w.Close()
		_ = r.Close()
	})
	replace(t, &termRaw, func(*os.File) (func() error, error) { return func() error { return nil }, nil })
	stdout, stderr := &syncBuffer{}, &syncBuffer{}
	caps := term.Caps{In: true, Out: term.Stream{TTY: true, Width: 80}, Err: term.Stream{Width: 80}}
	out := &Output{out: stdout, err: stderr, caps: caps}
	e := &runEnv{ctx: t.Context(), out: out, stdin: r, logs: &lazyLog{out: out}, intr: &interrupts{}, now: time.Now}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	ownListings(t, e)
	return e, w, stderr
}

// typeInto writes s to w.
func typeInto(t *testing.T, w *os.File, s string) {
	t.Helper()
	if _, err := io.WriteString(w, s); err != nil {
		t.Fatal(err)
	}
}

// runAsync runs f and returns its result on a channel.
func runAsync(f func() error) <-chan error {
	done := make(chan error, 1)
	go func() { done <- f() }()
	return done
}

// waitDone waits for done within the harness's bound.
func waitDone(t *testing.T, done <-chan error, stderr *syncBuffer) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(10 * time.Second):
		t.Fatalf("the command did not end; stderr %q", stderr.String())
		return nil
	}
}

// gobble setup, on a terminal: the key prompt waits after the provider
// menu. Red for the defect with CRLF.
func TestSetupKeyWaits(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		rp := realSignIn(t)
		e, w, stderr := pipeEnv(t)
		done := runAsync(func() error { return (&SetupCmd{}).Run(e) })
		waitUntil(t, func() bool { return strings.Contains(stderr.String(), "Number [1]: ") })
		typeInto(t, w, strconv.Itoa(providerNumber("gemini"))+enter)
		waitUntil(t, func() bool { return strings.Contains(stderr.String(), "Enter your Gemini (Google) API key: ") })
		typeInto(t, w, "sk-test-typed\r")
		wantKey(t, rp, "sk-test-typed")
		_ = w.Close()
		_ = waitDone(t, done, stderr)
	})
}

// gobble acp --login openai, on a terminal: the key prompt waits after the
// method menu. Red for the defect with CRLF. A13 (row R72): Ctrl+C at the
// key prompt cancels with acp's words, and nothing offers a sign-in again.
func TestAcpLoginMethodMenuWaits(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		rp := realSignIn(t)
		e, w, stderr := pipeEnv(t)
		done := runAsync(func() error { return e.acpLogin("openai", false) })
		waitUntil(t, func() bool { return strings.Contains(stderr.String(), "Number [1]: ") })
		typeInto(t, w, methodItem(t, stderr.String())+enter)
		waitUntil(t, func() bool { return strings.Contains(stderr.String(), "Enter your OpenAI API key: ") })
		typeInto(t, w, "sk-test-typed\r")
		wantKey(t, rp, "sk-test-typed")
		_ = w.Close()
		_ = waitDone(t, done, stderr)
	})
	t.Run("cancel", func(t *testing.T) {
		realSignIn(t)
		e, w, stderr := pipeEnv(t)
		done := runAsync(func() error { return e.acpLogin("openai", false) })
		waitUntil(t, func() bool { return strings.Contains(stderr.String(), "Number [1]: ") })
		typeInto(t, w, methodItem(t, stderr.String())+"\r")
		waitUntil(t, func() bool { return strings.Contains(stderr.String(), "Enter your OpenAI API key: ") })
		typeInto(t, w, "\x03")
		err := waitDone(t, done, stderr)
		if err == nil || !strings.Contains(err.Error(), "sign-in cancelled; nothing was saved") {
			t.Fatalf("acp --login after Ctrl+C: %v", err)
		}
		if strings.Contains(stderr.String(), firstRunQuestion) {
			t.Fatalf("acp --login offered a sign-in:\n%s", stderr)
		}
	})
}

// Weakness (a): the offer's answer and the menu's, typed ahead in one
// press, both count: the key prompt appears with no second press. Red for
// the defect with every line end: the menu waits, the typed-ahead answer
// dropped with the first ask. (2026-11-21, P2, the P1 review's follow-up 1:
// the menu is answered with Claude, not its default, Gemini, so a stray
// line end answering the menu shows; and nothing follows the key prompt.)
func TestFirstRunTypeAhead(t *testing.T) {
	eachEnter(t, func(t *testing.T, enter string) {
		var o provider.SignInOptions
		replace(t, &signInWith, typedKeySignIn(&o))
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.enter = enter
		h.answer = "y" + enter + strconv.Itoa(providerNumber("claude")) + enter
		h.start(Prompt{})
		const keyPrompt = "Enter your Claude (Anthropic) API key: "
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), keyPrompt) })
		time.Sleep(300 * time.Millisecond)
		if s := h.stderr.String(); !strings.HasSuffix(s, keyPrompt) {
			t.Errorf("something followed the key prompt:\n%s", s)
		}
		_ = endSession(t, h)
	})
}

// Q3: keys typed past a key in one press are dropped when the session's
// prompt reads again; the agent never receives them as a prompt.
func TestSignInTypeAheadNotSentAsPrompt(t *testing.T) {
	withLister(t, keyLister("sk-test-GOOD"))
	var o provider.SignInOptions
	replace(t, &signInWith, typedKeySignIn(&o))
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{}, {}}}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	h.press("/login claude\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your Claude (Anthropic) API key: ") })
	h.press("sk-test-GOOD\rhello\r")
	waitUntil(t, func() bool { return storedKey(t, "claude") == "sk-test-GOOD" })
	waitUntil(t, func() bool { return len(agent.Prompts()) >= 1 }) // /login's /models turn
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	for _, p := range agent.Prompts() {
		if strings.Contains(p, "hello") {
			t.Fatalf("the agent received the typed-ahead keys as a prompt: %q", agent.Prompts())
		}
	}
}

// Q4: Enter on an empty key prompt cancels the sign-in, with today's words.
func TestEmptyKeyCancels(t *testing.T) {
	var o provider.SignInOptions
	replace(t, &signInWith, typedKeySignIn(&o))
	h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
	h.start(Prompt{})
	h.press("/login gemini\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your Gemini (Google) API key: ") })
	h.press("\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Sign-in cancelled; nothing was saved.") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
}

// A6 (row R56): keys typed in the same press as /login stay the editor's:
// the provider menu waits and does not take them, and they are the next
// prompt the agent receives.
func TestEditorKeepsKeysTypedWithLogin(t *testing.T) {
	var o provider.SignInOptions
	replace(t, &signInWith, typedKeySignIn(&o))
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{}, {}}}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	h.press("/login\r1\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Number [1]: ") })
	time.Sleep(300 * time.Millisecond)
	if strings.Contains(h.stderr.String(), "API key: ") || strings.Contains(h.stderr.String(), "No choice") {
		t.Fatalf("the provider menu took the keys typed with /login:\n%s", h.stderr)
	}
	h.press(strconv.Itoa(providerNumber("gemini")) + "\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Enter your Gemini (Google) API key: ") })
	h.press("sk-test-typed\r")
	waitUntil(t, func() bool { return len(agent.Prompts()) == 2 })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	p := agent.Prompts()
	if len(p) != 2 || !strings.Contains(p[0], "/models gemini") || !strings.Contains(p[1], `"text":"1"`) {
		t.Fatalf("prompts %q; want /login's /models turn, then 1", p)
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
