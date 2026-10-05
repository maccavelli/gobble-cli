package cli

import "testing"

// All eight combinations of D1's rule.
func TestResolveMode(t *testing.T) {
	for _, tc := range []struct {
		print, stdinTTY, stdoutTTY bool
		want                       Mode
	}{
		{false, true, true, ModeLine},
		{true, true, true, ModePrint},
		{false, false, true, ModePrint},
		{false, true, false, ModePrint},
		{true, false, true, ModePrint},
		{true, true, false, ModePrint},
		{false, false, false, ModePrint},
		{true, false, false, ModePrint},
	} {
		if got := resolveMode(tc.print, tc.stdinTTY, tc.stdoutTTY); got != tc.want {
			t.Errorf("resolveMode(print=%v, stdinTTY=%v, stdoutTTY=%v) = %v, want %v",
				tc.print, tc.stdinTTY, tc.stdoutTTY, got, tc.want)
		}
	}
}
