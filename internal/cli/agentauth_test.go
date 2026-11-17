package cli

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/maccavelli/go-llmprovider-sdk/authcontract"

	"github.com/maccavelli/gobble-cli/acpserver"
	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// hostIDFile is the host id file in the state directory a test's GOBBLE_HOME
// makes.
func hostIDFile(t *testing.T) string {
	t.Helper()
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dirs.State, "host-id")
}

// recordAgentStart replaces the ACP path's sign-in with one that records the
// options it was given and ends in an error of its own.
func recordAgentStart(t *testing.T) *[]provider.SignInOptions {
	t.Helper()
	got := new([]provider.SignInOptions)
	replace(t, &startSignInWith, func(_ context.Context, _, _ string, o provider.SignInOptions, _ provider.LoginStore) (*provider.SignInRun, error) {
		*got = append(*got, o)
		return nil, errors.New("recorded")
	})
	return got
}

// recordTerminalSignIn replaces the terminals' sign-in with one that records
// the options it was given and signs in to nothing.
func recordTerminalSignIn(t *testing.T) *[]provider.SignInOptions {
	t.Helper()
	got := new([]provider.SignInOptions)
	replace(t, &signInWith, func(_ context.Context, _ string, _ provider.Prompter, o provider.SignInOptions, _ provider.LoginStore) (provider.Credential, error) {
		*got = append(*got, o)
		return provider.Credential{}, nil
	})
	return got
}

// Both paths read the host id from <state>/host-id: gobble auth login, which
// /login, gobble setup and the first run share through signInTo, and the ACP
// path's sign-in; with no file the first call creates one and the other path
// reads the same id (0051-PLAN P6 step 1a).
func TestCLIFillsHostID(t *testing.T) {
	c := newCfgEnv(t)
	terminal := recordTerminalSignIn(t)
	acp := recordAgentStart(t)
	e := envHere(t)

	// No file yet: the terminal creates it.
	if r := c.run("auth", "login", "grok"); r.code != ExitOK {
		t.Fatalf("auth login: exit %d, stderr %q", r.code, r.stderr)
	}
	b, err := os.ReadFile(hostIDFile(t))
	if err != nil {
		t.Fatalf("no host-id after a terminal sign-in: %v", err)
	}
	id := strings.TrimSpace(string(b))
	if !strings.HasPrefix(id, "urn:uuid:") {
		t.Fatalf("host-id %q is no urn:uuid", id)
	}
	if len(*terminal) != 1 || (*terminal)[0].HostID != id {
		t.Errorf("the terminal sign-in's options %+v, want HostID %q", *terminal, id)
	}
	// /login's, setup's and the first run's site, signInTo.
	if err := e.signInTo(t.Context(), nil, "grok", provider.SignInOptions{}, nil); err != nil {
		t.Fatal(err)
	}
	if len(*terminal) != 2 || (*terminal)[1].HostID != id {
		t.Errorf("signInTo's options %+v, want HostID %q", *terminal, id)
	}
	// The ACP path reads the same file.
	if _, err := e.startSignIn(t.Context(), "grok", "device_code", false); err == nil || err.Error() != "recorded" {
		t.Fatalf("startSignIn: %v, want the recorder's error", err)
	}
	if len(*acp) != 1 || (*acp)[0].HostID != id {
		t.Errorf("the ACP sign-in's options %+v, want HostID %q", *acp, id)
	}
	// And the other way: the ACP path creates it when the terminal has not.
	if err := os.Remove(hostIDFile(t)); err != nil {
		t.Fatal(err)
	}
	if _, err := e.startSignIn(t.Context(), "grok", "device_code", false); err == nil {
		t.Fatal("the recorder gave no error")
	}
	b, err = os.ReadFile(hostIDFile(t))
	if err != nil || len(*acp) != 2 || (*acp)[1].HostID == "" || strings.TrimSpace(string(b)) != (*acp)[1].HostID {
		t.Fatalf("the ACP path's id %+v, file %q (%v), want the id it created", *acp, b, err)
	}
	if r := c.run("auth", "login", "grok"); r.code != ExitOK || (*terminal)[2].HostID != (*acp)[1].HostID {
		t.Errorf("the terminal after the ACP path: %+v, want the same id %q", *terminal, (*acp)[1].HostID)
	}
}

// When the host id cannot be read or made (the state directory is a file), a
// sign-in goes on with no host id and one warning that names the error and
// never an id, on both paths, whatever the provider (the owner's ruling,
// "Proceed, warn once"; 0051-PLAN P6 step 1a).
func TestSignInWithoutHostID(t *testing.T) {
	c := newCfgEnv(t)
	t.Setenv("GOBBLE_STATE_DIR", filepath.Join(c.dir, "state-file"))
	c.write(filepath.Join(c.dir, "state-file"), "not a directory")
	terminal := recordTerminalSignIn(t)
	r := c.run("auth", "login", "grok")
	if r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q, want the sign-in to go on", r.code, r.stderr)
	}
	if len(*terminal) != 1 || (*terminal)[0].HostID != "" {
		t.Errorf("the terminal sign-in's options %+v, want one call with no host id", *terminal)
	}
	if n := strings.Count(r.stderr, "no host id"); n != 1 || !strings.Contains(r.stderr, "host-id") || strings.Contains(r.stderr, "urn:uuid") {
		t.Errorf("stderr %q, want one warning naming the host-id error and no id", r.stderr)
	}

	// The ACP path, through the real engine against an issuer in process.
	ti := newLoginIssuer(t)
	target, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	var seen []provider.SignInOptions
	replace(t, &startSignInWith, func(ctx context.Context, id, method string, o provider.SignInOptions, store provider.LoginStore) (*provider.SignInRun, error) {
		seen = append(seen, o)
		o.HTTPClient = &http.Client{Transport: loginTransport{target}}
		return provider.StartSignIn(ctx, id, method, o, store)
	})
	e := envHere(t)
	run, err := e.startSignIn(t.Context(), "grok", "device_code", false)
	if err != nil {
		t.Fatalf("an ACP sign-in with no host id: %v", err)
	}
	for range run.Steps() {
	}
	res, err := run.Wait(t.Context())
	if err != nil || res.Yields != authcontract.YieldsOAuthSession {
		t.Fatalf("Wait = %+v, %v, want the sign-in to complete", res, err)
	}
	if len(seen) != 1 || seen[0].HostID != "" {
		t.Errorf("the ACP sign-in's options %+v, want one call with no host id", seen)
	}
	logged := e.logs.stderr.(*syncBuffer).String()
	if n := strings.Count(logged, "no host id"); n != 1 || strings.Contains(logged, "urn:uuid") {
		t.Errorf("the agent's log %q, want one warning and no id", logged)
	}
	if got := c.read(c.authFile()); !strings.Contains(got, `"access": "access-1"`) {
		t.Errorf("auth.json after the sign-in:\n%s", got)
	}
}

// A Kilo device sign-in over the ACP path stores its key through auth.SetKey,
// escaped as signInTo escapes it, and its result holds no key.
func TestAgentSignInStoresKiloKey(t *testing.T) {
	c := newCfgEnv(t)
	ti := newLoginIssuer(t)
	target, err := url.Parse(ti.URL)
	if err != nil {
		t.Fatal(err)
	}
	replace(t, &startSignInWith, func(ctx context.Context, id, method string, o provider.SignInOptions, store provider.LoginStore) (*provider.SignInRun, error) {
		o.HTTPClient = &http.Client{Transport: loginTransport{target}}
		return provider.StartSignIn(ctx, id, method, o, store)
	})
	e := envHere(t)
	run, err := e.startSignIn(t.Context(), "kilo", "device_code", false)
	if err != nil {
		t.Fatal(err)
	}
	for range run.Steps() {
	}
	res, err := run.Wait(t.Context())
	if err != nil || res.Key != "" || res.Yields != authcontract.YieldsDeviceKey {
		t.Fatalf("Wait = %+v, %v, want device_key and no key in the result", res, err)
	}
	f, err := auth.Load(c.authFile())
	if err != nil {
		t.Fatal(err)
	}
	en, ok := f.Entry("kilo")
	if !ok || en.Type != auth.TypeAPIKey || en.Key != "$!kilo$$token-0123456789" {
		t.Errorf("kilo's entry %+v, want the key stored escaped", en)
	}
}

// holds reads the stored credential's kind: a key is what a key method yields,
// a session what a session method of its kind yields, so Grok's browser and
// device sign-ins count as one (0051-MADR D4 step 2).
func TestAgentHolds(t *testing.T) {
	c := newCfgEnv(t)
	e := envHere(t)
	method := func(id, mid string) authcontract.Method {
		for _, m := range provider.Offered(id) {
			if m.ID == mid {
				return m
			}
		}
		t.Fatalf("%s offers no %s", id, mid)
		return authcontract.Method{}
	}
	if e.holds("grok", method("grok", "device_code")) {
		t.Error("holds with no credential")
	}
	c.write(c.authFile(), `{"grok": {"type": "api_key", "key": "sk-test-only"}}`)
	e = envHere(t)
	if !e.holds("grok", method("grok", "api_key")) || e.holds("grok", method("grok", "device_code")) {
		t.Error("a key held is a key's, not a session's")
	}
	if err := (auth.Logins{Path: c.authFile()}).Save(t.Context(), "grok", &provider.Login{Access: "a", Refresh: "r", Issuer: "https://auth.x.ai",
		ClientID: "b1a00492-073a-47ea-816f-4c329264a828", TokenURL: "https://auth.x.ai/oauth2/token", Kind: "grok"}); err != nil {
		t.Fatal(err)
	}
	e = envHere(t)
	if !e.holds("grok", method("grok", "device_code")) || !e.holds("grok", method("grok", "browser_oauth")) || e.holds("grok", method("grok", "api_key")) {
		t.Error("a session held is the session of both of grok's sign-ins, and no key's")
	}
}

// keyResult is a sign-in run whose Wait gives a device key at once, whatever its
// context says: the result provider.SignInRun.Wait returns for a Kilo sign-in
// the listing accepted (llm/provider/signinrun.go, finish), whose caller's
// context ended after it returned.
type keyResult struct{ acpserver.SignInRun }

func (keyResult) Wait(context.Context) (provider.SignInResult, error) {
	return provider.SignInResult{Key: "kilo-token-0123456789", Yields: authcontract.YieldsDeviceKey}, nil
}

// A device key is stored only for a sign-in still wanted: with the context done
// when Wait returns the key, auth.json is not written and the sign-in is
// canceled (0051-MADR D4 step 3); with the context live, the key is stored.
func TestKeyRunStoresNothingOnceCancelled(t *testing.T) {
	c := newCfgEnv(t)
	for _, cancelled := range []bool{true, false} {
		r := &keyRun{SignInRun: keyResult{}, id: "kilo", path: c.authFile()}
		ctx, cancel := context.WithCancel(t.Context())
		if cancelled {
			cancel()
		}
		res, err := r.Wait(ctx)
		cancel()
		_, statErr := os.Stat(c.authFile())
		switch {
		case cancelled && (err == nil || provider.ErrorKind(err) != string(authcontract.FailureCanceled) || res.Key != ""):
			t.Errorf("a cancelled key sign-in gave %+v, %v, want a canceled error and no key", res, err)
		case cancelled && statErr == nil:
			t.Error("a cancelled key sign-in wrote auth.json")
		case !cancelled && (err != nil || statErr != nil):
			t.Errorf("a live key sign-in gave %v, stat %v, want the key stored", err, statErr)
		}
	}
}

// A remote authenticate reaches the engine's start with Remote true, and a
// local one with it false: the option that sets no-loopback for every method
// (llm/provider's TestStartSignInRemoteSetsNoLoopback) is passed from here (the
// owner's ruling of 2026-11-16, "Assert what gobble passes").
func TestAgentRemoteReachesTheEngine(t *testing.T) {
	newCfgEnv(t)
	got := recordAgentStart(t)
	e := envHere(t)
	for _, remote := range []bool{true, false} {
		*got = nil
		if _, err := e.startSignIn(t.Context(), "grok", "device_code", remote); err == nil {
			t.Fatal("the recorder gave no error")
		}
		if len(*got) != 1 || (*got)[0].Remote != remote {
			t.Errorf("remote %v: the engine was started with %+v, want Remote %v", remote, *got, remote)
		}
	}
}
