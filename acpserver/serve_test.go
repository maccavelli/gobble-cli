package acpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/maccavelli/gobble-cli/llm"
)

// errBroken is a read failure, as a broken stdin gives one.
var errBroken = errors.New("the input broke")

type brokenReader struct{}

func (brokenReader) Read([]byte) (int, error) { return 0, errBroken }

// Serve returns the input's error when its input fails, and nil at its end
// (0031-MADR D5).
func TestServeReportsABrokenInput(t *testing.T) {
	if err := Serve(t.Context(), brokenReader{}, io.Discard, ServeOptions{}); !errors.Is(err, errBroken) {
		t.Fatalf("Serve over a broken input: %v, want its error", err)
	}
	if err := Serve(t.Context(), strings.NewReader(""), io.Discard, ServeOptions{}); err != nil {
		t.Fatalf("Serve at the end of its input: %v, want nil", err)
	}
}

// A request written to `gobble acp` and followed at once by the end of its
// input is answered before Serve returns: a script that pipes requests in
// and closes stdin gets its replies (acp-go-sdk2 issue 8, fixed in its
// v1.15.2; 0023-MADR D4).
func TestServeAnswersBeforeEndOfInput(t *testing.T) {
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":1,"clientCapabilities":{}}}` + "\n")
	var out bytes.Buffer
	done := make(chan error, 1)
	go func() { done <- Serve(t.Context(), in, &out, ServeOptions{Agent: Options{Version: "1.2.3"}}) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Serve did not return within 10 s of its input ending")
	}
	var reply struct {
		ID     int `json:"id"`
		Result struct {
			ProtocolVersion int `json:"protocolVersion"`
		} `json:"result"`
	}
	line, _, _ := bytes.Cut(out.Bytes(), []byte("\n"))
	if err := json.Unmarshal(line, &reply); err != nil || reply.ID != 1 || reply.Result.ProtocolVersion != 1 {
		t.Fatalf("no answer to initialize before the end of input (%v): %q", err, out.String())
	}
}

// When Serve's context ends during a turn, the SDK's graceful shutdown ends
// the turn's context, and the prompt is answered cancelled before the
// connection closes; Serve returns the context's cause (0035-MADR D11).
func TestServeShutdownCancelsPrompt(t *testing.T) {
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	model := slowStop{started: make(chan struct{}, 1)}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- Serve(ctx, inR, outW, ServeOptions{Agent: Options{Version: "1.2.3", NewID: counterIDs(),
			Provider: func(string, llm.Retry) (llm.Provider, error) { return model, nil }, Models: func() ModelChoice { return testModels }}})
		outW.Close() //nolint:errcheck,gosec // test pipe
	}()
	send := func(id int, method, params string) {
		t.Helper()
		if _, err := io.WriteString(inW, `{"jsonrpc":"2.0","id":`+itoa(id)+`,"method":"`+method+`","params":`+params+"}\n"); err != nil {
			t.Fatal(err)
		}
	}
	lines := make(chan string, 64)
	go func() {
		r := bufio.NewReader(outR)
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				close(lines)
				return
			}
			lines <- line
		}
	}()
	answer := func(id int) string {
		t.Helper()
		for {
			select {
			case line, ok := <-lines:
				if !ok {
					t.Fatalf("output ended before the answer to %d", id)
				}
				var frame struct {
					ID     *int   `json:"id"`
					Method string `json:"method"`
				}
				if json.Unmarshal([]byte(line), &frame) == nil && frame.ID != nil && *frame.ID == id && frame.Method == "" {
					return line
				}
			case <-time.After(10 * time.Second):
				t.Fatalf("no answer to %d within 10 s", id)
			}
		}
	}
	send(1, "initialize", `{"protocolVersion":1,"clientCapabilities":{}}`)
	answer(1)
	send(2, "session/new", `{"cwd":`+strconv.Quote(t.TempDir())+`,"mcpServers":[]}`)
	answer(2)
	send(3, "session/prompt", `{"sessionId":"s1","prompt":[{"type":"text","text":"go"}]}`)
	select {
	case <-model.started:
	case <-time.After(10 * time.Second):
		t.Fatal("the turn never started")
	}
	cancel()
	if line := answer(3); !strings.Contains(line, `"stopReason":"cancelled"`) {
		t.Fatalf("the prompt's answer at shutdown: %s, want stopReason cancelled", line)
	}
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Serve: %v, want context.Canceled", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Serve did not return within 10 s of its context ending")
	}
}
