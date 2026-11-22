package cli

import (
	"context"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inline"

	"github.com/maccavelli/gobble-cli/acpclient"
)

// The shared terminal input (0063-PLAN P2 steps 1, 2, 9, 10 and 12, A15 to
// A19, A22, A25, A28; 0063-MADR D1 to D4, D13, D17, D18; rows R1 to R11,
// R14, R17, R41, R43 to R48, R51, R57, R58, R84, R86 to R89).

// countedChunks gives its chunks one read at a time, then the end of input
// (or err, when set), and counts its reads and their sizes.
type countedChunks struct {
	mu    sync.Mutex
	c     []string
	err   error
	reads int
	sizes []int
}

func (r *countedChunks) Read(ctx context.Context, p []byte) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reads++
	r.sizes = append(r.sizes, len(p))
	if len(r.c) == 0 {
		return 0, orErr(r.err, io.EOF)
	}
	n := copy(p, r.c[0])
	if n < len(r.c[0]) {
		r.c[0] = r.c[0][n:]
	} else {
		r.c = r.c[1:]
	}
	return n, nil
}

func (r *countedChunks) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.reads
}

// orErr is err, or def when err is nil.
func orErr(err, def error) error {
	if err != nil {
		return err
	}
	return def
}

// inStep is one thing a prompt asks of the terminal input, and what it
// must give.
type inStep struct {
	do   string // line, key, read, raw\r, raw\n, discard, ended
	want string // the text, byte or bytes given; for ended, "true" or "false"
	err  error  // the error wanted, by errors.Is
}

// run does each step on t, over chunks.
func runSteps(tb testing.TB, t *terminalInput, steps []inStep) {
	tb.Helper()
	ctx := context.Background()
	for i, s := range steps {
		var got string
		var err error
		switch s.do {
		case "line":
			got, err = t.line(ctx)
		case "key":
			var c byte
			c, err = t.key(ctx)
			if err == nil {
				got = string([]byte{c})
			}
		case "read":
			buf := make([]byte, 64)
			var n int
			n, err = t.Read(ctx, buf)
			got = string(buf[:n])
		case "raw\r":
			t.rawEnd('\r')
			continue
		case "raw\n":
			t.rawEnd('\n')
			continue
		case "discard":
			t.discard()
			continue
		case "ended":
			got = strconv.FormatBool(t.ended())
		default:
			tb.Fatalf("step %d: no such step %q", i, s.do)
		}
		if s.err != nil {
			if !errors.Is(err, s.err) {
				tb.Fatalf("step %d, %s: %q, %v; want the error %v", i, s.do, got, err, s.err)
			}
			continue
		}
		if err != nil || got != s.want {
			tb.Fatalf("step %d, %s: %q, %v; want %q", i, s.do, got, err, s.want)
		}
	}
}

var errBroken = errors.New("the terminal broke")

// D3's rule by table: each line end, cooked and raw ends, the owed \n in
// the same read and a later one, at each entry point, \n then \r, and a
// byte that clears what is owed (step 2; A15 (i), (ii), (iii), (v)).
func TestTerminalInputRule(t *testing.T) {
	for _, c := range []struct {
		name   string
		chunks []string
		steps  []inStep
	}{
		{"LF", []string{"a\nb\n"}, []inStep{{do: "line", want: "a"}, {do: "line", want: "b"}}},
		{"CR", []string{"a\rb\r"}, []inStep{{do: "line", want: "a"}, {do: "line", want: "b"}}},
		{"CRLF in one read", []string{"a\r\nb\r\n"}, []inStep{{do: "line", want: "a"}, {do: "line", want: "b"}}},
		// R4: the \n of one end opening the next read is that end's.
		{"owed LF in a later read, line", []string{"a\r", "\nb\n"}, []inStep{{do: "line", want: "a"}, {do: "line", want: "b"}}},
		// A15 (ii), R10: the owed \n meets each entry point, now or later.
		{"owed LF in a later read, key", []string{"a\r", "\nk"}, []inStep{{do: "line", want: "a"}, {do: "key", want: "k"}}},
		{"owed LF in a later read, Read", []string{"a\r", "\nxy"}, []inStep{{do: "line", want: "a"}, {do: "read", want: "xy"}}},
		{"owed LF in the same read, key", []string{"a\r\nk"}, []inStep{{do: "line", want: "a"}, {do: "key", want: "k"}}},
		{"owed LF in the same read, Read", []string{"a\r\nxy"}, []inStep{{do: "line", want: "a"}, {do: "read", want: "xy"}}},
		// R6, Q1: \n then \r is two ends.
		{"LF then CR, line", []string{"y\n\r"}, []inStep{{do: "line", want: "y"}, {do: "line", want: ""}}},
		{"LF then CR, key", []string{"y\n", "\r"}, []inStep{{do: "line", want: "y"}, {do: "key", want: "\r"}}},
		// R10: any byte given clears what is owed; a \n after it is a line end.
		{"a byte clears the owed LF", []string{"a\r", "x", "\nb\n"}, []inStep{{do: "line", want: "a"}, {do: "key", want: "x"}, {do: "line", want: ""}, {do: "line", want: "b"}}},
		// R8: a raw \r drops a \n already read directly after it.
		{"raw CR, LF already read", []string{"k\r\nx\n"}, []inStep{{do: "key", want: "k"}, {do: "key", want: "\r"}, {do: "raw\r"}, {do: "line", want: "x"}}},
		// R7: a raw \r owes nothing: a \n in a later read is the next line's end.
		{"raw CR owes nothing", []string{"k\r", "\n"}, []inStep{{do: "key", want: "k"}, {do: "key", want: "\r"}, {do: "raw\r"}, {do: "line", want: ""}}},
		// A15 (i), R9: a raw read ended by \n, then \n: the second is given.
		{"raw LF then LF", []string{"k\n\n"}, []inStep{{do: "key", want: "k"}, {do: "key", want: "\n"}, {do: "raw\n"}, {do: "read", want: "\n"}}},
		// A15 (iii), R11: a lone \r, then the end of input.
		{"CR then the end of input", []string{"a\r"}, []inStep{{do: "line", want: "a"}, {do: "ended", want: "false"}, {do: "line", err: io.EOF}, {do: "ended", want: "true"}}},
		// A15 (v), R44: bytes that are not UTF-8, and control bytes, unchanged.
		{"bytes unchanged, line", []string{"\xff\xc3\x00\x1b\n"}, []inStep{{do: "line", want: "\xff\xc3\x00\x1b"}}},
		{"bytes unchanged, key", []string{"\xff\xc3\x00"}, []inStep{{do: "key", want: "\xff"}, {do: "key", want: "\xc3"}, {do: "key", want: "\x00"}}},
		{"bytes unchanged, Read", []string{"\xff\xc3\x00"}, []inStep{{do: "read", want: "\xff\xc3\x00"}}},
		// D4: discard drops what waits and keeps what is owed.
		{"discard keeps the owed LF", []string{"a\rjunk", "\nb\n"}, []inStep{{do: "line", want: "a"}, {do: "discard"}, {do: "line", want: "b"}}},
		// R16: a line with no end before the end of input is no answer.
		{"no line end before the end", []string{"abc"}, []inStep{{do: "line", err: io.EOF}, {do: "ended", want: "true"}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			runSteps(t, newTerminalInput(&countedChunks{c: c.chunks}, nil), c.steps)
		})
	}
}

// Row R14 and A15 (iv), R43: the end of input, and a read that fails with
// another error, are returned as the reader gives them, close the input for
// good, and cancel a question; an ended context does neither (R41).
func TestTerminalInputEnds(t *testing.T) {
	in := &countedChunks{}
	ti := newTerminalInput(in, nil)
	if _, err := ti.line(t.Context()); !errors.Is(err, io.EOF) || !ti.ended() {
		t.Fatalf("the end of input: %v, ended %v; want io.EOF, ended", err, ti.ended())
	}
	if _, err := ti.Read(t.Context(), make([]byte, 8)); !errors.Is(err, io.EOF) {
		t.Fatalf("a read after the end: %v; want io.EOF", err)
	}

	q, _, _ := askedFrom(t, &countedChunks{err: errBroken})
	if _, err := q.input("Name"); !errors.Is(err, errCancelled) || !errors.Is(err, errBroken) || !q.in.ended() {
		t.Fatalf("a failed read: %v, ended %v; want cancelled, the read's error, ended", err, q.in.ended())
	}

	b := &blockingIn{started: make(chan struct{})}
	ti = newTerminalInput(b, nil)
	ctx, cancel := context.WithCancel(t.Context())
	go func() {
		<-b.started
		cancel()
	}()
	if _, err := ti.line(ctx); !errors.Is(err, context.Canceled) || ti.ended() {
		t.Fatalf("an ended context: %v, ended %v; want context.Canceled, not ended", err, ti.ended())
	}
}

// Row R47: no read of the input while pending holds a byte; one read per
// need, with the prompt's own buffer.
func TestTerminalInputReadsOnlyWhenAsked(t *testing.T) {
	in := &countedChunks{c: []string{"a\nb\n", "cd"}}
	ti := newTerminalInput(in, nil)
	if s, err := ti.line(t.Context()); err != nil || s != "a" || in.count() != 1 {
		t.Fatalf("line = %q, %v after %d reads; want \"a\" after 1", s, err, in.count())
	}
	if s, err := ti.line(t.Context()); err != nil || s != "b" || in.count() != 1 {
		t.Fatalf("line = %q, %v after %d reads; want \"b\" with no new read", s, err, in.count())
	}
	buf := make([]byte, 7)
	if n, err := ti.Read(t.Context(), buf); err != nil || string(buf[:n]) != "cd" || in.count() != 2 {
		t.Fatalf("Read = %q, %v after %d reads; want \"cd\" after 2", buf[:n], err, in.count())
	}
	if !slices.Equal(in.sizes, []int{lineBuf, 7}) {
		t.Fatalf("the reads' buffers %v; want a question's %d, then the Read's 7", in.sizes, lineBuf)
	}
}

// Follow-up 3 of the P1 review: the owed \n is dropped however late it
// comes, here after the read that waits for it has waited a while.
func TestOwedLFAfterADelay(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })
	ti := newTerminalInput(inline.NewReader(pr), nil)
	go func() { _, _ = io.WriteString(pw, "a\r") }()
	if s, err := ti.line(t.Context()); err != nil || s != "a" {
		t.Fatalf("line = %q, %v", s, err)
	}
	got := make(chan string, 1)
	go func() {
		s, err := ti.line(t.Context())
		got <- fmt.Sprintf("%q, %v", s, err)
	}()
	time.Sleep(700 * time.Millisecond)
	if _, err := io.WriteString(pw, "\nb\n"); err != nil {
		t.Fatal(err)
	}
	select {
	case s := <-got:
		if s != `"b", <nil>` {
			t.Fatalf("the line after a late \\n = %s; want \"b\"", s)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the line was not read")
	}
}

// Step 9 (MADR D12): flushNext makes the next read of the input flush the
// terminal's queue first, once, and only when there is a terminal; with
// pending bytes to give, no read is made and the flush waits for one. P2b's
// repeated offer uses it, so the flush follows the offer's text.
func TestFlushNext(t *testing.T) {
	in := &countedChunks{c: []string{"a\nb\n", "c\n"}}
	var at []int
	ti := newTerminalInput(in, func() error { at = append(at, in.count()); return nil })
	if s, err := ti.line(t.Context()); err != nil || s != "a" || len(at) != 0 {
		t.Fatalf("line = %q, %v, %d flushes; want \"a\" and none", s, err, len(at))
	}
	ti.flushNext()
	if s, err := ti.line(t.Context()); err != nil || s != "b" || len(at) != 0 {
		t.Fatalf("line = %q, %v, %d flushes; want \"b\" from pending, no flush yet", s, err, len(at))
	}
	if s, err := ti.line(t.Context()); err != nil || s != "c" || !slices.Equal(at, []int{1}) {
		t.Fatalf("line = %q, %v, flushes at reads %v; want \"c\", one flush before the second read", s, err, at)
	}
	none := newTerminalInput(&countedChunks{c: []string{"x\n"}}, nil)
	none.flushNext()
	if s, err := none.line(t.Context()); err != nil || s != "x" {
		t.Fatalf("with no terminal: %q, %v", s, err)
	}
}

// A16 (row R58): one chunk a\rbc: the line takes a, Read takes bc, and the
// next Read makes a new read: each byte is in one place and given once.
func TestTerminalInputBytesInOnePlace(t *testing.T) {
	in := &countedChunks{c: []string{"a\rbc", "d"}}
	ti := newTerminalInput(in, nil)
	if s, err := ti.line(t.Context()); err != nil || s != "a" {
		t.Fatalf("line = %q, %v", s, err)
	}
	buf := make([]byte, 16)
	if n, err := ti.Read(t.Context(), buf); err != nil || string(buf[:n]) != "bc" {
		t.Fatalf("Read = %q, %v; want \"bc\"", buf[:n], err)
	}
	if n, err := ti.Read(t.Context(), buf); err != nil || string(buf[:n]) != "d" || in.count() != 2 {
		t.Fatalf("Read = %q, %v after %d reads; want \"d\" from a new read", buf[:n], err, in.count())
	}
}

// A17 (row R57): what askOn decoded past its answer is dropped when it
// returns (Q19): the next question makes a new read and does not see it.
func TestAskOnKeysPastAnswerNotShared(t *testing.T) {
	in := &countedChunks{c: []string{"1hello\r", "next\n"}}
	ti := newTerminalInput(in, nil)
	q := permissionQuestion(nil, acpclient.PermissionRequest{Tool: acpclient.ToolCall{Title: "write a.txt"}, Options: toolOptions()})
	resp, _, err := askOn(t.Context(), q, 80, false, io.Discard, ti.Read)
	if err != nil || resp.Canceled || resp.Answer.ID != "allow_once" {
		t.Fatalf("askOn = %+v, %v; want allow_once", resp, err)
	}
	if s, err := ti.line(t.Context()); err != nil || s != "next" || in.count() != 2 {
		t.Fatalf("the next line = %q, %v after %d reads; want \"next\" from a new read", s, err, in.count())
	}
}

// A18 (rows R45, D10's gobble auth login row): the package's non-test Go
// files make an inline.Reader in terminput.go alone: every reader of the
// terminal in the process goes through the one terminal input.
func TestOneReaderOfStdin(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	var calls []string
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr)
			if ok && sel.Sel.Name == "NewReader" {
				if id, ok := sel.X.(*ast.Ident); ok && id.Name == "inline" {
					calls = append(calls, fset.Position(sel.Pos()).String())
				}
			}
			return true
		})
	}
	if len(calls) != 1 || !strings.HasPrefix(calls[0], "terminput.go:") {
		t.Fatalf("inline.NewReader is called at %q; want once, in terminput.go", calls)
	}
}

// A19 (row R48): two goroutines read at once, through line and Read, from
// a known byte string: every byte is given once, and nothing races (run
// under -race in P4).
func TestTerminalInputConcurrentReads(t *testing.T) {
	var lines []string
	for i := range 200 {
		lines = append(lines, fmt.Sprintf("l%03d\n", i))
	}
	all := strings.Join(lines, "")
	var chunks []string
	for s := all; s != ""; {
		n := min(len(s), 7)
		chunks = append(chunks, s[:n])
		s = s[n:]
	}
	orig := slices.Clone(chunks)
	ti := newTerminalInput(&countedChunks{c: chunks}, nil)
	var mu sync.Mutex
	var got []string
	var wg sync.WaitGroup
	wg.Go(func() {
		for {
			s, err := ti.line(t.Context())
			if err != nil {
				return
			}
			mu.Lock()
			got = append(got, s+"\n")
			mu.Unlock()
		}
	})
	wg.Go(func() {
		buf := make([]byte, 3)
		for {
			n, err := ti.Read(t.Context(), buf)
			if err != nil {
				return
			}
			mu.Lock()
			got = append(got, string(buf[:n]))
			mu.Unlock()
		}
	})
	wg.Wait()
	given := 0
	for _, s := range got {
		given += len(s)
	}
	if given != len(all) {
		t.Fatalf("%d bytes given; want each of the %d once", given, len(all))
	}
	count := map[byte]int{}
	for _, s := range got {
		for i := range len(s) {
			count[s[i]]++
		}
	}
	for i := range len(all) {
		count[all[i]]--
	}
	for b, n := range count {
		if n != 0 {
			t.Fatalf("byte %q given %+d times more than it was read", b, n)
		}
	}

	// (2026-11-21, P3-2, the owner's "As above, plus tighten the P1 test"
	// ruling on plant 35: a third goroutine calls discard() and ended() while
	// the two read. Every reader holds the turn, a hand-off the race detector
	// sees, so only these methods, which do not take it, can race with a read
	// of pending taken without the lock. discard drops bytes, so here no byte
	// may be given more often than it was read.)
	ti = newTerminalInput(&countedChunks{c: orig}, nil)
	got = nil
	stop := make(chan struct{})
	var side sync.WaitGroup
	side.Go(func() {
		for {
			select {
			case <-stop:
				return
			default:
				ti.discard()
				_ = ti.ended()
			}
		}
	})
	var wg2 sync.WaitGroup
	wg2.Go(func() {
		for {
			s, err := ti.line(t.Context())
			if err != nil {
				return
			}
			mu.Lock()
			got = append(got, s+"\n")
			mu.Unlock()
		}
	})
	wg2.Go(func() {
		buf := make([]byte, 3)
		for {
			n, err := ti.Read(t.Context(), buf)
			if err != nil {
				return
			}
			mu.Lock()
			got = append(got, string(buf[:n]))
			mu.Unlock()
		}
	})
	wg2.Wait()
	close(stop)
	side.Wait()
	count = map[byte]int{}
	for _, s := range got {
		for i := range len(s) {
			count[s[i]]++
		}
	}
	for i := range len(all) {
		count[all[i]]--
	}
	for b, n := range count {
		if n > 0 {
			t.Fatalf("with discard and ended called alongside, byte %q was given %+d times more than it was read", b, n)
		}
	}
}

// A22 (row R46): terminalIn makes one value, called twice and from two
// goroutines at once.
func TestTerminalInMadeOnce(t *testing.T) {
	e := &runEnv{ctx: t.Context(), out: &Output{out: io.Discard, err: io.Discard}, stdin: stdinFile(t, "")}
	first := e.terminalIn()
	if e.terminalIn() != first {
		t.Fatal("a second call made another terminal input")
	}
	e2 := &runEnv{ctx: t.Context(), out: &Output{out: io.Discard, err: io.Discard}, stdin: stdinFile(t, "")}
	var got [2]*terminalInput
	var wg sync.WaitGroup
	for i := range got {
		wg.Go(func() { got[i] = e2.terminalIn() })
	}
	wg.Wait()
	if got[0] == nil || got[0] != got[1] {
		t.Fatal("two goroutines at once made two terminal inputs")
	}
}

// A25 (row R17): an answer has no bound: a 1 MiB line, and a 1 MiB secret,
// each read whole.
func TestLongAnswerHasNoBound(t *testing.T) {
	// countedChunks keeps what a read's buffer did not take for the next
	// read, as a terminal does; chunks would drop it.
	long := strings.Repeat("x", 1<<20)
	q, _, _ := askedFrom(t, &countedChunks{c: []string{long + "\n"}})
	if s, err := q.input("Long"); err != nil || len(s) != len(long) {
		t.Fatalf("input of %d bytes, %v; want %d", len(s), err, len(long))
	}
	q, _, _ = askedFrom(t, &countedChunks{c: []string{long + "\r"}})
	if s, err := q.secret("Key"); err != nil || len(s) != len(long) {
		t.Fatalf("secret of %d bytes, %v; want %d", len(s), err, len(long))
	}
}

// pipeInput is a terminal input over inline.Reader over a pipe, as the
// process's is over standard input: a read the context gave up on stays
// waiting in inline.Reader, which is what D13 is about.
func pipeInput(t *testing.T) (*terminalInput, *io.PipeWriter) {
	t.Helper()
	pr, pw := io.Pipe()
	t.Cleanup(func() { _ = pw.Close() })
	return newTerminalInput(inline.NewReader(pr), nil), pw
}

// giveUp starts a read through read with a context that the program ends
// once the read waits, and returns its error.
func giveUp(t *testing.T, read func(context.Context) error) error {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()
	return read(ctx)
}

// stubResult is what one read of sentIn gives.
type stubResult struct {
	b   string
	err error
}

// sentIn gives each read what the test sends, waiting until ctx ends; a
// send no read took goes to the next read, as a read inline.Reader keeps
// waiting does.
type sentIn struct{ results chan stubResult }

func (s *sentIn) Read(ctx context.Context, p []byte) (int, error) {
	select {
	case r := <-s.results:
		return copy(p, r.b), r.err
	case <-ctx.Done():
		return 0, ctx.Err()
	}
}

// write writes s to w, in the background: a pipe's write waits for a read.
func write(w io.Writer, s string) {
	go func() { _, _ = io.WriteString(w, s) }()
}

// lineWithin reads a line within d, or fails.
func lineWithin(t *testing.T, ti *terminalInput, d time.Duration) (string, error) {
	t.Helper()
	type res struct {
		s   string
		err error
	}
	got := make(chan res, 1)
	go func() {
		s, err := ti.line(t.Context())
		got <- res{s, err}
	}()
	select {
	case r := <-got:
		return r.s, r.err
	case <-time.After(d):
		t.Fatal("the line was not read")
	}
	return "", nil
}

// A28 (rows R84, R86 to R89, R102): the read of a prompt a program gave
// up. Its completion, a line or keys, is dropped, whoever reads next, and
// the prompt waiting reads again; a timed wait inside a prompt that reads
// again keeps its bytes; two give-ups drop one completion; the end of input
// is not dropped; a read the person gave up with Ctrl+C is not dropped.
func TestAbandonedRead(t *testing.T) {
	t.Run("line dropped", func(t *testing.T) {
		ti, pw := pipeInput(t)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		if err := giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err }); !errors.Is(err, context.Canceled) {
			t.Fatalf("the given-up read: %v", err)
		}
		write(pw, "hello\n")
		time.Sleep(100 * time.Millisecond)
		write(pw, "next\n")
		if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != "next" {
			t.Fatalf("the next line = %q, %v; want \"next\", the given-up line dropped", s, err)
		}
	})
	t.Run("keys dropped", func(t *testing.T) {
		ti, pw := pipeInput(t)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
		write(pw, "par")
		time.Sleep(100 * time.Millisecond)
		write(pw, "next\n")
		if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != "next" {
			t.Fatalf("the next line = %q, %v; want \"next\", the keys with no line end dropped whole", s, err)
		}
	})
	t.Run("timed wait keeps its bytes", func(t *testing.T) {
		ti, pw := pipeInput(t)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		if _, err := ti.Read(ctx, make([]byte, 16)); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("the timed wait: %v", err)
		}
		write(pw, "kept")
		buf := make([]byte, 16)
		bounded, stop := context.WithTimeout(t.Context(), 3*time.Second)
		defer stop()
		if n, err := ti.Read(bounded, buf); err != nil || string(buf[:n]) != "kept" {
			t.Fatalf("the same caller's next read = %q, %v; want \"kept\"", buf[:n], err)
		}
	})
	t.Run("two give-ups drop one", func(t *testing.T) {
		ti, pw := pipeInput(t)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		for range 2 {
			_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
		}
		write(pw, "first\n")
		time.Sleep(100 * time.Millisecond)
		write(pw, "second\n")
		if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != "second" {
			t.Fatalf("the next line = %q, %v; want \"second\": one completion dropped", s, err)
		}
	})
	t.Run("the end of input is not dropped", func(t *testing.T) {
		// Over a reader whose end of input is not kept for good, as
		// inline.Reader keeps it, so dropping it would show.
		in := &sentIn{results: make(chan stubResult)}
		ti := newTerminalInput(in, nil)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
		go func() {
			in.results <- stubResult{err: io.EOF}
			in.results <- stubResult{b: "more\n"}
		}()
		if s, err := lineWithin(t, ti, 3*time.Second); !errors.Is(err, io.EOF) || !ti.ended() {
			t.Fatalf("after the input ended: %q, %v, ended %v; want io.EOF, ended", s, err, ti.ended())
		}
	})
	t.Run("Ctrl+C keeps it", func(t *testing.T) {
		ti, pw := pipeInput(t)
		ctx, cancel := context.WithCancel(t.Context())
		q := &questions{ctx: ctx, in: ti, out: io.Discard}
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel() // ls.ask's interrupt cancel: the question's own context
		}()
		if _, err := q.line(); !errors.Is(err, errCancelled) {
			t.Fatalf("the interrupted question: %v", err)
		}
		write(pw, "n\n")
		if s, err := lineWithin(t, ti, 3*time.Second); err != nil || s != "n" {
			t.Fatalf("the next line = %q, %v; want \"n\", the person's next answer", s, err)
		}
	})
	t.Run("taken whole with the read's own size", func(t *testing.T) {
		ti, pw := pipeInput(t)
		q := &questions{ctx: t.Context(), in: ti, out: io.Discard}
		_ = giveUp(t, func(ctx context.Context) error { _, err := q.inputContext(ctx, "Paste"); return err })
		write(pw, strings.Repeat("p", 400))
		time.Sleep(100 * time.Millisecond)
		write(pw, "y")
		buf := make([]byte, 256) // askOn's
		n, err := ti.Read(t.Context(), buf)
		if err != nil || string(buf[:n]) != "y" {
			t.Fatalf("a waiting 256-byte read = %q, %v; want \"y\", none of the 400 given-up bytes", buf[:n], err)
		}
	})
}
