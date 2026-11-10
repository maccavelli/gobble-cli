package acpserver

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/llm"
)

// A turn or a listing refused for want of an Anthropic workspace says how
// to name one, and never the service's own message (0035-MADR D5).
func TestWorkspaceWords(t *testing.T) {
	err := fmt.Errorf("%w: service said secret-ish things", llm.ErrWorkspace)
	re, ok := errors.AsType[*acp.RequestError](promptError("claude", err))
	if !ok {
		t.Fatalf("promptError: %v", promptError("claude", err))
	}
	got := fmt.Sprint(re.Data)
	if !strings.Contains(got, workspaceAdvice) || strings.Contains(got, "secret-ish") {
		t.Fatalf("data %s, want only the advice", got)
	}
	if state, reason := StateOf("claude", err); state != StateNotListed || !strings.Contains(reason, "--workspace ID") {
		t.Fatalf("StateOf: %v %q", state, reason)
	}
}
