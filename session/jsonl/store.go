package jsonl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/maccavelli/gobble-cli/internal/filelock"
	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/session"
)

// Options configure a Store.
type Options struct {
	// Flat keeps every session directly in the store's directory, as Pi's
	// custom session directory does; List then filters by the header's cwd.
	// Otherwise each cwd has its own subdirectory, --<cwd>--.
	Flat bool
	// Logger receives one warning per skipped line. Nil discards them.
	Logger *slog.Logger
}

// Store keeps sessions as Pi-compatible JSONL files under one directory.
type Store struct {
	dir string
	o   Options

	mu       sync.Mutex
	creating map[session.ID]bool // ids created in this process, not yet written

	// openFile and truncate are the store's writes to a session file. Tests
	// replace them to make a write or a cut fail (0037-MADR D37).
	openFile func(name string, flag int, perm os.FileMode) (appendFile, error)
	truncate func(name string, size int64) error
}

// appendFile is what a log writes a session file through: an *os.File.
type appendFile interface {
	Write(b []byte) (int, error)
	Close() error
}

func openOSFile(name string, flag int, perm os.FileMode) (appendFile, error) {
	return os.OpenFile(name, flag, perm) //nolint:gosec // G304: a file of the store's own directory
}

var _ session.Store = (*Store)(nil)

// New returns the store rooted at dir. Nothing is created until a session
// has its first user or assistant message.
func New(dir string, o Options) *Store {
	if o.Logger == nil {
		o.Logger = slog.New(slog.DiscardHandler)
	}
	return &Store{dir: dir, o: o, creating: map[session.ID]bool{}, openFile: openOSFile, truncate: os.Truncate}
}

// Dir is the store's root directory.
func (s *Store) Dir() string { return s.dir }

// PathOf is session id's file, or "" when it has none: the parentSession a
// fork records, as Pi records an absolute path.
func (s *Store) PathOf(id session.ID) string {
	p, err := s.find(id)
	if err != nil {
		return ""
	}
	if abs, err := filepath.Abs(p); err == nil {
		return abs
	}
	return p
}

// EncodeCwd is Pi's directory name for a cwd: the leading separator removed,
// then / \ and : each made -, between -- and -- (session-manager.ts
// getDefaultSessionDirPath).
func EncodeCwd(cwd string) string {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(cwd, "/"), `\`)
	return "--" + strings.NewReplacer("/", "-", `\`, "-", ":", "-").Replace(trimmed) + "--"
}

// FileName is Pi's file name for a session: the header's timestamp with : and
// . made -, an underscore, the id, and .jsonl.
func FileName(h session.Header) string {
	return strings.NewReplacer(":", "-", ".", "-").Replace(h.Timestamp) + "_" + string(h.ID) + ".jsonl"
}

func (s *Store) dirFor(cwd string) string {
	if s.o.Flat {
		return s.dir
	}
	return filepath.Join(s.dir, EncodeCwd(cwd))
}

// files lists the store's session files.
func (s *Store) files() ([]string, error) {
	pattern := filepath.Join(s.dir, "*", "*.jsonl")
	if s.o.Flat {
		pattern = filepath.Join(s.dir, "*.jsonl")
	}
	return filepath.Glob(pattern)
}

// find is the file of session id, or "".
func (s *Store) find(id session.ID) (string, error) {
	if !session.ValidID(id) {
		return "", fmt.Errorf("%w: invalid id %q", session.ErrNotFound, id)
	}
	pattern := filepath.Join(s.dir, "*", "*_"+string(id)+".jsonl")
	if s.o.Flat {
		pattern = filepath.Join(s.dir, "*_"+string(id)+".jsonl")
	}
	m, err := filepath.Glob(pattern)
	if err != nil || len(m) == 0 {
		return "", err
	}
	return m[0], nil
}

// Create starts a session. Its file is written with its first user or
// assistant message (session.Log).
func (s *Store) Create(_ context.Context, h session.Header) (session.Log, error) {
	if !session.ValidID(h.ID) {
		return nil, fmt.Errorf("session: invalid id %q", h.ID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if path, err := s.find(h.ID); err != nil || path != "" || s.creating[h.ID] {
		return nil, session.ErrExists
	}
	s.creating[h.ID] = true
	return &fileLog{s: s, header: h, path: filepath.Join(s.dirFor(h.Cwd), FileName(h))}, nil
}

// Open opens a session for writing, taking its writer lock.
func (s *Store) Open(_ context.Context, id session.ID) (session.Log, error) {
	path, err := s.find(id)
	if err != nil {
		return nil, err
	}
	if path == "" {
		return nil, session.ErrNotFound
	}
	release, err := filelock.TryLock(path + ".lock")
	if err != nil {
		if held, ok := errors.AsType[*filelock.HeldError](err); ok {
			return nil, &session.ErrLocked{ID: id, PID: held.PID}
		}
		return nil, fmt.Errorf("session: lock: %w", err)
	}
	h, entries, _, b, err := s.readFile(path)
	if err != nil {
		return nil, errors.Join(err, release())
	}
	l := &fileLog{s: s, header: h, entries: entries, path: path, written: true, size: int64(len(b)), release: release}
	if v := max(h.Version, 1); v < session.Version {
		l.old = v
	}
	if err := l.cutTornTail(b); err != nil {
		return nil, errors.Join(err, release())
	}
	return l, nil
}

// Read reads a session without its lock. It returns the number of lines
// skipped as malformed.
func (s *Store) Read(_ context.Context, id session.ID) (session.Header, []session.Entry, int, error) {
	path, err := s.find(id)
	if err != nil {
		return session.Header{}, nil, 0, err
	}
	if path == "" {
		return session.Header{}, nil, 0, session.ErrNotFound
	}
	h, entries, skipped, _, err := s.readFile(path)
	return h, entries, skipped, err
}

// readFile reads and parses a session file. It also reports the lines
// skipped as malformed, and returns the bytes it read, from which a writer
// takes the file's size and finds a torn last line.
func (s *Store) readFile(path string) (h session.Header, entries []session.Entry, skipped int, b []byte, err error) {
	b, err = os.ReadFile(path) //nolint:gosec // G304: a file of the store's own directory
	if err != nil {
		return h, nil, 0, nil, fmt.Errorf("session: read: %w", err)
	}
	h, entries, skipped, err = parse(b)
	if err != nil {
		return h, nil, 0, nil, fmt.Errorf("%w: %s: %w", session.ErrCorrupt, filepath.Base(path), err)
	}
	if skipped > 0 {
		s.o.Logger.Warn("session: malformed lines skipped", slog.String("file", filepath.Base(path)), slog.Int("lines", skipped))
	}
	return h, entries, skipped, b, nil
}

// decodeEntry decodes one line as an entry: valid JSON, with a type, and
// not a header.
func decodeEntry(line []byte) (session.Entry, bool) {
	var e session.Entry
	if err := jsonx.Unmarshal(line, &e); err != nil || e.Type == "" || e.Type == session.TypeHeader {
		return e, false
	}
	return e, true
}

// parse reads a session file: the header, which must be a session header of
// version 1 to 3, then each entry. A malformed entry line is skipped and
// counted, as Pi skips it (0005-MADR "Sessions (changed)"). An older file's
// entries are migrated in memory (session.Migrate); the header keeps its
// version, so a log knows not to write the file. Lines are read in place, one
// at a time (bytes.Lines), not sliced up first (0037-MADR D41).
func parse(b []byte) (session.Header, []session.Entry, int, error) {
	var (
		h       session.Header
		header  bool // the header line has been read
		v       int
		entries []session.Entry
		skipped int
	)
	for line := range bytes.Lines(b) {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if !header {
			if err := jsonx.Unmarshal(line, &h); err != nil || h.Type != session.TypeHeader || h.ID == "" {
				return h, nil, 0, errors.New("the first line is not a session header")
			}
			v = max(h.Version, 1)
			if v > session.Version {
				return h, nil, 0, fmt.Errorf("unsupported session version %d (gobble reads versions 1 to %d)", v, session.Version)
			}
			header = true
			continue
		}
		e, ok := decodeEntry(line)
		if !ok {
			skipped++
			continue
		}
		entries = append(entries, e)
	}
	if !header {
		return h, nil, 0, errors.New("empty file")
	}
	if v < session.Version {
		migrated, err := session.Migrate(v, entries)
		if err != nil {
			return h, nil, 0, err
		}
		entries = migrated
	}
	return h, entries, skipped, nil
}

// List lists sessions newest first. A file that is not a valid session is
// left out, as Pi's discovery leaves it out.
func (s *Store) List(ctx context.Context, f session.Filter) iter.Seq2[session.Summary, error] {
	return func(yield func(session.Summary, error) bool) {
		paths, err := s.files()
		if err != nil {
			yield(session.Summary{}, err)
			return
		}
		var out []session.Summary
		for _, p := range paths {
			h, entries, _, _, err := s.readFile(p)
			if err != nil {
				s.o.Logger.WarnContext(ctx, "session: not listed", slog.String("file", filepath.Base(p)), slog.String("error", err.Error()))
				continue
			}
			if f.Cwd != "" && h.Cwd != f.Cwd {
				continue
			}
			out = append(out, session.Summarize(h, entries, p))
		}
		session.SortSummaries(out)
		for _, sum := range out {
			if !yield(sum, nil) {
				return
			}
		}
	}
}

// fileLog is one session open for writing.
type fileLog struct {
	s    *Store
	path string

	mu      sync.Mutex
	header  session.Header
	entries []session.Entry
	file    appendFile
	written bool  // the file exists
	size    int64 // the file's size, after its last complete write
	torn    bool  // its last line lacks a newline
	old     int   // the file's version, when it is older than session.Version
	release func() error
}

// cutTornTail handles a last line without its newline, which a write cut
// short leaves. When it is an entry, it is kept, and the next line starts on
// a line of its own. Otherwise, in a file of the current version, it is cut
// off and logged; an older file is never written (0037-MADR D37).
func (l *fileLog) cutTornTail(b []byte) error {
	if len(b) == 0 || b[len(b)-1] == '\n' {
		return nil
	}
	start := bytes.LastIndexByte(b, '\n') + 1
	tail := b[start:]
	if _, ok := decodeEntry(tail); ok || l.old > 0 || len(bytes.TrimSpace(b[:start])) == 0 {
		l.torn = true // an entry, an older file, or the header itself
		return nil
	}
	if err := l.s.truncate(l.path, int64(start)); err != nil {
		return fmt.Errorf("session: cut torn line: %w", err)
	}
	l.s.o.Logger.Warn("session: torn last line cut", slog.String("file", filepath.Base(l.path)), slog.Int("bytes", len(tail)))
	l.size = int64(start)
	return nil
}

func (l *fileLog) Header() session.Header {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.header
}

func (l *fileLog) Entries() []session.Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]session.Entry(nil), l.entries...)
}

// Writable is nil when Append may write, and the session.ErrOldVersion
// Append would return for an older file, so a caller can refuse before
// work whose result would be appended.
func (l *fileLog) Writable() error {
	if l.old > 0 {
		return &session.ErrOldVersion{ID: l.header.ID, Version: l.old}
	}
	return nil
}

// Append records an entry. Until the session has a user or assistant
// message it stays in memory; that message writes the header and every
// entry so far to a new file, and each later entry is one write of one line
// (0008-MADR D19 item 6). An older file is never written: its append is
// refused with session.ErrOldVersion, and nothing is recorded. An entry
// joins Entries only once it is wholly on disk; an append that fails leaves
// the file and Entries as they were (0037-MADR D37).
func (l *fileLog) Append(_ context.Context, e session.Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := l.Writable(); err != nil {
		return err
	}
	if !l.written {
		l.entries = append(l.entries, e)
		if !session.HasConversation(l.entries) {
			return nil
		}
		if err := l.create(); err != nil {
			l.entries = l.entries[:len(l.entries)-1]
			return err
		}
		return nil
	}
	line, err := jsonx.Marshal(e)
	if err != nil {
		return fmt.Errorf("session: encode entry: %w", err)
	}
	if l.torn {
		line = append([]byte("\n"), line...) // keep a torn last line on its own
	}
	if err := l.write(append(line, '\n')); err != nil {
		return err
	}
	l.torn = false
	l.entries = append(l.entries, e)
	return nil
}

// create writes a new file with the header and the entries so far, under the
// writer lock. When that write fails, the file is removed and the lock
// released: the log stays unwritten and its id pending, so the next Append
// creates it again (0037-MADR D37).
func (l *fileLog) create() error {
	if err := os.MkdirAll(filepath.Dir(l.path), 0o700); err != nil {
		return fmt.Errorf("session: %w", err)
	}
	release, err := filelock.TryLock(l.path + ".lock")
	if err != nil {
		if held, ok := errors.AsType[*filelock.HeldError](err); ok {
			return &session.ErrLocked{ID: l.header.ID, PID: held.PID}
		}
		return fmt.Errorf("session: lock: %w", err)
	}
	var buf bytes.Buffer
	for _, v := range append([]any{l.header}, entriesAny(l.entries)...) {
		b, err := jsonx.Marshal(v)
		if err != nil {
			return errors.Join(fmt.Errorf("session: encode: %w", err), release())
		}
		buf.Write(append(b, '\n'))
	}
	f, err := l.s.openFile(l.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return errors.Join(fmt.Errorf("session: create: %w", err), release())
	}
	if _, err := f.Write(buf.Bytes()); err != nil {
		// Close before Remove: Windows does not remove an open file.
		cerr := f.Close()
		return errors.Join(fmt.Errorf("session: write: %w", err), cerr, os.Remove(l.path), release())
	}
	l.file, l.release, l.written, l.size = f, release, true, int64(buf.Len())
	l.s.mu.Lock()
	delete(l.s.creating, l.header.ID)
	l.s.mu.Unlock()
	return nil
}

func entriesAny(es []session.Entry) []any {
	out := make([]any, len(es))
	for i := range es {
		out[i] = es[i]
	}
	return out
}

// write appends b to the file. A write that fails, wholly or in part, is cut
// back to the file's size before it, through the path: Windows refuses to
// truncate through an O_APPEND handle. When the cut fails too, the log is
// marked torn, so the next line starts on a line of its own (0037-MADR D37).
func (l *fileLog) write(b []byte) error {
	if l.file == nil {
		f, err := l.s.openFile(l.path, os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return fmt.Errorf("session: open: %w", err)
		}
		l.file = f
	}
	if _, err := l.file.Write(b); err != nil {
		err = fmt.Errorf("session: write: %w", err)
		if cerr := l.s.truncate(l.path, l.size); cerr != nil {
			l.torn = true
			return errors.Join(err, fmt.Errorf("session: cut failed write: %w", cerr))
		}
		return err
	}
	l.size += int64(len(b))
	return nil
}

// Close closes the file and releases the writer lock. A session never
// written leaves nothing behind.
func (l *fileLog) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	var errs []error
	if l.file != nil {
		errs = append(errs, l.file.Close())
		l.file = nil
	}
	if l.release != nil {
		errs = append(errs, l.release())
		l.release = nil
	}
	if !l.written {
		l.s.mu.Lock()
		delete(l.s.creating, l.header.ID)
		l.s.mu.Unlock()
	}
	return errors.Join(errs...)
}
