package cli

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	lib "github.com/maccavelli/go-selfupdate-lib/buildinfo"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/buildinfo"
)

var update = flag.Bool("update", false, "rewrite testdata/*.golden")

type result struct {
	code           int
	stdout, stderr string
}

// runMain runs Main with a fresh GOBBLE_HOME, so nothing touches the user's
// directories. stdin is nil unless given.
func runMain(t *testing.T, stdin *os.File, args ...string) result {
	t.Helper()
	t.Setenv("GOBBLE_HOME", t.TempDir())
	var out, errw bytes.Buffer
	code := Main(t.Context(), args, stdin, &out, &errw)
	return result{code: code, stdout: out.String(), stderr: errw.String()}
}

// stdinFile returns a regular file holding s, opened for reading: a
// non-terminal stdin, as a pipe would be.
func stdinFile(t *testing.T, s string) *os.File {
	t.Helper()
	p := filepath.Join(t.TempDir(), "stdin")
	if err := os.WriteFile(p, []byte(s), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Error(err)
		}
	})
	return f
}

func TestMainExitCodes(t *testing.T) {
	cases := []struct {
		name   string
		args   []string
		code   int
		stderr string
	}{
		{"print mode without a credential exits 3", []string{"-p", "x"}, ExitAuth, "Error: no model credential: set one of "},
		{"words alone are the default command", []string{"fix", "the", "test"}, ExitAuth, "no model credential"},
		{"--no-tools with --tools", []string{"--no-tools", "--tools", "read", "x"}, ExitUsage, "Error: --no-tools cannot be combined with --tools\n"},
		{"an unknown tool", []string{"--tools", "read,nope", "x"}, ExitUsage, `Error: unknown tool "nope" (the tools are read, write, edit, bash)`},
		{"--cwd must be a directory", []string{"--cwd", "no-such-dir", "-p", "x"}, ExitUsage, "Error: --cwd no-such-dir: not a directory\n"},
		{"no prompt in print mode", nil, ExitUsage, "Error: no prompt: give words, @path, --file or piped input\n"},
		{"--fork with --session", []string{"--fork", "a", "--session", "b", "x"}, ExitUsage, "Error: --fork cannot be combined with --session\n"},
		{"--fork with -c", []string{"--fork", "a", "-c", "x"}, ExitUsage, "--fork cannot be combined with --continue"},
		{"--session-id with --session", []string{"--session-id", "i", "--session", "s", "x"}, ExitUsage, "--session-id cannot be combined with --session"},
		{"--provider without --model", []string{"--provider", "anthropic", "x"}, ExitUsage, "--provider requires --model"},
		{"--file with @path", []string{"--file", "f.md", "@g.md"}, ExitUsage, "--file cannot be combined with an @path argument"},
		{"missing @path", []string{"-p", "@no-such-file.md"}, ExitUsage, "Error: @no-such-file.md: no such file\n"},
		{"missing --file", []string{"--file", "no-such-file.md"}, ExitUsage, "--file no-such-file.md:"},
		{"unknown flag", []string{"--bogus"}, ExitUsage, "Error: unknown flag --bogus\n"},
		{"bad enum", []string{"--output-format", "yaml", "x"}, ExitUsage, `--output-format must be one of "text","json","stream-json"`},
		{"bad log level", []string{"--log-level", "loud", "version"}, ExitUsage, "--log-level must be one of"},
		{"acp serves until end of file", []string{"acp"}, ExitOK, ""},
		{"completion needs a known shell", []string{"completion", "nu"}, ExitUsage, `<shell> must be one of "bash","zsh","fish","powershell"`},
		{"version", []string{"version"}, ExitOK, ""},
		{"config path", []string{"config", "path"}, ExitOK, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := runMain(t, nil, tc.args...)
			if r.code != tc.code {
				t.Fatalf("exit %d, want %d; stderr %q", r.code, tc.code, r.stderr)
			}
			if tc.stderr == "" && r.stderr != "" || !strings.Contains(r.stderr, tc.stderr) {
				t.Fatalf("stderr = %q, want it to contain %q", r.stderr, tc.stderr)
			}
			if tc.code != ExitOK && r.stdout != "" {
				t.Fatalf("a failing run wrote to stdout: %q", r.stdout)
			}
		})
	}
}

// Every flag a later plan delivers is parsed and rejected, naming that plan
// (0008-MADR D12: never silently ignored).
func TestLaterFlagsAreRejected(t *testing.T) {
	value := map[string]bool{
		"tools": true, "exclude-tools": true, "session": true,
		"session-id": true, "fork": true, "session-dir": true, "name": true, "system-prompt": true,
		"append-system-prompt": true, "provider": true, "model": true, "thinking": true, "api-key": true,
	}
	sample := map[string]string{"thinking": "low"}
	names := make([]string, 0, len(laterFlags))
	for n := range laterFlags {
		names = append(names, n)
	}
	slices.Sort(names)
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			args := []string{"--" + name}
			if value[name] {
				v := cmpOr(sample[name], "v")
				args = append(args, v)
			}
			if name == "provider" {
				args = append(args, "--model", "m")
			}
			r := runMain(t, nil, append(args, "x")...)
			want := "--" + name + " is not yet available (" + laterFlags[name] + ")"
			if name == "provider" {
				want = "--model is not yet available (0005-PLAN F4)" // the first, alphabetically
			}
			if r.code != ExitUsage || !strings.Contains(r.stderr, want) {
				t.Fatalf("exit %d, stderr %q; want 2 and %q", r.code, r.stderr, want)
			}
		})
	}
	r := runMain(t, nil, "--no-approve", "x")
	if r.code != ExitUsage || !strings.Contains(r.stderr, "--approve is not yet available (0005-PLAN F6)") {
		t.Fatalf("--no-approve: exit %d, stderr %q", r.code, r.stderr)
	}
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// --help prints usage and exits 0 without running a command. With an Exit
// that returned instead of panicking, Kong would go on to run chat.
func TestHelp(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"chat", "--help"}, {"config", "path", "--help"}} {
		r := runMain(t, nil, args...)
		if r.code != ExitOK || r.stderr != "" || !strings.Contains(r.stdout, "Usage: gobble") {
			t.Fatalf("%v: exit %d, stderr %q, stdout %.80q", args, r.code, r.stderr, r.stdout)
		}
	}
}

// gobble acp writes nothing to stdout but the protocol (0004-PLAN Phase 2
// Accept): with a closed stdin it serves, reaches end of file and exits 0
// without a byte on stdout or stderr (0002-PLAN Phase 1).
func TestACPWritesNothingToStdout(t *testing.T) {
	for name, stdin := range map[string]*os.File{"empty stdin": stdinFile(t, ""), "no stdin": nil} {
		r := runMain(t, stdin, "acp")
		if r.stdout != "" || r.stderr != "" {
			t.Fatalf("%s: gobble acp wrote stdout %q, stderr %q", name, r.stdout, r.stderr)
		}
		if r.code != ExitOK {
			t.Fatalf("%s: exit %d, want 0 at end of file", name, r.code)
		}
	}
}

func TestVersionJSONGolden(t *testing.T) {
	old := identity
	t.Cleanup(func() { identity = old })
	identity = func() buildinfo.Info {
		return buildinfo.Info{Version: "1.2.3", Commit: "4e25c8a0123456789abcdef0123456789abcdef0",
			Date: "2026-10-05T13:14:29Z", Go: "1.27.1", OS: "linux", Arch: "amd64", Release: true,
			Lib: lib.Info{Version: "v1.2.3", Kind: lib.KindRelease}}
	}
	for name, args := range map[string][]string{"version.json": {"version", "--json"}, "version.txt": {"version"}} {
		r := runMain(t, nil, args...)
		if r.code != ExitOK || r.stderr != "" {
			t.Fatalf("%v: exit %d, stderr %q", args, r.code, r.stderr)
		}
		golden(t, name, r.stdout)
	}
}

func TestConfigPath(t *testing.T) {
	home := t.TempDir()
	var out, errw bytes.Buffer
	t.Setenv("GOBBLE_HOME", home)
	if code := Main(t.Context(), []string{"config", "path"}, nil, &out, &errw); code != ExitOK {
		t.Fatalf("exit %d: %s", code, errw.String())
	}
	want := "config: " + home + "\ndata:   " + home + "\nstate:  " + home + "\ncache:  " + home + "\n"
	if out.String() != want {
		t.Fatalf("stdout = %q, want %q", out.String(), want)
	}
	out.Reset()
	if code := Main(t.Context(), []string{"config", "path", "--json"}, nil, &out, &errw); code != ExitOK {
		t.Fatalf("--json: exit %d", code)
	}
	for _, key := range []string{`"config":`, `"data":`, `"state":`, `"cache":`} {
		if !strings.Contains(out.String(), key) {
			t.Fatalf("JSON lacks %s: %s", key, out.String())
		}
	}
	des, err := os.ReadDir(home)
	if err != nil || len(des) != 0 {
		t.Fatalf("config path created %v (%v); it must create nothing", des, err)
	}
}

func TestConfigPathRelativeHomeFails(t *testing.T) {
	t.Setenv("GOBBLE_HOME", "relative")
	var out, errw bytes.Buffer
	if code := Main(t.Context(), []string{"config", "path"}, nil, &out, &errw); code != ExitFailure ||
		!strings.Contains(errw.String(), "GOBBLE_HOME") {
		t.Fatalf("exit %d, stderr %q", code, errw.String())
	}
}

// Piped stdin and words reach the composed prompt (D4): the agent receives
// it, and chat's debug record in the log file has its size.
func TestPipedInputIsComposed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{}}}
	withAgent(t, agent)
	var out, errw bytes.Buffer
	code := Main(t.Context(), []string{"--log-level", "debug", "ask"}, stdinFile(t, "hi\n"), &out, &errw)
	if p := agent.Prompts(); code != ExitOK || len(p) != 1 || !strings.Contains(p[0], `"text":"hi\n\nask"`) {
		t.Fatalf("exit %d, prompts %q, stderr %q; want 0 and the composed prompt sent", code, agent.Prompts(), errw.String())
	}
	log, err := os.ReadFile(filepath.Join(home, "logs", "gobble.log"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(log), `"mode":"print"`) || !strings.Contains(string(log), `"prompt_bytes":7`) {
		t.Fatalf("log does not show the composed 7-byte prompt \"hi\\n\\nask\":\n%s", log)
	}
	if strings.Contains(errw.String(), "chat") {
		t.Fatalf("a debug record reached stderr: %q", errw.String())
	}
}

func TestFileWithPipedStdinIsAUsageError(t *testing.T) {
	r := runMain(t, stdinFile(t, "hi"), "--file", "f.md")
	if r.code != ExitUsage || !strings.Contains(r.stderr, "--file cannot be combined with piped input") {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
}

// Commands that do not log create no state: no logs directory.
func TestVersionCreatesNoFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	var out, errw bytes.Buffer
	if code := Main(t.Context(), []string{"version"}, nil, &out, &errw); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if des, _ := os.ReadDir(home); len(des) != 0 {
		t.Fatalf("version created %v", des)
	}
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *update {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%v (run with -update to create it)", err)
	}
	if string(want) != got {
		t.Fatalf("%s differs from %s:\n--- got\n%s--- want\n%s", name, path, got, want)
	}
}
