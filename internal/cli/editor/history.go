package editor

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/maccavelli/go-tui-lib/v2/lineedit"
)

// maxHistory is the number of entries kept.
const maxHistory = 1000

// FileHistory is a History kept in a file, one entry per line, with "\n"
// written as `\n` and `\` as `\\`: lineedit.MemHistory's file form, which
// is gobble's (0035-PLAN P17). Every Add rewrites the file by atomic
// replace: a temp file in the same directory, then a rename. A write error
// does not stop the editor; Err reports the last one.
type FileHistory struct {
	path string
	mem  lineedit.MemHistory
	err  error
}

// OpenHistory loads the history at path. A missing file is an empty history.
func OpenHistory(path string) (*FileHistory, error) {
	h := &FileHistory{path: path, mem: lineedit.MemHistory{Max: maxHistory}}
	f, err := os.Open(path) //nolint:gosec // G304: gobble's own <state>/history, chosen by the caller
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return h, nil
	case err != nil:
		return nil, fmt.Errorf("read history: %w", err)
	}
	_, err = h.mem.ReadFrom(f)
	if err = errors.Join(err, f.Close()); err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}
	return h, nil
}

// Add appends entry unless it is empty or repeats the most recent entry,
// then saves the file.
func (h *FileHistory) Add(entry string) {
	if entry == "" || (h.mem.Len() > 0 && h.mem.At(0) == entry) {
		return
	}
	h.mem.Add(entry)
	h.err = h.save()
}

// Len returns the number of entries.
func (h *FileHistory) Len() int { return h.mem.Len() }

// At returns an entry; 0 is the most recent. It panics when idx is out of
// range, as the History contract allows: the editor checks Len first.
func (h *FileHistory) At(idx int) string { return h.mem.At(idx) }

// Err returns the error of the last save, or nil.
func (h *FileHistory) Err() error { return h.err }

func (h *FileHistory) save() (err error) {
	var b strings.Builder
	if _, err := h.mem.WriteTo(&b); err != nil {
		return fmt.Errorf("save history: %w", err)
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
