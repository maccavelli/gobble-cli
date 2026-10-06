package cli

import (
	"context"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// withAgent makes Main's agent the scripted one for the rest of the test.
func withAgent(t *testing.T, agent *acptest.ScriptAgent) {
	t.Helper()
	old := agentServe
	agentServe = func(*runEnv) acpclient.ServeFunc { return agent.Serve }
	t.Cleanup(func() { agentServe = old })
}

// fixedClock makes Main's clock start at a fixed instant and advance 100 ms
// per reading.
func fixedClock(t *testing.T) {
	t.Helper()
	old := clock
	var mu sync.Mutex
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	clock = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		now = now.Add(100 * time.Millisecond)
		return now
	}
	t.Cleanup(func() { clock = old })
}

const (
	textChunk = `{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":%q}}`
	usageCost = `{"sessionUpdate":"usage_update","used":1500,"size":200000,"cost":{"amount":0.0123,"currency":"USD"}}`
)

func chunk(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return strings.Replace(textChunk, "%q", string(b), 1)
}

// The prompt the print mode sends is the one a stdio SDK client sends for
// the same text (0002-PLAN Phase 2 step 3; acpclient's
// TestPromptParamsMatchSDKClient pins the SDK side), and the methods come in
// the plan's order.
func TestPrintSendsACP(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("ok")}}}}
	withAgent(t, agent)
	r := runMain(t, nil, "-p", "hi")
	if r.code != ExitOK || r.stdout != "ok\n" || r.stderr != "" {
		t.Fatalf("exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
	if got := strings.Join(agent.Methods(), ","); got != "initialize,session/new,session/prompt,session/close" {
		t.Fatalf("methods %s", got)
	}
	const want = `{"prompt":[{"text":"hi","type":"text"}],"sessionId":"s1"}`
	if p := agent.Prompts(); len(p) != 1 || p[0] != want {
		t.Fatalf("prompts %q, want %s", p, want)
	}
}

// Text is the turn's agent text, sanitised, with one trailing newline.
func TestPrintText(t *testing.T) {
	withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{
		chunk("Hel"), `{"sessionUpdate":"agent_thought_chunk","content":{"type":"text","text":"hidden"}}`,
		chunk("lo \x1b[2Jworld\n\n"), usageCost,
	}}}})
	r := runMain(t, nil, "-p", "x")
	if r.code != ExitOK || r.stdout != "Hello world\n" {
		t.Fatalf("exit %d, stdout %q, stderr %q", r.code, r.stdout, r.stderr)
	}
}

func scriptedTurn() acptest.Turn {
	return acptest.Turn{
		Updates: []string{chunk("Hel"), chunk("lo"), usageCost},
		Usage:   `{"inputTokens":10,"outputTokens":2,"totalTokens":12}`,
	}
}

func TestPrintJSON(t *testing.T) {
	withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{scriptedTurn()}})
	r := runMain(t, nil, "-p", "--output-format", "json", "x")
	if r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	golden(t, "print-json", r.stdout)
}

func TestPrintStreamJSON(t *testing.T) {
	fixedClock(t)
	agent := &acptest.ScriptAgent{
		Turns:          []acptest.Turn{scriptedTurn()},
		SessionUpdates: []string{`{"sessionUpdate":"available_commands_update","availableCommands":[]}`},
	}
	withAgent(t, agent)
	dir := t.TempDir()
	r := runMain(t, nil, "-p", "--output-format", "stream-json", "--cwd", dir, "x")
	if r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	enc, err := json.Marshal(dir)
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "print-stream-json", strings.ReplaceAll(r.stdout, string(enc), `"<cwd>"`))
}

// A failed session/prompt exits 1. The JSON formats still write the result,
// with stopReason "error" and the message (owner's decision of 2026-10-05);
// text writes nothing to stdout.
func TestPrintErrorResult(t *testing.T) {
	for _, format := range []string{"text", "json", "stream-json"} {
		t.Run(format, func(t *testing.T) {
			withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("part")}, ErrCode: -32000, ErrMessage: "provider down"}}})
			r := runMain(t, nil, "-p", "--output-format", format, "x")
			if r.code != ExitFailure || !strings.Contains(r.stderr, "Error: session/prompt:") || !strings.Contains(r.stderr, "provider down") {
				t.Fatalf("exit %d, stderr %q; want 1 and the error", r.code, r.stderr)
			}
			if format == "text" {
				if r.stdout != "" {
					t.Fatalf("text wrote %q on a failed turn", r.stdout)
				}
				return
			}
			lines := strings.Split(strings.TrimSuffix(r.stdout, "\n"), "\n")
			var res printResult
			if err := json.Unmarshal([]byte(lines[len(lines)-1]), &res); err != nil {
				t.Fatalf("last line %q: %v", lines[len(lines)-1], err)
			}
			if res.StopReason != "error" || !strings.Contains(res.Error, "provider down") || res.Text != "part" || res.SessionID != "s1" {
				t.Fatalf("result = %+v", res)
			}
		})
	}
}

// Stop reasons: cancelled that gobble did not ask for is a provider-side
// abort, exit 1; the others are a finished turn, exit 0 (D10).
func TestPrintStopReasons(t *testing.T) {
	for _, tc := range []struct {
		reason string
		code   int
	}{{"end_turn", 0}, {"max_tokens", 0}, {"max_turn_requests", 0}, {"refusal", 0}, {"cancelled", 1}} {
		t.Run(tc.reason, func(t *testing.T) {
			withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{{StopReason: tc.reason}}})
			r := runMain(t, nil, "-p", "--output-format", "json", "x")
			if r.code != tc.code || !strings.Contains(r.stdout, `"stopReason":"`+tc.reason+`"`) {
				t.Fatalf("exit %d, stdout %q, stderr %q; want %d", r.code, r.stdout, r.stderr, tc.code)
			}
			if tc.code == 1 && !strings.Contains(r.stderr, "Error: the agent cancelled the turn") {
				t.Fatalf("stderr %q", r.stderr)
			}
		})
	}
}

// An image the agent does not accept stops the run before the prompt is
// sent (D19 item 1).
func TestPrintImageRefused(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{}}}
	withAgent(t, agent)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "shot.png"), png, 0o600); err != nil {
		t.Fatal(err)
	}
	r := runMain(t, nil, "--cwd", dir, "-p", "@shot.png", "what is this")
	want := "Error: the agent does not accept images (" + filepath.Join(dir, "shot.png") + ")"
	if r.code != ExitFailure || !strings.Contains(r.stderr, want) {
		t.Fatalf("exit %d, stderr %q; want 1 and %q", r.code, r.stderr, want)
	}
	if n := len(agent.Prompts()); n != 0 {
		t.Fatalf("%d prompts sent", n)
	}
}

// --cwd is the session's cwd and the base of @path and --file.
func TestPrintCwd(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{}, {}}}
	withAgent(t, agent)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("note"), 0o600); err != nil {
		t.Fatal(err)
	}
	if r := runMain(t, nil, "--cwd", dir, "-p", "@notes.md"); r.code != ExitOK {
		t.Fatalf("@path: exit %d, stderr %q", r.code, r.stderr)
	}
	if r := runMain(t, nil, "--cwd", dir, "-p", "--file", "notes.md"); r.code != ExitOK {
		t.Fatalf("--file: exit %d, stderr %q", r.code, r.stderr)
	}
	if got := agent.Cwds(); len(got) != 2 || got[0] != dir || got[1] != dir {
		t.Fatalf("cwds %q, want %q twice", got, dir)
	}
	// The @path's absolute name is escaped twice over (%q, then JSON), so
	// check its directory's distinctive part and the wrapped body.
	p := agent.Prompts()
	if !strings.Contains(p[0], filepath.Base(filepath.Dir(dir))) || !strings.Contains(p[0], `notes.md`) ||
		!strings.Contains(p[0], `\nnote\n</file>`) || !strings.Contains(p[1], `"text":"note"`) {
		t.Fatalf("prompts %q", p)
	}
}

// --stats prints the status line on stderr in print mode, and only then.
func TestPrintStats(t *testing.T) {
	withAgent(t, &acptest.ScriptAgent{Turns: []acptest.Turn{scriptedTurn(), scriptedTurn()}})
	r := runMain(t, nil, "-p", "--stats", "x")
	if r.code != ExitOK || !strings.Contains(r.stderr, "ttft") || !strings.Contains(r.stderr, "$0.0123") {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	if r := runMain(t, nil, "-p", "x"); r.stderr != "" {
		t.Fatalf("without --stats, stderr %q", r.stderr)
	}
}

// A cancelled context during a turn sends session/cancel to the agent.
func TestPrintCancelSendsSessionCancel(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{WaitCancel: true}}}
	withAgent(t, agent)
	t.Setenv("GOBBLE_HOME", t.TempDir())
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		waitUntil(t, func() bool { return len(agent.Prompts()) == 1 })
		cancel()
	}()
	var out, errw syncBuffer
	if code := Main(ctx, []string{"-p", "x"}, nil, &out, &errw); code == ExitOK {
		t.Fatalf("a cancelled turn exited 0; stderr %q", errw.String())
	}
	waitUntil(t, func() bool { return agent.Cancels() == 1 })
}

// syncBuffer is a strings.Builder safe for the connection's goroutine and
// the test.
type syncBuffer struct {
	mu sync.Mutex
	b  strings.Builder
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

func waitUntil(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Error("condition not met within 5 s")
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}
