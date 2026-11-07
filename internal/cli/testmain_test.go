package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"reflect"
	"runtime/debug"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/catalog"
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

func noSignIn(context.Context, string, provider.Prompter, provider.SignInOptions, provider.LoginStore) (provider.Credential, error) {
	return provider.Credential{}, errNoSignIn
}

func noLogout(context.Context, string, provider.LoginOptions, provider.LoginStore) (revoke, err error) {
	return nil, errNoSignIn
}

// offlineModelFacts is the tests' model facts: provider.ModelInfo with no
// key, workspace or login, so the SDK reads the snapshot and models.json
// alone, asks no token source and starts no listing (0046-MADR D3).
func offlineModelFacts(id, _, _ string, _ provider.LoginStore, model string, mj *catalog.ModelsJSON) llm.Model {
	return provider.ModelInfo(id, "", "", nil, model, mj)
}

// The tests' browser opener and sign-ins are TestMain's, which open and
// call nothing; compared by identity, so this test calls none of them
// (0025-PLAN deviation 9).
func TestNoBrowserOrSignIn(t *testing.T) {
	same := func(a, b any) bool { return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer() }
	for name, ok := range map[string]bool{
		"openURL":     same(openURL, noBrowser),
		"signInWith":  same(signInWith, noSignIn),
		"logoutLogin": same(logoutLogin, noLogout),
		"modelFacts":  same(modelFacts, offlineModelFacts),
	} {
		if !ok {
			t.Errorf("%s is not TestMain's: a test could open a browser or sign in", name)
		}
	}
}

// TestMain lets this test binary serve as the stdio MCP fixture (0002-PLAN
// Phase 5), then clears every model credential variable before any test
// runs, so no test reaches a real provider, however the host is set up
// (0002-PLAN Phase 3), makes the model lister fail offline (0020-MADR D7)
// and the model facts read no listing (0046-MADR D3), and replaces the
// browser opener and the sign-ins with ones that open and call nothing
// (0025-PLAN deviation 9).
//
// The tests run behind a network guard (0046-MADR D4, as amended). net/http
// reads the proxy variables once, and the MCP SDK's oauthex package reads
// them while it is initialised, before TestMain. So the first process only
// starts the guard and runs this binary again, with the variables set and
// netGuardChild marking it; the child checks that the guard is in effect,
// then runs the tests. The first process exits with the child's code, or 1
// when a test tried to reach a host.
func TestMain(m *testing.M) {
	mcptest.ServeStdioIfAsked()
	if os.Getenv(netGuardChild) == "" {
		os.Exit(runBehindNetGuard()) //nolint:forbidigo // TestMain's exit
	}
	if err := checkNetGuard(); err != nil {
		fmt.Fprintf(os.Stderr, "network guard: %v\n", err) //nolint:forbidigo // the test binary's own stderr
		os.Exit(2)                                         //nolint:forbidigo // TestMain's exit
	}
	for _, v := range provider.CredentialVars() {
		if err := os.Unsetenv(v); err != nil {
			fmt.Fprintf(os.Stderr, "unset %s: %v\n", v, err) //nolint:forbidigo // the test binary's own stderr
			os.Exit(2)                                       //nolint:forbidigo // TestMain's exit
		}
	}
	listModels = offlineList
	modelFacts = offlineModelFacts
	openURL, signInWith, logoutLogin = noBrowser, noSignIn, noLogout
	unownedListing = recordUnowned
	code := m.Run()
	if stacks := unownedStacks(); len(stacks) > 0 {
		fmt.Fprintf(os.Stderr, "%d listing(s) started on a runEnv nobody owns; call ownListings(t, e) where it is built (0046-MADR D2):\n", //nolint:forbidigo // the test binary's own stderr
			len(stacks))
		for _, s := range stacks[:min(len(stacks), 5)] {
			fmt.Fprintf(os.Stderr, "%s\n", s) //nolint:forbidigo // the test binary's own stderr
		}
		code = 1
	}
	os.Exit(code) //nolint:forbidigo // TestMain's exit
}

// unowned holds the stacks of the listings started on a runEnv nobody owns
// (0046-MADR D2 as amended).
var unowned struct {
	sync.Mutex
	stacks []string
}

// recordUnowned is the tests' unownedListing: it keeps the caller's stack.
func recordUnowned() {
	unowned.Lock()
	defer unowned.Unlock()
	unowned.stacks = append(unowned.stacks, string(debug.Stack()))
}

func unownedStacks() []string {
	unowned.Lock()
	defer unowned.Unlock()
	return slices.Clone(unowned.stacks)
}

// netGuardChild marks the process that runs the tests behind the guard.
const netGuardChild = "GOBBLE_TEST_NETGUARD"

// proxyVars are the variables net/http reads for a proxy.
var proxyVars = []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"}

// runBehindNetGuard starts the guard, runs this binary again with the same
// arguments, the proxy variables naming the guard and netGuardChild set,
// and returns the child's exit code, or 1 when the guard recorded a host,
// which it names.
func runBehindNetGuard() int {
	guard, err := startNetGuard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "network guard: %v\n", err) //nolint:forbidigo // the test binary's own stderr
		return 2
	}
	cmd := exec.CommandContext(context.Background(), os.Args[0], os.Args[1:]...) //nolint:gosec // G204: this test binary, run again with its own arguments
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = guardedEnv(os.Environ(), guard.url())
	code := 0
	if err := cmd.Run(); err != nil {
		code = 2
		if ee, ok := errors.AsType[*exec.ExitError](err); ok {
			code = ee.ExitCode()
		} else {
			fmt.Fprintf(os.Stderr, "network guard: run the tests: %v\n", err) //nolint:forbidigo // the test binary's own stderr
		}
	}
	if seen := guard.close(); len(seen) > 0 {
		fmt.Fprintln(os.Stderr, "tests tried to reach the network (0046-MADR D4):") //nolint:forbidigo // the test binary's own stderr
		for _, host := range slices.Sorted(maps.Keys(seen)) {
			fmt.Fprintf(os.Stderr, "  %s x%d\n", host, seen[host]) //nolint:forbidigo // the test binary's own stderr
		}
		code = 1
	}
	return code
}

// guardedEnv is env with every proxy variable naming proxy, NO_PROXY and
// no_proxy removed, and netGuardChild set.
func guardedEnv(env []string, proxy string) []string {
	drop := append(slices.Clone(proxyVars), "NO_PROXY", "no_proxy", netGuardChild)
	out := slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.ContainsFunc(drop, func(d string) bool { return strings.EqualFold(d, name) })
	})
	for _, v := range proxyVars {
		out = append(out, v+"="+proxy)
	}
	return append(out, netGuardChild+"=1")
}

// checkNetGuard reports whether net/http sends a request off the machine
// to the guard: HTTPS_PROXY is set, and ProxyFromEnvironment names it.
func checkNetGuard() error {
	want := os.Getenv("HTTPS_PROXY")
	if want == "" {
		return errors.New("HTTPS_PROXY is not set in the child")
	}
	req, err := http.NewRequest(http.MethodGet, "https://example.com/", nil) //nolint:noctx // never sent
	if err != nil {
		return err
	}
	got, err := http.ProxyFromEnvironment(req)
	if err != nil {
		return fmt.Errorf("net/http's proxy for https://example.com/: %w", err)
	}
	if got == nil || got.String() != want {
		return fmt.Errorf("net/http proxies https://example.com/ through %v, not the guard %s", got, want)
	}
	return nil
}

// netGuard is TestMain's proxy. HTTPS_PROXY and HTTP_PROXY name it, so every
// request a test's HTTP client sends off the machine comes here: it records
// the request's host, answers 403 and forwards nothing (0046-MADR D4). Go
// proxies no request for localhost or a loopback address, so httptest
// servers are not affected.
type netGuard struct {
	ln   net.Listener
	wg   sync.WaitGroup
	mu   sync.Mutex
	seen map[string]int
}

// startNetGuard listens on a loopback port and serves the guard there.
func startNetGuard() (*netGuard, error) {
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	g := &netGuard{ln: ln, seen: map[string]int{}}
	g.wg.Go(g.serve)
	return g, nil
}

// url is the guard's address as a proxy variable names it.
func (g *netGuard) url() string { return "http://" + g.ln.Addr().String() }

func (g *netGuard) serve() {
	for {
		c, err := g.ln.Accept()
		if err != nil {
			return
		}
		g.wg.Go(func() { g.answer(c) })
	}
}

// answer records one connection's request and refuses it.
func (g *netGuard) answer(c net.Conn) {
	defer c.Close() //nolint:errcheck // the refusal is written; nothing reads the close
	_ = c.SetDeadline(time.Now().Add(5 * time.Second))
	line, _ := bufio.NewReader(c).ReadString('\n') //nolint:errcheck // a partial line still names the host
	g.mu.Lock()
	g.seen[requestHost(line)]++
	g.mu.Unlock()
	_, _ = io.WriteString(c, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
}

// requestHost is a proxied request's target: CONNECT's host:port, or the
// host of a plain request's URL.
func requestHost(line string) string {
	f := strings.Fields(line)
	switch {
	case len(f) < 2:
		return strings.TrimSpace(line)
	case f[0] == "CONNECT":
		return f[1]
	}
	if u, err := url.Parse(f[1]); err == nil && u.Host != "" {
		return u.Host
	}
	return f[1]
}

// close stops the guard and returns the hosts it was asked for, with how
// many times.
func (g *netGuard) close() map[string]int {
	_ = g.ln.Close()
	g.wg.Wait()
	g.mu.Lock()
	defer g.mu.Unlock()
	return maps.Clone(g.seen)
}
