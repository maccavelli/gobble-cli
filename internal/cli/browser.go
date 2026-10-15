package cli

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"runtime"
)

// openURL opens u, an https URL, in the user's browser, without a shell:
// open on macOS, the URL handler on Windows, xdg-open elsewhere. It does
// not wait for the browser. A test replaces it (0005-PLAN F4-3b decision 6).
var openURL = func(ctx context.Context, u string) error {
	name, args, err := browserCommand(runtime.GOOS, u)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // G204: a fixed opener, given one checked https URL
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("open browser: %w", err)
	}
	go func() { _ = cmd.Wait() }() //nolint:errcheck // reaped only; the opener's status says nothing of the login
	return nil
}

// browserCommand is the command that opens u on goos. A URL that is not
// https, with a host, is refused.
func browserCommand(goos, u string) (string, []string, error) {
	p, err := url.Parse(u)
	if err != nil || p.Scheme != "https" || p.Host == "" {
		return "", nil, fmt.Errorf("open browser: %q is not an https URL", u)
	}
	switch goos {
	case "darwin":
		return "open", []string{u}, nil
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", u}, nil
	}
	return "xdg-open", []string{u}, nil
}
