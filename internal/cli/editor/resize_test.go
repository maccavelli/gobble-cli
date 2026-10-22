package editor

import (
	"context"
	"io"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/internal/cli/term/termtest"
)

// lockedScreen is a termtest screen the editor's goroutine draws on while
// the test reads it.
type lockedScreen struct {
	mu sync.Mutex
	s  *termtest.Screen
}

func (l *lockedScreen) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s.Write(p)
}

func (l *lockedScreen) look() (lines []string, row, col int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.s.Lines(), l.s.Row, l.s.Col
}

func (l *lockedScreen) resize(columns int, how termtest.Narrowing) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.s.Resize(columns, how)
}

// keyFeed is stdin fed by the test, and by the screen's replies to cursor
// queries, as a terminal's are.
type keyFeed struct{ ch chan string }

func (k *keyFeed) Read(p []byte) (int, error) {
	s, ok := <-k.ch
	if !ok {
		return 0, io.EOF
	}
	return copy(p, s), nil
}

// resizeText is long enough for four rows at 80 columns, its last row
// longer than 43 columns, so the cursor's column after a narrowing to 43
// tells a cut row from a re-wrapped one.
var resizeText = strings.Repeat("words fill ", 25) + "the last row of the prompt here"

func trimmed(rows []string) []string {
	out := make([]string, len(rows))
	for i, r := range rows {
		out[i] = strings.TrimRight(r, " ")
	}
	return out
}

// resizeRun types resizeText on a screen 80 columns wide below a header,
// narrows the screen to 43 columns as how says, sends the resize notice,
// and waits for the screen to show the line laid out at 43: below the
// header when the header can still be on screen (wantTop false), else from
// the screen's first row. It fails the test if that never shows.
func resizeRun(t *testing.T, how termtest.Narrowing, answer, wantTop bool, during string) {
	t.Helper()
	ls := &lockedScreen{s: termtest.New(80, 20)}
	if _, err := ls.Write([]byte("header\r\n")); err != nil {
		t.Fatal(err)
	}
	keys := &keyFeed{ch: make(chan string, 16)}
	if answer {
		ls.s.Answer = func(reply string) { keys.ch <- reply }
		if during != "" { // typed after the query, before the reply
			ls.s.Answer = func(reply string) {
				go func() {
					keys.ch <- during
					time.Sleep(20 * time.Millisecond)
					keys.ch <- reply
				}()
			}
		}
	}
	var width atomic.Int64
	width.Store(80)
	resize := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(t.Context())
	var r Reader
	r.Start(ctx, keys)
	e := New(&r, ls, io.Discard, Options{Width: func() int { return int(width.Load()) },
		Height: func() int { return 20 }, Resize: resize})
	done := make(chan struct{})
	go func() {
		_, _ = e.ReadPrompt(ctx)
		close(done)
	}()
	defer func() {
		cancel()
		<-done
	}()
	keys.ch <- resizeText
	text := []rune(resizeText)
	before := layout("> ", text, len(text), 80)
	if last := before.rows[len(before.rows)-1]; len(last) < 44 {
		t.Fatalf("the test's text: its last row at 80 is %d columns, want more than 43", len(last))
	}
	wait := func(want []string, top, row, col int) bool {
		end := time.Now().Add(3 * time.Second)
		for time.Now().Before(end) {
			lines, r, c := ls.look()
			if slices.Equal(lines[top:top+len(want)], want) && slices.Equal(lines[top+len(want):], make([]string, len(lines)-top-len(want))) &&
				r == row && c == col {
				return true
			}
			time.Sleep(5 * time.Millisecond)
		}
		return false
	}
	if !wait(trimmed(before.rows), 1, 1+before.row, before.column) {
		lines, r, c := ls.look()
		t.Fatalf("the line was not drawn at 80: %q, cursor (%d, %d)", lines, r, c)
	}
	ls.resize(43, how)
	width.Store(43)
	resize <- struct{}{}
	text = []rune(resizeText + during)
	after := layout("> ", text, len(text), 43)
	top := 1
	if wantTop {
		top = 0
	}
	if !wait(trimmed(after.rows), top, top+after.row, after.column) {
		lines, r, c := ls.look()
		t.Fatalf("after the resize, screen %q, cursor (%d, %d); want %q from row %d, the cursor at (%d, %d)",
			lines, r, c, trimmed(after.rows), top, top+after.row, after.column)
	}
}

// A resize redraws the line at the new width at once, with no key, in each
// kind of terminal 0030-PLAN P1 measured: rows re-wrapped and the cursor
// moved down; rows re-wrapped and the cursor kept on its row, the top
// pushed above the screen (drawn from the screen's first row); rows cut
// (0030-MADR D2, as amended).
func TestResizeRedrawsAtOnce(t *testing.T) {
	t.Setenv("XTERM_VERSION", "")
	t.Run("re-wrapped, the cursor down", func(t *testing.T) { resizeRun(t, termtest.ReflowDown, true, false, "") })
	t.Run("re-wrapped, the cursor kept", func(t *testing.T) { resizeRun(t, termtest.ReflowKeepRow, true, true, "") })
	t.Run("cut, told by the cursor's column", func(t *testing.T) { resizeRun(t, termtest.Cut, true, false, "") })
}

// A terminal that does not answer the cursor query is redrawn as one that
// re-wraps, once the wait is over.
func TestResizeWithoutAReply(t *testing.T) {
	t.Setenv("XTERM_VERSION", "")
	resizeRun(t, termtest.ReflowDown, false, false, "")
}

// Keys typed while the cursor report is awaited wait with it, and are put
// in the line once it is drawn again at the new width; drawn before, they
// would be drawn where the line no longer is.
func TestKeysTypedWhileTheReportIsAwaited(t *testing.T) {
	t.Setenv("XTERM_VERSION", "")
	resizeRun(t, termtest.ReflowDown, true, false, " and more")
}

// A cursor report typed, or arriving late, is no key: nothing of it is put
// in the line.
func TestCursorReportIsNoText(t *testing.T) {
	s := termtest.New(40, 4)
	subs := onScreen(t, s, strings.NewReader("ab\x1b[12;40Rcd\r"), Options{})
	if !slices.Equal(subs, []string{"abcd"}) {
		t.Fatalf("sent %q, want [abcd]", subs)
	}
}
