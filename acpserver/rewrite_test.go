package acpserver

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// fakeMCPOnPath puts an executable gobble-fake-mcp, the only thing on PATH,
// in a temp directory, and returns its absolute path.
func fakeMCPOnPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	name := "gobble-fake-mcp"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil { //nolint:gosec // G306: it must be executable
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	return path
}

func stdioServer(command string) acp.McpServer {
	return acp.McpServer{Stdio: &acp.McpServerStdio{Name: "fx", Command: command, Args: []string{}, Env: []acp.EnvVariable{}}}
}

// A bare stdio command that resolves on PATH becomes its absolute path; one
// that does not, and one already absolute, are left as they are (0019-MADR
// D6).
func TestResolveBareCommands(t *testing.T) {
	fake := fakeMCPOnPath(t)
	abs := acptest.Abs("/bin/other")
	req := &acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{
		stdioServer("gobble-fake-mcp"), stdioServer("no-such-command-0019"), stdioServer(abs),
	}}
	if err := resolveBareCommands(t.Context(), acp.AgentMethodSessionNew, req); err != nil {
		t.Fatal(err)
	}
	got := []string{req.McpServers[0].Stdio.Command, req.McpServers[1].Stdio.Command, req.McpServers[2].Stdio.Command}
	if want := []string{fake, "no-such-command-0019", abs}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("commands %q, want %q", got, want)
	}
	for _, p := range []any{
		&acp.LoadSessionRequest{McpServers: []acp.McpServer{stdioServer("gobble-fake-mcp")}},
		&acp.ResumeSessionRequest{McpServers: []acp.McpServer{stdioServer("gobble-fake-mcp")}},
	} {
		if err := resolveBareCommands(t.Context(), "", p); err != nil {
			t.Fatal(err)
		}
		var servers []acp.McpServer
		switch p := p.(type) {
		case *acp.LoadSessionRequest:
			servers = p.McpServers
		case *acp.ResumeSessionRequest:
			servers = p.McpServers
		}
		if servers[0].Stdio.Command != fake {
			t.Errorf("%T: command %q, want %q", p, servers[0].Stdio.Command, fake)
		}
	}
}

// Under GODEBUG=execerrdot=0, exec.LookPath returns a command found through
// a relative PATH entry with no error; it is still not absolute, so it is
// left for the SDK to refuse (0019-PLAN P4 deviation 8).
func TestResolveBareCommandsKeepsRelative(t *testing.T) {
	dir := t.TempDir()
	name := "gobble-rel-mcp"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"), 0o755); err != nil { //nolint:gosec // G306: it must be executable
		t.Fatal(err)
	}
	t.Chdir(dir)
	t.Setenv("PATH", ".")
	t.Setenv("GODEBUG", "execerrdot=0")
	if path, err := exec.LookPath("gobble-rel-mcp"); err != nil || filepath.IsAbs(path) {
		t.Fatalf("LookPath = %q, %v; want a relative path and no error, or this case tests nothing", path, err)
	}
	req := &acp.NewSessionRequest{McpServers: []acp.McpServer{stdioServer("gobble-rel-mcp")}}
	if err := resolveBareCommands(t.Context(), acp.AgentMethodSessionNew, req); err != nil {
		t.Fatal(err)
	}
	if got := req.McpServers[0].Stdio.Command; got != "gobble-rel-mcp" {
		t.Fatalf("command %q, want it left as gobble-rel-mcp", got)
	}
}

// Through Serve, session/new with a bare command that resolves is answered
// with a session; with one that does not, the SDK refuses it, -32602
// (0019-MADR D6).
func TestServeResolvesBareCommand(t *testing.T) {
	fakeMCPOnPath(t)
	call, stop := rawServe(t, Options{Version: "1.2.3", NewID: counterIDs(),
		Provider: func(string) (llm.Provider, error) { return llmtest.NewScript(), nil }, Models: func() ModelChoice { return testModels }})
	defer stop()
	if line := call(1, "initialize", `{"protocolVersion":1,"clientCapabilities":{}}`); strings.Contains(line, `"error"`) {
		t.Fatalf("initialize: %s", line)
	}
	params := func(command string) string {
		return `{"cwd":` + jsonString(t.TempDir()) + `,"mcpServers":[{"name":"fx","command":` + jsonString(command) + `,"args":[],"env":[]}]}`
	}
	if line := call(2, "session/new", params("gobble-fake-mcp")); !strings.Contains(line, `"sessionId":"s1"`) {
		t.Fatalf("bare, resolvable: %s, want a session", line)
	}
	if line := call(3, "session/new", params("no-such-command-0019")); !strings.Contains(line, `"code":-32602`) || !strings.Contains(line, "is not an absolute path") {
		t.Fatalf("bare, unresolvable: %s, want -32602 naming the path", line)
	}
}
