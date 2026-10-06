package session

import (
	"context"
	"iter"
	"os"
	"slices"
	"sync"
	"time"
)

// NewMemoryStore is a Store that keeps sessions in memory, for --no-session
// and tests. Like the file store it lists only sessions with a conversation,
// and lets one writer open a session at a time.
func NewMemoryStore() Store { return &memoryStore{sessions: map[ID]*memorySession{}} }

type memoryStore struct {
	mu       sync.Mutex
	sessions map[ID]*memorySession
}

type memorySession struct {
	header  Header
	entries []Entry
	open    bool
}

func (s *memoryStore) Create(_ context.Context, h Header) (Log, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sessions[h.ID]; ok {
		return nil, ErrExists
	}
	ms := &memorySession{header: h, open: true}
	s.sessions[h.ID] = ms
	return &memoryLog{s: s, ms: ms}, nil
}

func (s *memoryStore) Open(_ context.Context, id ID) (Log, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ms, ok := s.sessions[id]
	switch {
	case !ok:
		return nil, ErrNotFound
	case ms.open:
		return nil, &ErrLocked{ID: id, PID: os.Getpid()}
	}
	ms.open = true
	return &memoryLog{s: s, ms: ms}, nil
}

func (s *memoryStore) Read(_ context.Context, id ID) (Header, []Entry, int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ms, ok := s.sessions[id]
	if !ok {
		return Header{}, nil, 0, ErrNotFound
	}
	return ms.header, slices.Clone(ms.entries), 0, nil
}

func (s *memoryStore) List(_ context.Context, f Filter) iter.Seq2[Summary, error] {
	s.mu.Lock()
	var out []Summary
	for _, ms := range s.sessions {
		if f.Cwd != "" && ms.header.Cwd != f.Cwd {
			continue
		}
		if sum := Summarize(ms.header, ms.entries, ""); sum.Messages > 0 {
			out = append(out, sum)
		}
	}
	s.mu.Unlock()
	SortSummaries(out)
	return func(yield func(Summary, error) bool) {
		for _, sum := range out {
			if !yield(sum, nil) {
				return
			}
		}
	}
}

type memoryLog struct {
	s  *memoryStore
	ms *memorySession
}

func (l *memoryLog) Header() Header {
	l.s.mu.Lock()
	defer l.s.mu.Unlock()
	return l.ms.header
}

func (l *memoryLog) Entries() []Entry {
	l.s.mu.Lock()
	defer l.s.mu.Unlock()
	return slices.Clone(l.ms.entries)
}

func (l *memoryLog) Append(_ context.Context, e Entry) error {
	l.s.mu.Lock()
	defer l.s.mu.Unlock()
	l.ms.entries = append(l.ms.entries, e)
	return nil
}

func (l *memoryLog) Close() error {
	l.s.mu.Lock()
	defer l.s.mu.Unlock()
	l.ms.open = false
	return nil
}

// Summarize is a session's list row: its name, first user message, message
// count, and the time of its last user or assistant message.
func Summarize(h Header, entries []Entry, path string) Summary {
	sum := Summary{ID: h.ID, Cwd: h.Cwd, Path: path}
	if t, err := time.Parse(time.RFC3339, h.Timestamp); err == nil {
		sum.Created = t
	}
	sum.Modified = sum.Created
	for _, e := range entries {
		switch e.Type {
		case TypeSessionInfo:
			if e.Name != nil {
				sum.Name = *e.Name
			}
		case TypeMessage:
			if e.Message == nil {
				continue
			}
			sum.Messages++
			m := e.Message
			if m.Role != RoleUser && m.Role != RoleAssistant {
				continue
			}
			if m.Timestamp > 0 {
				if t := time.UnixMilli(m.Timestamp).UTC(); t.After(sum.Modified) {
					sum.Modified = t
				}
			}
			if sum.First == "" && m.Role == RoleUser {
				sum.First = m.Text()
			}
		}
	}
	return sum
}

// SortSummaries orders sessions newest first, by their last activity.
func SortSummaries(s []Summary) {
	slices.SortStableFunc(s, func(a, b Summary) int { return b.Modified.Compare(a.Modified) })
}

// HasConversation reports whether entries hold a user or assistant message,
// which is when a session gets its file.
func HasConversation(entries []Entry) bool {
	return slices.ContainsFunc(entries, func(e Entry) bool {
		return e.Type == TypeMessage && e.Message != nil && (e.Message.Role == RoleUser || e.Message.Role == RoleAssistant)
	})
}
