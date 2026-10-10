package session

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// piFixture reads a fixture of Pi's (testdata/pi, 0005-PLAN "F2 made
// executable"): its header and its entries, migrated as a load migrates
// them, and its lines after the header.
func piFixture(t *testing.T, name string) (Header, []Entry, [][]byte) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "pi", name+".jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	ls := bytes.Split(bytes.TrimSuffix(b, []byte("\n")), []byte("\n"))
	var h Header
	if err := json.Unmarshal(ls[0], &h); err != nil {
		t.Fatal(err)
	}
	entries := make([]Entry, len(ls)-1)
	for i, l := range ls[1:] {
		if err := json.Unmarshal(l, &entries[i]); err != nil {
			t.Fatal(err)
		}
	}
	entries, err = Migrate(h.Version, entries)
	if err != nil {
		t.Fatal(err)
	}
	return h, entries, ls[1:]
}

func ids(es []Entry) string {
	var out []string
	for _, e := range es {
		out = append(out, e.ID)
	}
	return strings.Join(out, ",")
}

// add writes e as the tree's next entry, as a session appends one.
func add(t *Tree, e Entry) Entry {
	e = t.Next(e)
	t.Add(e)
	return e
}

func msg(text string) Entry {
	return Entry{Type: TypeMessage, Message: &Message{Role: RoleUser, Content: []byte(`"` + text + `"`)}}
}

// Branch starts a sibling branch, ResetLeaf a new root, and Path follows
// the leaf; a reload puts the leaf on the last entry in file order,
// whatever moved it before (Pi's branch, resetLeaf, _buildIndex).
func TestTreeLeaf(t *testing.T) {
	tr := NewTree(nil)
	if tr.Leaf() != "" || tr.Path() != nil {
		t.Fatalf("an empty tree has leaf %q", tr.Leaf())
	}
	a := add(tr, msg("a"))
	b := add(tr, msg("b"))
	if a.ParentID != nil || b.ParentID == nil || *b.ParentID != a.ID || ids(tr.Path()) != a.ID+","+b.ID {
		t.Fatalf("a line: %+v %+v, path %s", a, b, ids(tr.Path()))
	}
	if err := tr.Branch(a.ID); err != nil {
		t.Fatal(err)
	}
	if ids(tr.Path()) != a.ID {
		t.Fatalf("after Branch the path is %s", ids(tr.Path()))
	}
	c := add(tr, msg("c"))
	if *c.ParentID != a.ID || ids(tr.Path()) != a.ID+","+c.ID {
		t.Fatalf("the sibling's parent %s, path %s", *c.ParentID, ids(tr.Path()))
	}
	if err := tr.Branch("nope"); err == nil || tr.Leaf() != c.ID {
		t.Fatalf("Branch to an unknown entry: %v, leaf %s", err, tr.Leaf())
	}
	tr.ResetLeaf()
	if tr.Leaf() != "" || tr.Path() != nil {
		t.Fatalf("after ResetLeaf: leaf %q, path %s", tr.Leaf(), ids(tr.Path()))
	}
	d := add(tr, msg("d"))
	if d.ParentID != nil || ids(tr.Path()) != d.ID {
		t.Fatalf("after ResetLeaf the next entry has parent %v", d.ParentID)
	}
	if err := tr.Branch(b.ID); err != nil {
		t.Fatal(err)
	}
	if got := NewTree(tr.Entries()); got.Leaf() != d.ID || ids(got.Path()) != d.ID {
		t.Fatalf("reloaded leaf %s, want the last entry %s", got.Leaf(), d.ID)
	}
	if n := tr.Next(msg("e")); len(tr.Entries()) != 4 || *n.ParentID != b.ID {
		t.Fatalf("Next added the entry, or stamped the wrong parent: %d entries, parent %v", len(tr.Entries()), n.ParentID)
	}
}

// BranchSummary is Pi's branchWithSummary: from the root it has no parent
// and names root; from an entry it is that entry's child and names the leaf
// it leaves; an unknown entry is an error. It becomes the leaf through Add.
func TestBranchSummary(t *testing.T) {
	tr := NewTree(nil)
	a := add(tr, msg("a"))
	b := add(tr, msg("b"))
	s, err := tr.BranchSummary(a.ID, "went off and back", []byte(`{"readFiles":[]}`))
	if err != nil || s.Type != TypeBranchSummary || *s.ParentID != a.ID || s.FromID != b.ID || s.Summary != "went off and back" || s.ID == "" {
		t.Fatalf("summary %+v, %v", s, err)
	}
	if tr.Leaf() != b.ID {
		t.Fatalf("the leaf moved before Add: %s", tr.Leaf())
	}
	tr.Add(s)
	if ids(tr.Path()) != a.ID+","+s.ID {
		t.Fatalf("path %s", ids(tr.Path()))
	}
	tr.ResetLeaf()
	if root, err := tr.BranchSummary("", "from nothing", nil); err != nil || root.ParentID != nil || root.FromID != "root" {
		t.Fatalf("root summary %+v, %v", root, err)
	}
	if _, err := tr.BranchSummary("nope", "x", nil); err == nil {
		t.Fatal("an unknown entry was accepted")
	}
}

// Labels resolve as Pi's index does: the latest label, cleared by an empty
// or absent one, in the order first set since the last clear.
func TestLabels(t *testing.T) {
	label := func(target string, text *string, ts string) Entry {
		return Entry{Type: TypeLabel, TargetID: target, Label: text, Timestamp: ts}
	}
	tr := NewTree(nil)
	for _, e := range []Entry{
		label("x", new("one"), "t1"), label("y", new("why"), "t2"), label("x", new("two"), "t3"),
		label("z", new("zed"), "t4"), label("z", new(""), "t5"),
	} {
		add(tr, e)
	}
	// A label set again keeps its place; an empty one clears it.
	if got, want := tr.Labels(), []Label{{"x", "two", "t3"}, {"y", "why", "t2"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("labels %+v, want %+v", got, want)
	}
	// A label cleared, by an absent one, and set again goes last.
	for _, e := range []Entry{label("x", nil, "t6"), label("x", new("three"), "t7"), label("y", new("again"), "t8")} {
		add(tr, e)
	}
	if got, want := tr.Labels(), []Label{{"y", "again", "t8"}, {"x", "three", "t7"}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("labels %+v, want %+v", got, want)
	}
}

// Branched of v3.jsonl's leaf is Pi's createBranchedSession of it, with the
// new label entries' ids mapped by position (F2-1 deviation 1: Pi's v2
// branching does not re-chain, so only v3 is compared).
func TestBranchedMatchesPi(t *testing.T) {
	_, entries, _ := piFixture(t, "v3")
	tr := NewTree(entries)
	got := tr.Branched(tr.Leaf())
	_, _, pi := piFixture(t, "v3-branched")
	if len(got) != len(pi) {
		t.Fatalf("%d entries, Pi wrote %d", len(got), len(pi))
	}
	mapped := map[string]string{} // Pi's label ids → gobble's
	for i, line := range pi {
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			t.Fatal(err)
		}
		if e.Type == TypeLabel {
			mapped[e.ID] = got[i].ID
		}
	}
	for i, line := range pi {
		var want map[string]any
		if err := json.Unmarshal(line, &want); err != nil {
			t.Fatal(err)
		}
		for _, k := range []string{"id", "parentId"} {
			if s, ok := want[k].(string); ok && mapped[s] != "" {
				want[k] = mapped[s]
			}
		}
		b, err := json.Marshal(got[i])
		if err != nil {
			t.Fatal(err)
		}
		var have map[string]any
		if err := json.Unmarshal(b, &have); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(have, want) {
			t.Errorf("entry %d:\n got %s\nwant %s", i+1, b, line)
		}
	}
	if tr.Branched("nope") != nil {
		t.Fatal("an unknown leaf was branched")
	}
}

// Pi's v2 branching left a kept entry's parent on a removed label: the
// path of v2-branched.jsonl ends there, at its compaction.
func TestV2BranchedPathEnds(t *testing.T) {
	_, entries, _ := piFixture(t, "v2-branched")
	path := Path(entries)
	first := path[0]
	if first.Type != TypeCompaction || first.ParentID == nil {
		t.Fatalf("the path starts at %s %+v, want the compaction whose parent was a label", first.Type, first.ParentID)
	}
	for _, e := range entries {
		if e.ID == *first.ParentID {
			t.Fatalf("its parent %s is in the file", e.ID)
		}
	}
}

// A compaction whose first kept entry was a label keeps the entry after it
// when branched, and one that keeps nothing still names itself.
func TestBranchedMovesFirstKept(t *testing.T) {
	tr := NewTree(nil)
	a := add(tr, msg("a"))
	l := add(tr, Entry{Type: TypeLabel, TargetID: a.ID, Label: new("mark"), Timestamp: "t"})
	b := add(tr, msg("b"))
	c := add(tr, Entry{Type: TypeCompaction, Summary: "s", FirstKeptEntryID: l.ID})
	self := tr.Next(Entry{Type: TypeCompaction, Summary: "s2"})
	self.FirstKeptEntryID = self.ID
	tr.Add(self)
	got := tr.Branched(self.ID)
	if ids(got[:4]) != strings.Join([]string{a.ID, b.ID, c.ID, self.ID}, ",") {
		t.Fatalf("kept %s", ids(got))
	}
	if got[2].FirstKeptEntryID != b.ID || got[3].FirstKeptEntryID != self.ID || *got[1].ParentID != a.ID {
		t.Fatalf("first kept %s and %s, b's parent %s", got[2].FirstKeptEntryID, got[3].FirstKeptEntryID, *got[1].ParentID)
	}
	if len(got) != 5 || got[4].Type != TypeLabel || got[4].TargetID != a.ID || *got[4].ParentID != self.ID || got[4].Timestamp != "t" {
		t.Fatalf("the label entry %+v", got[len(got)-1])
	}
}
