package logging

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// openTerminal is a logger whose stderr is a terminal width columns wide,
// with home as the user's home directory.
func openTerminal(t *testing.T, width int, home string) (*slog.Logger, *bytes.Buffer, string) {
	t.Helper()
	var stderr bytes.Buffer
	path := filepath.Join(t.TempDir(), "state", "logs", "gobble.log")
	log, closer, err := Open(Options{Level: slog.LevelInfo, File: path, Stderr: &stderr, TerminalWidth: width, Home: home})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := closer.Close(); err != nil {
			t.Error(err)
		}
	})
	return log, &stderr, path
}

// On a terminal a warning is gobble's own diagnostic: no time, no level
// word, no quoting, its attributes in parentheses, a path under home with
// ~ (0024-MADR D6).
func TestTerminalWarning(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	log, stderr, path := openTerminal(t, 200, home)
	log.WarnContext(t.Context(), "prompt: a file was not used",
		slog.String("file", filepath.Join(home, ".agents", "skills")),
		slog.String("reason", "not loaded: the project is not trusted (gobble trust, or /trust)"))
	want := "Warning: prompt: a file was not used (file: " + filepath.Join("~", ".agents", "skills") +
		"; reason: not loaded: the project is not trusted (gobble trust, or /trust))\n"
	if got := stderr.String(); got != want {
		t.Fatalf("stderr %q\nwant   %q", got, want)
	}
	if file := readFile(t, path); !strings.Contains(file, `"msg":"prompt: a file was not used"`) || !strings.Contains(file, `"level":"WARN"`) {
		t.Fatalf("the file lacks the record as JSON:\n%s", file)
	}
}

// An error is "Error:", and a record with no attributes has no parentheses.
func TestTerminalError(t *testing.T) {
	log, stderr, _ := openTerminal(t, 200, "")
	log.ErrorContext(t.Context(), "turn failed")
	if got := stderr.String(); got != "Error: turn failed\n" {
		t.Fatalf("stderr %q", got)
	}
}

// A long diagnostic wraps at spaces within the width less one, its
// continuation lines indented two columns, and loses no word.
func TestTerminalWraps(t *testing.T) {
	log, stderr, _ := openTerminal(t, 60, "")
	reason := "not loaded: the project is not trusted (gobble trust, or /trust), and the rest of this reason is long"
	log.WarnContext(t.Context(), "prompt: a file was not used", slog.String("file", "/srv/work/.agents/skills"), slog.String("reason", reason))
	lines := strings.Split(strings.TrimSuffix(stderr.String(), "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("not wrapped: %q", stderr.String())
	}
	for i, line := range lines {
		if n := utf8.RuneCountInString(line); n > 59 {
			t.Errorf("line %d is %d columns: %q", i+1, n, line)
		}
		if i > 0 && (!strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "   ")) {
			t.Errorf("line %d is not indented two columns: %q", i+1, line)
		}
	}
	joined := strings.Join(strings.Fields(stderr.String()), " ")
	if want := "Warning: prompt: a file was not used (file: /srv/work/.agents/skills; reason: " + reason + ")"; joined != want {
		t.Fatalf("words lost or changed:\n%q\nwant\n%q", joined, want)
	}
}

// Redaction applies to the terminal form as to the text handler.
func TestTerminalRedacts(t *testing.T) {
	log, stderr, _ := openTerminal(t, 400, "")
	log.WarnContext(t.Context(), "request",
		slog.String("api_key", "sk-live-1"),
		slog.Group("http", slog.String("authorization", "Bearer tok-2"), slog.String("path", "/v1/messages")),
		slog.String("db_password", "pw-3"),
		slog.String("client_secret", "cs-4"),
		slog.String("refresh_token", "rt-5"))
	out := stderr.String()
	for _, secret := range []string{"sk-live-1", "tok-2", "pw-3", "cs-4", "rt-5"} {
		if strings.Contains(out, secret) {
			t.Errorf("stderr leaks %q: %s", secret, out)
		}
	}
	if !strings.Contains(out, "http.path: /v1/messages") || !strings.Contains(out, "http.authorization: "+redacted) || strings.Count(out, redacted) != 5 {
		t.Errorf("want 5 redactions, the group's keys and the path kept: %s", out)
	}
}

// With Show, each warning goes to it as one diagnostic, unwrapped, with its
// attributes, redaction and ~, and nothing is written to stderr: the caller
// wraps it when it is shown (0030-MADR D4, as amended).
func TestTerminalShow(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	var stderr bytes.Buffer
	var shown []string
	log, closer, err := Open(Options{Level: slog.LevelInfo, Stderr: &stderr, TerminalWidth: 40, Home: home,
		Show: func(d string) { shown = append(shown, d) }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := closer.Close(); err != nil {
			t.Error(err)
		}
	})
	reason := "not loaded: the project is not trusted (gobble trust, or /trust), and the rest of this reason is long"
	log.WarnContext(t.Context(), "prompt: a file was not used", slog.String("file", filepath.Join(home, "skills")),
		slog.String("api_key", "sk-live-1"), slog.String("reason", reason))
	log.InfoContext(t.Context(), "started")
	want := "Warning: prompt: a file was not used (file: " + filepath.Join("~", "skills") + "; api_key: " + redacted +
		"; reason: " + reason + ")"
	if len(shown) != 1 || shown[0] != want {
		t.Fatalf("shown %q\nwant  [%q]", shown, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("written to stderr as well: %q", stderr.String())
	}
}

// Wrap draws a diagnostic exactly as the handler writes it at that width.
func TestWrapIsTheHandlers(t *testing.T) {
	reason := "not loaded: the project is not trusted (gobble trust, or /trust), and the rest of this reason is long"
	for _, width := range []int{30, 60, 200} {
		log, stderr, _ := openTerminal(t, width, "")
		log.WarnContext(t.Context(), "prompt: a file was not used", slog.String("reason", reason))
		if got := Wrap("Warning: prompt: a file was not used (reason: "+reason+")", width); got != stderr.String() {
			t.Errorf("at %d: Wrap %q\nhandler %q", width, got, stderr.String())
		}
	}
}

// With no width known, Wrap leaves a diagnostic on one line.
func TestWrapNoWidth(t *testing.T) {
	for _, width := range []int{0, 1} {
		if got := Wrap("Warning: model listing failed (provider: kilo)", width); got != "Warning: model listing failed (provider: kilo)\n" {
			t.Errorf("at %d: %q", width, got)
		}
	}
}

// Below warning level nothing reaches a terminal; the file still has it.
func TestTerminalShowsWarningsOnly(t *testing.T) {
	log, stderr, path := openTerminal(t, 200, "")
	log.InfoContext(t.Context(), "started", slog.Int("pid", 42))
	if stderr.Len() != 0 {
		t.Fatalf("info reached the terminal: %q", stderr.String())
	}
	if !strings.Contains(readFile(t, path), `"msg":"started"`) {
		t.Fatal("the file lacks the info record")
	}
}

// A logger's own attributes and groups are shown with each record's.
func TestTerminalWithAttrs(t *testing.T) {
	log, stderr, _ := openTerminal(t, 200, "")
	log.With(slog.String("session", "s1")).WithGroup("tool").WarnContext(t.Context(), "slow", slog.Int("ms", 900))
	if got := stderr.String(); got != "Warning: slow (session: s1; tool.ms: 900)\n" {
		t.Fatalf("stderr %q", got)
	}
}
