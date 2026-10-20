package cli

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// /help lists the local commands and the agent's (0008-MADR D13), and
// makes no model call.
func TestSlashHelpUnion(t *testing.T) {
	model := llmtest.NewScript()
	h := newModelHarness(t, model, t.TempDir())
	h.start(Prompt{})
	h.press("/help\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Agent commands\n") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	out := h.stdout.String()
	for _, want := range []string{"\n  /new ", "Start a new session in this directory", "\n  /edit [text] ", "\n  /compact [focus] "} {
		if !strings.Contains(out, want) {
			t.Fatalf("help lacks %q:\n%s", want, out)
		}
	}
	if strings.Count(out, "\n  /help ") != 1 {
		t.Fatalf("/help listed twice:\n%s", out)
	}
	if len(model.Requests()) != 0 {
		t.Fatal("/help called the model")
	}
}

// An unknown command is an error printed here and never sent (D13).
func TestSlashUnknownStaysLocal(t *testing.T) {
	agent := &acptest.ScriptAgent{}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	h.press("/bogus now\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "unknown command") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.stderr.String(), "Error: unknown command /bogus; /help lists the commands") {
		t.Fatalf("stderr %q", h.stderr.String())
	}
	if p := agent.Prompts(); len(p) != 0 {
		t.Fatalf("the agent received %q", p)
	}
}

// /new closes the session; the next prompt opens a new one.
func TestSlashNew(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("one")}}, {Updates: []string{chunk("two")}}}}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	h.press("first\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "one") })
	h.press("/new\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "new session") })
	h.press("second\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "two") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	want := "initialize,session/new,session/prompt,session/close,session/new,session/prompt,session/close"
	if got := strings.Join(agent.Methods(), ","); got != want {
		t.Fatalf("methods %s, want %s", got, want)
	}
}

// /edit sends what the editor returns, starting from its argument.
func TestSlashEdit(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{}}}
	h := newLineHarness(t, agent)
	var initial string
	old := compose
	compose = func(_ context.Context, _ func(string) string, _, _, init string, _ func(*exec.Cmd) error) (string, error) {
		initial = init
		return "composed text", nil
	}
	t.Cleanup(func() { compose = old })
	h.start(Prompt{})
	h.press("/edit draft it\r")
	waitUntil(t, func() bool { return len(agent.Prompts()) == 1 })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if initial != "draft it" || !strings.Contains(agent.Prompts()[0], `"text":"composed text"`) {
		t.Fatalf("initial %q, prompts %q", initial, agent.Prompts())
	}
}

// Tab completes slash names from the registry and @paths from the
// directory.
func TestComplete(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"notes.md", "nothing.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	ls := &lineSession{base: dir}
	for _, c := range []struct {
		line, want string
		ok         bool
	}{
		{"/ne", "/new ", true},
		{"/e", "/e", true},
		{"/zz", "/zz", false},
		{"see @notes", "see @notes.md ", true},
		{"see @no", "see @not", true},
		{"@s", "@sub/", true},
		{"plain", "plain", false},
	} {
		got, pos, ok := ls.complete(c.line, len(c.line))
		if got != c.want || ok != c.ok || pos != len(c.want) {
			t.Errorf("complete(%q) = %q, %d, %v; want %q, %v", c.line, got, pos, ok, c.want, c.ok)
		}
	}
}

// Tab completes the agent's commands too, and /clone switches the line
// session to the clone, where the next prompt goes (owner's decision of
// 2026-10-06).
func TestCloneSwitches(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("A1"), llmtest.Text("A2"))
	h := newModelHarness(t, model, t.TempDir())
	h.start(Prompt{})
	h.press("first\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "A1") })
	h.press("/cl\t\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "resumed first (2 messages)") })
	h.press("second\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "A2") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.stdout.String(), "Cloned to session ") {
		t.Fatalf("stdout %q", h.stdout.String())
	}
	files, err := filepath.Glob(filepath.Join(os.Getenv("GOBBLE_HOME"), "sessions", "*", "*.jsonl"))
	if err != nil || len(files) != 2 {
		t.Fatalf("session files %v, %v", files, err)
	}
	var withSecond int
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), `"second"`) {
			withSecond++
		}
	}
	if withSecond != 1 {
		t.Fatalf("%d sessions hold the second prompt, want only the clone", withSecond)
	}
	if r := model.Requests()[1]; len(r.Messages) < 4 {
		t.Fatalf("the clone's prompt carried %d messages, want its history", len(r.Messages))
	}
}

// rawWrites records, for each write to a stream, whether the line
// harness's terminal was in raw mode at the time.
type rawWrites struct {
	h  *lineHarness
	mu sync.Mutex
	ws []rawWrite
}

type rawWrite struct {
	text string
	raw  bool
}

// rawWriter writes to w, recording each write with the terminal's state.
type rawWriter struct {
	rec *rawWrites
	w   io.Writer
}

func (r rawWriter) Write(p []byte) (int, error) {
	r.rec.h.mu.Lock()
	raw := r.rec.h.raws > r.rec.h.restores
	r.rec.h.mu.Unlock()
	r.rec.mu.Lock()
	r.rec.ws = append(r.rec.ws, rawWrite{text: string(p), raw: raw})
	r.rec.mu.Unlock()
	return r.w.Write(p)
}

// rawAt reports whether the write holding want was made in raw mode, and
// whether there was one.
func (r *rawWrites) rawAt(want string) (raw, found bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, w := range r.ws {
		if strings.Contains(w.text, want) {
			return w.raw, true
		}
	}
	return false, false
}

// The line session's own commands print with the terminal out of raw mode,
// as a turn does: in raw mode a bare line feed keeps the column, and /help
// ran off to the right (0024-MADR D1).
func TestLocalCommandsPrintOutOfRaw(t *testing.T) {
	h := newLineHarness(t, &acptest.ScriptAgent{})
	rec := &rawWrites{h: h}
	h.e.out.out = rawWriter{rec: rec, w: h.stdout}
	h.e.out.err = rawWriter{rec: rec, w: h.stderr}
	h.start(Prompt{})
	h.press("/help\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "Commands") })
	h.press("/nosuch\r")
	waitUntil(t, func() bool { return strings.Contains(h.stderr.String(), "unknown command") })
	h.press("/new\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "new session") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	// The editor echoes what is typed in raw mode, rightly; these are only
	// the commands' own output.
	for _, want := range []string{"Commands", "unknown command /nosuch", "new session"} {
		raw, found := rec.rawAt(want)
		switch {
		case !found:
			t.Errorf("nothing written holding %q", want)
		case raw:
			t.Errorf("%q was written with the terminal in raw mode", want)
		}
	}
}
