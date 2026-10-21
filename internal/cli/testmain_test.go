package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/maccavelli/gobble-cli/llm/provider"
	"github.com/maccavelli/gobble-cli/mcpclient/mcptest"
)

// errOfflineList is the tests' lister's answer: no test lists over the
// network (0020-MADR D7).
var errOfflineList = errors.New("tests list no models")

func offlineList(context.Context, string, string, provider.Options) (provider.Listing, error) {
	return provider.Listing{}, errOfflineList
}

// errNoSignIn is the tests' sign-ins' answer: no test opens a browser or
// signs in to a provider (0025-PLAN deviation 9).
var errNoSignIn = errors.New("tests open no browser and sign in nowhere")

func noBrowser(context.Context, string) error { return errNoSignIn }

func noBrowserLogin(context.Context, string, provider.LoginOptions, provider.LoginStore) error {
	return errNoSignIn
}

func noDeviceLogin(context.Context, string, provider.LoginOptions, func(provider.DeviceCode) error, provider.LoginStore) error {
	return errNoSignIn
}

func noKeyLogin(context.Context, string, provider.LoginOptions, func(provider.DeviceCode) error) (string, error) {
	return "", errNoSignIn
}

func noLogout(context.Context, string, provider.LoginOptions, provider.LoginStore) (revoke, err error) {
	return nil, errNoSignIn
}

// The tests' browser opener and sign-ins are TestMain's, which open and
// call nothing; compared by identity, so this test calls none of them
// (0025-PLAN deviation 9).
func TestNoBrowserOrSignIn(t *testing.T) {
	same := func(a, b any) bool { return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() }
	for name, ok := range map[string]bool{
		"openURL":      same(openURL, noBrowser),
		"browserLogin": same(browserLogin, noBrowserLogin),
		"deviceLogin":  same(deviceLogin, noDeviceLogin),
		"keyLogin":     same(keyLogin, noKeyLogin),
		"logoutLogin":  same(logoutLogin, noLogout),
	} {
		if !ok {
			t.Errorf("%s is not TestMain's: a test could open a browser or sign in", name)
		}
	}
}

// TestMain lets this test binary serve as the stdio MCP fixture (0002-PLAN
// Phase 5), then clears every model credential variable before any test
// runs, so no test reaches a real provider, however the host is set up
// (0002-PLAN Phase 3), makes the model lister fail offline (0020-MADR D7),
// and replaces the browser opener and the sign-ins with ones that open and
// call nothing (0025-PLAN deviation 9).
func TestMain(m *testing.M) {
	mcptest.ServeStdioIfAsked()
	for _, v := range provider.CredentialVars() {
		if err := os.Unsetenv(v); err != nil {
			fmt.Fprintf(os.Stderr, "unset %s: %v\n", v, err) //nolint:forbidigo // the test binary's own stderr
			os.Exit(2)                                       //nolint:forbidigo // TestMain's exit
		}
	}
	listModels = offlineList
	openURL, browserLogin, deviceLogin, keyLogin, logoutLogin = noBrowser, noBrowserLogin, noDeviceLogin, noKeyLogin, noLogout
	os.Exit(m.Run()) //nolint:forbidigo // TestMain's exit
}
