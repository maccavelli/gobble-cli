package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// The question reader, by bytes (0063-PLAN P1 step 3 and its additions A1
// to A4 and A26; 0063-MADR D3, rows R1 to R8, R16, R23, R28 to R31). Each
// test builds its questions through asked or askedFrom, so that P2's change
// of the reader they read through is made in one place.

// askedFrom is questions over in, through a terminal input with no
// terminal to flush (0063-PLAN P2 step 4), writing to a buffer, with a raw
// mode that counts its entries and restores.
func askedFrom(t *testing.T, in keysIn) (*questions, *bytes.Buffer, *[2]int) {
	t.Helper()
	q, out, raws := asked(t)
	q.in = newTerminalInput(in, nil)
	return q, out, raws
}

// endForm is one way the answers' line ends reach the reader: E, the line
// end, and whether the first answer's \r and \n come in two reads (split)
// or the next answer comes in the same read as the first (oneChunk).
type endForm struct {
	name     string
	enter    string
	split    bool // "CR|LF": the first answer and \r, then \n opening the next read
	oneChunk bool // the first answer, E and the next answer in one read
}

var endForms = []endForm{
	{name: "CR", enter: "\r"},
	{name: "LF", enter: "\n"},
	{name: "CRLF", enter: "\r\n"},
	{name: "CR|LF", enter: "\r\n", split: true},
	{name: "one chunk CR", enter: "\r", oneChunk: true},
	{name: "one chunk LF", enter: "\n", oneChunk: true},
	{name: "one chunk CRLF", enter: "\r\n", oneChunk: true},
}

// firstQuestion is K1: asked, answered with its text, and checked.
type firstQuestion struct {
	name   string
	answer string
	ask    func(q *questions) (string, error)
	want   string
}

// nextQuestion is K2: its answer as typed (raw Enter for a secret), and
// what it must give.
type nextQuestion struct {
	name   string
	answer func(enter string) string
	ask    func(q *questions) (string, error)
	want   string
}

var firstQuestions = []firstQuestion{
	{"choose", "2", func(q *questions) (string, error) {
		i, err := q.choose("Which?", []option{{"Alpha", ""}, {"Beta", ""}}, 0)
		return strconv.Itoa(i), err
	}, "1"},
	{"confirm", "y", func(q *questions) (string, error) {
		ok, err := q.confirm("Go?", false)
		return strconv.FormatBool(ok), err
	}, "true"},
	{"input", "first", func(q *questions) (string, error) { return q.input("Name") }, "first"},
}

var nextQuestions = []nextQuestion{
	// A secret is read in raw mode, where Enter is \r whatever the form.
	{"secret", func(string) string { return "the-key\r" }, func(q *questions) (string, error) { return q.secret("Key") }, "the-key"},
	{"choose", func(e string) string { return "1" + e }, func(q *questions) (string, error) {
		i, err := q.choose("Which?", []option{{"Alpha", ""}, {"Beta", ""}}, 1)
		return strconv.Itoa(i), err
	}, "0"},
	{"confirm", func(e string) string { return "n" + e }, func(q *questions) (string, error) {
		ok, err := q.confirm("Again?", true)
		return strconv.FormatBool(ok), err
	}, "false"},
	{"input", func(e string) string { return "second" + e }, func(q *questions) (string, error) { return q.input("Other") }, "second"},
}

// chunksFor is what the reader gives for K1's answer and K2's in form f.
func chunksFor(f endForm, k1 firstQuestion, k2 nextQuestion) []string {
	a2 := k2.answer(f.enter)
	switch {
	case f.split:
		return []string{k1.answer + "\r", "\n" + a2}
	case f.oneChunk:
		return []string{k1.answer + f.enter + a2}
	}
	return []string{k1.answer + f.enter, a2}
}

// Each first question, answered with its text and E, then each next
// question answered after it: the next question gets its own answer. Red
// for the defect where K2 is the secret and the \n of a \r\n reaches it
// (CRLF, CR|LF, one chunk CRLF).
func TestLineEndsBeforeEachQuestion(t *testing.T) {
	for _, f := range endForms {
		t.Run(f.name, func(t *testing.T) {
			for _, k1 := range firstQuestions {
				for _, k2 := range nextQuestions {
					t.Run(k1.name+"-"+k2.name, func(t *testing.T) {
						q, _, _ := asked(t, chunksFor(f, k1, k2)...)
						if got, err := k1.ask(q); err != nil || got != k1.want {
							t.Fatalf("%s = %q, %v; want %q", k1.name, got, err, k1.want)
						}
						if got, err := k2.ask(q); err != nil || got != k2.want {
							t.Fatalf("%s = %q, %v; want %q", k2.name, got, err, k2.want)
						}
					})
				}
			}
		})
	}
}

// Weakness (d): a secret ended by \r in raw mode owes nothing, so the next
// line question's Enter, a cooked \n in a later read, answers it at once.
// Red for the defect: the \n is dropped and the read meets the end of input.
func TestEnterAfterSecretAnswersTheNextLine(t *testing.T) {
	q, _, _ := asked(t, "the-key\r", "\n")
	if s, err := q.secret("Key"); err != nil || s != "the-key" {
		t.Fatalf("secret = %q, %v", s, err)
	}
	if s, err := q.input("Workspace"); err != nil || s != "" {
		t.Fatalf("input = %q, %v; want \"\" at once", s, err)
	}
}

// The \r and \n of one end split across two reads end one line.
func TestCRLFSplitAcrossReads(t *testing.T) {
	q, _, _ := asked(t, "a\r", "\nb\n")
	for _, want := range []string{"a", "b"} {
		if s, err := q.input("Line"); err != nil || s != want {
			t.Fatalf("input = %q, %v; want %q", s, err, want)
		}
	}
}

// A pasted key ending \r\n: the \n already read is the key's, and the next
// question gets its own line.
func TestPastedKeyEndingCRLF(t *testing.T) {
	q, _, _ := asked(t, "the-key\r\n", "x\n")
	if s, err := q.secret("Key"); err != nil || s != "the-key" {
		t.Fatalf("secret = %q, %v", s, err)
	}
	if s, err := q.input("Next"); err != nil || s != "x" {
		t.Fatalf("input = %q, %v; want \"x\"", s, err)
	}
}

// \n then \r is two ends (Q1): a yes answered y\n, then a secret given \r,
// is an empty secret, which the prompter cancels.
func TestLFThenCRIsTwoEnds(t *testing.T) {
	q, _, _ := asked(t, "y\n", "\r")
	if ok, err := q.confirm("Go?", false); err != nil || !ok {
		t.Fatalf("confirm = %v, %v", ok, err)
	}
	// (2026-11-21, P3-1, the owner's "As above, plus tighten the P1 test"):
	// the secret is cancelled by the empty answer itself, not by the end of
	// input, which a \r dropped from the input would also give.
	if s, err := (prompter{q}).Secret("Key"); !errors.Is(err, errCancelled) || errors.Is(err, io.EOF) {
		t.Fatalf("secret = %q, %v; want cancelled by the empty answer, not by the end of input", s, err)
	}
}

// A2 (row R16): a line with no end before the end of input is no answer.
func TestLineEndOfInputMidLine(t *testing.T) {
	q, _, _ := asked(t, "abc")
	if s, err := q.input("Name"); !errors.Is(err, errCancelled) {
		t.Fatalf("input = %q, %v; want cancelled", s, err)
	}
}

// countingIn counts its reads, and gives the end of input.
type countingIn struct {
	mu    sync.Mutex
	reads int
}

func (c *countingIn) Read(context.Context, []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.reads++
	return 0, errors.New("countingIn has nothing")
}

// A3 (row R28): raw mode that cannot be switched on: the secret returns its
// error, writes no prompt and reads nothing.
func TestSecretRawFailureReadsNothing(t *testing.T) {
	in := &countingIn{}
	q, out, _ := askedFrom(t, in)
	failed := errors.New("raw mode refused")
	q.raw = func() (func() error, error) { return nil, failed }
	if _, err := q.secret("Key"); !errors.Is(err, failed) {
		t.Fatalf("secret = %v; want the raw mode's error", err)
	}
	if out.Len() != 0 {
		t.Errorf("the secret wrote %q", out)
	}
	if in.reads != 0 {
		t.Errorf("the secret read stdin %d times; want none", in.reads)
	}
}

// blockingIn waits for its context, as inline.Reader does with no input.
type blockingIn struct{ started chan struct{} }

func (b *blockingIn) Read(ctx context.Context, _ []byte) (int, error) {
	close(b.started)
	<-ctx.Done()
	return 0, ctx.Err()
}

// A4 (row R30): the context ends while a secret reads: cancelled, raw mode
// entered once and restored once.
func TestSecretRestoredWhenContextEnds(t *testing.T) {
	in := &blockingIn{started: make(chan struct{})}
	q, _, raws := askedFrom(t, in)
	ctx, cancel := context.WithCancel(t.Context())
	q.ctx = ctx
	go func() {
		<-in.started
		cancel()
	}()
	if _, err := q.secret("Key"); !errors.Is(err, errCancelled) {
		t.Fatalf("secret = %v; want cancelled", err)
	}
	if *raws != [2]int{1, 1} {
		t.Fatalf("raw mode entered %d, restored %d; want once each", raws[0], raws[1])
	}
}

// panickingIn panics on its read.
type panickingIn struct{}

func (panickingIn) Read(context.Context, []byte) (int, error) { panic("the reader broke") }

// A26 (row R31): a panic while a secret reads still restores the terminal
// before it goes on. Red at the base: the restore is not deferred.
func TestSecretRestoredOnPanic(t *testing.T) {
	q, _, raws := askedFrom(t, panickingIn{})
	func() {
		defer func() {
			if r := recover(); r == nil || !strings.Contains(fmt.Sprint(r), "the reader broke") {
				t.Errorf("recovered %v; want the reader's panic", r)
			}
		}()
		_, _ = q.secret("Key")
	}()
	if *raws != [2]int{1, 1} {
		t.Fatalf("raw mode entered %d, restored %d; want once each", raws[0], raws[1])
	}
}
