package acptest

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

// Direction is which way a frame travelled.
type Direction uint8

// Directions.
const (
	ClientToAgent Direction = iota
	AgentToClient
)

func (d Direction) String() string {
	if d == AgentToClient {
		return "agent→client"
	}
	return "client→agent"
}

// Frame is one JSON-RPC message as written to the wire.
type Frame struct {
	Dir Direction
	Raw jsontext.Value
}

// Recorder records every frame written by either side, in write order. It
// is safe for concurrent use.
type Recorder struct {
	mu      sync.Mutex
	frames  []Frame
	partial [2][]byte
	changed chan struct{} // closed and replaced on each new frame
}

// tap returns a writer that records what passes through it to w.
func (r *Recorder) tap(dir Direction, w io.Writer) io.Writer {
	return writerFunc(func(p []byte) (int, error) {
		r.observe(dir, p)
		return w.Write(p)
	})
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) { return f(p) }

// observe splits written bytes into newline-delimited frames; a line split
// across writes is joined.
func (r *Recorder) observe(dir Direction, p []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	buf := slices.Concat(r.partial[dir], p)
	added := false
	for {
		i := bytes.IndexByte(buf, '\n')
		if i < 0 {
			break
		}
		if line := bytes.TrimSpace(buf[:i]); len(line) > 0 {
			r.frames = append(r.frames, Frame{Dir: dir, Raw: jsontext.Value(slices.Clone(line))})
			added = true
		}
		buf = buf[i+1:]
	}
	r.partial[dir] = slices.Clone(buf)
	if added && r.changed != nil {
		close(r.changed)
		r.changed = nil
	}
}

// Frames returns the frames recorded so far.
func (r *Recorder) Frames() []Frame {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Frame, len(r.frames))
	for i, f := range r.frames {
		out[i] = Frame{Dir: f.Dir, Raw: f.Raw.Clone()}
	}
	return out
}

// Wait blocks until at least n frames are recorded or ctx is done. Use it
// before reading a transcript that ends in a notification, which arrives
// asynchronously.
func (r *Recorder) Wait(ctx context.Context, n int) error {
	for {
		r.mu.Lock()
		if len(r.frames) >= n {
			r.mu.Unlock()
			return nil
		}
		if r.changed == nil {
			r.changed = make(chan struct{})
		}
		ch := r.changed
		r.mu.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			return fmt.Errorf("acptest: waiting for %d frames: %w", n, ctx.Err())
		}
	}
}

// Transcript is one line per frame: the direction, a space, and the frame
// as RFC 8785 canonical JSON, so field order never changes a golden file.
func (r *Recorder) Transcript() string {
	var b strings.Builder
	for _, f := range r.Frames() {
		v := f.Raw.Clone()
		if err := v.Canonicalize(); err != nil {
			v = f.Raw // not valid JSON: keep it as written
		}
		b.WriteString(f.Dir.String() + " " + string(v) + "\n")
	}
	return b.String()
}

// Golden compares the transcript with the file at path. On a mismatch it
// writes the actual transcript to t.ArtifactDir() and reports both the first
// differing line and that file. With ACPTEST_UPDATE=1 it rewrites path
// instead.
func (r *Recorder) Golden(t testing.TB, path string) {
	t.Helper()
	got := r.Transcript()
	if os.Getenv("ACPTEST_UPDATE") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil { //nolint:gosec // G306: a golden file is meant to be read
			t.Errorf("acptest: update golden %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path) //nolint:gosec // G304: the test names its golden file
	if err == nil && strings.ReplaceAll(string(want), "\r\n", "\n") == got {
		return
	}
	out := filepath.Join(t.ArtifactDir(), filepath.Base(path)+".got")
	if werr := os.WriteFile(out, []byte(got), 0o600); werr != nil {
		t.Errorf("acptest: write transcript artifact: %v", werr)
	}
	if err != nil {
		t.Errorf("acptest: golden %s: %v; the transcript is in %s (ACPTEST_UPDATE=1 creates the golden)", path, err, out)
		return
	}
	line, g, w := firstDiff(got, strings.ReplaceAll(string(want), "\r\n", "\n"))
	t.Errorf("acptest: transcript differs from %s at line %d\n got: %s\nwant: %s\nfull transcript: %s", path, line, g, w, out)
}

func firstDiff(got, want string) (line int, g, w string) {
	gl, wl := strings.Split(got, "\n"), strings.Split(want, "\n")
	for i := range max(len(gl), len(wl)) {
		var a, b string
		if i < len(gl) {
			a = gl[i]
		}
		if i < len(wl) {
			b = wl[i]
		}
		if a != b {
			return i + 1, a, b
		}
	}
	return 0, "", ""
}
