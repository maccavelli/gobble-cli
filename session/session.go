package session

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"iter"
	"regexp"
	"time"
)

// Version is the session format gobble reads and writes: Pi's v3.
const Version = 3

// Entry types gobble acts on. Every other type is kept verbatim.
const (
	TypeHeader              = "session"
	TypeMessage             = "message"
	TypeModelChange         = "model_change"
	TypeThinkingLevelChange = "thinking_level_change"
	TypeSessionInfo         = "session_info"
)

// Message roles gobble writes.
const (
	RoleUser       = "user"
	RoleAssistant  = "assistant"
	RoleToolResult = "toolResult"
)

// ID identifies one session.
type ID string

// idPattern is Pi's session id rule (session-manager.ts assertValidSessionId).
var idPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9._-]*[A-Za-z0-9])?$`)

// ValidID reports whether id is a valid session id.
func ValidID(id ID) bool { return idPattern.MatchString(string(id)) }

// Header is a session file's first line. It is not an entry of the tree.
type Header struct {
	Type          string         `json:"type"`
	Version       int            `json:"version,omitzero"`
	ID            ID             `json:"id"`
	Timestamp     string         `json:"timestamp"`
	Cwd           string         `json:"cwd"`
	ParentSession string         `json:"parentSession,omitzero"`
	Unknown       jsontext.Value `json:",embed"`
}

// Entry is one line of a session after the header. ParentID is nil at a
// root, and is written as null. The members gobble acts on are typed; any
// other member, and any other entry type's members, round-trip through
// Unknown.
type Entry struct {
	Type          string         `json:"type"`
	ID            string         `json:"id,omitzero"`
	ParentID      *string        `json:"parentId"`
	Timestamp     string         `json:"timestamp,omitzero"`
	Message       *Message       `json:"message,omitzero"`
	Provider      string         `json:"provider,omitzero"`
	ModelID       string         `json:"modelId,omitzero"`
	ThinkingLevel string         `json:"thinkingLevel,omitzero"`
	Name          *string        `json:"name,omitzero"`
	Unknown       jsontext.Value `json:",embed"`
}

// Message is Pi's AgentMessage. Content is a JSON string or an array of
// blocks (Blocks reads either). Timestamp is in milliseconds.
type Message struct {
	Role         string         `json:"role"`
	Content      jsontext.Value `json:"content,omitzero"`
	API          string         `json:"api,omitzero"`
	Provider     string         `json:"provider,omitzero"`
	Model        string         `json:"model,omitzero"`
	Usage        *Usage         `json:"usage,omitzero"`
	StopReason   string         `json:"stopReason,omitzero"`
	ErrorMessage string         `json:"errorMessage,omitzero"`
	ToolCallID   string         `json:"toolCallId,omitzero"`
	ToolName     string         `json:"toolName,omitzero"`
	IsError      *bool          `json:"isError,omitzero"`
	Timestamp    int64          `json:"timestamp,omitzero"`
	Unknown      jsontext.Value `json:",embed"`
}

// Block is one content block: text, image, thinking or toolCall.
type Block struct {
	Type              string         `json:"type"`
	Text              string         `json:"text,omitzero"`
	Data              string         `json:"data,omitzero"`
	MimeType          string         `json:"mimeType,omitzero"`
	Thinking          string         `json:"thinking,omitzero"`
	ThinkingSignature string         `json:"thinkingSignature,omitzero"`
	Redacted          bool           `json:"redacted,omitzero"`
	ID                string         `json:"id,omitzero"`
	Name              string         `json:"name,omitzero"`
	Arguments         jsontext.Value `json:"arguments,omitzero"`
	ThoughtSignature  string         `json:"thoughtSignature,omitzero"`
	Unknown           jsontext.Value `json:",embed"`
}

// Block types.
const (
	BlockText     = "text"
	BlockImage    = "image"
	BlockThinking = "thinking"
	BlockToolCall = "toolCall"
)

// Usage is Pi's usage: every count is present. Reasoning is part of Output.
type Usage struct {
	Input       int64          `json:"input"`
	Output      int64          `json:"output"`
	CacheRead   int64          `json:"cacheRead"`
	CacheWrite  int64          `json:"cacheWrite"`
	Reasoning   int64          `json:"reasoning,omitzero"`
	TotalTokens int64          `json:"totalTokens"`
	Cost        Cost           `json:"cost"`
	Unknown     jsontext.Value `json:",embed"`
}

// Cost is Pi's per-category cost.
type Cost struct {
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
	Total      float64 `json:"total"`
}

// Blocks reads a message's content: a JSON string is one text block.
func (m *Message) Blocks() ([]Block, error) {
	if len(m.Content) == 0 {
		return nil, nil
	}
	var s string
	if err := json.Unmarshal(m.Content, &s); err == nil {
		return []Block{{Type: BlockText, Text: s}}, nil
	}
	var bs []Block
	if err := json.Unmarshal(m.Content, &bs); err != nil {
		return nil, fmt.Errorf("session: message content: %w", err)
	}
	return bs, nil
}

// SetBlocks writes a message's content as an array of blocks.
func (m *Message) SetBlocks(bs []Block) error {
	if bs == nil {
		bs = []Block{}
	}
	b, err := json.Marshal(bs)
	if err != nil {
		return fmt.Errorf("session: message content: %w", err)
	}
	m.Content = b
	return nil
}

// Text is the message's text blocks joined by single spaces, as Pi's list
// shows a message.
func (m *Message) Text() string {
	bs, err := m.Blocks()
	if err != nil {
		return ""
	}
	var out string
	for _, b := range bs {
		if b.Type == BlockText && b.Text != "" {
			if out != "" {
				out += " "
			}
			out += b.Text
		}
	}
	return out
}

// NewEntryID is an id for a new entry: 8 hex characters not in taken, as
// Pi makes them, or 32 when 100 tries collide.
func NewEntryID(taken func(string) bool) string {
	for range 100 {
		if id := randomHex(4); !taken(id) {
			return id
		}
	}
	return randomHex(16)
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b) // crypto/rand.Read never fails (Go 1.24+)
	return hex.EncodeToString(b)
}

// Timestamp is an entry timestamp: ISO 8601 in UTC with milliseconds, as Pi
// writes them.
func Timestamp(t time.Time) string { return t.UTC().Format("2006-01-02T15:04:05.000Z") }

// Path is the active path through entries: from the leaf, which is the last
// entry in file order, to its root through ParentID, returned root first.
// A parent that is missing ends the path.
func Path(entries []Entry) []Entry {
	if len(entries) == 0 {
		return nil
	}
	byID := make(map[string]int, len(entries))
	for i, e := range entries {
		byID[e.ID] = i
	}
	var rev []Entry
	seen := map[string]bool{}
	for i, ok := len(entries)-1, true; ok && !seen[entries[i].ID]; {
		e := entries[i]
		seen[e.ID] = true
		rev = append(rev, e)
		if e.ParentID == nil {
			break
		}
		i, ok = byID[*e.ParentID]
	}
	out := make([]Entry, len(rev))
	for i, e := range rev {
		out[len(rev)-1-i] = e
	}
	return out
}

// Summary is one row of Store.List.
type Summary struct {
	ID       ID
	Cwd      string
	Name     string // the latest session_info name
	First    string // the first user message's text
	Path     string // the file, when the store has files
	Created  time.Time
	Modified time.Time // the last user or assistant message, else Created
	Messages int
}

// Filter selects sessions for Store.List. An empty Cwd selects every
// session.
type Filter struct {
	Cwd string
}

// ErrNotFound is returned by Store.Open and Store.Read for an unknown id.
var ErrNotFound = errors.New("session: not found")

// ErrExists is returned by Store.Create for an id already in use.
var ErrExists = errors.New("session: id already in use")

// ErrCorrupt is returned for a session whose header cannot be read, or whose
// version is not Version. A malformed line after the header is skipped, not
// an error.
var ErrCorrupt = errors.New("session: not a valid session")

// ErrLocked is returned by Store.Open when another process writes the
// session.
type ErrLocked struct {
	ID  ID
	PID int
}

func (e *ErrLocked) Error() string {
	return fmt.Sprintf("session %s is open in another gobble process (pid %d)", e.ID, e.PID)
}

// Log is one session open for writing. Entries appended before the first
// user or assistant message are held in memory; that message creates the
// file. Close releases the writer.
type Log interface {
	Header() Header
	Entries() []Entry
	Append(ctx context.Context, entry Entry) error
	Close() error
}

// Store creates, opens and lists sessions.
type Store interface {
	// Create starts a session. Nothing is written until the first user or
	// assistant message.
	Create(ctx context.Context, header Header) (Log, error)
	// Open opens a session for writing, taking its writer lock.
	Open(ctx context.Context, id ID) (Log, error)
	// Read reads a session without the lock. Malformed lines are skipped and
	// counted.
	Read(ctx context.Context, id ID) (Header, []Entry, int, error)
	// List lists sessions, newest first.
	List(ctx context.Context, filter Filter) iter.Seq2[Summary, error]
}
