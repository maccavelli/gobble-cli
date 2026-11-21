//go:build (linux || darwin) && pty

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
//
// (2026-11-21, MADR D22, Q26: moved word for word from pty_linux_test.go to
// this file, which builds on Linux and macOS; the openers are in
// pty_linux_test.go and pty_darwin_test.go. The cases after this move are
// D22's: A40 split, A43, A44, A45, A46 and A55.)

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/internal/cli/editor"
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
	{name: "pty-abandoned-empty-typed", want: "none-lost", drive: ptyAbandonedEmptyTyped},
	{name: "pty-abandoned-empty-pasted", want: "none-lost", drive: ptyAbandonedEmptyPasted},
	{name: "pty-menu-ctrlc-typed", want: "no-ab", drive: ptyMenuCtrlCTyped},
	{name: "pty-menu-ctrld", want: "ended-at-prompt", drive: ptyMenuCtrlD},
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
	if strings.HasPrefix(name, "pty-abandoned-empty") {
		name = "pty-abandoned" // the same stubbed browser sign-in
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

// ptyAbandonedEmptyTyped is A40 with hello, Enter typed a key at a time.
func ptyAbandonedEmptyTyped(g *ttyRun) string { return ptyAbandonedEmptyAs(g, true) }

// ptyAbandonedEmptyPasted is A40 with hello, Enter written at once.
func ptyAbandonedEmptyPasted(g *ttyRun) string { return ptyAbandonedEmptyAs(g, false) }

// ptyAbandonedEmptyAs (A40; MADR D21, row R102): as pty-abandoned, with
// nothing typed at the paste prompt before the stub's loopback ends it; then
// hello, Enter at gobble's prompt; then again, Enter. The abandoned read still
// waits when gobble's prompt puts the terminal in raw mode, so the first key
// typed there completes it and is dropped with it: a turn ello, and a turn
// again. none-lost names a turn hello; other-turns any other turns.
// (2026-11-21, MADR D22: no read waits, so nothing is lost: none-lost is the
// pass, typed a key at a time or written at once.)
// Typed, hello is typed a key at a time, 150 ms apart.
func ptyAbandonedEmptyAs(g *ttyRun, typed bool) string {
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
	if !g.waitFor(ttyLoopback, 10*time.Second) {
		return "no-loopback"
	}
	g.pause(700 * time.Millisecond)
	if typed {
		for _, k := range []string{"h", "e", "l", "l", "o", "\r"} {
			g.keys(k)
			g.pause(150 * time.Millisecond)
		}
	} else {
		g.keys("hello\r")
	}
	g.pause(1500 * time.Millisecond)
	g.keys("again\r")
	g.pause(1500 * time.Millisecond)
	g.keys("/exit\r")
	r := g.result(10 * time.Second)
	if !r.done {
		return "not-ended"
	}
	var texts []string
	for _, p := range r.prompts {
		for _, m := range regexp.MustCompile(`"text":"([^"]*)"`).FindAllStringSubmatch(p, -1) {
			texts = append(texts, m[1])
		}
	}
	switch {
	case slices.Contains(texts, "hello"):
		return "none-lost"
	case !slices.Contains(texts, "again"):
		return "no-turn-again"
	case slices.Contains(texts, "ello"):
		return "first-key-dropped"
	}
	return "other-turns"
}

// ptyMenuCtrlCTyped (A44; MADR D22, row R133): the first run; y, Enter; ab
// typed at the provider menu, then Ctrl+C: the round is cancelled, and no
// turn or answer holds ab, which the terminal empties with the interrupt.
func ptyMenuCtrlCTyped(g *ttyRun) string {
	if !g.waitFor(ttyOfferFirst, 20*time.Second) {
		return "no-offer"
	}
	g.keys("y\r")
	if !g.waitFor(ttyMenuPrompt, 5*time.Second) {
		return "no-menu"
	}
	g.pause(500 * time.Millisecond)
	g.keys("ab")
	g.pause(300 * time.Millisecond)
	g.keys("\x03")
	g.pause(1500 * time.Millisecond)
	if g.seen(ttyOfferRepeat) {
		g.keys("n\r")
		g.pause(time.Second)
	}
	g.keys("x\r")
	g.pause(1500 * time.Millisecond)
	g.keys("/exit\r")
	r := g.result(10 * time.Second)
	switch {
	case !r.done:
		return "not-ended"
	case r.turnHolds("ab"):
		return "ab-reached"
	}
	return "no-ab"
}

// ptyMenuCtrlD (A45, as moved; MADR D22, row R134): the first run; y,
// Enter; Ctrl+D on the provider menu's empty line, read out of raw mode:
// the round is cancelled, no repeated offer comes, and gobble exits 0 at
// the editor's first read, the input having ended.
func ptyMenuCtrlD(g *ttyRun) string {
	if !g.waitFor(ttyOfferFirst, 20*time.Second) {
		return "no-offer"
	}
	g.keys("y\r")
	if !g.waitFor(ttyMenuPrompt, 5*time.Second) {
		return "no-menu"
	}
	g.pause(500 * time.Millisecond)
	g.keys("\x04")
	r := g.result(10 * time.Second)
	switch {
	case !r.done:
		return "not-ended"
	case g.seen(ttyOfferRepeat):
		return "offer-again"
	case r.exit != "0":
		return "exit-" + r.exit
	}
	return "ended-at-prompt"
}

// ptyWord prints a case's line and fails the test when it is not want.
func ptyWord(t *testing.T, name, want, observed string) {
	t.Helper()
	result := "pass"
	if observed != want {
		result = "fail"
		t.Errorf("%s: observed %s, want %s", name, observed, want)
	}
	fmt.Printf("0063 case=%s result=%s observed=%s\n", name, result, observed)
}

// readyWithin reports whether fd is ready to read within d.
func readyWithin(fd int, d time.Duration) bool {
	var s unix.FdSet
	s.Zero()
	s.Set(fd)
	tv := unix.NsecToTimeval(d.Nanoseconds())
	n, err := unix.Select(fd+1, &s, nil, nil, &tv)
	return err == nil && n > 0 && s.IsSet(fd)
}

// TestPtyGaveUpHeld (A43, pty-gaveup-held; MADR D22, row R130): a line
// read through the layer is given up by the person with partial typed; raw
// mode then goes on; 200 ms later the terminal still holds the bytes and
// the layer made no read: nothing was waiting to take them. It does not
// depend on timing: a layer that left a read waiting would take them.
func TestPtyGaveUpHeld(t *testing.T) {
	m, s, err := openPty()
	if err != nil {
		ptyWord(t, "pty-gaveup-held", "held", "no-pseudo-terminal")
		return
	}
	defer m.Close()
	defer s.Close()
	var reads readCount
	g, err := term.OpenGated(s, term.GatedOptions{Read: reads.read})
	if err != nil {
		ptyWord(t, "pty-gaveup-held", "held", "no-layer")
		return
	}
	defer g.Stop()
	ctx, cancel := context.WithCancel(t.Context())
	q := &questions{ctx: ctx, in: gatedInput(g, nil), out: io.Discard}
	done := make(chan error, 1)
	go func() { _, err := q.line(); done <- err }()
	time.Sleep(200 * time.Millisecond)
	_, _ = m.WriteString("partial")
	time.Sleep(300 * time.Millisecond)
	cancel() // the person's give-up: nothing is flushed
	<-done
	restore, err := editor.Raw(s)
	if err != nil {
		ptyWord(t, "pty-gaveup-held", "held", "no-raw-mode")
		return
	}
	defer restore() //nolint:errcheck // the test's own terminal
	time.Sleep(200 * time.Millisecond)
	fd := int(s.Fd()) //nolint:gosec // a descriptor fits an int
	observed := "gone"
	switch {
	case reads.n.Load() > 0:
		observed = "taken"
	case readyWithin(fd, 100*time.Millisecond):
		observed = "held"
	}
	if q := queued(fd); observed == "held" && q >= 0 && q != len("partial") {
		observed = fmt.Sprintf("held-%d", q)
	}
	ptyWord(t, "pty-gaveup-held", "held", observed)
}

// Environment of TestPtyPrivateChild.
const (
	ptyPrivateEnv    = "GOBBLE_PTY_PRIVATE"
	ptyPrivateResult = "GOBBLE_PTY_PRIVATE_RESULT"
)

// TestPtyPrivateDescriptor (A55, with A46's terminal part; MADR D22, rows
// R124, R126, R146): on a pseudo terminal, the layer reads a descriptor of
// its own, non-blocking and close-on-exec, standard input stays blocking,
// terminalIn chooses the layer, and what is typed is read; then, in a child
// whose controlling terminal it is, an open of the path that fails falls
// back to /dev/tty, which reads what is typed. (2026-11-21, MADR D23: and
// pty-not-controlling, A59, and pty-same-terminal, A64.)
func TestPtyPrivateDescriptor(t *testing.T) {
	m, s, err := openPty()
	if err != nil {
		ptyWord(t, "pty-private-descriptor", "private", "no-pseudo-terminal")
		return
	}
	defer m.Close()
	defer s.Close()
	ptyWord(t, "pty-private-descriptor", "private", privateDescriptor(t, m, s))
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ptyWord(t, "pty-dev-tty-fallback", "dev-tty", devTTYFallback(t, exe))
	ptyWord(t, "pty-not-controlling", "refused", notControlling(t, exe))
	ptyWord(t, "pty-same-terminal", "own-terminal", sameTerminalOnly(t, exe))
}

func privateDescriptor(t *testing.T, m, s *os.File) string {
	g, err := term.OpenGated(s, term.GatedOptions{})
	if err != nil {
		return "no-layer"
	}
	defer g.Stop()
	fd, sfd := g.Descriptor(), int(s.Fd()) //nolint:gosec // a descriptor fits an int
	fl, err1 := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	fdfl, err2 := unix.FcntlInt(uintptr(fd), unix.F_GETFD, 0)
	sfl, err3 := unix.FcntlInt(uintptr(sfd), unix.F_GETFL, 0)
	switch {
	case fd == sfd:
		return "standard-input"
	case err1 != nil || fl&unix.O_NONBLOCK == 0:
		return "blocking"
	case err2 != nil || fdfl&unix.FD_CLOEXEC == 0:
		return "not-close-on-exec"
	case err3 != nil || sfl&unix.O_NONBLOCK != 0:
		return "standard-input-non-blocking"
	}
	out := &Output{out: io.Discard, err: io.Discard, caps: term.Caps{In: true}}
	e := &runEnv{ctx: t.Context(), out: out, logs: &lazyLog{out: out}, stdin: s}
	defer e.logs.close() //nolint:errcheck // a test's log
	if ti := e.terminalIn(); ti.gate == nil {
		return "no-gate-chosen"
	}
	g.Open()
	got := make(chan string, 1)
	go func() {
		b := make([]byte, 16)
		n, _ := g.Read(b)
		got <- string(b[:n])
	}()
	_, _ = m.WriteString("q\n")
	select {
	case r := <-got:
		if r != "q\n" {
			return "read-" + strconv.Quote(r)
		}
	case <-time.After(3 * time.Second):
		return "no-read"
	}
	return "private"
}

// devTTYFallback starts this test binary on a new pseudo terminal as its
// controlling one, where an open of a path that does not exist falls back
// to /dev/tty, and types q, Enter for it to read.
func devTTYFallback(t *testing.T, exe string) string {
	switch w := privateChild(t, exe, nil, ""); w {
	case "read":
		return "dev-tty"
	default:
		return w
	}
}

// notControlling (A59; MADR D22, M15; row R126): the child's controlling
// terminal is one pseudo terminal and its standard input another's slave,
// whose path fails: /dev/tty is not standard input's terminal, so the open
// is refused (row R138), never /dev/tty.
func notControlling(t *testing.T, exe string) string {
	m, s, err := openPty()
	if err != nil {
		return "no-pseudo-terminal"
	}
	defer m.Close()
	defer s.Close()
	switch w := privateChild(t, exe, s, ""); w {
	case "open-failed":
		return "refused"
	case "read", "no-read":
		return "dev-tty-opened"
	default:
		return w
	}
}

// sameTerminalOnly (A64; MADR D23, row R155): the child's controlling
// terminal and standard input are one pseudo terminal, and it is given the
// path of another's slave: that terminal is not standard input's, so the
// layer goes on to /dev/tty, and what is typed on the child's own terminal
// is read.
func sameTerminalOnly(t *testing.T, exe string) string {
	m, s, err := openPty()
	if err != nil {
		return "no-pseudo-terminal"
	}
	defer m.Close()
	defer s.Close()
	switch w := privateChild(t, exe, nil, s.Name()); w {
	case "read":
		return "own-terminal"
	case "no-read":
		return "other-terminal"
	default:
		return w
	}
}

// privateChild starts TestPtyPrivateChild on a new pseudo terminal as its
// controlling one; its standard input is that terminal, or in when in is
// not nil; path, when not "", is the path it gives OpenGated, else one that
// does not exist. Once it has opened, q, Enter is typed on its controlling
// terminal. Its word: open-failed, read (it read what was typed), no-read,
// or what kept it from starting.
func privateChild(t *testing.T, exe string, in *os.File, path string) string {
	m, s, err := openPty()
	if err != nil {
		return "no-pseudo-terminal"
	}
	defer m.Close()
	result := filepath.Join(t.TempDir(), "result")
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestPtyPrivateChild$", "-test.count=1")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = s, s, s
	attr := &syscall.SysProcAttr{Setsid: true, Setctty: true}
	if in != nil {
		cmd.Stdin = in
		attr.Ctty = 1 // the controlling terminal is standard output's
	}
	cmd.SysProcAttr = attr
	cmd.Env = append(os.Environ(), ptyPrivateEnv+"=1", ptyPrivateResult+"="+result, ptyPrivatePath+"="+path)
	if err := cmd.Start(); err != nil {
		s.Close()
		return "no-child"
	}
	s.Close()
	go func() { _, _ = io.Copy(io.Discard, m) }()
	exited := make(chan struct{})
	go func() {
		_ = cmd.Wait()
		close(exited)
	}()
	read := func() string { b, _ := os.ReadFile(result); return string(b) }
	deadline := time.Now().Add(15 * time.Second)
	for !strings.Contains(read(), "open ") && time.Now().Before(deadline) {
		time.Sleep(25 * time.Millisecond)
	}
	if !strings.Contains(read(), "open ok") {
		cancel()
		<-exited
		return "open-failed"
	}
	_, _ = m.WriteString("z\n")
	select {
	case <-exited:
	case <-time.After(5 * time.Second):
		cancel() // it reads another terminal, where nothing is typed
		<-exited
	}
	if !strings.Contains(read(), "read \"z\\n\"") {
		return "no-read"
	}
	return "read"
}

// ptyPrivatePath is the path privateChild's child gives OpenGated.
const ptyPrivatePath = "GOBBLE_PTY_PRIVATE_PATH"

// TestPtyPrivateChild is privateChild's child.
func TestPtyPrivateChild(t *testing.T) {
	if os.Getenv(ptyPrivateEnv) == "" {
		t.Skip("run by TestPtyPrivateDescriptor on a pseudo terminal")
	}
	path := os.Getenv(ptyPrivateResult)
	write := func(s string) {
		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = f.WriteString(s + "\n")
			_ = f.Close()
		}
	}
	open := os.Getenv(ptyPrivatePath)
	if open == "" {
		open = filepath.Join(t.TempDir(), "no-such-terminal")
	}
	g, err := term.OpenGated(os.Stdin, term.GatedOptions{Path: open})
	if err != nil {
		write("open failed")
		return
	}
	defer g.Stop()
	write("open ok")
	g.Open()
	b := make([]byte, 16)
	n, _ := g.Read(b)
	write("read " + strconv.Quote(string(b[:n])))
}

// The rest is the same in conpty_windows_test.go, which builds on another
// system: each file keeps its own copy, from here to the end. (2026-11-21:
// except ttyAbandoned's check, which on POSIX allows hello, MADR D21.)
// (2026-11-21, MADR D22: and requires it, A43.)

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
// again, Enter. No turn may hold partial; again is one. (2026-11-21, MADR
// D21: on POSIX partial completes the abandoned read as raw mode goes on, and
// hello, typed at gobble's prompt, may be a turn.)
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
	case r.turnHolds("partial"):
		return "reached-turn"
	case !r.turnHolds("again"):
		return "no-turn-again"
	case !r.turnHolds("hello"): // (2026-11-21, MADR D22, A43: nothing typed at gobble's prompt is lost)
		return "hello-lost"
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
