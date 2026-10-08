package builtin

import (
	"context"
	"encoding/json/jsontext"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/tool"
)

func needBash(t *testing.T) {
	t.Helper()
	if _, err := findBash(); err != nil {
		t.Skipf("no usable bash: %v", err)
	}
}

// testOptions are the defaults with the output kept in a test directory.
func testOptions(t *testing.T) Options {
	t.Helper()
	return Options{OutputDir: t.TempDir()}.withDefaults()
}

func runBash(t *testing.T, o Options, env tool.Env, args map[string]any) tool.Result {
	t.Helper()
	return call(t, bashWith(o), env, args)
}

func TestBash(t *testing.T) {
	needBash(t)
	o, env := testOptions(t), tool.Env{Cwd: t.TempDir()}
	if r := runBash(t, o, env, map[string]any{"command": "echo hi"}); r.IsError || r.Text() != "hi\n[exit status 0]" || r.Summary != "exit status 0: hi" {
		t.Fatalf("echo = %+v, text %q", r, r.Text())
	}
	if r := runBash(t, o, env, map[string]any{"command": "echo no >&2; exit 3"}); !r.IsError || r.Text() != "no\n[exit status 3]" {
		t.Fatalf("failing command = %q, isError %v", r.Text(), r.IsError)
	}
	if err := os.Mkdir(filepath.Join(env.Cwd, "sub"), 0o750); err != nil {
		t.Fatal(err)
	}
	if r := runBash(t, o, env, map[string]any{"command": `basename "$PWD"`, "workdir": "sub"}); r.Text() != "sub\n[exit status 0]" {
		t.Fatalf("workdir = %q", r.Text())
	}
	if r := runBash(t, o, env, map[string]any{"command": "true", "workdir": "missing"}); !r.IsError || !strings.HasPrefix(r.Text(), "bash: workdir missing: ") {
		t.Fatalf("missing workdir = %q", r.Text())
	}
	o.ShellCommandPrefix = "X=from-the-prefix"
	if r := runBash(t, o, env, map[string]any{"command": "echo $X"}); r.Text() != "from-the-prefix\n[exit status 0]" {
		t.Fatalf("prefix = %q", r.Text())
	}
}

// A command sees its own session's markers, never ones inherited from a
// gobble that started this one.
func TestBashMarkers(t *testing.T) {
	needBash(t)
	t.Setenv("GOBBLE_SESSION_ID", "the-parent's")
	t.Setenv("GOBBLE_MODEL", "inherited")
	env := tool.Env{Cwd: t.TempDir(), Environ: []string{"AI_AGENT=gobble", "GOBBLE_SESSION_ID=s1"}}
	r := runBash(t, testOptions(t), env, map[string]any{"command": `echo "$AI_AGENT $GOBBLE_SESSION_ID [$GOBBLE_MODEL]"`})
	if r.Text() != "gobble s1 []\n[exit status 0]" {
		t.Fatalf("markers = %q", r.Text())
	}
}

func TestCommandTimeout(t *testing.T) {
	for _, tc := range []struct {
		seconds int
		want    time.Duration
	}{{0, 120 * time.Second}, {5, 5 * time.Second}, {9999, 600 * time.Second}} {
		if got := commandTimeout(tc.seconds, defaultMaxTimeout); got != tc.want {
			t.Errorf("commandTimeout(%d) = %v, want %v", tc.seconds, got, tc.want)
		}
	}
}

// The timeout stops the command and what it started: the background
// subshell would write its marker after 2 s, and must not.
func TestBashTimeoutStopsTheTree(t *testing.T) {
	needBash(t)
	env := tool.Env{Cwd: t.TempDir()}
	start := time.Now()
	r := runBash(t, testOptions(t), env, map[string]any{"command": "(sleep 2; echo late > marker) & sleep 30", "timeout": 1})
	if d := time.Since(start); d > 6*time.Second {
		t.Fatalf("the stop took %v", d)
	}
	if !r.IsError || !strings.HasSuffix(r.Text(), "[stopped after 1 s; give a larger timeout, at most 600]") {
		t.Fatalf("stopped command = %q", r.Text())
	}
	time.Sleep(3 * time.Second)
	if _, err := os.Stat(filepath.Join(env.Cwd, "marker")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("the background subshell outlived the stop (marker: %v)", err)
	}
}

// SIGTERM comes first, so a command's trap can clean up before the kill.
func TestBashTermBeforeKill(t *testing.T) {
	needBash(t)
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no signal for a tree; its Job object ends it at once")
	}
	env := tool.Env{Cwd: t.TempDir()}
	runBash(t, testOptions(t), env, map[string]any{"command": "trap 'echo cleaned > trapped; exit 0' TERM; sleep 30 & wait", "timeout": 1})
	if b, err := os.ReadFile(filepath.Join(env.Cwd, "trapped")); err != nil || string(b) != "cleaned\n" { //nolint:gosec // test file
		t.Fatalf("the TERM trap did not run: %q, %v", b, err)
	}
}

// A cancelled context stops the command and ends the call at once.
func TestBashCancel(t *testing.T) {
	needBash(t)
	ctx, cancel := context.WithCancel(t.Context())
	time.AfterFunc(200*time.Millisecond, cancel)
	start := time.Now()
	_, err := bashWith(testOptions(t)).Run(ctx, tool.Call{Name: "bash", Args: jsontext.Value(`{"command":"sleep 30"}`)}, tool.Env{Cwd: t.TempDir()})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if d := time.Since(start); d > 6*time.Second {
		t.Fatalf("cancel took %v", d)
	}
}

// A long output gives the model its tail and the file holding all of it.
func TestBashLongOutput(t *testing.T) {
	needBash(t)
	o := testOptions(t)
	r := runBash(t, o, tool.Env{Cwd: t.TempDir()}, map[string]any{"command": `for i in $(seq 1 3000); do echo "line $i"; done`})
	text := r.Text()
	head, rest, _ := strings.Cut(text, "\n")
	const lead = "(output cut: the last 2000 lines of 3000 are shown; the whole output is in "
	if !strings.HasPrefix(head, lead) || !strings.HasSuffix(rest, "line 3000\n[exit status 0]") || strings.Contains(rest, "line 1000\n") {
		t.Fatalf("output starts %q and ends %q", head, rest[max(0, len(rest)-40):])
	}
	path := strings.TrimSuffix(strings.TrimPrefix(head, lead), ")")
	if filepath.Dir(path) != o.OutputDir || !strings.HasPrefix(filepath.Base(path), "gobble-bash-") {
		t.Fatalf("log %s is not a gobble-bash log in %s", path, o.OutputDir)
	}
	b, err := os.ReadFile(path) //nolint:gosec // the log the tool named
	if err != nil || strings.Count(string(b), "\n") != 3000 || !strings.HasPrefix(string(b), "line 1\n") {
		t.Fatalf("log has %d lines, %v", strings.Count(string(b), "\n"), err)
	}
}

func TestSweep(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	files := map[string]time.Duration{
		"gobble-bash-old.log": 8 * 24 * time.Hour,
		"gobble-bash-new.log": 6 * 24 * time.Hour,
		"other.txt":           8 * 24 * time.Hour,
	}
	for name, age := range files {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(p, now.Add(-age), now.Add(-age)); err != nil {
			t.Fatal(err)
		}
	}
	sweep(dir, now, outputKeep)
	for name, keep := range map[string]bool{"gobble-bash-old.log": false, "gobble-bash-new.log": true, "other.txt": true} {
		_, err := os.Stat(filepath.Join(dir, name))
		if (err == nil) != keep {
			t.Errorf("%s: kept %v, want %v", name, err == nil, keep)
		}
	}
}

// read may open the output directory without asking; write still asks.
func TestOutputDirIsAReadRoot(t *testing.T) {
	o := testOptions(t)
	cwd := t.TempDir()
	env := tool.Env{Cwd: cwd, Roots: []string{cwd}}
	c := tool.Call{Args: jsontext.Value(`{"path":` + quoteJSON(filepath.Join(o.OutputDir, "gobble-bash-1.log")) + `}`)}
	if got := readWith(o).(tool.Confined).Outside(c, env); got != nil {
		t.Fatalf("read asks about the output directory: %q", got)
	}
	if got := Write().(tool.Confined).Outside(c, env); len(got) != 1 {
		t.Fatalf("write does not ask about the output directory: %q", got)
	}
}

func TestPowerShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("powershell is a Windows tool")
	}
	if _, err := findPowerShell(); err != nil {
		t.Skip(err)
	}
	o, env := testOptions(t), tool.Env{Cwd: t.TempDir()}
	ps := powershellWith(o)
	if r := call(t, ps, env, map[string]any{"command": "Write-Output hi"}); r.IsError || r.Text() != "hi\n[exit status 0]" {
		t.Fatalf("Write-Output = %q", r.Text())
	}
	if r := call(t, ps, env, map[string]any{"command": "Write-Output ([char]0x00e9)"}); r.Text() != "é\n[exit status 0]" {
		t.Fatalf("non-ASCII output = %q", r.Text())
	}
}

func quoteJSON(s string) string {
	b, err := jsontext.AppendQuote(nil, s)
	if err != nil {
		panic(err)
	}
	return string(b)
}
