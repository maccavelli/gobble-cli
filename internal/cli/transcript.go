package cli

import (
	"slices"
	"strings"
	"sync"

	"github.com/maccavelli/gobble-cli/internal/cli/render"
)

// transcript is what the line session has shown on a terminal, in order,
// as pieces that can be drawn again at a width, so that a resize can
// reprint the conversation with no word cut (0030-MADR D3, D4). It keeps
// at most maxTranscript bytes; past that, the oldest pieces go. It is
// shared by the editor's, the view's and the agent's goroutines.
type transcript struct {
	mu      sync.Mutex
	pieces  []*piece
	size    int
	dropped bool // pieces went, past the bound
}

// piece is one thing shown: its source, which a reply's grows while it
// arrives, its size in bytes, and how it is drawn at a terminal's width.
type piece struct {
	src  string
	size int
	at   func(src string, width int) string
}

// maxTranscript bounds what the transcript keeps, in bytes (0030-MADR D4,
// as revised); maxReprintRows bounds a reprint (D3).
const (
	maxTranscript  = 4 << 20
	maxReprintRows = 10000
)

// fixed keeps s as written: output whose layout gobble does not make.
func (t *transcript) fixed(s string) {
	if s != "" {
		t.open(func(src string, _ int) string { return src }).add(t, s)
	}
}

// message keeps one of gobble's messages, drawn as render.Message draws it
// at the terminal's width, to 120 columns (0028-MADR D6).
func (t *transcript) message(s string) {
	t.open(func(src string, width int) string { return render.Message(src, min(width, 120)) + "\n" }).add(t, s)
}

// drawn keeps a piece of size bytes drawn at a width by at, such as /help's
// list or a line the editor sent.
func (t *transcript) drawn(size int, at func(width int) string) {
	t.open(func(_ string, width int) string { return at(width) }).grow(t, "", size)
}

// open starts a piece drawn by at from its source, empty until added to.
func (t *transcript) open(at func(src string, width int) string) *piece {
	p := &piece{at: at}
	t.mu.Lock()
	t.pieces = append(t.pieces, p)
	t.mu.Unlock()
	return p
}

// add appends s to p's source.
func (p *piece) add(t *transcript, s string) { p.grow(t, s, len(s)) }

func (p *piece) grow(t *transcript, s string, size int) {
	t.mu.Lock()
	defer t.mu.Unlock()
	p.src += s
	p.size += size
	t.size += size
	for t.size > maxTranscript && len(t.pieces) > 1 {
		t.size -= t.pieces[0].size
		t.pieces = t.pieces[1:]
		t.dropped = true
	}
}

// reprintText is t drawn at a terminal's width for a reprint, after a dimmed
// line saying so when earlier output was left out, by the bound or by the
// row limit (0030-MADR D3, D4).
func reprintText(t *transcript, width int, color bool) string {
	s, cut := t.render(width, maxReprintRows)
	if cut {
		s = dimmed("(earlier output is not shown again)", color) + "\n" + s
	}
	return s
}

// render is the transcript drawn at width: the newest pieces back to at
// most maxRows rows, the oldest of them cut to fit, and whether anything
// earlier was left out, by the bound or by maxRows.
func (t *transcript) render(width, maxRows int) (string, bool) {
	t.mu.Lock()
	pieces := make([]piece, len(t.pieces))
	for i, p := range t.pieces {
		pieces[i] = *p
	}
	cut := t.dropped
	t.mu.Unlock()
	var drawn []string
	rows := 0
	for i := len(pieces) - 1; i >= 0 && rows < maxRows; i-- {
		s := pieces[i].at(pieces[i].src, width)
		n := strings.Count(s, "\n")
		if rows+n > maxRows {
			lines := strings.SplitAfter(s, "\n")
			s = strings.Join(lines[len(lines)-(maxRows-rows)-1:], "")
			n, cut = maxRows-rows, true
		}
		drawn = append(drawn, s)
		rows += n
		if i > 0 && rows >= maxRows {
			cut = true
		}
	}
	var b strings.Builder
	for _, s := range slices.Backward(drawn) {
		b.WriteString(s)
	}
	return b.String(), cut
}
