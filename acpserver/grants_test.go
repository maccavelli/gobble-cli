package acpserver

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/permission"
)

// grantCall is a request_permissions call.
func grantCall(t *testing.T, id string, read, write []string, scope string) llmtest.Turn {
	t.Helper()
	perms := map[string]any{}
	if read != nil {
		perms["read"] = read
	}
	if write != nil {
		perms["write"] = write
	}
	args := map[string]any{"permissions": perms, "reason": "to fix the build"}
	if scope != "" {
		args["scope"] = scope
	}
	return llmtest.ToolCall(id, requestPermissionsTool, toolArgs(t, args))
}

// findGrant is the tool_search call that loads request_permissions, which
// is deferred (0012-MADR D9).
func findGrant() llmtest.Turn {
	return llmtest.ToolCall("s0", "tool_search", `{"query":"request_permissions"}`)
}

// granting answers request_permissions with grant, and every other
// question with reject.
func granting(r acp.RequestPermissionRequest) acp.RequestPermissionResponse {
	for _, o := range r.Options {
		if o.OptionId == grantYes {
			return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(grantYes)}
		}
	}
	return acp.RequestPermissionResponse{Outcome: acp.NewRequestPermissionOutcomeSelected(rejectOnce)}
}

// The spec is 0012-MADR's row: deferred in the permissions family,
// permissions with at least one property, a reason, and a scope of turn or
// session, turn by default (0005-PLAN F6-5 decision 1).
func TestRequestPermissionsSpec(t *testing.T) {
	s := requestPermissionsSpec
	params := string(s.InputSchema)
	for _, want := range []string{`"minProperties":1`, `"minLength":1`, `"enum":["turn","session"]`, `"default":"turn"`, `"read"`, `"write"`, `"network"`} {
		if !strings.Contains(params, want) {
			t.Errorf("schema lacks %s: %s", want, params)
		}
	}
	if s.Exposure != "deferred" || s.Family != "permissions" || !s.Annotations.ReadOnlyHint {
		t.Errorf("spec %+v", s)
	}
}

// A grant answers the questions about calls under its paths, for the turn
// or the session; a call outside it, a deny rule and gobble's limits still
// decide (decisions 2a and 4).
func TestGrants(t *testing.T) {
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "x.txt"), "42")
	cwd := t.TempDir()
	writeFile(t, filepath.Join(cwd, "src", "a.txt"), "a")
	s := llmtest.NewScript(findGrant(),
		grantCall(t, "g1", []string{outside}, []string{"src"}, ""),
		llmtest.ToolCall("r1", "read", toolArgs(t, map[string]string{"path": filepath.Join(outside, "x.txt")})),
		llmtest.ToolCall("w1", "write", `{"path":"src/b.txt","content":"b"}`),
		llmtest.ToolCall("w2", "write", `{"path":"other.txt","content":"c"}`),
		llmtest.ToolCall("w3", "write", `{"path":"src/AGENTS.md","content":"c"}`),
		llmtest.ToolCall("w4", "write", `{"path":"src/deny.txt","content":"c"}`),
		llmtest.ToolCall("w6", "write", toolArgs(t, map[string]string{"path": filepath.Join(outside, "y.txt"), "content": "c"})),
		llmtest.Text("done"),
		llmtest.ToolCall("w5", "write", `{"path":"src/c.txt","content":"c"}`),
		llmtest.Text("again"))
	client := &acptest.Client{Answer: granting}
	ss := SessionSettings{Rules: []permission.Spec{rule(t, permission.ModeDeny, "write(/src/deny.txt)", 3)}}
	c, id, logs := modeAgent(t, s, client, ss, "", "", cwd)
	if _, err := prompt(t, c, id, "go"); err != nil {
		t.Fatal(err)
	}
	var asked []string
	for _, p := range client.Permissions() {
		asked = append(asked, string(p.ToolCall.ToolCallId))
	}
	// g1 is the grant's question; w2 is outside it; w3 is AGENTS.md, a limit;
	// w6 writes where the grant reads only.
	if !slices.Equal(asked, []string{"g1", "w2", "w3", "w6"}) {
		t.Fatalf("questions about %q, want g1, w2, w3, w6", asked)
	}
	if got := lastResult(t, s, 2); !strings.Contains(got, "The user granted read access to "+outside+" and write access to "+filepath.Join(cwd, "src")+" for this turn") {
		t.Errorf("the grant's result %q", got)
	}
	if got := lastResult(t, s, 3); !strings.Contains(got, "42") {
		t.Errorf("the outside read %q", got)
	}
	if _, err := os.Stat(filepath.Join(cwd, "src", "b.txt")); err != nil {
		t.Errorf("the granted write did not run: %v", err)
	}
	if got := lastResult(t, s, 7); !strings.Contains(got, `the permission rule "deny: write(/src/deny.txt)"`) {
		t.Errorf("the deny under a grant: %q", got)
	}
	if !strings.Contains(logs.String(), `from="grant turn"`) {
		t.Errorf("log %q lacks the grant", logs.String())
	}
	// The turn's grant ended with it.
	n := len(client.Permissions())
	if _, err := prompt(t, c, id, "more"); err != nil {
		t.Fatal(err)
	}
	if len(client.Permissions()) != n+1 {
		t.Errorf("the next turn's write under the turn's grant was not asked")
	}
}

// A session's grant lasts past the turn; /permissions lists it after the
// rules and removes it (decisions 4 and 8).
func TestGrantSession(t *testing.T) {
	cwd := t.TempDir()
	s := llmtest.NewScript(findGrant(), grantCall(t, "g1", nil, []string{"src"}, "session"), llmtest.Text("ok"),
		llmtest.ToolCall("w1", "write", `{"path":"src/b.txt","content":"b"}`), llmtest.Text("done"),
		llmtest.ToolCall("w2", "write", `{"path":"src/c.txt","content":"c"}`), llmtest.Text("done"))
	client := &acptest.Client{Answer: granting}
	c, id, _ := modeAgent(t, s, client, SessionSettings{}, "", "", cwd)
	for _, p := range []string{"grant", "write"} {
		if _, err := prompt(t, c, id, p); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(client.Permissions()); n != 1 {
		t.Fatalf("%d questions; the session's grant should have answered the second turn's write", n)
	}
	list, _ := slash(t, c, client, id, "/permissions")
	want := "3. grant write: " + filepath.Join(cwd, "src") + " (session)"
	if !strings.Contains(list, want) {
		t.Fatalf("/permissions:\n%s\nwant %q", list, want)
	}
	if got, _ := slash(t, c, client, id, "/permissions remove 3"); !strings.Contains(got, "Removed grant write: ") {
		t.Fatalf("remove: %q", got)
	}
	if _, err := prompt(t, c, id, "again"); err != nil {
		t.Fatal(err)
	}
	if n := len(client.Permissions()); n != 2 {
		t.Errorf("%d questions; the removed grant still answered", n)
	}
}

// What the tool refuses without a question: no path and no network, no
// reason, a scope that is neither; network alone is answered; a write
// grant in plan mode is refused; the user's refusal and its reason
// (decisions 1, 3a, 5 and 7).
func TestRequestPermissionsAnswers(t *testing.T) {
	run := func(t *testing.T, call llmtest.Turn, mode string, answer func(acp.RequestPermissionRequest) acp.RequestPermissionResponse) (string, int) {
		t.Helper()
		s := llmtest.NewScript(findGrant(), call, llmtest.Text("done"))
		client := &acptest.Client{Answer: answer}
		c, id, _ := modeAgent(t, s, client, SessionSettings{}, "", mode, t.TempDir())
		if _, err := prompt(t, c, id, "go"); err != nil {
			t.Fatal(err)
		}
		return lastResult(t, s, 2), len(client.Permissions())
	}
	call := func(perms map[string]any, reason, scope string) llmtest.Turn {
		args := map[string]any{"permissions": perms, "reason": reason}
		if scope != "" {
			args["scope"] = scope
		}
		return llmtest.ToolCall("g", requestPermissionsTool, toolArgs(t, args))
	}
	for _, c := range []struct {
		name  string
		call  llmtest.Turn
		mode  string
		want  string
		asked int
	}{
		{"no permission", call(map[string]any{}, "x", ""), "", "needs at least one path", 0},
		{"no reason", call(map[string]any{"read": []string{"a"}}, " ", ""), "", "needs a reason", 0},
		{"a bad scope", call(map[string]any{"read": []string{"a"}}, "x", "forever"), "", "scope is turn or session", 0},
		{"network alone", call(map[string]any{"network": true}, "x", ""), "", textNetwork, 0},
		{"network with a path", call(map[string]any{"network": true, "read": []string{"a"}}, "x", ""), "", "The user granted read access to", 1},
		{"a write grant in plan mode", call(map[string]any{"write": []string{"a"}}, "x", ""), modePlan, "Plan mode is read-only, so write access cannot be granted", 0},
		{"a read grant in plan mode", call(map[string]any{"read": []string{"a"}}, "x", ""), modePlan, "The user granted read access", 1},
	} {
		got, asked := run(t, c.call, c.mode, granting)
		if !strings.Contains(got, c.want) || asked != c.asked {
			t.Errorf("%s: %q, %d questions; want %q, %d", c.name, got, asked, c.want, c.asked)
		}
	}
	got, _ := run(t, call(map[string]any{"read": []string{"a"}}, "x", ""), "", choose(grantNo, "not now"))
	if !strings.Contains(got, "The user refused the request. The user said: not now") {
		t.Errorf("refused: %q", got)
	}
	got, _ = run(t, call(map[string]any{"read": []string{"a"}, "network": true}, "x", ""), "", granting)
	if !strings.Contains(got, textNetwork) {
		t.Errorf("network with a path: %q lacks the network note", got)
	}
}

// A declined call a grant could cover loads the permissions family once,
// its refusal saying so; a deny, a limit and plan mode do not load it
// (decision 6).
func TestGrantLoad(t *testing.T) {
	load := func(t *testing.T, ss SessionSettings, mode string, calls ...llmtest.Turn) ([]string, *llmtest.Script) {
		t.Helper()
		s := llmtest.NewScript(append(calls, llmtest.Text("done"))...)
		c, id, _ := modeAgent(t, s, &acptest.Client{}, ss, "", mode, t.TempDir())
		if _, err := prompt(t, c, id, "go"); err != nil {
			t.Fatal(err)
		}
		reqs := s.Requests()
		return requestTools(t, reqs[len(reqs)-1]), s
	}
	write := func(id, path string) llmtest.Turn {
		return llmtest.ToolCall(id, "write", `{"path":"`+path+`","content":"x"}`)
	}
	tools, s := load(t, SessionSettings{}, "", write("w1", "a.txt"), write("w2", "b.txt"))
	if !slices.Contains(tools, requestPermissionsTool) {
		t.Errorf("a declined write did not load request_permissions: %q", tools)
	}
	if got := lastResult(t, s, 1); !strings.HasSuffix(got, " request_permissions is now available: call it to ask the user for read or write access to paths, for this turn or this session.") {
		t.Errorf("the first decline's result %q", got)
	}
	if got := lastResult(t, s, 2); strings.Contains(got, "request_permissions is now available") {
		t.Errorf("the second decline said it again: %q", got)
	}
	for _, c := range []struct {
		name string
		ss   SessionSettings
		mode string
		call llmtest.Turn
	}{
		{"a deny", SessionSettings{Rules: []permission.Spec{rule(t, permission.ModeDeny, "write", 3)}}, "", write("w", "a.txt")},
		{"a limit's question", SessionSettings{}, "", write("w", "AGENTS.md")},
		{"plan mode", SessionSettings{}, modePlan, write("w", "a.txt")},
		{"a command", SessionSettings{}, "", bashCall(t, "b", "ls")},
	} {
		if tools, _ := load(t, c.ss, c.mode, c.call); slices.Contains(tools, requestPermissionsTool) {
			t.Errorf("%s loaded request_permissions", c.name)
		}
	}
}
