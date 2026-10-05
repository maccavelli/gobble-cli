package render

import (
	"encoding/json/v2"
	"os"
	"slices"
	"strings"
	"testing"
)

// stream pushes chunks through a Stream and returns every non-empty release,
// then the flushed remainder if any (goose's test helper).
func stream(chunks []string) []string {
	var m Stream
	var out []string
	for _, c := range chunks {
		if s := m.Push(c); s != "" {
			out = append(out, s)
		}
	}
	if rest := m.Flush(); rest != "" {
		out = append(out, rest)
	}
	return out
}

func TestStreamFixtures(t *testing.T) {
	b, err := os.ReadFile("testdata/mdstream.json")
	if err != nil {
		t.Fatal(err)
	}
	var fx struct {
		Cases []struct {
			Name   string   `json:"name"`
			Chunks []string `json:"chunks"`
			Want   []string `json:"want"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(b, &fx); err != nil {
		t.Fatal(err)
	}
	if len(fx.Cases) == 0 {
		t.Fatal("no fixtures")
	}
	for _, c := range fx.Cases {
		t.Run(c.Name, func(t *testing.T) {
			if got := stream(c.Chunks); !slices.Equal(got, c.Want) {
				t.Fatalf("releases = %q\nwant      %q", got, c.Want)
			}
		})
	}
}

// Any chunking of an input must release exactly the input, in order.
func TestStreamReassembles(t *testing.T) {
	corpus := "# Title\n\nSome **bold** and `code`.\n\n" +
		"```python\nfor i in range(3):\n    print(i)\n```\n\n" +
		"A [link](https://example.com) and *italic*.\n" +
		"| a | b |\n|---|---|\n| 1 | 2 |\n\ntail"
	for _, size := range []int{1, 2, 3, 7, 16, len(corpus)} {
		var chunks []string
		for i := 0; i < len(corpus); i += size {
			chunks = append(chunks, corpus[i:min(i+size, len(corpus))])
		}
		if got := strings.Join(stream(chunks), ""); got != corpus {
			t.Fatalf("chunk size %d: reassembled %q", size, got)
		}
	}
}

// The checkpointed scan must release exactly what a full rescan releases,
// over random documents and random chunk boundaries (goose's differential
// test, same generator and seed).
func TestStreamIncrementalMatchesRescan(t *testing.T) {
	tokens := []string{
		"text ", "word", "é✓", "\n", "\n\n", " ", "**", "*", "___", "~~", "`", "``", "```",
		"```rust\n", "~~~", "#", "## ", "|", "| a | b |\n", "[", "](", ")", "![", "]", `\*`,
		"code line\n", "```\n",
	}
	seed := uint64(0x00C0_FFEE)
	next := func() int {
		seed = seed*6364136223846793005 + 1442695040888963407
		return int(seed >> 33)
	}
	for c := range 500 {
		var doc strings.Builder
		for range next()%80 + 1 {
			doc.WriteString(tokens[next()%len(tokens)])
		}
		d := doc.String()
		var inc, full Stream
		var incOut, fullOut []string
		for i := 0; i < len(d); {
			j := min(i+next()%9+1, len(d))
			for j < len(d) && !isRuneStart(d[j]) {
				j++
			}
			chunk := d[i:j]
			i = j
			incOut = append(incOut, inc.Push(chunk))
			full.cp = checkpoint{}
			fullOut = append(fullOut, full.Push(chunk))
			if inc.buf != full.buf {
				t.Fatalf("case %d: buffers diverged, doc=%q", c, d)
			}
		}
		incOut = append(incOut, inc.Flush())
		fullOut = append(fullOut, full.Flush())
		if !slices.Equal(incOut, fullOut) {
			t.Fatalf("case %d: releases diverged, doc=%q", c, d)
		}
	}
}

func isRuneStart(b byte) bool { return b&0xC0 != 0x80 }
