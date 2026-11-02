package builtin

import (
	"context"
	"encoding/json/jsontext"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/tool"
)

// lines is "Line 1" … "Line n" joined by \n, with no final newline.
func lines(n int, suffix string) string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("Line %d%s", i+1, suffix)
	}
	return strings.Join(out, "\n")
}

func readOf(t *testing.T, name, content string, args map[string]any) tool.Result {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	args["path"] = name
	return call(t, Read(), tool.Env{Cwd: dir}, args)
}

func TestReadText(t *testing.T) {
	cases := []struct {
		name, content string
		args          map[string]any
		want          string // the whole output, or a suffix when prefix is set
		suffix        bool
	}{
		{"whole file", "Hello, world!\nLine 2\nLine 3", map[string]any{},
			"1: Hello, world!\n2: Line 2\n3: Line 3\n(end of file, 3 lines)", false},
		{"a final newline is not a line", "a\nb\n", map[string]any{},
			"1: a\n2: b\n(end of file, 2 lines)", false},
		{"BOM and CR are not shown", "\uFEFFa\r\nb\r\n", map[string]any{},
			"1: a\n2: b\n(end of file, 2 lines)", false},
		{"empty", "", map[string]any{}, "(empty file)", false},
		{"one line", "only\n", map[string]any{}, "1: only\n(end of file, 1 line)", false},
		{"line limit", lines(2500, ""), map[string]any{},
			"2000: Line 2000\n(lines 1-2000 of 2500; continue with offset=2001)", true},
		// Numbered lines 1-9 are 212 bytes, 10-99 are 214, the rest 216: 99
		// lines take 21,168 bytes, and 139 more of 216 fit in the 51,200,
		// so line 238 is the last.
		{"byte limit", lines(500, ": "+strings.Repeat("x", 200)), map[string]any{},
			"\n(lines 1-238 of 500, 50 KB limit; continue with offset=239)", true},
		{"offset", lines(100, ""), map[string]any{"offset": 51},
			"100: Line 100\n(end of file, 100 lines)", true},
		{"limit", lines(100, ""), map[string]any{"limit": 10},
			"10: Line 10\n(lines 1-10 of 100; continue with offset=11)", true},
		{"offset and limit", lines(100, ""), map[string]any{"offset": 41, "limit": 20},
			"60: Line 60\n(lines 41-60 of 100; continue with offset=61)", true},
		{"long line", "short\n" + strings.Repeat("y", 3000), map[string]any{"offset": 2},
			"2: " + strings.Repeat("y", 2000) + "\u2026 (line cut at 2000 characters)\n(end of file, 2 lines)", false},
		// A streamed read holds only the start of a long line: enough
		// for 2000 characters of 4 bytes, and the cut still shows.
		{"long line past what is held", strings.Repeat("y", 20000) + "\nz", map[string]any{},
			"1: " + strings.Repeat("y", 2000) + "\u2026 (line cut at 2000 characters)\n2: z\n(end of file, 2 lines)", false},
		{"long line of 4-byte characters", strings.Repeat("\U0001F600", 3000), map[string]any{},
			"1: " + strings.Repeat("\U0001F600", 2000) + "\u2026 (line cut at 2000 characters)\n(end of file, 1 line)", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := readOf(t, "f.txt", tc.content, tc.args)
			got := r.Text()
			ok := got == tc.want
			if tc.suffix {
				ok = strings.HasSuffix(got, tc.want)
			}
			if r.IsError || !ok {
				t.Fatalf("output ends %q, want %q", got[max(0, len(got)-len(tc.want)-20):], tc.want)
			}
		})
	}
}

// The offset of the first line shown is its number.
func TestReadNumbersFromTheOffset(t *testing.T) {
	r := readOf(t, "f.txt", lines(100, ""), map[string]any{"offset": 51, "limit": 1})
	if !strings.HasPrefix(r.Text(), "51: Line 51\n") {
		t.Fatalf("output %q, want it to start with line 51", r.Text())
	}
}

func TestReadErrors(t *testing.T) {
	cases := []struct {
		name, file, content, path string
		args                      map[string]any
		want                      string
	}{
		{"past the end", "f.txt", "Line 1\nLine 2\nLine 3", "f.txt", map[string]any{"offset": 100},
			"read f.txt: offset 100 is past the end of the file (3 lines)"},
		{"binary", "f.bin", "PK\x03\x04\x00\x00data", "f.bin", map[string]any{},
			"read f.bin: a binary file; read cannot show it"},
		{"did you mean", "main_test.go", "package main", "main", map[string]any{},
			"read main: no such file; did you mean main_test.go?"},
		{"nothing similar", "main_test.go", "package main", "zzz", map[string]any{},
			"read zzz: no such file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tc.file), []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}
			tc.args["path"] = tc.path
			r := call(t, Read(), tool.Env{Cwd: dir}, tc.args)
			if !r.IsError || r.Text() != tc.want {
				t.Fatalf("result %q (isError %v), want the error %q", r.Text(), r.IsError, tc.want)
			}
		})
	}
}

func TestReadDirectory(t *testing.T) {
	dir := t.TempDir()
	for _, f := range []string{"b.txt", "c.txt"} {
		if err := os.WriteFile(filepath.Join(dir, f), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "A"), 0o750); err != nil {
		t.Fatal(err)
	}
	env := tool.Env{Cwd: dir}
	if got := call(t, Read(), env, map[string]any{"path": "."}).Text(); got != "A/\nb.txt\nc.txt\n(end of directory, 3 entries)" {
		t.Fatalf("listing %q", got)
	}
	if got := call(t, Read(), env, map[string]any{"path": ".", "limit": 2}).Text(); got != "A/\nb.txt\n(entries 1-2 of 3; continue with offset=3)" {
		t.Fatalf("paged listing %q", got)
	}
	empty := filepath.Join(dir, "A")
	if got := call(t, Read(), tool.Env{Cwd: empty}, map[string]any{"path": "."}).Text(); got != "(empty directory)" {
		t.Fatalf("empty listing %q", got)
	}
}

// An image is found by its magic bytes, not its name, and gets the note.
// An animated PNG is an image too.
func TestReadImages(t *testing.T) {
	png := append(append([]byte{}, pngSignature...), 0, 0, 0, 13, 'I', 'H', 'D', 'R', 0, 0, 0, 1, 0, 0, 0, 1, 8, 6, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 'I', 'D', 'A', 'T')
	apng := append(append([]byte{}, png[:33]...), 0, 0, 0, 8, 'a', 'c', 'T', 'L')
	cases := map[string][]byte{
		"image/jpeg": {0xff, 0xd8, 0xff, 0xe0, 0, 0x10, 'J', 'F', 'I', 'F'},
		"image/png":  png,
		"image/gif":  []byte("GIF89a\x01\x00\x01\x00"),
		"image/webp": []byte("RIFF\x24\x00\x00\x00WEBPVP8 "),
		"image/bmp":  bmp(),
	}
	for mime, b := range cases {
		r := readOf(t, "pic.txt", string(b), map[string]any{})
		if want := "Read image file [" + mime + "]\n" + imageNotSent; r.IsError || r.Text() != want {
			t.Errorf("%s: %q, want %q", mime, r.Text(), want)
		}
	}
	if got := imageType(apng); got != "image/png" {
		t.Errorf("an animated PNG sniffed as %q", got)
	}
	if got := imageType([]byte("BM is a text file that starts with BM")); got != "" {
		t.Errorf("text starting BM sniffed as %q", got)
	}
}

func TestBinaryContent(t *testing.T) {
	for in, want := range map[string]bool{
		"":                               false,
		"plain text\twith tabs\r\n":      false,
		"a\x00b":                         true,
		"\x01\x02\x03\x04ab":             true,
		"\x01" + strings.Repeat("a", 10): false,
	} {
		if got := binaryContent([]byte(in)); got != want {
			t.Errorf("binaryContent(%q) = %v, want %v", in, got, want)
		}
	}
}

// bmp is a 1x1 24-bit BMP header with a 40-byte DIB header.
func bmp() []byte {
	b := make([]byte, 58)
	copy(b, "BM")
	b[2] = 58
	b[10] = 54
	b[14] = 40
	b[26], b[28] = 1, 24
	return b
}

// read of a file that is not valid UTF-8 returns its text with U+FFFD for
// each invalid byte, where it panicked (0039-MADR defect 1; Confirmation 2).
func TestReadInvalidUTF8(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "latin1.txt"), []byte("caf\xe9 au lait\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	r := call(t, Read(), tool.Env{Cwd: dir}, map[string]any{"path": "latin1.txt"})
	if r.IsError || !strings.Contains(r.Text(), "caf\uFFFD au lait") {
		t.Fatalf("read = %+v, text %q; want U+FFFD for the invalid byte", r, r.Text())
	}
}

// fifo makes a FIFO at p with mkfifo(1), or skips where the platform
// cannot make one (0037-PLAN C10).
func fifo(t *testing.T, p string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("Windows has no FIFOs")
	}
	if out, err := exec.CommandContext(t.Context(), "mkfifo", p).CombinedOutput(); err != nil { //nolint:gosec // G204: a test's temporary path
		t.Skipf("cannot make a FIFO here: %v %s", err, out)
	}
}

// outcome is a call run by runBounded, and how long it took.
type outcome struct {
	r    tool.Result
	err  error
	took time.Duration
}

// runBounded runs tl with args and returns its outcome, failing the test
// when the call is still running at deadline. A call blocked for good is
// left behind.
func runBounded(t *testing.T, tl tool.Tool, env tool.Env, args string, deadline time.Duration) outcome {
	t.Helper()
	done := make(chan outcome, 1)
	start := time.Now()
	go func() {
		r, err := tl.Run(context.Background(), tool.Call{ID: "c1", Name: tl.Spec().Name, Args: jsontext.Value(args)}, env)
		done <- outcome{r, err, time.Since(start)}
	}()
	select {
	case o := <-done:
		return o
	case <-time.After(deadline):
		t.Fatalf("%s %s was still running after %v", tl.Spec().Name, args, deadline)
		return outcome{}
	}
}

// read of a FIFO is refused at once, not left waiting for a writer
// (0037-MADR D7).
func TestReadRefusesFIFO(t *testing.T) {
	dir := t.TempDir()
	fifo(t, filepath.Join(dir, "pipe"))
	o := runBounded(t, Read(), tool.Env{Cwd: dir}, `{"path":"pipe"}`, 2*time.Second)
	if o.err != nil || !o.r.IsError || o.r.Text() != "read pipe: not a regular file" {
		t.Fatalf("read of a FIFO = %q (isError %v), %v; want the error \"read pipe: not a regular file\"", o.r.Text(), o.r.IsError, o.err)
	}
	if o.took > 100*time.Millisecond {
		t.Fatalf("read of a FIFO took %v, want under 100ms", o.took)
	}
}

// edit and write read through read's open, so they refuse a FIFO too
// (0037-PLAN deviation 6).
func TestEditRefusesFIFO(t *testing.T) {
	dir := t.TempDir()
	fifo(t, filepath.Join(dir, "pipe"))
	for _, c := range []struct {
		tl         tool.Tool
		args, want string
	}{
		{Edit(), `{"path":"pipe","edits":[{"oldText":"a","newText":"b"}]}`, "edit pipe: not a regular file"},
		{Write(), `{"path":"pipe","content":"b"}`, "write pipe: not a regular file"},
	} {
		o := runBounded(t, c.tl, tool.Env{Cwd: dir}, c.args, time.Second)
		if o.err != nil || !o.r.IsError || o.r.Text() != c.want {
			t.Fatalf("%s of a FIFO = %q (isError %v), %v; want the error %q", c.tl.Spec().Name, o.r.Text(), o.r.IsError, o.err, c.want)
		}
	}
}

// A read with a limit holds the lines it shows, not the file: an 8 MiB
// file read with limit 50 allocates under 2 MiB, and its total is right
// (0037-MADR D7).
func TestReadStreamsLimit(t *testing.T) {
	dir := t.TempDir()
	const lineBytes, size = 64, 8 << 20
	if err := os.WriteFile(filepath.Join(dir, "big.txt"), []byte(strings.Repeat(strings.Repeat("x", lineBytes-1)+"\n", size/lineBytes)), 0o600); err != nil {
		t.Fatal(err)
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	r := call(t, Read(), tool.Env{Cwd: dir}, map[string]any{"path": "big.txt", "limit": 50})
	runtime.ReadMemStats(&after)
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > 2<<20 {
		t.Fatalf("read of 8 MiB with limit 50 allocated %d bytes, want under 2 MiB", alloc)
	}
	want := fmt.Sprintf("50: %s\n(lines 1-50 of %d; continue with offset=51)", strings.Repeat("x", lineBytes-1), size/lineBytes)
	if r.IsError || !strings.HasSuffix(r.Text(), want) {
		t.Fatalf("output ends %q, want %q", r.Text()[max(0, len(r.Text())-len(want)-20):], want)
	}
}
