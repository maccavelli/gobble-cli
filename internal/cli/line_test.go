package cli

import (
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// lineHarness runs a line session over a pipe, with raw mode stubbed and
// stdout and stderr captured.
type lineHarness struct {
	t              *testing.T
	agent          *acptest.ScriptAgent
	e              *runEnv
	keys           *io.PipeWriter
	cwd            string
	stdout, stderr *syncBuffer
	done           chan error

	mu             sync.Mutex
	raws, restores int
}

func newLineHarness(t *testing.T, agent *acptest.ScriptAgent) *lineHarness {
	t.Helper()
	t.Setenv("GOBBLE_HOME", t.TempDir())
	withAgent(t, agent)
	h := &lineHarness{t: t, agent: agent, stdout: &syncBuffer{}, stderr: &syncBuffer{}, done: make(chan error, 1)}
	caps := term.Caps{In: true, Out: term.Stream{TTY: true, Width: 80}, Err: term.Stream{Width: 80}}
	out := &Output{out: h.stdout, err: h.stderr, caps: caps}
	h.e = &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, intr: &interrupts{}, now: time.Now}
	t.Cleanup(func() {
		if err := h.e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	return h
}

// newModelHarness is a line harness over gobble's own agent, on the
// scripted model, in cwd.
func newModelHarness(t *testing.T, s *llmtest.Script, cwd string) *lineHarness {
	t.Helper()
	t.Setenv("GOBBLE_HOME", t.TempDir())
	withModel(t, s)
	h := &lineHarness{t: t, cwd: cwd, stdout: &syncBuffer{}, stderr: &syncBuffer{}, done: make(chan error, 1)}
	caps := term.Caps{In: true, Out: term.Stream{TTY: true, Width: 80}, Err: term.Stream{Width: 80}}
	out := &Output{out: h.stdout, err: h.stderr, caps: caps}
	h.e = &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, intr: &interrupts{}, now: time.Now}
	t.Cleanup(func() {
		if err := h.e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	return h
}

// start runs the session with first as the composed first prompt.
func (h *lineHarness) start(first Prompt) {
	keysR, keysW := io.Pipe()
	h.keys = keysW
	cwd := h.cwd
	if cwd == "" {
		cwd = "/work"
	}
	ls := &lineSession{e: h.e, cwd: cwd, base: cwd, in: keysR, raw: h.raw}
	go func() { h.done <- ls.run(first) }()
	h.t.Cleanup(func() { keysW.Close() }) //nolint:errcheck,gosec // test cleanup
}

func (h *lineHarness) raw() (func() error, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.raws++
	return func() error {
		h.mu.Lock()
		defer h.mu.Unlock()
		h.restores++
		return nil
	}, nil
}

func (h *lineHarness) press(s string) {
	h.t.Helper()
	if _, err := io.WriteString(h.keys, s); err != nil {
		h.t.Fatal(err)
	}
}

// wait returns the session's error once it ends.
func (h *lineHarness) wait() error {
	h.t.Helper()
	select {
	case err := <-h.done:
		return err
	case <-time.After(10 * time.Second):
		h.t.Fatalf("the line session did not end; stdout %q, stderr %q", h.stdout.String(), h.stderr.String())
		return nil
	}
}

// A prompt is one turn; Ctrl+D on an empty line ends the session with
// session/close, and every raw mode entered is restored.
func TestLineSessionTurn(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("reply-1\n")}}}}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	h.press("hello\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "reply-1") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatalf("session ended with %v", err)
	}
	if got := strings.Join(agent.Methods(), ","); got != "initialize,session/new,session/prompt,session/close" {
		t.Fatalf("methods %s", got)
	}
	if p := agent.Prompts(); len(p) != 1 || !strings.Contains(p[0], `"text":"hello"`) {
		t.Fatalf("prompts %q", p)
	}
	h.mu.Lock()
	raws, restores := h.raws, h.restores
	h.mu.Unlock()
	if raws != 2 || restores != 2 {
		t.Fatalf("raw mode entered %d times and restored %d; want 2 and 2 (start, after the turn)", raws, restores)
	}
	if !strings.Contains(h.stderr.String(), "0s") {
		t.Fatalf("no status line on stderr: %q", h.stderr.String())
	}
}

// Nothing but initialize reaches the agent before the first prompt (D5).
func TestLineSessionNoSessionBeforePrompt(t *testing.T) {
	agent := &acptest.ScriptAgent{}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "> ") })
	if got := strings.Join(agent.Methods(), ","); got != "initialize" {
		t.Fatalf("before any prompt the agent received %s; want only initialize", got)
	}
	if !strings.Contains(h.stderr.String(), "Warning: no model credential: set one of ") {
		t.Fatalf("stderr %q lacks the credential warning (D1)", h.stderr.String())
	}
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(agent.Methods(), ","); got != "initialize" {
		t.Fatalf("a session with no prompt sent %s", got)
	}
}

// The composed input is the first turn.
func TestLineSessionFirstPrompt(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{}}}
	h := newLineHarness(t, agent)
	h.start(Prompt{Text: "fix the test"})
	waitUntil(t, func() bool { return len(agent.Prompts()) == 1 })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if p := agent.Prompts(); !strings.Contains(p[0], `"text":"fix the test"`) {
		t.Fatalf("prompts %q", p)
	}
}

// Ctrl+C during a turn sends session/cancel and returns to the prompt
// (D5); the session then goes on.
func TestLineSessionCancel(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{WaitCancel: true}, {Updates: []string{chunk("after")}}}}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	h.press("wait\r")
	waitUntil(t, func() bool { return len(agent.Prompts()) == 1 })
	if !h.e.intr.take() {
		t.Fatal("the running turn did not take the interrupt")
	}
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "cancelled") })
	h.press("again\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "after") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if n := agent.Cancels(); n != 1 {
		t.Fatalf("%d session/cancel sent, want 1", n)
	}
	if h.e.intr.take() {
		t.Fatal("an interrupt was taken with no turn running")
	}
}

// A turn refused for want of a credential names the variables, and the
// session goes on.
func TestLineSessionNoCredential(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{ErrCode: -32000, ErrMessage: "Authentication required"}}}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	h.press("hi\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "Error: no model credential: set one of ") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h.stderr.String(), "-32000") {
		t.Fatalf("the raw JSON-RPC error reached the user: %q", h.stderr.String())
	}
}

// A turn's error is printed and the session goes on.
func TestLineSessionTurnError(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{ErrCode: -32603, ErrMessage: "provider down"}, {Updates: []string{chunk("fine")}}}}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	h.press("one\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "provider down") })
	h.press("two\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "fine") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.stderr.String(), "Error: session/prompt:") {
		t.Fatalf("stderr %q", h.stderr.String())
	}
}
