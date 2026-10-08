package builtin

import (
	"strings"
	"testing"
)

func TestCountingLines(t *testing.T) {
	for in, want := range map[string]int{"": 0, "a": 1, "a\n": 1, "a\nb": 2, "a\n\n": 2} {
		if got := len(countingLines(in)); got != want {
			t.Errorf("countingLines(%q) has %d lines, want %d", in, got, want)
		}
	}
}

func TestTruncateTail(t *testing.T) {
	got := truncateTail("a\nb\nc\nd\n", 2, 100)
	if got.content != "c\nd" || got.by != cutLines || got.totalLines != 4 || got.outputLines != 2 {
		t.Fatalf("by lines: %+v", got)
	}
	got = truncateTail("short\n"+strings.Repeat("\u00E9", 10), 10, 5)
	if got.content != "\u00E9\u00E9" || !got.lastLinePartial || got.by != cutBytes {
		t.Fatalf("partial last line: %+v", got)
	}
	if got = truncateTail("a\nb\n", 10, 100); got.truncated() || got.content != "a\nb\n" {
		t.Fatalf("within bounds: %+v", got)
	}
}

// truncateLine counts characters, not bytes or UTF-16 units.
func TestTruncateLine(t *testing.T) {
	if got, cut := truncateLine("abcdef", 3); got != "abc\u2026 (line cut at 3 characters)" || !cut {
		t.Fatalf("got %q, %v", got, cut)
	}
	if got, cut := truncateLine("abc", 3); got != "abc" || cut {
		t.Fatalf("got %q, %v", got, cut)
	}
	if got, _ := truncateLine("a\U0001F600bc", 2); got != "a\U0001F600\u2026 (line cut at 2 characters)" {
		t.Fatalf("an emoji is one character: got %q", got)
	}
}
