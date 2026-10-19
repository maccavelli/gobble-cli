package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// --mode takes the four modes; bypass needs permissions.allowBypass in the
// global file, and a project's file cannot give it; a mode the flag names
// is the sessions' start, so print mode writes in accept-edits without
// --yes (0005-PLAN F6-4 decisions 2 and 7).
func TestModeFlag(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()
	t.Chdir(dir)
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"--mode", "fast", "-p", "x"}, "--mode fast: the modes are default, accept-edits, plan, bypass"},
		{[]string{"--mode", "bypass", "-p", "x"}, "--mode bypass needs permissions.allowBypass: true in the global settings file"},
		{[]string{"acp", "--mode", "bypass"}, "--mode bypass needs permissions.allowBypass: true in the global settings file"},
	} {
		if r := runIn(t, home, c.args...); r.code != ExitUsage || !strings.Contains(r.stderr, c.want) {
			t.Errorf("%v: exit %d, stderr %q", c.args, r.code, r.stderr)
		}
	}
	// A project's allowBypass is not applied, trusted or not.
	proj := trustProject(t)
	writeFile(t, filepath.Join(proj, ".gobble", "settings.yaml"), "permissions:\n  allowBypass: true\n")
	t.Chdir(proj)
	mustRun(t, home, "trust")
	if r := runIn(t, home, "--mode", "bypass", "-p", "x"); r.code != ExitUsage || !strings.Contains(r.stderr, "needs permissions.allowBypass") {
		t.Errorf("a project's allowBypass: exit %d, stderr %q", r.code, r.stderr)
	}
	writeFile(t, filepath.Join(home, "settings.yaml"), "permissions:\n  allowBypass: true\n")
	withModel(t, llmtest.NewScript(llmtest.Text("ok")))
	mustRun(t, home, "--no-session", "--mode", "bypass", "-p", "x")

	t.Chdir(dir)
	for _, mode := range []string{"default", "accept-edits"} {
		withModel(t, llmtest.NewScript(llmtest.ToolCall("w1", "write", `{"path":"a.txt","content":"x"}`), llmtest.Text("done")))
		mustRun(t, home, "--no-session", "--mode", mode, "-p", "go")
		_, err := os.Stat(filepath.Join(dir, "a.txt"))
		if (err == nil) != (mode == "accept-edits") {
			t.Errorf("--mode %s: a.txt written %v", mode, err == nil)
		}
	}
}
