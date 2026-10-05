package editor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// maxHistory is the number of entries kept.
const maxHistory = 1000

// FileHistory is a History kept in a file, one entry per line, with "\n"
// written as `\n` and `\` as `\\`. Every Add rewrites the file by atomic
// replace: a temp file in the same directory, then a rename. A write error
// does not stop the editor; Err reports the last one.
type FileHistory struct {
	path    string
	entries []string // oldest first
	err     error
}

// OpenHistory loads the history at path. A missing file is an empty history.
func OpenHistory(path string) (*FileHistory, error) {
	h := &FileHistory{path: path}
	b, err := os.ReadFile(path) //nolint:gosec // G304: gobble's own <state>/history, chosen by the caller
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return h, nil
	case err != nil:
		return nil, fmt.Errorf("read history: %w", err)
	}
	for line := range strings.SplitSeq(strings.TrimSuffix(string(b), "\n"), "\n") {
		if line = strings.TrimSuffix(line, "\r"); line != "" {
			h.entries = append(h.entries, unescape(line))
		}
	}
	h.trim()
	return h, nil
}

// Add appends entry unless it is empty or repeats the most recent entry,
// then saves the file.
func (h *FileHistory) Add(entry string) {
	if entry == "" || (len(h.entries) > 0 && h.entries[len(h.entries)-1] == entry) {
		return
	}
	h.entries = append(h.entries, entry)
	h.trim()
	h.err = h.save()
}

// Len returns the number of entries.
func (h *FileHistory) Len() int { return len(h.entries) }

// At returns an entry; 0 is the most recent. It panics when idx is out of
// range, as x/term's History contract allows.
func (h *FileHistory) At(idx int) string {
	if idx < 0 || idx >= len(h.entries) {
		panic(fmt.Sprintf("editor: history index %d out of range [0,%d)", idx, len(h.entries)))
	}
	return h.entries[len(h.entries)-1-idx]
}

// Err returns the error of the last save, or nil.
func (h *FileHistory) Err() error { return h.err }

func (h *FileHistory) trim() {
	if extra := len(h.entries) - maxHistory; extra > 0 {
		h.entries = h.entries[extra:]
	}
}

func (h *FileHistory) save() (err error) {
	var b strings.Builder
	for _, e := range h.entries {
		b.WriteString(escape(e))
		b.WriteByte('\n')
	}
	tmp, err := os.CreateTemp(filepath.Dir(h.path), ".history-*")
	if err != nil {
		return fmt.Errorf("save history: %w", err)
	}
	name := tmp.Name()
	defer func() {
		if err != nil {
			err = errors.Join(err, os.Remove(name))
		}
	}()
	if _, err := tmp.WriteString(b.String()); err != nil {
		return errors.Join(fmt.Errorf("save history: %w", err), tmp.Close())
	}
	if err := tmp.Chmod(0o600); err != nil && !errors.Is(err, errors.ErrUnsupported) {
		return errors.Join(fmt.Errorf("save history: %w", err), tmp.Close())
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("save history: %w", err)
	}
	if err := os.Rename(name, h.path); err != nil {
		return fmt.Errorf("save history: %w", err)
	}
	return nil
}

func escape(s string) string {
	return strings.NewReplacer(`\`, `\\`, "\n", `\n`).Replace(s)
}

func unescape(s string) string {
	if !strings.Contains(s, `\`) {
		return s
	}
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i+1 == len(s) {
			b.WriteByte(s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			b.WriteByte('\n')
		case '\\':
			b.WriteByte('\\')
		default:
			b.WriteByte('\\')
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
