package acptest

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
)

// cleanupTB is a testing.TB whose cleanups the test runs itself, recording
// the errors they report.
type cleanupTB struct {
	testing.TB
	mu       sync.Mutex
	cleanups []func()
	errs     []string
}

func (c *cleanupTB) Helper() {}

func (c *cleanupTB) Cleanup(f func()) { c.cleanups = append(c.cleanups, f) }

func (c *cleanupTB) Errorf(format string, args ...any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errs = append(c.errs, fmt.Sprintf(format, args...))
}

func (c *cleanupTB) run() {
	for _, f := range slices.Backward(c.cleanups) {
		f()
	}
}

// hangClient's session/update handler holds until release closes,
// whatever its context says. Notifications are handled off the reader, so
// the connection still sees its input end.
type hangClient struct {
	Client
	entered, release chan struct{}
	once             sync.Once
}

func (c *hangClient) SessionUpdate(context.Context, acp.SessionNotification) error {
	c.once.Do(func() { close(c.entered) })
	<-c.release
	return nil
}

// Connect's cleanup fails a test whose client's handler outlives the
// connection, within its timeout (0031-MADR D5).
func TestCleanupWaitsForHandlers(t *testing.T) {
	old := closeTimeout
	closeTimeout = 300 * time.Millisecond
	t.Cleanup(func() { closeTimeout = old })
	agent := &stubAgent{}
	client := &hangClient{entered: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() { close(client.release) })
	fake := &cleanupTB{TB: t}
	c := Connect(fake, agent, client)
	agent.conn = c.Agent
	initialize(t, c.Client)
	if _, err := c.Client.NewSession(t.Context(), acp.NewSessionRequest{Cwd: Abs("/work"), McpServers: []acp.McpServer{}}); err != nil {
		t.Fatal(err)
	}
	// The prompt's answer waits for its update to be handled, which holds.
	go func() { //nolint:errcheck // the prompt ends with the connection
		_, _ = c.Client.Prompt(context.WithoutCancel(t.Context()), acp.PromptRequest{SessionId: "s1",
			Prompt: []acp.ContentBlock{acp.TextBlock("hi")}})
	}()
	select {
	case <-client.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("the client's update handler never ran")
	}
	start := time.Now()
	fake.run()
	t.Logf("the cleanup reported %q", fake.errs)
	if len(fake.errs) == 0 {
		t.Fatal("the cleanup passed with the client's update handler still running")
	}
	if d := time.Since(start); d > 2*time.Second {
		t.Fatalf("the cleanup took %v, past its timeout of 300ms", d)
	}
}
