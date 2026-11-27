//go:build linux || darwin

package cli

import (
	"bytes"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

func init() { interactiveEndCase = runEndInteractiveCommand }

// M28's interactive case, command-level: the interactive CLI itself, Main on
// a pseudo-terminal in this process, with the real trigger: a prompt's model
// call starts a refresh whose fetch blocks; Ctrl+D at the next prompt ends
// the run, which exits at once, not waiting for its refresh, and the
// refresh is recorded canceled (0072-MADR D20, M28, M84; the PLAN's Phase 2
// pass 2, R4). Windows' twin is the conpty test's, on the Windows laptop.
func runEndInteractiveCommand(t *testing.T) {
	newDataRun(t)
	realTrigger(t)
	withModel(t, llmtest.NewScript(llmtest.Text("done")))
	t.Setenv("OPENAI_API_KEY", "sk-test-only") // a credential: no first-run sign-in offer
	started := fetchBlocks(t)
	master, slave, err := openDataPty()
	if err != nil {
		t.Skipf("no pseudo-terminal here: %v", err)
	}
	t.Cleanup(func() {
		_ = master.Close() //nolint:errcheck // test cleanup
		_ = slave.Close()  //nolint:errcheck // test cleanup
	})
	screen := &ptyScreen{}
	go screen.drain(master)
	exited, code := make(chan time.Time, 1), make(chan int, 1)
	go func() {
		c := Main(t.Context(), []string{"--approve", "--cwd", t.TempDir()}, slave, slave, slave)
		exited <- time.Now()
		code <- c
	}()
	if !screen.waitFor("> ", 20*time.Second) {
		t.Fatalf("no prompt within 20 s; the screen:\n%s", screen.text())
	}
	if _, err := master.WriteString("hello\r"); err != nil {
		t.Fatal(err)
	}
	waitClosed(t, started, "the model call's refresh")
	if !screen.waitFor("done", 10*time.Second) || !screen.waitForCount("> ", 2, 10*time.Second) {
		t.Fatalf("the reply and the next prompt did not come; the screen:\n%s", screen.text())
	}
	from := time.Now()
	if _, err := master.WriteString("\x04"); err != nil { // Ctrl+D at an empty prompt ends the session
		t.Fatal(err)
	}
	var to time.Time
	select {
	case to = <-exited:
	case <-time.After(20 * time.Second):
		t.Fatalf("the interactive CLI did not exit within 20 s of Ctrl+D; the screen:\n%s", screen.text())
	}
	if c := <-code; c != ExitOK || to.Sub(from) >= 2*time.Second {
		t.Errorf("exit %d, %v after Ctrl+D; want 0 within 2 s, not waiting for its refresh", c, to.Sub(from))
	}
	dirs, _, err := systemDirs()
	if err != nil {
		t.Fatal(err)
	}
	_, attemptPath := modelDataPaths(dirs)
	waitRecorded(t, attemptPath)
}

// ptyScreen keeps what the terminal shows, read from its master.
type ptyScreen struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (s *ptyScreen) drain(master *os.File) {
	b := make([]byte, 4096)
	for {
		n, err := master.Read(b)
		s.mu.Lock()
		s.buf.Write(b[:n])
		s.mu.Unlock()
		if err != nil {
			return
		}
	}
}

func (s *ptyScreen) text() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.buf.String()
}

// waitFor waits, bounded, until the screen holds want.
func (s *ptyScreen) waitFor(want string, within time.Duration) bool {
	return s.waitForCount(want, 1, within)
}

// waitForCount waits, bounded, until the screen holds want n times.
func (s *ptyScreen) waitForCount(want string, n int, within time.Duration) bool {
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if strings.Count(s.text(), want) >= n {
			return true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return false
}
