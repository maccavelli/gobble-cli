package acpserver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/permission"
)

// bashCall is a bash call of command.
func bashCall(t *testing.T, id, command string) llmtest.Turn {
	t.Helper()
	return llmtest.ToolCall(id, "bash", toolArgs(t, map[string]string{"command": command}))
}

// A deny on a command reaches it wherever it sits: in a pipeline, a list, a
// subshell, a substitution and an inline shell (0005-PLAN F6-3 decisions 2
// and 6).
func TestShellDenyReachesSubcommands(t *testing.T) {
	for _, cmd := range []string{"rm x && ls", "ls | rm x", "true && (rm x)", "echo $(rm x)", "bash -c 'cd y; rm x'", "sudo rm x", "X=1 rm x"} {
		s := llmtest.NewScript(bashCall(t, "b1", cmd), llmtest.Text("done"))
		client := &acptest.Client{Allow: true}
		ss := SessionSettings{Rules: []permission.Spec{rule(t, permission.ModeAllow, "bash", 3), rule(t, permission.ModeDeny, "bash(rm *)", 4)}}
		c := answersAgent(t, s, client, ss, PermissionOptions{}, nil)
		if _, err := prompt(t, c, openSession(t, c, t.TempDir()), "go"); err != nil {
			t.Fatal(err)
		}
		if got := lastResult(t, s, 1); !strings.Contains(got, `bash did not run: the permission rule "deny: bash(rm *)"`) || len(client.Permissions()) != 0 {
			t.Errorf("%q: %d questions, result %q", cmd, len(client.Permissions()), got)
		}
	}
}

// A plain rm runs as delete, through delete's limits: under .git it is
// refused whatever the rules say, and elsewhere it asks with the limit's
// mark (decision 9; F6-2 decision 5).
func TestShellInterceptLimits(t *testing.T) {
	cwd := t.TempDir()
	writeFile(t, filepath.Join(cwd, ".git", "config"), "x")
	writeFile(t, filepath.Join(cwd, "a.txt"), "x")
	ss := SessionSettings{Rules: []permission.Spec{rule(t, permission.ModeAllow, "*", 3)}}
	s := llmtest.NewScript(bashCall(t, "b1", "rm -r .git"), bashCall(t, "b2", "rm a.txt"), llmtest.Text("done"))
	client := &acptest.Client{Allow: true}
	c := answersAgent(t, s, client, ss, PermissionOptions{}, nil)
	if _, err := prompt(t, c, openSession(t, c, cwd), "go"); err != nil {
		t.Fatal(err)
	}
	if got := lastResult(t, s, 1); !strings.Contains(got, "delete did not run: gobble never changes files under .git") {
		t.Errorf("rm -r .git: %q", got)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".git", "config")); err != nil {
		t.Errorf(".git touched: %v", err)
	}
	perms := client.Permissions()
	if len(perms) != 1 || perms[0].ToolCall.ToolCallId != "b2" || perms[0].Meta[gobble].(map[string]any)[keyLimit] == nil {
		t.Fatalf("questions %+v, want rm a.txt's, as delete's limit", perms)
	}
	if got := lastResult(t, s, 2); !strings.HasPrefix(got, "bash ran this as gobble's delete tool.\n") {
		t.Errorf("rm a.txt: %q", got)
	}
	if _, err := os.Stat(filepath.Join(cwd, "a.txt")); err == nil {
		t.Error("a.txt is still there")
	}
}

// gobble update asks by the default rule, before every file's, and a later
// rule changes it (decision 10).
func TestGobbleUpdateDefault(t *testing.T) {
	ask := func(rules []permission.Spec) []acp.RequestPermissionRequest {
		t.Helper()
		s := llmtest.NewScript(bashCall(t, "b1", "gobble update --check"), llmtest.Text("done"))
		client := &acptest.Client{}
		c := answersAgent(t, s, client, SessionSettings{Rules: rules}, PermissionOptions{}, nil)
		if _, err := prompt(t, c, openSession(t, c, t.TempDir()), "go"); err != nil {
			t.Fatal(err)
		}
		return client.Permissions()
	}
	perms := ask(nil)
	if len(perms) != 1 {
		t.Fatalf("%d questions", len(perms))
	}
	if r, _ := perms[0].Meta[gobble].(map[string]any)[keyRule].(map[string]any); r[keyRule] != "ask: bash(gobble update*)" || r[keyFrom] != "default" {
		t.Errorf("rule %v", perms[0].Meta)
	}
	if perms := ask([]permission.Spec{rule(t, permission.ModeDeny, "bash(gobble *)", 3)}); len(perms) != 0 {
		t.Errorf("a later deny still asked: %d", len(perms))
	}
}

// A shell call's "always" saves each command's arity pattern, which the
// options name, and covers a later call with the same prefix (decision 4a).
func TestShellAlwaysArity(t *testing.T) {
	s := llmtest.NewScript(bashCall(t, "b1", "git status -s && make test"), bashCall(t, "b2", "git status"), llmtest.Text("done"))
	var names []string
	client := &acptest.Client{Answer: func(r acp.RequestPermissionRequest) acp.RequestPermissionResponse {
		for _, o := range r.Options {
			names = append(names, o.Name)
		}
		return choose(allowSession, "")(r)
	}}
	c := answersAgent(t, s, client, SessionSettings{}, PermissionOptions{}, nil)
	id := openSession(t, c, t.TempDir())
	if _, err := prompt(t, c, id, "go"); err != nil {
		t.Fatal(err)
	}
	if n := len(client.Permissions()); n != 1 {
		t.Fatalf("%d questions, want the first call's alone", n)
	}
	if len(names) != 8 || names[1] != "Allow `git status *`, `make test *` for this session" || names[7] != "Reject `git status *`, `make test *` everywhere" {
		t.Fatalf("option names %q", names)
	}
	list, _ := slash(t, c, client, id, "/permissions")
	if !strings.Contains(list, "3. allow: bash(git status *) (session)") || !strings.Contains(list, "4. allow: bash(make test *) (session)") {
		t.Fatalf("/permissions:\n%s", list)
	}
}

// A file operand outside the workspace asks as external_directory, though
// the rules allow the command (decision 7).
func TestShellOutsideOperand(t *testing.T) {
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "x.txt"), "42")
	s := llmtest.NewScript(bashCall(t, "b1", "cat "+filepath.ToSlash(filepath.Join(outside, "x.txt"))), llmtest.Text("done"))
	client := &acptest.Client{}
	ss := SessionSettings{Rules: []permission.Spec{rule(t, permission.ModeAllow, "bash(cat *)", 3)}}
	c := answersAgent(t, s, client, ss, PermissionOptions{}, nil)
	if _, err := prompt(t, c, openSession(t, c, t.TempDir()), "go"); err != nil {
		t.Fatal(err)
	}
	perms := client.Permissions()
	if len(perms) != 1 {
		t.Fatalf("%d questions", len(perms))
	}
	if r, _ := perms[0].Meta[gobble].(map[string]any)[keyRule].(map[string]any); r[keyRule] != "ask: external_directory" {
		t.Errorf("rule %v", perms[0].Meta)
	}
}
