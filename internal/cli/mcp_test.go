package cli

import (
	"bytes"
	"encoding/json/v2"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/internal/cli/complete"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
)

// fixture is the stdio MCP fixture as an mcp.json entry, with extra
// environment.
func fixture(t *testing.T, extra map[string]string) string {
	t.Helper()
	cmd, env := mcptest.StdioServer(t)
	maps.Copy(env, extra)
	b, err := json.Marshal(struct {
		Command string            `json:"command"`
		Env     map[string]string `json:"env"`
	}{cmd, env})
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// writeHomeMCP writes home's mcp.json: the members are "name": entry
// pairs, in order.
func writeHomeMCP(t *testing.T, home string, members ...string) string {
	t.Helper()
	var b strings.Builder
	b.WriteString(`{"mcpServers": {`)
	for i := 0; i < len(members); i += 2 {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`"` + members[i] + `": ` + members[i+1])
	}
	b.WriteString("}}")
	p := filepath.Join(home, "mcp.json")
	if err := os.WriteFile(p, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func httpEntry(url string) string { return `{"url": "` + url + `"}` }

// `gobble mcp list` connects to each server, prints its state and tools,
// and exits 1 while one has failed (0002-PLAN Phase 5 Accept).
func TestMCPListAgrees(t *testing.T) {
	home := t.TempDir()
	writeHomeMCP(t, home, "fx", fixture(t, nil), "web", httpEntry(mcptest.HTTPServer(t, "")))
	r := runIn(t, home, "mcp", "list")
	if r.code != ExitOK {
		t.Fatalf("exit %d\n%s%s", r.code, r.stdout, r.stderr)
	}
	for _, want := range []string{"fx: connected, 4 tools (direct)\n", "web: connected, 4 tools (direct)\n", "  tools: ", "echo"} {
		if !strings.Contains(r.stdout, want) {
			t.Fatalf("stdout lacks %q:\n%s", want, r.stdout)
		}
	}
	writeHomeMCP(t, home, "fx", fixture(t, nil), "bad", fixture(t, map[string]string{mcptest.EnvFail: "boom-on-start"}))
	r = runIn(t, home, "mcp", "list")
	if r.code != ExitFailure || !strings.Contains(r.stdout, "bad: failed (direct)\n") || !strings.Contains(r.stdout, "boom-on-start") {
		t.Fatalf("with a failed server: exit %d\n%s", r.code, r.stdout)
	}
	r = runIn(t, home, "mcp", "list", "--json")
	var rep listReport
	if err := json.Unmarshal([]byte(r.stdout), &rep); err != nil || r.code != ExitFailure || len(rep.Servers) != 2 || rep.Servers[1].State != "failed" {
		t.Fatalf("--json: exit %d, %v\n%s", r.code, err, r.stdout)
	}
}

// An SSE-only config is rejected: list exits 1 with Pi's message.
func TestMCPListRejectsSSE(t *testing.T) {
	home := t.TempDir()
	p := writeHomeMCP(t, home, "old", `{"type": "sse", "url": "https://example.com/sse"}`)
	r := runIn(t, home, "mcp", "list")
	want := "config error: " + p + `: server "old": legacy SSE transport is not supported; use the streamable HTTP URL` + "\n"
	if r.code != ExitFailure || !strings.Contains(r.stdout, want) {
		t.Fatalf("exit %d, stdout %q, want %q", r.code, r.stdout, want)
	}
}

func TestMCPListEmptyAndSignIn(t *testing.T) {
	home := t.TempDir()
	if r := runIn(t, home, "mcp", "list"); r.code != ExitOK || !strings.HasPrefix(r.stdout, "No MCP servers configured. Add them to ") {
		t.Fatalf("empty: exit %d, %q", r.code, r.stdout)
	}
	writeHomeMCP(t, home, "web", httpEntry(mcptest.HTTPServer(t, "sekrit")))
	r := runIn(t, home, "mcp", "list")
	if r.code != ExitFailure || !strings.Contains(r.stdout, "web: needs sign-in (direct)\n") ||
		!strings.Contains(r.stdout, "which arrives with 0005-PLAN F8 (gobble mcp login)") {
		t.Fatalf("sign-in: exit %d\n%s", r.code, r.stdout)
	}
}

const keptHome = `{
    "mcpServers": {
        "keep": {"command": "keep"}
    },
    "zzz": true
}
`

// add and remove edit the file, keeping the rest of it.
func TestMCPAddRemove(t *testing.T) {
	home := t.TempDir()
	p := filepath.Join(home, "mcp.json")
	if err := os.WriteFile(p, []byte(keptHome), 0o600); err != nil {
		t.Fatal(err)
	}
	r := runIn(t, home, "mcp", "add", "fs", "--env", "A=1,2", "--", "npx", "-y", "srv", ".")
	if r.code != ExitOK || !strings.HasPrefix(r.stdout, `Added MCP server "fs" in `) || !strings.Contains(r.stdout, "Check it with: gobble mcp list\n") {
		t.Fatalf("add: exit %d %q %q", r.code, r.stdout, r.stderr)
	}
	if r := runIn(t, home, "mcp", "add", "fs", "npx", "-y", "other"); r.code != ExitOK || !strings.HasPrefix(r.stdout, `Replaced MCP server "fs"`) {
		t.Fatalf("replace without --: exit %d %q %q", r.code, r.stdout, r.stderr)
	}
	if r := runIn(t, home, "mcp", "add", "docs", "--url", "https://example.com/mcp", "--bearer-token-env-var", "DOCS_TOKEN", "--exposure", "direct"); r.code != ExitOK {
		t.Fatalf("add http: exit %d %q", r.code, r.stderr)
	}
	if r := runIn(t, home, "mcp", "remove", "keep"); r.code != ExitOK || !strings.HasPrefix(r.stdout, `Removed MCP server "keep" from `) {
		t.Fatalf("remove: exit %d %q", r.code, r.stdout)
	}
	got, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
    "mcpServers": {
        "fs": {
            "command": "npx",
            "args": [
                "-y",
                "other"
            ]
        },
        "docs": {
            "url": "https://example.com/mcp",
            "headers": {
                "Authorization": "Bearer ${DOCS_TOKEN}"
            },
            "exposure": "direct"
        }
    },
    "zzz": true
}
`
	if string(got) != want {
		t.Fatalf("file\n%s\nwant\n%s", got, want)
	}
	if r := runIn(t, home, "mcp", "remove", "nope"); r.code != ExitFailure || !strings.Contains(r.stderr, `Error: no MCP server named "nope" in `) {
		t.Fatalf("remove unknown: exit %d %q", r.code, r.stderr)
	}
	for _, args := range [][]string{
		{"mcp", "add", "x"},
		{"mcp", "add", "x", "--url", "https://example.com", "--", "cmd"},
		{"mcp", "add", "x", "--cwd", "d", "--url", "https://example.com"},
		{"mcp", "add", "x", "--header", "A=1", "--", "cmd"},
		{"mcp", "add", "x", "--env", "novalue", "--", "cmd"},
		{"mcp", "add", "bad name", "--", "cmd"},
	} {
		if r := runIn(t, home, args...); r.code != ExitUsage {
			t.Errorf("%v: exit %d %q, want %d", args, r.code, r.stderr, ExitUsage)
		}
	}
}

// requestToolNames are the tool names a request declares.
func requestToolNames(t *testing.T, req *llm.Request) []string {
	t.Helper()
	var specs []struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(req.Tools, &specs); err != nil && len(req.Tools) > 0 {
		t.Fatal(err)
	}
	out := make([]string, len(specs))
	for i, s := range specs {
		out[i] = s.Name
	}
	return out
}

func hasMCP(names []string) bool {
	return slices.ContainsFunc(names, func(n string) bool { return strings.HasPrefix(n, mcpPrefix) })
}

// The in-process agent connects mcp.json's servers, and their tools run.
func TestChatUsesMCPTools(t *testing.T) {
	home := t.TempDir()
	writeHomeMCP(t, home, "fx", fixture(t, nil))
	s := llmtest.NewScript(llmtest.ToolCall("c1", "mcp__fx__echo", `{"text":"hi"}`), llmtest.Text("done"))
	withModel(t, s)
	mustRun(t, home, "--no-session", "-p", "go")
	reqs := s.Requests()
	if len(reqs) != 2 || !slices.Contains(requestToolNames(t, reqs[0]), "mcp__fx__echo") {
		t.Fatalf("%d requests; tools %q", len(reqs), requestToolNames(t, reqs[0]))
	}
	last := reqs[1].Messages[len(reqs[1].Messages)-1]
	if last.Content[0].Text != "hi" {
		t.Fatalf("tool result %q", last.Content[0].Text)
	}
}

// The tool flags follow Pi: -t is an allowlist over every tool,
// --exclude-tools removes names, --no-tools turns everything off, and no
// server connects when no MCP name is selectable.
func TestToolFlagsSelectMCPTools(t *testing.T) {
	home := t.TempDir()
	marker := filepath.Join(t.TempDir(), "started")
	writeHomeMCP(t, home, "fx", fixture(t, map[string]string{mcptest.EnvMarker: marker}))
	run := func(args ...string) []string {
		t.Helper()
		s := llmtest.NewScript(llmtest.Text("ok"))
		withModel(t, s)
		mustRun(t, home, append([]string{"--no-session", "-p"}, append(args, "go")...)...)
		return requestToolNames(t, s.Requests()[0])
	}
	if got := run("-t", "read,mcp__fx__echo"); !slices.Equal(got, []string{"read", "mcp__fx__echo"}) {
		t.Fatalf("-t: %q", got)
	}
	if got := run("--exclude-tools", "mcp__fx__add"); slices.Contains(got, "mcp__fx__add") || !slices.Contains(got, "mcp__fx__echo") {
		t.Fatalf("--exclude-tools: %q", got)
	}
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"--no-tools"}, {"-t", "read"}} {
		if got := run(args...); hasMCP(got) {
			t.Fatalf("%v: %q", args, got)
		}
		if _, err := os.Stat(marker); err == nil {
			t.Fatalf("%v started the MCP server", args)
		}
	}
	if r := runIn(t, home, "-p", "-t", "nope", "go"); r.code != ExitUsage {
		t.Fatalf("unknown tool: exit %d", r.code)
	}
}

// The MCP resource tools' names are tool names the flags accept: -t can
// list one, which keeps the servers on and offers only it of the three,
// and --exclude-tools can remove one (0005-PLAN F1d-2 deviation 3).
func TestToolFlagsSelectResourceTools(t *testing.T) {
	home := t.TempDir()
	writeHomeMCP(t, home, "docs", httpEntry(mcptest.ResourceHTTPServer(t)))
	run := func(args ...string) []string {
		t.Helper()
		s := llmtest.NewScript(llmtest.Text("ok"))
		withModel(t, s)
		mustRun(t, home, append([]string{"--no-session", "-p"}, append(args, "go")...)...)
		return requestToolNames(t, s.Requests()[0])
	}
	if got := run("-t", "read,list_mcp_resources"); !slices.Equal(got, []string{"read", "list_mcp_resources"}) {
		t.Fatalf("-t: %q", got)
	}
	if got := run("--exclude-tools", "read_mcp_resource"); slices.Contains(got, "read_mcp_resource") || !slices.Contains(got, "list_mcp_resources") {
		t.Fatalf("--exclude-tools: %q", got)
	}
	if got := run(); !slices.Contains(got, "read_mcp_resource") || slices.Contains(got, "tool_search") {
		t.Fatalf("no flags: %q", got)
	}
}

// `mcp remove <TAB>` offers the file's servers and creates nothing.
func TestMCPServerCompletion(t *testing.T) {
	home := t.TempDir()
	writeHomeMCP(t, home, "fx", fixture(t, nil), "web", httpEntry("https://example.com/mcp"))
	t.Setenv("GOBBLE_HOME", home)
	t.Setenv(complete.EnvShell, "fish")
	t.Setenv(complete.EnvProtocol, complete.Protocol)
	var out, errw bytes.Buffer
	if code := Main(t.Context(), []string{"--", "mcp", "remove", ""}, nil, &out, &errw); code != ExitOK || errw.Len() != 0 {
		t.Fatalf("exit %d, stderr %q", code, errw.String())
	}
	lines := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], "fx\t") || lines[1] != "web\thttps://example.com/mcp" || !strings.HasPrefix(lines[2], ":") {
		t.Fatalf("candidates %q", lines)
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 1 {
		t.Fatalf("completion created files: %v %v", entries, err)
	}
}
