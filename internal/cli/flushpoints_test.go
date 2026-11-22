package cli

import (
	"context"
	"encoding/json/v2"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// Where the terminal's own input queue is flushed (0063-PLAN P2 step 9, A24
// as extended, A29; 0063-MADR D12, D17, D18; rows R77 to R82, R101, R103 to
// R108). The repeated offer's flush, after its text is written, is P2b's
// (its step 4): A24's case for it waits for P2b. (2026-11-21, P2b: added,
// "repeated offer".)

// flushLog records each call of the stubbed termFlush, with stderr's length
// at that moment.
type flushLog struct {
	mu sync.Mutex
	at []int
}

func (f *flushLog) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.at)
}

func (f *flushLog) calls() []int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]int(nil), f.at...)
}

// stubFlush replaces termFlush with one that records each call against h's
// stderr, empties h's terminal queue when it has one, and gives err.
func stubFlush(t *testing.T, h *lineHarness, err error) *flushLog {
	t.Helper()
	fl := &flushLog{}
	replace(t, &termFlush, func(*os.File) error {
		fl.mu.Lock()
		fl.at = append(fl.at, len(h.stderr.String()))
		fl.mu.Unlock()
		if h.tty != nil {
			h.tty.flush()
		}
		return err
	})
	return fl
}

// wantFlushes checks the count of flushes so far.
func wantFlushes(t *testing.T, fl *flushLog, n int, when string) {
	t.Helper()
	if got := fl.count(); got != n {
		t.Fatalf("%s: %d flushes; want %d", when, got, n)
	}
}

func TestFlushPoints(t *testing.T) {
	// Row R104: the first run declined at the first offer: one flush, at the
	// first prompt, after the warning; none at the offer (R81). Row R105:
	// none at the prompt after a turn; R77: a pipe's later bytes come, in
	// order.
	t.Run("first run declined", func(t *testing.T) {
		h, agent := firstRunHarness(t, "\r", acptest.Turn{}, acptest.Turn{})
		h.answer, h.flushes = "n\r", true
		fl := stubFlush(t, h, nil)
		h.start(Prompt{})
		waitUntil(t, func() bool { return fl.count() == 1 })
		if at := fl.calls()[0]; at < strings.Index(h.stderr.String(), "Warning: no model credential") {
			t.Fatalf("the flush came at stderr %d, before the first run ended:\n%s", at, h.stderr)
		}
		h.press("one\r")
		h.press("two\r")
		waitUntil(t, func() bool { return len(agent.Prompts()) == 2 })
		h.press("\x04") // read by the prompt after a turn, which flushes nothing
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		wantFlushes(t, fl, 1, "after two turns")
		if p := agent.Prompts(); !strings.Contains(p[0], `"text":"one"`) || !strings.Contains(p[1], `"text":"two"`) {
			t.Fatalf("prompts %q; want one, then two", p)
		}
	})

	// Rows R79, R80 (P2b step 4): a cancelled round, then the repeated offer:
	// no flush at the first offer or inside the round; one at the repeated
	// offer, after its text is written and before its read; one more at the
	// first prompt after the first run (R104).
	t.Run("repeated offer", flushAtRepeatedOffer)

	// Row R104, R81: the first run signed in: no flush inside the round,
	// one at the first prompt, after the /models turn (R106).
	t.Run("first run signed in", func(t *testing.T) {
		withLister(t, keyLister("sk-test-GOOD"))
		h, _ := firstRunHarness(t, "\r", acptest.Turn{}, acptest.Turn{})
		h.flushes = true
		fl := stubFlush(t, h, nil)
		h.start(Prompt{})
		waitUntil(t, has(h, "Number [1]: "))
		h.press(strconv.Itoa(providerNumber("claude")) + "\r")
		waitUntil(t, has(h, "Enter your Claude (Anthropic) API key: "))
		wantFlushes(t, fl, 0, "inside the round")
		h.press("sk-test-GOOD\r")
		waitUntil(t, func() bool { return fl.count() == 1 })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		wantFlushes(t, fl, 1, "at the end")
	})

	// Rows R103, R107: with a credential, no first run: no flush at the
	// first prompt; none after /login nosuch, a turn or /logout; one after
	// each /login that asked: cancelled, signed in, failed.
	t.Run("login", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-only")
		var o provider.SignInOptions
		replace(t, &signInWith, typedKeySignIn(&o))
		agent := &acptest.ScriptAgent{Turns: make([]acptest.Turn, 4)}
		h := newLineHarness(t, agent)
		h.flushes = true
		fl := stubFlush(t, h, nil)
		h.start(Prompt{})
		h.press("/login nosuch\r")
		waitUntil(t, has(h, "nosuch"))
		h.press("hi\r")
		waitUntil(t, func() bool { return len(agent.Prompts()) == 1 })
		h.press("/login gemini\r")
		waitUntil(t, has(h, "Enter your Gemini (Google) API key: "))
		wantFlushes(t, fl, 0, "at a first prompt with a credential, after /login nosuch and after a turn")
		h.press("\r")
		waitUntil(t, func() bool { return fl.count() == 1 })
		if at := fl.calls()[0]; at < strings.Index(h.stderr.String(), "Sign-in cancelled; nothing was saved.") {
			t.Fatalf("the flush came before the cancelled sign-in's line")
		}

		h.press("/login gemini\r")
		waitUntil(t, func() bool { return strings.Count(h.stderr.String(), "Enter your Gemini (Google) API key: ") == 2 })
		h.press("sk-test-typed\r")
		waitUntil(t, func() bool { return len(agent.Prompts()) == 2 }) // the /models turn
		waitUntil(t, func() bool { return fl.count() == 2 })

		replace(t, &signInWith, noSignIn)
		h.press("/login gemini\r")
		waitUntil(t, has(h, errNoSignIn.Error()))
		waitUntil(t, func() bool { return fl.count() == 3 })

		h.press("/logout gemini\r")
		waitUntil(t, func() bool {
			return strings.Contains(h.stdout.String(), "gemini") && strings.Contains(h.stdout.String(), "Removed")
		})
		h.press("bye\r") // read by the prompt after /logout
		waitUntil(t, func() bool { return len(agent.Prompts()) == 3 })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		wantFlushes(t, fl, 3, "after /logout and at the end")
	})

	// Row R106: keys typed while the /models turn after /login runs are
	// flushed at the prompt that follows, not given to it as text.
	t.Run("typed during the models turn", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-only")
		withLister(t, keyLister("sk-test-GOOD"))
		var o provider.SignInOptions
		replace(t, &signInWith, typedKeySignIn(&o))
		agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{WaitCancel: true}, {}}}
		h := newLineHarness(t, agent)
		h.flushes, h.tty = true, newTTYQueue()
		fl := stubFlush(t, h, nil)
		h.start(Prompt{})
		h.press("/login claude\r")
		waitUntil(t, has(h, "Enter your Claude (Anthropic) API key: "))
		h.press("sk-test-GOOD\r")
		waitUntil(t, func() bool { return len(agent.Prompts()) == 1 })
		h.press("typed-during\r")
		if !h.e.intr.take() {
			t.Fatal("the /models turn did not take the interrupt")
		}
		waitUntil(t, func() bool { return fl.count() == 1 })
		h.press("after\r")
		waitUntil(t, func() bool { return len(agent.Prompts()) == 2 })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		for _, p := range agent.Prompts() {
			if strings.Contains(p, "typed-during") {
				t.Fatalf("keys typed during the /models turn became a turn: %q", agent.Prompts())
			}
		}
		if p := agent.Prompts(); !strings.Contains(p[1], `"text":"after"`) {
			t.Fatalf("prompts %q; want /models claude, then after", p)
		}
	})

	// Row R108, as amended by MADR D21: nothing is flushed on the way out. A
	// session ended by Ctrl+D at the first prompt after a sign-in counts that
	// prompt's one flush and none after.
	t.Run("Ctrl+D at the first prompt after a sign-in", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-only")
		var o provider.SignInOptions
		replace(t, &signInWith, typedKeySignIn(&o))
		h := newLineHarness(t, &acptest.ScriptAgent{})
		h.flushes = true
		fl := stubFlush(t, h, nil)
		h.start(Prompt{})
		h.press("/login gemini\r")
		waitUntil(t, has(h, "Enter your Gemini (Google) API key: "))
		h.press("\r")
		waitUntil(t, func() bool { return fl.count() == 1 })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		wantFlushes(t, fl, 1, "a session ended by Ctrl+D at the first prompt after a sign-in")
	})

	// Row R108: the run's context ends during the /models turn after
	// /login, with the mark set: no flush.
	t.Run("run ended during the models turn", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-only")
		withLister(t, keyLister("sk-test-GOOD"))
		var o provider.SignInOptions
		replace(t, &signInWith, typedKeySignIn(&o))
		agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{WaitCancel: true}}}
		h := newLineHarness(t, agent)
		ctx, cancel := context.WithCancelCause(t.Context())
		h.e.ctx = ctx
		h.flushes = true
		fl := stubFlush(t, h, nil)
		h.start(Prompt{})
		h.press("/login claude\r")
		waitUntil(t, has(h, "Enter your Claude (Anthropic) API key: "))
		h.press("sk-test-GOOD\r")
		waitUntil(t, func() bool { return len(agent.Prompts()) == 1 })
		cause := errors.New("the run ended")
		cancel(cause)
		if err := h.wait(); !errors.Is(err, cause) {
			t.Fatalf("the session returned %v; want the run's cause", err)
		}
		wantFlushes(t, fl, 0, "a run ended with the mark set")
	})

	// Row R123 (MADR D21): keys the editor holds across a sign-in. /login
	// gemini and 1, Enter in one press; the key prompt answered with Enter.
	// The prompt that submits the held 1 flushes before its turn, and keys
	// typed while that turn runs reach the next prompt (R105).
	t.Run("keys held across a sign-in", func(t *testing.T) {
		t.Setenv("OPENAI_API_KEY", "test-only")
		var o provider.SignInOptions
		replace(t, &signInWith, typedKeySignIn(&o))
		agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{WaitCancel: true}, {}}}
		h := newLineHarness(t, agent)
		h.flushes, h.tty = true, newTTYQueue()
		fl := stubFlush(t, h, nil)
		h.start(Prompt{})
		h.press("/login gemini\r1\r")
		waitUntil(t, has(h, "Enter your Gemini (Google) API key: "))
		h.press("\r")
		waitUntil(t, func() bool { return len(agent.Prompts()) == 1 }) // the held 1's turn
		wantFlushes(t, fl, 1, "when the held keys' turn runs")
		h.press("typed-during\r")
		if !h.e.intr.take() {
			t.Fatal("the held keys' turn did not take the interrupt")
		}
		waitUntil(t, func() bool { return len(agent.Prompts()) == 2 })
		h.press("\x04")
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		wantFlushes(t, fl, 1, "at the end")
		if p := agent.Prompts(); !strings.Contains(p[0], `"text":"1"`) || !strings.Contains(p[1], `"text":"typed-during"`) {
			t.Fatalf("prompts %q; want the held 1, then typed-during", p)
		}
	})

	// Row R107: a permission prompt sets no mark: after the first run's
	// flush, none at the prompt after a turn that asked one.
	t.Run("permission prompt", func(t *testing.T) {
		h := permissionHarness(t)
		h.flushes = true
		fl := stubFlush(t, h, nil)
		h.start(Prompt{})
		waitUntil(t, func() bool { return fl.count() == 1 })
		h.press("write it\r")
		waitUntil(t, has(h, "[1] once"))
		h.press("1")
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "done") })
		h.press("\x04") // read by the prompt after the turn
		if err := h.wait(); err != nil {
			t.Fatal(err)
		}
		wantFlushes(t, fl, 1, "after a permission prompt")
	})

	// Row R81: gobble setup, gobble acp --login and the trust question ask
	// on a terminal and never flush.
	t.Run("setup, acp --login, trust", func(t *testing.T) {
		calls := 0
		replace(t, &termFlush, func(*os.File) error { calls++; return nil })
		var o provider.SignInOptions
		replace(t, &signInWith, typedKeySignIn(&o))
		for _, c := range []struct {
			name string
			run  func(e *runEnv) error
			keys [][2]string // what is shown, then what is typed
		}{
			{"setup", func(e *runEnv) error { return (&SetupCmd{}).Run(e) },
				[][2]string{{"Number [1]: ", strconv.Itoa(providerNumber("gemini")) + "\r"}, {"API key: ", "sk-test-typed\r"}}},
			{"acp --login", func(e *runEnv) error { return e.acpLogin("openai", false) },
				[][2]string{{"API key: ", "sk-test-typed\r"}}},
		} {
			e, w, stderr := pipeEnv(t)
			done := runAsync(func() error { return c.run(e) })
			for _, k := range c.keys {
				waitUntil(t, func() bool { return strings.Contains(stderr.String(), k[0]) })
				typeInto(t, w, k[1])
			}
			_ = w.Close()
			_ = waitDone(t, done, stderr)
			if calls != 0 {
				t.Fatalf("%s flushed %d times", c.name, calls)
			}
		}
		fakeRaw(t)
		t.Setenv("GOBBLE_HOME", t.TempDir())
		proj := trustProject(t)
		e, w, stderr := pipeEnv(t)
		typeInto(t, w, "1")
		e.decideTrustHere(proj)
		if _, d := e.projectTrust(proj); !d.Trusted {
			t.Fatalf("the trust question was not answered:\n%s", stderr)
		}
		if calls != 0 {
			t.Fatalf("the trust question flushed %d times", calls)
		}
	})
}

// errNotATerminal is the stubbed flush's failure.
var errNotATerminal = errors.New("not a terminal after all")

// A29 (rows R78, R101): a flush that fails is skipped; the first failure in
// the process is warned of on stderr, once, in R101's approved words, and
// both prompts still read.
func TestFlushFailureWarnsOnce(t *testing.T) {
	var o provider.SignInOptions
	replace(t, &signInWith, typedKeySignIn(&o))
	h, agent := firstRunHarness(t, "\r", acptest.Turn{})
	h.answer, h.flushes = "n\r", true
	fl := stubFlush(t, h, errNotATerminal)
	h.start(Prompt{})
	waitUntil(t, func() bool { return fl.count() == 1 })
	h.press("/login gemini\r")
	waitUntil(t, has(h, "Enter your Gemini (Google) API key: "))
	h.press("\r")
	waitUntil(t, func() bool { return fl.count() == 2 })
	h.press("hello\r")
	waitUntil(t, func() bool { return len(agent.Prompts()) == 1 })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	words := "Warning: could not clear what was typed ahead on this terminal (" + errNotATerminal.Error() + "); it may answer the next prompt."
	shown := strings.Join(strings.Fields(h.stderr.String()), " ")
	if n := strings.Count(shown, words); n != 1 {
		t.Fatalf("the warning %q shown %d times; want once:\n%s", words, n, h.stderr)
	}
	if n := strings.Count(shown, "could not clear"); n != 1 {
		t.Fatalf("a flush failure warned %d times; want once", n)
	}
	if p := agent.Prompts(); !strings.Contains(p[0], `"text":"hello"`) {
		t.Fatalf("prompts %q; want hello, read at the second prompt", p)
	}
}

// permissionHarness is a line session over gobble's agent, whose model
// asks to write a file, which asks the permission prompt, then says done.
func permissionHarness(t *testing.T) *lineHarness {
	t.Helper()
	args, err := json.Marshal(map[string]string{"path": "a.txt", "content": "x"})
	if err != nil {
		t.Fatal(err)
	}
	model := llmtest.NewScript(llmtest.ToolCall("w1", "write", string(args)), llmtest.Text("done"))
	return newModelHarness(t, model, t.TempDir())
}

// flushAtRepeatedOffer is TestFlushPoints's case "repeated offer" (rows R79,
// R80; P2b step 4).
func flushAtRepeatedOffer(t *testing.T) {
	h, _ := firstRunHarness(t, "\r")
	h.flushes = true
	fl := stubFlush(t, h, nil)
	h.start(Prompt{})
	toKeyPrompt(t, h)
	wantFlushes(t, fl, 0, "at the first offer and inside the round")
	h.press("\x03")
	expect(t, h, "the offer again, [y/N]", offeredAgain(h, 2))
	waitUntil(t, func() bool { return fl.count() == 1 })
	repeated := firstRunQuestion + " [y/N]: "
	if at, end := fl.calls()[0], strings.LastIndex(h.stderr.String(), repeated)+len(repeated); at < end {
		t.Fatalf("the repeated offer's flush came at stderr %d, before its text ended at %d:\n%s", at, end, h.stderr)
	}
	h.press("n\r")
	waitUntil(t, func() bool { return fl.count() == 2 })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	wantFlushes(t, fl, 2, "at the end")
}
