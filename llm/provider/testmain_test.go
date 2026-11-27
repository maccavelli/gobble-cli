package provider

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestMain runs the package's tests behind the network guard internal/cli's
// tests run behind (0046-MADR D4; 0072-MADR D10, M10): net/http reads the
// proxy variables once, so the first process starts the guard and runs this
// binary again with them naming it. The tests whose assertions need a dial
// with no proxy, unguardedTests, run in a second child without the proxy
// variables, which first checks that net/http names none; the guarded child
// leaves them out (0072-MADR M79, the coordinator's call on Q-C). No
// assertion of theirs changed.
func TestMain(m *testing.M) {
	switch {
	case os.Getenv(unguardedChild) != "":
		if err := checkNoProxy(); err != nil {
			fmt.Fprintf(os.Stderr, "unguarded child: %v\n", err) //nolint:forbidigo // the test binary's own stderr
			os.Exit(2)                                           //nolint:forbidigo // TestMain's exit
		}
	case os.Getenv(netGuardChild) == "":
		os.Exit(runGuardedAndUnguarded()) //nolint:forbidigo // TestMain's exit
	default:
		if err := checkNetGuard(); err != nil {
			fmt.Fprintf(os.Stderr, "network guard: %v\n", err) //nolint:forbidigo // the test binary's own stderr
			os.Exit(2)                                         //nolint:forbidigo // TestMain's exit
		}
	}
	if err := installTestModelData(); err != nil { // 0072-MADR M48
		fmt.Fprintf(os.Stderr, "model data: %v\n", err) //nolint:forbidigo // the test binary's own stderr
		os.Exit(2)                                      //nolint:forbidigo // TestMain's exit
	}
	os.Exit(m.Run()) //nolint:forbidigo // TestMain's exit
}

// unguardedTests dial with no proxy: a refused loopback port and a .invalid
// name (TestWireOffline), and the probe of a URL's own host (TestProbePort).
var unguardedTests = []string{"TestWireOffline", "TestProbePort"}

// unguardedChild marks the child that runs unguardedTests with no proxy.
const unguardedChild = "GOBBLE_TEST_UNGUARDED"

// runGuardedAndUnguarded runs the guarded child, leaving unguardedTests out,
// then, when -test.run selects any of them, the unguarded child with just
// those; it returns the first nonzero exit code, else 0.
func runGuardedAndUnguarded() int {
	flag.Parse()
	run, skip := flag.Lookup("test.run").Value.String(), flag.Lookup("test.skip").Value.String()
	only := "^(" + strings.Join(unguardedTests, "|") + ")$"
	guardedSkip := only
	if skip != "" {
		guardedSkip = skip + "|" + only
	}
	code := runBehindNetGuard("-test.skip=" + guardedSkip)
	if sel := selectedUnguarded(run, skip); len(sel) > 0 {
		args := append(slices.Clone(os.Args[1:]), "-test.run=^("+strings.Join(sel, "|")+")$"+subPattern(run),
			"-test.bench=", "-test.fuzz=")
		owned := childOutputs()
		for _, o := range owned {
			args = append(args, "-"+o.flag+"="+o.child)
		}
		defer mergeChildOutputs(owned)
		cmd := exec.CommandContext(context.Background(), os.Args[0], args...) //nolint:gosec // G204: this test binary, run again with its own arguments
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		drop := append(slices.Clone(proxyVars), "NO_PROXY", "no_proxy", netGuardChild, unguardedChild)
		cmd.Env = append(slices.DeleteFunc(slices.Clone(os.Environ()), func(kv string) bool {
			name, _, _ := strings.Cut(kv, "=")
			return slices.ContainsFunc(drop, func(d string) bool { return strings.EqualFold(d, name) })
		}), unguardedChild+"=1")
		if err := cmd.Run(); err != nil && code == 0 {
			code = 2
			if ee, ok := errors.AsType[*exec.ExitError](err); ok {
				code = ee.ExitCode()
			}
		}
	}
	return code
}

// childOutput is a file the unguarded child would write over the first
// process's: the flag naming it, the parent's path, and the child's own.
type childOutput struct {
	flag, parent, child string
	merge               bool
}

// childOutputs are the files a test binary writes that the go command
// reads afterwards, each given to the unguarded child under its own name,
// the parent's with ".unguarded" added, so the child truncates none of
// the guarded child's: the test log, which the test cache records the
// run's inputs from, and the coverage profile, both appended to the
// parent's afterwards; and the CPU, memory, block, mutex and trace
// profiles, which cannot be appended and are left beside the parent's
// (0072-PLAN's review deviation, R2).
func childOutputs() []childOutput {
	var out []childOutput
	for _, name := range []string{"test.testlogfile", "test.coverprofile", "test.cpuprofile", "test.memprofile",
		"test.blockprofile", "test.mutexprofile", "test.trace"} {
		f := flag.Lookup(name)
		if f == nil || f.Value.String() == "" {
			continue
		}
		p := f.Value.String()
		if dir := flag.Lookup("test.outputdir"); name != "test.testlogfile" && !filepath.IsAbs(p) && dir != nil && dir.Value.String() != "" {
			p = filepath.Join(dir.Value.String(), p)
		}
		out = append(out, childOutput{flag: name, parent: p, child: p + ".unguarded",
			merge: name == "test.testlogfile" || name == "test.coverprofile"})
	}
	return out
}

// mergeChildOutputs appends the child's test log and coverage profile to
// the parent's, without their first line ("# test log", "mode: …"), and
// removes them.
func mergeChildOutputs(owned []childOutput) {
	for _, o := range owned {
		if !o.merge {
			continue
		}
		b, err := os.ReadFile(o.child)
		if err != nil {
			continue
		}
		if _, rest, ok := strings.Cut(string(b), "\n"); ok {
			f, err := os.OpenFile(o.parent, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o600)
			if err == nil {
				_, err = io.WriteString(f, rest)
				err = errors.Join(err, f.Close())
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "unguarded child: %s: %v\n", o.flag, err) //nolint:forbidigo // the test binary's own stderr
			}
		}
		_ = os.Remove(o.child)
	}
}

// logProbeEnv, when set, makes TestLogProbe_0072 read logProbeInput, an
// input the test log records, in the guarded child.
const logProbeEnv, logProbeInput = "GOBBLE_TEST_LOGPROBE", "GOBBLE_TEST_LOGPROBE_A"

// TestLogProbe_0072 reads logProbeInput when logProbeEnv is set, for
// TestUnguardedChildKeepsTestLog_0072; otherwise it does nothing.
func TestLogProbe_0072(t *testing.T) {
	if os.Getenv(logProbeEnv) != "" {
		_ = os.Getenv(logProbeInput)
	}
}

// M79, R2: run with -test.testlogfile, the binary's test log holds one
// header and the guarded child's inputs: the unguarded child, run beside it,
// wrote its own log, appended after, and truncated nothing.
func TestUnguardedChildKeepsTestLog_0072(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "test.log")
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^(TestLogProbe_0072|TestProbePort)$", "-test.count=1",
		"-test.testlogfile="+logPath)
	drop := append(slices.Clone(proxyVars), "NO_PROXY", "no_proxy", netGuardChild, unguardedChild)
	cmd.Env = append(slices.DeleteFunc(slices.Clone(os.Environ()), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.ContainsFunc(drop, func(d string) bool { return strings.EqualFold(d, name) })
	}), logProbeEnv+"=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the binary: %v\n%s", err, out)
	}
	b, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	log := string(b)
	if n := strings.Count(log, "# test log\n"); n != 1 || !strings.Contains(log, "getenv "+logProbeInput+"\n") {
		t.Errorf("the test log holds %d headers and the guarded child's input %v:\n%s", n,
			strings.Contains(log, logProbeInput), log)
	}
	if _, err := os.Stat(logPath + ".unguarded"); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the unguarded child's own log is left (%v)", err)
	}
}

// selectedUnguarded are the unguarded tests -test.run's first element
// selects (all when it is empty) and -test.skip's does not.
func selectedUnguarded(run, skip string) []string {
	first := func(p string) string { f, _, _ := strings.Cut(p, "/"); return f }
	var out []string
	for _, name := range unguardedTests {
		if run != "" {
			if re, err := regexp.Compile(first(run)); err != nil || !re.MatchString(name) {
				continue
			}
		}
		if skip != "" {
			if re, err := regexp.Compile(first(skip)); err == nil && re.MatchString(name) {
				continue
			}
		}
		out = append(out, name)
	}
	return out
}

// subPattern is run's pattern below its first element, with its slash, or
// "".
func subPattern(run string) string {
	if _, rest, ok := strings.Cut(run, "/"); ok {
		return "/" + rest
	}
	return ""
}

// checkNoProxy reports whether net/http names a proxy for a request off the
// machine, which the unguarded child must not.
func checkNoProxy() error {
	req, err := http.NewRequest(http.MethodGet, "https://example.com/", nil) //nolint:noctx // never sent
	if err != nil {
		return err
	}
	got, err := http.ProxyFromEnvironment(req)
	if err != nil {
		return fmt.Errorf("net/http's proxy for https://example.com/: %w", err)
	}
	if got != nil {
		return fmt.Errorf("net/http proxies https://example.com/ through %v; want none", got)
	}
	return nil
}

// The guard below is internal/cli's (internal/cli/testmain_test.go), copied.

// netGuardChild marks the process that runs the tests behind the guard.
const netGuardChild = "GOBBLE_TEST_NETGUARD"

// proxyVars are the variables net/http reads for a proxy.
var proxyVars = []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"}

// runBehindNetGuard starts the guard, runs this binary again with the same
// arguments and extra, the proxy variables naming the guard and netGuardChild set,
// and returns the child's exit code, or 1 when the guard recorded a host,
// which it names.
func runBehindNetGuard(extra ...string) int {
	guard, err := startNetGuard()
	if err != nil {
		fmt.Fprintf(os.Stderr, "network guard: %v\n", err) //nolint:forbidigo // the test binary's own stderr
		return 2
	}
	cmd := exec.CommandContext(context.Background(), os.Args[0], append(slices.Clone(os.Args[1:]), extra...)...) //nolint:gosec // G204: this test binary, run again with its own arguments
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
