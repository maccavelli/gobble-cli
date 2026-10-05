package editor

import (
	"bytes"
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
)

// run feeds input to a new Editor and collects submissions until an error.
func run(t *testing.T, in io.Reader, opts Options) (subs []string, out, errw *bytes.Buffer, err error) {
	t.Helper()
	out, errw = new(bytes.Buffer), new(bytes.Buffer)
	e := New(in, out, errw, opts)
	for {
		s, err := e.ReadPrompt(t.Context())
		if err != nil {
			return subs, out, errw, err
		}
		subs = append(subs, s)
	}
}

// measurement10 is Console Host's byte stream for a three-line paste
// (0008-MADR measurement 10), followed by the user's Enter.
const measurement10 = "\x1b[200~line1\rline2\rline3\x1b[201~\r"

func TestReadPrompt(t *testing.T) {
	cases := []struct {
		name string
		in   io.Reader
		want []string
		err  error
	}{
		{"pasted lines are one submission", strings.NewReader(measurement10), []string{"line1\nline2\nline3"}, io.EOF},
		{"paste markers split across reads", iotest.OneByteReader(strings.NewReader(measurement10)), []string{"line1\nline2\nline3"}, io.EOF},
		{"Ctrl+C clears a non-empty line", strings.NewReader("abc\x03xyz\r"), []string{"xyz"}, io.EOF},
		{"Ctrl+C clears the whole line, not only left of the cursor", strings.NewReader("abcd\x1b[D\x1b[D\x03xy\r"), []string{"xy"}, io.EOF},
		{"Ctrl+C inside a paste is text", strings.NewReader("\x1b[200~a\x03b\x1b[201~\r"), []string{"a\x03b"}, io.EOF},
		{"backslash continues", strings.NewReader("one\\\rtwo\r"), []string{"one\ntwo"}, io.EOF},
		{"empty submissions are skipped", strings.NewReader("\r  \rhi\r"), []string{"hi"}, io.EOF},
		{"/exit ends", strings.NewReader("/exit\rmore\r"), nil, io.EOF},
		{"/quit ends", strings.NewReader("/quit\r"), nil, io.EOF},
		{"Ctrl+D on an empty line ends", strings.NewReader("\x04more\r"), nil, io.EOF},
		{"Ctrl+D on a non-empty line does not end", strings.NewReader("ab\x04\r"), []string{"ab"}, io.EOF},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subs, _, _, err := run(t, tc.in, Options{})
			if !slices.Equal(subs, tc.want) {
				t.Fatalf("submissions = %q (then err %v), want %q", subs, err, tc.want)
			}
			if !errors.Is(err, tc.err) {
				t.Fatalf("err = %v, want %v", err, tc.err)
			}
		})
	}
}

func TestPasteNoticeAndPrompts(t *testing.T) {
	_, out, errw, _ := run(t, strings.NewReader(measurement10+"a\\\rb\r"), Options{})
	if got := errw.String(); got != "… 3 pasted lines\r\n" {
		t.Fatalf("errw = %q", got)
	}
	o := out.String()
	if !strings.HasPrefix(o, "\x1b[?2004h") {
		t.Fatalf("bracketed paste not enabled first: %q", o)
	}
	if !strings.Contains(o, "> ") || !strings.Contains(o, "… ") {
		t.Fatalf("prompt or continuation prompt missing: %q", o)
	}
}

func TestASCIIPrompts(t *testing.T) {
	_, _, errw, _ := run(t, strings.NewReader(measurement10), Options{Prompt: "> ", Continue: "... "})
	if got := errw.String(); got != "... 3 pasted lines\r\n" {
		t.Fatalf("errw = %q", got)
	}
}

func TestRestoreDisablesBracketedPaste(t *testing.T) {
	var out bytes.Buffer
	e := New(strings.NewReader(""), &out, io.Discard, Options{})
	out.Reset()
	e.Restore()
	if got := out.String(); got != "\x1b[?2004l" {
		t.Fatalf("Restore wrote %q", got)
	}
}

type memHistory struct{ entries []string }

func (m *memHistory) Add(e string)      { m.entries = append(m.entries, e) }
func (m *memHistory) Len() int          { return len(m.entries) }
func (m *memHistory) At(idx int) string { return m.entries[len(m.entries)-1-idx] }

func TestHistoryGetsWholeSubmissions(t *testing.T) {
	h := &memHistory{}
	subs, _, _, _ := run(t, strings.NewReader(measurement10+"first\r\x1b[A\r"), Options{History: h})
	want := []string{"line1\nline2\nline3", "first", "first"}
	if !slices.Equal(subs, want) {
		t.Fatalf("submissions = %q, want %q", subs, want)
	}
	if !slices.Equal(h.entries, want) {
		t.Fatalf("history = %q, want %q (one entry per submission)", h.entries, want)
	}
}

func TestTabCallsComplete(t *testing.T) {
	var gotLine string
	var gotPos int
	complete := func(line string, pos int) (string, int, bool) {
		gotLine, gotPos = line, pos
		return "grep ", 5, true
	}
	subs, _, _, _ := run(t, strings.NewReader("gr\t\r"), Options{Complete: complete})
	if gotLine != "gr" || gotPos != 2 {
		t.Fatalf("Complete(%q, %d), want (\"gr\", 2)", gotLine, gotPos)
	}
	if !slices.Equal(subs, []string{"grep "}) {
		t.Fatalf("submissions = %q", subs)
	}
}

func TestContextCancelled(t *testing.T) {
	pr, pw := io.Pipe()
	t.Cleanup(func() {
		if err := pw.Close(); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithCancel(t.Context())
	r := &Reader{}
	r.Start(ctx, pr)
	e := New(r, io.Discard, io.Discard, Options{})
	cancel()
	if _, err := e.ReadPrompt(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}
