package catalog

import (
	"errors"
	"strings"
	"testing"
)

// ResolveValue is Pi's grammar: variables, escapes, literals and commands;
// a missing variable or a failed command is an error naming it, holding no
// resolved value (0005-PLAN F4-2b).
func TestResolveValue(t *testing.T) {
	env := map[string]string{"TOKEN": "s3cret", "USER_ID": "u1"}
	lookup := func(name string) (string, bool) { v, ok := env[name]; return v, ok }
	run := func(command string) (string, error) {
		if command == "fail" {
			return "s3cret-partial", errors.New("exit status 1")
		}
		return "  out:" + command + "\n", nil
	}
	for _, c := range []struct{ in, want string }{
		{"Bearer $TOKEN", "Bearer s3cret"}, {"${USER_ID}-x", "u1-x"}, {"$$TOKEN", "$TOKEN"}, {"$!x", "!x"},
		{"plain", "plain"}, {"cost $5", "cost $5"}, {"${not-a-name}", "${not-a-name}"}, {"${open", "${open"}, {"!echo hi", "out:echo hi"},
	} {
		got, err := ResolveValue(c.in, lookup, run)
		if err != nil || got != c.want {
			t.Errorf("%q: %q, %v; want %q", c.in, got, err, c.want)
		}
	}
	for _, c := range []struct{ in, want string }{
		{"$MISSING", "MISSING is not set"}, {"$A $B $A", "A, B are not set"}, {"!fail", `"fail" failed`}, {"!   ", "printed nothing"},
	} {
		_, err := ResolveValue(c.in, lookup, func(cmd string) (string, error) {
			if strings.TrimSpace(cmd) == "" {
				return "  ", nil
			}
			return run(cmd)
		})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: error %v, want %q", c.in, err, c.want)
		}
		if err != nil && strings.Contains(err.Error(), "s3cret") {
			t.Errorf("%q: the error %q holds a resolved value", c.in, err)
		}
	}
}
