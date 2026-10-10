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
	"slices"
	"strings"
	"time"
)

// Version is the session format gobble writes, Pi's v3, and the newest it
// reads. v1 and v2 are migrated when read (Migrate).
const Version = 3

// Entry types: Pi's (session-manager.ts SessionEntry). Any other type is
// kept verbatim.
const (
	TypeHeader              = "session"
	TypeMessage             = "message"
	TypeModelChange         = "model_change"
	TypeThinkingLevelChange = "thinking_level_change"
	TypeSessionInfo         = "session_info"
	TypeCompaction          = "compaction"
	TypeCustom              = "custom"
	TypeUsage               = "usage"
	TypeContextEdit         = "context_edit"
	TypeBranchSummary       = "branch_summary"
	TypeCustomMessage       = "custom_message"
	TypeLabel               = "label"
)

// Custom entry types gobble writes (Entry.CustomType).
const (
	// CustomTodo is the todo list after a todo call: data is {todos}.
	CustomTodo = "gobble.todo"
)

// Message roles gobble writes.
const (
	RoleUser       = "user"
	RoleAssistant  = "assistant"
	RoleToolResult = "toolResult"
)

// Pi's other message roles, which gobble reads (messages.ts). A
// branchSummary or compactionSummary message is never stored: it is how a
// branch_summary or compaction entry enters the context.
const (
	RoleCustom            = "custom"
	RoleBashExecution     = "bashExecution"
	RoleBranchSummary     = "branchSummary"
	RoleCompactionSummary = "compactionSummary"
	RoleSystem            = "system"
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
	Type          string   `json:"type"`
	ID            string   `json:"id,omitzero"`
	ParentID      *string  `json:"parentId"`
	Timestamp     string   `json:"timestamp,omitzero"`
	Message       *Message `json:"message,omitzero"`
	Provider      string   `json:"provider,omitzero"`
	ModelID       string   `json:"modelId,omitzero"`
	ThinkingLevel string   `json:"thinkingLevel,omitzero"`
	Name          *string  `json:"name,omitzero"`
	// Summary, FirstKeptEntryID, TokensBefore, Details, Usage and FromHook
	// are a compaction's members (Pi's CompactionEntry). A compaction that
	// keeps nothing names itself as its first kept entry.
	Summary          string         `json:"summary,omitzero"`
	FirstKeptEntryID string         `json:"firstKeptEntryId,omitzero"`
	TokensBefore     *int64         `json:"tokensBefore,omitzero"`
	Details          jsontext.Value `json:"details,omitzero"`
	Usage            *Usage         `json:"usage,omitzero"`
	FromHook         *bool          `json:"fromHook,omitzero"`
	// CustomType and Data are a custom entry's members (Pi's CustomEntry):
	// an extension's state, which takes no part in the model's context.
	CustomType string         `json:"customType,omitzero"`
	Data       jsontext.Value `json:"data,omitzero"`
	// TargetID is the entry a context_edit or a label names.
	TargetID string `json:"targetId,omitzero"`
	// Replacement is a context_edit's {content}, or null, which leaves its
	// target out of the context. A null is kept, and differs from absent.
	Replacement jsontext.Value `json:"replacement,omitzero"`
	// FromID is the leaf a branch_summary left, or "root".
	FromID string `json:"fromId,omitzero"`
	// Content and Display are a custom_message's (Pi's CustomMessageEntry),
	// with CustomType and Details.
	Content jsontext.Value `json:"content,omitzero"`
	Display *bool          `json:"display,omitzero"`
	// Label is a label entry's text; absent or empty clears its target's.
	Label *string `json:"label,omitzero"`
	// Kind, Model and Note are a usage entry's (Pi's UsageEntry), with
	// Provider and Usage.
	Kind    string         `json:"kind,omitzero"`
	Model   string         `json:"model,omitzero"`
	Note    string         `json:"note,omitzero"`
	Unknown jsontext.Value `json:",embed"`
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
	// CustomType, Display and Details are a custom message's (Pi's
	// CustomMessage); a tool result's details are Details too.
	CustomType string         `json:"customType,omitzero"`
	Display    *bool          `json:"display,omitzero"`
	Details    jsontext.Value `json:"details,omitzero"`
	// Summary, FromID and TokensBefore are a branchSummary's or a
	// compactionSummary's.
	Summary      string `json:"summary,omitzero"`
	FromID       string `json:"fromId,omitzero"`
	TokensBefore *int64 `json:"tokensBefore,omitzero"`
	// Command to ExcludeFromContext are a bashExecution's (Pi's
	// BashExecutionMessage). ExitCode is raw, as Pi writes null for a
	// command a signal ended.
	Command            string         `json:"command,omitzero"`
	Output             *string        `json:"output,omitzero"`
	ExitCode           jsontext.Value `json:"exitCode,omitzero"`
	Cancelled          *bool          `json:"cancelled,omitzero"`
	Truncated          *bool          `json:"truncated,omitzero"`
	FullOutputPath     string         `json:"fullOutputPath,omitzero"`
	ExcludeFromContext *bool          `json:"excludeFromContext,omitzero"`
	Timestamp          int64          `json:"timestamp,omitzero"`
	Unknown            jsontext.Value `json:",embed"`
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
// version is above Version. A malformed line after the header is skipped,
// not an error.
var ErrCorrupt = errors.New("session: not a valid session")

// ErrOldVersion refuses an append to a session whose file is an older
// version: gobble migrates it in memory each time it is read, and never
// rewrites it (0005-PLAN F2).
type ErrOldVersion struct {
	ID      ID
	Version int
}

func (e *ErrOldVersion) Error() string {
	return fmt.Sprintf("session %s is a version %d file, which gobble reads but does not rewrite; /clone it to continue in a new session", e.ID, e.Version)
}

// Name is a session's name as Pi reads it (session-manager.ts
// getSessionName): the latest session_info in file order, trimmed. An empty
// or absent name clears it.
func Name(entries []Entry) string {
	for _, e := range slices.Backward(entries) {
		if e.Type == TypeSessionInfo {
			if e.Name == nil {
				return ""
			}
			return strings.TrimSpace(*e.Name)
		}
	}
	return ""
}

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
