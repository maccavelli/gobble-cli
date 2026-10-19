package builtin

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/tool"
)

// Each simple command at any depth is a subject, with leading assignments
// left out, and its "always" pattern from the arity table; a command no
// rule can split is one raw subject (0005-PLAN F6-3 decisions 2 to 4).
func TestParseShellSubjects(t *testing.T) {
	for _, c := range []struct {
		name, cmd string
		subjects  []string
		saves     []string
	}{
		{"one command", "git status -s", []string{"git status -s"}, []string{"git status *"}},
		{"a list", "make test && git push || echo no; ls", []string{"make test", "git push", "echo no", "ls"},
			[]string{"make test *", "git push *", "echo *", "ls *"}},
		{"a pipeline", "cat f | grep x | sort", []string{"cat f", "grep x", "sort"}, []string{"cat *", "grep *", "sort *"}},
		{"a subshell and a group", "(cd /tmp && rm a); { rm b; }", []string{"rm a", "rm b"}, []string{"rm *", "rm *"}},
		{"a command substitution", "echo $(rm -rf x)", []string{"echo $(rm -rf x)", "rm -rf x"}, []string{"echo *", "rm *"}},
		{"a process substitution", "diff <(ls a) b", []string{"diff <(ls a) b", "ls a"}, []string{"diff *", "ls *"}},
		{"a function, a loop, a condition", "f() { rm x; }; for i in 1 2; do touch $i; done; if true; then git stash pop; fi",
			[]string{"rm x", "touch $i", "true", "git stash pop"}, []string{"rm *", "touch *", "true *", "git stash pop *"}},
		{"assignments left out", "X=1 Y=2 rm -rf build", []string{"rm -rf build"}, []string{"rm *"}},
		{"an assignment that changes what runs", "PATH=/tmp make", []string{"make", "PATH=/tmp make"}, []string{"make *", ""}},
		{"arity three", "docker compose up -d", []string{"docker compose up -d"}, []string{"docker compose up *"}},
		{"arity for a short command", "git", []string{"git"}, []string{"git *"}},
		{"a non-literal word in the prefix", "git $SUB x", []string{"git $SUB x"}, []string{""}},
		{"a quoted expansion in the prefix", `git "$SUB" x`, []string{`git "$SUB" x`}, []string{""}},
		{"quoted words", `"git" 'status'`, []string{`"git" 'status'`}, []string{"git status *"}},
		{"a redirection kept", "echo hi > out.txt", []string{"echo hi > out.txt"}, []string{"echo *"}},
	} {
		sc := parseShell(c.cmd)
		if !slices.Equal(sc.subjects, c.subjects) || !slices.Equal(sc.saves, c.saves) {
			t.Errorf("%s: %q, %q; want %q, %q", c.name, sc.subjects, sc.saves, c.subjects, c.saves)
		}
	}
}

// What the parser cannot prove becomes one raw subject, the whole command,
// saved exactly (decision 3).
func TestParseShellRaw(t *testing.T) {
	for _, cmd := range []string{
		`echo "unclosed`,
		"FOO=1",
		"(( n++ ))",
		"$CMD x",
		"$(which rm) x",
		"./run-*.sh",
		"cd $DIR && ls",
		`bash -c "$SCRIPT"`,
		"bash -c 'echo \"unclosed'",
		"sudo $(which rm) x",
	} {
		sc := parseShell(cmd)
		if !slices.Equal(sc.subjects, []string{strings.TrimSpace(cmd)}) || !slices.Equal(sc.saves, []string{""}) {
			t.Errorf("%q: %q, %q; want it raw", cmd, sc.subjects, sc.saves)
		}
	}
}

// Kilo's masking: an operator character is _ only where it is inert, and
// every other byte is as written (decision 5;
// packages/opencode/test/kilocode/tool/shell-pattern.test.ts:173-187).
func TestParseShellMasking(t *testing.T) {
	for in, want := range map[string]string{
		`grep "a|b" f 2>/dev/null`:        `grep "a_b" f 2_/dev/null`,
		`cat f 2>&1`:                      `cat f 2__1`,
		`grep a\|b f`:                     `grep a\_b f`,
		`echo test > output.txt`:          `echo test > output.txt`,
		`cat $'a\'b' > out.txt`:           `cat _'a\'b' > out.txt`,
		"cat <<'EOF'\na|b\nEOF":           "cat <<'EOF'_a_b_EOF",
		`echo 'x;y' "$HOME" >&-`:          `echo 'x_y' "$HOME" __-`,
		`printf '%s\n' "a&&b" </dev/null`: `printf '%s\n' "a__b" _/dev/null`,
	} {
		if got := parseShell(in).subjects; len(got) != 1 || got[0] != want {
			t.Errorf("%q: %q, want %q", in, got, want)
		}
	}
}

// A wrapper's command is a subject as well as the wrapper, when its options
// can be told apart; bash -c's literal script is parsed in turn, to a
// depth of 8 (decision 6).
func TestParseShellWrappers(t *testing.T) {
	for _, c := range []struct {
		cmd      string
		subjects []string
	}{
		{"sudo rm -rf x", []string{"sudo rm -rf x", "rm -rf x"}},
		{"env -i FOO=1 make all", []string{"env -i FOO=1 make all", "make all"}},
		{"nice -n 5 timeout 10 go test", []string{"nice -n 5 timeout 10 go test", "timeout 10 go test", "go test"}},
		// time is bash's keyword, not a program: its command is the subject.
		{"time -p command -- ls", []string{"command -- ls", "ls"}},
		{"sudo -u bob rm x", []string{"sudo -u bob rm x"}},
		{"command -v rm", []string{"command -v rm"}},
		{"bash -c 'rm x && ls'", []string{"bash -c 'rm x __ ls'", "rm x", "ls"}},
		{`sh -ec "git push"`, []string{`sh -ec "git push"`, "git push"}},
		{"bash script.sh", []string{"bash script.sh"}},
	} {
		if got := parseShell(c.cmd).subjects; !slices.Equal(got, c.subjects) {
			t.Errorf("%q: %q, want %q", c.cmd, got, c.subjects)
		}
	}
	nest := func(n int) string {
		s := "rm x"
		for range n {
			s = "bash -c " + shellQuote(s)
		}
		return s
	}
	if got := parseShell(nest(8)).subjects; len(got) != 9 || got[8] != "rm x" {
		t.Errorf("depth 8: %d subjects, last %q", len(got), got[len(got)-1])
	}
	if got := parseShell(nest(9)).subjects; len(got) != 1 || got[0] != nest(9) {
		t.Errorf("depth 9: %q, want it raw", got)
	}
}

// shellQuote is s in single quotes, as bash reads it.
func shellQuote(s string) string { return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'" }

// cd's directories, a file program's operands and a redirection's file are
// the command's paths; a sink, an option and a non-file program's operands
// are not (decision 7).
func TestParseShellPaths(t *testing.T) {
	for _, c := range []struct {
		cmd   string
		paths []string
	}{
		{"cd /etc && ls", []string{"/etc"}},
		{"pushd ../x", []string{"../x"}},
		{"cat -n a.txt -- -b", []string{"a.txt", "-b"}},
		{`cat a\ b.txt "c d"`, []string{"a b.txt", "c d"}},
		{"rm -rf /tmp/x y", []string{"/tmp/x", "y"}},
		{"echo hi > /tmp/out 2>/dev/null", []string{"/tmp/out"}},
		{"sort < in.txt >> /var/log/x", []string{"in.txt", "/var/log/x"}},
		{"echo /etc/passwd", nil},
		{"git add /tmp/x", nil},
		{"cat $F", nil},
	} {
		if got := parseShell(c.cmd).paths; !slices.Equal(got, c.paths) {
			t.Errorf("%q: %q, want %q", c.cmd, got, c.paths)
		}
	}
}

// bash's outside paths are its workdir and its paths, resolved against
// the workdir, ~ at the home directory (decision 7).
func TestBashOutside(t *testing.T) {
	root, other := t.TempDir(), t.TempDir()
	env := tool.Env{Cwd: root, Roots: []string{root}}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	call := func(cmd, workdir string) []string {
		args := `{"command":` + quote(t, cmd)
		if workdir != "" {
			args += `,"workdir":` + quote(t, workdir)
		}
		return bashOutside(tool.Call{Name: "bash", Args: []byte(args + "}")}, env)
	}
	if got := call("cat a.txt sub/b.txt > out", ""); len(got) != 0 {
		t.Errorf("inside: %q", got)
	}
	if got := call("cd "+shellQuote(other)+" && cat y", ""); !slices.Equal(got, []string{other}) {
		t.Errorf("cd outside: %q", got)
	}
	if got := call("cat ../x", ""); !slices.Equal(got, []string{filepath.Join(filepath.Dir(root), "x")}) {
		t.Errorf("a relative operand: %q", got)
	}
	if got := call("cat x", other); !slices.Equal(got, []string{other, filepath.Join(other, "x")}) {
		t.Errorf("against the workdir: %q", got)
	}
	if got := call("cat ~/.ssh/id", ""); !slices.Equal(got, []string{filepath.Join(home, ".ssh", "id")}) {
		t.Errorf("~: %q", got)
	}
}
