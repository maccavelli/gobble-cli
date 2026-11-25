package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/jsonx"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// Headless output holds what streamed when a cut is retried: the cut part,
// then the retry's whole reply, and JSON carries no error when the retry
// completes; the cut notice is a retry line on stderr (0070-MADR D12 (i);
// C24).
func TestPrintAfterCutRetry(t *testing.T) {
	cut := fmt.Errorf("%w: stream: read: connection reset by peer", llm.ErrUnavailable)
	for _, format := range []string{"text", "json"} {
		t.Run(format, func(t *testing.T) {
			withModel(t, llmtest.NewScript(llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: "half "}}, Err: cut}, llmtest.Text("whole")))
			r := runMain(t, nil, "-p", "--output-format", format, "x")
			if r.code != ExitOK || !strings.Contains(r.stderr, "cut") {
				t.Fatalf("exit %d, stderr %q; want 0 and the cut's retry line", r.code, r.stderr)
			}
			if format == "text" {
				if r.stdout != "half whole\n" {
					t.Fatalf("stdout %q, want the cut part then the whole reply", r.stdout)
				}
				return
			}
			var res printResult
			if err := jsonx.Unmarshal([]byte(strings.TrimSpace(r.stdout)), &res); err != nil {
				t.Fatalf("stdout %q: %v", r.stdout, err)
			}
			if res.Text != "half whole" || res.Error != "" || res.StopReason != "end_turn" {
				t.Fatalf("result %+v, want the cut part and the whole reply, no error", res)
			}
		})
	}
}
