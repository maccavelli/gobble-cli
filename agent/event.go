package agent

import (
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/tool"
)

// Event is one step of a turn. The set is sealed.
type Event interface{ event() }

// TextDelta is a piece of the model's reply.
type TextDelta struct{ Text string }

// ThinkingDelta is a piece of the model's reasoning.
type ThinkingDelta struct{ Text string }

// ToolStart is a tool call the model made, before it is approved. Title is
// the tool's Describe of it.
type ToolStart struct {
	Call  tool.Call
	Spec  tool.Spec
	Title string
}

// ToolRun is a call that was approved and is running.
type ToolRun struct{ Call tool.Call }

// ToolEnd is a call's result. A declined or failed call has IsError set.
type ToolEnd struct {
	Call   tool.Call
	Result tool.Result
}

// Usage is one model call's token count.
type Usage struct{ llm.Usage }

// Message is the assistant message of one model call, complete, with that
// call's usage and stop reason. It comes before the call's tools run, so a
// caller can record it first (0008-MADR D19 item 6).
type Message struct {
	Message    llm.Message
	Usage      llm.Usage
	StopReason llm.StopReason
}

// Queued is a queued user message entering the turn, before the model call
// that reads it, so a caller can record it first.
type Queued struct{ Message llm.Message }

// Loaded is deferred tools a call's result loaded, with their families, in
// the order they join the requests. It comes after that call's ToolEnd, and
// they are sent from the next model call on (0012-MADR D9).
type Loaded struct{ Names []string }

// End is the turn's last event. Messages are the turn's messages, the
// prompt first, for the caller to append to the history; Usage sums the
// turn's model calls.
type End struct {
	StopReason StopReason
	Messages   []llm.Message
	Usage      llm.Usage
}

func (TextDelta) event()     {}
func (ThinkingDelta) event() {}
func (ToolStart) event()     {}
func (ToolRun) event()       {}
func (ToolEnd) event()       {}
func (Usage) event()         {}
func (Message) event()       {}
func (Queued) event()        {}
func (Loaded) event()        {}
func (End) event()           {}

// StopReason is why a turn ended. The values are ACP's stop reasons.
type StopReason string

// Stop reasons.
const (
	StopEndTurn         StopReason = "end_turn"
	StopMaxTokens       StopReason = "max_tokens"
	StopMaxTurnRequests StopReason = "max_turn_requests"
	StopCancelled       StopReason = "cancelled"
)
