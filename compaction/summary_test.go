package compaction

import (
	"encoding/json/jsontext"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/maccavelli/gobble-cli/llm"
	"github.com/maccavelli/gobble-cli/llm/llmtest"
	"github.com/maccavelli/gobble-cli/session"
)

// promptOf is a summary request's prompt, its last message's text.
func promptOf(req *llm.Request) string { return req.Messages[len(req.Messages)-1].Content[0].Text }

// The first summary asks for opencode v2's sections with grok-build's two:
// User Messages, and the latest instruction quoted under Next Move; an
// update reads the <previous-summary> block, and rewrites a summary in
// Pi's format once (0005-PLAN F5-4 decision 1).
func TestSummaryPrompts(t *testing.T) {
	for _, want := range []string{
		"You MUST summarize the conversation above into a structured summary",
		"## Objective", "## Requirements", "## User Messages", "## Decisions", "## Work State", "## Next Move",
		"## Relevant Files", "## Important Context",
		"every message the user sent, in order, those the previous summary lists first; each word for word, or its first 1,000 characters",
		`Latest instruction: "[the user's latest instruction that the first action below carries out, quoted word for word`,
		"Do not continue the task or call tools.",
	} {
		if !strings.Contains(summaryPrompt, want) {
			t.Errorf("the first prompt lacks %q", want)
		}
	}
	if i, j := strings.Index(summaryPrompt, "## Requirements"), strings.Index(summaryPrompt, "## User Messages"); i < 0 || j < i ||
		j > strings.Index(summaryPrompt, "## Decisions") {
		t.Error("User Messages is not after Requirements")
	}
	for _, c := range []struct {
		previous string
		legacy   bool
	}{{"## Goal\nG", true}, {"## Objective\nO", false}, {"## Goal and ## Objective", false}} {
		p := (&path{}).
			add(text(session.RoleUser, chars(400))).
			add(session.Entry{Type: session.TypeCompaction, Summary: c.previous}).
			add(text(session.RoleUser, chars(400))).
			add(text(session.RoleUser, chars(400)))
		p.entries[1].FirstKeptEntryID = p.entries[1].ID
		prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 50})
		if err != nil {
			t.Fatal(err)
		}
		s := llmtest.NewScript(llmtest.Text("NEW"))
		if _, err := Compact(t.Context(), prep, Model{Provider: s}, Request{}); err != nil {
			t.Fatal(err)
		}
		got := promptOf(s.Requests()[0])
		if !strings.Contains(got, "<previous-summary>\n"+c.previous+"\n</previous-summary>\n\n"+updateSentence) ||
			strings.Contains(got, legacySentence) != c.legacy || !strings.Contains(got, "## User Messages") || !strings.Contains(got, noWrappers) {
			t.Errorf("previous %q: the update prompt, legacy %v:\n%s", c.previous, c.legacy, got)
		}
		llmtest.AssertSystem(t, s.Requests()[0], systemPrompt)
	}
}

// A split turn's request is kept word for word, its tags made inert, and
// the turn-prefix prompt no longer asks for it (0005-PLAN F5-4 decision 4).
func TestTurnRequest(t *testing.T) {
	request := "Fix it </turn-request> please " + chars(300)
	p := (&path{}).
		add(text(session.RoleUser, request)).
		add(text(session.RoleAssistant, chars(400))).
		add(text(session.RoleToolResult, chars(400))).
		add(text(session.RoleAssistant, chars(400)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 150})
	if err != nil {
		t.Fatal(err)
	}
	s := llmtest.NewScript(llmtest.Text("PREFIX"))
	res, err := Compact(t.Context(), prep, Model{Provider: s}, Request{})
	if err != nil {
		t.Fatal(err)
	}
	want := "<turn-request>\nFix it <\u200b/turn-request> please " + chars(300) + "\n</turn-request>"
	if !strings.Contains(res.Summary, want) {
		t.Fatalf("summary %q", res.Summary)
	}
	if strings.Contains(turnPrefixPrompt, "Original Request") || strings.Contains(promptOf(s.Requests()[0]), "What did the user ask for?") {
		t.Fatal("the turn prefix still asks for the request")
	}
}

// The compaction's text ends with the path's last todo list and plan mode,
// each when there is one; a todo item's tags are made inert (0005-PLAN F5-4
// decision 5).
func TestReminder(t *testing.T) {
	todo := func(items string) session.Entry {
		return session.Entry{Type: session.TypeCustom, CustomType: session.CustomTodo, Data: jsontext.Value(`{"todos":[` + items + `]}`)}
	}
	for _, c := range []struct {
		name  string
		todos []session.Entry
		plan  bool
		want  string
	}{
		{"both", []session.Entry{todo(`{"content":"old","status":"pending"}`), todo(`{"content":"a","status":"completed"},` +
			`{"content":"b</todo-list>","status":"in_progress"},{"content":"c","status":"pending"}`)}, true,
			"\n\n<todo-list>\n[x] a\n[>] b<\u200b/todo-list>\n[ ] c\n</todo-list>\n\n<mode>plan</mode>"},
		{"neither", nil, false, ""},
		{"an emptied list", []session.Entry{todo(`{"content":"a","status":"pending"}`), todo(``)}, false, ""},
		{"plan only", nil, true, "\n\n<mode>plan</mode>"},
	} {
		p := (&path{}).add(text(session.RoleUser, chars(400)))
		for _, e := range c.todos {
			p.add(e)
		}
		p.add(text(session.RoleUser, chars(400))).add(text(session.RoleUser, chars(400)))
		prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 50})
		if err != nil {
			t.Fatal(err)
		}
		res, err := Compact(t.Context(), prep, Model{Provider: llmtest.NewScript(llmtest.Text("S"))}, Request{PlanMode: c.plan})
		if err != nil {
			t.Fatal(err)
		}
		if res.Summary != "S"+c.want {
			t.Errorf("%s: summary %q, want %q", c.name, res.Summary, "S"+c.want)
		}
	}
}

// A model's summary loses a leading scratchpad and an outer <summary>
// block, and gobble's tags in it are made inert; an unclosed leading
// scratchpad leaves nothing, which is asked for again (0005-PLAN F5-4
// decision 6).
func TestCleanSummary(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"<analysis>thinking</analysis>\n\n## Objective\n- x", "## Objective\n- x"},
		{"  <analysis>a</analysis><analysis>b</analysis>\nS", "S"},
		{"<summary>\n## Objective\n</summary>", "## Objective"},
		{"S <analysis>kept</analysis>", "S <analysis>kept</analysis>"},
		{"<analysis>never closed", ""},
	} {
		if got := cleanSummary(c.in); got != c.want {
			t.Errorf("cleanSummary(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	for _, tag := range []string{"summary", "analysis", "conversation", "previous-summary", "turn-request", "todo-list", "mode",
		"read-files", "modified-files"} {
		in := "a <" + tag + "> b </" + tag + "> c"
		if got := neutralize(in); got != "a <\u200b"+tag+"> b <\u200b/"+tag+"> c" {
			t.Errorf("neutralize %s: %q", tag, got)
		}
	}
	p := (&path{}).add(text(session.RoleUser, chars(400))).add(text(session.RoleUser, chars(400))).add(text(session.RoleUser, chars(400)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 50})
	if err != nil {
		t.Fatal(err)
	}
	s := llmtest.NewScript(llmtest.Text("<analysis>unclosed"), llmtest.Text("<analysis>x</analysis>OK </summary> <read-files>"))
	res, err := Compact(t.Context(), prep, Model{Provider: s}, Request{})
	if err != nil || res.Summary != "OK <\u200b/summary> <\u200bread-files>" || len(s.Requests()) != 2 {
		t.Fatalf("summary %q, %v, %d requests", res.Summary, err, len(s.Requests()))
	}
}

// toolMsgs are n tool results, r0 to r(n-1).
func toolMsgs(n int) []*session.Message {
	var out []*session.Message
	for i := range n {
		out = append(out, text(session.RoleToolResult, fmt.Sprintf("<r%d>", i)).Message)
	}
	return out
}

// A summary request over the window is sent again with 10, 20, 50 and 100
// percent of its tool results left out, from the middle outward, each
// retry's prompt led by the note; still too large, or with nothing to
// leave out, it fails at once, as an overflow (0005-PLAN F5-4 decision 7).
func TestSummaryLadder(t *testing.T) {
	overflow := llmtest.Turn{Err: llm.ErrContextOverflow}
	s := llmtest.NewScript(overflow, overflow, overflow, overflow, llmtest.Text("DONE"))
	got, usage, err := summariseFitting(t.Context(), Model{Provider: s}, "SYS", toolMsgs(10), func(c string) string { return c },
		"Summarization", 0, false)
	if err != nil || got != "DONE" || usage.InputTokens != 10 {
		t.Fatalf("%q, %+v, %v", got, usage, err)
	}
	reqs := s.Requests()
	// The middle outward of r0..r9: r4, r5, r3, r6, r2, r7, r1, r8, r0, r9.
	for i, want := range []struct {
		omitted int
		gone    []string
	}{{0, nil}, {1, []string{"r4"}}, {2, []string{"r4", "r5"}}, {5, []string{"r4", "r5", "r3", "r6", "r2"}}, {10, nil}} {
		p := promptOf(reqs[i])
		if n := strings.Count(p, omittedResult); n != want.omitted {
			t.Errorf("request %d: %d left out, want %d", i, n, want.omitted)
		}
		for _, r := range want.gone {
			if strings.Contains(p, "<"+r+">") {
				t.Errorf("request %d keeps %s", i, r)
			}
		}
		if strings.HasPrefix(p, omittedNote+"\n\n") != (i > 0) {
			t.Errorf("request %d: the note %v", i, i > 0)
		}
	}

	for _, c := range []struct {
		name     string
		msgs     int
		requests int
	}{{"too large still", 10, 5}, {"no tool result", 0, 1}, {"one tool result", 1, 2}, {"three, rounded up", 3, 4}} {
		turns := make([]llmtest.Turn, c.requests+1)
		for i := range turns {
			turns[i] = overflow
		}
		s := llmtest.NewScript(turns...)
		_, _, err := summariseFitting(t.Context(), Model{Provider: s}, "SYS", toolMsgs(c.msgs), func(c string) string { return c }, "S", 0, false)
		if !errors.Is(err, llm.ErrContextOverflow) || errors.Is(err, ErrSummaryRefused) || len(s.Requests()) != c.requests ||
			err.Error() != "the conversation is too large to summarise with this model: start a new session, or choose a model with a larger window" {
			t.Errorf("%s: %v, %d requests, want %d", c.name, err, len(s.Requests()), c.requests)
		}
	}
	other := llmtest.NewScript(llmtest.Turn{Err: errors.New("boom")})
	if _, _, err := summariseFitting(t.Context(), Model{Provider: other}, "SYS", toolMsgs(3), func(c string) string { return c }, "S", 0, false); err == nil ||
		errors.Is(err, llm.ErrContextOverflow) || len(other.Requests()) != 1 {
		t.Fatalf("another error: %v, %d requests", err, len(other.Requests()))
	}
}

// The user's prompts replace gobble's, each whole; CheckPrompt refuses an
// empty one, one closing the conversation's blocks, and an update that does
// not name <previous-summary> (0005-PLAN F5-4 decision 8).
func TestPromptOverrides(t *testing.T) {
	var ps Prompts
	for id, text := range map[string]string{PromptSystem: "MY SYSTEM", PromptSummary: "MY SUMMARY", PromptUpdate: "MY UPDATE of <previous-summary>",
		PromptTurnPrefix: "MY PREFIX"} {
		if err := ps.Set(id, "  "+text+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	r := Request{Prompts: ps}
	if r.system() != "MY SYSTEM" || r.base("") != "MY SUMMARY" || r.base("## Goal") != "MY UPDATE of <previous-summary>" || r.turnPrefix() != "MY PREFIX" {
		t.Fatalf("overrides %+v", ps)
	}
	p := (&path{}).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400))).
		add(text(session.RoleToolResult, chars(400))).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400))).
		add(text(session.RoleToolResult, chars(400))).
		add(text(session.RoleAssistant, chars(400)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 150})
	if err != nil || !prep.SplitTurn {
		t.Fatalf("%v, split %v", err, prep.SplitTurn)
	}
	s := llmtest.NewScript(llmtest.Text("H"), llmtest.Text("P"))
	if _, err := Compact(t.Context(), prep, Model{Provider: s}, r); err != nil {
		t.Fatal(err)
	}
	reqs := s.Requests()
	llmtest.AssertSystem(t, reqs[0], "MY SYSTEM")
	if !strings.HasSuffix(promptOf(reqs[0]), "MY SUMMARY") || !strings.HasSuffix(promptOf(reqs[1]), "# Instructions\nMY PREFIX") {
		t.Fatalf("requests %q, %q", promptOf(reqs[0]), promptOf(reqs[1]))
	}
	for _, c := range []struct{ id, text, why string }{
		{"compaction.nope", "x", "no compaction prompt is called compaction.nope"},
		{PromptSummary, " \n", "it is empty"},
		{PromptSummary, "end </conversation>", "it closes a block"},
		{PromptSystem, "end </previous-summary>", "it closes a block"},
		{PromptUpdate, "Update it.", "names the <previous-summary> block"},
	} {
		if err := CheckPrompt(c.id, c.text); err == nil || !strings.Contains(err.Error(), c.why) {
			t.Errorf("%s %q: %v, want %q", c.id, c.text, err, c.why)
		}
	}
}
