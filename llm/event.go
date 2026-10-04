package llm

import "encoding/json/jsontext"

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
)

// Done ends a stream.
type Done struct {
	StopReason StopReason `json:"stopReason,omitzero"`
}

func (Done) event() {}
