//go:build windows && conpty

package cli

// The pseudo-console cases that need a stubbed sign-in (0063-PLAN P1b and
// its amendments; MADR Confirmation item 5 (A), D12, D13, D20): flush-prompt,
// flush-offer, abandoned and auth-paste-ctrlc. TestConptyCases creates a
// pseudo console and starts this test binary again inside it, running
// TestConptyChild, which runs Main with the stubs a case names, over a
// scripted agent whose prompts it writes to a result file. The child
// inherits this process's environment, the network guard's proxy among
// them, so nothing leaves the host, and GOBBLE_HOME is a temporary
// directory. The screen is read in memory and never printed: each case
// prints only its line,
//
//	0063 case=<name> result=<pass|fail> observed=<word>
//
// where result compares the observed word with the case's word once P2b
// is built. Run it with `make probe-conpty` on the Windows test laptop. Not
// part of preflight or CI.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

const (
	conptyCaseEnv   = "GOBBLE_CONPTY_CASE"   // the case TestConptyChild runs
	conptyResultEnv = "GOBBLE_CONPTY_RESULT" // where it writes its result
)

var conptySetCtrlHandler = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleCtrlHandler")

// conptyCase is one case: what it runs, how it is driven, and its word
// once P2b is built.
type conptyCase struct {
	name  string
	want  string
	drive func(g *ttyRun) string
}

var conptyCases = []conptyCase{
	{name: "flush-prompt", want: "no-turn", drive: ttyFlushPrompt},
	{name: "flush-offer", want: "offer-waits", drive: ttyFlushOffer},
	{name: "abandoned", want: "dropped", drive: ttyAbandoned},
	{name: "auth-paste-ctrlc", want: "ended", drive: func(g *ttyRun) string {
		if !g.waitFor(ttyPaste, 20*time.Second) {
			return "no-paste-prompt"
		}
		g.pause(time.Second)
		g.keys("\x03")
		r := g.result(10 * time.Second)
		switch {
		case !r.done:
			return "not-ended"
		case r.exit != strconv.Itoa(exitForSignal(os.Interrupt)):
			return "exit-" + r.exit
		case authSaved(g.home):
			return "saved"
		}
		return "ended"
	}},
}

// conptyArgs are the gobble arguments of each case; none is the line
// session.
var conptyArgs = map[string][]string{
	"auth-paste-ctrlc": {"auth", "login", "grok", "--method", "browser"},
}

func TestConptyCases(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range conptyCases {
		observed := runConptyCase(t, exe, c)
		result := "pass"
		if observed != c.want {
			result = "fail"
			t.Errorf("%s: observed %s, want %s", c.name, observed, c.want)
		}
		fmt.Printf("0063 case=%s result=%s observed=%s\n", c.name, result, observed)
	}
}

func runConptyCase(t *testing.T, exe string, c conptyCase) string {
	t.Helper()
	pc, err := newConptyConsole()
	if err != nil {
		t.Errorf("%s: create a pseudo console: %v", c.name, err)
		return "no-pseudo-console"
	}
	defer pc.close()
	home := t.TempDir()
	resultPath := filepath.Join(t.TempDir(), "result")
	t.Setenv("GOBBLE_HOME", home)
	t.Setenv(conptyCaseEnv, c.name)
	t.Setenv(conptyResultEnv, resultPath)
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	exited, err := pc.start(ctx, t.TempDir(), exe, "-test.run=^TestConptyChild$", "-test.count=1")
	if err != nil {
		t.Errorf("%s: start the child in the pseudo console: %v", c.name, err)
		return "no-child"
	}
	word := c.drive(&ttyRun{screen: pc, keysTo: pc.in, exited: exited, ctx: ctx, home: home, resultPath: resultPath})
	cancel()
	<-exited
	return word
}

// TestConptyChild is the child of one case, inside the pseudo console.
func TestConptyChild(t *testing.T) {
	name := os.Getenv(conptyCaseEnv)
	if name == "" {
		t.Skip("run by TestConptyCases inside a pseudo console")
	}
	// A program started from a shell in a console window takes Ctrl+C; one
	// started by a process that ignores it would not (0063-PLAN, P1 step 0b
	// build pass 1).
	conptySetCtrlHandler.Call(0, 0)
	ttyChild(t, name, conptyArgs[name], os.Getenv(conptyResultEnv))
}

// conptyConsole is a pseudo console and the parent's ends of its pipes.
type conptyConsole struct {
	h       windows.Handle
	in      *os.File
	out     *os.File
	drained chan struct{}
	ttyScreen
}

func newConptyConsole() (*conptyConsole, error) {
	var inR, inW, outR, outW windows.Handle
	if err := windows.CreatePipe(&inR, &inW, nil, 0); err != nil {
		return nil, fmt.Errorf("input pipe: %w", err)
	}
	if err := windows.CreatePipe(&outR, &outW, nil, 0); err != nil {
		windows.CloseHandle(inR)
		windows.CloseHandle(inW)
		return nil, fmt.Errorf("output pipe: %w", err)
	}
	var h windows.Handle
	err := windows.CreatePseudoConsole(windows.Coord{X: 120, Y: 30}, inR, outW, 0, &h)
	windows.CloseHandle(inR)
	windows.CloseHandle(outW)
	if err != nil {
		windows.CloseHandle(inW)
		windows.CloseHandle(outR)
		return nil, fmt.Errorf("CreatePseudoConsole: %w", err)
	}
	pc := &conptyConsole{h: h, in: os.NewFile(uintptr(inW), "conpty-in"), out: os.NewFile(uintptr(outR), "conpty-out"),
		drained: make(chan struct{})}
	go func() {
		defer close(pc.drained)
		_, _ = io.Copy(&pc.ttyScreen, pc.out)
	}()
	return pc, nil
}

func (pc *conptyConsole) close() {
	windows.ClosePseudoConsole(pc.h)
	pc.in.Close()
	select {
	case <-pc.drained:
	case <-time.After(5 * time.Second):
	}
	pc.out.Close()
}

// start runs argv attached to the pseudo console, in dir, as
// cmd/gobble's conpty test does.
func (pc *conptyConsole) start(ctx context.Context, dir string, argv ...string) (<-chan struct{}, error) {
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, fmt.Errorf("attribute list: %w", err)
	}
	defer attrs.Delete()
	hpc := *(*unsafe.Pointer)(unsafe.Pointer(&pc.h))
	if err := attrs.Update(windows.PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE, hpc, unsafe.Sizeof(pc.h)); err != nil {
		return nil, fmt.Errorf("PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE: %w", err)
	}
	si := windows.StartupInfoEx{ProcThreadAttributeList: attrs.List()}
	si.Cb = uint32(unsafe.Sizeof(si))
	si.Flags = windows.STARTF_USESTDHANDLES
	cl, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(argv))
	if err != nil {
		return nil, err
	}
	cwd, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return nil, err
	}
	var pi windows.ProcessInformation
	if err := windows.CreateProcess(nil, cl, nil, nil, false, windows.EXTENDED_STARTUPINFO_PRESENT, nil, cwd, &si.StartupInfo, &pi); err != nil {
		return nil, fmt.Errorf("CreateProcess: %w", err)
	}
	windows.CloseHandle(pi.Thread)
	exited := make(chan struct{})
	go func() {
		defer windows.CloseHandle(pi.Process)
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				windows.TerminateProcess(pi.Process, 1)
			case <-done:
			}
		}()
		windows.WaitForSingleObject(pi.Process, windows.INFINITE)
		close(done)
		close(exited)
	}()
	return exited, nil
}

// authSaved reports whether auth.json under home holds any provider's
// entry, or cannot be read.
func authSaved(home string) bool {
	b, err := os.ReadFile(filepath.Join(home, "auth.json"))
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	if err != nil {
		return true
	}
	for _, id := range provider.KeyProviders() {
		if strings.Contains(string(b), strconv.Quote(id)) {
			return true
		}
	}
	return false
}

// The rest is the same in pty_linux_test.go, which builds on another
// system: each file keeps its own copy, from here to the end.

// Words a case looks for on the screen, as ttyPlain gives them.
var (
	ttyOfferFirst  = ttyPlain("Sign in to one now? [Y/n]:")
	ttyOfferRepeat = ttyPlain("Sign in to one now? [y/N]:")
	ttyMenu        = ttyPlain("Sign in to which provider?")
	ttyMenuPrompt  = ttyPlain("Number [1]:")
	ttyWarning     = ttyPlain("Warning: no model credential")
	ttyPaste       = ttyPlain("Paste the redirect URL:")
	ttyLoopback    = ttyPlain("the stubbed loopback finished first")
)

var ttyEscapes = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)|\x1b[()][0-9A-Za-z]|\x1b[@-Z\\-_=>]`)

// ttyPlain is s without escapes, control characters or white space.
func ttyPlain(s string) string {
	s = ttyEscapes.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// ttyScreen keeps what the terminal showed, in memory only.
type ttyScreen struct {
	mu  sync.Mutex
	buf []byte
}

func (s *ttyScreen) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.buf = append(s.buf, p...)
	return len(p), nil
}

func (s *ttyScreen) text() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ttyPlain(string(s.buf))
}

// ttyRun is one case's child, its screen and its keys.
type ttyRun struct {
	screen interface{ text() string }
	keysTo io.Writer
	exited <-chan struct{}
	ctx    context.Context
	home   string
	// resultPath is where the child writes its result.
	resultPath string
}

func (g *ttyRun) keys(s string) { _, _ = io.WriteString(g.keysTo, s) }

func (g *ttyRun) seen(s string) bool { return strings.Contains(g.screen.text(), s) }

func (g *ttyRun) count(s string) int { return strings.Count(g.screen.text(), s) }

func (g *ttyRun) gone() bool {
	select {
	case <-g.exited:
		return true
	default:
		return false
	}
}

func (g *ttyRun) pause(d time.Duration) {
	select {
	case <-time.After(d):
	case <-g.ctx.Done():
	}
}

func (g *ttyRun) waitFor(s string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) && g.ctx.Err() == nil {
		if g.seen(s) {
			return true
		}
		if g.gone() {
			g.pause(200 * time.Millisecond)
			return g.seen(s)
		}
		time.Sleep(25 * time.Millisecond)
	}
	return g.seen(s)
}

// ttyResult is what the child wrote: Main's exit code, each prompt the
// scripted agent received, and whether it finished.
type ttyResult struct {
	exit    string
	prompts []string
	done    bool
}

func (r ttyResult) turnHolds(s string) bool {
	for _, p := range r.prompts {
		if strings.Contains(p, s) {
			return true
		}
	}
	return false
}

// result waits up to d for the child to end, and reads its result.
func (g *ttyRun) result(d time.Duration) ttyResult {
	select {
	case <-g.exited:
	case <-time.After(d):
	case <-g.ctx.Done():
	}
	var r ttyResult
	b, err := os.ReadFile(g.resultPath)
	if err != nil {
		return r
	}
	for line := range strings.Lines(string(b)) {
		line = strings.TrimRight(line, "\n")
		switch {
		case line == "done":
			r.done = true
		case strings.HasPrefix(line, "exit "):
			r.exit = strings.TrimPrefix(line, "exit ")
		case strings.HasPrefix(line, "prompt "):
			if p, err := strconv.Unquote(strings.TrimPrefix(line, "prompt ")); err == nil {
				r.prompts = append(r.prompts, p)
			}
		}
	}
	return r
}

// ttyFlushPrompt: n, Enter, hello, Enter in one write at the first offer;
// hello must not become a turn.
func ttyFlushPrompt(g *ttyRun) string {
	if !g.waitFor(ttyOfferFirst, 20*time.Second) {
		return "no-offer"
	}
	g.keys("n\rhello\r")
	if !g.waitFor(ttyWarning, 5*time.Second) {
		return "no-warning"
	}
	g.pause(2 * time.Second)
	g.keys("/exit\r")
	r := g.result(10 * time.Second)
	if !r.done {
		return "not-ended"
	}
	if r.turnHolds("hello") {
		return "turn"
	}
	return "no-turn"
}

// ttyFlushOffer: a stubbed round waits 1 s and fails; y, Enter is typed
// during it; the repeated offer must be shown and still wait.
func ttyFlushOffer(g *ttyRun) string {
	if !g.waitFor(ttyOfferFirst, 20*time.Second) {
		return "no-offer"
	}
	g.keys("y\r")
	if !g.waitFor(ttyMenuPrompt, 5*time.Second) {
		return "no-menu"
	}
	g.keys("1\r")
	g.pause(300 * time.Millisecond)
	g.keys("y\r") // while the round waits
	if !g.waitFor(ttyOfferRepeat, 5*time.Second) {
		return "no-offer-again"
	}
	g.pause(1500 * time.Millisecond)
	if g.count(ttyMenu) != 1 {
		return "offer-answered"
	}
	return "offer-waits"
}

// ttyAbandoned: a stubbed browser sign-in's paste prompt is given partial,
// then the stub's loopback ends it; hello, Enter at gobble's prompt; then
// again, Enter. No turn may hold partial or hello; again is one.
func ttyAbandoned(g *ttyRun) string {
	if !g.waitFor(ttyOfferFirst, 20*time.Second) {
		return "no-offer"
	}
	g.keys("n\r")
	if !g.waitFor(ttyWarning, 5*time.Second) {
		return "no-warning"
	}
	g.pause(700 * time.Millisecond)
	g.keys("/login grok\r")
	if !g.waitFor(ttyPaste, 5*time.Second) {
		return "no-paste-prompt"
	}
	g.pause(300 * time.Millisecond)
	g.keys("partial")
	if !g.waitFor(ttyLoopback, 10*time.Second) {
		return "no-loopback"
	}
	g.pause(700 * time.Millisecond)
	g.keys("hello\r")
	g.pause(1500 * time.Millisecond)
	g.keys("again\r")
	g.pause(1500 * time.Millisecond)
	g.keys("/exit\r")
	r := g.result(10 * time.Second)
	switch {
	case !r.done:
		return "not-ended"
	case r.turnHolds("partial") || r.turnHolds("hello"):
		return "reached-turn"
	case !r.turnHolds("again"):
		return "no-turn-again"
	}
	return "dropped"
}

// errLoopbackFirst is the abandoned case's stubbed loopback, which ends
// the paste prompt's read.
var errLoopbackFirst = errors.New("the stubbed loopback finished first")

// ttyChild runs Main for case name with args, over a scripted agent, with
// the case's stubbed sign-in, and writes its exit code and each prompt the
// agent received to resultPath.
func ttyChild(t *testing.T, name string, args []string, resultPath string) {
	agent := &acptest.ScriptAgent{Turns: make([]acptest.Turn, 8)}
	withAgent(t, agent)
	switch strings.TrimPrefix(name, "pty-") {
	case "flush-offer":
		replace(t, &signInWith, func(context.Context, string, provider.Prompter, provider.SignInOptions, provider.LoginStore) (provider.Credential, error) {
			time.Sleep(time.Second) // a round that reads nothing, then fails
			return provider.Credential{}, errors.New("the stubbed round failed")
		})
	case "abandoned":
		replace(t, &signInWith, func(ctx context.Context, _ string, p provider.Prompter, _ provider.SignInOptions, _ provider.LoginStore) (provider.Credential, error) {
			loop, cancel := context.WithCancel(ctx)
			defer cancel()
			go func() {
				time.Sleep(2500 * time.Millisecond) // the loopback arrives first
				cancel()
			}()
			_, _ = p.Input(loop, "Paste the redirect URL")
			return provider.Credential{}, errLoopbackFirst
		})
	case "auth-paste-ctrlc":
		replace(t, &signInWith, func(ctx context.Context, _ string, p provider.Prompter, _ provider.SignInOptions, _ provider.LoginStore) (provider.Credential, error) {
			_, err := p.Input(ctx, "Paste the redirect URL")
			return provider.Credential{}, cmpErr(err, errNoSignIn)
		})
	}
	restore, err := term.PrepareConsole(os.Stdout, os.Stderr)
	if err != nil {
		t.Logf("prepare the console: %v", err)
	}
	code := Main(t.Context(), args, os.Stdin, os.Stdout, os.Stderr)
	restore()
	var b strings.Builder
	fmt.Fprintf(&b, "exit %d\n", code)
	for _, p := range agent.Prompts() {
		fmt.Fprintf(&b, "prompt %s\n", strconv.Quote(p))
	}
	b.WriteString("done\n")
	if err := os.WriteFile(resultPath, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
}

// cmpErr is err, or def when err is nil.
func cmpErr(err, def error) error {
	if err != nil {
		return err
	}
	return def
}
