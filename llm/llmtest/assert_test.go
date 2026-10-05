package llmtest

import (
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

// recordingTB captures failures, so an assertion can be seen to fail.
type recordingTB struct {
	testing.TB
	errors []string
}

func (r *recordingTB) Helper() {}

func (r *recordingTB) Errorf(format string, args ...any) {
	r.errors = append(r.errors, format)
	_ = args
}

var sample = &llm.Request{
	Messages: []llm.Message{
		{Role: llm.RoleSystem, Content: []llm.Content{{Type: "text", Text: "You are gobble.\n# Tools\nread, edit"}}},
		{Role: llm.RoleUser, Content: []llm.Content{{Type: "text", Text: "fix it"}}},
		{Role: llm.RoleAssistant},
		{Role: llm.RoleTool},
	},
	Tools: []byte(`[{"name":"read","description":"Read a file"},{"name":"edit"}]`),
}

func TestAssertionsPass(t *testing.T) {
	AssertSystem(t, sample, "You are gobble.", "# Tools")
	AssertTools(t, sample, "read", "edit")
	AssertTail(t, sample, llm.RoleAssistant, llm.RoleTool)
	AssertTools(t, &llm.Request{})
}

func TestAssertionsFail(t *testing.T) {
	cases := map[string]func(testing.TB){
		"missing section": func(tb testing.TB) { AssertSystem(tb, sample, "# Skills") },
		"wrong tools":     func(tb testing.TB) { AssertTools(tb, sample, "edit", "read") },
		"tools not JSON":  func(tb testing.TB) { AssertTools(tb, &llm.Request{Tools: []byte(`{`)}, "x") },
		"wrong tail":      func(tb testing.TB) { AssertTail(tb, sample, llm.RoleUser) },
		"tail too long":   func(tb testing.TB) { AssertTail(tb, &llm.Request{}, llm.RoleUser) },
	}
	for name, assert := range cases {
		t.Run(name, func(t *testing.T) {
			r := &recordingTB{TB: t}
			assert(r)
			if len(r.errors) != 1 {
				t.Fatalf("got %d failures (%q), want 1", len(r.errors), strings.Join(r.errors, "; "))
			}
		})
	}
}
