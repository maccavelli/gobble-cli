package editor

import (
	"bytes"
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/inline"
)

// session drives an Editor through a pipe, as stdin would, inside a
// synctest bubble so the exit window runs on the fake clock.
type session struct {
	pw   *io.PipeWriter
	r    *inline.Reader
	e    *Editor
	errw *bytes.Buffer
	wg   sync.WaitGroup
}

func newSession(t *testing.T) *session {
	t.Helper()
	pr, pw := io.Pipe()
	errw := new(bytes.Buffer)
	r := inline.NewReader(pr)
	return &session{pw: pw, r: r, e: New(r, io.Discard, errw, Options{}), errw: errw}
}

// feed writes each step's bytes, sleeping before the ones with a delay.
func (s *session) feed(t *testing.T, steps ...step) {
	s.wg.Go(func() {
		for _, st := range steps {
			time.Sleep(st.after)
			if _, err := s.pw.Write([]byte(st.bytes)); err != nil {
				t.Error(err)
				return
			}
		}
	})
}

func (s *session) close(t *testing.T) {
	s.wg.Wait()
	if err := s.pw.Close(); err != nil {
		t.Error(err)
	}
	// The reader's goroutine reads only when asked: one more read takes
	// the pipe's end, and it returns.
	if _, err := s.r.Read(context.Background(), make([]byte, 1)); !errors.Is(err, io.EOF) {
		t.Errorf("the read after the end: %v, want io.EOF", err)
	}
	synctest.Wait()
}

type step struct {
	after time.Duration
	bytes string
}

func TestCtrlCTwiceInsideWindowExits(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSession(t)
		s.feed(t, step{0, "\x03"}, step{1999 * time.Millisecond, "\x03"})
		if _, err := s.e.ReadPrompt(t.Context()); !errors.Is(err, ErrExit) {
			t.Fatalf("err = %v, want ErrExit", err)
		}
		if got := s.errw.String(); got != "Press Ctrl+C again to exit\r\n" {
			t.Fatalf("errw = %q", got)
		}
		s.close(t)
	})
}

func TestCtrlCWindowExpires(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSession(t)
		s.feed(t, step{0, "\x03"}, step{2 * time.Second, "\x03"}, step{time.Second, "x\r"})
		got, err := s.e.ReadPrompt(t.Context())
		if err != nil || got != "x" {
			t.Fatalf("ReadPrompt = %q, %v; want \"x\" after the window expired", got, err)
		}
		if n := strings.Count(s.errw.String(), "Press Ctrl+C again to exit"); n != 2 {
			t.Fatalf("notice printed %d times, want 2: %q", n, s.errw.String())
		}
		s.close(t)
	})
}

func TestCtrlCOnTextDisarmsWindow(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		s := newSession(t)
		s.feed(t, step{0, "\x03"}, step{100 * time.Millisecond, "ab\x03"}, step{100 * time.Millisecond, "\x03"},
			step{100 * time.Millisecond, "ok\r"})
		got, err := s.e.ReadPrompt(t.Context())
		if err != nil || got != "ok" {
			t.Fatalf("ReadPrompt = %q, %v; want \"ok\": clearing text must disarm the window", got, err)
		}
		s.close(t)
	})
}

// stdinFiles hold or touch the stdin reader. None may call Close (F20);
// inline.Reader, the reader itself, never closes its input.
var stdinFiles = []string{"editor.go"}

func TestNoCloseOnStdin(t *testing.T) {
	for _, name := range stdinFiles {
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Close" {
				t.Errorf("%s calls Close: stdin is never closed (0008-MADR F20)", name)
			}
			return true
		})
	}
}
