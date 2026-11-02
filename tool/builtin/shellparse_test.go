package builtin

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/permission"
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
		{"quoted words, and their normalised form", `"git" 'status'`, []string{`"git" 'status'`, "git status"},
			[]string{"git status *", "git status *"}},
		{"a redirection kept, and the words without it", "echo hi > out.txt", []string{"echo hi > out.txt", "echo hi"},
			[]string{"echo *", "echo *"}},
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
// Each raw subject is followed by its normalised form: the words unquoted,
// with no redirection (0037-MADR D4).
func TestParseShellMasking(t *testing.T) {
	for in, want := range map[string][2]string{
		`grep "a|b" f 2>/dev/null`:        {`grep "a_b" f 2_/dev/null`, `grep a|b f`},
		`cat f 2>&1`:                      {`cat f 2__1`, `cat f`},
		`grep a\|b f`:                     {`grep a\_b f`, `grep a|b f`},
		`echo test > output.txt`:          {`echo test > output.txt`, `echo test`},
		`cat $'a\'b' > out.txt`:           {`cat _'a\'b' > out.txt`, `cat $'a\'b'`},
		"cat <<'EOF'\na|b\nEOF":           {"cat <<'EOF'_a_b_EOF", "cat"},
		`echo 'x;y' "$HOME" >&-`:          {`echo 'x_y' "$HOME" __-`, `echo x;y "$HOME"`},
		`printf '%s\n' "a&&b" </dev/null`: {`printf '%s\n' "a__b" _/dev/null`, `printf %s\n a&&b`},
	} {
		if got := parseShell(in).subjects; !slices.Equal(got, want[:]) {
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
		{"bash -c 'rm x && ls'", []string{"bash -c 'rm x __ ls'", "bash -c rm x && ls", "rm x", "ls"}},
		{`sh -ec "git push"`, []string{`sh -ec "git push"`, "sh -ec git push", "git push"}},
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
	// Each bash -c is a subject twice, raw and normalised (0037-MADR D4).
	if got := parseShell(nest(8)).subjects; len(got) != 17 || got[16] != "rm x" {
		t.Errorf("depth 8: %d subjects, last %q", len(got), got[len(got)-1])
	}
	if got := parseShell(nest(9)).subjects; len(got) != 1 || got[0] != nest(9) {
		t.Errorf("depth 9: %q, want it raw", got)
	}
}

// Under allow: bash then deny: bash(rm *), quoting, an escape, a tab, a
// path and, where the volume finds a program whatever its case, case no
// longer get round the deny: the command is matched in its raw and its
// normalised form (0037-MADR D4, F3's table). An "always" saves only the
// raw subjects' patterns (0037-PLAN deviation 3).
func TestCommandBypasses(t *testing.T) {
	rule := func(effect permission.Mode, text string) permission.Spec {
		s, err := permission.ParseSpec(effect, text)
		if err != nil {
			t.Fatal(err)
		}
		return s
	}
	e := permission.Engine{Layers: [][]permission.Spec{{rule(permission.ModeAllow, "bash"), rule(permission.ModeDeny, "bash(rm *)")}}}
	ruleOf := func(cmd string) permission.Rule {
		sc := parseShell(cmd)
		return permission.Rule{Tool: "bash", Kind: permission.SubjectCommand, Subjects: sc.subjects, Saves: sc.saves}
	}
	denied := []string{
		`\rm -rf build`, `"rm" -rf build`, `r''m -rf build`, "rm\t-rf build", "/bin/rm -rf build",
		// F3's rows denied before: they stay denied.
		"rm -rf build", "a && rm -rf build", "echo $(rm -rf build)", "command rm -rf build", "env rm -rf build",
		"sudo rm -rf build", "FOO=1 rm -rf build",
	}
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		denied = append(denied, "RM -rf build") // the volume finds RM as rm
	}
	for _, cmd := range denied {
		if v := e.Evaluate(ruleOf(cmd)); v.Effect != permission.ModeDeny {
			t.Errorf("%q: %+v, want denied", cmd, v)
		}
	}
	// Beyond pattern rules, as docs/architecture.md says: the program xargs
	// or find runs is not a word of the command.
	for _, cmd := range []string{"echo build | xargs rm -rf", "find build -delete"} {
		if v := e.Evaluate(ruleOf(cmd)); v.Effect != permission.ModeAllow {
			t.Errorf("%q: %+v, want allowed: it is beyond a pattern rule", cmd, v)
		}
	}
	// An "always" saves each raw subject's pattern once, and nothing for a
	// normalised form.
	for cmd, want := range map[string][]string{
		`\rm -rf build`:     {"rm *"},
		`"rm" -rf build`:    {"rm *"},
		"/bin/rm -rf build": {"/bin/rm *"},
		"echo hi > out.txt": {"echo *"},
		`"git" 'status'`:    {"git status *"},
	} {
		saved, ok := e.Remember(permission.ModeAllow, ruleOf(cmd))
		var got []string
		for _, s := range saved {
			got = append(got, s.Pattern)
		}
		if !ok || !slices.Equal(got, want) {
			t.Errorf("%q: always saves %q (%v), want %q", cmd, got, ok, want)
		}
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

// A command's writes are its output redirections' literal files, its
// writing programs' literal operands, and an in-place editor's files, its
// script left out; a sink, a read, an expansion and a command
// substitution are none. A writing program's every operand counts, such as
// chmod's mode, which names no protected path (0037-MADR D3).
func TestShellWrites(t *testing.T) {
	for _, c := range []struct {
		cmd    string
		writes []string
	}{
		{"echo x > out", []string{"out"}},
		{"echo x >> out", []string{"out"}},
		{"echo x >| out", []string{"out"}},
		{"echo x &> out", []string{"out"}},
		{"echo x &>> out", []string{"out"}},
		{"cat <> out", []string{"out"}},
		{`echo x > "a b"`, []string{"a b"}},
		{"echo x > /dev/null 2>/dev/null", nil},
		{"sort < in.txt", nil},
		{"tee a 'b c'", []string{"a", "b c"}},
		{"echo a | tee -a .git/config", []string{".git/config"}},
		{"truncate -s0 f", []string{"f"}},
		{"rm -rf x y", []string{"x", "y"}},
		{"rmdir d", []string{"d"}},
		{"mv a b", []string{"a", "b"}},
		{"cp -r a b", []string{"a", "b"}},
		{"mkdir -p d", []string{"d"}},
		{"touch f", []string{"f"}},
		{"ln -s a b", []string{"a", "b"}},
		{"chmod 600 f", []string{"600", "f"}},
		{"chown u f", []string{"u", "f"}},
		{"sed -i s/a/b/ f", []string{"f"}},
		{"sed -i.bak -e s/a/b/ f g", []string{"f", "g"}},
		{"sed --in-place s/a/b/ f", []string{"f"}},
		{"sed -i -f script.sed f", []string{"f"}},
		{"sed s/a/b/ f", nil},
		{"perl -i -pe 's/a/b/' f", []string{"f"}},
		{"ruby -i -pe '$_.upcase!' f", []string{"f"}},
		{"perl -pe 's/a/b/' f", nil},
		{"cat f", nil},
		{"echo x > $F", nil},
		{"tee $F", nil},
		{"echo x > $(mktemp)", nil},
		{"bash -c 'echo x > y'", []string{"y"}},
	} {
		if got := parseShell(c.cmd).writes; !slices.Equal(got, c.writes) {
			t.Errorf("%q: %q, want %q", c.cmd, got, c.writes)
		}
	}

	// bashSubject resolves them against the workdir, ~ at the home
	// directory, each once.
	root, other := t.TempDir(), t.TempDir()
	env := tool.Env{Cwd: root, Roots: []string{root}}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home directory")
	}
	subject := func(cmd, workdir string) []string {
		args := `{"command":` + quote(t, cmd)
		if workdir != "" {
			args += `,"workdir":` + quote(t, workdir)
		}
		return bashSubject(tool.Call{Name: "bash", Args: []byte(args + "}")}, env).Writes
	}
	if got := subject("echo x > a.txt; tee a.txt", ""); !slices.Equal(got, []string{filepath.Join(root, "a.txt")}) {
		t.Errorf("against the cwd, once: %q", got)
	}
	if got := subject("echo x > a.txt", other); !slices.Equal(got, []string{filepath.Join(other, "a.txt")}) {
		t.Errorf("against the workdir: %q", got)
	}
	if got := subject("echo x >> ~/.ssh/authorized_keys", ""); !slices.Equal(got, []string{filepath.Join(home, ".ssh", "authorized_keys")}) {
		t.Errorf("~: %q", got)
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
