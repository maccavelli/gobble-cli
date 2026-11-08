package cli

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
)

// region is s from the first want's start to the end of the last after it:
// the reply a turn drew, without the prompt around it.
func region(t *testing.T, s, first, last string) string {
	t.Helper()
	i := strings.Index(s, first)
	if i < 0 {
		t.Fatalf("%q not drawn in %q", first, s)
	}
	j := strings.Index(s[i:], last)
	if j < 0 {
		t.Fatalf("%q not drawn after %q in %q", last, first, s[i:])
	}
	return s[i : i+j+len(last)]
}

// Line mode offers session.compaction. It draws a manual compaction's rows
// as /compact's reply, byte for byte as /compact's agent text drew it, and
// an automatic one's as today's lines on stderr (0016-MADR D10; 0016-PLAN
// deviation 2).
func TestLineCompactionRows(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{
		{Updates: []string{
			`{"sessionUpdate":"compaction_update","compactionId":"c1","status":"in_progress","_meta":{"gobble":{"reason":"manual"}}}`,
			`{"sessionUpdate":"compaction_update","compactionId":"c1","status":"completed","summary":[{"type":"text","text":"## Goal\nSUMMARY-ONE"}],"_meta":{"gobble":{"reason":"manual","tokensBefore":1234}}}`,
		}},
		{Updates: []string{chunk("Compacting the conversation…\n\n"), chunk("Compacted from 1234 tokens.\n\n## Goal\nSUMMARY-TWO")}},
		{Updates: []string{
			`{"sessionUpdate":"compaction_update","compactionId":"c2","status":"in_progress","_meta":{"gobble":{"reason":"manual"}}}`,
			`{"sessionUpdate":"compaction_update","compactionId":"c2","status":"failed","error":"boom-one","_meta":{"gobble":{"reason":"manual"}}}`,
		}},
		{Updates: []string{chunk("Compaction failed: boom-two")}},
		{Updates: []string{
			`{"sessionUpdate":"compaction_update","compactionId":"c3","status":"in_progress","_meta":{"gobble":{"reason":"threshold"}}}`,
			`{"sessionUpdate":"compaction_update","compactionId":"c3","status":"failed","error":"offline","_meta":{"gobble":{"reason":"threshold","paused":"turn"}}}`,
			chunk("auto-done\n"),
		}},
	}}
	h := newLineHarness(t, agent)
	h.start(Prompt{})
	for _, marker := range []string{"SUMMARY-ONE", "SUMMARY-TWO", "boom-one", "boom-two", "auto-done"} {
		h.press("go\r")
		waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), marker) })
	}
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	caps := agent.ClientCapabilities()
	if len(caps) != 1 || caps[0].Session == nil || caps[0].Session.Compaction == nil || caps[0].Session.Notices != nil {
		t.Fatalf("client capabilities %+v, want session.compaction alone", caps)
	}
	out := h.stdout.String()
	rows := region(t, out, "Compacting the conversation…", "SUMMARY-ONE")
	text := region(t, out[strings.Index(out, "SUMMARY-ONE"):], "Compacting the conversation…", "SUMMARY-TWO")
	if strings.TrimSuffix(rows, "ONE") != strings.TrimSuffix(text, "TWO") {
		t.Fatalf("the rows drew %q; /compact's text drew %q", rows, text)
	}
	failedRows := region(t, out, "Compaction failed: ", "boom-one")
	failedText := region(t, out, "Compaction failed: boom-two", "boom-two")
	if strings.TrimSuffix(failedRows, "one") != strings.TrimSuffix(failedText, "two") {
		t.Fatalf("the failed row drew %q; /compact's text drew %q", failedRows, failedText)
	}
	errs := h.stderr.String()
	if strings.Contains(errs, "(manual)") || !strings.Contains(errs, "compacting the conversation (threshold)") ||
		!strings.Contains(errs, "compaction failed: offline (automatic compaction paused for this turn)") {
		t.Fatalf("stderr %q, want the automatic compaction's lines alone", errs)
	}
}

// /compact in line mode, against gobble's own agent, which sends the rows
// to a client that offered session.compaction, still draws its progress,
// its count and its summary, once each, in the session's voice (0016-PLAN
// deviation 2).
func TestLineCompactThroughTheAgent(t *testing.T) {
	model := llmtest.NewScript(llmtest.Text("r1"), llmtest.Text("r2"), llmtest.Text("THE-SUMMARY"))
	h := newModelHarness(t, model, t.TempDir())
	writeFile(t, filepath.Join(os.Getenv("GOBBLE_HOME"), "settings.yaml"), "compaction:\n  keepRecentTokens: 10\n")
	h.start(Prompt{})
	h.press("one " + strings.Repeat("x", 400) + "\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "r1") })
	h.press("two " + strings.Repeat("x", 400) + "\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "r2") })
	h.press("/compact\r")
	waitUntil(t, func() bool { return strings.Contains(h.stdout.String(), "THE-SUMMARY") })
	h.press("\x04")
	if err := h.wait(); err != nil {
		t.Fatal(err)
	}
	out := h.stdout.String()
	if strings.Count(out, "Compacting the conversation") != 1 || strings.Count(out, "THE-SUMMARY") != 1 ||
		!regexp.MustCompile(`Compacted from [0-9]+ tokens\.`).MatchString(out) {
		t.Fatalf("line mode's /compact drew %q", out)
	}
}

// Print mode offers neither capability: its result is the turn's agent
// text, and its stream-json lines are the session/update params as they
// were (0016-MADR D10).
func TestPrintOffersNoCompaction(t *testing.T) {
	agent := &acptest.ScriptAgent{Turns: []acptest.Turn{{Updates: []string{chunk("ok")}}}}
	withAgent(t, agent)
	if r := runMain(t, nil, "-p", "hi"); r.code != ExitOK {
		t.Fatalf("exit %d, stderr %q", r.code, r.stderr)
	}
	if caps := agent.ClientCapabilities(); len(caps) != 1 || caps[0].Session != nil {
		t.Fatalf("print mode's client capabilities %+v, want none", caps)
	}
}
