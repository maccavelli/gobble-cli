package acpserver

import (
	"encoding/json/v2"
	"fmt"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/acpext"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

var errCut = fmt.Errorf("%w: stream: read: connection reset by peer", llm.ErrUnavailable)

// cutAfter is a call that streams text and is then cut by err.
func cutAfter(text string, err error) llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: text}}, Err: err}
}

// After a cut retried, the live session's next request is the reloaded
// one's: the cut part is a failed entry in both, sent by neither
// (0070-MADR D12 (c), D7; C9).
func TestRetryAfterCutHistoryIsTheFiles(t *testing.T) {
	script := func() *llmtest.Script {
		return llmtest.NewScript(cutAfter("half", errCut), llmtest.Text("whole"), llmtest.Text("after"))
	}
	live := turnRequests(t, false, script(), nil, nil)
	reloaded := turnRequests(t, true, script(), nil, nil)
	sameRequest(t, live, reloaded)
	if sentOnWire(t, live, "half") || !sentOnWire(t, live, "whole") {
		t.Errorf("the next request sends the cut part, or not the retry's reply")
	}
}

// Live, a retried cut shows its part, a _gobble/retry notice of cause cut,
// attempt 1 of 3 with its delay, ended after the wait at the retry's first
// event, then the reply from its start; a load replays the cut part, its
// "Error: …" line, then the retry's reply (0070-MADR D12 (d), Q7; C12).
func TestRetryAfterCutShows(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir, cwd := t.TempDir(), t.TempDir()
		client := newOrderedClient()
		c, _ := sessionAgent(t, dir, llmtest.NewScript(cutAfter("half", errCut), llmtest.Text("whole")), client)
		id := newSession(t, c, cwd, nil).SessionId
		if resp, err := prompt(t, c, id, "go"); err != nil || resp.StopReason != acp.StopReasonEndTurn {
			t.Fatalf("the turn answered %+v, %v", resp, err)
		}
		log := client.entries()
		var order []string
		var start, end time.Time
		for _, l := range log {
			order = append(order, l.what)
			switch l.what {
			case "retry:start:cut":
				start = l.at
			case "retry:end:cut":
				end = l.at
			}
		}
		if got := strings.Join(order, "|"); got != "chunk:half|retry:start:cut|retry:end:cut|chunk:whole" {
			t.Fatalf("live %s, want the cut part, the cut notice's start and end, then the reply", got)
		}
		if d := end.Sub(start); d != 2*time.Second {
			t.Errorf("the cut notice ended %s after it started, want after the 2 s wait", d)
		}
		var first acpext.RetryNotice
		for _, n := range client.Notices() {
			if n.Method == acpext.MethodRetry {
				if err := json.Unmarshal(n.Params, &first); err != nil {
					t.Fatal(err)
				}
				break
			}
		}
		if first.Phase != "start" || first.Cause != "cut" || first.Attempt != 1 || first.MaxAttempts != 3 || first.DelayMs != 2000 {
			t.Errorf("the cut notice %+v, want cause cut, attempt 1 of 3, 2,000 ms", first)
		}
		if _, err := c.Client.CloseSession(t.Context(), acp.CloseSessionRequest{SessionId: id}); err != nil {
			t.Fatal(err)
		}
		loaded := &acptest.Client{}
		c2, _ := sessionAgent(t, dir, llmtest.NewScript(), loaded)
		if _, err := c2.Client.LoadSession(t.Context(), acp.LoadSessionRequest{SessionId: id, Cwd: cwd, McpServers: []acp.McpServer{}}); err != nil {
			t.Fatal(err)
		}
		var chunks []string
		for _, n := range loaded.Updates() {
			if u := n.Update.AgentMessageChunk; u != nil {
				chunks = append(chunks, u.Content.Text.Text)
			}
		}
		want := []string{"half", "\n\nError: " + errCut.Error(), "whole"}
		if strings.Join(chunks, "|") != strings.Join(want, "|") {
			t.Errorf("load replays %q, want %q", chunks, want)
		}
	})
}

// One entry per call: three retried cuts that showed something are three
// failed entries, and the fourth cut, which ends the turn, one more
// (0070-MADR D12 (h); C22).
func TestRetryAfterCutWritesOneEntryPerCall(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		dir := t.TempDir()
		c, _ := sessionAgent(t, dir, llmtest.NewScript(cutAfter("a", errCut), cutAfter("b", errCut), cutAfter("c", errCut), cutAfter("d", errCut)),
			&acptest.Client{})
		id := newSession(t, c, t.TempDir(), nil).SessionId
		if _, err := prompt(t, c, id, "go"); err == nil {
			t.Fatal("the fourth cut did not end the turn")
		}
		var texts []string
		for _, e := range savedEntries(t, dir, string(id)) {
			if m := e.Message; m != nil && m.StopReason == "error" {
				blocks, _ := m.Blocks()
				texts = append(texts, blocks[0].Text)
			}
		}
		if strings.Join(texts, ",") != "a,b,c,d" {
			t.Fatalf("failed entries %q, want one for each call: a, b, c, d", texts)
		}
	})
	synctest.Test(t, func(t *testing.T) {
		// A cut retry's wait aborted ends the turn on the cut call, already
		// its own entry: it is not written again.
		dir := t.TempDir()
		client := newOrderedClient()
		c, _ := sessionAgent(t, dir, llmtest.NewScript(cutAfter("half", errCut), llmtest.Text("never")), client)
		id := newSession(t, c, t.TempDir(), nil).SessionId
		done := startPrompt(t, c, id, "go")
		client.waitNotice(t, "start")
		mustExt(t, c, "_gobble/abort_retry", map[string]any{"sessionId": string(id)})
		if r := <-done; r.err == nil {
			t.Fatal("the aborted retry did not end the turn")
		}
		n := 0
		for _, e := range savedEntries(t, dir, string(id)) {
			if m := e.Message; m != nil && m.StopReason == "error" {
				n++
			}
		}
		if n != 1 {
			t.Fatalf("%d failed entries, want the cut call's one", n)
		}
	})
}

// A title is not a turn: a cut title call is not sent again (0070-MADR
// D12 (g); C21).
func TestTitleCutNotRetried(t *testing.T) {
	small := llmtest.NewScript(cutAfter("A ti", errCut), llmtest.Text("A title"))
	h := newTitleHarness(t, t.TempDir(), llmtest.NewScript(llmtest.Text("ok")), small)
	id := newSession(t, h.c, t.TempDir(), nil).SessionId
	if _, err := prompt(t, h.c, id, "Name this session"); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return strings.Contains(h.logs.String(), "title: the session was not named") })
	if n := len(small.Requests()); n != 1 {
		t.Fatalf("%d title requests, want 1", n)
	}
}
