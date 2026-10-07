package acpserver

import (
	"slices"
	"sync"

	"github.com/maccavelli/gobble-cli/llm"
)

// The queue modes (Pi's QueueMode): a drain takes the first message, or all
// of them.
const (
	queueOneAtATime = "one-at-a-time"
	queueAll        = "all"
)

// queue is a session's steering and follow-up queues (0002-PLAN Phase 7).
// It is filled by _gobble/steer and _gobble/follow_up while a turn runs, and
// drained by the turn, so it has a mutex of its own.
type queue struct {
	mu                         sync.Mutex
	steering, followUp         []string
	steeringMode, followUpMode string
}

func newQueue() *queue {
	return &queue{steeringMode: queueOneAtATime, followUpMode: queueOneAtATime}
}

func validQueueMode(m string) bool { return m == queueOneAtATime || m == queueAll }

func (q *queue) steer(text string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.steering = append(q.steering, text)
}

func (q *queue) followUpWith(text string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.followUp = append(q.followUp, text)
}

// clear empties both queues and returns what they held, in order.
func (q *queue) clear() (steering, followUp []string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	steering, followUp = q.steering, q.followUp
	q.steering, q.followUp = nil, nil
	if steering == nil {
		steering = []string{}
	}
	if followUp == nil {
		followUp = []string{}
	}
	return steering, followUp
}

func (q *queue) setSteeringMode(m string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.steeringMode = m
}

func (q *queue) setFollowUpMode(m string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.followUpMode = m
}

// modes are the steering and follow-up modes.
func (q *queue) modes() (steering, followUp string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.steeringMode, q.followUpMode
}

// pending is the number of queued messages.
func (q *queue) pending() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.steering) + len(q.followUp)
}

// Steering drains the steering queue by its mode.
func (q *queue) Steering() []llm.Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	var out []string
	out, q.steering = drain(q.steering, q.steeringMode)
	return userMessages(out)
}

// FollowUps drains the follow-up queue by its mode.
func (q *queue) FollowUps() []llm.Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	var out []string
	out, q.followUp = drain(q.followUp, q.followUpMode)
	return userMessages(out)
}

// drain is Pi's PendingMessageQueue.drain: the first message, or all.
func drain(texts []string, mode string) (out, rest []string) {
	if len(texts) == 0 {
		return nil, texts
	}
	if mode == queueAll {
		return texts, nil
	}
	return texts[:1], slices.Clone(texts[1:])
}

func userMessages(texts []string) []llm.Message {
	out := make([]llm.Message, len(texts))
	for i, t := range texts {
		out[i] = llm.Message{Role: llm.RoleUser, Content: []llm.Content{{Type: llm.ContentText, Text: t}}}
	}
	return out
}
