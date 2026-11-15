package acpserver

import (
	"fmt"
	"testing"
	"testing/synctest"
	"time"

	"github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/agent"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/llm/provider"
)

// The agent's own retry causes are acpext's, which are the auth contract's
// words: agent may not import acpext, so its literals are held equal here,
// and a turn that waits for an empty reply sends _gobble/retry with the cause
// empty_answer, as a provider's own retry sends the failure's class,
// rate_limited (0051-MADR D6; 0051-PLAN P4 step 4b).
func TestRetryCausesAreAcpexts(t *testing.T) {
	if string(agent.RetryOffline) != string(acpext.RetryOffline) || string(agent.RetryEmptyReply) != string(acpext.RetryEmptyReply) {
		t.Errorf("agent causes %q and %q, acpext's %q and %q", agent.RetryOffline, agent.RetryEmptyReply, acpext.RetryOffline, acpext.RetryEmptyReply)
	}
	synctest.Test(t, func(t *testing.T) {
		// the class a provider's retry carries is provider.ErrorKind's word for the failure
		pr := llm.ProviderRetry{Attempt: 1, MaxAttempts: 4, Wait: 1100 * time.Millisecond,
			Kind: provider.ErrorKind(fmt.Errorf("%w: slow down", llm.ErrRateLimited))}
		p := prober{llmtest.NewScript(reasoningOnly(), llmtest.Text("ok"),
			llmtest.Turn{Events: []llm.Event{pr, llm.TextDelta{Text: "ok"}, llm.Done{StopReason: llm.StopEndTurn}}})}
		r := newRetryAgent(t, p, nil)
		id := newSession(t, r.c, t.TempDir(), nil).SessionId
		for _, text := range []string{"one", "two"} {
			if resp, err := prompt(t, r.c, id, text); err != nil || resp.StopReason != acp.StopReasonEndTurn {
				t.Fatalf("%s: %+v, %v", text, resp, err)
			}
		}
		var causes []string
		for _, n := range r.notices(t) {
			if n["phase"] == "start" {
				cause, _ := n["cause"].(string)
				causes = append(causes, cause)
			}
		}
		if len(causes) != 2 || causes[0] != string(acpext.RetryEmptyReply) || causes[1] != string(acpext.RetryRateLimited) {
			t.Errorf("retry causes %q, want %q and %q", causes, acpext.RetryEmptyReply, acpext.RetryRateLimited)
		}
	})
}
