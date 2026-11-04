package jsonl

import (
	"bytes"
	"encoding/json/v2"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/session"
)

// benchEntries is how many entries benchFile holds.
const benchEntries = 2000

// benchFile is a session file: a v3 header and benchEntries user messages of
// about 512 bytes of text each.
func benchFile(b *testing.B) []byte {
	b.Helper()
	var buf bytes.Buffer
	write := func(v any) {
		line, err := json.Marshal(v)
		if err != nil {
			b.Fatal(err)
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	write(header("s1", "/w"))
	text := strings.Repeat("y", 512)
	parent := ""
	for i := range benchEntries {
		write(userEntry(entryID(i), parent, text))
		parent = entryID(i)
	}
	return buf.Bytes()
}

// BenchmarkParse measures reading a session file's bytes into its entries
// (0037-MADR D41).
func BenchmarkParse(b *testing.B) {
	data := benchFile(b)
	var (
		entries []session.Entry
		err     error
	)
	b.SetBytes(int64(len(data)))
	b.ReportAllocs()
	for b.Loop() {
		_, entries, _, err = parse(data)
	}
	if err != nil || len(entries) != benchEntries {
		b.Fatalf("parse: %d entries, %v; want %d", len(entries), err, benchEntries)
	}
}
