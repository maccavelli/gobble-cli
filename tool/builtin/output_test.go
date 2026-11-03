package builtin

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// Past spillCap the file stops, while the counts and the tail go on, and
// the result says where the file was cut (0037-MADR D9).
func TestSpillCap(t *testing.T) {
	old := spillCap
	spillCap = 1 << 20
	t.Cleanup(func() { spillCap = old })

	var all bytes.Buffer
	for i := 0; all.Len() < 4<<20; i++ {
		fmt.Fprintf(&all, "line %08d %s\n", i, strings.Repeat("x", 49))
	}
	out := &spill{dir: t.TempDir()}
	// Chunks that do not divide the cap, so one write straddles it.
	for p := all.Bytes(); len(p) > 0; {
		n := min(3000, len(p))
		_, _ = out.Write(p[:n]) //nolint:errcheck // spill records its own error
		p = p[n:]
	}
	out.close()
	if out.err != nil {
		t.Fatalf("spill error: %v", out.err)
	}

	b, err := os.ReadFile(out.path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) != 1<<20 || !bytes.Equal(b, all.Bytes()[:1<<20]) {
		t.Fatalf("file holds %d bytes, want the output's first %d", len(b), 1<<20)
	}
	if out.total != all.Len() || out.lines() != bytes.Count(all.Bytes(), []byte{'\n'}) {
		t.Fatalf("counted %d bytes and %d lines, want %d and %d", out.total, out.lines(), all.Len(), bytes.Count(all.Bytes(), []byte{'\n'}))
	}
	if len(out.mem) != tailKeep || !bytes.HasSuffix(all.Bytes(), out.mem) {
		t.Fatalf("tail is %d bytes, want the output's last %d", len(out.mem), tailKeep)
	}

	text := outputResult(out, 0, false, "", time.Minute, 10*time.Minute).Text()
	if want := "the output is in " + out.path + "; the file holds the first 1 MiB)"; !strings.Contains(text, want) {
		t.Fatalf("result does not say %q:\n%s", want, text[:min(300, len(text))])
	}
	if last := fmt.Sprintf("line %08d", bytes.Count(all.Bytes(), []byte{'\n'})-1); !strings.Contains(text, last) {
		t.Fatalf("result does not end with %s", last)
	}
}
