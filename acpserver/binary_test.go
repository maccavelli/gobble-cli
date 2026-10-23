package acpserver

import (
	"bytes"
	"encoding/json/v2"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"
	"github.com/maccavelli/acp-go-sdk2/agentproc"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

var buildGobble = sync.OnceValues(func() (string, error) {
	dir, err := os.MkdirTemp("", "gobble-acp-test-")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(dir, "gobble")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	root, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}").Output() //nolint:noctx // sync.OnceValues has no context
	if err != nil {
		return "", err
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/gobble") //nolint:noctx // as above
	cmd.Dir = strings.TrimSpace(string(root))
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", &buildError{out: string(out), err: err}
	}
	return bin, nil
})

// withoutCredentials is env with every model credential variable removed,
// so the child cannot reach a real provider.
func withoutCredentials(env []string) []string {
	vars := provider.CredentialVars()
	return slices.DeleteFunc(slices.Clone(env), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.ContainsFunc(vars, func(v string) bool { return strings.EqualFold(v, name) })
	})
}

type buildError struct {
	out string
	err error
}

func (e *buildError) Error() string { return "go build: " + e.err.Error() + "\n" + e.out }

// syncBuffer is a bytes.Buffer safe for the SDK's reader goroutine and the
// test reading it.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// `gobble acp` with the SDK's client over the child's stdio, and no model
// credential in its environment: initialize and session/new succeed, the
// prompt fails with auth_required and no network call, stdout carries only
// JSON-RPC, and closing stdin ends the process with 0 within 2 s (0002-PLAN
// Phases 1 and 3; 0008-MADR D19 item 6).
func TestGobbleACPBinary(t *testing.T) {
	bin, err := buildGobble()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), bin, "acp")
	cmd.Env = append(withoutCredentials(os.Environ()), "GOBBLE_HOME="+t.TempDir())
	var stderr, wire syncBuffer
	cmd.Stderr = &stderr
	// agentproc owns the pipes, records the wire, and on Close ends the
	// agent's input and waits Grace for it to exit (0031-MADR D8).
	proc, err := (&agentproc.Command{Cmd: cmd, Grace: 2 * time.Second, OutputTap: &wire}).Start(&acptest.Client{})
	if err != nil {
		t.Fatal(err)
	}
	conn := proc.Conn()

	init, err := conn.Initialize(t.Context(), acp.InitializeRequest{ProtocolVersion: acp.ProtocolVersionNumber})
	if err != nil || init.AgentInfo == nil || init.AgentInfo.Name != "gobble" || init.AgentInfo.Version == "" {
		t.Fatalf("initialize = %+v, %v", init, err)
	}
	if !semver.MatchString(init.AgentInfo.Version) {
		t.Fatalf("agentInfo.version %q is not SemVer", init.AgentInfo.Version)
	}
	sess, err := conn.NewSession(t.Context(), acp.NewSessionRequest{Cwd: t.TempDir(), McpServers: []acp.McpServer{}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = conn.Prompt(t.Context(), acp.PromptRequest{SessionId: sess.SessionId, Prompt: []acp.ContentBlock{acp.TextBlock("ping")}})
	if re, ok := errors.AsType[*acp.RequestError](err); !ok || re.Code != -32000 {
		t.Fatalf("prompt err = %v, want auth_required (-32000) without a credential", err)
	}
	if _, err := conn.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: sess.SessionId}); err != nil {
		t.Fatal(err)
	}

	// Close returns nil only for an exit of status 0, of the agent's own,
	// within the grace of its input closing (0008-MADR D19 item 6).
	if err := proc.Close(); err != nil {
		t.Fatalf("gobble acp did not exit 0 within 2 s of stdin closing: %v; stderr %q", err, stderr.String())
	}
	if s := stderr.String(); s != "" {
		t.Fatalf("gobble acp wrote to stderr: %q", s)
	}
	lines := strings.Split(strings.TrimSuffix(wire.String(), "\n"), "\n")
	if len(lines) < 4 {
		t.Fatalf("stdout carried %d lines, want the four responses", len(lines))
	}
	for i, l := range lines {
		var msg struct {
			JSONRPC string `json:"jsonrpc"`
		}
		if err := json.Unmarshal([]byte(l), &msg); err != nil || msg.JSONRPC != "2.0" {
			t.Fatalf("stdout line %d is not a JSON-RPC message: %q", i+1, l)
		}
	}
}
