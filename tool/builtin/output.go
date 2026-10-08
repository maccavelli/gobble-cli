package builtin

import (
	"bytes"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unicode/utf8"
)

// outputKeep is how long a command's whole output is kept (0005-PLAN F1b,
// owner's decision 4 of 2026-10-07).
const outputKeep = 7 * 24 * time.Hour

// outputPattern names the files: 0003-MADR's gobble-bash-<id>.log.
const outputPattern = "gobble-bash-*.log"

// spill collects a command's combined output. It holds the output in memory
// until it passes maxLines lines or maxBytes bytes; from then on everything
// goes to a file in dir, and only the last tailKeep bytes stay in memory for
// the model. exec calls Write from one goroutine at a time when stdout and
// stderr are the same writer, but the lock keeps that from being a promise
// the tool relies on.
type spill struct {
	dir string

	mu       sync.Mutex
	mem      []byte // the whole output, until it spills; then its tail
	newlines int
	total    int
	last     byte
	file     *os.File
	path     string
	err      error // the first error writing the file
}

// tailKeep is how much of a spilled output stays in memory: twice what the
// model is shown, so whole lines are left to choose from.
const tailKeep = 2 * maxBytes

func (s *spill) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.total += len(p)
	s.newlines += bytes.Count(p, []byte{'\n'})
	if len(p) > 0 {
		s.last = p[len(p)-1]
	}
	if s.file == nil {
		s.mem = append(s.mem, p...)
		if len(s.mem) > maxBytes || s.newlines > maxLines {
			s.open()
		}
		return len(p), nil
	}
	if s.err == nil {
		if _, err := s.file.Write(p); err != nil {
			s.err = err
		}
	}
	s.mem = append(s.mem, p...)
	s.trim()
	return len(p), nil
}

// open starts the file with what is in memory, once. A failure keeps the
// output in memory alone; the model is then told the file could not be
// written.
func (s *spill) open() {
	sweepOnce(s.dir)
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		s.err = err
		return
	}
	f, err := os.CreateTemp(s.dir, outputPattern)
	if err != nil {
		s.err = err
		return
	}
	s.file, s.path = f, f.Name()
	if _, err := f.Write(s.mem); err != nil {
		s.err = err
	}
	s.trim()
}

// trim keeps the last tailKeep bytes in memory, starting on a rune.
func (s *spill) trim() {
	if len(s.mem) <= tailKeep {
		return
	}
	cut := len(s.mem) - tailKeep
	for cut < len(s.mem) && !utf8.RuneStart(s.mem[cut]) {
		cut++
	}
	s.mem = append(s.mem[:0], s.mem[cut:]...)
}

// close closes the file, if there is one.
func (s *spill) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.file != nil {
		if err := s.file.Close(); err != nil && s.err == nil {
			s.err = err
		}
	}
}

// lines is how many lines the whole output had: a final newline does not
// start another.
func (s *spill) lines() int {
	if s.total == 0 {
		return 0
	}
	if s.last == '\n' {
		return s.newlines
	}
	return s.newlines + 1
}

var (
	sweptMu sync.Mutex
	swept   = map[string]bool{}
)

// sweepOnce sweeps dir the first time this process writes a log there.
func sweepOnce(dir string) {
	sweptMu.Lock()
	first := !swept[dir]
	swept[dir] = true
	sweptMu.Unlock()
	if first {
		sweep(dir, time.Now(), outputKeep)
	}
}

// sweep removes the output logs in dir last written more than keep before
// now. Errors are ignored: a log left behind is swept next time.
func sweep(dir string, now time.Time, keep time.Duration) {
	logs, err := filepath.Glob(filepath.Join(dir, outputPattern))
	if err != nil {
		return
	}
	for _, p := range logs {
		if info, err := os.Stat(p); err == nil && now.Sub(info.ModTime()) > keep {
			_ = os.Remove(p) //nolint:errcheck // swept again next time
		}
	}
}
