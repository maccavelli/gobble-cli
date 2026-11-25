package llm

import (
	"encoding/json/jsontext"
	"time"
)

// ProviderRetry is a retry the provider makes before any of the call's
// events: the try that failed, of how many, the wait about to begin, and
// the failure's class (0031-MADR D4). The call's events follow from the
// try that succeeds.
//
// RetryAfter is the wait the provider asked for, where Wait is the one
// taken, jitter included; ServerDirected reports that the provider asked for
// it; Code and Status are the failure's code and HTTP status, each zero when
// not known (0049-MADR D6).
type ProviderRetry struct {
	Attempt, MaxAttempts int
	Wait                 time.Duration
	Kind                 string

	RetryAfter     time.Duration
	ServerDirected bool
	Code           string
	Status         int
}

func (ProviderRetry) event() {}

// Event is the sealed stream value.
// The unexported method keeps the set closed.
type Event interface {
	event()
}

// TextDelta is a piece of assistant text.
type TextDelta struct {
	Text string `json:"text,omitzero"`
}

func (TextDelta) event() {}

// ThinkingDelta is a piece of model reasoning.
type ThinkingDelta struct {
	Text         string         `json:"text,omitzero"`
	ProviderData jsontext.Value `json:"providerData,omitzero"`
}

func (ThinkingDelta) event() {}

// ToolCallStart marks the start of a tool call.
type ToolCallStart struct {
	ID   string `json:"id,omitzero"`
	Name string `json:"name,omitzero"`
}

func (ToolCallStart) event() {}

// ToolCallDelta is a partial tool-call argument.
type ToolCallDelta struct {
	ID   string `json:"id,omitzero"`
	Args string `json:"args,omitzero"`
}

func (ToolCallDelta) event() {}

// ToolCallDone is a finished tool call.
type ToolCallDone struct {
	ID           string         `json:"id,omitzero"`
	Name         string         `json:"name,omitzero"`
	Args         string         `json:"args,omitzero"`
	ProviderData jsontext.Value `json:"providerData,omitzero"`
}

func (ToolCallDone) event() {}

// ReasoningRefusal is a thinking setting a model refused: its model, and an
// effort or a shape, one of the two (0036-MADR D4).
type ReasoningRefusal struct {
	Model  string `json:"model,omitzero"`
	Effort string `json:"effort,omitzero"`
	Shape  string `json:"shape,omitzero"`
}

// Warning is a note that the request was answered, but not as asked
// (0036-MADR D3). Kind is the SDK's string. Model and Detail are its own.
// Refused is set on a reasoning_learned warning (0036-MADR D4).
type Warning struct {
	Kind    string           `json:"kind,omitzero"`
	Model   string           `json:"model,omitzero"`
	Detail  string           `json:"detail,omitzero"`
	Refused ReasoningRefusal `json:"refused,omitzero"`
}

func (Warning) event() {}

const (
	// WarningEffortDegraded is the SDK's effort_degraded.
	WarningEffortDegraded = "effort_degraded"
	// WarningReasoningLearned is the SDK's reasoning_learned.
	WarningReasoningLearned = "reasoning_learned"
	// WarningImageDropped is the SDK's image_dropped.
	WarningImageDropped = "image_dropped"
	// WarningDeferralEmulated is the SDK's deferral_emulated. The agent logs
	// it at Debug, once per model (0012-MADR D21).
	WarningDeferralEmulated = "deferral_emulated"
	// WarningToolCallCut is the SDK's tool_call_cut: the output limit cut a
	// tool call before its arguments were whole, and the response dropped
	// it (0071-MADR P9, P15, P29).
	WarningToolCallCut = "tool_call_cut"
	// WarningLimitLearned is the SDK's limit_learned: a cap a service stated
	// when it refused a limit the SDK chose, kept for the process
	// (0071-MADR P15).
	WarningLimitLearned = "limit_learned"
)

// Usage is the token accounting for a turn.
// Estimated is true when the counts were not reported by the provider.
type Usage struct {
	InputTokens      int64 `json:"inputTokens,omitzero"`
	OutputTokens     int64 `json:"outputTokens,omitzero"`
	ReasoningTokens  int64 `json:"reasoningTokens,omitzero"`
	CacheReadTokens  int64 `json:"cacheReadTokens,omitzero"`
	CacheWriteTokens int64 `json:"cacheWriteTokens,omitzero"`
	Estimated        bool  `json:"estimated,omitzero"`
}

func (Usage) event() {}

// StopReason is why the model stopped.
type StopReason string

const (
	// StopEndTurn means the model finished its reply.
	StopEndTurn StopReason = "end_turn"
	// StopToolUse means the model asked for a tool.
	StopToolUse StopReason = "tool_use"
	// StopMaxTokens means the output limit was hit.
	StopMaxTokens StopReason = "max_tokens"
	// StopRefusal means the service stopped the reply for safety, or the
	// model refused (0005-PLAN F5-3 decision 5).
	StopRefusal StopReason = "refusal"
)

// Done ends a stream. OutputLimit is the output limit the call carried on
// the wire, as the provider reports it; zero when it reports none
// (0071-MADR P14).
type Done struct {
	StopReason  StopReason `json:"stopReason,omitzero"`
	OutputLimit SentLimit  `json:"outputLimit,omitzero"`
}

func (Done) event() {}
