package cli

import (
	"bytes"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/alecthomas/kong"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
)

// Every flag and argument that takes a value names its completion: an enum
// or a complete:"<source>" tag, and every source is registered. No value
// silently lacks completion (0008-MADR D17).
func TestCompletionCoverage(t *testing.T) {
	k, err := newParser(&Root{}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	reg := completionRegistry()
	check := func(where string, v *kong.Value) {
		if v.IsBool() || v.IsCounter() {
			return
		}
		src := ""
		if v.Tag != nil {
			src = v.Tag.Get("complete")
		}
		switch {
		case src == "" && v.Enum == "":
			t.Errorf("%s %s takes a value but has neither enum nor complete tag", where, v.Name)
		case src != "" && src != "none" && reg[src] == nil:
			t.Errorf("%s %s names complete:%q, which is not registered", where, v.Name, src)
		}
	}
	var walk func(n *kong.Node)
	walk = func(n *kong.Node) {
		for _, f := range n.Flags {
			check(n.Name, f.Value)
		}
		for _, p := range n.Positional {
			check(n.Name, p)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(k.Model.Node)
}

func TestThinkingLevelsMatchTheFlag(t *testing.T) {
	k, err := newParser(&Root{}, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range k.Model.Children {
		for _, f := range c.Flags {
			if f.Name == "thinking" {
				got := slices.DeleteFunc(f.EnumSlice(), func(s string) bool { return s == "" })
				if !slices.Equal(got, thinkingLevels) {
					t.Fatalf("--thinking enum %q differs from thinkingLevels %q", got, thinkingLevels)
				}
				return
			}
		}
	}
	t.Fatal("no --thinking flag")
}

func TestCompletionScriptsPrinted(t *testing.T) {
	for _, sh := range complete.Shells {
		r := runMain(t, nil, "completion", sh)
		want, err := complete.Script(sh, "gobble")
		if err != nil {
			t.Fatal(err)
		}
		if r.code != ExitOK || r.stdout != want || r.stderr != "" {
			t.Fatalf("completion %s: exit %d, stderr %q, stdout matches: %v", sh, r.code, r.stderr, r.stdout == want)
		}
	}
	r := runMain(t, nil, "completion", "--help")
	if !strings.Contains(r.stdout, "source <(gobble completion bash)") || !strings.Contains(r.stdout, "GOBBLE_COMPLETE") {
		t.Fatalf("completion --help lacks the install and completion-mode text:\n%s", r.stdout)
	}
}

// Completion mode answers through Main, before parsing: no usage error for
// the "--" words, nothing on stderr, nothing created.
func TestMainCompletionMode(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	t.Setenv(complete.EnvShell, "fish")
	t.Setenv(complete.EnvProtocol, complete.Protocol)
	var out, errw bytes.Buffer
	code := Main(t.Context(), []string{"--", "--log-level", ""}, nil, &out, &errw)
	if code != ExitOK || out.String() != "debug\ninfo\nwarn\nerror\n:36\n" || errw.Len() != 0 {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out.String(), errw.String())
	}
	if des, _ := os.ReadDir(home); len(des) != 0 {
		t.Fatalf("completion mode created %v", des)
	}
}

// Outside completion mode the variables are removed, so no child process
// inherits them; GOBBLE_COMPLETE=0 is not completion mode.
func TestCompletionEnvCleared(t *testing.T) {
	t.Setenv(complete.EnvShell, "0")
	t.Setenv(complete.EnvLine, "gobble x")
	r := runMain(t, nil, "version")
	if r.code != ExitOK || !strings.HasPrefix(r.stdout, "gobble ") {
		t.Fatalf("GOBBLE_COMPLETE=0 changed the run: exit %d, stdout %q", r.code, r.stdout)
	}
	for _, n := range complete.EnvNames {
		if v, ok := os.LookupEnv(n); ok {
			t.Fatalf("%s=%q survived Main", n, v)
		}
	}
}
