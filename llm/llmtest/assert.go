package llmtest

import (
	"encoding/json/v2"
	"slices"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// AssertSystem reports, through t, each section missing from req's system
// messages.
func AssertSystem(t testing.TB, req *llm.Request, sections ...string) {
	t.Helper()
	var b strings.Builder
	for _, m := range req.Messages {
		if m.Role != llm.RoleSystem {
			continue
		}
		for _, c := range m.Content {
			b.WriteString(c.Text)
			b.WriteByte('\n')
		}
	}
	system := b.String()
	for _, s := range sections {
		if !strings.Contains(system, s) {
			t.Errorf("system prompt lacks %q; it is:\n%s", s, system)
		}
	}
}

// AssertTools reports, through t, when req's tool names are not names, in
// order. req.Tools is a JSON array of objects with a "name".
func AssertTools(t testing.TB, req *llm.Request, names ...string) {
	t.Helper()
	var specs []struct {
		Name string `json:"name"`
	}
	if len(req.Tools) > 0 {
		if err := json.Unmarshal(req.Tools, &specs); err != nil {
			t.Errorf("request tools are not a JSON array of named specs: %v", err)
			return
		}
	}
	got := make([]string, len(specs))
	for i, s := range specs {
		got[i] = s.Name
	}
	if !slices.Equal(got, names) {
		t.Errorf("request tools = %q, want %q", got, names)
	}
}

// AssertTail reports, through t, when the roles of req's last messages are
// not roles.
func AssertTail(t testing.TB, req *llm.Request, roles ...llm.Role) {
	t.Helper()
	got := make([]llm.Role, len(req.Messages))
	for i, m := range req.Messages {
		got[i] = m.Role
	}
	if len(got) < len(roles) || !slices.Equal(got[len(got)-len(roles):], roles) {
		t.Errorf("request message roles = %q, want them to end with %q", got, roles)
	}
}
