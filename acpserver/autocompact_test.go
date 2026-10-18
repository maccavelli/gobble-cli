package acpserver

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	acp "github.com/maccavelli/acp-go-sdk2"

	"github.com/maccavelli/gobble-cli/acpclient/acptest"
	"github.com/maccavelli/gobble-cli/compaction"
	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session/jsonl"
)

// compactHarness is a connected agent over a script, whose models m1 and m2
// have windows of 120,000 and 60,000 tokens and output limits of 4,096
// and 2,048; each session takes settings; the models offered can be
// changed; SaveSetting records each write.
type compactHarness struct {
	t      *testing.T
	c      *acptest.Conn
	client *acptest.Client
	ag     *Agent
	script *llmtest.Script
	cwd    string

	mu      sync.Mutex
	offered []string
	saved   []string
}

var compactModels = map[string]llm.Model{
	"m1": {ID: "m1", ContextWindow: 120000, MaxTokens: 4096},
	"m2": {ID: "m2", ContextWindow: 60000, MaxTokens: 2048},
}

func newCompactHarness(t *testing.T, settings SessionSettings, turns ...llmtest.Turn) *compactHarness {
	t.Helper()
	h := &compactHarness{t: t, client: &acptest.Client{}, script: llmtest.NewScript(turns...), cwd: t.TempDir(), offered: []string{"m1", "m2"}}
	if settings.Compaction.KeepRecentTokens == 0 {
		settings.Compaction.KeepRecentTokens = 50
	}
	h.ag = New(Options{Version: "1.2.3", Store: jsonl.New(t.TempDir(), jsonl.Options{}),
		Provider: func(string, llm.Retry) (llm.Provider, error) { return h.script, nil },
		Models: func() ModelChoice {
			h.mu.Lock()
			defer h.mu.Unlock()
			return ModelChoice{Provider: "fake", Models: slices.Clone(h.offered), Default: "m1"}
		},
		ModelInfo: func(provider, model string) llm.Model {
			m := compactModels[model]
			m.Provider = provider
			return m
		},
		Settings: func(string) SessionSettings { return settings },
		SaveSetting: func(path, value string) error {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.saved = append(h.saved, path+"="+value)
			return nil
		}})
	h.c = acptest.Connect(t, h.ag, h.client)
	h.ag.SetConnection(h.c.Agent)
	t.Cleanup(func() {
		if err := h.ag.Close(); err != nil {
			t.Error(err)
		}
	})
	initialize(t, h.c.Client)
	return h
}

func (h *compactHarness) session() acp.SessionId {
	return newSession(h.t, h.c, h.cwd, nil).SessionId
}

func (h *compactHarness) prompt(id acp.SessionId, text string) error {
	h.t.Helper()
	_, err := prompt(h.t, h.c, id, text)
	return err
}

func (h *compactHarness) mustPrompt(id acp.SessionId, text string) {
	h.t.Helper()
	if err := h.prompt(id, text); err != nil {
		h.t.Fatalf("prompt %.20q: %v", text, err)
	}
}

// live is the session's state, for the pause, which no method reads.
func (h *compactHarness) live(id acp.SessionId) (compactPause, bool) {
	h.ag.mu.Lock()
	defer h.ag.mu.Unlock()
	s := h.ag.sessions[id]
	return s.pause, s.autoCompactOff
}

// notices are the _gobble/compaction notifications received, decoded.
func (h *compactHarness) notices() []map[string]any {
	h.t.Helper()
	var out []map[string]any
	for _, n := range h.client.Notices() {
		if n.Method != methodCompaction {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(n.Params, &m); err != nil {
			h.t.Fatal(err)
		}
		delete(m, "sessionId")
		out = append(out, m)
	}
	return out
}

// summaries are the summary requests made: those whose system prompt is
// the summariser's.
func (h *compactHarness) summaries() []*llm.Request {
	var out []*llm.Request
	for _, r := range h.script.Requests() {
		if strings.HasPrefix(systemOf(r), "You are a context summarization assistant.") {
			out = append(out, r)
		}
	}
	return out
}

// calls are the requests of the session's turns, without the summaries.
func (h *compactHarness) calls() []*llm.Request {
	var out []*llm.Request
	for _, r := range h.script.Requests() {
		if !strings.HasPrefix(systemOf(r), "You are a context summarization assistant.") {
			out = append(out, r)
		}
	}
	return out
}

// convText is a request's conversation, without its system prompt.
func convText(r *llm.Request) string {
	var b strings.Builder
	for _, m := range r.Messages {
		if m.Role == llm.RoleSystem {
			continue
		}
		for _, c := range m.Content {
			b.WriteString(c.Text + "\n")
		}
	}
	return b.String()
}

// full is a reply of text whose usage reports the context at used tokens.
func full(text string, used int64) llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.TextDelta{Text: text}, llm.Usage{InputTokens: used}, llm.Done{StopReason: llm.StopEndTurn}}}
}

// fullCall is a read call whose usage reports the context at used tokens.
func fullCall(id string, used int64) llmtest.Turn {
	return llmtest.Turn{Events: []llm.Event{llm.ToolCallStart{ID: id, Name: "read"}, llm.ToolCallDone{ID: id, Name: "read", Args: `{"path":"f.txt"}`},
		llm.Usage{InputTokens: used}, llm.Done{StopReason: llm.StopToolUse}}}
}

var (
	p1 = "one " + strings.Repeat("a", 400)
	p2 = "two " + strings.Repeat("b", 400)
)

func compactNotice(phase, reason string, extra map[string]any) map[string]any {
	m := map[string]any{"phase": phase, "reason": reason}
	maps.Copy(m, extra)
	return m
}

// sameCompactionNotices compares notices, a "tokensBefore" of -1 in want
// matching any count above 0.
func sameCompactionNotices(got, want []map[string]any) bool {
	return slices.EqualFunc(got, want, func(g, w map[string]any) bool {
		if w["tokensBefore"] == -1 {
			n, ok := g["tokensBefore"].(float64)
			if !ok || n <= 0 {
				return false
			}
			g = mapsWithout(g, "tokensBefore")
			w = mapsWithout(w, "tokensBefore")
		}
		gb, _ := json.Marshal(g) //nolint:errcheck // a decoded map always encodes
		wb, _ := json.Marshal(w) //nolint:errcheck // as above
		return string(gb) == string(wb)
	})
}

func mapsWithout(m map[string]any, key string) map[string]any {
	out := map[string]any{}
	for k, v := range m {
		if k != key {
			out[k] = v
		}
	}
	return out
}

// A session over the window less the reserve compacts before the call: the
// entry written, _gobble/compaction's start and end, a usage_update, the
// summary capped at the model's MaxTokens, and the call, and the next
// turn, sent from the compacted context (0005-PLAN F5-2 decisions 1, 3, 7
// and 9).
func TestAutoCompactAtThreshold(t *testing.T) {
	h := newCompactHarness(t, SessionSettings{}, full("r1", 110000), llmtest.Text("SUMMARY"), llmtest.Text("after"), llmtest.Text("third"))
	id := h.session()
	h.mustPrompt(id, p1)
	before := len(h.client.Updates())
	h.mustPrompt(id, p2)
	h.mustPrompt(id, "three")
	if got, want := h.notices(), []map[string]any{compactNotice("start", "threshold", nil), compactNotice("end", "threshold", map[string]any{"tokensBefore": -1})}; !sameCompactionNotices(got, want) {
		t.Fatalf("notices %v, want %v", got, want)
	}
	sums := h.summaries()
	if len(sums) != 1 || sums[0].MaxTokens != 4096 || !strings.Contains(convText(sums[0]), "one aaa") || strings.Contains(convText(sums[0]), "two bbb") {
		t.Fatalf("summaries %d, cap %v; want one, capped at m1's 4096, of the first prompt", len(sums), caps(sums))
	}
	calls := h.calls()
	if len(calls) != 3 {
		t.Fatalf("%d calls, want 3", len(calls))
	}
	for i, r := range calls[1:] {
		got := convText(r)
		if !strings.Contains(got, "compacted into the following summary") || !strings.Contains(got, "SUMMARY") ||
			strings.Contains(got, "one aaa") || strings.Count(got, "two bbb") != 1 {
			t.Fatalf("call %d after the compaction sent %q", i+2, got)
		}
	}
	if got := convText(calls[2]); strings.Count(got, "after") != 1 || !strings.Contains(got, "three") {
		t.Fatalf("the next turn sent %q, want the turn after the compaction once", got)
	}
	if !slices.Contains(entryTypes(t, h.c, id), "compaction") {
		t.Fatalf("entries %v, want a compaction", entryTypes(t, h.c, id))
	}
	usage := 0
	for _, n := range h.client.Updates()[before:] {
		if n.Update.UsageUpdate != nil {
			usage++
		}
	}
	if usage != 3 { // the compaction's, and each turn's end
		t.Fatalf("%d usage_updates in the last two turns, want 3", usage)
	}
}

func caps(reqs []*llm.Request) []int {
	out := make([]int, len(reqs))
	for i, r := range reqs {
		out[i] = r.MaxTokens
	}
	return out
}

// An overflow compacts and resends; with compaction off, it fails the turn
// with no compaction (0005-PLAN F5-2 decision 2).
func TestAutoCompactOverflow(t *testing.T) {
	overflow := llmtest.Turn{Err: llm.ErrContextOverflow}
	h := newCompactHarness(t, SessionSettings{}, full("r1", 10), overflow, llmtest.Text("SUMMARY"), llmtest.Text("ok"))
	id := h.session()
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2)
	if got, want := h.notices(), []map[string]any{compactNotice("start", "overflow", nil), compactNotice("end", "overflow", map[string]any{"tokensBefore": -1})}; !sameCompactionNotices(got, want) {
		t.Fatalf("notices %v, want %v", got, want)
	}
	calls := h.calls()
	if len(h.summaries()) != 1 || len(calls) != 3 || !strings.Contains(convText(calls[2]), "SUMMARY") || strings.Contains(convText(calls[2]), "one aaa") {
		t.Fatalf("%d summaries, %d calls; want one, and the call sent again from the compacted context", len(h.summaries()), len(calls))
	}

	off := newCompactHarness(t, SessionSettings{AutoCompactionOff: true}, full("r1", 10), overflow, llmtest.Text("never"))
	id = off.session()
	off.mustPrompt(id, p1)
	if err := off.prompt(id, p2); err == nil || !strings.Contains(err.Error(), "context overflow") {
		t.Fatalf("compaction off: err %v, want the overflow", err)
	}
	if len(off.notices()) != 0 || len(off.summaries()) != 0 {
		t.Fatalf("compaction off: notices %v, %d summaries; want none", off.notices(), len(off.summaries()))
	}
}

// Compaction off never compacts at the threshold; a session over it that
// Prepare refuses goes ahead with no notice (0005-PLAN F5-2 decisions 5
// and 8).
func TestAutoCompactNotRun(t *testing.T) {
	h := newCompactHarness(t, SessionSettings{AutoCompactionOff: true}, full("r1", 110000), llmtest.Text("ok"))
	id := h.session()
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2)
	if len(h.notices()) != 0 || len(h.summaries()) != 0 {
		t.Fatalf("compaction off: notices %v, %d summaries", h.notices(), len(h.summaries()))
	}

	small := newCompactHarness(t, SessionSettings{}, full("r", 110000), llmtest.Text("ok"))
	id = small.session()
	small.mustPrompt(id, "x")
	small.mustPrompt(id, "y")
	if len(small.notices()) != 0 || len(small.summaries()) != 0 || len(small.calls()) != 2 {
		t.Fatalf("nothing to compact: notices %v, %d summaries, %d calls; want the call alone",
			small.notices(), len(small.summaries()), len(small.calls()))
	}
}

// A turn's fourth compaction ends it (0005-PLAN F5-2 decision 4): each of
// four calls in one turn reports a context over the window, three
// compactions are made, and the fifth call is not.
func TestAutoCompactLimit(t *testing.T) {
	h := newCompactHarness(t, SessionSettings{},
		fullCall("c1", 110000), llmtest.Text("S1"),
		fullCall("c2", 110000), llmtest.Text("S2"),
		fullCall("c3", 110000), llmtest.Text("S3"),
		fullCall("c4", 110000),
		llmtest.Text("never"))
	if err := os.WriteFile(filepath.Join(h.cwd, "f.txt"), []byte(strings.Repeat("f", 2000)), 0o600); err != nil {
		t.Fatal(err)
	}
	id := h.session()
	err := h.prompt(id, p1)
	if code, text := errText(t, err); code != -32603 || text != errCompactionsExhausted.Error() {
		t.Fatalf("prompt: %d %q, want the fourth compaction refused", code, text)
	}
	if len(h.summaries()) != 3 || h.script.Remaining() != 1 {
		t.Fatalf("%d summaries, %d turns left; want 3 and the fifth call not made", len(h.summaries()), h.script.Remaining())
	}
}

// A turn that fails after a compaction leaves the history the file holds:
// the next turn sends its prompt once (0005-PLAN F5-2 decision 3).
func TestAutoCompactFailedTurn(t *testing.T) {
	h := newCompactHarness(t, SessionSettings{}, full("r1", 110000), llmtest.Text("SUMMARY"), llmtest.Turn{Err: errors.New("boom")}, llmtest.Text("ok"))
	id := h.session()
	h.mustPrompt(id, p1)
	if err := h.prompt(id, p2); err == nil {
		t.Fatal("the failing turn succeeded")
	}
	h.mustPrompt(id, "three")
	calls := h.calls()
	if got := convText(calls[len(calls)-1]); strings.Count(got, "two bbb") != 1 || strings.Contains(got, "one aaa") || !strings.Contains(got, "three") {
		t.Fatalf("the turn after the failure sent %q, want the second prompt once", got)
	}
}

// After a compaction within a turn, the turn sends the system prompt the
// compaction rebuilt: a patch made since the baseline is in it (0005-PLAN
// F5-2 deviation 1).
func TestAutoCompactSystemPrompt(t *testing.T) {
	h := newCompactHarness(t, SessionSettings{}, full("r1", 110000), llmtest.Text("SUMMARY"), llmtest.Text("ok"))
	writeFile(t, filepath.Join(h.cwd, "AGENTS.md"), "Old rule.")
	id := h.session()
	h.mustPrompt(id, p1)
	writeFile(t, filepath.Join(h.cwd, "AGENTS.md"), "New rule.")
	h.mustPrompt(id, p2)
	calls := h.calls()
	if sys := systemOf(calls[1]); !strings.Contains(sys, "New rule.") || strings.Contains(sys, "Old rule.") {
		t.Fatalf("the call after the compaction sent the system prompt %q, want the rebuilt one", sys)
	}
	if len(h.summaries()) != 1 {
		t.Fatalf("%d summaries, want 1", len(h.summaries()))
	}
}

// /compact and _gobble/compact send _gobble/compaction with manual: a
// success with its tokens before, a failure with its error (0005-PLAN F5-2
// decision 9).
func TestManualCompactionNotice(t *testing.T) {
	h := newCompactHarness(t, SessionSettings{}, full("r1", 10), full("r2", 10), llmtest.Text("SUMMARY"), full("r3", 10), llmtest.Text("SUMMARY2"))
	id := h.session()
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2)
	h.mustPrompt(id, "/compact")
	if _, err := ext(t, h.c, "_gobble/compact", map[string]any{"sessionId": id}); err == nil {
		t.Fatal("a second compaction in a row succeeded")
	}
	h.mustPrompt(id, "three "+strings.Repeat("c", 400))
	mustExt(t, h.c, "_gobble/compact", map[string]any{"sessionId": id})
	want := []map[string]any{
		compactNotice("start", "manual", nil), compactNotice("end", "manual", map[string]any{"tokensBefore": -1}),
		compactNotice("start", "manual", nil), compactNotice("end", "manual", map[string]any{"error": compaction.ErrAlreadyCompacted.Error()}),
		compactNotice("start", "manual", nil), compactNotice("end", "manual", map[string]any{"tokensBefore": -1}),
	}
	if got := h.notices(); !sameCompactionNotices(got, want) {
		t.Fatalf("notices %v, want %v", got, want)
	}
}

// _gobble/set_auto_compaction writes compaction.enabled and sets the
// session's at once, which get_state reports; enabled must be given
// (0005-PLAN F5-2 decision 10).
func TestSetAutoCompaction(t *testing.T) {
	h := newCompactHarness(t, SessionSettings{}, full("r1", 110000), llmtest.Text("ok"))
	id := h.session()
	if _, err := ext(t, h.c, "_gobble/set_auto_compaction", map[string]any{"sessionId": id}); err == nil {
		t.Fatal("set_auto_compaction with no enabled succeeded")
	} else if code, _ := errText(t, err); code != -32602 {
		t.Fatalf("no enabled: code %d, want -32602", code)
	}
	state := func() any {
		return mustExt(t, h.c, "_gobble/get_state", map[string]any{"sessionId": id})["autoCompactionEnabled"]
	}
	if state() != true {
		t.Fatal("get_state: automatic compaction off at the start")
	}
	mustExt(t, h.c, "_gobble/set_auto_compaction", map[string]any{"sessionId": id, "enabled": false})
	if state() != false {
		t.Fatal("get_state: automatic compaction on after set_auto_compaction false")
	}
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2)
	if len(h.summaries()) != 0 {
		t.Fatalf("off: %d summaries, want none", len(h.summaries()))
	}
	mustExt(t, h.c, "_gobble/set_auto_compaction", map[string]any{"sessionId": id, "enabled": true})
	if state() != true {
		t.Fatal("get_state: automatic compaction off after set_auto_compaction true")
	}
	if want := []string{"compaction.enabled=false", "compaction.enabled=true"}; !slices.Equal(h.saved, want) {
		t.Fatalf("saved %q, want %q", h.saved, want)
	}
}

// A threshold compaction that fails offline pauses the trigger for the
// rest of its turn alone, and says so (0005-PLAN F5-2 decision 11).
func TestAutoCompactPauseTurn(t *testing.T) {
	errOffline := llmtest.Turn{Err: llm.ErrOffline}
	h := newCompactHarness(t, SessionSettings{},
		full("r1", 110000),
		errOffline, fullCall("c1", 110000), full("ok", 110000), // the second turn: no second compaction
		llmtest.Text("SUMMARY"), llmtest.Text("third"))
	if err := os.WriteFile(filepath.Join(h.cwd, "f.txt"), []byte("f"), 0o600); err != nil {
		t.Fatal(err)
	}
	id := h.session()
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2)
	if len(h.summaries()) != 1 {
		t.Fatalf("%d summaries in the failing turn, want 1", len(h.summaries()))
	}
	if p, _ := h.live(id); p != pauseTurn {
		t.Fatalf("pause %q, want the turn's", p)
	}
	h.mustPrompt(id, "three")
	if len(h.summaries()) != 2 {
		t.Fatalf("%d summaries after the next turn, want it to compact", len(h.summaries()))
	}
	want := []map[string]any{
		compactNotice("start", "threshold", nil), compactNotice("end", "threshold", map[string]any{"error": "Summarization failed: llm: offline", "paused": "turn"}),
		compactNotice("start", "threshold", nil), compactNotice("end", "threshold", map[string]any{"tokensBefore": -1}),
	}
	if got := h.notices(); !sameCompactionNotices(got, want) {
		t.Fatalf("notices %v, want %v", got, want)
	}
}

// A compaction whose summary request overflows pauses the trigger for the
// session: the next overflow ends its turn with the paused error, and no
// summary is asked for. /compact, a model change and set_auto_compaction
// each end the pause (0005-PLAN F5-2 decision 11).
func TestAutoCompactPauseSession(t *testing.T) {
	overflow := llmtest.Turn{Err: llm.ErrContextOverflow}
	h := newCompactHarness(t, SessionSettings{}, full("r1", 110000), overflow, full("r2", 110000), overflow)
	id := h.session()
	h.mustPrompt(id, p1)
	h.mustPrompt(id, p2) // the compaction fails, and the call goes ahead
	if p, _ := h.live(id); p != pauseSession {
		t.Fatalf("pause %q, want the session's", p)
	}
	if got := h.notices(); len(got) != 2 || got[1]["paused"] != "session" {
		t.Fatalf("notices %v, want the end paused for the session", got)
	}
	err := h.prompt(id, "three")
	_, text := errText(t, err)
	if !strings.HasPrefix(text, "the context is over the model's window, and automatic compaction is paused after it failed: Summarization failed: llm: context overflow") ||
		len(h.summaries()) != 1 {
		t.Fatalf("paused overflow: %q, %d summaries; want the paused error and no summary asked", text, len(h.summaries()))
	}

	for _, c := range []struct {
		name string
		end  func(acp.SessionId)
	}{
		{"set_auto_compaction", func(id acp.SessionId) {
			mustExt(t, h.c, "_gobble/set_auto_compaction", map[string]any{"sessionId": id, "enabled": true})
		}},
		{"a model change", func(id acp.SessionId) {
			if _, err := setOption(t, h.c, id, configModel, "m2"); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		h.ag.mu.Lock()
		h.ag.sessions[id].pause = pauseSession
		h.ag.mu.Unlock()
		c.end(id)
		if p, _ := h.live(id); p != pauseNone {
			t.Errorf("%s: pause %q, want none", c.name, p)
		}
	}

	m := newCompactHarness(t, SessionSettings{}, full("r1", 10), full("r2", 10), llmtest.Text("SUMMARY"))
	id = m.session()
	m.mustPrompt(id, p1)
	m.mustPrompt(id, p2)
	m.ag.mu.Lock()
	m.ag.sessions[id].pause = pauseSession
	m.ag.mu.Unlock()
	m.mustPrompt(id, "/compact")
	if p, _ := m.live(id); p != pauseNone {
		t.Errorf("/compact: pause %q, want none", p)
	}
}

// After a model switch, the previous model writes the summary, with its
// own output limit; when it is no longer offered, the session's does
// (0005-PLAN F5-2 decision 12).
func TestAutoCompactPreviousModel(t *testing.T) {
	for _, c := range []struct {
		name    string
		offered []string
		model   string
		cap     int
	}{
		{"offered", []string{"m1", "m2"}, "m1", 4096},
		{"no longer offered", []string{"m2"}, "m2", 2048},
	} {
		h := newCompactHarness(t, SessionSettings{}, full("r1", 55000), llmtest.Text("SUMMARY"), llmtest.Text("ok"))
		id := h.session()
		h.mustPrompt(id, p1) // m1: 55,000 of 120,000
		if _, err := setOption(t, h.c, id, configModel, "m2"); err != nil {
			t.Fatal(err)
		}
		h.mu.Lock()
		h.offered = c.offered
		h.mu.Unlock()
		h.mustPrompt(id, p2) // m2: over 60,000 less the reserve
		sums := h.summaries()
		if len(sums) != 1 || sums[0].Model != c.model || sums[0].MaxTokens != c.cap {
			t.Errorf("%s: summaries %d, model %q, cap %v; want %s's, capped at %d", c.name, len(sums), modelOfReqs(sums), caps(sums), c.model, c.cap)
		}
		if calls := h.calls(); calls[len(calls)-1].Model != "m2" {
			t.Errorf("%s: the turn's call went to %q, want m2", c.name, calls[len(calls)-1].Model)
		}
	}
}

func modelOfReqs(reqs []*llm.Request) []string {
	out := make([]string, len(reqs))
	for i, r := range reqs {
		out[i] = r.Model
	}
	return out
}
