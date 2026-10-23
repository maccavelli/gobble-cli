package acpserver

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
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
