package complete

import (
	"bytes"
	"context"
	"iter"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func runProtocol(t *testing.T, reg Registry, env map[string]string, args ...string) string {
	t.Helper()
	var out bytes.Buffer
	if code := Run(t.Context(), newTestParser(t).Model, reg, func(k string) string { return env[k] }, args, &out); code != 0 {
		t.Fatalf("Run returned %d", code)
	}
	return out.String()
}

func shellEnv(shell string, kv ...string) map[string]string {
	m := map[string]string{EnvShell: shell, EnvProtocol: Protocol}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i]] = kv[i+1]
	}
	return m
}

func TestProtocolShells(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		args []string
		want string
	}{
		{"fish words after --", shellEnv("fish"), []string{"--", "--output-format", ""}, "text\njson\nstream-json\n:36\n"},
		{"zsh words after --", shellEnv("zsh"), []string{"--", "--output-format", "j"}, "json\n:36\n"},
		{"bash line, = word break stripped", shellEnv("bash", EnvLine, "gobble --log-level=", EnvPoint, "-1", EnvCur, "="),
			nil, "=debug\n=info\n:36\n"},
		{"bash line, : word break stripped", shellEnv("bash", EnvLine, "gobble --model=openai/gpt:h", EnvPoint, "-1", EnvCur, "h"),
			nil, "high\n:36\n"},
		{"bash point cuts the line", shellEnv("bash", EnvLine, "gobble --output-format json", EnvPoint, "25", EnvCur, "js"),
			nil, "json\n:36\n"},
		// The program name is not a word: "shell" must be read as the
		// command, not as chat's first argument.
		{"bash drops the program name", shellEnv("bash", EnvLine, "gobble shell z", EnvPoint, "-1", EnvCur, "z"),
			nil, "zsh\n:36\n"},
		{"powershell U+001F words, three fields", shellEnv("powershell", EnvWords, "--output-format\x1fs"),
			nil, "stream-json\t\tvalue\n:36\n"},
		{"powershell flag kind", shellEnv("powershell", EnvWords, "--outp"),
			nil, "--output-format\t\tflag\n:4\n"},
		{"protocol mismatch", map[string]string{EnvShell: "fish", EnvProtocol: "2"}, []string{"--", ""}, ":1\n"},
		{"no protocol", map[string]string{EnvShell: "fish"}, []string{"--", ""}, ":1\n"},
		{"unknown shell", shellEnv("tcsh"), []string{"--", ""}, ":1\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := runProtocol(t, testRegistry, tc.env, tc.args...); got != tc.want {
				t.Fatalf("output = %q, want %q", got, tc.want)
			}
		})
	}
}

func fixed(cs ...Candidate) Completer {
	return CompleterFunc(func(context.Context, string) (iter.Seq[Candidate], Directive) {
		return func(yield func(Candidate) bool) {
			for _, c := range cs {
				if !yield(c) {
					return
				}
			}
		}, DirectiveKeepOrder
	})
}

func TestProtocolSanitisesAndCaps(t *testing.T) {
	long := strings.Repeat("é", 100)
	reg := Registry{"model": fixed(
		Candidate{Value: "evil\x1b[2Jone", Description: "line1\nline2\t\x1b]0;title\x07end"},
		Candidate{Value: "tab\tvalue"},
		Candidate{Value: "two", Description: long},
		Candidate{Value: strings.Repeat("v", 120)},
	)}
	got := runProtocol(t, reg, shellEnv("fish"), "--", "--model", "")
	lines := strings.Split(strings.TrimSuffix(got, "\n"), "\n")
	want0 := "evilone\tline1 line2 end"
	if lines[0] != want0 {
		t.Fatalf("line 0 = %q, want %q", lines[0], want0)
	}
	if strings.Contains(got, "tab\tvalue") || strings.Contains(got, "tab") {
		t.Fatalf("a value holding a tab was emitted: %q", got)
	}
	desc := strings.SplitN(lines[1], "\t", 2)[1]
	if n := len([]rune(desc)); n != maxDescription || !strings.HasSuffix(desc, "…") {
		t.Fatalf("description is %d runes (%q), want %d ending in …", n, desc, maxDescription)
	}
	if lines[2] != strings.Repeat("v", 120) {
		t.Fatalf("a long value was cut: %q (0008-PLAN P5 deviation 3)", lines[2])
	}
}

// The deadline returns what was found so far, keep-order, without an error.
func TestProtocolDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		slow := CompleterFunc(func(ctx context.Context, _ string) (iter.Seq[Candidate], Directive) {
			return func(yield func(Candidate) bool) {
				if !yield(Candidate{Value: "first"}) {
					return
				}
				time.Sleep(time.Second)
				yield(Candidate{Value: "late"})
			}, 0
		})
		start := time.Now()
		got := runProtocol(t, Registry{"model": slow}, shellEnv("fish"), "--", "--model", "")
		if got != "first\n:36\n" {
			t.Fatalf("output = %q, want the first candidate and keep-order", got)
		}
		if el := time.Since(start); el > time.Second+deadline {
			t.Fatalf("took %v", el)
		}
	})
}

// One file left takes a space; one directory does not.
func TestProtocolPathSpacing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o700); err != nil {
		t.Fatal(err)
	}
	reg := Registry{"model": PathSource(false)} // any registered path source
	sep := string(filepath.Separator)
	for prefix, want := range map[string]string{
		dir + sep + "no": dir + sep + "notes.md\n:4\n",
		dir + sep + "su": dir + sep + "sub" + sep + "\n:6\n",
	} {
		if got := runProtocol(t, reg, shellEnv("fish"), "--", "--model", prefix); got != want {
			t.Errorf("%s: output = %q, want %q", prefix, got, want)
		}
	}
}

func TestProtocolDebugFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "debug.log")
	runProtocol(t, testRegistry, shellEnv("fish", EnvDebug, path), "--", "--out")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "shell=fish") || !strings.Contains(string(b), "n=1") {
		t.Fatalf("debug line = %q", b)
	}
}
