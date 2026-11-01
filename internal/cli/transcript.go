package cli

import (
	"sync"
	"time"

	"github.com/maccavelli/go-tui-lib/v2/msgcat"
	"github.com/maccavelli/go-tui-lib/v2/stream"
	tuitranscript "github.com/maccavelli/go-tui-lib/v2/transcript"

	"github.com/maccavelli/gobble-cli/internal/cli/editor"
	"github.com/maccavelli/gobble-cli/internal/cli/render"
	"github.com/maccavelli/gobble-cli/personality"
)

// transcript is what the line session has shown on a terminal, kept by one
// go-tui-lib Transcript, so that once a resize settles the conversation is
// reprinted with no word cut (0030-MADR D3, D4; 0035-MADR D14 step 5b): the
// newest 4 MB of pieces, a reprint the newest 10,000 rows, the library's
// defaults. The Transcript, and every reply's Printer it keeps, are used
// by the editor's, the view's, the agent's, the permission prompt's and the
// log's goroutines, so each call on them goes through mu, by AddReply's
// contract.
type transcript struct {
	mu sync.Mutex
	tr *tuitranscript.Transcript
	// voice speaks the notice that earlier output is not shown again; nil,
	// or a nil voice, is boring's.
	voice func() *personality.Voice
}

// growing is a piece whose source grows while it is shown, such as a
// thinking block; its source is read and written under the transcript's
// lock.
type growing struct{ src string }

// newTranscript keeps what is shown on a terminal terminal columns wide.
func newTranscript(terminal int, o ...tuitranscript.Option) *transcript {
	t := &transcript{}
	notice := msgcat.CatalogFunc(func(id msgcat.MessageID) (string, bool) {
		if id != tuitranscript.MsgEarlierOutput {
			return "", false
		}
		var v *personality.Voice
		if t.voice != nil {
			v = t.voice()
		}
		return v.Text(personality.TranscriptEarlierOutput, nil), true
	})
	t.tr = tuitranscript.NewTranscript(append([]tuitranscript.Option{tuitranscript.WithCatalog(notice)}, o...)...)
	t.tr.Resize(reprintWidth(terminal), time.Now())
	return t
}

// reprintWidth is the width the conversation is drawn again at on a
// terminal that many columns wide: the editor's, at most 120 columns, the
// widest a reply or a message is drawn (0035-PLAN PQ8).
func reprintWidth(terminal int) int {
	if terminal <= 0 {
		return editor.DefaultWidth
	}
	return min(terminal, editor.MaxWidth)
}

// setVoice says whose words the notice is in.
func (t *transcript) setVoice(voice func() *personality.Voice) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.voice = voice
}

// fixed keeps s as written: output whose layout gobble does not make. It
// may hold SGR, which the library's AddText would clean away.
func (t *transcript) fixed(s string) {
	if s != "" {
		t.drawn(len(s), func(int) string { return s })
	}
}

// message keeps one of gobble's messages, drawn as render.Message draws it
// at the terminal's width, to 120 columns (0028-MADR D6).
func (t *transcript) message(s string) {
	t.drawn(len(s), func(width int) string { return render.Message(s, min(width, 120)) + "\n" })
}

// drawn keeps a piece drawn at a width by at, such as /help's list or a
// line the editor sent. Its size is the library's to measure.
func (t *transcript) drawn(_ int, at func(width int) string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tr.AddDrawn(at)
}

// grow starts a piece drawn at a width by at from its source, which grows
// by add. The library counts its size once, when it starts, so what it
// grows by is not counted toward the bound (0035-PLAN PQ9).
func (t *transcript) grow(at func(src string, width int) string) *growing {
	g := &growing{}
	t.drawn(0, func(width int) string { return at(g.src, width) })
	return g
}

// add appends s to g's source.
func (t *transcript) add(g *growing, s string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	g.src += s
}

// reply keeps a reply's Printer, drawn again by its At at a reprint's
// width. From then on every call on p goes through do.
func (t *transcript) reply(p *stream.Printer) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tr.AddReply(p)
}

// do runs f under the lock: each call on a kept reply's Printer goes
// through it.
func (t *transcript) do(f func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	f()
}

// Resize notices that the terminal is now terminal columns wide; a
// reprint is due once resizes settle (editor.Reprinter).
func (t *transcript) Resize(terminal int, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.tr.Resize(reprintWidth(terminal), now)
}

// Timeout is when the settle ends, while resizes settle (editor.Reprinter).
func (t *transcript) Timeout() (time.Time, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tr.Timeout()
}

// HandleTimeout is the reprint, once resizes settled and the caller is
// idle, or "" (editor.Reprinter).
func (t *transcript) HandleTimeout(now time.Time, idle bool) string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.tr.HandleTimeout(now, idle)
}
