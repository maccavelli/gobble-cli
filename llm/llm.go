package llm

import (
	"context"
	"iter"

	"encoding/json/jsontext"
)

// Provider is the agent-facing model contract.
// Implementations live outside this phase. Stream yields a sealed Event
// and a terminal error. Cancel the context to stop.
type Provider interface {
	ID() string
	Capabilities() Capabilities
	Stream(ctx context.Context, req *Request) iter.Seq2[Event, error]
}

// Capabilities says what this process can actually send.
// ImageInput stays false until the SDK accepts image input.
// NativeStreaming stays false while the SDK's retry wrapper buffers the stream (0005-PLAN F4-1b).
type Capabilities struct {
	ImageInput      bool `json:"imageInput,omitzero"`
	NativeStreaming bool `json:"nativeStreaming,omitzero"`
}

// CacheRetention asks the model service to cache a request's prompt, and
// for how long (0021-MADR D1). CacheNone asks for nothing; CacheShort is
// the service's short lifetime, five minutes on Anthropic; CacheLong its
// longest, an hour on Anthropic, and CacheShort's where it has one. The
// values are go-llmprovider-sdk's.
type CacheRetention string

// The retentions.
const (
	CacheNone  CacheRetention = ""
	CacheShort CacheRetention = "short"
	CacheLong  CacheRetention = "long"
)

// Request is one model call.
// Tools is a JSON array of tool specs. This package does not interpret it.
type Request struct {
	Model    string         `json:"model,omitzero"`
	Messages []Message      `json:"messages,omitzero"`
	Tools    jsontext.Value `json:"tools,omitzero"`
	Thinking string         `json:"thinking,omitzero"`
	// Cache asks the service to cache the prompt; CacheNone asks for nothing.
	Cache CacheRetention `json:"cache,omitzero"`
}

// Role is the author of a message.
type Role string

const (
	// RoleSystem is the system prompt.
	RoleSystem Role = "system"
	// RoleUser is the end user.
	RoleUser Role = "user"
	// RoleAssistant is the model.
	RoleAssistant Role = "assistant"
	// RoleTool is a tool result.
	RoleTool Role = "tool"
)

// Message is one turn in a Request.
type Message struct {
	Role    Role      `json:"role"`
	Content []Content `json:"content,omitzero"`
}

// Content is one block inside a Message. Type is one of the Content*
// constants. Image fields are accepted by the struct and are not sent
// anywhere yet.
//
// A tool call is a ContentToolCall block in an assistant message: ToolCallID,
// ToolName and ToolArgs (the arguments as JSON text), with any provider
// signature in ProviderData. Its result is a ContentToolResult block in a
// RoleTool message: ToolCallID, the output in Text, and IsError.
type Content struct {
	Type         string         `json:"type"`
	Text         string         `json:"text,omitzero"`
	MediaType    string         `json:"mediaType,omitzero"`
	Data         []byte         `json:"data,omitzero"`
	ProviderData jsontext.Value `json:"providerData,omitzero"`
	ToolCallID   string         `json:"toolCallId,omitzero"`
	ToolName     string         `json:"toolName,omitzero"`
	ToolArgs     string         `json:"toolArgs,omitzero"`
	IsError      bool           `json:"isError,omitzero"`
}

// Content types.
const (
	ContentText       = "text"
	ContentImage      = "image"
	ContentThinking   = "thinking"
	ContentToolCall   = "tool_call"
	ContentToolResult = "tool_result"
)

// Model is a catalog row the facade can name.
// llm/provider fills it from the catalog (0005-PLAN F4-2a).
type Model struct {
	Provider string `json:"provider,omitzero"`
	ID       string `json:"id"`
	Name     string `json:"name,omitzero"`
	// ContextWindow and MaxTokens are the model's window and output cap, in
	// tokens; 0 is unknown.
	ContextWindow int `json:"contextWindow,omitzero"`
	MaxTokens     int `json:"maxTokens,omitzero"`
	// Input are the modalities the model reads, such as text and image.
	Input []string `json:"input,omitzero"`
	// Reasoning reports whether the model reasons, when ReasoningKnown;
	// Efforts are the effort levels its source lists, in the source's names.
	Reasoning      bool     `json:"reasoning,omitzero"`
	ReasoningKnown bool     `json:"-"`
	Efforts        []string `json:"efforts,omitzero"`
	// Cost is the model's price, when Priced.
	Cost   ModelCost `json:"cost,omitzero"`
	Priced bool      `json:"-"`
}

// ModelCost is a model's price, in USD per million tokens: its base rates,
// and the rates for longer inputs, ascending by Above.
type ModelCost struct {
	Input      float64    `json:"input"`
	Output     float64    `json:"output"`
	CacheRead  float64    `json:"cacheRead"`
	CacheWrite float64    `json:"cacheWrite"`
	Tiers      []CostTier `json:"tiers,omitzero"`
}

// CostTier is the price of an input of more than Above tokens.
type CostTier struct {
	Above      int     `json:"above"`
	Input      float64 `json:"input"`
	Output     float64 `json:"output"`
	CacheRead  float64 `json:"cacheRead"`
	CacheWrite float64 `json:"cacheWrite"`
}

// Cost is a currency amount for a turn.
type Cost struct {
	Currency string  `json:"currency,omitzero"`
	Input    float64 `json:"input,omitzero"`
	Output   float64 `json:"output,omitzero"`
	Total    float64 `json:"total,omitzero"`
}
