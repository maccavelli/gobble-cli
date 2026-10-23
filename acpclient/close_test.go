package acpclient_test

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/acpclient"
	"github.com/maccavelli/gobble-cli/acpclient/acptest"
)

// handled starts a connection to an agent whose one turn sends a message
// chunk, and a session whose handler runs hold on that chunk; it returns
// the connection once the handler is running.
func handled(t *testing.T, hold func()) *acpclient.Conn {
	t.Helper()
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{
		`{"sessionUpdate":"agent_message_chunk","content":{"type":"text","text":"hi"}}`}}}}
	c, err := acpclient.Start(t.Context(), agent.Serve, acpclient.Options{})
	if err != nil {
		t.Fatal(err)
	}
	entered := make(chan struct{})
	var once sync.Once
	s, err := c.NewSession(t.Context(), acptest.Abs("/w"), func(u acpclient.Update) {
		if u.Kind == acpclient.KindAgentText {
			once.Do(func() { close(entered) })
			hold()
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _, _ = s.Prompt(context.WithoutCancel(t.Context()), acpclient.Prompt{Text: "hi"}) }() //nolint:errcheck // the turn ends with the connection
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the session's handler never ran")
	}
	return c
}

// Close returns only once the session's update handlers have (0031-MADR
// D5).
func TestCloseWaitsForHandlers(t *testing.T) {
	var finished atomic.Bool
	c := handled(t, func() {
		time.Sleep(300 * time.Millisecond)
		finished.Store(true)
	})
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
	if !finished.Load() {
		t.Fatal("Close returned while a session update was still being handled")
	}
}

// A handler that never returns holds Close no longer than its timeout, and
// Close says so (0031-MADR D5).
func TestCloseBoundedWhenAHandlerHangs(t *testing.T) {
	acpclient.SetCloseTimeout(t, 300*time.Millisecond)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	c := handled(t, func() { <-release })
	start := time.Now()
	err := c.Close()
	if err == nil || !strings.Contains(err.Error(), "did not stop") {
		t.Fatalf("Close: %v, want an error saying the handlers did not stop", err)
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("Close took %v, past its timeout of 300ms", d)
	}
}
