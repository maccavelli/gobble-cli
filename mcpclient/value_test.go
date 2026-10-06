package mcpclient

import (
	"strings"
	"testing"
)

// Resolve follows Pi's template rules (resolve-config-value.ts).
func TestResolve(t *testing.T) {
	env := map[string]string{"TOKEN": "t0k", "A": "1", "EMPTY": ""}
	getenv := func(k string) string { return env[k] }
	const what = `MCP server "web" header "Authorization"`
	ok := []struct{ in, want string }{
		{"Bearer ${TOKEN}", "Bearer t0k"},
		{"$TOKEN-$A", "t0k-1"},
		{"plain", "plain"},
		{"$$TOKEN", "$TOKEN"},
		{"$!x", "!x"},
		{"cost $5", "cost $5"},
		{"${not a name}", "${not a name}"},
		{"${open", "${open"},
		{"trailing $", "trailing $"},
		{"${}", "${}"},
	}
	for _, c := range ok {
		got, err := Resolve(c.in, what, getenv)
		if err != nil || got != c.want {
			t.Errorf("Resolve(%q) = %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	bad := []struct{ in, want string }{
		{"Bearer ${MISSING}", `Failed to resolve MCP server "web" header "Authorization" from environment variable: MISSING`},
		{"${EMPTY}", `Failed to resolve MCP server "web" header "Authorization" from environment variable: EMPTY`},
		{"$X $Y $X", `Failed to resolve MCP server "web" header "Authorization" from environment variables: X, Y`},
		{"!gh auth token", `MCP server "web" header "Authorization": command values (!…) arrive with 0005-PLAN F8`},
	}
	for _, c := range bad {
		if _, err := Resolve(c.in, what, getenv); err == nil || err.Error() != c.want {
			t.Errorf("Resolve(%q) = %v, want %q", c.in, err, c.want)
		}
	}
}

// The vectors were computed independently, with Python's hashlib, the way
// Node computes Pi's createMcpToolName.
func TestToolName(t *testing.T) {
	long := strings.Repeat("t", 70)
	cases := []struct {
		server, tool string
		taken        bool
		want         string
	}{
		{"fx", "dotted.name", false, "mcp__fx__dotted_name"},
		{"fx", long, false, "mcp__fx__tttttttttttttttttttttttttttttttttttttttttttttt_c083de7e"},
		{"fx", "echo", true, "mcp__fx__echo_aa36cec7"},
		{"srv", "café\U0001F600", false, "mcp__srv__caf___"},
	}
	for _, c := range cases {
		got := ToolName(c.server, c.tool, func(string) bool { return c.taken })
		if got != c.want || len(got) > maxToolName {
			t.Errorf("ToolName(%q, %q, taken=%v) = %q (%d), want %q", c.server, c.tool, c.taken, got, len(got), c.want)
		}
	}
}
