package cli

import (
	"slices"
	"testing"
)

// openURL opens only an https URL, with each system's opener, never a shell.
func TestOpenURL(t *testing.T) {
	const u = "https://auth.x.ai/oauth2/authorize?a=1&b=2"
	for goos, want := range map[string][]string{
		"darwin": {"open", u}, "windows": {"rundll32", "url.dll,FileProtocolHandler", u}, "linux": {"xdg-open", u},
	} {
		name, args, err := browserCommand(goos, u)
		if got := append([]string{name}, args...); err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: %q, %v; want %q", goos, got, err, want)
		}
	}
	for _, bad := range []string{"http://auth.x.ai/x", "file:///etc/passwd", "javascript:alert(1)", "https://"} {
		if _, _, err := browserCommand("linux", bad); err == nil {
			t.Errorf("%q was not refused", bad)
		}
	}
}
