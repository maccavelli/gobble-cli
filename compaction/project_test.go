package compaction

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/maccavelli/gobble-cli/session"
)

// piFixtures are written by Pi's own code, with Pi's context of each in
// <name>.context.json: buildSessionContext's messages, and convertToLlm's
// of them (0005-PLAN "F2 made executable").
const piFixtures = "../session/testdata/pi"

// loadFixture reads a fixture as the store does: the header, the entries,
// migrated when the file is older.
func loadFixture(t *testing.T, name string) []session.Entry {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(piFixtures, name+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	ls := bytes.Split(bytes.TrimSuffix(b, []byte("\n")), []byte("\n"))
	var h session.Header
	if err := json.Unmarshal(ls[0], &h); err != nil {
		t.Fatal(err)
	}
	entries := make([]session.Entry, len(ls)-1)
	for i, l := range ls[1:] {
		if err := json.Unmarshal(l, &entries[i]); err != nil {
			t.Fatal(err)
		}
	}
	entries, err = session.Migrate(h.Version, entries)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func normalJSON(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// Each fixture's projection is Pi's context, message for message, and its
// conversion is what Pi sends the model.
func TestProjectMatchesPi(t *testing.T) {
	for _, name := range []string{"v1", "v1-branched", "v2", "v2-branched", "v3", "v3-branched"} {
		b, err := os.ReadFile(filepath.Join(piFixtures, name+".context.json"))
		if err != nil {
			t.Fatal(err)
		}
		var pi struct {
			Messages []jsontext.Value `json:"messages"`
			LLM      []jsontext.Value `json:"llm"`
		}
		if err := json.Unmarshal(b, &pi); err != nil {
			t.Fatal(err)
		}
		var msgs, llm []*session.Message
		for _, p := range Project(Context(session.Path(loadFixture(t, name)))) {
			for _, m := range p.Messages {
				msgs = append(msgs, m)
				if c := Convert(m); c != nil {
					llm = append(llm, c)
				}
			}
		}
		for _, c := range []struct {
			what      string
			got, want any
		}{{"messages", msgs, pi.Messages}, {"converted", llm, pi.LLM}} {
			got, want := normalJSON(t, c.got), normalJSON(t, c.want)
			if !reflect.DeepEqual(got, want) {
				gb, _ := json.Marshal(got)  //nolint:errcheck // decoded JSON encodes
				wb, _ := json.Marshal(want) //nolint:errcheck // as above
				t.Errorf("%s %s differ from Pi's:\n got %s\nwant %s", name, c.what, gb, wb)
			}
		}
	}
}

// Pi's estimate of each new role (compaction.ts estimateTokens).
func TestEstimateNewRoles(t *testing.T) {
	out := "1234"
	for _, c := range []struct {
		m    *session.Message
		want int
	}{
		{&session.Message{Role: session.RoleCustom, Content: jsontext.Value(`"12345678"`)}, 2},
		{&session.Message{Role: session.RoleCustom, Content: jsontext.Value(`[{"type":"image","data":"x","mimeType":"image/png"}]`)}, imageChars / 4},
		{&session.Message{Role: session.RoleBashExecution, Command: "ls", Output: &out}, 2},
		{&session.Message{Role: session.RoleBranchSummary, Summary: "123456789"}, 3},
		{&session.Message{Role: session.RoleCompactionSummary, Summary: "1234"}, 1},
		{&session.Message{Role: session.RoleSystem, Content: jsontext.Value(`"12345678"`)}, 0},
	} {
		if got := Estimate(c.m); got != c.want {
			t.Errorf("Estimate(%s) = %d, want %d", c.m.Role, got, c.want)
		}
	}
}

// A context_edit drops or replaces its target, the last one winning; a
// string replaces an assistant's content with one text block.
func TestProjectEdits(t *testing.T) {
	p := (&path{}).
		add(text(session.RoleUser, "kept")).
		add(text(session.RoleAssistant, "old"))
	a := p.entries[1].ID
	p.add(session.Entry{Type: session.TypeContextEdit, TargetID: a, Replacement: jsontext.Value(`null`)}).
		add(session.Entry{Type: session.TypeContextEdit, TargetID: a, Replacement: jsontext.Value(`{"content":"new"}`)})
	proj := Project(p.entries)
	if len(proj[1].Messages) != 1 || proj[1].Messages[0].Text() != "new" || string(proj[1].Messages[0].Content) != `[{"type":"text","text":"new"}]` {
		t.Fatalf("edited: %+v", proj[1].Messages)
	}
	if p.entries[1].Message.Text() != "old" {
		t.Fatal("the entry itself was changed")
	}
	p.add(session.Entry{Type: session.TypeContextEdit, TargetID: a, Replacement: jsontext.Value(`null`)})
	if proj := Project(p.entries); len(proj[1].Messages) != 0 || len(proj[0].Messages) != 1 {
		t.Fatalf("dropped: %+v", proj)
	}
}

// A custom message is a cut point and starts a turn, as a user message
// does (Pi's isCutPointMessage, isTurnStartMessage).
func TestCutAtNewRoles(t *testing.T) {
	p := (&path{}).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400))).
		add(session.Entry{Type: session.TypeCustomMessage, CustomType: "ext", Content: jsontext.Value(`"` + chars(400) + `"`)}).
		add(text(session.RoleAssistant, chars(400)))
	prep, err := Prepare(p.entries, Settings{KeepRecentTokens: 150})
	if err != nil {
		t.Fatal(err)
	}
	if prep.FirstKeptEntryID != p.entries[2].ID || prep.SplitTurn {
		t.Fatalf("first kept %s, split %v; want the custom message %s, no split", prep.FirstKeptEntryID, prep.SplitTurn, p.entries[2].ID)
	}
}

// A context_edit after the last usage makes it untrusted: the context is
// then the sum of estimates (Pi's estimateProjectedContextTokens).
func TestEstimateAfterEdit(t *testing.T) {
	answer := text(session.RoleAssistant, chars(400))
	answer.Message.Usage = &session.Usage{Input: 4000, Output: 1000, TotalTokens: 5000}
	p := (&path{}).add(text(session.RoleUser, chars(400))).add(answer)
	if got := EstimateEntries(p.entries); got != 5000 {
		t.Fatalf("before the edit %d, want the usage's 5000", got)
	}
	p.add(session.Entry{Type: session.TypeContextEdit, TargetID: p.entries[0].ID, Replacement: jsontext.Value(`{"content":"abcd"}`)})
	if got := EstimateEntries(p.entries); got != 101 {
		t.Fatalf("after the edit %d, want 1 + 100 estimated", got)
	}
}

// omission is a path whose last assistant message a null context_edit
// left out, as a recovery attempt leaves it (Pi's isRecoveryOmissionSuffix).
func omission() *path {
	p := (&path{}).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400))).
		add(text(session.RoleUser, chars(400))).
		add(text(session.RoleAssistant, chars(400)))
	return p.add(session.Entry{Type: session.TypeContextEdit, TargetID: p.entries[3].ID, Replacement: jsontext.Value(`null`)})
}

// The cut moves past an omitted recovery suffix, and not past a suffix that
// holds input still to be sent or that replaces another entry.
func TestOmissionAdvance(t *testing.T) {
	keep := Settings{KeepRecentTokens: 50}
	p := omission()
	prep, err := Prepare(p.entries, keep)
	if err != nil {
		t.Fatal(err)
	}
	if prep.FirstKeptEntryID != p.entries[3].ID || !prep.SplitTurn {
		t.Fatalf("recovery suffix: first kept %s, split %v; want %s, split", prep.FirstKeptEntryID, prep.SplitTurn, p.entries[3].ID)
	}
	unsent := omission().add(text(session.RoleUser, "go"))
	other := omission()
	other.add(session.Entry{Type: session.TypeContextEdit, TargetID: other.entries[0].ID, Replacement: jsontext.Value(`{"content":"` + chars(400) + `"}`)})
	for what, q := range map[string]*path{"unsent input": unsent, "another entry replaced": other} {
		prep, err := Prepare(q.entries, keep)
		if err != nil {
			t.Fatal(err)
		}
		if prep.FirstKeptEntryID != q.entries[2].ID {
			t.Errorf("%s: first kept %s, want %s", what, prep.FirstKeptEntryID, q.entries[2].ID)
		}
	}
}
