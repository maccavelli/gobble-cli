package logging

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/internal/appdirs"
)

// The bounds of magic-cli-remote 0157-MADR D5, as 0008-MADR D14 states them:
// about 60 MiB in all. There is no compression.
const (
	maxSize     = 10 << 20
	maxBackups  = 5
	maxAge      = 28 * 24 * time.Hour
	stampLayout = "2006-01-02T15-04-05.000"
)

// rolling is the file sink, written from magic-cli-remote 0157-MADR D6 and
// D7; that record has no code to copy. One mutex serialises writes and
// rotation, and the sink assumes one writer per file.
type rolling struct {
	mu        sync.Mutex
	path      string
	dir       string
	base, ext string
	limit     int64
	f         *os.File
	size      int64
	warn      io.Writer
	warned    bool
	now       func() time.Time
	rename    func(oldpath, newpath string) error
}

func openRolling(path string, warn io.Writer) (*rolling, error) {
	return newRolling(path, warn, maxSize, time.Now, os.Rename)
}

func newRolling(path string, warn io.Writer, limit int64, now func() time.Time,
	rename func(string, string) error) (*rolling, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("log file %q must be an absolute path", path)
	}
	path = filepath.Clean(path)
	dir := filepath.Dir(path)
	if err := appdirs.EnsurePrivateDir(dir); err != nil {
		return nil, err
	}
	ext := filepath.Ext(path)
	r := &rolling{
		path: path, dir: dir, base: strings.TrimSuffix(filepath.Base(path), ext), ext: ext,
		limit: limit, warn: warn, now: now, rename: rename,
	}
	if err := r.open(); err != nil {
		return nil, err
	}
	if r.size >= r.limit {
		r.rotate()
	}
	return r, nil
}

func (r *rolling) open() error {
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	fi, err := f.Stat()
	if err != nil {
		return errors.Join(fmt.Errorf("stat log file: %w", err), f.Close())
	}
	r.f, r.size = f, fi.Size()
	return nil
}

// Write appends p, rotating first when p would take the file past the
// limit. A record larger than the limit goes to a fresh file whole.
func (r *rolling) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return 0, os.ErrClosed
	}
	if r.size > 0 && r.size+int64(len(p)) > r.limit {
		r.rotate()
		if r.f == nil {
			return 0, os.ErrClosed
		}
	}
	n, err := r.f.Write(p)
	r.size += int64(n)
	return n, err
}

// Close closes the active file.
func (r *rolling) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.f == nil {
		return nil
	}
	err := r.f.Close()
	r.f = nil
	return err
}

// rotate moves the active file to a backup, opens a fresh one and prunes.
// It never loses the sink: when the rename fails, the active file is
// reopened and appended to, and the failure is reported once (0157 D7).
func (r *rolling) rotate() {
	if err := r.f.Close(); err != nil {
		r.report(err)
	}
	r.f = nil
	backup, err := r.backupName()
	if err == nil {
		err = r.rename(r.path, backup)
	}
	if err != nil {
		r.report(fmt.Errorf("rotate %s: %w", r.path, err))
	}
	if err := r.open(); err != nil {
		r.report(err)
		return
	}
	if err := r.prune(); err != nil {
		r.report(err)
	}
}

// backupName is <base>-<UTC stamp><ext>, with -1, -2, … before the extension
// when a backup with that stamp exists: os.Rename replaces an existing file
// silently on Windows (0157 F16), so a collision would destroy the earlier
// backup. The suffix is one more than the highest already used for the
// stamp, not the first free one: after pruning has removed the oldest, a
// first-free name would make the newest backup sort as the oldest.
func (r *rolling) backupName() (string, error) {
	stamp := r.now().UTC().Format(stampLayout)
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return "", err
	}
	next := 0
	for _, e := range entries {
		if b, ok := r.parseBackup(e.Name()); ok && b.at.Format(stampLayout) == stamp {
			next = max(next, b.seq+1)
		}
	}
	stem := filepath.Join(r.dir, r.base+"-"+stamp)
	name := stem + r.ext
	if next > 0 {
		name = stem + "-" + strconv.Itoa(next) + r.ext
	}
	if _, err := os.Lstat(name); !errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("backup name %s is taken: %w", name, cmp.Or(err, fs.ErrExist))
	}
	return name, nil
}

type backup struct {
	name string
	at   time.Time
	seq  int
}

// prune removes backups beyond maxBackups and older than maxAge. Only names
// this sink writes are considered.
func (r *rolling) prune() error {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return err
	}
	var backups []backup
	for _, e := range entries {
		if b, ok := r.parseBackup(e.Name()); ok && e.Type().IsRegular() {
			backups = append(backups, b)
		}
	}
	slices.SortFunc(backups, func(a, b backup) int {
		return cmp.Or(b.at.Compare(a.at), cmp.Compare(b.seq, a.seq))
	})
	cutoff := r.now().Add(-maxAge)
	var errs []error
	for i, b := range backups {
		if i >= maxBackups || b.at.Before(cutoff) {
			errs = append(errs, os.Remove(filepath.Join(r.dir, b.name)))
		}
	}
	return errors.Join(errs...)
}

func (r *rolling) parseBackup(name string) (backup, bool) {
	s, ok := strings.CutPrefix(name, r.base+"-")
	if !ok {
		return backup{}, false
	}
	if s, ok = strings.CutSuffix(s, r.ext); !ok || len(s) < len(stampLayout) {
		return backup{}, false
	}
	at, err := time.Parse(stampLayout, s[:len(stampLayout)])
	if err != nil {
		return backup{}, false
	}
	seq := 0
	if rest := s[len(stampLayout):]; rest != "" {
		n, ok := strings.CutPrefix(rest, "-")
		if seq, err = strconv.Atoi(n); !ok || err != nil || seq < 1 {
			return backup{}, false
		}
	}
	return backup{name: name, at: at, seq: seq}, true
}

// report writes the first failure to warn, once.
func (r *rolling) report(err error) {
	if r.warned || r.warn == nil {
		return
	}
	r.warned = true
	if _, werr := fmt.Fprintf(r.warn, "gobble: log rotation failed, still writing to %s: %v\n", r.path, err); werr != nil {
		r.warn = nil // the warning writer itself is broken; stop using it
	}
}
