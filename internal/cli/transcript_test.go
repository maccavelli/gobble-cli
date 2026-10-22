package cli

import (
	"strings"
	"testing"
)

// The transcript draws each piece at the width asked for: fixed text as it
// was written, a message wrapped as gobble's messages are, a piece kept as
// a function of the width by that function (0030-MADR D4).
func TestTranscriptRendersAtAWidth(t *testing.T) {
	var tr transcript
	tr.fixed("tool output that is wider than twenty columns, kept as written\n")
	tr.message("Warning: a message long enough to be wrapped at twenty columns")
	tr.drawn(10, func(width int) string { return strings.Repeat("-", width) + "\n" })
	got, cut := tr.render(20, maxReprintRows)
	want := "tool output that is wider than twenty columns, kept as written\n" +
		"Warning: a message\n  long enough to be\n  wrapped at twenty\n  columns\n" +
		strings.Repeat("-", 20) + "\n"
	if got != want || cut {
		t.Fatalf("render(20) = %q, %v; want %q, false", got, cut, want)
	}
	if got, _ := tr.render(40, maxReprintRows); !strings.Contains(got, strings.Repeat("-", 40)+"\n") {
		t.Fatalf("render(40) did not draw the function at 40: %q", got)
	}
}

// A reprint is at most maxRows rows, the newest; anything earlier left out
// is said to be (0030-MADR D3).
func TestTranscriptRenderKeepsTheNewestRows(t *testing.T) {
	var tr transcript
	for i := range 30 {
		tr.fixed(strings.Repeat("x", i%7) + "\n")
	}
	tr.fixed("last one\n")
	got, cut := tr.render(80, 10)
	if !cut || strings.Count(got, "\n") != 10 || !strings.HasSuffix(got, "last one\n") {
		t.Fatalf("render(80, 10) = %q, cut %v; want the last 10 rows, cut", got, cut)
	}
}

// A reprint draws only the pieces it shows, newest first, so a long
// conversation is not drawn whole to show its last rows (0030-MADR D3).
func TestTranscriptRenderDrawsOnlyWhatItShows(t *testing.T) {
	var tr transcript
	drawn := 0
	for range 100 {
		tr.drawn(4, func(int) string {
			drawn++
			return "row\n"
		})
	}
	tr.render(80, 10)
	if drawn != 10 {
		t.Fatalf("%d pieces drawn for 10 rows, want 10", drawn)
	}
}

// Past maxTranscript bytes the oldest pieces go, and the reprint says so
// (0030-MADR D4, as revised).
func TestTranscriptBound(t *testing.T) {
	var tr transcript
	row := strings.Repeat("y", 1023) + "\n"
	tr.fixed("the very first\n")
	for range maxTranscript / len(row) {
		tr.fixed(row)
	}
	got, cut := tr.render(2000, 1<<30)
	if strings.Contains(got, "the very first") || !cut {
		t.Fatalf("the oldest piece was kept past %d bytes, or not said to be gone (cut %v)", maxTranscript, cut)
	}
	if n := strings.Count(got, "\n"); n < maxTranscript/len(row)-1 {
		t.Fatalf("only %d rows kept", n)
	}
}
