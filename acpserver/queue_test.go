package acpserver

import (
	"slices"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
)

func texts(msgs []llm.Message) []string {
	out := []string{}
	for _, m := range msgs {
		out = append(out, m.Content[0].Text)
	}
	return out
}

// The modes are Pi's PendingMessageQueue: one-at-a-time drains the first
// message, all drains every one; each queue has its own.
func TestQueueModes(t *testing.T) {
	q := newQueue()
	for _, s := range []string{"s1", "s2"} {
		q.steer(s)
	}
	for _, f := range []string{"f1", "f2"} {
		q.followUpWith(f)
	}
	if got := texts(q.Steering()); !slices.Equal(got, []string{"s1"}) {
		t.Fatalf("one-at-a-time steering drained %q", got)
	}
	q.setFollowUpMode(queueAll)
	if got := texts(q.FollowUps()); !slices.Equal(got, []string{"f1", "f2"}) {
		t.Fatalf("all follow-ups drained %q", got)
	}
	if q.pending() != 1 {
		t.Fatalf("pending %d, want 1", q.pending())
	}
	if s, f := q.modes(); s != queueOneAtATime || f != queueAll {
		t.Fatalf("modes %s, %s", s, f)
	}
	if got := texts(q.Steering()); !slices.Equal(got, []string{"s2"}) {
		t.Fatalf("second steering drain %q", got)
	}
	if got := q.Steering(); len(got) != 0 {
		t.Fatalf("an empty queue drained %d", len(got))
	}
}

// clear returns both queues in order, as empty lists when empty.
func TestQueueClear(t *testing.T) {
	q := newQueue()
	if s, f := q.clear(); s == nil || f == nil || len(s)+len(f) != 0 {
		t.Fatalf("empty clear %v, %v", s, f)
	}
	q.steer("a")
	q.steer("b")
	q.followUpWith("c")
	s, f := q.clear()
	if !slices.Equal(s, []string{"a", "b"}) || !slices.Equal(f, []string{"c"}) || q.pending() != 0 {
		t.Fatalf("clear %q, %q, pending %d", s, f, q.pending())
	}
	if !validQueueMode(queueAll) || !validQueueMode(queueOneAtATime) || validQueueMode("some") {
		t.Fatal("validQueueMode")
	}
}
