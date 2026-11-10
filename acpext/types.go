package acpext

import "encoding/json/jsontext"

// MetaKey is gobble's member of an ACP _meta object.
const MetaKey = "gobble"

// CapabilitiesMeta is initialize's agentCapabilities._meta.gobble: the
// requests gobble's agent serves (0039-MADR D15).
type CapabilitiesMeta struct {
	// Extensions are the request names, sorted, as Requests returns them.
	Extensions []string `json:"extensions"`
}

// Phase is a retry wait's or a compaction's place: its start or its end.
type Phase string

// The phases.
const (
	PhaseStart Phase = "start"
	PhaseEnd   Phase = "end"
)

// RetryCause is why a turn waits to retry. It is one of the constants
// below, but a client should take a cause it does not know, which a later
// gobble may add.
type RetryCause string

// The causes: the network gone, an empty reply, or a provider retry's
// failure class (0031-MADR D4).
const (
	RetryOffline      RetryCause = "offline"
	RetryEmptyReply   RetryCause = "empty_reply"
	RetryWorkspace    RetryCause = "workspace"
	RetryQuota        RetryCause = "quota"
	RetryRateLimited  RetryCause = "rate-limited"
	RetryAuth         RetryCause = "auth"
	RetryNotPermitted RetryCause = "not-permitted"
	RetryUnavailable  RetryCause = "unavailable"
	RetryTimeout      RetryCause = "timeout"
	RetryCanceled     RetryCause = "canceled"
	RetryEmpty        RetryCause = "empty"
	RetryOther        RetryCause = "other"
)

// RetryNotice is MethodRetry's params (0005-PLAN F5-1 decision 5).
// Attempt, MaxAttempts and DelayMs are a start's; Error is why an end is
// not a success, and is empty for one that is.
//
// A start that a provider's own retry began also carries, when the provider
// said them, RetryAfterMs, the wait the provider asked for, where DelayMs is
// the wait gobble takes, jitter included; ServerDirected, that the provider
// asked for the wait; Code, the provider's own code; and HTTPStatus
// (gobble-cli 0049-MADR D2).
type RetryNotice struct {
	SessionID      string     `json:"sessionId"`
	Phase          Phase      `json:"phase"`
	Cause          RetryCause `json:"cause"`
	Attempt        int        `json:"attempt,omitzero"`
	MaxAttempts    int        `json:"maxAttempts,omitzero"`
	DelayMs        int64      `json:"delayMs,omitzero"`
	Error          string     `json:"error,omitzero"`
	RetryAfterMs   int64      `json:"retryAfterMs,omitzero"`
	ServerDirected bool       `json:"serverDirected,omitzero"`
	Code           string     `json:"code,omitzero"`
	HTTPStatus     int        `json:"httpStatus,omitzero"`
}

// CompactionReason is why a compaction runs.
type CompactionReason string

// The reasons: a client's request, the context past its threshold, or the
// model's window overflowed.
const (
	CompactionManual    CompactionReason = "manual"
	CompactionThreshold CompactionReason = "threshold"
	CompactionOverflow  CompactionReason = "overflow"
)

// CompactionPause is how long a failed automatic compaction pauses the
// automatic trigger: for the rest of the turn, or for the session. Its zero
// value is no pause (0005-PLAN F5-2 decision 11).
type CompactionPause string

// The pauses.
const (
	PauseTurn    CompactionPause = "turn"
	PauseSession CompactionPause = "session"
)

// CompactionNotice is MethodCompaction's params (0005-PLAN F5-2 decision
// 9). TokensBefore is a successful end's; Error is why an end is not a
// success; Paused is set when that failure paused automatic compaction.
type CompactionNotice struct {
	SessionID    string           `json:"sessionId"`
	Phase        Phase            `json:"phase"`
	Reason       CompactionReason `json:"reason"`
	TokensBefore int64            `json:"tokensBefore,omitzero"`
	Error        string           `json:"error,omitzero"`
	Paused       CompactionPause  `json:"paused,omitzero"`
}

// CompactionMeta is gobble's own facts about a compaction, sent under
// _meta[MetaKey] of ACP's standard compaction_update when the client offered
// session.compaction, where CompactionNotice is not sent (gobble-cli
// 0016-MADR D9, P3-Q3). The protocol replaces an update's _meta whole, so
// each update that carries it carries all of it: Reason on every update a
// live compaction sends, TokensBefore on its completed update and on a
// replayed one, and Paused on a failed one that paused automatic compaction.
type CompactionMeta struct {
	Reason       CompactionReason `json:"reason,omitzero"`
	TokensBefore int64            `json:"tokensBefore,omitzero"`
	Paused       CompactionPause  `json:"paused,omitzero"`
}

// PersonalityNotice is MethodPersonality's params: the session's
// personality, by its canonical name, and its level, 1 to 5 (0022-MADR D6).
type PersonalityNotice struct {
	SessionID string `json:"sessionId"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
}

// QueueMode is how a queue is taken: one message at each step, or all of
// them at once.
type QueueMode string

// The modes.
const (
	QueueOneAtATime QueueMode = "one-at-a-time"
	QueueAll        QueueMode = "all"
)

// Disposition is what became of a queued message.
type Disposition string

// DispositionQueued is the one disposition: the message is queued.
const DispositionQueued Disposition = "queued"

// SessionParams are the params of a request that names its session alone.
type SessionParams struct {
	SessionID string `json:"sessionId"`
}

// MessageParams are MethodSteer's and MethodFollowUp's params.
type MessageParams struct {
	SessionID string `json:"sessionId"`
	Message   string `json:"message"`
}

// ModeParams are MethodSetSteeringMode's and MethodSetFollowUpMode's
// params.
type ModeParams struct {
	SessionID string    `json:"sessionId"`
	Mode      QueueMode `json:"mode"`
}

// CompactParams are MethodCompact's params: the summary's extra focus, or
// "".
type CompactParams struct {
	SessionID          string `json:"sessionId"`
	CustomInstructions string `json:"customInstructions"`
}

// EnabledParams are MethodSetAutoRetry's and MethodSetAutoCompaction's
// params. Enabled must be set: a request without it is refused.
type EnabledParams struct {
	SessionID string `json:"sessionId"`
	Enabled   *bool  `json:"enabled"`
}

// SetSessionNameParams are MethodSetSessionName's params. A name that is
// empty once trimmed is refused.
type SetSessionNameParams struct {
	SessionID string `json:"sessionId"`
	Name      string `json:"name"`
}

// ForkParams are MethodFork's params: the user message's entry the fork
// ends before.
type ForkParams struct {
	SessionID string `json:"sessionId"`
	EntryID   string `json:"entryId"`
}

// GetEntriesParams are MethodGetEntries's params: Since, when set, is the
// entry after which the entries start.
type GetEntriesParams struct {
	SessionID string  `json:"sessionId"`
	Since     *string `json:"since"`
}

// SetPersonalityParams are MethodSetPersonality's params: the
// personality's name.
type SetPersonalityParams struct {
	SessionID string `json:"sessionId"`
	Name      string `json:"name"`
}

// QueuedResult is MethodSteer's and MethodFollowUp's result.
type QueuedResult struct {
	Disposition Disposition `json:"disposition"`
}

// ClearedResult is MethodClearQueue's result: the messages each queue held.
type ClearedResult struct {
	Steering []string `json:"steering"`
	FollowUp []string `json:"followUp"`
}

// CompactResult is MethodCompact's result. Details is gobble's compaction
// details, a session-file shape this package does not freeze.
type CompactResult struct {
	Summary          string         `json:"summary"`
	FirstKeptEntryID string         `json:"firstKeptEntryId"`
	TokensBefore     int64          `json:"tokensBefore"`
	Details          jsontext.Value `json:"details"`
}

// UsageResult is MethodUsage's result: /usage's text, and the figures it
// is made from.
type UsageResult struct {
	Text  string       `json:"text"`
	Stats SessionStats `json:"stats"`
}

// ForkResult is MethodFork's result: the new session's id, and the text of
// the message it ends before, for the client's editor.
type ForkResult struct {
	SessionID string `json:"sessionId"`
	Text      string `json:"text,omitzero"`
}

// CloneResult is MethodClone's result: the new session's id.
type CloneResult struct {
	SessionID string `json:"sessionId"`
}

// ModelRef is the model StateResult names.
type ModelRef struct {
	Provider string `json:"provider,omitzero"`
	ID       string `json:"id"`
}

// StateResult is MethodGetState's result. Mode is the session's ACP mode
// id; Personality its personality's name.
type StateResult struct {
	Model                 *ModelRef `json:"model,omitzero"`
	ThinkingLevel         string    `json:"thinkingLevel"`
	IsStreaming           bool      `json:"isStreaming"`
	IsCompacting          bool      `json:"isCompacting"`
	SteeringMode          QueueMode `json:"steeringMode"`
	FollowUpMode          QueueMode `json:"followUpMode"`
	SessionFile           string    `json:"sessionFile,omitzero"`
	SessionID             string    `json:"sessionId"`
	SessionName           string    `json:"sessionName,omitzero"`
	AutoCompactionEnabled bool      `json:"autoCompactionEnabled"`
	MessageCount          int       `json:"messageCount"`
	PendingMessageCount   int       `json:"pendingMessageCount"`
	Mode                  string    `json:"mode"`
	Cwd                   string    `json:"cwd"`
	Personality           string    `json:"personality"`
}

// MessagesResult is MethodGetMessages's result. Each message is gobble's
// session-file message, a shape this package does not freeze.
type MessagesResult struct {
	Messages []jsontext.Value `json:"messages"`
}

// EntriesResult is MethodGetEntries's result. Each entry is gobble's
// session-file entry, a shape this package does not freeze; LeafID is the
// tree's leaf, or nil when there is none.
type EntriesResult struct {
	Entries []jsontext.Value `json:"entries"`
	LeafID  *string          `json:"leafId"`
}

// TextResult is MethodGetLastAssistantText's result: the text, or nil when
// there is none.
type TextResult struct {
	Text *string `json:"text"`
}

// SessionStats is MethodGetSessionStats's result, and UsageResult's Stats.
// Cost is the sum of each usage's cost; ContextUsage is set when the
// model's window is known.
type SessionStats struct {
	SessionFile       string        `json:"sessionFile,omitzero"`
	SessionID         string        `json:"sessionId"`
	UserMessages      int           `json:"userMessages"`
	AssistantMessages int           `json:"assistantMessages"`
	ToolCalls         int           `json:"toolCalls"`
	ToolResults       int           `json:"toolResults"`
	TotalMessages     int           `json:"totalMessages"`
	Tokens            TokenStats    `json:"tokens"`
	Cost              float64       `json:"cost"`
	ContextUsage      *ContextUsage `json:"contextUsage,omitzero"`
}

// ContextUsage is the context's tokens, as the last usage update said, of
// the model's window.
type ContextUsage struct {
	Tokens        int     `json:"tokens"`
	ContextWindow int     `json:"contextWindow"`
	Percent       float64 `json:"percent"`
}

// TokenStats are a session's token counts.
type TokenStats struct {
	Input      int64 `json:"input"`
	Output     int64 `json:"output"`
	CacheRead  int64 `json:"cacheRead"`
	CacheWrite int64 `json:"cacheWrite"`
	Total      int64 `json:"total"`
}

// ForkMessage is a user message a fork may end before: its entry and its
// text.
type ForkMessage struct {
	EntryID string `json:"entryId"`
	Text    string `json:"text"`
}

// ForkMessagesResult is MethodGetForkMessages's result.
type ForkMessagesResult struct {
	Messages []ForkMessage `json:"messages"`
}

// PersonalityResult is MethodSetPersonality's result: the session's
// personality after it, which a declined level-5 question leaves as it was.
type PersonalityResult struct {
	Name  string `json:"name"`
	Level int    `json:"level"`
}

// ErrorKind is gobble's class of a failed turn: ErrorData.Kind. It is one
// of the constants below, but a client should take a kind it does not know,
// which a later gobble may add. Where a class is also a RetryCause, the
// name is the same.
type ErrorKind string

// The kinds (gobble-cli 0049-MADR D3).
const (
	KindWorkspace       ErrorKind = "workspace"
	KindQuota           ErrorKind = "quota"
	KindRateLimited     ErrorKind = "rate-limited"
	KindContextOverflow ErrorKind = "context-overflow"
	KindEmptyReply      ErrorKind = "empty_reply"
	KindIncomplete      ErrorKind = "incomplete"
	KindAuth            ErrorKind = "auth"
	KindNotPermitted    ErrorKind = "not-permitted"
	KindOffline         ErrorKind = "offline"
	KindUnavailable     ErrorKind = "unavailable"
	KindUnsupported     ErrorKind = "unsupported"
	KindTimeout         ErrorKind = "timeout"
	KindCanceled        ErrorKind = "canceled"
	KindOther           ErrorKind = "other"
)

// LimitScope is which limit a failed turn hit: ErrorData.Limit. It is set
// only when a source states it, from the provider's own code, and never from
// how long the wait is. It is one of the constants below, but a client should
// take a scope it does not know. gobble sends LimitWindow and LimitWeekly only
// when a source states the window.
type LimitScope string

// The scopes (gobble-cli 0049-MADR D1, D4).
const (
	LimitRate     LimitScope = "rate"      // throttling: a rate limit
	LimitWindow   LimitScope = "window"    // a rolling window of stated length
	LimitDaily    LimitScope = "daily"     // a per-day quota
	LimitWeekly   LimitScope = "weekly"    // a per-week quota
	LimitMonthly  LimitScope = "monthly"   // a per-month quota
	LimitCredits  LimitScope = "credits"   // a balance or billing, which does not lift by itself
	LimitFreeTier LimitScope = "free_tier" // a free tier's usage
	LimitPlan     LimitScope = "plan"      // a plan's usage limit, its window not stated
)

// ErrorData is the data of gobble's JSON-RPC errors. A refusal through a
// session carries SessionID and Reason; authenticate's -32000 carries
// Provider, Reason and Command; a failed prompt or compaction carries Error,
// or Reason when the session file could not be written.
//
// A failed prompt or compaction also carries Provider, gobble's id of the
// session's provider, and Kind; and, when the provider answered, what it
// said: HTTPStatus (absent for a failure inside a stream), Code, the
// provider's own code, ProviderMessage, the provider's redacted message
// without gobble's or its SDK's prefixes, RetryAfterMs, the wait it asked
// for, ResetAt, when the limit lifts, in RFC 3339 in UTC to the second,
// Retryable, whether the same request can succeed later (absent when gobble
// does not know), Attempts, the tries made when the failure came after
// retries, and Limit (gobble-cli 0049-MADR D1, D7). Error keeps the whole
// text. WindowMinutes is the limit's window as the service stated it, in
// whole minutes, whatever Limit is; it is absent when the service stated no
// length, and is never taken from the wait (gobble-cli 0049-MADR Q9).
//
// Its fields are in their JSON names' sorted order, which is the order
// gobble writes them.
type ErrorData struct {
	Attempts        int        `json:"attempts,omitzero"`
	Code            string     `json:"code,omitzero"`
	Command         string     `json:"command,omitzero"`
	Error           string     `json:"error,omitzero"`
	HTTPStatus      int        `json:"httpStatus,omitzero"`
	Kind            ErrorKind  `json:"kind,omitzero"`
	Limit           LimitScope `json:"limit,omitzero"`
	Provider        string     `json:"provider,omitzero"`
	ProviderMessage string     `json:"providerMessage,omitzero"`
	Reason          string     `json:"reason,omitzero"`
	ResetAt         string     `json:"resetAt,omitzero"`
	RetryAfterMs    int64      `json:"retryAfterMs,omitzero"`
	Retryable       *bool      `json:"retryable,omitzero"`
	SessionID       string     `json:"sessionId,omitzero"`
	WindowMinutes   int        `json:"windowMinutes,omitzero"`
}
