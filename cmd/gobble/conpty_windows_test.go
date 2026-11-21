//go:build windows && conpty

package main

// The pseudo-console test (0063-PLAN P1b, P1 step 0b;
// 0063-MADR-one-shared-terminal-input-so-the-key-prompt-waits.md,
// Confirmation item 5 (A)). The parent creates a pseudo console
// (CreatePseudoConsole) and starts this test binary again inside it, through
// PROC_THREAD_ATTRIBUTE_PSEUDOCONSOLE. TestMain runs the child mode an
// environment variable names. The parent types into the pseudo console's
// input pipe and drains its output into memory, where a gobble-mode case
// reads it; nothing of the screen is printed or kept after the case. A
// probe child writes what its reads gave to a file the parent names, and
// the parent prints one line per case:
//
//	0063 case=<name> result=<pass|fail> observed=<word>
//
// Probe mode measures the console itself (MADR D6, D14, rows R34, R35, R40,
// R90): each case is one or two reads of standard input, in the console's
// default line mode or, for probe-ctrlz-raw, in the raw mode gobble's secret
// uses. Before any read the child takes os.Interrupt with signal.Notify, as
// gobble does (internal/cli/main.go), and writes "signal" when it arrives.
//
// Gobble mode runs gobble's main in the pseudo console (TestConptyGobble,
// below). Run both with `make probe-conpty`, the probes alone with
// `make probe-conpty PROBE=1`, on the Windows test laptop. Not part of
// preflight or CI.

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/maccavelli/gobble-cli/internal/auth"
	"github.com/maccavelli/gobble-cli/internal/cli"
	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

const (
	probeEnv  = "GOBBLE_CONPTY_PROBE"  // the probe case a child runs
	gobbleEnv = "GOBBLE_CONPTY_GOBBLE" // "run:" and gobble's arguments, \x1f between them
	dirEnv    = "GOBBLE_CONPTY_DIR"    // where a child writes its result

	resultFile = "result" // the child's lines: ready, signal, read …, done
	goFile     = "go"     // the parent's word that the child may read

	startDeadline = 15 * time.Second // the child's start, to "ready"
	caseDeadline  = 30 * time.Second // a whole case
	settle        = time.Second      // for keys to reach the console, or a read to start
)

var procSetConsoleCtrlHandler = windows.NewLazySystemDLL("kernel32.dll").NewProc("SetConsoleCtrlHandler")

// TestMain runs a child mode when the parent started this binary: probe,
// or gobble, which runs gobble's main with the arguments it names.
func TestMain(m *testing.M) {
	if name := os.Getenv(probeEnv); name != "" {
		os.Exit(probeChild(name, os.Getenv(dirEnv)))
	}
	if args, ok := strings.CutPrefix(os.Getenv(gobbleEnv), "run:"); ok {
		os.Exit(gobbleChild(args))
	}
	os.Exit(m.Run())
}

// probeCase is one measurement: what the parent types and when, the word
// the child's result gives, and the word a pass needs.
type probeCase struct {
	name  string
	want  string
	drive func(d *driver)
	judge func(r childResult) string
}

var probeCases = []probeCase{
	{
		// MADR D6's third assumption, row R34: a line-mode read of 1, Enter.
		name: "probe-crlf", want: "crlf",
		drive: func(d *driver) {
			d.goAhead()
			d.sleep(settle)
			d.keys("1\r")
		},
		judge: func(r childResult) string {
			return r.firstRead(map[string]string{"1\r\n": "crlf", "1\r": "cr", "1\n": "lf"})
		},
	},
	{
		// Row R35: two lines typed before two line-mode reads.
		name: "probe-typeahead", want: "one-line-per-read",
		drive: func(d *driver) {
			d.keys("1\r2\r")
			d.sleep(settle)
			d.goAhead()
			if !d.waitReads(2, 5*time.Second) {
				d.keys("\r") // ends a second read the first one left nothing for
			}
		},
		judge: judgeTypeahead,
	},
	{
		// Row R40: Ctrl+C during a line-mode read, then x, Enter. The raw
		// console ends the read (measured at P1 step 0b; 0063-PLAN Deviation
		// 1): eof is expected, and first= says whether the interrupt or the
		// read's return came first (MADR D20).
		name: "probe-ctrlc", want: "eof",
		drive: func(d *driver) {
			d.goAhead()
			d.sleep(settle)
			d.keys("\x03")
			d.waitLine("signal", 5*time.Second)
			d.sleep(settle / 2)
			d.keys("x\r")
		},
		judge: func(r childResult) string {
			if !r.signal {
				return "no-signal"
			}
			return r.firstRead(map[string]string{"x\r\n": "waiting"})
		},
	},
	{
		// Row R90: Ctrl+Z, Enter in a line-mode read.
		name: "probe-ctrlz-line", want: "eof",
		drive: func(d *driver) {
			d.goAhead()
			d.sleep(settle)
			d.keys("\x1a\r")
		},
		judge: func(r childResult) string { return r.firstRead(nil) },
	},
	{
		// Row R90: Ctrl+Z in a raw read, the secret's mode.
		name: "probe-ctrlz-raw", want: "eof",
		drive: func(d *driver) {
			d.goAhead()
			d.sleep(settle)
			d.keys("\x1a")
		},
		judge: func(r childResult) string { return r.firstRead(nil) },
	},
}

func TestConptyProbe(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range probeCases {
		observed, r := runProbe(t, exe, c)
		result := "pass"
		if observed != c.want {
			result = "fail"
			t.Errorf("%s: observed %s, want %s", c.name, observed, c.want)
		}
		extra := ""
		if c.name == "probe-ctrlc" {
			extra = " first=" + cmp.Or(r.first, "none")
		}
		fmt.Printf("0063 case=%s result=%s observed=%s%s\n", c.name, result, observed, extra)
	}
}

// runProbe runs one case in a pseudo console of its own and returns its
// observed word. A pseudo console that cannot be created, or a child that
// cannot be started in it, is a word of its own and the error is reported:
// P1b's named stop.
func runProbe(t *testing.T, exe string, c probeCase) (string, childResult) {
	t.Helper()
	dir := t.TempDir()
	pc, err := newPseudoConsole()
	if err != nil {
		t.Errorf("%s: create a pseudo console: %v", c.name, err)
		return "no-pseudo-console", childResult{}
	}
	defer pc.close()
	t.Setenv(probeEnv, c.name)
	t.Setenv(dirEnv, dir)
	ctx, cancel := context.WithTimeout(t.Context(), caseDeadline)
	defer cancel()
	exited, err := pc.start(ctx, "", exe, "-test.run=^$")
	if err != nil {
		t.Errorf("%s: start the child in the pseudo console: %v", c.name, err)
		return "no-child", childResult{}
	}
	d := &driver{ctx: ctx, pc: pc, dir: dir, exited: exited}
	if d.waitLine("ready", startDeadline) {
		c.drive(d)
	}
	timedOut := false
	select {
	case <-exited:
	case <-ctx.Done():
		timedOut = true
		<-exited // start's watcher ends the child when ctx is done
	}
	r := readResult(dir)
	switch {
	case !r.done && timedOut:
		return "timeout", r
	case !r.done:
		return "killed", r
	}
	return c.judge(r), r
}

// judgeTypeahead names what two reads gave for "1\r2\r" typed before them.
func judgeTypeahead(r childResult) string {
	if len(r.reads) == 0 {
		return "killed"
	}
	first := r.reads[0]
	switch {
	case first.word != "nil":
		return first.word
	case first.data == "1\r\n2\r\n":
		return "two-lines-in-one-read"
	case first.data == "1\r\n" && len(r.reads) > 1 && r.reads[1].word == "nil" && r.reads[1].data == "2\r\n":
		return "one-line-per-read"
	}
	return "other"
}

// childResult is what the child wrote: whether the interrupt came, each
// read's outcome, and whether it finished.
type childResult struct {
	signal bool
	reads  []readOutcome
	done   bool
	first  string // probe-ctrlc: signal or read, whichever came first
}

type readOutcome struct {
	word string // nil, eof or error
	data string
}

// firstRead names the first read: by its bytes when it gave some (words
// maps the bytes a case knows; any others are "other"), else eof or error.
func (r childResult) firstRead(words map[string]string) string {
	if len(r.reads) == 0 {
		return "killed"
	}
	first := r.reads[0]
	if first.word != "nil" {
		return first.word
	}
	if w, ok := words[first.data]; ok {
		return w
	}
	if words == nil {
		return "bytes"
	}
	return "other"
}

func readResult(dir string) childResult {
	var r childResult
	b, err := os.ReadFile(filepath.Join(dir, resultFile))
	if err != nil {
		return r
	}
	for line := range strings.Lines(string(b)) {
		line = strings.TrimRight(line, "\n")
		switch {
		case line == "signal":
			r.signal = true
		case line == "done":
			r.done = true
		case strings.HasPrefix(line, "first "):
			r.first = strings.TrimPrefix(line, "first ")
		case strings.HasPrefix(line, "read "):
			word, quoted, _ := strings.Cut(strings.TrimPrefix(line, "read "), " ")
			data, err := strconv.Unquote(quoted)
			if err != nil {
				word, data = "error", ""
			}
			r.reads = append(r.reads, readOutcome{word: word, data: data})
		}
	}
	return r
}

// driver types into one case's pseudo console and watches its child.
type driver struct {
	ctx    context.Context
	pc     *pseudoConsole
	dir    string
	exited <-chan struct{}
}

// keys types s into the pseudo console's input.
func (d *driver) keys(s string) {
	_, _ = d.pc.in.Write([]byte(s))
}

// goAhead lets the child make its reads.
func (d *driver) goAhead() {
	_ = os.WriteFile(filepath.Join(d.dir, goFile), nil, 0o600)
}

func (d *driver) sleep(dur time.Duration) {
	select {
	case <-time.After(dur):
	case <-d.ctx.Done():
	case <-d.exited:
	}
}

// waitLine waits up to dur for the child to write line, and reports
// whether it did; it gives up when the child exits or the case ends.
func (d *driver) waitLine(line string, dur time.Duration) bool {
	return d.wait(dur, func(_ childResult, raw string) bool {
		return strings.Contains("\n"+raw, "\n"+line+"\n")
	})
}

// waitReads waits up to dur for n read outcomes, or the child's end.
func (d *driver) waitReads(n int, dur time.Duration) bool {
	return d.wait(dur, func(r childResult, _ string) bool { return len(r.reads) >= n || r.done })
}

func (d *driver) wait(dur time.Duration, ok func(childResult, string) bool) bool {
	deadline := time.After(dur)
	for {
		raw, _ := os.ReadFile(filepath.Join(d.dir, resultFile))
		if ok(readResult(d.dir), string(raw)) {
			return true
		}
		select {
		case <-time.After(25 * time.Millisecond):
		case <-deadline:
			return false
		case <-d.ctx.Done():
			return false
		case <-d.exited:
			raw, _ := os.ReadFile(filepath.Join(d.dir, resultFile))
			return ok(readResult(d.dir), string(raw))
		}
	}
}

// pseudoConsole is a pseudo console and the parent's ends of its pipes:
// in, where what is written is typed, and out, the screen, drained into
// memory for the cases to read and never printed or kept.
type pseudoConsole struct {
	h       windows.Handle
	in      *os.File
	out     *os.File
	drained chan struct{}
	mu      sync.Mutex
	screen  []byte
}

func (pc *pseudoConsole) Write(p []byte) (int, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.screen = append(pc.screen, p...)
	return len(p), nil
}

// text is what the screen showed so far, normalized (see plain).
func (pc *pseudoConsole) text() string {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return plain(string(pc.screen))
}

func newPseudoConsole() (*pseudoConsole, error) {
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
	// The pseudo console holds its own copies of its ends.
	windows.CloseHandle(inR)
	windows.CloseHandle(outW)
	if err != nil {
		windows.CloseHandle(inW)
		windows.CloseHandle(outR)
		return nil, fmt.Errorf("CreatePseudoConsole: %w", err)
	}
	pc := &pseudoConsole{
		h:       h,
		in:      os.NewFile(uintptr(inW), "conpty-in"),
		out:     os.NewFile(uintptr(outR), "conpty-out"),
		drained: make(chan struct{}),
	}
	go func() {
		defer close(pc.drained)
		_, _ = io.Copy(pc, pc.out) // the screen stays in memory
	}()
	return pc, nil
}

// close ends the pseudo console, after its child has exited.
func (pc *pseudoConsole) close() {
	windows.ClosePseudoConsole(pc.h)
	pc.in.Close()
	select {
	case <-pc.drained:
	case <-time.After(5 * time.Second):
	}
	pc.out.Close()
}

// start runs argv attached to the pseudo console, in dir when it is not
// empty, and returns a channel closed when it exits; the child is ended
// when ctx is done.
// STARTF_USESTDHANDLES with no handles keeps the child from inheriting the
// parent's standard handles (pipes under ssh): it gets the pseudo
// console's, as a program started in a console window does.
func (pc *pseudoConsole) start(ctx context.Context, dir string, argv ...string) (<-chan struct{}, error) {
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		return nil, fmt.Errorf("attribute list: %w", err)
	}
	defer attrs.Delete()
	// The attribute's value is the HPCON itself, not a pointer to it.
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
	var cwd *uint16
	if dir != "" {
		if cwd, err = windows.UTF16PtrFromString(dir); err != nil {
			return nil, err
		}
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

// childLog appends the child's result lines, from the reading goroutine
// and the signal's.
type childLog struct {
	mu sync.Mutex
	f  *os.File
}

func (l *childLog) line(s string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	fmt.Fprintln(l.f, s)
	l.f.Sync()
}

// probeChild is the child of one probe case: it takes the interrupt, says
// it is ready, waits for the parent's word, makes its reads of standard
// input and writes each outcome.
func probeChild(name, dir string) int {
	f, err := os.OpenFile(filepath.Join(dir, resultFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 2
	}
	defer f.Close()
	log := &childLog{f: f}

	// A process started by one that ignores Ctrl+C inherits that (as a
	// process group's first process does); a program started from a shell
	// in a console window does not. Undo it, so the interrupt reaches the
	// handler as it would reach gobble's.
	procSetConsoleCtrlHandler.Call(0, 0)
	intr := make(chan os.Signal, 1)
	signal.Notify(intr, os.Interrupt)
	signalled := make(chan struct{})
	go func() {
		<-intr
		close(signalled) // first, so a read that returns after it sees it
		log.line("signal")
	}()

	in := os.Stdin
	var mode uint32
	if windows.GetConsoleMode(windows.Handle(in.Fd()), &mode) != nil {
		// Standard input is not the console after all: read the console's
		// input itself, which is the same buffer gobble's stdin reads.
		conin, err := os.OpenFile("CONIN$", os.O_RDWR, 0)
		if err != nil {
			return 2
		}
		defer conin.Close()
		in = conin
	}
	if name == "probe-ctrlz-raw" {
		restore, err := editor.Raw(in)
		if err != nil {
			log.line("read error " + strconv.Quote(""))
			log.line("done")
			return 0
		}
		defer restore()
	}

	log.line("ready")
	if !waitForFile(filepath.Join(dir, goFile), 20*time.Second) {
		return 2
	}
	reads := 1
	if name == "probe-typeahead" {
		reads = 2
	}
	for range reads {
		buf := make([]byte, 512)
		n, err := in.Read(buf)
		word := "nil"
		switch {
		case errors.Is(err, io.EOF):
			word = "eof"
		case err != nil:
			word = "error"
		}
		if name == "probe-ctrlc" {
			first := "read"
			select {
			case <-signalled:
				first = "signal"
			default:
			}
			select { // so the signal's line is written before the read's
			case <-signalled:
			case <-time.After(2 * time.Second):
			}
			log.line("first " + first)
		}
		log.line("read " + word + " " + strconv.Quote(string(buf[:n])))
		if err != nil {
			break
		}
	}
	log.line("done")
	return 0
}

// waitForFile reports whether path appears within dur.
func waitForFile(path string, dur time.Duration) bool {
	deadline := time.Now().Add(dur)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(25 * time.Millisecond)
	}
	return false
}

// The gobble-mode cases (0063-PLAN P1b and its amendments; MADR
// Confirmation item 5 (A), Deviation 1, D20). Each runs gobble's main in a
// pseudo console of its own, with GOBBLE_HOME a temporary directory, no
// credential variable, and the proxy variables naming a closed local port,
// so nothing leaves the host; every key typed is made up. The screen is
// read in memory and never printed: each case prints only its 0063 line.
// The observed word names what was seen; "result" compares it with the
// word the case gives once P2b is built, so at the base the cases the
// plan expects to fail read "fail".

// closedPort is where the proxy variables point: nothing listens there.
const closedPort = "http://127.0.0.1:9"

// Words a case looks for on the screen, as plain gives them.
var (
	offerFirst  = plain("Sign in to one now? [Y/n]:")
	offerRepeat = plain("Sign in to one now? [y/N]:")
	menuPrompt  = plain("Number [1]:")
	geminiKey   = plain("Enter your Gemini (Google) API key:")
	openaiKey   = plain("Enter your OpenAI API key:")
	cancelLine  = plain("Sign-in cancelled; nothing was saved.")
	warning     = plain("Warning: no model credential")
)

// csi and the other escapes a pseudo console writes.
var escapes = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]|\x1b\][^\x07\x1b]*(\x07|\x1b\\)|\x1b[()][0-9A-Za-z]|\x1b[@-Z\\-_=>]`)

// plain is s without escapes, control characters or white space: a pseudo
// console moves the cursor rather than writing spaces, so text is matched
// with its spaces taken out.
func plain(s string) string {
	s = escapes.ReplaceAllString(s, "")
	return strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7f {
			return -1
		}
		return r
	}, s)
}

// gobbleRun is one case's gobble, in its pseudo console.
type gobbleRun struct {
	pc     *pseudoConsole
	exited <-chan struct{}
	ctx    context.Context
}

func (g *gobbleRun) keys(s string) { _, _ = g.pc.in.Write([]byte(s)) }

func (g *gobbleRun) seen(s string) bool { return strings.Contains(g.pc.text(), s) }

func (g *gobbleRun) count(s string) int { return strings.Count(g.pc.text(), s) }

func (g *gobbleRun) gone() bool {
	select {
	case <-g.exited:
		return true
	default:
		return false
	}
}

func (g *gobbleRun) pause(d time.Duration) {
	select {
	case <-time.After(d):
	case <-g.ctx.Done():
	}
}

// waitFor waits up to d for s on the screen, and reports whether it came.
func (g *gobbleRun) waitFor(s string, d time.Duration) bool {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) && g.ctx.Err() == nil {
		if g.seen(s) {
			return true
		}
		if g.gone() {
			g.pause(200 * time.Millisecond) // the last of the screen
			return g.seen(s)
		}
		time.Sleep(25 * time.Millisecond)
	}
	return g.seen(s)
}

// waitExit waits up to d for gobble to end.
func (g *gobbleRun) waitExit(d time.Duration) bool {
	select {
	case <-g.exited:
		return true
	case <-time.After(d):
		return false
	case <-g.ctx.Done():
		return false
	}
}

// providerNumber is id's number in the provider menu.
func providerNumber(id string) string {
	return strconv.Itoa(slices.Index(provider.KeyProviders(), id) + 1)
}

// apiKeyItem is the number of the method naming an API key in the last
// method menu on the screen, or "".
func apiKeyItem(screen string) string {
	at := strings.LastIndex(screen, plain("Choose how to authenticate with"))
	if at < 0 {
		return ""
	}
	seg := screen[at:]
	if end := strings.Index(seg, plain("Number [")); end >= 0 {
		seg = seg[:end]
	}
	if colon := strings.Index(seg, ":"); colon >= 0 {
		seg = seg[colon+1:]
	}
	for _, m := range regexp.MustCompile(`(\d+)(\D+)`).FindAllStringSubmatch(seg, -1) {
		if strings.Contains(strings.ToLower(m[2]), "apikey") {
			return m[1]
		}
	}
	return ""
}

// declineFirst answers the first offer n, Enter, and waits for the
// session's warning; false when either does not come.
func (g *gobbleRun) declineFirst() bool {
	if !g.waitFor(offerFirst, 20*time.Second) {
		return false
	}
	g.keys("n\r")
	if !g.waitFor(warning, 5*time.Second) {
		return false
	}
	g.pause(700 * time.Millisecond)
	return true
}

// toGeminiKey takes the first offer, picks Gemini and waits for its key
// prompt; false when the prompt does not come.
func (g *gobbleRun) toGeminiKey() bool {
	if !g.waitFor(offerFirst, 20*time.Second) {
		return false
	}
	g.keys("y\r")
	if !g.waitFor(menuPrompt, 5*time.Second) {
		return false
	}
	g.keys(providerNumber("gemini") + "\r")
	return g.waitFor(geminiKey, 5*time.Second)
}

// cancelledEarly reports whether line appears within 2 s of a key prompt
// with no key pressed: the report's defect.
func (g *gobbleRun) cancelledEarly(line string) bool {
	g.pause(2 * time.Second)
	return g.seen(line)
}

// warnedOnce names how many times the warning was printed.
func (g *gobbleRun) warnedOnce() string {
	g.pause(time.Second)
	if n := g.count(warning); n != 1 {
		return fmt.Sprintf("warned-%d", n)
	}
	return "warned-once"
}

// gobbleCase is one gobble-mode case: gobble's arguments, how it is
// driven, and the word it gives once P2b is built.
type gobbleCase struct {
	name  string
	want  string
	args  []string
	drive func(g *gobbleRun) string
}

var gobbleCases = []gobbleCase{
	{name: "report", want: "waits", drive: func(g *gobbleRun) string {
		if !g.toGeminiKey() {
			return "no-key-prompt"
		}
		if g.cancelledEarly(cancelLine) {
			return "cancel-line-before-key"
		}
		return "waits"
	}},
	{name: "masked", want: "masked", drive: func(g *gobbleRun) string {
		if !g.declineFirst() {
			return "no-session"
		}
		g.keys("/login gemini\r")
		if !g.waitFor(geminiKey, 5*time.Second) {
			return "no-key-prompt"
		}
		mark := len(g.pc.text())
		g.pause(300 * time.Millisecond)
		if g.seen(cancelLine) {
			return "cancel-line-before-key"
		}
		g.keys("not-a-real-key")
		g.pause(time.Second)
		after := g.pc.text()[mark:]
		g.keys("\x03")
		if strings.Contains(after, "real") {
			return "shown"
		}
		longest, run := 0, 0
		for _, r := range after {
			if r == '*' {
				run++
				longest = max(longest, run)
			} else {
				run = 0
			}
		}
		if longest != 14 {
			return fmt.Sprintf("stars-%d", longest)
		}
		return "masked"
	}},
	{name: "key-ctrlc", want: "offer-again", drive: func(g *gobbleRun) string {
		return g.keyThenOffer("\x03")
	}},
	{name: "empty-key", want: "offer-again", drive: func(g *gobbleRun) string {
		return g.keyThenOffer("\r")
	}},
	{name: "menu-ctrlc", want: "offer-again", drive: func(g *gobbleRun) string {
		if !g.waitFor(offerFirst, 20*time.Second) {
			return "no-offer"
		}
		g.keys("y\r")
		if !g.waitFor(menuPrompt, 5*time.Second) {
			return "no-menu"
		}
		g.pause(time.Second)
		g.keys("\x03")
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if g.seen(offerRepeat) {
				g.keys("n\r")
				if g.waitFor(warning, 5*time.Second) {
					return "offer-again"
				}
				return "offer-again-no-warning"
			}
			if g.gone() {
				return "ended"
			}
			time.Sleep(25 * time.Millisecond)
		}
		return "other"
	}},
	{name: "decline", want: "warned-once", drive: func(g *gobbleRun) string {
		if !g.toGeminiKey() {
			return "no-key-prompt"
		}
		if g.cancelledEarly(cancelLine) {
			return "cancel-line-before-key"
		}
		g.keys("\x03")
		if !g.waitFor(offerRepeat, 5*time.Second) {
			return "no-offer-again"
		}
		g.keys("n\r")
		if !g.waitFor(warning, 5*time.Second) {
			return "no-warning"
		}
		return g.warnedOnce()
	}},
	{name: "typeahead", want: "waits", drive: func(g *gobbleRun) string {
		g.keys("y\r" + providerNumber("gemini") + "\r") // before the offer is shown
		if !g.waitFor(geminiKey, 20*time.Second) {
			if g.seen(menuPrompt) {
				return "menu-waits"
			}
			return "no-key-prompt"
		}
		if g.cancelledEarly(cancelLine) {
			return "cancel-line-before-key"
		}
		return "waits"
	}},
	{name: "login-no-loop", want: "no-offer", drive: func(g *gobbleRun) string {
		if !g.declineFirst() {
			return "no-session"
		}
		g.keys("/login gemini\r")
		if !g.waitFor(geminiKey, 5*time.Second) {
			return "no-key-prompt"
		}
		g.keys("\r")
		if !g.waitFor(cancelLine, 5*time.Second) {
			return "no-cancel-line"
		}
		g.pause(2 * time.Second)
		if g.seen(offerRepeat) || g.count(offerFirst) > 1 {
			return "offer"
		}
		return "no-offer"
	}},
	{name: "method-menu", want: "waits", drive: func(g *gobbleRun) string {
		if !g.declineFirst() {
			return "no-session"
		}
		g.keys("/login\r")
		if !g.waitFor(menuPrompt, 5*time.Second) {
			return "no-menu"
		}
		g.keys(providerNumber("openai") + "\r")
		if !g.waitFor(plain("Choose how to authenticate with OpenAI:"), 5*time.Second) || !g.waitFor(plain("Number ["), time.Second) {
			return "no-method-menu"
		}
		g.pause(300 * time.Millisecond)
		item := apiKeyItem(g.pc.text())
		if item == "" {
			return "no-api-key-item"
		}
		g.keys(item + "\r")
		if !g.waitFor(openaiKey, 5*time.Second) {
			return "no-key-prompt"
		}
		if g.cancelledEarly(cancelLine) {
			return "cancel-line-before-key"
		}
		return "waits"
	}},
	{name: "key-ctrlz", want: "ended", drive: func(g *gobbleRun) string {
		if !g.declineFirst() {
			return "no-session"
		}
		g.keys("/login gemini\r")
		if !g.waitFor(geminiKey, 5*time.Second) {
			return "no-key-prompt"
		}
		g.pause(300 * time.Millisecond)
		g.keys("\x1a")
		if g.waitExit(5 * time.Second) {
			return "ended"
		}
		return "not-ended"
	}},
	{name: "setup", want: "cancelled", args: []string{"setup"}, drive: func(g *gobbleRun) string {
		if !g.waitFor(menuPrompt, 20*time.Second) {
			return "no-menu"
		}
		g.keys(providerNumber("gemini") + "\r")
		if !g.waitFor(geminiKey, 5*time.Second) {
			return "no-key-prompt"
		}
		if g.cancelledEarly(plain("setup cancelled")) {
			return "cancel-line-before-key"
		}
		g.keys("\x03")
		if g.waitFor(plain("setup cancelled; nothing was saved"), 5*time.Second) {
			return "cancelled"
		}
		return "no-cancel-line"
	}},
	{name: "acp-login", want: "cancelled", args: []string{"acp", "--login", "openai"}, drive: func(g *gobbleRun) string {
		if !g.waitFor(menuPrompt, 20*time.Second) {
			return "no-menu"
		}
		g.pause(300 * time.Millisecond)
		item := apiKeyItem(g.pc.text())
		if item == "" {
			return "no-api-key-item"
		}
		g.keys(item + "\r")
		if !g.waitFor(openaiKey, 5*time.Second) {
			return "no-key-prompt"
		}
		if g.cancelledEarly(plain("sign-in cancelled")) {
			return "cancel-line-before-key"
		}
		g.keys("\x03")
		if g.waitFor(plain("sign-in cancelled; nothing was saved"), 5*time.Second) {
			return "cancelled"
		}
		return "no-cancel-line"
	}},
	{name: "offer-ctrlc", want: "warned-once", drive: func(g *gobbleRun) string {
		if !g.waitFor(offerFirst, 20*time.Second) {
			return "no-offer"
		}
		g.pause(time.Second)
		g.keys("\x03")
		if g.waitExit(2 * time.Second) {
			return "ended"
		}
		g.keys("/exit\r")
		if !g.waitExit(5 * time.Second) {
			return "not-ended"
		}
		return g.warnedOnce()
	}},
	{name: "setup-menu-ctrlc", want: "ended", args: []string{"setup"}, drive: func(g *gobbleRun) string {
		if !g.waitFor(menuPrompt, 20*time.Second) {
			return "no-menu"
		}
		g.pause(time.Second)
		g.keys("\x03")
		if g.waitExit(5 * time.Second) {
			return "ended"
		}
		return "not-ended"
	}},
}

// keyThenOffer takes the first offer to Gemini's key prompt, presses key
// there, and names whether the offer came again.
func (g *gobbleRun) keyThenOffer(key string) string {
	if !g.toGeminiKey() {
		return "no-key-prompt"
	}
	if g.cancelledEarly(cancelLine) {
		return "cancel-line-before-key"
	}
	g.keys(key)
	if !g.waitFor(cancelLine, 5*time.Second) {
		return "no-cancel-line"
	}
	if g.waitFor(offerRepeat, 3*time.Second) {
		return "offer-again"
	}
	return "no-offer-again"
}

func TestConptyGobble(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	var homes []string
	report := func(name, want, observed string) {
		result := "pass"
		if observed != want {
			result = "fail"
			t.Errorf("%s: observed %s, want %s", name, observed, want)
		}
		fmt.Printf("0063 case=%s result=%s observed=%s\n", name, result, observed)
	}
	for _, c := range gobbleCases {
		home := t.TempDir()
		homes = append(homes, home)
		report(c.name, c.want, runGobble(t, exe, home, c))
	}
	// After every case: no case saved a credential.
	observed := "none"
	for _, home := range homes {
		if saved(home) {
			observed = "saved"
		}
	}
	report("nothing-saved", "none", observed)
}

// saved reports whether auth.json under home holds any provider's entry.
func saved(home string) bool {
	f, err := auth.Load(filepath.Join(home, "auth.json"))
	if err != nil {
		return true // unreadable: not shown to be empty
	}
	for _, id := range provider.KeyProviders() {
		if _, ok := f.Entry(id); ok {
			return true
		}
	}
	return false
}

// runGobble runs one gobble-mode case and returns its observed word.
func runGobble(t *testing.T, exe, home string, c gobbleCase) string {
	t.Helper()
	pc, err := newPseudoConsole()
	if err != nil {
		t.Errorf("%s: create a pseudo console: %v", c.name, err)
		return "no-pseudo-console"
	}
	defer pc.close()
	t.Setenv("GOBBLE_HOME", home)
	for _, v := range []string{"HTTPS_PROXY", "HTTP_PROXY", "https_proxy", "http_proxy"} {
		t.Setenv(v, closedPort)
	}
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "")
	for _, v := range provider.CredentialVars() {
		t.Setenv(v, "")
	}
	t.Setenv(gobbleEnv, "run:"+strings.Join(c.args, "\x1f"))
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	exited, err := pc.start(ctx, t.TempDir(), exe, "-test.run=^$")
	if err != nil {
		t.Errorf("%s: start gobble in the pseudo console: %v", c.name, err)
		return "no-child"
	}
	word := c.drive(&gobbleRun{pc: pc, exited: exited, ctx: ctx})
	cancel()
	<-exited
	return word
}

// gobbleChild is gobble's main, as cmd/gobble's main runs it, with args.
func gobbleChild(joined string) int {
	// As in probeChild: a program started from a shell in a console window
	// takes Ctrl+C; one started by a process that ignores it would not.
	procSetConsoleCtrlHandler.Call(0, 0)
	var args []string
	if joined != "" {
		args = strings.Split(joined, "\x1f")
	}
	restore, err := term.PrepareConsole(os.Stdout, os.Stderr)
	code := cli.Main(context.Background(), args, os.Stdin, os.Stdout, os.Stderr)
	if err != nil && code == cli.ExitOK {
		code = cli.ExitFailure
	}
	restore()
	return code
}
