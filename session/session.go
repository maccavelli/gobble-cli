package session

import (
	"context"
	"iter"
	"time"

	"encoding/json/jsontext"
)

// ID identifies one session.
type ID string

// Header is the data Store.Create records for a new log.
type Header struct {
	ID      ID        `json:"id,omitzero"`
	Title   string    `json:"title,omitzero"`
	Created time.Time `json:"created,omitzero"`
}

// Entry is one JSON object in a session log.
// Type is the union discriminant. Unknown holds object members this version
// does not declare, via the json embed fallback.
type Entry struct {
	Type    string         `json:"type"`
	ID      string         `json:"id,omitzero"`
	Unknown jsontext.Value `json:",embed"`
}

// Summary is one row from Store.List.
type Summary struct {
	ID    ID     `json:"id"`
	Title string `json:"title,omitzero"`
}

// Filter selects sessions for Store.List.
// Phase 1 has no criteria.
type Filter struct{}

// Log is one open session.
type Log interface {
	ID() ID
	Append(ctx context.Context, entry Entry) error
	Entries(ctx context.Context) iter.Seq2[Entry, error]
}

// Store opens and lists session logs.
type Store interface {
	Create(ctx context.Context, header Header) (Log, error)
	Open(ctx context.Context, id ID) (Log, error)
	List(ctx context.Context, filter Filter) iter.Seq2[Summary, error]
}
