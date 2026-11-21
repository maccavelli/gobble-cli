//go:build linux && pty

package cli

// The pseudo-terminal cases (0063-PLAN P1b and its amendments; MADR
// Confirmation item 5 (A), D12, D13, D19): pty-flush-prompt,
// pty-flush-offer, pty-abandoned and pty-ctrlc-ctrld, the POSIX twins of
// conpty_windows_test.go's cases. TestPtyCases opens a pseudo-terminal
// (golang.org/x/sys/unix, already required) and starts this test binary
// again on it, as a session leader with the terminal as its controlling
// one, running TestPtyChild, which runs Main with the stubs a case names,
// over a scripted agent whose prompts it writes to a result file. The child
// inherits this process's environment, the network guard's proxy among
// them, so nothing leaves the host, and GOBBLE_HOME is a temporary
// directory. The screen is read in memory and never printed: each case
// prints only its line,
//
//	0063 case=<name> result=<pass|fail> observed=<word>
//
// where result compares the observed word with the case's word once P2b
// is built. Run it with `make probe-pty` on the Linux machine. Not part of
// preflight or CI.

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

const (
	ptyCaseEnv   = "GOBBLE_PTY_CASE"   // the case TestPtyChild runs
	ptyResultEnv = "GOBBLE_PTY_RESULT" // where it writes its result
)

// ptyCase is one case: how it is driven, and its word once P2b is built.
type ptyCase struct {
	name  string
	want  string
	drive func(g *ttyRun) string
}

var ptyCases = []ptyCase{
	{name: "pty-flush-prompt", want: "no-turn", drive: ttyFlushPrompt},
	{name: "pty-flush-offer", want: "offer-waits", drive: ttyFlushOffer},
	{name: "pty-abandoned", want: "dropped", drive: ttyAbandoned},
	{name: "pty-ctrlc-ctrld", want: "warned-once", drive: ptyCtrlCCtrlD},
}

func TestPtyCases(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range ptyCases {
		observed := runPtyCase(t, exe, c)
		result := "pass"
		if observed != c.want {
			result = "fail"
			t.Errorf("%s: observed %s, want %s", c.name, observed, c.want)
		}
		fmt.Printf("0063 case=%s result=%s observed=%s\n", c.name, result, observed)
	}
}

// openPty opens a pseudo-terminal's master and its slave, 120 by 30.
func openPty() (master, slave *os.File, _ error) {
	m, err := os.OpenFile("/dev/ptmx", os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		return nil, nil, err
	}
	fd := int(m.Fd()) //nolint:gosec // a descriptor fits an int
	if err := unix.IoctlSetPointerInt(fd, unix.TIOCSPTLCK, 0); err != nil {
		m.Close()
		return nil, nil, fmt.Errorf("unlock the pseudo-terminal: %w", err)
	}
	n, err := unix.IoctlGetInt(fd, unix.TIOCGPTN)
	if err != nil {
		m.Close()
		return nil, nil, fmt.Errorf("the pseudo-terminal's number: %w", err)
	}
	s, err := os.OpenFile("/dev/pts/"+strconv.Itoa(n), os.O_RDWR|syscall.O_NOCTTY, 0)
	if err != nil {
		m.Close()
		return nil, nil, err
	}
	if err := unix.IoctlSetWinsize(int(s.Fd()), unix.TIOCSWINSZ, &unix.Winsize{Row: 30, Col: 120}); err != nil { //nolint:gosec // a descriptor fits an int
		m.Close()
		s.Close()
		return nil, nil, fmt.Errorf("the pseudo-terminal's size: %w", err)
	}
	return m, s, nil
}

func runPtyCase(t *testing.T, exe string, c ptyCase) string {
	t.Helper()
	m, s, err := openPty()
	if err != nil {
		t.Errorf("%s: open a pseudo-terminal: %v", c.name, err)
		return "no-pseudo-terminal"
	}
	home := t.TempDir()
	resultPath := filepath.Join(t.TempDir(), "result")
	t.Setenv("GOBBLE_HOME", home)
	t.Setenv("TERM", "xterm-256color")
	t.Setenv(ptyCaseEnv, c.name)
	t.Setenv(ptyResultEnv, resultPath)
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestPtyChild$", "-test.count=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = s, s, s
	cmd.Dir = t.TempDir()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if err := cmd.Start(); err != nil {
		m.Close()
		s.Close()
		t.Errorf("%s: start the child on the pseudo-terminal: %v", c.name, err)
		return "no-child"
	}
	s.Close()
	screen := &ttyScreen{}
	drained := make(chan struct{})
	go func() {
		defer close(drained)
		_, _ = io.Copy(screen, m) // ends with EIO once the child has gone
	}()
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	word := c.drive(&ttyRun{screen: screen, keysTo: m, exited: exited, ctx: ctx, home: home, resultPath: resultPath})
	cancel()
	<-exited
	select {
	case <-drained:
	case <-time.After(5 * time.Second):
	}
	m.Close()
	return word
}

// TestPtyChild is the child of one case, on the pseudo-terminal.
func TestPtyChild(t *testing.T) {
	name := os.Getenv(ptyCaseEnv)
	if name == "" {
		t.Skip("run by TestPtyCases on a pseudo-terminal")
	}
	ttyChild(t, name, nil, os.Getenv(ptyResultEnv))
}

// ptyCtrlCCtrlD: the first run; y, Enter; Ctrl+C at the provider menu;
// Ctrl+D at the repeated offer: the warning once (row R118).
func ptyCtrlCCtrlD(g *ttyRun) string {
	if !g.waitFor(ttyOfferFirst, 20*time.Second) {
		return "no-offer"
	}
	g.keys("y\r")
	if !g.waitFor(ttyMenuPrompt, 5*time.Second) {
		return "no-menu"
	}
	g.pause(500 * time.Millisecond)
	g.keys("\x03")
	if !g.waitFor(ttyOfferRepeat, 3*time.Second) {
		g.keys("\x04")
		return "no-offer-again"
	}
	g.keys("\x04")
	r := g.result(10 * time.Second)
	if !r.done {
		return "not-ended"
	}
	if n := g.count(ttyWarning); n != 1 {
		return fmt.Sprintf("warned-%d", n)
	}
	return "warned-once"
}

// The rest is the same in conpty_windows_test.go, which builds on another
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
