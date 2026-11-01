package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tuitranscript "github.com/maccavelli/go-tui-lib/v2/transcript"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/internal/cli/term"
	"github.com/maccavelli/gobble-cli/personality"
)

// reprintAt is t's reprint at width, as a settled resize to it gives.
func reprintAt(t *transcript, width int) string {
	var out string
	t.do(func() {
		t.tr.Resize(reprintWidth(width), time.Now())
		out = t.tr.Reprint(true)
	})
	return out
}

// reprinted is what a reprint draws after it clears the screen, its line
// ends LF: the notice, when one is due, and the pieces.
func reprinted(out string) string {
	if _, after, ok := strings.Cut(out, "\x1b[H"); ok {
		out = after
	}
	return strings.ReplaceAll(strings.TrimSuffix(out, "\x1b[?2026l"), "\r\n", "\n")
}

// The transcript draws each piece at the reprint's width: fixed text as it
// was written, a message wrapped as gobble's messages are, a piece kept as
// a function of the width by that function (0030-MADR D4).
func TestTranscriptRendersAtAWidth(t *testing.T) {
	tr := newTranscript(80)
	tr.fixed("tool output that is wider than twenty columns, kept as written\n")
	tr.message("Warning: a message long enough to be wrapped at twenty columns")
	tr.drawn(10, func(width int) string { return strings.Repeat("-", width) + "\n" })
	got := reprinted(reprintAt(tr, 20))
	want := "tool output that is wider than twenty columns, kept as written\n" +
		"Warning: a message\n  long enough to be\n  wrapped at twenty\n  columns\n" +
		strings.Repeat("-", 20) + "\n"
	if got != want {
		t.Fatalf("at 20: %q; want %q", got, want)
	}
	if got := reprinted(reprintAt(tr, 40)); !strings.Contains(got, strings.Repeat("-", 40)+"\n") {
		t.Fatalf("at 40 the function was not drawn at 40: %q", got)
	}
}

// The reprint's notice that earlier output is not shown again is in the
// session's voice (0022-MADR; 0035-PLAN P18b step 6).
func TestTranscriptNoticeInTheVoice(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, personality.Boring), 0o750); err != nil {
		t.Fatal(err)
	}
	messages := "## transcript.earlier-output\n\n- older output is gone\n"
	if err := os.WriteFile(filepath.Join(dir, personality.Boring, "messages.md"), []byte(messages), 0o600); err != nil {
		t.Fatal(err)
	}
	set, errs := personality.LoadSet(dir)
	if len(errs) != 0 {
		t.Fatal(errs)
	}
	voice, err := set.New(personality.Options{Name: personality.Boring})
	if err != nil {
		t.Fatal(err)
	}
	tr := newTranscript(80, tuitranscript.WithMaxRows(2))
	for _, s := range []string{"one\n", "two\n", "three\n"} {
		tr.fixed(s)
	}
	if got := reprinted(reprintAt(tr, 60)); got != "(earlier output is not shown again)\ntwo\nthree\n" {
		t.Fatalf("boring's notice: %q", got)
	}
	tr.setVoice(func() *personality.Voice { return voice })
	if got := reprinted(reprintAt(tr, 70)); got != "older output is gone\ntwo\nthree\n" {
		t.Fatalf("the voice's notice: %q", got)
	}
}

// A reply's Printer, kept by the transcript, is written by the view while
// other goroutines add to the transcript, each Add reading every kept
// reply's Source: so it is written under the transcript's lock (go-tui-lib
// AddReply's contract; 0035-PLAN P18b step 2). make race checks it.
func TestReplyPrinterUnderTheLock(t *testing.T) {
	s := term.Stream{TTY: true, Color: true, Width: 80}
	o := &Output{out: io.Discard, err: io.Discard, caps: term.Caps{Out: s, Err: s}, rec: newTranscript(80)}
	v := newView(o, display{width: func() int { return 80 }}, stepClock())
	done := make(chan struct{})
	go func() {
		defer close(done)
		for range 200 {
			o.rec.fixed("a line another goroutine shows\n")
		}
	}()
	for range 200 {
		v.on(acpclient.Update{Kind: acpclient.KindAgentText, Text: "word word word "})
	}
	<-done
	if err := v.finish(acpclient.Result{StopReason: "end_turn"}); err != nil {
		t.Fatal(err)
	}
	if got := reprinted(reprintAt(o.rec, 60)); !strings.Contains(got, "word word word") {
		t.Fatalf("the reply is not kept: %q", got)
	}
}
