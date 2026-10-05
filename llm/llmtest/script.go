package llmtest

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"slices"
	"sync"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
)

// ErrExhausted is the error of a Stream call made after the last turn.
var ErrExhausted = errors.New("llmtest: script exhausted")

// Turn is one scripted answer to a Stream call: its events, in order, and
// an optional error that ends the stream after them.
type Turn struct {
	Events []llm.Event
	Err    error
}

// Script is an llm.Provider that answers each Stream call with the next
// Turn and records every request. It is safe for concurrent use.
type Script struct {
	mu       sync.Mutex
	turns    []Turn
	requests []*llm.Request
	caps     llm.Capabilities
}

var _ llm.Provider = (*Script)(nil)

// NewScript returns a Script that answers with turns, in order.
func NewScript(turns ...Turn) *Script {
	return &Script{turns: slices.Clone(turns)}
}

// WithCapabilities sets what Capabilities reports, and returns s.
func (s *Script) WithCapabilities(c llm.Capabilities) *Script {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.caps = c
	return s
}

// ID is "llmtest".
func (*Script) ID() string { return "llmtest" }

// Capabilities reports what WithCapabilities set.
func (s *Script) Capabilities() llm.Capabilities {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.caps
}

// Stream records a copy of req and plays the next turn. It stops with the
// context's error once ctx is done, and yields ErrExhausted when no turn is
// left.
func (s *Script) Stream(ctx context.Context, req *llm.Request) iter.Seq2[llm.Event, error] {
	s.mu.Lock()
	s.requests = append(s.requests, cloneRequest(req))
	var turn Turn
	exhausted := len(s.turns) == 0
	if !exhausted {
		turn, s.turns = s.turns[0], s.turns[1:]
	}
	s.mu.Unlock()
	return func(yield func(llm.Event, error) bool) {
		if exhausted {
			yield(nil, ErrExhausted)
			return
		}
		for _, ev := range turn.Events {
			if err := ctx.Err(); err != nil {
				yield(nil, err)
				return
			}
			if !yield(ev, nil) {
				return
			}
		}
		if turn.Err != nil {
			yield(nil, turn.Err)
		}
	}
}

// Requests returns copies of the requests Stream received, in order.
func (s *Script) Requests() []*llm.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*llm.Request, len(s.requests))
	for i, r := range s.requests {
		out[i] = cloneRequest(r)
	}
	return out
}

// Remaining is the number of turns not yet played.
func (s *Script) Remaining() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.turns)
}

func cloneRequest(r *llm.Request) *llm.Request {
	if r == nil {
		return nil
	}
	c := *r
	c.Tools = slices.Clone(r.Tools)
	c.Messages = make([]llm.Message, len(r.Messages))
	for i, m := range r.Messages {
		c.Messages[i] = llm.Message{Role: m.Role, Content: slices.Clone(m.Content)}
	}
	return &c
}

// Text is a turn that answers s and ends the turn.
func Text(s string) Turn {
	return Turn{Events: []llm.Event{
		llm.TextDelta{Text: s},
		llm.Usage{InputTokens: 10, OutputTokens: int64(len(s))},
		llm.Done{StopReason: llm.StopEndTurn},
	}}
}

// ToolCall is a turn that calls one tool and stops for its result.
func ToolCall(id, name, args string) Turn {
	return Turn{Events: []llm.Event{
		llm.ToolCallStart{ID: id, Name: name},
		llm.ToolCallDone{ID: id, Name: name, Args: args},
		llm.Done{StopReason: llm.StopToolUse},
	}}
}

// RateLimited is a turn that fails with HTTP 429, asking the caller to wait
// retryAfter (zero: no Retry-After).
func RateLimited(retryAfter time.Duration) Turn {
	return Turn{Err: &llm.APIError{Status: 429, RetryAfter: retryAfter, Err: fmt.Errorf("%w (scripted)", llm.ErrRateLimited)}}
}
