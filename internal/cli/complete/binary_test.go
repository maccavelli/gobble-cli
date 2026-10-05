package complete

import (
	"bytes"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// buildGobble builds cmd/gobble once per test run.
var buildGobble = sync.OnceValues(func() (string, error) {
	dir, err := os.MkdirTemp("", "gobble-complete-test-")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(dir, "gobble")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	root, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output() //nolint:noctx // sync.OnceValues has no context
	if err != nil {
		return "", err
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/gobble") //nolint:noctx // as above
	cmd.Dir = strings.TrimSpace(string(root))
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", &buildError{out: string(out), err: err}
	}
	return bin, nil
})

type buildError struct {
	out string
	err error
}

func (e *buildError) Error() string { return "go build: " + e.err.Error() + "\n" + e.out }

func gobbleBinary(t *testing.T) string {
	t.Helper()
	bin, err := buildGobble()
	if err != nil {
		t.Fatal(err)
	}
	return bin
}

// isolatedEnv points every directory gobble could read or write at home,
// and keeps only what a process needs to start.
func isolatedEnv(home string, kv ...string) []string {
	env := []string{"PATH=" + os.Getenv("PATH")}
	for _, k := range []string{"SYSTEMROOT", "TEMP", "TMP"} {
		if v := os.Getenv(k); v != "" {
			env = append(env, k+"="+v)
		}
	}
	for _, k := range []string{"HOME", "USERPROFILE", "APPDATA", "LOCALAPPDATA", "GOBBLE_HOME",
		"XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_STATE_HOME", "XDG_CACHE_HOME"} {
		env = append(env, k+"="+home)
	}
	return append(env, kv...)
}

func runBinary(t *testing.T, cwd string, env []string, args ...string) (stdout, stderr string) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), gobbleBinary(t), args...)
	cmd.Dir, cmd.Env = cwd, env
	var out, errw bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errw
	if err := cmd.Run(); err != nil {
		t.Fatalf("gobble %v: %v\n%s", args, err, errw.String())
	}
	return out.String(), errw.String()
}

// The built binary in completion mode, with an explicit environment: U+001F
// words, a multibyte path, and C6: an empty home stays empty, nothing goes
// to stderr.
func TestBinaryCompletionMode(t *testing.T) {
	home, cwd := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(cwd, "dés.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		env  []string
		args []string
		want string
	}{
		{"powershell U+001F words", []string{EnvShell + "=powershell", EnvProtocol + "=1", EnvWords + "=--output-format\x1fj"},
			nil, "json\t\tvalue\n:36\n"},
		{"multibyte path", []string{EnvShell + "=fish", EnvProtocol + "=1"}, []string{"--", "--file", "dé"},
			"dés.txt\n:4\n"}, // one file left: the shell may add its space
		{"sessions read nothing yet", []string{EnvShell + "=zsh", EnvProtocol + "=1"}, []string{"--", "--session", ""},
			":36\n"},
		{"bash line", []string{EnvShell + "=bash", EnvProtocol + "=1", EnvLine + "=gobble completion f", EnvPoint + "=-1", EnvCur + "=f"},
			nil, "fish\n:36\n"},
		{"protocol mismatch", []string{EnvShell + "=fish", EnvProtocol + "=2"}, []string{"--", ""}, ":1\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, errText := runBinary(t, cwd, isolatedEnv(home, tc.env...), tc.args...)
			if out != tc.want {
				t.Fatalf("stdout = %q, want %q", out, tc.want)
			}
			if errText != "" {
				t.Fatalf("completion mode wrote to stderr: %q", errText)
			}
		})
	}
	var created []string
	if err := filepath.WalkDir(home, func(p string, _ fs.DirEntry, err error) error {
		if p != home {
			created = append(created, p)
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 {
		t.Fatalf("completion mode wrote under the home directory (C6): %q", created)
	}
}
