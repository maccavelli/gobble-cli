package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// withModel makes the real agent's model the scripted one for the test.
func withModel(t *testing.T, s *llmtest.Script) {
	t.Helper()
	old := modelProvider
	modelProvider = func(*runEnv) func() (llm.Provider, error) {
		return func() (llm.Provider, error) { return s, nil }
	}
	t.Cleanup(func() { modelProvider = old })
}

// runIn runs Main with GOBBLE_HOME set to home, so runs share a store.
func runIn(t *testing.T, home string, args ...string) result {
	t.Helper()
	t.Setenv("GOBBLE_HOME", home)
	var out, errw bytes.Buffer
	code := Main(t.Context(), args, nil, &out, &errw)
	return result{code: code, stdout: out.String(), stderr: errw.String()}
}

func mustRun(t *testing.T, home string, args ...string) result {
	t.Helper()
	r := runIn(t, home, args...)
	if r.code != ExitOK {
		t.Fatalf("%v: exit %d, stderr %q", args, r.code, r.stderr)
	}
	// Session files are ordered by their messages' millisecond stamps.
	time.Sleep(5 * time.Millisecond)
	return r
}

func storeFiles(t *testing.T, dir string) []string {
	t.Helper()
	m, err := filepath.Glob(filepath.Join(dir, "*", "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	flat, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	return append(m, flat...)
}

// texts are a request's user and assistant texts, in order.
func texts(r *llm.Request) []string {
	var out []string
	for _, m := range r.Messages {
		if m.Role == llm.RoleUser || m.Role == llm.RoleAssistant {
			for _, c := range m.Content {
				if c.Type == llm.ContentText {
					out = append(out, c.Text)
				}
			}
		}
	}
	return out
}

// -c resumes the newest session of the cwd, and its history reaches the
// model.
func TestContinueResumesTheNewest(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	s := llmtest.NewScript(llmtest.Text("A1"), llmtest.Text("B1"), llmtest.Text("B2"))
	withModel(t, s)
	mustRun(t, home, "--cwd", cwd, "-p", "first")
	mustRun(t, home, "--cwd", cwd, "-p", "second")
	mustRun(t, home, "--cwd", cwd, "-c", "-p", "third")
	if got := strings.Join(texts(s.Requests()[2]), ","); got != "second,B1,third" {
		t.Fatalf("the continued request carries %q, want the newest session's history", got)
	}
	if n := len(storeFiles(t, filepath.Join(home, "sessions"))); n != 2 {
		t.Fatalf("%d session files, want 2", n)
	}
}

// -c with no session starts a new one, as Pi's continueRecent does.
func TestContinueWithNoneStartsNew(t *testing.T) {
	home := t.TempDir()
	withModel(t, llmtest.NewScript(llmtest.Text("a")))
	mustRun(t, home, "--cwd", t.TempDir(), "-c", "-p", "x")
	if n := len(storeFiles(t, filepath.Join(home, "sessions"))); n != 1 {
		t.Fatalf("%d session files", n)
	}
}

// --session takes an id, a unique prefix of an id or title, or a path in
// the store.
func TestSessionValues(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	s := llmtest.NewScript(llmtest.Text("1"), llmtest.Text("2"), llmtest.Text("3"), llmtest.Text("4"), llmtest.Text("5"), llmtest.Text("6"))
	withModel(t, s)
	mustRun(t, home, "--cwd", cwd, "--session-id", "alpha-1", "-p", "apples")
	mustRun(t, home, "--cwd", cwd, "--session-id", "alpha-2", "-p", "avocados")
	mustRun(t, home, "--cwd", cwd, "--session-id", "beta-1", "-p", "bananas")
	mustRun(t, home, "--cwd", cwd, "--session", "beta", "-p", "more")
	if got := texts(s.Requests()[3]); got[0] != "bananas" {
		t.Fatalf("prefix beta resumed %q", got)
	}
	mustRun(t, home, "--cwd", cwd, "--session", "AVOC", "-p", "by title")
	if got := texts(s.Requests()[4]); got[0] != "avocados" {
		t.Fatalf("title prefix resumed %q", got)
	}
	file := storeFiles(t, filepath.Join(home, "sessions"))
	var alpha1 string
	for _, f := range file {
		if strings.HasSuffix(f, "_alpha-1.jsonl") {
			alpha1 = f
		}
	}
	mustRun(t, home, "--cwd", cwd, "--session", alpha1, "-p", "by path")
	if got := texts(s.Requests()[5]); got[0] != "apples" {
		t.Fatalf("path resumed %q", got)
	}

	outside := filepath.Join(t.TempDir(), "copy.jsonl")
	b, err := os.ReadFile(alpha1)
	if err != nil {
		t.Fatal(err)
	}
	stranger := strings.Replace(string(b), `"id":"alpha-1"`, `"id":"stranger"`, 1)
	if err := os.WriteFile(outside, []byte(stranger), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ value, want string }{
		{"alpha", "--session alpha matches 2 sessions: "},
		{"zzz", "--session zzz: no session matches"},
		{outside, "session stranger is not in the session store"},
	} {
		r := runIn(t, home, "--cwd", cwd, "--session", tc.value, "-p", "x")
		if r.code != ExitUsage || !strings.Contains(r.stderr, tc.want) {
			t.Errorf("--session %s: exit %d, stderr %q; want 2 and %q", tc.value, r.code, r.stderr, tc.want)
		}
	}
}

// --session-id and -n go through session/new's _meta; a taken id is a
// usage error; -n with -c renames the resumed session (0002-PLAN Phase 7).
func TestSessionIDAndName(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	withModel(t, llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b")))
	mustRun(t, home, "--cwd", cwd, "--session-id", "my.id", "-n", "My name", "-p", "x")
	files := storeFiles(t, filepath.Join(home, "sessions"))
	if len(files) != 1 || !strings.HasSuffix(files[0], "_my.id.jsonl") {
		t.Fatalf("files %v", files)
	}
	b, err := os.ReadFile(files[0])
	if err != nil || !strings.Contains(string(b), `"type":"session_info"`) || !strings.Contains(string(b), `"name":"My name"`) {
		t.Fatalf("file lacks the name: %s %v", b, err)
	}
	r := runIn(t, home, "--cwd", cwd, "--session-id", "my.id", "-p", "y")
	if r.code != ExitUsage || !strings.Contains(r.stderr, "a session with this id exists") {
		t.Fatalf("taken id: exit %d, stderr %q", r.code, r.stderr)
	}
	mustRun(t, home, "--cwd", cwd, "-c", "-n", "Renamed", "-p", "y")
	if after := storeFiles(t, filepath.Join(home, "sessions")); len(after) != 1 || after[0] != files[0] {
		t.Fatalf("-n with -c: files %v, want only %s", after, files[0])
	}
	b, err = os.ReadFile(files[0])
	s := string(b)
	if err != nil || strings.Index(s, `"name":"Renamed"`) < strings.Index(s, `"name":"My name"`) || !strings.Contains(s, `"content":"y"`) {
		t.Fatalf("-n with -c did not rename the resumed session: %s %v", b, err)
	}
}

// --fork starts a new session with the source's history, recording the
// source as its parent.
func TestFork(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	s := llmtest.NewScript(llmtest.Text("one"), llmtest.Text("two"))
	withModel(t, s)
	mustRun(t, home, "--cwd", cwd, "--session-id", "src", "-p", "original")
	mustRun(t, home, "--cwd", cwd, "--session-id", "dst", "--fork", "src", "-p", "forked")
	if got := strings.Join(texts(s.Requests()[1]), ","); got != "original,one,forked" {
		t.Fatalf("fork request %q", got)
	}
	st := jsonl.New(filepath.Join(home, "sessions"), jsonl.Options{})
	h, _, _, err := st.Read(t.Context(), "dst")
	if err != nil || !strings.HasSuffix(h.ParentSession, "_src.jsonl") {
		t.Fatalf("fork header %+v, %v", h, err)
	}
	src, entries, _, err := st.Read(t.Context(), "src")
	if err != nil || src.ID != "src" || len(entries) != 2 {
		t.Fatalf("the source changed: %d entries, %v", len(entries), err)
	}
}

// --no-session keeps the session in memory; --session-dir keeps it flat in
// the directory given.
func TestNoSessionAndSessionDir(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	withModel(t, llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b")))
	mustRun(t, home, "--cwd", cwd, "--no-session", "-p", "x")
	if got := storeFiles(t, filepath.Join(home, "sessions")); len(got) != 0 {
		t.Fatalf("--no-session wrote %v", got)
	}
	dir := filepath.Join(t.TempDir(), "mine")
	mustRun(t, home, "--cwd", cwd, "--session-dir", dir, "-p", "y")
	flat, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if err != nil || len(flat) != 1 {
		t.Fatalf("--session-dir files %v, %v", flat, err)
	}
	if got := storeFiles(t, filepath.Join(home, "sessions")); len(got) != 0 {
		t.Fatalf("the default store got %v", got)
	}
}

// A session another process writes exits 1 with the holder's pid
// (0008-MADR D19 item 8).
func TestLockedSessionExits1(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	withModel(t, llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b")))
	mustRun(t, home, "--cwd", cwd, "--session-id", "held", "-p", "x")
	held, err := jsonl.New(filepath.Join(home, "sessions"), jsonl.Options{}).Open(t.Context(), "held")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Close() //nolint:errcheck // test cleanup
	r := runIn(t, home, "--cwd", cwd, "--session", "held", "-p", "y")
	if r.code != ExitFailure || !strings.Contains(r.stderr, "Error: session held is open in another gobble process (pid ") {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
}

// The line session resumes at start and prints one line: the title and the
// message count (owner's decision of 2026-10-06).
func TestLineSessionResumedLine(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	s := llmtest.NewScript(llmtest.Text("A"), llmtest.Text("B"))
	withModel(t, s)
	mustRun(t, home, "--cwd", cwd, "--session-id", "s1", "-p", "earlier question")

	t.Setenv("GOBBLE_HOME", home)
	var stdout, stderr syncBuffer
	caps := term.Caps{In: true, Out: term.Stream{TTY: true, Width: 80}, Err: term.Stream{Width: 80}}
	out := &Output{out: &stdout, err: &stderr, caps: caps}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, intr: &interrupts{}, now: time.Now}
	t.Cleanup(func() { e.logs.close() }) //nolint:errcheck,gosec // test cleanup
	keysR, keysW := io.Pipe()
	ls := &lineSession{e: e, cwd: cwd, in: keysR, raw: func() (func() error, error) { return func() error { return nil }, nil },
		choice: sessionChoice{resume: "s1"}}
	done := make(chan error, 1)
	go func() { done <- ls.run(Prompt{}) }()
	waitUntil(t, func() bool { return strings.Contains(stdout.String(), "> ") })
	if _, err := io.WriteString(keysW, "next\r"); err != nil {
		t.Fatal(err)
	}
	waitUntil(t, func() bool { return len(s.Requests()) == 2 })
	if _, err := io.WriteString(keysW, "\x04"); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout.String(), "resumed earlier question (2 messages)\n") {
		t.Fatalf("stdout %q lacks the resumed line", stdout.String())
	}
	if got := strings.Join(texts(s.Requests()[1]), ","); got != "earlier question,A,next" {
		t.Fatalf("resumed request %q", got)
	}
}

// --session completes from the store, newest first, with keep-order, and
// creates nothing; --tools completes the built-ins (0008-MADR D17).
func TestSessionAndToolCompletion(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	withModel(t, llmtest.NewScript(llmtest.Text("a"), llmtest.Text("b")))
	mustRun(t, home, "--cwd", cwd, "--session-id", "older", "-n", "Old one", "-p", "x")
	mustRun(t, home, "--cwd", cwd, "--session-id", "newer", "-p", "new prompt")
	before := storeFiles(t, home)
	t.Setenv("GOBBLE_HOME", home)
	t.Setenv(complete.EnvShell, "fish")
	t.Setenv(complete.EnvProtocol, complete.Protocol)
	var out, errw bytes.Buffer
	if code := Main(t.Context(), []string{"--", "--session", ""}, nil, &out, &errw); code != ExitOK || errw.Len() != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw.String())
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	// :36 is keep-order with no file completion: the engine adds no-file to
	// every value flag, as 0008-PLAN's record found for enum sources, where
	// 0008-MADR's Confirmation line expected :32. Long descriptions are cut,
	// so the cwd is checked only as far as it shows.
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "newer\tnew prompt · ") || !strings.HasPrefix(lines[1], "older\tOld one · ") ||
		!strings.Contains(lines[1], " · "+cwd[:3]) || lines[2] != ":36" {
		t.Fatalf("session candidates %q", lines)
	}
	if after := storeFiles(t, home); !slices.Equal(before, after) {
		t.Fatalf("completion changed the store: %v → %v", before, after)
	}
	out.Reset()
	if code := Main(t.Context(), []string{"--", "--tools", ""}, nil, &out, &errw); code != ExitOK {
		t.Fatalf("tools exit %d", code)
	}
	for _, n := range []string{"read", "write", "edit", "bash"} {
		if !strings.Contains(out.String(), n+"\n") {
			t.Fatalf("tool candidates %q lack %s", out.String(), n)
		}
	}
}
