package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/permission"
	"github.com/maccavelli/gobble-cli/session"
)

// Print mode rejects every question; with --yes it allows once what a rule
// or the default asks, and still refuses a limit's question and a deny
// rule (0005-PLAN F6-2 decisions 5 and 8).
func TestPrintYes(t *testing.T) {
	for _, tc := range []struct {
		name, settings string
		yes            bool
		call           string
		file           string
		written        bool
	}{
		{name: "without --yes a write is declined", call: `{"path":"a.txt","content":"x"}`, file: "a.txt"},
		{name: "--yes allows a write once", yes: true, call: `{"path":"a.txt","content":"x"}`, file: "a.txt", written: true},
		{name: "--yes leaves an AGENTS.md write, a limit's question", yes: true, call: `{"path":"AGENTS.md","content":"x"}`, file: "AGENTS.md"},
		{name: "--yes leaves a deny rule", yes: true, settings: "permissions:\n  rules:\n    - deny: write\n",
			call: `{"path":"a.txt","content":"x"}`, file: "a.txt"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home, dir := t.TempDir(), t.TempDir()
			t.Chdir(dir)
			if tc.settings != "" {
				writeFile(t, filepath.Join(home, "settings.yaml"), tc.settings)
			}
			withModel(t, llmtest.NewScript(llmtest.ToolCall("w1", "write", tc.call), llmtest.Text("done")))
			args := []string{"--no-session", "-p", "go"}
			if tc.yes {
				args = append([]string{"--yes"}, args...)
			}
			mustRun(t, home, args...)
			if _, err := os.Stat(filepath.Join(dir, tc.file)); (err == nil) != tc.written {
				t.Fatalf("%s written %v, want %v", tc.file, err == nil, tc.written)
			}
		})
	}
	// A delete is a limit's question whatever the rules say.
	home, dir := t.TempDir(), t.TempDir()
	t.Chdir(dir)
	writeFile(t, filepath.Join(dir, "f.txt"), "keep")
	writeFile(t, filepath.Join(home, "settings.yaml"), "permissions:\n  rules:\n    - allow: delete\n")
	withModel(t, llmtest.NewScript(llmtest.ToolCall("d1", "delete", `{"path":"f.txt"}`), llmtest.Text("done")))
	mustRun(t, home, "--yes", "--no-session", "-p", "go")
	if _, err := os.Stat(filepath.Join(dir, "f.txt")); err != nil {
		t.Fatalf("--yes approved a delete: %v", err)
	}
}

// --yes leaves a repeated call's question, so the turn halts at the third
// identical call (0005-PLAN F5-3 decision 1, F6-2a's deviation 2).
func TestPrintYesRepeated(t *testing.T) {
	home, dir := t.TempDir(), t.TempDir()
	t.Chdir(dir)
	writeFile(t, filepath.Join(dir, "f.txt"), "hello")
	read := func(id string) llmtest.Turn { return llmtest.ToolCall(id, "read", `{"path":"f.txt"}`) }
	s := llmtest.NewScript(read("a"), read("b"), read("c"), llmtest.Text("went on"))
	withModel(t, s)
	r := mustRun(t, home, "--yes", "--no-session", "-p", "go")
	if n := len(s.Requests()); n != 3 || strings.Contains(r.stdout, "went on") {
		t.Fatalf("%d requests, stdout %q; want the turn halted after the third call", n, r.stdout)
	}
}

// An untrusted project's rules are not applied; a trusted one's follow the
// global file's (0005-PLAN F6-1 decision 8, F6-2 decision 4).
func TestProjectRulesNeedTrust(t *testing.T) {
	home := t.TempDir()
	t.Setenv("GOBBLE_HOME", home)
	writeFile(t, filepath.Join(home, "settings.yaml"), "permissions:\n  rules:\n    - allow: bash(git *)\n")
	proj := trustProject(t)
	writeFile(t, filepath.Join(proj, ".gobble", "settings.yaml"), "permissions:\n  rules:\n    - deny: read(.env)\n")
	out := &Output{out: &bytes.Buffer{}, err: &bytes.Buffer{}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, store: session.NewMemoryStore()}
	t.Cleanup(func() {
		if err := e.logs.close(); err != nil {
			t.Error(err)
		}
	})
	layers := func() []string {
		var out []string
		for _, r := range e.agentOptions().Settings(proj).Rules {
			out = append(out, r.String()+" @"+r.Source.Layer)
		}
		return out
	}
	if got := layers(); len(got) != 1 || got[0] != "allow: bash(git *) @global" {
		t.Fatalf("untrusted: %q, want the global rule alone", got)
	}
	if _, err := (permission.TrustStore{Path: filepath.Join(home, permission.TrustFile)}).Save(t.Context(), proj, true); err != nil {
		t.Fatal(err)
	}
	if got := layers(); len(got) != 2 || got[1] != "deny: read(.env) @project" {
		t.Fatalf("trusted: %q, want the project's rule after the global one", got)
	}
}
