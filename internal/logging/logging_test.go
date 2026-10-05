package logging

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
)

func openTest(t *testing.T, level slog.Level, stderr *bytes.Buffer) (*slog.Logger, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "state", "logs", "gobble.log")
	opts := Options{Level: level, File: path}
	if stderr != nil {
		opts.Stderr = stderr
	}
	log, closer, err := Open(opts)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := closer.Close(); err != nil {
			t.Error(err)
		}
	})
	return log, path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestFileGetsLevelStderrGetsWarnings(t *testing.T) {
	var stderr bytes.Buffer
	log, path := openTest(t, slog.LevelInfo, &stderr)
	ctx := t.Context()
	log.DebugContext(ctx, "detail")
	log.InfoContext(ctx, "started", slog.Int("pid", 42))
	log.WarnContext(ctx, "slow provider")
	log.ErrorContext(ctx, "turn failed")

	file := readFile(t, path)
	for _, want := range []string{`"msg":"started"`, `"pid":42`, `"msg":"slow provider"`, `"msg":"turn failed"`} {
		if !strings.Contains(file, want) {
			t.Errorf("file lacks %s:\n%s", want, file)
		}
	}
	if strings.Contains(file, "detail") {
		t.Errorf("debug reached the file at info:\n%s", file)
	}
	if !strings.HasPrefix(file, "{") {
		t.Errorf("file is not JSON lines:\n%s", file)
	}

	errText := stderr.String()
	if !strings.Contains(errText, "slow provider") || !strings.Contains(errText, "turn failed") {
		t.Errorf("stderr lacks the warning or error:\n%s", errText)
	}
	if strings.Contains(errText, "started") || strings.Contains(errText, "detail") {
		t.Errorf("stderr shows below-warning records:\n%s", errText)
	}
}

func TestDebugGoesToTheFileOnly(t *testing.T) {
	var stderr bytes.Buffer
	log, path := openTest(t, slog.LevelDebug, &stderr)
	log.DebugContext(t.Context(), "detail")
	if !strings.Contains(readFile(t, path), "detail") {
		t.Error("debug did not reach the file at debug")
	}
	if stderr.Len() != 0 {
		t.Errorf("debug reached stderr: %q", stderr.String())
	}
}

func TestNoStderrUnderACP(t *testing.T) {
	log, path := openTest(t, slog.LevelInfo, nil)
	log.ErrorContext(t.Context(), "turn failed")
	if !strings.Contains(readFile(t, path), "turn failed") {
		t.Error("the file lacks the error")
	}
}

func TestNoSinksDiscards(t *testing.T) {
	log, closer, err := Open(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if log.Enabled(t.Context(), slog.LevelError) {
		t.Error("a logger with no sinks is enabled")
	}
	if err := closer.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestRedaction(t *testing.T) {
	var stderr bytes.Buffer
	log, path := openTest(t, slog.LevelInfo, &stderr)
	log.WarnContext(t.Context(), "request",
		slog.String("api_key", "sk-live-1"),
		slog.Group("http", slog.String("authorization", "Bearer tok-2"), slog.String("path", "/v1/messages")),
		slog.String("db_password", "pw-3"),
		slog.String("client_secret", "cs-4"),
		slog.String("refresh_token", "rt-5"))

	for name, out := range map[string]string{"file": readFile(t, path), "stderr": stderr.String()} {
		for _, secret := range []string{"sk-live-1", "tok-2", "pw-3", "cs-4", "rt-5"} {
			if strings.Contains(out, secret) {
				t.Errorf("%s leaks %q:\n%s", name, secret, out)
			}
		}
		if !strings.Contains(out, "/v1/messages") || strings.Count(out, redacted) != 5 {
			t.Errorf("%s: want 5 redactions and the path kept:\n%s", name, out)
		}
	}
}

func TestSensitive(t *testing.T) {
	for key, want := range map[string]bool{
		"api_key": true, "API_KEY": true, "access_token": true, "client_secret": true,
		"db_password": true, "Authorization": true, "token": true,
		"path": false, "tokens": false, "keyboard": false, "authorization_url": false, "msg": false,
	} {
		if got := sensitive(key); got != want {
			t.Errorf("sensitive(%q) = %v, want %v", key, got, want)
		}
	}
}

func TestLogFileIsOwnerOnly(t *testing.T) {
	log, path := openTest(t, slog.LevelInfo, nil)
	log.InfoContext(t.Context(), "started")
	ok, err := appdirs.FileIsOwnerOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("the log file is not owner-only")
	}
}

func TestOpenLeavesTheDefaultLogger(t *testing.T) {
	before := slog.Default().Handler()
	openTest(t, slog.LevelInfo, nil)
	if slog.Default().Handler() != before {
		t.Fatal("Open installed a global logger")
	}
}

func TestOpenRejectsARelativeFile(t *testing.T) {
	if _, _, err := Open(Options{File: "logs/gobble.log"}); err == nil {
		t.Fatal("a relative log path was accepted")
	}
}

func TestParseLevel(t *testing.T) {
	for in, want := range map[string]slog.Level{"debug": slog.LevelDebug, "INFO": slog.LevelInfo, " warn ": slog.LevelWarn, "error": slog.LevelError} {
		got, err := ParseLevel(in)
		if err != nil || got != want {
			t.Errorf("ParseLevel(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	if _, err := ParseLevel("loud"); err == nil {
		t.Error("ParseLevel accepted \"loud\"")
	}
}
