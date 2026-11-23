package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// planRun records what a ChatGPT plan sign-in did, in order: each question
// it asked, each request it made, each notice, and the URL it opened.
type planRun struct {
	mu     sync.Mutex
	events []string
}

func (r *planRun) add(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, fmt.Sprintf(format, args...))
}

func (r *planRun) seen() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.events)
}

// RoundTrip is the plan issuer's wire here: it records the request and
// publishes no discovery document, so the sign-in goes on from the SDK's
// built-in endpoints for OpenAI, and any other request fails. No request
// leaves the process.
func (r *planRun) RoundTrip(req *http.Request) (*http.Response, error) {
	r.add("http %s %s", req.Method, req.URL.Path)
	return nil, errors.New("the test's plan issuer answers nothing")
}

// errAtBrowser ends a plan sign-in when it opens its URL.
var errAtBrowser = errors.New("the test stops the sign-in at the browser")

// planPrompter is a Prompter that records what it is asked, and asks the
// surface's own.
type planPrompter struct {
	provider.Prompter
	run *planRun
}

func (p planPrompter) Choose(title string, choices []provider.Choice, def int) (int, error) {
	p.run.add("choose %s", title)
	return p.Prompter.Choose(title, choices, def)
}

func (p planPrompter) Notify(format string, args ...any) {
	p.run.add("notify "+format, args...)
	p.Prompter.Notify(format, args...)
}

// planSignIn is provider.SignIn against the test's plan issuer: the real
// wizard, the surface's prompter, and a browser that records its URL and ends
// the sign-in.
func planSignIn(run *planRun) signInFunc {
	return func(ctx context.Context, id string, p provider.Prompter, o provider.SignInOptions, store provider.LoginStore) (provider.Credential, error) {
		o.HTTPClient = &http.Client{Transport: run}
		o.OpenURL = func(u string) error {
			run.add("url %s", u)
			return errAtBrowser
		}
		return provider.SignIn(ctx, id, planPrompter{p, run}, o, store)
	}
}

// wrote checks the wording is in what the surface wrote to its stderr, where
// its own prompter puts a question: the text the user reads, not only the
// text put to the prompter.
func wrote(t *testing.T, surface string, e *runEnv) {
	t.Helper()
	b, ok := e.out.err.(*bytes.Buffer)
	if !ok {
		t.Fatalf("%s: stderr is %T, not the test's buffer", surface, e.out.err)
	}
	if want := provider.ConsentText["chatgpt_plan.background_use"]; !strings.Contains(b.String(), want) {
		t.Errorf("%s wrote\n%s\nwant it to hold gobble's wording", surface, b)
	}
}

// shownFirst checks that run asked the consent's question, in gobble's
// wording, and opened no URL: the test answers by cancelling the sign-in,
// or no terminal could answer, so none came before the wording and none came
// at all.
func shownFirst(t *testing.T, surface string, run *planRun) {
	t.Helper()
	want := provider.ConsentText["chatgpt_plan.background_use"]
	events := run.seen()
	asked, url := -1, -1
	for i, e := range events {
		if asked < 0 && strings.HasPrefix(e, "choose ") && strings.Contains(e, want) {
			asked = i
		}
		if url < 0 && strings.HasPrefix(e, "url ") {
			url = i
		}
	}
	switch {
	case asked < 0:
		t.Errorf("%s: gobble's wording was never asked: %q", surface, events)
	case url >= 0:
		t.Errorf("%s: the sign-in opened a URL (event %d) with the wording at event %d, though it was cancelled or could not be asked: %q", surface, url, asked, events)
	}
	for _, e := range events {
		if strings.HasPrefix(e, "http ") && !strings.Contains(e, "/.well-known/") {
			t.Errorf("%s: a request beyond the issuer's discovery: %s", surface, e)
		}
	}
}

// Every surface that signs in to OpenAI's ChatGPT plan, against the test's
// plan issuer, shows gobble's wording (0052-MADR D2) before any URL: gobble
// auth login, /login, gobble setup, the first run's offer, and gobble acp
// --login, each on a terminal, where the test cancels the sign-in at the
// question; and with no terminal, where a question cannot be asked, the
// sign-in fails closed after the wording was put to its prompter, with no URL
// opened and nothing saved.
func TestConsentWordingOnEverySurface(t *testing.T) {
	openai := slices.Index(provider.KeyProviders(), "openai") + 1
	const (
		plan   = "2\r" // OpenAI's menu: its key, then its ChatGPT plan
		cancel = "3\r" // the wizard's consent: allow, do not allow, cancel the sign-in
		// here answers the question go-llmprovider-sdk v1.32.0 asks after the
		// method menu and before the consent: the browser is on this machine
		// (0068-MADR W6, F14).
		here = "1\r"
	)
	methodMenu := func(h *lineHarness) {
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Choose how to authenticate with OpenAI:") })
		h.press(plan)
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Where is the browser you will sign in with?") })
		h.press(here)
	}

	t.Run("/login", func(t *testing.T) {
		run := &planRun{}
		replace(t, &signInWith, planSignIn(run))
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.start(Prompt{})
		h.press("/login openai\r")
		methodMenu(h)
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Allow gobble to make background requests") })
		h.press(cancel)
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "openai sign-in failed (other)") })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		shownFirst(t, "/login", run)
	})

	t.Run("the first run", func(t *testing.T) {
		run := &planRun{}
		replace(t, &signInWith, planSignIn(run))
		h := newModelHarness(t, llmtest.NewScript(), t.TempDir())
		h.answer = "\r"
		h.start(Prompt{})
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Sign in to which provider?") })
		h.press(fmt.Sprintf("%d\r", openai))
		methodMenu(h)
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Allow gobble to make background requests") })
		h.press(cancel)
		waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "openai sign-in failed (other)") })
		// (2026-11-21, 0063-PLAN P2b, F5, the owner's answer "Answer the
		// repeated offer (Recommended)": the failed round brings the offer
		// again, 0063-MADR D8; n, Enter declines it.)
		waitUntil(t, func() bool { return strings.HasSuffix(h.stderr.String(), "Sign in to one now? [y/N]: ") })
		h.press("n\r")
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), sessionPrompt) })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		shownFirst(t, "the first run", run)
	})

	t.Run("gobble setup", func(t *testing.T) {
		run := &planRun{}
		replace(t, &signInWith, planSignIn(run))
		e, _ := modelEnv(t)
		q, out, _ := asked(t, fmt.Sprintf("%d\n", openai), "2\n", "1\n", "3\n") // the browser on this machine, then cancel
		if _, err := e.signIn(t.Context(), q, ""); !signInDeclined(err) {
			t.Fatalf("setup's sign-in: %v, want the sign-in declined", err)
		}
		if !strings.Contains(out.String(), provider.ConsentText["chatgpt_plan.background_use"]) {
			t.Errorf("setup's questions lack the wording:\n%s", out)
		}
		shownFirst(t, "gobble setup", run)
	})

	t.Run("gobble auth login", func(t *testing.T) {
		run := &planRun{}
		replace(t, &signInWith, planSignIn(run))
		e, _ := modelEnv(t)
		e.out.caps.In, e.stdin = true, answers(t, "1\n3\n") // the browser on this machine, then cancel
		if err := e.loginWith(t.Context(), "openai", false, nil); !signInDeclined(err) {
			t.Fatalf("auth login openai on a terminal: %v, want the sign-in declined", err)
		}
		shownFirst(t, "gobble auth login", run)
		wrote(t, "gobble auth login", e)
	})

	t.Run("gobble acp --login", func(t *testing.T) {
		run := &planRun{}
		replace(t, &signInWith, planSignIn(run))
		e, _ := modelEnv(t)
		// The menu first (0051-PLAN P7): the plan, the browser on this machine,
		// then the consent's cancel.
		e.out.caps.In, e.stdin = true, answers(t, "2\n1\n3\n")
		if err := e.acpLogin("openai", false); !signInDeclined(err) {
			t.Fatalf("acp --login openai on a terminal: %v, want the sign-in declined", err)
		}
		shownFirst(t, "gobble acp --login", run)
		wrote(t, "gobble acp --login", e)
	})

	t.Run("no terminal", func(t *testing.T) {
		for name, args := range map[string][]string{"gobble auth login": {"auth", "login", "openai"}, "gobble acp --login": {"acp", "--login", "openai"}} {
			run := &planRun{}
			replace(t, &signInWith, planSignIn(run))
			c := newCfgEnv(t)
			r := c.run(args...)
			if r.code != ExitFailure || !strings.Contains(r.stderr, errNoTerminal.Error()) || r.stdout != "" {
				t.Errorf("%s with no terminal: exit %d, stdout %q, stderr %q; want exit 1 and the no-terminal error", name, r.code, r.stdout, r.stderr)
			}
			shownFirst(t, name+" with no terminal", run)
			if _, err := os.Stat(c.authFile()); err == nil {
				t.Errorf("%s with no terminal wrote auth.json", name)
			}
		}
	})
}

// signInDeclined reports that err is a sign-in the user ended at the consent's
// question, by cancelling it, as gobble reports a sign-in that failed: its
// own words, with the SDK's error wrapped.
func signInDeclined(err error) bool {
	se, ok := errors.AsType[*provider.SignInError](err)
	return ok && strings.Contains(errors.Unwrap(se).Error(), "sign-in declined")
}

// answers is a stdin that gives text and then waits, as a terminal does, so
// the reader never sees the end of input.
func answers(t *testing.T, text string) *os.File {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = r.Close()
		_ = w.Close()
	})
	if _, err := w.WriteString(text); err != nil {
		t.Fatal(err)
	}
	return r
}
